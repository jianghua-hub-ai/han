# 多 Agent 协作与上下文管理协议

> **定位**：本文只规定 Lead 与专业 Agent 怎样划分边界、装载上下文、交付结果和处理失败；具体启动、等待、续派与中断语法由 Host 投影表达。
> **核心价值**：多 Agent 首先解决上下文隔离和责任隔离，其次才是并行提速。

## 1. 稳定原则

### 1.1 Lead 对交付负责

Lead 负责确认目标、限定范围、分派、复核证据和向用户交付。Agent 报告是输入，不是完成证明；涉及文件、测试、分支和提交的结论必须由 Lead 在真实工作区复核。独立评审结论属于 Reviewer，业务接纳属于项目指定的实际人，Lead 不代签。

### 1.2 一次委派只有一个有界目标

每次委派明确：

- 唯一目标与可验证完成条件；
- 固定仓库、worktree、分支和读写文件；
- 必读 Rule/Skill/Profile、设计、ADR 与真实代码；
- 已拍板且不得重新决定的事项；
- 允许/禁止动作、验证和必须停止的问题；
- 产出位置、返回摘要和未验证限制。

不得把互相独立的调研、设计、实现、测试和评审塞给同一 Agent，也不得让执行 Agent继续委派下级角色。

### 1.3 不并行写重叠产物

- 同一文件、同一契约或有明确先后依赖的写入必须串行；
- 文件与契约范围互斥且无前置依赖时才可小批并行；
- 只读调研可并行，但仍限定证据范围；
- 并行前声明精确所有权，参与者不得回滚、覆盖或格式化他人改动；
- 任务的 `[-]` 只表示真实认领或进行中，不是跨会话、跨设备的实时锁；发现重叠仍须依据当前可见证据协调。

### 1.4 固定执行边界

会话确认 worktree 与分支后，Lead 和 Agent 只能在该边界内写入、验证和交付。既有未提交内容默认属于用户或其他执行者；只修改明确归属文件，不能为得到“干净工作树”清理无关内容。

## 2. 直接执行、委派与接力

Lead 可以直接处理只读核对、解释、明确文案或上下文已充分的小改。凡要形成正式需求、架构、边界、特性设计、任务规划、实施、测试、独立评审、知识整理或对外材料成果，且入口已指定对应 Role，必须优先真实委派该 Role；不能因为主会话已经读到 Skill 就跳过正式 Role。普通问答、只读调查和非正式局部修改不触发委派。Host 不具备委派能力或委派失败时，先报告真实状态；只有用户明确授权，Lead 才能在原范围内接管，并标明没有 Role 产出，不得把主会话结果冒充 Role 返回。

一次协作链为：

```text
明确当前目标和真实制品
  → 选择直接执行或委派
  → 下发 Context Pack 与文件所有权
  → Agent 返回成果和证据
  → Lead 在真实工作区复核
  → 必要时由非作者 Reviewer 独立评审
  → 作者修复、原 Reviewer 复审
  → 更新对应项目制品并交付下一责任
```

需求、设计、Task、代码、Review、Issue、测试和 E2E 是接力依据；不得依赖另一会话中的隐含记忆。普通接力、返工、下一阶段和评审沿当前 Workflow 继续，不因换 Role 新建 Route 或运行状态。只有新会话首次请求，以及用户明确新建/切换工作、目标或 Workflow 时执行 Route。

## 3. Context Pack 契约

每次委派提供最小充分的 Context Pack：

| 字段 | 内容 |
|---|---|
| Agent / Reason | 计划调用的 Role 及为何需要独立专业执行 |
| Goal / Done | 唯一目标与可验证完成条件 |
| Scope | 仓库、固定 worktree、分支和任务边界 |
| Ownership | 允许读写的精确文件或目录；只读任务明确无写入所有权 |
| Inputs | 需求、设计、ADR、diff、接口、项目制品或本轮其他输入 |
| Read set | 被选 Role 启动后必须读取的 Role / Rule / Skill / Profile / 真实代码位置 |
| Known facts | 已验证事实及 file:line、revision 或命令证据 |
| Decisions | 用户已拍板且不得重新讨论的事项 |
| Allowed / Forbidden | 允许动作、禁止动作和外部状态边界 |
| Validation | 必须执行的验证和成功判据 |
| Stop | 方向性问题、越界、冲突写者或新增授权等停止信号 |
| Deliverables | 产出路径、返回摘要、证据和未验证风险格式 |

### 3.1 渐进披露

先给完成当前目标所需的最小集合；只有证据表明缺少上游契约时才追加精确文件。Lead 只下发 Read set 的逻辑引用或物理路径，不替执行 Agent 预读正文。不要复制整个会话历史、完整规则库或未来阶段材料。

### 3.2 事实、设计与决定分开

- 代码、Git 和命令说明现在可见的事实；
- 需求、design 与 ADR 说明目标行为；
- 用户已确认项说明本轮不可改变的决定。

三者冲突时必须报告来源和影响，不得默选一方。Git 只能证明当前工作区或已同步 revision 的可见事实；本地未提交内容要单列，其他机器或其他人未同步成果一律 unknown。

### 3.3 上下文超量时拆分

需要读取完整大型知识库、跨多个独立特性、无法给清晰文件边界，或同一阻塞反复出现且没有新证据时，应按真实产物边界拆分。接力摘要说明目标、有效成果、责任、待办、未决决定和下一步，不另造业务状态表。

## 4. 角色路由

