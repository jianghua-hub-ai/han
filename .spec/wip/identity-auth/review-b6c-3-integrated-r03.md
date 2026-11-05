# B6C-3 Integrated Code Review R03

- 日期：2026-10-25
- objectType：code
- Reviewer：Copernicus（原 integrated 非作者独立 Reviewer）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- `IDENTITY-B6C3-REV-R01-F001`：**CLOSED**
- R02 残余 MEDIUM 测试缺口：**CLOSED**
- `IDENTITY-S10.1-REV-R04-F002`：**CLOSED，未回退**
- 新 findings：**0**
- **B6C-3 integrated Review Gate: PASS**

## 1. 固定对象、revision 与防漂移

| 对象 | SHA256 |
|---|---|
| `internal/identity/password_reset.go` | `8DE34FA4BDB7181893AB297FE3A316325DE93DB7C6B9348DDA0E482E4F4DCD1A` |
| `internal/identity/password_reset_test.go` | `FC5C69B6D1F07FBD48C8CA437AFA3DD4C7BADBC6393F482F6906E86AF878B29F` |
| 原报告 `.spec/wip/identity-auth/review-b6c-3-integrated-r02.md` | `BD882730034DC80ADB919EDF8775CB4375D630E200495E90709A8C42AA9C850E` |

开始与结束均核对分支、HEAD、固定对象及报告哈希。另重算 R01 integrated 清单：除上述两个历次修复对象外，其余 26 个代码对象均保持 R01 固定哈希；本轮没有第三文件、公共合同、schema、migration、store、operator、worker 或 production composition 变更。两代码文件均处于共享未提交工作树中，没有用空 `git diff` 代替全文检查、固定哈希与实际测试。

## 2. 两轮实际检查范围

### Round 1：新增状态化测试与修复对象

- 全文复核 `password_reset.go`；确认 R02 已验证的生产修复未变：unknown evidence只持久化 `commit_unknown/unknown`，不加速 recovery job；accepted evidence仍立即加速。
- 全文检查 `password_reset_test.go`，重点审查 `TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow` 及其 state、pool、transaction、Gate、consumer assertion harness。
- 逐项核对 R02 §3.2 和五项复审关闭条件，确认测试执行路径而非仅依赖测试名称或本地时间比较。

### Round 2：真实 worker/backend/operator seam 与消费者

- 跟踪测试从 coordinator dispatch 到实际 `PasswordResetRecoveryWorker.Reconcile`。
- 确认 worker 使用 production `passwordResetPostgresBackend` 的 `Scan → Claim → Renew`，并经 worker `settle` 调用 `PasswordResetOperator.executeHeld` 的 operator seam。
- 核对 before/boundary/after、accepted、三消费者 Gate、old capability、permit、operation/job/credential/idempotency/audit状态断言。
- 复跑全部 PasswordReset、Login/PasswordChange/Session及 production wiring/lifecycle定向普通/race回归，并检查 vet、gofmt、whitespace和测试发现。

## 3. `IDENTITY-B6C3-REV-R01-F001` 定点结论

### 3.1 unknown 不提前 recovery job

**CLOSED。**

- `password_reset.go:366-381` 的 unknown evidence路径仍只执行 `RecordUnknownDelivery`与事务提交。
- 新 harness 的 transaction seam识别实际 acceleration SQL并计数；unknown路径要求 `accelerationCalls == 0`。
- 如果恢复旧版 unknown acceleration mutation，before-boundary case会把 `nextAttemptAt`提前至当前时刻，实际 worker scan会返回候选并进入claim/settlement，从而违反“仅scan”和unresolved断言；测试能够捕获该回退。

### 3.2 before protection boundary

**CLOSED。**

`password_reset_test.go:1604-1675` 的 before case实际调用 `harness.worker.Reconcile`：

- worker使用 production `passwordResetPostgresBackend.Scan`；
- pool seam核对实际 SQL等于 `passwordResetRecoveryScanSQL`且仍含 `next_attempt_at <= $1`；
- protected job返回零候选，调用序列严格为 `scan`，没有 `claim`、`renew`、operator或settlement；
- operation保持 `commit_unknown`、delivery保持 `unknown`、settlement保持pending、new credential保持pending_delivery、idempotency保持processing、job保持ready；
- attempt、terminal audit均为零，old capability和已有Session数量不变。

同一状态和同一 Gate实例随后实际进入：

- `Service.Login`：在密码验证和Session创建前fail closed；
- `Service.ChangePassword`：在credential验证、digest写入和Session rotation前fail closed；
- `session.Store.ResolvePrincipal`：返回无principal并回滚caller-owned transaction。

三条消费者均实际调用共享 Gate，且测试断言总调用次数为3。

### 3.3 at / after protection boundary

**CLOSED。**

- boundary和after case均实际经过 production backend的 `Scan → Claim → Renew`，调用序列为 `scan, claim, renew, operator`。
- scan与claim seam都核对production SQL仍含时间资格谓词。
- operator seam只把new credential转为`recovery_invalidated`，operation/idempotency/job进入对应invalidated/failed/succeeded终态并写一次terminal audit。
- old capability保持，已有Session数量保持；随后实际调用 `Service.Login`，旧密码验证和Session创建成功。

