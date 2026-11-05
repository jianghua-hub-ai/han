# Article 容量与性能基线

- 任务：`article-management-S2.3 建立 article 容量与性能基线`
- 固定基线：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 运行日期：2026-11-02（Asia/Shanghai）
- 本次证据：`.spec/wip/article-management/evidence-s23/report.md`

**本基线不含阈值判定。** 本文只记录本次真实运行得到的观测值；未获批准的 SLO、容量目标、P95、积压或 RTO 门槛均未使用，也不对任何指标作“达标/不达标”结论。本基线不表示 `article-management-S2.3` 已关闭或已验收。

## 1. 范围

本文覆盖当前代码库中可在进程内直接调用的 Article 服务路径：

- 作者创建文章：`CommandService.Create`
- 作者完整替换标题与正文：`CommandService.ReplaceContent`
- 会员文章列表的分页扫描与可见性投影：`ArticleQueryService.ListArticles`
- 删除 worker 的单次调度：`DeletionWorker.Reconcile`
- 已完成删除操作的 finalizer 幂等短路：`DeletionFinalizer.ConvergeDeletionOperation`
- 已完成 tombstone 集合的 replay 协调循环：`DeletionFinalizer.ReplayAllTombstones`
- 通知 dispatcher 的空轮调度：`TransferNoticeDispatcher.Reconcile`（无待处理通知）

明确不覆盖：

- PostgreSQL 连接、SQL 解析、索引、锁等待、网络与磁盘
- HTTP 请求解码、`262144` wire bytes 边界、响应编码
- SMTP 投递、通知 lease 与重试积压未覆盖；进程内空轮调度开销已测
- Unix 文件持久卷上的 Journal append/file+directory `fsync`
- 真实 `deletion_pending` 年龄、真实物理删除、真实 evidence 重算与 RTO
- 并发竞争、连接池、缓存命中率与生产流量分布

## 2. 方法

### 2.1 被测对象

基准调用仓库中真实存在的生产服务与策略代码，不复制业务逻辑：

- `internal/article/command_service.go`
- `internal/article/query_service.go`
- `internal/article/policy.go`
- `internal/article/deletion_worker.go`
- `internal/article/deletion_finalizer.go`
- `internal/article/notice_dispatcher.go`

持久化、身份状态、事务与时钟使用确定性的进程内实现，以满足既有接口：

- `ArticleTransactor` / `pgx.Tx`：只实现被调用路径实际使用的 `Begin`、`Commit`、`Rollback`；其余 `pgx.Tx` 方法通过嵌入的 nil 接口保留，若被意外调用会直接暴露。
- `CreateArticleRepository`、`ArticleContentRepository`、`ArticleQueryRepository`、`DeletionFinalizerRepository`：内存实现。
- `ArticleAuthorStateReader`：内存状态表。
- `ArticleIDGenerator`、`ArticleClock`：预生成 UUID 与固定 UTC 时间。
- `NoticeTransactions` / `pgx.Tx`（dispatcher）：fake 事务只实现 `QueryRow`、`Commit`、`Rollback`；`QueryRow` 固定返回 `pgx.ErrNoRows`，不执行 SQL。

因此本文数字是**服务逻辑、分页投影、调度与协调循环的进程内基线**，不是数据库或端到端容量结论。

### 2.2 运行命令

```text
go test -run "^$" -bench . -benchmem ./test/performance/article
```

本次运行 exit code 为 `0`，`go test` 输出 `PASS`，包结果为 `ok han/test/performance/article 26.087s`。

### 2.3 统计口径

- 使用 Go `testing` 基准的聚合 `ns/op`、`B/op`、`allocs/op`。
- `MB/s` 来自 `b.SetBytes(title+body 字节数)`，只含标题与正文字节，不含协议与存储开销。
- 分页的 `candidates/op`、`batches/op`、`visible-items/op` 由基准内计数直接报告。
- dispatcher 的 `rounds/op`、`transactions/op` 由 `BeginTx` 实际调用计数报告；空轮恒为 1 round/op、2 transactions/op。
- 单次运行，未计算置信区间、P95 或其他分位数；`go test -bench` 本身不产出请求级分位数。