| 工作 | 首选 Role | 主 Skill |
|---|---|---|
| 需求分析 | req-analyst | requirement-grill |
| 项目架构/决策 | arch-designer | architecture-design / adr-decision |
| 特性边界 | boundary-griller | boundary-grill |
| 详细设计 | feature-designer | design-tdd |
| 测试规划与真实验证 | test-engineer | test-design / e2e-verification |
| Task 与批次 | task-planner | task-decompose / batch-plan |
| 后端或前端实施 | backend-dev / frontend-dev | task-implementation |
| 独立评审 | reviewer | code-review |
| 知识整理 | steward | knowledge-curate |
| 只读证据调研 | explorer | research |
| 对外价值材料 | mo | release-communication |

Role 只决定专业视角和边界，具体目标仍由 Context Pack 决定。不得因为 Role 名相同就假设不同 Host 有相同能力。

## 5. 调用透明度

### 5.1 启动前

Lead 在启动前向用户说明 Agent/Reason、Goal/Done、Scope、Ownership、Inputs、Read set、Allowed/Forbidden、Validation、Stop 和 Deliverables。正式 Role 成果不得用“主会话直接执行”静默替代；仅当本次动作属于解释、只读、小改，或 Host 委派能力不可用且用户已授权接管时，才记录不委派原因。

### 5.2 启动结果

- 启动成功：报告 Host 返回的真实 Agent/Session ID 和状态；
- 启动失败：报告原因，核实没有执行者在运行，再说明是否由 Lead 在原边界内接管；
- 计划调用、已发起、取得 ID、已返回是不同事实，不能混写；
- Role 文件、计划名称或主会话模拟回答都不等于调用成功。

### 5.3 执行中与返回

长任务按 Host 能力报告有新增信息的状态；跑偏、越界或冲突时停止新增写入。Agent 返回至少包含状态、实际读取、产出文件、验证结果和需 Lead 决定的问题，大段成果落盘。

完成后分开报告：Agent 的返回与 Lead 的现场复核。无真实 ID 不声称委派，无真实返回不声称完成，无现场复核不声称交付闭环。

## 6. 文件所有权与共享 Git

写任务必须声明精确所有权，并明确两条纪律：

1. 你不是仓库中唯一工作者，不得回滚、覆盖或格式化他人改动；
2. 发现边界外文件也需修改时，停止受影响写入并报告 Lead，不自行扩范围。

同一时刻只有一个写入者拥有同一文件。Reviewer 默认只读被审对象，Explorer 全程只读。准备并行或汇总时，在当前可见范围识别共享文件、契约冲突、重复分工和遗漏；冲突未解决就保持待协调，不任取一方覆盖。

Git 共享的是实际已同步记录。汇报必须区分当前工作区可见状态（含未提交改动）与共同 revision；不得声称知道他人未同步成果，也不得要求先提交才能查看或继续本地已授权工作。

## 7. Lead 复核与独立评审

涉及文件修改时，Lead 至少核对：

1. worktree 与分支和约定一致；
2. `git status --short` 中本任务改动在所有权内，其他人的改动保持不动；
3. diff 内容、规模和消费者与目标一致；
4. 定向测试、语法、链接、构建或其他适用验证真实通过。

若约定提交，再核真实 SHA、文件清单和本任务残留；若明确不提交，交付 diff 即可，不为追求干净状态擅自提交。Agent 自报、命令启动或 Git 合并都不能替代 Lead 复核和整体验收。

需要独立评审时必须由非作者 Reviewer 审实际对象和 revision。finding 回真实作者修复，修复与复审串行，原 Reviewer 根据新 revision 复审。缺独立 Reviewer 就保持待审；作者自检不能冒充独立通过。

任务 `[x]`、批次通过和需求整体接纳是不同范围；任务三态与关闭条件见 [task-execution-workflow](task-execution-workflow.md)。

## 8. 失败处理

| 情况 | 处理 |
|---|---|
| 跑偏或越界 | 立即停止新增写入，保留证据，由 Lead 核实际工作区后决定最小修复、重派或接管 |
| 上下文不足 | 指出缺少的精确契约；只补所需内容，任务过大则按产物边界拆分 |
| 同一阻塞反复出现 | 返回已尝试路径、证据和缺失条件；无新证据不机械重试 |
| Reviewer finding | HIGH 必修复复审；MEDIUM 明确本轮处置或登记；LOW 可进入已有 backlog；不以改名自审造 PASS |
| Host 委派/等待能力不足 | 区分未启动、已启动但等待不可用、进度 unknown、权限不符和已返回；核真实 ID、文件副作用和写入者后再接管 |
| 写入者或远端成果未知 | 保持 unknown 并避免并发覆盖；能恢复原调用则先恢复，取消请求不等于已经终止 |

接管不能扩大原授权和文件边界，也不能把独立评审降为作者自审。安全、数据、发布、费用和破坏性动作仍需各自精确授权。

## 9. 与其他内容的关系

- 授权、G0、成果权威和任务三态：[task-execution-workflow](task-execution-workflow.md)；
- 内容对象和渐进加载：[agent-framework](agent-framework.md)；
- 当前 Workflow 的进入、返工与结束：对应唯一 Workflow Skill；
- 单任务实施：[add-task-implementation](../skills/add-task-implementation/SKILL.md)；
- 独立评审：[add-code-review](../skills/add-code-review/SKILL.md)；
- Git 操作：[add-git-workflow](../skills/add-git-workflow/SKILL.md)。

本 Rule 不复制这些 Skill 的步骤，也不要求为协作新建流程状态或第二任务系统。
