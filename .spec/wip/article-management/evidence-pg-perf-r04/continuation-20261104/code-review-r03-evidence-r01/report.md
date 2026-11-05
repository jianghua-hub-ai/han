# S2.4 R03新增证据有界独立复核

**结论：PARTIAL / PENDING，未达到最终CodeReview或执行Review PASS。** 本次新增核销 F001、F003、F004、F018；原F008/F010保持CLOSED，不重开。原18项累计 **6 CLOSED / 12 PARTIAL或PENDING**。新增证据映射问题 **R03-E001 CLOSED**。没有从本次证据确认新增源码缺陷；F005的当前负例未到达注入路径，不能核销，也不能仅凭该事实宣布源码仍有HIGH。

完整校准/small/medium/large/canonical本轮均 **NOT RUN（NRU）**，执行门PENDING、S2.4 active。原12:32:04.586115条件门已过，当前局部故障/核心证据不恢复完整性能资格，不启动新window、不洗elapsed或磁盘基线。

## 固定对象、方法和实际读取

本轮沿原独立非作者Reviewer，使用add-code-review复核方法；只审 `author/r03-evidence/` 新增证据、R03-E001补正及原16项的证据增量。没有第四轮源码返工或全库源码重审，未代作者修复。自身Role、Skill与相关规则沿用并核有效正文。固定工作区 `D:/AID/VSCode/Codex/Han` main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；不执行Git操作，分支/HEAD/index及268文件范围结论来自Lead现场复核，不伪称本Reviewer重新检查了268份源码。

正式源码仍为 `author/r02/source-freeze-r02/manifest.json`，captured2026-11-04T04:57:00.857871Z。独立重算五snapshot的bytes/SHA，以及实际工作树五文件摘要，均一致：

| 文件 | bytes | SHA256 |
|---|---:|---|
| H pg_perf_harness_test.go | 131747 | B8D11EBFB3E4D15F01BC15263E1899BF168732CB06B232099B0F1FD9759E8E86 |
| W pg_perf_write_test.go | 19395 | 38ADA0ABCDC5698EAB67BC47F6D5F728C35FCFB00563A54FABC00FA2E9AC0022 |
| R pg_perf_read_test.go | 30182 | F25D0339EE3904590C3F00F361807B3F2978F0E9609C9E26904F0EE6D55AB485 |
| N pg_perf_notice_delete_test.go | 49416 | 3E5316B97059BD1AB3E85B07A6F831EF799BD0665FA590C65A5B2F85AEEFD4C7 |
| D docs/benchmarks/article.md | 13531 | D47605C85226A618680A9BA455557236F4D202A7FB3BA9C4FB388D9049650C70 |

Round 1：仅消费有result.json且带ended_at/exit的8个稳定leaf（最初6个，随后追加2个正向组），复算result及stdout/stderr摘要、源码before/after五hash和实际expected marker，不把exit1自动视为故障回归成功。核build→binary→args/CWD、57个core子项pass事件、隔离夹具、两份索引/两次STOP与保存probe。没有读取运行中未完成leaf输出作结论。

Round 2：按原finding消费者反查新增raw/repetition/summary/inventory/resources/seed guard及受阻后点，核E001逐run补正与旧raw/preclose/probe，核Main七check真实result/log及对应摘要。复核旧R03报告/verdict、旧map、旧35命令result/log与已登记实物摘要未变，保持失败链。未重复R03已核的原本版7命令、未重新执行任何测试。

本Reviewer只运行独立Python离线核证、读取及hash；**没有运行Go/PG、作者脚本、凭据读取、Git、清理或委派**。唯一新增写集为本目录report/verdict与核证脚本/结果。

## 已完成证据的真实结果

目录以下均相对 `author/r03-evidence/`。完整命令/原始摘要、输出事件、实物JSON见本目录 `evidence-audit.json`；作者index已逐项与原result核对，不以作者expected_outcome断言代替核证。

