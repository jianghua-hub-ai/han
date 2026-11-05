# R09实际正常语义独立执行Review

**结论：整体PARTIAL/PENDING；8个子测试正常PASS有效，10k seed失败且drain未到达，整组不能PASS。** 从已接收7 CLOSED/11 PENDING基线，本次仅新增F015/F016 CLOSED，原18项累计9 CLOSED/9 PENDING。两项竞争真实运行及正确性断言已PASS，但缺原合同要求保存的返回值/最终行与审计快照，F011/F012保留PENDING。没有从本轮确认新的源码HIGH；一个MEDIUM执行证据缺口见SEM-R09-EV001。

## 固定对象与两轮实读

唯一实际执行对象`author/followup-attempt-r01/se-114540-42888/`，依R09 source-freeze-r09、单stage semantics grant及test-plan-followup-r04最低L/N/W合同判断。工作区固定Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca；本Reviewer未运行Git核分支。复用同一add-reviewer/add-code-review及已取得的适用规则，不Route、不委派、不重跑P0/leak/N9。

第一轮实读start/grant、launch/PID/actual-exit、完整stdout/stderr/Go JSON、preflight/snapshot-observation/final、全部e目录JSON与material原件；扩展Windows绝对长路径枚举，逐份重算SHA/bytes并比较原件值和material内值。自行复算五freeze snapshot/live摘要、实际nonrace binary及policy/采用/前置接收引用。作者report/execution-index作为索引读取，未直接采信其核销结论。

第二轮只沿真实直接消费者：冻结H的RealSemantics case表、raceRow、VerifyInitialDrain、captureReadPlans/writePlan、seed父context及recipient COPY；冻结W的E02Iteration、N的FreshT1/D02Iteration/LateNoticeResult/LockChain、R的CrossPageIteration及SQL oracle/续页调用。比对三份计划的SQL与真实工作树query_service.go生产常量，核参数、节点、actual rows、buffers。读取98组原始只读query的launch/pid/actual-exit/result，重新由stdout解析payload，核精确app/backend PID/start、实际wait及外部静止核账。没有全库源码重审或修改真实消费者。

自有offline-audit.py只读/hash/解析，932项离线核证均成立；其数量是绑定与一致性检查，不是932次测试或全部业务要求已满足。原件SHA/bytes、每分支结果、资源指标与限制在evidenceaudit.json。末尾重算所有已消费输入无漂移。

## 版本、授权与真实结果

R09 H241407B，SHA C3DFDC66FDBFB025760ED78B0E0061CC49BD7AFB2DA0BF76685D94D0D89B8B99；五份snapshot/live及start/launch前/final后source map完全一致。实际binary71B3EF47D466A8EAB899C266311EAD224E5F5316D6B244A8E2D7EFDF705521D5；当前policy7C94C0DAE104A177FE82E976FD12154F97880ED749CEFBFEFCE572EAA4A33E04保持。source manifest18B99D50…4364C02和单stage grant的路径/hash与start保存值匹配，runner仍r03，前置leak接收及live-accounting采用引用匹配。

grant创建11:45:24.326360Z，实际Go40712于11:45:43.221962Z启动，11:53:40.711913Z自然wait exit1、forced=false；入口精确`^TestPgPerfR02_RealSemantics_Directed$`、count1、test.timeout20m，CWD为真实test/integration/article。公开env未设置INJECT_FAULT=1；并未把带故障入口混入本正常组。Go JSON与原始stdout逐分支结果一致，转档进程单独自然wait，转档exit0不替代Go FAIL。

