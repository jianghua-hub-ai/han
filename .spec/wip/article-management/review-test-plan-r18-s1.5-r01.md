# S1.5 TestPlanReview R01：test-plan r17→r18

2026-10-28。Pauli / 实际非作者 add-reviewer；本修订作者 Ohm 已 completed/stopped。objectType=`design`（测试规划合同），沿原 development-delivery 续接，未重新 Route、未委派。本报告不构成 CodeReview、业务测试、实施授权或人的接受。

**有界 TestPlanReview Gate：PASS，仅覆盖固定 r18 的 S1.5 规划增量；两轮检查完成，无新增 findings。** §4.7.1 足以作为 Lead 判断原四文件中不依赖缺失恢复能力的组件实施子集的输入。四 top 尚未实现/运行，不据此证明 A-CON/A-REC 全部通过；完整 S1.5 接收/关闭仍受原 requires、真实审计 SQL/应用 ACL、适用恢复证据等约束。Case18 整体及 S8.2 关闭继续 BLOCKED。未发 CODE/LOCAL/PG START。

## 1. 固定对象、授权与修订

工作区 `D:\AID\VSCode\Codex\Han`，实际核 `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。唯一仓库写权限是本报告；未覆盖任何前轮报告。

| 固定输入 | 实际 SHA256 |
|---|---|
| 当前 `test-plan.md` r18 / 作者 fixed 副本 | `BCBCFDD8CD1747540726A46C35D565FF1CC211AAB43F6059145474568841DE42` |
| r17 before-image | `93071AB5E9E68ED976FA376EF52874E7B1A74DEE3FA9860561DD46A596A8DFA2` |
| `r17-to-r18.diff` | `4ACAFD95B979381C5B88981F940C1205D637015E2EAD44D69592A950372F5707` |
| design r21 | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA` |
| 前轮 r17 TestPlanReview | `600C9BC2AAECF668D50B9A96090250021B57FDF44AF9965FD6FF2440AB550A3D` |

本轮固定对象及材料由 Lead 交接；作者输入目录为 `C:\Users\J00010385\AppData\Local\Temp\han-s15-testplan-r18-e9baed6003ee445f8d91268b52f58db8`；实际读取完整 diff、handoff、actual-readset，并直接读取 before/fixed/current 字节独立核证，不以作者静态结果代替本轮判断。r17/r21 已审的未变合同复用，不重审历史全 feature。

末核 33 项业务输入指纹：r18、design、证据及源码未发现未知漂移。唯一变化是 Tasks 整体从 `D9054472CE84390844380E3E3C944046421342674DCCBF3D500CEC2565EE53C5` 变为 `28576192AF670683D266909AC57B218B392A703BC8AADDCBAD8395762612601C`，与 Lead 本轮明确同步的 Planner METADATA ONLY 准备认领一致；不把它算作 r18 漂移或代码 writer。本报告不评新 metadata diff，也未读取 Plan；Lead 提供的 Plan `EC832DA255FBA89526E242E043BEDC479C54E747211793E7A76A5F29F2DE3E5D` 仅为同步事实，非本轮独立核证结果。最新准备认领 `[-]` 不表示 CODE START；四个候选文件核查仍不存在。

## 2. Round 1：完整实际增量与保留性

从原始字节独立重建 unified diff，与交回 diff 完全相同，实际 **+113/-7、五个 hunk**。范围限 metadata/历史措辞、来源说明、A-CON-01/04 的层次说明、新增 §4.7.1（当前470–565行）和 §12 的 r18 结论；未夹带既有验收删减。

独立逐字节比较结果：

