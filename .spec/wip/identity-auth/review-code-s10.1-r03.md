# identity-auth S10.1 独立代码复审 R03

- 评审日期：2026-10-25
- Reviewer：非作者独立 Reviewer（add-reviewer）
- objectType：code
- 分支：`main`
- 基线 HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 旧报告：`.spec/wip/identity-auth/review-code-s10.1-r02.md`
- 目标：定点复审 `IDENTITY-S10.1-REV-R01-F002`，并检查 User-first durable operation/job 改造引入的新缺陷
- 结论：旧 F002 **CLOSED**；发现 **2 Major、1 Minor**；**Code Review Gate：CHANGES REQUIRED**

## 1. 固定对象与防漂移

本次对象是共享工作树中的未跟踪文件，不能用空 `git diff` 代替评审。Reviewer 按全文审查，并在落报告前重新核对 SHA256：

| 文件 | 固定 SHA256 |
|---|---|
| `internal/identity/handler.go` | `FCCF0290BEE93E7730AD8EE8B65A07B8BFC475D0BEA14C5A3AB2372D2C49C36C` |
| `internal/identity/password_reset.go` | `943BA61E495D86AE959E8AC2AA04112BD0F0D65105524251FB89BDE8E1850C92` |
| `internal/identity/password_reset_test.go` | `8D254B33F4B18B22613113DE8EE51A0105EAC6E062FA6FA7326F1A876AD60A5D` |

落报告前分支、HEAD、三文件哈希均与委托固定值一致，R03 报告路径此前不存在。

## 2. 评审依据

实际读取并应用：

- Reviewer Role、`add-code-review` 方法；
- `code-review-checklist`、`abstraction-principles`；
- core coding、no-hardcode、observability、tracing；
- 已启用 `go-service` coding/security rules；
- `add-code-review/references/产品安全红线.md`；
- identity design r44、stories r16、test-plan r20 中 S10、I-PRD、I-CRED 相关合同；
- tasks/Plan 中 S10.1～S10.4 现役合同；
- R02 报告及本轮三个固定对象。

重点合同：

- S10.1 消费 S10.2/S10.3/S10.4 既有唯一 PasswordResetStore、Gate、operator、worker；
- User-first durable operation/job；
- same-key replay 不生成新 secret、不增代、不发信，稳定重放既有结果；
- delivery outcome 必须区分 accepted、明确 rejected、unknown；
- unknown 保持 unresolved Gate，不能被伪装成明确失败；
- 请求线程不直接激活凭据或撤销 Session；
- 公共 REST、权限、`Idempotency-Key`、响应 envelope、稳定错误语义和 secret safety 不变；
- 禁止第二 Store/Gate/classifier/operator/composition root 和隐藏全局。

## 3. 两轮实际检查范围

### Round 1：固定对象直接审查

逐行检查三个固定对象，覆盖：

- Handler 的权限、请求解析、幂等键、公共 envelope 与错误映射；
- coordinator 的候选准备、分类、dispatch、delivery evidence、job 加速与返回语义；
- commit unknown、cleanup 双失败、accepted/rejected/unknown、succeeded/replay 与 credential 状态一致性；
- secret 生命周期、日志/错误泄露、硬编码、隐藏状态及重复抽象；
- 测试是否真实注册为顶层 Go test，以及断言是否覆盖目标合同。

### Round 2：真实消费者与上游依据追踪

按真实调用关系有界追踪：

- `password_reset_store.go`、相关 model/SQL；
- 既有 password-reset operator、worker；
- Login、PasswordChange、Session Gate；
- `cmd/han-server` 唯一 production composition；
- 必要的 mailer adapter/test 与相关回归测试。

未预读或评价无关全仓内容。

## 4. 旧 Finding 复审

### `IDENTITY-S10.1-REV-R01-F002` — CLOSED

R02 所述 durable operation/job 与 credential/session 一致性缺口，在本固定 revision 中已闭合：

