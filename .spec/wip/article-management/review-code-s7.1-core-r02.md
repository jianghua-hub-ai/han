# S7.1 Article3 独立 Code Review R02

**CodeReviewGate：PASS（仅本报告固定 Core3 的原 finding 修复及实际新增改动）。**

**`AM-S71-CORE-R01-F001`：CLOSED；`AM-S71-CORE-R01-F002`：CLOSED。新增 findings：0。**

本结论为独立代码局部复审，不是九文件 joint Review、父 S7.1 完成、真实 PostgreSQL/SMTP 联合验证、Host、部署或 E2E PASS。父 S7.1 继续 `[-]`。原 R01 保持不可变，通过本报告追加修复结论。

## 对象、依据与写入边界

固定工作区 `D:\AID\VSCode\Codex\Han`，沿用原 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 的未提交工作对象。唯一授权新写本报告；未修改源码、原报告、design/test-plan、Tasks/Plan、Rule、Git 或临时证据，未清理或委派。

| R02 固定源码 | SHA256 |
|---|---|
| `internal/article/notice_dispatcher.go`，1090 行 | `C49CF9D7C78E5DC99E2177B6F760BE5B191D2120F3FC47DBBE8A7BEF93BFFB97` |
| `internal/article/repository.go`，776 行，R01 后未变 | `80529712F7D62ACD4A5A976621FF1E7AB11069D11CDE2D73B5050CDCA58758F8` |
| `internal/article/notice_dispatcher_test.go`，1537 行 | `7761947F288890CCC66125E7C5B0DF153F3568C72C7EF05ED9291A702F9D8ADD` |

