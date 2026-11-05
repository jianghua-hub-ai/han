# identity-auth-S4.19 返工实施报告（s419-author-r04）

## 结论

- 目标 finding `B13H-GATE-F001`（HIGH）与 `N01`（LOW）已在本层 (b) 处理。
- 唯一代码写文件：`cmd/han-server/recovery_main_dispatcher_integration_test.go`（零 production 改动，未改 `main.go`）。
- `TestRecoveryMainBusinessDispatcher` 现有 6 个顶层子测试；新增第 6 个 `PublicReadyAdmitsSameSession`。
- 最终 targeted 运行：`pass=12 / fail=0 / skip=0`（含 package、父测试与嵌套子测试事件）；6 个顶层子测试全部出现并通过。
- 未发生 STOP 条件；未写 Git，未运行 PG/HTTP/service/restore/部署/E2E，未写任何凭据。

工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

## 改动摘要

1. 注册第 6 个顶层子测试 `PublicReadyAdmitsSameSession`；既有 5 个子测试语义未改。
2. 新子测试使用真实 `recovery.NewRuntime`（经同包既有 `newOrdinaryFixture` 构造）并通过真实 `RunNormal` 进入 `public_ready`；未使用 `setState`、fake world、测试 transport 或单测适配器完成状态转换。
3. 新增薄观察包装 `s419ObservedRuntime` / `s419ObservedLease`：嵌入真实 `recovery.Runtime`，所有方法委托；仅覆写 `Acquire` 与 lease 的 `Release` 做计数。状态转换仍由真实 runtime 完成。
4. 删除未调用的 `publishPublicReady`（N01）。`s419RecoveryRuntime` / `newS419RecoveryRuntime` / `setState` 保留，仍被 `recovery_isolation_matrix_integration_test.go` 与探针测试使用。
5. `LiveReadyProbes` 中的 fake-runtime `setState("public_ready")` 保持原语义，仅补注释说明其探针边界。

## 真实 runtime 进入方式与层 (a)

- 构造：`newOrdinaryFixture(t)` 内部执行 `recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{...})`，返回关闭态真实 runtime。
- 进入：`ordinary.runtime.RunNormal(ctx)` 走真实 ordinary startup（Exclusive -> proof -> startup phase -> `ReconcileAndStart` -> `activate`），成功后 `Verify` 返回 `State="public_ready"`。
- 层 (a)：`TestRecoveryRuntimePostgres`（`test/integration/recovery`）承担完整恢复链（Begin -> Consume -> succeeded -> Exit/join -> activate）。本层 (b) 在 production dispatcher 处停止，不重建完整 restore world；因此选择官方允许的 `RunNormal` 真实进入，而非 `runtime.Enter`。
- 薄包装只统计 dispatcher guard 的 `Acquire`/`Release`；fixture 的 workers 绑定真实 runtime，不经过包装，不会污染计数。

## 同 Session / 同 dispatcher 断言

同一 `newS419ProductionDispatcher` handler、同一 provider 返回的同一包装 runtime、同一 `*http.Request`（cookie 始终为 `admin-session-token`）：

- closed：
  - `GET /api/v1/admin/articles` -> 503，envelope code `platform.recovery_in_progress`，Trace header 与 envelope Trace 一致且非空。
  - acquire/release = `1/0`。
  - Session 未到达（`resolver.inputs==0`）、Article repository/reader 均未到达（`lists+gets==0`、`reader.calls==0`）。
  - 事件仅 `trace`。
- 真实进入 `public_ready` 后，同一请求重放：
  - 200，Trace 保留。
  - Session 到达（`resolver.inputs==1`），repository 被调用（`lists>0`），author reader 被调用（`reader.calls>0`）。
  - 事件序列精确为 `trace, session, principal, repository, identity`。
  - acquire/release = `2/1`。
  - `/readyz` -> 200 `{"status":"ready"}`。
- 失败负控（同一 Session 的新请求，repository 返回合成错误）：
  - 下游自身映射为 503 `platform.database_unavailable`，不是 `platform.recovery_in_progress`；响应体不含 recovery code。
  - Trace 保留；Session 与 repository 均到达；acquire/release 相比失败前各 +1。
  - 说明：repository 传输错误在既有 `ArticleQueryService` 中映射为 `platform.database_unavailable`（同一 503 状态但不同错误码），因此断言以错误码、Trace 与下游到达证据区分，而非仅凭 HTTP 状态。

## 首次失败尝试（保留证据）

`dispatcher-attempt01-fail.json` 是首次实现版本的运行结果（exit 1）：`RunNormal() error = recovery admission closed`。

根因：closed 状态下先调用 `/readyz` 会执行 `Verify`，对关闭态 runtime 临时取 exclusive lease 并在结束时 `Close()`；`ordinaryAuthority` 的内存 exclusive 随即永久标记 closed，导致后续 `RunNormal` 失败。这是测试 fixture 的端口语义，不是 production runtime 缺陷。

