---
name: task-decompose
description: 任务粒度跃迁——把 feature-designer 产的 Story 转化为可执行 task 条目，登记到 tasks*.md SSOT。两种模式：G 新建（greenfield Story→新 task）/ R 对账（reconcile——已有实现的特性先做四标签 gap-map 再登记）。task-planner 的 Step A，先于 batch-plan。
requires:
  - design-tdd
  - rule:agent-framework
  - rule:task-execution-workflow
  - rule:abstraction-principles
  - rule:design-doc-writing-style
  - batch-plan
  - test-design
produces:
  - <tasks-dir>/tasks*.md 新增或对账原 task（§10 SSOT 三态，共享事实只引用）
  - R 模式额外产 gap-map（每特性 Story × 四标签 × 证据，落特性节顶部）
---

# Task Decompose — 任务粒度跃迁

## 核心理念

按 [`agent-framework`](../../rules/agent-framework.md)，"按什么步骤把一件事做完"归 SKILL。本 SKILL 干的事：**Story → task 粒度跃迁**——这是一件独立的事，跟"批次组织"分离。

```text
Story（业务功能切片，feature-designer 产）
  粒度："实现用户注册流程"
  关注：业务价值

        ↓ task-decompose SKILL（本 SKILL）

task（可执行单元，task-planner 产）
  粒度："改 X 文件加 Y 函数 + 验收命令 Z"
  关注：可执行性 / 验收命令 / 依赖关系 / 形态共享
```

**"先 task 后 plan"**——task 是 SSOT 长期累积的资产；**拆完在同一规划 pass 内紧接着把它们全部分到批次**（batch-plan），不是拆一堆散 task 搁着、以后零散挑。完整的 task + 批次地图是多人协作的前提。

## 两种模式（G 新建 / R 对账）

task-decompose 支持两种调用模式，由 task-planner 在调用时指定：

| 模式 | 适用 | 起点 | 产出 |
|---|---|---|---|
| **G 新建（greenfield）** | 全新特性、仓库无任何对应实现 | Story | Story → 新 `[ ]` task |
| **R 对账（reconcile）** | 设计已改 / 已有存量实现的特性 | Story + 既有代码 + 冻结账本 | 先 gap-map 四标签 → 按标签登记 |

判断：特性涉及的代码**已有实现**（仓库存在对应文件 / 函数）→ 走 R；纯新建（仓库无任何对应实现）→ 走 G。若旧任务清单已冻结到 `<archive>/legacy-tasks-*/`，它只提供历史线索，不能替代现状代码与当前设计。

**R 模式是两级粒度，两级都要产出**：

- **第一级 · 对账 gap-map（Story 粒度）**：每特性 N 个 Story，每 Story 一个四标签 + 证据。只为"看全貌 + 判状态"，**Story 不是任务单位**。
- **第二级 · 具体任务（可独立闭环的执行单元，通常小于 Story）**：仅对待做/需改 Story 下钻为 task。每个 task 有单一行为目标、明确文件所有权、独立验证与评审闭环；不以固定分钟数或文件数机械切割，也不是一 Story 一 task。编号在 Story 下展开 `<feature>-S<n>.<seq>`。

两级共用 Step 2-5 粒度跃迁内核；R 模式前置 Step R 出第一级，Step 5 登记按四标签扩展。第一级 gap-map 得到本次范围的真实确认后——设计稳的待做/需改 Story 全部细拆；已建成项明确验证单元及原批次，不重复拆实现；废弃项留追溯。触及未拍板决策（`pending-<决策号>`）的 Story 暂停依赖部分，等决定后再补拆。

🔒 **为什么设计稳就要全拆、不搞 just-in-time**：多人协作时，只有**完整的 task 地图**（谁碰哪些文件、依赖谁）才能让人**分批认领、规划时避开冲突**。边做边拆 → 别人看不见你要碰什么 → 必撞车。所以"全规划（全拆+分批）→ 再分批执行"，不是"挑中才拆"。

## 何时启用

特性的稳定设计切片已有有效确认、准备形成执行安排时，task-planner 启动 Step A：

