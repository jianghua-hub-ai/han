# identity-auth-S9.1 Code Review R01

- 日期：2026-10-25
- Reviewer：add-reviewer（当前会话直接执行，非实现作者）
- 实现作者：Poincare `01a1716c-b94a-7602-b84b-02978e3d1db8`（已关闭）
- objectType：code
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 任务：`identity-auth-S9.1`
- 结论：**CHANGES REQUIRED**

## 1. 固定对象

本次结论只绑定以下五个文件及审查结束时的 SHA256：

| 文件 | SHA256 |
|---|---|
| `internal/identity/deletion.go` | `87F697959265240EE303B3BE4E2A9A5B49644A84B6ED58D83E3433685B5E9821` |
| `internal/identity/article_port.go` | `F12DB7648F36CD530879434A9B0D0E4231E955253614ADFA5F3999AF40A1ED10` |
| `internal/identity/deletion_test.go` | `A3E2F1871A88455149805B1F0B9B798D1E67F39599C2508BD5E41617570E1663` |
| `internal/identity/service.go` | `7E5064E40447A35EFC4F6537ED845E6EBD4A7199169208302B63C5905643C70D` |
| `internal/identity/handler.go` | `FC4B1711F8B25DB8508DAA8DE1ACD87AB38C7B0F1655679768FD64C80304C8FC` |

审查开始、定向验证后及报告写入前均未发现固定对象漂移。Reviewer 未修改被审文件、Tasks/Plan、其他代码或 `han-server.exe`，未暂存、提交、推送、切分支、回滚或清理。

## 2. 两轮实际检查范围

### Round 1：固定五文件直接审查

逐行检查五个固定文件，覆盖：

- 日常删除与 recovery `kept_deleted` 是否复用唯一协调器；
- User、Identifier、Session、TemporaryCredential、Article、Audit、RecoveryAccountReview 的事务顺序、失败回滚与 commit-unknown；
- 零文章、多文章、违规、已无归属、deletion pending 的计数和 retained_email/email Identifier 清理；
- actor/target 权限、initial admin 保护、reason、HTTP 状态与公共错误最小披露；
- identity→article 消费端 port 的依赖方向和 caller-owned transaction；
- 单元测试的失败注入、幂等、权限和 HTTP 覆盖。

### Round 2：有界消费者与上游合同审查

交叉核对：

- tasks SSOT `identity-auth-S9.1`、Story S09、test-plan I-DEL-01～07；
- identity design S4/S5/S14，尤其 §3.3.10 与恢复流程 §4.2；
- identity User/Identifier/Session/TemporaryCredential/Audit/RecoveryAccountReview 模型与仓储；
- password-reset recovery worker 对删除后凭据状态的消费；
- identity-owned `RecoveryRunStore` 与 `recovery_runs` 状态机；
- article canonical `OrphanArticlesCanonical` port、command、审计 trace 与矩阵测试；
- registration/administrator-create 对 Identifier 唯一真相的真实消费；
- migration 中 users、identity_identifiers、recovery_account_reviews、recovery_runs 约束。

范围止于上述真实消费者和直接依据；未扩展为全库评审。production composition/router 与真实 PostgreSQL 跨模块集成按用户给定限制仅记录为未验证风险。

## 3. Findings

### MAJOR

#### IDENTITY-S9.1-REV-R01-F001 — 删除可在 phone 墓碑缺失或错绑时提交，手机号可被重新注册

- **状态**：OPEN
- **位置**：
  - `internal/identity/deletion.go:183-203`
  - `internal/identity/deletion.go:807-848`
  - 关联事实：`migrations/0002_identity.sql:53-62`
  - 真实消费者：`internal/identity/registration.go:173-176,207-215`
- **违反依据**：
  - Story S09：email/phone Identifier 清理前必须拒绝复用；
  - I-DEL-05：删除后 username/email/phone 注册须受生命周期约束；
  - design §3.3.10：phone 清理条件仍未确认，删除时须按既有生命周期保留；
  - checklist：关键持久化不变量应 fail closed，不能以不完整 mock 代替生产约束。
- **触发场景 / 证据**：
  1. 目标 User 删除前有 phone，但对应 `identity_identifiers(kind='phone')` 行因历史数据损坏而缺失，或其 `normalized_value` 与 User.phone 错绑；
  2. `MarkUserDeleted` 先把 `users.phone` 清为 NULL；
  3. `ReserveIdentifiers` 只更新现存行；
  4. `verifyCanonicalDeleteIdentifiersSQL` 只要求 username，以及 retained_email 非空时的 email，完全不验证 phone，也不验证墓碑值与被删 User 的原值一致；
  5. schema 只约束 Identifier 表内部唯一性，没有保证 User.phone 与 phone Identifier 一一存在且值一致；
  6. 后续注册只依赖插入 `identity_identifiers` 的唯一约束，因此原手机号可以被新账号插入。
