# S4.4 第三路径（合成 fixture 旧备份恢复测试）独立评审报告

**对象与修订**
- 评审对象：`test/integration/recovery/old_backup_restore_test.go`
- 固定修订：SHA256 `dff24362ab58af8e2532957c67f2f763674ae282d502e14ca278990bf5f3763c`，394 行（本轮实测一致）
- 证据目录：`.spec/wip/article-management/evidence-s4.4/old-backup-r01/`
- 任务卡：`.add/project/knowledge/tasks/tasks.md:1547`（S4.4 行）、`:1576`（关闭条件）
- 既有核证：`.spec/wip/article-management/verification-s4.4.md`
- 工作区：`D:\AID\VSCode\Codex\Han`，main；只读评审，未执行 Git 写操作，未修改被评审文件、共享 harness、生产代码、任务卡或既有证据
- 评审日期：2026-10-31

## 结论

**CONDITIONAL**

技术子集真实通过：真实 PostgreSQL 18.6、真实 `article.PostgresReplayEvidenceStore`、`article.NewReplayPublicationPorts`、`article.NewDeletionReplayProvider`、`article.NewProductionDeletionReplaySource` 与真实 `article.DeletionFinalizer` 写路径，4/4 PASS。未发现 blocker，也未发现伪造 PASS、`t.Skip`、`DROP`/`TRUNCATE`/`DELETE FROM`、硬编码 DSN、越界清理、改动共享 harness/生产代码/任务卡 checkbox。

但本报告保留 2 条 major 与 4 条 minor：

- **major-01**：测试内直接执行非幂等 `0011`，与正在修改的共享 harness 迁移责任冲突；一旦 harness 把 `0011` 纳入 `migrate()`，本测试会在同一 schema 二次执行而失败。
- **major-02**：测试在测试内重建带 publication ports 的 Store/Provider，绕过了生产装配；当前 `cmd/han-server/main.go:3290-3304` 未调用 `NewReplayPublicationPorts`，该缺口已登记在 `tasks.md:2864`，但本测试 PASS 不能外推为生产可用。
- **minor**：合成 hash 复刻生产未导出函数、合成 Runtime 仅实现部分接口、证据索引的 `run-01`/`run-02` 重复与 `retained_schemas` 遗漏。

因此：**第三路径技术子集可有条件核收；S4.4 整卡不建议关闭。**

## 评审范围与方法

- 第一轮：逐段读取测试全文、四个子测试、生产引用（`internal/article/replay_publication.go`、`internal/article/deletion_journal.go`、`internal/article/replay_evidence.go`、`internal/article/deletion_replay_source.go`）、共享 harness、证据索引与全部日志、任务卡与既有核证。
- 第二轮：针对候选 finding 复核行号、`0011` 幂等性、生产装配调用点、证据 SHA256、attempt1 修复前状态；按允许清单复跑 `go build ./...`、`gofmt -l`、`go vet -tags=integration`、真实 PostgreSQL 测试。
- 有界追踪：Provider→Store→PostgreSQL；Source→Journal/Facts/Inventory；证据 index→日志→ndjson；任务卡→核证文档。未扩展到真实备份、真实 POSIX Journal、identity 联合恢复或部署 E2E。

## 实际命令与输出摘要

| 命令 | 实际结果 |
|---|---|
| `go build ./...` | exit 0，无输出 |
| `gofmt -l test/integration/recovery/old_backup_restore_test.go` | 空输出，exit 0 |
| `go vet -tags=integration ./test/integration/recovery` | exit 0，无输出 |
| `HAN_TEST_POSTGRES_DSN=postgres://postgres:<redacted>@127.0.0.1:5433/postgres go test -tags=integration ./test/integration/recovery -run '^TestOldBackupRestore$' -count=1 -v` | **PASS 4/4**，`--- PASS: TestOldBackupRestore (3.17s)`，`ok han/test/integration/recovery 3.603s`（本轮）；`replay_preflight=passed endpoint=literal_loopback:5433 version=180006 identity_checked=true`；新 schema `han_s44red_67e4501f5ca0e5826b859b64`、`han_s44red_5a544f1315fbec05873f9192`、`han_s44red_ae2910a93e32e9a4545d81b1` 全部 `retained=true` |
| 证据日志复跑 | `run-02.log`：PASS 4/4，5.03s；`lead-verify-r01.log`：PASS 4/4，4.77s；本轮复跑 PASS 4/4，3.17s，三次独立 schema 不同，结果一致 |

