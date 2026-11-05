# identity-auth-S4.4 模拟等级执行证据报告

- 任务：`identity-auth-S4.4 验证 Recovery Journal、keyring 与联合恢复`
- 执行角色：Backend-Dev（模拟等级执行，test scope only）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 报告日期：2026-11-03
- 任务状态：**未关闭，待独立评审；本报告不构成评审通过。**
- 生产结论：**不得据此宣称生产就绪、生产解锁、ACL 已验证或真实环境 PASS。**

## 1. 授权范围与 D09 边界

- `identity-auth-S4.5` 与 `identity-auth-S4.6` 未实施；本次依据 `waived-for-test-scope`（范围豁免，test scope only）执行模拟验证。
- 该标签**不表示** `identity-auth-S4.5` / `identity-auth-S4.6` 已完成，也**不改变** `.add/project/knowledge/tasks/tasks.md:649` 的 requires 原文。
- 生产 Anchor / 生产 provider 不可用；凡依赖生产 Anchor / provider 的断言一律 `NOT RUN`，未使用模拟替身声称通过。
- 独立故障域不取得证据，结论为 `unknown` / 未验证。
- 本任务只新增三个测试文件与这份报告；未修改任何既有 `*.go`、`migrations/`、`cmd/`、`internal/`、`tasks.md`、design/workflow 文档或 ADR。
- 未执行 `git add`、`git commit`、`git stash`、`git restore`、`git checkout`。

## 2. 交付文件

| 文件 | 字节数 | SHA256 |
|---|---:|---|
| `test/integration/recovery/identity_recovery_test.go` | 8959 | `E882B63163F9FA255ABF115F308F82527B56DEAF01FE81B54C8F3DDE142726F0` |
| `test/integration/recovery/journal_faults_test.go` | 9713 | `F8487702E6895D19B6749D9AE446D7DE3DF822E26C48A195C3287D3212943725` |
| `test/integration/recovery/joint_readiness_test.go` | 20692 | `5E8D5B267F1A43B9D0FBF39FC25E44952EDEB318D5CB8DFEC08F487E1F5326D2` |

包名均为 `recovery_test`，均使用 `//go:build integration`，沿用 `fixtures_test.go` / `runtime_fixtures_test.go` 的 DSN 读取、schema 隔离与测试清理约定。

## 3. G0 基线

改前命令：

```text
go test -tags=integration ./test/integration/recovery -count=1
```

真实结果：

```text
ok  han/test/integration/recovery  31.240s
```

外部计时约 33.82s，无既有失败。基线未发现需要修复的既有缺陷或环境故障。环境事实为 PostgreSQL 18.6 监听 5433，`HAN_TEST_POSTGRES_DSN` 已由进程环境提供；本报告与所有命令输出均未回显 DSN 值或任何口令。

## 4. TDD RED 证据

三个测试文件先以骨架和 `t.Fatal("RED: pending implementation")` 运行分组命令：

```text
go test -tags=integration ./test/integration/recovery -run 'TestIdentityRecovery|TestJournalFaults|TestJointReadiness' -count=1 -v
```

真实结果：

```text
FAIL  han/test/integration/recovery  0.086s
```

16 个子测试全部因 `RED: pending implementation` 失败，外部计时约 2.83s。

在补齐实现过程中发现一次测试 helper 缺陷：`s44InjectRecoveryEntry` 注入的 entry 未设置 `RecordHMAC`，导致只修改 `PreviousRecordHash` 的负例先被 HMAC 格式校验拦截并返回 `ErrAdmissionIncomplete`，无法覆盖预期的 `ErrAdmissionBinding` 分支。修复仅限本任务新增的 `joint_readiness_test.go`：在 mutate 后为空的 `RecordHMAC` 写入格式合法的 HMAC；未修改生产代码或既有测试骨架。修复前该子测试真实失败输出为：

```text
first PreviousRecordHash err=recovery admission incomplete; want ErrAdmissionBinding
```

## 5. 必覆场景、用例与 I-REC 映射

### 5.1 崩溃：进程/事务中断后重放与一致性

- `TestJournalFaults/ReplayAfterControlledRestart` — I-REC-10
  - 受控软件 Journal 在 Begin + MarkConsumed 后由新读取者重放，断言 2 条 entry、head=2、阶段严格为 `begun,consumed`、SecretFingerprint 不漂移、链哈希非空。
  - 交叉读取 `article-deletion/v1` 返回 `ErrAdmissionBinding`，fail-closed。
  - 真实进程崩溃 / 事务中断的持久卷与 fsync 证据：`NOT RUN`，原因见第 6 节。
