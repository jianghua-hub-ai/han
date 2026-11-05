# Han S4.16 / B13H 独立代码评审报告（r02 复审）

## 0. 对象与修订指纹
- objectType: code（两轮独立评审）
- worktree: D:\AID\VSCode\Codex\Han；branch: main；HEAD: 6da8abbceb0fa197364810a92a8fa2d97b38ddca
- 目标修订：当前未提交工作树，以 `.spec/wip/recovery-runtime/evidence-b13h-r01/lead-gate-r04/files-sha256-r02.json` 的 15 个文件 SHA256 固定。
- 现场复核：15/15 文件 SHA256 + 字节数与该 manifest 完全一致（ALL15_MATCH=True），无漂移。
- 边界：`internal/recovery` 最新 mtime 仍为 2026-10-30 16:43:50（未动）；本轮无第 16 文件；未执行任何 git 写操作。
- 上游依据现场 SHA256 与本轮一致：design.md `0FC3DE78…A19277`、test-plan.md `34826759…921DAB`、tasks.md `8BE08873…6B350`。
- 与 r03 相比仅 8 文件变化；其余 7 文件 SHA 不变（见下表标注）。
- 评审者真实 Agent ID：`01a18e02-451f-78c1-89d2-dea8b9af0f63`（add-reviewer）。作者 add-backend-dev `01a18d45-b04b-7581-a172-0296b0341ccb`，其自述未作为验证证据。

### 15 文件指纹（对照 r04 manifest）
| # | 文件 | SHA256 | bytes | 相比 r03 |
|---|---|---|---|---|
| 1 | cmd/han-server/recovery_runtime.go | 8B753A14…AABFEB7 | 33773 | 变化 |
| 2 | cmd/han-server/recovery_runtime_test.go | C23E4638…B1564F6 | 22948 | 变化 |
| 3 | cmd/han-server/recovery_config.go | 83FFC39F…E606B0 | 9842 | 未变 |
| 4 | cmd/han-server/recovery_config_test.go | 9071DCAB…A095E | 2959 | 未变 |
| 5 | cmd/han-server/recovery_command.go | 4BCF661D…123F3 | 2110 | 未变 |
| 6 | cmd/han-server/recovery_command_test.go | 2FA4BFA0…4FB17A | 2326 | 未变 |
| 7 | cmd/han-server/recovery_sources.go | 935DD7BD…DF90C | 24136 | 变化 |
| 8 | cmd/han-server/recovery_sources_test.go | BBCB4735…188EF4 | 14797 | 变化 |
| 9 | internal/identity/recovery_handler.go | 4754B880…5A00E52 | 25057 | 变化 |
| 10 | internal/identity/recovery_handler_test.go | 6643613F…BEA2115 | 42139 | 变化 |
| 11 | cmd/han-server/recovery_runtime_integration_test.go | E4BFA62E…4616BB2 | 21992 | 变化 |
| 12 | cmd/han-server/main.go | 3050CE4E…D32EE97 | 169631 | 变化 |
| 13 | cmd/han-server/main_test.go | 0E61F4A3…185820 | 399522 | 未变 |
| 14 | cmd/han-server/lifecycle.go | 0CA9C339…88F663 | 21788 | 未变 |
| 15 | cmd/han-server/lifecycle_test.go | 33D641DF…E4F05 | 45066 | 未变 |

## 1. 结论摘要
- **CodeGate：CHANGES_REQUIRED**（不代替人工接纳、Lead 接收或任务关闭）。
- 原 finding 复核：F001–F008 全部 **CLOSED**；其中 F002 附 LIMITATION（生产 sealed 成功路径未覆盖，另立 F010）。
- 新增 finding：**F009（HIGH，阻断）**、**F010（LOW，测试深度）**。
- 阻断理由（F009）：生产构造函数 `newRecoveryRootAssembly` 从未把已构造的 Runtime 赋给 `assembly.runtime`；`bindWorkers`/`verify`/`start` 均因 `a.runtime == nil` 在启动期返回 `ErrAdmissionBinding`。即 normal/verify/resume 三种恢复子命令在生产路径上全部 fail closed、整个 recovery runtime 不可达；而现有测试无一构造生产 assembly，故未暴露。此为「失败安全但功能整体不可用」的高影响缺陷，属返工回归，必须修复并加覆盖。

