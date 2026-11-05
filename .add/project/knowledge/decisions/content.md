# 文章与内容治理领域决策

本文件承载文章数据模型、可见性与内容治理类决策。领域边界由项目架构定义。

---

## ADR-0010: 文章可见性正交权威状态与作者归属可空 (2026-10-19)

**目的**: 数据模型 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-19 明确选择正交权威状态模型；业务依据为 `requirements-ir01-r13` checkpoint `decision-bb5fccc92fcd78ed0648c940f21c8edb59c8e7db72e40cc7b3ef4a388115298b`（artifact digest `sha256:8c68b38bd66f53b6f7ea9386def25c3fce7d34e40aee320e8574ac84034f13a9`）。已接受的 `requirements-ir01-r14` checkpoint `decision-f167ac06ba4e751c2fa8681835c58c6dba2d8fea758ba86e44a9ce63d0f25159` 继续采用本决定，并确认删除后进入 `held`、首期无草稿或下线状态。

**修订关系**: 本次修订以同一编号替代原“单一互斥 `visibility` 枚举”结论；原结论中 `author_id` 使用可空外键表达无归属的部分继续有效，不新增竞争 ADR。

**问题**: 作者停用、违规判定与无归属隐藏可以同时成立，解除其中一个约束不得清除其他约束；同时，用户删除后文章必须保留。文章应如何保存各自的权威状态并派生最终可见性？

**决策**: 文章使用 `publication_state`（发布意图）、`violation_state`（违规事实）和可空 `author_id`（作者归属）作为文章管理模块的正交权威状态，作者账号状态由身份认证模块的 `User.status` 权威持有。`User.status` 的首期值为 `active`、`suspended`、`deleted`，其中 `deleted` 是不可恢复的账号墓碑。成员有效可见性与全部隐藏原因由这些状态派生，不持久化单一互斥 `visibility` 枚举。任何作者状态或归属变化都不得自动清除违规状态。

### 现状 / 痛点 / 改造 / 效果

原模型把作者停用、违规和无归属视为互斥原因，并以单一 `visibility` 保存当前原因。`requirements-ir01-r13` 已确认这些约束可以叠加：恢复用户只能移除停用约束，删除作者必须保留违规判定，违规无归属文章解除违规后仍需单独恢复公开，转移作者也不得清除违规。

单一互斥枚举会覆盖仍然有效的潜在约束。例如公开匿名文章被判违规后，若只保存 `hidden_by_violation`，解除违规时无法判断它原本应恢复匿名公开，还是继续保持无归属隐藏。必须保存发布意图、违规事实、归属和账号状态各自的权威值。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 正交权威维度并派生可见性 | 分别保存发布意图、违规事实、作者归属与账号状态，读取时计算有效可见性和原因集合 | 不丢失叠加约束；状态所有权清楚；停用/恢复无需批量改文章；无组合爆炸 | 文章读取需结合作者状态；隐藏原因不能从单字段取得 |
| 文章内冗余正交标志 | 在 Article 保存作者停用、违规、无归属等独立标志 | 查询简单；可直接看到多个原因 | 重复 `User.status`；停用/恢复需批量更新文章；失败后可能漂移 |
| 组合枚举 | 枚举 `hidden_owner_violation`、`hidden_orphan_violation` 等合法组合 | 单字段查询；数据库检查直接 | 状态数和转换矩阵随原因增长；混合身份与内容治理所有权 |
| 持久化隐藏原因集合 | 以枚举数组或子表保存活动原因集合 | 原生表达叠加；新增原因不产生组合枚举 | 集合重复来源事实；跨模块同步与查询约束更复杂 |

项目选择正交权威维度。模块化单体与单一 PostgreSQL 使文章读取能够在同一查询边界结合作者状态；文章查询本就需要读取作者信息，因此该劣势不引入分布式一致性。无法从单字段取得隐藏原因是可接受且必要的，因为业务已确认同一文章可以同时存在多个隐藏原因。

作者归属继续使用可空外键。需求 `US-IDENTITY-03.7` 要求用户删除后文章保留并转为无归属，`US-IDENTITY-03.8` 要求无归属文章可以恢复公开、转移或永久删除。删除用户时，身份认证模块保留不可登录、不可恢复的账号墓碑及专用 `retained_email`；文章以 `orphaned_from_user_id` 关联该墓碑，供后续无归属转移取得已删除原作者通知邮箱。该关联不是当前作者归属，也不参与有效可见性计算。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 作者外键可空 | 删除用户时把 `author_id` 置空 | 文章保留且归属明确为「无」；无需额外表 | 查询展示作者时需处理空值 |
| 级联删除文章 | 删除用户时连带删除文章 | 无空值 | 直接违反 `US-IDENTITY-03.7`（文章不删除） |
| 保留指向已删除用户的引用 | 软删除用户，文章外键不变 | 保留历史归属 | 「已删除」与「无归属」两种语义混在一个字段，恢复公开时的「匿名」判定需额外区分 |

**场景特征匹配**：本项目的发布意图、违规事实、作者归属和账号状态由不同动作、不同模块改变，且约束可以叠加。正交状态使每个动作只改变自己拥有的事实；可空作者外键直接表达删除作者后文章仍存在。