- 已有 design + 对应 Story，或项目既有等价设计/故事切片
- 当前切片的设计范围与确认来源有效；不要求无关部分全部完工
- 准备拆批次但还没拆——本 SKILL 是拆批次的前置

**不适用**：

- 业务需求阶段（活动 1）—— Story 还没出，谈不上 task
- 当前切片尚未稳定、缺真实设计决定——先补该缺口，不因为其他切片仍在设计就阻断已稳定范围
- 项目启动期初始任务（如"搭框架"/"建初始仓库"）—— 这类由用户手工写到 tasks*.md，不走本 SKILL

## 加载的规则

- [`task-execution-workflow.md`](../../rules/task-execution-workflow.md) §10 任务状态标记规则（checkbox 三态 + 行末日期 SSOT）
- [`abstraction-principles.md`](../../rules/abstraction-principles.md) §3 多形态隔离（共性改动列出各消费者，实际验证分别核对）
- [`design-doc-writing-style.md`](../../rules/design-doc-writing-style.md)（task 描述按写作准则——直陈事实）

## 输入契约

调用方（task-planner 主 Agent）必须提供：

| 字段 | 说明 |
|---|---|
| **调用模式** | `G` 新建 / `R` 对账（见 §两种模式） |
| **feature 标识** | 项目占位映射（见 项目入口 `<feature>`） |
| **stories 路径** | `<design-workspace>/<feature>/stories.md` |
| **design 路径** | `<design-workspace>/<feature>/design.md`（参考 §Step 4 Story 设计 + §Step 8 实现清单） |
| **目标 tasks 文件清单** | 按项目任务管理策略读取，不预设形态文件名；例如 `tasks.md` + `tasks-<form>.md` |
| **形态约束** | 项目支持的形态清单（见 项目入口 `<edition>`） |
| **task-id 命名规则** | Story 锚定展开 `<feature>-S<n>.<seq>`（详见 Step 3） |
| **当前任务与关联**（R 必需） | 原 task 引用或已核实无条目、当前批次关联、责任、已有验证/评审证据及未结束工作；明确未安排与无法确认分开，未知/冲突待核，不由 checkbox 猜认领 |
| **冻结账本路径**（R，如有） | 旧 task 只作历史线索，**非权威**；保留原来源，不以旧 `[x]` 替当前代码与设计核对 |
| **测试任务建议**（如有） | test-design 的场景/风险/层次/环境、验证范围与所有权建议；由 task-planner 实际登记并回填原覆盖映射 |
| **代码现状入口**（仅 R） | 涉及仓库 / 路径，供 Host 可用的文本/路径检索能力判定"已建成" |
| **未稳决策标记**（仅 R，可选） | 设计未拍定的 Story 标 `pending-<决策号>`（如 `pending-decision-002`），不强行对账（见 design-alignment-pass） |

## 输出契约

完工时：

- **R 模式先落 gap-map**：每特性一张"Story × 四标签 × 证据"表，写到该特性在新 SSOT 的特性节顶部（粗粒度对账全貌，先交付拍板）
- **落盘**：在原 `<tasks-dir>/tasks*.md` 新增或对账 task 条目（每条按 §10 SSOT + Step 5 四标签规则）；复用已存在条目，不复制状态库
- **摘要返回主会话**：
  - R 模式：四标签分布（已建成 / 待做 / 需改 / 废弃）+ pending 决策的 Story
  - 新增/复用 task 总数 + 按形态分布；已建成待验项的原任务、验证范围、批次/责任/证据及待协调项
  - 依赖关系预判（哪些 task 之间有硬依赖）
  - 跨形态共享 task 清单（如有）

---

## 流程

- **G 模式**：Step 1 → Step 2 → 3 → 4 → 5 → 6
- **R 模式**：Step 1 → **Step R（对账 gap-map）→ 用户拍状态** → Step 2 → 3 → 4 → 5 → 6（**设计稳的 待做 / 需改 Story 全拆**；`pending-<决策号>` 的 Story 停在 Step R 不拆）

### Step 1：读 Story 列表

读 `stories.md`，提取所有 Story 的：

- Story ID + 名称
- 用户价值
- 主成功场景 + 扩展场景
- 验收标准
- 依赖关系（Story 间）
- 纵切 / 横切划分
- 形态收窄（哪些 Story 在哪些形态上）

