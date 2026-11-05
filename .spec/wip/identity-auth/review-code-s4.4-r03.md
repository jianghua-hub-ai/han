# identity-auth-S4.4 r03 独立复审报告

- 评审角色：独立评审人（非作者），只读评审
- 评审对象：`identity-auth-S4.4` Recovery Journal、keyring 与联合恢复集成测试的 r03 修订
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 报告位置：`.spec/wip/identity-auth/review-code-s4.4-r03.md`
- 结论等级：**PASS WITH CONDITIONS**
- 新发现：2 项 LOW + 2 项 INFO；无 BLOCKER / HIGH / MEDIUM
- 原 R02 发现：`R02-F001`／`R02-F002`／`R02-F003` 均 **CLOSED**（F003 仅代码闭合；POSIX 实跑仍 `NOT RUN`）
- 任务状态：**仍不得关闭**。本轮绿灯只代表测试执行结果，不代表任务验收、真实持久卷验证或生产结论。

## 1. 评审对象与哈希（本轮实测）

| 文件 | 字节 | SHA256（实测） |
|---|---:|---|
| `test/integration/recovery/identity_recovery_test.go` | 15,490 | `5E9FBCA44EB759F29135C21866739D13585C673A1825624F0B7462717472211A` |
| `test/integration/recovery/journal_faults_test.go` | 11,082 | `42EEFE8D22CCD0674D356FA7457FDBD376B46B8B989DF6FE360ED4AAF427E2C5` |
| `test/integration/recovery/joint_readiness_test.go` | 22,162 | `885766D80D96EC19048316A0E3EA2D0A444BD644106F3581DADC06B56B883C3F` |
| `.spec/wip/identity-auth/evidence-s4.4/report-r03.md` | 25,582 | `70E62D5A18B453B448320A18C109A79309DC4B3AE1BD4037882D890D63AD4578` |

历史基线本轮复核后保持原样：

| 文件 | 字节 | SHA256（实测） |
|---|---:|---|
| `.spec/wip/identity-auth/evidence-s4.4/report.md`（r01） | 15,324 | `A1986FDEE6B0669B57546A0A65AF75A1AB9E9EB6CF653EB8629812B6BE9C2D22` |
| `.spec/wip/identity-auth/evidence-s4.4/report-r02.md` | 26,950 | `D3D18A13B6FF67772A05AD35DF0AC6EEFEFC040AECB4BB3D7E0F15CA8487EA6D` |
| `.spec/wip/identity-auth/review-code-s4.4-r01.md` | 28,084 | `BF926AAA70E9C4523EA04D7D1DEE20B2665DB8829D8A0CB21DB3CA9B5B276163` |
| `.spec/wip/identity-auth/review-code-s4.4-r02.md` | 24,041 | `F7EB4F8C514BA2DF4553905A546D1207D3605FF688FA9FA7928331A4C24F52DE` |

## 2. 评审依据

- `.codex/skills/add-code-review/SKILL.md`（完整正文）
- `.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`
- `.codex/skills/add-tdd-go/SKILL.md`
- `.codex/guidance/task-execution-workflow.md`、`.codex/guidance/coding-standards.md`
- 任务卡 `.add/project/knowledge/tasks/tasks.md:638-655`（`:649` requires 含 `identity-auth-S4.5`／`identity-auth-S4.6`；`:652` 关闭条件）
- 测试来源 `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:746-755`（I-REC-01～10）
- `.add/project/knowledge/decisions/framework.md:1227-1233`、`:1278-1284`、`:1387`（ADR-0030 r17 决策 3、r18 补充、r20 门禁）
- 上游实现交叉核对：`internal/recovery/journal.go`（构造、`Begin`、`MarkConsumed`、`writable`）、`journal_snapshot.go`（`OpenVerifiedRecoveryJournal` 与只读 reader）、`keyring.go`（`NewKeyring`／`retained`）、`filelock_flock.go`／`filelock_unsupported.go`（POSIX build tag）、`test/integration/recovery/runtime_fixtures_test.go`（`softwareJournal`／`consumedMarkCount`）
- r01／r02 评审报告与 r03 作者证据全文

