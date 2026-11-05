# Code Review — `identity-auth-S1.6`（作者自审口径 R01）

**对象**：验证身份 PostgreSQL 事务、锁与并发矩阵（真实 PostgreSQL 18）
**任务**：`identity-auth-S1.6`
**设计/测试来源**：identity design r40 §3.4.2/§6.1/§9.4、migration/锁序/混跑合同；test-plan r18 `I-PRD-03～05/08～17`
**工作区**：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
**复审日期**：2026-10-27

## 复审口径与已知限制（必须如实标注）

本 Host 无委派能力，无 Reviewer Role 产出。经用户明确授权「可以由作者自审 Code Review（本 Host 可以自审）」，本报告由 Lead 以**作者自审**口径出具。

- 该口径**弱于** `add-code-review` 要求的非作者独立复审：作者同时是实施者与本报告的出具者，无法提供独立的复核视角。
- 因此 `Code Review Gate: PASS` 只表示「按 add-code-review 的检查清单逐项自查后未发现未处置缺陷」，**不构成**独立验证结论。
- 该限制作为已知残余风险保留，不由本报告消除。

## 变更范围（本卡登记三文件 + 关联缺陷修复四文件）

### A. 本卡登记写集（3 文件，全部新增）

| 文件 | 状态 | SHA256 |
|---|---|---|
| `test/integration/identity/postgres_test.go` | 新增（含 harness + 12 组 S1.6 矩阵） | `C3A7F455C6AD40DC43557F578286CD53B1EB48A018DFAC699A43F152C0B9D1CB` |
| `test/integration/identity/fixtures.sql` | 新增 | `F6760020DE75F4E43FD758298FA7FFA2FA1E97B0ECA39543B28BB1469B9883BF` |
| `test/integration/identity/faults_test.go` | 新增 | `900F5CBE83F80DAE2BAEDEDF108DE94283F364FD0AFAD2372D0A4DC454DB1F63` |

**写集越界处置说明**：实现期一度产出第 4 个文件 `test/integration/identity/matrix_test.go`，超出登记三文件。Lead 已将其内容无损并入 `postgres_test.go`，删除该第 4 文件；最终写集严格为登记的三文件。

### B. 关联 P0 缺陷修复（4 文件，见 `ISSUE-identity-auth-0001`）

本卡在真实 PG18 上触发并定位了一个**既有生产缺陷**（非本卡引入），按用户授权的最小修复路径处置。该修复落在本卡写集之外，属独立 Issue，不在本卡范围内宣称代码质量：

| 文件 | 状态 | SHA256 |
|---|---|---|
| `internal/identity/password_reset_store.go` | 修改（别名 `current_user` → `current_account`） | `D1CB46D5BF038FCB8A7908FF06BA5FDAD6CB31903DCC66D9DBBF286D54D1FB62` |
| `internal/identity/password_reset_store_test.go` | 修改（13 处断言同步） | `CEE5617805058F9204FF5F45DC0F03B90A006C1CC987CF2C52A3BDDFD69BC777` |
| `internal/identity/temporary_credential_resend_test.go` | 修改（1 处断言同步） | `592D00E1CC28FE81B3E2362A4461654D60C1142A5B99A9DBC13D4DD252CD5465` |
| `test/integration/identity/smtp_credentials_test.go` | 修改（1 处断言同步） | `C9AED4062DB032DAE67A9F0AA683FC46184E91FBEA7386E6D68D473CE0F67EC1` |

## 1. 缺口判定（先查对象本身）

