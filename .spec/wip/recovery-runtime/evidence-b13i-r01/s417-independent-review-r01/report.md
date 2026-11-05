# identity-auth-S4.17 / B13I 独立评审报告（r01）

- review_id: `s417-independent-review-r01`
- 角色: add-reviewer（非作者独立评审；不修改被审对象）
- 工作区: `D:\AID\VSCode\Codex\Han`
- 分支 / HEAD: `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审方法: `.codex/skills/add-code-review/SKILL.md`（完整读取）+ `code-review-checklist.md` + `abstraction-principles.md`；按代码对象分支加载 `coding-standards` / `no-hardcode-policy`，未启用前端 profile。
- 对象类型: code（test-only 跨层软件专项）
- 写入范围: 仅 `.spec/wip/recovery-runtime/evidence-b13i-r01/s417-independent-review-r01/`

## 结论

**CHANGES REQUIRED**

7 条验收命令与 gofmt 在本机全部 exit 0，JSON 无 fail / 无 skip；但这些通过不能抵消以下阻断问题：多个关键判定（ABA/stale、consumed 窗口、maintenance gate、anchor 与 finalizer journal 的一致性）由测试内软件适配器直接给出，而不是由真实 runtime/store 产生；Leaf4 未接入真实的 identity recovery execution 链；Leaf3 的路由是手搭的，违反 article test-plan 明确禁止项。因此“全部命令通过”只能证明代码可编译、可运行、无数据竞争，不能证明任务卡要求的行为已被真实覆盖。

## 0. 对象与修订

| 文件 | SHA256 | 本机复核 |
|---|---|---|
| `test/integration/recovery/runtime_admission_test.go` | `AF87672131C50D6552C696F445C7C976B30173ABBE1DEDEC2F4151BC539BE158` | 匹配 |
| `test/integration/recovery/runtime_recovery_test.go` | `08F8589717B5E8620AA1CA3D3C96CEA007AA0B7B48B61B6AEA7969A2BA9F0064` | 匹配 |
| `test/integration/recovery/runtime_fixtures_test.go` | `DC504CF4170B2AF42A51BFD7FE01CA5EE00DA759D5539C886A15540DD8AF4621` | 匹配 |

上游依据：任务卡 S4.17（`tasks.md` 中 `<a id="task-identity-auth-s4-17"></a>` 至 `<a id="task-identity-auth-s4-18"></a>`）、identity test-plan §4.7.1、article test-plan §4.8.2 / Case18、D-B13I-01、作者证据 `author-r01/`、Lead Gate `lead-gate-r01/`。评审期间未发现被审三文件内容变化（评审前后 SHA 一致）。

## 1. 两轮实际检查范围

**第一轮（对象正确性 / 契约）**
- 完整阅读三个被审文件，逐 leaf 核对断言与任务卡最低事实。
- 逐 adapter 核对“是否只做介质读写”。
- 核对负路径：拒绝、回滚、stale、unknown、503、零下游、幂等重复。
- 检查 `t.Skip`、fake 冒充、跨 leaf 共享可变状态、schema 清理、越界修改。
- 对照生产代码：`internal/identity/recovery_execution.go`、`internal/identity/recovery_runtime.go`、`cmd/han-server/main.go`、`internal/recovery` 端口定义。

**第二轮（独立复现 / 反证）**
- 本机设置 DSN 环境变量后独立复跑全部 7 条验收命令 + gofmt（不采信他人日志）。
- 另行以 `-v` 复跑 Case18 子测试，核对两次 503、零下游、开流 200 的真实日志。
- 复核 Lead Gate 与作者证据是否存在缺口（未作为通过依据）。
- 沿真实消费者有界检查：`cmd/han-server` 生产链测试、`article.NewDeletionFinalizer`、`identity.NewRecoveryRunStore` 的实际装配点。

## 2. D-B13I-01 正当性判定

**前提成立，授权范围不成立。**

- 前提：“本机 Windows 下 internal/recovery 文件 journal / 文件锁为既定不支持契约”。该前提有真实依据：`filelock_unsupported.go` 与 `TestFileRecoveryJournalWindowsUnsupportedHasZeroExternalChanges` 确实是既有契约，不是本次为放行而临时编造。
- 因此，用受控软件 adapter 替代**文件与介质端口**（journal 介质、部署独占、anchor 介质、commitment 介质）在原则上可接受。
- 但 D-B13I-01 明确要求 PG 侧端口必须真实：`recovery.NewRuntime`、`identity.NewRecoveryRunStore`、`identity.NewRecoveryIdentityExecution` / `session.NewStore` + `platform.NewHTTPAuthorizer`、`article.NewDeletionFinalizer`。
- 实测：`identity.NewRecoveryIdentityExecution` 在三个被审文件中**完全没有出现**；Leaf4 直接驱动 `softwareJournal.Begin/MarkConsumed`；ABA/stale 由 `softwareArticleMedia.CompareAndSwap` 判定；maintenance gate 恒返回 nil。这三处已越过“只做介质读写”的授权边界，属于**替被测逻辑做决定**。
- 结论：D-B13I-01 可作为**前提**接受，但不能作为当前实现的**授权**。需要作者收窄 adapter 职责并接入真实 PG/identity 端口后再评审。
## 3. Findings（按严重度排序；15 open / 0 closed）

### HIGH

#### H01 — ABA / stale 判定被软件 adapter 取代，未由真实 runtime/journal 产生

- 位置：`runtime_fixtures_test.go:886-907`（`softwareArticleMedia.CompareAndSwap`）；`runtime_recovery_test.go:159-169`（Leaf2 直接调用）
- 违反依据：任务卡 S4.17 要求 ABA 回退判定由真实 runtime/store 产生；D-B13I-01 仅授权介质端口 adapter。
- 证据：`CompareAndSwap` 内部自行比较 `expectedRevision`、检查 `next.Committed.Sequence < m.head.Sequence` 并返回 `ErrAnchorStaleRevision` / `ErrAnchorRollback`；测试随后断言这些错误——等于断言自己实现的判定。
- 触发场景：真实 `recovery.NewRuntime` 的 anchor 提交/回退路径若实现错误，本测试仍会通过。
- 影响：ABA 与 stale 是 S4.17 的核心最低事实，现为自证，属阻断。
- 修复建议：保留介质 adapter，但把 ABA/stale 判定交给真实 runtime（真实 `NewRuntime` + 真实 anchor 端口语义），测试只观察 runtime 返回值。
- 状态：open

#### H02 — 缺少真实身份恢复执行链；consumed 窗口顺序由软件 journal 直接给出

- 位置：`runtime_recovery_test.go:426-457`（Leaf4 直接 `journal.Begin` / `MarkConsumed`）；`runtime_fixtures_test.go:259-302`（`softwareJournal.Begin/MarkConsumed`）；`runtime_recovery_test.go:1326-1335`（注入 stub gate）
- 违反依据：D-B13I-01 要求 `identity.NewRecoveryIdentityExecution` 为 PG 侧真实端口；任务卡要求 consumed 窗口顺序由真实 store 产生。
- 证据：三个被审文件中 `identity.NewRecoveryIdentityExecution` 零出现（本机 `rg` 复核）；测试对 `ErrRunAlreadyExists` / `ErrRunNotFound` / `ErrJournalMismatch` 的期望全部来自 `softwareJournal` 自身分支。
- 与作者声明矛盾：`author-r01/decisions.md` 声明“从不决定 consumed 窗口”，但代码中恰由 adapter 决定；该声明不成立。
- 影响：启动后只消耗一次、乱序拒绝、重复消费拒绝这些最低事实未被真实链证明。
- 修复建议：接入真实 `identity.NewRecoveryIdentityExecution`（PG），journal 介质 adapter 只提供字节读写。
- 状态：open

#### H03 — Case18 使用手搭路由，违反测试计划明确禁止项

- 位置：`runtime_fixtures_test.go:1599`（`world.router = platform.TraceMiddleware(...)`）、`:1603-1652`（`route` switch 手写 404/405/Allow）
- 违反依据：article test-plan §4.8.2 明令不得手搭 router、不得伪造 503。
- 证据：`route` 用 switch 复制了生产 exact-route 契约，`Allow` 值在 adapter 与断言中各自硬编码。
- 触发场景：生产路由注册若漏掉 `PUT`/`DELETE` 或返回不同 Allow，测试仍通过。
- 影响：Case18 的 405/Allow/404 事实为自证，阻断。
- 修复建议：使用生产 route 注册函数构造 handler，测试只发请求并观察。
- 状态：open

#### H04 — F002 maintenance gate 恒开，测不出“维护期外”语义

- 位置：`runtime_fixtures_test.go:1279-1283`（`s417SoftwareMaintenanceGate.CheckRecoveryOpenGate` 直接 `return nil`）；使用点 `:1334`；`runtime_recovery_test.go:332-354`
- 违反依据：真实 gate 位于 `internal/identity/recovery_execution.go:201`（`recoveryExecutionGate.CheckRecoveryOpenGate`），会依维护窗口返回错误；任务卡 F002 要求维护窗口外拒绝。
- 证据：`softwareMaintenanceGate` 无条件返回 nil，测试实际走的是“能力缺失”路径，但日志把该用例标为“outside maintenance”，标签与事实不符。
- 影响：F002 的维护窗口判定未被覆盖，且日志陈述误导。
- 修复建议：使用真实 gate 或可切换窗口的受控 gate，并修正日志标签。
- 状态：open

#### H05 — finalizer journal 与 runtime anchor 读取源不一致

- 位置：`runtime_fixtures_test.go:1125`（`Journal: w.deletionJournal`）、`:1105`（`ArticleAnchorReader: w.article`）；`runtime_recovery_test.go:73`（手工 `world.article.advance(...)`）
- 违反依据：生产装配 `cmd/han-server/main.go:3300` 与 `internal/identity/recovery_runtime.go:546` 中 anchor 读取与 journal 为同一权威源。
- 证据：测试中 finalizer 写 `w.deletionJournal`，runtime 却从 `w.article` 读 anchor，测试再用 `advance` 手工对齐二者；这掩盖了真实装配不一致的风险。
- 影响：genesis-deletion 历史重启的最低事实被人工桥接，阻断。
- 修复建议：让两个端口共享同一真实/受控介质实现，删除手工 `advance` 对齐。
- 状态：open

### MEDIUM

#### M01 — closure reader 恒返回未完成，没有正向 published closure

- 位置：`runtime_fixtures_test.go:521-540`（`softwareClosureReader.ReadPublishedClosure` 恒 `ErrAdmissionIncomplete`）
- 证据：Leaf2 的“fresh”仅是空历史，不是真实的已发布 closure 正向路径。
- 影响：fresh/stale 的判定只有一半被覆盖。
- 修复建议：补充正向已发布 closure 用例。
- 状态：open

#### M02 — 双命名空间独立性是两个硬编码 adapter 自比

- 位置：`runtime_recovery_test.go:140-155`
- 证据：比较对象是两个构造时即独立的 adapter 实例，独立性由构造方式保证，比较必然成立。
- 影响：该断言近乎恒真，不能证明真实命名空间隔离。
- 修复建议：改用真实介质或至少引入可失败的共享状态检查。
- 状态：open

#### M03 — 部署独占 adapter 不保证互斥，且无二次获取测试

- 位置：`runtime_fixtures_test.go:375-387`（`softwareExclusive.Acquire` 仅 `held++`）
- 影响：独占语义未被验证。
- 修复建议：实现真实互斥或补充二次获取应失败的负路径。
- 状态：open

#### M04 — lifecycle / installation writer 为计数桩，真实 Bind/join 未证

- 位置：`runtime_fixtures_test.go:448-456`（`StopAndJoin` 仅计数）、`:960-970`（`BindInstallation` 仅计数）、`:1028`、`:1299-1308`（安装行预置）
- 影响：安装绑定与停机汇合的真实行为未被证明。
- 修复建议：至少对真实 PG 安装绑定做一次正向断言。
- 状态：open

#### M05 — F002 零写入检查只覆盖 audit_events

- 位置：`runtime_recovery_test.go:341-353`
- 证据：仅断言 `audit_events` 未增长；未断言 ACK/原子回滚。F001 则在 `:251-262` 复查了 users/articles。
- 影响：零下游写入证明不完整。
- 修复建议：补充 recovery_runs / 文章表的零变更断言。
- 状态：open

#### M06 — Leaf1 无真实 Session，端口日志缺 session.NewStore

- 位置：`runtime_recovery_test.go:31`（`s417LogPorts` 未列 `session.NewStore`）；fixture 使用 `softwareSourceVerifier` 与预置安装行
- 影响：Leaf1 声称的“真实身份链”不完整。
- 修复建议：接入真实 `session.NewStore` + `platform.NewHTTPAuthorizer`。
- 状态：open

### LOW

#### L01 — 自动 DROP SCHEMA CASCADE 与任务卡“不得自动 DROP”冲突

- 位置：`runtime_fixtures_test.go:132-140`
- 影响：可能与保留/取证要求冲突；不阻断行为正确性。
- 状态：open

#### L02 — Lead Gate 证据包缺 cmd-04.log 与 gofmt.log

- 位置：`.spec/wip/recovery-runtime/evidence-b13i-r01/lead-gate-r01/`（`run-results.json` 引用 cmd-04，但文件缺失）
- 影响：交付完整性缺口；本次已用本机复跑补齐，故不单独阻断。
- 状态：open

#### L03 — 测试自造 oracle（ETag、Allow 硬编码）

- 位置：`runtime_fixtures_test.go:1690`（ETag 构造）、`:1612` / `:1620`（Allow 值）
- 影响：与 H03 同源，单独记为低风险。
- 状态：open

#### L04 — 开流未断言 trace 计数

- 位置：`runtime_recovery_test.go:658-662`
- 证据：`trace` 变量被读取但未进入 `if` 条件，仅在 Fatalf/Logf 中打印；实际日志显示 `trace=1`。
- 影响：追踪上下文覆盖未被断言。
- 状态：open
## 4. 5 个 leaf 逐项判定

| Leaf | 判定 | 关联 findings | 说明 |
|---|---|---|---|
| GenesisDeletionHistoricalRestart | CHANGES REQUIRED | H05, M04, M06 | 历史重启本身可运行，但 anchor 与 finalizer journal 由测试人工桥接，安装绑定/停机为计数桩，Leaf1 无真实 Session。 |
| CrossNamespaceABAAndExactPublication | CHANGES REQUIRED | H01, M01, M02 | ABA/stale 由 adapter 判定；closure 无正向 published 路径；命名空间独立性为自比。 |
| F001F002AndCanonicalIdempotencyReview | CHANGES REQUIRED | H02, H04, M05 | consumed/幂等由软件 journal 给出；maintenance gate 恒开；F002 零写入只查 audit_events。 |
| CrashConsumedWindowAndDrain | CHANGES REQUIRED | H02, M03, M04 | 未接真实 `identity.NewRecoveryIdentityExecution`；独占不互斥；停机汇合为计数。 |
| Case18SameRevisionAndNoProductionClaim | CHANGES REQUIRED | H03, L04 | 手搭 router；开流 trace 未断言。 |

补充：5 个 leaf 均确实执行了负路径（拒绝 / 回滚 / stale / unknown / 503 / 零下游）并打印了真实日志，未使用 `t.Skip`，未发现跨 leaf 共享可变状态（每个 leaf 构造独立 world；Leaf2 的 `advance` 只在本 leaf 内使用）。问题不在“有没有跑”，而在“断言由谁产生”。

## 5. 独立复跑与真实计数

环境（不打印 DSN 值）：`HAN_TEST_POSTGRES_DSN` / `HAN_B13H_ROOT_DSN` → `127.0.0.1:53519/han_b13h_root`（用户 han_b13h，trust）；`HAN_B13H_BACKUP_DSN` → `han_b13h_backup`。

命令与真实结果（本机执行，日志见本目录 `cmd-01.log` … `cmd-07.log`、`gofmt.log`、`cmd-08-case18-verbose.log`）：

| ID | 命令 | exit | 计数 |
|---|---|---|---|
| 01 | `go test -tags=integration ./test/integration/recovery -list '^TestRecoveryRuntimePostgres$'` | 0 | 列出目标测试 |
| 02 | `go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | 57 JSON 行；pass 7（pkg 1 + top 1 + 5 subtest），fail 0，skip 0；5 个唯一 subtest |
| 03 | `go test -race -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | 同 02：pass 7 / fail 0 / skip 0；5 个 subtest |
| 04 | `go vet -tags=integration ./test/integration/recovery` | 0 | 输出为空 |
| 05 | `go test -tags=integration ./cmd/han-server -list '^TestRecoveryRuntimeProductionChain$'` | 0 | 列出目标测试 |
| 06 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json` | 0 | 10 JSON 行；pass 2，fail 0，skip 0（无 subtest） |
| 07 | `go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json` | 0 | 同 06：pass 2 / fail 0 / skip 0 |
| gofmt | `gofmt -l runtime_admission_test.go runtime_recovery_test.go runtime_fixtures_test.go` | 0 | 0 行（已格式化） |
| 抽查 | `go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$/^Case18SameRevisionAndNoProductionClaim$' -count=1 -v` | 0 | PASS |

