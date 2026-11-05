# Han S4.16 / B13H 独立代码评审报告

- 卡片：`identity-auth-S4.16`，批次 `B13H`
- objectType：`code`（两轮独立评审）
- 工作区（固定）：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 作者：`add-backend-dev`，Agent ID `01a18d45-b04b-7581-a172-0296b0341ccb`
- 独立 Reviewer：`add-reviewer`，Agent ID `01a18e02-451f-78c1-89d2-dea8b9af0f63`
- 独立性：Reviewer ID 与作者 ID 不同；未接受作者自述作为证据，所有结论以现场源码、测试源码与本人执行的命令日志为准。
- 设计依据：`.add/project/knowledge/design-workspace/identity-auth/design.md`，SHA256 `0FC3DE78C950661C2947680F678F92B7390843883FAD4F90362DB2C3F5A19277`（现场复核一致）
- 测试依据：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md`，SHA256 `34826759C79A58EB0B4A296D2DD5C9157014406859153DC6CA0FA6E2BD921DAB`（现场复核一致）
- 卡片依据：`.add/project/knowledge/tasks/tasks.md#task-identity-auth-s4-16`，SHA256 `8BE0887359F641A7C6D707D6A84BC5DBEFBEE92C4A67671FB32EA57B9EE6B350`（现场复核一致）
- 范围依据：`.spec/wip/recovery-runtime/evidence-b13h-r01/s416-G0-main-reception-and-scope-decision-r01.json`
- Lead Gate 对照：`.spec/wip/recovery-runtime/evidence-b13h-r01/lead-gate-r03/gate-summary-r02.json`、`run-results-r01.json`、`cmd-01..cmd-13.log`

## 1. 结论摘要

**CodeGate：CHANGES_REQUIRED**

- findings 共 8 条：HIGH 4（F001-F004）、MEDIUM 2（F005、F008）、LOW 2（F006、F007）；其中 F006 为 F001 的同根子项。
- 最高严重度：HIGH。
- 阻断项：F001、F002、F003、F004 必须修复并回归；F005、F008 建议修复；F006 随 F001 一并修复；F007 为低风险建议（受卡片文件边界约束，不要求本轮越界新增配置）。
- 本结论不代替 Lead 接收、不代替人工业务接纳或签字，不关闭任务，不声明 release 或生产就绪。

## 2. 两轮实际检查范围

### Round 1（独立全量：基线 + 边界 + 卡片命令 + 源码通读）
- 现场复核 15 个文件 SHA256，全部与 Lead 基线一致（`ALL15=True`）；`internal/recovery` 最新 mtime 仍为 `2026-10-30 16:43:50`，未动。
- 通读 `recovery_runtime.go`、`recovery_sources.go`、`recovery_workers.go`、`recovery_config.go`、`recovery_command.go`、`main.go`、`lifecycle.go`、`internal/identity/recovery_handler.go`、`internal/recovery/runtime.go`、`internal/session/store.go`、`internal/session/maintenance_store.go`、`internal/identity/recovery_auth.go`、`internal/identity/login.go`、`internal/platform/http_security.go`。
- 通读测试：`recovery_runtime_test.go`、`recovery_runtime_integration_test.go`、`recovery_handler_test.go`、`main_test.go`、`lifecycle_test.go`、`recovery_sources_test.go`、`recovery_command_test.go`、`recovery_config_test.go`。
- 独立执行 13 条卡片命令 + `gofmt`，全部 exit 0（日志见第 3 节）。
- 判定 9 项重点核对项，定位 F001-F008。

### Round 2（定向复核：原 finding 与测试覆盖）
- 重跑 4 条定向测试（Composition、ProductionChain、ControlHandlers、PasswordResetMaintenanceRoot），全部 exit 0。
- 定向读测试源码，确认 Fake/Seam 与未覆盖生产路径：`recovery_runtime_test.go:282/341` 与 `recovery_runtime_integration_test.go:338` 直接调用 composition `ExitRecovery`；integration `:108-120` fake resolver、`:223-241` fake authorizer/executor/runtime。
- 补跑 2 条只读复验（ProductionChain、RootStartup），全部 exit 0。

## 3. 实际执行命令与计数（只读）

Round 1（`r01-cmd-01..13.log`、`r01-gofmt.log`、`r01-command-results.jsonl`；DSN 仅置于环境变量，未落盘）：

