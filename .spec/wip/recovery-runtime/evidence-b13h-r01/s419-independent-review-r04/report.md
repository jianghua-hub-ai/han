# S4.19 层 (b) 返工后独立复审报告（s419-independent-review-r04）

- 角色：非作者独立 Reviewer（add-reviewer / add-code-review，objectType=code，两轮独立 + 有界消费者检查）
- 对象：identity-auth-S4.19 层 (b) 返工修订 `cmd/han-server/recovery_main_dispatcher_integration_test.go`
- 工作区 / 分支 / HEAD：D:\AID\VSCode\Codex\Han / main / 6da8abbceb0fa197364810a92a8fa2d97b38ddca
- 评审时间：2026-10-31 21:45–22:00（Asia/Shanghai）
- 结论：**PASS WITH MINOR**（无 HIGH / MEDIUM；F001、N01 CLOSED；新增 2 条 LOW 非阻断）
- 边界：本结论为 AI 代理独立核证，非真人签字；不代替 Lead 现场决定、批次 Gate 最终判定、人的业务接收或 Git 写授权。本轮只读源码/测试/任务/证据，唯一写入为本目录 report.md / verdict.md / commands.log 与 16 个原始日志/JSON。

## 1. 判定对象与停止条件核对

| 项 | 期望 | 实测 | 结论 |
|---|---|---|---|
| 被复审文件 SHA256 | BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A | 一致 | 通过 |
| 被复审文件字节 / 行 | 23693 B / 623 行 | 23693 / 623 | 通过 |
| 编码 | UTF-8 无 BOM、纯 LF | BOM=False、CR=0、LF=623 | 通过 |
| 上一版（Gate FAIL 版） | B89DA3791616353ACBAC602B1F22AC98298CFE888FBD55E1D3A4686F7BA67A97 / 16500 B | 文件已被覆盖，无法重算；该值在独立记录 `tasks.md:2869`（Gate FAIL 记录）与 `tasks.md:2951`（返工基线）及 `b13h-batch-gate-r01/verdict.md` 三处一致登记 | 作为历史值接受，不作“当前文件”判定 |
| main.go（零 production 改动主张） | 6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727 / 173317 B | 一致；且与 `tasks.md:2951` 记录的 Gate 后基线逐字节相同 | 零 production 改动成立 |

停止条件（哈希不符 / 测试失败 / 需改生产代码）均未触发；未修改任何源码或任务卡。

## 2. 两轮实际检查范围

- **第一轮（对象全文 + 上游依据）**：`add-code-review/SKILL.md`、`code-review-checklist.md`、`abstraction-principles.md`、`coding-standards.md` 全文；`tasks.md:2820-2869`（S4.19 卡：`:2859` 层 (b)、`:2862` 关闭条件、`:2868` 两层缺一不通过、`:2869` 返工合同）；`b13h-batch-gate-r01/report.md` 与 `verdict.md`（F001/F002/N01）；`s419-author-r04/report.md`；被复审文件 623 行逐行；`recovery_workers_test.go:70-180`（fixture/authority）；`internal/recovery/runtime.go:861-891/948-1068`（RunNormal/Verify/lease）；`internal/platform/recovery_guard.go` 全文；`main.go:2424`、`:3534`（production handler/router 真实入口）；`main_test.go:1369-1386`（fixture serve/request）；`test/integration/recovery/runtime_admission_test.go:1-27`（层 a）。
- **第二轮（返工点定向 + 复跑）**：`:30` 注册、`:169-281` 子测试全文、`:445-453` 泄漏断言、`:498-623` fake/薄包装；`recovery_isolation_matrix_integration_test.go:30/99/136-146`（符号复用与掩码先例）；`internal/platform/logging.go:12-27`（Trace 字符集）；`cmd/han-server/recovery_exclusive.go:20-71`（生产 exclusive lease 语义）；工具链与四组测试两波复跑（见 §7）；200000 次随机 Trace 子串模拟。
- **有界消费者检查**：本包内 `s419AssertNoRecoveryLeak`、`s419RecoveryRuntime`、`setState` 的其他引用文件；`TestRecoveryRuntimeProductionChain`（`recovery_runtime_integration_test.go:165`）保持性（静态，本轮 NOT RUN）。

