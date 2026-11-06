---
status: draft
revision: web-spa-design-ir02-r01
feature: web-spa
source_requirements: requirements-ir02-web-spa-r03
source_boundary: web-spa-boundary-ir02-r02
last_updated: 2026-11-06
---

# Han Web 前端（独立 SPA）详细设计

> **产出说明**：本 Host 无法拉起 `add-feature-designer` Role，本文由主会话按 `add-design-tdd` 产出，**没有该 Role 的独立产出**，也无独立设计评审。

## 1. 目标与范围

在**不改动既有认证与授权合同**的前提下，为已交付的 12 个业务能力提供浏览器界面：注册、登录、登出、文章列表（游标分页）、文章详情、我的文章（新建/编辑/删除）。新增后端能力仅一项：**登出端点**（需求 r03 Q1 已批准）。

技术方向（用户 2026-11-06 确认）：React + TypeScript + Vite，独立构建、同源部署。

不在本设计范围：管理员治理界面、密码自助、SSR/SEO、原生客户端、国际化。

## 2. 现有系统调查（事实，非假设）

| 事实 | 证据 | 对本设计的约束 |
|---|---|---|
| 会话 Cookie `__Host-han_session` | `internal/platform/http_security.go:16` | 必须 HTTPS + 同源；前端不得改名或复制 |
| `SameSite=Lax` | 同上 `:274` | 跨站 XHR 不带 Cookie → 只能同源 |
| CSRF 头 `X-CSRF-Token` | 同上 `:18` | 所有非安全方法必须携带 |
| 登录返回 `csrf_token` | `internal/identity/handler.go:1530` | 前端登录后取该字段存内存 |
| `/api/v1/sessions` 仅 `POST` | `cmd/han-server/main.go:3570-3573` | 无登出/查询端点 → 本设计新增登出 |
| 列表响应为 `items` + `next_cursor` | `internal/article/admin_query_test.go:569`（反向断言） | 前端按游标分页，不用页码 |
| 列表项投影 | `internal/article/model.go:291-298` | `id/title/author{kind,id,label}/created_at/updated_at` |
| 详情投影 | `internal/article/model.go:300-311` | 追加 `body/publication_state/violation_state` |
| 作者视图 | `internal/article/model.go:322-327` | `title/body/moderation_status/editable`，编辑页据此预填 |
| 错误响应 | `internal/platform/errors.go:164-165` | `{code, message}`，前端只做展示映射 |
| 无 HTML 模板/静态托管 | 全仓 `html/template`、`FileServer` 零命中 | 前端产物必须由反代托管，Go 服务不改 |

## 3. UX 交互

### 3.1 路由与页面

| 路由 | 页面 | 登录要求 | 主要动作 |
|---|---|---|---|
| `/register` | 注册 | 否 | 填写、确认隐私声明、提交 |
| `/login` | 登录 | 否 | 提交用户名与密码 |
| `/` | 文章列表 | 是 | 翻页、进入详情 |
| `/articles/:id` | 文章详情 | 是 | 进入编辑（仅本人） |
| `/me/articles` | 我的文章 | 是 | 新建入口、编辑、删除 |
| `/me/articles/new` | 新建文章 | 是 | 标题 + 正文 |
| `/me/articles/:id/edit` | 编辑文章 | 是 | 预填、保存 |

未登录访问受保护路由 → 重定向 `/login` 并保留目标路径；登录成功回到该路径。

### 3.2 状态模型

前端只有三个会话态：`unknown`（启动探测中）、`anonymous`、`authenticated`。

- 启动：置 `unknown`，发起一次受保护只读请求；401 → `anonymous`；成功 → `authenticated`。
- 任一请求收到 401 → 立即 `anonymous` + 清空 CSRF token + 重定向登录。
- 登出成功 → `anonymous`。

**不持久化**：会话、CSRF token、用户标识一律不写 `localStorage` / `sessionStorage`。

### 3.3 交互约定

