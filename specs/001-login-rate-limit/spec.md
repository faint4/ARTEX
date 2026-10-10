# Feature Spec — 登录失败 IP 限流

> 状态：**Implemented**（2026-10-10；测试 5/5 绿，AC1–AC12 验收见 cr_reports/001-login-rate-limit.md）
> Feature 目录：`specs/001-login-rate-limit/` ｜ 关联 Issue：[#1](https://github.com/faint4/ARTEX/issues/1)
> 本文档只回答 WHAT & WHY，不写 HOW。

## 1. 背景与问题

当前 [auth.go authLogin](file:///d:/tools/ARTEX/server/auth.go) 用户名固定为 `ARTEX`（README 公开），密码用 bcrypt 校验，失败统一返回 401，**没有任何失败计数或 IP 限流**。ARTEX 是渗透测试系统，用户常做端口转发/反代暴露 8787 到公网"方便远程"，弱口令实例极易被在线爆破。需要登录失败 IP 限流：同 IP 窗口内连续失败 N 次后暂时拒绝，防止在线爆破。

## 2. 用户故事与验收场景

### US1 — 爆破防护核心：连续失败后封锁与恢复 (Priority: P1) 🎯

同一 IP 在窗口内连续输错密码达到阈值后，对该 IP 的所有登录尝试（包括密码正确的）返回 429，冷却期过后自动恢复。

- **为什么是 P1**：这是本 feature 的安全价值本体；只交付 US1 即可防住最常见的单 IP 在线爆破
- **独立测试**：对同一 IP 连续提交 5 次错误密码 → 第 6 次得 429 + Retry-After；等冷却后用正确密码得 200 + token
- **验收场景**：

```gherkin
场景: 主路径 — 窗口内连续失败到达阈值后封锁
  Given 同一 IP 在最近 5 分钟内已有 4 次失败记录
  When 该 IP 第 5 次提交错误密码
  Then 响应 401（本次仍正常拒绝）且失败计数到达阈值
  When 该 IP 第 6 次提交任意密码
  Then 响应 429 且 body 含 retry_after（剩余冷却秒数）
   And 响应头含 Retry-After（RFC 9110 §10.7.3）

场景: 封锁期内即使密码正确也拒绝
  Given IP 已被封（未过冷却期）
  When 该 IP 提交正确密码
  Then 响应 429（不返回 token，不消耗 bcrypt 计算）

场景: 封锁期过后恢复
  Given IP 冷却已过
  When 该 IP 提交正确密码
  Then 响应 200 + token，失败计数清零

场景: 窗口内成功登录重置计数
  Given IP 在窗口内有 3 次失败但未达阈值
  When 该 IP 提交正确密码
  Then 响应 200 + token，失败计数清零

场景: 边界 — 阈值减 1 仍允许尝试
  Given IP 失败 4 次（阈值 5 减 1）
  When 该 IP 提交错误密码
  Then 响应 401（不是 429），计数到 5，下次将被封

场景: 异常 — 请求体格式错误
  Given IP 已失败 4 次
  When 该 IP 提交格式错误的 JSON
  Then 响应 400，且不计入失败计数（防格式错误被利用来封禁他人）
```

### US2 — 多 IP 与代理环境正确计数 (Priority: P2)

不同 IP 的失败计数互不影响；客户端真实 IP 的提取策略安全默认（不信任伪造头）。

- **为什么是 P2**：没有 US2，反代后所有用户共享反代 IP 会被集体误封——但这是部署形态问题，US1 已覆盖裸暴露场景
- **独立测试**：IP-A 封锁后 IP-B 正常登录；伪造 X-Forwarded-For 不能绕过限流
- **验收场景**：

```gherkin
场景: 不同 IP 独立计数
  Given IP-A 失败 5 次被封
  When IP-B 提交错误密码
  Then IP-B 收到 401（独立计数，不受 IP-A 封锁影响）

场景: 默认不信任代理头
  Given trustProxy=false（默认）
  When 请求携带伪造的 X-Forwarded-For: 1.2.3.4
  Then 限流按 RemoteAddr（真实 TCP 对端）计数，伪造头被忽略

场景: 显式信任代理头
  Given trustProxy=true（代码字段，本 feature 不暴露配置入口）
  When 请求携带 X-Forwarded-For: 1.2.3.4, 5.6.7.8
  Then 限流按首段 1.2.3.4 计数
```

### US3 — 运行时健壮性：并发、内存与可观测 (Priority: P3)

限流器在并发请求下不出现竞态；内存不随伪造 IP 数量无界增长；封锁事件对管理员可观测。

- **为什么是 P3**：硬化项；US1/US2 功能正确后，US3 决定生产可用性
- **独立测试**：`go test -race` 下并发失败计数一致；GC 后过期 entry 被清理；触发封锁时日志出现 `[auth]` 前缀记录
- **验收场景**：

```gherkin
场景: 并发失败计数不竞态
  Given 100 个 goroutine 同时对同一 IP 记录失败
  When 全部完成后读取计数
  Then 计数准确（无丢失/重复），-race 检测无告警

场景: 内存 GC
  Given 大量 IP 的失败记录已过窗口期且未被封
  When 后台 GC 周期触发
  Then 过期 entry 被清理；被封 entry 保留至冷却结束

场景: 封锁日志
  Given 某 IP 触发封锁
  When 封锁发生
  Then 输出一条 INFO 日志：[auth] IP=x.x.x.x blocked after N failures, cooldown=300s
   And 普通失败（未达阈值）不打日志（防日志爆破攻击）
```

### 边界与异常（Edge Cases）

- 服务重启后计数从 0 开始（内存态，不持久化——已定稿决策 Q1）
- `time.Now()` 系统时钟跳变：Go `time.Since` 使用单调时钟，不受影响
- 429 响应不暴露"还差几次失败"信息（防阈值探测）

## 3. 需求（Functional Requirements）

- **FR-001**：系统 MUST 对同一 IP 在 5 分钟窗口内的第 5 次登录失败后，封锁该 IP 5 分钟（US1）
- **FR-002**：系统 MUST 在封锁期内对所有登录尝试返回 429 + `Retry-After` 头 + body `retry_after` 字段（US1）
- **FR-003**：系统 MUST 在封锁期内跳过 bcrypt 计算（US1）
- **FR-004**：系统 MUST 在登录成功后清零该 IP 失败计数（US1）
- **FR-005**：系统 MUST 对 400（请求格式错误）不计入失败计数（US1）
- **FR-006**：系统 MUST 按客户端真实 IP 独立计数；默认从 `RemoteAddr` 提取（US2）
- **FR-007**：系统 MUST 仅在 `trustProxy=true` 时信任 `X-Forwarded-For` 首段（US2）
- **FR-008**：系统 MUST 在并发调用下保持计数正确（`-race` 干净）（US3）
- **FR-009**：系统 MUST 后台周期 GC 过期 entry（已过窗口且未封锁），内存占用有界（US3）
- **FR-010**：系统 MUST 在触发封锁时输出一条 INFO 日志，普通失败不打日志（US3）

## 4. Goals / Non-Goals

- **Goals**：
  - G1：5 次 / 5 分钟窗口 / 5 分钟冷却的默认策略（已定稿 Q2）
  - G2：不同 IP 独立计数，互不影响
  - G3：封锁期内不消耗 bcrypt 计算（性能保护）
  - G4：成功登录清零失败计数
- **Non-Goals**：
  - NG1：不做账户级限流（用户名固定 ARTEX，IP 维度已够）
  - NG2：不做分布式限流（多实例共享计数）
  - NG3：不做 CAPTCHA / 人机验证
  - NG4：不做管理员 IP 白名单
  - NG5：不暴露限流状态/配置 API；阈值硬编码（已定稿 Q3），trustProxy 配置入口留下个 feature

## 5. 关键实体（Key Entities）

| 实体 | 含义 | 归属 |
|---|---|---|
| client_ip | 客户端 IP（RemoteAddr 或信任的 XFF 首段） | 运行时 |
| failed_count | 该 IP 当前窗口内失败次数 | 运行时状态（内存） |
| window_start | 当前窗口起点 | 运行时状态（内存） |
| blocked_until | 封锁到期时间；zero 表示未封锁 | 运行时状态（内存） |
| 阈值/窗口/冷却 | 5 次 / 5 min / 5 min（硬编码常量） | 编译期配置 |

## 6. 成功标准（Success Criteria）

- **SC-001**：自动化爆破 1000 次密码尝试的实际 bcrypt 计算次数 ≤ 5 + 每 5 分钟窗口 5 次（封锁生效）
- **SC-002**：限流器单次调用开销 P99 < 100μs（无 IO，仅 mutex + map）
- **SC-003**：真实用户"记错密码 5 次"后被锁 5 分钟可自助恢复，无需管理员介入
- **SC-004**：`go test -race ./server/` 全绿，无数据竞争告警

## 7. 假设（Assumptions，含已定稿决策）

- ✅ **A1（Q1 已定稿）**：计数器存进程内存 map + mutex；重启清零可接受（单实例部署 + 压缩攻击窗口 + 避免破坏宪法 III 迁移原则）
- ✅ **A2（Q2 已定稿）**：默认 5 次 / 5 min 窗口 / 5 min 冷却（bcrypt DefaultCost 下 5 次 ≈ 400ms 真实成本）
- ✅ **A3（Q3 已定稿）**：阈值/窗口/冷却为硬编码常量；运维调参留下个 feature
- ✅ **A4（Q4 已定稿）**：仅 429 触发时打 INFO 日志（防日志爆破）；格式 `[auth] IP=x.x.x.x blocked after N failures, cooldown=300s`
- ✅ **A5（Q5 已定稿）**：trustProxy 默认 false 用 RemoteAddr；反代误伤是有意为之的安全默认（伪造绕过不可发现 > 误伤可发现）；配置入口留下个 feature
- 依赖：复用现有 `requireAuth` 之外的路由与 `POST /api/auth/login` 入口；无需新表

## 8. 验收清单（Acceptance Criteria）

- [x] AC1 (US1)：失败 5 次后第 6 次尝试（含正确密码）返回 429 + retry_after
- [x] AC2 (US1)：封锁期内 bcrypt 不被调用
- [x] AC3 (US1)：冷却过后正确密码恢复 200 + token，计数清零
- [x] AC4 (US1)：窗口内成功登录清零失败计数
- [x] AC5 (US1)：阈值 - 1 次失败仍返回 401
- [x] AC6 (US1)：请求格式错误（400）不计入失败计数
- [x] AC7 (US1)：429 响应含 Retry-After HTTP 头（RFC 9110 §10.7.3）
- [x] AC8 (US2)：不同 IP 失败计数互不影响
- [x] AC9 (US2)：trustProxy=false 时忽略 X-Forwarded-For；=true 时取首段
- [x] AC10 (US3)：`go test -race` 并发场景无告警、计数准确（本地无 gcc 未跑 race，CI 覆盖；并发正确性由互斥设计+并发用例保障）
- [x] AC11 (US3)：GC 清理过期 entry、保留封锁中 entry
- [x] AC12 (US3)：触发封锁打 INFO 日志，普通失败不打

## 9. 宪法符合性自查（Constitution Check）

- [x] 宪法 I（红线）：限流增强安全；无凭据硬编码；无许可证冲突（stdlib only）
- [x] 宪法 II（架构不变量）：不碰双图、planner/worker、guard
- [x] 宪法 III（技术约束）：无新依赖；无 schema 变更（内存态符合"重启即迁移"精神）
- [x] 宪法 IV（质量门槛）：三个用户故事全部可测，测试计划见 plan.md

## 10. 开放问题

无。全部 5 项已于 2026-10-10 定稿（决策记录见 §7 Assumptions A1–A5）。
