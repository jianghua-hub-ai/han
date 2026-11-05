# Article S6.2 r15 独立 Design Review R01

结论：**DesignReviewGate = CHANGES REQUIRED / FAIL**。共 **1 HIGH**；未发现其他可确认的阻断缺陷。结论仅覆盖指定增量与直接消费者，不代表整份设计、现役实现或 PostgreSQL 验证通过。

## 固定对象与权限

- objectType：`design`；修订：`article-management-design-ir01-r15`。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 对象：`.add/project/knowledge/design-workspace/article-management/design.md`。
- SHA256：`0435103681F17C01474A401872F7D0229EEA1B1A2ADD14A7693658F4553863D6`；审查前及写报告前均核对一致。
- 前文副本：`C:\Users\J00010~1\AppData\Local\Temp\han-s62-4d7a5257f49745bc82ad754d1798d845\.add\project\knowledge\design-workspace\article-management\design.md`；SHA256 `AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`。
- 实际差异：新增 114 行、删除 39 行，共 153 行；顶部版本/记录、§3.3.7、§3.4.4、§3.4.5。未重审全部 2534 行设计。
- 独立性：本执行者按 Reviewer 职责审查，非所给原作者 `Parfit01a17bcd-fffc-78b1-ba18-ea12b850a6bd`；未委派、未自称额外 Agent ID。
- 唯一写入文件为本报告；未修复设计或代码，未 Git 写入、清理或生成 checkpoint。
- 当前活动：development-delivery 的 design 增量独立 review；交回当前 Lead/设计作者。S6.2 原三文件任务仍待设计及写集复核；本报告不推进任务或实施授权。

## 依据与两轮实际范围

已完整读取本端 Reviewer Role、`add-route-work`、`add-code-review`、`add-delivery-workflow`、`code-review-checklist`、`abstraction-principles`、`design-doc-writing-style`、`decisions-guide` 与产品安全红线；补充取证按 `add-research`。go-service 为唯一启用 profile，未将代码修改/构建条款强套本次文档评审。

Round 1：独立核对真实前后文差异，通读 §3.3.7 与新 §3.4.4/§3.4.5 必要上下文；核对状态转换、强 If-Match、锁内 source/version 重验、事务事实、provider 信任边界、双通知快照、审计、清理、回滚、幂等及无来源 replay 边界。额外核对版本顶部与旧确认记录的历史限定，未把 r14/r06 接纳当成 r15 签字。

Round 2：沿指定消费者反查，不展开全库：

- 原任务 S6.2（`tasks.md:1528`）、Story 6（`stories.md:144`）、A-ORP-03～07（`test-plan.md:255`）。
- 原需求 US-IDENTITY-03.8/03.9（`requirements.md:268`、`:282`），及通知邮箱清理原始澄清 `IR-01-clarification-14.md:20`。
- identity S5/清理接口与 canonical 设计（identity `design.md:2039`、`:2043`、`:2062`、`:2091`、`:2934`）；ADR-0013 的条件保留/清理及 ADR-0020 的两权威、恢复边界。
- 现役 canonical 的 User/Identifier→Article、Article coordinator 的 Article→operation、finalizer 的 operation→Article、worker 分类/lease discovery；相关 schema 和 canonical pending 测试只读取证。
- 仅为新增精确 lease 核对，读取 production SystemClock 与本机已安装 pgx v5.11.0 的 timestamptz codec；官方 PostgreSQL 18 文档只用来确认语义，不作为本机执行证明。

## Findings

### [HIGH] ordinary pending 随后删除作者，最后完成删除没有邮箱清理消费者

