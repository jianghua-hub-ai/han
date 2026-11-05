# article-management-S2.3 实施证据

## 1. 任务与固定上下文

- 角色：`add-backend-dev`
- 主 Skill：`.codex/skills/add-task-implementation/SKILL.md`
- 任务：`article-management-S2.3 建立 article 容量与性能基线`（`tasks.md:1420`–`:1435`）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 本次未执行任何 Git 写操作，未暂存、未提交、未切换分支。

## 2. 交付文件

| 文件 | 状态 | 字节数 |
|---|---|---:|
| `test/performance/article/benchmark_test.go` | 新增 | 18205 |
| `docs/benchmarks/article.md` | 新增 | 10712 |
| `.spec/wip/article-management/evidence-s23/report.md` | 新增 | 本文件 |

未修改 `go.mod`、`go.sum`、`cmd/**`、`internal/**`、既有测试、`tasks.md`、design、Plan 或证据目录外文件。

## 3. 实现说明

基准调用当前代码库真实存在的 Article 服务与协调器：

- `CommandService.Create`
- `CommandService.ReplaceContent`
- `ArticleQueryService.ListArticles`
- `DeletionWorker.Reconcile`
- `DeletionFinalizer.ConvergeDeletionOperation`
- `DeletionFinalizer.ReplayAllTombstones`

持久化、身份状态、事务与时钟使用确定性进程内实现，以满足既有接口。`pgx.Tx` 只实现被调用路径实际使用的 `Begin`/`Commit`/`Rollback`，其余方法通过嵌入的 nil 接口保留；若被意外调用会直接暴露，而不是伪造 SQL。

本文数字是服务逻辑、分页投影、调度与协调循环的进程内基线，不是 PostgreSQL、HTTP、SMTP、Journal 或 RTO 结论。

## 4. 运行环境

| 项 | 值 |
|---|---|
| 日期 | 2026-11-02（Asia/Shanghai） |
| Go | `go1.27.0 windows/amd64` |
| OS | Microsoft Windows 11 专业版，版本 10.0.22621，64 位 |
| CPU | 12th Gen Intel(R) Core(TM) i7-12700H，14 物理核 / 20 逻辑处理器 |
| 内存 | 系统可见 16495096 KiB（约 15.73 GiB） |
| `GOMAXPROCS` | 环境变量未设置；Go runtime 默认，基准名后缀 `-20` |

## 5. 验证命令与真实结果

### 5.1 `gofmt -l test/performance/article`

- exit code：`0`
- 输出：空

```text
gofmt_exit=0
```

### 5.2 `go vet ./test/performance/article/...`

- exit code：`0`
- 输出：空

```text
vet_exit=0
```

### 5.3 `go build ./...`

- exit code：`0`
- 输出：空

```text
build_exit=0
```

### 5.4 `go test -run "^$" -bench . -benchmem ./test/performance/article`

- exit code：`0`
- 完整输出：

```text
goos: windows
goarch: amd64
pkg: han/test/performance/article
cpu: 12th Gen Intel(R) Core(TM) i7-12700H
BenchmarkCreateArticleMaxContent/ascii-20       	  118797	     10313 ns/op	1958.74 MB/s	      16 B/op	       1 allocs/op
BenchmarkCreateArticleMaxContent/unicode-3-byte-20         	   21871	     50512 ns/op	1199.70 MB/s	      16 B/op	       1 allocs/op
BenchmarkReplaceContentMaxContent/ascii-20                 	  135840	      9242 ns/op	2185.76 MB/s	       0 B/op	       0 allocs/op
BenchmarkReplaceContentMaxContent/unicode-3-byte-20        	   22378	     55042 ns/op	1100.98 MB/s	       0 B/op	       0 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-000-20    	   49725	     28974 ns/op	         1.000 batches/op	       100.0 candidates/op	        20.00 visible-items/op	   31605 B/op	     143 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-050-20    	   37382	     34853 ns/op	         1.000 batches/op	       100.0 candidates/op	        20.00 visible-items/op	   34810 B/op	     193 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-090-20    	   10000	    111019 ns/op	         3.000 batches/op	       300.0 candidates/op	        20.00 visible-items/op	   93681 B/op	     639 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-100-20    	    3554	    351991 ns/op	        11.00 batches/op	      1000 candidates/op	         0 visible-items/op	  309488 B/op	    2131 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-000-20    	   36522	     37699 ns/op	         1.000 batches/op	       100.0 candidates/op	        50.00 visible-items/op	   43909 B/op	     173 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-050-20    	   17383	     73667 ns/op	         2.000 batches/op	       200.0 candidates/op	        50.00 visible-items/op	   72062 B/op	     386 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-090-20    	    6519	    242608 ns/op	         6.000 batches/op	       600.0 candidates/op	        50.00 visible-items/op	  200310 B/op	    1278 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-100-20    	    2985	    391189 ns/op	        11.00 batches/op	      1000 candidates/op	         0 visible-items/op	  347504 B/op	    2131 allocs/op
BenchmarkDeletionWorkerReconcile100-20                     	   25698	     49739 ns/op	       100.0 operations/op	   27472 B/op	     404 allocs/op
BenchmarkDeletionFinalizerCompletedNoop-20                 	15888588	        71.24 ns/op	       0 B/op	       0 allocs/op
BenchmarkReplayAllCompletedTombstones1000-20               	   23178	     52204 ns/op	      1000 tombstones/op	       0 B/op	       0 allocs/op
PASS
ok  	han/test/performance/article	25.745s
bench_exit=0
```

