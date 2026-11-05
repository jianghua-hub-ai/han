# identity-auth-S4.4 r04 独立复审报告

- 评审角色：独立评审人（非作者），只读评审
- 评审对象：`identity-auth-S4.4` Recovery Journal、keyring 与联合恢复集成测试的 r04 收口修订
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮 `git rev-parse` 实测）
- 报告位置：`.spec/wip/identity-auth/review-code-s4.4-r04.md`（不覆盖 r01／r02／r03 报告）
- 结论等级：**PASS WITH CONDITIONS**
- 新发现：**1 项 LOW + 4 项 INFO**；无 BLOCKER / HIGH / MEDIUM
- 原 R03 发现：`R03-F001`／`R03-F002`／`R03-F003`／`R03-F004` 全部 **CLOSED**（F001 的 `ios` 编译证据仍缺，另记 `R04-F002`）
- 任务状态：**仍不得关闭**。本轮绿灯只代表 Windows + 真实 PostgreSQL 上的执行结果，不代表 POSIX 分支运行行为、真实持久卷故障注入、S4.5／S4.6 或任何生产结论。

## 1. 评审对象与哈希（本轮实测）

| 文件 | 字节 | SHA256（本轮实测） | 与 Lead 给定 |
|---|---:|---|---|
| `test/integration/recovery/identity_recovery_test.go` | 16,345 | `AC5EB4B520EB9379D12E8923E3C33DCF7303A4ADD7DB72D386C1386FC3E2AD47` | 一致 |
| `test/integration/recovery/journal_faults_test.go` | 11,128 | `61BC9925A60BEC5CBA31597D02F654C171B089F4C3FF5509E1DACD8B01FA7558` | 一致 |
| `test/integration/recovery/joint_readiness_test.go` | 22,162 | `885766D80D96EC19048316A0E3EA2D0A444BD644106F3581DADC06B56B883C3F` | 一致，且与 r03 相同（本轮确实未改） |
| `.spec/wip/identity-auth/evidence-s4.4/report-r04.md` | 27,633 | `E9E850BC7AC7A7C1BF0EB54478918B478944B5D992155AF5D3B85E0438C7107B` | 一致 |

历史基线（本轮逐字节复核，均保持原样，未被改写）：

| 文件 | 字节 | SHA256（本轮实测） |
|---|---:|---|
| `.spec/wip/identity-auth/evidence-s4.4/report.md`（r01） | 15,324 | `A1986FDEE6B0669B57546A0A65AF75A1AB9E9EB6CF653EB8629812B6BE9C2D22` |
| `.spec/wip/identity-auth/evidence-s4.4/report-r02.md` | 26,950 | `D3D18A13B6FF67772A05AD35DF0AC6EEFEFC040AECB4BB3D7E0F15CA8487EA6D` |
| `.spec/wip/identity-auth/evidence-s4.4/report-r03.md` | 25,582 | `70E62D5A18B453B448320A18C109A79309DC4B3AE1BD4037882D890D63AD4578` |
| `.spec/wip/identity-auth/review-code-s4.4-r01.md` | 28,084 | `BF926AAA70E9C4523EA04D7D1DEE20B2665DB8829D8A0CB21DB3CA9B5B276163` |
| `.spec/wip/identity-auth/review-code-s4.4-r02.md` | 24,041 | `F7EB4F8C514BA2DF4553905A546D1207D3605FF688FA9FA7928331A4C24F52DE` |
| `.spec/wip/identity-auth/review-code-s4.4-r03.md` | 16,833 | `1711CEC81B1039791D189C44CCFA9C2184AABB4DA5CFB5DA81176BE89A72095C` |

## 2. 评审依据

