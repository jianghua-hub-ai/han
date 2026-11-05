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
