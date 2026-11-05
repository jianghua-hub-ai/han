# identity-auth-S10.2 定点代码复审 r03

- Reviewer：Gibbs，原非作者独立 Reviewer
- 日期：2026-10-24
- 活动：`development-delivery / code review re-review`
- objectType：`code`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审目标：R01 `IDENTITY-S10.2-REV-R01-F003`、`IDENTITY-S10.2-REV-R01-F004` 及本轮修订新增风险

## 1. 结论摘要

**Code Review Gate: CHANGES REQUIRED**

| Finding | 严重度 | R03 状态 | 摘要 |
|---|---|---|---|
| `IDENTITY-S10.2-REV-R01-F003` | HIGH | **OPEN** | 跨 User/缺 predecessor 的 hold 指针已改为可提交的 NULL，且 operation/credential authority 校验显著补强；但“普通 permanent successor”和“superseded stale”白名单仍未核验 User 当前永久能力的 root lineage，或当前 temporary 能力确由合法后续 operation 形成，完整性漂移仍可降级为普通拒绝。 |
| `IDENTITY-S10.2-REV-R01-F004` | HIGH | **CLOSED（真实 PG18 apply 仍未验证）** | preflight 现阻断全部不可分类的 processing/uncertain，且 succeeded 仅放行精确封闭的 administrator-create legacy/v1 成功快照；password-reset、未知 purpose、NULL/类型/字段漂移保持阻断。 |

未登记新的 R03 finding ID；本轮阻断问题仍直接属于 R01 F003 的“完整 current capability/lineage 才能普通 stale”复审条件。

本 Gate 不关闭 S10.2、不启动 S10.3，也不改变代码 finding `IDENTITY-S10.1-REV-R01-F002` 的独立 OPEN 状态。

## 2. 固定对象与哈希

开始检查时八个固定对象均与指定 SHA256 匹配；报告写入后的结束核对仍匹配、无漂移：

| 对象 | 开始 / 结束 SHA256 |
|---|---|
| `migrations/0008_password_reset_durable.sql` | `37FB3077709462D62C763FE87D4E04B4BDA4CFC498C14C2860B27429D5036980` |
| `internal/identity/model.go` | `4D17E1DBC264C69BAFE3330619098F0C57AD703EC5EA5E611BCE8E2BB548B5C7` |
| `internal/identity/temporary_credential.go` | `66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5` |
| `internal/identity/temporary_credential_test.go` | `659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D` |
| `internal/identity/password_reset_store.go` | `517009D14049F998AF4C88C12BD0E83AFE7704246BCE2AE23682A75D99D8A8C4` |
| `internal/identity/password_reset_store_test.go` | `33EC4BBDDBFD9D3D644662F34E85BD889F5C9B8EAAA4DBAE22521795EFB51F2C` |
| `internal/idempotency/store.go` | `AA0483969B12B058B42DB70A9763501B60ED055AFB91E5ECD6AA7F5C45FDD657` |
| `internal/idempotency/store_test.go` | `3A6EF9313839ED19BE28D4FB635C016DF51F6C58213DDAE88D93A2A65CC49BE7` |

R01、R02 均只读；本报告是唯一写入。

## 3. 两轮实际检查范围

### Round 1：原 findings 定点直接检查

1. 完整读取 Reviewer Role、`add-code-review`、评审清单、抽象原则、安全红线及 core/go-service coding、no-hardcode、security 规则。
2. 完整读取 R01、R02 和新八对象；逐行核对：
   - resend evidence 查询、Go 分类白名单、hold operation pointer 与测试矩阵；
   - migration preflight 的 SQL 三值逻辑、封闭字段集和正常/异常 fixture。
3. 将作者声明与真实 SQL/Go 控制流及测试断言逐项对照，不以声明或 PASS 命令替代行为检查。

### Round 2：有界生产者/消费者与上游合同深审

有界回读：

- identity design r39 §3.3.11/12、§3.4.1/2/8、§9.4；
- stories r15、test-plan r16 I-PRD-02～05/16～18、accepted ADR-0023；
- S10.2 task 卡与 Planning Review r02；
- migrations `0002_identity.sql`、`0004_idempotency.sql`、`0007_idempotency_response_snapshot_v1.sql`；
- 直接生产者/消费者 `password_reset.go`、`administrator_create.go`、`internal/idempotency/store.go`。

重点确认：

- legacy password-reset succeeded 确实写 `result_code=password_reset` 与 legacy snapshot；
- administrator-create 当前 succeeded 写封闭 v1 success snapshot，历史 legacy success 为封闭 `{outcome,resource_id}`；
-通用 Idempotency 已有 `recovery_review_required` 与受控 Recover 路径，因此无法分类的 processing/uncertain 可在 migration 前先受控收敛，而不是由 `0008` 猜测 purpose。

