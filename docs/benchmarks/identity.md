# Identity 容量与性能基线

- 任务：`identity-auth-S2.2 建立 identity 容量与性能基线`
- 固定基线：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 运行日期：2026-11-03（Asia/Shanghai）
- 本文证据：`.spec/wip/identity-auth/evidence-s2.2/report.md`
- 修订：review r01 返工（F001–F004），指标口径与 worker 覆盖范围已按评审要求调整

**本基线不含阈值判定。** 本文只记录本次真实运行得到的观测值。未获批准的生产 SLO、容量目标、P95、压测或 RPO/RTO/retention 阈值均未使用，因此本文不对任何指标作“达标/不达标”结论。本文不表示本卡已关闭或已通过独立评审。

## 1. 范围

本基线覆盖仓库内可在进程内直接调用、且不依赖真实外部系统的 identity 路径：

- Argon2id 哈希与校验：`security.NewArgon2idProcessGate`、`security.NewGatedArgon2idHasher`。
- Session：`session.Store.Create`、`session.Store.Revoke`、`session.Store.ResolvePrincipal`（含生产 password-reset Gate 查询）。
- 治理事务：`identity.AdministratorGovernanceService.RevokeAdministrator`（锁用户、更新角色、撤销会话、审计事件）。
- Password reset Gate 查询：`identity.PasswordResetStore.CheckLockedUser`。
- Password reset worker：`identity.PasswordResetRecoveryWorker.Reconcile`，分四个基准覆盖空扫描、候选扫描（claim 未命中）、成功 claim + Renew + settlement 批量（100 个候选）、结算失败后的 retry/dead-letter 批量（100 个候选）。
- SMTP：`mailer.SMTPMailer.Send`（implicit TLS，进程内 `net.Pipe` 替身）。

明确不覆盖：

- 真实 PostgreSQL 连接、SQL planner、索引、锁等待、MVCC、WAL/`fsync`。
- 真实 password-reset lease 竞争、真实 retry/backoff 调度、unresolved age 查询与真实表数据。
- 真实 SMTP/TLS 认证、网络、DNS、超时与投递结果。
- 独立持久卷与文件系统耐久性。
- 实际 4 vCPU / 512 MiB cgroup 配额与 OS RSS 峰值。
- HTTP wire 层、请求解析、限流与连接复用。
- 任何生产 SLO、容量目标或目标环境验收。

## 2. 环境事实

### 2.1 实测主机（本次全部数字的运行环境）

| 项目 | 实测值 |
|---|---|
| OS / arch | `windows/amd64` |
| Go | `go1.27.0 windows/amd64` |
| CPU | `12th Gen Intel(R) Core(TM) i7-12700H` |
| 逻辑处理器 | `20` |
| 物理内存 | `16890978304` bytes（约 `15.73 GiB` / `16495096 KiB`） |
| `GOMAXPROCS` | 环境变量未设置；实测基准输出后缀为 `-20`（Go runtime 默认等于逻辑处理器数） |
| 既有依赖 | `github.com/jackc/pgx/v5 v5.11.0`、`golang.org/x/crypto v0.58.0`（未新增依赖） |

### 2.2 目标部署环境建议值（模拟基线口径，不是 SLO，也不是已执行环境）

| 项目 | 建议值 | 来源 |
|---|---|---|
| vCPU 配额 | `4 vCPU` | 项目 owner 于 2026-11-03 授权的建议值 |
| 进程内存上限 | `512 MiB` = `524288 KiB` | 项目 owner 于 2026-11-03 授权的建议值；与 design r26 应用硬上限一致 |
| Go | `1.27` | 项目 owner 于 2026-11-03 授权的建议值 |
| `GOMAXPROCS` | `4` | 项目 owner 于 2026-11-03 授权的建议值 |

