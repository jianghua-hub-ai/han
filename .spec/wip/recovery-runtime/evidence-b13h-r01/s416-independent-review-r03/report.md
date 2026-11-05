# Han S4.16 / B13H 独立代码评审报告（r03 复审）

## 0. 对象与修订指纹
- objectType: code（两轮独立评审）；review_round: r03
- worktree: `D:\AID\VSCode\Codex\Han`；branch: main；HEAD: `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（现场 `git rev-parse` 复核）
- 目标修订：当前未提交工作树，以 `.spec/wip/recovery-runtime/evidence-b13h-r01/lead-gate-r05/files-sha256-r03.json` 的 15 个文件 SHA256 + 字节数固定。
- 现场复核（第一手）：`ALL15_MATCH=True`（15/15 文件 SHA256 与 bytes 全部一致），无漂移。
- 与 r04 manifest 相比：**仅 3 个文件变化** —— `cmd/han-server/recovery_runtime.go`、`cmd/han-server/recovery_runtime_test.go`、`cmd/han-server/recovery_runtime_integration_test.go`；其余 12 个 SHA 完全不变。
- 生产文件变更量：`recovery_runtime.go` 33773 → 33806 bytes（**+33 字节，恰为一行 `\tassembly.runtime = runtimeValue\n`**）；`recovery_runtime_test.go` 22948 → 26928（新增 F009 测试与 fixture）；`recovery_runtime_integration_test.go` 21992 → 31638（新增 F010 测试）。
- 边界：`internal/recovery` 最新 mtime 仍为 2026-10-30 16:43:50（`runtime_test.go`），未动；扫描 `cmd/`+`internal/` 全部 `.go` 中 mtime > 2026-10-30 17:30 且不在 manifest 内的文件 = **0**（无第 16 个产品文件）。
- 本报告不引用 Lead 的 exit code 作为结论；第 3 节全部命令为本轮第一手重跑。
- 评审者真实 Agent ID：`01a18e02-451f-78c1-89d2-dea8b9af0f63`（add-reviewer；由本会话 `CODEX_THREAD_ID` 现场读取）。作者 add-backend-dev `01a18d45-b04b-7581-a172-0296b0341ccb`，其自述未作为验证证据。

### 15 文件指纹（对照 r05 manifest，现场复核）
| # | 文件 | SHA256（前缀…后缀） | bytes | 相比 r04 |
|---|---|---|---|---|
| 1 | cmd/han-server/recovery_runtime.go | 47CCE68A…DA7216B2 | 33806 | **变化**（+33B，F009 回填） |
| 2 | cmd/han-server/recovery_runtime_test.go | AB1782FF…AD56363 | 26928 | **变化**（F009 测试） |
| 3 | cmd/han-server/recovery_config.go | 83FFC39F…C2EE606B0 | 9842 | 未变 |
| 4 | cmd/han-server/recovery_config_test.go | 9071DCAB…78A095E | 2959 | 未变 |
| 5 | cmd/han-server/recovery_command.go | 4BCF661D…052123F3 | 2110 | 未变 |
| 6 | cmd/han-server/recovery_command_test.go | 2FA4BFA0…054FB17A | 2326 | 未变 |
| 7 | cmd/han-server/recovery_sources.go | 935DD7BD…35B7DF90C | 24136 | 未变 |
| 8 | cmd/han-server/recovery_sources_test.go | BBCB4735…5A21188EF4 | 14797 | 未变 |
| 9 | internal/identity/recovery_handler.go | 4754B880…8755A00E52 | 25057 | 未变 |
| 10 | internal/identity/recovery_handler_test.go | 6643613F…3BEBEA2115 | 42139 | 未变 |
| 11 | cmd/han-server/recovery_runtime_integration_test.go | B8DEBCA4…DE44DB8C54A | 31638 | **变化**（F010 测试） |
| 12 | cmd/han-server/main.go | 3050CE4E…59ED32EE97 | 169631 | 未变 |
| 13 | cmd/han-server/main_test.go | 0E61F4A3…6005185820 | 399522 | 未变 |
| 14 | cmd/han-server/lifecycle.go | 0CA9C339…09988F663 | 21788 | 未变 |
| 15 | cmd/han-server/lifecycle_test.go | 33D641DF…E71E4F05 | 45066 | 未变 |

## 1. 结论摘要
- **CodeGate：PASS**（仅代表本独立评审对 code 对象的结论，**不代替**人工业务接纳、Lead 接收裁决或任务关闭）。
- 原 finding：**F001–F008 全部维持 CLOSED**（其中 F002 由 F010 升级覆盖，仍 CLOSED）。
- 本轮修复项：**F009 CLOSED**（HIGH，已修复且有真测试）；**F010 CLOSED**（LOW，覆盖已补齐，附残余限制）。
- 新增 finding：**无（F011 起无）**。
- 未验证限制仍存在（生产 CAS/anchor/commitment/verifier/lease/keyring/restore、真实 provider/ACL、真实部署 E2E、Windows journal 合法 closed、goimports/golangci 未跑、设计 draft、人的接纳 unknown），见第 7 节；这些不构成本对象可判缺陷，但不能被表述为已通过。

## 2. 两轮实际检查范围
### Round 1（基线与 F009/F010 修复审阅）
- 现场核对 15 文件 SHA256 + 字节数 vs `lead-gate-r05/files-sha256-r03.json`；对照 r04 manifest 求差异集；扫描第 16 文件；复核 `internal/recovery` mtime 与 HEAD。
- 逐个通读本轮 3 个变更文件的关键段与全文结构：`recovery_runtime.go`（构造函数 457–598、composition 42–129、exit router 360–392、joined server 396–424、bindWorkers 633、bindArticleOwner 645、verify 744、start 754、joinWork 824、bindControl 685、wiring 601、newRecoveryControlAuthorizer 956–969）、`recovery_runtime_test.go`（F009 测试 714–770 与 fixture 681–712）、`recovery_runtime_integration_test.go`（F010 测试 564–769 全文）。
- 复核生产装配调用链：`main.go:596–660`（plan → assembly → bindWorkers/bindArticleOwner → verify 早于 listen → listen）、`main.go:669–745`（additionalServers + bindControl）、`lifecycle.go:306–380`（shutdown 收集/Close 兜底）、`lifecycle.go:575–590`（nil/incomplete additionalServer 拒绝）。
- 复核 F001–F008 修复点在当前行号的存续：`recovery_runtime.go:89–111`（capture→ExitRecovery→revokeSessions→Article.settle）、`:100–104`（bounded exit）、`:904–940`（Article join 跳过未启动）、`:744–768`（verify/normal bounded）、`recovery_sources.go:24/51–58/280`（phaseBudget 注入）、`internal/identity/recovery_handler.go:426–460`（complete 预检→响应+Flush→RequestComplete）、`newRecoveryControlAuthorizer`→`identity.NewRecoveryMaintenanceStore`→`platform.NewHTTPAuthorizer`。
- 复核测试真实性：确认 `TestRecoveryRootAssemblyWiresRuntime` 是**全包唯一**调用生产 `newRecoveryRootAssembly` 的测试（grep 全 `*_test.go` 仅 `recovery_runtime_test.go:731` 一处）；`recoveryOpenJournal`/`recoveryLoadKeyring` 是生产 seam 变量（`recovery_runtime.go:287/294`，非为本测试新增）；测试用 `t.Cleanup` 还原。
- 复核 F010 真实性：确认未使用测试本地 authorizer、未用 stub authority；走 `identity.NewRecoveryRunStore` → `identity.NewRecoveryIdentityExecution` → `identity.NewRecoveryIdentityForwarder().Seal` → 生产 `newRecoveryControlAuthorizer`，真实 PG 种子行（installation/users/audit/recovery_runs/restore_bindings/sessions）。
- 静态核对 F009 缺陷可被新测试捕获：构造函数返回后断言 `assembly.runtime != nil`，若删去回填则该断言必失败（红色可证明性为推理证明，见第 7 节限制）。
- 秘密扫描：`lead-gate-r05` 全目录、`s416-independent-review-r02` 目录与本轮自产日志，按“DSN 凭据 / 数据库用户 / 口令参数 / API key”四类模式扫描 → **0 命中**（为避免产物自身触发扫描，模式字面量不写入本报告）。

### Round 2（第一手重跑与归因）
- 第一手运行 build / vet（普通+integration）/ gofmt(15 文件) / F009 普通+race / F010 普通+race / scope 普通+race / ProductionChain race / 整包 unit 普通+**race** / 全量 integration，并自行解析 JSON 事件计数（见第 3 节）。
- 独立归因全量 integration 唯一顶层失败：读 `cmd/han-server/recovery_restore_integration_test.go:38`（硬编码 Host=127.0.0.1、Port=51934、Database=han_b13d_restore，否则 `t.Fatal("explicit authorized RestorePG required")`），该文件 mtime 2026-10-29 21:15（早于 B13H 起始），不在 15 文件范围。
- 扫描本轮自产日志中的 DSN/密钥 → **0 命中**（DSN 仅存在于子进程环境变量，未落盘）。

## 3. 实际执行命令与结果（本轮第一手）
| # | 命令 | 退出 | 结果 | 日志 |
|---|---|---|---|---|
| C1 | `go build ./...` | 0 | 无输出 | r03-cmd-build.log |
| C2 | `go vet ./cmd/han-server ./internal/identity ./internal/recovery` | 0 | 无输出 | r03-cmd-vet.log |
| C3 | `go vet -tags=integration ./cmd/han-server` | 0 | 无输出 | r03-cmd-vet-integration.log |
| C4 | `gofmt -l <15 文件>` | 0 | 0 行输出 | r03-cmd-gofmt.log |
| C5 | `go test ./cmd/han-server -run ^TestRecoveryRootAssemblyWiresRuntime$ -count=1 -json` | 0 | 顶层 pass=1 fail=0 skip=0 | r03-cmd-f009.log |
| C6 | `go test -race ./cmd/han-server -run ^TestRecoveryRootAssemblyWiresRuntime$ -count=1 -json` | 0 | 顶层 pass=1 fail=0 | r03-cmd-f009-race.log |
| C7 | `go test -tags=integration ./cmd/han-server -run ^TestRecoveryControlAuthorizerSealedMaintenance$ -count=1 -json` | 0 | 顶层 pass=1 fail=0（真实 PG） | r03-cmd-f010.log |
| C8 | `go test -tags=integration ./cmd/han-server -run ^TestRecoveryMaintenanceAuthorizerScope$ -count=1 -json` | 0 | 顶层 pass=1（含 3 子测试）fail=0 | r03-cmd-scope.log |
| C9 | `go test -race -tags=integration ./cmd/han-server -run ^TestRecoveryControlAuthorizerSealedMaintenance$|^TestRecoveryMaintenanceAuthorizerScope$ -count=1 -json` | 0 | 顶层 pass=2（含 3 子测试）fail=0 | r03-cmd-f010-scope-race.log |
| C10 | `go test -race -tags=integration ./cmd/han-server -run ^TestRecoveryRuntimeProductionChain$ -count=1 -json` | 0 | 顶层 pass=1 fail=0 | r03-cmd-productionchain-race.log |
| C11 | `go test ./cmd/han-server ./internal/identity -count=1 -json` | 0 | 顶层 pass=418 fail=0 skip=0 | r03-cmd-unit-packages.log |
| C12 | `go test -race ./cmd/han-server ./internal/identity -count=1 -json` | 0 | 顶层 pass=418 fail=0 skip=0；`DATA RACE` 0 次 | r03-cmd-unit-race.log |
| C13 | `go test -tags=integration ./cmd/han-server -count=1 -json` | 1 | 顶层 pass=168 fail=1 skip=0；唯一顶层 FAIL = `TestRecoveryRestoreTargetReadbackPostgres` | r03-cmd-full-integration.log |
| C14 | `Get-FileHash` 15 文件 × manifest（SHA256+bytes） | 0 | ALL15_MATCH=True（15/15） | （内联输出，见 §0） |
| C15 | 秘密扫描（r05 / r02 / 本轮日志） | 0 | 0 命中 | （内联输出） |

- 说明：Lead 的 A3–A6 记 pass=2 是把 package 级 pass 事件一并计入；本轮按“顶层 Test 事件”计为 1，两者不矛盾。
- C13 的唯一失败与 Lead A10 一致，且失败输出全部为 `explicit authorized RestorePG required`（7 条 error 输出均属于该测试的子测试），**无任何 B13H 测试失败**。
- 不把 C1–C4 的构建/静态检查成功当作测试或 E2E 通过；C13 的 exit 1 已如实记录。

## 4. 原 finding F001–F010 逐条结论
### F001 — CLOSED（维持） — exitRouter→composition，revoke/settle 可达
- 当前证据：`recovery_runtime.go:540` `Exit: exitRouter`；`:586` `exitRouter.bind(composition)`；`:368–379` 未绑定即 `ErrAdmissionBinding`，重复 bind 为 `ErrAdmissionBusy`；`:89–111` `capture(PhaseRecoveryReplay)` → `ports.Exit.ExitRecovery(bounded)` → `bridge.revokeSessions` → `ports.Article.settle`，`exitOnce` 保证一次（`:83–86`）。
- 本轮文件未改该段（`recovery_runtime.go` 仅 +1 行 F009 回填；差异集已证明）。全包 unit 普通/race 均绿。

### F002 — CLOSED（维持；覆盖由 F010 补齐）
- 当前证据：生产工厂 `recovery_runtime.go:956–971` `newRecoveryControlAuthorizer` → `identity.NewRecoveryMaintenanceStore` → `platform.NewHTTPAuthorizer`；装配 `:1007` 一带 `composeRecoveryControl: newProductionRecoveryControlHandler`（`main.go:3930`）。
- 覆盖升级：F010 新集成测试直接调用生产工厂并取得真实 maintenance 授权成功（见 F010），消除了 r02 记录的“生产 sealed 成功路径未执行”限制。

### F003 — CLOSED（维持） — wrapper 真正被 Serve，join 可等 done
- 当前证据：`recovery_runtime.go:685` `bindControl`；`:396–424` `recoveryJoinedServer.Serve` 关闭 `done`、`join` 先 Shutdown 再等 `done`/ctx；`main.go:705` bindControl、`:712–716` 注册 **wrapper** 进 `additionalServers`、`:743` 传入 lifecycle。`lifecycle.go` SHA 未变。

### F004 — CLOSED（维持） — Article join 只等已启动持有者
- 当前证据：`recovery_runtime.go:904–940`；`:919–927` `recoveryStartedWorker` 断言 + 非阻塞 `Started()` 检查，未启动 `continue`；`:928–936` 已启动则等 `done`/ctx。`main.go:610–620` 经 `recoveryStartedWorkerFrom` 强制真实 worker 暴露 `Started()`。

### F005 — CLOSED（维持，附观察） — normal/resume/verify 使用预算
- 当前证据：`recovery_runtime.go:744–752` verify bounded；`:754–763` normal bounded `RunNormal`；`:765+` resume bounded；`:100–104` composition exit 用 `JoinBudget` bounded。
- 观察（非 finding，与 r02 一致）：resume 的两个 goroutine 共享同一 bounded ctx，任一先返回会 cancel 对方；`flowOnce` 归一化首结果，本轮未观察到可复现错误路径。

### F006 — CLOSED（维持） — composition 用 bounded 调 surface exit
- 当前证据：`recovery_runtime.go:100–104`。

### F007 — CLOSED（维持） — 无硬编码 phase 预算
- 当前证据：`recovery_sources.go:24`（`phaseBudget` 字段）、`:51–58`（构造注入校验 `phaseBudget <= 0` 拒绝）、`:280`（`context.WithTimeout(ctx, b.phaseBudget)`）。该文件本轮 SHA 未变。

### F008 — CLOSED（维持） — complete 先响应再推进
- 当前证据：`internal/identity/recovery_handler.go:426–460`：`:434–439` `CheckComplete` 预检（closed 拒绝，不返回 200）；`:443–451` 写 200 + `Flush`；`:454–456` `RequestComplete` 失败结构化记录（响应已发出）。该文件本轮 SHA 未变。

### F009 — CLOSED（本轮修复项） — 生产 assembly 回填 runtime
- 原缺陷：`newRecoveryRootAssembly` 未给 `assembly.runtime` 赋值，normal/verify/resume 启动即 `ErrAdmissionBinding`。
- 修复证据（现场）：`cmd/han-server/recovery_runtime.go:596` `assembly.runtime = runtimeValue`，紧随 `:597` `return assembly, nil`；文件 +33 字节与单行增量吻合，无其他生产逻辑改动。
- 测试证据（现场全文核对，`cmd/han-server/recovery_runtime_test.go:719–770`）：
  - 真实调用生产构造函数 `:731`，输入来自生产 `planRecoveryStartup` 的 verify plan（`:727`）与真实 `preparedLoginDependencies`；`ensure plan.present && plan.verify`（`:728`）。
  - 断言 `assembly.runtime != nil`（`:743`）。
  - 断言与 composition 单一实例相同：`assembly.composition.runtime()` 非 nil 且 `== assembly.runtime`（`:746`）。
  - 断言 `assembly.bindWorkers(nil,nil,nil)` 在 verify plan 下成功（`:749`），即 F009 的直接症状消失。
  - 分发证据：换入 probe（`:754–755`）后 `assembly.verify` 使 `Verify` 计数=1（`:756–761`），`plan.verify=false / action="normal"` 后 `assembly.start` 使 `RunNormal` 计数=1（`:762–769`），证明 verify/start 经 `a.runtime`（源码 verify `:751`、normal `:762`）而非旁路。
- 是否测试专用绕过 / 只测 fixture：**否**。构造函数、plan、prepared 依赖均为生产代码；仅 journal/keyring 用生产 seam 变量注入（`recovery_runtime.go:287/294`，`t.Cleanup` 还原）；probe 只用于分发断言，不影响“构造函数回填”这一被断言事实（该断言在换入 probe 之前）。
- 回归可捕获性：全包仅此一处调用生产构造函数；删除 `:596` 会同时令 `:743`/`:746`/`:749` 失败（推理证明；未见持久化红色日志）。
- 第一手运行：C5、C6 均 exit 0，顶层 pass=1；C11/C12 整包（普通与 race）无回归。

### F010 — CLOSED（本轮修复项，附残余限制） — 生产 sealed 成功路径 + 真实 `CheckLockedMaintenanceUser`
- 位置：`cmd/han-server/recovery_runtime_integration_test.go:571–769`。
- 真实性核对（逐项）：
  - **真实 PostgreSQL**：`recoveryChainOpen`（`:574`）连接隔离 harness；真实建表/种子行 `:630–696`（installation、users、两条 audit、recovery_runs、restore_bindings、`access_scope='recovery_maintenance'` 的 sessions）。
  - **真实执行体**：`identity.NewRecoveryRunStore`（`:698`）、`identity.NewRecoveryIdentityExecution`（`:710–721`）、`identity.NewRecoveryIdentityForwarder().Seal(execution)`（`:725–728`）。
  - **真实生产工厂**：`newRecoveryControlAuthorizer(forwarder, gate, clock)`（`:731`）——不是测试本地 authorizer。
  - **成功路径**：`authorizer.Protect(session.AccessCurrentIdentity, probe)` 对 maintenance 会话返回 **204**（`:739–748`）。
  - **非无条件放行**：将运行时 permit 的 `Generation` 改为 `generation+1` 后同一 authorizer 返回 **401**（`:753–765`），并要求 `runtime.reads > 0`（`:766–768`），证明真实 `CheckLockedMaintenanceUser` 的比较逻辑被执行，而非 stub 恒真。
- 与 r02 的 F010 对比：r02 指出的两个缺口（生产工厂 sealed 成功路径、真实 `CheckLockedMaintenanceUser` 成功返回）均已由本测试覆盖；不再存在“stub authority 冒充 sealed 成功”。
- 残余限制（**非阻断，不另立 finding**）：permit 来源仍是测试 fixture `recoveryTestRuntime{binding: …}`（`:702–709`），即真实 `internal/recovery` Runtime 的 permit 读取语义（密封 permit → binding 投影）未被执行。该缺口与“生产 runtime 在 Windows 上为合法 closed 路径、真实 anchor/keyring/restore 未交付”同源，无法在本环境闭合；已记录于第 7 节，建议在恢复运行时真实可启动的迭代补一条走真实 Runtime 的 permit 读取断言。
- 第一手运行：C7、C9 exit 0，顶层 pass 分别 1 / 2（含 scope 3 子测试）。

## 5. 新增 findings（F011 起）
**无。** 本轮未发现新的阻断或非阻断缺陷；F010 的 fixture permit 绑定属已声明残余限制（见 §4 F010 与 §7），不构成本轮新 finding。

## 6. 重点核对项独立结论
| # | 核对项 | 结论 | 关键依据 |
|---|---|---|---|
| 1 | closed 适配器不是 no-op | PASS（文件未变，语义复核） | 未登记端口注入 closed 实现：`recovery_runtime.go:498/531/538/545/558` 等返回 `ErrAdmissionClosed`；本轮未改这些段 |
| 2 | verify 零 listener/worker 且早于监听 | PASS | `main.go:632–637` verify 在任何 `listen`（`:639`）之前并直接 return；`bindWorkers`（`:633–641`）在 `plan.verify` 时直接 return nil，不绑 worker |
| 3 | complete 先真实响应再由 root 独立控制 | PASS | `recovery_handler.go:443–456` 写响应+Flush 后才 `RequestComplete`；root 控制流独立 goroutine（r01/r02 已核，文件未变） |
| 4 | 三 server 真 join + lifecycle 收集/兜底 + 拒 nil | PASS | `main.go:705/712–716/743`；`lifecycle.go:306–380` 收集所有 served server 的 Shutdown 结果并对失败项 Close；`:582–586` 拒绝 nil/不完整 additionalServer |
| 5 | normal/verify 不因未使用的安装来源配置而拒绝 | PASS | `recovery_config.go` 本轮 SHA 未变；门控逻辑 r01 已核；C5/C6 的 verify plan 构造成功 |
| 6 | 单一 Article handle Close/join；deployment 归还后关 pool | PASS | `main.go:607–627` 强制 close/join owner；`recovery_runtime.go:904–940` 仅 join 已启动者；lifecycle 关序（r01，lifecycle.go 未变） |
| 7 | cmd 内无复制身份 SQL/authority；root 不持 key/HMAC | PASS（文件未变段） | `newRecoveryControlAuthorizer` 仅委托 `internal/identity` + `platform`；本轮变更未引入 SQL/密钥常量 |
| 8 | 15 文件边界、无第 16 文件、internal/recovery 未动 | PASS | ALL15_MATCH=True；差异仅 3 文件；扫描=0；mtime 16:43:50 |
| 9 | 测试真触发根因 / PG 真用库 | PASS | F009 走生产构造函数并断言回填与分发；F010 真实 PG + 真实执行体 + 真实 authority 正/负路径；C11–C13 计数与失败归因 |

## 7. 未验证限制与剩余风险
- 生产 CAS/anchor、installation commitment、source verifier、deployment lease、keyring loader、restore adapter、真实身份 provider/ACL 与真实部署 E2E 仍未交付；recovery 仍显式 closed，本报告不对其给 PASS。
- Windows `NewFileRecoveryJournal` 的 `ErrUnsupportedPlatform` 仍是合法 closed 路径；本轮未绕开，也未新增 seam。
- F009 的红色证据仅在实施者会话被观察到，**未持久化为文件**；本轮以其“代码缺失则断言必失败”的结构推理替代，属限制而非通过证据。
- F010 的 permit 绑定为 fixture（`recoveryTestRuntime`）；真实 `internal/recovery` Runtime 的 permit 读取未执行。
- `TestRecoveryRuntimeProductionChain` 仍使用 fixture executor，不是真实 provider E2E。
- `goimports` / `golangci-lint` 未运行；仅 `gofmt -l` 与 `go vet`。
- 设计仍为 draft；人的精确接纳与 Lead 接收状态 unknown。
- 全量 integration 的 1 条顶层失败为既有 S4.14 环境前置（`recovery_restore_integration_test.go:38` 要求 127.0.0.1:51934/han_b13d_restore，harness 为 53519），**非 B13H 回归**；本轮未修改该测试。
- DSN 仅存在于子进程环境变量，未写入本报告、日志或仓库文件。

## 8. 声明
- 本结论由独立 Reviewer 基于现场只读证据给出，**不代替人工业务接纳、Lead 接收裁决或任务关闭**。
- 未修改任何产品源码、tasks.md、design、test-plan、manifest、 `.codex`；未执行 git add/commit/push；未删除/移动/清理；仅在本目录写入报告/回执/日志。
- 复审条件：若后续再次改动 15 文件中的任一文件，需重新固定指纹并重跑本报告第 3 节命令。
