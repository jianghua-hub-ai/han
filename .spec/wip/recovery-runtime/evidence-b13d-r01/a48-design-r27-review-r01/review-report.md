# B13D Article r27 独立 Design Review R01

**结论：有界设计层 PASS，新增 finding 0。** 在指定现役消费者范围内，未确认新增 verifier 签名、授权、sameStore装配或first-use时序矛盾。该结论仅覆盖r27设计增量，不是原十源已实现、编译/PG通过、真实旧批准、人的接纳或DesignCheckpoint，也不授SOURCE/LOCAL/PG。

原Dirac/add-reviewer，独立非作者；继续development-delivery。固定 `D:/AID/VSCode/Codex/Han` / 用户指定main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，本轮不执行Git。

| 唯一对象 `.add/project/knowledge/design-workspace/article-management/design.md` | SHA256 |
| --- | --- |
| before r26 | 443C55E3E89EDBE4CF0870036BB662ED2E64B1A70F8BC5609F18025DBA71D267 |
| r27 entry/exit/live/authorafter | 4D2321D4A2D9514EA1AC2A4812E11AA6A83CA8046202F596B2F99F1DE39C7CEE |

## 两轮实际范围

Round1完整读r27 diff、作者receipt/consumer-map和Main freeze，读§3.4.7既有publication/legacy/fresh/history/ACK正文及新块、最低验证建议；Noether receipt/assembly-proposal/selftest-draft只作来源提案读取，没有应用Temp代码/SQL或把Python oracle当Go/PG实证。独立重建全部6hunks（+45/-2）；原技术行除两version元数据全部保留。原5个Go块逐字节保留、仅新增1个Go声明块；全部原SQL/DDL相关正文、28个PlantUML/5text/7JSON围栏块、**整个§3.4.6/C2**及确认tail逐字节不变，无BOM/LF状态保留。没有重审不变C2或旧Gate。

Round2直接读现役replay_evidence.go的base interface/同pool构造/LoadAttempt/Begin/Publish/ReadCurrent，ProductionDeletionReplaySource.journal及构造，deletion_replay.go的provider构造、ReadStatus和ACK重用路径；直接追Runtime.NewRuntime、Install前证/producer/CheckPhase/PhaseContext/ReadPermitBinding/CheckArticleCoordination与中性Publication/Closure ports。再读原A48整卡十文件/五风险组及Article旧TP publication对应行。实际段落/hash见读集；不是全库语义盘点、不是另一轮完整代码审查。

自身完整Role/主add-code-review/currentworkflow/checklist/abstraction/安全红线/design-writing/decisions-guide同版本人全文按hash有效复用，见methods-reuse.json，非Lead/作者代读。**S4.14 cmd5、活跃identity TP零读取/零hash；identity r55亦未重读/取hash**，原r55 PASS及其边界原样保留。

## 直接合同与消费者核查

- **唯一verifier及scope**（design:2553–2554,2578–2582）：`VerifyLegacyPublication(context.Context, LegacyPublicationProof) error`由Import唯一消费。签名没有新增grant、Store或caller bool输入；expected InstallationID/固定article-deletion/v1由固定装配、实际安装许可与verifier既存scope交叉绑定。proof保持原字段，逐项匹配独立既存批准checkpoint的source/run/backup/evidence/time/journal/epoch/head/原发布时间，不能用当前行的自洽digest、自签、仅存在或Manifest替代。现役runtime.go:189–192的InstallationSourceVerifier只验Manifest，确实不是该旧publication证明的替代方法。
- **先授权、后来源、返回后再核当前权力**（design:2578–2582）：Import先用实际Runtime/permit/PhaseContext和固定安装/id/epoch、部署独占及同namespace控制handle/资源核权，才在短run事务/行锁外同步调用verifier。nil仅表示checkpoint验证结果，不授许可；error/deadline/cancel返回前真实Join已开始I/O/子任务，保留errors.Is/cause与原分类，不泄露旧正文。返回后再次核grant/ctx/独占/资源，失权或未知零新publication INSERT/零RecoveryRun写。现役CheckPhase/ReadPermitBinding/CheckArticleCoordination可分别校实际grant、ctx绑定和封存安装/锚定起点；不能以观察DTO或构造成功代之。
- **sameStore/同pool及固定源**（design:2557–2576）：typed构造返回输入同一PostgresReplayEvidenceStore指针，不增base ReplayEvidenceStore required method，不新pool。现役replay_evidence.go:64–99确有该base interface和只存pool的旧构造形状；新增私有配置/封存可在原文件及新replay_publication.go内承接。Source是现役具体ProductionSource，snapshot能力从其同一个journal取得，不另收可错配head/journal。类型/引用校验不是来源或grant证明，实际读/写仍须原授权/完整锚定证明。
- **first-use及缺依赖分流**（design:2572–2576）：配置与所有原Store/新publication入口在同mutex下确定一次顺序；纯读、参数/ctx失败、I/O失败均算首次使用，second bind同值也拒，失败/unknown不清配置。mutex只保护状态，不跨外部I/O/Stop。IDs/Clock仅真正新receipt消耗，reuse保ref/time/旧计数oracle。LegacySources missing/typed-nil只固定禁Import；其他合法已装配Publish/Ensure/history不调用它。未配置Store的合法legacy fresh读保留，publication端口/新写不得退回旧二表Publish；已先读Store的fixture必须重建，不以补绑绕过封存。
- **冷/normal装配图可实施**：现役Runtime.NewRuntime:725–738校ctx/非nil端口并创建closed实例（含实例ID），不调用Closures/Article Store或开展准入；Source构造:69–95与provider构造:26–34也只装配引用。故可先建同Store，把其closure端口交closed Runtime，再固定同source/同journal并在Store首次使用前封存，之后normal/受控执行；延后控制handle按既有单次绑定、本地先供证再Runtime后证处理，不运行中换source或先造Ready。实际Install:431–471/518–560前证读取identity AdmissionStore、Recovery Journal、Article Anchor，不提前调用Article closure，因而没有要求Article Store先用起来才能取得安装grant的循环。该可行性是接口/依赖分析，真实组件装配正控仍NOT RUN。
- **原publication协议不缩水**（design:2520–2546）：17canonical/来源摘要、DDL五列组合FK、immutable/source_kind/generation-token、三表atomic、精确历史与fresh分流仍是原义务。history普通ctx只读，不加无关Install或verifier前置；Ensure只重申现在，Import才可核独立旧checkpoint，OriginalPublishedAt留批准来源，published_at只记实际导入。已有receipt复用，article不代写RecoveryRun，仅identity原BindLegacyPublication一次补原NULL来源；无绑定normal仍closed。
- **现役实现是实施接缝，不是新功能PASS**：现Publish:317–356仍为二表事务，ReadCurrent:481–522仍refresh并前后核attempt；tryReuseReady:257–272会把非Ready/错误汇成false。原十源中需按已定三表/ACK合同升级具体Store及私有精确回读，不能把这个bool当“明确未提交”而Fail/Replay，也不需为此扩base接口要求所有mock加方法。旧Source.journal和现有ports提供可适配接缝，但未来publication符号缺失不作为本design失败。

