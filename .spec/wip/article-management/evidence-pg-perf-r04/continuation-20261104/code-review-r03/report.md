# S2.4 原18项第三轮有界独立复审

结论：**PARTIAL / PENDING；证据交接 CHANGES REQUIRED**。原18项在正式 freeze 中均已找到对应实现修复；本轮正式核销 **F008、F010 共2项**，其余16项为 **PARTIAL（实现已修、核销证据不足）**。本轮未确认仍然存在的原 HIGH 源码缺陷，也未发现新增 HIGH；新增1项 MEDIUM 证据映射缺陷 R03-E001。不能据此签发最终 CodeReview / 执行 Review PASS。

完整 new calibration / small / medium 本轮 **NOT RUN**，large/canonical 继续 NOT RUN，执行验收门 PENDING，S2.4 保持 active。已错过12:32:04.586115条件门；静态修复或后续普通项目检查不能恢复本轮完整性能启动资格。

## 身份、固定修订与方法

本报告由原独立、非作者、非运行者 Reviewer 续接产生；未重新 Route、未委派、未替作者修复。沿用已全文读取的自身 Reviewer Role、add-code-review、适用 code-review-checklist / abstraction-principles / Go / coding / observability / no-hardcode 依据及原规划、R01/R02复审上下文。作者 finding-map 仅作为18项索引，不作为核销结论。工作区固定 `D:/AID/VSCode/Codex/Han`，main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（manifest及既有固定依据）。未运行Go、PG或作者脚本，未读凭据，未Git写或清理。

审查对象为 `author/r02/source-freeze-r02/manifest.json` 的不可变 snapshot，kind 为 `Backend author source freeze r02; independent Review pending; complete performance NOT RUN`，captured `2026-11-04T04:57:00.857871+00:00`。逐份自行复算 bytes/SHA，并核 manifest 内两份 patch 摘要；没有读取变化中的 live pg_perf 文件替代快照。H/W/R/N/D 下文均指该目录快照，行号按快照正文。

| 对象 | bytes | SHA256 |
|---|---:|---|
| H pg_perf_harness_test.go.snapshot | 131747 | B8D11EBFB3E4D15F01BC15263E1899BF168732CB06B232099B0F1FD9759E8E86 |
| W pg_perf_write_test.go.snapshot | 19395 | 38ADA0ABCDC5698EAB67BC47F6D5F728C35FCFB00563A54FABC00FA2E9AC0022 |
| R pg_perf_read_test.go.snapshot | 30182 | F25D0339EE3904590C3F00F361807B3F2978F0E9609C9E26904F0EE6D55AB485 |
| N pg_perf_notice_delete_test.go.snapshot | 49416 | 3E5316B97059BD1AB3E85B07A6F831EF799BD0665FA590C65A5B2F85AEEFD4C7 |
| D article.md.snapshot | 13531 | D47605C85226A618680A9BA455557236F4D202A7FB3BA9C4FB388D9049650C70 |

candidate-delta.patch SHA `22A03308A190D6599801B9FCC985750C6513785E3EFB479F1A260AED0E98A71E`；source-r01-delta.patch SHA `36790941E9DF3E5848A1EC3A806EFD57A4B549D7EAB6640A9744D50F77315EB2`。本目录另保存独立离线 candidate→freeze 五份 diff，docs 差分为空。实际四Go都变，未按先前“只有H/N变”的过时范围作结论。

用户/Lead最新交接：作者 SOURCE/LOCAL/PG 于12:57真实 STOP，13:06只读复核无 owned PID/窗口子进程、五hash一致。本Reviewer消费该正式回复，但未自行运行进程/PG探针；以下外部 probe 是独立读取和核摘要的已存证据。Lead已核 live=freeze5/5，最终接收前的真实树/所有权/范围/命令复核仍由Lead承担。`lead/project-checks-r03/` 普通/contract/race检查由Main负责，本报告不等待或预判其结果，也不把它当本Reviewer执行。

## 两轮实际检查范围

Round 1：复核pin、逐份生成并审查candidate→final四Go实际差分；H大量变化含格式调整，按真实语义检查 phase / point / resource close / inventory / context admission / fault入口；W检查正文混合、c4诊断及E02传ctx；R检查跨页helper传ctx及恢复边界；N检查合法双recipient、N03 due计数、N04 fencing与异常收口、X08传ctx。复用未变化docs和R02已审消费者结论，不重读无关全库或设计。

