# S4.4 第三路径（合成 fixture 旧备份恢复测试）独立评审 R02

**对象与修订**
- 复审对象：`test/integration/recovery/old_backup_restore_test.go`，SHA256 `E0C202F25A11A7584E43B734A1BD7FE7A73CA9DA113795CE08EA666C92AE61DA`，15848 B / 385 行（非空 350 行）
- 同树共享 harness（范围外，仅作依赖核对）：`article_replay_test.go` `0417235496741176A98D28548BCF9789C3FF9AD9E13D20B075342514FF2DF560`（86039 B）、`replay_evidence_test.go` `C23C5C776627C4D065FE5E805A64BB582ED4EE9FB8A3FBB157FD1818949E88DB`（29878 B）
- 本轮修复证据：`.spec/wip/article-management/evidence-s4.4/harness-publication-r01/`（`index.json` `4E3D54C1…`、`lead-verify-r01.log` `BCE990A6…`、`retained-resources.ndjson` `ACE4E82D…`，102 行）
- 旧路径证据：`.spec/wip/article-management/evidence-s4.4/old-backup-r01/`（`index.json` r03 `6CA7332B…`）
- 工作区：`D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abb`；只读评审，未做任何 Git 写操作
- 复审方式：`add-code-review` revision 路径，对 R01 的 2 major / 4 minor 逐条复核，并独立核验本轮修复自述与 Lead 裁定
- 评审日期：2026-10-31

## 结论

**PASS（有界：仅第三路径 test-only 技术子集）；S4.4 整卡仍不可关闭。**

- 第三路径 `TestOldBackupRestore` 4/4 在真实 PostgreSQL 18.6 上通过，且确实经过真实 `article.PostgresReplayEvidenceStore` / `article.NewReplayPublicationPorts` / `article.NewDeletionReplayProvider` / `article.NewProductionDeletionReplaySource` / `article.DeletionFinalizer` 写路径；未发现 blocker，未发现伪造 PASS。
- R01 六条 findings：`major-01`、`minor-03`、`minor-04` **CLOSED**；`major-02` 作为“已知边界已如实登记”**CLOSED**，但**生产装配缺口本身仍 OPEN**（归 `identity-auth-S4.20`）；`minor-01`、`minor-02` **OPEN（非阻断）**。
- 新发现问题 2 条，均为 minor（非阻断）：`R02-N01` 证据索引 `retained_resources` 哈希/计数与实际文件不一致；`R02-N02` 两套 assembler 共存的分叉风险及过时注释。
- R01 的“整卡不关闭”结论复核后仍成立：生产 composition root 仍未调用 `NewReplayPublicationPorts`；真实旧备份、真实 POSIX Journal/fsync/flock、真实 keyring/HMAC、identity 双 namespace/MarkConsumed/联合开流、RPO/RTO 均未证明；双 backup 并发等关闭条件未覆盖。

## 逐条复核表

