# identity-auth-S10.2 代码定点复审 R02

- Reviewer：Gibbs，非作者独立 Reviewer
- 日期：2026-10-24
- 活动：`development-delivery / code review re-review`
- objectType：`code`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原作者：Backend-Dev Bacon `01a16ec3-1d9a-7920-b014-4d38a44b5b31`
- 原报告：`.spec/wip/identity-auth/review-code-s10.2-r01.md`
- 原报告 SHA256：`DFA7039C9ABB0C9B2BCF2B8B2FEDE71D28AD4A747415F281E981B5892F0A6822`

## 1. 结论摘要

**Code Review Gate: CHANGES REQUIRED**

R01 的五项 finding 复审结果：

| Finding | 严重度 | R02 状态 | 摘要 |
|---|---|---|---|
| `IDENTITY-S10.2-REV-R01-F001` | HIGH | CLOSED（真实 PG18 commit 仍未验证） | 双向 deferred FK、User current-operation 指针和 counter constraint trigger 已形成新 password-reset writer 的 commit-time 五对象约束；通用 Idempotency 行仍可保持 NULL marker。 |
| `IDENTITY-S10.2-REV-R01-F002` | HIGH | CLOSED（真实 PG18 transition/constraint 仍未验证） | DB authority matrix、单调转换、terminal evidence 冻结和 Go exact classification 已对齐；合法 `temporary_active → permanent_successor` 保留。 |
| `IDENTITY-S10.2-REV-R01-F003` | HIGH | **OPEN** | resend SQL 仍会把部分 settlement/User 漂移降级为普通 stale；跨 User predecessor 的 drift hold 又会被 hold-operation FK 在 commit 时拒绝。 |
| `IDENTITY-S10.2-REV-R01-F004` | HIGH | **OPEN** | 新 preflight 无 purpose 判别地阻断所有 `processing/uncertain/succeeded` 通用 Idempotency 历史，正常 administrator-create succeeded 记录也会使 `0008` 永久不可采用；没有受控可审计分类路径。 |
| `IDENTITY-S10.2-REV-R01-F005` | MEDIUM | CLOSED | 按真实函数体统计，最大函数体 34 行，低于 go-service 强制 50 行上限。 |

未登记新的 R02 finding ID：本轮发现的两个阻断问题分别属于 R01 F003 的 lineage/drift 边界和 R01 F004 的 migration preflight 复审条件，避免重复计数。

本 Gate 不关闭 S10.2、不启动 S10.3，也不改变 `IDENTITY-S10.1-REV-R01-F002` 的独立 OPEN 状态。

## 2. 固定对象与哈希

开始检查时，原报告及八个固定对象均与用户给定 SHA256 一致。报告写入后再次核对八对象，结果仍一致、无漂移：

| 对象 | 开始 / 结束 SHA256 |
|---|---|
| `migrations/0008_password_reset_durable.sql` | `000A6345A60C916AEAE5420962881F05D39C9759458353CA39046C2754826DB7` |
| `internal/identity/model.go` | `4D17E1DBC264C69BAFE3330619098F0C57AD703EC5EA5E611BCE8E2BB548B5C7` |
| `internal/identity/temporary_credential.go` | `66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5` |
| `internal/identity/temporary_credential_test.go` | `659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D` |
| `internal/identity/password_reset_store.go` | `30CB06E68E0F25B78F19527C3BC0C9375A5EE9258FC2C2A312D61DBFC3FF6B95` |
| `internal/identity/password_reset_store_test.go` | `6BE9C590A58E77D1F1DA9C00AE8702BF868311A614D9CCFC210ABF1E8C060D8F` |
| `internal/idempotency/store.go` | `AA0483969B12B058B42DB70A9763501B60ED055AFB91E5ECD6AA7F5C45FDD657` |
| `internal/idempotency/store_test.go` | `3A6EF9313839ED19BE28D4FB635C016DF51F6C58213DDAE88D93A2A65CC49BE7` |

## 3. 两轮实际检查

### Round 1：固定八对象直接检查

完整读取新八对象和新测试，逐项检查：

1. password-reset marker、六字段 identity、operation/claim/job/credential/User counter 的双向 deferred FK、UNIQUE、CHECK 与 constraint trigger；
2. `BeginOrClassifyCandidate` 的 User-first 顺序、same-key/orphan/bad-binding 分类、五对象写入和 caller-owned transaction 错误路径；
3. operation `state × delivery × settlement × terminal` 当前组合、转换单调性、terminal evidence 冻结和 exact replay authority matrix；
4. resend eligible/ineligible/integrity-drift 查询、hold 写入和 stale 普通拒绝边界；
5. migration preflight 对 legacy credential、通用 Idempotency 历史和正常升级样本的实际谓词；
6. 新测试是行为证据、静态结构证据还是仅字符串存在检查；
7. 真实函数体行数，而非“当前函数到下一个函数”的文本跨度。