Round 2：沿真实调用入口反查原18项，不只检查新增测试函数。独立核六份直接生产消费者/迁移文件摘要仍与R02一致，覆盖 query_service / notice_dispatcher / repository / command_service / deletion_coordinator / 0003_article.sql。recipient恰两种各一、真实SQL分页边界、dispatcher attempt/lease条件未被测试夹具修改。逐条消费 command-index、execution-map、finding-map；复算35条 result 与 stdout/stderr摘要，重新计算每命令before/after五hash是否与正式freeze一致，不直接相信作者的 exact_final_source 标签。

详见 `pin-verification.json`、`evidence-audit.json`、`verify_offline.py`、`mapping-discrepancy.json`。离线核证先发现diagnostic集合不等；最终脚本将该真实差异保存为finding，而非删掉差异或修改作者数据以让断言通过。

## 新增 finding（按严重度）

### R03-E001 — MEDIUM / OPEN：diagnostics 实物归属遗漏一个W02 schema

位置：`author/r02/execution-map-r02.json:749` 对应 `pg-directed-safety-r02` / run `r02-20261104T044227-5348be77` 的 schemas 数组；上游 `author/r02/measurements/r02-20261104T044227-5348be77/environment/schema-inventory.json`。

触发：该命令实际先执行取消/收口诊断，再执行W02双正文诊断，共保留 r2join 与 r2body 两个schema。旧局部 inventory 仅列 r2join，execution-map照录后漏掉 `han_pgperf_r04_sr2body_e8bc2e0c9f144c91`。同run已有该r2body的 `.json` / `.pre-close.json` 资源文件，实际 failed=false、acquired=0、backend=0、goroutine2→2；最终外部probe及 execution-map 顶层30清单也均包含它。

精确对账：planned_units 50个展开条目与原scenario-map逐项键集合相等，均为正式性能 NOT RUN；顶层 actual_r02_retained_schemas 30个互异并与最终probe30完全相等；但 diagnostics[].schemas 的并集仅29，无额外项，恰漏上述1个。不是数据库实物消失，也不是本版源码已被证实仍漏checkpoint；它是当前交接材料中的“命令→schema”归属缺口。不能把顶层总数30正确当逐项映射已齐。

违反依据：既定完整实际inventory/失败实物/命令来源可追溯义务。影响：消费者按诊断记录归集W02资源或核验实际创建集合时会漏1个schema，无法逐命令闭合。

最小修复方向：返回真实作者，在新证据修订或追加勘误中把该schema明确归到上述run与命令，说明旧局部inventory漏项由资源文件及最终probe补证；保留原日志和旧失败事实。**无需仅为此再改四Go或重跑PG**。复审只核修订后的命令schema并集=最终30、映射来源及摘要，不能自行补改作者文件。本目录 mapping-discrepancy.json 保存了缺项及相关源证据SHA。

## 原18项逐项状态

PARTIAL在下表专指“本版静态实现落点已修，但原finding要求的执行核销证据尚不充分”，不表示仍确认有同一源码缺陷。CLOSED仅针对原局部缺陷，不代表性能通过。没有将较早GREEN改称正式五hash运行成功。

