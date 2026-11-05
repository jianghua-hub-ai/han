# S8.2 管理员文章读取：Lead 取证记录

固定工作区 `D:\AID\VSCode\Codex\Han`，`main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本记录只保管本次授权和核证，任务状态及关闭条件以现役 [Tasks](../../../.add/project/knowledge/tasks/tasks.md) 为准；不是另一份任务台账。

**最新核证结论（2026-10-28）：九文件的管理员读取实现、局部与真实 PG 验证、独立代码复核及项目七项检查已完成；S8.2 保持 `[-]`，整卡接收/关闭仍受上游恢复门禁阻塞。** 两个 exact GET 已接入唯一 root，同一查询服务支持 all/violating/hidden、严格身份检查、独立 hidden 游标、12 字段及同源 ETag。Lead 已实读实际代码差异、完整独立报告并核原始日志/摘要/消费者范围；后文早期 pending 均为当时记录。

| 实际验证层（普通、race 各一次） | 每次 run/pass 节点 | 每次 leaf | fail/skip |
|---|---:|---:|---|
| U：服务及 Handler | 108 | 94 | 0 / 0 |
| C：游标 | 27 | 20 | 0 / 0 |
| R：production factory/接缝/路由局部 | 93 | 76 | 0 / 0 |
| P：真实 PG→reader→service→Handler | 99 | 83 | 0 / 0 |

四 top 合计每次327节点/273 leaf；父子节点不作独立场景，普通与race集合不相加。三类负控都在编译后产生实际行为 FAIL，再由当前源码 GREEN；不是构建/环境错误。原七文件在 Root 阶段未变。当前174项输入的文件集合、字节及最后七项检查均核一致，见[逐 Case/层原始映射](evidence-s8.2/final-r01/lead/lead-local-validation-r01.json)。名义38层映射不等于38项业务验收全部满足，Case18的R仍是局部证据。

[联合独立 CodeReview R02](review-code-s8.2-joint-r02.md) SHA256 `F1EE8BF2605C3755835E2A1C0F4DFDA354697BD4EEBE52FE8B8A80E0681B7808`：有界 PASS、无 material findings。Reviewer 独立 R 普通/race各93/76，另独立六组 PG 各34节点/28 leaf、0fail/skip；Lead 已实读报告并独立核31份 Reviewer 原始文件摘要和实际集合。作者58、Reviewer34，共92个本次运行记录的schema均保留；这不是整库schema普查。资源结论依据源绑定断言及逐schema创建/迁移/取消/join/关闭日志，未打印的精确counter数值不补造。详见[Lead联合复核](evidence-s8.2/final-r01/lead/lead-joint-review-verification-r01.json)。

Lead 亲自依序执行 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`，七项 exit0、格式输出为空、14份原始输出摘要已核，174输入前后不变。[Manifest回执](evidence-s8.2/final-r01/lead/lead-manifest-final-r01-results.json) SHA256 `D333D9E1146C84E157774FD2C32E058E6B5B27C25066A53B51AA83CBA09CF62C`。此结果限实际七命令；未安装的 goimports/golangci-lint及完整外部平台、ACL、producer、部署/E2E不由此取得PASS。

**保留阻塞与写窗口交回：** Case18尚缺“全局恢复请求隔离”和“完整恢复完成后生产开流”的实际生产承载/证据；完整恢复/S16/二次核验/MarkConsumed后才开流、失败保持隔离的义务不变。Case18整体、S8.2整卡接收/关闭继续BLOCKED，不skip、不豁免、不把局部Go GREEN当整体通过。[恢复主线交接](architecture-admin2-recovery-handoff-r01.md)已形成，实际上游任务/owner仍待落实；不擅扩S10.5或新增requires。原 S1.5/16 endpoint/ACL/S1–S16/producer及真实E2E责任不变。

