# S2.4 probe budget r02独立有界复审

**计划delta Gate：PASS。PROBE-BUDGET-R01-001 CLOSED；新增finding 0。** D1明确选择原准确累计观测式fail-stop，D2明确保留root observer所有权；本轮接受其规划合同边界，不代Lead实际采用15/5/20，不接受在途H实现，不核销原代码12项或给运行/性能PASS。

## 固定对象、两轮实读与版本

objectType=`design`；对象 `test-plan-probe-budget-r02.md`，实际18136 bytes，SHA256 **4B5D2F9A0BE894E2D478ECAB0B6E4661DF72E0BB140FFD0909735EF5D924208A**。起审与收口离线核同一摘要。固定 `D:/AID/VSCode/Codex/Han` main，原TestEngineer已STOP/关闭；原非作者Reviewer沿原Workflow续接，不重Route、代修或委派。

Role、add-code-review及适用规则沿已取得有效全文，按非代码合同及真实消费者作有限复审，不要求先实施或先Go/PG。用户已授权建议值继续，Lead实际采用仍是独立后续记录，不再询问同项人工授权。

Round 1：读取r01→r02全文差分、重读新增§2新鲜度状态/首次/新采集/独立到期/反例，及§3～§5 D1/D2/消费者mapping；对照原唯一finding关闭条件和已接受的逻辑反例，核原子成功发布、terminal状态、mono/UTC、跨进程及费用边界。没有重开r04 clock或原12项完整规划。

Round 2：仅复核原freeze-r02 H的直接接缝：checkDiskBudget的diskMu/5s/专用pool/错误返回，root/diskCtx及monitor注册，closeResources的取消/等待，size缓存reader和point真实起止位置；其余初始化/seed定位沿上轮有效已读上下文。新合同明确列出初始化、seed/CopyFrom/事务/ANALYZE、monitor、关闭、统计消费者，不把旧代码当已实现新状态机。

所有定位仅使用固定旧snapshot `author/r02/source-freeze-r02/pg_perf_harness_test.go.snapshot`，131747B / B8D11EBFB3E4D15F01BC15263E1899BF168732CB06B232099B0F1FD9759E8E86。**未读取在途live H正文或其编译/race日志，不混审新旧hash。** 用户所述部分转绿是背景，不是本设计结论的执行证据；后续新H仍须新freeze与代码Review。

再次独立核原budget-r01、其report/verdict、clock-r04及其report/verdict、旧small80B/SHA原字节均保持。离线差分、pin与合同反例判定见 `r01-r02.diff`、`offline-evidence.json`、`freshness-closure.json`。唯一写集为本review-r02目录；未Go/PG/SQL/凭据/Git/清理/源写/Tasks/Plan/委派。

## 原finding逐项关闭

| 原关闭条件 | r02位置与明确合同 | 判断 |
|---|---|---|
| 20s不是变化识别保证 | §1及§2：改为可用观察的新鲜度/停止上限，不从5+15推导变化识别、风险暴露只增加10s或硬实时保证 | CLOSED（规划合同） |
| 从成功采集START单调计龄 | 第32～33行：last_complete_success保留开始/结束及采集区间，deadline=该成功probe_started_mono+20s，读取cache/完成/tick均不续期，UTC仅取证 | CLOSED |
| 首次UNKNOWN不得放行 | 第36行：首次无值关闭DDL/写入/新批，仅按许可采初值；等待与采集共用15s/更早parent，完整成功后才建立anchor；跨进程重新观察但baseline不重置 | CLOSED |
| 新采集不能延长旧freshness | 第38～40行：总deadline=min(request_entered+15s,existing_freshness_deadline,applicable_parent)，包含等待/Acquire/SQL/Scan，不另起15s或以部分Scan更新 | CLOSED |
| 独立expiry与每次准入核验 | 第42行：独立于阻塞SQL/Scan及diskMu的到期控制取消root；每次DDL/场景/有限写入准入自己核mono时间和terminal状态，不等下一tick或scan返回 | CLOSED |
| 完整成功原子发布、late不复活 | 第34、40、42行：仅完整合法且finish严格早于旧deadline/更早期限、root仍有效才能发布新started+20；terminal优先，迟到只能late/rejected/raw | CLOSED |
| 原反例t=20停止 | 第44行：首0→14.9成功仍到20；新采集15开始有效截止20，未完成即EXPIRED/UNKNOWN/cancel/拒准入；29.9不刷新。另区分19合法成功可发布15+20=35 | CLOSED |
| 调度/IO及资源限制如实披露 | 第46行、§4/§6：20为逻辑强制截止，不保证精确时刻获CPU或完成IO取消；记录实际超界/取消/退出，不能晚执行也称履约或多给关闭尾部 | CLOSED |