关键的 `publication_configured`、`restore_and_replay`、`ready_drift`、`inventory_unavailable` 日志均出现，且 `synthetic=true`、`evidence=1/2`、`probes=2/0` 与断言一致。

## Findings

### blocker

无。

### major

#### major-01 测试内二次执行非幂等 `0011`，与共享 harness 迁移责任冲突（脆弱耦合）

- 位置：`test/integration/recovery/old_backup_restore_test.go:291-302`；`migrations/0011_article_replay_publication.sql:2-6`；共享 harness `test/integration/recovery/article_replay_test.go:239-241`
- 事实：
  - 测试在 `s44OldBackupApplyPublicationMigration` 内读取并直接执行 `migrations/0011_article_replay_publication.sql`。
  - 该迁移不是幂等的：`ALTER TABLE deletion_replay_evidence ADD CONSTRAINT ... UNIQUE`（`0011:2-4`）与 `CREATE TABLE article_replay_publications`（`0011:6`）均没有 `IF NOT EXISTS` 或存在性判断。
  - 当前共享 harness 的 `migrate()` 只迁移到 `0009_article_replay_attempt.sql`（`article_replay_test.go:239-241`），所以当前能通过。
  - 同一目录下另一套 harness 已把 `0011` 纳入迁移清单（`test/integration/recovery/runtime_fixtures_test.go:182-188`），说明“harness 负责迁移 0011”已有先例。另一位实施者正在修改 `article_replay_test.go`/`replay_evidence_test.go` 补 publication ports，若同时把 `0011` 加入 `article_replay_test.go` 的 `migrate()`，本测试会在同一 schema 内二次执行并报 duplicate constraint/table。
- 触发场景：共享 harness 补 publication ports 时顺带迁移 `0011`；或未来任何把 `0011` 加入 `s44Fixture.migrate()` 的改动。
- 影响：测试从“技术通过”变为环境性失败；若为绕开失败而删除测试内迁移，可能出现 publication 表缺失或配置未生效，导致 PASS 语义变化。它也可能诱发两边各自维护迁移清单的长期漂移。
- 建议：明确单一迁移责任。优先让共享 harness 提供“迁移到 0011”的统一入口并让本测试复用；或在测试内先探测 `to_regclass('article_replay_publications')` / `pg_constraint` 是否存在，存在则跳过本次 DDL；不要用静默忽略错误的方式处理重复迁移。该修复需与共享 harness 所有者协调，不应由本评审代改。
- 验证：`migrations/0011_article_replay_publication.sql` 实际内容已核对；`article_replay_test.go:239-241` 当前只到 0009；`runtime_fixtures_test.go:187` 已包含 0011。当前测试 4/4 PASS，但该 finding 是“改变 harness 后必现”的前向风险。

#### major-02 测试内重建 publication ports 绕过生产装配，PASS 不能外推为生产可用