- `internal/identity/password_reset.go:140-162`：请求先持久化候选事务，再进入 dispatch；请求线程不直接激活凭据或撤销 Session。
- `internal/identity/password_reset.go:270-302`：只有 `prepared → dispatching` 的提交结果明确成功后才发送邮件；dispatch commit error 不发信。
- `internal/identity/password_reset.go:305-329`：accepted 后仅持久化 delivery evidence 并加速既有 job。
- `internal/identity/password_reset_operator.go:474-519`：既有 operator 负责原子激活新 credential、推进 generation、撤销 Session，并完成 operation/idempotency。
- `internal/identity/password_reset_operator.go:522-535`：明确未受理或没有 accepted evidence 时只失效新候选，不削弱旧 credential。
- `internal/identity/password_reset_worker.go:87-159`：既有 durable worker 负责收敛；settlement commit unknown 不被误标为明确失败。
- Login、PasswordChange、Session 均在 User lock 后检查同一 Gate：
  - `internal/identity/login.go:1240-1285`
  - `internal/identity/service.go:474-498`
  - `internal/session/store.go:292-322`
- `cmd/han-server/main.go:2081-2095,2125-2153,2216-2244` 只装配一套 production Store/Gate/operator/worker。

因此，旧 F002 针对 commit unknown、cleanup 双失败、邮件凭据/派生 Session、succeeded/replay 与 credential 状态一致性的原始阻断项判定为 **CLOSED**。该关闭不覆盖下述新发现。

## 5. 新 Findings

### Major — `IDENTITY-S10.1-REV-R03-F001`：same-key replay 在分类前生成新秘密、ID 和 hash

- 位置：`internal/identity/password_reset.go:195-236`
- 违反依据：
  - design r44：same-key replay/稳定结果合同（相关段落约 2122、2198）；
  - stories r16：S10 replay 合同（相关段落约 349）；
  - test-plan r20：`I-PRD-02`。
- 证据：
  - `Secrets.Generate`、密码策略校验、两个 `NewID`、读取 latest credential、构造并 hash 新 credential，均发生在 `BeginOrClassifyCandidate` 之前。
  - Store 直到 `internal/identity/password_reset_store.go:129-166` 才分类 same-key。
  - 当前测试文件中的 `runPasswordResetIdempotencyCase` 已包含 `fixture.secrets.calls == 0` 意图，但该 helper 没有被顶层测试调用。
- 触发场景：
  1. 客户端使用同一 `Idempotency-Key` 重放已经 succeeded、failed 或 unresolved 的请求；
  2. 重放在分类前遇到随机源、ID 生成、hash 或 latest credential 查询故障。
- 影响：
  - 本应稳定重放的结果可能被改写成 500、503 或 uncertain；
  - replay 无必要地产生敏感 secret；
  - 违反“不生成 secret、不增代、不发信”的明确幂等合同。
- 修复建议：
  - 在生成 secret、ID、hash 及读取候选上下文前完成同键分类；
  - 只有确认是新 key/new candidate 后才准备候选；
  - 增加 coordinator 级 replay 测试，断言 secret、ID、hash、latest credential query、Mailer 调用均为零。
- 复审条件：
  - 提供固定修订及可执行 replay 回归；
  - 覆盖 succeeded、failed、unresolved 三类同键重放和准备阶段依赖故障。

### Major — `IDENTITY-S10.1-REV-R03-F002`：Mailer 任意 error 均被持久化为明确 rejected，无法表达 provider unknown

- 位置：
  - `internal/identity/password_reset.go:289-295`
  - `internal/identity/password_reset.go:332-367`
  - `internal/mailer/smtp.go:131-145`
- 违反依据：
  - design r44：delivery outcome 与 unknown 收敛合同（相关段落约 2103-2110、2130）；
  - stories r16：accepted/rejected/unknown 合同（相关段落约 351-352）；
  - test-plan r20：provider unknown 场景及 `I-PRD-07`。
- 证据：
  - `Mailer.Send` 返回任意 error 时，coordinator 都调用 `recordRejectedPasswordReset`，直接落为 `failed/rejected/invalidated`。
  - SMTP adapter 在 DATA 内容已经发送后等待最终 `250` 时，若超时或断链，`writer.Close()` 会返回 error；此时无法证明 provider 是否已受理。
  - `internal/mailer/smtp_test.go:24-103` 的 DATA acceptance reply 场景证明该阶段会返回 error。
  - schema/model 已支持 `delivery_state=unknown` / `state=commit_unknown`，但 coordinator 没有写入该 outcome 的路径。
- 触发场景：
  1. SMTP provider 已接收 DATA；
  2. 最终确认响应在客户端收到前超时或连接中断；
  3. `Mailer.Send` 返回 error，但远端是否受理不可判定。
- 影响：
  - 邮件可能已经送达，系统却持久化并稳定重放 `email-delivery-failed`；
  - operation/job 被终结，新 secret 被失效，丢失 unknown durable evidence；
  - Gate 被错误解除，违背 unknown 必须保持 unresolved 且不得重发的合同。