| 完成leaf | 实际wait exit / wall | 独立结论 |
|---|---|---|
| local-core-build | 0 / 6.719s | main工作树冻结源码，integration + race编译到本目录tools/core.test.exe，构建成功；不是运行通过。 |
| local-core-race3 | 0 / 10.781s | 同一路径binary经test2json运行，count3、timeout1m；57个有Test的pass、0fail/skip，另有package pass。涵盖baseline独立child、phase期限/分类/unknown/取消、Outcome错误/截断/缺repeat/超tail/wall/零成功、不可覆盖/部分drain拒绝/两正文分类。 |
| pg-fault-measurement | 1 / 26.688s | **实际测量错误分支命中，另伴随disk SQL超时**：完整warmup后raw1成功+1非context错误、NONCONFORMING correctness errors=1，后点NOT RUN且无新schema；日志另有acquire0/sql5000ms超时。不能仅凭exit1判预期故障，也不能隐去并发disk失败。不是Go成功或性能通过。 |
| pg-fault-seed | 1 / 18.360s | **预期故障命中**：seed真实写批5464，tier_delta581402624 > 注入的更严格tier_cap486129664，guard_failed/root_canceled=true。原正式1GiB/15GiB及原基线未改。 |
| pg-fault-volume | 1 / 5.328s | **预期故障命中**：PGDATA volume free space unavailable; no DDL authorized，inventory0，外部probe无对应schema。 |
| pg-fault-leak | 1 / 10.672s | **非预期失败；目标分支NOT RUN**：建schema前disk SQL acquire0/sql5000ms超时，实际marker不含owned transactions leaked。不得用exit1、backend0或资源文件名核销F005。 |
| pg-positive-safety-r01 | 0 / 27.188s | **后续独立正向组实际成功**：CanceledWorkers release/join 13.94s、W02Bodies 11.32s；当前两schema acquired0/backend0/goroutine2→2。W02 c1/c16各三组均验证实际存储rune/byte，两正文计数各为workers；另有四worker103787800ns诊断。不是正式性能采样。 |
| pg-positive-semantics-r01 | 1 / 13.235s | **非预期失败；业务路径NOT RUN**：fresh-t1在首DDL前disk SQL acquire0/sql5003ms超时，未建schema/未执行T1。其余8项因前错NOT RUN；Go fail事件不等于各业务分支都被运行并分别失败。 |

F001直接实物：`measurements/r03-20261104T052446-aa49da05/repetitions/`的warmup completed=true、observed_ns10000401500；measurement attempts2/successes1/errors1、observed_ns519500、completed=false；summary N1/exploratory=true/window_truncated=true/completed_repeats0。raw两行分别seq0成功、seq1失败（error=*errors.errorString），不是context取消/PG超时的错误类别。该同版helper H:3358-3366在measurement第二次操作返回injected E02 correctness assertion；H:2969-2981先保存再统一拒绝errors。measurement时间轴13:25:11.0679383起+519500ns已完成该错误，之后13:25:12.3909694日志又报acquire0/sql5000ms disk SQL超时，13:25:12.3958612统一报告execution/correctness errors=1。**同时发生的disk失败真实存在，故该exit1并非隔离环境下单一错误原因；核销F001依据是raw/时间轴/具体统一拒绝分支，不能仅据exit1或marker断言。** 下一点日志明确NOT RUN，inventory只有r2ferr；外部probe也没有后点schema。此处只证统一测量错误拒绝功能，不把短失败测量冒充30s正式采样。失败点原始tail出现负差值是提前退出的原始记录，不能拿它形成合规性能结论。

最后全文消费作者 `final-review-index.md` 的追加勘误：作者也明确measurement为混合失败、leak注入未达，13:40:39仅做只读hash/PID检查，无新PG观察。Reviewer不照抄核销建议；独立依据上述raw及实际统一拒绝分支，**保留F001局部CLOSED，整次measurement不记“干净负例PASS”**。更早的disk超时输出与NONCONFORMING均保留，不能以作者原expected_outcome=true掩去污染。最终索引/勘误不把13:35:20 probe改写为13:40的新运行，最终STOP仍为13:35:47。

measurement资源：acquired0、backend0、goroutines3→3；seed资源：acquired0、backend0、goroutines2→2；volume资源：backend0、goroutines2→2。leak前置失败资源亦backend0/goroutines2→2，但未创建schema/事务，不能证明泄漏收口功能。后追加safety两schema资源正常；semantics预检失败资源backend0/goroutines3→3，但无schema，不能证明正常业务路径。原6命令索引/首次STOP未覆盖后2条，另由positive-evidence-index/positive-stop追加，不修改旧交接历史。

### core的源码、binary与CWD链

build在 `D:/AID/VSCode/Codex/Han` 对 `./test/integration/article` 执行 `go test -c -tags=integration -race -o <r03-evidence>/tools/core.test.exe`，before/after五hash均为freeze；随后记录的run命令明确通过 `go tool test2json` 执行该**相同绝对路径**，参数包括 `-test.count=3` 与指定的core选择，不存在新源码工作树。