- `TestJournalFaults/MarkConsumedDuplicateAndConcurrent` — I-REC-10
  - 覆盖 MarkConsumed 前后顺序与并发收敛，详见 5.6。

### 5.2 损坏：Journal 记录损坏被检出并拒绝

- `TestJournalFaults/CorruptRecoverySnapshotRejectedByAdmission` — I-REC-04、I-REC-05、I-REC-09
  - 非法 `RecordHMAC` → `ErrAdmissionIncomplete`。
  - 首条 entry 带非空 `PreviousRecordHash` → `ErrAdmissionBinding`。
  - entry 存在但 head 未推进 → `ErrAdmissionIncomplete`。
  - 存在 entry 但无对应 RecoveryRun → `ErrAdmissionIncomplete`。
  - 以上每种情况均断言 runtime 为 `closed` / `proof_unavailable`。
- `TestJointReadiness/CorruptRecoveryEntryFailsClosed` — I-REC-04、I-REC-05
  - 注入非法 `RecordHMAC` 后 runtime 拒绝，状态 `closed` / `proof_unavailable`，错误 `ErrAdmissionIncomplete`。
- 文件 Journal 的部分写、HMAC/链损坏、重复 sequence、只读、ENOSPC、fsync 失败注入 — I-REC-04：`NOT RUN`，原因见第 6 节；Windows 上两个构造器均在触盘前 fail-closed。

### 5.3 跨 key 版本重放被拒

- `TestIdentityRecovery/KeyringRotationFingerprintMatrix` — I-REC-02、I-REC-03
  - `rotated_complete` fixture 的 v1/v2/v3 三个真实指纹与 fixture 期望逐一相等，且三者互不相同（distinct=3）。
  - 8 个负例（无 key、双 active、无 active、空 material、重复 ref、非法 state、空白 ID、控制字符 version）全部返回 `ErrInvalidKeyring`。
- `TestIdentityRecovery/CrossVersionReplayRequiresFileJournal` — I-REC-02
  - 真实持久卷上的 retained v1/v2 跨版本重放：`NOT RUN`。
  - Windows 上 `NewFileRecoveryJournal` 在触盘前返回 `ErrUnsupportedPlatform`，为 fail-closed 证据，不作为跨版本重放通过证据。
- `TestIdentityRecovery/VerifyOnlyKeyRetirementRequiresS46` — I-REC-03
  - 生产 key retirement 属 `identity-auth-S4.6`，未实施：`NOT RUN`。
  - 仅 verify-only key 的 keyring 被拒绝为 `ErrInvalidKeyring`；verified media 构造器返回 `ErrAdmissionUnsupported + ErrUnsupportedPlatform`。

### 5.4 `kept_deleted` 处理

- `TestJointReadiness/KeptDeletedCanonicalReview` — I-REC-01
  - 真实 PostgreSQL schema + 真实 identity/recovery/coordinator/finalizer 链路。
  - 断言 run `outcome=succeeded`、`journal_stage=consumed`、`journal_consumed_at` 非空。
  - 断言目标成员 `status=deleted`。
  - 断言 `recovery_account_reviews.decision=kept_deleted`、`evidence_ref` 非空（本次沿用既有 fixture 的 `s417-recovery-review-evidence`）、review 行数=1。
  - 断言 `credential_reset_at IS NULL`。
  - 断言 `identity.user.delete.canonical` 且 `metadata->>'mode'='recovery_kept_deleted'` 的审计计数=1。
- `TestJointReadiness/KeptDeletedZeroPermitRejected` — I-REC-01
  - 零值 `PhasePermit` 调用被拒绝，未写状态。
  - 断言目标仍 `status=active`、reviews=0、audits=0；`platform.PublicCode(err)=platform.internal_error`。

### 5.5 双 namespace 隔离

- `TestJointReadiness/PositiveGenesisAdmission` — I-REC-08、I-REC-09
  - 正向：`RunNormal` 后 runtime 为 `public_ready` / `verified`；recovery head=0、article head=0、runs=0。
  - recovery journal 读取 `article-deletion/v1` → `ErrAdmissionBinding`。
  - article journal 读取 `recovery/v1` → `ErrAdmissionBinding`。
- `TestJointReadiness/RecoveryNamespaceMismatchFailsClosed` — I-REC-08、I-REC-09
  - recovery snapshot namespace 被改为 article namespace → `ErrAdmissionBinding`，runtime `closed` / `proof_unavailable`。
- `TestJointReadiness/ArticleNamespaceMismatchFailsClosed` — I-REC-08、I-REC-09
  - article snapshot namespace 被改为 recovery namespace → `ErrAdmissionBinding`，runtime `closed` / `proof_unavailable`。
