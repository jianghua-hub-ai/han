# article-management-S4.4 整卡关闭条件独立复核（closure-review-r01）

- 对象：`article-management-S4.4`（tasks.md:1558-1593）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮 `git rev-parse` 实核）
- 角色：非作者独立 Reviewer（add-reviewer / add-code-review，code 对象分支）；不参与本卡实施
- 方法：`task-execution-workflow.md` §10.3 六条 + `add-code-review` 两轮范围（对象事实 → 契约/装配 → 边界与未验证项）
- 授权来源：ADR-0028（`decisions/framework.md:815-858`）+ 用户 2026-10-31 项目级授权原文（见 ADR-0028 `:820`）
- **AI 代理评审，非真人签字**：本报告由 AI 代理 Reviewer 签发，记录授权来源与代理范围；不冒充真人评审、不代替人的业务接收或部署授权
- 写集：仅本目录 `report.md`、`verdict.md`；未改任何源码/测试/任务卡/ADR/既有证据，未做 Git 写，未清理
- 复跑约束：按任务要求**未自行运行任何 PG 集成测试**（避免与 Lead 并发建 schema）；PG 执行证据取自 Lead 日志并逐项核对计数

## 1. 评审基线与实测哈希

### 1.1 被评审文件实测（本轮 `Get-FileHash` + `Get-Item`）

| 文件 | SHA256 | 字节 | mtime |
|---|---|---:|---|
| `test/integration/recovery/article_replay_test.go` | 1BD2B0ED9385973533C611EEEF9CD17A9B70149AFBD5339829B9EDEF9906B0CD | 86718 | 2026-10-31 19:22:52 |
| `test/integration/recovery/replay_evidence_test.go` | C23C5C776627C4D065FE5E805A64BB582ED4EE9FB8A3FBB157FD1818949E88DB | 29878 | 2026-10-31 17:05:20 |
| `test/integration/recovery/old_backup_restore_test.go` | 36B13366F968FF9BA1BD1AA00AF90458A7EE89DD3A7FCBDFE69F6FF07317E536 | 15482 | 2026-10-31 19:22:52 |
| `internal/article/deletion_journal_hash_test.go` | 10F940D8C18984BD5D806BB9D5D6379DC644B22ACE626EB2BF6E4A59AC33FB48 | 1349 | 2026-10-31 19:22:52 |
| `cmd/han-server/main.go` | 6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727 | 173317 | 2026-10-31 19:26:09 |
| `cmd/han-server/recovery_runtime.go` | 900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7 | 35832 | 2026-10-31 19:25:50 |
| `.add/project/knowledge/tasks/tasks.md` | 6C6AD5F7BA7F1F90770EB52B388D08717193011718C3E981CCD977B9D156744E | 759386 | 2026-10-31 22:17:08 |
| `.add/project/knowledge/decisions/framework.md` | 5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A | 126590 | 2026-10-31 17:48:45 |

行数（`Get-Content` 计数）：article_replay 2154；replay_evidence 693；old_backup 385；hash 钉住测试 31。

### 1.2 与卡面/证据索引的交叉核对（事实：一致）

| 文件 | 登记位置 | 登记值 | 判定 |
|---|---|---|---|
| `article_replay_test.go` | tasks.md:1560；`sim-closeout-r01/index.json` `changed_files[].sha256_after` | 1BD2B0ED… | 一致 |
| `old_backup_restore_test.go` | tasks.md:1560；`sim-closeout-r01/index.json` | 36B13366… | 一致 |
| `deletion_journal_hash_test.go` | tasks.md:1560；`sim-closeout-r01/index.json` | 10F940D8… | 一致 |
| `replay_evidence_test.go` | `harness-publication-r01/index.json`（本卡第二路径） | C23C5C77… / 29878 B | 一致（该文件本轮未改动；卡面 :1560 只登记三文件，不含本文件） |

补充：`review-sim-closeout-r01.md` 报告中 deletion hash 有 1 字符笔误（`…ACE626BB2BF…`），`review-sim-closeout-r01-disposition.md` 已记录更正并以实测/索引为准；报告作为独立记录未改，属已披露的记录性笔误，非事实偏差。

## 2. 生产装配链路结论（原阻塞是否解除）

### 2.1 seal 回调位于真实 production composition —— 成立

调用链全部位于 `runComposed`（`cmd/han-server/main.go:396`）这一唯一 production root：

