# S2.4 combined probe R05 独立有界 CodeReview

**固定代码 Gate：CHANGES REQUIRED。CLOCK-R04-F001 CLOSED；新增唯一实质 finding PROBE-R05-F001（MEDIUM）：正常关闭与真实 Acquire/SQL 异常并发时，reader 适配器可能把异常覆盖成纯 context 取消，使资源失败漏报。无确认 HIGH。** 本结论只覆盖 clock-r04→combined R05 的 budget delta、F001 与必要直接消费者，不重开原18项全量源码评审。

静态评审、作者同版离线回归与真实执行门分别判断。当前无 PG 运行证据；P0 的准确SQL/增长/取消/归还尚待真实验证，Tnew 未产生，原12 pending 不自动核销，旧批次完整性能 NRU 永久保留，S2.4 active。

## 固定对象和边界

objectType=`code`；固定 Han/main，指定 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。原非作者 Reviewer，沿有效 add-reviewer Role、add-code-review 和适用规则继续，未重新 Route、未委派。

对象：`author/probe-budget-r05/source-freeze-r05/manifest.json`，captured `2026-11-04T08:06:47.304659+00:00`，source_stop/local_stop=true。H **209145B / E97318D68C8BE670DF11A196D469A3D91F5BAC979A1CF65EFDDCBAE4F09DAFC7**。已审基线为 clock-r04 H **168839B / B8A395DE6F1E50A6AC55DEC9E4D4B3AB87EEA5AD1295C138AAB8D4B9D5B6158E**。

五 snapshot 的 bytes/SHA 均自行复算一致；W/R/N/docs 保持原摘要，详见 pin-verification.json。旧 global/window/policy/small 四文件再次复算与 manifest 相符；未改原 origin/base/80B small。未读取 live H 代替快照；Lead 的现场 live/source/范围检查仍由 Lead 负责。

完整消费已采用 budget-r02（4B5D2F9A0BE894E2D478ECAB0B6E4661DF72E0BB140FFD0909735EF5D924208A）、Lead budget adoption、clock-r04 §4.1/采用与原 code-review-clock-r04/report。作者 report 后续到盘后亦实读，其摘要记录在 evidence-audit.json；作者 STOP/自检不充作独立执行结果。

唯一写集为本目录。只离线读取、Python hash/差分/JSON核证；**未运行 Go/LOCAL/PG/SQL、作者 runner 或测试 binary，未读凭据，未 Git/清理/源写/作者物证改写/Tasks/Plan/委派。** 离线脚本和输出可复核，但不称为独立 Go 回归。

## 两轮实际范围

Round 1：完整阅读 r04→r05 的新增/修改实现和回归，独立生成1628行 diff；把作者 `harness-r04-to-r05.patch` 的31个 hunk 在内存校验上下文并重建，与固定 R05 snapshot 的换行归一化文本逐行完全相同。检查 strict budget policy、资源状态机、collector、初始化、seed/clone、monitor/expiry/关闭、统计、F001 metadata 和新增只读入口。累计 r02 patch未作为重开全量的理由。

Round 2：沿真实 policy→init→准确采集→admission/BeforeAcquire→operationCtx/point/seed→root monitor/expiry→close，以及 baseline/receipt→实际 fingerprint 的生产者/消费者核状态所有权和失败传播。W/R/N仅定位原直接调用，仍经统一 newHarness、point/operation context、owned transaction，不重审其未变业务实现。核只读入口直接复用的 pool config、collector、SQL与关闭，不扩生产库设计。

21条已完成 command-index 记录逐项核 result SHA、真实 wait exit、stdout/stderr SHA、前后五 hash，独立计算 current 标记；没有把早期版本绿色借给最终版。最终 raw 的状态/owner/Closing/F001事件和实际文件另核，详见 evidence-audit.json、selected-events 及 verify_offline.py。

## Finding 与有限关闭条件

### PROBE-R05-F001 — MEDIUM：正常 root 关闭可覆盖真实资源 SQL/Acquire 错误

