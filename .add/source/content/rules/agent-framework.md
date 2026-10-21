# Agent 框架元规则

> **定位**：ADD Entry、Workflow、Role、Skill、Rule、Profile 与 Host 投影的结构 SSOT。修改这些资产前必须先读本规则。

## 1. 内容对象与唯一责任

| 对象 | 唯一回答 | 不能承担 |
|---|---|---|
| Entry | Host 首轮必须知道的最小授权、安全、Route 触发和直达入口 | 全流程、全部角色或规则库 |
| Workflow | 一类业务工作怎样进入、分阶段、交接、返工、恢复和结束 | 专业方法全文、Host 调度或运行状态库 |
| Role | 谁以何种专业视角负责，输入、权限、返回和停止条件是什么 | 复制 Skill 步骤或整个 Workflow |
| Skill | 一项方法怎样完整执行、验证和异常恢复 | 永久业务状态、全局调度 |
| Rule | 某作用域内长期稳定的原则、约束和违规处理 | 长流程、运行状态或同义 Skill |
| Profile | 某技术栈或交付形态的显式可选实现变体 | 默认启用或项目硬编码 |
| Project knowledge | 客户项目事实、需求、设计、ADR、任务、术语和差异 | 通用方法副本 |

同一句完整约束只保留一个权威正文，其他位置使用逻辑 ID 引用。步骤属于 Skill，职责属于 Role，稳定约束属于 Rule，输出结构属于模板，项目事实属于项目知识。

Workflow 不是物理内容容器或运行引擎，其正文由唯一编排 Skill 拥有。客户研发只注册两条：`workflow:development-delivery` → `skill:delivery-workflow`，`workflow:issue-resolution` → `skill:issue-handling`。`skill:bounded-change-workflow` 是主交付中途的影响分级方法；知识维护是跨流程方法；项目初始化、采用、升级、恢复和知识刷新是生命周期 Skill，不另注册客户研发 Workflow。

Route 由 `skill:route-work` 拥有，只在新会话首次请求或用户明确新建/切换工作、目标、Workflow 时定位。普通续接、澄清、返工、下一阶段、评审和角色接力沿当前上下文继续。

## 2. Agent-neutral 要求

Canonical 不出现：

- Host 专属 frontmatter/TOML 字段、工具名、原生委派命令和发现目录；
- 某运行时版本的探测、等待或重试语法；
- 用户机器绝对路径、客户项目名、固定仓库/服务和凭据；
- 未启用语言、框架、协议或业务领域的默认要求。

Canonical 使用 `entry:<id>`、`workflow:<id>`、`role:<id>`、`skill:<id>`、`rule:<id>`、`profile:<id>` 逻辑引用。HostAdapter 在投影时映射真实路径和格式；ADD 自己的稳定 CLI 可出现在确定性操作 Skill 中，但不能充当意图分类、流程推进或业务状态权威。

## 3. Role 规范

Role 正文只允许六段：

1. 性格属性；
2. 职责与边界；
3. 主要方法（Skill 逻辑引用）；
4. 规则加载（Rule/Profile 逻辑引用）；
5. 委派接口；
6. 停止条件。

禁止在 Role 中出现 Step 1-N、命令脚本、完整模板、长案例、Host 工具白名单和运行时恢复流程。Role 不继续委派下级角色；活动交接返回 Lead。六段是职责骨架，不是压缩专业深度的篇幅配额。

### 当前角色阵容

| 阶段/横切 | 角色 |
|---|---|
| 需求 | req-analyst |
| 架构与决策 | arch-designer |
| 边界与详细设计 | boundary-griller、feature-designer |
| 任务规划 | task-planner |
| 实现 | backend-dev、frontend-dev |
| 测试与评审 | test-engineer、reviewer |
| 知识与传播 | steward、mo |
| 横切调研 | explorer |

一个 Role 只承担一种稳定职责。新增场景优先新增或扩展 Skill；只有出现新的责任主体和独立边界时才新增 Role。

## 4. Skill 与 Workflow 规范

Skill 必须包含稳定 ID、触发/排除场景、输入、完整方法、验证、输出和失败恢复。模板放 `references/`，不塞进 Role。

- 一份 Skill 只闭环一类方法；
- 前置用 `requires` 或正文条件明确，不把全部流程变成硬前置；
- 不依赖某技术的 Skill 保持 core，语言/框架专用 Skill 放对应 Profile；
- 运行时没有某能力时报告降级，不在 Canonical 编造客户端语法；
- Workflow Skill 只拥有稳定业务骨架和交接，专业步骤必须逻辑直链到唯一 Skill。

