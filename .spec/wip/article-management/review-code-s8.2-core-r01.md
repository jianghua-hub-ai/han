# S8.2 Core CodeReview R01：冻结七文件

日期：2026-10-28。实际独立 Reviewer：Erdos / add-reviewer；作者 Dalton。objectType=`code`。沿当前 development-delivery，不重新 Route，不下级委派，不替作者修复。

**CodeReviewGate：有界 PASS，无 material findings。** 两轮覆盖下列七文件完整实际 diff、两份新测试全文及必要调用链。U/C 普通、race 与必要旧组件回归真实执行通过；P 仅静态审查和 integration 编译通过。此结论不是全部九文件、S8.2 最终 G4、Case18 整体或任务接收/关闭 PASS。

## 1. 固定对象、基线和所有权

实际只读核 Han/main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。保留未提交用户内容及其他会话身份工作。报告开始及落盘前均确认目标报告不存在。

| 冻结路径 | 本轮实际 SHA-256 | 实际 diff + / - |
|---|---|---|
| `internal/article/query_service.go` | `D3151AC329C8C9EB8A1B8E00FF45A58CC467E06E1C9A00F9A63830048C0108B4` | 207 / 1 |
| `internal/article/handler.go` | `C684B4C874083035C1BC6B1F22DF81833D8DF490544E8586719C028B203D3750` | 94 / 0 |
| `internal/article/model.go` | `42B2BEEEAC0B4CB3B6E2138314956F2D1FCBD0E32D6E7B32BB2352C656E8851D` | 2 / 0 |
| `internal/article/policy.go` | `6CD1543BC2E715F76EE7B6C5CDDDB3153A9A425B84C55A366648E600B78C8774` | 1 / 1 |
| 新 `internal/article/admin_query_test.go` | `DBAF93B7600831D61DC74804D82905CAE0719014357A9107321DFCD8C3EF3F61` | 874 / 0 |
| `internal/article/policy_test.go` | `B67C708416E4CCBD988EBF43BAED45484177B6B6BCD54988E8EDE1461D7BB47E` | 29 / 0 |
| 新 `test/integration/article/admin_query_test.go` | `3ADC5BFA00410F2BF7B520C0035E0089B6E24889770BF6BD1F283A7E5B798263` | 934 / 0 |

证据根 B：`C:\Users\J00010385\AppData\Local\Temp\han-admin2-phase1-379a75dccaad4d34ad14ece04e0a2d4e`。本 Reviewer 唯一 Temp 写目录为 `B\review-core-r01`，以下个人验证文件均位于该目录。

实际读取 `phase1-stopped.json`、`phase1-hashes.json`；从 `phase1-before` 五份前像和当前七文件独立重建 unified diff，与 `phase1-seven-files.patch` 逐行一致。五份前像另与 Lead `han-admin2-impl-e5199290ceb54e33a7c310eb02ab8e21\before\internal__article__*.txt` 对应文件逐一核 SHA 相同；新两文件无前像。个人 `independent-seven.diff` SHA=`9352E8B2E7541F0CC63E76F92BA33ABA58029C956BB4ABE937BF48200CACE638`，逐文件旧/新 hash 和行数见 `diff-verification.json`。

七对象在自跑验证前、审查末次均为上表 SHA，零漂移。必要稳定依赖与作者固定 `fixed-source-v2/manifest.json` 的对应输入相符，末次再核无漂移。没有读取、哈希或冻结当前动态 `cmd/han-server/main.go/main_test.go`；也没有将其合法变化判作本轮对象漂移。历史报告/meta 内的 root 指纹仅作为历史证据消费，没有当作当前 root 结论。

## 2. 上游与方法

