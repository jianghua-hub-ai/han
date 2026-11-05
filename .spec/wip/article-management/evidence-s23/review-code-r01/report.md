# S2.3 独立代码评审报告（review-code-r01）

## 1. 结论与固定对象

- objectType：`code`
- 评审角色：非作者独立 Reviewer（`add-reviewer`）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审时点：2026-11-02（Asia/Shanghai）
- verdict：**CHANGES REQUIRED**
- finding：**HIGH 1 / MEDIUM 1 / LOW 1**
- 本报告只记录独立评审结论；不勾选任务卡，不声称 S2.3 已关闭，不代替 Lead 或人的接收。

固定对象及本次核对值：

| 对象 | 字节数 | 行数（LF） | SHA256 |
|---|---:|---:|---|
| `test/performance/article/benchmark_test.go` | 18,205 | 696 | `9E429892185CBAB40BAD51F632E52013BECD42F5EC3AC8C94594A33F04651689` |
| `docs/benchmarks/article.md` | 10,712 | 177 | `0FC529EA77348EF36C2606A123FD781ECCDAE76E107F378BBACCC583CA5E2CFD` |
| `.spec/wip/article-management/evidence-s23/report.md` | 10,033 | 196 | `B5FD4FCDF1831F76671916EAEFEFABC0391A5E43422C85FB08267A1FE4CBE016` |
| `.add/project/knowledge/tasks/tasks.md`（S2.3 段依据） | 896,458 | 3,287 | `4EB9B3665D96253197B9FA36D91759DBD8F646271DBD2ECC0D62356CBCEDE0E0` |

三份交付对象的字节数和 SHA256 与输入一致；完成 target benchmark 后再次核对，四个 hash 均未变化。工作树包含他人既有未提交改动和大量未跟踪文件，本评审没有把“干净工作树”作为条件，也没有回滚、移动或删除任何他人内容。

## 2. 依据、范围与第三文件判断

验收依据：

- 任务卡关闭条件：`.add/project/knowledge/tasks/tasks.md:1433` 要求产出内容、分页隐藏密度、dispatcher、finalizer 与 replay 基线；不编造阈值；AM-Q2、容量和 retention 未确认时保留阻塞注记；独立 Review 通过。
- Article test-plan：`.add/project/knowledge/design-workspace/article-management/test-plan.md:876`–`:884` 的 A-PERF-01～05，以及 `:1245` 的性能基准规划。
- 已启用 profile：`go-service`。评审依据为 `add-code-review`、`code-review-checklist`、`abstraction-principles`、`coding-standards`、`no-hardcode-policy`、`task-execution-workflow` §7/§10 及 Go 安全底线。

第三文件判断：`.spec/wip/article-management/evidence-s23/report.md` 是 **WIP 证据载体，不是第三个产品交付物**。它位于 ADD 约定的 `.spec/wip/<scope>/` 临时交接范围，记录固定上下文、命令、原始 benchmark 输出、状态和无阈值边界；任务卡产品文件清单 `:1424`–`:1426` 只列 benchmark 与基线文档。因此不将第三文件判为越界交付；但它仍是未跟踪 WIP 文件，保留、归档或清理由 Lead/Steward 按证据保留规则另行决定。评审期间未修改、删除或移动它。

两轮实际检查范围：

- **Round 1（对象完整性、行为与断言）**：逐字核对 benchmark 696 行、doc 177 行、evidence 196 行及任务卡 S2.3 段 `:1420`–`:1435`；检查全部 benchmark 循环、错误处理、结果/计数断言、预生成数据是否在计时外、b.N 与每轮操作数是否一致；对照 doc 与 evidence 的原始输出、算术换算、A-PERF 状态及是否出现未经批准的阈值。
- **Round 2（生产者、消费者与上游依据）**：沿真实路径核对 `internal/article/command_service.go` 的 `Create`/`ReplaceContent`、`internal/article/query_service.go` 的 `ListArticles`、`internal/article/policy.go` 的可见性/分页策略、`internal/article/deletion_worker.go` 的 `Reconcile`、`internal/article/deletion_finalizer.go` 的 `ConvergeDeletionOperation`/`ReplayAllTombstones`；核对 `internal/article/notice_dispatcher.go` 的 `TransferNoticeDispatcher.Reconcile` 及其依赖接口；核对 test-plan、默认 benchmark 的数据库/环境变量依赖和 evidence 的边界声明。
- 结论不扩展到真实 PostgreSQL、HTTP、SMTP、Unix Journal、E2E、RTO、全仓重构或未来性能任务。