实际核对上述 hash，并确认 temp 根三份 `core-r02-final-*.txt` 与工作区固定源码逐字节相同。temp 根为 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672`。

`core-r02-before-*.txt` 分别与 R01 dispatcher `8F08F7E3A7430E81859D8B6B8F73A686729AC9BE8E16A6282BD6457A3BCBE503`、Repository 上述不变 hash、test `D662ACCF349AFACCF8A56833E74042F7ED32426FE9CA7917AFAE12532DB8E0F1` 一致。按实际正文逐段比较，dispatcher 新增52行/删除6行，test 新增268行/删除0行，Repository 零差异；旧 Repository 的21131 bytes 前缀仍 exact。未把作者 diff 摘要当作真实差异的替代。

沿用同版 Reviewer Role、add-code-review、当前 add-delivery-workflow 及本对象适用的 checklist、abstraction、安全、编码、可观测性、task-execution 和 go-service 规则；不重新 Route 或全库预读。设计依据仍为 r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`；测试规划仍为 r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`。重点复核 r18 §3.4.8 的 owner lost、独立风险审计、未知提交回读，以及 bigint 和共享 shutdown 截止合同。

本次全文重读原 Core R01，SHA256 `489851117E270D8B0B711D0D01E50B1D8FB73252181D3B772B4E7190514E2B0A`。原 Design R02、TestPlan R01、SMTP R01 仅在各自已核范围内有效，不推导本次代码或运行验证 PASS。Tasks/Plan 和 Lorentz 动态 root4 不作为冻结评审输入，未读取动态 root 代码给 Gate。

## 两轮实际检查

第一轮从固定对象自身检查：完整重读 R02 dispatcher 1090行；逐项核全部 R01→R02 增量、268行测试新增及 fixture 改动，比较 before/RED/final 三阶段快照。追踪 claim 成功提交后 issuance、实际 Send 标记、结果 lost、事务退出、独立风险审计、故障回退、计数保存和严格绑定比较。原 Repository 全文已在 R01 读取；本次验证整文件 hash 不变，并重读通知 Repository 方法/SQL 第611～776行，不假称再次通读未变的全部旧正文。

第二轮从稳定消费者和交错检查：重读真实 AuditStore 的 Append/Get/JSON 解码链，沿 Repository 锁/完整 fence、model int64、schema JSONB/recipient 状态约束及固定 SMTP/privacy 合同核修复。重走 A 发送暂停→B 回收且终态→A 迟到结果、独立审计 begin/append/commit ACK 故障、共享截止取消，以及 JSON 往返后的正反绑定。核 fixture 的 JSON 边界是否确实执行，未将仅内存 Event 直返或日志节点数当作缺陷已修复的证明。

| 稳定依据/消费者 | 当前 SHA256 |
|---|---|
| `internal/audit/store.go` | `091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F` |
| `internal/article/model.go` | `3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4` |
| `internal/privacy/store.go` | `6E555D74E5210EA20CE0C33DB08DA40CD6BA4B7D1549B9A8031E0F4D82EEDD9D` |
| `internal/mailer/smtp.go` | `28237089CE76B353CC01D36847378CF4775AA9281210175AAC59905DCC192853` |
| `internal/mailer/smtp_test.go` | `9833F766F24CEA8067D9A2C46BB13C01DF2568DE5A7EA998675318EDCA2B964E` |
| `migrations/0001_shared.sql` | `CEE72F1470B49185B3B33C7655F792ADFFCBC1E030AF6669E0902775104B7A14` |
| `migrations/0003_article.sql` | `FCAECE696DCE29D2F8D252EA36C4550F0C1736347E921F40703540B99E100D4A` |

未发现本增量改变公共依赖接口、导出方法签名、HTTP、业务 int64、共享 AuditStore、schema 或 S6.2 原 Repository 正文。新增 `issued`/`attempted` 是私有执行事实，不授予租约发送权，不绕过原完整 fencing。未发现修复增加第二次 Send、在 SMTP 期间占用 DB tx/conn、重置 attempt/max，或修改新 owner/终态的路径。

## 原 finding 复审

### AM-S71-CORE-R01-F001 — 原 MEDIUM — CLOSED

原缺陷：真实旧 owner 的发送结果到达时，saveResult 只记录 lease_competition 日志，未履行 r18 第2226行的独立风险审计义务；之前回收旧 attempt 的风险审计不能替代后来到达的结果。

**修复位置及判断：**

- `notice_dispatcher.go:508` 仅在 claim commit 成功后保存私有 issuance 原像；`:712` 仅在实际调用 Mailer.Send 前标记 attempted，`:717` 将事实带入结果。未知 claim commit 不产生此已发行 handle，也不获得 Send 权。
- `:846`、`:853` 将未找到/已失 owner 的结果交 lostNoticeResult；`:861` 先 Rollback 结束原结果事务，`:864` 取消其操作预算，再调用独立审计。Rollback 异常时返回固定低敏失败类别，不继续占用原事务开第二连接。
- `:865–871` 只接受实际 attempted 且与内部 issuance 一致的 handle；错 PK/kind/attempt、无 issuance 或未发送结果不伪造耐久风险。
- `:876–892` 使用新的有界事务和 fresh DB time，仅 append `article.transfer_notice.duplicate_risk` / outcome=risk 并 commit，不调用 recipient UPDATE 或 Send。Actor/Target/trace、owner hash 和 metadata 沿现有通知审计构造；不记录邮箱、邮件或 SMTP 原文。
- begin/clock/append/commit 失败统一返回 `late_result_risk_unpersisted`；经 roundFailure 只输出固定 bucket 和 trace。提交 ACK 不确定时不宣称已耐久、不盲目再 append 或 Send。外层 error 的可打印字符串不暴露 cause 原文；停机时错误链仍保留取消事实供协调器接收。
- 新事务由原结果 ctx 派生，受 DBOperationBudget 及已绑定 sharedCtx 共同限制；没有重新创建共享截止。原 settlementCancel/Wait 仍覆盖这条保存链，active 直到调用实际退出才解除，Done 未被提前关闭。

**验证证据：** `notice_dispatcher_test.go:1043` 的真实 Reconcile/claim/Send 交错，在 B 已 accepted 后让 A 返回；断言独立审计是原结果事务之后的第二个 Begin、当时 active tx 为1、recipient 全字段不变、仅增加一条 risk、无额外 Send。`:1094` 覆盖错 handle/未发送；`:1127` 覆盖独立 begin/append、ACK rollback/durable，并核低敏 fallback；`:1175` 覆盖共享截止取消、回滚及实际返回。读代码同时确认过期但未回收的真实 attempted handle 也走相同独立路径，未将这一点冒称为新增 PG 执行用例。

原 R01 要求的“有真实旧 claim、DB 正常耐久风险、DB 故障低敏回退、保护新状态、不重发、共享截止”已在本地代码及专项测试层面满足，故 CLOSED。原先回收/耗尽原子风险路径保留；此关闭不解除 PG 多连接/崩溃集成验证。

### AM-S71-CORE-R01-F002 — 原 MEDIUM — CLOSED

原缺陷：真实 AuditStore 将 JSON 数字解码为 float64，noticeAuditBound 用 fmt.Sprint 比较 int64 与回读值，导致合法大计数已提交结果被误报 unknown。

**修复位置及判断：** `notice_dispatcher.go:1021` 对私有 metadata 的 attempt_count/max_attempts 使用 `strconv.FormatInt(...,10)`；`:1025` 对 prior_attempt 同样保存精确十进制文本。业务字段仍为 int64，未缩小合法 bigint 范围，也未修改共享 AuditStore 或 schema。真实 JSONB 存取中的字符串不会转为 float64。

`:1082–1087` 要求全部受核 metadata 实际类型为 string 且逐值相等，去掉 fmt.Sprint 宽松类型格式比较；不存在“把浮点近似再转整数”的放宽。原 event ID、system actor、target/action/outcome、trace、OccurredAt 以及 recipient 完整结果比较仍保留。私有类型漂移或实际绑定不同继续返回 unknown，不重复 Send/Append。

**验证证据：** fixture `notice_dispatcher_test.go:946–999` 的 Append/Get 均执行 json.Marshal/Unmarshal，不再保留 int64 map 原样直返。`:1070` 覆盖 `(attempt,max)` 为 `(1,3)`、`(1,9007199254740993)`、`(9007199254740993,MaxInt64)`、`(MaxInt64,MaxInt64)` 的 durable commit ACK 丢失；断言 accepted、一次 Send、一次审计、无错误 unknown 日志及精确计数字符串。`:1203` 分别破坏 target/trace/owner/attempt/max/lease/kind 或注入 numeric_type，仍要求 result_commit_unknown，且不重发/重复审计。真实 AuditStore 的第61行 JSON 编码及第151/162行 JSON 解码未变，与新 fixture 表示相符。

原误判原因已移除且严格绑定未被放宽，故 CLOSED。上述为实际 JSON 往返的本地验证，不声称已执行真实 PG commit ACK 故障。

## RED/GREEN 与本次独立验证

实际读取和独立解析 temp 根原始日志；UTF-16 BOM 解码后，各日志均为有效 JSON 行、命中非零，与作者摘要一致。另读取各 exit 文件、checks 摘要及空 stderr/vet/build 输出，未仅转述 evidence.json。

| 原始日志 | 实际结果 | SHA256 |
|---|---|---|
| `core-r02-red.json` | exit记录1；2 top / 6 run nodes，1 pass、5 fail test nodes；另 package fail；0 skip | `F0E0AC586AF56A8AAB971950137C44073D8E28E739C20C4F4582B34D395ADFB4` |
| `core-r02-findings-green.json` | exit记录0；6 top / 27 run/pass nodes；0 fail/skip；package pass | `F1F579810880F897F57A3105E98FF4C400B364348B2533CEC3D32EFBF96F01C7` |
| `core-r02-green.json` | exit记录0；39 top / 117 run/pass nodes；0 fail/skip；package pass | `C475C8B7B5B3A016E56BA3B26E1563FBDA3DB2125B5A65086D8D27A8F8FF39DA` |
| `core-r02-race.json` | exit记录0；39 top / 117 run/pass nodes；0 fail/skip；package pass | `34F89E44995684907D3B91330D55BD317A856E02BC3C685FF8EEBEEACC1357CA` |
| `core-r02-article-regression.json` | exit记录0；182 top / 614 run nodes，612 pass、0 fail、2 skip；package pass | `8396FB4BCBF2503D7A74F8D1D681D18871A464A07F786726A266BC44414D4ACE` |

RED 真实失败为 `TestNoticeDispatcherR02F001_LateRealResult_IndependentRiskAudit`（缺独立耐久风险）和 `TestNoticeDispatcherR02F002_JSONRoundTripBigints_AlreadyApplied/1`、`/2`、`/3`（JSON 往返后被误报 unknown），加后者父节点共5个失败 test nodes；常规 `/0` 通过。日志是行为 RED，不是编译失败。

本次三份 RED source 快照齐全：dispatcher/Repository 与 R01 固定源码一致，RED test SHA256 `7F8DB93EDDC79034BB722C7DA63C35AFAFF6C0CCF0B992A765E67D91AA8E7B24`。已比较 RED test 与 before 的真实差异，其内容包含上述两项行为用例及 JSON fixture。`core-r02-red-source-hashes.json` SHA256 `3D64BEA6183CF7069F56D6D89762058D1E8F0CEE425A76A3AE28CBC5865396DB` 与实际三快照 hash 相符。本次不改回旧源码重跑 RED；也不以本次完整快照补称原 R01 之前那次 stub 缺失已经被历史补齐。

本次最终 regression 属于 R02 证据集；两项 skip 仍为 `TestFileBackupInventoryReader/symlink_rejected`、`world_writable_rejected`。没有借补正前 `core-article-regression.json` 的旧584 pass 代替本次612 pass。作者 `go vet ./internal/article`、`go build ./internal/article` 退出摘要均0，输出为空；这是已核作者证据，不冒称 Reviewer 重跑全量检查。

Reviewer 为原两 finding 及其新增故障/取消分支，实际独立执行一次窄复核：

```text
go test ./internal/article -run '^TestNoticeDispatcherR02F00[12]_' -count=1 -json
```

实际 exit0，6 top / 27 run/pass nodes，0 fail/skip，package pass（0.247s），stderr 为空，无 JSON 解析错误。另对固定三源码执行 `gofmt -l`，exit0、无输出。测试输出仅工具捕获，未新增 temp 日志或临时源码；未无故追加全量/race 测试。

证据索引：`core-r02.diff` SHA256 `C7315B99E0F5A3BD8918CC9A7CC806F91EC5351BC7B0176850693108D4AEF306`；`core-r02-evidence.json` SHA256 `7FCDBFBBB2D2B72B16DAEF3D5CAC14EB873D604BD3660516023A1511D0D00339`。后者是作者索引，不代替本报告的源码与原始日志核查。

## 限制、交回与停写

新增具体缺陷0；无须改变冻结公共 API、设计或共享存储合同。没有新增方向决策或需 Lead 接受的剩余 finding。结论仅绑定上述固定 hash；后续对 issuance、lost-result、审计表示/绑定、共享截止或相关依赖的实质修改须按影响复审。

本地 fixture 调用真实 Repository 方法并执行真实 JSON 编解码，但仍模拟 Tx/SQL/时钟/commit ACK；不能证明 PG 语法、多连接锁隔离、实际连接池释放、真实崩溃或网络故障。未连接 PG、读取/打印 DSN、外发邮件、连接外部 SMTP、执行新 PG/TLS 集成、启动 Host 或运行 E2E。未用 SMTP/identity 历史 PASS 替代通知联合验证。

root4 仍是 Lorentz 动态写入窗口，本次未审其 wiring、首次 signal 协调或依赖关闭顺序的实际实现。Core 局部 PASS 不证明 root 接入正确；SMTP2/Core 局部结论不能相加替代最终九文件独立 joint Review。父 S7.1 保持 `[-]`，后续等待 root4 交付、九文件稳定及 joint Review。

原 Core R01 及其他旧报告、design/test-plan 和稳定消费者 hash 已复核保持原值。本次仅新增本 R02；交付实际报告 SHA256 后停止写入。
