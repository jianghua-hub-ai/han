# S4.19 层 (b) 返工后独立复审 — 结论（s419-independent-review-r04）

- 角色：非作者独立 Reviewer（add-reviewer / add-code-review，objectType=code）
- 对象：`identity-auth-S4.19` 层 (b) 返工修订 `cmd/han-server/recovery_main_dispatcher_integration_test.go`
- 工作区 / 分支 / HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 时间：2026-10-31（Asia/Shanghai）
- 结论：**PASS WITH MINOR**

## 1. 判定摘要

F001 与 N01 均 **CLOSED**。返工后的层 (b) 不再用 `setState` 或测试替身冒充进入：它使用真实 `recovery.NewRuntime`，经真实 `RunNormal` 到达 `public_ready`，并在同一 production dispatcher、同一 handle、同一 Session 上完成：

- closed 时同一业务请求返回 503 `platform.recovery_in_progress`，Trace 保留，Article repository/author-reader/Session 零调用，acquire/release = 1/0；
- `public_ready` 后 `/readyz` 返回 200 `ready`，同一请求（同一 Session cookie）返回 200，Article 正常调用，事件顺序为 `trace,session,principal,repository,identity`，acquire/release = 2/1；
- 失败负控返回 `platform.database_unavailable`，未被 `platform.recovery_in_progress` 掩盖，Trace 保留，repository/Session 各 +1，acquire/release 各 +1。

新增 2 条 LOW finding，均为测试卫生/覆盖建议，不阻断本次结论；无 HIGH 或 MEDIUM。

## 2. F001 — CLOSED

逐条对应 Gate 返工点（详见 `report.md` §4）：

- 新增第 6 个子测试 `PublicReadyAdmitsSameSession`，`:30` 注册、`:179-281` 实现；子测试总数 6，与作者说明一致。
- `:181` 调用 `newOrdinaryFixture(t)`；`recovery_workers_test.go:156` 实测调用 `recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{...})`，返回真实 runtime；`:166-170` 注册 `Stop` 清理。
- `:220` 调用真实 `ordinary.runtime.RunNormal(ctx)`（`internal/recovery/runtime.go:861-892`：`beginNormal` → `Exclusive.Acquire` → `readProof` → `startupPhase` → `ReconcileAndStart` → `activate`）；`:223` 调用 `Verify`；`:227` 断言 `State == "public_ready"`。不是 `setState`，真实 runtime 也没有 `setState` 方法。
- `:583-623` 的 `s419ObservedRuntime`/`s419ObservedLease` 是薄包装：嵌入 `recovery.Runtime`，只统计 `Acquire` 与 lease `Release`，其余方法全量委托；不改状态、不吞错误、不建第二 runtime。
- `publishPublicReady` 已删除（`rg` 在 `cmd/han-server --glob '*.go'` 命中 0）；`setState` 仅留在测试 fake/探针与矩阵测试，`:144-147` 已补 probe-only 边界注释。
- closed 断言：`:194` 503、`:199` code、`:202` Trace 与 header 一致、`:205` 泄漏断言、`:206-214` 下游零调用、`:216` 1/0。
- public_ready 断言：`:230` `/readyz` 200、`:234` 同一 `*http.Request` 200、`:238/241` Article repository 与 author reader 被调用、`:244` Session 1 次、`:247` 事件精确、`:248` 2/1。
- 失败负控：`:252-280`，code 为 `platform.database_unavailable`，双重否定确认不是 recovery code，Trace 保留，计数 +1/+1。

## 3. N01 — CLOSED

- `publishPublicReady` 全包 0 命中，未调用的 helper 已删除。
- `s419RecoveryRuntime` / `newS419RecoveryRuntime` / `setState` 仍被 `recovery_isolation_matrix_integration_test.go` 与 `LiveReadyProbes` 使用，不是残留死代码；`:144-147` 已注明 fake 边界，真实转换由第 6 子测试与层 (a) 承担。
- `progress.md:44` 为历史证据，未回改；更正披露已登记在 `tasks.md:2869` 与 `:2950`，本轮不重复计为缺陷。

## 4. 新 findings（无 HIGH / MEDIUM）

| ID | 严重级 | 位置 | 影响 | 建议 |
|---|---|---|---|---|
| `S419-R04-F001` | LOW（测试卫生，非阻断） | `cmd/han-server/recovery_main_dispatcher_integration_test.go:445-453` 的 `s419AssertNoRecoveryLeak`，调用点 `:92` 与 `:205` | 该 helper 将整个 body 小写后禁 `"ref"`；envelope body 含随机 Trace（26 位 Crockford base32，字母表 `0123456789ABCDEFGHJKMNPQRSTVWXYZ`）。实测 200000 个 Trace 中 145 个含 `REF`（0.0725%），`RUN` 为 0；每次 dispatcher 约 5 个含 Trace 的 envelope，估算约 0.36% 偶发假失败。同包 `recovery_isolation_matrix_integration_test.go:136-146` 的 `s15AssertNoRecoveryLeak` 已掩码 Trace 并注明同类风险。 | 在泄漏断言前掩码 Trace，或新增 masked helper；不阻断本次结论。 |
| `S419-R04-F002` | LOW（覆盖缺口，非缺陷） | `:189-192`（显式跳过 closed 态 `/readyz` 探针）、`:230`（仅断言 public_ready 200） | 真实 runtime 的 closed → `/readyz` 503 分支未在本文件断言；该分支仅由 `LiveReadyProbes` 的 fake runtime 与 `internal/platform` 单测覆盖。production dispatcher 上“真实 runtime 未就绪时 readiness 拒绝”缺少同修订证据。 | 用独立 `newOrdinaryFixture`/dispatcher 实例补一次 closed `/readyz` 503，避免与 `RunNormal` 共用会被 `Verify` 永久关闭的 authority；非阻断。 |

