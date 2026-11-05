# article-management-S4.1 r03 定点复审

- Reviewer：Turing（原独立非作者 Reviewer）
- 日期：2026-10-23
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 关联报告：
  - r01 SHA256：`8BA84FB6A39539A10D7E3A1E5613D36CECB5EC27742F6C8A3B60DA84BFE0617D`
  - r02 SHA256：`D1B47617B17F2907F3B93F5D25449AC62578AF19D00E03AEAEF355C7D49F902C`
- **Code Review Gate：PASS**

## 1. 固定对象

开始复审、运行验证后及报告写入前，分支、HEAD 和六文件 SHA256 均与交接值一致：

| 文件 | 最终 SHA256 |
|---|---|
| `internal/article/deletion_coordinator.go` | `51B720A6ACFBDE81966AA959BF38898C68EB61622D7D428D3344DEE2D4CA0D1D` |
| `internal/article/deletion_finalizer.go` | `9ABEEE4D1314F52E44B25553C19B3E9989FFC396C738ED43375C3AA5CC319B17` |
| `internal/article/deletion_journal.go` | `E2539017B9C88EC6C60E3BBC0DA6DFF581EE885AE337E7B17E368129F7193343` |
| `internal/article/command_service.go` | `58A1FFDFD19C7C7214C07C28F90CF99F50A172B9AD054787A16C90BF08CC0603` |
| `internal/article/handler.go` | `874A2BAE6AE69C1C96A888E80AC1E35FE94452398B2CABE5F38C81696CE60F0A` |
| `internal/article/deletion_test.go` | `7D8841D39647EB0AE22E051B89FDE8C30917CF1732EEC6B7F337E5D4517F297B` |

r02 到 r03 仅 Journal 与 deletion test 的固定哈希变化；其余四文件保持 r02 固定版本。共享工作树其他内容未修改、回滚、格式化或归属判断。

## 2. 实际读取与方法

复用同版本完整 Reviewer / add-code-review 规则及 Go/security profile。实际读取：

- r01、r02 报告；
- 六个固定文件；其中四个未变文件复用 r02 已完整读取的同哈希正文，完整重读本轮变化的 `deletion_journal.go`、`deletion_test.go`；
- 同一 `tasks.md`、article `design.md`、`test-plan.md`、ADR（含 ADR-0020）；
- 有界直接接缝：`model.go`、`repository.go`、`policy.go`、`article_port.go`、`internal/audit/store.go`、`migrations/0003_article.sql`、生产 composition/router。

上述业务依据与直接接缝 SHA256 均保持 r02 读取版本。生产消费者搜索仍未发现 Journal/Coordinator/Finalizer 的生产构造和 DELETE 路由接线；该事项不属于本轮 F001 修复，但仍限制结论外推。

## 3. 两轮复审范围

### Round 1：F001 修订定点核对

核对 `FindByOperation` 的验证顺序、directory fsync、失败传播、独立 finalizer 状态不变性、reader 重试收敛及 Windows fail-closed。

### Round 2：回归与直接接缝核对

核对 F002～F005 原关闭依据、并发 reconciliation 测试事务 seam、operation 状态/lease、三态删除、Journal 验链、独立 writer 和临时文件语义；检查新增 CRITICAL/HIGH/MEDIUM/LOW finding。

## 4. Findings 状态

| Finding | r01 严重度 | r03 状态 |
|---|---:|---|
| `ARTICLE-S4.1-R01-F001` | HIGH | **CLOSED** |
| `ARTICLE-S4.1-R01-F002` | HIGH | **CLOSED，无回退** |
| `ARTICLE-S4.1-R01-F003` | HIGH | **CLOSED，无回退** |
| `ARTICLE-S4.1-R01-F004` | HIGH | **CLOSED，无回退** |
| `ARTICLE-S4.1-R01-F005` | MEDIUM | **CLOSED，无回退** |

### F001 CLOSED

#### 代码证据

`FindByOperation` 现在按以下顺序执行：

1. `readVerified` 读取完整目录快照；
2. 严格解析 JSON、拒绝未知/尾随数据；
3. 验证文件名与连续 sequence；
4. 验证 namespace、stage、article/operation/version、canonical UTC `deleted_at`、previous hash、key ID/version 与 HMAC；
5. 对 Journal 目录执行 `syncDirectory`；
6. 只有目录同步成功后才查找并返回 operation tombstone。

位置：`internal/article/deletion_journal.go:183-223,337-405,493-506`。

生产构造器把 `syncDirectory` 固定为 `syncDeletionJournalDirectory`，该函数实际打开目录并调用 `directory.Sync()`，同步或关闭失败均向上传播（`:87-100,424-441`）。因此 reader fsync 失败时，finalizer 在 Journal 读取阶段返回错误，不会进入 operation claim。

#### 耐久重试判断

writer 在发布正式路径前已完成记录文件 fsync（`:276-315`）；正式记录通过 no-replace hard link 发布，生产路径不覆盖或修改既有记录。独立 reader 在完整验证其读取快照后成功执行 directory fsync，可以为该快照中已读取的目录项建立新的耐久证明，不依赖 writer 的进程内成功状态。

若 writer 的 directory fsync 失败：

