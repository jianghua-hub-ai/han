# article-management-S4.1 r01 修复定点复审 r02

- Reviewer：Turing（原独立非作者 Reviewer）
- 日期：2026-10-23
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：固定未提交六文件
- r01 基线报告：`.spec/wip/article-management/review-code-s4.1-r01.md`
- **Code Review Gate：FAIL**

## 1. 固定对象

开始复审、运行验证后及报告写入前，分支、HEAD 和六文件 SHA256 均与交接值一致：

| 文件 | 最终 SHA256 |
|---|---|
| `internal/article/deletion_coordinator.go` | `51B720A6ACFBDE81966AA959BF38898C68EB61622D7D428D3344DEE2D4CA0D1D` |
| `internal/article/deletion_finalizer.go` | `9ABEEE4D1314F52E44B25553C19B3E9989FFC396C738ED43375C3AA5CC319B17` |
| `internal/article/deletion_journal.go` | `BF32827EE6F5B3926967DB362BA25954F31605987AC1244DA59DF6170E896065` |
| `internal/article/command_service.go` | `58A1FFDFD19C7C7214C07C28F90CF99F50A172B9AD054787A16C90BF08CC0603` |
| `internal/article/handler.go` | `874A2BAE6AE69C1C96A888E80AC1E35FE94452398B2CABE5F38C81696CE60F0A` |
| `internal/article/deletion_test.go` | `F03FEBED8F735F269E69D12A4027C8E8A15932C7D9BBD835E3FE40669B944377` |

r01 到 r02 固定对象中，coordinator、finalizer、journal、test 的哈希变化；`command_service.go`、`handler.go` 哈希保持不变。共享工作树的其他未提交内容无法按 Git 提交归属到本次作者，本复审未修改、回滚、格式化或处置它们。

## 2. 实际读取与依据

