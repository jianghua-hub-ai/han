# S1.5 Components CodeReview R01

2026-10-28。实际非作者 add-reviewer Erdos；作者 Dalton。objectType=`code`，沿当前 development-delivery，不重新Route、不委派、不代修。技术包、修订及metadata状态由 **Lead交接**，以下区分本角色独立核验、作者执行与Lead执行，不写成用户逐项直接提供或验收。

**CodeReviewGate：CHANGES REQUIRED，仅本次四文件组件增量。** 两轮完成，发现一项 **MEDIUM / AM-S15-COMP-R01-F001**：wire边界预期随生产常量共同漂移，违反固定262144-byte合同的实现仍可令整个组件包GREEN。当前原生产代码没有观察到该超限放行缺陷；finding属于新增测试的检错盲区。独立普通/race与tagged vet均通过，不能替代此问题的修复/复审。

## 1. 固定对象、依据与源绑定

实际核工作区 `D:\AID\VSCode\Codex\Han`、main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。四文件原先不存在，全文件即本轮actual diff，统一contract tag / articlecontract包：

| `test/contract/article/` 下路径 | 独立核SHA256 | 全文行数 |
|---|---|---:|
| `rest_contract_test.go` | `1386903086B0399C39CE33BCBDF3D802B72105AE51E4CE6D58374A924CD433D7` | 858 |
| `internal_ports_test.go` | `DC6791D4793ABE67FE17B83736FC77E55C347859B13A4FB5071E6AC51BF64F25` | 880 |
| `projection_contract_test.go` | `D7585EE66538EECA4364BF01CC2BE3A4AC69D98C9818560D6533457E8A572033` | 274 |
| `replay_status_producer_test.go` | `B519DDE178CB3F4357560681490FD00639549E8CA0B369E2EFB01785488D8FDC` | 282 |

四份全文2294行均实际读取；独立重建 `/dev/null → current` diff与交回patch逐行相同（个人保存LF副本，不将换行规范化后的hash冒充原patch字节hash）。原patch实际SHA=`4D4B6ECF424C57EADF568E90F161063D42A0E3C5D15E6D14CD21E086FEE84711`；个人 `independent-four-new.patch` SHA=`D8B1010889D073D5BBEEC4BF8F2B81216BBBF4E940C952A62D578396C830F834`。

独立核上游：test-plan r18=`BCBCFDD8CD1747540726A46C35D565FF1CC211AAB43F6059145474568841DE42`（重点完整§4.7.1）；Pauli TestPlan R01=`85A9E111677DAC31E4A99A31D07F1FB8CFBB3E263CF1574196BFF9EA30D50A9E`（有界规划PASS及证据边界，不是CodeReview）；design r21=`A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA`；stories r10=`492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9`。本轮补读固定wire合同§2.4.4及r18相应要求；其余同版业务合同复用前轮已得正文。

同版完整Reviewer Role、add-code-review/current delivery、checklist/abstraction/安全红线、coding/no-hardcode/observability/tracing和唯一启用go-service适用规则继续复用。没有预读其他Role、未来E2E方法或全规则库。

作者根 B=`C:\Users\J00010385\AppData\Local\Temp\han-s15-candidate-c63dac5fc52d4ea0af1d51efb9c76c18`；固定交接在 `B\handoff-r02`，个人唯一Temp写目录为 `B\review-components-r01`。独立核 `fixed-178-r02.json` 与当前178项、`fixed-inputs-178-r02`全部副本完全相同；旧174又与Lead初始baseline逐项相同。每个个人命令均保存178项before/after，末次源码零漂移。hash绑定不代表178份全文均被审读，也不解决共享root writer的未知事实或产生root写授权。

## 2. Finding

### [MEDIUM] wire边界预期与生产常量同源，固定上限漂移时整包假绿

