# article-management-S4.1 独立代码评审 r01

- Reviewer：add-reviewer（独立非作者 Reviewer）
- 日期：2026-10-23
- objectType：code
- 任务：`article-management-S4.1`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 对象状态：固定未提交代码
- 评审结论：**FAIL**

## 1. 固定对象与起始哈希

开始评审时，下列 SHA256 均与交接值一致：

| 文件 | SHA256 |
|---|---|
| `internal/article/deletion_coordinator.go` | `6095A276AFF139CA7294BE60050E09D9A4CB07B83BD86113176D8BF4364C35E2` |
| `internal/article/deletion_finalizer.go` | `DB1CFE354FE7F6BAC693DE41E961D29163762E1F693DE08A46984B1B9A515F07` |
| `internal/article/deletion_journal.go` | `6B3325AA9E217F787D020B7726BF71B8CA08FE4C038D63D72EA35A26F1518159` |
| `internal/article/command_service.go` | `58A1FFDFD19C7C7214C07C28F90CF99F50A172B9AD054787A16C90BF08CC0603` |
| `internal/article/handler.go` | `874A2BAE6AE69C1C96A888E80AC1E35FE94452398B2CABE5F38C81696CE60F0A` |
| `internal/article/deletion_test.go` | `3E8101493E0660243DA7B49F463B8855764994439397BF272E01CFD5D86E5A07` |

工作树存在其他工作者的既有修改和未跟踪文件。本次未回滚、覆盖、格式化或修改
任何代码、设计、Tasks、依赖或其他报告；唯一写入为本报告。

## 2. 依据与实际读取

已读取并应用：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-route-work/SKILL.md`
- `.codex/skills/add-delivery-workflow/SKILL.md`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.codex/guidance/observability-standards.md`
- `.codex/guidance/tracing-standards.md`
- `.codex/profiles/go-service/README.md` 及其五份适用 rules

业务和验收依据：

| 依据 | SHA256 / 位置 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `A3DBE8625EA1879711740F7DE9E63C423C9B597CA163EF0B63A86F4914789400`；`:1010-1030` |
| `.add/project/knowledge/design-workspace/article-management/design.md` | `70EE8FB04196FA78EB925133EC8C796E1B34F5B2060273BCE2A704DE3F797B91`；`:1298-1355,1730-1768` |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | `5A1FE258792CF281B27DE2B9E51131E1C46E738F8B3589A184ACFF383A9DDF3A`；`:140-153` |
| `.add/project/knowledge/decisions/content.md` | `A0594229FCFDA0B8F243A476024BF51825F8DFB12671B41675FC270FA12DBF88`；ADR-0020 `:87-141` 及 ADR-0016 相关审计约束 |

直接接缝读取包括六份固定对象，以及
`internal/article/model.go`、`policy.go`、`repository.go`、`article_port.go`、
`repository_integration_test.go`、`internal/audit/store.go`、
`migrations/0003_article.sql`、`internal/recovery/filelock_flock.go` 和 Windows
file-lock 实现、`cmd/han-server/main.go` / `main_test.go` 的生产 composition/router。
另读取了当前 Go 标准库 Windows 文件打开与 `Fsync` 实现，用于核对目录同步分支。

## 3. 两轮实际检查范围

### Round 1：固定对象直接审查

逐行检查六份固定对象，重点核对：

- T1 的授权、强 If-Match、`deletion_pending` 与唯一 operation；
- Journal 的 allowlist、HMAC/链、幂等追加、锁、临时文件、file/dir fsync；
- journal binding 失败及 commit outcome unknown 后的收敛；
- finalizer claim/lease、双 worker、物理删除、审计和 completed 的事务边界；
- 重复 DELETE、三类 delete kind、错误映射和 204 双完成条件；
- 测试是否覆盖生产实现所声明的失败与并发语义。

### Round 2：有界消费者、上游依据与迁移核对

沿真实生产者/消费者有界检查：

- Article 状态、可见性策略和 PostgreSQL 约束是否支持“失败不重新可见”；
- `Repository` SQL 与 migration 的 operation 状态约束、不可变字段和外键；
- Audit caller-owned transaction 接缝；
- recovery Journal 已采用的进程生命周期文件锁实现；
- production composition/router 是否构造 Journal/Coordinator/Finalizer 并注册 DELETE；
- S4.2～S4.5、S5.1、S6.2、S8.1 的后继责任边界。

未扩展为无界全仓审查。

## 4. Findings

### HIGH — `ARTICLE-S4.1-R01-F001`：Windows fallback 未完成目录元数据耐久化，却把 tombstone 判为成功

- **位置**：
  - `internal/article/deletion_journal.go:261-268`
  - `internal/article/deletion_journal.go:374-401`
