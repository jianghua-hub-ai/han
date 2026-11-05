# article-management-S4.6 独立代码复审 r02

- Reviewer：Faraday（独立非作者 Reviewer）
- 评审日期：2026-10-23
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/article-management/review-code-s4.6-r01.md`
- 原报告 SHA256：`778D8704132991844C4201A34C9EA4F583372AECA6A50B716BF1F1C312923C99`
- **Code Review Gate：FAIL**

## 1. 固定对象与漂移核对

评审开始时已确认新报告路径不存在、分支与 HEAD 匹配，并逐项核对以下 13 个固定修复对象。落报告后再次核对；本结论只绑定这些精确对象。

| 文件 | git blob | SHA256 |
|---|---|---|
| `cmd/han-server/main.go` | `7f0ba704e503bcb5bb415731e9bf418dcd4c36df` | `FABE00B9B0925D41288B8A42E1FAC2E31859C2A0AD53D144830D713C6830BC10` |
| `cmd/han-server/main_test.go` | `38a6a6aa6664676d7461a91746ec33e78c2f6c12` | `2DB871D4C90D56E5804E5140271336A8456B9C8692EE3B588E3D66BC3292A24D` |
| `cmd/han-server/lifecycle.go` | `9100ff4bba6ea1741b24754254e8b131fe1505c2` | `B6B246667F9279800B023EF67E8FB8D5EB8A2F4E56B6DA403C6D591E3B391816` |
| `cmd/han-server/lifecycle_test.go` | `b79ba7e06262b60ec04c4c416728b232933c748d` | `8C59F61FAD9A65E637EB9991C2951293ECDC7417418B38044D3243EBDF32E66B` |
| `internal/article/deletion_worker.go` | `05eee80b08c238a609b95c39ec15a2326fcf911d` | `80CDDA2AD244C462BC5D84B6A1EBAA921680D6E735C8922CBCF1D236ACAFFBF0` |
| `internal/article/deletion_worker_test.go` | `6962e25651032ec9461b3a9059925f2d3cec03f0` | `65A19A5413D64A16EA271A2E556A88E465ED1E93C12F3CDD717F685A6FA645AD` |
| `internal/article/deletion_finalizer.go` | `66e7a8d47eaa390d9243db824afa10f05d9273e9` | `73E0E967EA9A3AA85F610E2146B1A03B9CA96DAB001E8B0F071524BA5FB470E0` |
| `internal/article/deletion_journal.go` | `810a8017dbecf7ee71b386802f49a1f830d6803c` | `B70E65B56AEAE06E465A52D96C4CC2CDC1AAB021A66C5C9700261EA448317B4F` |
| `internal/article/deletion_test.go` | `c345e5db13d27ac93ab46fb6a59f67194b5c0b54` | `C4879E9B7507B0C7FCE6DCE061C069E6E25810BF56AC83EB4A9B77780B10F236` |
| `internal/article/repository.go` | `41c18954de82ff9253f27ea0b2fffda08496263e` | `5A03F79D15706CB5CDD4F9A51129D388E192AF7967DD848C6585686FA511DDC7` |
| `internal/article/repository_integration_test.go` | `a5f44b8a9d98197729e7c2c6ab71ff533237cc76` | `EE9A68CCA141AC6F84C412C6BCE54457C543003ABB2812D4B8C3690D9B269355` |
| `internal/article/deletion_filelock_flock.go` | `b3d33952515d1eb67dd59c89fa843c6349d454a1` | `C72AC42EAF78B279CB774C79B9024A7E330C1FCCE2D40D912782E5B0B50A5C96` |
| `internal/article/deletion_filelock_unsupported.go` | `ad26bfcdb0b422c8801774cfe8dbdcb5e9aa3b35` | `BF140294B11D068B2971DFEFF4D3598FFAEC591FCF1F5BE0435640370C5FEB8B` |

## 2. 实际读取与评审依据

完整复用并执行：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- 适用的 observability、tracing、产品安全红线
- 已启用 `go-service` profile 的 README、coding、no-hardcode、security、observability、tracing 规则

上游与接缝读取范围：

- r01 原报告及其 F001～F004
- `tasks.md` 的完整 `article-management-S4.6` 条目、B6A-W Plan
- article design r10、stories r07、test-plan r08 §4.10.2/§9.1 与 A-WIRE-CFG/JRN/RTR/WKR/LIFE
- 13 个固定文件
- Article 删除 Coordinator/CommandService/Handler/model、repository/Journal/finalizer/worker
- 唯一 production composition/router、server lifecycle、session/audit/platform trace/error 直接接缝
- migration `0003_article.sql` 与 `internal/recovery` 私有锁隔离

未扩展为全仓无界审计；未读取、打印或持久化 `HAN_TEST_POSTGRES_DSN`。

## 3. 两轮复审范围

### Round 1：固定修复对象

逐项核查 r01 F001～F004 的实现和测试：路径分量安全、A-WIRE 确定性矩阵、lease competition 分类、worker backoff/no-op、Journal/lock/composition 错误去敏，以及修复是否引入竞态、错误分类回归或公共契约变化。

### Round 2：有界消费者深审

沿唯一 production composition/router、Article 删除调用链、repository discovery、Journal/provider、finalizer、worker 与 lifecycle shutdown/unwind 核对真实消费者；检查第 14 文件、新 migration、新依赖、第二 composition/router、Recovery 私有原语变化和公共契约越界。

结果：未发现上述范围越界，也未发现 F001/F003/F004 修复引入的新缺陷；F002 的既定关闭条件仍未满足。

## 4. 原 findings 逐项状态

### `ARTICLE-S4.6-R01-F001` — **CLOSED**

**证据**

- `internal/article/deletion_journal.go:258-307` 从卷/路径根开始，对到目标的所有既存路径分量逐项执行 `Lstat`，拒绝 symlink、非目录和不满足 owner-only 的对象。
- `ValidateArticleDeletionJournalRoot` 使用同一分量校验；`ensureArticleDeletionDirectory` 在创建前校验 parent、创建后校验 child；`articleDeletionChildPath` 拒绝 `..`、分隔符和路径逃逸。
- `internal/article/deletion_test.go:499-568` 覆盖祖先 symlink、直接 symlink、非目录、group 权限及 child traversal。

**结论**

原可复现的祖先 symlink 绕过已关闭。路径检查与后续文件系统操作间仍存在操作系统级 TOCTOU 窗口，但真实 Unix 持久卷、跨进程竞争与抗崩溃验证属于既定后继范围；当前未发现可在本卡已确认设计和 13 文件内进一步消除且尚未覆盖的确定性逃逸路径。

### `ARTICLE-S4.6-R01-F002` — **OPEN（HIGH）**

修订显著增加了 A-WIRE 命名测试，但仍未完成 task 明确要求的确定性测试矩阵；测试数量不能替代行为约束。

**仍缺失的可触发证明**

1. **A-WIRE-CFG-04 边界矩阵不完整**
   - `cmd/han-server/main_test.go:2039-2127` 验证了合法边界，但越界仅单侧抽样：poll 缺上越界、batch 缺 `0`、lease 缺 `<5s`、operation budget 缺 `>1m`、initial backoff 缺 `>10s`、max backoff 缺 `<1s`、max attempts 缺 `0`、shutdown budget 缺 `>1m`。
2. **A-WIRE-CFG-06 零副作用矩阵不完整**
   - `cmd/han-server/main_test.go:2156-2229` 仅抽样三种配置失败，未对各配置失败类型逐项证明 DB parse/connect/ping/migrate、composition 与 listener 调用均为零，也未逐项执行敏感 canary 扫描。
3. **A-WIRE-RTR-03/RTR-05 安全链证明不完整**
   - `cmd/han-server/main_test.go:1763-1845` 覆盖错误方法和 Origin 首层拒绝，但没有用有序 spy 证明完整 `Trace → Origin → Session → CSRF → Principal/授权 → Handler` 顺序，也没有逐层拒绝后零后续调用的矩阵；三类 DELETE route 的角色/状态组合未在 production composition 层逐项证明。
4. **A-WIRE-RTR-06 共享实例证明不完整**
   - `cmd/han-server/main_test.go:2401-2518` 证明 Identity 路由基本共存回归，但未证明 DB pool、Session store、OriginGuard、AuditStore、logger、clock/ID provider 各自为唯一共享实例。
5. **A-WIRE-WKR-03/WKR-07 行为矩阵不完整**
   - WKR-03 仍主要通过 SQL 字符串片段断言 discovery 契约，未形成 fake repository 的状态、稳定排序、limit 行为矩阵。
   - WKR-07 已覆盖 lease competition/no-op、trace failure 和 operation deadline 等部分分类，但未逐项覆盖 DB/Journal 暂不可用及全部 terminal 分类与敏感 canary。
   - `DeletionWorkerDependencies` 仍没有可测试的低敏 category/reason bucket observer/log seam，无法证明 design r10 与 test-plan WKR-07 要求的低敏分类记录及原值不泄露。

**影响**

配置 fail-closed、路由安全链、共享 composition 实例和 worker 分类/可观测性仍可能在当前 suite 全绿时回归。该缺口是 r01 F002 的剩余关闭条件，不另立重复 finding。

**最小复审条件**

仅在固定 13 文件内补齐上述缺失的行为测试；如为证明 WKR-07 所必需，可增加私有、低敏、可注入的分类 observer seam。不得引入第 14 文件、新依赖、migration、第二 composition/router 或公共契约变化；不得把真实 PostgreSQL、真实 Unix 卷、Host/部署/E2E 错误地变成本卡前置。

### `ARTICLE-S4.6-R01-F003` — **CLOSED**

**证据**

- `internal/article/deletion_finalizer.go:28,296-316` 使用私有 `errDeletionLeaseCompetition`；活动 lease 返回同时满足 `errors.Is(err, ErrDeletionInProgress)` 与私有 competition sentinel 的错误。
- `internal/article/deletion_worker.go:124-166` 只把带 competition sentinel 的 `ErrDeletionInProgress` 归为 transient 并进入 full-jitter/backoff；裸 `ErrDeletionInProgress` 保持 completed/not-claimed no-op。
- `internal/article/deletion_worker_test.go:249-314` 分别证明 competition backoff 和裸 no-op；`internal/article/deletion_test.go:572-590` 证明 finalizer 的公共与私有错误语义。

**结论**

competition、completed/no-op、批次退出与 backoff 分类已分离，且保留既有公共 `errors.Is(err, ErrDeletionInProgress)` 语义。

### `ARTICLE-S4.6-R01-F004` — **CLOSED**

**证据**

- `internal/article/deletion_journal.go:46-48,329-329,413-413,508-605,680-684` 将 operation/close 文件系统失败映射为私有稳定 filesystem 类别和固定 operation 文本，不保留 `*os.PathError` 的绝对路径。
- `internal/article/deletion_filelock_flock.go:27-73` 对 open/inspect/chmod/flock/unlock/close 采用同一去敏边界，同时保留 busy/corrupt 等安全分类。
- `cmd/han-server/main.go:1791-1805` 将 startup provider 错误限制为 busy/corrupt/unsupported/not-found/unavailable 稳定类别。
- `internal/article/deletion_test.go:642-727` 与 `cmd/han-server/main_test.go:2232-2310` 对 startup、operation、close 的 path/key/body canary 做扫描。

**结论**

在已审查的 startup/operation/close 路径中未发现绝对路径、key 或 record body 泄漏；`errors.Is` 所需的内部安全分类仍可判定，worker 的 transient/terminal 判断未被 sanitizer 吞掉。

## 5. 新 findings、待决定与建议

### 新 findings

无。F002 的剩余测试与可观测 seam 缺口属于原 finding 的关闭条件。

### 待决定

无。未发现必须改变已确认架构、公共 REST、migration、依赖或 13 文件边界的问题。

### 非阻断建议

- 后续补测以 A-WIRE ID 与具体行为命名，保留“错误实现会失败”的明确断言，避免只检查构造成功或源代码字符串。
- worker 低敏分类 seam 只记录稳定 category/reason bucket；不要记录 operation ID、Journal 路径、key、record body 或底层原始错误。

## 6. A-WIRE 风险映射

| 范围 | 本轮结论 |
|---|---|
| A-WIRE-CFG-01～06 | **部分满足**；CFG-01～03/05 与部分 CFG-04/06 已有证明，F002 所列完整边界和逐失败零副作用矩阵仍缺 |
| A-WIRE-JRN-01～07 | **本卡确定性范围内满足**；F001/F004 已关闭，真实 Unix 卷与崩溃/断电仍为外部边界 |
| A-WIRE-RTR-01～06 | **部分满足**；exact DELETE、JSON 404/405/Allow/nosniff 与 Identity 共存已有证明，完整安全链拒绝矩阵和共享实例证明仍缺 |
| A-WIRE-WKR-01～10 | **部分满足**；competition/no-op、jitter、batch、wake、deadline 等已有证明，discovery fake 行为矩阵与完整分类/低敏观测仍缺 |
| A-WIRE-LIFE-01～08 | **满足已审查的本卡确定性范围**；共享 deadline、worker failure、one-shot、reverse close/unwind 等已有直接测试 |

## 7. Reviewer 独立命令与结果

环境：`go1.27.0 windows/amd64`，`CGO_ENABLED=1`。

```text
git branch --show-current
git rev-parse HEAD
Test-Path .spec/wip/article-management/review-code-s4.6-r02.md
Get-FileHash -Algorithm SHA256 <r01 and 13 fixed files>
git hash-object -- <13 fixed files>
git status --short -- <13 fixed files> <r01> <r02>
```

结果：开始时 branch/HEAD、r01 与 13 个对象均匹配；r02 创建前不存在。

```text
go test ./internal/article -run 'TestDeletionJournal_AWIREJRN02RejectsUnsafePathComponents|TestDeletionFinalizer_AWIREWKR07ClassifiesActiveLeaseCompetition|TestDeletionWorker_AWIREWKR07DistinguishesCompetitionFromNoOp|TestDeletionJournal_AWIREJRN07SanitizesOperationAndCloseFailures' -count=1

