# identity password-reset durable fail-closed Design / ADR 独立评审 R01

- 日期：2026-10-24
- Reviewer：add-reviewer（非作者独立评审）
- objectType：复合对象 `design` + `adr`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 当前活动：development-delivery，B6C/S10.1 `blocked-for-design-revision`
- 目标：判断 accepted ADR-0023、architecture r25、identity design r35、stories r11 是否足以作为 S10.1 test-plan 修订与 Task-Planner R 对账的稳定输入
- 综合结论：**CHANGES REQUIRED**

本报告只评审设计与 ADR，不评审当前三文件代码已经修复。`IDENTITY-S10.1-REV-R01-F002` 在代码层继续为 Major OPEN；本报告不关闭 F002、不解除 B6C/S10.1 阻塞、不启动 B6D，也不声明实现、测试、部署或生产就绪。

## 1. 固定对象与修订

| 对象 | 修订 / 范围 | SHA256 |
|---|---|---|
| `.add/project/knowledge/decisions/identity.md` | ADR-0023 | `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978` |
| `.add/project/knowledge/decisions/README.md` | ADR 索引 | `B51C6265BB2493BE3AA2EE0C32425F29974B38A177018DC5D75F72A12437E14E` |
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r25` | `DB6BE0493051519489A4F6451D71CF52CDE086AA338B04FD7AFCBC94EBEB51A5` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r35` | `C7EC77496F4646DD7CA742DC069ED4B28DC0C7CD46BF1E4E46A56B47694C4B31` |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r11` | `24A37DA68EFE6012B4943655AC092600F63C1A1113E77AB7EBD8B9DD11AD5F07` |

上游判断依据包括：

- `.spec/wip/identity-auth/review-code-s10.1-r02.md`，尤其 `IDENTITY-S10.1-REV-R01-F002`；
- 用户于 2026-10-24 对建议值继续推进的授权及 ADR-0023 方案 A 的接受；
- ADR-0011、ADR-0015、ADR-0018；
- identity boundary、现役 requirements 的相关安全与认证合同；
- 当前 password-reset、Login、PasswordChange、Session resolve、idempotency、migration 与 `cmd/han-server` 代码，仅作为真实消费者和迁移/装配证据，不用于降低已确认安全预期。

## 2. 两轮实际检查范围

### 2.1 Design Round 1：固定设计对象直接检查

逐项检查 architecture r25、identity design r35、stories r11：

- operation 是否为唯一安全权威，幂等 succeeded/replay 是否明确从属于 operation；
- SMTP 前 prepared operation、pending credential、job、保护窗口及 stale prepared 语义；
- operation/job 状态机、终态、`consumed`、派生 Session、永久密码能力和 AuditEvent 收敛；
- Login、PasswordChange、Session resolve Gate 的检查点、正常用户 clear、故障 fail closed；
- worker 租约、接管、退避、dead-letter、关闭顺序和运行责任；
- migration、CHECK/FK/唯一约束/索引、旧 writer 停止、混跑阻断及不可安全回退；
- architecture/design/stories 的术语、状态、依赖、交付 Gate 和 F002 状态一致性；
- test-plan 与 Task-Planner R 对账所需的可验证输入是否完整，是否存在不必要抽象或遗漏真实消费者。

### 2.2 Design Round 2：真实消费者与上游依据有界追踪

在允许上限内追踪：

- `internal/identity/password_reset.go`
- `internal/identity/login.go`
- `internal/identity/service.go` 的 PasswordChange
- `internal/identity/bootstrap.go`
- `internal/session/store.go`
- `internal/idempotency/store.go`
- `migrations/0002_identity.sql`
- `migrations/0004_idempotency.sql`
- `migrations/0007_idempotency_response_snapshot_v1.sql`
- `cmd/han-server/main.go`
- `cmd/han-server/lifecycle.go`

重点核对真实幂等主键、当前锁序、Session credential lineage、永久密码写入、现有 Article deletion worker 的 composition/lifecycle 占用，以及设计影响面能否被当前消费者实际承接。

### 2.3 ADR Round 1：ADR-0023 直接检查

检查 ADR 的问题陈述、A/B/C 选项、选择理由、确认来源、状态、owner、影响、与 ADR-0011/0015/0018 的关系、升级/回退、F002/B6C/B6D 状态声明及非完成性声明。

### 2.4 ADR Round 2：采用与引用追踪

核对：

- ADR-0023 编号在 ADR-0001～0023 中连续且唯一；
- `.add/project/knowledge/decisions/README.md` 的全局索引、领域表、场景表三处登记；
- 引用次数 `3` 与 architecture r25、identity design r35、stories r11 三份引用记录一致；
- accepted 来源、长期 owner、旧 writer 停止、禁止混跑、不可安全回退和关闭顺序未被下游弱化；
- ADR 接受未被误写成 F002 CLOSED、B6C/B6D 解锁、实现或验证完成。

## 3. 评审问题结论

| 问题 | 结论 |
|---|---|
| 1. operation 是否唯一权威且幂等不能绕过 | **方向满足，细节未闭环**。ADR/architecture/design 均声明 operation 权威，但 operation 与真实 IdempotencyRecord 的原子创建及精确键绑定不足，见 F001。 |
| 2. prepared/job/protection window/stale prepared | **满足**。SMTP 前持久 prepared credential/job、延迟领取和 stale prepared 禁止重发的安全方向完整。 |
| 3. committed/not-committed/unknown、consumed、Session、永久能力、审计 | **未闭环**。状态机覆盖场景，但缺少合法 successor 与非法 F002 能力的持久 lineage/settlement 表达，见 F003。 |
| 4. 三消费者 Gate、锁序、故障和正常 clear | **未闭环**。正常用户 clear 与故障拒绝语义存在，但锁序合同自相矛盾且缺少无 operation 行的线性化算法，见 F002。 |
| 5. worker 租约、接管、退避、dead-letter、关闭与运行责任 | **部分满足**。单 worker 机制充分；现有 Article deletion worker 与新 worker 的共同生命周期未设计，见 F004。 |
| 6. migration、旧 writer、混跑、回退 | **方向满足，受 F001/F003 影响**。前向迁移、停旧 writer、禁止混跑与不可安全回退已明确，但精确 FK/唯一约束及能力证据 schema 尚不能落地。 |
| 7. ADR A/B/C、来源、owner、README/引用 | **满足**。未发现 ADR 直接缺陷。 |
| 8. design/stories/architecture 一致及规划输入稳定性 | **不满足**。四个 Major 使实现与测试算法不稳定；stories 另有 Q1 过时阻塞文本，见 F005。 |
| 9. 是否误关 F002、解锁 B6C/B6D、虚构完成 | **满足**。五个对象均明确 F002 OPEN、B6C/S10.1 blocked、B6D 不启动，且未虚构实现或验证。 |

## 4. Findings

### BLOCKER

无。

### MAJOR

#### IDENTITY-DESIGN-R35-S10.1-R01-F001 — operation 与真实 IdempotencyRecord 的原子创建和精确身份绑定未闭环

- **位置**：
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:2057-2059,2063-2074,2127-2133,2266-2267,2290,2305`
  - `internal/identity/password_reset.go:231-300`
  - `internal/idempotency/store.go:123-139,158-180`
  - `migrations/0004_idempotency.sql:1-25`