JSON 统计口径：仅解析以 `{` 起始的行，跳过 `#` 等非 JSON 行。

Case18 `-v` 真实日志核对：
- closed 场景：`statuses=[503 503]`，两个不同 trace ID，`article_handler=0 article_service=0 article_repository=0 article_author_reader=0`（两次 503、零下游成立）。
- open 场景：`status=200`，`etag="article-<uuid>-v1"`，`handler=1 trace=1 service=1 repository=1 author_read=1`。
- conditional：origin 403、csrf 403、If-Match 428、stale 412、accepted 204、`remaining_articles=0`、`completed_operations=1`、`audit_events=1`。

## 6. 未验证风险与限制

- **介质层面完全未验证**：真实 POSIX 文件 journal、真实 CAS、ACL、批准备份、真实 E2E 均未被本测试证明；软件 adapter 的通过不得被表述为这些能力已证明。作者 `runtime_recovery_test.go:26` 的日志已带免责声明，这一点表述正确。
- **归属不可核**：工作树 `go.mod` 为 modified、`test/` `internal/` `cmd/` 整体 untracked，无法用 diff 证明无越界修改生产代码（unknown）。已用 SHA 固定被审三文件，但无法固定其余生产文件基线。
- **环境单点**：仅本机 local trust 单实例 PostgreSQL；未验证多实例/网络故障。
- **非压测**：仅一次 `-race`，无 soak、无并发压力。
- **Lead Gate 证据缺口**：`lead-gate-r01/` 缺 `cmd-04.log`、`gofmt.log`；已由本次复跑补齐，但仍建议作者补档。
- **未覆盖对象**：未评审本批次其他文件与历史修订；未评审实现正确性（仅评审测试是否真实覆盖）。

