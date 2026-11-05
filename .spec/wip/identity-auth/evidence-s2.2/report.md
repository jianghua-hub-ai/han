# identity-auth-S2.2 实施证据报告

- 角色：Backend-Dev（ADD Role 投影）
- 任务：`identity-auth-S2.2 建立 identity 容量与性能基线`
- 固定基线：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 运行日期：2026-11-03（Asia/Shanghai）
- 修订：review r01 返工（`review-code-s2.2-r01.md`，18,492 B / `ABCA25D5C12310BC3F4DB36C5044BBF7DA4ED2A219645B637B2527CCCFC279FA`）
- 状态：返工证据已交付；**未关闭本卡，未通过独立评审，`tasks.md` 未修改**

## 1. G0 读取清单

- `.codex/skills/add-task-implementation/SKILL.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.spec/wip/identity-auth/review-code-s2.2-r01.md`（全文逐条处理）
- `.add/project/knowledge/tasks/tasks.md` 中 `identity-auth-S2.2` 整段（只读）
- `.add/project/knowledge/design-workspace/identity-auth/design.md` §3.3.1 容量包络（只读）
- `test/performance/article/benchmark_test.go`、`docs/benchmarks/article.md`（结构与文档范式）
- 只读参考：`internal/security/argon2.go`、`internal/identity/password_reset_worker.go`、`internal/identity/password_reset_operator.go`、`internal/identity/administrator_governance.go`、`internal/identity/password_reset_store.go`、`internal/session/store.go`、`internal/mailer/`
- 只读参考：`.spec/wip/identity-auth/evidence-s2.2/profile-adoption-r01.md`（Lead 撰写的采用记录，本任务未创建、未修改）

## 2. 约束遵循

- 写集严格限定 3 个文件：`test/performance/identity/benchmark_test.go`、`docs/benchmarks/identity.md`、本文件。未修改 `tasks.md`、design/ADR、`internal/**`、`cmd/**`、`migrations/**`、article benchmark/docs、评审报告、采用记录，或任何他人文件。
- 未执行任何 Git 写操作；只读使用 `git status`。
- 未删除或归档任何文件。
- 未新增 Go 依赖；`go.mod`/`go.sum` 未改动。
- 未连接真实 PostgreSQL / SMTP / 外网 / 持久卷；全部为进程内确定性替身。
- 未输出或落盘任何凭据、DSN、口令或 API key；测试字面量不是凭据。
- 未编造任何通过阈值；三类结论分离记录，采用状态按 owner 授权与本轮实测如实区分。

## 3. 交付文件

| 文件 | 字节数 | SHA256 |
|---|---:|---|
| `test/performance/identity/benchmark_test.go` | 35418 | `CB59FE4CB420BE9A9A8C0F8C8D85ED7260DA0031F5CD3A33E0E93FA3605C9293` |
| `docs/benchmarks/identity.md` | 21718 | `71E5A103049E282184D14F4447BE1135C8DB218419025B2564D2E62D24787692` |
| `.spec/wip/identity-auth/evidence-s2.2/report.md` | 本文件 | 见最终答复（自引用不计入本表） |

## 4. 本轮修改摘要

包名 `identityperf`，仅测试代码。本轮（review r01 返工）修改集中在 worker 覆盖范围与指标派生：

- `BenchmarkPasswordResetWorkerClaimBatch100` 改为真实成功路径：候选扫描 100 → claim 返回 `attempt_count=1` → Renew `Exec` → settlement 事务（invalidate credential / invalidated operation / idempotency / complete job）全部可达；失败分支另由 `BenchmarkPasswordResetWorkerSettlementRetry100/{retry_wait,dead_letter}` 覆盖。
- 新增 `BenchmarkPasswordResetWorkerScanCandidates100`：仅扫描 100 个候选并让 claim 返回 `pgx.ErrNoRows`，不再用 ClaimBatch 名称暗示成功批量。
- `BenchmarkSessionResolvePrincipal` 改为使用生产 `identity.PasswordResetStore.CheckLockedUser` 作为 Gate，并用可计数 `pgx.Tx` 替身断言 Gate 每 op 恰好一次、总查询数 `4 * b.N`（不满足即 `b.Fatalf`）；`queries/op`、`transactions/op` 由计数派生。
- 治理 `locks/op`、Gate `queries/op`、worker 全部计数指标改为从 fake 的实际调用/扫描结果派生。
- 文档新增指标图例（诊断字段、适用基准、未展示原因），并更新采纳口径与 NOT RUN 清单。

