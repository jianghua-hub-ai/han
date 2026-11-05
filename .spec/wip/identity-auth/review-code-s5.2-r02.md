# Identity S5.2 r01 定点代码复审 r02

- **评审日期**：2026-10-23
- **Reviewer**：Faraday（独立非作者 Reviewer）
- **objectType**：code
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **原报告**：`.spec/wip/identity-auth/review-code-s5.2-r01.md`
- **结论**：**FAIL；原 F001、F003、D001 CLOSED，原 F002 LIMITED；新增 1 项 HIGH finding**

本结论仅绑定第 1 节列出的固定代码对象及已核对的现役依据。Reviewer 未修改被审代码、设计、测试规划、任务、计划、migration 或旧报告；本轮只新增本报告。

## 1. 固定对象与依据

评审开始及落报告前均核对分支、HEAD、git blob 与 SHA-256。十个固定对象均与委托值一致：

| 固定对象 | git blob | SHA-256 |
|---|---|---|
| `internal/identity/administrator_create.go` | `02b4fb03c7bd51c0dcf50af54ba91a3fa11937f0` | `7A73F17BED8FD6EBE321AC5DAF3F02AE1EE7BBA8E4A32D06216608AA1A1E75D7` |
| `internal/identity/administrator_create_test.go` | `9b2d38a89db3b8a75863268ffcf044326478b30e` | `E94CC8EB8F3B4F0D2633C55354B350DE3CC65F6080A78F853C4F0D85C9C12552` |
| `migrations/0007_idempotency_response_snapshot_v1.sql` | `92c5a163e84d58ec56f28082b86bbc4766309be1` | `9D3E642D4F5C5DE81113E23AEBD70E2E07BF583E140697D335DB4DDA35F01F9F` |
| `internal/idempotency/store.go` | `e221eaa45698480b670cca037a7e0d3b4cd2fdfc` | `E9AAE069942F310EB155FBCAE3E981B55C87ABCAFCF3ACA74730292483C57FA5` |
| `internal/idempotency/store_test.go` | `2ba775052044541569716efedb5c60fce4e4eec9` | `DB291F9AE7F501F60ABF71F715FC2DD2961067E8598CED64348A7A6AC77DAC5C` |
| `internal/idempotency/store_integration_test.go` | `813bf5700a792b28e5a60f2ec4a666f791d637d7` | `BAE146223ABADF5E3884962E2BDD119AE46964D694B3BE4A38B331DAA956CD5A` |
| `internal/identity/handler.go` | `685148bcafc803b701a083e15cb684f5a3f0b752` | `F21E1E9B824DADCD784AA89E4A9865CDD85BF17A372B4D9C2964A91250C456D9` |
| `internal/identity/handler_test.go` | `c33eec776c607cd1b9f1b78b13c0a0ee65471f6d` | `7A32FEF8C9C94489D8D8E055C6B5ACEB7E2EA0304BCF028BDE0CF1D0BCA6373F` |
| `cmd/han-server/main.go` | `afa2e52e924bf01c698a6a6510f44a59ec8cbdc0` | `FDED5C74F9BB06D3BACD6E3B349D05ABF59B75C93C74C4EDA59D28EE0E9ABA30` |
| `cmd/han-server/main_test.go` | `f16d31e4922b0ef8e25bb4f83df7301b214f4819` | `2B5336CF9994F795CB01928BB07CDAE3D47C0FED4E0D0AF07E66BAD9E3442884` |

直接消费者快照：

| 文件 | SHA-256 |
|---|---|
| `internal/mailer/smtp.go` | `3B66245D0CDF4A7FB1E878EF68CE04342A32DABF0214AD1F5B8871F32C444F4F` |
| `internal/mailer/smtp_test.go` | `08FC9DF16CEBC7A1A47E293647EA8B6E2F0B5A2EED0154791B019F7FCEA9412E` |

现役依据摘要均匹配委托值：

- design r33：`47FC3598CE2BCA7A1E80032FEC040BCF63344C869B4EF12F9BE5998589563513`
- design review r05：`8E3E478BD94D4CF827B2A7E1E5FD31DE052EFC2174172AB2C3191C2AE38F245D`
- test-plan r13：`F5A3E2CEF38C36E595D8217BDD85BBA7CF55FC0F1DBB311D773E7DDBD1076749`
- tasks S5.2：`6D7558F833F19FED9C5A3CE921F71E0E2A9F56EF31B86C5BE0FF44498AA75886`
- B6A Plan：`C9CB97FB48E5D92544635BB8B7C5C4BAB45E122149CF817F665D7B31983B7E3D`