- **违反依据**：
  - design `:1741`、Journal 合同 `:1760` 要求 file 与 directory fsync；
  - ADR-0020 `:111-114` 要求 tombstone 耐久后才能 finalizer/204；
  - Task `:1028` 要求 fsync 失败不得返回 204。
- **触发场景**：Windows 上 `directory.Sync()` 返回权限错误，代码进入
  `syncDeletionJournalWindowsBarrier`；随后发生进程崩溃、主机故障或断电。
- **证据**：
  1. tombstone 先通过 `os.Rename` 发布；
  2. fallback 只在同一目录新建一个无关临时文件、同步该临时文件并关闭；
  3. fallback 没有同步目录句柄，也没有对已发布 tombstone 或其 rename 元数据建立
     耐久保证，却返回 `nil`；
  4. 当前测试只注入 `syncFile` 失败，没有验证该目录 fallback 的崩溃耐久语义。
- **影响**：API 可在 finalizer 完成后返回 204，但崩溃恢复时 tombstone rename
  仍可能丢失；恢复旧备份后文章可被复活，直接破坏 ADR-0020 的跨备份永久删除保证。
- **修复建议**：使用目标平台可证明的目录/rename flush 原语；若 Windows 无法满足，
  启动或 Append 必须 fail closed，不能用同步无关文件代替。S4.3 在真实目标持久卷上
  加入 rename 后断电/崩溃验证。
- **复审条件**：平台分支具备可复核耐久保证或明确拒绝不支持的平台；目录同步失败
  测试证明不会成功返回，固定对象和 S4.3 故障用例回归通过。

### HIGH — `ARTICLE-S4.1-R01-F002`：finalizer 无法对账已存在 tombstone 的 `pending_journal` operation

- **位置**：
  - `internal/article/deletion_finalizer.go:212-230`
  - `internal/article/deletion_finalizer.go:294-328`
  - `internal/article/deletion_coordinator.go:112-120,193-210`
- **违反依据**：
  - design `:1742` 明确要求 binding DB 更新失败时 tombstone 仍为权威事实，
    finalizer 可按 operation ID 对账；
  - design `:1326` 要求后台继续收敛；
  - A-DEL-04 要求后台/授权重试继续收敛。
- **触发场景**：Journal 已耐久追加，但 `MarkDeletionOperationJournaled` 的数据库
  更新明确失败；客户端不再发送原授权 DELETE，后台取得已验证 tombstone 后调用
  `Finalize`。
- **证据**：
  1. `requireTombstoneBinding` 明确接受 `pending_journal`；
  2. `ClaimDeletionOperation` 的 SQL 只允许 `journaled|finalizing`；
  3. 因此 pending operation 的 claim 更新为零行，`Finalize` 返回
     `ErrDeletionInProgress`，不能完成 binding 或物理删除；
  4. 有界搜索未发现其他后台 reconciliation 消费者。
- **影响**：只要原调用方不再重试，文章和 operation 可永久停留在 pending；系统没有
  达成设计要求的后台收敛，也无法把权威 tombstone 转化为 completed 删除事实。
- **修复建议**：在持有 operation 行锁时，以从 Journal 取得并完整验证的 tombstone
  原子完成 pending binding，再 claim；或提供等价的幂等 reconciliation 操作。
  必须保持 migration 对 tombstone 引用和状态的原子约束。
- **复审条件**：新增“tombstone 已存在、binding 失败、无原 HTTP 重试”的直接
  finalizer/reconciler 用例，并证明双 worker 下只绑定和完成一次；明确实际后台
  消费者及其 composition owner。

### HIGH — `ARTICLE-S4.1-R01-F003`：finalizer 未验证 tombstone 的 HMAC/链和完整绑定即可执行不可逆删除

- **位置**：
  - `internal/article/deletion_finalizer.go:89-114,212-230`
  - `internal/article/deletion_test.go:618-624`
- **违反依据**：
  - design `:1747` 要求 Finalizer 先验证 tombstone；
  - ADR-0020 `:111-115,132` 要求只有可验证 Journal 事实才能完成删除或开放恢复。
- **触发场景**：内部调用者向导出的 `Finalize` 传入伪造或被篡改的 tombstone；
  其 operation/article/version/sequence/key 引用匹配数据库，但
  `record_hmac`、`previous_record_hash` 或 `deleted_at` 不匹配 Journal。
- **证据**：
  1. `Finalize` 只检查 namespace、stage 和 operation ID；
  2. `requireTombstoneBinding` 只比较 article/version/sequence/key 引用；
  3. finalizer 没有 Journal/verifier 依赖，无法验证 HMAC、前链或 canonical record；
  4. finalizer 测试使用 `"hmac:synthetic"` 仍可完成删除。
