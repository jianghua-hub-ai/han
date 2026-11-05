# S2.4 原18finding候选快照有界复审 R02

**Candidate Static Gate: CHANGES REQUIRED。原18项中14项已有静态整改落点、核销PENDING；4项仍有实质缺口（F002/F008/F009/F015）。正式CLOSED：0。**

**作者最终freeze与候选一致性：PENDING。本轮完整新校准/small/medium：NOT RUN，12:32:04.586115启动门已过，后续静态变绿不恢复本轮完整验收资格。执行证据Review：PENDING，S2.4仍active。**

## 固定对象与证据身份

本次沿原非作者 Reviewer，只复审R01的F001～F018及修复触及的直接消费者。对象为Lead只读点时candidate，**不是author freeze、不是live代码最终状态**。未重Route、未下级委派、未运行Go/PG；未读取live四个pg_perf源代替快照。原Role、add-code-review与适用规则、原规划PASS及18finding全文在当前上下文有效，复用其已读内容，不重开无关设计/全库审查。

固定 `D:/AID/VSCode/Codex/Han`，manifest声明main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；未Git写。`lead/source-candidate-r02/manifest.json` 的Kind为 `Lead read-only point-in-time source candidate, not author freeze or review PASS`，CapturedUtc=`2026-11-04T04:29:59.3592855+00:00`。五份snapshot均独立核bytes/SHA，与manifest一致。manifest自身摘要见verification-start.json和evidence-audit.json。

下文H/W/R/N分别指该目录四份 `.go.snapshot`，行号全部按快照；D为article.md.snapshot。

| 对象 | bytes | 实际SHA256 |
|---|---:|---|
| H `pg_perf_harness_test.go.snapshot` | 109,555 | `96291E78CBB194B2581A455354746917D8C5B376F40512FDF5D0BC08294CBC9A` |
| W `pg_perf_write_test.go.snapshot` | 16,271 | `7F1AD7DE6A3B5C28B663B960D3A59075C8CFD09839FA591EB518F822925EAD7D` |
| R `pg_perf_read_test.go.snapshot` | 29,862 | `E4A7A9C5A973E023E0B103C35882155578854C5251CB9F8F2CC00A47058B5A40` |
| N `pg_perf_notice_delete_test.go.snapshot` | 45,417 | `A6C749537A5A89CC030EF068B8911EE18852ABAF1B923DDC141B251DD69D6E5E` |
| D `article.md.snapshot` | 13,531 | `D47605C85226A618680A9BA455557236F4D202A7FB3BA9C4FB388D9049650C70` |

离线解码 `author/r02/source-r01/*.base64`，每份先与R01已审五SHA核对，再生成R01→candidate差分，保存于本报告目录。diff行数H1281/W134/R122/N643/D0（含diff格式空行，不作代码增删统计）。没有拿交接前更早baseline代替R01被审版本。

Lead在本次复审期间明确提供live后续变更：H当前为 `EFE435D6BD9F17153A78F63C5AADB9954B7BB09FD2C113B4BFEA371353D364C5`，N当前为 `455DF3927BA32D53DC02E95CDFA446C1FBA35927D28910F36947C9BFBDFA3176`，W/R/D仍与candidate相同。**这些当前值来自Lead，本Reviewer未读取live测试文件或自行复算其当前值；本报告不判断这两份新内容是否修复缺口。** 作者STOP/newfreeze后只对H/N相对candidate实际delta及受影响消费者追加复审；另三份无新变化不重开全量。

## 两轮实际范围

Round 1：实际核manifest/五snapshot/旧R01 base64，读全部W/R差分、N差分及相关函数完整上下文；H按F001～F005/F008/F017/F018对应改动和上下游定向读事务所有权、资源关闭、磁盘监控、持久基线、policy、迁移、phase/point、plan、回归入口、校准finalize。没有要求作者尚未完成的工作已完成。

Round 2：逐项反查修复是否真实进入生产测试入口及旧合同消费者。定向核实际工作树 `migrations/0003_article.sql:188-248` 双recipient延迟约束、`internal/article/query_service.go:759`起SQL常量；通知dispatcher/repository、命令/删除生产者沿R01已读的直接契约继续取证，并自核六个必要生产/迁移文件摘要，记录于evidence-audit.json。另读本机Go testing benchmark的run1/processBench相关控制流，确认初次benchmark失败不会保证调用到第三次cleanup。未执行工具链。

