# identity-auth-S4.19 独立 Code Review 报告（R01）

**最终结论：不通过（FAIL）**

- 代码接线本身未发现阻断性缺陷；ExactRouteBeforeGuard、ClosedUnknownTwoAdminGets503、LiveReadyProbes（有 runtime 部分）、SingleRuntimeHandle 的静态与测试证据成立。
- 但 `OpenAfterCompleteSoftwareRecoverySameSession` 与 `NoRecoveryConfig` 两个 leaf 的实际证据没有覆盖任务卡要求的完整行为，见 `S419-CR-F001`、`S419-CR-F002`。任务卡关闭条件因此不成立。

## 1. 固定对象与修订

| 项目 | 值 |
|---|---|
| 仓库 | `D:\AID\VSCode\Codex\Han` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 评审角色 | add-reviewer（非作者独立评审，未参与实施） |
| 评审时间 | 2026-10-31（Asia/Shanghai） |
| 唯一写入目录 | `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r01/` |

| 评审对象 | SHA256（实际复算） | 字节 | Git 状态 |
|---|---|---|---|
| `cmd/han-server/main.go` | `B012EAE8814BD47B16797A277AAB2E0CDA7C1885C839A8EE9D5435D63E3F03B8` | 171424 | 未跟踪（`?? cmd/`） |
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `173BEEBC15AB43ADCD42536C6AB71BCBE1A66C5B2D59820EF794BDA6097142E6` | 17654 | 未跟踪（`?? cmd/`） |

两个 SHA256 与委派输入完全一致，未触发“对象变化即停止”条件。`git ls-files --error-unmatch cmd/han-server/main.go` 返回 exit 1，确认 `cmd/` 整棵目录不在当前 HEAD 的 Git 索引中，因此本次没有可用的 `git diff` 基线。

## 2. 两轮实际检查范围

### Round 1：变更快速审查

- 因两个对象均未跟踪，无法生成 commit diff。采用“全文阅读 + 变更符号定位 + 任务卡 R 对账”的方式：
  - 逐个核对任务卡 `tasks.md:2802-2806` 列出的未建成/可复用/不足证据三项。
  - 定位本卡新增/改动的实现符号：`newLoginBusinessRouterWithTraceAndRecovery`（`cmd/han-server/main.go:3497`）、`recoveryGuardFromRuntimeHandle`（`main.go:3665`）、`businessComposition.recoveryHandle` 的注入（`main.go:538-544`、`main.go:552-579`、`main.go:2419-2424`）以及新测试文件全部 505 行。
  - 核对旧 `newLoginBusinessRouterWithTrace` 的委托路径（`main.go:3489-3495`）与全部调用方（`cmd/han-server/main_test.go` 中 11 处 `newLoginBusinessRouter` 调用；生产路径只有 `newProductionBusinessHandler`）。
  - 核对写集：除两个授权对象外的 `cmd/han-server` 与 `internal/platform` 文件 mtime 均早于本次实施窗口（详见 §8）。

### Round 2：whole-repo 深度审查

- 生产消费者链：`main` 启动（`main.go:163-181`、`:538-544`、`:552-579`、`:606-618`）→ `newProductionBusinessHandler`（`:2385-2425`）→ exact router/Trace/guard（`:3497-3663`）。
- 只读依赖：`internal/platform/recovery_guard.go:17-83`（guard 与 probe 语义）、`cmd/han-server/recovery_runtime.go:528-597`（单一 runtime 构造与装配）、`cmd/han-server/recovery_config.go:88-123`（present 判定）、`cmd/han-server/recovery_runtime_test.go:719-769`（assembly/composition 同一 runtime 既有证据）。
- 上游合同：`architecture.md:1379/1381`（公共 503+Trace、下游零调用、probe 语义）、`design-workspace/identity-auth/design.md:3904`（唯一 exact registry、livez/readyz 固定响应、405/Allow、404、不泄漏）。
- 测试真实性：`recovery_main_dispatcher_integration_test.go:299-363` 是否真实走 `prepareProductionLoginDependencies` + `productionBusinessComposition` + `newProductionBusinessHandler`；是否存在 `t.Skip`、`http.ServeMux` fixture、拒启代证、只断言接口名（结论：均无）。
- 状态与兼容：closed/unknown/maintenance/verify/public_ready、404/405/Allow、Origin/CSRF/Session/AccessGeneral、panic 后 lease 释放、晚绑定、第二 runtime/authority/fallback。
- 安全与交付：敏感信息、硬编码、死代码、格式化、写集越界、证据完整性。

