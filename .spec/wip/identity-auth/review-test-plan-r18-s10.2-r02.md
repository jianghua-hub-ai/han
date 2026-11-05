# identity-auth test-plan r18 定点复审 R02

- Reviewer：Gibbs（原非作者独立 Reviewer）
- 日期：2026-10-24
- objectType：design（test-plan）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 主对象：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
- Revision：`identity-auth-test-plan-ir01-r18`
- 复审目标：定点复审 R01 F001/F002，并回归关键状态、测试边界与既有覆盖
- 结论：**F001 CLOSED；F002 CLOSED；无新增 finding；Test-plan Review Gate: PASS**

本报告只评价固定 test-plan r18 是否关闭原 test-plan finding，并继续构成 planned-not-executed 的稳定测试设计输入。它不关闭代码 `IDENTITY-S10.2-REV-R01-F003`，不关闭 S10.2，不恢复 S10.1，也不启动 S10.3、S10.4、B6D/S11.1、Task-Planner 或测试执行。

## 1. 固定对象与写前完整性

| 对象 | Revision / 用途 | 写前 SHA256 | 结果 |
|---|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `identity-auth-test-plan-ir01-r18` | `7A782D373684FD65114BBDA297FF4FF22A237F1868F5B1534E5FECEA2C9F88D4` | MATCH |
| `.spec/wip/identity-auth/review-test-plan-r17-s10.2-r01.md` | 原 R01 / F001-F002 | `7F429C034A6651CEC1B2E0B25C24C419A03093CB483DE7B14680A64EDE9B3039` | MATCH |
| `.spec/wip/identity-auth/review-test-plan-r16-s10.1-r03.md` | 现役 source review / PASS | `E9ADD4D11B36A36D2AE5711C796E19CCA48FE720CD298E8AACC85E320F6002EF` | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r40` | `B8D2171DEF5DF3E0B47B521C9898E9B19F77EFC44388A0B461ED34B774416336` | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r16` | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` | MATCH |
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r27` | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` | MATCH |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `identity-auth-boundary-ir01-r13` | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` | MATCH |
| `.add/project/knowledge/decisions/identity.md` | accepted ADR-0023 所在文件 | `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978` | MATCH |

目标报告写前不存在；分支、HEAD 匹配；暂存区为空。tasks/Plan 只作状态和所有权证据，读取时 SHA256 分别为 `009E03A85B885F466622A605243CD9E9D721693F8BF87F766D26FB50BD83B003`、`0BFFCD302924692C21B2277CC14DDD09A0A1A2B8D938D71C176A1FF6E5344182`。

## 2. 两轮实际检查范围

### Round 1：F001/F002 固定对象直接复审

1. 复用并执行同版本 Reviewer Role、`add-code-review` design 分支、`add-test-design`、评审清单、抽象原则、设计写作规范和安全红线。
2. 检查 r18 metadata、来源表、修订记录、成功证据、I-PRD-03/05/12/16～18、数据夹具、任务映射、执行建议、readiness 与阶段结论。
3. 逐项检查 `I-PRD-05.CP-*` 的前置、单字段/单行变异、预期分类、hold、去敏 Audit、零写入证据和清理边界。
4. 静态核对 I-PRD-01～20 仍为 20 个唯一主定义，无缺号或重复。

### Round 2：有界上游与真实所有权回归

1. 对照 design r40、stories r16、architecture r27、boundary r13、ADR-0023、Design R01 和 r16 Test-plan R03，确认 history/current 权威、合法 root replacement 与 fail-closed 合同未改变。
2. 有界读取 tasks/Plan，核对 S10.1、S10.2、S10.3、S10.4、S11.1 的三态、顺序和八文件所有权；未从当前实现反推测试预期。
3. 静态核对现有 `0008`/model/store 字段能够表达 r18 矩阵引用的 mode、temporary pointer/source/generation 与 permanent source pair；未进行代码正确性评审。
4. 未运行 Go、PostgreSQL、SMTP、部署或 E2E。

## 3. 原 findings 定点复审

### `IDENTITY-TESTPLAN-R17-S10.2-R01-F001` — CLOSED

原问题是 r17 metadata 仍把失败的 r15 R02 作为现役 test-plan source，而没有引用已经取代它的 r16 R03 PASS。

r18 已完成闭环：

