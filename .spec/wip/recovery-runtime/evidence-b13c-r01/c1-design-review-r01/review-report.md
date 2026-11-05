# B13C C1+C2 两轮独立 Design Review R01

结论：CHANGES REQUIRED（1 MEDIUM，0 HIGH / CRITICAL）。C1 的 snapshot 供证条件与 Runtime 后证顺序存在一处循环前置，见 B13C-DESIGN-R01-F001。C2 新排序合同未发现新增设计缺陷；既有实现失败仍 OPEN，不是本 Reviewer 新发现或本次已修复。此结论仅覆盖固定两设计差量，不代表人的接纳、技术替代决策、实施完成或完整安装/运行 PASS。

## 固定对象与归属

- objectType：design；Workflow：development-delivery / 独立 design Review；本会话实际执行 Reviewer，非 Banach 自审，无子委派，无虚构 Agent ID。
- workspace：D:/AID/VSCode/Codex/Han；branch：main；HEAD：6da8abbceb0fa197364810a92a8fa2d97b38ddca。开始与交回再次核对。
- 对象1：.add/project/knowledge/design-workspace/identity-auth/design.md，r53，SHA256 DD61F4B89C94CE91F1E1317056ACAC1D7F8F464B2963DFDC59FF9C358590A6F0。
- 对象2：.add/project/knowledge/design-workspace/article-management/design.md，r24，SHA256 69C56796285F8187FBF2261D90183A519962A7F3EAF0A7788D558370C7CAC619。
- 精确基线分别为 author pack 的 identity-auth.before.md（5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB）和 article-management.before.md（2223BE4D005753FBF8D8AE0116F6926A7E8C044482BCBB0FADAB811677066350）。live 与对应 after 字节相等；before→after 的完整 diff 经独立重新生成核对，换行规范化后与作者保存 diff 相等；确认尾部字节相等。
- 作者来源：Banach / add-feature-designer / 01a18389-57ab-7b70-8c86-82a61e7729f1，来自 receipt/index 的归属陈述。本次 index.json 是 Main wrapper；未发现同目录 source-index。未把 wrapper 当作者原产物或人的签字。
- 源码、SSOT、Task/Plan、规则零写；所有本次产物只在本报告所在新 OS Temp han-*。未清理、删除、移动或覆盖 peer 内容。

## 两轮实际范围

Round 1：完整 identity-auth.diff（160行）、article-management.diff（118行），并独立重算全部 before/after 差量。语义审查受影响的接口/依赖、动作分流、install grant/checker、cold open、bound 重入、追加成功边界、异常/关闭、最小验证与责任映射。identity 正文重点 1430–1472、1574–1688、3540–3616、3800–3888；Article 正文 2270–2382；元数据、可测试性、任务映射其余新增段完整按 diff 检查。完整对象仅做字节完整性处理，未把未改两份设计全文算成语义全审。结论是差量有一处前后置矛盾，其余目标未见新增可定位缺陷。

Round 2：有界追直接生产者/消费者。实际读取 internal/recovery/runtime.go 的 NewRuntime、beginAction/RunNormal、opaque grant/PhaseContext/checkers、Acquire、Complete/Fault、revoke/drain/Stop，以及 Enter/Resume 的关闭/独占先例；admission.go 的 readProofAttempt、pending 拒绝、双 namespace/Store revision 重核；journal.go 的中性 snapshot/Anchor 类型、writer anchor 深拷贝和 lock/I/O 收口。实际阅读 runtime_test.go 中公开 producer/mode dispatch、own-cause、phase scope、跨实例/过期、取消 Stop、真 join、独占不早释和 startup ctx 撤销的对应测试正文，未执行测试。

Article 使用 a47-core-partial-r01 的冻结 after 源码：anchor.go 的构造、Append、withLock、appendNew；snapshot.go 的纯 snapshot、全链和 pending 拒绝；flock.go 的 namespace Lock/Check/Close；anchor_test.go 的真实 NewRuntime/RunNormal、实际完成转发、late-cancel、late-close 和 join 片段。核 source-freeze 七 present 哈希及第八 PG 文件缺失状态、complete-eight-source.diff 的路径/hunk 结构；该大实现 diff 仅做来源/路径与定向锚点对应，不声称整份实现 code Review。原 c2-main-observation-r01 两 raw 全文读取且哈希核对，未重跑失败探针。ADR-0027 r02 精确段 760–803 作为上游边界依据。