| # | 命令 | exit | 证据 |
|---|---|---|---|
| 01 | `go test ./cmd/han-server -list ^TestRecoveryRuntimeComposition$` | 0 | 命中 |
| 02 | `go test ./cmd/han-server -run ^TestRecoveryRuntimeComposition$ -count=1 -json` | 0 | pass events=9, fail=0 |
| 03 | 02 + `-race` | 0 | pass events=9, fail=0 |
| 04 | `go vet ./cmd/han-server` | 0 | 无输出 |
| 05 | `go test -tags=integration ./cmd/han-server -list ^TestRecoveryRuntimeProductionChain$` | 0 | 命中 |
| 06 | `go test -tags=integration ./cmd/han-server -run ^TestRecoveryRuntimeProductionChain$ -count=1 -json` | 0 | top-level PASS=1，无 subtest；真实 PG |
| 07 | 06 + `-race` | 0 | top-level PASS=1；真实 PG |
| 08 | `go vet -tags=integration ./cmd/han-server` | 0 | 无输出 |
| 09 | `go test ./internal/identity -list ^TestRecoveryControlHandlers$` | 0 | 命中 |
| 10 | `go test ./internal/identity -run ^TestRecoveryControlHandlers$ -count=1 -json` | 0 | pass events=67, fail=0 |
| 11 | 10 + `-race` | 0 | pass events=67, fail=0 |
| 12 | `go test ./cmd/han-server -run ^TestPasswordResetMaintenanceRoot$ -count=1 -json` | 0 | pass events=169, fail=0 |
| 13 | `go test ./cmd/han-server -run ^TestRecoveryRootStartup$ -count=1 -json` | 0 | pass events=5 |
| — | `gofmt -l` 15 文件 | 0 | 无输出 |

Round 2（`r01-round2-cmd-01..04.log`、`r01-round2-results.jsonl`）：Composition pass=9；ProductionChain top PASS=1；ControlHandlers pass=67；PasswordResetMaintenanceRoot pass=169，全部 exit 0。

本轮定向复验：
- `r01-verify-productionchain.log`：`go test -tags=integration ./cmd/han-server -run ^TestRecoveryRuntimeProductionChain$ -count=1 -json`，EXIT=0，PASS=2 FAIL=0（真实 PG，临时 schema）。
- `r01-verify-rootstartup.log`：`go test ./cmd/han-server -run ^TestRecoveryRootStartup$ -count=1 -json`，EXIT=0，PASS=6 FAIL=0。

> 说明：#06/#07 与 ProductionChain top-level PASS=1 是**真实 PostgreSQL 链顺序断言**，日志证明 `recovery_runtime_integration_test.go:169/170` 打开并迁移真实 B13H root/backup 临时 schema（migrations 0001-0010、真实 pgxpool、`session.NewStore`、`identity.NewRecoveryReviewLister` 真查库）。但认证、授权、maintenance/trust、runtime 与 executor 为 Fake（`:108-120`、`:223-241`），未实例化生产 root assembly。详见 F001-F004 的测试盲区。

## 4. 九项重点核对结论