Dalton及Erdos均实际completed/stopped，本次九文件写窗口交回、无活动写者，不以任务`[-]`充当文件锁。后续上游任务需在协调后的精确窗口保留Admin2接线，修改相关源码/契约后仅复核受影响证据，不沿用本修订PASS冒充新版本。未stage/commit/push、部署或清理；人的accepted/checkpoint不代签。最终原件及[289份逐字节副本](evidence-s8.2/final-r01/lead-copied-evidence.json)保留。

最终回填已通过独占文件窗口落到 Tasks 的 S8.2 与 Plan 的 B9F，状态仍`[-]`；[Lead交接回执](evidence-s8.2/final-handoff-r01/lead-final-handoff.json)确认其他58卡（含35张身份卡）逐字节保留、统计59/44/3/12不变、Task BOM/LF与Plan CRLF保留，174源码集合及字节无增删漂移、main/HEAD不变、暂存区为空。回填前被编码检查拦住的草稿仅存在Temp，未应用到项目；正确UTF-8候选及实际前后像见[回填副本清单](evidence-s8.2/final-handoff-r01/lead-copied-handoff.json)。本次没有整卡接收、任务关闭或业务签字。

2026-10-28，Ptolemy 登记 S8.2/B9F 后，Dalton 完成九路径只读 G0。Pauli [Planning R01](review-tasks-plan-s8.2-r01.md) 的唯一 MEDIUM `AM-S82-PLAN-R01-F001` 要求明确承接全局恢复 Guard 缺口；原 Planner 修复后，[R02](review-tasks-plan-s8.2-r02.md) 确认规划 finding CLOSED / metadata PASS，无新增 finding。Lead 已读取两份完整报告、实际修订差异及保留回执，并核统计仍为 59/44/3/12、变更仅在 S8.2/B9F。此 PASS 不表示 Guard 已交付。

Lead 在 `2026-10-28T00:40:17+08:00` 依据用户采用建议继续的授权，向实际 Backend-Dev Dalton `01a17dcf-f2ed-7dc2-b7f3-6c72cf771448` 发出七文件 CODE/LOCAL START：`internal/article/{query_service,handler,model,policy,admin_query_test,policy_test}.go` 和 `test/integration/article/admin_query_test.go`。仅允许对应 U/C 实施与定向普通/race、相关旧文章回归及 P 测试编译；子进程移除 PostgreSQL DSN。`cmd/han-server/main.go/main_test.go` 及数据库运行未放行。即时核九路径前像、148 个范围外源码无漂移，见[前置材料及授权回执](evidence-s8.2/pre-start-r01/lead-copied-inputs.json)。实际 Agent 返回和 Lead 最终复核将分别记录，不提前写实现完成。

首个真实行为 RED 已由 Lead 核验：[原始日志、meta 与测试前像](evidence-s8.2/codec-red-r01/lead-red-verification.json)证明测试成功编译运行，合法 `admin_hidden` 被旧 codec 拒绝；23 个测试节点运行、20 个 PASS、3 个 FAIL（含父节点）、0 SKIP，命令退出 1。测试前后生产源码与基线一致。此证据只覆盖游标缺失行为，不代替其他合同的测试或后续 GREEN。

[Harvey 当前接线调查](research-admin2-guard-r01.md)已区分实际 Trace/Session/AccessGeneral 链与缺失的全局恢复隔离/生产开流承载。原设计者 Hubble 交回 design r21 / stories r10，明确七文件核心合同不变，ADR-0018/0020 不开流义务、Case18 恢复子断言及整卡关闭阻塞均保留；[固定副本与实际差异](evidence-admin-query-design/guard-r21/lead-input-verification.json)已由 Lead 逐字节核证，独立设计评审另行进行。Ohm 对齐测试规划，Pascal 只读核架构责任归属；两者不能签发运行或关闭结果。

截至上述取证点，开发仍在进行，P/root/系统恢复验证没有本轮 PASS，S8.2 不关闭。没有 stage/commit/push、部署、清理或人的 accepted/checkpoint；身份模块由另一会话负责，其内容保留。

