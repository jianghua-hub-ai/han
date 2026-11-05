# identity-auth-S10.1 定点代码复审 R02

- 日期：2026-10-24
- Reviewer：add-reviewer（独立复审，非实现作者）
- 实现作者：Backend-Dev Cicero `01a16de0-8446-7b12-8c86-86dfc5e8aca1`
- 对象：B6C / `identity-auth-S10.1`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审范围：仅 R01 F001～F004及返工引入的新问题
- 结论：**CHANGES REQUIRED**

## 1. 固定对象

| 文件 | SHA256 |
|---|---|
| `internal/identity/handler.go` | `FCCF0290BEE93E7730AD8EE8B65A07B8BFC475D0BEA14C5A3AB2372D2C49C36C` |
| `internal/identity/password_reset.go` | `72758A1AC997CA9917F3B8749788577E1B43B5BB2CD8F4C569745F58EC8FE7C9` |
| `internal/identity/password_reset_test.go` | `32148484AAC5F2FCF20DB24B4928996661126D80DA1C93A0F1534761B0646616` |
| R01 `.spec/wip/identity-auth/review-code-s10.1-r01.md` | `322D9963EF8CE7A2CD64CECBE5A4C67C218D3187BEFF2CEC67CAFCEA09B5E663` |

工作树含大量合法既有未提交内容。本次除本报告外未修改、整理或覆盖任何文件，也未执行 stage、commit、push、切分支或清理。

## 2. 两轮实际检查范围

### Round 1：逐项复核 R01

- F001：claim 后准备失败、SMTP 未受理后的 cleanup 故障、双次 cleanup 失败及 expired processing 的收敛。
- F002：commit error 的 committed / not-applied / unknown 精确核对、凭据失效及同键重试。
- F003：SMTP accepted uncertain 的凭据失效、secret-free Audit/trace、cleanup retry/commit unknown、Audit ID 冲突和 `MarkUncertain` 冲突。
- F004：`IdempotencyLock > SMTPOperationBudget`、`IdempotencyTTL > IdempotencyLock` 的严格关系。

### Round 2：返工接缝与真实消费者

沿真实 PostgreSQL idempotency、Audit、TemporaryCredential、登录、密码修改和 Session 消费链检查新增私有 Repository/SQL。重点核对：

- `internal/idempotency/store.go` 的 `MarkUncertain` 仅允许 `processing -> uncertain`；
- 登录事务对 User/TemporaryCredential 的锁及临时凭据 Session 创建；
- 密码修改对临时凭据的消费、永久密码写入和 Session 旋转；
- Session 解析不会按临时凭据当前状态自动撤销已创建 Session；
- 新增 SQL 使用现有表、状态值和唯一身份字段，未改动公共 `PasswordResetRepository` 接口。

## 3. 复审结果

| R01 Finding | 状态 | 结论摘要 |
|---|---|---|
| F001 | **CLOSED** | claim 后失败统一尝试 `MarkUncertain`；双次 cleanup 均失败时，后续请求可将已过租约的 processing 原子转为 uncertain。 |
| F002 | **OPEN** | commit unknown 的安全收敛仍不能保证“返回 uncertain 时邮件值不可用”，且新增终态/并发路径会留下已消费凭据或成功快照与失效凭据的不一致。 |
| F003 | **CLOSED** | accepted uncertain 的凭据失效、Audit、trace 与 uncertain transition 在同一 cleanup 事务执行，并有重试及精确 Audit ID 核对；其 `MarkUncertain` 冲突后的业务一致性问题计入 F002。 |
| F004 | **CLOSED** | 构造器和边界测试均落实两项严格大于关系。 |

## 4. Findings

### BLOCKER

无。

### MAJOR

#### IDENTITY-S10.1-REV-R01-F002 — commit unknown 仍可能返回 uncertain，但邮件凭据已可用或已被消费

- **状态**：OPEN
- **位置**：
  - `internal/identity/password_reset.go:874-927`
  - `internal/identity/password_reset.go:952-1012`
  - `internal/identity/password_reset.go:1460-1497`
  - `internal/identity/password_reset.go:1686-1699`
  - 真实消费者：`internal/identity/login.go:1174-1206,1238-1329,1419-1482,2173-2213`
  - 真实消费者：`internal/identity/service.go:415-516`
  - 真实消费者：`internal/identity/bootstrap.go:236-337`
  - 测试缺口：`internal/identity/password_reset_test.go:808-888,890-1018`
- **违反合同**：
  - B6C Context Pack `.add/project/knowledge/plans/ir-01-content-community-batches.md:647`；
  - test-plan I-CRED-06 `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:287`；
  - design/architecture 的 SMTP residual-window 合同：返回 uncertain 时邮件值不可用，重试不得重复产生有效副作用。