- **影响**：错误接线、内部缺陷或受影响调用者可以绕过外部耐久删除证据，物理删除
  Article 并写成功审计；恢复时却可能没有可验证 tombstone，破坏不可恢复删除合同。
- **修复建议**：让 finalizer 只消费由 Journal 返回的不可伪造“已验证记录”能力，
  或在 finalizer 内按 operation ID 从 Journal 重新读取并验证全链/HMAC，再比较全部
  canonical 字段。不要把普通可构造 struct 当作验证证明。
- **复审条件**：篡改 HMAC、previous hash、deleted_at、sequence/key、namespace/stage
  均 fail closed 且不 claim、不删 Article、不写 Audit；合法记录和双 finalizer 回归通过。

### HIGH — `ARTICLE-S4.1-R01-F004`：物理 DELETE 未命中时仍可写成功审计并把 operation 标为 completed

- **位置**：
  - `internal/article/deletion_finalizer.go:184-208`
  - `internal/article/deletion_finalizer.go:331-346`
- **违反依据**：
  - design `:1317-1321,1748-1754` 和 ADR-0020 `:111-113` 要求物理删除与
    Audit/operation completed 同事务，双完成后才能成功；
  - Task `:1028` 要求失败不得产生重新可见窗口。
- **触发场景**：`articles` 中仍存在该 ID，但状态不是精确的
  `deletion_pending + deletion_operation_id` 绑定（例如恢复/数据不一致或适配器缺陷）。
- **证据**：
  1. `DeletePendingArticle` 返回 `bool` 表示是否删除一行；
  2. `complete` 丢弃该返回值；
  3. 零行命中后仍追加 `article.delete` 成功审计、更新 operation completed 并提交；
  4. SQL 无法区分“Article 已不存在，可幂等完成”和“Article 仍存在但谓词不匹配，
     必须 fail closed”。
- **影响**：API 可返回 204、operation/audit 均显示成功，但 Article 仍存在甚至可见；
  后续重复 DELETE 又会因既有唯一 operation/状态不一致而不能可靠修复。
- **修复建议**：把删除结果建模为至少“deleted / already absent / present mismatch”；
  只有前两种可以完成，present mismatch 必须回滚并关闭 readiness/返回依赖错误。
- **复审条件**：测试分别覆盖命中删除、确实 absent、active 高版本、错误 operation
  绑定和其他谓词不匹配；后几类不得产生普通成功审计或 completed。

### MEDIUM — `ARTICLE-S4.1-R01-F005`：存在性锁文件在崩溃或删除失败后会永久阻塞 Journal

- **位置**：`internal/article/deletion_journal.go:341-354`
- **违反依据**：
  - A-DEL-03/04/05 要求锁失败后保持不可见并可重试收敛；
  - design `:1743,1760` 要求失败 fail closed，而不是形成无恢复所有者的永久锁。
- **触发场景**：进程在 `O_CREATE|O_EXCL` 创建锁文件后崩溃，或 defer 中
  `os.Remove` 失败。
- **证据**：
  1. 锁所有权只由路径是否存在表示；
  2. 进程退出不会自动释放；
  3. release 丢弃删除错误；
  4. 后续所有 Append 都把遗留文件当作活跃 writer，持续返回 busy；
  5. 项目内 recovery Journal 已使用句柄/flock 的进程生命周期锁，表明存在可复用边界。
- **影响**：所有后续删除在 T1 后永久停留 pending，重复 DELETE 无法自行收敛，需要
  人工判断并删除锁文件。
- **修复建议**：使用目标平台进程生命周期锁（POSIX flock、Windows 等价原语），
  验证 owner-only regular file；不要仅凭锁文件存在判断活跃所有者。
- **复审条件**：进程崩溃/句柄关闭后可重新取得锁；活跃双 writer 仍互斥；释放错误
  可观测且不会静默制造永久 busy。

## 5. 待决定与非阻断建议

### 待决定

1. **生产接线 owner 未登记**：有界搜索确认 `cmd/han-server/main.go` 当前只装配
   identity 路由；仓库中除测试外没有
   `NewFileArticleDeletionJournal`、`NewDeletionFinalizer`、
   `NewPermanentDeletionCoordinator` 的消费者，也没有 Article DELETE 路由。
   S4.1 写集又明确不含 production composition 文件。Lead 需指定后继任务/所有权；
   在此之前不能把本对象结论外推为生产 DELETE 可用。
2. **目标平台策略**：项目尚未确认生产是否支持 Windows。无论最终选择“支持”还是
   “启动时拒绝”，F001 当前把未证明的 fallback 当成功，不能接受。

### 非阻断建议