- finding ID：**AM-S15-COMP-R01-F001**；状态：**OPEN**。
- 对象/位置：本次固定 `rest_contract_test.go:512–513`，`restWire` 的N/Nplus1 fixture长度均由 `article.MaxArticleWriteBodyBytes` 推导。
- 依据：design r21 §2.4.4（:328/:332）明确 `N=262144`，实际N+1 payload octets必须400且不调用下游；r18 §4.7.1（test-plan.md:506）要求创建/替换共同冻结此边界。该值是固定外部合同，不能由待测实现重新决定预期。
- 触发：仅在Temp overlay将生产 `handler.go:22` 的 `MaxArticleWriteBodyBytes = 262144` 改为 `262145`，其余生产及四份被审断言原样不变。原测试随之把N/N+1改成262145/262146，四top仍286run/pass、236leaf、0fail/skip，exit0。
- 实际后果：独立固定262145-byte合法JSON+尾随空白见证，经现役Create/Update Handler，在原生产源码下均400/零下游；同一见证在mutant下分别变为201/200且各调用下游一次。由此证明新增组件测试无法发现共用wire上限的合同漂移。这里没有声称原生产代码现在错误，也没有声称mutant通过了全Manifest。
- 最小修复方向：在测试侧独立固定262144合同值，构造两个入口的实际N/N+1请求及零下游断言；可以同时断言公开生产常量与该合同值相等，避免两个预期再次跟随实现变化。仅改原四文件范围内所需测试，不改生产上限或上游合同。
- 检错证据：个人 `wire-limit-mutant/mutation.patch`只改一行数值；生产mutant SHA=`D8E7B2324EBC131F0C8967C180FCB4C35039B9F0724D17ADB3AD83A3A9F66F90`。原四测试均指向固定snapshot。另加的独立见证只在Temp虚拟overlay文件中，不写共享tree，不作为作者原测试冒用。
- 复审条件：原作者修复后交固定新SHA/diff；同一数值漂移负控须令新增合同断言真实RED，原源码普通/race完整四top及tagged vet GREEN，N与N+1对Create/Replace均实际命中。由本独立Reviewer核原finding；不能只提高计数、改断言接受新上限或以原Manifest GREEN关闭。

个人最小对照链（全部contract tag、count1、JSON，child移除DSN）：

| label / overlay | 命令范围与exit | 实际节点/leaf |
|---|---|---|
| `wire-mutant-suite` / mutant-original-tests | `go test -overlay <个人Temp>/wire-limit-mutant/mutant-original-tests.overlay.json -tags=contract ./test/contract/article -count=1 -json`，exit0 | 原4top，286run/pass、236leaf、0fail/skip，集合与原GREEN完全相同 |
| `wire-control-witness` / control-witness | 同scope，`-run '^TestReviewWireLiteralBoundary$'`，exit0 | 3run/pass、2leaf，原源码拒绝固定262145-byte请求 |
| `wire-mutant-witness` / mutant-witness | 同一独立见证/同一固定字节数，exit1 | 3run/fail、2leaf，Create=201/downstream1，Replace=200/downstream1 |

以上均编译后真实Handler行为，不是编译错误、缺环境或无条件hardfail。原四断言的overlay为156个固定Go输入，仅替换handler；独立见证另用157项映射增加虚拟测试，当前共享路径 `test/contract/article/reviewer_wire_boundary_test.go`不存在。补充见证用于证明盲区，没有代作者修复被审测试。

## 3. Round 1：完整四文件与风险充分性

逐段核全新增文件、实际fixture/capability、独立oracle与失败断言。除F001外，未发现第二个有具体触发/后果的material finding，不以风格偏好扩返工。