- 位置：`test/integration/recovery/old_backup_restore_test.go:247-289`；生产装配 `cmd/han-server/main.go:3290-3304`；已登记缺口 `tasks.md:2864`；共享 harness `article_replay_test.go:326-360`
- 事实：
  - 测试注释明确说明共享 harness 故意不配置 publication ports，测试内重建 `Source/Store/Provider` 并注入合成 Runtime/Journal/PublicationIDs，才能进入 evidence 写路径（`old_backup_restore_test.go:247-252`）。
  - 生产 `newArticleReplayProvider` 构造 `article.NewPostgresReplayEvidenceStore(b.pool)` 后直接传给 `NewDeletionReplayProvider`（`main.go:3290-3304`），没有调用 `article.NewReplayPublicationPorts`。
  - `tasks.md:2864` 已把“修正 `cmd/han-server/main.go:3290` `newArticleReplayProvider` 自建第二个 `article.NewPostgresReplayEvidenceStore` 的问题”列为后续卡待做。也就是说，这是已知的生产装配缺口，不是本测试文件引入的缺陷。
- 触发场景：读者把本测试 4/4 PASS 理解为“生产旧备份恢复链路已通”；或后续有人以该 PASS 关闭生产恢复门禁。
- 影响：本测试证明的是“组件在 ports 已配置时的行为”，不是“生产 composition root 会配置 ports”。若不在证据口径中显式区分，可能掩盖生产装配缺陷；但它不掩盖组件写路径本身，因为测试确实通过真实 Store/Provider/SQL 断言。
- 建议：保留本测试为 test-only 组件证据；在证据索引/核证文档中明确“production assembly not covered，缺口见 tasks.md:2864”；生产装配修复归后续卡，不在本测试内补。若要在本卡增加保护，可增加一个只读断言或文档检查，确认生产入口最终调用 ports，但不得修改生产代码。
- 验证：`rg -n "NewReplayPublicationPorts" cmd internal --glob '!*_test.go'` 未发现生产调用；`main.go:3290-3304` 实际代码已读；`tasks.md:2864` 已登记该待做项。

### minor

#### minor-01 合成 `s44OldBackupRecordHash` 复刻生产未导出 `deletionRecordHash`，构成脆弱耦合

- 位置：`test/integration/recovery/old_backup_restore_test.go:317-329`、`:361-371`；生产 `internal/article/deletion_journal.go:690-694`、`:730-742`、`:768-777`
- 事实：测试定义了自己的 `s44OldBackupDiskRecord` 与 `s44OldBackupRecordHash`，字段、JSON tag、字段顺序、`DeletedAt.UTC().Format(time.RFC3339Nano)`、`sha256:` 前缀与生产 `deletionDiskRecord`/`deletionMACPayloadFromModel`/`deletionRecordHash` 当前逐项一致，字节等价。测试的 `ReadVerifiedSnapshot` 用该 hash 构造 synthetic prefix，不读取真实 Journal，也不验证真实 HMAC。
- 触发场景：生产 Journal 的 hash 算法、字段、tag、时间格式或前缀变化；测试仍会自洽通过，但 synthetic snapshot 与真实 Journal 已不兼容。
- 影响：本测试无法发现真实 Journal 兼容性回归；它只证明“合成 snapshot 自洽”。当前测试已在文件头声明不证明真实 POSIX Journal/HMAC，因此影响限于测试的防回归能力，不构成当前功能错误。
- 建议：优先让生产暴露 test-only（非导出但同包）hash helper，或增加由真实 Journal 生成的 golden vector；至少增加一条注释/契约测试，明确该复刻必须与 `deletion_journal.go` 同步。严重级别维持 **minor**（当前字节等价，且边界已声明）。
- 验证：逐字段对照 `deletion_journal.go:730-742`、`:768-777` 与测试 `317-329`、`361-371`；测试运行通过。

#### minor-02 合成 Runtime 仅实现 `ReadPermitBinding`，未覆盖 `CheckPhase` 路径

