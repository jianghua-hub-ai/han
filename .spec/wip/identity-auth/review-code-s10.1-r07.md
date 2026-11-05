# identity-auth S10.1 独立 Code Review R07

- 日期：2026-10-25
- objectType：code
- Reviewer：非作者 `add-reviewer`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **Code Review Gate：CHANGES REQUIRED**
- 新增 finding：1 项（HIGH）

## 1. 固定对象与漂移检查

五个被审对象均按全文和固定 SHA256 审查；它们在共享工作树中为 untracked，未用空 Git diff 代替对象核验。

| 文件 | 固定且复核一致的 SHA256 |
|---|---|
| `internal/mailer/mailer.go` | `A2F103D6AF3664288D0C4F460EC5733E33FA6461FB8554A189FBC3194C5B4D0D` |
| `internal/mailer/smtp.go` | `4AE1CF55EC4197E9BD6660585E69B044B5251F87408FA401172B030ECAE0C2F0` |
| `internal/mailer/smtp_test.go` | `62EC6D231A7137E4160745FEC4D10C6856D4F43AA7B5CD9E0C0BE8211BC4DF88` |
| `internal/identity/password_reset.go` | `5F2D0E1B534E1A4F47AF25675FB94AA3B2830A01C408F6C615E41FAB74A81575` |
| `internal/identity/password_reset_test.go` | `68BC37A6EBB63A62B458D461C8C4AC89082EE7A4674EB10016B6349A59405210` |

开始、测试后和报告落盘前均确认分支、HEAD、五个哈希匹配，报告路径此前不存在，staged 为空。未发现对象漂移或未知写者对五个对象的中途改写。

## 2. 两轮实际检查范围

### Round 1：五文件直接正确性与测试

- 完整检查 `Mailer.Send(context.Context, Message) error` 签名、稳定 sentinel、`errors.Is` / `errors.As` 分类和禁止字符串匹配。
- 沿 SMTP 阶段检查 Dial、TLS、EHLO/HELO、AUTH、MAIL、RCPT、DATA command、DATA body、DATA final reply、QUIT、取消和断链。
- 检查 accepted / rejected / unknown 到 coordinator 的分流、独立 evidence transaction、begin/write/commit 不确定性和公共错误保持。
- 检查首次 reset、普通 resend、S11.1 resend 的新测试，以及 old credential、auth generation、Session、Gate 和 exact new candidate 的断言。
- 对照 Go 本机标准库 `net/smtp` 和 `net/textproto` 的真实错误来源，未仅依据测试替身推断协议阶段。

### Round 2：真实消费者、Store 与持久权威

有界追踪：

- `PasswordResetCoordinator.ResetPassword` 与 `ResendPasswordResetCredential`；
- `TemporaryCredentialResendRouter` 的 S11.1 `password_reset` 分支；
- `PasswordResetStore` 的 candidate classification、operation authority、same-key replay 和 `RecordRejectedDelivery`；
- `PasswordResetOperation` / credential / job 状态；
- 唯一 production coordinator 的构造与 S11.1 共享注入。

确认除下述 finding 外：

- rejected evidence 成功后 operation=`failed`、delivery=`rejected`、settlement=`invalidated`，只把本次 pending credential 置 `recovery_invalidated`，job 终结；
- evidence begin/write/commit 失败仍返回 `identity.operation_uncertain`；
- failed operation 可由现有 operation authority 稳定 same-key replay，不要求另造第二 authority；
- 首次 reset、普通 resend 和 S11.1 consumer 共用同一 coordinator；
- public REST、schema、migration、错误码、Mailer 接口签名和 production composition 未变化。

## 3. Findings

### IDENTITY-S10.1-REV-R07-F001 — HIGH — `client.Mail` 的隐式 EHLO/HELO 错误会被误判为 MAIL negative reply

**位置**

- `internal/mailer/smtp.go:126-127`
- `internal/mailer/smtp.go:149-153`
- 缺失回归：`internal/mailer/smtp_test.go:223-302`

**问题**

`SMTPMailer.Send` 把 `client.Mail(...)` 返回的任意 `*textproto.Error` 交给 `classifySMTPMessageError`，后者无条件附加 `ErrMessageRejected`。但 Go `net/smtp.Client.Mail` 在真正发送 `MAIL FROM` 前会先调用内部 `hello()`；在 implicit TLS 且未配置 AUTH 的路径上，本实现此前没有执行 EHLO/HELO。因此，服务器对 EHLO/HELO 的结构化失败也会从 `client.Mail` 返回为 `*textproto.Error`，并被错误标记为“MAIL/RCPT/DATA 明确未受理”。

