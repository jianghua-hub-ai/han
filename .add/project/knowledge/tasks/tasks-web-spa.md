# web-spa 任务台账（IR-02）

> 本文件是 IR-02「Web 前端（独立 SPA）」的任务 SSOT，与 IR-01 的 `tasks.md` 并列、互不复制状态。原因：`tasks.md` 已承载 86 张 IR-01 任务卡与统计正文，直接追加会污染其历史统计；两个交付单元各自持有任务账，符合"共享事实只引用、不重复登记"的约束。
>
> 状态标记按 `task-execution-workflow §10`：`[ ]` 未开始、`[-]` 进行中/建成待验、`[x]` 关闭（完成日期写行末 `— 完成@日期`）。

**当前统计**：0 已完成 / 0 进行中 / 12 未开始。

## 批次 W1 · 前端骨架与会话

- [ ] `web-spa-S1.1` 建立前端工程骨架
  - 目标：在 `web/` 建立可构建、可 lint、可跑测试的最小 React + TypeScript + Vite 工程。
  - 文件清单：`web/package.json`（新增）、`web/package-lock.json`（新增）、`web/tsconfig.json`（新增）、`web/vite.config.ts`（新增）、`web/index.html`（新增）、`web/src/main.tsx`（新增）、`web/eslint.config.js`（新增）、`web/src/setupTests.ts`（新增）
  - 验收命令：工作目录 `web/` → `npm ci`；`npm run build`；`npm run lint`；`npm run test`
  - 依赖：无
  - 类型：开发
  - 本任务关闭条件：四条命令 exit 0（测试可为空集但 runner 必须真实执行）、`web/dist` 生成、未改 `add.manifest.yaml`；作者自测 + Lead 核 diff
  - 后续验证关联：无（E2E 另立）

- [ ] `web-spa-S1.2` 应用外壳与路由骨架
  - 目标：建立路由表、布局与受保护路由守卫骨架（暂用占位页面）。
  - 文件清单：`web/src/app/router.tsx`（新增）、`web/src/app/layout.tsx`（新增）、`web/src/pages/*`（占位，新增）
  - 验收命令：工作目录 `web/` → `npm run build`；`npm run test`
  - 依赖：requires: `web-spa-S1.1`
  - 类型：开发
  - 本任务关闭条件：7 条路由可达（含 404 兜底）、未登录访问受保护路由重定向且保留目标路径（组件测试）
  - 后续验证关联：无

- [ ] `web-spa-S2.1` 后端新增登出端点
  - 目标：新增 `POST /api/v1/sessions/logout`，需会话与 CSRF，成功 204 并清 Cookie，未认证 401，重复调用不得 5xx。
  - 文件清单：`internal/identity/session_logout.go`（新增）、`internal/identity/session_logout_test.go`（新增）、`cmd/han-server/main.go`（修改：路由与装配）、`test/contract/identity/session_logout_test.go`（新增）
  - 验收命令：仓库根 → `go build ./...`；`go vet ./...`；`go test ./internal/identity -run TestSessionLogout -count=1`；`go test -tags=contract ./test/contract/identity -run TestSessionLogout -count=1`
  - 依赖：无（后端独立）
  - 类型：开发（②档：先写失败的契约测试再实现）
  - 本任务关闭条件：契约测试红→绿、`gofmt -l` 为空、既有 identity 测试无回归；独立评审待安排
  - 后续验证关联：`web-spa-S2.2`（前端消费）；集成层的会话撤销验证见 test-plan T07

- [ ] `web-spa-S2.2` 前端会话上下文与 API 客户端
  - 目标：实现会话三态、启动探测、401 全局处理、CSRF 注入与错误归一。
  - 文件清单：`web/src/api/client.ts`（新增）、`web/src/api/auth.ts`（新增）、`web/src/app/session.tsx`（新增）、`web/src/api/client.test.ts`（新增）、`web/src/app/session.test.tsx`（新增）
  - 验收命令：工作目录 `web/` → `npm run test`（覆盖 T01–T06）；`npm run lint`；`npm run build`
  - 依赖：requires: `web-spa-S1.1`、`web-spa-S2.1`
  - 类型：开发（②档：401/CSRF 分支先失败）
  - 本任务关闭条件：会话与 token 不出现在 storage（测试断言）、401 触发统一失效、写请求带 CSRF 头
  - 后续验证关联：无

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
