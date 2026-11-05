# ISSUE-article-management-0001 / S4.4：TestPlanReview r20 R01

2026-10-28。Pauli / 实际非作者 add-reviewer；作者 Ohm 已 fixed/stopped。objectType=`design`（测试规划合同）。固定稿、采用范围、运行限制及补充澄清来自 **Lead交接**；本角色实际独立检查另列。沿原交付及Issue必要修复接力，不重新Route、不替作者修计划。

**有界 TestPlanReview Gate：PASS。** 已完成两轮独立检查，固定r19→r20增量无新增findings。r22风险已映射到真实链/PG与必要单元、契约回归；旧API阶段可先取得针对计数/Trace的行为RED，后续候选边界为四生产、五既有测试及原S4.4一个新增文件。此结论只是规划PASS，不证明已编译、Go/PG通过、Issue修复或任务关闭，不发CODE/LOCALPG/overlay信号。

## 1. 固定对象与来源

实际核工作区 `D:\AID\VSCode\Codex\Han`、`main`、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。正式开始前报告路径不存在，唯一仓库写权限为本报告；PREP未读取动态r20或写最终报告。

| 固定输入 | 实际 SHA256 |
|---|---|
| 当前 test-plan.md r20 / fixed副本 | `DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA` |
| r19 before-image | `88296F772A19600A237CCA9F47A16D4B673828EF41E5C1D0C2D88924BE2C6F66` |
| r19-to-r20.actual.diff | `8E2DE6C637B4A89721DE13A72C517908745C763EBCD5577A887C3EB4CFF28A65` |
| design r22 | `8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E` |
| Pauli r22 DesignReview | `74BB3F80998A08EA917347DD68D042CB995BAF576CD3EA9B9203300A58FD99A1` |
| durable planning index（hash-only） | `DDE625915ECA2A3EC1447C8639B6F275AC0661C7B9BDEF8C4D1D74206D4CAA42` |

作者材料目录：`C:\Users\J00010385\AppData\Local\Temp\han-issue0001-s44-testplan-r20-d4f71d8b083246d7bbde1daa7e24d99e`。实际读取完整actual diff、handoff、actual-readset、coverage；读取before/fixed/current字节独立重建unified diff，与交回文件逐字节相同，fixed与当前相同。Lead额外机械核对回执实际读取，归属Lead而非本Reviewer；本结论不依赖其摘要代替差异审查。

## 2. Round 1：完整增量及原层保留

实际 **+98/-11**，仅metadata/设计来源与历史措辞、新§4.8.1（当前632–706行）及§12结论。原A-REC-01～08原文保留，由§4.8.1明确澄清受r22影响的现役稳定输入、计数与提交未知期望；没有删减公共协议或以旧层PASS关闭新风险。

独立检查结果：

- 197条原表格行（按 `| A-` 前缀取原Case/映射行，非197个唯一测试）内容/顺序全等。完整Admin2 Case区块、20 Case/38映射、Case18分项以及§9.6九路径/四top/命令区块字节全等。
- §4.7.2审计SQL规划区块字节全等；三份非producer contract与AuditSQL现役文件指纹相同，未削弱既有S1.5义务。producer仅为内部port/mocks联动候选，原四top/23组/16Handler/236leaf基线断言保留，新增语义测试自然增数须另解析。
- 原S1.5十二requires含未闭S8.2保持；原S4.4六requires精确为 `project-bootstrap-B0.1`、`identity-auth-S4.2`、`identity-auth-S4.3`、`article-management-S4.2`、`article-management-S4.3`、`article-management-S4.5`。不新增业务requires或竞争任务。
- 原S4.4三路径仍为 `article_replay_test.go`、`replay_evidence_test.go`、`old_backup_restore_test.go`（均在 `test/integration/recovery/`）；本片只新增第一个，其余未完成责任/fullclose保留，不为凑路径创建空文件。当前三文件核查均不存在，执行状态仍为准备/未授权。
- r22设计层PASS、三次既有组件诊断、历史S4.2作者自审及已交付S1.5局部层分别归属，不冒称本次PG/修复通过。未知完整恢复环境/人责保留为后续范围，未当作本局部规划新增finding。

## 3. Round 2：真实链、可实施用例与证据强度