1. **closed 适配器不是 no-op — PASS**：`recovery_sources.go:440-445` `recoveryClosedPort.closed()` 返回 `ErrAdmissionClosed`；`LoadInstallation/CompareAndSwapInstallation/VerifyInstallationSource/Load/Acquire`（:448-466）全部返回错误；`recoveryClosedVerifiedJournalReader`（:486-495）、`recoveryClosedRestoreDriver`（:501-514）、`recoveryClosedConsumer`（:520-521）、`recoveryClosedInstallationExecutor`（:527-528）、`recoveryClosedLifecycle`（:535）均只返回错误；`newRecoveryAnchorRegistry` 未登记返回 `ErrAdmissionBinding`（:402）。四接口断言在 `:468-473`。无成功 no-op。
2. **verify 零 listener/worker/repair — PASS**：`main.go:632-637` verify 分支在监听前直接返回；`recovery_runtime.go:582-590` verify 下 `bindWorkers` 返回 nil；`:606-615` 仅 Verify。RootStartup 子测试 `verify_opens_no_listener_worker_or_repair` 覆盖。
3. **complete 先真实响应再 root 独立控制 — PASS（主体）**：`internal/identity/recovery_handler.go:417-435` 先 `WriteJSON(200)` + `Flush`（:426-433），再 `RequestComplete`（:434）；`recovery_runtime.go:717-730` 独立 goroutine `completeOnRequest`；integration `:300-345` 证明 handler 返回时 executor complete=0、root 调用后=1。错误吞掉问题另见 F008（不改顺序结论）。
4. **三 server join / lifecycle 收集错误 — 结构 PASS，实际 join FAIL（F003）**：`lifecycle.go:342-390` collectShutdown 收集非 worker server 错误，失败时 `finishServers` 对所有 served server Close；`:573-593` 拒绝 nil/incomplete additionalServer（`:582-586`）；`:325-338` 顺序 joinWork→关 listener→closeArticle→最后 closePool（PASS）。但 `main.go:698/711-713` 注册原始 `controlServer`，而 wrapper `recoveryJoinedServer` 从未 Serve，join 等不到 done（F003）。
5. **normal/verify 不因未使用安装来源配置拒绝 — PASS**：`recovery_config.go:88-98` 统计键；`:110-139` 全空且无 command→zero plan；`readAction` 仅 enter/resume 需要 restore；`readCommon` 不含 restore/control 键。Lead 裁决 1 成立。normal/resume 未使用 ioBudget 另见 F005。
6. **单一长期 Article control handle — handle 实现 PASS，生产 settle 不可达（F001/F004）**：`recovery_runtime.go:188-262` Close/Join/settleOnce 正确；生产 composition settle 因 F001 不执行，且 F004 时序会阻塞。
7. **cmd 无身份 SQL / root 不持 key/HMAC — PASS**：cmd recovery 源仅命中配置键名、`recoveryOpenJournal`、`recoveryLoadKeyring`；keyring loader 默认拒绝（`recovery_runtime.go:291-295`）；review SQL 在 `internal/identity/recovery_handler.go`（:503 所在文件），不在 cmd。
8. **15 文件边界 / internal/recovery 未动 — PASS**：15 文件 SHA256 全一致；`internal/recovery` 最新 mtime 仍 `2026-10-30 16:43:50`；`cmd/han-server` 2026-10-30 17:00 后修改文件恰为清单内 15 个。`go.mod` 为 pre-existing（2026-10-21），非第 16 文件，仅作观察说明。
9. **测试是否触发根因 / PG 链 — 部分 FAIL/盲区**：Composition 测试直接调 composition，不覆盖生产 assembly Exit 注入（F001）；ProductionChain 真用 PG 但 fake auth/executor/runtime，未覆盖 F002/F003/F004；Control handler 与 PasswordResetMaintenanceRoot 全绿，但未覆盖生产 `newProductionRecoveryControlHandler` 的普通 store 配 maintenance 登录链；RootStartup 覆盖 verify 零副作用。

## 5. Lead 三条技术裁决独立判断

1. **normal/verify 配置门控 — 成立**：依据 `tasks.md:2694` 与 `recovery_config.go` 源码，全空不构造 recovery runtime、缺配置 fail closed。不影响 F005（normal/resume ioBudget 缺失）。
2. **Windows `NewFileRecoveryJournal` 返回 `ErrUnsupportedPlatform` 为合法 closed 路径 — 成立**：`var recoveryOpenJournal` 仅测试 seam，默认真实构造（`recovery_runtime.go:291-295`）；不构成 no-op；design.md:4646 支持矩阵一致。
3. **closed 适配器不是 no-op — 成立**：见第 4 节第 1 项，依据 `recovery_sources.go` 与 design.md:1675/1691。

## 6. Findings

### F001 — HIGH — 生产 Runtime 注入 surface exit，revoke/settle 不可达
- object/revision：`code` / HEAD `6da8abb`
- current：`cmd/han-server/recovery_runtime.go:494`（`Exit: assembly.exit`）、`:531`（composition `Exit: assembly.exit`）；`internal/recovery/runtime.go:1917` replay 调 `deps.Exit.ExitRecovery`；`recovery_runtime.go:332-352` `recoverySurfaceExit.ExitRecovery` 只 `joinConfigured`；真正 exit→`bridge.revokeSessions`→`Article.settle` 只在 composition `exitRecovery`（`:89-111`）。
- trigger：成功 replay 到达 Exit。
- impact：maintenance sessions 不撤销、Article handle 不 settle；违反 `RootExitTrueJoinAndSameExecutor` / `LongLivedControlHandleTrueJoin`；无法可靠归还资源并开放普通服务。
- 最小修复：Runtime 的 Exit 指向 composition；composition 只调一次 surface exit，再 revoke+settle；补 assembly 路径测试。
- 验证方式：断点/测试断言 assembly 注入的 Exit 触发 revoke+settle 各一次；补真实 assembly Exit 测试。
- 子项 F006：`recovery_runtime.go:100-104` 已建 `bounded` 却用原 `ctx` 调用 `ports.Exit.ExitRecovery`；改用 `bounded`。

