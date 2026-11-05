# identity-auth-S11.1 Code Review R02 定点复审

- Reviewer：Copernicus（原 R01 非作者 Reviewer）
- 日期：2026-10-25
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：R01 F001、F002、F003 均 **CLOSED**；新增 1 项 **MEDIUM** finding。
- **Code Review Gate: CHANGES REQUIRED**

## 1. 固定对象与输入

### 1.1 六文件代码 revision

| 对象 | SHA256 |
|---|---|
| `internal/identity/password_reset.go` | `F1BB31B3726E19F100184C80AC517243584F6A9299995C682DAB64285D80C439` |
| `internal/identity/handler.go` | `29961F44AD74A8380C5DF8187359456066EF729666F7DF4ED5AE776D5AAFCD18` |
| `internal/identity/temporary_credential_resend.go` | `D712E504B0D9310DEFBD88299FADE4EE61AD6F09BD2937D9F8F838EEAA56FAAA` |
| `internal/identity/temporary_credential_resend_test.go` | `E36FCE8C394EAF182873876FBC3FE3323B232032A4F3D0AEA06D1D903B76700F` |
| `cmd/han-server/main.go` | `1895BAE07E222004CE5191779A0880C008B6B00A53C039826B58AA331D2241C8` |
| `cmd/han-server/main_test.go` | `4EA1BDE6B6FAF4B79028CE0816130AE2E92545570C3A6C4790E29AA2353EF861` |

开始、落报告前均重新计算以上哈希并匹配；六文件均为共享工作树中的 untracked 文件，未以空 Git diff 代替全文与定向检查。

### 1.2 复审输入

| 对象 | SHA256 |
|---|---|
| `.spec/wip/identity-auth/review-code-s11.1-r01.md` | `BE60E271B0A294199E4D3F8DC378F3FD3E218F6BB8EE03A89636C38A89C12103` |
| `.spec/wip/identity-auth/review-tasks-plan-s11.1-r02.md` | `C7F07124A2441D657956AA074289A425EADBFAC2774613FD6349B43CF3C96CFA` |
| `.add/project/knowledge/tasks/tasks.md` | `F0C07FC4CEDA2532470694FDE1EF7F077CAC9A6F6383896CA8F209C5500967C0` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `6BA503907E0BDB03228BE4F0D0D95C069DCCC248B92CD97F9A4F5AB89819B7F7` |

依据还包括 identity design r44、stories r16、test-plan r20，以及 Reviewer Role、`add-code-review`、代码评审/抽象/coding/no-hardcode/observability/tracing/security 与 go-service 适用规则。

## 2. 两轮实际检查范围

### Round 1：原 findings 与六文件直接实现

- 全文/同 revision 复用检查六文件；对本轮变化重点逐行检查：
  - `password_reset.go:100-117,180-309,675-685,760-791`
  - `temporary_credential_resend.go:19-287,314-715,805-898`
  - `temporary_credential_resend_test.go:149-784,858-1018,1036-1333`
  - `main_test.go:4732-5052,5748-6081`
- 核对 same-key 分类、new-key preflight、secret/ID/hash 顺序、唯一 durable Store/Gate/lineage classifier、两 purpose expiry 事实与 replay 副作用。
- 核对 production 角色矩阵、route-derived purpose、body/header/query 串改、403/envelope 与拒绝路径调用计数。

### Round 2：真实消费者与公共边界回归

- 有界追踪：
  - `password_reset_store.go` 的 `CheckLockedUser`、`BeginOrClassifyCandidate`、shared lineage classifier；
  - `idempotency.NewPasswordResetClaim` 的 `CreatedAt=request.Now`；
  - production builder 中唯一 `PasswordResetStore`、两个既有 coordinator、route/middleware 与 test seam；
  - Handler 的 exact path、Idempotency-Key、成功 envelope。
- 检查无第二 Store/Gate/coordinator/root/global，无新增 migration/schema/公共字段；`handler.go` 与 `main.go` 哈希相对 R01 未变。
- 独立执行精确普通/race、全部 resend 普通/race、相关包普通/race、vet、build、格式与 whitespace 检查。

## 3. R01 findings 定点结论

### 3.1 `IDENTITY-S11.1-REV-R01-F001` — CLOSED

**关闭证据：**

