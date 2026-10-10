# ARTEX 项目宪法

本宪法优先于所有其他开发实践和指导原则。任何 Feature / Plan / Task 都不得违背本文条款；确需突破时，先修订本宪法并说明理由，再进入开发。

## I. 定位与红线（不可协商）

- ARTEX 是 **LLM 多 agent 驱动的自主渗透测试系统**，仅用于**本地隔离环境**的技术学习、研究与验证。
- 严禁任何将本工具用于未授权目标扫描、探测、利用的代码、文档或默认配置进入主干。
- 遵循 **AGPL-3.0**：衍生/网络服务化必须开源；引入第三方依赖前必须确认协议兼容（禁 AGPL 冲突协议、禁无 License 的复制代码）。
- 凭据与密钥（API Key、数据库密码、JWT key）不得硬编码；一律走环境变量或运行时配置。

## II. 架构不变量

1. **双图架构**是核心资产模型：
   - 资产图（assets/companies/task_scope）：**全局共享**的资产真值库，去重 key 由程序计算，agent 只提交原始信息；
   - 探索图（exploration_nodes/anchors/activity）：**每任务独立**的推进链（goal/intent/fact/finding/hint）；
   - 两图只能通过 `exploration_anchors` 锚点相连，禁止在资产图上写任务私有状态。
2. **planner / worker 分工**：planner 是唯一意图生成者，读态势、派意图进 frontier；worker 领一条意图、执行、写回两图后即停。禁止 worker 自行规划新意图。
3. **事件驱动闭环**：引擎由图变更（debounce）唤醒，不引入轮询循环。
4. **人在环路**：高危工具调用必须经过 guard/intercept 审批门；不得新增绕过审批的执行路径。
5. **流量留痕**：worker 的 HTTP 出网默认走记录代理（MITM + CA），新工具接入必须保持可审计。

## III. 技术约束

- **后端**：Go 1.26+，单二进制交付；前端静态导出经 `go:embed` 内嵌（`-tags embedui`）。
- **存储**：PostgreSQL（pgx）；schema 变更只允许修改 `db/schema.sql` 的幂等语句（`ADD COLUMN` / `CREATE INDEX IF NOT EXISTS`），**重启即迁移**，禁止要求手动迁移步骤。
- **Agent 能力**：基于 `norma` SDK（agentcore/tool/permission/harness/memory/transcript），不重复造轮子。
- **前端**：Next.js 静态导出 + shadcn/ui；不得引入 SSR-only 特性破坏 `next build` 静态导出。
- **依赖**：新增 Go/npm 依赖需在 plan-spec 中说明理由；优先 stdlib 与既有依赖。

## IV. 质量门槛

- `go test ./...` 全绿；新增/修改的后端逻辑必须带表驱动测试。
- 前端 `biome check` 通过；类型错误视为编译失败。
- 涉及 schema、API 路由、agent 工具集的变更，必须同步更新对应 spec 与 README，否则不得合并。
- 演示模式（demo）不得请求真实目标。

## V. Fork 治理

- 上游 `Hinln/ARTEX` 已转私有；**本 fork（faint4/ARTEX）是事实主干**，独立演进。
- 页面一键更新、Release 工作流指向本仓库；发版用 `v*` tag 触发 `.github/workflows/release.yml`，附件含 `SHA256SUMS`。
- 镜像发布不受 Docker Hub 上游仓库影响：本仓库自建镜像流程，或使用本地构建（`web build:static` → embed → `go build`）。

## VI. 开发流（Specification First，采用 github/spec-kit 方法论）

```
GitHub Issue（需求入口，feature/bug 模板）
  → specs/{NNN}-{name}/spec.md（WHAT/WHY；不确定处用 [NEEDS CLARIFICATION] 标记，
     澄清后定稿 Final；用户故事按 P1/P2/P3 优先级切片，各自独立可测）
  → specs/{NNN}-{name}/plan.md（HOW；先过 Constitution Check Gates，
     违规记录进 Complexity Tracking）
  → specs/{NNN}-{name}/tasks.md（按用户故事分阶段拆解，[P] 并行标记，TDD 先红后绿）
  → Analyze（只读跨工件一致性检查：spec/plan/tasks 覆盖齐、术语一致、无缺口）
  → 实现 + 测试（每个 Checkpoint 跑全量测试）
  → GitHub PR（spec + 代码 + 测试 + CR 报告同 PR；CI 必绿；合并时 Closes #Issue）
  → Code Review 报告存 cr_reports/{commit}.md
```

- 分支命名：`{NNN}-{feature-name}`（与 specs 目录同名）或 `fix/{issue}-{slug}`（轻量路径）。
- 轻量路径：纯缺陷修复可省略 spec.md，但必须有回归测试 + 根因说明（写入 PR 描述）。
- spec-kit 的 FR-xxx/SC-xxx/USx 编号贯穿 spec → plan → tasks → 测试用例，保证可追溯。

## Governance（宪法治理）

- 本宪法优先于一切实践与规范；冲突时以宪法为准，不得在 spec/plan 中稀释或重释条款。
- **修订流程**：修改宪法必须 ① 在 PR 中说明修订理由 ② 评估对既有 specs 的向后兼容影响 ③ 更新版本号与修订日期 ④ 迁移方案（如有）。
- 宪法冲突在 Analyze 阶段自动升级为 CRITICAL——必须调整 spec/plan/tasks，而不是绕过原则。
- 所有 PR / Review 必须核对宪法符合性；复杂度必须在 Complexity Tracking 中显式论证。

版本：2.0.0 | 批准：2026-10-10 | 最近修订：2026-10-10（v2.0.0：引入 spec-kit 目录约定、Gate 检查、用户故事切片与 Governance 机制）
