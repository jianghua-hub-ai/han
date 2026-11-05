# R07关闭delta独立有界CodeReview

**代码Gate：CHANGES REQUIRED；唯一新增finding CLOSE-R07-F001（MEDIUM）：末尾goroutine核验可在共享关闭deadline已过后返回成功。无确认HIGH。** R06即时Acquired快照误判的修复方向与真实puddle离线证据成立；归还等待、Close本身已显式有界，但末尾直接消费者还有这一处迟到成功缺口。

结论仅覆盖R06→固定R07关闭delta及必要消费者，非全库/原18项重审。R06物证的整体BLOCKED、两正例有限接收、第三exit86/第四NOT RUN保持；当前R07尚无PG结果，原6 CLOSED/12pending与旧完整性能NRU、任务active不变。

## 对象和两轮范围

objectType=`code`，原非作者Reviewer，固定Han/main/指定HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。Role/add-code-review及适用规则复用有效正文。唯一写集为本code-review-close-r07目录；未Go/LOCAL/PG/SQL、未运行作者runner/binary、未读凭据、未源或旧物证改写/任务/采用决定/Git/清理/委派。

正式对象 `author/probe-close-r07/source-freeze-r07/manifest.json`，captured2026-11-04T09:18:49.546774+00:00，source/local STOP、PG NOT OPENED。H **223792B / 9F08F70B2E2063D5F178064F5EA0D9F84DABB1BA625CEF90402DE70D4CB7CAC2**；基线R06 H **214370B / 550B16614224A13F31CEE931C016C3B6841F59E80F27CE2A61F24720219392E7**。五snapshot及五before摘要自行核；另三Go/docs保持原值，旧global/window/policy/small四摘要匹配。未以live源替代快照。Lead现场live、范围和Manifest7checks仍由Lead接手，本Reviewer未运行。

Round1完整读独立diff与作者harness-r06-to-r07.patch、新增测试/报告。离线校验9个hunk上下文并重建，与固定快照LF文本完全一致；核共享closureEnd、tx/settle/各pool/probe/末尾goroutine、新真实puddle回归及fixture迁至R07。

Round2仅沿closeResources→reclaim/waitReturned→awaitResourcePool/closeResourcePool→closeProbe/pollUntil的真实调用及必要依赖核所有权/截止/失败输出，反查R06错误保留和只读入口。消费growth-reuse-addendum并核现有FaultSeedDisk入口映射，不扩源码或新增规划审批。

8条command-index记录均独立核result/raw SHA、actual wait、前后五hash和事件；其中6条最终同版。核新binary构建→core sandbox路径、16条关闭模式child实证以及真实依赖；详见verify_offline.py/evidence-audit.json/selected-events.json。Reviewer仅Python离线读/hash，不是独立运行Go回归。

## CLOSE-R07-F001 — MEDIUM：末尾goroutine检查绕过已到期的共享deadline

- **位置：** 固定H **1143～1150**；直接消费者 `pollUntil` **2444～2463**，尤其2449～2454。
- **触发：** 各pool返回/Close在D=closureEnd之前结束，随后正常调度延迟或同步证据IO使最后goroutine检查在D之后才运行；此时计数已回到baseline，可能也是D之后才回落。此路径不需改变采集/配置，也不需新场景。
- **行为：** closeResources传了已过期的ctx，但回调仅比较runtime.NumGoroutine。pollUntil先check，ok即直接return nil，ctx.Done与时间检查都在其后。因此末尾不会t.Errorf；如果此前无错误，resources仍可failed=false，尽管这次核验已跨共享5s/更早hard截止。记录closure_elapsed_ns本身不能让该超界自动失败。
- **依据/边界：** 本轮明确要求同一5s或更早hard传播至goroutine且不抹迟到失败。预算合同也要求恢复调度后按真实deadline拒绝迟到成功，不把OS调度不可保证当延期授权。这不是要求硬实时，也不是声称所有资源实际晚归还；缺陷是已到期时仍把“现在计数符合”接受为有效关闭成功。没有声称此反例已发生在作者raw。
- **现有测试为何不足：** awaitResourcePool回调已显式核时间，Close也核completedAt.Before(deadline)，所以暂态/持续Acquired/Close阻塞各测试有效；但这些测试没有走“最后goroutine观察时共享ctx已经过期且predicate=true”的closeResources接缝。core57pass不排除这个顺序缺口。
- **最小关闭条件：** 由原Backend仅在末尾关闭消费者补真实deadline/ctx核验，截止后不能接受success；无需全局改pollUntil或重构资源框架、不加预算。有限离线回归覆盖：expired+count已符合仍失败、count在截止后才符合不能清除失败、截止前符合正常成功；保留既有Close持续阻塞86/错误保留。固定小delta后只复审此项及直接回归，不要求另跑PG或重开全部旧raw。

