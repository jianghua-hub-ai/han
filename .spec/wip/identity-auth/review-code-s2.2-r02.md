# identity-auth-S2.2 代码评审报告（r02 复审）

- 评审角色：Reviewer（ADD 独立评审；非作者、非 Lead）
- objectType：复合对象——`code`（benchmark 测试）、`docs`（基线文档）、`evidence`（实施证据）、`adoption`（采用记录）；分别按代码/文档口径评审
- 固定工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审日期：2026-11-03（Asia/Shanghai）
- 评审轮次：r02（对 r01 `CHANGES REQUIRED` 的返工复审）
- 上游合同：
  - 任务卡 `.add/project/knowledge/tasks/tasks.md:326-343`（当前仍为 `[ ]`，`:337` 仍写 `blocked-for-execution`）
  - 设计包络 `.add/project/knowledge/design-workspace/identity-auth/design.md:1225`（`HAN_ARGON2_MEMORY_KIB` 19456..262144、`MemoryKiB*MaxConcurrency<=524288`）、`:1231`（并发 1..4）
  - 授权口径：ADR-0028（`.add/project/knowledge/decisions/framework.md:816-854`）项目级验证授权；用户（项目 owner）2026-11-03 明确授权「目标部署环境与容量预算采用建议值」「其他需要我输入的按照建议值」，并确认 `(iv)` 独立故障域不需要、两处秘密位置可同机、本项目为测试项目
- 写权限：仅新增本报告文件 `.spec/wip/identity-auth/review-code-s2.2-r02.md`；其余文件全部只读，未修改任何既有文件、未执行任何 Git 写操作

## 1. 评审对象与实测哈希

四份对象均由本 Reviewer 在本轮重新计算字节数与 SHA256，与 Lead 给定值一致：

| # | 文件 | 字节 | SHA256 | mtime |
|---:|---|---:|---|---|
| 1 | `test/performance/identity/benchmark_test.go` | 35418 | `CB59FE4CB420BE9A9A8C0F8C8D85ED7260DA0031F5CD3A33E0E93FA3605C9293` | 2026-11-03 16:19:19 |
| 2 | `docs/benchmarks/identity.md` | 21718 | `71E5A103049E282184D14F4447BE1135C8DB218419025B2564D2E62D24787692` | 2026-11-03 16:21:28 |
| 3 | `.spec/wip/identity-auth/evidence-s2.2/report.md` | 15044 | `A309E00AACE0B29A424FD40B73A1EF8B0D53707D44A11A50D33972B40924F068` | 2026-11-03 16:22:08 |
| 4 | `.spec/wip/identity-auth/evidence-s2.2/profile-adoption-r01.md` | 5373 | `70DCB73DBA5350BB8A8223FEC4E0F016F138A3C68D95145EE2502BB43E996088` | 2026-11-03 16:14:23 |

关联基线：

- r01 报告 `.spec/wip/identity-auth/review-code-s2.2-r01.md`：18492 B / `ABCA25D5C12310BC3F4DB36C5044BBF7DA4ED2A219645B637B2527CCCFC279FA`（仅读取，未修改）。
- `go.mod`：400 B / mtime 2026-10-21 18:25:48；`go.sum`：2031 B / mtime 2026-10-21 17:59:38。均早于本任务返工，未因 S2.2 改动。

## 2. G0 输入与检查范围

已读取：

- `.codex/skills/add-code-review/SKILL.md`（完整 120 行，两轮合同与 finding 格式）
- `.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`
- 四份评审对象全文（逐行）
- 上游任务卡 S2.2 整段、设计 §3.3.1 包络、ADR-0028 正文

只读核对的生产代码与消费链：

- `internal/identity/password_reset_worker.go`（`Reconcile`、`Claim`、`Renew`、`Fail`、`reconcileCandidate`、SQL 常量）
- `internal/identity/password_reset_operator.go`（`settleUnacceptedPasswordReset`、idempotency 与 complete job 语句）
- `internal/identity/password_reset_store.go`（`CheckLockedUser`、`passwordResetGateSQL`）
- `internal/session/store.go`（`ResolvePrincipal` 的 pre-read / lock / Gate / session read 四段）
- `internal/security/argon2.go`、`internal/mailer`（仅核对生产入口）