实际binary 32140211 bytes，Reviewer离线复算SHA：`5B0EACA4CF170198E69F705C4E09E0AA2C80A59BB41466B9ABE9D4779054ACF8`，与作者evidence-index一致；文件mtime05:24:34.918444Z在build期间，run于05:24:35.161077Z启动，binary中有race=true构建标记。新runner源码顺序为build后run，中间没有替换binary的动作。Reviewer在读作者最终index前已记录同一binary摘要，最终再次核一致。足以将本次局部core行为归到本次构建链；不声称曾由Reviewer运行binary。

run的CWD是 `<r03-evidence>/core-sandbox/a/b/c`，编译源码仍来自main。测试里的 `../../../.spec/.../author/r02/unit-fixtures` 实际解析到 **r03-evidence/core-sandbox/.spec/.../author/r02/unit-fixtures**，不会落到真实旧author/r02。sandbox没有Go源码文件；它只承载现有测试的相对夹具路径。对子进程fail-stop原物证实际读取到三份PID32044/9580/41168、exit86、workers_joined=false；raw日志也记录三次actual_wait_exit=86。测试名带FailsAndJoins不代表不合作Go worker已被强制join，而是owned child进程退出收口。三个文件超过Windows普通260字符路径限制，Reviewer用绝对长路径只读成功，未把枚举告警当证据丢失。

身份限制：build/run两个result没有分别记录当时binary SHA；作者index及Reviewer记录的是**运行后现存binary摘要**，结合相同路径、构建期间mtime、顺序命令与稳定源码建立关联，不冒充运行时前后binary摘要或签名溯源。该限制单列，不虚构中间源码，不要求为补一个历史字段重跑已有有效core组。

## E001勘误核销

**R03-E001 CLOSED**，对象 `execution-map-r03-supplement.json`，SHA `B5DC35059A0376D4038B71F531A652D52AAE29714931F53E147876172673169B`。

独立复算旧map SHA仍为 `F6390CB2132B9714B266F5FD222659D5FB1E0ABFA97FF3846884771D416A7FBA`。旧diagnostics并集29；补正仅把 `han_pgperf_r04_sr2body_e8bc2e0c9f144c91` 加到run `r02-20261104T044227-5348be77` / `pg-directed-safety-r02`。逐run核其余集合均保持原值，无重复run、无跨run挪用。补正后并集恰30，与旧顶层30及旧final probe完全相等，无缺项/额外。

补正引用的14份旧来源摘要逐份匹配；其中created_raw_event实际存在于旧stdout，对应W02Bodies测试；resource/preclose内容与旧实物逐字段相同，acquired0/backend0/goroutine2→2；旧inventory仅列r2join的历史事实保留。补正没有改旧map/inventory/log，没有把本次r03新增2schema混进旧30，也不把较早W02运行当当前五hash的PG正向结果。

## 18项逐项增量结论

下表承接旧R03，保留其静态修复18/18的判断。PARTIAL/PENDING均表示核销证据或正式执行条件未满足，不代表本轮重新发现源码同一缺陷仍存在。行号/详细源码触发依据沿旧R03固定snapshot，本次不重开源码审查。