## 4. 原 findings 复审

### 4.1 `IDENTITY-S10.2-REV-R01-F003` — OPEN

#### 已关闭的子问题

本轮修订已满足 R02 的部分条件：

- `password_reset_store.go:660-686` 默认分类为 `integrity_drift`，先验证 predecessor operation authority、delivery evidence 与 exact credential。
- `password_reset_store.go:689-697` 只有已证明 predecessor 属于目标 User 时才保存 hold operation ID；跨 User 或缺行使用 NULL，避免 `users_auth_integrity_hold_operation_fk` 反向拒绝安全 hold。
- `password_reset_store_test.go:836-905,907-979` 增加 consumed 无 successor、错误 mode/generation、跨 User、缺行及 NULL hold pointer 的确定性矩阵。

#### 仍未闭环：current capability 的 root lineage 未进入分类证据

`passwordResetResendLineageStatusSQL` 在 `password_reset_store.go:1158-1189` 只读取当前 User 的：

- `credential_mode`
- `auth_generation`

它没有读取：

- `permanent_password_generation`
- `permanent_password_source_operation_id`
- `permanent_password_source_credential_id`
- 当前 temporary capability 对应的 operation/credential lineage

因此两个白名单仍不封闭：

1. `password_reset_store.go:786-796` 将 `permanent_successor + consumed + permanent mode + auth_generation=successor` 直接归为普通 ineligible，但没有证明永久摘要 generation/root operation/root credential 与该 predecessor 的 exact succeeded transition 一致。
2. `password_reset_store.go:798-807` 将 `temporary_active + old superseded + current temporary mode + auth_generation>old target` 直接归为普通 stale，但没有证明当前 temporary 能力来自一个合法、同 User、succeeded 的后续 operation。

schema 的 `users_permanent_password_source_fk`（`0008:560-576`）只证明 source operation/credential 与 User 的 FK 关系；`users_permanent_password_generation_check`（`0008:158-174`）只证明 generation 非负且不大于 auth generation。二者没有把当前永久 generation、source operation 的 successor generation 和 settlement 原子绑定，因此应用分类仍必须读取并核验这些事实。

可达反例：

- predecessor 为合法 `succeeded/permanent_successor`，old credential 为 consumed，User mode/auth generation 表面匹配，但 permanent source 指向另一条旧 operation/credential，或 permanent generation 与 predecessor successor generation 不一致；
- old credential 为 superseded，User mode 为 temporary、generation 更大，但当前 temporary capability 没有合法后续 succeeded operation/root lineage。

当前实现均返回普通 `ErrPasswordResetGateClosed`，零 hold 写入；这把设计要求的跨 operation、证据不足或 capability lineage 漂移降级为普通 stale。

违反依据：

- design r39 `design.md:2144-2152,2377-2379,2702,3402-3403`；
- stories r15 `stories.md:344-346`；
- test-plan r16 I-PRD-03/I-PRD-05；
- R02 复审条件“先完整验证 User capability，只对白名单普通 stale 放行分类”。

影响：损坏或越代的当前认证能力不会形成 durable integrity hold；后续 reset/resend 只能看到普通资格拒绝，丢失 fail-closed 故障标记、审计关联和受控恢复责任。

修复建议：

1. resend lineage evidence 必须读取并核验当前 capability 的完整 authority：
   - permanent：generation、root operation、root credential 与 predecessor settlement/successor 的精确关系；
   - superseded stale：当前 temporary credential 及其合法后续 succeeded operation/root generation。
2. 只有上述证据完整时才归为普通 ineligible；缺失、跨 operation、跨 generation 或 settlement 不一致继续归为 drift，并只保存已证明同 User 的 hold operation ID。
3. 扩展矩阵，至少覆盖 wrong permanent root operation、wrong root credential、wrong permanent generation、无合法后续 operation 的 superseded stale及合法后续 reset。

复审条件：所有普通 stale 白名单均能证明 current capability 的合法 root lineage；任一证据缺失/漂移稳定产生可提交的 durable hold、零新 candidate 写入，正常 permanent successor、后续改密和后续 reset 不误触发 hold。

### 4.2 `IDENTITY-S10.2-REV-R01-F004` — CLOSED

`0008:13-107` 已将 preflight 收口为：

1. `processing/uncertain` 全阻断；
2. `succeeded` 仅放行：
   - 精确 legacy administrator-create success：`result_code=created`，无 schema version，exact `{outcome,resource_id}`；
   - 精确 `identity-administrator-create-response/v1` success：固定外层四字段、201/created 与固定八字段 active admin body；
