# Go 后端代码风格与工程约束

本规范用于 ARTEX 后端（`cmd/ agent/ server/ db/ traffic/ guard/ …`）。

## 核心原则

### I. 单一职责与分层
- 一个包一个职责；禁止在 `server/` 里写 agent 决策逻辑，禁止在 `agent/` 里直接写 HTTP handler。
- 数据库访问集中在 `db/`；其他层不得直接拼 SQL。

### II. 显式错误处理
- 逐层检查 `err`，用 `fmt.Errorf("…: %w", err)` 包装带上下文；禁止 `_ = err` 吞错（日志类可显式 `_ = ` 并注释原因）。
- handler 层错误统一走 `writeErr(w, code, msg)`，中文用户可读消息。

### III. Context 纪律
- 所有跨层/IO 函数首参收 `ctx context.Context`；goroutine 退出必须可被 ctx 取消。
- 引擎循环（plannerLoop / worker）必须响应暂停/drain 信号。

### IV. 并发安全
- 共享状态用 mutex 或 channel，注释标明「谁写谁读」；
- 禁止在无锁情况下读写 map；图写操作必须经 db 层事务/唯一约束保证幂等。

### V. 幂等与兼容
- schema 变更仅限幂等语句（`ADD COLUMN IF NOT EXISTS` 等）；旧数据必须可读（列可空或有默认值）。
- API 变更保持向后兼容；破坏性变更需在 plan-spec 中给迁移方案。

## 代码风格

- 格式以 `gofmt` 为准；命名遵循 Go 惯例（缩写词全大写：`URL`、`ID`、`SSE`）。
- 注释以 `// FuncName …` 开头，解释「为什么」而非复述代码；导出符号必须有注释。
- 表驱动测试：`test := []struct{ name string; … }{…}` + `t.Run`；测试文件与被测包同目录 `_test.go`。
- 禁止 `panic` 穿越包边界；仅程序初始化阶段（配置缺失等）允许 `log.Fatal`。
- 日志用 `log.Printf` 带模块前缀（如 `[pg]`、`[engine]`、`[traffic]`），与现有风格一致。

## 安全约束（宪法 I 的落地）

- 不写「默认开启扫描外网目标」的代码路径；目标范围（task_scope）校验不得放宽。
- 用户输入进入 SQL 一律参数化（pgx 占位符）；进入 shell 的参数必须转义或白名单。
- 密钥只从环境变量/配置读取；日志不得打印完整密钥与凭据。

## 提交前检查

- [ ] `gofmt -l .` 为空
- [ ] `go vet ./...` 无告警
- [ ] `go test ./...` 全绿（新逻辑有表驱动测试）
- [ ] 对照 `spec/constitution.md` II/III 自查
