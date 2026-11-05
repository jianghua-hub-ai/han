# R09实际leak独立执行证据Review

**结论：局部leak Gate PASS；S24-CODE-R01-F005 CLOSED。** 无新增阻断finding。原18项累计7 CLOSED/11 PENDING，其他finding未扩大核销。实际Go自然exit1/FAIL原样保留；本结论接收泄漏负例的目标命中及正常有界清理，不称Go测试GREEN、完整性能或E2E通过。

## 固定对象与两轮实际范围

本次只审`author/followup-attempt-r01/le-113346-18724/`、直接grant/版本绑定、r02→r03外层预检差异，以及原F005最低关闭条件。固定Han/main、HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca为当前授权边界；未执行Git自查分支。Reviewer非源码作者及PG运行者，独立沿add-code-review两轮方法，全文补读自身Role、Skill及适用清单/抽象规则；复用既有R09 Code Gate与growth独评，不重做其全量内容。

第一轮核实际leaf：start/grant、preflight/snapshot-observation、launch/PID/actual-exit、stdout/stderr、e目录全部JSON、material和final；独立扩展长路径枚举、逐文件重算SHA/bytes并比较material内值与原件。核五份冻结snapshot/live摘要、binary实际文件、policy、grant与source-before/after的一致性。没有将作者report/execution-index或final机器门作为独立结论。

第二轮沿直接消费者核事务和清理：冻结H的FaultLeakedTransaction、beginOwnedOptions、OwnedTx.Rollback、reclaimTransactions、waitReturned、closeResources/closeResourcePool；只读实际pgxpool v5.11.0 tx.go的Rollback/Release语义及go.mod依赖。读取10组真实只读query的launch/pid/actual-exit/result，由stdout重新解析payload，核精确app/PID/start、实际wait、静止SQL、完整catalog/inventory。比对r02→r03 runner精确diff及require谓词，核本次采用文件和先前预检拒绝。未扩展为全库或新一轮源码Review。

自有offline-audit.py仅做离线读取/哈希/AST比较，**111项核证均成立**。原件清单及SHA、指标和检查结果见evidenceaudit.json；窄工具diff另存runner-r02-to-r03.patch。所有消费原件在核证结束时再次重算无漂移。本次唯一写集为本目录；没有Go/PG/SQL、作者helper执行、凭据读取、源/任务/旧物证修改、Git、清理或委派。

## 版本、准入及实际FAIL

| 绑定 | 独立重算 |
| --- | --- |
| H冻结snapshot及live摘要 | 241407B / C3DFDC66FDBFB025760ED78B0E0061CC49BD7AFB2DA0BF76685D94D0D89B8B99 |
| 其余三Go/docs | 五份snapshot/live各自SHA/bytes全部匹配R09 manifest |
| source manifest | 18B99D5058C82EB282579535AE02F9D64B17E6EA981CA409D6BF1AE354364C02 |
| 实际非race binary | 20050944B / 71B3EF47D466A8EAB899C266311EAD224E5F5316D6B244A8E2D7EFDF705521D5 |
| 当前policy | 7C94C0DAE104A177FE82E976FD12154F97880ED749CEFBFEFCE572EAA4A33E04 |
| 本次actual grant | C8A20432094764C19C169413BB580EA23410DAA1720C7FCCC4D900E13FEE2FEC |
| 实际r03 runner | 22DE74DD17163FA3BA36256ACCFFD0FA3B4E825BCC1291219D27FEACB15E15D4 |

grant创建11:33:30.606226Z，先于11:33:46.356595Z start及Go启动；source、binary、policy、前置growth接收及live-accounting采用文件的hash引用均匹配原件。实际入口是`^TestPgPerfR02_FaultLeakedTransaction_BoundedCleanup$`，count1、test.timeout5m，CWD真实`test/integration/article`，INJECT_FAULT=1。没有凭据值落入argv/公开env，仅记录敏感env键名。

