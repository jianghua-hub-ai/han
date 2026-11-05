# S1.5 TestPlanReview R01：r19 真实审计 INSERT 失败子集

2026-10-28。Pauli / 实际非作者 add-reviewer；测试规划作者 Ohm 已停写。objectType=`design`（测试规划合同）。固定对象、环境时点及采用的载体选择来自 **Lead 交接**，不是用户逐项直接提供，也不是本 Reviewer 的运行结果。沿当前 development-delivery 续接，不重新 Route、不代作者修复。

**有界 TestPlanReview Gate：PASS。** 已完成两轮独立检查，本次固定 r18→r19 增量无新增 findings。第五文件、唯一 integration top 及四个必要 leaf 的规划可实施；这是规划结论，不是 CodeReview、SQL/PG/race 运行 PASS、CODE/LOCALPG START 或 S1.5 接收/关闭。应用 ACL、原十二 requires、S8.2/Case18 与完整恢复门禁均保留。

## 1. 固定对象及实际差异

实际核工作区 `D:\AID\VSCode\Codex\Han`、分支 `main`、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。唯一仓库写入为本报告；此前准备阶段未写报告。

| 对象 | 实际 SHA256 |
|---|---|
| 当前 `test-plan.md` r19 / fixed 副本 | `88296F772A19600A237CCA9F47A16D4B673828EF41E5C1D0C2D88924BE2C6F66` |
| r18 before-image | `BCBCFDD8CD1747540726A46C35D565FF1CC211AAB43F6059145474568841DE42` |
| `r18-to-r19.diff` | `D75CD57C0D4545AA5FAFAF4069133F4FC1FA3B72A372F0BC1E01B4DD3DBD3332` |
| 原 r18 Review（来源措辞更正后的最终版） | `85A9E111677DAC31E4A99A31D07F1FB8CFBB3E263CF1574196BFF9EA30D50A9E` |
| design r21（同版复用） | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA` |

作者交接原件目录：`C:\Users\J00010385\AppData\Local\Temp\han-s15-audit-testplan-r19-f580e3ea855c46c18ac6f688d98c150c`。实际读取完整 diff、handoff、actual-readset；独立读取 before/fixed/current 原字节，重建 unified diff 与交回文件逐字节相同，当前稿与 fixed 相同。实际 **+55/-10、七个 hunk**，不是采用作者摘要代替差异检查。

## 2. Round 1：增量范围、载体与原层保留

实际检查 metadata/来源/状态措辞、§4.7.1 的审计载体更正、新 §4.7.2（当前574–608行）及 §12 增量结论。第五路径为已采用的测试载体选择，不改生产公共合同：原四个 `test/contract/article/` 文件冻结，新增候选仅 `test/integration/article/audit_insert_rollback_test.go`，首行 integration tag、package `articleintegration`、唯一 top `TestArticleAuditInsertRollbackContract`；其下四个必要 leaf 为 `MarkRollbackAndRetry`、`RestoreRollbackAndRetry`、`TransferRollbackAndRetry`、`FinalizerSQLRollbackAndRetry`。

文件级 tag 将真实 PG 测试与默认 contract 隔离；撤销旧两审计 top 在 contract 合跑的建议，ACL 候选另保持 pending。没有用默认无 DSN 时 skip、placeholder 或成功空跑解决环境依赖，也没有为本轮改变旧测试或新增生产接口。

独立字节/映射检查：所有既有 Case 行及其顺序相同；完整 Admin2 Case 区块相同；§4.8 至 §12 前的剩余正文、§9.6 九路径/四 top/命令区块均相同，因此既有20 Case、38层映射和 Case18 分项边界未被本增量替换。原 S1.5 的12 requires逐项与 handoff 清单相同，含未闭合的 S8.2；当前 S1.5/S8.2 均为 `[-]`，没有将局部交付视作前置满足或整卡关闭。

原四组件源码逐项匹配 r03 固定哈希；286节点/236 leaf、23风险组、16 Handler分支只按固定 Lead 核证及 Erdos 独立R02记录消费，未在本轮重新执行、解析旧raw或重签其正确性。该236 leaf集合不由本次第五文件规划增删。

## 3. Round 2：真实直接接口与四 leaf 可实施性

| 风险面 | 实际接口核对与本次判断 |
|---|---|
| 真实审计 SQL 失败 | `audit.Store.Append` 在 caller tx 直接执行 `INSERT INTO audit_events`，以 `%w` 保留数据库错误；`0001_shared.sql` 的 UUID 主键可由先提交的合法合成行触发 `23505 / audit_events / audit_events_pkey`。计划要求同时核实际 Exec 和返回链中的 PgError、对应 schema，排除其他表冲突、文案匹配、伪 PgError、成功 INSERT 后 synthetic error与 trigger55000。`platform.PublicError.Unwrap` 保留 cause，治理服务包装不妨碍 errors.As，故此证据路径可实施。 |
| 已进入业务写的证据 | 审计 wrapper 在真实 Append之前，使用收到的同一事务读取本例已发生的业务写；随后原样调用 Append，记录真实23505。这样可区分“业务写前失败”与真正的回滚保护。观测 wrapper 不提交事务、不替换目标 SQL，不新增状态 authority。失败后独立连接的全字段/集合快照与预置碰撞行对照，补足仅 count 或最终错误码的假绿风险。 |
| Mark / Restore | 现役 `Moderate` 依次锁行、校验合法状态/version、更新 Article、Append审计，错误时 rollback。两例可由现有 IDs依赖在审计位置提供重复合法UUID，真实 Repository/Auditor/事务不替换。Restore 的 category oracle来自不可变前像；换新审计ID、保留原合法version重试后，由另一连接见version+1和准确mark/clear审计，符合真实责任。 |
| Transfer | 真实 `TransferOrphan` 经身份锁、Article锁后插Notice/双recipient并更新Article，再Append审计；失败时整事务回滚。故前置观测能看见待回滚的归属及通知写；失败后可核 Article、Notice/recipient、Audit、源retained-email/目标身份不变。重试使用独立有效ID序列，避免在notice或清理审计位置先发生另一唯一冲突。 |
| Transfer公开身份接缝 | 精确补查 `identity/article_disposition.go` 的公开生命周期构造/完成 verifier/reference及清理入口，与 article `DispositionCompletionProof.Reference/VerifyCompletion` 对照。现有类型adapter可捕获真实工作流生成的 proof，使用同tx/trace和真实回读完成字段转换；不需要构造私有proof、复制状态机或读取/改动 main。仅检查这个直接接缝，不重审身份业务/恢复文档。 |
| Finalizer合法前像 | `Finalize` 公共路径从 Journal.FindByOperation取tombstone，再claim→complete。最小anonymous published/source=NULL普通删除符合 Article约束；operation先存在、pending Article version应为operation.article_version+1，tombstone绑定operation/version/time/sequence/key。该前像在lockDeletionDisposition中可合法返回nil proof，不必伪造身份完成权威。计划明示合法绑定与真实Repository路径，既有类型和迁移可支撑。 |
| Finalizer事务边界 | claim 独立提交finalizing/owner/lease；complete另开事务，先DELETE Article，再Append审计，最后才CompleteDeletionOperation。23505回滚的是completion，不能要求operation整行等于调用前。计划保留调用前、claim提交后/完成首业务写前及失败后三份前像，允许已提交claim存在，核pending Article恢复、audit/completed字段为空及无通知完成副作用，边界正确。 |
| Finalizer重试 | 现役 Claim 对未到期finalizing返回未claim，即使同worker也不能直接重入完成。计划从真实locked_until推进受控UTC微秒clock至过期，再经公开Finalize真实重claim，完整审计绑定后清owner/lease；不改DB lease、不调用私有complete。该重试符合现役claim与completion租约检查。 |
| Journal证明范围 | 公开 ArticleDeletionJournal 接缝可返回固定且正确绑定的tombstone，支撑真实PG完成事务；仅证明该普通匿名删除SQL子集。计划明确未证明FileJournal验链/fsync/POSIX、真实旧备份、全类型删除或orphan清理；未把受控Journal称作持久化Journal。 |
| 锁/可见性 | 失败返回后须实际rollback和连接归还；另一实际backend连接按适用锁序用FOR UPDATE NOWAIT核锁已释放，不能用同连接重入或timeout替代。fresh-ID重试commit后再由独立连接见完整业务/Audit绑定。计划同时要求先前同tx写入观测及失败后独立前像，因此不是“从未写入所以快照没变”的证明。 |
| carrier / harness / 资源 | 新文件内承新增helper，允许同package只读复用合法片段。旧newNoticeHarness的CREATE与迁移同tx不满足失败retain；r19明确禁止直接调用其建schema阶段，参考newAdminPG的独立CREATE提交、立即log，再另tx执行0001～0009。每连接search_path、合法CHECK/FK、schema成功失败均保留、rows/tx/context/pool及实际goroutine收束均有要求。未借此改旧harness或清理数据。 |
| 环境与命中 | 执行前对本次实际注入的主地址及全部fallback逐项限定literal loopback:5433，连接后核地址/port/version180006/身份，均在CREATE之前。环境事实来自Lead时点交接，本Reviewer未解析DSN或连接。精确integration -list/普通/race/tagged vet与四leaf命中门禁清楚；缺DSN、缺leaf、skip、空跑不能记PASS，默认contract/旧S22/S82/全包结果不能替代本top。 |

计划允许现实现正确时first-GREEN，预置合法冲突本身提供真实数据库故障场景；不强造产品RED。若实施另做检错负控，则须授权Temp完整snapshot/overlay产生可编译的真实行为失败，不能使用编译/环境错误或硬编码Fatal。若Finalizer实际接缝出现缺口，仅停该leaf并交具体缺项，其余三例可交独立证据，但整个SQL子集不能因此假称完整PASS。

## 4. 证据归属、固定读集与漂移

| 固定证据/输入 | 本轮消费方式 |
|---|---|
| `verification-s1.5.md` | 实际读组件r03交付、独立R02/Lead Manifest与未验边界；SHA=`AA6E5F3DC1CEE85B62AD65245EAE70C1CAF39FF9EBA8D75399AA7854C4F13F56` |
| `review-code-s1.5-components-r02.md` | 实际读对象/F001修复与关闭、286/236结果及限制；SHA=`52298FEDD5F13E9A9017334C1FEA72A1BD72C9A9658A2384660D9075D6F59933`。这是原Erdos的独立CodeReview，不是本轮规划评审运行 |
| r03 Manifest回执 | 仅核实际文件SHA=`8904DA4630D4441B7EBE06C15A86FD24C49190CBE46AFCC3CFE7B06F37E61047`；其执行/结果消费自上述固定核证，不冒称本Reviewer执行或本专项已通过 |
| 四contract文件 | 仅指纹，与r03相同：REST=`CEBA105C2CBF70B91CB5DFF93D9D4FC37077153A7EDB4F9C2806C29E650E9533`，Ports=`DC6791D4793ABE67FE17B83736FC77E55C347859B13A4FB5071E6AC51BF64F25`，Projection=`D7585EE66538EECA4364BF01CC2BE3A4AC69D98C9818560D6533457E8A572033`，Producer=`B519DDE178CB3F4357560681490FD00639549E8CA0B369E2EFB01785488D8FDC`；未重审其代码 |
| Tasks / Plan | Tasks指纹=`3B1EE1F9BD6A89A3F9B0A82C666A6E6251EB423E14D352A7A5C137292F75B373`，只定向核原S1.5 requires及S1.5/S8.2状态；Plan=`DE377B618B1FF3E31C9CBF1F778ACED31B275CB7605DB9BB338EAA04A6C787A1`仅hash，不审正文/其他任务 |

实际读集（包含此前只读准备，不将hash-only算作全文审读）：

- 完整有效Reviewer Role、add-code-review/design分支已取得；同版当前delivery、checklist/abstraction/test-design、安全红线/写作/decisions及go-service适用边界复用。未另读其他Role、未来方法或重新Route。
- 作者Temp的完整actual diff、handoff、actual-readset；r18 before/current/fixed字节及新§4.7.2/相关元数据、载体替换和结论；未影响区块按字节核对。design r21仅hash及既有同版合同复用；原r18 Review仅hash/已读结论复用。
- `audit/store.go:24/50` 的Event/Append/Exec；`command_service.go` 的依赖/构造、Moderate事务/审计、Transfer完整写入顺序、公开proof/ref/VerifyCompletion及错误包装；`deletion_finalizer.go` 的公开构造/Finalize、claim/complete/lease、合法pending/proof分支、DELETE/Audit/Complete与Repository claim/完成SQL；`deletion_journal.go`的公开Journal接口、`model.go`的Article/operation/tombstone类型。
- `identity/article_disposition.go`仅公开生命周期/锁调用、verifier/reference与ClearRetainedEmailIfEligible必要片段；`platform/errors.go`仅PublicError/New/Unwrap。没有读取身份设计/恢复全链、架构或root源码。
- `0001_shared.sql`全文的audit主键/约束/append-only trigger；`0003_article.sql`的operation及Article合法CHECK/FK。trigger拒绝与ACL分开，不重审全迁移。
- `notice_dispatcher_test.go`的配置/现有schema迁移与资源harness；`admin_query_test.go`的precommit fixture/迁移；`postgres_state_test.go`的s22 fixture/IDs/合法数据与AuditFault/回滚重试必要片段。旧s22AuditFault确实真实Append成功后synthetic error，不作为本次INSERT失败证据。未执行这些测试。

核心源码指纹：audit/store=`091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F`；command_service=`585BB51D19B3FCD7BFDC9C84465D2BB5D24F41BD8285553FC20BC10B161B4D68`；deletion_finalizer=`546C69B7CC363453F6A4E866D4E3B1B68D9E6B611463159EDE69ACEC0D9B8750`；identity/article_disposition=`EB053611CE9346A9A51C0485DD9AE1003EA99C57AF0A3842A4621D67ECE04593`。预读与本轮核心三文件指纹相同。

个人静态材料目录：`C:\Users\J00010385\AppData\Local\Temp\han-pauli-audit-r19-review-cf283a4b308348dcbd98884ca3c3784c`。`readset-start.json`记录23项业务输入/指纹（SHA=`C99D807C2597B9C632B270FDD8B9A66B6F6FBF93DCFD15A5A64987673D61CDFF`）；`independent-static.json`（SHA=`534CE4912D0F42372B5325FD47CC7C5EEBB7C6D20B911BF2340892D39A605CD7`）记录独立diff/保留/12requires检查；`end-check.json`（SHA=`32A983A357E4DA5554555C6808003E0773810118AFC69A3635A1C7F94FEA351E`）核读集无漂移、第五文件仍不存在、未启动业务进程。指纹固定不等于23份全文均审读。

## 5. Findings、Gate 与交回限制

新增 findings：**无，CRITICAL/HIGH/MEDIUM/LOW均0**，不制造finding ID。本次没有新方向决定或必改风格建议。已采用的第五载体选择不重开；旧有效Review/任务不重审。未来若出现真实缺陷，回原作者Ohm修订规划，再由原Reviewer核固定修订；本Reviewer不代设计修复。

| 范围 | 结论 |
|---|---|
| r19 §4.7.2及相关元数据/替换 | 有界规划PASS，可交Lead/原Planner登记原卡第五载体与命令；本报告不发实施/运行信号 |
| 新第五文件/四leaf | 核查文件尚不存在，全部待实施/运行；不要求本轮先编译，不给SQL/normal/race PASS |
| 原四contract /236 leaf | 原r03冻结及既有证据保留；不改为本轮执行结果，不替代真实审计/ACL/恢复 |
| A-CON-03应用ACL | 真实应用身份载体/注入位置unknown，独立义务继续pending；不得以测试DSN、superuser、trigger或无UPDATE/DELETE方法冒充ACL通过 |
| 原12requires /S8.2/Case18 | 全部保留；S8.2仍[-]、其局部交付不算完整前置满足；Case18全局隔离/完成开流与整体接收/关闭继续blocked |
| S1.5完整接收/关闭 | 继续blocked，真实层、依赖、独立Review/Lead核证及适用责任不得由本规划PASS代替 |
| 环境与人的确认 | Lead已有只读PG环境时点事实不是本Reviewer取证或LOCALPG START；实际代码/PG窗口另发，人的accepted/checkpoint仍不代签 |

本轮仅执行读取/rg、文件SHA、只读Git分支/HEAD查询和个人Python字节/diff/映射分析；未读DSN/env值、未连接PG/SMTP/服务、未运行Go测试/list/vet/build/race，未修改测试/设计/Tasks/Plan/生产/identity/root，无Git mutation、删除/移动/清理或下级委派。本角色没有启动测试/数据库/服务或持续后台进程，所有本轮只读命令均已结束，无本轮遗留进程；未作全机进程普查。

**stopped：r19两轮独立规划评审完成并停写，交Lead核接收与后续窗口；不把规划当作代码/业务执行通过，不关闭S1.5。**
