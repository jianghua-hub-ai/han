# identity password-reset durable Tasks/Plan 独立规划评审 r01

- 日期：2026-10-24
- Reviewer：Gibbs，add-reviewer，非作者独立评审
- objectType：design/planning 复合对象
- 工作区 / 分支 / HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 当前边界：`identity-auth-S10.1` 仍为 `[-]`、implementation stopped；`IDENTITY-S10.1-REV-R01-F002` 仍为 Major OPEN；B6D/S11.1 未启动

## 1. 固定对象与输入

### 1.1 被审对象

| 对象 | 固定 SHA256 | 开始核对 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `ADB19B164F0F544530B814BE9580A3E0C0BE6829583378176A6DF818E84A366C` | 匹配 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `48458CDDD7C0CF0DC0424C8A0B72F478CE8E36A6AE664C34F7B95C14832382F9` | 匹配 |

报告路径在评审开始时不存在。共享工作树存在大量合法既有未提交/未跟踪内容；本评审未清理、格式化、覆盖或回滚这些内容。

### 1.2 固定上游

| 输入 | SHA256 | 开始核对 |
|---|---|---|
| architecture r27 | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` | 匹配 |
| identity boundary r13 | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` | 匹配 |
| identity design r39 | `71EEC596875EE9CBF44CF300C72A4203F5F2C8976522F478A1857D882AE86230` | 匹配 |
| identity stories r15 | `715F5BEB95FC7AF141F7F9ED18D900979921C520D7E11A857AC017F65C481803` | 匹配 |
| identity test-plan r16 | `FE8AB4B420D16C6E74F1BEB4AE7E4D41B03B5D0637CB95EC82B4818A550A6B8D` | 匹配 |
| design Review R03 | `CAE91411FD4D126BF033896186D761756C41AC1E7C8B6810FD36F34F8601B0B2` | 匹配 |
| test-plan Review R03 | `E9ADD4D11B36A36D2AE5711C796E19CCA48FE720CD298E8AACC85E320F6002EF` | 匹配 |
| S10.1 code Review r02 | `02CBAA4A336858ED76D83953BDE988CD2F6378D0B6FB26180E8082A222CBDCFC` | 匹配 |

## 2. 两轮实际检查

### Round 1：固定 Tasks/Plan 直接检查

1. 全文检查 S10/S11 R gap-map、S10.1～S10.4/S11.1 条目、B6C/B6D 外层表和 B6C Context Pack。
2. 静态解析全部 task checkbox：
   - 56 total；
   - 22 completed；
   - 3 active；
   - 31 todo；
   - active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`identity-auth-S10.1`；
   - 未发现重复 task ID。
3. 静态解析外层批次表：56 次引用、56 个唯一 task；无遗漏、重复或额外 task。
4. 静态解析 requires 图：56 个节点、237 条边、0 个未知目标、0 个环。
5. 核对 B6C 顺序与写集：
   - S10.2：8 文件；
   - S10.3：7 文件；
   - S10.1：3 文件；
   - S10.4：8 文件；
   - 四卡写集互斥；
   - S11.1 只与 S10.1 共享 `internal/identity/handler.go`；
   - B6D 在 B6C 四卡前保持未启动。
6. 核对各卡 TDD ②档失败证据、定向普通/race 命令、scoped diff-check、Manifest 七项、停止条件、专项测试与 E2E 分界。
7. 核对规划未硬编码 lease duration、retry backoff、自动重试上限、批量领取上限或连接池预算；这些值继续作为实施前显式运行输入，缺失时 fail closed。

### Round 2：上游合同与真实消费者有界深审

1. 从 stories S10/S11、design r39、test-plan r16 反向追踪 durable schema、operation/job、三消费者 Gate、worker/operator、双 worker lifecycle、resend successor generation 及专项验证责任。
2. 有界核对当前代码接缝：
   - `internal/idempotency/store.go` 仍是通用 existing classification/Begin；
   - Login 尚无 durable reset Gate；
   - PasswordChange 尚未消费 operation settlement；
   - Session resolve 尚无 caller-owned transaction/commit 后 Principal；
   - `cmd/han-server/lifecycle.go` 当前只管理 Article worker；
   - 现役 migration 最大编号为 `0007`，规划中的 `0008`、password-reset Store/worker/operator 尚不存在。
