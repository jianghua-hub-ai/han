# S2.4 follow-up r03 测试安排delta独立评审

**Design delta Gate：CHANGES REQUIRED（1项MEDIUM，时钟与旧资源账接缝未具体闭合）。** L/N/W映射、145/150/53分钟算术、旧批次永久NRU与累计资源保护原则已核自洽；不能将这些局部结论签成整份可执行安排PASS。P0仍conditional，不批准任何LOCAL/PG或完整性能启动，不核销原12项。

## 固定对象、身份与两轮范围

objectType=`design`，对象是 `test-plan-followup-r03.md`，实际21064 bytes，SHA256 **7B0553AEBBCD998A4BAB70B1B92CA87DA0398058191F31CCFEDEA40FEAF1C106**。起审与收口均独立复算同一摘要。TestEngineer作者已STOP；本Reviewer为原非作者独立评审者，沿原Workflow继续，无Route、无委派、无代修。工作区固定 `D:/AID/VSCode/Codex/Han`、main / HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca（用户固定依据，不执行Git核查）。

自身Reviewer Role、add-code-review全文及相关规则沿有效已读正文；code-review-checklist、abstraction-principles按design对象适用，设计写作/决策规则只应用于当前安排与接缝。没有把源码编译/执行要求强套到非代码计划，也不加载无关库或重新全量源码审查。当前用户已授权建议值继续推进，无需再问同一人工审批；Lead实际采用仍是后续动作，独立审查不能代Lead选择修复方案。

Round 1实际读取：候选全文、test-plan-execution-addendum-r02全文、Lead execution-policy-r02-adoption全文、原code-review-r03-evidence-r01/report.md全文；核12项集合、L/N/W各自证据及P0～P6顺序、Tnew定义、145/150/53门、预算与保留旧履历。原报告SHA仍78A109D77EB2BFA55420EDAF6C6E6E1B0AEA6345B2A178888D42E6F86478D94F。

Round 2实际读取：固定freeze-r02 H的policy/window、baseline创建/加载、建schema前guard、leak入口/关闭资源、9分支RealSemantics、校准及small/medium顶层；W/R/N只按所列12入口的注册和直接消费者定向读取，未重开业务库或源码缺陷全审。五snapshot与对应live五文件摘要自行核一致，沿用正式H B8D11…/W38ADA…/RF25D…/N3E531…/DD476…，具体值保存在offline-evidence.json。

另只读消费 `probe-diagnosis-r01/` 已完成sql-size-1～5、catalog、filesystem、final共8份JSON，核其相关时长/退出/快照及SHA；未运行这些作者工具或SQL，未把尚待交回的根因最终报告/角色STOP当已收到。Feynman仍持LOCAL/PG，Reviewer不取用该窗口。

唯一写集为本 `test-plan-followup-review-r03/` 的report/verdict及离线核证输出。未Go/PG、未读凭据、未Git写、未清理、未改作者文/旧报告/Tasks/Plan或源码。

## Finding

### FUP-R03-001 — MEDIUM：新attempt时钟与旧baseline的Window校验存在确定冲突，当前安排尚未给出具体绑定合同

- 对象位置：候选 **第97行**（§4现有配置接缝）及 **第105行**（§5沿用原持久ledger）；P1/P2执行顺序见第64～66行。直接消费者是freeze H:1451、H:3011、H:3023、H:3038。
- 现状：候选将新policy.window_start与环境WINDOW_START绑定Tnew，同时要求直接沿用原ledger和全部原档位baseline。H:3023创建待比较record.Window取同一WINDOW_START；H:3038加载旧文件后要求prior.Window逐字符串等于record.Window。现场唯一既有small.json记录Window=`2026-11-04T03:29:04.586115+00:00`、Bytes=1406932671。H:1451～1453在CREATE之前加载并校验它。
- 触发：P0完成后开始独立新attempt，WINDOW_START=Tnew且Tnew不同于旧11:29起点，baseline目录仍指原目录。新policy与新WINDOW_START虽可通过H:1311，但必然被H:3038的旧Window校验拒绝，报persistent tier baseline belongs to another window or tier，P2的leak业务仍无法到达。把policy里的父批次/ledger摘要加齐也不会改变这个消费者条件。
- 影响：候选描述的“仅绑定新policy+旧ledger”在当前五hash下不能直接执行。执行者若临时换空目录/重新按当前库大小建账，会违反本候选不重置累计资源的核心合同；若改回旧WINDOW_START，则只能取得已过期的旧窗口。当前源码是安全拒绝，不是新发现的源码HIGH，也没有证据表明已经发生了重置或越权。
- 违反依据：设计必须沿真实消费者给出可执行的字段/状态/兼容及前置合同；本次明确要求新时钟与旧全档资源账同时成立。候选第97行的“若现有消费者不能安全表达则返回Backend/Reviewer、不运行”是正确的停止保护，但这里已证实不能直连，不宜继续作为假设性未决情况留给运行时首次失败。
- 最小修正方向：返回原TestEngineer，把它列为**Tnew/P1之前必闭合的确定接缝前置**，明确独立attempt时钟与资源ledger身份如何分别传入、所有旧档位Bytes/来源SHA/旧Window如何保持，以及对应执行入口、责任人和有限验收。若拟用追加适配，须具体说明其输入/输出与累计不变证明；若必须改消费者，明确小范围delta、newfreeze及原Reviewer复审门。选哪条由Lead/真实作者决定，本Reviewer不选实现、不改源。
- 关闭条件：提交固定修订或明确追加合同，能够逐项对上H policy/window与baseline加载条件，证明新Tnew不会落回旧截止、small/已有medium等Bytes不重置、旧文件原字节保留、首次未登记medium仍只原子登记一次；列出同一边界的拒绝分支及资源核证要求。**计划复审可用字段/记录与静态消费者映射闭合，不要求为非代码计划先跑Go/PG。** 如采用后需要真正实现适配/消费者修订，执行前另有相应实现/版本门，不能用计划PASS替代。

