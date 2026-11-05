# S7.2 独立 Code Review R02

**CodeReviewGate：PASS（固定R02修订，原finding及受影响接缝）。**

**AM-S72-R01-F001：CLOSED；AM-S72-R01-F002：CLOSED。新增material finding：0。**

非作者 Reviewer Tesla 完成两轮有界复审。原R01和全部旧报告不变；本结论是本地专项代码复审，不代表生产Host、部署E2E、项目整体验收或人的签字，不自行关闭Task。

## 固定对象与依据

工作区 `D:\AID\VSCode\Codex\Han`，沿用main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。Ohm已在handoff声明stopped，Lead随后确认Host completed/stopped；实际12源码hash前后核对一致，未审活动版本。

| 测试文件 | R02 SHA256 |
|---|---|
| `test/integration/article/notice_dispatcher_test.go`（未变） | `54C74D6A014F37B257AADFF6D25C34B9776E50890A5BADFAA660CB27D8C8CEC0` |
| `test/integration/article/notice_smtp_test.go`（唯一修改） | `9E96006CD8036A00DCB577CCA5975CECB27931224B34AE4C0B3441693201CF48` |
| `test/integration/article/notice_crash_test.go`（未变） | `D2A75075C439D512896EC038A9E0EF8263CA5C11866A3F86B12FDB2E8C705479` |

生产9仍匹配 [S7.1固定索引](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/lead-fixed-source-after-joint-r02.json)，索引SHA256 `FCDFE9A132EFD50AF7DDC1213090C2B9FC68AE0AF5C04534C7C7D8B7FC7E4B5C`。未改变设计/API/生产实现、共享AuditStore、schema或其他测试文件。

原R01 SHA256 `5546FA045B6AEB5E89428367CA246562EE987DADE41AD7C8F2F16B391E0D8034`；保管的R01 SMTP前像SHA256 `76CAA71056BC38E49F382EA1C9684372CBC2ED61DE45AB1845A06516588332C5`。逐段实际比较为新增108行/删除7行；author-r02三份fixed测试快照与当前文件逐字节一致。未按“约150行”摘要推断范围。

fixed design r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`、test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` 及原R01 finding关闭要求继续适用。同版Role/Skill/Rule/Profile有效全文复用，沿原development-delivery，不重新Route、委派或全库预读。

## 两轮实际核查

第一轮读完实际修复diff与新helper的完整正文：协议采集/加锁快照/安全负例断言、成功TLS先后次序、accepted屏障、Reconcile返回处理、首行结果和真实审计回读。对照原F001/F002要求，检查是否仅迎合日志或删除原断言。原发送次数、后一recipient未claim、Wait/Done和pool归还检查均保留。

第二轮沿相关真实状态与协议核验：server.Handshake→protocol记录→MAIL/RCPT/DATA、实际Mailer错误返回和同连接负控；Send成功→BeginShutdown共享ctx→fenced保存→PG recipient→AuditStore.Get/JSON metadata。审查完整mutant相对固定生产文件的实际差异和overlay映射，独立解析RED/GREEN/full ordinary/race原始JSON；结合独立关键leaf运行核新helper。其余同hash正文复用R01，不重复整包长测。

### AM-S72-R01-F001 — 原MEDIUM — CLOSED

修复位置 `notice_smtp_test.go:157`、`:196`、`:435–471`。TLS握手成功后在相同锁下记录TLS事件，SMTP命令也进入同一有序protocol记录。新helper加锁复制commands/protocol与握手/受理计数，确实由每个原矩阵case调用。

缺STARTTLS、证书/主机/信任、Auth及greeting失败路径现在断言零MAIL/RCPT/DATA且零受理；成功/消息拒绝路径要求一次真实握手、envelope存在且发生在TLS事件之后，成功case还要求恰一次受理。原DB状态、错误链、bucket和一次连接检查没有被替换。

负控不是直接t.Fatal：`f001-early-envelope-mutant.go.txt`只在固定smtp.go缺STARTTLS分支中增加真实`client.Mail(m.config.From)`，继续返回ErrTLSRequired；overlay仅替换该生产文件的编译输入，没有落改生产源码。原始RED在真实MissingSTARTTLS_max_3连接上由新增“security failure emitted SMTP envelope”断言失败。已核完整差异和精确overlay，不把编译失败当行为证据。

本轮独立窄测同时命中implicit TLS、required STARTTLS及缺STARTTLS/Auth/错主机负例，通过协议断言。原“采集但不检查”的缺口已关闭。

### AM-S72-R01-F002 — 原MEDIUM — CLOSED

修复位置 `notice_smtp_test.go:547`、`:564–635`。after_accepted仅在真实m.Send==nil后进入屏障；新barrier helper对提前Reconcile返回立即失败，防止未成功受理却伪装该窗口。测试现在消费并检查Reconcile返回值，不再直接丢弃out。

新增真实PG断言区分两条路径：after_accepted要求accepted/attempt1、accepted_at与updated_at一致、owner/lease/next/bucket清理；during_data要求unknown/retry_wait、accepted_at空、next_attempt_at晚于DB更新时间。两者均保留recipient绑定和last_attempt_at，核准确的动作总数及每个唯一动作，再经真实AuditStore.Get核notice/target/action/trace/DB时间、system actor、kind、计数字符串和outcome。DATA中取消须有retry_scheduled＋duplicate_risk，accepted须唯一受理审计。