| 子分支 | 真实结果/秒 | 独立核到的实际消费者与物证 |
| --- | --- | --- |
| fresh-t1 | PASS / 36.51 | N239-266每次新Create，再真实beginT1；SQL核pending/version2/operation关联及业务detail不可见。H5968-5977执行3次且deletion_operations恰3，非循环旧ID幂等读取；本次无独立逐ID结果JSON。 |
| moderation-race | PASS / 39.50 | W417-475实际并发Replace与Moderate同version1，H raceRow barrier及两真实contender依赖holder；硬断言恰一赢家、败方ErrPreconditionFailed、最终version2、赢家对应audit/category/version、无幽灵审计。锁图在原件；具体返回值/最终行与audit原值未保存，见EV001。 |
| delete-race | PASS / 41.37 | N278-326实际T1与ReplaceContent竞争，非holder手工锁代替UPDATE。硬断言恰一赢家、败方预期错误、最终version2，T1赢则pending+operation，UPDATE赢则active且无operation；有真实锁图，但具体赢家/行值未保存，见EV001。 |
| cross-page | PASS / 41.83 | R455-512确为page1→Create/Replace/作者suspend→独立SQL oracle→续页；创建的新head不应进入续页，之后恢复作者状态。原interleaving有边界/插入/编辑/作者ID、expected_remaining188及remaining_pages2。 |
| late-notice-results | PASS / 45.25 | N888及940-1049实际运行sameWorker=false/true，fake mailer等待A，SQL回拨lease，B接管并保存accepted，再放行A晚结果；两份原lease回执保留owner/attempt/lease/accepted_at及稳定性结果，详下。 |
| three-lock-chain | PASS / 44.92 | N1243-1340实际A holder、B/C两个worker，原始C→B→A阻塞图；后续释放A、等待B取得、释放B并join B/C，错误均传播，非仅看到55P03。 |
| production-read-plans | PASS / 43.06 | 三份真实生产SQL EXPLAIN ANALYZE BUFFERS，first/tail-edge/detail参数与节点原值核验；不是SELECT1或仅文件计数。 |
| initial-drain-1000 | PASS / 55.98 | 原initial-drain JSON accepted增量=sent=expected1000、failed_terminal0；由真实dispatcher/Reconcile循环和独立SQL数量检查产生。 |
| initial-drain-10000 | FAIL / 128.84 | newHarness在seed transfer_notice_recipients时父context到期；无完成seed manifest/10k drain JSON/完成marker，tc.run未进入。保留已创建schema，不称10k drain失败已覆盖其业务断言。 |

上述N分支耗时包含初始化、fixture、原guard与关闭；并非正式10/30s measurement，不产生合规TIMED/SINGLE性能点。

## 竞争、lease、锁图和计划的原值

E02锁图holder39608，contenders23800/18772；23800等待holder的transactionid，18772经tuple等待23800，两个真实参与者都传递依赖holder。D02 holder19792，contenders13228/42080，同样分别transactionid与tuple传递依赖。两份race.json均both_depend_on_holder=true、probe_error空，三PID不同。固定raceRow在释放holder前观察依赖，之后收齐两返回；E02/D02函数业务SQL硬断言必须全部返回nil，才能打印对应marker并PASS。因此本次确实超出“只按测试名”的证据：竞争和正确性断言到达可确认；实际哪方赢、marks/matching/最终行原值仍不能从现存JSON读取。

N04两回执分别notice dd2cc621-34b6-4751-be8a-afb680c5a073（不同worker）与7ab8195e-4ebd-4226-baec-448821f5be05（同worker）。均first_attempt1→successor/accepted attempt2，successor lease严格晚于first lease；同worker分支owner确实相同，另一分支不同。late_result_rejected=true不是凭空标签：只在旧A结束后再次SQL核status=accepted、attempt_count2、accepted_at未改、claimed_by/locked_until均NULL后写出；A dispatcher实际done/join已消费。该实际结果保存链满足原F015缺失的晚到与同ID不同attempt fence，无SMTP或sleep凑窗。其余recipient预先置accepted，两个新notice隔离运行，未将旧fixture当两个竞争target。

X08原图A38348、B39868、C43312，B.blockers=[A]且C.blockers=[B]，B wait_event=transactionid、C=tuple，均Lock。当前消费者在5s lock_timeout内有界观察，再正常release/join；最终资源0补足后置条件。原F016缺C排队的缺陷可以核销，不要求另造一个同名测试或依赖通用worker取消证据代替该图。