| R01 finding | R01 严重度 | R02 判定 | 关键证据 |
|---|---|---|---|
| major-01：测试内重复执行非幂等 `0011` | major | **CLOSED** | `article_replay_test.go:237-255`、`:260-279`；`old_backup_restore_test.go:290-293`；run-05 日志 `article_replay_test.go:254 stage=applied_in_tx` → `old_backup_restore_test.go:109/171 stage=already_applied` |
| major-02：测试内重建 ports 绕过生产装配 | major | **CLOSED（仅“边界登记”）**；生产缺口 **OPEN** | `old_backup_restore_test.go:246-288` 仍为合成组装；`old-backup-r01/index.json:79` 与 `harness-publication-r01/index.json:153` 明确登记；`cmd/han-server/main.go:3290` 仍直接建 Store、未调用 ports；`tasks.md:2859` 归 S4.20 |
| minor-01：合成 `s44OldBackupRecordHash` 复刻未导出 `deletionRecordHash` | minor | **OPEN（非阻断，LOW）** | `old_backup_restore_test.go:352-362`；共享侧另有 `article_replay_test.go:471-489`；生产 `internal/article/deletion_journal.go:690-694`、`:730-742`、`:757-768` 当前字节等价 |
| minor-02：合成 Runtime 未实现 `CheckPhase` | minor | **OPEN（非阻断，LOW）** | `old_backup_restore_test.go:366-385` 仅实现 `ReadPermitBinding`；共享 `article_replay_test.go:493-512` 同样；生产 `CheckPhase` 仅在 `replay_publication.go:663/709/762` 的 `EnsureCurrentPublication` / `ImportLegacyPublication` 路径 |
| minor-03：`run-01.log`/`run-02.log` 重复、index path/run 不一致 | minor | **CLOSED** | `old-backup-r01/index.json:22-35`：`final_log` 指向 `run-02.log`，`duplicate_copies` 明确 `run-01.log` 为字节相同副本；两文件 SHA 均为 `337DB5B4…` |
| minor-04：`retained_schemas` 6 条 vs ndjson 12 条 | minor | **CLOSED** | `old-backup-r01/index.json:58-71`：`retained_schema_count: 12`，12 条含 `has_log`；`retained-resources.ndjson` 实际 12 行 `4953D174…` |

### major-01（CLOSED）

R01 事实：旧测试在 `s44OldBackupApplyPublicationMigration` 内直接执行非幂等 `migrations/0011_article_replay_publication.sql`；一旦共享 harness 把 `0011` 纳入 `migrate()`，同一 schema 会二次执行 DDL 并因 duplicate constraint/table 失败。

修复核验：
- `article_replay_test.go:237-255` 的 `migrate()` 在 0001–0009 单事务提交后调用唯一入口 `s44ApplyPublicationMigration`（`:254`）。
- `article_replay_test.go:260-279`：先 `to_regclass('<schema>.article_replay_publications')`（`:264-265`），已存在则记录 `stage=already_applied` 并返回（`:266-268`），否则在事务内执行并提交（`:270-279`）。
- `old_backup_restore_test.go:290-293` 已改为纯委托，无第二份 DDL。
- 证据日志 `harness-publication-r01/run-05.log:24-26`：`article_replay_test.go:254 … stage=applied_in_tx` 后，`old_backup_restore_test.go:109 … stage=already_applied`；`:43-45` 同样为 `:171` 的 `already_applied`。本次独立复跑日志同样出现该序列（见“复核命令与证据引用”）。

结论：判定 CLOSED。单入口 + 存在性探测使“重复执行非幂等 0011”在当前树与未来 harness 变更下均不可达。残余风险：单入口的正确性依赖所有调用方都经过 `s44ApplyPublicationMigration`；当前仅共享 `migrate()` 与旧路径委托两处，可接受。

### major-02（CLOSED 作为边界登记；生产缺口 OPEN）

R01 事实：测试在 `s44OldBackupConfigurePublication`（`old_backup_restore_test.go:246-288`）内重建 Source/Store/Provider 并注入合成 Runtime/Journal/PublicationIDs，绕过生产 composition root；`cmd/han-server/main.go:3290` 直接构造 `NewPostgresReplayEvidenceStore`，`rg` 在 `cmd/`、`internal/` 生产代码中未发现 `NewReplayPublicationPorts` 调用。

本轮变化与独立核验：
- 生产代码未改（写集允许但未动）：`main.go:3290-3304` 仍直接建 Store 并交给 `NewDeletionReplayProvider`，`:3290` 未调用 ports。
- 证据已明确登记：`old-backup-r01/index.json:79` 的 `production_assembly_not_covered` 原文声明“publication ports are constructed inside the test; cmd/han-server/main.go does not call NewReplayPublicationPorts at this HEAD. PASS does not imply production restore path works. Tracked as identity-auth-S4.20.”；`harness-publication-r01/index.json:153` 在 `not_proven` 中同样声明 production wiring 未证明。
- 测试自身边界：`old_backup_restore_test.go:5-13` 明确声明 manifest/inventory/keyring/HMAC/Journal snapshot/publication ID/runtime binding 全为合成数据，并逐项列出不证明的内容。
- 缺口归属：`tasks.md:2859` 将 production root 的恢复链打通登记为 `identity-auth-S4.20`。