**劣势可接受性**：有效可见性查询需要结合 `User.status`，但当前为单数据库模块化单体，且文章管理模块已有读取作者信息的真实需要。跨实体条件不能全部由一个字段的数据库检查约束保证，因此必须以单一策略入口、事务和集成测试验证；项目尚无容量证据，不为未经证明的查询性能问题复制状态。作者空值由展示层统一映射为“匿名”。

### 约束

成员可见当且仅当文章存在、`publication_state = published`、`violation_state = clear`，并且文章无作者或作者 `User.status = active`。管理员按治理权限可以查看未永久删除的隐藏文章。当前隐藏原因派生为：

- `violation`：`violation_state = violating`；
- `author_suspended`：`author_id` 非空且作者状态为 `suspended`；
- `orphan_hold`：`author_id` 为空且 `publication_state = held`。

十二项关键不变量：

1. `violation_state` 属于文章事实；停用、恢复、删除作者和转移作者均不得自动改变它。
2. `User.status` 只能由身份认证模块写入；文章管理模块只读取其判定结果。
3. `Article.publication_state`、`Article.violation_state` 与 `Article.author_id` 只能由文章管理模块写入。
4. 解除一个隐藏约束只能移除该约束，不得顺带移除其他约束。
5. 成员可见性必须按上述公式派生；调用方不得提交或覆盖有效可见性。
6. `author_id IS NULL` 不等于必然隐藏；公开的无归属文章以匿名作者展示。
7. 删除作者时必须在同一业务事务内置空 `author_id`、把 `publication_state` 置为 `held`，并保留 `violation_state`。
8. 永久删除是终态，不能通过任何发布意图、归属或违规状态转换恢复。
9. 首期合法状态中，`publication_state = held` 只用于无归属文章待处置，因此必须同时满足 `author_id IS NULL`；`author_id IS NULL` 与 `publication_state = published` 表示已恢复公开的匿名文章。
10. 用户删除写入不可恢复的 `User.status = deleted` 账号墓碑并保留专用 `retained_email`；该状态不能登录、不能恢复，也不能成为文章转移目标。
11. 删除用户产生的隐藏无归属文章必须保存 `orphaned_from_user_id` 并指向对应删除账号墓碑；恢复公开、转移或永久删除完成后，该文章不再属于待处置集合。成功转移时必须先把通知邮箱快照写入 `TransferNotice`，再清除该来源关联。`orphaned_from_user_id` 不等于作者归属。
12. 首期 `publication_state` 只允许 `published` 与 `held`，不提供 `draft` 或 `offline`；新增发布状态须重新评估派生规则。

状态转换：

- 停用与恢复作者只改变 `User.status`；文章状态不批量改写。
- 判定或解除违规只改变 `violation_state`，随后重新派生有效可见性。
- 删除作者把 `User.status` 置为 `deleted`，并在同一业务事务内删除会话、置空文章 `author_id`、把 `publication_state` 置为 `held`、写入 `orphaned_from_user_id`；`violation_state` 保持不变。
- 无归属文章恢复公开要求 `violation_state = clear`，并把 `publication_state` 从 `held` 改为 `published`、清除 `orphaned_from_user_id`；该文章完成无归属处置并以匿名作者展示。
- 转移只允许 `author_id IS NULL` 的无归属文章，目标必须是正常且未停用用户；成功后写入新 `author_id`、把 `publication_state` 置为 `published`，并在保存已删除原作者通知邮箱快照后清除 `orphaned_from_user_id`。非违规文章立即公开；违规文章继续隐藏并保留违规判定。
- 标题、正文长度与格式属需求 Q4；首期已明确无草稿或下线状态。API 是否暴露派生隐藏原因由详细设计决定，本 ADR 不预设。

### 引用记录

- 2026-10-19 架构设计 `architecture-ir01-r01` 采用原结论：单一互斥 `visibility` 枚举与可空 `author_id`。来源：当时已确认需求与架构阶段数据模型决定。
- 2026-10-19 架构设计修订采用本结论：`architecture-ir01-r02` 以正交权威状态替代单一互斥枚举，保留可空 `author_id`。来源：当前用户明确技术决定及 `requirements-ir01-r13` checkpoint `decision-bb5fccc92fcd78ed0648c940f21c8edb59c8e7db72e40cc7b3ef4a388115298b`。
- 2026-10-20 架构修订 `architecture-ir01-r03` 继续采用本结论，并补齐删除账号墓碑、无归属来源关联、`held` 合法状态及首期无草稿/下线约束。来源：已接受的 `requirements-ir01-r14` checkpoint `decision-f167ac06ba4e751c2fa8681835c58c6dba2d8fea758ba86e44a9ce63d0f25159`。
- 2026-10-20 架构修订 `architecture-ir01-r04` 继续采用本结论，并按 r15 对齐无归属处置完成与专用通知邮箱清理前置关系。来源：已接受的 `requirements-ir01-r15` checkpoint `decision-09cd3493c4eced164f39341156ecd422a1eec46d1837f115da8d55bc013ca532`。

---

## ADR-0020: 业务数据库外 Article Deletion Journal 保证永久删除跨备份恢复 (2026-10-20；修订 2026-10-25)