## 3. 工作负载定义

下列数值均为**测试专用假设**，不是生产配置、容量目标或批准的工作负载。

### 3.1 创建与修改

- 标题：200 个 rune，对应生产上限 `maxArticleTitleRunes = 200`。
- 正文：20000 个 rune，对应生产上限 `maxArticleBodyRunes = 20000`。
- `ascii`：标题 200 字节、正文 20000 字节，共 20200 字节。
- `unicode-3-byte`：标题 600 字节、正文 60000 字节，共 60600 字节；rune 数与 ASCII 组相同。
- 选择理由：同时观察 rune 上限校验与 UTF-8 字节量差异；不把 HTTP 层的 `262144` wire bytes 混入服务层结果。
- 创建时 ID 预生成，时钟固定为 `2026-01-01T00:00:00Z`（UTC）；修改时对同一篇文章连续执行完整内容替换，版本号随每次成功修改递增。

### 3.2 会员列表

- 数据集：1000 篇文章，1000 个作者，时间严格降序且均为 UTC。
- 隐藏方式：作者状态为 `suspended`；按索引每 100 篇中前 N 篇隐藏，形成均匀的 0%、50%、90%、100% 隐藏密度。
- 查询配置（测试专用）：`DefaultLimit=20`、`MaxLimit=50`、`CandidateBatchSize=100`。
- 请求 limit：分别测 20 与 50。
- 选择理由：1000 篇足以触发多候选批次；均匀隐藏密度便于观察候选读取量随密度增长；limit 20/50 只代表本基准的默认与上限，不是生产值。
- `candidates/op` 表示内存仓库返回给服务的候选行数，不等同于 PostgreSQL 实际扫描行数。

### 3.3 删除 worker

- 100 个合法 UUID 操作，`BatchSize=100`、`MaxBatchesPerWake=1`，converger 为无操作实现。
- 调度配置（测试专用）：`PollInterval=1ms`、`OperationBudget=1s`、`BackoffInitial=1ms`、`BackoffMax=10ms`、`ShutdownBudget=1s`。
- 选择理由：一次 `Reconcile` 覆盖一个完整批次的发现、trace 生成、逐操作 context 与收敛调用开销；不覆盖真实收敛 SQL 与 Journal。

### 3.4 Finalizer 与 replay

- Finalizer：操作状态为 `completed`，`ConvergeDeletionOperation` 在加载后直接返回；只测幂等短路路径。
- Replay：1000 个 `completed` tombstone，`ReplayAllTombstones` 每次都返回 `AlreadyAbsent`；只测协调循环与每项加载边界。
- 选择理由：这两条路径不依赖 PostgreSQL 或真实 Journal 即可执行，可观察“已完成/已不存在”情况的固定协调开销；不代表真实 finalizer 物理删除、Journal `fsync`、tombstone inventory、evidence 重算或 RTO。

### 3.5 通知 dispatcher 空轮

- 对象：真实 `TransferNoticeDispatcher.Reconcile`，依赖真实 `Repository`。
- 构造：`NoticeTransactions` 返回只实现 `QueryRow`/`Commit`/`Rollback` 的 fake `pgx.Tx`；`QueryRow` 固定返回 `pgx.ErrNoRows`，因此维护查询与抢占查询都得到“无待处理通知”。
- 其他依赖：Gate 恒放行；Mailer、Audit、IDs、Logger 为最小实现；`NewTraceID` 返回合法 trace 标识；`WorkerID` 为合法 UUID。
- 配置（测试专用，取自 `internal/article` 已验证配置）：`PollInterval=1s`、`BatchLimit=2`、`LeaseDuration=10s`、`DBOperationBudget=1s`、`BackoffInitial=1s`、`BackoffMax=8s`、`SMTPOperationBudget=2s`。
- 选择理由：空轮是 dispatcher 不依赖真实 SMTP 或 PostgreSQL 即可执行的最小生产调度路径；每轮固定打开 2 个事务（`maintain` 一次、`claimDue` 一次），基准用 `BeginTx` 计数断言 `2 × b.N` 并在不足时失败。
- 不覆盖：真实 SMTP 投递、lease 竞争、重试积压、并发、超时与 `max_attempts`。

