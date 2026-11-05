# 集成套件实跑地图（2026-11-05）

命令：`go test -tags=integration ./... -count=1 -timeout 30m`
环境：`HAN_TEST_POSTGRES_DSN` → `postgres@127.0.0.1:5433/postgres`（本地通用实例）

## 结果

| 包 | 结果 | 说明 |
|---|---|---|
| `han/internal/audit` | ok 3.975s | — |
| `han/internal/idempotency` | ok 10.123s | — |
| `han/internal/mailer` | ok 8.713s | — |
| `han/internal/migration` | ok 24.772s | — |
| `han/internal/platform` | ok 0.300s | — |
| `han/internal/privacy` | ok 0.415s | — |
| `han/internal/recovery` | ok 4.662s | — |
| `han/internal/security` | ok 0.538s | — |
| `han/internal/session` | ok 0.319s | — |
| `han/internal/testenv` | ok 0.526s | — |
| `han/test/integration/crossmodule` | ok 130.432s | 真实 PG |
| `han/test/integration/recovery` | ok 84.112s | 真实 PG |
| `han/cmd/han-server` | FAIL 85.405s | 3 项需专用实例 `han_b13h@127.0.0.1:53519`，连接被拒绝 |
| `han/internal/article` | FAIL 106.351s | 仅 `TestAnchoredArticleFinalizerPostgres`：要求 `han_b13c_article`/`han_b13c`；同包其余（含 `TestReplayPublicationPostgres`）通过 |
| `han/internal/identity` | FAIL 19.286s | 8 项报 `explicit isolated identity PostgreSQL configuration required`，需专用实例配置；s413 系列另有 `han_b13e@51935` 与固定数据目录要求 |
| `han/test/integration/article` | FAIL 280.997s | 3 项 perf 需 `HAN_PG_PERF_EVIDENCE_DIR` 指向 fresh continuation 目录 |
| `han/test/integration/identity` | FAIL 317.268s | `TestS33LoginAbuseChildProbe` 需浏览器/专用实例门禁 |

## 结论

1. 失败全部来自**本机缺失的专用测试实例或必填环境变量**，无一项是代码回归；失败信息逐条包含实例名、端口或环境变量名。
2. 12 个包实跑通过，其中 `crossmodule`（130s）与 `test/integration/recovery`（84s）是真实 PG 上的跨模块与恢复链路，之前未在本机验证过。
3. 今日 4 处测试脚手架改动的覆盖：`s414Directory` 真实跑通（+0 残留）、`anchorCoreFixture` 单测跑通（+0 残留）、replay 两处真实 PG 跑通（+0 残留）、`s413Checkpoint` 因 `han_b13e@51935` 实例下线只能编译级覆盖。

## 复跑建议

- 全套：`go test -tags=integration ./... -count=1 -timeout 30m`
- 仅验证今日改动：`go test -tags=integration ./internal/article -run '^TestReplayPublicationPostgres$' -count=1`
- 专用实例缺失时不要把这些 FAIL 读成回归；逐条看失败信息里的实例名/端口/环境变量。

## race 与静态检查核实（同日）

- `go test -race ./internal/session -count=1` → `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`（exit 2）。
- 环境：`go env CGO_ENABLED` = `0`；`CC` = `gcc` 但 PATH 上无 `gcc`/`clang`，无 C 工具链。**结论：`-race` 在本机不可用，属工具链缺失，不是测试失败**；启用需安装 C 工具链并设 `CGO_ENABLED=1`，属环境变更，未做。
- `go vet -tags=integration ./...` → exit 0（全仓，含集成构建约束）。
- `gofmt -l cmd internal test` → 0 个文件。

## 离线包自足性验证（同日）

- `git clone --no-hardlinks .spec/wip/transfer/han-ir01-20261105.bundle Temp/bundle-check-20261105` → 成功；检出 **1,814** 个跟踪文件（与源仓 `git ls-files` 计数一致），HEAD `0a4b8e3`，历史 **24** 个提交。
- 克隆内 `go build ./...` → exit 0。
- 克隆内 `go test ./... -count=1` → exit 0（无 FAIL）。
- 克隆内 `go test -tags=contract ./... -count=1` → exit 0（无 FAIL）。
- 结论：离线包可独立复现工作树并完成构建与测试，未发现因 `.gitignore` 误排除而缺失的文件。验证用克隆已删除，源仓与包未受影响。

## 被挡住的测试：落地条件核查（同日）

`5433` 实例（`127.0.0.1`）上，测试所需的对象**已存在**：

