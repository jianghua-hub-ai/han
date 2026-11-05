# identity-auth-S3.2 / F005 tasks 与 B4 Plan 独立评审 r01

## 1. 结论

- **Reviewer**：add-reviewer，本批次非作者独立 Reviewer；未委派子 Agent。
- **对象类型**：design（任务设计/规划对象、批次计划对象）。
- **固定工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- **评审结论**：**不建议 PASS，退回原 Task-Planner 修订后复审**。
- **Findings**：MEDIUM 1 项，LOW 1 项。

`identity-auth-S3.2` 的主体条目与 B4 细节能够承接 F005 r17、test-plan r02 和 I-F005-01～12：唯一 task、三态、文件所有权、串行顺序、PostgreSQL 18 阻塞、production observability TBD、r17 独立设计 Review Gate、代码 Review/复审 Gate及未授权边界均已保留，未发现遗漏 task、重复 task、越权实施授权或将 Plan 变成第二份 checkbox 状态源。

但两份对象的顶层“依据/当前活动/精确输入”仍保留旧的 r09/r01、S04-only 与“Manifest 未提供检查命令”表述，和同文件内已登记的 S03/F005 r17/r02 对账及现役 Manifest 事实冲突。直接消费者若只按顶层摘要恢复工作，会取得错误基线或错误判断项目 Gate 来源，因此当前固定版本不满足“准确承接”的要求。

## 2. 固定对象与开始前核对

| 对象 | 固定 SHA256 | 开始核对 | 写入前核对 |
|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `55D3FD7C953234FEC44F78CEDB5C07773EB0694517186C8BEE84B3BF3C68D44F` | 匹配 | 匹配 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `40B00FB747093B767CAD8BEA5E7B761C4AA28D54597E5F77567B7562E71172E5` | 匹配 | 匹配 |

报告路径 `.spec/wip/identity-auth/review-tasks-plan-s3.2-f005-r01.md` 在写入前不存在。评审期间未修改 tasks、Plan、design、test-plan、checkpoint、代码、配置、任务状态或其他报告。

## 3. 适用依据与实际读取

### 3.1 方法与规则

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-route-work/SKILL.md`
- `.codex/skills/add-delivery-workflow/SKILL.md`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/task-execution-workflow.md`
- `.codex/guidance/security-red-lines.md`

本次按 planning/design 对象检查正确性、来源一致性、契约追踪、任务唯一性、状态 SSOT、授权边界、验证与独立评审 Gate；未加载未启用 profile，也未读取未来实施方法。

### 3.2 被审对象与上游

- 两个固定对象全文及必要相邻任务/批次条目。
- `.add/project/knowledge/design-workspace/identity-auth/design.md` 中 r17/F005 合同。
- `.add/records/checkpoints/ir-01-content-community/design/decision-0bc3a0de5aa6e86fae6d7fa874c26c0fb29437762310c5e925b62e1493fabca3.json`。
  - checkpoint object digest：`sha256:41bb2c83f63fb26c8ea8da7e4b01a8ca3cc9a53f918d25b7a4c9893610a0bf87`。
  - checkpoint 只接受 r17 作为 test-plan/R 对账输入，明确不授权 F005 编码。
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
  - SHA256：`0215043EECF2504756D84A5C2959BCCC2CE0183A3E96CA2A8B2605004EAA7560`，匹配。
- `.spec/wip/identity-auth/review-test-plan-r02.md`
  - SHA256：`5197DE52C393B61FA57D1BB13054CE2EC3D26206009AB7778A1A918CEA9BDD59`，匹配。
  - 结论为 test-plan PASS / 0 findings，但不代表设计 Review、实施、环境或 S3.2 关闭。
- `add.manifest.yaml` 中现役 `check_commands`。
- 必要真实代码消费者：identity 登录 Handler/Service、OriginGuard、审计只读 Store、production composition 入口及关联测试，用于有界核对 R 标签和文件边界；未进行代码质量评审。

## 4. Round 1：对象直接检查

### 4.1 tasks / `identity-auth-S3.2`

实际检查：

1. 核对任务统计为 49 total / 12 completed / 1 active / 36 todo；唯一 active 为 `identity-auth-S3.2`，未发现重复 task ID。
2. 核对 S03 R 对账为“已建成 1 / 待做 0 / 需改 3 / 废弃 0”，总体“需改”；复用原 `identity-auth-S3.2`，未新增 F005 task。
3. 核对十个允许修改/新增文件、两个只读文件和一个禁止文件与给定执行合同一致。
4. 核对串行顺序为 identity/fake → platform Origin observer → PostgreSQL 18 → production composition → 非作者代码 Review/F005 复审。
5. 核对 I-F005-01～12 均有阶段承接：
   - identity/fake：01、04、07、08、10、12；
   - platform：09；
   - PostgreSQL 18：02、03、05、06、08、12；
   - production composition：11。