3. 其他 succeeded，包括 password-reset、未知 result、extra/missing/NULL/type drift 全阻断；
4. allowlist 外层使用 `(...) IS NOT TRUE`，SQL NULL 三值逻辑不能意外放行。

直接消费者证据与谓词一致：

- `administrator_create.go:514-523,1173-1189` 当前只写上述 v1 success；
- `password_reset.go:707-714` 的 succeeded 使用 `result_code=password_reset`，不会命中管理员 allowlist；
- `0007` 的 strict v1 schema 与 `0008` allowlist 字段集合一致。

无法从 HMAC scope 反推 purpose 时，对 processing/uncertain 全阻断是可采用的 fail-closed 选择：现役 Idempotency 提供 `RequireRecoveryReview/Recover` 的受控 reviewer、review time 与稳定结果路径；升级前可先收敛这些记录。migration 不把不确定记录猜成 password-reset，也不错误放行正常 administrator-create succeeded 历史。

`password_reset_store_test.go:87-155,1327-1445` 覆盖空库、两类合法管理员 success、结构漂移、password-reset/unknown succeeded、processing/uncertain、legacy current credential 与混合数据。该 fixture 是静态分类器而非 PostgreSQL 执行；结合直接 SQL 和生产者检查，足以关闭代码/schema 级 finding，真实 PG18 apply 仍保留为专项限制。

复审条件已满足；未发现本修订把 generic Idempotency 全局强制关联 PasswordResetOperation。

## 5. 新增 findings

无独立新增 finding。F003 的剩余缺陷仍是原 finding 的 current User capability/lineage 复审条件未完成，不重复登记 R03 ID。

## 6. 待决定与非阻断建议

### 待决定

无新的公共 REST、错误、权限、ADR 或架构方向待决定。F003 可在既有八文件范围和已确认设计内修复。

### 非阻断建议

F004 的静态 fixture 应保留为快速回归，但任务关闭时不得将其表述为真实 PostgreSQL 18 migration apply、JSONB 三值逻辑或混合历史实证。

## 7. 实际命令与结果

以下命令均在仓库根目录执行，未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`：

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current` / `git rev-parse HEAD` | `main` / 固定 HEAD |
| 八对象开始 SHA256 | 全部匹配 |
| `go test ./internal/migration ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store\|Durable\|Operation\|Job)\|TestTemporaryCredential\|TestIdempotencyPasswordReset\|TestMigration" -count=1` | PASS |
| `go test -race ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store\|Durable\|Operation\|Job)\|TestTemporaryCredential\|TestIdempotencyPasswordReset" -count=1` | PASS |
| F003 current capability/root lineage 静态字段与控制流检查 | FAIL：上述字段未进入 resend evidence |
| F004 producer/snapshot/preflight 封闭矩阵检查 | PASS |
| 八对象结束 SHA256 | 全部匹配 |
| 报告 scoped `git diff --check` | PASS |

Lead 提供的 gofmt、mod verify、vet、build、contract、全仓 race、全仓 test 七项 PASS 是外部现场证据；Reviewer 本轮只独立复跑上述两条定向命令，不把 Lead 证据冒充本轮自跑。

## 8. 未验证限制

按授权未运行：

- 真实 PostgreSQL 18 或 integration tag；
- `0008` 在空库、合法管理员历史、legacy password-reset、NULL/类型漂移和混合历史上的真实 apply；
- deferred FK/constraint trigger、JSONB 谓词与 commit-time 行为实证；
- multi-connection、死锁/serialization、同 User 并发 generation；
- commit-ack-loss；
- SMTP/TLS、部署、Host、E2E、容量或 production readiness。

因此 F004 的 CLOSED 是静态 SQL、真实生产者与确定性测试层结论，不代替 S1.6 的 PG18 migration 专项。F003 是直接控制流和缺失证据字段即可确定的阻断缺陷，不因真实环境未运行而降级。

## 9. 交接与复审条件

1. 返工 F003，使 permanent successor 与 superseded stale 的普通拒绝均证明 current capability 的合法 root lineage。
2. 补 wrong root operation/credential/generation、无合法后续 operation 与合法后续 reset 的确定性矩阵。
3. 保持 F004 已关闭的封闭 allowlist、`IS NOT TRUE` 和 processing/uncertain fail-closed 行为。
4. 固定八对象形成新 hash，作者完成定向普通/race、适用 Manifest 和 scoped diff-check。
5. 由原非作者 Reviewer 再做定点复审，Lead 再核真实工作树、文件所有权、diff 与未验证边界。

在此之前，S10.2 保持 active、不得 task-level 关闭，S10.3 不得启动；本报告不关闭或替代 `IDENTITY-S10.1-REV-R01-F002`。
