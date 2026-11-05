# identity-auth-S4.4 独立评审报告（r02 复审）

- 评审角色：add-reviewer（独立评审人，非作者）
- 评审对象：`identity-auth-S4.4 验证 Recovery Journal、keyring 与联合恢复` 的 r02 返工
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮实测）
- 评审日期：2026-11-03
- 授权边界：只读评审；仅写入本报告；未执行任何 Git 写操作；未修改被评审文件
- r01 基线：`.spec/wip/identity-auth/review-code-s4.4-r01.md`，28,084 B / `BF926AAA70E9C4523EA04D7D1DEE20B2665DB8829D8A0CB21DB3CA9B5B276163`，本轮核验保持原样
- r01 作者证据：`.spec/wip/identity-auth/evidence-s4.4/report.md`，15,324 B / `A1986FDEE6B0669B57546A0A65AF75A1AB9E9EB6CF653EB8629812B6BE9C2D22`，本轮核验保持原样

## 0. 结论摘要

- 结论等级：**CHANGES REQUIRED**
- Finding 计数：BLOCKER 0 / HIGH 0 / MEDIUM 1 / LOW 2 / INFO 0，合计 3
- 一句话判断：r02 对 F001～F003 的代码修正、F004～F006 与 F008～F010 的口径修正基本到位，测试仍可重复通过；但 F007 的返工把一条**并非完全重复**的正向 `kept_deleted` 用例删除，连带丢失四项既有用例没有的断言，形成覆盖回退，需恢复或等价补回后才能把本轮作为 S4.4 证据基线。
- 任务状态：**仍不得关闭**。真实持久卷下的崩溃/损坏/跨 key 版本重放、S4.5 生产 Anchor、S4.6 生产 key retirement 与独立故障域仍无证据；本轮 PASS 只是测试执行结果，不是生产结论，也不代表任务验收完成。

## 1. 评审对象与修订固定

评审前后逐文件实测，r02 三份测试与作者 r02 证据的字节数、SHA256 均与任务方给定值一致；r01 证据与 r01 评审报告未被覆盖。

| 文件 | 字节 | SHA256 | mtime（实测） |
|---|---:|---|---|
| `test/integration/recovery/identity_recovery_test.go` | 9,346 | `0742ECA24D00884703133EB625B9F3E2FC060706CB6E64AC0F22DA21D730696D` | 2026-11-03 14:47:06 |
| `test/integration/recovery/journal_faults_test.go` | 9,791 | `7049DF0B1B843E9342EC12902AA9DAF7755C99EE8C21DFBBC3A2D79620DA54B4` | 2026-11-03 14:45:03 |
| `test/integration/recovery/joint_readiness_test.go` | 17,647 | `B3D0E3364C94A353630E3227989867D26176AC41BC557D7C2951E5FFCC99B562` | 2026-11-03 14:45:17 |
| `.spec/wip/identity-auth/evidence-s4.4/report-r02.md`（作者证据） | 26,950 | `D3D18A13B6FF67772A05AD35DF0AC6EEFEFC040AECB4BB3D7E0F15CA8487EA6D` | 2026-11-03 14:51:18 |
| `.spec/wip/identity-auth/evidence-s4.4/report.md`（r01 证据） | 15,324 | `A1986FDEE6B0669B57546A0A65AF75A1AB9E9EB6CF653EB8629812B6BE9C2D22` | 2026-11-03 14:35:04 |
| `.spec/wip/identity-auth/review-code-s4.4-r01.md`（r01 报告） | 28,084 | `BF926AAA70E9C4523EA04D7D1DEE20B2665DB8829D8A0CB21DB3CA9B5B276163` | 2026-11-03 14:43:16 |

对象判定：`git status --porcelain -- test/integration/recovery` 返回 `?? test/integration/recovery/`；三个测试文件仍为未跟踪新增。r02 目录扫描显示同目录既有文件最晚为 `2026-10-31 19:22:52`，只有本批次三个文件在 2026-11-03 14:45～14:47 被写入。

## 2. 评审依据（本端实际读取）

