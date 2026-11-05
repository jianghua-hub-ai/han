# C2 Task/Plan 映射差量：独立两轮 PlanningReview R01

结论：本固定映射增量 **PASS，新增 findings 0**（CRITICAL/HIGH/MEDIUM/LOW均0）。没有另行待决定项或非阻断修改建议。这是 objectType=design 的任务/计划合同差量 Review，不是代码 Review、设计重审、任务关闭或采用决定。

## 固定对象与方法

- 原 Dirac/add-reviewer 独立执行，非作者 Goodall 自评；Workflow=development-delivery。固定 D:/AID/VSCode/Codex/Han / main / HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca 按用户给定；本轮禁止 Git，未另执行 Git 核 HEAD/branch。保留既有脏树。
- Tasks：`.add/project/knowledge/tasks/tasks.md`，before **EB9B8129587E41640D51EB3D32FB77C56842C4F508617A80F75B6D5E31B11367** → after **EAFCC8E05617F0B0CD027A47548C1E9F995534F8D51982646590C4F79D4ABB3D**。
- Plan：`.add/project/knowledge/plans/ir-01-content-community-batches.md`，before **358BEFD285E2C73E85681BB0905413D1DBE15EEB8EC4A576F0E28A5B8E66A836** → after **E07E98509DC14C7566FC6ACB59A86AC893E1A0FED5842A6F450597F54B26E9AD**。
- 上述两对象 entry/exit hash 必须相同；详见 entry-hashes.json、exit-hashes.json 及 receipt.json。只声明这两对象的评审期稳定，不作全库冻结证明。
- 已实际完整读取本角色 Role 与主 add-code-review；本角色先前全文取得的 checklist/abstraction/design-writing/decisions/安全红线/delivery Workflow 及 task-decompose R/batch-plan、task-execution §10 按当前完全相同 hash 复用，checklist/abstraction本轮另完整重读。不是以父会话或作者读取替代本角色。见 reused-methods.json、readset.jsonl。

## 两轮实际范围

**Round 1：直接任务/计划缺陷。** 完整 tasks.md.diff 49行/4hunks、Plan diff 18行/2hunks；Tasks Article S4.7 的2315–2375及 S4.16 的2601–2659受影响正文/原文件清单/命令/最低层；Plan B13C1511–1522和B13H1562–1568。检查新验收能否落到原owner/文件/原tops，失败与成功边界、范围/状态/启动授权、局部关闭及后继责任是否一致。未将源码实现/符号存在当规划前提。

**Round 2：沿真实依据有界追查。** 当前 identity r54 / Article r26 精确合同（identity3614–3626、Article2365–2407）与原本角色 C2两轮设计报告全文54行、Main c2-design-main-reception-r01.json全文；作者 receipt.md/json、verification/graph/card-protection结构数据及 MainTemp/c2-planning-main-verified-r01.json。没有读当前Runtime/Article源码或追S4.18 claim代码hash；原设计Review已追旧before-image/冻结消费者的结论仅按其原范围复用，不冒本轮重新代码验证。

上游设计 SHA256：identity r54 **E823D25D9CDDFD4F2D7E3A869FBB6B15EAF5275E55147A21546D4C2836472522**；Article r26 **443C55E3E89EDBE4CF0870036BB662ED2E64B1A70F8BC5609F18025DBA71D267**。实际读取并独立核当前字节。原设计报告 **1354859C65D125AC5E6C3EF7576539C0250F2B8E5D677F7049529A6E18D9E4C3** 的 PASS0与Main技术采用是本轮来源，未替代本轮判断，也未等同人的签字。

## 有界判断与映射证据

