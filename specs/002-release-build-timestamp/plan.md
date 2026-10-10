# Plan Spec — Release 构建时间戳（build_time）

> 状态：Final ｜ Spec：`specs/002-release-build-timestamp/spec.md`
> 本文档回答 HOW：技术方案、触点清单、契约修订与风险。

## 1. 方案总览

构建期元数据走与 version 完全同构的 ldflags 注入链路，新增平行字段 build_time：

```
release.yml / build.sh（ARTEX_BUILD_TIME）
  → go build -ldflags "-X main.buildTime=<YYYYMMDD-HHMM>"
    → cmd/artex: var buildTime = "dev" → server.BuildTime
      → GET /api/system/info 响应新增 "build_time"（字段白名单 6→7）
        → web main-content.tsx 顶栏: 版本 · {version} · {build_time|省略}
```

## 2. 触点清单（文件级）

| 文件 | 变更 |
|---|---|
| `build.sh` | 新增 `ARTEX_BUILD_TIME`（默认 `date -u +%Y%m%d-%H%M`），ldflags 追加 `-X main.buildTime=${ARTEX_BUILD_TIME}` |
| `.github/workflows/release.yml` | binaries job 增 `ARTEX_BUILD_TIME` env：由 `${{ github.event.head_commit.timestamp }}`（ISO8601 UTC）转换为 YYYYMMDD-HHMM —— 保证五平台矩阵一致（同一 commit 时间，而非各 runner 本地时钟） |
| `cmd/artex/main.go` | `var buildTime = "dev"`（ldflags 注入点，与 version 同款注释）；启动时 `server.BuildTime = buildTime` |
| `server/server.go` | `var BuildTime = "dev"`（与 BuildVersion 并列，同款注释） |
| `server/system_info.go` | 响应 map 增加 `"build_time": BuildTime`；字段白名单注释 6→7 |
| `server/system_info_test.go` | AC1 断言字段集 6→7；新增断言 `build_time == BuildTime`（对齐 AC3 模式）；保存/恢复 BuildTime 全局态 |
| `web/src/lib/api.ts` | `systemInfo()` 返回类型增加 `build_time: string`（确认现有 systemInfo API 封装位置；若无则新增取数函数） |
| `web/src/app/(main)/_components/main-content.tsx` | 取数从 `api.health()` 改为 system/info（一次请求同时拿 version + build_time）；显示 `版本 · {version}` + 条件段 ` · {build_time}`（build_time 为 "dev" 时省略） |

## 3. 契约修订记录（对既有 spec 的影响）

- **000-system-info-endpoint**：/api/system/info 字段白名单 6 → 7（新增 build_time，AC1/AC5 断言同步）。修订理由：构建时间戳属于"系统信息"语义；封闭集测试需同步放行。修订方式：002 的测试改动即为权威实现，000 spec 文档正文不再回改（历史 spec 保持不可变，演进以测试 + 本文档 §3 为准）。
- **/api/health**：**不变**（000 spec AC6 严格字段集 + liveness 极简语义；本 spec NG2）。
- **selfupdate**：**不变**（version 语义不动，CompareVersions 无涉）。

## 4. Constitution Check Gates

1. **红线**：无凭据/路径泄露——build_time 是无敏感性的编译时间；system/info 保持认证门槛 ✓
2. **架构不变量**：不触碰双图/planner/worker/guard，纯元数据透传 ✓
3. **技术约束**：零新增依赖；零 schema 变更；ldflags 注入模式与 version 完全同构 ✓
4. **质量门槛**：AC1-AC6 均可测试断言（后端 HTTP 测试 + 前端类型检查 + release 产物冒烟）✓

## 5. 风险与权衡

- **前端取数端点切换**（health → system/info）：system/info 需认证——main-content 仅渲染于已登录区，无影响；health() 调用在别处（update-card 轮询）保持不动。
- **旧前端 + 新后端**：多一个字段无害（前端不读即忽略）；**新前端 + 旧后端**：build_time 缺失 → 显示退化为 `版本 · {version}`（组件内容错）。
- **矩阵时间一致性**：用 head_commit.timestamp 而非 runner date；本地构建用 build.sh 默认（date -u），两者语义一致（编译时间近似值）。
- **显示格式**：YYYYMMDD-HHMM（UTC）紧凑、排序友好；前端原样展示不做时区换算（NG4）。

## 6. 测试计划

- 后端：扩展 `TestSystemInfoEndpoint`（字段集 7、build_time == BuildTime、保存/恢复全局态）；CI `-race -p 1` 全量回归。
- 前端：biome + `build:static` 通过（类型层面保证 build_time 字段存在性）。
- Release 冒烟：release.yml 自带 `dist/artex-linux-amd64/artex -h`；合并后打 tag v0.1.0，验证 Release 产物 + 顶栏显示。
