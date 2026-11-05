# S7.1 九文件独立 Joint Code Review R01

**9fileJointGate：CHANGES REQUIRED。Root4CodeReviewGate：CHANGES REQUIRED。**

**新增 finding：1 MEDIUM，`AM-S71-JOINT-R01-F001` OPEN。** 原 Core `AM-S71-CORE-R01-F001`、`AM-S71-CORE-R01-F002` 保持 CLOSED；SMTP/Core 原局部报告不可变。本报告没有将局部 PASS 相加为整体 PASS。

独立 Reviewer 完成两轮检查。父 S7.1 继续 `[-]`，B9A 其他 Task 不在本报告范围内；本地 Code Review 不代表真实 PostgreSQL、SMTP 联合集成、Host、部署、E2E 或 production readiness。

## 固定对象与权限

固定工作区 `D:\AID\VSCode\Codex\Han`，沿用 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本次对象是以下已停写的九份实际源码，已逐项核 SHA256；不以 HEAD 旧正文代替未提交实现。

| 固定源码 | SHA256 |
|---|---|
| `internal/mailer/smtp.go` | `28237089CE76B353CC01D36847378CF4775AA9281210175AAC59905DCC192853` |
| `internal/mailer/smtp_test.go` | `9833F766F24CEA8067D9A2C46BB13C01DF2568DE5A7EA998675318EDCA2B964E` |
| `internal/article/notice_dispatcher.go` | `C49CF9D7C78E5DC99E2177B6F760BE5B191D2120F3FC47DBBE8A7BEF93BFFB97` |
| `internal/article/repository.go` | `80529712F7D62ACD4A5A976621FF1E7AB11069D11CDE2D73B5050CDCA58758F8` |
| `internal/article/notice_dispatcher_test.go` | `7761947F288890CCC66125E7C5B0DF153F3568C72C7EF05ED9291A702F9D8ADD` |
| `cmd/han-server/main.go` | `2B0599AEEB1154B18C44F75992C58663A144634C4D806B2FDEA2C2503EDAC115` |
| `cmd/han-server/main_test.go` | `C8C1E94FCDEAB777364B582165E34B9F86EB2428653A1298E644F2E3944FE42F` |
| `cmd/han-server/lifecycle.go` | `8F1FE0EFEEF230BED7EFDB02D93190415C072DD6F6CC3EE5EC80E4AAAFB62BEB` |
| `cmd/han-server/lifecycle_test.go` | `662A2693547A687F33BEEAD581B937023F9525306B375F773D15FD4F1228FBB6` |

唯一写权为本报告；源码、Task/Plan、design/test-plan、Rule、全部旧报告及用户既有未提交均未修改。没有 Git 写、清理、委派或对外发送。

依据沿用同版 Reviewer Role、add-code-review、当前 add-delivery-workflow 及已核 checklist/abstraction、编码、安全、task-execution、go-service 规则；续接当前 Workflow，不重新 Route。本轮针对生产接缝补核 core/profile 可观测性规则全文、真实 `platform.LoggerWithTrace` 与 ADR-0022。fixed design r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`、test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` 未变。不因 draft/pending 历史 metadata 另起确认循环，不重开 S6.2 正文或无关任务。

## 第一轮：root4 真实增量及局部正确性

实际比较 temp 根 `root-r01-before-*` 与固定 root4，读完全部真实增量及所需上下文；完整读取现役 lifecycle.go 699行，沿 main 的生产入口、配置加载、资源取得/移交、builder、runtime、coordinate 调用核查。main/main_test 未变的全部旧正文不作无边界重审。SMTP/Core 同 hash 正文和测试复用先前实际全文审查，并定点复核本次接入依赖的签名、分类、Gate、context、Err/Done、审计和日志实现。

| root4 | before SHA256 | 实际正文差异（新增/删除） |
|---|---|---|
| main.go | `4B9D83BA52E322E9E7A430F18A59DAA0FC651D4D4B21BEAFBA825E7E991A7569` | 122 / 0 |
| main_test.go | `1FB995EA3DE93A3F9AC2CC575D4AC86A9084592742EEC0C86F1ACF4E9CF28C9E` | 342 / 2 |
| lifecycle.go | `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888` | 343 / 97 |
| lifecycle_test.go | `B3AA659B1A98E0AE8A3EF09678309866DC3D46345B2A6C29A5A220D700B36DDF` | 480 / 31 |