读 `design.md` §Step 4 Story 设计 + §Step 8 实现清单，提取：

- 每个 Story 涉及的服务 / 文件 / 接口
- 数据库 / 配置 / 缓存改动
- 测试要求

### Step R（仅 R 模式）：现状对账 gap-map

Step 1 读完 Story / design 后、Step 2 跃迁前，先对账。对每个 Story 核四类依据：

- **desired（要什么）**：本特性 design.md + stories.md
- **现状代码（建成了什么）**：用 Host 可用的文本/路径检索能力与 read 检查涉及仓库
- **当前任务**：先核原 task 的批次关联、责任、验证证据及未结束工作；当前账本仍是任务状态权威
- **冻结账本（如有）**：旧 task 只作历史线索，旧 `[x]` 不能替现状代码、当前设计和本次验收

给每个 Story 落一个标签 + 证据：

| 标签 | 判据 | 登记去向（Step 5） |
|---|---|---|
| **已建成** | 代码在 + 初判符合新设计 | `[-]`（建成，**待回归验证**）+ 证据 file:line——**不是 `[x]`**（见下方关键区分） |
| **待做** | 新设计要、代码没有 | `[ ]` |
| **需改** | 代码有、但不符合新设计（设计改了 / 实现有偏差） | `[ ]` + `(返工)` + 一句"原实现哪不对" |
| **废弃** | 旧实现有、新设计已砍 | 不进新 SSOT，gap-map 留一行追溯 |

**🔒 证据强制**：每个“已建成 / 需改”标签必须有当前代码/设计的可定位证据，并引用已有 task（如有）；只有旧任务引用不足以证明当前符合性。缺证据保留未知，不补造标签。

**🔒 已建成 ≠ 已完成（关键区分）**：

- gap-map 里**判定**"已建成"是轻量核对——用 Host 可用的文本/路径检索能力与 read 交叉看"代码在 + 初判符合新设计"，不跑重型验收。
- 但"已建成"**不等于** `[x]` 已完成——存量代码多是旧设计下写的、旧设计下测的，新设计下未必仍成立。故已建成 Story 登记 `[-]`（建成待验证），其**回归验证（跑测试 + 对新设计 review）**作为该特性实施批次的一部分，过 [`task-execution-workflow §10.3`](../../rules/task-execution-workflow.md) 的本 task 关闭条件（适用测试、独立评审及本 task 接收）才能 `[x]`；需求整体的真实 E2E 不自动下压为该 task 的前置。

**未稳决策**：Story 触及未拍板决策（如数据库选型 / SSO）→ 标 `pending-<决策号>`，不强行判标签，待决策拍定再回归。

产出 gap-map（每特性一张"Story × 标签 × 证据"表，落该特性在新 SSOT 的特性节顶部）。**先把 gap-map 交用户拍**（这批状态判得对不对）。确认后：废弃 Story 不下降（留追溯）；已建成 Story 优先复用原 task，补清验证范围、证据与关联；确无承接条目才登记最小验证单元并补 Step 3 五字段，不重复拆一套已建实现。已建成 `[-]` 不代表已认领，须交 batch-plan 核原批次/责任：无批次的待验项纳入验证批，有有效批次的不重复分配，未知/冲突待协调。**设计稳的 待做 / 需改 Story 一律续跑 Step 2-5 全部细拆**——只有 `pending-<决策号>` 的 Story 停在这、等决策拍定再补拆。

### Step 2：粒度跃迁——Story → task 候选

（R 模式只对 待做 / 需改 Story 跑本步；已建成 / 废弃 Story 不下降）

**拆分前先做两件事**（借 Spec-Kit Foundational gate + [`design-doc-writing-style §H`](../../rules/design-doc-writing-style.md)）：

1. **识别共享基础 task 作前置阻塞组**：多 Story 共用的公共包 / 接口契约 / DB schema migration 先拆出、排最前作"阻塞组"——它们没完成，依赖它们的 Story task 不能开工（避免"Story 干到一半才发现缺公共前置"）。其余 task 按 Story 分组排其后。
2. **定 TDD 档位**（见 Step 2 末 §TDD 三档），决定要不要单独拆"测试前置 task"。

