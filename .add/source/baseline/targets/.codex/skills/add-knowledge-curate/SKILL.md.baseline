---
name: add-knowledge-curate
description: 在材料产生或 handoff/close/cancel/supersede/query 触点整理当前范围的知识和过程材料。扫描、分类、处置、同步索引、提出改进并交回；不替专业决策，不自动删除材料或采用永久规则。；仅适用于 Codex；其他 Host 不使用此投影。
produces:
  - <archive>/legacy-plans/<batch>.md（归档 plan）
  - <archive>/design-workspace-archive/<feature>/（归档过程文档）
  - <design-index>/INDEX.md / <decisions-dir>/README.md（索引同步）
  - <archive>/feedback-drafts/<name>.md（feedback memory 草稿）
---

> 仅适用于 Codex；其他 Host 不使用此投影。


# Knowledge Curate — 知识整理

## 1. 身份与适用边界

本方法不拥有独立 Workflow；它是贯穿现有 Workflow 与项目生命周期的跨流程方法，沿用完整整理链，任一研发环节都可局部启用。

- **交接与收尾**：单项任务、Batch、特性或整体工作到达实际退出点时，核对本范围材料，不以批次名或固定 G1–G5/PR 顺序决定归档
- **材料产生与活动返回**：说明新过程材料的用途、关联工作、责任人和退出安排；相关材料有待办时定向处理，不等研发全结束
- **反馈沉淀**：对话中识别出用户纠正/验证的教训，形成候选；是否写入已选记忆/知识位置另看明确采用范围
- **INDEX 梳理**：发现 `<design-index>/INDEX.md` 或 `<decisions-dir>/README.md` 跟现实脱节
- **跨阶段对账**：大 batch 收尾 / 季度整理时跨多 plan 状态对账

**不适用**：

- 不替专业负责人决定需求、设计、测试或评审结论；这些环节中仍可整理获授权的引用和过程材料
- 单次小修补（无需归档）
- 跨项目复用（这是单项目知识整理）

### 按动作选择规则

- [`agent-framework.md`](../../guidance/agent-framework.md)（有方法候选需要归属判断时）
- [`decisions-guide.md`](../../guidance/decisions-guide.md)（决策索引同步 + 引用计数规则）
- [`task-execution-workflow.md`](../../guidance/task-execution-workflow.md)（任务状态与成果归属的横切约束，不从这里再取活动操作链）
- [`design-doc-writing-style.md`](../../guidance/design-doc-writing-style.md)（写归档总结 / feedback 时遵循直陈）

### 输入契约

调用方（steward 主 Agent / 主会话）必须提供：

| 字段 | 说明 |
|---|---|
| **当前范围** | 当前 Workflow/调用方、项目制品引用与版本、材料消费者、边界及交回位置；不另建业务状态 |
| **触发原因** | 一句话上下文（批次完工 / 反馈沉淀 / 季度对账 / etc） |
| **整理范围** | 涉及哪些 plan / feature / batch / 跨形态 tasks |
| **期望产出** | 归档清单 / 索引修订 / feedback 草稿 / 等 |
| **处置依据** | 当前材料及已登记记录的引用、owner、保留/使用约束、真实写入与删除授权；未知则指出缺口 |

### 输出契约

完工时返回主会话：

- **摘要 5-10 行**：变更 + 发现 + 待用户决策项
- **落盘**：归档文件 / 索引修订 / feedback 草稿
- **对账清单**：文件清单、关键变更、处置/恢复来源、未完动作及真正待决定事项；不另造活状态账

---

### 现役与历史的核对依据

归档 / 回灌场景常需判断"archive 旧设计内容是否已被现役覆盖 / 是否跟现役冲突 / 缺口还在不在"——**这个判断只认当前 design.md，不信任何中间对账报告或 audit 的二手结论**。

**为什么**：design.md 是持续演进的 SSOT。任何对账报告（深扫报告 / explorer audit）里"现役 design.md 覆盖状态"的话，都是**写报告那一刻的快照**——report 落盘后 design.md 继续变，"现役怎样"的判断立刻可能过时。

