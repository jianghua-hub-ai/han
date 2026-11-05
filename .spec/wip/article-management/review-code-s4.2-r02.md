# article-management-S4.2 代码评审 R02（作者自审修订复审）

- 日期：2026-10-26
- Reviewer：主会话 Lead（当前 Host 无法产出非作者 Reviewer）；**依据用户 2026-10-26 明确授权「由作者自审 Code Review（本 Host 可以自审）」**
- objectType：code
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 任务：`article-management-S4.2`
- 上游：R01 `.spec/wip/article-management/review-code-s4.2-r01.md`（SHA256 `777CF2B5029049F86E8B2484DC69C28C11A068A9252F65AB17E8EC97647165C1`，FAIL：3 HIGH + 2 MEDIUM）
- 结论：**PASS**（F001～F005 全部 CLOSED，0 新 HIGH/MEDIUM；F005 的 A-REC-05 真实 PostgreSQL 缺口已在本轮补齐）

> 独立性声明：本次复审由实现作者同一会话执行，**不满足 add-code-review 的「非作者独立复审」要求**。用户已明确接受作者自审口径，故本轮结论在原授权范围内成立；独立性弱于非作者复审，作为已知限制保留。

## 1. 固定对象与漂移核对

复审开始、验证后与报告写入前核对以下 19 个文件 SHA256。**除 `repository_integration_test.go` 外**均与 tasks SSOT 记录的 Lead 返工后值一致。

| 文件 | SHA256（本轮实测） |
|---|---|
| `internal/article/deletion_replay.go` | `1A77C574BD134B59C8203702B022FD1F510BBC23689CE4B0ED8FA4448D34C446` |
| `internal/article/replay_evidence.go` | `54FAB8529DB4275E10FEB65A5ADBBF64B8CB79E82B39B21AFDD7538079774351` |
| `internal/article/replay_status_v1.go` | `244D606F45DC4BC885083F52A3ACCFD065A2F3148ED04B0AECBF7C0F8C62FA16` |
| `internal/article/deletion_replay_test.go` | `2E0BDD1E7B5081364C6412F53269658BF159AE17F39772F3CE992FBE4D7D3194` |
| `internal/article/replay_status_v1_test.go` | `9788B89EA4590F87AA5D99798A670F13A5D416DB1D8CC0DDB8DE65FED7B3E8CD` |
| `internal/article/model.go` | `3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4` |
| `internal/article/deletion_replay_source.go` | `0D53B7FF460FF6923EF6E8B6134B3FB40F9D921EE0498C45AD63F1B77B04492C` |
| `internal/article/deletion_replay_source_test.go` | `DB669095E4C77ACB30FFFBD610BE99BCC8B169C88386D1059349CB5395B66B9A` |
| `internal/article/backup_inventory.go` | `B247DB0CC10723EB1D4464F59A9D543C6421AE296E7AADFF21A440F61C90B7EF` |
| `internal/article/backup_inventory_test.go` | `7260756A356795C913AAB391E8CF19F9C53EB779585D81DEB6C8A20BF6049B12` |
| `internal/article/deletion_replay_integration_test.go` | `F132361840952A2B4111FDC5289807DB993737F72D536C86D01423C21C8EFA0B`（本轮新增 A-REC-05 真实 PostgreSQL 用例后变更；SSOT 记录为 `38FE33F115A136A8D6586CEDE2C898ED927AEC11DF2EAA5CC1318B717EA34B14`） |
| `migrations/0009_article_replay_attempt.sql` | `4E10F50108A57D709C866044DE6678A00E11A4D6ACB589E708C7DC3779A9AA1D` |
| `internal/article/deletion_journal.go` | `0A3F2AE9AEADDDB85C62A1F7CA1BD284AACF149DA0D5ED02184F1E30F0EB4333` |
| `internal/article/deletion_finalizer.go` | `989C51D418F10A46880FE38FBB3069CAAA80D66319708E44A6FED8B4D3AB255F` |
| `internal/article/deletion_test.go` | `91DE34174E2F0E8EA07257CD2EE98605E383DCCC469A5853552A961DAF0015AB` |
| `internal/article/repository.go` | `22551B3B96888A2361A2A55D1C6C650A64494D2198886F85B34BBA8090C3F14C` |
| `internal/article/repository_integration_test.go` | `3A15B7635D97D66AB4DE5CE69155C37E414F75BC3DBB0AD4F14D869019A0DCF4`（SSOT 记录为 `EE9A68CCA141AC6F84C412C6BCE54457C543003ABB2812D4B8C3690D9B269355`，**不一致**） |
| `cmd/han-server/main.go` | `1C9D22526BB84D6E00CE8D3BD1C453CCEB75642C2BD7F2C87CCFF4B3987EAB48` |
| `cmd/han-server/main_test.go` | `CD86C9012928AA80167FD6915D0D36F75C778C8B9DCC313C0F8B094B8904014A` |

**漂移说明**：`repository_integration_test.go` 实际哈希与 tasks SSOT 记录不符（文件含 `TestArticleRepository_ReplayAttemptConstraints`、0009 attempt 约束回归，集成测试实测通过）。此为 SSOT 记录值滞后于文件，需由 Task-Planner/Lead 对账更正，不构成本轮代码缺陷。

