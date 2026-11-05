# identity-auth-S10.2 代码定点复审 R05

- Reviewer：Gibbs，原非作者独立 Reviewer
- 日期：2026-10-24
- 活动：`development-delivery / code review re-review`
- objectType：`code`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审目标：R04 中 HIGH OPEN `IDENTITY-S10.2-REV-R01-F003` 的 current capability root lineage 修订；回归 `IDENTITY-S10.2-REV-R01-F004` migration/preflight 及既有安全分类
- 上轮报告：`.spec/wip/identity-auth/review-code-s10.2-r04.md`
- 上轮报告 SHA256：`5A8DFE21E1089E88A4EAB89DC4B738E8CDF0FC5B83DF1D3DDF29B36D1C0FEB66`

## 1. 结论摘要

**Code Review Gate: PASS**

| Finding | 原严重度 | R05 状态 | 结论 |
|---|---|---|---|
| `IDENTITY-S10.2-REV-R01-F003` | HIGH | **CLOSED** | A 后正常改密、A→B current permanent、A→B→current temporary C 均不再因合法 current root 替换而误判 drift/写 hold；same-key replay 保持 A 原终态。current projection 缺失、跨 User、generation/source/exact binding/authority 漂移仍默认 fail closed 并原子写 hold + 去敏 Audit。 |
| `IDENTITY-S10.2-REV-R01-F004` | HIGH | **CLOSED，无回归；真实 PG18 apply 仍未验证** | `0008` 未漂移；processing/uncertain 全阻断，succeeded 仅精确放行封闭 administrator-create legacy/v1 snapshot，password-reset、未知和结构漂移继续阻断。 |

本轮未发现新增 HIGH/MEDIUM/LOW finding。PASS 只表示本固定八文件 revision 的独立代码评审通过，不自动关闭 `identity-auth-S10.2`，不启动 `identity-auth-S10.3`，也不改变 `IDENTITY-S10.1-REV-R01-F002` 的独立 OPEN 状态。

## 2. 固定对象与哈希

开始检查与报告写入后的结束复核均匹配下列 SHA256，无对象漂移：

| 对象 | 开始 / 结束 SHA256 |
|---|---|
| `migrations/0008_password_reset_durable.sql` | `37FB3077709462D62C763FE87D4E04B4BDA4CFC498C14C2860B27429D5036980` |
| `internal/identity/model.go` | `4D17E1DBC264C69BAFE3330619098F0C57AD703EC5EA5E611BCE8E2BB548B5C7` |
| `internal/identity/temporary_credential.go` | `66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5` |
| `internal/identity/temporary_credential_test.go` | `659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D` |
| `internal/identity/password_reset_store.go` | `9FAE66FB1C9BF73405291FDFB0406044AC5E4390D690005D4A3B122656CD42CE` |
| `internal/identity/password_reset_store_test.go` | `B316B55FCC7A7BFD2D10886643F974ECAA2E8CD4C2BF696DAB7C8D9A268CDA40` |
| `internal/idempotency/store.go` | `AA0483969B12B058B42DB70A9763501B60ED055AFB91E5ECD6AA7F5C45FDD657` |
| `internal/idempotency/store_test.go` | `3A6EF9313839ED19BE28D4FB635C016DF51F6C58213DDAE88D93A2A65CC49BE7` |

固定上游亦在开始检查时匹配：

- design r40：`B8D2171DEF5DF3E0B47B521C9898E9B19F77EFC44388A0B461ED34B774416336`
- stories r16：`057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3`
- Design Review PASS：`EB1AB24C659CB8281FD4FC2058470C88ACA7FACFC33D1665900C091CAE9A46FB`
- test-plan r18：`7A782D373684FD65114BBDA297FF4FF22A237F1868F5B1534E5FECEA2C9F88D4`
- Test-plan Review PASS：`3A241B10EFB64EFA7A09E452F20F22CB84051AAB7D3503E511C5298451AF794C`
- Planning R03 PASS：`B0839C80C59B1761D33F362207D496F0E8E7D419D36CD0EA28FDAA4DC1E981E1`

## 3. 两轮实际检查

### Round 1：固定修订直接检查