| 源 | 性质 | 用法 |
|---|---|---|
| archive 原文 | 不变 | 可信源——"旧设计说了什么" |
| **当前 design.md** | **唯一真相** | 判断"缺口/冲突在不在"**必须逐条 Read 核实** |
| 中间对账报告 / audit | 会过时 | 仅作"线索导航"，**不作执行依据**——每条回 design.md 复核 |

**已验证的二手报告失真场景**：

- **现状已演进**：阶段报告声称某特性采用单一路径，实际当前 design 的决策追溯已改为混合方案——报告的"现役判断"已过时
- **动作被转述丢失**：audit 说"目标章节不存在 = 落点错"，实际原文写的是"**新增**该章节"——二手转述把新增动作误成既有引用

**铁律**：合入前判断缺口 / 冲突，先 Read 当前 design.md 对应章节；报告只导航不裁决。**不基于二手报告产新 audit 草稿**（越产越偏 + 文档爆炸）——archive 原文 ↔ 当前 design.md 直接对照即可。

<a id="wf-definition"></a>

## 2. 步骤合同（完整整理链）

六步各有稳定身份，允许从已有有效成果接续；不是每步都扫描全项目。程序只辅助保存与明确检查，不能判断语义过期、可删除性或代人批准。

<a id="wf-step-scan"></a>

### Step 1：扫描

按真实触点 + 整理范围扫描。已登记材料在 handoff、close、cancel、supersede 或显式 query 时，只计算提醒与处置候选；触点事件缺失或不可观察时结果是 `unknown`，不补造事件、不自动执行处置。

**Batch 收尾场景**：

- Read `<plans-dir>/<batch>.md` —— 核对本次约定的任务、独立评审/测试及交付条件；未授权 push/PR 不补造，也不据此归档未完成工作
- Read 当前 batch 实际引用的 tasks 条目；共享任务核原权威与真实使用处，不全库预读
- Read 本 batch 涉及的 `<design-workspace>/<feature>/stories.md` + `test-plan.md` 临时过程产物（按 K8s 控制器思想，本期可归档；前提是 core 已内化到 design.md / tasks*.md）

**反馈沉淀场景**：

- 回溯主会话对话记录中"用户纠正 / 验证 / 教训"片段
- 只在项目已登记且获授权的现有记忆/反馈位置查重复；不因找不到 MEMORY.md 自动创建或扫描用户目录

**INDEX 梳理场景**：

- Read `<design-index>/INDEX.md` 现状
- Glob `<design-workspace>/*/design.md` 看实际特性
- Glob `<decisions-dir>/*.md` 看实际 ADR
- 对比 INDEX 与实际，找出 drift 项

**全过程材料**：除 plan，还包括需求/边界讨论、设计过程说明、评审发现、交接记录、测试 Case/Run/日志、Issue 复现与处置证据、反馈候选。先列当前涉及的材料与消费者，再定位现役正文；不能按目录名“一键清空”。

模型通过 Host 按已登记来源扫描，在查询/交接/收尾只形成三类轻量提示：`missing-local-reference`（明确本地引用缺失）、`input-changed`（当前输入与已有摘要不同）、`material-pending`（已登记材料待处理）。访问失败或没有可比旧摘要是覆盖未知；不是文件缺失或业务过期。不定时巡检、不自己推进 Gate。

Agent 判断差异是否影响本次工作；只暂停确实依赖未知/冲突的动作。未登记的旧材料不是违规，未启用 Wiki/Memory 不阻断整理；缺访问权限仍如实交回，不冒称已读。

<a id="wf-step-classify"></a>

### Step 2：分类

按 [`task-execution-workflow §8 产物归属唯一表`](../../guidance/task-execution-workflow.md) 把扫描结果分类：

