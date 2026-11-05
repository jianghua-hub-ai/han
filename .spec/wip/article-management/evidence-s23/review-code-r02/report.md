# S2.3 独立代码评审复审报告（review-code-r02）

## 1. 结论与固定对象

- objectType：`code`，并核对配套性能文档、WIP 证据与任务卡 S2.3 段。
- 评审角色：非作者独立 Reviewer（`add-reviewer`）。
- 工作区：`D:\AID\VSCode\Codex\Han`。
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 复审时点：2026-11-02（Asia/Shanghai）。
- verdict：**PASS**（无阻断项；1 个 LOW 非阻断 finding）。
- 原 finding 状态：`S23-R01-F001` **CLOSED**；`S23-R01-F002` **CLOSED**；`S23-R01-F003` **CLOSED**。
- 本报告不勾选任务卡，不声称 S2.3 已关闭，不代替 Lead 的现场核证或人的接收。

固定对象及本次核对值：

| 对象 | 字节数 | 行数（LF） | SHA256 |
|---|---:|---:|---|
| `test/performance/article/benchmark_test.go` | 22,314 | 821 | `8895BD90F131AC803D60D1FC9EC75B5E5B373BA5388B308E67FB794246BA2E58` |
| `docs/benchmarks/article.md` | 12,921 | 199 | `3E2482DAC54F4DF9F2398F81CEC24036990BD658C80FB7B21787E0CBE53315BB` |
| `.spec/wip/article-management/evidence-s23/report.md` | 16,326 | 274 | `E7755A3B30E2A302BC33A5E60E0BB72D9A8CEE838EDE9FBE7900CF6CF7D0F782` |
| `.add/project/knowledge/tasks/tasks.md`（S2.3 段依据） | 896,608 | 3,287 | `97EFE563B093FEA7C85CD8290CE8B7D808AFE5368AC950EE405D4F9DEDBC9B7B` |

四项字节数和 SHA256 与输入固定值一致；完成目标 benchmark 后再次核对，四个 hash 均未变化。工作树包含他人既有未提交改动和未跟踪文件，本次没有要求干净工作树，也没有回滚、移动、删除或归档任何他人内容。

`.spec/wip/article-management/evidence-s23/report.md` 仍判定为 **WIP 证据载体**，不是产品交付物。任务卡产品文件清单只列 benchmark 和 baseline 文档；evidence 位于 ADD 约定的 `.spec/wip/<scope>/`，记录固定上下文、原始输出和边界。其保留、归档或清理由 Lead/Steward 另行决定，本次不处置唯一证据。

## 2. 两轮实际检查范围

- **Round 1（修复对象与验收陈述）**：逐字读取 821 行 benchmark、199 行 doc、274 行 evidence 和 `tasks.md:1420`–`:1435`；重点检查新增 `BenchmarkNoticeDispatcherEmptyRound`（`benchmark_test.go:700`–`:821`）、doc §4.4（`:151`–`:157`）与 §7/§8（`:184`–`:199`）、evidence §10（`:199`–`:274`）及固定时钟说明。
- **Round 2（真实生产者、依赖与上游依据）**：沿真实调用链核对 `internal/article/notice_dispatcher.go` 的 `Reconcile`（`:141`–`:160`）、`reconcile`（`:393`–`:419`）、`transaction`（`:421`–`:433`）、`claimDue`（`:461`–`:474`）、`maintain`（`:561`–`:569`）、`dispatchWork`（`:1103`–`:1116`）；核对 `internal/article/repository.go:610`–`:647` 的真实 `LockExhaustedNoticeRecipient`、`LockDueNoticeRecipient` 与 `readNoticeRow`；抽查 `command_service.go:527`/`:626`、`query_service.go:112`/`:316`、`policy.go`、`deletion_worker.go:126`、`deletion_finalizer.go:134`/`:186` 的基准实际入口；核对 Article test-plan `:876`–`:884`、`:1025`、`:1245` 的 A-PERF 依据。
- 结论不扩展到真实 PostgreSQL、HTTP、SMTP、Unix Journal、真实 E2E、RTO、全仓重构或未来性能任务。

## 3. 原 finding 复审

### S23-R01-F001（dispatcher 基线）：CLOSED

依据：

