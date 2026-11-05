# Identity S5.2 code review r04 定点复审

- **评审日期**：2026-10-23
- **Reviewer**：Faraday（独立非作者 Reviewer）
- **objectType**：code
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **原报告**：`.spec/wip/identity-auth/review-code-s5.2-r03.md`
- **结论**：**PASS；`IDENTITY-S5.2-R03-F001` CLOSED，`IDENTITY-S5.2-R01-F002` CLOSED；无新增 finding**

本轮只复审 r03 的唯一 HIGH finding、再次判定原 F002，并沿直接消费者与 production composition 做有界回归核对；不修改被审代码。结论绑定第 1 节固定对象，不代表真实 SMTP/TLS、Host、部署、E2E 或 production readiness 已通过。

## 1. 固定对象与依据

评审开始及落报告前均核对分支、HEAD、git blob 和 SHA-256，结果与委托值一致：

| 对象 | git blob | SHA-256 |
|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md`（r33） | `1c6ea6827a18158a9936a4612aac523b258e4e5d` | `47FC3598CE2BCA7A1E80032FEC040BCF63344C869B4EF12F9BE5998589563513` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md`（r13） | `c1b7ec4e9c201c2c24a5afe7300532eb7e06e4e3` | `F5A3E2CEF38C36E595D8217BDD85BBA7CF55FC0F1DBB311D773E7DDBD1076749` |
| `.add/project/knowledge/tasks/tasks.md` | `e91793e7ebc6ed7959d1c9fa5bb3f3425cfcda53` | `6D7558F833F19FED9C5A3CE921F71E0E2A9F56EF31B86C5BE0FF44498AA75886` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `3227e4812fcccc6597d89377a75b3426973b7a37` | `C9CB97FB48E5D92544635BB8B7C5C4BAB45E122149CF817F665D7B31983B7E3D` |
| `.spec/wip/identity-auth/review-code-s5.2-r02.md` | `f9e74c7c6a651b725e893eb21e66d1166539b087` | `D2078E25FADA76F266B73CD7B441EE08DC0A6E54B520242D7FE7319661B246C4` |
| `.spec/wip/identity-auth/review-code-s5.2-r03.md` | `29410c9305d58a05e4f708f33b5c141c2f93aad4` | `1066A6381B97E621081C478120ED7CA9D7296B701A813F31373A51C0770C5DB6` |
| `internal/identity/administrator_create.go` | `47fa2b8894dd588921fc9b5eacd49defe6de8221` | `DD1D1DA23CB9D9E13E3200E09867B2A2C2F6BCBF06894C46074F60AC94D75813` |
| `internal/identity/administrator_create_test.go` | `1ed2dc4abb6c583fe7bacc6ef66531586d2dd07b` | `26E059CD710160FFE7E6F6DCE793859DA478689D0038641EFDC2E5D335463B34` |
| `internal/mailer/smtp.go` | `62c984be0efc708f2ee3578f93231ecf6cbbbf11` | `EA42BC63AFE1E8577E4F5C7C01FDCBF30C5A00496D8ED53051A199234CF633D8` |
| `internal/mailer/smtp_test.go` | `812ec8e465829b86a0f0c2f2b0bdeb6523b148b1` | `A4B7784F224800F6476C585526E7ACAB1D87E5882427799155BA2EE35B6D5364` |

新报告路径在创建前不存在。

## 2. 两轮实际评审范围

### Round 1：固定修复对象

检查：

1. `administratorReconciliationContext` 是否保留 request values、隔离 parent cancellation/deadline、重新施加有限预算并在同步返回时 cancel。
2. SMTP 未受理，以及已受理后 Activate、Audit、Idempotency Complete、Commit 各失败分支，是否均使用 cleanup context 执行 rollback 与稳定 failed/uncertain 收敛。
3. commit outcome unknown 是否保持“不错误 rollback、转 uncertain”的既有语义；cleanup 是否会继续业务或再次发送邮件。
4. context-aware fake 是否会拒绝 canceled context；回归是否覆盖未受理稳定失败/重放、已受理 uncertain/重试不发信、审计和资源收敛，而非仅迎合实现。
5. 固定 Mailer 的取消 watcher、活动连接关闭、DATA acceptance 后成功语义及 race 回归。

### Round 2：直接消费者与 production 接缝

有界核对：

`AdministratorCreateCoordinator → production AdministratorCreateDependencies → SMTPMailer → pgx transaction / idempotency reconciliation`

检查 SMTP budget 的生产注入、Mailer 共享实例、公共 API/错误、幂等状态机及 production wiring 未被本次修复越权改变。未扩展为全仓审计。

## 3. 原 finding 状态

### `IDENTITY-S5.2-R03-F001` — CLOSED

关闭证据：

- `internal/identity/administrator_create.go:466-480`：SMTP 未受理后创建 cleanup context，以该 context 回滚业务事务并同步持久化稳定 `identity.email_delivery_failed`。
- `internal/identity/administrator_create.go:483-545`：邮件已受理后，Activate、Audit、Complete 或 Commit 失败均以 cleanup context 收敛为 uncertain；Commit 失败不执行可能错误的二次 rollback，保留 commit outcome unknown 语义。
- `internal/identity/administrator_create.go:551-557`：cleanup context 使用 `context.WithoutCancel(ctx)` 保留 values、隔离已触发的 parent cancellation/deadline，再以 `SMTPOperationBudget` 设置新上界。所有调用均同步执行并 `defer cancelCleanup()`，未新增后台任务或邮件发送路径。
- `internal/identity/administrator_create_test.go:425-523`：Mailer 在 Send 内取消 parent 并返回未受理；context-aware transaction/store/audit 会拒绝 canceled context；测试证明有限 deadline、value 保留、返回后 cancel、rollback、stable failed snapshot、failed audit、同键重放且 Mailer 总调用一次。
- `internal/identity/administrator_create_test.go:525-616`：Mailer 模拟 DATA 已受理后取消 parent；原 request context 下 Activate 失败，cleanup context 完成 rollback 与 uncertain 持久化；重试返回 uncertain 且 Mailer 总调用一次。
- 定向测试、identity 全包、identity race、四包普通/race/vet 均 PASS。