上述建议值仅用于描述“目标环境形状”。本轮未在真实 4 vCPU / 512 MiB cgroup 中运行，因此不能据此声称目标环境 readiness 通过。第 5.3 节提供一次 `GOMAXPROCS=4` 的补充模拟运行，仍不是目标环境实测。

## 3. Argon2 target-profile readiness

### 3.1 候选 tuple

| 参数 | 值 |
|---|---:|
| `MemoryKiB` | `65536`（64 MiB） |
| `Iterations` | `3` |
| `Parallelism` | `4` |
| `SaltLength` | `16` |
| `KeyLength` | `32` |
| `MaxConcurrency` | `4`（覆盖 c1..c4） |
| 应用内存硬上限 | `524288 KiB` |

### 3.2 与设计包络的机械对账

设计包络来源：`.add/project/knowledge/design-workspace/identity-auth/design.md` §3.3.1。

| 约束 | 候选值 | 结果 |
|---|---|---|
| `HAN_ARGON2_MEMORY_KIB` ∈ `19456..262144` | `65536` | 合法 |
| `HAN_ARGON2_ITERATIONS` ∈ `2..6` | `3` | 合法 |
| `HAN_ARGON2_PARALLELISM` ∈ `1..4` | `4` | 合法 |
| `MemoryKiB >= 8 * Parallelism` | `65536 >= 32` | 合法 |
| `HAN_ARGON2_SALT_BYTES` ∈ `16..64` | `16` | 合法 |
| `HAN_ARGON2_KEY_BYTES` ∈ `32..64` | `32` | 合法 |
| `MemoryKiB * MaxConcurrency <= 524288` | 见 3.3 | c1..c4 均合法 |

### 3.3 `memory * concurrency` 与 `524288 KiB` 对账

| concurrency | `memory*concurrency` (KiB) | 硬上限 (KiB) | 余量 (KiB) | 是否在包络内 |
|---:|---:|---:|---:|---|
| 1 | 65536 | 524288 | 458752 | 是 |
| 2 | 131072 | 524288 | 393216 | 是 |
| 3 | 196608 | 524288 | 327680 | 是 |
| 4 | 262144 | 524288 | 262144 | 是 |

上述乘积由基准输出的 `memory*concurrency-KiB` 与 `limit-KiB` 逐行对账，见第 5.1、5.2 节。

结论①（合法 token/tuple）：**PASS（仅机械校验）**。该结论只说明 token 与交叉约束合法，不说明目标环境可运行，也不说明满足任何预算。

## 4. 工作负载定义（测试专用假设，不是生产配置）

- 全部持久化、事务、审计、ID、时钟与邮件传输均使用确定性 in-memory 替身；不连接真实 PostgreSQL、SMTP、网络或持久卷。
- 时钟固定为 `2026-11-03T00:00:00Z`（UTC）；ID 为顺序生成的合法 UUID 文本。
- Argon2 salt 使用确定性字节流，不使用 `crypto/rand`；密码与摘要仅存在于进程内。
- 密码与 token 均为固定测试字面量（`benchmark-password`、`session-token`、`csrf-token`），不是凭据。
- Session 创建/撤销各计 1 条语句。
- `Session.ResolvePrincipal` 走生产调用链：pre-read（1）+ 事务内用户锁（1）+ 生产 `identity.PasswordResetStore.CheckLockedUser` Gate 查询（1）+ session 读取（1）= **4 查询/op**。`queries/op` 与 `transactions/op` 由 fake 的实际调用计数派生；基准内断言 Gate 恰好每次 op 调用一次、总查询数等于 `4 * b.N`，不满足即 `b.Fatalf`。
- 治理事务固定 1 次事务、2 次加锁、1 次角色更新、1 次会话撤销、1 条审计事件；所有计数由 fake 的实际调用派生。
- Password reset worker：`LeaseDuration=1s`、`RetryBackoff=1s`、`MaxRetries=3`（dead-letter 场景用 1）、`ClaimBatchLimit=100`、`ConnectionBudget=4`；均为测试专用值。四个基准：
  - `BenchmarkPasswordResetWorkerEmptyScan`：扫描返回 0 个候选。
  - `BenchmarkPasswordResetWorkerScanCandidates100`：扫描返回 100 个候选，claim 语句返回 `pgx.ErrNoRows`（claim 未命中），只覆盖扫描与未成功领取的 claim 尝试。
  - `BenchmarkPasswordResetWorkerClaimBatch100`：扫描返回 100 个候选，claim 返回 `attempt_count=1` 的成功行，随后 Renew（`UPDATE ... SET lease_expires_at=$1,updated_at=$2`）与 settlement（invalidate credential / invalidated operation / idempotency / complete job）的 `Exec` 分支全部可达，真实驱动生产 `Reconcile -> Claim -> Renew -> settlePasswordResetRecovery` 成功路径。
  - `BenchmarkPasswordResetWorkerSettlementRetry100`：claim 与 Renew 成功，settlement 首个 `Exec` 返回合成错误，驱动生产 `Fail` 分支；`retry_wait`（`MaxRetries=3`）与 `dead_letter`（`MaxRetries=1`）两个子基准。
