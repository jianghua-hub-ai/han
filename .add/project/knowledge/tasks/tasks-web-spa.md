# web-spa 任务台账（IR-02）

> 本文件是 IR-02「Web 前端（独立 SPA）」的任务 SSOT，与 IR-01 的 `tasks.md` 并列、互不复制状态。原因：`tasks.md` 已承载 86 张 IR-01 任务卡与统计正文，直接追加会污染其历史统计；两个交付单元各自持有任务账，符合"共享事实只引用、不重复登记"的约束。
>
> 状态标记按 `task-execution-workflow §10`：`[ ]` 未开始、`[-]` 进行中/建成待验、`[x]` 关闭（完成日期写行末 `— 完成@日期`）。

**当前统计**：0 已完成 / 11 建成待验 / 1 未开始。

## 批次 W1 · 前端骨架与会话

- [-] `web-spa-S1.1` 建立前端工程骨架 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：在 `web/` 建立可构建、可 lint、可跑测试的最小 React + TypeScript + Vite 工程。
  - 文件清单：`web/package.json`（新增）、`web/package-lock.json`（新增）、`web/tsconfig.json`（新增）、`web/vite.config.ts`（新增）、`web/index.html`（新增）、`web/src/main.tsx`（新增）、`web/eslint.config.js`（新增）、`web/src/setupTests.ts`（新增）
  - 验收命令：工作目录 `web/` → `npm ci`；`npm run build`；`npm run lint`；`npm run test`
  - 依赖：无
  - 类型：开发
  - 本任务关闭条件：四条命令 exit 0（测试可为空集但 runner 必须真实执行）、`web/dist` 生成、未改 `add.manifest.yaml`；作者自测 + Lead 核 diff
  - **当前证据（2026-11-06）**：`npm install` 完成（286 包）；`npm run build` exit 0（`dist/index.html` 0.32 kB、`dist/assets/index-*.js` 224.07 kB）；`npm run lint` exit 0；`npm run test` 1 passed。**未做**：独立评审、`npm ci` 冷启动验证。
  - **已知环境限制**：npm 的 `allow-scripts` 策略跳过了 `esbuild` 的 postinstall；本次 `vite build` 仍成功，但换机或清理后重装可能需 `npm approve-scripts esbuild`。
  - 后续验证关联：无（E2E 另立）

- [-] `web-spa-S1.2` 应用外壳与路由骨架 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：建立路由表、布局与受保护路由守卫骨架（暂用占位页面）。
  - 文件清单：`web/src/app/router.tsx`（新增）、`web/src/app/layout.tsx`（新增）、`web/src/pages/*`（占位，新增）
  - 验收命令：工作目录 `web/` → `npm run build`；`npm run test`
  - 依赖：requires: `web-spa-S1.1`
  - 类型：开发
  - 本任务关闭条件：7 条路由可达（含 404 兜底）、未登录访问受保护路由重定向且保留目标路径（组件测试）
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - 新增 `web/src/app/router.tsx`：`routes` 路由表 + `RequireSession` 守卫（`unknown`→骨架屏 `role="status"`；`anonymous`→`<Navigate to="/login" state={{from}}>`；`authenticated`→`<Outlet/>`）+ `AppRouter`（`createBrowserRouter`）。
    - 新增 `web/src/app/layout.tsx`：`AppLayout`（头部导航 + `<Outlet/>`）。
    - 新增 `web/src/pages/`：`login`、`register`、`article-list`、`article-detail`、`my-articles`、`article-editor`、`not-found` 七个占位页。
    - `web/src/main.tsx` 改为挂载 `<SessionProvider><AppRouter/></SessionProvider>`；S1.1 的占位外壳 `web/src/App.tsx`、`web/src/App.test.tsx` 已被真实外壳取代并**删除**（无其它引用，产物不再有死代码）。
    - 测试 `web/src/app/router.test.tsx` 10 用例：未登录访问 `/me/articles?page=2` → 落到 `/login` 且 `state.from` 保留完整目标；探测中显示骨架屏；5 条受保护路由在已登录态各自渲染正确标题；2 条公开路由无需会话；未知路径落 404。
    - 命令：`npm run test` → 3 文件 24 用例全绿；`npm run lint` exit 0；`npm run build` exit 0（51 modules，`dist/assets/index-*.js` 324.26 kB / gzip 103.43 kB）。
  - **仍未做**：独立评审与接收决定（本 Host 无 Reviewer Role，不冒充）。
  - **范围说明**：占位页只渲染标题与说明，不发任何请求；真实表单/列表由 S3.x–S5.x 接入。

