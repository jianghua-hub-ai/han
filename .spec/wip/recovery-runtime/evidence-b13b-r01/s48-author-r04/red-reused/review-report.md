# Curie B13B 跨任务 Code Review R02（最终 R03 assembled 对象）

**结论：CHANGES_REQUIRED。新增1 HIGH：B13B-BATCH-CR-R02-F001 OPEN。** 原单卡 R01失锁/R02writer panic 两项 HIGH 已由原Ptolemy R03独立关闭，Main已接收；本报告不重开它们。新增缺陷位于未改变的纯读路径，阻断本次有界B13B批次接收，不因两卡或七项结果而消失。不是Root/SQL/E2E后继缺失造成的阻断。

Reviewer为原非作者 Curie/add-reviewer；沿development-delivery/review及同版add-code-review/Role/Rules/go-service上下文继续，无新Route/子委派。固定D:/AID/VSCode/Codex/Han、main、HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca（Main交接固定；本轮未执行Git）。源码始终只读。

## 精确对象

主对象原S4.8十一＋HTTP2共13；compat2单独接收补充依赖、Runtime2保护。Main R03 assembled11于probe前逐项核当前hash；实际probe后及最终又核全部15源及Runtime2。R03生产修复仅journal.go:+7/-4 defer，其余reader/Coordinator/HTTP稳定。完整对象如下；raw绑定此版本，非未来修复。

| 主对象13 | SHA256 |
|---|---|
| internal/platform/recovery_guard.go | 0187E9A0B40FFC465240BFA8FF40FFD9DA7DB8A69B378F1931D94B6974C1826D |
| internal/platform/recovery_guard_test.go | EB4598546D62B932D053304BE4DEA45DE9C3CD5BD3BE6B4558C168A8764397F3 |
| internal/recovery/journal_snapshot.go | 5F603DD09BAEC5DF69D6C2BD2B2D44F9BB2D4D1425FD2068EEFB94D4D56E38AC |
| internal/recovery/journal_snapshot_test.go | 8D6CA0D4E5F4A82B7347BB137A538BD6CE1167AC3423BD150CB4379AC1DCA004 |
| internal/recovery/coordinator_context_test.go | 33CC74827F8A204E630681CEE25814996C978EAFA8869B5E93B859C53DF551BC |
| internal/recovery/journal.go | D1EE224C848CC29B97B1A49482D40648ACE958D35AAA3CFB400FAE48437B05D5 |
| internal/recovery/coordinator.go | 14E56074C88B197E64500BD4E9121C5683AD5CF3ED2CE602F36689C9EBBD0BFA |
| internal/recovery/journal_test.go | F872120F08A2D36F1168C09E9ED0EF42204C5CB01604F776A68B97400CC8F256 |
| internal/recovery/journal_windows_test.go | 03FC45B08A248F428D63219479A0D720136CEB9D027F4F9156F0DF342BB27E7F |
| internal/recovery/filelock_flock.go | D7A5BA43B1E9E47BC2F4C7B8AD72AED6F206ED5E56688AD665463C96036457E3 |
| internal/recovery/filelock_unsupported.go | 4AD82AEE6250919AB4236FCF367FAC006D26205378409591E88611E160359765 |
| internal/recovery/filelock_flock_test.go | 2FDDAA752658E0CB22E5C9E25F6E927F0E2A1FE496BEDB5AC50EB00762BC0198 |
| internal/identity/recovery_run_store_integration_test.go | CADC4A4529ECAB2135FA2BAF3CCF632A5FC0E0B922272F2B528C50C78B7DB1F2 |

| compat依赖／protected Runtime | SHA256 |
|---|---|
| internal/recovery/admission.go | E234CC71C5A21D42EFD6E8B345F2499800B607F2D8AF66043A0EB1A4D38FD3F0 |
| internal/recovery/admission_test.go | 02E9232EAC4A87F81582038304C1B06D6C6381DE2959CD45B95F7286B4CBFB0A |
| internal/recovery/runtime.go | 284886C23EA1038B685F1EA5BD1A6B1BC423077042939687C6EF062392594384 |
| internal/recovery/runtime_test.go | 991EA06139EF4F4C2E2845C2EC5EA08B5D35AB718FD4869BF85C4BEDCF95D5B6 |

## 两轮实际检查与复用

Round1复用本人R01的三新增文件完整正文、原11完整修改hunk（含旧测试/锁/Windows及integration三处ctx/budget迁移）与已核live=frozen对象，以及原HTTP完整独立Review。R02作者完整四文件修复由原Ptolemy独立评审及Main核收，本人消费其报告/精确对象而不冒称重新逐行审过所有中间fixture。当前完整读取R03 source2.diff、实际writerOperation/withLock相关段并核其修复是writer独立路径，不替纯读finally。