- **违反依据**：
  - ADR-0023 要求 operation 是唯一安全权威，IdempotencyRecord 不得单独放行；
  - 原 R02 F002 要求同键重试、succeeded 冲突和 durable recovery 必须能精确分类；
  - 设计自身要求“operation 应存在却缺失”时 Gate fail closed，并要求 operation 与 IdempotencyRecord 精确绑定。
- **证据**：
  1. design 写明 SMTP 前同事务持久化 operation、pending credential、job及“幂等 scope/key/request digest”，但未明确真实 `IdempotencyRecord` 的首次 claim 行也属于该原子创建集合。
  2. 现役 claim 位于独立事务并在 operation 尚不存在时即可提交（`password_reset.go:272-300`）。
  3. 真实幂等主键为 `(scope_hash,key_locator)`；精确身份还包括 `key_digest,digest_key_version,request_digest,execution_generation`。operation schema 只列出 `idempotency_scope_hash,idempotency_key_digest,request_digest,execution_generation`，缺少 `key_locator` 与 `digest_key_version`。
  4. IdempotencyRecord 本身不保存 `user_id`。若 claim 已提交而 operation 未建立，Gate 无法从不可逆 scope/key 摘要稳定反查目标账号，因而无法实现“关联证据存在但 operation 缺失时锁定正确用户”。
