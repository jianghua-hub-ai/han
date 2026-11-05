# identity-auth-S8.1 Code Review R01

- 日期：2026-10-24
- Reviewer：add-reviewer（当前会话直接执行，非实现作者）
- 实现作者：Backend-Dev James `01a16dc2-80e8-7603-b685-ea20cfe4887d`
- objectType：code
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 批次 / 任务：B6B / `identity-auth-S8.1`
- 结论：**CHANGES REQUIRED**

## 1. 固定对象

本次结论只绑定以下五个文件及审查结束时的 SHA256：

| 文件 | SHA256 |
|---|---|
| `internal/identity/handler.go` | `40E4D8D6D4BE5497B3DDD7C513DA144AB105EDC0A5DADF068BDFDF06020450C6` |
| `internal/identity/account_status.go` | `B8F3D9172E9705C820132AF978A2426AAEF03B4A75DE5EC788792F176D46EFEE` |
| `internal/identity/article_reader.go` | `6EB3D10F508561193E5AF0AD9A99B49DAE58850517028BE173445E636EFED183` |
| `internal/identity/account_status_test.go` | `44ED7A12B153D44FEE933A4EF77DD7B22EB2BF87A95A21048079D0B09FE1B54A` |
| `internal/identity/article_reader_test.go` | `71DCFF76C340C36F9F6E1E20C47AF65DFCC59E142B2EFD7E2D71F072DCE744DB` |

工作树存在大量合法既有修改和未跟踪内容；Reviewer 未整理、覆盖、暂存、提交、切分支或清理任何内容。

## 2. 两轮实际检查范围

### Round 1：固定实现直接审查

逐行检查五个固定文件，覆盖：

- `active ↔ suspended`、重复操作 `changed=false`、`deleted` 终态；
- Handler 的 user/administrator 路径、opaque UUID、HTTP 响应与错误传递；
- actor/target 权威行锁、每请求授权复核、自操作和 initial admin 保护；
- 状态更新、仅 suspend 撤销全部 Session、Audit、commit/rollback；
- no-change 审计、ID/Clock/依赖失败和内部错误封装；
- Article 批量读取、去重、缺失、非法状态、行读取错误、资源关闭和只读 SQL；
- 单元测试是否证明业务行为，而非只证明 stub 调用。

### Round 2：有界消费者与上游合同审查

交叉核对：

- `tasks.md` 的 `identity-auth-S8.1` 与 B6B Context Pack；
- Story S08、design §3.3.9、§3.4.3/6、公共错误注册表和恢复状态合同；
- test-plan I-GOV-01～05；
- `session.Principal`、`ResolvePrincipal`、`Restricted` 与 credential-mode 派生；
- `passwordChangeRequired`、User 模型、users migration 约束；
- platform 公共错误定义；
- 当前 `cmd`、article 消费者与后续任务登记。

确认生产路由/组合根接线不在本任务五文件写集内；article 消费接线属于后续 `article-management-S2.1`，真实 PostgreSQL 事务/锁专项属于 `identity-auth-S1.6`，均未误报为本任务实现缺陷。

## 3. Findings

### MAJOR

#### IDENTITY-S8.1-REV-R01-F001 — `recovery_locked` 的最新 actor 仍可通过事务内授权

- **状态**：OPEN
- **位置**：
  - `internal/identity/account_status.go:245-252`
  - 关联证据：`internal/identity/service.go:824-830`
  - 关联消费者：`internal/session/store.go:281-284,334-340`
- **违反合同**：
  - Story S08：每次请求都重新鉴权；
  - task S8.1：每请求复核目标状态、权限和资格；
  - design 的 `must_change_password` 派生包含 `recovery_locked`，且恢复锁定账号不得使用日常能力；
  - 产品安全红线 3.1.1：每个授权请求必须核实当前授权。
- **触发场景 / 可复现证据**：
  1. initial admin 的请求先通过 Session middleware，此时 Principal 仍是 active、unrestricted；
  2. 并发恢复终结把该账号改为 `credential_mode=recovery_locked`、清除凭据并删除 Session；按设计 initial admin 不依赖 `recovery_required=true`；
  3. 请求随后进入 `SetAccountStatus` 并锁到最新 User；
  4. `authorizeAccountStatusActor` 只调用 `passwordChangeRequired`，而该 helper 不包含 `CredentialModeRecoveryLocked`，所以当前锁定状态仍通过授权。