- finding ID：**AM-DES-R15-S62-R01-F001**。
- 对象/修订：design / `article-management-design-ir01-r15`，上述固定 SHA256。
- 位置：article `design.md:1956`、`:1959`；与 `:1924` 的 active/pending 统计、`:1928` 的最终清理及既有 `:1864` 不一致。
- 现状：新增 T2 只为 `delete_kind=orphan` 且 operation.source 可证明的操作取得 Identity 清理资格。canonical 则明确把 ordinary pending 的作者关联计入保留条件，但不将它改造成 orphan operation。该 pending 最终消失时没有清理调用方。
- 触发交错（正常在线路径，不依赖损坏数据或旧备份）：
  1. 用户 U 的最后一篇普通文章 A 开始 ordinary 删除；T1 提交 pending，operation.kind=ordinary、operation.source=NULL，A.author=U。Journal/T2 尚未完成。
  2. canonical 删除 U：保留邮箱到 retained_email；锁到 A，但 A 已 pending，跳过 active 无归属化。A.author 仍为 U、source 仍 NULL；pending count=1，因此保留 retained_email/email Identifier。
  3. ordinary finalizer 继续删除 A，写审计并完成 operation。按 r15 的 orphan-only 条件不会调用清理。此后所有 active/pending 关联文章均为零，无转移通知待留存，仍遗留专用邮箱和邮箱唯一性占用。
- 影响：正常删除顺序可使专用通知邮箱和 email Identifier 无限期保留，破坏最终清理与邮箱复用合同；完成 operation 不再由 worker discovery 调度。额外受控 canonical 重核不是已定义的在线自动收敛步骤，不能替代缺失消费者。
- 违反依据：US-IDENTITY-03.9 第 6 条（`requirements.md:291`）、原始澄清 `IR-01-clarification-14.md:23`、ADR-0013 的条件清理；设计写作规则 J 的状态/失败收敛责任，以及评审清单 §2 的契约与生命周期完整性。
- 直接证据：
  - `internal/article/deletion_coordinator.go:171`：operation.source 直接复制 Article.orphaned_from_user_id；ordinary Article 此字段为空。`:182` 随后仅写 pending。
  - `internal/article/article_port.go:59`：canonical 候选同时包含 active/pending，按 author 或 source 关联查询。
  - `internal/article/command_service.go:499`：只无归属化 active；`:473` 将关联 pending 计入 PendingOrphanCount，`:513` 关联包含 author。
  - `internal/article/orphan_canonical_test.go:176`：现役 mixed active/pending 测试明确包含 pending-owned，计数包含它、更新只涉及 active；本轮读取但未运行。
  - `internal/identity/deletion.go:429`：只有 PendingOrphanCount=0 才释放邮箱；`:1281` 先保留原邮箱。
  - `internal/article/deletion_finalizer.go:487`、`:515`：现役路径完成 DELETE/operation，不含 Identity 条件清理；r15 新增调用仍限定 orphan。
  - `internal/article/repository.go:424`：discovery 不再发现 completed operation。
  - `migrations/0003_article.sql:22` 与 `:70`：operation kind/source 绑定且不可变，不能把旧 ordinary operation 简单补 source 或改 kind 来规避缺口。
- 最小兼容建议：补齐“最后一个关联 pending 消失”的清理责任，不改变 REST、schema、墓碑或数据所有权。对于当前仍存在的 ordinary/violating pending Article，可从 T2 已锁定前像的 author 取得候选身份，以 NOWAIT 锁 User/Identifier，确认该 User 已 deleted，才构造同事务清理事实；保持 operation 首锁、Article NOWAIT，新增 Identity 取锁也必须 NOWAIT，失败整体 rollback。若作者尚未 deleted，则本次不清理；User 锁持有到 commit，使随后 canonical 在文章已删后自行观察零集合。身份来源只能是受锁前像/现有可信绑定，不能是 actor 或 tombstone。
- 建议需要同步的合同：允许该分支的 CompletionRef.SourceID 绑定受锁 author 前像；completed operation 仍按原 article/version/id 验证，并承认其 ordinary source=NULL，不能继续要求它伪装为 orphan.source 全等。保留审计、双计数门禁、同事务回滚和精确 claim 校验。Article 已 absent 且没有可信来源的 replay 继续保守处理，不凭此建议猜来源。
- 可行性边界：以上方向可利用现有 author、User 状态、tx、NOWAIT 与临时完成事实表达；未证明 SQL/真实锁行为已实现。若实现发现需改变持久绑定、恢复协议或所有权，交 Lead/架构决定，Reviewer 不代选或修改。
- 复审条件：原作者在新固定修订补齐该分支与所有受影响事实/锁序说明；独立复审核对本 finding。后续实施需真实 PG 多连接验证上述三步交错，及 ordinary T2/作者 canonical 互相竞争、仍有另一 pending 时保留、最后删除清理、释放审计失败整体回滚、重试无重复审计、无来源 replay 不误清理。验证必须包含“没有后续人工 canonical 调用仍可收敛”。

