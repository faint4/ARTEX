# 前端 TypeScript / React 代码风格（web/）

本规范用于 `web/`（Next.js App Router + shadcn/ui + biome）。

## 核心原则

### I. 静态导出兼容（不可协商）
- 不得使用 SSR-only 特性（server actions、动态路由服务端渲染、`getServerSideProps`）。
- 鉴权态从 localStorage 的 `artex_token` 读取（见 `web/src/lib/api.ts`），API 走同源 `/api/*`。
- SSE 用 `lib/sseUrl` 产出的同源地址；不要写死 `:8787`。

### II. 组件化优先
- 页面只做编排，业务拆成可复用组件；组件单一职责、props 显式类型。
- UI 基建统一用 `components/ui/*`（shadcn）；新组件先查是否已有等价物。

### III. 类型安全
- 接口响应类型集中在 `lib/types.ts`；禁止 `any`（biome 已约束）。
- API 调用统一走 `lib/api.ts` 的 `http` 封装（带 token、统一 401 跳转、统一错误消息），组件内不得直接 `fetch("/api/...")`。

## 代码风格

- 命名：组件/类型 PascalCase，变量/函数 camelCase，常量 SCREAMING_SNAKE_CASE。
- 组件文件 PascalCase.tsx；hooks `useXxx.ts`；工具函数 camelCase.ts。
- 服务端数据用 SWR/既有数据获取模式；列表页必须有加载/空/错误三态。
- 样式用 Tailwind 语义类，与主题变量（`--primary` 等）保持一致，禁止硬编码色值。

## 错误与反馈

- 用户可读错误来自后端 `error` 字段，用既有 toast/dialog 组件展示；不得静默失败。
- 破坏性操作（删除任务、停止会话）需二次确认。

## 提交前检查

- [ ] `biome check` 通过（lint + format）
- [ ] `npm run build:static` 成功
- [ ] 新页面在「演示模式」（`NEXT_PUBLIC_MOCK=1`）下不请求真实后端也能渲染
