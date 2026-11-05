# R09实际growth独立执行证据Review

**结论：有界growth Gate PASS；无新增阻断finding。** 本次接收共享采集、真实增长触限及正常资源关闭的执行证据。实际Go故意保留负例exit1，本报告没有将其改称普通测试GREEN。原6 CLOSED/12 PENDING不变，未执行leak/N9/W不核销，S2.4 active。

对象仅为`author/followup-attempt-r01/gr-111112-25876/`及其直接grant/build/冻结版本/资源origin原件。固定Han/main、HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca为交接边界；本次未运行Git。Reviewer非作者、非执行者，只运行自有离线读取/哈希脚本，唯一写集为本目录。未运行Go、PG、SQL或作者helper，未读凭据、改源码/任务/旧物证或作下一阶段采用决定。

## 两轮实读及版本绑定

第一轮核实际版本、准入和目标：读取source-freeze-r09 manifest及五snapshot并自行复算SHA/bytes；读取binary-r09及其父metadata/build result，核binary实际20,050,944B和SHA、build前后五源hash；读取fresh-snapshot grant、policy、start/preflight/launch/pid/actual-exit、stdout/stderr、全部e目录JSON与material.json。逐份比较原件内容和SHA；未仅信runner机器门。自有offline-audit.py离线核证138项全部成立，原件清单、SHA和指标在evidenceaudit.json。

第二轮限执行消费者与收口：核共享回执与源flight、固定deadline、逐批RowsAffected与typed错误；只读冻结H中的FaultSeedDisk、waitReturned和closeResources对应实现，确认字段的实际生成语义。读取13组外部query的launch/pid/actual-exit/result原件，从stdout重新解析payload，核精确app/PID/backend_start与真实wait。另核converter独立wait、所有owned客户端、资源关闭明细、静止全库SQL和完整catalog/inventory。复用既有R09源码Gate，不重审旧18项或旧R04/R05 raw。

| 绑定 | 自行核定结果 |
| --- | --- |
| H snapshot/live | 241407B / C3DFDC66FDBFB025760ED78B0E0061CC49BD7AFB2DA0BF76685D94D0D89B8B99 |
| 其余三Go/docs | 逐份snapshot/live SHA/bytes与manifest一致 |
| source manifest | 18B99D5058C82EB282579535AE02F9D64B17E6EA981CA409D6BF1AE354364C02 |
| 实际非race binary | 71B3EF47D466A8EAB899C266311EAD224E5F5316D6B244A8E2D7EFDF705521D5 |
| execution policy | 7C94C0DAE104A177FE82E976FD12154F97880ED749CEFBFEFCE572EAA4A33E04 |
| 本次fresh grant | 8CFC63E9FD87D990047F683B80E071684981043C55243FF96F9322B2FE9DB41A |

实际入口为`^TestPgPerfR02_FaultSeedDisk_CancelsDuringSeed$`、count1、test.timeout5m，CWD是真实`test/integration/article`；没有拿归档转换进程的exit0代替被测进程。Go4876从11:11:14.390651Z到11:11:41.804513Z，真实wait、自然exit1、forced=false。独立test2json converter40560在Go结束后运行，exit0仅证明归集成功，其时间不是业务phase时间。源码launch前/收口后五hash、实际binary、policy全部相同；本次live只做哈希核验。

grant创建11:10:52.867281Z，早于本次启动；仍是唯一attempt`followup-20261104T102237932710Z`。gr-110730-42772保留预检FAIL：无Go launch、无新增schema，原83集合保留；最终全库2,029,926,079B。旧expected2,029,893,311B到该值的32,768B漂移原因仍unknown；新grant刷新的是准入snapshot，policy SHA、resource origin、old small ledger均未更换。

## 共享flight与SQL次数

原`e/diagnosis/disk.*.json`共9份。5次成功原始采集flight1..5、2次共享回执、1次真实typed cap读、1次terminal拒绝。因此**实际resource-size SQL读共6次**，并非9次；外部before/after两次`pg_database_size`另算。最后terminal回执bytes0/SQL0，没有被当成成功样本或新SQL。

| 源flight | 源真实START Z | 源完成 Z | 源SQL | 共享B请求 Z / 固定截止 Z | B自身SQL / 等待 |
| --- | --- | --- | --- | --- | --- |
| 4 | 11:11:23.3385082 | 11:11:28.7245408 | 5.386018s | 11:11:25.4131799 / 11:11:40.4126479 | 0 / 3.3113519s |
| 5 | 11:11:28.7349196 | 11:11:34.600193 | 5.8652575s | 11:11:30.4130264 / 11:11:43.3385082 | 0 / 4.1871552s |

共享回执保留同一FlightID、源Requested/ReadStarted/START/Finished/Bytes、SourceSQLNS；没有把B请求或完成时间当新采集START。两B均在原冻结截止前得到完整成功；其截止仍受此前成功START+20s约束。flight4完成后新的实际请求flight5有新START及非零SQL；之后实际cap读35.4132500Z启动、有6.1457471s SQL。此链证明本次活性优化真实经过并发等待/共享路径，而非仅离线mock或复用历史R08结果。仍只接收本次已到达的路径，不外推所有竞争时序已在PG穷尽。

## 准确增长、typed cap及观测差额

`queries/3-size-before`的独立只读全库SQL为2,029,926,079B，与harness最初成功采集及fingerprint database_size_base一致。migration committed，fixture为100 users/1000 articles；有限循环在11:11:34.6607378Z开始。`seed-guard-batches.json`完整列出4622条连续Sequence、每批10行，合计46220条成功Exec的RowsAffected；首批34.6607378→34.6647176，末批41.5547815→41.5570845。该计数是已确认批次回执，没有伪称另跑了外部全表COUNT，也未把可能取消中的未确认批次加进成功总数。