- S4.3 专项应使用真实 PostgreSQL 与目标持久卷验证 T1/T2 原子回滚、binding commit
  unknown、审计失败回滚、lease 并发、部分写、空间/权限/fsync 和进程崩溃；当前
  `deletion_test.go` 的内存 stub 不具备事务回滚语义。
- 后台 writer/finalizer 应定义低敏、可关联 trace/run 的积压、重试、永久失败和
  lock-busy 可观测性；不得记录 title/body/hash/email/PII。

## 6. Reviewer 实际验证

运行环境：`windows/amd64`，`go1.27.0`。

```text
go test ./internal/article -run 'TestPermanentDeletion|TestDeletionJournal|TestDeletionFinalizer' -count=1
go test ./internal/article -count=1
go test -race ./internal/article -count=1
go vet ./internal/article
gofmt -l internal/article/deletion_coordinator.go internal/article/deletion_finalizer.go internal/article/deletion_journal.go internal/article/command_service.go internal/article/handler.go internal/article/deletion_test.go
```

结果：

- 两条普通测试、race、vet 全部 PASS；
- `gofmt -l` 无输出；
- 定向搜索确认没有生产构造消费者或 DELETE 路由；
- 分支、HEAD 和六份固定对象起始哈希匹配。

Lead 提供的全仓 gofmt、`go mod verify`、vet、build、contract、race、test PASS
只记录为上游证据，不冒充 Reviewer 本轮重跑结果。

## 7. 未验证风险与限制

- 未运行真实 PostgreSQL；未验证真实事务回滚、行锁、lease SQL 并发和 commit
  outcome unknown；
- 未执行真实断电/崩溃、持久卷、磁盘满、权限、文件系统缓存或跨进程故障注入；
- Windows fallback 仅做代码路径和 Go 标准库实现核对，未以硬件断电证明；
- 未运行 production server、浏览器、部署或 E2E；
- 未重跑 Lead 已完成的全仓七项检查；
- S4.2/S4.3/S4.4/S4.5 尚未实施，旧备份重放和 retention/inventory 不在本轮证明范围；
- 当前对象是固定未提交字节，不能从 HEAD 重建逐文件 Git diff；本轮按固定哈希全文审查；
- 工作树中的其他修改和未跟踪文件均未评审或处置。

## 8. Gate 与复审条件

| Finding | 严重度 | 状态 |
|---|---|---|
| `ARTICLE-S4.1-R01-F001` | HIGH | **OPEN** |
| `ARTICLE-S4.1-R01-F002` | HIGH | **OPEN** |
| `ARTICLE-S4.1-R01-F003` | HIGH | **OPEN** |
| `ARTICLE-S4.1-R01-F004` | HIGH | **OPEN** |
| `ARTICLE-S4.1-R01-F005` | MEDIUM | **OPEN** |

- 未关闭阻断 findings：5
- **Code Review Gate：FAIL**

复审必须提供：

1. 对应每个 finding 的修订对象与固定 SHA256；
2. F001 的平台耐久策略及真实/可复核验证；
3. F002 的 pending binding 对账与后台消费者责任；
4. F003 的 tombstone 全量验证负例；
5. F004 的 deleted/absent/mismatch 三态证据；
6. F005 的崩溃释放与双 writer 互斥证据；
7. 相关定向、article 包、race、vet、格式和适用集成回归。

若修复需要改变已确认的 Journal/finalizer 契约、生产平台支持范围或 production
composition 所有权，先交 Lead/用户作方向决定；Reviewer 不替作者修改。

## 9. 结束哈希

报告写入后再次核对，分支仍为 `main`，HEAD 仍为
`6da8abbceb0fa197364810a92a8fa2d97b38ddca`，六份固定对象无漂移：

| 文件 | 最终 SHA256 |
|---|---|
| `internal/article/deletion_coordinator.go` | `6095A276AFF139CA7294BE60050E09D9A4CB07B83BD86113176D8BF4364C35E2` |
| `internal/article/deletion_finalizer.go` | `DB1CFE354FE7F6BAC693DE41E961D29163762E1F693DE08A46984B1B9A515F07` |
| `internal/article/deletion_journal.go` | `6B3325AA9E217F787D020B7726BF71B8CA08FE4C038D63D72EA35A26F1518159` |
| `internal/article/command_service.go` | `58A1FFDFD19C7C7214C07C28F90CF99F50A172B9AD054787A16C90BF08CC0603` |
| `internal/article/handler.go` | `874A2BAE6AE69C1C96A888E80AC1E35FE94452398B2CABE5F38C81696CE60F0A` |
| `internal/article/deletion_test.go` | `3E8101493E0660243DA7B49F463B8855764994439397BF272E01CFD5D86E5A07` |

报告自身最终 SHA256 在文件停止写入后计算，并在交回消息中报告。