## 2. 两轮实际检查范围
### Round 1（基线 + 边界 + 九项重点源码通读 + 生产调用链）
- 现场复核 15 文件 SHA256/bytes 与 r04 manifest；复核 `internal/recovery` mtime、HEAD、上游 design/test-plan/tasks SHA。
- 通读生产入口 `cmd/han-server/main.go:596-737` 的 recovery 组装、bind、verify、start、control surface、lifecycle 装配链。
- 逐条重读 `recovery_runtime.go` 的 assembly/composition/router/joinedServer/handle/join/authorizer 全段，及 `recovery_sources.go`、`recovery_workers.go`、`internal/identity/recovery_handler.go`、`internal/identity/recovery_auth.go` 相关路径。
- 对原 F001–F008 逐条定位 r04 修复点与对应测试。

### Round 2（定向复核 + 重跑验证）
- 重跑 build / vet / 定向单测 / 定向集成，独立解析完整 integration 计数。
- 对 F002 覆盖充分性做独立裁定（见 F002/F010）。
- 独立确认完整 integration 唯一失败为 S4.14 环境前置，而非 B13H 回归。

## 3. 实际执行命令与结果（本轮第一手，均只读）
| 命令 | 退出 | 结果 | 日志 |
|---|---|---|---|
| `Get-FileHash` 15 文件 × manifest 比对 | 0 | ALL15_MATCH=True | （内联） |
| `git rev-parse HEAD` / `internal/recovery` mtime | 0 | HEAD 6da8abb…；mtime 2026-10-30 16:43:50 | （内联） |
| `go build ./...` | 0 | 干净 | r02b-build.log |
| `go vet ./cmd/han-server` | 0 | 干净 | r02b-vet.log |
| `go vet -tags=integration ./cmd/han-server` | 0 | 干净 | r02b-vet-integration.log |
| `go test ./cmd/han-server -run <10 个 F001-F008 测试> -count=1 -json` | 0 | pass=19 fail=0 | r02b-unit-targeted.jsonl |
| `go test -tags=integration ./cmd/han-server -run ^(ProductionChain|MaintenanceAuthorizerScope)$ -count=1 -json` | 0 | pass=6 fail=0 skip=0（真实 PG 临时 schema） | r02b-integration-targeted.jsonl |
| 解析完整 integration（`r02-integration-full.jsonl`） | — | 顶层 pass=166 / fail=1 / skip=0 | r02-integration-full.jsonl |

- 完整 integration 唯一顶层失败：`TestRecoveryRestoreTargetReadbackPostgres`（Elapsed 0，未进主逻辑）。根因见 `cmd/han-server/recovery_restore_integration_test.go:38`：硬编码要求 Host=127.0.0.1、Port=51934、Database=han_b13d_restore，否则 `t.Fatal("explicit authorized RestorePG required")`；本 harness 为 53519。该文件不在 15 文件范围，属既有 S4.14 环境前置，**非 B13H 回归**。
- 说明：#06/#07 的 top-level PASS 与 ProductionChain/MaintenanceAuthorizerScope 是真实 PostgreSQL 链断言：测试打开并迁移真实临时 schema、真实 pgxpool、`session.NewStore`/`session.NewMaintenanceStore`、`identity.NewRecoveryReviewLister` 真查库；但认证主体/执行器多为 fixture。其语义深度另见 F002/F010。

## 4. 原 finding F001–F008 逐条结论（均为 CLOSED）
### F001 — CLOSED — Runtime 经 exitRouter 到 composition，revoke/settle 可达
- r04 证据：`recovery_runtime.go:540` `Exit: exitRouter`；`:586` `exitRouter.bind(composition)`；`:577` composition `Exit: assembly.exit`；router 未绑定即拒绝 `:388-390`；composition `exitRecovery` `:89-111` 执行 capture→`ExitRecovery`→`bridge.revokeSessions`→`Article.settle`，`exitOnce` 保证一次（`:83-86`）。
- 覆盖：`TestRecoveryRuntimeExitRouterReachesComposition`、`TestRecoveryRuntimeExitRouterRefusesBeforeBind` 通过。

