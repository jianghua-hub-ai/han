# Article S6.2 r16 独立复审 R02

**DesignReviewGate：PASS（本次定点设计复审）。** 原 `AM-DES-R15-S62-R01-F001` 在设计层 **CLOSED**；N001、N002 的澄清亦 **CLOSED**。两轮检查未发现新增实质 finding。

**TestPlanReviewGate：PASS（仅下文指定的 S6.2 规划范围）。** 测试计划仍为 **planned-not-executed**；不表示代码、测试执行、任务承接或其他历史切片 Gate 通过。

## 一、固定对象、权限与依据

工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本执行者为两份文档的非作者，未委派、未修复、未 Git 写入或清理；唯一新增文件是本报告，R01 保留。

| 对象 | 修订与固定 SHA256 |
|---|---|
| objectType `design`；`.add/project/knowledge/design-workspace/article-management/design.md` | `article-management-design-ir01-r16`；`CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC` |
| 原设计基线 r15 | 本会话前轮实际核读的 `article-management-design-ir01-r15`；`0435103681F17C01474A401872F7D0229EEA1B1A2ADD14A7693658F4553863D6` |
| 原独立报告 `.spec/wip/article-management/review-design-r15-s6.2-r01.md` | SHA256 `4AEC5F04C0D1461E96B175D53AB6951E0FCFD61018A7333623FA870F39BF394C` |
| objectType `design/testplan`（按 design 评审分支消费）；`.add/project/knowledge/design-workspace/article-management/test-plan.md` | `article-management-test-plan-ir01-r14`；`AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C` |

方法与规则复用本会话已取得的完整同版本 Reviewer Role、add-code-review、development-delivery 当前 review 环节、设计 core 规则、产品安全红线及只读取证方法；不重复路由，不加载其他角色或未来活动。

当前交回位置仍是 S6.2 的 Lead/原作者。设计与测试计划的通过不扩大原三文件写集，不替代 Task-Planner 对账、开发授权或人的接纳，不生成 checkpoint/签字。

## 二、Design R02 两轮实际范围

Round 1：针对 F001/N001/N002，对照 R01 及本会话已核读 r15 合同，读取 r16 顶部修订、§3.3.7 必要组件一致段、§3.4.4/§3.4.5 新分支及邻近上下文。重点检查候选来源、CompletionRef、completed operation 回读、计数快照、清理事务、锁序、lease 与无来源 replay。交接声明相对 r15 为 37 行；本轮未另取得完整 r15 物理副本重算全文件 diff，不把原 r14 临时副本冒作 r15。

Round 2：沿原消费者有界复核 canonical 的 pending 选择/计数与 active-only 无归属化、Identity 零集合释放、operation kind/source 不可变约束、T2 completed 短路，以及 worker transient/no-op 分类。R01 证据表中的八个关键代码/schema 文件本轮 SHA256 全部与 R01 相同，确认这里只发生设计修订，不把代码意图当作实现完成。未扩大到全库锁图或恢复专项。

### 原意见逐项结论

| 原 ID / 级别 | 状态 | r16 位置与关闭依据 |
|---|---|---|
| `AM-DES-R15-S62-R01-F001` / HIGH | **CLOSED，设计层** | `design.md:1960`～`:1964` 补 ordinary/violating pending 的 author 候选→Identity NOWAIT→确认 deleted→同事务清理；`:1913`、`:1914`、`:1921`、`:1927` 同步来源与 completed operation 证明，不再要求非 orphan operation.source 非空。`:1938` 明确原三步交错须自动收敛，不依赖再次人工 canonical。 |
| R01 N001 / 非阻断澄清 | **CLOSED** | `design.md:1876` 明确只有 transfer 要求双方邮箱非空；publish 不需目标邮箱/通知快照，仍核原有绑定和成对空值幂等。未削弱 `:1934` 单侧缺失/漂移拒绝。 |
| R01 N002 / 非阻断实施注意 | **CLOSED，合同澄清层** | `design.md:1959` 明确使用数据库实际保存/返回的权威 UTC 微秒 lease，取得 operation 锁后用 fresh clock 核有效期；`:1972` 保留 55P03 的可识别根因与 worker transient 要求，`:1973` 保留已提交 claim、到期后另轮重领。 |

### 原触发与新增风险复核

