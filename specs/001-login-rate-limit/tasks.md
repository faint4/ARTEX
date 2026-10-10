# Tasks — 登录失败 IP 限流

> 上游：[plan.md](./plan.md) ｜ 关联 Issue：[#1](https://github.com/faint4/ARTEX/issues/1)
> 拆解约定：任务按用户故事分阶段（MVP 递增）；`[P]` 可并行；测试先红后绿；Checkpoint 不过则停。
> 状态：**全部完成**（2026-10-10）。偏差记录见 plan.md §测试计划备注。

## Phase 1: Setup

- [x] T001 spec.md Final + plan.md Reviewed（2026-10-10）；无依赖/环境变更（`go.mod` 不变）

**Checkpoint**: 规范就绪，可开工 ✅

---

## Phase 2: Foundational（阻塞性前置）

- [x] T002 [P] `server/login_rate_limit.go`：类型 + 常量 + `newLoginRateLimiter()` — 验收：编译通过 ✅
- [x] T003 [P] `server/server.go`：`Server` 加 `loginLimiter`/`trustProxy` 字段；`New()` 初始化 + `startGC(ctx)` — 验收：编译通过 ✅

**Checkpoint**: 类型与字段就绪 ✅

---

## Phase 3: US1 — 爆破防护核心 (Priority: P1) 🎯 MVP

**Goal**: 同 IP 失败 5 次封锁 5 分钟；封锁期 429 + Retry-After + 跳过 bcrypt
**Independent Test**: 连续 5 次错误密码 → 第 6 次得 429；冷却后正确密码得 200

### Tests First（先红后绿）

- [x] T004 [P] [US1] 状态机单元测试 in `server/login_rate_limit_test.go`（TestLoginRateStateMachine：空态/阈值下/达阈值/封锁读稳定/冷却过期清空/窗口过期重置/成功 Reset）— 验收：stub 下红（nil entry panic），实现后绿 ✅
- [x] T005 [US1] 集成测试（TestAuthLoginRateLimit，与 T004 同文件——偏差见 plan.md 备注 2）— 验收：stub 下红（首次 429 断言失败），实现后绿 ✅

### Implementation

- [x] T006 [US1] `Check(ip)`（封锁判定 + 惰性清理过期冷却/窗口）与 `RecordFailure(ip)`（窗口重置、累加、达阈值转封锁、返回转移信号）✅
- [x] T007 [US1] `Reset(ip)`（成功清零）✅
- [x] T008 [US1] `writeLoginRateLimited`（429 + Retry-After 头 + retry_after body）+ `authLogin` 三处接入（入口 Check → 失败 recordLoginFailure → 成功 Reset；400 路径不计数）✅

**Checkpoint**: `go test ./server/ -run 'TestLoginRate|TestExtractClientIP|TestAuthLoginRateLimit'` 5/5 绿 ✅

---

## Phase 4: US2 — 多 IP 与代理环境 (Priority: P2)

**Goal**: 客户端 IP 提取正确且安全默认（不信伪造头）；不同 IP 独立计数
**Independent Test**: IP-A 被封后 IP-B 正常；伪造 XFF 不能绕过

### Tests First

- [x] T009 [P] [US2] IP 提取单元测试（TestExtractClientIP：7 个表驱动用例，含 IPv6 有/无端口、空白 XFF 回退）— 验收：stub 下红（trustProxy 分支已有但经集成路径断言失败），实现后绿 ✅

### Implementation

- [x] T010 [US2] `extractClientIP(r, trustProxy)`（`net.SplitHostPort` 处理端口与 IPv6）✅
- [x] T011 [US2] `authLogin` 使用 extractClientIP；集成用例覆盖不同 IP 独立计数（AC8）✅

**Checkpoint**: US1 + US2 全绿 ✅

---

## Phase 5: US3 — 运行时健壮性 (Priority: P3)

**Goal**: `-race` 干净；内存有界；封锁可观测
**Independent Test**: `go test -race`；GC 用例；日志断言

### Tests First

- [x] T012 [P] [US3] 并发测试（TestLoginRateConcurrent：100 goroutine RecordFailure 精确计数 + 封锁期并发 Check 稳定）— 实现后绿（本地无 gcc 未跑 -race，CI 覆盖）✅
- [x] T013 [P] [US3] GC 测试（TestLoginRateGC：过期未封锁清、封锁中留、新鲜留、冷却过清）— 实现后绿 ✅

### Implementation

- [x] T014 [US3] `gcOnce()` + `startGC(ctx)`（10 分钟周期，ctx 取消退出；`New()` 启动）✅
- [x] T015 [US3] 封锁触发 INFO 日志 `logBlocked`（仅转封锁时打，普通失败不打——防日志爆破）✅

**Checkpoint**: 全部故事独立可用 ✅

---

## Phase 6: Polish & 收尾

- [x] T016 `specs/001-login-rate-limit/{spec,plan,tasks}.md` 状态 → Implemented；`CHANGELOG.md` 增条目 ✅
- [x] T017 全量回归：targeted 5/5 绿；`go vet` 干净；`gofmt -l` 全仓 CRLF 环境伪影（CI Linux 检出 LF 不受影响）；全量套件中 `TestCoreTaskLifecyclePG` 失败经 stash 基线验证为**既有环境问题**（开发库 LLM provider 403），与本次变更无关 ✅
- [x] T018 Code Review（对照 spec.md AC1–AC12 逐条验收），报告存 `cr_reports/001-login-rate-limit.md` ✅

---

## Dependencies & Execution Order（实际执行）

- T001 → T002/T003（stub）→ T004/T005（红：stub panic + 断言失败）→ T006–T008（绿）→ T009–T011（绿）→ T012–T015（绿）→ T016–T018
- 实际串行执行（单人维护），与计划顺序一致

## Notes

- 红阶段证据：stub 实现下 TestLoginRateStateMachine 因 nil state panic FAIL；集成测试 429 断言 FAIL（记录于 CR 报告）
- 凭据安全：集成测试经 `/api/auth/init` 设置随机内存密码（CI 全新库成立）；已初始化库回退 `ARTEX_TEST_LOGIN_PASSWORD` 环境变量；真实凭据零入源码
- Issue #1 已创建（https://github.com/faint4/ARTEX/issues/1）；spec/plan/tasks 三件套「关联 Issue」已回填