- `password_reset.go:210-227` 先构造 begin identity 并分类 same-key；命中 succeeded/failed/unresolved/hold replay 时在 latest credential、secret、ID、hash 与 Mailer 前返回。
- 新 key 才读取 current credential，并于 `password_reset.go:243-254` 调用 resend-only `AuthorizeNewCandidate`；敏感准备从 `password_reset.go:255` 才开始。
- production preflight 在 `temporary_credential_resend.go:133-169` 复用同一 `PasswordResetStore.CheckLockedUser` 与同一 `passwordResetResendLineageStatus`，最终写入仍由同一 Store 的 `BeginOrClassifyCandidate` 复核；没有复制 SQL、Store 或持久化 authority。
- missing、legal-history stale、consumed、unresolved 测试均断言 secret/ID/hash/Mailer/业务写为零；durable-authority 测试覆盖 Gate、eligible、consumed、legal-history stale，并断言 preflight 无 Exec/write。
- 精确测试普通 `-count=10`、race 与全部 resend 普通/race均通过。

**限制：** 未以真实 PostgreSQL 18 执行 lineage SQL；结论基于共享 classifier 的代码、状态化 stub 与 race 证据。错误映射回归另见新增 finding，不回开本 finding 的“敏感准备顺序”缺陷。

### 3.2 `IDENTITY-S11.1-REV-R01-F002` — CLOSED

**关闭证据：**

- password-reset 首次 credential 使用 `begin.Now`：`password_reset.go:270-276`；durable claim 使用相同 `request.Now`：`internal/idempotency/store.go:222-243`；succeeded replay 使用 `claim.CreatedAt + temporaryCredentialLifetime`：`password_reset.go:287-303,760-778`。
- new-administrator 首次 credential 使用 durable `beginRecord.CreatedAt`：`temporary_credential_resend.go:645-703`；replay 通过既有 record 的 `CreatedAt` 重建同一 expiry。
- 两个 advancing Clock 测试逐对象比较 `replay == first`，并证明 replay 不增加 secret/ID/hash/mail/业务写。
- 未改变公共 response shape、snapshot schema、Store/schema/migration。

### 3.3 `IDENTITY-S11.1-REV-R01-F003` — CLOSED

**关闭证据：**

- `main_test.go:4755-4784` 构造真实 `PasswordResetCoordinator`、真实 `AdministratorCreateCoordinator` 与真实 `TemporaryCredentialResendRouter`；`5812-5945` 将该真实业务服务接入 production composition/router，而非用简单 service stub 冒充授权。
- 矩阵覆盖 member、普通 administrator、initial administrator 与当前有效 administrator：
  - member 两 route 均 403；
  - 普通 administrator 可进入 password-reset authority、不可选择 new-administrator purpose；
  - initial administrator 可进入两类对应 authority。
- 每例同时注入恶意 body/header/query purpose，正向调用轨迹仍由 exact route 决定；拒绝路径断言稳定 envelope、403，以及 secret/ID/hash/mail/业务写全零。
- 原 composition route、Origin/Session/CSRF/AccessGeneral、Idempotency-Key、404/405/Allow、共享 coordinator wiring 测试保持通过；test seam 只承担受控依赖注入和观测，没有进入 production 启动配置。

## 4. 新 findings

### MEDIUM

#### `IDENTITY-S11.1-REV-R02-F001` — unresolved new-key resend 被错误降格为 `identity.invalid_state`

- **位置：**
  - `internal/identity/temporary_credential_resend.go:148-155,191-203,266-270`
  - `internal/identity/temporary_credential_resend_test.go:581-670`
- **违反依据：**
  - design r44 §3.3.12：unresolved operation 或 integrity hold 使用既有 `identity.operation_uncertain`；missing/stale/consumed 才使用既有 invalid-state 映射。
  - design r44 公共错误表：`identity.invalid_state` 为 409 / 状态变化后重试 / 无错误日志；`identity.operation_uncertain` 为 503 / 受控核对 / ERROR。
  - S11.1 要求公共错误语义不变。
- **触发场景：** 同键未命中、当前 old credential/lineage仍可形成 resend context，但目标 User 已有 unresolved password-reset operation。`CheckLockedUser` 返回 `ErrPasswordResetGateClosed`；preflight 将其统一改为 `errTemporaryCredentialResendContextUnavailable`，外层再统一映射为 `accountStatusInvalidStateError()`。
- **影响：**
  - durable Gate 仍 fail closed，且不会生成 secret、ID、hash、邮件或业务记录；
  - 但公共结果从应有的 503 `identity.operation_uncertain` 变成 409 `identity.invalid_state`，丢失“等待受控核对”的重试/责任/日志语义，客户端可能把 unresolved 当作普通终态上下文无效并持续换 key。