未确认需超原十源、改Runtime API/签名、选择新provider、第二authority或新任务的具体证据。最终root/独立来源实例仍由原owner装配，未交付root/完整S4.13不反向成为合法组件fixture的关卡。

## 后继映射与未验限制

原五风险组应精确接收以下增量，保持原Case/Task/graph与最低层；这仅是Review映射建议，不记TP/Planner已同步：

| 原风险组 | 必须保留/补入的证据 |
| --- | --- |
| PublicationCurrentEvidenceAtomic | same-pointer/pool、first-use含失败纯读、装配竞争/二次绑定拒绝；新写无二表fallback；三表真实PG atomic及actual Commit成功后ACKerror、先回读不盲写。 |
| CompositeFKImmutableCanonical | 原17字段/外部golden/组合FK/immutable、source_kind和generation/token边界；typed-nil/构造零端口I/O、ref/time不消耗。 |
| ExactHistoricalClosureAllowsRequiredReads | 无verifier/无Install普通ctx历史正向；必需精确DB/同journal snapshot/Anchor-source与前后revision读取为正，Refresh/Replay/Consume/repair/CAS写等分别0。 |
| HeadAdvanceAndReadDrift | 同一journal可信历史前缀、连续head后继仍合法、读间revision/ABA关闭，fresh仍拒stale；不换源或裁历史。 |
| LegacyReassertionVersusTrustedImport | 实际公开Runtime.Install+同控制handle及独立synthetic checkpoint正控；先核权才调用verifier、固定安装/namespace及全部proof错绑、nil后撤权/取消/失锁、I/O真实Join及零INSERT/RecoveryRun写；missing verifier仅禁Import而其他正控仍可用；复用不新ref/time，重申不冒过去。 |

Import的同handle资源必须贯穿核源、返回后复核及实际操作生命周期，不能仅凭接口类型或verifier=nil视为已经持锁/Join。构造/first-call race以及verifier阻塞时取消/Close/失权的实际行为应在原组件验证中落证，不以新Runtime hold或第二锁替代。原C2资源生命期/成功Complete及回归条件均保留。

Main原批准的sameStore NULL补绑/forward-only真实P仍归S4.13，不倒挂Article组件；Temp提出的0005 witness或0010迁移闭包只是提案，本报告未选新migration执行集或批准PG。实际独立旧checkpoint/provider/批准主体/ACL未知时真实Import继续拒绝；synthetic软件正向不冒真实批准。原技术行、整段C2和确认尾不变只证明本差量没有改它们，不把历史文字更新为新执行结论。

所有Go/gofmt/vet/build/test/编译、SQL/PG、进程/服务、native/root/E2E、PlantUML验证全部**NOT RUN by Reviewer**。本轮只运行自有Python字节/差量静态分析；未应用Temp草稿、未读取秘密ENV/DSN、无网络/安装/Git/仓内写/清理/子委派。设计仍draft、人的精确接纳unknown，无DesignCheckpoint或源码窗口。

**STOP。交Main接收本Article r27有界Design PASS/new0；后继TE/Planner/实施由Main另派，不改原identity r55 PASS，不触活跃identity TP或cmd5。**