1. `BenchmarkNoticeDispatcherEmptyRound` 在 `benchmark_test.go:761` 调用真实 `article.NewTransferNoticeDispatcher`，在 `:767` 注入真实类型 `article.Repository{}`，并在 `:791` 与 `:806` 调用生产 `TransferNoticeDispatcher.Reconcile`。不存在以自写 dispatcher 或 mock repository 替换业务实现的情况。
2. 真实 Repository 的 `LockExhaustedNoticeRecipient` / `LockDueNoticeRecipient` 在 `repository.go:611`–`:617` 执行生产 SQL 参数，并统一进入 `readNoticeRow`（`:630`–`:647`）；基准仅把事务边界替换为 `noticeTransactionCounter.BeginTx`（`benchmark_test.go:733`）和固定返回 `pgx.ErrNoRows` 的 `noticeEmptyTx.QueryRow`（`:722`）。因此空结果仍经过真实 Repository 读取与错误翻译路径，没有复制仓储逻辑。
3. “2 次事务/轮”的推导与源码一致：`reconcile` 的维护循环先调用 `maintainWork`→`maintain`→`transaction`（1 次），`LockExhaustedNoticeRecipient` 无结果后 `found=false` 并退出维护循环；随后 `dispatchWork`→`claimDue`→`transaction`（1 次），`LockDueNoticeRecipient` 无结果时 `dispatchWork` 返回 `stop=true`（`notice_dispatcher.go:1109`–`:1112`），结束本轮。`BatchLimit=2` 不会使空轮产生 4 次事务。
4. 防空转充分：setup 先执行一轮并断言 `BeginTx == 2`（`benchmark_test.go:791`–`:800`），重置计数后计时循环每轮只调用一次 `Reconcile`（`:805`–`:808`），循环后断言计数严格等于 `b.N × 2`（`:815`–`:817`）；同时断言 `Reconcile` 返回 nil 和 `dispatcher.Err()` 为 nil。事务数偏少或偏多都会失败，`rounds/op=1`、`transactions/op=2` 不是无来源的静态标签。
5. 范围判断：该 benchmark 只测 dispatcher 的无工作调度轮，不覆盖 SMTP 投递、lease、重试积压、并发、超时或 `max_attempts`。任务卡要求的是 dispatcher baseline，且 A-PERF-03 仍由 AM-Q2、容量、RPO/RTO、retention 阻塞；空轮是 r01 建议允许的“无工作或单次受限 claim 轮次”最小值，不把 A-PERF-03 说成通过。因此原 HIGH finding 关闭，但覆盖范围仍明确受限。

边界说明：被执行路径还替换了 `AcquireWork`（无锁直通）和 `NewTraceID`（固定合法 trace），`Gate`/`Mailer`/`Audit` 在空结果路径上未被调用。替换的是事务/工作获取等边界，而非 Repository 查询或 dispatcher 调度逻辑；doc §2.1、§3.5 与 evidence §10.2 已如实披露这些确定性依赖。

### S23-R01-F002（任务卡状态文本）：CLOSED

依据：

- `tasks.md:1429` 已替换为本卡真实状态：仅陈述 S2.3 的 gofmt/vet/build/目标 benchmark 已由 Lead 与独立 Reviewer 实跑 exit 0，`go test ./...` 仅作者自报且 Lead 未复跑，r01 为 CHANGES REQUIRED（F001/F002），S2.3 未关闭且本行不构成 PASS 证据。
- 该行不再把 identity integration 或 Manifest 七项 PASS 当作 S2.3 证据；只把旧内容作为 r01 F002 的历史说明。
- `tasks.md:1420` 仍为 `[ ]`，本次未勾选、未改写。该 checkbox 保持开放，r02 PASS 不表示任务关闭；后续状态与 checkbox 由 Lead/任务责任人按流程处理。

### S23-R01-F003（固定时钟说明）：CLOSED

依据：

- `docs/benchmarks/article.md:81` 已明确写为 `2026-01-01T00:00:00Z`（UTC）。
- `test/performance/article/benchmark_test.go:39` 的实际值为 `time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)`，两者日期、时刻和时区一致。
- evidence `:206` 仍把该时钟写成 `benchmark_test.go:37`，属于行号漂移的 LOW 引用问题，不影响 doc 与代码值的一致性；单独记录为 `S23-R02-F001`。

## 4. Findings（按严重度排序）

### [LOW] 证据报告的固定时钟行号已漂移

