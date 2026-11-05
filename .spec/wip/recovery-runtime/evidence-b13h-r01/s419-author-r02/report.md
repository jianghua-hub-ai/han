# identity-auth-S4.19 作者 R02 证据报告（F001/F002 补证与 N1 修正）

- 报告性质：作者侧 R02 证据汇总（执行由 R02 作者在 2026-10-31 完成，本文件由 Lead 依据同目录原始日志整理）。
- 适用授权：用户 2026-10-31 项目级授权 + [ADR-0028](../../../../.add/project/knowledge/decisions/framework.md#adr-0028)。
- 结论：F002 已按原 finding 最小修复闭合；F001 无法在“两写文件、只读依赖不改”的写集内用单测试串联真实恢复链，按 ADR-0028 正式收窄为“生产 dispatcher 同 Session 观察”+“真实 Runtime/真实 PG 全链独立证据”，残余风险已登记，未声称单测试完整链 PASS。N1 已修正。

## 1. 固定对象

| 项目 | 值 |
|---|---|
| 仓库 | `D:\AID\VSCode\Codex\Han` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| staged | 0 |
| 执行时间 | 2026-10-31（Asia/Shanghai） |
| 本轮写入目录 | 本目录 `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-author-r02/` |

| 写集文件 | SHA256（复算） | 字节 | 本轮状态 |
|---|---|---|---|
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `66D2AB7A582B78018146CF5E976FE77FD683B4C63FC0FC72EC470D0DC887DC46` | 20300 | 本轮修改（唯一代码写文件） |
| `cmd/han-server/main.go` | `B012EAE8814BD47B16797A277AAB2E0CDA7C1885C839A8EE9D5435D63E3F03B8` | 171424 | 未修改，与 R01 评审对象一致 |

- 未新增第三写文件；未修改只读依赖 `cmd/han-server/recovery_runtime.go`、`internal/platform/recovery_guard.go`、`cmd/han-server/recovery_runtime_integration_test.go`、`cmd/han-server/main_test.go`。
- `cmd/`、`internal/`、`test/` 整体未纳入 Git 索引，无 commit diff 基线；写集归因依赖 §5 的 SHA + mtime + 全目录清单。
- 未执行 Git add/commit/push/切分支/回滚；未删除、移动或归档任何文件。

## 2. S419-CR-F002 关闭（无 recovery 配置 present:false）

位置：`cmd/han-server/recovery_main_dispatcher_integration_test.go:302-338`（`testRecoveryMainNoRecoveryConfig`）。

实际执行链：

1. `validCompositionEnvironment()` 构造不含任何 recovery data key 的环境（与 `main_test.go` 既有无配置用例同源）。
2. 真实调用 `planRecoveryStartup(lookup, nil)`，断言 `err == nil`。
3. 断言 `plan.present == false`、`plan.action == ""`、`plan.verify == false`、`plan.maintenance == false`、`recoveryRootConfigurationPresent(lookup) == false`。
4. 仅当 `plan.present` 为真才创建 runtime；实际 `present=false` 使 handle 保持 `nil`，dispatcher 通过 `func() recovery.Runtime { return handle }` 拿到 nil。
5. 断言 `/livez` 200 `alive`；`/readyz` 503 `unavailable`；普通业务 `GET /api/v1/admin/articles` 200 且响应不含 `platform.recovery_in_progress`（无 runtime 即跳过 guard）；未知路径 404；`POST /api/v1/admin/articles` 405 + `Allow: GET`。

实际日志（`13-norecovery-verbose.log`，`go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$/^NoRecoveryConfig$' -count=1 -v`，exit 0）：

```text
planRecoveryStartup present=false action="" verify=false maintenance=false handle_nil=true
```

原始证据：`13-norecovery-verbose.log`、`13-norecovery-verbose.exit`（exit 0，PASS）；全量单元套件另见 `07-unit.log` / `07-unit.exit`（exit 0，`ok han/cmd/han-server 37.683s`）。

对照 R01 finding：原测试直接注入 nil provider、未调用 `planRecoveryStartup`；本版已由 production 判定函数驱动，finding 指出的“空环境误判 present:true / 仍写 handle 不会被发现”场景现由 `present=false` 与 `handle_nil=true` 直接拦截。

## 3. S419-CR-F001 处理：可执行部分闭合 + ADR-0028 正式收窄

### 3.1 已在生产 dispatcher 上证明的部分

位置：`:157-261`（`testRecoveryMainOpenAfterCompleteSoftwareRecoverySameSession`）。

- closed 下用同一 `*http.Request` 请求 `GET /api/v1/admin/articles` → 503，acquire/release = 1/0。
- 辅助函数 `publishPublicReady`（`:509-515`）先核对 closed 阶段计数为 acquire=1、release=0，再切到 `public_ready`；不再裸切换。
- 同一请求对象、同一 Session cookie 再请求 → 200；断言 Trace 非空、repository `lists=1`、Session resolver 调用 1 次、acquire/release = 2/1、cookie 仍为 `admin-session-token`。
- 子测试 `AdmissionIsNotSticky`：重新 closed 后 `/readyz` 503、业务 GET 503、acquire 累计 3，证明放行不是粘性开关。
- 子测试 `FailureNegativesNotMasked`：session / origin / csrf 三类失败负控仍走 production 链返回 401/403，且 Article 下游零调用。

### 3.2 未在单测试中串联的部分与收窄依据

任务卡 `OpenAfterCompleteSoftwareRecoverySameSession` 原文要求 `Begin → … → succeeded → 撤维护/真 join → 最终 proof → public open` 后同 Session 200。该完整链需要：

- 真实 `recovery.Runtime` 与真实 PostgreSQL（窗口消费、join、final proof、activate）；
- 真实恢复测试夹具，当前只存在于 `test/integration/recovery` 的 `_test` 包内，Go 不允许从 `cmd/han-server` 导入外部测试包；
- 在 `cmd/han-server` 内重建等价夹具将超出本卡“两写文件、不得新增第三写文件、只读依赖不得修改”的写集，且会把 PG 依赖强加给每个执行该 integration 测试的环境。

因此按 ADR-0028“真实外部环境不可得时可用模拟环境替代并显式标注”的规则，采用**分层证据**，并在测试代码注释（`:167-171`）与任务卡关闭记录中显式声明“单测试跨层链未实现”：

| 证据层 | 载体 | 证明内容 |
|---|---|---|
| 真实恢复全链 | `TestRecoveryRuntimePostgres`（`test/integration/recovery`，真实 PG `127.0.0.1:5433`） | Begin → Consume → succeeded → Exit/join（三 worker，holders=0）→ activate → `public_ready` |
| 生产 dispatcher 同 Session | `testRecoveryMainOpenAfterCompleteSoftwareRecoverySameSession` | 同一 handler、同一 Session 在 closed→public_ready 前后分别 503→200，lease 计数与下游调用正确 |
| 交叉层缺口 | 本报告 §3.3 | 单测试未串联上表两层，作为已登记残余风险 |

真实链关键日志（`08-real-runtime-postgres.json`）：

```text
real runtime.Enter consumed window: ... outcome=succeeded stage=consumed consumed_marks=1 caller=coordinator.go:275 ... observation=public_ready
three worker join complete: workers=3 holders=0 lifecycle_stop=1 acquire_after=ErrAdmissionClosed
```

### 3.3 残余风险（不得隐藏）

- 若真实恢复链在 join、最终对账或 open 许可上回归，而分层测试未被同时执行，本卡定向测试不会失败；该风险由 B13H 批次 Gate 与 `TestRecoveryRuntimePostgres` 的独立执行共同承担，不由 S4.19 单测试承担。
- 本收窄是验收措辞收窄，不是生产行为变更；`main.go` 未修改，dispatcher 行为仍由 closed/public_ready 两侧真实断言覆盖。
- 收窄决定需由 R02 独立评审与 Lead Gate 确认后方可作为 S4.19 关闭依据；本条不代替评审结论。

## 4. N1 修正

位置：`:110-155`（`testRecoveryMainLiveReadyProbes`）。

- closed 下先 `/livez` 200，紧接着同一 runtime 的业务 GET 仍 503，并断言 acquire=1：证明 liveness 不留下业务授权，而不是“先改回 closed 再测”。
- unknown / maintenance / verify 三状态 `/readyz` 均 503 `unavailable`。
- 错 method `/livez`、`/readyz` 均 405 + `Allow: GET`；`/livezz`、`/livez/`、`/readyz/` 均 404。
- public_ready 下 `/readyz` 200 `ready` 后业务 GET 200，acquire/release = 2/1，且 readyz 确实调用共享 runtime（verifies > 0）。
- 最终重新 closed 后业务 GET 503，错误码 `platform.recovery_in_progress`。

## 5. 验证命令与结果

全部命令在仓库根目录执行；原始输出见同目录。

| 序号 | 命令 | exit | 结果 |
|---|---|---|---|
| 01 | `gofmt -l cmd/han-server/main.go cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | 无输出（`01-gofmt.log` 0 字节） |
| 02 | `go build ./...` | 0 | 无输出 |
| 03 | `go vet ./cmd/han-server` | 0 | 无输出 |
| 04 | `go vet -tags=integration ./cmd/han-server` | 0 | 无输出 |
| 05 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | pass=17 / fail=0 / skip=0 |
| 06 | 同 05 + `-race` | 0 | pass=17 / fail=0 / skip=0，0 DATA RACE |
| 07 | `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 37.683s` |
| 08 | `HAN_TEST_POSTGRES_DSN=postgres://postgres:<redacted>@127.0.0.1:5433/postgres?sslmode=disable go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | pass=7 / fail=0 / skip=0 |
| 09 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$/^NoRecoveryConfig$' -count=1 -v` | 0 | 输出 `present=false ... handle_nil=true`，PASS |

写集证据：`00-object.txt`、`00-status-before.txt`、`09-status-after.txt`、`10-writeset-inventory.txt`、`11-status-delta.txt`（5 字节，delta 仅本目录新增）、`12-gate-inputs.txt`。

## 6. 未验证边界

- 未启动真实 HTTP 服务、真实部署、Restore、SMTP、任何端到端演练。
- 未验证真实 POSIX/FileJournal、CAS、ACL、真实备份与 keyring/HMAC；Windows 环境无 FileJournal。
- 未证明 `canonical_recovery_replay_same_result`。
- `TestRecoveryRuntimePostgres` 使用本机 5433 的 PostgreSQL，属于真实 PG 但非生产拓扑。
- 本卡 PASS 不等于 `article-management-S8.2` / Case18 整卡关闭，也不等于人的业务签收。

## 7. 写集与安全声明

- 仅修改 `cmd/han-server/recovery_main_dispatcher_integration_test.go` 一个代码文件；`main.go` 未修改（SHA 与 R01 一致）。
- 无第三写文件；无删除/移动/归档；无 Git 写操作；`staged=0`，HEAD 未变。
- `cmd/han-server` 下其余文件 mtime 早于本轮窗口（测试文件 `2026-10-31T14:20:36`，`main.go` `10:27:35`）。
- 测试使用 `admin-session-token` 等合成值，断言覆盖 run/account/ref/secret 不泄漏；未引入真实凭据。

---

**2026-10-31 脱敏说明（Lead，按用户确认执行）**：按用户 2026-10-31 确认，本文件中原引用的本机 loopback 合成 DSN 口令已替换为 `<redacted>`（共 1 处）；其余内容与结论未变。脱敏前 SHA256 为 `4D2C6B7EFABFAB770BC9B33A642B20CCF6364FEE3FDE7CFE05AA11D9AA69DB0D`，脱敏后哈希见 tasks.md 同日记录。