Round2在原partial基础继续，覆盖实际Journal alias/Context接口、纯validator/retained lookup、Runtime Resume/PhaseContext/ReadPermitBinding/prepareTrust/replay/refreshAfter/exact append/final proof、Coordinator phase/current source观察/fresh publication/唯一consumed分支、HTTP Acquire/Verify/Session边界及资源生命周期。由真实公共Runtime入口产证，私有ctx观察用于限制既有授权，不伪造private grant。相关正文在既有R01与本目录readset中可逐范围核对；未扩成全库/root/SQL审查。

两轮发现的纯读panic释放风险经本人的最小合成LOCAL补证；原HTTP0 finding/原56+6没有重开或整套重跑。本次新HTTP1叶验证新增上游revision漂移的组合影响。

## [HIGH] B13B-BATCH-CR-R02-F001：纯读端口panic绕过持有lease释放

- 对象：journal_snapshot.go SHA5F603DD09BAEC5DF69D6C2BD2B2D44F9BB2D4D1425FD2068EEFB94D4D56E38AC，含于当前R03 assembled11；状态OPEN。
- 位置：`internal/recovery/journal_snapshot.go:96`，函数withLock自:79，下一行:97只有fn正常返回才执行Close。ReadVerifiedSnapshot:107和VerifyBeginSecret:231共用该路径；与journal.go:798的writer defer不同。
- 违反依据：core coding-standards资源关闭权/生命周期、go-service coding-standards §1.1 defer释放资源；本批纯读持锁/ctx/真实结束与释放的契约。问题是实际资源泄漏，不是写法偏好。
- 触发：有效reader取得有效lease后，真实Anchor.Load端口panic。snapshot内部展开绕过 `return errors.Join(e,l.Close())`，上层即使恢复panic也无法替隐藏lease关闭。
- 后果：held read lease/底层映射句柄仍持有；native共享锁映射下可阻塞后续受控writer/恢复操作。现场仅实证合成lease计数，没有声称native POSIX执行或真实provider故障已验证。
- 独立证据：TestB13BCrossReaderPanicReleasesLease/Snapshot 和 /VerifyBeginSecret，普通/race分别实际执行且编译成功，两者均日志 `original_panic_preserved=true anchor_calls=1 lease_closes=0`，对Close恰一次断言FAIL。不是未命中、编译错误、skip、超时或race告警。外层test recover仅观察原panic对象及资源，不修生产实现。
- 最小修复方向：在确认取得有效纯读lease后立即注册deferred Close；正常/错误保持errors.Join及内部cause，panic展开仍释放恰一次并保留原panic。保留纯读零写，不将writerOperation/持钥信息提升为reader或Runtime/root的新前置/authority。由Main交原作者在其精确授权范围修复，Reviewer不写生产源。
- 复审条件：两个原失败在新hash下保持同一真实端口panic/原panic身份/Close1；增加正常/error/Close-error合并与取得lease后的Check/读取端口panic必要边界；保持实际reader→Runtime/Coordinator/HTTP正负控、normal零repair及原writer两项关闭不回退。修改源后重新核精确对象/适用格式处置，而非继承旧SHA结论。

## 自有实际执行与raw

只在唯一自有Temp创建测试overlay：recovery只覆盖当前coordinator_context_test.go（原全文＋自有probe）；platform只覆盖当前recovery_guard_test.go。生产overlay0、生产修改0。未仿造private phase或为Windows Unsupported装生产fallback；软件reader使用明确合成media/Anchor/lease端口，真实parser/HMAC/reader/Runtime/Coordinator代码实际执行。

四个Go调用均用从Main逐字节复制的OS/path allowlist runner，强制GOENVoff/GOTOOLCHAINlocal/GOPROXY+GOSUMDBoff/GOFLAGS+testDSN空。没有真实ENV秘密/PG/integration/服务/worker/restore/root/下载。全部调用已退出且子测试进程实际join，无pending session/wait cell。

| checks目录 | 实际结果 |
|---|---|
| cross-normal / cross-race | 各8叶：6PASS、同2FAIL；exit1，0skip，无race告警 |
| http-cross-normal / http-cross-race | 各1叶PASS；exit0，0skip，无race告警 |

每种模式合计9叶，7PASS/2FAIL。`checks/<label>/stdout.log,stderr.log,result.json`保存完整argv/时间/env/exit/raw；`probe-summary.json`保存逐叶及rawSHA。完整冻结overlay为 `coordinator_context_test.go.frozen` / `http-cross_test.go.frozen`；原失败链保留，不覆盖为将来Green。

