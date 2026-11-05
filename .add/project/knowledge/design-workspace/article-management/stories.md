---
feature_id: article-management
artifact: design-stories
status: draft
revision: article-management-stories-ir01-r11
created: 2026-10-20
last_updated: 2026-10-29
source_design: design.md
source_design_revision: article-management-design-ir01-r23
---

# article-management 设计 Story 切片

本文件列出 8 个与已确认用户故事一一对应的业务设计切片，以及 1 个不新增用户故事的 production wiring 后继切片，供后续 task-decompose 使用。它不创建 task、批次、认领、实施或完成状态；永久设计合同以 [design.md](design.md) 为准。

r08 仅按 design r19 同步共享依赖与 Story2/3/8 的普通管理员读取验收和交叉引用；其余切片正文保持原样，不声明全量重新对账、实现或验证完成。Lead 已依据用户持续授权采用核心方案并明确设计 WRITE START；本修订仍为 draft，未取得精确人的成果确认或独立设计 Review，不新增 checkpoint。

r09 仅同步 design r20 对 `AM-DES-R19-ADMIN2-R01-F001` 的有界修复：新列表 Handler 完整 RawQuery 解析与 Story8 HTTP 验收，保留 Pauli R01 CHANGES REQUIRED，待原 Reviewer R02；未声明 finding CLOSED、Review PASS 或人的 accepted/checkpoint。

r10 仅纠正 Story8 对现役 readiness/recovery 请求链的误称，并按 design r21 §8.6 明确 Case18/整卡关闭的上游门禁；其余 Story/组件合同不变，原 R01/R02 历史结论保留，本增量待独立评审与实际接收。

r11 仅按 design r23/architecture r32、ADR-0027 r02 同步共享恢复依赖、原 Story4 和 Story8/后继接线前提。输入 r10 SHA256 `492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9`；Main 授四文件写窗，保持 draft、待非作者评审/精确人的接纳。原 StoryID、用户故事归属、r22计数语义及其他旧Story正文不作全量重新对账，不新增业务特性/Task。恢复边界不迁移，identity currentr13不提升accepted。

**本轮恢复增量现行元数据（2026-10-29）**：初稿/返工说明中的“待独立评审／两项 OPEN”保留为各自提交时的历史；RECOVERY-DES-R01 CHANGES REQUIRED → Kuhn [R02报告](../../../../../.spec/wip/recovery-runtime/evidence-design-r01/review-r02/review-report.md)／[receipt](../../../../../.spec/wip/recovery-runtime/evidence-design-r01/review-r02/receipt.md) 有界设计复审 PASS，F001/F002 均 CLOSED（仅设计层），OPEN/LIMITED/NEW=0；Main 已全文核报告、receipt、修复 diff 并技术接收。R02 结论仅适用报告列明的受审 SHA，不倒签原 R01，也不预签本次最终元数据核验。

**本轮实际产出者**：Banach／add-feature-designer／runtimeID `01a18389-57ab-7b70-8c86-82a61e7729f1`（Agent 来源，非人的签字）；版本仍 r11、status 仍 draft、人的精确接纳 unknown，不代 implementable/checkpoint/Code/Case/test PASS。ADR/index 的最终 SHA 刷新由 Arch 另串行处理。

## 共享依赖

