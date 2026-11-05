# identity-auth-S10.1 Code Review R01

- 日期：2026-10-24
- Reviewer：add-reviewer（当前会话独立执行，非实现作者）
- 实现作者：Backend-Dev Cicero `01a16de0-8446-7b12-8c86-86dfc5e8aca1`
- objectType：code
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 批次 / 任务：B6C / `identity-auth-S10.1`
- 结论：**CHANGES REQUIRED**

## 1. 固定对象

本结论只绑定以下工作树快照：

| 文件 | SHA256 |
|---|---|
| `internal/identity/handler.go` | `FCCF0290BEE93E7730AD8EE8B65A07B8BFC475D0BEA14C5A3AB2372D2C49C36C` |
| `internal/identity/password_reset.go` | `82C5B7BD5AEF124E55F6617010F1E22CDBE410F2651997A0EE0165AA30065CF5` |
| `internal/identity/password_reset_test.go` | `ADC478EC3A84314894769B77D25FADE00DF7EF6A07BEB83364C4D7780F9246EF` |

工作树含大量合法既有修改与未跟踪内容。Reviewer 未整理、覆盖、暂存、提交、切分支或清理任何内容；除本报告外未写入文件。

## 2. 两轮实际检查范围

### Round 1：固定三文件直接审查

逐行检查：

- 两条管理员 password-reset 路径、POST、UUID、Principal、trace、单一 `Idempotency-Key`、公共 envelope 与秘密响应边界；
- actor/target 在请求事务内的最新状态、role、initial-admin、credential/recovery/onboarding 复核；
- self、deleted、recovery-review、角色错配与 initial-admin 目标保护；
- digest 的 actor/target/role/purpose 绑定、replay/conflict/in-progress/uncertain 分类；
- durable processing claim、SMTP 前后状态、事务/rollback/commit 顺序及 residual window；
- TemporaryCredential generation、跨 purpose supersede、credential mode、Session 全撤销、Audit 与幂等完成；
- 密码生成 rejection sampling、shuffle、policy、秘密在响应/错误/审计/快照中的边界；
- 错误映射、context budget、函数长度及测试矩阵。

### Round 2：消费者与上游合同交叉检查

有界核对：

- Tasks `identity-auth-S10.1`、B6C Context Pack、Story S10；
- design §3.3.11、幂等状态机、SMTP residual window、事务/审计/错误章节；
- test-plan I-CRED-02/03/05/06；
- 现有 `idempotency.Store`、`mailer.Mailer`/SMTP accepted 合同、TemporaryCredential、Audit、Session、platform error；
- coding/no-hardcode、go-service profile与安全 checklist。

确认 production composition、真实 SMTP/PostgreSQL、Browser、部署与 E2E 属于后续范围，未将这些未执行项误报成本卡缺陷。

## 3. Findings

### BLOCKER

无。

### MAJOR

#### IDENTITY-S10.1-REV-R01-F001 — durable claim 后的多条失败出口会永久遗留 `processing`

- **状态**：OPEN
- **位置**：
  - `internal/identity/password_reset.go:194-202`
  - `internal/identity/password_reset.go:346-405`
  - `internal/identity/password_reset.go:423-451`
  - `internal/identity/password_reset.go:147-149`
  - `internal/identity/password_reset.go:647-679`
  - 真实消费者证据：`internal/idempotency/store.go:478-527`
- **违反合同**：
  - design 幂等状态机：`processing` 必须收敛为 succeeded/failed/uncertain，租约过期不得永久停留；
  - test-plan I-CRED-02：并发/重试不得重复执行，不确定状态必须 fail closed；
  - B6C 要求 processing claim 与 residual window 有可恢复的确定状态。
- **触发场景与证据**：
  1. `claimPasswordReset` 已在独立事务提交 processing；
  2. 随后的随机源、PasswordPolicy、ID、Hasher 或当前凭据查询失败；
  3. `preparePasswordReset` 的错误直接从 `ResetPassword` 返回，仅回滚主事务，没有 Complete/MarkUncertain；
  4. SMTP 未受理后的 failure transaction 若 Begin/Complete/Audit/Commit 失败，也会留下同样残留；
  5. `idempotency.classifyExisting` 对任意 processing 一律返回 `BeginInProgress`，不检查 `LockedUntil` 是否已过期。