| 场景 | 行为 |
|---|---|
| 提交中 | 按钮禁用 + 行内加载态，防重复提交 |
| 表单校验失败 | 字段旁提示；后端 422 的字段错误映射到对应字段 |
| 网络失败 | 顶部错误条 + 重试按钮；已输入内容保留 |
| 列表空 | 空状态文案，不显示错误 |
| 删除 | 二次确认弹层，明确"不可恢复" |
| 401 | 回登录页，提示"登录已失效" |
| 403 | 提示"没有权限"，不跳转 |
| 404 | 提示"内容不可用"，列表页可返回 |

### 3.4 可访问性与响应式

- 关键流程可用键盘完成（Tab 顺序、Enter 提交、Esc 关闭确认层）。
- 断点：≥1024px 三栏列表、<1024px 单栏；最小支持宽度 360px。
- 不做 WCAG AA 全量达标（需求范围外），但表单必须有 `<label>`、焦点可见。

## 4. API 契约

### 4.1 前端消费的现有端点

| 用途 | 方法与路径 | 关键字段 |
|---|---|---|
| 登录 | `POST /api/v1/sessions` | 请求 `{username, password}`；响应含 `csrf_token` |
| 列表 | `GET /api/v1/articles?limit=20&cursor=<opaque>` | 响应 `{items:[...], next_cursor}` |
| 详情 | `GET /api/v1/articles/{id}` | 见 §2 详情投影 |
| 创建 | `POST /api/v1/articles` | 请求 `{title, body}` |
| 编辑 | `GET/PUT /api/v1/articles/{id}`（作者视图） | `title/body/moderation_status/editable` |
| 删除 | `DELETE /api/v1/articles/{id}` | 成功语义以后端为准 |
| 我的文章 | `GET /api/v1/users/me/articles` | 与列表同构 |

> 上述路径与字段来自代码实测；具体请求体字段名与状态码在实施 G0 时逐条核对 `internal/article/handler.go`，不符即回本设计修正，不在实现中私自改契约。

### 4.2 新增：登出端点（本设计唯一的后端变更）

| 项 | 设计 |
|---|---|
| 方法与路径 | `POST /api/v1/sessions/logout` |
| 认证 | 需要有效会话；同样校验 `X-CSRF-Token` |
| 语义 | 撤销当前会话（服务端失效），幂等：已失效会话再次调用返回 204 |
| 成功 | `204 No Content`，并下发过期 Set-Cookie 清除客户端 Cookie |
| 未认证 | `401`，沿用统一错误体 |
| 影响面 | 新增 handler + 路由 + 契约测试；不改既有端点行为 |

### 4.3 错误映射

| HTTP | 前端表现 |
|---|---|
| 400/422 | 表单内联提示（按 `code` 分类，未知码显示通用文案） |
| 401 | 回登录页 |
| 403 | "没有权限"提示，停留当前页 |
| 404 | "内容不可用" |
| 409 | "内容已被修改，请刷新后重试"（编辑冲突） |
| 5xx / 网络 | 错误条 + 重试 |

## 5. 实现结构

### 5.1 目录

```
web/
  index.html
  package.json
  tsconfig.json
  vite.config.ts
  src/
    main.tsx          应用入口与路由挂载
    app/
      router.tsx       路由表与登录守卫
      session.tsx      会话态与 CSRF 持有（内存）
    api/
      client.ts        fetch 封装：同源、credentials、CSRF、错误归一
      articles.ts      文章与「我的文章」调用
      auth.ts          登录/登出
    pages/            register / login / list / detail / my-articles / editor
    components/       表单控件、错误条、空状态、确认层、分页器
    lib/              格式化、校验提示映射
```

### 5.2 关键机制

- **统一客户端**：`api/client.ts` 负责 `credentials: "same-origin"`、注入 `X-CSRF-Token`、把 `{code,message}` 归一为 `ApiError`，并在 401 时触发会话失效。
- **会话持有**：`session.tsx` 用 React context 保存 `{state, csrfToken}`；token 只存在内存变量中，刷新即丢，靠启动探测恢复登录态。
- **登录态探测**：`GET /api/v1/articles?limit=1`；不新增端点（Q2 决定）。探测失败（网络错误）显示"无法连接"，不误判为未登录。
- **路由守卫**：受保护路由在 `state==="unknown"` 时显示骨架屏，避免闪烁跳转。
- **数据获取**：使用 `fetch` + 组件内状态，不引入数据缓存库；写操作成功后按需重新拉取列表/详情（数据量小、失效简单）。
- **表单**：受控组件 + 最少量前端校验（必填、长度）；业务规则以后端为准，不复制密码/唯一性规则。