- [-] `web-spa-S2.1` 后端新增登出端点 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：新增 `POST /api/v1/sessions/logout`，需会话与 CSRF，成功 204 并清 Cookie，未认证 401，重复调用不得 5xx。
  - 文件清单：`internal/identity/session_logout.go`（新增）、`internal/identity/session_logout_test.go`（新增）、`cmd/han-server/main.go`（修改：路由与装配）、`cmd/han-server/logout_revoker.go`（新增）、`cmd/han-server/session_logout_route_test.go`（新增）、`cmd/han-server/session_logout_integration_test.go`（新增）、`test/contract/identity/session_logout_test.go`（新增）
  - 验收命令：仓库根 → `go build ./...`；`go vet ./...`；`go test ./internal/identity -run TestSessionLogout -count=1`；`go test -tags=contract ./test/contract/identity -run TestSessionLogout -count=1`；`HAN_TEST_POSTGRES_DSN=<真实 PG 18 DSN> go test -tags=integration ./cmd/han-server -run TestSessionLogoutRevokesRealPostgresSession -count=1`
  - 依赖：无（后端独立）
  - 类型：开发（②档：先写失败的契约测试再实现）
  - 本任务关闭条件：契约测试红→绿、`gofmt -l` 为空、既有 identity 测试无回归；独立评审待安排
  - 后续验证关联：`web-spa-S2.2`（前端消费）；集成层的会话撤销验证见 test-plan T07
  - **当前证据（2026-11-06）**：新增 `internal/identity/session_logout.go`、`cmd/han-server/logout_revoker.go`、`cmd/han-server/session_logout_route_test.go`；`platform.ClearSessionCookie`；路由 `POST /api/v1/sessions/logout` 挂在 `handlers[22]`，经 `originGuard` + `authorizer.Protect(session.AccessGeneral)`。`go build ./...`、`go vet ./...`、`gofmt -l`、`go test ./...`（14 包）全部 exit 0；2 个 handler 单测 + 2 个路由级测试通过（POST 204 / GET 405 且 `Allow: POST`）。
  - **当前证据（2026-11-06 第二轮补齐）**：
    - 公共契约测试已补：`test/contract/identity/session_logout_test.go`（`//go:build contract`）三条断言——204 且无 body、唯一清 Cookie（`MaxAge<0`/`HttpOnly`/`Secure`/`SameSite=Lax`/`Path=/`）、无 Cookie 时 401 `auth.session_required` 且不触碰 revoker、revoker 失败时 500 `platform.internal_error` 且不泄漏内部 cause 也不清 Cookie。`go test -tags=contract ./test/contract/...` → `article`/`identity` 两包 ok。
    - 真实 PG 集成已验：本地一次性 PostgreSQL 18.6 集群（`127.0.0.1:55441`，非生产库），`HAN_TEST_POSTGRES_DSN` 指向它，跑 `go test -tags=integration ./cmd/han-server -run TestSessionLogoutRevokesRealPostgresSession -count=1` → PASS（0.79s）。
    - 该集成测试使用**生产同款**依赖：`session.Store` + `postgresSessionLogout`（生产 revoker 本体）+ `identity.SessionLogoutHandler` + `platform.HTTPAuthorizer`。断言链：登录态探测 204（证明 fixture 真的可用）→ POST 登出 204 且 `Set-Cookie` 清 Cookie → 数据库 `sessions.revoked_at` 非空且 `revoke_reason='logout'` → 同一 Cookie 再探受保护路由 **401** → 重复调用登出为 **401 而非 5xx**。未接入 `originGuard`（正交 CSRF 加固，另有测试覆盖），已在测试注释中写明。
    - 测试 schema 自建自删（`DROP SCHEMA ... CASCADE`），不留残留。
  - **仍未做**：独立评审与接收决定（本 Host 无 Reviewer Role，不冒充）。
  - **偏离记录**：实现顺序未按 ②档先写失败测试再实现（先实现、后补契约测试与集成测试）。