- `.codex/skills/add-code-review/SKILL.md`（完整正文）与 `.codex/guidance/code-review-checklist.md`
- `.codex/skills/add-tdd-go/SKILL.md`；`.codex/guidance/task-execution-workflow.md`、`.codex/guidance/coding-standards.md`
- 任务卡 `.add/project/knowledge/tasks/tasks.md:638-655`（`:649` requires 含 `identity-auth-S4.5`／`identity-auth-S4.6`；`:652` 关闭条件）
- 测试来源 `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:746-755`（I-REC-01～10）
- `.add/project/knowledge/decisions/framework.md:1227-1233`、`:1278-1284`、`:1387`
- 上游只读交叉核对：
  - `internal/recovery/journal.go:76-84`（`JournalHeadAnchor` 仅 `Load`／`CompareAndSwap`）、`:239-293`（`NewFileRecoveryJournal` 校验顺序：`ErrUnsupportedPlatform` → `validateJournalOptions` 的 `ErrJournalCorrupt`）
  - `internal/recovery/journal_snapshot.go:36-60`（`OpenVerifiedRecoveryJournal`：空 `MaxBytes` → `ErrAdmissionBinding`；非 flock 平台 → `ErrAdmissionUnsupported+ErrUnsupportedPlatform`）
  - `internal/recovery/filelock_flock.go:1`、`filelock_unsupported.go:1`（POSIX build tag）
  - `test/integration/recovery/runtime_fixtures_test.go:251-345`（`softwareJournal` 替身：硬编码测试 key、`consumedCalls` 统计含被拒调用）
  - `test/integration/recovery/fixtures_test.go:410-438`、`:489-493`（keyring fixture 校验与 `syntheticFingerprint`）
- r01／r02／r03 评审报告与 r04 作者证据全文（253 行）

## 3. 本轮实际执行的命令与结果

| 命令 | 结果 |
|---|---|
| `go test -tags=integration ./test/integration/recovery -run 'TestIdentityRecovery|TestJournalFaults|TestJointReadiness' -count=1 -v` | exit 0；`=== RUN` 19 条（3 个顶层 + 16 个子测试）；`--- PASS` 子测试 16 条；包 `ok han/test/integration/recovery 8.182s`；子测试名全部为 r04 新名；输出含 5 条显式 `NOT RUN` 日志 |
| 同上 `-v` 关键日志 | `joint_readiness_test.go:447: ... journal_stage=consumed journal_consumed_at_present=true ... reviews=1 credential_reset=0 canonical_audit=1 journals=softwareJournal(test double)`；`joint_readiness_test.go:477: ... code=platform.internal_error status=active reviews=0 audits=0 consumed_marks=0`；`identity_recovery_test.go:226/270/332` 与 `journal_faults_test.go:221` 均为 `branch=non-posix runtime.GOOS=windows ... NOT RUN`；`journal_faults_test.go:150` 为真实崩溃 `NOT RUN` 说明 |
| `go test -tags=integration ./test/integration/recovery -count=1` | `ok han/test/integration/recovery 41.276s`；exit 0（Lead 复跑 33.445s，属环境波动） |
| `go test -tags=integration ./test/integration/recovery -race -count=1` | `ok han/test/integration/recovery 40.111s`；exit 0，无数据竞争报告 |
| `gofmt -l test/integration/recovery` | 无输出（0 行），exit 0 |
| `go vet -tags=integration ./test/integration/recovery`（linux／darwin／windows） | 三次均 exit 0、无输出 |
| `GOOS=android GOARCH=arm64 CGO_ENABLED=0 go vet -tags=integration ./test/integration/recovery` | exit 0（白名单新增别名确实可编译） |
| `GOOS=illumos GOARCH=amd64 CGO_ENABLED=0 go vet -tags=integration ./test/integration/recovery` | exit 0 |
| `GOOS=ios GOARCH=arm64 CGO_ENABLED=0 go vet -tags=integration ./test/integration/recovery` | exit 1：`ios/arm64 requires external (cgo) linking, but cgo is not enabled`（本机工具链限制，非代码错误；详见 `R04-F002`） |
| `go vet ./...`／`go build ./...` | 均 exit 0、无输出 |
| `rg -n "t\.Skip|t\.Parallel|DROP SCHEMA|DROP DATABASE|DROP TABLE|password=|HAN_TEST_POSTGRES_DSN"`（recovery 目录） | 无 `t.Skip`、无 `t.Parallel`、无任何 `DROP`；DSN 仅在 `joint_readiness_test.go:29-38`、`runtime_fixtures_test.go:104-113` 读取，空值时 `t.Fatal("... no Skip")`，不回显 DSN 值 |
| mtime 扫描（近 3 小时，含 `*.go/*.sql/*.json/*.md`） | 本批次窗口（15:23:59–15:27:58）内仅 `journal_faults_test.go`（15:23:59）、`identity_recovery_test.go`（15:25:22）、`_scratch/s44-r04-grouped.log`（15:27:05）、`report-r04.md`（15:27:58）；`joint_readiness_test.go` 停在 15:08:46（r03）；`internal/`、`cmd/`、`migrations/`、`tasks.md`、设计文档与 ADR 无本批次写入 |
| `git status --porcelain -- test/integration/recovery`／`git diff --stat -- test/integration/recovery` | 目录整体为未跟踪 `??`；无跟踪文件改动，diff 为空 |
| `git rev-parse --short HEAD`／`--abbrev-ref HEAD` | `6da8abb`／`main` |