- **位置：** R05 H `readDiskObservation` **1854～1858、1867～1871**；消费链 `stopLocked` 1677起、collect **1794及1801～1805**、closeResources **1115～1122**。行号均为本次固定 snapshot。
- **真实触发条件：** observer 的 Acquire/SQL 已返回非 context 错误，例如服务端权限/IO SQLSTATE；在 reader 随后检查 ctx.Err 前，正常场景收尾调用 h.cancel。该并发并不要求新增接口或改变 SQL，也不等同于纯取消导致的正常驱动错误。
- **当前行为：** 两个分支只要 ctx.Err 非空就直接返回 ctx.Err，丢弃底层非 context 错误。collect 虽专门保留 Closing 期间的非 context reader 错误，此时却只看到纯 context.Canceled。若正常 root 关闭先产生 terminal，其 failedOwn=false，后续也无法升级；ReadErrorClass 同时丢失原 SQLSTATE/type。关闭汇总因此可能写 resource_failure=false，不对该实际 observer 异常报错。其他独立失败仍可能令整条测试失败，不能据此说任何取消都导致假 PASS。
- **依据：** budget-r02 §3/§4要求真实采集错误保留FAIL、point成功不能抵消harness资源失败；新代码自身 collect 的 Closing 非context升级分支也明确承接该义务。不能用 root 已正常关闭为理由覆盖同时发生的真实错误。
- **为什么现有 GREEN 不关闭：** `TestPgPerfR05_ClosingKeepsUnexpectedReadFailure`（4900起）直接向 s.collect 注入 reader 错误，绕过真实 readDiskObservation 的 ctx.Err 替换分支。它证明 collector 能保留收到的异常，未证明实际适配器会把原异常送达。当前raw没有证明本竞态实际发生；本 finding 来自固定源码的可达控制流，见 finding-trace.json。
- **最小修复/关闭：** 原Backend仅在H保留底层非context Acquire/SQL失败与取消状态的安全分类，送达现有 collector/关闭汇总；不要为此改policy、owner、窗口或新增资源框架。保留既有凭据脱敏，不能直接把潜在敏感错误原文写raw。有限离线回归须覆盖实际适配器或其真实调用的窄归一化helper：已取消ctx＋非context Acquire/SQL错误仍使resource_failure=true并保留安全错误分类；纯context正常关闭不误报；Closing原测试继续有效。固定新delta后只复审这项及必要受影响路径，无需Reviewer运行Go/PG，不要求重跑全部旧raw。

没有确认其他新增实质 finding 或 HIGH。不把下列真实运行缺证据另包装为源码bug。

## CLOCK-R04-F001：CLOSED

H **1978～1983** 调用 pgPerfLoadTierBaseline并存入h.actualBaseline；**2034～2038**真实初始化fingerprint输出 `resource_actual_baseline`，原policy.Resource仍单列授权声明。H **3711～3782** 从实际校验的ledger字节返回origin/tier/path/Bytes/FileBytes/SHA和receipt身份，3785起receipt核验返回真实摘要；LoadState区分首次与既有复用，未回写不可变policy。

最终同版 `ActualBaseline_FirstReuseAndBoundReceipt` count3通过真实fingerprint writer落盘。独立沿raw定位3份 fingerprint，每份3 harness，逐份读取实际 medium ledger/receipt、复算文件长度/SHA，首次基线1406932771、同policy current+300及后续existing current+500均复用原有效基线和摘要。前两条授权仍unregistered/Bytes0，后条为显式existing；实际观察另列。原small四摘要核证不变。该关闭仅是clock证据消费者修复，不是新PG首登、容量或原12项执行核销。

## 其余已核 delta 和限制

