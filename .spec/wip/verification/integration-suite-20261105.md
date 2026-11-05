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
