# Code Review — `identity-auth-S6.1`（作者自审口径 R01）

**对象**：实现其他管理员查询、停用、恢复、删除与撤权
**任务**：`identity-auth-S6.1`（B8A）
**设计/测试来源**：identity Story S06（`design.md` §3.3.7、§3.4.3、§3.4.6）；test-plan I-ADM-04～05
**工作区**：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
**复审日期**：2026-10-27

## 复审口径与已知限制（必须如实标注）

本 Host 无委派能力，无 Reviewer Role 产出。经用户明确授权「可以由作者自审 Code Review（本 Host 可以自审）」，本报告由 Lead 以**作者自审**口径出具。

- 该口径**弱于** `add-code-review` 要求的非作者独立复审：作者同时是实施者与本报告的出具者，无法提供独立的复核视角。
- 因此 `Code Review Gate: PASS` 只表示「按 add-code-review 的检查清单逐项自查后未发现未处置缺陷」，**不构成**独立验证结论。
- 该限制作为已知残余风险保留，不由本报告消除。

## 变更范围（四文件写集）

| 文件 | 状态 | SHA256 |
|---|---|---|
| `internal/identity/handler.go` | 修改 | `EC4533A6CAA6819255BF9FC7D19EEE3AEEAA90B9E3A9128A5D24B5017D1F1EA3` |
| `internal/identity/administrator_governance.go` | 新增 | `DC43A60525D1A4B7E40587A40EB5EE85183FE4E46F79FE13020DA48AB0C5DBA3` |
| `internal/identity/administrator_governance_test.go` | 新增 | `E64924FF592F1CA25F5158BB248A47530DF1C59E2AB5960B79F6C09C4902A122` |
| `internal/identity/service.go` | **未修改** | `7E5064E40447A35EFC4F6537ED845E6EBD4A7199169208302B63C5905643C70D` |

`service.go` 在任务登记中列出，但本卡复用既有依赖声明（`RegistrationTransactor`、`RegistrationIDGenerator`、`RegistrationClock`、`AccountStatusAuditAppender` 均已在 `account_status.go` / `service.go` 中声明），无需新增或修改；按「外科手术式改动」准则**不留空改动**。越界检查：未触碰四文件之外的任何文件。

## 1. 缺口判定（先查对象本身）

按 route 名称与实际代码逐一核对，`ListAdministrators`、`GetAdministrator`、`SuspendAdministrator`、`RestoreAdministrator`、`DeleteAdministrator` 五个 handler 在本卡开始前**已存在**并已由 S7.1/S8.1/S9.1 关闭链覆盖其查询、状态与删除语义。S6.1 的真实缺口是：

1. **`revoke`（`admin → member` 角色转换）完全缺失**：无 handler、无 service、无路由解析、无审计动作 `identity.role.revoke`。
2. **初始 admin 保护是否在统一 Policy 内执行**：需核对既有 `authorizeAccountStatusActor` 是否已满足 §3.3.7「初始 admin 保护在统一 Policy 中执行，不依赖路由名称」。
3. **撤权后的旧会话即时失效**：既有 `SetAccountStatus` 只在 `suspended` 时撤销会话；撤权必须同样撤销全部目标会话。

## 2. 逐项检查结果

### 2.1 初始 admin 是唯一治理者 — PASS

- `lockAdministratorGovernanceActor` 直接复用 `accountStatus.go` 的**同一** `authorizeAccountStatusActor(principal, actor, RoleAdmin)`：`targetRole == RoleAdmin && !actor.IsInitialAdmin → auth.forbidden`。保护由统一 Policy 承担，**未**按路由名重复实现，符合 §3.3.7 与 §3.4.6 权限矩阵「查看/治理其他管理员：仅初始 admin 允许」。
- actor 侧还复核了 `principal` 与 DB 事实一致性（role/status/initial 标志/`DeletedAt`/`RecoveryRequired`/`CredentialMode == permanent`/`OnboardingCompletedAt`），防止陈旧 Principal 越权。
- 用例覆盖：`TestAdministratorGovernancePermissionMatrix`（普通管理员撤权他人 → `auth.forbidden`）与 `TestAdministratorGovernanceActorEligibilityMatrix`（8 类失效 actor → `auth.forbidden`，且零副作用）。