判定理由：R01 的 major-02 是针对“本测试/证据可能被读成生产可用”的 claim 缺陷。当前测试与证据均显式限定为 test-only 合成组装、缺口有持久登记、生产装配归独立卡；同时该测试确实经过真实 Store/Provider/SQL 写路径（不是 mock 假绿）。因此就本测试与证据链而言 **CLOSED**。但必须同时记录：**生产装配缺口本身仍 OPEN**，不得由本测试或本报告关闭，也不得据此释放生产恢复 Gate。

### minor-01（OPEN，非阻断，LOW）

- 现状：`old_backup_restore_test.go:352-362` 仍自建 `s44OldBackupDiskRecord` 并计算 `sha256:` 前缀 hash；共享 harness 另有一份 `s44PublicationRecordHash`（`article_replay_test.go:471-489`）。生产 `deletionRecordHash`（`deletion_journal.go:690-694`）使用 `deletionDiskRecordFromModel`（`:757-768`），字段顺序/JSON tag 与 `deletionDiskRecord`（`:730-742`）当前一致，`DeletedAt` 为 `UTC().Format(time.RFC3339Nano)`（`:757-768` 调用的 MAC payload 构造，`:775-777` 附近）。
- 当前等价性：字段集合、顺序、tag、时间格式、`sha256:` 小写十六进制前缀与生产一致；因此当前 4/4 PASS 的 snapshot 结构自洽。
- 脆弱点：生产 `validateReplayPublicationSnapshot`（`replay_publication.go:163-178`）只校验 `sha256:<64 hex>` 格式、prefix 序列连续性与 head 相等，**不会重算** record hash。若未来生产修改 `deletionRecordHash` 的字段投影（增删字段、改 tag/顺序），测试不会失败，而是继续用旧投影“自洽通过”，形成语义漂移（假绿而非误报）。这比普通重复代码更值得记录，但因为测试已明确不证明真实 Journal/HMAC，且漂移不会造成生产误判（只影响测试模拟保真度），故维持 **LOW / 非阻断**。
- 建议：生产侧导出可复用的 hash helper，或至少把两份测试 helper 合并为一份并加注释指向 `deletion_journal.go:690`；迁出本测试写集时处理。

### minor-02（OPEN，非阻断，LOW）

- 现状：`old_backup_restore_test.go:366-385` 的 `s44OldBackupRuntime` 嵌入 `recovery.Runtime`（nil 接口），只显式实现 `ReadPermitBinding`；共享 `s44PublicationRuntime`（`article_replay_test.go:493-512`）同样只实现该方法。
- 当前路径：`Publish`/`ReadCurrent` 只经 `checkReplay` → `Runtime.ReadPermitBinding`（`replay_publication.go:132-147`，调用点 `:139`）；`CheckPhase` 仅出现在 `EnsureCurrentPublication`（`:663`）、`ImportLegacyPublication`（`:709`、`:762`），本测试未走这些路径（`ReplayAndVerify` 走 `Store.Publish`，`ReadStatus` 走 `Store.ReadCurrent`）。因此当前无 nil panic、无假绿。
- 风险：若未来生产把 `CheckPhase` 前移到 `Publish`/`ReadCurrent` 的公共前置，合成 Runtime 会因 nil 接口方法调用直接 panic，测试失败方式将从断言失败变为崩溃；这不是当前缺陷，但需要显式防护。
- 建议：为合成 Runtime 显式实现 `CheckPhase`（返回与 `ReadPermitBinding` 一致的允许结果），或至少加编译期断言与注释说明“仅覆盖 ReadPermitBinding”。

### minor-03（CLOSED）

