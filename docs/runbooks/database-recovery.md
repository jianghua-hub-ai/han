# 数据库恢复运行手册

> 状态：**仅用于合成 fixture 的恢复顺序与交接准备，不是生产可执行或生产就绪证明。**

本文只登记 `identity-auth-S4.3` 的恢复输入合同。仓库不包含真实备份、真实恢复凭据、生产 Journal 路径、生产 CAS Anchor provider 配置或生产 key-retirement 能力。

## 合成输入

| 输入 | 位置 | 用途 |
| --- | --- | --- |
| backup manifest | `testdata/recovery/backup-manifest.json` | 固定合成 `backup_ref`、初始 RecoveryRun、数据类别、未知运营值和清理责任 |
| keyring fixtures | `testdata/recovery/keyring-fixtures.json` | 固定一个 active key 与两个 verify-only 历史 key，并提供 v1/v2/v3 可复核合成 HMAC 向量 |
| fixture 说明 | `testdata/recovery/README.md` | JSON 字段与现役 Go 合同的映射、限制和机械校验方式 |

所有值都必须保持明显合成。fixture 中的 key material 和 synthetic input 是公开测试向量，不是秘密，也不得复制到任何部署。

## 当前阻塞与 unknown

| 项目 | 当前值 | 必须由谁在真实演练前决定 |
| --- | --- | --- |
| RPO | `unknown` | `TBD-by-owner` |
| RTO | `unknown` | `TBD-by-owner` |
| 恢复批量容量 | `unknown` | `TBD-by-owner` |
| 恢复运行责任 | `unknown` | `TBD-by-owner` |
| 逐账号核对责任 | `unknown` | `TBD-by-owner` |
| Recovery Journal / 恢复记录 retention | `unknown` | `TBD-by-owner` |
| key retention / archive exit evidence | `unknown` | `TBD-by-owner` |
| article tombstone / inventory retention | `unknown` | `TBD-by-owner` |
| Q10 周期 | `unknown` | `TBD-by-owner` |
| 历史非规范时间迁移方案 | `unknown` | `TBD-by-owner` |
| evidence 版本兼容窗口 | `unknown` | `TBD-by-owner` |
| DeletionOperation 容量 | `unknown` | `TBD-by-owner` |
| finalizer 容量 | `unknown` | `TBD-by-owner` |
| key 退役条件 | `unknown` | `TBD-by-owner` |
| 清理责任 | `unknown` | `TBD-by-owner` |
| 生产 Journal 持久卷与路径 | `unknown` | `TBD-by-owner` |
| 生产 CAS Anchor provider/config/wiring | `unknown`；未实现 | `identity-auth-S4.5` 后续范围 |
| CAS Anchor 独立故障域验证 | `unknown`；未完成 | `identity-auth-S4.5` 后续范围 |
| 生产 key-retirement capability 闭环 | `unknown`；未实现且必须保持禁用 | `identity-auth-S4.6` 后续范围 |
| 目标 POSIX 文件系统证明 | `unknown`；未完成真实验证 | `TBD-by-owner` |
| 未来 Windows 允许 SID 清单 | `unknown` | `TBD-by-owner` |

上述任一必填值为 `unknown` 时不得开始真实演练，不得从 fixture 推断或自行设默认。某项确实不适用时，必须由对应责任人确认后显式登记 `not_applicable` 及依据；不得静默省略。缺少恢复授权、责任或一致性证据时保持停服。

## 不变量与安全 Gate

1. 真实演练入口先执行 unknown Gate；通过后立即进入 maintenance。maintenance 必须先于任何会改变 Journal、keyring、Anchor、数据库或 RecoveryRun 状态的动作，恢复成功前不得接收业务流量。
2. `run_id` 每次恢复唯一；`backup_ref` 在 `Begin` 后不可变。
3. Recovery Journal、identity-owned RecoveryRunStore 与 article evidence 必须分别读取并精确匹配同一 `run_id + backup_ref`。
4. maintenance 后先只读预检**当前** Recovery Journal、完整 keyring 与**当前独立 Anchor**；不得在预检中写入、替换或恢复这些材料。keyring 必须恰好一个 active key，并保留恢复所需的全部 verify-only 历史 key；不完整时 fail closed。
5. recovery secret 只从受控运行环境注入；仅持久化带 `key_id/key_version` 的 HMAC 指纹，明文不得写入本文、manifest、Journal、数据库、日志或审计。
6. `MarkConsumed` 必须沿用 `Begin` 绑定的 key ID/version/fingerprint 与 `backup_ref`；轮换 active key 不得改变既有 run 的绑定。
7. 所有 Journal、Anchor 与 RecoveryRun 持久时间使用 UTC 微秒规范值。
8. 生产 `FileRecoveryJournal` 只允许在已验证本地 POSIX 语义、owner-only 边界、原子替换、文件/目录 fsync 和 OS 锁均成立时启用。当前仓库证据不满足该生产 Gate。
9. 禁止恢复、重置、bootstrap 或回滚既有 Anchor。当前独立 Anchor 缺失或不可判定时保持停服，等待 `identity-auth-S4.5` 定义并批准的处理流程；不得用 Journal 快照或备份中的 Anchor 替代当前 Anchor。
10. key retirement 在生产 verifier、retention/archive exit evidence 和单次消费闭环完成前保持禁用。