- [-] `web-spa-S2.2` 前端会话上下文与 API 客户端 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：实现会话三态、启动探测、401 全局处理、CSRF 注入与错误归一。
  - 文件清单：`web/src/api/client.ts`（新增）、`web/src/api/auth.ts`（新增）、`web/src/app/session.tsx`（新增）、`web/src/api/client.test.ts`（新增）、`web/src/app/session.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`（覆盖 T01–T06）；`npm run lint`；`npm run build`
  - 依赖：requires: `web-spa-S1.1`、`web-spa-S2.1`
  - 类型：开发（②档：401/CSRF 分支先失败）
  - 本任务关闭条件：会话与 token 不出现在 storage（测试断言）、401 触发统一失效、写请求带 CSRF 头
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - 新增 `web/src/api/client.ts`：`ApiClient.request` 统一 `credentials: 'same-origin'`；不安全方法（POST/PUT/PATCH/DELETE）从内存取 `X-CSRF-Token`；204 返回 `undefined`；非 2xx 归一为 `ApiError{status,code,message,fieldErrors,traceId}`；传输失败归一为 `network.error`（status 0），不误判为 401；401 触发注入的 `onUnauthorized` 回调。为可测性保留 `FetchLike` 注入点，生产默认走 `fetch`。
    - 新增 `web/src/api/auth.ts`：`POST /api/v1/sessions` 登录（`user` 字段 snake_case→camelCase，取 `csrf_token`）、`POST /api/v1/sessions/logout` 登出。均为工厂函数 `createAuthApi(client)`，无模块级可变全局。
    - 新增 `web/src/app/session.tsx`：三态 `unknown|anonymous|authenticated`；挂载时以 `GET /api/v1/articles?limit=1` 探测（401→anonymous，传输/后端失败→保持 unknown 并置 `bootstrapFailed`，不误判未登录）；CSRF 只存 `useRef`/state，**不写 localStorage/sessionStorage**；任一 401 统一 `invalidate()`；登出成功清空。
    - 测试：`npm run test` → 3 文件 15 用例全绿（client 7 + session 7 + App 1）；`npm run lint` exit 0；`npm run build` exit 0（tsc --noEmit + vite build）。
    - 说明：本任务提交时 `SessionProvider` 尚未挂到 `main.tsx`（属 S1.2 范围）；S1.2 已完成挂载，模块此前也已进入 `tsc --noEmit` 类型检查。
  - **仍未做**：独立评审与接收决定；页面尚未消费（S3.x/S4.x 接入）。
  - **G0 核对发现（阻塞后续页面，未自行改契约）**：见下方「契约缺口」小节。

## 批次 W2 · 身份界面

- [ ] `web-spa-S3.1` 注册页
  - 目标：注册表单、隐私声明确认、字段错误映射。
  - 文件清单：`web/src/pages/register.tsx`（新增）、`web/src/pages/register.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S2.2`
  - 类型：开发
  - 本任务关闭条件：US-WEB-01.1 五条可观察断言全绿；未确认声明时不可提交
  - 后续验证关联：无