6. 核对 PostgreSQL 18 环境未确认时使用 `blocked-for-execution`，不得 Skip 或冒充通过。
7. 核对 production logger/metrics/wiring 未交付，Q13 backend、阈值、dashboard、值守入口保持 TBD，不得编造 SDK、依赖或配置。
8. 核对 r17 独立设计 Review 报告缺失、F005 编码未授权、非作者代码 Review 与 F005 复审均是关闭条件。

直接缺陷见 F001、F002。

### 4.2 B4 Plan

实际检查：

1. 核对 B4 仍引用原三个 task，S4.3 与 article S1.2 已关闭、S3.2 active、B4 未关闭。
2. 核对 B4 没有新增 F005 task，也没有 checkbox 状态副本；任务三态仍声明以 tasks SSOT 为准。
3. 核对 B4 Context/Gate 对 F005 的范围、顺序、文件所有权、PG18、composition/Q13、设计 Review、代码 Review、复审和未授权事项与 tasks 主体一致。
4. 核对 B5、B9C、B9D、S4.5、S4.6、Git、发布与真实 E2E 未被本次规划授权。

直接缺陷见 F001、F002。

## 5. Round 2：上游合同与真实消费者复核

第二轮未沿用第一轮结论，重新从 r17、checkpoint、test-plan r02、test-plan 独立评审、Manifest 与真实代码接缝反向核对：

1. **来源与授权**：checkpoint 的 scope/receiver statement 与 tasks/Plan 细节均保留“可进入 test-plan/R 对账、不可实施 F005”。
2. **R 对账证据**：现有登录主链、缺失的事件 owner/事务不确定性、写后 finalizer/观测、Origin observer 与 production composition，分别能对应 1 个“已建成”和 3 个“需改”切片。
3. **测试追踪**：I-F005-01～12 未被阶段缩写遗漏；局部、integration、race、contract、项目级检查与 PG18 条件均被登记。
4. **消费者边界**：Handler/Service、OriginGuard、audit Store、server composition 的实际接缝与十写、二只读、一禁止文件合同相容；未发现必须新增 task 或扩大文件范围的证据。
5. **Gate 独立性**：test-plan PASS 未被冒充为 r17 设计 Review PASS；本次 tasks/Plan Review 也未解除设计 Review、编码、环境、代码 Review 或 B4 整体 Gate。
6. **顶层恢复入口**：反向检查两份对象的顶部依据和 Plan 精确输入时，确认其与后部 S3.2/B4 细节及 Manifest 发生冲突，形成 F001、F002。

## 6. Findings

### MEDIUM — F001：顶层依据与当前活动仍指向 r09/r01 和 S04-only，未准确暴露 S3.2/F005 的 r17/r02 R 对账

- **位置**：
  - `.add/project/knowledge/tasks/tasks.md:10`
  - `.add/project/knowledge/tasks/tasks.md:13`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:8`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:9`
- **违反依据**：
  - 设计/规划对象必须使当前依据、范围和下游消费合同一致，不能让顶层权威摘要与对象主体给出互斥的当前事实。
  - `add-code-review` 的直接对象正确性及真实消费者检查要求；`code-review-checklist` 的追踪性、来源一致性与交付完整性要求；`design-doc-writing-style` 的当前事实清晰、避免歧义要求。
- **证据**：
  - tasks 顶层将 identity 依据无范围限定地写为 design r09 / test-plan r01，并称“本轮仅 identity Story S04 进入 R 对账”。
  - Plan 顶层设计依据仍只列 identity design r09，并称 Step A 仅对 S04 执行 R gap-map。
  - 同一 tasks 的 `:301`、`:331`、`:344`、`:346-359` 已将 S3.2/F005 当前依据登记为 r17、test-plan r02，并明确 S03 R Step 2～6 已完成。
  - 同一 Plan 的 `:14`、`:22`、`:41`、`:583-584`、`:603` 也确认 S03/F005 已完成 R 对账同步。
- **触发场景**：Lead、后续 Task-Planner、实施者或自动恢复流程只读取文档顶部“依据与范围/背景/当前活动”，据此选择当前设计和判断 Step A 是否已完成。
- **影响**：
  - 可能按 r09/r01 恢复 S3.2，漏用已接受的 F005 r17/r02 合同；
  - 可能误判 S03 尚未 R 对账，重复执行 Step A，或将后部已登记内容视为无来源增量；
  - 破坏固定规划对象作为交接入口的单义性。主体细节正确，因此未上调为 HIGH。
- **修复建议**：
  1. 由原 Task-Planner 同步两份对象的顶层依据与当前活动。
  2. 明确 r09/r01 继续适用于未受影响 identity Story；`identity-auth-S3.2` / F005 使用 r17 + test-plan r02。
  3. 明确 S03 R 对账已经完成、总体“需改”、复用原 task 且不新增 F005 task。
  4. 不改变现有 task 三态、数量、B4 状态或任何授权边界。

