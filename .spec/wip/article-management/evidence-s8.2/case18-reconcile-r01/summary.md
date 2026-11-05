# S8.2 Case18 复核对账 r01（2026-11-05，Lead 直接执行）

本文件只记录本轮在现役 revision 上的真实执行与逐 finding 事实复核；不代原 Reviewer 关闭 finding，不关闭 S8.2/S1.5。

## 固定输入

| 项 | 实际值 |
| --- | --- |
| 工作区 / 分支 | `D:/AID/VSCode/Codex/Han` / `main` |
| HEAD | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `cmd/han-server/recovery_runtime.go` | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` / 37,431 B |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173,317 B |
| PostgreSQL | 18.6，`127.0.0.1:5433`，`HAN_TEST_POSTGRES_DSN` 已设置（不打印值） |

## 本轮实际执行（全部 exit 0）

| 命令 | 结果 | 日志 SHA256 / 字节 |
| --- | --- | --- |
| `go test -tags=integration -timeout=20m ./cmd/han-server -run '^TestRecoveryProductionChainComplete$' -count=1 -v` | PASS，10.87s；8 个子测试全 PASS | `5C5604603DEF586660D314E6361B36173056F98A9861111656517AD22E6392A9` / 9,746 B |
| `... -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -v` | PASS，0.30s；8 个子测试全 PASS | `B84D211E5ECE47B5530A00F1A139A11016B0217C18FA6AC6954B90FA62EF9C9F` / 1,895 B |
| `... -run '^TestRecoveryIsolationMatrix$' -count=1` | PASS，0.19s | `C5392EB087AA560832066421D727358DAFE030B834CD4FE161E5731729241DB2` / 35 B |

生产链实际子测试：`ResumeCoordinatorRecoveryCompleteAndOpen`、`SecondReadStatusAndMarkConsumed`、`CanonicalReplaySameResult`、`ClosedUnknownKeepIsolation`、`SingleRuntimeStorePool`、`SimulatedJournalKeyringBoundary`、`NoSecondRuntimeRouterPoolAuthorityFallback`、`harness`。

## 逐 finding 复核（只读判断）

- **C18-COV-F003（HIGH）**：F003 的事实前提是「S4.20 d7 证据绑定旧 `recovery_runtime.go 900F31F5…/35832`，当前 revision 无法核完整恢复后开流」。现役文件实为 `1C679377…/37431`，与 S4.20 关闭证据同一哈希；本轮在该 revision + 真实 PG 上 `TestRecoveryProductionChainComplete` PASS，覆盖 resume→replay→journal consume→succeeded→activate/开流、二次 ReadStatus、MarkConsumed。**旧哈希事实层已消除**；但 Journal/HMAC/keyring 为 simulated（`windows_unsupported`），production Enter/Install 仍按 ADR-0030/0031 设计关闭。**建议交原 Reviewer 复审后决定是否降级/关闭，本轮不自行 CLOSED。**
- **C18-COV-F004（MEDIUM）**：**仍 OPEN**。`PublicReadyAdmitsSameSession` 实读使用内存 fixture（`newOrdinaryFixture` + `articleAuthorRootResolver`），未执行真实 `SessionStore` SQL/撤销证据。
- **C18-COV-F002（HIGH）**：**仍 OPEN**。`TestRecoveryIsolationMatrix` 实读 `newS419RecoveryRuntime` 仅区分 `public_ready` / `unknown` / 其他=`closed`，缺 `failed` / `expired` / `undetermined`，且为 fake runtime、无真实 PG/双系统。

## 结论与下一步

- S8.2 保持 `[-]`，`blocked-upstream` 不再由 S4.20 能力缺口单独构成，但仍受 F002/F004 未闭环与 F003 待复审约束；S1.5 仍受 Admin2 整体接收阻塞。
- 最小增量：F004 增加真实 PG `SessionStore` 读取/撤销用例；F002 把隔离矩阵扩到 `failed`/`expired`/`undetermined` 并接真实 runtime。
- 本轮无源码修改、无 Git 写、无清理；未关闭或重开任何 finding。

## 2026-11-05 F004 实施与 F002 G0

- **F004（已实施，待原 Reviewer 复核）**：新增 `cmd/han-server/case18_session_store_integration_test.go`（`//go:build integration`），用真实 Postgres `session.Store`（与生产 root 同一依赖集）解析活动会话，再执行 `Revoke`，并断言撤销后同一 token 不再解析。单跑 PASS 2.45s；与 `TestRecoveryProductionChainComplete`、`TestRecoveryMainBusinessDispatcher`、`TestRecoveryIsolationMatrix` 合并回归 PASS 8.503s（该指针已过期；当前日志为 8.672s / 35 B / `ADD9EB0CE4AD9389E95878B2C3F0E1CC09EDD9FE195CCC406723749C8D018F41`，见 2026-11-05 finding 级复审 R01 的指针漂移说明）；`gofmt -l` 无输出；`go vet -tags=integration ./cmd/han-server` exit 0。
- **F002（G0 结论：需先定映射，未实施）**：生产 admission 观测只有 `closed` / `unknown`（错误路径）/ `public_ready`（`internal/recovery/runtime.go:27-29`）；`failed` 是运行结果，且被 admission 直接拒绝（`internal/recovery/admission.go:726` → `ErrAdmissionBinding`）。仓库中不存在 `failed` / `expired` / `undetermined` 三种 admission 状态。要按原 finding 字面覆盖，必须先明确「运行结果/生命周期 → 准入观测」的映射，并让真实 runtime 进入 started/checking 态后诱导失败；当前 harness 无该入口，属更大的测试切片，本轮未写代码。

## 2026-11-05 F002 映射 G0（未实施）

见 [f002-mapping-g0.md](f002-mapping-g0.md)：Case18 fixture 的 Verify 走 temporary 路径，实测返回 internal_ready/verified，与 ecovery_runs.outcome 无关；把 outcome 改为 ailed 结果不变。故 F002 必须走 live/exclusive 路径（full harness + Exclusive 端口 + 启动后失败注入），本轮未实现，试验文件已删除。