基准全集：

- `BenchmarkArgon2idHash/concurrency=1..4`
- `BenchmarkArgon2idVerify/concurrency=1..4`
- `BenchmarkSessionCreate`、`BenchmarkSessionRevoke`、`BenchmarkSessionResolvePrincipal`
- `BenchmarkGovernanceRevokeAdministrator`
- `BenchmarkPasswordResetGateCheck`
- `BenchmarkPasswordResetWorkerEmptyScan`
- `BenchmarkPasswordResetWorkerScanCandidates100`
- `BenchmarkPasswordResetWorkerClaimBatch100`
- `BenchmarkPasswordResetWorkerSettlementRetry100/{retry_wait,dead_letter}`
- `BenchmarkSMTPImplicitTLSInMemory/concurrency=1..4`

## 5. 验证命令与结果（本轮实际执行）

| # | 命令 | exit code | 实际结果 |
|---:|---|---:|---|
| 1 | `gofmt -l test/performance/identity` | 0 | 无输出 |
| 2 | `go vet ./test/performance/identity` | 0 | 无输出 |
| 3 | `go build ./...` | 0 | 无输出 |
| 4 | `go test -run "^$" -bench . -benchmem -benchtime=200ms ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 9.288s` |
| 5 | `go test -run "^$" -bench . -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 35.189s`（权威运行，文档数值取此结果；进程内总耗时约 36.8s） |
| 6 | `go test -run "^$" -race -bench "BenchmarkArgon2idHash/concurrency=4$" -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 2.835s` |
| 7 | `go test -run "^$" -race -bench "BenchmarkPasswordResetWorkerClaimBatch100$" -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 2.717s` |
| 8 | `go test -run "^$" -race -bench "BenchmarkPasswordResetWorkerSettlementRetry100" -benchmem ./test/performance/identity` | 0 | `PASS`，`ok han/test/performance/identity 4.277s` |
| 9 | `git status --short` / `git status --short --untracked-files=all` | 0 | 只读核对，见第 10 节 |

补充模拟（非验收命令）：`$env:GOMAXPROCS=4; go test -run "^$" -bench "Argon2" -benchmem ./test/performance/identity` → exit 0，`PASS`，`ok han/test/performance/identity 10.843s`，基准后缀 `-4`。

环境事实（实测）：Go `go1.27.0 windows/amd64`；CPU `12th Gen Intel(R) Core(TM) i7-12700H`；逻辑处理器 20；物理内存 `16890978304` bytes（约 `15.73 GiB`）；`GOMAXPROCS` 未设置，基准输出后缀 `-20`。

实现期修正：首次编译暴露两处返工残留（`transactor.begins` 应为 `counters.begins`；新增 `errors` 导入缺失），已修复后重跑全部验证命令。

## 6. Argon2 c1..c4 关键指标（权威运行）

哈希：

| c | ns/op | p50 us | p95 us | p99 us | peak-heap KiB | steady-heap KiB | memory*conc KiB | failure |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 38582383 | 38192 | 47037 | 48700 | 197251 | 700 | 65536 | 0 |
| 2 | 26129111 | 51535 | 59580 | 62329 | 263001 | 857 | 131072 | 0 |
| 3 | 19291217 | 57139 | 61748 | 62357 | 394146 | 930 | 196608 | 0 |
| 4 | 16855507 | 65982 | 77322 | 81797 | 525275 | 985 | 262144 | 0 |

校验：

| c | ns/op | p50 us | p95 us | p99 us | peak-heap KiB | steady-heap KiB | memory*conc KiB | failure |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | 39072675 | 38933 | 43589 | 44651 | 132064 | 989 | 65536 | 0 |
| 2 | 27357336 | 54421 | 60793 | 62053 | 263159 | 1014 | 131072 | 0 |
| 3 | 21083348 | 61286 | 72162 | 73668 | 394241 | 1024 | 196608 | 0 |
| 4 | 19789876 | 78149 | 89517 | 94195 | 525324 | 1031 | 262144 | 0 |

对账：`memory*concurrency` = 65536 / 131072 / 196608 / 262144 KiB，均 ≤ `524288 KiB`（余量 458752 / 393216 / 327680 / 262144 KiB）。该对账仅证明应用配置合法，不证明目标环境预算通过（`peak-sampled-heap-KiB` 是 Go 堆采样，不是 OS RSS）。