- `.codex/skills/add-code-review/SKILL.md`（完整正文）
- `.codex/skills/add-tdd-go/SKILL.md`（含 `Test<Function>_<Scenario>_<Expected>` 三段式要求）
- `.codex/guidance/code-review-checklist.md`、`.codex/guidance/coding-standards.md`
- `.add/project/knowledge/tasks/tasks.md:638-655`（S4.4 卡、requires 与关闭条件原文）
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:746-755`（I-REC-01～10）
- `.add/project/knowledge/decisions/framework.md` ADR-0030 r17 决策 3（`:1227-1233`）、r18 补充段（`:1278-1284`）、r20 门禁边界（`:1387`）
- r02 三份测试文件与 `report-r02.md` 完整正文
- 交叉核验：`test/integration/recovery/runtime_recovery_test.go:519-605`、`internal/identity/recovery_execution_postgres_test.go`、`internal/identity/recovery_run_store.go`、`internal/recovery/admission.go`、`internal/recovery/journal_test.go`
- r01 报告全文与 r01 作者证据相关段落

## 3. 实际执行命令与结果

以下均为本轮真实执行，未把作者或 Lead 的自述结果当作本评审的独立证据。

| 命令 | 结果 |
|---|---|
| `git rev-parse HEAD` / `git branch --show-current` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` / `main` |
| DSN 存在性检查（只输出 PRESENT/MISSING） | `DSN_PRESENT` |
| `go test -tags=integration ./test/integration/recovery -run 'TestIdentityRecovery|TestJournalFaults|TestJointReadiness' -count=1 -v` | 15 个子测试全部 `--- PASS`；包 `ok han/test/integration/recovery 5.581s`，退出码 0；输出 5 条 NOT RUN/部分执行日志 |
| `go test -tags=integration ./test/integration/recovery -count=1` | `ok han/test/integration/recovery 44.546s`，退出码 0 |
| `go test -tags=integration ./test/integration/recovery -race -count=1` | `ok han/test/integration/recovery 40.324s`，退出码 0 |
| `go vet ./...` | 退出码 0，无输出 |
| `go build ./...` | 退出码 0，无输出 |
| `gofmt -l test/integration/recovery` | 无输出，退出码 0 |
| WSL 环境核验：`wsl.exe -- bash -lc "uname -a; command -v go; go version"` | Linux WSL2 内核可见，但 `go: command not found`，退出码 127；确认 POSIX 侧无 Linux Go 工具链 |
| `rg` 扫描三文件中的 `t.Skip`、`t.Parallel`、`DROP SCHEMA`、`DROP DATABASE`、DSN 回显 | 未发现 `t.Skip`、`t.Parallel`、DROP 或 DSN 值回显；DSN 缺失仅命中 `t.Fatal("... no Skip")` |
| `git status --porcelain --untracked-files=all -- test/integration/recovery ...` | 三个测试文件与两份证据/r01 报告均为未跟踪；未显示这些文件的跟踪改动 |
| `git diff --stat` / `git diff --cached --stat` | 工作树有 18 个**既存**跟踪文件改动，暂存区为空；与本批次文件无关。`tasks.md` mtime 为 2026-11-02 19:35:13，`framework.md` mtime 为 2026-11-03 14:02:39，均早于本批次 14:45 |
| 目录扫描：`internal/`、`cmd/`、`migrations/`、`test/`、`.add/project/knowledge/`、`.codex/` 中 mtime 晚于 2026-11-03 14:40 的文件 | 只有 `test/integration/recovery/identity_recovery_test.go`、`journal_faults_test.go`、`joint_readiness_test.go` 三个文件 |

分组 `-v` 的实际子测试与 NOT RUN 输出：

