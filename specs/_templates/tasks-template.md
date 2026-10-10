# Tasks — {对应 spec.md 的功能名}

> 上游：[plan.md](./plan.md) ｜ 关联 Issue：#{N}
> 拆解约定（源自 github/spec-kit）：
> - 任务按用户故事分阶段；每个故事独立可测、可独立交付（MVP 递增）
> - `[P]` = 可并行（不同文件、无依赖）；`[US1]` 等标记任务归属故事，保证可追溯
> - 测试任务先于实现任务（TDD：先红后绿）
> - 每个阶段末尾有 Checkpoint；不过则停，禁止带着失败进入下一阶段

## Format: `[Txxx] [P?] [US?] 描述（含确切文件路径）`

## Phase 1: Setup（共享基建）

- [ ] T001 {依赖/工具/配置变更} — 验收：{命令} 通过

**Checkpoint**: {基建就绪的判定}

---

## Phase 2: Foundational（阻塞性前置）

**⚠️ CRITICAL**: 本阶段未完成前，任何用户故事不得开工

- [ ] T002 {所有故事依赖的基础设施，如类型定义、字段注入} — 验收：{}

**Checkpoint**: {基础设施就绪判定}

---

## Phase 3: US1 — {故事标题} (Priority: P1) 🎯 MVP

**Goal**: {本故事交付什么}
**Independent Test**: {独立验证方式}

### Tests First（先红后绿）

- [ ] T003 [P] [US1] {单元测试：状态机/核心逻辑} in `server/xxx_test.go` — 验收：编译通过、用例红
- [ ] T004 [US1] {集成测试：走完整 mux} in `server/auth_test.go` — 验收：红

### Implementation

- [ ] T005 [US1] {核心实现} in `server/xxx.go` — 验收：T003/T004 转绿

**Checkpoint**: US1 独立可用，`go test ./server/ -run {Xxx}` 全绿

---

## Phase 4: US2 — {故事标题} (Priority: P2)

（同上结构；可含对 US1 组件的集成，但保持独立可测）

**Checkpoint**: US1 + US2 独立可用

---

## Phase 5: US3 — {故事标题} (Priority: P3)

（同上结构）

**Checkpoint**: 全部故事独立可用

---

## Phase 6: Polish & 收尾

- [ ] Txxx [P] 文档同步（spec/README/CHANGELOG）
- [ ] Txxx 全量回归 `go test -race ./...` + `go vet` + `gofmt`
- [ ] Txxx Code Review（对照 spec 验收清单），报告存 `cr_reports/{commit}.md`

---

## Dependencies & Execution Order

- Setup → Foundational（阻塞一切）→ US1 → US2 → US3 → Polish
- 同故事内：测试先行 → 模型/核心 → 集成 → 收尾
- `[P]` 任务可并行；跨故事默认串行（单人维护），必要时按优先级穿插

## Notes

- 每个 commit 引用 Issue（`ref #N`），合并时 `Closes #N`
- 发现任务超出 plan.md 范围：停下来回改 plan，禁止顺手做
- Checkpoint 处跑一次全量测试，防回归累积