## 4. 实测结果

运行环境见第 5 节；以下全部数字来自同一次真实运行。

### 4.1 创建与修改

| 基准 | ns/op | MB/s | B/op | allocs/op |
|---|---:|---:|---:|---:|
| `BenchmarkCreateArticleMaxContent/ascii` | 8615 | 2344.62 | 16 | 1 |
| `BenchmarkCreateArticleMaxContent/unicode-3-byte` | 43165 | 1403.91 | 16 | 1 |
| `BenchmarkReplaceContentMaxContent/ascii` | 7445 | 2713.35 | 0 | 0 |
| `BenchmarkReplaceContentMaxContent/unicode-3-byte` | 41080 | 1475.17 | 0 | 0 |

### 4.2 会员列表隐藏密度

| limit | hidden | ns/op | candidates/op | batches/op | visible-items/op | B/op | allocs/op |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 20 | 0% | 24432 | 100 | 1 | 20 | 31605 | 143 |
| 20 | 50% | 32770 | 100 | 1 | 20 | 34809 | 193 |
| 20 | 90% | 93038 | 300 | 3 | 20 | 93680 | 639 |
| 20 | 100% | 404895 | 1000 | 11 | 0 | 309488 | 2131 |
| 50 | 0% | 34016 | 100 | 1 | 50 | 43909 | 173 |
| 50 | 50% | 62893 | 200 | 2 | 50 | 72062 | 386 |
| 50 | 90% | 202214 | 600 | 6 | 50 | 200312 | 1278 |
| 50 | 100% | 366175 | 1000 | 11 | 0 | 347504 | 2131 |

100% 隐藏时出现 11 个批次，是因为前 10 批各返回 100 条候选、最后再用一个空批次确认耗尽。

### 4.3 Worker、Finalizer 与 Replay

| 基准 | ns/op | 其他报告值 | B/op | allocs/op |
|---|---:|---|---:|---:|
| `BenchmarkDeletionWorkerReconcile100` | 39649 | 100 operations/op | 27472 | 404 |
| `BenchmarkDeletionFinalizerCompletedNoop` | 59.42 | — | 0 | 0 |
| `BenchmarkReplayAllCompletedTombstones1000` | 42202 | 1000 tombstones/op | 0 | 0 |

由上述实测值换算：worker 约 `396.49 ns/operation`，replay 约 `42.202 ns/tombstone`。换算只做除法，不引入其他假设。

### 4.4 Dispatcher 空轮

| 基准 | ns/op | rounds/op | transactions/op | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| `BenchmarkNoticeDispatcherEmptyRound` | 2073 | 1.000 | 2.000 | 2128 | 21 |

该行只表示无待处理通知时一次 `Reconcile` 的进程内调度开销；不代表 SMTP 投递、lease 或重试路径。

## 5. 运行环境

| 项 | 值 |
|---|---|
| 日期 | 2026-11-02（Asia/Shanghai） |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| Go | `go1.27.0 windows/amd64` |
| OS | Microsoft Windows 11 专业版，版本 10.0.22621，64 位 |
| CPU | 12th Gen Intel(R) Core(TM) i7-12700H，14 物理核 / 20 逻辑处理器 |
| 内存 | 系统可见 16495096 KiB（约 15.73 GiB） |
| `GOMAXPROCS` | 环境变量未设置；Go runtime 默认，基准名后缀 `-20` |
| 包 | `han/test/performance/article` |
| 工作树 | 包含其他角色的既有改动；本次只修改基准、本文档与证据报告 |