- §4.3 至 §4.4 前完整 Admin2 Case/恢复分项/RawQuery 合同相同；20 Case 的顺序、正文及 U17/C1/P16/R4 共38映射相同。
- §7.7 Admin2 数据区块、§9.6 九路径/四 top/命令/负控区块相同。§4.8 至 §12 前的全部正文也相同，故原 producer/恢复、数据、分层、命令、资源和历史 gate 未被本增量替换。
- 新增 endpoint 表有16个不同 leaf，S1–S16 表按顺序完整16项；不是用 orphan 两 GET 或裸 DTO 顶替普通管理员 GET。
- 直接核读时原 S1.5 卡恰12个 requires，含 S8.2；r18 没有宣称其局部实现满足完整前置。最新认领 metadata 的保留声明不在本轮重新核验范围。
- 新四 top、后续两个审计 top 均明示候选、NOT RUN；没有编译成功、旧 PASS、作者自审或局部 Go GREEN 冒充本次执行的叙述。

## 3. Round 2：公开接口、行为证明与组合边界

| 规划面 | 实际接口/证据核对与结论 |
|---|---|
| `TestArticleRESTContract` /16 endpoint | 逐项对照固定 design 接口表及现役 Handler 公开构造/方法。基础 creator 结构绑定 replacer/Moderate、独立 member/author/admin/orphan query 和 orphan command 工厂、`PermanentArticleDeleter` 均有现成接缝；无需扩生产 API。16个 leaf 可以用合法受控结果驱动真实 Handler，成功表示与相应拒绝/错误分开，未配置能力500不算业务命中。 |
| HTTP 风险 | 规划保留创建/替换 wire 字节与 UTF-8/严格 JSON、适用 If-Match 428/400/412、错误 envelope/trace/nosniff、同源强 ETag及低敏输出。新 Admin RawQuery 整体422先于字段400/422，授权拒绝零下游；N001 可经真实 QueryService 的仓储 seam 提供非法读状态，再走 Handler 输出500，对照旧治理409；不是只注入 public code。写响应失败组可由现有创建 logger 接缝核已提交后的失败及零二次命令，不改变业务提交语义。 |
| `TestArticleProjectionContract` | 真实 Policy 具有 MemberList/MemberDetail/Author/Admin 四个公开投影入口，允许合法同一 Article 构造不同身份/状态并核 JSON。列表5字段、详情/创建8字段、作者4字段、Admin12字段、匿名嵌套/null/[]和独立逐值 oracle 与固定设计及现役公开类型一致；不需要第二 DTO/投影规则或私有状态机。 |
| `TestArticleInternalPortsContract` | 实际 identity reader 可接受受控 Query/Rows，核 ID 去重、空输入零 query、Query/Scan/Rows.Err/Close；真实 QueryService 可承接消费者差异。`ArticleAccountPortFunc` 确有独立 result 类型转换接缝，不能把 CommandService 当作直接实现。Disposition 接口、真实 command 的 notice 双快照→audit→proof→清理顺序、caller tx/actor/trace、Notice Gate/Mailer purpose和删除端口均有可观察接缝；SQL/锁/网络等更高层仍须消费适用原证据或保持未验。签名/反射与 stub 返回 forbidden 均不能单独计行为通过。 |
| S1–S16 | 每项注明实际消费者、组件或旧证据层及未证明边界。S13/S15/S16 保留恢复隔离、失败不开流和完成开流义务；ProcessingGate、Session 账号限制、startup、组件 envelope 均不能代通用恢复 Guard。不为这四文件指定全局 authority、扩大 S10.5 或分配未知恢复 owner。 |
| `TestArticleReplayStatusProducerContract` | 公开 Provider 构造器要求 Store/Source/IDs/Tokens/Clock；单次受控 Store.ReadCurrent 返回固定 run/evidence/error 可驱动真实 ReadStatus→encoder→共享 decoder。其他方法显式失败的最小替身无需 Begin/Publish 状态机；snapshot callback 参数存在不等于真实 source 已运行。合法 fixture 可通过公开字段转换构造 evidence，唯一 canonical/digest 由 recovery 包提供。Ready20、前16固定顺序/摘要、metadata 排除、Unavailable6及非法 evidence fail-closed均有实际组件路径；consumer mutation 走共享 decoder，不另造 authority。 |
| A-CON-03 真实 INSERT 失败 | 实际 CommandDependencies/FinalizerDependencies 有可控审计 ID generator，真实 audit.Store.Append 使用 caller tx 执行 INSERT。预置合法重复 audit ID 触发 SQL 唯一冲突的建议有可实施接缝；仍须将到达 INSERT、SQL 错误、前后快照、锁释放及合法重试作为未来实证。适用治理/转移与 finalizer 不互相替代；缺真实 Journal 载体的部分仍未验。 |
| 应用 ACL | 权限检查与命中行 UPDATE/DELETE 是独立义务。规划明确真实应用身份/安全注入位置 unknown，拒绝 superuser/owner、trigger、零行成功或“无方法”替代有效 ACL。Lead/DB 环境责任提供既有载体及授权后才能执行；两个 PG top 不得先写 skip/placeholder。未知环境是显式阻塞，不是已实现权限或本轮规划缺陷。 |
| 数据/资源/负控/命令 | 保留合法 CHECK/FK、pending 对应 operation、随机 schema 预先提交/完整0001～0009迁移/连接 search_path及失败 retain；资源 close/rollback/cancel/join 与数据清理分开。四文件内承 helper，不跨包借私有 harness。投影泄露/producer encoder 的可编译 Temp overlay 应触发真实断言失败，first-GREEN 须说明等价检错证据；不认编译错误/硬编码失败/缺环境。四 top 精确-list、普通/race、tagged vet与两审计 top 独立命令、逐 leaf 非零命中和 fail/skip/缺席门禁均清楚，未来须实际取证，不认旧 S2.2 PASS。 |