| 原finding | 本次状态 | 新证据增量与仍缺内容 |
|---|---|---|
| F001 HIGH 测量错误不失败 | **CLOSED（本次新增）** | 当前五hashmeasurement故障命中，raw1成功+1错误、统一NONCONFORMING、后点NOT RUN/inventory1；core错误/unknown/Outcome辅助回归通过。核销原错误可假成功缺陷，不代表正式性能通过。 |
| F002 HIGH 实际期限/截断 | **PARTIAL / PENDING** | 当前core race3覆盖合作取消、末phase不完整、混合business+ctx、unknown、零成功、超tail/wall拒绝、三次owned child exit86。加旧R03已核同五hashexpired-helper零Acquire及PG fail-stop证据，局部缺证已大幅补齐；**正式3×(10+30) nominal120、累计tail<=1s/wall<=121实际完整点未运行**，不能将合成5ms/计数单元证据代实际窗口。 |
| F003 HIGH seed前无磁盘监控 | **CLOSED（本次新增）** | 当前五hashseed fault真实写批5464后触发增量guard并取消root，保留schema、资源收敛；补齐原R03欠的同版seed阶段负例。 |
| F004 HIGH 跨进程重置medium基线 | **CLOSED（本次新增）** | 当前构建链的MediumRetry独立child回归三次PASS，复用同一个持久基线并区分tier/total增量；原window/global/small baseline摘要另已独立复算未变。核销跨进程baseline重置缺陷；真实medium NOT RUN，binary身份记录限制如上单列。 |
| F005 HIGH 泄漏时无界Close | **PARTIAL / PENDING** | 本次leak在前置probe SQL5s超时，不到CREATE/泄漏。后追加当前CanceledWorkers实际成功，race_failed/release_wait_canceled=true、acquired0及资源归零，取消收口子条件已补；仍缺同版真正到达泄漏分支的负例。不把预检空资源状态当泄漏清理成功。未确认残留源码HIGH。 |
| F006 MEDIUM c1缺near-limit正文 | **PARTIAL / PENDING** | 当前W02Single unit三次PASS，后追加W02Bodies PG实际成功，c1/c16各三组两正文数量及实际存储rune/byte已验证。剩余是正式各repeat采样窗口内两桶证据，不能把每worker两次固定操作诊断冒充每次30s正式measurement。 |
| F007 MEDIUM c4不可达 | **PARTIAL / PENDING** | 后追加当前c4 PG诊断实际观察4worker、103787800ns；静态warmup c4消费者保持。正式三次10s c4 warmup仍未运行，不以100ms诊断补写。 |
| F008 MEDIUM 首次失败inventory | **CLOSED（沿用，不重开）** | 旧R03同版首校准K1故障证据保持，不重复运行。 |
| F009 MEDIUM N03完整批断言/非法追加 | **PARTIAL / PENDING** | 当前InitialDrain拒部分批unit三次PASS；原同版append1000+future retry4+terminal4证据保留。当前初始1k/10k正向组未启动、正式N03窗口NRU，不能用部分拒绝单元或旧hash1k/10k补成完整当前场景。 |
| F010 MEDIUM workload覆盖 | **CLOSED（沿用，不重开）** | 当前ImmutableWorkloads三次PASS仅增加佐证，不改变旧R03核销边界。 |
| F011 MEDIUM E02非真实竞争 | **PARTIAL / PENDING** | 当前RealSemantics正常E02未启动；旧candidate子项PASS与旧首轮timeout/残留失败仍分版保留。 |
| F012 MEDIUM D02无UPDATE竞争 | **PARTIAL / PENDING** | 当前正常D02未启动，不能借普通contract/race检查替代真实PG竞争。 |
| F013 MEDIUM D01幂等尾部 | **PARTIAL / PENDING** | 后追加freshT1子测试已启动，但在首DDL前probe SQL超时，T1业务未到达；静态每op新建首次T1已修，未增加当前正常路径或正式TIMED证据。 |
| F014 MEDIUM R05变更均在分页前 | **PARTIAL / PENDING** | 当前正常R05/异常恢复组未启动；旧同版expired-helper证据仍有效但只证明零准入。 |
| F015 MEDIUM N04晚到/同worker | **PARTIAL / PENDING** | 当前N04两分支未启动；合法双recipient静态已修，旧较早五hash运行不冒充当前完整链。 |
| F016 MEDIUM X08未观察C | **PARTIAL / PENDING** | 当前三PID正常链子项因freshT1前错NOT RUN；后追加通用lock worker取消release/join已成功，但不替代X08三PID图和正常释放终态，旧图版本限制保持。 |
| F017 MEDIUM read EXPLAIN不可达 | **PARTIAL / PENDING** | 当前plans组未启动；静态生产SQL可达修复保持，旧三plan不跨版本变成当前运行/正式全读点覆盖。 |
| F018 MEDIUM PGDATA失败fallback | **CLOSED（本次新增）** | 当前五hash目标卷不可测负例命中、DDL前失败、inventory0、probe无schema，补齐同版故障证据。 |

合计CLOSED：F001/F003/F004/F008/F010/F018；剩余12项不核销。E001独立证据问题已关闭，不并入原18分母。F002/F006/F007/F009中依赖正式窗口的剩余条件继续挂起，不能靠增加短诊断强行核销。

## STOP、原窗口、资源和Main七check

实际消费 `handoff-stop-r03.md` 与 `evidence-index-r03.json` 的第一组STOP **13:29:58.9901728**，以及其后新增的 `positive-stop.md`、`positive-evidence-index.json` 与 `positive-process-stop.json`：作者SOURCE持续NO WRITE，后续正向组于13:34:06～13:34:47执行，**最终LOCAL/PG STOP为2026-11-04 13:35:47.482879+08**。旧STOP只描述当时释放，不再被当作最终状态。第一组leak失败停止原runner后点；后续独立positive runner没有重试leak，先成功执行safety，再因freshT1预检超时终止semantics。作者没有修改5s门或Go/docs。本Reviewer没有发起或代派任何运行，不把尚余原窗时间作为自动补测理由。