两条客户 Workflow 允许按真实项目制品中途进入、跳过不适用阶段和范围化返工。只有需求、设计、真实 E2E 是框架级软确认；安全、数据、发布、费用和破坏性动作仍需精确授权，不能被软确认替代。

## 5. Rule 与 Profile 规范

- Core Rule 只写跨语言、框架和项目成立的稳定语义；可以要求运行项目检查，不能默认具体命令。
- 专业方法顺序留在 Skill，不在 Rule 维护第二份步骤或 Workflow。
- 技术细则保留在 Profile，项目显式选择后才加载；未选 Profile 不投影、不评审、不当默认。
- Profile 可以追加或收窄实现要求，不能削弱 Core 的安全、权限和证据约束。
- 项目知识与经验按客户项目 owner 维护；确需改通用单元时保留显式 replacement，不以整文件覆盖客户增量。

## 6. 三层渐进加载

| 层 | 内容 | 加载时机 |
|---|---|---|
| L1 | 目标、授权边界、固定工作区、两类 Route 触发、验证与业务入口 | 根入口始终加载，保持最小 |
| L2 | 当前选定 Role/Skill、必要 Rule 和已启用 Profile | 当前方法/角色真正使用时读取 |
| L3 | 当前任务的需求、设计、ADR、真实文件、消费者和验收 | Context Pack 给精确引用，由执行者实际读取 |

完整规则库、全部 Profile、所有 Role 和全部 Skill 不得无条件内联根入口。每条常驻指令只写一次。

### 6.1 文件选择契约

- 新会话首请求执行一次 Route；上下文建立后仅用户明确新建/切换时再次 Route。属于两条研发 Workflow 的工作，先取得选定编排 Skill 的完整有效正文，再按请求与 `description` 选择当前专业 Skill；独立方法不强行套用研发 Workflow。总览和索引不是专业工作或正常续接的必经前置。
- 回答包含具体步骤、文件位置、状态、失败恢复或关闭条件时，必须实际取得对应 Skill；未读取时只能给高层入口。
- 查阅仓内现役 SSOT 不自动加载 research；只有目标本身是外部、历史、跨仓或不确定证据调查时才使用 `skill:research`。
- Lead 准备委派时只把 Role/Rule/Skill/Profile 的逻辑引用或物理路径放入 Context Pack，不替被选角色预读；真实降级接管时才自行读取。
- 选定 Role/Skill 必须取得完整有效正文；当前上下文确有同版本全文可复用，内容丢失或修订变化时补读。Rule 只选必需项，模板按实际创作范围读取。

## 7. 权限与 Host 映射

Canonical 只描述读取、搜索、执行、增量编辑、外部研究等能力类别。HostAdapter 根据真实能力映射最小权限：

- 评审和调研角色默认只读；
- 实现角色只写任务明确文件范围；
- 运行时不能物理限制时必须说明文字约束的降级风险；
- 不得把文字声明冒充沙箱或硬隔离。

构建期只做路径、格式和逻辑 ID 转换；运行期异常处理放 Host 的条件加载 runbook。专业入口生成精确 Role/Skill 链接；已选 Profile 的内容只投影一份，未启用不生成。父会话读过不等于子角色已经取得正文。

## 8. 修改与验收

Canonical 是内容唯一源。修改顺序固定为：

1. 在精确授权和文件所有权内修改 Canonical；
2. 构建各 Host 候选并检查逻辑引用、来源与项目定制保护；
3. 做资产、链接、中性、包闭包和体积检查；
4. 在母仓外 fixture 做真实发现、读取、Role/Skill 调用及负向未加载验证；
5. 失败不覆盖既有投影；通过后独立 cutover，由 Lead 在真实工作区复核。

ADD 母仓研发材料留在隔离研发区，不进入客户发行或 Host 根入口。投影、快照和客户替换内容不得成为第二份 Canonical；移动、覆盖、归档或删除仍须精确对象授权。

## 9. 自检

- [ ] Entry 是否仍是最小 L1，Route 是否只有两类触发？
- [ ] 客户研发是否只注册 `development-delivery` 与 `issue-resolution`？
- [ ] Role 是否只有六段，专业步骤是否只在 Skill？
- [ ] Rule 是否只保留稳定约束，未复制 Workflow/Skill 操作链？
- [ ] Core 是否出现未启用技术或客户固定值？
- [ ] Profile 是否按选择加载，客户 replacement 是否受保护？
- [ ] 真实 Host 验证是否覆盖正向加载、负向未加载和失败恢复，而不只检查文件存在？