plans三份query文本与当前生产listArticleCandidatesSQL/getArticleQuerySQL相同，schema同r2plan。first参数null/null/200，root Actual Rows200、Shared Hit Blocks45，底层实际Seq Scan300行；tail-edge使用真实最尾key，Actual Rows0、Shared Hit Blocks4，并有articles_created_at_id_idx的Bitmap路径；detail使用同ID，Index Scan/Actual Rows1、Shared Hit Blocks3。均有Execution Time/Actual Loops和BUFFERS；minimal300行使用seq scan不是本次计划失败，也不能推为medium索引/容量合格。其N子条件有效，F017正式各R/X点计划及扫描W仍欠。

1k seed记录750 notices/1500 recipients，源规则每notice一个new_author pending、每三条另一个former pending，其余former预accepted，故初始due恰750+250=1000，而非把1500规模缩成1000。VerifyInitialDrain按expected/50+5有界轮数执行，SQL核完整accepted增量/failed_terminal和fake发送数；本次原回执四项1000/1000/1000/0一致。未记录实际轮数原值，不虚构轮数；完整数量门已过，F009仍缺10k及正式W。

## MEDIUM证据缺口 SEM-R09-EV001

- 对象/位置：当前semantics leaf material及locks/<r2e02|r2d02>*.race.json；冻结W417-475、N278-326的返回结果/最终行查询；test-plan-followup-r04 §2表F011/F012及通用E。
- 现状：原件只有竞争锁图、marker/PASS及资源，没有双方返回值、最终article快照、E02 audit行/计数原值。原物证完整枚举并核SHA，不能把原件缺失解释为长路径漏读。
- 触发/影响：需要复核实际谁赢、拒绝类别及该具体赢家的version/审计归属时，只能知道固定函数中的关系断言通过，不能展示原合同要求的实际值；不能把marks0/1或赢家身份猜成观察事实。
- 本次处理：F011/F012保留PENDING；不称其竞争未运行或仍存在原源码MEDIUM/HIGH。其他已有充分原件的纯N项F015/F016可独立CLOSED。
- 有限补证方向：现有schema保留，优先由获许可的原owner追加同schema/同目标的最终article/audit原值与原执行绑定，结合原PASS/固定断言给出可复核赢家/结果分类。没有保存的原参与者返回日志须如实标缺，不能事后制造为旧raw；若只能从终态推断，明确推断和控制流依据，再交原Reviewer判断联合证据。不要求为此重跑8成功分支、不要求加新资源框架，也不由本Reviewer查询PG或修改源。

## 10k未完成与资源/库存

small root仍300s，seed父预算仍120s（H2232创建seedCtx）；10k父截止11:53:40.4154537Z。最后成功采集flight20从11:53:30.6027262至37.0555527Z，bytes2,129,606,335；后一个采集从37.0850372至40.4180558Z，SQL约3.333s便因更早seed父截止返回context_deadline，并非证明它独立耗满15s。stdout首个错误在recipient seed，resource terminal同context deadline。原seed JSON没有10k完成项，也无10k drain物证；实际COPY进展/提交行数未知，7500 notices/15000 recipients是配置，不伪装成实际已落行数。未等待Backend根因诊断即可判10k正常业务目标未达，不能把此预期之外FAIL算负例PASS。后续seed候选/SourceR10另审，本结论不提前采用预算或放宽120。

9套资源每套共享原5s closure截止，actual closure分别：t1 22.8729ms、E02 13.9271、D02 14.6784、R05 13.6744、N04 21.8535、X08 22.4841、plan16.6965、1k122.229、10k124.5542。正常8套failed/resource_failure=false，失败10k相应true；所有worker joined/root canceled、pre-close acquired0/return_error空、各return/Close完成/acquired_after0，goroutine3→3、residual_backends0，无fail-stop。没有把已存在更早seed等待时间算成延长closure或用外层kill制造0残留。