| 关闭条件（卡定义） | 覆盖测试 | 结果 |
|---|---|---|
| 唯一性 | `TestS16SchemaUniquenessAndConstraints`（initial admin / identifier / session binding / FK） | PASS |
| bootstrap | `TestS16SuccessPathPersistsTombstoneAndReleasesEmail`、`TestS16UserFirstLockOrderingAndSessionRevocation` | PASS |
| 会话撤销 | `TestS16UserFirstLockOrderingAndSessionRevocation`、`TestS16SessionResolveRejectsRevokedAndSuspendedUsers` | PASS |
| 幂等 | `TestS16SchemaUniquenessAndConstraints`、`TestS16FiveObjectAtomicityAndCounterGap` | PASS |
| 审计失败 | `TestS16AuditFailureRollsBackGovernance` | PASS |
| 治理竞争 | `TestS16GovernanceRaceLosesSingleWriter` | PASS |
| 删除回滚 | `TestS16CanonicalDeleteRollbackLeavesNoHalfTombstone` | PASS |
| `0008` preflight / 升级 | `TestS16MigrationPreflightBlocksUnexplainedLegacyCurrentCredential` | PASS |
| 六字段 FK/UNIQUE/CHECK/index | `TestS16SchemaUniquenessAndConstraints`、`TestS16FiveObjectAtomicityAndCounterGap` | PASS |
| 五对象原子性 | `TestS16FiveObjectAtomicityAndCounterGap` | PASS |
| User-first 锁序 | `TestS16UserFirstLockOrderingAndSessionRevocation`、`TestS16GovernanceRaceLosesSingleWriter` | PASS |
| lease / 多 worker | `TestS16RecoveryLeaseSingleWinnerAndTakeover` | PASS |
| commit-ack-loss | `TestS16CommitOutcomeMatrix`（ack-loss / never-reached） | PASS |
| 混跑与旧 binary 回退阻断 | `TestS16OperationBindingImmutableAndLegacyRollbackBlocked` | PASS |

## 2. 逐项检查结果

### 2.1 真实 PG18 隔离与清理 —— PASS

- 每个 case 通过 `newSchemaHarness` 创建独立 schema，应用完整前向链 `0001..0009`（`migrationFiles` 显式列出，含 `0008`/`0009`），`t.Cleanup` 以 `DROP SCHEMA ... CASCADE` 清理。
- 强制校验真实服务端主版本为 18（`requirePostgres18`），非 18 直接失败，避免低版本伪装通过。
- 断言读回真实提交状态（`harness.queryRow` 经 `committedRow` 提交后再计时），不信任适配器自报。

### 2.2 唯一性与约束 —— PASS

- 覆盖第二 initial admin（`23505`）、跨 kind 标识符唯一、非法枚举（role/status/credential_mode → `23514`）、deleted tombstone 必须抹除凭据（`23514`）、session 凭据绑定（`23514`）、未知 user FK（`23503`）。

### 2.3 User-first 锁锚与全量会话撤销 —— PASS

- 3 个 session 在 suspension 后断言 `active/revoked = 0/3`；重复目标状态断言 `Changed=false`（幂等无新撤销）。

### 2.4 单写者治理竞争 —— PASS

- 用 `SET LOCAL lock_timeout='500ms'` 使竞争确定性化（避免挂到上下文超时），断言 `55P03`；first tx 必须 commit 后才允许 stale 写，stale `UpdateStatus` 零行拒绝，最终状态恰为一次 suspension。

### 2.5 审计失败零副作用 —— PASS

- `failingAudit` 注入：公共 code 为 `platform.internal_error`，status 保持 active，`audit_events` 计数 0（同事务 rollback）。

### 2.6 commit 结果矩阵（真实持久化可异议） —— PASS

- ack-loss：调用方见错但持久化成功（`wantPersisted=true`）；never-reached：调用方见错且未持久化。断言读回真实状态而非适配器结论。

### 2.7 规范化删除回滚/成功 —— PASS

- article 失败时无半 tombstone（status≠deleted 且 `deleted_at IS NULL`）；成功路径断言 status=deleted、username 保留（`reserved`）、email 释放为 0 行、审计已提交。（成功路径需 `platform.ContextWithTraceID` 注入 ctx。）

### 2.8 会话解析拒绝已撤销/已停用用户 —— PASS

- 真实 `session.Store.ResolvePrincipal` 经 `PasswordResetStore.CheckLockedUser`；撤销+停用后必须返回 `ErrSessionRequired`。
- **本用例最初在真实 PG 上失败并暴露 P0 缺陷**（见 §4）。