- 位置：`test/integration/recovery/old_backup_restore_test.go:375-394`；生产 `internal/article/replay_publication.go:132-147`、`:655-663`、`:721-762`
- 事实：`s44OldBackupRuntime` 嵌入 nil `recovery.Runtime`，只覆写 `ReadPermitBinding`。当前 `Publish`/`ReadCurrent` 路径经 `checkReplay` 只调用 `ReadPermitBinding`（`replay_publication.go:132-147`，调用点 `:846/:867/:910`），所以本测试不会 nil panic。`EnsureCurrentPublication` 与 `ImportLegacyPublication` 使用 `Runtime.CheckPhase`（`:663`、`:762`），本测试未覆盖；若未来路径调整为先调用 `CheckPhase`，nil 嵌入接口会 panic。
- 触发场景：测试扩展到 ensure/import 路径，或生产把 `CheckPhase` 前移到 Publish/ReadCurrent。
- 影响：当前无功能影响；属于未覆盖边界与未来脆弱点。
- 建议：为 `s44OldBackupRuntime` 实现显式 `CheckPhase`（可返回 `ErrAdmissionBinding` 或记录调用），并加编译期接口断言；或在测试中明确注释“仅覆盖 ReadPermitBinding”。严重级别 **minor**。
- 验证：`replay_publication.go` 的 `checkReplay` 与 `CheckPhase` 调用点已核对；本测试 4/4 PASS。

#### minor-03 证据索引 `run-01.log` 与 `run-02.log` 字节相同，`final_log` 的 path/run 语义不一致

- 位置：`.spec/wip/article-management/evidence-s4.4/old-backup-r01/index.json:8-12`；`run-01.log`；`run-02.log`
- 事实：`run-01.log` 与 `run-02.log` SHA256 均为 `337db5b4f20934c751661b41314959744231cec1ba314a53ea777c04dce11d9c`，均为 7681 字节，LastWriteTime 均为 2026/10/31 16:00:12，内容中的 schema 为 `bf28d454.../718f7412.../4ae4c65b...`（run-02）。`index.json` 的 `final_log.path` 指向 `run-01.log`，但同一对象 `run` 写的是 `run-02`。没有独立的 run-01 运行日志。
- 触发场景：审计者按 index 查找 run-01 原始输出，实际拿到的是 run-02 的副本。
- 影响：证据链可追溯性下降，但不影响技术结论；run-02 与 lead-verify-r01 均为真实 PASS。
- 建议：把 `final_log.path` 改为 `run-02.log`，或将文件明确重命名/标注为 run-02 的副本；如确实保留 run-01 名称，应在 index 中注明“duplicate of run-02, run-01 raw log absent”。严重级别 **minor**。
- 验证：两文件 SHA256/大小/时间戳实测一致；index `final_log` 的 path 与 run 字段实测不一致。

#### minor-04 `retained_schemas` 只列 6 个，`retained-resources.ndjson` 实有 12 个

- 位置：`.spec/wip/article-management/evidence-s4.4/old-backup-r01/index.json:36-67`；`retained-resources.ndjson`
- 事实：index 的 `retained_schemas` 只有 attempt1 3 个 + run-02 3 个，共 6 个；ndjson 有 12 条，额外包含 `lead-rerun-unlogged` 3 个与 `lead-verify-r01` 3 个。`lead_verification.note` 说“created 3 additional retained schemas recorded in retained-resources.ndjson”，该说法只在 ndjson 成立，index 未列；`lead-rerun-unlogged` 3 个在 index 中完全未提。
- 触发场景：审计者只读 index，会漏记 6 个 retained schema。
- 影响：资源留存账不完整；ndjson 仍是权威记录，因此不影响技术结论。
- 建议：将 6 个 schema 补入 `retained_schemas`，或把 `retained_schemas` 改为指向 ndjson 的权威引用，并标注 `lead-rerun-unlogged` 的来源。严重级别 **minor**。
- 验证：index 与 ndjson 逐条对照；ndjson 实际 12 条。

## 四个子测试各自证明与未证明