检查维度：范围与写集、行为与失败分支、并发与竞态、指标口径与文档对应、三类结论诚实性、采用记录一致性、依赖与工作树状态。

明确不在本轮范围：真实 PostgreSQL/SMTP/外网、目标 4 vCPU / 512 MiB cgroup、OS RSS、HTTP wire、生产消费者接线、容量预算达标判定。

## 3. 实际执行命令与结果

工作目录：仓库根目录。全部命令均为本轮真实运行。

| # | 命令 | exit code | 实际结果 |
|---:|---|---:|---|
| 1 | `gofmt -l test/performance/identity` | 0 | 无输出 |
| 2 | `go vet ./test/performance/identity` | 0 | 无输出 |
| 3 | `go build ./...` | 0 | 无输出 |
| 4 | `go test -run "^$" -bench . -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 36.863s` |
| 5 | `go test -run "^$" -race -bench "BenchmarkArgon2idHash/concurrency=4$" -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 5.160s`；28923436 ns/op，failure-rate 0，`memory*concurrency-KiB` 262144 |
| 6 | `go test -run "^$" -race -bench "BenchmarkPasswordResetWorkerClaimBatch100$" -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 2.765s`；3495211 ns/op；`claimed/op`、`renews/op`、`renewed/op`、`settled/op` 均 100，`failed/op` 0，`settlement-execs/op` 400 |
| 7 | `go test -run "^$" -race -bench "BenchmarkPasswordResetWorkerSettlementRetry100" -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 4.329s`；两个子基准 `failed/op` 均 100、`settled/op` 均 0、`rollbacks/op` 均 100 |

完整基准关键观测（本轮独立运行，不是作者那一轮）：

| 基准 | ns/op | p50/p95/p99 (us) | 关键计数 | failure-rate |
|---|---:|---|---|---:|
| `Argon2idHash/concurrency=4` | 24295205 | 88950 / 133096 / 147114 | `memory*concurrency-KiB` 262144，`peak-sampled-heap-KiB` 525252 | 0 |
| `Argon2idVerify/concurrency=4` | 18294966 | 70976 / 83497 / 88204 | `peak-sampled-heap-KiB` 525307 | 0 |
| `SessionResolvePrincipal` | 2534 | 0 / 0 / 0 | `4.000 queries/op`、`1.000 transactions/op` | 0 |
| `GovernanceRevokeAdministrator` | 678.3 | 0 / 0 / 0 | 2 locks、1 tx、1 update、1 revoke、1 audit | 0 |
| `PasswordResetGateCheck` | 255.3 | 0 / 0 / 0 | `1.000 queries/op` | 0 |
| `PasswordResetWorkerEmptyScan` | 1843 | 0 / 0 / 0 | candidates/claimed/settled/tx 均 0 | 0 |
| `PasswordResetWorkerScanCandidates100` | 97307 | 0 / 999.3 / 1261 | candidates 100、claim-attempts 100、claimed 0、tx 100、locks 100 | 0 |
| `PasswordResetWorkerClaimBatch100` | 556346 | 0 / 1249 / 2069 | candidates/claimed/renewed/settled 均 100、settlement-execs 400、tx 300、locks 400、failed 0 | 0 |
| `SettlementRetry100/retry_wait` | 456075 | 0 / 1159 / 1626 | failed 100、settled 0、rollbacks 100、tx 400、locks 500 | 0 |
| `SettlementRetry100/dead_letter` | 453783 | 0 / 1085 / 1593 | failed 100、settled 0、rollbacks 100、tx 400、locks 500 | 0 |
| `SMTPImplicitTLSInMemory/concurrency=4` | 233561 | 1000 / 2000 / 2194 | connections/op 1、peak-connections 4 | 0 |

与文档记录的关系：

- `identity.md:124` 与 `report.md:71` 记录作者权威运行为 `ok ... 35.189s`；本轮独立运行 `36.863s`，属基准耗时正常抖动，不构成矛盾。
- 文档表格数值与 `report.md` §6/§7 相互一致（逐个抽查 c4、Session、Gate、worker 与 SMTP 行均一致）。
- 本轮 `SessionResolvePrincipal` 2534 ns/op 与文档 4802 ns/op 差异较大，属亚毫秒基准在此 Windows 主机上的跨运行波动；文档已声明单次运行、无置信区间，见 `IA-S2.2-R02-F004`（INFO）。