## 跨卡集成、覆盖和必要性能评估

独立正负组合不是单卡PASS相加：

1. 实际authenticated pure reader/parser＋K1 verify-only/K2 active→公开Runtime.RunNormal/Verify/WorkRequest，原完整历史/原HMAC进入同Runtime且正常许可绑定正确。tampered MAC、pending、failed历史全部关闭，零Begin/Mark/CAS，不normal repair；候选secret经真实retained-key verifier核对。
2. 实际reader→公开Runtime.Resume→实际Coordinator，精确ready_to_consume+已committed consumed窗口只走2个Store转换、1次exit/startup、0Mark/CAS；相同consumed放到更早identity_ready拒绝，0转换/exit/startup。fixture外部Store/publication等为合成端口，不能当真实SQL原子性证明。
3. 单独真实Runtime→真实guard/HTTPAuthorizer：上游Recovery Journal revision漂移，Readiness实际Verify拒绝并撤旧ctx；新请求503在Session前截断；阻塞同步Write保持原lease、取消不假join，Write返回后Release1、设施才关闭。该HTTP fixture的上游端口为合成；没有声称单个生产factory已串完整reader至HTTP或同实例root装配已证。
4. 纯reader的Snapshot/VerifyBeginSecret acquired-lease生命周期测试实际失败，补出Main59/单卡90未覆盖的panic消费边界。因此覆盖评估不能只看原top数。

fresh publication四参数顺序/精确源/Ensure缺失才写/ACKunknown独立回读、旧代际/ctx/来源ABA等由本人有界静态检查、原PtolemyR01/R02/R03与Main当前Coordinator42/Runtime271/reader59证据共同支持相应范围；不把他人执行记成本人执行。单卡两项旧HIGH保持CLOSED，新纯读HIGH单独OPEN。详表在coverage-and-performance.json。

性能适用性：纯读多次完整认证/编码、Coordinator多次authorize/source重读存在全历史扫描常数开销；检查显式byte/record预算、完整而非截前N条、固定证明尝试/有限阶段循环及合作ctx边界。自有fixture是小样本，未执行吞吐/P95/最大历史容量benchmark，不将8MiB/100000/30s合成取值、race或测试耗时叫生产性能PASS。当前两卡最低U/C＋race，不因无生产SLO测量新增Task/反向前置；真实规模评估保留为未验证风险。

## 最终输入与Main证据的独立核对

已全文读原PtolemyR03 report、结构化receipt及Main s48-review-main-r03.json：原writer失锁/原writer panic CLOSED、新增0，90叶normal/race由Ptolemy执行并由Main核收。本报告只接收这个精确对象/修复范围；未把其“纯读无writer前置”判断外推为纯读panic清理已验证。

Main main-joint-evidence-r03/index.json 的47payload已逐项核SHA；本人解析required normal/race实际raw，各428叶=59reader+42Coordinator+271Runtime+56HTTP，逐叶均PASS、normal/race名字相同、0requiredskip/fail。七个result/raw独立核为2/0/0/0/0/0/0，source15fmt空输出PASS；没有重跑这些检查。旧23/rootraw与baseline逐字节同一事实归Main证明，本人未重复哈希旧证据源。format-disposition-main-r03.json完整结构/当前15SHA已核匹配，rootfmt仍FAIL2，有限软件格式处置不是7/7PASS、不豁免本HIGH，源变化后须新核。

## 结论、限制与交回

最终单卡输入现已齐全；独立跨卡检查新增一个内部组件HIGH，因此**不建议接收/关闭B13B或原两卡，当前batch CHANGES_REQUIRED**。这不是“等待Ptolemy旧finding未回”的pending，也不因后继Root/SQL/E2E未实现而增加本批前置。由Main安排原作者定点修复，再由本Reviewer对本finding及受影响delta复审；稳定Round1、原HTTP及本次未变正负证据可复用，不整包重做。

未验证：真实nativePOSIX no-follow/flock/fsync/介质/独立Anchor/provider/keyservice，真实PG/Store/Session/publication SQL/事务，完整restore/三worker/唯一productionfactory/root/同实例注入、Case18/S8.2、TCP/Host/部署/E2E/人的接纳；均归原S4.16/S4.17及各关联责任，不伪造已证。不承诺任意I/O强杀、撤回已发送字节或Hijack后ownership。

本目录primary为review-report.md / receipt.json，另有readset、current-objects、probe-summary、Main证据audit、coverage/performance、完整raw与测试overlay。**STOP / FROZEN**；无共享/源码/Task/Plan/设计/Rule/Scope写，无Git/清理/子委派/真实环境动作，无运行中的Go/服务/待join句柄。