- **触发场景**：
  - 首次 idempotency claim 提交后、operation/credential/job 原子创建前进程崩溃或事务失败；
  - 同键 retry/replay 读取到 processing/succeeded 记录，但无法定位或锁定唯一 operation/目标用户；
  - worker 或 Gate 试图按真实幂等 PK 建 FK、行锁或一致性核对。
- **影响**：
  - IdempotencyRecord 与 operation 权威仍可能脱钩；
  - succeeded/replay 分类不能稳定证明对应 operation 和账号；
  - “missing operation” fail-closed 规则不可实现，migration/FK/查询/锁序也无法稳定规划；
  - test-plan 与 Task-Planner 无法确定 claim/创建事务边界和故障注入点。
- **修复建议**：
  - 明确首次 claim、PasswordResetOperation、exact pending credential、首次 recovery job 的唯一原子事务边界；
  - operation 持久化真实 IdempotencyRecord identity，至少覆盖 `(scope_hash,key_locator)`，并以 `key_digest,digest_key_version,request_digest,execution_generation` 做精确核验；
  - 固定 FK/唯一约束、创建顺序、同键查询/锁定、缺失 operation 的定位方式，以及 claim 已存在时的恢复分类；
  - 增加“claim committed、operation not committed”及相反不可能性测试合同。

#### IDENTITY-DESIGN-R35-S10.1-R01-F002 — Gate 与 worker 的锁序/线性化合同自相矛盾，存在死锁和 Principal 放行竞态

- **位置**：
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:651,2057,2106,2113-2117,2176-2178,2556,2610`
  - `internal/identity/password_reset.go:231-269,346-389`
  - `internal/identity/login.go:1174-1206,2173-2192`
  - `internal/identity/service.go:415-448`
  - `internal/identity/bootstrap.go:236-285`
  - `internal/session/store.go:214-272,374-397`
- **违反依据**：
  - code-review checklist 的并发、锁序、错误路径和真实消费者检查；
  - ADR-0023 的“三消费者统一 fail closed、worker 固定锁序”合同；
  - design 自身要求所有请求消费者与 worker 使用兼容锁序且不得倒序。
- **证据**：
  1. design 一处声明所有消费者与 worker 固定采用 `Operation → User → Credential → Sessions → Idempotency`，worker 条款再次固定该顺序。
  2. 同一 design 的 Login Gate 表却规定“锁定 User 后”检查 operation。
  3. 真实 Login、PasswordChange 与现有 password-reset 路径均先锁 User；worker 若先锁 Operation 再等待 User，会与“User 后等 Operation”的消费者形成循环等待。
  4. 对从未 reset 的正常用户不存在 operation 行；普通 `SELECT ... FOR UPDATE` 不能锁住不存在的行。设计未给出防止“Gate 查无 operation → 并发插入 unresolved operation → 消费者继续验证/返回 Principal”的线性化屏障。
  5. 现役 Session resolve 是无事务普通查询；仅写“每次 ResolvePrincipal 检查 Gate”不能证明读取、operation 插入和 Principal 返回之间无竞态。
- **触发场景**：
  - worker 持有 operation 行等待 User，同时 Login/PasswordChange 持有 User 等待 operation；
  - Login/Session resolve 查到无 unresolved operation，随后 reset 事务创建 operation，而前者继续创建 Session或返回 Principal；
  - operation 缺失、坏状态或查询失败时，消费者无法在统一事务/屏障内完成拒绝与必要撤销。
- **影响**：
  - 数据库死锁、请求失败或 worker 长期无法收敛；
  - unresolved operation 下仍可能短暂返回可授权 Principal 或创建 Session；
  - Gate clear 语义不能稳定实现，竞态测试也没有确定 barrier。
- **修复建议**：
  - 为 operation 创建、Login、PasswordChange、Session resolve、worker 和幂等分类定义一个统一、逐步可执行的锁序与线性化协议；
  - 明确正常无 operation 行时锁定什么稳定对象以阻止并发插入竞态，例如统一先锁 User/专用 per-user anchor 后再按固定子顺序访问 operation，或采用具有同等证明力的条件写/隔离方案；
  - 明确 Session resolve 是否进入 caller-owned transaction、何时锁 User/operation、何时撤销 Session，以及何时才允许返回 Principal；
  - 增加确定性 deadlock、no-row insert race、Login/PasswordChange/ResolvePrincipal barrier 测试合同。

#### IDENTITY-DESIGN-R35-S10.1-R01-F003 — 终态后的合法能力演进与 F002 非法能力缺少持久 lineage/settlement 模型

- **位置**：
  - `.add/project/knowledge/design/architecture.md:561,613,658`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:2063-2074,2097,2107,2123-2132,2248-2272,2556,2700,3019`
  - `internal/identity/service.go:491-605`
  - `migrations/0002_identity.sql:64-98`
