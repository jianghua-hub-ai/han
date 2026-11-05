# identity-auth-S9.1 Code Review R02（作者自审修订复审）

- 日期：2026-10-26
- Reviewer：主会话 Lead（当前 Host 无法产出非作者 Reviewer）；**依据用户 2026-10-26 明确授权「由作者自审 Code Review（本 Host 可以自审）」**
- objectType：code
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 任务：`identity-auth-S9.1`
- 上游：R01 `.spec/wip/identity-auth/review-code-s9.1-r01.md`（SHA256 `813B8B947935101CA5C313DB77432B984994B3F71FED21360C9B3A67D325167B`，`CHANGES REQUIRED`，F001～F004 全 OPEN）
- 结论：**PASS**（F001～F004 全部 CLOSED，0 新 HIGH/MEDIUM finding）

> 独立性声明：本次复审由实现作者同一会话执行，**不满足 add-code-review 的「非作者独立复审」要求**。用户已明确接受作者自审口径，故本轮结论在原授权范围内成立；其独立性弱于非作者复审，作为已知限制保留。

## 1. 固定对象与漂移核对

复审开始、定向验证后与报告写入前均核对以下五个固定对象 SHA256，无漂移；与 tasks SSOT 记录的返工后值一致。

| 文件 | SHA256（本轮实测） |
|---|---|
| `internal/identity/deletion.go` | `119D378C3508A790FFC6E7371C5AFC0D8CE100A33C2F47AFACD305F7683A2721` |
| `internal/identity/deletion_test.go` | `2F5975867E74EC6B6E8A0AD1DFD128382EC5123FB308A7924A5B6D4294ED6687` |
| `internal/identity/article_port.go` | `F12DB7648F36CD530879434A9B0D0E4231E955253614ADFA5F3999AF40A1ED10` |
| `internal/identity/service.go` | `7E5064E40447A35EFC4F6537ED845E6EBD4A7199169208302B63C5905643C70D` |
| `internal/identity/handler.go` | `FC4B1711F8B25DB8508DAA8DE1ACD87AB38C7B0F1655679768FD64C80304C8FC` |

`article_port.go`、`service.go`、`handler.go` 三个文件相对 R01 基线未变，与 Plan 记录一致。

## 2. 逐条 finding 复审

### F001（原 MAJOR）— 标识墓碑缺失/错绑时仍可提交 → **CLOSED**

- 修复位置：`internal/identity/deletion.go:300-311`（`prepareCanonicalDelete` 内前置校验）、`deletion.go:635-750`（`validateCanonicalDeleteIdentifierTombstones` + 路径 A/B 与逐行/可选行校验）、`deletion.go:1062-1093`（`LockIdentifiers`，含零行）、`deletion.go:1096-1119`（`ReserveIdentifiers` 复算完整性）。
- 核对：`LockIdentifiers` 在任何写入前以 `FOR UPDATE` 锁定并读取该 User 的全部 identifier 行；`validateCanonicalDeleteIdentifierTombstones` 逐 kind 校验存在性、`UserID` 归属、重复行、`NormalizedValue` 与 `users.username/email/phone` 值一致、`ReleasedAt` 为空、lifecycle 属于 `{active,reserved}`；路径 A（未删状态）对 username 必需、email/phone 按 User 列存在性做“存在即值必须一致、缺失即不得有孤儿行”；路径 B（已删状态）只证明存在性与未释放，并对 `retained_email` 做值一致校验。任何缺失/错绑/孤儿/已释放均返回 `CodeIdentityInvalidState`，且发生在所有写入之前。
- 回归：`TestCanonicalDeleteIdentifierTombstoneFailClosed` 覆盖「missing phone row」「phone value drift」「orphan email row」，断言错误码为 invalid_state、状态不变、事务未提交。
- 结论：满足 R01 复审条件（缺 phone 行与值错绑全事务失败）。**CLOSED**。
- 残余限制（延续 design r45 §3.3.10，非本轮 finding）：路径 B 无法证明 phone 的“原值”一致（恢复备份中 phone 原值不落库），只做存在性/未释放校验；「新增 retained_phone 列」已登记为需单独授权的后续候选。