- 所有 Story 依赖 [共享基础](design.md#331-共享基础)：Principal、Session/CSRF、公共错误、集中 CursorCodec、强 ETag/version、VisibilityPolicy、AuditStore。
- 所有写 Story 依赖 PostgreSQL 事务和 Article 检查约束。
- 管理 Story 依赖 identity-auth 的管理员 Principal；转移依赖目标资格、墓碑联系和 retained_email 清理端口。
- Story8 两个普通管理员 GET 复用同一 ArticleQueryService、ArticleQueryRepository、identity reader 和 QueryConfig，既有构造/成员/作者/orphan 接口不变；独立 admin_hidden cursor 与严格身份完整性检查仅用于新管理员读取。
- 所有永久删除 Story 依赖 ADR-0020 的 `deletion_pending`、DeletionOperation、独立 `article-deletion/*` tombstone、finalizer 与旧备份重放。
- production wiring 后继切片依赖 ADR-0008/0021/0022、唯一 `cmd/han-server`、共享 DB pool 与现役双 server lifecycle，不引入第二 composition root、MQ 或外部 scheduler。
- S16 provider 依赖唯一 20 字段 v1 fixture；canonical 固定前 16 字段，identity 只消费完整 DTO并重算摘要。
- 恢复期间 readiness 关闭；S1–S16 任一门禁失败都不得开放，任何 Story 都不提供文章恢复。
- 独立 article Anchor 写者/纯读 snapshot、可信迁移/协调与精确 published closure 以 design §3.4.6/7 为唯一合同，identity只经ports消费；全部调用共享同一Journal/source/root，production无旧未锚定fallback。

## Story 1：创建文章（US-ARTICLE-01.1）

**切片目标**：已认证 active 用户提交标题和正文后直接得到 published/clear 的文章。

**验收**

1. POST 只从 Principal 取得作者，不接受客户端 author_id。
2. 标题或正文缺失/全 Unicode 空白或超过 title 200/body 20000 rune 时返回 422；请求体超过 262144 bytes 或解码前发现非法 UTF-8 时返回既有 400，数据库无新行。
3. 成功返回 201、Location、ETag；不存在草稿或下线状态。
4. 数据库失败不得显示创建成功；日志不含标题和正文。

**依赖**

- 身份 Session/CSRF 接入。
- AM-Q1 已固定纯文本、Unicode 空白/rune 长度、262144-byte wire 上限与非法 UTF-8 fail-closed 合同；本 Story 仍是待实施切片。

**设计引用**

- [§3.3.2 创建文章](design.md#332-story创建文章us-article-011)
- [Article 数据结构](design.md#数据结构设计)

## Story 2：发现并阅读文章（US-ARTICLE-02.1）

**切片目标**：已认证用户通过稳定倒序 cursor 浏览成员可见文章，并读取详情。

**验收**

1. 列表按 `created_at,id` 倒序；候选分批扫描，批量读取作者状态并持续过滤到 `limit+1` 可见项或耗尽。
2. 列表不返回 body；详情返回标题和正文。
3. Cursor 基于最后返回可见项；静态混合状态集合中可见文章不漏不重，同时间戳稳定。
4. 不存在、deletion pending、永久删除或任一隐藏约束成立时统一返回 404，不泄露原因或内容。
5. 浏览无写副作用；成员列表过滤 identity 作者状态缺项并继续扫描，成员详情缺项返回 404，身份调用故障整请求 503；不把缺项当作 active 或匿名。
6. Story8 的管理员读取复用此分页底座但不采用成员可见性过滤；其严格身份缺项 503、完整 DTO 和独立 cursor 验收归 Story8，不改变本 Story 的成员语义。

**依赖**

- VisibilityPolicy 与 identity `GetAuthorStates`。
- 分页默认/上限和容量 AM-Q2。

**设计引用**

- [§3.3.3 发现并阅读](design.md#333-story发现并阅读文章us-article-021)
- [分页选择](design.md#241-分页选择)
- [正交状态矩阵](design.md#341-正交状态与有效可见性)

## Story 3：修改现有文章（US-ARTICLE-03.1）

**切片目标**：作者基于受控自有详情的当前 ETag，完整替换自己仍有归属、未进入永久删除协议的文章内容，包括 violating 文章。

**验收**

1. `GET /api/v1/users/me/articles/{id}` 只向当前作者返回 `title,body,moderation_status,editable` 和强 ETag；违规文章为 `restricted/true`，deletion pending 不返回。
2. PUT 要求单个强 If-Match；拒绝 weak、多值和 `*`，缺失 428，不匹配 412。
3. 只有作者可修改；普通非作者和管理员非作者继续拒绝。
4. 修改只改变 title/body/updated_at/version，不改变归属、publication、violation/category、source 或 deletion 状态。
5. 作者响应不得出现 `violation_state`、`violation_category`、`hidden_reasons` 或账号治理状态；经 Story8 的 `GET /api/v1/admin/articles/{id}` 读取仍保持既有 12 字段管理员 DTO 和同一 Article version 的详情 ETag，不新增字段。
6. violating 文章修改后仍仅管理员治理详情和作者受控详情可见，成员详情继续 404；三入口读取对照不得用单独的 Policy/DTO 测试冒充管理员 GET 已实现。
7. PUT 与创建共用 262144-byte N/N+1 和解码前 UTF-8 门禁；失败时内容、version、updated_at 与治理状态不变。

**依赖**

- ArticleStatePolicy、行锁、ETag middleware。
- Story 2 提供当前资源和 ETag。

**设计引用**

- [§3.3.4 修改文章](design.md#334-story修改现有文章us-article-031)
- [并发更新选择](design.md#242-并发更新选择)

## Story 4：作者删除文章（US-ARTICLE-04.1）

**切片目标**：作者永久删除自己的普通文章，不留下可恢复正文。

**验收**

1. DELETE 要求作者身份和单个强 If-Match。
2. T1 将 Article 置 deletion_pending 并创建 DeletionOperation，立即对所有业务查询隐藏。
3. `article-deletion/*` tombstone 依 design §3.4.6 完成 staged/file+dir sync→精确CAS pending→不覆盖发布/目录sync/清已核alias→CAS committed及独立回读后，finalizer 物理 DELETE、AuditEvent 和 operation 完成同事务；文件sync单独不确认耐久。
4. 只有 tombstone 与物理删除双完成返回 204；Journal 成功 DB 失败时 503 且继续收敛。
5. 旧备份开放前全量重放 tombstone；Journal 不保存内容、摘要、邮箱或其他 PII。
6. 无软删除正文、版本历史或恢复入口。
7. Ready variant 固定完整 20 字段、全部非 null，`inventory_scope` 为 string；canonical 仅为前 16 字段，fixture digest 固定 `169e0f...926a7`。
8. DeletionReplayRun 唯一 `(run_id,namespace)`，backup 首绑不可变；并发双 backup 只能一方成功，另一方稳定 failed。
9. stable input_digest 不变时复用同一 evidence_ref/evidence_digest/completed_at；covered input 变化才追加 evidence 并更新 current。
10. Unavailable variant 仅含 version/run/backup/namespace/status/reason_code，status 限 pending/unknown/failed，禁止 canonical/digest/ref/time。
11. ReplayAndVerify/ReadStatus 每次只返回一个 variant；共享 fixture 与负向 variant 测试必须通过。
12. 正常append在pending前同步禁新普通permit/三worker claim；unknown升级同一关闭原因并真实join，未确认不物理删除/204；健康仅解除自身临时原因，不清恢复/其他故障。
13. normal构造器/Find/ReadAll/snapshot/history零初始化、零CAS、零清pending；受控Replay仅精确committed operation幂等读，startup Reconcile可用内部许可而不先开HTTP。
14. publication与evidence/current原子提交，normal按Store精确ref/digest/time/不可变来源及可信前缀读取历史，不刷新Facts；fresh仍拒stale。无来源旧行/旧合法前缀不能自证，可信迁移只在关闭/排空/独占/namespace锁下执行，原payload/HMAC/计数/wire不改。

**依赖**

- Article Deletion Journal、完整 keyring、backup inventory、DeletionReplayRun/EvidenceStore、共享 v1 fixture 与 AuditStore。
- UI 二次确认 AM-Q3 不影响 API 安全。

**设计引用**

- [§3.3.5 作者删除](design.md#335-story作者删除现有文章us-article-041)
- [永久删除选择](design.md#243-永久删除选择)
- [S16 replay evidence](design.md#347-deletionreplayevidence-与恢复开放门禁)

## Story 5：管理员删除他人普通文章（US-IDENTITY-03.2）

**切片目标**：管理员可永久删除他人普通文章，但不能修改他人文章或绕过专用治理路径。

**验收**

1. 管理员使用普通 DELETE 路径并提供 If-Match。
2. 普通文章必须 deletion active、有作者、published、clear；违规或无归属返回 invalid_state。
3. 管理员调用 PUT 修改他人文章必须拒绝。
4. 删除复用 pending/tombstone/finalizer；双完成才 204，最小审计不保存正文。

**依赖**

- identity 管理员 Principal。
- Story 4 的永久删除基础。

**设计引用**

- [§3.3.6 管理员删除](design.md#336-story管理员删除他人普通文章us-identity-032)
- [权限与 endpoint 总表](design.md#接口设计)

## Story 6：处置无归属文章（US-IDENTITY-03.8）

**切片目标**：管理员对待处置无归属文章执行匿名公开、转移或永久删除，并维护 retained_email 清理条件。

**验收**

1. 三种写操作只接受 `author_id=NULL` 且 `orphaned_from_user_id` 非空的文章。
2. 违规文章不能直接恢复公开；转移后继续隐藏且违规事实不变。
3. 目标用户必须 active；仍有归属的文章拒绝转移。
4. 公开/转移事务完成后退出集合；无归属删除由 finalizer 物理删除后统计并清理 retained_email。
5. 零文章、多文章、违规文章、deletion pending 和 kept_deleted 重放矩阵全部可验证。

**依赖**

- identity 转移目标/墓碑联系与清理端口。
- AuditStore、TransferNotice/Recipient、canonical 删除端口和 Article Deletion Journal。

**设计引用**

- [§3.3.7 无归属处置](design.md#337-story处置无归属文章us-identity-038)
- [canonical 删除端口](design.md#343-canonical-删除端口)
- [地址清除顺序](design.md#344-转移与通知地址清除顺序)

## Story 7：发送文章转移通知（US-IDENTITY-03.9）

**切片目标**：已提交转移通过持久通知记录 best effort 告知新作者和已删除原作者。

**验收**

1. 两路地址在转移事务中写入两个独立 pending recipient，公共响应和日志不披露邮箱。
2. 通知 purpose 固定为事务性转移通知，禁止营销。
3. ProcessingGate 缺实际输入时不 claim、不增加 attempt_count、不连接 SMTP，转移不回滚。
4. 每收件人状态固定为 `pending/retry_wait/processing/accepted/failed_terminal`，含 claim owner、lease、next attempt、attempt/max。
5. claim 与结果保存使用短事务，SMTP 调用期间不持有数据库事务；双 worker 和 lease 到期可恢复。
6. 瞬时错误重试，永久错误或最大尝试进入 failed_terminal；两个收件人独立。
7. SMTP accepted 后崩溃可能重复投递，作为 at-least-once 残余风险审计。
8. 生产 SMTP 只允许隐式 TLS 或强制 STARTTLS，并校验证书链与主机名；首期无人工重发 endpoint。

**依赖**

- Story 6 的转移事务和快照。
- ProcessingGate、Mailer、SMTP TLS 合同。

**设计引用**

- [§3.3.8 转移通知](design.md#338-story发送文章转移通知us-identity-039)
- [TransferNotice 数据结构](design.md#数据结构设计)
- [隐私分析](design.md#54-隐私风险分析)

## Story 8：违规文章治理（US-IDENTITY-03.12）

**切片目标**：管理员通过既有治理列表/详情查看对象，判定暴力/色情违规、解除违规或永久删除，且不覆盖其他隐藏约束。普通管理员两个 GET 归属现有 SR-IDENTITY-03 / US-IDENTITY-03.12 的查看与治理验收，也支撑 US-IDENTITY-03.2 删除前读取，不新增 US；实现与责任合同见 design §3.3.9/§8.6。

**验收**

1. 判定只允许 category `violence|sexual`，写 violating/category/version。
2. 成员在判定提交后立即得到统一 404；管理员仍可查看治理详情。
3. restore 只清除 violation/category；author_suspended 或 orphan_hold 仍保持隐藏。
4. 违规删除复用 pending/tombstone/finalizer，双完成才 204；finalizer 物理删除与 AuditEvent 同事务。
5. 重复判定、clear 状态 restore 和错误删除入口返回 invalid_state；并发冲突返回 412。
6. `GET /api/v1/admin/articles` 的 all 返回全部合法 deletion-active 对象，包括匿名、held orphan、violating、作者 suspended；violating 只筛当前违规，hidden 只筛 AdminProjection 的非空 hidden_reasons。两者可重叠，不因 orphan 有专用接口而从普通管理员集合排除。
7. 新管理员列表 Handler 在授权后先 error-aware 解析完整 RawQuery，任意解析错误丢弃部分 map、整体用既有 422 article.validation_failed 拒绝，service/repository/identity 均零调用；不能通过 URL.Query 或原丢错助手把错误变成遗漏。仅解析成功后从同一完整 map 判断 presence/重复：遗漏 view 显式转 typed all；单个精确 all/violating/hidden 通过，显式空、重复、未知、空白/大小写变体仍 422，cursor/limit 各沿原 400/422。typed 零值由 service 规范为 all，但不承载或消除 HTTP 解码错误；旧 member/orphan 助手与消费者不改。
8. 列表沿用现役配置和 `(created_at,id)` 倒序，跨批扫描到 `limit+1` 匹配项或耗尽，内部 cursor 按最后扫描项推进、公共 cursor 按最后返回项推进；空列表 `items=[]/next_cursor=null`。hidden 使用新增 admin_hidden，member/admin_all/admin_violating/admin_hidden/admin_orphan 全部互拒；非法 cursor 400，limit 沿既有解析/上下限 422，不加生产数值默认。
9. 新管理员读取对非空作者缺项、nil/非法/deleted 状态或身份故障整请求 503，无空成功页或半页；详情 pending/不存在先 404、不读身份、不返回正文/ETag。非法 active Article 状态或候选顺序错误 500，不吞错筛掉。成员与 orphan 现役语义不变。
10. 列表每项与详情均复用既有 12 字段 AdminArticleView，包含正文、完整类别/原因；详情 Header ETag 对应同一读取的 Article version。GET 复用实际 Trace→精确路由→HTTPAuthorizer/SessionStore→AccessGeneral 链与可信管理员 Principal，不要求 CSRF；普通有效管理员可用，非管理员 403。每用户 Gate 不等于全局恢复隔离，后者仍为必须交付的上游保障。新增查询零业务写，不扩称认证链零数据库写；状态 race 不承诺长快照。
11. 分别取得服务/HTTP 单元、codec、真实 PostgreSQL→HTTP 和唯一 production root 四层新增证据；既有 S2.2 SQL/投影/分页结果不替代新 GET 验证。原 S1.5 四个 contract 文件、16 endpoint 和全部关闭条件不缩减；S1.6 三项旧 Handler 接线由 Planner 另行登记，共享 main/main_test 写入串行，不重开旧有效卡。
12. 实际 Admin 列表 Handler 的 HTTP 验收覆盖 `view=%ZZ`、`view=hidden&view=%ZZ`、`view=all&cursor=%ZZ`、`view=hidden&limit=%ZZ`、`view=hidden;view=violating`：整体 422、零 service/repository/identity，不能退回默认或部分成功。对照正常遗漏/单值合法 view 及完整解码后的 view 422、cursor 400、limit 422；N001 通过真实 WriteArticleError 核新读取内部 500，不污染旧治理 409，不以 service PublicCode 或 typed 枚举测试替代 HTTP 输出。

Case18 的恢复/readiness 子断言及 S8.2 整卡接收/关闭仍待上游实际生产承载及同版证据；design r23/identity r50已固定中央软件合同，但设计完成不等于已接线。完整恢复前不得开放业务，失败/不可判定保持隔离，不能删除或豁免该断言、以 startup/Session 403 或 fake 503 记成功。内层四生产文件与 U/C/P 三测试的组件合同不受本次补充影响，实际运行/写入以 Lead 对应窗口授权为准；组件完成不等于 Case18 或整卡完成。两GET必须经真实root返回503/零文章下游，正常Auth/Trace/exactrouter及开放链保持；精确上游责任及恢复条件见 design §8.6。

**依赖**

- identity 管理员 Principal。
- VisibilityPolicy、AuditStore、ETag/version 和 Article Deletion Journal/finalizer。
- 读取复用现役 ArticleQueryService/候选 SQL/identity reader/QueryConfig；权限和 User.status 仍归 identity，隐藏原因仍由 Policy 派生。具体五生产文件与四类测试建议见 design §8.6，未登记、认领或实施任务。

**设计引用**

- [§3.3.9 违规治理](design.md#339-story隐藏解除判定与删除违规文章us-identity-0312)
- [正交状态矩阵](design.md#341-正交状态与有效可见性)

## 后继 Story：Article production composition、router、config 与 finalizer worker

**切片目标**：在唯一 `cmd/han-server` 中把已存在的 Article 永久删除能力接入生产配置、Journal provider、三条 DELETE 路由和受控 finalizer worker；这是横切实现切片，不新增业务 US，也不表示前述 8 个业务 Story 已实施。

**验收**

1. `startupConfig` 集中解析全部 Article 环境变量；关键 Journal root、current/retained key 与对应 HMAC key、worker ID 和所有调度值缺失或非法时 fail closed。默认、范围与交叉约束精确遵循 design r10，并在 DB parse/connect/ping/migrate 前完成可前置的配置、权限、平台、namespace/domain 与 keyring 校验。
2. Journal provider 使用固定 `article-deletion/v1` namespace、`article-deletion-hmac` key ID 和 `han/article-deletion/v1/tombstone\x00` HMAC domain；可复用 loader/file-safety 实现，但不得接受 Recovery 环境变量、key material、目录、writer lock、记录链或 domain。owner-only 目录、排他 writer lock、active 写入/active+retained 验证及 key material 清零均可验证。
3. 只扩展既有 production business router：精确注册 `DELETE /api/v1/articles/{id}`、`DELETE /api/v1/admin/articles/{id}`、`DELETE /api/v1/admin/orphan-articles/{id}`；Trace 仍最外层，正确方法保持 Origin→Session→CSRF→Principal/授权→Handler。已知路径错误方法为 JSON 405 + `Allow: DELETE` 且不进入安全链，未知路径为 JSON 404，既有 Identity routes 不回归。
4. PostgreSQL discovery 稳定选择 `pending_journal`、`journaled` 和 lease 到期 `finalizing` operation，按 `started_at,id`、固定 batch 收敛。T1 后/Append 前崩溃可由 operation 不可变事实幂等发布 tombstone，再绑定、claim/lease 和 finalizer；不伪造事实，不引入 MQ、外部 scheduler 或第二进程。
5. worker 单并发；HTTP Serve 前同步即时调谐一次，之后由 ticker 与容量 1 合并 wake 驱动。每 operation 有预算，每 wake 有批次数上限，transient 使用有界 full-jitter 退避，terminal security/invariant 错误触发同一 lifecycle 失败。
6. shutdown 先停止新 discovery/claim，并让已开始工作只在 worker budget 与全局 shared deadline 较早者内 drain；worker、两个 HTTP server、listeners、Journal/keyring 和共享 DB pool 由同一 owner 协调，DB pool 最后关闭。
7. 构造失败按长期资源获得顺序严格 reverse unwind；保留首个根因并附加 cleanup error，不 double-close，不泄漏 goroutine/listener，不提前关闭 DB pool。
8. 后续测试规划必须能唯一覆盖 production config、Journal/keyring/domain、三条 router、worker discovery/claim/lease/即时首轮/ticker+wake/batch/backoff/cancel/shutdown，以及逐步构造失败回收；本切片不修改 test-plan，也不宣称测试已执行。

**依赖**

- ADR-0008、ADR-0020、ADR-0021、ADR-0022 与 accepted Article boundary。
- 既有 `cmd/han-server` production composition/lifecycle、`internal/article` DeletionOperation/Journal/Coordinator/Finalizer、共享 Session/Origin/Audit/DB pool。
- 容量、吞吐与告警目标仍为 TBD，不阻塞 design r10 已固定的 interval/batch/lease/budget/backoff/shutdown 实现基线。

**设计引用**

- [生产配置、Journal provider 与路由组合](design.md#生产配置journal-provider-与路由组合)
- [永久删除协议与 finalizer](design.md#345-永久删除协议与-finalizer)
- [可测试性](design.md#61-可测试性)

## Story 依赖顺序

```text
Story 1 创建 ─┬─> Story 2 浏览 ─> Story 3 修改
              └─> Story 4 作者删除 ─> Story 5 管理员删除
共享正交状态 ─> Story 8 违规治理
canonical 删除 ─> Story 6 无归属处置 ─> Story 7 转移通知
Story 4/5/6/8 永久删除基础 ─> production wiring 后继 Story
```

该顺序只表达设计依赖，不是实施批次。task-planner 应结合测试、迁移和共享基础再拆任务。
