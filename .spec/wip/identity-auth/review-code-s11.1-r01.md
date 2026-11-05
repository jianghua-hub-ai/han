# identity-auth-S11.1 Code Review R01

- Reviewer：Copernicus（非作者）
- objectType：code
- 日期：2026-10-25
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：**Code Review Gate: CHANGES REQUIRED**

## 1. 固定对象与 revision

开始与报告落盘前均核对；五个代码对象无漂移。

| 对象 | SHA256 |
|---|---|
| `internal/identity/handler.go` | `29961F44AD74A8380C5DF8187359456066EF729666F7DF4ED5AE776D5AAFCD18` |
| `internal/identity/temporary_credential_resend.go` | `26B45D995848C83C37B0EDA15C99896B6FA7A7CBB621E679FB05069488F2C040` |
| `internal/identity/temporary_credential_resend_test.go` | `9C1522ED2B8B228368B19500FDAD4D5B03897F8E1A64EBDB41A9D791AB0D1D35` |
| `cmd/han-server/main.go` | `1895BAE07E222004CE5191779A0880C008B6B00A53C039826B58AA331D2241C8` |
| `cmd/han-server/main_test.go` | `66896850C057F884AA4443B81C07FA2D68B2092B77AA84F025F4444B67E839CF` |
| Planning Review `review-tasks-plan-s11.1-r01.md` | `D5316A308E8F8A560CD263AD65C35EDFA74666DD5388A2379AAD4C135474B7AB` |

五个代码文件均为 untracked；本评审以全文、固定哈希和实际测试为证据，不以空 `git diff` 为证据。

## 2. 两轮实际检查范围

### Round 1：固定对象直接审查

- 全文检查 `temporary_credential_resend.go` 与对应测试。
- 检查 `handler.go` 的 exact route 解析、method/path、Idempotency-Key、响应 envelope 与 secret 输出面。
- 检查 `main.go` 的唯一 production builder、coordinator 复用、Origin/Session/CSRF/AccessGeneral 顺序、route registration 与 fail-fast。
- 检查 `main_test.go` 的 production router、test seam、共享实例和拒绝路径证据。
- 逐项对照 design r44 §3.3.12、stories r16 S11、test-plan r20 I-PRD、Tasks/Plan S11.1 五文件合同及 ADR-0011/ADR-0023。

### Round 2：真实消费者与持久权威有界追踪

- 追踪既有 `PasswordResetCoordinator`、`PasswordResetStore`、operation/job/lineage classifier、operator/worker。
- 追踪 `AdministratorCreateCoordinator`、Mailer/SMTP budget、Audit、Idempotency、Clock/IDs/Hasher/Policy。
- 核对 Login、PasswordChange、Session 继续消费同一 durable Gate。
- 核对 `cmd/han-server` 仍是唯一 composition root，未新增 Store/Gate/operator/worker/global。
- 核对 worker 不持有邮件发送权，普通 password-reset resend 继续沿既有 dispatch/settlement 链。

## 3. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

#### IDENTITY-S11.1-REV-R01-F001 — 无效或 stale 的新 key 在权威上下文校验前已生成并哈希新秘密

- **位置**：
  - `internal/identity/password_reset.go:220-257`（本卡真实复用消费者）
  - `internal/identity/temporary_credential_resend.go:133-165`
  - 测试缺口：`internal/identity/temporary_credential_resend_test.go:499-563`
- **违反依据**：
  - design r44 §3.3.12 lines 2384-2385、2397-2405：只有 key 不存在、Gate clear 且 current old credential/predecessor 一致时才建立 successor；User-first 锁内应先完成 Gate/current/predecessor 校验。
  - stories r16 S11 lines 383-385、390：无上下文、unresolved/hold、stale/ineligible 必须拒绝，且候选创建保持零副作用。
- **触发场景**：
  - 新 Idempotency-Key 指向没有 password-reset predecessor 的用户；
  - old credential 已 consumed/superseded/expired/recovery_invalidated；
  - 合法历史已被后续 B/C 替换而当前上下文 stale；
  - Gate/lineage classifier 最终拒绝该候选。
