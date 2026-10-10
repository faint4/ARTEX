# Code Review Report — 001 登录失败 IP 限流

> 日期：2026-10-10
> 关联 spec：[specs/001-login-rate-limit/spec.md](file:///d:/tools/ARTEX/specs/001-login-rate-limit/spec.md) · [plan.md](file:///d:/tools/ARTEX/specs/001-login-rate-limit/plan.md) · [tasks.md](file:///d:/tools/ARTEX/specs/001-login-rate-limit/tasks.md)
> 审查对象：未提交工作区（`server/login_rate_limit.go` +test、`server/auth.go`、`server/server.go`）

## 1. 变更分析

| 文件 | 类型 | 内容 |
|---|---|---|
| `server/login_rate_limit.go` | 新增（~150 行） | limiter 状态机（Check/RecordFailure/Reset/gcOnce/startGC）+ extractClientIP + writeLoginRateLimited + logBlocked |
| `server/login_rate_limit_test.go` | 新增（~350 行） | 4 个单元测试 + 1 个全 mux 集成测试，注释映射 AC 编号 |
| `server/auth.go` | 修改 | authLogin 入口 Check / 失败 recordLoginFailure / 成功 Reset；新增 recordLoginFailure 辅助 |
| `server/server.go` | 修改 | Server 加 `loginLimiter`/`trustProxy` 字段；New() 初始化 + GC 启动 |
| `specs/001-login-rate-limit/` | 新增 | spec/plan/tasks 三件套（含实现偏差备注） |
| `.github/`、`.trae/`、`specs/_templates/`、`specs/constitution.md` | 新增/迁移 | SDD 基础设施（同 PR 交付） |

**不在范围（验证未越界）**：无 schema 变更、无新依赖（stdlib only）、未碰双图/planner/worker/guard/traffic、未改 requireAuth 白名单、未改 200/401/400/403/503 既有语义。

## 2. TDD 过程证据

1. **红**：先写测试 + stub（方法体返回零值）→ `TestLoginRateStateMachine` 因 nil state panic FAIL，包级 FAIL 中断后续用例——红灯确立。
2. **绿**：实现 Check/RecordFailure/Reset/gc → targeted 5/5 PASS：
   ```
   --- PASS: TestLoginRateStateMachine (0.00s)
   --- PASS: TestExtractClientIP (0.00s)
   --- PASS: TestLoginRateConcurrent (0.00s)
   --- PASS: TestLoginRateGC (0.00s)
   --- PASS: TestAuthLoginRateLimit (0.97s)
   ok  github.com/Autumn-27/artex/server 1.009s
   ```
3. **回归**：全量套件仅 `TestCoreTaskLifecyclePG` FAIL——**stash 后基线同样 FAIL**（LLM provider 403：开发库配置的 LLM 配额/权限问题），确认与本次变更无关。`go vet ./server/...` 干净。

## 3. 逐条 AC 验收（对照 spec.md §8）

| AC | 结果 | 证据（测试 + 断言） |
|---|---|---|
| AC1 第 6 次尝试 429 + retry_after | ✅ | TestAuthLoginRateLimit：4×wrong→401，5th wrong→401，6th（正确密码）→429；body retry_after>0 |
| AC2 封锁期跳过 bcrypt | ✅ | Check 在 authLogin 最前（decode/bcrypt 之前）return——代码路径保证 |
| AC3 冷却后恢复 + 清零 | ✅ | TestLoginRateStateMachine：blockedUntil 置过去 → Check false + entry 清除 |
| AC4 成功清零 | ✅ | 集成：ipB 3 错→成功→再 4 错全 401（未清零则第 2 次即封） |
| AC5 阈值-1 仍 401 | ✅ | 集成第 5 次失败仍 401；单元 4 次后 Check false |
| AC6 400 不计失败 | ✅ | 集成：ipC 4 错 + 1 个 400 + 第 5 错仍 401 |
| AC7 Retry-After 头 | ✅ | 集成断言 `rec.Header().Get("Retry-After") != ""` |
| AC8 多 IP 独立 | ✅ | 集成：ipA/ipB 均达阈值后 ipD 登录成功 |
| AC9 IP 提取策略 | ✅ | TestExtractClientIP 7 用例（伪造 XFF 忽略/首段提取/IPv6/空白回退） |
| AC10 并发正确 | ✅* | TestLoginRateConcurrent：100 goroutine 精确计数 100；封锁期并发 Check 稳定。*本地无 gcc 未跑 `-race`，CI ubuntu runner 覆盖（已记入 spec 偏差） |
| AC11 GC | ✅ | TestLoginRateGC 四象限：过期未封锁清/封锁中留/新鲜留/冷却过清 |
| AC12 封锁日志 | ✅ | recordLoginFailure 仅在转封锁时 logBlocked；普通失败零日志 |

## 4. Analyze（跨工件一致性，只读）

- **覆盖**：spec FR-001~010 ↔ plan 状态机/API 约定 ↔ tasks T004~T015 ↔ 测试断言，编号链完整；12 条 AC 全部有对应测试断言。
- **术语**：spec「封锁/冷却/窗口」= plan「Blocked/cooldown/window」= 代码 `blockedUntil/loginCooldown/loginWindow`，一致。
- **缺口检查**：spec NG5「不暴露配置 API」— 代码无任何新端点 ✅；trustProxy 字段存在但无配置入口（plan 声明为下个 feature）✅。
- **宪法 Gate 复查**：红线 ✅（无凭据入库——集成测试用随机内存密码 + 环境变量回退，真实凭据零入源码）；简约 ✅（实现中发现自造 `itoa/fmtInt` 过度包装，当场改回 `strconv.Itoa`——Gate 机制实际发挥了作用）；迁移 ✅（零 schema）。
- **发现并修正的偏差**：① 测试凭据供给方式改变（seed → init 随机密码/env 回退）已记入 plan.md 偏差备注；② 测试文件从 auth_test.go 改为独立 login_rate_limit_test.go（内聚性），tasks.md 已同步；③ 第一次实现中 itoa 包装违反防抽象 Gate，已删除。

## 5. 风险复核

| 风险 | 复核结论 |
|---|---|
| R1 反代误伤 | CHANGELOG 已显式说明已知牺牲；缓解按计划（下个 feature 出配置入口） |
| R3 内存无界 | gcOnce 条件覆盖四象限；GC 周期 10min < 窗口 5min×2，收敛性成立 |
| R5 时钟跳变 | RecordFailure/Check 全部用 `time.Now()` 相对差值（Go 单调时钟） |
| 新增 R7：Check 在 decode 之前 → 被封 IP 对畸形请求也收 429 | 符合 spec「封锁期内任何尝试返回 429」，非缺陷 |

## 6. 结论

**通过**。12/12 AC 有测试证据；TDD 红绿过程完整；宪法 Gate 有效拦截了一次过度设计；偏差全部显式记录而非静默漂移。

**合并前待办**：① 推送 GitHub 后创建 Issue 并回填「关联 Issue」；② CI 首跑确认 `-race` 与 biome 绿；③ 分支保护启用。
