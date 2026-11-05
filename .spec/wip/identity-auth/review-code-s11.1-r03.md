# identity-auth-S11.1 Code Review R03 定点复审

- Reviewer：Copernicus（原 R01/R02 非作者 Reviewer）
- 日期：2026-10-25
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：`IDENTITY-S11.1-REV-R02-F001` **CLOSED**；R01 F001/F002/F003 均保持 **CLOSED**；无新增 finding。
- **Code Review Gate: PASS**

## 1. 固定对象与输入

### 1.1 六文件代码 revision

| 对象 | SHA256 |
|---|---|
| `internal/identity/password_reset.go` | `F1BB31B3726E19F100184C80AC517243584F6A9299995C682DAB64285D80C439` |
| `internal/identity/handler.go` | `29961F44AD74A8380C5DF8187359456066EF729666F7DF4ED5AE776D5AAFCD18` |
| `internal/identity/temporary_credential_resend.go` | `EF7F9CBC121A18040BAA937E9217C53CE961C7072166019AB6863B4B1E9AD50C` |
| `internal/identity/temporary_credential_resend_test.go` | `B579E97953DD258E5B98B75778879C092206053A65609595BFBF33C859372E68` |
| `cmd/han-server/main.go` | `1895BAE07E222004CE5191779A0880C008B6B00A53C039826B58AA331D2241C8` |
| `cmd/han-server/main_test.go` | `4EA1BDE6B6FAF4B79028CE0816130AE2E92545570C3A6C4790E29AA2353EF861` |

开始、写报告前及写报告后均重新计算六文件哈希；均匹配以上固定值。六文件均为共享工作树中的 untracked 文件，本报告未以空 Git diff 代替内容检查。

### 1.2 复审输入

| 对象 | SHA256 |
|---|---|
| `.spec/wip/identity-auth/review-code-s11.1-r02.md` | `07B0EA307519770737B8C17B87D12849B4A725D03CD66915EA466D884F8FA1B2` |
| `.spec/wip/identity-auth/review-code-s11.1-r01.md` | `BE60E271B0A294199E4D3F8DC378F3FD3E218F6BB8EE03A89636C38A89C12103` |
| `.spec/wip/identity-auth/review-tasks-plan-s11.1-r02.md` | `C7F07124A2441D657956AA074289A425EADBFAC2774613FD6349B43CF3C96CFA` |

还读取并适用 Reviewer Role、`add-code-review`、代码评审清单、抽象/coding/no-hardcode/observability/tracing、Go service coding/security 规则及产品安全红线；沿直接消费者有界读取 `password_reset_store.go`、公共错误注册、handler、production composition/router 与对应测试。

## 2. 两轮实际检查范围

### Round 1：R02 finding 与修复文件直接检查

- 全文检查 `temporary_credential_resend.go` 和 `temporary_credential_resend_test.go`，重点逐行检查：
  - `temporary_credential_resend.go:117-223,248-305`
  - `temporary_credential_resend_test.go:581-849,924-1084`
- 核对 new-key preflight 的 lineage 分类、Gate 分类、最终 authority 复核及外层错误映射。
- 核对 missing/stale/consumed、integrity drift、unresolved Gate、same-key integrity hold 的稳定公共 code/HTTP 状态。
- 核对拒绝路径的 secret、ID、hash、Mailer、candidate begin、持久业务写和 commit 计数。
- 独立执行精确普通/race测试、expiry replay 回归及 production 授权矩阵回归。

### Round 2：真实 authority、消费者与边界回归

- 有界追踪同一 `PasswordResetStore` 的 `CheckLockedUser`、`BeginOrClassifyCandidate`、共享 resend lineage classifier。
- 复核 `password_reset.go:180-307,760-791` 的 same-key 分类、preflight-before-sensitive-preparation、最终 durable authority 与 replay expiry。
- 复核 handler 的 exact route-derived purpose、Idempotency-Key、稳定 envelope；复核 `cmd/han-server` 唯一 production composition、真实 coordinator 授权矩阵和共享实例测试。
- 静态搜索确认仍只有一个 `PasswordResetStore` 类型、一个 resend authority、一个 resend service builder 接入点；未发现第二 authority/Store/Gate/classifier/composition root/global。
- 执行全部 resend 定向普通/race、相关包 vet、六文件格式与 whitespace 检查。

