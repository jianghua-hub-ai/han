# identity-auth-S4.20 独立 Planning Review R01

- 评审角色：add-reviewer（非作者）
- 评审类型：Planning Review（非代码评审）
- 评审对象：`identity-auth-S4.20` 任务卡、B13H 批次段及其 ADR/设计/测试依据
- 评审基线：输入声明 `tasks.md` SHA256 `ACF3A73A671FE02DC15F31F502145406C0E2B6E6AE8F4213859DEB4182CCB922`（699495 bytes）；评审期间实测已变为 `4B56A026DC25D70C619D92BD473991CA8CD5B493B4604A838D07F0FCF30FC1E7`（701371 bytes）
- 总结论：**CHANGES REQUIRED**
- 评审主体：AI 代理；actual Agent ID **unknown / 本会话**；非真人签字
- 验收命令：**全部 NOT RUN**。本报告未声称任何代码、测试、build、PG、HTTP 或 E2E 已通过。

## 1. 固定对象与哈希

### 1.1 文档对象

| 对象 | 实测 | 结论 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | 输入基线 `ACF3A73A…` / 699495 bytes；评审期间实测 `4B56A026…` / 701371 bytes，mtime `2026-10-31T09:02:30.4188935Z` | **评审期间发生漂移**；非 Reviewer 写入。已重读当前 S4.20 卡（`tasks.md:2856-2919`）、B13H（Plan `:1587-1602`）、S1.5 状态（`tasks.md:1226-1231`）并重锚行号；未对整份 `tasks.md` 的其他无关变更做全量审计 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `1AB1474D1B330184D5FD80E998410FFDA5B3C1A1FF0B0B7BFBAB2450F498D268` / 487344 bytes | 与输入一致 |
| `.add/project/knowledge/decisions/framework.md` | `E19A4189BED343FABD1D996F5ECD4229373FD34C487315D1082C2F841FA70983` / 121274 bytes | 与输入一致；ADR-0027/0028 依据可定位 |
| `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-lead-narrowing-decision-r01.md` | `476FC625E52E79D33B8803BA7887A210BBD0F29151CA498DA9F5EE7533F9281F` / 3801 bytes | 可定位 |
| `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-world-hold-r01/README.md` | `C72B5D2FF32A861F5B676FD2263045B21A162182C4456B3AB4D25964624866A4` / 2293 bytes | 可定位 |
| `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r03/report.md` | `4BB0C821174A7D5839FEC4952175F3454A19BB33EDD50FFCAE76572793E5DB8F` / 15833 bytes | 可定位 |
| `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-lead-gate-r03/report.md` | `2CC765A9DB7289B0A174900B22624A11CB1C80C7C6BE74D6113CE0C3433208C3` / 3979 bytes | 可定位 |

### 1.2 代码与工作树事实

| 对象 | 实测 |
|---|---|
| `cmd/han-server/main.go` | SHA256 `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` / 171448 bytes，mtime `2026-10-31T08:36:42.7242751Z` |
| `cmd/han-server/recovery_runtime.go` | SHA256 `47CCE68A33DB57AD2414AE9D26FB6FD79459D3D96C415F02F61FAB75DA7216B2` / 33806 bytes，mtime `2026-10-30T19:22:10.0976143Z` |
| `cmd/han-server/recovery_runtime_integration_test.go` | 31638 bytes，mtime `2026-10-30T19:30:10.3254380Z` |
| Git 工作树 | `git status --porcelain` 共 284 行；`tasks.md` 为 ` M`；`cmd/` 整体为 `?? cmd/`；`git ls-files cmd/han-server` 返回 0 个文件 |
| S4.19 Gate 记录的 `main.go` | `B012EAE8814BD47B16797A277AAB2E0CDA7C1885C839A8EE9D5435D63E3F03B8` / 171424 bytes（`s419-lead-gate-r03/report.md:15`） |