### F002 — CLOSED（附 LIMITATION）— maintenance 控制面改用 maintenance store/authorizer
- r04 证据：`newRecoveryControlAuthorizer` `recovery_runtime.go:955-968` → `identity.NewRecoveryMaintenanceStore`（`internal/identity/recovery_handler.go:461-475`，解封 forwarder、要求 `*recoveryIdentityExecution`）→ `identity/recovery_auth.go:167-172` 用同一 pool + Authority 调 `session.NewMaintenanceStore` → `platform.NewHTTPAuthorizer`；生产装配 `:1011`。
- 覆盖：`TestRecoveryControlAuthorizerRequiresSealedExecutor`（`recovery_runtime_test.go:670-677`）证明未密封拒绝；集成 `TestRecoveryMaintenanceAuthorizerScope`（`recovery_runtime_integration_test.go:423-560`）真用 PG，证明 maintenance scope 通过、ordinary 拒绝、写操作需 CSRF。**但**该集成用测试本地 `recoveryChainControlAuthorizer`（`:407-417`）+ stub `recoveryChainMaintenanceAuthority`（`:382-402`/`:501`），未走生产 `newRecoveryControlAuthorizer` 的真实 sealed executor 成功路径。
- 独立裁定：根因（用普通 store 拒绝 maintenance cookie）已在实现层修正，且集成测试用的是与生产工厂同名的 `session.NewMaintenanceStore`，存储层 SQL 语义得到真实 PG 证明；唯一残留是「真实 `*recoveryIdentityExecution` 作为 Authority 的成功路径」与「真实 `CheckLockedMaintenanceUser` 成功返回」未被执行。故实现层 CLOSED，覆盖不足另立 **F010（LOW）**，不作为实现未修复。

### F003 — CLOSED — wrapper 真正被 Serve，join 可等 done
- r04 证据：`bindControl` `:684-706` 创建并返回 `*recoveryJoinedServer` 并 `exit.bind(joined.join)`；`main.go:712-716` 把 wrapper（非原始 server）注册进 `additionalServers`；wrapper `Serve` `:408-411` 关闭 `done`；`join` `:413-424` Shutdown 后等 `done` 或 ctx。
- lifecycle 侧未变（`lifecycle.go` SHA 未变），r01 已确认收集 served server Shutdown 错误并在失败时 Close 兜底、拒绝 nil/incomplete additionalServer。`TestRecoveryControlReleaseIsJoined` 通过。

### F004 — CLOSED — Article join 只等已启动持有者
- r04 证据：`newRecoveryArticleJoin` `:905-939`；对每个 worker 先做 `recoveryStartedWorker` 断言（`:918-921`），再做非阻塞 `Started()` 检查，未启动即 `continue`（`:922-926`）。生产侧 `main.go:610-620` 通过 `recoveryStartedWorkerFrom`（`recovery_runtime.go:1035-1044`）强制真实 worker 暴露 `Started()`，因此真实 worker 会命中断言、按启动态正确等待，不会误跳过已启动者。
- 覆盖：`TestRecoveryArticleJoinSkipsUnstartedOwners` 通过。

### F005 — CLOSED（附观察）— normal/resume/verify 使用 ioBudget
- r04 证据：normal `:759-761` 派生 `context.WithTimeout(ctx, budget)` 后 `RunNormal(bounded)`；resume `:765-768` 同样 bounded；verify `:747` bounded。`TestRecoveryRuntimeStartHonoursIOBudget` 通过。
- 观察（非 finding）：resume 的 `Resume` 与 `completeOnRequest` 两个 goroutine 共享同一 `bounded`/`cancel`，任一先返回即 cancel 对方；`Resume` 主体走 `r.generationCtx`（`internal/recovery/runtime.go:1547`），预检/beginAction 走传入 ctx。当前被 F009 掩盖、且 `flowOnce` 归一化首结果，未发现可复现错误路径，仅记为剩余风险。

### F006 — CLOSED — composition 用 bounded 调 surface exit
- r04 证据：`recovery_runtime.go:100-104` 用 `bounded` 调 `ports.Exit.ExitRecovery`。`TestRecoveryRuntimeCompositionExitUsesJoinBudget` 通过。

### F007 — CLOSED — 移除硬编码 phase 预算
- r04 证据：`recovery_sources.go:24` 增 `phaseBudget` 字段，构造函数 `:51-58` 由参数注入，使用点 `:280`；硬编码常量已移除。`TestRecoveryBridgePhaseBudgetIsConfigured` 通过。