### 未执行的范围

- 未运行 PostgreSQL、真实 HTTP 服务、Restore、SMTP、任何 E2E；未连接真实 provider/CAS/ACL/POSIX。
- 未对被审对象做任何修改、格式化或 Git 写操作。

## 3. 逐 leaf 结论

| 必要风险 leaf | 结论 | 主要证据 |
|---|---|---|
| `ExactRouteBeforeGuard` | **PASS** | 测试 `recovery_main_dispatcher_integration_test.go:34-69`；实现 `main.go:3515-3662` |
| `ClosedUnknownTwoAdminGets503` | **PASS（带证据保真度备注）** | 测试 `:71-108`；`recovery_guard.go:27-47` |
| `LiveReadyProbes` | **部分 PASS / 无配置子案例未闭合** | 测试 `:110-148`、`:275-297`；实现 `main.go:3632-3640`、`recovery_guard.go:51-65` |
| `OpenAfterCompleteSoftwareRecoverySameSession` | **不通过（证据不足）** | 测试 `:150-234` 仅直接 `setState(public_ready)` |
| `SingleRuntimeHandle` | **PASS（静态证据 + 测试间接证据）** | 测试 `:236-273`；`main.go:538-544/616-618`；`recovery_runtime.go:528-597`；既有 `recovery_runtime_test.go:719-769` |
| `NoRecoveryConfig` | **不通过（验收分支未覆盖）** | 测试 `:275-297` 直接注入 `return nil` provider，未调用 `planRecoveryStartup` |

### 3.1 ExactRouteBeforeGuard（PASS）

- 实现顺序正确：`switch` 先完成 exact path/method 识别（`main.go:3515-3622`），未知路径 `default` 直接 404（`:3615-3621`），已知路径错 method 直接 405 + `Allow`（`:3623-3631`），probe 在 exact route 之后（`:3632-3640`），普通业务才进入 `guard.Protect`（`:3649-3652`）。
- `platform.TraceMiddleware` 位于最外层（`:3662`），trace 回调包装在 exact router 之外（`:3655-3661`），因此 404/405/503 均带 Trace。
- 测试在 closed 状态断言：未知路径 404 且带 Trace；`POST /api/v1/admin/articles` 405 + `Allow: GET`；`DELETE /api/v1/articles` 405 + `Allow: GET, POST`；`runtime.snapshot()` 的 acquire 为 0；事件只有 `trace`。这直接证明 guard 没有在路由判定前取得 lease，也没有把 404/405 改写成 503。
- F008（guard 必须在 exact router 内晚绑定、不得外层包装）已按建议实现：guard 在 router 闭包内部、exact route 判定之后、`handler.ServeHTTP` 之前创建并调用（`main.go:3649-3652`），没有对整棵 router 做外层包装。

### 3.2 ClosedUnknownTwoAdminGets503（PASS，带备注）

- 测试对 `closed` 与 `unknown` 两个状态，分别请求 `GET /api/v1/admin/articles` 和 `GET /api/v1/admin/articles/{id}`，均断言 503、错误码 `platform.recovery_in_progress`、envelope Trace 与响应头 Trace 一致、无内部标记泄漏（`:71-108`）。
- 下游零调用断言覆盖 repository `lists/gets`、author reader `calls`、Session resolver `inputs`（`:93-101`），并断言事件只有 `trace`（`:102`），说明 guard 在 Session/AccessGeneral 与 Article 下游之前阻断。
- 测试走真实 `newProductionBusinessHandler` 与真实 Article query service/handler（`:299-363`、`main_test.go:1319-1367`），不是 `http.ServeMux` fixture，也没有拒启或 spy 代证。
- 备注：`unknown` 由 fake runtime 设置 `ErrAdmissionClosed` 表示（`:457-466`），不是真实 runtime 的 unknown 分类。`RecoveryGuard.Protect` 对任何 Acquire 错误一律 503（`recovery_guard.go:33-39`），因此行为等价，未发现实际代码风险；仅作为测试保真度的非阻断建议（N2）。