静态逐步反例见finding-trace.json。无其他确认新增实质finding或HIGH，不代作者修复。

## 已成立的关闭修复与证据

本机真实固定依赖（已读相应正文并记录文件SHA）：go.mod/go.sum为pgx v5.11.0、puddle v2.2.2，puddle已是原间接依赖，本次仅H测试直接使用。pgxpool/conn.go:32～37 Release的closed/busy/nonidle分支调用Destroy；puddle/pool.go:72～76另起goroutine，687～694先destructor再移除资源/释放计数。pgxpool/pool.go:314～325 destructor等待CleanupDone；pgxpool Close调用puddle Close，后者等待destructWG。由此真实Release已调用不等于Acquired立即归零；依赖内部15s并不扩张本项目关闭5s。

| delta | 独立判断 |
|---|---|
| 单一closureEnd | H1098～1099只创建一次min(start+5s,更早hard)，tx回收、settle、业务可关闭分支、disk/probe及末尾ctx都传此值。没有逐阶段另赠5s/15s；末尾有效性受上述finding限制。 |
| 归还/异步Destroy | H1194～1211按同一deadline等待真实acquired0，失败保留return记录并86；不删除Acquired守门，不sleep凑成功。真实puddle Constructor/Acquire/Destroy+屏障回归，不是模拟计数。 |
| 有界Close | H1229～1258在明确owned goroutine调用Close、h.group记账/完成channel同步；记录completedAt并要求严格早于deadline。超界保留证据当前child86，不返回遗弃工作线程。持续占用/Close阻塞不是被改成成功。 |
| probe/SQL/tx | probe先等异步归还，再以共享deadline做实际backend SQL，defer有界Close；tx回滚/原settle用min(原3s,剩余closureEnd)。业务pool未归还分支原t.Errorf/连接处置不被改绿。 |
| 取消与原错误 | 纯取消可做关闭，不以root已取消跳过cleanup；真实resource.failedOwn/安全分类保留。R06 helper/readDiskObservation/error classifier文本未变，同版8组合再次实际到达。PROBE-R05-F001不重开。 |
| 不变合同 | collector/stopLocked/context-only/budget parser/ReadOnlySQL函数段与manifest摘要逐字核同；readDiskObservation、pool config、资源cap检查、expiry/monitor亦逐函数核同。15/5/20、身份/准确SQL/准入、clock/resource/ledger/caps与phase规则未改；没有枚举fallback。 |

行为RED red-async-r01 count3父exit1/4.218s，三个实际owned child **22288/43672/3372**各wait86，原即时快照拒绝真实异步Destroy。它是带回归的中间版本，不冒精确原R06全文，也不是PG故障复现。green-async-r01属较早hash，不借给最终版。

| 最终同五hashleaf | actual wait / wall | 核证 |
|---|---|---|
| green-close-r01 | 0 / 9.609s | 20pass、0fail/skip，含子测试。 |
| race-close-r01 | 0 / 35.094s | count3合计60pass、0fail/skip，非60再乘3。 |
| core-r07 | 0 / 11.281s | 当前binary在R07新sandbox，count3合计57pass、0fail/skip。 |
| build-r07 | 0 / 6.578s | 同版integration/race binary。 |
| vet-r07 / format-r07 | 0 / 1.812s；0 / 0.125s | vet/格式diff空。 |