同一分类器也没有限制回复码必须为 SMTP negative reply（4xx/5xx）；任何由 `textproto.ReadResponse` 因“与期望码不匹配”产生的 `*textproto.Error` 都会被当成 rejected，包括异常的非 4xx/5xx 回复。

**影响**

这违反“仅结构化 SMTP MAIL/RCPT/DATA negative reply 为明确未受理；其他阶段/不可判定结果保持 unknown”的固定合同。该错误会沿 `password_reset.go:336-344` 进入 `RecordRejectedDelivery`，把 operation/job 终结、失效新候选并清除 Gate，且 same-key 稳定重放 `identity.email_delivery_failed`；本应保持 unknown / unresolved 的协议协商错误因此被永久降格为 rejected。

现有 SMTP 测试只覆盖 MAIL/RCPT/DATA 的 450/451/550/554、AUTH 535、TLS 和 cancellation；测试服务器始终接受 EHLO/HELO，也没有非 4xx/5xx 的 unexpected reply 反例，所以当前全部测试通过不能捕获该阶段来源混淆。

**要求**

1. 在五文件边界内把 EHLO/HELO 协商与 MAIL 分类明确分离，确保只有可归因于 MAIL、RCPT 或 DATA 的回复才可能带 `ErrMessageRejected`。
2. typed classification 还须验证结构化回复确为 4xx/5xx negative reply；不得用错误字符串判断。
3. 增加 implicit TLS、AUTH=nil 下 EHLO/HELO 拒绝的回归，证明不带 `ErrMessageRejected`，coordinator 走 unknown、不会调用 rejected evidence。
4. 增加 unexpected 非 4xx/5xx reply 反例，证明不被终结为 rejected；保留现有 MAIL/RCPT/DATA 4xx/5xx、AUTH/TLS/cancel/I/O 和 DATA accepted 后成功回归。

该修复不需要第六源码文件、公共合同、schema、migration、ADR 或设计变化。

## 4. 待决定项

无新的产品或设计待决定项。现役合同已足够明确；本 finding 是五文件内的实现与测试缺陷。

## 5. 实际命令与结果

| 命令 | 结果 |
|---|---|
| SMTP accepted/rejected/AUTH/TLS/cancellation 定向测试 | PASS，`han/internal/mailer 2.266s` |
| `go test ./internal/identity -run '^TestPasswordReset.*(Accepted\|Rejected\|Unknown\|Delivery\|ProtectionWindow\|Replay)' -count=1` | PASS，`0.086s` |
| 两包相关专项 `go test -race` | PASS，mailer `2.988s`；identity `1.151s` |
| `go test ./internal/mailer ./internal/identity -count=1` | PASS，mailer `2.641s`；identity `0.203s` |
| `go vet ./internal/mailer ./internal/identity` | PASS |
| 五文件 `gofmt -l` | PASS，无输出 |
| 五文件 `git diff --check` | PASS，无输出；对象为 untracked，故只作补充检查 |
| 本机 Go 标准库只读核查 | `net/smtp.Client.Mail` 在 MAIL 前调用 `hello()`；`textproto.ReadResponse` 对任意期望码不匹配可返回 `*textproto.Error` |

未运行真实 SMTP、PostgreSQL、部署或 E2E。

## 6. 未验证风险与限制

- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未连接真实 PostgreSQL 或真实/外部 SMTP provider；未验证真实 PG 锁、commit acknowledgement loss、多连接竞态、migration 混跑、provider receipt、Host、部署、E2E、容量或 production readiness。
- 本轮只执行相关专项 race，未重复 Lead 已执行的全仓 Manifest 或全包 race。
- 共享工作树存在大量其他 modified/untracked 内容；它们不在本结论范围，Reviewer 未回滚、覆盖、格式化、清理、stage、commit、push 或切分支。
- `han-server.exe` 保留且未修改。

## 7. Gate 与复审条件

**Code Review Gate：CHANGES REQUIRED**

复审至少要求：

1. `IDENTITY-S10.1-REV-R07-F001` 按同一五文件边界修复；
2. 新增 EHLO/HELO 阶段来源和非 4xx/5xx 反例，且首次 reset、普通 resend、S11.1 consumer 均证明 unknown 不会进入 rejected evidence；
3. 现有 accepted/rejected/unknown、same-key replay、old credential/auth generation/Session/Gate、evidence failure、Mailer=1 与 replay/worker Mailer=0 回归继续通过；
4. 非作者 Reviewer 与 Lead 在同一固定 revision 复核五文件 SHA、status、staged、scoped diff、定向普通/race、vet、格式和越界；
5. 不新增第六源码文件，不改变公共 REST/schema/error/migration/ADR、Mailer 签名或唯一 authority/composition root。

在该 finding 关闭前，不应再次关闭 S10.1，也不应据此恢复 S5.3。