### 2.2 initial admin 永久受保护 — PASS

- 目标侧 `lockAdministratorGovernanceTarget` 在任何写入前判定：`target.IsInitialAdmin → identity.initial_admin_protected`（复用 `accountStatusInitialAdminProtectedError`，与既有停用/删除路径同一 code，未新增并列公共 code）。
- 「不得为同一原因并列两个公共 code」得到满足：越权用 `auth.forbidden`，初始 admin 受保护用 `identity.initial_admin_protected`，二者互斥。
- 自治理拒绝：`TargetID == actorID → auth.forbidden`，与 §3.4.6「不能是自己」一致。
- 用例：`initial administrator target is protected`、`initial administrator cannot revoke self`。

### 2.3 旧会话即时失效 — PASS

- `persistAdministratorRevocation` 在**同一事务**内：`UpdateRole(admin→member)` → `RevokeSessions(全部目标未撤销会话, reason="role_revoked")` → `Append(identity.role.revoke)`；任一失败即 rollback，不留部分状态。
- 两层保证即时失效：(a) Session 被持久撤销，无法再解析；(b) Principal 的 role 来自 `ResolvePrincipal` 时的 User 行读取，角色已降为 `member`，即使存在竞态窗口也不会保留 admin 权限。二者叠加使「旧会话不得继续持有旧权限」成立。
- 会话撤销 SQL 复用 `revokeAccountStatusSessionsSQL`（`WHERE user_id=$1 AND revoked_at IS NULL`），未复制或另立第二份 SQL。
- 用例：`TestAdministratorGovernanceRevokeTransitionsTargetToMember` 断言 revoke 调用次数与 reason；真实 PostgreSQL 探针断言 `sessions.revoked_at`/`revoke_reason` 落库。

### 2.4 重复目标状态稳定 / 幂等语义 — PASS

- 按 §3.3.7 表格，本卡明确区分：`suspend`/`restore` 为「目标状态幂等」（由既有 `SetAccountStatus` 承担，本卡不改）；`revoke` **不是**目标状态动作，重复撤权返回 `identity.invalid_state`。
- `revoke` 实现：目标已非 `admin`（即已撤权）→ `identity.invalid_state`；`deleted`/`DeletedAt` 非空 → `identity.invalid_state`；目标不存在或 UUID 非法 → `identity.user_not_found`（不泄露存在性）。
- `UpdateRole` 的 `WHERE id=$1 AND role=$2 AND deleted_at IS NULL` 在 SQL 层再次把「期望旧角色」作为条件，`RowsAffected != 1` 直接报错，使并发下的乐观校验与事务内 `FOR UPDATE` 双保险一致。
- 用例：`repeat revoke is an invalid state`、`ordinary member target is not revocable`、`deleted administrator is terminal`、`missing target is not found`、`malformed target identifier is not found`。

### 2.5 事务与失败后置（零副作用） — PASS

- 结构对齐已关闭 S8.1：`Begin → 锁 actor → 判定 actor → 锁 target → 判定 target → 取 now/auditID → 角色更新 → 会话撤销 → 审计 → Commit`；`defer` 在未 commit 时 rollback。
- `TestAdministratorGovernanceDependencyFailureMatrix` 覆盖 begin 失败、nil tx、actor/target 缺失与锁失败、零时钟、ID 失败/空 ID、角色更新失败、会话撤销失败、审计失败、提交失败共 13 例，每例断言公共 code、lock/role/revoke/audit 调用数与 commit/rollback，证明失败后不继续后续副作用。
- `TestAdministratorGovernanceRequestValidation` 证明请求级拒绝（空 actor、受限 Principal、空 trace）在 Begin **之前**发生，零锁零事务。
- 错误映射：`auth.forbidden` / `identity.initial_admin_protected` / `identity.user_not_found` / `identity.invalid_state` / `platform.internal_error`，全部为 `platform.DefaultRegistry` 既有 code，未新增公共错误码。