## 3. Findings（按严重度排序）

### [HIGH] 任务卡要求的 dispatcher 基线缺失

- finding ID：`S23-R01-F001`
- 对象/修订：`code`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 下的 `test/performance/article/benchmark_test.go`
- 位置：`test/performance/article/benchmark_test.go` 全文；`.add/project/knowledge/tasks/tasks.md:1433`；`docs/benchmarks/article.md:21`–`:25`、`:169`
- 现状：benchmark 没有任何 `Dispatcher` 或 `Notice` 引用，也没有构造 `TransferNoticeDispatcher` 或调用 `Reconcile` 的 benchmark。整份文件只覆盖创建、修改、列表、worker、completed finalizer 和 replay。
- 触发：按任务卡逐项核对五类基线时，dispatcher 一项没有可执行测量；文档也明确写出 dispatcher 未执行。
- 影响：任务卡的关闭条件“内容、分页隐藏密度、dispatcher、finalizer 与 replay 基线”未满足。即使 A-PERF-03 的 SMTP、lease、retry 和 max_attempts 仍因 AM-Q2/容量/retention 未确认而 blocked，dispatcher 的进程内调度基线仍可作为不依赖真实 SMTP/PG 的有限观测；当前连该有限基线也没有。不能把 A-PERF-03 的 blocked 注解解释为可以省略任务卡明列的 baseline。
- 建议：最小修复是在同一 benchmark 包中使用 `NoticeDispatcherDependencies` 的确定性 fake（事务、Repository、Gate、Mailer、Audit、ID、Logger、AcquireWork）构造真实 `TransferNoticeDispatcher`，对 `Reconcile` 至少测一个无工作或单次受限 claim 轮次，并报告真实调用次数/结果；SMTP 延迟、lease、重试积压、并发和 max_attempts 继续保持 blocked，不编造阈值。若 Lead/Planner 认为 dispatcher 基线应移出 S2.3，必须明确修订任务卡关闭条件并记录范围变化，不能由 Reviewer 自行改写。
- 验证：修复后重新运行 `go test -run ^$ -bench . -benchmem ./test/performance/article`，确认 dispatcher benchmark 实际调用生产 `Reconcile`，计数与结果断言防止空转；更新 doc/evidence 原始输出并保持 A-PERF-03 的未覆盖边界。
- 证据：`rg -n Dispatcher|Notice test/performance/article/benchmark_test.go` 无命中（exit 1）；`tasks.md:1433` 明列 dispatcher；`docs/benchmarks/article.md:25` 与 `:169` 明示 dispatcher 未执行/blocked；`internal/article/notice_dispatcher.go:113`、`:141` 证明生产 dispatcher 存在且依赖可注入。

### [MEDIUM] S2.3 任务卡“当前状态”写入了无关的 identity/Manifest 内容

- finding ID：`S23-R01-F002`
- 对象/修订：任务卡依据；`.add/project/knowledge/tasks/tasks.md:1429`
- 位置：`.add/project/knowledge/tasks/tasks.md:1429`
- 现状：S2.3 的“当前状态”写的是专用 identity integration 命令和 Manifest 七项 PASS，与本卡 S2.3 的 article 容量/性能证据不匹配；这些内容不能作为 S2.3 的 PASS 证据。
- 触发：Lead 或后续读者按任务卡状态判断关闭条件时，会把 identity integration/Manifest 的通过错误地归属到 S2.3。
- 影响：卡面状态与本次固定对象、命令和证据不一致，存在错误关闭、错误引用证据或错误转交的风险。共享工作树中 `tasks.md` 已有他人未提交改动，本次无法从对象本身证明该行由谁写入；因此这是卡面一致性缺陷，不据此认定作者越界修改。
- 建议：由任务卡责任人或 Lead 在 S2.3 实际评审/修复完成后，把该行改为与固定证据一致的 S2.3 状态；不得把 identity integration 或 Manifest 结果写成 S2.3 PASS。Reviewer 不修改该文件。
- 验证：复审时读取 `tasks.md:1429`，确认其只引用 S2.3 的真实命令/证据或如实标明未完成状态，并复核原行已替换；不以该行替代独立 Review。
- 证据：`:1429` 原文与 S2.3 目标、benchmark 命令和 evidence 5.1–5.6 的 article 结果不一致；本次固定 hash 为 `4EB9B3665D96253197B9FA36D91759DBD8F646271DBD2ECC0D62356CBCEDE0E0`。

### [LOW] 固定时钟说明写成“此次运行日期”，与代码实际值不符

