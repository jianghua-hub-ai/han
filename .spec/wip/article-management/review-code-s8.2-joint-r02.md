# S8.2 Joint CodeReview R02：固定九文件与局部运行

2026-10-28。Erdos / 实际非作者 add-reviewer；作者 Dalton，CODE、Root、LOCALPG 均已 completed/stopped。objectType=`code`。续接同一 development-delivery，不重新 Route，不委派，不替作者修复。

**有界 CodeReviewGate PASS，无 material findings。** 本结论覆盖固定九文件已实现的内层查询、唯一 Root 接线及本报告分层列明的运行证据。两轮检查已完成；Core R01 同版全文结论复用，追加 Root2 完整 diff/消费者和真实 PG 证据。**Case18 整体、S8.2 接收/任务关闭继续 BLOCKED**：全局恢复请求隔离及完整恢复后开流两项上游尚未交付，本 PASS 不解除它们，也不是最终完整 G4/生产验收或人的接受。

## 1. 固定对象与输入

工作区 Han/main；实际核 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。当前未提交用户内容及并行身份工作保留。开始及落盘前确认本报告路径不存在；只写本报告与个人 Temp。

| 固定路径 | SHA-256 |
|---|---|
| `internal/article/query_service.go` | `D3151AC329C8C9EB8A1B8E00FF45A58CC467E06E1C9A00F9A63830048C0108B4` |
| `internal/article/handler.go` | `C684B4C874083035C1BC6B1F22DF81833D8DF490544E8586719C028B203D3750` |
| `internal/article/model.go` | `42B2BEEEAC0B4CB3B6E2138314956F2D1FCBD0E32D6E7B32BB2352C656E8851D` |
| `internal/article/policy.go` | `6CD1543BC2E715F76EE7B6C5CDDDB3153A9A425B84C55A366648E600B78C8774` |
| `internal/article/admin_query_test.go` | `DBAF93B7600831D61DC74804D82905CAE0719014357A9107321DFCD8C3EF3F61` |
| `internal/article/policy_test.go` | `B67C708416E4CCBD988EBF43BAED45484177B6B6BCD54988E8EDE1461D7BB47E` |
| `test/integration/article/admin_query_test.go` | `3ADC5BFA00410F2BF7B520C0035E0089B6E24889770BF6BD1F283A7E5B798263` |
| `cmd/han-server/main.go` | `6B45B5509A2D01C9F5DF0393CA00CAD5C8430353B5F849BF6E71BA3A47C52AC4` |
| `cmd/han-server/main_test.go` | `B53DD34D57AD6A2797E9FC080FD90034FC52CB26675686CB075D3153F193A9BC` |

复用 [Core R01](review-code-s8.2-core-r01.md)，实际 SHA=`9C84342FFB0965D6185931EB5C768FE23AEAD3C201F696A7653CA79A721A1619`；七对象无变化，不重跑全面 Review/U/C/P编译。原报告保留其当时 P 未运行的边界，本 R02补充同版运行结果。

固定上游重新核 SHA、复用已取得正文：design r21=`A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA`（§3.3.9/8.6）；stories r10=`492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9`；test-plan r17=`93071AB5E9E68ED976FA376EF52874E7B1A74DEE3FA9860561DD46A596A8DFA2`（§4.3.1/7.7/9.6/11.1）；Design R03=`195334B08EB9F106D1ABA2C1BCC5E0D21767295DD5CE33670BA449C1B2F86BF2`；TestPlan R02=`600C9BC2AAECF668D50B9A96090250021B57FDF44AF9965FD6FF2440AB550A3D`；Harvey全文报告=`213653A8A07EEDFFC65F07B834D5F78E6E66EB27C8FCE7C5EA2D2CB774FC2E4A`。未重审身份文档、Plan统计或恢复全链设计。

方法继续复用同版已完整读取的 Reviewer Role、add-code-review code 分支、checklist/abstraction/安全红线、coding/no-hardcode/observability/tracing、唯一启用 go-service，以及当前 delivery/task-execution/multi-agent合同。没有借父会话读取代替本角色读取，不加载无关 Workflow 或未启用 profile。

## 2. 证据目录与源绑定

以下均为实际目录，原件只读：

- A：`C:\Users\J00010385\AppData\Local\Temp\han-admin2-phase1-379a75dccaad4d34ad14ece04e0a2d4e`；Core前像/diff/UC/m1/m2，个人 `review-core-r01`。
- B：`C:\Users\J00010385\AppData\Local\Temp\han-admin2-phase2-51e9e9a916544471b1871872aad1febb`；Root前像/patch/raw/meta，个人 `review-root-r02`。
- P：`C:\Users\J00010385\AppData\Local\Temp\han-admin2-localpg-0d583b77ed3b4273bde1c7082b441961`；完整PG包/固定snapshot/m3，个人 `review-joint-r02`。