### 2.6 审计 — PASS

- 动作 `identity.role.revoke` 与 design §3.4.2 审计动作表逐字一致。
- event 携带 actor（user id + role）、`target_type=user`、`target_id`、`outcome=changed`、`trace_id`、metadata `previous_role=admin`/`target_role=member`；与 §3.4.2「所有高风险结果有 AuditEvent」一致。
- 审计在**同事务**内追加，失败即 rollback（`audit failure` 用例证明角色/会话写入被回滚）。
- 未记录 capability/secret/凭据类字段。

### 2.7 契约表面（handler/接口） — PASS

- `Handler.RevokeAdministrator` 与既有 `SuspendAdministrator` 等保持同一风格：从 ctx 取 Principal 与 trace，缺 Principal → `auth.session_required`，缺 trace → `platform.internal_error`；路径解析失败 → `identity.user_not_found`（不泄露）。CSRF/Origin/Session 由既有 `HTTPAuthorizer` + `protectOrigin` 承担，本卡不重复实现，符合「写请求要求 CSRF」。
- 成功响应 `200 {"user":{"id":...,"role":"member"}}`，与 §3.3.7 `POST .../{id}/revoke | 200 {user}` 一致；**不包含** `changed` 字段（撤权非目标状态动作），与设计表格一致。
- `parseAdministratorRevokePath` 严格要求 `prefix + uuid + suffix`，拒绝尾随段、空目标、非 UUID，避免宽松 Trim 造成路径混淆（与既有 `parseAccountStatusTargetID` 的严格闭合风格一致）。
- 新增 `AdministratorGovernanceService` 接口与 `NewAdministratorGovernanceHandler` 构造器，与既有 `AccountStatusService`/`NewAccountStatusHandler` 形态一致；构造器拒绝 nil service，`NewAdministratorGovernanceService` 用 `nilIdentityDependency` 拒绝 typed-nil（`typed nil repository` 用例覆盖）。

## 3. 验证证据（Lead 现场复核）

### 3.1 RED → GREEN

- RED ①（编译期表达能力）：首次运行 `go test ./internal/identity -run "TestAdministratorGovernance"` 失败 `undefined: administratorGovernanceAuditOutcomeChanged`，证明测试确实引用尚不存在的生产符号；修正为引用既有 `accountStatusAuditOutcomeChanged` 后继续。
- RED ②（行为失败）：`missing_target_is_not_found` → `identity.invalid_state`（应为 `user_not_found`）；`malformed_target_identifier_is_not_found` → 事务未开启（应为请求级拒绝）。修正测试夹具（显式 `pgx.ErrNoRows` / 无对应 rollback 期望）后转 GREEN。
- GREEN：`go test ./internal/identity -run "TestAdministratorGovernance" -count=1` → `ok 0.088s`。

### 3.2 专用命令与回归

| 命令 | 结果 |
|---|---|
| `go test ./internal/identity -run "TestAdministratorGovernance" -count=1` | **PASS** `ok 0.088s` |
| 同上 `-race` | **PASS** `ok 1.105s` |
| `go test ./internal/identity -count=1` | **PASS** `ok 0.187s` |

### 3.3 Manifest 七项（固定顺序）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `gofmt -l .` | PASS（无输出） |
| 2 | `go mod verify` | `all modules verified` |
| 3 | `go vet ./...` | PASS |
| 4 | `go build ./...` | PASS |
| 5 | `go test -tags=contract ./...` | PASS（全包 ok） |
| 6 | `go test -race ./...` | PASS（全包 ok） |
| 7 | `go test ./...` | PASS（全包 ok） |

