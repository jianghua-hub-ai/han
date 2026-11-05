# identity-auth-S5.3 独立代码评审 R01

- 评审日期：2026-10-25
- objectType：`code`
- Gate：**CHANGES REQUIRED**
- 评审者：非作者 `add-reviewer`
- 作者/实施者：Kant（Agent `01a1716c-b694-71c3-8d72-2a4fc1ab4eb3`）
- 固定工作区：`D:\AID\VSCode\Codex\Han`
- 固定分支/HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`

## 1. 固定评审对象

| 文件 | SHA256 |
|---|---|
| `test/integration/identity/smtp_credentials_test.go` | `B77862484C4015F9BD65786F3BC70C41F48BED0B4C26BC2A7B1FFBDC87C913D9` |
| `test/integration/identity/smtp_test_server.go` | `2958C16FA9216FF9FDD06C2469B48039F51963B798579A94151F86948C7893A8` |
| `test/integration/identity/testdata/certificates/README.md` | `B9CE292F5A8701756646170C128825E6D955BF302D422E928B6664FE4847F73F` |

最终落报告前已再次核对以上哈希、分支和 HEAD，均与委托固定值一致。三文件均为当前工作树中的未跟踪、未暂存文件；评审未修改它们。

## 2. 依据与独立性

已完整读取并应用：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- 已启用 `go-service` profile 的相关正文
- `.codex/guidance/task-execution-workflow.md` 中评审、授权和任务关闭合同
- `.add/project/knowledge/tasks/tasks.md:739-758` 的 S5.3 合同
- identity-auth design r44、test-plan r20 中 S5.3、SMTP、password-reset 相关合同
- 三个固定评审文件

有界核对的真实消费者/上游依据：

- `internal/mailer` 的 SMTP/TLS、认证和结构化 `ErrMessageRejected` 分类；
- `PasswordResetCoordinator`、对应 Store、首次 reset/resend 实现和测试；
- `AdministratorCreateCoordinator` 及 resend/残余窗口相关实现和测试。

本评审由非作者独立完成，未采用作者结论代替检查；Lead 提供的测试和 Manifest PASS 仅作为输入证据。未修改生产代码、测试对象、任务、设计、计划或其他共享文件。

## 3. 两轮实际检查范围

### Round 1：对象直接检查

- 逐行检查三个固定文件；
- 核对 loopback TCP、implicit TLS/required STARTTLS、证书信任与主机名校验；
- 核对 SMTP AUTH/MAIL/RCPT/DATA 是否真实经过协议；
- 检查内存证书/私钥生成、README 边界、build tag、文件范围；
- 检查 fixture 状态机、并发同步、deadline、关闭顺序、goroutine/channel 生命周期；
- 检查 Windows 可运行性、外网/真实 PostgreSQL 依赖、时间等待和错误字符串匹配。

### Round 2：真实消费者与契约检查

- 沿 `SMTPMailer` 核对 accepted/rejected/unknown 的结构化分类；
- 沿 PasswordReset 首次/reset resend 核对 durable delivery/settlement、凭据状态、旧凭据及 session/auth generation 语义；
- 沿 Administrator 创建/resend 核对新凭据残余窗口及明确拒绝后的失效语义；
- 核对 worker/replay 不重复发信及任务、design r44、test-plan r20 的边界；
- 未扩为全库泛审。

## 4. 独立执行证据

以下命令由 Reviewer 在固定工作区独立执行，均 PASS：

```text
go test -tags=integration ./test/integration/identity -run 'TestCredentialSMTP|TestSMTPResidualWindow' -count=10
go test -race -tags=integration ./test/integration/identity -run 'TestCredentialSMTP|TestSMTPResidualWindow' -count=1
go vet -tags=integration ./test/integration/identity
```

重复测试约 13.6 秒完成。运行环境为当前 Windows 工作区。

Lead 已提供以下现场 PASS 作为输入证据，但 Reviewer 未独立复跑整套 Manifest：

```text
gofmt -l .
go mod verify
go vet ./...
go build ./...
go test -tags=contract ./...
go test -race ./...
go test ./...
```

## 5. Findings

### IDENTITY-S5.3-REV-R01-F001 — HIGH — PasswordReset 三分流没有闭环证明凭据状态，resend 也只覆盖 accepted

- 位置：
  - `test/integration/identity/smtp_credentials_test.go:209-327`
  - `test/integration/identity/smtp_credentials_test.go:793-807`
  - `test/integration/identity/smtp_credentials_test.go:863-880`
  - `test/integration/identity/smtp_credentials_test.go:985-999`
- 违反依据：
  - S5.3 要验证 accepted/rejected/unknown 三分流、普通 password-reset 首次/resend 的 settlement，以及凭据残余窗口；
  - 明确拒绝必须使本次新凭据失效；unknown 必须保持 unresolved，不能冒充 rejected；
  - 评审清单要求测试对关键持久化副作用具有失败敏感性。
- 证据：
  - `smtpPasswordResetSnapshot` 已包含 `credential`，fixture 在匹配 `SET state='recovery_invalidated',invalidated_at=` 时也会把它改为 `TemporaryCredentialRecoveryInvalidated`；
  - 主断言 `:285-305` 只比较 operation、delivery、settlement、evidence，并未比较 `state.credential`；
  - 因此若生产 Store 遗漏明确拒绝后的新凭据 invalidation，而 operation/delivery/settlement 更新仍存在，此测试仍可通过；
  - resend 矩阵只有 `ordinary resend ... accepted`，没有真实 SMTP rejected/unknown resend。
- 触发场景：
  1. PasswordReset 明确 SMTP 拒绝；
  2. Store 未执行或回归丢失新凭据 `recovery_invalidated` 更新；
  3. 现有集成测试只看到 rejected/invalidated settlement evidence，未检查凭据本身，形成假阳性。
- 影响：
  - 测试不能证明明确拒绝后新凭据已退出残余窗口；
  - 不能证明 resend 的 rejected 只失效新值且保留旧值，也不能证明 unknown 保持 unresolved；
  - accepted/rejected/unknown 与凭据、旧 authority、auth generation/session revocation 的组合契约未完整闭环。
- 修复条件：
  - 对首次和 resend 的适用三分流显式断言新凭据状态；
  - rejected 必须断言仅本次新凭据失效，旧凭据/旧 authority 按合同保持；
  - unknown 必须断言仍为 pending/unresolved，且不产生 rejected invalidation；
  - 同时断言 auth generation、session revocation 和 dispatch/mail attempt 不发生越权变化；
  - 增加 mutation-sensitive 证据：删除明确拒绝后的凭据 invalidation 时，测试必须失败。

### IDENTITY-S5.3-REV-R01-F002 — MEDIUM — SMTP fixture 未强制协议顺序或 AUTH 前置条件

- 位置：
  - `test/integration/identity/smtp_test_server.go:163-299`
  - `test/integration/identity/smtp_credentials_test.go:40-59`
  - `test/integration/identity/smtp_credentials_test.go:181-203`
- 违反依据：
  - S5.3 要证明真实 loopback SMTP/TLS 经 AUTH/MAIL/RCPT/DATA，而不是只对命令作宽松应答；
  - 测试替身必须能拒绝违反生产协议合同的行为，不能让错误实现获得 accepted。
- 证据：
  - server 只按当前命令 verb 分支应答，不保存 greeting、TLS 后 EHLO、AUTH、MAIL、RCPT、DATA 的状态；
  - `advertiseAuth=true` 时没有要求 AUTH 成功后才允许 MAIL；
  - accepted AUTH 用例只断言 message 数、TLS 版本和“未在明文发送”，未断言 AUTH 必须出现及完整顺序；
  - 当前 fixture 会接受跳过 AUTH 的 `MAIL`，也会接受 RCPT/DATA 越序。
- 触发场景：
  - production mailer 回归为未执行 AUTH，或把 MAIL/RCPT/DATA 顺序打乱；fixture 仍可能返回成功并计入一封 accepted message。
- 影响：
  - “真实经过 SMTP AUTH/MAIL/RCPT/DATA”的核心验收可能假阳性；
  - 当前 PASS 不能证明认证要求和协议状态机均被 production mailer 遵守。
- 修复条件：
  - 为每个连接维护最小 SMTP 状态机并拒绝越序命令；
  - advertise/required AUTH 场景必须在认证成功前拒绝 MAIL；
  - accepted 场景显式断言预期完整序列，至少覆盖 greeting、TLS 后重新 greeting、AUTH（适用时）、MAIL、RCPT、DATA、QUIT/连接结束。

### IDENTITY-S5.3-REV-R01-F003 — MEDIUM — 通过错误字符串吞掉预期 TLS 失败

- 位置：`test/integration/identity/smtp_test_server.go:105-117`
- 违反依据：
  - 任务合同明确禁止错误字符串匹配；
  - coding/checklist 要求错误按结构化类型或明确场景处理，避免依赖 Go/平台文本。
- 证据：

```go
!strings.Contains(s.serveErr.Error(), "bad certificate")
```

  此判断用于忽略主机名不匹配时服务端观察到的 TLS 错误。
- 触发场景：
  - Go 版本、TLS 实现或 Windows 错误文本变化；
  - 无关错误恰好包含该子串。
- 影响：
  - 预期 hostname mismatch 用例可能因文本变化而不稳定；
  - 无关服务端故障可能被错误吞掉，掩盖真实失败。
- 修复条件：
  - 用场景配置/预期关闭阶段或可结构化判断的错误链表示预期 TLS 握手失败；
  - 不再读取错误文本决定 PASS/FAIL，并确保非预期服务端错误仍导致测试失败。

### IDENTITY-S5.3-REV-R01-F004 — MEDIUM — cleanup 不关闭活动连接，超时后静默遗留 goroutine

- 位置：
  - `test/integration/identity/smtp_test_server.go:60-71`
  - `test/integration/identity/smtp_test_server.go:130-160`
  - `test/integration/identity/smtp_test_server.go:163-173`
- 违反依据：
  - 集成 fixture 必须有确定的 deadline、关闭顺序和 goroutine 生命周期；
  - cleanup 失败不能静默，race/hang 风险应可观测。
- 证据：
  - server 只保存 listener，不保存 `Accept` 后的活动 connection；
  - connection 的 `ReadString` 没有自身 deadline；
  - `close()` 只关闭 listener；若已 accept，无法打断活动连接读取；
  - 等待五秒超时后 `close()` 静默返回，不报告仍存活的 serve goroutine。
- 触发场景：
  - 客户端回归为不发送 QUIT、未关闭连接或卡在半包命令；
  - 测试提前失败并进入 `t.Cleanup`，但服务端已 accept。
- 影响：
  - 单例可额外等待五秒并泄漏 goroutine/连接；
  - 重复运行、race 或后续测试可能出现偶发挂起，且 cleanup 不暴露根因。
- 修复条件：
  - 受锁保护地跟踪活动连接；
  - 为连接读写设置有界 deadline；
  - cleanup 按明确顺序关闭 listener 和活动连接，再等待 goroutine；
  - 超时必须通过测试错误暴露，不能静默返回。

## 6. Finding 统计与结论

- 既有 finding：`0`
- 新增 finding：`4`
  - HIGH：`1`
  - MEDIUM：`3`
- 待决定项：`0`
- 非阻断建议：`0`

正面证据：

- 测试使用 `127.0.0.1:0` 的真实 loopback TCP，并真实调用 production `SMTPMailer`；
- 覆盖 implicit TLS、required STARTTLS、降级、hostname mismatch、RCPT rejection 和 post-DATA disconnect unknown；
- TLS 最低版本为 1.2，CA/server key 在内存生成，README 明确禁止持久化私钥；
- unknown 用例明确要求不匹配结构化 `ErrMessageRejected`；
- 未发现外部邮件、真实 PostgreSQL 或 production/public contract 修改。

但 F001 使凭据残余窗口和 resend 三分流的核心验收尚未被失败敏感地证明；F002 使 AUTH/SMTP 命令链存在协议假阳性；F003 直接违反“不得使用错误字符串匹配”；F004 留有可重复触发的清理挂起风险。因此本 revision：

**Gate = CHANGES REQUIRED**

## 7. 未验证限制

本评审不外推、也未验证：

- 真实 SMTP provider；
- 外部真实收件；
- 真实 PostgreSQL；
- 部署、真实 Host 或 E2E；
- 容量、性能或 production readiness；
- 真实日志后端中的凭据泄漏。静态检查未发现本三文件主动输出密码，但未做 logger capture/日志后端验证。

Manifest 七项采用 Lead 输入证据，Reviewer 未独立全量复跑。未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。

## 8. 复审条件

复审须提供：

1. 对应原 finding 的修复 revision 与三个固定文件的新 SHA256；
2. F001 的首次/resend accepted/rejected/unknown 凭据状态及旧 authority 回归；
3. F002 的状态机负例和完整命令序列证据；
4. F003 的零错误字符串匹配证据；
5. F004 的异常客户端/提前失败 cleanup 回归及适用 race 结果；
6. 定向普通、`-race`、`go vet` 以及适用 Manifest 结果。

若修复需要第四个文件、生产/public contract、任务/design/test-plan 变化，须先停止并取得新的范围/合同授权，不得在本评审结论下直接扩写。
