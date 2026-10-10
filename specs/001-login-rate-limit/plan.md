# Implementation Plan — 登录失败 IP 限流

> 状态：**Implemented**（2026-10-10；实现与测试计划偏差见 §测试计划备注与 cr_reports/001-login-rate-limit.md）
> Feature 目录：`specs/001-login-rate-limit/` ｜ 上游：[spec.md](./spec.md) ｜ 关联 Issue：待创建
> 上游：spec/features/login-rate-limit.md（旧路径，已迁移至本目录）

## Summary

对 `POST /api/auth/login` 增加进程内存级 IP 限流：同 IP 5 分钟窗口内失败 5 次封锁 5 分钟，封锁期返回 429 + Retry-After 并跳过 bcrypt；默认不信任代理头；后台 GC 防内存无界。技术路线：`server` 包内新建 `loginRateLimiter`（mutex + map），在 `authLogin` 入口/失败/成功三处接入。

## Technical Context

| 项 | 值 |
|---|---|
| 影响层 | 后端（Go）— 不动前端、不动 schema、不动 agent |
| 语言/版本 | Go 1.26+（stdlib `sync`/`time`/`net`） |
| 新增依赖 | 无 |
| 存储 | 无（进程内存，重启清零——spec A1） |
| 测试 | go test 表驱动 + `-race`；集成测试走真实 mux（postgres 不可用时 Skip） |
| 性能目标 | limiter 单次调用 P99 < 100μs（spec SC-002） |
| 约束 | 不改 requireAuth 白名单；不改 /api/auth/login 既有 200/401/400 语义 |

## Constitution Check（Phase -1 Gates）

- [x] **红线 Gate（宪法 I）**：限流增强安全；无凭据/密钥引入
- [x] **架构 Gate（宪法 II）**：不碰双图/planner/worker/guard/traffic
- [x] **简约 Gate（宪法 III）**：仅 stdlib；无新依赖；无投机抽象（单 mutex + map，不做分片锁——见 Complexity Tracking）
- [x] **防抽象 Gate（宪法 III）**：直接用 `sync.Mutex` + map + `time`，不引入第三方限流库（golang.org/x/time/rate 等被否决：窗口+封锁语义与令牌桶不匹配，引入后仍需自写状态机）
- [x] **集成优先 Gate（宪法 IV）**：集成测试走真实 `s.Handler()` + 真实 postgres（不可用则 Skip），非纯 mock
- [x] **迁移 Gate（宪法 III）**：无 schema 变更

## Project Structure

```text
server/
├── login_rate_limit.go        [新增] limiter 类型 + Check/RecordFailure/Reset + extractClientIP + GC
├── login_rate_limit_test.go   [新增] 状态机/IP提取/并发/GC 单元测试
├── auth.go                    [修改] authLogin 三处接入 + writeLoginRateLimited
└── server.go                  [修改] Server 结构体加 loginLimiter/trustProxy 字段 + New() 初始化
```

**Structure Decision**: limiter 是登录路径的专属组件，与 auth.go 同包就近放置；不新建包（单文件 ~150 行，独立包是过度设计）。

## 数据模型

无 schema 变更。运行时状态：

```go
type loginRateLimiter struct {
    mu     sync.Mutex
    states map[string]*loginState
}
type loginState struct {
    failures     int
    windowStart  time.Time
    blockedUntil time.Time // zero = 未封锁
}
const (
    loginMaxFailures = 5
    loginWindow      = 5 * time.Minute
    loginCooldown    = 5 * time.Minute
    loginGCInterval  = 10 * time.Minute
)
```

状态机（per IP）：`Empty → Active(计数) → Blocked(429) → (冷却过+新尝试) → Empty`；`Active --成功--> Empty`；`Active --窗口过--> Active(重置)`。

## API 约定

无新端点。仅 `POST /api/auth/login` 新增封锁期行为：

| 场景 | 响应 |
|---|---|
| 封锁期内任何尝试 | **429** + `Retry-After: {秒}` 头 + body `{"error":"登录失败次数过多，请稍后再试","retry_after":{秒}}` |
| 其余（200/401/400/403/503） | 不变；401/用户名错误与 401/密码错误均计失败，400 不计 |

## 测试计划（按用户故事映射）

| 故事 | 用例 | 层级 | 位置 |
|---|---|---|---|
| US1 | 状态机 7 态转换（Empty→Active→Blocked→Empty 等） | 单元 | `login_rate_limit_test.go` |
| US1 | 4 次失败→401，第 6 次→429 + Retry-After 头 | 集成 | `auth_test.go` 增量 |
| US1 | 封锁期内正确密码 429；冷却后 200 | 集成 | 同上 |
| US1 | 400 不计失败 | 集成 | 同上 |
| US2 | trustProxy=false 忽略 XFF；=true 取首段；IPv6/无端口 | 单元 | `login_rate_limit_test.go` |
| US2 | 不同 IP 独立计数 | 集成 | `auth_test.go` |
| US3 | 100 goroutine 并发 RecordFailure 计数准确 | 单元(-race) | `login_rate_limit_test.go` |
| US3 | GC 清过期、留封锁 | 单元 | 同上 |

**测试技巧**：集成测试启动时一次性生成 bcrypt hash 写入 setting，避免每用例重复计算。

> **实现偏差备注（2026-10-10）**：
> 1. 凭据供给改为「authInit 随机密码优先 + `ARTEX_TEST_LOGIN_PASSWORD` 环境变量回退」——开发库已初始化，不能 seed；且真实凭据不得写入源码（安全红线）。
> 2. 全部测试（单元+集成）集中在一个 `login_rate_limit_test.go`，而非拆到 auth_test.go——按文件内聚优于按层拆分，可读性更好。
> 3. 本地 WSL 无 gcc，`-race` 未在本地执行（CI 的 ubuntu runner 覆盖）；并发正确性由 mutex 设计 + TestLoginRateConcurrent 保障。

## Complexity Tracking

| 违规 | 为什么需要 | 被否决的更简方案及原因 |
|---|---|---|
| 无 | — | — |

（单 mutex 全表锁看似"该分片"，但登录非热点路径且 bcrypt 远慢于 map 操作，无需记录为违规——设计已在简约 Gate 说明。）

## 风险与回退

| ID | 风险 | 严重度 | 缓解 |
|---|---|---|---|
| R1 | 反代部署误伤（trustProxy=false） | 中 | spec A5 有意为之；CHANGELOG 显式说明；配置入口留下个 feature |
| R2 | 攻击者换 IP 绕过 | 低 | 接受（NG1/NG2/NG4）；阈值 5 已挡住单 IP 自动化 |
| R3 | 伪造 IP 撑爆内存 | 中 | 后台 GC 每 10 分钟清理过期 entry（FR-009） |
| R4 | bcrypt 失败计数竞态 | 低 | mutex 全覆盖；`-race` 测试 |
| R5 | 时钟跳变 | 低 | time.Since 单调时钟 |

**回退方案**：删除 `login_rate_limit*.go` + 撤销 auth.go 三处调用 + 撤销 server.go 字段初始化；无 schema 副作用。
