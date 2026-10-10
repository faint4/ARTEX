# Feature Spec — Release 构建时间戳（build_time）

> 状态：Final（已定稿）
> Feature 目录：`specs/002-release-build-timestamp/` ｜ 关联 Issue：待建
> 本文档只回答 WHAT & WHY（做什么、为什么），不写 HOW（技术实现）。

## 1. 背景与问题

用户无法从界面判断当前运行的 ARTEX 构建是什么时候发布的：顶栏"版本 · dev"位置在 Release 构建下只显示 tag 号（如 `0.1.0`），无发布时间信息。仓库已具备 tag 触发的 Release 工作流（release.yml），需要把构建时间戳作为版本信息的一部分自动暴露到前端，让每次 Release 后一眼可辨构建时间。

## 2. 用户故事与验收场景（必填，按优先级排序）

### US1 — Release 构建的顶栏显示 tag + 构建时间戳 (Priority: P1)

管理员登录后查看顶栏，Release 构建显示 `版本 · {tag} · {构建时间戳}`，一眼确认当前运行的版本与发布时间。

- **为什么是 P1**：这是本需求的核心价值（发布可追溯）。
- **独立测试**：打 tag 构建（或本地以 ARTEX_BUILD_TIME 模拟注入）后启动服务，登录查看顶栏。
- **验收场景**：

```gherkin
场景: Release 构建的顶栏版本显示
  Given 一个以 tag v0.1.0 构建且注入了 build_time 的后端
  When 管理员登录并查看主界面顶栏
  Then 顶栏显示 "版本 · 0.1.0 · {YYYYMMDD-HHMM}"
  And 时间戳为 UTC 格式 YYYYMMDD-HHMM
```

### US2 — 开发构建显示保持不变 (Priority: P2)

本地开发构建（无时间戳注入）顶栏保持 `版本 · dev`，不出现冗余的 "dev · dev"。

- **为什么是 P2**：保护现有开发体验，属于约束性故事。
- **独立测试**：本地 `go build`（不注入）后查看顶栏。
- **验收场景**：

```gherkin
场景: 开发构建不显示时间戳
  Given 一个未注入 build_time（默认 "dev"）的后端
  When 管理员登录并查看主界面顶栏
  Then 顶栏显示 "版本 · dev"（与现状一致）
```

### 边界与异常（Edge Cases）

- 前端拿不到 build_time（旧后端 + 新前端）：顶栏退回只显示 `版本 · {version}`。
- /api/system/info 未认证访问：401，无信息泄露（该端点维持认证要求）。

## 3. 需求（Functional Requirements，编号化）

- **FR-001**：后端 MUST 提供独立于 version 的构建时间戳字段 `build_time`，支持构建期注入，默认值 "dev"。
- **FR-002**：`GET /api/system/info` MUST 在响应中返回 `build_time`（字段白名单 6 → 7）。
- **FR-003**：前端顶栏 MUST 在 build_time 非 "dev" 时显示 `版本 · {version} · {build_time}`；build_time 为 "dev" 时保持 `版本 · {version}`。
- **FR-004**：Release CI MUST 为所有平台矩阵注入**同一个** build_time（取 tag 指向 commit 的 UTC 时间，格式 YYYYMMDD-HHMM）。
- **FR-005**：version 字段语义 MUST 保持纯三段式 semver（selfupdate 的可比较性与一键更新不受影响）。

## 4. Goals / Non-Goals

- **Goals**：
  - G1：Release 构建可从界面辨识发布时间。
  - G2：本地/开发构建行为与现状完全一致。
- **Non-Goals**（明确不做，防发散）：
  - NG1：不把时间戳拼进 version 字符串（会破坏 selfupdate 的 semver 比较与一键更新）。
  - NG2：不修改 `/api/health` 响应形状（liveness 探针语义 + 000 spec AC6 严格字段集）。
  - NG3：release.yml Docker Hub job 的 secrets 缺失问题不在本 spec 范围（产物不受影响）。
  - NG4：不做版本历史列表、时区切换等增强。

## 5. 关键实体（Key Entities）

| 实体 | 含义（不含实现细节） | 归属 |
|---|---|---|
| build_time | 构建被编译的 UTC 时间戳（YYYYMMDD-HHMM），"dev" 表示未注入 | 配置/运行时（构建期元数据） |
| version | 现有 semver 版本号（语义不变） | 配置/运行时（构建期元数据） |

## 6. 成功标准（Success Criteria，可测量、技术无关）

- **SC-001**：Release 构建登录后顶栏可见 `版本 · 0.1.0 · {时间戳}`。
- **SC-002**：本地构建顶栏显示与改动前一致（`版本 · dev`）。
- **SC-003**：`/api/health` 响应字段集保持 `{ok, service, version}` 不变。
- **SC-004**：selfupdate 版本比较行为不受影响（version 仍为纯 semver）。

## 7. 假设（Assumptions）

- 时间戳格式采用紧凑的 `YYYYMMDD-HHMM`（UTC），排序友好且无时区歧义。
- 五平台矩阵注入同一时间戳（来源 tag 指向的 commit 时间，而非各 job 的本地时钟）。
- 前端 main-content 顶栏已登录环境，可调用认证端点 /api/system/info。

## 8. 验收清单（Acceptance Criteria，映射用户故事）

- [ ] AC1（US1）：Release 注入构建的 /api/system/info 返回 build_time == 注入值，且字段集为 7 个（原 6 个 + build_time）
- [ ] AC2（US1）：前端顶栏显示 `版本 · {version} · {build_time}`
- [ ] AC3（US2）：未注入构建的 build_time == "dev"，前端顶栏显示 `版本 · {version}`（无时间戳段）
- [ ] AC4（约束）：/api/health 字段集仍为 {ok, service, version}（000 spec AC6 不破）
- [ ] AC5（约束）：CompareVersions 对含时间戳/无时间戳的 version 行为不变（selfupdate 测试全绿）
- [ ] AC6（FR-004）：release.yml 五平台产物携带同一 build_time

## 9. 宪法符合性自查（Constitution Check）

- [x] 宪法 I（红线）：不引入未授权路径/凭据硬编码/许可证冲突
- [x] 宪法 II（架构不变量）：不碰双图/planner/worker/guard 边界（纯构建元数据 + 展示层）
- [x] 宪法 III（技术约束）：无新增依赖；无 schema 变更
- [x] 宪法 IV（质量门槛）：每个用户故事可测（AC1-AC6 全部可自动化断言）

## 10. 开放问题（[NEEDS CLARIFICATION] 汇总）

| ID | 问题 | 默认假设 | 影响范围 |
|---|---|---|---|
| （无）| 用户已确认：显示语义 = tag + 时间戳并存；首个 Release = v0.1.0 | — | — |

定稿规则：本表清空（或用户确认默认假设）后状态改 Final，方可进入 plan.md。