3. 核对专项边界：
   - S1.6：真实 PostgreSQL 18、锁/多连接、commit-ack-loss、migration 混跑/回退；
   - S5.3：local real-protocol SMTP，不发送外部邮件；
   - S3.4：approved-provider、外部收件、真实 Host/部署 E2E；
   - S2.2：性能/容量；
   - 上述证据未被并入 B6C 开发卡关闭条件，也未被写成已执行。
4. 对照 add-task-decompose 的 R 模式两级产出和独立 task 闭环合同，以及 add-batch-plan 的 task-level 关闭与 batch-level Gate 分离合同，发现下列两项阻断问题。

## 3. Findings

### [HIGH] gap-map 不是 Story 粒度的四标签对账，且“已建成”缺当前可定位证据

- finding ID：`IDENTITY-S10.1-PLAN-R01-F001`
- 对象/修订：design/planning；固定 Tasks SHA `ADB19B...A366C`
- 位置：
  - `.add/project/knowledge/tasks/tasks.md:92-106`
  - `.add/project/knowledge/design-workspace/identity-auth/stories.md:327-393`
  - `.codex/skills/add-task-decompose/SKILL.md:53-56,138-163`
- 现状：
  - 上游实际有两个 Story：S10 与 S11；当前表却按 11 个能力/接缝行分别贴标签，并据此得出“已建成 1 / 待做 4 / 需改 5 / 废弃或不适用 1”。
  - 表中没有给 S10、S11 各自一个 Story-level 四标签结论，因此该计数只是能力行计数，不是 R 模式要求的 `Story × 四标签 × 证据` 分布。
  - “S3.2/F005、S5.2、S8.1 已关闭共享能力”这一“已建成”行只引用历史 Review/Lead closure，没有给出当前代码/设计的 `file:line` 证据；旧 task/关闭记录不能单独证明当前符合 r39。
- 触发：Lead、后续 Task-Planner 或实施交接按该 gap-map 判断 S10/S11 是否已完整对账，或用四标签计数检查是否有 Story 遗漏。
- 影响：
  - 无法证明两个 Story 均取得唯一、可复核的整体分类；
  - 能力切片虽然覆盖广，但不能替代 Story-level 完整性证明，可能隐藏未下钻或被重复计数的 Story 行为；
  - “已建成”缺当前证据，无法区分真实现状与历史账本事实。
- 建议：
  1. 为 S10、S11 各登记一个明确 Story-level 四标签结论及当前代码/设计证据；
  2. 可保留现有能力/接缝矩阵作为第二级覆盖说明，但不得再把其行数称为 Story 四标签分布；
  3. 为“已建成”共享能力补当前代码/设计 `file:line`，同时保留原 task/Review 作为历史与关闭证据。
- 验证/复审条件：
  - 新固定修订中可机械得到 S10、S11 各一条且仅一条 Story-level 标签；
  - 标签有当前设计与代码位置证据；
  - 能力矩阵、任务承接和 `1/4/5/1` 等旧行计数不再冒充 Story 分布；
  - 不因修订重开 S3.2/S5.2/S8.1 写集或改变任务三态。

### [HIGH] S10.1 与 S10.4 的任务关闭条件形成隐藏的共同关闭环

- finding ID：`IDENTITY-S10.1-PLAN-R01-F002`
- 对象/修订：design/planning；固定 Tasks/Plan
- 位置：
  - `.add/project/knowledge/tasks/tasks.md:877-879`
  - `.add/project/knowledge/tasks/tasks.md:959-961`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:59`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:651-654`
  - `.codex/skills/add-task-decompose/SKILL.md:53-56,186-190,226-234`
  - `.codex/skills/add-batch-plan/SKILL.md:137-148`
- 现状：
  - S10.1 的“本任务关闭条件”要求 S10.4 完成，并与 S10.4 一起通过 integrated Review、F002 定点复审和 Lead Gate。
  - S10.4 的“本任务关闭条件”又要求与 S10.1 共同完成 F002 定点复审和 integrated Gate。
  - Plan 明确规定任一卡未通过则两卡都不关闭。形式化 requires 图虽无环，叙述性关闭条件却把两个 task 变为只能共同关闭的状态单元。
- 触发：S10.1 实现切片和自身局部测试/Review 已满足，但 S10.4 尚未关闭；或 S10.4 自身成果已满足，而 B6C-3 integrated Gate 尚未完成。
- 影响：
  - 两卡不能按自身行为目标独立完成 §10 task-level 关闭；
  - task 状态与 B6C-3 batch-level integrated Gate 混为同一状态，形成形式化 DAG 未呈现的隐藏闭环；
  - 后续失败归属、独立 Review 结论和 Lead 状态回填不再能精确落到单卡。