### F008 — CLOSED — complete 先响应再推进，错误可观察
- r04 证据：`internal/identity/recovery_handler.go:434-439` 先 `CheckComplete` 预检（closed 时拒绝而非 200）；`:443-451` `WriteJSON(200)` + `Flush`；`:454-456` `RequestComplete` 失败结构化记录；bridge `recordCompleteFailure` `recovery_sources.go:233-243` 唤醒 root。`TestRecoveryBridgeFailedCompleteSignalWakesRoot` 与 handler 子测试通过。

## 5. 新增 findings（按严重度排序）

### F009 — HIGH — 生产 assembly 从未给 `runtime` 字段赋值，normal/verify/resume 全部启动即失败
- object/revision：`code` / 未提交工作树，`recovery_runtime.go` SHA `8B753A14…AABFEB7`（HEAD 6da8abb）
- 位置：`cmd/han-server/recovery_runtime.go:437`（字段 `runtime recovery.Runtime`）；构造函数 `:457`；结构体字面量 `:508` 只设 `plan/exit/flowDone`；`:528` 构造 `runtimeValue` 但仅用于 bridge/executor/reviews/composition，末尾回填 `:589-595` 只设 `bridge/maintenance/executor/reviews/composition/ordinary/article`，**独缺 `runtime`**；全包搜索 `assembly.runtime =`/`.runtime = runtimeValue` 零命中。
- 触发条件：任何 `recoveryPlan.present` 且 action 为 normal/verify/resume 的生产启动（`main.go:600-720`）。
- 影响：
  - normal/resume：`main.go:622` `assembly.bindWorkers` 因 `a.runtime == nil` 返回 `ErrAdmissionBinding`（`:633`），`main` 以 “bind recovery workers” 失败返回，业务面根本不 listen。
  - verify：`main.go:633` `assembly.verify` 因 `a.runtime == nil`（`:744`）返回 `ErrAdmissionBinding`，verify 永久失败。
  - 即使绕过前面，`start`（`:754`）同样因 nil 返回失败；`joinWork`（`:827-828`）对 nil runtime 直接返回 nil，导致流程 join 沦为 no-op。
  - 结论：整个 recovery runtime 在生产不可达；属「失败安全但功能整体不可用」。现有单测仅用结构体字面量手动设 `runtime`（如 `recovery_runtime_test.go:407-412`），无任何测试调用 `newRecoveryRootAssembly`，故缺陷未被发现。
- 最小修复建议：在构造函数 return 前补 `assembly.runtime = runtimeValue`（或在 `:508` 字面量中直接 `runtime: runtimeValue`，但需注意 `runtimeValue` 在 `:528` 才产生，宜在 `:589` 回填 `assembly.runtime = runtimeValue`）；或统一改为经 `a.composition.runtime()` 取 Runtime，消除双源。
- 验证方式：新增测试用 `newRecoveryRootAssembly`（可用真实 PG + 合法 plan，或最小可构造输入）构造 assembly，断言 `bindWorkers`/`verify`/`start` 在正常输入下不再返回 `ErrAdmissionBinding`；并补一条集成测试走生产 assembly 的 verify 与 normal 启动路径。

### F010 — LOW — F002 生产 sealed 成功路径与真实 `CheckLockedMaintenanceUser` 成功路径未被任何测试执行
- object/revision：`code` / `recovery_runtime_test.go` SHA `C23E4638…B1564F6`、`recovery_runtime_integration_test.go` SHA `E4BFA62E…4616BB2`
- 位置：生产工厂 `cmd/han-server/recovery_runtime.go:955-968`、装配 `:1011`；单测仅拒绝 `recovery_runtime_test.go:670-677`；集成用测试本地 `recovery_runtime_integration_test.go:407-417` + stub `:382-402`；真实 `CheckLockedMaintenanceUser` 仅拒绝路径覆盖 `internal/identity/recovery_auth_test.go:77`。
- 触发条件：真实 sealed `*recoveryIdentityExecution` 作为 Authority，对合法 maintenance session 的成功授权。
- 影响：若真实 `CheckLockedMaintenanceUser` 对合法 maintenance 会话仍返回错误，或生产工厂解封/装配出现偏差，现有测试全部仍绿，属「实现静态正确但未被端到端证明」的残余风险（叠加 F009 当前不可达）。
- 最小修复建议：新增集成测试，用真实 PG 构造 sealed `*recoveryIdentityExecution`（或 seal 生产 forwarder），调用 `newRecoveryControlAuthorizer` 得到 authorizer，对 `recovery_maintenance` 会话断言 `Protect` 返回 2xx；并覆盖一次真实 `CheckLockedMaintenanceUser` 成功返回。
- 验证方式：上述集成测试真实运行通过（非 stub Authority）。