## 4. 两轮检查范围

- 第一轮（原 finding 对账）：逐条核对 `R03-F001`～`F004` 在代码中的真实落点，并以 r03 作者分组日志（`.spec/wip/identity-auth/_scratch/s44-r03-grouped.log`）的行号与 r04 代码行号对位，判断是否存在删行或弱化。
- 第二轮（上游与消费者）：核对 `NewFileRecoveryJournal`／`OpenVerifiedRecoveryJournal` 的真实失败分类与 POSIX 分支断言预期是否一致；核对 `JournalHeadAnchor` 接口成员；核对 `softwareJournal` 替身语义；扫描过度声明、`t.Skip`、`DROP`、DSN 回显、`t.Parallel`；复核越界与历史证据哈希。

## 5. 原 R03 finding 逐条处置

| 原 finding | 等级 | 判定 | 一句话依据 |
|---|---|---|---|
| `IA-S4.4-R03-F001`：POSIX 白名单漏 `android`／`ios`（以及 `illumos`） | LOW | **CLOSED** | `identity_recovery_test.go:27-34` 白名单已含 `aix, android, darwin, dragonfly, freebsd, illumos, ios, linux, netbsd, openbsd, solaris`，与 `filelock_flock.go:1` 的 tag 经 GOOS 别名（android→linux、ios→darwin、illumos→solaris）一一对齐；本轮 `GOOS=android`、`GOOS=illumos` 的 `go vet -tags=integration` 均 exit 0，linux／darwin／windows 亦 exit 0。 |
| `IA-S4.4-R03-F002`：未披露 POSIX 分支 anchor 是内存替身 | LOW | **CLOSED** | 代码注释 `identity_recovery_test.go:42-45` 明写 test double、非生产 anchor；POSIX 日志 `:252`、`:308`、`journal_faults_test.go:253` 均带 `anchor=s44MemoryAnchor test double`；证据 `report-r04.md:99-105`、`:252` 明确真实范围仅文件 journal IO/链/校验 + 真实 keyring 指纹。附带一处新的事实笔误，另记 `R04-F001`。 |
| `IA-S4.4-R03-F003`：`NotRun`／`WindowsOnly` 命名与行为不一致 | INFO | **CLOSED** | 四个子测试已改为 `CrossVersionReplay_PersistentVolume`、`VerifyOnlyKeyRetention_PerPlatform`、`ConstructorValidation_PerPlatform`、`FileJournalCrashCorruption_PersistentVolume`（`identity_recovery_test.go:213`／`:255`／`:311`、`journal_faults_test.go:210`）；`-v` 实测 16 个子测试全部以新名被 `-run 'TestIdentityRecovery|TestJournalFaults|TestJointReadiness'` 命中并 PASS；`rg` 确认任何 `t.Run` 名中都不再含 `NotRun`／`WindowsOnly`。 |
| `IA-S4.4-R03-F004`：报告措辞与日志数值顺序可误读 | INFO | **CLOSED** | `report-r04.md` 已无「实际执行断言」表述，`report-r04.md:97`、`:123`、`:251` 统一为「POSIX 分支已实现真实断言，本轮 Windows 未执行」；实测日志字段顺序为 `branch` → `runtime.GOOS` → `result`（如 `identity_recovery_test.go:226`）。 |