## 3. 本轮实际执行的命令与结果

| 命令 | 结果 |
|---|---|
| `go test -tags=integration ./test/integration/recovery -run 'TestIdentityRecovery|TestJournalFaults|TestJointReadiness' -count=1 -v` | 16 个 `=== RUN` 子测试、16 个 `--- PASS`；包 `ok han/test/integration/recovery 6.199s`；exit 0 |
| 同上分组输出的关键日志 | `joint_readiness_test.go:447: ... journal_stage=consumed journal_consumed_at_present=true ... reviews=1 credential_reset=0 canonical_audit=1`；`joint_readiness_test.go:477: ... status=active reviews=0 audits=0 consumed_marks=0` |
| `go test -tags=integration ./test/integration/recovery -count=1` | `ok han/test/integration/recovery 37.147s`；exit 0（Lead 复跑 36.943s，环境波动） |
| `go test -tags=integration ./test/integration/recovery -race -count=1` | `ok han/test/integration/recovery 39.468s`；exit 0，无数据竞争报告 |
| `GOOS=linux GOARCH=amd64 go vet -tags=integration ./test/integration/recovery` | exit 0（POSIX 分支可类型检查/编译，不是不可达死代码） |
| `go vet ./...` | exit 0 |
| `go build ./...` | exit 0 |
| `gofmt -l test/integration/recovery` | 无输出（0 行），exit 0 |
| `rg -n "t\.Skip|t\.Parallel|DROP SCHEMA|DROP DATABASE|DROP TABLE|DSN|password=|HAN_TEST_POSTGRES"`（三文件） | 无 `t.Skip`、无 `t.Parallel`、无任何 `DROP`；仅 `joint_readiness_test.go:29-38` 读取 `HAN_TEST_POSTGRES_DSN`，空值时 `t.Fatal("... no Skip")`，错误信息不含 DSN 值 |
| mtime 扫描（近 90 分钟，`test/`、`internal/`、`cmd/`、`migrations/`） | 仅三个被评审文件被写：`joint_readiness_test.go` 15:08:46、`journal_faults_test.go` 15:08:51、`identity_recovery_test.go` 15:08:51 |
| 越界交叉核（tracked 文档 mtime） | `tasks.md` 11/2 19:35、`framework.md` 11/3 14:02、`identity-auth/test-plan.md` 10/29 23:57，均早于本批次；`git status` 中这些 `M` 项为既存改动，非本批次写入 |
| `git status --porcelain -- test/integration/recovery` | 三个被评审文件与同目录既有 8 个测试文件均为未跟踪 `??`；本批次没有对既有文件的跟踪改动 |
| `git rev-parse HEAD` / `--abbrev-ref HEAD` | `6da8abb...` / `main` |

## 4. 两轮检查范围

- 第一轮（原 finding 对账）：逐条核对 `R02-F001`／`F002`／`F003` 的代码落点和实际执行证据，不看作者自述单独下结论。
- 第二轮（上游与消费者）：核对 `FileRecoveryJournal` 构造/`Begin`/`MarkConsumed`/只读 reader 的真实语义、`keyring.retained()` 的重放检测、POSIX build tag 与测试平台判据是否一致、`softwareJournal.consumedMarkCount()` 是否统计被拒绝调用、任务卡 requires 与 ADR 门禁是否被误读。

## 5. 原 R02 finding 逐条处置