| 对象 | 实际核查与边界 |
|---|---|
| REST16动作 | 实际公开Handler构造，creator同时实现replacer/Moderate，删除用PermanentArticleDeleter；各success/error/missing Principal与三DELETE in-progress都真调用对应方法。核ID、target、cursor/limit/view、IfMatch、Principal、parent marker及response Trace传入context；创建201/Location、八字段表示、同源ETag，作者四字段与Admin12、列表envelope、transfer两recipient pending表示。table不冒充production注册。 |
| Wire/条件请求/错误 | 除F001的定值盲区外，UTF8、未知/重复/尾随JSON/type、Content-Type与literal Unicode的真实解码/零调用断言成立；body-free入口、八写入口IfMatch缺失/weak/多值/星号/畸形与412映射、真实ETag matcher、真实内容命令stale边界均有对应检查。19个独立message/status/retry golden、field_errors字段与code、trace/header/nosniff/低敏cause实际核查。 |
| Admin优先级/写响应失败 | RawQuery整体422先于字段规则且零port，授权拒绝先于非法query；另用真实QueryService核三view、typed零值、非法读状态500/pending404/缺身份503，旧治理sentinel仍409。响应Write失败实际走已返回Create结果的encoder/logger，核命令一次与记录一次，不当作业务回滚。 |
| 端口八组 | 真实公开consumer而非仅反射：内容权限/角色、真实identity reader规范UUID/typed参数去重顺序/稀疏结果/Query-Scan-RowsErr-Close；canonical adapter显式转换不同result类型，caller context/tx不被自提交；真实TransferCommand及Dispatcher消费脚本SQL事实。错误/顺序/零下游/资源断言按对应层检查。 |
| 四种投影 | 真实Policy→JSON，5/8/4/12顶层精确键、user/anonymous嵌套键、逐字段fixture值、null/[]、原因顺序和正文原值；错误身份/状态真实Policy拒绝。隐私canary与精确键双重约束，不把预期DTO标签反射当独立oracle。 |
| producer | 公开Provider.ReadStatus→受控ReadCurrent→真实encoder→共享decoder/canonical。Ready20全值/无null、固定fixture golden、canonical16与metadata排除；Unavailable6不调用Ready摘要；坏evidence failclosed及九类consumer mutation均真命中。未预期Store/Source方法显式失败，不实现Begin/Publish/authority状态机。 |

Lead预检补强属于作者修订历史，本Reviewer实际核其当前断言及前后patch，不把预检意见伪造成已独立关闭的finding。UUID用含a–f的合法大小写输入，核实际 `[]pgtype.UUID` 值、序列和三状态/缺项结果；REST新增参数/parentTrace与独立错误message/code断言确已在当前源码，不只复述Lead摘要。

## 4. Round 2：直接消费者、状态/事务/资源与旧层证据

实际追踪：

- `handler.go`公开capability/16方法、共享wire decoder、MapArticleError/WriteArticleError/JSON及创建写响应失败出口；`query_service.go`真实Admin非法状态/身份/typed view消费者。旧S8.2同版直接读集可复用；本轮未重审整个Article实现。
- `command_service.go`内容授权、OrphanArticlesCanonical/caller tx/计数与TransferOrphan的locate→身份双方→锁前像→两recipient快照→update→audit→真实proof.VerifyCompletion→cleanup→commit。`article_port.go`相关依赖定义；`identity/article_port.go`全文的公开函数adapter；真实identity reader的规范化/rows合同沿同版已读正文复用。
- `notice_dispatcher.go`constructor/Reconcile/context、claim/preflight/deliver/purpose及失败分类；`privacy/store.go`真实ProcessingGate的RequireOpen拒绝链。测试只提供SQL事实序列/记录调用，不复刻notice租约、retry或发送状态机；Mailer.Send是记录+取消替身，没有SMTP。
- `deletion_replay.go`公开构造/ReadStatus与refresh接缝、`replay_evidence.go`的Store/Source接口、`replay_status_v1.go`真实serializer、`recovery/replay_status_v1.go`严格decoder/canonical/digest及两份固定JSON fixture。共享canonical唯一；preset evidence/source callback不证明真实PG ReadCurrent/Publish/source验证。
- platform Trace/Authorizer/JSON/error、Session Principal、Policy/DTO/ETag必要定义复用已核同版范围；所有被审测试与这些必要输入均受178项指纹绑定。没有加载身份全设计或恢复未来方法。

