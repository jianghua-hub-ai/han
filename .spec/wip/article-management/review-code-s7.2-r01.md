# S7.2 独立 Code Review R01

**CodeReviewGate：CHANGES REQUIRED。2项 MEDIUM：AM-S72-R01-F001、AM-S72-R01-F002，均 OPEN。**

非作者 Reviewer Tesla 完成两轮有界审查。两项均为本次测试的保护性断言缺口；没有据此判定固定生产九文件存在新产品缺陷，不重开 S7.1。整改交原作者 Ohm，固定新修订后由同一 Reviewer 复审；本报告不关闭 S7.2。

## 固定对象与范围

工作区 `D:\AID\VSCode\Codex\Han`，沿用 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。三文件均为新未提交文件，没有旧版本，全文就是本次 diff；已逐项核 hash，并在独立执行前后核生产9＋测试3共12项不变。

| 被审文件 | 行数 | SHA256 |
|---|---:|---|
| `test/integration/article/notice_dispatcher_test.go` | 894 | `54C74D6A014F37B257AADFF6D25C34B9776E50890A5BADFAA660CB27D8C8CEC0` |
| `test/integration/article/notice_smtp_test.go` | 753 | `76CAA71056BC38E49F382EA1C9684372CBC2ED61DE45AB1845A06516588332C5` |
| `test/integration/article/notice_crash_test.go` | 240 | `D2A75075C439D512896EC038A9E0EF8263CA5C11866A3F86B12FDB2E8C705479` |

生产9固定输入为 [S7.1 固定索引](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.1/lead-fixed-source-after-joint-r02.json)，索引 SHA256 `FCDFE9A132EFD50AF7DDC1213090C2B9FC68AE0AF5C04534C7C7D8B7FC7E4B5C`；逐项与实际源码一致。fixed design r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`、test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`，沿用其通知合同。Tasks只读取S7.2卡第1613～1633行，不扩读规划库或重开历史 pending/draft。

已取得自身完整 Reviewer Role、add-code-review 和 multi-agent-coordination；同版当前 delivery Workflow、checklist/abstraction、task-execution、编码/配置/可观测/追踪/安全红线与 go-service 规则复用本会话有效全文。正式续接 development-delivery，不重新 Route、不委派。

唯一项目写入为本报告；验证原始日志仅写授权 Temp `s72-review-r01`。未修源码、改Task/design/其他报告/Rule、Git写、push/部署或清理。所有新 schema 保留。

## 两轮实际审查

第一轮完整读完三文件1887行，核 fixture、所有父/子用例、断言、SQL wrapper、loopback server、合成CA、child入口与关闭顺序。三文件所在 article integration 包没有其他旧 harness 文件，未调用旧自动 DROP helper。检查不是仅看测试名/总数，两个 finding 即来自场景名与实际断言之间的缺口。

第二轮按 r15 §4.6 全子例、§7.6、§9.5 及 S7.2 任务卡映射实际 leaf，再沿固定 Repository SQL、真实 AuditStore JSON、privacy Store/Gate、SMTP适配器、model/schema 与 child协议核证据语义。复用相同 hash 的生产正文既有审查，定点核实际调用；核完整0001～0009迁移执行链及相关通知/审计DDL，没有重新审查全部无关密码重置DDL。

| 风险组 | 实际源码/证据核查 |
|---|---|
| PG、Gate与状态 | 随机 retained schema；真实pgx pool/短事务；1～9完整迁移后提交fixture。SameTransaction提供tx可见/pool不可见登记；claim后提交closed、aborted preflight、max=1 deferred到期维护；SKIP LOCKED、4 worker、past/future维护矩阵均有实际用例 |
| DB时间与完整fence | pg_stat_activity确认真实锁等待后取fresh DB时间；规范微秒和同次审计时间；相同worker ID接管、过期无接管，以及owner/attempt/单微秒lease独立不匹配均有用例 |
| unknown与审计 | wrapper先真实commit/rollback，再返回ACK故障；真实AuditStore.Append/Get与JSONB；bigint/target mismatch/getfailure、after-insert审计失败回滚、真正issued/attempted旧结果及独立风险/fallback均有用例。未把包装器返回error本身当作真实DB提交 |
| SMTP与隐私 | 127.0.0.1:0多会话、合成CA与真实TLS；四拒绝阶段×4xx/5xx×max1/3、Auth/greeting、两安全模式、证书负例、QUIT/ACK丢失、双recipient/同地址快照、实际退避后重试、MAIL/安全故障结束本轮；协议安全断言仍有F001缺口 |
| 网络期资源 | DATA屏障时检查同pool AcquiredConns=0并执行独立SQL；cleanup cancel/Wait/Done、server close及goroutine join；停止/清算预算组合实际运行。accepted取消落库保护仍有F002缺口 |
| child crash | 四真实点AfterClaim/DuringDATA/AcceptedBeforeSave/AfterResultCommit×max1/3；真实进程非零退出、阶段输出/服务器DATA屏障、Wait输出join/socket退出、PG processing/终态与SMTP计数/风险审计。accepted-crash重复窗口有明确计数对照 |