| 类别 | 处置 |
|---|---|
| 已完工 plan | → Step 3 归档到 `<archive>/legacy-plans/` |
| 过程产物（stories.md / test-plan.md 等） | → Step 3 核对用途、当前消费者和保留要求后选择保留/更新/延期/归档/回收；仅名为“临时”不构成删除条件 |
| 已完工 tasks 条目 | → Step 4 状态同步（标 `[x]` + 行末日期，按 §10 SSOT；跨形态共享同步） |
| 设计 design / ADR | 不归档（永久 SSOT，仅索引同步） |
| feedback 候选 | → Step 5 起草 feedback memory |
| 流程 / 规范卡点 | → Step 5 回灌 rules / SKILL 改进建议 |

**材料唯一记录**：新过程材料都要说明 `purpose`（用途）、`owner`（维护责任）、`authorityRef`（权威位置）、`retentionCondition`（保留条件）和 `exitCondition`（退出条件），并引用关联工作；只有需要交接/收尾追踪的才登记。优先复用既有材料索引的一行；项目已用 frontmatter 时在材料的 `add_maintenance` 小节记录，二选一，禁止长期双写或另建数据库。材料记录只描述材料生命周期并引用项目制品，不复制业务状态或成果正文。

登记状态为 `pending`（尚有动作）、`deferred`（说明延期理由，可给明确 UTC 日期）、`done`（真实核验过的处置，包括决定保留）。在登记的 handoff/close/cancel/supersede 触点计算提醒，query 可看现状；这些触点只产生提醒与处置候选。无日期延期不猜到期日，done 不反复提醒、不因记录自身 hash 变化自动重开，确需重处理由 owner 说明原因后显式改回 pending。

<a id="wf-step-disposition"></a>

### Step 3：归档

<a id="cleanup-consent"></a>

#### 清理清单确认

所有移动、归档、覆盖或删除都使用本方法，包括过程材料、构建/验证临时件、Git 清理、安装升级 stage、备份和诊断日志；审计不在任意删除范围，仍遵守其更强的追加/归档与保留约束。Host 必须先展示精确对象、当前内容或摘要、动作与目标、引用和其他影响、保留与恢复方式，并取得用户对当前清单的当前确认；提醒或处置候选本身不得自动执行。

1. **展示清单**：模型经 Host 向用户逐项列出精确路径、动作、理由、保留去向（不保留须明确说明）、引用影响和可恢复性，包括恢复来源与条件、拟执行的失败回退或取消清理范围。只核显式对象与引用，不证明全仓已枚举；范围不清保留原件。
2. **核对确认**：Host 核对真实用户确认的清单及适用范围；未确认或拒绝时保留原件、不执行。只批准部分时收窄并重列计划，必要时重新确认；确认后范围或内容变化时保留原件，重新展示并确认。到期、有备份、普通“继续”、默认保留策略、提醒、Gate 结束、流程完成或任务结项都不构成处置授权；研发成果接受或设计/代码实施许可也不能替代。
3. **按清单执行**：模型负责理解文本差异，形成精确的 ChangePlan 候选；Host 逐项展示对象、before/after、引用影响、备份与恢复方式。只在用户确认当前清单后，模型才通过 Host 按计划写入、归档或恢复，并核验实际结果。Git 清理、安装采用暂存等沿各自已有方法执行，只共用本清单确认与结果核验；内部 candidate/plan/apply/verify/record 是方法阶段，不对外暴露为新的命令组。ChangePlan、DecisionRef 或模型自填 `confirmed` 不能生成用户同意，ADD 不按生命周期自行发起，不新增审批服务、调度器或清理台账。只建议或只生成计划时不改业务文件、不标 done；执行失败保持 pending，保留原因与下一步；完成按本步后文核验实际结果、引用和恢复条件再收口。

**plan 归档**：候选位置为 `<archive>/legacy-plans/<batch>.md`，先确认源与目标、引用、必要历史和恢复安排，在获授权的同一材料处置计划中修改，不能直接按示例路径移动现役内容。

归档前确认：

- plan §变更记录段含实际交付版本/commit，PR 仅实际存在且属于本次交付时登记
- 本次适用的检查和真实确认有来源，不伪造未发生的 Gate/日期
- 若本计划代表整批已完成，核其关联任务真实完成与验收；若计划被替代/取消或仅局部交接完毕，记录接替计划及未完任务去向，不把归档动作当作任务完成