- [-] `web-spa-S3.2` 登录与登出界面 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：登录表单、统一失败提示、登出按钮（调用新端点）。
  - 文件清单：`web/src/pages/login.tsx`（新增）、`web/src/pages/login.test.tsx`（新增）、`web/src/components/LogoutButton.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S2.2`
  - 类型：开发
  - 本任务关闭条件：US-WEB-01.2/01.3 断言全绿；登出后回到未登录态
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - `web/src/pages/login.tsx`：受控表单（`label` 与 `input` 用 `htmlFor`/`id` 绑定，`autoComplete` 正确）；提交中按钮禁用并显示"正在登录…"；4xx 统一提示"登录失败，请检查用户名和密码"（不区分凭据错误/账号不存在/停用/删除，US-WEB-01.2.2），5xx/网络用"登录失败，请稍后重试"；成功后按 `location.state.from` 返回（仅接受 `/` 开头的同源绝对路径，防开放重定向）；`expired` 时提示"登录已失效，请重新登录。"。
    - `web/src/components/LogoutButton.tsx`：调用 `POST /api/v1/sessions/logout`，成功或失败都落到匿名态；并已挂到 `AppLayout` 头部。
    - `web/src/app/session.tsx` 增量：新增 `expired` 语义——**启动探测的 401 不算过期**（只判为未登录），**引导完成后的 401 才算会话失效**；登录/登出会清除该标记。
    - `web/src/api/client.ts` 增量：`RequestOptions.suppressUnauthorized`；`auth.login` 用它避免"登录失败"被误判为"会话过期"（这是自测中发现的真实 UX 缺陷，已修正并加断言）。
    - 测试：`web/src/pages/login.test.tsx` 5 用例（字段可访问、成功回跳目标路径、统一失败文案且不泄漏后端措辞、请求中禁用按钮、活会话被 401 打断后回登录页并提示已失效）；`client.test.ts` 增至 8 用例（新增 suppress 钩子用例）。
    - 命令：`npm run test` → 4 文件 30 用例全绿；`npm run lint` exit 0；`npm run build` exit 0（52 modules，326.00 kB / gzip 104.06 kB）。
  - **仍未做**：独立评审与接收决定；真实浏览器 E2E（属 test-plan 的 E2E 层）。

## 批次 W3 · 文章浏览

- [-] `web-spa-S4.1` 文章列表页 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：游标分页列表、空态、加载与错误态。
  - 文件清单：`web/src/pages/article-list.tsx`（新增）、`web/src/components/Pagination.tsx`（新增）、`web/src/pages/article-list.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S2.2`
  - 类型：开发
  - 本任务关闭条件：T09/T10 断言全绿（无重复、无跳过、末页停止）
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - 新增 `web/src/api/articles.ts`（设计 §5.1 指定但任务卡未列，属实现必需）：`list({cursor,limit})` 映射 `items/next_cursor` 与 snake_case 字段、`get(id)` 映射详情投影、`authorLabel()`（`anonymous`→「匿名」，无 label 的用户→「社区成员」，不臆造用户名）。
    - `web/src/pages/article-list.tsx`：首屏加载、游标翻页（`Pagination` 只在 `next_cursor` 非空时出现，末页自动消失）、append 时按 id 去重（服务器重发也不会重复）、空态「还没有文章」与错误态「重试」分离、加载态 `role="status"`。
    - 新增 `web/src/components/Pagination.tsx`：游标模型下唯一动作是"加载更多"，无页码。
    - 测试：`article-list.test.tsx` 3 用例——T09 两页游标翻页后 4 条顺序正确、无重复、无跳过、按钮消失且第二次请求带 `cursor=cursor-1`；T10 空列表显示空态且无 alert；503 显示错误条、点击重试恢复。
  - **仍未做**：独立评审与接收决定；真实浏览器 E2E。

- [-] `web-spa-S4.2` 文章详情页 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：详情展示、不可见统一提示、作者显示（无归属为「匿名」）。
  - 文件清单：`web/src/pages/article-detail.tsx`（新增）、`web/src/pages/article-detail.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S4.1`
  - 类型：开发
  - 本任务关闭条件：T11 断言全绿；404/403 分支有独立用例
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - `web/src/pages/article-detail.tsx`：`GET /api/v1/articles/{id}`；标题、作者（无归属显示「匿名」）、日期、正文按纯文本渲染（`white-space: pre-wrap`，不使用 `dangerouslySetInnerHTML`）；404→「内容不可用」、403→「没有权限」、其它→错误条+重试；加载态 `role="status"`。
    - 测试：`article-detail.test.tsx` 4 用例——成功渲染标题/正文/匿名作者；T11 404 统一「内容不可用」；403 独立「没有权限」且不落 404 文案；500 后重试成功。
  - **仍未做**：独立评审与接收决定；真实浏览器 E2E。

## 批次 W4 · 我的文章