## 恢复步骤

以下是合同顺序，不是可直接复制执行的生产命令。

1. **执行入口 Gate（只读、无状态变更）**：登记本次恢复授权、运行责任、逐账号核对责任、清理责任，以及 Q9/Q10/Q11 的全部运营输入。任一必填项仍为 `unknown` 时不得开始真实演练；不适用项必须已有责任人确认和显式依据。
2. **进入 maintenance**：阻止业务流量，确认恢复步骤具有独占运行门禁。此步骤必须早于任何状态变更。
3. **确定不可变备份引用**：在受控系统中选择真实备份并生成唯一、不可变的 `backup_ref`。不要修改仓库中的合成 manifest 来承载真实值。
4. **只读预检当前信任材料**：只读检查当前 Recovery Journal、完整 keyring 与当前独立 CAS Anchor，并验证它们可解释且彼此一致。不得恢复、替换或改写 Journal/keyring；尤其禁止恢复、重置、bootstrap 或回滚既有 Anchor。Anchor 缺失或不可判定时保持停服，等待 S4.5 批准流程。
5. **创建 Journal begun 记录**：以唯一 `run_id`、步骤 3 的 `backup_ref` 和受控环境注入的 recovery secret 调用 `Begin`。保存返回的 key ID/version、HMAC 指纹和 UTC 微秒时间；不保存 secret 明文。
6. **恢复 PostgreSQL**：从步骤 3 绑定的同一备份恢复数据库。恢复工具、位置和命令由真实环境责任人提供；本文不登记生产路径或 provider。
7. **创建 identity RecoveryRun**：使用 begun 记录返回的 `run_id`、`backup_ref`、key ID/version、指纹、`stage=begun`、`outcome=running` 和相同 started time 创建权威 RecoveryRun。
8. **执行 identity 恢复终结**：失效全部 Session、永久密码、未 consumed/未 superseded 临时凭据；除初始 `admin` 外账号默认进入 `suspended + recovery_required`。初始 `admin` 只允许使用独立一次性 recovery secret，并在首次登录强制改密。
9. **逐账号核对**：由已登记责任人核对角色、状态与删除事实。无法重建的事实不得授权；`kept_deleted` 必须幂等重放完整 canonical 删除事务。
10. **执行 article deletion replay**：article 侧必须基于同一 `run_id + backup_ref` 生成并持久化独立 evidence。缺失、漂移或 unavailable 时停止。
11. **推进恢复状态机**：按 `identity_ready → article_deletion_ready → ready_to_consume` 短事务推进，每步重新读取 Store、Journal 和 article evidence，不跨 PostgreSQL 长事务执行文件或外部 I/O。
12. **消费 Journal**：只有三方对账完成后才调用 `MarkConsumed(run_id)`。若重启时 Journal 已有精确匹配的 consumed 记录而数据库仍为 `ready_to_consume`，完整验链与 Anchor 后只补写数据库 `journal_consumed`，不得再次消费。
13. **最终对账并开放**：再次核对 RecoveryRun、Journal、Anchor 与 article evidence，推进到 `succeeded` 后才允许 readiness。任何不一致均保持停服。
14. **清理合成/临时资源**：只清理由本次演练创建且已被清理责任人明确确认的隔离资源；禁止清理真实生产数据或未经确认的目录。保留清理结果与失败证据。

## 失败处理

| 失败 | 行为 |
| --- | --- |
| run、backup、key、fingerprint、stage 或时间不一致 | fail closed；不得改写 fixture 或历史记录来“对齐” |
| 历史 key 缺失或 keyring 不完整 | fail closed；不得跳过全 retained-key 判重 |
| Journal/keyring 不完整、损坏或不可解释 | fail closed；不得从备份替换当前信任材料来绕过预检 |
| 当前独立 Anchor 缺失、回退或不可判定 | fail closed；禁止 restore/reset/bootstrap/rollback，等待 S4.5 批准流程 |
| article evidence 缺失、Unavailable 或发生 ref/digest 漂移 | fail closed；不得 `MarkConsumed` |
| `MarkConsumed` 后数据库转换失败 | 保留现场；重启后只允许执行已批准的精确收敛路径 |
| Q9/Q10/Q11 任一必填运营值仍 unknown | 不得开始真实演练；不适用项必须显式确认并登记依据 |

## 机械校验

在仓库根目录执行：

```text
Get-Content -Raw testdata/recovery/keyring-fixtures.json | ConvertFrom-Json | Out-Null
Get-Content -Raw testdata/recovery/backup-manifest.json | ConvertFrom-Json | Out-Null
go test -tags=integration ./test/integration/recovery -run "TestRecoveryFixtures"
```

若 `test/integration/recovery` 不存在，最后一条命令必须记录真实失败，不能写成 PASS，也不得由本任务越权创建测试包。