- worker 指标 `claimed/op`、`renews/op`、`renewed/op`、`settled/op`、`failed/op`、`claim-attempts/op`、`locks/op`、`candidates/op`、`scan-queries/op`、`transactions/op`、`settlement-execs/op`、`rollbacks/op` 全部由 fake 的 `QueryRow`/`Exec`/扫描结果计数派生，不使用常量。基准内断言：ClaimBatch100 的 `claimed`、`renewed`、`settled` 均等于 `100 * b.N` 且 `failed = 0`；ScanCandidates100 的 `claimed`、`settled` 均为 0；SettlementRetry100 的 `failed` 等于 `100 * b.N` 且 `settled = 0`，并校验实际写入的 job 状态为 `retry_wait` / `dead_letter`。
- SMTP 使用进程内 `net.Pipe` + 运行时生成的临时自签名证书（仅进程内测试 fixture，不落盘、不外连、不是凭据）；每个 op 建立 1 条连接。
- `peak-sampled-heap-KiB` 来自 `runtime/metrics` 的 `/memory/classes/heap/objects:bytes`，是 Go 堆对象字节采样，**不是 OS RSS**；`steady-heap-KiB` 为 `runtime.GC()` 后的同类采样。

## 5. 实测结果

权威运行命令（验收命令）：

```text
go test -run "^$" -bench . -benchmem ./test/performance/identity
```

本次 exit code `0`，输出 `PASS`，包结果为 `ok han/test/performance/identity 35.189s`（进程内总耗时约 36.8 秒）。

统计口径：

- `ns/op`、`B/op`、`allocs/op` 来自 Go `testing` 基准。
- `p50-us`、`p95-us`、`p99-us` 为每个 op 的进程内 `time.Since` 分位数，单位微秒。
- 本机对亚毫秒操作的分位数存在明显的采样量化（大量 `0` 与约 `1 ms` 的台阶），因此亚毫秒操作的 `ns/op` 更可信，分位数仅供参考。
- `failure-rate` 为失败 op 数 / 总 op 数。
- 单次运行，未计算置信区间，不代表生产分布。

### 5.1 Argon2id 哈希（c1..c4）

| concurrency | ns/op | p50 (us) | p95 (us) | p99 (us) | peak-sampled-heap (KiB) | steady-heap (KiB) | memory*conc (KiB) | failure-rate |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 38582383 | 38192 | 47037 | 48700 | 197251 | 700 | 65536 | 0 |
| 2 | 26129111 | 51535 | 59580 | 62329 | 263001 | 857 | 131072 | 0 |
| 3 | 19291217 | 57139 | 61748 | 62357 | 394146 | 930 | 196608 | 0 |
| 4 | 16855507 | 65982 | 77322 | 81797 | 525275 | 985 | 262144 | 0 |