1. 完整复核 Reviewer Role、`add-code-review`、review checklist、abstraction、core/go-service coding、no-hardcode、安全规则及产品安全红线。
2. 逐行检查本轮变化的 `password_reset_store.go` 与 `password_reset_store_test.go`，重点核对：
   - 历史 operation A 与当前 permanent root B/current temporary C 分权；
   - recursive SQL 的方向、base/target、cycle 防护、root 唯一性和 `QueryRow` 基数；
   - current temporary/permanent projection 的 exact binding、generation、delivery、settlement和 lineage；
   - 普通 stale/ineligible、same-key replay、integrity drift/hold 的判定顺序；
   - hold operation pointer 的 same-user 证明；
   - hold 与去敏 Audit 的同语句原子写入；
   - transient query/User lock/commit 失败的零新 hold；
   - 测试是否直接驱动 Go 分类与写入行为。
3. 回归检查六个未改实现/测试对象与 `0008`：六字段 Idempotency、temporary credential、operation/job/claim 约束、preflight 分类及公共合同均未漂移。
4. 按真实 AST 函数体检查安全关键函数复杂度；现有 AST 测试覆盖所有函数且上限为 50 行。

### Round 2：上游合同与有界消费者深审

沿 design r40、stories r16、Design Review、test-plan r18、Planning R03、R04 根因，以及 `password_reset.go` 的真实 generation writer、`0002/0008` schema和 Idempotency 直接合同核对：

- current capability root 只证明当前能力；immutable legal history 独立证明 A 的合法历史；
- current root 合法替换不是 drift 的充分条件；
- password-reset credential generation 由最高 generation `+1` 产生；`generation_ordinal` 的密度检查不会把现役合法 writer 的正常序列误判为 drift，异常缺口则应 fail closed；
- `temporary_credentials_settled_current_idx` 保证同 User/purpose 至多一个 active/expired current credential；operation/credential ID 唯一性使当前 CTE 与 `QueryRow` 不产生歧义多行；
- operation↔claim、operation↔job 双向 deferred FK 与 User counter constraint 保持五对象 commit-time 全有或全无；本轮 lineage 分类没有削弱这些约束；
- 未扩展到 S10.3 Gate、S10.1 coordinator 或 S10.4 worker/composition 的完成性，也未从当前实现反向降低设计预期。

## 4. 原 finding 定点复审

### 4.1 `IDENTITY-S10.2-REV-R01-F003` — CLOSED

#### 合法历史与当前能力已正确分权

- `password_reset_store.go:980-1003` 的 `permanentRootAuthority` 独立验证当前永久能力：User source pair、当前 root operation、exact consumed credential、同 User、succeeded/permanent-successor authority、accepted delivery和 current permanent generation。
- `password_reset_store.go:1052-1094` 不再要求当前永久 root 必须等于历史 A：
  - A 仍是 current root 时允许 A 后普通改密；
  - 当前 root 为后续合法 B 时，要求 B 的 base/target 位于 A successor 之后；
  - 当前为 temporary C 时，同时要求 B permanent root 合法、C temporary capability 合法、C lineage root base 等于 current permanent generation，且 C target 等于 User auth generation。
- 历史 A 自身仍先经过 `passwordResetOperationAuthorityValid`、accepted delivery和 exact operation↔credential 验证；修订没有用“root 不同即普通 stale”放宽真实漂移。
- `password_reset_store.go:1096-1117` 的 superseded/recovery-invalidated 普通 stale 白名单仍要求相应合法 successor/current mode 证据；无法证明时保留默认 drift。

#### SQL 基数、递归与 generation

- `password_reset_store.go:1496-1740` 从唯一 current active/expired password-reset credential 反向递归 predecessor；每层要求同 User、exact replaced credential、base=parent target、parent succeeded/temporary-active/accepted，`operation_path` 阻断 cycle。
- schema 的 partial unique current index和 operation/credential唯一键保证起点与每层 predecessor 都是单值；没有发现多 root 或 `QueryRow` 多行放行路径。
- `password_reset.go:526-554` 的真实 writer总是取最高 credential generation 后 `+1`；current temporary 的 generation 与 ordinal 不一致表示缺失/异常历史，按 fail closed 处理符合 current projection 合同。

#### 分类与写入副作用

- `password_reset_store_test.go:813-924` 直接驱动：
  - A 后正常改密；
  - A→B current permanent；
  - A→B→current temporary C；
  - 每例 A same-key replay仍返回原 succeeded 终态；
  - 新 key引用旧 A返回 `ErrPasswordResetGateClosed`，且 Exec 写入为 0。