- finding ID：`S23-R02-F001`
- 对象/修订：`.spec/wip/article-management/evidence-s23/report.md`，SHA256 `E7755A3B30E2A302BC33A5E60E0BB72D9A8CEE838EDE9FBE7900CF6CF7D0F782`
- 位置：`.spec/wip/article-management/evidence-s23/report.md:206`
- 现状：evidence §10.1 写“未改 `benchmark_test.go:37` 的时钟”；当前文件中 `:37` 是 import 列表的右括号，实际 `benchmarkNow` 在 `:39`。
- 触发：审计者按行号定位固定时钟值时会看到错误行，可能误判证据引用过期或文件版本不一致。
- 影响：不影响 benchmark 数值、doc 内容或 A-PERF 状态；仅降低证据的可追溯性。
- 建议：后续证据修订把 `:37` 改为 `:39`，或改为引用符号 `benchmarkNow` 以避免行号漂移。不得为此改动产品代码或已运行结果。
- 验证：复核 `benchmark_test.go:39` 的值，并确认 evidence 引用指向该行。

## 5. 非阻断建议与待决定项

- **建议（证据历史段）**：evidence §7 的 A-PERF-03 行仍保留 r01 前的“仅有 worker 无操作调度开销”表述；§10 已追加 dispatcher 空轮结果并明确其边界，因此不把历史段当作当前结论。若 evidence 长期作为交付证据保留，建议在 §7 标题或该行加“修复前快照，以 §10 为准”的指针。
- **待决定（dispatcher 覆盖）**：是否在未来把 dispatcher 基线扩展到真实 claim、lease 或 SMTP 路径由 Lead/Planner 依据 A-PERF-03 阻塞项另行安排；本次不以空轮外推吞吐、P95 或容量。
- **待决定（任务状态）**：`tasks.md:1420` 仍为 `[ ]`，S2.3 是否关闭、是否改 `[x]` 以及证据归档由 Lead/任务责任人处理；Reviewer 不修改 checkbox，也不把本报告当作关闭授权。

## 6. 数值、文档与 evidence 核对

- doc §4.4 的 dispatcher 行（`article.md:155`）为 `2073 ns/op | 1.000 rounds/op | 2.000 transactions/op | 2128 B/op | 21 allocs/op`，与 evidence §10.3 原始输出 `:255` 完全一致；本次独立运行得到 `2353 ns/op | 1.000 rounds/op | 2.000 transactions/op | 2128 B/op | 21 allocs/op`，属于不同样本的正常波动，不改写固定证据。
- doc §4.1–§4.3 的创建、修改、分页、worker、finalizer、replay 数值逐项与 evidence §10.3 `:240`–`:254` 一致；doc `:60` 的 `26.087s` 与 evidence `:257` 一致。
- 换算复核：
  - evidence §5.4/§6 的 worker `49739 ns/op ÷ 100 operations = 497.39 ns/operation`，replay `52204 ns/op ÷ 1000 tombstones = 52.204 ns/tombstone`。
  - evidence §10.3/doc §4.3 的 worker `39649 ns/op ÷ 100 = 396.49 ns/operation`，replay `42202 ns/op ÷ 1000 = 42.202 ns/tombstone`。
  - 循环结构与换算相符：worker benchmark 每轮调用一次 `worker.Reconcile`，`Reconcile` 覆盖 100 个 operation，循环后断言 `b.N × 100` 次 converger 调用（`benchmark_test.go:493`–`:507`）；replay 每轮传入 1000 个 tombstone，并断言 `AlreadyAbsent == 1000`（`:678`–`:697`）。
- doc §4.4 只描述“无待处理通知时一次 Reconcile 的进程内调度开销”，没有把空轮说成 SMTP、lease、重试或生产吞吐；§7/§8 继续将 A-PERF-03 标为 blocked，并明确“不构成 A-PERF-03 通过”。A-PERF-01/04 保持 partial，A-PERF-02/03/05 保持 blocked；未发现未批准的 SLO、容量、P95、积压、retention 或 RTO 阈值，也没有“达标/不达标”结论。
- 默认 benchmark 使用内存 repository、固定时钟和进程内 fake，不读取 PG/DSN 或环境变量；目标命令实际 exit 0。

## 7. 实际运行命令与结果

以下命令均由本评审在当前工作区实际执行，未以作者或 Lead 摘要代替：

