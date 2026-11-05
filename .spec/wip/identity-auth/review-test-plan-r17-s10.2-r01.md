# identity-auth test-plan r17 独立评审 R01

- Reviewer：Gibbs（非作者独立 Reviewer）
- 日期：2026-10-24
- objectType：design（test-plan）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 主对象：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
- Revision：`identity-auth-test-plan-ir01-r17`
- 结论：**2 个 MEDIUM finding；Test-plan Review Gate: CHANGES REQUIRED**

本报告只评价固定 r17 是否准确消费 design r40/stories r16/Design R01 PASS，并形成完整、可执行的后续测试合同。它不关闭代码 `IDENTITY-S10.2-REV-R01-F003`，不关闭 S10.2，不恢复 S10.1，也不启动 S10.3、S10.4、B6D/S11.1、Task-Planner 或任何测试执行。

## 1. 固定对象与写前完整性

| 对象 | Revision / 用途 | 写前 SHA256 | 结果 |
|---|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `identity-auth-test-plan-ir01-r17` | `2B8EF73A562B98E00467ECDB274BCD49DF4B6FD53E64417CB42F1DBA86AE765B` | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r40` | `B8D2171DEF5DF3E0B47B521C9898E9B19F77EFC44388A0B461ED34B774416336` | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r16` | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` | MATCH |
| `.spec/wip/identity-auth/review-design-r40-s10.2-r01.md` | Design R01 / PASS | `EB1AB24C659CB8281FD4FC2058470C88ACA7FACFC33D1665900C091CAE9A46FB` | MATCH |
| `.spec/wip/identity-auth/review-test-plan-r16-s10.1-r03.md` | r16 Test-plan R03 / PASS | `E9ADD4D11B36A36D2AE5711C796E19CCA48FE720CD298E8AACC85E320F6002EF` | MATCH |
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r27` | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` | MATCH |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `identity-auth-boundary-ir01-r13` | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` | MATCH |
| `.add/project/knowledge/decisions/identity.md` | accepted ADR-0023 所在文件 | `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978` | MATCH |
| `.add/project/knowledge/decisions/README.md` | 决策索引 | `B51C6265BB2493BE3AA2EE0C32425F29974B38A177018DC5D75F72A12437E14E` | MATCH |

写前报告路径不存在，暂存区为空。现役 tasks/Plan 只作状态与所有权证据，读取时 SHA256 分别为 `009E03A85B885F466622A605243CD9E9D721693F8BF87F766D26FB50BD83B003`、`0BFFCD302924692C21B2277CC14DDD09A0A1A2B8D938D71C176A1FF6E5344182`。

## 2. 两轮实际检查范围

### Round 1：r17 直接检查

1. 完整读取 Reviewer Role、`add-code-review` design 分支、`add-test-design`、评审清单、抽象原则、设计写作规范与产品安全红线。
2. 检查 metadata、revision、source、review、status、修订历史和阶段结论，确认未把 Design PASS 外推为实现或测试通过。
3. 逐项检查 I-PRD-01～20 的主定义、风险映射、覆盖映射、数据夹具、自动/人工边界、清理和任务建议；静态计数为 20 项、20 个唯一 ID、无缺号或重复主定义。
4. 定点检查 A 后正常改密、A→B current permanent、A→B→current temporary C、same-key replay、新 key `legal_history_stale`、合法路径零写入、坏证据 drift/hold、瞬时 query/lock/commit 故障和禁止 sleep。
5. 检查状态边界、性能 TBD、专项分层、零新增 task/schema/migration/ADR/boundary/公共合同/Case/Run。

### Round 2：上游与真实所有权有界深审

1. 沿 design r40、stories r16、Design R01、architecture r27、boundary r13 与 accepted ADR-0023 核对 current capability root / immutable legal history 的两权威合同。
2. 沿 prior test-plan r16 R03 核对现役 Test-plan Gate、F006/F005/S5.2/B6B 历史关闭和未验证边界。
3. 有界读取 tasks/Plan，确认 S10.1 implementation stopped、S10.2 active 但 implementation paused、S10.3/S10.4/S11.1 未启动，以及八文件所有权和专项任务映射未被 r17 扩大。
4. 静态核对 S10.2 八个直接消费者对象仍与 R04 基线一致；未将当前实现反推为设计正确性。
5. 未扩展为代码评审，未执行 Go、PostgreSQL、SMTP、部署或 E2E。

## 3. 已满足的核心合同

r17 已正确覆盖以下内容：