已读 `author/r02/progress.md`、execution-policy.json、run-command-r02.py和Lead execution-policy-r02-adoption.md。新政策明确已采用TIMED名义活跃120/累计尾部≤1/wall≤121，SINGLE≤60，原起点11:29:04.586115不重置；F002按此合同审，不再拿旧严格120ms上限拒绝新候选。该政策不核销任何finding，现已过最迟门，完整重验本轮NOT RUN。

原始证据读取包括phase/body/medium-baseline/initial-drain的RED及GREEN、编译中间失败、disk阶段失败/隔离pool后GREEN、四个PG故障负例、作者负例assert脚本/结果、两次directed semantics命令result/stdout/stderr及资源/inventory。Reviewer只解析已存在文件，复算原日志SHA，不运行作者脚本或Go/PG。所有命令前后源码SHA和是否匹配candidate均进入evidence-audit.json，不能跨版本借GREEN。

## 原18项逐项状态

“已处理/PENDING”仅指candidate中原缺陷有具体静态修复落点；不是正式核销，不代表真实完整场景通过。“仍OPEN/PARTIAL”指candidate仍有明确缺口。所有finding沿用 `S24-CODE-R01-Fxxx`，不另造一套替换编号。

| 原finding | candidate静态结果 | 具体证据和未完成条件 |
|---|---|---|
| F001 HIGH：测量错误不使测试失败 | 已处理；PENDING | H`:2517-2520/2542/2613-2632`先存raw/summary，再统一Errors/截断/实际费用门Fatal并设stopReason；W继承调用不再能仅丢summary绕过统一失败。需当前最终版测量期真实错误、部分成功后错误、正确性错及后点不建schema回归；边界分类受F002剩余问题限制。 |
| F002 HIGH：实际期限/截断门缺失 | **仍OPEN，部分修复** | H`:2357-2409`有phase ctx、准入、completed/canceled和late/unknown；`:2474-2638`有121/60及TailNS门。但`:2404`无界Wait，且长h.ctx helper逃出phase期限等残余，详见下节。已有race仅验证响应取消的op，不能证明忽略取消的收敛。 |
| F003 HIGH：seed期间无磁盘监控 | 已处理；PENDING | H`:1386-1400/1422-1437`专用diskPool和monitor前移至首DDL之前，覆盖迁移/seed；guard错误取消h.ctx。此前seed故障exit1有真实超限、取消、资源记录，源码版本为527F…而非candidate，需最终对应版本或严格相关差分证据，不能说同版已过。 |
| F004 HIGH：medium重试重置基线 | 已处理；PENDING | H`:1303-1320/2654-2676`每档O_EXCL文件与window/tier校验，重复载入原Bytes；r02 wrapper固定baseline目录、原global base。medium反例RED→GREEN已见，但不同全量五hash，最终核对应helper和实际原窗基线记录。 |
| F005 HIGH：泄漏时无界Close | 已处理原阻塞分支；PENDING | H`:1000-1070`cancel、5s worker收敛检查、owned tx登记/3s回滚、close前记录、acquired非零拒绝同步pool.Close并关闭owned连接；旧故障负例exit1/7.532s、acquired0/残留backend0有实物，但候选同版泄漏/异常join负例仍待核。F002局部phase Wait不能被这些场景级cleanup代替。 |
| F006 MEDIUM：c1缺near-limit | 已处理；PENDING | W新增pgPerfBodyClass；H`:2367-2381/2333`逐op交替逻辑body并记录实际Bucket，W`:237-254`核两桶非空。原纯worker0反例已有RED/GREEN；同版c1/c16真实两桶、每正式重复及rune/bytes仍待核。 |
| F007 MEDIUM：W02-c4不可达 | 已处理；PENDING | W给c16设置WarmupConcurrency=4，H`:2505-2516`在原warmup窗实际驱动4worker并记录concurrency/attempts，无新增点/schema或独立P95。需真实c4观测/费用证据，不能以字段存在声称执行过。 |
| F008 MEDIUM：半成品schema漏记 | **仍OPEN，部分修复** | H`:875-889`CREATE成功即addSchema，并写独立迁移status；故障migration负例已有1 retained schema。但addSchema仍仅内存，校准仅repeat3 Finalize，首调用创建后失败可没有inventory，详下节。 |
| F009 MEDIUM：N03完整批断言回退 | **PARTIAL，仍OPEN** | N`:709-713/744-752`恢复cycles≥1、初始accepted==expected、无terminal，50/10000反例已有RED。新增批helper仍违反双recipient约束，实际N03复合路径不能完成；1k/10k directed均因前错NOT RUN，详下节。 |
| F010 MEDIUM：N03覆盖工作量证据 | 已处理；PENDING | N`:797-800`路径含schema/run/batch且记录case；H`:686-694`O_EXCL拒绝覆盖。静态键隔离成立，但尚缺同版双点batch1/拒覆盖的原始回归；F009阻止真实新增批产物完成。 |
| F011 MEDIUM：E02串行替代竞争 | 已处理；局部同版观测，核销PENDING | W`:338`起真实Replace/Moderate动作，H`:1655-1700`holder、两个distinct PID、递归阻塞依赖、释放后收双方结果；核唯一赢家与audit/category/version。candidate同五hash directed-r02此子项PASS7.85s；整个命令exit1，不能称整个directed通过。首次r01竞争timeout/连接残留失败保留。 |
| F012 MEDIUM：D02无真实UPDATE竞争 | 已处理；局部同版观测，核销PENDING | N`:254-279`真实versioned T1/Replace竞争与赢家终态，H`:1735`T1额外expectedVersion守卫；candidate directed-r02此子项PASS8.18s。仍待作者最终freeze及相应回归链，非完整SINGLE窗口证据。 |
| F013 MEDIUM：D01幂等尾部 | 已处理；局部同版观测，核销PENDING | N`:229-251`每次Create全新active行再首次T1，核pending/version/operation和业务不可见；candidate directed fresh-t1 PASS7.80s、3个独立operation。它是定向功能观测，不是三重复/120秒性能验收；准备费用/单位仍须如实报告。 |
| F014 MEDIUM：R05变化全在分页前 | 已处理；局部同版观测，核销PENDING | R`:440-508`首page后Create/Edit/Suspend，独立SQL按真实boundary求后续期望，再从原cursor续页，恢复fixture有独立清理context；candidate directed cross-page PASS8.63s。尾部取消/恢复资源仍受F002及最终回归限制。 |
| F015 MEDIUM：N04缺晚到/同worker | **仍OPEN，已有同版实际失败** | N`:827-895`设计了不同/相同worker的真实dispatcher晚到结果fence，但FreshNotice两次autocommit且只插一个recipient；candidate directed-r02在此SQLSTATE23514失败，后半语义未到达，详下节。 |
| F016 MEDIUM：X08未观察C链 | 已处理静态结构；PENDING | N`:1023-1091`三PID、A∈blockers(B)/B∈blockers(C)有界查询与graph、release和done，worker release感知取消。candidate同版该子项因N04前错NOT RUN，不得借前面的PASS核销。 |
| F017 MEDIUM：读EXPLAIN不可达 | 已处理静态可达性；PENDING | H`:1415-1418/1848-1887`read fixture计时外调用真实生产SQL常量的EXPLAIN ANALYZE BUFFERS，记录schema/query/args/plan，权限失败单列，其余错误失败；当前query_service.go摘要已核。candidate同版production-read-plans因N04前错NOT RUN，plans内容与覆盖仍待实际核。 |
| F018 MEDIUM：PGDATA失败fallback | 已处理；PENDING | H`:1322-1330/1354-1358`在任何DDL前取得目标卷且probe失败Fatal，已无evidence卷代替；旧故障负例exit1/3.610s，inventory0/backend0，但五hash非candidate，同版故障与最终冻结关联待核。 |

