# identity-auth durable lineage Design 独立复审 R01

- Reviewer：Gibbs（非作者独立 Reviewer）
- 日期：2026-10-24
- objectType：design（feature design + stories）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 目标：复审 r40/r16 是否消除代码 R04 `IDENTITY-S10.2-REV-R01-F003` 暴露的 durable lineage 设计歧义
- 结论：**设计层歧义 CLOSED；无新增 design finding；Design Review Gate: PASS**

本结论只表示固定设计增量足以作为后续任务影响核对、test-plan 修订和 S10.2 实现返工的稳定输入。它不关闭代码 finding `IDENTITY-S10.2-REV-R01-F003`，不关闭 S10.2，不恢复 S10.1，不启动 S10.3/B6D，也不表示实现、测试、部署、E2E 或生产就绪。

## 1. 固定对象与完整性

| 对象 | Revision | 开始 SHA256 | 结果 |
|---|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r40` | `B8D2171DEF5DF3E0B47B521C9898E9B19F77EFC44388A0B461ED34B774416336` | 匹配 |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r16` | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` | 匹配；`source_design=identity-auth-design-ir01-r40` |

开始时分支、HEAD、两个固定 SHA 均匹配，暂存区为空，目标报告不存在，具备下结论条件。

有界消费者八文件也与 R04 基线全部匹配：

| 文件 | SHA256 |
|---|---|
| `migrations/0008_password_reset_durable.sql` | `37FB3077709462D62C763FE87D4E04B4BDA4CFC498C14C2860B27429D5036980` |
| `internal/identity/model.go` | `4D17E1DBC264C69BAFE3330619098F0C57AD703EC5EA5E611BCE8E2BB548B5C7` |
| `internal/identity/temporary_credential.go` | `66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5` |
| `internal/identity/temporary_credential_test.go` | `659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D` |
| `internal/identity/password_reset_store.go` | `BD6B7CAC16842260A88EE1D3F0E3EA2C927D8FD4DB96F8F09DE2C0E29AFA5CA2` |
| `internal/identity/password_reset_store_test.go` | `0544938226A422077A5FFD3B22582B0E6EE0EE81DA6320EC7EBEA1C26E3290A9` |
| `internal/idempotency/store.go` | `AA0483969B12B058B42DB70A9763501B60ED055AFB91E5ECD6AA7F5C45FDD657` |
| `internal/idempotency/store_test.go` | `3A6EF9313839ED19BE28D4FB635C016DF51F6C58213DDAE88D93A2A65CC49BE7` |

## 2. 两轮实际检查范围

### Round 1：固定 design/stories 直接检查

完整读取 Reviewer Role、`add-code-review` design 分支、评审清单、抽象原则、设计写作规范、决策指南、产品安全红线，以及 go-service README 和直接适用的 coding/no-hardcode/security 规则。直接检查：

1. revision、source、draft/未批准状态和 F003/F006/S10.1/S10.2/S10.3 状态边界；
2. current capability root 与 immutable legal history 的定义、owner 和职责分离；
3. A 后正常改密、A→B current permanent、A→B→current temporary C 的状态矩阵；
4. same-key replay、新 key stale/ineligible、integrity_drift/hold 的判定顺序；
5. 查询、锁、commit 等瞬时故障是否只 fail closed 而不制造 durable hold；
6. existing schema/公共 REST/错误/权限/ADR 不变声明；
7. stories S10/S11 与公共验收是否完整同步 design r40；
8. 验证矩阵是否要求行为断言，而非仅 SQL 字符串迎合。

### Round 2：上游依据与真实消费者有界深审

读取并交叉核对：

- R04 代码报告及其 F003 根因/停止点；
- r39 Design R03，确认 F006 CLOSED 与既有 Design PASS 不被重开；
- boundary r13、architecture r27、accepted ADR-0023 与决策索引；
- requirements 中 US-IDENTITY-03.10/03.11；
- test-plan r16 的现有 I-PRD 场景与任务映射；
- tasks/plan 中 S10.1、S10.2、S10.3/B6D 的所有权和顺序；
- `0008`、model、temporary credential、password-reset store/test、idempotency store/test 的静态字段和直接消费关系。

本轮没有扩展到 S10.3 三消费者实现、S10.1 coordinator、S10.4 worker/composition 的代码完整性。

## 3. R04 F003 设计歧义复审

### `IDENTITY-S10.2-REV-R01-F003` — 设计层 CLOSED；代码层仍 OPEN

R04 的确定性根因是旧分类把“当前 permanent root 必须仍等于历史 A”作为普通 stale/ineligible 的前提，导致合法后续 reset B 替换 root 后，旧 succeeded A 被误判为 drift，并可能写入全局 durable hold。

r40/r16 已充分消除该歧义：

1. **权威分离明确**：`design.md:2147-2159` 定义 current capability root 只证明当前能力，immutable legal history 独立证明历史 operation 的合法完成；后续正常改密或 reset 替换 current root 不得改写旧历史。
2. **判定顺序可实施**：`design.md:2163-2173` 要求先验证 A 自身 operation、exact credential、delivery、settlement、generation、Idempotency identity 和 resend predecessor/replaced，再独立验证 current projection；后一步不得推翻已证明的历史。
3. **请求语义无混淆**：同 key 精确 replay 返回 A 原 terminal snapshot；新 key 继续引用已被合法 B/C 替换的 A 时为普通 `stale/ineligible`，且零 claim/operation/credential/job/generation/hold 写入；只有具体持久证据缺失或冲突才进入 drift/hold。
4. **A/B/C 矩阵完整**：`design.md:2177-2186` 分别覆盖 A 后正常改密、A→B permanent root 替换、A→B→current temporary C 和真实证据冲突；明确 `current root != A` 不是 drift 的充分条件。
5. **故障与恢复边界正确**：`design.md:2165` 把读取、锁、commit 暂时失败限定为既有 uncertain/internal fail-closed 映射，不得把单次查询失败持久化为新的 drift。
6. **接口与验证闭环**：`design.md:2225-2227` 固定 classifier 至少区分 `current_eligible`、`legal_history_stale`、`integrity_drift`、`unresolved`；`design.md:2285-2288,3231-3232,3441-3442` 要求使用真实 store 行为覆盖 A→B/A→B→C，并逐字段破坏证据作为 hold 对照，禁止只做 SQL 字符串断言。
7. **Story 同步完整**：`stories.md:345-356,384-392,412` 同步两类权威、A/B/C、same-key/new-key、瞬时故障、零写入和逐字段 drift 对照，没有把实现现状反写为设计正确性。

因此，R04 F003 暴露的“合法 current root 替换会否使旧 succeeded operation 变成漂移”在设计层已有唯一答案：**不会**。旧 A 的 immutable legal history 继续有效；是否可作为新候选上下文由当前能力判断，通常为 stale/ineligible；真实 current projection 或历史证据漂移仍 fail closed。

该关闭不等于代码修复。R04 所审实现仍是固定旧基线，代码 `IDENTITY-S10.2-REV-R01-F003` 必须在后续受控返工和原代码 Reviewer 独立复审后才可能关闭。

## 4. Schema、架构与公共合同可实施性

### 4.1 现有 `0008` 字段足以承载 r40 语义

静态核对表明无需为 A→B 新增专用 FK：

- current permanent projection：`users.permanent_password_generation`、`permanent_password_source_operation_id`、`permanent_password_source_credential_id`（`0008:143-148,159-174`）；
- current temporary projection：User current operation、TemporaryCredential 的 `auth_generation/source_operation_id`（`0008:146,212-227`）；
- immutable operation evidence：request kind、predecessor/replaced、base/target/settled/successor generation、delivery、settlement、terminal 状态（`0008:239-291`）；
- exact credential、同 User 与六字段 Idempotency identity 的复合唯一/FK（`0008:292-339`）；
- settlement/delivery/state 合法组合（`0008:340-443`）。

`design.md:2155,2186,2402,3449` 还明确要求在 Idempotency replay/operation lineage 保留期内保留 predecessor、replaced 和 exact new credential 证据；提前删除应被识别为 evidence missing，而不是推测合法后继。

结论：r40 是读取和分类语义澄清，不暗含新列、新状态、新 migration、跨 reset A→B 专用 FK或第九实现文件。若实现时发现必须扩 schema 或公共合同，应停止并返回设计/规划，而不能在 S10.2 内自行扩大范围。

### 4.2 上游一致性

- ADR-0023 仍要求 operation 为安全结果权威、关系不一致时 fail closed、IdempotencyRecord 不得单独放行；r40 只澄清“合法 current root 演进不是关系不一致”，未改变方案 A。
- boundary r13 继续拥有 password-reset operation/Gate/worker、purpose 分流、旧值 settlement 与公共 REST/错误/权限不变合同，没有要求历史 operation 永远等于 current root。
- architecture r27 已声明历史 succeeded replay不得改写后续正常改密或后续 reset 形成的当前能力（`architecture.md:571`），r40 是 feature-level 精确化，不需要重开 architecture 或 ADR。
- requirements 的管理员权限、邮件成功/失败、最新临时凭据和公共行为不变；内部 stale/drift 分类未增加公共错误或客户端字段。

**ADR-0023 consistency check: PASS。** 这不是重新评审或重新接受 ADR。

## 5. F006 回归

`IDENTITY-DESIGN-R37-S10.1-R02-F006` 保持 **CLOSED，未复发**：

- `new_administrator` resend 继续 ADR-0011；
- 普通用户 `password_reset` resend 继续 ADR-0023 operation/job/Gate/worker；
- new key 使用 successor operation/execution generation，same-key replay 零新副作用；
- old active/new pending、唯一发送权、stale candidate 和 succeeded settlement 不变；
- r40 只改变历史/current 分类边界，没有恢复 operation 旁路或改变公共合同。

## 6. Findings

未发现 HIGH、MEDIUM 或 LOW design finding。

未发现需要改变 ADR-0023、boundary、architecture、requirements、公共 REST/错误/权限或 `0008` schema 的新问题。

## 7. 待决定与非阻断建议

### 7.1 待决定

无新的设计决定。用户已经采用“current capability root 只证明当前能力；immutable legal history 独立保存；`current root != A` 不是 drift 充分条件”的方向。

### 7.2 非阻断任务影响建议

1. **S10.2 任务影响核对**：无需新增 task 或扩大八文件写集；应由 Lead/Task-Planner 把 S10.2 的设计来源从 r39 精确更新到 r40/r16，并把 A 后正常改密、A→B permanent root 替换、A→B→current temporary C、逐字段 drift、瞬时 query/commit 失败不写 hold 纳入 task-level TDD/关闭证据。
2. **test-plan 影响核对**：现有 test-plan r16 仍以 r39 为来源，且未显式写出 A→B/A→B→C、`legal_history_stale` 与 current/history 双权威。后续修订应把 `design.md:3231-3232` 的矩阵映射到 I-PRD-02～05/12/16～18 或等价既有场景；继续保持 `planned-not-executed`，不得把当前单元测试或静态 SQL 检查写成已验证。
3. **状态同步**：现有 tasks/plan 正文仍停留在“S10.2 仅启动准备”的历史描述；当前 handoff 与 r40/r16 已明确 S10.2 active 但因设计增量暂停实现。Lead 在恢复任何实现前应先核对任务状态与 owner，不因此启动 S10.3。
4. **代码复审**：S10.2 实现返工后仍由原代码 Reviewer 按 R04 F003 条件复审；Design PASS 不能替代该 Code Review Gate。

## 8. Design Review Gate

**Design Review Gate: PASS**

理由：

- r40/r16 已给出唯一、无冲突且可测试的历史/current 权威分离；
- 合法多代 reset 的旧 succeeded operation 不会仅因 current root 被 B/C 替换而进入 drift/hold；
- current projection 或历史证据的真实缺失、跨 User、generation/delivery/settlement/幂等/lineage 冲突仍 fail closed；
- same-key replay、新 key stale、瞬时故障、恢复动作和零写入后置均已明确；
- 现有 `0008` 字段足以实施，不需要新 schema、ADR、boundary 或公共合同；
- F006 和 r39 已通过合同未被重开。

Gate 交接仅允许 Lead 进入任务/test-plan 影响核对并决定后续受控返工；不构成实施授权。

## 9. 状态边界

- `IDENTITY-S10.2-REV-R01-F003`：**代码层 OPEN**。
- `IDENTITY-S10.1-REV-R01-F002`：继续独立 OPEN。
- S10.1：implementation stopped，不恢复。
- S10.2：active，但本轮实现暂停；不关闭。
- S10.3、S10.4、B6D/S11.1：不得据本报告启动。
- F006：保持 CLOSED。
- 本报告不声明代码、测试、PostgreSQL、SMTP、部署、E2E 或 production readiness 通过。

## 10. 实际命令与静态检查

执行了：

- `git branch --show-current`
- `git rev-parse HEAD`
- `git diff --cached --name-only`
- `Get-FileHash -Algorithm SHA256`：固定 design/stories、上游输入和 R04 八个消费者基线；
- `Get-Content -Encoding utf8` / `Select-String -Encoding utf8`：按章节读取和定位 design、stories、R04、R03、ADR、boundary、architecture、requirements、test-plan、tasks/plan、migration 与 store/test；
- PowerShell 静态断言：revision/source、current/history 定义、A/B/C 矩阵、`current root != A`、F006/F003 状态、公共错误/schema 不变声明，以及 `0008` 所需字段存在性。

未运行 Go、PostgreSQL、SMTP、部署或 E2E；未读取或使用任何敏感环境变量。

## 11. 未验证限制

1. 未执行 r40 对应的新实现或行为测试；当前代码仍为 R04 固定基线，不能据 Design PASS 推断 F003 已修复。
2. 未以真实 PostgreSQL 验证递归查询、约束、事务、锁、commit-ack-loss、多连接或 migration 行为。
3. 未验证真实 SMTP、worker、生产 composition、部署、E2E、容量或性能。
4. test-plan r16 尚未同步 r40 的显式 A/B/C 行为矩阵；本报告只给出后续修订建议。
5. tasks/plan 未由本 Reviewer 修改，当前状态/owner 仍须 Lead 现场对账。

## 12. 复审条件

本 design 增量无须再次返工作 design 复审。后续关闭代码 F003 至少需要：

1. 在既有 S10.2 八文件授权范围内实现 history/current 分离，或在确需扩界时先回 Lead/Task-Planner；
2. 行为测试直接覆盖 A 后正常改密、A→B current permanent、A→B→current temporary C；
3. 每例同时断言 A same-key replay、A 作为新 key 旧上下文的 stale/ineligible、零 hold/零能力写入；
4. 逐字段坏证据仍进入 drift/hold，瞬时 query/commit 失败不制造 durable hold；
5. 由原代码 Reviewer 独立复审并给出 Code Review Gate，随后再由 Lead 决定任务状态。