### 5.1 R03-F001 细节核对

- 判据等价性成立：Go 在 `GOOS=android` 时同时满足 `linux` tag，`GOOS=ios` 时同时满足 `darwin` tag，`GOOS=illumos` 时同时满足 `solaris` tag，因此 `filelock_flock.go`（flock 后端）在这些 GOOS 上参与编译，测试白名单列出它们是正确方向。
- 反对向也已覆盖：`GOOS=windows` 分支仍断言 `ErrUnsupportedPlatform` 与 `ErrAdmissionUnsupported+ErrUnsupportedPlatform`（`identity_recovery_test.go:218-227`、`:265-271`、`:324-331`；`journal_faults_test.go:211-222`），与 `journal.go:243-245`、`journal_snapshot.go:40-42` 的真实返回一致。
- 残余：`ios` 仅有语义论证、无本机编译证据（`R04-F002`，不阻断）。

### 5.2 R03-F002 细节核对

- `s44MemoryAnchor`（`identity_recovery_test.go:46-76`）只实现 `Load` 与 `CompareAndSwap`，是本地内存 CAS 替身；`s44NewFileJournal`（`:91-103`）把它作为 `JournalOptions.Anchor` 传入真实 `recovery.NewFileRecoveryJournal`。
- 证据报告已把「真实部分」收窄到文件 journal 的 append／read、链连续性、HMAC／记录校验、真实 `recovery.Keyring` 的 secret fingerprint，且不把 POSIX 分支称为真实 recovery 链路——与代码一致。
- `report-r04.md:102` 把替身方法写成 `Load / CompareAndSwap / Store`，其中 `Store` 在接口与替身中都不存在（见 `R04-F001`）。

### 5.3 R03-F003 细节核对

- 旧名仅作为报告中的改名对照出现（`report-r04.md:111-114`）；代码侧无残留。
- `-v` 输出的 16 条子测试名单与 `report-r04.md` §7 的 A/B 分组逐项吻合，分组运行仍有效。

### 5.4 R03-F004 细节核对

- `report-r04.md:130` 声明修正的是日志字符串（`identity_recovery_test.go:270` 现为 `result=verify-only keyring ...`），实测日志确为 `verify-only`。
- `identity_recovery_test.go:263` 的 `t.Fatalf("verify-only-only keyring err=...")` 仍是失败消息文本，属措辞残留、不影响日志与断言，仅记 INFO（见 `R04-F004` 备注）。

## 6. r04 新发现

### IA-S4.4-R04-F001（LOW）证据把 `s44MemoryAnchor` 的方法写成含 `Store`，与代码不符

- 位置：`report-r04.md:102`
- 依据：`internal/recovery/journal.go:76-84` 的 `JournalHeadAnchor` 只有 `Load`、`CompareAndSwap`；`identity_recovery_test.go:56`、`:65` 也只实现这两个方法，全文无 `Store`。
- 触发场景：读者按报告去核对「替身模拟了哪些契约」时，会以为存在第三个方法，或误判接口形状。
- 影响：仅证据描述准确性；不影响测试执行、覆盖口径或任何生产结论，故不阻断。
- 建议：改为 `Load / CompareAndSwap`，或直接写「只模拟 revision 生成与 snapshot 保存」。

### IA-S4.4-R04-F002（INFO）白名单包含 `ios`，但本轮没有任何 `ios` 编译证据

