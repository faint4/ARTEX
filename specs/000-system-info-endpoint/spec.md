# Feature Spec — GET /api/system/info 系统信息端点

> 状态：已定稿
> 创建：2026-10-10
> 关联：plan-spec → spec/plans/system-info-endpoint.md

## 1. 背景与问题

当前 `GET /api/health` 仅返回 `{ok, service, version}` 用于存活检查，前端登录页与设置页在排查"我跑的是什么版本/什么运行时"时缺乏专门入口。把详细诊断信息塞进 `/api/health` 会让存活检查变重（容器探针也会拉这些字段），违反单一职责。

需要一个独立的轻量诊断端点 `/api/system/info`，专门承载运行时与构建信息。存活检查 (`/api/health`) 保持原样不动。

## 2. 用户场景（Given-When-Then）

```gherkin
场景: 已登录用户查询系统信息
  Given 用户已登录（携带有效 JWT）
  When GET /api/system/info
  Then 响应 200 且 body 含字段：version, go_version, go_os, go_arch, uptime, start_time
   And version 与 /api/health.version 一致（同源 server.BuildVersion）
   And uptime ≈ now - start_time（误差 < 1s）

场景: 未登录用户访问
  Given 请求未携带 JWT 或携带无效 JWT
  When GET /api/system/info
  Then 响应 401

场景: 服务首次启动后立即调用
  Given 服务刚启动 < 1s
  When GET /api/system/info
  Then uptime 字段为 0 或正数（不得为负数）
   And start_time 不为空字符串
```

## 3. Goals / Non-Goals

- **Goals**：
  - G1：提供独立端点返回构建版本 + Go 运行时 + 进程启动信息
  - G2：与 `/api/health` 解耦，存活检查不变重
  - G3：与现有 JWT 中间件一致鉴权
- **Non-Goals**：
  - NG1：不返回任何敏感信息（环境变量、密钥指纹、文件路径、主机名）
  - NG2：不返回依赖/LLM 配置状态（那是 `/api/stats` 的职责）
  - NG3：不做缓存、不做 SSE、不做分页

## 4. 关键实体与概念

| 实体 | 含义 | 归属 |
|---|---|---|
| version | 构建版本（同 `server.BuildVersion`） | 配置（构建期注入） |
| go_version | `runtime.Version()` | 运行时 |
| go_os / go_arch | `runtime.GOOS` / `runtime.GOARCH` | 运行时 |
| start_time | 进程启动 UTC 时间 RFC3339 | 运行时（包级变量） |
| uptime | `time.Since(start)` 秒数（浮点） | 运行时（计算） |

## 5. 验收清单（可验证）

- [ ] AC1：`GET /api/system/info` 携带有效 JWT 返回 200，body 含全部 6 字段
- [ ] AC2：未携带 JWT 返回 401
- [ ] AC3：version 字段等于 `server.BuildVersion` 的当前值
- [ ] AC4：uptime 字段为非负数；连续两次调用，第二次 uptime > 第一次（单调递增）
- [ ] AC5：响应 body 不包含 env / secret / path / hostname 任何字段
- [ ] AC6：`/api/health` 响应字段不变（向后兼容）

## 6. 约束检查（对照宪法）

- [x] 不违反宪法 I（红线）：端点不暴露凭据、不引入未授权目标扫描路径
- [x] 不违反宪法 II（架构不变量）：不碰双图、不新增 planner/worker 路径
- [x] 不违反宪法 III（技术约束）：复用 `server.BuildVersion`，无新增依赖
- [x] 不违反宪法 IV（质量门槛）：表驱动测试覆盖 200/401/字段集/uptime 单调

## 7. 开放问题（需用户澄清后才能定稿）

无。状态：已定稿。
