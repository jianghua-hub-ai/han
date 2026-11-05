# S7.1 九文件独立 Joint Code Review R02

**9fileJointGate：PASS。Root4CodeGate：PASS。**

**原 `AM-S71-JOINT-R01-F001`：CLOSED。新增 findings：0。** 原 Core `AM-S71-CORE-R01-F001/F002` 保持 CLOSED；原 SMTP/Core/Joint 报告不可变。

本结论绑定下列新九文件 hash，表示原 finding 与受影响接缝的独立代码复审通过；不代替父任务关闭、真实 PG/SMTP 联合验证、Host、部署或 E2E。父 S7.1 仍由 Lead 收口，本报告不将其改为完成。

## 固定对象与依据

工作区 `D:\AID\VSCode\Codex\Han`，沿用 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 的已冻结未提交对象。原作者 Lorentz 已停写；本次唯一新写本报告，其他文件及原报告不改。

| 固定源码 | SHA256 |
|---|---|
| `cmd/han-server/main.go` | `D6A82DBDD9C72D3A2435A32F1059494ABEA94E5F3B05ED1B5C93C52FAF97AFB4` |
| `cmd/han-server/main_test.go` | `0346F1AA96C6EDDB6409148E648D37FD15448F2C73AC966BA18AFE3ADA3DBA68` |
| `cmd/han-server/lifecycle.go` | `8F1FE0EFEEF230BED7EFDB02D93190415C072DD6F6CC3EE5EC80E4AAAFB62BEB` |
| `cmd/han-server/lifecycle_test.go` | `662A2693547A687F33BEEAD581B937023F9525306B375F773D15FD4F1228FBB6` |
| `internal/article/notice_dispatcher.go` | `C49CF9D7C78E5DC99E2177B6F760BE5B191D2120F3FC47DBBE8A7BEF93BFFB97` |
| `internal/article/repository.go` | `80529712F7D62ACD4A5A976621FF1E7AB11069D11CDE2D73B5050CDCA58758F8` |
| `internal/article/notice_dispatcher_test.go` | `7761947F288890CCC66125E7C5B0DF153F3568C72C7EF05ED9291A702F9D8ADD` |
| `internal/mailer/smtp.go` | `28237089CE76B353CC01D36847378CF4775AA9281210175AAC59905DCC192853` |
| `internal/mailer/smtp_test.go` | `9833F766F24CEA8067D9A2C46BB13C01DF2568DE5A7EA998675318EDCA2B964E` |

逐项核实际 hash。仅 main.go/main_test.go 较 Joint R01 变化；其他七源不变。`root-r02-fixed-` 四源码快照与当前 root4 逐字节一致，**不能以名字含 final 的旧 main_test 快照代替当前0346…修订**。before 四快照与 Joint R01 固定 root4 hash 一致。实际正文差异为 main 新增9行/删除2行，main_test 新增116行/删除2行，lifecycle 两文件零差异。

原 Joint R01 SHA256 `16BFE3466B4727600B943A37F7827CA656F72408CFAE40A09D366E15FE38543F`；Core R02 SHA256 `08187C0005EB9BE074DCA7E0E07EC478D18FE184877E8588CA59BA42412F2C21`；SMTP R01 SHA256 `4762EFC4ED1F312835B70A8D4B8C3BE2BB4C73982DB069755D414FEC1BD7D018`。本次追加复审结论，不改旧 Gate/失败历史。

沿用同版 Reviewer Role、add-code-review、当前 add-delivery-workflow，以及已实际取得的 checklist/abstraction、编码、安全、task-execution、core/go-service 可观测性规则和项目 ADR-0022；不重新 Route 或重读无关全库。fixed design r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`、test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` 保持原值。不以 draft/pending metadata 制造新确认，不重开 S6.2 或其他 Task。

## 两轮实际核查