未发现 cleanup context 导致继续业务、再次发信、无界后台执行、公共错误变化或新配置需求。

### `IDENTITY-S5.2-R01-F002` — CLOSED

r03 已关闭 SMTP adapter 的活动 I/O 取消缺口，并确认必填 budget、较早 deadline、及时 cancel 与 production 注入；本轮进一步关闭其最后限制：

- parent cancellation 后不再复用 canceled request context 做 rollback 或 failed/uncertain 持久化；
- SMTP 未受理可形成稳定失败、零可用管理员并稳定重放；
- DATA 已受理后的 DB 失败可形成 uncertain，后续同键重试不重发邮件；
- cleanup 有 `SMTPOperationBudget` 上界、同步完成并及时 cancel。

因此 F002 在本次固定对象和本地 deterministic 接缝范围内闭合。

### 其他历史项

- `IDENTITY-S5.2-R02-F001`：保持 **CLOSED**；本轮 Mailer targeted/race 回归通过，无直接回归证据。
- `IDENTITY-S5.2-R01-F001`、`IDENTITY-S5.2-R01-F003`、`IDENTITY-S5.2-R01-D001`：保持既有 **CLOSED**；本轮未发现 strict legacy/v1、D001、production composition/route 合同的直接回归证据。

## 4. 新 findings

### CRITICAL / HIGH / MEDIUM / LOW

无。

## 5. 待决定与非阻断建议

### 待决定

无。本次修复未要求新增配置、公共错误、状态、schema、migration 或设计决策。

### 非阻断建议

无独立建议。

## 6. Reviewer 独立验证命令

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current` / `git rev-parse HEAD` | PASS：`main` / 固定 HEAD |
| `git hash-object -- <fixed object>` + `Get-FileHash -Algorithm SHA256` | PASS：第 1 节全部对象开始及落报告前均匹配 |
| `go test ./internal/identity -run 'TestCreateAdministrator(ReconcilesSMTPFailureAfterParentCancellation\|MarksUncertainAfterAcceptedMailParentCancellation)$' -count=1` | PASS |
| `go test ./internal/identity -count=1` | PASS |
| `go test -race ./internal/identity -count=1` | PASS |
| `go test ./internal/mailer -run 'TestSMTPMailer(CancellationInterruptsBlockedSMTPIO\|CancellationAfterDataAcceptanceKeepsSuccess\|NormalCompletionStopsCancellationWatcher)$' -count=1` | PASS |
| `go test -race ./internal/mailer -count=1` | PASS |
| `go test ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1` | PASS |
| `go test -race ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1` | PASS |
| `go vet ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server` | PASS |
| `gofmt -l internal/identity/administrator_create.go internal/identity/administrator_create_test.go internal/mailer/smtp.go internal/mailer/smtp_test.go` | PASS：无输出 |
| `git -c core.safecrlf=false diff --no-index --check -- NUL <四个固定 Go 文件及新报告>` | PASS：无 whitespace error；退出码 1 仅表示文件与空文件存在内容差异 |

Lead 提供的 PostgreSQL 18 integration 与 Manifest 7/7 PASS 仅作为外部证据记录。Reviewer 未读取、打印或持久化 User scope `HAN_TEST_POSTGRES_DSN`，也未冒充独立复跑 PG18。

## 7. 未验证风险与限制

1. 未连接真实 SMTP/TLS provider，未执行真实 DNS/TCP/TLS/AUTH 或外部邮件受理；本轮只验证本地 adapter、可控 SMTP server/fake 与直接消费者。
2. 未独立复跑 PostgreSQL 18 integration；数据库 CHECK/真实事务集成继续以先前独立评审与 Lead 外部证据为限。
3. 未启动真实 Host，未执行部署或 E2E；本结论不证明容量、部署、外部 provider 或 production readiness。
4. 未复跑 Lead 的全仓 Manifest 七项，只执行第 6 节有界命令。
5. 共享工作树中的其他未提交成果保持原状；本结论不覆盖固定对象和直接接缝之外的文件。

## 8. Code Review Gate

### Code Review Gate: PASS

`IDENTITY-S5.2-R03-F001` 已关闭，原 `IDENTITY-S5.2-R01-F002` 可转为 CLOSED；无新增阻断 finding。

该 PASS 仅表示固定 S5.2 代码对象在本轮两轮有界评审中满足已确认合同，可交回 Lead 继续后续责任确认；不替代真实 SMTP/TLS、Host、部署、E2E、production readiness 或人的最终接受决定。若任一固定对象漂移，须以新修订重新独立评审。

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

- 第 1 节六份上游依据/历史报告
- `internal/identity/administrator_create.go`
- `internal/identity/administrator_create_test.go`
- `internal/mailer/smtp.go`
- `internal/mailer/smtp_test.go`
- `cmd/han-server/main.go` 中 production SMTP 与 AdministratorCreate composition
- 与上述路径直接相关的 idempotency、pgx transaction、audit 和 platform error 定义
