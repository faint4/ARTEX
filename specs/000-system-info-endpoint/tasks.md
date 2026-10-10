# Task Spec — GET /api/system/info 系统信息端点

> 上游：spec/plans/system-info-endpoint.md
> 拆解原则：每个任务可独立提交、可独立验证；测试任务先于对应实现任务（TDD）。

## 0. 配置任务

- [x] T0.1 无依赖/环境变更 — 验收：无新增 Go module（`go.mod` 不变）

## 1. 测试任务（先行）

- [ ] T1.1 在 `server/system_info_test.go` 写表驱动测试，覆盖：
  - 已登录 → 200 + 6 字段齐（version/go_version/go_os/go_arch/start_time/uptime）
  - 未登录 → 401
  - version 字段等于测试中设置的 `BuildVersion`
  - 连续两次调用 `uptime` 单调递增
  - 响应不含黑名单字段（env/secret/path/hostname/api_key）
  - `/api/health` 字段不变（回归）
  — 验收：`go test ./server/ -run SystemInfo` 编译通过，用例失败（红）
- [ ] T1.2 测试通过 `s.Handler()` 走完整中间件链，而非直接调 handler 函数 — 验收：测试代码含 `s.Handler().ServeHTTP` 调用

## 2. 核心开发任务

| ID | 任务 | 依赖 | 涉及文件 | 验收 |
|---|---|---|---|---|
| T2.1 | 新建 `server/system_info.go`，定义包级 `startTime = time.Now()` 与 `func (s *Server) systemInfo(w, r)` handler，返回 6 字段 JSON | T1.1 | `server/system_info.go` | T1.1 中 200/字段集 用例转绿 |
| T2.2 | 在 `server.go` 的 `Handler()` 注册 `mux.HandleFunc("GET /api/system/info", s.systemInfo)`，紧邻 `/api/health` 之后 | T2.1 | `server/server.go` | T1.1 中 401 用例转绿（默认经 `requireAuth`） |

## 3. 集成任务

- [ ] T3.1 端到端：`go test ./server/ -run SystemInfo` 全绿 — 验收：所有 6 用例通过
- [ ] T3.2 回归：`go test ./server/ -run Health` 通过 — 验收：`/api/health` 行为不变
- [ ] T3.3 全量：`go test ./...` 全绿 — 验收：无破坏既有测试

## 4. 收尾任务

- [ ] T4.1 `spec/plans/system-info-endpoint.md` 状态 → 已实现；feature-spec 状态 → 已实现
- [ ] T4.2 在 `CHANGELOG.md` 增条目（如不存在则创建）
- [ ] T4.3 `go vet ./server/...` 无告警
- [ ] T4.4 Code Review（对照 spec），报告存 `cr_reports/{commit}.md`

## 完成定义（DoD）

全部任务勾选 + 宪法 IV 质量门槛满足（`go test` 全绿、`go vet` 无告警、新逻辑有表驱动测试）+ spec 与代码状态一致。
