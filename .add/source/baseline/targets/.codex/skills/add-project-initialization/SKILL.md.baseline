---
name: add-project-initialization
description: 在首次采用或获准重新初始化时引导项目确认；初始化或客户明确通过 ADD 管理 Profile 时，形成完整 Profile 候选并移交现役采用链。；仅适用于 Codex；其他 Host 不使用此投影。
---

> 仅适用于 Codex；其他 Host 不使用此投影。


# 项目初始化

本方法是 ProductRelease 交付给客户项目的首次采用 Skill，由 Coding Host 按 `add init` 返回的精确方法引用读取。它不是 ADD 母仓执行方法，也不是客户研发 Workflow，不注册第三条研发流程。日常会话只有在客户明确要求通过 ADD 新建、修改、管理或投影 Profile 时，才按 [add-project-adoption-workflow](../add-project-adoption-workflow/SKILL.md) 的入口复用本方法中的完整 Profile 候选合同，不重跑其他初始化分组。安装、采用版本、升级、恢复和移除由 [add-project-adoption-workflow](../add-project-adoption-workflow/SKILL.md) 负责；初始化后的知识刷新由 [add-project-context](../add-project-context/SKILL.md) 负责。

首次采用的输入是受控 InitializationStage、固定项目根、所选 Host、获准发现范围和客户说明；日常 Profile 意图的输入是现役采用事实、所选 Host、客户本次目标和获准读取范围，不建立新的 bootstrap stage。输出是带证据的 findings、逐项 decisions、Manifest/项目知识或完整 Profile 候选、精确 ChangePlan、应用及验证结果。发现授权不等于写入授权；本方法不自动写、删或覆盖任何客户内容。

## 1. Scope

先核对本次对象和边界：

- 确认项目根、首次采用或获准重新初始化的意图、所选 Host，以及哪些目录、文档、配置和外部来源可以读取。
- 核对 `stageRef`、stage/release/instruction digest；摘要不符、stage 过期或项目对象变化时停在 `needs-attention`，不靠会话记忆继续。
- 分开记录只读发现范围、候选形成范围和可能写入范围。只有 bootstrap 暂存可由首次 `add init` 建立；现役配置和投影在精确计划获授权前保持不变。
- 尚无 HostProjection 时，只读已安装 ProductRelease 返回的精确方法引用，不假定客户项目已经装有或会自动发现本 Skill。

Scope 不清只追问会改变结果的项目、Host 或授权边界；不得把全仓读取、外部上传或写权限当默认值。

## 2. Discover → Confirm：按组推进

绿地主要依据项目目标、需求、已选技术和客户说明；棕地再读取获准范围内的现有代码、配置、构建测试和文档。两者使用同一组顺序，**当前组确认并交回结果后才问下一组**，不在第一轮列出完整 Manifest 参数表让客户一次回答。

| 顺序 | 当前组要解决的问题 | 对应的结构化输出 |
| --- | --- | --- |
| 1. 项目与范围 | 这个项目要做什么；本次覆盖哪些仓、模块及其职责？先展示从现有材料识别的名称、目标、目录和模块，问客户哪里不准或缺失 | `project.name/purpose/repo_layout/module_map` |
| 2. 开发与约束 | 实际用什么技术、如何构建和验证；哪些项目安全约束影响 Agent 工作？从真实文件提取可运行命令的候选，不猜命令可用，也不先执行客户脚本 | `project.enabled_profiles/check_commands/security_policy` |
| 3. 知识来源 | 哪些现有文档、规范和架构知识应供 Agent 使用；是否已有可读取的 Code Wiki？对每个来源核入口、适用范围和对应代码版本，形成引用或最小知识候选 | `sources` 和项目知识候选 |
| 4. 使用入口与成果位置 | 选定 Host 如何接入；需求、设计、任务、Issue 等成果放在哪儿？展示产品默认位置与现有入口的差异，请客户确认要保留或调整的部分 | `artifact_profiles/runtimes/paths` |
| 5. 既有定制与维护 | 现有 Role、Skill、Rule、Hook 或客户定制是否与新投影相遇；各工作环节的职责和材料提醒怎样接续？只列真实发现的冲突和需要客户决定的保留项 | `owners/replacements/materials` |

每组都先做只读发现，为组内每项形成 `itemId`、识别值、证据及摘要、建议值、`observed | reported | unknown | conflict` 和限制。然后**只问当前组**：已识别值也展示给客户核对；未知只问取得该值所必需的问题；冲突并列证据和影响，不让模型默选。提问用项目语言，通常一次不超过三条聚焦问题；字段名是内部落盘映射，不作为一长串表单直接抛给客户。

