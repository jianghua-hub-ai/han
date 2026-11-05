# S2.4 资源采集预算合同r01独立design评审

**Gate：CHANGES REQUIRED。1项MEDIUM：20s完成间隔与变化识别上界混用，缺少相应可执行的新鲜度期限。** 另有两项新增更强合同待明确采用范围：每批物理增长上界，以及harness observer逐phase收口。后两者不是原合同必需，也不因实施更复杂就单独判成Bug。没有新源码PASS、运行许可、原12项核销或性能PASS。

## 固定对象和两轮范围

objectType=`design`；对象 `test-plan-probe-budget-r01.md`，实际14802 bytes，SHA256 **22F490EBEA79ADE0E6C8D459CE88781C3A19750461A04CC7B2230AF71BD69112**。起审/收口复算固定同版。TestEngineer已STOP；本Reviewer为原非作者，沿原Workflow，固定 `D:/AID/VSCode/Codex/Han` main，不执行Git或委派。

自身Reviewer Role、add-code-review及适用规则沿有效已读全文；本次按非代码合同与真实消费者核，不要求计划先build，也不替Lead挑参数/采用或替作者改文。用户已授权建议值继续，未新增同项人工审批请求。

Round 1：候选全文、`author/probe-fix-r03/report.md`、`open-handle-growth/stdout.log`全文及已完成result，核固定摘要、真实反例及15/5/20合同语义。定向补读原已采用 `test-plan-addendum.md:40` 的完整上下文、execution-addendum-r02的point边界，区分旧合同与新增约束。旧r04 clock计划/Review沿有效结论，不重开其已关闭planning finding。

Round 2：仅沿旧freeze-r02 H的checkDiskBudget、newHarness、startDiskMonitor、closeResources、seed及直接size统计消费者定向阅读：H:1318～1376、1435～1556、1567～1587、1083～1128、2166～2410、2578～2581、2835/2839/2907/2910。核互斥/Acquire/SQL/Scan、monitor owner、seed事务/CopyFrom/ANALYZE、取消与point计时之间的接缝；未全库源码重审，未读正在写的live H正文代替freeze。

映射用旧H为131747B / B8D11EBFB3E4D15F01BC15263E1899BF168732CB06B232099B0F1FD9759E8E86。只读hash观察时live H已变为140028B / 65BF8E9C3117466A2D93CDF9286F99EEFE2B5E3D8A9DCA5D6D8B502E811E09AB；这是在途clock/resource准备的版本观察，**没有阅读/审查或接收这份新H**。本报告不把旧H缺口当新H仍存在的结论，后续必须固定新freeze再审实现。

唯一写集为本独立目录的report/verdict及离线核证结果。未Go/PG/SQL、未读凭据、未Git写、未清理、未改源码/候选/旧文档/Tasks/Plan，未运行作者实验。

## Finding

### PROBE-BUDGET-R01-001 — MEDIUM：成功结果间隔不能推出20s变化识别，且新鲜度到期未进入有效deadline合同

- 位置：候选第27行的“变化被下一次完成采集识别”及5+15=20s；第35行把它改称成功结果间包络；第52行有效deadline列表；第75行再次以最坏完成识别包络10→20描述覆盖。
- 现状：15s总采集、5s tick、单飞及跳过tick可以限制在途采集与某些完成间隔，但SQL扫描期间并非瞬间统一读全部文件。候选允许已在途有限写入继续完成，只禁止新的写入/批准入；其“非原子快照”说明并未定义变化识别期限、样本有效期锚点，或者将更早freshness deadline加入采集/写入准入与取消边界。
- 具体触发：以同一measurement内部相对时刻为例，第一次资源SQL在t=0开始，已于t=0读过目标文件，t=14.9完成；一个采集前已获准的有限在途写者在t=0.1使该文件增长。t=5/10的tick不并发启动采集；第二次在t=15开始、t=29.9完成并报告增长。两次各14.9s，结果完成间隔15s，均符合15s预算/5s节拍/单飞/<=20s结果间隔；phase可持续到t=30，无调度停顿、无新写者在采集中准入、也无需超cap。然而变化到被结果识别为29.8s，超过20s。
- 影响：若实现按last-success completion更新时间或只校验两次完成间隔，上述执行不会触发“包络未兑现”的停止，却被文字声明成20s变化识别保障。代价也不能据此精确宣称仅增加10s风险暴露。“不是硬实时/不是原子快照”的免责声明不能让两个不等价的时限成为同一合同。
- 最小修订方向：返回原TestEngineer，按Main本轮已给的倾向，明确20s是**单调时钟强制的新鲜度/停止上限，不是变化识别保证**。年龄从上次完整成功采集的开始时刻（保守最早观察时刻）计算，不从finish虚刷新；15s仅为单次最大机会，5s tick保留。本报告记录这个聚焦修订方向，不把它当固定r01已被修改、政策已采用或实现已通过。
- 具体关闭条件：① 定义 `freshness_deadline=last_complete_success.probe_started_mono+20s`，保留UTC取证但不用墙钟跳变刷新年龄；尚无成功值时，首DDL/新写入保持关闭，仅在明确前置内取得完整成功值，不能以0/旧cache启动。② 新采集的单飞等待/Acquire/SQL/Scan deadline同受15s、既有freshness截止及适用更早parent截止限制；新采集尚未成功不能延长旧期限，过期即取消/停止后续准入，不等下一次scan返回再判断，也不把迟到结果复活成成功。③ 由独立到期控制和每次准入时间核验落实截止，记录实际取消/退出，明确不保证OS/IO硬实时；超界或不能归还保留失败。④ 有限时序回归安排必须覆盖本反例：首结果t=14.9成功但年龄锚点仍t=0，第二次t=15采集有效截止为t=20；到t=20仍未完成则UNKNOWN/取消/停止，不能等t=29.9后刷新。另覆盖首次成功、超时/父取消不刷新、coalesced tick及资源有界收口。合同/离线时序足以判断计划Gate，实际回归另由后续固定源码验证，**本Reviewer不跑Go/PG、不改15/5/20参数、不新增资源框架**。