首个保存probe `final-owned-pg-probe.json` 的identity阶段wait exit2，PGSERVICE空服务定义错误，remaining_backends为UNKNOWN；该失败保留，不能声称此时已backend0。随后只修新增helper环境处理完成分阶段只读probe，`final-owned-pg-probe-r03.json` 四阶段均wait0，授权身份127.0.0.1:5433 / PG180006 / han_perf_s24已核；各stage文件与汇总内容逐项一致。阶段记录保存stdout/stderr摘要与解析值，没有另存原始stdout/stderr文件，本Reviewer不伪称重新获得或复算了不存在的原始字节。probe的源码before/after五hash与freeze一致。

较早保存probe05:29:09.746398Z～05:29:13.109362Z覆盖前四个r03 run，backend0/schema2/累计581754880 bytes，不能当后续正向组结束后的最终值。**最新 `final-positive-pg-probe.json` 05:35:15.987882Z～05:35:20.186524Z** 覆盖全部六个本作者r03 run的application_name精确集合，remaining_backends=[]；四stage均wait0，stage文件与汇总逐项对应、五hash一致。保留schema恰4个：r2ferr_86c4321399bef5bf、r2seed_a16769db036ed830、r2join_81c4ad7b4475c016、r2body_81c4ad7b4475c016。leak/volume及freshT1预检失败均未创建schema，不能把resources文件名算成实物。DB1992988351 bytes，减原baseline1406932671 = **586055680 bytes**，小于1GiB；原total15GiB与剩余磁盘门保持。最新保存probe卷C余48533991424、D余142958940160 bytes。probe范围不是全库所有会话，也不是Reviewer独立运行的当前数据库观察。

原window仍11:29:04.586115起，145min停新操作13:54:04.586115、150min硬停13:59:04.586115；两套runner引用原policy/baseline并对既有文件校验。独立核 `inherited-policy-baselines.json` 列出的window/global/policy/small四来源摘要未变，实际阶段累计没有换起点。最终停止声明早于全部硬边界；源码五hash在本次最终离线检查仍一致。保存 `positive-process-stop.json` 的observed_at为13:35:47.482879+08:00，记录owned PID及positive runner hits均为空；Reviewer未自行查进程，因此不伪称独立PID核实。

Main七check `lead/project-checks-r03/` 已逐条读result和stdout/stderr，format/modules/vet/build/contract/race/tests全部exit0，每项source before/after五hash匹配、SourceDrift为空。命令分别是gofmt-l、go mod verify、go vet ./...、go build ./...、go test -tags=contract ./...、go test -race ./...、go test ./...。它们是真实项目检查，**后3项不带integration标签的默认检查不能替代本轮pg_perf定向PG或正式性能入口**。未把Main执行冒充Reviewer执行。全部原始文件摘要已落本目录audit。

## 阻塞与最小下一方向

本轮没有新增源码修改建议。实际阻塞是 **leak前置disk SQL超时使目标负例未到达；后续positive-safety虽成功，positive-semantics仍在freshT1首DDL前因同类SQL超时停止，其余8项NOT RUN**。记录分别是acquire0/sql5000ms与acquire0/sql5003ms，不能推断数据库锁、IO或CPU的具体根因，不能写成预期泄漏负例GREEN，也不能声称各semantics业务都被运行并失败。

保持作者最终STOP与12项pending，不要求为了凑PASS放大5s探针门、绕过磁盘guard、再改四Go、重跑已有效leaf或旧本版7命令。Lead最小可采用方向是登记本次已完成的4项新核销与E001关闭、safety/W02局部成功，保留两条probe超时失败链/实物/elapsed；如后续需要补证，先针对执行环境与probe超时取得可核事实，再在明确运行安排内只补实际未达的局部路径。完整性能资格本轮不恢复，不能为此新开窗口洗原履历。

原18的静态修复18/18与6项局部核销足以精确更新评审证据状态；不足以宣布最终CodeReview/执行Review/性能PASS或关闭S2.4。最终接收仍由Lead在真实树核五hash、命令/范围/所有权，必要补交只核新增证据，不重复无关源码全量。旧R03报告保持原字节，本报告仅追加证据结论。

本次有界证据复核完成，**STOP**。不继续轮询、不启动工具窗口或任何运行。