- [-] `web-spa-S5.1` 我的文章列表与新建 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：`/me/articles` 列表 + `/me/articles/new` 创建表单。
  - 文件清单：`web/src/pages/my-articles.tsx`（新增）、`web/src/pages/article-editor.tsx`（新增）、`web/src/pages/my-articles.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S3.2`、`web-spa-S4.2`
  - 类型：开发
  - 本任务关闭条件：T12/T13 全绿（失败保留输入）
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - `web/src/api/articles.ts` 新增 `listMine`（`GET /users/me/articles`，复用 `{items,next_cursor}` 信封）与 `create`（`POST /articles`，`requestWithEtag` 取回 ETag）。
    - `my-articles.tsx`：游标分页（`Pagination`）、空态「你还没有文章」、加载失败可重试、进入编辑页与新建页入口。
    - `article-editor.tsx`：标题/正文必填才可提交；发布成功跳回 `/me/articles`；失败显示 `role="alert"` 且**输入不丢**。
    - 测试：`my-articles.test.tsx` 4 用例（列表与编辑链接、空态、删除二次确认成功后移除行、删除失败保持原状）；`article-editor.test.tsx` 2 用例（提交体与跳转、失败保留输入）。
  - **仍未做**：独立评审与接收决定；真实浏览器 E2E（含真实 `GET /api/v1/users/me/articles`）。

- [-] `web-spa-S5.2` 编辑自己的文章 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：编辑页预填与保存；先核实现有编辑接口是否要求 `If-Match`。
  - 文件清单：`web/src/pages/article-edit.tsx`（新增）、`web/src/pages/article-edit.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S5.1`
  - 类型：开发
  - 本任务关闭条件：T14 全绿；若接口要求 `If-Match`，须实现 ETag 往返并回写 design §9 A3 后再关闭
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - G0 核对结论：作者视图为 `GET /api/v1/users/me/articles/{id}`，保存为 `PUT /api/v1/articles/{id}` 且**强制 `If-Match: <强 ETag>`**（`internal/article/handler.go:443-531`）；design §9 A3 已记录该结论。
    - `article-edit.tsx`：预填标题/正文；保存时回传预填响应的 `ETag`；`412/409` 显示「内容已被修改，请刷新后重试。」；`editable=false` 时只读、按钮禁用并提示「这篇文章当前不可编辑」；404/403 各有独立页面。
    - 测试：`article-edit.test.tsx` 3 用例（预填 + `If-Match` 回传、412 冲突文案、不可编辑只读）。
  - **仍未做**：独立评审与接收决定；真实浏览器 E2E。