### 3.3 LiveReadyProbes（部分 PASS）

- 已覆盖：closed 下 `/livez` 200 `alive`、`/readyz` 503 `unavailable`；unknown/maintenance/verify 下 `/readyz` 503；`POST /livez`、`POST /readyz` 均 405 + `Allow: GET`；`/livezz`、`/livez/`、`/readyz/` 404；public_ready 下 `/readyz` 200 `ready`；probe 不 Acquire lease 且 readyz 确实调用共享 runtime 的 Verify；响应泄漏检查（`:110-148`、`:365-392`）。
- 零值 guard 分支经独立核对为 fail-closed 且语义正确：`NewRecoveryGuard` 失败时 `recoveryGuardFromRuntimeHandle` 返回零值 guard + `true`（`main.go:3675-3678`），业务分支因 `guarded=true` 进入 `Protect`，零值 runtime 触发 503（`recovery_guard.go:29-31`）；probe 分支使用零值 guard 时 `/livez` 恒 200 `alive`、`/readyz` 因 runtime 缺失恒 503 `unavailable`（`recovery_guard.go:51-65`）。因此 probe 分支忽略第二返回值不构成开放风险。
- 缺口：任务卡明确要求“无任何 recovery 配置（`planRecoveryStartup` 返回 `present:false`、无 runtime handle）”这一子案例。当前 `NoRecoveryConfig` 只注入 `func() recovery.Runtime { return nil }`，没有调用 `planRecoveryStartup`，也没有证明 `present:false` 是 production startup 的实际判定结果。见 `S419-CR-F002`。
- 非阻断备注：`/livez` 不授业务权的断言在 `:139-147` 先把状态重新设回 `closed` 再请求业务路由，因此它证明的是“closed 仍阻断”，而不是“刚才的 livez 200 没有留下授权状态”。见 N1。

### 3.4 OpenAfterCompleteSoftwareRecoverySameSession（不通过）

- 测试实际流程只有三步：closed 下请求 503 → `runtime.setState(public_ready)`（`:157`）→ 同一 `*http.Request`/Session 再请求 200（`:158`）。
- 没有执行任务卡要求的 `Begin → … → succeeded → 撤维护/真 join → 最终 proof → public open`；fake `s419RecoveryRuntime.setState` 直接改写 observation 与 `acquireErr`（`:457-466`），绕过了真实状态机、join、最终对账与 open 许可。
- 同一 Session 的 200、Trace、repository/reader 调用次数断言本身成立（`:158-170`），失败负控 session/origin/csrf 也真实走 production 链（`:172-233`）。但这只能证明“runtime 已处于 public_ready 时 dispatcher 放行”，不能证明“完整软件恢复完成后同一 Session 放行”。
- 既有 `TestRecoveryRuntimeProductionChain`（`recovery_runtime_integration_test.go:165`）是组件级链路，且需要真实 PG DSN（`:171-172`），并未在 public open 之后走 `cmd/han-server` 的 production dispatcher。两者相加仍缺少本卡要求的交叉证据。

### 3.5 SingleRuntimeHandle（PASS，带证据限制）