四份 `root-r01-final-*` 与工作区逐字节一致。`root-r01-final.diff` SHA256 `023E5C1CE259E053FCBB45E27E1DB8E196F88DFBBC6D2FF7E5EA784427DE962F`。before/final 差异独立重算，未仅采纳作者摘要。

核查结果：六个新键无默认值；duration 拒绝空白、非正、溢出与亚微秒，BatchLimit 使用规范正十进制 signed32；core.Validate 与 root 共同检查 Lease>SMTP+4DB、Shutdown>SMTP+2DB，等号/加乘溢出拒绝。调用在 DB parse/connect/ping/migrate 之前，配置错误不回显原值。旧 MAX_ATTEMPTS 仍只用于创建快照，未被 dispatcher 新配置覆盖。测试中 SMTP 单字段60s端点与完整 tuple 交叉约束已分开，没有为修通知改窄原单字段 parser。

新 builder 实际调用 `article.NewTransferNoticeDispatcher`，使用同 pool、Repository、AuditStore、IDs、prepared.administratorMailer；worker UUID 每实例生成一次，构造不启动工作。processingGate 缓存原 pool-bound 展示实例，tx adapter 复用同 config、以 caller tx 构造原 privacy.Store/ProcessingGate，未另造隐私权威。runtime 发布真实 worker；生产 `requireNoticeWorker=true`，nil/typed nil/无 BeginShutdown 能力在 listen 前拒绝。

runLifecycle 将初次调用派发到有界结果通道，同时 select 原 signals/父取消；每个初次调谐、Start、Serve 前检查停止，完成分支也先核停止，避免已排队事件被完成分支越过。首次观察停止时建立一次 shared deadline，先 BeginShutdown(sharedCtx)，再取消 work/Stop；后继事件不重置预算。初次调用实际退出、适用 worker Wait/Done 全部 join 后才关闭依赖，未 Serve 不等待不存在的 Serve 结果。

结果通道容量为2 servers+3 workers，WaitGroup 后 drain 不造成五结果死锁；错误仍按 business>metrics>password-reset>deletion>notice>stop/cleanup 排序，保留根因链；Journal/pool 关闭各一次。纯取消过滤同时检查真实 worker.Err，未将本轮 core 保存失败包装的取消一概视为正常信号退出。第一轮另发现 notice 使用未绑定生产关联字段的 logger，第二轮确认后列 F001。

## 第二轮：九文件跨组件链与真实消费者

重走实际生产链：`main → productionDependencies → run/runComposed → loadStartupConfig → newProductionBusinessHandler/newProductionBusinessBuilder → buildNoticeDispatcher → businessRuntime.noticeWorker → lifecycleConfig.noticeWorker → runLifecycle → Reconcile/Start/BeginShutdown/Stop/Wait/Done`。这条链的字段传递、真实类型和 nil 拒绝分别结合源码及 production factory 测试核实，未使用测试自装 worker 替代生产构造证明。

| 接缝 | 实际核查及结论 |
|---|---|
| root → SMTP → core 分类 | prepared Mailer 已支持 transfer purpose；实际四拒绝点只包装400..599，core 消费 Stage/Code/Is/As 并核原 textproto cause。MAIL 配置类与 RCPT/DATA4xx/5xx分开；Auth/greeting/TLS不伪装 typed rejection；DATA accepted 后 QUIT 错误仍成功。无 error 字符串匹配或原文日志新增 |
| root Gate → claim/preflight | tx adapter 的 config 与 pool Gate 同源；claim 同 tx 读取原 privacy 登记；SMTP 前再开短 tx 核 full fence/Gate。claim/preflight commit/释放后才 Send；没有为 SMTP 保持 DB 事务或连接的应用层路径 |
| core → Repository/model/schema | PK/status/owner/attempt/精确微秒 lease 和 DB fresh clock 仍约束结果；同 ID 同 lease 的 ABA 由 attempt 隔离。attempt/max 业务 int64 和数据所有权不变；最终 processing 崩溃、Gate deferred→耗尽 retry_wait、饱和退避及原子风险/终态审计路径保持 |
| core → AuditStore | 原 F001 真实 attempted 旧 claim 先结束结果 tx 后独立有界风险记录；失败低敏回退。原 F002 私有计数字符串经真实 JSON 编解码保真，严格绑定拒绝类型漂移。未知 claim 不发；未知结果仅回读，不重发或盲补审计 |
| lifecycle → core 保存/退出 | core 初次 Reconcile 计入 active，sharedCtx 先绑定，当前结果最多一次保存及必要 unknown 回读。真实保存错误在 stopping 时进入 Err；root 读取 authoritative Err，取消链不覆盖其失败。Wait 预算到期取消 settlement 后仍等待实际 Done；root 在实际 join 前不关 pool |
| 旧 worker/双 server | 初次顺序仍 deletion→password-reset→notice；未 Start 的旧 worker 不盲等 Done，而初次调用仍 join；notice 可在初次活动中 Wait。既有 worker 和 server 相对错误优先级、fallback Close、listener/Journal/pool 取得与释放边界已核必要正文及现役测试 |
| core 日志 → root sink | `main.go:412/509/2760` 到 `notice_dispatcher.go:350` 无 service/module 注入；见 F001 |

