# B6C-3 Integrated Code Review R02

- 日期：2026-10-25
- objectType：code
- Reviewer：Copernicus（原 integrated 非作者独立 Reviewer）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- `IDENTITY-B6C3-REV-R01-F001`：**OPEN（生产代码缺陷已修正，但原 finding 要求的跨卡回归证据未闭环）**
- `IDENTITY-S10.1-REV-R04-F002`：**CLOSED，未回退**
- 新的独立 production finding：**0**
- 残余验证缺口：**1 MEDIUM（沿用 R01-F001，不另建重复 finding）**
- **B6C-3 integrated Review Gate: CHANGES REQUIRED**

## 1. 固定 revision 与防漂移

| 对象 | SHA256 |
|---|---|
| `internal/identity/password_reset.go` | `8DE34FA4BDB7181893AB297FE3A316325DE93DB7C6B9348DDA0E482E4F4DCD1A` |
| `internal/identity/password_reset_test.go` | `12CEC6BDA65045697F9C7CA7707371E7BDE1881DD0391A646C5D854435B567D4` |
| 原报告 `.spec/wip/identity-auth/review-b6c-3-integrated-r01.md` | `BFBD0E6F669D816A28F00B3262D4D9A7BCFCF9A7DE7822E6E355096A0226D0A5` |

开始与结束均核对固定两文件、分支和 HEAD。另逐项重算 R01 清单中的 28 个 integrated 代码对象：除本轮预期变化的上述两文件外，其余 26 个对象均保持 R01 哈希；未发现第三个代码、公共合同、schema、migration、store、operator、worker 或 composition 对象变化。三者均处于共享未提交工作树中，本报告没有用空 `git diff` 代替哈希、全文检查或实际测试。

## 2. 两轮实际检查范围

### Round 1：修复对象与原 finding 定点复审

- 全文检查 `password_reset.go` 的候选创建、dispatch ownership、Mailer evidence、unknown/accepted 持久化及 reconciliation context。
- 全文检查 `password_reset_test.go` 本轮新增的 protection-window、accepted acceleration、parent cancellation 和 Gate/旧能力测试。
- 将 R01-F001 的四项复审条件逐项映射到生产语句、测试执行路径和断言，而非仅按测试名判断覆盖。

### Round 2：真实消费者与未变生产链

- 有界追踪未变的 `password_reset_worker.go`、`password_reset_operator.go`、PasswordResetStore/Gate、Login、PasswordChange、Session Resolve 及 `cmd/han-server` production composition/lifecycle。
- 核对 worker 的 scan/claim 时间谓词、operator 的 accepted/unaccepted settlement、同一 Gate 注入和唯一 Store/operator/worker/permit owner。
- 运行 identity 全部 PasswordReset 定向普通/race、三消费者与 production wiring/lifecycle 定向普通/race、相关 vet、gofmt、diff-check及测试发现检查。

## 3. `IDENTITY-B6C3-REV-R01-F001` 逐项结论

### 3.1 unknown evidence 不再提前 recovery job

**生产代码结论：已修正。**

- `password_reset.go:250-256` 仍把初始 recovery job 的 `next_attempt_at` 设为 `now + IdempotencyLock`。
- `password_reset.go:366-381` 的 unknown 分支现在仅调用 `RecordUnknownDelivery` 并提交；不再执行 recovery-job acceleration。
- Store 的既有 evidence SQL 仍只把 operation 写为 `state='commit_unknown'`、`delivery_state='unknown'`；请求线程没有 credential invalidation、operation/idempotency terminal、terminal audit 或 settlement。

### 3.2 protection window before / boundary / after

**生产代码静态链结论：语义单义。测试闭环结论：不足。**

- 未变 worker 在 `password_reset_worker.go:87-133,395-398` 只扫描/领取 `next_attempt_at <= now` 的 ready/retry_wait job。
- 因 unknown 保留初始 `next_attempt_at`，窗口前不会由该 SQL 返回或领取；边界时 `<=` 成立；边界后可领取。
- 窗口前没有 worker settlement，故 durable Gate 和旧能力保护不会被 operator 提前解除。边界后，未变 operator 才可走 `password_reset_operator.go:185-190,522-535` 的 unaccepted settlement，仅失效 new credential并终结该 stale operation。