Snapshot 疑点已独立排除：seed虽先以active INSERT两用户，但第198～200行在第二轮将其UPDATE为deleted并设置retained_email；第762行清除deleted用户的retained_email有目标行，不是“两个用户始终active”的零行分支。未因此新增 finding。

## Findings

### AM-S72-R01-F001 — MEDIUM — 安全负例缺少失败前零envelope的协议断言

**位置：** `test/integration/article/notice_smtp_test.go:383`（testNoticeSMTPCase），关联 `:157`、`:195`、`:415`。依据 r15 A-NOT-09b 第353行要求真实协议/握手断言，安全失败前无 MAIL/RCPT/DATA；S7.2卡要求真实TLS负例证据。

**实际问题：** server收集commands与handshakes，但三文件内没有读取它们的断言。证书/STARTTLS/Auth/greeting负例主要检查DB bucket/state、一次连接及部分typed error；这些不能独立证明安全失败前没有发送envelope。成功TLS也没有用采集到的握手记录作协议证据。当前9源码实现安全不等于此专项已保护该性质。

**触发/影响：** 若后继实现错误地先发出envelope命令，再返回同类安全/配置错误，现有状态/bucket/连接次数断言不直接检查这一违规。采集数据被闲置，A-NOT-09b的“零 MAIL/RCPT/DATA”没有可持续回归断言；仅有99 leaf全绿无法补齐它。这是测试盲区，不是声称本轮观察到实际明文或越权投递。

**最小方向：** 在原三文件内，以加锁快照读取真实server命令/握手/受理记录；安全失败路径断言MAIL/RCPT/DATA及受理为零，成功两模式断言确有安全握手后发送。按阶段区分Auth/greeting与消息拒绝，保留现有类型/码、DB状态和一次连接断言。不改生产Mailer或降低TLS验证。

**验证：** 当前负例和成功模式普通/race真实命中且新增协议断言通过；用受控负向对照证明提前envelope可被新断言拒绝。Reviewer本轮没有写生产mutation/新测试，源码证据是所有commands/handshakes引用仅有声明与采集，未伪装已有失败复现。

### AM-S72-R01-F002 — MEDIUM — accepted后取消用例未保护当前结果的耐久保存

**位置：** `test/integration/article/notice_smtp_test.go:502–511`、`:531`、`:550–561`。依据 r18当前结果受控保存合同、r15 A-NOT-09g/h及§7.6第三层数据符合性；本场景应区分已accepted与DATA中取消。

**实际问题：** after_accepted分支不要求真实m.Send返回nil便关闭accepted屏障；退出时直接丢弃`<-out`错误。共用helper只核Wait/Done、Send一次、下一recipient未claim、pool无占用，没有核首个recipient accepted/字段清理或accepted审计耐久。

**触发/影响：** 一个回归若在m.Send已成功后因work取消直接返回context.Canceled，跳过sharedCtx保存，可能留下processing且没有accepted审计；若worker未把纯取消留为Err，现有Wait/Done、下一attempt和pool检查仍能通过，丢弃out错误也无法指出它。场景名不能证明“accepted后正确保存”。

**最小方向：** 先确认m.Send成功才进入accepted屏障；消费Reconcile返回值并按场景核期望。after_accepted回读首行accepted/attempt1、accepted_at非空、owner/lease/next/bucket清理，以及唯一受理审计的绑定/trace等必要字段；DATA中取消则核预期unknown/retry与风险审计。保留零后继claim/Send、共享截止和join/连接断言，不要求扩成部署Host。

**验证：** 新断言在真实PG/loopback SMTP普通/race通过；受控遗漏当前结果保存的负向对照必须失败。本轮独立回读确认当前生产实现实际正确（见下），因此没有把现有生产代码判为错误；finding要求把这项已观察事实固化为测试保护。

## 实际验证与证据边界

独立解析作者原始JSON，非只信handoff/coverage摘要；与source映射核所有99个leaf。包级结果和非零命中如下：