## 仍有实质缺口及复审条件

### F002 [HIGH] point context尚未形成有界实际退出保证

位置：H`:2352-2409/2474-2638`，以及H`:1513`的activeUserIDs和N/W/R新增复合op对pgPerfMemberPrincipal(h)的调用。

已修名义费用/phase取消/最终repeat状态，但phase执行后直接`group.Wait()`，没有受point/尾部预算约束的退出裁定；pointWall/TailNS检查只有所有op返回之后才能执行。op忽略取消或落入不受phase约束的阻塞路径时，121秒和整点1秒只是事后拒绝，无法在该界限内收敛。现有TestPgPerfR02_PhaseDeadline_CancelsInflight的80ms timer分支会主动select ctx.Done，验证的是配合取消的op，未覆盖忽略取消/持续阻塞。

复合调用仍有真实期限逃逸：例如pgPerfFreshT1、E02/D02、R05、X08的op内调用pgPerfMemberPrincipal(h)，最终activeUserIDs用h.ctx（长场景context）而非operationCtx查询。该DB步骤可能在phase已截止后继续准入/等待；BeforeAcquire看到的也是长h.ctx，不能证明每个新DB步骤均受Di约束。不能因外层op曾在窗内开始就免除内部准入门。

另：phase原始记录仍以observed_ms/tail_ms和completed布尔为主，未见新合同要求的完整Si/Di、实际关闭/取消、最后完成和全部worker退出的原始精度时间轴；未到同版完整窗口执行，不能据point总NS单独证成每phase符合性。统一point outcome也未要求有效成功样本大于0，不能把全部边界删失的零成功点产物当有效性能结论。

