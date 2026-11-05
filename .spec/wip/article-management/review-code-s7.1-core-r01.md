# S7.1 Article3 独立 Code Review R01

**CodeReviewGate：CHANGES REQUIRED（仅固定 Article3）。Findings：2 MEDIUM，0 HIGH/CRITICAL。**

评审角色：Reviewer；原作者：Nash。两轮独立核查已完成。结论不关闭父 S7.1，不替代九文件 joint Review、真实 PostgreSQL/SMTP 集成、Host 或 E2E。设计侧原 F001 CLOSED 不等于本次实现通过。

## 固定对象与边界

工作区 `D:\AID\VSCode\Codex\Han`，沿用 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。对象是明确冻结的未提交文件，不将 HEAD 版本混作本次实现。唯一写入是本报告；原报告及用户已有改动保留。

| 冻结对象 | SHA256 |
|---|---|
| `internal/article/notice_dispatcher.go`，1044 行，新文件 | `8F08F7E3A7430E81859D8B6B8F73A686729AC9BE8E16A6282BD6457A3BCBE503` |
| `internal/article/repository.go`，776 行 | `80529712F7D62ACD4A5A976621FF1E7AB11069D11CDE2D73B5050CDCA58758F8` |
| `internal/article/notice_dispatcher_test.go`，1269 行，新文件 | `D662ACCF349AFACCF8A56833E74042F7ED32426FE9CA7917AFAE12532DB8E0F1` |
| design r18 | `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3` |
| test-plan r15 | `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` |