- 角色：`han_b13c`、`han_b13d`、`han_b13e`
- 数据库：`han_b13c_article`、`han_b13c_identity`、`han_b13d_identity`、`han_b13e_identity`
- 实测 `postgres://han_b13c:probe_only@127.0.0.1:5433/han_b13c_article` → `Password 认证失败`，即口令未知（scram），不是对象缺失
- 因此 `TestAnchoredArticleFinalizerPostgres` 只缺一个可用的 `han_b13c` DSN；提供后即可实跑，无需新建任何对象

另两个专用端口当前**关闭**：`51935`（`han_b13e`，s413 系列）、`53519`（`han_b13h`，cmd/han-server 恢复链三项）→ 需先启动对应实例，属环境变更。

结论：集成套件的 5 个失败包中，1 个只缺口令（可立即解锁），2 个缺专用实例（需启动），1 个缺 `HAN_PG_PERF_EVIDENCE_DIR`（需指定目录），1 个缺浏览器/实例门禁。

### 同日补充：A47 已用临时集群实跑通过

- 在 `Temp/pg-verify-20261105` 用 `initdb`（PostgreSQL 18.6、trust、UTF8）起临时集群于 `127.0.0.1:55432`，建 `han_b13c` 角色与 `han_b13c_article` 库；**未改动 5433 实例上的同名对象**。
- DSN：`postgres://han_b13c:synthetic@127.0.0.1:55432/han_b13c_article?sslmode=disable`。注意默认 `sslmode=prefer` 会因临时集群未启 SSL 报 `tls error: server refused TLS connection`，必须显式 `disable`。
- `go test -tags=integration ./internal/article -run '^TestAnchoredArticleFinalizerPostgres$' -count=1` → **ok 8.434s**。
- 收尾：`pg_ctl stop -m fast` 成功，临时数据目录已删除（`Test-Path` 为 False）。
- 仍被挡：s413 系列要求固定端口 `51935` 与固定数据目录字符串 `d:/temp/han-b13e-lead-20261030-r01/postgres/data`；cmd/han-server 恢复链要求 `53519` 上的 `han_b13h`。重建这类“按旧会话字形匹配”的实例没有验证价值，未做。

### 同日补充：B13E fixture 仍在磁盘，s413 已实跑通过

- `D:\temp\han-b13e-lead-20261030-r01\postgres\data` 仍存在（PG 18），`pg_hba.conf` 对 `127.0.0.1` 为 `trust`，不需要口令；只是当时未运行。
- 以 `pg_ctl -D <data> -o "-p 51935"` 启动后核对：`han_b13e` / `han_b13e_identity` / `180006` / `D:/temp/han-b13e-lead-20261030-r01/postgres/data`，与 s413 准入条件逐项一致。
- `HAN_TEST_POSTGRES_DSN=postgres://han_b13e@127.0.0.1:51935/han_b13e_identity?sslmode=disable`：
  - `go test -tags=integration ./internal/identity -run '^TestRecoveryInstallationPostgres$' -count=1` → **ok 11.621s**（覆盖本次改动的 `s413Checkpoint`）。
  - 同包整包运行：本测试通过；另有 7 项要求 `han_b13c` / `han_b13c_identity`（5433 上存在该库但口令未知，可用一次性集群复现）。
- 收尾：`pg_ctl stop -m fast`，恢复运行前状态（`51935` 已不再监听）。

### 同日补充：identity 包此前 8 项失败已全部实跑通过

- `TestRecoveryInstallationPostgres`（s413）：用既有 B13E fixture（`51935`）跑通，`ok 11.621s`。
- 其余 7 项用一次性集群 `Temp/pg-verify-b13c`（`initdb` PG 18.6、端口 `55433`、trust）实跑通过：

| 测试 | 需要 | 结果 |
|---|---|---|
| `TestRecoveryAdmissionStorePostgres` | `han_b13c` / `han_b13c_identity` | ok |
| `TestRecoveryRunStoreMainlineUTCAndJournalVerification` | 同上 | ok |
| `TestRecoveryRunStoreRejectsDirectIdentityReadyDuplicateAndTriggerBypass` | 同上 | ok |
| `TestRecoveryRunStoreConcurrentTransitionCAS` | 同上 | ok |
| `TestRecoveryMaintenanceAuthenticationPostgres` | `HAN_B13E_S411_PG_CONFIG` 信号 + `han_b13e` / `han_b13e_identity` | ok |
| `TestRecoveryRestoredPasswordResetPostgres` | 同上 | ok |
| `TestRecoveryIdentityExecutionPostgres` | `HAN_TEST_POSTGRES_DSN` → `han_b13d_identity` / `han_b13d` | **ok 68.422s** |