| 原项 / 严重度 | 本轮状态 | 本版实现证据、已读执行证据与剩余条件 |
|---|---|---|
| F001 HIGH 测量错误不失败 | PARTIAL | H:2779先落raw/summary，H:2979统一拒绝errors、truncated、缺repeat、零有效成功；W继承调用不能丢弃summary绕过。较早pg-fault-measurement-r01确有完整10s warmup后1成功+1正确性错误、exit1、后点无schema，但非最终五hash；最终core的混合错误/unknown/outcome运行也不是最终H。尚缺最终相关代码对应的测量错误fail-stop与不启动后点证据或可核的精确版本连续性。 |
| F002 HIGH 实际期限/截断 | PARTIAL | H:1663 memberPrincipal(ctx)；W:417/R:455/N:239,278,1243传operationCtx。H:2590每次准入/deadline/真实ns轴、H:2779剩余累计tail、H:2979 zero/tail/wall门已落。最终五hash expired-helper检查T1/E02/D02/R05/X08均DeadlineExceeded且AcquireCount不增；最终PG不合作worker命令exit1并留fail-stop实物。尚缺同版或严格delta可核的混合business+context、unknown、合作取消/末phase取消、零成功/累计tail的局部回归关联；正式120/121整点执行没有发生。 |
| F003 HIGH seed前无磁盘监控 | PARTIAL | H newHarness首DDL前创建专用diskPool并启动monitor（startDiskMonitor H:1567），覆盖migration/seed/ANALYZE/append；guard失败取消根ctx。candidate→final未恢复seed后启动。旧seed-disk负例12.531s/exit1、真实超限/取消/backend0已核，非最终五hash；最终相关监控/资源收口证据或严格差分关联待补。 |
| F004 HIGH 跨进程重置medium基线 | PARTIAL | H:3011 O_EXCL持久基线按window/tier校验，H:3050独立child读取原文件，断言tier增量200>150而total280<1000。没有采用重试即新baseline。local-final-core-r01 race×3含该child，但H为DCE7…，非B8D11…；最终函数连续性证据待补。真实medium NOT RUN。 |
| F005 HIGH 泄漏时无界Close | PARTIAL | H:1083 cancel/worker等待/tracked tx rollback；H:1132有界等acquired=0后才Pool.Close，失败保留FAIL并关闭owned连接。原同步Close死等分支已修；pg-directed-safety-r01残留FAIL与r02修后0均保留。最终calibration/append/context正常收口均backend0、goroutine回基线；但最终泄漏/异常race与lock取消的定向负例未齐，外部probe0不能倒推内部所有worker均join。 |
| F006 MEDIUM c1缺near-limit正文 | PARTIAL | W按每worker iteration交替正文，H样本按实际body class分桶；W:230真实存储rune/byte诊断覆盖c1/c16三组，两桶均非空。旧safety-r02该诊断exit0，最终W有所新增/传ctx变化，未把旧五hashGREEN借给最终。正式每repeat两桶采样未发生。 |
| F007 MEDIUM c4不可达 | PARTIAL | W正式c16的warmup concurrency=4，H每repeat原10s窗口确用该concurrency，无增schema/独立P95。W:282另100ms四worker SELECT诊断是执行器观察；其旧运行不能代正式三次10s c4观测。本版正式运行NOT RUN。 |
| F008 MEDIUM 首次失败schema/inventory遗漏 | **CLOSED** | H:461 CREATE后立刻不可变created事件+checkpoint；H:469写actual schema/K；H:3674每次calibration Cleanup checkpoint并在failed时Finalize，不依赖第3次。最终五hash pg-fault-calibration-r01首bc001迁移step3注入失败，exit1/9.094s，K=1/schema1/created=true/committed=false/failed_at3、backend0/goroutine2→2。足以核销原首次调用失败缺实物问题；R03-E001是较早诊断的独立交接补证缺项。 |
| F009 MEDIUM N03完整批断言/非法追加 | PARTIAL | N:682场景保留初始due完整accepted及无terminal门，N:807每notice事务内两recipient；非due former预accepted与pgperf_due分开，acceptedAfter只计due keys，发送数不虚增。最终append/context命令追加1000due全部accepted/sent=1000，future retry4、terminal4，未降断言。原1k/10k完整drain成功仅较早hash；最终N03完整场景与small/medium原规模窗口均NOT RUN，不能将1000追加代10k初始或正式SINGLE<=60。 |
| F010 MEDIUM workload覆盖 | **CLOSED** | N:807追加产物路径包含schema/run/batch，H:701使用O_EXCL拒绝同名并保留错误；本版代码直接核路径及不可覆盖语义。最终append命令确有独立batch1/2/3产物及成功回归；local-final-core双schema batch1/重复拒写仅为较早H辅助证据，不称本版重跑。此项核销是产物身份/不可覆盖局部缺陷，不代替所有N03性能采样。 |
| F011 MEDIUM E02非真实竞争 | PARTIAL | W:417真实Replace/Moderate经H raceRow（H:1814起）观察独立PID/递归锁依赖后释放，核唯一赢家/audit/category/version。candidate同版子项PASS；最终新增ctx准入已由expired-helper验证，但最终正常race及异常收口尚缺五hash运行或精确delta关联，不能以初次timeout/残留失败已不存在作无据断言。 |
| F012 MEDIUM D02无UPDATE竞争 | PARTIAL | N:278真实versioned T1/Replace竞争与赢家终态门，未弱化预期version。candidate子项PASS、最终零准入已核；最终正常竞争与资源收口对应证据同F011待核。 |
| F013 MEDIUM D01幂等尾部 | PARTIAL | N:239每op Create新active行再首次T1，核pending/version/operation及业务不可见，非有限id循环。candidate fresh-t1三个独立op曾PASS；最终ctx变化无新增准入，但最终新鲜路径对应验证/严格delta连续性仍待核，正式TIMED重复NOT RUN。 |
| F014 MEDIUM R05变更均在分页前 | PARTIAL | R:455先第一页，再insert/edit/suspend，独立SQL按实际boundary核后续页、原cursor继续；defer恢复有独立3sctx，实际耗时计入phase/join。candidate cross-page曾PASS；最终ctx拒准入已核，当前正常跨页/恢复异常的对应证据待核。 |
| F015 MEDIUM N04晚到/同worker不可达 | PARTIAL | N:910 FreshNotice在一个owned tx形成双recipient，N:940真实不同/相同worker两个分支，旧A晚到不能覆盖B accepted，attempt1→2/lease不同，异常亦release/cancel/join。原23514已由合法结构修复；最终N摘要与pg-final-directed-r01中N一致且两支真实成功，但该命令H=DCE7…，不与正式五hash相同，尚不能用作者“最后H仅改测试”取代可核中间快照/差分。 |
| F016 MEDIUM X08未观察C | PARTIAL | N:1243真实三PID A←B←C观察graph，release holder/B后join、异常感知ctx；helper改传operationCtx。pg-final-directed-r01确有链图/成功及资源证据，但H非最终；安全取消成功又更早hash。最终正常链/取消收口对应证据待核。 |
| F017 MEDIUM read EXPLAIN不可达 | PARTIAL | H:2084在读fixture计时外调用生产SQL常量EXPLAIN ANALYZE BUFFERS，保留query/args/schema/plan；仅权限不足unavailable，其余错误失败。query_service直接消费者hash与R02一致。较早pg-directed-semantics-r03有三实际plan；最终全读点plan覆盖及相关版本关联待核，静态可达不等于已经执行。 |
| F018 MEDIUM PGDATA失败fallback | PARTIAL | H newHarness首DDL前取目标卷/probe，失败直接Fatal，无evidence卷代替；DB loopback:5433/固定han_perf_s24验证仍在，不新建其他库。旧volume负例3.610s exit1/inventory0/backend0有效，非最终五hash；对应最终保护路径的负例或严格delta证据待核。 |