- `old-backup-r01/index.json:22-35`：`final_log.path` 指向 `run-02.log`，`run: "run-02"` 与文件一致；`duplicate_copies` 明确 `run-01.log` 是 `run-02.log` 的字节相同副本、“not an independent run”。
- 独立核验：`run-01.log` 与 `run-02.log` SHA256 均为 `337DB5B4F20934C751661B41314959744231CEC1BA314A53EA777C04DCE11D9C`；`run-02.log` 为 PASS 4/4（5.03s），`run-01.attempt1.log` 为修复前 FAIL 2/4（`old_backup_restore_test.go:110/160: first durable Ready did not create exactly one evidence row`），与 R01 记录的修复前状态一致。

### minor-04（CLOSED）

- `old-backup-r01/index.json:58-71`：`retained_schema_count: 12`，数组 12 条，均含 `has_log`；其中 `lead-diagnostic-probe` 3 条为 `has_log:false`（已如实披露），其余 9 条 `has_log:true`。
- 独立核验：`old-backup-r01/retained-resources.ndjson` 实际 12 行，SHA256 `4953D17495D9E2B75DDE981AC9735C32554649B876DAABDBC5395691411E9E78`，与 index 一致。

## 新发现问题

### R02-N01 [minor，非阻断] 共享 harness 证据索引的 retained resources 哈希/计数与实际文件不一致

- 位置：`.spec/wip/article-management/evidence-s4.4/harness-publication-r01/index.json:131-137`
- index 记录：`sha256 = A9833A8A56C3B19B9C77AEB4872BB5C3DDF90D2511383F98812D2A029D75EAC5`，`record_count = 80`，`attributed_by_transcript = 39`，`unmapped_debug_iterations = 41`（39+41=80）。
- 实际文件：`retained-resources.ndjson` SHA256 `ACE4E82DD55424B3B47B582CB04B4DFB706D9610A517D648CDE7C91E3844B522`，102 行；其中 80 行不带 `oid`，22 行带 `oid`。
- 影响：按 index 校验会失败；审计者无法用 index 直接验证 102 行文件。所有 102 行实际保留（`policy: nothing dropped`），不影响第三路径 4/4 技术结论，但削弱证据链自洽性。
- 附带观察：`harness-publication-r01/index.json` 未登记 `lead-verify-r01.log` 的 SHA256（任务卡与实测均为 `BCE990A6…`），建议一并补齐。
- 建议：把 `retained_resources.sha256` 更新为 `ACE4…`，`record_count` 改为 102，或明确写“80 attributed/unmapped + 22 oid-bearing debug rows”；归档前修正，不阻断技术核收。
- 对照：`old-backup-r01/index.json` r03 已把同类问题修正，说明修复方有能力处理，此处属于遗漏而非方向错误。

### R02-N02 [minor/LOW，非阻断] 两套 publication assembler 共存，存在未来分叉与过时注释

- 位置：共享 `article_replay_test.go:397-430`（`s44WireExistingReplay` / `s44ConfigureReplayPublication`）；旧路径 `old_backup_restore_test.go:246-288`（`s44OldBackupConfigurePublication`）。
- 已共享的部分：两者都经唯一迁移入口 `s44ApplyPublicationMigration`，迁移责任不分叉。
- 未共享的部分：Runtime 实现（`s44PublicationRuntime` vs `s44OldBackupRuntime`）、Journal 包装（`s44PublicationJournal` vs `s44OldBackupJournal`）、record hash 复刻（`s44PublicationRecordHash` vs `s44OldBackupRecordHash`）、Publication ID 前缀（`77000000…` vs `66000000…`）、InstallationID 各自一套。
- 过时注释：`old_backup_restore_test.go:248` 仍写“The shared harness intentionally leaves them unconfigured”，与当前 HEAD（共享 harness 已在 `s44WireExistingReplay` 中配置 ports）矛盾，容易误导维护者。
- 影响：当前两套行为等价，4/4 PASS；但未来只改一边（例如给 Runtime 加 `CheckPhase`、改 hash 字段、改 ID 前缀/InstallationID 校验）会造成语义漂移，且测试可能仍绿。
- 建议：旧路径尽量复用 `s44ConfigureReplayPublication`/`s44WireExistingReplay`，只注入不同的 inventory/Journal/Runtime；至少消除过时注释并互相引用。该修改属于作者写集，不由评审代改。