### F002（原 MAJOR）— recovery `kept_deleted` 未绑定维护锁与 RecoveryRun 权威状态 → **CLOSED**

- 修复位置：`internal/identity/deletion.go:43-78`（`RecoveryDeleteCapability`，字段不可导出、`NewRecoveryDeleteCapability` 校验并拒绝空白/未规范化输入）、`deletion.go:132-147`（`RecoveryRunLocker`/`RecoveryOpenGate` 只读接口）、`deletion.go:314-316` 与 `deletion.go:757-797`（`authorizeCanonicalDeleteRecovery` 双重证明）、`deletion.go:1000-1009`（唯一 authority 上的只读 `FOR UPDATE` 锁定读）。
- 核对：`recovery_kept_deleted` 必须携带 `Recovery` 请求且 `Capability.valid()`；`RecoveryRuns`/`RecoveryGate` 缺失即 fail closed；经**同一 identity-owned `RecoveryRunStore`** 的 `LockRecoveryRunForDeletion` 只读锁定 run，要求 `RunID`/`BackupRef` 与能力逐字节一致且 `Outcome == running`，否则非法状态；随后 `CheckRecoveryOpenGate` 必须返回 nil。`LockRecoveryRunForDeletion` 定义在 S9.1 已拥有的 `deletion.go`（复用 `recovery_run_store.go` 的 `selectRecoveryRunForUpdateSQL`），未引入第二 authority、第二 store，也未扩写集为第六文件。
- 回归：`TestCanonicalDeleteRecoveryAuthorizationFailClosed` 覆盖 missing capability / unknown recovery run / run not running / backup mismatch / maintenance gate closed，断言 fail closed、状态不变、pending review 未被完成。
- 结论：满足 R01 复审条件（缺失 run、错 backup、非 running、gate 关闭均拒绝，pending review 不被完成）。**CLOSED**。
- 待决定（非 finding，延续 R01）：`RecoveryOpenGate` 的生产实现属恢复主线/后续部署卡（候选 `identity-auth-S10.5`），本卡只消费并在缺失时 fail closed；未在本卡内装配不构成 S9.1 实现缺陷。

### F003（原 MAJOR）— trace_id 双真相源 → **CLOSED**

- 修复位置：`internal/identity/deletion.go:285-291` 与 `deletion.go:599-626`（`canonicalDeleteTrace`）、`deletion.go:222-264`（拆分后所有子步骤使用同一 `plan.traceID`），并以 `plan.traceID` 贯穿 `applyCanonicalDeleteIdentity`/`appendCanonicalDeleteAudits`/`releaseCanonicalDeleteEmail`。
- 核对：`context.Context` 成为唯一 trace 权威（`platform.TraceIDFromContext`）；缺失或非规范化 ctx trace 一律 `CodePlatformInternalError` fail closed；`request.TraceID` 仅在非空时作为一致性断言，不一致返回 `CodeIdentityInvalidState`，**从不回落**。真实 article consumer 亦从 ctx 读取 trace（`internal/article/command_service.go:172-175`），因此同一 ctx 传入后 identity 与 article 审计使用同一 trace。
- 回归：`TestCanonicalDeleteTraceAuthorityFailClosed` 覆盖「context trace missing」「explicit trace mismatch」，断言禁止删除且状态不变；`deletion_test.go` 其余用例均以 `canonicalDeleteContext(t)`（含 trace 的 ctx）驱动。
- 结论：满足 R01 复审条件（缺 trace / mismatch 在任何写入前 fail closed；一致时同 trace）。**CLOSED**。
- 残余限制：真实 article adapter 的 trace 传播在内存 fake 与 unit 未替代真实跨模块集成；真实 PostgreSQL 跨模块联合删除仍属 S9.2 发布前 Gate（见 §4）。