## 7. Findings 处理（F001–F004）

| Finding | 级别 | 结论 | 证据 |
|---|---|---|---|
| F001 ClaimBatch100 未覆盖成功 lease/retry/batch | HIGH | **已修复** | ClaimBatch100 现驱动成功 claim+Renew+settlement；扫描型基准改名 `ScanCandidates100`；新增 `SettlementRetry100/{retry_wait,dead_letter}` 覆盖 retry/dead-letter；指标由 fake 计数派生并断言 |
| F002 Session resolve 绕过 Gate、`3 queries/op` 硬编码 | MEDIUM | **已修复** | Gate 改为生产 `identity.PasswordResetStore.CheckLockedUser`，查询计数派生；断言 Gate 每 op 一次且总查询 `4 * b.N`；输出 `4.000 queries/op` |
| F003 部分指标硬编码 | LOW | **已修复（涉及 F001/F002 的 fake）** | 治理 locks、Gate queries、worker 全量计数均由 fake 实际调用/扫描结果派生；不同基准观测值不同（EmptyScan 0 / Scan 100 / ClaimBatch 400 locks）可佐证非常量 |
| F004 文档诊断指标未列全 | INFO | **已修复** | `identity.md` 新增 §5.9 指标图例，覆盖 `heap-samples`、`workers`、`memory-KiB`、`limit-KiB`、`iterations`、`parallelism`、`peak-sampled-heap-KiB`、`steady-heap-KiB` 等 |

### 7.1 F001 细节

- 生产路径核对：`internal/identity/password_reset_worker.go`（`Claim` 的 `UPDATE ... SET status='leased' ... RETURNING j.attempt_count`；`Renew` 的 `SET lease_expires_at=$1,updated_at=$2`；`Fail` 的 `SET status=$1,next_attempt_at=$2,last_failure_code...`）与 `internal/identity/password_reset_operator.go`（`passwordResetSettlementLockUserSQL`、`passwordResetSettlementLockOperationSQL`、`settleUnacceptedPasswordReset` 的三条 `Exec`、`passwordResetSettlementCompleteJobSQL`）。
- 权威运行的实际计数（ClaimBatch100）：`candidates/op` 100、`claim-attempts/op` 100、`claimed/op` 100、`renews/op` 100、`renewed/op` 100、`settled/op` 100、`settlement-execs/op` 400、`failed/op` 0、`transactions/op` 300、`locks/op` 400、`rollbacks/op` 0；并断言 `claimed == renewed == settled == 100 * b.N`、`failed == 0`、`unexpected == 0`、`result.failures == 0`。
- retry 路径实际计数（SettlementRetry100）：`claimed/op` 100、`renewed/op` 100、`failed/op` 100、`settled/op` 0、`rollbacks/op` 100，`retry_wait` 与 `dead_letter` 两个子基准的实际写入状态分别为 `retry_wait` / `dead_letter`（由 fake 记录的 `Fail` 状态参数断言）。
- 仍未覆盖：真实 SQL 结果、真实行锁语义、索引命中、多 worker lease 竞争与真实 backoff 调度时序（见第 9 节 NOT RUN）。

### 7.2 F002 细节

- 生产路径核对：`internal/session/store.go` 的 `preReadSessionUser`（1 查询）、`lockSessionUser`（1 查询）、`passwordResetGate.CheckLockedUser`（1 查询）、`readResolvedSession`（1 查询）。
- 基准断言：`gateQueries == b.N`（否则 `b.Fatalf`），`preReads + lockQueries + gateQueries + sessionReads == 4 * b.N`。
- 权威运行：`BenchmarkSessionResolvePrincipal` = 4802 ns/op、1297 B/op、31 allocs/op、`4.000 queries/op`、`1.000 transactions/op`、failure-rate 0。

### 7.3 F003 细节

- 派生来源：`governanceRepository.locks`、`lockedUserTx.queries`、`sessionCounters`、`workerCounters`（`QueryRow`/`Exec`/扫描行计数）。
- 常量已消除：`locks/op`、`queries/op`、`candidates/op`、`claim-attempts/op`、`claimed/op`、`renews/op`、`renewed/op`、`settled/op`、`failed/op`、`transactions/op`、`rollbacks/op`、`scan-queries/op`。
- 佐证：同一轮运行中不同基准的 `locks/op` 分别为 0（EmptyScan）、100（ScanCandidates100）、400（ClaimBatch100）、500（SettlementRetry100），说明指标随 fake 实际调用变化，不是常量。

