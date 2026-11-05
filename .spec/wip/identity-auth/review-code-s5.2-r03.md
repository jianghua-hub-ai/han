# Identity S5.2 code review r03 定点复审

- **评审日期**：2026-10-23
- **Reviewer**：Faraday（独立非作者 Reviewer）
- **objectType**：code
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **原报告**：`.spec/wip/identity-auth/review-code-s5.2-r02.md`
- **结论**：**FAIL；`IDENTITY-S5.2-R02-F001` CLOSED，原 `IDENTITY-S5.2-R01-F002` 仍为 LIMITED；新增 1 项 HIGH finding**

本轮只复审 SMTP cancellation 修复及其直接生产消费者，不修改被审代码。结论绑定第 1 节固定对象；真实 SMTP/TLS、Host、部署、E2E 和 production readiness 不在本 Gate 的已验证范围。

## 1. 固定对象与依据

评审开始及落报告前均核对分支、HEAD、git blob 与 SHA-256，结果与委托值一致：

| 对象 | git blob | SHA-256 |
|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md`（r33） | `1c6ea6827a18158a9936a4612aac523b258e4e5d` | `47FC3598CE2BCA7A1E80032FEC040BCF63344C869B4EF12F9BE5998589563513` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md`（r13） | `c1b7ec4e9c201c2c24a5afe7300532eb7e06e4e3` | `F5A3E2CEF38C36E595D8217BDD85BBA7CF55FC0F1DBB311D773E7DDBD1076749` |
| `.add/project/knowledge/tasks/tasks.md` | `e91793e7ebc6ed7959d1c9fa5bb3f3425cfcda53` | `6D7558F833F19FED9C5A3CE921F71E0E2A9F56EF31B86C5BE0FF44498AA75886` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `3227e4812fcccc6597d89377a75b3426973b7a37` | `C9CB97FB48E5D92544635BB8B7C5C4BAB45E122149CF817F665D7B31983B7E3D` |
| `.spec/wip/identity-auth/review-code-s5.2-r02.md` | `f9e74c7c6a651b725e893eb21e66d1166539b087` | `D2078E25FADA76F266B73CD7B441EE08DC0A6E54B520242D7FE7319661B246C4` |
| `internal/mailer/smtp.go` | `62c984be0efc708f2ee3578f93231ecf6cbbbf11` | `EA42BC63AFE1E8577E4F5C7C01FDCBF30C5A00496D8ED53051A199234CF633D8` |
| `internal/mailer/smtp_test.go` | `812ec8e465829b86a0f0c2f2b0bdeb6523b148b1` | `A4B7784F224800F6476C585526E7ACAB1D87E5882427799155BA2EE35B6D5364` |

修复前基线为 `smtp.go` blob `335112...` / SHA `3B6624...`，`smtp_test.go` blob `b4c5ab...` / SHA `08FC9D...`。本轮报告创建前路径不存在。

## 2. 两轮实际评审范围

### Round 1：固定修复对象

检查：

1. `SMTPMailer.Send` 中 `context.AfterFunc`、活动 `net.Conn.Close`、stopper、`defer client.Close`、`Quit` 的并发和资源所有权。
2. AUTH、MAIL、RCPT、DATA command、DATA acceptance 阻塞时取消能否及时解除同步 I/O。
3. DATA 已获 acceptance 后，QUIT/close 因取消失败仍返回成功，避免把已受理邮件误报为可安全重试。
4. 正常完成后 watcher 已停止，不会在调用返回后再次触碰连接。
5. 回归测试是否以同步阻塞点触发旧实现缺陷，并结合 `-race` 检查竞态。

### Round 2：直接消费者与生产接缝

沿以下有界链路核对：

`AdministratorCreateCoordinator → production SMTP composition → SMTPMailer → pgx transaction/idempotency failure convergence`

检查 coordinator 的较早 deadline、及时 cancel、邮件失败/已受理后的失败语义、真实 pgx rollback 行为、稳定失败/uncertain 持久化、production 注入及公共 API/错误语义。未扩展为全仓审计。

## 3. 原 finding 状态

### `IDENTITY-S5.2-R02-F001` — CLOSED

关闭证据：

