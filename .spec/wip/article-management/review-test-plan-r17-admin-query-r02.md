# Admin2 Test-Plan 独立评审 R02：r17 增量

日期：2026-10-28。原 TestPlan Reviewer Pauli，独立于作者 Ohm。objectType：`design`（测试规划合同）。沿当前上下文复用同版完整有效 Role、`add-code-review` design 分支与适用规则，不重新 Route、不重审 r16 全历史。

**结论：PASS，仅固定 r17 的恢复边界/分项状态与证据引用增量。两轮完成，无新增 findings。** 规划足以作为 Lead 判断局部 Root/PG 窗口的输入；本报告不发 START。Case18 整体及 S8.2 接收/关闭仍 blocked，不能以局部 Go GREEN、skip、删断言或 fake503 替代系统恢复证据。

## 1. 固定对象与完整实际差异

Han/main，实际只读核验 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

| 对象 | 实际 SHA-256 |
|---|---|
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` r17 | `93071AB5E9E68ED976FA376EF52874E7B1A74DEE3FA9860561DD46A596A8DFA2` |
| r16 before-image | `6941C1D7B96F78B341CBB7B88AE74D548CF860CFEAD37D3D92E208548D896E38` |
| design r21 | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA` |
| stories r10 | `492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9` |
| Design R03 | `195334B08EB9F106D1ABA2C1BCC5E0D21767295DD5CE33670BA449C1B2F86BF2` |