- **违反依据**：
  - 原 R02 F002 明确要求不能把 `consumed` 当安全，必须处置派生 Session 与永久密码能力；
  - ADR-0023 要求 worker 原子收敛 consumed 后永久密码能力、Session、幂等与审计；
  - architecture r25 将“能力收敛证据”列为 PasswordResetOperation 数据，但 detailed design schema 应提供可实现的权威表达。
- **证据**：
  1. design 要求 worker 区分“合法 succeeded operation 形成的能力”与“竞态产生的非法永久密码能力”，并在终态关系漂移时重新关闭 Gate。
  2. operation 字段只有用户、exact credential、幂等身份、状态、原因和时间；未定义能力 settlement/version、成功消费证据、永久密码版本或 integrity hold。
  3. User 只有当前 `permanent_password_digest`；PasswordChange 直接覆盖该摘要。
  4. Session schema 仅允许 temporary Session 保存 credential ID；permanent Session 的 `authenticated_credential_id` 必须为空，因此无法从 permanent Session 回溯到 exact reset credential。
  5. operation=`succeeded` 后，用户合法消费临时凭据、写入永久密码并轮换为 permanent Session 是正常 successor state；若只按 operation 成功时快照比较，会被误判为终态关系不一致并永久重关 Gate。反之，若把任意 consumed + permanent 状态视为合法，则无法排除 F002 竞态派生能力。
- **触发场景**：
  - reset 成功后正常 PasswordChange；
  - unresolved/commit-unknown 期间 PasswordChange 抢先提交永久摘要和 permanent Session；
  - succeeded operation 的历史 replay 发生在后续正常改密或下一次 reset 之后；
  - 终态后发现幂等/credential/Session/Audit 漂移，需要按 design 重新关闭 Gate。
- **影响**：
  - worker 无法证明应保留还是失效当前永久密码与 Session；
  - 可能错误保留 F002 派生能力，或错误永久拒绝已正常完成改密的用户；
  - `succeeded` replay、终态完整性处置和后续 reset 的测试预期不稳定；
  - architecture 的“能力收敛证据”无法映射到 migration。
- **修复建议**：
  - 定义持久的 capability settlement/provenance/version 模型，能区分 operation 成功后的合法 successor 与 unresolved 竞态能力；
  - 明确永久密码写入版本、Session lineage 或等价可验证证据如何与 exact credential/operation 关联；
  - 定义终态后 integrity hold/latch 的持久表达、Gate 查询、进入/清除条件、审计和受控处置；
  - 固定历史 succeeded replay 在正常 PasswordChange、后续 reset、Session 轮换后的合法一致性规则；
  - 增加正常 successor、非法竞态、后续 reset 和终态漂移回归矩阵。

#### IDENTITY-DESIGN-R35-S10.1-R01-F004 — production lifecycle 只支持现有 Article deletion worker，未定义两个 worker 的共同装配与关闭