补充：`go vet -tags=integration ./...` PASS；`git diff --check -- internal/identity` 无输出；`git diff --cached --name-only` 为空（未 add）；HEAD 仍为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

### 3.4 真实 PostgreSQL 18.6 SQL 探针（临时，已删除）

在 `HAN_TEST_POSTGRES_DSN=postgres://postgres:<redacted>@127.0.0.1:5433/postgres?sslmode=disable`（本机真实 PostgreSQL 18.6）上，用临时 `//go:build integration` 探针（文件 `internal/identity/zz_temp_s61_probe_test.go`，**不在**四文件写集内，验证后已删除）在事务级 schema 内验证：

- `PostgresAdministratorGovernanceRepository.LockUser` 正确读回 role/is_initial_admin/status；
- `UpdateRole(admin→member)` 落库为 `role='member'` 且 `status` 不变；
- `RevokeSessions` 使真实 `sessions.revoked_at`/`revoke_reason='role_revoked'` 落库；
- 重复 `UpdateRole(admin→member)`（角色已非 admin）影响 0 行并报错，证明期望旧角色条件在 SQL 层生效。

结果：`--- PASS: TestTempS61AdministratorGovernanceSQLProbe (0.33s)`。探针删除后 `Test-Path` 为 `False`，`go test -tags=integration ./internal/identity -run "TestNothingSelected"` → `ok [no tests to run]`，工作树恢复至四文件状态。

## 4. Findings

**0 CRITICAL / 0 HIGH / 0 MEDIUM / 0 LOW**。

自查中未发现未处置缺陷。以下为**已记录的边界与限制**，非 finding：

1. **非作者独立复审缺失（已知残余风险）**：见「复审口径与已知限制」。作者自审不能替代独立评审。
2. **production 路由未接线**：`/api/v1/admin/administrators/{id}/revoke` 的生产注册位于 `cmd/han-server/main.go`，不在本卡四文件写集内。`ListAdministrators`/`GetAdministrator`/`SuspendAdministrator`/`RestoreAdministrator`/`DeleteAdministrator` **同样**尚未在该 router 注册（既有事实），故本卡与既有治理面保持一致；整片治理路由的 production wiring 属独立 composition 工作的范围，须由后续卡按所有权认领，本卡不越界修改。
3. **`service.go` 未修改**：登记列出但本卡无需改动，已在变更范围表如实记录。
4. **未验证项**：真实 HTTP/浏览器端到端、真实部署、E2E、容量与 production readiness 均未验证；真实 PostgreSQL 探针只覆盖本卡两条新 SQL 的落库行为，不外推并发竞争、回滚物理性与 commit-ack-loss（由 `identity-auth-S1.6` 承接）。

## 5. 结论

`Code Review Gate: PASS（作者自审口径）`

- 初始 admin 是唯一治理者：成立（统一 Policy）。
- 旧会话即时失效：成立（同事务角色降级 + 全会话撤销 + Principal 逐请求重读）。
- 重复目标状态稳定：成立（`revoke` 重复即 `identity.invalid_state`；`suspend`/`restore` 幂等由既有实现承担）。
- initial admin 永久受保护：成立（`identity.initial_admin_protected`，单一 code）。
- 失败证据与 Review：RED→GREEN 保留，专用命令/race/Manifest 七项/真实 PostgreSQL 探针 PASS，本报告完成。

该 PASS 弱于非作者独立复审，作为已知限制保留；不证明部署、E2E 或 production readiness。

---

**2026-10-31 脱敏说明（Lead，按用户确认执行）**：按用户 2026-10-31 确认，本文件中原引用的本机 loopback 合成 DSN 口令已替换为 `<redacted>`（共 1 处）；其余内容与结论未变。脱敏前 SHA256 为 `B76A590923A6F61F1E9E0085B2EA26B6B7DAAA5340DCB46460AAD06C6B2AF944`，脱敏后哈希见 tasks.md 同日记录。