## 其他重点核对结论与非阻断建议

| 重点 | 有界结论 |
|---|---|
| Identity→Article 与 canonical 反序 | 新路径所有新取行锁 NOWAIT，冲突整体 rollback，可在声明调用链中消除新路径的行锁等待环；不是全库无死锁证明。 |
| coordinator Article→operation 与 T2 operation→Article | T2 显式 NOWAIT 读锁在 DELETE 前，遇 coordinator 占 Article 即退出并释放 operation；claim 独立提交、不跨 Journal I/O 持事务，合同可实施。 |
| 事务完成事实/信任边界 | 固定 production article provider、精确 tx/前像绑定、回读 Article/Notice/审计/计数及 identity 地址绑定均有明确责任；不信布尔、外部 JSON 或任意 callback。现役尚无该完整生产接线，不判实现 PASS。 |
| 来源绑定/并发最后两篇 | 对 r15 覆盖的 orphan 分支，来源 User 持锁至 commit、动作后 READ COMMITTED 统计 active+pending，可串行化处置与清理。ordinary pending 的退出责任见 F001；两个计数实现时应使用同一统计快照，避免将 active→pending 迁移分散到不一致的计数时点。 |
| 通知与审计 | 当前 transfer 在同 tx 写 Notice、两种 pending recipient、文章和处置审计；地址回读对比锁内 Identity 值，之后才清理；Gate/SMTP 不决定事务成功。Repository 现有 InsertTransferNotice 只是接缝证据，不证明上述合同已实现。 |
| 清理幂等 | retained_email/email Identifier 成对验证；均已不存在返回 unchanged，不重复释放审计；单边缺失视漂移；新 HTTP 请求仍受原 ETag/已处置语义约束。 |
| replay | 最小 ordinary 重建没有来源，不可从 actor/tombstone猜清理 source；active 旧备份删除也未被现役 pending-only DELETE 证明。r15 明确保留 Gate 关闭/交 Lead 的边界，未伪称恢复实现通过。该保守限制不能覆盖 F001 的在线正常交错。 |

**N001，非阻断措辞澄清：§3.4.4 第 3 步邮箱条件。** `design.md:1874` 第二句的“新作者邮箱和来源 retained_email 必须非空”应显式限定 transfer；publish 不需要目标邮箱或通知快照，仍应核来源墓碑及已有邮箱/Identifier 一致性，清理遵循 `:1930` 的成对空值幂等规则。没有仅凭 nullable 提出缺陷：identity `design.md:2052` 确实允许可空路径 A，但实际 `migrations/0002_identity.sql:32` 要求普通未删除账号 email/phone 非空；唯一例外 initial admin 又受 `internal/identity/deletion.go:526` 保护不得删除。正常首次 canonical 删除有 pending 时保留原邮箱，不能据此证明“合法 held orphan 且邮箱已空”的在线触发。当前列为作用域澄清，不作为第二条 finding，也不据此要求削弱真实漂移检查。

