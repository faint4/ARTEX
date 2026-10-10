# Implementation Plan — {对应 spec.md 的功能名}

> 状态：Draft / Reviewed（已评审）/ Implemented
> Feature 目录：`specs/{NNN}-{feature-name}/` ｜ 上游：[spec.md](./spec.md) ｜ 关联 Issue：#{N}
> 本文档回答 HOW，承接已定稿（Final）的 spec.md。
> 模板约定（源自 github/spec-kit）：
> - 主文档保持高层可读；详细算法/代码样例放对应小节的附属文件（如需要）
> - 宪法 Gate 必须先过；违规项必须在 Complexity Tracking 表中记录理由

## Summary

{从 spec.md 提取：主需求 + 技术路线一句话}

## Technical Context

| 项 | 值 |
|---|---|
| 影响层 | 后端(Go) / 前端(Next.js) / schema / agent 工具 / 部署 |
| 语言/版本 | Go 1.26+ / Node {x} |
| 新增依赖 | {package @ version + 理由；无则写「无」} |
| 存储 | PostgreSQL 变更 or 无 |
| 测试 | go test（表驱动）/ biome |
| 性能目标 | {如 P95 < 200ms；无则写「无特殊要求」} |
| 约束 | {宪法/平台相关约束} |

## Constitution Check（Phase -1 Gates，先于任何设计）

*GATE: 必须通过才能进入任务拆解；设计完成后复查一次。*

- [ ] **红线 Gate（宪法 I）**：无未授权路径、无凭据硬编码、许可证兼容
- [ ] **架构 Gate（宪法 II）**：不破坏双图/planner-worker/guard 边界
- [ ] **简约 Gate（宪法 III）**：优先 stdlib 与既有依赖；无投机性"以后可能用"的抽象
- [ ] **防抽象 Gate（宪法 III）**：直接用框架/标准库能力，不过度包装
- [ ] **集成优先 Gate（宪法 IV）**：测试走真实 mux/真实依赖（postgres 不可用时显式 Skip）
- [ ] **迁移 Gate（宪法 III）**：schema 变更幂等、旧数据可读

## Project Structure（本 feature 触达的文件）

```text
{列出新建/修改的文件树，标注 [新增]/[修改]}
```

**Structure Decision**: {一句话说明为什么放这些位置}

## 数据模型（如涉及 schema）

```sql
-- 仅幂等语句
```

## API 约定（如涉及）

| Method | Path | 请求 | 响应 | 鉴权 |
|---|---|---|---|---|

## 测试计划（按用户故事映射）

| 故事 | 用例（来自 spec.md 验收场景） | 层级 | 位置 |
|---|---|---|---|

## Complexity Tracking（仅在 Gate 有违规时填写）

| 违规 | 为什么需要 | 被否决的更简方案及原因 |
|---|---|---|

## 风险与回退

| ID | 风险 | 严重度 | 缓解 |
|---|---|---|---|

**回退方案**：{如何撤销本 feature 的全部变更}
