# S2.4 下一执行门 r01

仅测试规划，固定Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca。本次只读+新增本目录，不运行Go/PG、不改源、不建Tnew/新policy、不作采用或核销。Role/add-test-design完整读取，已读适用rules沿用。原11:29批次完整验证永久NRU；large/canonical仍NRU，原12项仍按独立证据逐项接续。

## 依据与当前阻塞

已核现役S2.4卡、followup-r04（B1A645…E283）及budget-r02（4B5D2F…208A）和两份Lead采用；已读 `code-review-error-r06/report.md`、`lead/readonly-implementation-adoption-r06.md`、`lead/readonly-r06-reception-and-r07-grant.md`（9498BDEC…F4682）、`author/readonly-pg-r06/report.md`（A619267E…452D6）、runner及第四NRU记录。R06 H固定550B1661…9392E7、binary固定99F50049…958A5D；当前H仅定向核入口/政策消费者，不将并行在写R07当已冻结或已评审。

R06 natural3：准确SQL2456.4574/4960.4788/5840.7232ms，bytes1992988351，exit0；controlled6：实际11.9759854s，exit0，注入与自然SQL分账。第三实到15s取消，但probe Acquired1→exit86，正常summary缺失且旧runner先读summary再保存exit，结果链不完整；是关闭门FAIL，不是预期exit1通过。第四NRU。外部最终三精确application的backend0只证明其最终收口，不能倒推R06关闭门曾成功。旧只读窗口也已结束，不复活。

R07正离线修异步归还/销毁与boundedClose，尚无本文可接收的最终freeze/独评/新只读许可；**目前任何Go/PG执行都阻断**。

## G1：R07后只补第三、第四只读

最低前置：R07最终五hash+新binary/构建链、原Reviewer窄delta静态PASS、Lead接收该版本并另给独占只读安排。离线须区分“Release已调用但稍后归零”与“期限内始终未归还/Close未完成”：原5s或更早hard截止不变，后者仍fail-stop86，不能删检查/无限Close/新增15s尾部。若这类窄改影响reader/预算/新鲜度逻辑，先扩大影响说明及必要回归，不假定只改关闭。

推荐新只读许可最多10min、8min停新组；是待Lead具体采用的安排，不是既有窗口延长，不创建Tnew。每child仍owner90s、test timeout2m，外部截止更早者优先。只读入口 `TestPgPerfR05_ReadOnlySQL` 未调用newHarness，强制default_transaction_read_only=on；目标只限loopback:5433/PG180006/既有han_perf_s24。不得DDL/DML/新schema。精确argv与env见 `argv-env.json` 前两项，R07 binary路径/哈希须从受审manifest绑定，当前未给不存在的binary赋值。

| 顺序 | 最低接受证据 | 拒绝条件 |
|---|---|---|
| 第三total-fifteen-failure | 注入16s SQL确到（不要求后续size SQL已到）；总deadline为15s且实际取消，accepted=false、root失败、late不发布；**预期进程exit1**、真实probe归还及boundedClose完成、worker/goroutine回落。瞬态Acquired1快照可原样保留，关键是在原关闭deadline内实际归零；外部核精确app/PID/backend_start集合backend0 | 仍exit86/外层杀、前置失败、unknown、关闭超界或漏物证均不通过；停第四 |
| 第四earlier-freshness-failure | 当前入口先完整真实size成功，受控7s等待，再注入16s；deadline确等**先前成功Started+20s**且早于本次请求+15s，expiry/rootcancel、accepted=false、late不复活、exit1及相同资源收口 | 前置size失败、目标注入未达、只看到超时而无旧锚点、等到scan完成才续期均失败 |

现入口sample只带即时acquired快照，不能强制将所有即时值0作为唯一接受条件；需结合R07关闭时序。外层spawn时持久化argv/CWD/实际环境非敏感键/五hash/binary SHA/PID/start；wait后**先落真实exit**，再读取正常或fail-stop材料，缺正常summary不得掩盖86。禁止打印/存DSN值/环境全集。旧R06 runner原样保留，其固定R06 binary、过期截止与先读summary逻辑均不适于直接重启。