### 5.2 Argon2id 校验（c1..c4）

| concurrency | ns/op | p50 (us) | p95 (us) | p99 (us) | peak-sampled-heap (KiB) | steady-heap (KiB) | memory*conc (KiB) | failure-rate |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 39072675 | 38933 | 43589 | 44651 | 132064 | 989 | 65536 | 0 |
| 2 | 27357336 | 54421 | 60793 | 62053 | 263159 | 1014 | 131072 | 0 |
| 3 | 21083348 | 61286 | 72162 | 73668 | 394241 | 1024 | 196608 | 0 |
| 4 | 19789876 | 78149 | 89517 | 94195 | 525324 | 1031 | 262144 | 0 |

### 5.3 补充模拟：`GOMAXPROCS=4`（不是验收命令，不是目标环境）

补充命令（仅用于模拟 4-P 形状）：

```text
$env:GOMAXPROCS=4
go test -run "^$" -bench "Argon2" -benchmem ./test/performance/identity
```

本次 exit code `0`，输出 `PASS`，包结果为 `ok han/test/performance/identity 10.843s`，基准后缀为 `-4`。

哈希：

| concurrency | ns/op | p50 (us) | p95 (us) | p99 (us) | peak-sampled-heap (KiB) | steady-heap (KiB) |
|---:|---:|---:|---:|---:|---:|---:|
| 1 | 36882550 | 37109 | 39975 | 40021 | 131443 | 377 |
| 2 | 35907133 | 70788 | 77426 | 86475 | 262547 | 400 |
| 3 | 34240980 | 100879 | 116233 | 121058 | 393627 | 401 |
| 4 | 34985830 | 138338 | 151902 | 158292 | 524695 | 401 |

校验：

| concurrency | ns/op | p50 (us) | p95 (us) | p99 (us) | peak-sampled-heap (KiB) | steady-heap (KiB) |
|---:|---:|---:|---:|---:|---:|---:|
| 1 | 37238513 | 37212 | 41084 | 44096 | 131479 | 401 |
| 2 | 34283981 | 67611 | 75333 | 78563 | 262549 | 405 |
| 3 | 35799190 | 105079 | 128875 | 129910 | 524696 | 406 |
| 4 | 38415581 | 153823 | 169734 | 171611 | 459162 | 407 |

该补充运行说明：在 4 个 P 上，单次 Argon2 哈希/校验的 p50 随并发近似线性增长（约 37、71、101、138 ms 与 37、68、105、154 ms），符合共享 4 个 P 的争用预期。它仍运行在 20 逻辑核、15.73 GiB 的实测主机上，**不是** 4 vCPU / 512 MiB cgroup 实测。

### 5.4 Session

| 基准 | ns/op | B/op | allocs/op | 附加指标 | failure-rate |
|---|---:|---:|---:|---|---:|
| `BenchmarkSessionCreate` | 641.5 | 480 | 12 | 1 statements/op | 0 |
| `BenchmarkSessionRevoke` | 248.2 | 144 | 5 | 1 statements/op | 0 |
| `BenchmarkSessionResolvePrincipal` | 4802 | 1297 | 31 | 4 queries/op、1 transactions/op（均由计数派生；Gate 调用数已断言等于 op 数） | 0 |

### 5.5 治理事务

| 基准 | ns/op | B/op | allocs/op | 附加指标 | failure-rate |
|---|---:|---:|---:|---|---:|
| `BenchmarkGovernanceRevokeAdministrator` | 841.5 | 552 | 12 | 2 locks/op、1 transactions/op、1 updates/op、1 session-revokes/op、1 audit-events/op | 0 |

### 5.6 Password reset

延迟与分配：