| 子测试 | 实际证明 | 未证明 |
|---|---|---|
| `SyntheticManifestIsNotRealBackup`（`old_backup_restore_test.go:49-106`） | fixture 守卫：`synthetic_only`、`production_use=forbidden`、backup artifact `not_included`、entry gate 禁止真实演练；manifest 与 keyring fixture 的 active key 绑定；用 fixture material 计算 HMAC-SHA256 并与 `JournalSecretFingerprint` 比对；active key 状态为 `active`。不触 PostgreSQL。 | 真实旧备份、真实 keyring/HMAC runtime、真实备份 artifact。该子测试是 fixture 自洽检查，不是恢复证明。 |
| `RestoreAndReplayThreeTombstones`（`:108-138`） | 真实 PostgreSQL；三类 tombstone 经真实 `DeletionFinalizer` 完成；`ReplayOldBackup` 返回 `0 replayed / 3 already absent` 且无错误；首次 Ready 恰好写 1 行 evidence（`probes=2`）；`s44CheckReady` 校验 Ready 字段、canonical digest、run 绑定（`current_evidence_ref/input_digest/status/backup`）、Facts10（head 3、tombstones 3）、独立 SQL digest/count oracle；连续两次 `ReadStatus` 字节一致且业务镜像不变。 | 真实旧备份恢复；真实 POSIX Journal/fsync/flock；真实 HMAC/keyring；真实 inventory 签发/挂载。数据是“恢复后状态”的合成 seed，不是从真实备份恢复出来的。 |
| `ReadyDriftInvalidatesEvidence`（`:170-200`） | 真实 PostgreSQL；真实提交的 audit drift 后 `ReadStatus` 返回 `unknown/evidence_stale`；重新 Ready 产生新 evidence ref/digest，evidence 行数为 2；历史行 `xmin/ctid` 保留；republished Ready 两次读取稳定且业务镜像不变。 | 真实 drift 来源/真实备份漂移；真实审计源；真实旧备份。drift 是通过测试内 SQL 变更模拟的。 |
| `InventoryUnavailable`（`:202-235`） | 真实 PostgreSQL；缺失 inventory 时返回 `article.ErrBackupInventoryUnavailable`；probe/journal 零调用；六表计数不变、完整镜像不变，证明 fail-closed。 | 真实 inventory 签发、挂载、授权、ACL；真实备份；RPO/RTO。 |

## 约束核对

| 约束 | 结果 | 证据 |
|---|---|---|
| 未改共享 harness | 通过 | 评审期间只读；`article_replay_test.go`/`replay_evidence_test.go` 未由本评审修改 |
| 未改生产代码 | 通过 | 只读；`internal/article` 与 `cmd` 未修改 |
| 未改任务卡 checkbox | 通过 | `tasks.md:1547` 仍为 `[-]`；本评审未编辑 |
| 未 `t.Skip` | 通过 | 测试文件无 `t.Skip` |
| 未 `DROP` schema | 通过 | 无 `DROP`/`TRUNCATE`/`DELETE FROM`；日志均 `retained=true` |
| 未硬编码 DSN | 通过 | 走 `HAN_TEST_POSTGRES_DSN`；harness 强制 `literal_loopback:5433`、`version=180006`（`article_replay_test.go:162-163,228-232`） |
| 未清理既有证据 | 通过 | 证据目录只读；未删除/移动任何文件 |
| 未越界声称 | 通过 | 测试 `:9-13`、`:104-105` 明确列出未证明项；index `synthetic_guards` 同样声明 |

## 证据链核对