本角色自行取得完整 reviewer.toml、add-code-review、code-review-checklist、abstraction-principles、design-doc-writing-style、decisions-guide、产品安全红线、当前 delivery Workflow；首次 Route 仅定位为既有工作接力。未使用父会话阅读作为本角色取得正文。对象为文档差量，不加载 Go 实施/构建前置或未启用 profile，不以无 Go 运行判设计失败。

## Findings（按严重度）

### [MEDIUM] B13C-DESIGN-R01-F001：控制 handle 的 snapshot 被它要支撑的独立后证反向阻塞

- 对象/修订：Article design r24；与 identity design r53 的跨文档合同一致性。
- 主位置：.add/project/knowledge/design-workspace/article-management/design.md:2333，新增 cold-open 段末句要求“只有后续原状态表收敛及独立后证成立才能提供 committed snapshot”。
- 关联位置：同文件 :2365、:2380；identity-auth/design.md:3882、:3885。这些位置要求 executor 返回前接入同一 handle，Runtime 后证读取其纯 VerifiedJournalSnapshot，然后才能认定独立后证成立。
- 现状：snapshot 成功提供被写为后证成立的前置；后证成立又必须消费这个 snapshot。绑定转发只解决构造/实例引用循环，没有消除这项数据供证条件的循环。本文未将这是代码现状或已运行死锁；这是新设计接口前后置互相冲突。
- 触发：一个来源正确、closed/drained/exclusive 成立的 bound 同 manifest Install，真实控制 opener 已取得 namespace 锁，CoordinateAnchor 已合法收敛到 H0 或 H1；executor 绑定同一 handle 后返回，Runtime 首次调用该 handle 的 ReadVerifiedSnapshot 以作独立后证。
- 影响：逐条实现 Article :2333 的门槛会在这个本应成功的路径拒绝或等待尚未成立的后证；实现 identity :3882 则须自行忽略该门槛。两个 owner 无法据当前文本唯一实现 C1 正向闭环。该问题不授予普通业务访问，不声称权限绕过，故列 MEDIUM。
- 违反依据：code-review-checklist §2（前后置/公共契约一致）、design-doc-writing-style §J（可实施的责任、接口、失败/恢复合同）、abstraction-principles §1/§5（状态归属与可验证协作）。本次明确目标还要求受控 cold-open 构造与真实 Runtime→consumer 闭环。
- 最小建议：由原作者将“允许产生供证 snapshot”和“Runtime 接受独立后证”分开写清。真实本地协调/完整纯读核验收敛后，同一控制 handle 即可提供观测值给 Runtime 后证；Runtime 验完独立 Anchor、snapshot、原批准前缀及未变 Store/recovery/commitment 后才接受本次 Install。若原句的“独立后证”本意是 article 局部文件/Anchor 回读，应明确该 owner 和范围，避免与 Runtime 后证使用同一词。保持 normal 入口拒 pending、grant/锁与不开放合同；不需要新 authority、公开 setter、源码或架构改动。
- 验证/复审条件：原作者给出新固定设计修订/哈希和精确 diff。纸面逐调用追 bound pending H0→H1、撤回 H0 新 revision、committed 精确只读，证明首次 ReadVerifiedSnapshot 不需要先有由该读产生的 Runtime 后证结果；Runtime 后证仍不得跳过，故障/漂移仍关闭。实施时由原 owner 在授权窗口补真实 NewRuntime→Install→真实控制 opener/CoordinateAnchor→snapshot→Runtime 后证的先后证据，无 private phase 或 fake Runtime；不要求先完成 S4.13/root/PG。复审仍引用本 finding ID。

## 已核对且未另立 finding 的合同