| 原始日志 | 实际解析 | SHA256 |
|---|---|---|
| ordinary-final-r01.jsonl | exit0；3top/112run-pass nodes/99leaf；0fail/skip；package pass 140.047s | `77F90F7524D6A8D76C85F92FBAE7E0DEE4A491D2B9BC9356D882EE21C4079B44` |
| race-final-r01.jsonl | exit0；3top/112run-pass nodes/99leaf；0fail/skip；package pass 145.822s | `875BEBDA83F34C083EBBDC4DA1F2E6863B4156DE9200FA7F218EB285B321E64F` |
| core-negative-final-r02.jsonl | exit1；1top/7run-fail nodes/5leaf；旧core overlay负向对照 | `3E1DBCB1AC7C77168F9EDF3629F1AC6F47B6D962060F1107D79D454BCD7BE489` |

negative五leaf是4个bigint绑定及LateRealResult；固定测试配旧core已知缺陷的行为失败，**不是当前生产实现FAIL**。初始late夹具panic仅属保留中间历史，不代替最终日志。coverage索引虽有A-NOT-01～09映射，其映射/计数不能消除F001/F002的语义断言缺口。

已读取并核Lead Manifest七项的实际命令、exit0和日志hash：gofmt -l .零输出→go mod verify→go vet ./...→go build ./...→contract→race→unit。普通Manifest不覆盖integration tag；本报告没有把它冒充通知专项或Reviewer重跑。Lead的196个最终schema全存在和11个样本回读来自Lead记录，不称本角色执行；331历史schema清单也不等于本轮新建数量。

Reviewer只执行一次针对关键疑点的窄组，User-scope DSN仅注入进程，不在命令行/输出；仅本包新fixture、随机schema完整迁移，全部保留：

```text
go test -tags=integration ./test/integration/article -run '^TestNotice(DispatcherPostgres|SMTP|Crash)$/(BigIntCommitUnknown|LateRealResult|WrongHostname_max_3|InitialCancellationSharedDeadline|AcceptedBeforeSave_max_1)$' -count=1 -timeout=90s -json
```

结果exit0，3top/14run-pass nodes/9leaf，0fail/skip；新增9个retained schema，执行前后12hash相同，stderr为空。没有重复两套2.5分钟全量；此窄组通过也不是上述断言盲区的失败复现。

又对本轮during_data/after_accepted两个schema执行psql只读事务回读：during_data为retry_wait/attempt1/smtp_outcome_unknown，retry_scheduled＋duplicate_risk；after_accepted为accepted/attempt1/accepted_at存在、owner/lease清空，一条accepted审计；另一recipient均pending/attempt0。由此确认当前生产结果正确。首次补充psql连接方式未成功（exit2，无数据结果）原日志保留；修正为原DSN各连接参数的进程环境注入后exit0，未改变目标或安全策略。检查本角色raw日志不含完整DSN或password。

独立日志目录：[s72-review-r01](C:/Users/J00010385/AppData/Local/Temp/han-s71-1a90c49b44f5450a82add4222dfd1672/s72-review-r01)。`targeted.jsonl` SHA256 `52B35DC95CC05F87B7B1DF8E7CDBF537B2CE119197A6B2AB0B6164B621EF4691`；`targeted.result.json` SHA256 `B5DE6462C33245777662C4BF7BF48603B1BCF13CE64A76F2E17BC7E39C0CE235`（含精确命令、前后12hash及9schema）；`readback-corrected.stdout` SHA256 `F8C1AC85132EAF0197B7FED8C2FA95267B47E1832C7F430F945A9B36EE4A8338`。stderr空文件hash为标准空SHA；首次失败文件另留原名readback.stdout/stderr，未覆盖。

长期副本由Lead保管：[ordinary](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/ordinary-final-r01.jsonl)、[race](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/race-final-r01.jsonl)、[旧core负向对照](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/core-negative-final-r02.jsonl)、[coverage](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/coverage-s72.json)、[Lead Manifest](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/lead-manifest-r01-results.json)、[Lead PG回读](D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s7.2/lead-postgres-readback.json)。53份复制一致性由Lead核证；本角色解析原始temp证据，未重复全量copy核验或删除原件。

## 交回与限制

两项finding已在工作中提前交回，当前均OPEN；最小整改均可在原Ohm三文件内完成，无设计/API/生产九源修改要求。未将函数物理行数、框架选择或历史soft确认作为缺陷。无新增方向决定；现阶段不以总PASS数替代缺失保护。

本轮是真实本地PG/TLS/child专项的有界独立核查，不是生产启动、真实Host或部署E2E；没有外发SMTP、旧所有integration执行、DROP schema/cleanup、新依赖或生产代码变更。root信号协调器/production factory完整接入仍引用S7.1独立证据，S7.2直接调用dispatcher的取消不冒充Host信号执行。原S7.1 JointR02及Core关闭结论保持原范围。

交新固定测试hash及针对F001/F002的回归/必要负向证据后复审。本报告交付前再核12hash和旧报告不变；仅新增本报告及上述授权Temp日志，交付真实报告SHA256后停写。
