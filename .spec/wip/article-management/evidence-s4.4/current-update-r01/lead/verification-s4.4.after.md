# article-management S4.4 / Issue0001：Lead 核证与交接

2026-10-28。固定工作区 `D:\AID\VSCode\Codex\Han`，main，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。修复版已完成局部及真实PG子集回归、非作者CodeReview PASS及Lead技术核收。Issue精确两方接收仍pending，S4.4整卡未关闭；任务状态只取[原Tasks](../../../.add/project/knowledge/tasks/tasks.md)。

最新更新：原第二路径 `replay_evidence_test.go` 已新增 `CurrentUpdateRollback`，作者与非作者Curie分别完成三个场景的真实PG普通/race验证，两轮独立Review有界PASS、零finding，Lead已核收。本轮完整输入为185项，另外184项未变；当前仅原第三路径 `old_backup_restore_test.go` 尚未创建，S4.4整卡及完整真实恢复门禁保持。

181输入阶段更新（历史）：原第二路径 `replay_evidence_test.go` 的两个PG场景已交付并经独立评审和Lead核收，固定输入增至181项、原180项全部保留。具体来源与当时结果见文末“后续证据PG子集”。

## current UPDATE 回滚子集：2026-10-28 本轮核收

依据为现役design r22的Ready原子发布/回滚与test-plan r20 A-REC-07。Bohr（`01a180d5-28af-72e3-9a0a-d0c3855053dd`）只读确认无需新增设计、任务、路径或依赖；Euler（`01a180d5-f688-7930-ad40-dd54be13fa6f`）仅修改原第二测试文件，从BE6修订至SHA256 `CF1B07C58F5A7DDCDEA256484CFC9E5F1C0E595B0A1A13B3B1B49DDB410E9048`，685行。原两个leaf/helper主体逐字节保持，新增一leaf及同文件辅助实现。其余184输入、设计、迁移、生产代码、共享启动和身份模块均未变；完整185绑定见[本轮证据索引](evidence-s4.4/current-update-r01/index.json)。

新场景从无目标历史证据的真实Begin/Source候选开始。新schema的BEFORE current UPDATE trigger只有在同事务内查到已成功INSERT且绑定正确的evidence后才抛出真实SQLSTATE 23514，并返回低敏witness。它是测试故障，不是原生产CHECK缺陷或commit-unknown。失败后七表的行值/xmin/ctid与基线完整相同；owner/token/Input在内存完整比较，公开pending零写，独立NOWAIT可取得原run锁。保留trigger，以相同handle/input使用新ref重试成功，只有一份目标evidence，current/input/status绑定正确，双次Ready稳定；没有额外Begin、Fail或Replay。

作者与非作者Curie（`01a180dc-e21d-7b70-a4e5-b1a779546b8b`）各自执行完整 `TestArticleReplayEvidencePostgres`，普通/race每种精确4个run/pass节点、3个leaf、0fail/skip/extra；各自tagged vet exit0。作者的预检构建/预检/两测试/vet五个child，Curie的预检/两测试/vet四个child，均实际wait。两方各6个新schema、各92份payload完整保留，集合不交叉，旧两例的真实blocking/23505回归亦通过。原始argv/cwd/UTC/PID/exit/rawSHA、逐child185 live/snapshot前后图及留存资料均在本轮索引。敏感配置仅在获准PG child的内存/env使用，无凭据输出；没有额外DROP、disable、清理或角色权限变更。

[Curie正式两轮独立CodeReview](review-code-s4.4-current-update-r01.md)有界PASS、零finding，报告SHA256 `938C2EBE4907E9B1D85F529E3BF12CBF512277C8487A5FBC98919E3AA2027470`。Lead已实读报告与实际diff，独立核双方全部原始事件、185前后图、184份payload哈希、四次新leaf的witness/七表回滚/重试/calls/resource记录，并从8组Ready20/SQL/Facts10独立重算first16 canonical与operation/audit摘要。完整owner比较、trigger enabled、最终连接归还等固定代码断言实际通过，但不伪称存在单独持久token/observerPID事件快照。Lead另在当前185版本新执行Manifest七项，全部exit0、gofmt空、源码无漂移；默认race不替代上述integration race。