真实候选链为 FileBackupInventoryReader→ProductionDeletionReplaySource→真实Finalizer/Repository/Audit/PG Facts→PostgresReplayEvidenceStore→Provider→共享strict decoder；仅Journal使用公开受控接口。新文件内承harness/适配/屏障/child helper，不能跨包import article测试私有helper，也不能复制DTO、canonical、内存Store状态机或完成proof权威。

| 风险/场景 | 实际检查及规划足够性 |
|---|---|
| `CountsAndReadback` 四场景 | AllPendingN0的ordinary/violating/orphan合法前像及ExistingTrace/GeneratedTrace、AllAbsent0N真实预完成、Mixed2And1、RepeatAndRebuildZeroReplay分别核物理分项与独立PG行。所有Ready均对完整first16及current/evidence，重复/重建核零Replay/Begin/ref/clock和原计数/ref/digest/time，不用Source缓存或总数等式冒充真实结果。 |
| 旧API计数RED | 首批仅新增PG文件，不提前引用新Facts/Begin签名。可先以真实Finalizer+合法Trace预完成，隔离Source的0/N与Snapshot旧total/0矛盾；公开ReplayAndVerify期望Ready而观察真实结果。不会因Future API尚不存在故意编译失败；未选场景/初始阶段不冒称整个top通过。缺环境/fixture/编译错误须单列，实际行为不同先定位。 |
| 旧API Trace RED | all-pending分别带合法parent Trace与无Trace，真实Source→Finalizer；固定23位字面量与现役26位/context一致性不兼容只是静态证据，计划要求另保留真实错误/Provider variant及claim、Article/audit/evidence边界。禁止adapter重写ctx/参数、stub Replayer或放宽平台校验；Source.go内按既有机制对齐合法Trace，保留parent deadline/cancel，不扩公共构造或root。 |
| `InterruptedReplayAndPublish` 三场景 | 部分真实完成后明确失败，不发布部分计数；真实子进程ReplayAll后/Ready前握手退出，再从PG持久事实诚实重跑；实际PGStore.Publish提交及独立连接可见之后才丢ACK，要求先ReadCurrent且不先Fail/Replay。child分流内承唯一top，父核实际exit/数据边界并join，不以helper退出码计业务PASS；不是预置Ready、纯内存崩溃模拟或真实网络断链。 |
| `AttemptAndFreshFacts` 三场景 | 同public Begin/join返回的handle，barrier控制2/1胜出、0/3落后Publish，核先复用赢家。Operation/Audit分别独立fixture合法改变实际FactReader读取的行；ReadStatus stale后真实Replay/Publish重新Ready。Refresh的Publish/ReadCurrent子例以真实回调屏障、另一backend NOWAIT/run前后像及池/事务资源观测核事务外刷新，期间通过合法公开Store改变owner/current，旧组合须零写/丢弃；不以sleep、标签或“外部原子”口号替代。 |
| `HistoryAndLegacy` 三场景 | Facts A→B→A用真实生产reader可接受的inventory bytes往返，不倒改PG Audit/completed事实；重放0/3与选择历史2/1分开核，确保原ref/time及无第三份语义evidence。gen0/tokenNULL/2/1明确是合成兼容前像，核公开复用零回写；歧义历史合法INSERT、current先指其他Facts，强制实际发布搜索后拒绝，不能由current早返回假称覆盖歧义。 |
| 三类/proof可实施性 | 现有Finalizer/Repository/公开identity生命周期和proof.Reference/VerifyCompletion有可复用形状；已读现役AuditSQL的真实adapter片段，可在新文件映射并委托同tx真实proof，不必改identity/root或自造proof。pending先有绑定operation，Finalizer claim独立commit与completion失败回滚/lease必须分开核。静态可行不等于三类已经运行通过。 |
| 合法事实drift | 已精确读PostgresReplayFactReader：operation查询全体id/status/version/completed_at，Audit只取article目标和四种删除动作。可合法新增被选中的操作/审计事实并独立核变化，不需要删append-only Audit或修改completed不可变字段，也不能用未被reader读取的Audit行假装drift。 |
| 足够分层 | Facts10逐项变动、完整绑定/版本/计数/时间/坏current等组合落于原Source/ReplayEvidence/ReplayStatus及Provider单元；PG只承担真实SQL/物理结果/并发、历史选择和提交后故障关键场景；producer签名适配后完整四contract普通/race保留golden/Ready20/Unavailable6。未将所有组合重复堆到PG或把签名/编译当行为。 |