直接调 Handler 只证明 transport/参数/encoder/错误行为；method/path、安全链和共享实例须由真实 `cmd/han-server` 同包 factory/root 证据承接。r18 没有要求为 contract 导出 main factory 或手造第二 router。四个局部 top 的 GREEN 只能覆盖实际声明集合，不能覆盖缺席的两个审计 top、全局恢复子断言或全部 S1.5 接收条件。

## 4. 旧证据索引的实际核证

| 证据（均已核固定 SHA） | 本轮实际消费层与限制 |
|---|---|
| `verification-s1.6.md` `A5ECED19F435E233EF3C68E02C40CEA0238D70BF0962C6C596CC586C5931F0A6` | 旧14/16 root、真实 Handler受控成功响应/安全链及独立 Erdos Review；旧 main 哈希不当作现役。不是实际 Session SQL/PG/Journal。 |
| `verification-s8.2.md` `42AE7A0A7488EFFC28B7EF9060E94519826D9232FC7EFCEF46FAA0910B44A412`；联合 R02 `F1EE8BF2605C3755835E2A1C0F4DFDA354697BD4EEBE52FE8B8A80E0681B7808` | 已读最新核证和 R02 的对象、Root/PG分层、个人/作者运行与限制。R 普通/race各93节点/76leaf、P与U/C为原执行者证据，本 Reviewer 未重跑。现役 main2 指纹与 r18/联合R02对应，不能用总数顶替16 endpoint/Allow/拒绝顺序逐项映射。Case18 后两项及整卡关闭仍 blocked。 |
| `verification-s2.2.md` `83610991D4DACA09C031FEB5EE6C7F714D1B5C239843851AE9DDC6AEDF261A70` | 对照实际 `s22AuditFault.Append`：真实 Append 成功后才返回 synthetic error；其 rollback/取锁/合法重试断言证明事务回滚，不能证明 INSERT SQL 本身失败，更不证明应用 ACL。r18 对这两缺口的表述准确。 |
| `review-code-s4.2-r02.md` `9D06B84370E196C4881479701EA0B96D6B991A8DD52FB300F964B6061320F3AA` | 完整读原报告并核六个指定 Provider/Store/encoder/source/reader/PG-test 指纹对应原行。抽查 A-REC-05 实际 fixture：真实 FileBackupInventoryReader、ProductionDeletionReplaySource、Postgres Store/事实读取；Journal 与 ReplayAll 为受控替身，manifest 为合成数据。原普通/race、drift→stale→republish 是原报告记录，本轮未运行。作者自审归属及 repository_integration_test 当时记录差异均如实保留，不扩大为当前全部源码/真实旧备份恢复已验证。 |
| S6.2/S7.2/S8.1等索引 | r18 明示具体 leaf/源hash/raw未核到则未验。本轮未追加审查其全量原件，不据此签发这些层或完整 S1.5 PASS，不重开旧有效卡。 |

