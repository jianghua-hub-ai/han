# ISSUE-article-management-0001：DesignReview r22 R01

2026-10-28。Pauli / 实际非作者 add-reviewer；原设计作者 Hubble 已 completed/stopped。objectType=`design`。固定对象、采用的发布时绑定方案及窗口来自 **Lead 交接**；用户一般“建议值继续”授权不等于逐项设计 accepted。沿原交付上下文和本 Issue 的必要设计修订接力，不重新 Route、不替作者修复。

**有界 DesignReview Gate：PASS，仅覆盖固定 r21→r22 的恢复计数合同；两轮独立检查完成，无新增设计 findings。** 现有 Source→Finalizer trace 接缝存在静态确认的实施不兼容，详见第4节，须由后续 G0/代码与真实链验证处理，不能由本设计PASS豁免。计数缺陷尚未代码修复或运行回归，本报告不关闭 Issue、不授代码/PG、不代签人的确认。

## 1. 固定对象与实际范围

实际核工作区 `D:\AID\VSCode\Codex\Han`、分支 `main`、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。正式开始前报告路径不存在；唯一仓库写权限为本报告。

| 输入 | 实际 SHA256 |
|---|---|
| 当前 design.md r22 / 作者 candidate | `8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E` |
| design.r21.before.md | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA` |
| design.r21-to-r22.actual.diff | `3735F448C99DE29724347CB95BBDAFA1B88E590F8BCC2324A1E6C195F2228A62` |
| stories r10（hash-only，未变） | `492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9` |
| test-plan r19（hash-only，既有合同复用） | `88296F772A19600A237CCA9F47A16D4B673828EF41E5C1D0C2D88924BE2C6F66` |
| Issue 当前事实/边界 | `7642C37243842C5C924454E295B00AC987CA59B6EF31F82BA33DD85F85172C3B` |
| Hubble consult（全文） | `E94FD56824D9C7854481946363B7A4A89407F598A7B8ED9B8E4D631FEB5D9CAF` |
| 诊断 index.json | `FE693C3DD308408DD78D43FCBA56654903FE46C19F3B9BE23DE9D72468E52999` |

作者固定材料目录：`C:\Users\J00010385\AppData\Local\Temp\han-replay-design-r22-8515ba6ac19f438080f00f790e578d93`。实际读取完整 actual diff、actual-readset、stopped、static-checks及哈希材料；作者静态自检不是本独立结论。独立从r21前像和当前字节重建diff，与actual diff逐字节相同，candidate与当前相同，实际 **+115/-61**。

## 2. Round 1：全部增量、归属与保护区块

实际检查元数据/来源/确认措辞、共享 Source/Facts/Input/内部port、持久字段说明、§3.4.7全部新状态/发布/刷新/崩溃合同、§6.1验证落点、§6.5兼容及§8.7影响面。设计只定义 `DeletionReplayFacts` 十项当前事实，以及嵌入Facts后另带两个真实执行分项的Input；没有新表/列、Fact摘要、布尔proof或第二canonical authority。

独立保留性检查为真：

- §3.3.2起全部Story主体（含Admin2）及§3.4.1～6字节不变；§8.6 Admin2影响/权限/关闭门禁字节不变。
- 恢复顺序至原FMEA/安全正文不变；REST endpoint表行、全部JSON代码块、Ready20/Unavailable6示例、全部PlantUML均相同。
- 需求分解/引用/附录区块相同；历史 checkpoint标记起至文末字节相同。r22只更新当前draft确认责任，不伪造签名、accepted或新checkpoint。
- 已核fixture仍为Ready20、total/replayed/absent=`3/2/1`及Unavailable6；fixture文件指纹与诊断source179相同。

生产候选精确四文件为 `internal/article/deletion_replay_source.go`、`replay_evidence.go`、`deletion_replay.go`、`replay_status_v1.go`。既有五测试包括Source/Provider/PG/serializer与contract producer回调/mocks；S4.4原唯一新增候选 `test/integration/recovery/article_replay_test.go / TestArticleReplayPostgres`，当前核查尚不存在。候选写集不等于授权；不扩Finalizer/Repository/model/schema、recovery authority、identity或root，不重开历史S4.2。

## 3. Round 2：直接消费者、状态、安全与可实施性

| 风险/合同 | 独立检查与结论 |
|---|---|
| Facts与物理计数 | Source当前ReplayOldBackup携真实Replayer结果，Snapshot却构造total/0；Finalizer按Deleted/AlreadyAbsent分别累计，completed operation直接计absent。新合同正确把Snapshot限定十项可重读事实，ReplayOldBackup才返回完整成功执行分项；不清零、不倒推首次删除结果、不跨失败轮累加。十项Facts加version/namespace/run/backup与两个执行分项仍正好构成原first16。 |
| 首次耐久绑定 | 无已提交evidence时，没有已承诺物理分项；删除后/Ready前崩溃可以完整重跑并诚实产生0/N。首次合法完整结果经freshFacts十项全等及完整canonical验证，再在同Ready短事务INSERT/reuse evidence与安装current/input_digest/status。原schema已有全部字段、run行锁及唯一/FK约束，可承担该绑定，无需新回执authority。 |
| 同handle胜出优先 | Publish先核backup/exact generation+token及同handle合法Ready，先用胜出行的持久分项组合freshFacts核完整摘要；不能先拿落后0/N对胜出2/1比较而误判stale，也不能改写胜出分项/ref/time。只有尚未Ready的running/failed exact handle才进入候选/历史选择；旧owner始终零写。 |
| 历史唯一复用 | 仅在真实Replay后的发布阶段，按同run/backup/namespace/version和十项Facts筛选现有immutable行，逐行核完整绑定/时间/first16；可用原run索引，不需新schema/索引。唯一合法匹配复用旧分项/ref/digest/time，无匹配才新建；多outcome/digest、坏匹配行或唯一性不明fail closed，不能任选、跳过坏行或改历史。当前执行0/3与选择旧evidence2/1明确分开，不改报本轮物理结果。 |
| current独立校验 | ReadCurrent只校验当前指针行，先核原字段/版本/完整digest/UTC微秒，再用其持久分项+freshFacts重组完整first16同时匹配evidence/run摘要；不搜索历史替坏current，不Replay、不修行、不写库。对Journal/inventory/operation/audit摘要及事实计数真漂移仍当次stale，不用总数等式替代完整校验。 |
| owner/CAS | Begin接Facts而非伪完整digest，Ready分支仍须先核current完整合法及freshFacts；candidate相同只负责确定性恢复owner，不豁免Ready校验。首绑、exact expected、generation/token及running join规则保留；LoadAttempt/公共DTO均不授写权。Publish允许同generation failed→Ready但不复活旧owner，Fail永不覆盖Ready或新generation。 |
| 提交未知 | Begin/Publish/Fail未知先ReadCurrent；合法已提交Ready直接返回数据库同一计数/ref/digest/time，零Fail/ReplayAll。无合法Ready才依LoadAttempt观察、同candidate重试Begin取得authoritative handle；观测值不代授权。存活真实候选在Facts未漂移且重新取得权限后可重试Publish；候选不在或不再适用才完整重放，无法判定则Unavailable。实际PGStore.Publish提交后丢ack必须真实覆盖，预置Ready不能顶替。 |
| legacy/不可变性 | generation0、合法2/1的current evidence按完整canonical及freshFacts零回写复用；真drift后才按旧规则持锁升级，旧evidence不改。0003已允许无首Ready时input_digest=NULL；0009的gen0/tokenNULL及gen>0/token非空约束保持。无新migration，切换必须停旧replay writer，禁止混跑。摘要合法不被包装为历史物理来源已审计。 |
| 事务外IO | Source/ReplayAll/文件与PG Facts刷新均在run事务/锁外；Publish刷新后短事务重读权威owner/status/current，变化则丢弃旧组合。ReadCurrent刷新后复核backup/generation/token/status/input_digest/current_ref，不一致返回stale而非旧Ready。只承诺所观察事实与run的一致，不承诺跨文件/Journal/PG瞬时原子；维护隔离和完整恢复前不开流仍由上游承担，不能证明时系统恢复仍不签完成。 |
| 直接port与消费者 | 新Source.Snapshot、Begin参数与Refresh/ReadCurrent回调可在现四生产文件内适配。现Provider已有LoadAttempt/Begin/Publish/ReadCurrent，Store已有行锁/scan/完整canonical/helpers；历史匹配SQL可内承原Store。五个既有测试中Source/Store stub、memory helper、integrationSource及producer callback均有真实签名依赖，需同步，不能只改生产声称可编译。无须改公共构造或root接线。 |
| public/资源/安全 | 仍由recovery包提供版本/namespace/类型/CanonicalJSON/Digest/严格decoder；tokens/generation/current不进DTO/日志。全部参与摘要时间先UTC微秒规范化；坏绑定/版本/计数、不可读source/重读失败都fail closed，无缓存/Source.lastResult/调用方计数fallback。不新增生产重试数字或无界循环，不借修复新增Guard或开放流量。 |

新验收落点覆盖N/0、0/N、mixed2/1、重建Provider后读取零Replay、发布前崩溃、同handle胜出、事实drift后重发布/回归历史、legacy与歧义、真实提交后丢ack以及事务外刷新barrier。它们是后续规划/实现/运行义务，不是本轮执行结果；没有因未实施或未运行而判设计缺陷。

## 4. Source→Finalizer trace：必须交回的实施限制

Lead在评审中补充线索后，本角色精确静态核到：`deletion_replay_source.go:115` 固定传 `"article-deletion-replay"`；`deletion_finalizer.go:511` 的deletionCompletionContext在已有context trace时要求显式参数相同，否则调用平台校验；`platform/logging.go:15/73` 要求26位限定字符集。该字面量不合法，已有合法trace也会与它不一致。真正进入completion的未完成删除会受此接缝阻断；completed operation的absent短路和controlledReplayer测试可能看不到它。

这是**现有实现的静态不兼容及后续真实链验收限制**，不是本轮已执行的失败或新PG复现。r22没有要求保留该错误字面量，Source.go处于已采用四生产候选范围；沿既有Tracing规范对齐context/显式trace即可保持构造、公共协议与其他模块边界。因此未列为要求Hubble修改r22的设计finding，也不把它降为可以忽略的风格建议或本报告已修复事项。

交原Dalton G0/实施核实际合法透传或缺失时的合法新根关联，保留未来真实Source→Finalizer首批RED及修复回归；**禁止测试adapter改写context/trace隐藏它、放宽平台校验或把真实Finalizer换回受控Replayer**。本Reviewer不替作者实施或另定追踪方案；若实际需要扩大文件/公共契约或额外authority，停受影响部分交Lead。本设计PASS不能冒充该接缝或真实S4.4链已可运行通过。

## 5. 诊断证据、实际Readset与指纹

此前PREP已全文读Issue/consult及diagnosis，独立解析三次原始JSON/meta：每次3 run节点/2 leaf/0skip，control1/0为PASS、absent0/1与父节点FAIL，exit1；三组raw/meta/stderr SHA均对应固定raw-meta-index，诊断overlay保留原source-test完整字节前缀。三套absent canonical对照只差两个执行字段，三套control完全相同；原Provider+既有memoryStore在absent链输出unknown/evidence_stale、evidence0，随后pending。证据证明受控Journal/Replayer/facts下的真实Source/FileReader/Provider组件矛盾，**不证明真实PG/Finalizer物理删除/文件Journal/旧备份运行**；1/0 control也不冒称完整真实删除历史。

原S4.2 CodeR02是历史授权作者自审，本次未将其变成非作者PASS或新回归；不重开/抹除旧卡和失败链。PG Store的完整摘要比较只静态核，后续必须实测，Issue仍待修复/回归/独立代码复审/实际接收。

| 实际读集 | 内容与限制 |
|---|---|
| Role/方法/规则 | 同版完整Reviewer Role、add-code-review/design、当前delivery及checklist/abstraction/design-writing/decisions/安全红线/go-service边界复用；PREP完整读取本Issue适用add-issue-handling。本次trace线索补读tracing规范。未加载其他Role/未来实施/E2E方法或reroute。 |
| r22及作者材料 | 固定前像/当前/candidate原字节、完整actual diff、actual-readset/stopped/static/hash材料；定向新§3.3.1/§3.4.7/§6.1/兼容/影响/确认。保护区块独立比较，不声称重审全design。 |
| Issue/consult/诊断 | Issue与consult全文；diagnosis、Lead核证、observations、raw-meta-index及三轮raw/meta；必要canonical/public输出对照，未运行诊断脚本。正式阶段复用PREP已核同版结果并核索引/来源指纹。 |
| 四生产候选 | Source ReplayOldBackup/Snapshot/Facts生成/构造；Provider编排/Begin重试/发布/读取/错误分类；Store接口/Begin/Fail/Publish/ReadCurrent/完整evidence scan/唯一与current SQL；serializer/canonical映射。现役源码未修复，不能反推新设计已经实施。 |
| 五既有测试 | 必要Source测试符号、Provider Source/memoryStore helper、integrationSource/PG叶范围、serializer fixture测试、contract producer Store/Source callback/mocks；定向符号搜索，不作全测试代码Review。 |
| Finalizer/共享authority/schema | Finalizer ReplayAll/Completed短路/实际分项/complete context；recovery完整canonical计数不变量和编码；0003 replay run/evidence/唯一/FK/append-only及0009 gen/token约束。无需扩身份/root正文。 |
| hash-only保护 | stories r10、test-plan r19、model.go、FileBackupInventoryReader、main/main_test、两JSON fixture。其余必要既有合同沿前轮复用，不把hash固定说成全文审查。 |

正式指纹记录含31项仓内/外部输入：定向20项source179输入无漂移，未重读/重审全部179项。当前root仅hash为 main.go=`6B45B5509A2D01C9F5DF0393CA00CAD5C8430353B5F849BF6E71BA3A47C52AC4`、main_test.go=`B53DD34D57AD6A2797E9FC080FD90034FC52CB26675686CB075D3153F193A9BC`；这不证明共享root窗口释放。Ready fixture=`586A0B8F76ADE8B720E44094A0D13163C827D231AC9CBC786C6D4AF92E90F0F2`，Unavailable fixture=`929193CD06BA566402F3F3C1D6321E57BDA95D0270DED666B5C951D97B21AC1B`，均不变。

个人静态目录：`C:\Users\J00010385\AppData\Local\Temp\han-pauli-design-r22-review-8743193a6dd44f9397f5ff5116d53006`。`readset-start.json` SHA=`69FC6E64DF3B6D6905E36852C0E88E25F89DF5FBCB1BBA004B1CF266553854DA`；`independent-static.json` SHA=`22DADA4CC061F90BC1845FACED90740CB55C48C4E6828929E44C1D74767B43C7`；`end-check.json` SHA=`4004FA32B649A8CC544DC2D2AF873251027E4F97C31CACC008376A3D1B9ED006`。PREP原静态读集/原诊断原件均保留。末核固定对象与必要输入无未知漂移。

## 6. Findings、Gate、未验与停止

新增**设计**findings：无，CRITICAL/HIGH/MEDIUM/LOW均0，不制造finding ID。无本轮待Hubble决定的新方案；第4节是必须交代码G0/真实链验证的实施限制，不是新设计豁免。若正式修订发现真实设计缺陷，仍回原Hubble最小修复，再由原Reviewer核新固定修订。

| 范围 | 结论 |
|---|---|
| 固定r22计数/Facts/evidence/history/current/owner/CAS/未知提交/IO边界 | 两轮独立有界DesignReview PASS |
| 生产修复、五测试适配及S4.4新增真链 | 未实施/未由本报告验证；第4节trace接缝须处理，不能由adapter隐藏；写窗/测试/LOCALPG另核 |
| ISSUE-article-management-0001 | 未修复、未回归、未验收关闭；新设计不关闭历史或当前Issue |
| Ready20/first16/fixture/authority/Admin2/恢复顺序/旧checkpoint | 原保护合同与独立字节/指纹检查保持，不新增权限/authority/schema |
| 真实Journal/keyring/旧备份/POSIX/双namespace/MarkConsumed/全局开流 | 原未验/上游义务不变；受控Journal+真实PG子集也不能替代完整恢复 |
| S1.5/S4.4、S8.2/Case18、ACL与人的accepted/checkpoint | 原完整门禁继续，不由本设计PASS解除或代签 |

实际动作仅为只读文件/rg/Git分支HEAD、哈希与个人Python字节diff/JSON证据解析，以及唯一报告写入。**无Go测试/list/vet/build、PG/DSN/env探针、SMTP/服务/Browser、代码或Design/Story/TestPlan/Tasks/Plan/ADR/identity/root修改、Git mutation、cleanup/删除移动或子委派。** 未启动业务/测试/数据库/持续后台进程；本轮只读工具命令均结束，无本角色遗留进程，未作全机进程普查。Hubble停写依据Lead交接及实际stopped.json，不由静态hash推断其他writer已停止。

**stopped：固定r22两轮独立设计评审完成，报告交Lead；不授实施/PG，不关闭Issue或代签人的成果接受。**