## 2. 逐条 finding 复审

### F001（HIGH）— 旧备份重放只有未实现端口与 stub → **CLOSED**

- 修复位置：`internal/article/deletion_replay_source.go`（`ProductionDeletionReplaySource` + `NewProductionDeletionReplaySource`，强制 journal/inventory/replayer/operations/audits/clock 全非 nil）、`internal/article/backup_inventory.go`（`FileBackupInventoryReader`）、`cmd/han-server/main.go:2879-2928`（唯一 composition root `newArticleReplayProvider`）。
- 核对：`ProductionDeletionReplaySource.ReplayOldBackup` 走「读 inventory → 校验 backup 绑定 → `ReadAllVerified` → `ReplayAllTombstones` → 读取 operation/audit 事实 → 计算 digest」，不接受调用方计数/摘要/verified 标记；`NewProductionDeletionReplaySource` 对任一 nil 依赖返回错误。composition root 从必填 `HAN_ARTICLE_BACKUP_INVENTORY_PATH` 显式装配 `NewFileBackupInventoryReader` + `NewPostgresReplayFactReader` + `ProductionDeletionReplaySource` + `NewPostgresReplayEvidenceStore` + `NewDeletionReplayProvider`，缺 journal owner 或路径为空即 fail closed，无 stub 回落。
- 回归：`TestProductionDeletionReplaySource`（真依赖缺失 fail closed、ReplayAll 调用、Snapshot 不重复 ReplayAll、wrong backup fail closed）；`TestArticleReplayProductionWiring`（inventory 路径必填、相对路径拒绝、缺依赖构造失败、唯一 composition 构造成功）。
- 结论：**CLOSED**。

### F002（HIGH）— 旧并发调用可把较新的 Ready 降级为 Failed → **CLOSED**

- 修复位置：`migrations/0009_article_replay_attempt.sql`（前向增加 `attempt_generation`/`attempt_token` + 一致性 CHECK，未改 0001～0008）、`internal/article/replay_evidence.go:252-309`（`Fail`）、`replay_evidence.go:311-405`（`Publish`）。
- 核对：`Fail` 先持 `FOR UPDATE` 读 run，`status == ready` 直接 no-op 返回（Ready 不可降级）；否则要求 `status='running' AND attempt_generation=$4 AND attempt_token=$5` 才执行 `markReplayRunFailedSQL`，`RowsAffected != 1` 返回 `ErrDeletionReplayAttemptNotOwner`。`Publish` 要求 `current.Generation == request.Attempt.Generation && current.Token == request.Attempt.Token`，写入 current 的 `setReplayRunCurrentSQL` 亦带 generation/token 条件。旧 handle 失败无法覆盖较新 Ready。
- 回归（真实 PostgreSQL）：`TestDeletionReplayPostgres/A-REC-06 concurrent same backup converges and stale fail cannot override`（8 并发收敛同一 Ready、旧 generation Fail 为 no-op、Ready/ref/digest/time 不变）；`A-REC-07 join existing running`（candidate-only token 不能 Fail 所有者）。
- 结论：**CLOSED**。

### F003（HIGH）— 时间未规范化，破坏 completed_at 稳定性 → **CLOSED**

- 修复位置：`internal/article/replay_evidence.go:625-627`（`canonicalReplayTime` = `UTC().Truncate(time.Microsecond)`）、`replay_evidence.go:594-595,608,619`（digest 计算、evidence 构造、持久化前规范化）、`replay_status_v1.go:138,156`（响应前规范化）、`deletion_replay_source.go:179,227,259`（inventory cutoff/operation/audit 时间规范化）。
- 核对：digest、evidence 创建、写库与首次响应全部先经 `canonicalReplayTime`，消除亚微秒精度导致的 digest/时间漂移。
- 回归：`TestReplayEvidence_SubMicrosecondTimesNormalizeToStableUTC`（单元）；`A-REC-04 stable ref digest and microsecond time`（真实 PostgreSQL，含 999ns 偏移，断言两次 Ready 全等且 Evidence 行数 = 1）。
- 结论：**CLOSED**。

### F004（MEDIUM）— producer 复制 schema version/namespace authority → **CLOSED**

- 修复位置：`internal/article/model.go:11-16`（`ArticleDeletionNamespace = recovery.ArticleDeletionNamespaceV1`、`DeletionReplayEvidenceVersion = recovery.ReplayEvidenceVersionV1`）。
- 核对：全部非测试 Go 文件中不再存在第二份 `article-deletion/v1` / `article-deletion-replay/v1` schema 字面量（仅 `deletion_journal.go:25` 的 HMAC domain 前缀含不同语义的 `han/article-deletion/v1/tombstone`，非 schema version authority）；producer 直接复用 `internal/recovery` 唯一权威。
- 回归：编译期 alias + 现有 strict decoder 契约测试通过。
- 结论：**CLOSED**。