### Round 2：有界上游与真实消费者深审

沿 R01 已使用的边界重新核对受影响部分：

- S10.2 task 的八文件边界、五对象全有或全无、preflight 和真实 PG18 后续专项边界；
- design r39 的 operation authority、lineage/settlement、migration/混跑和记录保留合同；
- test-plan r16 的 I-PRD-02～05、I-PRD-16～18；
- accepted ADR-0023；
- 通用 Idempotency Store 及现役 password-reset claim/completion；
- administrator-create 对同一 `idempotency_records` 的正常 succeeded 消费证据；
- `0002_identity.sql`、`0004_idempotency.sql`、`0007_idempotency_response_snapshot_v1.sql` 和 migration runner 的直接定义。

没有扩成全仓泛审；未评价未来 S10.3 Gate、S10.1 coordinator 或 S10.4 worker/composition 是否已经实现。

## 4. R01 findings 定点复审

### 4.1 `IDENTITY-S10.2-REV-R01-F001` — CLOSED

修订已建立可静态核证的 commit-time 关系：

- `0008:25-52` 为 password-reset claim 增加成对 marker 与六字段唯一约束；marker 同为 NULL 的通用 Idempotency 行不被错误强制关联 operation。
- `0008:224-254` 建立 operation → claim/credential/predecessor/replaced 的 deferred FK。
- `0008:377-440` 建立 operation ↔ recovery job、claim ↔ operation 双向 deferred FK。
- `0008:442-454` 用 User current-operation 指针绑定当前 execution generation。
- `0008:523-587` 用 deferred constraint trigger 检查 User counter 与 operation generation。
- `password_reset_store.go:304-330` 仍保持 User counter → operation → exact credential → job → claim 的单事务写序。
- `idempotency/store.go:247-283,1152-1159` 只由 password-reset 专用入口写非 NULL marker，通用入口 `store.go:1143-1150` 仍保持通用语义。

由此，当前新 writer 的 counter-only、带 marker 的 claim-only、operation-only，以及任一后续写点返回零行后误 commit，都会由 current-operation/FK/constraint trigger 在 commit 时拒绝；第一步 User UPDATE 为零行时没有部分写入。

测试 `password_reset_store_test.go:491-580` 已补“每个写点失败后 Commit”和三种独立 partial shape，但 Commit 是确定性 transaction stub，不是 PostgreSQL constraint 实执行。结合 schema 直接检查，本 finding 在本代码 Gate 内关闭；真实 PG18 deferred commit、约束名/执行时点和事务 abort 行为继续是明确未验证限制。

注意：旧 generic writer 在 marker 为 NULL 时仍可留下无法识别 purpose 的 claim，必须由既定停旧 writer/混跑阻断和 F004 preflight 闭环处理；这不是把 generic Idempotency 全局强制成 password-reset operation 的理由。

### 4.2 `IDENTITY-S10.2-REV-R01-F002` — CLOSED

证据：

- `0008:266-359` 封闭 settlement evidence、delivery evidence、terminal time 和 authority 组合。
- `0008:594-730` 固定 binding、delivery/settlement/state 单调转换并冻结 terminal evidence；唯一保留的 terminal settlement 演进是 succeeded 的 `temporary_active → permanent_successor`。
- `password_reset_store.go:471-534` 的 Go authority matrix 与 DB 当前组合一致。
- `password_reset_store.go:787-929` 的 exact replay 查询同时核六字段关系、credential/job 关系、authority、terminal、settlement 和 delivery evidence；generic Idempotency status 没有被提升为 operation authority。
- `password_reset_store_test.go:383-450,761-823` 覆盖坏 authority 进入 hold 和 Go authority 正反矩阵。

未发现当前约束阻断设计允许的 worker 收敛路径：prepared/dispatching/commit-unknown/recovery-required/converging 的单向推进仍可达，succeeded temporary settlement 后的合法永久 successor 也可达。真实 PostgreSQL transition/CHECK/trigger 矩阵未执行，作为限制保留。

### 4.3 `IDENTITY-S10.2-REV-R01-F003` — OPEN

修订加入了三分类和 User/current lineage 条件，但边界仍不封闭。