边界语义与production的`<=`一致：before不可领取，at和after可领取。

### 3.4 accepted立即加速并成功收敛

**CLOSED。**

- accepted dispatch实际执行 recovery-job acceleration，`nextAttemptAt`变为当前时刻且计数为1。
- 在初始保护窗口尚未到期时立即调用同一 worker链，仍实际经过 `Scan → Claim → Renew → operator`。
- 最终operation为succeeded、new credential为active、idempotency为succeeded、job完成、Session撤销、terminal audit一次，旧能力被替换。
- 若accepted acceleration丢失，该case在当前时刻只会scan零候选并直接违反settled与调用序列断言。

### 3.5 最终状态

`IDENTITY-B6C3-REV-R01-F001`：**CLOSED**。

R01 的生产缺陷已由固定 `password_reset.go` 修复；R02 保持OPEN的残余 MEDIUM测试缺口已由本轮状态化跨卡测试闭环。普通与race执行均通过。

## 4. `IDENTITY-S10.1-REV-R04-F002`

**CLOSED，完整消费链无回退。**

- handler、store、worker、operator、Login、PasswordChange、Session、production composition/lifecycle等非本轮对象均保持R01固定哈希。
- production仍复用唯一 PasswordResetStore/Gate、operator、worker和permit owner。
- 三消费者及production wiring/lifecycle定向普通/race回归均通过。
- 新状态化测试使用同一 Gate实例进入真实消费者，没有引入第二production authority、composition root或隐藏global。

## 5. Findings

### CRITICAL / HIGH / MEDIUM / LOW

**无新 finding。**

本轮新增内容位于测试对象；未发现测试seam污染production、绕开production worker/backend控制流、伪造消费者调用或削弱原安全/幂等/secret合同。

## 6. 实际执行命令与结果

```text
go test ./internal/identity -run '^TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow$' -count=1
go test -race ./internal/identity -run '^TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow$' -count=1
go test ./internal/identity -run '^TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow$' -count=1 -v
```

全部PASS；verbose输出确认四个case均实际执行：

- `unknown_before_protection_boundary`
- `unknown_at_protection_boundary`
- `unknown_after_protection_boundary`
- `accepted_immediately`

```text
go test ./internal/identity -run '^TestPasswordReset' -count=1
go test -race ./internal/identity -run '^TestPasswordReset' -count=1
```

全部PASS。

```text
go test ./internal/identity ./internal/session ./cmd/han-server -run '^(Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|SessionResolvePasswordResetGate|ProductionPasswordReset|PasswordResetProductionWiring|RunLifecyclePasswordReset))' -count=1
go test -race ./internal/identity ./internal/session ./cmd/han-server -run '^(Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|SessionResolvePasswordResetGate|ProductionPasswordReset|PasswordResetProductionWiring|RunLifecyclePasswordReset))' -count=1
```

三包定向普通/race均PASS。

```text
go test ./internal/identity -list '^TestPasswordResetUnknownDeliveryPreservesRecoveryProtectionWindow$'
go vet ./internal/identity ./internal/session ./cmd/han-server
gofmt -l internal/identity/password_reset.go internal/identity/password_reset_test.go
git diff --check -- internal/identity/password_reset.go internal/identity/password_reset_test.go
```

- 精确测试可发现；
- vet无输出；
- gofmt无输出；
- diff-check PASS。两代码文件为untracked，diff-check仅作为whitespace检查，revision证据仍为SHA256。

另执行分支、HEAD、两个修复对象、R02报告及R01其余26项代码对象SHA256核对，均无评审期漂移。

## 7. 未验证边界

- 未连接真实PostgreSQL；pool/transaction为确定性状态化seam，但实际执行了production worker、production Postgres backend方法及其SQL常量。
- operator使用production worker到`PasswordResetOperator.executeHeld`的调用链，最终数据库settlement由状态化operator seam承载；未重复真实PG事务集成。
- 未连接真实SMTP，Mailer使用test stub。
- 未执行部署、Browser或E2E；test-plan r20仍为planned-not-executed。
- 未读取或使用`HAN_TEST_POSTGRES_DSN`或其他凭据。
- 未重跑Lead的全仓Manifest；本轮独立证据限定为精确跨卡测试、全部PasswordReset、直接消费者/production wiring普通与race及相关静态检查。
- 共享工作树其他大量改动不在本结论范围；本评审未回滚、覆盖、格式化、stage、commit、push、删除或移动任何既有内容，未触碰根目录`han-server.exe`。

## 8. Gate

| 项目 | 结论 |
|---|---|
| unknown不提前recovery job | 通过 |
| before仅scan、零claim/renew/operator/settlement | 通过 |
| 三消费者共享Gate并fail closed | 通过 |
| boundary/after领取且仅invalidated new | 通过 |
| old capability保留 | 通过 |
| accepted立即加速并同链成功收敛 | 通过 |
| 旧unknown acceleration mutation可被测试捕获 | 通过 |
| R02残余MEDIUM | **CLOSED** |
| `IDENTITY-S10.1-REV-R04-F002` | **CLOSED，未回退** |
| 新 findings | 0 |

**B6C-3 integrated Review Gate: PASS**
