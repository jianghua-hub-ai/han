# 普通管理员文章查询：Lead 设计交接核证

2026-10-27，固定工作区 `D:\AID\VSCode\Codex\Han`，`main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本记录保存本次核证，不替代现役设计、测试计划或 Tasks。

Feature-Designer Hubble `01a17daa-c655-73c2-a373-335fea4d3b54` 在原设计和 Story 两文件内完成 r19/r08 及有界返工 r20/r09。原非作者 Pauli `01a17dda-43ab-7040-b02b-db6a5e67ac9b` 的 [R01](review-design-r19-admin-query-r01.md) 为 CHANGES REQUIRED；[R02](review-design-r20-admin-query-r02.md) 确认 `AM-DES-R19-ADMIN2-R01-F001` 在设计层 CLOSED，无新增 finding，Design Review PASS。R02 SHA256 为 `5B2E39D0A491B173BB651AA20E966BE40A0E03DFC1645E8D3CFFB0594B9E62B0`。两角色均已交回停写。

Lead 已阅读实际 r18→r19、r19→r20 及对应 Story 完整变更、独立报告和测试准备，现场核固定哈希与五项非 root 直接依赖。核证输入及结果见 [交接回执](evidence-admin-query-design/lead-design-handoff.json)。

| 固定成果 | SHA256 |
|---|---|
| [design r20 快照](evidence-admin-query-design/design-r20.md.txt) | `6884431B6164140E5593A8253AC20E8EB1C59C1AD71B86558B6D1A8C99BF037A` |
| [stories r09 快照](evidence-admin-query-design/stories-r09.md.txt) | `A2BA7746F30B1C9B60DB4337995FC0A0E52DD55BFF984BAA784C20331D80C569` |

本增量补齐原接口总表内普通管理员列表和详情两个 GET：复用同一查询服务、候选 SQL、身份 reader、QueryConfig 与现有 12 字段投影；固定 all/violating/hidden、独立 hidden cursor、严格身份失败、pending 详情优先 404 和同源 ETag。新列表必须完整解析 RawQuery，任意解析错误先整体 422，不能使用部分 map 或回退缺省；解析成功后才采用 view/limit 422、cursor 400 的字段规则。N001 保留新读取 HTTP500 与旧治理409的实施注意。

采用依据是用户持续授权对待确认/决策采用建议并继续，以及 Lead 的实际设计/返工指令。专业成果已核后继续测试规划；实际人的成果签收、责任确认和 checkpoint 仍 pending，未代填 accepted、人员或时间。

[十份逐字节副本目录](evidence-admin-query-design/lead-copied-inputs.json)保留 r19/r20 设计、相关 Story、before-image 清单及 Reviewer 标准库解析探针。原 R01 失败链和 Temp 原件保留；探针以 `.go.txt` 保存，不加入 Go 构建或格式化范围。

后续覆盖与实施责任分别进入现役 test-plan 和 Tasks；本记录不另建任务状态。原 S1.5 的四文件、16 endpoint 和完整验证要求保留。S1.6 的三个现役 Handler 接线由 Dalton 独占两个 root 文件，与未来 Admin2 接线串行；其进行中的源码不属于本次设计冻结对象。

后续测试规划交接已核：Ohm `01a17c83-60f4-7803-b0cc-9b19c0b81fa3` 的 r16 SHA256 `6941C1D7B96F78B341CBB7B88AE74D548CF860CFEAD37D3D92E208548D896E38`，Pauli [独立 TestPlanReview](review-test-plan-r16-admin-query-r01.md) SHA256 `267CAA23481E31C4ECF3654FAD851772747B5ED6A04DB50823F5E973A978CC1B` 两轮 PASS、无 finding。Lead 已读完整增量/报告并核哈希，[交接回执](evidence-admin-query-design/testplan-r16/lead-testplan-handoff.json)保留具体范围；20 Case 的 U/C/P/R 共38组映射全部 NOT RUN。旧 harness 的 schema/迁移同事务不能直接充当预提交与失败保留证据，新 P 文件必须落实明确的创建/提交/早期记录边界。

S1.6 随后已由独立 CodeReview 和 Lead 核证关闭，原两文件窗口正式释放，详见[本卡验证记录](verification-s1.6.md)。Admin2 新任务与只读 G0 接续仍按 Tasks/真实授权，不因设计或测试规划 PASS 自动启动代码或数据库执行。

本次 PASS 仅为固定设计增量与独立复审。Admin2 服务、HTTP、真实 PG、生产路由、部署/E2E、图渲染与性能均未在本次核证中执行，不用原 S2.2 或身份模块豁免代替。

2026-10-28 G0 补充：Dalton 已交回只读准备；Lead 现场核九路径的七个现存前像一致、两个新增路径仍不存在，148 个范围外源码摘要无漂移，分支/HEAD 不变、暂存区为空。该核证见 Temp `han-admin2-impl-e5199290ceb54e33a7c310eb02ab8e21/lead-g0-followup-r01.json`，不是代码或测试运行授权。

[Harvey 调查原文](research-admin2-guard-r01.md)（SHA256 `213653A8A07EEDFFC65F07B834D5F78E6E66EB27C8FCE7C5EA2D2CB774FC2E4A`）确认当前生产 HTTP 链没有接入独立通用 readiness/recovery 请求 Guard；启动/lifecycle、每用户 password-reset Gate、Login mode 和恢复删除 OpenGate 不能替代它。本文保留上面的固定历史设计/测试规划 PASS，不据此声称当前 Case18 全链可验。Lead 已现场读取 root 包装、router/server 入口和 OpenGate 接口语义，核对上述区别；未运行恢复场景。

[独立 Planning R01](review-tasks-plan-s8.2-r01.md) 为 CHANGES REQUIRED，唯一 `AM-S82-PLAN-R01-F001` 是 S8.2/B9F 未专门承接此 G0 缺口。原 Ptolemy 已修订 Tasks/Plan，Lead 已读实际 +5/-1、+3/-1 差异及保留回执，现交原 Pauli 复审。该次固定摘要分别为 `BD544FAAB0996A6AB277DFCD62273CEFC67082A9AE8799838718E588538A43F9`、`3EAE1C19B7BA5625DC5EBB3705CBF66F73B0E75356BB7169B0A47E04FC685DA8`；后续状态仍取现役 Tasks。

原设计者 Hubble 已接收调查及原 r20/r16 冲突，核对精确验收含义和上游责任。ADR-0018/0020 的恢复完成前不开流义务保留。受影响 root 恢复承接、Case18 的 readiness/recovery 子断言及整卡关闭暂停；内层代码、本地测试与真实 PG 均须 Lead 分别明确放行，此时尚未 START。不在 Admin2 九文件内另造全局 authority，不删除断言求绿，不将人的签收改为 accepted。