本finding不要求重开原18项源码审查，不要求重新审批用户已授权的新安排数值；也不授权执行者绕过Window校验。返回原规划作者形成具体delta后有界复核即可。

## 12项L/N/W逐项核对

L=实际局部负例；N=真实PG正常语义；W=正式活跃窗口。原6项CLOSED与E001关闭沿权威报告保留，本轮没有任何新核销。

| 原pending | 候选分类 | 实际直接消费者与判断 |
|---|---|---|
| F002 | W | H:2998/3653→统一runner；当前core/expired-helper/fail-stop证据直接引用合理。必须真实完整TIMED及逐phase记录，不能以一合规点代全套；不先要求F002全CLOSED才准W。 |
| F005 | L | H:3210～3223真实newHarness→beginOwned→SELECT→故意保留tx，H:1083资源关闭及泄漏guard；仅exit1/前置size超时不足。已有效CanceledWorkers证据不重复，先L实到达并核收口再N/W的顺序正确。受时钟/ledger接缝finding及P0阻塞。 |
| F006 | W | W:110～112注册medium/W02-c1与c16，W:298→TIMED runner。现有固定操作/rune-byte诊断不重复；正式每repeat两桶raw关联仍必须取证。 |
| F007 | W | W:311～312 c16的WarmupConcurrency=4；需三次真实10s c4 warmup，103787800ns诊断不替代，不新增独立点/P95/schema，合理。 |
| F009 | N+W | H:3292～3293初始drain1000/10000，H:3317～3340核accepted/sent/terminal；N:96～114注册正式N03-1k/10k，N:732为SINGLE。保留初始整批与追加工作量，N不替代正式SINGLE的10/30、<=60s。 |
| F011 | N | H:3274→W:417真实E02双竞争，正式medium/E02仍在W:116。先核真实赢家/版本/审计/锁依赖，原candidate或顶层exit不能代替当前子项。 |
| F012 | N | H:3275→N:278真实D02竞争，N:66注册正式D02；T1/UPDATE唯一赢家与相应operation终态仍在，不以人工holder代两业务参与者。 |
| F013 | N+W | H:3263三个新T1及三operation→N:239，N:63正式D01；当前前置probe失败不等于到达。N先证明新ID首次T1，W再证明实际新T1活跃计数，未降为幂等读取。 |
| F014 | N+W | H:3276→R:455跨页helper，R:689正式R05；page1后变化和独立SQL oracle保留，expired-helper不替正常续页/恢复。 |
| F015 | N | H:3277→N:888两轮sameWorker false/true→late result；N:116正式N04。fake mailer/attempt+lease fence/旧结果不能覆盖新accepted，未引入真实SMTP。 |
| F016 | N | H:3283→N的pgPerfLockChain；N:121注册正式X08。三PID C真实排队/等待图、正常释放join保留，通用取消成功不替三PID图或仅55P03判断。 |
| F017 | N+W | H:3284→H:2084三份生产SQL EXPLAIN ANALYZE BUFFERS，R:683～693正式读点。核真实plan内容/参数，而非数文件；三minimal plan不替正式读点计划/扫描证据。 |

集合核验：12项无遗漏/重复；按主路径是1个L、4个仅N、3个仅W、4个N+W。P3顶层锚定9分支，late-notice-results内有两worker变体，1000/10000为两个独立分支；不是10或11个顶层场景。

P4按整个表解释：F002/F006/F007以及F009/F013/F014/F017的W部分可以继续PENDING，N/L必须按各自合同先被独立接收。候选“余项仅为预先列明W条件”的总门覆盖这7项，不能误读为只允许P4举例的4项pending。依此解释不会造成“先全关闭才能取正式证据、无正式证据又不让开跑”的循环，也不能以该解释放过F005安全证据或N分支失败。

