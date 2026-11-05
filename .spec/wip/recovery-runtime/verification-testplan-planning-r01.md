# 恢复运行时：测试规划与任务批次交接

2026-10-29；固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。用户授权采用建议值继续；本轮由 Main 依该授权委派专业 Role，未据此推定人的设计签字、checkpoint、环境事实或发布许可。

**当前交接状态：测试与任务批次规划收口，PlanningReview R02 PASS；Main 已现场技术接收相同六稿。** 两次独立评审各发现一项 MEDIUM，分别由原作者修复、同 Reviewer 定点关闭，新增 finding 均为0。本记录只描述本轮成果和接手位置，不替代 Task SSOT、专业 Review 或 Case/Run；没有实施、真实恢复或 E2E 通过结论。

## 本轮成果与真实归属

| 成果 | 实际 Role / Agent | 结果与边界 |
|---|---|---|
| 两份原测试计划 | Jason / add-test-engineer / `01a18395-a83b-7512-b7d1-824b073ab907` | identity r24、article r21；恢复场景和最低验证层补齐，执行 NOT RUN |
| TP 非作者评审 | Kuhn / add-reviewer / `01a1835a-8010-7a73-a8dd-3192719b6965` | R01 唯一 MEDIUM，经原作者定点修复后 R02 CLOSED、新增 0；仅规划判据 |
| R 对账、正式任务与完整批次、TP 映射回填 | Goodall / add-task-planner / `01a18414-b2ab-7392-9b04-fd974203bf5e` | 13 开发＋1 软件专项；B13A～B13I 九批；执行者未派发、未认领 |
| ADR-0027 实际采用记录 | Pascal / add-arch-designer / `01a17e3d-5550-7f91-bb7e-6e6fad3d082c` | 仅两文件元数据，引用 3→6；技术正文不变 |
| 最终六对象独立规划评审 | Kuhn，同上，非四稿及元数据作者 | R01 唯一 MEDIUM；原作者五行修复后 R02 CLOSED、新增0，六稿有界文档合同 PASS；Main 另作现场接收 |

所有 ID 均来自 Main 的实际 Host spawn/resume。部分 Agent 原回执中的 ID=null 或“用户指定”是其接收 Main 上下文时的原始表述；原文留存，实际派发人为 Main，不作用户亲指定 Role 或人的成果接纳依据。文件所有权是协作约定，Main 用实际 diff、哈希和清单复核范围，不把文字约定描述为操作系统硬隔离。

## 失败、修复与证据