- `:573` `dependencies.composeBusiness(...)` → 默认 `:4041 composeBusiness = newProductionBusinessHandler` → `:2424` → `:2436 buildArticleHandlers` → `:3094 newArticleComponents` → `:3288 newArticleReplayProvider`
- `:634` `newRecoveryRootAssembly(ctx, recoveryRootInputs{ ... seal: func(sealed recovery.Runtime) error {...} })`，同一 `runComposed` 顺序内、非测试专用分支、非 `_test.go` 文件

### 2.2 recovery runtime 实际调用 seal 且错误阻断启动 —— 成立

`cmd/han-server/recovery_runtime.go:611-615`：

```go
if inputs.seal != nil {
    if err := inputs.seal(runtimeValue); err != nil {
        return nil, fmt.Errorf("seal recovery publication ports: %w", err)
    }
}
```

`seal` 返回错误 → `newRecoveryRootAssembly` 返回错误 → `main.go:651-653` 包装 `compose recovery runtime` 并终止启动。无静默降级。

### 2.3 `replaySlot.source` 先于 recovery assembly 赋值 —— 成立

- `:565 replaySlot := &articleReplaySourceSlot{}`，`:589` 传入 composeBusiness
- `:3330-3331` `if slot := b.composition.articleReplaySource; slot != nil { slot.source = source }`
- composeBusiness 在 `:573` 执行，recovery assembly 在 `:634` 执行 → 顺序保证 seal 读到的 `replaySlot.source` 已赋值

### 2.4 `articleEvidence` 单一 Store 复用，`NewPostgresReplayEvidenceStore` 仅为 nil 保护回退 —— 成立

- `:566` 唯一构造 `articleEvidence`；`:568` 主构造点
- `:588` 传给 composeBusiness；`:636` 作为 `closures` 传给 recovery root
- `:3333` `store := b.composition.articleEvidence`；`:3334-3339` 仅 `store == nil` 时回退新建
- `recovery_runtime.go:495-501` 同语义：仅 `inputs.closures == nil` 才自建
- 生产 `NewPostgresReplayEvidenceStore` 命中：`main.go:568`（主构造）、`main.go:3335`（nil 回退）、`recovery_runtime.go:497`（nil 回退）；其余命中均在测试文件

结论：**S4.4 原阻塞“生产 `main.go` 未调用 `NewReplayPublicationPorts`（归 identity-auth-S4.20）”已解除**。生产全仓 `NewReplayPublicationPorts` 仅 `main.go:641` 一处（另为测试侧 `recovery_production_chain_integration_test.go:1111` 等）。

### 2.5 但“生产重放入口”仍未接通（重要边界，非本卡写集缺陷）

- seal 的语义是“封存引用、不调用端口”：`internal/article/replay_publication.go:82-83` 注释 `NewReplayPublicationPorts only seals references; it never invokes a port`；真实消费方是 runtime admission 的 `ReadPublishedClosure`（`internal/recovery/admission.go:500/551-553`）。
- `productionArticleComponents.gate`（`main.go:2411-2414`、`:3286-3293`）在 `buildArticleHandlers`（`:3094-3104`）**只被丢弃**：该方法只取 `components.worker`、`components.handler`；全仓无 `components.gate` 读取点。
- `recovery.NewCoordinator`（`internal/recovery/coordinator.go:32`，其中 `:438` 调 `gate.ReplayAndVerify`）**生产调用方 0**，仅测试调用。
- 生产 root 未注入 `consumer`：`main.go:634-650` 只传 `pool/plan/prepared/closures/seal`；`recovery_runtime.go:551-553` 因此使用 `recoveryClosedConsumer{}`，其 `Consume` 返回 `ErrAdmissionClosed`（`recovery_sources.go:576-581`）。

因此：解除的是“未调用 `NewReplayPublicationPorts`”这一具体阻塞；**“生产恢复 HTTP 重放端点可用”仍未证明**，与 S4.20 卡面 `:2881`“待做”项一致（但 S4.20 已 `[x]`），按跨卡边界登记为 `S44-CR-F002`。

## 3. 卡面关闭条件逐条判定（tasks.md:1591）

原文：“三类 tombstone、operation/audit 对账、双 backup 并发、输入漂移、双次 ReadStatus 与旧备份恢复通过；缺 inventory/RPO/RTO 时不关闭真实演练部分；独立 Review 通过。”