## 4. r01 findings 逐项处置

| r01 ID | 原级别 | 处置 | 依据 |
|---|---|---|---|
| `IA-S2.2-R01-F001` | HIGH | **关闭** | `benchmark_test.go:1001-1030`、`:976-999`、`:1032-1075`；生产路径 `password_reset_worker.go:412-448`、`:91-116`；运行输出 100/100/100、settlement-execs 400、failed 0 |
| `IA-S2.2-R01-F002` | MEDIUM | **关闭** | `benchmark_test.go:516-550`；生产 Gate `password_reset_store.go:59-75`；`session/store.go:305`；输出 `4.000 queries/op`、`1.000 transactions/op` |
| `IA-S2.2-R01-F003` | LOW | **关闭** | `benchmark_test.go:686-690`、`:714`、`:947-962`；同一轮 locks/op 随场景为 0/100/400/500 |
| `IA-S2.2-R01-F004` | INFO | **关闭** | `identity.md:241-263` 指标图例覆盖全部 `b.ReportMetric` 名；`peak-sampled-heap-KiB` 在 `:114`、`:256`、`:299` 声明为 Go 堆采样而非 OS RSS |

### 4.1 F001（关闭）

- **ClaimBatch100 真实成功路径**：fake 池扫描返回 100 个候选（`benchmark_test.go:891-897`、`:868-874`）；claim 语句 `SET status='leased'` 命中时返回 `attempt_count=1`（`:803-814`），生产 `Claim` 将其转为 lease（`password_reset_worker.go:316-336`）；随后生产 `Renew`（`:338-353`，SQL 含 `lease_expires_at=$1,updated_at=$2`，fake 于 `:827-830` 计数）与 settlement（`password_reset_operator.go:539-570`，fake 于 `:840-859` 覆盖 invalidate credential / invalidated operation / idempotency / complete job）全部可达。
- **计数派生并断言**：`claimed == renewed == settled == 100*b.N`、`failed == 0`、`unexpected == 0`、`result.failures == 0`（`benchmark_test.go:1010-1028`）。本轮输出与断言一致。
- **扫描型分离**：`ScanCandidates100` 的 claim 返回 `pgx.ErrNoRows`（`:805-807`），断言 `claimed == 0`、`settled == 0`（`:989-994`），不再用 ClaimBatch 名称暗示成功批量。
- **Fail 分支**：`SettlementRetry100` 让 settlement 首个 `Exec` 返回合成错误（`:840-843`），驱动生产 `Fail`（`password_reset_worker.go:127-137`、`:355-376`）；fake 记录写入的状态参数并断言 `retry_wait` / `dead_letter`（`:1038-1039`、`:1063-1066`），同时断言 `failed == 100*b.N`、`settled == 0`、`rollbacks == 100*b.N`（`:1057-1062`）。`MaxRetries=3` 与 `1` 分别对应 retry_wait 与 dead_letter 分支。
- **文档一致性**：`identity.md:109-112` 与 `report.md:109,117-119` 的描述与真实覆盖一致；未发现“ClaimBatch 仅扫描”的残留过度声明。真实 SQL 结果、行锁语义、索引命中仍如实列入 NOT RUN，不作为本 finding 的未关闭项。
- **处置**：关闭。残余边界（真实 PG 语义）已登记，不属于原 finding 的“分支不可达”问题。

### 4.2 F002（关闭）

- Session resolve 使用生产 `identity.PasswordResetStore{}` 作为 Gate（`benchmark_test.go:522-525`），`session/store.go:305` 在事务内经 `CheckLockedUser` 执行 `passwordResetGateSQL`（`password_reset_store.go:59-75`）。
- fake 事务按 SQL 形状分派并计数：pre-read（`:393-402`）、用户锁（`:434-438`）、Gate（`:429-433`）、session 读取（`:439-458`）。
- 断言 `gateQueries == b.N`（`:540-543`），总查询数 `== 4*b.N`（`:544-547`），`queries/op`、`transactions/op` 由计数派生（`:548-549`）。本轮输出 `4.000 queries/op`、`1.000 transactions/op`。
- **处置**：关闭。Gate 每 op 恰好一次且有硬断言，硬编码已消除。