上述是同一个design finding的分项核证，不新增八个finding。其关闭依据是固定合同及静态调用mapping，不是假定新H已经做到。

原逻辑反例在新合同下的判定：旧成功开始0、完成14.9→freshness截止20；下一请求15+15=30，但有效截止取min(30,20)=20。到20仍在途就停止并进入terminal；29.9的正确bytes不能复活。若19已完整成功，则旧期限内可更新观察起点15、截止35，二者不混用。`freshness-closure.json`只是离线合同判定，没有运行时钟、Go或PG。

## D1/D2及新增边界

**D1已明确选择原观测合同。** §3保留准确pg_database_size、原global/tier累计、真实PGDATA余量和预留；触限停止新点/新批，观测超限如实FAIL并保留delta，资源安全失败取消在途。明确不要求每批表/索引/TOAST物理上界或任意瞬间零突破，不新增影子quota。有限批、必要同步guard、首次/失败/过期关闭门仍在；已有成功值有效期间后台新scan不自动使它失效，在途结果也不预先算成功。该选择没有将未采用的更强义务强加给原12项。

**D2已明确真实owner。** §4将harness observer保留在root/diskCtx+h.group，受15s/旧freshness/root/scenario/attempt更早期限约束，正常phase结束不自动取消它或将其全部关闭算入phase tail。真正phase拥有/同步调用的资源操作才受该phase/point更早deadline，其实际等待/SQL/取消收口计原121/60与累计tail1s。两类费用都计场景/attempt，root关闭立即取消、按既有有界协议回收，不等待剩余15s采满。point成功片段不能掩盖随后harness资源失败。

post-point统计及diskLastBytes只作带真实观察区间的指标，不续freshness、不自动代表guard成功，不改变point结束时刻或混入业务measurement。它与旧H root monitor、scene close及point统计位置的区分一致；未隐式改原采样边界。

未确认新增设计finding。后续实现仍需验证并发发布/expiry/准入之间的竞态：既不能让迟到成功复活，也不能由过时的expiry回调错误终止已合法续期的观察；同一diskMu不能堵塞expiry/cancel。合同已要求原子发布、terminal优先和独立控制，这些是代码实现/回归责任，不因当前代码尚未实现而阻断本次非代码计划Gate。

## 最小实施与执行限制

§5/§6已经把直接H消费者与有限区分性证据对齐：首次准确观察、自然与受控>5<15分账、原反例20到期、coalesced tick/UTC跳变/失败不刷新、真正到达SQL或Acquire的最早期限取消、实际seed增长及观测超限、D1有效值与在途采集、D2两类owner及资源有界归还。受控逻辑回归不冒充真实SQL/PG增长/IO及时取消；exit1或前置失败仍不能代替目标路径到达。

15/5/20为候选，只有Lead实际采用后新policy才生效；新字段应表达freshness_limit而非detection_envelope，不静默改旧字段/旧policy或历史失败。clock-r04的execution/resource-origin、原Bytes/SHA及receipt合同保持；新进程重取mono观察不重置资源账，也不重置共享attempt时钟。

本次PASS不接收H资源origin或预算源码。后续须固定包含真实改动的新H/完整五hash freeze、针对性回归、原Reviewer代码及证据复核、Lead实际采用/运行安排。P0资源增长/取消/收口条件未满足，不运行依赖业务或完整采样；不能在新鲜度无法落实时反复续窗直到变绿。既有证据仅按真实hash与影响复用。

准确全库SQL、同步guard与expiry仍可能增加负载、暂停或提前失败；不能保证原57～71min估计成立，须重估且保持r04的145/150/53门。15s不延长phase/point/关闭，121/60/tail1及10/30/三重复不变；无法在H局部安全履约、费用装不下或资源不足则保留局部限制/正式全量NRU。

原6 CLOSED/12 pending、E001 CLOSED、旧批次完整NRU、large/canonical NRU及任务active保持。原budget-r01 CHANGES REQUIRED历史和clock-r04 PASS均原字节保留，不追溯改绿。复审只关闭本规划finding与接受本版D1/D2安排，无SLO/容量/整卡或E2E结论。

完成有限独立复审后 **STOP**，不等待源码、不调用Go/PG或取用LOCAL/PG窗口。