- 建议：
  1. 保留 `S10.1 implementation slice → S10.4` 的单向串行交接；
  2. 分别定义 S10.1、S10.4 可独立核证的 task-level 成果、测试、非作者 Review 与 Lead §10 关闭条件；
  3. 将跨卡 integrated Review、代码 finding F002 的最终关闭和 B6C-3 接纳放在 batch-level Gate，不作为两卡互相等待关闭的条件；
  4. F002 在实际代码复审前继续 OPEN，不得用规划修订关闭。
- 验证/复审条件：
  - S10.1 和 S10.4 各自满足自身条件即可单独回填状态；
  - B6C-3 仍保留跨卡集成、Hubble F002 定点复审与 Lead Gate，且不会反向伪造任一卡的局部 PASS；
  - requires 图和叙述性顺序表达一致、无显式或隐藏环；
  - B6D 仍只在 B6C 四卡及 batch Gate 完整闭环后启动。

CRITICAL、MEDIUM、LOW：无其他新增 finding。

## 4. 已确认通过的范围

除 F001/F002 外，本固定版本确认：

- 56 / 22 / 3 / 31 计数成立，active 集合准确；
- 56 个 task 在外层批次表各出现一次，无遗漏或重复；
- 237 条 requires 边目标均存在，形式化 DAG 无环；
- 新增且仅新增 S10.2～S10.4，复用 S10.1/S11.1；
- S10.2～S10.4 的行为分层、文件所有权、TDD 失败证据、定向命令和停止条件总体与上游相符；
- B6C 四卡写集互斥；S11.1 只与 S10.1 共享 `handler.go`，串行延后合理；
- S1.6/S5.3/S3.4/S2.2 专项与外部 E2E 没有被开发卡吞并；
- 真实 provider/外部收件仍归 S3.4，S5.3 仅归 local real-protocol SMTP；
- runtime 参数保持显式输入/TBD，没有被规划硬编码；
- F002 保持 Major OPEN，S10.1 未恢复实施，B6D/S11.1 未启动。

## 5. 待决定与非阻断建议

### 待决定

无新的架构、安全或产品决定需要 Reviewer 代替责任人作出。F001/F002 均为规划产出合同与状态模型问题，应由原 Task-Planner 在既有 r39/r16 方向内修订。

### 非阻断建议

1. 保留当前能力/接缝矩阵作为 Story-level gap-map 下面的二级 traceability 表，可继续复用其中较完整的代码和 test-plan 映射。
2. 在 B6C Context Pack 中分别列“task close”和“B6C-3 batch Gate”，避免后续再次把 task checkbox 与跨卡 finding/Gate 合并。

## 6. 未验证风险与限制

1. 按授权未运行 Go、build、test、race、vet 或 Manifest 七项；本报告只评审规划，不证明未来代码可编译或测试通过。
2. 未运行 PostgreSQL、SMTP/TLS、真实 provider、部署、Host、浏览器或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
3. 未验证尚不存在的 `0008` migration、约束/index、Store、worker、operator、锁序、租约接管、关闭顺序或运行参数的真实行为。
4. 未执行 PlantUML 或运行配置；静态规划检查不能证明未来 production composition 或不可回退部署实际安全。
5. 真实代码只作为 gap-map 与消费者接缝证据，不是本次代码质量复审；代码 finding `IDENTITY-S10.1-REV-R01-F002` 不受本报告关闭。
6. 共享工作树中的其他未提交成果不属于本评审，未对其正确性作结论。

## 7. Planning Review Gate

**Planning Review Gate: CHANGES REQUIRED**

F001 使 R 模式缺少必需的 Story-level gap-map 完整性证明；F002 使两个开发 task 的独立关闭被 batch Gate 隐式耦合。两项均需由原 Task-Planner 修订后，固定新对象 SHA 并交非作者 Reviewer 定点复审。

本 Gate 不关闭代码 F002，不解除 S10.1 implementation stopped，不构成实施授权，不启动 Task-Planner 后续实施角色，也不启动 B6D/S11.1。

## 8. 实际写入

仅新增本报告：`.spec/wip/identity-auth/review-tasks-plan-s10.1-r01.md`。

未修改 Tasks/Plan、architecture、boundary、design、stories、test-plan、既有 Review、代码、migration、Manifest、checkpoint 或 task 状态；未 stage、commit、push、切分支、移动或清理文件。