- `test-plan.md:4-15,81` 正确登记 `planned-not-executed`、r40/r16 与 Design R01 PASS，没有宣称代码 F003 已关闭。
- `test-plan.md:385-388,395,409-412,575-577,710-711,756` 覆盖 A 后正常改密、A→B、A→B→temporary C、A same-key replay、新 key stale/ineligible、合法路径的 claim/operation/credential/job/execution generation/auth generation/current credential/Session/hold 零写入。
- `test-plan.md:386-388,412,577` 区分具体持久证据漂移与仅 current root 合法替换；`test-plan.md:386,380,711,756` 要求 query/User lock/commit 瞬时故障 fail closed 但不新增 hold。
- `test-plan.md:575-578` 提供可控 Clock/UUID、故障注入、调用与写入计数，明确禁止用 sleep 或仅检查 SQL 字符串代替行为证据。
- `test-plan.md:601,610-618,657-667` 保持开发卡、真实 PG18、local real-protocol SMTP、provider/E2E、性能与公共契约的独立所有权。
- `test-plan.md:661-667,680-682` 保持 F003 仅设计层 CLOSED、代码层 OPEN；F006 CLOSED；S10.1 stopped；S10.2 active/paused；S10.3/S10.4/S11.1 未启动。

## 4. Findings

### IDENTITY-TESTPLAN-R17-S10.2-R01-F001 — MEDIUM — 当前 source review 仍指向已失败的 r15 R02

- **定位**：`test-plan.md:20-22`。
- **现状**：metadata 登记 `review-test-plan-r15-s10.1-r02`、SHA `F1EC...`、结果 `changes-required`；但本次固定现役基线是 `.spec/wip/identity-auth/review-test-plan-r16-s10.1-r03.md`，SHA `E9ADD4...`，其 Test-plan Review Gate 为 PASS。
- **违反依据**：设计制品必须准确区分当前接受来源和 historical review。r17 明确从 r16 增量形成，却把被后续 R03 取代的失败结果继续作为唯一 `source_test_plan_review`。
- **触发**：后续 Lead、Task-Planner 或实现者只消费 metadata/source chain，而不重新扫描历史报告。
- **影响**：当前稳定基线同时表现为“r16 R03 已 PASS”和“source review changes-required”，无法可靠判断 r17 是基于已接受 test-plan 继续增量，可能错误重开已关闭 finding，或反向忽略真正的 review Gate。
- **修复建议**：把当前 source review 精确更新为 r16 R03 的对象名、SHA256 和 PASS；若需保留 r14 R01/r15 R02，明确标为 historical，而不是现役 source。
- **证据**：
  - `test-plan.md:20-22`：当前错误 source；
  - `review-test-plan-r16-s10.1-r03.md:1-8,100-119`：固定 r16 R03 对 F003/F004 CLOSED、无新增 finding并给出 PASS；
  - r16 R03 固定 SHA256 已在 §1 核为 MATCH。
- **复审条件**：metadata/current-source 与正文的 r16 R03 PASS 基线一致，旧 R01/R02 只保留历史追溯。

### IDENTITY-TESTPLAN-R17-S10.2-R01-F002 — MEDIUM — current projection 的坏证据矩阵未达到逐字段可执行

- **定位**：`test-plan.md:386-388,412,577,710-711`。
- **现状**：r17 已逐项列出历史 A 的 operation、exact credential、generation、delivery、settlement、Idempotency identity、predecessor/replaced，并有“只改变 current root 且两侧合法”的非 drift 对照；但 current capability projection 的实际权威字段没有同等展开：
  - `users.credential_mode`；
  - current temporary credential 指针、state、`source_operation_id` 与 generation 的交叉自洽；
  - current permanent 的 generation、`permanent_password_source_operation_id`、`permanent_password_source_credential_id` 的成对存在和 exact binding。
- **违反依据**：`design.md:2151-2168,2172,2184,2418-2440` 把上述字段定义为 current projection 的独立权威，并要求 A 或 current projection 的正向可证明不一致进入 `integrity_drift`/hold。Design R01 也明确要求逐字段破坏历史和 current projection，而不是只验证合法 root inequality。
- **触发**：例如 `credential_mode=temporary` 但 current temporary 行缺失/非 active/指向别的 operation，或 permanent source operation/credential 只改一侧；现有用例作者可只实现历史字段破坏和“合法 current root 替换”，仍满足当前文字表面要求。
- **影响**：真实 current projection drift 可能缺少确定性失败证据，后续 classifier 返工可能把内部冲突降为普通 stale/ineligible，绕过应有 durable hold；这正是本轮要求保留的 fail-closed 对照。
- **修复建议**：在既有 I-PRD-03/05/12/16～18 范围内补一张 current projection 逐字段 missing/conflict 表或等价子用例，逐项写明前置、单字段变异、预期 `integrity_drift`/hold、零候选/能力写入、Audit 和清理；同时保留“两侧均合法但 root 不同”为唯一非 drift 对照。无需新增 task、schema、migration 或公共合同。
- **证据**：
  - `test-plan.md:388,577` 的当前逐字段清单没有上述 current projection 字段；
  - `test-plan.md:386,388` 只概括“current projection 关系可变/只改变 current root”，不足以唯一约束坏投影变体；
  - `design.md:2151-2158,2168-2172` 定义 current projection owner、验证顺序和 drift 条件；
  - `design.md:2418-2440` 给出具体持久字段。