## 专项核验（Lead 裁定与新风险）

### 1. ACK-loss 断言“更严而非放宽” → 判断成立

- 现状：`article_replay_test.go:1222` 断言精确序列 `[ReadCurrent, Begin, Publish, CommitObservedThenAckLost]`；`:1225` 保留 `begins==1 && publishes==1 && evidenceIDs.calls==1 && evidenceClock.calls==1`；`:1228-1232` 禁止出现 `Fail`；`:1233` 要求 evidence 行数为 1。
- 探针为什么不再观察三元组回读：`s44StoreProbe` 改为嵌入具体类型别名 `ReplayEvidenceStore = article.PostgresReplayEvidenceStore`（`article_replay_test.go:966-981`），使生产 `tryReuseAfterUnknown` 依赖的 package-private 回读方法被提升并绕过探针；探针仍显式实现 `Begin`/`ReadCurrent`/`Fail`/`Publish`（`:999-1043`），所以主要写事件仍可观测。
- 为什么没有丢失 commit 可见性验证：`s44StoreProbe.Publish` 在返回 ACK 丢失前，用独立 observer 连接执行 SQL 校验（`:1024-1036`）：`r.status='ready'`、`r.input_digest`/`e.evidence_digest` 与 `e.evidence_ref` 匹配；不可见则返回错误且不记录 `CommitObservedThenAckLost`，序列等值断言必然失败。另有 `s44AckLoss:1205` 要求 `ready == s44ReadyOf(f.storeProbe.committed)`，`s44EvidenceCount(f)==1`（`:1233`）。
- 结论：新断言比旧断言多约束了“无额外 Begin/Publish/Fail/回读事件”，同时 commit 可见性由独立 SQL oracle 覆盖；Lead 的“更严而非放宽”判断成立。观察项：该 SQL oracle 用同一 `evidence.EvidenceDigest` 同时匹配 `input_digest` 与 `evidence_digest`，在本路径两者相等时成立；若未来生产明确区分两者，需要同步调整 oracle（当前为 fail-closed，不会假绿）。

### 2. snapshotHook barrier 死锁/竞态

- 实现：`article_replay_test.go:1281-1305` 的 `snapshot()` 使用局部 `sync.Once`，只让第一次调用阻塞；`entered` 为容量 1 的缓冲通道；两个 `select` 均包含 `<-ctx.Done()` 退出分支。
- 清理：`s44NewBarrier`（`:1244-1248`）注册 `Cleanup(b.open)`；`s44LaunchPublish`（`:1307-1325`）的 Cleanup 顺序为 `barrier.open()` → `cancel()` → 5s 内 join，超时报错。
- 结论：未发现可达的死锁路径；并发调用时第二次 `once.Do` 返回 `held=false` 直接放行，不会重复阻塞。残余证据边界：本轮未对全量 barrier 场景跑 `-race`（仅 ACK-loss leaf 在修复方证据中跑过 `-race` PASS），属于证据覆盖不足，不是代码缺陷。

### 3. s44StoreProbe 嵌入具体类型是否导致“测不到”

- 探针仍显式实现并记录 `Begin`/`ReadCurrent`/`Fail`/`Publish`（`article_replay_test.go:999-1043`），只有 package-private 回读被提升绕过。
- “测不到”风险被三重补偿：独立 observer SQL 校验 commit 可见性（`:1024-1036`）、Ready 与 committed evidence 等值（`:1205`）、evidence 行数唯一（`:1233`）。
- 结论：当前测试变绿不是因为“测不到 Publish/写路径”；真实 INSERT 仍由真实 Store 执行并由 SQL 计数验证。未发现新 blocker。残余风险：若未来生产回读逻辑回归，ACK-loss 用例只能通过 Ready 结果间接发现，无法直接断言回读调用次数；该限制已在 `harness-publication-r01/index.json:157` 的 `known_risks` 中披露。