- `password_reset_store_test.go:1102-1271` 按 CP-01、T01～T05、P01～P04 对 current mode、pointer/行/state/source/generation、permanent generation/source pair/exact binding 做单变异，直接调用 `BeginOrClassifyCandidate`，验证 drift只产生一次原子 hold/Audit写入，且参数不包含 secret digest或 delivered email；不是仅做 SQL 字符串断言。
- `password_reset_store_test.go:1273-1345` 验证只有已证明同 User 的 predecessor 才写 hold operation ID；跨 User/缺行写 NULL，避免错误 FK 反向拒绝安全 hold。
- `password_reset_store_test.go:1347-1420` 验证 User lock、lineage query和 caller commit 瞬时故障不产生 drift写入；合法 stale路径本身零写入。
- `password_reset_store.go:1438-1485` 用同一个 CTE/Exec 原子更新 durable hold并追加去敏 Audit；任一部分失败时整条语句失败，不存在只有 hold或只有 Audit 的已提交结果。

R04 的确定性反例已由本轮 Reviewer 独立重跑并通过，因此 F003 关闭。

### 4.2 `IDENTITY-S10.2-REV-R01-F004` — CLOSED，无回归

- `migrations/0008_password_reset_durable.sql` 与 R04 固定 SHA 相同。
- preflight仍阻断全部 processing/uncertain。
- succeeded只精确放行封闭 administrator-create legacy `{outcome,resource_id}` 或 `identity-administrator-create-response/v1`；外层 `IS NOT TRUE` 继续阻断 NULL 三值逻辑误放行。
- password-reset、未知 result、missing/extra/type drift、混合异常历史与 legacy current credential继续阻断。
- 静态 preflight矩阵与 migration package测试通过；未发现通用 Idempotency 行被错误强制关联 PasswordResetOperation。

F004 的代码/schema层结论保持 CLOSED；真实 PostgreSQL 18 apply仍归 S1.6 专项，未由本评审替代。

## 5. 新增 findings

无。

## 6. 待决定与非阻断建议

### 待决定

无新的设计、公共合同或第九文件决定项。

### 非阻断建议

后续 S1.6 的真实 PostgreSQL 18 fixture应复用 r18 的完整 CP-P04与 preflight矩阵，实际执行 deferred FK/constraint trigger、JSONB/NULL三值逻辑、current projection单字段漂移和 migration混合历史；不得以本轮组件测试替代真实数据库证据。

## 7. 实际命令与结果

以下 Reviewer 命令均在仓库根目录执行。测试子进程显式移除了 `HAN_TEST_POSTGRES_DSN`，未读取、输出或使用其值。

| 命令 / 检查 | Reviewer 结果 |
|---|---|
| `git branch --show-current` / `git rev-parse HEAD` | `main` / 固定 HEAD |
| 八对象与固定上游 SHA256 | 全部匹配 |
| `go test ./internal/identity -run TestPasswordResetStoreLegalHistoryStaleAfterRootReplacement -count=1` | PASS |
| `go test ./internal/migration ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store\|Durable\|Operation\|Job)\|TestTemporaryCredential\|TestIdempotencyPasswordReset\|TestMigration" -count=1` | PASS；该过滤式在 `internal/migration` 中无匹配测试 |
| `go test ./internal/migration -count=1` | PASS |
| `go test -race ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store\|Durable\|Operation\|Job)\|TestTemporaryCredential\|TestIdempotencyPasswordReset" -count=1` | PASS |
| 八文件 scoped `git diff --check` | PASS |
| staged 状态 | 空 |
| 报告 scoped `git diff --check` | PASS |

Lead提供的定向普通/race及 gofmt、mod verify、vet、build、contract、全仓race、全仓test PASS属于外部现场证据；Reviewer没有把它们冒充本轮自跑。本轮独立自跑仅为上表所列命令。

## 8. 未验证限制

按授权未执行：

- 真实 PostgreSQL 18或 integration tag；
- `0008` 对空库、合法管理员历史、legacy password-reset、NULL/类型漂移和混合历史的真实 apply；
- deferred FK/constraint trigger、JSONB谓词及 commit-time五对象原子性的真实数据库实证；
- multi-connection、死锁/serialization、同 User并发 generation；
- commit-ack-loss；
- SMTP/TLS、部署、Host、浏览器/E2E、容量和 production readiness。

因此本 PASS 不等于 migration专项、部署或E2E通过。

## 9. 状态边界

- `identity-auth-S10.2` 仍由 Lead依据任务级全部证据决定是否关闭；本报告不直接改 task状态。
- `identity-auth-S10.3` 不由本报告启动。
- `identity-auth-S10.1` implementation stopped及 `IDENTITY-S10.1-REV-R01-F002` OPEN状态不变。
- 未修改八个被审对象、Tasks/Plan、设计、旧 Review、Manifest、checkpoint或任何任务状态。