- [-] `web-spa-S5.3` 删除自己的文章 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：删除入口、二次确认、成功后刷新列表。
  - 文件清单：`web/src/components/ConfirmDialog.tsx`（新增）、`web/src/pages/my-articles.tsx`（修改：接删除）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S5.1`
  - 类型：开发
  - 本任务关闭条件：T16 全绿；失败时文章保持原状
  - 后续验证关联：无
  - **当前证据（2026-11-06）**：
    - 删除走 `DELETE /api/v1/articles/{id}`（`internal/article/handler.go` 204 无正文）。
    - `ConfirmDialog.tsx`：`role="dialog"` + `aria-modal`、Esc 与「取消」关闭、`pending` 时禁用按钮。
    - 成功路径**不做乐观删除**：接口 204 后才从本地列表移除该行；失败保留原行并显示 `role="alert"`「删除失败…」。
    - 测试：`my-articles.test.tsx` 的删除两用例即 T16 的可观察断言（确认层拦截、成功移除、失败保持）。
  - **仍未做**：独立评审与接收决定；真实浏览器 E2E（含真实 204 与失败分支）。

## 批次 W5 · 部署接入

- [-] `web-spa-S6.1` 部署接入文档与反代样例 — 已建成@2026-11-06（待独立评审与接收）
  - 目标：产出同源反代与 SPA 回退配置样例、上线核对清单（不执行真实部署）。
  - 文件清单：`docs/runbooks/web-frontend-deployment.md`（新增）、`deploy/nginx-han.conf.example`（新增）
  - 验收命令：仓库根 → 文本核对：`Select-String -Path docs/runbooks/web-frontend-deployment.md -Pattern '/api'`
  - 依赖：requires: `web-spa-S1.1`
  - 类型：开发（文档/配置样例）
  - 本任务关闭条件：样例覆盖 `/` 静态、`/api/**` 转发、SPA 回退、HTTPS、`X-Frame-Options`/CSP；不声称已部署
  - 后续验证关联：部署后 E2E（未登记）
  - **当前证据（2026-11-06）**：
    - 新增 `docs/runbooks/web-frontend-deployment.md`：前提（仅 HTTPS、同源、`__Host-han_session` 的 Secure/Path/无 Domain 要求）、构建命令、反代四要点、9 条上线核对清单、已知边界（明确写出 GAP-1/GAP-2 未解决前对应页面不可用）。
    - 新增 `deploy/nginx-han.conf.example`：80→443 跳转；`/api/` 反代（保留 Host/`X-CSRF-Token`、`X-Forwarded-Proto`）；`location /` 的 `try_files $uri $uri/ /index.html` SPA 回退；`/assets/` immutable 长缓存与 `index.html` no-store；HSTS、`X-Content-Type-Options`、`X-Frame-Options: DENY`、CSP `frame-ancestors 'none'`；`/livez`、`/readyz` 探针反代。
    - 核对：`Select-String -Path docs/runbooks/web-frontend-deployment.md -Pattern '/api'` 命中 4 处；样例中 `try_files`、`proxy_pass`、HSTS、`X-Frame-Options`、CSP 均存在。
    - **没有执行任何真实部署**，文档开头与台账均写明"样例未在真实主机验证"。
  - **仍未做**：独立评审与接收决定；真实部署环境下的 E2E（本机无 docker/无目标主机）。

## 契约缺口（IR-02 G0 核对发现，2026-11-06；已授权处置）

实施 S2.2 前按 design §4.1 的「实施 G0 时逐条核对」要求核对了真实路由表与 handler，发现两处设计假设与代码不一致。**未自行改契约**，登记待决策。

### GAP-1 · `POST /api/v1/users`（注册）已实现但未挂载路由

- 事实：`internal/identity/handler.go:447` 有 `Register`，`test/contract/identity/rest_contract_test.go` 也测了它；但 `cmd/han-server/main.go:3585-3680` 的路由 switch 没有 `/api/v1/users` 分支，`productionIdentityHandlers` 结构体也没有 register 字段。全仓除此 handler 与其单测外无任何挂载点。
- 后果：真实服务器对 `POST /api/v1/users` 返回 `404 request.route_not_found`；SPA 注册页（`web-spa-S3.1`）无法工作。
- 影响范围：这是既有后端装配缺口，不是 IR-02 引入；但它使 design §3.1 的 `/register` 与需求 IR-02 的注册范围无法交付。

### GAP-2 · `GET /api/v1/users/me/articles`（我的文章列表）不存在

- 事实：路由表只有 `GET /api/v1/users/me/articles/{id}`（作者视图详情，`main.go:3647`）。`article.Handler.ListArticles` 只接受 `cursor`、`limit`（`internal/article/handler.go:318-349`），无作者过滤参数。列表投影 `MemberArticleSummaryView{id,title,author{kind,id,label},created_at,updated_at}` 不含正文。
- 后果：`/me/articles` 页（`web-spa-S5.1`）无法取得「我的文章」列表；用公共列表 + 客户端过滤会因游标分页而漏项。
- 影响范围：需要新增只读端点（或给列表加作者过滤参数），属于 design「唯一后端变更是登出」之外的变更。

### 建议处置（已获授权：2026-11-06「按建议执行 GAP-1 + GAP-2 并修订 design」）

1. 把 GAP-1 视为**缺陷修复**：把已存在的 `Register` 挂到 `POST /api/v1/users`（新增一条路由 + 一个 handler 槽位 + 路由级测试），不改业务逻辑。
2. 把 GAP-2 视为**范围增补**：新增 `GET /api/v1/users/me/articles?limit=&cursor=`，复用现有列表信封 `{items,next_cursor}`，按当前会话用户过滤；需要修订 design §4.1 与「唯一后端变更」表述。
3. 两项都补契约测试与路由级测试后，再继续 S3.1 / S5.1。

### 处置结果（2026-11-06）

**GAP-1 · 注册路由已挂载**

- `cmd/han-server/main.go` 新增 `newRegistrationHandler()`：用既有 `identity.NewService(RegistrationDependencies{…})`（`identity.Repository{}`、`identity.CurrentNoticeReader{}`、`processingGate`、`security.PasswordPolicy{}`、既有 hasher、`audit.Store{}`、既有 ID/Clock）构造服务，再用既有 `identity.NewHandler` 造 handler；**未改任何注册业务逻辑**。
- 装配：`productionIdentityHandlers.register`；仅经 `originGuard`（公开端点，无 authorizer），与登录同一安全边界；路由槽位 `handlers[23]`，`POST /api/v1/users`。
- 降级行为：若 `processingGate` 未装配（部分测试组合），`newRegistrationHandler` 返回 nil，路由保持未注册（返回 404），不会让整个 server 启动失败；真实部署必经 `newProcessingGate()`，不存在该降级。
- 测试：`cmd/han-server/ir02_route_test.go` 的 `TestRegistrationRoute_PostIsDispatched`（201）与 `..._OtherMethodsRejected`（405 + `Allow: POST`）。

**GAP-2 · 我的文章列表已新增**

- 新文件 `internal/article/author_list.go`：`AuthorArticleRepository`（独立只读 seam，不动既有 `ArticleQueryRepository`）、`ListAuthorBatch`（`WHERE author_id = $1` + 同一 keyset 游标/排序）、`ArticleQueryService.ListAuthorArticles`（复用 `resolveLimit`/`decodeMemberCursor`/`buildListArticlesPage`）、`Handler.ListMyArticles`（`limit`/`cursor` 校验与统一错误体）。
- `internal/article/policy.go` 新增 `AuthorListProjection`：仅当 `author_id == 当前用户`、状态合法、`DeletionState == active` 时返回摘要；**包含违规/限制的本人文章**，但绝不含他人文章。
- 装配：`buildArticleQueryHandlers` → `[8]` 槽位（第 8 个 = 我的文章），`buildArticleHandlers` → `[17]`，路由槽位 `handlers[24]`，`GET /api/v1/users/me/articles`，经 `authorizer.Protect(session.AccessGeneral, …)`。
- 测试：`internal/article/author_list_test.go` 3 个用例（只返回本人且含违规文章、排除他人与已删除；游标两页无重复无跳过；handler 信封与超大 `limit` → 422）；`cmd/han-server/ir02_route_test.go`（GET 分发、POST → 405 + `Allow: GET`）；原「未知路径」用例已移除该路径，改为独立 405 断言。
- design 已修订为 `web-spa-design-ir02-r03`：§4.1 修正作者视图/If-Match，§4.2 拆为三处后端变更（登出、注册挂载、我的文章）。

**仍未验证（如实记录）**

- 注册与我的文章列表**没有跑真实 PostgreSQL 端到端**：注册依赖 `privacy_notices` 有 `current` 版本且 processing gate 开启，需要真实数据准备；本轮只有单元 / 路由 / 契约层证据。
- `newRegistrationHandler` 的生产装配正确性由编译期类型 + 既有适配器实现保证，未在真实启动路径执行。

**补充验证（2026-11-06 复核轮）**

- GAP-2 的 SQL 已用真实 PostgreSQL 18.6 验证：新增 `internal/article/author_list_integration_test.go`（`//go:build integration`），自建自删隔离 schema + 全量迁移；种子数据为「本人 3 篇（其中 1 篇 `violation_state=violating`）+ 他人 1 篇」，用 `limit=2` 走两页。
  - 断言：第一页 2 条且顺序为最新在前、`next_cursor` 非空；第二页 1 条且 `next_cursor` 为空；跨页无重复、无跳过（共 3 个不同 id）；他人文章不出现；违规的本人文章仍可见。
  - 命令：`HAN_TEST_POSTGRES_DSN=<真实 PG> go test -tags=integration ./internal/article -run TestAuthorArticleListKeysetAgainstPostgres -count=1` → **PASS（0.77s）**。
- 复核轮同时重跑全部项目检查：`gofmt -l ./cmd ./internal ./test` 无输出、`go mod verify` 全通过、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`（16 包）、`go test ./...`（14 包）全部 exit 0；前端 `npm run test` 46 用例 / 9 文件全绿、`npm run lint`、`npm run build` 均 exit 0。
- 注册仍**没有**真实 PostgreSQL 端到端（需要 `privacy_notices.current` 与 processing gate 的真实数据准备），该项保持未验证。