稳定消费者 hash 与先前记录相符：privacy.Store `6E555D74E5210EA20CE0C33DB08DA40CD6BA4B7D1549B9A8031E0F4D82EEDD9D`；AuditStore `091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F`；article model `3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4`。本轮读取的 platform/logging.go SHA256 `F362D36954DF0119D5DC25040640C796069CC8E178A1D92B828C6D6149CF30A9`；framework.md（含 ADR-0022）SHA256 `8D381C9D753242B47E1B6CF0C269228AD4ED591A4605CA4F119CAF03DB4E0AC7`。

原 Core R02 SHA256 `08187C0005EB9BE074DCA7E0E07EC478D18FE184877E8588CA59BA42412F2C21`，原 SMTP R01 SHA256 `4762EFC4ED1F312835B70A8D4B8C3BE2BB4C73982DB069755D414FEC1BD7D018`。原 Core F001/F002 的修复没有被 root 改动撤销，继续 CLOSED；设计侧首次取消 finding 的设计关闭也不被重新打开。当前 joint finding 是生产装配接缝，不改写旧报告。

## 新增 finding

### AM-S71-JOINT-R01-F001 — MEDIUM — notice 生产告警未绑定必填 service/module

**位置：** `cmd/han-server/main.go:2760`（注入 Logger）；来源 `:412`、`:509`；直接消费者 `internal/article/notice_dispatcher.go:350`。责任交回 root4 原作者 Lorentz。

**违反依据：** core observability-standards §1 要求结构化日志包含服务/模块、事件、关联和结果；已启用 go-service observability-standards §1.1 明列 service、module 必填。项目已接受 ADR-0022 第618行及 architecture 第585行要求生产 JSON logger 经既有 `platform.LoggerWithTrace` 等项目统一关联边界输出；该 helper 明确绑定 service/module/trace。当前其他实际 Login/Origin/Article response logger 也在各自适配器补关联字段，没有依赖 JSONHandler 自动补齐。

**触发场景：** 生产 notice 一轮碰到 ProcessingGate closed、租约竞争或结果/独立风险审计保存失败等需要低敏告警的路径。runComposed 在第412行创建基础 `slog.NewJSONHandler(output,nil)`，第509行原样存入 composition.logger；第2760行原样传给 dispatcher。core.observe 只输出固定 msg、reason_bucket、trace_id，未补 service/module；实际链上没有 `.With`、LoggerWithTrace 或额外 handler 包装。JSONHandler 自身只生成 time/level/msg 等内置字段，不从 context 推断 service/module。本机 `go doc log/slog.JSONHandler.Handle` 也核实这一行为。

**实际影响：** notice 的正常可恢复故障往往结束本轮后继续运行，顶层未必再报同一失败；其低敏告警尤其在审计无法耐久时承担诊断职责。缺少必填归属字段，按项目 service/module 维度的结构化查询无法把这些记录归入 han-server/article。当前也没有结构化 event 字段，只有固定 msg；本 finding 的明确阻断依据是 service/module 缺失，并非 msg 文案或时间键命名偏好。未据此声称现有已部署告警实际漏报，也未发现这里泄露 PII。

**最小建议：** 由 Lorentz 在现有 root 装配处从同一 logger/sink 派生 notice logger，绑定既有 serviceName 和 article 模块归属，并按项目约定提供稳定事件标识；core 继续提供每轮 trace_id/固定 reason_bucket。复用同一 handler/sink，不新增日志框架、pool 或公共 API，不把 SMTP cause/邮箱/配置原值带入日志，也不改共享业务状态或旧消费者语义。观察 seam/测试应核实际注入的 notice logger，而不是仅断言与未绑定的 base logger 指针相同。