| 观测量 | 原始bytes |
| --- | ---: |
| 本次准确before | 2029926079 |
| 测试绝对触限阈值（before+2MiB） | 2032023231 |
| 最后完整成功flight5 | 2031539903 |
| typed cap真实扫描值 | 2103916223 |
| cap观测增长 | 73990144 |
| 外部静止after | 2104227519 |
| before→after真实总增量 | 74301440 |
| cap扫描值→after差额 | 311296 |

真实失败为`*articleintegration.pgPerfResourceCapError`，Observed=2,103,916,223B≥target。终止字符串保留`tier_delta=696983552 tier_cap=625090560 total_delta=696983552 total_cap=16106127360`。guard记`termination_kind=cap`、target_reached/root_canceled/guard_failed=true；此判断由errors.As的实际terminal提供，不是t.Failed、普通取消、deadline或freshness错误冒充触限。循环约6.899s结束，未耗完60s；stdout明确后seed检查失败、后点NOT RUN。额外失败文本均对应本次预期cap及停点；未见SQL timeout/事务泄漏/关闭失败污染此目标。

**2MiB是严格注入触限阈值，不是预分配quota或物理增长上界。** 实际扫描35.4132500→41.5590137Z与密集插入重叠，末个成功批次在扫描完成前约1.929ms完成。该非瞬时扫描值与11:11:43.338987→48.699399Z在Go退出后取得的静止全库值，属于不同观察区间；311,296B差额必须计入最终累计。已有物证能说明扫描与在途写入重叠，不能逐bytes分配至某批/TOAST/index，也不能把全部差额说成“触限返回以后仍新增”。本结论接收原观测式fail-stop语义，不提供瞬时cap保证或逐对象物理归因；所有原值均未clamp或抹平。

## 关闭、精确所有权与inventory

`e/resources/<schema>.resource-close.json`记录root canceled、workers joined及同一typed terminal。关闭START11:11:41.5743881Z，原共享截止11:11:46.5743881Z；所有return/Close明细均使用该截止。pre-close acquired=0、return_error为空；按冻结H waitReturned的实际谓词，是事务回收后ownedTx为空且业务pool acquired为0。business/disk/probe返回和Close均完成、acquired_after0，没有给各组件重新加5s。最终closure133,985,000ns（133.985ms）、goroutine2→2、residual backend0；无fail-stop文件或强制杀进程收口。

外部live查询精确捕获同一app`han_pgperf_r04_fu_gro_111112_25876`的3个不同backend身份，重复快照没有被计成13个独立backend：

| backend PID | backend_start（+08） |
| --- | --- |
| 42504 | 2026-11-04T19:11:14.617489+08:00 |
| 35316 | 2026-11-04T19:11:20.455626+08:00 |
| 5172 | 2026-11-04T19:11:22.732756+08:00 |

各query另有自己的app/PID/backend_start及实际wait。query9精确业务app为空，query13覆盖该业务app及本次observer前缀，排除自身后为空；其observer自身另有exit0/真实wait，未声称SQL能在连接仍在时证明自身已退出。13个psql客户端+Go+converter共15个owned客户端与final逐个对齐，全部实际wait，无forced或unresolved。query2/query10 active_other均0，after size在Go自然wait后取得，服务器/DB身份前后一致。

catalog-before83、catalog-after84，旧83集合全部包含；仅新增`han_pgperf_r04_sr2seed_bc59478014d93ae6`，与created event、migration、seed、fingerprint、资源、inventory和final实际集合一致。R08首次失败schema`sr2seed_8c8eec0a1ed97edd`仍在旧83中，失败履历不移植。本次schema retained=true，无清理授权/操作。

原small ledger实际80B、SHA`2B368282C7667FC2B3563F82FC293FBBEEE59A350A52CB41D08C57BDB87FC4D2`保持，Window仍`2026-11-04T03:29:04.586115+00:00`，global/tier origin1406932671B。final累计=2,104,227,519−1,406,932,671=697,294,848B；small剩=1GiB−累计=376,446,976B（约359.01MiB）。该余量是本次收口时读数，不能替后续阶段容量预检或承诺足够空间。pgdata available45,576,126,464B，物理卷与累计tier限额没有混为一项。

## 局部状态、限制与STOP

本次补齐R09共享flight、真实growth/cap取消及正常关闭的局部执行证据。无新增需作者返工的finding；原finding的growth子条件可引用本报告，**整项仍维持6 CLOSED/12 PENDING**，不替Lead修改任务/状态。未执行leak、N9及其他W条件仍PENDING；当前未授权后续PG。本次执行没有完成正式采样、性能SLO或E2E。

已实读`lead/followup-attempt-r01-full-not-run.md`：完整启动门11:15:37.932710Z已过，Main11:16:19Z确认无完整cal/small/medium命令/目录，登记本attempt完整calibration/small/medium永久NRU。本局部PASS不能追溯恢复资格；原11:29完整NRU及large/canonical NRU保持。Tnew仍10:22:37.932710Z，全部review/返工计时，12:47:37.932710Z停新/12:52:37.932710Z硬停不延长。

证据完整性：evidenceaudit.json列全部实际消费原件的SHA/bytes、138项核证结果和最终复算未漂移检查。offline-audit.py可离线重核，不启动作者代码或数据库。最终报告/结论/audit自身哈希及Reviewer actual STOP记录在本目录index.json；旧Review和作者物证保持原字节。