| 实际固定依据 | SHA-256 / 消费范围 |
|---|---|
| article design r21 | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA`；§3.3.9、§8.6 的 API、参数、状态、分页、身份、DTO、ETag、安全及七文件/系统门禁边界 |
| stories r10 | `492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9`；Story8 验收及其与旧读取消费者的关系 |
| test-plan r17 | `93071AB5E9E68ED976FA376EF52874E7B1A74DEE3FA9860561DD46A596A8DFA2`；§4.3.1、§7.7、§9.6、§11.1 的 Case/层次、数据资源、负控及分项门禁 |
| Design R03 | `195334B08EB9F106D1ABA2C1BCC5E0D21767295DD5CE33670BA449C1B2F86BF2`；r21 有界 PASS、七文件 ACK 与 Case18 阻塞边界 |
| TestPlan R02 | `600C9BC2AAECF668D50B9A96090250021B57FDF44AF9965FD6FF2440AB550A3D`；全文，规划与运行证据边界 |
| Harvey recovery/readiness 报告 | `213653A8A07EEDFFC65F07B834D5F78E6E66EB27C8FCE7C5EA2D2CB774FC2E4A`；全文，未重新扩大调查身份恢复或当前 root |

实际取 S8.2 原卡的对象、授权与验收段，不重审 Plan2 统计、不要求整 Tasks/Plan 或并行身份文档维持全文 hash。设计/规划文件的历史“main 尚未授权”不覆盖本轮用户已明确的动态 root2 窗口。

复用同会话已完整取得且仍有效的 `.codex/agents/add-reviewer.toml`、add-code-review、checklist、abstraction、安全红线，以及 coding/no-hardcode/observability/tracing、唯一启用 go-service 的适用正文；task-execution、multi-agent 和当前 delivery 合同沿用。本轮另完整刷新 add-code-review 正文，按 code 分支执行两轮；没有加载其他 Workflow、未来实施方法或全规则库。未将未执行的完整 profile/Manifest 声称 PASS。

## 3. Round 1：完整 diff 与测试断言

四份生产增量逐段核对；新 U 874 行、新 P 934 行全文读取，C 增量及旧保留区差异核对。不存在以测试名或作者摘要替代断言审查的情况。

| 风险 | 实际实现/断言及判断 |
|---|---|
| RawQuery/presence/错误优先级 | Handler 先可信管理员授权，再一次 `url.ParseQuery(RawQuery)`；任何 error 直接 422，丢弃部分 map；后续转换只消费该 map。U 的非法编码、重复编码键、分号及未知键编码错误都经实际 Handler，核 service/repository/identity 零调用。typed 零值 all 与 HTTP 显式空 422 分开；成功解析后 cursor 空/重复/无效为400，limit/view为422；Atoi 合法词法保留。旧 URL.Query 助手未改。 |
| 两级权限/依赖缺失 | Handler 与 service 均用可信 Principal/AccessGeneral/admin；普通 admin 无 initial-admin 前提。U 覆盖 member、suspended、onboarding、缺 Principal、直接 service 拒绝与零数据调用；nil 构造、空 Handler、nil service fail closed。受控 resolver 证明组件接缝，不证明真实 SessionStore SQL 或 production 注册。 |
| 严格身份整批失败 | 对全部非 pending 合法候选先收集当前 author 去重，再校验 active/suspended，最后投影/筛 view；非目标 view 不绕过此步骤。nil/缺项/空/非法/deleted/error 均503，后批失败返回零 page，不保留前页。U 六种后批故障核 HTTP503、无 items/cursor、停止扫描；匿名/held/pending 与 orphan source 不查身份。 |
| pending/非法状态 | pending 列表跳过、详情优先404且不查身份；其余非法状态整体500。新局部 internal cause 不保留 ErrArticleInvalidState，U 实际走 WriteArticleError 核500，同时旧 sentinel 仍409。P 用真实 SQL Scan 后 typed fault，不关闭 CHECK 或破坏 FK。 |
| keyset/游标/限额 | 原候选 SQL 不变；具体 repo 只扩四种普通候选 view，orphan cursor 仍拒绝。三 view 静态序列均有三时间戳和 ID 交错；U 显式 fixture、P 独立 SQL 为 oracle。密集过滤连续跨批，内部边界 last-scanned 保留 admin view，公开 cursor 为 last-returned，lookahead 下页仍出现；空/过滤尽/末页 [] 与 null，列表无整体 ETag。 |
| Admin12/隐私/同读 ETag | 原 DTO/Policy 未改业务语义，model 仅加 hidden cursor 常量。U/P 列表逐项先取 raw JSON 的精确12键，再对独立 fixture/SQL 逐值核；详情含匿名嵌套值、category/null、完整有序原因、Unicode/标记及时间。U 使用微秒值，P clock 来源为真实 PG UTC 微秒。详情一次 Article 读取，受控后续 writer 不与 body/version/ETag 拼接；无查询层写接口、worker 或新 authority。 |

特别复核 Case09：`admin_query_test.go:567` 的 P 载体对 active、suspended、held 三类初态，真实调用 `s22.command().Moderate` mark 和 restore，而非直接改 violation 字段；每阶段 `:592` 核三个 admin view、完整详情/类别/原因及 ETag，`:624` 核旧成员/作者入口与作者四字段/正文。orphan 旧集合另在兼容组核查。此处是代码/断言充分性判断，尚无该 P 用例运行 PASS。

特别复核 Case16：U 的 `adminUnitListFields/adminUnitAssertPayload`、P 的 `:800/:819` 对每个 raw list item 核12键并逐值对账；遍历完成还核独立预期序列。P oracle `adminPGViewOracleSQL` 独立从真实表构造字段及原因，没有调用被测 Policy 重算期望；不是仅检查解码到 DTO 后的字段数或只测详情。

## 4. Round 2：真实消费者、状态、SQL 与资源

沿真实链复核 `Handler → ArticleQueryService → PostgresArticleQueryRepository → identity reader → Policy → DTO/StrongETag → WriteArticleJSON/WriteArticleError`。旧 member/author/orphan 的方法体、接口/构造、原 SQL、DTO 和共用错误优先级未因本 diff 改变。候选公共底座扩 view 不将 orphan SQL 合并，不引入身份 JOIN；只在测试 oracle 使用 JOIN。

实际必要源码读取：

- `query_service.go`：新增完整代码及现役 repository、readCandidateBatch、validateCandidateOrder、readAuthorStates、member/author/orphan、SQL 与错误助手。
- `handler.go`：新增完整代码、可信 Principal、旧参数助手、MapArticleError、JSON/error 输出；`policy.go/model.go` 的状态、三投影、隐藏原因、codec/ETag/DTO 对应定义及变化。
- `internal/identity/article_reader.go` 全文：参数化 SQL、去重/canonical ID、真实 rows/Scan/Err/Close 与通用 deleted 状态语义；没有改此 reader 或身份所有权。
- `internal/platform/http_security.go:309` 起的 Protect/Principal/GET-CSRF 接缝，`http.go` 的 Trace/JSON 输出、`errors.go` 的 cause 与公共 envelope；`internal/session/store.go:109` 起的 Principal/Allows。真实会话 SQL、生产路由与全局 Guard 不在本执行证据内。
- `internal/article/command_service.go:145` 起 Moderate/事务/锁内更新/审计调用及相关投影；`repository.go` 的真实 InsertArticle/InsertDeletionOperation；用于辨别 fixture 写与 GET 的责任，不重新审整个治理历史。
- retained `notice_dispatcher_test.go:33–143` 的资源/低敏错误/UUID/pool/search_path/迁移/clock helper；`postgres_state_test.go:70–145` 的合法用户、Article、pending operation 先插入、command 和身份读取 helper；`internal/article/query_test.go` 对应 fixture/受控 resolver 定义。未展开 SMTP 历史或运行旧全 integration。
- `migrations/0003_article.sql` 的 articles/deletion_operations CHECK、复合 FK 和排序索引；0001～0009 的九个实际文件存在且 hash 与固定 snapshot 相符。其他迁移仅核 fixture 按完整序列加载及指纹，未将其全文业务作为本次审查目标。

P 静态资源核查：专属 fixture 在 `:60` 注册 cleanup，再独立 autocommit CREATE，成功后立即记 retained schema，之后 `:114` 同 tx 执行0001～0009；失败不 DROP。pool 配置对所有连接固定 search_path，先核 loopback:5433 / 18.6。pending helper 先插合法 operation，再 Article，用户 FK 不破坏；missing/deleted 等是实际 reader 完成后的受控结果变换。

`adminPGDatabase.Query/Rows` 委托真实 Query/Scan/Rows.Err，然后注错，真实 rows 关闭；detail Scan 完成后才运行 writer。作者状态 barrier 通过真实 Query 完成点、channel 和独立 pool 连接提交 UPDATE，不用 sleep；cleanup cancel→group.Wait→检查 acquired→pool.Close。迁移/插入 tx 有 commit/rollback；错误日志只输出类型/SQLSTATE，不输出 DSN/原始协议错误。上述为静态检查，实际 PG 锁、取消、join/pool 收敛和失败 schema 保留仍须运行确认。

零业务写证据分层：P `:751` 快照覆盖 articles/users/audit_events/transfer_notices/recipients/deletion_operations，`:762` 比较成功、拒绝、故障 GET 前后；作者/治理/ETag writer 用例单列，未把其合法变化算成 GET 副作用。认证合法锁 User/撤销 Session 继续归身份链，不将查询零写扩为整条链零数据库写。

## 5. 真实独立执行与 raw

个人 runner 仅在 child env 移除 `HAN_TEST_POSTGRES_DSN`，没有读取 User DSN、PG 连接、integration testbinary 执行或 PG -list。全部命令 workdir 为仓库根；普通/race 均使用 count=1。JSON 逐行解析，仅计 Test 节点，不把包 pass 或父子重叠当 leaf。

| 个人 label / 命令 | exit | top / run=pass / leaf / fail / skip |
|---|---:|---|
| `uc`：`go test ./internal/article -run '^(TestAdminQuery|TestAdminCursor)$' -count=1 -json` | 0 | 2 / 135 / 114 / 0 / 0 |
| `uc-race`：上述命令加 `-race` | 0 | 2 / 135 / 114 / 0 / 0 |
| `old-uc`：`go test ./internal/article -run '^(TestArticleQuery|TestVisibility|TestCursor|TestStrongETag|TestAuthor|TestOrphan)' -count=1 -json` | 0 | 44 / 150 / 133 / 0 / 0 |
| `old-query`：`go test ./internal/article -run '^(TestVisibleScan|TestListArticles|TestGetArticle|TestUpdateArticle)' -count=1 -json` | 0 | 17 / 44 / 37 / 0 / 0 |
| `old-query-race`：`go test -race ./internal/article -run '^(TestVisibility|TestCursor|TestVisibleScan|TestListArticles|TestGetArticle|TestUpdateArticle|TestOrphan)' -count=1 -json` | 0 | 56 / 189 / 165 / 0 / 0 |
| `vet`：`go vet ./internal/article` | 0 | stdout/stderr 均空，无测试计数 |
| `p-compile`：`go test -c -tags=integration -o B\review-core-r01\admin-query-review.test.exe ./test/integration/article` | 0 | 仅编译；0 个 P 测试被执行，不能当作 P PASS |

首条旧回归前缀没有覆盖 List/Get/VisibleScan，因此另精确补 `old-query` 并跑上述实际旧查询/投影/cursor/orphan race；未将任何未命中的前缀算作覆盖。不同回归命令有交集，不将表中计数相加冒充独立用例总数。

U 实际108 run/pass、94 leaf、17风险组；C 实际27 run/pass、20 leaf，五种合法 view 各 round-trip，20个有向互拒均执行。个人普通/race 的完整 run 集合相同，逐 Case 的 U/C leaf 与作者 `phase1-case-to-leaf.json` 对应集合相同。P16组仅从源码核断言存在，不计运行；R不读取/运行。

个人 stdout SHA-256（每项另有同 label `.meta.json`、`.stderr.log`；meta 保存完整 argv/exit/duration/child env 处理及 raw SHA）：

| raw | SHA-256 |
|---|---|
| `uc.stdout.jsonl` | `64D0A557F16DC3C432267C27B770BE629BB1B0FD08FC5DB501E94D895FC49FAC` |
| `uc-race.stdout.jsonl` | `A815966E806626EAFCAD7108E4B83713F9DBBEB61B51C57FFBB7F0BF37E9FA4D` |
| `old-uc.stdout.jsonl` | `FD21805653AF2857BDE7804FC8B200CF7F0E3FBCD0733327045CE6B3CD09E978` |
| `old-query.stdout.jsonl` | `27E924006ADF8FE295A023DDBACD24751CD75F7F82EFDFB0F1C75973293FC9BD` |
| `old-query-race.stdout.jsonl` | `F965395AC5AB374F385170E2664BC509D3BF0CB4D70B6B027EBB74BF27104007` |
| 全部个人 stderr、vet/compile stdout（均0字节） | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| 未执行的编译 binary | `0BBC767970232631B9870BDEFBB05DAC472B83AA96EC301F92FBEED990A7FEF5` |

`own-run-sets.json` SHA=`785B9137D7BB69AD8D0F5B08598B5FB9BBEB64CEB9F23CFD43BEC90FFA382B36`；`own-raw-index.json` SHA=`065170FA46F232778D659486E48911C33C0734EDC2305A83AD33085AFE8E3987`。Reviewer 新增 schema=0，没有 PG runtime 资源或后台服务被创建；编译 binary 保留未运行。

## 6. 作者证据与有效负控核证

独立复算 `phase1-raw-meta-index.json` 23条记录的69个 meta/stdout/stderr hash，零不符。实际解析最终 UC 普通/race、旧两组普通/race、codec RED 和 m1/m2 的 JSON；最终 UC 同为135/114，旧 query 普通/race189节点165leaf，旧 author/moderation 普通/race99节点86leaf，均0fail/skip。它们是作者执行，经本 Reviewer 核证，不冒称个人执行。

| 固定对照 | 实际差异、失败与判断 |
|---|---|
| codec RED | `admin_hidden` Encode 被旧 allowlist 拒绝，raw 有实际断言失败；不是编译失败或缺环境。最终同一 C 测试现源 GREEN。 |
| m1 RawQuery | 只将新 Handler 的 ParseQuery 错误拒绝改为丢弃 error。实际 `invalid_05` 即 `view=all&other=%ZZ`，HTTP200而预期422；exit1、3父子 run/fail、1leaf。raw SHA=`41F5CE074E2E79C1CDA817E13ED5A6C156B08BAFB605426B0E10D6907DD92605`。 |
| m2 identity | 只将严格完整性检查的 dependency error 改为补 active。后批 missing 实际 HTTP200而预期503；exit1、3父子 run/fail、1leaf。raw SHA=`ABE8C20E4561BF08E65F3C161A3E91EDB2AEFCA67EE15B1EAA611CDE70E092AD`。 |

实际读并重建两份 mutant diff；mutation 描述的 baseline/mutant/overlay SHA 均与文件相符。m1 mutant=`51AFC9E3971B5B3B8D216105C977510F4D2B92C7E626279429076AAFE00655BA`，m2 mutant=`478C61CECC26078782687370E60BA1D89CC26249F2C8A22626088FE2E2D8F4F1`。两 overlay 各152个替换输入，其余对应固定 snapshot 的输入 hash 相符；这152不是完整167项 Go/SQL/JSON/module manifest 的同义计数。失败确在已编译测试内，stderr为空，无硬编码 t.Fatal 或环境失败；个人现源 U/C 同对应 leaf GREEN。早期 prepare-overlay KeyError/编译失败不计 RED。

本轮未执行负控，更没有执行第三 SQL/分页 PG 负控；现有两条 U 层失败不能替代后续真实 PG 第二批 fault 或跨时间 SQL mutation。P runtime/raw/schema/资源清单与第三负控继续未验。

## 7. 指纹、限制与交回

个人 `final-source-binding.json` SHA=`733E6518FA21DC0FFCD473130BC9F5DC46365846B812804AE68DBAEF2D0A35DC`，记录七对象执行前后、21项必要输入（10份源码/测试 helper、go.mod/go.sum、9 migrations）与固定167 snapshot 的对应 hash、六份上游、零 drift、逐 Case leaf 核对和 root 排除。具体依赖 hash 另在 `stable-readset-first.json`，上游在 `upstream-first.json`。七对象是在自跑前固定；21项依赖在检查时与固定 snapshot 逐项比对并在末次复核，不宣称每个子命令启动时另捕获了完整编译闭包或全库无漂移。148保护集、167 snapshot、174 Lead 集合均不冒称本 Reviewer 全库读集。

缺陷 findings：无；本轮新增方向决定：无；非阻断风格建议：无。未因恢复上游的诚实 pending 新建七文件代码缺陷，也未以此放宽内层安全/状态合同。

仍未验证：P16组实际 SQL/reader/HTTP 运行及 ordinary/race、真实故障 rows/取消/join/pool 收敛、schema 失败保留与实际数据回查；第三 SQL/分页行为负控；动态 root2、同实例 factory/slots/exact routes/GET-DELETE 共存的最终版本；真实 SessionStore SQL；完整 Manifest/profile、真实 ACL/producer/Journal 卷、Host/Browser/部署/E2E/容量与人的验收。

全局恢复请求隔离与完整恢复后生产开流仍是上游未交付的两分项：完整恢复/S16/二次核验/MarkConsumed 后才开放，失败或不可判定继续隔离。原 `503 platform.recovery_in_progress`/零文章下游义务保留；Case18整体、S8.2接收/关闭继续 blocked。七文件不拥有新 authority/全局 Guard，局部 PASS 不修改这些条件，不重开 S2.2/S7，不缩减 S1.5 四文件16端点或继承身份豁免。

后续交回条件：Lead 核本报告和个人 raw/最终指纹；P 获对应窗口后取得真实运行/负控/资源证据，root2作者 stopped 后固定最终对象，再由同一独立 Reviewer 核相关九文件装配与运行增量。若本七文件或必要合同再变，固定新修订后只复审受影响差异。此报告不签任务关闭、不授权下一写入或运行、不代签人的接受。

唯一仓库写入为本报告；没有改动源码、测试、Task/Plan/Design 或配置，没有 Git 写、外发、服务、PG、Browser、下级委派或清理操作。所有个人日志/分析/binary保留于授权 Temp。

**stopped：七文件有界两轮独立 CodeReview 完成，交 Lead 收口。**