98个psql客户端+Go+converter合计100个owned客户端与final逐PID/实际exit对齐，均真实wait、无forced/unresolved。live精确app为han_pgperf_r04_fu_sem_114540_42888，去重PID/start清单在audit；after query94精确业务app为空，query98覆盖本次observer前缀并排除自身后为空，自身另有真实wait，不声称这是全Host连接数0。前后identity相同，catalog前后active_other0，外部准确size-after在Go退出之后执行。

旧85个schema全部保留，新增9个与9个created events、inventory和实际库差集严格一致，最终94；失败10k半成品也retained。准确before2,105,767,615B，相比grant历史snapshot2,105,743,039B有24,576B漂移，按已采用live-accounting记录并继续原cap核账，不归因未知底层原因。静止after2,129,835,711B，本组实际增加24,068,096B。

全部9个fingerprint加载同一原small80B/SHA2B368282…7FC4D2、resource Window03:29:04.586115+00:00、origin1,406,932,671B；最终累计722,903,040B，small余350,838,784B。未以9次新harness当前DB初值重发余额，未删除/移动旧schema或重置ledger。该余额仅为本次收口读数，不保证未来fixture装得下。

## 原18项逐项本轮状态

| Finding | 本次后状态 | 新证据及尚欠 |
| --- | --- | --- |
| F001 | CLOSED沿用 | 本组确实FAIL，不重开既有统一错误拒绝核销。 |
| F002 | PENDING | 没有正式3×10/30点、累计tail及121/60真实W；正常诊断耗时不能替代。 |
| F003 | CLOSED沿用 | 原seed保护/实际growth已接收，不重跑。 |
| F004 | CLOSED沿用 | 原持久资源绑定保持，不重跑跨进程基线。 |
| F005 | CLOSED沿用 | 本次前版leak独评已接收7C基线。 |
| F006 | PENDING | 本组未跑正式W02 c1/c16各repeat两桶。 |
| F007 | PENDING | 本组未跑正式三次10s c4 warmup观测。 |
| F008 | CLOSED沿用 | 失败10k新增schema也实际入inventory，补充佐证，不重开。 |
| F009 | PENDING | N1k完整数量通过；N10k seed失败/drain未到，且W未跑。 |
| F010 | CLOSED沿用 | 未覆写旧workload，无新增整项动作。 |
| F011 | PENDING | 真实竞争及唯一赢家/审计关系断言PASS；EV001所需返回/最终行与audit原值缺失。 |
| F012 | PENDING | 真实T1-vs-UPDATE及最终状态断言PASS；EV001所需双方结果/最终行原值缺失。 |
| F013 | PENDING | N三次新T1、pending/version/operation及不可见断言到达；W每phase新T1计数未跑。 |
| F014 | PENDING | N真实跨页变更/独立oracle已过；正式R05活跃窗口W未跑。 |
| F015 | **新增CLOSED** | 两种worker实际晚结果及完整attempt/lease/accepted稳定回执满足纯N最低合同。 |
| F016 | **新增CLOSED** | 原始C→B→A锁图与正常释放/join/资源闭环满足纯N最低合同。 |
| F017 | PENDING | N三份生产SQL actual rows/index/buffers计划有效；正式全部R/X点计划/扫描W未跑。 |
| F018 | CLOSED沿用 | 目标卷/准确字节保护未更改；本次卷余量有真实读数。 |

原E001保持CLOSED且不计入18分母。新EV001是本轮取证缺口，不能与原E001混同。任务状态未改；无整卡关闭或性能/E2E通过结论。

完整calibration/small/medium本attempt已永久NRU，旧完整性能及large/canonical NRU不变。唯一Tnew10:22:37.932710Z不重置；12:47:37.932710Z停新、12:52:37.932710Z硬停保持，Review耗时仍计原窗。本报告不批准下一命令或SourceR10，未运行Go/PG/SQL、未改源/Tasks/Plan/旧物证；后续定版预算delta仅在本目录另文有界审查，不反写本轮事实。

最终报告/verdict/audit/脚本SHA及actual STOP见index.json。