**临时过程产物归档**：

候选位置为 `<archive>/design-workspace-archive/<feature>/`；核对每份材料仍有无消费者或未迁移的唯一证据，更新引用与内容移动必须一起计划。

**change/spec 载体归档**（项目显式启用且达到该变更真实关闭条件时）：

```text
先在项目声明的工作目录单独执行 `<project-change-validate-command>`；通过后，再单独执行 `<project-change-archive-command>`。不要要求 `cd`、`&&` 或 shell 变量。
# delta spec 合回 <specs-dir>/<capability>/spec.md + change 壳移入 <changes-dir>/archive/
# validate 不过 → 停手，回对应活动修，不许带病归档
```

保留：

- `<design-workspace>/<feature>/design.md`（永久 SSOT，含顶部 §决策追溯段）
- `<design-workspace>/<feature>/boundary.md`（永久 SSOT，边界拼图）

判断（归档 stories.md / test-plan.md 前必走）：

- stories 是否仍用于实施/验收？仍有消费者就保留；被替代、取消或局部交接完成时，核对有效接替材料、原任务引用和单独退出授权，允许归档已退出材料，任务保持真实未完状态
- test-plan 的长期目标/关键场景是否已在现役设计或其正式测试依据自包含？仅移除当前唯一测试依据会断链时，先由对应设计/测试负责人确认归位；有效接替计划已承接时不重复抄回所有内容
- 本次归档是否依赖整体实现完成？若依赖，核对现役 design 的实际状态、集成与验收证据；若只是已完成局部材料，则核对其单独退出安排，不强迫整项开发先结束

#### task / batch / feature 适用材料清单

收尾逐项核对本范围材料，不是全部搬走。用于跨会话交接/验收的成果保存在项目指定可追踪位置；自建独立测试临时目录可以承载可重建的运行副产物，但不能成为唯一长期交接依据。

**子目录命名规范**（按 scope-date 模式）：

| Scope 类型 | 目录名模板 | 示例 |
|---|---|---|
| 单 task 完工 | `<archive>/<task-id>-<YYYY-MM-DD>/` | `<archive>/task-001-<YYYY-MM-DD>/` |
| 批次 batch 完工 | `<archive>/<batch-id>-<YYYY-MM-DD>/` | `<archive>/batch-a-<YYYY-MM-DD>/` |
| 单 feature 完工 | `<archive>/design-workspace-archive/<feature>/` | `archive/design-workspace-archive/<feature>/` |

**归档完整性 checklist**（适用而缺失时暂停对应处置，不阻断无关研发）：

| ✓ | 产物类型 | 来源 | 必含？ |
|---|---|---|---|
| □ | 已有索引的归档条目或必要时 README | steward / 主会话 | ✅ 实际位置/工作引用/现役 SSOT 跳转；能复用不另造文件 |
| □ | 实际已有的评审/测试结论及证据引用 | 原 reviewer / test-engineer | 有该活动才核其实际对象、修订、作者与结论；不强制合成 quality-audit 或补做不适用的代码覆盖率 |
| □ | test-plan 测试规划 | test-engineer | ✅ 任务有测试规划则必含 |
| □ | 决策对比报告（如有真冲突点）| feature-designer / steward | ✅ 任务含决策对比则必含 |
| □ | 进度记录 | dev | 核对是否含 Git 未覆盖的决策/交接；可证明重复且获授权才回收 |
| □ | coverage raw / HTML / 运行日志 | test-engineer | 核对唯一复现证据、审计和复核需求；仅能重跑或已有数字摘要不足以自动删 |

特性的现役设计由其唯一 design.md 承载（见 [add-design-tdd](../add-design-tdd/SKILL.md)），不为每个 task 另造设计；现有片段是否保留仍按有效内容、引用和历史价值判断，不以文件名认定垃圾。

若发现 task 目录下有独立设计片段：

1. 核对其有效内容（字段表 / 接口契约）是否已在特性 design.md
2. 缺 → 回 feature-designer 增量补进特性 design.md
3. 补齐后该片段可作"历史快照"归档，README 注明"正式 SSOT 在 <特性 design.md 路径>"