修复/复审：把真实调用中的全部业务DB步骤绑定operation/phase期限；对持续不响应取消的操作保留实际所有权和有界关闭失败证据，不能留下无人join的goroutine假装退出。补忽略取消、复合helper跨Di、最终phase提前取消、边界写unknown、零有效成功、累计尾部超1秒与真实错误的区分性回归。F002按新121/1秒合同核，不要求退回旧120秒，但不能只改比较值或只看合作取消的测试。

### F008 [MEDIUM] 校准首调用失败仍可能没有实际inventory

位置：H`:458-462/593-600/2902-2919`。

迁移路径现在CREATE后立即加入内存e.schemas，解决了原“迁移返回后才登记”的主体缺陷；但addSchema不落盘，inventory仍依赖Finalize。Benchmark仅`repeat==3`的Cleanup调用Finalize。若第一个benchmark调用创建schema后在迁移/计时查询失败，不能保证调用到第三次cleanup；本机testing.run1会在初次失败返回false，不能把count3当作总能完成三调用。全局evidence cleanup本身仅建目录，不替它保存inventory。

影响：校准K和失败/半成品清单仍可能缺失，违反原addendum的“所有实际调用、失败实物也计入”义务。已有fault-migration测试注册了独立Finalize，不能覆盖真实benchmark这一不同消费者。

修复/复审：schema创建事件及时持久化，或保证校准任意提前失败都有完整checkpoint并不冻结后续调用；核首校准已CREATE后故障的raw/exit、保留schema、迁移状态、K/inventory及关闭记录，不固定K=3，不跨尝试拼正式重复。

### F009 [MEDIUM] 初始批门已恢复，但后续批仍无法满足真实recipient约束

位置：N`:696-703/718/755-795`；生产契约 `migrations/0003_article.sql:188-205/238-248`。

pgPerfAppendNoticeBatch为每notice插new_author，只在index%3==0时插former_deleted_author；首三条notice的recipient数为[2,1,1]。真实延迟约束要求每条notice都恰有两种各一。一个事务内COPY并不能消除约束，提交包含index1时即23514。

该helper在原batch0完整drain后立即调用，且retry/terminal前也调用，故不是可忽略的可选负例。恢复初始全量断言和纯函数反例GREEN不代表N03完整场景已可执行；candidate同版1k/10k directed根本未启动，不能借此认为新增批已过。此处与F015共享同一已核生产约束，但为不同helper/事务路径，分别保留原ID的部分整改状态。

修复/复审：保持完整两种recipient结构，同时按既定fixture区分due与已终态行，维持1k/10k原完整批和发送/accepted/terminal计数。核初始批、至少一个完整新增批及retry/terminal，保留schema/run/batch独立工作量证据；不得通过删约束、缩批或改期望数绕过。

### F015 [MEDIUM] N04新路径在晚到/同worker断言之前真实失败

位置：N`:845-854/857-895`；同版证据 `author/r02/pg-directed-semantics-r02/`。

FreshNotice先在pool.Exec中autocommit创建notice，随后才另一条Exec插单个new_author recipient。第一条提交时已触发“必须恰有两recipient”的延迟约束；即便合成一个事务，也仍缺former_deleted_author。当前同candidate五hash的真实定向运行已返回SQLSTATE23514，晚到A/接管B/相同worker两个分支没有成功完成。