- 生产装配只有一个 runtime 值：`recovery_runtime.go:528-547` 构造一次 `runtimeValue`，同一值交给 bridge（`:551`）、identity execution（`:556`）、review lister（`:570`）、composition（`:575`），最后写入 `assembly.runtime`（`:596`）；`start` 使用 `a.runtime`（`:754-778`）。
- `main` 在 `plan.present` 为真时把 `assembly.runtime` 写入闭包变量（`main.go:610-618`），business handler 早于 assembly 组合，但拿到的是读取闭包的函数（`:540-544`、`:552-579`），因此是晚绑定而非第二 runtime。
- 测试证明晚绑定后同一 fake handle 被 guard 使用，acquire/release 计数为 2/1；panic 场景 acquire/release 为 1/1，说明 lease 在 handler 返回或 panic 后释放（`:236-273`）。
- 未发现第二 runtime/router/pool/authority/fallback，也没有第二个 public surface。
- 证据限制：新测试通过注入 fake provider 证明 handle 身份，没有从 `main` startup 直接断言 `recoveryAssembly.runtime` 被交给 guard/probes；该身份由 `main.go:610-618` 静态路径与既有 `TestRecoveryRootAssemblyWiresRuntime`（`recovery_runtime_test.go:719-769`）补强。见 N3。

### 3.6 NoRecoveryConfig（不通过）

- 测试直接构造 `func() recovery.Runtime { return nil }` 的 dispatcher（`:275-297`），随后断言 `/livez` 200、`/readyz` 503、业务路由未被 503 阻断、未知路径 404、错 method 405 + `Allow: GET`。这些 dispatcher 行为本身正确。
- 但它没有调用 `planRecoveryStartup`，没有断言 `present:false`，也没有走 `main.go:606-618` 的“无配置不创建 assembly、不写 handle”分支。测试环境 `validCompositionEnvironment()` 确实不含任何 recovery data key（`main_test.go:8134-8181`；`recovery_config.go:58-98`），因此具备构造 `present:false` 的条件，但测试未使用该条件。
- 作者 RED 证据显示：实现前 `NoRecoveryConfig` 已经 PASS，只有 `LiveReadyProbes` 因 `/livez` 404 失败（`.spec/wip/recovery-runtime/evidence-b13h-r01/s419-author-r01/red-test-output.txt`）。这说明该用例当时并未检测到本次实现的核心缺失，进一步支持“该 leaf 证据不足”的判断。
- 缓解事实：既有 `TestRun_RecoveryMaintenanceIsolation/normal_pipeline_control`（`main_test.go:10843-11001`）在无 recovery data key 的环境下执行真实 `run()`，期望 `compose=1`、`listen=2`、`new_server=2` 且无 recovery 装配副作用；本次独立复跑通过（`10-run-isolation.log`）。这降低了生产 startup 分支的剩余风险，但不满足本卡“同文件、真实 dispatcher、`present:false`”的验收措辞。

## 4. Findings

### [MEDIUM] OpenAfterCompleteSoftwareRecoverySameSession 未执行完整软件恢复

- **Finding ID**：`S419-CR-F001`
- **位置**：`cmd/han-server/recovery_main_dispatcher_integration_test.go:150-234`（关键 `:157`）；fake 状态写入 `:437-490`。
- **现状**：测试在 closed 得到 503 后直接调用 `runtime.setState(public_ready)`，随即以同一 Session 请求 200。没有执行 Begin、identity/article 消费、succeeded、撤维护/真 join、最终 proof、public open 中任何一步。
- **触发场景**：完整恢复链在 join、最终对账、open 许可或状态持久化上回归，但 runtime 的 `Verify` 仍返回 `public_ready`（或测试直接伪造该 observation）时，本测试仍会 PASS。
- **影响**：任务卡 `tasks.md:2834` 的 `R/P` leaf 未被证明；本卡关闭条件中的“完整软件恢复后同一 Session 200”不成立。失败负控（session/origin/csrf）虽已覆盖，但不能替代正向完整链。
- **最小修复**：在同一测试中通过 runtime 公共 API 驱动完整恢复序列（可使用 bounded in-memory ports，但不得直接 `setState`），记录 Begin→…→public open 的状态转换与 join/final proof 调用；开放后再用同一 production dispatcher、同一 Session 请求 200，并保留现有 session/origin/csrf 负控。若设计 owner 认为组件链已由 S4.16 覆盖、本卡只需状态翻转到 public_ready，则应由 Lead 明确收窄 leaf 措辞并记录风险；不能以当前测试名直接主张已覆盖。
- **验证**：`go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` 与 `-race` 均通过；新增日志需显示完整状态序列与 join/final proof 实际调用。
- **复审条件**：修复提交产生新 SHA 后，复核原 finding、完整状态序列断言与负控是否仍成立。