| # | 条件 | 当前修订证据（实读） | 真实 PG / 模拟 | 判定 |
|---|---|---|---|---|
| 1 | 三类 tombstone | `article_replay_test.go:59-66` `CountsAndReadback`（AllAbsent0N / AllPendingN0 / Mixed2And1）；`old_backup_restore_test.go:105-135` 三目标 | 真实 PG（Finalizer/Store/FactReader 真实）+ 合成 seed | 覆盖 |
| 2 | operation/audit 对账 | `article_replay_test.go:75-77` `OperationAuditDriftRepublish`（Operation/Audit 两支）；`old_backup_restore_test.go:137-165` 断言 `deletion_operations.status=completed` 且 `audit_events` 计数=1 | 真实 PG；drift 由 SQL 注入模拟 | 覆盖 |
| 3 | 双 backup 并发 | `replay_evidence_test.go:31-34` `DifferentBackupFirstBinding`（真实 backend Begin/blocking、PK 23505） | 真实 PG；manifest 合成 | 覆盖 |
| 4 | 输入漂移 | `old_backup_restore_test.go:167-197` `ReadyDriftInvalidatesEvidence`（`unknown/evidence_stale` + 重发布新 ref/digest + 历史保留）；`replay_evidence_test.go` 漂移用例 | 真实 PG；drift 注入模拟 | 覆盖 |
| 5 | 双次 ReadStatus | `old_backup_restore_test.go:121-132`、`:187-194`：两次字节级相等且零额外写入 | 真实 PG | 覆盖 |
| 6 | 旧备份恢复 | `old_backup_restore_test.go:105-135` `RestoreAndReplayThreeTombstones`：**全部 synthetic**（文件头 `:5-13` 自述不证明真实旧备份恢复；`:46-52` `SyntheticManifestIsNotRealBackup` 断言 `synthetic_only=true`/`production_use=forbidden`；“恢复后状态”是合成 seed，`TombstonesReplayed=0/AlreadyAbsent=3`） | **模拟 fixture（未真实验证）** | **仅模拟子集**（不得写成真实旧备份恢复 PASS） |
| 7 | inventory 缺失 fail-closed | `old_backup_restore_test.go:199-232` `InventoryUnavailable`：`ErrBackupInventoryUnavailable` + 全表零写入 + 零 probe/journal 读取 | 真实 PG；缺文件模拟 | 覆盖 |
| 8 | 真实 inventory / RPO / RTO | 无真实 inventory 签发/挂载，无 RPO/RTO 实测 | 未验证 | **NOT CLOSED**（卡面 `:1591` 明示“缺 inventory/RPO/RTO 时不关闭真实演练部分”，属设计内） |
| 9 | 独立 Review | `review-code-s4.4-old-backup-r02.md`（有界 PASS，SHA256 1BC70302…）+ `review-sim-closeout-r01.md`（PASS WITH MINOR，SHA256 3BD91393…）+ 本整卡复核 | — | 覆盖（有界，均声明不对整卡 readiness 背书） |

核心判断：条件 1-5、7 有当前修订的真实 PG 证据；条件 6 只有模拟 fixture 子集；条件 8 按卡面口径本就不关闭。故本卡只能在“模拟子集 + 登记真实演练不关闭”的口径下关闭。

## 4. 证据索引与 Lead 复跑日志核验

- `sim-closeout-r01/index.json`（D2A07736…/7205 B）：`changed_files` 三项与实测逐项一致；`scope.production_code_changed=false`；`not_closed` 登记真实 POSIX/keyring/旧备份/identity 联合流与 Case18 `NOT RUN`；`production_readiness_claimed=false`。
- `harness-publication-r01/index.json`（6878BFE6…/15186 B）：仍登记上一轮旧值（`article_replay_test.go` 0417235496…/86039 B/1990 行；`old_backup_restore_test.go` E0C202F2…/15848 B/350 行），且 `not_proven` 仍写“cmd/han-server does not call NewReplayPublicationPorts at this HEAD”；文件内无 `superseded_for_current_tree` 指针 → `S44-CR-F001`。
- Lead 复跑日志 `.spec/wip/article-management/evidence-s4.4/lead-closure-r01/s44-lead-rerun.log`（E1EAA235…/105322 B）：尾行 `PASS` / `ok han/test/integration/recovery 20.478s`；`--- PASS` 33 行、`--- FAIL` 0、`--- SKIP` 0、`DATA RACE` 0；三个顶层用例 `TestArticleReplayPostgres`（含 16 叶）、`TestOldBackupRestore`（4 子测试）、`TestArticleReplayEvidencePostgres`（3 叶）全 PASS；schema 均 `retained=true`、`resources=observer_released_cancelled_pool_closed acquired=0`。