对每个 Story，按以下规则拆成 1 ~ N 个 task 候选：

| Story 类型 | 拆分准则 | 判断重点 |
|---|---|---|
| 单模块局部行为 | 可在一个所有权边界内完成时保持 1 task | 是否能独立验证与回滚 |
| 同模块多文件 | 按行为闭环拆，不按文件机械一刀切 | 每个 task 是否自带测试与评审证据 |
| 跨模块/跨服务 | 按所有权与契约先后拆 | 公共契约先行，消费者依赖明确 |
| 数据模型/迁移 | 按 schema、兼容、迁移与消费者的真实依赖拆 | 混跑、回退和历史数据 |
| 交互式前端 | 按用户旅程与公共组件/数据契约边界拆 | 开发闭环与真实 E2E 分开 |

**粒度上限自检**：每个 task 候选必须：

- 一个清晰行为闭环，可独立评审、验证和回滚
- 文件所有权与同批其他 task 不重叠；确有强契约依赖时改为串行
- 有具体可执行的验收命令
- 边界清晰（哪些文件改 / 哪些不改）

**§测试闭环三档**（按 Story 风险选择，对齐项目 Layer 1 + [`test-design`](../test-design/SKILL.md)）：

| 档 | 何时 | 怎么拆 |
|---|---|---|
| **① 局部测试随实现（默认基线）** | 所有行为改动 | 测试是实现 task 的一部分，验收命令使用项目 profile 的真实测试工具 |
| **② 失败证据前置（显式触发）** | 核心算法、复杂状态机、契约敏感分支、安全路径或缺陷修复 | 在同一 task 内先建立失败证据；只有所有权确实独立时才拆前置测试 task |
| **③ 场景 / 集成 / E2E / 覆盖率** | 按设计风险提前安排，执行区分 batch Layer 2 与部署后 E2E | test-engineer 用 [`test-design`](../test-design/SKILL.md) 提出专项测试建议；task-planner 在本 SKILL 中登记独立 test task，再交 batch-plan，回填原覆盖映射 |

口诀：实现 task 永远自带局部测试（①）；高风险行为先证明会失败再实现（②）；跨边界测试（③）交 test-engineer。测试与实现改同一文件或强耦合时不强拆两个 task。

**②档判定标准**——命中任一即先建立失败证据：

- **多行为分支、每分支错了有安全 / 契约后果**（校验 / 鉴权 / 状态转换 / 路由分流）
- **实现的是被别人依赖的契约**（拦截器 / 协议 handler / API 契约 / SDK 公共接口）
- **核心算法 / 非平凡逻辑**（打分排序 / 调度 / 解析 / 计算），正确性不显然、易写错
- **复杂状态机 / 生命周期**（须挡非法转换）
- **修 bug**——先写能复现 bug 的失败测试再修（经典 TDD 修 bug 法）
- **test-engineer 测试规划标注 或 用户显式要求**

**保持 ①档**：无独立行为变化的机械改动，或已有测试能够清楚证明回归；不能因为改动小就省略必要验证。

「先让测试 FAIL」的意义：亲眼看测试因功能缺失而红、补码后转绿，才证明测试真在盯目标行为，杜绝"假绿测试"（先写码后补测、或断言根本没测到东西）。

### Step 3：补全 task 字段

每个 task 卡片保留原五字段，并补清任务类型和关闭范围（按 [`batch-plan SKILL`](../batch-plan/SKILL.md) 用法）：