本次仅补齐A-REC-07的成功INSERT后current UPDATE失败子集，不构成完整A-REC-07、真实旧备份、真Unix FileJournal/HMAC/POSIX、双namespace/MarkConsumed/global opening、inventory签发挂载、应用ACL、RPO/RTO/retention或S8.2/Case18验收。三位Agent均已关闭；全部测试结束后，Lead只在原S4.4卡/B8A行回填局部证据，其他58卡逐字节保持，原三路径/六requires/fullclose/[-]/开始日期及人的精确接收、Issue状态不变。台账仍59总/47完成/3进行/9待办。此为技术核收，不替人的accepted/checkpoint或整体关闭；未暂存、提交、push、部署或清理。

## 固定依据与角色

| 对象 | 固定版本与结论 |
|---|---|
| [设计](../../../.add/project/knowledge/design-workspace/article-management/design.md) | r22，`8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E` |
| [独立设计Review](review-design-r22-replay-counters-r01.md) | Pauli，两轮有界PASS/0findings，`74BB3F80998A08EA917347DD68D042CB995BAF576CD3EA9B9203300A58FD99A1` |
| [测试规划](../../../.add/project/knowledge/design-workspace/article-management/test-plan.md) | r20 §4.8.1，`DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA` |
| [独立TestPlanReview](review-test-plan-r20-s4.4-r01.md) | Pauli，两轮有界PASS/0findings，`1F5FBF2FC190AA7429C79441FD88CEADC35F52C7AAC1E0166041B3BB7B6E4402` |
| [原问题单](../../issues/ISSUE-article-management-0001.md) | P1；真实故障/修复/接收关闭各自记录，不用规划PASS代运行 |

原Dalton（`01a17dcf-f2ed-7dc2-b7f3-6c72cf771448`）负责定位、首批旧API实测及后续九个既有文件；Hegel（`01a17f82-6dce-7531-967b-0ccadd3ec777`）在Dalton停写后接收唯一新增PG文件。两位作者均已停写，Hegel实际PG执行已结束；非作者Erdos（`01a17da1-6bd9-7d43-8642-b8b502127296`）已完成两轮独立CodeReview与本人PG验证并停写。Lead另外核实际范围、原始证据与项目检查；作者、Reviewer和Lead结果分别归属。

## 首批真实PG失败：已完成

仅新增 `test/integration/recovery/article_replay_test.go`，固定 `B71D378C91758BA4179E0C9103EC70D25CEDE2D9BF70BC0340D94C1EB2AD412B`，形成180输入；原179项逐项不变。gofmt/list exit0，唯一 `TestArticleReplayPostgres` 命中。两个普通命令实际exit1，共8个run/fail节点、3leaf、0skip，无编译、环境或fixture失败：

| leaf | 实际行为及独立SQL边界 |
|---|---|
| `CountsAndReadback/AllAbsent0N` | 真实Finalizer先完成ordinary/violating/orphan三条，Source实际0/3；预期Ready，实际unknown/evidence_stale。Article0、operation均completed、audit3、evidence0，run仍running |
| `CountsAndReadback/AllPendingN0/ExistingTrace` | 实际context_trace_mismatch；Article3/audit0/evidence0，首条claim独立提交、completion未提交，run failed |
| `CountsAndReadback/AllPendingN0/GeneratedTrace` | 实际invalid_trace_id；相同业务边界，非缺少未来API的编译失败 |

真实链为Finalizer/Repository/Audit/PGFactReader/FileInventory/PGStore/Provider与共享decoder；仅Journal使用公开受控seam。普通/违规/无归属三类及公开proof合法路径实际运行，不是stub Replayer。测试SQL使用与业务事务不同backend；全部三schema保留，锁可用、observer/池/进程收束。Lead核raw/meta、三份业务前后像、每个完整180snapshot/overlay和哈希，并独立检查operation→audit的ID/目标/动作/trace绑定；未重复连接制造另一轮PG结果。

完整argv/raw/meta、三schema与前后像、固定源码绑定见[66份old-API RED索引](evidence-s4.4/oldapi-red-r01/index.json)，SHA256 `5EC14AFB680F8AF2985B9B57E0A15BDB0A23047F7C714E14BE566D02AEC719B3`。它只证明旧实现三条行为失败，不是修复版或S4.4完整PASS。

## 修复版回归与独立评审

