# Plan Spec — GET /api/system/info 系统信息端点

> 状态：已评审
> 创建：2026-10-10
> 上游：spec/features/system-info-endpoint.md

## 1. 技术上下文

| 项 | 值 |
|---|---|
| 影响层 | 后端（Go）— 不动前端、不动 schema、不动 agent |
| 关键依赖（新增） | 无（仅用 stdlib `runtime`、`time`） |
| 存储 | 无（无 DB 变更） |
| 性能目标 | P95 < 5ms；无非阻塞 IO，直接计算即返回 |

## 2. 现状与影响模块

- 现有实现：
  - [server.go:34-37](file:///d:/tools/ARTEX/server/server.go#L34-L37) `var BuildVersion = "dev"`（包级变量，由 `main.go` 在启动时设置）
  - [server.go:660](file:///d:/tools/ARTEX/server/server.go#L660) `GET /api/health` 返回 `{ok, service, version}`，**不经 JWT 鉴权**（白名单见下）
  - [auth.go:120-126](file:///d:/tools/ARTEX/server/auth.go#L120-L126) `requireAuth` 放行 `strings.HasPrefix(p, "/api/auth/")` 或 `p == "/api/health"`，其他路径必须 JWT
  - [server.go:650-694](file:///d:/tools/ARTEX/server/server.go#L650-L694) `Handler()` 注册所有路由

- 变更模块清单：
  - `server/system_info.go`（新建）— 端点 handler + 包级 `startTime` 变量
  - `server/server.go` — 在 `Handler()` 注册 `GET /api/system/info`；**不动 `requireAuth` 白名单**（默认要求 JWT，正合需求）
  - `server/system_info_test.go`（新建）— 表驱动测试

## 3. 数据模型

无 schema 变更。运行时状态由 `server` 包级变量承载：

```go
// startTime is set at package initialization; used by /api/system/info
// to report uptime. Read-only after init.
var startTime = time.Now()
```

利用 Go 包级变量初始化时机：`server` 包首次被 `main` import 时即记录。比 `init()` 函数更早，足够近似"进程启动"。

## 4. API 约定

| Method | Path | 请求 | 响应 | 鉴权 |
|---|---|---|---|---|
| GET | `/api/system/info` | 无 body，无 query | 200 + JSON | JWT 必需（不在 `requireAuth` 白名单） |

**成功响应**（200）：

```json
{
  "version": "v1.2.3",
  "go_version": "go1.26.3",
  "go_os": "linux",
  "go_arch": "amd64",
  "start_time": "2026-10-10T08:00:00Z",
  "uptime": 1234.5
}
```

- `version`：`string`，与 `/api/health.version` 同源 `BuildVersion`
- `go_version`：`string`，`runtime.Version()`
- `go_os`：`string`，`runtime.GOOS`
- `go_arch`：`string`，`runtime.GOARCH`
- `start_time`：`string`，RFC3339 UTC
- `uptime`：`float64`，`time.Since(startTime).Seconds()`，保留 1 位小数由 `json` 默认编码处理（不强制 round）

**未鉴权响应**（401）：沿用 `requireAuth` 现有 401 格式（不修改）。

**字段黑名单**（测试需断言不存在）：`env`、`secret`、`path`、`hostname`、`api_key` 等任意敏感字段。

## 5. 测试计划（测试先行）

| 层级 | 用例（来自 feature-spec 场景） | 位置 |
|---|---|---|
| 单元 | 已登录 → 200 + 6 字段齐 | `server/system_info_test.go` |
| 单元 | 未登录 → 401 | `server/system_info_test.go` |
| 单元 | `version` 字段等于测试中设置的 `BuildVersion` | 同上 |
| 单元 | 连续两次调用 `uptime` 单调递增 | 同上 |
| 单元 | 响应不含黑名单字段 | 同上 |
| 回归 | `/api/health` 字段不变（仍为 `{ok, service, version}`） | 同上 |

测试用 `httptest.NewRecorder` + 走完整 `s.Handler()`，覆盖中间件路径（不只是 handler 函数），更贴近真实行为。

## 6. 阶段与检查点

| 阶段 | 产出 | 检查点（不过则停） |
|---|---|---|
| S1 测试先行 | `system_info_test.go`，所有用例红 | `go test ./server/ -run SystemInfo` 编译通过、用例失败 |
| S2 实现 | `system_info.go` + `server.go` 路由注册 | S1 测试转绿 |
| S3 回归 | `/api/health` 测试通过 | `go test ./server/ -run Health` 绿 |
| S4 全量 | `go test ./...` | 全绿 |
| S5 文档同步 | 本 plan-spec 状态→已实现；CHANGELOG 加条目 | 文档与代码一致 |

## 7. 风险与回退

- 风险 R1（信息泄露）：未来扩展时误把环境变量塞进响应。
  - 缓解：测试用例 AC5 显式断言黑名单字段不存在；plan-spec 注释提示"只允许 6 个字段"。
- 风险 R2（`startTime` 时机偏差）：包级变量初始化 vs `main` 函数运行有几十毫秒差。
  - 接受：uptime 误差 < 1s 在 AC4 容忍范围内；不为精度引入 `init()` 同步。
- 回退：删除 `system_info.go` + `system_info_test.go` + 撤销 `server.go` 路由注册即可，无 schema 副作用。