1. `test-plan.md:20-22` 精确登记：
   - `source_test_plan_review: review-test-plan-r16-s10.1-r03`
   - SHA256 `E9ADD4D11B36A36D2AE5711C796E19CCA48FE720CD298E8AACC85E320F6002EF`
   - result `pass`
2. `test-plan.md:50` 明确 r16 R03 是现役 source，并准确记录 F003/F004 CLOSED、无新增 finding、Test-plan Review Gate PASS。
3. `test-plan.md:52-53` 把 r14 R01、r15 R02 明确标为历史来源，并声明不作为现役 source。
4. `test-plan.md:34,84,780` 的修订说明与阶段结论使用同一来源语义，没有再出现相互冲突的 current review。

复审条件全部满足，**F001 CLOSED**。

### `IDENTITY-TESTPLAN-R17-S10.2-R01-F002` — CLOSED

原问题是 r17 只概括 current projection 关系，未把 mode、temporary current、permanent source pair 展开为可直接生成 fixture 的逐字段坏证据矩阵。

r18 已形成可执行闭环：

1. **统一基线与后置**：`test-plan.md:420` 为 temporary/permanent 分别定义完整合法基线，要求每例只改变一个字段或删除一行；真实坏例必须 `integrity_drift`、同事务留下 durable hold 和去敏 Audit，并逐对象证明除 hold/Audit 外候选与能力零写入。
2. **credential mode**：`CP-01`（`test-plan.md:424`）覆盖 temporary/permanent mode 交叉、空/未知 mode 的独立单变异。
3. **current temporary projection**：
   - `CP-T01`：operation 的 current credential pointer missing/conflict；
   - `CP-T02`：current exact credential 行缺失；
   - `CP-T03`：current credential state 非 active；
   - `CP-T04`：`source_operation_id` missing/cross-operation/cross-user；
   - `CP-T05`：credential generation/auth_generation、source operation target generation 与 User generation 的逐字段冲突。
4. **current permanent projection**：
   - `CP-P01`：permanent generation missing/conflict；
   - `CP-P02/P03`：source operation/source credential 的单边缺失；
   - `CP-P04`：source pair 跨 root/跨 User、exact credential/source operation/settlement 不一致。
5. **每例证据与清理**：`test-plan.md:422-433` 每行均给出前置、单变异、`integrity_drift`/hold、去敏 Audit、全对象快照和逐例 rollback/drop；明确不通过普通请求清 hold。
6. **合法非 drift 对照**：`CP-N01`（`test-plan.md:434`）要求 A 与 current B/C 两侧均合法时为 `legal_history_stale`/普通 stale-ineligible，same-key replay 保持原终态，hold/drift Audit/候选/能力写入全部为零。
7. **瞬时故障对照**：`CP-F01`（`test-plan.md:435`）不修改持久字段，分别注入 query、User lock、commit 瞬时故障；只 fail closed，禁止持久 drift、新 hold 或 drift Audit。
8. **跨层复用**：`test-plan.md:389,391,398,402-404,437,601-602,625,636,681,686,735` 将同一矩阵映射到 classifier、PasswordChange、migration/preflight、混跑/readiness、去敏 Audit、deterministic harness、S10.2 和 S1.6，没有另造 task/schema/migration。

矩阵共 12 个唯一子例 ID：`CP-01`、`CP-T01`～`CP-T05`、`CP-P01`～`CP-P04`、`CP-N01`、`CP-F01`。它们已满足直接生成 fixture、单变异、预期、证据和清理要求，**F002 CLOSED**。

## 4. 关键回归

1. **A/B/C 与 replay/stale**：`test-plan.md:388,398,408-416,434,600,734` 继续覆盖 A 后正常改密、A→B current permanent、A→B→current temporary C；same-key replay 返回 A 原终态，新 key 旧 A 为 `legal_history_stale`/普通 stale-ineligible，并断言候选、能力和 hold 零写入。
2. **瞬时故障**：query/User lock/commit 继续只 fail closed，不新增 hold、drift Audit、claim、operation、credential、job、generation 或能力写入。
3. **规划状态**：metadata 和全文继续为 `planned-not-executed`；没有把 Design/Test-plan Review PASS 写成行为测试、PG18、SMTP、部署或 E2E 已通过。
4. **finding 边界**：代码 `IDENTITY-S10.2-REV-R01-F003` 继续 OPEN；F006 保持 CLOSED；代码 `IDENTITY-S10.1-REV-R01-F002` 继续 Major OPEN。
5. **任务状态**：S10.1 implementation stopped；S10.2 active 但 implementation paused / must not close；S10.3、S10.4、S11.1 todo/not started。
6. **范围守恒**：r18 明确没有新增 task、schema、migration、ADR、boundary、公共合同或 Case/Run，也未扩大 S10.2 八文件所有权。
7. **专项边界**：真实 PG18、S5.3 local real-protocol SMTP、S3.4 approved-provider/external-receipt E2E、S2.2 性能仍分层独立，性能阈值保持 TBD。