## 新旧时间、预算及资源核证

旧批次11:29:04.586115→13:54:04.586115停新操作/13:59:04.586115硬停，以及旧完整启动门12:32已失，永久NRU。候选明确区分独立新attempt、父批次与新结果，无追溯洗旧履历；用户已授权新建议安排，不把旧execution-r02解释成自动追加150min。实际采用/绝对Tnew尚不存在，不编造开跑时刻。

候选145min正常费用=5预检+25局部+10局部Review+13灵活+72全量+20执行Review；硬停尾段5min使总上限150。完整启动门145−72−20=53min，启动时至少剩92min，算术正确。25/72/20均是估算或预留，不能替代实际耗时/证据深度；估算上升、局部未完或容量不足就拒绝完整启动，不压缩采样或延迟截止。所有失败/等待/复审均计Tnew，P0只读诊断单列、P1起包含所有新attempt前置，不允许先建schema再补Tnew。

TIMED名义120s=3×(10+30)，全部phase累计非活跃tail<=1s且wall<=121；SINGLE仍<=60且包含tail，warmup/measurement下限不降；unknown不成功，真实错误、截断、资源不归还均FAIL。N500软门及near-limit/完整批/独立oracle保留。new baseline/window绑定问题未闭合前，不能说这些时钟已被真实消费者正确执行。

资源原则符合现有合同：全局base **1406932671 bytes**，small1GiB/medium5.5GiB/large8.5GiB/total15GiB；旧schema/失败半成品/校准/局部诊断均累计，不清理、不换库、不降fixture。现场baseline目录目前只有small.json；以后已有medium/large文件须沿用，不能为新attempt重新登记，确实未开始的medium可仅首次原子登记。这里的“目前只有small”是只读文件证据，不是未来无medium义务或允许空目录重建。

历史DB1992988351−base1406932671=586055680；small剩余1073741824−586055680=**487686144 bytes = 465.09375MiB**，候选约465.09MiB正确。它是历史累计余额，不是当前测量，更不是完整small装得下的保证。P0/P1须刷新相同资源范围、真实PGDATA余量及新L/N/校准/small预估；读不到或装不下即NRU，Review不为容量背书。最终大/canonical仍不调用，K按实际调用另计，41逻辑/42单元义务未被诊断替换。

## P0诊断事实与限制

八份已完成诊断artifact的独立离线摘要已保存。五次独立psql的wall为7.359/4.204/4.453/5.016/4.265秒；SQL自身timing分别6955.451/3905.985/3990.506/4567.028/3825.867ms，不能把含启动/观察的5.016秒误写成SQL已超过5秒。前3次诊断deadline20s，后2次5000ms，全部wait0；这些是诊断配置，不是harness生产5s门被放宽。活动采样未观察到blockers，不能据此排除所有等待或认定无锁根因。

catalog记录86 schemas、82 perf schemas、11067 relations、active_others0；filesystem同一路径11137 entries，枚举31ms、cached_entry_stat47ms、explicit_path_stat5.891s，后两者byte合计均1992988351。该差异支持文件元数据访问路径是调查方向，**不足以签署唯一根因、稳定<=5s或某种替代probe方案安全**；PG进程级取证有error5限制，不推断CPU/锁/IO的唯一原因。

final.json结束06:36:19.037173Z，保存诊断activity为空、列出的owned psql均wait0，前后五hash一致；这不代表作者最终根因报告或LOCAL/PG角色释放已经交回，也不是本Reviewer实时查询。现有SQL自身最高6.955s确实会与当前5s预算冲突，少量较快样本不能解除P0。候选继续FAIL/UNKNOWN、不得盲调5s/跳guard，正确。P0保持conditional不构成新增finding；本轮不选探针修复、不要求跑更多PG、不将只读诊断改变为含schema的验证。

## 交回与停止

需返回原TestEngineer处理FUP-R03-001的具体合同，Lead据已有用户授权选择方案和采用范围；无需再次询问同一建议值授权。修订后只核新时钟/旧ledger接缝delta、必要直接消费者与未决前置，不重开旧r04/18项全量源码审查。

本次已确认L/N/W映射与数值/不降合同的安排逻辑，但**整份计划delta尚不能PASS或直接执行**。即使后续计划PASS，也仅为计划delta，不关闭12项、不能替代源码修订Review/真实全量calibration-small-medium/独立执行Review及Lead接收。原6 CLOSED、12 pending、E001 CLOSED、旧批次完整NRU与任务active均不因本报告改变。

所有候选/旧报告/源码/资源ledger只读。独立评审完成后 **STOP**，不运行Go/PG、不等待或占用Feynman窗口。