### 4.3 F003（关闭）

- 治理：`locks/op`、`updates/op`、`session-revokes/op`、`audit-events/op` 由 fake repo/audit 调用计数派生（`:686-690`）；Gate：`queries/op` 由 fake `QueryRow` 计数派生（`:714`）；worker：12 项指标由 `QueryRow`/`Exec`/扫描结果派生（`:947-962`）。
- 横向佐证：同一轮运行中 `locks/op` 为 0（EmptyScan）、100（ScanCandidates100）、400（ClaimBatch100）、500（SettlementRetry100），说明不是常量。
- **处置**：关闭。未发现残留硬编码常量。

### 4.4 F004（关闭）

- 逐项枚举代码中全部 `b.ReportMetric` 名并与 `identity.md:241-263` 对照，无缺失：`p50-us`、`p95-us`、`p99-us`、`failure-rate`、`heap-samples`、`workers`、`memory-KiB`、`limit-KiB`、`iterations`、`parallelism`、`memory*concurrency-KiB`、`peak-sampled-heap-KiB`、`steady-heap-KiB`、`statements/op`、`queries/op`、`transactions/op`、`locks/op`、`updates/op`、`session-revokes/op`、`audit-events/op`、`scan-queries/op`、`candidates/op`、`claim-attempts/op`、`claimed/op`、`renews/op`、`renewed/op`、`settled/op`、`failed/op`、`settlement-execs/op`、`rollbacks/op`、`connections/op`、`peak-connections`。
- `peak-sampled-heap-KiB` 的 Go 堆采样语义已在正文与图例明确，且明确“不是 OS RSS”。
- **处置**：关闭。图例完整；`peak-connections` 的语义精度问题另见 `IA-S2.2-R02-F003`（INFO）。

## 5. 新增 findings（r02）

### [MEDIUM] `unresolved age` 关闭条件仍为 NOT RUN，未见测点或豁免记录

- finding ID：`IA-S2.2-R02-F002`
- 对象/修订：`benchmark_test.go` 35418 B / `CB59FE4CB420BE9A9A8C0F8C8D85ED7260DA0031F5CD3A33E0E93FA3605C9293`；`identity.md` 21718 B / `71E5A103049E282184D14F4447BE1135C8DB218419025B2564D2E62D24787692`；`report.md` 15044 B / `A309E00AACE0B29A424FD40B73A1EF8B0D53707D44A11A50D33972B40924F068`
- 位置：`.add/project/knowledge/tasks/tasks.md:341`；`docs/benchmarks/identity.md:25`、`:288`；`.spec/wip/identity-auth/evidence-s2.2/report.md:154`
- 现状：任务卡要求关闭条件纳入 `unresolved age`；三份交付物均把它登记为“不覆盖/NOT RUN”；`benchmark_test.go` 没有 unresolved age 基准或指标。生产 `passwordResetGateSQL`（`internal/identity/password_reset_store.go:1513-1608`）只做状态判定，未提供 age 查询或 age 指标。
- 触发：Lead 按 `tasks.md:341` 逐项核对关闭条件时。
- 影响：任务卡明确关闭条件存在测试盲区。由于文档已如实登记 NOT RUN，这不构成伪造 PASS；但若无豁免记录即关闭，等于静默跳过一项要求。
- 建议：二选一：(a) 增加基于模拟行/派生指标的 unresolved age 基准，显式标注模拟，不伪造生产 SQL；(b) 由 owner/Lead 以书面测试范围豁免（`waived-for-test-scope`）显式处置该子项，并在关闭记录登记未验证边界与授权来源。
- 验证：补测则给出基准输出与文档对应行；豁免则给出授权来源、范围与登记位置。

### [LOW] 采用记录 §2.3 仍引用返工前 r00 观测值且未标注历史