## 5. 新增 findings

无。CRITICAL、HIGH、MEDIUM、LOW 新增 finding 均为 0。

未发现 r18 引入新的架构、安全、迁移、恢复、并发、状态所有权或消费者契约风险。

## 6. 待决定与非阻断建议

### 待决定

无新的产品、架构或 ADR 决定。

### 非阻断建议

后续实施和代码复审引用子例时，宜显式列出 `CP-01`、`CP-T01`～`CP-T05`、`CP-P01`～`CP-P04`、`CP-N01`、`CP-F01`，避免把正文中的范围简写误读为只包含首尾编号。该建议不阻断本 Gate，也不要求修改 task 或创建 Case/Run。

## 7. Test-plan Review Gate

- R01 F001：CLOSED
- R01 F002：CLOSED
- 新增 finding：0
- 关键回归：PASS

**Test-plan Review Gate: PASS**

理由：r18 已准确修复现役 review 来源，并把 current capability projection 展开为可直接生成 fixture 的逐字段单变异矩阵；合法 root replacement 与真实 drift、瞬时依赖故障之间的分类和写入后置均清晰、无冲突。既有 A/B/C、状态、专项与公共合同边界没有退化。

该 PASS 只允许 Lead 现场复核后决定后续任务影响同步或受控返工；它不关闭代码 F003/F002，不关闭 S10.2，不恢复 S10.1，也不授权启动 S10.3/S10.4/S11.1 或任何测试。

## 8. 实际命令

执行了：

- `git branch --show-current`
- `git rev-parse HEAD`
- `git diff --cached --name-only`
- `Test-Path`：确认目标报告写前不存在
- `Get-FileHash -Algorithm SHA256`：固定主对象、原报告、r16 R03、design/stories、architecture/boundary/ADR、tasks/Plan
- `Get-Content -Encoding utf8` / `Select-String -Encoding utf8`：定点读取 r18、原 R01、上游和 tasks/Plan
- PowerShell 静态计数：
  - I-PRD 主定义 20 个、唯一 20 个、无缺号；
  - `I-PRD-05.CP-*` 共 12 个唯一子例 ID
- 对 `0008`、model/store 进行只读字段/所有权核对

未运行 Go、PostgreSQL、SMTP、部署或 E2E；未读取或使用敏感环境变量。

## 9. 未验证风险与限制

1. 本轮没有执行 r18 规划的任何 fixture 或行为测试。
2. 未用真实 PostgreSQL 验证非法行能否由约束直接拒绝、由 migration preflight 阻断，或由 classifier 写入 hold；具体分层仍须按 I-PRD/S1.6 执行。
3. 未验证真实事务锁、commit actual state、多连接、多 worker、migration、混跑或不可安全回退。
4. 未验证 SMTP/TLS、provider 受理、外部收件、真实 Host、部署、性能、容量、可观测平台或 E2E。
5. 未创建或执行 Case/Run，未验证真实审批、责任人、数据或清理执行。
6. tasks/Plan 当前来源文字仍需由 Lead 在后续影响同步中核对；本 Reviewer没有修改任务状态或规划对象。

## 10. 复审与状态边界

本轮没有遗留 test-plan 阻断 finding，无须再次进行 test-plan 返工复审。后续仍必须：

1. 由 Lead 复核固定哈希、报告、唯一写入和工作树边界；
2. 在既有 S10.2 授权和八文件范围内建立矩阵对应的失败证据；若需扩大 schema、migration、公共合同或写集，立即停止并回上游；
3. 实现返工后由原代码 Reviewer 独立复审代码 F003；
4. 只有代码和任务各自 Gate 满足后，才由 Lead 决定任务状态和后续启动。

## 11. 写后验证

- 固定对象写后 SHA256 应与 §1 一致。
- 对本新增报告执行 scoped `git diff --check`，要求无 whitespace error。
- 本轮唯一写入只允许新增本报告；报告最终 SHA256 不嵌入正文，以完成返回中的最终计算值为准。
