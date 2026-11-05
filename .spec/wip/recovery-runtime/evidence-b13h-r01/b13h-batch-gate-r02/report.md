# B13H 批次 Gate 重跑 r02 — 独立评审报告

- 评审角色：非作者独立 Reviewer（add-reviewer / add-code-review），不是作者、不是 Lead。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮实核）。
- 评审日期：2026-10-31（Asia/Shanghai）。
- 评审对象：B13H 唯一 root 批次（identity-auth-S4.16 / S4.19 / S4.20），17 唯一文件 / 21 声明槽。
- 结论：**PASS WITH MINOR**。无 HIGH/MEDIUM；`B13H-GATE-F001`、`N01`、`B13H-GATE-F002` 均 CLOSED；4 条 LOW 非阻断（r04 两条按已披露登记 + 本轮新增 2 条观察）。
- 写集：仅本目录 `report.md`、`verdict.md`、`commands.log` 与原始 JSON/日志；未改源码、测试、任务卡、Plan 或既有证据。
- 独立性声明：未转述 Gate r01、r04 或 Lead 结论；17 文件哈希、两层测试与静态检查均在本轮重新复算/复跑。

## 1. 实读输入

- 批次合同：`.add/project/knowledge/plans/ir-01-content-community-batches.md:1587-1602`（B13H 段）与外层表 `:1309`、文件槽位表 `:1329-1332`。
- 任务卡：`tasks.md:2659-2682`（S4.16）、`:2820-2869`（S4.19）、`:2873-2951`（S4.20），含 `:2859` 层 (a)/(b)、`:2862` 关闭条件、`:2868` 两层缺一不通过、`:2933-2935` S4.20 的 S419Reverified 条件、`:2949` S4.20 关闭记录、`:2950` F002 更正、`:2951` 返工基线。
- 方法：`.codex/skills/add-code-review/SKILL.md`、`.codex/guidance/task-execution-workflow.md`（§7/§10）、`.codex/guidance/multi-agent-coordination.md`、`.codex/guidance/coding-standards.md`、`.codex/guidance/code-review-checklist.md`。
- 代码：`cmd/han-server/main.go`（recovery handle/assembly/dispatcher/probes）、`cmd/han-server/recovery_runtime.go`（root assembly、唯一 NewRuntime/Stores）、`cmd/han-server/recovery_main_dispatcher_integration_test.go`（层 (b) 全文）、`cmd/han-server/recovery_workers_test.go:148-156`（`newOrdinaryFixture` → 真实 `recovery.NewRuntime`）。
- 既有证据：`b13h-batch-gate-r01/{report.md,verdict.md,commands.log}`、`s419-independent-review-r04/`、`s419-author-r04/report.md`、`s419-code-review-r03/`、`s419-lead-gate-r03/`、`s420-code-review-r01/{report-r02.md,verdict-r02.md,lead-recheck-r02.md}`、`s420-implementation-r01/d7-*.json`、`d7-tooling-r02.log`。
## 2. 17 唯一文件 / 21 声明槽 Inventory

**槽位数学**：S4.16 15（13 核心 + lifecycle 2 support，`tasks.md:2666-2681`）+ S4.19 2 写（`:2832-2838`）+ S4.20 4 写（`:2887-2891`）= 21。共享重复 4 槽：`main.go` 由三卡共享（重复 2）、`recovery_runtime.go` 由 S4.16/S4.20 共享（重复 1）、`recovery_runtime_integration_test.go` 由 S4.16/S4.20 共享（重复 1）；21 − 4 = **17 唯一文件**，与 Plan `:1309`/`:1600` 登记一致。

**清单外写入**：0。逐文件核对均落在登记写集内；`cmd/han-server/recovery_workers_test.go`（`B5CFC39DA1304026449549AAAC58C8231840A7F2A4166241AC8CD2B55D7B9532` / 28928 B / mtime 2026-10-30T08:32Z）为只读复用 harness，不在 17 文件清单，本轮未被修改（与 `tasks.md:2951` 基线一致），不构成清单外写入。