| 基准 | ns/op | p50 (us) | p95 (us) | p99 (us) | B/op | allocs/op | failure-rate |
|---|---:|---:|---:|---:|---:|---:|---:|
| `BenchmarkPasswordResetGateCheck` | 334.2 | 0 | 0 | 0 | 185 | 8 | 0 |
| `BenchmarkPasswordResetWorkerEmptyScan` | 1627 | 0 | 0 | 0 | 3648 | 6 | 0 |
| `BenchmarkPasswordResetWorkerScanCandidates100` | 130248 | 0 | 1003 | 1433 | 60736 | 2107 | 0 |
| `BenchmarkPasswordResetWorkerClaimBatch100` | 721704 | 959.5 | 1643 | 2420 | 363597 | 10008 | 0 |
| `BenchmarkPasswordResetWorkerSettlementRetry100/retry_wait` | 603709 | 544.8 | 1508 | 2030 | 312304 | 9407 | 0 |
| `BenchmarkPasswordResetWorkerSettlementRetry100/dead_letter` | 579659 | 542.0 | 1465 | 2010 | 312319 | 9407 | 0 |

调用计数（全部由 fake 实际调用/扫描结果派生，均为每 op 值）：

| 基准 | 扫描查询 | 候选 | claim 尝试 | claimed | Renew 调用 | renewed | settlement | settled | failed | 事务 | 加锁 | 回滚 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `BenchmarkPasswordResetWorkerEmptyScan` | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| `BenchmarkPasswordResetWorkerScanCandidates100` | 1 | 100 | 100 | 0 | 0 | 0 | 0 | 0 | 0 | 100 | 100 | 0 |
| `BenchmarkPasswordResetWorkerClaimBatch100` | 1 | 100 | 100 | 100 | 100 | 100 | 400 | 100 | 0 | 300 | 400 | 0 |
| `BenchmarkPasswordResetWorkerSettlementRetry100/retry_wait` | 1 | 100 | 100 | 100 | 100 | 100 | 0 | 0 | 100 | 400 | 500 | 100 |
| `BenchmarkPasswordResetWorkerSettlementRetry100/dead_letter` | 1 | 100 | 100 | 100 | 100 | 100 | 0 | 0 | 100 | 400 | 500 | 100 |

- `BenchmarkPasswordResetGateCheck` 的 `queries/op` 为 1，由 `pgx.Tx` 替身的 `QueryRow` 计数派生。
- `settlement` 列是 settlement 事务内成功执行的语句数（ClaimBatch100 每候选 4 条：invalidate credential、invalidated operation、idempotency、complete job）；`failed` 列是生产 `Fail` 分支成功写入的 job 数。
- `加锁` 列统计生产代码的 `FOR UPDATE` 行锁查询次数（claim 前用户锁、Renew 前用户锁、settlement 前用户锁与 operation/job 锁、Fail 前用户锁）。

### 5.7 SMTP（implicit TLS，进程内）

| concurrency | ns/op | p50 (us) | p95 (us) | p99 (us) | connections/op | peak-connections | B/op | allocs/op | failure-rate |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 817198 | 999.5 | 1708 | 2319 | 1 | 1 | 143802 | 935 | 0 |
| 2 | 409301 | 999.8 | 1686 | 2272 | 1 | 2 | 143112 | 931 | 0 |
| 3 | 353560 | 1001 | 2004 | 2522 | 1 | 3 | 142695 | 929 | 0 |
| 4 | 276257 | 1012 | 2062 | 2532 | 1 | 4 | 142615 | 929 | 0 |

### 5.8 `-race` 复跑（不参与上表数值比较）