### [MEDIUM] NoRecoveryConfig 未覆盖 planRecoveryStartup 的 present:false

- **Finding ID**：`S419-CR-F002`
- **位置**：`cmd/han-server/recovery_main_dispatcher_integration_test.go:275-297`，helper `:299-363`；对照 `cmd/han-server/main.go:606-618`、`cmd/han-server/recovery_config.go:88-123`。
- **现状**：测试直接注入 `func() recovery.Runtime { return nil }`，只验证 dispatcher 在 nil handle 下的 probe/路由行为；没有调用 `planRecoveryStartup(environment, nil)`，没有断言 `present=false`，也没有经过 `main` 的无配置 startup 分支。
- **触发场景**：`recoveryRootConfigurationPresent` 或 `planRecoveryStartup` 回归（例如空环境下错误返回 `present:true`、错误创建 assembly），或在无配置时仍写入 recovery handle；当前测试不会失败。
- **影响**：任务卡 `tasks.md:2833` 的无配置子案例与关闭条件“含无 recovery 配置用例”未被真实覆盖。该缺口是验收证据缺口，不是已证实的生产运行缺陷（既有 `TestRun_RecoveryMaintenanceIsolation/normal_pipeline_control` 提供了部分缓解）。
- **最小修复**：在同一测试文件内使用 `validCompositionEnvironment()`（已确认不含 recovery data key）调用 `planRecoveryStartup(env, nil)` 并断言 `present=false`；由该计划驱动的生产 wiring seam 构造 dispatcher（handle 必须保持 nil），再断言 `/livez` 200、`/readyz` 503、业务路由跳过 guard、404/405/Allow 不变。若可安全复用 `run()` 测试 seam，另断言未调用 recovery assembly/compose。
- **验证**：普通与 `-race` 定向 integration 均通过；日志需明确打印 `present=false` 与无 assembly 创建证据。
- **复审条件**：修复提交产生新 SHA 后，复核原 finding、`present:false` 断言与无配置 dispatcher 行为。

## 5. 非阻断观察与建议

- **N1（LiveReadyProbes 断言因果偏弱）**：`:139-147` 在 public_ready 检查后把状态设回 closed，再请求业务路由得到 503。该断言证明 closed 仍阻断，但不能证明“此前 `/livez` 200 没有留下业务授权”。建议在 closed 状态下先 `/livez` 200，紧接着业务 GET 503，再进入 public_ready 分支。
- **N2（unknown 保真度）**：`s419RecoveryRuntime` 用 `ErrAdmissionClosed` 表示所有非 public_ready 状态（`:457-466`），包括 unknown。guard 对错误类型不敏感，行为等价；若真实 runtime 的 unknown 错误与 closed 在 Trace/错误码上有差异，本测试无法区分。建议至少用真实 runtime 的错误构造器或为该 fake 增加未知错误分支。
- **N3（SingleRuntimeHandle 测试间接）**：新测试注入 fake provider，证明的是 dispatcher 对同一 handle 的使用；`assembly.runtime` 与 `composition.runtime` 的同一性由 `recovery_runtime_test.go:719-769` 和生产静态路径证明。未发现缺陷，但若要单卡自证，可在测试中记录 provider 被调用时返回的 runtime 指针身份并与装配值比较（不要求跨包暴露新字段）。
- **N4（旧 helper 的 probe 行为变化）**：旧 `newLoginBusinessRouterWithTrace` 委托后，非 probe 路由的 404/405/Allow/Trace 行为不变；`/livez` 与 `/readyz` 从“未知路径 404”变为 200/503。这是设计 `design.md:3904` 要求的新公共 surface，且旧 helper 目前只被 `main_test.go` 使用，生产路径只在 `main.go:2419` 使用新函数，因此不构成生产兼容缺陷。若未来有外部消费者复用旧 helper，需要知会该行为差异。
- **N5（Lead Gate 证据目录不完整）**：`s419-lead-gate-r01/` 实际只有 `00-exitcodes-part1/2.txt`、`05-list.txt`、`06-integration-json.txt`、`07-integration-race-json.txt`、`08-unit.txt`，缺少 01–04 的 build/vet/gofmt 原始输出。本次已独立复跑并留存 `02-go-build.log`、`03-go-vet.log`、`04-go-vet-integration.log`、`05-gofmt.log`，验证结论不受影响，但 Lead 证据包应补齐或注明。
- **N6（probe 忽略 bool）**：`main.go:3633` 忽略 `recoveryGuardFromRuntimeHandle` 的第二返回值是安全的：`NewRecoveryGuard` 失败时零值 guard 的 `/livez` 仍 200、`/readyz` 仍 503，符合“liveness 不授许可、readiness 只认 public_ready”的合同；业务分支 `:3649` 正确使用 bool 实现 fail-closed。
- **N7（写集可归属性）**：两个对象未纳入 Git，无法用 diff 证明“只改两个文件”。mtime 显示 `cmd/han-server` 下只有两对象在 10:27–10:33 实施窗口内更新，`internal/platform/recovery_guard.go` 停留在 10-29，`internal/recovery` 停留在 10-30；这与任务卡写集一致，但仍是弱证据。