## 3. 原 findings 定点结论

### 3.1 `IDENTITY-S11.1-REV-R02-F001` — CLOSED

**代码证据：**

- `temporary_credential_resend.go:148-169` 先使用既有共享 lineage classifier：
  - `ineligible` 返回既有 `errTemporaryCredentialResendContextUnavailable`；
  - lineage drift 返回 `identity.operation_uncertain`；
  - lineage eligible 后，同一 `PasswordResetStore.CheckLockedUser` 的 Gate closed 返回 `identity.operation_uncertain`。
- `temporary_credential_resend.go:193-221` 在最终 `BeginOrClassifyCandidate` 前后继续复核同一 Gate/authority；Gate 在窗口内变为 unresolved 时保持 fail closed 并返回 `identity.operation_uncertain`。
- `temporary_credential_resend.go:284-289` 只把明确的 context-unavailable sentinel 映射为 `identity.invalid_state`，不会把上述 public uncertain 错误降格。
- 公共错误注册仍为：
  - missing/stale/consumed/ineligible → HTTP 409 / `identity.invalid_state`；
  - integrity drift/hold/unresolved Gate → HTTP 503 / `identity.operation_uncertain`。

**测试证据：**

- `temporary_credential_resend_test.go:581-707` 对 missing predecessor、legal-history stale、consumed predecessor 精确断言 409 / `identity.invalid_state`，对 unresolved Gate、integrity drift、same-key hold 精确断言 503 / `identity.operation_uncertain`，不再接受二选一错误码。
- 同一测试逐 case 断言 Mailer、commit、candidate begin/write、secret generator、ID generator、hasher 全部为零。
- `temporary_credential_resend_test.go:709-849` 实际调用 production `passwordResetResendAuthority.AuthorizeNewCandidate`，经过共享 lineage 查询与 `PasswordResetStore.CheckLockedUser`，分别证明 unresolved Gate 和 integrity drift 为 uncertain、consumed/stale 为 context unavailable，并断言 delegate begin/write及 SQL Exec 为零。
- 上述精确测试普通 `-count=10`、race，以及全部 resend 定向普通/race均通过。

因此 R02 指出的“unresolved new-key resend 被降格为 invalid_state”已按原复审条件关闭。

### 3.2 `IDENTITY-S11.1-REV-R01-F001` — 保持 CLOSED

- `password_reset.go:210-254` 在 same-key 分类后、secret/ID/hash 前完成 current predecessor、lineage 与 Gate preflight。
- 本轮新增精确错误语义未改变该顺序；拒绝测试继续证明 secret/ID/hash/mail/持久业务写为零。
- `password_reset.go` 哈希与 R02 已关闭 revision 相同。

### 3.3 `IDENTITY-S11.1-REV-R01-F002` — 保持 CLOSED

- password-reset 与 new-administrator 两个 purpose 仍从 durable claim/record `CreatedAt` 重建同一 expiry。
- 两个 advancing Clock 的首次成功与 same-key replay 测试均逐结果比较相等，并断言 replay 不增加 secret/ID/hash/mail或业务写。
- `password_reset.go` 未漂移，相关精确普通/race回归通过。

### 3.4 `IDENTITY-S11.1-REV-R01-F003` — 保持 CLOSED

- `handler.go`、`main.go`、`main_test.go` 哈希均与 R02 已关闭 revision 相同。
- production 测试仍经过真实 resend service/coordinator，覆盖 member、普通 administrator、initial administrator/current administrator 的角色矩阵及 body/header/query purpose 串改。
- 拒绝路径保持 403/稳定 envelope和零 secret/ID/hash/mail/业务写；exact route、Origin/Session/CSRF/Access、Idempotency-Key、404/405/Allow 与共享 coordinator wiring 未回退。

