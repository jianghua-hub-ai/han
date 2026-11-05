# Architecture r32 / ADR-0027 独立 Review R01

结论：**CHANGES REQUIRED；1 项 MEDIUM（ARCH-R32-R01-F001）**。Round 1 未形成直接缺陷 finding；Round 2 沿真实 article Journal 生产者发现实施责任与安全合同不闭合。该结论只针对冻结文档的可交接性，不是实现、安全运行、测试或 E2E 通过，也不替代 Main 接收与人的决定。

**身份、对象与权限**

本轮由用户明确指定的非作者 add-reviewer 执行；本会话没有编写或修改被审三个对象。已完整读取自己的 Role、add-code-review、首轮定位方法，以及 code-review-checklist、abstraction-principles、design-doc-writing-style、decisions-guide、architecture-principles、产品安全红线、可观测性/追踪规范；development-delivery 只用于恢复 review 环节交接。go-service 入口确认适用范围，本轮是文档审查，不激活 Go 编码/构建要求，不加载未启用 profile。没有子委派，也不虚构 Agent/Session ID。

固定工作区 `D:\AID\VSCode\Codex\Han`，main，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，未切分支。ObjectType=architecture/design + ADR + index。共享仓库只读；唯一写入位置为本报告所在独占 Temp。没有源文件、知识、Task、共享 Review 写入。

| 冻结对象 | 版本 | 审前与审后均相同的 SHA256 |
|---|---|---|
| `.add/project/knowledge/design/architecture.md` | architecture-ir01-r32 | `676234D255B2453FA462B11B998F27E51685C373DDDE5172082014D840702961` |
| `.add/project/knowledge/decisions/framework.md` | append ADR-0027 / adr-0027-r01 | `15A6333B9B89B86CF70A6C9596141CD4700A8519B71107AD3DA276AB8BBE6AA0` |
| `.add/project/knowledge/decisions/README.md` | ADR0027 索引/领域/场景 | `AA09924A20534C6CF28EA200F78711ED0F01ABC7986726DB603BECE6B044777F` |

作者交接包为 `C:\Users\J00010385\AppData\Local\Temp\han-arch-r32-80f5dbbb5ffb45bb9c44e14ab9ca97e0`。已读 handoff、verification、readset、精确增量及当前目标对应全文段落；没有把相对 HEAD 的全部存量 diff 混入本次修订。原前像 SHA 采用该包 verification 的来源记录，不把作者前像静态验证冒充本轮重跑。

**Round 1：完整增量与直接缺陷**

审查 architecture 全部增量及必要上下文：revision/采用说明、§6.2.4（538–601）、§7.1（1018）、§8.3（1126–1128）、§9.1.1（1338–1391）、§11（1546–1559）、§12 与图目录/渲染限制。完整检查 ADR0027（framework:759–801）及索引增量；核方案比较、状态/授权来源、迁移与兼容影响、所有权、失败与取消、图文顺序、后继责任及未验声明。

按直接合同逐项推演首装、normal 重启、running→identity_ready→Consume、已消费崩溃、失败/缺行与更晚成功、维护面→普通面切换。没有仅以关键词出现判定正确，也没有要求架构文档先通过不存在的实现或 Go 测试。Round 1 无独立直接缺陷 finding；下述 F001 由真实生产者核查后成立。

**Round 2：真实契约、生产者与消费者**