## 5. Findings

| ID | 严重级 | 位置 | 影响 | 状态 |
|---|---|---|---|---|
| S44-CR-F001 | MEDIUM | `.spec/wip/article-management/evidence-s4.4/harness-publication-r01/index.json`（`changed_files` 两项、`not_proven` 第二项） | 登记的是已被 `sim-closeout-r01` 取代的旧哈希/字节，且 `not_proven` 的“main.go 未调用 NewReplayPublicationPorts”在本 HEAD 已失效；缺 `superseded_for_current_tree` 指针，后续读者可能误判阻塞未解除或误用旧哈希核验当前树 | OPEN（建议 Lead 追加 superseded 指针/新索引段；不阻断本次关闭，现行权威索引为 sim-closeout-r01） |
| S44-CR-F002 | LOW | `cmd/han-server/main.go:2411-2414`、`:3286-3293`、`:3094-3104`；`internal/recovery/coordinator.go:32/438`；`cmd/han-server/recovery_runtime.go:551-553`、`recovery_sources.go:576-581` | `productionArticleComponents.gate` 无生产消费者；`recovery.NewCoordinator` 生产 0 调用；root 未注入 consumer → `recoveryClosedConsumer`（`ErrAdmissionClosed`）。故 seal 已封存 publication 配置，但生产“旧备份重放/二次 ReadStatus”HTTP 路径未接通 | OPEN（已披露边界；不得据此宣称生产恢复重放入口可用） |
| S44-CR-F003 | LOW | `cmd/han-server/main.go:633`（`if recoveryPlan.present`） | 无 recovery 配置启动时 Store 已创建但未 seal，当前无断言；异常配置组合下 seal 的负向行为未被覆盖 | OPEN（设计条件，登记为边界） |
| S44-CR-F004 | LOW | `internal/article/replay_publication.go:163-180`；`internal/article/deletion_journal_hash_test.go:10-30` | `validateReplayPublicationSnapshot` 只校验结构与形状，不重算 `deletionRecordHash`；hash 钉住测试只保证“投影函数 = 钉住值”，若投影变更需人工同步（minor-01 的残余面） | OPEN（已登记残余，非阻断） |
| S44-CR-F005 | LOW | `test/integration/recovery/article_replay_test.go:518-535`（`s44PublicationRuntime.CheckPhase`）、`old_backup_restore_test.go:368-385` | `CheckPhase` 在当前三路径不可达（无 `EnsureCurrentPublication`/`ImportLegacyPublication` 调用），正确但未被执行；测试通过不能证明该实现 | OPEN（`S15-SIM-F003` 已披露，非阻断） |

无 HIGH、无新增阻断项。已登记并复核为非阻断的既有 LOW：minor-01（CLOSED，单一 `s44PublicationRecordHash` + hash 钉住）、minor-02（CLOSED，`CheckPhase` 补齐）、`S15-SIM-F001`（已修复，Trace 掩码）、`S15-SIM-F002`（已修复，guard-open 正控）、`S15-SIM-F003`（接受为已披露边界）、`S15-SIM-F004`（接受为已披露并发写者事实）、`R02-N01`（Lead 已修正 index 的 retained ndjson 摘要）、`R02-N02`（`old_backup_restore_test.go:244-251` 已注明两套 assembler 等价且登记 R02-N02；两套 assembler 仍共存，属已披露残余）、`B13H-GATE-R02-L02`（nil 回退缺运行期“无第二 Store/pool”断言，跨批次已披露）。

## 6. 未验证边界（关闭后仍不得外推）