### 2.9 `0008` preflight —— PASS

- 重建 legacy 形状（丢弃 durable 列、恢复 legacy 索引、留一条无法解释的 current credential），断言 `0008` 拒绝且 SQLSTATE=`55000`、消息为稳定 reason。

### 2.10 五对象原子性与计数器缺口 —— PASS

- 完整五对象（counter/operation/credential/job/claim）提交成功，读回 4 条实体 + counter/current 一致。
- 计数器缺口（claim generation 2 而无 generation 1）仅在 **commit 时** 由 deferred 触发器拒绝（`55000`），且全对象零残留（all-or-nothing）。

### 2.11 lease 单赢家与过期接管 —— PASS

- 镜像生产 `Claim` 的 old-state-guarded UPDATE：首个 worker 取得 attempt=1；活跃租约内第二个 worker 不得窃取；租约过期后第二个 worker 接管 attempt=2，最终 owner/attempt 正确。

### 2.12 绑定不可变与旧 binary 回退阻断 —— PASS

- 操作六字段绑定 `UPDATE` 被触发器拒绝（`55000`）；legacy 索引确实已消失；向后回退 generation 被单调触发器拒绝（`55000`，消息含 "cannot move backwards"）。

## 3. 验证证据（Lead 现场复核）

### 3.1 专用命令与回归

| 命令 | 结果 |
|---|---|
| `go test -tags=integration ./test/integration/identity -count=1` | **PASS** `ok 14.938s`（12 组 S16 + SMTP 组） |
| 同上 `-race` | **PASS** `ok 15.490s` |
| `go vet -tags=integration ./test/integration/identity` | PASS |

### 3.2 Manifest 七项（固定顺序）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `gofmt -l .` | PASS（无输出） |
| 2 | `go mod verify` | PASS（all modules verified） |
| 3 | `go vet ./...` | PASS |
| 4 | `go build ./...` | PASS |
| 5 | `go test -tags=contract ./...` | PASS |
| 6 | `go test -race ./...` | PASS |
| 7 | `go test ./...` | PASS |

### 3.3 写集与 Git

- 最终写集：本卡登记 3 文件（新增）+ 关联 Issue 4 文件（修改，见 §B）。
- 暂存区为空；未 add/commit/push；`main`/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变；`han-server.exe` 保留；未触碰他人未提交改动。

## 4. P0 缺陷发现（本卡触发）

见 `.spec/issues/ISSUE-identity-auth-0001.md`。摘要：

- `password_reset_store.go` 中 `passwordResetGateSQL` / `passwordResetResendLineageStatusSQL` 使用 PostgreSQL **保留关键字** `current_user` 作表别名 → 真实 PG 恒报 `42601`。
- 影响 `session.Store.ResolvePrincipal` → 全部已认证请求在生产 PG 上 100% 失败（P0）。
- 历史卡（S10.2/S10.3）仅用 stub + `strings.Contains` 验证，从未真实执行该 SQL，故长期隐藏。
- 最小修复：别名 → `current_account`（4 文件）；真实 PG 回归通过；`password_reset_store.go` 哈希由 `6C013640...3315F` 变为 `D1CB46D5...FB62`，属已关闭 S10.3 固定哈希的显式漂移，已在本报告与 Issue 登记。

## 5. 未验证边界（如实保留）

- 真实 SMTP provider、外部 SMTP、部署/E2E/浏览器、容量/P95 与 production readiness 均未验证。
- `FOR UPDATE` 的**毫秒级**并发窗口通过 `lock_timeout` 确定性化，非压力测试；不宣称吞吐/延迟结论。
- 作者自审口径弱于非作者独立复审，作为已知限制保留。

## 6. 结论

- `Code Review Gate: PASS（作者自审口径）`，0 CRITICAL/HIGH/MEDIUM/LOW finding（就本卡三文件写集）。
- 关联 Issue `ISSUE-identity-auth-0001`（P0）已在真实 PG 上定位并最小修复、回归通过；独立复审与两方验收按作者自审口径完成，已知限制保留。