**N002，非阻断实施注意：精确 claim 与 55P03。** `design.md:1955` 新增 status/owner/locked_until 匹配且 lease 仍有效的要求可实施，但现役 `deletion_finalizer.go:479` 只核 status/WorkerID，尚未达到该要求。应在 T2 取得 operation 锁后使用新鲜时钟验证有效期，并使用数据库精度下的权威 claim 时间做匹配。现役 claim 的 `:422`/`:428` 从 SystemClock 生成时间，`:647` 直接回传 Go lockedUntil；production `internal/identity/registration.go:74` 未截微秒。本机 pgx v5.11.0 `pgtype/timestamptz.go:173` 编码会除以 1000，因此不能把未规范的 Go 纳秒值直接与 DB 回读值比较。可以返回数据库实际保存的 lease 值，或在写入/返回前统一 UTC 微秒；r15 没有禁止这些必要适配，不将未实现状态本身列为设计缺陷。[PostgreSQL 18 时间类型精度](https://www.postgresql.org/docs/18/datatype-datetime.html)

worker 直接调用 `ConvergeDeletionOperation`→`Finalize`（`deletion_finalizer.go:167`），其现役 switch 在 `deletion_worker.go:198` 优先处理 lease competition，`:204` 把 ErrDeletionInProgress 当 no-op，`:209` 将其余非 terminal 错误列为 transient。保留可识别错误链的 55P03 可复用后者；不要从 HTTP coordinator 的 `errors.Join(ErrDeletionInProgress, err)` 借道。r15 `:1967` 已明确这一点。回滚保留 claim，`:1968` 等 lease 过期再由现有 discovery 重领，与现役 `repository.go:425` 相容。没有声称当前代码已有 NOWAIT/精确租约验证或专门 55P03 实测。

NOWAIT 的行锁立即报错与表级 ROW SHARE 仍可能等待的区分，与设计声明一致；只确认官方语义，未据此证明本机锁序或事务回滚。[PostgreSQL 18 SELECT 锁子句](https://www.postgresql.org/docs/18/sql-select.html#SQL-FOR-UPDATE-SHARE)

## 证据修订、验证限制与交回

现役代码为当前工作树证据，不冒充固定 HEAD 已提交内容。关键文件取证 SHA256：

| 文件 | SHA256 |
|---|---|
| `internal/article/article_port.go` | `215A224CF416D81FA217F7524EDEEA671A25AC1110BA0147CC1D1BD6315CE044` |
| `internal/article/command_service.go` | `7162FA880F56E95814C46347A75041B81FFDFBA00231E863F7D02F2317D8A2E2` |
| `internal/article/deletion_coordinator.go` | `307BEEDE0F62878D5FA39D81A69529792E3155735D776C565AC3E4AFD46D1C70` |
| `internal/article/deletion_finalizer.go` | `E914B290DBCB59D24116ECBBD4592AD171038FCEA25081739AF4A6CFB17FD12C` |
| `internal/article/deletion_worker.go` | `9E6BDD23089B41A3870DB0A35420589DD2210972C0610A43184D6E7C60F6612D` |
| `internal/identity/deletion.go` | `119D378C3508A790FFC6E7371C5AFC0D8CE100A33C2F47AFACD305F7683A2721` |
| `migrations/0002_identity.sql` | `F4EC1005EA72B1774DF5E370C5ABA57CA3ED7A0B13099DBBE2D16AA66AEB9D50` |
| `migrations/0003_article.sql` | `FCAECE696DCE29D2F8D252EA36C4550F0C1736347E921F40703540B99E100D4A` |

实际执行仅为文件读取、范围搜索、前文 diff/哈希核对、`git status --short`、分支/HEAD 只读查询、`go env GOMODCACHE` 和两页官方文档查阅。**没有运行 Go 测试、构建、数据库连接、SQL 实验、PG 多连接测试、SMTP、PlantUML 渲染或 E2E**；读到现役测试不等于运行 PASS。未验证部署、真实旧备份/Journal 恢复、全库锁图或未声明消费者。

待决定：F001 的具体事实绑定与最小实现落点由 Lead/原作者决定；如需超出 schema/API/所有权兼容边界，返回架构决定。N001/N002 为非阻断澄清/实施注意，不混充新增缺陷或人的风险接受。

r15 保持 draft；本次无接受记录、checkpoint 或角色签字。原作者修订后以新固定 hash 交原 finding 独立复审；本报告不代替 Lead 现场复核、人的结果接纳或后续动作授权。报告完成后停止写入。