- **影响**：已经进入恢复锁定、凭据和 Session 均应失效的 initial admin，仍可能借已进入请求继续停用或恢复账号，破坏恢复期 fail-closed 和“最新资格”授权边界。
- **最小修复方向**：事务内授权对治理能力使用明确 credential-mode allowlist，至少拒绝 `recovery_locked`；不要只依赖进入 Service 前的 Principal。补充“stale unrestricted Principal + 当前 User 为 recovery_locked”的回归。
- **复审验证**：证明上述竞态返回 `403 auth.forbidden`，且 target 状态、Session、Audit 均无写入并发生 rollback。

#### IDENTITY-S8.1-REV-R01-F002 — initial admin 目标返回了错误的稳定公共 code

- **状态**：OPEN
- **位置**：
  - `internal/identity/account_status.go:135-136`
  - `internal/identity/account_status_test.go:148-152`
- **违反合同**：
  - design 公共错误注册表规定停用 initial admin 使用 `identity.initial_admin_protected`；
  - design §3.3.7 明确 initial admin 作为治理目标时只返回该专用 code；
  - `platform.CodeIdentityInitialAdminProtected` 已存在并固定映射为 HTTP 403。
- **触发场景 / 可复现证据**：initial admin 调用 administrator suspend/restore 路径，以 initial admin 为目标；实现返回 `accountStatusForbiddenError()`，测试也把 `auth.forbidden` 固化为期望值。
- **影响**：HTTP 状态虽为 403，但稳定业务 code/message 与既有公共合同不一致，客户端无法区分普通越权与受保护账号，安全审计分类也失真。
- **最小修复方向**：目标锁定并确认 `IsInitialAdmin` 后返回既有 `identity.initial_admin_protected` PublicError；普通权限失败继续使用 `auth.forbidden`。
- **复审验证**：Service 和 HTTP 测试分别断言 403、专用 code/canonical message、零状态/Session/Audit 写入。

#### IDENTITY-S8.1-REV-R01-F003 — 当前测试未覆盖本任务要求的关键授权与失败矩阵

- **状态**：OPEN
- **位置**：
  - `internal/identity/account_status_test.go:89-124,126-221`
  - `internal/identity/article_reader_test.go:70-93,95-154`
- **违反合同**：
  - test-plan I-GOV-01、I-GOV-04、I-GOV-05；
  - code-review checklist：测试须覆盖实际风险，mock 不得掩盖生产行为；
  - 本次验收明确要求 actor 最新资格、ID/Clock/事务失败、安全错误映射，以及 Article 去重、Scan/rows error/Close。
- **触发场景 / 可复现证据**：
  - 名为 `ReauthorizesEveryRequest` 的测试连续两次返回完全相同 actor，未改变最新 role/status/initial-admin/credential/recovery/onboarding 事实，因而未发现 F001；
  - initial-admin 用例直接期待错误的 `auth.forbidden`，因而固化 F002；
  - transaction/repository/ID/Clock stubs 缺少 begin、actor/target lock、update、revoke、commit、ID、零时钟失败注入；
  - Article rows stub 的 `Err()` 永远为 nil、`Scan()` 无可注入失败，测试也未断言输入去重参数及 `Close()`。
- **影响**：P0 授权与事务失败路径没有自动化保护；当前绿灯不能证明最新资格 fail-closed、关键失败回滚或 Article 行读取失败合同。
- **最小修复方向**：在当前两个测试文件内补最小表驱动矩阵：
  - stale Principal 对应最新 actor 的 role/status/initial-admin/credential/recovery/onboarding 变化；
  - begin/lock/update/revoke/audit/commit、ID 和零时钟失败的 code、调用顺序、rollback/零后续副作用；
  - malformed path/UUID 和专用 public code 的 HTTP envelope；
  - Article 去重参数、Scan error、`rows.Err()`、冲突重复行和 Close。
  真实 PostgreSQL 锁与物理回滚仍留给已登记的 S1.6，不要求本卡越界新增集成文件。
- **复审验证**：新增用例先能复现 F001/F002，再在修复后通过；所有失败分支断言稳定公开 code 与零越界副作用。

### MINOR

#### IDENTITY-S8.1-REV-R01-F004 — 两个核心函数超过 go-service 强制 50 行上限

- **状态**：OPEN（非阻断）
- **位置**：
  - `internal/identity/account_status.go:84-203`
  - `internal/identity/article_reader.go:39-92`
