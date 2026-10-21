---
name: add-batch-plan
description: 批次组织——消费 task-decompose 的原任务，把本次稳定范围内的未安排任务与待验证存量组成全套依赖批次，核已有认领避免遗漏和重复。task-planner Step B；只消费任务，不创建任务或重复设计。；仅适用于 Codex；其他 Host 不使用此投影。
requires:
  - task-decompose
  - rule:task-execution-workflow
  - rule:multi-agent-coordination
  - rule:agent-framework
  - task-implementation
  - code-review
  - git-workflow
produces:
  - <plans-dir>/<batch>.md（原任务引用、依赖安排、有效设计与验收证据引用）
  - 原 tasks 的批次/责任关联与必要状态推进；R 的 [-] 待验证项对账
---

> 仅适用于 Codex；其他 Host 不使用此投影。


# Batch Plan — 批次组织流程

## 核心理念

按 OO ISP（接口隔离），task-planner Step B 的核心动作 = **批次组织**——把 task-decompose 刚拆出的**整个特性的 task 一次分成全套批次**（批1..批N），明确当前 Workflow 下的设计依据、任务与整体完成条件。**不是只挑几个打一个批**——要一次分完，形成完整批次地图（多人协作的认领底座）。

**"先 task 后 plan"严格语义**：

```text
Step A: task-decompose SKILL（粒度跃迁，task-planner 主导）
  ↓ 产 tasks*.md 条目
Step B: batch-plan SKILL（本 SKILL，task-planner 主导）
  ↓ 消费 tasks 条目 + 组 Plan
```

**本 SKILL 只消费 tasks 条目，不创建**——如发现 task-decompose 漏了某 task → 回 task-decompose 补登记。

## 分批标准 + 多人防冲突（🔒 核心）

批次既是**执行+收尾的单位**，也是**多人协作的认领单位**。一次把特性的 task（待做 `[ ]` + 需改 + 已建成 `[-]` 待验证）分成全套批次，按三条切：

1. **依赖排序**：有 `requires` 的排在前置批之后；同批内无相互依赖。
2. **文件不重叠（并发批）**：可能被不同人**并发**做的批，**改动文件不能重叠**（用 task 的"文件清单"字段交叉查）——这是防冲突的根。重叠的拆到不同批，或安排串行（一前一后，不并发）。
3. **规模可复核**：根据依赖、评审/验证成本和实际团队能力控制单批范围；较大批继续分，不能为凑数量破坏一个行为闭环。

**多人防冲突**（🔒）：先核实际所有权、契约和依赖，再安排并发/串行；不靠盯别人的 `[-]`，不把整批分工当实时锁。未同步的其他机器改动不可见，必须保留已知责任来源和未知范围；冲突时协调，不能承诺规划必然阻止所有重复开工。

**`[-]` 只表示进行中**，可能是已认领，也可能是 R 模式已建成待验证；认领要看原 task 的批次关联、责任和证据，不能从 checkbox 推定。实际分工按原任务/当前活动记录来源，Git 同步按本次授权，不要求领批就提交到某条主线。三态遵守 [task-execution-workflow](../../guidance/task-execution-workflow.md) §10，不另建锁或任务状态库。

核对 task-decompose 提供的任务类型、关闭条件和后续验证关联。开发自测随实现，独立专项测试/E2E 按依赖安排，不为每条开发 task 重跑同一套 E2E；安排缺项回 task-decompose 补。

## 何时启用

task-decompose 完成后（tasks*.md 已有相关条目），需要把本次确认范围的 task 组成完整批次安排时：

- 纲领任务（按项目特定 `<task-id>` 前缀，见 项目入口）跨服务链路
- 跨多个 submodule 的 batch
- 一个模块的多个相依执行单元需要分批协作
- 任何用户明确要求"做个 Plan"的任务

不适用：

- 无需成批编排的局部 bug 修复（是否由 Lead 直接执行按 multi-agent-coordination §2 判断）
- 探索 / 调研型任务（无明确产出清单）
- 紧急线上修复的处置本身由 issue-handling 负责；不把事后补 Plan 当自动义务
- **tasks*.md 还没有相关条目**——先走 task-decompose SKILL 产条目

## 核心机制

Plan 拥有本批依赖安排和证据引用，不拥有第二份任务状态或人的批准事实。G1/G2 等名称仅沿用项目当前 Workflow 的检查点：已批准设计明确覆盖本任务时直接引用其位置、修订与确认来源，不为每个 task 重写 TDD；只有真实缺口才交原设计 owner 补齐受影响范围。