`cmd/` 未被 Git 跟踪，意味着 `main.go` 与 `recovery_runtime.go` **没有普通 commit diff 基线**；只能以 SHA256、字节数和全文快照核验。

## 2. 实际 Read set

已读取或定向核对：

- `.codex/agents/add-reviewer.toml`（完整）
- `.codex/skills/add-code-review/SKILL.md`（完整）
- `.codex/guidance/task-execution-workflow.md` 的 §4（授权/G0）、§7（评审）、§10（三态与关闭）
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`、`.codex/guidance/no-hardcode-policy.md`（仅实施约束判断所需部分）
- `tasks.md` 当前 S4.20 卡 `:2856-2919`、B13H Plan `:1587-1602`、批次映射 `:1309`/`:1331-1332`、S1.5 `:1226-1231`、S4.16 `:2643-2680`
- `decisions/framework.md` `:843-852`（ADR-0027/0028）
- `design/architecture.md:1375-1391`；`design-workspace/identity-auth/design.md:3568`、`:4430`；`design-workspace/article-management/design.md:2430`、`:3051`；identity test-plan `:759`；article test-plan `:732`
- `cmd/han-server/recovery_runtime.go`、`cmd/han-server/main.go`、`cmd/han-server/recovery_runtime_integration_test.go` 的相关行段
- `internal/recovery/coordinator.go`、`internal/recovery/runtime.go`、`internal/article/deletion_replay.go`、`internal/article/replay_publication.go`、`internal/identity/recovery_execution.go`、`internal/identity/article_port.go`、`internal/article/command_service.go` 的相关行段
- `test/integration/recovery/runtime_recovery_test.go:605`、`runtime_fixtures_test.go:104-106`、`article_replay_test.go:172-174`
- S4.19 收窄决定、world hold README、独立复审 R03、Lead Gate R03

设计文件只用于核对卡内声明是否有可定位依据，**未重审设计本身**。

## 3. 评审方法

- Round 1：任务卡、B13H、ADR-0027/0028、architecture §9.1.1、identity/article 设计与测试计划的直接一致性。
- Round 2：真实消费者与代码事实、4 文件写集是否足够、测试函数命名、关闭条件、依赖/串行写窗、S4.19 基线漂移、S1.5 `main.go` 共享写窗。
- 发现 `tasks.md` 在评审期间由 699495 bytes 变为 701371 bytes 后，做了一次有界重读：重新锚定 S4.20 卡、B13H、S1.5 状态；未把旧行号直接套到新修订。
- 未运行任何项目命令；所有结论是静态规划审查结论。

## 4. 逐项核验

### 4.1 R 对账逐条结果（18 项）

| # | 卡内引用 | 实际核验 | 结果 |
|---|---|---|---|
| 1 | `cmd/han-server/recovery_runtime.go:457` | `newRecoveryRootAssembly` 构造器存在 | 存在且语义相符 |
| 2 | `recovery_runtime.go:469` | `identity.NewRecoveryRunStore` 调用存在 | 存在且语义相符 |
| 3 | `recovery_runtime.go:596` | `assembly.runtime = runtimeValue` 存在 | 存在且语义相符 |
| 4 | `cmd/han-server/main.go:2418` | production dispatcher 接线存在 | 存在且语义相符 |
| 5 | `main.go:3648` | `recoveryGuardFromRuntimeHandle` + `guard.Protect` 存在 | 存在且语义相符 |
| 6 | `internal/recovery/coordinator.go:133` | `Coordinator.Consume` 存在 | 存在且语义相符 |
| 7 | `coordinator.go:274` | `markAndRead` 调用 `MarkConsumed` 并 `ReadVerifiedSnapshot` | 存在且语义相符 |
| 8 | `internal/recovery/runtime.go:1538` | `Resume` 存在 | 存在且语义相符 |
| 9 | `runtime.go:1895` | `replay` 存在 | 存在且语义相符 |
| 10 | `runtime.go:1917` | 该行实际是 `ExitRecovery`，不是 `activate`；`activate` 在 `:917`，调用在 `:1943` | **行存在但标签不符**（F007） |
| 11 | `internal/article/deletion_replay.go:26` | `NewDeletionReplayProvider` 存在 | 存在且语义相符 |
| 12 | `deletion_replay.go:40` | `ReplayAndVerify` 存在 | 存在且语义相符 |
| 13 | `deletion_replay.go:244` | `ReadStatus` 存在 | 存在且语义相符 |
| 14 | `internal/article/replay_publication.go:83` | `NewReplayPublicationPorts` 存在 | 存在且语义相符 |
| 15 | `internal/identity/recovery_execution.go:41` | `RecoveryIdentityExecutionDependencies` 存在 | 存在且语义相符 |
| 16 | `recovery_execution.go:65` | `NewRecoveryIdentityExecution` 存在 | 存在且语义相符 |
| 17 | `internal/identity/article_port.go:17` | `ArticleAccountPort` 存在 | 存在且语义相符 |
| 18 | `internal/article/command_service.go:342` | `OrphanArticlesCanonical` 存在 | 存在且语义相符 |

结论：18/18 引用行存在；17/18 语义相符；`runtime.go:1917` 是标签错误，不是代码缺失。

### 4.2 验证项 1–6

| 验证项 | 结论 | 依据 |
|---|---|---|
| 1. R 对账的 file:line 是否真实且语义相符 | **PASS（静态）/ 17 of 18** | 18 个显式引用全部存在；`runtime.go:1917` 标签不符（F007） |
| 2. 4 文件写集是否足以达成关闭条件；同名测试如何处理 | **FAIL / PARTIAL** | F003：canonical replay oracle 未定义且现有测试标 `unproven_port`；F004：同名顶层测试与 S4.16 已关闭验收冲突 |
| 3. 关闭条件是否可验证、是否符合 ADR-0027/0028 与 architecture §9.1.1 | **FAIL / PARTIAL** | F001：完整恢复范围未覆盖 enter/install/secret 关闭点；F005：缺 `go build`；F003：缺可执行 oracle |
| 4. 依赖与串行条件是否完整，`main.go` 与 S1.5 写窗是否被覆盖 | **FAIL / PARTIAL** | F002：只有泛化 writer/写窗检查，无 S1.5 ACL R02 显式门禁；F008：S4.19 关闭基线 `main.go` SHA 已漂移 |
| 5. 设计仍 draft、人的精确接入 unknown 是否导致不可实施 | **PARTIAL / UNKNOWN** | 卡面已登记 draft 与人的精确接纳 unknown；ADR-0028 允许 AI 代理，但不等于真人签字，也不使 draft 自动成为已确认合同。未发现仅因此就应单独阻断的代码缺陷，但相关范围必须保留 unknown |
| 6. `-race`/integration/PG 命令是否与仓库既有形态一致 | **PARTIAL** | race/integration/PG 的形态与 S4.19 一致；但缺 `go build`；现有 PG 测试在缺 `HAN_TEST_POSTGRES_DSN` 时直接 `Fatal` 而非 `Skip`，执行前置必须记录；全部命令 NOT RUN |

## 5. Findings

### S420-PLAN-R01-F001 — HIGH

**标题：完整恢复范围与关闭入口/secret 关闭点不一致。**

**证据**

- `tasks.md:2858`、`:2860` 声称打通“完整恢复→成功开流”，`tasks.md:2904` 的 `CompleteCoordinatorRecoveryAndOpen` 也要求完整恢复。
- `tasks.md:2866` 的“待做”只列出替换 `recovery_runtime.go:498/531/538/539/545/558` 的 closed 占位，没有列出 `:465` 与 `:523`。
- `cmd/han-server/recovery_runtime.go:462-465`：`enter`/`install` 直接返回 `ErrAdmissionClosed`。
- `recovery_runtime.go:522-524`：recovery secret candidate source 直接返回 `ErrAdmissionClosed`。
- `recovery_runtime.go:758-777`：`start` 只支持 `normal` 与 `resume`。
- `recovery_runtime.go:831-840`：`joinWork` 只支持 `normal` 与 `resume`。
- `internal/recovery/runtime.go:1828-1831`、`:1857`：resume 在 `InitialAdminBootstrapGeneration == nil` 时仍可能读取 recovery secret source。

**触发场景**

实施者按卡面“待做”只替换列出的占位，并让 resume 路径通过测试；卡面仍可被解释为“完整恢复”，但 `enter`/`install` 和 secret candidate source 仍是 closed。测试可能全部通过，关闭条件却只证明了 resume 子路径。

**影响**

卡的范围、写集与关闭条件不一致。若“完整恢复”必须包含 enter/install/secret，则 4 文件写集和“待做”不完整，关闭条件不可成立；若本卡只做 resume，则卡面必须明确收窄，不能继续使用“完整恢复”措辞。

**建议**

由 Lead 在 START 前二选一并写入卡面：

1. **收窄为 resume-only**：修改目标/leaf 措辞，明确排除 `recovery_runtime.go:465`、`:523`，并说明 enter/install/secret 由哪个后续任务承接。
2. **保留完整恢复**：将 enter/install、secret candidate、相关 S4.13/S4.18 能力纳入范围，扩大写集/依赖并重写关闭条件。

在决策前不得以“完整恢复”名义关闭。

### S420-PLAN-R01-F002 — HIGH

**标题：`main.go` 与 S1.5 ACL 修复共享写窗没有显式门禁。**

**证据**

- `tasks.md:1226`：`article-management-S1.5` 仍为 `[-]`。
- `tasks.md:1230`：S1.5 ACL 独立评审 R01 为 `CHANGES REQUIRED`，含 3 个 HIGH。
- `tasks.md:1231`：ACL R02 已交付且 Lead 现场复核，但明确“待独立复审”；在 R02 PASS 与 Lead Gate 前，ACL 子集不构成可交付证据。
- `tasks.md:2916`：S4.20 只有泛化的“实施前 Main 须重核实际 writer/写窗，未知即停”，没有点名 S1.5 ACL R02、独立复审或 Lead Gate。
- `tasks.md:1231` 记录的 `main.go` 为 `1d47cc94…`；当前 `main.go` 实测为 `1D47CC94E415E80C…` / 171448 bytes。
- `.spec/wip/article-management/evidence-s1.5/acl-r02/index.json` 的 mtime 为 `2026-10-31T08:58:02.4403006Z`，晚于 `main.go` 的 `2026-10-31T08:36:42.7242751Z`；没有证据证明 S1.5 writer 已释放写窗（unknown）。

**触发场景**

S4.20 在 S1.5 ACL R02 独立复审尚未 PASS 时启动；两个任务都修改 `cmd/han-server/main.go`。若 S1.5 复审要求继续修复，S4.20 的根接线会覆盖或混淆 S1.5 的修复结果；若 S4.20 先改，S1.5 的 R02 证据也会失效。

**影响**

共享写窗可能造成覆盖、回归或双方证据同时失效。泛化的“未知即停”依赖 Main 自行判断，不能保证在 S1.5 尚未闭合时阻止启动。

**建议**

在 S4.20 卡增加显式门禁：

`S1.5 ACL R02 非作者独立复审 PASS + Lead Gate PASS + S1.5 writer 明确 STOP/写窗冻结 + 当前 main.go SHA256 记录` 后才允许 S4.20 START；任一不满足即 STOP。实施前、实施后都要记录 `main.go` 的 SHA256、字节数、mtime 与 `git status`。

### S420-PLAN-R01-F003 — HIGH

**标题：`canonical_recovery_replay_same_result` 缺少可执行 oracle，4 文件充分性未证明。**

**证据**

- `tasks.md:2866` 要求证明“两次 replay 结果相等/零额外写入，不得只查日志字符串”。
- `tasks.md:2906` 的 `CanonicalReplaySameResult` 要求 P 层证据。
- `tasks.md:2912` 将其列入关闭条件。
- `test/integration/recovery/runtime_recovery_test.go:605` 明确记录：`unproven_port=canonical_recovery_replay_same_result; reason=... same live permit plus RecoveryRuns/RecoveryGate wiring inside identity_execution ...`。
- `tasks.md:2877` 默认不修改 `internal/identity/*` 或 `internal/article/*`；若需要则停止并回 Main 扩写集。

**触发场景**

实施者在 `cmd/han-server` 层构造两次 replay，但只比较日志、调用次数或 fake 返回值；或依赖 identity execution 内部的 RecoveryRuns/RecoveryGate wiring，却因 4 文件限制无法写入该层。测试名称和日志可能显示通过，实际没有证明持久结果相等或零额外写入。

**影响**

关闭条件不可验证；4 文件写集是否足够处于 unknown。若 oracle 必须触及 `internal/identity`，当前卡不能合法完成。

**建议**

在实施前把 oracle 写成可执行条款：

1. 明确比较哪些持久结果字段（run/outcome、journal entry、article evidence ref/digest、snapshot ref/digest、completed_at、consumed 标记等）。
2. 明确第二次 replay 的触发方式（同 run、同 live permit、同 binding）。
3. 明确“零额外写入”的观测方式（事务计数、row version、insert/update 计数或等价可审计证据）。
4. 若必须依赖 identity execution 内部 wiring，先由 Lead 明确扩写集；否则收窄该 leaf，不得用日志字符串或 fake world 代替。

### S420-PLAN-R01-F004 — HIGH

**标题：同名 `TestRecoveryRuntimeProductionChain` 与 S4.16 已关闭验收直接冲突。**

**证据**

- `cmd/han-server/recovery_runtime_integration_test.go:1` 为 `//go:build integration`。
- 同文件 `:165` 已有顶层 `func TestRecoveryRuntimeProductionChain(t *testing.T)`；`:171-172` 使用真实 B13H root/backup DSN 占位；`:181` 使用 `recoveryTestRuntime` fake。
- `tasks.md:2662` 将该文件列入已关闭 S4.16 的写集。
- `tasks.md:2678-2680` 的 S4.16 验收命令精确匹配 `^TestRecoveryRuntimeProductionChain$`。
- `tasks.md:2875-2876` 要求修改旧文件，并在新文件 `recovery_production_chain_integration_test.go` 使用同名顶层 `TestRecoveryRuntimeProductionChain`。
- `tasks.md:2888-2890` 的 S4.20 验收命令也精确匹配同名。

**触发场景**

新文件加入同 package 后，两个同名顶层函数无法编译。若把旧测试改名，S4.16 已关闭验收的精确符号和语义发生变化，可能构成重开；若只把新测试改名而保留 S4.20 的旧命令，命令会命中旧 fake 测试，不能证明完整链。

**影响**

当前卡无法在不作 Lead 决策的情况下实现和验收。编译失败会阻断整个 `cmd/han-server` integration package；改名旧测试则会触碰已关闭 S4.16 的验收证据。

**建议**

采用不重开 S4.16 的方案：

- 新完整链测试使用不同顶层名，例如 `TestRecoveryProductionChainComplete`；
- 同步修改 S4.20 的 `-list`/`-run` 命令与关闭条件；
- 保持旧 `recovery_runtime_integration_test.go:165` 不动，不改变 S4.16 的精确验收符号。

若 Lead 认为必须改旧测试名，应作为独立决定记录，不能让 S4.20 自行绕过 S4.16 的关闭记录。

### S420-PLAN-R01-F005 — MEDIUM

**标题：关闭条件缺 `go build`，且未跟踪文件没有普通 Git diff 基线。**

**证据**

- `tasks.md:2888-2897` 的验收命令只有 `go test`、`go vet`、`gofmt -l`，没有 `go build`。
- `tasks.md:2912` 的关闭条件只列 `go vet` 与 changed-file `gofmt -l`。
- `.add/project/knowledge/decisions/framework.md:846`：ADR-0028 将 `go build`、`go vet`、`gofmt`、单元/组件/契约/集成测试列为必须实际执行的可执行层。
- `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-lead-gate-r03/report.md:34` 实际执行了 `go build ./...`。
- `git status --porcelain` 显示 `?? cmd/`；`git ls-files cmd/han-server` 为 0，`main.go` 与 `recovery_runtime.go` 没有普通 commit diff 基线。

**触发场景**

实施者用 `go test`/`go vet`/`gofmt` 通过后关闭任务，但没有运行 ADR 要求的 `go build ./...`；Lead 也无法用普通 Git diff 审核未跟踪文件的实际改动，只能依赖 SHA/全文快照。

**影响**

ADR-0028 可执行层不完整；构建失败或未审计改动可能在关闭后才暴露。

**建议**

在 S4.20 验收命令与关闭条件中加入 `go build ./...`；明确未跟踪文件以 SHA256、字节数和全文快照核验，不能把“没有 diff”当作“没有改动”。

### S420-PLAN-R01-F006 — LOW

**标题：S1.5 证据尚未通过却被表述为可消费。**

**证据**

- `tasks.md:2862`、`:2881` 将 `article-management-S1.5` 列为消费证据。
- `tasks.md:1226` 的 S1.5 仍为 `[-]`。
- `tasks.md:1230`、`:1231` 明确 ACL 子集在 R02 PASS 与 Lead Gate 前不构成可交付证据；R02 独立复审仍待完成。

**触发场景**

实施者读到“消费 S1.5 证据”后，假定 ACL 子集已经成立，或把 Lead 的 R02 现场复核当作已通过。

**影响**

依赖表述误导，可能导致错误的前置假设。由于卡面同时写了“仅消费证据、不作硬 requires”，当前不构成硬阻断。

**建议**

改为“未来关联/条件性引用”：只有在 S1.5 ACL R02 非作者独立复审 PASS 且 Lead Gate PASS 后，才可把其证据列为可消费输入。

### S420-PLAN-R01-F007 — LOW

**标题：R 对账 `runtime.go:1917` 标签与代码不符。**

**证据**

- `tasks.md:2865` 将 `runtime.go:1538/1895/1917` 对应为 `Resume`/`replay`/`activate`。
- 实际 `internal/recovery/runtime.go:1917` 是 `ExitRecovery`。
- `activate` 在 `runtime.go:917`，调用在 `:1943`。

**触发场景**

实施者按卡面把 `:1917` 当作 activate 的 leaf 证据，可能漏掉真正的 `:1943` 调用点或误解 exit 顺序。

**影响**

证据追溯不精确；不影响当前代码行为，但降低 Planning Review 的可核验性。

**建议**

将 `:1917` 标注为 `ExitRecovery`；若需引用 activate，使用 `:917` 或 `:1943`。

### S420-PLAN-R01-F008 — MEDIUM

**标题：S4.19 关闭基线 `main.go` SHA 已漂移，S4.20 未明确记录重验义务。**

**证据**

- `s419-lead-gate-r03/report.md:15` 记录 S4.19 关闭时 `main.go` 为 `B012EAE…` / 171424 bytes。
- 当前 `main.go` 为 `1D47CC94…` / 171448 bytes；S1.5 ACL R02 已在 `main.go` 上产生修改。
- `tasks.md:2862`、`:2865`、`:2881` 将 S4.19 的 production dispatcher/guard/同 handle 作为已闭依据。
- `tasks.md:2891-2894` 虽包含 `TestRecoveryRuntimePostgres` 与 `TestRecoveryMainBusinessDispatcher`，但卡面未明确这是对**当前 SHA** 的 S4.19 回归重验，也未把“`main.go` 再次变化即 STOP”写成显式条件。

**触发场景**

S4.20 实施者假定 S4.19 的 guard/probe 行为仍成立，直接从当前 `main.go` 开始改；若 S1.5 或其他人再次修改 `main.go`，S4.19 的关闭证据不再对应当前对象。

**影响**

可能继承一个未复验的 root 接线基线；现有命令虽可部分缓解，但没有绑定到当前 SHA 与变更停止条件。

**建议**

S4.20 START 前固定当前 `main.go` SHA；关闭条件明确要求在当前 SHA 上重跑 S4.19 的 dispatcher/probe 两层回归并记录；任何 `main.go` 的进一步变化都触发 STOP/重核，而不是继续沿用旧 Gate。

## 6. 总交接结论

**CHANGES REQUIRED**

阻断项：`S420-PLAN-R01-F001`、`F002`、`F003`、`F004`（HIGH）。

需补正/接受：`F005`、`F008`（MEDIUM）。

非阻断但应修正：`F006`、`F007`（LOW）。

这不是“代码已失败”的结论，而是**任务卡当前不可实施或关闭条件不自洽**的规划结论。任何代码、测试、build、PG、HTTP 或 E2E 均未运行，不能据此声称实现通过。

### 需 Lead 决定的问题

1. S1.5 ACL R02 的独立复审与 Lead Gate 是否作为 S4.20 START 的硬门禁。
2. 新完整链测试是否改用不同顶层名，并同步修改 S4.20 命令，保持 S4.16 旧测试不动。
3. `canonical_recovery_replay_same_result` 是否能在 4 文件内取得可执行 oracle；若不能，是扩写集还是收窄 leaf。
4. `enter`/`install`/secret candidate 是否纳入 S4.20；若不纳入，必须把卡面“完整恢复”收窄为 resume-only。
5. 是否补 `go build ./...`，并明确未跟踪文件的 SHA/全文核验方式。
6. 是否在当前 `main.go` SHA 上重验 S4.19 dispatcher/probe 两层，并记录为新基线。

## 7. 未验证边界与 unknown

- **S1.5 ACL writer 是否释放写窗：unknown。** `acl-r02` 最新写入晚于 `main.go` mtime，不能据此认定 writer 已停止；S1.5 R02 独立复审仍未完成。
- **4 文件是否足以完成 `canonical_recovery_replay_same_result`：unknown。** 现有测试明确标记 `unproven_port`。
- **`recovery_runtime.go:465`/`:523` 是否属于 S4.20 范围：unknown。** 需 Lead 决定“enter/install/secret 纳入”还是“收窄为 resume-only”。
- **S4.16 的 `recovery_runtime.go` 关闭基线 SHA：unknown。** 输入未提供该基线，当前文件未跟踪。
- **S4.19 guard/probe 在当前 `main.go` SHA 下的行为：unknown。** 未运行，不能沿用旧 Gate 的对象一致性结论。
- **设计仍 draft，人的精确接纳：unknown。** ADR-0028 允许 AI 代理评审/签收，但不冒充真人签字，也不使 draft 自动成为已确认合同。
- **Reviewer actual Agent ID：unknown / 本会话。** Host 未返回独立 ID，未伪造。
- **输入 `tasks.md` 基线已漂移。** 输入声明 `ACF3…`/699495，当前实测 `4B56…`/701371；本报告已重读当前 S4.20/B13H/S1.5 段落并重锚行号，但未对整份文件的无关变更做全量审计。
- **所有验收命令：NOT RUN。** 未运行 Go、PG、HTTP、service、restore 或 E2E。
- **真实部署/E2E/POSIX/ACL/provider/备份：unknown。** 卡面本身也明确不证明这些范围。