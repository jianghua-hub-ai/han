# Han S7.1 通知设计 r17 独立评审 R01

**DesignReviewGate：CHANGES REQUIRED。1 HIGH，0 CRITICAL/MEDIUM/LOW。**

未关闭 finding：`AM-DES-R17-NOTICE-R01-F001`。本结论仅针对固定 r17 通知增量，不作 TestPlanGate、代码验收、部署或 E2E 结论。

## 固定对象、权限与依据

- objectType：design；作者 Ampere 已停写；执行者为本次独立 Reviewer，会话未参与 r17 编写，未再委派。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，现场核对一致。
- 目标：`.add/project/knowledge/design-workspace/article-management/design.md`，`article-management-design-ir01-r17`，SHA256 `670EFF7E1206DA197E9DF557DB0C150A3D78C9952319F0E30F68CA2482FE644C`。
- 基线：r16，SHA256 `CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC`。实际读取指定临时根 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672` 下的 `.add__project__knowledge__design-workspace__article-management__design.md` before-image。
- 精确逐段对比（SequenceMatcher，autojunk=False）为新增 215 行、删除 49 行，与交接一致。所有差异均检查；S6.2 受保护正文不重开，不把公共文件后继候选当成修改既有业务合同的授权。
- 唯一可写与实际新增文件：本报告。其余全部只读；不改 design、tasks、test-plan、plan、代码、Rule、Git，不清理、不发邮件、不连接生产 SMTP。
- 业务依据：requirements r16 的 US-IDENTITY-03.9 第 1～9 条；stories r07 的 Story 7；article boundary r13 的 S5/S6/S10/S12 与排除项；ADR-0006/0008/0011/0013/0016/0017/0022 的适用约束。
- 固定测试规划背景：同临时根 `.add__project__knowledge__design-workspace__article-management__test-plan.md` r14，SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C`，读取 A-NOT-01～09 及其需求/风险映射。未读取 Ohm 正在写的现役 r15 正文，也未对其定版、采用或给 Gate。
- 用户已授权采用有依据建议；不以 draft、外部部署值 unknown 或历史确认未扩用为缺陷，不要求重走 Route/边界或补造人的 accepted/checkpoint。

## 两轮实际检查范围

### 第一轮：固定差异与设计内一致性

审阅 r16→r17 全部差异及对应上下文：header/追溯/假设、§3.1.5/6、§3.3.1 的配置与 Notice/Recipient 结构、§3.3.8 全节、§3.4.8 全节、§3.5、§4.1/4.4/4.5、§5.4、§6.1/6.6、§8.1/8.5、§11/附录 B。逐项核需求推导、对象/输入输出/字段类型、数据所有权、前后置条件、五态、过期与恢复、审计、兼容和验证接收位置。

从独立反例重新走查：max_attempts=1 的 deferred 与崩溃、最后有效 attempt 的成功/瞬时失败、相同 ID/相同微秒 lease 的旧 attempt、DB 故障与 commit unknown、accepted 后崩溃和两 recipient 分离。未将已有模型、未来测试和原任务卡当成通知已实现证据。

### 第二轮：现役消费者与上游合同

沿指定依赖范围核真实工作树，未扩成全仓预读：