| 文件 | 实测 SHA256 | 与 index 一致 | 说明 |
|---|---|---|---|
| `old_backup_restore_test.go` | `dff24362ab58af8e2532957c67f2f763674ae282d502e14ca278990bf5f3763c` | 是 | 394 行 |
| `index.json` | `5d7082416afa722e1c957445e0b76b7f0415474f440dc846ce3aa1b9bdcd9e1c` | — | 索引自身 |
| `run-01.log` | `337db5b4f20934c751661b41314959744231cec1ba314a53ea777c04dce11d9c` | 是 | 与 run-02 字节相同，7681 字节 |
| `run-02.log` | `337db5b4f20934c751661b41314959744231cec1ba314a53ea777c04dce11d9c` | 是 | 真实 PASS 4/4，5.03s |
| `run-01.attempt1.log` | `1f24f8768c03df343208f2cf91a79e165d3d2437d7d60c1625b94b3f1844b0eb` | 是 | FAIL 2/4；失败信息 `old_backup_restore_test.go:110: first durable Ready did not create exactly one evidence row` |
| `lead-verify-r01.log` | `dccdb5aa0261f0774af88b1a7655ee81306e1124af43d877bad77e2b722707e4` | 是 | PASS 4/4，4.77s；3 个新 schema |
| `retained-resources.ndjson` | `4953d17495d9e2b75dde981ac9735c32554649b876daabdbc5395691411e9e78` | 是 | 12 条 schema 记录 |

**attempt1 是否真实对应修复前状态**：是。attempt1 日志中的测试行号为 `93/94/98/110/159/160/190/221`，与当前文件的对应位置不同（当前 `retained_dir` 在 `:109/:171/:203`，evidence 检查已下沉到 harness `s44ProviderReady` `article_replay_test.go:970-979`）；失败信息与 index 的 `note` 一致，且两个 FAIL 子测试正好是 `RestoreAndReplayThreeTombstones` 与 `ReadyDriftInvalidatesEvidence`，与 `FAIL 2/4` 自洽。它不是当前文件的重跑结果。

**Lead 复跑是否可复现**：是。`lead-verify-r01.log` 显示 4/4 PASS、4.77s、三个独立 schema；本轮独立复跑同样 4/4 PASS、3.17s、三个新 schema，关键日志行一致。三次运行 schema 不同但结果一致，说明不是固定 schema 残留导致的假绿。

## 未证明边界清单

本测试明确不证明，评审也确认不能由本测试外推：

- 真实旧备份恢复、真实备份 artifact、真实备份完整性/可挂载性。
- 真实 POSIX Journal 的 `fsync`/`flock`、崩溃/断电/磁盘故障韧性。
- 真实 keyring/HMAC runtime；本测试只有合成 HMAC 向量。
- identity 双 namespace、`MarkConsumed`、联合开流。
- 真实 inventory 签发、挂载、授权与应用 ACL。
- RPO/RTO/retention。
- 生产 composition root 的 publication ports 装配（当前 `main.go:3290-3304` 未调用；缺口见 `tasks.md:2864`）。
- `EnsureCurrentPublication`/`ImportLegacyPublication`/`CheckPhase` 路径。
- 真实 drift 来源；本测试通过 SQL 变更模拟 audit drift。
- S8.2/Case18、部署后 E2E、真实 Host/进程级恢复。
- 真实 Journal 的 revision/epoch/anchor 与 HMAC 校验；合成 snapshot 只保证结构自洽。

## 是否建议 S4.4 可关闭

**不建议关闭 S4.4 整卡。**

理由：
- 任务卡关闭条件（`tasks.md:1576`）要求三类 tombstone、operation/audit 对账、双 backup 并发、输入漂移、双次 `ReadStatus` 与旧备份恢复通过；本测试覆盖前三类 tombstone、operation/audit 对账、双次 `ReadStatus`、模拟输入漂移，但“旧备份恢复”仅为合成 fixture，不是真实旧备份。
- 双 backup 并发不在本测试中；真实 Journal/inventory/RPO/RTO 仍为 `unknown/blocked`（`tasks.md:1575`）。
- 生产 publication ports 装配缺口仍在（`tasks.md:2864`），本测试不能替代生产装配验证。
- 既有核证 `verification-s4.4.md:5,76,94` 仍称第三路径未创建，需由作者/Lead 在接收时更新；本评审不代改。