Dalton唯一九existing为 `internal/article/` 下四生产 `deletion_replay_source.go`、`replay_evidence.go`、`deletion_replay.go`、`replay_status_v1.go`，同目录四测试 `deletion_replay_source_test.go`、`deletion_replay_test.go`、`deletion_replay_integration_test.go`、`replay_status_v1_test.go`，外加 `test/contract/article/replay_status_producer_test.go`。Hegel唯一新增 `test/integration/recovery/article_replay_test.go`。其他170项受保护，包括三个其他contract、AuditSQL、共享authority、Finalizer/Repository/model、migration、identity/main。

双方停写后统一固定180输入，flat manifest SHA256 `18270024690941D7A20294FE78C56696FCFD8B19414B07823ED3E5E8368D2071`，含158项Go文件映射，实际编译选择取各命令/tag；9个既有文件与唯一新增文件均匹配作者最终回执。13命名风险场景及必需嵌套展开为16个实际执行leaf。

| 同版实际检查 | 结果及归属 |
|---|---|
| 相关单元普通/race | Dalton各67节点/59leaf，0fail/skip；Lead核原始JSON及全部源绑定 |
| 四contract普通/race | Dalton各286节点/236leaf，0fail/skip；Lead逐名对照原286节点，旧Case/断言完整保留 |
| tag vet/build/旧integration编译 | Dalton实际exit0；旧integration零运行只算编译，非旧六PG PASS |
| 新PG普通/race | Hegel各24节点/16leaf，0fail/skip；32个不同schema及所有数据保留 |
| Manifest七项 | Lead按原顺序全部exit0、gofmt空，每项前后180输入无漂移 |
| 非作者CodeReview/亲自PG | Erdos两轮有界PASS/0findings；本人新PG两mode各24节点16leaf、旧六PG两mode各7节点6leaf，0fail/skip，tagged vet通过 |

Lead已实际核新PG原始argv/exit/raw/source180、32case的366份副产物、58份Ready/SQL/Facts，并从SQL行像独立重算operation/audit摘要，一致。same-handle赢家与A→B→A两类场景实际新执行0/3而复用原持久2/1，原ref/digest/time保持；child握手与发布前无evidence/真实完成数据已核。ACK次序、NOWAIT屏障及exit23依固定代码已执行的断言与保留像，不冒称新增独立事件trace；child主动关闭资源后非零退出不是实际OS崩溃，丢ACK wrapper不是实际网络故障。401份运行/前像资料及180项durable源码绑定见[真实PG执行与Lead核证](evidence-s4.4/pg-execution-r01/index.json)，实际落盘SHA256 `91C326F7C0E50EA66E18A7854C671F747D481E909AB6021C40821E634EAFACA1`。

九existing的前后像/实际diff/原始结果见[修复核证索引](evidence-s4.4/repair-nine-r01/index.json)，实际SHA256 `6907F001C14F48F8F6D56A8D91AB4AA6F7BF33CB370DEA428D1BC68303369CB2`；新增PG代码固定 `B65FAE92A2AE95E291C8CB2CC136769E22209C4E888546CEDF43CC68727F659A`，见[独立新文件交接](evidence-s4.4/new-pg-code-r01/index.json)。

[Erdos正式独立CodeReview](review-code-s4.4-issue0001-r01.md) 实际SHA256 `68C412EC34D66272AA4F4783ABC8F6EB4105AB0990E838B2A9AE9736A7F49BAC`。本人运行cwd为完整固定180快照，七条子进程均真实wait完成；新PG32schema、旧PG14schema（每mode额外保留原top的unused pool）共46，均与作者32不同，非全库盘点。Lead已核21份raw/meta及每次live/snapshot180前后像、本人58Ready/SQL/Facts与366份副产物、完整harness-only差异；独立有界Gate据此技术核收。414份独立证据见[review-r01索引](evidence-s4.4/review-r01/index.json)，实际SHA256 `B6691BAAAA23113EC02A996394B5619B163ECE4371C6FE0A9CB244B6B4D859A4`。

旧六PG只在授权Temp retained/preflight harness overlay下运行，原六断言与原四迁移内容不改，唯一目录资源点改为retain；真实业务源码仍为同180。它们仍多为fixedSource/controlledReplayer，不能外推真实Finalizer/真Journal；旧inventory仅最终文件保留，没有另拍初始/漂移前独立字节档。原仓库harness仍有autoDROP，安全复现必须保留overlay，不可直接去掉该参数运行。