但 `TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow`（`password_reset_test.go:1602-1700`）没有执行 worker `Reconcile`、backend `Scan/Claim`、operator settlement或三个 Gate 消费者。它只在 coordinator fake transaction 中保留 `nextAttemptAt`，随后于 `:1681` 用测试代码直接重复 `!nextAttemptAt.After(reconcileAt)`。因此：

- before/boundary/after 表格证明了本地时间比较，但没有证明 production worker SQL/claim 真正遵守该窗口；
- 没有实际断言窗口内 operator 零调用、无 settlement/terminal audit；
- 没有把同一 unknown operation 接入 Login、PasswordChange、Session Resolve 来证明窗口内 Gate 持续关闭；
- worker/operator 或连接代码以后回退时，该测试仍可能保持绿色。

这不满足 R01 明确要求的 coordinator→job→worker→operator 确定性链路回归及三消费者 Gate 断言。故原 finding 不能仅凭当前测试关闭。

### 3.3 accepted evidence

**CLOSED。**

`password_reset.go:339-363` 仍仅在 accepted evidence 持久化后执行 `acceleratePasswordResetRecoveryJobSQL`。`TestPasswordResetAcceptedDeliveryAcceleratesRecoveryJob` 验证 acceleration 调用与立即可领取时间；普通/race均通过。未发现 accepted 路径被 unknown 修复延迟。

### 3.4 terminal 状态与旧能力

**请求线程修复已确认；完整窗口回归仍受 3.2 限制。**

- unknown 请求线程测试确认没有 recovery acceleration、credential/session write 或 accepted/rejected 混写。
- `TestPasswordResetUnknownDeliveryPreservesGateAndOldCapability` 继续证明 unknown 落盘时 Gate 和旧 credential 能力状态未被直接破坏。
- 由于 protection-window 测试没有实际进入 worker/operator，原复审条件中的“窗口内不可 claim/settle”仍主要依赖静态组合证据，而非能防回退的跨卡执行证据。

### 3.5 最终状态

`IDENTITY-B6C3-REV-R01-F001`：**OPEN**。

原 HIGH production 缺陷的触发语句已经删除，当前生产组合按静态追踪具有正确窗口行为；剩余阻断是 **MEDIUM 验证缺口**：原 finding 的明确复审条件尚未由真实 worker/operator/Gate 消费链测试证明。

复审关闭条件：

1. 增加确定性测试，把 unknown coordinator 产生/保留的 recovery job 接到实际 worker `Reconcile` 与 operator seam；
2. before boundary 断言 scan/claim/settlement/operator 均未发生，operation/idempotency/credential/audit无终态，并验证 Login、PasswordChange、Session Gate 继续 fail closed；
3. at/after boundary 断言可领取且仅 invalidated new、old capability保留；
4. accepted evidence仍立即加速并走成功 settlement；
5. 上述定向普通与 race 回归通过，固定修订后再独立复审。

## 4. `IDENTITY-S10.1-REV-R04-F002` 状态

**CLOSED，完整消费链无回退。**

- R01 清单中 `handler.go`、store、worker、operator、Login、PasswordChange、Session及 `cmd/han-server` production composition/lifecycle均保持原哈希。
- production builder仍只构造一个 PasswordResetStore/Gate，并将同一实例注入三个消费者且复用于 operator/worker。
- 两条 public route、fail-fast依赖检查、唯一 operator/worker/permit owner和 lifecycle未变化。
- 三消费者与 production wiring/lifecycle 定向普通/race测试均通过。

本轮没有把 R01-F001 的残余测试缺口误归为历史 production wiring finding。

## 5. Findings

### CRITICAL / HIGH

无新的 CRITICAL 或 HIGH finding。R01-F001 的已知 HIGH 生产触发链已被代码修正。

### MEDIUM

无新的独立 ID。R01-F001 保持 OPEN 的残余部分为 MEDIUM：`password_reset_test.go:1602-1700` 以 fake transaction 加测试内时间比较代替真实 worker/operator/Gate执行，未满足原 finding 的跨卡回归关闭条件。