## 2. 两轮实际评审范围

### Round 1：固定代码与测试直接复审

逐项检查：

1. `AdministratorCreateCoordinator` 的 initial-admin 授权、只读幂等分类、稳定终态重放、动态 Gate、事务、SMTP、审计与失败/uncertain 状态。
2. legacy/v1 严格 codec、`0007` CHECK、canonical success/failed snapshot、identifier conflict 空 `field_errors`、当前请求 trace 与零副作用回放。
3. SMTP operation budget 的配置、较早 deadline、cancel 调用及 coordinator 测试。
4. r33 startup config、effective-at parser、secret clear、prepared/factory ownership、三条 production route、中间件顺序、404/405/`Allow`、登录与改密回归。

### Round 2：真实消费者与存储/资源接缝

沿以下有界链路核对：

`production startup/composition/router → identity Handler/Coordinator → idempotency PostgreSQL store/0007 → mailer SMTP adapter → session/audit/security/platform`

重点检查真实 `SMTPMailer` 是否在连接建立后的 AUTH/MAIL/RCPT/DATA 阻塞期间响应 parent context cancellation；并核对 Go 1.27 标准库 `net/smtp` 的同步 I/O 行为及现有 adapter 测试能否发现该问题。未扩展为全仓无界审查。

## 3. 原 finding / 决定复审状态

### `IDENTITY-S5.2-R01-F001` — CLOSED

关闭证据：

- `administrator_create.go:309-322` 每个请求先从 repository 重新确认当前主体仍为 active initial admin。
- `administrator_create.go:341-373` 授权通过后先执行只读 `ResolveExisting`；稳定 succeeded/failed 直接进入 snapshot replay，processing、uncertain 与冲突保持独立分类。
- `administrator_create.go:374-405` 仅在没有既有记录时执行 ProcessingGate、密码策略、current notice 与新占位。
- `administrator_create.go:1003-1046` 重放只消费版本化 response snapshot，不再从可变用户、凭据或 Gate 状态重建。
- `administrator_create.go:1154-1198` success 固定八个业务字段；failed 固定 `code/message/field_errors`，identifier conflict 为 `409 / identity.identifier_conflict / 身份标识已被使用 / []`。
- `store.go` 的 strict legacy/v1 reader/writer 与 `0007` CHECK 拒绝 extra、missing、null、type drift、混合格式和未知版本；快照不保存 trace，Handler 使用重放请求本次 trace。
- 单元及 PostgreSQL fixture 覆盖状态变化后重放、trace A→B、canonical success/failed、坏快照 fail closed 与零业务副作用。

### `IDENTITY-S5.2-R01-D001` — CLOSED

现役顺序与已采用决定一致：

1. 每请求先重新授权 active initial admin；
2. 授权通过后，同键同摘要稳定终态重放优先于 ProcessingGate、current notice、密码策略和后续业务状态变化；
3. 同键不同摘要仍冲突；
4. processing、uncertain 及无记录的新执行保持各自既定顺序。

无记录检查后到 `Begin` 的并发窗口由 `Begin` 再分类，不构成绕过稳定终态或动态 Gate 的证据。

### `IDENTITY-S5.2-R01-F002` — LIMITED

原 finding 中“production 注入真实 adapter 后因没有 deadline 而稳定失败”的触发条件已消除：

- `HAN_SMTP_OPERATION_BUDGET` 必填、无代码默认、合法闭区间 `1s..60s`；
- coordinator 使用 `context.WithTimeout`，自然取得 request deadline 与配置预算的较早者；
- `Send` 返回后及时调用 cancel；
- production composition 注入真实 `SMTPMailer`；
- 测试覆盖 missing/invalid budget、较早 deadline、already-canceled、timeout 与 SMTP 未受理零可用管理员。

但 r33 要求的**中途取消及时停止发送并释放连接**尚未由真实 adapter 满足，见新增 HIGH finding `IDENTITY-S5.2-R02-F001`。因此原 F002 只能标为 LIMITED，不能 CLOSED。

### `IDENTITY-S5.2-R01-F003` — CLOSED

关闭证据：

