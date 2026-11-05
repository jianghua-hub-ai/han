# 恢复运行时详细设计：Lead 接收与后继交接

2026-10-29；固定工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。依据用户本轮“对待确认或决策项采用建议值并继续本项目”的授权推进技术建议；不把该授权写成人的逐份成果签字或真实环境事实。

本轮完成恢复运行时详细设计及独立评审闭环：identity design r50 / Story r17，article design r23 / Story r11；ADR-0027 r02 原两条设计采用记录及索引已同步。仍复用两份既有 feature design，不增设第三份设计或任务状态源。四份设计/Story 保持 `draft`，人的精确成果接纳仍 unknown，未生成 accepted checkpoint 或标为 implementable。

## 真实产出与独立评审

| 工作 | 实际 Role、Agent 与结果 |
|---|---|
| 四份设计/Story 及两项修复 | Banach / add-feature-designer / `01a18389-57ab-7b70-8c86-82a61e7729f1`，最终 STOP/frozen |
| 覆盖输入咨询 | Jason / add-test-engineer / `01a18395-a83b-7512-b7d1-824b073ab907`，16 组输入；非 test-plan 更新或测试执行 |
| ADR 采用元数据 | Pascal / add-arch-designer / `01a17e3d-5550-7f91-bb7e-6e6fad3d082c`，实际引用仍为 architecture 1＋design 2＝3，Story 不另计 |
| 非作者独立评审 | Kuhn / add-reviewer / `01a1835a-8010-7a73-a8dd-3192719b6965`，R01 两 HIGH，原作者修复后 R02 设计层关闭，R03 最终元数据核验通过 |

Lead 已全文核对四份原始增量、两项修复增量、最终四个元数据插入块、两次采用登记精确差异及 Review 报告/receipt，并在真实工作树现场复核。该接收是 Main 的技术接收；各原报告当时的 pending、OPEN 和失败证据原样保留。上述实际 Agent 产出不代人的接纳，也不将作者说明写成独立评审。

- [R01 报告](evidence-design-r01/review-r01/review-report.md)：CHANGES REQUIRED，`RECOVERY-DES-R01-F001/F002` 为 HIGH。
- [R02 报告](evidence-design-r01/review-r02/review-report.md)：两项均在**设计层 CLOSED**，新增 finding 0。F001 补齐恢复 maintenance session、初始管理员改密的同事务状态迁移与切流撤销；F002 补齐恢复专用重置资格、可信审批绑定、持久化发送权及原子结算，保持日常流程的拒绝语义与 unknown 保护窗口。
- [R03 报告](evidence-design-r01/review-r03/review-report.md)：六个最终对象的**纯元数据检查 PASS**，新增 finding 0；不重审技术，不代 Code、Case 或测试 PASS。

## 最终六对象

| 对象 | SHA256 |
|---|---|
| identity-auth/design.md r50 | `6634BED32EDE264F6DA6F47C9A7E8152ADD0B19C2DDCBCF3ED4708AE4BA4109D` |
| article-management/design.md r23 | `2223BE4D005753FBF8D8AE0116F6926A7E8C044482BCBB0FADAB811677066350` |
| identity-auth/stories.md r17 | `8B5EBCD5A78A20769FAF3D31492562C271F0DFB3B770E38383B87DF250559616` |
| article-management/stories.md r11 | `F2AB369EE2259F043ADD608A013660904B201AB5F4066D76747A81348D65998C` |
| decisions/framework.md ADR-0027 r02 | `36F4B49FD5C9B6FC0B351BAF326E03CF74889A5FA91CD052516D4B2542708B79` |
| decisions/README.md | `157A549AD375B650479900A350464FCB842E828EFD296F3A50FA406A996FBAA1` |

四份最终元数据各仅一个插入块，逐字节移除后即还原 R02 技术受审版本。framework 最后仅原两条采用行替换，README 相对首次采用登记零差异；ADR 技术正文及旧架构评审记录保持。

## Lead 现场证据与适用限制