## 3. 指纹复核（复跑前后一致，未观测外部改动）

| 文件 | SHA256 | 字节 | 行/换行/编码 | mtime |
|---|---|---:|---|---|
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A | 23693 | 623 / LF=623 CR=0 / 无 BOM | 2026-10-31 21:45:39 |
| `cmd/han-server/main.go` | 6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727 | 173317 | 5044 / LF=5044 CR=0 / 无 BOM | 2026-10-31 19:26:09 |
| `.add/project/knowledge/tasks/tasks.md` | F662C05814344289410CA7192E7B54F3D30F2207C4BA2AC4DED92E94E0808939 | 756211 | 2951 / CRLF=2951 / 无 BOM | 2026-10-31 21:38:09 |
| `test/integration/recovery/runtime_admission_test.go` | AF87672131C50D6552C696F445C7C976B30173ABBE1DEDEC2F4151BC539BE158 | 1021 | 27 | — |
| `cmd/han-server/recovery_runtime.go` | 900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7 | 35832 | 与 `tasks.md:2949` 记录一致 | — |
| `cmd/han-server/recovery_runtime_integration_test.go` | B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A | 31638 | 与 S4.16 基线一致 | — |
| `cmd/han-server/recovery_production_chain_integration_test.go` | 65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75 | 167339 | 与 S4.20 记录一致 | — |
| `cmd/han-server/recovery_workers_test.go` | B5CFC39DA1304026449549AAAC58C8231840A7F2A4166241AC8CD2B55D7B9532 | 28928 | 与 `tasks.md:2951` 基线一致 | — |

并发写者：`xDesk.exe` 6 个进程存活（PID 22928/25764/29960/35384/40736/49420），复跑期间静默；上表 8 个文件哈希与 mtime 前后未变，未观测到外部改动。`git status --short` 仅作参考（`cmd/` 未纳入版本控制，不出现于 status，本报告不据此判定）。

## 4. F001 逐条判定 — **CLOSED**

Gate F001 的实质指控：层 (b) 以 stub `setState("public_ready")` 假进入，未经真实 runtime，断言不足。逐条对应返工点：

1. **第 6 个子测试存在且已注册**：`:30` `t.Run("PublicReadyAdmitsSameSession", testRecoveryMainPublicReadyAdmitsSameSession)`；实现 `:179-281`。子测试列表 6 项，与作者报告一致。
2. **真实 recovery.NewRuntime**：`newOrdinaryFixture(t)`（`recovery_workers_test.go:148-172`）内 `:156` 调用 `recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{...})`，返回关闭态真实 runtime；`:166-170` 注册 `Stop` 清理。生产 dispatcher 由 `newProductionBusinessHandler`（`main.go:2424`，非测试替身）构建（`:418`），provider 于 `:417` 注入同一包装 runtime。
3. **真实 RunNormal 进入**：`:220` `ordinary.runtime.RunNormal(ctx)`（`internal/recovery/runtime.go:861-891`：Exclusive→readProof→startupPhase→ReconcileAndStart→activate）；`:223` `Verify`，`:227` 断言 `State=="public_ready"`。这不是 `setState`，`setState` 只存在于测试 fake 类型上（真实 runtime 无此方法）。
4. **薄包装仅计数**：`:583-588` 嵌入 `recovery.Runtime`；`:590-599` 仅 `Acquire` 计数后全量委托；`:601-605` 释放计数；`:613-623` lease 的 `Context` 委托、`Release` 先真实释放后计数。无状态改写、无错误吞并、无第二 runtime。
5. **closed 断言**：`:194` 503（`serve` 同时断言 Trace 最外，`main_test.go:1382-1384`）、`:199` code=`platform.recovery_in_progress`、`:202` envelope Trace 与 header 一致且非空、`:205` 泄漏断言、`:206-214` repository/reader/resolver 零调用、`:215` 事件仅 `trace`、`:216` acquire/release=`1/0`。
6. **public_ready 后同请求 200**：`:230` `/readyz` 200 `ready`（同一 handle 经 `Readiness`→`Verify`）、`:234` 同一 `*http.Request`（同一 `admin-session-token` cookie，`:235` 复核）返回 200、`:238/241` Article repository 与 author reader 均被调用、`:244` Session 恰 1 次、`:247` 事件精确 `trace,session,principal,repository,identity`、`:248` acquire/release=`2/1`。
7. **失败负控**：`:252-280` 注入 repository 错误 → `:263` code=`platform.database_unavailable`（非 recovery code）、`:266` 双重否定断言（code 与 body 均不含 `platform.recovery_in_progress`）、`:269` Trace 保留、`:272/275` repository/Session 各 +1、`:278` acquire/release 各 +1。
8. **层 (a) 分工披露**：`:169-178` 注释明示层 (a) `TestRecoveryRuntimePostgres` 承担完整恢复链（Begin→Consume→succeeded→Exit/join→activate），本层 (b) 停于 production dispatcher 并使用 `RunNormal`。该路径为 `tasks.md:2869` 返工合同明确允许的分支，且实测层 (a) 日志含 `RunNormal`/`Enter` 与 `observation=public_ready`（见 §7）。
9. **假进入残留清查**：`publishPublicReady` 全包 0 命中；`setState` 仅用于测试 fake（`:131/148/158/301/512/520`）与探针/矩阵测试，且 `:144-147` 已补“probe-only boundary”注释，与 `tasks.md:2869` 的“不得以 setState 冒充真实状态转换”不冲突（真实转换已由第 6 子测试承担）。

