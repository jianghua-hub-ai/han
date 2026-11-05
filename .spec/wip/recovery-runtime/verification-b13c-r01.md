# B13C：存储元数据与文章恢复锚定

最新接收：本批明确组件范围已技术闭环；实际任务状态与后继责任以[Tasks原卡](../../../.add/project/knowledge/tasks/tasks.md)及[Plan本批末条](../../../.add/project/knowledge/plans/ir-01-content-community-batches.md#b13c)为准，完整当前结果见[最终Main交付收据](evidence-b13c-r01/main-final-delivery-receipt-r01.json)。下文“当前/待审/NOT RUN”保留各形成时点，最新范围和限制见文末。

2026-10-29，沿[已接收 B13B](verification-b13b-r01.md)继续 development-delivery。用户再次授权待确认或决策采用建议值。固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；保留已有未提交内容，不提交、push、切分支、部署、清理或补人的设计签字。

## 原任务与真实分工

仅消费原 [identity-auth-S4.9](../../../.add/project/knowledge/tasks/tasks.md#task-identity-auth-s4-9) 六文件与 [article-management-S4.7](../../../.add/project/knowledge/tasks/tasks.md#task-article-management-s4-7) 八文件及原 [B13C](../../../.add/project/knowledge/plans/ir-01-content-community-batches.md#b13c)。原 Tasks 是状态、依赖和验收唯一源；不新增任务、不重排九批。各卡技术正文、文件、requires、命令和关闭条件保持，人的设计接纳仍unknown。

- 身份存储：原 Zeno / add-backend-dev，Main实际恢复 `01a18486-3e11-7852-9723-ad380e0bb392`；当前仅只读G0。
- 文章锚定：新 Noether / add-backend-dev，Main实际Host spawn `01a1861b-9405-7942-8dd2-2db8b8d39eb2`；当前仅只读G0。
- 原 Goodall / add-task-planner，实际恢复 `01a18414-b2ab-7392-9b04-fd974203bf5e`，完成[两原卡认领](evidence-b13c-r01/claim-r01/receipt.md)后STOP并Host关闭；Main全文核receipt/两diff和实际哈希、其他71卡及技术字节不变，见[现场认领接收](evidence-b13c-r01/claim-main-reception-r01.json)。认领不等于SOURCE/LOCAL/PG信号。
- Main负责写前基线、精确窗口、隔离数据库、实际差量/验证及接收；文件归属是协作约定，不当作操作系统硬隔离。

## 现有规划适用性

Main全文读原Kuhn Planning R01/R02。R01明确覆盖S4.9和article S4.7，R02关闭其他卡finding并沿用未受影响范围。Main取得原0F197F受审Tasks精确前像，逐字节核当前两卡除checkbox/startdate、owner、batch认领三行外完全相同；不是仅看摘要或将旧报告扩新范围。见[原覆盖的现场适用性核对](evidence-b13c-r01/planning-applicability-main-r01.json)。旧待PlanningReview措辞为历史，不构成当前缺失Gate；此核对不是新Kuhn独立Review，当前源码消费者/契约充分性仍由各自G0核实。

## 成功证据、基线与资源

Main留存3179文件写前基线，14源中9新增路径不存在、5既有源留存精确前像；原5个不同前置Task全[x]。当前只有认领两metadata变化，其他3177基线文件和旧23证据不变，main/HEAD/staged0符合约定。源码、Go/PG执行仍需各卡G0和Main精确信号；范围不足、契约变化或未知写者只停受影响部分，不另加authority、production fallback或Background兼容壳。

身份任务须在真实PG闭环全run短RR只读投影、Rows.Err/NULL/UTC微秒/固定canonical、counter覆盖相关实际写与溢出拒写、0010组合FK/不可变/legacy升级；旧0005/0008不改，唯一0010写者不移给后继。文章任务须覆盖同锚定实例/双namespace权限、全durable边界/ACKunknown、纯读零repair与精确prefix、受控协调/ABA及own-cause/迟到成功；旧payload/HMAC/finalizer业务算法不改。按原风险组验证实际leaf，父top/编译/零命中/Skip不算PASS，行为Red和诊断分开。

Main已启动全新本机 PostgreSQL 18.6 cluster，只绑定127.0.0.1，分配身份/文章两个独立测试数据库，使用新synthetic测试用户和空私有pgpass；不读已有ENV凭据、不连接原业务库。数据/日志保留，完成后只停止并join自有server，不DROP/删除。新retention-only harness及各新PG top需Main另发精确LOCAL/PG信号；旧自动DROP harness不执行。SQL事务、故障注入及回滚只限这套新测试资源。该环境就绪不是产品测试PASS。

[本次写前格式基线](evidence-b13c-r01/baseline-format.json)实际rootfmt FAIL2，原23哈希和两raw与B13B相同；不继承B13B精确格式处置，不报7/7PASS。新源稳定后Main另跑适用项目检查及定向普通/race/PG实际raw，再核精确新SHA格式适用性。

本批组件及真实PG不冒native POSIX/真实介质或Anchor/keyservice、生产root、完整restore/worker/Case18、部署/E2E或生产性能通过。ADR-0024、S3.3/S3.5具名窄范围决定及S3.6、原专项/E2E责任保持。当前尚无本批源码实施、测试、代码Review或最终Task/batch接收。

## G0 实际交回与两条接缝

[原Zeno G0](evidence-b13c-r01/s49-g0-r01/receipt.md)确认六源业务方案可行，但三处清单外历史integration fixture从活动目录Load却硬断言总数9；Main全文核receipt/真实三方法/loader及共享旧清单，接受显式冻结历史0001–0009的建议。已向同Zeno发[独立三源兼容SOURCE/LOCAL](evidence-b13c-r01/compat-source-local-start-r01.json)，仅原三测试文件，原六源仍closed；新top仅纯加载回归，不给PG或旧业务集成运行权限。不能只把9改10、先加载未来SQL后截断、改旧migration/loader或放宽旧业务断言。此缺口原为静态证据，不冒执行Red；未来行为证据另核。

[Noether G0](evidence-b13c-r01/a47-g0-r01/receipt.md)发现C1：r23协调要求opaque PhasePermit及完整closed/drain/exclusive核证，但当前Runtime的公开执行入口不产生coordinate/install，private phaseLocked的原单元正控不证明跨包生产者。Main实读当前API/CheckJournalUse/对应测试及r52承诺Install、r23接口/协调表后，仅停文章受影响范围，八源/LOCAL/PG继续closed；不可伪造permit或用其他purpose/fake Ready绕过。Main实际恢复原Banach/add-feature-designer `01a18389-57ab-7b70-8c86-82a61e7729f1` 做有界只读方案咨询，先给真实最小producer/consumer合同与精确影响，再在用户standing授权内采用建议，不声称设计已修改或新Review已通过。

[Main G0核收](evidence-b13c-r01/g0-main-reception-r01.json)实际核十四源仍原状；两原Task仅追加受理事实，认领/技术清单和73/51/5/17不变。Noether原receipt“用户指定”实际为Main Host派发，不推人签字；其workspace Temp只含自有文档/JSON并已完整复制留存，原诊断零字节材料保持，后续产物使用OS Temp。没有删除/搬移原材料，环境准备不是PG行为测试PASS。

## 兼容缺口闭环与独立范围实施

原Zeno交[三fixture修订](evidence-b13c-r01/compat-author-r01/receipt.md)后STOP/Hostclosed。Main全文核3diff及实际原始Red/Green：两top各9叶中6行为FAIL，最终普通/race各18叶全PASS；原业务断言和SQL/loader/shared清单保持。Main另跑精确pure双top count1普通/race各18叶、三源fmt空PASS，均testDSN空、未PG。原Ptolemy[两轮独立Review](evidence-b13c-r01/compat-review-r01/review-report.md)有界PASS/新增0、普通/race各24叶（原18＋新6），实证只读历史9文件、未来零读取、每个历史读错保cause/短路、真实loader拒坏旧SQL。Main全文核report/receipt/newFSprobes并实际解析四raw及三源entry/exit/live，完成[具体兼容缺口接收](evidence-b13c-r01/compat-review-main-reception-r01.json)；只闭这一G0缺口，不冒完整S4.9或PG/批次通过。

Main已向原Zeno发[原S4.9六源SOURCE/LOCAL/隔离PG](evidence-b13c-r01/s49-source-local-pg-start-r01.json)；只身份新PG top/独立保留schema，全部四风险组P仍须实际执行。与此同时，向原Noether发[八源独立Core SOURCE/U-C LOCAL](evidence-b13c-r01/a47-core-source-local-start-r01.json)：追加/纯读可先实施，C1 coordination保持failclosed/NOT RUN、不可伪造phase签收，文章PG仍closed。Main runner现显式区分pg_s49 START / pg_a47 closed，cluster就绪与core许可不扩PG。Banach继续只读C1咨询，尚未修改设计或Runtime；两原卡执行中，无最终接收。

## C1 推荐采用与设计差量启动

Main全文读取原Banach[只读咨询](evidence-b13c-r01/c1-design-consult-r01/consult.md)，按用户持续授权采用沿既承诺 Runtime.Install 补真实 producer 的建议，不新设通用 Coordinate callback/许可签发入口。已向原设计者发两份原 design 的精确 WRITE：安装专用 executor、独立来源/Anchor 观察、已 bound 同 manifest 协调重入、closed/drain/exclusive 与 ctx/grant 真 join、受控冷打开及同实例回读；完整安装算法仍由 S4.13、root 接线仍由 S4.16 承担，不把局部接缝叫完整安装交付。

见[Main 采用与窗口记录](evidence-b13c-r01/c1-design-start-main-r01.json)。设计差量完成后须独立评审及原 Planner 精确映射，再开放 Runtime 精确源窗口；此时 Runtime 源、Article C1 正控和 Article PG 仍 closed。S49 原六源/真实 PG 与 Article 原八源 core 继续，Store/canonical/metadata 原合同不变。Main 另读取四份实施中间快照，仅供现场理解，不绑定最终对象、不签代码 Review/PASS；实际 guard 核 main/HEAD/staged0、精确窗口及旧23保护不变。人的责任确认未知继续保留，不以技术采用代签。

## C2 追加完成线性化的实测缺口

Main读取 Noether 的两个真实 Runtime 组件探针及实际消费者：CompleteArticleAppend 成功清自身 token 后，Append 仍可能因后到取消或 namespace Close 失败返回错误/unknown；两个定向行为探针实际各一叶 FAIL，后续真实普通 Acquire 却成功。见[原始失败与现场观察](evidence-b13c-r01/c2-main-observation-r01/index.json)。这是一项当前 Article 消费者缺口，不自动重开旧 finding。Main已交当前原设计者在两 design 精确差量中核成功通知/资源真 join/取消边界，文章受影响完成切片保持待修，不用过滤后的 core PASS 代整个 OwnCauseAndLateCompletion 通过；其余实施继续。

## 冻结候选与独立评审/PG接续

原Noether交[partial core](evidence-b13c-r01/a47-core-partial-r01/receipt.md)后STOP/Hostclosed，Main匹配原八路径/七present并实际解析完整普通/race：各117叶PASS、C2两叶FAIL、0skip；旧回归各39PASS。C1正向/P尚未交付，不能闭卡。Main随后只给原owner新增原清单内 PG integration 一路径的独立SOURCE/LOCAL/隔离article库窗口，其余七源冻结；不是C1/C2修复或第九源授权，仍禁旧autoDROP。

原Banach已交[两design差量](evidence-b13c-r01/c1-design-author-r01/receipt.md)，identity r53 DD61F4B8…590A6F0 / article r24 69C56796…7CAC619；Main逐份before/live/after/确认尾部匹配并冻结。Dirac/add-reviewer实际ID `01a18663-d9ac-79c2-b843-ba6dcca3de7e` 正在非作者设计评审。原Goodall `01a18414-b2ab-7392-9b04-fd974203bf5e` 仅只读核新Runtime两源义务与原任务映射，Task/Plan无新写；Runtime SOURCE仍closed，不把未实现义务隐藏在旧[x]或倒挂完整安装/root。

原Zeno已交[原S49六源](evidence-b13c-r01/s49-author-r01/receipt.md)后STOP/Hostclosed，Main匹配全部六源并读生产投影/0010全文及既有两源diff、新PG harness/RR/Rows.Err/overflow关键段，实际解析作者普通/race Encoding38、PG22、recovery390全PASS/0skip。Ptolemy原Reviewer `01a18492-af9e-7ea2-9cd5-e4dd368d7bee` 已实际恢复做精确六源CODE Review，未改被审源。

[Main独立定向执行证据](evidence-b13c-r01/main-s49-directed-r01/receipt.json)另含count1普通/race编码各38叶、真实隔离PG各22叶（四组4/7/4/7）全PASS，所有required无skip/fail/unfinished，五Go源fmt空PASS及各新schema保留记录。这是现场核验，不冒独立Review或Task/batch/全局七项通过；Article失败、C1、完整安装/root/真实环境和人的确认边界保持。

## C1 设计复审接收与 C2 独立修复

Dirac[设计 R01](evidence-b13c-r01/c1-design-review-r01/review-report.md)发现唯一 MEDIUM `B13C-DESIGN-R01-F001`：snapshot 供证等待它本应支撑的 Runtime 后证。Main全文核报告，向原Banach仅授 article design 最小修订；[r25 修复](evidence-b13c-r01/c1-design-author-f001-r02/receipt.md)明确本地协调/完整纯读核验后先供观测，Runtime随后完成独立后证。Main匹配原r24/live/final及identityr53不变，原Dirac[独立R02](evidence-b13c-r01/c1-design-review-r02/review-report.md)已关闭原设计F001、新增0，原未变范围/限制承接。见[Main精确设计接收](evidence-b13c-r01/c1-design-main-reception-r02.json)，不是代码或人的确认PASS。

C2合同经R01未发现新增设计问题，且与F001独立、r25字节不变；Main因此仅向原Noether追加原anchor.go/anchor_test.go两源局部修复与定向验证窗口，要求所有fallible Close/check/取消先于最终Complete、真实join及完成边界两侧正负证据，保留旧两FAIL。其原PGtest一文件继续，C1正向实现仍closed。原Zeno已恢复只读Runtime两源G0，原Goodall仍只读任务映射；Runtime源码未写、Task/Plan尚无新增义务登记，后续需正式映射和精确SOURCE，不因设计接收跳过。

## S49 独立代码评审接收

原Ptolemy[六源两轮代码评审](evidence-b13c-r01/s49-review-r01/review-report.md)有界PASS、finding0，普通/race各67叶（原编码38＋PG22逐名保留，独立新增编码3＋PG4）。Main全文读report/receipt、全部新增unit/PG探针及Python显式oracle构造，实际解析四份最终raw：41＋26每模式全PASS、无skip/fail/unfinished/race，核六源entry/exit/live及52唯一schema保留。独立真实Runtime.Verify消费真实Store/PG与外部显式fixture的正负来源闭环，仍不授ordinary work；不冒真实restore/provider/root。详见[Main现场接收](evidence-b13c-r01/s49-review-main-reception-r01.json)。

S49尚未勾闭：Main协调的全局检查仍待稳定源码窗口，后续Runtime补丁须核对本结论的依赖适用性。Article两个完成边界FAIL及C1组件义务不因S49评审关闭；B13C还需跨卡/覆盖/必要性能与独立整体Review。原Planner已开始仅Tasks/Plan的正式新增producer卡及映射修订，Runtime SOURCE仍closed，Main不与其并写这两文件。

## C2 未终结 namespace 资源的新实测边界

Noether修排序后又以真实NewRuntime/RunNormal/WorkLease/Fault/Stop复现一叶FAIL：namespace Close返回未知并实际仍持有synthetic文件handle时，WorkLease终结及Stop之后DeploymentExclusive.Close已被调用。Main全文读原gap、失败raw末段/实际测试体及当前Runtime drain消费者并匹配raw SHA，见[原始复现与现场观察](evidence-b13c-r01/c2-ownership-main-observation-r01/index.json)。这是新的资源所有权接缝，原两个通知排序失败的修复不等于完整C2关闭；不自动重开旧HIGH，也不冒nativePOSIX实证。

Main已令受影响实现停止探索并向原Banach仅发只读有界咨询，先核现有cleanup债/生命周期能力及最小合同和精确影响，再正式处理；Runtime两源尚未开放。当前两Task/Plan冻结的独立PlanningReview可以在精确对象内完成并客观列此新增上游限制，不能把它当新合同已经映射。Article PG独立场景继续完成/保留，未获新hold API/第9源权。

## 新任务规划接收、C1实施与Article阶段核收

原Goodall[正式两稿](evidence-b13c-r01/c1-planning-author-r01/receipt.md)实际新增identity-auth-S4.18初始todo并在原B13C消费，保留旧S4.7[x]/S49原卡。Main全文读两diff/receipt、核before/live/after并独立解析74节点319边，仅3条新边、无环/未知/重复，原73状态及70未改卡保持；原Dirac[PlanningReview](evidence-b13c-r01/c1-planning-review-r01/review-report.md)有界PASS/findings0，明确L-C2-OWNERSHIP-01新资源场景不在实施充分性结论内，Main实际接收见[记录](evidence-b13c-r01/c1-planning-main-reception-r01.json)。

原Noether[当前C2/P阶段](evidence-b13c-r01/a47-c2-pg-partial-r02/receipt.md)STOP/Hostclosed，Main匹配源8/前五冻结及实际6份raw：core普通/race各126PASS/1FAIL、C2同名组9PASS/1FAIL，PG各13PASS、旧reg各39PASS，原117通过名/39回归名逐名保留；79作者schema保留。Main另亲跑新PG top普通/race各13叶全PASS，并亲自复现unknown Close资源ownership一叶真实FAIL；见[Main定向证据](evidence-b13c-r01/main-a47-directed-r02/receipt.json)。这不是Article CodeReview/卡关闭，C1正向NOT RUN、C2资源问题OPEN。

PlanningReview明确独立C1 producer不被新C2场景倒挂。Main因此在全部Article8已冻结、所有Go/PG结束后，实际恢复原Zeno并记录S4.18认领，仅[C1两源SOURCE/LOCAL](evidence-b13c-r01/s418-source-local-start-r01.json)START。Main此处是已受审技术的状态/日期/owner/batch小改，非新Planner产出；Task技术逐行不变，Plan保持358BE，当前Tasks EB9B8129…1B11367，74/51完成/6active/17todo。Runtime新cleanup合同/API仍未授权，由原Banach只读consult；不因C1开工伪称资源问题已解决或设计/人的签字已完成。

## C2 原资源生命期方案与设计冻结

Main 按用户持续授权采用原 Banach [最小原生命周期建议](evidence-b13c-r01/c2-ownership-consult-r01/consult.md)：私有 namespace lease 的 `WaitReleased(ctx)` 任何返回都须表示实际资源终结；未知先 Fault/撤权，原 Append/WorkLease/action 留在途直至真 join，沿同一 Runtime drain 收口，不新增 Runtime debt/hold/Resolve API。底层无法证明终结时保持 pending/独占，不以超时、FD/布尔或重试 Close 冒证明。当前原一叶 FAIL 保留，尚无修复通过结论。

原 Banach 在两 design 窗口交付 [identity r54/article r26](evidence-b13c-r01/c2-join-design-author-r01/receipt.md) 后 STOP/Host CLOSED。Main 已核实际 before/live/after、旧技术行保留及两差量，见[冻结记录](evidence-b13c-r01/c2-design-main-freeze-r01.json)。原 Dirac 已恢复独立评审本新 C2 差量（runtime ID `01a18663-d9ac-79c2-b843-ba6dcca3de7e`），旧 F001 CLOSED 与原 Planning PASS/限制分别保留；当前 C2 Review 待返回。Zeno 独占 S4.18 C1 Runtime 两源窗口继续，Article 八源/TasksPlan/其余 source 冻结。本记录不构成人的接纳、代码关闭、native/完整安装/root/E2E 通过。

## C2 设计独立接收与 S4.18 源码候选

原 Dirac [C2设计两轮独立评审](evidence-b13c-r01/c2-join-design-review-r01/review-report.md)有界PASS、新增0；Main全文核report/receipt及冻结两对象后[采用技术差量](evidence-b13c-r01/c2-design-main-reception-r01.json)，旧Planning L-C2-OWNERSHIP-01仍按历史保留，代码仍OPEN。原Goodall现仅对Tasks/Plan中Article/S4.16生命周期/验证做精确映射，保护S418整卡、原74状态/依赖/路径/命令；尚待新delta独立PlanningReview。原Noether[只读G0](evidence-b13c-r01/a47-c2-join-g0-r01/receipt.md)核source8全部匹配，候选5路径，无第9源/RuntimeAPI需要；已STOP/Hostclosed，无SOURCE。

原Zeno交付[S4.18两源候选](evidence-b13c-r01/s418-author-r01/receipt.md)源码STOP/frozen/Hostclosed。Main匹配before/live/after、生产新增差量及关键test消费者，实际解析作者6份final raw：Install105（六组28/22/15/15/22/3）、旧Admission271逐名及正文完整保留、HTTP56，ordinary/race均零fail/skip；真实编译行为Red保留。见[现场对象及raw核验](evidence-b13c-r01/s418-author-main-freeze-parse-r01.json)。原Ptolemy（`01a18492-af9e-7ea2-9cd5-e4dd368d7bee`）已恢复独立两源CodeReview，Main不并发Go；当前不能说本卡/Article C1/批次已通过。

作者构建曾因Windows commit不足失败，串行有界重试后定向测试通过，ENV/BUILD失败未冒Red。原自有linker55164被请求终止后exitcode2但原生等待仍258；[Main亲验](evidence-b13c-r01/own-linker-main-observe-r01.json)确认该资源尚未signaled，原parent73092已signaled。该次失败编译未启测试二进制/PG；源码冻结与测试结果可分别核收，但**不声称全部原生进程已join**。此宿主收尾限制保留，未改其他进程/服务或清理文件。[Main候选接收](evidence-b13c-r01/s418-candidate-main-reception-r01.json)。

## C2 任务映射冻结待独立评审

原Goodall交付[两SSOT精确差量](evidence-b13c-r01/c2-planning-author-r01/receipt.md)后STOP/Hostclosed：Tasks `EAFCC8E05617F0B0CD027A47548C1E9F995534F8D51982646590C4F79D4ABB3D`、Plan `E07E98509DC14C7566FC6ACB59A86AC893E1A0FED5842A6F450597F54B26E9AD`。Main全文读两diff/receipt并独立核before/live/after、原74状态不变/其余72卡逐字节同/S418整卡认领保护、全部文件/命令/requires不变、74节点319边无环/增边0、九批同，见[现场验证](evidence-b13c-r01/c2-planning-main-verified-r01.json)。当前仍74/51完成/6active/17todo；原Dirac已实际恢复（ID `01a18663-d9ac-79c2-b843-ba6dcca3de7e`，submission `01a186d8-1136-7a42-9e2f-36f65a838da5`）做本次C2映射独立增量PlanningReview。全部源码/metadata冻结；Ptolemy只有S418定向评审LOCAL，Noether无SOURCE。旧L-C2限制保留历史，本次新增待审不冒实现PASS。

## C2 规划独立接收与五源修复启动

原Dirac[新C2映射两轮PlanningReview](evidence-b13c-r01/c2-planning-review-r01/review-report.md)有界PASS、新增0，入口/出口Tasks EAFCC8/Plan E07E985一致；Main全文读report/receipt并按[原现场验证](evidence-b13c-r01/c2-planning-main-verified-r01.json)接收，见[精确技术接收](evidence-b13c-r01/c2-planning-main-reception-r01.json)。旧L-C2限制留历史，本结果仍不是代码实现/人的签字。

Main已恢复原Noether（`01a1861b-9405-7942-8dd2-2db8b8d39eb2`，submission `01a186e3-96a7-71c1-8b58-c550fb066050`），按其G0仅授[原五源C2 SOURCE](evidence-b13c-r01/a47-c2-join-source-start-r01.json)：anchor/原两平台适配/anchor及snapshot tests；其他三Article和Runtime两源/S49/metadata均冻结。LOCAL/Go/PG此时CLOSED，先编写代码/双侧测试，待Ptolemy S418唯一Go窗口真结束后另发LOCAL。原Main行为FAIL仍保留，当前Runtime原NewRuntime起方法suffix字节未改，原失败根因证据适用但不冒新Red执行；C1正控实现仍closed，等S418接收及后续精确窗口。

长评审期间Main另做[隔离PG只读preflight](evidence-b13c-r01/pg-health-readonly-r01.json)：18.6/127.0.0.1:59672/原自有data/user及两新库匹配，identity保留240、Article105个本范围schema；仅SELECT、不改任何数据或服务，不记产品PG断言重跑。后续测试仍只用该隔离范围，最终由Main stop/join且保留数据。

## S4.18 独立代码评审接收与 C2 测试窗口交接

原Ptolemy[S4.18两轮独立CodeReview](evidence-b13c-r01/s418-review-r01/review-report.md)有界PASS/finding0，源码两对象entry/exit/live一致；Main全文核report/receipt、独立10探针及两个Python显式wire oracle构造/仅test overlay，实际解析ordinary/race各489节点442叶全PASS、Install115＋Admission271＋HTTP56，原105/271/56逐名保留、新10均命中、零skip/fail/unfinished/race。两个Go源码fmt空及vet0。见[Main现场接收](evidence-b13c-r01/s418-review-main-reception-r01.json)。作者HTTP raw的test harness SHA是较早追加测试前版本、production Runtime相同；Reviewer本次以精确当前对象实际重跑，旧作者结果按此限制保留，不冒当前HTTP新执行。

原Reviewer全部Go已实际返回、STOP/Hostclosed，Main因此向原Noether追加[五源C2 LOCAL及原新PG13专项](evidence-b13c-r01/a47-c2-join-local-pg-start-r01.json)（submission `01a186e6-aebb-7431-b10a-0ffcf308d544`）。仅其独占编译/测试，GOMAXPROCS2/-p1/offline，普通DSN空；PG只原隔离article库/原新top、schema保留，允许Linux compile-only而非native执行。其他3Article/Runtime/S49/metadata冻结，C1源码仍closed；待C2结果冻结后Main做同revision检查/格式有限处置及S418局部接收，再授真实Article C1。S418/S49均未因此勾闭，本批还无整体Review/最终接收，旧linker OSjoin限制独立保留。

## C2 五源核收、同修订联合实测与两卡关闭

原Noether[C2完整阶段包](evidence-b13c-r01/a47-c2-join-author-r01/receipt.md)STOP/Hostclosed，Main核原8 before/live/after、仅5改3同，全文读生产差量及关键panic/Stop heldfile用例，独立解析作者ordinary/race core145/oldreg39/PG13全PASS；新3组5/1/10、Complete12逐叶非零，旧117/126passing/39/13清单保留，原ownership行为Red1与中间completion-panic Red2/精确中间before-overlay Red2都是真实编译后失败且最终已通过。中间555913源不是原始1BC81源；C1仍未实现。详见[Main对象冻结](evidence-b13c-r01/main-current-joint-r01/a47-c2-join-main-freeze-r01.json)及[实际raw解析](evidence-b13c-r01/main-current-joint-r01/a47-c2-join-author-main-parsed-r01.json)。作者52新schema保留，Linux仅compile-only，未冒native证明。

原Ptolemy[Article完整8源第一段两轮静态审查](evidence-b13c-r01/a47-review-phase1-r01/review-report.md)已读4911行、沿Runtime/Coordinator/Finalizer/HTTP/replay消费者追核，0 established finding，但本阶段其Go/PG/独立运行NOT RUN、最终CODE仍PENDING。PV01–06包括真实C1/当前C2独立探针、外部固定wire、actualcommit后ACKerror等剩余核证；Main已全文读报告/receipt并保留，不将静态0finding签整卡CodeReviewPASS。原Dirac[批次第一段前检](evidence-b13c-r01/batch-precheck-r01/precheck-report.md)同样0直接缺陷、整批PENDING；P01/02/03/04及native/host下游边界单列。

Main此后唯一Go窗口亲跑[当前联合执行包](evidence-b13c-r01/main-current-joint-r01/receipt.json)：普通/race每模式**696具体leaf全PASS**，联合616=RuntimeInstall105+Admission271+HTTP56+Articlecore145+oldreg39，另S49编码41/真实PG26与ArticlePG13；所有required名称/组和ordinary-race集合相同、0fail/skip/unfinished。S49真实PG/Store→当前D404 Runtime.Verify独立正负probe已实际重跑，补旧284886版本边界；全来源internal_ready观察仍拒Acquire。78个新Main schema逐名SQL只读核实仍存在，全部保留。Main使用同一测试项目隔离127.0.0.1:59672及两库，未跑旧autoDROP。原Review设计探针的Main重跑按实际执行归属，不冒Main新独立设计。

Main以原Manifest精确顺序执行七项：mod/vet/build/contract/race/ordinary六项PASS；根gofmt仍FAIL2，stdout D01225…/stderr53258F…与本批preSOURCE旧23原始证据基线字节完全同，旧23全部hash未变，另18Go源fmt空PASS。Main按standing授权采用[本次仅S49/S418精确8源任务有限格式处置](evidence-b13c-r01/root-format-disposition-before-c1-r01.json)，没有格式化/删移旧证据、没有把根fmt或全七标PASS；未来Article C1新SHA必须再实测/处置，不自动继承。首次runner准备list/dict类型错误发生于任何Go命令前，保留空r01目录；真实七项为r02，未冒产品FAIL。

Main现场status/diff/HEAD/staged/源19与旧23保护核对、独立Review/当前必要测试与成果接收齐备，已[task-level关闭S49/S418](evidence-b13c-r01/s49-s418-task-main-close-r01.json)：Tasks **C20F983FCA66440ADE7B44AB17416B5CDCCD814D2F9E2492E78DFEB98DFBB86D**；**74/53完成/4active/17todo**，active为原ArticleS1.5/S4.4/S8.2及本ArticleS4.7。只两header/完成date/两Main接收事实notes/当前counter改变，inverse exact保护全部技术/原依赖/清单/命令及其他72卡；Plan仍E07E985…不变。Main这一步是实际任务接纳/状态事实，不是新Planner产出或人的角色签字。Source2/六源交付仅局部技术关闭，不冒整批、完整安装/root/native/E2E/容量。

[性能适用性Main取值](evidence-b13c-r01/performance-applicability-main-r01.json)保留全历史排序/可能平方prefix成员检查、重复Store RR与singletonrevision串行成本；本测试项目当前仅功能/组件/取消真join/真实PG子集，未给容量SLO，不造benchmark阈值或裁历史/让join超时退。最终Dirac仍须独立评估P04，Main未代闭整批。旧自有linker55164 OSjoin限制仍按最后亲验单列，不因任务关闭称全宿主join；PG最后仍由Main stop/join并保留数据。

当前所有源码/metadata仍冻结，所有Main Go/PG已完成；原Noether（ID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`）仅在做真实Article C1只读G0，当前无SOURCE/LOCAL。其前置S418已实际释放，G0返回后Main将给原8精确窗口。Article最终CodeReview/批次最终Review尚待真实C1/final freeze及同修订核证，不把当前组件结果直接相加成整批PASS。

## 真实 Article C1 四源实施启动

原Noether[C1只读G0](evidence-b13c-r01/a47-c1-g0-r01/receipt.md)已STOP/Hostclosed，Main全文核receipt、接口映射、port/lifetime notes与component/risk/path建议并匹配source8/Runtime2冻结哈希；原S418已[x]真实释放。Main按用户standing推荐值采用原8内最小四路径方案，仅授[anchor/snapshot及各test SOURCE/LOCAL/原PG13回归](evidence-b13c-r01/a47-c1-source-local-pg-start-r01.json)，实际owner仍Noether（ID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`，submission `01a1872e-bc3e-70d2-8053-9c888c4d6a4e`）。其他4Article/Runtime2/S49six/compat3及全部metadata/design冻结；唯一Go/隔离articlePG窗口给作者，Main不并跑。

组件正控必须真实public Runtime.Install/PhaseContext/checker→同一实际control-open核心/Coordinate writer→同handle本地完整纯snapshot先供→Runtime独立postproof；同一个真实fixture backend/key、封存起点不rebase，generationctx纯读、holding namespace不重入，保留C2/145+39+13。Windows原生public wrapper继续unsupported负控，portable核心正控不冒原生public-wrapper正向执行；原生正控NOT RUN。Install后长期handle/后证失败的root资源先Close/join再归还deployment/pool仍S416明确未验，不以fakeproviderhold/提前关snapshot或普通work自动计数假证；当前在途组件join另验。G0/源码授权不是C1测试/CodeReview通过。

原Ptolemy已恢复仅OS Temp构造最终PV01–05独立probe（submission `01a18731-f7b0-72a0-a93b-a4dca711e7cc`），不读取活跃4、不跑Go/PG，特别准备实际SQLcommit后注ACKerror并独立读状态及重复完成负控；是验证建议而非已知源码finding/修复。原Dirac也恢复仅只读核本G0组件/native/root证据边界及Main新696/P03/P04适用性（source4零读/零hash、不跑Go），最终G4仍需finalfreeze和真实完整当前证据。当前Tasks仍C20/74-53-4-17，PlanE07不变。


## ENTRY-MAP 规划增量独立接收

原Goodall[正式映射](evidence-b13c-r01/entry-map-author-r01/receipt.md)冻结Tasks FE938545…/Plan734616DB…；Main独立核before/live/after、其余73卡字节一致、74节点319边零增删无环、九批及原命令/风险/最低层不变。原Dirac[两轮独立PlanningReview](evidence-b13c-r01/entry-map-review-r01/review-report.md)有界PASS、新finding0，Main全文读report/receipt并匹配两对象后[实际接收](evidence-b13c-r01/entry-map-main-reception-r01.json)。当前仍74/53完成/4active/17todo。

本次只闭验收文字澄清：组件正控用真实公开Runtime.Install/pc/checker与具名wrapper共用的实际core，全部共同校验/状态/协议仍须最终源码独立核实；Windows具名wrapper真实unsupported负控与Linux仅compile-only另验，具名原生正控固定NOT RUN/未验证。S416长期/后证失败handle的真实root Close/join责任不变，未套用旧豁免，未冒人的接纳或新TE批准。实际C1/CODE/G4仍pending。

原Ptolemy[独立probe预备包](evidence-b13c-r01/a47-review-probe-prep-r01/receipt.json)含PV01/02/03/05共26叶及外部wire oracle，Main已读receipt/说明；当前只是prepared，Go/PG均NOT RUN，PV04待真实C1冻结适配。Noether仍独占C1四源及唯一Go/原PG13窗口；Main不并跑或提前冻结活跃源。


## C1 最终四源冻结及独立执行窗口

原Noether[C1最终作者包](evidence-b13c-r01/a47-c1-author-r01/receipt.md)已实际STOP/Hostclosed。Main全文读两生产diff、receipt及entrymap，并[现场核对原8 before/live/after与raw](evidence-b13c-r01/a47-c1-main-freeze-parse-r01.json)：仅原授权四源变化，全部其余frozen对象匹配；普通/race每模式core200（old145保留＋新55：Journal45/Snapshot10）、oldreg39、原PG13共252叶全PASS，零fail/skip/unfinished及race，C2四关键组12/5/1/10保留。作者78个新schema按其包保留；Main未在此句冒全cluster census。原编译/timeout/portable adapter诊断按原像保留，不计产品Red/PASS，不称旧timeout graceful join；Main首次解析group误将Snapshot同名也计入而assert退出，收窄到原Journal后解析通过，无源码修改或新Go执行。

最终四源SHA为anchor AB9D369A8BF0BD8671C4F5230121EDDDA5C34C74E2298002D132364BF074C05D、snapshot45A0F9473D75750C53FEE24D46345430859C1440D356BEB252DD85EFE4958FB4、anchorTest16E9D452C4CC1949B8442754787988972877767C8A809E25D918CA58192CA517、snapshotTest579683EBB4DB25D846B9572C95457CD51F154C9B1DF2FD680CD243A20ADD0CDE。Windows具名nativewrapper unsupported为作者实际负控，Linuxcompile-only/原生正控NOT RUN按各自范围，仍待原Reviewer独立核。

Main实际恢复原Ptolemy（ID `01a18492-af9e-7ea2-9cd5-e4dd368d7bee`，submission `01a18754-db06-7490-bb85-33c2539426cf`），按[另授LOCAL/ReviewerPG窗口](evidence-b13c-r01/a47-final-review-local-pg-start-r01.json)独占唯一Go，当前test-only overlay适配PV01–05，特别实际Commit后ACKerror3叶；SOURCE及metadata全部冻结，Main不并跑Go。原Dirac（ID `01a18663-d9ac-79c2-b843-ba6dcca3de7e`，submission `01a18755-7fee-7411-aaab-13080d1c5db1`）只读补最终跨卡接口/依赖/覆盖/P04成本，Go/PG禁止，等待Ptolemy和Main最终证据后才出G4。作者自测及本冻结尚非最终CodeReview、Main最终全局检查或整卡关闭；Tasks仍74/53完成/4active/17todo。


## Article 独立R01取消边界HIGH与返工准备

原Ptolemy[完整8源两轮CodeReview R01](evidence-b13c-r01/a47-review-r01/review-report.md)为CHANGES REQUIRED，唯一HIGH `ARTICLE-B13C-S47-CR-R01-F001` OPEN：anchor.go419–426候选Open成功返回、实际操作ctx已取消后，仍新调用Sync。普通/race独立反例各FAIL1、同路径健康对照各PASS1；Main全文读报告/receipt/探针及raw，并[现场匹配八源和逐叶事件](evidence-b13c-r01/a47-review-main-reception-r01.json)。不是编译/timeout/no-match或root缺失，也未证明提前DELETE/204/数据损坏；原Canceled及cleanup真实返回保持。

Reviewer原core200＋PV01/02/03共23＝223、oldreg39、原PG13＋PV05three＝16，各模式278叶全PASS；实际SQLCommit nil后注syntheticACKerror、503与独立SQL已提交/同op不重复完成/公有重试404已真实独立验证，32新schema保留，非真实网络故障。剩余独立PV04矩阵须修复后续审，不能以278通过盖此HIGH。原13命令与所有自有callback实际返回、STOP/Hostclosed，Go/ReviewerPG窗已闭。

原Dirac[最终两轮整体静态第一段](evidence-b13c-r01/batch-final-static-r01/static-batch-review-report.md)0新finding、19源＋4设计/meta entry/exit稳定；Main全文读report/receipt并保留。G4仍PENDING，静态0finding不关闭上述动态HIGH；原P01–P04及后继/native/host边界原样等待最终新源码及实证。

Main已恢复原Noether（ID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`）仅只读F001 G0，先核候选Open→Sync间真实许可/ctx/ownership检查及原fileClose/namespacejoin保留，预计原anchor/anchorTest两源内；当前SOURCE/LOCAL/PG全部closed，待其精确G0才单独开窗。Tasks仍74/53完成/4active/17todo，没有新Task/新契约/用户重复批准或整卡关闭。


## F001 原两源最小修复启动

Main按已核原Reviewer真实HIGH/原r54-r26已确认预期及用户standing授权，实际向原Noether发[精确两源SOURCE/LOCAL/原PG13](evidence-b13c-r01/a47-c1-f001-source-local-pg-start-r01.json)（submission `01a1876b-06a8-7c02-896c-0acecbe78e32`）。只anchor.go/anchor_test.go，Main精确before AB9D/16E已保留；其余6Article/Runtime2/S49six/compat3/meta/design全冻结。原Ptolemy所有13命令真返回、STOP/Hostclosed已核收，因此作者独占Go，不存在并跑。

原作者仍须写前完成G0核读；本已明确预期/两路径的最小修复不再自造另一次G0审批等待，可把G0事实与实施包同交。若需要第三/第九源/新API/契约则停受影响部分。普通/race须保留core200/old39/原PG13及新afterOpen取消、ownership相关与健康回归；先实际行为Red再Green，原fileClose/namespace真join与根因保留。作者PG仅原新13回归、隔离article库新schema保留，不借ReviewerPV05权。作者Green不关闭HIGH，仍须原Ptolemy复审及剩余PV04/Main最终检查/Dirac最终整体验收；当前Tasks状态不改。


## F001 作者最小修复核收与原Reviewer R02开窗

原Noether[修复包](evidence-b13c-r01/a47-c1-f001-author-r01/receipt.md)实际STOP/Hostclosed。Main全文G0/fulltwo.diff/receipt并核[八源before-live-after及逐叶raw](evidence-b13c-r01/a47-c1-f001-main-freeze-parse-r01.json)：仅anchor.go261AEB4F…FAE9ED、anchorTestA49C129E…9287FC变，其余6/Runtime2/S49six/compat/meta/design保持。3行既有checkControl在候选Open成功/fileClose defer已登记后、Sync前；[Main反向精确保护](evidence-b13c-r01/a47-f001-main-exact-delta-r01.json)证明除guard外全部旧生产字节、除新3叶block/registration外原200测试正文保持，各final命令before/after源匹配当前冻结。

作者实际Red普通/race各取消FAIL＋namespace失效FAIL＋健康PASS，同测试字节/旧AB9D生产保留；Green三叶各PASS。最终core203＋old39＋原PG13每模式255PASS/0fail/skip，原200/145/C2风险名保留，fmt空/vet/Linuxintegrationcompile-only通过，26新schema保留。早期fixture诊断未冒行为Red。此为作者修复/自测/Main核原像，HIGH尚待原Reviewer关闭，整卡/批次未关。

原Ptolemy同ID当前准备剩余PV04期间，Main已[明确R02 LOCAL及ReviewerPG](evidence-b13c-r01/a47-review-r02-local-pg-start.json)START（submission `01a1877b-0eb0-7653-bbc5-c3ecdf330bd6`）：全部源码/meta/design冻结，它独占Go，当前test-only overlay须用A49新test重建并保留原/新3叶；真实复跑原F001失败反例＋健康对照、补剩余独立PV04，保持PV01–03/PV05/current203/39/13，另授隔离ArticlePG16新schema全保留。Main不并跑。仍须原Reviewer R02与Main最终联合/global7/当前格式处置、原Dirac最终整体接收。Tasks74/53完成/4active/17todo保持。


## Article 独立R02核收及Main最终同源实测

原Ptolemy[独立R02完整代码结论](evidence-b13c-r01/a47-review-r02/review-report.md)在明确ENTRYMAP组件范围CODEPASS，原HIGH `ARTICLE-B13C-S47-CR-R01-F001` CLOSED、新0/无剩余已承诺PV04点。Main全文读report/receipt/新增PV04片段、[现场核source8/所有payloadhash及actualraw](evidence-b13c-r01/a47-review-main-reception-r02.json)：每模式226（当前core203＋PV01/02/03共23）＋PV04与原F001pair14＋old39＋PG16＝295叶，全run/pass同名、0skip/fail/unfinished/race，当前source261A/A49与其他6保持。两类探针诊断（Windows closedStat oracle/selector拼名）原像保留，未记生产Red或漏测PASS，修正后精确226/14重跑。实际Code/PV05与原R01来源保留，32新schema保留；STOP/Hostclosed/all14wait与自有callback真join已核。

Main随后[另授本机最终执行](evidence-b13c-r01/main-final-local-pg-start-r01.json)，亲跑[当前最终联合包](evidence-b13c-r01/main-final-joint-r01/receipt.json)：普通/race各757叶＝联合674（Runtime105＋Admission271＋HTTP56＋Article203＋old39）＋S49编码41＋真实PG26＋Article真实PG16，全部旧/新必需名保持、0fail/skip/unfinished/race。S49真实Store→当前D404 publicVerify正负链及真实SQLCommit后syntheticACKerror/503/独立committedSQL/idempotence/公有404均在当前19源修订实际重跑；后者复用原Reviewer设计，Main不冒独立新设计。52identity＋32article＝84新schema经Main只读SQL逐名核存在并保留，未跑旧autoDROP或业务DB。源码19/meta前后稳定。

Main精确Manifest七项mod/vet/build/contract/race/ordinary六PASS；rootgofmt仍FAIL2，stdout D01225…/stderr53258F…与原preSOURCE旧23基线字节完全同，旧23hash逐个同，当前18Go fmt空PASS。Main按用户standing采用[当前精确19源/本组件任务及批次有限格式处置](evidence-b13c-r01/root-format-disposition-final-r01.json)，不继承旧8源处置、不把rootfmt或全七记PASS，也不豁免代码/security/PG/native/root/E2E。Main实际guard核main/HEAD/staged0/全部frozen及保护基线无越界。所有Main Go/测试PG命令已actualexit、local与pg测试窗口closed；隔离server88611仍由Main持有，待最终收口stop/join保留全部数据。

原Dirac[F001两轮静态增量](evidence-b13c-r01/batch-f001-static-r01/partial-review-report.md)已全文Main核收、新0finding，但保留当时HIGH/G4pending；当前已恢复原Dirac同ID读取R02/Main最终证据并作最终整体G4，仍不从单卡PASS相加推断。当前Tasks仍74/53完成/4active/17todo，Article/批次尚待Main接纳。

Main另[亲自最终只读观察旧linker](evidence-b13c-r01/own-linker-main-final-r02.json)：PID55164 image与创建时间08:18:16.541477匹配原自有失败build，exit2且Wait0=258仍未signaled；原parent73092 Wait0=0。没有终止/修改其他进程，没有称全部Host进程已join，也不作产品FAIL；当前全部本批测试命令实际退出与此旧Host限制分列。


## 本批最终接收与环境收尾

原Dirac[最终whole-batch独立报告](evidence-b13c-r01/batch-review-final-r01/final-review-report.md)在正式ENTRYMAP组件范围及当前19源有限格式处置下G4技术PASS/new0，P01–P04在明确范围实证齐备；Main全文read并[现场接收](evidence-b13c-r01/batch-final-main-reception-r01.json)，未把单卡标签或数量相加冒整批。原Noether作者修复、原Ptolemy Code R01 HIGH→R02真实CLOSED及当前独立295、Main当前757/每模式与global6/rootfmtFAIL2保持分别归属。

Main已[实际关闭Article原任务并接收本B13C](evidence-b13c-r01/b13c-final-task-batch-main-close-r01.json)：仅header/date/单卡接收note/currentcounter和Plan B13C一段事实变化，原73卡/全部技术/命令/files/requires/11风险最低层及9批DAG保留。现场计数74/54完成/3active/17todo，权威状态见原Tasks；当前active仍仅原ArticleS1.5/S4.4/S8.2。独立结构重核74节点319边、delta0、无环/重复/未知；只变接收事实，代码19字节仍为已测同修订，未为metadata重复Go。

Main自有[隔离PostgreSQL真实停止与Join](evidence-b13c-r01/postgres-final-stop-r01/receipt.json)完成：session88611/helper exit0、实际pg_ctl仅指自有data stop0、server.wait0；PID13224停止。数据目录/PG_VERSION18/pg_control实际存在、全部DB/schema/log/fixture/证据保留，无DROP/删除/移动；84个新Main schema已在停止前逐名SQL核存在。所有本批新测试命令与Role已实际STOP/Hostclosed，Source/LOCAL/PG窗口closed。旧linker55164仍按Main12:05实际identity+exit2/Wait258保留Host限制，不把本自有PG已join说成全Host进程已join。

[最终Main交付收据](evidence-b13c-r01/main-final-delivery-receipt-r01.json)汇集精确Source19/SSOT哈希、统计/图、测试/评审/格式处置与环境证据。原生具名wrapper正控NOT RUN、完整S4.13/生产rootS4.16长期idle/postexecutor/postproof失败handle与pool/deployment顺序、三worker/实际provider/真实部署E2E/人的接纳仍各自未验，未变窄豁免或S3.6责任；不宣布需求整体/容量通过。下一原批次为B13D身份执行/文章发布历史/恢复工具，未在此收据预认领、预验收或发Source。当前Git不提交/不推送，固定main与原HEAD、staged0及其他人已有变更保留。


最终现场审计已完成，见[最终审计接收](evidence-b13c-r01/final-audit-reception-r01.json)：48个索引/1530个登记payload逐项hash/大小/路径匹配，canonical raw-Go0；当前唯一关闭Article卡、其他73卡正文及原技术/命令/文件/风险/依赖不变，Plan本批外/9批与UTF8 BOM/换行/form-feed保持；15条新增SSOT链接通过，所有当前交接链接另核存在。最终固定main/原HEAD、staged0、3165受保护文件及旧23未变，Code19与已测版本相同，无新增越界。没有为元数据事实变更重复执行已经通过的代码测试。