反例已保存为 `freshness-counterexample.json`，是逻辑时序，不冒充真实Host/SQL运行。它不依赖“任意瞬间绝不超cap”这一新增强要求，保持原观测式fail-stop也需要避免对20s作错误保证。

## 待决定与实施限制（不是新增finding）

### D1：§3.3～3.4的物理增长上界是新约束，不能当原资源合同

原已采用addendum第40行按实际pg_database_size累计观测、核真实卷余量/既有预留；触限停止新点，安全失败取消在途，所有失败/半成品累计保留。它没有要求为每个批或并发组预先证明表/索引/TOAST物理增长的确定上界，也没有保证任意瞬间从未越过cap。

候选第43～44行增加物理增长上界/装入余额/不能证明即拒写，属于更强合同，可能使当前H或完整负载无法准入；“无法则NRU”是安全停止，但不能反过来把这一新证明责任作为原12项必须补齐的义务，或据旧反例曾真实超限重新打开已核销的安全负例。

用户/Lead本轮明确倾向最小保留观测式fail-stop、同步guard、unknown停止及真实超限FAIL/原样记录。该方向可由原作者形成新明确delta；当前固定候选尚未据此修订。若Lead另行采用更强预留约束，应明确新增义务、实施范围与费用，不能称其只是5→15s的字面变化。Reviewer不代选或写入影子quota，也不把无法给出物理上界本身判成源码Bug。

### D2：harness observer的生命周期不能自动算作原phase tail

原point合同是首warmup起，到相关phase操作/取消/回滚/join退出，TIMED wall<=121、SINGLE<=60、累计tail<=1s。旧H的monitor由harness root/diskCtx启动（H:1531～1532/1567），登记到h.group，场景关闭时在H:1083取消并有界等待；它不是旧phase局部worker组。H:2839开始point计时，H:2907先采pointWall，再在H:2910读取size统计；observer统计和harness一般关闭与phase计时不是天然同一个生命周期。

候选第54～56行要求phase取消时停止并join关联monitor，把其SQL取消/归还/join装入原1s tail。这可以是更强的新owner合同，但不能称原121/60数字没变就等于测量定义未变。应明确哪类probe是phase拥有的依赖、哪类是harness后台观察；前者计入对应point的实际工作/收口，后者按其root/scenario/attempt有界关闭、费用单列并保留失败，均不得延长业务phase或让未关闭资源假PASS。是否把harness observer重归phase所有，需作为新增采用范围而非隐式前提。

取消、漏采、过期、unknown仍不能放行新写入；不允许无限Close、丢弃worker、额外15s尾部或后台失败后继续后点。这里需要澄清责任边界，不要求为文案新增一套架构，也不重开原F002代码审查。

## 沿真实消费者的可实施性检查