实际读取交接目录 `C:\Users\J00010385\AppData\Local\Temp\han-admin2-testplan-r17-60e30efa2c544f91a4b50073bafc889f\` 的 before/fixed 副本、完整 actual diff、handoff/static/evidence-inputs。当前目标与 fixed r17 副本逐字节相同；独立重建 unified diff 与 `r16-to-r17.diff` 逐行相同，该 diff SHA 为 `CC0B4D355454C750D05D11D0F549F257F37915AC275F3AD01E91CC031E595209`。按关闭 autojunk 的确定性对齐独立计得 68 行增加/30 行删除，与作者静态回执一致；结论以字节/hash/完整 diff 为准。

独立复算得 20 个 Case ID 不变、38 组 Case×层映射完全相同（U17/C1/P16/R4）；仅 Case01/15/18 的场景行变化，其余17行原样保留。九路径、四 top 与全部 go 命令模板相同。逐段字节核旧风险/创建读取编辑治理 Case、Admin12/RawQuery 严格示例、删除/无归属/通知/REST/恢复场景、数据环境及旧任务章节、执行顺序、r15以前历史 Gate、r16以前结论后缀均保留。未以作者的 static JSON 单独代替这些检查。

## 2. Round 1：变化到验收/状态的映射

完整阅读本次 metadata、§1 来源、Case01/15/18、Case18 五分项、§6 对应映射、§9.6 责任/命中、§11.1 当前 Gate/执行引用及结论的实际 diff。固定 design r21/r10 与 Design R03 的已审安全边界原样复用，没有重复架构调查。

| 增量 | 核查结论 |
|---|---|
| Case01 账号/全局恢复 | 保留普通 active admin/两级 Principal/401/403 与 GET 安全方法；账号级 recovery_required/每用户 Gate 按身份合同处理，不作为全局恢复503证据；“零文章下游”不误拒认证必需读取 |
| Case15 零写 | 限定新增查询层 Article/User/Audit/Notice/recipient/operation 等业务快照与禁调用；认证链合法锁 User/撤销 Session 另核，未削弱查询零业务写，也未宣称整条认证链绝对零数据库写 |
| Case18 可独立部分 | 真实 factory/共享实例、Trace/exactrouter、Session/AccessGeneral 三项分别保留所需证据；resolver spy 只证接缝，不证 SessionStore SQL、Article SQL 或通用恢复保护 |
| Case18 受阻部分 | 通用恢复请求隔离、完整恢复后生产开流两项均明确 blocked-upstream，保留503 recovery_in_progress/零文章下游/失败继续隔离及 S16/二次核验/MarkConsumed 后开流义务 |
| 整体状态 | 五项仍属同一 `R:ProductionSharedChain`，未增加/减少 Case 或层映射。前三项局部结果不得令 Case18 整体或 S8.2 关闭 PASS；不能 t.Skip 后计成功、删断言或改名绕过 |
| 责任与授权 | 恢复主线/架构 owner 先明确承载/authority/装配，随后测试 owner/Planner/Lead 对齐；不扩大 S10.5 或给本卡九路径新增全局装配。当前七内层 CODE/LOCAL 与 main/PG 未授权分别记录，计划本身不授新权 |

§4.3.1 :307–317 与 §9.6/§11.1 的条件相符：局部测试可以运行并 GREEN，但必须带清晰子集/修订/范围，不能从 Go 测试顶层或总数推导 Case18 业务验收整体通过。对未交付的系统义务标 blocked 与局部子集0fail/skip并不冲突，完整四 top/38 映射的最终门禁仍未满足。

## 3. Round 2：保留合同与历史执行引用核证

### 3.1 数据/资源/负控与实际层映射

本次未改变 r16 已审的三 view/五 cursor/20 有向互拒、真实复合 keyset/过滤跨批/两 cursor 边界、第二批身份故障无半页、合法 pending404、N001 HTTP500/旧409、Admin12 独立值 oracle/同源 ETag、真实 SQL→reader→service→Handler 分层等合同。

完整0001～0009迁移、随机先提交 schema/失败 retain、FK/CHECK、真实 reader 后 fault、真实 rows/事务/context/join/pool 收敛、独立快照与动态 writer delta、无 DSN/PG 信号前不运行、三类可编译行为负控、ordinary/race/integration tag、真实 top/leaf 与零测试防假绿均在原文保留。codec 的已发生 RED 不替代 RawQuery/身份/分页三类尚待执行的有效性证据。

未读取 Dalton 动态七文件或当前 root/测试声明；新 top 的当前实现、Case leaf 完整性及最终命中留待同版代码/执行核证。表内38组仅是计划映射，本轮不把它变成38组实际通过。

### 3.2 codec RED：固定历史行为，非本轮执行

实际只读核 `han-admin2-phase1-379a75dccaad4d34ad14ece04e0a2d4e` 的固定证据：

- `codec-red.meta.json` SHA `0943722DA08508BF50C990735E2F0F17EC4EAD91E799D62050561845BF46EDED`：精确命令 `go test ./internal/article -run '^TestAdminCursor$' -count=1 -json`、exit1、移除 DSN、当次九路径前后摘要相同。
- `codec-red.stdout.jsonl` SHA `214EEE4BBDE545AAA5E8EFBAF42D0338814C8B49E29000F11BF463EF7FB656C9`：实际 JSON 在 `2026-10-28T00:41:28.4101645+08:00` 记录 `TestAdminCursor/FiveViewIsolation/admin_hidden` 的 `valid view admin_hidden rejected: article cursor is invalid` 及 fail；另有父节点/包 fail，不把它们误数为独立缺陷。
- 固定 `policy_test.red.txt` SHA `B67C708416E4CCBD988EBF43BAED45484177B6B6BCD54988E8EDE1461D7BB47E`：只读 :14–35，失败断言实际依赖 `CursorCodec.Encode` 返回错误，非编译失败/缺环境或无条件硬编码失败；stderr 为0字节。

这证明 r17 对该次 C 层行为 RED 的引用有据；不证明 Case03 所有层、其后 GREEN/race 或最终源通过。本 Reviewer 没有执行 Go 或重现该测试。

### 3.3 Lead UC 中间回执：严格限精确时点

实际读取 `evidence-s8.2/uc-intermediate-r01/lead-json-verification.json`，SHA `31FC79A17477FFDB5BB46107E9E0ACE1324DB218B6F379BAF75BEB99D5DC3A1A`。回执时点为 `2026-10-28T00:51:34.845426+08:00`，普通命令精确匹配 U/C 两 top，exit0、122 run/pass、102 leaf、0fail/skip、当次 source meta 前后相等；明确不声称最终修订、完整 Case、race/回归/PG/root 或全局恢复。

r17 :1192 正确保留上述时点、命令和边界，没有将回执扩为最终 Case/代码 Review，也没有从旧结果冻结当前动态源码。本轮只核 Lead 回执及其摘要，未重新审核UC全部 raw JSON/源代码或独立重跑其122节点，因此不另发该次执行 PASS。

## 4. Findings、局部窗口判断与 Gate

**新增 findings：无。** 原 r16 TestPlanReview 仅保留其原有界结论；本轮 PASS 来自对 r17 实际变化的独立两轮检查，不把旧报告或中间测试结果作为本次评审的替代。

**足以供 Lead 判断局部 Root/PG 窗口。** 三项可独立 root 子断言、P层真实数据/资源/故障/负控、相应命令和证据限制已明确；Lead 可在核最终 scope、同版源码/测试、writer/环境及授权后，独立决定对应局部窗口。当前 main/PG 未授权的状态没有被本报告改变；不要求先交付全局 Guard 才能评估不依赖它的局部证据，也不因局部窗口开放允许处理未经完整恢复的真实业务流量。

恢复隔离/完成开流的五分项后两项仍须上游生产合同/装配、独立评审、真实根拒绝/零下游与成功开流证据，Ohm/Planner/Lead 对齐后才能解除阻塞。未达到时 Case18 整体及 S8.2 接收/关闭继续 blocked；原 S1.5 四 contract/16 endpoint/ACL/S1–S16/producer/外部真实层责任不缩。

| Gate | 本轮结论 |
|---|---|
| r17 测试规划增量 | PASS，无新增 findings，仅本次有界对象 |
| 20 Case / 38 map / 九路径 / 四 top / 数据资源负控与命令 | 保留，未等同实际执行覆盖 |
| codec RED / UC 中间结果引用 | 精确历史证据可追溯，不外推最终 Case/race/PG/root |
| Case18 前三项局部 R 证据 / P 运行 | 可供 Lead 分项判断窗口；本报告不授权、不执行 |
| Case18 后两项、Case18 整体、S8.2 接收/关闭 | blocked-upstream，不可 drop/skip/fake PASS |
| 架构归属/系统门禁/人的 accepted/checkpoint | 原责任继续 pending；不扩大 S10.5、不代签 |

## 5. 实际 Readset 与限制

本轮读取：r17 当前/固定副本与 r16 before 全字节/完整 diff；handoff/static/evidence-inputs 完整 JSON；静态检查脚本仅查计数实现的必要行，未执行作者脚本。独立复核 Case定义/映射、九路径与命令、未影响区域/历史后缀字节保留。r21/r10/Design R03 核摘要并复用当前会话已实际取得的有效正文；Harvey、r16 TestPlan R01、Planning R02 核摘要并复用原读取范围。

执行引用只读上述固定 codec meta/raw JSON、0字节 stderr及固定测试前像 :14–35、Lead UC 回执。未读动态 Dalton 源码、当前 root/测试、Task/Plan、身份文档/ADR 或新架构归属成果，没有重新做 Guard 调查。Reviewer 的 Temp `review-r02-readset-hashes.json` 仅记录本轮固定规划/上游/报告/回执，排除动态实现。

未验证：最终 U/C/P/R 实现及普通/race/回归/PG/root 新运行、完整 Case/leaf 实际命中、数据/资源/三负控实际效果、通用恢复隔离/成功开流、架构最终承接、Host/部署/E2E/容量。若后续固定设计或 Case分项/数据/边界变化，按新修订只复审受影响差异。

唯一仓库写入为本报告；原 r16 Review 保留。只读 branch/HEAD、文本/字节/hash/diff/映射和历史 JSON 解析；无测试/PG/DSN/SMTP/服务/Browser、Git 写、下级委派或清理。末次核固定目标与所查固定依据无漂移。

**stopped：Pauli r17 两轮独立 TestPlanReview 完成，交回 Lead；不发 START。**