- **证据与触发场景**：
  1. 主事务实际已经提交 active 临时凭据、temporary mode、Session 撤销、成功 Audit 和 succeeded 幂等结果，但 `Commit` 返回确认丢失。
  2. 若验证事务无法建立/查询失败，`securePasswordResetUnknown` 只尝试两次 cleanup；两次 Begin、失效或 commit 均失败后仍返回 `identity.operation_uncertain`。数据库中实际提交的 active 邮件凭据和 succeeded 快照可继续存在；同键重试会 replay success。
  3. 即使 cleanup 可执行，邮件接收者也可在原提交释放 User 锁后先完成临时凭据登录；登录会创建新的临时凭据 Session。其后还可与 cleanup 竞争密码修改事务并把该凭据置为 `consumed`、写入永久密码。
  4. `InvalidatePasswordResetCredential` 把 `consumed` 等任意非 active/pending/expired 状态直接视为已经安全，不撤销竞态窗口中新建的 Session，也不回滚已完成的永久密码变更。
  5. 若幂等记录已经是 `succeeded`，`MarkUncertain` 必然冲突；当前实现仍提交“凭据失效 + uncertain Audit”，使记录保持 succeeded。后续同键请求可 replay success，而对应邮件凭据已被 cleanup 失效。现有冲突测试只断言错误和 Audit 被保留，没有证明最终幂等状态、凭据可用性及同键重试一致。
- **影响**：
  - 对外返回 uncertain 时，邮件值可能仍能登录，或已经成功派生 Session/永久密码；
  - 也可能先失效凭据却保留 succeeded 快照，随后 replay 一个实际不可使用的“成功”结果；
  - 因此 I-CRED-06 的安全语义和幂等重试语义均未闭环。
- **最小修复方向**：
  - cleanup 必须与真实消费者使用兼容的锁顺序，原子核对并处置目标 User、精确临时凭据及其派生 Session；不得把 `consumed` 无条件视为安全。
  - `MarkUncertain` 冲突后必须读取并精确分类当前幂等状态：只有完整证明原成功且结果仍一致时返回原成功；否则必须进入持久化的 fail-closed/recovery 状态，不能留下 succeeded 快照与失效凭据组合。
  - 在任何返回 uncertain 的出口前，必须有可证明的 durable invariant，确保邮件值及其派生认证能力不可用；若两次同步 cleanup 仍不能建立该事实，需要受控恢复机制，而不是仅用有限重试后返回。
  - 增加状态化/并发测试，至少覆盖：原 commit 实际成功且验证与两次 cleanup 全失败；登录先于失效；凭据先被 consumed；`MarkUncertain` 因 succeeded 冲突后的同键重试。

### MINOR

无。

### SUGGESTION

无。

## 5. 已关闭 Finding 的证据

### F001 — CLOSED

- `password_reset.go:203-215,303-343,744-872` 为 claim 后准备失败和 cleanup 失败提供 uncertain 收敛与两次尝试。
- expired processing 在当前事务内转 uncertain 后提交。
- `password_reset_test.go:374-405,488-645` 覆盖准备阶段各故障、cleanup Begin/Commit 故障、SMTP failure cleanup 故障及 expired processing。
- 双次 cleanup 都失败时记录可能暂留 processing，但租约到期后的下一请求具有确定恢复路径，不再永久卡死。

### F003 — CLOSED

- `password_reset.go:874-949` 在同一 cleanup 事务执行精确凭据失效、secret-free residual-window Audit、Audit ID 核对和 `MarkUncertain`。
- cleanup commit unknown 会重试，并用完整 Audit identity 避免重复/冲突写入。
- `password_reset_test.go:699-1018` 覆盖 accepted persistence 故障、commit 三态、Audit、trace、secret safety、冲突和 cleanup retry。
- `MarkUncertain` 冲突所暴露的 succeeded replay/凭据状态不一致属于 F002 的剩余安全语义，不重复登记 finding。

### F004 — CLOSED

- `password_reset.go:161-171` 严格要求：
  - `IdempotencyLock > SMTPOperationBudget`
  - `IdempotencyTTL > IdempotencyLock`
- `password_reset_test.go:1155-1188` 覆盖等于、小于及严格大于边界。

## 6. 实际命令与结果

```text
git rev-parse HEAD
git status --short
Get-FileHash -Algorithm SHA256 -LiteralPath internal/identity/handler.go,internal/identity/password_reset.go,internal/identity/password_reset_test.go,.spec/wip/identity-auth/review-code-s10.1-r01.md
rg -n "TemporaryCredential|credential_mode|Consume|Session|session" internal/identity -g "*.go"
rg -n "authenticated_credential_id|AuthenticatedCredentialID|temporary_credentials|credential_id" internal/session internal/identity
go test ./internal/identity -run 'TestPasswordReset' -count=1
go test -race ./internal/identity -run 'TestPasswordReset' -count=1
go vet ./internal/identity
gofmt -l -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go
git diff --check -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go
```

结果：

- 定向普通测试：PASS
- 定向 race 测试：PASS
- `go vet ./internal/identity`：PASS
- `gofmt -l`：无输出
- scoped `git diff --check`：PASS
- HEAD、三文件和 R01 SHA256 均与 Lead 固定值一致
- 未读取、打印或使用 `HAN_TEST_POSTGRES_DSN`

## 7. 未验证风险

- 未连接真实 PostgreSQL，因此没有执行真实 commit-ack-loss、跨事务锁竞争和登录/密码修改并发故障注入；本报告的 F002 由当前 SQL、事务顺序及真实消费者合同静态证明。
- 未连接真实 SMTP/TLS；production composition、真实 SMTP/PG/E2E 仍属后续范围，未作为本卡新增缺陷。
- 当前工作树其余既有未提交内容不在本次定点复审结论范围。

## 8. Gate

- blocker：0
- major：1 OPEN
- minor：0
- suggestion：0

**Code Review Gate: CHANGES REQUIRED**

复审条件：修复 F002 后固定新哈希，并提供上述四类状态化/并发回归证据，再定点复核。