- [机器接收记录与校验索引](evidence-design-r01/lead-reception/index.json) 包含基线、检查脚本、最终 status、hash、文档静态检查及本记录。190 个源码/构建依赖文件的清单与字节未变，33 个业务知识文件中仅获准的 6 个发生本轮变化，其余 27 个保持；控制文件、分支、HEAD 与空暂存区保持。Git status 中另有既存改动和未跟踪文件，不能把它们算作本轮产出。
- 四份设计/Story 的版本、draft、原有确认记录与 BOM/原 CRLF 保持；新相对文件链接有效。PlantUML 仅检查成对界定符；本机缺 PlantUML CLI/jar，**语法执行和渲染 NOT RUN**。
- 本轮未改源代码、Task/Plan、两份 test-plan、边界或 architecture r32；未运行 Go/build/vet/race、PG/DDL、HTTP/SMTP、worker、服务或真实恢复；未读取凭据/DSN/ENV/密钥，未网络安装、Git 写入、清理删除或归档移动。设计中的 0010/0011 正向迁移仅是待实施方案。
- `Case18`、文章 REST/内部契约、旧备份恢复与管理员文章入口整体验收仍未完成；当前启动拒绝不等于真实两个管理员 GET 返回 503 且 Article 下游调用为零。真实 CAS/provider、可信审批/安装输入、目标隔离、ACL、备份介质、issuer/retirement、RPO/RTO 与 Host E2E 仍须后续证据，不从文档补齐。
- ADR-0024 的真卷韧性范围决定不扩至 S4.4；S3.3 仅免真实浏览器连接、S3.5 仅免真实平台/工具验证，均非 PASS，S3.6 不变。真实 S4.5 CAS/S4.6 retirement 门槛保留，不重开已关闭 S4.2/S10.5。

现役 Task SSOT 仍为 **59 / 48 completed / 3 active / 8 todo**；active 精确为 `article-management-S1.5`、`article-management-S4.4`、`article-management-S8.2`。这是对现役 SSOT 的现场引用，本记录不另立任务台账。本轮设计完成不关闭上述任何任务。

## 下一环节的具体输入和停止条件

1. **Test-Engineer 更新原两份 test-plan**：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md`（当前 r23，SHA `4CDC882308A2CCF6EE249880EBA5EEA469C2C983933C4EE6D42AE1168F5F03D2`）与 article 同目录 `test-plan.md`（当前 r20，SHA `DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA`）。精确消费本记录六对象、architecture r32、现役边界、[16 组覆盖输入](evidence-design-r01/test-input/coverage-input.md) 及 R01→R02 失败/修复链。输出可执行场景、环境/数据/真实 provider 前提、分层断言和证据归属，特别覆盖维护会话切流、恢复专用密码重置、双 namespace/ABA、历史 run、ACK loss、published closure、三 worker 和真实入口隔离。规划结果与 NOT RUN 分开，不自行执行或关闭 Case/任务；新的文件写集须由 Lead 精确登记后开始。
2. **非作者测试计划评审后，Task-Planner 做 R 对账再组织批次**：只使用现有 `tasks/tasks.md` 与 `plans/ir-01-content-community-batches.md`，先对照真实代码/既有任务作四标签 gap-map，再决定复用或新建；不得预先替 Planner 编任务号或把设计变更映射成旧任务已完成。显式核对 recovery neutral 核心、identity/session/reset/Store、article CAS/Journal/publication、root restore/verify 驱动、0010/0011 迁移、共享启动入口及其测试的串行所有权与可独立验收边界。
3. **实施另行完成 G0、独占文件边界和验证安排**：真实 provider/审批输入缺失时应如实阻塞相应真实链；不得以空库、普通 flag、当前进程布尔、fallback fake 或启动拒绝代替恢复准入/开流证明。用户已授权采用建议继续，下一环节无需重复询问常规建议值；不得将授权用于捏造环境事实、秘密、人的成果接纳或未执行的验证结果。

本轮只完成上述设计交接，没有宣称后继 test-plan 已更新、Planner 已派发或实施已开始。原始与修订证据均保留，不清理唯一证据。