| 路径 | 独立静态判断与本版证据 |
|---|---|
| policy，H1474起 | 全缺新组保持5s/5tick兼容；新组要求显式15000/5000/20000且固定采用路径/hash匹配。partial/null/空/错数/错采用及退役detection字段拒绝；保留clock的execution/resource分离和旧账验证。 |
| collector，H1673～1825 | 以保留单调部分的time.Now为请求起点；可取消单飞等待、Acquire/SQL共享最早deadline；旧freshness=last完整success START+20。首次UNKNOWN拒绝准入，完整及时结果才原子发布，terminal优先、late不复活；取消hook停止或等待完成。真实adapter异常传播受新F001限制。 |
| expiry/admission，H1246、1706、2131 | 独立expiry只持短状态锁，不等待flight token；准入自己再核严格早于freshness、root/ctx和terminal。raw受控0/14.9/15/20/29.9及19及时成功分支各count3；20ms expiry/10ms取消是明示缩放fixture，不冒真实20s OS/SQL证明。 |
| 初始化/准确SQL，H1903起/1845起 | 初始完整资源观察后才有成功值；载入旧global/tier baseline，再同步guard；pre-DDL受seed更早deadline。仍是准确全库pg_database_size，没有枚举cache fallback或物理bytes预分配。真实慢SQL可消耗大部分20s新鲜度，使后续同步guard更早失败；这是真实成本/执行风险，无自动续期保证。 |
| seed/clone，H3169、2571及各CopyFrom调用 | CopyFrom最多1000行一批，保留全部原行；每批同步guard及operation准入，生成循环定期取消检查；clone表边界guard。观测达到cap停止，超过cap记录原值失败并cancel root；卷预留检查保持。未证明真实增长/取消响应或保证瞬间不越cap。 |
| owner/point | root observer属于diskCtx/h.group，tick5s且合并，不因正常phase结束而取消；phase自有同步collect受其phase更早期限。phaseContext/context-only/outcome及pool/DB配置函数文本与r04未变（脚本逐函数核）。point仅新增准入；121/60/tail1、真实10/30和重复不削减，业务样本未改成sleep补窗。 |
| 统计，H3184起 | cached DB bytes仅指标，标记原成功start/finish及等待，不刷新success/freshness，不冒guard。point结束后统计未延长业务measurement或修改已取pointWall；observer/统计费用仍在场景/attempt。 |
| close，H1090起 | 先root/disk取消，join沿既有5s及更早hard截止，无额外15s尾部；资源pool/probe acquired非零时fail-stop而非无界Close。非合作owned-child真实wait86、UNKNOWN/inventory留证，各count3；已读取并核3份PID/exit/inventory。真实PG连接/backend/goroutine归还仍未执行。 |
| 只读入口，H4926起 | 连接前要求既有han_perf_s24及loopback:5433配置，设置default_transaction_read_only；连接后校验PG180006/DB/只读/地址。未调用newHarness/EnsureDatabase/Migrate，SQL仅身份/准确size/显式pg_sleep；无DDL/业务DML。自然3次、controlled-six与超时组分账，延迟和injection_reached明记，预期资源负例仍真实FAIL。复用同一resource.collect/readDiskObservation/close链，仍受新F001影响；仅list/编译，未PG运行。 |

## 同版证据和历史失败

| 最终同五hash leaf | actual wait exit / wall | 实际结果 |
|---|---|---|
| race-budget-final | 0 / 33.954s | count3共243个有Test的pass事件，含子测试，0fail/skip；不是243再乘3。含r04兼容/父子及R05状态/owner/Closing/F001。 |
| build-core-final | 0 / 5.812s | 同版integration/race binary构建。 |
| core-race-final | 0 / 10.906s | count3共57pass、0fail/skip。 |
| format-final | 0 / 0.125s | diff空。 |
| list-final | 0 / 4.437s | 含ReadOnlySQL；列举不等于PG执行。 |
| vet-final | 0 / 1.829s | 静态检查。 |

其余15条current=false，旧绿色和旧失败原样保留。red-policy-r01 exit1/3.656s实际到达partial预算误接受，是行为RED；local-evidence-r01 exit1/1.812s为把readJSON bool当error的编译失败，不能冒行为RED。较早race 240/243事件不是最终版证据。

binary.json指向 tools/frozen.test.exe，独立复算 **32629456B / C4670AA1471DAE47847EED0E28234111D4943899C0377B752C69C630162C2DB2**；build参数输出与core run路径一致，build完成早于run，五hash一致。core CWD为作者新core-sandbox/a/b/c，使相对fixture落到新sandbox，**不是新worktree/新源码**。binary摘要属交接时现存文件核证，不能虚构每次构建/执行时另捕获的摘要；Reviewer未执行binary。

## 交回与停止

最小剩余修复仅 **PROBE-R05-F001（MEDIUM）**，回真实作者；CLOCK-R04-F001已关闭，原clock报告字节不变。下一次仅核实际新delta、错误适配器与有限回归，不重复本轮全部raw、不重开18项全量。是否采用或接受风险由Lead按现有授权处理，Reviewer不代修、不替Lead采用。

P0独立执行仍需当前实际修订的自然准确SQL三次、受控>5<15分账、超15/更早freshness目标阶段取消及后点零准入、真实增长/观测cap、资源/owned child/backend/inventory归还与版本连续性。未跑不标绿，不为取得这些证据强开完整性能窗口。本次没有Tnew/P1启动授权或性能PASS；原12pending、旧完整NRU、large/canonical NRU及任务active保持。

两轮有界评审完成，**STOP**。报告/verdict及核证文件最终摘要与actual STOP见artifact-sha256.json。
