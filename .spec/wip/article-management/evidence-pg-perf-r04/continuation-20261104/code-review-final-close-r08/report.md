# R08最终关闭消费者独立有界复审

**有界静态code Gate：PASS；CLOSE-R07-F001 CLOSED，无新增confirmed finding。** 修复已进入真实closeResources最终goroutine消费者，predicate前后及最终成功返回前均核ctx与共享closureEnd；观察本身返回的真实错误仍保留。结论不代表PG、P0、性能或整卡通过。

## 固定对象与范围

objectType=`code`，原非作者Reviewer，固定Han/main/指定HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，复用有效Role/add-code-review和适用规则。唯一写集为本code-review-final-close-r08目录；未Go/LOCAL/PG/SQL、未运行作者runner/binary、未读凭据、未源/Tasks/旧制品修改/Git/清理/委派/采用或执行决定。

正式对象 `author/probe-final-close-r08/source-freeze-r08/manifest.json`，captured2026-11-04T09:37:54.683421+00:00，source/local STOP、PG未授权/未使用。H **226915B / AC2AD1AC172C2E78D898DAAC624CEC35965719023B78E24D87101144E2B23878**。基线R07 H **223792B / 9F08F70B2E2063D5F178064F5EA0D9F84DABB1BA625CEF90402DE70D4CB7CAC2**。

五snapshot及五before均自行复算摘要，另三Go/docs保持原值，旧global/window/policy/small四摘要匹配。未用live源替代快照。Main的现场scope/live与Manifest7checks由Main独立完成，本Reviewer不跑或代报；在途readonly runner不在本次对象中，未读取或判定。

## 两轮实际检查

Round1完整读r07-to-r08.patch、独立snapshot diff及新增helper/测试；离线核4个hunk上下文并重建结果，与固定R08的LF文本完全相同。delta仅为最终消费者换调用、新窄helper、新回归和fixture root迁至R08。

Round2核closeResources→waitFinalClose→未变pollUntil→t.Errorf/resources.failed/stopReason的真实接线；反查共享deadline来源及R07已审Close/归还/真实错误保留未退回。按同一提取器直接比较R07/R08实际文本，核采集/身份/只读/预算函数未变，不重审全库或旧18项。

实际读作者交接报告（已到盘）、manifest及7条result/raw，独立核前后五hash、真实wait、raw SHA和目标测试到达。manifest函数hash的提取边界与R07可能不同，**没有跨提取器比数字作版本变更结论**；实际snapshot diff及共同提取范围的文本相等才是本报告依据。审计脚本仅Python离线读/hash，详见verify_offline.py/evidence-audit.json。

## CLOSE-R07-F001逐项关闭

| 原关闭条件 | 固定R08落点/证据 |
|---|---|
| 接到真实最终goroutine消费者 | H **1145** 将原h.pollUntil改为h.waitFinalClose，传同一个ctx/closureEnd及runtime.NumGoroutine predicate。返回错误仍走原t.Errorf，resources.failed与stopReason消费t.Failed，不会只增加一个未被调用的helper。 |
| expired+已符合不能成功 | H **1157起** expired先核ctx.Err，再要求Now严格早于closureEnd；predicate前先核。ExpiredCompliant实际RED得到error=nil并失败，最终普通与race明确PASS。shared-deadline-with-live-context还证明仅共享截止已过时不调用predicate。 |
| 晚回落不能清除失败 | predicate返回成功后再次expired；pollUntil返回nil后还做最终expired。late-return回归在ctx.Done后才返回true，必须得到DeadlineExceeded；不会因pollUntil先看true而通过。 |
| 期限内成功 | in-time用未过期ctx/end和true predicate，实际普通/race成功，无额外sleep或新窗口。 |
| ctx取消与原错误 | cancel-during-observation不能成功；real-error-survives-cancel返回原42501对象，未用取消覆盖真实predicate错误。helper原错优先返回，不改全局错误分类。 |
| 预算/实现保持最小 | 全局pollUntil原文未改；helper仍调用原4s局部poll，所有判断沿原共享5s/min-hard，不重新计算5s/15s额度。无采集/owner/phase/clock/ledger/cap修改。 |