**可有条件核收的范围**：第三路径的技术子集——真实 PostgreSQL 下的 Store/Provider 写路径、三类 tombstone 的 `ReplayOldBackup` 幂等、operation/audit 对账、双次 `ReadStatus` 稳定、模拟 audit drift 失效与重发布、inventory 缺失 fail-closed。

**核收前条件**：
1. 与共享 harness 所有者协调 `0011` 迁移责任，确保每个 schema 只执行一次；在 harness 加入 `0011` 后重跑仍 4/4 PASS（关闭 major-01）。
2. 修正证据索引：`final_log` 的 path/run 语义、`retained_schemas` 6→12 或改为 ndjson 权威引用（关闭 minor-03/04）。
3. 在证据索引或核证文档中明确“production assembly not covered；缺口见 `tasks.md:2864`”，避免把本测试 PASS 外推为生产可用（约束 major-02 的口径风险）。
4. 保留合成边界声明；真实旧备份、真实 Journal、真实 inventory、RPO/RTO、identity 联合恢复继续由独立任务验证。

**复审条件**：上述 1–3 完成后，按允许命令重跑；若 harness 已迁移 `0011` 且测试仍 4/4 PASS、index 哈希/清单自洽、无新增 blocker，则可将本报告降为 PASS（范围仅限第三路径技术子集）。S4.4 整卡仍需真实旧备份/真实 Journal/inventory/RPO/RTO 与独立 Review 的完整条件。

## 待决定与非阻断建议

**待 Lead/用户决定**：
- 是否接受本测试作为“组件级第三路径证据”，而非生产恢复证据。
- 是否要求先完成 `0011` 迁移协调再合并/核收。
- 是否把 `tasks.md:2864` 的生产装配缺口写入本证据索引的显式边界。

**非阻断建议**：
- 为 `s44OldBackupRuntime` 补 `CheckPhase` 显式实现或编译期断言。
- 为合成 hash 增加“必须与生产 `deletion_journal.go` 同步”的注释/契约测试。
- 在 `s44OldBackupApplyPublicationMigration` 增加存在性探测，避免未来 harness 迁移清单变化导致重复 DDL。
- 在 `run-01.log`/`run-02.log` 的命名上消除歧义，保留真实原始输出。

## 观察项（不作为 blocker）

- 共享 harness 当前 `article_replay_test.go:239-241` 只迁移到 `0009`，`wireExisting`（`:326-360`）未配置 publication ports。按本轮范围澄清，这不作为 old_backup 测试的 blocker；它只说明为何测试内自建配置在当前 HEAD 必要。该状态也使“既有两点证据在当前 HEAD 可复现性”需要另行核对，但不影响本测试实际 4/4 PASS。
- `tasks.md:1555,1575` 与 `verification-s4.4.md:5,76,94` 仍有“old_backup_restore_test.go 尚未创建”的历史表述；接收时应由作者/Lead 更新，本评审不代改。
- `runtime_fixtures_test.go:182-188` 已把 `0011` 纳入另一套 harness 的迁移清单，是 major-01 的直接先例。
- 本轮未运行 Git 命令；未暂存、提交、推送、部署或清理任何 schema/证据。

---

**评审结论重申**：对象真实、测试真实、4/4 可复现；无 blocker；因 `0011` 迁移冲突风险、生产装配绕过和证据索引 bookkeeping，结论为 **CONDITIONAL**。仅第三路径技术子集可有条件核收，S4.4 整卡不建议关闭。

---

**2026-10-31 脱敏说明（Lead，按用户确认执行）**：按用户 2026-10-31 确认，本报告中原引用的本机 loopback 合成 DSN 口令已替换为 `<redacted>`（共 1 处）；其余内容、结论与 findings 未变。脱敏前文件 SHA256 为 `A1BEB1E454E89A92B7D26D429311DF4216CB2C779F714BFFF2D16A0D25BFB00D`，脱敏后哈希见 tasks.md 同日记录。