- **测试缺口：** `TestTemporaryCredentialResendPasswordResetRejectsMissingOrStaleContext` 对所有 case 接受 `identity.invalid_state` **或** `identity.operation_uncertain`，因此无法捕捉上述稳定错误语义回退。
- **建议：** 在继续复用现有 Gate/lineage authority且不新增公共 code、Store或 classifier的前提下，精确保留 unresolved/hold → `identity.operation_uncertain`、missing/stale/consumed → `identity.invalid_state`；最终 Store 复核不变。测试应逐 case 断言唯一公共 code/HTTP 状态，不再接受二选一。
- **复审条件：**
  1. new-key unresolved 普通/race测试精确断言 503 / `identity.operation_uncertain`；
  2. missing/stale/consumed 分支精确断言既有 invalid-state 映射；
  3. 各分支仍断言零 secret/ID/hash/mail/claim/operation/credential/job 写入；
  4. 六文件新 revision 固定后由非作者 Reviewer 定点复审。

### HIGH / CRITICAL / LOW

无新增 HIGH、CRITICAL 或独立 LOW finding。

## 5. 架构、范围与回归结论

- 仍只有 production builder 在 `cmd/han-server/main.go:2088` 构造一个 `PasswordResetStore`；Login、PasswordChange、Session、operator、worker、首次 reset 与 resend 继续消费该共享实例。
- resend router只复用既有 `PasswordResetCoordinator` 与 `AdministratorCreateCoordinator`；未发现第二 coordinator/Store/Gate/root/global。
- 两条 exact POST route、Origin → Session/CSRF → AccessGeneral、Idempotency-Key、成功 envelope、404/405/Allow 与 secret safety 未回退。
- 未发现第七个 S11.1 实现文件、migration/schema 或公共 DTO/错误 code 变更。共享工作树存在大量其他未提交内容，本结论只针对固定六文件及其直接定义/消费者。

## 6. 实际执行命令与结果

以下命令均未读取或使用 `HAN_TEST_POSTGRES_DSN`，未连接真实 PostgreSQL/SMTP：

```text
PASS  go test ./internal/identity -run 'TestTemporaryCredentialResend(PasswordResetRejectsMissingOrStaleContext|PasswordResetPreflightUsesDurableAuthority|SucceededReplayPreservesPasswordResetExpiry|SucceededReplayPreservesAdministratorExpiry)$' -count=10
PASS  go test -race ./internal/identity -run 'TestTemporaryCredentialResend(PasswordResetRejectsMissingOrStaleContext|PasswordResetPreflightUsesDurableAuthority|SucceededReplayPreservesPasswordResetExpiry|SucceededReplayPreservesAdministratorExpiry)$' -count=1
PASS  go test ./cmd/han-server -run '^TestProductionTemporaryCredentialResendBusinessAuthorizationAndPurposeIsolation$' -count=10
PASS  go test -race ./cmd/han-server -run '^TestProductionTemporaryCredentialResendBusinessAuthorizationAndPurposeIsolation$' -count=1
PASS  go test ./internal/identity ./cmd/han-server -run 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1
PASS  go test -race ./internal/identity ./cmd/han-server -run 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1
PASS  go test ./internal/identity ./cmd/han-server -count=1
PASS  go test -race ./internal/identity ./cmd/han-server -count=1
PASS  go vet ./internal/identity ./cmd/han-server
PASS  go build ./cmd/han-server
PASS  go test ./internal/identity ./cmd/han-server -list 'TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend'
PASS  gofmt -l -- <六个固定文件>（无输出）
PASS  git diff --check -- <六个固定文件>（无 whitespace error；untracked caveat）
PASS  六文件只读 trailing-whitespace / final-LF 扫描
PASS  staged 为空
PASS  开始及落报告前 branch、HEAD、六文件 SHA256 复核
```

`go build` 后根目录 `han-server.exe` 保持存在；按禁止删除/清理要求未处理该文件。

## 7. 未验证边界

- 未运行真实 PostgreSQL 18、真实 transaction/lock/constraint、commit-ack-loss 或 migration 混跑；preflight lineage 仅由共享 SQL/分类代码与 stub 证明。
- 未运行真实 SMTP/TLS/AUTH/provider accepted/unknown/rejected，未验证外部收件。
- 未部署，未运行 Browser/E2E/真实 Host、容量或 production readiness。
- 未独立重跑 Lead 的全仓 Manifest 七项；本次独立证据限定为相关包普通/race、vet、build与静态检查。
- test-plan r20 仍是 planned-not-executed，只作为验收和风险依据。

## 8. Gate

R01 三项 MEDIUM 均已按各自原始复审条件关闭，但新增 `IDENTITY-S11.1-REV-R02-F001` 仍为 OPEN。

**Code Review Gate: CHANGES REQUIRED**