结论：F001 的四项指控（假进入、未走真实 runtime、断言不足、以 setState 代替）均不成立，判 **CLOSED**。

## 5. N01 判定 — **CLOSED**

- `rg -n 'publishPublicReady' cmd/han-server --glob '*.go'` → 0 命中：未调用的 helper 已删除，非死代码。
- `s419RecoveryRuntime` / `newS419RecoveryRuntime` / `setState` 仍被 `recovery_isolation_matrix_integration_test.go:30,99` 与 `LiveReadyProbes` 使用，不是残留死代码。
- `progress.md:44` 的旧披露未回改（属历史证据，不改写），已由 `tasks.md:2869` 与 `:2950` 记录更正披露；本轮不再重复计为缺陷。

结论：**CLOSED**。

## 6. 其他判定点

### 6.1 注释与行为一致性（非 finding）
`:189-192` 注释称“closed 下先探 `/readyz` 会因 `Verify` 关闭临时 exclusive lease 而永久 retire 本 fixture 的 in-memory authority”。核对属实：`Verify` 在无 live lease 时走 `beginObservation`+`deps.Exclusive.Acquire` 临时租约（`internal/recovery/runtime.go:986-1000`），结束经 `closeObservationLease`（`:1022-1032`）调用 `ordinaryExclusive.Close`（`recovery_workers_test.go:134`）→ `closed` 永久置位（`:128-132`），随后 `RunNormal` 的 `Exclusive.Acquire`/`Check` 失败。生产实现 `restoreDeploymentExclusive.Acquire`（`recovery_exclusive.go:20-41`）每次新建 `restoreExclusiveLease`，不会因一次 Verify 永久关闭，故注释“this fixture's in-memory authority”限定准确，不是生产缺陷。作者首次失败尝试日志 `s419-author-r04/dispatcher-attempt01-fail.json` 与根因一致。

### 6.2 与任务卡契约一致性
- `:2859` 层 (b) 的 closed 503（acquire=1/release=0）、public_ready 同请求 200、Trace 保留、Article 正常调用、acquire/release=2/1 全部有对应断言。
- `:2859` 层 (a) 同修订必跑：本轮由我独立复跑 `TestRecoveryRuntimePostgres` 普通与 `-race`，真实 PG，pass=7/fail=0/skip=0（§7）。
- `:2862` 关闭条件中的“无 recovery 配置用例”保持（`NoRecoveryConfig`，`:322-358`）。
- `:2868` “两层缺一不通过”：本次两层在同一修订、同一会话窗口内均实际跑通。
- `:2935`（Gate 报告引作 `:2934`，因 tasks.md 后续追加记录导致行号 +1）“不得用直接 setState 或 build 成功代替 …”：真实转换已由 RunNormal 承担，未违反。