## 5. 独立复跑

| 命令 | exit | pass | fail | skip | DATA RACE |
|---|---:|---:|---:|---:|---:|
| `gofmt -l cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | — | — | — | — |
| `go build ./...` | 0 | — | — | — | — |
| `go vet ./cmd/han-server` | 0 | — | — | — | — |
| `go vet -tags=integration ./cmd/han-server` | 0 | — | — | — | — |
| dispatcher 普通（`-tags=integration -count=1 -json`） | 0 | 12 | 0 | 0 | 0 |
| dispatcher `-race` | 0 | 12 | 0 | 0 | 0 |
| `TestRecoveryRuntimePostgres` 普通（真实 PG） | 0 | 7 | 0 | 0 | 0 |
| `TestRecoveryRuntimePostgres` `-race`（真实 PG） | 0 | 7 | 0 | 0 | 0 |

- 两波复跑结果一致；证据见同目录 `r4-01`…`r4-16`，命令记录见 `commands.log`。
- `postgresql-x64-18` 为 Running；`$env:HAN_TEST_POSTGRES_DSN=$env:HAN_DATABASE_DSN` 已设置，仅记录 `dsn_set=True`，未记录值；16 个原始证据文件 DSN/口令模式复扫命中 0。
- dispatcher 明细含 `PublicReadyAdmitsSameSession` PASS；PG 日志含 `observation=public_ready`、`workers=3 holders=0`、`acquire_after=ErrAdmissionClosed`。

## 6. 指纹与写集

| 文件 | SHA256 | 字节 |
|---|---|---:|
| 被复审 `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A` | 23693（623 行、纯 LF、无 BOM） |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317（与 `tasks.md:2951` 基线一致，零 production 改动成立） |
| `.add/project/knowledge/tasks/tasks.md` | `F662C05814344289410CA7192E7B54F3D30F2207C4BA2AC4DED92E94E0808939` | 756211（2951 行、CRLF） |
| `test/integration/recovery/runtime_admission_test.go` | `AF87672131C50D6552C696F445C7C976B30173ABBE1DEDEC2F4151BC539BE158` | 1021 |
| `cmd/han-server/recovery_runtime.go` | `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` | 35832 |
| `cmd/han-server/recovery_runtime_integration_test.go` | `B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A` | 31638 |
| `cmd/han-server/recovery_production_chain_integration_test.go` | `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` | 167339 |
| `cmd/han-server/recovery_workers_test.go` | `B5CFC39DA1304026449549AAAC58C8231840A7F2A4166241AC8CD2B55D7B9532` | 28928 |

复跑前后上述文件哈希与 mtime 未变。`xDesk.exe` 有 6 个进程存活但复跑期间静默，未观测到外部改动；未做回滚。`git status --short` 仅作参考，`cmd/` 未纳入版本控制，不以 status 判定这些文件。

## 7. 未验证边界（未跑 ≠ 通过）

- 性能/压力/容量/延迟预算。
- 真实部署环境、真实 Host E2E、真实 HTTP 服务端到端（本轮仅 `httptest` 与测试装配）。
- POSIX journal/fsync/flock/CAS 语义。
- 真实 keyring/HSM/密钥托管（S4.20 使用模拟边界并已披露）。
- approved backup inventory 与真实 restore adapter。
- production dispatcher 全路由 closed/unknown 矩阵（仅验证 `/api/v1/admin/articles` 与 `/{id}` 等已测路径）。
- 进程级“全局唯一 runtime”的运行期隔离断言（仅指针/装配层与单进程测试证明）。
- 真实 provider/ACL/anchor/lease/restore 行为。
- `enter` / `install` 与 recovery secret candidate 的承接任务（未登记，unknown）。
- `TestRecoveryRuntimeProductionChain`（S4.16，`127.0.0.1:53519`）本轮 NOT RUN。
- `TestRecoveryProductionChainComplete`（S4.20 链测试）本轮 NOT RUN。
- 全包 `go test -tags=integration ./cmd/han-server`、全仓 `gofmt`、`goimports`、`golangci-lint` 未运行。
- B13H 批次 Gate 最终结论：本轮只覆盖 S4.19 层 (b) 返工与两层同修订复跑；17 文件/21 声明槽的批次级 Gate 需 Lead 重跑，单卡 PASS 不自动等于 G4-batch PASS。
- 人的业务接收、部署授权、Git 写授权未发生；本结论为 AI 代理核证，非真人签字。

## 8. 结论

**PASS WITH MINOR**。F001、N01 CLOSED；无 HIGH/MEDIUM；2 条 LOW 为测试卫生与覆盖建议，不阻断。返工已真实经过 `recovery.NewRuntime` 与 `RunNormal`，并以同 production dispatcher、同 Session 覆盖 closed 503 → public_ready 200、Trace、下游调用、acquire/release 计数与失败负控；工具链与两层测试普通/`-race`（真实 PG）均 exit 0，fail=0，skip=0，DATA RACE=0。未验证边界不因本结论扩张。