Code Wiki 仅是棕地知识来源之一。客户已有 Wiki 时，先定位其本地文件、Host 工具或浏览器入口，试读与当前代码对应的页面；确认可访问后，将入口 `locator`、访问方式 `access.via`、覆盖范围 `scope` 和对应代码版本登记为可选 `sources`。需要新建索引或向外部工具提供代码时，先取得相应读取/传输授权。没有 Wiki 或暂时不可访问，继续使用获准源码和现有文档，当前组只标记未核实的知识范围，不阻塞其他组。

每组确认后，给客户一份简短结果：本组已确认的值及证据、待补/冲突项、本组状态（已确认 / 部分待补 / 有冲突）、已写入当前 stage 的累计 findings/decisions 引用和下一组。使用同一 stage 按累计结果更新受控暂存，不能因后续组尚未完成就把本组报成失败；`awaiting-confirmation` 是整体阶段状态，不是对已确认组的否定。跨会话恢复先读 stage 与累计决定，仅重核证据变化的项，不重问有效决定。

对每项只接受以下一种决定：

| 决定 | 含义 |
| --- | --- |
| `accept` | 接受当前建议值，并绑定该 finding digest |
| `edit` | 客户给出明确 effective value 后采用该值 |
| `defer` | 暂缓，不进入本次计划或应用，记录继续条件 |
| `reject` | 拒绝本次采用，不进入本次计划或应用 |

`unknown` 先请求必要补充，再把补充后的值和证据展示确认；`conflict` 展示各方证据与受影响范围，由客户决定 edit、defer 或 reject。客户可对**已逐项展示**的当前组明确表示“本组所列值均正确”，Agent 仍须分别生成各项决定；泛泛的“继续”不等于接受。`defer/reject` 不进入计划或应用，未受影响项可以继续。

决定必须绑定精确 finding digest、范围和实际决定来源。证据、建议或对象摘要变化后，原决定变为 stale，只重核受影响项；批量“继续”或模型自报不能代替逐项决定。Agent 可以解释和起草，但不能替客户接受项目事实。

## 4. Form

模型根据 active 的 `accept/edit` 决定引导形成 Manifest 候选，不要求客户从空白文件手填完整 YAML，也不让客户发明 schema 或 Host 路径默认值：

- `add.manifest.yaml` 候选只保存结构化采用选择、知识引用和 replacement 引用；项目知识正文仍归客户项目或获准外部来源。
- 三处成果确认的实际提交人、接收人和决定只在该成果交接时记录；初始化不填人名、不预签，未采用项目审批表单时使用 ADD 的确认记录。
- Adapter 的安全路径默认值也要展示最终值、证据、冲突和建议，仍由客户逐项决定。新项目选择 Codex 时，默认入口为根 `AGENTS.md`，team overview / roles / guidance / skills / profiles / runbook 分别在 `.codex/team-overview.md`、`.codex/agents`、`.codex/guidance`、`.codex/skills`、`.codex/profiles`、`.codex/runtime-runbook.md`；不把 `.agents/skills` 当作本端默认值。
- 同时形成最小项目知识候选，带来源、版本、适用范围、payload 摘要和限制；不把整份 Wiki、源码或历史材料复制进 Rule。
- `defer/reject`、未处理 `conflict` 和必要 `unknown` 留在 unresolved，不用占位值伪装完整。

Form 只产生受控候选和摘要，不写现役 Manifest、根入口、HostProjection 或客户知识。候选须能从每个字段追溯到 finding 与 active decision。

### 完整 Profile 候选

只有项目初始化，或日常会话中客户明确要求“通过 ADD”新建、修改、管理或投影 Profile，才进入本段。普通工作、一般问答、临时修改和 Agent 偶然发现都不形成候选；日常入口也只复用本段，不重新盘点整个项目。

AI 以已确认的项目事实、客户本次目标和适用的官方模板为输入，形成包含 README、rules、skills/references 的完整项目 Profile 候选。模板提供可核对的起点，不能用默认值替代未确认事实；没有适用模板时可以形成 `origin.kind: project` 的完整候选，但仍须满足同一确认边界。Profile 内 Rule/Skill 随 `profile:<id>` 整体拥有，不转交 [add-project-customization](../add-project-customization/SKILL.md) 的普通项目 Rule/Skill 合同。

