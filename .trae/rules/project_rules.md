# Project Rules（ARTEX — Specification Driven Development）

### 项目宪法（最高优先级）

- **描述**：定位红线、双图架构不变量、技术约束、质量门槛、fork 治理、spec-kit 开发流与 Governance
- **加载规则文件**：`specs/constitution.md`
- **生效范围**：全局生效，优先于本文件与其余所有规则

### SDD 工作流（全局约束，采用 github/spec-kit 方法论）

- 新功能：`specs/{NNN}-{name}/spec.md`（用户故事 P1/P2/P3 切片，[NEEDS CLARIFICATION] 消解后定稿）→ `plan.md`（先过 Constitution Check Gates，违规记 Complexity Tracking）→ `tasks.md`（按用户故事分阶段，TDD 先红后绿）→ Analyze（只读一致性检查）→ 实现 → PR（spec+代码+测试+CR 报告同 PR，CI 必绿，Closes #Issue）。
- 纯缺陷修复：可免 spec.md（轻量路径），但必须有回归测试，并在 PR 描述中写明根因。
- 修改 schema / API 路由 / agent 工具集时，必须同步更新 `specs/` 下对应文档与 README。
- 发现实现偏离 spec：停下来修 spec 或修实现，二选一，禁止静默漂移。
- spec/plan/tasks 中的编号（FR-xxx/SC-xxx/USx/ACx）必须贯穿到测试用例命名，保证可追溯。

### 项目概述

- **描述**：技术栈、架构分层、目录结构、常用命令
- **加载规则文件**：`rules/project_overview.md`
- **生效范围**：全局生效

### 代码风格规范

- **描述**：Go 后端代码风格与工程约束
- **加载规则文件**：`rules/code_style_go.md`
- **生效范围**：仅用于 `*.go` 文件

- **描述**：前端 TypeScript / React 代码风格
- **加载规则文件**：`rules/code_style_ts.md`
- **生效范围**：仅用于 `web/` 下 `*.ts`、`*.tsx` 文件

### 代码审查

- **描述**：每次提交对照宪法与本次 spec/plan/tasks 做系统化审查，产出标准化报告
- **报告位置**：`cr_reports/{feature-NNN-or-commit}.md`
- **生效范围**：合并前必须完成；报告需覆盖变更分析、功能正确性（逐条对照 spec.md 的 AC 验收清单）、架构一致性（对照宪法 II）、测试覆盖（含 -race）

### GitHub 维护约定

- Issue 模板：feature（映射 spec.md 结构）/ bug（轻量路径）；feature Issue 创建后回填到 spec.md 与 tasks.md 的「关联 Issue」。
- 分支：feature 用 `{NNN}-{feature-name}`（与 specs 目录同名）；bug 用 `fix/{issue}-{slug}`。
- PR 模板强制：关联 Issue、spec 链接、自测清单、CR 报告链接、宪法自查。
- Labels：`type:feature` / `type:bug` / `status:spec-draft` / `status:spec-final` / `status:in-progress` / `status:done`。
- main 分支保护：禁止直推、要求 PR + CI 绿。