go test ./cmd/han-server -run 'TestArticleBusinessRouter_AWIRERTR01RTR02RTR04UsesExactDeletePaths|TestProductionBusinessHandler_AWIRERTR03RTR05ArticleSecurityChain|TestLoadStartupConfig_ArticleDeletionMatrix|TestLoadStartupConfig_ArticleDeletion_AWIRECFG06HasZeroSideEffects|TestArticleJournal_AWIRECFG06SanitizesStartupFailure|TestArticleJournal_AWIREJRN07SanitizesLifecycleCloseFailure|TestRunLifecycle_AWIRE' -count=1

go test ./internal/article ./cmd/han-server -count=1
go test -race ./internal/article ./cmd/han-server -count=1
go vet ./internal/article ./cmd/han-server
go build -o NUL ./cmd/han-server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -o NUL ./internal/article
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go test -c -o NUL ./internal/article
go test -tags=integration ./internal/article -run '^$' -count=1
go mod verify
gofmt -l <13 fixed Go files>
```

结果：全部 PASS；`gofmt -l` 无输出。integration-tag 命令显示 `[no tests to run]`，仅作为编译证据，不冒充 PostgreSQL 集成验证。双 GOOS 命令均为 compile-only，未执行目标平台代码。

落报告后执行：

```text
git diff --check -- <13 fixed files> .spec/wip/article-management/review-code-s4.6-r02.md
PowerShell trailing-whitespace scan <13 fixed files> <r02>
```

结果记录于最终漂移核对。

最终漂移核对结果：branch/HEAD 与 r01 SHA256 保持匹配，13 个固定对象的 SHA256/git blob mismatch 为 `0`；限定 `git diff --check` PASS，PowerShell trailing-whitespace 计数为 `0`。

Lead 提供的验证仅作为外部证据；上述命令为 Reviewer 本轮独立执行。未访问或打印 DSN。

## 8. 未验证限制

- 未连接真实 PostgreSQL；未验证真实事务、锁、lease、commit unknown 或双连接竞争。
- 未在受支持 Unix 目标持久卷执行真实 `flock`、file/directory `fsync`、崩溃、断电、磁盘故障或跨进程竞争；Windows 仅证明 fail closed 与 compile-only。
- 未运行真实 Host、部署、signal/exit E2E、浏览器或外部服务。
- 未执行容量、吞吐、积压或目标环境 benchmark。
- 本报告不声明 production readiness。

## 9. Gate 与复审条件

| 原 finding | 状态 |
|---|---|
| `ARTICLE-S4.6-R01-F001` | CLOSED |
| `ARTICLE-S4.6-R01-F002` | **OPEN（HIGH）** |
| `ARTICLE-S4.6-R01-F003` | CLOSED |
| `ARTICLE-S4.6-R01-F004` | CLOSED |

新增 finding：无。

由于 `ARTICLE-S4.6-R01-F002` 仍为阻断项：

**Code Review Gate：FAIL**

下一次复审应固定修订后的 13 文件，定点检查 F002 所列 CFG/RTR/WKR 行为矩阵及低敏分类观测，并重跑对应定向测试、普通测试、race、vet、双 GOOS compile-only、integration-tag compile-only、gofmt 和限定 diff/whitespace 检查。若关闭 F002 需要第 14 文件、公共契约、migration、新依赖或架构变化，应停止并返回上游对齐。
