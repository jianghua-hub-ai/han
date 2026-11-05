# identity-auth-S2.2 独立代码评审 r01

## 0. 评审身份、对象与授权

- 角色：Reviewer（ADD 非作者独立评审）。
- `objectType`：code。
- 固定工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；基线 `HEAD=6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 评审对象（精确三文件，哈希按本轮读取复核）：

| 对象 | 字节数 | SHA256 |
|---|---:|---|
| `test/performance/identity/benchmark_test.go` | 25618 | `951D1CA075670AAB7663BDB0DE1E017E6CA09D9DD71CCCD07ED27A306A608CEE` |
| `docs/benchmarks/identity.md` | 12846 | `C41400D126C02CCCB97CBBCA67BDE3D5E3A2B33CADEA7AE9C99C57532F0FB6C4` |
| `.spec/wip/identity-auth/evidence-s2.2/report.md` | 7845 | `511472896FE053AA7FA054D1CBC78FFC50481BE1EC314A7B44A4AA1C8F360CAA` |

- 上游任务卡：`.add/project/knowledge/tasks/tasks.md:326` 仍为 `[ ]`；`:337` 仍写 `blocked-for-execution`；`:341` 要求 Argon2id、Session、治理事务、SMTP、恢复批量，以及 password-reset worker lease/retry/batch、unresolved age、Gate 查询、事务/index、SMTP 并发资源测量。**本报告不修改 `tasks.md`，不声称该卡已关闭。**
- 设计包络：`.add/project/knowledge/design-workspace/identity-auth/design.md` §3.3.1，候选值与约束为 `HAN_ARGON2_MEMORY_KIB` 19456..262144、`MemoryKiB * MaxConcurrency <= 524288`、并发 1..4；配置表位置为 `:1225`、`:1227`、`:1231`。
- 授权来源：ADR-0028（`.add/project/knowledge/decisions/framework.md:816-854`，用户 2026-10-31 项目级授权；本轮输入声明用户 2026-11-03 授权目标部署环境与容量预算采用建议值）。ADR-0028 允许以模拟/建议值替代不可得的外部环境，但要求显式标注“模拟/未真实验证”，不得伪造 PASS。
- **AI 代理评审，非真人签字。** 本报告只代理评审与记录，不代理生产采用、真实环境验收或真人签收；代理范围是本次 code review 及其 findings。

## 1. G0 输入核对

- 已固定上述三文件的内容、字节数与 SHA256；评审期间未替换对象。
- 已读取任务卡整段、设计 §3.3.1、ADR-0028，以及被评审代码对应的生产消费者/生产者。
- 已读取项目 profile 适用的 coding/no-hardcode/security 规则；未启用 profile 不作为要求。
- 作者报告、Lead 摘要和聊天中的断言不作为通过证据；关键结论均回到代码、文档、命令输出和文件状态核对。
- 写权限仅限新建本报告；未修改、删除、归档任何既有文件，未连接真实 PostgreSQL/SMTP/外网，未执行 Git 写操作。

## 2. 两轮实际审查范围

### Round 1：对象与直接行为

- 全文读取三份对象，检查 benchmark 名称、生产调用路径、替身、指标名、文档表格、NOT RUN 清单与三类结论。
- 检查 Argon2id 候选 tuple、`security.NewGatedArgon2idHasher`、Session Store、治理服务、password-reset Gate/worker、`mailer.SMTPMailer.Send`。
- 实际运行格式、vet、短基准与 race 检查，见第 3 节。
- 主要直接发现：worker 的 claim 全部为 `ErrNoRows`（F001）；Session resolve 绕过 Gate 且查询数硬编码（F002）；若干指标硬编码（F003）；文档诊断指标覆盖不完整（F004）。

### Round 2：消费者与契约

- 沿生产调用链核对 `internal/identity/password_reset_worker.go`、`internal/identity/password_reset_store.go`、`internal/session/store.go`、`internal/security/argon2.go`、`internal/mailer/**`，以及任务卡、设计包络和 ADR-0028。
- 核对 worker 的 `Claim -> Renew -> settle -> Fail` 分支、Session resolve 的 pre-read/lock/gate/read 查询、治理事务与 SMTP 进程内 TLS 路径。
- 核对三文件写集、`go.mod`/`go.sum` mtime 和现有工作树的其他 dirty 项。
- 未扩大到全仓所有消费者；未执行真实 PG/SMTP、目标 cgroup、OS RSS、HTTP wire 或端到端路径。因此对这些范围不给 PASS。

## 3. 命令与实际结果

工作目录为仓库根目录；以下为本轮实际执行结果。

| # | 命令 | 实际结果 |
|---:|---|---|
| 1 | `gofmt -l test/performance/identity` | exit 0，无输出 |
| 2 | `go vet ./test/performance/identity` | exit 0，无输出 |
| 3 | `go test -run '^$' -bench . -benchmem -benchtime=100ms ./test/performance/identity` | exit 0，`PASS`，`ok han/test/performance/identity 5.301s` |
| 4 | `go test -run '^$' -race -bench 'Argon2idHash/concurrency=4' -benchtime=100ms ./test/performance/identity` | exit 0，`PASS`，`ok han/test/performance/identity 1.640s`；4 workers，failure-rate 0 |

第 3 次短基准的关键实际值（单次短跑，仅用于本轮可执行性与指标核对）：

| 路径 | 关键结果 |
|---|---|
| Argon2id Hash c1..c4 | 45,518,833 / 50,960,533 / 28,728,225 / 21,675,267 ns/op；`memory*concurrency-KiB` 65536 / 131072 / 196608 / 262144；failure-rate 0 |
| Argon2id Verify c1..c4 | 44,215,833 / 27,916,725 / 25,816,800 / 22,462,833 ns/op；failure-rate 0 |
| Session | Create 521.0 ns/op；Revoke 192.7 ns/op；ResolvePrincipal 2085 ns/op，报告 `3 queries/op`、`1 transactions/op` |
| 治理 | RevokeAdministrator 840.0 ns/op；`2 locks/op`、`1 transactions/op`、`1 updates/op`、`1 session-revokes/op`、`1 audit-events/op` |
| Password reset | GateCheck 281.2 ns/op；EmptyScan 2155 ns/op；ClaimBatch100 88870 ns/op、`100 candidates/op`、`1 scan-queries/op`、`100 transactions/op`、p95 999.7 us、failure-rate 0 |
| SMTP implicit TLS | c1..c4 678774 / 486522 / 347942 / 267331 ns/op；均 `1 connections/op`，peak-connections 1 / 2 / 3 / 4 |

说明：作者报告记录的是默认 `-benchtime` 的 34.526s 运行，Lead 记录的是 31.774s 运行；本轮 100ms 短跑数值受运行时间、GC、调度和 `-20` 默认并行度影响，与文档表格不同不构成数值造假。短跑验证了基准可运行、指标可输出、失败率为 0，但不用于重算作者表格。

## 4. Findings

### [HIGH] ClaimBatch100 实际执行 100 次失败的 claim，未覆盖成功 lease/retry/batch

- finding ID：`IA-S2.2-R01-F001`
- 对象/修订：code；三文件固定对象（`benchmark_test.go`、`identity.md`、`evidence-s2.2/report.md`）。
- 位置：`test/performance/identity/benchmark_test.go:701-702`、`:785-800`；生产行为见 `internal/identity/password_reset_worker.go:316-336`、`:429-431`。
- 现状：`workerTx.QueryRow` 对所有 `UPDATE password_reset_recovery_jobs` 返回 `pgx.ErrNoRows`。生产 `Claim` 将该错误映射为 `(lease, claimed=false, nil)`，`reconcileCandidate` 随后在 `:429-431` 直接返回，不会调用 `Renew`（`:436`）或 settlement（`:444`）。因此 `BenchmarkPasswordResetWorkerClaimBatch100` 对 100 个候选只产生 100 次**未成功领取**的 claim 事务；`100 transactions/op` 不是 100 次成功 claim，`failure-rate=0` 也只表示未返回 error，不表示领取成功。
- 触发：运行 `BenchmarkPasswordResetWorkerClaimBatch100`，或生产 worker 扫描到候选后 claim 条件不满足/无行。当前 fixture 的每次 claim 都触发该路径。
- 影响：任务卡 `:341` 明确要求 password-reset worker lease/retry/batch；该证据只覆盖候选扫描与失败 claim 尝试，没有覆盖成功 lease、Renew、retry/backoff、dead-letter 或 Gate 收敛。报告 `evidence-s2.2/report.md:134` 仍写“已交付 ... worker 基线”，文档 `identity.md:194` 的 `100 transactions/op` 容易被误读为成功批量处理。该缺口阻断本卡按声明范围关闭。
- 建议：让 fixture 返回一个成功 claim（例如 `attempt_count=1`）并实现后续 `Exec` 计数，增加 `claimed/op`、`renewed/op`、`settled/op`、`failed/op`；若本轮只测扫描，应将基准重命名为 `ScanCandidates100` 并在报告、NOT RUN 中明确“无成功 claim、lease/retry/backoff/dead-letter 未测”，不要用 ClaimBatch 名称暗示已覆盖。
- 验证：修复后断言 `claimed/op > 0`、Renew 与 settlement 调用计数可观察；若仅保留扫描，断言 `claimed/op=0` 并在三份文档中登记该限制；重跑第 3 节基准及独立复评。

### [MEDIUM] Session ResolvePrincipal 绕过 password-reset Gate，`3 queries/op` 不是生产口径

- finding ID：`IA-S2.2-R01-F002`
- 对象/修订：code；三文件固定对象。
- 位置：`test/performance/identity/benchmark_test.go:434-438`、`:492-516`、`:513`；生产 `internal/session/store.go:305`、`:350`、`:365`、`:387`；文档 `docs/benchmarks/identity.md:101`、`:180`。
- 现状：`allowPasswordResetGate.CheckLockedUser` 直接 `return nil`，不执行查询，也不计数；`BenchmarkSessionResolvePrincipal` 使用该 no-op Gate。生产 `ResolvePrincipal` 路径实际包含 pre-read（`:350`）、用户锁（`:365`）、Gate 查询（`:305`）和 session 读取（`:387`）四次查询。基准却硬编码 `3 queries/op`，文档也重复“3 queries/op”。
- 触发：运行 `BenchmarkSessionResolvePrincipal`；所有测试 fixture 调用均走 no-op Gate。生产真实请求则会在同一事务内执行 Gate 查询。
- 影响：`queries/op` 少算一次，Session resolve 的查询负载被低估；更重要的是原始要求的“推动生产 Gate 路径”在 Session 集成路径上未实现。虽然 `BenchmarkPasswordResetGateCheck` 直接调用了生产 `PasswordResetStore.CheckLockedUser`（见 `benchmark_test.go:663-671`），但不能替代 Session resolve 的真实集成口径。
- 建议：用返回 nil 但计数的 Gate fixture，或直接接 `identity.PasswordResetStore` + 可观察 `pgx.Tx`，由计数器派生 `queries/op`；若有意绕过 Gate，必须在文档和报告中明确写成“bypassed/no-op”，不能把 3 当作生产查询数。
- 验证：修复后断言 `queries/op` 为 4（pre-read + lock + Gate + read），并断言 Gate 调用恰好一次；重跑短基准和 race 检查。

### [LOW] 部分指标为硬编码，生产调用数变化时会静默漂移

- finding ID：`IA-S2.2-R01-F003`
- 对象/修订：code；三文件固定对象。
- 位置：`test/performance/identity/benchmark_test.go:649`、`:671`、`:782`、`:797`。
- 现状：`locks/op=2`、Gate `queries/op=1`、空扫描 `candidates/op=0`、ClaimBatch `candidates/op=100` 均直接传常量；这些值没有从 fake 事务、repo 或扫描结果派生。当前 fixture 下数值与设计意图接近，但无法证明生产调用数量仍一致。
- 触发：生产代码增加/减少锁、Gate 查询或候选扫描逻辑，而基准 fixture 未同步。指标仍显示旧值，可能形成静默维护风险。
- 影响：不改变当前运行结果，但降低基线对回归的敏感度；属于局部质量/测试可观测性问题。
- 建议：让 fake 的 `QueryRow`/`Exec`、repo 锁计数和扫描结果计数成为唯一指标来源；空扫描也应由结果长度派生。
- 验证：在 scratch test 中故意改变 fake 调用次数，确认 `locks/op`、`queries/op`、`candidates/op` 随实际计数变化。

### [INFO] 文档未完整列出部分诊断指标

- finding ID：`IA-S2.2-R01-F004`
- 对象/修订：code/文档；三文件固定对象。
- 位置：`docs/benchmarks/identity.md:127-203`；报告 `evidence-s2.2/report.md:52`。
- 现状：代码实际输出 `heap-samples`、`workers`、`memory-KiB`、`limit-KiB`、`iterations`、`parallelism` 等指标；文档表格只展示部分指标，`steady-heap-KiB` 仅在 Argon2 表格出现。报告 §4 列出了这些指标名。
- 触发：读者试图从基准输出逐项复现文档表格时，无法从表格本身确认哪些是诊断指标、哪些被省略。
- 影响：不构成阈值伪造或 PASS 错误；属于文档可审计性和指标口径完整性不足。
- 建议：在文档表格下增加指标图例，标注诊断-only 字段、适用基准和未展示原因。

## 5. 关闭条件逐项覆盖表

| 关闭条件/测量项 | 状态 | 证据与说明 |
|---|---|---|
| Argon2id 候选 tuple 与 c1..c4 hash/verify | 已测 | 使用 `security.NewGatedArgon2idHasher`，短基准与 race 均 PASS；仅本机/in-process，非目标环境。 |
| `memory*concurrency <= 524288 KiB` | 已测（仅机械对账） | 65536/131072/196608/262144 均不超过 524288；不证明目标 cgroup 或 OS RSS 通过。 |
| Session Create/Revoke | 已测 | 生产 `session.Store` + fake executor；只覆盖成功路径，未覆盖失败/回滚。 |
| Session ResolvePrincipal | 部分 | 生产 Store 路径大部分被调用，但 Gate 为 no-op，查询数硬编码为 3；见 F002。 |
| 治理事务 `RevokeAdministrator` | 已测 | 生产 `NewAdministratorGovernanceService.RevokeAdministrator` + fake transactor/repo/audit；locks 为硬编码计数，见 F003。 |
| SMTP `SMTPMailer.Send` | 已测 | 生产 mailer 路径 + `net.Pipe` + 临时自签 TLS；非真实 SMTP/网络。 |
| SMTP 并发资源测量 c1..c4 | 已测 | 手工 worker 数与 `workers` 一致，peak-connections 1..4；仅进程内连接资源。 |
| 恢复批量（batch=100） | 部分 | 扫描 100 个候选，但 100 次 claim 全部 `ErrNoRows`；无成功领取/收敛，见 F001。 |
| password-reset worker lease/retry/batch | 未测 | `identity.md:226`、`report.md:112` 已列 lease 竞争、retry/backoff、dead-letter 为 NOT RUN；成功 lease 与 batch settlement 也未测。 |
| unresolved age | 未测 | `identity.md:226`、`report.md:112` 明确为 NOT RUN；本对象未产生该指标。 |
| Gate 查询 | 部分 | `BenchmarkPasswordResetGateCheck` 直接驱动生产 `CheckLockedUser`；Session resolve 集成路径被 no-op Gate 绕过，见 F002。 |
| 事务/index | 部分 | 仅验证 fake 事务/锁 shape；真实 PostgreSQL planner、索引选择、MVCC、WAL 均 NOT RUN（`identity.md:225`、`report.md:111`）。 |
| 目标环境 CPU/内存/Go 参数与延迟/峰值 RSS | 未建立 | 仅有本机 20 逻辑核/15.73 GiB 与 `GOMAXPROCS=4` 模拟；真实 4 vCPU/512 MiB cgroup、OS RSS 未测；`peak-sampled-heap-KiB` 是 Go 堆对象采样，不是 RSS。 |
| 三类结论分离 | 已核对 | `identity.md:209-211`、`report.md:99-101` 将合法 tuple、目标环境、批准预算/SLO 分开；未发现把后两者写成 PASS。 |
| `profile ID -> benchmark -> 采用记录` | 部分 | 有 local profile ID 和 benchmark 报告；采用记录明确为无/pending，未代签。 |

## 6. 阈值与结论诚实性核验

| 结论 | 文件中的实际状态 | 评审结论 |
|---|---|---|
| ① 合法 token/tuple | `PASS（仅机械校验）`，`identity.md:93`、`report.md:99` | 范围有限且标注了“仅机械校验”，可接受；不等于目标环境或生产可用。 |
| ② 目标环境可运行 | `NOT ESTABLISHED`，`identity.md:210`、`report.md:100` | 与 NOT RUN 清单一致，未伪造 PASS。 |
| ③ 满足批准预算/生产 SLO | `NOT ADOPTED / PENDING`，`identity.md:211`、`report.md:101` | 无批准预算、无采用记录，未伪造 PASS。 |

`peak-sampled-heap-KiB` 明确声明为 `runtime/metrics` 的 Go 堆对象字节采样，`identity.md:105`、`:236` 与 `report.md:120` 均明确“不是 OS RSS”；未发现把它写成 RSS 或 cgroup 通过。总体未发现“目标环境可运行/满足批准预算”的编造通过；F001 是 worker 覆盖范围与命名/报告表述的过度声明，不是目标环境 PASS 伪造。

## 7. 待决定、限制与非阻断事项

- 目标部署环境、容量预算、延迟目标、采用记录仍未形成；属于 Lead/用户或授权责任人的决定，不是 Reviewer 可直接代签的通过结论。
- 真实 PostgreSQL、真实 SMTP/TLS/AUTH、真实网络、实际 cgroup、OS RSS、HTTP wire 和端到端路径均未验证；ADR-0028 允许模拟替代，但本文不把模拟结果外推为生产 readiness。
- 当前 `identity.md:226`/`report.md:112` 的 NOT RUN 登记是诚实的，但不足以解除任务卡要求的 worker lease/retry/batch 和 unresolved age 缺口。
- 工作树整体不是干净基线：存在其他角色/历史任务的 dirty 与未跟踪项；本任务三对象均为新增。`tasks.md` 与 `go.mod` 在当前工作树已有修改，但其 mtime 早于三对象（`tasks.md` 2026-11-03 15:41:09；`go.mod` 2026-10-21 18:25:48；`go.sum` 2026-10-21 17:59:38），未发现本任务对 `internal/**`、`cmd/**`、`migrations/**`、既有测试、design/ADR 的写后文件（检查阈值 2026-11-03 15:40 后无命中）。这不等于工作树整体可交付，只是本对象的写集核对。
- 未发现新增依赖的可执行证据：benchmark 只使用仓库既有导入；未执行 `go mod tidy`，`go.mod`/`go.sum` 时间戳早于本任务对象。

## 8. 复审条件与结论

### 复审条件

1. 处理 F001：要么真实驱动成功 claim、Renew、settle/retry 并提供可观察计数，要么将基准降级为扫描基准，并在三份文件中明确“成功 lease/retry/batch 未测”。仅改测试名而不修文档/指标不足以关闭 HIGH。
2. 处理 F002：让 Session resolve 走真实 Gate 或明确登记 no-op 边界，并将 `queries/op` 改为计数派生；修后应显示 4 次查询并覆盖 Gate 调用。
3. F003 可作为非阻断质量项排期，但若 F001/F002 修复涉及相关 fake，应一并改为计数派生，防止新证据再次硬编码。
4. 重新运行 `gofmt -l`、`go vet`、短基准及适用的 race 检查；补做修改后的 worker 成功路径证据。复审必须引用 F001–F003，逐项给出关闭/仍存在/受限。
5. 不得把本轮 `CHANGES REQUIRED` 解释为任务关闭；`tasks.md:326` 的 `[ ]`、`:337` 的 `blocked-for-execution` 仍有效。

### 结论

**`CHANGES REQUIRED`**。

- 无 BLOCKER；1 个 HIGH（F001）、1 个 MEDIUM（F002）、1 个 LOW（F003）、1 个 INFO（F004）。
- F001 违反任务卡明确要求的 worker lease/retry/batch 证据范围，阻断当前任务；F002 使 Session 查询基线不完整；F003/F004 不单独阻断，但应在修订中保持指标/文档一致性。
- 未发现目标环境或批准预算的伪造 PASS；三类结论分离且 `peak-sampled-heap-KiB` 的 Go heap 语义已明确。
- **AI 代理评审，非真人签字**；授权来源为 ADR-0028（`framework.md:816-854`）及本轮输入声明的用户 2026-11-03 目标环境/预算建议值授权。代理范围仅限本报告评审，不代表生产采用、真实环境验收或真人签收。
- 本卡**未关闭**；`tasks.md` 未修改，三份被评审文件未修改。