### LOW

无。

## 6. 实际执行命令与结果

以下命令均在固定工作区独立执行，未读取或使用 `HAN_TEST_POSTGRES_DSN`：

```text
go test ./internal/identity -run '^(TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow|TestPasswordResetAcceptedDeliveryAcceleratesRecoveryJob|TestPasswordResetDeliveryEvidenceIgnoresParentCancellation|TestPasswordResetUnknownDeliveryPreservesGateAndOldCapability)$' -count=1
go test -race ./internal/identity -run '^(TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow|TestPasswordResetAcceptedDeliveryAcceleratesRecoveryJob|TestPasswordResetDeliveryEvidenceIgnoresParentCancellation|TestPasswordResetUnknownDeliveryPreservesGateAndOldCapability)$' -count=1
```

PASS。

```text
go test ./internal/identity -run '^TestPasswordReset' -count=1
go test -race ./internal/identity -run '^TestPasswordReset' -count=1
```

PASS。

```text
go test ./internal/identity ./internal/session ./cmd/han-server -run '^(Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|SessionResolvePasswordResetGate|ProductionPasswordReset|PasswordResetProductionWiring|RunLifecyclePasswordReset))' -count=1
go test -race ./internal/identity ./internal/session ./cmd/han-server -run '^(Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|SessionResolvePasswordResetGate|ProductionPasswordReset|PasswordResetProductionWiring|RunLifecyclePasswordReset))' -count=1
```

三包定向普通/race均 PASS。

```text
go test ./internal/identity -list '^TestPasswordReset(UnknownDeliveryPreservesRecoveryProtectionWindow|AcceptedDeliveryAcceleratesRecoveryJob|DeliveryEvidenceIgnoresParentCancellation|UnknownDeliveryPreservesGateAndOldCapability)$'
```

四个精确测试均可发现。

```text
go vet ./internal/identity ./internal/session ./cmd/han-server
gofmt -l internal/identity/password_reset.go internal/identity/password_reset_test.go
git diff --check -- internal/identity/password_reset.go internal/identity/password_reset_test.go
```

全部 PASS；vet与gofmt无输出。两文件为 untracked，因此 diff-check仅作为空白错误检查，不作为 revision 证据，revision由 SHA256 固定。

另执行分支/HEAD、两修复对象、R01报告及 R01 28项代码对象 SHA256 核对；除两修复对象外的26项无漂移。

## 7. 未验证边界

- 未连接真实 PostgreSQL，未执行真实 SQL 时间边界、事务隔离或 migration integration；本轮依据生产 SQL全文、unit harness和未变对象哈希。
- 未连接真实 SMTP，未验证 provider acknowledgement丢失的真实网络行为。
- 未执行部署、Browser或 E2E。
- test-plan r20仍为 planned-not-executed，不冒充真实环境证据。
- 未重跑 Lead 的全仓 Manifest；本轮按定点复审要求执行修复精确测试、全部 PasswordReset、消费者/production wiring普通/race及相关静态检查。
- 共享工作树的其他大量改动不在本结论范围；本评审未回滚、覆盖、格式化、stage、commit、push、删除或移动任何既有内容，未触碰根目录 `han-server.exe`。

## 8. Gate

| 项目 | 结论 |
|---|---|
| unknown 不提前 recovery job | 生产代码通过 |
| protection-window生产状态机 | 静态组合通过 |
| before/boundary/after真实 worker/operator/Gate回归 | **未闭环** |
| accepted立即加速 | 通过 |
| 请求线程无 invalidation/terminal回退 | 通过 |
| `IDENTITY-S10.1-REV-R04-F002` | **CLOSED，未回退** |
| 新的独立 production finding | 0 |

**B6C-3 integrated Review Gate: CHANGES REQUIRED**

阻断原因：R01-F001 的生产触发语句虽已修正，但当前 before/boundary/after 测试没有实际执行 worker、operator和三个 Gate消费者，未达到该 finding 预先声明的跨卡回归关闭条件。
