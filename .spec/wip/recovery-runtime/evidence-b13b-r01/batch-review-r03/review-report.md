# Curie B13B batch Review R03：reader HIGH 复审与最终有界结论

**结论：PASS（当前 R04 assembled 精确对象的 B13B 软件机制独立 code review）。B13B-BATCH-CR-R02-F001：CLOSED；新增 finding 0，未关闭阻断项 0。** 原 Ptolemy writer 失锁及 panic 两 HIGH 保持 CLOSED。此结论由本人的跨卡分析、原失败的独立 Green、追加生命周期正负控与最终 Main raw 审计共同支持；没有把两个单卡 PASS 相加为 batch。建议 Main 在其现场核收后按原 Workflow 完成原两卡接收；本 Reviewer 不改 Task/Plan、不代人的确认/接收/提交/部署。

沿原 Curie/add-reviewer、development-delivery review、add-code-review 继续，不重新 Route/子委派。本文 Review 修订为 R03，被审作者 assembled 源修订为 R04。固定 D:/AID/VSCode/Codex/Han / main / HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca（沿 Main 固定交接；本轮未调用 Git）。被审对象只读，唯一新写目录为本目录。

## 本轮固定差量与所有权

Main assembled11、当前 live、原作者 frozen 已逐项匹配；主对象13＋compat2＋protected Runtime2共17个文件在入口、探针后及最后交回前均实际核 SHA 一致。只有 reader 两文件相对原 R03 对象变化。独立从原像重建 diff：journal_snapshot.go +4/-3，journal_snapshot_test.go +155/-0；与作者 source2.diff 实质一致，没有 writer/Runtime/Coordinator/HTTP/API/authority 新改动。其余13源不变，原稳定两轮和 HTTP 全套证据复用，没有重跑整包。

作者是原 Zeno，作者 receipt SHA34AE6C493581B7025F62EC977975D93A0544BD731FB12F279E2814951FB791EA；其390及原8 probes当前Green均标为作者回归，本报告的 closure 使用下面四个独立调用。作者 SOURCE/LOCAL已STOP/Hostclosed为 Main 交接事实，本轮没有运行或管理作者进程。

## 两轮真实范围与复用

Round1：复用本人原 complete bounded差量审查（原11的新文件/旧调用ctx迁移/锁/测试和 integration三处ctx及显式budget）及原 HTTP 独立审查。当前完整核两文件新 delta、有效资源取得→注册defer的顺序、named-result error join、panic展开、失败 proof 清零；新增18作者叶的触发/断言与原叶保留也已核。代码不吞 panic，不把资源释放转交外层调用方。

Round2：复用已完成的实际 ContextRecoveryJournal/private validator/Runtime proof和ctx/source连续性/Coordinator publication与consumed窗口/HTTP guard消费者核对；重新沿 reader 两个 public入口到 withLock/snapshot/VerifyBeginSecret 的释放及错误回传检查，并运行实际 reader→公开 Runtime/Coordinator、另一路 Runtime→guard/HTTPAuthorizer 的组合。合成端口只替外部介质、Anchor、Store及 resolver；真正 parser/HMAC/reader/Runtime/Coordinator/guard/HTTPAuthorizer 执行，权限由 public Runtime.RunNormal/Resume 产生。未构造私有 phase permit，未为 Windows Unsupported 添加生产 fallback，未声称实际生产 factory 串起 reader 至 HTTP。

复用边界：原HTTP Trace→exact router（404/405/Allow优先）→guard→Session→business、closed/unknown503及原 ResponseWriter透传/真实尾部返回持许可、live零观察/ready精确public_ready/不泄密等来自本人冻结原56+6 Review；本轮新9旧跨卡探针只补受上游变化影响的组合验证。原稳定来源/key/namespace/type迁移没有重复整库阅读。相关既有 readset 可从本轮 readset.json 的 reuse 链追溯。

## 原 HIGH 的复审处置

**[HIGH] B13B-BATCH-CR-R02-F001 — CLOSED。** 原位置 journal_snapshot.go:96–97（旧 SHA5F603DD09BAEC5DF69D6C2BD2B2D44F9BB2D4D1425FD2068EEFB94D4D56E38AC）；原触发为有效纯读 lease 取得后 Anchor.Load panic，旧代码跳过 Close，两个 public 入口实际 Close0。违反资源所有权/异常展开释放契约及 Go profile defer规则；共享读锁/句柄泄漏可能阻塞后续 writer。原原始失败链在本人旧目录及 canonical batch-review-r02 保留，不覆盖、不改写为 Green。

新位置 journal_snapshot.go:97–99，SHA55A05FE15DD5A35E9B45E624158068E5B96AD5DE4EC09EA85B08F04EE0505E52：在取得有效lease后、调用fn之前立即defer Close；正常结果和Close错误errors.Join，原panic自然传播。ReadVerifiedSnapshot/VerifyBeginSecret在正常错误（含Close错误）都返回零proof，未增加writerOperation/调用者授权前置或写入能力。