### F004（原 MINOR，非阻断）— 事务编排函数超过 profile 50 行上限 → **CLOSED**

- 修复位置：`internal/identity/deletion.go:222-264`（`canonicalDeleteInTransaction`）拆为 `prepareCanonicalDelete`（`deletion.go:280-327`）、`applyCanonicalDeleteIdentity`（`deletion.go:365-444`）、`appendCanonicalDeleteAudits`（`deletion.go:448-`）、`completeCanonicalDeleteRecoveryReview`（`deletion.go:331-359`），并引入 `canonicalDeletePlan` 承载锁定值。
- 核对：`canonicalDeleteInTransaction` 主体已收敛至约 43 行，满足 go-service profile ≤50 行；调用顺序与回滚语义保持（prepare → identity → audits → review，任一失败返回错误，最外层 `CanonicalDelete` defer rollback）；diff 未扩出授权五文件。
- 回归：`TestCanonicalDelete_FailureAtEveryOwnedStepRollsBack`、`TestIdentifierLifecycle_EmailReleaseFailureRollsBack`、`TestCanonicalDelete_RecoveryReviewFailureRollsBack` 均通过。
- 结论：**CLOSED**。

## 3. 已确认无新 finding 的关键检查

- 日常删除与 recovery `kept_deleted` 仍复用同一 `CanonicalDeleteCoordinator`，未建立第二删除 authority。
- User/Identifier/Session/TemporaryCredential/Article/Audit/pending review 的写入仍位于同一 caller-owned PostgreSQL 事务；`MarkUserDeleted` 的 `WHERE id=$1 AND NOT is_initial_admin` 保护、`ReserveIdentifiers` 的完整性复算、`verifyCanonicalDeleteIdentifiersSQL` 双条件校验保持。
- `article_port.go` 依赖方向与 caller-owned transaction 未变；identity 未复制 article SQL/状态 authority。
- 未发现 schema/migration 变更、新依赖、硬编码凭据或越界写入。

## 4. 限制与未验证风险

- **独立性**：作者自审，非 add-code-review 要求的非作者复审（用户已授权）。
- 未运行真实 PostgreSQL 跨模块集成（真实 FK/行锁/物理 rollback、article adapter、并发交错）；真实 PostgreSQL 与 production composition/router 装配仍是 S9.2 发布前 Gate，未在本卡内闭环。
- 未运行浏览器、部署或真实 E2E。
- 工作树含他人既有 unstaged/untracked 内容；本报告不评价其正确性。

## 5. Reviewer 实际命令与结果

```text
git branch --show-current   -> main
git rev-parse HEAD          -> 6da8abbceb0fa197364810a92a8fa2d97b38ddca
git diff --cached --name-only -> (空)
Get-FileHash -Algorithm SHA256 <五个固定文件> -> 全部匹配
go test ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle" -count=1        -> ok
go test -race ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle" -count=1  -> ok
go test ./internal/article -run "TestOrphanArticlesCanonical" -count=1                          -> ok
go vet ./internal/identity ./cmd/han-server                                                     -> (无输出)
gofmt -l <五个固定文件>                                                                          -> (无输出)
go build ./... / go vet ./... / go test ./... / go test -race ./...                             -> 全部 PASS
```

未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`（本卡不要求真实数据库）。

## 6. Gate

严重度统计：blocker 0 / major 0 / minor 0。

**Code Review Gate：PASS**（F001～F004 全部 CLOSED，无新增 HIGH/MEDIUM）。

该 PASS 仅覆盖上述固定五文件与 R01 finding 的关闭；不证明真实 PostgreSQL 跨模块、production composition、部署、E2E 或 production readiness。task-level 关闭仍需 Lead 现场核 status/diff/哈希/命令/越界。