## 4. 新 findings

### CRITICAL / HIGH / MEDIUM / LOW

无新增 finding。

未发现需要第七实现文件、migration、公共合同、architecture/ADR 变化，亦未发现第二 authority/Store/Gate/classifier/root/global；停止条件未触发。

## 5. 实际执行命令与结果

以下命令均未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`，未连接真实 PostgreSQL/SMTP：

```text
PASS  go test ./internal/identity -run '^(TestTemporaryCredentialResendPasswordResetRejectsMissingOrStaleContext|TestTemporaryCredentialResendPasswordResetPreflightUsesDurableAuthority)$' -count=10
PASS  go test -race ./internal/identity -run '^(TestTemporaryCredentialResendPasswordResetRejectsMissingOrStaleContext|TestTemporaryCredentialResendPasswordResetPreflightUsesDurableAuthority)$' -count=1
PASS  go test ./internal/identity -run '^(TestTemporaryCredentialResendSucceededReplayPreservesPasswordResetExpiry|TestTemporaryCredentialResendSucceededReplayPreservesAdministratorExpiry)$' -count=10
PASS  go test -race ./internal/identity -run '^(TestTemporaryCredentialResendSucceededReplayPreservesPasswordResetExpiry|TestTemporaryCredentialResendSucceededReplayPreservesAdministratorExpiry)$' -count=1
PASS  go test ./cmd/han-server -run '^TestProductionTemporaryCredentialResendBusinessAuthorizationAndPurposeIsolation$' -count=10
PASS  go test -race ./cmd/han-server -run '^TestProductionTemporaryCredentialResendBusinessAuthorizationAndPurposeIsolation$' -count=1
PASS  go test ./internal/identity ./cmd/han-server -run 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1
PASS  go test -race ./internal/identity ./cmd/han-server -run 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1
PASS  go test ./internal/identity ./cmd/han-server -list 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend'
PASS  go vet ./internal/identity ./cmd/han-server
PASS  gofmt -l -- <六个固定文件>（无输出）
PASS  git diff --check -- <六个固定文件>（无 whitespace error；untracked caveat）
PASS  六文件只读 trailing-whitespace/final-LF 扫描
PASS  branch/HEAD、三份输入报告及六文件 SHA256 在开始、写报告前、写报告后复核
PASS  staged 为空
```

未重复 Lead 的全仓 Manifest 七项；本次独立证据限定为直接受影响的 identity/server 定向普通/race、vet及静态检查。

## 6. 未验证边界

- 未运行真实 PostgreSQL 18、真实事务锁/约束、多连接竞态、commit-ack-loss或 migration 混跑；lineage 与 Gate 结论基于共享 SQL/classifier 代码、状态化 stub及 race测试。
- 未运行真实 SMTP/TLS/AUTH/provider accepted/unknown/rejected，未验证外部收件。
- 未部署，未运行 Browser/E2E、真实 Host、容量或 production-readiness 验证。
- test-plan r20 仍为 planned-not-executed，仅作为验收与风险依据。
- 未执行无必要的全仓测试；Lead 声明的全量 Manifest 结果未被本报告冒充为 Reviewer 独立执行证据。

## 7. 写入与越界核对

- 实际写入仅：`.spec/wip/identity-auth/review-code-s11.1-r03.md`。
- 未修改代码、Tasks、Plan、设计、旧报告或其他文件；未 stage/commit/push/切分支，未删除、移动、清理文件，未启动 S10.5。
- 共享工作树中的既有改动均保留。

## 8. Gate

`IDENTITY-S11.1-REV-R02-F001` 已关闭，R01 F001/F002/F003 均保持关闭，且本轮未发现新的阻断项。

**Code Review Gate: PASS**
