## 关联

- Issue: #{N}（合并时 `Closes #N` 自动关闭）
- Spec: `specs/{NNN}-{name}/spec.md`（bug 修复写「轻量路径，免 spec」）

## 变更说明

<!-- 一段话说清做了什么、为什么；bug 修复必须写确切根因 -->

## SDD 工件（feature 必填，bug 可跳过）

- [ ] spec.md 状态已更新（Draft → Final → Implemented）
- [ ] plan.md 的 Constitution Gates 已过
- [ ] tasks.md 已逐项勾选
- [ ] 编号可追溯：FR-xxx/ACx 映射到测试用例名

## 自测清单（宪法 IV 质量门槛）

- [ ] `go test -race ./...` 全绿（新增逻辑有表驱动测试，测试先红后绿）
- [ ] `go vet ./...` 无告警；`gofmt -l` 为空
- [ ] 前端改动：`biome check` 通过 + `npm run build:static` 成功
- [ ] schema 改动：幂等、旧数据可读、连续两次启动无错
- [ ] spec/README/CHANGELOG 与实现同步

## Code Review

- [ ] CR 报告已存 `cr_reports/{feature-NNN}.md`，逐条对照 AC 验收
