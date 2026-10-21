---
name: add-issue-handling
description: 处理测试缺陷、现网问题、告警或安全问题；按严重级别组织登记、定位、授权修复、回归、独立评审、两方验收和沉淀。只授权定位则返回证据不关闭问题；新需求另走交付流程。；仅适用于 Codex；其他 Host 不使用此投影。
requires:
  - systematic-debugging
  - task-implementation
  - code-review
  - knowledge-curate
  - rule:multi-agent-coordination
---

> 仅适用于 Codex；其他 Host 不使用此投影。


# 问题单驱动(ADD 第二入口)

## 1. 身份与适用边界

本方法拥有 [add-issue-handling](SKILL.md#wf-definition)，继续使用项目已有 Issue 文档/台账，不另建问题数据库。问题入口复用原专业方法，不是重造开发流程。

**启用**:有问题单进来——测试发现的缺陷 / 现网客户问题 / 监控告警 / 安全漏洞。
**不启用**:新需求(那是第一入口,走需求 → 设计 → 实现的完整流程)。

输入：问题/告警/Case 引用及实际版本/环境、影响、固定工作区/分支、原 Issue 与关联主交付 ProjectArtifact 引用、授权范围、产出/接收责任人与验收依据。缺事实列未知，不用级别或职业名称推定修改、发布或签字权。

输出：本次定位/修复/回归/评审/验收证据与版本、未决事项、状态及来源、返回阻塞核对结果、必要复盘和候选改进。修复已合入、Issue 已接受关闭和原工作已完成是三个不同结论。

### 严重度与节奏

| 级别 | 含义 | 走法 |
|---|---|---|
| **P0** | 影响生产可用，紧急 | 按项目实际应急授权先定位、最小止血、回归/冒烟、加速独立评审；不为已有有效依据重跑完整设计，但结构决定/权限/隐私不能跳过，后续闭环不遗漏 |
| **P1** | 核心功能受损 / 重大客户影响 | 按影响组织紧凑修复，必要契约/设计补齐后实施；核本次修订 |
| **P2** | 非核心功能，可排后续版本 | 保留 Issue 与排期，范围明确可局部修；涉及正式需求/结构变更才关联交付工作 |
| **P3** | 体验 / 优化类 | 明确是否缺陷或新需求，按真实优先级安排，不能仅因低级别强制重走完整设计 |

响应、止血、发布和复盘时限取项目既有策略/本次人类决定；母本不预设统一小时数、会议时长或擅自切 hotfix 分支。发布与同步按实际授权和项目 Git 合同进行。

<a id="wf-definition"></a>

## 2. 步骤合同

| 步骤 | 输入与实际动作 | 输出与继续条件 |
| --- | --- | --- |
| <a id="wf-step-register"></a>register · 登记 | 核项目已有 Issue，关联实际告警/Case/用户问题、版本环境、发生时间、影响与责任；按项目编号约定登记，已有项续接不另造。 | 原问题引用、关联 ProjectArtifact 和未知项；根因/复盘按项目选定位置保存，台账留链接，不要求 Memory 产品。 |
| <a id="wf-step-classify"></a>classify · 定级 | 按上表和项目策略核影响、紧迫性、止血权限及处置范围；有人已确认的优先复用。 | 严重度及决定来源；只影响节奏，不授予权限或自动批准后续动作。 |
| <a id="wf-step-diagnose"></a>diagnose · 定位 | 用 [add-systematic-debugging](../add-systematic-debugging/SKILL.md) 核复现、假设、实验和隔离，复用同版本有效失败证据，不先乱改。 | 根因与位置、已排除证据、未知、最小建议范围；只授权定位时到此返回事实，不实施、不把 Issue 标关闭。 |
| <a id="wf-step-remedy"></a>remedy · 修复 | 在真实修改授权和已确认预期/范围内用 [add-task-implementation](../add-task-implementation/SKILL.md) 最小修复；真正缺需求、边界、契约或架构决定时停受影响写入，以精确 ProjectArtifact 引用衔接 [add-delivery-workflow](../add-delivery-workflow/SKILL.md#wf-definition) 的对应环节。 | 实际变更/版本和局部验证；有效设计直接复用，不为所有 Issue 新写设计。主交付制品更新后仍由本 Issue 接续 remedy/regress，原 Issue 不移交关闭权威。 |
| <a id="wf-step-regress"></a>regress · 回归 | 核本次版本与根因专项测试、消费者回归；复杂场景规划用 [add-test-design](../add-test-design/SKILL.md)，真实联通必要时用 [add-e2e-verification](../add-e2e-verification/SKILL.md)。P0 加适用冒烟。 | 实际用例/命令/结果/环境及未测范围；不以旧版本通过、新 mock 或部署成功代回归。复用实施已有效验证，只补缺项。 |
| <a id="wf-step-review"></a>review · 独立复审 | 由非作者用 [add-code-review](../add-code-review/SKILL.md) 核真实 diff/设计、风险和回归，P0 可加速但不自审。 | findings 与范围；修复回作者步骤，新尝试关联原 finding，复审关闭原问题，不覆盖失败历史。 |
| <a id="wf-step-accept-close"></a>accept-close · 验收关闭 | 核修复、当前版本回归、独立复审及适用发布/接收条件；产出负责人交付本次证据，接收负责人按当前对象/修订/范围作真实接受决定。 | 两方来源齐备才由指定责任人接受关闭；未答/拒绝/未测保持未完成，不把提交、工具成功或定位交回当 closed。 |
| <a id="wf-step-learn"></a>learn · 沉淀交回 | 用 [add-knowledge-curate](../add-knowledge-curate/SKILL.md) 整理必要根因/时间线/有效措施、引用与材料安排；P0/P1 必有相称复盘，使用 [RCA 模版](references/RCA-%E5%A4%8D%E7%9B%98%E6%A8%A1%E7%89%88.md)。 | 复盘/候选及真实待办；永久规则改动只形成候选，不自动采用。按关联 ProjectArtifact 核查本 Issue 所阻塞的条件，不能清掉主交付的其他阻塞或替整个工作标完成。 |

## 3. 合法进入与返回

- 已明确 Issue 和当前环节时核对既有证据直接接续，不每轮重跑 Router；正常步骤推进由本流程完成。
- 只授权定位或只授权单一回归/复审时，返回对应成果及未决事项，不自动推进修复或关闭。已有 Issue 保持其真实状态；一次普通诊断不强制新建 Issue。
- 结构缺口通过精确需求/设计 ProjectArtifact 与版本引用衔接主交付，只处理必要缺口；制品更新不自动接受原 Issue，返回后仍在本次 Issue 验证实际修复和所有适用条件。
- 验收拒绝、问题复发或旧结论失效时，按项目准则重开原 Issue，追加原因、当前版本/环境、真实决定来源和本次待处理范围；旧关闭事实保留为当时版本的真实结论，连同旧失败、修复与验收链继续可查，重新定位当前需要的步骤，不沿用失效通过记录，也不改旧证据制造“从未失败”。
- 可选批量结果评审会可由测试负责人和相关方按实际约定组织；逐单保持对象、版本、结论、责任和证据。不因批量“已评”替所有接收人自动签字；共性只形成改进候选。

## 4. 专业方法与资源

本方法只组织 Issue 链，不复制专业步骤：定位→[add-systematic-debugging](../add-systematic-debugging/SKILL.md)；修改→[add-task-implementation](../add-task-implementation/SKILL.md)；规划回归→[add-test-design](../add-test-design/SKILL.md)；必要真实联通→[add-e2e-verification](../add-e2e-verification/SKILL.md)；独立评审→[add-code-review](../add-code-review/SKILL.md)；整理→[add-knowledge-curate](../add-knowledge-curate/SKILL.md)。真正设计缺口在 [add-delivery-workflow](../add-delivery-workflow/SKILL.md#wf-definition) 的相应环节用 [add-design-tdd](../add-design-tdd/SKILL.md) / [add-adr-decision](../add-adr-decision/SKILL.md)，不是先修后补批准。

[RCA-复盘模版](references/RCA-%E5%A4%8D%E7%9B%98%E6%A8%A1%E7%89%88.md) 定义事件、复现、时间线、根因证据、临时/根因方案、效果和改进字段，不另设审批或自动规则写入。需要持续改进时由 [add-project-experience](../add-project-experience/SKILL.md) 在当前项目内保留候选、评价和采用决定；向外部或母仓提供候选及证据另需精确授权。项目未启用 Wiki/Memory 不阻断 Issue 闭环，已有证据位置仍须可查。

## 5. 责任与完成证据

问题处置产出负责人负责本次修复与验证完整性，使用/接收负责人负责责任人确认与验收关闭；实际人从本次 Issue/关联制品及交接决定中明确，不从初始化配置预填的固定人推定。AI 专业角色可执行，不从其名称、总结或工具权限推定人已批准；真实确认必须带来源和适用对象/修订/范围。

需要连续跟踪、接力或并行时，以原 Issue、Task、Review、测试/E2E 和需求/设计等 ProjectArtifact 的精确引用与版本保存输出、未决、责任和下一步。子问题与主交付只通过项目制品互相引用，不建立第二套业务状态。

定位、修复、回归、独立复审和接受/拒绝分别落到对应 Issue 与关联制品；缺少可记录依据时如实说明缺口并保留当前成果，不能猜调用 ID、补造事实或代替责任人确认。定位不等于关闭，修复完成也不等于已验收。

输出明确区分“定位完成”“已修复待回归/验收”“已接受关闭”“复盘/经验候选待处理”。本次有必要后续措施时保留责任和期限，不因 Issue 修复而抹掉待办；整体工作的关闭仍由关联主交付制品的负责人按其独立条件核验。

## 6. 失败交回

| 情况 | 处理 |
| --- | --- |
| 不能复现或根因证据不足 | 返回已证/排除/未知与下一安全实验，不猜根因或盲修；紧急止血仍须实际授权和验证。 |
| 修改、发布或结构决定越出授权 | 只暂停受影响动作，请对应人决定；P0 不越权限/隐私/安全边界。 |
| 回归或独立评审仍失败 | 关联本次失败版本与原 finding 回正确步骤修复；不重造问题单或宣称完成。 |
| 接收拒绝/未答、版本不符 | 保留未验收及具体缺口；已有旧关闭不能替当前复发问题接受。 |
| 记录失败或写者状态不明 | 报缺口、保留证据，按 [multi-agent-coordination](../../guidance/multi-agent-coordination.md) 核现场与影响后再接管；不以超时重派并发写者。 |
| 唯一证据拟回收或复盘拟直接写 Rule | 交知识整理核引用/保留；规则先留候选，未经明确采用不入生效内容。 |