**漂移**：`g2-12-inventory-postrun.json`（测试后）17/17 `match=true`；收尾 `g2-13-inventory-recheck.json` 与 postrun 逐项比较 SHA256 + 字节 + mtime，**drift=0**。xDesk.exe 6 进程（PID 22928/25764/29960/35384/40736/49420）存活，但复跑窗口内未观测到工作区文件被外部改动。

| # | 文件 | SHA256 | 字节 | 行 | 漂移 |
|---|---|---|---:|---:|---|
| 1 | cmd/han-server/recovery_runtime.go | 900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7 | 35832 | 1110 | 0 |
| 2 | cmd/han-server/recovery_runtime_test.go | AB1782FF7F9AB398D3BFB1695D6FEBAD7517109A32B6D73688746D264AD56363 | 26928 | 770 | 0 |
| 3 | cmd/han-server/recovery_config.go | 83FFC39F498468BB81F4ED092862B46D6EE80B45B1BE74D6F502941C2EE606B0 | 9842 | 280 | 0 |
| 4 | cmd/han-server/recovery_config_test.go | 9071DCABB0619FD482656BA9FE8CBA9A19225048805FF44733472315A78A095E | 2959 | 75 | 0 |
| 5 | cmd/han-server/recovery_command.go | 4BCF661DFFD166554C1FF7210936C7B3EB9AF9CC31CE8B555B73E130052123F3 | 2110 | 65 | 0 |
| 6 | cmd/han-server/recovery_command_test.go | 2FA4BFA09674BED725D4F02F0FDBFBB9EA07AF851D56E1D6DEBBA661054FB17A | 2326 | 55 | 0 |
| 7 | cmd/han-server/recovery_sources.go | 935DD7BDC3C3DF4B7FCF045622EEBC95D3BF30DEE313C93D19E1CC335B7DF90C | 24136 | 662 | 0 |
| 8 | cmd/han-server/recovery_sources_test.go | BBCB47359E41A274FC5414FD2F5DFF35FBAE73FE816B76D241764E5A21188EF4 | 14797 | 390 | 0 |
| 9 | internal/identity/recovery_handler.go | 4754B880E557F0B15BB1385CF13E2B37F08074E9C0F26F6194A4548755A00E52 | 25057 | 649 | 0 |
| 10 | internal/identity/recovery_handler_test.go | 6643613FE9A9518DC9A242424529DA64CD08BDD2C0A65902E752EC3BEBEA2115 | 42139 | 925 | 0 |
| 11 | cmd/han-server/recovery_runtime_integration_test.go | B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A | 31638 | 769 | 0 |
| 12 | cmd/han-server/main.go | 6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727 | 173317 | 5044 | 0 |
| 13 | cmd/han-server/main_test.go | 0E61F4A37D314D2D35B80EA8184BEF17EB6227FCAF1CA0F57C09736005185820 | 399522 | 11189 | 0 |
| 14 | cmd/han-server/lifecycle.go | 0CA9C3391382BDE21C2B71796FE0FC5494319C66697BD9F5C6FB65809988F663 | 21788 | 737 | 0 |
| 15 | cmd/han-server/lifecycle_test.go | 33D641DF144BC9B39EF9F89A3E8AA310205C0A76C2137CB93522960EE71E4F05 | 45066 | 1364 | 0 |
| 16 | cmd/han-server/recovery_main_dispatcher_integration_test.go | BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A | 23693 | 623 | 0 |
| 17 | cmd/han-server/recovery_production_chain_integration_test.go | 65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75 | 167339 | 3868 | 0 |
## 3. 跨卡接口一致性（独立核读）