四组共13个**必需命名风险场景**与coverage文件完全对应；AllPending、OperationAuditDrift、Refresh各另有两个必需嵌套子例。最终必须解析真实展开run/leaf及fail/skip集合，不能把“13”当已执行的底层leaf数或遗漏嵌套断言。当前新增测试全部NOT RUN。

原API兼容阶段与之后九既有文件的同步适配可在原候选4+5+1内分阶段完成；旧API阶段不能先用新的typed port制造编译失败。新recovery目录现役文件确为integration tag、package `recovery_test`，public constructors/ports可直接调用。旧child测试已有唯一top显式分流的参考形状，仅参考、不跨包借私有helper，不新增默认Skip/helper top或隐藏部署配置。

## 4. 环境、retention、旧六SQL leaf与执行边界

| 边界 | 本次判断 |
|---|---|
| 新PG环境 | 批准User DSN仅在未来执行窗口安全注入；先解析主地址和全部fallback为literal loopback:5433，再核实际endpoint/version180006/身份，均在CREATE前。Lead已有只读连接时点事实不作本Reviewer探针或本专项PASS。 |
| 新证据/PG schema | 单文件retained harness：每例随机schema独立提交后立即log，再另tx全0001～0009迁移，每连接search_path一致；成功/失败/RED schema及新任务manifest/source snapshot/raw保留。无DROP/RemoveAll、GRANT/新role/认证变更或额外secret读取。输出可自动MkdirTemp并log，不另造隐藏必填env。 |
| 资源与child | 屏障所有分支可取消/release；rows/tx/context/连接/pool acquired归零与Close，goroutine/child/输出读取join及后台连接释放均需实证。释放运行资源与清理保留数据分开，child崩溃不冒充完整Host/旧备份恢复。 |
| 旧六PG leaf | 静态读原newReplayIntegrationPool确认会DROP schema且仅四迁移，故不能直接跑。`-tags=integration ./internal/article -run '^$'`只作tag编译，零执行是其预期，绝不记旧PG回归PASS。若需旧六例实跑，仅按Lead另授的精确retention-only Temp overlay保留原断言及源/overlay/hash；无授权即NOT RUN，不能借历史PASS。 |
| 旧层消费 | 旧库存forward drift使用真实reader/source/PG Store但受控Replayer；旧Begin“commit unknown”名称本身不证明真实丢ACK。新Operation/Audit、事实历史往返及真实Publish提交后ACK补的是差额，不重签旧六例为当前真实Finalizer/计数链验证。 |
| unit t.TempDir澄清 | 按Lead本轮明确澄清，**既有普通unit fixture的t.TempDir生命周期不等同Host删除项目材料**，不据其存在新增全量harness改写/审批/overlay或机械阻断局部回归。本任务新证据及PGschema必须retain，旧PG的自动DROP仍禁止直跑；确需保留旧PG临时材料的retention-only overlay另授。未把这项澄清扩大为项目材料cleanup授权。 |
| 命中/负控 | 精确top/子例普通及race、tagged vet和必要旧tag编译均需按真实argv/exit/raw/版本取证；无DSN/缺leaf/skip/零命中/无断言返回不能PASS，编译/环境失败不算产品RED。既有三次组件诊断不是新PG FAIL；必要额外语义overlay也须精确授权、完整snapshot及可编译真实断言FAIL。 |

Windows公开FileJournal不支持、真Journal/keyring/POSIX/旧备份/inventory签发挂载、RPO/RTO/retention与双namespace/MarkConsumed/生产开流等未验层仍按原职责保留，不变成本局部规划finding，也不被局部GREEN解除。未知identity/root writer不因哈希相同释放窗口；本片不写这些路径、不扩大S10.5或设计新恢复架构/Gate。

## 5. 实际Readset及独立证据

方法复用本角色已取得的完整有效Reviewer Role、add-code-review/design、当前delivery/Issue适用方法、add-test-design及checklist/abstraction/写作/decisions/安全与唯一go-service适用边界；未reroute、未加载实施/E2E执行方法或下级委派。非代码对象不强套本轮编译/PG先PASS要求。