- **证据**：同键不存在后，coordinator 先调用 `Secrets.Generate`、PasswordPolicy、两个 ID、latest credential 和 `NewTemporaryCredential`（含 Hash），随后才由 resend authority / Store 执行 `CheckLockedUser` 与 `authorizeNewPasswordResetCandidate`。现有拒绝测试仅断言 Mailer/commit/store writes 为零，没有断言 secret/ID/hash 为零。
- **影响**：
  - 被拒绝请求仍进行敏感秘密准备和昂贵哈希，并在 User 锁事务内扩大资源占用；
  - Go string 中已生成的秘密不能可靠原地清除；
  - 当前测试无法捕捉该顺序回退。
- **建议**：在 secret/ID/hash 前完成 same-key 分类后的 writer Gate、current old credential、predecessor/stale/hold 校验；补充 missing/stale/consumed/unresolved 的零 secret、零 ID、零 hash 断言。
- **复审条件**：普通/race 测试证明所有上述拒绝分支在敏感准备前停止，并仍保持零持久写、零邮件、old capability 不变。
- **范围说明**：修复点落入既有 `password_reset.go` 或其 authority 接缝，超出当前固定五文件。应先由 Lead/Task-Planner 对账文件所有权；Reviewer 不自行扩权。

#### IDENTITY-S11.1-REV-R01-F002 — same-key succeeded replay 不能保证返回首次响应的原 expiry

- **位置**：
  - `internal/identity/temporary_credential_resend.go:567-579,635-658,760-766`
  - 真实复用：`internal/identity/password_reset.go:663-665,740-758`
  - 测试：`internal/identity/temporary_credential_resend_test.go:639-735`
- **违反依据**：
  - design r44 §3.3.12 lines 2419-2421：succeeded 必须重放原 accepted 快照。
  - stories r16 S11 lines 381-384：same-key replay 保持原 operation/generation 与原终态。
  - test-plan r20 I-PRD-02：稳定重放原 terminal snapshot。
- **触发场景**：production `SystemClock` 在幂等 begin 与新秘密实际签发之间跨越一个 RFC3339 秒边界；Argon2、锁或调度延迟会放大该窗口。
- **证据**：
  - 首次响应使用 `pending.ExpiresAt = 第二次 Clock.Now() + 7d`；
  - 成功快照只保存 `Outcome/ResourceID`；
  - replay 用 `record.CreatedAt + 7d` 重建 expiry；
  - Handler 按 RFC3339 输出，因此跨秒时同键重放会返回不同的 `temporary_credential_expires_at`；
  - 现有测试使用固定 Clock，且 succeeded replay 未比较首次与重放 expiry，掩盖该差异。
- **影响**：幂等重放不再是原公共响应；客户端可观察到同键成功结果漂移，违反稳定 terminal snapshot 合同。
- **建议**：使用唯一、持久且可重放的 credential issuance/expiry 事实；不得从不同时间点近似重建。补充可前进 Clock 的首次成功→same-key replay 测试，逐字节比较公共 expiry。
- **复审条件**：两 purpose 的首次成功和同键重放在跨秒 Clock 下返回完全相同 expiry，且 replay 仍零 secret/ID/hash/mail/业务写入。
- **范围说明**：普通 password-reset 分支的根因位于既有 `password_reset.go`/持久快照合同；如修复需要第六文件或 migration，须先停止并重新授权写集/契约，不得在本卡暗改。

#### IDENTITY-S11.1-REV-R01-F003 — production 安全验收测试绕过业务授权，且未覆盖 body purpose 串改

- **位置**：
  - `cmd/han-server/main_test.go:5445-5617`
  - `internal/identity/temporary_credential_resend_test.go:21-145`
  - production seam 注入：`cmd/han-server/main.go:2263-2270`
- **违反依据**：
  - Tasks S11.1：production route 必须证明 Cookie/Origin/Session/CSRF/角色授权、Idempotency-Key、404/405 与拒绝路径零业务下游。
  - test-plan r20 I-PRD-01/I-CRED-03：body/query/header 均不能串改 purpose；两 purpose 的角色组合必须自动验证。
- **证据**：
  - production route 正例和拒绝例用 `articleProductionTestSeam.temporaryCredentialResend` 替换真实 purpose router/coordinator；
  - 当前“authorization”例只覆盖 onboarding restriction；
  - 未覆盖 member principal、普通 admin 调用 `new_administrator` 的 403、initial admin/current-admin 正向授权；
  - purpose 串改只覆盖 header/query，没有 body；
  - identity 层 new-administrator 测试全部使用 initial-admin 正例，没有无权 actor 的零下游断言。