| 命令 | 实际结果 |
|---|---|
| `go test -run "^$" -race -bench "BenchmarkArgon2idHash/concurrency=4$" -benchmem ./test/performance/identity` | exit 0，`PASS`，`ok han/test/performance/identity 2.835s`；26404646 ns/op，failure-rate 0，`memory*concurrency-KiB` 262144 |
| `go test -run "^$" -race -bench "BenchmarkPasswordResetWorkerClaimBatch100$" -benchmem ./test/performance/identity` | exit 0，`PASS`，`ok han/test/performance/identity 2.717s`；3299186 ns/op，`claimed/op`、`renewed/op`、`settled/op` 均 100，`failed/op` 0 |
| `go test -run "^$" -race -bench "BenchmarkPasswordResetWorkerSettlementRetry100" -benchmem ./test/performance/identity` | exit 0，`PASS`，`ok han/test/performance/identity 4.277s`；两个子基准 `failed/op` 均 100、`settled/op` 均 0 |

### 5.9 指标图例（诊断字段与未展示原因）

基准输出包含多于主表列的字段。下表说明每个字段的含义、适用基准，以及未进入主表的原因。

| 指标名 | 含义 | 适用基准 | 主表是否展示 / 原因 |
|---|---|---|---|
| `p50-us`、`p95-us`、`p99-us` | 每个 op 的进程内分位数（微秒） | 全部 | 展示（分位数参考值，受本机量化影响） |
| `failure-rate` | 失败 op 数 / 总 op 数 | 全部 | 展示 |
| `heap-samples` | 堆采样循环取到的样本数（`runtime/metrics` 读次数） | Argon2id 哈希/校验 | 未展示：仅反映采样频率，与容量无关的诊断字段 |
| `workers` | 基准内部并发 goroutine 数 | Argon2id、SMTP | 未展示：与子基准名 `concurrency=N` 一一对应，属重复信息 |
| `memory-KiB` | 候选 tuple 的 `MemoryKiB` 参数值 | Argon2id | 未展示：由第 3.1 节 tuple 表给出 |
| `limit-KiB` | 应用内存硬上限 `524288` | Argon2id | 未展示：常量，见第 3.3 节 |
| `iterations` | 候选 tuple 的 `Iterations` 参数值 | Argon2id | 未展示：由第 3.1 节 tuple 表给出 |
| `parallelism` | 候选 tuple 的 `Parallelism` 参数值 | Argon2id | 未展示：由第 3.1 节 tuple 表给出 |
| `memory*concurrency-KiB` | `MemoryKiB * concurrency`，用于与硬上限对账 | Argon2id | 展示（第 5.1、5.2 节） |
| `peak-sampled-heap-KiB` | 采样期 Go 堆对象字节峰值（非 OS RSS） | Argon2id | 展示（第 5.1、5.2 节） |
| `steady-heap-KiB` | `runtime.GC()` 后的同类采样 | Argon2id | 展示（第 5.1、5.2 节） |
| `statements/op` | 每次 op 的 SQL 语句数（fake executor 计数） | Session Create/Revoke | 展示（第 5.4 节附加指标） |
| `queries/op` | 每次 op 的查询数（fake 计数派生） | Session ResolvePrincipal、GateCheck | 展示（第 5.4、5.6 节） |
| `transactions/op` | 每次 op 的 Begin/BeginTx 次数（fake 计数派生） | Session ResolvePrincipal、治理、worker | 展示（第 5.4、5.5、5.6 节） |
| `locks/op`、`updates/op`、`session-revokes/op`、`audit-events/op` | fake repo/audit 的实际调用次数 | 治理 | 展示（第 5.5 节） |
| `scan-queries/op`、`candidates/op`、`claim-attempts/op`、`claimed/op`、`renews/op`、`renewed/op`、`settled/op`、`failed/op`、`settlement-execs/op`、`rollbacks/op` | fake 池/事务的实际调用与扫描结果计数 | worker | 展示（第 5.6 节调用计数表） |
| `connections/op`、`peak-connections` | 每个 op 建立的 SMTP 连接数与并发峰值连接数 | SMTP | 展示（第 5.7 节） |

## 6. 三类结论（必须分开）

