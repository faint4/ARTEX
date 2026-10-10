# Code Review Report — GET /api/system/info

> 日期：2026-10-10
> 关联 spec：[spec/features/system-info-endpoint.md](file:///d:/tools/ARTEX/spec/features/system-info-endpoint.md) · [spec/plans/system-info-endpoint.md](file:///d:/tools/ARTEX/spec/plans/system-info-endpoint.md) · [spec/tasks/system-info-endpoint.md](file:///d:/tools/ARTEX/spec/tasks/system-info-endpoint.md)
> 审查对象：未提交（工作区状态）

## 1. 变更分析

### 1.1 变更范围

| 文件 | 类型 | 行数（±） | 作用 |
|---|---|---|---|
| `server/system_info.go` | 新增 | +43 | handler 实现 + 包级 `startTime` |
| `server/system_info_test.go` | 新增 | +144 | 表驱动测试覆盖 AC1–AC6 |
| `server/server.go` | 修改 | +4 / -0 | 在 `Handler()` 注册 `GET /api/system/info` |
| `spec/features/system-info-endpoint.md` | 新增 | — | feature-spec |
| `spec/plans/system-info-endpoint.md` | 新增 | — | plan-spec |
| `spec/tasks/system-info-endpoint.md` | 新增 | — | task-spec |

**净代码量**：47 行 Go 实现代码 + 144 行测试代码。测试/实现比 ≈ 3:1，符合宪法 IV「新逻辑必须有表驱动测试」。

### 1.2 不在变更范围（验证未越界）

- ✅ 未碰 `db/schema.sql`（无 schema 变更）
- ✅ 未碰 `agent/`（无 planner/worker 路径修改）
- ✅ 未碰 `auth.go` 的 `requireAuth` 白名单（新端点默认要求 JWT）
- ✅ 未碰前端 `web/`（本次仅后端）
- ✅ 未新增 Go 依赖（`go.mod` 不变）

## 2. 功能正确性（对照 feature-spec 验收清单）

| AC | 描述 | 测试位置 | 结果 |
|---|---|---|---|
| AC1 | 已登录 → 200 + 6 字段齐 | `system_info_test.go` L70-89 | ✅ PASS |
| AC2 | 未登录 → 401 | `system_info_test.go` L62-68 | ✅ PASS |
| AC3 | version == BuildVersion | L96-98 | ✅ PASS |
| AC4 | uptime 单调非递减 | L101-110 | ✅ PASS |
| AC5 | 响应不含黑名单字段 | L82-88（closed-set 检查） | ✅ PASS |
| AC6 | /api/health 字段不变 | L119-132 | ✅ PASS |

**测试运行证据**：
```
=== RUN   TestSystemInfoEndpoint
--- PASS: TestSystemInfoEndpoint (0.22s)
PASS
ok      github.com/Autumn-27/artex/server       0.256s
```

`go vet ./server/...` 无告警。

## 3. 架构一致性（对照宪法）

### 宪法 I（红线）
- ✅ 端点不暴露凭据、不引入未授权目标扫描路径
- ✅ 响应字段集为闭集白名单（`version / go_version / go_os / go_arch / start_time / uptime`），无 env/path/hostname/secret
- ✅ 默认要求 JWT 鉴权（不在 `requireAuth` 白名单）

### 宪法 II（架构不变量）
- ✅ 不碰双图：未引入新的资产图/探索图节点
- ✅ 不碰 planner/worker：未新增 agent 决策路径
- ✅ 不碰事件驱动：未引入轮询
- ✅ 不碰 guard/intercept：未碰审批门
- ✅ 不碰流量记录：未改 traffic 包

### 宪法 III（技术约束）
- ✅ 单二进制交付不变（`go:embed` 路径未碰）
- ✅ 无 schema 变更（无 DB 迁移）
- ✅ 复用 `server.BuildVersion` 包级变量（与 `/api/health` 同源）
- ✅ 仅用 stdlib `runtime / time / encoding/json / net/http`，无新依赖
- ✅ 不碰 norma SDK

### 宪法 IV（质量门槛）
- ✅ `go test ./server/ -run SystemInfo` 全绿
- ✅ 表驱动测试（`allowedFields` map + 闭集断言）
- ✅ `go vet ./server/...` 无告警
- ✅ spec 与代码同 PR（spec 文件与实现代码一并写入工作区）

### 宪法 V（fork 治理）
- ✅ 无发版/镜像变更

### 代码风格（对照 rules/code_style_go.md）
- ✅ 单一职责：`system_info.go` 只承担 `/api/system/info` 的 handler
- ✅ 显式错误：`_ = json.NewEncoder(w).Encode(resp)` 显式忽略（编码器写 ResponseWriter 不会失败到值得处理）
- ✅ Context 纪律：handler 不涉及跨层 IO，未引入 goroutine
- ✅ 并发安全：`startTime` 包级变量只读
- ✅ 注释以 `// FuncName ...` 开头（包级 `startTime` 和 `systemInfo` 均有）
- ✅ 安全约束：响应字段为闭集，未引入 SQL/shell 注入面
- ✅ 日志风格：本 handler 不打日志（与 `/api/health` 一致，避免 noisy 探针请求刷屏）

## 4. 风险评估

| ID | 风险 | 严重度 | 缓解 |
|---|---|---|---|
| R1 | 未来扩展 `systemInfo` 时误把环境变量塞进响应 | 中 | 测试 AC5 用闭集断言；plan-spec §4 字段白名单；handler 注释明示「只允许 6 个字段」 |
| R2 | `startTime` 包级变量初始化时机 vs `main` 函数入口有几十毫秒差 | 低 | feature-spec AC4 容忍 < 1s；不引入 `init()` 同步避免过度工程 |
| R3 | `uptime` 为 `float64`，JSON 编码可能出现精度尾长差异 | 低 | 客户端只读不计算，精度足够；如需稳定可后续切 `int64` 秒 |
| R4 | `_ = json.NewEncoder(w).Encode(resp)` 吞掉编码错误 | 极低 | 写入 `http.ResponseWriter` 的编码失败无有意义处理路径（连接已断等）；与既有 `writeJSON` 行为一致 |

## 5. 改进建议（不阻塞合并）

- S1（可选）：将 `_ = json.NewEncoder(w).Encode(resp)` 换成 `s.writeJSON(w, 200, resp)` 复用既有 helper（如 `writeJSON` 存在且语义一致），减少重复代码。
- S2（可选）：在 `CHANGELOG.md` 增条目（task-spec T4.2 已要求）。
- S3（可选）：前端登录页可调用此端点展示版本信息（Non-Goal 之一，留待后续 feature）。

## 6. 完成定义（DoD）核查

- [x] 宪法 IV 质量门槛满足：`go test` 全绿、`go vet` 无告警、表驱动测试存在
- [x] spec 与代码状态一致：feature-spec / plan-spec 状态字段待 PR 提交时改为「已实现」（task-spec T4.1）
- [x] 测试先行（TDD）证据：先写测试 → 编译失败（红）→ 写实现 → 测试转绿
- [x] CR 报告存档：本文件

## 7. 结论

**通过**。变更范围与 plan-spec 一致，无越界；6 条验收清单全绿；宪法 I–IV 全部满足；风险可控。

建议提交前完成：
1. `spec/plans/system-info-endpoint.md` 状态 → 已实现
2. `CHANGELOG.md` 增条目
3. 提交时把 spec/ + server/system_info*.go + server.go 修改放同一 commit
