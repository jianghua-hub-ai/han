# Recovery 合成 fixtures

本目录只包含公开、固定、可复核的**合成测试数据**。它不包含真实备份、真实用户数据、真实 recovery secret、生产路径、生产 provider 配置或生产责任承诺。

`keyring-fixtures.json` 中的 key material 与 `synthetic_recovery_input` 都是刻意公开的测试向量，值中包含 `SYNTHETIC-TEST-ONLY` 标记。它们不是秘密，生产使用被明确禁止。

## 文件与消费合同

### `keyring-fixtures.json`

顶层字段：

| 字段 | 含义 |
| --- | --- |
| `schema_version` | fixture 文件版本，不是生产协议版本 |
| `synthetic_only` | 必须为 `true` |
| `production_use` | 固定为 `forbidden` |
| `material_encoding` | `material` 的编码；当前固定为 `utf8` |
| `fixtures` | 命名 keyring 场景 |

每个 `keys[]` 项直接映射到 `internal/recovery.Key`：

| JSON | Go |
| --- | --- |
| `id` | `Key.ID` |
| `version` | `Key.Version` |
| `state` | `Key.State`，仅 `active` / `verify_only` |
| `material` | `Key.Material`，按 `material_encoding` 解码为字节 |

`rotated_complete` 固定包含一个 active v3 key，以及 verify-only v2、v1 两个历史 key，满足现役 `NewKeyring` 的“恰好一个 active”合同，并覆盖 v1/v2/v3 跨版本 retained-key 判重输入。

`test_vectors[]` 为 active 与每个 retained key 提供同一公开合成输入的固定向量。`synthetic_recovery_input` 只允许在测试中作为 HMAC/`RecoveryJournal.Begin` 输入；`expected_fingerprint` 必须等于：

```text
hmac-sha256:<hex(HMAC-SHA256(referenced key material, synthetic recovery input))>
```

### `backup-manifest.json`

`run` 对象映射到创建初始 `internal/recovery.RecoveryRun` 所需字段：

| JSON | Go |
| --- | --- |
| `run_id` | `RecoveryRun.RunID` |
| `backup_ref` | `RecoveryRun.BackupRef` |
| `journal_key_id` | `RecoveryRun.JournalKeyID` |
| `journal_key_version` | `RecoveryRun.JournalKeyVersion` |
| `journal_secret_fingerprint` | `RecoveryRun.JournalSecretFingerprint` |
| `journal_stage` | `RecoveryRun.JournalStage` |
| `started_at` | `RecoveryRun.StartedAt`，RFC3339 UTC 微秒 |
| `outcome` | `RecoveryRun.Outcome` |

合同核对：

- `backup.backup_ref`、`run.backup_ref` 和未来 article evidence 的 `backup_ref` 必须逐字一致。
- `run.journal_key_id/version` 必须引用 `keyring_fixture` 指定 fixture 中的 active key。
- `run.journal_secret_fingerprint` 必须等于该 fixture 的固定 HMAC 测试向量。
- 初始 run 只允许 `journal_stage=begun` 与 `outcome=running`。
- manifest 只描述合成数据类别；`artifact.status=not_included` 表示仓库没有可恢复的备份文件。
- `operational_unknowns` 逐项保留 identity design r13 Q9/Q10/Q11 的 RPO、RTO、容量、责任、retention、周期、历史时间迁移、evidence 兼容窗口、DeletionOperation/finalizer 容量及 key 退役条件等输入。
- `entry_gate.real_exercise_allowed` 必须为 `false`，且 `unknown_policy=block`。任一必填值为 `unknown` 时不得开始真实演练；不适用必须经责任人确认后显式登记，不能静默省略。
- `anchor_preflight.mode` 必须为 `read_only`；禁止恢复、重置、bootstrap 或回滚既有 Anchor。当前独立 Anchor 缺失或不可判定时必须停止并等待 S4.5 批准流程。
- `cleanup.responsible_owner` 必须保持 `unknown`，直到真实责任人另行登记。不得在本 fixture 中填写生产承诺。

## 固定样本用途

样本只用于验证：

1. 两份文件是有效 JSON；
2. keyring 恰好有一个 active v3 key，且 v2/v1 均为 verify-only；
3. v1/v2/v3 固定 HMAC 测试向量均可复算；
4. manifest 的 `backup_ref`、run、keyring 引用和 UTC 微秒时间一致；
5. Q9/Q10/Q11 operational unknowns 完整且入口 Gate 明确阻断真实演练；
6. maintenance 排在任何状态变更前，Anchor 预检为只读并禁止 restore/reset/bootstrap/rollback。

样本不证明：

- 真实备份可恢复；
- PostgreSQL 恢复成功；
- Journal 持久卷、POSIX 权限/fsync/锁或 CAS Anchor 可用；
- 生产 Anchor provider 或生产 key-retirement 已实现；
- 真实恢复满足任何 RPO/RTO；
- 清理动作已获授权或执行。

## 清理边界

fixture 文件本身是版本库输入，不是清理脚本。真实演练只可清理由本次演练创建、位于隔离测试环境、且经责任人明确确认的资源。当前清理责任为 `unknown / TBD-by-owner`；不得删除真实数据或未经确认的目录。

## 校验

在仓库根目录执行：

```text
Get-Content -Raw testdata/recovery/keyring-fixtures.json | ConvertFrom-Json | Out-Null
Get-Content -Raw testdata/recovery/backup-manifest.json | ConvertFrom-Json | Out-Null
go test -tags=integration ./test/integration/recovery -run "TestRecoveryFixtures"
```

登记的 Go 测试包当前若仍不存在，必须保留命令失败证据；本任务不得创建该测试包。