| 主题 | 本轮独立结论与边界 |
|---|---|
| Install producer / 分支 | 设计明确唯一公开 Install、独立批准/commitment 和 bound 同 manifest 分支；现役 Runtime 接口仍无 Install，这是此次拟补的已知实现缺口，不自动重开旧 HIGH。fresh/prepared 完整实现归 S4.13。 |
| 绑定与 authority | 同 backend/实际 key、原批准初始前缀、完整 Store/namespace tokens、opaque 当前 grant/ctx/精确 install/id/epoch、Pending 深拷贝已有明确目标合同；Checker 返回起始观察不能作为可重放授权。实际 checker 尚未交付。 |
| 关闭与资源 | 文本覆盖普通/全部维护、三 worker/子工作、当前控制流程排除、自等待禁止、mutex 不跨 I/O/join、取消/Stop 撤权与真实 join、不早释锁。真实 production drain/wiring 仍 S4.16；本轮没有验证实现。 |
| 协调状态 | 原 H0/H1/唯一候选、撤回新 revision、第三头/ABA/ACKunknown 拒绝与不重放保留。snapshot 供证顺序只受 F001 影响，不建议放宽其他门槛。 |
| C2 完成边界 | 设计要求所有 fallible I/O/check/namespace Close 在 Complete 前；接受前取消拒绝并 Fault，成功后固定耐久返回，不再 Fault 已完成 token；不清他人 cause、不授后续 Finalizer 权限。与实际 Complete 删除 token/后 Fault 拒绝机制相容，无需新 API。 |
| C2 原失败 | 两 raw 是原作者/Main 观察，本 Reviewer 只读核对。正常/race 各117 PASS/2 FAIL/0 skip 仅引用其原 receipt/status；代码仍 OPEN，本次设计结论不把 FAIL 改为 PASS。晚取消原断言在采用新成功线性化后应保留为历史失败，并另补成功边界两侧验收，不能只删/弱化测试。 |
| 安全与范围 | 差量未替换 wire/manifest kind/批准主体、S49 字段/canonical/metadata、F001/F002 的原业务合同或 run authority。冻结 Article8 与先行 Runtime 两源/完整 S4.13/S4.16 的责任划分清楚；未评活动 peer 六源或重新跑其 Planning Gate。 |

## 待决定、建议与证据限制

无新增架构/批准主体决策由本 Reviewer 代选。F001 是同一目标合同的顺序澄清，修复方向是否采纳交 Main/原作者；不因 standing 推荐值授权而由 Reviewer 改对象。人的精确接纳仍 unknown，两设计仍 draft。无其他风格或非阻断优化要求。

NOT RUN：全部 Go/build/test/race/vet、PG/SQL、HTTP/服务、完整 fresh/prepared Install、root/真实维护 drain、部署、真实批准/provider、POSIX/fsync 耐久、PlantUML 语法/渲染、性能/E2E。没有秘密 ENV/DSN/key 原文读取、网络、下载、Git 写或子委派。只运行只读 Git 身份/status 查询、定向文本读取/检索与 OS Temp 内静态完整性分析。

剩余实现风险保留：current Runtime 无 Install/精确 coordination checker；C1 正向及 PG 未交付；C2 Close/check/取消排序、Close 未知时实际持有 namespace/deployment 资源、并发 Stop/CAS 的 join、其他 cause 与 WorkLease 边界必须由后续真实组件验证。不能从本设计文字推导这些已运行成功。本轮不扩大到 S49 或全库安全/实现审计，不给未验证范围 PASS。

## 取证文件与交回

- readset.jsonl：每次经读取助手取得的真实文件、SHA256、行范围、目的与时间；是实际读取范围记录，初始有工具显示截断的批次已对相关必读/差量关键范围重读。仓库搜索为有界定位；一次 PowerShell 正则引号导致输出额外路径定位行，未将那些路径作为语义审查输入或全库覆盖证据。
- static-readset.jsonl：完整字节分析的路径/哈希，与语义阅读区分。
- static-integrity.json：Main wrapper 11个 artifact 哈希/长度，两个 live/after 固定哈希、独立 diff 重算、确认尾部、diagram/fence 序列，七个冻结 Article源及缺失第八路径，直接 Runtime源与失败 raw 的精确哈希。
- receipt.json：本次结论、写入边界、最终固定对象/直接 Runtime源稳定性与报告哈希；artifacts.sha256.json：本次自有产物摘要。

STOP。交回 Main 接收本报告并安排原 Banach 修复/解释 F001；本 Reviewer 不继续修改、采用、签人的确认、开启源码窗口、关闭代码问题或推进 Task/Plan。原作者修订后仍须固定新版本并引用 B13C-DESIGN-R01-F001 复审。
