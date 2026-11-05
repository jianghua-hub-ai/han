# identity-auth-S10.2 最终定点代码复审 R04

- Reviewer：Gibbs，原非作者独立 Reviewer
- 日期：2026-10-24
- 活动：`development-delivery / code review re-review`
- objectType：`code`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审目标：R01 `IDENTITY-S10.2-REV-R01-F003` current capability root lineage 修复，以及 `IDENTITY-S10.2-REV-R01-F004` preflight 回归
- 上轮报告：`.spec/wip/identity-auth/review-code-s10.2-r03.md`
- 上轮报告 SHA256：`57E2F90966C5CB9D1BE09283CBBDCE89AE97522E5AEC431A8D5DCCD27FBDA4D1`

## 1. 结论摘要

**Code Review Gate: CHANGES REQUIRED**

| Finding | 严重度 | R04 状态 | 摘要 |
|---|---|---|---|
| `IDENTITY-S10.2-REV-R01-F003` | HIGH | **OPEN** | 递归 CTE 已能验证直接 current temporary/resend chain，永久 root 字段也已进入分类；但合法的“旧 reset A 已生成永久能力，后续 reset B 又生成新的永久能力”会因当前永久 root 不再等于 A 而被误判为 integrity drift，并对正常 User 写入 durable hold。设计明确规定后续 reset 不使更早的 succeeded 历史变成漂移。 |
| `IDENTITY-S10.2-REV-R01-F004` | HIGH | **CLOSED（无回归；真实 PG18 apply 仍未验证）** | migration 与 preflight SQL 哈希未变；processing/uncertain 全阻断，succeeded 仅精确放行两类 administrator-create 成功快照，password-reset、未知与结构漂移继续阻断。 |

没有登记新的 R04 finding ID。当前阻断仍属于 R01 F003 的复审条件：普通 stale/ineligible 必须容纳所有合法后续改密和后续 reset 历史，而不能把合法 root 更替写成全局 integrity hold。

F003 已连续多轮未收敛，本报告按用户指定停止条件结束于 `CHANGES REQUIRED`：不自行要求或启动下一轮作者返工，由 Lead 决定后续路由。本报告不关闭 S10.2、不启动 S10.3，也不关闭或替代 `IDENTITY-S10.1-REV-R01-F002`。

## 2. 固定对象与哈希

开始检查时八个固定对象均与指定 SHA256 匹配；报告写入后的结束核对仍匹配、无漂移：

| 对象 | 开始 / 结束 SHA256 |
|---|---|
| `migrations/0008_password_reset_durable.sql` | `37FB3077709462D62C763FE87D4E04B4BDA4CFC498C14C2860B27429D5036980` |
| `internal/identity/model.go` | `4D17E1DBC264C69BAFE3330619098F0C57AD703EC5EA5E611BCE8E2BB548B5C7` |
| `internal/identity/temporary_credential.go` | `66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5` |
| `internal/identity/temporary_credential_test.go` | `659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D` |
| `internal/identity/password_reset_store.go` | `BD6B7CAC16842260A88EE1D3F0E3EA2C927D8FD4DB96F8F09DE2C0E29AFA5CA2` |
| `internal/identity/password_reset_store_test.go` | `0544938226A422077A5FFD3B22582B0E6EE0EE81DA6320EC7EBEA1C26E3290A9` |
| `internal/idempotency/store.go` | `AA0483969B12B058B42DB70A9763501B60ED055AFB91E5ECD6AA7F5C45FDD657` |
| `internal/idempotency/store_test.go` | `3A6EF9313839ED19BE28D4FB635C016DF51F6C58213DDAE88D93A2A65CC49BE7` |

输入报告哈希：

- R01：`DFA7039C9ABB0C9B2BCF2B8B2FEDE71D28AD4A747415F281E981B5892F0A6822`
- R02：`8BA34055690CBDEBAFE849B4926A77CCD23BECB3E5194C90DF4820852ABA9FBC`
- R03：`57E2F90966C5CB9D1BE09283CBBDCE89AE97522E5AEC431A8D5DCCD27FBDA4D1`

## 3. 两轮实际检查范围

### Round 1：F003/F004 固定对象直接检查

1. 复核 Reviewer Role、`add-code-review`、review checklist、abstraction、安全红线及 core/go-service coding、no-hardcode 规则。
2. 对当前 `password_reset_store.go`、`password_reset_store_test.go` 完整修订主体进行逐行检查，重点追踪：
   - `passwordResetResendLineageStatusSQL` 的递归方向、root/base/target、generation 连续、cycle 防护、歧义行与 `QueryRow` 基数；
   - permanent source generation/operation/credential 的 Go 分类；
   - superseded predecessor 是否确实位于同 User 的合法 succeeded successor chain；
   - drift 的 durable hold 与 same-user hold pointer；
   - 新测试是直接驱动分类/写入行为还是只搜索 SQL 字符串。