- finding ID：`S23-R01-F003`
- 对象/修订：`code` + 配套 baseline 文档
- 位置：`test/performance/article/benchmark_test.go:37`；`docs/benchmarks/article.md:77`
- 现状：benchmark 将时间固定为 `2026-01-01T00:00:00Z`，文档却写成“时钟固定在此次运行日期”。本次运行日期是 2026-11-02，两者不是同一日期。
- 触发：读者按文档复现或解释结果时，会误以为基准依赖运行当天的时间，或误判固定数据的时间语义。
- 影响：不影响性能数值和业务正确性，但影响基线文档的复现说明与证据可审计性。
- 建议：把文档改成明确写出 `2026-01-01 UTC`，或让代码使用文档所述的可审计固定日期；优先修正文档，避免为了文档修改已固定的基准数据。
- 验证：重读 `benchmark_test.go:37` 与文档对应句，确认日期和 UTC 时区一致；重新生成文档 hash 后复审。
- 证据：`benchmark_test.go:37` 为 `time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)`；文档 `:77` 写“此次运行日期”。

## 4. 非阻断建议与待决定项

- **待决定（dispatcher 范围）**：由 Lead/Planner 决定是让作者补一个进程内 dispatcher baseline，还是正式修订 S2.3 关闭条件。Reviewer 不建议用真实 SMTP/PG 替代进程内基线，也不把 A-PERF-03 的 blocked 当作已完成的 dispatcher 证据。
- **证据载体保留**：evidence 报告可接受为 WIP 证据，但它是未跟踪文件；是否长期保留、归档或清理由 Lead/Steward 按证据保留规则决定，Reviewer 不处置唯一证据。
- **测试稳健性建议**：`BenchmarkReplaceContentMaxContent` 当前只校验最终标题/正文，没有显式断言最终版本为初始版本加 `b.N`；`BenchmarkDeletionFinalizerCompletedNoop` 只用错误检查，没有记录 repository 加载次数。当前生产路径经源码核对确实执行了替换和 finalizer 加载，因此不作为本次缺陷；可在后续提高防空转能力时增加调用/版本断言。
- **文档一致性建议**：doc 的数值与 author evidence 原始输出逐项一致；本次独立运行的时间和作者运行不同，属于基准波动，不据此判文档数字错误。复核时应保留本次独立运行的原始输出，不把不同运行的数字混写成同一次样本。

## 5. 实际运行命令与结果

以下均为本评审在本工作区实际执行，未使用作者摘要代替：

```text
gofmt -l test/performance/article
exit=0，标准输出为空

go vet ./test/performance/article/...
exit=0，标准输出为空

go build ./...
exit=0，标准输出为空

go test -run ^$ -bench . -benchmem ./test/performance/article
exit=0
PASS
ok  	han/test/performance/article	24.040s
```

本次独立运行的关键输出：

```text
BenchmarkCreateArticleMaxContent/ascii-20       	  152840	      8532 ns/op	2367.60 MB/s	16 B/op	1 allocs/op
BenchmarkCreateArticleMaxContent/unicode-3-byte-20 	   28000	     44432 ns/op	1363.89 MB/s	16 B/op	1 allocs/op
BenchmarkReplaceContentMaxContent/ascii-20        	  152192	      7822 ns/op	2582.31 MB/s	 0 B/op	0 allocs/op
BenchmarkReplaceContentMaxContent/unicode-3-byte-20	   26317	     53597 ns/op	1130.67 MB/s	 0 B/op	0 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-000-20	   41704	     28685 ns/op	1.000 batches/op	100.0 candidates/op	20.00 visible-items/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-050-20	   41019	     39720 ns/op	1.000 batches/op	100.0 candidates/op	20.00 visible-items/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-090-20	   10000	    133281 ns/op	3.000 batches/op	300.0 candidates/op	20.00 visible-items/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-100-20	    2678	    465321 ns/op	11.00 batches/op	1000 candidates/op	 0 visible-items/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-000-20	   23472	     45618 ns/op	1.000 batches/op	100.0 candidates/op	50.00 visible-items/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-050-20	   15528	     70593 ns/op	2.000 batches/op	200.0 candidates/op	50.00 visible-items/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-090-20	    6032	    228846 ns/op	6.000 batches/op	600.0 candidates/op	50.00 visible-items/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-100-20	    2240	    449377 ns/op	11.00 batches/op	1000 candidates/op	 0 visible-items/op
BenchmarkDeletionWorkerReconcile100-20                     	   23770	     43664 ns/op	100.0 operations/op
BenchmarkDeletionFinalizerCompletedNoop-20                 	17696269	        62.02 ns/op
BenchmarkReplayAllCompletedTombstones1000-20               	   24838	     58124 ns/op	1000 tombstones/op
```