- **唯一 handle**：`main.go:344` 单字段 `recoveryHandle func() recovery.Runtime`；`:546-552` 单闭包 + RWMutex；`:601` 注入业务 composition；`:634-657` 唯一 `newRecoveryRootAssembly` 并把 `assembly.runtime` 回填同一 handle。无第二 handle。
- **唯一 runtime / Store（生产代码）**：全仓 `rg` 显示生产非测试代码只有 `recovery_runtime.go:588` 调用 `recovery.NewRuntime`；`identity.NewRecoveryRunStore` 生产仅 `recovery_runtime.go:490`；`article.NewPostgresReplayEvidenceStore` 生产仅 `main.go:568`（主构造）与 `main.go:3335`、`recovery_runtime.go:497`（均为 nil 回退，生产路径已注入非 nil）。
- **同一 evidence Store**：`main.go:566-572` 在 consumer 之前创建；`:588` 注入业务 composition；`:636` 作 root `closures`；`:641` 以同一 Store 建 `NewReplayPublicationPorts`，满足“同 Store 同源封装”。
- **guard 位置/顺序**：`main.go:3554` router；`:3561/3565` `/livez`、`/readyz` exact match；`:3668-3675` 错 method 405 + `Allow`；`:3677-3683` probe 经同一 `recoveryGuardFromRuntimeHandle(recoveryHandle)`；`:3694-3696` 仅在 exact route + method 通过后 `guard.Protect(handler)`（Session/AccessGeneral 在 handler 内，故 guard 在其之前）；`:3700-3707` Trace 最外。符合“exact router 之后、Session/AccessGeneral 之前”。
- **无第二 router/pool**：`main.go:4034` 的 `http.NewServeMux()` 位于 `serve` 的维护/控制 listener（`:4029-4042`），不是业务 dispatcher；业务 pool 仅 `main.go:3992` 一处 `pgxpool.NewWithConfig`；未发现第二 authority/fallback。
- **`/livez` `/readyz` 同 handle**：`:3677-3683` 使用同一 `recoveryHandle`；无 runtime 时 `recoveryGuardFromRuntimeHandle` 返回 `ok=false`，probe 走零值 guard（`/livez` 200、`/readyz` 503），与 `NoRecoveryConfig` 断言一致。
- **resume 完整完成 → `public_ready` → 同会话开流**：完整恢复链由层 (a) `TestRecoveryRuntimePostgres`（真实 PG）承担，本轮普通/race 各 7 条 pass 记录；开流侧由层 (b) `PublicReadyAdmitsSameSession` 以真实 `recovery.NewRuntime` + `RunNormal` 进入 `public_ready` 后同一 Session 200 证明；两层同修订同跑。跨层单测试串联仍由 S4.20 `TestRecoveryProductionChainComplete` 承担（本轮 NOT RUN，见 §7）。
## 4. 迁移与回归