- `TestJointReadiness/ArticleHeadDriftFailsClosed` — I-REC-08、I-REC-10
  - 正向准入后推进 article journal head → `ErrAdmissionDrift`；再次 Verify 仍 `closed` / `proof_unavailable`，未缓存旧的 ready 结论。

### 5.6 `MarkConsumed` 顺序（重复、乱序、并发）

- `TestJournalFaults/MarkConsumedDuplicateAndConcurrent` — I-REC-10
  - 未 Begin 直接 MarkConsumed → `ErrRunNotFound`。
  - 重复 Begin → `ErrRunAlreadyExists`。
  - 重复 MarkConsumed → `ErrJournalMismatch`。
  - 16 个并发 MarkConsumed 对同一 run：恰好 1 个成功、15 个 `ErrJournalMismatch`，无其他错误。
  - consumed 调用计数=19（3 次前序调用 + 16 次并发调用），调用者快照=19。
  - 最终 4 条 entry：两个 run 各恰好一条 begun + 一条 consumed；sequence 从 1 连续，`PreviousRecordHash` 链连续，head sequence=4 且 hash 等于重算值。
- `TestJournalFaults/ReplayAfterControlledRestart` — I-REC-10
  - 重放顺序保持 begun→consumed，链不漂移。

### 5.7 identity keyring 恢复

- `TestIdentityRecovery/SoftwareJournalSecretFingerprintAndStages` — I-REC-01、I-REC-02、I-REC-05
  - `Begin` 的 `SecretFingerprint` 等于 active key 的 HMAC。
  - `RecordHMAC` 绑定链 hash 与 `SecretFingerprint`。
  - 重复 Begin → `ErrRunAlreadyExists`。
  - 正确 secret 的 `VerifyBeginSecret` → `begun`；错误 secret → `ErrAdmissionBinding`；缺失 run → `ErrAdmissionIncomplete`。
  - `MarkConsumed` → `consumed`。
- `TestIdentityRecovery/KeyringRotationFingerprintMatrix` — I-REC-02、I-REC-03
  - 旋转 keyring 三版本指纹与全部负例见 5.3。
- `TestIdentityRecovery/UnsupportedPlatformConstructorsFailClosed` — I-REC-03、I-REC-04
  - `NewFileRecoveryJournal` → `ErrUnsupportedPlatform`。
  - `OpenVerifiedRecoveryJournal` → `ErrAdmissionUnsupported + ErrUnsupportedPlatform`。

### 5.8 联合恢复就绪（recovery 与 article 侧联合判定）

- `TestJointReadiness/PositiveGenesisAdmission` — I-REC-01、I-REC-09
  - 两侧 journal 均通过真实 runtime 联合判定后才得到 `public_ready` / `verified`。
- `TestJointReadiness/RecoveryNamespaceMismatchFailsClosed`、`ArticleNamespaceMismatchFailsClosed`、`ArticleHeadDriftFailsClosed`、`CorruptRecoveryEntryFailsClosed` — I-REC-04、I-REC-05、I-REC-08、I-REC-09、I-REC-10
  - 任一 namespace 混用、article head 漂移或 recovery entry 损坏，联合 readiness 均不开放，runtime 保持 `closed` / `proof_unavailable`。
- `TestJointReadiness/KeptDeletedCanonicalReview`、`KeptDeletedZeroPermitRejected` — I-REC-01
  - recovery 侧 run 终结与 article 侧 canonical 删除证据联合成立时才记录 `kept_deleted`；零 permit 被拒绝且零状态写入。

### 5.9 I-REC-06 / I-REC-07 说明

本次三个新文件的必覆场景来自任务卡关闭条件（崩溃、损坏、跨 key、`kept_deleted`、双 namespace、`MarkConsumed` 顺序、keyring、联合就绪）。I-REC-06（Ready20/Unavailable6 consumer 契约）与 I-REC-07（Ready canonical 前 16 字段/排除字段）未在本报告新增专项断言，也未在本报告中声明通过；它们不属于本次新增三个文件所针对的必覆场景。

## 6. NOT RUN / unknown 清单