**目的**: 数据模型 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 批准 Article Deletion Journal 方案；2026-10-25 又明确“待确认或决策项采用建议值并继续”，据此接受本 ADR 的 production backup inventory authority/provider/config/manifest 原位修订

**修订关系**: 保留既有 Journal、ReplayStatus tagged union、canonical fixture 与 evidence 决定；本次只补齐其必填 inventory 输入的生产正向路径，不新增竞争 ADR，不改变 boundary、公共 REST 或 retention/RPO/RTO 数值。

**问题**: 物理删除发生在数据库备份之后时，如何防止恢复旧备份使已永久删除文章重新出现？

**决策**: 在业务 PostgreSQL 恢复域之外建立 Article Deletion Journal，并由 article-management 持久化不可变 `DeletionReplayEvidence`。重放状态绑定 `run_id + backup_ref + namespace`；identity 只消费版本化状态和摘要，不读取文章内部表。备份/灾备运行责任角色是 production backup inventory 的唯一生成与签发 authority；它在业务 PostgreSQL 恢复域外生成严格版本化 JSON manifest，由部署/恢复责任角色以只读单文件挂载交付，article-owned `BackupInventoryReader` 只验证消费。唯一生产装配位于 `cmd/han-server`，配置键为 `HAN_ARTICLE_BACKUP_INVENTORY_PATH`。删除流程及物理删除成功条件保持不变。

### 现状 / 痛点 / 改造 / 效果

仅物理 DELETE 会被旧备份复活；在业务库保留软删正文又违反永久删除和不保留内容副本。外部最小 tombstone 可以在恢复开放前重放删除，同时不保存文章内容。

| 方案 | 优势 | 劣势 |
|---|---|---|
| 外部最小 tombstone Journal | 可跨备份恢复重放；不保存内容 | 引入跨介质删除 finalizer 与保留管理 |
| 业务库软删除 | 查询简单 | 旧备份仍回退；保存内容副本并形成恢复能力 |
| 仅依赖备份退役 | 无新增运行组件 | 无法保证所有旧备份及时退役，删除语义不可验证 |

production inventory provider 的候选权衡：

| 方案 | 结论 | 理由 |
|---|---|---|
| 恢复域外、只读挂载的严格 JSON manifest | 选定 | 无网络或新依赖；可离线验证、原子替换并绑定精确 backup/cutoff；符合现有模块化单体 |
| 远程备份 API | 拒绝 | 引入新服务、网络、凭据、超时和可用性协议，超出当前授权 |
| 业务 PostgreSQL 表 | 拒绝 | 会随恢复点一起回退，不能成为恢复输入 authority |
| article S4.5 synthetic fixture/stub | 拒绝 | 只用于测试和 runbook，不能证明生产备份事实 |

### 约束