- **影响**：同一键后续永远得到 `identity.idempotency_in_progress`；请求不能稳定重放、不能自动进入受控 uncertain，也缺少本次失败审计。现有测试只证明 claim 在 SMTP 前提交，没有覆盖 claim 后准备失败或 failure cleanup 自身失败。
- **最小修复方向**：为“claim 已提交”建立统一终结责任；所有后续出口在有界 cleanup context 内原子写入安全稳定失败或保守 uncertain，并在 cleanup 自身失败时保持可受控恢复，而不是遗留不可过期的 processing。补随机源、policy、ID、hash、credential query 与 failure-cleanup 故障回归。

#### IDENTITY-S10.1-REV-R01-F002 — commit unknown 可能实际已提交有效临时凭据，却仍返回 uncertain

- **状态**：OPEN
- **位置**：
  - `internal/identity/password_reset.go:476-480`
  - `internal/identity/password_reset.go:707-734`
  - `internal/identity/password_reset.go:770-781`
  - `internal/identity/password_reset_test.go:469-512`
- **违反合同**：
  - B6C Context Pack 与 test-plan I-CRED-06：返回 uncertain 时邮件中的值不可用，重试不得重复有效副作用；
  - design：成功提交才允许新 credential active；SMTP accepted 后提交未知须按 residual window 受控核对。
- **触发场景与证据**：
  1. 主事务已经写入 active credential、temporary mode、Session 撤销、成功 Audit 与 succeeded 幂等完成；
  2. `tx.Commit` 返回“结果未知”的传输错误，但数据库实际已经提交；
  3. cleanup 的 `MarkUncertain` 因幂等记录已是 succeeded 而影响零行并失败；
  4. 实现忽略该事实差异，统一返回 `identity.operation_uncertain`，但邮件中的临时凭据实际可登录。
- **影响**：公共响应宣称结果未确认，安全合同却允许该邮件秘密成为有效凭据；测试 stub 只断言返回 uncertain/调用 MarkUncertain，没有模拟“Commit 返回错误但事务已落库”的真实 unknown outcome。
- **最小修复方向**：commit error 后按原 record、credential ID、audit ID 做只读精确核对；若完整原子成功已提交，应返回已确认的原成功结果；若未提交，才转 uncertain；无法证明时必须保证邮件值不可用并保持受控核对。禁止盲写或把零行 MarkUncertain 当作已完成。

#### IDENTITY-S10.1-REV-R01-F003 — SMTP accepted 后的 uncertain 路径没有持久审计

- **状态**：OPEN
- **位置**：
  - `internal/identity/password_reset.go:633-734`
  - `internal/identity/password_reset_test.go:477-512`
- **违反合同**：
  - Story S10：最新凭据、Session 与审计属于同一纵切；
  - Story 公共验收：高风险业务状态与 AuditEvent 同事务，审计失败时 fail closed；
  - test-plan I-CRED-06：residual window 必须有 trace、告警和审计；
  - 安全 checklist：凭据管理失败必须可审计。
- **触发场景与证据**：SMTP 已 accepted，随后 supersede/insert/mode/session/audit/idempotency completion 任一步失败。主事务回滚后，`markPasswordResetUncertain` 仅更新 IdempotencyRecord，未调用 `Audit.Append`；`platform.WriteError` 也只写公共 envelope，不能替代审计。现有 accepted-failure 表测试未断言任何 uncertain AuditEvent。
- **影响**：高风险邮件已发送且凭据结果不确定，但数据库中可能只有 uncertain 幂等记录，没有 actor、target、trace、purpose 与失败分类的审计事实，无法满足安全调查与恢复核对。
- **最小修复方向**：在不误判 commit unknown 的前提下，为 accepted-but-uncommitted/unknown 形成 secret-free uncertain 审计；已证明原事务未提交时可与 uncertain transition 原子追加，commit unknown 则先按不可变 audit ID 做精确只读核对，禁止盲目重复成功/失败事件。补每个 accepted persistence failure 的审计与 trace 断言。

### MINOR

#### IDENTITY-S10.1-REV-R01-F004 — 构造器未拒绝破坏幂等与 SMTP 时序的 duration 组合

- **状态**：OPEN
- **位置**：
  - `internal/identity/password_reset.go:104-119`
  - `internal/identity/password_reset.go:491-492`
  - `internal/identity/password_reset.go:750-757`
  - `internal/identity/password_reset_test.go:673-686`