- [TP R01](evidence-testplan-r01/review-r01/review-report.md)：`RECOVERY-TP-R01-F001` MEDIUM，文章计划把历史 closure 核验写成“所有外部调用为零”，禁止了合同要求的 Journal/Anchor 纯读取，结论 CHANGES REQUIRED。
- [原作者修复](evidence-testplan-r01/author-r02/article-test-plan.diff)：只改 article 原第786行的证据判据。禁止的 Refresh/ReplayAll/Consume/Fail/Publish/Anchor CAS-write/repair 分别为零；DB 精确 lookup、Journal snapshot、Anchor/source 与前后 revision 正向记账，读取间漂移仍关闭。
- [TP R02](evidence-testplan-r01/review-r02/review-report.md)：原 finding CLOSED，新增 0。受审技术稿为 identity `E6906617E51CC349B783896A3A1D828DA4AA8A03716D0C104E77E9AB40680CF6` 与 article `E39C980C041CAC0C0A5441A9DDC52E51024C75E1911A7EE22B5332EDE66B9EDE`，原 R01/修复链不覆盖。
- [正式规划原作者包](evidence-testplan-r01/planning-author-r01/index.json) 保存阶段 A 先登记 Tasks、阶段 B 消费原卡安排批次、最终四稿原像与差异；[采用登记包](evidence-testplan-r01/adoption-final-r01/index.json) 保存末尾三条采用记录及 README 单元格差异。辅助静态脚本的修正/失败说明保留，均未冒称产品或 Go 测试失败。
- [PlanningReview R01](evidence-testplan-r01/planning-review-r01/review-report.md)：`RECOVERY-PLAN-R01-F001` MEDIUM。article S4.8 本卡的必要 PG 验收要求后继 identity S4.13 才实现的同 Store 补绑定，形成实际关闭反向等待。Main 已采纳修复：article 保留可信导入与 RecoveryRun 零写，真实绑定验收归既有 identity 安装任务；不改技术合同、不加反向依赖或新任务。两 TP 回填和两采用元数据在 R01 固定对象上的有界核验通过，联合交付尚未通过。
- [原 Task-Planner 五行修复](evidence-testplan-r01/planning-author-r02/tasks.diff)保留文章本卡真实 PG 导入、来源/immutable 约束、原字段与精确 binding 输出；既有安装卡承担真实同 Store 一次 NULL 绑定、非 NULL 精确幂等/错绑拒绝、revision递增与原计数/结果/证据/时间/审计保持。Plan、两TP、文件清单、requires与命令均未变。[原采用行刷新](evidence-testplan-r01/adoption-final-r02/framework.md.diff)只更新最终 Tasks SHA及修复来源，README零diff，计数仍6。
- [PlanningReview R02](evidence-testplan-r01/planning-review-r02/review-report.md)：原 `RECOVERY-PLAN-R01-F001` CLOSED，新增0、OPEN/受限0，六稿联合规划在本轮有界文档合同内 PASS。Reviewer 复核精确修复及相关新风险、同源采用行，不重新全量评审未变部分，不读取源码或运行 Go/PG。Main 全文读取报告/回执并核对现场六SHA、diff和保护集后作技术接收；不是人的设计签字或环境验收。

两份 TP 的后续映射/元数据各只有两个插入块；移除后逐字节还原上述 R02 受审技术稿，原场景、层次、历史与已修复判据不变。后续最终六对象评审单独核对，不借 TP R02 预签新任务、批次或采用登记。

## 最终六个冻结对象（PlanningReview R02 与 Main 接收一致）

| 对象 | SHA256 |
|---|---|
| Tasks | `D7059350422894480AE2B0E10F310E973B3CAAC27FA6B49B2AE5DFEB5A7C5993` |
| Plan | `1B2540B5F59756FF3D4B397B90FE44E6111700DCA8F8538A75BE863B99CCF9C7` |
| identity TP r24，映射补充 | `468864D2FBA7281CABDCA0C771D56E1316BAFAE8B0F9B524269435E544BDF41D` |
| article TP r21，映射补充 | `E2E7ADFD0581BCCE9CD2782F7181C12BA67A8496A1A321EF88006EA94D9FCA27` |
| framework ADR-0027 r02 | `681D89C65C1384280C386F4B59BE8C8E3308CD3C10516E23D561D5A23E1CCFDD` |
| decisions README | `7112FA46EF75F8130AA900CF46D5532521D9E5002160FB8BBBDB3031FFF6AFB4` |

四规划稿实际消费的 framework `36F4…` / README `157A…` 是当时的技术输入，已保存 immutable 副本。新增采用账的哈希是后续登记结果，不反向改写设计、TP 或规划的来源。四稿中的当时 pending、引用数 3 和后继登记说明按其交付时点保留；当前登记结果只由 ADR 原引用记录及派生索引持有。采用行的“R02待核”同样是冻结时点描述；后续结论由本轮 R02 报告与 Main 接收记录承载，不再循环改源哈希。

## 任务、批次与现场核对