过程纠错单独保留：Hegel首执行器因多纳两README（182而非180）在Go/DSN/PG前exit2，随后仅Temp枚举修正并成功运行；不是产品RED。Lead修正自己证据副本的`.go.txt`后缀并保留原索引/全部字节，另纠正四份容器索引的写入前文本摘要为实际落盘摘要，未改变索引本体/原始日志/源码。格式检查重核exit0、输出空、180源未变；过程错误不与产品P1根因混淆。

Manifest14份日志、固定180/158映射及过程纠错记录见[Lead检查索引](evidence-s4.4/lead-checks-r01/index.json)，实际SHA256 `314393F2A6FBB8EE92F54A4D48C131C4F374F0D0F5B22A0DA433A158345C4609`；[P1相称复盘与保留安排](rca-issue0001-r01.md)仍不代人的结果接收或整卡关闭。

## 门禁与材料

S4.4原三路径、六requires、完整关闭条件保留。原修复交回时其余两测试路径尚未实施；本次第二路径已补齐，第三路径仍缺。Windows公开FileJournal不支持，未发现可用本机Unix载体；受控Journal的真实SQL链不证明真Journal/HMAC/POSIX、真实旧备份、双namespace/MarkConsumed或系统开流。RPO/RTO/retention、真实inventory签发挂载、应用ACL、S8.2/Case18及人的精确结果接收仍分别待证。ADR0024是原S4.3范围决定，不是S4.4豁免或真卷PASS。

准备与设计/G0见[prep](evidence-s4.4/prep-r01/index.json)、[初始组件诊断](evidence-s4.4/diagnostic-r01/index.json)、[设计/G0](evidence-s4.4/design-g0-r01/index.json)；本轮[规划前后像与Lead核对](evidence-s4.4/testplan-r20-r01/index.json)、[独立规划Review](evidence-s4.4/testplan-review-r20-r01/index.json)、[Tasks/B8A同步](evidence-s4.4/start-metadata-r01/index.json)均保留。没有提交、暂存、push、部署或Host清理；另一会话identity/root所有权不因本轮局部结果释放。

原计数/Trace修复阶段的最终两轮Tasks/B8A前后像、实际差异、RCA定稿及Lead字节核对共27份见[交回索引](evidence-s4.4/final-metadata-r01/index.json)，实际SHA256 `0E9E3CF98262BF8BBCE4E826CBF75666BEBD97C25C876B7D29C732DD7FF7E2E6`。当时Tasks为 `86284D1E14D527F8B46DEF400EF359620284AA38C33ACDE9D8D7606DD4F5D604`，Plan为 `E0C2C647460959E34EA8E7789E8BEF7BD943AFD4DBBA1EFD48A98FCF53A66021`；Lead实核其余58卡、35张identity卡、B8A以外计划、原三路径/六依赖/关闭条件及编码均保留，状态仍59总/44闭/5进行/10待做。RCA定稿SHA256 `A05DB5A797B676AD4E6864A94EC07141F57F70E3463EE0349EDE209E1463CAFE`；180输入与固定快照一致，原170项保护。该轮最终文档核对未增加运行实验，未代人的结果接收或整卡关闭。

## 后续证据PG子集

沿用户采用建议继续的授权，Gibbs只读测试设计与Ptolemy原卡核对确认：既有r22/r20的A-REC06/07可直接承接两项缺口，无新增需求、设计合同或任务。Hegel仅新增原第二路径 [replay_evidence_test.go](../../../test/integration/recovery/replay_evidence_test.go)，482行，SHA256 `BE6C689273783CFCB4939EE987397E09DB175B30506D9C0E49881FA8CE83A925`。唯一top `TestArticleReplayEvidencePostgres`，仅 `DifferentBackupFirstBinding`、`PublishRollback` 两leaf；原180项逐字节保护。固定[181输入清单](evidence-s4.4/evidence-followup-prep-r01/lead/source181.flat.json) SHA256 `D8EB37B52BC50CFC78C2E6F6014DAA790B9B10C1711C29D537756BAFEE7B745A`；159项为Go子集，非全部输入。真实Go cwd为保留的完整181快照，运行期迁移/fixture亦来自该快照。