- `internal/mailer/smtp.go:107-108,148-159` 使用 `context.AfterFunc` 将 parent cancellation 绑定到活动连接；callback 关闭连接，解除 `net.Conn` 上阻塞的同步读写。
- stopper 只有两条互斥路径：成功阻止 callback 时由 stopper 关闭 `watcherDone`；callback 已开始时由 callback 关闭，stopper 等待完成。未发现 double-close、panic 或等待自身造成的死锁路径。
- `smtp.go:142-145` 在 DATA acceptance 后忽略 QUIT 关闭错误并返回 nil，保留“已受理不可作为安全重试失败”的既有语义。
- `smtp_test.go:24-103` 确定性覆盖 AUTH、MAIL、RCPT、DATA command、DATA acceptance 阻塞取消；`105-140` 覆盖 acceptance 后 QUIT 阻塞取消仍成功；`142-190` 覆盖正常完成后 watcher 停止。
- targeted、全 mailer、四包普通测试、四包 race 与 vet 均 PASS。未发现公共 API、TLS/auth 顺序或资源 ownership 被越权改变。

### `IDENTITY-S5.2-R01-F002` — LIMITED（不能转 CLOSED）

SMTP adapter 的“已连接后及时响应取消”缺口已关闭；必填预算、较早 deadline、production 注入和及时调用 cancel 也保持实现。但 Round 2 发现 parent cancellation 后 coordinator 使用已取消的请求 context 执行事务回滚及失败/uncertain 收敛，不能可靠满足设计与 I-S52-07 的稳定结果合同，见 `IDENTITY-S5.2-R03-F001`。

## 4. 新 findings

### CRITICAL

无。

### HIGH

#### `IDENTITY-S5.2-R03-F001` — parent cancellation 后复用已取消 context，稳定失败或 uncertain 状态无法可靠落库

- **位置**
  - `internal/identity/administrator_create.go:455-478`
  - `internal/identity/administrator_create.go:481-535`
  - `internal/identity/administrator_create.go:1049-1128`
  - `internal/identity/administrator_create.go:1201-1246`
  - 测试缺口：`internal/identity/administrator_create_test.go:347-374,733-750`
- **违反依据**
  - design r33 `§3.4.2`：请求已取消、请求 deadline 或预算到期均应映射稳定 `identity.email_delivery_failed`，回滚 pending 数据并保存稳定失败快照。
  - test-plan r13 `I-S52-07`：parent 已取消场景应形成稳定 email-delivery-failed 快照、回滚 pending 状态并保持零可用账号/成功审计/额外邮件。
  - DATA 已受理后的后续 DB 失败应进入既有 uncertain 收敛语义，而非因同一已取消 context 无法执行状态记录。
- **触发场景**
  1. 请求 context 在 SMTP I/O 中被取消；新 adapter 关闭连接并及时返回错误，或在 DATA 已获 acceptance 后返回 nil。
  2. 未受理分支在 `:468` 用已经取消的 `ctx` 执行 `tx.Rollback(ctx)`，随后在 `:473-478` 把同一 `ctx` 传给 `persistKnownCreateFailure`。
  3. `persistKnownCreateFailure` 在 `:1055` 以该 context 开始新事务，并在后续 Begin/Complete/Audit/Commit 全程复用它；真实 pgx 会立即受 cancellation 影响，无法可靠写入稳定 failed snapshot。
  4. 已受理分支同样在 Activate/Audit/Complete/Commit 失败后，把已取消 `ctx` 传入 `markCreateUncertain`；uncertain 记录也无法可靠收敛。
  5. pgx v5.11 的 `dbTx.Rollback(ctx)` 通过 `Exec(ctx, "rollback")` 执行；rollback 失败会关闭状态未定义的连接。连接释放可最终清除事务，但不等于稳定失败或 uncertain 记录已持久化。
- **测试为何未发现**
  - “already canceled request” 使用的 `administratorSequenceTransactor.Begin(context.Context)`、fake repository 和 fake transaction 均忽略 context；测试只断言无可用管理员，没有断言稳定 failed snapshot/audit。
  - 该 fake 在真实 pgx 会拒绝的已取消 context 下仍允许第二事务和 Complete 成功，掩盖了 production consumer 路径。
- **影响**
  - SMTP 未受理时，调用方可能收到内部持久化错误而非稳定 `identity.email_delivery_failed`，同键重放也无法获得设计要求的稳定失败快照。
  - DATA 已受理后发生 parent cancellation 时，邮件可能已经送达，但系统无法可靠记录 uncertain；后续重试可能再次发送凭据，失去幂等恢复证据。
  - 原 F002 的端到端 cancellation/零副作用/稳定状态合同仍未闭合，因此阻断 Code Review Gate。
- **最小修复方向**
  - 为 rollback 后的稳定失败或 uncertain 收敛提供受控、独立且有界的 cleanup/reconciliation context；不得无界运行、后台发送或改变已确认的公共错误与幂等状态机。
  - 增加 context-aware transactor/transaction fake，确保已取消 context 会像真实 pgx 一样失败；分别覆盖 SMTP 未受理与 DATA 已受理后 parent cancellation，并断言 rollback、稳定 failed/uncertain、审计及重放结果。
  - 若实施证明必须改变已确认事务/幂等时序，应停止并回到设计，而不是在代码中自行选择新契约。