### 6.3 测试有效性（负控 / 变异推理，未改生产代码）
若下列缺陷回归，现有断言会失败：
- 删除 `:220` `RunNormal`（保留真实 runtime 但不启动）：`:223 Verify` 对关闭态返回非 `public_ready`（`closedObservation`/`internal_ready`）→ `:227` 失败；即便绕过，`:234` 会得到 503 而非 200。
- 用 fake 状态翻转替换真实进入（把 provider 换成 `s419RecoveryRuntime` 并 `setState`）：真实 `Verify`/`Acquire` 不被驱动，`:223`/`:248` 计数与状态断言失败；且真实 runtime 无 `setState`，无法编译通过。
- guard 移到 exact router 之前：`:41/50/59` 的 404/405/Allow 断言被 503 破坏。
- guard 移到 Session/AccessGeneral 之后：`:212` `resolver.inputs==0` 与 `:215` 事件仅 `trace` 失败。
- 薄包装吞错或改写状态：`:199/:216/:248/:263/:278` 失败。
- 失败映射被 recovery 掩盖：`:263/:266` 失败。
- lease 未释放或重复释放：`:216`（1/0）、`:248`（2/1）、`:278`（+1/+1）失败。
本项为静态变异推理（用户允许不实际改生产代码），已逐条对照被测断言。

## 7. 复跑结果（第一手，两波一致）