- **影响**：P0 标识生命周期被破坏；删除事务成功后原手机号可能失去唯一性墓碑并被其他账号复用。相同的值错绑还可能让 username/email 墓碑保留错误值。
- **最小修复方向**：在清空 User 标识前保存锁定的权威值，并在同一事务中验证 username/email/phone Identifier 的必需行、kind、normalized value 和 lifecycle 与目标一致；phone 清理条件未确认时必须保留正确 reserved 墓碑。任何缺失或错绑均全事务失败。
- **复审条件**：增加缺 phone 行、phone 值错绑（以及适用的 username/email 错绑）回归，证明返回失败、整事务 rollback，User/Identifier/Session/Credential/Article/Audit 均不改变；正常删除后原 phone 注册仍稳定冲突。

#### IDENTITY-S9.1-REV-R01-F002 — recovery `kept_deleted` 未绑定维护锁和 RecoveryRun 权威状态

- **状态**：OPEN
- **位置**：
  - `internal/identity/deletion.go:42-56,98-106`
  - `internal/identity/deletion.go:343-379`
  - `internal/identity/deletion.go:446-485`
  - `internal/identity/service.go:339-351`
  - 关联 schema：`migrations/0002_identity.sql:215-247`、`migrations/0005_recovery_runs.sql:1-45`
- **违反依据**：
  - design §3.3.10：recovery 模式要求维护锁、RecoveryRun、RecoveryAccountReview decision 和 initial admin 恢复身份；
  - design §4.2：RecoveryCoordinator 必须按 run_id 经 identity-owned RecoveryRunStore 驱动受控恢复，并保持 readiness 关闭；
  - ADR-0018：RecoveryRunStore 是 RecoveryRun 状态与事务的唯一 owner；
  - I-DEL-07：任一步失败不得完成 review/readiness。
- **触发场景 / 证据**：
  1. 内部调用者构造 `Mode=recovery_kept_deleted`、非空 run ID，并提供 active initial-admin Principal；
  2. coordinator 只锁 `recovery_account_reviews`，不读取/锁定 `recovery_runs`，也没有维护锁/readiness Gate 依赖；
  3. `recovery_account_reviews.recovery_run_id` 没有到 `recovery_runs` 的外键；
  4. 因而 run 不存在、已 failed/succeeded、backup 不匹配或服务不在维护模式时，只要存在一条 pending/kept_deleted review，仍可执行完整删除并把 pending review 完成。
- **影响**：恢复状态机和 canonical 删除之间出现未受权威状态保护的旁路；错误批次或日常运行期可完成不可恢复删除和 review，RecoveryRun 计数/阶段/readiness 证据无法证明与该事务一致。
- **最小修复方向**：由受控 recovery consumer 持有不可伪造的调用能力，并在执行前通过 identity-owned RecoveryRunStore/维护 Gate 校验并锁定允许的 run、backup、阶段及 readiness 状态；不要仅信任 request 中的 run ID 或孤立 review 行。若边界决定由外层协调器承担，port 必须显式表达并由生产唯一消费者证明，不能保留任意内部调用入口。
- **复审条件**：提供不存在 run、错误 backup、failed/succeeded run、非维护模式和 stale review 的拒绝/rollback 回归；正常 kept_deleted 须证明来自受控 recovery consumer，且 review 完成与允许的 RecoveryRun 阶段一致。

#### IDENTITY-S9.1-REV-R01-F003 — trace_id 有两个真相源，真实 article consumer 可使 recovery 重放失败或审计链分裂

- **状态**：OPEN
- **位置**：
  - `internal/identity/deletion.go:48-56,221-225,488-533`
  - `internal/identity/deletion_test.go:123-145,464-473,692-721`
  - 真实消费者：`internal/article/command_service.go:162-175,218-242`
- **违反依据**：
  - checklist：关联上下文必须跨边界传播；
  - architecture：同一高风险事务中的 AuditEvent 使用可关联的最小 `trace_id`；
  - I-DEL-07：真实 kept_deleted article 协作成功后才可完成 review。
- **触发场景 / 证据**：
  - identity 审计使用 `CanonicalDeleteRequest.TraceID`，但 article canonical authority 从 `context.Context` 读取 trace；
  - coordinator 不验证二者一致，也不把 request trace 注入 context；
  - recovery 单测明确以 `context.Background()` 调用，却用同时充当 identity/article 的 fake，未执行 article 的真实 trace 前置条件；
  - 用真实 article consumer 且存在待更新文章时，相同调用会因 context 无 trace 而失败并无法完成 kept_deleted；若 context 与 request 各有不同 trace，则同一事务的 identity/article 审计被写入不同关联链。
- **影响**：恢复矩阵在 fake 中通过但真实消费者失败；或删除虽然提交，却无法通过单一 trace 关联同事务内的身份和文章审计。
- **最小修复方向**：只保留一个权威 trace 来源。优先从 ctx 读取并用于所有审计；若内部 command 必须携带显式字段，则入口验证字段与 ctx 完全一致并把同一 ctx 传递给 article port。
- **复审条件**：使用真实 article adapter/contract test 覆盖 recovery + 待更新文章；缺 trace 和 trace mismatch 必须在任何写入前 fail closed，匹配 trace 时 identity/article AuditEvent 必须一致。

### MINOR

#### IDENTITY-S9.1-REV-R01-F004 — 核心事务编排函数超过 go-service 强制 50 行上限