### 5.3 构建与部署

- `npm run build` 产出 `web/dist`；由反向代理在 `/` 托管，`/api/**` 转发后端。
- SPA 回退：非 `/api` 且非静态资源的路径回 `index.html`（客户端路由）。
- 环境无关：前端不读后端环境变量，唯一依赖是同源路径。

## 6. 安全设计

| 风险 | 措施 |
|---|---|
| XSS 窃取令牌 | 不持久化会话与 CSRF；React 默认转义；不使用 `dangerouslySetInnerHTML` |
| 跨站请求 | 依赖 `SameSite=Lax` + CSRF 头；不引入 CORS |
| 点击劫持 | 由反代下发 `X-Frame-Options: DENY` / CSP `frame-ancestors 'none'`（部署项） |
| 混合内容 | 全站 HTTPS，禁止 http 资源 |
| 越权展示 | 前端隐藏入口只是体验优化，权限判定仍由后端 403 兜底 |
| 敏感信息泄漏 | 错误提示只展示 `message`，不在界面输出堆栈或标识符 |

## 7. 构建与检查命令（建议）

| 命令 | 用途 | 是否纳入 `add.manifest.yaml` |
|---|---|---|
| `npm ci` | 依赖安装（锁文件固定） | 否（环境准备） |
| `npm run lint` | ESLint + TypeScript 类型检查 | 建议纳入（需授权改采用配置） |
| `npm run test` | Vitest 单元/组件测试 | 建议纳入 |
| `npm run build` | 生产构建 | 建议纳入 |

> 采用配置变更属项目级决定，本设计只提建议；未获授权前不修改 `add.manifest.yaml`。

## 8. 测试策略（层次，详细见 test-plan）

| 层次 | 覆盖 | 工具 |
|---|---|---|
| 单元 | 错误映射、CSRF 注入、会话状态机、分页游标拼接 | Vitest |
| 组件 | 登录/注册表单校验、列表空态、删除确认、401 跳转 | Vitest + Testing Library |
| 契约对齐 | 对照真实响应结构（可用录制样本），防止字段漂移 | Vitest（fixture） |
| E2E | 真实部署后走通注册→登录→发文→编辑→删除→登出 | 部署环境，另立任务 |

**本设计不声称任何测试已执行。**

## 9. 假设与待确认

| # | 项 | 当前处理 | 影响 |
|---|---|---|---|
| A1 | 登出端点最终形状 | 按 §4.2 实施；若后端评审改路径，前端同步 | 契约测试与前端调用 |
| A2 | 列表 `limit` 上限与默认 | 前端固定 20，后端为准 | 分页验收值 |
| A3 | 编辑冲突（ETag/If-Match） | 现有编辑是否要求 `If-Match` 待 G0 核对；若要求，前端需保存 ETag | 可能新增一次契约细节 |
| A4 | 隐私声明文本与版本 | 由部署方提供，前端按内容展示 | 注册页文案 |
| A5 | 前端检查命令是否入 manifest | 未授权，暂不入 | 项目检查覆盖 |
| A6 | 部署域名/证书/反代归属 | 未定，按同源假设设计 | 环境验收 |

## 10. 依据与版本

| 依据 | 版本 |
|---|---|
| 需求 | `requirements-ir02-web-spa-r03`（`sha256:3676fdf0…be258e`） |
| 边界 | `web-spa-boundary-ir02-r02`（r01 正文 + 接收记录） |
| 安全合同 | `internal/platform/http_security.go`（Cookie/CSRF/SameSite） |
| 接口投影 | `internal/article/model.go:284-360` |

**本设计不构成**：设计确认、实现授权、部署批准或对既有接口的修改许可。