- `cmd/han-server/main.go` 的唯一 production factory 构造并共享登录、改密、AdministratorCreateCoordinator、AdministratorPrivacyCoordinator、静态声明 reader、幂等 keyring/store、mailer、session、audit 与 repository 依赖。
- 五条既有业务 route 从唯一 `newProductionBusinessHandler` 注册；三条管理员 route 的 method/path、JSON 404/405、route-specific `Allow` 与 Trace/Origin/Session/CSRF/Principal/Gate 顺序符合 r33。
- startup 配置覆盖管理员声明四键、幂等 current/retained keyring、lock/TTL、SMTP endpoint/from/auth/CA/TLS/budget；缺失、空白、非法和交叉约束失败在 DB/listener/route 副作用前 fail closed。
- secret clear、factory nil、prepared/factory/bind failure 逆序 unwind、正常关闭和共享实例有对应测试。
- `2026-10-24T00:00:00Z` 经 config→reader→Handler HTTP 逐字节保持；fractional、offset、非规范、零值及解析失败在启动阶段零调用拒绝。
- 现役 login/password route 的 method、middleware 与错误合同未见回退。

## 4. 新 findings

### CRITICAL

无。

### HIGH

#### `IDENTITY-S5.2-R02-F001` — SMTP 连接建立后不能及时响应 parent context cancellation

- **位置**
  - `internal/mailer/smtp.go:86-110`
  - `internal/mailer/smtp.go:115-142`
  - 测试缺口：`internal/mailer/smtp_test.go`
- **违反依据**
  - design r33 要求 timeout/cancel 后立即停止发送尝试、同步返回并释放 timer、连接及临时资源。
  - test-plan r13 的 R-I17 / I-S52-07 要求取消后不得继续后台发送，并验证 goroutine、timer、connection 回到基线。
- **触发场景**
  1. 请求 context 的 deadline 晚于当前时刻，真实 adapter 成功建连，并执行一次 `connection.SetDeadline(deadline)`。
  2. SMTP server 在 AUTH、MAIL、RCPT、DATA 写入或 DATA acceptance reply 任一同步 I/O 阶段阻塞。
  3. 客户端断开或上游主动取消 parent context，取消发生在原 deadline 之前。
  4. `SMTPMailer.Send` 没有 cancellation watcher 去关闭连接或把活动连接 deadline 提前到当前时刻；`net/smtp` 的这些同步命令不接受 context。
  5. 因此阻塞调用可继续到原 deadline，甚至在取消后继续推进 SMTP 命令；defer 的连接关闭只能在阻塞调用返回后发生。
- **证据**
  - `smtp.go:86-105` 仅取得 context deadline并一次性写入 socket deadline。
  - `smtp.go:120-121` 仅在 `MAIL` 前检查一次 `ctx.Err()`。
  - `smtp.go:123-138` 的 `Mail/Rcpt/Data/io.WriteString/writer.Close` 均无 context cancellation 接缝。
  - Go 1.27 标准库 `net/smtp.Client` 的 `Mail/Rcpt/Data/cmd` 使用同步 `textproto` I/O，不会因传入本 adapter 的 context 被取消而自行返回。
  - 现有 coordinator fake 证明派生 context 的 deadline/cancel，但没有证明真实 adapter 在已建连且命令阻塞时及时中断；现有 SMTP adapter tests 也没有该场景。
- **影响**
  - 已取消请求仍可能占用 SMTP 连接和数据库事务直到 operation budget 上界（当前允许最高 60 秒）。
  - SMTP 命令或 DATA acceptance 可能在取消后继续发生，形成调用方已放弃但邮件仍被受理的副作用窗口。
  - 不能满足 S5.2 对 cancel、无后台发送和资源及时释放的明确验收合同。
- **最小修复建议**
  - 将 context cancellation 与活动连接生命周期绑定；例如使用受控 cancellation callback 在取消时立即关闭连接或把其 deadline 设置为当前时刻，并在正常完成时可靠停止 watcher。
  - 明确 DATA 已受理竞态的返回/状态语义，避免把已受理误报为可安全重试。
  - 使用 deterministic fake transport 或 `net.Pipe`，分别在 AUTH/MAIL/RCPT/DATA 阻塞阶段取消，断言有界及时返回、连接释放且不再发送后续命令；配合 `-race` 检查 watcher 清理。

### MEDIUM / LOW

无。

## 5. 待决定与非阻断建议

### 待决定

无。新增 finding 不要求改变已确认公共契约、schema、migration 或架构；属于现役 SMTP adapter 对既定 cancellation 合同的实现缺口。

### 非阻断建议

除修复 finding 所需的 deterministic cancellation 回归外，无独立建议。

## 6. Reviewer 独立验证命令