**两个R06正例可保留，不自动移植。** 它们永久是R06真实历史正例，不因R07窄修订失效。若原Reviewer核R06→R07仅关闭归还等待/Close边界，未改采集/身份/只读入口/15/5/20与成功结果判定，可将其作为未变逻辑的历史佐证，避免无故重跑；报告必须写“R06两正例+R07两负例补证”，不能写“R07四模式全测”。若依赖受影响或Reviewer要求新版本成功关闭证明，只补一条最小正向smoke，不能预先称原两组已经覆盖新close路径。R06七项目checks亦仅R06；R07按实际修改影响及项目门提供所需同版检查。

## G2：genuine新attempt的policy/资源门

G1两条只读证据须先经原Reviewer/Lead接收。计划PASS/既有采用不等于已有真实新attempt。Lead另记录允许写入范围、执行owner、最终freeze与policy实际采用；绑定后才有Tnew（本文件不创建）。政策必须满足当前H `pgPerfParsePolicy/pgPerfExplicitResource/pgPerfParseProbeBudget`：

- execution：revision、attempt_id、parent_old_batch（旧批次NRU）、adoption_source为原真实绝对 `lead/followup-clock-contract-adoption-r04.md`，其SHA为 `3F73880358AE83819CE469AD75453B5ED4FB97F46E18F917A65DDDF04385CDAE`；另附本次实际执行安排记录，不能只拿旧合同采用当新的PG许可。window_start=实际Tnew，与环境一致；stop/hard=8700/9000、timed/single/tail=121/60/1，禁止旧legacy或离线fixture policy冒充新执行。
- resource_window_start精确旧字符串 `2026-11-04T03:29:04.586115+00:00`；resource_base_bytes=1406932671；resource_baseline_dir为原绝对 `author/r02/baselines`。small binding=file small.json/state existing/window旧origin/tier small/bytes1406932671/file_bytes80/SHA `2B368282C7667FC2B3563F82FC293FBBEEE59A350A52CB41D08C57BDB87FC4D2`/allow_first=false；原global文件 `author/pg-base-r01.json` SHA `2E5207B7EE17C5657D31E8F5F836231A9356099E37FCAABB4CA9CDBED2D6026D`。既有medium/large必须绑定实际长度/SHA/Bytes/receipt；确未登记medium才unregistered+allow_first，首次原子登记及receipt仍属同origin；large不启动。
- probe：resource_probe_timeout_ms=15000、resource_monitor_tick_ms=5000、resource_freshness_limit_ms=20000；不得旧detection_envelope字段。resource_probe_adoption_source为真实绝对 `lead/probe-budget-contract-adoption-r02.md`，SHA `2A0B0525933B80ACADF2AE66D399B3B25A5C109F872837DFCD1EE0EA6FBD6638`。时间准入与resource身份分别核，不改旧window或复制空baseline目录。
- env：见同目录JSON。新leaf各自绝对新evidence目录/唯一run-id；所有leaf共享同一Tnew/policy/global与档位ledger。清除残留故障/child开关，只有预期负例明确启用INJECT_FAULT=1，正常9语义/正式采样禁用。凭据仅安全注入既有目标，不写真实值。
- 新鲜度锚点为最后完整成功采集Started，首次unknown不准入，15s受旧freshness和真实owner更早期限；root observer不重归phase。准确SQL累计观测/触限停止，所有超限原值FAIL不clamp；单实例独占、真实PGDATA余量/原预留、原82schema集合、所有新增/半成品保留，无DROP/清理/新余额。

## G3：growth→leak→九语义，当前确实缺什么

当前H的真实已有入口在JSON中给出；Go命令在repo根运行，Go test会在包目录执行，迁移/生产SQL helper所需 `../../../migrations` 路径因此正确。直接运行binary做这些含fixture测试则必须CWD=`D:/AID/VSCode/Codex/Han/test/integration/article`，不能照搬只读binary的repo根CWD。