174项现役 Go/SQL/JSON/module/README输入按 Lead `lead-localpg-start-r01.json` 的 source map 独立逐项核当前hash。Root阶段每个个人命令、PG阶段每个个人命令均保存174项before/after；九对象包含其中。末次174项及六份上游均零drift。P的 `fixed-inputs-174` 全174文件逐项匹配当前固定map；没有把旧148保护集、167 snapshot或152个Go overlay替换项冒称174。

Root独立重建 `B\review-root-r02\independent-root.patch` 与作者完整patch逐行一致；main.go相对前像48新增/16删除，main_test.go为572行新增测试加唯一旧Allow一行替换（573新增/1删除）。前像分别是原S1.6 `2B75C96E61F5BF758F68935F4656238CD8D891D6DC646C32B55EFD0C1A5A6A14` / `092BD3BD78169A40780169A27948D3BD1C6E423406E2C8E658B0EFC59BF3BCBA`。Core五份旧前像/两份新文件的完整diff重建事实及旧/新hash保留于R01，不混用活动版本。

## 3. Round 1：全部增量和断言

Core四生产增量、U/P两个新文件全文及C增量已在同版R01完成。RawQuery先完整ParseQuery，任意error整体422且零service/repository/identity；同map后置presence/cursor400/view-limit422。可信普通admin在Handler/service两级授权；pending详情优先404，其他非法权威状态500且局部cause不连接旧409 sentinel。严格身份先覆盖整批active候选，再筛view；nil/缺项/空/非法/deleted/error整503，后批故障不返半页。三view/五cursor、复合keyset、last-scanned/last-returned/lookahead、Admin12独立值oracle及同次读取ETag均按真实断言核过。

Root完整生产diff、新增测试全文和旧Allow上下文另独立核查：

| 关注点 | 实际判断及证据 |
|---|---|
| 唯一构造/同一依赖 | `main.go:2913`起原repo/identity reader/config构造同ArticleQueryService，经新增buildProtectedArticleQueryHandlers给Admin/member/author构造。没有另一router/pool/reader或新authority。actual_factory_references不替换终端Handler，除observer唯一引用外，实际比较Handler私有依赖、service.repository、repo/reader.database、SessionStore.transactions及同pool，逐项核config；identity闭包捕获关系同时由真实构造实参链核对。 |
| 槽位/copy | query5→7、article14→16、router20→22：新增query[5,6]→article[14,15]→router[20,21]。旧member→[9,10]、orphan→[13,14]、author→[18]；九mutation→[4,5,6,11,12,15,16,17,19]保留。copyTail起点和长度无错配。 |
| 安全链 | 原Trace最外层→exactrouter→同HTTPAuthorizer/Session接缝→AccessGeneral→AdminHandler；query包装无Origin/CSRF/IfMatch前置。可信Principal拒绝零文章下游，GET query角色不提升权限。九写包装保留Origin→Session/CSRF→权限→Handler；旧IfMatch拒绝未变。 |
| 路由/兼容 | collection仅exact GET；detail GET/DELETE且Allow完整；violation/restore不被详情吞掉；空ID/额外段/相似前缀404，已知错method405先于Session，缺Handler不伪成功。旧Allow扩为GET, DELETE对应新增合同。S1.6/Identity/member/author/orphan旧入口回归实际执行。 |
| 证据层次 | factory构造使用未连接测试pool；成功/业务错误另以受控read ports安装真实service/AdminHandler，经真实production factory/router/security链执行。resolver替身只证认证接缝/传播，不证真实SessionStore SQL；P真实SQL→Handler也不单独证明production root注册。两类证据分开记录。 |

没有以测试组名、observer标签、编译成功或作者摘要替代行为审查。Root详细两轮准备记录 `B\review-root-r02\stage-root-preparation.md`，SHA=`C4BC3893BB5DFB6BE5FBDB53A520D8B25AE4B72D21463B3972087B463DBC8DB7`，保留actual readset、负向入口、旧消费者与原raw链。

## 4. Round 2：实际调用链、PG数据与资源

沿 Handler→ArticleQueryService→Postgres repo→identity reader→Policy→DTO/StrongETag→公共JSON/error复核；原三投影/旧member缺项处理、orphan集合、治理409及接口/构造保持兼容。生产候选SQL没有身份JOIN或view筛选；测试独立SQL oracle负责对照集合和值。schema未改，不新增默认/缓存/长事务、业务写接口或恢复authority。