信号文件格式（`HAN_B13E_S411_PG_CONFIG`）：`Purpose=B13E-S411`、`Host=127.0.0.1`、`Database=han_b13e_identity`、`User=han_b13e`、非零 `Port`、`ServerVersion>=180000`；DSN 必须带 `sslmode=disable`；文件路径必须是 `d:/temp/` 下的绝对路径。

收尾：`pg_ctl stop -m fast`，删除 `Temp/pg-verify-b13c` 与 `D:\temp\han-b13e-verify`；既有 `D:\temp\han-b13e-lead-20261030-r01` 未改动。

### 同日补充：cmd/han-server 集成包由 3 项失败降为 1 项

- `TestRecoveryRuntimeProductionChain`、`TestRecoveryMaintenanceAuthorizerScope`、`TestRecoveryControlAuthorizerSealedMaintenance`：一次性集群 `Temp/pg-verify-b13h`（`127.0.0.1:53519`，`han_b13h` 角色 + `han_b13h_root` / `han_b13h_backup` 两库）→ 三项同时跑 **ok 2.304s**。
- `TestRecoveryRestoreTargetReadbackPostgres`：一次性集群 `Temp/pg-verify-restore`（`127.0.0.1:51934`，`han_b13d` 角色 + `han_b13d_restore` 库）→ 通过；整包复跑时该项已不再失败。
- 剩余 1 项 `TestAppendOnlyApplicationACL`：断言 `requires literal loopback:5433 for primary and all fallbacks`，且会变更 ACL/角色；未在共享 5433 实例上执行（需明确授权）。
- 收尾：两集群 `pg_ctl stop -m fast` 后删除目录，`53519` 与 `51934` 均不再监听。

### 同日补充：cmd/han-server 全部集成测试实跑通过（须分两轮）

- `TestAppendOnlyApplicationACL` 要求 `HAN_TEST_POSTGRES_DSN` 指向字面 `127.0.0.1:5433`；`TestRecoveryRestoreTargetReadbackPostgres` 要求同一变量指向 `127.0.0.1:51934/han_b13d_restore`。两者**不能在同一次运行中同时满足**，因此整包必须分两轮执行，这不是缺陷。
- 5433 轮：`TestAppendOnlyApplicationACL` → **ok 2.408s**。该测试按设计**永久保留**创建物，实测 `han_s15_acl_*` schema 54 → 56、`han_s15_*` 角色 62 → 64，与历史运行产物一致（未清理）。
- 51934 轮（一次性集群）：恢复链三项 + restore 读回通过；集群已删除。
- 结论：`cmd/han-server` 集成包在本机可分两轮全部跑通。

### 同日补充：最后两项“未运行”的真实性质

- `TestS33LoginAbuseChildProbe`：源码注释明确 `This top is NOT in the parent selector`，它是父测试经 `f.child(...)` 拉起的**子进程探针**；单独运行必然在 DSN 访问前失败（`helper mode required before DSN access`）。该包其余测试已通过，故不构成覆盖缺口。
- `TestPgPerf*`（3 项）：除 `HAN_PG_PERF_EVIDENCE_DIR` 外还要求 `HAN_PG_PERF_RUN_ID`、`HAN_PG_PERF_DATABASE`（默认 `han_perf_s24`）、`HAN_PG_PERF_WINDOW_START`、`HAN_PG_PERF_BASE_BYTES`、`HAN_PG_PERF_BASELINE_DIR`，且需 5433 上的 PG 18.6。性能结论依赖受控窗口与基线；人为拼凑这些条件只会产出不可信数字，**不跑**。

至此本机可运行的集成测试已全部实际执行；其余缺口均由 ADR-0036/0037 的范围决策或专用受控环境所致。

### 同日补充：README 记载命令的可执行性核对

- `go mod verify` → `all modules verified`（exit 0）。
- README 引用的 5 份文档全部存在：`docs/testing.md`、`docs/dependencies.md`、`docs/runbooks/database-recovery.md`、`docs/runbooks/article-deletion-recovery.md`、`docs/runbooks/identity-observability.md`。
- 无 `HAN_DATABASE_DSN` 时执行 `go run ./cmd/han-server`：2.5s 内退出，stderr 为 `han-server startup failed` + `exit status 1`，与 README 的 fail-closed 描述一致，未挂起。
- README 标注 `-race` 需要 CGO，本机实测确为 `-race requires cgo`，文档与事实一致。