- Identity 5 项：`KeyringRotationFingerprintMatrix`、`SoftwareJournalSecretFingerprintAndStages`、`CrossVersionReplay_PersistentVolume_NotRun`、`VerifyOnlyKeyRetirement_S46_NotRun`、`UnsupportedPlatformConstructors_WindowsOnly_FailClosed` 全部 PASS；后 3 项含 Windows-only 构造器级 fail-closed 或 POSIX NOT RUN 说明。
- Joint 6 项：`PositiveGenesisAdmission`、`RecoveryNamespaceMismatchFailsClosed`、`ArticleNamespaceMismatchFailsClosed`、`ArticleHeadDriftFailsClosed`、`CorruptRecoveryEntryFailsClosed`、`KeptDeletedZeroPermit_NoStateWritten` 全部 PASS。
- Journal 4 项：`MarkConsumedDuplicateAndConcurrent`、`ReplayAfterControlledRestart_SoftwareJournalOnly`、`CorruptRecoverySnapshotRejectedByAdmission`、`FileJournalCrashCorruption_PersistentVolume_NotRun` 全部 PASS。
- `ReplayAfterControlledRestart_SoftwareJournalOnly` 另输出 `NOT RUN: real process crash / transaction interruption durability ...`；这属于受控软件 journal 的证据边界，不是真实崩溃通过。

未能独立复现的部分：作者 r02 报告中的 G0 基线、RED 失败输出属于改前/历史状态，本轮未重测，也不作为本轮结论依据。耗时差异（作者 `34.581s` / Lead `33.422s` / 本评审 `44.546s`）属于环境波动，不构成 finding。

## 4. 第一轮：r02 对象六维检查

### 4.1 范围与越界

- 只新增/修改三个授权测试文件与 `report-r02.md`；未发现对既有 `*.go`、`internal/`、`cmd/`、`migrations/`、`tasks.md` 或设计文档的本批次写入。
- r01 证据与 r01 报告哈希保持原样，未被 r02 覆盖。
- 既有 8 个 recovery 集成测试文件未被修改；本批次没有把新用例写成替换、删除或弱化既有测试。

### 4.2 行为与断言强度

- 错误分类仍以 `errors.Is` 精确断言：`ErrInvalidKeyring`、`ErrRunAlreadyExists`、`ErrAdmissionBinding`、`ErrAdmissionIncomplete`、`ErrRunNotFound`、`ErrJournalMismatch`、`ErrAdmissionDrift` 等均有对应断言。
- 并发用例仍断言精确不变量：16 个并发 `MarkConsumed` 恰好 `successes=1`、`mismatches=15`，并把第三种错误送入 `failures` channel 直接失败；同时断言 19 次调用、4 条 entry、sequence 连续与链闭合。不是「至少 1 成功」。
- 所有 runtime fail-closed 用例仍同时断言 `closed` + `proof_unavailable` + 非 nil 错误；`ArticleHeadDriftFailsClosed` 再次 `Verify` 确认未缓存旧 ready 结论。
- F003 新增的 `platform.PublicCode(err) != platform.CodePlatformInternalError` 是真实比较，不是日志；分组运行输出证明该断言执行并通过。

### 4.3 安全、资源与测试卫生

- 无 `t.Skip`；DSN 缺失走 `t.Fatal` 且错误文案不回显 DSN/口令。
- schema 为 `han_s44_<随机十六进制>`，通过 `search_path` 隔离；没有 `DROP SCHEMA`、`DROP DATABASE` 或全库清理。
- `t.Cleanup` 只关闭连接池与取消 context；未发现遗留连接或 `t.Parallel` 造成的共享 schema 竞争。
- 并发用例自建 `WaitGroup` 与有缓冲 `failures` channel，未发现共享可变状态未被同步保护的问题。

### 4.4 配置与可观测性

- 未新增硬编码凭据；测试替身仅用于软件 journal/端口，报告与代码注释已标明。
- 每个子测试有可诊断的 `t.Logf`；NOT RUN 有明确原因；未打印 secret、DSN 或口令。
- 报告把「Go PASS」与「底层行为已验证」分开，F009 的计数口径已修正。

## 5. 第二轮：消费者与上游依据的有界检查

### 5.1 任务卡与 requires