Go PID35312从11:33:48.598622Z到11:34:00.924891Z，真实wait、自然exit1、forced=false。stdout仅有目标相关FAIL：先记录真实synthetic owned transaction intentionally left，随后owned transactions leaked before cleanup:1；无前置SQL timeout、cap/freshness错误、reclaim错误或关闭失败混入。内部资源记录failed=true符合泄漏guard应报FAIL，不能因外层final.failed=false而覆盖。

converter35748在Go退出后11:34:00.928446→01.977664Z自然exit0，仅归集Go事件；其exit0及归集时间不替代被测Go结果/业务时间。外层final11:34:05.367464Z problem/closeout_error=null、failed=false，意义仅为预期负例目标与归集收口机器门满足，仍需本次独立接收。

## 真实事务、Rollback与F005关闭

原R01报告F005（HIGH，旧H1045-1083）指出：AcquiredConns非零仍同步Pool.Close，且取消ctx不自动回滚pgx事务，导致后续资源日志、残留检查及有界join不可达。最低复审要求是显式tx/worker所有权和回收、真实故意遗留连接/事务分支有界结束并留证，不得仅把Close放到无人等待的goroutine。

原R03新增证据Review已接收取消worker/release/join子条件，F005当时仍PENDING的具体原因是leak在前置probe超时，未创建事务。本次没有复用那个空资源状态。R02与当前R09的CanceledWorkers_ReleaseAndJoin测试body离线逐字一致；历史PG成功仍标历史，当前代码/关闭行为沿既有R09 Code Gate，不冒充本次重跑了该worker场景。

当前冻结H5872-5886的实际负例在newHarness后执行h.beginOwned(h.ctx)；beginOwnedOptions（1050-1078）真实pool.BeginTx并将wrapper登记到ownedTx。随后tx.QueryRow("SELECT 1").Scan成功才打印遗留marker，函数返回而不主动Commit/Rollback。cleanup实报ownedTx=1，证明到达真实事务遗留检测，而非只输出固定错误字符串。

closeResources（1091起）取消root/observer、join worker后，实际进入reclaimTransactions（1204-1221）。它先快照1个owned事务并报guard，再以min(now+3s,同一closureEnd)的独立context调用每个tx.Rollback。OwnedTx.Rollback（1026-1044）调用底层tx.Tx.Rollback，再清登记；实际pgxpool/tx.go37-43在底层回滚后Release关联连接。原stdout未报reclaim错误。随后waitReturned（1195-1202）真实谓词同时要求ownedTx为空及AcquiredConns=0；pre-close的return_error为空、acquired=0，之后真正pool Close完成、外部backend为空。

**证据强度边界：没有独立逐tx Rollback结果JSON、独立tx_after数值或wire级SQL回滚轨迹。** 本次能由已固定且不可替换的实际二进制/控制流、真实before=1 guard、未报回滚错误、成功waitReturned谓词及后续Close/外部空集合，联合核定必经Rollback与归还完成。不把缺失字段写成已有raw，也不将“map清登记”单独当物理连接已归还。live轮询未单独捕捉这段短暂泄漏瞬间的idle-in-transaction/PID，不声称捕捉到了；它不构成原F005最低合同之外的新抓包门。

因此原F005缺陷最低合同闭合：本次真实遗留连接/事务可以触发FAIL，再正常清理/关闭并生成后续完整物证，无需外层超时杀进程。**S24-CODE-R01-F005 CLOSED**；这不是对任意未登记第三方资源或所有不合作worker时序的无限担保，其他未执行业务finding不随之关闭。

## 同一5s关闭及精确外部所有权

| 项目 | 原件结果 |
| --- | --- |
| closure start / deadline | 11:34:00.8803883Z / 11:34:05.8803883Z（共享原5s） |
| 总closure | 16051400ns = 16.0514ms |
| pre-close | acquired0、return_error空，return_wait549000ns |
| business / disk / probe-close Close | 完成、各自acquired_after0，全部在同一截止前 |
| resource observer | root_canceled=true、workers_joined=true、resource_failure=false、terminal=nil |
| 总资源 | goroutine2→2、residual_backends0、无fail-stop、无forced |

实际business Close仅545500ns，disk532800ns，probe-close520900ns；没有给各pool另开5s窗口。关闭goroutine被等待、记录完成；最终goroutine回到基线，没有把后台悬挂伪装成功。