- 原 Append 返回失败；
- reader directory fsync 也失败时 fail closed；
- 后续 writer Append 可对既有记录重新同步后返回；
- 或独立 reader/finalizer 在后续 directory fsync 成功后继续收敛。

这关闭了 r02 所述“失败 Append 暴露正式路径后被无耐久证明的独立 finalizer 消费”窗口。

#### 回归测试证据

`TestDeletionJournal_DirectoryFsyncFailureBlocksIndependentFinalizerUntilProof`
（`deletion_test.go:486-582`）实际执行：

1. writer 成功完成文件写入/发布，但注入 directory fsync 失败；
2. 确认正式 regression record 已可见；
3. 使用独立 Journal 实例和真实 `DeletionFinalizer`；
4. reader directory fsync 失败时断言 operation 保持 `pending_journal`，且 claim/delete/complete/audit 均为零；
5. reader 获得耐久证明后重试，断言物理删除、单次审计和 completed 收敛。

原 `TestDeletionJournal_DirectoryFsyncFailureRequiresRetry` 同时保留 writer 重试收敛证据。

#### Windows

Windows production 构造器仍返回 `ErrDeletionJournalUnsupportedPlatform`，不会用 barrier 或测试注入冒充目录同步（`deletion_journal.go:87-100,434-441`；测试 `deletion_test.go:624-648`）。

### F002～F005 无回退

- **F002**：finalizer 仍凭 operation ID 从 Journal 读取，`pending_journal` binding 与 claim 位于同一事务；并发 reconciliation 仅完成一次。
- **F003**：Journal 的 canonical/sequence/chain/key/HMAC/namespace/stage/deleted_at 验证保持；篡改负例仍在 claim 前失败。
- **F004**：`deleted/already_absent/present_mismatch` 三态及 mismatch 不审计、不 completed 的逻辑未变。
- **F005**：无存在性锁永久阻塞；独立 writer 仍使用 no-replace hard-link 竞争，临时文件清理错误仍可见。

## 5. 并发测试 seam 复审

`deletionTransactorStub` 现在从 `Begin` 成功取得锁起，直到对应 transaction `Commit` 或 `Rollback` 才释放；`sync.Once` 防止重复释放（`deletion_test.go:1277-1321`）。这模拟同一 operation 上 PostgreSQL `FOR UPDATE` 在事务边界内持锁，而不是只在 `Begin` 调用期间短暂互斥。

`TestDeletionFinalizer_ConcurrentPendingReconciliationCompletesOnce`
通过 barrier 明确证明第二 finalizer 已尝试进入事务且被第一事务阻塞，再释放第一事务并断言 binding/delete/complete/audit 各一次（`:1010-1103`）。

该变更只加强测试替身的事务级串行语义；未修改生产代码、未放宽生产错误，且不会用 repository 方法自身的短锁掩盖原测试竞态。

## 6. 新 findings

本轮未发现新增 CRITICAL/HIGH/MEDIUM/LOW finding。

## 7. Reviewer 独立运行命令

环境：`go1.27.0 windows/amd64`。

```text
go test ./internal/article -run '^TestDeletionJournal_DirectoryFsyncFailureBlocksIndependentFinalizerUntilProof$' -count=100
go test ./internal/article -run '^TestDeletionFinalizer_ConcurrentPendingReconciliationCompletesOnce$' -count=1000
go test ./internal/article -run 'TestDeletionJournal_(DirectoryFsyncFailureRequiresRetry|DirectoryFsyncFailureBlocksIndependentFinalizerUntilProof|ProductionWindowsFailsClosed|ConcurrentIndependentWritersConverge)|TestDeletionFinalizer_(ConcurrentPendingReconciliationCompletesOnce|RejectsTamperedJournalRecordBeforeClaim|PhysicalDeleteThreeStateContract)' -count=100
go test ./internal/article -count=1
go test -race ./internal/article -run 'TestPermanentDeletion|TestDeletionJournal|TestDeletionFinalizer' -count=1
go vet ./internal/article
gofmt -l internal/article/deletion_coordinator.go internal/article/deletion_finalizer.go internal/article/deletion_journal.go internal/article/command_service.go internal/article/handler.go internal/article/deletion_test.go
```

结果全部 PASS；`gofmt -l` 无输出。

## 8. 未验证限制

- 未运行真实 PostgreSQL；未验证真实事务回滚、行锁、lease、外键和 commit outcome unknown。
- 未执行真实进程 kill、主机崩溃、断电、磁盘满、权限变化或目标持久卷故障注入。
- 当前 Windows 环境只验证 production fail closed；未在受支持 Unix 目标持久卷上执行真实 directory fsync/断电实验。
- 未运行 production server、后台 worker、部署、浏览器或 E2E。
- 生产 composition/router 尚未接入固定对象，不能把代码级 Gate PASS 表述为生产 DELETE 已可用。
- 未把 Lead 的验证结果冒充 Reviewer 本轮命令。

## 9. Gate

F001 已关闭，F002～F005 无回退，未发现新增 finding。

**Code Review Gate：PASS**

该结论仅绑定本报告列出的 HEAD、六文件 SHA256、依据版本和 Reviewer 实际命令；不替代后续真实 PostgreSQL、持久卷故障注入、生产接线或 E2E 验收。