### LOW — F002：两份对象错误声明 Manifest 未提供项目检查命令

- **位置**：
  - `.add/project/knowledge/tasks/tasks.md:13`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:26`
- **违反依据**：
  - 规划对象中的环境与验证事实应以现役项目配置为准；不得用已过期描述覆盖可直接核对的配置事实。
  - `code-review-checklist` 的验证、配置一致性与可复核证据要求。
- **证据**：
  - `add.manifest.yaml:18-25` 与 `:42-49` 均已登记：
    `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、
    `go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。
  - S3.2 自身在 `tasks.md:324-330` 已逐条登记同一组项目检查，因此实际验收命令未遗漏。
- **触发场景**：消费者按 tasks 顶层或 Plan“精确输入”判断项目检查来源、后续模板是否应继承 Manifest Gate。
- **影响**：可能把已登记的项目检查错误归类为临时基线，后续规划继续复制“未配置”事实或遗漏配置漂移检查。由于 S3.2 主体已实际列出完整命令，本固定范围内尚未造成测试命令缺失，故为 LOW。
- **修复建议**：将两处改为 Manifest 已登记上述项目检查；S3.2 的 PG18 integration 条件仍作为额外、受环境阻塞的专用命令，不与 Manifest 项目检查混为一谈。

## 7. 待决定

无需要 Reviewer 代替 Lead/用户决定的架构、契约或风险接受事项。

F001/F002 都是现有事实同步问题，不要求重新设计 F005，也不授权修改已确认范围。若修订过程中发现必须改变 r17、test-plan r02、task 数量、文件合同、串行顺序或 Gate，应停止并另行取得对应授权。

## 8. 非阻断建议

1. 顶层依据可采用“全局稳定基线 + 按 task 的受控覆盖”写法，直接链接 S3.2 的 r17/r02 scoped override，减少后续局部修订造成的摘要漂移。
2. Manifest 检查命令宜直接引用现役配置并仅在 task 中补充专用命令，避免重复维护“是否已配置”的叙述性事实。

## 9. 已确认通过的范围

除上述 findings 外，本固定范围内确认：

- 49 / 12 / 1 / 36 统计相符，唯一 active 为 S3.2；
- 无重复 task ID，无新增 F005 task；
- R 对账四标签完整，S03 总体“需改”；
- I-F005-01～12 可追踪且无语义遗漏；
- 文件所有权、只读和禁止边界相符；
- 串行依赖和单一写者约束相符；
- PG18 阻塞表达明确；
- production observability/Q13 TBD 未被擅自拍板；
- r17 独立设计 Review Gate 保留；
- 非作者代码 Review、F005 复审与 Lead 现场核对 Gate 保留；
- Plan 未复制 checkbox 状态，未更新 tasks 三态；
- 未授权的编码、测试执行、后续批次、S4.5/S4.6、Git、发布与 E2E 均未被授予。

## 10. 未验证风险与限制

1. 本次是 planning/design Review，未执行 F005 编码、Go 测试、race、vet、build、contract、PostgreSQL 18 或真实 E2E。
2. PostgreSQL 18 DSN/major 未确认；相关命令仍是 `blocked-for-execution`，本报告不评价未来执行结果。
3. production logger/metrics/wiring 与 Q13 仍未确定；本报告只确认规划未越权选型，不证明 composition 可实施或可通过。
4. 仓内没有固定 r17 对象的独立设计 Review 报告；本报告不能替代该 Gate，也未对 r17 本身作 PASS 结论。
5. 真实代码只用于核对任务/文件/测试消费者，没有进行完整代码 Review。
6. 工作树存在大量既有未提交/未跟踪成果；本评审未清理、覆盖、格式化或回滚它们。
7. 未对本次固定范围之外的 48 个 task 或全部批次进行完整重规划。

## 11. 复审条件

原 Task-Planner 修订后，复审至少需要：

1. 提供新的 tasks/Plan 固定修订与 SHA256。
2. 对照 F001 核对两份对象顶层依据/当前活动已明确 S3.2 的 r17/r02 scoped basis 与 S03 R 对账完成事实。
3. 对照 F002 核对 Manifest 检查命令事实已同步，且未删减 S3.2 的专用 PG18、integration、race、contract 或项目级命令。
4. 核对任务仍为 49 total / 12 completed / 1 active / 36 todo，唯一 active 仍为 S3.2，且无新增 F005 task。
5. 核对 task 三态、文件合同、串行顺序、PG18/production/r17 Review/编码授权/代码 Review Gate 均无非预期变化。
6. 修订若超出上述事实同步，必须说明变更依据并对受影响范围重新做两轮评审。

在 F001 修复并完成对应复审前，不建议通过 B4 规划 Gate；不得用接受 F002 风险的方式绕过 F001。