1. 真实 POSIX Journal / `fsync` / `flock` / 崩溃与磁盘故障韧性：未验证（Windows 公开 FileJournal 明确 unsupported）。
2. 真实 keyring / HMAC / HSM 运行时：未验证（测试用合成 keyring 与 HMAC 向量）。
3. 真实旧备份恢复与 RPO/RTO：**未验证**（`old_backup_restore_test.go` 全部 synthetic；卡面 `:1591` 明确不关闭真实演练部分）。
4. 真实 backup inventory 的签发/挂载/授权：未验证（仅“缺失即 fail-closed”）。
5. identity 双 namespace / `MarkConsumed` / 联合开流：未验证。
6. 真实部署 / 真实 E2E：未运行；Case18 `NOT RUN/unknown`。
7. `enter`/`install` 与 recovery secret candidate：代码层仍为 `ErrAdmissionClosed` 占位（`recovery_runtime.go:480-486`、`:582-584`），后续承接任务未登记（unknown）。
8. 生产恢复重放入口（Coordinator/gate 装配）：未接通（见 S44-CR-F002）。
9. 无 recovery 配置时 seal 行为：无断言（S44-CR-F003）。
10. 进程级唯一 runtime/Store/pool 的运行期断言、性能/P95：未验证（仅编译期/指针层与测试内计数）。
11. 测试执行时点：Lead 日志 `ok 20.478s` 为当前 revision 执行；本轮未自行复跑 PG（遵守与 Lead 串行约束），故“测试通过”以该日志与 fail/skip/race 计数为准。

## 7. ADR-0028 合规与可关闭性

- 约束 2（分级验证）：可执行层已实际执行并留证（本轮 `go build ./...`、`go vet` 普通/`-tags=integration`、`gofmt -l` 均 exit 0；PG 集成由 Lead 实跑留日志，fail=0/skip=0/race=0）；真实环境层以模拟替代并显式标注。
- 约束 3（不得伪造 PASS）：旧备份恢复未被写成真实 PASS，`SyntheticManifestIsNotRealBackup` 与文件头自述明确 synthetic；未宣称 production readiness。
- 约束 4（AI 代理评审）：本报告记录授权来源（ADR-0028 + 用户 2026-10-31 原文）、代理主体（AI 代理 Reviewer）与范围，并声明“AI 代理评审，非真人签字”。
- 约束 6（任务关闭）：允许在 ADR 证据范围内关闭，但凡含模拟/未真实验证证据，必须同时登记未验证边界与残余风险。

**结论：在“登记未真实验证边界 + AI 代理评审、非真人签字”的前提下，S4.4 可以关闭**，但关闭记录必须同时写明：

- 条件 6“旧备份恢复”仅完成**模拟 fixture 子集**，真实旧备份恢复与真实演练部分按卡面 `:1591` **不关闭**；
- inventory / RPO / RTO 缺失 → 真实演练部分 `NOT CLOSED`；
- 生产重放入口未接通（S44-CR-F002），不得宣称生产恢复重放 HTTP 端点可用；
- 不得宣称 production readiness、真实部署/E2E、真实 POSIX/keyring/HMAC 通过；
- 关闭记录应引用本报告与 `sim-closeout-r01` 证据，并处置 `S44-CR-F001`（追加 superseded 指针）。

## 8. 本轮实际执行命令与结果（只读）

| 命令 | 结果 |
|---|---|
| `git rev-parse HEAD` / `git rev-parse --abbrev-ref HEAD` | `6da8abb…` / `main` |
| `Get-FileHash -Algorithm SHA256` + `Get-Item`（8 个对象） | 见 §1.1；与卡面/索引核对见 §1.2 |
| `go build ./...` | exit 0 |
| `go vet ./cmd/han-server` | exit 0 |
| `go vet -tags=integration ./cmd/han-server` | exit 0 |
| `go vet -tags=integration ./test/integration/recovery` | exit 0 |
| `gofmt -l`（main.go、recovery_runtime.go、三测试文件、hash 钉住测试） | 输出为空，exit 0 |
| `rg` 静态核对（gate 消费者、`NewReplayPublicationPorts`、`NewPostgresReplayEvidenceStore`、`NewCoordinator`、`ReplayAndVerify`、`ArticleSnapshots`、closed consumer） | 见 §2 |
| PG 集成测试 | **未自行运行**（任务禁止，避免与 Lead 并发建 schema）；引用 Lead 日志并核计数（§4） |

## 9. 写后自校验

- 本目录仅新增 `report.md`、`verdict.md`。
- 写入后复算两文件 SHA256/字节，并复算 §1.1 六个被评审对象哈希，确认 drift=0（若发现外部改动，记录并报告，不回滚）。