Transfer脚本返回合法锁前像/notice/两recipient/audit/完成计数，真实Command及真实Repository SQL调用、真实opaque proof校验运行；proof/reference复制与外tx拒绝被核，不从fixture新造完成权威。快照失败、audit失败、proof mismatch与cleanup失败不返回半成功；两快照、audit、proof counts先于cleanup。caller事务与组件自持事务分清，commit/rollback/rows.Err/Close、notice操作context取消及Stop/Done按实际断言核，未声称真实PG锁/回滚或网络期数据库占用证明。

直接Handler的URL表只证明参数解析/方法消费，非第二router。对 `consumed-old-root-evidence.json` 限定的四份旧Root raw/meta独立核SHA、174源绑定与93/76、180/148集合事实，结合本角色原S8.2同版Review复用；未重新跑root。其resolver/terminal seam不证明真实Session SQL，Publish/Transfer等未逐项核到的旧root证据继续未验。S14/S15明确无新增组件leaf，不伪装成新测试通过；S4.2作者自审报告/已证明子集不变成新的非作者SQL/source/旧备份PASS。

真实审计INSERT失败和应用身份UPDATE/DELETE ACL两个候选top仍NOT IMPLEMENTED/NOT RUN，无placeholder/skip；签名、mock audit失败、成功INSERT后synthetic error、owner/superuser或trigger拒绝均不代替这些层。全局恢复503 envelope仅测公共组件，不证明生产隔离/零下游/MarkConsumed后开流。Case18两系统分项、S8.2及full S1.5原requires/fullgate不删，不套身份豁免。

## 5. 个人验证、实际集合及raw

全部个人命令在仓库根运行，child env按键过滤 `HAN_TEST_POSTGRES_DSN` 后才读取其余env值，不查询/打印DSN或secret，不运行PG/SMTP/service/Browser/network。仅个人Temp分析/overlay/日志及本报告写入；未更改生产、四测试、metadata或他人文件，未Git写、清理或委派。

| 个人命令 | exit | 实际结果 |
|---|---:|---|
| `go test -tags=contract ./test/contract/article -count=1 -json` | 0 | 4top、23groups、16 Endpoint分支、286run/pass、236leaf、0fail/skip |
| `go test -race -tags=contract ./test/contract/article -count=1 -json` | 0 | 相同286/236、0fail/skip，完整run/leaf集合与普通一致 |
| `go vet -tags=contract ./test/contract/article` | 0 | stdout/stderr空 |
| F001三个个人对照命令 | 0 / 0 / 1 | 上述最小对照表；不计入作者四top或当作当前源码失败 |

独立逐行解析JSON，不将包pass/父节点重算为leaf。实际top分布：

| top | run/pass | leaf | 风险组 |
|---|---:|---:|---:|
| TestArticleRESTContract | 182 | 149 | 6 |
| TestArticleInternalPortsContract | 62 | 53 | 8 |
| TestArticleProjectionContract | 15 | 11 | 5 |
| TestArticleReplayStatusProducerContract | 27 | 23 | 4 |

23组及全部宣称actual leaf与case_map逐项相同，16 Endpoint各success/error/missing Principal实际存在，三DELETE另有in-progress，共51 Endpoint leaf。S1–S16所宣称组件leaf均存在；空映射/旧证据/blocked与实际执行分开。286/236不是全S1–S16/全16生产root或full S1.5验收计数。

个人raw（完整meta含argv、exit、时刻及178before/after）：

| stdout JSONL | SHA256 |
|---|---|
| normal | `69D7F0D8AB796188C4262EF18317556FE65F2DED71D7DB812767D5C067FFE495` |
| race | `2BCEAF16EA4E2F75D092CFBC4CF793A90D47D1E54343F4B7A0A14ED44131FC75` |
| wire-mutant-suite | `6FE7C98CCC08EB0E5AC76F5C2369DB83627444EC3CCF784BF7920AE6693B9F60` |
| wire-control-witness | `C453C71BF8A08C48B94728974E157C34045CAE536FB96ED179C40BC0B15BA8B9` |
| wire-mutant-witness | `B9ABB0ADC50DA198C13378678DA728853ABFC123A6E10271C83EDB420C8538B4` |
| 全个人stderr、vet stdout（均0字节） | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |

`B\review-components-r01\raw-index.json` SHA=`B5E9F643502753AE4F5D06A1FAAD4EF8E72EF1B546AB62493AF0EAF93A4EAC03`；`run-set-verification.json` SHA=`52179A7C0EB541FA08782B3910606AB069E0A3513473DAB73D8D6ED8EA9C1556`；`wire-limit-mutant/proof.json` SHA=`582210666DF5B24C20738CA1935E2D98941594A01960CFB16CA480B15B672E9B`；actual readset/source binding SHA=`2A9613DC5F7D44DD516B70F03E567AA84AF0783E4AD88F5D6750CBD7D15591EC`。个人创建schema=0；无实际数据库/SMTP连接或后台服务。所有诊断材料保留，无清理。

## 6. 作者/Lead证据核证与限制

独立复算handoff所列15文件SHA、46历史/最终run共138个raw/meta hash，均match；完整178 snapshot逐项match。作者最终全pkg和精确四top普通/race四份JSON独立解析均286/236/0fail-skip且集合与个人相同；最后八个命令meta的before=after=当前178，旧174与Lead初始baseline相同。历史fixture/常量预期错误、编译和脚本失败按failure-chain保留，不计行为RED，也不混作当前产品失败。

两条作者语义负控：实际核完整156Go overlay指向与mutant源diff，其余输入/四assertion源码不变；m1只改model/policy让作者DTO泄漏version，normal/restricted两leaf在精确键断言编译后失败；m2只改真实Provider.ReadStatus让Ready返回Unavailable6，Ready20 leaf在真实producer输出键断言编译后失败。随后原源码全四top普通/race GREEN，两条有效负控证明相应风险，未证明F001的固定wire值。未机械重复其负控运行。

Lead Manifest回执实际SHA=`0DC59DEBEDC538D4853F95F513C268A4A68BC10E12A55BFA691531EE1B8BEAAD`，核source_before=after=当前178、format/modules/vet/build/contract/race/unit七项顺序exit0及format空输出。它是Lead执行，不冒称本角色执行/逐叶复核整Manifest；本角色没有重复全七项，也未对wire mutant运行全Manifest。

Lead新增durable导航 `.spec/wip/article-management/evidence-s1.5/implementation-r02/index.json` 实际SHA=`67BF9A2AA31D17F0C1B7E585DE95630C82887471F54907D7E398F2E5593ABFA8`，只作原技术包副本导航，原Role绝对路径保持；371副本/约10.9MB属Lead保管说明，未把数量当审查覆盖。verification-s1.5.md是动态过程receipt，不是本轮固定对象，未扩审其全文。Lead追加核证不代替上述个人判断和F001对照。

未验证范围：真实审计INSERT SQL失败/应用角色ACL、实际PG/source ReadCurrent/Publish、真实通知TLS/投递、全16生产root当前逐项新运行、真实SessionSQL、Journal真卷/旧备份/联合恢复、全局隔离及完整恢复开流、Host/部署E2E/容量与人的accepted。缺项按原责任/固定旧证据层或blocked保留；既有有效任务不重开，不把诚实未验项新造成本四文件缺陷。

## 7. 交回

本次仅 **AM-S15-COMP-R01-F001 / MEDIUM OPEN** 要求返工；没有新增方向决定或风格建议。Lead可恢复原Dalton的原四文件写窗口安排最小测试修复，固定新修订后交本Reviewer复审；本报告本身不授权源码写入、不代修或替责任人接受风险。

组件CodeReview为CHANGES REQUIRED；full S1.5接收/关闭本就继续受12requires、真实层及Case18阻塞约束，未因组件运行286/236或Manifest七项GREEN获得PASS。本报告不关闭任务、不代签人的接受。

**stopped：两轮独立审查完成，唯一仓库新产物为本报告；源码178末次零漂移，证据交Lead。**