`tasks.md:649` 的 requires 原文仍包含 `identity-auth-S4.5`、`identity-auth-S4.6`；r02 报告 §1/§11 明确 `waived-for-test-scope` 不改变 requires、不使 S4.5/S4.6 完成、不构成安全豁免。与 ADR-0030 r18 `:1280` 的定义句一致，未发现把 waiver 读成 requires 已满足。

`tasks.md:652` 的关闭条件仍要求真实备份/持久卷环境覆盖崩溃、损坏、跨 key 版本重放、`kept_deleted`、双 namespace 与 `MarkConsumed` 顺序，并要求生产 Anchor 来自 S4.5、生产 key retirement 仅在 S4.6 双前置 evidence 与 capability 原子单次消费成功后开放。r02 仍只取得软件 journal、Windows 构造器级 fail-closed 与本机模拟证据；任务不能关闭。

### 5.2 test-plan I-REC 映射

r02 §6 的映射比 r01 明显收窄：

- I-REC-01、I-REC-06、I-REC-07 已明确标为未断言/未运行；不再把 keyring 指纹矩阵读作 I-REC-02。
- I-REC-02、I-REC-03、I-REC-04、I-REC-05、I-REC-08、I-REC-09、I-REC-10 均标为「部分断言」，并列出缺口。
- I-REC-04 明确区分内存 snapshot 篡改与文件系统故障注入；I-REC-02 明确区分软件 journal 指纹与真实跨版本重放。

残余问题见 R02-F002：I-REC-08 行写「不 MarkConsumed 只在零 permit 分支断言」，但 `joint_readiness_test.go` 没有出现 `MarkConsumed`、`consumedMarkCount` 或 `StageConsumed` 的断言；零 permit 用例只断言 `status/reviews/audits` 零写入，不能证明「不 MarkConsumed」。

### 5.3 既有测试与 F007 的覆盖关系

`runtime_recovery_test.go:519-525` 只断言零 permit 调用返回错误，并记录 `platform.PublicCode`；`:541-605` 的真实 H04 用例断言：

- `liveRun.Outcome == succeeded`；
- `liveStatus == deleted`；
- `liveDecision == kept_deleted`；
- `liveEvidence == s417-recovery-review-evidence`；
- canonical delete audit 恰好 1 条。

它**没有**断言：

- `recovery_runs.journal_stage == consumed`；
- `journal_consumed_at` 非空；
- `credential_reset_at IS NULL`；
- `recovery_account_reviews` 行数恰好为 1（`:585-588` 使用 `QueryRow`，读取到第一行即可通过，不能证明没有重复行）。

因此 r01 F007 所指的「重复正向流程」在整体链路上确实重复，但它附带的四项断言是独立的、未被既有 H04 覆盖的。r02 删除该用例后，这四项覆盖消失；r02 §2 称「唯一删除项是重复正向流程」、§4.7 称「不削弱既有覆盖」，不能精确描述这一覆盖回退。

### 5.4 ADR 边界与生产结论

- 未出现把模拟结果写成「生产就绪」「生产解锁」「ACL 已验证」「真实环境 PASS」的正面断言；这些词只出现在禁止性声明中。
- 生产 Anchor/provider、S4.5、S4.6、独立故障域均登记为 `NOT RUN`/`unknown`，未用替身声称通过。
- 符合 ADR-0030 r17 决策 3、r18 补充段与 r20 门禁边界；本报告不替代 Lead/用户的任务关闭或生产授权。

## 6. Findings

### [MEDIUM] F007 返工删除的四项断言并非重复，既有 H04 用例没有等价覆盖