作者普通/race各3个run/pass节点、2leaf、0fail/skip，tagged vet exit0，3个child实际wait；四个schema全部保留。[Erdos两轮独立CodeReview](review-code-s4.4-evidence-r01.md) 有界PASS、0finding，最终报告SHA256 `15C4EE2344DFDC6CF407D51934A9CC91C6343C7C1D1B2A4B097945D66F8B45CB`。Reviewer本人普通/race同为各3/2、0fail/skip，独立只读preflight和tagged vet亦exit0，4个child实际wait；本人四schema与作者四个互不重叠。没有重跑原16/旧6专项或将旧结果算作本次命令。

Lead已实际读新增全文及修订差异、完整独立报告，核作者与Reviewer的全部raw/meta、各次181 live/snapshot前后图、各58份保留payload和6份Ready/SQL/Facts，并分别从SQL像独立重算operation/audit摘要。不同backup的真实阻塞边、loser绑定拒绝与零副作用均核；四次实际等待方XID均为null，不虚构两个已分配XID。两个真实backend、两端T状态及锁边共同证明争用；T状态、owner等值和无额外Begin/Replay由固定代码已通过的断言支撑，不声称另有独立token事件日志。23505前后七表完整像（含xmin/ctid）一致、独立NOWAIT可锁、foreign与业务记录保留、同owner新ID重试及稳定Ready均核。Lead另外按Manifest原顺序完成七项检查，全部exit0、gofmt空、181前后无漂移；默认全包测试不代integration场景。

两项实测范围：合成A/B inventory经过真实reader/Source，两独立Store.Begin争用只保留首绑；foreign run合法预留ref使真实evidence INSERT返回 `23505/deletion_replay_evidence_pkey`，失败无部分发布且同handle/input新ID可重试。仅证明INSERT自身失败子集；INSERT成功后current UPDATE失败仍未验证，不计完整A-REC07或真实旧备份恢复。

持久原件分别为：[测试设计/CODE/静态/181绑定/Manifest](evidence-s4.4/evidence-followup-prep-r01/index.json)，51份、SHA256 `2AB5AF916504D0F0848E5AA803E2453817808588CE379E1DA7027974E367B683`；[作者PG与Lead核证](evidence-s4.4/evidence-followup-pg-r01/index.json)，85份/4case、SHA256 `AAB90277426341B5ADA4BB8E2E32C8A8983548BB6742026E0AFF00C432F262F7`；[独立Review与本人PG/Lead核证](evidence-s4.4/evidence-followup-review-r01/index.json)，88份/4schema、SHA256 `767D86D0B6B8883C9FFD9CFA8CEE445DC0119D940509BB2378B615ADEC50B7FF`。报告仅澄清S4.4六依赖与S1.5十二依赖，技术结论不变；前版145C字节像经原实际hash核同后保留，未改原始运行证据。G0早期必须分配XID的静态设想与最终BE6可选XID断言亦分开保留。

S4.4仍为[-]，原三路径、六requires和完整关闭条件不变；唯一未建原路径仍为 `old_backup_restore_test.go`。受控Journal、合成backup引用/foreign fixture及真实PG不能证明真Unix FileJournal/HMAC/POSIX、真实旧备份、双namespace/MarkConsumed/global opening、inventory真实签发挂载、RPO/RTO/retention、应用ACL或S8.2/Case18，亦不代人的精确结果接收或Issue关闭。用户已明确另一会话结束；本片未修改共享启动、身份或恢复架构代码。未暂存、提交、部署、清理或移动任何材料。

本轮Tasks/B8A最终前后像、差异与Lead核对共17份见[台账收口索引](evidence-s4.4/evidence-followup-final-r01/index.json)，实际SHA256 `C3F9BB451E8781525B3004BF227F8564373230B65CBA741F07D90ADE4DF9C0B5`。Tasks为 `1560602DE69EC52AA6AF087B45417FAC58BC8679ACC2C2A2EB0EB807BC8E126F`，Plan为 `CBCC228146E5D260E5D47F70B0FD5B14E7E1B99DD4F7CFF6F26292A7C3BACB41`。Lead已核实际差异仅原S4.4卡/原B8A行，其余58卡、35identity卡、S1.5/B10、原文件/依赖/关闭条件/开始日期及编码均保留；仍59总/44闭/5进行/10待做。181 live/snapshot一致，分支/HEAD固定、暂存为空、tracked metadata diff-check exit0；此收口未增加运行实验。