满足原finding要求的expired、晚回落、有效成功及真实消费者接线/错误保留。原R07 finding-trace是静态反例，R08才取得这条离线行为RED；不把它写成旧PG反例或Reviewer独立跑出的Go结果。

新helper只包这一个最终消费者。predicate为实际快速runtime计数，新增受控测试可把观察阻塞到ctx到期，以验证观察后拒绝；本报告不据此承诺任意不可合作callback都可在硬实时截止瞬间退出。

## 同版证据、计数口径与不变函数

7条命令前后hash/raw逐项核：RED是中间H **B03D725F16C9FB0780996D73D455DDE1CBB070E20C319AAB7064D207C7D85FFB**，exit1/3.500s，stdout实际 `expired closure accepted compliant predicate: error=<nil>`；不是编译失败，也不是精确原R07全文重跑。其余6条均为最终AC2AD1…。

| 最终同五hash命令 | actual wait / wall | 独立核到的结果 |
|---|---|---|
| green-close-r01 | 0 / 8.250s | 父进程实际测试/子测试PASS记录 **27**，0fail/skip。 |
| race-close-r01 | 0 / 32.703s | count3父进程PASS记录 **81**，0fail/skip；R08目标7个测试/子测试记录每次均到达，共21条PASS。 |
| core-r08 | 0 / 10.656s | test2json count3合计57个有Test的pass事件，0fail/skip。 |
| build-r08 | 0 / 4.891s | 当前integration/race binary构建。 |
| vet-r08 / format-r08 | 0 / 1.578s；0 / 0.109s | vet/格式diff空。 |

**计数口径澄清：** 作者29/87为纯文本stdout中全部PASS行数，其中2/6行来自父t.Logf回显的owned child输出。去除回显后为27/81个父进程测试/子测试完成记录。它们不能直接叫作此前JSON的29/87个独立测试事件；实际命令exit0、目标回归及child真实wait不受影响，不据此追加源码finding或重开返工。

原R07暂态Destroy、idle Close、持续Acquired、early-hard/close-blocked及R06/R05错误保留仍在本版普通/race实际执行。沿raw核16条关闭mode记录；12份故障JSON/inventory对应实际child PID/exit86，4个idle成功。父测试接受预期86不等于资源成功，旧R06实际PG关闭FAIL也不因离线回归转绿。

新binary **32800117B / C9AFE05D0C48B0A0D126DB2971B958F5F72836A2C49A42A9977A3C9FD8866B71**独立复算。build输出路径、core实际参数/五hash和先构建后运行关系一致；core CWD为本R08/core-sandbox/a/b/c，最终子证据均在R08范围，非新worktree。不冒用R07/R06 binary；摘要属交接现存实物核证，不虚构每次执行瞬间的独立binary签名。

实际文本相等的必要函数包括：pollUntil、ReadOnlySQL、budget parser、stopLocked/collect/context-only、resource reader/helper/classifier、pool config、cap校验、monitor/expiry、awaitResourcePool/closeResourcePool/closeProbe及pgPerfClosureEnd。共同提取范围的摘要记录在audit；declared manifest摘要仅作其自身提取口径保留。diff没有采集/身份/只读SQL/15/5/20/clock/resource/ledger/phase规则变更，R07已接受的其他部分复用。

## 交回边界与STOP

**CLOSE-R07-F001 CLOSED，固定R08有界静态code Gate PASS；无新增confirmed finding。** 原R07 CHANGES REQUIRED报告保留历史字节。Lead现场检查/接收与后续运行授权独立，不由此PASS替代。

P0当前版真实PG负例尚未执行；R06两正例仅历史有限佐证，第三exit86及第四NOT RUN不变。后续最小只读闭环沿原报告条件：同版准确成功前置、真正15s/更早旧START+20s取消、完整argv/source/binary/wait与正常有界归还/外部scope，不自动增加独立成功smoke。growth复用安排仍沿已读补充，实际物证充分才判断，不因入口名称新增门。

原6closed/12pending、旧完整性能NRU、large/canonical NRU、任务active保持；不核销执行finding、不创Tnew、不批准PG或完整性能。本次仅代码窄复审，不评在途runner。完成 **STOP**，report/verdict/audit最终摘要与actual Reviewer STOP见artifact-sha256.json。