## 5. 实际 Readset 与取证方式

专业方法：补读取同版 Reviewer Role、add-code-review 完整正文/design 分支、checklist、abstraction、design-writing、decisions、当前 delivery Workflow；读取 add-test-design 作为规划评价依据。安全红线及此前同版安全/追踪规则复用；唯一启用 go-service 的入口与安全加载边界已核，文档评审不套用修改 Go 文件的构建/测试要求。未加载其他 Role、未来活动方法或身份专项设计/ADR0025。

| 实际读取对象 | 范围（hash-only 不冒充正文审读） |
|---|---|
| 作者 Temp 四主输入及 actual-readset | diff/handoff/读集全文，before/fixed/current 原字节比较；未运行作者脚本 |
| test-plan r18/r17 | 完整实际 diff、新§4.7.1及元数据/结论；未影响区块字节核对，既有合同沿用前轮 |
| design r21 | 公共表示/ETag/错误与16接口、S1–S16、可测试性及 source/producer 分工；其余同版复用 |
| Tasks | 直接原 S1.5 卡、12 requires及 S8.2状态行；最新 metadata 仅核整体哈希对应 Lead 同步，不评其 diff |
| handler.go | 公开 interfaces/工厂及16动作、capability绑定、错误出口/ETag/RawQuery、transfer表示和创建写响应失败 |
| policy.go / model.go | state合法性、member/author/admin真实投影及公开字段、replay run/evidence；非完整代码评审 |
| query_service.go | 公开 repo/reader/config/构造器，必要 Admin非法状态与详情顺序；未重审全部 Admin2 实现 |
| command_service.go | 公开依赖/Moderate事务及审计ID/Append、canonical符号、Disposition接口/处置/快照/清理顺序 |
| identity/article_reader.go、article_port.go | 全文；仅这两个 article 直接接缝，不扩身份恢复调查 |
| audit/store.go | Event/QueryRower/Append实际 INSERT |
| notice_dispatcher.go、privacy/store.go | Notice Gate/Mailer依赖、固定purpose/两recipient模板及 ProcessingGate公开构造/RequireOpen；不运行通知 |
| deletion_coordinator.go、deletion_finalizer.go | PermanentArticleDeleter/request及Auditor/IDs/finalizer dependencies定义 |
| deletion_replay.go、replay_evidence.go、replay_status_v1.go | Provider构造/ReadStatus/状态原因、Store接口与ReadCurrent绑定/refresh、真实serializer及canonical使用 |
| recovery/replay_status_v1.go | DecodeReady/Unavailable、严格字段/顺序/类型/不变量、CanonicalJSON/Digest；相关测试仅符号搜索，不独立复审 |
| 两个 article-deletion-replay JSON fixture | 全文字段/固定摘要/variant；不修改fixture |
| deletion_replay_integration_test.go | 实际top/leaf、A-REC-04与05断言及A-REC-05真实source/reader/PG装配；未执行 |
| postgres_state_test.go | 审计wrapper及失败回滚/取锁/重试断言；未执行 |
| 五份旧核证/Review及前轮r17报告 | 上节列明的内容/归属/限制及固定哈希；r17同版复用 |
| deletion_replay_source.go / backup_inventory.go | 仅指纹与S4.2报告核对，不新增source调查 |
| main.go / main_test.go | main仅指纹；main_test仅定向测试符号搜索与指纹。当前Root能力承接固定核证/独立R02，未审其代码正确性 |