| 实际取证 | 检查内容 |
|---|---|
| `internal/article/model.go:61–84,142–168` | RecipientKind/五态、int64 attempt/max、可空 lease/时间、独立快照与 Notice 关联 |
| `internal/article/repository.go:78–128,334–380,467–608` 及关联 SQL | caller-owned tx、双 recipient INSERT、完成事实只读回查、无现成通知消费 SQL；后继增加位置可行 |
| `internal/article/command_service.go:1080–1162` 与调用点 | 转移事务内地址与 max 快照、两 pending、同事务审计/完成核验/邮箱清理；dispatcher 可在提交后独立消费 |
| `migrations/0003_article.sql:143–248` | PK、bigint/check、状态与 accepted_at、两 recipient 延迟完整性约束、claim 索引与无 Article 内容 FK |
| `internal/mailer/mailer.go`、`smtp.go` | purpose 已支持；四处明确拒绝共享 sentinel；原 textproto cause；Auth/greeting 无 sentinel；强 TLS/deadline/取消、DATA 受理后 QUIT 失败不改成功 |
| `internal/mailer/smtp_test.go:25–195,223–345,398–601` 等定点 | 原四阶段拒绝/Is-As、Auth/greeting、阻塞 I/O 取消、DATA accepted/QUIT、隐式 TLS/STARTTLS 和证书拒绝测试源码 |
| `internal/privacy/store.go` | QueryRower 可绑定 pgx.Tx；沿用原配置和 RequireOpen；now() 是事务起点，READ COMMITTED 不等于锁住未来关闭 |
| `internal/audit/store.go` | Append 在 caller tx，Get 可读本次 EventID；系统主体/metadata 最小化可承接，不需第二 authority |
| `internal/identity/password_reset.go:330–344` | 现役消费者仅依 errors.Is(ErrMessageRejected) 分显式拒绝，结构化阶段必须保留该链 |
| `cmd/han-server/main.go` 的 startup/config、builder/Gate、runtime/factory、productionDependencies | 同 pool/Mailer/Audit/ID 来源、DB 前校验、非 nil worker、通知内部接缝及真实 signal 来源；发现 F001 |
| `cmd/han-server/lifecycle.go` 的启动/trigger/shutdown/清理主体；`lifecycle_test.go:534–690` 等定点 | 六方法、两 server+两 worker 的现状、优先序、容量4与group.Wait顺序、Done后资源关闭、初次Reconcile与signal接入间隙 |
| `cmd/han-server/main_test.go:562–581,758–766,4041–4107,6562–6609`；相关测试名称定位 | 原 max bigint 无 fallback、缺依赖拒绝和 DB 前失败模式；均仅源码审阅 |
| S6.2 verification/joint R02 的通知边界与固定结论 | 仅确认原交付止于双 pending/快照；不复用其 PG/Manifest PASS 证明新增 dispatcher |

两轮均由本 Reviewer 独立于作者执行；第二轮从实际运行入口反查设计，随后再次核对 F001 是否已被 §3.3.8/§6.1/§8.5 解决。未找到启动调谐期间 signal→取消/停止的明确接线合同。

## Findings（按严重度）

### [HIGH] 首次通知调谐执行发信，但 signal 要等调谐结束才被消费

- **finding ID**：`AM-DES-R17-NOTICE-R01-F001`，OPEN。
- **对象/修订**：design，article-management r17，目标 SHA256 如上。
- **位置**：`.add/project/knowledge/design-workspace/article-management/design.md:1797–1800`（核心落点 `:1799`）；关联 `:617`。
- **现状**：r17 把初次 Reconcile 定义为包含逐条 claim/preflight/Send/保存的完整轮次，要求它在 HTTP Serve 前完成；Stop/Wait、共享 shutdown 与 signal 的约定却只描述已经进入运行阶段后的停止，未规定初次调谐如何接入同一 signal 协调器。沿用当前 lifecycle 接法时，通知首次外部投递处于 signal 监听之前。
- **直接证据**：
  1. `main.go:157–158` 以 `context.Background()` 进入 run；`:3508–3510` 用单独的 signal.Notify 通道收 SIGTERM/SIGINT，并不取消该 context。
  2. `lifecycle.go:60–82` 同步完成全部初次 Reconcile 后才 Start；`:84–90` 才进入 waitForLifecycleTrigger。
  3. `lifecycle.go:203–204` 才消费 config.signals；`:97–103` 的 Stop 和共享 shutdown deadline 要在该消费之后发生。
  4. 现有 lifecycle 测试 `lifecycle_test.go:557–566,685–687` 在 Serve 已启动后才发 signal，不能证明启动调谐会被取消。SMTP 的取消实现可响应 context，但这里的 context 未被 signal 取消。