| 阶段 | 入口/期望 | DB增长与接受门 |
|---|---|---|
| growth正向准确性 | **未发现独立现成入口**。只读模式无写入，`ActualBaseline`是文件/账本回归，不能替真实PG增长。需Backend在后续明确授权范围提供固定H入口及argv，或由Lead批准并受审的有界真实写批核证步骤 | 未补入口/精确参数/同版证据就阻断，不能发明命令或擅自DDL。核真实准确SQL前后差额、增长路径、原base/卷余量及取消；不可用Windows缓存枚举对照替代准确SQL |
| growth触限负例 | `TestPgPerfR02_FaultSeedDisk_CancelsDuringSeed`，预期exit1；确认seed/COPY/追加确到、guard_failed/root_canceled与后点NRU，真实收口 | 当前test使用相对开始DB的**更严格+2MiB注入门**；100 users/1000 articles后循环每批10行。2MiB是测试停止阈值，不是新增DB字节上界，准确增长正向仍不能只靠此负例 |
| leak | `TestPgPerfR02_FaultLeakedTransaction_BoundedCleanup`，预期exit1 | 创建1个独立minimal/skipSeed schema及owned tx；实际到注入和泄漏guard、关闭/归还成功。前置probe超时不算到达；新schema元数据/迁移仍有真实大小 |
| 9semantics | `TestPgPerfR02_RealSemantics_Directed`，预期exit0；九分支全部有真实结果、资源证据 | 每支独立small minimal schema，约9×(20users+300articles)，notices配置为7支各1、两drain750/7500；另有业务新增行/索引。初始1000/10000 drain原断言不减，逐分支确认；一支先错后面NRU |

九分支顺序仍fresh-t1、moderation-race、delete-race、cross-page、late-notice-results（两个worker变体）、three-lock-chain、production-read-plans、initial-drain-1000、initial-drain-10000。Growth实际证据先接收再leak、leak接收再九语义；原Reviewer复核L/N后，余项仅为原先列明W条件才能进入G4，不能要求F002等先有正式窗才能开正式窗。

**容量预计的实际边界：**现DB1992988351−global1406932671=586055680，small仅余487686144 bytes（465.09375MiB）；为保存快照，不是当前可用容量。只读两条应无预期测试DB增长；growth至少增加fixture/迁移及写批，leak增加1schema，九语义增加9schema及上述行，具体物理bytes未实测，**不能给可靠MB上界/保证可装下**。2MiB也不等于总增长。后续执行者以准确SQL逐stage前后记录实际增量，再刷新small剩余额度；不得把unknown当0。无每批物理预分配义务，但没有足够准确余量/卷预留就停止，不为凑全套重置base/caps。

## G4：全量与证据接收

完整入口为JSON末三条：精确BenchmarkPgPerfHarness三次30s校准→TestPgPerfSmall→TestPgPerfMedium，正常fault开关清空。A/B/C仍41逻辑/42单元，K按实物，全部新schema记同一资源账。large/canonical不调用。原full57～71min（保守72）+执行Review20min只为估算；准确size扫描负载/同步guard可使其增长，必须阶段性重估。所有等待/编译/seed/Review/失败均计同一Tnew，正常完整启动最迟Tnew+53min且至少剩92min；若新估算更长应更早拒绝，不利用最后5min采样或重开attempt。Tnew+145停新点/批/op，+150硬停。

新calibration/small本身也耗small剩余资源；**当前没有它们在最新H下新增DB成本的可靠实测**，不能从465MiB余额推导全量可行。局部成本+预计完整small成本未可解释/实际余额不足，或G1/G2/G3任一门未满足，则正式全量NRU。无SLO/容量/生产就绪或E2E结论。

| 接收对象 | 必须同时成立 |
|---|---|
| R07只读两负例 | 新freeze/binary/窄差分与独评、单独只读许可，持久化真实argv/PID/hash/wait、目标实际到达、资源关闭与外部精确backend0；失败停止后组 |
| 新attempt写入与局部 | policy/adoption/resource-origin/ledger完整，真实Tnew及绝对门由Lead记录；每leaf精确版本、实际增长/超限原值、正常/预期失败分账、全inventory/关闭证据；缺目标路径NRU |
| 全量采样 | 同版全套命令/原始JSON/bench、每repeat真实10/30、TIMED121/SINGLE60/整点tail1、删失/unknown不算成功、独立oracle/完整批/计划、实际库内inventory/所有尝试及全资源/预算；原Reviewer执行Review+Lead同revision Gate |

本目录仅建议，JSON含待绑定R07路径与未创建Tnew占位，**不是已就绪可直接执行的脚本**；真实值缺失必须拒绝。当前下一动作仍为Backend离线R07→原Reviewer窄复审→Lead明确后续只读安排；本角色没有启动任何运行、创建policy/Tnew或扩大写集。完成后STOP交Lead。