| 项目 | 状态 | 原因 |
|---|---|---|
| 真实进程崩溃 / 事务中断后的持久卷重放（I-REC-10） | `NOT RUN` | 当前 Windows 环境无持久卷文件 Journal 后端；受控软件 Journal 不能证明 fsync 或落盘重放。 |
| 文件 Journal 部分写、HMAC/链损坏、重复 sequence、只读、ENOSPC、fsync 失败（I-REC-04） | `NOT RUN` | Windows 上 `NewFileRecoveryJournal` / `OpenVerifiedRecoveryJournal` 在触盘前 fail-closed，无法注入真实文件系统故障。 |
| retained v1/v2 跨 key 版本重放（I-REC-02） | `NOT RUN` | 需要真实持久卷文件 Journal 与备份环境；Windows 构造器仅提供 `ErrUnsupportedPlatform` fail-closed 证据。 |
| 生产 key retirement / verify-only 真实双前置 evidence（I-REC-03） | `NOT RUN` | `identity-auth-S4.6` 未实施；仅验证了 verify-only-only keyring 与构造器的 fail-closed 分类，不声称生产 retirement 通过。 |
| 生产 Anchor / 生产 provider 依赖断言 | `NOT RUN` | 生产 Anchor / provider 不可用；未使用模拟替身声称通过。 |
| 独立故障域证据 | `unknown` / 未验证 | 本次未取得独立故障域证据。 |
| `identity-auth-S4.5` / `identity-auth-S4.6` 完成性 | 未实施 | 依 `waived-for-test-scope` 执行 test scope only 模拟验证，不改变 requires 原文。 |

## 7. 命令与真实结果

| 命令 | 结果 | 耗时 |
|---|---|---|
| `go test -tags=integration ./test/integration/recovery -count=1`（G0 基线） | `ok`，无失败 | 31.240s（外部约 33.82s） |
| 分组 RED（骨架 `t.Fatal`） | `FAIL`，16 子测试全部 `RED: pending implementation` | 0.086s（外部约 2.83s） |
| 分组最终：`go test -tags=integration ./test/integration/recovery -run 'TestIdentityRecovery|TestJournalFaults|TestJointReadiness' -count=1 -v` | `PASS`，16/16 子测试通过 | 包 5.774s（外部 8.1s）；`TestIdentityRecovery` 0.00s、`TestJointReadiness` 4.82s、`TestJournalFaults` 0.87s |
| `go test -tags=integration ./test/integration/recovery -count=1`（整包） | `ok` | 35.809s（外部 38.1s） |
| `gofmt -l .` | **退出码 2**，24 行输出，全部为既有 `.spec/wip/identity-auth/evidence-s10.5/**` 下的不可编译摘录/探针文件；三个新文件及源码树 `cmd internal test` + 根级 `*.go` 均无输出 | — |
| `go vet ./...` | 退出码 0，无输出 | 2.25s |
| `go build ./...` | 退出码 0，无输出 | 3.56s |
| `go test -tags=integration ./test/integration/recovery -race -count=1` | `ok` | 38.674s（外部 45.19s） |

`gofmt -l .` 的全仓输出未达到“零输出”，原因是既有 `.spec/wip/identity-auth/evidence-s10.5/**` 证据目录中存在非 Go 源码摘录/探针文件；按文件所有权约束未修改这些既有文件。三个交付文件单独执行 `gofmt -l` 无输出，源码树（`cmd`、`internal`、`test`、根级 `*.go`）`gofmt -l` 也无输出。

## 8. 测试数据与清理方式

- 使用 `HAN_TEST_POSTGRES_DSN` 连接真实 PostgreSQL 18.6；DSN 为空时按既有约定 `t.Fatal`，不 Skip。
- 每个用例创建 `han_s44_<随机十六进制>` 隔离 schema，通过连接参数 `search_path` 限定到该 schema，`MaxConns=8`，执行真实 migrations。
- 清理仅通过 `t.Cleanup` 关闭连接池；**不执行 `DROP SCHEMA` 或全库清理**，不触碰既有 4,533 个残留 schema。
- 由于本任务文件所有权仅允许新增三个 `*_test.go` 与报告，`s44NewFixture` 有意不写入 S4.17 retained schema registry（写入该 registry 会修改其他任务文件）；schema 保留不删除。
- 测试中未打印 DSN 或口令。

## 9. 证据文件与边界说明

- 验证命令的输出临时保存在 `.spec/wip/identity-auth/_scratch/s44-*.log`（例如 `s44-g0-baseline.log`、`s44-grouped.log`、`s44-full.log`、`s44-vet.log`、`s44-build.log`、`s44-race.log`）。这些是执行日志，不是交付代码，未纳入 git 改动；因环境策略阻止删除而保留。
- 本报告只陈述上述真实执行结果，不把 `NOT RUN`、`unknown` 或模拟替身写成通过。
- 本任务**未关闭**；需要独立评审。本报告及上述测试结果**不构成独立评审通过**，也未声称已通过独立评审。