## 7. 复审条件

作者整改后需提供：
1. 被审三文件的新 SHA 与修复 diff（逐条对应 H01–H05、M01–M06）。
2. 真实 `identity.NewRecoveryIdentityExecution`、`recovery.NewRuntime`、`session.NewStore` + `platform.NewHTTPAuthorizer` 的装配证据（日志或代码位置）。
3. 生产路由注册函数的使用证据（替代手搭 router）。
4. 真实/可切换 maintenance gate 证据与修正后的日志标签。
5. `gofmt`、7 条命令、Case18 `-v` 复跑日志。
6. Lead Gate 补齐 `cmd-04.log`、`gofmt.log`。

复审将对应原 finding 逐条核对，不替代作者修复。

## 8. 待决定（与缺陷分开）

- 若 Lead 决定接受“ABA/stale 由 adapter 判定”作为临时降级，需要显式记录为风险接受（含到期条件与后续真实介质验证计划）；在此之前本评审维持 CHANGES REQUIRED。
- 若决定把 `softwareClosureReader` 正向路径留到后续批次，需要登记为明确的覆盖缺口，而不是以当前测试宣称已覆盖。

## 9. 证据索引

- 本目录：`report.md`、`receipt.json`、`cmd-01.log`…`cmd-07.log`、`cmd-08-case18-verbose.log`、`gofmt.log`、`rerun-exit-codes.json`
- 作者证据：`.spec/wip/recovery-runtime/evidence-b13i-r01/author-r01/`
- Lead Gate：`.spec/wip/recovery-runtime/evidence-b13i-r01/lead-gate-r01/`