## 6. 九项重点核对（独立结论）
| # | 核对项 | 结论 | 关键依据 |
|---|---|---|---|
| 1 | closed 适配器非 no-op | PASS（结构性，文件语义未变） | 未登记端口返回 `ErrAdmissionClosed/ErrAdmissionBinding`；r04 未改这些 adapter 语义；`recovery_runtime.go:465/498/523/531/538/545/558` 等均注入 closed 实现而非成功桩 |
| 2 | verify 零 listener/worker/repair 且早于监听 | 结构 PASS，实际被 F009 阻断 | `main.go:632-637` verify 在任何 listen 前返回；`bindWorkers:636-638` verify 不绑 worker。但 `a.runtime==nil` 使 verify 恒失败（F009） |
| 3 | complete 先真实响应再 root 独立控制 | PASS | `recovery_handler.go:443-456`；root 控制 `recovery_runtime.go:782-795` 在独立 goroutine |
| 4 | 三 server 真 join + lifecycle 收集/兜底 + 拒 nil | PASS（lifecycle 未变，r01 维持） | `main.go:705/712-716` wrapper 注册；`lifecycle.go` SHA 未变；F003 修复闭合 |
| 5 | normal/verify 不因未使用来源配置拒绝 | PASS | `recovery_config.go` 未变（门控逻辑 r01 已核） |
| 6 | 单一 Article handle Close/join，deployment 归还后关 pool | 结构 PASS，join 实际被 F009 弱化 | handle `:188-262`；`main.go:727-728` 与 lifecycle 关序（r01）；`joinWork:827-828` 对 nil runtime 直接 nil |
| 7 | cmd 无复制身份 SQL/authority；root 不持 key/HMAC | PASS | `recovery_runtime.go` 无身份 SQL/authority 复制，仅复用生产构造器；main.go HMAC 引用为既有 keyring 配置名，非 root 持密钥 |
| 8 | 15 文件边界，无第 16 文件，internal/recovery 未动 | PASS | ALL15_MATCH=True；mtime 16:43:50；git 现状未变 |
| 9 | 测试真触发根因 / PG 真用库 | 部分 PASS | 集成真用 PG；但生产 assembly 构造路径零覆盖（F009）、F002 生产成功路径未覆盖（F010） |

## 7. 未验证限制与剩余风险
- 生产 CAS/anchor/commitment/source verifier/deployment lease/keyring loader/restore adapter、真实 provider/ACL、真实部署 E2E 仍未交付；recovery 明确 closed。
- Windows `NewFileRecoveryJournal` 的 `ErrUnsupportedPlatform` 仍是合法 closed 路径；本轮未绕开。
- 设计仍为 draft；人的精确接纳 unknown；Lead 接收未知。
- `goimports`/`golangci-lint` 未运行；`gofmt -l` 由 Lead r04 补检无输出，本轮未重跑 gofmt。
- 完整 integration 1 条失败为既有 S4.14 环境前置（51934/han_b13d_restore），非 B13H 回归；本轮未修改该测试。
- F009 使 normal/verify/resume 全路径不可达，任何「运行期行为」结论（join 预算、settle 时序、authorizer 成功授权）在当前修订上均无法端到端验证。
- DSN 仅存在于进程环境变量，未写入任何报告/日志/仓库文件。
- 不将 build/vet 成功等同于测试或 E2E 通过；不声称未验证范围已通过。

## 8. 声明
- 本结论由独立 Reviewer 基于现场只读证据给出，**不代替人工业务接纳、Lead 接收裁决或任务关闭**。
- 复审条件：修复 F009（并补生产 assembly 覆盖）后重跑定向与集成；F010 建议同批补生产 sealed 成功路径集成测试。
- 未修改任何产品源码、tasks.md、design、test-plan、manifest、.codex；未 git add/commit/push；未删除/移动/清理。