- 位置：`identity_recovery_test.go:29`；`report-r04.md:89-97`
- 依据：本机 `GOOS=ios GOARCH=arm64 CGO_ENABLED=0 go vet` 因 `ios/arm64 requires external (cgo) linking` 失败；`android`、`illumos` 则实测 exit 0。`report-r04.md` 只声明 linux／darwin／windows 三种 GOOS 通过，未把 `ios` 写成已验证，因此不构成过度声明。
- 影响：`ios` 判据目前只有语义论证；若未来在 iOS 上编译或执行，仍需实测。
- 建议：在 NOT RUN／未取证清单中显式登记「`ios` 编译未取证（本机 cgo 工具链缺失）」，具备 macOS／cgo 环境后补验。

### IA-S4.4-R04-F003（INFO）本批次额外写出一个运行日志文件，超出「三文件 + 报告」清单

- 位置：`.spec/wip/identity-auth/_scratch/s44-r04-grouped.log`（16,124 B，15:27:05）
- 依据：`report-r04.md:218`、`:242` 已主动披露该日志；同目录 r01–r03 的同类日志为历史留存，`probe-r03.txt` 已确认删除。
- 影响：不是代码、不是测试数据、不是交付证据；不影响仓库构建与测试，但严格按「仅三文件 + report-r04.md」口径属清单外写入。
- 建议：由 Lead 决定保留或按清理清单另行授权清理；本评审不做删除。

### IA-S4.4-R04-F004（INFO）`verify-only` 负例按 fixture 下标取 key，未按下标无关的状态选取

- 位置：`identity_recovery_test.go:261-263`
- 依据：`verifyOnlyOnly := []recovery.Key{keys[1], keys[2]}` 依赖 `keyring-fixtures.json` 的键顺序（当前为 v3 active、v2／v1 verify_only，本轮已核对）。若 fixture 顺序调整，负例对象会变化；失败方式是 `NewKeyring` 成功导致 Fatal，属「响亮失败」，不会产生假通过。
- 影响：可维护性风险，不影响本轮结论。
- 建议：按 `State == verify_only` 过滤构造负例，或复用 `fixtures_test.go:410-438` 的状态校验结果。
- 备注：`identity_recovery_test.go:263` 失败消息中的 `verify-only-only` 笔误可顺手统一，非缺陷。

### IA-S4.4-R04-F005（INFO）`MarkConsumed` 顺序与软件 journal 指纹的「实质断言」实为替身自证，表格未逐项标注

- 位置：`report-r04.md:151`（§5 第 6 行）、`:177`、`:185`（§7 第 2／10 项）
- 依据：`MarkConsumedDuplicateAndConcurrent` 与 `SoftwareJournalSecretFingerprintAndStages` 直接调用 `softwareJournal`（`runtime_fixtures_test.go:251-345`，硬编码测试 key），断言的是替身自身语义；`report-r04.md:104` 已声明该替身，但 §5／§7 的这两行未再标注。
- 影响：不影响「未把替身写成生产」的总体口径（§4.2、§11 已披露），仅表格可读性；这是 r01–r03 沿用口径，不是本轮新增过度声明。
- 建议：在这两行追加「softwareJournal 替身」字样，避免读者把「已断言」理解为生产 `FileRecoveryJournal` 的 `MarkConsumed` 排序/并发语义已验证。

## 7. 覆盖真实性与计数核对

