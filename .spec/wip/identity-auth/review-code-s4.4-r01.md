# identity-auth-S4.4 独立评审报告（r01）

- 评审角色：add-reviewer（独立评审人，非作者）
- 评审对象：`identity-auth-S4.4 验证 Recovery Journal、keyring 与联合恢复` 的三个新增测试文件及作者证据
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮实测）
- 评审日期：2026-11-03
- 授权边界：只读评审；仅写入本报告；未执行任何 Git 写操作；未修改被评审文件

## 0. 结论摘要

- 结论等级：**PASS WITH CONDITIONS**
- Finding 计数：BLOCKER 0 / HIGH 0 / MEDIUM 3 / LOW 4 / INFO 3，合计 10
- 任务状态：**仍不得关闭**。关闭条件中的「真实备份/持久卷环境覆盖崩溃、损坏、跨 key 版本重放」以及生产 Anchor（S4.5）、生产 key retirement（S4.6）均无证据；作者已如实登记 `NOT RUN`，本评审不改变该事实，也不把它读成通过。
- 一句话判断：测试代码断言具体、测试卫生合格、无越界写入、未削弱既有覆盖；问题集中在**名义覆盖**（跨版本重放与文件 journal 崩溃/损坏子测试在 Windows 仅验证 unsupported-platform fail-closed）与**证据陈述夸大**（一处并不存在的断言），需按第 9 节条件修正后才能作为 S4.4 证据基线。

## 1. 评审对象与修订固定

评审前后逐文件实测，字节数与 SHA256 与任务方给定值完全一致，评审期间对象未变化。

| 文件 | 字节 | SHA256 |
|---|---:|---|
| `test/integration/recovery/identity_recovery_test.go` | 8,959 | `E882B63163F9FA255ABF115F308F82527B56DEAF01FE81B54C8F3DDE142726F0` |
| `test/integration/recovery/journal_faults_test.go` | 9,713 | `F8487702E6895D19B6749D9AE446D7DE3DF822E26C48A195C3287D3212943725` |
| `test/integration/recovery/joint_readiness_test.go` | 20,692 | `5E8D5B267F1A43B9D0FBF39FC25E44952EDEB318D5CB8DFEC08F487E1F5326D2` |
| `.spec/wip/identity-auth/evidence-s4.4/report.md`（作者证据，非代码） | 15,324 | `A1986FDEE6B0669B57546A0A65AF75A1AB9E9EB6CF653EB8629812B6BE9C2D22` |

对象判定：`git status --porcelain -- test/integration/recovery` 返回 `?? test/integration/recovery/`，三文件均为未跟踪新增；HEAD 与任务方声明一致。

## 2. 评审依据（本端实际读取）