| 原 finding | 等级 | 判定 | 一句话依据 |
|---|---|---|---|
| `IA-S4.4-R02-F001`：`KeptDeletedCanonicalReview` 四项断言被删除 | MEDIUM | **CLOSED** | `joint_readiness_test.go:366-449` 已恢复：`:404-413` 用 SQL 断言 `journal_stage=consumed` 且 `journal_consumed_at` 非空；`:415-426` 断言 review 行数 `=1` 且 `credential_reset_at` 非空数 `=0`；`:443-445` 另断言 `status=deleted`、`decision=kept_deleted`、canonical audit `=1`；本轮 `-v` 实测四项值 `consumed / true / 1 / 0`。 |
| `IA-S4.4-R02-F002`：零 permit 分支「不 MarkConsumed」无断言 | LOW | **CLOSED** | `joint_readiness_test.go:474-476` 新增 `world.journal.consumedMarkCount() != 0` 即 Fatal；`runtime_fixtures_test.go:318-345` 在每次调用入口先自增 `consumedCalls`（含被拒绝调用），`:347-351` 返回计数；本轮实测输出 `consumed_marks=0`。 |
| `IA-S4.4-R02-F003`：POSIX 分支无条件 return、固定环境原因 | LOW | **CLOSED（代码闭合；POSIX 实跑仍 `NOT RUN`）** | `identity_recovery_test.go:19-26` 按 `runtime.GOOS` 白名单分流，白名单与 `filelock_flock.go:1` 的 build tag（aix/darwin/dragonfly/freebsd/linux/netbsd/openbsd/solaris）逐项一致；`:28-32` 返回当次 `runtime.GOOS` 原因；`:202-242`、`:244-298`、`:300-323`、`journal_faults_test.go:210-253` 的 POSIX 分支均为真实断言，`GOOS=linux go vet` exit 0 证明可编译。Windows 上仍按 `NOT RUN` 登记，未当作已执行证据。 |

补充：`R02-F001` 的恢复没有挤掉零 permit 负向用例——`KeptDeletedZeroPermit_NoStateWritten`（`joint_readiness_test.go:450-478`）仍保留 `platform.internal_error`、`status=active`、`reviews=0`、`audits=0`、`consumed_marks=0` 五类断言；本轮 `-v` 两个子测试均独立 PASS。

## 6. r03 新发现

### IA-S4.4-R03-F001（LOW）平台判据与 Go build tag 语义不完全一致，android/ios 上会走错分支

- 位置：`test/integration/recovery/identity_recovery_test.go:19-26`
- 依据：`filelock_flock.go:1` 的 build tag 为 `aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris`；Go 在 `GOOS=android` 时同时满足 `linux` tag、`GOOS=ios` 时同时满足 `darwin` tag。因此 android/ios 上文件 journal 实际受支持，但 `s44FileJournalSupported()` 返回 `false`。
- 触发场景：在 android/ios（或未来以交叉编译方式执行集成测试）运行四个 POSIX 分支时，测试会进入「非 POSIX」分支并断言 `ErrUnsupportedPlatform`；而 `NewFileRecoveryJournal` 会先进入参数校验（例如缺 `Anchor` 返回 `ErrJournalCorrupt`），`OpenVerifiedRecoveryJournal` 返回 `ErrAdmissionBinding`，断言将失败。
- 影响：目标平台（Windows/Linux）无影响；属于非目标平台的潜在假红/错误结论，不阻断本轮。
- 建议：白名单补 `android`、`ios`，或改用与 build tag 同源的探测（新增一个 build-tag 测试文件不在本任务文件所有权内，需 Lead 决定）。
- 证据：上述 build tag 原文与 `s44FileJournalSupported` 源码；本轮 `GOOS=linux go vet` 通过只能证明 Linux 分支可编译，不能证明 android/ios 判据正确。

### IA-S4.4-R03-F002（LOW）POSIX 分支的 `JournalHeadAnchor` 是内存替身，r03 证据未显式披露

- 位置：`identity_recovery_test.go:34-67`（`s44MemoryAnchor`）；证据 `report-r03.md:109-112`、`:194`
- 依据：`s44MemoryAnchor` 是本地内存 CAS 替身，代码注释 `:34-36` 已说明「production anchor is the recovery store」；r03 报告 §4.3 只写「用真实 `FileRecoveryJournal`」，§8 只列「生产 Anchor NOT RUN」，未说明 POSIX 分支的 anchor 是内存替身。
- 触发场景：读者若把「真实 file journal」理解为连同生产 Anchor 一起验证，会高估 POSIX 分支的覆盖范围。
- 影响：不构成生产结论过度声明（POSIX 本轮本就没有执行，且 §8 明确生产 Anchor `NOT RUN`），但证据口径不完整，建议补齐。
- 建议：在报告 §4.3 或 §8 明确「journal 为真实 `FileRecoveryJournal`；`JournalHeadAnchor` 为内存 CAS 替身；生产 Anchor 依赖 S4.5，仍 `NOT RUN`」。
- 证据：`s44MemoryAnchor` 源码与 `s44NewFileJournal`（`:80-92`）的传参。