- finding ID：`IA-S4.4-R02-F001`
- 对象/修订：`joint_readiness_test.go` `B3D0E336…B562`；`runtime_recovery_test.go`（既有文件，仅交叉核验）；`report-r02.md` `D3D18A13…EA6D`
- 位置：`test/integration/recovery/joint_readiness_test.go:366-392`；`test/integration/recovery/runtime_recovery_test.go:581-604`；`report-r02.md` §2、§4.7、§6 I-REC-08 行
- 现状：r02 删除了 `KeptDeletedCanonicalReview` 正向流程，只保留 `KeptDeletedZeroPermit_NoStateWritten`。被删用例原先断言 `journal_stage=consumed`、`journal_consumed_at` 非空、`credential_reset_at IS NULL`、review 行数=1；既有 H04 用例只断言 `outcome=succeeded`、`status=deleted`、`decision=kept_deleted`、`evidence_ref`、canonical audit=1。`runtime_recovery_test.go:585-588` 的 `QueryRow` 也不校验 review 行数。
- 触发：任何以 r02 测试套件作为 S4.4 `kept_deleted` 证据的复核；或在 PostgreSQL 中制造重复 review 行、错误 journal 阶段、非空 `credential_reset_at` 后运行该套件。
- 影响：`kept_deleted` 的 journal 消费状态、消费时间、凭据重置保持和 review 基数四项不变量失去自动化覆盖；任务关闭条件中的 `kept_deleted` 证据被削弱。该问题不改变运行时代码，但属于测试覆盖回退，不能由「既有 H04 已覆盖」替代。
- 建议：在授权范围内恢复 `KeptDeletedCanonicalReview` 正向子测试，保留上述四项断言；如不希望恢复整段，可在 `joint_readiness_test.go` 增加等价断言（读取 `recovery_runs` 的 `journal_stage`/`journal_consumed_at`、读取 `credential_reset_at`、用 `count(*)` 断言 review 行数=1）。不要修改既有 `runtime_recovery_test.go` 来扩大所有权。
- 验证：恢复后分组 `-v` 应重新出现该子测试并通过；使用一条重复 review 行或非 consumed journal 阶段的负向注入应使新增断言失败。

### [LOW] I-REC-08 映射声称「不 MarkConsumed」已在零 permit 分支断言，但代码没有该断言

- finding ID：`IA-S4.4-R02-F002`
- 对象/修订：`report-r02.md` `D3D18A13…EA6D`；`joint_readiness_test.go` `B3D0E336…B562`
- 位置：`report-r02.md` §6 I-REC-08 行；`test/integration/recovery/joint_readiness_test.go:366-392`
- 现状：报告写「『不 MarkConsumed』只在零 permit 分支断言」。实际上 `KeptDeletedZeroPermit_NoStateWritten` 只查询 `users.status`、`recovery_account_reviews` 计数和 `audit_events` 计数；整个文件未出现 `MarkConsumed`、`StageConsumed` 或 `consumedMarkCount`。
- 触发：按 I-REC-08 的「不 MarkConsumed」验收项复核测试代码时。
- 影响：证据映射再次出现「报告声称、代码未断言」的小范围不一致；不会造成运行时错误，但会高估 I-REC-08 的自动化覆盖。
- 建议：删除该句，或在零 permit 用例增加 `journal.consumedMarkCount()==0`（或读取 snapshot/Stage 确认未推进）后保留；若增加断言，应同时说明它只证明零 permit 分支未消费，不代表 I-REC-08 全矩阵。
- 验证：逐字对照 `joint_readiness_test.go:366-392` 的断言集合；增加后分组运行应看到该断言执行。

### [LOW] POSIX 分支无条件 NOT RUN，未来真实 Linux/持久卷环境也不会执行，且会输出错误的 WSL 原因