## 本版执行证据与失败链

35条已记录命令均核result及log摘要，只有以下7条before/after五hash与正式freeze完全相等；其余28条不自动作为最终版GREEN。原始路径/完整SHA/命令/test事件/JSON实物保存在本目录evidence-audit.json。

| 命令 | 实际exit / wall | 本轮能证明的范围 |
|---|---|---|
| final-owned-pg-probe-r01 | 0 / 3.297s | 保存probe observed04:56:08.033414Z，20个作者r02 run、30schema、remaining_backends=[]；DB约1.889GB，卷C约49.66GB/D约142.99GB余量。是当时观察，不是13:06独立实测或原累计15GiB证成。 |
| local-final-format-r01 | 0 | gofmt-d空；仅格式。 |
| local-final-list-r01 | 0 | 仅测试入口枚举；不是执行测试。 |
| local-final-vet-r01 | 0 | 静态vet；不是contract/race/性能通过。 |
| pg-fault-calibration-r01 | 1 / 9.094s | 首次校准调用故障拒绝与完整K1实物/资源，非完整校准。 |
| pg-fault-phase-stop-r01 | 1 / 13.625s | 不合作worker真实失败退出；不是成功join或性能通过。 |
| pg-final-append-context-r02 | 0 / 31.625s | append回归18.58s、expired-helper10.29s，两个schema acquired0/backend0、goroutine2→2；不是N03完整正式窗口。 |

phase fail-stop原始记录：owned test PID8640、exit_code声明86、workers_joined=false、unknown=true；实际go父命令wait exit1。phase_started/deadline原始ns存在，context done在5.8778ms观察，observed1.0052182s。5ms合成phase故意不响应取消，超过剩余尾部则checkpoint+failure并os.Exit(86)，没有运行Go cleanup/resources文件；之后外部probe无该run backend。不能把snapshot声明86冒充父命令实际wait86，也不能将本次故障观测误称严格实时调度保证或完整120s active采样。host调度与落证耗时仍须以真实ns判定，越界即失败。

保留失败链：