- 计数：实测 16 个子测试 = 5（identity）+ 7（joint）+ 4（journal_faults），与 `report-r04.md` §7 的「11 项实质断言 + 5 项部分/未取证」逐项对得上（实质 11 = identity 2 + joint 7 + journal_faults 2；部分/未取证 5 = identity 3 + journal_faults 2）。
- 断言强度未被削弱（抽查全部关键不变量）：
  - 并发：`journal_faults_test.go:44-71` 精确断言 `successes=1`、`mismatches=15`（不是「至少 1 成功」），`:72-77` 断言 `consumedMarkCount()==19` 与 caller 快照 19 条（含被拒调用），`:83-111` 断言 4 条 entry、sequence 连续、链哈希逐条衔接、每个 run 恰好一个 begun + 一个 consumed。
  - 错误分类：`ErrRunNotFound`（`:25`）、`ErrRunAlreadyExists`（`:31`、`identity_recovery_test.go:192`）、`ErrJournalMismatch`（`:37`）、`ErrAdmissionIncomplete`（`:170`、`:194`、`:204`、`identity_recovery_test.go:203`）、`ErrAdmissionBinding`（`:182`、`identity_recovery_test.go:200`）、`ErrAdmissionDrift`（`joint_readiness_test.go:336`）、`ErrSecretReused`（`identity_recovery_test.go:249`）、`ErrJournalCorrupt`（`journal_faults_test.go:250`）、`ErrInvalidKeyring`（`identity_recovery_test.go:171`、`:262`）、`ErrAdmissionClosed`（`identity_recovery_test.go:305`）均逐条断言。
  - 拒绝后的运行态：`s44AssertClosedProofUnavailable`（`joint_readiness_test.go:257-265`）同时断言 `state=closed`、`reason=proof_unavailable` 且 error 非 nil，四个篡改分支与命名空间/漂移用例都调用它。
  - `kept_deleted` 正向四项：`joint_readiness_test.go:402-426` 用真实 SQL 断言 `journal_stage=consumed`、`journal_consumed_at IS NOT NULL`、review 行数=1、`credential_reset_at` 非空数=0；本轮 `-v` 实测 `consumed / true / 1 / 0`。零 permit 负向 `:450-478` 仍在，断言 `platform.internal_error`、`status=active`、`reviews=0`、`audits=0`、`consumed_marks=0`。正负两用并存，没有互相挤掉。
- `joint_readiness_test.go` 本轮哈希与 r03 相同（`885766…C3F`），其 7 个子测试与上述 11 项中的 7 项实质断言全部保持。
- 未删除断言的证据：r03 作者分组日志 `s44-r03-grouped.log` 中 identity 的日志行号 `164／199／215／259／321` 与 r04 代码 `175／210／226／270／332` 呈**统一 +11** 偏移；journal_faults 的 `112／149／150／207／221` 与 r04 **完全相同**。说明 identity 只在前部新增注释、journal_faults 只改字符串，断言行未被删改（无 r03 文件快照，无法做字节级 diff，此为按行号对位的推断证据，已如实标注）。
- 与既有 8 个测试文件的关系：本批次未修改它们（mtime 与 `git diff` 均无写入），也未发现重复替代；`kept_deleted` 的 journal 四项断言在既有 `runtime_recovery_test.go` 中并不存在（`rg` 无 `journal_consumed_at`／`journal_stage` 命中），因此 r03 恢复的正向用例是必要覆盖，不能删。
- requires 口径：`report-r04.md:23-24`、`:210`、`:247` 明确 `waived-for-test-scope` 不表示 S4.5／S4.6 完成、不改 `tasks.md:649` 原文；未出现「requires 已满足」的表述。

## 8. NOT RUN / unknown 边界（本轮独立确认）

| 项目 | 状态 |
|---|---|
| POSIX 分支四条（跨版本重放、verify-only 读取、构造器校验、partial trailing write） | `NOT RUN`：本轮运行平台 `runtime.GOOS=windows`，代码已实现真实断言且 `GOOS=linux/android/illumos/darwin/windows go vet` 通过，但未实跑 |
| 真实进程崩溃／事务中断后的持久卷重放与 fsync 证据（I-REC-10） | `NOT RUN`；`journal_faults_test.go:150` 自述受控软件 journal 不能证明 fsync／落盘重放 |
| 文件 journal HMAC/链损坏、重复 sequence、只读、ENOSPC、fsync 失败（I-REC-04） | `NOT RUN` |
| 生产 Anchor（S4.5）、生产 provider 依赖断言（I-REC-01 等） | `NOT RUN` |
| 生产 key retirement／verify-only 真实双前置 evidence（S4.6，I-REC-03） | `NOT RUN`，S4.6 未实施 |
| I-REC-06／I-REC-07 | 未断言 / `NOT RUN`（不在三文件范围） |
| 独立故障域 | `unknown` / 未验证 |
| `ios` 编译与运行 | 未取证（`R04-F002`） |
| 非 POSIX 平台运行行为（Windows 仅构造器级 fail-closed） | 已断言构造器分类，其余 `NOT RUN` |