| 字段 | 内容 | 示例 |
|---|---|---|
| **task-id** | **Story 锚定 + 设计顺序**：`<feature>-S<n>.<seq>`——`S<n>` 即 Story 在 design / stories 中的出现顺序，`.<seq>` 是该 Story 拆出的 task 序号 | 例：`<feature-a>-S1.1` / `<feature-a>-S1.2`（第 1 个 Story 拆出的第 1、2 个 task） |
| **目标** | 一句话 | "为 X 服务添加 Y 字段" |
| **文件清单** | 精确路径 + 新增/修改 | `<module>/path/to/existing`（修改）/ `<module>/path/to/new`（新增） |
| **验收命令** | 具体可执行 | 记录工作目录与命令参数，按顺序单独执行 `<profile-check-cmd>`、`<targeted-test-cmd>`；不要要求 `cd`、`&&` 或 shell 变量 |
| **依赖** | 前置 task `requires:` + 并行 `[P]`（判定见下） | `requires: <feature-a>-S2.1` / `[P]` |
| **类型** | 开发 / 专项测试 / E2E 等工作性质；用文字说明，不新增程序枚举或状态 | 开发 |
| **本任务关闭条件** | 当前任务的成果、验证、独立评审及适用接收；不只写含义不明的“真实接纳” | 局部用例通过、Review 通过，按 §10 核证关闭 |
| **后续验证关联** | 需要独立专项/E2E 验证时引用其 task/Case；待登记如实注明，确不适用才写无 | E2E 任务 `<feature-a>-S1.2` |

**拆分与关闭走查**：

- 开发 task 包含实现和必要局部测试；另有跨模块集成、性能、安全或真实 E2E 验证时，登记独立测试 task，写清环境、操作和证据。不把同文件的开发自测机械拆开。
- 后续测试通常依赖被测实现；“后续验证关联”不是开发 task 的反向依赖。确有测试前置要求时写清实际原因与依赖，检查不能形成循环。
- 登记前逐条回答“后续 E2E 未执行，这条开发 task 能否按自身条件关闭？”若仅因整体验收未结束而不能关闭，说明仍混有两个范围，先拆清。不省略本 task 必需的测试或评审。
- 对已有混合任务只做局部对账：保留原开发 task 编号和证据，将独立测试复用或补登记为另一 task。未变且证据有效的完成项不重做；缺局部测试/评审的开发 task 仍待完成，不批量勾选。

**`[P]` 判定**（借 Spec-Kit "different files, no dependencies"）：仅当**改动文件与同组其他 task 零重叠** + **不 requires 未完成 task** 才可标 `[P]`——用本表"文件清单"字段做重叠检查。

### Step 4：跨形态共享 task 同步

按 [task-execution-workflow §10.5](../../rules/task-execution-workflow.md)：先识别原任务和全部受影响消费者，保持共享事实的单一归属。

- 同一执行/验收范围的共享 task 只登记一条；其他计划、活动和形态视图只引用原任务，不同步登记多份状态。
- 已存在多形态业务任务时保留各自真实范围与历史来源并互引；确认是重复事实后才按项目约定对账，不删除未知条目。
- 通用改动列全消费者并分别验证；形态专属 task 只覆盖其授权范围。共享实现不证明另一形态已完成，也不扩大本次写账权限。
- 形态收窄在 task 的范围/验收中明示；真正独立的实现或验证工作各有唯一 task-id，不靠同一 ID 多份状态表达。

### Step 5：登记到 tasks*.md SSOT

按 [`task-execution-workflow §10`](../../rules/task-execution-workflow.md) SSOT：

```markdown
- [ ] <task-id> <task-name> ...
```

- **checkbox 三态** `[ ] / [-] / [x]`
- **G 模式**：初始登记一律 `[ ]`，无日期
- **R 模式按 Step R 四标签登记**：
  - 已建成 → 原 task 补验收/证据，缺条目才新增 `[-]` 验证单元；**不是 `[x]`**。保留实际批次/责任来源，无批次待验也须纳入全套批次，未知关联待协调
  - 待做 → `[ ]`
  - 需改 → `[ ]` + 描述内加 `(返工)` + 一句"原实现哪不对"
  - 废弃 → 不登记新 SSOT，gap-map 留追溯
- **日期一律行末标签**（`— 完成@日期` / `— 已建成@日期`）；绝不把日期塞 `- [ ]`（`[完成@日期]` 不是 checkbox 值）

**跨形态 ID**：保留同一 Story 锚；不同执行/验收单元使用不同 task 序号，共享事实只引用原 task-id，按 Step 4 对账。

**纲领任务**（按项目 `<task-id>` 前缀，见 项目入口）双日期：

```markdown
- [ ] <task-id> <task-name> ...  # 初始无日期；完工时加 — 完成@日期 — 验证@日期
```

