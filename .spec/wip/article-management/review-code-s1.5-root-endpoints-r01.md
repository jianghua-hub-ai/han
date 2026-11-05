# S1.5 真实 root 16 端点证据包 r01 — 独立非作者评审

## 评审身份与对象

- 角色：`add-reviewer`（本次唯一非作者独立 Reviewer）。作者/Lead 为另一主体，本评审未参与该证据的产生。
- 评审对象：`.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/` 全部 9 个文件。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；实际 HEAD（`git rev-parse HEAD`）= `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，与预期一致。
- 评审方式：只读。唯一写入为本报告；未修改任何生产代码、测试、证据、`tasks.md`、Plan 或其他制品，未执行任何 git 写操作。
- **结论：PASS WITH MINOR**（HIGH 0 / MEDIUM 0 / LOW 8）。

## 1. 自算哈希与字节

### 1.1 被审源码（自算，非采信报告）

| 文件 | SHA256 | 字节 |
| --- | --- | --- |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 |
| `cmd/han-server/main_test.go` | `0E61F4A37D314D2D35B80EA8184BEF17EB6227FCAF1CA0F57C09736005185820` | 399522 |

- 与 `index.json` 的 `source` 字段、`source-hashes-before.txt`（BEFORE 行）、`source-hashes-after.txt`（AFTER 行）逐字节、逐哈希一致。
- 我在独立重跑（含 `-race`）之后再次计算两文件，哈希与字节仍一致，`SOURCE_STABLE` 在本轮观察范围内成立。
- 注意：`git ls-files cmd` 为空，`git status` 显示 `?? cmd/han-server/`。即这两文件属于未跟踪的工作树内容，详见 S15-RE-F004。

### 1.2 证据包文件（自算）

| 文件 | SHA256 | 字节 | 与 index.json 比对 |
| --- | --- | --- | --- |
| `index.json` | `B7EA614ABD9022351DDB460EADA49E54C1CFCB1F7124B30BEF53FC280F83EB79` | 5723 | （清单本体，不在 artifacts 内） |
| `16-endpoint-map.md` | `7F51DB78ACD070D680D4E24941CA662A2612D1E4111F5992798ACBB008060610` | 6625 | 一致 |
| `lead-verification.md` | `B810D977986520C1DB9342838C0A925079A7796AA3F7D3F9998DD468E60A685D` | 1551 | 一致 |
| `root-normal-json.log` | `1D00130C86142DE585AE683DC73C9972D263118DD7B7B759A6DDB8EDE73E73ED` | 325529 | 一致 |
| `root-race-json.log` | `E475A8E0749654117785037DBA9D626E2CE0BC119E9451FAF43730778E9D25D7` | 325501 | 一致 |
| `root-normal-json-r02.log` | `B30D7329CE750B5041839A63FE474896ABEC3B9DF4137897A3C863108530D723` | 333414 | 一致 |
| `root-race-json-r02.log` | `AED0882E5D66A26F0727108EEC75E3B03DF2F45A474F8A02B90432D8CC2D27C9` | 333365 | 一致 |
| `source-hashes-before.txt` | `CAC695B6E0D410FB71605B3EA401E925622BCF46A20EEE7FB58B006EDAC9C5F5` | 211 | 一致 |
| `source-hashes-after.txt` | `10EFE4B42A0D964D544676DDBABF4DD51CD4115D4F03AC19491195330F67BC9D` | 209 | 一致 |

结论：清单与实际文件完全相符，未发现篡改或版本混用迹象。

## 2. 实际检查范围

**第一轮（包内一致性）**：`index.json` ↔ 前后哈希 ↔ 4 份 JSON 日志事件计数 ↔ `16-endpoint-map.md` 映射表 ↔ `lead-verification.md` 陈述，逐项对齐。

**第二轮（独立核证，不采信报告）**：
- 实读 `cmd/han-server/main.go` 3542–3708（`newLoginBusinessRouterWithTraceAndRecovery` 全文）及 3758–3775（路径判定辅助函数）、2424–2463（槽位装配）、3467–3528（`protectOrigin(authorizer.Protect(...))` 组合）。
- 实读 `internal/platform/http_security.go` 309–341（`HTTPAuthorizer.Protect` 内部顺序）。
- 实读 `cmd/han-server/main_test.go` 断言正文：2068–2236、3338–3451、3519–3647、5310–5438、5525–5735、8379–8420、10066–10187，以及测试替身 1818–1839、8796–8815。
- 独立重跑 `go vet` 与定向测试（普通 / `-race`），见 §5。

**上游依据**：test-plan §4.7.1（`test-plan.md` 527–542 行）16 端点清单、design 接口表（`design.md` 1061–1078 行）。二者与 `16-endpoint-map.md` 逐字一致，无增漏、无把同一路径不同方法混记为一行（`/api/v1/articles`、`/api/v1/articles/{id}`、`/api/v1/admin/articles/{id}`、`/api/v1/admin/orphan-articles/{id}` 均按方法拆行）。

**消费者/组合面**：`newProductionBusinessHandler`（2424–2463）把业务 handler 装入同一 22 槽位数组并调用同一 router 构造函数；`protectArticleMutationRoutes`（3505–3528）对 9 个写入消费者统一做 `protectOrigin(authorizer.Protect(...))`。故“测试命中的是生产同一 router 与同一安全链组合”成立。

## 3. 待证主张逐条结论

### 3.1 十六端点 method/path 精确注册 — 成立

| # | 方法 | 路径 | main.go 分支 | 槽位 | Allow | 可执行断言来源（均在本次运行集内） |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | POST | `/api/v1/articles` | 3618 | 17 | `GET, POST` | `AWIRERTR01RTR02RTR04`（create 204）、`ExactRouting_RejectsBeforeSecurity/collection_*`、`SuccessResponses/POST` |
| 2 | GET | `/api/v1/articles` | 3618 | 9 | `GET, POST` | `AWIRERTR01RTR02RTR04`（member list）、`AdminArticleQuery/OldConsumerCompatibility/read_projections/member_list` |
| 3 | GET | `/api/v1/articles/{id}` | 3632 | 10 | `GET, PUT, DELETE` | `AWIRERTR01RTR02RTR04`（member detail）、`QueryRoutes.../detail`、`read_projections/member_detail` |
| 4 | GET | `/api/v1/users/me/articles/{id}` | 3628 | 18 | `GET` | `ExactRouting_RejectsBeforeSecurity/author_*`、`AWIRERTR01RTR02RTR04`（author detail）、`read_projections/author_detail` |
| 5 | PUT | `/api/v1/articles/{id}` | 3632 | 19 | `GET, PUT, DELETE` | `AWIRERTR01RTR02RTR04`（update）、`SuccessResponses/PUT`、`SecurityChain/StopsEveryRejectedLayer/PUT/*` |
| 6 | DELETE | `/api/v1/articles/{id}` | 3632 | 4 | `GET, PUT, DELETE` | `AWIRERTR01RTR02RTR04`（ordinary）、`AWIRERTR05ThreeDeleteRoutesCarryRoleAndStatus` |
| 7 | GET | `/api/v1/admin/articles` | 3626 | 20 | `GET` | `AdminArticleQuery/ExactMethodsAndCoexistence/list`、`.../method_path_matrix/list`（7 方法 + Allow） |
| 8 | GET | `/api/v1/admin/articles/{id}` | 3644 | 21 | `GET, DELETE` | `.../ExactMethodsAndCoexistence/detail`、`.../method_path_matrix/detail` |
| 9 | POST | `/api/v1/admin/articles/{id}/violation` | 3604 | 11 | `POST` | `Moderation_Router_RegistersExactPOSTPaths/violation`、`Moderation_Production_RealHandlerSecurityAndRouting/violation/*`（12 变体） |
| 10 | POST | `/api/v1/admin/articles/{id}/restore` | 3608 | 12 | `POST` | 同上 `restore` |
| 11 | DELETE | `/api/v1/admin/articles/{id}` | 3644 | 5 | `GET, DELETE` | `AWIRERTR01RTR02RTR04`（violating）、`ThreeDeleteRoutes...`（violating）、`method_path_matrix/detail` |
| 12 | GET | `/api/v1/admin/orphan-articles` | 3612 | 13 | `GET` | `OrphanDispositionRouter_ListRoute_IsRegistered`、`RealHandlersEnforceSecurity/list/*` |
| 13 | GET | `/api/v1/admin/orphan-articles/{id}` | 3652 | 14 | `GET, DELETE` | `RealHandlersEnforceSecurity/detail/*`、`ExactPathsAndSharedAdapter`、`read_projections/orphan_detail` |
| 14 | POST | `/api/v1/admin/orphan-articles/{id}/publish` | 3614 | 15 | `POST` | `RealHandlersEnforceSecurity/publish/*`、`ExactPathsAndSharedAdapter` |
| 15 | POST | `/api/v1/admin/orphan-articles/{id}/transfer` | 3616 | 16 | `POST` | `RealHandlersEnforceSecurity/transfer/*`、`ExactPathsAndSharedAdapter` |
| 16 | DELETE | `/api/v1/admin/orphan-articles/{id}` | 3652 | 6 | `GET, DELETE` | `RealHandlersEnforceSecurity/delete/*`、`AWIRERTR01RTR02RTR04`（orphan）、`ThreeDeleteRoutes...`（orphan） |

逐行核对结果：分支行号、槽位下标、`Allow` 字符串与代码完全一致；不存在“路径字面量出现但无可执行断言”的端点（差异仅在 `Allow` 头断言覆盖，见 S15-RE-F008）。

### 3.2 method 拒绝先于安全链 — 成立

- 代码：method 判定（3668–3676）位于 probe（3677）、recovery guard（3694）、handler（3698）之前；未命中路由走 default（3660–3666）直接 404。`platform.TraceMiddleware(router)` 在 3707 作为最外层返回。
- 测试：`SecurityChain_StopsEveryRejectedLayer`（2068–2172）逐层断言事件序列 `trace → origin → session → csrf → principal → authorization → handler`，并在每层拒绝时断言业务下游零调用；`ExactRouting_RejectsBeforeSecurity`（2185–2212）对 3 条多方法路径的每种非法方法断言 405 + 精确 `Allow` + `nosniff` + 仅 `trace` 事件；`adminRootMethodCase`（1662–1686）同样断言 405 时事件仅 `trace`；moderation 的 `method` 变体断言 `Allow: POST` 且 resolver 未被调用（3622–3628）。

### 3.3 Allow 与 404/405 语义 — 语义成立，断言覆盖有缺口

- 代码核对：全部 `Allow` 字符串与 HTTP 语义一致（多方法路径以 `", "` 连接，顺序为 GET、PUT、DELETE 约定）；404 分支不写 `Allow`、不写 `Location`；405 会写 `Allow` 与 `nosniff`。
- 测试核对：13/16 端点存在 `Allow` 头断言；`GET /api/v1/admin/orphan-articles`、`POST .../publish`、`POST .../transfer` 仅断言 405 状态，未断言 `Allow` 值（见 S15-RE-F008）。

### 3.4 Origin→Session→CSRF→Principal 与 Trace 最外层 — 层级成立

- 组合：`protectOrigin(authorizer.Protect(session.AccessGeneral, observeAuthorization(...handler)))`（3515–3519），Origin 在最外；`HTTPAuthorizer.Protect`（309–341）在调用 resolver 前完成会话 Cookie 存在性检查、在 resolver 成功后做 `principal.Allows(access)`，`RequireCSRF: requiresCSRF(request.Method)` 由真实平台代码决定。
- 测试：`SecurityChain` 的 fixture 使用真实 `platform.OriginGuard`（1887、1870–1875）与真实 router；`AWIRERTR03RTR05OrdersAndStopsEverySecurityLayer`（5525–5635）断言 origin/session/csrf/principal 各层拒绝时后续层与 handler 均不执行。
- 精度限制：`session`/`csrf`/`principal` 三个事件由测试替身 `articleAuthorRootResolver`（1818–1839）与 `articleSecurityResolverSpy`（8796–8815）产生，CSRF 校验本身位于真实 session resolver 内，本轮未由真实 resolver 断言（见 S15-RE-F007）。

### 3.5 普通与 `-race` 均 0 fail / 0 skip — 成立

本次独立重跑（普通与 `-race`）均为 320 run / 321 pass / 0 fail / 0 skip / 20 个顶层测试，且顶层测试集合与 `root-normal-json-r02.log` 逐名完全一致（`Compare-Object` 无差异），无 `DATA RACE`、无 skip 事件、无 fail 事件。

## 4. Findings

### S15-RE-F001 — LOW：`{id}` 约束描述的归属不精确

- 位置：`16-endpoint-map.md:37`。
- 证据：该行称“`{id}` 段由 `exactArticleDeletePath` 约束”。实际代码中，moderation 两个端点走 `exactArticleModerationPath`（`main.go:3766`），orphan 的 publish/transfer 走 `exactOrphanActionPath`（`main.go:3772`），二者再委托给 `exactArticleDeletePath`（`main.go:3758`）。`/api/v1/users/me/articles/`、`/api/v1/articles/`、`/api/v1/admin/articles/`、`/api/v1/admin/orphan-articles/` 才直接调用后者（3628/3632/3644/3652）。
- 影响：仅文档精度。四种路径族的实际行为一致（非空、无 `/`、无尾斜杠、无额外段，否则 404），我已用 `ExactRouting_UnknownPathsDoNotRedirect`、`ExactPathsAndSharedAdapter`、moderation 的 `trailing slash`/`extra segment`/`missing id` 变体独立验证。
- 建议：改为“`{id}` 段统一经 `exactArticleDeletePath` 及其两个包装函数约束”。

### S15-RE-F002 — LOW：一个补充用例被归入“真实 root”覆盖，但绕过 router

- 位置：`16-endpoint-map.md:56`（publish 行），`lead-verification.md` 的“同包、真实 root”表述。
- 证据：`TestOrphanDispositionRoot_HTTPNOWAIT_ReturnsDatabaseUnavailableWithoutLoop`（`main_test.go:3474–3496`）直接调用 `authorizer.Protect(session.AccessGeneral, http.HandlerFunc(handler.PublishOrphan)).ServeHTTP(...)`（3491），不经过 `newLoginBusinessRouterWithTraceAndRecovery`，且该用例不在本次任何一次运行的 19/20 个顶层测试中。
- 影响：该引用不能为“publish 路由已注册”提供证据；同一行另外引用的 `RealHandlersEnforceSecurity/publish/*` 与 `ExactPathsAndSharedAdapter` 已充分证明注册与精确路径，故主张不受影响。
- 建议：从该单元格移除该用例，或标注为“handler 层补充，不经 router、不在本轮运行集”。

### S15-RE-F003 — LOW：覆盖表引用的子测试选择器在实际日志中不存在

- 位置：`16-endpoint-map.md:44–55`（`adminRootOldReadProjections/*`、`adminRootRouteMatrix/*`、`method_path_matrix`）。
- 证据：`adminRootRouteMatrix`、`adminRootOldReadProjections` 是传给 `t.Run` 的函数名（`main_test.go:1222`、`1700`），在 4 份日志中出现 0 次；实际子测试路径为 `TestProductionAdminArticleQuery/ExactMethodsAndCoexistence/{list,detail,method_path_matrix[/<method>]}` 与 `TestProductionAdminArticleQuery/OldConsumerCompatibility/read_projections/{member_list,member_detail,author_detail,orphan_list,orphan_detail}`（已在 r02 日志逐条确认存在且 pass）。
- 影响：按表中写法执行 `-run` 选择器会匹配不到用例，削弱交接时的可复核性；不影响端点确有断言的事实。
- 建议：覆盖表统一改用完整 `t.Run` 路径（或注明为表函数名）。

### S15-RE-F004 — LOW：`head`/`SOURCE_STABLE` 的 revision 语义不足以重建被审代码

- 位置：`index.json` 的 `head`、`source`、`source_stable`；`16-endpoint-map.md:3,12`；`lead-verification.md` 时点行。
- 证据：`git ls-files cmd`、`git ls-files internal` 均为空，`git status` 显示 `?? cmd/han-server/`、`?? internal/`、`?? go.sum`，`go.mod` 为 `M`；HEAD `6da8abb` 的提交信息为“chore: add ADD project framework, requirements, design and Codex configuration”，不含任何 Go 代码。`source` 只固定了 2 个文件，未包含 `internal/platform`（`TraceMiddleware`、`HTTPAuthorizer.Protect`、`OriginGuard` 所在）与模块依赖图（`go.mod`/`go.sum`）。
- 影响：本证据在“取证时点”有效（我已在本工作树复现），但第三方无法仅凭 `git checkout 6da8abb` 重建可测试树；若日后 `internal/*` 或依赖版本变动而两文件不变，`SOURCE_STABLE=True` 会被误读为“整棵被审代码未变”。
- 建议：在 `index.json` 中把 `head` 明确标注为“工作树快照，非可检出 revision”，并把 `internal/platform/http_security.go`、`internal/session/*`（若涉及）、`go.mod`、`go.sum` 一并纳入哈希清单；或在 S1.5 收口时提交代码并重新绑定真实提交。

### S15-RE-F005 — LOW：`-run` 表达式被脱敏，r01 无法逐字复现

- 位置：`index.json` runs 的 `command` 字段（`<11 tops>`、`<12 tops>`）；`16-endpoint-map.md:66–69`。
- 证据：r01 日志含 19 个顶层测试，r02 含 20 个（r02 多出 `TestProductionBusinessHandler_AWIRERTR05ThreeDeleteRoutesCarryRoleAndStatus`）；`index.json` 的 `run_events`/`pass` 与日志一致（313/314、320/321），但未记录顶层测试数与精确表达式。
- 影响：r02 可由本次指定正则精确复现（见 §5），r01 只能被“超集运行全部通过”间接支持；r01 自身不可逐字重放。
- 建议：后续轮次在 `index.json` 中保存完整 `-run` 表达式与解析后的顶层测试名列表。

### S15-RE-F006 — LOW：“唯一生产 router”的措辞过宽

- 位置：`16-endpoint-map.md:14,76`；`lead-verification.md` 首段。
- 证据：`main()` 除业务服务器外还启动 metrics HTTP server（`main.go:695–706`），并在维护模式下启动 recovery-control HTTP server（`main.go:746–768`，`newServer("recovery-control", controlHandler)`）。
- 影响：措辞层面。“进程内唯一生产**业务 API** router”是准确的；16 端点主张不受影响。
- 建议：改为“唯一生产业务 API router（业务监听端口）”。

### S15-RE-F007 — LOW：安全链次序断言的“CSRF 段”来自测试替身

- 位置：`16-endpoint-map.md:76`；`main_test.go:1818–1839`、`8796–8815`、`5583–5602`。
- 证据：`trace`、`origin` 由真实 `TraceMiddleware`/`OriginGuard` 路径产生（`SecurityChain` 用例），但 `session`/`csrf`/`principal` 三个事件由测试 resolver 自行按序 append；`AWIRERTR03RTR05OrdersAndStopsEverySecurityLayer` 中的 origin 拒绝亦由 seam 合成的 `protectOrigin` 返回（5583–5601）。真实的平台保证是：Cookie 存在性检查先于 resolver、`RequireCSRF` 由 `requiresCSRF(method)` 决定、`principal.Allows` 先于 handler（309–341）。
- 影响：主张在“层级边界”成立；若解读为“真实 session 实现内部的 Session→CSRF→Principal 次序已被断言”，则不成立。真实 resolver 的内部次序不在本证据范围。
- 建议：将表述限定为“平台层边界次序（trace→origin→会话入口→resolver→principal 访问→授权观察→handler）由生产代码断言；resolver 内部次序由替身模拟”。

### S15-RE-F008 — LOW：3 个端点缺少 `Allow` 头的可执行断言

- 位置：`16-endpoint-map.md:54,56,57`；`main_test.go:3394–3396`、`3513–3516`。
- 证据：`GET /api/v1/admin/orphan-articles`、`POST .../publish`、`POST .../transfer` 在运行集内只有 405 状态断言（orphan `method` 变体只比 `response.Code`；list 注册用例只比对 204），没有 `Allow` 值断言，也未断言该 405 未进入安全链。其余 13 个端点的 `Allow` 值有显式断言（`2202`、`1677`、`3622`、`5397` 等）。
- 影响：当前 `Allow` 语义由代码阅读+本次评审确认正确，但未来改动不会被测试捕获。
- 建议：为上述三条路径补充 405 + `Allow` 精确值 + “仅 trace 事件”的断言。

## 5. 独立执行记录

全部在仓库根目录执行；测试输出写入工作区外的 `%TEMP%`，未改动工作树。

| 命令 | 退出码 | 结果 |
| --- | --- | --- |
| `go vet ./cmd/han-server` | 0 | 无输出 |
| `go test ./cmd/han-server -run '^(TestProductionArticleAuthorRoutes_[A-Za-z0-9_]+|TestProductionAdminArticleQuery|TestModeration_Router_RegistersExactPOSTPaths|TestModeration_Production_RealHandlerSecurityAndRouting|TestOrphanDispositionRouter_ListRoute_IsRegistered|TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity|TestOrphanDispositionProductionRoutes_ExactPathsAndSharedAdapter|TestArticleBusinessRouter_AWIRERTR01RTR02RTR04UsesExactDeletePaths|TestProductionArticleQueryRoutesUseTraceSessionPrincipalAndCoexist|TestProductionBusinessHandler_AWIRERTR03RTR05OrdersAndStopsEverySecurityLayer|TestProductionBusinessHandler_AWIRERTR05ThreeDeleteRoutesCarryRoleAndStatus)$' -count=1 -json` | 0 | 320 run / 321 pass / 0 fail / 0 skip / 20 顶层测试 |
| 同上加 `-race` | 0 | 320 run / 321 pass / 0 fail / 0 skip / 20 顶层测试；`DATA RACE` 出现 0 次 |

- 重跑前后 `main.go`、`main_test.go` 哈希与字节不变（见 §1.1）。
- 顶层测试名集合与 `root-normal-json-r02.log` 完全一致（`Compare-Object` 无差异）。
- 未运行 `go test ./cmd/han-server`（整包）、未运行其他卡的测试。

## 6. 保留日志一致性

| 日志 | run | pass | fail | skip | 顶层测试 | ThreeDeleteRoutes | DATA RACE |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `root-normal-json.log` | 313 | 314 | 0 | 0 | 19 | 否 | 0 |
| `root-race-json.log` | 313 | 314 | 0 | 0 | 19 | 否 | 0 |
| `root-normal-json-r02.log` | 320 | 321 | 0 | 0 | 20 | 是 | 0 |
| `root-race-json-r02.log` | 320 | 321 | 0 | 0 | 20 | 是 | 0 |

- 与 `index.json` 的 `pass`/`fail`/`skip`/`run_events` 完全一致；无 skip 事件、无 fail 事件。
- r02 相对 r01 只多 `TestProductionBusinessHandler_AWIRERTR05ThreeDeleteRoutesCarryRoleAndStatus`，与 map §4 的说明一致。
- 说明：`pass` 数比 `run` 多 1，是包级 `pass` 事件，属正常。

## 7. 边界与未验证

本评审**不关闭 S1.5 整卡**，不改变任务状态、`requires` 或人的接收结论。本证据与本评审**不能证明**：

- 真实 Session/凭证解析与真实 SQL 行为；
- 真实 PostgreSQL、连接池与 NOWAIT 行锁行为；
- 应用角色 ACL / 数据库权限最小化；
- producer 的真实 `ReadCurrent` / `Publish` 语义；
- Case18 全局恢复隔离、Journal/旧备份；
- 部署后 E2E、真实网络、真实前端消费者；
- 任何性能、容量或生产就绪性结论。

本评审未做：连接数据库、访问外部网络、部署、E2E、清理任何产物、运行其他卡测试、逐字复现 r01 的原始 `-run` 表达式（已被脱敏，见 S15-RE-F005）。

残余风险：证据为未提交工作树快照，`internal/*` 与模块依赖未纳入哈希（S15-RE-F004）；若这些部分在后续轮次变化，本包不能自证“行为未变”。

现场说明：工作树存在与本对象无关的既有未提交改动（`.add/project/...`、`add.manifest.yaml`、`go.mod` 等）与未跟踪目录；本评审未触碰、未清理，也不对其正确性作任何结论。

## 8. 复审条件

出现下列任一情况需重新评审：

- `cmd/han-server/main.go` 或 `main_test.go` 哈希变化；
- 证据包重新出具（例如补齐 F004/F005/F008 所述 pin 与断言）；
- S1.5 整卡或 S8.2 等 `requires` 状态变更；
- 需要以 git revision 而非工作树快照复现时（当前不可行，见 F004）；
- 需要把本证据用于关闭整卡或对外声明生产就绪时（超出本包范围）。