- **触发场景（静态反例，未执行）**：重启时已有至少 50 条到期通知。合法隔离 tuple 为 Poll=1s、BatchLimit=50、DB budget=1s、SMTP budget=2s、Lease=10s、Backoff=1s..8s、shutdown=5s、max_attempts=3；满足 r17 两个预算不等式。首次 Reconcile 的首个 Send 期间收到 SIGTERM；各 Send 均在约1.5s后成功，属于正常结果，不触发“错误结束本轮”。signal 只留在缓冲通道，剩余 claim/Send 仍可执行。仅 SMTP 时间即可约75s后才读 signal，远超5s；这不是生产参数建议或实测数据。
- **影响**：启动期间停服请求不能及时取消在途 SMTP，也不能及时阻止下一 recipient 的外部发送；共享 shutdown 预算不覆盖这段等待。部署若随后强杀，可能在已受理但结果未落库处退出并放大重复风险。单次 I/O 有 deadline、单轮有 BatchLimit 均不能替代从 signal 开始的关闭协调。
- **违反依据**：r17 `:1798–1800` 自身的 Stop/cancel/不再发/同一关闭链合同；ADR-0022 `framework.md:675–680` 的 signal 触发同一关闭协调器、共享预算与错误暴露；code-review-checklist §2 的取消与资源收敛、coding-standards §3 的取消/超时合同。此 finding 只针对 r17 新增的首次通知消费与接线缺项，不重开 S6.2 业务实现或其报告。
- **最小建议**：由原作者明确：在进入首次通知 Reconcile 前，同一 lifecycle 协调器已能接收 signal/父 context 取消，并取消传给该轮的操作 context、禁止后续 claim/Send/Start/Serve；只允许有界、fenced 的当前结果清算，等待实际退出后按原 owner 顺序关闭资源。signal、fatal error 和清算错误仍按原优先级/退出合同暴露。可在已列 main/lifecycle 候选内实现，不必新增队列、root、schema 或修改 S6.2 业务规则。若作者选择把实际发信移到 signal 已受控的运行阶段，应同时说明首次 Reconcile 的职责，消除与当前“完整消费轮”的矛盾。
- **验证/复审**：设计阶段补清 signal 从首次调谐起的 context owner、时序及取消后置条件，静态走查“首个 Send 阻塞/已受理未保存/第二条尚未 claim/初次调谐前已来 signal”，并同步 §3.1.6/§3.3.8 的相关描述即可送设计复审；不要求先实现或编译。后续实现应以 channel/barrier 驱动现役 lifecycle 和可取消 Mailer 验证：Serve 前发 signal 即取消当前调用，零后继 claim/Send/Start/Serve，结果最多一次有界保存，Done/Wait 后才关 pool，关闭错误非零；同时回归原两个 worker/server 的顺序与恰一次关闭。

## 其他重点的独立核对结果

以下为设计静态覆盖结论，不是执行 PASS，也不抵消 F001。