P真实载体与固定源码对应，fixture独立autocommit CREATE并立即记录，再完整0001～0009迁移；每连接固定search_path，合法用户/FK/CHECK，pending先建DeletionOperation。故障wrapper先执行真实Query/Scan/Rows.Err或真实identity reader，再注错；非法状态在真实Scan后typed fault，不破约束。上下文、rows.Close、tx commit/rollback、barrier/group、pool链已在R01静态核过，现补实际通过/cleanup日志。

Case09个人普通/race真实运行active/suspended/held三初态的现役CommandService mark/restore，每阶段核三个admin view、详情/category/完整原因/ETag及旧成员/作者投影，保留suspended/held约束。治理fixture写不计GET副作用。

Case16个人普通/race真实运行详情及三个view的raw list item精确12键、逐字段独立SQL oracle、匿名/null/[]、Unicode/标记/UTC微秒与遍历序列；没有只看解码DTO的字段数或仅测详情。Case05独立三时间戳/交错ID序列；Case10实际reader后第二批六种身份故障整503；Case14实际Query/Scan/Rows.Err/detail Scan/cancel五分支；Case15真实六表SQL快照核成功/拒绝/故障GET零业务写。

作者其余10个P组按完整普通/race raw和已审同版源码消费，包含身份/治理跨请求事实、独立连接barrier、细节优先级、scope、ETag后续writer与旧消费者。允许逐批当前事实及并发变化，不将其扩为全页长快照/线性一致；动态writer与静态零写快照分开。

### 4.1 保留schema与资源证据层

独立从实际raw重建schema台账，逐schema核CREATE提交→九个迁移begin→migration commit→created=true/retained=true及cancelled_joined_pool_closed日志顺序。与作者p-retained-schemas的schema集合和case归属一致，无缺项：

| 来源 | 普通 | race | mutant | control | 合计 |
|---|---:|---:|---:|---:|---:|
| 作者固定PG包 | 28 | 28 | 1 | 1 | 58 |
| Reviewer六组独立运行 | 17 | 17 | 未重跑 | 未重跑 | 34 |

92个是本报告两方这些运行所记录的schema数，不是整库现存schema普查；均保留，失败mutant也保留。没有DROP/清理/角色/GRANT/新DB/实例/服务。个人完整schema名/case/日志位置在 `P\review-joint-r02\p-schema-resource-ledger.json`。

资源结论限于固定源码中的rows-close/acquired-connection等断言确实通过，以及每schema实际cancel/join/pool.Close后记录的cleanup日志。没有逐counter数值输出，不虚构精确读写/连接/事务计数或额外监控探针。零写依据是独立SQL快照断言通过及相应低敏日志，不公布正文/身份数据。

## 5. 个人真实执行、命中和原始日志

Core R01个人U/C普通/race各2top、135run/pass、114leaf、0fail/skip（U108/94、C27/20），article vet/P compile及必要旧UC回归的原命令/raw见原报告，未重跑。

Root阶段个人在仓库根运行精确 `go test ./cmd/han-server -list '^TestProductionAdminArticleQuery$' -count=1`，普通/race的 `-run '^TestProductionAdminArticleQuery$' -count=1 -json`，以及旧Root前缀 `^(TestProductionArticleAuthorRoutes|TestProductionArticleQuery|TestArticleBusinessRouter|TestProductionBusinessHandler|TestModeration_Router|TestOrphanDispositionRouter)` 的普通/race/count1/json，另 `go vet ./cmd/han-server`。每child移除DSN，全部exit0：R各1top/93run-pass/76leaf/0fail-skip；旧Root各20top/180run-pass/148leaf/0fail-skip；vet输出为空。四R风险组实际leaf为18/11/36/11。各普通/race集合一致，分别与作者最终run集合及风险map逐项相同。

本轮个人LOCALPG仅从既有User注册项取HAN_TEST_POSTGRES_DSN注入child env，不取Process DSN替代，不把值写CLI/文件/日志或系统env。先以个人Temp helper调用pgx ParseConfig检查main及全部fallback均literal loopback:5433，exit0；不连接。随后**现役fixture在CREATE前**实际查询endpoint/version并要求PG18.6。没有另一个独立live探针，测试成功不被冒称第二次独立endpoint验证。

个人实际命令（同一六组选择，普通/race各一次）：

