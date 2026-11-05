# B13B：Journal 只读与 HTTP 工作许可屏障

最新接收（2026-10-29）：B13B 两原开发卡已按自身条件关闭，任务状态见原 Tasks；三个 HIGH 均由各自原 Reviewer 独立关闭，当前新增/未关闭 finding 为零。[最终现场接收](evidence-b13b-r01/main-final-reception-r01.json)与[证据核查](evidence-b13b-r01/evidence-audit-final-r01.json)记录当前精确对象。下文按发生顺序保留原认领、失败和待验时点，不能把历史 pending 当当前状态。

2026-10-29，沿[已接收B13A](verification-b13a-s4.7-r01.md)进入原B13B。固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，不提交、推送、切分支、部署或清理。用户再次授权采用建议值继续；人的设计签字及真实环境事实不由此补造。

## 真实分工与范围

- Zeno / add-backend-dev，`01a18486-3e11-7852-9723-ad380e0bb392`：原执行者恢复，S4.8的G0发现HMAC格式消费缺口，原11文件尚未开窗；当前仅获[admission两文件有界兼容修复](evidence-b13b-r01/compat-start-r01.json)及其局部验证窗口。
- Pauli / add-backend-dev，`01a1854a-0e79-7f02-abfd-632b91c4fb04`：Main实际spawn，原2文件[G0已接收](evidence-b13b-r01/s83-g0-r01/receipt.md)，已获[精确SOURCE/LOCAL](evidence-b13b-r01/s83-source-local-start-r01.json)，实现与自测中。
- Goodall / add-task-planner，`01a18414-b2ab-7392-9b04-fd974203bf5e`：完成[两原卡真实认领](evidence-b13b-r01/claim-r01/receipt.md)，[Main现场接收](evidence-b13b-r01/claim-main-reception.json)；已STOP并关闭，不授源码、局部运行或关闭权。
- Main：基线、授权、现场差量/验证与最终接收；实际Role产出和Main核验分开。文件归属是协作约定，不冒操作系统硬隔离。