第一轮核对象自身：完整读取实际 R01→R02 差异、main 的 logger 创建/传递/构造/观察 seam，以及新增 factory/helper/JSON 断言和既有共享依赖测试修订。核 before/RED/fixed 四源码快照、原 finding 的触发与关闭要求，确认变更没有散入业务代码或旧 lifecycle。没有把作者 diff 或测试名当作修复证明。

第二轮沿真实消费者核接缝：重走基础 JSON sink→composition.logger→buildNoticeDispatcher 派生 logger→真实 NewTransferNoticeDispatcher→core.observe；核 observer seam 确实记录构造器所收的同一 logger。检查 `.With` 的固定低基数字段、trace/bucket 每轮传递、基础 logger 隔离、真实 production factory/runtime、共享依赖身份和错误/数据边界。复用同 hash 的七源及稳定消费者先前全文核查，定点重读 core.observe；不无理由重审未变正文或重跑全量。

新日志只绑定 serviceName、article 模块和固定 warning event，未捕获实例 trace/邮箱/claim 等动态字段；core 仍在每次调用传入 trace_id 与 reason_bucket。派生 logger 复用原 handler/sink，没有新增 writer、SDK、goroutine 或连接池。公开 API、设计、model int64、共享 AuditStore、schema、HTTP、privacy Gate 及 SMTP 错误语义不变。

## 原 finding 复审

### AM-S71-JOINT-R01-F001 — 原 MEDIUM — CLOSED

原缺陷是 notice 直接收到基础 JSON logger，告警缺 service/module，无法按项目必填结构化字段归属。整改位置为 `main.go:132–133`、`:2757–2767`、`:2779`。

实际实现从 `b.composition.logger.With(...)` 派生 noticeLogger，绑定 `service=han-server`、`module=article`、`event=article.transfer_notice.warning`。该实例直接传入真实 dispatcher；shared seam 也观察这个实际注入者，不再把基础 logger 指针冒充 notice logger。`runComposed` 原基础 JSON logger/sink 及其他消费者保持原接入方式。

关闭证据：

- `main_test.go:216` 的测试通过真实 newProductionBusinessHandler 构造，取得 runtime 中真实 TransferNoticeDispatcher，并从 factory seam 取得实际注入 logger；没有用测试自装 worker 或自装带字段 logger 代替被评审装配。
- 以 core.observe 相同的 Log 参数合同输出一条 notice WARN，再用基础 logger 输出一条 root probe。JSON 解码断言 service/module/event、msg、level、trace_id、reason_bucket、time；两条记录都进入同一个 bytes.Buffer，基础记录不带派生 module，且没有多余记录。
- SMTP响应、邮箱、凭据使用合成 context canary，均未进入记录。这只证明本次 logger 不擅自序列化这些 context 值；没有把它说成实际 PG/SMTP 故障或原错误链安全测试的全量重跑。原 core 低敏参数与 error 链合同按未变 hash 继续有效。
- `main_test.go:4189` 的原共享依赖测试只对 notice logger 改为派生实例/同 JSON sink 断言；其他依赖和非 notice logger 的实例唯一性仍逐项检查，消费者数量要求保留。没有为了转绿整体删除共享依赖约束。

行为 RED 确实在旧 main 实现上失败，输出 service/module/event 为 nil；当前源码和新增字段/同 sink 断言已通过最终作者证据及 Reviewer 独立窄测。原关闭条件满足，故 CLOSED；未发现新具体缺陷或需改变设计/API的方向问题。

## 受影响九文件接缝结论

本次变化仅发生在 logger 装配及其测试，不改变真实 factory→runtime→lifecycle 的 worker 实例和强制缺 worker 拒绝；tx Gate 仍使用同 config、caller tx 与原 pool。六键的 DB 前校验、严格预算、无默认及 typed-nil 验证保留。