- **复审条件**：坏投影字段矩阵可直接生成测试数据，并能分别证明合法 root replacement 不写 hold、真实 current projection drift 必须 fail closed 且按合同写 hold。

## 5. 回归与新增风险检查

1. I-PRD-01～20 各且仅各有一个主定义；除 F002 所述可执行粒度外，其他已确认覆盖未退化。
2. `planned-not-executed`、性能阈值 TBD、真实环境未验证、零新增 task/Case/Run 均保持。
3. F006、S3.2/F005、S5.2 与 B6B/S8.1 的既有关闭未被重开；真实容量、平台、部署、SMTP/provider、PG18、E2E 仍独立未验证。
4. 公共 REST、错误、权限、ADR-0023、boundary 与 schema/migration 边界未被改写。
5. 未发现除 F001/F002 外的新架构、安全、迁移、恢复、并发或状态所有权 finding。

## 6. 待决定与非阻断建议

### 待决定

无新的产品、架构或 ADR 决定。两项 finding 都是既定来源和既定测试合同的文档收敛问题。

### 非阻断建议

修订时可为 A/B/C、current projection drift 和 transient failure 子用例分配稳定子编号，方便 S10.2 TDD、代码复审和后续 PG18 专项引用；不应据此创建新 task 或 Case/Run。

## 7. Test-plan Review Gate

- MEDIUM：2
- HIGH / CRITICAL：0
- 其他新增 finding：0

**Test-plan Review Gate: CHANGES REQUIRED**

理由：核心 A/B/C 与 history/current 分权方向已经进入 r17，但现役 source review 仍错误指向已失败 R02，且 current projection 的真实坏证据没有形成逐字段可执行矩阵。两项修订均可在当前 test-plan 内完成，不需要改变架构、公共合同、schema、migration、task 或写集。

## 8. 状态边界

- `IDENTITY-S10.2-REV-R01-F003`：代码层继续 OPEN。
- `IDENTITY-S10.1-REV-R01-F002`：继续 OPEN。
- S10.1：implementation stopped，不恢复。
- S10.2：active，但 implementation paused / must not close。
- S10.3、S10.4、B6D/S11.1：不得据本报告启动。
- F006：保持 CLOSED。
- 本报告不声明任何实现、测试、PostgreSQL、SMTP、部署、E2E 或 production readiness 通过。

## 9. 实际命令

执行了：

- `git branch --show-current`
- `git rev-parse HEAD`
- `git diff --cached --name-only`
- `Test-Path`：确认目标报告写前不存在
- `Get-FileHash -Algorithm SHA256`：固定输入、architecture/boundary/ADR、tasks/Plan 与 S10.2 八个有界消费者
- `Get-Content -Encoding utf8` / `Select-String -Encoding utf8`：读取和定位规则、test-plan、design、stories、两份 review、architecture、boundary、ADR、tasks/Plan 与有界消费者
- PowerShell 静态计数：I-PRD-01～20 共 20 个、唯一 20 个、无缺号

未运行 Go、PostgreSQL、SMTP、部署或 E2E；未读取或使用敏感环境变量。

## 10. 未验证风险与限制

1. 本轮是文档设计评审，没有执行 r17 规划的任何测试。
2. 未用真实 PostgreSQL 验证约束、事务、锁、commit actual state、migration、混跑或不可回退。
3. 未验证真实 SMTP/TLS、provider 受理、外部收件、worker、受控命令或 production composition。
4. 未执行真实 Host、部署、浏览器、安全、性能、容量、可观测平台或 E2E。
5. 未创建或审核 Case/Run，也未核验真实数据、审批、责任人和清理执行。
6. tasks/Plan 仍引用 r39/r16 的旧规划来源；本轮只读核对，没有修改或授权任务影响同步。

## 11. 复审条件

定点复审只需确认：

1. F001 的 current source review 已指向固定 r16 R03 PASS，旧失败 review 仅作 historical；
2. F002 的 current projection 字段已逐项具备 missing/conflict、预期分类、hold/Audit、零写入和清理证据；
3. 既有 A/B/C、same-key/new-key、瞬时故障、状态边界、专项分层和零新增 task/Case/Run 未退化；
4. 固定对象与修订报告哈希无漂移。

## 12. 写后验证

- 固定对象写后 SHA256 应与 §1 一致。
- 对本报告执行 scoped `git diff --check`，要求无 whitespace error。
- 本轮唯一写入只允许新增本报告；报告最终 SHA256 不写入正文，避免自引用改变摘要，以完成返回中的最终计算值为准。