| 核查链 | 本轮实际证据与判断 |
|---|---|
| 上游职责与唯一权威 | ADR0018/0019 的状态机、短事务、独立来源核对、Begin-key 与唯一不对称收敛；ADR0020 的删除/证据/DTO 边界；identity r49 §4.2、双方 boundary S13–S16。r32:554–560/ADR27:772 保留 identity consumer、唯一 Store 与联合开放业务责任，framework 只承担机制。没有据机制归属推导新业务权威。 |
| Store 与完整 run 集合 | `internal/identity/recovery_run_store.go:17–285` 实读。当前仅按 run Load/Transition，不存在新全量投影；r32:587、597、599 明确新增投影、完整集合及字段 revision，failed/unfinished/缺行优先，不能 latest success 覆盖。新接口不是已实现声明。 |
| Recovery Journal 的只读与副作用 | `internal/recovery/journal.go:67–152,179–199,333–496,611–765`。构造器、Read/VerifyAgainst 均可进入 loadRecords；pending 可触发 rename/remove/CAS。r32:588、601、1389–1390/ADR27:792、794 已禁止 normal 修复，后继必须覆盖构造与读取全路径，不能仅替换方法名。 |
| Article Journal / 两 namespace | `internal/article/deletion_journal.go:50–100,107–190,345–421,524–679` 沿实际 Append→persist/readVerified 检查。当前 article Journal 无 CAS/id/epoch/revision 生产接缝，与 recovery Journal 的 Anchor 接口不同。这是 F001 的事实基础。 |
| published closure 与 fresh currentFacts | `internal/article/deletion_replay.go:225–251`；`replay_evidence.go:316–522`。当前 ReadStatus 刷新 Facts；private findReplayHistory 也筛 fresh Facts。r32:589、593、595 正确要求独立 article-owned 精确历史 port 与发布来源，不授权 identity 查私表，不允许把 candidate 行/旧 current 直接作为历史已发布证据。 |
| Consume 与崩溃窗口 | 完整读 `internal/recovery/coordinator.go:1–231`，从 identity_ready 起；二次及后续 fresh revalidation 比较 ref/digest/completed_at；ready_to_consume 读取精确 consumed 后补 DB，其他更早状态不能自动前滚。r32 将 running 前半段交 identity 执行者，没有把 Consume 宣称为完整恢复入口，也未把所有 commit unknown 泛化为可重放。 |
| 恢复授权与 Auth 消费 | `identity/deletion.go:35–80,132–146,752–796`、`identity/login.go:46–65,1222–1285,1438–1485`、`PasswordResetStore.CheckLockedUser`。canonical 必须 Store running＋maintenance/public-closed 双证明；nil OpenGate 不是公共准入。login 的 mode 只控制 recovery bootstrap，不控制全局 HTTP；普通 Principal 不因 mode 获恢复许可。代码现状不等于本轮代码合格结论。 |
| root / exactrouter / Trace | `cmd/han-server/main.go:369–421,2245–2283,3027–3070,3344–3527`；platform Trace 与 recovery 错误定义定位。当前 root maintenance 拒启，无正向 normal 准入；两 admin GET 在同一 registry、有精确 GET/DELETE、Trace 外层与 Session/AccessGeneral。r32:1342–1344 保留真实 503/Trace/零 Article 下游及正常链兼容；当前拒启不能算该证据。 |
| 排空、worker、requeue 与 join | `cmd/han-server/lifecycle.go` 的 reconcile/start、stop、joinInitial、shutdown、waitLifecycleWorker；实际 worker 列表还含 Article notice。r32:1126、1348–1355 的“全部普通 worker/请求”适用于全部这些 owner，不能只停两种。未真实 join 阻止 Begin/替换；单操作 Journal 锁不等于部署独占；旧 S10.5 受部署禁并发和 launcher 后继接线约束，不扩已闭卡。 |
| 验收与状态边界 | article r22 §8.6/8.7、article TP Case18:330–340、identity TP I-REC-01–10。先定位 TE index 再读 `architecture-test-input.md`，仅作为咨询。未把咨询、测试计划、历史局部 PASS 相加成独立 Review 或 Case18/整卡 PASS。 |

本轮按契约检查了 Store 集合 revision 与两 namespace 前后重核、正常 append 单调前进与恢复 Begin 对 permit 的影响。没有为 revisions 写数值/类型方案：详细设计必须用能覆盖相关变更、不会因 ABA 重用而误判稳定的版本来源；内存计数/MVCC 或文件 hash 不能替代外部防回退。F001 修订前，不能声称 article 侧现有实现已提供这种 revision。

**Finding（按严重度；共 1 项）**

### [MEDIUM] ARCH-R32-R01-F001：article 后继“仅补历史 port”遗漏双 namespace 锚点的生产写入合同

- 对象/修订：architecture r32；关联 ADR0027 r01 的两 namespace 来源稳定性及后继范围。
- 目标位置：`.add/project/knowledge/design/architecture.md:1551`（“article仅补精确历史published closure port”）。关联 :588、:593、:599、:1018、:1134；`framework.md:794,796`。
- 违反依据：本版 §6.2.4 要求 article namespace 可核 id/epoch、CAS revision/committed head、可信历史前缀及前后稳定性；§8.3:1134 要求每 namespace 独立锚点；设计写作准则 §J 要求关键状态写入者、协作与失败合同可实施，不能让下游再猜中央安全责任。
- 已核事实：实际 `ArticleDeletionJournalOptions`/`FileArticleDeletionJournal`（`internal/article/deletion_journal.go:84–100`）没有 Anchor、journal_id、epoch 或 revision 依赖。`Append:345–392` 读取文件链后调用 `persist`；`persist:542–603` 只有临时文件、fsync、link、移除临时文件与目录同步；`readVerified:606–654` 按目录序列校验 HMAC 链，无独立锚点核对。`ReadAllVerified:400–421` 也不产生外部 revision。相反 recovery Journal 已有独立 Anchor 接口，不能把它的实现能力投射给 article namespace。
- 现状与触发：按 :1551/:1018 只增加 article 历史 closure 与只读 snapshot 投影，并保持当前 Article Journal 写者时，即使首装曾把 article 初始 head H0 绑定到可信 genesis/CAS，随后一次正常删除追加 H1 也不会推进任何 article Anchor。下一次 normal 的双 namespace 核验无法取得与 H1 一致的外部 committed head/revision。
- 影响：严格执行中央合同会拒绝这次合法删除后的重启；若为了重启而仅从当前文件链合成 revision、或在 normal 自动重置/补锚，则无法区分合法当前链与被回退的合法前缀，并违反只读/防回退合同。上述是静态调用链与状态推演，不声称已动态复现或存在已发生的生产事故。它不是把“旧失败历史保守拒绝”判成 bug，也不是仅因代码尚未实现就报缺陷；缺陷在于目标必须依赖有写者维护的权威，而明确后继范围只允许补读者。
- 最小修订要求：由原架构作者在本次三文档范围内消除“仅补”限制，明确 article-owned Journal 写者/受控适配层与独立 article namespace Anchor 的维护责任，并纳入全局 runtime 后继范围。固定正常 append 如何提交锚点、何时可向删除调用方确认耐久、pending/写入 unknown/崩溃由谁受控协调，以及既有文件链的 id/epoch/初始锚点如何经可信一次性迁移绑定；不能让 normal 首读自动认可任意现存前缀。保持既有独立 CAS 信任边界、S16/wire、业务 owner 及真实 S4.5 前置，不新造 run authority，不重开 S4.2/S10.5，不在本轮写代码或新 taskID。局部 Go 类型、配置名与精确文件可由 FD 细化，但是否存在写入生产者及失败协议不能留给 Task 选择。
- 复审条件：原作者提供修订后三 SHA 与精确 diff；独立 Reviewer 对同一 F001 核查生产者→双 namespace snapshot→历史 closure→准入的责任闭合，并静态推演首装→正常删除→重启、合法旧前缀回退、append/Anchor 两侧各崩溃/unknown、pending normal 零修复、两 namespace/Store revision 及 ABA。详细设计和后续实施另在获授权窗口留下对应软件/真实环境证据；本轮不要求运行 Go/PG。