## 6. 额外重点逐项核对

| 重点 | 结论 | 证据 |
|---|---|---|
| 旧 `newLoginBusinessRouterWithTrace` 委托后 404/405/Allow/Trace 是否不变 | 非 probe 路由不变；probe 路径按设计新增 200/503 | `main.go:3489-3495`、`:3515-3662`；旧 helper 仅测试调用 |
| `recoveryGuardFromRuntimeHandle` 第二返回值使用 | 业务分支正确使用；probe 忽略 bool 安全 | `main.go:3633`、`:3649`；`recovery_guard.go:29-31/51-65` |
| 写集是否只改两个授权文件 | 未发现越界；受 Git 未跟踪限制 | §8；mtime；`git status` 282 行（281 基线 + 本评审目录） |
| 重排/格式化/死代码/敏感信息/硬编码/第二 authority | 未发现 | `gofmt -l` 无输出；`go build`/`go vet` 通过；无新增 secret；路由字面量为合同固定值；单一 runtime |
| 测试是否行为断言、真实 production 组合、无 skip/mock 替代 | 是；fake 仅作为 runtime 边界 | 测试 `:299-363`；无 `t.Skip`/`ServeMux`/拒启代证 |
| F008 是否按建议实现 | 是 | guard 在 exact router 内晚绑定（`main.go:3649-3652`），Trace 最外（`:3662`） |

## 7. 命令与退出码

所有命令在 `D:\AID\VSCode\Codex\Han` 根目录执行，日志位于本报告同目录。

| 序号 | 命令 | 退出码 | 结果摘要 | 日志 |
|---|---|---|---|---|
| 01 | `Get-FileHash` 两对象 + 字节数 | 0 | 两个 SHA256 与输入一致 | `01-filehash.log` |
| 02 | `go build ./...` | 0 | 无输出 | `02-go-build.log` |
| 03 | `go vet ./cmd/han-server` | 0 | 无输出 | `03-go-vet.log` |
| 04 | `go vet -tags=integration ./cmd/han-server` | 0 | 无输出 | `04-go-vet-integration.log` |
| 05 | `gofmt -l cmd/han-server/main.go cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | 无输出 | `05-gofmt.log` |
| 06 | `go test -tags=integration ./cmd/han-server -list '^TestRecoveryMainBusinessDispatcher$'` | 0 | 只列出目标测试 | `06-list.log` |
| 07 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | 6 个顶层子测试全部 PASS | `07-integration.log` |
| 08 | `go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | 全部 PASS，无 DATA RACE/WARNING | `08-integration-race.log` |
| 09 | `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 37.908s` | `09-unit.log` |
| 10 | `go test ./cmd/han-server -run '^TestRun_RecoveryMaintenanceIsolation$' -count=1 -v` | 0 | 全部子测试 PASS；`normal_pipeline_control` 无 recovery 配置正常装配 | `10-run-isolation.log` |

