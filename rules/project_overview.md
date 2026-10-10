# ARTEX 项目概述

## 定位

LLM 多 agent 驱动的自主渗透测试系统。Go 单体后端（内嵌 Next.js 前端）+ PostgreSQL。
仅限本地隔离环境学习研究用（宪法 I）。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.26+、net/http、JWT、SSE、pgx(v5)、norma SDK（agent 能力） |
| 前端 | Next.js 静态导出、TypeScript、shadcn/ui、biome |
| 存储 | PostgreSQL（schema 经 `go:embed` 幂等建表） |
| 流量 | go-mitmproxy（记录代理，MITM + CA） |
| 部署 | 单二进制 / Docker Compose（本仓库自建镜像） |

## 目录结构与职责

```
agent/        goals(目标分解) planner(唯一意图生成者) worker(意图执行) mainagent(人在环路)
server/       REST /api/* + JWT + SSE；Manager 托管任务/引擎生命周期
db/           schema.sql（幂等迁移）+ 双图存储层（pgx）
engine 相关   事件驱动闭环：图变更 → 唤醒 planner → 派意图 → worker 领取执行 → 写回
traffic/      MITM 记录代理（:8788）
guard/intercept/  工具审批门（高危操作必须经此）
enrich/       DNS/HTTP 异步补全
report/ notify/ skills/ llmpool/ mcphttp/ sidequestion/ selfupdate/
web/          Next.js 前端（npm run build:static 产出 → server/webui/dist 内嵌）
specs/          ├─ constitution.md（宪法 v2.0，含 Governance）
                ├─ _templates/（spec/plan/tasks 模板，源自 github/spec-kit 方法论）
                ├─ 000-system-info-endpoint/（已完成 demo）
                └─ 001-login-rate-limit/（进行中）
rules/        代码规范（由 .trae/rules/project_rules.md 索引）
cr_reports/   Code Review 报告（对照 specs/{NNN}/spec.md 的 AC 清单）
```

## 双图架构速记（详见 specs/constitution.md II）

- 资产图：全局共享真值库（root_domain/subdomain/ip/service/app/endpoint）
- 探索图：每任务独立推进链（goal/intent/fact/finding/hint + 血缘边）
- 连接：仅通过 `exploration_anchors` 锚点

## 常用命令

```bash
# 后端测试
go test ./...

# 前端
cd web && npm ci && npm run build:static   # 静态导出到 web/out
cp -r web/out server/webui/dist            # 拷入内嵌目录

# 单二进制编译（内嵌前端）
CGO_ENABLED=0 go build -tags embedui -o artex ./cmd/artex

# 本地开发（后端:8787 代理:8788 前端:5173）
./dev.sh

# 本地运行（WSL，容器化 artex-app + artex-postgres）
docker start artex-postgres artex-app
```

## 环境配置

- 数据库：环境变量 `ARTEX_PG_DSN` 优先于 config.json
- LLM：`ANTHROPIC_API_KEY` / `OPENAI_API_KEY`，或 UI「LLM 配置」页（含 OpenAI 兼容中转 base_url）
- 端口：`-addr :8787`（前端+API）、`-proxy :8788`（流量记录）