| 关注点 | 核对结果与边界 |
|---|---|
| 原 privacyGate 与两道检查 | §3.3.8 指定 root adapter 在调用者 tx 构造 privacy.NewStore(tx)，配置同源；claim/preflight 分别新事务；不复制 SQL、不缓存 open。§3.4.8 正确承认 now() 事务起点及在途关闭窗口，未声称零窗口 |
| DB/SMTP 资源 | claim 明确提交成功才 preflight；preflight 提交/释放 rows、tx、pool 连接才 Send；预算扣除 monotonic 已耗时间；无 DB 连接跨 SMTP |
| 五态/耗尽 | 最后有效 attempt 仍能 accepted；过期 processing(max) 维护终结 unknown+风险；Gate deferred(max) 先 retry_wait，到期维护在 Gate closed 时仍执行；未到期不提前终结 |
| 旧 owner/ABA | PK/status/claimed_by/attempt_count/精确微秒lease 全匹配且当前未过期；相同 workerID+lease 仍由 attempt 隔离；过期未回收亦 lost |
| DB 时间与 commit unknown | 锁后 fresh clock_timestamp、UTC 微秒、DB RETURNING authority；unknown claim不发、不盲claim；结果unknown按当前行+EventID有界回读，不再Send；审计失败不报持久成功 |
| accepted 崩溃与审计 | 回收、耗尽和风险事件同事务；accepted后保存/审计失败保留processing并告警、停止本轮，后续租约恢复补风险；明确不能证明从未受理或 exactly-once |
| SMTP 分类/兼容 | 四个明确拒绝点兼容加 Stage；MAIL 4xx/5xx作sender/config，RCPT/DATA 4xx退避、明确5xx永久；Auth/greeting/TLS不冒充recipient永久拒绝；保留sentinel/textproto；nil受理后QUIT故障不重试 |
| 隐私与内容 | 单 recipient、purpose已支持；固定ASCII Subject、无链接纯文本、仅UUID引用；只用快照，不读当前邮箱/Article内容；不输出原始SMTP错误或PII；不伪造外部处理输入 |
| 正常运行 shutdown | 设计已要求 Stop取消、一次有界结果保存、Wait/Done后关依赖、五个结果容量、保留原错误优先序。F001限定首次Reconcile尚未接入signal的间隙 |
| 配置/接线/所有权 | 六新增键及交叉预算明确，无生产数值默认；同pool/Mailer/Audit/ID，非nil真实worker。九文件候选具有实际接缝依据；原三文件不足已明示，仍须Planner对账/Lead分配，不是已认领或已实现 |
| 兼容与回退 | 无新HTTP/schema/业务边界；worker回退暂停消费并保留队列，不重置终态/计数；停止不主动释放processing。共享mailer需原消费者回归，不能改凭据邮件语义 |

A-NOT-01/02/03/04/05/06/07/08/09 在固定 r14 分别覆盖快照、Gate、竞争、状态到期、分类上限、Gate竞态、accepted崩溃、旧owner和TLS；r17补充的max=1、ABA、commit-unknown、原子风险审计、阶段错误、production wiring和lifecycle细化均需后续固定测试规划承接。这里没有评审现役 r15，不以 r14 的旧 at-least-once 表述覆盖 r17 明确的有界 best effort，也未另造测试计划。

## 真实读取、命令与限制

完整有效正文已读取：`.codex/agents/add-reviewer.toml`、`add-code-review/SKILL.md`、当前 `add-delivery-workflow/SKILL.md`；规则为 code-review-checklist、abstraction-principles、design-doc-writing-style、task-execution-workflow、产品安全红线，以及与通知接缝有关的 core coding/no-hardcode/observability、decisions-guide、go-service README与相应 coding/no-hardcode/observability 规则。Go security-coding 加载门禁已读，其正文声明只对实际源码修改适用，本次没有据此实施。未启用其它 profile，未将gRPC/DDD/资源平台示例强加到本设计。

实际执行的是只读 `git status --short`、`git branch --show-current`、`git rev-parse HEAD`、`rg` 定位、UTF-8定点正文读取、SHA256和内存逐段对比；无Git写操作。首个默认编码读取出现乱码，已改UTF-8重读；含中文路径的stdin脚本读取失败后改按目录实际文件定位补读，无写入副作用。评审结论不依赖乱码或失败读取。

没有运行 build、Go测试、lint、PostgreSQL、TLS SMTP或生产Host；没有新增通知实现可供执行。F001来自固定设计与真实入口的静态控制流反例，不伪装为已执行失败测试。现有测试仅审阅源码，既有identity/S6.2测试和报告均不是本轮执行证据。

