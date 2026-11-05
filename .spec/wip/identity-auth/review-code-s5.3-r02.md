# identity-auth-S5.3 独立代码复审 R02

- 复审日期：2026-10-25
- objectType：`code`
- Gate：**PASS**
- 复审者：R01 原非作者 `add-reviewer`
- 作者/实施者：Kant（Agent `01a1716c-b694-71c3-8d72-2a4fc1ab4eb3`）
- 固定工作区：`D:\AID\VSCode\Codex\Han`
- 固定分支/HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`

## 1. R01 与固定复审对象

R01 固定报告：

| 文件 | SHA256 |
|---|---|
| `.spec/wip/identity-auth/review-code-s5.3-r01.md` | `78835EC544357CFB6BD970DF5F59EC5E20C9FD3CFCFA00AC3495B82EE0E7049B` |

R02 固定代码 revision：

| 文件 | SHA256 |
|---|---|
| `test/integration/identity/smtp_credentials_test.go` | `9CB60DE64DB1B00B3AA92EB32788CA042A3390FE583D60709FA1F295CA4C8826` |
| `test/integration/identity/smtp_test_server.go` | `FEFF245A580D7D848A780EB63C5AD1D0E2B13582ADDCC3BE6C9506C9F82DFB26` |
| `test/integration/identity/testdata/certificates/README.md` | `B9CE292F5A8701756646170C128825E6D955BF302D422E928B6664FE4847F73F` |

落报告前已再次核对 R01 与三个对象的 SHA256，均与委托固定值一致；R02 报告路径此前不存在。三对象与 R01 均未被本次复审修改。

## 2. 独立性、依据与范围

本次由 R01 原 Reviewer 复审原四个 finding，不替作者修复，不采用作者或 Lead 的 PASS 代替检查。

继续适用 R01 已读取的：

- `add-reviewer` Role 与 `add-code-review` 方法；
- code-review checklist、abstraction、coding、no-hardcode；
- 已启用 `go-service` profile；
- task-execution-workflow 的评审、授权、关闭合同；
- S5.3 tasks、identity-auth design r44、test-plan r20；
- `internal/mailer`、PasswordResetCoordinator/Store、AdministratorCreateCoordinator/resend 的有界真实调用链。

两轮实际范围：

1. **Finding 定向复审**：逐项对照 R01 F001-F004 的证据、影响和修复条件，检查新 revision 的实现与回归；
2. **同范围独立扫描**：重新检查 SMTP 状态机、TLS 错误分类、连接生命周期、PasswordReset 三分流及凭据残余语义，确认修复未引入新的 HIGH/MEDIUM finding。

未扩为全库泛审，未修改 production/public contract。

## 3. 独立执行命令

Reviewer 独立执行并 PASS：

```text
go test -tags=integration ./test/integration/identity -run 'TestCredentialSMTP|TestSMTPResidualWindow' -count=10
ok han/test/integration/identity 28.403s

go test -race -tags=integration ./test/integration/identity -run 'TestCredentialSMTP|TestSMTPResidualWindow' -count=1
ok han/test/integration/identity 5.438s

go vet -tags=integration ./test/integration/identity
PASS

gofmt -l test/integration/identity/smtp_credentials_test.go test/integration/identity/smtp_test_server.go
无输出

rg -n "\\.Error\\(\\)|bad certificate|Contains\\([^\\r\\n]*(err|Err|error)|Contains\\([^\\r\\n]*Error" \
  test/integration/identity/smtp_credentials_test.go \
  test/integration/identity/smtp_test_server.go \
  test/integration/identity/testdata/certificates/README.md