- **S4.20 迁移**：`cmd/han-server/recovery_runtime_integration_test.go` 当前 `B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A` / 31638 B / 769 行，与 `tasks.md:2949` ③ 登记值一致，未发生漂移。
- **S4.16 既有链**：`TestRecoveryRuntimeProductionChain` 仍在 `cmd/han-server/recovery_runtime_integration_test.go:165`，函数名、文件与断言未被本轮改动（文件哈希与 S4.16/S4.20 基线一致）。**本轮 NOT RUN**，不作为通过证据。
- **S4.19 同修订两层**：层 (a) `TestRecoveryRuntimePostgres`（`test/integration/recovery/runtime_admission_test.go:11`，真实 PG）+ 层 (b) `TestRecoveryMainBusinessDispatcher`（`cmd/han-server`），在 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`、17 文件 drift=0 的同一工作树上普通与 `-race` 均通过。
- **S4.20 四文件未改动**：`main.go` `6D5373…` / 173317 B、`recovery_runtime.go` `900F31…` / 35832 B、`recovery_runtime_integration_test.go` `B8DEBC…` / 31638 B、`recovery_production_chain_integration_test.go` `65602F…` / 167339 B，与 `tasks.md:2949` ③ 登记逐项一致。
- **S4.19 卡面一致性**：`:2859` 定义两层（(a) 真实 PG 完整链；(b) 本 leaf 同 production handler/同 Session）；`:2862` 关闭条件；`:2868` 两层缺一不通过；`:2933-2935` 要求 S4.20 在冻结的 `main.go` SHA 上重跑 S4.19 dispatcher/probe 两层。本轮证据逐项对应；且未用 `setState`、测试 transport 或 fake world 冒充层 (b)——`PublicReadyAdmitsSameSession`（`:179-281`）经 `newOrdinaryFixture`（`recovery_workers_test.go:148,156` 真实 `recovery.NewRuntime`）与 `RunNormal`（`:220`）进入状态，全文无 `setState`。
- **checkbox 未动**：S4.16 `[x]`(:2659)、S4.17 `[x]`(:2725)、S4.19 `[-]`(:2820)、S4.20 `[x]`(:2873)、S8.2 `[-]`(:1859)、S8.3 `[x]`(:2519)。本轮写集仅证据目录，未改任何 checkbox、日期或写集。
## 5. 上轮 FAIL 项处置判定

### 5.1 `B13H-GATE-F001`（HIGH）→ **CLOSED**

1. **新增第 6 子测试**：`PublicReadyAdmitsSameSession` 注册于 `recovery_main_dispatcher_integration_test.go:30`，实现 `:179-281`；`g2-08` JSON 实测该子测试 pass。
2. **真实 runtime**：`newOrdinaryFixture`（`cmd/han-server/recovery_workers_test.go:148-159`）调用真实 `recovery.NewRuntime`；测试 `:220` 调 `RunNormal(ctx)`，`:223-229` 以 `Verify` 断言 `public_ready`。该子测试无 `setState`；全文 `setState` 只在 probe fake（`:131/148/158`，`:144-147` 注释声明 probe-only 边界）与 single-handle fake（`:301/512/520`）。
3. **薄包装**：`s419ObservedRuntime` / `s419ObservedLease`（`:579-623`）嵌入真实 `recovery.Runtime`，仅对 Acquire/Release 计数并委托真实调用，不是 fake 状态机。
4. **断言闭环**：closed 下 503 + `platform.recovery_in_progress` + Trace（`:194-204`）、无内部标记泄漏（`:205`）、Article repository/reader/Session 零调用（`:206-214`）、仅 trace 事件（`:215`）、acquire/release=1/0（`:216`）；`public_ready` 后 `/readyz` 200（`:230`）、同一请求 200（`:234`）、同一 Session cookie（`:235`）、Article 正常调用与事件顺序（`:238-247`）、acquire/release=2/1（`:248`）；失败负控返回 `platform.database_unavailable` 且明确排除 recovery code（`:257-279`）。
5. **删除未调用函数**：全包 `rg publishPublicReady` 0 命中（exit=1）。

结论：层 (b) 不再是假进入，真实经过 `recovery.NewRuntime` 与 `RunNormal`，且覆盖 closed 503 / `public_ready` 同请求 200 / acquire-release 计数 / 失败负控。**F001 CLOSED**。

### 5.2 `N01`（LOW）→ **CLOSED**

`rg -n "publishPublicReady" --glob "*.go"` 退出码 1、命中 0；与 `tasks.md:2869` 返工合同一致。

### 5.3 `B13H-GATE-F002`（MEDIUM）→ **CLOSED**

`tasks.md:2950` 已追加更正且未改 `:2949` 原文：明确 `:2949` “两层闭合”表述作废；限定 `d7-main-dispatcher-r02.json` 仅证明 production dispatcher 在 closed/unknown/probe/single-handle/no-config 范围通过；层 (b) 由 S4.19 返工承接；S4.20 resume 链仍以 `TestRecoveryProductionChainComplete` 真实 PG 为准；S4.20 `[x]` 与完成日期不变；批次 Gate 在两层同修订通过前不得 PASS。该处置满足“保留历史记录 + 追加限定”的更正方法；本轮已实测两层同修订通过，附加条件已满足。**F002 CLOSED**。

### 5.4 r04 两条 LOW → 已披露、非阻断

- `S419-R04-F001`（LOW，测试有效性）：`s419AssertNoRecoveryLeak` 对 payload 子串 `run/ref` 等做包含检查，随机 Trace/ID 可能误命中（概率很低，约 0.36%），表现为偶发假失败，不掩盖真实缺陷。
- `S419-R04-F002`（LOW，覆盖边界）：真实 runtime 在 closed 下 `/readyz` 503 分支未在本文件断言；本文件以 probe fake 覆盖状态矩阵，真实 runtime probe 由层 (a)/S4.20 链承担。

两条按已披露登记、不要求修复；**不影响本次批次 Gate 判定**，建议列入后续测试卫生改进。
### 5.5 回归有效性（变异思路，不修改生产代码）

- 若 guard 被移除或移到 Session 之后：closed 的 503/`recovery_in_progress` 与“Session 零调用”断言会失败（`:194-214`）。
- 若以 `setState` 代替真实恢复：缺少 `RunNormal` 时 `Verify` 不会返回 `public_ready`（`:223-229`），`/readyz` 仍为 503（`:230`）。
- 若未真实 acquire/release：计数断言 (1,0)/(2,1)/(+1,+1) 失败（`:216`、`:248`、`:278`）。
- 若 `public_ready` 后仍隔离：同请求 200（`:234`）与 Session 调用次数（`:244`）失败。
- 若下游失败被 503 掩盖：负控 `database_unavailable` 断言失败（`:263-268`）。

上述断言与生产分支一一对应，缺陷回归会使测试失败；按边界要求未实际改生产代码做变异。

## 6. 新 finding

- **B13H-GATE-R02-L01**（LOW，测试有效性）：位置 `cmd/han-server/recovery_main_dispatcher_integration_test.go:26-33`。子测试注册在顶层函数内，若未来删除某个 `t.Run`，`-run '^TestRecoveryMainBusinessDispatcher$'` 仍会 PASS，覆盖回退不会被测试自身捕获（本轮靠 6 个顶层子测试 + 11 条具名记录核对）。影响：低，不削弱当前证据；建议在评审/关闭清单固定子测试清单或增加显式断言。
- **B13H-GATE-R02-L02**（LOW，运行期隔离）：位置 `cmd/han-server/main.go:3333-3339`（及 `recovery_runtime.go:497`）。nil 回退分支若被触发会构造第二个 evidence Store；当前只有指针层证明（生产 composition 传入非 nil），缺运行期“进程内只有一个 Store/pool”的断言。影响：低，不阻断；建议后续以进程级/装配级断言补强。
- **观察 O01（非缺陷，不阻断）**：`plans/ir-01-content-community-batches.md:1599` 仍记“S4.19 已关闭为 `[x]`”，而当前 `tasks.md:2820` 为 `[-]`（Gate FAIL 后重开）。Plan 自述不维护状态（`:1329-1332`），该句为历史时点记录；建议 Lead 在 S4.19 正式关闭后补一条 B13H 重跑/返工指针，避免读者误读。
## 7. 未验证边界（不得当作通过）

- 性能：无负载/延迟/吞吐/容量测试。
- 真实部署/E2E：未起真实服务进程、未做 HTTP/TLS/反代/多实例部署验证。
- POSIX journal/fsync/flock/CAS：未验证真实文件系统语义（层 (a) 使用受控软件适配器范围）。
- 真实 keyring/HSM 与 recovery secret candidate：未验证；`enter/install` 仍为 closed 占位（`recovery_runtime.go:482-483`）。
- approved backup inventory 与真实 provider：未验证（S4.20 关闭记录已披露）。
- production dispatcher 全路由：本批仅覆盖 exact route 子集、404/405/Allow、`/livez` `/readyz` 与 admin articles 两路由；未逐路由回归。
- 进程级唯一 runtime/Store/pool：仅静态指针层 + 单测证明（见 L02），无进程级断言。
- S4.16 `TestRecoveryRuntimeProductionChain`：本轮 **NOT RUN**（仅确认文件/函数/哈希未变）。
- S4.20 `TestRecoveryProductionChainComplete` 与五叶、`d7-*` 证据：本轮 **NOT RUN**，仅引用历史证据，不视为本轮通过。
- `goimports`/`golangci-lint`：项目未要求，本轮未跑。
- 人的业务接收、Case18 整体验收、Git 写授权：均未发生；本报告不代表真人签字。

## 8. 复跑命令与结果

环境：`postgresql-x64-18` = Running；`$env:HAN_TEST_POSTGRES_DSN = $env:HAN_DATABASE_DSN`（只记录已设置，不记录值）。

| 命令 | exit | pass | fail | skip | DATA RACE | 证据 |
|---|---:|---:|---:|---:|---:|---|
| `gofmt -l`（17 登记文件） | 0 | — | — | — | — | `commands.log`（输出 0 行） |
| `go build ./...` | 0 | — | — | — | — | `g2-05-build.log` |
| `go vet ./cmd/han-server` | 0 | — | — | — | — | `g2-06-vet.log` |
| `go vet -tags=integration ./cmd/han-server` | 0 | — | — | — | — | `g2-07-vet-integration.log` |
| dispatcher 普通 | 0 | 12（11 具名 + 1 包级） | 0 | 0 | 0 | `g2-08-dispatcher-normal.json` |
| dispatcher `-race` | 0 | 12（11 具名 + 1 包级） | 0 | 0 | 0 | `g2-09-dispatcher-race.json` |
| PG 普通（真实 PG） | 0 | 7（6 具名 + 1 包级） | 0 | 0 | 0 | `g2-10-pg-normal.json` |
| PG `-race`（真实 PG） | 0 | 7（6 具名 + 1 包级） | 0 | 0 | 0 | `g2-11-pg-race.json` |

子测试清单（`g2-08` 实测）：`ExactRouteBeforeGuard`、`ClosedUnknownTwoAdminGets503/{closed,unknown}`、`LiveReadyProbes`、`PublicReadyAdmitsSameSession`、`SingleRuntimeHandle/{LateBinding,ReleaseAfterPanic}`、`NoRecoveryConfig`。层 (a) 子测试（`g2-10`）：`GenesisDeletionHistoricalRestart`、`CrossNamespaceABAAndExactPublication`、`F001F002AndCanonicalIdempotencyReview`、`CrashConsumedWindowAndDrain`、`Case18SameRevisionAndNoProductionClaim`。

## 9. 证据文件

- `g2-05-build.log` 0 B、`g2-06-vet.log` 0 B、`g2-07-vet-integration.log` 0 B
- `g2-08-dispatcher-normal.json` 10501 B、`g2-09-dispatcher-race.json` 10498 B
- `g2-10-pg-normal.json` 31404 B、`g2-11-pg-race.json` 31400 B
- `g2-12-inventory-postrun.json` 4564 B（17/17 match）、`g2-13-inventory-recheck.json` 4482 B（drift=0）
- `g2-01`～`g2-04`：前批交叉观察，计数同样为 12/7、fail/skip/race=0

**结论：PASS WITH MINOR。** F001/N01/F002 CLOSED；无 HIGH/MEDIUM 新增；4 条 LOW（r04 两条 + 本轮 L01/L02）与 O01 观察均不阻断。S4.19 当前仍为 `[-]`，是否关闭由 Lead 按 `task-execution-workflow.md` §10.3 决定。