```text
=== gofmt -l test/performance/article ===
exit=0
stdout_bytes=0

=== go vet ./test/performance/article/... ===
exit=0

=== go build ./... ===
exit=0
```

```text
go test -run ^$ -bench . -benchmem ./test/performance/article
goos: windows
goarch: amd64
pkg: han/test/performance/article
cpu: 12th Gen Intel(R) Core(TM) i7-12700H
BenchmarkCreateArticleMaxContent/ascii-20       	  122774	     10495 ns/op	1924.78 MB/s	      16 B/op	       1 allocs/op
BenchmarkCreateArticleMaxContent/unicode-3-byte-20         	   28413	     42903 ns/op	1412.47 MB/s	      16 B/op	       1 allocs/op
BenchmarkReplaceContentMaxContent/ascii-20                 	  127544	      7979 ns/op	2531.63 MB/s	       0 B/op	       0 allocs/op
BenchmarkReplaceContentMaxContent/unicode-3-byte-20        	   27004	     43486 ns/op	1393.57 MB/s	       0 B/op	       0 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-000-20    	   51174	     22345 ns/op	         1.000 batches/op	       100.0 candidates/op	        20.00 visible-items/op	   31605 B/op	     143 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-050-20    	   38215	     31833 ns/op	         1.000 batches/op	       100.0 candidates/op	        20.00 visible-items/op	   34809 B/op	     193 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-090-20    	   10000	    113221 ns/op	         3.000 batches/op	       300.0 candidates/op	        20.00 visible-items/op	   93679 B/op	     639 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-100-20    	    3864	    361723 ns/op	        11.00 batches/op	      1000 candidates/op	         0 visible-items/op	  309488 B/op	    2131 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-000-20    	   42394	     32511 ns/op	         1.000 batches/op	       100.0 candidates/op	        50.00 visible-items/op	   43908 B/op	     173 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-050-20    	   20112	     77291 ns/op	         2.000 batches/op	       200.0 candidates/op	        50.00 visible-items/op	   72064 B/op	     386 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-090-20    	    5064	    204926 ns/op	         6.000 batches/op	       600.0 candidates/op	        50.00 visible-items/op	  200312 B/op	    1278 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-100-20    	    3844	    529469 ns/op	        11.00 batches/op	      1000 candidates/op	         0 visible-items/op	  347505 B/op	    2131 allocs/op
BenchmarkDeletionWorkerReconcile100-20                     	   25899	     43137 ns/op	       100.0 operations/op	   27472 B/op	     404 allocs/op
BenchmarkDeletionFinalizerCompletedNoop-20                 	18986049	        60.34 ns/op	       0 B/op	       0 allocs/op
BenchmarkReplayAllCompletedTombstones1000-20               	   28332	     42380 ns/op	      1000 tombstones/op	       0 B/op	       0 allocs/op
BenchmarkNoticeDispatcherEmptyRound-20                     	  743397	      2353 ns/op	         1.000 rounds/op	         2.000 transactions/op	    2128 B/op	      21 allocs/op
PASS
ok  	han/test/performance/article	25.593s
bench_exit=0
```

## 8. 未验证风险与边界

本次 PASS 仅覆盖上述对象、进程内路径和命令结果。以下仍未验证或仍 blocked：

- 真实 SMTP 投递、通知 lease、重试积压、并发、超时和 `max_attempts`。
- 真实 PostgreSQL 的查询计划、索引、锁等待、事务提交、连接池和实际扫描行数。
- HTTP 请求解码、`262144` wire bytes 边界与响应编码。
- Unix 文件持久卷上的 Journal append、file/directory `fsync` 与崩溃恢复。
- 真实 `deletion_pending` 年龄、物理删除、tombstone inventory、evidence 重算和 RTO。
- AM-Q2、容量目标、生产工作负载、P95、SLO、retention、RPO/RTO 及告警阈值仍未确认，不作达标判断。
- 未独立运行 `go test ./...`、`-race` 或 integration benchmark；不把作者自报当作本评审实测。
- 未执行真实 E2E、生产流量、并发/缓存/热点或部署验证。

## 9. 写入声明

本次复审只读被评审对象；唯一写入为本报告 `.spec/wip/article-management/evidence-s23/review-code-r02/report.md`。未修改 benchmark、doc、evidence、任务卡、设计或生产代码；未执行 Git 写操作；未删除、移动或归档任何文件；未勾选 checkbox；未声称 S2.3 已关闭或已验收。