- 固定r20完整actual diff、handoff、actual-readset、coverage，r19前像/r20 fixed/current字节；r22固定设计及74BB Review同版正文/结论复用并核哈希。PREP只读固定r19 A-REC及原S4.4 gap建议，不把历史r21措辞当现役r22。
- Dalton固定 `G0.md` 全文（SHA=`63ACD355DA5A9991158E838CBCB31731A2831831D02D6ADD4817296AD9F301FA`），只消费静态九existing+一new接缝分析；未把G0作为已编译/运行证据。
- 原S4.4卡全文及S1.5/S4.4 requires定向抽取/状态/三路径；不审Tasks全体或Plan，不写metadata。S1.5/S8.2完整条件沿原已核边界复用。
- Source/Provider/Store/serializer四生产、五既有测试及Finalizer/平台Trace的必要同版直接接缝沿上轮r22审读复用并核指纹；没有重审全源码。新补读repository.go的公开PostgresReplayFactReader/两SQL过滤与资源关闭、recovery两个现役fixture文件tag/package、旧PG harness的自动DROP/迁移、notice_crash_test.go的child分流/真实提交参考片段。
- AuditSQL文件只核固定SHA及公开proof/lifecycle adapter必要片段；最新verification-s1.5读取审计SQL交付范围/限制，SHA=`E69F2B74404C3F3F4CA5DEB4F2C5F2FC612A6AD1FBC80559B9055E4D4C4428EB`，不重解析旧raw/重审其CodeReview或重新签运行PASS。三份其他contract仅hash保护。
- Lead机械核对回执实际读取，归属Lead；durable8份planning index仅核所给DDE625哈希，不冒称逐份重验全部内容/179源。

个人静态目录：`C:\Users\J00010385\AppData\Local\Temp\han-pauli-s44-r20-review-837b7f8dbc3a4169872b1d0cd9bf0dc5`。`independent-static.json` SHA=`75631BA4AEF2C47FB1834ACA18C2D608F52F96BF9FA037E58A7909A990A62949`记录独立diff、197行/保护区块、4组13场景/嵌套、十二/六依赖及新文件缺失；`readset-start.json`（30项正文/符号/hash-only输入）SHA=`0E40B7F246CDBD5E9BC83DE58AEDF56AAD7383003086F69EA517D023C4391259`；`end-check.json` SHA=`7B25CB7FCE92C660FC51730CBD52A785938C0BD45C52F1493B663A1E34232D05`。定向源码与诊断source179绑定无漂移；本角色未重审/重读全部179项。固定对象与必要输入末核无未知漂移。

## 6. Findings、Gate与停止

新增findings：**无（CRITICAL/HIGH/MEDIUM/LOW均0）**，不制造finding ID、风格整改或新的恢复/人责义务。若后续真实缺陷需要改测试规划，回原Ohm固定新稿，再由原Reviewer有界复审；本角色不代修。

| 范围 | 结论 |
|---|---|
| 固定r20/§4.8.1及相关metadata | 两轮独立有界规划PASS，供Lead判断原卡规划/后续窗口；不发START |
| 新真实链/13命名场景及嵌套、单元/契约回归 | 全部待实施/运行；old-API计数/Trace预计行为仍须实际取证，不预签RED/GREEN |
| 原20Admin/38映射/Case18/S1.5十二依赖 | 完整保留，S8.2及全局恢复隔离/完成开流阻塞不解除；应用ACL仍独立未验 |
| S4.4三路径/六依赖/fullclose | 全部保留；只本片一个新增文件，不把局部PG/Journal seam扩称真旧备份/联合恢复或整卡关闭 |
| Issue/代码Review/人的接收 | 另闭环，规划PASS不证明已修复、不关闭Issue、不代签accepted/checkpoint |

本轮仅文件读取/rg/哈希、只读Git分支/HEAD及个人Python字节diff/映射分析和唯一报告写入。**无Go/test/list/vet/build/race、PG/DSN/env值探针、SMTP/服务/Browser/Unix探针、源码/设计/TaskPlan/Issue修改、Git mutation、cleanup/删除移动、overlay执行或子委派。** 没有启动测试/业务/数据库/持续后台进程，本轮命令均已结束，无本角色遗留进程；未作全机进程普查。

**stopped：固定r20两轮独立TestPlanReview完成并停写，报告交Lead；不授Code/PG、不关闭任务或代签人。**