本人从原冻结 cross-probes.inc 逐字节复用，实际独立重跑原两条同名叶：

- TestB13BCrossReaderPanicReleasesLease/Snapshot
- TestB13BCrossReaderPanicReleasesLease/VerifyBeginSecret

普通/race均实际命中Anchor.Load一次、保留同一原panic指针、lease_closes=1，PASS。原断言没有放宽、没有换触发端口；旧8叶全部逐名保留并PASS。追加22叶也PASS：两个入口各覆盖普通ctx成功（无需writer authority）、read错误、Close错误、两错误链、Check/records/artifacts panic（Close返回错误仍保留原panic）、取消时records真实join、取消时Close真实join、预取消零acquire及acquire失败零Close。零CAS、失败不泄露proof均有断言。

Close阻塞用entered/release/done通道，取消后函数仍未完成，释放阻塞端口后真实返回；不是提前Release或后台假join。取消发生在已经完成proof并进入Close之后的用例只断言真实join及Close根因，不外推成对最后proof之后ctx的新授权校验或任意I/O强杀。Close自身panic等未测试端口故障不作额外保证。

原Ptolemy R03的writer两HIGH closure/report/receipt及Main核收作为未变writer的独立证据保留；本次reader delta不重开它们。当前无其他可操作finding、无待决定阻断项，无必要非阻断改码建议。

## 自有 LOCAL 与 raw

四个Go调用，全部exit0、全部真实结束，测试节点无unfinished/skip/fail，无race告警。原两个probe inc字节未变；本轮只新增独立 reader-lifecycle.inc。只overlay当前coordinator_context_test.go及recovery_guard_test.go（各原全文＋测试probe），无生产overlay/共享测试改写。

| 自有 checks label | normal | race |
|---|---:|---:|
| cross-normal / cross-race | 30叶PASS（原8＋新22） | 30叶PASS |
| http-cross-normal / http-cross-race | 1叶PASS | 1叶PASS |
| 每种模式总计 | 31叶PASS | 31叶PASS |

原8中的两FAIL现在Green；其他6维持实际reader→Runtime正负/Coordinator窗口。HTTP1保持revision漂移→Verify503/旧ctx撤销、新请求Session前503、原同步Write阻塞时lease仍持有、真实返回才Release1及设施Close1。真正跨组件状态/许可/资源交接均有正负控，覆盖不是只累加单卡叶数。

每项完整argv、env策略、时长及退出码保存在 checks/<label>/result.json；实际 stdout.log/stderr.log 不删不合并，逐叶/rawSHA在probe-summary.json。恢复原8的命令是go test [-race] -overlay=<本目录>/cross-overlay.json ./internal/recovery -run ^TestB13BCross -count=1 -timeout=45s -json；HTTP为对应http-overlay与 ^TestB13BHTTPSourceDriftRevokesInflightLease$、30s。所有运行均用逐字节复制Main安全runner（OS/path allowlist，GOENVoff/GOTOOLCHAINlocal/GOPROXY+GOSUMDBoff/GOFLAGS及testDSN空），无网络、真实secretENV或integration/PG执行。

## 最终 Main 实测与格式适用性

新 main-joint-evidence-r04/index.json 47payload逐项实际核hash；本人解析current normal/race JSON原始流，各446叶=reader77+Coordinator42+Runtime271+HTTP56，所有run节点实际结束，0requiredskip/fail/race告警，normal/race逐叶一致。与旧R03实际raw的428逐名subset核对全部保留，与作者当前390 recovery叶逐名相等、另HTTP56。这里是审计Main/作者执行，未记为本人执行。

七项result与stdout/stderr实际核为2/0/0/0/0/0/0：rootfmt仍FAIL2；mod verify/vet/build/contract/race/test六项PASS。source15fmt空stdout/stderr、exit0。format-disposition-main-r04.json全文结构及exact_current_source_sha256与当前15逐项匹配；是本新SHA的明确有限软件格式处置，不继承R03/S4.7、不把全仓写为7/7PASS、不豁免codefinding。本人直接比较R04/R03 root两raw逐字节相等；old23及preSOURCE基线不重复读取/重hash，其同基线事实明确归Main证明。新发现的代码阻断现由独立probe关闭，不靠此格式处置豁免。

## 必要集成、性能与未验边界

当前原两卡最低层为软件U/C＋race及原位迁移静态证据。本人的两个组合分支验证真实软件组件之间的读证明、ctx撤销、独占生命周期/唯一consumed与Session边界；共享真实数据库、provider和production root尚未装配的事实不会被此报告改写。有关Context alias/私有ctx观察不授新权/retained Begin K1不流向Runtime/root、fresh publication四参/Ensure后独立回读/ACKunknown/normal不修复、唯一ready_to_consume consumed窗口由先前静态消费者核对及当前正负probe、Main相应群组共同支持。