### F005（MEDIUM）— A-REC-04～07 关键 PostgreSQL 行为缺测试证据 → **CLOSED**

- 修复位置：`internal/article/deletion_replay_integration_test.go`（本卡已登记的第 11 项写集内）。
- A-REC-04～07 真实 PostgreSQL 覆盖（多连接/多事务，`han_test` 隔离 schema，127.0.0.1:5433）：
  - **A-REC-04**（stable ref/digest/microsecond time，含 999ns 偏移）PASS；
  - **A-REC-05**（原子替换 inventory → 当次读取 `unknown/evidence_stale` → Replay 后新增唯一 evidence、current 原子切换、再次读取新 Ready）PASS；
  - **A-REC-06**（8 并发收敛 + 旧 Fail 不能覆盖新 Ready + join running candidate-only 不能 Fail）PASS；
  - **A-REC-07**（Begin commit-unknown 重入同一 candidate、join running 返回权威 handle、Ready-stale 推进 generation 并重新发布新 evidence）PASS。
- 核对：新增 `newInventoryDriftFixture` 实例化**真实 `FileBackupInventoryReader` + 唯一 `ProductionDeletionReplaySource`**（非 stub source），以 `os.Rename` 原子替换 manifest，经 `provider.ReadStatus`→真实 `PostgresReplayEvidenceStore.ReadCurrent` 读取；drift 后返回 `unknown/evidence_stale`，Replay 后 evidence 行数由 1 增至 2 且 current 指向新 evidence。这同时消除了 R01 F005 原有的两处缺口（无真实 `ReadCurrent` 调用、集成层未用真实 source/reader）。
- 实测（本轮重跑，非缓存）：`go test -tags=integration ./internal/article -run '^TestDeletionReplayPostgres$' -count=1` → `ok 4.87s`（全部 6 子用例 PASS）；`-race` → `ok 6.55s`。
- 结论：**CLOSED**。

## 3. 已确认正确的范围

- Ready 使用 `internal/recovery` DTO/严格 decoder，20 字段与固定 digest fixture 正向通过；Unavailable 仅 6 字段。
- SQL 具备 `(run_id,namespace)` 首绑唯一、evidence digest 唯一、append-only、current 外键与 `FOR UPDATE`。
- `ReadCurrent` 对错 backup/未知版本/不可读 evidence/摘要不一致/input 漂移均 fail closed（已由 A-REC-05 真实 PostgreSQL 用例证明）。
- 0009 仅前向增加 generation/token 与 CHECK，未改 0001～0008。
- 五个固定文件 gofmt、定向普通/race、Article 包、vet、全仓 build/vet/contract/race/all 均 PASS。

## 4. 限制与未验证风险

- **独立性**：作者自审（用户已授权）。
- 未恢复真实旧备份、未运行真实 Journal/keyring/inventory/ReplayAll 端到端、未运行 identity 联合恢复（A-REC-08）、真实 Host、部署或 E2E。
- `HAN_TEST_POSTGRES_DSN` 使用 User scope 值（127.0.0.1:5433），仅在本机隔离 schema 内执行；不改动任何外部环境。
- A-REC-05 的真实 inventory 为测试构造的合规 manifest（非真实生产备份 manifest）；真实旧备份/manifest 签发与挂载授权仍属后续部署/E2E，不在本卡范围。
- 工作树含他人既有 unstaged/untracked 内容；本报告不评价其正确性。

## 5. Reviewer 实际命令与结果

```text
git branch --show-current -> main ; git rev-parse HEAD -> 6da8abbceb0fa197364810a92a8fa2d97b38ddca ; git diff --cached --name-only -> (空)
Get-FileHash -Algorithm SHA256 <19 个固定对象> -> 18 匹配，repository_integration_test.go 不匹配（见 §1）
gofmt -l . / go mod verify / go vet ./... / go build ./... -> 全部 PASS
go test ./... / go test -race ./... / go test ./internal/article -count=1 -> 全部 PASS
go test -tags=integration ./internal/article -run '^TestDeletionReplayPostgres$' -count=1       -> ok 4.87s（A-REC-04/05/06/07 全 PASS）
go test -tags=integration -race ./internal/article -run '^TestDeletionReplayPostgres$' -count=1 -> ok 6.55s
go test -tags=integration ./internal/article -run 'Repository' -count=1 -v -> PASS（含 TestArticleRepository_ReplayAttemptConstraints）
检索（修复后）：`deletion_replay_integration_test.go` 内 `A-REC-05` 与真实 source/reader 装配均已就位
```

## 6. Gate

严重度统计：blocker 0 / HIGH 0 / MEDIUM 0 / LOW 0。

**Code Review Gate：PASS**（R01 F001～F005 全部 CLOSED，无新增 HIGH/MEDIUM）。

S4.2 的 task-level 关闭仍需 Lead 现场核 status/diff/哈希/命令/越界。该 PASS 仅覆盖上述 19 文件与 A-REC-04～07 的真实 PostgreSQL 边界；不证明真实旧备份恢复、manifest 真实签发/挂载授权、retention/cleanup 授权、灾备、Host、部署、E2E 或 production readiness。