#### 触发 A：非法 settlement/User 漂移被降级为普通 stale

`password_reset_store.go:971-1000` 先识别两个合法特例，随后无条件把
`replaced.state IN ('superseded','consumed','recovery_invalidated')`
归为 `ineligible`。

可达反例：

1. predecessor 仍为 `succeeded + temporary_active`；
2. exact replaced credential 已为 `consumed`；
3. User 已为 `credential_mode=permanent`、`auth_generation=target+1`；
4. predecessor 没有原子推进到 `permanent_successor`。

这组跨对象状态表示 PasswordChange settlement 漂移，应进入 integrity hold；当前查询会落到 `:998-999` 的普通 `ineligible`，`authorizeNewPasswordResetCandidate` 在 `:245-247` 仅返回 `ErrPasswordResetGateClosed`，不持久化 hold。类似地，`permanent_successor` 与错误 current User generation 的 consumed 组合也会被同一兜底掩盖。

#### 触发 B：跨 User predecessor 的 drift hold 无法提交

- SQL 在 `password_reset_store.go:975-980` 把 predecessor/replaced 属于其他 User 判为 `integrity_drift`。
- Go 在 `password_reset_store.go:248-251` 对所有 drift 直接把调用方给出的 predecessor ID 写入 User hold。
- `0008:487-490` 又要求 `auth_integrity_hold_operation_id + users.id` 必须引用同一 User 的 operation。

因此 predecessor 实际属于其他 User 时，方法可返回 `PasswordResetCandidateIntegrityHold`，但 caller commit 会被 deferred FK 拒绝，目标 User 没有得到 durable hold。现有 `password_reset_store_test.go:658-758` 让 stub 直接返回分类值或只搜索 SQL 片段，没有执行上述 SQL truth table，也没有模拟 deferred hold FK。

影响：非法 lineage 可能被当成普通 stale，或 hold 写入在 commit 时失败；后续请求无法依靠持久 hold 暴露并受控恢复完整性故障。

修复建议：

1. 先完整验证 predecessor authority、settlement、current User capability 和 exact credential 的合法终态，再仅把明确列举的正常历史 successor/stale 组合归为 `ineligible`；其余组合归为 drift。
2. drift hold 只能保存已证明属于目标 User 的 operation ID；跨 User、缺行或不可安全绑定时应以 NULL operation ID 持久 hold，不能让安全动作被 FK 反向拒绝。
3. 增加真实分类矩阵，至少覆盖 temporary→consumed 但 operation 未推进、permanent successor/current generation 漂移、跨 User predecessor、正常 consumed successor、正常 superseded stale 和 recovery-invalidated stale，并验证 hold 的 commit shape。

复审条件：上述矩阵中所有非法跨对象组合稳定产生可提交的 durable hold 和零新 candidate 写入；正常 stale 仍仅普通拒绝。

### 4.4 `IDENTITY-S10.2-REV-R01-F004` — OPEN

`0008:13-21` 当前谓词是：

```sql
FROM idempotency_records
WHERE status IN ('processing', 'uncertain', 'succeeded')
```

它没有 password-reset purpose 判别，异常消息却把所有命中行称为 legacy password-reset claim。

真实通用消费者证据：

- `internal/identity/administrator_create.go:514-523` 会把合法 administrator-create IdempotencyRecord 完成为 `succeeded`；
- `internal/identity/password_reset.go:700-715` 的旧 password-reset succeeded 也使用同一张通用表；
- HMAC scope 不可由数据库明文反推 purpose。

因此，只要数据库中存在任意合法、仍在 TTL 内的 administrator-create succeeded 历史，`0008` 就会在创建新 schema 前失败。它满足“宁可拒绝”却不满足 R01 复审条件中的“正常样本可前向迁移”，会把正常通用历史变成全局、无解除路径的升级阻断。

`password_reset_store_test.go:86-108` 只检查 preflight 区域同时出现表名、三个 status 和 `purpose='password_reset'` 字符串；后者来自 TemporaryCredential 查询，不能证明 Idempotency 谓词按 purpose 限定，所以测试无法发现该误阻断。

修复建议：

- 对可由稳定 result/snapshot 明确排除的正常通用终态不得全局阻断；
- 对 processing/uncertain 等 SQL 内无法从 HMAC scope 安全识别的行，提供在停旧 writer 后、migration/writer 开放前执行的受控、可审计分类/处置证据；不得猜测 scope，也不得静默略过；
- 增加至少包含“空库、正常 administrator succeeded、旧 password-reset processing/uncertain/succeeded、legacy current credential、混合正常与异常历史”的确定性 preflight 矩阵。