| 直接接缝（旧freeze定位） | 本候选已提出的必要变化 / 后续实施限制 |
|---|---|
| H:1318 checkDiskBudget | 现有diskMu.Lock先于5s context，候选已明确等待也纳入15s且可取消，不能只换字面timeout。Acquire/SQL/Scan同一绝对deadline；错误要向调用者/root传播，不只t.Errorf。单飞只限制采集并发，不证明20s识别。 |
| H:1435及1463 newHarness | 初始化size目前用长ctx，随后还有一次guard；两处必须统一受资源采集预算/更早截止约束，首DDL之前未知/超时零业务写。baseline/origin仍须r04新实现与SHA绑定，不产生新余额。 |
| H:1523～1587 monitor | 旧设计有专用pool、5s ticker和单owner，适合承接有界单飞；SQL>tick会跳/合并tick，不能排队补采。需披露占用/跳过次数、总采集时间及前台守门暂停；15s单次上限不等于总采集负载很低。新鲜度语义受finding限制，phase归属受D2限制。 |
| H:2166～2410 seed | 旧seed一个owned事务、多次CopyFrom，直到2400才Commit，并非已有“每批前后同步guard”。有限批/事务边界、迁移/ANALYZE以及新增工作量要有具体H映射和取消证据，不能凭计划说已经实现，也不能降低fixture/原断言。D1新增物理证明不自动适用。 |
| H:1083 closeResources | root/disk取消、worker wait、tx回收/连接归还、pool/probe关闭、backend/goroutine核证仍必需。15s上限不增加关闭预算；需按选定owner及更早截止取消并证实归还，非合作分支保留真实失败。当前新H未审，不声称其已安全。 |
| H:2578 pgPerfDatabaseSize及point统计 | 旧reader取diskLastBytes并受同mutex保护，不能拿它代准确guard成功或未知=0。作为指标/差额快照时需标明观测时间/区间与等待成本，不默认为新的写入准入许可；不能通过post-point统计等待偷移业务计时终点。 |

这些消费者确实说明需局部行为delta，不是仅将5改成15。候选将新增写集建议限制H、不能履约则返回具体缺口，不默认扩域，合理；当前没有可据此签发“已经可安全执行全量”的实现证据。

## 根因依据、回归与覆盖关系

真实open-handle raw共8样本，其中6次全新枚举不等于live handle EOF：同进程8192/16384/24576时枚举0，跨进程28672/32768/36864时枚举24576。Sync/关闭后才相等。stdout SHA CF87FADC0A0F2966AACF6D501A3CAD6613C47204E68D0D31599A6FFB497ED57B与result匹配，实验actual_wait_exit1是准确性反例，不是编译失败；其前后五hash相同。仅D盘fixture，没有PG连接/DDL，不泛化成C盘每种PG写法的实测。

因此拒绝缓存枚举替代有真实依据。准确SQL此前确有约6.955s成本，5s与已观测成本失配；15s是显式固定工程余量，不是查询提速或底层所有原因已被证明。候选保留准确全库SQL、禁止自动上调/重试/近似fallback、错误不吞，是有依据的合同候选；Reviewer没有替Lead采用15s，也不把任意调大timeout都称为根因修复。

现有有限回归表基本覆盖：同SQL >5且<15（三次、自然/受控延迟分账）；Acquire+SQL总超15及SQL单独阻塞、目标阶段真正到达；超时root取消/后点NRU/资源归还；实际seed增长/按所采用观测或更强约束触限；更早phase/point/attempt取消及非合作失败。需在上述finding关闭后增加对应coalescing/扫描中增长/新鲜度过期的区分性场景，并按D1/D2实际采用范围解释通过条件，不能强套未采用的配额或observer生命周期。

5s tick加每批同步guard可能造成显著后台扫描和前台暂停；旧57～71min估计不能直接沿用。所有成本计新attempt与适用场景/point，不暂停采样时钟、不sleep补窗、不减10/30/三重复；放不进r04的145/150/53门就NRU。候选已经承认这一限制，不要求为本次非代码Review先执行采样，也不给完成时间或容量保证。

候选§6明确只覆盖未来实际采用policy的资源5→15及相关合同，旧5s日志、旧policy、r02/r03/r04原文不变。新policy须记录候选/独立Review/Lead实际采用来源，明确与r04 P0原5s门的覆盖范围及资源origin不变；未采用仍按旧门，不回溯追绿。原clock-resource-r04规划PASS保持，不能替本预算采用或在途新H代码Review。

## 交回与STOP

返回原TestEngineer一次聚焦修订PROBE-BUDGET-R01-001：采用Main本轮提出的新鲜度停止语义，结合Lead最小观测式方向明确D1/D2为新增采用取舍；不由Reviewer代修。复审只核该固定delta及必要新增风险，不重开原12项/旧规划或新增资源框架。

固定r01尚不能PASS。即便后续计划delta通过，仍需Lead采用、H新freeze/独立代码评审、区分性实际回归及资源/取消证据，P0未满足不运行依赖活动；原代码6 CLOSED / 12 pending、E001 CLOSED、旧批次完整NRU及任务active保持。Reviewer不创造新attempt或占用任何LOCAL/PG。

独立两轮有界审查完成，**STOP**。