最终普通/race共4次暂态Destroy child实际wait0；四关闭mode合计16条记录，12份故障JSON/inventory逐份核真实child PID/exit86、4条idle成功。persistent-acquired按原5s失败、early-hard/close-blocked按约150ms逻辑截止失败；父进程观察耗时含启动/退出费用，并非精准150ms。parent PASS只证明正确识别预期child失败，不能当资源成功。

R07 binary **32780533B / B7579CF4D1F9626F663DCDDD878D3ECA36A6AEDAF4C244BA10598B08A216590A**已独立复算，build路径/五hash/先构建后core执行一致；不是R06 binary。core CWD及最终child证据均在R07或其新sandbox，不是新worktree。摘要属交接时实物观察，不虚构每次执行时另捕获的binary签名。

## 下一只读P0：最小证据与成功smoke判断

**当前代码finding先交回；以下仅说明证据充分性，不批准PG或启动窗口。** R06两正例只支持R06实际运行的有限成功事实；R07采集/身份/入口正文未变可复用为历史佐证，但不改标R07实测。

在关闭缺口修复、实际新freeze/本版检查/Lead接收及运行许可满足后，R07后继版本的 `total-fifteen-failure` 和 `earlier-freshness-failure` 两个真实负例，可构成下一只读门的最小闭环，**不因“缺独立成功smoke”自动加跑第三条正例**：更早freshness组本来先做同版完整准确SQL成功采集，随后老START+20约束新注入；两负例都实际消费新的正常关闭链。

充分条件是raw真实证明目标阶段到达、对应最早deadline、取消后不发布/不续freshness、正常有界资源/SQL/backend/goroutine归还，真实预期测试exit1而非关闭86、完整argv/PID/wait/source/binary及normal/failure分支证据。尤其earlier组须有其同次成功前置的准确bytes/START/finish和新的更早截止。若关闭仍86、前置准确采集未成功、summary/版本链缺失，则只按实际缺段补证；不能把R06正例拼成当前版完整闭环，也不凭两个命令名字宣布满足。

这些条件仍是原最早deadline/资源归还和证据链，不是新增独立命名入口、固定重复次数或新性能窗口。不能用只读局部闭环核销growth/leak/九语义/原12finding。

## Growth现入口复用补充

实际读 `test-next-gates-r01/growth-reuse-addendum.md`，摘要见evidence-audit。与固定H **5506起 FaultSeedDisk_CancelsDuringSeed**、newHarness的testDiskLimitDelta、准确guard、beforeSeed/monitor/seed映射一致：可在一条真实阶段内完成初始准确baseline→真实seed/追加增长→观测触限取消→外部静止SQL核原账。**不要求新增独立growth正向函数/程序。**

注入cap对应初始准确DB大小+2MiB，再受正式tier cap更严者限制；它不重置global/tier，也不是瞬时最多增2MiB。DDL/迁移占用必须计账，不能把全部差额称正文增长。现入口追加loop每批10行只有最终batches计数，后台周期monitor并非每批同步SQL；若seed自身先触限，可能尚未走到该终点记录。只有实际新物证才能决定增长子条件是否闭合。

最小接受链沿补充原义：明确写前准确SQL和阶段；真实写到达且有可关联的增长后/触限准确样本；触发原因确为cap而非SQL timeout/freshness/前置失败；取消/后点停止；同次client wait、所有写者静止后外部准确SQL/身份/schema/backend核同一global/tier累计差额并解释B_end与B_hit差别。现有同次物证若已满足即可分别接收增长准确性及停止，不另强增正例；不足才补外部静止SQL或必要关联检查点，不要求每10行加一次SQL。本文未取得或核销任何新growth执行结果。

## 交回与STOP

仅 **CLOSE-R07-F001（MEDIUM）**需最小返工；其余已核关闭修复及同版局部证据有效。原作者修复后只核真实delta与末尾deadline回归，不重做全部R04/R05/R06 raw或18项。Lead的Manifest7checks独立进行，不能覆盖该控制流缺口。

本轮不授权执行、不选采用、不更改任务；R06实际运行BLOCKED及历史exit86保留。原6closed/12pending、完整旧NRU、large/canonical NRU、任务active不变。两轮有界复审完成，**STOP**。固定report/verdict/audit摘要和actual STOP见artifact-sha256.json。
