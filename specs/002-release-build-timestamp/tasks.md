# Tasks — Release 构建时间戳（build_time）

> Spec：`specs/002-release-build-timestamp/spec.md` ｜ Plan：`plan.md`
> 分支：`002-release-build-timestamp`

## Phase 1 — Spec（本文档所在目录）

- [x] T001 用户决策回收（显示语义 A：tag+时间戳并存；首版 v0.1.0）→ spec.md 定稿
- [x] T002 plan.md（触点清单 + 契约修订记录 + Gates）

## Phase 2 — 后端与构建链

- [x] T003 `build.sh`：`ARTEX_BUILD_TIME` 支持（默认 `date -u +%Y%m%d-%H%M`）+ ldflags 追加 `-X main.buildTime=`
- [x] T004 `.github/workflows/release.yml`：binaries job 注入 `ARTEX_BUILD_TIME`（head_commit.timestamp → YYYYMMDD-HHMM，五平台一致）
- [x] T005 `cmd/artex/main.go`：`var buildTime = "dev"` + `server.BuildTime = buildTime`
- [x] T006 `server/server.go`：`var BuildTime = "dev"`
- [x] T007 `server/system_info.go`：响应增加 `build_time`，白名单注释 6→7
- [x] T008 `server/system_info_test.go`：AC1 字段集 6→7；`build_time == BuildTime` 断言；全局态保存/恢复

## Phase 3 — 前端

- [x] T009 `web/src/lib/api.ts`（或 types）：system/info 类型补 `build_time`
- [x] T010 `web/src/app/(main)/_components/main-content.tsx`：取数切 system/info；显示 `版本 · {version}` + 条件 ` · {build_time}`（"dev" 省略）

## Phase 4 — 验证与交付

- [x] T011 gofmt + go vet + 容器内 `-race -p 1` 全量回归（含 system_info / selfupdate）
- [x] T012 web `build:static` 通过（biome + 类型检查）（本地 tsc --noEmit 通过；完整 build 受 Windows/Linux 原生二进制混装限制在 CI 验证）
- [ ] T013 commit → push → PR → CI 绿 → 合并 main
- [ ] T014 tag `v0.1.0` push → release.yml 触发 → 验证 GitHub Release 产物（zip + SHA256SUMS）
- [ ] T015 验证产物内二进制 `-h` 正常、顶栏显示 `版本 · 0.1.0 · {时间戳}`