### IA-S4.4-R03-F003（INFO）`..._NotRun`／`..._WindowsOnly_` 命名与 POSIX 实际行为不一致

- 位置：`identity_recovery_test.go:202`、`:244`、`:300`；`journal_faults_test.go:210`
- 依据：这些子测试在 POSIX 上会执行真实断言（跨版本重放、verify-only 读取、空参数构造校验、partial trailing write），名字却仍标注 `NotRun`／`WindowsOnly`；只有 Windows 分支才是 `NOT RUN`。
- 影响：不影响执行与结论；未来按名字筛选或汇总时可能误分类。报告 §4.3、§7 已解释，属可读性问题。
- 建议：改名或在测试内加注释说明「名称描述 Windows 口径；POSIX 分支实际执行」。

### IA-S4.4-R03-F004（INFO）证据中「实际执行断言」与数值顺序存在可误读表述

- 位置：`report-r03.md:64`、`:81`
- 依据：`:64`「POSIX 分支实际执行断言」可被读成「本轮已执行」，但同报告 `:120`、`:197` 明确 POSIX 本轮 `NOT RUN`；`:81` 写「`consumed / true / 0 / 1`」，与日志字段顺序 `reviews=1 credential_reset=0` 不同，但与其 §4.1 的断言枚举顺序（stage、present、resetReviews、reviews）一致，数值本身正确。
- 影响：无代码或结论影响；建议改成「POSIX 分支包含真实断言（本轮未执行）」并标明数值顺序。
- 建议：措辞级修订即可。

## 7. 覆盖真实性与计数核对

- 本轮分组实跑确为 **16 个子测试 = 11 项实质断言 + 5 项部分/未取证**，与 `report-r03.md` §7 一致：
  - 实质 11 项：`KeyringRotationFingerprintMatrix`、`SoftwareJournalSecretFingerprintAndStages`、`PositiveGenesisAdmission`、`RecoveryNamespaceMismatchFailsClosed`、`ArticleNamespaceMismatchFailsClosed`、`ArticleHeadDriftFailsClosed`、`CorruptRecoveryEntryFailsClosed`、`KeptDeletedCanonicalReview`、`KeptDeletedZeroPermit_NoStateWritten`、`MarkConsumedDuplicateAndConcurrent`、`CorruptRecoverySnapshotRejectedByAdmission`。
  - 部分/未取证 5 项：`CrossVersionReplay_PersistentVolume_NotRun`、`VerifyOnlyKeyRetirement_S46_NotRun`、`UnsupportedPlatformConstructors_WindowsOnly_FailClosed`、`ReplayAfterControlledRestart_SoftwareJournalOnly`、`FileJournalCrashCorruption_PersistentVolume_NotRun`。