- finding ID：`IA-S2.2-R02-F001`
- 对象/修订：`profile-adoption-r01.md` 5373 B / `70DCB73DBA5350BB8A8223FEC4E0F016F138A3C68D95145EE2502BB43E996088`
- 位置：`.spec/wip/identity-auth/evidence-s2.2/profile-adoption-r01.md:49-52`、`:58-59`；对照 `docs/benchmarks/identity.md:170`、`:179`、`:206`、`:231`；`report.md:149` 已披露该问题
- 现状：采用记录写 Argon2 c4 hash 164.8 ms、verify 157.2 ms、p99 177.4/180.8 ms、worker batch p95 984 us、SMTP c4 p95 1.67 ms；当前权威运行对应为 151902 us、169734 us、158292/171611 us、1643 us、2062 us。`report.md:149` 已说明“采用记录 §2.3 引用返工前 r00 数值，本任务无权修改”，但采用记录本身与 `identity.md` §7 未标注“r00 历史观测”。
- 触发：读者按 §7 的“唯一链”核对采用记录与基线文档数值时。
- 影响：文档一致性与可追溯性下降；不改变预算结论（当前观测仍低于采用预算），也不是虚假 PASS。
- 建议：由 Lead 在采用记录 §2.3 标注“r00 历史观测，不代表当前权威运行”，或同步为当前值；`identity.md` §7 可加一句指向该标注。
- 验证：两处数值/标注一致，唯一链引用可对齐。

### [INFO] `peak-connections` 可瞬时超过 `workers`，不应读作并发上限

- finding ID：`IA-S2.2-R02-F003`
- 对象/修订：`benchmark_test.go` / `CB59FE4C...`；`identity.md` / `71E5A103...`
- 位置：`benchmark_test.go:1137-1149`、`:1266-1268`；`identity.md:263`
- 现状：dialer 在建立连接时递增 `current`，在 server goroutine 退出时递减（`:1146`）；client `Send` 返回后 server 可能尚未退出。本轮独立运行 c2 观测到 `peak-connections=3` 而 `workers=2`。
- 触发：连续 op 时上一连接 server 侧尚未关闭。
- 影响：指标仍是真实连接峰值，但语义是“dialer/server 会话峰值”，不是客户端并发上限；若被读作并发上限会误判。
- 建议：图例补一句“峰值可在相邻 op 间短暂高于 `workers`”；无需改代码。
- 验证：图例更新后与实测 c2=3 不矛盾。

### [INFO] 亚毫秒基准跨运行波动较大

- finding ID：`IA-S2.2-R02-F004`
- 对象/修订：`identity.md` / `71E5A103...`
- 位置：`identity.md:130-132`、`:189`、`:195`、`:203`
- 现状：本轮 `SessionResolvePrincipal` 2534 ns/op（文档 4802）、`Governance` 678.3（文档 841.5）、`GateCheck` 255.3（文档 334.2）。
- 触发：同一主机不同时点运行。
- 影响：表格数值不应被当作稳定回归判据；文档已声明单次运行、无置信区间，且亚毫秒优先看 `ns/op`。
- 建议：无需改动；若后续用作回归门禁，应先固定采样轮次或置信区间。
- 验证：不适用。

### [INFO] `identity.md:299` 对 c3 峰值措辞不精确

- finding ID：`IA-S2.2-R02-F005`
- 对象/修订：`identity.md` / `71E5A103...`
- 位置：`identity.md:299`
- 现状：写作“c3/c4 的采样峰值接近或略高于 `524288 KiB`”；实际 c3 为 394146 KiB（约上限 75%），c4 为 525275 KiB（略高）。
- 触发：读者据此估计 c3 余量时。
- 影响：仅为措辞精度，不影响“不能读作预算通过/失败”的结论。
- 建议：改为“c4 略高于上限，c3 约为上限的 75%”。
- 验证：文字与 §5.1 表格一致。

## 6. 关闭条件逐项覆盖表

对应 `tasks.md:341` 与用户本轮列出的核查项：