若可审计分类必须超出固定八文件或需要新的部署/运行责任合同，应停止返工并由 Lead 返回 Task-Planner/设计对齐；Reviewer 不替作者选择新架构。

### 4.5 `IDENTITY-S10.2-REV-R01-F005` — CLOSED

独立按 Go 词法边界统计 `password_reset_store.go` 的真实函数体，最大为：

- `validPasswordResetCandidateShape`：`745-778`，34 行；
- `passwordResetOperationAuthorityValid`：`471-503`，33 行；
- `passwordResetCandidateBindingRow.toBinding`：`439-469`，31 行。

其余函数体均更短，满足 go-service profile 的强制 50 行上限。现有 AST 测试 `password_reset_store_test.go:825-843` 以完整 declaration span 检查，口径比仅函数体更严格且本轮通过。

## 5. 新增 findings

无独立新增 finding。F003、F004 的开放原因均直接属于原 finding 的复审条件。

## 6. 待决定

无新的业务行为、公共 REST/权限/错误合同或 ADR 方向待决定。

若 F004 的 purpose 分类和审计前置无法在固定八文件内安全完成，Lead 需要决定是否返回 Task-Planner/设计调整文件所有权；在该决定前不得用 blanket global block 代替可采用 migration。

## 7. 非阻断建议

1. 当前 counter completeness trigger 默认历史 operation 持续存在。design 的清理周期仍为 TBD；未来引入 operation/claim/job/credential 清理前，应重新验证删除历史 generation 后不会让 User counter completeness 在下一次 User 更新时永久失败。本轮没有清理 writer，因此不另登记当前阻断 finding。
2. F001/F002 的 stub/static tests 可保留为快速回归，但不得在报告或任务关闭时冒充真实 PostgreSQL deferred constraint、transition trigger 和 commit 行为证据。

## 8. 实际命令与结果

以下命令均在仓库根目录执行，未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`：

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current` | `main` |
| `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 原报告与八对象 `Get-FileHash -Algorithm SHA256` | 开始哈希全部匹配 |
| S10.2 定向 `go test` | PASS |
| identity/idempotency 定向 `go test -race` | PASS |
| 七个固定 Go 对象 `gofmt -l` | PASS，无输出 |
| `go vet ./internal/migration ./internal/identity ./internal/idempotency` | PASS |
| `go build ./...` | PASS |
| `go mod verify` | PASS，`all modules verified` |
| 独立真实函数体行数扫描 | PASS，最大 34 行 |
| 八对象结束 SHA256 | 全部匹配，无漂移 |
| 报告 scoped `git diff --check` | PASS |

未独立复跑 Lead 报告的全量 test、全量 race、contract 七项全集；其现场 PASS 只作外部证据，不替代本轮定向结论。

## 9. 未验证限制与剩余风险

本轮按授权未运行：

- 真实 PostgreSQL 18；
- `0008` 在空库、正常历史、异常历史和混合历史上的真实 apply；
- deferred circular FK、constraint trigger 和 partial-write commit 的真实行为；
- multi-connection、deadlock/serialization、并发同 User generation；
- commit-ack-loss；
- integration tag；
- SMTP/TLS、部署、Host、E2E、容量或 production readiness。

因此，F001/F002 的 CLOSED 是代码/schema 级独立评审结论，不代表真实 PG18 专项通过。F003/F004 是静态控制流、SQL 语义和真实消费者即可确定的阻断缺陷，不因上述环境未执行而降级。

## 10. 复审与交接条件

再次复审前至少需要：

1. 修复 `IDENTITY-S10.2-REV-R01-F003` 的完整合法/非法 lineage 分类及跨 User hold commit；
2. 修复 `IDENTITY-S10.2-REV-R01-F004`，证明正常通用 Idempotency 历史可迁移、password-reset 遗留风险被受控且可审计地阻断；
3. 对两项修复补确定性行为矩阵，而非仅增加 SQL 字符串存在检查；
4. 固定八对象形成新的明确 hash，作者完成定向普通/race、适用 Manifest 和 scoped diff-check；
5. 原非作者 Reviewer 定点复审，Lead 再核真实工作树、文件所有权、diff 和未验证边界。

在此之前，S10.2 保持 active、不得 task-level 关闭，S10.3 不得启动；本报告也不关闭或替代代码 finding `IDENTITY-S10.1-REV-R01-F002`。