## 9. 测试卫生与越界

- 无 `t.Skip`（DSN 缺失一律 `t.Fatal`，符合 S4.17 约定）；无 `t.Parallel`，不存在共享 schema 竞争。
- 无 `DROP SCHEMA`／`DROP DATABASE`／`DROP TABLE`；schema 固定 `han_s44_<hex>`，仅 `t.Cleanup` 关闭连接池；不清理他人 schema，未写 S4.17 registry，残留为已知资源项（延续 r03 条件，不新增缺陷）。
- 未回显 DSN／口令：`report-r04.md:32` 只写存在性与长度，代码侧仅 `os.Getenv`／`LookupEnv`。
- 越界：本批次仅 `identity_recovery_test.go`、`journal_faults_test.go` 被改写（`joint_readiness_test.go` 哈希与 r03 相同），外加 `report-r04.md` 与一个已在报告中披露的运行日志（`R04-F003`）；`internal/`、`cmd/`、`migrations/`、`tasks.md`、设计文档、ADR 在本批次窗口（15:08–15:28）内无写入。窗口外的其他 `M`／`??` 项属其他工作流与历史状态，非本任务写入。
- Git：`test/integration/recovery/` 整体未跟踪，`git diff --stat` 为空；本评审未执行任何 Git 写操作。

## 10. 未验证风险

- POSIX 分支从未在任何平台上执行过：可编译不等于语义正确（例如 `FileRecoveryJournal.Read` 对 partial trailing write 的 `ErrJournalCorrupt` 分类、`OpenVerifiedRecoveryJournal` 的 verify-only 读取与 `Begin=ErrAdmissionClosed`）仍需实跑确认。
- 真实崩溃／fsync／ENOSPC 属持久卷故障域，Windows + 软件替身无法替代。
- `ios` 判据缺编译证据；`android`／`illumos` 仅编译证据，未运行。
- `han_s44_<hex>` schema 持续累积，清理需另行授权。
- 本轮结论只在 Windows + PostgreSQL 18.6 环境成立，不能外推到 Linux 持久卷、其他故障域或生产。

## 11. 复审条件

1. `R04-F001` 为措辞级修订（把 `Store` 去掉即可），不阻断本轮 PASS WITH CONDITIONS。
2. 具备 Linux Go 工具链后实跑四条 POSIX 分支并保存 `-v` 输出；在此之前 POSIX 行为保持 `NOT RUN`。
3. 真实持久卷补做崩溃／损坏／fsync／ENOSPC 注入（I-REC-04／10）。
4. S4.5 生产 Anchor、S4.6 生产 key retirement 未实施；`waived-for-test-scope` 仅为 test scope。
5. 独立故障域证据仍 `unknown`，不得用本轮结果替代。
6. 任务卡关闭条件（`tasks.md:652`）未满足，任务不得关闭。

## 12. 声明

- 本报告为独立评审结论，不代替 Lead／用户对结果的接受、任务关闭授权、风险接受或后续生产授权。
- 本轮只读取被评审文件与依据，只新增本报告；未修改任何被评审文件、r01–r04 证据或 r01／r02／r03 评审报告，未执行任何 Git 写操作。
- 报告中的通过结论仅指第 3 节实际执行的命令结果；`NOT RUN` 与 `unknown` 范围未被写成通过。