| 关闭条件 | 覆盖状态 | 证据 | 边界 |
|---|---|---|---|
| Argon2id 基线（候选 tuple、c1..c4、峰值/稳态内存、p50/p95/p99） | **已测** | `benchmark_test.go:212-287`；`identity.md` §3、§5.1-5.3；本轮 c1..c4 全部出值 | 本机进程内与 `GOMAXPROCS=4` 模拟；目标 cgroup 未测 |
| Session（Create/Revoke/Resolve） | **已测** | `benchmark_test.go:464-550`；`identity.md` §5.4 | 进程内 fake；真实 PG 未测 |
| 治理事务 | **已测** | `benchmark_test.go:639-691`；`identity.md` §5.5 | 进程内 fake |
| SMTP 基线 | **已测** | `benchmark_test.go:1216-1269`；`identity.md` §5.7 | `net.Pipe` 进程内 TLS；真实 SMTP 未测 |
| 恢复批量 | **已测** | `BenchmarkPasswordResetWorkerClaimBatch100`（100 候选/100 claimed/100 settled） | fake 池；真实表/索引未测 |
| password-reset worker lease | **部分** | `password_reset_worker.go:316-336`；ClaimBatch100 覆盖单 lease | 多 worker lease 竞争未测（NOT RUN） |
| worker retry | **已测** | `SettlementRetry100/{retry_wait,dead_letter}`，断言状态写入 | 合成错误；真实 backoff 时序未测 |
| worker batch | **已测** | ClaimBatch100 / ScanCandidates100 / SettlementRetry100 | batch=100 固定 |
| `unresolved age` | **未测** | `identity.md:25`、`:288`；`report.md:154`（NOT RUN） | 无基准、无指标；见 `IA-S2.2-R02-F002` |
| Gate 查询 | **已测** | `benchmark_test.go:705-715`；生产 `password_reset_store.go:59-75`；`identity.md` §5.6 | 单条 Gate SQL 形状 |
| 事务/index | **部分** | 事务数由 fake `BeginTx` 派生（ClaimBatch 300、Retry 400） | 真实 PG 事务语义、索引选择与命中未测 |
| SMTP 并发资源测量 | **已测（进程内连接）** | `workers`、`peak-connections`；`identity.md` §5.7 | OS 资源/RSS 未测 |
| 唯一 `profile ID → benchmark 报告 → 采用记录` | **已建立** | `identity.md` §7；采用记录 §4；ID `identity-argon2id-local-20261103-r01` | 状态 `adopted / not verified` |

结论：除 `unresolved age` 外，任务卡列出的测试项均已实际执行或有明确的部分覆盖边界；未覆盖项均在 NOT RUN 中如实登记，未发现伪造通过。

## 7. 三类结论与采用口径一致性

| 结论 | `identity.md` | `report.md` | 采用记录 | 评审判断 |
|---|---|---|---|---|
| ① 合法 token/tuple | `PASS（仅机械校验）`（`:269`） | 同（`:137`） | 同（`:68`） | 一致；仅机械校验，未升级语义 |
| ② 目标环境可运行 | `NOT ESTABLISHED`（`:270`） | 同（`:138`） | 同（`:69`） | 一致；真实 4 vCPU / 512 MiB cgroup 未运行 |
| ③ 满足批准预算 / 生产 SLO | `ADOPTED / NOT VERIFIED`（`:271`） | 同（`:139`） | 同（`:70`） | 一致；“采用”未被写成“达标” |

逐项核对：

- `identity.md:273`、`:282`、`report.md:148`、采用记录 §4/§5 均明确“采用 ≠ 目标环境实测”“不提供达到预算结论”。
- 全文检索“达标/满足预算/生产就绪/预算通过”等词，未见把目标环境或预算写成 PASS 的表述；出现处均为否定或边界说明（`identity.md:9`、`:271`、`:273`、`:282`；`report.md:103`、`:139`、`:148`、`:163`；采用记录 `:70`）。
- `identity.md` §7 与采用记录的唯一链一致：同一 profile ID、同一报告路径、同一采用记录文件、同一 `adopted / not verified` 状态。
- 采用记录 §2.3 的历史观测值问题见 `IA-S2.2-R02-F001`（LOW），不影响三类结论口径。

## 8. 写集合规与依赖