按下列四组依次展示并逐项确认；每组都给出候选值、来源与摘要、差异、unknown、冲突和对现役面的影响：

| 组 | 必须确认的完整对象 |
| --- | --- |
| 身份与适用范围 | 稳定 Profile ID、description、适用/排除范围，以及新增、修改或复用关系 |
| 采用的项目事实 | 候选实际使用的技术、架构、命令、安全约束和其他已确认项目事实；未确认事实不猜 |
| 完整 README/rules/skills 内容 | 完整目录树、每个成员正文、逻辑引用、来源模板和目录摘要，不以名称或文件清单代替内容确认 |
| 所选 Host 投影影响 | 每个所选 Host 的目标、增删改保留项、同名冲突和未选择 Host 的不变范围 |

四组都沿本方法既有 `accept / edit / defer / reject` 决定合同逐项绑定精确摘要。`unknown`、冲突和未确认项保持显式；四组任一未完成时，不进入项目有效源，也不进入任何 Host 投影。AI 可以继续完善候选，不能把“继续”、模板命中或模型自报解释为确认。

四组均达到可应用条件后，才把完整实例、`enabled_profiles + profile_instances` 绑定和 Host 影响作为一个整体，移交现役 `init/config/upgrade` 的 `candidate → plan → apply → verify` 链。初始化候选进入当前 adoption stage；日常明确意图进入对应 config/upgrade operation。目标不存在时形成 create，内容完全相同为 no-op，内容不同时保留现役实例并标记 pending；无关已确认候选可以继续。这里不建立 Profile 专用状态机、Role、Workflow、命令或恢复协议。

单独调用 `materializeProfileTemplate`（materialize）、项目 Profile `discover`、`compile` 或 `apply`，或其中任何一步单独成功，都只表示底层原语完成，不能产生 Profile 候选、确认或写入授权。普通工作始终是零候选、零写入，不改写 `.add/project/profiles/**`。

## 5. Plan/Apply

把同一 stage 的候选交给 `add init` 做 schema、路径、来源引用、摘要、决定完整性和冲突的机械校验。计划必须逐项展示 `create`、`update`、`preserve`、`conflict`、`skip`，并给出目标路径、ownership、before/after 摘要、所选 Host、恢复依据和 `planDigest`。

默认只返回候选与计划，不写现役项目。Host 向客户展示精确变化和限制；只有客户授权当前 `planDigest` 后，才调用：

```text
add init --apply <planDigest>
```

摘要、项目现场、所选 Host 或授权范围变化时，计划立即失效并重新确认。未授权不得写入；本方法不自动写客户文件、不自动删除任何对象、不自动覆盖现有内容。只生成所选 Host 的投影，未选择 Host 和 marker/field ownership 外的字节保持不变。

Profile 与其他初始化内容共用 `plan → apply → journal/rollback → verify`：新增目录由通用计划创建，完全相同不重复写入；冲突组由客户或 AI 处理后重新走同一条链，不建立 Profile 专用状态机或恢复协议。禁用 Profile 只撤下派生投影，保留 `profile_instances` binding 和项目 Profile 源，便于以后重新启用；不把禁用解释为删除客户源。

## 6. Verify

应用后核验分层，不把一种证据冒充另一种；客户初始化不要求登录 Host、执行真实请求或补齐 smoke 结果：

1. 核对 journal、实际 Manifest/知识引用、文件摘要及静态校验，逐项对账计划中的 create/update/preserve/conflict/skip。
2. CLI 返回静态应用完成后，正常会话即可使用正式 HostProjection；未选择 Host 的内容不应被新增或改变。
3. 独立的线下发布验证才在已配置好的测试环境里执行真实 Host 加载和代表工作；其结论不能从初始化命令成功推导。

客户初始化静态核验失败时只报告已完成部分、失败范围和继续条件。已有副作用按原 operation journal 选择 resume/rollback；第三种现场内容保留为冲突，不另开操作覆盖。线下 Host 验证缺失时只将 Host 效果记为 unknown，不反向阻塞已静态完成的初始化。

## 结果交回

交回时列出 stage/release/instruction/candidate/plan 摘要、每项决定、实际新增/修改/保留/冲突/跳过、知识候选落点、静态结果、恢复依据、未完成项和下一责任人；不要求客户补交真实 Host 证据。临时候选或 bootstrap 材料的归档、移动或删除另按精确对象取得授权；初始化完成本身不是清理许可。
