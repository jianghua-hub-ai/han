# identity-auth S10.1 Code Review R05

- 日期：2026-10-25
- objectType：code
- Reviewer：identity-auth-S10.1 原非作者独立 Reviewer
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/identity-auth/review-code-s10.1-r04.md`
- 原报告 SHA256：`CA0AE21ED8A4B2AC61144CC165597BF3ED6DB3DFDD4D1054E7AF791EF0D8D3DE`
- 结论：
  - `IDENTITY-S10.1-REV-R04-F001`：**CLOSED**
  - `IDENTITY-S10.1-REV-R04-F002`：**OPEN / 不在本轮修复范围**
  - 新 findings：**无**
- **Code Review Gate：CHANGES REQUIRED**

## 1. 固定对象与防漂移

三文件均为共享工作树中的 untracked 对象，空 `git diff` 不构成审查证据。本轮按固定 SHA256、全文检查和实际测试评审。

| 文件 | 固定 SHA256 |
|---|---|
| `internal/identity/handler.go` | `FCCF0290BEE93E7730AD8EE8B65A07B8BFC475D0BEA14C5A3AB2372D2C49C36C` |
| `internal/identity/password_reset.go` | `926E4507580F608AB37D13ACA88EDF8295DCC50BAE541168655326B8D33724FA` |
| `internal/identity/password_reset_test.go` | `046BF47AD5804AA3E1B5308C0AB3A8458F9AD245C8017F094C563BAF874C3040` |

开始审查时已核：

- 分支为 `main`；
- HEAD 为固定 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；
- 三文件及 R04 报告 SHA256 全部匹配；
- R05 报告路径不存在。

## 2. 两轮实际检查范围

### Round 1：F001 修复与固定三文件

- 检查 Mailer context 的 deadline/cancel 生命周期；
- 检查 Mailer 返回后 evidence context 是否：
  - 保留 request values；
  - 隔离 parent cancellation；
  - 重新施加有限 deadline；
- 检查 accepted/unknown evidence 的 Begin、SQL、job acceleration、Commit 是否全部使用该 context；
- 检查 context-aware transactor/transaction/Mailer fake 是否会真实拒绝已取消 context，而非只检查调用次数；
- 检查 accepted/unknown 两个分支、credential/session 零直接写入、terminal audit 零写入和 unknown 非 rejected/invalidated。

### Round 2：先例与真实消费者边界

- 复核 `administrator_create.go` 既有 reconciliation context 先例；
- 复核 pgx transaction begin/statement/commit 消费传入 context 的生产行为；
- 有界复核 Store/operator/worker 的 accepted/unknown durable settlement 约束；
- 定点搜索唯一 `cmd/han-server` composition 对 password-reset coordinator/Handler/routes 的当前消费情况，确保 R04-F002 未被误关。

未扩展为无关全仓评审。

## 3. R04 Findings 定点结论

### `IDENTITY-S10.1-REV-R04-F001` — CLOSED

原问题：Mailer 内取消 parent 后，accepted/unknown evidence 复用已取消请求 context，production pgx 事务可能无法开始，导致 durable evidence 丢失。

当前修复证据：

1. `internal/identity/password_reset.go:310-315`
   - SMTP 调用仍使用 `context.WithTimeout(ctx, SMTPOperationBudget)`；
   - `Mailer.Send` 返回后立即调用 `cancelMail()`；
   - secret 随即清空；
   - evidence 写入使用单独构造的 `evidenceCtx`。

2. `internal/identity/password_reset.go:330-337`
   - `context.WithoutCancel(ctx)` 隔离 parent cancellation/deadline；
   - 再用 `context.WithTimeout(..., SMTPOperationBudget)` 添加正数有限 deadline；
   - `WithoutCancel` 保留 request context values；
   - 模式与 `administrator_create.go:551-557` 的既有 reconciliation context 一致。

3. `internal/identity/password_reset.go:316-325`、`:339-385`
   - accepted 与 unknown 分支都把 `evidenceCtx` 传入 evidence 方法；
   - transaction `Begin`、Store SQL、recovery job acceleration、Commit 均沿同一 evidence context 执行；
   - unknown 仍持久化 `commit_unknown/unknown`，不直接失效能力、不写 terminal audit、不重发。

4. `internal/identity/password_reset_test.go:513-609`
   - 新增 context-aware transactor/tx；
   - Begin、QueryRow、Exec、Commit、Rollback 均观察 context；
   - 已取消 context 会实际返回错误，不再由忽略 context 的旧 fake 掩盖 production 行为。

5. `internal/identity/password_reset_test.go:1449-1577`
   - accepted/unknown 两个子用例均由 Mailer stub 在 `Send` 内取消 parent；
   - 断言 Mailer context 有 deadline；
   - 断言 evidence Begin 和每个事务操作看到：
     - `ctx.Err() == nil`；
     - 存在 deadline；
     - request value `"request-value"` 被保留；
   - 断言 evidence transaction 实际 Commit 一次；
   - accepted evidence 与 recovery job acceleration SQL 实际执行；
   - unknown evidence包含 `commit_unknown/unknown` 与 job acceleration，且不含 rejected、`recovery_invalidated` 或 terminal audit；
   - credential/session 直接写入为零。

独立测试结果：

- 精确测试普通、race、`-count=20` 普通/race均 PASS；
- 全部 `TestPasswordReset` 普通/race均 PASS。

因此 R04-F001 的触发场景、production context 语义和 durable outcome 均已有直接实现与真实执行测试覆盖，结论 **CLOSED**。

### `IDENTITY-S10.1-REV-R04-F002` — OPEN / 受限

该项明确不在本轮两文件修复范围，本轮不得误关。

当前定点搜索仍显示：

- `NewPasswordResetService` 与 `NewPasswordResetHandler` 的实际调用只存在于 `password_reset_test.go`；
- `cmd/han-server` 没有构造首次 reset coordinator/Handler；
- 两条公共路由仍未注册：
  - `POST /api/v1/admin/users/{id}/password-reset`
  - `POST /api/v1/admin/administrators/{id}/password-reset`

该 finding 需要 `cmd/han-server` composition/router及测试的跨文件所有权对齐。按既有停止条件，本 Reviewer 不自行修改或推定归属。

结论：**OPEN / 跨文件所有权待 Lead、Task-Planner或设计对齐**。

## 4. 新 Findings

**无。**

本轮没有发现 F001 修复引入新的正确性、安全、secret safety、context 生命周期、durable状态机或测试真实性缺陷。

## 5. 实际命令与结果

以下命令均在仓库根目录实际执行：

| 命令 | 结果 |
|---|---|
| `go test ./internal/identity -run '^TestPasswordResetDeliveryEvidenceIgnoresParentCancellation$' -count=1` | PASS |
| `go test -race ./internal/identity -run '^TestPasswordResetDeliveryEvidenceIgnoresParentCancellation$' -count=1` | PASS |
| 同一精确测试 `-count=20` | PASS |
| 同一精确 race 测试 `-count=20` | PASS |
| `go test ./internal/identity -run '^TestPasswordReset' -count=1` | PASS |
| `go test -race ./internal/identity -run '^TestPasswordReset' -count=1` | PASS |
| `go test ./internal/identity -list '^TestPasswordReset'` | PASS；新 cancellation/evidence 测试可见 |
| `go vet ./internal/identity ./internal/mailer ./internal/session ./cmd/han-server` | PASS |
| `git diff --check -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go` | PASS；三文件 untracked，故仅作补充检查 |
| `gofmt -l -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go` | PASS；无文件输出 |
| `rg` 定点搜索 `NewPasswordResetService`、`NewPasswordResetHandler` 与 production password-reset routes | F002 仍存在；production无构造/路由 |
| `Get-FileHash -Algorithm SHA256`、`git branch --show-current`、`git rev-parse HEAD` | 固定对象、分支及HEAD匹配 |

Lead 提供的全量 Manifest PASS 作为交付背景保留，但本表只登记 Reviewer 本轮独立实际执行的命令。

## 6. 未验证风险与限制

- 按禁止项未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未连接真实 PostgreSQL或SMTP，未部署、未执行E2E。
- 未验证真实 provider 在边界时刻取消/超时后的实际 acknowledgement；本轮证明的是 Mailer 返回后应用侧 evidence context 与 durable写入行为。
- 未执行真实多连接、commit-ack-loss、worker crash/restart、migration混跑、容量或 production readiness 验证。
- F002 的真实 HTTP route 仍不可达，本轮没有因 F001 关闭而外推公共 REST、部署或整体 Story PASS。
- 共享工作树其他大量改动不在本结论范围；本轮未回滚、覆盖、格式化、stage、commit、push或清理任何他人内容。

## 7. Gate

| 项目 | 结论 |
|---|---|
| `IDENTITY-S10.1-REV-R04-F001` | **CLOSED** |
| `IDENTITY-S10.1-REV-R04-F002` | **OPEN / 跨文件所有权待对齐** |
| 新 findings | **无** |

**Code Review Gate：CHANGES REQUIRED**

F001 的本轮修复通过定点复审；Gate 继续被未纳入本轮修复范围的 R04-F002 阻断。只有在唯一 production composition 中完成合法所有权下的 coordinator/Handler/两条公共路由接线，并经独立复审后，才可重新判断 PASS。