### F002 — HIGH — maintenance control surface 用普通 session store/authorizer，维护 cookie 无法认证
- object/revision：`code` / HEAD `6da8abb`
- current：`recovery_runtime.go:881-882` 第二个 `PasswordResetStore` + `session.NewStore`；`:891-899` `LoginService{RecoveryMode: wiring.Mode}`（`wiring.Mode` 来自 `:552-558`，实为 `*recoveryIdentityExecution`）；`internal/identity/login.go:1341-1342` → `recovery_auth.go:186-207` → `session.NewMaintenanceStore`；`internal/session/maintenance_store.go:229-232` 写 `access_scope='recovery_maintenance'`；`recovery_runtime.go:914` `platform.NewHTTPAuthorizer(store, ...)` 用普通 store；`internal/session/store.go:553-694`（`:692`）普通 resolver 要求 `access_scope='ordinary'`。
- trigger：维护登录成功后访问 `/api/v1/users/me`、改密、reviews、resolve、complete 等 `Authorizer.Protect` 路由。
- impact：登录成功但全部维护受保护路由拒绝，恢复流程无法完成；构造第二 gate/store 而非复用同一生产身份执行者。
- 最小修复：用同一 sealed executor 构造 maintenance store（参考 `internal/identity/recovery_handler.go:437-453` `NewRecoveryMaintenanceStore`；内部 `recovery_auth.go:167-171` 用 sealed executor 的 pool + Authority），复用同一 PasswordResetGate/authority；authorizer 用该 maintenance store；补真实 admin 登录→维护路由链测试。
- 验证方式：真实维护 Login→HTTPAuthorizer→`/users/me`/password 链测试；断言 scope 为 `recovery_maintenance` 时通过、越权被拒。

### F003 — HIGH — `recoveryJoinedServer` 从未被 Serve，join 永远等不到 done
- object/revision：`code` / HEAD `6da8abb`
- current：`recovery_runtime.go:634-649` `bindControl` 创建 wrapper 并绑定 `joined.join`；`:356-384` 只有 wrapper `Serve` 会 `close(s.done)`；`main.go:698/711-713` 把原始 `controlServer` 放入 `additionalServers`；`lifecycle.go:606-613` 对原始 server 调 `Serve`，wrapper.Serve 未执行。
- trigger：maintenance 模式 shutdown / RecoveryExit join control surface。
- impact：等 done 到预算超时返回 ctx error；三 server「真实 join」失败，违反 complete 后真实 join。
- 最小修复：`bindControl` 把 wrapper 注册到 lifecycle（或返回 wrapper 给 main 放 additionalServers）；补真实 Serve+Shutdown join 测试。
- 验证方式：真实启动 control surface 后触发 shutdown，断言 join 在预算内成功且 served server 的 Shutdown 错误被收集。

### F004 — HIGH — Article control handle settle 等待尚未启动的 worker Done，pipeline 可能超时阻塞
- object/revision：`code` / HEAD `6da8abb`
- current：`main.go:625` `newRecoveryArticleJoin(runtime)`；`recovery_runtime.go:838-861` 遍历 `runtime.articleWorker/passwordResetWorker/noticeWorker` 的 `Done()`；`internal/recovery/runtime.go:1917` replay 先 `ExitRecovery`，`:1940` 之后才 `Lifecycle.ReconcileAndStart`；worker Done 仅在 run 退出后 close（`internal/identity/password_reset_worker.go:166/191`、`internal/article/deletion_worker.go:246/281`、`internal/article/notice_dispatcher.go:297`）；`cmd/han-server/recovery_workers.go:84-111` 才 Start 三 worker。
- trigger：F001 修复后 Exit 真正进入 composition settle；replay 时三 worker 尚未 Start（或 Stop 先于/绕过正常启动）。
- impact：Article settle 等 Done 到 drain budget 超时，`RevokeRecoverySessions` 不执行，恢复失败/关闭；当前被 F001 掩盖，修 F001 即暴露。
- 最小修复：Article join 只等已启动的实际持有者；启动阶段与 Exit 时序分离，或 join 前先确保 worker 生命周期已收敛；补 replay（Exit 先于 ReconcileAndStart）测试。
- 验证方式：构造 replay 顺序测试，断言 Exit 的 settle 不等待未启动 worker 且 revoke 执行。