### 5.5 integration 基准命令

未创建 `-tags=integration` 基准；该命令为 **NOT RUN**，不记为 PASS。

原因：dispatcher、真实 PG finalizer、tombstone inventory 与 evidence 重算需要真实 PostgreSQL/SMTP/Journal，超出本次两个新增文件的可验证边界；对应 A-PERF-02/03/05 保持 blocked。

### 5.6 `go test ./...`

- exit code：`0`
- 完整输出：

```text
ok  	han/cmd/han-server	32.716s
ok  	han/internal/article	2.960s
?   	han/internal/audit	[no test files]
ok  	han/internal/idempotency	3.863s
ok  	han/internal/identity	2.855s
ok  	han/internal/mailer	5.981s
ok  	han/internal/migration	0.303s
ok  	han/internal/platform	0.361s
ok  	han/internal/privacy	0.337s
ok  	han/internal/recovery	3.149s
ok  	han/internal/security	0.347s
ok  	han/internal/session	0.285s
ok  	han/internal/testenv	0.281s
ok  	han/test/performance/article	0.391s [no tests to run]
gotest_exit=0
```

## 6. 实测数字摘要

| 维度 | 观测值 |
|---|---|
| 创建 200/20000 rune ASCII | 10313 ns/op，1958.74 MB/s，16 B/op，1 alloc/op |
| 创建 200/20000 rune 3 字节 UTF-8 | 50512 ns/op，1199.70 MB/s，16 B/op，1 alloc/op |
| 修改 200/20000 rune ASCII | 9242 ns/op，2185.76 MB/s，0 B/op，0 alloc/op |
| 修改 200/20000 rune 3 字节 UTF-8 | 55042 ns/op，1100.98 MB/s，0 B/op，0 alloc/op |
| 列表 limit 20 / hidden 0% | 28974 ns/op，100 candidates/op，1 batch/op |
| 列表 limit 20 / hidden 50% | 34853 ns/op，100 candidates/op，1 batch/op |
| 列表 limit 20 / hidden 90% | 111019 ns/op，300 candidates/op，3 batches/op |
| 列表 limit 20 / hidden 100% | 351991 ns/op，1000 candidates/op，11 batches/op |
| 列表 limit 50 / hidden 0% | 37699 ns/op，100 candidates/op，1 batch/op |
| 列表 limit 50 / hidden 50% | 73667 ns/op，200 candidates/op，2 batches/op |
| 列表 limit 50 / hidden 90% | 242608 ns/op，600 candidates/op，6 batches/op |
| 列表 limit 50 / hidden 100% | 391189 ns/op，1000 candidates/op，11 batches/op |
| Worker 100 操作 | 49739 ns/op，约 497.39 ns/operation |
| completed finalizer no-op | 71.24 ns/op |
| ReplayAll 1000 completed tombstone | 52204 ns/op，约 52.204 ns/tombstone |

**本基线不含阈值判定。** 未使用未批准的 SLO、容量目标、P95、积压或 RTO 门槛，不作“达标/不达标”结论。

## 7. A-PERF-01～05 状态

| ID | 状态 | 依据 |
|---|---|---|
| A-PERF-01 | partial | 已测 200/20000 rune 的 ASCII 与 3 字节 UTF-8 创建/修改；未覆盖 `262144` wire bytes HTTP 路径；无批准容量/时延目标 |
| A-PERF-02 | blocked（AM-Q2 / 容量目标 / RPO-RTO / retention 未确认） | 仅有测试专用 1000 篇、limit 20/50、隐藏密度 0/50/90/100% 的候选扫描与投影观测；生产工作负载与 P95 未确认 |
| A-PERF-03 | blocked（AM-Q2 / 容量目标 / RPO-RTO / retention 未确认） | dispatcher、SMTP 延迟、lease、重试积压、并发/超时/`max_attempts` 未覆盖；仅有 worker 无操作调度开销 |
| A-PERF-04 | partial | 已测 worker 批次调度、completed finalizer no-op、replay no-op；未测 `deletion_pending` 年龄、Journal `fsync`、真实 finalizer、吞吐与积压 |
| A-PERF-05 | blocked（AM-Q2 / 容量目标 / RPO-RTO / retention 未确认） | 仅有 1000 个 completed tombstone 的协调循环；tombstone inventory、真实 ReplayAll、evidence 重算、备份规模与 RTO 未覆盖 |