| 结论 | 状态 | 依据 |
|---|---|---|
| ① 合法 token/tuple | **PASS（仅机械校验）** | 第 3.2、3.3 节的包络与乘积对账 |
| ② 目标环境可运行 | **未建立（NOT ESTABLISHED）** | 仅完成本机（20 逻辑核 / 15.73 GiB）与 `GOMAXPROCS=4` 模拟运行；真实 4 vCPU / 512 MiB cgroup 未运行。该结论不因预算建议值被授权采用而改变 |
| ③ 满足批准预算 / 生产 SLO | **ADOPTED / NOT VERIFIED** | 容量预算采用项目 owner 于 2026-11-03 授权的建议值（见第 7 节采用记录）；因目标环境未实测，仍不作达标/不达标判定 |

“ADOPTED”只表示预算数值已被 owner 授权采用，“NOT VERIFIED”表示尚未在目标环境验证。二者都不等于目标环境 readiness，也不等于生产 SLO 达标。

## 7. `profile ID → benchmark 报告 → 采用记录`

| profile ID | benchmark 报告 | 采用记录 | 状态 |
|---|---|---|---|
| `identity-argon2id-local-20261103-r01` | `docs/benchmarks/identity.md` | `已采用（owner 于 2026-11-03 授权建议值；见 .spec/wip/identity-auth/evidence-s2.2/profile-adoption-r01.md）` | `adopted / not verified` |

- 该 profile ID 是本轮本地/模拟基线登记；被采用的只是第 2.2 节的建议值（4 vCPU、512 MiB、Go 1.27、`GOMAXPROCS=4`），不是目标环境实测结果。
- **采用不等于目标环境实测**：采用记录确认的是预算数值与口径，不确认任何延迟/内存指标在目标环境达标。
- 目标环境验证仍需在真实 4 vCPU / 512 MiB cgroup 中运行，并核对 OS RSS、SQL/索引与真实 SMTP 行为；本轮未执行。

## 8. NOT RUN 清单

- 真实 PostgreSQL：连接、SQL planner、索引选择、锁等待、MVCC、WAL/`fsync`。
- 真实 password-reset：多 worker lease 竞争、真实 retry/backoff 调度时序、unresolved age 查询、真实批次表数据与索引。
- 真实 settlement SQL：本轮的 settlement/retry 分支仅在 in-memory 替身上验证了生产代码分支可达性与调用数，未验证真实 SQL 结果、行锁语义与索引命中。
- 真实 SMTP：TLS 认证、真实网络、DNS、投递结果、超时与限流。
- 独立持久卷与文件系统耐久性。
- 实际 4 vCPU / 512 MiB cgroup 配额与 OS RSS 峰值。
- 目标环境延迟/吞吐目标与生产 SLO 判定。
- HTTP wire 层与端到端请求路径。
- 目标环境实测型 adoption verification（采用记录已由 owner 授权建立，但验证未执行）。

## 9. 未验证边界与限制

- `peak-sampled-heap-KiB` 是 Go 堆对象字节采样，不是 OS RSS；c3/c4 的采样峰值接近或略高于 `524288 KiB`，但该数值包含瞬时分配与 GC 时序影响，不能直接读作 cgroup 预算通过或失败。
- `steady-heap-KiB` 是 `runtime.GC()` 后的采样，不代表进程常驻集。
- 亚毫秒操作的分位数受本机计时量化影响；这些路径请优先参考 `ns/op`。
- worker 的成功 claim、Renew、settlement 与 Fail 分支由 in-memory 替身驱动生产代码路径，证明的是分支可达性与每 op 调用数，不是真实数据库行为。
- 单次运行，无置信区间；不同主机、GC 状态与后台负载会产生波动。
- 未新增任何依赖；仅使用仓库既有 `pgx/v5` 与 `x/crypto`。
- 本文不修改 `tasks.md`，不声称本卡已关闭或已通过独立评审。