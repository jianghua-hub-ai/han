# web-spa 任务台账（IR-02）

> 本文件是 IR-02「Web 前端（独立 SPA）」的任务 SSOT，与 IR-01 的 `tasks.md` 并列、互不复制状态。原因：`tasks.md` 已承载 86 张 IR-01 任务卡与统计正文，直接追加会污染其历史统计；两个交付单元各自持有任务账，符合"共享事实只引用、不重复登记"的约束。
>
> 状态标记按 `task-execution-workflow §10`：`[ ]` 未开始、`[-]` 进行中/建成待验、`[x]` 关闭（完成日期写行末 `— 完成@日期`）。

**当前统计**：0 已完成 / 3 建成待验 / 9 未开始。

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

- [ ] `web-spa-S1.2` 应用外壳与路由骨架
  - 目标：建立路由表、布局与受保护路由守卫骨架（暂用占位页面）。
  - 文件清单：`web/src/app/router.tsx`（新增）、`web/src/app/layout.tsx`（新增）、`web/src/pages/*`（占位，新增）
  - 验收命令：工作目录 `web/` → `npm run build`；`npm run test`
  - 依赖：requires: `web-spa-S1.1`
  - 类型：开发
  - 本任务关闭条件：7 条路由可达（含 404 兜底）、未登录访问受保护路由重定向且保留目标路径（组件测试）
  - 后续验证关联：无

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
    - 说明：`SessionProvider` 尚未挂到 `main.tsx`（属 S1.2 路由骨架的范围），因此本轮 bundle 内容未变；模块已进入 `tsc --noEmit` 类型检查。
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

- [ ] `web-spa-S3.2` 登录与登出界面
  - 目标：登录表单、统一失败提示、登出按钮（调用新端点）。
  - 文件清单：`web/src/pages/login.tsx`（新增）、`web/src/pages/login.test.tsx`（新增）、`web/src/components/LogoutButton.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S2.2`
  - 类型：开发
  - 本任务关闭条件：US-WEB-01.2/01.3 断言全绿；登出后回到未登录态
  - 后续验证关联：无

## 批次 W3 · 文章浏览

- [ ] `web-spa-S4.1` 文章列表页
  - 目标：游标分页列表、空态、加载与错误态。
  - 文件清单：`web/src/pages/article-list.tsx`（新增）、`web/src/components/Pagination.tsx`（新增）、`web/src/pages/article-list.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S2.2`
  - 类型：开发
  - 本任务关闭条件：T09/T10 断言全绿（无重复、无跳过、末页停止）
  - 后续验证关联：无

- [ ] `web-spa-S4.2` 文章详情页
  - 目标：详情展示、不可见统一提示、作者显示（无归属为「匿名」）。
  - 文件清单：`web/src/pages/article-detail.tsx`（新增）、`web/src/pages/article-detail.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S4.1`
  - 类型：开发
  - 本任务关闭条件：T11 断言全绿；404/403 分支有独立用例
  - 后续验证关联：无

## 批次 W4 · 我的文章

- [ ] `web-spa-S5.1` 我的文章列表与新建
  - 目标：`/me/articles` 列表 + `/me/articles/new` 创建表单。
  - 文件清单：`web/src/pages/my-articles.tsx`（新增）、`web/src/pages/article-editor.tsx`（新增）、`web/src/pages/my-articles.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S3.2`、`web-spa-S4.2`
  - 类型：开发
  - 本任务关闭条件：T12/T13 全绿（失败保留输入）
  - 后续验证关联：无

- [ ] `web-spa-S5.2` 编辑自己的文章
  - 目标：编辑页预填与保存；先核实现有编辑接口是否要求 `If-Match`。
  - 文件清单：`web/src/pages/article-edit.tsx`（新增）、`web/src/pages/article-edit.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S5.1`
  - 类型：开发
  - 本任务关闭条件：T14 全绿；若接口要求 `If-Match`，须实现 ETag 往返并回写 design §9 A3 后再关闭
  - 后续验证关联：无