- 本任务返工对象的 mtime 集中在 2026-11-03 16:14–16:22：采用记录 16:14:23、`benchmark_test.go` 16:19:19、`identity.md` 16:21:28、`report.md` 16:22:08；r01 报告 16:13:23。
- `tasks.md` mtime 2026-11-03 15:41:09，早于返工对象；`design.md` mtime 2026-11-02 19:05:51；`internal/identity/password_reset_worker.go` 2026-10-30 15:20:04、`password_reset_store.go` 2026-10-30 11:52:34、`internal/session/store.go` 2026-10-30 06:48:14、`internal/security/argon2.go` 2026-10-23 17:19:39。未发现本任务对生产代码、设计/ADR、既有测试或 `tasks.md` 的写入证据。
- `git status --short` 显示 `tasks.md` 相对 HEAD 有修改、`go.mod` 有修改、`go.sum` 未跟踪；这些状态在任务开始前已存在（mtime 分别为 15:41 与 2026-10-21），其 diff 内容与本任务无关（`go.mod` diff 仅为已有的 pgx / x/crypto 依赖块）。不能把工作树既有改动归因于本轮作者。
- `go.mod` 与 `go.sum` 的 mtime 均为 2026-10-21，早于本任务；`benchmark_test.go` 的导入仅为既有 `pgx`、`pgx/pgconn` 与 `han/internal/*`。**未新增依赖。**
- 本 Reviewer 未执行任何 Git 写操作，未修改四份对象、`tasks.md`、生产代码或采用记录，仅新增本报告。

## 9. 结论与剩余条件

**结论：`PASS WITH CONDITIONS`。**

- r01 `F001`–`F004` 全部关闭；未发现 BLOCKER 或 HIGH。
- 新增 findings：1 MEDIUM（`IA-S2.2-R02-F002`）、1 LOW（`IA-S2.2-R02-F001`）、3 INFO（`F003`–`F005`）。
- 所有实际执行命令 exit 0、`PASS`；完整基准 `ok han/test/performance/identity 36.863s`；`-race` 三项全绿。

**本报告是否足以支持 Lead 按测试范围关闭本卡：**

- **不足以无条件关闭**。`tasks.md:341` 明确要求纳入 `unresolved age`，而本对象未测，故不能声称该关闭条件已覆盖。
- 在该唯一剩余条件被显式处置并登记后，本报告足以支持 Lead **按测试范围**关闭本卡（其余条件已覆盖或有如实登记的部分覆盖边界）。

**唯一剩余条件：**

- 由 owner/Lead 对 `tasks.md:341` 的 `unresolved age` 子项作出显式处置：补测（基于模拟行/派生指标，标注模拟）或书面测试范围豁免（`waived-for-test-scope`），并在关闭记录登记未验证边界与授权来源。

**非阻断同步事项：**

- `IA-S2.2-R02-F001`（采用记录 §2.3 旧值标注）建议由 Lead 同步，不阻断关闭。
- `F003`–`F005` 为 INFO，无需作者返工。

**明确不声称：**

- 不声称生产环境 readiness、目标环境可运行、容量预算达标、生产 SLO 达标或生产 Gate 已解锁。
- 不声称本卡已关闭；`tasks.md` 未修改，仍为 `[ ]` 与 `blocked-for-execution`。
- 不声称真实 PostgreSQL、真实 SMTP、OS RSS、HTTP wire 或目标 cgroup 已验证。

## 10. 代理评审声明

**AI 代理评审，非真人签字。**

- 授权来源：ADR-0028《测试项目采用分级验证、模拟环境替代与 AI 代理评审签收》（`.add/project/knowledge/decisions/framework.md:816-854`），以及项目 owner（用户）2026-11-03 的明确授权（目标部署环境与容量预算采用建议值、其他需输入项按建议值）。
- 代理主体：本 Reviewer（AI/Agent）。
- 代理范围：仅限本测试项目的代码与证据评审；不构成生产采用、SLA、容量承诺、灾备就绪或人的成果签收。
- 本报告不覆盖 ADR-0028 排除项（伪造 PASS、泄露凭据、未经确认的删除/覆盖、Git push）。

## 附：评审限制

- 未运行真实 PostgreSQL、真实 SMTP/TLS 认证、目标 4 vCPU / 512 MiB cgroup、OS RSS、HTTP wire 或端到端路径；相关结论一律 NOT RUN/unknown。
- 未核对工作树中他人并发改动；写集判断基于 mtime 与本任务声明路径，不对工作树整体归属作判断。
- 基准为单次本机运行，跨运行波动已记录；不构成生产容量结论。
