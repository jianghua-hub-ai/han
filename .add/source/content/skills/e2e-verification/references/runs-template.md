# `run-<YYYYMMDD>-<n>.md` 模板（单次执行记录 · 临时件）

> 落 `.spec/wip/e2e-<suite>/run-<YYYYMMDD>-<n>.md`，**一次执行（重验）一份文件**，Step 0 判定时新建、Step 5 执行中回填。
> **Case／Run／Issue 分层定位**（对标 TestRail / Xray）：`cases.md` = Case 库（规格 SSOT，只随规格变更而改，diff 即重新评审信号）；本文件 = Run 记录（每次执行的独立快照，结果回写 run 不改写 case），失败项引用 Issue。新增 run 文件**不触发** Step 2 重新评审。
> **材料处置**：本文件与 `evidence/run-<id>/` 依 `skill:knowledge-curate` 的保留准则处理；唯一复现证据、当前验收/Issue/审计仍引用的证据不得仅因“已转 Issue”而删除。实际归档须保留引用、摘要和恢复位置；本模板不另定清理流程或授权。

---

# <suite> 执行记录 run-<YYYYMMDD>-<n>

## 元信息（Step 0 判定落盘）

| 项 | 内容 |
|---|---|
| run-id | run-<YYYYMMDD>-<n>（同日多次执行 n 递增） |
| 日期 | <YYYY-MM-DD> |
| Case／Run／Issue 引用 | <cases.md 的 revision/digest、本 run-id、关联 Issue；无 Issue 时明确无> |
| 精确成果/版本 | <被测 ProjectArtifact 引用及 revision/digest，绑定本次 Case 集和环境> |
| 交接确认 | 本 Run 的固定 `## 确认记录` 与结构化记录共用 decisionId；当前状态和双落点一致性由 `add status` 查询 |
| 触发原因 | <首次全量 / 换包 / 重部署 / 重建环境 / 仅前端刷新 / 仅用例改动——按 Step 0 判定表> |
| 环境动作 | <换包 / 重部署 / 重建 / 刷新前端 / 不动> |
| 环境与被测版本 | <站点（引 `../../e2e/env.md` 行）+ 被测服务 / 输入数据版本（commit / tag / 校验值）> |
| 本 run 范围 | <TIA 选出的 case 集 = 部署类型粗筛 ∪ 覆盖 SR 精筛；范围外 case 不列入下表（不重跑 ≠ 失效，上次 run 结论仍有效）> |
| 前次失败与本次修复 | <前 Run、原 Issue、实际修复修订及关联主交付 ProjectArtifact 引用；首次无则注明> |

## 逐 case 结果表（Step 5 回填）

> 状态 = 通过 / 失败 / BLOCKED / 未执行或 skip；证据按 run 分目录 `evidence/run-<id>/`；真问题必先登记 `.spec/issues/issues.md` 再链号。skip 不等于 PASS。

| CASE-ID | 优先级 | 状态 | 证据 | issue |
|---|---|---|---|---|
| CASE-<id> | <P0/P1/P2> | <通过 / 失败 / BLOCKED / 未执行或 skip（注明原因）> | <evidence/run-<id>/CASE-<id>-*.png / trace / log> | <ISSUE-<日期>-<序号>，如有> |

## 终态小计（Step 6 对账源）

| 优先级 | 本 run 范围 | 通过 | 失败 | BLOCKED | 未执行或 skip |
|---|---|---|---|---|---|
| P0 | | | | | |
| P1 | | | | | |
| P2 | | | | | |

<!-- ADD:CHECKPOINTS -->
## 确认记录

> 本区由 `add checkpoint` 机器写入，请勿手填 decisionId。任一落点失败都不声称完成，按同一真实决定补录。完整语义由 Workflow 的“三处框架级软确认”统一定义。

> 表格与结构化记录来自同一真实决定，逐行列出责任角色、实际确认人、确认内容、结论、时间和备注，并绑定本次 IR 范围及成果版本。未决定保持 pending；跳过、延后和带风险继续按原意记录，不写成已确认。

| 项 | 内容 |
|---|---|
| 确认对象 | <本 run-id + 精确成果/版本 + 环境 + 逐 Case 结果摘要> |
| 当前确认状态 | 以 `add status` 对本 Run 的查询结果为准；未决定不代签或写成已验收 |
| 确认记录入口 | 本区确认行、结构化记录及当前适用性由 `add status` 联接查询 |