现役 [Tasks](../../../.add/project/knowledge/tasks/tasks.md#recovery-runtime-planning-r01) 为 **73 / 48 completed / 3 active / 22 todo**。原 59 卡状态、依赖和历史完成证据保持；active 仍为 article S1.5、S4.4、S8.2。本记录仅引用现役 SSOT，不另立状态账。

新增 identity S4.7～S4.17、article S4.7/S4.8/S8.3 全部为待办，唯一安排到原 [Plan 的 B13A～B13I](../../../.add/project/knowledge/plans/ir-01-content-community-batches.md#recovery-runtime-batches-r01)，未安排新软件任务为 0。既有 8 todo/3 active 沿原有效批次引用，不重新认领或分配。

Main 已全文核对完整相关增量、两阶段原像/receipt，并在真实工作树核实：新增依赖无环，62 条图边逐条对应 requires，同批写集无重叠，两处共享文件有传递依赖串行，146 处新增或替换链接与显式锚有效；所有拟新增源路径尚不存在、拟修改源路径存在。原 Tasks 除获准统计/附录、原 Plan 除顶部说明/新增批次外字节保留，两 TP 原技术稿可逆还原。Meta 仅追加三条来源，README 只改 ADR-0027 引用次数；原 framework 全字节和三个采用源保持。

返工后 Main 另证 Tasks 仅两卡五行变化、其他三规划稿字节不变，重核结构结果保持；metadata 同一联合采用行仅+1/-1，原 framework 技术前缀不变，新增或替换 metadata 的17处链接/显式锚有效。R02 后现场六SHA与受审对象一致，377基线文件中371个保护文件字节不变，源码/依赖及知识目录清单无新文件，分支/HEAD/空暂存区保持。辅助比较脚本首次误设3组替换、实际4组共5行，修正后静态证明通过；该失败未写共享文件，不是产品测试。最终 [Main 接收及证据索引](evidence-testplan-r01/lead-reception/index.json) 分开记录 Agent 回报、独立Review与现场核验。

独立目标回读的 PG 载体归 S4.14 本卡，完整 production factory/Session/SQL 软件链的 P/R 载体归 S4.16 本卡；S4.17 另验跨层组合。开发卡自带必要局部验证和评审，后继专项/E2E 不反向成为其关闭前置；必要 SQL 原子性或完整 root 正向也不能全部推给后继。

## 未执行范围与下一接手位置

本轮只改六份业务 Markdown 及本 scope 的存证/Lead 记录。源码、迁移、设计/Story、architecture、边界、控制文件和旧证据保持；分支、HEAD、空暂存区保持，无 Git 写入。所有 Go/build/vet/race、PG/DDL、HTTP/SMTP、服务/worker、真实恢复、部署和 E2E **NOT RUN**；未读取秘密 ENV/DSN/key，未网络安装、清理删除或归档移动。

所有未来 top、risk leaf 和验收命令是已登记的待实施方案，不声称符号已存在或运行已命中。0010/0011 仅为设计与任务中计划的迁移，没有创建或执行。真实 CAS/provider、审批/安装/旧发布来源、锁后端、目标隔离、工具/native receipt、ACL、备份/inventory、issuer/retirement、RPO/RTO、运行责任及人的接纳等仍分别 unknown/blocked。

Case18、原两 S4.4、S1.5、S8.2 整体未闭；两个真实管理员 GET503/零 Article 下游及完整恢复后开流仍须后继实际证据。S4.5 真实 CAS/S4.6 retirement 门槛不降，已闭 S4.2/S10.5 不重开。ADR-0024 只限 S4.3 指定真卷范围；S3.3 只免真实浏览器连接、S3.5 只免真实平台/工具验证，均非 PASS，S3.6 不变。

PlanningReview/Main 已同版接收，下一开发入口为 **B13A / identity-auth-S4.7**。固定写集为 `internal/recovery/runtime.go`、`admission.go`、`runtime_test.go`、`admission_test.go`（新增）及 `journal.go`（仅中性投影/port 修改）。后继 Backend-Dev 按原卡完成 G0、消费者/文件充分性核对、先失败证据和精确运行窗口，再实施；journal 后续 ctx/reader 改造顺序交 S4.8。本轮尚未派发该开发者、启动实施或认领新卡。
