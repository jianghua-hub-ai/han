# identity-auth-S4.19 独立 Code Review 报告（R03 / 收窄后复审）

**最终结论：通过（PASS — Code Review Gate R03）**

- 授权来源：用户 2026-10-31 项目级授权 + `ADR-0028`（`.add/project/knowledge/decisions/framework.md:815-856`）。
- 代理主体：`add-reviewer` 角色 / AI 代理 Reviewer（Host 未返回独立 Agent ID，记为 `unknown / 本会话`）。
- 代理范围：identity-auth-S4.19 R03 收窄后只读复审、允许命令的实际执行与证据报告；不含实施、不含 Git 写、不含签收生效。
- 声明：**AI 代理评审，非真人签字**。本结论不代替 Lead Gate R03、任务卡关闭或人的业务签收。

---

## 0. 复审摘要

- `S419-CR-F001`（`S419-CR-F001-R02`，MEDIUM，测试名夸大完整链）：**CLOSED**。子测试已改名 `PublicReadyAdmitsSameSession`（注册行 `:29`、函数 `:157`），注释 `:167-173` 明确引用正式收窄决定并写明“no single test links both layers”；正式收窄决定、任务卡 leaf `:2841`、关闭条件 `:2844`、B13H 计划 `:1593` 四者措辞一致，且 B13H Gate 记录“两层缺一不通过”。收窄后 `TestRecoveryRuntimePostgres` 本轮真实通过（真实 PG，全链），`TestRecoveryMainBusinessDispatcher/PublicReadyAdmitsSameSession` 断言未弱化。
- `S419-CR-F002`：**保持 CLOSED**（本轮实跑 `present=false … handle_nil=true`）。
- N1：**保持 CLOSED**（closed 下 `/livez` 200 后紧接业务 503，acquire=1）。
- 无新增阻断 finding。写集仍为两文件，`main.go` 未改，无第三代码写文件，无 Git 写/删除/归档。
- 边界：`TestRecoveryRuntimeProductionChain`（`127.0.0.1:53519`）端口不可达，**NOT RUN**，不作通过证据（§8）。

## 1. 固定对象与修订

| 项目 | 值 |
|---|---|
| 仓库 | `D:\AID\VSCode\Codex\Han` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| staged | 0（`git diff --cached` 空） |
| 评审角色 | `add-reviewer`（非作者独立复审，未参与实施） |
| 评审时间 | 2026-10-31（Asia/Shanghai） |
| 唯一写入目录 | `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r03/` |

| 评审对象 | SHA256（本次实算） | 字节 | mtime | 本轮状态 |
|---|---|---:|---|---|
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `F761D7E4FE4FCE03C50A3916228EA97243C7730BD2C1BE6EAEA958C8B73AAFD7` | 20376 | 2026-10-31T14:35:59.1469739+08:00 | 本轮唯一修改的代码文件 |
| `cmd/han-server/main.go` | `B012EAE8814BD47B16797A277AAB2E0CDA7C1885C839A8EE9D5435D63E3F03B8` | 171424 | 2026-10-31T10:27:35.8807765+08:00 | 未修改，与 R01/R02 一致 |

- 两对象 SHA 与委派输入一致，`main.go` 字节/哈希未变（`01-objects.log`）。评审开始与报告写入前两次复算一致。
- `cmd/`、`internal/`、`test/` 整体未纳入当前 HEAD 的 Git 索引，无 commit diff 基线；写集归因依赖 SHA + mtime + 全目录清单（§7）。

## 2. 检查范围

### Round 1：针对上轮 finding 的收窄修复复审

- 全文阅读 556 行测试文件，核对改名与注释：`TestRecoveryMainBusinessDispatcher` 注册 `PublicReadyAdmitsSameSession`（`:29`）、函数 `testRecoveryMainPublicReadyAdmitsSameSession`（`:157`）、分层证据注释（`:167-173`）。
- 对照正向断言（`:162-190`）与失败负控（`:201-262`）确认未被弱化。
- 对照正式收窄决定 `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-lead-narrowing-decision-r01.md` 与任务卡 `:2841`/`:2844`、B13H 计划 `:1593`。

### Round 2：上游依据与真实执行

