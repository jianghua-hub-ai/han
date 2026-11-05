# S2.4 source-freeze-r01 非作者静态代码评审 R01

**Code Review Static Gate: CHANGES REQUIRED。5 HIGH / 13 MEDIUM / 0 LOW；无 CRITICAL。Execution Evidence Review Gate: PENDING（本轮未评）。整卡不能关闭。**

## 固定对象、权限与依据

本会话沿当前 S2.4 继续 `add-reviewer`，为非源码作者、非运行者的独立评审；未重新 Route、未另启 Agent、未下级委派。对象类型 code，附带批准 docs 段落。工作区固定 `D:/AID/VSCode/Codex/Han`，现场 `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。原规划 PASS 的 report/verdict 保持原字节，本报告不撤销规划结论，也不把它扩展为实现 PASS。

开始审查前五个 SHA256 全部独立复算匹配 `author/source-freeze-r01.md`；完成离线核证时再次匹配。以下 H/W/R/N 分别简写四个 `test/integration/article/pg_perf_*_test.go`，全部位置均指本冻结修订。

| 简写/对象 | bytes | SHA256 |
|---|---:|---|
| H：`pg_perf_harness_test.go` | 80,166 | `14C5DB1EAEA35BB2C48965EFC43B21A4032C09724275F2B6BC949096AA12182D` |
| W：`pg_perf_write_test.go` | 16,126 | `032722CCA8B0F0698CAB838CC97D6F04CEA2CB729FD9C50F3D4EB4429769FEAA` |
| R：`pg_perf_read_test.go` | 28,123 | `56DD9572452AF71F32C8B1733E6E9EAF048D26D9F8F8B1E082D6C69FF058A93F` |
| N：`pg_perf_notice_delete_test.go` | 44,456 | `1BFD916432DDA76BA7E3059BE27C5E9B6EB23A2EA1C84257282E260793048673` |
| `docs/benchmarks/article.md` | 13,531 | `D47605C85226A618680A9BA455557236F4D202A7FB3BA9C4FB388D9049650C70` |

全文读 source-freeze、G0、baseline JSON、四个 Go 文件；解码五个 source-baseline base64，核 baseline SHA 后作离线 diff。仅 H/N/docs 改变，W/R 为继承内容，但仍属本次明确要求核对的合同消费者。docs diff 仅批准的 `:189`，其周围 `:176-200` 已读。作者 G0 和源码实现声明只作输入，不作 Reviewer 事实结论。

上游为原 S2.4 卡（本轮现场 `tasks.md:1448-1479`，已采用80m与addendum）、原 r04 的相关场景全文§3、命令/环境/资源§4～5、采样§6、负例§7、数据预算/关闭§8～11；已在上下文完整读取的 addendum/map 和自身 Role、add-code-review 继续有效，固定哈希分别 `2C3F874B43653D4E31C4FD65AA21AAB91CA795386413235E829AD4057B96C774` / `383215FA893DC9AC0A8CD8777B4B71C22C1BA411DD79DEF864FBC03B4CE95AD7`。未读百万字设计，也不重开旧 r04 全量 Review。

规则：code-review-checklist、abstraction-principles、core coding/no-hardcode/observability 与 go-service 对应适用规则、安全加载门禁、项目 Manifest 的实际检查命令。测试授权的固定 endpoint/预算/fixture 不按生产硬编码缺陷处理；未采用的技术/profile 不作为要求。

唯一写入本 `code-review-r01/` 新目录：报告、verdict、离线脚本、verification JSON、三份基线 diff。首次查目录不存在，以 mkdir(exist_ok=False) 创建。未改源码、Tasks/Plan、旧报告；未运行 Go/PG、未读取凭据、未删除/移动或 Git 写。

## 两轮实际检查

Round 1：全文和基线 diff 核环境授权、错误/取消/计时循环、跨进程基线、schema登记、清理回收、D04/N03/N04改动、docs边界；从每一新增分支追到当前调用者，识别仅配置校验与实际运行保护的差别。

Round 2：逐个顶层入口追 small/medium 41逻辑/42展开单元，包含继承 W/R、N02-c4、W02-c4、X01 两 fixture、独立校准及被硬性禁止的 large。对照 r04 每个相关场景的真实动作与断言，核并发是否真正发生、有限批是否完整、读计划采集是否可达；定向读取直接生产者 `internal/article/notice_dispatcher.go` 的 reconcile/ownership/saveResult、`repository.go:611-702` 的 Lock/Claim/Save、`deletion_coordinator.go:291-343`、`command_service.go:626` 起内容更新事务、既有 noticeConfig/fake/logger helper。为核关闭与校准语义，只读本机 pgx v5.11.0 的 pool/tx 文档实现及 Go testing benchmark 的 b.Loop/runN 相关段落，未执行工具链。

运行编排的直接消费者 `author/run-command.py` 已读：它持久化一个 pg-window-r01.json 和一个全局 pg-base-r01.json，设置共同开始时间、fresh evidence root、新 run-id 和既有目标库；没有独立 medium 档累计基线。该脚本不在五文件冻结范围，其内容只作本次消费者观察，不能扩展本代码结论的版本。

局部作者证据 `local-freeze-tests-r02`、vet/list/format-r01 的 result/stdout/stderr 已读，日志哈希复算一致。race/count3确有三个局部测试各3次 pass、进程exit0；它只测重复费用公式、预取消无操作、warmup错误计数。vet/list/format 的前后五哈希匹配冻结修订、exit0；list恰3个顶层，format输出空。race result 未带源码前后hash，仅freeze声明把该结果关联本版，不能当独立精确修订证明。上述结果未覆盖以下实际场景缺陷。Reviewer没有复跑。

## Findings（按严重度排序）

每项均针对以上冻结五哈希及其直接消费者；F001～F018 稳定全称为 `S24-CODE-R01-F001`～`S24-CODE-R01-F018`。修复须交真实作者，不由 Reviewer 代改。

### [HIGH] F001 测量期错误可只入统计而测试仍通过

- 位置：H`:2105-2116`、`:2210-2215`、`:2265-2272`；W`:193`、`:226`、`:249`、`:273`、`:324`、`:379`。
- 证据/触发：phase把op返回的错误记入summary，但runner不使测试失败；W01/W02/W03/W04/E01/E02/E03均丢弃summary。warmup正常、测量中一笔业务或正确性断言返回error后，循环继续；其他写入成功仍能满足末尾“created>0”等断言，E01～E03甚至无末尾summary断言。
- 影响/依据：错误或一成功一冲突不成立、审计/ETag断言错误可留下go test PASS并继续后点，违反r04§4.4、§9.3和addendum`:78-82`“正确性失败停止”，不仅是允许观察的性能数字。
- 最小方向/复审：在统一边界区分预期拒绝与执行/正确性失败，先持久化失败样本，再使该点失败并停止后续点；回归覆盖warmup成功后测量中出错、部分成功后出错及E场景断言失败，证明真实命令非零且后点不建schema。

### [HIGH] F002 120/60秒只校验名义参数，在途操作无单点截止

- 位置：H`:2065-2066`、`:2082-2084`、`:2121-2124`、`:2165-2167`、`:2181-2215`；R`:106-114`。
- 证据/触发：phase timer只关闭下次准入，op使用300/780秒级h.ctx；随后无限等待本轮op返回。10/30秒的最后一次操作若在窗口末开始并运行数秒，实际phase超窗；TIMED六个phase各多2秒即132秒，而配置校验仍为120秒。D04整批和连续翻页等多语句op尤其可长尾。最终测量phase内ctx耗尽后没有完成时长/实际重复核验，Truncated仅在重复前或warmup后设置；已有成功且无error的提前退出也可能被AssertPoint接受。
- 影响/依据：不能证明每次活跃窗≥30s、真实三重复、sum实际warmup+window≤120s或SINGLE≤60s，违反addendum`:73-80`、原卡`:1472`。这不是要求因微小调度误差误判，而是根本没有对实际长尾/截断设门。
- 最小方向/复审：为真实单点期限、phase完成状态和取消尾部建立统一裁定，不能拿配置窗冒充实测；超限/不足按FAIL/NONCONFORMING保留。用受控长尾op与最终repeat取消回归，核单调时钟原始值及测试非零；不得缩短既定窗口或重复来通过。

### [HIGH] F003 磁盘周期保护在迁移和seed全部完成后才启动

- 位置：H`:1187-1205`、`:1209-1225`，seed写入`:1700-1915`。
- 证据/触发：监控goroutine直到pgPerfMigrate、COPY、提交、ANALYZE全完成才创建；这之前只有seed前/后的核查。medium可在600秒seed期间持续写入/放大，超过档位或卷余量后仍继续，直到seed结束/超时才观察到。
- 影响/依据：危险最大的迁移/seed阶段不受累计增量与真实卷余量的持续停止保护，违反addendum`:40`和原卡`:1471`。最终seed后报FAIL不能追回已分配磁盘。
- 最小方向/复审：在首个DDL/COPY之前建立可取消的资源观察，覆盖迁移、seed、ANALYZE、测量和新增批；受控在seed中跨过上限，证明及时取消、停止后点、保留半成品/增量证据，且不清理腾空间。

### [HIGH] F004 medium重试新进程重置档位累计基线

- 位置：H`:1104-1116`；直接编排 `author/run-command.py` 的pg-base-r01载入/写入分支。
- 证据/触发：每个进程tierBase首次设为当前DB大小，传入的共同base只对small回填tierBase；medium无持久化档位基线。第一次medium已留4GiB后重启同档，第二次再增4GiB时局部tierDelta仍4<5.5，全局total也可能低于15，实际medium已8GiB。
- 影响/依据：换run-id/进程可突破medium5.5GiB，违反addendum`:40`“跨进程持续累计”和原卡`:1471`明确包含重试。共同90min和total15并不能替代分档上限。
- 最小方向/复审：由共同执行窗口保留每档唯一开始基线，并验证不能以新run-id重置；离线/局部回归至少覆盖“medium已有保留增量→新进程重试→总量尚未达15但medium先超限”。原窗口时间不得重置。

### [HIGH] F005 检出连接泄漏后同步Pool.Close会阻塞后续关闭核证

- 位置：H`:1045-1052`，后续资源记录`:1053-1083`；N`:1137-1138` 的非取消感知release等待为相关路径。
- 证据/触发：AcquiredConns非零时只t.Errorf，随后同步h.pool.Close。项目实际pgx v5.11.0 `pgxpool/pool.go:468-474`明确Close阻塞直到所有连接归还；tx.go`:92-100`明确取消ctx不自动回滚事务。若有漏还连接/事务，资源日志、残留检查、goroutine有界轮询永远到不了，90min取消也无法解开Close。X08背景事务取得锁后若异常路径没有close(release)，也可留下此类持有者。
- 影响/依据：在本来最需证明失败收敛的泄漏分支中卡住，最终只能依赖外层进程超时，违反addendum`:36/90`与原r04§5.4的有界收敛和真实失败证据要求。
- 最小方向/复审：声明并回收全部tx/worker所有权，异常和取消路径必须释放channel/连接；关闭失败须在共同截止内落真实失败证据，不能把阻塞Close包装成成功。回归故意持有连接、worker等release时取消/断言失败，证明有界结束且无残留。不得只把Close移到无人等待的goroutine隐藏泄漏。

### [MEDIUM] F006 W02-c1不可能测到near-limit正文

- 位置：W`:109`、`:218-229`；H`:2069` worker编号来源。
- 证据/触发：c1仅worker0，正文和bucket都按worker%2选择，永远为typical/索引0；单个c1点没有near-limit样本，不可能分桶报告两类正文。离线核证结果记录可达worker集合[0]和正文索引[0]。
- 影响/依据：违反原r04`:179/465`要求每个c1/c16点均混合并分别报告；继承文件零diff不意味着该义务已满足。
- 最小方向/复审：正文类型按单worker实际操作序列分布，bucket随每笔实际正文记录，避免仅按worker推断；证明c1/c16每个正式重复均有两桶真实样本、正确rune/bytes与分位数，不加新schema/点。

### [MEDIUM] F007 W02-c4中间观测仅在计划存在，源码无执行分支

- 位置：W`:214-235`及`:108-115`；map `embedded_assertions[W02-c4]`。
- 证据/触发：W02只按c1或c16跑同一固定worker数的runner；全文无c4分段、观测或结果，N02-c4的claim正确性不能替代W02写入中间观测。
- 影响/依据：原r04`:179`、addendum`:57`、原卡`:1472`保留的点内观测不可达。
- 最小方向/复审：在原c16点预算和schema内落实c4的明确观测，记录其角色而不单独报P95、不新增性能点；核实际可追溯证据及总时长。

### [MEDIUM] F008 schema在迁移成功后才登记，迁移失败会漏掉已创建半成品

- 位置：H`:849-855`、`:873-881`、`:1190-1195`、`:582-589`。
- 证据/触发：CREATE SCHEMA独立autocommit，之后任一迁移失败/超时会Fatal；addSchema和created=true在整个pgPerfMigrate返回后才执行，故真实已存在的schema不进e.schemas，inventory actual_schemas也不包含它。
- 影响/依据：违反addendum`:65-67`和原卡`:1477`“失败/半成品全部清单、与实际库对账”，可能把已创建误作未创建。fingerprint中的预期名称不能替代retained inventory。
- 最小方向/复审：CREATE成功即登记创建事件/时间，迁移与seed结果独立追加；失败仍finalize已得证据。注入中途迁移故障/超时，核命令失败、库内schema存在且inventory/attempt一致，保持无清理。

### [MEDIUM] F009 N03丢失原完整1k/10k drain断言

- 位置：N`:745-761`；基线同函数`:798-830`，离线diff已保存。
- 证据/触发：只有remaining==0的分支才比较accepted==expectedDue；窗口结束后的唯一数量门变为sent==acceptedAfter-acceptedBefore，不要求cycles≥1或原batch_no=0完成。10k只处理50行，sent=50与accepted增量50相等，就能通过测量后的断言，随后转入retry/terminal模式。
- 影响/依据：原“完整批accepted delta/发送数==expectedDue、无failed_terminal”的基准正确性被部分一致性代替，直接违反addendum`:80`及原卡`:1472`“原完整基准批断言不减”。静态反例在verification.json，未声称已运行该场景。
- 最小方向/复审：独立保留初始1k/10k批的全量完成门及状态/发送核对；未能在预算内完成即真实FAIL/NOT RUN。窗口维持新增批与初始批闭环分开记，不能把部分批当完整基准通过。

### [MEDIUM] F010 两个N03点会覆盖同名新增批工作量证据

- 位置：N`:830-833`；H`:414-425`。
- 证据/触发：N03-1k与N03-10k共用medium进程的evidence root，cycles各从0开始；新增批都写`workloads/N03.batch1.json`等固定名。第二点的同号批通过os.WriteFile覆盖第一点文件；该记录甚至不含schema/point用于区分。
- 影响/依据：丢失第一点新增批规模与准备成本的唯一记录，违反addendum`:67/80/92`追加、不覆盖与实际工作量可追溯要求。
- 最小方向/复审：工作量文件键包括schema/point/run/invocation/batch并防覆盖；两个N03点均创建batch1时各自记录仍存在且能关联样本/seed/资源，不把校验和重算当覆盖许可。

### [MEDIUM] F011 E02用串行旧版本调用替代并发更新与治理竞争

- 位置：W`:333-370`。
- 证据/触发：奇数轮先Moderate返回后Replace，偶数轮先Replace返回后Moderate；无并行连接/barrier、无阻塞观测。它只能验证已提交后旧version被拒绝，无法暴露两个同时持旧version的审计/赢家竞态。
- 影响/依据：原r04`:190`要求barrier同步、pg_blocking_pids有界轮询及Audit action/category/version绑定实际赢家；本场景没有实际竞争，原卡覆盖不能靠行名E02成立。
- 最小方向/复审：在同schema/点预算内建立可观察的竞争和双方结果，核唯一赢家、category/version及无幽灵audit；保存阻塞/同步证据，不用sleep推断。

### [MEDIUM] F012 D02未真正执行与DELETE并发的UPDATE

- 位置：N`:273-305`、`:312-323`。
- 证据/触发：holder只SELECT FOR UPDATE，T1在其后等待；holder直接Rollback让T1完成，最后才串行ReplaceContent。无第二DELETE，也无UPDATE在T1竞争时执行。
- 影响/依据：原r04`:193`的双删或DELETE-vs-UPDATE竞争未实现；当前结果只证明手工锁阻塞后顺序删改，不能覆盖并发失败方和稳定pending。
- 最小方向/复审：真实发起所选竞争操作并固定时序、核双方结果/状态/version/operation；保留阻塞证据及无残留资源，仍在原点预算内。

### [MEDIUM] F013 D01耗尽有限集合后把幂等查询当新T1吞吐

- 位置：N`:231-259`；H`:1473-1478`。
- 证据/触发：index跨warmup与三重复持续增加，`ids[next%len(ids)]`循环已删除集合；一旦全部active行转pending，beginT1走existing!=nil直接提交返回，后续没有InsertDeletionOperation或SetArticleDeletionPending，但仍计成功T1样本。源码注释明确承认idempotent tail。
- 影响/依据：后续重复可全部测既有operation读取，违反原r04`:192`“对active文章执行DELETE T1”的测量对象。末尾pending正确不证明测量阶段仍有真实状态转换。
- 最小方向/复审：各正式窗口保持真实active→pending工作，或将幂等观测独立标识且不替代原D01；warmup/各重复分别记录实际新T1数量和新增工作量，受原资源/时间门约束。

### [MEDIUM] F014 R05所有变化发生在第一页之前

- 位置：R`:441-469`。
- 证据/触发：Create、Replace、Suspend与after oracle全部在pgPerfRunPoint/CompareWalk之前完成；正式分页期间数据不再变。多页walk只能证明变更后的静态集合，无法发现翻页间当前事实变化引发的漏/重/半页问题。
- 影响/依据：原r04`:186`的“跨页插入、编辑、作者状态变化、不承诺跨请求快照”用例没有发生，虽然同时间戳平局仍有检查。
- 最小方向/复审：在两次明确分页请求之间注入受控变更，以当前事实与keyset边界推导独立期望；不得要求跨请求快照，也不能只把静态walk改名为跨页变化。

### [MEDIUM] F015 N04只覆盖claim旧镜像，未覆盖晚到结果及同worker重入

- 位置：N`:857-925`；真实消费者 `internal/article/notice_dispatcher.go:555-558/830-850`、`repository.go:675-692`。
- 证据/触发：循环固定A→回拨→B接管→第三worker用旧镜像Claim被拒→reset；从不提交旧A的投递结果，不使同一worker ID在旧lease与新attempt间重入，也不检查accepted终态唯一。真实结果保存另受claimed_by/attempt_count/locked_until完整fence，不能由Claim旧preimage被拒证明。
- 影响/依据：原r04`:199`及addendum`:82`保留的“旧worker晚到、同worker ID重入、无双重accept/终态唯一”未可达；真实窗口重复已有claim步骤不补齐该路径。
- 最小方向/复审：在原N04里加入可控晚到结果、同ID不同attempt/lease的完整fence断言，核最终状态与不重复accept；保留SQL回拨、不sleep、不连SMTP。

### [MEDIUM] F016 X08未观察C排队及三连接锁链

- 位置：N`:1153-1154`、`:1170-1189`。
- 证据/触发：仅在启动测量C之前对holder做waitBlocking(want1)，只能看见B。C发起查询后直接等55P03返回，未保存C的PID或轮询C→B/A阻塞关系/pg_locks/wait_event。C因任何行锁超时都能满足当前断言，不能证明排队链。
- 影响/依据：不满足原r04`:207`三连接链有界观测，旧PASS报告LOW-3要求捕捉C排队形态；该注意事项没有被实现。与F005的资源关闭是不同根因。
- 最小方向/复审：5秒lock_timeout有效窗内取得三连接PID并有界证明链形，再释放和join；保存真实pg_blocking_pids/锁证据与释放后继续，不把预期超时码当链证明。

### [MEDIUM] F017 读场景的EXPLAIN/BUFFERS采集入口不可达

- 位置：R全文各R01～R07及X02/X03调用；H`:1589-1604`；W`:265`。
- 证据/触发：四文件定向调用检索表明h.writePlan(EXPLAIN ANALYZE BUFFERS)只有定义、没有调用；唯一plan消费者是W03/W04的EXPLAIN FORMAT JSON（不执行）。R场景仅业务调用与SQL oracle，正常完成也不会产生所需扫描行/索引/buffers计划；不是权限失败后记录unavailable。
- 影响/依据：原r04`:182-188`、§6.1及原卡`:1472`要求的真实PG扫描/索引/buffers关键观测没有实施，raw延迟/正确性不能替代。
- 最小方向/复审：在不污染计时样本的明确位置采集真实查询对应计划，保留档位/参数/schema关联及实际失败原因；无权限才如实unavailable。核plans文件内容与实际SQL消费者一致，不用通用SELECT 1或plan-only冒充。

### [MEDIUM] F018 无法核PGDATA卷余量时仍可先建schema和seed

- 位置：H`:1118`（pgdataPath尚空）、`:1149-1163`、`:1190-1205`。
- 证据/触发：初次checkDiskBudget时pgdataPath尚未设置，不核卷；GetDiskFreeSpaceExW无法得到PGDATA余量时hardStopFree改用evidence路径余量，而两者可能在C/D不同卷。如果evidence卷足够，就允许迁移/seed；直到seed后checkDiskBudget才因PGDATA probe unavailable报错。
- 影响/依据：将证据卷空间替代目标卷安全判断，违背addendum`:40/90`和原卡`:1471`在当前磁盘余量未知/不足时不得启动的授权前置。与F003正常可测但seed期间缺监控不同。
- 最小方向/复审：目标PGDATA卷真实余量无法取得时在任何DDL前fail closed，不能用另一卷fallback；回归两卷余量不同/目标卷probe失败，证明零schema创建。

## 已确认的实现落点及证据边界

- 五hash/基线所有权匹配；无本轮生产/迁移/依赖变更。数据库创建代码已移除：H`:302-336`校验loopback:5433、固定han_perf_s24、版本18.6，连接已有目标库；pool同样限制endpoint，空DSN Fatal，解析/连接错误不回显值。在所读代码未发现凭据字面量或把DSN序列化的路径；未读取真实配置值，故不宣称运行时全日志脱敏验证。
- 总bytes与1/5.5/8.5/15 GiB相符；window start由持久记录传入，90min deadline跨进程保留；原窗口不会因Review重置。large顶层明确Fatal NOT RUN，代码后续large分支本轮不可达，保持D8/canonical无虚构执行。
- 校准改用b.Loop，helper按bc序号独立schema；本机testing实现证实b.Loop在单次benchmark调用内自调，不能把b.N当schema数。当前成功路径K须由实际清单计；第三次Cleanup Finalize，早期失败时不能保证完整inventory，受F008及后续实际失败留证核查约束。未据源码把K固定为3。
- D04真实COPY新ID整批并对每批pending/version/operation关联、首行不可见作断言；N03真实Reconcile/COPY后续批并加入retry/terminal；N04真实SQL回拨/Claim循环。手工completion summary已移除，warmup与measurement有独立记录。以上是静态动作存在，不等于窗口/原断言完整通过，仍受F002/F009等限制。
- A/B/C计划映射仍为41逻辑/42单元，X01两个独立harness、D04-100、N02-c4及所有调度入口存在；N02顶层名N02与计划N02-c2需显式运行映射。只核行名/数量不能证明场景语义完整，F006～F017即具体差距。
- docs原批准行保留A-PERF状态、large NOT RUN与非目标环境边界，没有声称本版执行PASS；所引用author/handoff-r01.md在检查时尚不存在，属于待交付执行证据，不当成已有结果。

## 待决定、限制与复审交回

待决定：本轮无需要重新请求用户审批的预算/规划问题；已有建议值授权不免除以上实现缺陷。D04目前统计单位是包含COPY/核验的一整批，N03是含probe/准备的reconcile round；作者已在note标明。后续报告必须按该真实单位解读，不能标作单个T1延迟/单recipient QPS，若仍需逐操作指标须回到原合同补证，不由Reviewer替换验收对象。

本报告没有独立运行时复现；所有finding为固定源码和真实契约可达路径的静态证据。离线Python只核哈希、基线diff、已有日志完整性和算术反例，不是Go测试、不模拟PG结论。未审全库其他模块、全设计、全量故障注入、Windows以外平台；本机Windows专属syscall限制保持G0-06边界，不冒称Unix支持。

`pg-calibration-r01`按本轮交接为进程DSN未注入导致运行环境FAIL，不构成源码行为RED；本Reviewer未读凭据/安全注入值，也未重试或等待r02。没有以当前Go/PG执行中的片段改写代码Gate，作者在途测量仍由Lead与作者独占治理。Manifest七项项目检查未由Reviewer运行；本轮只核已有局部四组证据，其他项目证据待交回。

交回真实作者：逐项引用F001～F018修复并提供新source-freeze/五hash和针对性回归；不得静默改变本次冻结对象。源码改变会使旧执行与新修复不是同一revision，需保留旧失败链和旧窗口增量/耗时，不能把Review当新90min授权。原W/R继承文件如需修复，仍须遵守Lead既有四文件写集，Reviewer不代修。

复审顺序：先对新冻结修订逐项核销静态finding及受影响消费者，再核当前同版校准/small/medium完整命令参数/进程exit、stdout/stderr、bench三正式重复、逐phase原始单调时间/错误/样本、map逐单元结果与所有尝试、新增工作量/seed、迁移SHA、失败/未启动原因、实际retained inventory与同窗口库内清单、逐档/总体累计与PGDATA余量、rows/tx/pool/backend/goroutine真实关闭。核全部使用同一共同窗口及未重置baseline；large D8/canonical按原因NOT RUN保留。缺项/FAIL不拼接、不择优、不冒充通过。

即使以后静态finding全部关闭，也只能先给静态子门；**Execution Evidence Review和整卡关闭仍须真实small/medium当前同版完整证据、非作者独立复核及Lead同revision Gate**。本轮静态结果及时交Lead，不等待测量结束，不更改在途进程；完成后STOP，等待原Reviewer后续续派。