## 6. 不可比因素

- 无 PostgreSQL：没有 SQL 解析、索引扫描、行锁、事务提交、网络往返与磁盘 I/O。
- 无 HTTP：没有请求解码、`262144` wire bytes 限制、响应序列化与中间件。
- 无 SMTP 与通知 lease：dispatcher 只执行无待处理通知的空轮，不执行真实投递、lease 或重试。
- 无 Unix Journal：Windows 上不执行 file+directory `fsync`；finalizer 只测 completed 短路。
- 内存仓库与状态表比真实持久化更快，候选批返回量也不等于数据库实际扫描量。
- 隐藏密度与文章规模是合成数据，不代表真实内容分布、热点或长尾。
- 单次运行、共享开发机、无并发压力与连接池竞争；未收集 P95、错误率或积压曲线。
- `MB/s` 只按标题与正文字节计算，不包含协议、存储或编码开销。

## 7. A-PERF-01～05 状态

| ID | 本次状态 | 本次覆盖 | 未覆盖 / 结论边界 |
|---|---|---|---|
| A-PERF-01 | partial | 200/20000 rune 的 ASCII 与 3 字节 UTF-8 创建/修改进程内吞吐、延迟与分配 | 未覆盖 `262144` wire bytes 的 HTTP 协议路径；无批准容量或时延目标，不作达标判断 |
| A-PERF-02 | planned | 仅有测试专用 1000 篇、limit 20/50、隐藏密度 0/50/90/100% 的候选扫描与投影观测 | AM-Q2 已于 2026-11-02 关闭（design r30），现行口径见 `.add/project/knowledge/design-workspace/article-management/test-plan.md:883` = `planned`；分页 20/100/200、首期规模 100000/10000 与 P95 首期验收基线（开发机测量参考，非目标环境结论）已定；目标环境真实 P95/并发/数据分布、告警阈值、retention、RPO/RTO 仍 TBD（AM-Q6/AM-Q8），目标环境测量 NOT RUN。S2.4 续接以 r04 及 2026-11-04 test-plan-addendum 为依据，实际执行/未验边界见 `.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/handoff-r01.md`；本轮 large/canonical 明确 NOT RUN，开发机 PG 观测不构成目标环境结论；不得把本表当作分页容量结论 |
| A-PERF-03 | blocked（容量目标 / RPO-RTO / retention 未确认） | 仅有 dispatcher 进程内空轮调度开销（1 round/op、2 transactions/op）与 worker 对 100 个无操作操作的调度开销 | 真实 SMTP 投递、lease、重试积压、并发/超时/`max_attempts` 仍未覆盖且参数 TBD；不构成 A-PERF-03 通过 |
| A-PERF-04 | partial | worker 批次调度、completed finalizer no-op、replay no-op 协调开销 | `deletion_pending` 年龄、Journal `fsync`、真实 finalizer poll/batch/lease/budget/backoff、吞吐与积压未覆盖；容量、P95 与告警阈值 TBD |
| A-PERF-05 | blocked（容量目标 / RPO-RTO / retention 未确认） | 仅有 1000 个 completed tombstone 的 replay 协调循环观测 | tombstone inventory、真实 ReplayAll、evidence 重算、备份规模、retention、RPO/RTO 未确认；不得外推为 RTO 结论 |

## 8. 结论边界

- 本文件提供的是可复跑的基线数值，不是验收判定。
- A-PERF-02 为 planned（目标环境 NOT RUN）；A-PERF-03、A-PERF-05 仍为 blocked；A-PERF-03 只新增了 dispatcher 空轮进程内调度观测，SMTP 投递、lease、重试积压、并发/超时和 `max_attempts` 仍未覆盖，阻塞注记必须保留到对应决策与工作负载获批。
- A-PERF-01、A-PERF-04 仍为 partial；本次未补齐协议层、PG、Journal、SMTP 或 RTO 证据。
- 未勾选任务卡、未声称独立评审通过、未声称 S2.3 关闭。