- **影响**：现有自动化不能证明任务要求的角色授权和 body purpose 防串线，也不能阻止 test seam 日后掩盖 production 业务授权回退。
- **建议**：保留 seam 的 router/wiring 测试，同时增加经过真实 `TemporaryCredentialResendService`/coordinator 授权路径的角色矩阵；为 body/header/query 串改分别断言 route-derived purpose、稳定 403/错误 envelope及零 secret/mail/持久写。
- **复审条件**：定向普通/race 测试覆盖上述角色与三种串改入口，且生产 seam 不替代业务授权证据。

### LOW

无独立 LOW finding。

## 4. 已确认未回退的合同

- purpose 由 exact path 固定，header/query 当前不能改变。
- production builder 复用同一 `PasswordResetCoordinator` 与同一 `AdministratorCreateCoordinator`。
- 同一 `PasswordResetStore`/Gate/operator/worker 继续被 production composition 与三消费者复用。
- 两条 exact POST route 可达；错误 method 为 405 并带 `Allow: POST`，未知 path 为稳定 JSON 404。
- Origin → Session/CSRF → AccessGeneral 顺序存在；缺关键依赖时 constructor fail closed。
- `new_administrator` 保持 ADR-0011，不创建 PasswordReset operation/job；普通 password-reset 分支继续复用 ADR-0023 durable authority。
- 未发现第六实现文件、schema/migration、公共错误码、第二 authority/root/global 或 worker 邮件发送权。

上述正向事实不关闭 §3 的 findings。

## 5. 实际执行命令

以下均未读取或使用 `HAN_TEST_POSTGRES_DSN`，未连接真实 PostgreSQL/SMTP：

```text
PASS  go test ./internal/identity ./cmd/han-server -run 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1
PASS  go test -race ./internal/identity ./cmd/han-server -run 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1
PASS  go test ./internal/identity ./cmd/han-server -run 'TestTemporaryCredentialResend(NewAdministratorFailurePreservesOld|NewAdministratorReplayHasNoSensitivePreparation|PasswordResetRejectsMissingOrStaleContext|PasswordResetUsesDurableRecoveryChain|PasswordResetStalePreAcceptanceInvalidatesOnlyNew)|TestProductionTemporaryCredentialResend(CompositionRoutes|OriginAuthorizationMethodNotFound|WiringReusesCoordinators)' -count=10
PASS  go test ./internal/identity ./cmd/han-server -count=1
PASS  go vet ./internal/identity ./cmd/han-server
PASS  gofmt -l -- <五个固定文件>（无输出）
PASS  git diff --check -- <五个固定文件>（无输出；文件为 untracked，不能单独作为内容证据）
PASS  对五个固定文件执行只读 trailing-whitespace / final-LF 扫描
PASS  branch=main；HEAD=6da8abbceb0fa197364810a92a8fa2d97b38ddca
PASS  结束前五文件与 Planning Review SHA256 全部匹配固定值
```

`git diff --no-index --check NUL <file>` 另仅产生 Git 的 LF→CRLF 工作区提示，未报告 whitespace error；未据此改写文件。

## 6. 未验证边界

- 未运行真实 PostgreSQL 18、真实多连接/commit-ack-loss、migration 混跑。
- 未运行真实 SMTP/TLS/AUTH、provider accepted/unknown/rejected。
- 未部署、未运行 Browser/E2E、未验证外部收件。
- 未做容量/性能基准；F001 的 Argon2/锁占用影响为代码路径推导，未量化。
- 未验证 production wall clock 跨秒的外部 HTTP 实测；F002 由 `SystemClock`、两次取时、持久快照字段和 RFC3339 formatter 的确定性代码证据得出。
- test-plan r20 是 planned-not-executed，仅作为验收与风险依据。

## 7. Gate

**Code Review Gate: CHANGES REQUIRED**

三个 MEDIUM finding 均未关闭。F001/F002 还显示当前五文件写集可能不足；在完成文件所有权对账并由作者修复、补齐普通/race 回归、固定新 revision 后，需由非作者 Reviewer 定点复审。未发现需要 Reviewer 自行修改公共合同或架构的授权。