**精简原则**：

- ✅ **保留**：有独立分析 / 决策 / 证据 价值的报告
- ✅ **先核再回收**：没有独立用途且内容已完整归位的重复材料，仍核对引用、保留要求与授权；审计不在任意回收范围
- ✅ **复用报告**：评审与测试结果优先在既有报告交叉引用，保留各自作者和结论来源；不为整理重做专业评审或抹掉独立证据

**临时工作区位置**（重要）：

- 交接材料用项目确认的过程文档区；客户项目可选择 `.spec/wip/<scope>/`，ADD 自身研发材料使用其独立研发区，不注入可分发模板
- 测试副产物可在明确所有权的隔离临时目录运行，必要证据先保存再清理；不得把别的会话、项目或宽目录作为清理目标

委派给执行者时明确本次产物位置、文件边界和保留安排。

**归档状态 header 模板**（已归档但部分内容仍被引用时使用）：

当 archive 内容不全废弃——比如某 archive 文档的部分章节经重新分析后归位到现役 SSOT，部分章节仍是历史快照——在 archive 文档顶部加"归档状态 header"声明归位结果：

```markdown
> **归档状态**（YYYY-MM-DD 更新）：[一句话现状——如 "本文档大部分内容已过时弃用，X 段共 N 项已重新分析归位"]
>
> | 分类 | 项数 | 处置 |
> | --- | --- | --- |
> | ✅ 已被 <SSOT-A> / <SSOT-B> 充分覆盖 | N | 弃 archive 内容 |
> | ❌ 缺口已补到 <SSOT-X> | N | 见 [<SSOT-X 链接>] |
> | ⚠️ 规格差异修正到 <SSOT-Y> | N | 见 [<SSOT-Y 链接>] |
>
> **本文档保留为历史快照，所有现役约束以上述 SSOT 为准。**
```

效果：

- 读者扫一眼 archive 就知道哪些项已被覆盖 / 哪些已修正 / 哪些是历史快照
- SSOT 跳转链接避免引用断裂
- 历史与现役区分清楚

归档文档头部写法：开头加 overlay 注记，用一张小表标"哪些已被现役覆盖（弃）/ 哪些缺口已补到何处 / 哪些是历史快照"，并在末尾给「现役 SSOT 跳转」段（指回现役 design / tasks / decisions / 代码），避免读者把归档当现役 + 引用断裂。