修复/复审：在合法事务内形成完整合成fixture后再驱动dispatcher，保持fake mailer、过期SQL回拨、不同/同worker ID、attempt/lease完整fence、旧结果不能覆盖终态的真实断言；实际同版定向exit/原始结果及资源关闭须证明两个分支均到达。不可只凭函数出现或后续另一revision局部GREEN核销本candidate。

## 原始失败链与版本限制

1. `pg-directed-semantics-r01`：exit1/23.359s，H=`D0A7D2BC…`、W=`D4E0CB07…`、R/N/D见audit，**不是candidate**。fresh-t1 PASS；moderation-race bounded poll timed out，随后记录未归还连接2、goroutine基线3→4，其余因stopReason NOT RUN。该实际失败链保留，不归因于环境或编译，也不被后续四子项PASS覆盖。
2. `pg-directed-semantics-r02`：前后五SHA与candidate完全一致，exit1/44.156s；2026-11-04T04:28:16.047980Z开始，04:29:00.210588Z结束，stdout SHA `800435B6D7438478B250BD10840FD8A12991C4B28F4563327C28DC3AE55BCCBD`，stderr为空。fresh-t1、moderation-race、delete-race、cross-page四子项分别PASS；N04 23514 FAIL；three-lock-chain/production-read-plans/initial-drain1000/10000四子项均**前错NOT RUN**。不能把它们的Go fail事件误说成各自已运行失败，也不能把前四PASS当整条命令PASS。
3. `pg-fault-migration/volume/leak/seed-disk-r01`均为527F…等较早源码：预期故障exit1分别7.828/3.610/7.532/12.531s，原始marker匹配，inventory分别1/0/1/1，保存的资源记录backend0且goroutine回基线；迁移status显示created=true/committed=false/failed_at3，seed有真实guard超限和取消。它们是较早修订下有效的拒绝/收敛证据，不是“Go成功”，也不是candidate同版验证。作者assert脚本exit0只说明所列旧产物满足其断言，Reviewer已直接核日志/JSON，未以作者PASS代独立核销。
4. core race×3/phase/body/baseline的RED→GREEN、initial drain RED和中间编译FAIL均保留，摘要与scope见audit。编译失败不冒充行为RED；旧局部GREEN不自动绑定新snapshot。
5. Lead后来提供`pg-append-green-r01` exit0/11.39s及H/N新的livehash，仅作**后续修订线索**登记，本轮未读它的raw/源码修复，也未将其用来推翻candidate的N03/N04约束结论。等作者STOP/newfreeze后对H/N实际delta及相关证据追加核查即可。

## 交回与下一次有界核验

本轮未给任何finding正式CLOSED，原因包括candidate不是author freeze、四个实质缺口、当前同版directed整体FAIL、关键定向子项未运行、负例/局部回归多属更早五hash。14个静态修复落点可直接复用，避免再次全量重读无关文件。

下一次输入只需：作者明确STOP/新freeze与五hash、H/N相对本candidate实际diff、W/R/D是否保持不变、相关定向命令原始hash/exit/stdout/stderr、失败链与资源/inventory证据。若W/R/D哈希不变，沿用本报告已审内容；若改变，先固定真实delta再补审，不把Lead提供的当前值当已审字节。必要生产消费者也须核对应实际hash；审完由Lead再核live来源、命令、diff/所有权及范围后接收。

新policy的完整性能启动门已经错过，这是本轮最终执行事实。不得因为修复或静态Gate后来转绿启动本轮完整new calibration/small/medium，不开新窗口重置elapsed，不用旧r01/定向片段拼全量验收。执行NOT RUN与静态整改状态分开；执行证据Review继续PENDING，整卡active，不据任何candidate结论关闭。定向返工验证须仍由Lead治理既有窗口/资源/所有权，本Reviewer未启动任何运行或申请新审批。

唯一写集为本code-review-r02报告/verdict、五份离线diff、核证脚本和结果。旧R01及规划报告哈希已核保持原字节，源/候选/Tasks/Plan/旧证据未写；无凭据读取、Go/PG、Git写或cleanup。完成本candidate有界复审准备后STOP，等待作者最终freeze与两个文件delta，不追读变化live源。