### F005 — MEDIUM — normal/resume 未使用 ioBudget
- object/revision：`code` / HEAD `6da8abb`
- current：`recovery_runtime.go:697-712` normal `RunNormal(ctx)`、resume `Resume(ctx)`；verify `:687-695` 用 bounded。
- trigger：normal/resume 路径。
- impact：normal/resume 不受 `HAN_RECOVERY_IO_BUDGET` 限制，可能无限等待；违反 design.md:3894「deadline 取父截止与相应预算之较早者」。
- 最小修复：normal/resume 从父 ctx 派生 `context.WithTimeout(..., budget)`；补短 budget 超时测试。
- 验证方式：设置极小 ioBudget，断言 normal/resume 在预算内返回 deadline 错误。

### F006 — LOW（并入 F001）— composition 用未 bounded ctx 调 surface exit
- object/revision：`code` / HEAD `6da8abb`
- current：`recovery_runtime.go:100-104` 已建 `bounded`，`:104` 却用原 `ctx`。
- trigger：composition exit 且 surface join 阻塞。
- impact：surface join 不共享同一关闭截止，可能先阻塞到父 ctx 超时，后续 revoke/settle 无机会。
- 最小修复：`:104` 改用 `bounded`。
- 验证方式：随 F001 测试覆盖。

### F007 — LOW — 硬编码 `recoveryBridgePhaseBudget = 5s`
- object/revision：`code` / HEAD `6da8abb`
- current：`cmd/han-server/recovery_sources.go:19`，用于 `:221`。
- trigger：维护请求或延迟 executor phase。
- impact：phase deadline 不可部署调整；5s 可能短于合法维护操作；违反 no-hardcode-policy。
- 最小修复：绑定既有配置/启动输入；若卡片文件边界不允许新增配置，写明「低风险建议，受卡片文件边界限制，不要求本轮越界新增配置」。
- 验证方式：配置可覆盖时断言注入值生效。

### F008 — MEDIUM — complete handler 忽略 RequestComplete 错误
- object/revision：`code` / HEAD `6da8abb`
- current：`internal/identity/recovery_handler.go:434` `_ = h.dependencies.Complete.RequestComplete()`；响应已在 `:426-433` 写 200 + Flush。
- trigger：bridge 已 used/closed 或信号失败。
- impact：客户端收到 200，但 root 未收到完成信号，可能等超时/失败；错误被吞（违反禁止 `_ = err`）。
- 最小修复：至少记录结构化错误并让 root flow 得到失败结果；保持「先返回再推进」顺序；补注入 error 测试。
- 验证方式：注入 RequestComplete 错误，断言记录且 root flow 以失败收敛。

## 7. 未验证限制与剩余风险

- Windows 文件 Journal 不支持（`ErrUnsupportedPlatform` 合法 closed 路径）；真实 CAS、provider、keyring、restore driver 未交付（Lead known limitations）。
- 未做真实部署 E2E；未启动真实 maintenance 登录 / 浏览器 cookie 链（F002 即此路径的静态证据）。
- `goimports` / `golangci-lint` 未跑（profile 要求，NOT PASS，未豁免）。
- 未跑全量 suite，仅卡片命令 + 定向回归；`-race` 仅覆盖指定 top-level。
- 真实外部 Host 依赖未验证。
- DSN 仅存在于进程环境变量，未写入仓库任何文件。

## 8. 声明

- 本报告为独立评审结论，不代替 Lead 接收，不代替人工业务接纳或签字，不关闭任务，不声明 release 或生产就绪。
- 复审条件：F001-F004 修复并附回归后，对原 finding 与修复修订重新复核；F005/F008 建议同批；F006 随 F001；F007 按边界决定。