按项目已采用的任务组织登记，不把按形态分文件变成通用要求。测试建议由 task-planner 实际登记后把原 task 引用回填 test-design 的覆盖映射；未登记或关联冲突明确交回，不报任务已安排。

### Step 6：返回摘要

```text
📋 task-decompose 完成：<feature>

新增 task：
- 核心（tasks.md）：N 个
- <form-a>（tasks-<form-a>.md）：M 个
- <form-b>（tasks-<form-b>.md）：P 个
跨形态共享：K 个（原任务及引用）
R 待验：原 task / 验证范围 / 批次关联 / 责任与证据 / 待协调项

依赖关系预判：
- 拓扑链：T-001 → T-002 → T-003
- 并行 [P]：T-004 / T-005 / T-006

任务范围：按开发/专项测试/E2E 返回数量、本任务关闭条件和后续验证关联；不从“需求整体未验收”推导开发 task 未完成。

下一步：同一已授权规划 pass 内接 batch-plan，把本范围全部未安排 task 和待验项组成批次，并引用已有有效批次；待协调项明确列出，不静默遗漏
```

返回真实当前活动的原任务/设计修订、产出/接收责任和确认来源；未答不当确认。仅授权拆 task 时交回结果及批次建议，不自动开始实施。完整批次图是原任务引用与依赖安排，不是第二任务状态表。

---

## 失败模式速查

| 错误 | 修正 |
|---|---|
| 把 Story 当 task 直接登记 | 必须粒度跃迁——Story 是业务切片，task 是可执行单元 |
| task 无单一行为闭环或文件/契约所有权冲突 | 按实际依赖拆分或串行，不用固定文件数机械切割 |
| task 无具体验收命令 | 补具体 build / test / 文本检索命令 |
| 共享 task 被复制成多份状态 | 按 §10.5 保留单一事实与各消费者引用，独立验收范围分别核对 |
| 把日期塞 `- [ ]` | 按 §10 SSOT，日期一律行末标签 |
| batch-plan 所需 task 尚未登记 | 先由 task-planner 补本步骤；已有有效条目直接消费，不为方法调用而重复登记 |
| feature-designer 顺手产 task 条目 | 不允许——feature-designer 产 Story，task 由本 SKILL 产（粒度跃迁归 task-planner） |
| 项目启动期初始任务走本 SKILL | 不适用——初始任务由用户手工写到 tasks*.md |
| 存量特性走 G 模式当 greenfield 登记成 `[ ]` | 错——有既有实现的特性必走 R 对账，否则已建成代码被误标未开始，历史白丢 |
| gap-map 标"已建成 / 需改"但无 file:line 证据 | 无证据 = 没对账，重来 |
| 旧账本 `[x]` 直接信成"已建成" | 旧账本非权威——必须以现状代码 + 当前设计交叉验证 |
| 已建成 Story 直接标 `[x]` | 错——已建成 ≠ 已完成；标 `[-]` 待回归验证，过本 task 的 §10.3 条件才 `[x]` |
| 设计未稳的 Story 强行判标签 | 标 `pending-<决策号>`，待决策拍定再回归，不硬对账 |

## 与其他规则 / SKILL 的关系

- **前置**：[`design-tdd`](../design-tdd/SKILL.md) 的有效设计/Story 切片，已批准依据可复用；`requires` 表示方法输入关系，不要求每次重跑全篇设计
- **后置**：[`batch-plan`](../batch-plan/SKILL.md)（消费本 SKILL 产的 tasks*.md 条目，组装 Plan；batch-plan frontmatter `requires: [task-decompose]`）
- **状态推进**：本 SKILL G 新增 `[ ]`，R 已建成待验可登记/复用 `[-]`；batch-plan 核实际关联再安排；完成由 Lead/指定关闭责任依 Rule §10 核证，steward 只对账
- **写作准则**：[`design-doc-writing-style`](../../rules/design-doc-writing-style.md)（task 描述按写作准则——直陈事实 / 不空洞动词）
- **形态隔离**：[`abstraction-principles §3`](../../rules/abstraction-principles.md)（消费者与实际验证）