本次程序耗时约 25.809 秒。作者 evidence 中记录的原始 `ok ... 25.745s` 和各项数值与 doc 表一致；本次运行在共享开发机上出现正常时间波动，分页 batches/candidates/visible 计数保持一致。

未在本评审中独立运行：`go test ./...`（作者 evidence 5.6 记录 exit 0，但本评审没有把它当作自己的实测证据）、`-tags=integration` article benchmark（NOT RUN）、`-race`、真实 PostgreSQL、HTTP、SMTP、Unix Journal、真实 E2E/RTO 和全量 Manifest 七项。不能由本次结果外推。

## 6. 一致性与边界核对

- **真实生产路径**：benchmark 调用真实 `CommandService`、`ArticleQueryService`、`DeletionWorker`、`DeletionFinalizer`，内存实现只替代持久化、事务、身份状态和时钟接口，没有复制业务策略；未发现仅供 mock 自证的情况。
- **防止空转**：Create 校验持久化结果和 Version 1；Replace 每轮调用服务并校验最终标题/正文；分页在 setup 校验可见条数并重置计数，循环内每轮调用真实查询；Worker 校验 `b.N*100` 次 converger 调用；Replay 校验 `AlreadyAbsent == 1000`。未发现当前空转。
- **换算**：author evidence/doc 的 worker `49739 ns/op ÷ 100 = 497.39 ns/operation`、replay `52204 ns/op ÷ 1000 = 52.204 ns/tombstone` 与循环结构一致。本次独立运行对应为 `43664 ÷ 100 = 436.64` 和 `58124 ÷ 1000 = 58.124`，同样是不同样本的正常波动。
- **文档数字**：doc 4.1–4.3 的数字与 evidence 5.4 原始输出一致；未发现 doc 自行放大或编造数值。doc 明确说明单次运行、无 P95、无 SLO/容量/RTO 阈值；未发现“达标/不达标”结论。
- **默认依赖**：默认 benchmark 只使用内存 fake、固定时钟和进程内接口，不读取 DSN、环境变量或建立 PostgreSQL 连接；`go test` 目标命令通过。
- **A-PERF 状态**：A-PERF-01 与 A-PERF-04 保持 partial；A-PERF-02、A-PERF-03、A-PERF-05 保持 blocked。尤其 A-PERF-03 明确保留 dispatcher/SMTP/lease/retry 未覆盖，未编造阈值。

## 7. 未验证风险与复审条件

未验证边界：

- 真实 PostgreSQL 的查询计划、索引、锁等待、事务提交、连接池和实际扫描行数。
- HTTP 请求解码、`262144` wire bytes 边界、响应编码及协议层容量。
- dispatcher 的真实 SMTP 投递、lease、重试积压、并发、超时和 max_attempts。
- Unix 文件持久卷上的 Journal append、file/directory `fsync` 和崩溃恢复。
- 真实 `deletion_pending` 年龄、物理删除、tombstone inventory、evidence 重算、RTO。
- 真实 E2E、Host、部署、并发/缓存/热点、生产流量和容量目标。
- 真实 PostgreSQL integration benchmark、race 回归、`go test ./...` 的独立复跑和全量 Manifest 七项。
- 未获批准的 SLO、P95、容量、积压和 retention 阈值；本评审不作达标判断。

复审条件：

1. 作者补充 dispatcher 进程内 baseline，或 Lead/Planner 明确修订 S2.3 关闭条件并记录范围变化；复审原 finding `S23-R01-F001`。
2. 任务卡责任人修正 `tasks.md:1429` 的无关状态文本；复审 `S23-R01-F002`，确认不再把 identity/Manifest 结果当成 S2.3 证据。
3. 修正 doc 的固定时钟说明；复审 `S23-R01-F003`。
4. 重新提供 benchmark、doc 和 evidence 的固定 hash、原始命令输出；重新运行目标 benchmark，并核对新证据中的计数、状态和未覆盖边界。
5. 对原 finding 逐项复核后，仍需 Lead 现场核证实际 diff、所有权和交付；本报告不代替人的接收或后续授权。

## 8. 写入声明

本次评审只读被评审对象，唯一写入为本报告路径。未修改 benchmark、doc、evidence、任务卡、设计或生产代码；未执行 Git 写操作；未删除、移动或归档任何文件；未勾选 checkbox；未声称 S2.3 已关闭。