### 4. NewReplayPublicationPorts 双 Store 各封一次是否符合生产 sealing 语义

- 生产语义：`replay_publication.go:82-105` 在锁内检查 `used || configuration != nil`，重复封存返回 `errReplayPublicationSealed`；即每个 Store 只能封一次。
- harness 行为：`s44WireExistingReplay` 每次调用都 `NewPostgresReplayEvidenceStore` 新建 Store 再封一次（`article_replay_test.go:405-410`）；`replay_evidence_test.go:205-210` 为 peer A/B 各建独立 Store 并分别封存；旧路径 `old_backup_restore_test.go:266-277` 新建 Store 后只封一次，再重绑探针（`:279`）。
- 结论：符合生产 sealing 语义，未发现对同一 Store 二次封存。观察项：`s44New` 已封存过一个 Store，旧路径随后新建第二个 Store 并丢弃第一个，属于重复构造的少量浪费，不是语义违规。

### 5. 两套 assembler 共存是否会在未来分叉

见 R02-N02：当前等价、迁移入口统一；未来存在语义漂移风险，判 minor/LOW，不阻断。

## 四个子测试证明/未证明

| 子测试 | 实际证明 | 未证明 |
|---|---|---|
| `SyntheticManifestIsNotRealBackup`（`old_backup_restore_test.go:48-105`） | fixture 守卫：`synthetic_only`、`production_use=forbidden`、backup artifact `not_included`、entry gate 禁止真实演练；manifest 与 keyring fixture 的 active key 绑定；用 fixture material 计算 HMAC-SHA256 并与 `JournalSecretFingerprint` 比对；active key 状态为 `active`。不连 PostgreSQL。 | 真实旧备份、真实 keyring/HMAC runtime、真实备份 artifact。该子测试是 fixture 自洽检查，不是恢复证明。 |
| `RestoreAndReplayThreeTombstones`（`:107-137`） | 真实 PostgreSQL；三类 tombstone 经真实 `DeletionFinalizer.Finalize` 完成；`ReplayOldBackup` 返回 `0 replayed / 3 already absent` 且无错误；首次 Ready 恰好写 1 行 evidence（`probes=2`）；`s44CheckReady` 校验 Ready 字段、canonical digest、run 绑定；`s44OldBackupRequireCompleted` 校验文章不可见、operation `completed`、audit 成功事件；连续两次 `ReadStatus` 字节一致且业务镜像不变。 | 真实旧备份恢复；真实 POSIX Journal/fsync/flock；真实 HMAC/keyring；真实 inventory 签发/挂载。数据是“恢复后状态”的合成 seed，不是从真实备份恢复出来。 |
| `ReadyDriftInvalidatesEvidence`（`:169-199`） | 真实 PostgreSQL；真实提交的 audit drift 后 `ReadStatus` 返回 `unknown/evidence_stale`；重新 Ready 产生新 evidence ref/digest，evidence 行数为 2；历史行 `xmin/ctid` 保留；republished Ready 两次读取稳定且业务镜像不变。 | 真实 drift 来源/真实备份漂移；真实审计源；真实旧备份。drift 是通过测试内 SQL 变更模拟。 |
| `InventoryUnavailable`（`:201-234`） | 真实 PostgreSQL；缺 inventory 时返回 `article.ErrBackupInventoryUnavailable`；probe/journal 零调用；六表计数不变、完整镜像不变，证明 fail-closed。 | 真实 inventory 签发、挂载、授权、ACL；真实备份；RPO/RTO。 |

## 约束核对

