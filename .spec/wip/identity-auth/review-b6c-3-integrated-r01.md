# B6C-3 Integrated Code Review R01

- 日期：2026-10-25
- objectType：code
- Reviewer：Copernicus（非作者独立 Reviewer）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审对象：当前共享未提交工作树中 S10.1～S10.4 password-reset 完整生产消费链
- `IDENTITY-S10.1-REV-R04-F002`：**CLOSED，完整消费链未回退**
- 新 findings：**1 HIGH**
- **B6C-3 integrated Review Gate: CHANGES REQUIRED**

## 1. 固定 revision、对象与防漂移

开始与结束均核对分支、HEAD 和下列 28 个代码对象 SHA256。对象在评审期间无漂移；本报告没有用空 `git diff` 代替固定哈希、全文检查和实际测试。

| 文件 | SHA256 |
|---|---|
| `migrations/0008_password_reset_durable.sql` | `37FB3077709462D62C763FE87D4E04B4BDA4CFC498C14C2860B27429D5036980` |
| `internal/identity/model.go` | `4D17E1DBC264C69BAFE3330619098F0C57AD703EC5EA5E611BCE8E2BB548B5C7` |
| `internal/identity/temporary_credential.go` | `66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5` |
| `internal/identity/temporary_credential_test.go` | `659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D` |
| `internal/identity/password_reset_store.go` | `6C0136408EF62A7B5FC80C4C557AEBCB557457B772EB96E8DA90024C42D3315F` |
| `internal/identity/password_reset_store_test.go` | `62EAE06BD41E5F3CFA7AAB03B3721B8D3A964834A0E7C98B194181A2791827E6` |
| `internal/idempotency/store.go` | `AA0483969B12B058B42DB70A9763501B60ED055AFB91E5ECD6AA7F5C45FDD657` |
| `internal/idempotency/store_test.go` | `3A6EF9313839ED19BE28D4FB635C016DF51F6C58213DDAE88D93A2A65CC49BE7` |
| `internal/identity/handler.go` | `FCCF0290BEE93E7730AD8EE8B65A07B8BFC475D0BEA14C5A3AB2372D2C49C36C` |
| `internal/identity/password_reset.go` | `926E4507580F608AB37D13ACA88EDF8295DCC50BAE541168655326B8D33724FA` |
| `internal/identity/password_reset_test.go` | `046BF47AD5804AA3E1B5308C0AB3A8458F9AD245C8017F094C563BAF874C3040` |
| `internal/identity/login.go` | `FDB1ED13FB499A1219AE24884F053A3626567BB81CAAB56EC02B1FC197B6864C` |
| `internal/identity/login_test.go` | `FB5483FB93649F9B01B060D54F35B5CC7B7A3305704D19DFCA445A2C7400414F` |
| `internal/identity/login_integration_test.go` | `15C28D7BA7EE39499D970E8C7AE53F7CE28A1A450D2530FD7697BF8BE4D407BA` |
| `internal/identity/service.go` | `4371ED7348A7035BDB6CAA57CE55F8621FF41CDE5D8D7E5615D54789F88A4F40` |
| `internal/identity/bootstrap.go` | `AEF53946AB8C55A3CC03DAE03DC1CD0E42BE12553D9619D45C7F491F154CAAA5` |
| `internal/identity/bootstrap_test.go` | `C33C31A22C32DA0FFE57099C37C3E6EE4826F4F082359F2B7A7E2B1CB43E2B45` |
| `internal/identity/administrator_create_test.go` | `AC9FE098FA3F1F09FA4944BB5BD7653BF375B23C08ED1E63F13DCEE45683EEAE` |
| `internal/session/store.go` | `4B12462E29D46B730F0DD58E6A0BAA57EC1D91B99D9F43FC91179319FF88E679` |
| `internal/session/store_test.go` | `3D32E65D407F24211355B642DAB3933F1AB36E10B60CAFC7F039B2F9B8201E09` |
| `internal/identity/password_reset_worker.go` | `9BF1A9B1230004422E9ABC563329C9C7E44BB48CC1E1452CB5DF18E20D704C90` |
| `internal/identity/password_reset_worker_test.go` | `DC6544C4390CC36FC93ED4B66D530E2E0FBC067A7C733FFF6692944CBC4801CD` |
| `internal/identity/password_reset_operator.go` | `4DC973ECB2272E07083884224A898D0C7C38E4CFEE8B03E44DA14E9D9F951F42` |
| `internal/identity/password_reset_operator_test.go` | `9463D7B8B098A21BA3F68BDB190CFCF51C6FA47038BA0BD8A7E7505ADD210E02` |
| `cmd/han-server/main.go` | `38419E8708CE1FACAFC164D344D3D9A9F5E673D873529CD61810C2A6E63C6166` |
| `cmd/han-server/main_test.go` | `77062DAE8D69251EF731AF548D0A63294C4D5648E61BE8EF3F9958EC9D26085C` |
| `cmd/han-server/lifecycle.go` | `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888` |
| `cmd/han-server/lifecycle_test.go` | `B3AA659B1A98E0AE8A3EF09678309866DC3D46345B2A6C29A5A220D700B36DDF` |

