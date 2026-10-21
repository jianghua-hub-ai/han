---
name: delivery-workflow
description: "`development-delivery` 的唯一主交付方法，组织产品、特性、小改与局部重构从真实需求到真实 E2E，支持中途进入、连续推进、多人接力与返工。"
---

# 研发交付流程

## 1. 身份与适用边界

本方法唯一拥有 `workflow:development-delivery`。它固定主交付的进入条件、阶段骨架、输入输出、交接、返工、恢复和结束条件；专业步骤由引用的 Skill 承担，不在这里复制，也不实现为不可跳过的 CLI 状态机。

输入是用户目标与授权、固定工作区/分支、有关需求/设计/决定/任务及真实成果、可见的责任与验证条件。输出是本次范围内的真实项目制品、局部与整体验证、独立评审/复审、三处人的决定、未完成事项与可接手位置。

与当前交付或本流程环节无关的普通交流不进入本 Workflow；询问当前环节的专业问题仍按本流程定位适用 Skill，仅回答不改制品。正式 Issue、故障和复发由 `workflow:issue-resolution` 负责；小改、Debug 和局部重构可从本流程中途进入，并用 `skill:bounded-change-workflow` 判定影响，不另建第三条研发 Workflow。

<a id="wf-definition"></a>

## 2. 阶段骨架

下表不是强制瀑布。先用可核对的项目制品判断当前可进入阶段；已有前置足够时可跳过不适用阶段，并说明依据和剩余风险。仅当前阶段启用时，才完整读取其专业方法。

| 稳定阶段与专业职责 | 必需输入与方法 | 输出、确认及交回 |
|---|---|---|
| <a id="wf-step-requirements"></a>requirements · 需求分析 | 原始诉求、已有需求与采用约束；`skill:requirement-grill`，`role:req-analyst` | IR/SR/US、可观察验收、假设与 unknown 进入需求制品；在精确版本上取得需求软确认，或如实记录 pending/rejected/skipped/proceed-with-risk。 |
| <a id="wf-step-architecture"></a>architecture · 架构设计 | 已有明确处置的目标与现有系统，且确有项目级结构、跨服务契约或重大选型；`skill:architecture-design`，`role:arch-designer` | 系统对象、模块职责、4+1、契约与演进设计；不适用时复用现有架构依据，结构性取舍进入 decision。 |
| <a id="wf-step-boundary"></a>boundary · 边界梳理 | 需求、架构/ADR 与必要现状；`skill:boundary-grill`，`role:boundary-griller` | 五段叙事、四栏边界、接缝与未决；冲突交实际责任人处理，不由设计者默选。 |
| <a id="wf-step-design"></a>design · 详细设计 | 有效需求、边界、ADR 与对应事实；`skill:design-tdd`，`role:feature-designer` | 可实施 design 与必要 Story；在精确版本上取得设计软确认。边界缺口回 boundary，架构取舍回 decision。 |
| <a id="wf-step-test-plan"></a>test-plan · 测试规划 | 已稳定的设计切片、验收、风险与环境；`skill:test-design`，`role:test-engineer` | 风险/验收映射、最低足够层次、负例、回归、性能与真实环境建议；可与未受影响的设计工作并行。 |
| <a id="wf-step-plan"></a>plan · 任务与批次规划 | 有效设计/Story、测试建议、tasks SSOT 与真实实现状态；`skill:task-decompose`、`skill:batch-plan`，`role:task-planner` | 唯一 task 条目、依赖和批次；已有实现按 R 对账，不因代码存在直接标完成。 |
| <a id="wf-step-implement"></a>implement · 实施与局部验证 | 已批准设计或已确认预期行为、精确范围、消费者、修改授权和验收；`skill:task-implementation`，匹配开发 Role | G0 精确读取、最小实现、定向测试和实际交付；Lead 核 status、diff、验证与越界。失败先修当前任务，越界停止相关写入。 |
| <a id="wf-step-review"></a>review · 独立评审 | 精确对象/修订、上游合同、允许依赖范围和实际测试；`skill:code-review`，`role:reviewer` | 非作者独立 finding 与限制；问题回真实作者修复，再由原 Reviewer 独立复审。单 task 与跨任务整体分别核对。 |
| <a id="wf-step-integrate"></a>integrate · 集成与发布准备 | 范围内任务、整体评审、集成/必要性能/覆盖结果和真实发布授权；`skill:git-workflow` | 提交、制品、部署版本、检查和回退安排；push/PR/merge/发布均按实际授权，不因进入阶段自动获权。 |
| <a id="wf-step-e2e"></a>e2e · 真实旅程验证 | 可用版本/环境、独立 Case/任务、权限与人辅助安排；`skill:e2e-verification`，`role:test-engineer` | 真实 Run、三层断言、Issue 与重跑结果；在精确版本上取得 E2E 软确认。失败进入 `skill:issue-handling`，修复后重验原阻塞。 |
| <a id="wf-step-close"></a>close · 整体收口 | 当前需求的验收映射、集成/E2E、评审、三处决定与材料维护结果 | 区分已交付、未验、延期和责任；个人 task 完成或 Git 合并不替代需求整体集成与验收。按需使用 `skill:knowledge-curate` 和 `skill:release-communication`。 |
| <a id="wf-step-decision"></a>decision · 决策旁支 | 任一阶段中的结构性取舍、现状/候选/影响与原成果引用；`skill:adr-decision`，`role:arch-designer` | 决定写入对应 ADR 并同步受影响设计，随后返回原阶段；只暂停依赖未决取舍的部分。 |

