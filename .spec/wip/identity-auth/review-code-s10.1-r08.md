# Identity S10.1 定点独立 Code Review R08

- 日期：2026-10-25
- objectType：code
- Reviewer：非作者 `add-reviewer`（续接原 Reviewer Euler）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：仅原 finding `IDENTITY-S10.1-REV-R07-F001` 的修复 revision、相关回归及该修复引入的新风险
- **Code Review Gate：PASS**
- 原 finding：**CLOSED**
- 新增 findings：**0**

## 1. 固定对象与漂移检查

开始、测试前及报告落盘前均逐一复核固定对象；哈希一致，分支和 HEAD 一致，staged 为空，报告路径此前不存在。

| 文件 | 固定且复核一致的 SHA256 |
|---|---|
| `internal/mailer/mailer.go` | `A2F103D6AF3664288D0C4F460EC5733E33FA6461FB8554A189FBC3194C5B4D0D` |
| `internal/mailer/smtp.go` | `0C64C229B7EE8101B0DCAD95C14431DCAF8ED55809C9FBFD50AEE75AF0221199` |
| `internal/mailer/smtp_test.go` | `656E12CB62C113C60AB074AE5374F9E206D226CCEE5B194BB1F0F8AA12B96D15` |
| `internal/identity/password_reset.go` | `5F2D0E1B534E1A4F47AF25675FB94AA3B2830A01C408F6C615E41FAB74A81575` |
| `internal/identity/password_reset_test.go` | `59AAC51742A61ADD318D8ADE5527231D4D5A198790687B101A35A5A843666E2A` |

五个对象在当前基线下均为共享工作树中的 untracked 文件，因此未用空 Git diff 代替固定哈希与全文审查。

## 2. 实际读取范围