## 8. 未覆盖项与 unknown

- PostgreSQL 真实查询计划、索引、锁等待、事务提交、轮询与连接池。
- HTTP `262144` wire bytes 边界、请求解码与响应编码。
- dispatcher、SMTP 投递延迟、通知 lease、重试积压与 `max_attempts`。
- Unix 文件持久卷上的 Journal append/file+directory `fsync`。
- 真实 `deletion_pending` 年龄、物理删除、tombstone inventory、evidence 重算与 RTO。
- AM-Q2、容量目标、RPO/RTO、retention、生产文章规模、隐藏分布、QPS、P95。
- 并发、缓存、热点与生产流量下的行为。
- 数据库实际扫描行数与内存仓库返回候选行的差异。
- 单次运行未提供置信区间或分位数。

## 9. 停止条件与越界自查

停止条件均未触发：

- 未修改写集外文件即可让基准通过。
- 默认基准命令无需数据库或环境变量，exit code 为 `0`。
- 未发明容量阈值或 SLO。
- 未新增依赖到 `go.mod`。
- 现有 Article 实现具备进程内构造被测对象所需的能力。

边界自查：

- 仅新增 3 个文件：基准、基线文档、本证据报告。
- 未修改 `cmd/**`、`internal/**`、既有 `test/**` 文件、`go.mod`、`go.sum`、`tasks.md`、design、Plan。
- 未执行 Git 写操作；未删除、移动或归档任何文件。
- 未勾选卡面 checkbox。
- 未声称任何人验收、未声称独立 Review 通过、未声称 S2.3 已关闭。

## 10. r01 修复轮次（2026-11-02）