| 事项 | 独立判断及落点 |
|---|---|
| 私有接口与平台owner | Tasks2362对齐Article2371–2389：嵌入原ExclusiveLease，ANY WaitReleased返回含error都已terminal；attempted/terminal/closed分开，一次physical Close、原owned+error可保留、禁止unknown defer丢handle、raw Close重试/FD复用及假terminal。由Article原8承接，无Runtime新API。 |
| 故障及最终Complete | Tasks2361、2364、2366对齐Article2395–2400：armed错误先尝试原Fault再join，H1/原cause保留；Stop错代际不复权，无barrier reader不造Fault权；cancel/timeout不提前离开，panic终结后按原值重抛。成功全部fallible清算及最后ctx检查在Complete前；接受后仅无失败内存解锁和原记录返回，不补Fault/Close/check。 |
| 同一drain及可验证失败链 | Tasks2363–2364要求真实NewRuntime/RunNormal/WorkLease/Append、原Main FAIL同名叶和heldfile终结前后断言：仍held时原调用/work/action、重复Stop不结束、Close0/Complete0/H1保留/Acquire拒绝；同owner独立driver实际一次终结并join后才返回原error并结束原工作，同一drain按其他cleanup条件Close1。driver不能等Append/Stop/Lifecycle完成或重取journal mutex；Runtime mutex不跨I/O/join，覆盖自等待风险。 |
| 长期handle与root | Tasks2655、2657和Plan1565对齐identity3620/Article2402：Install返回不自动加入普通work，原root/Lifecycle显式持有同handle的Close/join及并发Stop清算；原调用使用者先终结，长期handle清算不能推导为普通work map自动处理；全部使用者/资源终结才归还deployment、pool最后关。双侧资源/cause/panic及重复Stop证据留S4.16原13、原factory生产链U/C/R/P，不推S4.17。 |
| 最低层及可实施文件责任 | Article原8组保留、只强化Complete组并新加3组U/C/race；原P/Finalizer/SQL、PG13 ordinary/race及oldreg39回归不减。root原9组全保留并加LongLivedControlHandleTrueJoin U/C/R/P，原Composition/ProductionChain/ControlHandlers命令保留。风险组是未来要求，非已存在/已运行PASS。 |
| 批次及依赖方向 | Plan仅B13C/H两行插入，消费原Task而不创第二状态。Article原8由原owner Noether、S4.16原13仍未认领；源码/运行窗需Main后续精确信号。S4.18 C1原claim整卡、S49及旧closed S4.7整卡不变；不新增RuntimeAPI/第9源/反向S4.18或S4.13/root关卡。原producer→Article有序接续与root下游方向保留。 |
| 平台未知与接收边界 | Tasks2362/2366/2368、2657保留native terminal无确切来源则retained pending，不让portable fixture冒native/root通过；终结driver若落不进原8或存在等待环，停受影响部分报Main。新验证NOT RUN、C2代码OPEN、draft/人的接纳unknown及TP未改均如实。Noether G0可行与S4.18作者/peer代码进度按用户告知，不作为本Reviewer代码PASS依据。 |

## 独立静态核对

只运行本新OS Temp自有Python静态分析，不运行作者脚本或任何项目验证命令；源码零读/hash/copy。

- 两份 before匹配精确基线，after与live逐字节相同。完整diff的context/deletion逐项匹配，并独立重建完整after：Tasks4hunks/+12/-4、Plan2hunks/+2/-0。
- 74/51completed/6active/17todo；所有74标题/状态、原文件清单、owner字段、requires及命令块相同；其余72整卡和两卡外全文逐字节相同。S4.18含claim整卡SHA256 **2D3D30B3B3F80D229C6D689166BC0C48CB0041833A068E1D352A19B7993361E5** 与before一致。该hash只保护任务卡，不是producer源码冻结hash。
- 独立从卡片requires重算全图74nodes/319edges，edge delta0，无未知/重复/环；与作者graph的requires映射相同。Runtime局部15卡65incomingrefs是局部计数。九B13批次标题与完整Mermaid字节一致，复用原不变批次图语义审查。
- 原风险组/层次独立逐行保护：Article8→11（原层不减）；S4.16 9→10（原组全同）；原文件预算/原验收命令未改。Task UTF8 BOM/LF、Plan UTF8 BOM/CRLF保留；历史form-feed原字节保留，不顺手清理。
- 作者/Main静态断言均作对照，未代替上述独立计算。结构JSON按完整机器解析消费，不声称手工逐条重审其所有无关卡语义。

## 限制与交回

本轮全部新增实现验证 **NOT RUN**：Go/build/test/race/vet、PG/SQL/服务、native终结、真实root/长期handle生产清算、provider/Host/E2E；未签代码、task close、G4-batch或TP/TE批准。旧R02 F001 CLOSED、EF531/358BE Planning PASS0+L-C2-OWNERSHIP-01保持原历史，不因本映射PASS被删除或变成C2实现关闭。当前两SSOT的后续技术接收仍交Main；人的精确接纳unknown/draft不变。

工具限制：初始若干展示有截断，完整两diff已分段补读；大结构JSON由独立脚本完整解析，非全文语义证明。自有静态脚本初次把历史form-feed当换行导致byte reconstruction断言失败，已改为保留原line-ending chunk后精确重建成功；这是Reviewer分析器修正，不是产品测试失败或对象改动。没有清理这些自有诊断脚本。

**STOP，交Main接收。** 仓内零写；仅本新OS Temp报告/readset/receipt/hash/静态分析。无Git、Go、PG、网络、secretENV、cleanup/删除/移动或子委派，未替作者修复或替人采用。