- 未修改共享 harness、生产代码、任务卡 checkbox 或既有证据；本报告是唯一写入。
- 未执行 Git 提交/推送/切换分支；未清理 schema 或证据；未 `DROP`/`TRUNCATE`。
- `old_backup_restore_test.go` 内无 `t.Skip`、无 `DROP`、无 `TRUNCATE`、无 `DELETE FROM`、无硬编码 `postgres://` DSN（`rg` 无匹配）。
- DSN 通过 `HAN_TEST_POSTGRES_DSN` 注入；测试 preflight 日志为 `endpoint=literal_loopback:5433 version=180006 identity_checked=true`。
- 所有测试 schema 日志均为 `retained=true`；`old-backup-r01` 与 `harness-publication-r01` 的 `retained-resources.ndjson` 均保留。
- `tasks.md:1548` 仍为 `[-]`；`tasks.md:1578` 的关闭条件（三类 tombstone、operation/audit 对账、双 backup 并发、输入漂移、双次 ReadStatus 与旧备份恢复通过；缺 inventory/RPO/RTO 时不关闭真实演练；独立 Review 通过）尚未全部满足。

## 证据链自洽

| 文件 | index 记录 | 实测 SHA256 | 结论 |
|---|---|---|---|
| `article_replay_test.go` | `04172354…` | `04172354…` | 一致 |
| `replay_evidence_test.go` | `C23C5C77…` | `C23C5C77…` | 一致 |
| `old_backup_restore_test.go` | `E0C202F2…` | `E0C202F2…` | 一致 |
| `harness-publication-r01/run-00-baseline.log` | `AE38357C…` | `AE38357C…` | 一致（FAIL 基线，`article_replay_test.go:543 type=*pgx.proxyError`） |
| `harness-publication-r01/iter-full-article.log` | `83AA2DF9…` | `83AA2DF9…` | 一致（中间过程，已被 run-03 取代） |
| `harness-publication-r01/run-05.log` | `BA7C1061…` | `BA7C1061…` | 一致（PASS 4/4） |
| `harness-publication-r01/lead-verify-r01.log` | 未登记（任务卡 `BCE990A6…`） | `BCE990A6…` | 文件一致，index 缺登记（R02-N01 附带观察） |
| `harness-publication-r01/retained-resources.ndjson` | `A9833A8A…` / 80 | `ACE4E82D…` / 102 行 | **不一致（R02-N01）** |
| `old-backup-r01/index.json` | `6CA7332B…` | `6CA7332B…` | 一致（r03） |
| `old-backup-r01/run-02.log` | `337DB5B4…` | `337DB5B4…` | 一致；与 `run-01.log` 字节相同 |
| `old-backup-r01/run-01.attempt1.log` | `1F24F876…` | `1F24F876…` | 一致（修复前 FAIL 2/4，`:110/:160`） |
| `old-backup-r01/lead-verify-r01.log` | `DCCDB5AA…` | `DCCDB5AA…` | 一致（PASS 4/4 4.77s） |
| `old-backup-r01/retained-resources.ndjson` | 12 行 | `4953D174…` / 12 行 | 一致 |

补充说明：`harness-publication-r01/index.json` 的 `lines_after` 采用“非空行”口径（`article_replay_test.go` 1990、`replay_evidence_test.go` 651、`old_backup_restore_test.go` 350），与三个文件实测非空行数一致，不是缺陷；任务卡中的 385 行为总行数，两者口径不同但均可核对。

## 未证明边界清单

- 真实旧备份恢复、真实备份 artifact、真实备份完整性与可挂载性。
- 真实 POSIX Journal 的 `fsync`/`flock`、崩溃/断电/磁盘故障韧性。
- 真实 keyring/HMAC runtime；本测试只有合成 HMAC 向量。
- identity 侧双 namespace、`MarkConsumed`、联合开流。
- 真实 inventory 签发、挂载、授权与应用 ACL。
- RPO/RTO/retention。
- 生产 composition root 的 publication ports 装配；`cmd/han-server/main.go:3290` 未调用 `NewReplayPublicationPorts`，归 `identity-auth-S4.20`。
- `EnsureCurrentPublication` / `ImportLegacyPublication` / `CheckPhase` 路径。
- 真实 drift 来源；本测试通过 SQL 变更模拟 audit drift。
- 双 backup 并发与真实第二安装。
- S8.2/Case18、部署后 E2E、真实 Host/进程级恢复。
- 真实 Journal revision/epoch/anchor 与 HMAC 校验；合成 snapshot 只保证结构自洽。
- 新 barrier 的全量 `-race` 覆盖；仅 ACK-loss leaf 有修复方 race 证据。
- `harness-publication-r01/index.json` 中 `article_replay_test.go` 的 `sha256_before` 仅前后缀、`sha256_before_complete:false`，完整修复前 blob 记为 `unknown`；由 `run-00-baseline.log` 行为证据补足。