负控只在固定core的Send返回后、分类/settlement之前加入`if ctx.Err()!=nil { return true,nil }`；没有改测试直接制造失败。两个真实取消leaf均在PG首行仍processing时被新状态断言拒绝，说明原R01所述“跳过保存但其余退出断言仍绿”的盲区确实被保护。固定实现的两leaf及新审计断言普通/race通过；本轮独立运行也通过。故CLOSED，不要求改生产代码或新增Host测试。

## 原始证据与独立验证

Temp来源仍为 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672\s72`。独立逐行解析原始JSON、读取result中的精确命令/exit/overlay及固定快照；不是仅转述handoff/coverage。当前固定实现结果与故意mutant结果分别记账：

| 日志 | 实际解析 | SHA256 |
|---|---|---|
| r02-f001-red.jsonl | exit1；1top/2run-fail nodes/1leaf；协议断言行为失败 | `54AF801C3B32F154E321D41540F03A6E16D486B65BBF6D8BEF65DFD2B857DB78` |
| r02-f002-red.jsonl | exit1；1top/4run-fail nodes/2leaf；真实PG processing行失败 | `22A9059938955768B02F2D87A0E251A770A3C3620064C155F19118C6D002C7C9` |
| r02-findings-green.jsonl | exit0；1top/12run-pass nodes/10leaf；0fail/skip | `20421F1587A060E53EBC0F3E1E4242162AE5B7529F4478DFEB912640C1231720` |
| r02-ordinary-final.jsonl | exit0；3top/112run-pass nodes/99leaf；0fail/skip；package pass158.2s | `C892A12C3922F5120FA81A9C2311DB28A5BB5F3603D5305A175A3C1550C863B2` |
| r02-race-final.jsonl | exit0；3top/112run-pass nodes/99leaf；0fail/skip；package pass163.484s | `059000942EDE71AF206508963FC1666B72BC6871D30107A50396C8B6A6689D0B` |

RED命令分别以作者既存overlay运行MissingSTARTTLS_max_3、InitialCancellationSharedDeadline；full ordinary/race为`go test [-race] -tags=integration ./test/integration/article -run '^TestNotice(DispatcherPostgres|SMTP|Crash)' -count=1 -timeout=12m -json`。没有把这两个故意生产mutant的FAIL说成当前固定实现FAIL。

Reviewer独立执行一次有依据的窄组：

```text
go test -tags=integration ./test/integration/article -run '^TestNoticeSMTP$/(ImplicitTLS_max_3|RequiredSTARTTLS_max_3|MissingSTARTTLS_max_3|Auth535_max_3|WrongHostname_max_3|InitialCancellationSharedDeadline)$' -count=1 -timeout=60s -json
```

实际exit0，1top/9run-pass nodes/7leaf，0fail/skip，stderr为空；新增7个随机schema全部保留，执行前后12源码hash一致。DSN从User-scope仅注入进程，命令行/输出不含DSN；仅授权loopback5433 PG18.6与127.0.0.1:0合成CA SMTP，无DROP/清理。未重跑两套全量或自行再写mutant。

独立日志目录：[s72-review-r02](C:/Users/J00010385/AppData/Local/Temp/han-s71-1a90c49b44f5450a82add4222dfd1672/s72-review-r02)。`targeted.jsonl` SHA256 `95F8E9399EF00B3DB5CF25F5BAF66B57D29510D5220230C5E4E7DDABC385CB31`；`targeted.result.json` SHA256 `52BFD02097606AC510650A3ECAC14682D7B6108556F27E8803A5641835911DE3`，包含精确命令、exit/count、前后12hash和7个retained schema名；`targeted.stderr`为空文件。

## 项目检查、保管及边界

本轮逐项核真正最终 `lead-manifest-r03-results.json` 的顺序、命令、exit与七个日志SHA：gofmt -l .零输出、go mod verify、go vet ./...、go build ./...、contract、race、unit，全部exit0。索引SHA256 `5F2E463E9D7BF446ECF1484F83CEF3C460B05EFC688EB04C05223B9E57B91464`。这是Lead执行、本轮核证，不冒称Reviewer重跑，也不把普通Manifest替代integration证据。

中间Manifest R02第一项格式失败按Lead correction保留：Lead曾复制未格式化AST工具为证据inspect.go；原bytes保留为inspect.go.raw-r01.txt，仅格式化Lead自己的证据副本，source12未改。已读取correction中old/newhash及source12_changed=false、deletion_or_move=false；记录SHA256 `F0A2ECA2DCCC320DE52908E59F6013359746B8F5AAB216106C7FA19C955774EC`。这是证据保管格式历史，不是作者SMTP测试失败，不覆盖旧R01报告。

长期证据由Lead保管：[固定12索引](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/lead-fixed-source-r02.json)、[46份复制索引](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/lead-copied-evidence-r02.json)、[F001 RED](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/r02-f001-red.jsonl)、[F002 RED](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/r02-f002-red.jsonl)、[ordinary](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/r02-ordinary-final.jsonl)、[race](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/r02-race-final.jsonl)、[Manifest R03](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/lead-manifest-r03-results.json)。复制一致性与最终196schema/两取消case再次只读回读为Lead核证，不称本角色重做；本轮独立新建7schema另在自己的result中登记。

新增material finding为0，无新方向决定或风险接受要求。该复审没有扩大成全库、旧integration或部署测试；没有更改生产9、三测试、Task/design/Rule/旧报告、Go依赖或Git，没有外发SMTP/清理/下级委派。root信号协调器/Host边界仍沿S7.1与R01说明，不将直接dispatcher取消冒充生产Host执行，不代人签字。

仅新增本报告及授权Temp验证日志；交付前再核12hash及旧报告未漂移。报告SHA256由交付消息给出，交付后停止写入，S7.2收口交Lead。