完整读取：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/profiles/go-service/README.md`
- `.codex/profiles/go-service/rules/coding-standards.md`
- `.codex/profiles/go-service/rules/no-hardcode-policy.md`
- `.codex/profiles/go-service/rules/observability-standards.md`
- `.codex/profiles/go-service/rules/security-coding-standards.md`
- `.codex/profiles/go-service/rules/tracing-standards.md`
- `.spec/wip/identity-auth/review-code-s10.1-r07.md`
- 五个固定被审文件全文

有界读取：

- `.add/project/knowledge/design-workspace/identity-auth/design.md` 中 SMTP、S10.1 durable operation、明确未受理、unknown、普通 resend、旧能力保持及 S11 验证合同；
- `internal/identity/temporary_credential_resend.go` 与相关测试中的 S11.1 `password_reset` 路由、`ResendPasswordResetCredential` 和 unknown durable recovery chain；
- Go 1.27.0 本机标准库 `net/smtp/smtp.go` 中 `hello`、`Hello`、`StartTLS`、`Auth`、`Mail`、`Rcpt`、DATA final reply 的真实调用顺序。

未扩成全仓预审。

## 3. 两轮独立检查

### Round 1：原缺陷根因与修复 revision

检查 `SMTPMailer.Send`、`secureClient`、typed classifier 和协议级测试后确认：

1. implicit TLS 路径在 TLS handshake 和 `smtp.NewClient` 成功后，于返回 client、AUTH、MAIL 之前显式执行 `client.Hello("localhost")`。因此 `client.Mail` 不再可能替第一次 EHLO/HELO 协商；原 finding 的阶段来源混淆已消除。
2. STARTTLS 路径的 pre-TLS hello、STARTTLS 和 post-TLS EHLO 均由 `Extension` / `StartTLS` 在 `secureClient` 返回前完成；这些失败不会进入 MAIL/RCPT/DATA classifier。
3. `classifySMTPMessageError` 只在 MAIL、RCPT、DATA command 和 DATA final reply 调用点使用；只有 typed `*textproto.Error` 且 code 在 `400..599` 才附加 `ErrMessageRejected`。未使用错误字符串匹配。
4. Dial、TLS、EHLO/HELO、AUTH、context cancellation、断链、I/O 和异常 250/354 reply 均不匹配 rejection sentinel，继续保持 unknown；DATA 收到 250 后的 QUIT/关闭失败仍不反转既有 success。
5. 新测试通过真实 `net.Pipe` + TLS + SMTP command/reply 交互驱动 adapter，而非直接调用分类 helper：
   - EHLO 与 fallback HELO 都返回 550，断言未发送 MAIL 且不匹配 `ErrMessageRejected`；
   - MAIL/RCPT/DATA command/DATA final reply 的 250/354 异常响应均保留 typed protocol error，但不匹配 rejection sentinel；
   - 既有 MAIL/RCPT/DATA 450/451/550/554 仍稳定匹配 rejection sentinel，AUTH 535 仍为 unknown。

### Round 2：coordinator、持久化 authority 与直接消费者

有界追踪后确认：

1. `PasswordResetCoordinator.dispatchPasswordReset` 仍是唯一分类消费点：仅 `errors.Is(mailErr, mailer.ErrMessageRejected)` 进入 `RecordRejectedDelivery`，其他 Mailer error 全部进入 `RecordUnknownDelivery` 并返回既有 `identity.operation_uncertain`。
2. `RecordRejectedDelivery` 仍只终结本 operation、失效 exact pending credential并完成对应 job；写入或 commit 失败保持 operation uncertain。此次修复未改变该持久化 authority。
3. greeting 550 与 MAIL 354 的 coordinator 回归分别覆盖 initial reset / ordinary resend，断言 unknown evidence、无 rejected evidence、无新候选 invalidation。
4. S11.1 `password_reset` 路由直接调用同一 `ResendPasswordResetCredential`，最终复用同一 `dispatchPasswordReset`；既有真实 S11 resend unknown durable-chain 回归继续证明 unknown 保持 Gate、worker 不重发邮件，旧能力在保护边界前不变。明确 rejection 的 S11 回归也继续证明只有 sentinel 才走 rejected settlement。
5. `Mailer.Send(context.Context, Message) error` 签名、公共 REST/schema/error/migration/ADR、唯一 operation authority和 production composition 均未变化；不需要第六源码文件或新增授权。

## 4. Finding 复审

### `IDENTITY-S10.1-REV-R07-F001` — HIGH — CLOSED

原问题的两个必要条件均已消除：

- EHLO/HELO 不再隐式藏在可分类的 `client.Mail` 调用内；
- typed protocol error 只有在明确的 MAIL/RCPT/DATA 调用点且回复码为 4xx/5xx 时才成为 rejected。

协议级正反例、coordinator unknown 回归、S11 resend 共用链及定向 race/vet 均通过。未发现该修复引入新的 correctness、安全、并发、资源、authority 或兼容性缺陷。

## 5. 新增 findings

**0。**

## 6. 实际命令与结果

| 命令 | 结果 |
|---|---|
| `go test ./internal/mailer -run '^(TestSMTPMailerDoesNotClassifyGreetingRejectionAsMessageRejection\|TestSMTPMailerDoesNotClassifyUnexpectedNonNegativeReplyAsMessageRejection\|TestSMTPMailerClassifiesExplicitMessageRejection\|TestSMTPMailerDoesNotClassifyAuthenticationRejectionAsMessageRejection)$' -count=1` | PASS，`han/internal/mailer 1.372s` |
| `go test ./internal/identity -run '^(TestPasswordResetUnclassifiedSMTPProtocolErrorsRemainUnknown\|TestTemporaryCredentialResendPasswordResetUsesDurableRecoveryChain\|TestPasswordResetExplicitRejectionRecordsStableFailure\|TestPasswordResetExplicitRejectionFlowsThroughS11ResendConsumer)$' -count=1` | PASS，`han/internal/identity 0.080s` |
| 上述 mailer 定向测试加 `-race` | PASS，`2.869s` |
| 上述 identity 定向测试加 `-race` | PASS，`1.108s` |
| `go vet ./internal/mailer ./internal/identity` | PASS，无输出 |
| 五个固定文件 `gofmt -l` | PASS，无输出 |
| 五个固定文件 `git diff --check` | PASS，无输出；因对象 untracked，仅作补充检查 |

测试均使用本地 fake / `net.Pipe`，未连接真实 SMTP 或 PostgreSQL。

## 7. 未验证风险与证据限制

- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`；未运行任何依赖该变量的测试。
- 未连接真实 PostgreSQL、真实/外部 SMTP provider、生产服务、部署或 E2E。
- 未验证真实 provider 对非标准回复的现场行为、真实网络断链、PG commit acknowledgement loss、多进程竞态、migration 混跑、容量或 production readiness。
- 本 Gate 仅关闭 R07 F001 的固定 revision；不外推为全包、全仓、真实 PG/SMTP、部署或 E2E PASS。
- 共享工作树中的其他 modified/untracked 内容不在本结论范围；Reviewer 未回滚、覆盖、格式化、清理、stage、commit、push 或切分支。
- `han-server.exe` 保留且未修改。

## 8. Gate

**Code Review Gate：PASS**

`IDENTITY-S10.1-REV-R07-F001` 已关闭；新增 findings 为 0。后续仍须由 Lead 在同一固定 revision 现场复核五哈希、报告哈希、staged、共享工作树越界和本报告所列证据限制。