后续上游交接核证：Pauli [Design R03](review-design-r21-admin-query-r03.md) SHA256 `195334B08EB9F106D1ABA2C1BCC5E0D21767295DD5CE33670BA449C1B2F86BF2`、[TestPlan R02](review-test-plan-r17-admin-query-r02.md) SHA256 `600C9BC2AAECF668D50B9A96090250021B57FDF44AF9965FD6FF2440AB550A3D` 分别对固定 r21/r10、r17 有界 PASS，无新增 findings。Lead 已阅读完整 actual diff、报告和交接回执，核 r17 保留 20 Case / 38 映射 / 九路径 / 四 top，受影响范围仅 Case01/15/18 和关联责任/证据边界；[r17 固定副本](evidence-admin-query-design/testplan-r17/lead-copied-inputs.json)已逐字节核证。两份 PASS 允许 Lead 评估后续局部 root/PG 窗口，不签发执行信号，也不解除 Case18 或整卡关闭阻塞。

[Pascal 架构咨询原文](architecture-admin2-recovery-handoff-r01.md) SHA256 `22810DC285A3905BC71C996026F909C37DDDDB986B22CCDF0D2C502CA3B0DB5B` 已由 Lead 保管。Lead 对照原 S4.2 排除 cmd 的组件写集、S10.5 的 password-reset 维护候选边界、联合恢复测试卡及身份设计 OpenGate 责任段，核对不得把候选误称为通用恢复门禁已正式分配；未重审这些身份任务的全部正确性。此咨询不是独立 Code Review 或运行结果。

首轮 U/C 普通和 race JSON 的实际运行/通过集合由 Lead 独立解析一致：U 95 节点/82 leaf、17 风险组；C 27 节点/20 leaf，对应五视图的20个有向拒绝组合，0 fail/skip。这仍是中间修订。Lead 初读新增测试后要求补齐 Case09 的真实治理前后三视图及旧入口断言、Case16 的 U/P 列表原始 JSON 键及独立逐值 oracle；交原开发者在七文件内修复，不能以此前中间 GREEN 作为最终接收。

七文件阶段随后实际交回并冻结，补强已落实。Lead 实读四生产文件完整差异和两处测试补强，现场核七个最终摘要、原148保护集、完整167项编译输入 snapshot 的文件字节及当前对应文件，独立验证23条 meta 索引的69份文件摘要、普通/race最终运行集合和两项行为负控。最终 U 为108节点/94 leaf，C为27节点/20 leaf，合计135/114、0 fail/skip；两负控分别实际报 HTTP200≠422、HTTP200≠503。最终命令及负控的七路径 before/after 均与冻结修订一致。PG只编译；第三SQL负控尚未执行。详见[Lead七文件核证](evidence-s8.2/phase1-r01/lead-phase1-verification.json)、[267份逐字节副本清单](evidence-s8.2/phase1-r01/lead-copied-evidence.json)。overlay生成/差异检查脚本的操作性失败保留，未计为行为RED。

Lead 随后独立发出 **Phase2 root CODE/LOCAL START**，仅由原Dalton写 `cmd/han-server/main.go/main_test.go`，七文件保持冻结；真实PG仍未放行。授权及扩展为174项Go/SQL/JSON/module与迁移README的输入绑定见[实际回执](evidence-s8.2/phase1-r01/lead/lead-root-code-start-r01.json)。原148保护集/155项旧记录只按原范围解读，不扩称此前已覆盖所有测试输入。Root仅做同实例factory、现役Trace/Session/AccessGeneral及两个exact GET，Case18全局两分项与整卡关闭限制保持。原七文件作者已停写后，Erdos独立审该固定对象；其评审不读取动态root，也不能替代后续九文件及PG的完整代码复核。此时Root开发与七文件评审均在进行，没有预记其结果。