关键依据固定哈希：

| 依据 | SHA256 |
|---|---|
| `architecture.md` r27 | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` |
| `boundary.md` r13 | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` |
| `design.md` r44 | `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23` |
| `stories.md` r16 | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` |
| `test-plan.md` r20 | `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB` |
| `decisions/identity.md` | `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978` |
| S10.1 R06 | `2EA7503C7F33DEED7D697ECEC26ECFBA6BB7E0CD5995EF52BA12936A3D4117FC` |
| S10.2 R05 | `8B904EEDA7B3DF93AA6E1B3D632A9970F0CEAACDDF3E3BB0B83C37F728ABC724` |
| S10.3 R02 | `40D264C2E694E9CD5596DAD3FC1C574385F4CC37B08A22CA3D3127AB4769287F` |
| S10.4 R03 | `6CAC72E65FC85678CB0DAEF371367D4465EAB5ED41A3282A65DCD032E746D37C` |

Tasks/Plan 仅按结束时可见合同只读消费，按委派要求不作为代码对象防漂移条件。

## 2. 两轮实际检查范围

### Round 1：生产者、状态模型与直接实现

- 全文检查 `0008_password_reset_durable.sql`、model、TemporaryCredential、PasswordResetStore、共享 Idempotency store。
- 核对 operation/job、六字段幂等 identity、execution generation、request kind、predecessor/replaced、delivery/settlement、deferred FK、唯一性、CHECK、索引及 credential lineage。
- 全文检查首次 reset Handler/coordinator、same-key 预分类、dispatch ownership、Mailer evidence、secret 清理和公共错误/envelope。
- 全文检查 operator/worker 的 scan/claim/renew/fail/settle、lease CAS、permit、commit acknowledgement 三态、dead-letter 和生命周期。
- 全文检查 production builder/router/config/lifecycle，核唯一 Store/Gate/operator/worker/permit owner、两条 route、安全 middleware 与五项必填无默认配置。

### Round 2：真实消费者与跨卡失败链

- 沿同一 production Gate 追踪 Login、PasswordChange、Session Resolve；核 User-first、fail closed、错误映射和 Session caller-owned transaction/commit barrier。
- 从 Mailer non-nil error 逐步追踪到 unknown evidence、job 调度、worker claim、operator settlement、credential/User/Session/Idempotency/Audit 终态。
- 检查 production composition/router 测试是否经过真实 builder/router/security chain，确认 test seam 未进入 production 默认路径。
- 复核 R03、R04、R05、R06 关闭链及四张单卡关闭报告，区分历史 finding 与当前跨卡组合缺陷。
- 检查 B6D/S11.1 所需 primitive 是否已由 schema/合同预留；未评审或实现尚未启动的 S11.1。

## 3. Findings

### HIGH — `IDENTITY-B6C3-REV-R01-F001`：Mailer unknown 被立即加速并由 worker 不经保护窗口终结为 invalidated

- **位置**：
  - `internal/identity/password_reset.go:366-385,676`
  - `internal/identity/password_reset_worker.go:87-133,395-398`
  - `internal/identity/password_reset_operator.go:185-190,522-535,574-577`
  - 测试缺口：`internal/identity/password_reset_test.go:1449-1668` 只证明请求线程落盘时没有直接 invalidation；没有把该 unknown operation 接到真实 worker/operator 后继续断言保护窗口内保持 Gate。
- **违反依据**：
  - design r44 `:2154-2157`：`commit_unknown` 仅在保护期后仍无 accepted evidence 才可转为 invalidated。
  - design r44 `:2403-2404,2971`：unknown 保持 unresolved/Gate，由 worker 收敛；无 accepted 证据的 stale candidate 才只失效 new。
  - stories r16 `:351-352,387-388`：worker 不重发，stale `prepared/dispatching/commit_unknown` 才可 invalidated。
  - test-plan r20 `I-PRD-06～10` 与 `:652`：保护窗口必须有 before/boundary/after 的确定性验证。
  - tasks 当前合同 `:120`：请求线程有限 cleanup 不能证明 commit-unknown 后 durable fail-closed。
- **触发场景**：
  1. SMTP provider 实际可能已受理邮件，但 `Mailer.Send` 因 acknowledgement 丢失、连接错误或等价不确定结果返回 non-nil error。
  2. Coordinator 正确写入 `state=commit_unknown`、`delivery_state=unknown`，但同一事务又把 recovery job 的 `next_attempt_at` 改为当前时间。
  3. worker 的扫描/领取只检查 `next_attempt_at <= now`，因此下一次 Reconcile 可立即领取。
  4. operator 对任意非 accepted 的 `prepared/dispatching/commit_unknown` 无条件执行 unaccepted settlement，失效 new credential、把 operation 终结为 invalidated、把幂等记录写成稳定失败并完成 job；没有 stale/保护窗口判定。
- **影响**：
  - durable Gate 只维持到下一次 worker Reconcile，而不是合同要求的保护窗口；管理员发起 reset 后，旧 credential/Session 能力可能过早恢复。
  - provider 已受理的邮件中 new secret 被不可逆失效，same-key replay 从 uncertain 变为稳定 delivery failure。
  - operation、credential、IdempotencyRecord 和 job 被写入终态，后续无法再用 accepted evidence 安全归并；这是跨 coordinator→worker 的状态机错误，不是单纯可观测性或测试风格问题。
- **证据**：
  - 新候选 initial job 原本在 `password_reset.go:256` 使用 `now + IdempotencyLock`，且 production 配置保证 lock 大于 SMTP budget，用于形成保护窗口。
  - unknown 路径在 `:378` 将该时间覆写为当前时间。
  - worker SQL `password_reset_worker.go:395-398` 不含 operation age、delivery-result age 或其他保护期谓词。
  - operator switch `password_reset_operator.go:185-190` 和 invalidation SQL `:574-576` 也不含保护期谓词。
  - 精确 unknown、operator、worker 普通/race 测试均为绿色，但现有测试没有串起上述三段真实控制流，因此绿色结果不能反证本 finding。
- **建议**：
  - unknown evidence 不得把 job 提前到当前时间；保留 initial durable `next_attempt_at`，或以等价 durable `eligible_after` 条件同时约束 scan/claim/operator。
  - accepted evidence仍可立即加速 settlement；unknown 必须在保护窗口 before/boundary/after 上有单义行为。
  - 增加 coordinator→job→worker→operator 的确定性链路测试：窗口前不可 claim/settle且三消费者 Gate 持续关闭；边界后仍无 accepted evidence才只 invalidated new并保留 old；accepted 分支仍立即收敛成功。
- **复审条件**：
  1. 提供固定修复 revision与受影响文件哈希。
  2. 上述保护窗口 before/boundary/after 普通与 race 回归通过。
  3. unknown 在窗口内不写 credential invalidation、operation/idempotency终态或 terminal audit；accepted仍可及时 settlement。
  4. Login、PasswordChange、Session Resolve 在窗口内继续使用同一 Gate fail closed。

除该项外，未发现新的 CRITICAL/MEDIUM/LOW finding。

## 4. 原 `IDENTITY-S10.1-REV-R04-F002` 完整消费链状态

**CLOSED，未回退。**

- 唯一 production builder 构造一次 `*identity.PasswordResetStore`，同一实例作为 Gate 注入 Login、PasswordChange、Session Resolve，并复用于 operator/worker。
- production 构造一次 operator、worker 和 connection permit owner；未发现第二 Store/Gate/operator/worker、第二 composition root、包级隐藏 global 或 service locator。
- 两条 exact POST route 已进入 production local router，并经 Trace、method/path、Origin、Session、CSRF、Access 与业务管理员授权；拒绝路径不进入业务下游。
- 缺少 durable/security 依赖、五项 worker 配置非法或 budget 超出 pool 上限均在 route/listener开放前 fail closed。
- R03 三项及 R04-F001 的 same-key replay、unknown evidence context、secret safety 与 production route关闭链均未回退。

本报告的 HIGH finding 是 R04-F002 关闭后沿完整 coordinator→worker 消费链发现的独立跨卡状态机缺陷，不把历史“production route 不可达”重新标成 OPEN。

## 5. 其他合同结论

- migration/store/model 已提供 S10.1～S10.4 所需 operation/job、六字段幂等、credential lineage、request kind/counter/predecessor/replaced、delivery/settlement primitive。
- schema/合同也已预留 B6D/S11.1 所需 successor generation、predecessor/replaced 和 purpose 分流 primitive；这不表示 S11.1 已实施或可启动。
- S10.5 maintenance mode 不属于本 Gate；当前未发现伪 production maintenance 入口。

## 6. 实际执行命令与结果

```text
go test ./internal/identity -run '^TestPasswordReset' -count=1
```

PASS：`ok han/internal/identity 0.153s`

```text
go test -race ./internal/identity -run '^TestPasswordReset' -count=1
```

PASS：`ok han/internal/identity 1.184s`

```text
go test ./cmd/han-server -run '^(TestProductionPasswordReset|TestPasswordResetWorker|TestRunLifecyclePasswordReset)' -count=1
go test -race ./cmd/han-server -run '^(TestProductionPasswordReset|TestPasswordResetWorker|TestRunLifecyclePasswordReset)' -count=1
```

均 PASS。

```text
go test ./internal/identity ./internal/session ./internal/idempotency ./internal/migration -run '^(Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|PasswordReset(Gate|Store|Operator|Worker|Durable|Delivery|Unknown|RequiresExisting|Dispatch|Accepted|SameKey|Replay)|SessionResolvePasswordResetGate|IdempotencyPasswordReset|Load_ValidFiles|ValidateSQLTransactionControlConformance|Load_TransactionControlConformance))' -count=1
go test -race ./internal/identity ./internal/session ./internal/idempotency ./internal/migration -run '^(Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|PasswordReset(Gate|Store|Operator|Worker|Durable|Delivery|Unknown|RequiresExisting|Dispatch|Accepted|SameKey|Replay)|SessionResolvePasswordResetGate|IdempotencyPasswordReset|Load_ValidFiles|ValidateSQLTransactionControlConformance|Load_TransactionControlConformance))' -count=1
```

四包普通/race均 PASS。

```text
go test ./internal/identity -run '^(TestPasswordResetUnknownDeliveryPreservesGateAndOldCapability|TestPasswordResetDeliveryEvidenceIgnoresParentCancellation|TestPasswordResetOperatorAcceptedSettlementExecutesCompleteTransaction|TestPasswordResetWorkerCommitUnknownDoesNotFailOrStop|TestPasswordResetWorkerRetryAndDeadLetterKeepAuthorityHeld)$' -count=1
go test -race ./internal/identity -run '^(TestPasswordResetUnknownDeliveryPreservesGateAndOldCapability|TestPasswordResetDeliveryEvidenceIgnoresParentCancellation|TestPasswordResetOperatorAcceptedSettlementExecutesCompleteTransaction|TestPasswordResetWorkerCommitUnknownDoesNotFailOrStop|TestPasswordResetWorkerRetryAndDeadLetterKeepAuthorityHeld)$' -count=1
```

均 PASS；同时确认这些绿色测试没有执行 unknown evidence 后的完整 worker invalidation链。

```text
go vet ./internal/migration ./internal/identity ./internal/idempotency ./internal/session ./cmd/han-server
```

PASS，无输出。

```text
gofmt -l <27个Go固定对象>
```

PASS，无输出。

```text
git diff --check -- <28个固定对象>
```

PASS，无输出；未跟踪对象另以 SHA256、全文检查和实际测试固定，未把该命令单独作为评审证据。

```text
go mod verify
```

PASS：`all modules verified`

另执行 `go test -list` 定位 production route/lifecycle、三消费者、store/operator/worker/idempotency 的真实测试入口；没有运行会覆盖根目录既有 `han-server.exe` 的构建命令。

## 7. 未验证风险与边界

- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`；未连接真实 PostgreSQL。
- 未执行真实 SMTP/TLS/AUTH/provider受理、外部收件、Browser、部署或 E2E。
- 未验证 PostgreSQL 18 的真实锁等待、commit acknowledgement loss、多连接/多 worker、migration混跑/回退和 crash/restart。
- 未执行 target-environment 容量、worker backlog、unresolved age、性能或 production readiness 验证。
- test-plan r20 是 planned-not-executed 验收依据，不在本报告中冒充已执行真实环境证据。
- 共享工作树其他大量改动不在本结论范围；未回滚、覆盖、格式化、stage、commit、push、删除或移动任何既有内容，根目录 `han-server.exe` 未触碰。

## 8. Gate

| 项目 | 结论 |
|---|---|
| migration/store/model与三消费者共享Gate | 通过本轮静态与定向回归检查 |
| production composition/routes/operator/worker/lifecycle | 构造与可达性通过；unknown保护窗口状态机存在 HIGH finding |
| `IDENTITY-S10.1-REV-R04-F002` | **CLOSED，未回退** |
| B6D/S11.1 primitive预留 | 已预留；S11.1未评审、未实施 |
| 新 findings | **1 HIGH / 0 MEDIUM / 0 LOW** |

**B6C-3 integrated Review Gate: CHANGES REQUIRED**

阻断原因：`IDENTITY-B6C3-REV-R01-F001` 使 Mailer unknown 在完整生产消费链上绕过保护窗口并过早终结 durable Gate。修复和独立复审通过前，不应关闭 B6C-3，也不应据本批次启动 B6D/S11.1。