外部SMTP凭据/处理登记/部署值、容量与保留、通知专项PG/TLS/崩溃、支持Host上的完整production启动、真实部署/E2E均未验证。Windows FileJournal能力Gate保持原状；本报告不推导Linux或真实卷韧性结果。未对保护的S6.2正文、全仓其它消费者或未声明范围作PASS。

无另增方向决策或非阻断改写建议。现有部署unknown继续由真实部署owner补齐并保持相关能力Gate；人的精确结果确认与后续实施授权交回Lead，不从本报告或旧accepted推导签字。

## 固定工作树取证摘要

以下SHA256固定本次所见未提交消费者/上游内容，不能只用HEAD代替。定点阅读范围见前表，并非每个大文件全文审查；交付前复核这些内容哈希。

| 路径 | SHA256 |
|---|---|
| `.add/project/knowledge/design-workspace/article-management/design.md` | `670EFF7E1206DA197E9DF557DB0C150A3D78C9952319F0E30F68CA2482FE644C` |
| `.add/project/knowledge/requirements/requirements.md` | `999CBA3C0CD25D2760E400A41D2A373611406854E76E24E453321913D71CBC4F` |
| `.add/project/knowledge/design-workspace/article-management/stories.md` | `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05` |
| `.add/project/knowledge/design/article-management/boundary.md` | `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E` |
| `.add/project/knowledge/decisions/identity.md` | `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978` |
| `.add/project/knowledge/decisions/framework.md` | `8D381C9D753242B47E1B6CF0C269228AD4ED591A4605CA4F119CAF03DB4E0AC7` |
| `internal/article/model.go` | `3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4` |
| `internal/article/repository.go` | `E0C2430A981A5F90E84F197159158CC4A218A4FB399C74E90C203B19AC0FE078` |
| `internal/article/command_service.go` | `585BB51D19B3FCD7BFDC9C84465D2BB5D24F41BD8285553FC20BC10B161B4D68` |
| `migrations/0003_article.sql` | `FCAECE696DCE29D2F8D252EA36C4550F0C1736347E921F40703540B99E100D4A` |
| `internal/mailer/mailer.go` | `A2F103D6AF3664288D0C4F460EC5733E33FA6461FB8554A189FBC3194C5B4D0D` |
| `internal/mailer/smtp.go` | `0C64C229B7EE8101B0DCAD95C14431DCAF8ED55809C9FBFD50AEE75AF0221199` |
| `internal/mailer/smtp_test.go` | `656E12CB62C113C60AB074AE5374F9E206D226CCEE5B194BB1F0F8AA12B96D15` |
| `internal/privacy/store.go` | `6E555D74E5210EA20CE0C33DB08DA40CD6BA4B7D1549B9A8031E0F4D82EEDD9D` |
| `internal/audit/store.go` | `091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F` |
| `internal/identity/password_reset.go` | `5F2D0E1B534E1A4F47AF25675FB94AA3B2830A01C408F6C615E41FAB74A81575` |
| `cmd/han-server/main.go` | `4B9D83BA52E322E9E7A430F18A59DAA0FC651D4D4B21BEAFBA825E7E991A7569` |
| `cmd/han-server/lifecycle.go` | `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888` |
| `cmd/han-server/lifecycle_test.go` | `B3AA659B1A98E0AE8A3EF09678309866DC3D46345B2A6C29A5A220D700B36DDF` |

## 交回与停止条件

交回 Lead：DesignReviewGate CHANGES REQUIRED，`AM-DES-R17-NOTICE-R01-F001` HIGH OPEN。原作者按获授权范围最小修订design后，提供新revision/hash与该finding对应差异，由独立Reviewer复审；实施回归仍归后续开发/验证任务，不要求本设计评审先产生代码。

本次只新增本报告，不修改其它成果，不登记任务、不更改进度、不创建checkpoint、不代签接收、不启动下一角色。报告交付后停止写入。