| 命令/检查 | 结果 |
|---|---|
| `git branch --show-current` / `git rev-parse HEAD` | `main` / 固定 HEAD 匹配 |
| `git hash-object -- <10 fixed files>` + `Get-FileHash -Algorithm SHA256` | 十个固定对象全部匹配 |
| `go test ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1` | PASS |
| `go test -race ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1` | PASS |
| `go vet ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server` | PASS |
| `gofmt -l <本轮固定 Go 文件>` | PASS，无输出 |
| `go version` | `go1.27.0 windows/amd64` |
| 读取 `$GOROOT/src/net/smtp/smtp.go` | 完成；确认 AUTH/MAIL/RCPT/DATA 为无 context 参数的同步 I/O |
| 落报告后再次执行十对象 blob/SHA-256 核对 | PASS，全部未漂移 |
| `git diff --check -- <10 fixed objects>` | PASS |
| `git -c core.safecrlf=false diff --no-index --check -- NUL <new report>` | 无 whitespace error 输出；退出码 1 仅表示新文件与空文件存在差异 |

一次检查曾误将 `migrations/0007_idempotency_response_snapshot_v1.sql` 传给 `gofmt -l`，命令因 SQL 不是 Go 语法而失败；该失败不是代码 finding。随后仅对 Go 文件重新执行正确命令并 PASS。

Lead 提供的 PostgreSQL 18 integration 与 Manifest 七项 PASS 作为外部已有证据记录，不冒充 Reviewer 独立执行。Reviewer 未读取或打印 User scope `HAN_TEST_POSTGRES_DSN`。

## 7. 未验证风险与限制

1. 未独立复跑 PostgreSQL 18 integration；`0007` 的真实升级、CHECK 与 store fixture 依赖 Lead 已提供的安全子进程证据及本轮静态/普通测试核对。
2. 未连接真实 SMTP/TLS provider，未执行 DNS/TCP/TLS/AUTH 或真实邮件受理；本 finding 来自真实 adapter 与标准库 I/O 接缝，而不是生产网络实测。
3. 未启动真实 Host、浏览器、部署环境或 E2E；PASS/FAIL 不外推为部署、容量或 production readiness。
4. 未独立复跑 Lead 的全仓 build/contract/race/test 七项矩阵；本轮只运行第 6 节的有界命令。
5. 共享工作树含大量范围外未提交成果；结论不覆盖未列入固定对象或直接接缝的其他文件。

## 8. Code Review Gate 与复审条件

### Code Review Gate: FAIL

F001、F003 与 D001 已关闭；F002 的 deadline/config/composition 主缺口已修复，但真实 SMTP adapter 在连接建立后的 parent cancellation 上仍违反已确认的及时停止和资源释放合同，因此 F002 为 LIMITED，新增 HIGH finding `IDENTITY-S5.2-R02-F001` 保持 OPEN。

复审条件：

1. 作者提交固定的新代码对象，说明如何把 parent cancellation 传播到活动 SMTP 连接，并保持 DATA 已受理后的不可安全重试语义。
2. 增加已建连后 AUTH/MAIL/RCPT/DATA 阻塞取消的确定性 adapter 测试，证明及时返回、无后续命令和连接/watcher 释放。
3. 重跑受影响 mailer/identity/cmd 测试、`-race`、`vet` 及项目适用 Gate。
4. 由独立非作者 Reviewer 对 `IDENTITY-S5.2-R02-F001` 与相关回归做定点复审。

本结论不代替 Lead/用户的风险接受、修复授权或后续交付决定。

## 9. 实际读取范围

### Role / Skill / Rule / Profile

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- add-code-review 引用的产品安全红线
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.codex/guidance/observability-standards.md`
- `.codex/guidance/tracing-standards.md`
- 已启用 `go-service` profile 的适用规则

### 依据与历史

- `.spec/wip/identity-auth/review-code-s5.2-r01.md`
- `.add/project/knowledge/design-workspace/identity-auth/design.md`
- `.spec/wip/identity-auth/review-design-r33-s5.2-r05.md`
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
- `.add/project/knowledge/tasks/tasks.md`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`
- 适用 identity/framework ADR、boundary 与 architecture

### 固定代码与有界直接接缝

- 第 1 节十个固定对象
- `internal/mailer/smtp.go`
- `internal/mailer/smtp_test.go`
- 与本路径直接相关的 session、audit、security、platform 配置/错误/trace 定义及测试
- Go 1.27 标准库 `net/smtp` 直接实现