本节为独立评审 `review-code-r01` 后的修复追加，保留上文原始记录不删改。固定上下文仍为 `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

### 10.1 finding 对照

- `S23-R01-F001`（HIGH，dispatcher 基线缺失）：已新增 `BenchmarkNoticeDispatcherEmptyRound`，构造真实 `article.TransferNoticeDispatcher` 与真实 `article.Repository`，调用生产 `Reconcile` 空轮路径。
- `S23-R01-F003`（LOW，固定时钟说明）：已把 `docs/benchmarks/article.md` 的“时钟固定在此次运行日期”改为 `2026-01-01T00:00:00Z`（UTC）；未改 `benchmark_test.go:37` 的时钟。
- `S23-R01-F002`（MEDIUM，`tasks.md:1429`）不在本次写集；未修改 `tasks.md`，交由任务卡责任人或 Lead 处理。

### 10.2 实现与防空转

- 新增 fake：`NoticeTransactions.BeginTx` 返回自定义 `noticeEmptyTx`；其 `QueryRow(...).Scan(...)` 恒返回 `pgx.ErrNoRows`，`Commit`/`Rollback` 返回 nil；`Repository` 使用真实无状态 `article.Repository`，未伪造仓储。
- Gate 恒放行；Mailer/Audit/IDs/Logger 为最小 fake；`WorkerID` 为合法 UUID。
- `NewTraceID` 返回 `01ARZ3NDEKTSV4RRFFQ69G5FAV`。生产 `platform.ContextWithTraceID` 要求 26 位 Crockford base32 trace 标识，不接受 UUID 字符串；该值取自 `internal/article` 已验证测试 fixture。
- 配置取自 `internal/article/notice_dispatcher_test.go` 的已验证合法配置：`PollInterval=1s`、`BatchLimit=2`、`LeaseDuration=10s`、`DBOperationBudget=1s`、`BackoffInitial=1s`、`BackoffMax=8s`、`SMTPOperationBudget=2s`。
- `BeginTx` 期望次数推导：空轮 `Reconcile` 先调用 `maintainWork` → `maintain` → `transaction`（1 次），无待处理通知即退出维护循环；随后 `dispatchWork` → `claimDue` → `transaction`（1 次），无待处理通知返回 `stop=true` 结束本轮。因此每轮固定 2 次，断言 `2 × b.N`；setup 先执行一轮并断言 2 次，循环后再次断言，读数不足即 `b.Fatalf`。每轮同时断言 `Reconcile` 返回 nil 且 `Err()` 为 nil。
- 指标：`ns/op` 由 Go 基准报告；`transactions/op` 与 `rounds/op` 由 `b.ReportMetric` 报告。

### 10.3 本次原始命令输出

```text
=== gofmt -l test/performance/article ===
gofmt_exit=0
```

```text
=== go vet ./test/performance/article/... ===
vet_exit=0
```

```text
=== go build ./... ===
build_exit=0
```

```text
goos: windows
goarch: amd64
pkg: han/test/performance/article
cpu: 12th Gen Intel(R) Core(TM) i7-12700H
BenchmarkCreateArticleMaxContent/ascii-20       	  143481	      8615 ns/op	2344.62 MB/s	      16 B/op	       1 allocs/op
BenchmarkCreateArticleMaxContent/unicode-3-byte-20         	   28994	     43165 ns/op	1403.91 MB/s	      16 B/op	       1 allocs/op
BenchmarkReplaceContentMaxContent/ascii-20                 	  155080	      7445 ns/op	2713.35 MB/s	       0 B/op	       0 allocs/op
BenchmarkReplaceContentMaxContent/unicode-3-byte-20        	   28098	     41080 ns/op	1475.17 MB/s	       0 B/op	       0 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-000-20    	   56012	     24432 ns/op	         1.000 batches/op	       100.0 candidates/op	        20.00 visible-items/op	   31605 B/op	     143 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-050-20    	   32910	     32770 ns/op	         1.000 batches/op	       100.0 candidates/op	        20.00 visible-items/op	   34809 B/op	     193 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-090-20    	   12825	     93038 ns/op	         3.000 batches/op	       300.0 candidates/op	        20.00 visible-items/op	   93680 B/op	     639 allocs/op
BenchmarkMemberListHiddenDensity/limit-20/hidden-100-20    	    2846	    404895 ns/op	        11.00 batches/op	      1000 candidates/op	         0 visible-items/op	  309488 B/op	    2131 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-000-20    	   35694	     34016 ns/op	         1.000 batches/op	       100.0 candidates/op	        50.00 visible-items/op	   43909 B/op	     173 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-050-20    	   18110	     62893 ns/op	         2.000 batches/op	       200.0 candidates/op	        50.00 visible-items/op	   72062 B/op	     386 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-090-20    	    6562	    202214 ns/op	         6.000 batches/op	       600.0 candidates/op	        50.00 visible-items/op	  200312 B/op	    1278 allocs/op
BenchmarkMemberListHiddenDensity/limit-50/hidden-100-20    	    3506	    366175 ns/op	        11.00 batches/op	      1000 candidates/op	         0 visible-items/op	  347504 B/op	    2131 allocs/op
BenchmarkDeletionWorkerReconcile100-20                     	   26412	     39649 ns/op	       100.0 operations/op	   27472 B/op	     404 allocs/op
BenchmarkDeletionFinalizerCompletedNoop-20                 	17070453	        59.42 ns/op	       0 B/op	       0 allocs/op
BenchmarkReplayAllCompletedTombstones1000-20               	   28794	     42202 ns/op	      1000 tombstones/op	       0 B/op	       0 allocs/op
BenchmarkNoticeDispatcherEmptyRound-20                     	  666784	      2073 ns/op	         1.000 rounds/op	         2.000 transactions/op	    2128 B/op	      21 allocs/op
PASS
ok  	han/test/performance/article	26.087s
bench_exit=0
```

### 10.4 新字节数与 SHA256

| 文件 | 字节数 | 行数（LF） | SHA256 |
|---|---:|---:|---|
| `test/performance/article/benchmark_test.go` | 22314 | 821 | `8895BD90F131AC803D60D1FC9EC75B5E5B373BA5388B308E67FB794246BA2E58` |
| `docs/benchmarks/article.md` | 12921 | 199 | `3E2482DAC54F4DF9F2398F81CEC24036990BD658C80FB7B21787E0CBE53315BB` |
| `.spec/wip/article-management/evidence-s23/report.md` | 本文件追加后变化 | — | 文件不能内嵌自身最终哈希；最终值见作者交付报告并由 Lead 现场复核 |

### 10.5 边界与声明

- 只测 dispatcher 进程内空轮调度；未测真实 SMTP 投递、lease、重试积压、`max_attempts`、并发/超时、真实 PostgreSQL。A-PERF-03 仍为 blocked。
- 未运行或声称运行 `go test ./...`、`-race`、integration。
- 未修改 `internal/**`、`tasks.md` 或其他文件；未执行 Git 写操作。
- 未声称 S2.3 关闭或独立评审通过。
