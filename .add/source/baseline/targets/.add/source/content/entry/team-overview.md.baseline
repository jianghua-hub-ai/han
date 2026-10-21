# 团队与工作流程导航

> 只用于看全局或寻找位置，不作为专业工作的前置。属于研发 Workflow 的工作先读选定编排 Skill，具体方法再按 `description` 选择；流程交接由对应编排 Skill 拥有，专业方法由各 Skill 拥有，职责由 Role 拥有。

## 1. 从这里去哪里

首次 Route 完成后，或当前会话已定位时：已知所属 Workflow 就读其编排 Skill，再选当前需要的方法；独立专业问题读匹配的 Skill，未知流程位置查下表。新会话首次请求或用户明确新建/切换工作、目标、Workflow 时才使用 `skill:route-work`。读取和执行透明度以 `entry:project-guide`、`rule:agent-framework`、`rule:multi-agent-coordination` 为准，不在此重写契约。

## 2. 工作对象

Workflow 是一类研发工作的稳定进入、阶段、交接、返工与结束合同。需求、设计、Task、代码、Review、Issue、测试和 E2E 等真实项目制品分别保存业务事实与进度；12 个专业角色不是常驻服务，也不必自动串成一条流水线。

同一个人跑全程、多人在同一需求不同范围并行或换人接力，都引用原业务成果。运行记录只补充可观测事实和恢复线索，不另建需求、任务或流程状态权威。

## 3. 两条客户研发 Workflow

| 常见工作 | 唯一编排方法 | 逻辑定义 |
|---|---|---|
| 产品、特性、小改与局部重构的交付 | `skill:delivery-workflow` | `workflow:development-delivery` |
| 正式 Issue 与故障闭环 | `skill:issue-handling` | `workflow:issue-resolution` |

`skill:bounded-change-workflow` 是 `development-delivery` 中途进入时的影响分级方法，不是独立 Workflow。`skill:knowledge-curate` 跨流程处理材料与项目知识；初始化、配置、升级、恢复和知识刷新属于项目生命周期方法，也不注册为客户研发 Workflow。

Route 是两类触发下的定位方法，不是第三条 Workflow，也不替代当前流程推进。

## 4. 开发与整体完成

开发任务、批次及整体交付的不同完成条件见 `skill:delivery-workflow`；中途小改与局部重构的影响分级见 `skill:bounded-change-workflow`。实施和独立评审的专业步骤分别见 `skill:task-implementation`、`skill:code-review`。

## 5. 测试入口

测试规划见 `skill:test-design`，真实环境验证见 `skill:e2e-verification`，失败问题闭环见 `skill:issue-handling`。本导航不复制 Case/Run、人工确认或 E2E 关闭条件。

## 6. 十二个专业角色

| 角色 | 职责入口 |
|---|---|
| 需求分析 | `role:req-analyst` |
| 项目架构 | `role:arch-designer` |
| 业务边界 | `role:boundary-griller` |
| 特性设计 | `role:feature-designer` |
| 任务/批次规划 | `role:task-planner` |
| 后端与非 UI 实施 | `role:backend-dev` |
| 前端实施 | `role:frontend-dev` |
| 测试规划与验证 | `role:test-engineer` |
| 独立评审 | `role:reviewer` |
| 只读证据调研 | `role:explorer` |
| 知识整理与维护 | `role:steward` |
| 对外价值材料 | `role:mo` |

角色的完整责任/禁止项不在这里重复。专业产出、人的业务确认与下一操作授权不互相替代。

## 7. 产物接力

产物归属与任务状态见 `rule:task-execution-workflow`；每个流程步骤的输入/输出和责任见上表的唯一编排方法。无需另读一份完整产物链才能开始已明确的当前活动。

## 8. 协作与上下文

Context Pack、真实 Agent ID、文件所有权、Lead 复核及失败接管见 `rule:multi-agent-coordination`。按当前活动提供精确引用，不传整库，不为准备委派替角色预读。

## 9. 非线性工作

新会话首次请求或用户明确新建/切换工作、目标、Workflow 时见 `skill:route-work`。澄清、续接、返工、下一阶段、评审和角色接力直接遵守当前合同，不重新 Route。

## 10. 维护入口

框架结构与三端内容规则见 `rule:agent-framework`；日常材料处置见 `skill:knowledge-curate`，项目采用与知识刷新按需读取对应生命周期 Skill。这张导航不是运行台账，不登记项目过程记录或母仓研发状态。