**证据与验证：** 上述真实构造/赋值/调用链和本机 JSONHandler 合同构成源码证据；本轮未新增临时程序运行生产告警，未伪装已有测试曾捕获这条缺字段 JSON。当前 `TestNoticeProductionWiring_ActualFactory_PublishesSharedWorker` 仅断言 observed logger 为 composition.logger，不能验证字段。整改后用实际 production factory 所注入的 logger 与同一 JSON sink 捕获低敏 notice 记录，断言 service/module、稳定事件语义、trace_id、reason_bucket，并用敏感 canary 核无 SMTP/地址/凭据原文；继续验证同 sink、真实 worker、原 factory/runtime 与退出测试。交新固定 hash，由同一 Reviewer 核原 finding 和受影响九文件接缝。

## 四种初始取消时序的实际覆盖边界

未把测试名称当作真实行为：`lifecycle_test.go:937–965` 的 send blocked、accepted before save、next recipient before claim 三组使用同一 noticeLifecycleProbe 回调，均等 work.Done；只有 accepted 组多执行 `saves++`。这些测试证明协调器在活动初次调用期间绑定共享截止、取消、阻止后继 Start/Serve 和 join 后清理，**不证明实际 DATA/SQL/下一 recipient 的三个内部阶段已经在 production worker 中发生**。PrequeuedStop 另证明初次调用前已排队停止不派发；ReadyStopAndReconcile 证明完成与排队停止不推进阶段。

本轮实际查看真实 factory→runtime→lifecycle 配置传递测试，同时沿同 hash Core 的 `TestNoticeDispatcherShutdown_InitialActive_AcceptedSavedOnceBeforeDone`、SaveAlreadyStarted/Wait timeout 等测试和源码核互操作：它们使用真实 dispatcher 方法、内存事务与 Mailer 替身，覆盖实际取消/一次保存/零后继发送/Err/Done 行为。因此这里按 r15 第358行允许的局部 fake 与真实 factory 分工核本地合同，不把这些证据升级为四时序的真实 PG/SMTP 组合执行。其组合运行、连接/事务资源三层证据仍待 S7.2；本报告没有把它列为已执行，也不由测试节点数代替该边界。

## 原始证据与验证结果

