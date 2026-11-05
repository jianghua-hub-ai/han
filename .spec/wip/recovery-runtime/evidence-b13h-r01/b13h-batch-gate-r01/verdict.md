# B13H 批次 Gate（G4-batch）独立结论

- 角色：非作者独立 Reviewer（`add-reviewer` / `add-code-review`，objectType=code）
- 对象：B13H 批次 Gate，含已关闭卡 `identity-auth-S4.16` / `S4.19` / `S4.20`（17 唯一文件 / 21 声明槽）
- 工作区 / 分支 / HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审时间：2026-10-31（Asia/Shanghai）
- 结论：**FAIL**（HIGH 阻断 1 项）
- 边界：本结论为 AI 代理独立核证，非真人签字；不代替 Lead 现场决定、人的业务接收、部署授权或 Git 写授权。本轮只读，未修改源码/测试/设计/任务/Plan/既有证据。

## 批次级理由

批次 Gate 不能成立，原因不是单卡问题相加，而是批次合同一项必跑证据在 gate 修订上缺失：

1. S4.19 卡面 `tasks.md:2859` 要求 `PublicReadyAdmitsSameSession` 的“两层证据同修订必跑”：(a) `TestRecoveryRuntimePostgres` 真实 runtime + 真实 PostgreSQL 全链；(b) 同一 production handler、同一 Session 在 closed 下 503，共享 runtime 进入 `public_ready` 后同一请求 200，Trace 保留、Article 正常、acquire/release=2/1。`tasks.md:2868` 明确“B13H 批 Gate 仍须同修订同跑两层，缺一不通过”。
2. 在 gate 修订（HEAD）上：
   - 层 (a) `TestRecoveryRuntimePostgres`（位于 `test/integration/recovery`，非 `cmd/han-server` 包）真实 PG 全链存在并通过（本会话普通与 `-race` 均 fail=0/skip=0/DATA RACE=0）。
   - 层 (b) 在 production dispatcher 上已不存在：`cmd/han-server/recovery_main_dispatcher_integration_test.go` 现仅 5 个子测试（`ExactRouteBeforeGuard`、`ClosedUnknownTwoAdminGets503`、`LiveReadyProbes`、`SingleRuntimeHandle`、`NoRecoveryConfig`），全文无 `recovery.NewRuntime`、无 `runtime.Enter`、无 `PublicReadyAdmitsSameSession`；`LiveReadyProbes:143` 直接 `setState("public_ready")` 后断言业务 200。
   - R03 时期认证的层 (b)（真实 `recovery.NewRuntime` + `runtime.Enter` → 同会话 200）因 world-hold 移出（`.original` 留存，SHA `36DA30AB…`/22209 B，`:169` `recovery.NewRuntime`、`:201` `runtime.Enter`）；S4.20 链测试未在 production dispatcher 上重建，仅用测试 transport 适配器（`:3767-3773`），并在 `:3867` 把 `production_dispatcher_coverage` 显式交回 `TestRecoveryMainBusinessDispatcher`。
3. 违反 `tasks.md:2934`：“不得用直接 `setState` 或 build 成功代替 resume 恢复运行完整完成、测试、评审或 E2E”。本合同要求的层 (b) 恰被 stub 状态翻转替代。

因此本批**未达到可交回的批次级覆盖**；单卡 R03 或 S4.20 的 PASS 不能相加替代批次 Gate（Plan `:1598`）。

## Finding 清单

### B13H-GATE-F001 — HIGH（阻断）
- 位置：`cmd/han-server/recovery_main_dispatcher_integration_test.go`（SHA `B89DA3791616353ACBAC602B1F22AC98298CFE888FBD55E1D3A4686F7BA67A97` / 16500 B）`LiveReadyProbes:143`；对照 `s419-world-hold-r01/recovery_main_dispatcher_integration_test.go.original:169,201`；`cmd/han-server/recovery_production_chain_integration_test.go:3767-3773,3814,3867`；`s420-implementation-r01/d7-main-dispatcher-r02.json`。
- 违反依据：`tasks.md:2859`（层 b）、`:2862`、`:2868`（两层缺一不通过）、`:2934`（禁 setState 代替 resume）；`s419-lead-gate-r03/report.md` §5；Plan `:1597-1599`。
- 触发场景：在 gate 修订上执行 S4.19 两层同跑，并核对“真实恢复完成→`public_ready`→同会话 200 经 production dispatcher”。
- 影响：层 (b) 在 gate 修订缺失，合同“两层缺一不通过”被违反；join/final proof/open 许可回归缺少合同要求的同修订证据；批次 Gate 不能判 PASS。
- 修复建议（本轮不执行）：作者在 production dispatcher 测试中以真实 `recovery.NewRuntime` + 真实恢复完成重建同 Session 200（含 Trace、Article 下游、acquire/release=2/1、失败负控）；或由 Lead 正式收窄合同并书面同步卡面/Plan/Gate 口径。不得用 `setState` 或测试 transport 冒充。