现役 root 指纹：main.go=`6B45B5509A2D01C9F5DF0393CA00CAD5C8430353B5F849BF6E71BA3A47C52AC4`；main_test.go=`B53DD34D57AD6A2797E9FC080FD90034FC52CB26675686CB075D3153F193A9BC`。Handler/QueryService/Policy/Model 指纹也与 S8.2 联合R02对应；不是沿用旧 S1.6 源码版本。

个人静态证据目录：`C:\Users\J00010385\AppData\Local\Temp\han-pauli-s15-r18-review-288b0d4e98f344e9a3d42c9f6872cde0`。`readset-first-hashes.json`（33项）SHA=`F6CA6B772C0A689F1B6E51F36DE3B7FC3E5A191B46713B9023A65485766D0931`；`static-independent-r01.json` SHA=`31E13F9F9DB796B409A0A46B800B1446945D358845E66DED01BE53EFFB5DEAE2`；`readset-end-check.json` SHA=`45AC5F9FEF84C5623E4646C9B3D2B642B2C3AEB57BEEE64F0633CE0617ECECBB`。其指纹包括明确标出的hash-only文件，不声称33份全文均审读。

## 6. Findings、Gate 与交回

新增缺陷 findings：无（CRITICAL/HIGH/MEDIUM/LOW均0），不制造 finding ID；没有新方向决定或必改风格建议。已诚实披露的 pending/未实施/NOT RUN不作为设计缺陷。

| 范围 | 结论/继续条件 |
|---|---|
| r17→r18 本次 S1.5 四文件组件规划 | 有界 PASS，足以给 Lead 判断不依赖缺失恢复能力的局部窗口；本报告不发 START |
| 四 top 当前实现与测试 | 四文件仍缺；全部候选未实现/未运行，不给运行PASS，不要求本轮先编译 |
| 真实审计 INSERT失败/应用ACL | 适用真实SQL与身份载体/授权/命中证据仍待落实；两个后续top不得skip/placeholder，无实证不接收对应义务 |
| 完整16root/ACL/S1–S16/producer/source | 原责任完整保留；组件、spy、签名、旧同版证据必须分别归层，不以四top GREEN相加关闭全部合同 |
| Case18全局隔离/完整恢复后开流，Case18整体/S8.2关闭 | 继续BLOCKED；恢复owner/生产承载尚待真实上游落实，不扩S10.5，不自建Guard/authority |
| S1.5整卡接收/关闭 | 原12 requires及全部合同、真实分层证据、独立Review/Lead同revision核证均仍必需；S8.2 `[-]`/局部通过不算前置满足 |
| 最新S1.5准备认领metadata | 不在本轮对象，仅承接 Lead 同步事实；是否另作局部Review由Lead决定 |

本轮只执行只读 Git 分支/HEAD查询、文件读取/rg、SHA与个人 Python 字节/diff/映射分析；未执行任何项目测试、-list、vet/build/race/PG/DSN/SMTP/服务/Browser，未重新验证旧raw或数据库状态，未进行 Git写/清理/删除/移动/下级委派。没有新增业务运行PASS。真实Session SQL、应用身份ACL、完整Journal/旧备份/联合恢复/生产开流、Host/部署E2E/容量及人的accepted/checkpoint均不由本报告取得确认。

**stopped：两轮独立r18局部TestPlanReview完成，报告交Lead。若后续发现真实缺陷，交原Ohm修订固定新稿，再由原Reviewer有界复审；本角色不代作者修复、不关闭任务、不代签人的接受。**