1. **pending→canonical→ordinary finalizer。** canonical 已提交后，pending Article 的锁内 author 仍可证明关联 U；r16 的新端口 `LockDeletionCandidate`（`:1904`）在当前 tx 锁定 U/Identifiers，确认 deleted 后才绑定 SourceID。T2 完成 DELETE、审计与原 operation.completed，再按同 SourceID 核计数/释放，均在一次 commit 内。原泄漏路径已有明确在线消费者。仍有另一 active/pending 时必须 unchanged；原 kind/source 保留，符合 `migrations/0003_article.sql:70` 的不可变约束。
2. **作者还未删除时的相反交错。** ordinary/violating T2 也锁候选 User，但 not_deleted 不构造清理资格，正常删除并持 User 锁到 commit。随后 canonical 获得身份锁，在文章已删的状态下统计；最后一篇时可由原 Identity 零集合分支释放。该分支未把 active/suspended 作者送入“非 deleted 即错误”的清理入口。
3. **反序锁不新增等待环。** 新路径为 operation→Article NOWAIT→User/Identifier NOWAIT（`:1892`、`:1960`、`:1972`）。canonical 若已持 User 等 Article，T2 取 User 失败便整事务 rollback、释放 Article/operation；T2 若先持 User，canonical 只能等其提交。与 coordinator Article→operation 的竞争同理：T2 对 Article 不等待。其他文章/operation 不为计数加锁。结论只覆盖这几条已声明链路的行锁设计，不证明全库无死锁或表级锁无等待。
4. **来源不能伪造或丢失后猜测。** 前像取自当前 tx 已锁 pending Article，版本/operation ID 重验，可信 provider 把 author 前像与已核 deleted User 共同绑定；完成回读核原 kind/source=NULL、article/version/id、审计及 Article absent（`:1913`～`:1927`）。不能从 actor、tombstone、事务外预读或调用者布尔取得清理权限。Article absent 且 operation 无来源时保持不清理的保守边界（`:1975`），不借该边界跳过在线正常清理。
5. **并发最后两篇。** 来源 User 锁保持到 commit（`:1929`），动作后同一条查询、同一快照同时计 active/pending（`:1928`）；不存在把 active→pending 迁移切在两次独立计数中而误报零的合同缺口。失败方释放锁后以新 tx/新事实重试。
6. **重复清理与失败原子性。** completed operation 先短路，不再清理/追加审计（`:1959`）；未完成 T2 的 DELETE、completed、完成审计与邮箱/Identifier/释放审计共同提交或回滚（`:1964`）。同 tx 重入及成对已空返回 unchanged、不重复释放审计，单边缺失仍报错（`:1934`）。commit unknown 不能沿用旧事实；已提交则清理已经原子完成，未提交则重新 claim/核验。T1/Journal 与已提交 lease 不被 T2 回滚撤销。

现役直接证据仍是 R01 引用的 `internal/article/article_port.go:59`、`command_service.go:473`/`:499`/`:513`、`internal/identity/deletion.go:429`、`deletion_finalizer.go:470`～`:523` 及 `deletion_worker.go:198`～`:214`。现役 finalizer 尚无本次 Identity 清理、精确 lease 与 NOWAIT 完整实现；这些代码证据支持兼容分析，**不构成 CodeReview PASS**。

新增 design findings：**0**。无新增必须由 Reviewer 代选的方向决定。上述关闭仅针对设计缺口；实际 SQL、锁、tx/provider 注入和回滚仍待实施及真实 PG 证据。

## 三、TestPlanReview：独立范围与两轮检查

本部分由用户追加授权，在设计复审后审查固定 test-plan r14；作者为 Ohm，本执行者非作者。没有完整 r12/r13 前文副本，本轮**不宣称已核 r12→r14 精确 diff**，按用户允许的声明新增节作完整局部评审，未伪造或重建测试计划历史差异。

Round 1：读取元数据及 S6.2 修订限定，A-ORP-03～07（`:264`～`:268`）、新增 §4.5.1 十组 PG 矩阵（`:270`～`:289`）、覆盖追溯 `:468`、局部数据环境 `:483`、§9.4 精确候选/命令/停止条件（`:865`～`:890`）、局部执行前 Gate（`:910`～`:918`）与当前结论 `:958`。其他 S4.2/S4.5 历史状态只识别为范围外，不重新评审或关闭。

Round 2：逐项反向映射到固定 design r16 的 F001/N001/N002 及清理/事务合同，再核原 Story 6、本次 R01 hash，以及精确候选文件和既有 harness 的真实声明。仅读取被 §9.4 引用的三个 harness 必要定义、Identity reader 和 `internal/testenv/env.go`；未运行测试、读 DSN、连接数据库或检查整个测试库。

### 核心覆盖与证据要求