原Task是状态和验收唯一源：[identity S4.8](../../../.add/project/knowledge/tasks/tasks.md#task-identity-auth-s4-8)、[article S8.3](../../../.add/project/knowledge/tasks/tasks.md#task-article-management-s8-3)；[B13B](../../../.add/project/knowledge/plans/ir-01-content-community-batches.md#b13b)消费原安排，不新增任务或第二套状态账。S4.7、identity S4.2/S3.2、article S1.2四个前置均已实核关闭。

Journal源清单仅原11项：journal_snapshot.go、journal_snapshot_test.go、coordinator_context_test.go、journal.go、coordinator.go、journal_test.go、journal_windows_test.go、filelock_flock.go、filelock_unsupported.go、filelock_flock_test.go，以及identity/recovery_run_store_integration_test.go。HTTP仅platform/recovery_guard.go与recovery_guard_test.go。其余runtime/admission及两个测试、root、SQL、迁移、http_security.go和领域Handler保持保护；需清单外消费者时停受影响部分，先给精确依据。

## 成功证据与当前限制

Main留存2041文件基线，原8源码精确前像，5新增路径确认为不存在。初始所有2041文件均未漂移，main/HEAD与空暂存区符合约定。两卡G0须核实际契约和消费者，源码及LOCAL须Main实际信号，不从认领推断开窗。

S4.8须有真正无副作用的reader、retained Begin key/HMAC与constant-time验证、ctx取消/锁/ACKunknown真join，以及fresh Consume/publication和唯一consumed补写窗口。S8.3须在exact router之后、Session之前关闭请求，lease覆盖完整响应、取消不沿旧许可，probe不授业务权。两卡按原命名风险组实际list/普通/race命中；父top、编译或只拒绝负例均不能代替完整证据。开发自测后另作非作者代码及本批机制评审，Main实核同版范围再接收。

项目七项检查仍适用。[写前格式基线](evidence-b13b-r01/baseline-format.json)为根gofmt FAIL exit2，精确旧23文件哈希及两raw与前次写前相同。旧S4.7有限例外不自动扩本两卡；保留旧证据，后续据真实新版本结果分别处置，不报7/7PASS。

本地执行沿OS/path allowlist，GOENV/GOPROXY/GOSUMDB关闭、GOTOOLCHAIN本地、GOFLAGS与测试DSN为空。当前未向作者开放LOCAL；后续只准合成unit/contract fixture和测试binary子进程。S4.8原命令 `go vet -tags=integration ./internal/identity` 仅可在明确LOCAL后作编译/静态检查，不运行integration测试、真实PG/DDL、服务/worker/restore或读取真实secret ENV/DSN/key。原具名豁免、两S4.4、Case18、S4.5/S4.6和真实E2E门槛保持。

## G0兼容缺口与有界补充修复

[S4.8 G0](evidence-b13b-r01/s48-g0-r01/receipt.md)是静态证据，尚非执行FAIL或独立finding。Main实核keyring.go:173–180与Journal Begin/computeRecordHMAC原码，真实fingerprint及recordMAC均为`hmac-sha256:`加64位小写hex；admission.go:239/452却对三个HMAC字段使用SHA-only正则。纯reader必须返回原Begin，改Journal前缀不能作为解决方案。

Main按用户standing授权采用最小局部兼容修正：原Zeno仅改admission.go与admission_test.go，保留原HMAC算法/wire及真正SHA字段严格规则；先以真实现有算法函数和显式合成key建立编译可达的行为Red，再修专用字段校验及旧误用SHA的fixture，保留原230用例并加专用正负例。同原Ptolemy独立复审后核解除S4.8此具体阻断，再开启原11文件。属于当前交付的已确认格式修正，未新增业务/设计/公共API/TaskID；既有Task与B13A旧SHA证据保留历史，不把旧报告当新对象复审。

两卡认领的checkbox由原Task-Planner实际登记，状态仍以Tasks为准；认领与G0、源码窗口、测试、独立Review分别核证。Pauli G0原文“用户指定/用户告知”来自Main context pack；真实Role/ID来源是Main Host spawn及转交，并非用户亲指定人员或亲验2041基线。原raw保留，Main信号已明确此归属。

当前尚未宣称本批测试、Review或任务通过。新临时证据复制留存，源码副本使用`.frozen`；清理、移动、删除或归档均未授权。

## 实现冻结与独立评审交接

Zeno仅完成[兼容两源交付](evidence-b13b-r01/compat-author-r01/delivery.json)后STOP并由Host关闭，Journal原11项仍未开窗。Main全文核两源差量、G0与交付，实际解析作者普通/race各271叶，原230叶保留、新增41叶、九个所核前缀非零，零required skip/fail；Main在这两个新SHA上另跑普通/race亦各271叶通过。原Ptolemy / add-reviewer（`01a18492-af9e-7ea2-9cd5-e4dd368d7bee`）已实际恢复，独立评审此新对象；旧B13A finding关闭链不重开，旧报告不冒新对象结论。

Pauli完成[HTTP两源交付](evidence-b13b-r01/s83-author-r01/receipt.md)后STOP并由Host关闭；Main全文核两源及交付、现场与冻结SHA相同，实际解析作者四组普通/race各56叶。作者证据先于兼容修订，Main已在兼容稳定版另跑HTTP普通/race各56叶全通过。Curie / add-reviewer（`01a18579-fde7-7a12-a3b1-9017a86b98c3`）由Main实际spawn，独立核两源及真实消费者；仅自有Temp评审证据与合成定向LOCAL，无生产源码/任务状态写权。

上述是交付、现场核验与真实委派事实，两个独立结论尚待返回，Task保持执行中。Main实核2041基线，允许差量仅admission两源、HTTP新增两源和两份Task/Plan的已授权记录；其他2037基线文件未变、main/HEAD不变、暂存区空。全仓七项检查仍待本批全部相关源码稳定后联合执行；本地组件通过不外推PG、native POSIX、root/Case18、部署或E2E。

## HMAC阻断接收与Journal实施开始

[Ptolemy兼容独立报告](evidence-b13b-r01/compat-review-r01/review-report.md)对两源组件PASS、新增0，实际独立普通/race各58叶；Main全文核receipt、report、probes及实际raw，现场两SHA和Journal写前SHA一致后完成[该具体缺口接收](evidence-b13b-r01/compat-main-reception-r01.json)。这是新对象独立结论，非旧B13A报告继承；没有Task/batch/全仓格式例外或真实环境结论。

Main已向原Zeno实际发送[原11文件SOURCE/LOCAL START](evidence-b13b-r01/s48-source-local-start-r01.json)，原G0同版输入复用；新增admission测试对私有validateRecords的调用纳入精确消费者delta，须保留纯校验合同，无法在11项完成时仅停受影响部分给证据，不添加Background兼容壳或默改冻结两源。只准合成定向验证；integration标签只准原约定vet静态检查，绝不运行真实数据库测试。HTTP两源继续冻结、Curie独立评审中；已告知其Journal并行变动属于Main授权peer，过渡编译结果不得伪装HTTP缺陷。

## HTTP组件独立评审返回

[Curie完整独立报告](evidence-b13b-r01/s83-review-r01/report.md)对固定HTTP两源有界PASS、新增0，原四组普通/race各56叶、独立探针各6叶、platform回归各259叶全通过。Main全文读report/receipt/probes，现场核HTTP2、稳定Runtime与compat2 SHA，实际解析Reviewer原56叶及最终6探针普通/race零required skip/fail，完成[组件评审核收](evidence-b13b-r01/s83-review-main-reception-r01.json)。Host已关闭STOP的Curie，完整证据复制留存；raw中“Main latest user message”指Main工具交接消息，非用户亲验或签字。执行发生于Journal迁移前，迁移后联合验证和全仓七项仍待；Task未关闭。

## Journal实施中的Lead核对

原Zeno只读[新增私有消费者delta](evidence-b13b-r01/s48-g0-delta-r01/delta.md)确认保留pure validateRecords签名、keyring-only可用性，无第12源；其中closed/pending是其取得START前的协调视图，当前实际授权以Main已发的11源START为准。后续实现临时包为 `han-b13b-s48-author-fxgujh6i`，尚在写入，不能当冻结交付。

Main已全文读中间reader/coordinator、两个新增测试及Journal/lock原位迁移差量，临时版本与SHA各自保留在Main Temp `han-b13b-lead-_oq3rky1/s48-main-midread*`，不冒最终同版审查或Role独立评审。实际中间Red为3个可编译行为FAIL；Ensure确认未知另有2个行为FAIL→2个Green。当前某次组合race实际reader30、Coordinator26、原Runtime271叶全PASS无skip/fail；后续源仍在变动，最终命令和冻结版本尚未接收。integration标签只执行vet，另有固定linux/amd64/CGOoff静态vet，不是native POSIX执行。

Main指出的Ensure提交但ACK error须独立回读已由原作者在当前窗口处理；文件Create/Chmod/Write/Sync/Rename之间的取消检查和真实join仍交原作者完成最终核对。两者是Lead实施中差量核对，不是正式非作者finding。Main再次实核2041基线，仅原11源、冻结compat2/HTTP2及已授权metadata出现允许差量；其余2029受保护基线文件未变，四冻结源精确SHA保持，固定main/HEAD、空暂存区。

Main已实际恢复原Curie（同ID）作B13B跨任务整体Review准备：仅固定r52/原两Task/B13B合同与已冻结证据，无活动Journal读取、无LOCAL、无最终codefinding或batch结论；等待Main交冻结11源、单卡Review及全仓实测后再正式评审。单卡代码Review和跨任务机制Review分开，不从局部PASS直接推导整批通过。

## Journal冻结与同版联合检查

原Zeno交[最终11源回执](evidence-b13b-r01/s48-author-r01/receipt.md)后STOP并由Host关闭。Main现场11live=authorfrozen、原8前像=baseline、3新增路径符合授权，记录[冻结核对](evidence-b13b-r01/s48-author-freeze-main-r01.json)。Main已全文核作者回执、11最终差量（以先前完整mid快照/迁移diff加最终remaining delta覆盖）、旧测试调用迁移及集成文件仅3处ctx/显式budget，完整证据复制留存。作者普通/race reader39、Coordinator42、Runtime271叶全部经Main实际JSON解析，原271叶名完整保持。原Ptolemy已实际恢复评审本11源；这与其compat两源报告是不同对象，单卡结论尚未返回。

Main[联合实测证据](evidence-b13b-r01/main-joint-evidence-r01/index.json)在全部15源冻结后取得：七项实际退出为2/0/0/0/0/0/0，六项通过；全15源单独fmt空输出通过。另count1普通/race四top各408叶（39＋42＋271＋56）全部通过、逐叶与各作者当前对象一致。Rootfmt失败的两raw与写前基线逐字节相等，精确旧23文件哈希未变；Main依用户standing采用建议授权作[当前精确对象有限格式处置](evidence-b13b-r01/format-disposition-main-r01.json)，适用当版compat/S4.8/S8.3/B13B软件门槛，不继承旧S4.7、不改FAIL为PASS、不豁免代码或真实环境。源变化需重新核适用性。

Curie已完成[合同准备清单](evidence-b13b-r01/batch-review-prep-r01/prep.md)，Main全文核后留存并关闭。为并行推进，Main随后又实际恢复同Curie，现已给冻结13主对象＋compat依赖及联合实测、隔离测试LOCAL，开始跨卡机制检查；最终batch结论仍须等待原Ptolemy单卡结论和任何修复闭环，不把尚未返回当通过。两位非作者只准各自Temp证据/测试overlay，所有生产源仍冻结。两原Task仍[-]，Task/batch最终接收尚未作出；PG/nativePOSIX/root/Case18/E2E事实不由组件验证补造。

## 单卡R01阻断与四源修复

[Ptolemy S4.8完整单卡R01](evidence-b13b-r01/s48-review-r01/review-report.md)为CHANGES REQUIRED，唯一HIGH `RECOVERY-B13B-S48-CR-R01-F001`仍OPEN：writer仅operation前后验锁，失锁后仍写396bytes、仍第二CAS。Main全文核报告/探针、receipt身份/结论/对象及实际raw；独立普通/race各66叶64PASS/2行为FAIL，含组件各89叶87PASS/2FAIL，零skip/race告警，见[失败现场核收](evidence-b13b-r01/s48-review-main-r01.json)。58兼容探针、reader39、Coordinator42在新源通过也不关闭该HIGH；Main原全仓/408结果是R01当时范围，格式处置不豁免代码缺陷。

Curie按Main信号在安全节点[partial R01交回](evidence-b13b-r01/batch-review-r01-partial/review-report.md)：第一轮差量已读、第二轮未完成，没有自己的新finding或测试执行，没有Go/后台/待join；CHANGES_REQUIRED来自原Ptolemy HIGH，不冒本人独立复现或全面0缺陷。两Reviewer均实际STOP并Host关闭，原15源现场仍完全匹配后，Main读/接受[原Zeno四源修复准备](evidence-b13b-r01/s48-r02-prep-r01/repair-plan.md)并发[精确四源SOURCE/LOCAL](evidence-b13b-r01/s48-r02-source-local-start.json)：仅journal.go、journal_snapshot_test.go、journal_test.go、filelock_flock_test.go。其余11冻结、原Task仍11文件不扩清单；scope guard增加独立fix窗口，未重开全11写窗。修后由原Ptolemy复审，再由Curie补剩余/新delta，源变更须新全仓/格式适用性，不挪用旧SHA结果。

Main还实核原规划报告只覆盖r52四卡四批，其中含S4.8但不含S8.3；后者原卡待PlanningReview注记不能用代码PASS覆盖。已实际恢复原Kuhn / add-reviewer（`01a1835a-8010-7a73-a8dd-3192719b6965`）只读核S8.3及B13B相应映射，S4.8原规划结论复用，不重做六稿/14卡或代码Review，无共享/源码/LOCAL权限。Task/Plan只追加这些真实受理事实，checkbox/73及49/5/19/原技术清单/依赖/命令不变。两卡和本批最终接收继续pending。

## S8.3规划覆盖已核清

[Kuhn当前有界规划报告](evidence-b13b-r01/s83-planning-review-r01/review-report.md)PASS/新增0。其找到原Planning R01:46对S8.3的明确覆盖，R02:3沿用未受影响范围，当前原技术正文与原精确前像相同；因此卡上的旧待审措辞已过时。r52四卡报告自身确不覆盖S8.3，但不能据此推断此前从未评审。Main全文读当前报告/receipt、核六文档SHA、另实读两历史精确行并核报告摘要，记录[现场规划接收](evidence-b13b-r01/s83-planning-main-r01.json)。本轮当前对象另作两轮审查，未把旧六稿或代码PASS替代当前结论；原证据与Main当时待核状态均保留，Task/Plan追加正确范围。

Kuhn实际STOP/frozen并Host关闭，无代码/Go/共享写。无规划gap需要Goodall返工，原任务/依赖/最低层不改。原作者四源修复继续，S4.8代码HIGH不由规划结论关闭；Task和批次最终接收仍待真实修复/独立复审及对应版本证据。

## 单卡R02复审与异常退出修复

- **Main代码R02接收及两源返工（2026-10-29）**：[原Ptolemy复审](evidence-b13b-r01/s48-review-r02/review-report.md)已独立确认R01失锁F001 CLOSED；新增HIGH `RECOVERY-B13B-S48-CR-R02-F001` OPEN：Anchor及retirement真实端口panic时跳过operation失效与锁释放，普通/race各2行为FAIL，原panic保持但Close=0、逃逸ctx仍可用。Main全文核报告/探针及实际raw，见[现场核收](evidence-b13b-r01/s48-review-main-r02.json)。R02当版[联合实测](evidence-b13b-r01/main-joint-evidence-r02/index.json)四top普通/race各423叶PASS、全仓六项PASS、根fmt仍FAIL2且另有有限处置，不关闭代码HIGH。Main已向原Zeno发[精确两源R03 SOURCE/LOCAL](evidence-b13b-r01/s48-r03-source-local-start.json)，只改原11内journal.go/journal_snapshot_test.go，其他13源冻结；修复后仍由原Ptolemy复审、Curie补批次剩余评审，并对新SHA重核适用检查。两Task保持[-]、73/49/5/19不变，尚无本卡或批次最终接收。

## R03冻结与新对象联合验证

原Zeno交[R03两源回执](evidence-b13b-r01/s48-author-r03/receipt.md)后STOP并由Host关闭。Main全文核最小两源差量，实际核前后/live/frozen及其余13源不变，形成[当前原11源组合与两源冻结](evidence-b13b-r01/s48-r03-author-freeze-main.json)。生产变更在绑定、检查或端口调用之前注册defer，退出时先使operation失效再释放锁；保留原panic传播及正常错误聚合。Main实际解析作者普通/race各372叶（59/42/271）、旧367逐名保留；作者自测不关闭其HIGH。

Main另在全15冻结SHA上完成[R03联合实测](evidence-b13b-r01/main-joint-evidence-r03/index.json)：四top count1普通/race各428叶（59/42/271/56）全PASS、原叶保留；全仓七项退出2/0/0/0/0/0/0，15源fmt空输出PASS。根fmt两raw和旧23哈希仍与写前完全一致；[本版精确有限格式处置](evidence-b13b-r01/format-disposition-main-r03.json)仅适用当前15SHA，不继承旧版、不把FAIL改PASS、不豁免代码或真实环境。

原Ptolemy实际恢复同ID进行R03定点独立复审；原Curie实际恢复同ID补批次Round2、新差量及最小跨卡合成探针。所有生产源码保持冻结，二者仅自有Temp证据与隔离测试LOCAL；最终结论、Main接收及两Task关闭尚待真实返回。

## 跨卡R02纯读异常退出阻断

- **Main批次R02阻断与R04返工（2026-10-29）**：原Ptolemy R03已独立关闭writer两项HIGH、各90叶普通/race全PASS，Main[现场核收](evidence-b13b-r01/s48-review-main-r03.json)。[Curie完整跨卡R02](evidence-b13b-r01/batch-review-r02/review-report.md)新增HIGH `B13B-BATCH-CR-R02-F001` OPEN：纯reader withLock在实际Anchor.Load panic后跳过Close，Snapshot/VerifyBeginSecret各calls1/Close0；每模式自有9叶7PASS/2FAIL，非Root/SQL/E2E缺失。Main全文核报告/探针和实际raw，见[批次现场核收](evidence-b13b-r01/batch-review-main-r02.json)。所有15源匹配且Reviewer STOP/Hostclosed后，Main向原Zeno发[仅原reader两源R04 SOURCE/LOCAL](evidence-b13b-r01/s48-r04-source-local-start.json)，其余13源冻结；修后由发现者原Curie复审，不重开未受影响writer关闭链。原R03全仓六项/428叶及有限fmt处置不关闭此新HIGH、不自动覆盖新SHA。两卡仍[-]，73/49/5/19不变，无最终Task/batch接收。

## 最终有界接收

原Zeno完成[R04两源交付](evidence-b13b-r01/s48-author-r04/receipt.md)后STOP并由Host关闭；Main全文核最小差量与实际前后/live/frozen，原11源清单不扩、其余13冻结源不变，[最终组合](evidence-b13b-r01/s48-r04-author-freeze-main.json)已留存。纯读有效lease取得后立即defer Close，两个公共入口的端口panic保留原对象并释放一次，正常错误和Close根因仍聚合、失败不暴露proof。作者390普通/race及旧8跨卡回归是作者证据；原失败raw保持原样。

[原Curie批次R03](evidence-b13b-r01/batch-review-r03/review-report.md)对当前assembled R04独立PASS：本人的普通/race各31叶全PASS（原8恢复＋新22生命周期＋HTTP1），原两panic叶未放宽，Close0变Close1；新HIGH CLOSED、原writer两HIGH保持CLOSED、无新增或未关闭finding。Main全文核报告/receipt、新生命周期和原cross/HTTP探针，实际解析四raw，17对象SHA与入口/出口/live相同，见[批次现场核收](evidence-b13b-r01/batch-review-main-r03.json)。原角色的真实ID为Main Host派发/恢复的Curie `01a18579-fde7-7a12-a3b1-9017a86b98c3`；receipt里的user-designated措辞不推定用户亲指定或人的签字。

Main当前[R04联合实测](evidence-b13b-r01/main-joint-evidence-r04/index.json)count1普通/race各446叶（77/42/271/56）全PASS、原428叶逐名保留，全仓六项PASS、15源fmt空输出PASS。根fmt仍FAIL2，旧23哈希及两raw同写前；[当前精确有限格式处置](evidence-b13b-r01/format-disposition-main-r04.json)不是7/7PASS、不是代码或真实验证豁免。此次源码稳定后没有为元数据回填重复运行Go测试。

Main按原开发卡完成条件关闭S4.8与S8.3；仅两checkbox改变，其他71任务状态与原ID/文件清单/requires/DAG保持。当前统计见原Tasks为73/51 completed/3 active/19 todo，仍进行中的原专项为article S1.5/S4.4/S8.2。Main实际核固定main/HEAD、staged0、基线保护、实际差量/精确原requires及接收；所有专业执行者已STOP并Host关闭。回填中的三处旧测试总数已改为446，初次状态输出解析诊断及修正文档记录单独保留，不把它们算测试失败或通过。

本次仅软件组件U/C＋race及静态迁移/契约核证，不证明真实PG/SQL/provider/nativePOSIX/唯一production root/Case18/worker/恢复环境/部署/E2E或生产规模性能；设计人的精确接纳仍unknown，不补签。下一原批次是B13C（Store元数据/文章锚定），本接收不自动启动源码、数据库或进程窗口；依已有持续授权按原卡前置继续核对。全部历史失败、旧证据和未验边界保留，无提交/push/部署/清理。