- **违反合同**：go-service profile `rules/coding-standards.md` §1.2 要求函数最大 50 行，超出必须拆分。
- **触发场景 / 可复现证据**：`SetAccountStatus` 约 120 行，`GetAuthorStates` 约 54 行。
- **影响**：事务、授权和副作用编排集中在长函数中，增加安全分支遗漏和失败路径复审成本；F001 正发生在该集中编排内。
- **最小修复方向**：只做行为保持的局部提取，例如事务内授权/锁定、状态副作用、审计提交及 Article 行验证；不得借机扩大抽象或改公共契约。
- **复审验证**：函数满足 profile 上限，现有及新增行为测试保持通过，diff 不扩出五文件授权边界。

### BLOCKER

无。

### SUGGESTION

无独立 suggestion；后续生产接线与 PostgreSQL 专项已有明确任务，不在本报告重复创建建议项。

## 4. 已确认无 finding 的关键检查

- `active→suspended`、`suspended→active`、重复 suspend/restore 的 `changed` 行为正确。
- `deleted` 或已有 `deleted_at` 均被视为终态。
- actor 与 target 都通过 `SELECT ... FOR UPDATE` 在同一事务中读取。
- 状态更新带旧状态条件；非 1 行更新会 fail closed。
- 仅实际进入 suspended 时撤销该用户全部未撤销 Session；restore 和 no-change 不撤销。
- 状态更新、Session 撤销、Audit append 和 commit 使用同一 `pgx.Tx`；失败路径由 defer rollback 收敛。
- no-change 仍生成 `Outcome=no_change` 的 AuditEvent。
- ID 生成错误/空 ID、零时钟及 repository/audit 错误均包装为 `platform.internal_error`，内部 cause 不作为公共 code 暴露。
- Handler 区分 users/administrators 与 suspend/restore 路径；额外路径段和空 ID 返回不可用目标，Service 在查询前验证 UUID。
- Article seam 对输入 UUID 做规范化和去重，一次批量 SELECT；缺失用户不补造，非法/冲突状态和 `rows.Err()` fail closed，并通过 defer Close。
- Article SQL 仅查询 identity-owned `users`，没有 Article 写入或跨模块写端口。
- 未发现凭据、Session token、CSRF、DSN 或其他敏感内容进入错误、审计 metadata 或测试输出。

## 5. Reviewer 实际命令

以下命令由 Reviewer 在固定工作区实际运行：

```text
git status --short --branch
git rev-parse HEAD
git branch --show-current
Get-FileHash -Algorithm SHA256 <五个固定文件>
go test ./internal/identity -run 'TestAccountStatus|TestGetAuthorStates' -count=3
go test -race ./internal/identity -run 'TestAccountStatus|TestGetAuthorStates' -count=1
go vet ./internal/identity
gofmt -l -- <五个固定文件>
git diff --check -- <五个固定文件>
```

结果：

- 定向测试通过；
- 定向 race 通过；
- `go vet ./internal/identity` 通过；
- `gofmt -l` 无输出；
- scoped `git diff --check` 无错误，仅出现 Git 的 LF/CRLF 工作区提示；
- 审查前后分支、HEAD 和五文件哈希一致。

Lead 已报告 go mod verify、全仓 vet/build、contract、race 和全量测试通过；Reviewer 记录该上游证据，但未冒充本轮重跑。

Reviewer 未读取、打印或使用 `HAN_TEST_POSTGRES_DSN`。

## 6. 限制与未验证风险

- 按明确禁止未连接真实 PostgreSQL，因此没有执行真实行锁竞争、事务物理回滚、Session 下一请求失效或 PG rows codec；这些属于已登记的 `identity-auth-S1.6`。
- 未运行浏览器、部署或真实 E2E。
- 当前 production composition 尚未注册本能力；因任务写集和后续任务已明确，本次不列为 S8.1 finding，但开放能力前仍须完成生产接线和契约验证。
- 当前工作树包含其他工作者内容；结论不覆盖五文件哈希以外的未提交实现。

## 7. Gate 与复审条件

严重度统计：

- blocker：0
- major：3 OPEN
- minor：1 OPEN
- suggestion：0

**结论：CHANGES REQUIRED**

复审至少需要：

1. 固定修复后的五文件修订和 SHA256；
2. 逐项关闭 F001～F003；
3. 提供 stale Principal / current `recovery_locked` 回归；
4. 提供 initial-admin 专用错误的 Service 与 HTTP 回归；
5. 补齐本卡范围内的关键失败矩阵，并明确 S1.6 继续承担的真实 PostgreSQL部分；
6. 对 F004 完成最小拆分，或由 Lead 明确记录非阻断风险接受；
7. 复审开始和结束均重核分支、HEAD、工作树越界情况及固定对象哈希。