1. 最初directed-r01真实锁等待超时、acquired2/残余goroutine；r02在candidate N04遇真实23514，后续chain/plans/1k/10k NOT RUN。不是环境错误，不能删掉或把这些NOT RUN写成通过。
2. append早期编译失败与后续真实23514分别保留；合法recipient修复后GREEN不能抹去失败修订来源。
3. pg-directed-safety-r01真实取消收口失败，safety-r02较早源码修后成功；最终Close逻辑静态合理，但不能把不同hash的运行包装同版。
4. pg-final-append-context-r01实际terminal rows=1 want4；其后ExpiredCompositeHelper因前错NOT RUN。最终H修正为每轮InvalidMessage停后继续有界round直到4，保留4条断言；r02同五hash确实达到terminal4并执行expired-helper。不是降低断言或忽略首错。
5. pg-final-directed-r01 exit0/24.735s只有N04两个分支与X08；local-final-core-r01 race×3 exit0/15.454s覆盖baseline child/outcome/phase/body/immutable等，但两者H均DCE7…，不是B8D11…。缺可复核中间H字节时，仅凭作者解释不足以称最终同版回归。
6. 原r01 diskprobe超时STOP是原实际失败背景，和本轮静态缺陷、后续23514、诊断测试失败分别保留；本Reviewer未重现。

## 合同、计数和资源界限

按已有效采用execution-r02：原起点11:29:04.586115，小+中失败/修复/复审/验证均计入共同145min；145min停新操作、150min硬停，不开新窗。TIMED仍名义120s=3×(10+30)，实际总tail<=1s、wall<=121；SINGLE仍wall<=60。H真实phase记录Si/Di、取消/最后完成/join offsets及ns，完整性/真实错误/unknown/零成功均拒绝；没有sleep凑窗。小样本N<500仍为exploratory，不能升级精确P95结论。小/中/大1/5.5/8.5GiB及total15GiB、目标卷剩余空间门与跨进程原baseline累计stop仍保留；单次probe DB大小/空闲量不能代替累计增量证明。

原scenario-schema-map的A/B/C41逻辑场景、42schema（X01两单元）没有被diagnostics替换；K按实际调用另计，本版故障K1单列。execution-map 50条planned unit键集合与原map50条一致（42个A/B/C展开单元+8个large非canonical规划条目；完整large再含canonical为原9）。均NOT RUN，不把30个诊断schema当正式42或旧43完成数。真实诊断30顶层数已核，逐命令归属受R03-E001限制。继承W/R消费者仍真实进入runner，不能以只改H/N或仅诊断入口代替继承义务。

## 最小交回方向与关闭条件

本轮不再给同一根因机械新增一轮源码返工：原18实施落点均已在freeze中修复，**剩余主要是证据归属/版本连续性/执行门**。当前明确需要作者修的是R03-E001证据勘误；不要仅因16项PARTIAL再盲改源码或重开完整性能窗口。

Lead可采用本报告精确登记：五hash冻结、作者正式STOP、静态修复18/18、局部核销2/18、16项验证pending、新增E001证据归属open、完整性能NOT RUN、整卡active。已有授权足够继续本范围证据返工，不需要再问相同建议值审批；这不是由Reviewer代Lead接受残余风险。

后续核销只做以下有界工作，由Lead安排所有权/窗口，不由本Reviewer启动运行：

1. E001追加正确命令/schema归属并保留旧inventory事实；核集合30及摘要即可，不需要重新PG。
2. 对PARTIAL给出可核的实际中间源码/精确相关delta与原始结果，使既有GREEN能证明相关被调用主体连续；若不能提供，只补对应的局部负例/定向回归，保留当前五hash与before/after/raw/resource，不重新跑无关全量。主要分组为测量错误/phase分类与累计尾部、磁盘seed/目标卷、基线child/泄漏取消收口、W02两正文及c4、正常竞争/跨页、N04/X08/plan与初始1k/10k。已有本版证据不重复要求。
3. 严格区分普通/contract/race检查与性能验收；Main Manifest7结果由Lead实读日志后再登记，不能提前预判，也不能补成缺失的正式性能采样。
4. 完整校准/small/medium仍为本轮NOT RUN；将来如需完整验收，另由Lead明确后续执行安排，保留原窗失败/elapsed/磁盘实物及原baseline，不以本次Review伪造恢复资格。large/canonical本轮亦NOT RUN。
5. 最终接收前Lead再核live五hash、真实命令/范围/所有权，Reviewer只对新增实际delta或补交证据继续，不重开已核无关库。最终执行Review PASS与整卡关闭仍未满足。

唯一写集为本code-review-r03的report/verdict、离线diff/核证脚本/结果；R01/R02和规划报告未覆盖，旧code-review-r02 report/verdict摘要已再次核保持原字节。完成此轮有界独立复审后 **STOP**，不追读正在运行的Main检查，不启动任何Go/PG。