补充核对：`s419-author-r01/red-test-output.txt` 中 `NoRecoveryConfig` 在实现前已 PASS；`red-extended-before-implementation.txt` 显示实施前因 `businessComposition.recoveryHandle` 字段不存在而编译失败，说明 RED 证据分阶段产生，不能单独作为本卡验收通过依据。

## 8. 写集与安全审计

- **SHA 冻结**：评审开始与报告写入前两次复算一致；未修改被审对象。
- **写集**：本评审只向唯一授权目录写入日志与本报告；未写 `cmd/`、`internal/`、`.add/`、`.codex/`；未执行 Git add/commit/push/checkout/clean；未删除/移动文件。
- **他人改动**：未回滚、覆盖或格式化任何他人内容。
- **敏感信息**：测试使用 `admin-session-token` 等合成值；响应泄漏断言覆盖 `run/secret/backup/account/ref`；未发现真实凭据或 API Key。
- **硬编码**：`/livez`、`/readyz` 为设计固定的 exact route 字面量，属于合同常量；未发现可配置值被硬编码。
- **第二 authority/runtime**：未发现第二 runtime、router、pool、authority 或 production fallback；`recovery_runtime.go:528-597` 只构造一个 runtime 值。
- **Git 状态**：`git status --porcelain` 当前 282 行，已知基线 281 行，新增 1 行为本评审证据目录；`cmd/`、`internal/` 整体未跟踪，不能据此清理或提交。

## 9. 未验证边界与剩余风险

- 未运行 PostgreSQL、真实 HTTP 服务、Restore、SMTP、任何部署 E2E；未验证真实 POSIX/CAS/ACL/备份/provider。
- 未证明 `canonical_recovery_replay_same_result`。
- 新测试的 `newS419ProductionDispatcher` 直接组合 production handler，未执行 `main`/`planRecoveryStartup` 的完整 startup wiring；这是 `S419-CR-F002` 的核心缺口。
- 该测试使用 fake runtime 作为边界，真实恢复状态机的 join、最终 proof、open 许可未在本卡测试中执行；这是 `S419-CR-F001` 的核心缺口。
- 两对象未纳入 Git，Round 1 无法使用 commit diff，变更归因依赖任务卡 R 对账与 mtime。
- 本次 PASS 的 build/vet/gofmt/定向测试只证明编译、静态检查与定向测试通过，不等于部署 E2E、人工签收或 PlanningReview 通过。

## 10. 待 Lead 决定

1. **F001/F002 是否按阻断处理**：建议退回作者补齐两项验收证据后再复审；若 Lead 决定收窄 leaf 解释（以 S4.16 组件链 + 状态翻转替代完整链、以 nil provider 替代 `present:false`），需设计 owner/Lead 明确记录该决定与剩余风险，不能由评审默认接受。
2. **N1/N2 是否要求同批修正**：两项不阻断，但同属测试保真度，修复成本低，建议随 F001/F002 一并处理。
3. **Lead Gate 证据补齐**：`s419-lead-gate-r01/` 缺少 01–04 原始输出，建议补齐或注明由独立复跑替代。

## 11. 最终结论

**不通过（FAIL）。**

生产 dispatcher 的接线方向正确：exact route 先行、Trace 最外、guard 晚绑定且在 Session/下游之前、错误 fail-closed、旧调用方非 probe 行为不变、单一 runtime 与 lease 释放均有实现与测试证据。当前阻断来自验收证据缺口而非已证实的运行缺陷：`S419-CR-F001`（完整软件恢复后同 Session 200 未真实执行）与 `S419-CR-F002`（无配置 `present:false` 未真实覆盖）。在两项 finding 关闭或 Lead 明确收窄验收并记录风险之前，本卡不满足关闭条件，不能声称 S4.19 已通过。
