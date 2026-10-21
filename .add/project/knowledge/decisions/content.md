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

## ADR-0020: 业务数据库外 Article Deletion Journal 保证永久删除跨备份恢复 (2026-10-20)

**目的**: 数据模型 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 批准 Article Deletion Journal 方案，用于关闭 AM-DES-R01-F001

**问题**: 物理删除发生在数据库备份之后时，如何防止恢复旧备份使已永久删除文章重新出现？

**决策**: 在业务 PostgreSQL 恢复域之外建立 Article Deletion Journal，并由 article-management 持久化不可变 `DeletionReplayEvidence`。重放状态绑定 `run_id + backup_ref + namespace`；identity 只消费版本化状态和摘要，不读取文章内部表。删除流程及物理删除成功条件保持不变。

### 现状 / 痛点 / 改造 / 效果

仅物理 DELETE 会被旧备份复活；在业务库保留软删正文又违反永久删除和不保留内容副本。外部最小 tombstone 可以在恢复开放前重放删除，同时不保存文章内容。

| 方案 | 优势 | 劣势 |
|---|---|---|
| 外部最小 tombstone Journal | 可跨备份恢复重放；不保存内容 | 引入跨介质删除 finalizer 与保留管理 |
| 业务库软删除 | 查询简单 | 旧备份仍回退；保存内容副本并形成恢复能力 |
| 仅依赖备份退役 | 无新增运行组件 | 无法保证所有旧备份及时退役，删除语义不可验证 |

### 约束

- tombstone 仅含 `article_id`、`article_version`、`deletion_operation_id`、`deleted_at`、`key_id/version`、阶段和校验链；禁止 title、body、内容摘要、作者邮箱或其他 PII。
- 授权与 `If-Match` 成功后，事务将 Article 置 `deletion_pending`；该内部状态对成员和管理员均不可见，不是可恢复业务状态。
- Journal 原子追加并 fsync 成功后，finalizer 幂等物理 DELETE Article，并同事务写 AuditEvent 与 DeletionOperation 完成。
- API 只在 Journal tombstone 耐久且物理删除完成后返回 204。
- Journal 成功但数据库删除失败或崩溃时返回失败，Article 保持不可见的 `deletion_pending`；finalizer 必须继续完成，禁止恢复公开。
- Journal 失败时不得返回成功；事务回滚或保留可安全重试的不可见状态，不得产生重新可见窗口。
- 恢复旧备份后、readiness 开放前，重放全部有效 tombstone，幂等删除恢复出的 Article；与 Recovery Journal/账号恢复门禁共同完成后才开放。
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
- tombstone 及对应 key 至少保留到所有早于 `deleted_at` 的可恢复备份退役；备份 inventory、周期和退出条件为运行输入 TBD。
- 不新增文章恢复 API，不保存内容副本；Deletion Journal 不是 Article 真相源，也不替代 AuditEvent。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r09` 采用本决定，约束永久删除、部署、恢复重放和测试。来源：当前用户批准文章删除日志方案。
- 2026-10-20 架构修订 `architecture-ir01-r10` 采用本次修订，补齐绑定当前备份和重放快照的不可变 evidence Gate。来源：当前用户批准跨设计恢复证据整改。
- 2026-10-20 架构修订 `architecture-ir01-r11` 采用本次修订，固定 v1 DTO/canonical schema、幂等 evidence 与并发 backup 绑定。来源：当前用户批准跨设计 schema 整改。
- 2026-10-20 架构修订 `architecture-ir01-r12` 采用本次修订，固定 tagged union、`inventory_scope` string 与共享 fixture digest。来源：当前用户批准最终 DTO variant 对齐。