修复：移除进入前的 `/readyz` 探针；closed 语义由业务请求的 503 本身证明。进入 `public_ready` 后的 `/readyz` 200 仍保留。首次失败输出未被删除，作为诊断证据保留。

## LiveReadyProbes 边界

`LiveReadyProbes` 继续用 `s419RecoveryRuntime.setState` 驱动探针状态机，这是探针测试的既有边界，不在 `B13H-GATE-F001` 点名范围。真实 admission 转换由 `PublicReadyAdmitsSameSession`（真实 runtime + `RunNormal`）与层 (a) `TestRecoveryRuntimePostgres` 承担；未为探针伪造真实恢复。

## 文件指纹（改动前后）

| 文件 | 前 SHA256 | 前 bytes/lines | 后 SHA256 | 后 bytes/lines | 编码 |
|---|---|---|---|---|---|
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `B89DA3791616353ACBAC602B1F22AC98298CFE888FBD55E1D3A4686F7BA67A97` | 16500 / 466 | `BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A` | 23693 / 623 | UTF-8 无 BOM，纯 LF |

只读文件复核（改动后与任务给定基线一致，未修改）：

| 文件 | SHA256 | bytes |
|---|---|---|
| `cmd/han-server/recovery_runtime.go` | `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` | 35832 |
| `cmd/han-server/recovery_production_chain_integration_test.go` | `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` | 167339 |
| `cmd/han-server/recovery_runtime_integration_test.go` | `B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A` | 31638 |
| `cmd/han-server/recovery_workers_test.go` | `B5CFC39DA1304026449549AAAC58C8231840A7F2A4166241AC8CD2B55D7B9532` | 28928 |
| `cmd/han-server/main_test.go` | `0E61F4A37D314D2D35B80EA8184BEF17EB6227FCAF1CA0F57C09736005185820` | 399522 |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 |
| `internal/recovery/runtime.go` | `4C0946AF7C29DE472CAE4848CE2F55A68288003071957FB62B66BFF3B59604C4` | 59138 |
| `internal/platform/recovery_guard.go` | `0187E9A0B40FFC465240BFA8FF40FFD9DA7DB8A69B378F1931D94B6974C1826D` | 3348 |

## 命令与 exit code

| 命令 | exit | 结果 |
|---|---|---|
| `gofmt -l cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | 无输出（已格式化） |
| `go build ./...` | 0 | `build.log` 为空 |
| `go test -tags=integration ./cmd/han-server -list '^TestRecoveryMainBusinessDispatcher$'` | 0 | 列出 `TestRecoveryMainBusinessDispatcher`（`list.log`） |
| `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | `dispatcher.json`，pass=12/fail=0/skip=0 |
| `go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | `dispatcher-race.json`，pass=12/fail=0/skip=0 |
| `go vet ./cmd/han-server` | 0 | `vet.log` 为空 |
| `go vet -tags=integration ./cmd/han-server` | 0 | `vet-integration.log` 为空 |

首次实现版本的 dispatcher 运行为 exit 1，输出保存为 `dispatcher-attempt01-fail.json`；修复后最终运行为 exit 0。

## 6 个顶层子测试

1. `ExactRouteBeforeGuard`
2. `ClosedUnknownTwoAdminGets503`
3. `LiveReadyProbes`
4. `PublicReadyAdmitsSameSession`（新增）
5. `SingleRuntimeHandle`
6. `NoRecoveryConfig`

普通与 race JSON 中均出现以上 6 项且全部 pass；`PublicReadyAdmitsSameSession` 内部不再嵌套 `t.Run`，失败负控为同子测试内顺序断言。

## 证据文件

- `report.md`（本文件）
- `dispatcher.json`（最终普通运行）
- `dispatcher-race.json`（最终 race 运行）
- `dispatcher-attempt01-fail.json`（首次失败诊断，保留）
- `build.log`、`list.log`、`vet.log`、`vet-integration.log`

未写入或修改 `s419-rework-r01/`、`b13h-batch-gate-r01/` 或其他历史证据目录。

## 未验证项 / 限制

- 层 (a) `TestRecoveryRuntimePostgres` 未在本任务复跑（禁止 PG/E2E；由 Lead 后续执行）。
- 未运行整个 `cmd/han-server` integration 包，仅运行指定 dispatcher 测试。
- 本层未覆盖 `runtime.Enter` 的完整恢复世界；按要求使用 `RunNormal`，完整恢复链由层 (a) 承担。
- 失败负控采用 repository 传输失败，映射为 503 `platform.database_unavailable`（非 500）；已验证错误码、Trace 与下游到达，未被 `platform.recovery_in_progress` 掩盖。
- `Go build ./...` 通过不等于行为验证；本次行为证据为上述 targeted 普通/race 测试。
- 工作树存在其他任务的既有修改/未跟踪文件（`cmd/` 在 HEAD 下整体未跟踪）；本任务只新增/修改上述唯一代码文件与本证据目录。