- finding ID：`IA-S4.4-R02-F003`
- 对象/修订：`identity_recovery_test.go` `0742ECA2…696D`；`journal_faults_test.go` `7049DF0B…54B4`
- 位置：`test/integration/recovery/identity_recovery_test.go:132-150`；`test/integration/recovery/journal_faults_test.go:207-221`
- 现状：`s44FileJournalSupported()` 在 linux/darwin 等平台返回 true；两个子测试在 true 分支直接 `t.Log(...)` 后 `return`，日志固定写「WSL has no Linux Go toolchain and system package installation is not authorized」。本轮 WSL 确实没有 Linux Go（实测 `go: command not found`），所以在当前环境该 NOT RUN 是诚实的；但在未来提供 Linux Go/持久卷的环境里，代码仍会提前 return，且原因不再是事实。
- 触发：在原生 Linux、Linux 容器或具备 POSIX 文件 journal 的 CI 上运行这两个子测试。
- 影响：假阴性（本可执行的持久卷场景永远不执行）和诊断误导；不会伪造 PASS，但会让任务关闭证据无法通过这两个文件取得。
- 建议：把「是否具备环境」与「是否 POSIX」分离。可用显式环境变量或探测真实文件 journal 能力决定是否执行；不具备时输出准确的本地原因。若本轮确实无法执行，至少把原因改为当前检测事实，而不是把 WSL 限制写成所有 POSIX 的固定结论。
- 验证：在 Linux 环境运行 `-run 'CrossVersionReplay|FileJournalCrashCorruption' -v`；具备工具链时应执行真实 journal 断言，不具备时应输出与环境一致的 NOT RUN/失败原因，而不是固定的 WSL 文案。

## 7. 原 Finding 逐条处置

| 原 finding | r02 处置 | 判定 | 依据 |
|---|---|---|---|
| `IA-S4.4-R01-F001` 跨 key 版本重放无断言 | 子测试改名为 `CrossVersionReplay_PersistentVolume_NotRun`；Windows 仅断言构造器 fail-closed，POSIX 明确 NOT RUN，不再计入已覆盖 | **CLOSED（按 r01 条件 b 降级）** | `identity_recovery_test.go:132-150`；r02 §4.1、§6 已明确 I-REC-02 真实重放未取得证据。注意：这是口径关闭，不是持久卷证据已取得。 |
| `IA-S4.4-R01-F002` 文件 journal 崩溃/损坏无执行 | 子测试改名为 `FileJournalCrashCorruption_PersistentVolume_NotRun`；Windows 仅构造器级断言，POSIX 明确 NOT RUN | **CLOSED（按 r01 条件 b 降级）** | `journal_faults_test.go:207-221`；r02 §4.2、§8 已登记文件故障注入 NOT RUN。注意：这是口径关闭，不是故障注入已执行。 |
| `IA-S4.4-R01-F003` 报告声称 `platform.internal_error` 已断言但只记录日志 | 代码新增 `platform.PublicCode(err) != platform.CodePlatformInternalError` 真实比较 | **CLOSED** | `joint_readiness_test.go:372-374`；分组运行输出 `asserted code=platform.internal_error`，断言实际执行。 |
| `IA-S4.4-R01-F004` I-REC 映射夸大 | §6 重写为「已断言/部分断言/未断言」并明确 keyring 矩阵不等价 I-REC-02 | **CLOSED（主体）** | r02 §6 逐项列出 I-REC-01/06/07 未断言、I-REC-02/03/04/05/08/09/10 部分断言；残余 I-REC-08 单句见 R02-F002。 |
| `IA-S4.4-R01-F005` 未标明 journal 为 `softwareJournal` 替身 | 代码注释与报告 §4.5 明确 recovery/article journal 为测试替身 | **CLOSED** | `joint_readiness_test.go:187-191`；r02 §4.5 列出真实实现与替身范围。 |
| `IA-S4.4-R01-F006` 顶层测试名非三段式 | 三个顶层函数改为 `TestIdentityRecovery_..._FailClosed`、`TestJournalFaults_..._FailClosed`、`TestJointReadiness_..._FailClosed` | **CLOSED** | 三文件 `func Test...` 实测；分组 `-run` 仍命中并执行 15 个子测试。 |
| `IA-S4.4-R01-F007` `kept_deleted` 用例重复 | 删除 `KeptDeletedCanonicalReview` 正向流程，只保留零 permit 负向 | **STILL OPEN（覆盖回退）** | 被删的四项断言未被 `runtime_recovery_test.go:541-605` 覆盖，详见 R02-F001。 |
| `IA-S4.4-R01-F008` schema 不删除且不登记 registry | 明确登记为已知残留，不 DROP、不写 S4.17 registry | **CLOSED（降级登记）** | `joint_readiness_test.go:22-26`；r02 §4.8、§9 与实现一致。 |
| `IA-S4.4-R01-F009` 「16/16」计数含义 | 改为「15 个子测试 = 10 实质 + 5 部分/未取证」，并解释 Go PASS 不等于底层行为已验证 | **CLOSED** | r02 §7 与实际分组运行 15 项一致；5 条 NOT RUN/部分日志与 §8 对应。 |
| `IA-S4.4-R01-F010` waiver 口径 | 保持 `waived-for-test-scope` 为 test scope only，不宣称 S4.5/S4.6 完成 | **CLOSED** | r02 §1、§11 与 `framework.md:1280` 定义句一致；未发现正面生产结论。 |