- tombstone 仅含 `article_id`、`article_version`、`deletion_operation_id`、`deleted_at`、`key_id/version`、阶段和校验链；禁止 title、body、内容摘要、作者邮箱或其他 PII。
- 授权与 `If-Match` 成功后，事务将 Article 置 `deletion_pending`；该内部状态对成员和管理员均不可见，不是可恢复业务状态。
- Journal 原子追加并 fsync 成功后，finalizer 幂等物理 DELETE Article，并同事务写 AuditEvent 与 DeletionOperation 完成。
- API 只在 Journal tombstone 耐久且物理删除完成后返回 204。
- Journal 成功但数据库删除失败或崩溃时返回失败，Article 保持不可见的 `deletion_pending`；finalizer 必须继续完成，禁止恢复公开。
- Journal 失败时不得返回成功；事务回滚或保留可安全重试的不可见状态，不得产生重新可见窗口。
- 恢复旧备份后、readiness 开放前，重放全部有效 tombstone，幂等删除恢复出的 Article；与 Recovery Journal/账号恢复门禁共同完成后才开放。
- backup inventory 的事实 authority、生成与签发 owner 是备份/灾备运行责任角色；部署/恢复责任角色负责只读文件挂载和精确路径交付；article-management 只读验证，禁止写回、补全、推断或把业务 PostgreSQL、远程服务、缓存、目录扫描结果、调用方计数/摘要、stub/fixture 作为第二 authority。
- 唯一 production provider 是 article-owned 文件 `BackupInventoryReader`，只在 `cmd/han-server` 构造并注入唯一 `ProductionDeletionReplaySource`。环境变量 `HAN_ARTICLE_BACKUP_INVENTORY_PATH` 必填且无默认、别名或配置文件回退；值必须无首尾空白、为绝对路径且 `filepath.Clean` 后字节不变。恢复能力可以延迟到调用时打开文件，但 composition 缺 reader/source 或出现第二 source/global/service locator 时 production Gate 关闭。
- v1 顶层字段只允许且固定顺序为 `inventory_version,inventory_scope,backup_ref,backup_cutoff,generated_at,recoverable_backups,inventory_digest`。`inventory_version="article-backup-inventory/v1"`；`inventory_scope="recoverable-backups-before-cutoff"`。
- `recoverable_backups` 必须是非空数组；每项只允许且固定顺序为非空 `backup_ref,backup_completed_at`。数组按 `(backup_completed_at,backup_ref)` UTF-8 字节序升序，组合不得重复。顶层 `backup_ref` 必须精确出现一次，且该项 `backup_completed_at == backup_cutoff`；其余项不得晚于 inclusive cutoff，`generated_at` 不得早于 `backup_cutoff`。调用参数、manifest、RecoveryRun、Recovery Journal 与 ReplayStatus 的 `backup_ref` 必须全等，scope/cutoff 不匹配按 wrong inventory 处理。
- `backup_cutoff`、`generated_at`、`backup_completed_at` 必须为 UTC、恰好六位小数的 `YYYY-MM-DDTHH:MM:SS.ffffffZ`；拒绝非 UTC、非微秒精度、非法日期或静默舍入。canonical JSON 固定 UTF-8、无 BOM、无额外字段、无多余空白，并使用上述字段顺序、类型和数组顺序。digest payload 是排除末尾 `inventory_digest` 的完整 canonical 顶层对象；`inventory_digest` 固定 `sha256:` + 64 位小写十六进制 SHA-256。未知版本、缺字段/null/别名、重复键、类型漂移、乱序/重复 entry 或摘要不符均 fail closed。
- v1 的“签发”是备份/灾备责任角色对受控 manifest 的发布与交付批准；`inventory_digest` 是 canonical 完整性校验，不冒充密码学签名。若未来要求密钥签名、远程 API 或新信任服务，必须重新修订架构/ADR。
- 文件最大 `1,048,576` bytes；配置路径与最终对象不得是 symlink，打开后必须为常规文件。descriptor 只读，文件不得含任何 write mode bit，部署还必须证明服务进程看到只读挂载；POSIX 至少拒绝 group/other writable。无法证明或解释等价 owner/ACL、常规文件、只读挂载、no-follow 或原子替换语义的平台 unsupported/fail closed。
- producer 必须在同目录写临时常规文件、同步完整 canonical bytes、原子 rename 并同步目录，禁止原地改写。consumer 每次使用 `lstat → no-follow open → fstat → 有界读取 → fstat → lstat` 核对路径与 descriptor 身份、类型、大小、权限和稳定性；读取期间路径替换、身份/大小变化或权限漂移使本次调用失败。
- 每次 `ReplayAndVerify` 与 `ReadStatus` freshness 都重新打开并验证一个 snapshot，不长期缓存、不使用 watcher。验证后的 `inventory_digest` 纳入稳定 `input_digest`；漂移立即使旧 Ready 失效，并执行同一幂等 ReplayAll、DeletionOperation/AuditEvent 对账和 evidence 生成路径，成功前只能返回 unavailable/错误。
- `inventory_scope` 的 JSON 类型固定为非空 string；v1 fixture 值为 `recoverable-backups-before-cutoff`，不得使用 object 或隐式转换。
- `ArticleDeletionReplayStatus` v1 是严格 tagged union：
  - ready variant：完整返回 `evidence_version,run_id,backup_ref,namespace,journal_head_sequence,journal_head_hash,inventory_scope,backup_cutoff,inventory_digest,deletion_operations_digest,audit_events_digest,tombstones_total,tombstones_replayed,already_absent,deletion_operations_reconciled_count,audit_events_verified_count,status,evidence_digest,evidence_ref,completed_at`；`status=ready`，全部字段必填且非 null，并执行 digest 重算。
  - unavailable variant：只返回 `evidence_version,run_id,backup_ref,namespace,status,reason_code`；`status` 只允许 `pending|unknown|failed`。禁止携带空字符串、零值或 null 的 canonical/digest/ref/time 字段伪装 ready。
- `evidence_digest = SHA-256(canonical JSON)`；canonical JSON 固定 UTF-8、无 BOM、无多余空白、固定字段顺序和 JSON 类型，仅包含 ready variant 的前十六个 canonical 字段，明确排除 `status`、`evidence_digest`、`evidence_ref`、`completed_at`。`completed_at` 只记录首份 evidence 的元数据时间。
- 共享 ready fixture 的 canonical bytes 为：
  `{"evidence_version":"article-deletion-replay/v1","run_id":"run-01","backup_ref":"backup-2026-10-20T00:00:00Z","namespace":"article-deletion/v1","journal_head_sequence":7,"journal_head_hash":"sha256:aaaa","inventory_scope":"recoverable-backups-before-cutoff","backup_cutoff":"2026-10-20T00:00:00Z","inventory_digest":"sha256:bbbb","deletion_operations_digest":"sha256:cccc","audit_events_digest":"sha256:dddd","tombstones_total":3,"tombstones_replayed":2,"already_absent":1,"deletion_operations_reconciled_count":3,"audit_events_verified_count":3}`。
  固定 digest 为 `sha256:169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`。producer/consumer 必须嵌入同一 fixture 或由同一生成源产生。