- 上游验收依据：任务卡 `.add/project/knowledge/tasks/tasks.md`（S4.19 卡 `:2802` 起，leaf 表 `:2838-2842`，关闭条件 `:2844`，R02/R03 记录 `:2849`）、ADR-0028、B13H 计划 `ir-01-content-community-batches.md:1582-1593`。
- 只读依赖：`cmd/han-server/recovery_config.go:88-122`（`planRecoveryStartup`/`recoveryRootConfigurationPresent`）、`internal/platform/recovery_guard.go`（Protect/Release/Liveness/Readiness）。
- 真实执行：`TestRecoveryMainBusinessDispatcher`（普通/`-race`）、`go test ./cmd/han-server`、`TestRecoveryRuntimePostgres`（真实 PG `127.0.0.1:5433`）、`gofmt`/`build`/`vet`（§6）。

### 未执行的范围

- 未运行 `TestRecoveryRuntimeProductionChain`（53519 不可达）→ NOT RUN。
- 未启动真实 HTTP 服务、部署、Restore、SMTP、E2E；未连接真实 provider/CAS/ACL/POSIX。
- 未修改任何被审对象、任务卡、计划、ADR 或他人证据；未做 Git 写。

## 3. 逐条复核结论

### 3.1 子测试改名与注释是否消除“单测试证明完整链”的夸大 —— 是

- 测试名 `OpenAfterCompleteSoftwareRecoverySameSession` → `PublicReadyAdmitsSameSession`（注册 `:29`、函数 `:157`），不再主张“完整软件恢复后”。
- 注释 `:167-173` 明确：引用收窄决定文件；`TestRecoveryRuntimePostgres` 以真实 runtime + 真实 PG 驱动 `Begin → Consume → succeeded → Exit/join → activate` 且“required companion run at the same revision”；本 leaf 只证明共享 runtime 进入 `public_ready` 后 production dispatcher 放行同一 Session；**“no single test links both layers”**。
- 一致性核对：收窄决定 §1.2、任务卡 leaf `:2841`、关闭条件 `:2844`、B13H 计划 `:1593` 与测试注释四者措辞一致，均区分“层 a 真实全链/层 b 生产 dispatcher 同 Session”，并声明无单测试跨层。**无残留夸大。**
- 全文无 `OpenAfterCompleteSoftwareRecovery` 残留引用（`13-static-audit.log`）。

### 3.2 TestRecoveryRuntimePostgres 是否被明确为同修订必跑伴随证据，且 Gate 写“两层缺一不通过” —— 是

- 收窄决定 §1.2/§3 明确其为 S4.19 同修订必跑伴随证据、不得单独跳过。
- 任务卡 leaf `:2841`：`TestRecoveryRuntimePostgres`… “是该 leaf 的必跑伴随证据”；关闭条件 `:2844`：“收窄后的两层证据（真实 PG 全链 + public_ready 后同 Session 200）均按上表闭合”。
- B13H 计划 `:1593`：“本批 Gate 义务：S4.19 关闭核证时必须同修订同跑 `TestRecoveryRuntimePostgres`（真实 PG）与 `TestRecoveryMainBusinessDispatcher`，**两层缺一不通过**；不得把分层证据相加声称为单测试完整链。”
- 本轮实跑 `TestRecoveryRuntimePostgres` 通过，输出含 `real runtime.Enter consumed window … observation=public_ready` 与 `three worker join complete: workers=3 holders=0 … acquire_after=ErrAdmissionClosed`，证明为真实全链而非空跑（§6）。

### 3.3 F002 与 N1 是否保持 CLOSED；F001-R02 是否因正式收窄可关闭 —— 三者均 CLOSED

- **F002 CLOSED（保持）**：`testRecoveryMainNoRecoveryConfig`（`:304-340`）真实调用 `planRecoveryStartup(lookup, nil)`（`:307`），断言 `present=false/action=""/verify=false/maintenance=false`（`:311`），`present=false` 时 handle 保持 nil（`:314-318`），并在无配置 dispatcher 覆盖 `/livez` 200、`/readyz` 503、业务跳过 guard、404、405+`Allow`。本轮实跑 `10-norecovery-verbose.log`（exit 0）打印 `recovery_main_dispatcher_integration_test.go:339: planRecoveryStartup present=false action="" verify=false maintenance=false handle_nil=true`。
- **N1 CLOSED（保持）**：`LiveReadyProbes`（`:110-155`）closed 下 `/livez` 200（`:114`）后紧接业务 GET 503（`:118`），断言 `acquire=1`（`:119-121`）。
- **F001-R02 CLOSED（因正式收窄）**：R02 的最小修复路径 2 要求“由 Lead/设计 owner 对任务卡 leaf 正式收窄并落书面记录”。现：
  - 正式收窄决定已存在（`s419-lead-narrowing-decision-r01.md`），记录授权来源、决定主体（AI 代理，非真人签字）、原/新措辞、明确“不存在任何单测试跨层串联”“不得声称单测试证明完整链”。
  - 任务卡 leaf `:2841` 已按决定改写为 `PublicReadyAdmitsSameSession` 两层证据，关闭条件 `:2844` 同步；R02/R03 记录写入 `:2849`。
  - 测试改名且注释如实；真实 PG 伴随证据本轮通过。
  - 据此，R02 指出的“测试名夸大/把未验证写成 PASS”已被消除，F001-R02 在收窄口径下 **CLOSED**。残余“无单测试跨层”风险显式登记，由 B13H Gate 两层同跑承担（§8）。