## 8. NOT RUN / unknown 边界（本评审确认）

| 项目 | 状态 | 本评审判断 |
|---|---|---|
| 真实进程崩溃 / 事务中断后的持久卷重放（I-REC-10） | NOT RUN | 受控软件 journal 重启不能证明 fsync/落盘；作者如实登记 |
| 文件 journal 部分写、HMAC/链损坏、重复 sequence、只读、ENOSPC、fsync 失败（I-REC-04） | NOT RUN | Windows 构造器在触盘前 fail-closed；POSIX 无 Linux Go 工具链 |
| retained v1/v2 跨 key 版本重放（I-REC-02） | NOT RUN | Windows 只有构造器级 fail-closed；真实重放未执行 |
| 生产 key retirement / S4.6 双前置 evidence（I-REC-03） | NOT RUN | S4.6 未实施；只有 verify-only-only keyring 与构造器分类 |
| 生产 Anchor / 生产 provider（S4.5） | NOT RUN | S4.5 未实施；未用模拟替身声称通过 |
| I-REC-06 / I-REC-07 | 未断言 / NOT RUN | 不在三个文件范围；作者未声称通过 |
| 独立故障域证据 | unknown / 未验证 | 本轮未取得；不得据此判定 ADR-0030 `(c)` 通过 |
| POSIX 全部行为 | NOT RUN | 本轮实测 WSL 无 Linux Go（`go: command not found`）；未授权安装系统包 |

## 9. 未验证风险与限制

- 本评审没有 POSIX 持久卷、真实进程崩溃、文件系统故障注入或生产 Anchor/provider，无法对 F001/F002 的底层行为给出 PASS。
- 软件 journal 的并发与阶段断言不能替代文件 journal 的 fsync、部分写、只读、ENOSPC 行为。
- `report-r02.md` 中的 G0 基线、RED 失败输出属于历史/改前状态，本轮未独立复现；本报告不把它们作为当前通过依据。
- 仓库内 18 个既存跟踪文件改动与本批次无关；本报告只按 mtime 与文件边界确认 r02 没有改这些文件，未对其内容作全面复核。
- 本报告是独立评审结论，不代替 Lead/用户对结果的接受、任务关闭授权、风险接受或后续生产授权。

## 10. 复审条件与建议下一步

1. 按 R02-F001 恢复 `KeptDeletedCanonicalReview` 正向流程，或等价补回四项断言；修复后重新执行分组 `-v` 与整包测试。
2. 按 R02-F002 修正 I-REC-08 映射，或补上「不 MarkConsumed」的真实断言。
3. 按 R02-F003 修正 POSIX 分支的环境判定与 NOT RUN 原因，避免未来 Linux 环境继续假阴性。
4. 上述修订后提交 r03 三文件新哈希与作者证据；本评审按原 finding ID 复核，不覆盖本报告或 r01 报告。
5. 任务关闭仍需独立取得真实持久卷崩溃/损坏/跨版本重放证据、S4.5、S4.6 与独立故障域证据；本轮测试通过不等同于这些条件满足。

## 11. 声明

- 本报告仅写入 `.spec/wip/identity-auth/review-code-s4.4-r02.md`。
- 未修改任何被评审文件；未执行 `git add`、`git commit`、`git stash`、`git restore`、`git checkout`、`git branch` 等 Git 写操作。
- 报告中的命令与结果均为本轮真实执行；作者自述的 G0/RED 已标注未独立验证。