- **违反合同**：现有幂等/SMTP 配置合同要求 `IdempotencyLock > SMTPOperationBudget` 且 `IdempotencyTTL > IdempotencyLock`；错误配置应 fail closed。
- **触发场景与证据**：构造器仅验证三个 duration 均大于零，因此可创建“SMTP budget 长于 processing lease”或“record TTL 不长于 lease”的服务。
- **影响**：租约/清理消费者启用后可能在 SMTP 尚未结束时把执行判为超租约，或过早清除 processing 记录，削弱单次发送与受控恢复保证。
- **最小修复方向**：在 `NewPasswordResetService` 校验两项严格大小关系，并补边界表测试；不要求本卡实现 production 环境变量解析或 composition。

### SUGGESTION

无独立 suggestion。

## 4. 已确认无 finding 的关键检查

- 普通有效管理员可重置 member；仅 initial admin 可重置 admin；initial admin 目标受专用错误保护。
- actor 在事务内重核 ID、role、initial flag、status、deleted、recovery、credential mode 与 onboarding；self/目标 role mismatch/deleted/recovery-review 均 fail closed。
- Handler 只接受两条精确 POST 路径与 UUID；错误使用 canonical envelope/trace，成功响应不含秘密。
- digest 实际绑定 actor ID、target ID、target role 与 password-reset purpose；同键同摘要 replay 不重发/不重写，同键异摘要冲突。
- processing claim 在 SMTP 前独立提交。
- SMTP 明确未受理时主业务事务回滚，旧凭据、credential mode 与 Session 不写入。
- 正常成功路径在同一事务 supersede 当前跨-purpose credential、插入最新 active password-reset credential、设置 temporary、撤销全部 Session、追加 Audit、完成幂等。
- 临时密码生成使用 rejection sampling 与无偏 Fisher-Yates shuffle，并再次通过 PasswordPolicy。
- 秘密未进入 HTTP 响应、Audit metadata 或幂等快照。
- 本文件中函数均未超过 go-service 50 行上限。

## 5. Reviewer 实际命令

在固定工作区实际运行：

```text
git rev-parse HEAD
git status --short
Get-FileHash -Algorithm SHA256 -LiteralPath internal/identity/handler.go,internal/identity/password_reset.go,internal/identity/password_reset_test.go
git diff --no-ext-diff --unified=80 -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go
go test ./internal/identity -run 'TestPasswordReset' -count=1
go test -race ./internal/identity -run 'TestPasswordReset' -count=1
go vet ./internal/identity
gofmt -l -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go
git diff --check -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go
```

结果：

- 定向普通测试：PASS；
- 定向 race：PASS；
- `go vet ./internal/identity`：PASS；
- `gofmt -l`：无输出；
- scoped `git diff --check`：PASS；
- HEAD 与三文件 SHA256 和固定对象一致。

Reviewer 未读取、打印或使用 `HAN_TEST_POSTGRES_DSN`。

## 6. 限制与未验证风险

- 未连接真实 PostgreSQL，因此未验证真实 `FOR UPDATE` 竞争、物理 rollback、commit transport unknown 与 Session 下一请求失效；这些仍由 S1.6/专项验证承担，但 F002 是三文件内已可证明的 unknown-outcome 接缝错误。
- 未连接真实 SMTP/TLS，未执行 provider accepted、证书、deadline/cancel 或进程崩溃故障注入；属于 S5.3。
- 未验证 production composition、真实路由注册、Host、部署、Browser 或 E2E；这些不作为本卡 finding。
- 当前工作树其他未提交内容不在本结论覆盖范围。

## 7. Gate 与复审条件

严重度统计：

- blocker：0
- major：3 OPEN
- minor：1 OPEN
- suggestion：0

**结论：CHANGES REQUIRED**

复审至少需要：

1. 固定修复后的三文件修订与 SHA256；
2. 逐项关闭 F001～F004；
3. 增加 claim 后准备失败与 cleanup 失败的终结状态测试；
4. 增加“commit 返回错误但实际已提交”的 unknown-outcome 测试；
5. 增加 accepted persistence failure 的 secret-free Audit/trace 测试；
6. 增加 duration 关系构造失败测试；
7. 重跑定向普通/race、vet、gofmt、scoped diff-check，并重核 HEAD、工作树越界和固定对象哈希。