### 3.4 写集是否仍为两文件、main.go 未改、无第三代码文件、无 Git 写 —— 是

- `cmd/han-server` 内 13:00 之后仅 `recovery_main_dispatcher_integration_test.go` 更新（14:35:59）；`main.go` 停留 10:27:35 且 SHA 未变；`internal/platform` 最新仍是 `recovery_guard.go`（10-29）与 `recovery_guard_test.go`（10-29）。
- `git status --porcelain` 282 行（与 R01/R02 基线一致）；`staged=0`；HEAD 未变。未发现第三代码写文件、越界或删除/移动/归档。

### 3.5 新 SHA 下断言是否真实可执行且未弱化 —— 是

- closed 503（`:162`）、public_ready 200（`:175`）、Trace 非空（`:176`）、`repository.lists==1`（`:179`）、resolver 调用 1 次（`:182`）、`acquire/release==2/1`（`:185`）、同 Session cookie 保持（`:188`）全部保留。
- 失败负控 `FailureNegativesNotMasked`（`:201-262`）：session/origin/csrf 分别断言 401/403、错误码、事件序列、Article 下游零调用、无 recovery 泄漏，均保留。
- `AdmissionIsNotSticky`（`:192-199`）重新 closed 后 `/readyz` 503、业务 503、acquire=3，保留。
- 无 `t.Skip`、无 `http.ServeMux`/`NewServeMux` fixture 代证（`13-static-audit.log`）。生产 `recovery.NewRuntime` 仍仅 `recovery_runtime.go:528` 一处（无第二 runtime）。

## 4. Findings

| Finding | 严重级 | 状态 | 说明 |
|---|---|---|---|
| `S419-CR-F001`（`S419-CR-F001-R02`） | MEDIUM | **CLOSED** | 正式收窄 + 改名 + 注释 + 双层证据同修订必跑 + B13H Gate“缺一不通过”，夸大消除；真实 PG 伴随证据本轮通过。 |
| `S419-CR-F002` | MEDIUM | **CLOSED（保持）** | 真实 `planRecoveryStartup`、`present=false`、handle nil、无配置 dispatcher 全覆盖；本轮实跑支撑。 |
| N1 | 非阻断 | **CLOSED（保持）** | closed 下 `/livez` 200→业务 503、acquire=1。 |

- 无新增阻断或非阻断 finding（R03）。

## 5. 非阻断观察

- **N2（unknown 保真度，未回归）**：`setState` 仍以 `ErrAdmissionClosed` 表示所有非 public_ready 状态；guard 对错误类型不敏感，行为等价，非阻断。
- **N3（SingleRuntimeHandle 间接证据，未回归）**：由 fake provider 证明 dispatcher 使用同一 handle；`assembly.runtime`/`composition.runtime` 同一性由 `recovery_runtime_test.go:719-769` 与静态路径补强。
- **计数口径说明（非缺陷）**：作者“17 pass”与本次 `pass_all=17` 一致；其中含 1 个 package 级 pass 事件，测试级 pass 事件为 16（父测试 + 5 子测试组 + 2 子分组 + 3 负控 + 2 内层，见 `06-integration.log`）；fail=0、skip=0。PG 侧同理 `pass_all=7`（6 测试级 + 1 package）。
- **作者 `.exit` 命名差异（非缺陷）**：作者 R03 目录部分 `.exit` 为 `*.log.exit`/`*.exit` 混用且内容为 `0`；不影响其命令真实性（本轮已全部独立复跑）。

## 6. 命令与退出码

所有命令在 `D:\AID\VSCode\Codex\Han` 根目录用 PowerShell 原生调用执行；原始日志与 `.exit` 位于本报告同目录。

