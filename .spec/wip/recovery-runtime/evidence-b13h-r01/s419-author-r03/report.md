# identity-auth-S4.19 作者 R03 证据报告（正式收窄路径重出）

- 报告性质：作者侧 R03 证据汇总。R03 只做验收措辞收窄后的重出与重跑，不改变生产行为。
- 适用授权：用户 2026-10-31 项目级验证授权 + ADR-0028（`.add/project/knowledge/decisions/framework.md` ADR-0028）。
- 结论：测试子测试改名与注释已落地，`main.go` 未改；全部可执行命令 exit 0，定向普通/race 各 17 pass、真实 PG 7 pass、单元包 `ok 40.415s`。本报告只证明“分层证据”，不声称存在单测试跨层完整链。

## 1. 固定对象

| 项目 | 值 |
|---|---|
| 仓库 | `D:\AID\VSCode\Codex\Han` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| staged | 0 |
| 执行时间 | 2026-10-31（Asia/Shanghai） |
| 本轮写入目录 | `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-author-r03/` |

| 写集文件 | SHA256 | 字节 | 本轮状态 |
|---|---|---|---|
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `F761D7E4FE4FCE03C50A3916228EA97243C7730BD2C1BE6EAEA958C8B73AAFD7` | 20376 | 本轮修改（唯一代码写文件） |
| `cmd/han-server/main.go` | `B012EAE8814BD47B16797A277AAB2E0CDA7C1885C839A8EE9D5435D63E3F03B8` | 171424 | 未修改，与 R01/R02 审查对象一致 |

- 未新增第三写文件；未修改只读依赖 `cmd/han-server/recovery_runtime.go`、`internal/platform/recovery_guard.go`、`cmd/han-server/recovery_runtime_integration_test.go`、`cmd/han-server/main_test.go`。
- 未执行 Git add/commit/push/切分支/回滚；未删除、移动或归档任何文件。
- `09-status-after.txt` 与 `00-status-before.txt` 比较结果为 identical（`11-status-delta.txt`）。

## 2. 本轮代码改动（仅验收措辞）

位置：`cmd/han-server/recovery_main_dispatcher_integration_test.go`。

1. 子测试名 `OpenAfterCompleteSoftwareRecoverySameSession` → `PublicReadyAdmitsSameSession`。
2. 对应测试函数 `testRecoveryMainOpenAfterCompleteSoftwareRecoverySameSession` → `testRecoveryMainPublicReadyAdmitsSameSession`。
3. 原地注释改为分层证据说明，显式引用收窄决定 `s419-lead-narrowing-decision-r01.md`，并写明：`TestRecoveryRuntimePostgres` 以真实 runtime + 真实 PostgreSQL 驱动 Begin → Consume → succeeded → Exit/join → activate，是同修订必跑伴随证据；本 leaf 只证明共享 runtime 进入 `public_ready` 后 production dispatcher 放行同一 Session；没有任何单测试跨层串联两层。

生产代码 `main.go` 未改；dispatcher 行为仍由 closed 503 与 public_ready 200 两侧真实断言覆盖。

## 3. 验证命令与结果

全部命令在仓库根目录执行；原始输出与 `.exit` 见同目录。

| 序号 | 命令 | exit | 结果 |
|---|---|---|---|
| 01 | `gofmt -l cmd/han-server/main.go cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | 无输出（`01-gofmt.log` 0 字节） |
| 02 | `go build ./...` | 0 | 无输出 |
| 03 | `go vet ./cmd/han-server` | 0 | 无输出 |
| 04 | `go vet -tags=integration ./cmd/han-server` | 0 | 无输出 |
| 05 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | pass=17 / fail=0 / skip=0 |
| 06 | 同 05 加 `-race` | 0 | pass=17 / fail=0 / skip=0，无 DATA RACE |
| 07 | `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 40.415s` |
| 08 | `HAN_TEST_POSTGRES_DSN=postgres://postgres:<redacted>@127.0.0.1:5433/postgres?sslmode=disable go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | pass=7 / fail=0 / skip=0 |
| 09 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$/^NoRecoveryConfig$' -count=1 -v` | 0 | `present=false ... handle_nil=true`，PASS |

`13-norecovery-verbose.log` 关键行：

```text
=== RUN   TestRecoveryMainBusinessDispatcher/NoRecoveryConfig
recovery_main_dispatcher_integration_test.go:339: planRecoveryStartup present=false action="" verify=false maintenance=false handle_nil=true
--- PASS: TestRecoveryMainBusinessDispatcher/NoRecoveryConfig (0.05s)
```

过程说明（不掩盖）：05/06/13 首轮用 `cmd /c` 传参时，cmd 不解析单引号，Go 收到带引号的 `-run` 模式并输出 `no tests to run`。该三份输出已作废并覆盖；上表与现存文件均为改用 PowerShell 原生传参后的有效结果，均真实执行了目标测试。

## 4. 分层证据对照

| 证据层 | 载体 | 证明内容 | 本轮状态 |
|---|---|---|---|
| 真实恢复全链 | `TestRecoveryRuntimePostgres`（真实 PG `127.0.0.1:5433`） | Begin → Consume → succeeded → Exit/join（三 worker，holders=0）→ activate → `public_ready` | pass=7，真实验证 |
| 生产 dispatcher 同 Session | `TestRecoveryMainBusinessDispatcher/PublicReadyAdmitsSameSession` | 同一 handler、同一 Session：closed 503（acquire=1 release=0）→ 共享 runtime `public_ready` → 同一请求 200，Trace 保留，repository lists=1，Session resolver 调用 1 次，acquire/release=2/1 | pass（含于 17） |
| 跨层缺环 | 本报告 §5 | 无单测试串联两层 | 已登记残余风险 |

## 5. 残余风险与边界

- 若真实恢复链在 join、最终 proof 或 open 许可上回归，而分层定向测试未同跑，本 leaf 不会失败；该风险由 B13H 批 Gate 强制同修订同跑 `TestRecoveryRuntimePostgres` 承担，不由 S4.19 单测承担。
- `TestRecoveryRuntimeProductionChain`（`cmd/han-server/recovery_runtime_integration_test.go`，硬绑 `127.0.0.1:53519`）本轮未运行；端口不可达，状态记为 NOT RUN，不作任何通过证据。
- 本收窄只改验收措辞，不是生产行为变更；不证明真实 HTTP 部署、真实 POSIX/CAS/ACL、批准备份或 provider。
- 本报告是作者证据，不等于独立评审或 Lead Gate 结论。

## 6. 证据文件清单

`00-object.txt`、`00-status-before.txt`、`01-gofmt.log/.exit`、`02-build.log/.exit`、`03-vet.log/.exit`、`04-vet-integration.log/.exit`、`05-integration.json/.exit`、`06-integration-race.json/.exit`、`07-unit.log/.exit`、`08-real-runtime-postgres.json/.exit`、`09-status-after.txt`、`10-writeset-inventory.txt`、`11-status-delta.txt`、`12-gate-inputs.txt`、`13-norecovery-verbose.log/.exit`。

---

**2026-10-31 脱敏说明（Lead，按用户确认执行）**：按用户 2026-10-31 确认，本文件中原引用的本机 loopback 合成 DSN 口令已替换为 `<redacted>`（共 1 处）；其余内容与结论未变。脱敏前 SHA256 为 `27179A39E2330DDD73DB614E95FE506DBF1668F8D93F5871C952C9537BA3C510`，脱敏后哈希见 tasks.md 同日记录。