研发交接和具体出口见 [delivery plan](../add-delivery-workflow/SKILL.md#wf-step-plan) 及 implement/review/integrate/e2e；其他已选流程使用自己的合同。规划完成不等于实施授权、独立评审、人接受或发布许可。

## 流程

### Step 1：把特性的 task 全部分成批次（从 tasks*.md 已有条目消费）

⚠️ **前置**：原 tasks*.md 已有当前范围的有效条目与设计/验收依据（含 `[ ]` 和 R 的 `[-]`）。已有输入可复用，不为了调用次序重复 task-decompose；本 SKILL 不创建新条目。

task-planner 先按当前明确范围核对已有事实；能查明的不反复问，缺责任/选择才协调：

1. **定位任务**：用 Host 可用的文本/路径检索能力按任务编号或关键词，在 `<tasks-dir>/tasks*.md`（按形态分文件——见 项目入口 任务管理段）中定位
2. **核未完成子任务**：同时检查 `[ ]` 与 `[-]`，先查原任务的批次关联、责任及实现/验证证据，按下表确定本次安排，不把 `[-]` 一律当已领
3. **识别依赖关系**：读任务头部"前置 / 后续 / 排期"说明（task-decompose Step 3 已补全 `requires` / `[P]`）
4. **按三条标准把 task 全部分批**（见 §分批标准 + 多人防冲突）：依赖排序 + 文件不重叠（并发批）+ 可复核规模——一次分成全套批次，不是只挑一个
5. **查本范围仓库状态**：核固定工作区/分支、对应修订和 task 文件 diff/status；保留他人修改，不切其他工作区或以全树干净作规划前提

| 原任务关联事实 | 本次处理 |
| --- | --- |
| 已核实无批次，验证责任明确未分配或来源清楚，且证据明确为已建成待验 | 纳入本次验证批，明确原 task 的验证范围、责任及依赖，不另造实现 task |
| 有有效批次，责任/范围/证据相符 | 引用既有安排，不重复分配；必要时将依赖接到该批真实交付条件 |
| 批次/责任缺来源、失效或关联冲突 | 标待协调及具体缺项，保留原任务和证据；未知不当已领、未领或已完成，不抢派覆盖 |
| 其他未安排的待做/需改条目 | 按现有设计、所有权和依赖纳入本范围全套批次 |

**如发现 tasks*.md 缺少应有条目**（如 Story 列表里有但 tasks*.md 没有） → ❌ 停手，回 task-decompose SKILL 补登记，禁本 SKILL 内私自创建条目。

需停下来问用户的例外：

- tasks 文件与设计文档冲突
- 任务编号在多个 tasks 文件能匹配（形态归属模糊）
- 依赖链里有已完成项但状态未闭环
- 现有范围无法形成可独立评审/验证的批次，或发现未知在途写者

### Step 2：建 Plan 文件并填外层表 + 工作区约定

按 [references/template.md](references/template.md) 的适用字段建立或续接 Plan，路径由项目现有 plans 约定解析，不重复创建已有有效批次。

**外层表**保留任务 ID/引用、一句话目标、角色/实际责任来源、依赖、预计文件数和形态收窄；状态只引用原 task 的可见修订，不在 Plan 手工维护第二状态。给完整 DAG/批间启动条件及未安排原因，不能把“待协调”静默排除出总览。

**工作区约定**：逐个列本次获准范围内的仓库、固定 worktree/分支、涉及 task 与实际修订。规划不自动建分支、切 checkout 或改其他仓；确需新工作区/分支由当前 Git 约定和相应授权决定，不预填可执行切换命令。

**设计依据按缺口展开**：

- 每个 task 引用已批准设计的精确切片、修订、确认来源、接口/数据/异常/测试要求；已足够则直接消费，不复制第二份 TDD。
- 只有真实缺口或已失效依据才交实际设计 owner 补相关正文，记录受影响任务与继续条件；不因批次大小统一重做首批若干 task 的设计。
- 待决项保留原决定引用，暂停依赖范围；未受影响的已授权安排可以继续。
- Plan 内层只保留接手必需输入/证据引用；正文仍在原设计，既有有效上下文可复用，丢失才补读。

### Step 3：汇报完整批次与实际确认

一条消息说明：

- 全套批次、原 task 引用、依赖图与可并行/必须串行范围；
- 本次起点及后续启动条件，R 待验纳入、既有有效批次引用与待协调项；
- 获准仓库的当前分支/修订、精确输入清单和设计缺口；
- 当前活动的产出负责人、接收负责人及确认来源，已确认/待确认/拒绝的真实范围。

确认要求按当前 Workflow。未答不报已确认，岗位或 Agent 名不推定人的批准权；已有效接受的未变范围可复用，规划许可不等于后续编码/发布许可。

### Step 4：单任务输入、实施与独立评审交接

依当前 Workflow 逐个核对相依任务，不在 Plan 重写方法或自动派角色：

| 检查/交接 | 所需事实与唯一方法 |
| --- | --- |
| 设计依据 | 核已批准设计与当前 task 相符；真缺口回设计 owner，不默认委派 feature-designer 重写 TDD |
| 实施输入 | 任务/变更引用、精确文件、消费者、实际设计/预期、Rule/Profile、验收与 Git 约定；交 [task-implementation](../add-task-implementation/SKILL.md) |
| 独立评审 | 非作者取得实际对象/修订、允许 diff/依赖范围、原设计与真实测试；交 [code-review](../add-code-review/SKILL.md) |
| 修复复审 | finding 回真实作者，保留原 finding 与新修订/验证，再独立复审；设计问题不一律送编码 |

单 task 核本任务验收范围的成果/测试、Lead 现场 diff、G4-task 和适用接收，完成证据在原 task 引用；共享文件/契约依赖按实际先后处理。没有独立 Reviewer 则保留待审，不以作者换角色名补齐。自身条件满足即按 Rule §10 回填原 task，不延至批次或需求整体验收才统一回填。

真实调用、Context Pack、透明度和失败处理由 [multi-agent-coordination](../../guidance/multi-agent-coordination.md) 规定。角色表是职责，不证明 Host 已调用，也不自动授予后续操作。

### Step 5：批次整体核对与交回

局部 task 通过后另核批次范围的跨任务依赖、接口一致性、适用集成/性能/覆盖结果及 G4-batch 整体评审；不能把多个 G4-task PASS 相加冒充整体接纳。测试缺口回 test-design/task-decompose 明确责任，失败回实际作者/问题 owner。

按当前 Workflow 取得本次产出/接收责任人的真实决定，说明已完成、未测、延期与后续 owner。部署后的真实 E2E 独立安排；缺环境/权限保留未验范围，不用构建或部署成功代替。

原任务按 [task-execution-workflow](../../guidance/task-execution-workflow.md) §10 由 Lead/指定关闭责任核证后更新，Plan 仅回填成果、评审、决定与原 task 引用。项目确实要求的验证日期才填写，不能编未来签字或验证。

集成/发布仅在授权范围沿 [git-workflow](../add-git-workflow/SKILL.md) 与当前 Workflow 承接，不默认 push/PR/merge；约定仅交 diff 或本地提交时如实返回，不为收尾扩权。回真实当前活动，未完事项仍有接手引用。

## Lead 验收三件套（按当前对象适用）

Lead 不以 Agent 摘要替代现场证据：

1. **交付核对**：按 Git 约定核实际 SHA/文件，或未提交 diff；禁止提交时不要求补 SHA。
2. **验证核对**：核真实命令/操作、版本、完整结果与未测范围，不只看最后一行。
3. **范围核对**：在固定工作树读取本任务 status/diff、原任务及验收/独立评审引用，保留他人改动；diff 基线用实际约定，不默认 HEAD~1。

不符就说明受影响范围，先补证/协调；不能为让工作树干净回滚他人或擅自 push。

## 反事故清单

| 错误 | 修正 |
|------|------|
| Agent 只报完成而无实际变更/验证证据 | 要求补真实 diff 或约定 SHA、结果与限制；不能虚构提交 |
| 所需方法正文缺失/失效 | 补选定方法的完整有效正文；仍有效的全文可复用，不一律重读库 |
| 本任务实施/验证失败 | 停依赖其结果的推进，核原尝试/写者及影响后回真实作者；不默认 revert 或覆盖现场，回退另核精确范围与授权 |
| TDD 与 tasks.md 冲突 | 停，问用户，不自行改 TDD 或 tasks |
| 集成目标或 Git 授权不明 | 沿当前 Workflow/Git 方法请明确；共享历史/他人基线不擅改，不默认跨仓推送 |

## 与其他规则/技能的关系

- **前置依据**：原 task 与有效设计/测试建议，已有批准设计可复用；缺 task 才回 task-decompose
- **当前流程**：[delivery](../add-delivery-workflow/SKILL.md#wf-step-plan) 或已选 Workflow 拥有具体交接/出口；[task-execution-workflow](../../guidance/task-execution-workflow.md) 拥有 G0、授权、成果归属与三态
- **Sub-agent 协作**：[`multi-agent-coordination.md`](../../guidance/multi-agent-coordination.md)
- **Git 推送**：[`git-workflow SKILL`](../add-git-workflow/SKILL.md)
- **模板文件**：[`references/template.md`](references/template.md)（新批次复制改填）