无匹配
```

Lead 提供的 scoped diff-check 与 Manifest 七项 PASS 作为输入证据；Reviewer 未重复执行全量 Manifest。

## 4. 原 Finding 复审

### IDENTITY-S5.3-REV-R01-F001 — CLOSED

R01 问题：PasswordReset 三分流未闭环断言凭据状态，resend 仅覆盖 accepted。

修复证据：

- `smtp_credentials_test.go:447-537` 现在同时覆盖：
  - 首次 reset accepted/rejected/unknown；
  - resend accepted/rejected/unknown。
- `:555-590` 对每个场景共同断言：
  - delivery、settlement、operation；
  - 新凭据状态和 request kind；
  - auth generation；
  - accepted/rejected/unknown evidence；
  - 旧 authority 仍可用；
  - session revocation 为 0；
  - dispatch claim/mail attempt 均恰为 1。
- `:569-579` 要求新凭据 ID 非空；明确拒绝必须恰好发生一次 invalidation，且目标必须是本次新凭据；非 rejected 场景禁止 invalidation。
- `:581-587` 对 resend 进一步断言旧临时凭据：
  - ID 非空；
  - 与新凭据 ID 不同；
  - 状态仍为 active。
- `:1245-1319` 的事务 fixture 分别记录候选凭据 ID、凭据状态、invalidation 次数与目标；若生产代码删除拒绝后的 invalidation，`state.credential`、计数和目标 ID 断言会失败，已具备 mutation-sensitive 性。
- `:602-620` 保留首次 accepted replay 不重复 dispatch/mail，并检查凭据、auth generation、旧 authority、session revocation 和 invalidation evidence 不漂移。
- 独立 `count=10` 与 race 均通过。

结论：首次与 resend 的 accepted/rejected/unknown 已对新凭据、旧 authority、auth generation、session revocation、dispatch/mail attempt 和拒绝 mutation 敏感性形成闭环。**CLOSED**。

### IDENTITY-S5.3-REV-R01-F002 — CLOSED

R01 问题：SMTP fixture 未强制命令顺序或 AUTH 前置条件。

修复证据：

- `smtp_test_server.go:362-404` 新增每连接 `smtpTestSessionState`，记录 TLS、greeting、AUTH、MAIL、RCPT、DATA 状态；
- `:371-404` 对以下越序返回协议违规：
  - greeting 前 STARTTLS；
  - TLS/greeting 前 MAIL；
  - advertised AUTH 未完成前 MAIL；
  - MAIL 前 RCPT；
  - RCPT 前 DATA；
  - envelope 后重新 greeting；
  - 非安全或非 advertised 阶段 AUTH。
- `:221-227` 在响应业务命令前执行状态检查，记录 violation 并返回 `503 bad command sequence`，不会计 accepted message。
- `smtp_credentials_test.go:230-375` 用真实 loopback 协议客户端逐项验证上述越序均返回 503、无 accepted message，且恰记录一次 violation。
- `:43-65` 与 `:205-209` 对 production mailer 的 accepted 路径断言完整命令序列：
  - implicit TLS + AUTH：`EHLO, AUTH, MAIL, RCPT, DATA, QUIT`；
  - required STARTTLS：`EHLO, STARTTLS, EHLO, MAIL, RCPT, DATA, QUIT`。
- `smtp_test_server.go:389-393` 明确要求 advertiseAuth 场景在 MAIL 前已认证。

结论：fixture 已成为每连接最小 SMTP 状态机，AUTH 前置、越序拒绝和 production 完整序列均有失败敏感的自动化证据。**CLOSED**。

### IDENTITY-S5.3-REV-R01-F003 — CLOSED

R01 问题：使用错误文本 `"bad certificate"` 吞掉预期 TLS 失败。

修复证据：

- scoped 扫描对 `.Error()`、`bad certificate` 及错误 `Contains` 模式均无匹配；
- `smtp_test_server.go:48-56` 通过显式 `expectedCloseStage` 描述预期关闭阶段；
- `:463-478` 按阶段记录预期关闭；非预期阶段保留 `%w` 包装的结构化 cause；
- `smtp_credentials_test.go:121-130` 的 hostname mismatch 场景同时要求：
  - production mailer 返回结构化 `ErrTLSRequired`；
  - server 关闭阶段为 `smtpStageTLS`。
- `:378-395` 独立回归证明：
  - 配置为预期 TLS 关闭时记录 TLS close stage；
  - 非预期 server failure 保留原结构化 cause，可由 `errors.Is` 识别。

结论：已消除错误字符串匹配，并通过显式场景/阶段区分预期 TLS 关闭与非预期服务端错误。**CLOSED**。

### IDENTITY-S5.3-REV-R01-F004 — CLOSED

R01 问题：cleanup 不关闭活动连接，超时后静默遗留 goroutine。

修复证据：

- `smtp_test_server.go:76-85` 增加受 mutex 保护的活动连接集合、closing 状态和一次性关闭/报告控制；
- `:165-180` 在 accept 后立即跟踪 raw connection，并设置有界 I/O deadline；
- `:406-420` TLS handshake 后重新设置 deadline；
- `:143-163` cleanup 顺序为：
  1. 标记 closing；
  2. 快照活动连接；
  3. 关闭 listener；
  4. 关闭所有活动连接；
  5. 等待 serve goroutine；
  6. 报告非预期 server error。
- `:158-160` cleanup 超时通过 `t.Errorf` 可观测，不再静默返回；
- `:450-460` 明确注册/注销活动连接；
- `smtp_credentials_test.go:397-428` 覆盖：
  - stalled TLS handshake；
  - 已建立 TLS 后的 partial SMTP command；
  两种场景均调用 cleanup 并断言活动连接归零。
- 定向 `count=10` 和 race 均通过，未观察到 hang、泄漏或数据竞争。

结论：活动连接跟踪、deadline、cleanup 顺序、超时可观测性和异常客户端回归均已闭环。**CLOSED**。

## 5. 新 Finding 扫描

- R01 原 finding：`4`
  - CLOSED：`4`
  - OPEN：`0`
- 本轮新增 finding：`0`
  - HIGH：`0`
  - MEDIUM：`0`
- 待决定项：`0`
- 非阻断建议：`0`

同范围检查未发现修复引入新的 HIGH/MEDIUM 正确性、安全、契约、并发、稳定性或兼容性问题。

## 6. Gate

四个 R01 finding 均已满足修复与复审条件；固定 revision 的定向普通、race、vet、gofmt 和错误文本扫描均通过；未发现新 HIGH/MEDIUM finding。

**Gate = PASS**

该结论仅代表本次固定代码对象的独立评审结果，不代替 Lead 的接受、任务状态变更或后续授权。

## 7. 未验证限制

本复审未验证且不得外推：

- 真实 SMTP provider 或外部真实收件；
- 真实 PostgreSQL；
- 部署、真实 Host、E2E；
- 容量、性能或 production readiness；
- 真实日志后端中的凭据泄漏；
- Lead 所报 Manifest 七项之外的环境。

本轮未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。未修改 R01、三个评审对象、production、Tasks/Plan 或其他文件。