| 风险/合同 | test-plan 位置 | 本次判断 |
|---|---|---|
| F001 原三步自动收敛 | `:268`、`:284`、`:468` | ordinary/violating 分别覆盖；明确 T1 pending→canonical 提交保留→最后 T2 自动释放，禁止依赖再次人工 canonical；包含 active 作者持锁、另有 pending、反序竞争、失败回滚及不改 kind/source。 |
| N001 transfer-only | `:266`、`:268`、`:283` | transfer 双有效地址/绑定为必要条件；publish/可信来源删除不附加目标、Notice、SMTP 前置；成对空值与单边漂移分开。 |
| N002 UTC 微秒与 fresh clock | `:285` | 真实 PG 回传权威 lease，非零亚微秒/时区、锁后过期、owner/status/lease 不匹配及有效对照均明确；过期/丢失 claim 零 DELETE/完成/清理副作用。 |
| 真实 55P03 消费及 lease 退避 | `:277`、`:278`、`:284`、`:285`、`:287` | 显式 NOWAIT，不能以 lock_timeout 或死锁检测替代；错误经真实 HTTP/worker 链路消费，HTTP 无 loop、worker transient 而非 no-op；保留 claim/lease，到期后原调度重领。 |
| 当前 tx 与可信 provider | `:280`、`:282`、`:287` | 每字段篡改、跨 tx/已结束 tx/前像不符拒绝；成功对照走 production provider/SQL，回调调用与同连接可观察，不能由 stub 自报 verified。 |
| 同快照计数/最后两篇 | `:279` | 同 tx 同一统计语句同时计 active/pending、author/source 谓词、异常关联不漏、匿名/转移排除；并发以 barrier/锁状态同步，保留每次提交与审计证据。 |
| 双快照、审计、原子失败与幂等 | `:267`、`:281`、`:282` | Notice 四字段/双 kind/地址/pending、零行/present mismatch/absent 区分；逐点故障后全事务快照、completed 与释放审计去重、commit unknown 新事实均有落点。 |
| 无来源 replay | `:283`、`:284` | absent+无来源不猜；有 pending 前像才用 F001 分支；只做局部合同回归，不伪称备份恢复演练或 Ready。 |

十组最小 PG 矩阵与 Story 6/设计章节/既有任务的追溯清楚；组件/HTTP 层补负例与映射，真实多连接/多事务负责锁、SQL 与原子性。未发现把 memory、fixture、编译成功或环境可达当执行证明的新增要求。

### 精确候选与可执行性边界

- `test-plan.md:871`～`:874` 区分原 S6.2 三文件、建议新增 `internal/article/orphan_disposition_integration_test.go`、候选顶层符号 `TestOrphanDispositionPostgres` 以及 repository/finalizer/worker/Identity/composition 接缝；明确未登记、未认领，不据计划自动扩写集。
- 现场核对两个 `orphan_disposition*_test.go` 目前均不存在，与文档“候选、未执行”一致；没有把候选命令评为现成可运行或 PASS。`:886` 明确 integration tag、符号/十组子例前置、skip/零用例不通过，普通 go test 不替代 integration 专项。
- `test/integration/identity/postgres_test.go:43`～`:51` 的完整 migration 列表、`:118` schema 提交、`:125` 每 tx search_path，与复用建议相符。`repository_integration_test.go:948` 的 schema 留在未提交 tx；`deletion_replay_integration_test.go:515` 绑定 pool schema、`:522` 只载部分 migration，不能直接冒充本卡完整跨模块 fixture。`:876` 已明确这些限制及包内 helper 不可直接跨目录 import。
- `internal/testenv/env.go:20` 读取进程环境；计划没有假定自动取得 User-scope DSN。执行前由 owner 注入、隔离 fixture/连接一致性、goroutine/事务退出及具体回收授权均保持待确认。本轮未执行既有 harness，也未触发其 cleanup。
- Story r07 实际 SHA256 为 `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05`；R01 实际 hash 与计划 `:14` 一致。计划对设计 Review 仍标 pending，符合其在本 R02 之前形成的事实；Lead 后续同步本报告引用，不由本执行者改该对象。

新增 test-plan findings：**0**；无阻断建议。**TestPlanReviewGate PASS 仅表示上述新增规划可交 Lead/Task-Planner 承接**，不覆盖历史 test-plan findings，不将其他卡旧 PASS 借给 S6.2。

## 四、限制、交回与停止条件

本轮仅文件/哈希/静态搜索及 Git 分支、HEAD、指定路径 status 的只读核对。**未运行 Go 测试、构建、SQL、PG 多连接实验、SMTP、渲染、部署、E2E 或恢复演练。** 没有因计划足够而声称 provider、NOWAIT、清理或精确租约已实现。

代码与真实 PG 证据仍待后续；原任务、精确写集/新测试符号、fixture 与运行/回收责任需由 Lead/Task-Planner/执行 owner 真实承接。两份文档保持原 draft / planned-not-executed 状态，未生成接纳记录或签字。

后续实施评审需验证原交错、反序 NOWAIT、同快照计数、真实 production tx/provider、原子失败/重复释放及权威 lease；设计若改变来源、锁序、事务或恢复边界，或测试计划变更上述核心期望，须以新固定修订重新核受影响项。本报告不豁免这些证据。

本报告写入后复核两份目标与 R01 哈希；完成即停止写入，只交回当前 Lead。