- **位置**：
  - `.add/project/knowledge/design/architecture.md:867,1119,1377`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:2165,2612,3099,3127,3246-3253`
  - `cmd/han-server/lifecycle.go:20-40,61-112,150-205`
  - `cmd/han-server/main.go:2161-2188,2217`
- **违反依据**：
  - ADR-0023 要求同一 `han-server` 增加受控 worker，并规定 worker-before-pool 关闭；
  - 设计应沿真实 production composition/lifecycle 消费者检查兼容性和交付完整性；
  - abstraction-principles 要求抽象服从实际多个消费者，而不是以单数接口掩盖已有所有权。
- **证据**：
  1. 现役 `lifecycleConfig` 只有一个 `worker` 槽位，错误消息、Reconcile/Start/Stop/Wait 及 `closeArticle` 配对均明确属于 Article deletion worker。
  2. `main.go` 已把 Article deletion worker 写入 `composition.runtime.worker`。
  3. identity design 只写“唯一 root 构造 password-reset worker并接入 lifecycle”，未定义如何在不覆盖 Article worker 的情况下同时装配两者。
  4. 两个 worker 的启动 reconcile 顺序、任一 worker 失败时的主错误、停止/等待并发、共享 shutdown deadline、错误聚合、资源 close 顺序和 Article 回归均未固定。
- **触发场景**：
  - 同一生产进程同时启用 Article deletion 与 password-reset recovery；
  - 任一 worker 初始 reconcile 或运行 loop 失败；
  - SIGTERM/server error 触发共享 shutdown；
  - password-reset worker 等待数据库，而 Article 资源/pool 正在关闭。
- **影响**：
  - 当前 composition 无法同时承载两个 worker而不覆盖既有 worker；
  - 关闭顺序和错误优先级可能破坏任一领域的 durable 收敛；
  - Task-Planner 会遗漏 lifecycle coordinator 重构与 Article 回归范围。
- **修复建议**：
  - 明确最小多 worker coordinator 或复合 worker 设计，保留各领域 worker 所有权；
  - 固定两个 worker 的 Reconcile/Start、运行失败触发、Stop/Wait、共享 deadline、错误聚合及 pool/Article 资源关闭顺序；
  - 明确 production config 验证和缺任一 worker 时哪些路由 fail closed；
  - 把现有 Article deletion worker 的 lifecycle 回归纳入 test-plan 与任务影响面。

### MINOR

#### IDENTITY-DESIGN-R35-S10.1-R01-F005 — stories r11 仍把已关闭的 Q1 写成实施阻塞

- **位置**：
  - `.add/project/knowledge/design-workspace/identity-auth/stories.md:133,191-195,403`
  - 对照 `.add/project/knowledge/design-workspace/identity-auth/design.md:85-86,185,3039,3379`
  - 对照 `.add/project/knowledge/design/architecture.md:29,1289`
- **违反依据**：
  - design/stories/architecture 应为 Task-Planner 提供一致状态；
  - design r35 与 architecture r25 均记录 Q1 已于 2026-10-23 拍板并形成可实施合同。
- **证据**：
  - stories 仍写“Q1 阻塞”“Q1 未决前不得标可直接实施”和“初始管理员轮换和首次改密规则未拍板”。
- **影响**：
  - Task-Planner R 对账可能继续错误阻塞 S04/S10 相关切片，或把已确认合同重新当待决定项。
- **修复建议**：
  - 仅同步 Q1 已接受的状态、规则与“实现/验证仍未完成”，不扩展周期轮换、密码历史或其他业务范围。

## 5. ADR 独立结论

### 5.1 ADR 直接 findings

未发现 ADR-0023 的直接缺陷：

- 问题与 R02 F002 的 durable invariant 缺口一致；
- A/B/C 选项及拒绝理由可审计，方案 A 的安全取舍明确；
- 接受来源、日期、状态与长期 owner 完整；
- operation 权威、三消费者 Gate、worker、dead-letter、升级/回退及旧 writer 混跑风险均有明确后果；
- README 三处索引和三次引用记录一致；
- 未把 ADR 接受写成设计批准、代码修复或生产就绪。

上述 Design findings 表明方案 A 的详细落地合同仍需修订，但不否定“采用持久 operation + recovery worker + 三消费者 fail-closed Gate”这一架构决定本身。

### 5.2 ADR Review Gate

**ADR Review Gate: PASS**

限制：该 PASS 只适用于固定 SHA 的 ADR-0023 与索引质量，不代表 architecture/design/stories 综合可进入 test-plan 或任务规划。

## 6. 待决定与非阻断建议

### 6.1 待 Lead 决定

1. identity boundary 当前未同步 ADR-0023。由于本次变化仍位于 identity 内部、未新增公共错误或跨领域写入，未将其登记为阻断 finding；Lead 可决定在详细设计修订时是否做最小边界引用同步。
2. 设计修订后是否把 requirements 中历史 Q1 待确认表述另行同步，应由需求 owner/Lead 决定；本报告只要求固定 stories 不再与现役 design/architecture 冲突。

### 6.2 非阻断建议

1. dead-letter 手工“重试或终结”在后续修订中应固定：可执行角色、允许命令、租约并发保护、审批/审计、禁止直接 UPDATE operation/IdempotencyRecord，以及失败后的 Gate 状态。
2. worker 观测名称和阈值可继续留给 observability/容量阶段，但 test-plan 应先固定最低可观察事实：operation/job/lease/generation/Audit 关联、最老 unresolved、retry/dead-letter 和 worker 结果。
3. 保持当前简单边界：不需要为本问题引入外部消息系统或新微服务；PostgreSQL 内 operation/outbox 足以承载方案 A，缺口是精确契约而非技术栈扩张。

## 7. 未验证风险与限制

- 未运行 Go 测试、race、vet、构建或项目 Manifest 检查；本次对象是设计/ADR，代码只用于真实消费者静态证据。
- 未连接或运行 PostgreSQL，未验证真实 commit-ack-loss、事务隔离、无行锁竞态、多实例租约接管、死锁检测或 migration。
- 未连接真实 SMTP/TLS，未验证 provider 受理、超时、延迟和 stale prepared 的真实保护窗口。
- 未运行部署、真实 Host 或 E2E。
- 未执行 PlantUML CLI/PNG 实渲染，只检查设计文本与引用。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 当前代码和 migration 仅作为消费者证据；F002 仍 OPEN，不表示代码应按本报告直接修改。
- 工作树存在大量合法既有未提交内容，均不属于本报告修改归属。

## 8. Gate 与交回

统计：

- blocker：0
- major：4 OPEN
- minor：1 OPEN
- ADR direct finding：0

**Design Review Gate: CHANGES REQUIRED**

**ADR Review Gate: PASS**

**综合 Design / ADR Review Gate: CHANGES REQUIRED**

当前五对象不足以作为 S10.1 test-plan 修订和 Task-Planner R 对账的稳定输入。交回 Lead 后应先由设计作者定点修订 F001～F005，再对新的固定 revision/SHA 做独立复审：

1. operation/IdempotencyRecord 原子创建与真实键绑定闭环；
2. 所有消费者和 worker 的统一线性化/锁序算法闭环；
3. 永久密码与 Session 的持久 lineage/settlement 及 integrity hold 闭环；
4. Article deletion 与 password-reset 双 worker production lifecycle 闭环；
5. stories Q1 状态同步。

复审需提供：

- 新 architecture/design/stories 固定 revision 与 SHA；
- 原五个 finding 的逐项整改映射；
- 对应 schema/约束、查询/锁序、状态机、composition/lifecycle 和测试矩阵；
- 若改变 ADR-0023 的核心选择、公共契约或边界，则须先更新对应上游决定/成果后再评审。

在复审 PASS 前：

- 不进入 S10.1 test-plan 修订；
- 不进入 Task-Planner R 对账；
- B6C/S10.1 保持 `blocked-for-design-revision`；
- B6D 不得启动；
- `IDENTITY-S10.1-REV-R01-F002` 保持 Major OPEN。