Repository before-image 位于 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672\internal__article__repository.go`，与 `core-repository-before.txt` 同 SHA256 `E0C2430A981A5F90E84F197159158CC4A218A4FB399C74E90C203B19AC0FE078`。实际比较确认原 **21131 bytes 前缀逐字节一致**；新增两文件没有前像。`core-final.diff` SHA256 `29DA32D591D56AC9C3FD9FA1D0E0BB8E7EE756462C0239952314950E6177C893`，三份 `core-final-*.txt` 源码快照的解码正文与固定对象一致。

沿用已取得的 Reviewer Role、add-code-review、当前 add-delivery-workflow 及本对象适用的 checklist/abstraction/task-execution、安全、编码、可观测性与 go-service 规则，续接原 Workflow，不重新 Route。依据是固定 r18/r15、原 Design R02/TestPlan R01、原通知业务边界；不机械重开纯 pending source-review metadata 或 S6.2 受保护正文。

原不可变报告：Design R02 SHA256 `115289B8216BD5CB0AD49CB9B82B748E84555FD58C307AAFA272213F387B13E7`；TestPlan R01 SHA256 `29BC5E15647A2BB0C853A59B205080DE4114FAD2134F8D82F4FDD7111474E1EE`；SMTP R01 SHA256 `4762EFC4ED1F312835B70A8D4B8C3BE2BB4C73982DB069755D414FEC1BD7D018`。这些是各自范围的结论，不相加为本次或父任务 PASS。

按 Lead 最新调度，Tasks/Plan 进入 Planner 写窗口，**不作为本报告冻结评审对象**；root4 若开始写也不读取为固定对象。本次未审不存在或动态的 production wiring，不评价其实施完成度。以下修复建议无需改变冻结公共 API；若整改确须改变 API，原作者应交 Lead 协调对应 root 接入。

## 第一轮：对象自身完整检查

逐段读完三文件全部正文，核对实际差异与旧 Repository 全部保留前缀。检查配置与依赖验证、六个方法及 BeginShutdown、active/running/stopping/Done 协调、事务预算与连接释放、DB 权威时间、claim/preflight/result/maintenance 状态机、完整 fencing、SMTP 分类、邮件内容、审计与 unknown 回读；逐项检查测试实现而非仅看测试名。

具体核过：正整数微秒及预算加乘溢出；typed-nil 依赖；READ COMMITTED 短事务；claim 同 tx Gate；锁后 fresh DB clock 与 UTC 微秒；attempt/max 的 int64 所有权；SKIP LOCKED 和精确前像；同 ID/同微秒 lease 的 attempt ABA；最后 processing 崩溃及 Gate deferred 的耗尽；饱和退避；两个 recipient 独立；每 claim 一次 Send；preflight 释放后网络；accepted 与审计同 tx；不确定 claim 不发；结果未知仅有界回读；shutdown 使用同一个共享截止、初始 Reconcile 可 join、Wait timeout 取消共享保存并等实际 Done。

第一轮发现 owner-lost 分支仅日志退出，并对审计绑定比较的运行时类型提出疑点；两项均在第二轮沿真实消费者核实后列为 finding，未直接根据 Lead 提示或 fixture PASS 下结论。

## 第二轮：稳定消费者与故障交错

沿实际冻结 SMTP、privacy、audit、model、migration 及 UUID/trace 约束核类型、签名、数据所有权和错误链。重点检查旧 owner 结果晚于回收/新终态的时序，以及真实 AuditStore 的 JSON 往返而非内存 Event 直返。未读取动态 root 作固定依据。

| 稳定消费者 | SHA256 |
|---|---|
| `internal/mailer/smtp.go` | `28237089CE76B353CC01D36847378CF4775AA9281210175AAC59905DCC192853` |
| `internal/mailer/smtp_test.go` | `9833F766F24CEA8067D9A2C46BB13C01DF2568DE5A7EA998675318EDCA2B964E` |
| `internal/privacy/store.go` | `6E555D74E5210EA20CE0C33DB08DA40CD6BA4B7D1549B9A8031E0F4D82EEDD9D` |
| `internal/audit/store.go` | `091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F` |
| `internal/article/model.go` | `3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4` |
| `migrations/0003_article.sql` | `FCAECE696DCE29D2F8D252EA36C4550F0C1736347E921F40703540B99E100D4A` |

确认代码保留以下边界：privacy 可使用 caller tx；没有为网络持有 claim/preflight tx；结果保存不重新受 Gate 阻断；SMTP 通过 Is/As、阶段与数字码分类，未使用 error 文本匹配；MAIL 配置类与 RCPT/DATA 明确拒绝分流，Auth/greeting/TLS 未伪装 typed rejection；DATA accepted 后 QUIT 故障遵循 frozen Mailer 的成功语义。邮件为固定最小内容、transfer purpose、无链接/正文/标题查询；日志不输出 SMTP 原文或邮箱。数据库隔离、真实池行为及网络故障的运行效果仍待后续集成验证。

## Findings

### AM-S71-CORE-R01-F001 — MEDIUM — 有效旧 owner 的迟到结果缺少独立风险审计

**位置：** `internal/article/notice_dispatcher.go:844`（saveResult 的 lost 分支）；关联 `:718`、`:492`、`:585`、`:931`，测试 `notice_dispatcher_test.go:257`。

**依据：** r18 §3.4.8 第2193行要求 owner lost 不改 recipient，并记录竞争告警/受控独立审计；第2226行故障表进一步明确“旧 owner 的结果到达”应有独立风险审计，DB 不可用才只留低敏告警。第2228行明确日志不替代审计。这项义务不同于第2215行的过期回收风险。

**触发场景：** A 持真实有效 claim 进行一次 Send，暂停越过 lease；B 回收该 recipient，记录回收风险，并完成新结果成为终态；A 随后拿到 SMTP 结果进入 saveResult，此时 DB 可用。锁和 fresh time 均成功，但 `noticeOwns` 为 false，代码只 `observe("lease_competition")`，返回 `false,nil`。deliver 不为 applied=false 补风险事件。

**实际后果与边界：** fencing 正确保护了 B 的状态，没有覆盖新 owner 或重复改终态。但 A 迟到结果这一新发生事实没有耐久风险记录。claimDue 的回收审计记的是当时上一 attempt 尚未知；maintain 只处理耗尽行。两者不能记录后来到达的结果，且 B 已终态时不会再进入回收/维护。这里不是声称所有重复风险都未审计，而是明确缺失设计单列的迟到结果独立审计。

**最小建议：** 由 Nash 对真实内部旧 claim 的 lost-result 路径补受控、有界、低敏的独立风险审计；结束占用的结果事务后再进行独立记录，遵守现有 Actor/Target/metadata 白名单及共享 shutdown 截止。不得修改当前 recipient、重新 Send 或延长共享截止。DB 不可用时保留低敏告警，不宣称审计已耐久；无需改变冻结公共 API 或设计。

**证据与复审验证：** 当前 `:844–846` 无 Audit.Append/commit；`:492–496`、`:585–594` 的风险范围已逐项核实。现测试在 owner/attempt/lease mismatch 时要求 `len(m.events)==0`，不能证明设计要求已实现。补真实 A claim→过期→B claim/终态→A 结果到达的交错：DB 正常时新增受控风险事件，B 的全部字段及终态审计不变，Send 次数不增加；DB/审计故障时只有允许的低敏回退，且不误报审计成功。不要把错 PK 等伪造 fixture 不加甄别地一概登记为真实迟到结果。

### AM-S71-CORE-R01-F002 — MEDIUM — 审计 JSON 数值往返使合法结果回读误判 unknown

**位置：** `internal/article/notice_dispatcher.go:1038`（noticeAuditBound）；关联 `:977`、`:1005`；真实消费者 `internal/audit/store.go:151`、`:162`；fixture `notice_dispatcher_test.go:938`、`:950`。

**依据：** r18 第2224行要求 commit outcome unknown 后按本次 EventID 核状态/attempt/审计绑定，全部匹配才 already_applied。r18 第820行、model 和现役 schema 允许 max_attempts 为完整正 signed bigint；不应因私有审计表示变化误判本来完全一致的绑定。

**触发场景：** recipient 合法快照 `max_attempts=9007199254740993`、当前 attempt=1；SMTP accepted 后结果和审计已经提交，仅 commit ACK 丢失。noticeAudit 将 max_attempts 作为 int64 写入 map。真实 AuditStore.Append 以 JSON 存储，Get 使用 `json.Unmarshal` 解码到 `map[string]any`，JSON 数字因此成为 float64；上述整数不能被 float64 精确表示。noticeAuditBound 对读回的 float64 和原 int64 使用 `fmt.Sprint` 比字符串，必然不相等。即便未发生精度损失，足够大的 float64 的 `%v`/`%g` 表示也可能与 int64 的十进制表示不同。

**实际后果与边界：** 真实状态与本次事件已耐久且正确绑定，却返回 result_commit_unknown、产生误告警并结束本轮，无法走 already_applied。accepted 状态仍然受保护；本 finding 不声称该路径会直接重发或破坏数据。大 max 快照在第一次尝试即可触发，不需要实际执行海量重试。

**最小建议：** 由 Nash 使通知私有审计计数的保存和比较具有精确、稳定表示，例如在该私有 metadata 中保存规范十进制字符串，并按相同表示核绑定；保留 max/attempt 的原 int64 业务字段与 schema。不要通过截断、浮点近似等方式放宽绑定，也无需为本缺陷修改公共 AuditStore、配置范围或冻结 API。

**证据与复审验证：** 已沿真实 AuditStore 的 Marshal→DB metadata→Unmarshal→noticeAuditBound 核完链路，并用本地 `go doc encoding/json.Unmarshal` / `go doc fmt` 核解码及格式规则。这是源码/类型推导，未声称已运行 PG 故障复现。现 fixture Append/Get 直接保留/返回同一个 Event 的 int64 metadata，不经历 JSON 往返；普通 unknown 与 target mismatch 测试因此不会命中该缺口。补真实 JSON 往返的绑定测试，覆盖合法大整数（含 bigint 上界）及常规值；durable commit ACK 丢失应识别 already_applied，不重复 Send/Append；实际错 target/owner/attempt/lease/trace 等绑定仍须拒绝。后续 PG 专项再验证真实存储与 ACK 故障链。

## 验证证据与限制

实际独立解析 temp 根的原始 JSON 日志（UTF-16 BOM），没有仅采用作者摘要。最终两日志均无 JSON 解析错误，run/pass 命中不为零；退出文件另核为作者记录 exit0。

| 作者证据 | 独立解析结果 | SHA256 |
|---|---|---|
| `core-final-green.json` | 33 top / 90 run、pass nodes；0 fail、0 skip；package pass | `1CEED5614619AF04407B9717B9A5CE26CE6CE4FFC9B6D1D3BC533DC6931DF949` |
| `core-final-race.json` | 33 top / 90 run、pass nodes；0 fail、0 skip；package pass | `D1B70554062BBDAE6C7037F38197E3B9D6E1202DC5CFB3563862354B8512C48C` |
| `core-article-regression.json` | 175 top / 586 run nodes；584 pass、0 fail、2 既有 platform skip | `DB44C6EBB8A7C47DD0A15E2BD76BE9561EB04481BE9A2BEBAF924BD9B70BC23A` |
| `core-red.json` | 3 top / 10 run nodes 均失败；另 package fail；0 skip，行为失败而非编译失败 | `727EB12EAD7FF64C6F108B682AE017AF74D475D374F825994BC54F4599454195` |

第三行是 **最后 Wait timeout 共享保存修正之前** 的回归，不是最终修订全量回归。两项 skip 为 `TestFileBackupInventoryReader/symlink_rejected`、`world_writable_rejected`。最终 vet/build 的作者退出摘要均为0，输出文件为空；未冒称 Reviewer 重跑了这些检查。

`core-final-evidence.json` SHA256 `6E37371D23A5451E0A617393094EBD0AC87F30CF7DE1C34DF2E30E12FD9C248C`，明确承认 RED stub 未在替换前及时快照。RED 日志与 RED 测试快照保留，后者 SHA256 `F06E13092B984347387786F4BF56E34A41E797699F44666E0839694594536332`；本报告不据此宣称具备完整 RED 实现源码或补造其可复现性。

Reviewer 为上述具体 concerns 实际独立执行一次窄测试，未写新测试或临时源码：

```text
go test ./internal/article -run '^(TestNoticeDispatcherFencing_AnyOwnershipMismatch_Lost|TestNoticeDispatcherResult_CommitUnknown_VerifiesBindingWithoutResend|TestTransferNoticeAudit_BindingMismatch_NotAlreadyApplied|TestNoticeDispatcherWait_TimeoutDuringSharedSave_CancelsSettlement)$' -count=1 -json
```

结果 exit0，4 top / 11 run、pass nodes，0 fail/skip，package pass，stderr 为空。它验证现有 fencing/unknown/shutdown 测试仍通过，**不是上述两个缺口的失败复现**；这些测试的盲区已在 findings 中具体说明。未无理由重跑全量或 race。

当前测试 fixture 虽调用真实 Repository 方法，但以内存 Tx 模拟 SQL 结果，不能证明 PG 语法、锁隔离、多连接、池释放和真实 commit unknown。局部 SMTP fixture/冻结 SMTP PASS 也不能替代真实 PG+TLS 联合验证。本次没有连接 PG、读取或打印 DSN、外发邮件、连接生产 SMTP、运行新集成/E2E 或启动 Host。未使用 identity 旧 PASS 充当通知证据。

## 交回与停止

两项 finding 均 OPEN，交原作者 Nash 最小整改，再由同一独立 Reviewer 对新固定 hashes、原 findings 及受影响回归复审；不由 Reviewer 修代码。不要求为这两项改设计或扩大 API；方向若变化先交 Lead。

本次仅给 Article3 局部 CHANGES REQUIRED；SMTP2 原局部结论保持其原范围，root4 不在固定对象内。父 S7.1 继续 `[-]`；九文件稳定且各 finding 闭环后仍需独立 joint Review，不能用局部 PASS 相加。

本报告交付后停止写入。报告自身实际 SHA256 由交付消息给出，不在文件内自引用。
