# Han 通知 test-plan r15 独立评审 R01

**TestPlanGate：PASS（固定 r15 的 Story7/S7.1/S7.2 通知增量规划充分性）。Findings：0。**

这是规划评审结论，不是通知测试执行、任务关闭、代码评审、部署、production readiness 或 E2E PASS。原 Design R01/R02 不可变，本报告不修改其结论或内容。

## 对象、修订与边界

- objectType：design（测试规划子对象）；固定作者 Ohm 已停写。本次由独立 Reviewer 两轮审查，未参与规划编写，未再委派。
- 工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，现场核对一致；真实未提交内容以本文hash及读取范围固定。
- 被审 `.add/project/knowledge/design-workspace/article-management/test-plan.md`，revision `article-management-test-plan-ir01-r15`，SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`。
- r14 before-image：`C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672\.add__project__knowledge__design-workspace__article-management__test-plan.md`，SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C`。
- 唯一设计依据：fixed design r18，SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`，重点§3.3.1/§3.3.8/§3.4.8/§6.1/§8.5；沿用本会话已核requirements US-IDENTITY-03.9、Story7、accepted boundary和适用ADR的业务合同。
- 原 Design R01 SHA256 `1C51D766F9F2B811558F4BE124878875CC5A5FEBCEC49FD5E04B41C875382950`；Design R02 SHA256 `115289B8216BD5CB0AD49CB9B82B748E84555FD58C307AAFA272213F387B13E7`。R02已对同一r18给DesignReviewGate PASS、原F001设计侧CLOSED、0新finding；不据此推导通知已实现或测试通过。
- 唯一可写/实际新增文件为本报告。design、test-plan、Task/Plan、code、Rule、Git及旧报告不改，不清理、不委派、不运行PG/SMTP。
- Ptolemy正在写的Tasks/Plan未读取，也不作为固定对象。九+三文件仅核本测试规划与design候选是否一致，不对实际任务登记、认领、DAG、接收或关闭给Gate。

同一Workflow续接，不Route；复用本会话已完整读取的Reviewer Role、add-code-review、add-delivery-workflow、code-review-checklist、abstraction-principles、design-doc-writing-style、task-execution-workflow、安全红线及有关core/go-service配置、错误/并发/可观测规则。按测试规划对象核风险、可观察期望、证据与交付边界，不把代码编译/执行硬套为本次前置。

## 第一轮：全部差异及验收—风险—证据映射

固定r14→r15的精确逐段比较（SequenceMatcher，autojunk=False）为**新增153行、删除18行**。实际读完全部增量及必要上下文：metadata/§1/§1.1，§4.6父例和三组子例全文，§6通知追溯行，§7.6全文，§9任务建议通知两行和§9.5全文，§10对应顺序、§11通知Gate、§12增量结论。未把文件存在或测试名称当作场景充分性。

| 风险/设计合同 | 计划落点 | 独立判断 |
|---|---|---|
| 双快照/用途/Article及身份所有权 | A-NOT-01/01a，§7.6 | 覆盖地址变化/retained清理/Article删除后仍只读快照；同地址双kind独立、max快照不可变、HTTP pending非实时送达承诺；无链接最小模板见09a |
| 原privacy Gate同tx与新preflight | 02/02a、06/06a～c | 真实caller tx与pool不可见登记对照、now()事务起点、更新登记、坏tx先回滚再新tx、关闭前后及在途窗口均有可观察期望 |
| 原子claim与SMTP期零DB占用 | 03/03a/03b | 多连接/barrier、只锁recipient、SKIP LOCKED首行绕过、逐条claim及0行不授发送权；Send阻塞时限制同pool并独立查询/检查非idle-in-transaction，不靠扩大池掩盖资源泄漏 |
| DB权威微秒/锁后freshclock/预算 | 04a、02c | 锁等待后fresh clock_timestamp、等于即过期、RETURNING authority、DB/审计时间及微秒边界、时间/预算溢出；monotonic已耗时间扣减、预算不足零连接 |
| 最后attempt/耗尽收敛 | 04b～d、06a/06b、07c/07d | 最后有效claim可成功；max=1 Gate deferred先retry_wait再到期维护，Gate始终closed仍终结且零SMTP；过期processing耗尽unknown+风险、未到期不动、终态不重审计、退避安全饱和 |
| claim/result commit unknown与原子审计 | 07a/07b，03a | 实际提交/回滚后丢ACK对照；claim未知零preflight/Send且不盲claim；accepted后UPDATE/audit故障、结果unknown按PK/EventID回读、不再Send、不盲写第二审计；回收审计失败整tx回滚 |
| accepted崩溃与重复风险 | 07/07c/07d | 真实子进程阶段握手/退出，发送前/DATA/受理后/结果提交前及已提交对照；有余次与最后次分开，DB/SMTP计数和耐久风险相互核对，不把terminal说成从未受理 |
| 旧owner/attempt fencing | 08/08a | 不同owner、同workerID旧attempt、无人接管但已过期、错kind/attempt/精确微秒lease逐项不符均lost；不覆盖新owner/accepted，告警不冒充耐久审计 |
| SMTP类型/阶段/数字码 | 05a～g | MAIL4xx/5xx为sender/config并结束本轮；RCPT/DATA4xx退避、明确5xx永久；Auth/greeting/TLS不混入永久recipient桶；sentinel alone/链不一致/未知按unknown+风险；本地校验零连接 |
| DATA accepted/QUIT、原errorchain | 05a/05g | QUIT拒绝/断连/取消不改accepted；受理ACK丢失另判unknown；四明确拒绝点附加Stage仍保留Is/As/textproto cause，身份/reset/resend既有语义回归 |
| 最小披露/安全TLS | 09/09a～c | 两安全模式真实loopback；负例无MAIL/RCPT/DATA；固定模板/purpose仅UUID引用，canary覆盖bucket/Audit/logger参数/可输出error；指标低基数、owner仅hash，无外发或信任弱化 |
| 配置与真实production factory | 02b～d、03c | 六必填键/严格边界/交叉预算/typed nil及DB前零副作用；实际builder/runtime/runComposed/lifecycle链和共享依赖身份、缺worker拒绝；不拿测试自行装配替代生产factory |
| r18原F001启动取消/共享截止 | 09d～h | 初调前已来signal、Send阻塞、accepted未保存、下条未claim、同时就绪均覆盖；唯一signal消费者、BeginShutdown先绑定共享context、停止后零后继、重复signal不重置、一次fenced保存及unknown回读、不重启已开始保存 |
| 实际join/错误与退出 | 09d～h | 未Start但活动的初次调用纳入Done；共享截止与DB预算取较早、到期非零且实际退出后关依赖；未Serve不等不存在的结果；五结果容量、优先序、父取消、fatal及cleanup错误、原资源恰一次均明确 |

A-NOT-01～09原ID保留，子例继承US/SR、风险和原任务映射；§6将R-A05/06/09与worker/lifecycle风险R-A17/18关联到新增通知场景，不改其它已保护切片。三组细化对生产事实与固定设计有明确依据，没有凭空增加新的业务能力或保证exactly-once。

## 第二轮：真实载体、消费者、命中门禁与交接可行性

本轮沿真实源码/测试定义核规划能否实施，独立于作者的“已核对”描述：

| 实际取证 | 规划匹配与限制 |
|---|---|
| `test/integration/identity/orphan_disposition_test.go:1–101` | newOrphanPGHarness为integration tag、同包私有；每连接search_path/完整migration/已提交schema及retained日志真实存在。不能从article测试包直接import；r15§7.6已安排最小helper内承notice_dispatcher_test.go |
| `test/integration/identity/postgres_test.go:39–105` | migration链确为0001～0009；旧newSchemaHarness的Cleanup含DROP CASCADE，r15明确禁止照搬清理，区分释放rows/tx/连接和删除schema |
| `test/integration/identity/smtp_test_server.go:87–180` | loopback动态端口、合成CA、验证配置真实存在，但只Accept一次后关listener；r15明确需在候选notice_smtp_test.go内适配多会话，而非声称当前helper可跨包或已支持重试/双recipient |
| `internal/testenv/env.go`全文 | 只读取进程环境，缺失t.Fatal；r15区分User-scope注入与进程实际可见，禁止回显/硬编码凭据，缺环境不skip |
| `internal/mailer/smtp.go:118–157`及本会话已核Mailer/SMTP测试 | 四个拒绝点现役共享sentinel、Auth/greeting无该sentinel、DATA最终受理后QUIT失败忽略。05a～g针对真实现状提出兼容Stage扩展与回归，不把计划类型误写成现役已实现 |
| `internal/identity/password_reset_test.go:1451–1485,1529–1566,1618–1644,1696–1725` | §9.5身份正则静态命中4个真实顶层，包括ordinary resend与S11 resend consumer；不是仅近似名称。这里只核符号/源码，不声称执行 |
| `internal/audit/store.go:46–64,106–110`及本会话已核repository/privacy/model | Append绑定caller tx、Get可用同类QueryRower；计划的状态/审计同事务与EventID readback、tx Gate、int64/微秒fencing均有实际消费者接缝 |
| 本会话Design R02已经核对的 `main.go/lifecycle.go/lifecycle_test.go`，本轮hash复核未变 | 真实factory与现役初调/关闭接缝仍待修改。r15用真实factory组件、09f～h启动期取消/清算/join矩阵补风险，不用旧Serve后signal测试或Design PASS替代执行 |
| `add.manifest.yaml:42–49` | 当前七项为gofmt/mod verify/vet/build/contract/race/test。计划要求适用检查另行核证，明确普通go test ./...不含integration tag，未拿全套包级成功代替专项命中 |
| 目录/符号存在性 | 当前 `test/integration/article` 不存在；新增通知符号尚未具备。§11/§9.5如实标planned/pending并要求实现后先核文件/tag/顶层/leaf，未把候选命令说成已可运行或通过 |

§7.6的三层证据具有不同的可核验对象：第一层真实dispatcher/factory、TLS/协议阶段/子进程退出；第二层真实PG claim/preflight/资源归还/结果审计事务流；第三层最终五态、attempt/lease、快照、审计绑定、发送次数与字段符合性。关键场景指定真实PG多连接、实际SMTP和child crash；mock/内存时钟不能冒充SQL/锁、DB时间或真实退出，符合当前专项规划目标。

§9.5候选命令要求普通及race覆盖三个新integration顶层，逐包/逐顶层/逐必需leaf核JSON run和pass；零用例、no-tests、关键leaf缺失或skip均不能PASS。测试子进程的helper退出码不冒充父例通过，父例还须核阶段、退出状态及DB/SMTP证据。该命中合同防止只靠旧TestNotice近似命中或包级ok假绿；当前只静态核符号，不运行候选命令。

九+三文件与design r18§8.5一致：S7.1为article3、mailer2、main/lifecycle4，S7.2为PG、SMTP、crash三个测试文件；私有helper可内承这些测试文件，不强加第四fixture。共享repository/mailer/main/lifecycle需唯一owner串行；实际登记仍由Planner，运行窗口、证据位置及具体接收人仍由Lead/执行owner落实，本计划没有把建议变成已认领任务。原requires/DAG不在本次评审对象内。

## Findings、metadata与不重开范围

**新增findings：0；无CRITICAL/HIGH/MEDIUM/LOW缺陷，无新增方向决策或必须改写的非阻断建议。**

r15 metadata、§1、§4.6.3、§11、§12仍记source DesignReview pending，是作者固定交回时的快照。它绑定的r18/hash与已完成Design R02完全一致；R02没有改设计。Lead可用不可变R02及本报告分别核验设计/规划结果，不需仅为把pending文字换为PASS制造新revision循环。这不是替作者改状态、替Lead现场复核或授予实施/运行权限；若设计hash/合同实际改变仍应只复核受影响映射。

S6.2§4.5.1、§9.4与r14 before-image的分节文本比较一致；其它S6.2结果/计划及Task/Plan为保护范围，不重开、不据历史措辞推断当前状态。r15中对todo、owner、九+三的描述只作为该规划的候选/交回边界，未用正在变化的Tasks/Plan给本次Gate背书。

旧identity PG/Manifest、S6.2双pending快照及其日志只被允许作为前置背景；本计划没有把它们作为新dispatcher/lease/TLS/crash执行PASS。Lead报告的PG18.6只读探针只说明其报告的环境可达，不是本Reviewer执行或通知测试结果。

## 证据摘要、未验证与复审条件

当前关键hash：

- 固定r15：`71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`。
- fixed r18：`CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`。
- 旧Design R01/R02分别为 `1C51D766F9F2B811558F4BE124878875CC5A5FEBCEC49FD5E04B41C875382950` / `115289B8216BD5CB0AD49CB9B82B748E84555FD58C307AAFA272213F387B13E7`；未修改。
- 补充取证源码hash如下，定点阅读范围见前表；不声称全部文件全文重审。

| 载体 | SHA256 |
|---|---|
| `test/integration/identity/orphan_disposition_test.go` | `7B38993E04D342F424C11C4D9E136DD3F06967D7E10A939C1103186F9B289D12` |
| `test/integration/identity/postgres_test.go` | `C3A7F455C6AD40DC43557F578286CD53B1EB48A018DFAC699A43F152C0B9D1CB` |
| `test/integration/identity/smtp_test_server.go` | `FEFF245A580D7D848A780EB63C5AD1D0E2B13582ADDCC3BE6C9506C9F82DFB26` |
| `internal/testenv/env.go` | `5E5763F31CB24A90E7975A0CF60CEABA01506E56155C4D51D4B3A6781D965630` |
| `internal/identity/password_reset_test.go` | `59AAC51742A61ADD318D8ADE5527231D4D5A198790687B101A35A5A843666E2A` |
| `internal/mailer/smtp.go` | `0C64C229B7EE8101B0DCAD95C14431DCAF8ED55809C9FBFD50AEE75AF0221199` |
| `cmd/han-server/main.go` | `4B9D83BA52E322E9E7A430F18A59DAA0FC651D4D4B21BEAFBA825E7E991A7569` |
| `cmd/han-server/lifecycle.go` | `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888` |
| `add.manifest.yaml` | `E8F5E9C1CDB0548BE00F20A187DEA22347D40C8FF2E5E8D61E563699DA253F80` |

本轮只执行只读分支/HEAD查询、rg定位、UTF-8定点读取、SHA256、内存diff/分节一致性及正则符号匹配；初次PowerShell显式通配路径搜索失败后改用rg -g在目录内定位，所需符号已补核。没有执行gofmt/build/vet/Go测试、PG、SMTP、部署或E2E，没有读取实际DSN/secret，没有创建/删除schema或其它运行材料。

本PASS只证明固定增量的风险与场景、真实载体、证据及交接规划足够。未验证通知实现、普通/race实际命中、PG事务/锁/时间/commit unknown、TLS/多会话、child crash、真实factory执行、初次取消/join及完整Host；生产部署值/登记/凭据、容量/留存与真实部署E2E保持未验证，Windows FileJournal独立Gate不解除。计划中具体用例执行是否充分，仍需实现后的新消费者和非作者Review证据，不能从本规划PASS推导。

后续若固定design、这些预期/层次、关键leaf、helper安全行为、写集或命中Gate实质改变，应按实际影响复审；仅补同hash已有Review的引用或Lead实际登记，不自动要求重新审全篇。本轮没有判定保护的旧test-plan所有切片或动态Tasks/Plan整体PASS。

交回Lead：**TestPlanGate PASS，findings 0，绑定上述r15/hash；实际执行全部NOT RUN。** 仅新增本报告；交付报告hash后停止写入。