完整读取并应用：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.codex/guidance/observability-standards.md`
- `.codex/guidance/tracing-standards.md`
- `.codex/profiles/go-service/README.md` 及其五份适用规则
- r01 报告、`tasks.md`、article `design.md`、`test-plan.md`、相关 ADR（含 ADR-0020）
- 六个固定实现文件全文

有界直接接缝：

- `internal/article/model.go`
- `internal/article/repository.go`
- `internal/article/policy.go`
- `internal/article/article_port.go`
- `internal/audit/store.go`
- `migrations/0003_article.sql`
- `cmd/han-server/main.go`、`main_test.go`
- 删除构造器、finalizer、Journal 与 DELETE 路由的生产消费者搜索

关键输入的本轮实际 SHA256：

| 输入 | SHA256 |
|---|---|
| r01 报告 | `8BA84FB6A39539A10D7E3A1E5613D36CECB5EC27742F6C8A3B60DA84BFE0617D` |
| `tasks.md` | `E64F0B17EB4F07C7633ACA5A6EFB27A6C2F4CC2DCBBD5EE365D482722840E275` |
| `design.md` | `70EE8FB04196FA78EB925133EC8C796E1B34F5B2060273BCE2A704DE3F797B91` |
| `test-plan.md` | `5A1FE258792CF281B27DE2B9E51131E1C46E738F8B3589A184ACFF383A9DDF3A` |
| ADR 汇总 `content.md` | `A0594229FCFDA0B8F243A476024BF51825F8DFB12671B41675FC270FA12DBF88` |

## 3. 两轮复审范围

### Round 1：原 findings 定点核对

逐行核对 F001～F005 对应修订及测试：平台目录同步、失败重试、operation-ID 对账、Journal 全链验证、三态物理删除、并发 finalizer、独立 writer 发布与临时文件清理。

### Round 2：消费者与状态/迁移有界核对

沿 `DeletionOperation`、Article 外键/状态约束、caller-owned audit transaction、可见性策略及生产 composition/router 核对契约；检查修复是否引入新的 CRITICAL/HIGH/MEDIUM/LOW 缺陷。未扩展为全仓审查。

## 4. F001～F005 结论

| Finding | r01 严重度 | r02 结论 |
|---|---:|---|
| `ARTICLE-S4.1-R01-F001` | HIGH | **OPEN** |
| `ARTICLE-S4.1-R01-F002` | HIGH | **CLOSED** |
| `ARTICLE-S4.1-R01-F003` | HIGH | **CLOSED** |
| `ARTICLE-S4.1-R01-F004` | HIGH | **CLOSED** |
| `ARTICLE-S4.1-R01-F005` | MEDIUM | **CLOSED** |

### HIGH — F001 OPEN：目录 fsync 失败后，未耐久记录仍可被独立 finalizer 消费

Windows barrier 已删除，生产构造器在 Windows 明确返回 `ErrDeletionJournalUnsupportedPlatform`，该部分修复成立（`deletion_journal.go:87-100,416-433`）。

但失败/重试语义仍未关闭原风险：

1. `persist` 先用 `os.Link` 把正式 `*.json` 路径发布，再删除临时路径，最后才执行目录同步（`deletion_journal.go:306-324`）。
2. 目录同步失败时 `Append` 返回错误，但已经发布的正式路径不会撤回，且没有“未耐久/待确认”状态。
3. `FindByOperation` 只调用 `readVerified` 做内容、链和 HMAC 验证，不执行目录同步，也无法区分该记录来自成功 Append 还是失败 Append（`deletion_journal.go:183-214,329-397`）。
4. finalizer 取得该记录后，可以把 `pending_journal` 对账为 `journaled` 并继续物理删除、审计和 completed（`deletion_finalizer.go:122-139,142-195`）。

**触发场景**：writer 完成文件 fsync 和正式路径 hard-link，随后 directory fsync 失败；API 正确收到非成功，但后台或另一进程的 finalizer 在重试完成前调用 `FindByOperation`。它会把该可见但尚未证明目录元数据耐久的记录当作权威 tombstone 并完成 T2。之后主机崩溃/断电可能丢失该目录项，造成数据库已物理删除但跨备份 tombstone 缺失。

**影响**：违反 design `:1741-1743,1747-1752`、ADR-0020 `:111-114,132` 和 A-DEL-03/04；“Journal 失败不得推进 finalizer”的保证未成立，仍存在旧备份恢复文章的风险。

现有 `TestDeletionJournal_DirectoryFsyncFailureRequiresRetry` 只证明调用方重试可再次同步，没有证明失败 Append 暴露的正式记录不能在重试前被独立 finalizer 消费。

**修复方向**：失败 Append 不能留下可被 verifier/finalizer 当成已耐久的正式记录；或 `FindByOperation` 在返回权威记录前必须建立并验证目录耐久性，失败则 fail closed。也可采用具备明确 prepare/commit 状态的发布协议，但不得以内容 HMAC 代替目录元数据耐久证明。

**复审条件**：新增确定性用例覆盖“正式路径已发布、directory fsync 失败、尚未成功重试时调用独立 finalizer”，断言不 claim、不删 Article、不写 Audit、不 completed；成功重试后才允许收敛。真实断电/持久卷验证仍由 S4.3 承担。

### F002 CLOSED：finalizer 可凭 operation ID 对账 pending_journal

- `Finalize` 只接收 operation ID/trace ID，并从 Journal 调用 `FindByOperation`（`deletion_finalizer.go:113-139`）。
- 持有 operation 行锁的同一 claim 事务内，`pending_journal` 先写 tombstone binding，再 claim（`:142-195`）。
- 双 worker 由 operation 行锁、状态和 lease 收敛；修订测试覆盖无 HTTP 重试对账与并发只完成一次（`deletion_test.go:664-702,912-975`）。

未发现原 F002 的残留直接缺陷。实际后台扫描/调度消费者尚未在生产 composition 中接线，仍属于待决定/后续任务，不能把本结论外推为生产后台已运行。

### F003 CLOSED：finalizer 从 Journal 读取，文件 Journal 完整验链

- finalizer 不再接受外部 tombstone struct，而是从 Journal 读取。
- `readVerified`/`verifyRecord` 校验严格 JSON、无尾随数据、文件名/sequence、namespace、stage、请求字段、UTC `deleted_at`、previous hash、key ID/version 和 HMAC（`deletion_journal.go:329-397,436-498`）。
- 负例覆盖 HMAC、previous hash、deleted_at、sequence、key、namespace、stage，且断言不 claim、不删除、不审计、不完成（`deletion_test.go:704-800`）。

### F004 CLOSED：物理删除严格区分三态

- repository 返回 `deleted`、`already_absent`、`present_mismatch`（`deletion_finalizer.go:28-36,385-420`）。
- mismatch 在生成审计和 completed 之前返回错误（`:226-258`）。
- 测试覆盖命中删除、确实不存在、active 高版本、pending 高版本、错误/缺失 operation binding，并断言 mismatch 不审计、不完成（`deletion_test.go:802-866`）。

### F005 CLOSED：遗留锁不再永久阻塞，独立 writer 不覆盖

- 当前实现不再以存在性锁文件判定活跃 writer；遗留 lock/temp 不参与 Journal 记录读取。
- 独立 writer 通过目标 sequence 的 `os.Link` no-replace 语义竞争；冲突者重读全链后使用下一 sequence，不覆盖已发布记录（`deletion_journal.go:268-327`）。
- 临时文件删除失败通过命名返回值和 `errors.Join` 暴露；冲突分支清理错误也不会被静默吞掉（`:280-320`）。
- 测试覆盖遗留 lock 不阻塞、遗留 temp 与两个独立 writer 收敛为唯一连续 sequence（`deletion_test.go:552-662`）。

本结论不等于已执行真实进程 kill、文件系统故障或断电测试。

## 5. 新 findings、待决定与建议

### 新 findings

除仍为 OPEN 的原 F001 外，本轮未发现新增的独立 CRITICAL/HIGH/MEDIUM/LOW finding。

### 待决定

1. 生产 composition/router 仍未发现 `NewFileArticleDeletionJournal`、`NewDeletionFinalizer`、`NewPermanentDeletionCoordinator` 或 Article DELETE 路由消费者；需由 Lead 指定后续接线与后台收敛 owner。
2. Windows 当前策略是生产 fail closed，而不是支持永久删除 Journal。若目标生产平台必须包含 Windows，需要上游明确平台策略及可证明的目录元数据耐久原语。

### 非阻断建议

- F001 回归应显式编排 failed Append 与独立 `FindByOperation/Finalize` 的时序，避免仅测试“原调用方随后重试”。

## 6. Reviewer 独立运行命令

运行环境：`windows/amd64`。

```text
go test ./internal/article -run 'TestDeletionJournal_(ProductionWindowsFailsClosed|DirectoryFsyncFailureRequiresRetry|StaleExistenceLockDoesNotBlockAppend|ConcurrentIndependentWritersConverge)|TestDeletionFinalizer_(ReconcilesPendingJournalWithoutHTTPRetry|RejectsTamperedJournalRecordBeforeClaim|PhysicalDeleteThreeStateContract|DuplicateAndConcurrentCalls_CompleteOnce|ConcurrentPendingReconciliationCompletesOnce)' -count=1
go test ./internal/article -count=1
go test -race ./internal/article -run 'TestPermanentDeletion|TestDeletionJournal|TestDeletionFinalizer' -count=1
go vet ./internal/article
gofmt -l internal/article/deletion_coordinator.go internal/article/deletion_finalizer.go internal/article/deletion_journal.go internal/article/command_service.go internal/article/handler.go internal/article/deletion_test.go
```

结果：

- 两条普通测试命令 PASS。
- 定向 race PASS。
- `go vet` PASS。
- `gofmt -l` 退出码 0、无输出。
- 这些通过结果不覆盖 F001 的跨实例 durability 时序缺口。

## 7. 未验证风险与限制

- 未运行真实 PostgreSQL，未验证真实行锁、lease、事务回滚、外键和 commit outcome unknown。
- 未执行真实进程崩溃、断电、持久卷、磁盘满、权限变化或文件系统缓存故障注入。
- 未在 Linux/Unix 目标持久卷上验证真实 directory fsync；Windows 仅验证生产构造器 fail closed。
- 未运行 production server、后台 worker、浏览器、部署或 E2E。
- 未把 Lead 的全仓门禁当作 Reviewer 本轮独立命令。
- 固定对象均为未跟踪文件，无法从 HEAD 重建逐行 Git diff；本轮按 r01 哈希和 r02 固定哈希全文复审。
- 共享工作树其他变更不在本次有界审查范围，也无法仅凭当前状态归属到本修复作者。

## 8. Gate

F002、F003、F004、F005 已关闭；F001 仍有一个 HIGH 严重度的原合同缺口。

**Code Review Gate：FAIL**

下一次复审只需绑定新的固定修订，重点核对 F001 的“directory fsync 失败后独立 finalizer 不可消费”证据及相应回归；Reviewer 不代作者修复，也不代 Lead 决定平台或生产接线。
