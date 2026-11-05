# identity-auth-S4.10 独立 CodeReview R01：静态阶段交回

结论：静态两轮完成；1 条 HIGH finding，1 条待执行核实的静态疑点。**本角色 Go/PG/普通/race 全部 NOT RUN；不出 CodePASS、不关闭 finding/Task/G4。STOP，交 Main 分配精确 actualIdentityPG 窗口。**

本角色为本次委派的非作者 add-reviewer；不是 Zeno，也不是 Design/Task 作者，无再次委派、修复、采用风险或业务签字动作。Host 未向本执行上下文暴露自身 Session ID，因此不编造 ID；身份依据为本次委派输入，Main 保留实际调用归属。

## 固定对象与授权

objectType=code。唯一工作树 `D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。以当前 dirty 工作树源码为权威，不改变他人改动。原三文件是完整新增 diff：

| 文件 | 行数 | SHA256 |
|---|---:|---|
| internal/identity/recovery_execution.go | 2694 | 7C24CA226D0D8DF5FC722A35630C090DD01B26F655C56F79F6C4F9FBEE85E407 |
| internal/identity/recovery_execution_test.go | 1053 | E51D1B82BC7E510A3DAB944C2CA16B85BFB49B132ED28AF12EED45EF596BE9CC |
| internal/identity/recovery_execution_postgres_test.go | 1869 | 5907393CAC84B4D92B590A77B7B10A462D4D1C94D3510A063071D31787202980 |

三源合计 5616 行逐段全文核读；覆盖记录见 `readset.jsonl`、`coverage-two-rounds.md`。作者 full additions diff 重建后与工作树三源逐字节相同，见 `diff-reconstruction.json`；source-final 三 frozen 与 live 同哈希，已在作者清单核验中确认。三源以及 Main frozen24 全部匹配，见 `source-sha.json`、`frozen24-static-check.json`。

技术依据：tasks.md 的原 S4.10 卡 2174–2213；identity design r56 SHA `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF`，重点 §4.2 r55/r56、F001/F002/退出收口、§6.1、§9.1；identity test-plan r26 SHA `34826759C79A58EB0B4A296D2DD5C9157014406859153DC6CA0FA6E2BD921DAB`，§4.7.1 与 r55/r56 块。既有独立技术采用按用户提供与 Main freeze 的各自范围消费；历史 pending 原文不反推未采用，也不升级为人的设计接纳。

完整读取本角色、route/delivery/code-review Skill、用户指定六条 Rule及 abstraction；go-service 入口、编码/硬编码/安全门禁与审计/追踪规则、安全红线。不存在独立的数据库/测试 profile 文件；相应约束来自已启用 profile 的测试部分、设计/TP、实际 DDL 和原 owner，不加载未启用 profile。记录见 `methods-readset.json`。

## 已确立 finding（按严重度）

### [HIGH] S410-ICR-R01-F001：失败 restoration pointer 只凭状态标签通过 Complete

- 对象：上述 7C24 三源；核心位置：[recovery_execution.go:1201](D:/AID/VSCode/Codex/Han/internal/identity/recovery_execution.go:1201)，调用链为 `completeFacts -> reviewProof -> restricted -> pointer`（1366、1243/1245、1190）。
- 现状：pointer 非 NULL 时，SQL 仅要求该账号 operation 为 failed/invalidated、settlement_state=invalidated、job.status=succeeded、claim.status=failed。它不读取原域 result_code/response_snapshot，也不验证对应实际失败结算 Audit。随后 Complete 的全局未决检查只排除非 terminal 状态；cohort 完整性检查针对 finalize 原集合，不能替代本次 finalize 之后新建的 restoration candidate 的合法终态证明。
- 触发：原 ND0 pending review 在 finalize 后保留一个本 run 的失败恢复候选；精确 operation/user/六字段/credential 和组合 FK 全部合法，operation=invalidated、job=succeeded、claim=failed，但 claim 为 schema 允许的 generic legacy failure，或该 invalidated 结算缺对应原 owner 审计。明确作 denied 后，其余共同 C/P 均齐、原 cohort 合法为空。当前 pointer 查询仍为 true，Complete 可转 identity_ready。
- 证据：`0007` 允许通用 legacy outcome；`0008` 的 reciprocal composite FK 确保六字段绑定，但没有证明领域结果或审计的约束，不能把这层 FK 当缺陷，也不能拿它替代结果证明。`password_reset_operator.go:445–467` 的真实终态分类器对 invalidated 要求 code=password_reset_invalidated、outcome=invalidated、resource_id=原 operation；537–563/215–220 的真实 invalidated 结算原子写 claim 和对应审计/job。现有 pointer 查询均未消费这些事实。完整 ACK image 前后相等只证明数据未变，不能把该不合法终态变成合法。
- 违反依据：design r56 中保留的 r55 §4.2:3689–3701（共同 C、合法失败候选及无未决副作用）；test-plan r26:897–898；tasks.md:2207、2211。该要求明确由本卡消费合法持久 fixture 验证，无需等待 S4.12/root。
- 影响：恢复检查提前接受不完整的失败候选历史，输出 identity_ready 给后继消费者。这里只指身份阶段错误接受，**不声称已实际 PublicReady 或发生生产越权**。
- 最小修复方向：由原作者在本卡 pointer/Complete 的原路径核原 owner 对应失败路径的完整终态、实际响应与应有审计/job 事实；复用原分类器及既有不可变绑定，不清指针、不改旧 owner、不授 generic Retry。不同合法失败路径应按其原合同核审计，不能一律发明同一种审计要求。
- 验证/复审：Temp probe `TestS410IndependentFailedRestorationPointer` 已准备合法 invalidated 正控、generic failure 和缺原 invalidated 结算 Audit 两负控；保留指针、原 claim 不变，负控不得转 identity_ready。当前仅静态控制流和 DDL 证据，probe **未编译/未运行**，没有伪造 Red。Main 可先在精确窗口取得当前反例，再交原 Zeno 最小整改；整改后仍由本 Reviewer 引用此 ID 复审新 SHA、对应回归及六组普通/race。此报告不标 CLOSED。

上述反例针对设计明确要求拒绝的不完整持久证据，以与现有 restored-generic 负例相同层次的 schema-legal fixture 隔离。它不假设 Reader/provider 具有改写任意数据库的权限，也不声称原 owner 当前正常执行必然产生该损坏数据。

## 静态疑点与执行计划（不混入 finding）

**S410-ICR-R01-S001，退出 Audit 的缺字段默认值**：`recovery_execution.go:1549–1560` 把退出 metadata 解码为 `IDs []string / Count int64`；缺失字段或 `revoked_ids:null` 会成为 nil/0，当无未撤会话时满足 len(ids)=0、len(m.IDs)=m.Count。`recoveryExecutionAudit` 已核 run/backup/install，但不验证退出结果字段必填/非 NULL。design:3778–3780 要求完整退出审计。已准备 `TestS410IndependentExitAuditCompleteness` 两个缺证负例；当前未确认现实生产 writer 会产出这类 metadata，故保持静态疑点，不冒充实际攻击或已运行 FAIL。实际窗口同时保留原首次零会话合法正控；若组件确实接受不完整 receipt，再按实际结果定级。

**S410-ICR-R01-S002，原域 generation+1/linked 历史的正向边界**：Complete 对原 cohort 固定读取原 generation，并逐项要求 failed resolution；原 r56 还保留“原 owner 已独立授权且连续无 ABA 的 generation+1 链”合同。当前公开 maintenance 入口本来不授 Retry，不能凭 generic Store.Recover 能力制造一个实际业务授权，也不能据此阻断当前原域失败正向。尚未找到本次边界内现役授权的完整 retry producer，保留未验边界；不新建任务、不重开旧卡。durable password-reset resend 的原 scope 是 temporary_credential.resend + password_reset，当前 allowlist 只接该 scope 的 new_administrator；它的合法终结来源同样须先由原 owner/精确历史证明，不能靠伪造“源已批准”扩 allowlist。

另准备 `TestS410IndependentPersistedDomainReplay` 三正控：实际 Recover 后用原 Store.ResolveExisting 从 PostgreSQL 解码持久 snapshot，再交真实管理员创建/重发 replay。现有 P 测试在回读状态后手工把 src.ResponseSnapshot 放入 Record（postgres_test:1170），没有覆盖该完整读取/解码消费者链；这是补充独立验证方向，不据此宣称 codec 有 Bug。

## 两轮范围与已有证据

Round 1：完整三源本体，核实际新增 diff、五方法与 typed DI、权限/作用域、User/Store 锁序、资源退出、finalize/cohort、D0/ND0、canonical/denied/restored、ACK、失败来源包装与深拷贝、一次 Now/原 Recover-Audit、退出及全部测试断言。未以组名、计数或 fixture bool 代替整组证明。

Round 2：沿真实 S4.9 RecoveryRunStore/readAdmissionTransaction/finishAdmissionRead、全部 0001–0010 DDL/索引/约束；S9.1 CanonicalDeleteCoordinator/原 A/B validator/Review Store；真实 ArticleAccountPort、CommandService.OrphanArticlesCanonical 与 Article repository；原 idempotency scope/hash/Recover/decoder；PasswordResetStore.CheckLockedUser、Gate SQL、真实 replay/settlement 分类器/SQL、resend scope；Runtime 的实际 CheckPhase/PhaseContext/ReadPermitBinding 与公开 identity producer。精确读取范围见 readset；没有宣称全仓消费者已查，没有把未实现的 root 注册当现役事实。

本角色静态复核作者 canonical packet 的 report/receipt/两轮 self-check、validation、TDD、diff/source-final，以及所有 51 raw stdout/stderr、result/joins 和 leaf 结果。766 个 manifest 项全部哈希匹配；自行从 raw JSON 重新计算 leaf，最终 U64/P79 各普通/race 名称集合相同，最终九检查记录退出 0；ACK TDD 为同测试字节五 FAIL/两 PASS 后七 PASS；restored TDD 为真实一 FAIL 后七 PASS，最终79保留新负例。所有历史诊断仍在。原 raw 记录的 parentWait/kernel Job empty/PipesJoined 均满足；758 个 raw 记录 schema 与最终 raw 内 recorded physical 集合相同。

以上为**对作者/Main 已记录执行证据的独立静态复核**，不是本 Reviewer 重新运行 Go、连接 PG、查物理 SQL 或观察进程。79 optional PID、1829 kernel accounting 和 Main100 known PID 各自范围不混同；本角色未再观察任何 PID/handle/服务。见 `author-evidence-static-audit.json`、`author-manifest-hashes.json`。

## 未验证、后续窗口与停止

- 本角色 U64/P79 普通/race、list/vet/fmt、独立 probe 全 NOT RUN；当前不得以作者执行签本角色 PASS。
- 原测试前缀在 Temp 全字节保留，仅 integration 测试末尾附 probe；production overlay=0，test overlay=2。见 `prepared-tests-preservation.json`、`test-only-overlay.json`。未调用 Go/gofmt/编译器。
- 后续仅 Main 精确 start 后执行 `execution-plan.json`：独立 D 缓存/work/binary 与 packet 分离；local/offline/GOMAXPROCS=1/-p=1；用 Main 给定 fresh IdentityPG 版本/连接，不从本阶段读取资源/DSN；逐例新 schema、独立物理 SQL 留存、源3前后哈希、父 Wait＋kernel Job-empty＋双 pipes EOF/真 join。编译/SQL准备/超时不能算行为 Red。现阶段不启动这些动作。
- 完整真实 root/HTTP/维护认证/生产 Reader/provider/批准主体/ACL、SMTP/native restore/真实恢复/E2E/容量未验；fixture只证组件软件。相关 Reader 缺失只关闭受影响 resolve/非空 cohort Complete，不能倒挂 S4.12/13/16/17/root/native 为本卡局部前置。
- 原 unknown hold/mail 保护窗、canonical A/B/phone和缺失行残余限制保留；不以宽泛豁免替代证据。draft 与人的接纳 unknown 仍分别存在，不增 checkpoint、不重复 G0/设计前审。
- 无源码、设计、Tasks/Plan、Git、cleanup/archive/move 写入；无 Go/SQL/PG/网络/API/服务/进程检查或下级委派。所有报告/准备文本在本独有 D Temp。初始化时额外写了一份 `D:/Temp/han-s410-review-packet-pointer.txt` 文本指针，超出“独有目录”写边界；已停止该范围外写入并如实保留，不以清理/移动掩盖。它不是项目文件或秘密。

交回位置：Main 当前 S4.10 首轮独立 Review。修复方向/风险处置由 Main/用户决定；本角色不替作者修复，不替人批准。静态包交付后 STOP 等精确运行窗口。