3. 回归核对 `0008` preflight SQL 和既有 preflight 矩阵，确认 F004 未因本轮 F003 修订退化。
4. 以开始 SHA、结束 SHA 和 staged 状态保证没有混用版本或修改被审对象。

### Round 2：有界上游与真实状态演进深审

有界回读 design r39、stories r15、test-plan r16、ADR-0023、R01/R02/R03，以及 `0002/0004/0007` 和 F003/F004 已界定的直接生产者/消费者事实。重点把当前分类与以下已确认合同对照：

- 历史 succeeded operation 在后续正常改密或后续 reset 后仍是合法历史，不要求旧 credential/current Session 仍 current；
- 后续正常改密保留当前 reset root；后续 reset 成功生成永久 successor 时，当前永久 root 合法更新为该次 reset；
- 只有证据缺失、跨 User、跨代、跨 operation 或非法 settlement 才进入 hold；
- processing/uncertain 与无法分类的 succeeded legacy claim 在 migration 前 fail closed，正常 administrator-create succeeded 历史仍可迁移。

未扩展到 S10.3 Gate、S10.1 coordinator 或 S10.4 worker/composition 的实现完整性。

## 4. 原 findings 定点复审

### 4.1 `IDENTITY-S10.2-REV-R01-F003` — OPEN

#### 本轮已成立的修复

当前实现已经关闭 R03 指出的若干直接缺口：

- `password_reset_store.go:1310-1426` 从唯一 current password-reset credential 逆向沿 `predecessor_operation_id` 递归到 root；递归条件核同 User、exact replaced credential、generation 连续、每层 `succeeded + temporary_active + accepted` authority，并以 operation path 阻止 cycle。
- `password_reset_store.go:1478-1504` 要求链抵达字段形状正确的 `initial_reset` root，并能证明请求指定的 predecessor/credential 位于有效链中。
- `migrations/0008_password_reset_durable.sql:229-237` 的 settled-current partial unique index使同一 User/purpose 最多一条 active/expired credential，因此当前 schema 下最终查询不会因多个 current credential 产生多行 `QueryRow`。
- `password_reset_store.go:904-945` 已读取并检查 permanent generation/source operation/source credential；错误 root operation、credential 或 generation 会保留 drift。
- `password_reset_store.go:735-742` 仍只在 predecessor 已证明属于目标 User 时保存 hold operation ID；跨 User/缺行使用 NULL，安全 hold 不被同 User FK 反向拒绝。
- `password_reset_store_test.go:847-1024,1068-1139` 直接驱动 Go 分类与 hold-only 写入矩阵；不只是 SQL 字符串检查。合法 immediate successor、错误 current authority、跨 User和缺行均有行为断言。

#### 仍未闭环：合法的后续 reset root 更替被误写为 integrity hold

`passwordResetPermanentRootValid` 在 `password_reset_store.go:932-945` 要求当前 User 的：

- `permanent_password_source_operation_id == 被请求重发的历史 predecessor operation`
- `permanent_password_source_credential_id == 被请求重发的历史 credential`

`passwordResetResendCurrentPermanent` 和 `passwordResetResendPermanentLaterReset` 在 `:904-929` 都把该 exact-current-root 条件作为普通 ineligible 的必要条件。

以下是设计允许且可由 schema 表达的正常历史：

1. reset A `succeeded/temporary_active`，随后 PasswordChange 将 A 推进为 `permanent_successor`，A credential consumed；
2. 后续 reset B 再次成功并生成新的永久 successor；User 当前永久 source operation/credential 合法更新为 B，generation 单调增加；
3. 此时收到引用更早 A operation/credential 的 resend/stale 请求。

A 自身仍是 exact、同 User、合法 `succeeded/permanent_successor` 历史，后续 reset B 不会把 A 变成完整性漂移。design r39 `:2149-2150` 与 stories r15 `:344-346` 明确规定后续正常改密和后续 reset 不要求旧 credential/current Session 仍 current，历史 replay 不应改写或否定当前能力。

当前分类却会：

1. 因当前 permanent root 已从 A 合法更新为 B，使 `passwordResetPermanentRootValid(A)` 返回 false；
2. 不能命中 current-permanent 或 later-reset 普通 ineligible 白名单；
3. 保持默认 `integrity_drift`（`password_reset_store.go:705-731`）；
4. 由 `authorizeNewPasswordResetCandidate` 在 `:238-252` 对正常 User 写入 durable integrity hold，且不只是普通零写入拒绝。

若 B 后又有当前 temporary reset C，同样问题仍存在：当前 temporary C 的 lineage 可以完整抵达 B 之后的合法 base generation，但旧 A 仍因 current permanent source 已指向 B 而被误判为 drift。

