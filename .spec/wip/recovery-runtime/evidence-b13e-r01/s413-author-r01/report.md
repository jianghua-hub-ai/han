# identity-auth-S4.13 作者交付

原5个新增 Go 文件的实施与作者自测已交回，SOURCE / LOCAL / PG 已 RELEASE。Main已明确实核接收release、原5源冻结与6docs/8400guard，并将窗口转Lorentz原12源。接收release只释放窗口，不代CodeGate；Task不在本报告关闭，非作者独立评审与17源全仓验证仍由Main承接，人的接纳未签。任何后续修复先交Main调窗，作者不自行复占source/Go/SQL窗口。

固定工作树 Han/main，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。精确5SHA见 release.json/source-freeze-candidate.json；未暂存、提交、push、checkout、写 Tasks/Plan/设计或清理。共享树原有内容保留；8400基线现场检查仅 Tasks/Plan 有 Main 已说明的认领变动，其余基线无丢失/改动，原23格式失败文件字节未改。

实现包括 original v1 扁平 manifest codec、当前真实 install grant、独立批准、fresh/prepared 的 prepared→双namespace初始Anchor→同Store绑定→bound及中断精确续接；bound同manifest只协调，不重装。安装 writer 是同一个 RecoveryRunStore receiver，原NULL publication pair一次补绑，精确非NULL重试零UPDATE，错绑/旧namespace/version/缺批准拒绝。normal仅消费独立bound来源，Begin之后永久进入完整历史核证。

| 最低风险组 | 实际证据 |
|---|---|
| IndependentApprovalAndPreparedBound U/C/P | 真实公开Runtime；独立批准拒绝零写；双Anchor/同Store真实PG；SQL失败回滚保留prepared，重试revision=1 |
| InterruptedSameManifestOnly U/P | prepared、两Anchor、DB真实提交后、bound等边界；同manifest续接，不重置id/epoch/初始head；异manifest拒绝 |
| DeleteThenRestartAndBeginDisablesGenesis P | 实际Article finalizer/Repository/身份锁/事务/Audit完成物理DELETE及operation；合成外部Journal/CAS推进head；normal重建通过，外部Begin缺Store历史则拒绝 |
| LegacyBindingAndForwardOnly P | 原9迁移旧succeeded，再前向0010/0011；同Store真BindLegacyPublication两次，原NULL只更新一次、revision=2；run全字段与原Audit整行不变；实际S4.8历史SQL reader，缺批准/错evidence/旧错version/namespace/已错绑定拒绝；publication不可变 |
| InstallationExecutorAndBoundReentry U/C/P | 原top精确命中，原flat-v1字段负例/100次roundtrip；bound无安装元数据回写、不启普通worker、同Store完整前后像相同 |

scope-leaves.json逐叶列出最终55个scope leaf：普通55/55、race55/55，均无fail/skip。final-normal-and-regression原始一次执行611 leaf全绿，包含上述55及556个相关回归leaf，不把重复执行相加当覆盖。Recovery race37、Identity Store+PG race18；局部vet（含integration）/build exit0，原5源gofmt输出为空。

证据版本边界见evidence-scope.json：Recovery race发生时5SHA中的identity三源与最终冻结不同，实际编译/涉及的recovery两源精确相同，因此只复用这两源的37个race leaf，不声称该命令所有5SHA一致。后两包当前normal/regression与identity race的源前后5SHA均与最终冻结一致；逐命令原值全部保留。4次-w formatter是授权的有意源变更，其result.source_unchanged=false如实保留，不冒每次unchanged。

原result缺ended的命令不补造ended或Wait时间；time-evidence.json只保留原值，原result文件mtime仅作为post-Wait上界。已有ended也只表示runner在joins/源快照后记录完成，不升级为精确kernel Wait时间。Main另有实际PID采样与窗口接收，作者的Jobzero/Wait证据不冒全Host PID核验。

原23格式问题本轮用精确原23路径真实gofmt复核，exit2及完整stderr仍保留于 raw/retained-old23-format。它不是PASS，未把B13D有限处置外推为本轮17源豁免。B13E全17源格式与项目manifest由Main在17源稳定后另跑。

diagnostics/classification.json严格分账。初始空实现/缺fixture、S16 synthetic cutoff、Trace格式属于诊断，不算行为Red。真实行为失败包括：已实现Anchor分支外层err被遮蔽、合法等值时间文本被误拒、错原evidence version/namespace曾被实际接受；每项都有完整raw和修复后当前回归，不改失败为PASS。

111个本轮精确 `han_b13e_s413_` schema已在指定新PG逐名实际SQL证明存在并retain；见schemas/physical-receipt.json与完整SQL/raw。所有37个自有job均assigned-before-resume、真实Wait、kernelJob active=0、两pipe EOF和pump join；失败也同样终结，见lifecycle.json。Main PostgreSQL服务未由作者启停，不把Main helper存活/停止推断成作者join。

边界：外部CAS/文件/批准为明确合成fixture，只证明软件协议；legacy以已交付PublicationBinding作前置并实际消费S4.8历史SQL reader，本轮不冒称另跑了Article ImportLegacyPublication/native提供方。未进行真实provider/ACL/POSIX/native恢复、browser、SMTP、部署、E2E或人的接纳。无已知需扩这5源的设计/API gap；独立评审仍pending。

交付文件：source/与source.diff为冻结源/新增diff，readset.json为实际读集，scope-leaves.json为逐叶层次，raw/保留全部执行原始输出/argv/结果/源前后摘要（后期最终验证另有源字节快照），lifecycle.json为进程树终结，schemas/为新schema与物理证据，media/为合成检查点与删除介质副本，runner/为亲读来源与本轮实际runner源码。早期命令只留源前后SHA，最终验证保留完整源前后字节；不伪造早期字节副本。原D缓存/数据/原始介质仍retain，packet不含缓存。

交回Main：以release.json五SHA冻结/实核，指派非作者独立review；整合Lorentz的12源与Main全17源manifest。没有新增人的审批请求或预签；本报告属于作者交付，不是独立review PASS。