```text
go test -tags=integration ./test/integration/article -run '^TestAdminQueryPostgres$/(StaticCompositeKeyset|CurrentModerationTransitions|StrictIdentityWholeFailure|SQLRowsCancel|ZeroBusinessWrites|TwelveFieldsAndPrivacy)$' -count=1 -timeout=8m -json
go test -race -tags=integration ./test/integration/article -run '^TestAdminQueryPostgres$/(StaticCompositeKeyset|CurrentModerationTransitions|StrictIdentityWholeFailure|SQLRowsCancel|ZeroBusinessWrites|TwelveFieldsAndPrivacy)$' -count=1 -timeout=8m -json
```

两次exit0，各1top、6groups、34run/pass、28leaf、0fail/skip；普通/race完整run/leaf集合相同，并恰等于作者全P中的对应六组子集。不是零测试，也不计包pass为测试节点。

| 个人组 | 每次实际leaf |
|---|---:|
| StaticCompositeKeyset | 3 |
| CurrentModerationTransitions | 3 |
| StrictIdentityWholeFailure | 6 |
| SQLRowsCancel | 5 |
| ZeroBusinessWrites | 1 |
| TwelveFieldsAndPrivacy | 10 |

个人P stdout SHA：normal=`456936068AC7C68DFDDE2302EF0C93C8EB7CD1964E0D230B1B0DB60E227C84B2`；race=`3FD144C97A00EDFFFF16EE645CFEEDA3C97B20AF5B9733AB907D8529168EFC12`。stderr和两次preflight stdout/stderr均0字节，SHA=`E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。个人helper只做解析，build命令及其exit0/raw/174before-after也保留；不是业务测试或新依赖。

Root个人stdout SHA：normal=`2DF030040B217300E957F4C8E8B403F27165CA4B0E877863F71FD47BE89808F6`；race=`8DCEF6C432C73424124A9059807D3379FEDBF964A5668AEAA7C3594EF737C4F6`；旧normal=`29A2EEC6CB6D7B4F09C8BE0D09D9A8AF910524E34F72D18B802DF64AC145012E`；旧race=`92C2FDE87A80B4783AD2A516F4B76BDD2A7A1385B2EC4656594D1E3632FAB105`。

个人P精确argv、exit、时刻、rawhash和每child174项before/after在同label.meta.json；`review-raw-index.json` SHA=`E75CE1C5A91C83FBC4DF0395F21D6D5838A4E5BE807995B44A722F971A6F5E10`；`p-run-verification.json` SHA=`40ECB787205723F4F6ABF88E38AC6716A09FEEE0106C78E25798D5BA215768BB`；schema/resource ledger SHA=`93A41E631670AA159BC2F8D13507218A610C5D67F8BA8B300081072CA0874385`。不把多命令交集或普通/race两份集合相加冒称独立场景总数。

## 6. 作者全量证据与三类行为负控

原Core23meta/69hash、Root15meta/45artifact的独立核证保留于前两阶段。Root入口RED真实list404/detail405而期望Session边界401，实际2leaf失败；对应旧main与main_test.red指纹相符；编译/脚本失败未计RED。Root四份最终meta174 before=after=当前，最终普通/race raw集合与个人一致。

P固定包6labels/28artifacts SHA逐项独立复算无不符。四次作者普通/race/mutant/control各174项before=after=当前；snapshot174也逐项一致。全P普通/race实际各1top、16groups、99run/pass、83leaf、0fail/skip，集合一致；16个Case×P的实际leaf集合与p-case-to-leaf完全对应，schema侧表与raw重建一致。作者运行不是个人执行。

作者P normal raw SHA=`16AE602B9528C18438A7C14A7E9EC4D4D4D596CA15424CA03371BCDACC75B08E`；race=`2BB2731B48CABA24C864BBC697C9CA672BBFF1DEEA92785F20948A2632AB5542`；mutant=`A5840301C80374468077069BF00C476D670B2C66C03B5A916AA5893EE3108C79`；control=`CC4459DA8F0C50C200156ADAF828501D403DBC98EB0410D39E2A742808F11D9E`。

三负控分别证明不同风险，未混为现源产品失败：

- m1/m2沿Core证据：只丢弃RawQuery error导致实际HTTP200≠422；只误补active导致实际HTTP200≠503，均编译后断言RED。当前同leaf GREEN。
- m3独立读actual diff、mutation描述、174固定snapshot及152项overlay：只删 `listArticleCandidatesSQL` 的 `OR created_at < $1::timestamptz` 一行，保留同时间UUID分支。mutant SHA=`D04780BC2231A2B810A6F2568DAF7E6406918C491270218AC71A3A6011A5CE4F`，overlay SHA=`06C05F339FB015CA0F97B98110866F2EAA1B7F9F55AA8CD5AACFF6439A79B29D` 均匹配；其余151个替换指向固定同版输入，无未知/漂移项。不是改断言或t.Fatal硬失败。
- m3命令实际使用 `-overlay .../m3-sql-older-time/overlay.json`、integration tag、精确 `^TestAdminQueryPostgres$/^StaticCompositeKeyset$`。真实PG/HTTP all、violating、hidden三个leaf均在`:466`报复合keyset丢失/乱序，exit1、5父子run/fail、3leaf；不是编译/缺环境失败。control同3leaf exit0、5run/pass；个人六组也含同3leaf且GREEN。失败与control各1schema均保留。Reviewer没有重复跑mutant制造额外schema。

`P\review-joint-r02\author-input-proof.json` SHA=`15E791ABC066C18CB88F328161CA5984130E3631911747C86FB0E65D7D22D799`保存artifact/snapshot/overlay/四meta核证；完整负控原件仍在P目录。

## 7. Readset、Gate与交回限制

本角色实际完整七文件diff/新U-P全文及必要旧定义读集见CoreR01；Root完整diff/新增572行全文、唯一旧Allow、factory/query/mutation/authorizer/router/path helper与受控fixture读集见个人Root记录。外部源码仅复用同版已核的article repo/command/policy/DTO、identity article_reader、platform HTTP/security/error、Session Principal及retained fixture/migration必要片段；没有全库预读、重审身份历史、生产恢复authority或未来方法。174项hash绑定不等于174份正文审读，也不等于所有工具链/生成缓存均被指纹覆盖。

本轮追加完整读取固定PG stopped/index、case/leaf、schema/resource/frozen-inputs等侧表；逐行解析全P及mutant/control和个人两份PG JSON；用raw独立重建集合、schema/case及迁移/cleanup顺序，核实际source/overlay。此处证据已补足原Core的P未运行项，但不扩为部署或跨介质恢复验证。

最终源/证据绑定 `P\review-joint-r02\joint-source-evidence-binding.json` SHA=`5296E9BF2C0E607344D45DBAD8CD39F18864DFEC4A42F38A7B7F4A4D7FEA1E9B`：含174before/after、六份上游、PG原件SHA、个人raw/leaf/schema索引、零drift及Case18边界。Root个人readset/source绑定、每项raw和原完整集合留在B个人目录；Core原证据留在A个人目录。

新增缺陷 findings：无；待决定的本轮新方向：无；非阻断风格建议：无。未制造新的finding ID。若九文件或必要合同后续修订，固定新hash，只复审受影响diff/消费者/回归；修复仍交原作者，不由Reviewer代修。

| Gate/范围 | 结论 |
|---|---|
| 九文件已实现的局部查询、Root接线及分层运行 | 有界CodeReviewGate PASS；本报告列明个人/作者证据 |
| 规划U17/C1/P16/R4，名义38层映射 | 相应已实现局部测试有实际命中；R Case18仅前三分项，不能说38项业务验收全部完成 |
| Case18真实factory/Trace-router/Session-Access接缝 | 有局部证据；resolver替身不是SessionStore SQL |
| Case18全局恢复隔离/完整恢复后开流 | 上游未交付/未验，继续BLOCKED，无skip/fake503/豁免 |
| Case18整体、S8.2接收/TaskClose、最终完整G4 | 继续BLOCKED，不由本代码PASS解除 |
| Lead Manifest7 | 本Reviewer未重跑；未收到完整固定最终回执，不将进行中结果签为完成 |
| S1.5/人的接纳/部署与外部真实层 | 原责任保留，不代签、不关闭 |

未验证：真实SessionStore SQL与全局恢复权威→隔离→完整恢复/S16/二次核验/MarkConsumed后开流生产连接；完整16endpoint、真实ACL/producer、Journal真卷/恢复、Host/Browser/部署E2E/容量与人的接纳。没有把诚实的上游pending列成九文件新增代码bug，也没有因此放宽既有权限、路由或数据合同。S2.2/S7不重开，S1.5四文件16端点不缩，不继承身份豁免。

唯一仓库写入为本报告；未改九文件、metadata、上游、配置或他人内容，未stage/commit/push、委派、外发、部署或清理。全部个人schema、原始日志、snapshot/overlay及Temp helper保留。

**stopped：九文件两轮独立R02完成，交Lead核真实结果与后续门禁；本Reviewer不关闭任务或代签人的验收。**