- 修复建议：
  - 先明确 Mailer 内部 outcome contract，区分 accepted、明确 rejected、unknown；
  - unknown 持久化为 unresolved/commit_unknown，保留 Gate，worker 不重发；
  - SMTP adapter 必须按协议阶段提供足够信息，不能把所有 error 等价为明确未受理。
- 方向决策：
  - 该修复可能改变 mailer 内部契约及 adapter 边界，需由 Lead/设计责任方确认；Reviewer 不代替作者修改。
- 复审条件：
  - 提供已确认的 delivery outcome 契约；
  - 提供 accepted、明确 rejected、DATA 后 ack loss/timeout 三类可执行测试；
  - 证明 unknown 保持 durable unresolved Gate 且不会重复发送。

### Minor — `IDENTITY-S10.1-REV-R03-F003`：大量 password-reset 回归 helper 未被任何顶层测试调用

- 位置：
  - `internal/identity/password_reset_test.go:44-129`
  - `internal/identity/password_reset_test.go:196-600`
- 违反依据：测试必须真实执行并证明权限、公共错误、幂等 replay、失败路径和 secret safety；仅存在 helper 不构成覆盖。
- 证据：
  - 固定测试文件只有 7 个顶层 `TestPasswordReset...`（约 `1308-1530`）。
  - `passwordResetAuthorizationCases`、`passwordResetIdempotencyCases`、preparation/failure/handler case tables 及对应 `run...` helper 没有调用者。
  - `go test ./internal/identity -list '^TestPasswordReset'` 未列出这些场景入口。
  - 未执行的 replay helper 已含 secret 调用次数断言，因此没有在测试中暴露 F001。
- 影响：
  - 测试代码外观造成覆盖错觉；
  - 权限、公共错误、replay 与失败路径可在 suite 全绿时回归。
- 修复建议：
  - 恢复顶层表驱动入口；或删除过期 helper，按当前 durable 协议重写可执行测试；
  - 用 `go test -list` 和定向故障注入证明目标场景实际运行。
- 复审条件：
  - 提供测试列表与运行输出，证明相关 case 被执行而非仅编译。

## 6. 独立验证

Reviewer 独立重跑以下命令，均 PASS：

```text
go test ./internal/identity -run '^TestPasswordReset' -count=1
go test -race ./internal/identity -run '^TestPasswordReset' -count=1
go test ./internal/identity -run '^(TestLoginPasswordResetGate|TestOnboardingPasswordChangePasswordResetGate)' -count=1
go test ./internal/session -run '^(TestPasswordResetGateConstruction|TestSessionResolvePasswordResetGate)' -count=1
go test ./cmd/han-server -run 'PasswordReset' -count=1
go test ./internal/mailer -run '^TestSMTPMailerCancellation' -count=1
go vet ./internal/identity ./internal/session ./internal/mailer ./cmd/han-server
git diff --check -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go
gofmt -l -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go
go test ./internal/identity -list '^TestPasswordReset'
```

说明：

- `gofmt -l` 无输出。
- `git diff --check` 返回成功；但三个固定对象均为 untracked，该命令对它们的覆盖有限，不能单独证明全文质量，已用固定哈希、全文审查和 `gofmt` 补充。
- Lead 提供的全量 Gate 仅作为背景，本报告结论只依赖 Reviewer 实际读取与上述独立重跑。

## 7. 未验证风险与限制

- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未连接真实 PostgreSQL、SMTP，未执行部署或真实 E2E。
- 未在真实环境模拟数据库 commit acknowledgment loss、多连接锁竞争、provider accepted-but-client-unknown。
- 三个固定对象处于共享未跟踪工作树；本报告仅对上列哈希对应全文负责。
- 共享工作树中的其他未提交/未跟踪内容不在本结论范围。
- F002 新 finding 涉及 mailer outcome 边界，公共 REST 合同不应随内部修复漂移；若拟改变公共契约，必须先走方向决策。

## 8. Gate 与交回

- 旧 `IDENTITY-S10.1-REV-R01-F002`：**CLOSED**
- 新 findings：**2 Major、1 Minor**
- **Code Review Gate：CHANGES REQUIRED**

本 Gate 的阻断原因是 `IDENTITY-S10.1-REV-R03-F001` 与 `IDENTITY-S10.1-REV-R03-F002`。是否接受风险及修复方向由 Lead/用户决定；本独立评审不代替采用决定或后续授权。