两个外部live快照共4条记录，去重后为同一业务app`han_pgperf_r04_fu_lea_113346_18724`的3个身份：

| backend PID | backend_start（+08） |
| --- | --- |
| 33580 | 2026-11-04T19:33:48.823602+08:00 |
| 42556 | 2026-11-04T19:33:53.825225+08:00 |
| 15996 | 2026-11-04T19:33:55.849565+08:00 |

query6在Go退出后查询精确业务app为空；query10查询业务app及本次observer前缀为空，排除自身，observer自身另有自然exit0/真实wait。没有把observer自排除结果冒充全Host无连接。10个psql客户端+Go+converter共12个owned客户端，与final逐个PID/exit对齐，全部真实wait、无forced/unresolved。各query的应用名、backend_start和客户端launch/wait时序匹配；query2/query7 active_other均0，前后服务器/数据库身份一致。

## 准确核账、全量inventory与窄工具差异

准确before全库SQL=2,104,252,095B，与fingerprint初值、preflight.actual_live一致；当前historical expected也是该值，所以本次drift=0。Go退出后query8于11:34:02.406140→04.958847Z取得准确静止after=2,105,743,039B，较本次before增1,490,944B。原schema构建/迁移与全部物理增量继续入账，未clamp或改base。

旧84个schema全部保留，catalog-after85；唯一新增`han_pgperf_r04_sr2leak_336a400e1247dcd8`，实际catalog差集、final、inventory、created/migration和资源身份一致，retained=true。旧growth新增schema及更早失败集合均包含在旧84中，没有清理/移动或重复记同一个schema。

原small ledger实际80B、SHA`2B368282C7667FC2B3563F82FC293FBBEEE59A350A52CB41D08C57BDB87FC4D2`不变，resource Window仍`2026-11-04T03:29:04.586115+00:00`，global/tier origin1,406,932,671B。最终累计2,105,743,039−1,406,932,671=698,810,368B；small剩1GiB−累计=374,931,456B。PGDATA available45,640,740,864B是独立物理卷读数，不替代small累计cap；该余量只表示本次收口，不能承诺后续阶段容量。

先前`le-112621-42188`因历史snapshot2,104,227,519B到live2,104,252,095B的24,576B差额拒绝，保持FAIL、无Go launch/新schema，旧84保留；差额根因unknown，不编造autovacuum归因。独立重算其final hash与新grant snapshot_source匹配。

实读Lead已采用的live-preflight-snapshot-distinction-r09.md（SHA50E70C30…C810B283）及r02→r03精确diff：只将`size==expected_database_bytes`由准入等式改为snapshot_expected/actual_live/drift/reason记录；原inventory完整相等、active_other0、small/medium/total累计谓词、卷reserve、source/binary/policy/单stage绑定与关闭/目标门保持。AST对比require谓词仅该一处变动，其余相同。实际本次snapshot drift=0，**没有真实PG执行非零漂移准入分支**；对该新增分支只有精确静态mapping及已采用合同支持，不伪称本例覆盖所有差额/越限边界，也不重评无关脚本架构。

## 累计状态和停止边界

| 原finding | 本次后状态 |
| --- | --- |
| F001/F003/F004/F008/F010/F018 | CLOSED沿用 |
| **F005** | **本次新增CLOSED** |
| F002/F006/F007/F009/F011/F012/F013/F014/F015/F016/F017 | 11项PENDING，未实际达标不改 |
| E001 | CLOSED沿用，不计入原18项 |

N9尚未授权/执行接收。本attempt完整calibration/small/medium已经永久NRU；原11:29完整性能NRU、large/canonical NRU保持。本局部PASS不追溯恢复full-start资格，不缩原15/5/20或point121/60/tail1、不批准后续运行。唯一Tnew仍10:22:37.932710Z，所有返工和Review计原elapsed；12:47:37.932710Z停新、12:52:37.932710Z硬停不延长。未改Tasks/Plan，S2.4 active。

报告、verdict、audit及自有离线脚本/差分的最终SHA和Reviewer actual STOP见index.json。旧报告及作者物证全部只读保留。