| # | 命令 | exit | pass | fail | skip | DATA RACE | 证据 |
|---|---|---:|---:|---:|---:|---:|---|
| 1 | `gofmt -l cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | — | — | — | — | r4-01 / r4-09（均 0 B） |
| 2 | `go build ./...` | 0 | — | — | — | — | r4-02 / r4-10（0 B） |
| 3 | `go vet ./cmd/han-server` | 0 | — | — | — | — | r4-03 / r4-11（0 B） |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | — | — | — | — | r4-04 / r4-12（0 B） |
| 5 | dispatcher 普通（`-tags=integration -count=1 -json`） | 0 | 12 | 0 | 0 | 0 | r4-05 / r4-13 |
| 6 | dispatcher `-race` | 0 | 12 | 0 | 0 | 0 | r4-06 / r4-14 |
| 7 | `TestRecoveryRuntimePostgres` 普通（真实 PG） | 0 | 7 | 0 | 0 | 0 | r4-07 / r4-15 |
| 8 | `TestRecoveryRuntimePostgres` `-race`（真实 PG） | 0 | 7 | 0 | 0 | 0 | r4-08 / r4-16 |

- 环境：`postgresql-x64-18` Running；`$env:HAN_TEST_POSTGRES_DSN=$env:HAN_DATABASE_DSN`（`dsn_set=True`，未记录值）；所有原始文件 DSN/口令模式复扫命中 0。
- dispatcher 明细（r4-14）：ExactRouteBeforeGuard、ClosedUnknownTwoAdminGets503/{closed,unknown}、LiveReadyProbes、**PublicReadyAdmitsSameSession（0.06s PASS）**、SingleRuntimeHandle/{LateBinding,ReleaseAfterPanic}、NoRecoveryConfig、父测试、package。
- PG 日志关键行（r4-15）：`observation=public_ready`、`workers=3 holders=0 lifecycle_stop=1 acquire_after=ErrAdmissionClosed`、`real runtime.Enter consumed window … observation=public_ready`。
- 无 fail / skip / DATA RACE；未观察到 flaky。

## 8. 新 findings（无 HIGH / MEDIUM）

| ID | 严重级 | 位置 | 影响 | 建议 |
|---|---|---|---|---|
| S419-R04-F001 | LOW（测试卫生，非阻断） | `cmd/han-server/recovery_main_dispatcher_integration_test.go:445-453` 的 `s419AssertNoRecoveryLeak` 与调用点 `:92`（4 个 envelope）、`:205`（1 个 envelope） | 该 helper 将整个 body 小写后禁 `"ref"`（同时禁 `"run"` 等）。envelope body 含随机 Trace：26 位 Crockford base32（`internal/platform/logging.go:15-27`，字母表 `0123456789ABCDEFGHJKMNPQRSTVWXYZ`）。实测 200000 个同分布 Trace 中 145 个含 `REF`（0.0725%），`RUN` 为 0（`U` 不在字母表）。每次 dispatcher 运行约 5 个含 Trace 的 envelope → 约 0.36%（≈1/275）偶发假失败。同包已有先例：`recovery_isolation_matrix_integration_test.go:136-146` 的 `s15AssertNoRecoveryLeak` 先掩码 Trace，并注释“ULID Trace can randomly contain a forbidden substring such as \"ref\", which would otherwise make the check flaky”。 | 将 `:92`/`:205` 改用掩码后的 helper（或新增 `s419AssertNoRecoveryLeakMasked(body, traceID)`）；不阻断本次结论。 |
| S419-R04-F002 | LOW（覆盖缺口，非缺陷） | `:189-192`（显式跳过 closed 态 `/readyz` 探针）+ `:230`（仅断言 public_ready 200） | 真实 runtime 的 closed→`/readyz` 503 分支在本文件未断言（该分支仅由 `LiveReadyProbes` 的 fake runtime 与 `internal/platform` 单测覆盖）。production dispatcher 上“真实 runtime 未就绪时 readiness 拒绝”缺少同修订证据。 | 用独立 `newOrdinaryFixture`/dispatcher 实例补一次 closed `/readyz` 503，避免与 `RunNormal` 共用被 `Verify` 永久关闭的 authority；非阻断。 |

非 finding 的已核对项：`:189-192` 注释准确（§6.1）；`s419ObservedLease.Release` 非幂等计数在当前 guard 单次 `defer` 语义下无影响；薄包装不引入第二 runtime。

## 9. 未验证边界（未跑 ≠ 通过）

- 性能 / 压力 / 容量 / 延迟预算。
- 真实部署环境、真实 Host E2E、真实 HTTP 服务端到端（本轮仅 `httptest` 与测试装配）。
- POSIX journal / fsync / flock / CAS 语义。
- 真实 keyring / HSM / 密钥托管（S4.20 使用模拟边界并已披露）。
- approved backup inventory 与真实 restore adapter。
- production dispatcher 全路由 closed/unknown 矩阵（仅 `/api/v1/admin/articles` 与 `/{id}` 两条）。
- 进程级“全局唯一 runtime”的运行期隔离断言（仅指针/组装层 + 单进程测试证明）。
- 真实 provider / ACL / anchor / lease / restore 行为。
- `enter` / `install` 与 recovery secret candidate 的承接任务（未登记，unknown）。
- `TestRecoveryRuntimeProductionChain`（S4.16，`127.0.0.1:53519`）本轮 NOT RUN（保持静态保持性核对；符号仍在 `recovery_runtime_integration_test.go:165`，文件哈希与基线一致）。
- `TestRecoveryProductionChainComplete`（S4.20 链测试）本轮未跑；本次不据此扩展 S4.20 结论。
- 全包 `go test -tags=integration ./cmd/han-server`、全仓 `gofmt`、`goimports`、`golangci-lint` 未运行。
- B13H 批 Gate 的最终批次结论：本报告只覆盖 S4.19 层 (b) 返工与两层同修订复跑；批次级 17 文件 / 21 槽复核需 Lead 在本次基础上重判，单卡 PASS 不自动等于 G4-batch PASS。
- 人的业务接收、部署授权、Git 写授权；本结论为 AI 代理核证，非真人签字。

## 10. 结论

**PASS WITH MINOR**。F001 与 N01 均 CLOSED；返工以真实 `recovery.NewRuntime` + 真实 `RunNormal` 进入 `public_ready`，同 production dispatcher/同 Session 的 closed 503 与 public_ready 200、Trace、下游调用、acquire/release 计数、失败负控均有可失败断言；工具链与两层测试（普通与 `-race`，真实 PG）exit 0、fail=0、skip=0、DATA RACE=0；被复审文件与 `main.go` 指纹与声明一致，未观测外部改动。新增 2 条 LOW（`S419-R04-F001`/`S419-R04-F002`）为测试卫生与覆盖建议，不阻断。