- [ ] `web-spa-S5.3` 删除自己的文章
  - 目标：删除入口、二次确认、成功后刷新列表。
  - 文件清单：`web/src/components/ConfirmDialog.tsx`（新增）、`web/src/pages/my-articles.tsx`（修改：接删除）
  - 验收命令：工作目录 `web/` → `npm run test`；`npm run lint`
  - 依赖：requires: `web-spa-S5.1`
  - 类型：开发
  - 本任务关闭条件：T16 全绿；失败时文章保持原状
  - 后续验证关联：无

## 批次 W5 · 部署接入

- [ ] `web-spa-S6.1` 部署接入文档与反代样例
  - 目标：产出同源反代与 SPA 回退配置样例、上线核对清单（不执行真实部署）。
  - 文件清单：`docs/runbooks/web-frontend-deployment.md`（新增）、`deploy/nginx-han.conf.example`（新增）
  - 验收命令：仓库根 → 文本核对：`Select-String -Path docs/runbooks/web-frontend-deployment.md -Pattern '/api'`
  - 依赖：requires: `web-spa-S1.1`
  - 类型：开发（文档/配置样例）
  - 本任务关闭条件：样例覆盖 `/` 静态、`/api/**` 转发、SPA 回退、HTTPS、`X-Frame-Options`/CSP；不声称已部署
  - 后续验证关联：部署后 E2E（未登记）

## 契约缺口（IR-02 G0 核对发现，2026-11-06）

实施 S2.2 前按 design §4.1 的「实施 G0 时逐条核对」要求核对了真实路由表与 handler，发现两处设计假设与代码不一致。**未自行改契约**，登记待决策。

### GAP-1 · `POST /api/v1/users`（注册）已实现但未挂载路由

- 事实：`internal/identity/handler.go:447` 有 `Register`，`test/contract/identity/rest_contract_test.go` 也测了它；但 `cmd/han-server/main.go:3585-3680` 的路由 switch 没有 `/api/v1/users` 分支，`productionIdentityHandlers` 结构体也没有 register 字段。全仓除此 handler 与其单测外无任何挂载点。
- 后果：真实服务器对 `POST /api/v1/users` 返回 `404 request.route_not_found`；SPA 注册页（`web-spa-S3.1`）无法工作。
- 影响范围：这是既有后端装配缺口，不是 IR-02 引入；但它使 design §3.1 的 `/register` 与需求 IR-02 的注册范围无法交付。

### GAP-2 · `GET /api/v1/users/me/articles`（我的文章列表）不存在

- 事实：路由表只有 `GET /api/v1/users/me/articles/{id}`（作者视图详情，`main.go:3647`）。`article.Handler.ListArticles` 只接受 `cursor`、`limit`（`internal/article/handler.go:318-349`），无作者过滤参数。列表投影 `MemberArticleSummaryView{id,title,author{kind,id,label},created_at,updated_at}` 不含正文。
- 后果：`/me/articles` 页（`web-spa-S5.1`）无法取得「我的文章」列表；用公共列表 + 客户端过滤会因游标分页而漏项。
- 影响范围：需要新增只读端点（或给列表加作者过滤参数），属于 design「唯一后端变更是登出」之外的变更。

### 建议处置（待确认）

1. 把 GAP-1 视为**缺陷修复**：把已存在的 `Register` 挂到 `POST /api/v1/users`（新增一条路由 + 一个 handler 槽位 + 路由级测试），不改业务逻辑。
2. 把 GAP-2 视为**范围增补**：新增 `GET /api/v1/users/me/articles?limit=&cursor=`，复用现有列表信封 `{items,next_cursor}`，按当前会话用户过滤；需要修订 design §4.1 与「唯一后端变更」表述。
3. 两项都补契约测试与路由级测试后，再继续 S3.1 / S5.1。

未获确认前不实施上述后端改动。