## 3. 中途进入、继续与返工

- 用户明确单项专业工作且输入足够时可直接进入相应阶段，不补造历史；用户已授权完整目标且下一阶段前置有效时连续推进，到三处软确认、能力不足或新增授权点才停止受影响范围。
- 小改、Debug 或局部重构先用 `skill:bounded-change-workflow` 判断实际影响。既有预期和契约稳定时从 locate/implement/review 等必要位置进入；发现需求、边界、公共契约或架构缺口时返回本流程相应上游成果，只刷新受影响部分。
- 当前会话中的澄清、返工、下一阶段、评审和角色接力沿本 Workflow 继续，不重新 Route。只有用户明确新建/切换工作、目标或 Workflow 时才使用 `skill:route-work`。
- 原型已有时仍以真实设计和接口为准；mock 不冒真实 API。测试缺口回 test-plan/plan 明确责任，评审 finding 回真实作者并关联复审。
- 同一需求的并行任务按不重叠文件/契约和依赖分工；多项需求分别保存范围、责任和证据。共享 Git 只能说明已同步记录，本地可见未提交改动与他人未同步成果必须区分。

## 4. 专业方法与协作

阶段表只声明输入输出责任，不展开需求、设计、调试、实施、测试或评审的专业步骤。授权、G0、成果归属与任务三态见 `rule:task-execution-workflow`；真实委派、Context Pack、文件所有权、真实 Agent ID 和 Lead 复核见 `rule:multi-agent-coordination`。

Workflow 不要求普通问答、只读核对或小改都 spawn；但正式活动产出必须由阶段表指定的 Role 负责。只有 Host 返回真实 Agent/Session ID 才报告已委派；委派失败时保持未完成，除非用户明确授权 Lead 接管，并明确标注没有该 Role 的独立产出。无法独立评审时保持待审，不由作者换角色名自审。Role/Skill 在真正使用时读取完整有效正文，Rule/Profile 只选择当前必要项。

## 5. 责任、确认与完成证据

需求、设计和真实 E2E 是仅有的三处框架级业务软确认。成果提交后，主会话主动展示精确版本、范围、结果与未决风险，并按下表发起确认，不只返回 draft 状态。专业 Agent 返回不等于交接完成；主会话负责把成果交给人。各责任角色在实际交接时对应到具体人，用户明确兼任时仍分别保留确认内容和决定，不能擅自取消角色或从 AI Role、岗位和旧记录推断签字。

| 确认点 | 责任角色 | 确认内容 |
|---|---|---|
| 需求 | 需求提出人（产品经理或产品管理） | 目标、业务行为、范围和验收是否准确、完整 |
| 需求 | 设计负责人 | 需求是否清楚、能否承接设计；不在此批准技术方案 |
| 需求 | 项目负责人 | 交付范围、依赖及责任是否清楚、是否接受交接 |
| 设计 | 当前设计负责人 | 设计成果、需求覆盖与提交依据 |
| 设计 | 下一任务规划/开发责任人或指定技术代表 | 方案、契约、实现及验证条件能否承接 |
| E2E | 测试/E2E 结果责任人 | 被测版本、环境、逐 Case 结果与残留问题 |
| E2E | 指定业务验收责任人 | 业务验收范围、实际结果及未验风险是否接受 |

确认表逐行列出责任角色、实际确认人、确认内容、结论、时间和备注；人员及决定未知标 pending，不预签。人作出真实决定后，Host 调用 `add checkpoint`，在成果 `## 确认记录` 生成同源表格，并保留对应结构化记录，记录字段与双落点合同见 `rule:task-execution-workflow` §9。已明确的决定不二次询问。人可以拒绝、延后、改序、明确跳过或带风险继续，按原意和授权范围接续；未答复不当作跳过，跳过不写成已确认，不使用 CLI 强制会签或推进阶段。数据、权限、对外发布、费用以及删除/移动/覆盖等安全授权不属于软确认。

业务进度始终由需求、设计、tasks、代码、Review、Issue、测试和 E2E 等各自项目制品持有。CLI journal、Host 事件和会话说明只补充可观察事实与恢复线索，缺失即 unknown，不另造业务主键或阶段状态机。

单 task 收口核实现、适用测试、Lead diff、独立评审/复审和相关接纳；批次再核跨任务依赖、接口一致性、集成及必要性能/覆盖。需求整体完成还须核真实 E2E、适用人的决定、未完事项和材料归属，不能把局部 PASS 相加成整体通过。

## 6. 失败与恢复

| 情况 | 处理 |
|---|---|
| 需求、边界、契约或架构缺口 | 给精确证据和影响，暂停依赖部分；决定范围后回实际 owner，只重做受影响成果与验证 |
| 实施或评审失败 | 回真实作者最小修复，关联原 finding 后由非作者复审；不循环自评到 PASS |
| 测试规划缺漏或实测失败 | 前者回 test-plan/plan，后者回失败实现或独立 Issue；保留版本、复现和未验子集 |
| 环境、模型或人辅助不足 | 说明受影响验证和准备责任，继续不依赖它的已授权部分；未运行不算通过 |
| 共享文件冲突或他人写入未知 | 保留现状与双方证据，按可见范围协调；不覆盖、不假称掌握未同步成果 |
| 用户暂停、取消或缩小范围 | 停止受影响新增动作，保留已发生结果、责任和待处置事项 |