### MEDIUM / LOW

无。

## 5. 待决定与非阻断建议

### 待决定

无。本 finding 要求落实既有设计和测试合同，不要求 Reviewer 选择新的公共行为。

### 非阻断建议

除修复 finding 所需的 context-aware 回归外，无独立建议。

## 6. Reviewer 独立验证命令

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current` / `git rev-parse HEAD` | PASS：`main` / 固定 HEAD |
| `git hash-object -- <fixed object>` + `Get-FileHash -Algorithm SHA256` | PASS：第 1 节七个对象开始及落报告前均匹配 |
| `go test ./internal/mailer -run 'TestSMTPMailer(CancellationInterruptsBlockedSMTPIO\|CancellationAfterDataAcceptanceKeepsSuccess\|NormalCompletionStopsCancellationWatcher)$' -count=1` | PASS |
| `go test ./internal/mailer -count=1` | PASS |
| `go test -race ./internal/mailer -count=1` | PASS |
| `go test ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1` | PASS |
| `go test -race ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1` | PASS |
| `go vet ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server` | PASS |
| `gofmt -l internal/mailer/smtp.go internal/mailer/smtp_test.go` | PASS：无输出 |
| 读取 pgx v5.11 `dbTx.Rollback` / `pgxpool.Tx.Rollback` 直接实现 | 完成；确认 canceled context 会作用于 rollback，失败连接被关闭并释放 |
| `git -c core.safecrlf=false diff --no-index --check -- NUL <smtp.go/smtp_test.go/new report>` | 三个对象均无 whitespace error 输出；退出码 1 仅表示各文件与空文件存在内容差异 |

Lead 提供的 PostgreSQL 18 integration 与 Manifest 7/7 PASS 仅作为外部已有证据记录；Reviewer 未读取、打印或持久化 User scope `HAN_TEST_POSTGRES_DSN`，也未冒充独立复跑。

## 7. 未验证风险与限制

1. 未连接真实 SMTP/TLS provider，未执行真实 DNS/TCP/TLS/AUTH 或邮件受理；本轮只验证本地 adapter、fake transport 和直接消费者代码。
2. 未独立复跑 PostgreSQL 18 integration；新 finding 依据现役 coordinator、pgx v5.11 直接实现和合同分析，尚无本轮真实 PG18 动态复现。
3. 未启动真实 Host、部署环境或 E2E；结论不代表容量、部署或 production readiness。
4. 未复跑 Lead 的全仓 Manifest 七项，只执行第 6 节有界命令。
5. 共享工作树的其他未提交成果均保留；结论不覆盖固定对象及直接接缝之外的文件。

## 8. Code Review Gate 与复审条件

### Code Review Gate: FAIL

`IDENTITY-S5.2-R02-F001` 已关闭，但 `IDENTITY-S5.2-R01-F002` 仍为 LIMITED；新增 HIGH `IDENTITY-S5.2-R03-F001` 保持 OPEN。

复审条件：

1. 作者提供固定修订，证明 parent cancellation 后业务事务能有界回滚，并能可靠持久化稳定 failed 或已受理后的 uncertain 状态。
2. context-aware 测试不得忽略 canceled context；应覆盖未受理、DATA 已受理、稳定重放及资源收敛。
3. 重跑受影响 identity/idempotency/mailer/cmd 普通、race、vet 及适用项目 Gate；PG18 集成须继续使用不泄露 DSN 的安全方式。
4. 由独立非作者 Reviewer 定点复审 `IDENTITY-S5.2-R03-F001`，并再次判断原 F002 是否可 CLOSED。

本结论不代替 Lead/用户的风险接受、修复授权或后续交付决定。

## 9. 实际读取范围

### Role / Skill / Rule / Profile

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- add-code-review 引用的产品安全红线
- 已启用 `go-service` profile 的 coding、no-hardcode 与 security 规则

### 上游依据、固定对象与直接接缝

- 第 1 节五份上游依据
- `internal/mailer/smtp.go`
- `internal/mailer/smtp_test.go`
- `internal/identity/administrator_create.go`
- `internal/identity/administrator_create_test.go`
- `cmd/han-server/main.go` 中 production SMTP/AdministratorCreate composition
- pgx v5.11 `tx.go`、`pgxpool/tx.go`
- Go 1.27 `context.AfterFunc`、`net.Conn.Close`、`net/smtp.Client.Quit` 直接文档/实现