- **状态**：OPEN（非阻断）
- **位置**：`internal/identity/deletion.go:162-280`
- **违反依据**：go-service profile `rules/coding-standards.md` §1.2：函数最大 50 行，超出必须拆分。
- **触发场景 / 证据**：`canonicalDeleteInTransaction` 约 119 行，集中处理授权后的七类副作用、清理条件、两个审计和 recovery review。
- **影响**：关键失败顺序和模式分支难以局部验证，增加再次遗漏安全 Gate 或回滚步骤的风险。
- **最小修复方向**：仅做行为保持的局部提取，例如身份墓碑阶段、article/retained-email 阶段、审计/review 终结阶段；保持单一事务 owner，不引入第二 authority。
- **复审条件**：函数满足 profile 上限，调用顺序与 rollback 测试保持通过，diff 不扩出授权文件。

### BLOCKER

无。

### 待决定 / 非缺陷

- production composition/router、identity→article 结果 DTO adapter 尚未在本次固定对象中出现；tasks 已把生产接线和真实跨模块验证留给后续工作，本轮不把“尚未装配”单列为 S9.1 实现 finding。
- phone 最终清理条件仍为 Q2；本报告只要求在条件未确认期间可靠保留墓碑，不替业务决定何时释放。

## 4. 已确认无 finding 的关键检查

- 日常删除与 recovery kept_deleted 复用同一 `CanonicalDeleteCoordinator`，未建立第二套删除 authority。
- User、Identifier、Session、TemporaryCredential、Article、Audit 和 pending RecoveryAccountReview 的当前写入位于 caller-owned 同一 PostgreSQL 事务；已覆盖的步骤错误会 rollback，commit 失败映射 operation uncertain。
- article port 位于 identity 消费端，规则仍由 article canonical command 拥有；未复制 article SQL/状态 authority。
- article canonical 对 active owned、already orphaned、deletion pending、disposed 和 violation 保留的生产规则与合同一致。
- 零 pending 时才清理 retained_email/email Identifier；存在 pending 时保留。
- DELETE users/administrators 路径、204、reason allowlist/default、initial admin 保护和公共错误最小披露未发现新增直接缺陷。
- 未发现 schema/migration、新依赖、硬编码凭据或第二 canonical authority 越界。
- password-reset worker 在 canonical 删除后不能把 `recovery_invalidated` credential 再激活；删除事务与 worker 均先锁 User，当前交错下未发现删除后认证能力复活。

## 5. Reviewer 实际命令与结果

主要命令：

```text
git branch --show-current
git rev-parse HEAD
git status --short
git diff --cached --name-only
Get-FileHash -Algorithm SHA256 <五个固定文件>
rg / Get-Content：固定五文件、identity 有界消费者、article canonical 合同、tasks/Story/test/design/migrations
go test ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle" -count=1
go test -race ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle" -count=1
go test ./internal/article -run "TestOrphanArticlesCanonical" -count=1
```

结果：

- identity 定向测试：PASS；
- identity 定向 race：PASS；
- article canonical 定向测试：PASS；
- 分支仍为 `main`，HEAD 仍为固定基线；
- 五个固定文件 SHA256 全部匹配；
- staged 区为空；
- Reviewer 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。

Lead 提供的固定 SHA 现场 gofmt、scoped diff-check、目标普通/race、identity 包普通与 vet PASS 作为上游证据保留；Reviewer 未把其冒充为本轮重跑。

## 6. 限制与未验证风险

- 未运行真实 PostgreSQL 跨模块集成，因此未验证真实 FK/行锁/物理 rollback、article adapter、retained_email 清理及并发交错；这些仍是 S9.2 的发布前 Gate。
- 未验证 production composition/router，不能据此宣称 endpoint 已在生产可达或 recovery consumer 已安全装配。
- 未运行浏览器、部署或真实 E2E。
- 当前工作树包含其他 Agent 的既有 unstaged/untracked 内容；本报告不评价或归属这些变更。
- 单元测试的内存 fake 不执行 production Identifier inventory SQL、RecoveryRun Gate 或 article trace 前置条件，因此当前绿灯不能关闭 F001～F003。

## 7. Gate 与复审条件

严重度统计：

- blocker：0
- major：3 OPEN
- minor：1 OPEN

**Code Review Gate：FAIL / CHANGES REQUIRED**

复审至少需要：

1. 固定修复修订及五个对象的新 SHA256；若修复需要越过五文件，先由 Lead 明确新的文件所有权与契约方向；
2. 逐项关闭 F001～F003，并提供对应失败先复现、修复后回归证据；
3. F001 必须证明 phone/值绑定异常全事务失败且删除后不可复用；
4. F002 必须证明 maintenance/RecoveryRun 权威 Gate 不可由 request 或孤立 review 伪造；
5. F003 必须以真实 article consumer/adapter 证明单一 trace 传播；
6. F004 完成最小拆分，或由 Lead 明确记录非阻断风险接受；
7. 复审开始和结束均重核分支、HEAD、固定对象哈希、staged 与越界写入；真实 PostgreSQL和 production composition 仍单独闭环。