- `DeletionReplayRun` 的唯一所有权键为 `(run_id,namespace)`；`backup_ref` 首次绑定后不可变。创建使用原子 `INSERT ... ON CONFLICT` 或等价行锁；同 run+namespace 的不同 backup 并发必须稳定失败，不能创建第二行。
- 生产者计算稳定 `input_digest`，覆盖 Journal head、inventory、operation/audit 输入及 run/backup/namespace。输入未变时 ReplayAndVerify 复用同一 `evidence_ref/evidence_digest/completed_at`；仅 covered input 变化时追加新不可变 evidence 并原子更新 current 指针。
- DeletionReplayEvidence 唯一键为 `(run_id,namespace,evidence_version,evidence_digest)` 或等价约束；并发同 backup 重算不能产生语义重复证据。
- `ReplayAndVerify`、`ReadStatus` 均接收 `run_id + backup_ref`。同 run+backup 续跑幂等读取或重算同一证据；backup 变化不得复用旧 ready。
- ReplayAndVerify 成功只返回 ready variant；进行中或失败只返回 unavailable variant 或公共错误，不能同时使用两套失败编码。
- identity 只从 DTO 按 v1 canonical schema 重算 digest，并比较 RecoveryRun.backup_ref；不得读取 article 内部表。二次 ReadStatus 必须返回同一 `evidence_ref/evidence_digest`。缺字段、未知版本、unknown/failed 均 fail closed。
- producer/consumer 必须共享同一 schema fixture 和 canonical digest 契约测试。
- Journal head、inventory、operation 或 audit 输入变化使旧 ready 失效。证据缺失、旧版本、不可读、摘要或 namespace/backup 不匹配时返回 `unknown/failed` 并保持 readiness 关闭。
- Journal 不可用、损坏、缺 key、迁移不完整或与 DeletionOperation 不一致时，日常删除 fail closed，灾备恢复保持停服。
- tombstone 及对应 key 至少保留到所有早于 `deleted_at` 的可恢复备份退役；production inventory authority/provider/config/v1 格式已固定，具体 retention、备份周期、RPO、RTO、容量与退役数值仍为运行输入 TBD。
- 不新增文章恢复 API，不保存内容副本；Deletion Journal 不是 Article 真相源，也不替代 AuditEvent。
- 分工固定：article S4.2 实现 production 文件 reader/provider、唯一 wiring、严格解析/安全/刷新/fail-closed 合同；article S4.5 只生成 synthetic fixture 与 runbook，不提供 production provider 或 fallback；article S4.4 使用真实旧备份、真实 Journal/keyring/inventory 完成 ReplayAll、operation/audit 对账、identity 双次 ReadStatus/MarkConsumed 联合 Gate。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r09` 采用本决定，约束永久删除、部署、恢复重放和测试。来源：当前用户批准文章删除日志方案。
- 2026-10-20 架构修订 `architecture-ir01-r10` 采用本次修订，补齐绑定当前备份和重放快照的不可变 evidence Gate。来源：当前用户批准跨设计恢复证据整改。
- 2026-10-20 架构修订 `architecture-ir01-r11` 采用本次修订，固定 v1 DTO/canonical schema、幂等 evidence 与并发 backup 绑定。来源：当前用户批准跨设计 schema 整改。
- 2026-10-20 架构修订 `architecture-ir01-r12` 采用本次修订，固定 tagged union、`inventory_scope` string 与共享 fixture digest。来源：当前用户批准最终 DTO variant 对齐。
- 2026-10-25 架构修订 `architecture-ir01-r28` 采用本次原位修订，固定 production inventory authority、文件 provider、配置、strict v1 manifest、安全/快照/漂移合同与 article S4.2/S4.4/S4.5 owner。来源：当前用户明确“待确认或决策项采用建议值并继续”及 article design r13 独立评审 F001。
---

## ADR-0024: Article Deletion Journal 真卷韧性验证不纳入本版交付范围 (2026-10-27)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-27 明确确认本项目为测试项目、非真实交付项目，“这版不做真卷韧性验证”、后续也无需验证，并要求按变更后的新合同关闭 `article-management-S4.3`

**问题**: `article-management-S4.3` 的原合同要求在受支持 Unix 独立持久卷上验证 Article Deletion Journal/finalizer 的真实 `flock`、file/directory `fsync`、崩溃与磁盘故障韧性；开发机为 Windows、无受支持平台，该卡长期 `blocked-for-execution`，并使 B7 批次整体 Gate 保持 PARTIAL/OPEN、阻塞其下游。

**决策**: 对测试性质的本项目，不执行 Article Deletion Journal/finalizer 的真实 Unix 独立持久卷韧性验证；`article-management-S4.3` 按变更后的新合同关闭，未验证边界如实登记，不以 Windows 或 fake 结果冒充真实卷结论。不改变 ADR-0020 的产品合同、tombstone 语义、Journal/finalizer 实现，或 `article-management-S4.6` 已完成的组件与双平台编译闭环。

### 现状 / 痛点 / 改造 / 效果

S4.3 的真实卷结论（跨进程 `flock`、file/directory `fsync` 耐久性、崩溃重启、磁盘空间与介质故障）只在受支持 Unix 平台加真实可持久、可隔离、可回收卷上才有意义；Windows 与 fake 结果按定义不构成有效证据，因此该卡无法在本机推进，也无法被任何本机替代品关闭。

| 方案 | 优势 | 劣势 |
|---|---|---|
| A. 执行真实 Unix 独立持久卷韧性验证（原合同） | 取得真实卷 `flock`/`fsync`/崩溃/磁盘故障结论 | 需受支持 Unix 主机、真实可持久卷与明确环境 owner；测试项目无此交付需求，成本无对应收益 |
| B. 经决策不验证，按新合同关闭并登记未验证边界（选定） | 与项目测试性质一致；不再虚构真实卷结论；解除对 B7 及其下游的错误阻塞 | 真实卷韧性结论缺失，`flock`/`fsync`/崩溃/磁盘故障行为无实测证据 |
| C. 用 Windows 或 fake 冒充真实卷结论 | 无需新增环境 | 伪造证据；违反 S4.3 原卡面与 test-plan 既有禁止项，明确拒绝 |

- **场景匹配**：本项目为测试项目而非真实交付项目，无生产卷、无灾难恢复义务，被验证行为不产生对外承诺，故 B 的收益（范围如实、停止空转）与本项目一致。
- **劣势可接受性**：真实卷韧性结论缺失在测试项目中可接受，前提是它被显式登记为未验证，且不得外推为任何 production readiness、灾备或容量结论。

### 约束

- 本决定只改变 S4.3 的验证范围，不改变 ADR-0020 的产品合同、tombstone/evidence 语义、Journal/finalizer 实现、`article-management-S4.6` 已完成的 13 文件组件与 linux/amd64、windows/amd64 双 build-tag 编译闭环。
- `article-management-S4.3` 按“经决策不验证”关闭；关闭理由必须写明为范围决策，不得写成验证通过；其三文件写集保持未创建、未执行。
- 未验证边界必须保留：真实跨进程 `flock`、file/directory `fsync` 耐久性、崩溃/断电、磁盘空间与介质故障、真实 journal provider/root/keyring 均无实测证据。
- 不得据此宣称 production readiness、灾备就绪、容量或 P95 结论；也不得重启该验证，除非另有新的用户授权并重新走设计/ADR。
- `article-management-S4.4`（ReplayStatus producer 与旧备份恢复）与 `article-management-S1.4`（部署后 E2E）的其余未验证边界不变，不因本决定获得证据。

### 引用记录

- 2026-10-27 `article-management-S4.3` 按本决定关闭。来源：当前用户明确确认本项目为测试项目、“这版不做真卷韧性验证”且后续无需验证。
---

## ADR-0033: Article 真实 PostgreSQL 容量/性能补测不纳入本版交付范围 (2026-11-05)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人于 2026-11-05 明确确认并授权「经决策不验证：像 ADR-0024 那样落一条 ADR，按变更后合同关闭 `article-management-S2.4`」

**问题**: `article-management-S2.4` 的原合同要求在真实 PostgreSQL 18.6 上完成 calibration + small + medium 的容量/性能观测（TIMED-3REP / SINGLE-RUN 共 20 点、并发/QPS/分位数、seed 与磁盘预算），并保持正确性断言；R09 只取得「八项 PASS、一项 seed FAIL」，R10 只完成源码接收与离线/构建检查、整组永久 NOT RUN，且 R10 冻结 harness 把 adoption 路径硬编码到一份明文声明「不开放新 schema、PG 写入或完整性能窗口」的文件（2026-11-05 R11 复核见 `evidence-pg-perf-r04/continuation-20261104/author/r11-attempt-20261105/r11-structural-block.md`），在冻结源上不存在可合法签发的新窗口。该卡因此长期 `[-]`。

**决策**: 对测试性质的 Han 项目，不再执行 `article-management-S2.4` 的真实 PostgreSQL 容量/性能补测；该卡按「经决策不验证」以变更后的新合同关闭，未验证边界如实登记，不以历史局部观测、组件测试或 fake 结果冒充整组性能结论。本决定不改变 `article-management-S2.2` / `S2.3` 的已关闭结论、AM-Q2 参数、产品合同或任何生产代码。

### 现状 / 痛点 / 改造 / 效果

现状：S2.2（identity Argon2 target-profile readiness）与 S2.3（article 进程内性能观测）已各自关闭，AM-Q2 已于 2026-11-02 固定 `DefaultLimit=20`、`MaxLimit=100`、`CandidateBatchSize=200` 与「首期 100,000 篇 / 单用户 10,000 篇」的产品参数。S2.4 补的是真实 PG 上的容量/性能观测层，不是功能正确性层。

痛点：本卡的可交付结论依赖真实 PG 18.6 的完整校准与三档采样、稳定的执行窗口、磁盘/资源基线和非作者独立评审。R09 的 8 PASS/1 FAIL 是混合结果；R10 冻结源无法再合法开窗（adoption 硬编码且已声明不开放窗口）；继续投入只能重复空转，且任何拼接都会把「未执行」说成「已通过」。

| 方案 | 优势 | 劣势 |
|---|---|---|
| A. 继续原合同，先修订 harness 再重跑（calibration→small→medium） | 可取得本版真实 PG 容量/性能观测 | 需新源码修订＋非作者独立评审＋新冻结，再投入约 70–90 分钟真实负载；测试项目无对应交付需求 |
| B. 经决策不验证，按变更后合同关闭并登记未验证边界（选定） | 与项目测试性质一致；停止空转；不伪造性能结论 | 真实容量/分位数/并发/QPS/数据分布结论缺失 |
| C. 用 R09 八项 PASS、组件测试或 fake 拼接成整组通过 | 无需任何新执行 | 伪造证据；违反「未运行不冒充通过」与 `task-execution-workflow.md` §10.3，明确拒绝 |

- **场景匹配**：Han 为测试项目、非真实交付项目，无目标部署环境的容量承诺、无 SLO 合同、无对外性能义务；S2.4 的结论不被任何下游卡消费为阻塞条件（S1.5 / S8.2 的阻塞来自 Case18 恢复主线，与本卡无关）。
- **劣势可接受性**：真实 PG 容量/性能结论缺失在测试项目中可接受，前提是显式登记为未验证，且不得外推为 SLO、目标容量、P95 或 production readiness。

### 约束

- 本决定只改变 `article-management-S2.4` 的验证范围，不改变 `article-management-S2.2`、`article-management-S2.3` 的已关闭结论，不改变 AM-Q2 参数（`DefaultLimit=20`、`MaxLimit=100`、`CandidateBatchSize=200`）与「首期 100,000 篇 / 单用户 10,000 篇」，不改变任何产品合同或生产代码。
- `article-management-S2.4` 按「经决策不验证」关闭；关闭理由必须写明为范围决策，不得写成验证通过，也不得把 R09 八项 PASS、R10 源码/离线检查或组件测试拼成整组通过。
- 未验证边界必须保留：真实 PG 的完整 calibration、small/medium/large 三档容量与分位数、并发/QPS、数据分布、目标部署环境 P95/容量、large 与 canonical、旧 backup/restore 关联均无本版结论。
- 不得据此宣称 SLO、目标容量、P95 或 production readiness；不得重启该验证，除非另有新的用户授权并重新走设计/ADR。
- 既有 `pg_perf_*` 四个测试文件、`docs/benchmarks/article.md` 与 `.spec/wip/article-management/evidence-pg-perf-r04/` 全部证据保留原字节，不清理、不删除、不删除任何 schema。
- 本决定不解除 `article-management-S1.5`、`article-management-S8.2` 或任何其它卡的阻塞。

### 引用记录

- 2026-11-05 `article-management-S2.4` 按本决定关闭。来源：项目负责人明确确认并授权「经决策不验证：像 ADR-0024 那样落一条 ADR，按变更后合同关闭 S2.4」。
---

## ADR-0034: Case18 恢复运行结果到准入的门禁不纳入本版验证范围 (2026-11-05)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人于 2026-11-05 明确确认「不需要这项门禁」（测试项目、经决策不验证）

**问题**: `C18-COV-F002` 要求 `article-management-S8.2` 的 Case18 覆盖恢复隔离的 `failed` / `expired` / `undetermined` 三状态并接真实 runtime/PG。2026-11-05 实测发现：admission 的 `Verify` 不读取 `recovery_runs.outcome`——Case18 fixture 的 temporary 路径返回 `{State:"internal_ready", Reason:"verified"}`，到达 `public_ready` 后再把 outcome 改为 `failed` 仍返回 `{State:"public_ready", Reason:"verified"}`、`err=nil`；启动前注入 `failed` 时 `RunNormal` 也成功。该语义若需强制，必须落在 coordinator 的 `Resume`/start 路径并新增设计与测试切片，而不仅是补一个用例。

**决策**: 对本测试项目，不新增「运行结果 → 开流门禁」的设计与验证；`C18-COV-F002` 按范围决策关闭，未验证边界如实登记，不以现有 `internal_ready` / `public_ready` 观测冒充三状态覆盖。不改变 runtime、coordinator、admission 或任何生产代码，也不改变 `article-management-S8.2` 的其余 finding 与状态。

### 现状 / 痛点 / 改造 / 效果

现状：admission `Verify` 证明的是 journal / installation / closure 连续性；`recovery_runs.outcome` 属于 coordinator 的运行生命周期，`admission.go:726` 的 `RecoveryOutcomeFailed → ErrAdmissionBinding` 只在 live/controlled 校验路径生效，且现有 harness 的 `Verify` 不读取 run 行。

痛点：`C18-COV-F002` 的字面要求与现役设计不匹配；要满足它必须先定义「运行结果 → 门禁位置」的映射（coordinator `Resume`/start 还是 admission），再补失败注入与隔离断言，属新的设计＋实现切片；本项目为测试项目，无对应交付义务。

| 方案 | 优势 | 劣势 |
|---|---|---|
| A. 新增门禁设计并实施（coordinator Resume/start 失败注入 + 隔离断言） | 可得到 failed/expired/undetermined 对开流的真实结论 | 需新设计、新代码与新测试切片；测试项目无对应收益 |
| B. 经决策不验证，F002 按范围决策关闭并登记未验证边界（选定） | 与项目测试性质一致；不伪造覆盖；解除 S8.2 的一条错误阻塞 | 三状态对开流门禁的影响无本版结论 |
| C. 用现有 `internal_ready`/`public_ready` 观测冒充三状态覆盖 | 无需新增工作 | 伪造证据；违反「未运行不冒充通过」，明确拒绝 |

- **场景匹配**：Han 为测试项目，无生产恢复 SLA、无对外灾备承诺；该门禁不被任何下游卡消费为关闭前置。
- **劣势可接受性**：三状态门禁结论缺失可接受，前提是显式登记为未验证，且不得外推为恢复安全、灾备或 production readiness。

### 约束

- 本决定只关闭 `C18-COV-F002` 的验证要求；不改变 runtime、coordinator、admission、guard 或任何生产代码，不改变 ADR-0018/0019/0027 的恢复合同。
- `C18-COV-F002` 按「经决策不验证」关闭；理由必须写为范围决策，不得写成验证通过，也不得以 `internal_ready` / `public_ready` 观测冒充 `failed` / `expired` / `undetermined` 覆盖。
- 未验证边界保留：`failed` / `expired` / `undetermined` 运行结果对开流门禁的影响、coordinator `Resume`/start 的失败注入与隔离断言，均无本版结论。
- 不改变 `article-management-S8.2` 的其余 finding 与状态：`C18-COV-F003` 待原 Reviewer 复审、`C18-COV-F004` 待原 Reviewer 复核、真实部署 E2E 与双系统仍按原卡未验证。不解除 `article-management-S1.5` 的阻塞。
- 既有测试、证据与本轮 G0 记录保留原字节；不重启该验证，除非另有新的用户授权并重新走设计/ADR。

### 引用记录

- 2026-11-05 `C18-COV-F002` 按本决定关闭。来源：项目负责人明确确认「不需要这项门禁」（测试项目、经决策不验证）。
---

## ADR-0035: Case18 真实部署 E2E 与双系统验证不纳入本版交付范围 (2026-11-05)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人于 2026-11-05 确认「经决策不验证」（承接 ADR-0034 同口径：本项目为测试项目）

**问题**: Case18 的关闭条件第 5 条要求 `article-management-S8.2` 具备真实部署 E2E 与双系统证据（评审 LOW `C18-COV-F007`）。本项目为测试项目，无部署目标环境、无真实平台/网络/receiver/账号、无演练窗口与业务验收责任人；该要求无法在本机以任何真实形式满足，且与 ADR-0025/0028/0034 已确立的测试项目验证口径不一致。

**决策**: 对测试性质的 Han 项目，`article-management-S8.2` 的 Case18 **不执行真实部署 E2E 与双系统验证**，该要求按「经决策不验证」关闭并在原卡登记未验证边界；不以模拟部署、单机链路或既有本地 PG 证据冒充真实部署结论。不改变 `C18-COV-F003` / `C18-COV-F004` 的复审要求，也不改变 S8.2 的其余关闭条件。

### 现状 / 痛点 / 改造 / 效果

现状：S8.2 的九文件实现、局部与真实 PG 功能验收、联合 CodeReview R02、七项项目检查均已完成；Case18 的隔离、开流、SessionStore 证据已在本机取得。缺的是真实部署环境与双系统对照。

痛点：真实部署 E2E 需要部署目标、平台与网络、账号与 receiver、演练窗口和业务验收人，测试项目均不具备；继续挂起只会让已完成的实现与证据无法收口，且不得据此宣称部署就绪。

| 方案 | 优势 | 劣势 |
|---|---|---|
| A. 搭建真实部署环境并执行双系统 E2E | 可取得真实部署结论 | 需真实环境、平台/网络/账号/窗口与责任人；测试项目无对应交付义务 |
| B. 经决策不验证，按范围关闭并登记未验证边界（选定） | 与测试项目定位一致；停止空转；不伪造部署结论 | 真实部署 E2E、双系统、平台差异结论缺失 |
| C. 用本机模拟或既有单机证据冒充真实部署 | 无需新增环境 | 伪造证据；违反「未运行不冒充通过」，明确拒绝 |

- **场景匹配**：Han 为测试项目，无生产部署、无跨系统一致性义务；S8.2 的其余证据均为本机真实执行，不需部署环境即可成立。
- **劣势可接受性**：部署与双系统结论缺失可接受，前提是显式登记为未验证，且不得外推为部署就绪、production readiness 或跨平台一致性。

### 约束

- 本决定只关闭 Case18 的「真实部署 E2E / 双系统」验证要求；不改变 `C18-COV-F003`（待原 Reviewer 复审）与 `C18-COV-F004`（待原 Reviewer 复核）的关闭链，也不改变 S8.2 的其余 finding。
- 关闭理由必须写为范围决策，不得写成部署验证通过；不得以本机链路、模拟部署或既有单机 PG 证据冒充真实部署结论。
- 未验证边界保留：真实部署 E2E、双系统/跨平台一致性、真实平台与网络、receiver/账号、演练窗口与业务验收，均无本版结论。
- 不改变任何生产代码、部署材料、runtime 或 guard；不解除 `article-management-S1.5` 的其余阻塞。
- 不得据此宣称部署就绪、production readiness 或灾备就绪；不重启该验证，除非另有新的用户授权并重新走设计/ADR。

### 引用记录

- 2026-11-05 `article-management-S8.2` 的 Case18 真实部署 E2E/双系统要求按本决定关闭。来源：项目负责人确认「经决策不验证」。