临时来源仍为 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672`。本轮独立解析以下 root 原始 JSON（UTF-16 BOM，非空行均成功解析），并读取 exit/commands/limitations；没有仅转述作者 evidence 摘要。

| root 原始日志 | 独立解析结果 | SHA256 |
|---|---|---|
| root-r01-red2.json | exit1；4 top /59 run、fail test nodes；另 package fail；0 pass/skip | `2F77D1DAC9DACF2A0F35E6DD4F7A3E4BB8BA99EB9357AF32962E7F39C92BC20C` |
| root-r01-final-green.json | exit0；30 top /107 run、pass nodes；0 fail/skip；package pass | `FAA74F837F1ED7CD476020DA607F530B08F505BEAD9CEEFCB4750ADDB114AF39` |
| root-r01-final-race.json | exit0；30 top /107 run、pass nodes；0 fail/skip；package pass | `E1A8A66367A1F178364EBF5FC61C5305D242622B54E50BEF854719AD995A4D5E` |
| root-r01-final-regression.json | exit0；116 top /764 run、pass nodes；0 fail/skip；package pass | `7483C6071A48DDCC8D09EF1ACED3AA7E06787AB1BE48CC854BBD8125D3C97DA5` |

权威 RED2 的四顶层为 PrequeuedStop、InvalidFields、CrossBudgets、ProcessingGate_IsShared；原始失败包括停止后仍 Reconcile/Start/Serve、非法配置/预算被接受、Gate 被重构，均为行为 RED。四份 red2 源码快照齐全：生产 main/lifecycle 与 before hash 相同；red2 main_test SHA256 `CEB035FF87733FDB143660A1D235FC44BCD2C5CFBFB93A5289AA2756F03E0F6E`，lifecycle_test SHA256 `94797A338A81F70C2F3DED26F7ACE9F93C3865FAB0F27C0E42EB96A62EB2E18D`。first RED 的不合法 fixture 仅作历史，不替代 RED2。

作者 root 定向命令为 `go test ./cmd/han-server -run '^(TestNoticeConfig|TestNoticeProductionWiring|TestNoticeLifecycle|TestRunLifecycle|TestLifecycleExitCode)' -count=1 -timeout=90s -json`；race 为同命令加 `-race`。root regression 精确排除 `TestMain_ConnectionFailureIsNonZeroAndDoesNotLogConfiguration|TestMainHelperProcess` 两项，命令为 `go test ./cmd/han-server -skip '^(TestMain_ConnectionFailureIsNonZeroAndDoesNotLogConfiguration|TestMainHelperProcess)$' -count=1 -timeout=120s -json`，故不称该116顶层覆盖被排除的真实连接 failure child/helper。作者 root vet/build exit 文件均0，stdout/stderr为空。

Lead 在最终九文件稳定后执行了独立的完整 Manifest 检查；本轮读取现役 add.manifest.yaml，核顺序和命令、逐份读取七日志并核其 hash 与 results 索引一致：

| 顺序 | 命令 | 已核结果 |
|---|---|---|
| 1 | gofmt -l . | exit0，零输出 |
| 2 | go mod verify | exit0，all modules verified |
| 3 | go vet ./... | exit0，零输出 |
| 4 | go build ./... | exit0，零输出 |
| 5 | go test -tags=contract ./... | exit0，包结果成功 |
| 6 | go test -race ./... | exit0，包结果成功 |
| 7 | go test ./... | exit0，包结果成功 |

这些为已核 Lead 命令证据，非 Reviewer 重跑。日志部分包含 cached，且非逐叶 JSON；未从七项成功外推 integration tag、PG/外部 SMTP、Host 或真实四时序。作者 root 有限回归、Lead 完整 Manifest 和本次独立定点测试分别记账；不把前者排除项混称为后者也排除。静态 finding 不被已有命令 exit0 消除。

Reviewer 为实际 factory 接线、初次停止分支及取消包装的保存错误问题，独立执行一次窄测试：

```text
go test ./cmd/han-server -run '^(TestNoticeProductionWiring_ActualFactory_PublishesSharedWorker|TestNoticeLifecycle_InitialStages_StopAndJoinBeforeResources|TestNoticeLifecycle_ReadyStopAndReconcile_NoNextStage|TestNoticeLifecycle_SaveFailureWrappingCancellation_RemainsNonzero)$' -count=1 -timeout=30s -json
```

实际 exit0，4 top /12 run、pass nodes，0 fail/skip，package pass（0.162s），stderr为空。该命令证明现有定点断言仍通过，不是 F001 日志缺字段的失败复现。未新写测试或临时源码，输出仅工具捕获；未无故重复全量或 race。

## 长期证据位置与交回

Lead 已将42份证据复制到仓内保管。下列链接是本轮同名 temp 证据的长期副本；本轮读取两份索引，复制逐字节一致性及复制前敏感扫描由 Lead 核证，未重复42份拷贝比对，也未删除 temp：

- [复制索引](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/lead-copied-evidence-r01.json)、[九文件固定索引](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/lead-fixed-source-before-joint-r01.json)。
- [root diff](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/root-r01-final.diff)、[root evidence](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/root-r01-evidence.json)、[RED2](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/root-r01-red2.json)。
- [最终 root green](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/root-r01-final-green.json)、[race](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/root-r01-final-race.json)、[有限回归](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/root-r01-final-regression.json)。
- [Manifest 七项索引](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/manifest-s71-r01-results.json)，SHA256 `6950A224FDF0F0BBD9B3983498BBBC4926C7D755750D8A3482629C123006069D`。root evidence SHA256 `9610A3E7E90BC10E0F6BD6806A3CCBF4C632DB75B4C56C95DAC12307A29FCA0B`。

本轮没有运行 PG、外部 SMTP、通知真实集成/崩溃子进程、Host 或 E2E，没有读取/打印用户 DSN、SMTP 凭据或其他 secret。SQL/锁/连接池、真实网络安全和 crash 窗口仍需已规划专项执行；Windows FileJournal unsupported 等独立边界不被解除。

交回 Lorentz 原 root4 写集最小修复 F001，固定新 hash 后交同一 Reviewer 复审，并核受影响九文件接缝；无需改变设计、公共 API 或 Core 原两项关闭结论。当前没有其他新增具体 finding 或需决策的方向变更。父 S7.1 保持 `[-]`，由 Lead 在实际整改/复审及适用交付证据齐备后收口；本报告不代替人的接受或下一动作授权。

报告交付前复核九源码、design/test-plan 和原报告 hash；仅新增本报告，交付实际 SHA256 后停止写入。