## 8. 三类结论与采纳口径

| 结论 | 状态 | 说明 |
|---|---|---|
| ① 合法 token/tuple | PASS（仅机械校验） | 设计包络与乘积约束均满足；本轮不变 |
| ② 目标环境可运行 | 未建立（NOT ESTABLISHED） | 仅本机（20 逻辑核 / 15.73 GiB）与 `GOMAXPROCS=4` 模拟；真实 4 vCPU / 512 MiB cgroup 未运行。**该结论不因预算被授权采用而改变** |
| ③ 满足批准预算 / 生产 SLO | ADOPTED / NOT VERIFIED | 预算采用 owner 于 2026-11-03 授权的建议值；目标环境未实测，故不作达标/不达标判定 |

`profile ID → benchmark 报告 → 采用记录`：

| profile ID | benchmark 报告 | 采用记录 | 状态 |
|---|---|---|---|
| `identity-argon2id-local-20261103-r01` | `docs/benchmarks/identity.md` | `已采用（owner 于 2026-11-03 授权建议值；见 .spec/wip/identity-auth/evidence-s2.2/profile-adoption-r01.md）` | `adopted / not verified` |

- 采用记录由 Lead 撰写（5,373 B / `70DCB73DBA5350BB8A8223FEC4E0F016F138A3C68D95145EE2502BB43E996088`），本任务只引用、不修改。
- **采用不等于目标环境实测**：采用确认的是预算数值与口径，不确认任何指标在目标环境达标。
- 说明（供 Lead 复核）：采用记录 §2.3 引用的观测值来自返工前（r00）的权威运行；本轮权威运行数值不同（例如 Argon2 c4 hash p95 由约 89.8 ms 变为 77.3 ms，worker batch p95 由约 984 us 变为 1643 us）。预算本身未变，本任务无权修改该记录。

## 9. 未验证边界 / NOT RUN

- 真实 PostgreSQL：连接、SQL planner、索引选择、锁等待、MVCC、WAL/`fsync`。
- 真实 password-reset：多 worker lease 竞争、真实 retry/backoff 调度时序、unresolved age、真实批次表数据与索引。
- 真实 settlement/retry SQL 结果与行锁语义（本轮仅验证 in-memory 替身上生产分支可达性与每 op 调用数）。
- 真实 SMTP：TLS 认证、真实网络、DNS、投递结果、超时与限流。
- 独立持久卷与文件系统耐久性。
- 实际 4 vCPU / 512 MiB cgroup 配额与 OS RSS 峰值。
- 目标环境延迟/吞吐目标与生产 SLO 判定。
- HTTP wire 层与端到端请求路径。
- 目标环境实测型 adoption verification（采用记录已由 owner 授权建立，验证未执行）。

限制：`peak-sampled-heap-KiB` 为 Go 堆对象采样，不是 OS RSS；c3/c4 采样峰值接近或略高于 `524288 KiB`，含瞬时分配与 GC 时序影响，不能直接读作预算通过/失败。亚毫秒操作的分位数受本机计时量化影响，优先参考 `ns/op`。

## 10. 工作树核对

`git status --short` 显示工作树在本任务开始前已包含大量其他角色改动与未跟踪文件（`test/`、`docs/` 整体未跟踪）。本任务写入的路径仅：

- `test/performance/identity/benchmark_test.go`
- `docs/benchmarks/identity.md`
- `.spec/wip/identity-auth/evidence-s2.2/report.md`

未触碰、未还原、未清理任何他人改动。说明：该工作树存在其他并发活动，按 mtime 反查写集外改动不可靠，故本节只声明本任务自身的写入路径与动作，不对他人改动归属作判断。

## 11. 完成度判断

- 已完成：F001–F004 返工；进程内 Argon2id / Session（含生产 Gate）/ 治理事务 / password-reset Gate 与 worker（空扫描、claim 未命中扫描、成功 lease+settle、retry/dead-letter）/ SMTP 基线；指标派生化；文档指标图例与采纳口径更新。
- 未完成：真实目标环境（4 vCPU / 512 MiB cgroup）实测、OS RSS、真实 PG/SMTP 行为、生产 SLO 判定。
- 结论：返工证据已交付，但**不得关闭本卡**，也不得声称已通过独立评审（待 r02 复评）。`tasks.md` 未修改。