### B13H-GATE-F002 — MEDIUM（交付完整性）
- 位置：`tasks.md:2932`、`:2948`；`s420-implementation-r01/d7-main-dispatcher-r02.json`（SHA `A673F00E92532CFCC1FDBEADA5FF17EC468F2DF0E0734DC139568514503A79F7` / 9596 B）、`d7-main-dispatcher-race-r02.json`（`08407B0FF13538A9B0ADE38C2C1CC39B0740FC2B9057C098A6C44BCB864EA106` / 9595 B）。
- 违反依据：`tasks.md:2859`/`:2868` 与 `s419-lead-gate-r03/report.md` §5 的“两层”要求；`task-execution-workflow` §10 的“成果/证据与验收一致”原则。
- 影响：S4.20 关闭记录把 dispatcher 复跑表述为满足“两层 / 按卡面 oracle 闭合”，但 `d7` 证据仅含 5 个子测试，不含真实 resume 的层 (b)；证据表述强于实际覆盖，可能使 Lead/后续消费者误判 S4.19 门禁已在本批复跑。
- 修复建议（本轮不执行）：在关闭记录中如实限定 dispatcher 复跑覆盖范围（层 a 由 PG 测试、层 b 待重建），或补齐层 (b) 后再声明“两层闭合”。

### N01 — LOW（测试卫生 / 披露，非阻断）
- 位置：`cmd/han-server/recovery_main_dispatcher_integration_test.go:421 publishPublicReady` 定义后全文件无调用；`s420-implementation-r01/progress.md:44` 自述 `s419RecoveryRuntime`“closed/unknown 用，禁止成功链”，但 `LiveReadyProbes:143` 用它走 `setState("public_ready")` 成功链。
- 影响：披露与实现不一致；与 F001 同源，不单独阻断。

## 四维判定摘要

- 维度 1（跨卡接口一致性）：PASS（静态/指针层）。唯一 `recoveryHandle`（`main.go:344`，晚绑定 `:546-551`，注入 `:601`，组装 `:634`，回填 `:655-657`）；guard 在 exact router（`:3560-3676`）之后、Session/AccessGeneral 之前（`:3694-3696`）；`/livez` `/readyz` 同 handle（`:3678-3684`）；未发现第二 runtime/router/pool/authority/fallback（`http.NewServeMux` 仅在 legacy 路径 `:4034`）。
- 维度 2（迁移与回归）：**FAIL**。S4.16 `TestRecoveryRuntimeProductionChain` 符号/断言/语义保持（`recovery_runtime_integration_test.go:165`，本轮 NOT RUN）；S4.19 层 (a) PASS、层 (b) FAIL（F001）。
- 维度 3（所有权与串行）：PASS。21 槽/17 唯一文件数学一致；17 文件均在登记清单内；mtime 呈串行交接；复核前后 mtime/SHA 未变；无清单外越界写入。
- 维度 4（批次级集成/覆盖）：不足，受 F001 阻断；单卡 PASS 不能相加。

## 未验证边界（未跑≠通过）

- 性能 / 压力 / 容量与延迟预算。
- 真实部署环境、真实 Host E2E、真实 HTTP 服务端到端（本轮仅 httptest / 测试装配）。
- POSIX journal / fsync / flock / CAS 语义。
- 真实 keyring / HSM / 密钥托管（S4.20 使用模拟边界，已披露）。
- approved backup inventory 与真实 restore adapter。
- production dispatcher 全路由 closed/unknown 矩阵（当前仅选定路由）。
- 进程级全局唯一 runtime 的运行期隔离断言（仅指针/组装层证明）。
- 真实 provider / ACL / CAS / anchor / lease / restore 行为。
- `enter` / `install` 与 recovery secret candidate 的承接任务（仍未登记，unknown）。
- `TestRecoveryRuntimeProductionChain`（`127.0.0.1:53519`）本轮 NOT RUN。
- `goimports` / `golangci` 未运行；非全仓格式检查。
- 人的业务接收、部署授权、Git 写授权。
- S4.20 `R02-F004` 残余：closed/unknown 非 production dispatcher E2E（R02 已列为 LOW 并披露）。

## 结论

**FAIL**。阻断项为 `B13H-GATE-F001`（HIGH）；未修复代码，按 FAIL/HIGH 要求停止并报告。复核条件：F001 修复或在 Lead 书面收窄合同后重新提交同修订证据，并重跑 S4.19 两层（层 a 真实 PG + 层 b production dispatcher 真实 resume→同会话 200）与 17 文件哈希核验。