- `.codex/skills/add-code-review/SKILL.md`（完整正文）
- `.codex/skills/add-tdd-go/SKILL.md`（含 §1.3 命名要求引用）
- `.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`、`.codex/guidance/coding-standards.md`、`.codex/guidance/task-execution-workflow.md`
- `.add/project/knowledge/tasks/tasks.md:638-665`（S4.4 卡与 S4.5 卡原文）
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:746-755`（I-REC-01～10）及 `:961-964`、`:1011-1012` 映射
- `.add/project/knowledge/decisions/framework.md` ADR-0030 r17 `:1215-1240`、r18 补充 `:1248-1292`、r20 `:1378-1395`，以及 `:1428-1431` 修订记录
- 三份被评审测试文件完整正文；作者证据报告完整正文
- 交叉核验：`test/integration/recovery/runtime_fixtures_test.go`、`runtime_recovery_test.go`、`fixtures_test.go`、`internal/recovery/journal_test.go`、`journal_snapshot_test.go`、`internal/recovery/keyring.go`、`internal/recovery/admission.go`

## 3. 实际执行命令与结果

以下均为本评审人在本轮真实执行，未引用作者或 Lead 的自述结果。

| 命令 | 结果 |
|---|---|
| `git rev-parse HEAD` / `git rev-parse --abbrev-ref HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` / `main` |
| DSN 存在性检查（只输出 PRESENT/MISSING，不回显值） | `DSN_PRESENT` |
| `go test -tags=integration ./test/integration/recovery -count=1` | `ok han/test/integration/recovery 37.580s`，退出码 0 |
| `go test -tags=integration ./test/integration/recovery -run 'TestIdentityRecovery|TestJournalFaults|TestJointReadiness' -count=1 -v` | 16 个子测试全部 `--- PASS`；包 `ok ... 7.203s`，退出码 0；输出 4 条 `NOT RUN` 日志（见 §8） |
| `go test -tags=integration ./test/integration/recovery -race -count=1` | `ok han/test/integration/recovery 36.431s`，退出码 0 |
| `go vet ./...` | 退出码 0，无输出 |
| `go build ./...` | 退出码 0，无输出 |
| `gofmt -l` 三个被评审文件 | 无输出（三文件格式合格） |
| `go test ./internal/recovery -run TestRecoveryJournalReadOnlyContext -count=1 -v` | `--- PASS`，`ok 0.313s`；其中 `RetainedBeginSecretReadOnlyVerifier` 与独立 Python HMAC oracle 在 Windows 实际执行 |
| `go test ./internal/recovery -run 'TestFileRecoveryJournalCorruptionMatrixFailsClosed|TestFileRecoveryJournalRetainedReplayAndMissingKeyFailClosed|TestFileRecoveryJournalBindsBackupAndBeginKeyAcrossRotation' -v` | 三个用例全部 `--- SKIP`（`FileRecoveryJournal is unsupported on Windows`） |
| `Get-FileHash -Algorithm SHA256` / `[IO.File]::ReadAllBytes` | 见 §1 表格，与任务方给定值一致 |
| `git status --porcelain -- test/integration/recovery` | `?? test/integration/recovery/` |
| 目录 mtime 扫描 | 三新文件 `2026-11-03 14:31:52`、`14:31:52`、`14:32:46`；同目录既有文件最晚 `2026-10-31 19:22:52` |
| `.add` 关键文件 mtime / 哈希 | `tasks.md` 912,450 B / `E0341550…1EFB`，mtime `2026-11-02 19:35:13`；`framework.md` mtime `2026-11-03 14:02:39`（均早于 14:31 批次） |
| `.spec/wip/identity-auth/_scratch/s44-*.log` | 存在 `s44-full.log`、`s44-g0-baseline.log`、`s44-grouped.log`、`s44-race.log`（与报告 §9 一致） |

未能独立复现的部分：作者声称的 G0 基线 `ok 31.240s` 与 RED「16 子测试全 FAIL」需要改前状态，现已不存在，本报告按**作者自述、未独立验证**处理。耗时差异（35.809s / 34.409s / 37.580s）属正常波动，不构成发现。

## 4. 第一轮：对象本身的六维检查

### 4.1 断言强度（强）

- 错误分类均为 `errors.Is` 精确断言，不是「无错误」或「计数大于 0」：`ErrInvalidKeyring`（`identity_recovery_test.go:90`）、`ErrRunAlreadyExists`（`:111`）、`ErrAdmissionBinding`（`:119`）、`ErrAdmissionIncomplete`（`:122`）、`ErrRunNotFound`（`journal_faults_test.go:22`）、`ErrJournalMismatch`（`:34`）、`ErrAdmissionDrift`（`joint_readiness_test.go:333`）。
- 并发用例断言精确不变量：`successes == 1 && mismatches == 15`，任何第三种错误直接 `t.Fatalf`（`journal_faults_test.go:51-68`）；并断言调用计数 `3+16=19`、caller 快照 19、4 条 entry、sequence 连续、链哈希闭合（`:69-108`）。这不是「至少 1 成功」。
- 所有负向 runtime 用例都同时断言 `closed` + `proof_unavailable` + 非 nil 错误（`joint_readiness_test.go:254-262` 的 `s44AssertClosedProofUnavailable`），且 `ArticleHeadDriftFailsClosed` 再次 `Verify` 确认未缓存旧 ready 结论（`:336-337`）。

### 4.2 覆盖真实性（部分名义）

- 真实执行且断言到关键行为的：keyring 负例矩阵、软件 journal 指纹/阶段、`MarkConsumed` 顺序与并发、受控重启重放、snapshot 篡改后的 admission 拒绝、双 namespace 隔离与 head 漂移、`kept_deleted` 真实 PostgreSQL 链路、零 permit 拒绝。
- 名义覆盖 1：`CrossVersionReplayRequiresFileJournal`（`identity_recovery_test.go:132-150`）在 Windows 只断言 `ErrUnsupportedPlatform`，在 POSIX 直接记录 NOT RUN 并 `return`；两种分支都不执行「retained key 命中即拒绝」。报告 §5.3 标题「跨 key 版本重放被拒」与 §5.7 把 `KeyringRotationFingerprintMatrix` 映射到 I-REC-02，容易被读成已覆盖（详见 F001）。
- 名义覆盖 2：`FileJournalCrashAndCorruptionFailFast`（`journal_faults_test.go:207-223`）在 Windows 只断言两个构造器错误分类，在 POSIX 直接记录 NOT RUN 并 `return`；没有任何部分写、HMAC/链损坏、只读、ENOSPC、fsync 注入（详见 F002）。
- 「崩溃」降级为「受控软件 journal 重启重放」，作者在 `journal_faults_test.go:147` 与报告 §6 明确写为 NOT RUN，**未写成通过**，该降级诚实。

### 4.3 契约与错误分类（强）

错误分类与按钮语义对齐，且区分了 `ErrAdmissionIncomplete`（结构/引用不完整）与 `ErrAdmissionBinding`（绑定/命名空间不符）与 `ErrAdmissionDrift`（已准入后 head 漂移），不是一律 `closed` 了事。`s44TamperedJournal` 只读包装，不伪造通过（`joint_readiness_test.go:67-85`）。

### 4.4 测试卫生（合格）

- 无 `t.Skip`；DSN 缺失走 `t.Fatal("... required for S4.4; no Skip")`（`joint_readiness_test.go:29-32`）。
- 不回显 DSN/口令；`pgxpool.ParseConfig` 失败只报通用错误（`:35-39`）。
- schema 为 `han_s44_<hex>`（`:34`），无 `DROP`/全库清理；`t.Cleanup` 只关闭连接池（`:56`、`:22-26`）。
- 无 `t.Parallel`；并发用例自带 `WaitGroup` 与 `failures` channel，无共享 schema 竞争。
- 未关闭连接：`fixture.close` 释放 pool 与 ctx cancel；未发现遗留 goroutine 路径。

### 4.5 可观测性与可诊断性（合格）

每个子测试都有 `t.Logf` 输出关键状态与分类；NOT RUN 有显式 `t.Log` 原因；失败信息包含实际值与期望值。日志未包含密钥、DSN、secret 原文。

### 4.6 交付完整性（合格）

新增仅三文件（`git status` 为目录级未跟踪）；未改 `internal/`、`cmd/`、`migrations/`、既有 `*.go`、`tasks.md` 或设计文档；`tasks.md` 哈希与 ADR r19/r20 记录一致。作者证据 §2/§8 与实测一致。

## 5. 第二轮：消费者与上游依据的有界检查

### 5.1 任务卡（`tasks.md:638-655`）

关闭条件原文要求「真实备份/持久卷环境中覆盖崩溃、损坏、跨 key 版本重放、kept_deleted、双 namespace 与 MarkConsumed 顺序；生产 Anchor 须来自已完成的 S4.5，生产 key retirement 仅在 S4.6 …」。本轮测试覆盖了 `kept_deleted`、双 namespace、`MarkConsumed` 顺序，但前四项中的「跨 key 版本重放」与「文件 journal 崩溃/损坏」在本机无执行证据；S4.5/S4.6 均未实施。结论：**任务不能关闭**，与作者报告一致。

### 5.2 `requires` 口径（`tasks.md:649`）

`requires` 原文包含 S4.5、S4.6。`waived-for-test-scope` 在 ADR-0030 r17 `:1216`、r18 `:1260`/`:1280`、r20 `:1387` 有正式登记，定义句明确「不改变 requires 文本、不使 S4.5/S4.6 变为已实施、不构成安全豁免、不得读作 requires 已满足」。作者报告 §1 与 §6 的引用口径**正确**，未宣称 S4.5/S4.6 完成；未发现该口径被误读。

### 5.3 test-plan I-REC 映射（部分夸大）

- 达到的：I-REC-09（双 namespace）；I-REC-10 的「MarkConsumed 前后顺序」；I-REC-08 的 namespace/head 漂移分支；I-REC-05 的部分绑定分支。
- 未达到但被映射的：I-REC-01（永久密码/Session/临时凭据失效、非 initial 默认不授权）没有任何断言，`s417BootstrapIdentity` 只是夹具步骤；I-REC-02（同 run 同 secret 跨 v1/v2/v3 重放、不同 secret 只用 active key 写新 begun）未执行；I-REC-03（缺历史 key、未知 key、只迁 Journal、只迁 active key）只覆盖了 verify-only-only keyring 构造拒绝；I-REC-04 只覆盖了「篡改已验 snapshot 后 admission 二次校验」，未覆盖文件系统故障注入。
- I-REC-06/07 作者已在 §5.9 明确排除，未声称通过，处理正确。

### 5.4 ADR-0030 边界（遵守）

未出现「生产就绪」「生产解锁」「ACL 已验证」「真实环境 PASS」；模拟替身只用于 fail-closed/状态机验证；独立故障域登记为 `unknown`。符合 r17 决策 3、r18 补充段与 r20 门禁边界。

### 5.5 与既有测试的关系（未削弱，部分重复）

- 未修改、未替换、未削弱既有 8 个测试文件；既有 `TestRecoveryFixtures`、`runtime_admission_test.go` 五子测试、`runtime_recovery_test.go` 的 H04 kept_deleted 链路仍在。
- 重复：`KeptDeletedCanonicalReview`（`joint_readiness_test.go:363-425`）与 `runtime_recovery_test.go:541-605` 基本同构；新增价值是 `journal_stage=consumed`、`journal_consumed_at` 非空、`credential_reset_at IS NULL`、review 行数=1。`KeptDeletedZeroPermitRejected`（`:427-449`）与 `runtime_recovery_test.go:519-525` 同构。
- 既有生产指纹 oracle 在 `internal/recovery/journal_snapshot_test.go` 用独立 Python HMAC 常量校验 `fingerprintSecret`，且在 Windows 实际执行（本轮 PASS）；因此新矩阵自算 HMAC 属于重复而非新盲区。

### 5.6 生产代码边界（未越界）

被评审文件只调用 `internal/recovery`、`internal/identity`、`internal/article`、`internal/platform` 的公开 API；未修改生产代码；未新增依赖。

## 6. Findings

### [MEDIUM] 跨 key 版本重放（I-REC-02）在可执行路径上没有任何断言

- finding ID：`IA-S4.4-R01-F001`
- 对象/修订：`identity_recovery_test.go` `E882B631…26F0`；作者证据 `A1986FDE…2D22`
- 位置：`test/integration/recovery/identity_recovery_test.go:132-150`、`:41-95`；`journal_faults_test.go:112-148`；报告 §5.3（`:92-102`）、§5.7（`:144-151`）
- 现状：`CrossVersionReplayRequiresFileJournal` 在 Windows 只断言 `NewFileRecoveryJournal` 返回 `ErrUnsupportedPlatform`；在 POSIX 分支直接 `t.Log("NOT RUN…")` 后 `return`。两种分支都不构造「v1 记录 + 仅 v2 active」的 keyring，也不断言 `ErrSecretReused`/`ErrJournalCorrupt`。`KeyringRotationFingerprintMatrix` 只比较测试辅助 `syntheticFingerprint` 与 fixture 期望值，`NewKeyring` 只断言「无错」。
- 触发：在 Windows 运行 S4.4 全套，或在 POSIX 运行同一子测试（提前 return）；均不会执行重放拒绝路径。
- 影响：任务关闭条件明确列出「跨 key 版本重放」，但当前 16/16 PASS 不含该项实质执行；若把报告标题/映射当作已覆盖，会高估 S4.4 证据。全仓唯一的重放拒绝断言在 `internal/recovery/journal_test.go:496-520`，该用例被 `requirePOSIXJournal` 在 Windows `t.Skip`（本轮实测三用例 SKIP）。
- 建议：二选一。(a) 在真实持久卷/POSIX 环境执行并保留证据；(b) 将子测试重命名为 `FileJournalConstructorsFailClosedOnUnsupportedPlatform`，正文与报告明确「本子测试只证明 unsupported-platform fail-closed，不证明跨版本重放」，并把 I-REC-02 从 §5.3/§5.7 的已覆盖映射中移除。
- 验证：POSIX 上运行 `go test ./internal/recovery -run TestFileRecoveryJournalRetainedReplayAndMissingKeyFailClosed -v` 应 PASS；或在 S4.4 增加真实文件 journal 用例断言 `ErrSecretReused`。Windows 上重命名后 `-v` 输出不再出现「跨版本重放」语义。

### [MEDIUM] 文件 journal 崩溃/损坏（I-REC-04）在 S4.4 无执行，子测试名称构成名义覆盖

- finding ID：`IA-S4.4-R01-F002`
- 对象/修订：`journal_faults_test.go` `F8487702…43725`；作者证据 `A1986FDE…2D22`
- 位置：`test/integration/recovery/journal_faults_test.go:207-223`；报告 §5.2（`:90`）、§7（`:187`）
- 现状：`FileJournalCrashAndCorruptionFailFast` 在 Windows 仅断言 `NewFileRecoveryJournal=ErrUnsupportedPlatform`、`OpenVerifiedRecoveryJournal=ErrAdmissionUnsupported+ErrUnsupportedPlatform`；POSIX 分支记录 NOT RUN 后 `return`。没有任何部分写、HMAC/链损坏、重复 sequence、只读、ENOSPC、fsync 注入。报告 §7 仍把它计入「16/16 子测试通过」。
- 触发：任何平台运行该子测试；Windows 只走构造器分类，POSIX 直接返回。
- 影响：任务关闭条件要求「损坏」与「崩溃」；当前证据只证明平台不支持时 fail-closed。既有 `internal/recovery/journal_test.go:537-587` 的损坏矩阵（partial write/hmac/chain/sequence/namespace）与 `:804-925` 的同步失败矩阵在 Windows 全部 SKIP（本轮实测），因此本机无任何文件 journal 故障注入执行。只读/ENOSPC 在既有矩阵中也未见覆盖。
- 建议：同 F001 二选一；至少把子测试重命名为 `FileJournalConstructorsFailClosedOnUnsupportedPlatform`，并把「16/16 子测试通过」改述为「12 项实质执行 + 4 项部分执行（含 NOT RUN）」，避免读者把 PASS 计数当作崩溃/损坏已验。
- 验证：`go test -tags=integration ./test/integration/recovery -run TestJournalFaults -v` 中该子测试的输出应只包含构造器分类；重命名后语义与输出一致。

### [MEDIUM] 报告声称 `platform.PublicCode(err)=platform.internal_error` 被断言，实际只记录日志

- finding ID：`IA-S4.4-R01-F003`
- 对象/修订：作者证据 `A1986FDE…2D22`；`joint_readiness_test.go` `5E8D5B26…26D2`
- 位置：报告 §5.4（`:113-115`）；`test/integration/recovery/joint_readiness_test.go:448`
- 现状：报告写「断言目标仍 `status=active`、reviews=0、audits=0；`platform.PublicCode(err)=platform.internal_error`」。代码在 `:445-447` 只断言 `status/reviews/audits`，`:448` 是 `t.Logf("… code=%v …", platform.PublicCode(err), …)`，没有比较 `platform.internal_error`。
- 触发：按报告复核该用例时，检查代码会发现缺少该断言；不会导致测试误通过（拒绝本身由 `err != nil` 与零写入断言），但证据陈述与代码不符。
- 影响：证据可信度；若下游把「已断言错误码」写进验收记录，会形成无法从测试复现的结论。该问题不改变运行时行为，也不构成伪造 PASS。
- 建议：把报告改为「记录 `platform.PublicCode(err)`（未断言具体码）」；或补 `if platform.PublicCode(err) != platform.internal_error { t.Fatalf(...) }` 后再保留原句。
- 验证：修改后重新读取 `joint_readiness_test.go:427-449` 与报告 §5.4 逐字对照。

### [LOW] I-REC 映射夸大：I-REC-01/I-REC-03 没有对应的可执行断言

- finding ID：`IA-S4.4-R01-F004`
- 对象/修订：作者证据 `A1986FDE…2D22`
- 位置：报告 §5.4（`:106`、`:113`）、§5.7（`:142-154`）、§5.8（`:158-163`）
- 现状：`SoftwareJournalSecretFingerprintAndStages` 被映射到 I-REC-01/I-REC-05，但它只验证软件 journal 的指纹与阶段；`KeptDeletedCanonicalReview` 被映射到 I-REC-01，但未断言「全部永久密码/Session/current 临时凭据失效；非 initial 账号默认不授权」。`KeyringRotationFingerprintMatrix` 被映射到 I-REC-03，但「缺历史 key、未知 key、只迁 Journal、只迁 active key」没有断言，只有 verify-only-only keyring 构造拒绝。
- 触发：以 I-REC-01/03 验收条件逐项核对断言时。
- 影响：映射表会让读者认为 I-REC-01/03 已有自动化证据，实际只有相邻行为被覆盖。
- 建议：把映射收窄到实际断言项（例如 I-REC-05 的绑定分支、I-REC-09 的双 namespace），I-REC-01/03 标注「未覆盖/NOT RUN」。
- 验证：逐条把映射 ID 对应到测试中的 `if`/`errors.Is` 断言；无对应断言者删除映射。

### [LOW] 报告称「真实 identity/recovery/coordinator/finalizer 链路」未标明 journal 为软件替身

- finding ID：`IA-S4.4-R01-F005`
- 对象/修订：作者证据 `A1986FDE…2D22`；`joint_readiness_test.go` `5E8D5B26…26D2`
- 位置：报告 §5.4（`:107`）；`test/integration/recovery/joint_readiness_test.go:193`、`:225`、`:230-244`
- 现状：`KeptDeletedCanonicalReview` 使用真实 PostgreSQL schema、真实 `identity.NewRecoveryRunStore`、真实 `recovery.NewRuntime`、真实 coordinator/finalizer，但 recovery journal 与 article deletion journal 都是内存 `softwareJournal`。报告 §1/§6 已声明模拟等级，§5.4 该句未标注 journal 替身。
- 触发：读者只看 §5.4 时，会把「联合恢复」理解为包含持久化 journal 的真实联合。
- 影响：证据边界表述不精确；不影响代码行为。
- 建议：在该句补「recovery/article journal 为 `softwareJournal` 测试替身；持久卷联合恢复为 NOT RUN」。
- 验证：对照 `joint_readiness_test.go:193`/`:225` 的 `s417NewSoftwareJournal` 与 `s417NewSoftwareDeletionJournal`。

### [LOW] 顶层测试函数命名不符合 `Test<Function>_<Scenario>_<Expected>`

- finding ID：`IA-S4.4-R01-F006`
- 对象/修订：三份测试文件
- 位置：`identity_recovery_test.go:40`、`journal_faults_test.go:16`、`joint_readiness_test.go:264`
- 现状：顶层函数为 `TestIdentityRecovery`、`TestJournalFaults`、`TestJointReadiness`，只有两段；`add-tdd-go` 引用 `coding-standards §1.3` 要求三段式。
- 触发：按 Go 测试命名规范检查。
- 影响：可读性与规范一致性；子测试名清晰，实际诊断价值未受损。既有 `runtime_admission_test.go`/`runtime_recovery_test.go` 同样使用非三段式顶层名，属仓库既有风格。
- 建议：保持现状并在评审记录说明沿用既有风格，或改为三段式；不建议为此扩大改动面。
- 验证：`gofmt`/`go test` 不受影响；命名检查以规范文本为准。

### [LOW] `kept_deleted` 两个新用例与既有 H04 用例高度重复

- finding ID：`IA-S4.4-R01-F007`
- 对象/修订：`joint_readiness_test.go` `5E8D5B26…26D2`
- 位置：`test/integration/recovery/joint_readiness_test.go:363-425`、`:427-449`；对照 `test/integration/recovery/runtime_recovery_test.go:519-525`、`:541-605`
- 现状：既有 H04 已覆盖 `runtime.Enter` + 真实 `s417BootstrapIdentity` + `kept_deleted` + canonical audit=1，以及零 permit 拒绝；新用例重复该链路，新增断言仅为 `journal_stage=consumed`、`journal_consumed_at` 非空、`credential_reset_at IS NULL`、review 行数=1。
- 触发：对比既有覆盖时。
- 影响：增加约 60 行与一次约 5 秒的真实 PG 执行，边际价值有限；但不削弱既有覆盖，也不构成错误。
- 建议：保留（新断言有独立价值），在报告 §5.4 注明与既有 H04 的差异，避免被当作新增覆盖。
- 验证：对比两处断言行即可确认差异集合。

### [INFO] schema 永不删除且不登记 retained registry

- finding ID：`IA-S4.4-R01-F008`
- 位置：`test/integration/recovery/joint_readiness_test.go:22-26`、`:51-59`；报告 §8（`:198-201`）
- 现状：`s44NewFixture` 有意不写 S4.17 registry（文件所有权原因），schema 只关闭连接池、不 DROP；报告称保留 4,533 个残留 schema。
- 影响：长期资源占用与清理成本；当前符合「只改本任务文件」与「不清理他人 schema」约束，不是缺陷。
- 建议：由后续知识整理/运维任务统一清理，不在本卡处理。
- 验证：`SELECT count(*) FROM information_schema.schemata WHERE schema_name LIKE 'han_s44_%'`（本轮未执行写/删）。

### [INFO] 「16/16 子测试通过」包含 4 个提前 return 的 NOT RUN 部分用例

- finding ID：`IA-S4.4-R01-F009`
- 位置：报告 §7（`:187`）
- 现状：16 个子测试中，`CrossVersionReplayRequiresFileJournal`、`VerifyOnlyKeyRetirementRequiresS46`、`UnsupportedPlatformConstructorsFailClosed`（POSIX 分支）、`FileJournalCrashAndCorruptionFailFast` 均可能提前 return；Windows 下前两者与 `FileJournalCrashAndCorruptionFailFast` 只执行部分断言，`ReplayAfterControlledRestart` 也附带 NOT RUN 日志。Go 层面均记 PASS。
- 影响：计数本身不假，但与 §6 NOT RUN 清单并列时容易被读成「16 项全部实质验证」。
- 建议：改述为「12 项实质执行 + 4 项部分执行（详见 §6）」。
- 验证：`-v` 输出中 4 条 `NOT RUN` 日志与本条一一对应。

### [INFO] `waived-for-test-scope` 引用口径正确，无生产结论越界

- finding ID：`IA-S4.4-R01-F010`
- 位置：报告 §1（`:12-19`）、§6（`:169-179`）、§9（`:204-208`）
- 现状：报告明确 S4.5/S4.6 未实施、waiver 不改 `tasks.md:649` requires、不使 S4.5/S4.6 完成；全文未出现「生产就绪」「生产解锁」「ACL 已验证」「真实环境 PASS」；独立故障域为 `unknown`。
- 影响：正面结论，无缺陷。
- 建议：保持；后续引用时继续同时引用 `framework.md:1280` 定义句与 `:1272` 取代表行。
- 验证：全文关键词扫描（本轮已执行 `Select-String`）。

## 7. 复审对应关系

本轮为首次评审（r01），无原 finding 需要逐条复核。F001/F002/F003 为条件项，修订后需按 §9 的复审条件复核；F004/F005 为证据表述修正，随同一修订复核。

## 8. NOT RUN / unknown 边界（本评审确认）

| 项目 | 状态 | 本评审判断 |
|---|---|---|
| 真实进程崩溃 / 事务中断后的持久卷重放 | NOT RUN | 作者已如实登记；受控软件 journal 重启不能替代 |
| 文件 journal 部分写、HMAC/链损坏、重复 sequence、只读、ENOSPC、fsync 失败 | NOT RUN | 作者已如实登记；S4.4 无执行，既有 POSIX 单测在 Windows SKIP |
| retained v1/v2 跨 key 版本重放 | NOT RUN | 作者已如实登记；报告标题/映射仍易误读（F001） |
| 生产 key retirement / verify-only 真实双前置 evidence（S4.6） | NOT RUN | 作者已如实登记；未宣称完成 |
| 生产 Anchor / 生产 provider（S4.5） | NOT RUN | 作者已如实登记；未用替身声称通过 |
| 独立故障域证据 | unknown | 作者已如实登记；本评审同样未取得 |
| 作者自述的 G0 基线 31.240s、RED 16 FAIL | 未独立验证 | 改前状态已不存在；不作为本轮 PASS 依据 |

## 9. 结论、条件与复审条件

### 9.1 结论

**PASS WITH CONDITIONS**。测试代码本身可接受：断言精确、并发不变量正确、fail-closed 分类清晰、测试卫生合格、无越界写入、未削弱既有覆盖、无 BLOCKER/HIGH。条件项均为**证据表述与名义覆盖**修正，不涉及生产代码或测试正确性。

### 9.2 条件（须在把本报告作为 S4.4 证据基线前完成）

1. 修正报告 §5.4：删除或改为「记录」`platform.PublicCode(err)`，不得声称已断言（F003）。
2. 重命名或明确限定 `CrossVersionReplayRequiresFileJournal` 与 `FileJournalCrashAndCorruptionFailFast` 的实际范围，使名称/报告不再暗示已执行跨版本重放、崩溃与损坏注入（F001/F002）。
3. 把「16/16 子测试通过」改述为「12 项实质执行 + 4 项部分执行（含 NOT RUN）」（F009）。
4. 收窄 I-REC 映射，I-REC-01/02/03/04 未覆盖项标为 NOT RUN；不得把 keyring 指纹矩阵读作 I-REC-02 已覆盖（F001/F004）。
5. 报告 §5.4 标明 recovery/article journal 为 `softwareJournal` 替身（F005）。

### 9.3 任务关闭

本评审**不构成任务关闭**。`tasks.md:638-655` 的关闭条件仍缺：真实持久卷环境的崩溃/损坏/跨 key 版本重放证据、S4.5 生产 Anchor、S4.6 生产 key retirement 与独立故障域。上述任一未取得前，任务保持未关闭；不得把本报告的 PASS WITH CONDITIONS 读作生产就绪、生产解锁、ACL 已验证或真实环境 PASS。

### 9.4 复审条件

- 提交修订后的作者证据报告（新哈希）与三份测试文件（若改子测试名）；若仅改报告，三份测试文件哈希应保持不变。
- 复核 F001/F002/F003 的逐条修正；F004/F005/F009 随同一修订核对。
- 若在 POSIX/持久卷环境补跑文件 journal 用例，需附命令、退出码与关键输出；未补跑则保持 NOT RUN。
- 修订后重新执行 `go test -tags=integration ./test/integration/recovery -count=1`，并记录实测耗时与结果。

## 10. 声明

- 本报告仅写入 `.spec/wip/identity-auth/review-code-s4.4-r01.md`。
- 未修改任何被评审文件，未执行任何 Git 写操作（无 `add`/`commit`/`stash`/`restore`/`checkout`/`branch`）。
- 本报告中的命令与结果均为本轮真实执行；作者自述的 G0/RED 已标注未独立验证。
- 本报告是独立评审结论，不代替 Lead/用户对结果的接受、任务关闭授权或后续生产授权。