| 序号 | 命令 | exit | pass/fail/skip | 日志 |
|---|---|---:|---|---|
| 01 | SHA256/字节/mtime/branch/HEAD/staged/status 复算 | 0 | — | `01-objects.log` / `.exit` |
| 02 | `gofmt -l cmd/han-server/main.go cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | 无输出（0 字节） | `02-gofmt.log` / `.exit` |
| 03 | `go build ./...` | 0 | 无输出 | `03-build.log` / `.exit` |
| 04 | `go vet ./cmd/han-server` | 0 | 无输出 | `04-vet.log` / `.exit` |
| 05 | `go vet -tags=integration ./cmd/han-server` | 0 | 无输出 | `05-vet-integration.log` / `.exit` |
| 06 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | pass=17（16 测试级 + 1 package）/ fail=0 / skip=0 | `06-integration.log` / `.exit` |
| 07 | 同 06 + `-race` | 0 | pass=17 / fail=0 / skip=0；DATA RACE=0 | `07-integration-race.log` / `.exit` |
| 08 | `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 38.486s` | `08-unit.log` / `.exit` |
| 09 | `$env:HAN_TEST_POSTGRES_DSN=...5433...; go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | pass=7（6 测试级 + 1 package）/ fail=0 / skip=0 | `09-postgres.json` / `.exit` |
| 10 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$/^NoRecoveryConfig$' -count=1 -v` | 0 | PASS；`present=false … handle_nil=true`（`:339`） | `10-norecovery-verbose.log` / `.exit` |
| 11 | 拓扑端口探测（5433/53519） | 0 | 5433=True；53519=False | `11-topology.log` / `.exit` |
| 12 | 写集清单（mtime/`internal/platform`/status/HEAD） | 0 | 仅目标测试文件更新 | `12-writeset-inventory.log` / `.exit` |
| 13 | 静态审计（`t.Skip`/`ServeMux`/`OpenAfterComplete`/`NewRuntime`/改名引用） | 0 | 无 skip/ServeMux/旧名；单一生产 `NewRuntime` | `13-static-audit.log` / `.exit` |

## 7. 写集与安全审计

- **SHA 冻结**：`F761…AAFD7`（20376B，14:35:59）与 `B012…03B8`（171424B，10:27:35）评审前后一致；`main.go` 未改。
- **写集**：本轮唯一修改的代码文件为 `recovery_main_dispatcher_integration_test.go`；`cmd/han-server` 其余文件与 `internal/platform` mtime 早于本轮窗口；无第三代码写文件。
- **越界/删除/Git**：未写 `cmd/`、`internal/`、`test/`、`.add/`、`.codex/`；未 Git add/commit/push/切分支/回滚；未删除/移动/归档；`staged=0`，HEAD 未变。
- **本评审写入**：仅本报告目录下 `report.md` 与 01–13 原始日志/`.exit`。
- **敏感信息/硬编码/第二 authority**：测试用 `admin-session-token` 等合成值；`/livez`、`/readyz` 为合同固定字面量；`13-static-audit.log` 显示生产 `recovery.NewRuntime` 仅一处，无第二 runtime/router/authority/fallback。

## 8. 残余风险与未验证边界

- **无单测试跨层串联**（收窄后已知边界）：若真实恢复链在 join/final proof/open 许可回归，而层 b 定向测试未与层 a 同跑，S4.19 定向测试不会失败；由 B13H Gate 两层同修订同跑（缺一不通过）承担。
- `TestRecoveryRuntimeProductionChain`（`cmd/han-server/recovery_runtime_integration_test.go`，硬绑 `127.0.0.1:53519`）：端口不可达，**NOT RUN**，不作任何通过证据。
- 未运行真实 HTTP 服务、部署、Restore、SMTP、E2E；未验证真实 POSIX/CAS/ACL/备份/provider；未证明 `canonical_recovery_replay_same_result`。
- 两对象未纳入 Git，变更归因依赖 SHA+mtime+清单（弱证据）。
- 本次 PASS 为 Code Review Gate R03 结论，不代替 Lead Gate R03 同 revision 核证、任务卡关闭或人的业务签收。

## 9. 最终结论

**通过（PASS — Code Review Gate R03）。**

- `S419-CR-F001`（`S419-CR-F001-R02`）：**CLOSED**（收窄口径）。
- `S419-CR-F002`：**CLOSED（保持）**。
- N1：**CLOSED（保持）**。
- 无新增 finding。写集两文件、`main.go` 未改、无 Git 写；新 SHA 下断言完整可执行且未弱化，真实 PG 伴随证据本轮通过。残余风险与 53519 NOT RUN 边界已登记。

上述为 **AI 代理评审，非真人签字**；结论不代替 Lead 的接受与后续授权。收窄决定 §4 的“独立复审 R03 确认一致”条件已满足；最终关闭仍待 Lead Gate R03 同修订核证。