这不是未验证环境推测，而是当前 Go 布尔条件的确定结果。现有 `"normal later password reset"` 测试（`password_reset_store_test.go:924-928,1910-1926`）只建模“A 仍是 current permanent root，随后出现一个 current temporary reset”；它没有覆盖“后续 reset B 已经生成新的永久 root，再回放更早 A”的正常多代历史。因此新测试验证了实现声明的一部分行为，但没有覆盖设计要求的 root replacement。

影响：合法 stale 请求可以把正常账号置为全局 durable hold；Login、PasswordChange、Session resolve 和后续 reset/resend 都会被 fail closed，造成持久可用性与恢复责任错误，并把正常历史审计成完整性故障。

关闭所需事实仍是：分类必须同时区分“当前 root 与历史 operation 不同但由合法后续 reset 形成”和“current capability/root 漂移”。本报告不在超过三轮未收敛后自行发起下一轮返工或选择实现方案。

### 4.2 `IDENTITY-S10.2-REV-R01-F004` — CLOSED，无回归

- `migrations/0008_password_reset_durable.sql` SHA 与 R03 相同。
- `0008:13-107` 仍阻断全部 `processing/uncertain`。
- succeeded 仍仅精确放行封闭 legacy administrator-create `{outcome,resource_id}` 和 `identity-administrator-create-response/v1` 成功快照；外层 `IS NOT TRUE` 继续阻止 NULL 三值逻辑误放行。
- password-reset、未知 result、extra/missing/NULL/type drift 与混合异常历史继续阻断。
- `password_reset_store_test.go:87-155` 的空库、两类合法管理员、password-reset/unknown succeeded、processing/uncertain、legacy current credential 与混合数据矩阵仍在并通过。

未发现本轮 lineage 修订削弱 preflight，亦未发现 generic Idempotency 被全局强制关联 PasswordResetOperation。F004 保持代码/schema 静态层 CLOSED；真实 PostgreSQL 18 apply 仍归未验证专项。

## 5. 新增 findings

无独立新增 finding。上述合法多代 reset 被误 hold 是 R01 F003 “后续正常改密/后续 reset 必须是普通历史，而非 drift”的原复审条件仍未满足。

## 6. 待决定与非阻断建议

### 待决定

F003 已触发用户指定的超过三轮未收敛停止条件。Reviewer 不选择下一修复方案、不启动下一轮作者返工；由 Lead 决定返回设计/任务所有权对齐、停止该实现切片或采取其他受控动作。

### 非阻断建议

F004 的静态 preflight classifier 可继续保留为快速回归，但不得表述为真实 PG18 migration apply、JSONB 三值逻辑或混合历史实证。

## 7. 实际命令与结果

以下命令均在仓库根目录执行，未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`：

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current` / `git rev-parse HEAD` | `main` / 固定 HEAD |
| 八对象开始 SHA256 | 全部匹配 |
| `go test ./internal/migration ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store\|Durable\|Operation\|Job)\|TestTemporaryCredential\|TestIdempotencyPasswordReset\|TestMigration" -count=1` | PASS |
| `go test -race ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store\|Durable\|Operation\|Job)\|TestTemporaryCredential\|TestIdempotencyPasswordReset" -count=1` | PASS |
| recursive SQL 方向/root/generation/cycle/基数静态审查 | immediate current lineage PASS；未发现方向、cycle 或 schema 内多行问题 |
| permanent 多代 root replacement 控制流审查 | FAIL；合法历史被归为 drift/hold |
| F004 SQL、哈希与 preflight 矩阵回归 | PASS |
| 八对象结束 SHA256 | 全部匹配 |
| 报告 scoped `git diff --check` | PASS |

Lead 提供的 gofmt、mod verify、vet、build、contract、全仓 race、全仓 test 七项 PASS 是外部现场证据；Reviewer 本轮独立自跑的是表中两条定向普通/race 命令，不把 Lead 证据冒充本轮自跑。

## 8. 未验证限制

按授权未运行：

- 真实 PostgreSQL 18 或 integration tag；
- `0008` 在空库、合法管理员历史、legacy password-reset、NULL/类型漂移和混合历史上的真实 apply；
- deferred FK/constraint trigger、JSONB 谓词与 commit-time 行为实证；
- multi-connection、死锁/serialization、同 User 并发 generation；
- commit-ack-loss；
- SMTP/TLS、部署、Host、E2E、容量或 production readiness。

因此 F004 的 CLOSED 不替代 S1.6 的 PG18 migration 专项。F003 的 OPEN 来自确定的 Go 分类条件与上游明确允许的状态演进，不因真实环境未运行而降级。

## 9. 状态边界与停止

- `identity-auth-S10.2` 继续保持 active，不得由本报告 task-level 关闭。
- `identity-auth-S10.3` 不得由本报告启动。
- `IDENTITY-S10.1-REV-R01-F002` 继续独立 OPEN。
- 本 R04 已给出最终 `CHANGES REQUIRED`；Reviewer 不自行要求、安排或启动下一轮作者返工。