## 复核命令与实际输出

| 命令 | 实际结果 |
|---|---|
| `go build ./...` | exit 0，无输出 |
| `gofmt -l test/integration/recovery` | 空输出，exit 0 |
| `go vet -tags=integration ./test/integration/recovery` | exit 0，无输出 |
| `HAN_TEST_POSTGRES_DSN=postgres://postgres:<redacted>@127.0.0.1:5433/postgres go test -tags=integration ./test/integration/recovery -run '^(TestArticleReplayPostgres|TestArticleReplayEvidencePostgres|TestOldBackupRestore)$' -count=1 -v` | **exit 0**，`ok han/test/integration/recovery 29.100s`；`TestArticleReplayPostgres` 16/16 leaf PASS；`TestOldBackupRestore` 4/4 PASS；`TestArticleReplayEvidencePostgres` 3/3 PASS |
| `go test -tags=integration ./test/integration/recovery -run '^TestOldBackupRestore$' -count=1 -v`（本轮独立复跑） | PASS 4/4；`--- PASS: TestOldBackupRestore (4.04s)`；新 schema 均 `retained=true` |
| 证据日志复跑 | `run-05.log` PASS 4/4 3.19s；`old-backup-r01/run-02.log` PASS 4/4 5.03s；`old-backup-r01/lead-verify-r01.log` PASS 4/4 4.77s；三次 schema 不同、结果一致 |

关键日志摘录：
- `run-05.log:24` `article_replay_test.go:254 … migration=0011_article_replay_publication.sql stage=applied_in_tx`
- `run-05.log:25/44` `old_backup_restore_test.go:109/171 … stage=already_applied`
- `run-05.log:26/45` `publication_configured … synthetic=true`
- `run-05.log:27` `restore_and_replay … replayed=0 already_absent=3 evidence=1 probes=2`
- `run-05.log:46` `ready_drift … evidence=2 stale=unknown/evidence_stale`
- `run-05.log:63` `inventory_unavailable … probes=0 evidence=0 tombstone_writes=0`
- `run-01.attempt1.log:19/35` `first durable Ready did not create exactly one evidence row`（修复前 FAIL，与修复后 PASS 对照）

## 待 Lead / 人决定

1. **第三路径（旧备份恢复）技术子集：可核收（有界 PASS）**。前提：R02-N01 的证据索引在归档前修正；`minor-01`/`minor-02` 可作为后续清理项，不阻断本次技术核收。
2. **S4.4 整卡：不可关闭**。R01 结论复核成立；至少还需：生产 composition root 装配（`identity-auth-S4.20`）、真实旧备份/Journal/HMAC/keyring/identity/RPO-RTO 证据、双 backup 并发与真实第二安装、人的精确接收与 Lead Gate。
3. **不得越界声称**：本测试与证据不得用于声称已证明真实旧备份恢复、真实 POSIX Journal fsync/flock、真实 keyring/HMAC、identity 双 namespace/MarkConsumed/联合开流、RPO/RTO 或生产恢复路径可用。

---

**2026-10-31 脱敏说明（Lead，按用户确认执行）**：按用户 2026-10-31 确认，本报告中原引用的本机 loopback 合成 DSN 口令已替换为 `<redacted>`（共 1 处）；其余内容、结论与 findings 未变。脱敏前文件 SHA256 为 `7770F3EB2B761C08EED738E0DD1EE348EA439F37425A4A54693D63F974529645`，脱敏后哈希见 tasks.md 同日记录。