**实际处置与恢复**：沿[清理清单确认](#cleanup-consent)执行。普通材料整理不关闭 Host、不启动三端 smoke；只有确实改变框架投影才适用采用流程的维护条件。

回收自带 frontmatter 的源文件前，在同一计划中先把同一 ID 的 pending 记录迁到明确存续索引并更新所有已登记引用，再删除源文件；保存 originalPath、operationId/source/action，回收后 path=null，resultPaths 只列真实保留结果，可为空。存续索引本次不能删除；落点或 materialRecordRefs 不明则不回收。源/索引暂时并存由未结 journal 表示迁移中，不变成两个长期权威。

中断按原计划/journal 的实际 before/after 续接或回退，第三种内容必须保留现场。核验源处置、存续索引、引用、实际成果和恢复记录齐备后，模型才通过 Host 在存续条目按当前 expected 摘要写 done，再把新 ArtifactRef 同步到已登记引用。不向已删除的 frontmatter 写回。任何未完写入/同步报告 partial、保留原因与下一步；未核验时维持 pending，不因删除或计划生成成功就关闭。新会话按 materialRecordRefs 和原 operationId 找回核验；已核实 done 但引用摘要落后只补同步，不重开处置。之后要撤销且现状不同于旧 after，须明确范围的新计划，旧恢复不覆盖后续记录。

<a id="wf-step-index"></a>

### Step 4：索引同步

**`<decisions-dir>/README.md` 三处索引**（按 [`decisions-guide.md`](../../guidance/decisions-guide.md)）：

- 决策索引表（次数从唯一引用记录派生；只查询不增加记录）
- 按仓库查询表
- 决策文件组织表

**`<design-index>/INDEX.md` 持续演进**——5 种触发场景：

| 触发场景 | 何时 | 同步内容 |
|---|---|---|
| 新特性 design.md 落盘 | 活动 5 完工 | 在 INDEX.md "特性清单"段加一行：`<feature-id> - <一句话> - status: draft - link` |
| 特性 status 流转 | 活动 5 / 6 / 9 Gate 通过时 | INDEX.md 同步 status（draft → implementable → implemented） |
| 微服务边界变化 | 活动 2 / 3 涉及微服务划分时 | INDEX.md "微服务清单"段更新职责 / 边界 / 接口契约 |
| 跨服务接口契约变化 | 活动 5 / 8 发现 | INDEX.md "接口契约"段记录契约版本 + 引用 ADR |
| 演进路线推进 | 活动 9 / 10 阶段性收尾 | INDEX.md "演进路线"段标 milestone 达成 |

**INDEX.md 持续演进 vs 一次性写满**：

- ❌ 不允许"先空着，等所有特性完工再补 INDEX"——INDEX 必须**跟着 design.md 同步生长**
- ✅ 每次特性首次落 design.md（status=draft）时，**同步在 INDEX.md 加最简一行**：feature-id / 一句话 / status / link
- ✅ 后续 status 流转跟随更新；不重复造索引
- ✅ INDEX.md 是**导航**不是**复述**——一行 100 字内，详情走 design.md 跳转

**引用一致性自检**（本次明确整理的范围）：

- `<design-workspace>/*/design.md` 全部都能从 INDEX.md 跳转到（否则 INDEX 缺项）
- INDEX.md 列的特性都能在 `<design-workspace>/` 找到 design.md（否则 INDEX 引用悬空）
- 检查实现按项目入口登记的 `<design-workspace>` 和 feature-id 规则遍历；不得在母本写死特性编号正则或系统临时路径。

**`<tasks-dir>/tasks*.md` 状态同步**：

- 按 §10 SSOT 标 `[x]` + 行末日期（`— 完成@日期`）
- 纲领任务双日期（`— 完成@日期 — 验证@日期`）
- 共享 task 只改业务权威条目；其他汇总更新其引用/派生展示，不再维护第二份 owner、签字和活状态

判断：

- 逐项核对本范围的真实完成证据；仍未验收的保留待办，不以“清零勾选”证明整体完工

<a id="wf-step-feedback"></a>

### Step 5：回灌 + feedback 沉淀

**feedback memory 起草**：

对话中识别出的关键教训（用户纠正 / 验证 / 不明显约束）—— 按 项目知识规范 类型起草到 `<archive>/feedback-drafts/<name>.md`：

```markdown
---
name: <short-kebab-case>
description: <一句话>
metadata:
  type: <user | feedback | project | reference>
---

<内容——按类型对应结构>

# 链接相关：[[other-memory-name]]
```

**rules / SKILL 改进建议**：

- 如某流程卡点反复出现 → 起草 rules / SKILL 修订建议（不直接改 rules，写到对账清单交用户拍）
- 如新角色 / 新动作需求 → 建议新增 Agent / SKILL（按 `agent-framework §7 自检清单` 评估）

先在既有反馈/问题条目留 `proposed` 候选、来源、适用条件、反例与验证方向；客户项目的材料、规则与方法候选统一交 [add-project-experience](../add-project-experience/SKILL.md) 评价，明确采用后才改变客户项目内的生效内容。一次纠正/RCA 不自动新增 Rule；不无授权读取全量用户记忆、跨仓知识或上传项目资料。

任何向母仓或外部提供资料都属于另一个独立动作，必须明确精确对象、接收方、用途、许可、脱敏方案与外传授权；任一项未知就不执行。本方法只返回客户项目内的候选，不直接触发上传、同步或发送。

<a id="wf-step-return"></a>

### Step 6：返回对账清单

在既有交接位置返回，分已完成/延期/失败及真实待确认：

发布或对外沟通只消费已验证结果；草稿、提醒、处置候选和未核验整理结果不得被改写成已发布能力或完成事实。

```text
🧹 整理完成：<触发原因>

变更：
- 归档：<列表>
- 索引同步：<列表>
- tasks 状态推进：<N 条>
- feedback 草稿：<列表>（待用户拍是否入库）

发现：
① ...
② ...

待用户决策：
① <feedback 草稿是否入 <runtime-user-dir>/.../memory/？>
② <过程文档是否归档？>
③ <某 rules / SKILL 修订建议>
```

---

## 3. 合法进入与返回

需求、设计、开发、评审、测试及交接时都可定向整理，既有有效扫描/证据直接复用。不用完整 Router 重跑已明确的当前活动，也不为了单处引用修订新建全套工作。

有追踪或接力需要时，以当前项目制品、材料索引、实际引用和可见版本续接；由调用方/当前 Workflow 保存必要交回信息。本方法不创建并行的业务进度、焦点或活动状态账。

## 4. 专业方法与引用

本方法负责整理，业务结论仍归当前需求、design、ADR、任务、Issue 等各自权威。真缺口交原专业 owner 修订，索引只导航不重新批准；项目可用的外部知识来源可按需引用，不作为收尾“四项齐备”条件。

客户项目的材料、规则与方法候选交 [add-project-experience](../add-project-experience/SKILL.md)，不在回灌时直接写母仓或外部位置。跨活动与委派遵守 [multi-agent-coordination](../../guidance/multi-agent-coordination.md)；状态和成果归属遵守 [task-execution-workflow](../../guidance/task-execution-workflow.md)。

## 5. 责任与完成证据

材料 owner 负责用途、保留与处置决定，使用/接收负责人核验引用和交回成果；具体人从本活动/项目配置明确，真实确认需有对象、修订、范围与来源，Agent 名称不等于签字人。

当前执行者只消费真实可观察的 handoff/close/cancel/supersede/query 触点和已登记来源；事件缺失只报 `unknown`，取消请求不冒称终止。记录失败说明缺口，不补造调用、批准或送达，不重复所有材料正文；实际处置和登记完成分开验证。

退出证据包括保留/归档/回收及延期清单、现役/历史落点、实际引用检查范围、owner/接收意见、恢复引用、记录同步与未知项。只完成扫描就说已扫描；只产反馈就说候选；没有真实采用不声称方法已增强。

## 6. 失败模式与交回

| 错误 | 修正 |
|---|---|
| 归档前未确认 design 自包含 | 核对实际消费者与必要设计内容，缺口回设计负责人；不以关键词计数认定完整 |
| 缺实际退出或恢复依据 | 暂停对应处置，补真实证据，不伪造 commit/PR 或批准 |
| 汇总与权威任务不一致 | 核业务权威与证据，修引用/派生展示，不建第二状态账 |
| 把日期塞 `- [ ]` | 按 §10 SSOT，日期行末 |
| 替用户拍 feedback 入库 | 无实际采用授权则只保留候选；已明确采用时按指定位置执行，不伪造个人记忆能力 |
| 整理时直接改 rules / SKILL | 本整理步骤只提出候选，转已授权的方法演进活动后才能改生效内容 |
| 归档冲突信息（INDEX 跟现实不一致） | 升级用户决定按哪个为准 |
| 重复登记已有 feedback memory | 在已选且授权的位置查重复；未采用记忆工具不阻断普通反馈 |

### 原有导航的承接关系

- **触发场景**：材料产生或 handoff/close/cancel/supersede/query 时按当前范围定向整理
- **横切机制**：作为跨流程方法由当前调用方或 steward 按需使用，不注册独立 Workflow
- **产物归属**：[`task-execution-workflow §8 产物归属唯一表`](../../guidance/task-execution-workflow.md)
- **决策索引**：[`decisions-guide.md`](../../guidance/decisions-guide.md)
- **任务状态 SSOT**：[`task-execution-workflow §10`](../../guidance/task-execution-workflow.md)
- **rules / SKILL 归位**：[`agent-framework.md`](../../guidance/agent-framework.md)（回灌建议时按 L1/L2/L3 归类）