同 hash Core/SMTP 的错误阶段、Is/As 原链、强 TLS/deadline、DATA accepted/QUIT 语义、purpose 和最小模板保持；没有新增 error 文本匹配、原始 SMTP 日志或跨网络 DB 事务。完整 fencing、微秒/attempt ABA、耗尽维护、原子审计、unknown 回读和独立迟到风险记录保持。原 Core F001/F002 继续 CLOSED。

同 hash lifecycle 的初次 signal 监听、BeginShutdown先绑定共享截止、零后继阶段、一次结果保存、Err/Done、实际 join 后关闭依赖、五结果容量、错误优先序和资源恰一次结论沿用 Joint R01 的实际核查。本次 logger 修复不改变这些路径。R01 已说明的局部 probe 与真实 factory/Core 分工仍适用；没有新增“四时序已在真实 PG/SMTP 联合执行”的主张。

## 新证据的实际解析

临时根仍为 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672`。按实际 `root-r02-evidence.json` catalog 选择权威 **fixed-** 日志，独立解码 UTF-16 BOM 并逐非空行解析，全部无 JSON 错误、命中不为零；另核 exit 文件和精确命令。

| 本修订原始日志 | 实际解析 | SHA256 |
|---|---|---|
| root-r02-red.json | exit1；1 top/1 run、fail test node；另 package fail；0 pass/skip | `D06D9D322E6A635EA7DDB47AE61FAC5DEC478A1E325EBF1BD65E6E5F2B9B13E6` |
| root-r02-fixed-finding-green.json | exit0；2 top/2 run、pass nodes；0 fail/skip；package pass | `6385B62655BD857E46FEC3206B707D4A2886FD8D94DACC7A4AA85298AC08F422` |
| root-r02-fixed-green.json | exit0；31 top/108 run、pass nodes；0 fail/skip；package pass | `3842EE937D8EB54C3C276D89F7181DE9EFACF5E52934A5A7F2F69EE45ED6BE80` |
| root-r02-fixed-race.json | exit0；31 top/108 run、pass nodes；0 fail/skip；package pass | `68B49FD80C85D331EB76F6AF053A441E1272D4C0E148EEEC95B811ED51F89E7F` |
| root-r02-fixed-regression.json | exit0；117 top/765 run、pass nodes；0 fail/skip；package pass | `D9EB4D866E3CF538B21DE43454679857CC348F2F7993D7E787F86F131EFFED3E` |

RED test 为 `TestNoticeProductionWiring_JointR01F001_BoundLoggerUsesProductionSink`，失败是缺三个字段而非编译失败。RED 四源码齐全，main/lifecycle/lifecycle_test 与 R01相同；RED main_test SHA256 `57D66A190A522036F5E122478847A5FE7FF62A1C464CF3D81BA4343C59D4ED35`，相对 before 实际新增78行测试/helper。没有回写源码重跑 RED，也不虚构失败前后时间签核。

`root-r02-final-regression.json` 是**中间历史失败**：exit1，117 top/765 run nodes，764 pass、1 fail，失败为 `TestProductionBusinessHandler_AWIRERTR06UsesOneSharedInstancePerDependency` 的旧基础 logger 指针约束；SHA256 `2B76C2B63FD4E7A1B97357EA81312C1FE975F370C186A8EAF7368AA69976E7D7`。本次已经实际解析此失败并核对应测试修订，未忽略、覆盖或误称它为当前回归结果。旧 final main_test 快照也不等于当前 hash。权威最终回归是上表 fixed-regression。

最终 root 定向命令仍为 `go test ./cmd/han-server -run '^(TestNoticeConfig|TestNoticeProductionWiring|TestNoticeLifecycle|TestRunLifecycle|TestLifecycleExitCode)' -count=1 -timeout=90s -json`；race 加 `-race`。最终 root regression 仍使用 `-skip '^(TestMain_ConnectionFailureIsNonZeroAndDoesNotLogConfiguration|TestMainHelperProcess)$' -count=1 -timeout=120s -json`，因此其117顶层不包括这两个排除对象。fixed-vet/build exit均0，输出为空；这些是已核作者证据。

索引 hash：`root-r02.diff` 为 `1B31BF167F0D634D9D975AB53BE6E089FE1731E080F05A665E3CF9CCCFF2FFAA`；`root-r02-evidence.json` 为 `3B112B14610D6B38C0EAF7D8871689102DB39ED5F632E11DE6488C9B3CF98DF0`；`root-r02-fixed-inputs-before.json` 为 `54F823B9D86B761A084C79F12298FC4B332BFBAD379266360D00F20541BB165C`。

## 新 Manifest R02 与独立窄测

收到 Lead 完成通知后，以**本次** `manifest-s71-r02-results.json` 核全部七项；逐份读取日志并计算 SHA256，与索引一致，顺序匹配现役 Manifest。没有借用 R01 PASS。R02 索引 SHA256 `24B1AE429D5598A8D935ECF0B4E0FC98B3BB663D2102F4AD71B4E60011D81DDB`。

| 顺序 | 新命令 | 已核实际结果 |
|---|---|---|
| 1 | gofmt -l . | exit0，零输出 |
| 2 | go mod verify | exit0，all modules verified |
| 3 | go vet ./... | exit0，零输出 |
| 4 | go build ./... | exit0，零输出 |
| 5 | go test -tags=contract ./... | exit0，包结果成功 |
| 6 | go test -race ./... | exit0，包结果成功 |
| 7 | go test ./... | exit0，包结果成功 |

contract/race/unit 日志 hash 分别为 `B2C5AAF2ECF4D5D12A4424B9EB343C23D4B68D66DD291AB33F80E6ABD3C4B8E8`、`9E0347B0353ACB2C75DAD24571C7AB25F4CD4198FD2B417B09A0EC03E6DE76A8`、`4625DC2C94625CDF86FF5CFF7B289D072786EB5626E3995F015FDF195EECB134`。部分未变包为 cached；包级文本不是逐叶或 integration 证据。完整 Manifest 与作者带两项 exclude 的 root 有限回归分别记账。

Reviewer 为 F001 和受影响共享依赖约束，实际独立执行一次：

```text
go test ./cmd/han-server -run '^(TestNoticeProductionWiring_JointR01F001_BoundLoggerUsesProductionSink|TestProductionBusinessHandler_AWIRERTR06UsesOneSharedInstancePerDependency)$' -count=1 -timeout=30s -json
```

实际 exit0，2 top/2 run、pass nodes，0 fail/skip，package pass（0.317s），stderr为空，无 JSON 错误。仅工具捕获输出，未新增临时程序、测试或日志；未重复全库/全量/race。Manifest 是 Lead 执行、本轮核证，不冒称 Reviewer 重跑。

## 限制、交回与停止

本轮没有连接 PG、外部 SMTP、启动 Host 或运行 E2E，没有读取/打印真实 DSN/凭据、修补源码、改设计/TaskPlan/Rule、Git写、清理或新委派。JSON sink 测试验证的是实际 factory 注入的 logger 与低敏参数合同，未执行真实通知故障；SQL/锁/池、TLS/网络、crash 与四时序联合资源证据仍由已规划专项验证。独立 Windows/部署 Gate 不因本地 PASS 解除。

原 Joint F001 已关闭，原 Core 两 finding 保持关闭，新增缺陷0；不需新方向决策、设计/API修改或风险接受。后续若派生 logger、字段合同、实际 sink、worker/context 或相关装配再变，按影响固定新 hash 复审。

R02 证据本次按 temp catalog 的真实文件名和 hash 引用；Lead 后继保管副本不应覆盖中间失败或旧 R01 索引。本报告不提前声明尚未核定路径的新保管副本已存在。

交付前复核九源码、固定 design/test-plan 及旧报告未漂移；仅新增本 R02，实际报告 SHA256 由交付消息给出。父 S7.1 交 Lead 收口，本角色交付后停止写入。
