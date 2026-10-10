# Feature Spec — {功能名称}

> 状态：Draft / Final（已定稿）/ Implemented
> Feature 目录：`specs/{NNN}-{feature-name}/` ｜ 关联 Issue：#{N}
> 本文档只回答 WHAT & WHY（做什么、为什么），不写 HOW（技术实现）。
> 模板约定（源自 github/spec-kit）：
> - 不确定的内容用 `[NEEDS CLARIFICATION: 问题]` 内联标记，禁止靠猜补全
> - 需求编号化（FR-xxx / SC-xxx），每条必须可测
> - 用户故事按优先级排序（P1 最先），每个故事独立可测、独立可交付（MVP 切片）

## 1. 背景与问题

一段话说清：用户遇到了什么问题，为什么现在做。

## 2. 用户故事与验收场景（必填，按优先级排序）

### US1 — {一句话标题} (Priority: P1)

{用平实语言描述这条用户旅程}

- **为什么是 P1**：{价值与优先级理由}
- **独立测试**：{只实现本故事时如何独立验证}
- **验收场景**：

```gherkin
场景: {场景名}
  Given {前置状态}
  When {用户/系统动作}
  Then {可观察的结果}
```

### US2 — {一句话标题} (Priority: P2)

（同上结构；P2 起的故事可与 US1 独立交付）

### 边界与异常（Edge Cases）

- {边界条件发生时会发生什么？}
- {错误场景系统如何处理？}

## 3. 需求（Functional Requirements，编号化）

- **FR-001**：系统 MUST {具体能力}
- **FR-002**：系统 MUST {具体能力}
- **FR-00x**：系统 MUST {含 [NEEDS CLARIFICATION] 的条目在定稿前必须全部消解}

## 4. Goals / Non-Goals

- **Goals**：
  - G1 …
- **Non-Goals**（明确不做，防发散）：
  - NG1 …

## 5. 关键实体（Key Entities）

| 实体 | 含义（不含实现细节） | 归属（资产图/探索图/配置/运行时） |
|---|---|---|

## 6. 成功标准（Success Criteria，可测量、技术无关）

- **SC-001**：{可测量指标，如"管理员可在 X 分钟内完成 Y"}
- **SC-002**：{性能/可靠性指标，如"P95 < N ms"}

## 7. 假设（Assumptions）

- {关于用户/环境的假设，如"默认单实例部署"}
- {范围假设，如"移动端不在 v1 范围"}
- {对既有系统的依赖，如"复用现有 JWT 中间件"}

## 8. 验收清单（Acceptance Criteria，映射用户故事）

- [ ] AC1：{可观察、可测量；标注来源 US}
- [ ] AC2：…

## 9. 宪法符合性自查（Constitution Check）

- [ ] 宪法 I（红线）：不引入未授权路径/凭据硬编码/许可证冲突
- [ ] 宪法 II（架构不变量）：不碰双图/planner/worker/guard 边界
- [ ] 宪法 III（技术约束）：无新增依赖或已说明理由；schema 幂等
- [ ] 宪法 IV（质量门槛）：每个用户故事可测

## 10. 开放问题（[NEEDS CLARIFICATION] 汇总）

| ID | 问题 | 默认假设 | 影响范围 |
|---|---|---|---|

定稿规则：本表清空（或用户确认默认假设）后状态改 Final，方可进入 plan.md。