R04生产只改常量成本的defer finalization，仍是原先应执行的同一次同步Close，无新全历史扫描/重试/状态推进。复用前轮预算/ctx/有限阶段循环与历史扫描成本分析；小fixture和race耗时不当吞吐/P95/最大容量基准，未运行生产规模benchmark，没有性能SLO PASS主张。没有证据要求本delta新增性能Task或把后继真实环境倒设为本批前置。详见coverage-and-performance.json。

未执行：nativePOSIX no-follow/flock/fsync/真实介质、独立Anchor/provider/keyservice；真实PG/Store/Session/publication SQL及事务；完整restore、三worker、唯一生产factory/root同Runtime实例、Case18/S8.2、TCP/Host/部署/E2E/人的接纳。仍由原S4.16/S4.17及关联后继范围承接；本结论不伪称这些能力已证，也不新增前置或授权。合作取消不承诺强杀任意I/O、撤回已写字节或Hijack后ownership。

## 当前精确主对象13

| 文件 | SHA256 |
|---|---|
| internal/platform/recovery_guard.go | 0187E9A0B40FFC465240BFA8FF40FFD9DA7DB8A69B378F1931D94B6974C1826D |
| internal/platform/recovery_guard_test.go | EB4598546D62B932D053304BE4DEA45DE9C3CD5BD3BE6B4558C168A8764397F3 |
| internal/recovery/journal_snapshot.go | 55A05FE15DD5A35E9B45E624158068E5B96AD5DE4EC09EA85B08F04EE0505E52 |
| internal/recovery/journal_snapshot_test.go | 7A992AB920C597F4B011A25FD33A9691E6F179623BDD21C2136C1125DDAC0307 |
| internal/recovery/coordinator_context_test.go | 33CC74827F8A204E630681CEE25814996C978EAFA8869B5E93B859C53DF551BC |
| internal/recovery/journal.go | D1EE224C848CC29B97B1A49482D40648ACE958D35AAA3CFB400FAE48437B05D5 |
| internal/recovery/coordinator.go | 14E56074C88B197E64500BD4E9121C5683AD5CF3ED2CE602F36689C9EBBD0BFA |
| internal/recovery/journal_test.go | F872120F08A2D36F1168C09E9ED0EF42204C5CB01604F776A68B97400CC8F256 |
| internal/recovery/journal_windows_test.go | 03FC45B08A248F428D63219479A0D720136CEB9D027F4F9156F0DF342BB27E7F |
| internal/recovery/filelock_flock.go | D7A5BA43B1E9E47BC2F4C7B8AD72AED6F206ED5E56688AD665463C96036457E3 |
| internal/recovery/filelock_unsupported.go | 4AD82AEE6250919AB4236FCF367FAC006D26205378409591E88611E160359765 |
| internal/recovery/filelock_flock_test.go | 2FDDAA752658E0CB22E5C9E25F6E927F0E2A1FE496BEDB5AC50EB00762BC0198 |
| internal/identity/recovery_run_store_integration_test.go | CADC4A4529ECAB2135FA2BAF3CCF632A5FC0E0B922272F2B528C50C78B7DB1F2 |

## Compat依赖2及保护Runtime2

| 文件 | SHA256 |
|---|---|
| internal/recovery/admission.go | E234CC71C5A21D42EFD6E8B345F2499800B607F2D8AF66043A0EB1A4D38FD3F0 |
| internal/recovery/admission_test.go | 02E9232EAC4A87F81582038304C1B06D6C6381DE2959CD45B95F7286B4CBFB0A |
| internal/recovery/runtime.go | 284886C23EA1038B685F1EA5BD1A6B1BC423077042939687C6EF062392594384 |
| internal/recovery/runtime_test.go | 991EA06139EF4F4C2E2845C2EC5EA08B5D35AB718FD4869BF85C4BEDCF95D5B6 |

## 交回与再触发条件

Primary为本目录review-report.md与receipt.json；另有readset、entry/exit17hash、derived diff、probe-summary、Main审计、覆盖性能、全部测试overlay与raw。后续若源/契约/装配改变，应针对改变及受影响消费者另核；任一新代码阻断或精确格式处置失配须重新评估。本结论不自动覆盖未来SHA。

**STOP / FROZEN。** 四个自有runner subprocess.run均返回，测试内gated goroutine由done真实join，Go/package final PASS，无待join会话/后台服务/worker。未写任何生产/共享文档/Task/Plan/设计/Rule/Scope，未执行Git/清理/子委派/真实环境动作。Main可据独立报告现场核收；Reviewer不自行关闭原两卡。