没有第二项已坐实缺陷。F001 未关闭，当前冻结稿不作为无未关 finding 的交接版本；Reviewer 未替作者修改，也未替用户接受风险。

**待决定、后继落实与非阻断注意（不混入 findings）**

已有用户 standing 授权足以沿建议方向继续，不请求再次确认。本轮没有额外方案采用决定或风险接受。实际 provider 产品/凭据、genesis 发布主体/ACL、介质、真实锁设施、运行签核人、RPO/RTO/retention 仍 unknown/TBD；不伪造值，也不因此停止独立的软件设计。

构造器也可能修 pending；FD 必须把 r32 的 normal 零修复约束落实到构造全过程。历史闭包必须可证明曾发布，不能直接导出 findReplayHistory。全 worker 列表应从当前 lifecycle 核对，包含 Article notice。以上是已有中央合同的实施注意，不另记 defect；不允许改成可选验收或把观察器/fake 注册为生产 fallback。

对于 failed/缺行历史不能自动 supersede 的可用性限制、缺 genesis 或真实安全前置时拒启、retirement 缺 verifier 时禁用，均按已采用保守规则保留，没有要求放宽。新增类型/argv/ENV、元数据编码与只读初始化的具体写法可在 FD 明确。

**验证来源、未运行项与接收边界**

本轮实际执行：只读 branch/HEAD/status、三个冻结对象前/中/后 SHA 核对、范围内内容读取/符号定位、两轮静态契约/调用链与失败时序审查、47 个实读路径的 SHA 稳定性核查。实际检查和原始范围见 `readset.json` / `readset.jsonl`；部分宽定位输出曾截断，关键目标与消费者已分段补读，本报告不声称所有依赖全文审查。未检查全部业务路由/全库代码、全部旧 Review raw 或真实部署状态。

Main 在本轮追加的现场证据单独归属 Main：151 行 architecture 增量、46 行 ADR 和索引 diff 全读；links/BOM/append-only/1 引用检查；4490 原保护文件逐 hash 0 变化/0 缺失；Main source188=186 树文件+go.mod/sum，作者 source188=188 树文件（含两既有 README），并集190现存文件与旧前像一致。未把口径差异说成新增源码。按 Main 指示，不重复跑这些静态集合，也不把 Main 的检查记成 Reviewer 自跑。

未运行 Go/PG/任何构建测试；未读 DSN、环境秘密或真实 key；未启动服务、listener、worker 或发 HTTP；无网络、安装、Git 写、清理及子委派。PlantUML 未安装按用户/交接限制保留，-checkonly 与 PNG 渲染均 NOT RUN；没有渲染 PASS。真实 POSIX/fsync/跨进程锁、独立故障域 CAS、旧备份/inventory 批准、应用身份 ACL、epoch/retirement、RPO/RTO、部署 E2E 与人的接纳均未由本轮验证。

article S1.5/S4.4/S8.2 仍 active，Case18 整项未通过；identity S4.5 真实 CAS 和 S4.6 retirement 条件不减，S4.2/S10.5 不重开。报告不是 Task 状态更新或接收签字。由 Main 现场核本报告、冻结对象、来源与边界后决定接收并交原作者修订；Reviewer 不代签、不采用、不推进其他角色。

最终对象复核与本报告/readset/verification 的哈希见 `receipt.json`。交回后 **STOP 所有写入**，保留独占 Temp 内容，不做清理。