- 并发用例不是「至少 1 成功」：`journal_faults_test.go:69-77` 精确断言 `successes=1`、`mismatches=15`，并断言 `consumedMarkCount()==3+16`、caller 快照同为 19 条；`:83-110` 断言 4 条 entry、sequence/链连续、每 run 恰好一个 begun 一个 consumed。
- 崩溃口径诚实：`ReplayAfterControlledRestart_SoftwareJournalOnly` 在 `journal_faults_test.go:150` 显式打印 `NOT RUN: real process crash / transaction interruption durability ... cannot prove fsync or on-disk replay`；`FileJournalCrashCorruption_PersistentVolume_NotRun` 在 Windows 只做构造器级 fail-closed。作者没有把受控重放写成真实崩溃通过。
- r02 的 15 个子测试全部仍在 r03 中（逐项对照 `report-r02.md` §7 与 r03 实跑列表），r03 只新增恢复的 `KeptDeletedCanonicalReview`，未删除其他用例或用例内断言；不存在「删除覆盖换取绿灯」。
- 既有 8 个测试文件未被修改：`git status` 显示它们与本批次三个文件同为未跟踪历史文件，mtime 扫描近 90 分钟仅三个被评审文件被写；本轮未发现对新文件的削弱、替换或掩盖。
- 测试卫生：无 `t.Skip`、无 `t.Parallel`、无 `DROP`；schema 固定 `han_s44_<hex>` 并只通过 `t.Cleanup` 关闭连接池；DSN 只读环境变量、空值 `t.Fatal`、错误不回显 DSN；未写入 S4.17 registry、不清理他人 schema。
- 过度声明扫描：`report-r03.md` 中「生产就绪／生产解锁／ACL 已验证／真实环境 PASS」只出现在否定句（`:17`、`:234`）；I-REC-01/06/07 明确「未断言」；`tasks.md:649` 的 requires 未被读成已满足（`:198`、`:231` 明确 `waived-for-test-scope` 不改变 requires、S4.5/S4.6 未完成）。

## 8. NOT RUN / unknown 边界（本轮独立确认）

| 项目 | 状态 |
|---|---|
| POSIX 文件 journal 的跨 key 版本重放、verify-only 只读、partial trailing write | `NOT RUN`（Windows；代码已写且 `GOOS=linux go vet` 通过，但未实跑） |
| 真实进程崩溃／事务中断后的持久卷重放与 fsync 证据（I-REC-10） | `NOT RUN`；受控软件 journal 不能替代 |
| 文件 journal HMAC/链损坏、重复 sequence、只读、ENOSPC、fsync 失败（I-REC-04） | `NOT RUN` |
| 生产 Anchor（S4.5）、生产 provider 依赖断言（I-REC-01 等） | `NOT RUN` |
| 生产 key retirement／verify-only 真实双前置 evidence（S4.6，I-REC-03） | `NOT RUN`，S4.6 未实施 |
| I-REC-06 / I-REC-07 | `未断言 / NOT RUN`（不在三文件范围） |
| 独立故障域 | `unknown` / 未验证 |
| android/ios 平台分支 | 未执行；且存在 `R03-F001` 的判据风险 |

## 9. 未验证风险

- POSIX 分支虽可编译，但真实运行时的文件锁交互、Anchor CAS、磁盘错误注入行为本轮无证据；`R03-F001`／`R03-F002` 是这一范围的透明度风险。
- `han_s44_<hex>` schema 不 DROP、未登记 registry，会持续累积残留；本轮未统计精确增量，属已知资源残留，不是代码缺陷但需 Lead 决策清理策略（清理需另行授权）。
- 本轮绿灯依赖 Windows + 真实 PostgreSQL；不能外推为 Linux 持久卷、故障域或生产环境结论。

## 10. 复审条件与后续

1. 具备 Linux Go 工具链后实跑四个 POSIX 分支（跨版本重放、verify-only 读取、partial write、构造器校验），保存 `-v` 输出；本轮不计入已覆盖。
2. 在真实持久卷上补做崩溃/损坏/fsync/ENOSPC 注入（I-REC-04/10）；在此之前不得宣称任务关闭。
3. S4.5 生产 Anchor、S4.6 生产 key retirement 未实施；`waived-for-test-scope` 仅 test scope，不改变 `tasks.md:649` 的 requires。
4. 独立故障域证据仍 `unknown`，不得以本轮结果替代。
5. 建议按 `R03-F001`／`R03-F002` 做低成本修订；两者均不阻断本轮 PASS WITH CONDITIONS。

## 11. 声明

- 本报告为独立评审结论，不代替 Lead/用户对结果的接受、任务关闭授权、风险接受或后续生产授权。
- 本轮只读取被评审文件与依据，只新增本报告；未修改任何被评审文件、r01/r02 证据或评审报告，未执行任何 Git 写操作。
- 报告中的通过结论仅指本轮实际执行的命令结果；`NOT RUN` 与 `unknown` 范围不被写成通过。