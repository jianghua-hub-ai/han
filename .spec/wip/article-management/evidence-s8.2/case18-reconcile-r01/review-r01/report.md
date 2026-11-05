# Case18 覆盖对账：C18-COV-F003 / C18-COV-F004 finding 级复审报告（r01）

> **AI 代理评审，非真人签字。** 本报告仅作为 Lead 的 finding 级复审输入，不代替 Lead Gate、人的业务接收、部署授权或 Case18/S8.2 整体验收。

## 1. 评审对象与边界

| 项 | 值 |
|---|---|
| 工作区 / 分支 / HEAD | `D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 原评审 | `.spec/wip/article-management/evidence-case18-coverage-r01/verdict.md`，SHA256 `C75CFC5662E537C0D40F6A3B19DA80C8A84EFA93E8E5942A65DF28E33BD14C97`，3464 B |
| 本轮对象 | `C18-COV-F003`（原 HIGH）、`C18-COV-F004`（原 MEDIUM） |
| objectType | code / evidence（finding 级复审，不是整卡或整批评审） |
| 写入范围 | 仅本目录 `report.md`、`verdict.md`；未改源码、测试、tasks、Plan、ADR 或既有证据 |
| 排除范围 | 不重开 `C18-COV-F002`（ADR-0034 范围决策关闭）与 Case18 真实部署 E2E/双系统（ADR-0035 范围决策关闭）；不评审 S8.2 其余实现质量；不评价 S1.5 |

本轮方法是：先固定真实工作树和对象哈希，再逐条核对原 finding 的事实前提；对代码/证据做只读检查；运行允许的 `gofmt -l`、`go vet -tags=integration ./cmd/han-server`；独立复跑一次指定集成测试命令。所有结论来自实际文件、日志和命令结果，不采用作者摘要作为通过依据。

## 2. 固定输入与哈希核验

### 2.1 代码与测试文件

| 文件 | 字节 | SHA256 |
|---|---:|---|
| `cmd/han-server/recovery_runtime.go` | 37,431 | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` |
| `cmd/han-server/main.go` | 173,317 | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` |
| `cmd/han-server/case18_session_store_integration_test.go` | 1,728 | `2E29CBFD252B9281D56882F4C8AE8D9868036274E29B70AE8F115F161B183172` |
| `cmd/han-server/recovery_production_chain_integration_test.go` | 167,339 | `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` |
| `cmd/han-server/recovery_enter_install_integration_test.go` | 20,564 | `6BFA7D761432C960740F4E1FE91566317B5707B8481CE0D9FF1CF46DD4106C3A` |
| `internal/session/store.go` | 20,715 | `F0659DFA4D0D922FB2A9083182CC18357C7A8F496525695906A0950A5084CB4D` |

### 2.2 证据文件与日志

| 文件 | 字节 | SHA256 |
|---|---:|---|
| `production-chain.log`（`case18-reconcile-r01`） | 9,746 | `5C5604603DEF586660D314E6361B36173056F98A9861111656517AD22E6392A9` |
| `f004-and-regression.log`（`case18-reconcile-r01`） | 35 | `ADD9EB0CE4AD9389E95878B2C3F0E1CC09EDD9FE195CCC406723749C8D018F41` |
| `b13h-batch-gate-r02/report.md`（S4.20 关闭证据） | 18,851 | `7893B4300E44C8BCFD9A326010EEB5FED6D565595F0DE40A6863B94E8E0EF3D4` |
| `s421-implementation-r01/report.md`（S4.21 实施证据） | 7,612 | `5B00BF776FC306C419F600E53A1EE59AA61C3E650ED3B2DC83D12D9C77CBD5FB` |
| `s421-independent-review-r01/report.md` | 15,873 | `0B27AF34363D9FF5B25E7E7D4DD88AB3D896C685221F5E70C82D8165773F2634` |
| `s421-independent-review-r01/verdict.md` | 1,581 | `C36A9D4C99B157389DF2009FD495C75B35A8ACD3EA3DAADDE19DF965591C6514` |
| `s421-handoff.md` | 3,155 | `2A0013FD4A92325DAD3BA35CF3444DBFA10B5A0884F17E335E26F591B8768019` |
| `case18-reconcile-r01/summary.md` | 5,098 | `74C8BF5C4BD972AAEF3C011B0F9E973F9758A1049DC71CD4F6EDA855C1F6E3F7` |

### 2.3 环境

- `Test-NetConnection 127.0.0.1:5433` 返回 `True`，说明测试可连接真实 PostgreSQL 端点。
- 本机未找到 `psql`，因此未独立重查 `version()`；测试通过真实 DSN 完成迁移与事务，但“18.6”版本号本身以既有环境记录为准，不在本报告重新声称。
- 工作树 `cmd/han-server/`、`internal/session/` 在 Git 中整体未跟踪（`git status --porcelain` 显示 `?? cmd/han-server/`、`?? internal/session/`）；因此以 SHA256+字节+实际读取作为对象证据，`git diff` 为空不能作为“未改动”的唯一证据。

## 3. C18-COV-F003 逐项核验

### 3.1 原 finding 的三个事实前提

原 finding 为 HIGH，文本为：

> S4.20 d7 证据绑定旧 `recovery_runtime.go 900F31F5…/35832`；S4.21 明写 production Enter/Install 仍 `ErrAdmissionClosed`、真实 PG S4.20 未重跑，当前 revision 无法核完整恢复后开流。

拆成四项可核事实：

1. S4.20 d7 证据是否绑定旧 `recovery_runtime.go`；
2. S4.21 是否仍声明 production `Enter`/`Install` 为 `ErrAdmissionClosed`；
3. 真实 PostgreSQL 上 S4.20 production chain 是否未在当前 revision 重跑；
4. 当前 revision 是否无法核“完整恢复后开流”。

### 3.2 事实 1：S4.20 证据确实绑定旧哈希，当前文件不是同一哈希

- S4.20 关闭证据 `b13h-batch-gate-r02/report.md:28` 与 `:59` 均记录 `cmd/han-server/recovery_runtime.go` 为 `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35,832 B。
- 现役文件实测为 `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` / 37,431 B。
- 结论：**不同哈希，原 finding 的第一条事实成立**。不能用“现役文件与 S4.20 关闭证据同哈希”表述；二者不是同一对象。

### 3.3 事实 2：S4.21 仍明确 production Enter/Install 为 closed

- `s421-implementation-r01/report.md:13` 记录 `recovery_runtime.go` 从 `900F31F5…/35832` 变为 `1C679377…/37431`，并注明“预期修改”。
- `s421-implementation-r01/report.md:34`：“`main.go` 未注入 `secretCandidate` 或其余 10 个端口，production `enter`/`install` 仍 closed。”
- `s421-implementation-r01/report.md:85`：“production `Enter`/`Install` 仍不可达，仍返回 `ErrAdmissionClosed`；本卡不声称 Case18/S8.2 PASS 或解除 blocked-upstream。”
- `s421-handoff.md:15-25` 列出 11 个仍 closed 的端口；`:45` 明确“模拟端口注入只证明装配路径可达，不证明 production `Enter`/`Install` 可达或完整恢复成功”。
- 代码核对：
  - `cmd/han-server/main.go:634-650` 的 `newRecoveryRootAssembly` 输入只有 `pool`、`plan`、`prepared`、`closures`、`seal`，没有 11 个 closed 端口。
  - `cmd/han-server/recovery_runtime.go:456-497` 的 `recoveryEnterInstallMissingPorts` 逐项检查 11 个端口；`:530-531` 在任一缺失时返回 `recoveryEnterInstallClosedError`。
  - `recovery_runtime.go:494-495` 以 `%w` 包装 `recovery.ErrAdmissionClosed` 并列出全部缺失端口。
- `s421-independent-review-r01/report.md:38`、`:58` 同样确认 `main.go:634-650` 未注入 11 端口，production `enter/install` 在 precheck 处即 closed。
- 结论：**第二条事实仍成立，但这是 ADR-0030/0031 下的设计边界**，不是本轮新增测试要改变的代码缺陷。任何“F003 CLOSED”都不能被解释为 production `Enter`/`Install` 已可达。

### 3.4 事实 3：真实 PostgreSQL 上 S4.20 production chain 已可在当前 revision 重跑

关键测试对象：

- `cmd/han-server/recovery_production_chain_integration_test.go:2742-2755`：`TestRecoveryProductionChainComplete` 的入口和 8 个子测试：
  - `harness`（`:2748`）
  - `ResumeCoordinatorRecoveryCompleteAndOpen`（`:2749`）
  - `SecondReadStatusAndMarkConsumed`（`:2750`）
  - `CanonicalReplaySameResult`（`:2751`）
  - `ClosedUnknownKeepIsolation`（`:2752`）
  - `SingleRuntimeStorePool`（`:2753`）
  - `SimulatedJournalKeyringBoundary`（`:2754`）
  - `NoSecondRuntimeRouterPoolAuthorityFallback`（`:2755`）
- `:2938` `s420RunProductionResume` 构造生产 root 装配，调用 `newRecoveryRootAssembly`（`:3011`），驱动 `assembly.runtime.Resume`，并在 `:3155-3158` 断言 `runtime.Verify` 返回 `public_ready`。
- `:3192` 起的 `testRecoverySecondReadStatusAndMarkConsumed` 断言 `MarkConsumed` 恰好一次、二次 `ReadStatus` 一致、无额外写入；`:3569` 输出 canonical replay 同结果；`:3827` 输出 `public_stream_open`；`:3867` 输出 closed/unknown 下游零调用。
- 测试使用 `s420NewFixture`（`:107`），在 `HAN_TEST_POSTGRES_DSN` 为空时 `t.Fatal("HAN_TEST_POSTGRES_DSN is required for S4.17; no Skip")`（`:111`），不是 skip。
- 该测试文件的 SHA256/字节为 `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` / 167,339，与 S4.20 关闭证据中登记的测试文件一致；说明本轮不是换了一个新测试，而是在当前 runtime 上重跑同一 production chain 测试。

现有日志：

- `production-chain.log`（9,746 B，`5C5604603DEF586660D314E6361B36173056F98A9861111656517AD22E6392A9`）：
  - `:1` `=== RUN TestRecoveryProductionChainComplete`
  - `:65` `--- PASS: TestRecoveryProductionChainComplete (10.87s)`
  - `:66-73` 8 个子测试逐项 PASS
  - `:75` `ok han/cmd/han-server 10.981s`
- 日志内的源码行号与当前文件匹配，例如 `:2818` 对应 `testRecoveryChainHarness` 调用点、`:3156` 对应 `s420RunProductionResume`、`:3206` 对应 `second_read` 日志、`:3569` 对应 canonical replay、`:3605` 对应 single runtime/store/pool、`:3616` 对应 simulated journal/keyring、`:3827` 对应 public stream open、`:3867` 对应 closed/unknown 隔离。这说明日志不是与当前测试文件无关的旧产物。

本轮独立复跑：

命令（仅执行一次）：

```text
go test -tags=integration -timeout=20m ./cmd/han-server -run '^(TestCase18RealSessionStoreResolveAndRevoke|TestRecoveryProductionChainComplete|TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix)$' -count=1
```

实际结果：`EXIT=0`，墙钟 `12.292s`，包级 `ok han/cmd/han-server 10.514s`。因为 `TestRecoveryProductionChainComplete` 的 8 个子测试都通过 `t.Run` 挂在父测试下，包级 PASS 意味着父测试和全部子测试均未失败；`go test -list` 也确认该测试名与 F004 测试名均在当前包中。

结论：**“真实 PG S4.20 未重跑”在当前 revision 已被本轮独立复跑消除**；现有日志和本轮复跑都指向同一测试对象和当前 runtime。

### 3.5 事实 4：当前 revision 的“完整恢复后开流”证据是否足够

支持闭合的部分：

- 当前 `recovery_runtime.go` 哈希为 `1C679377…/37431`，与 S4.21 独立评审锁定的对象一致（`s421-independent-review-r01/report.md:13`；`verdict.md:4`）。
- `S4.21` 独立评审结论为 `PASS WITH MINOR`，唯一 finding 是 Lead 日志命令文本偏差（LOW，非阻断），代码/测试无需返工（`s421-independent-review-r01/verdict.md`）。
- 本轮复跑在同一 revision + 真实 PostgreSQL 上，production chain 8 个子测试全部通过（包级 PASS）。
- `ResumeCoordinatorRecoveryCompleteAndOpen` 覆盖 resume 后 `public_ready`、run outcome `succeeded`、journal stage `consumed`、publication 行与 completion 字段；`SecondReadStatusAndMarkConsumed` 覆盖二次读取、MarkConsumed 恰好一次、零额外写入；`CanonicalReplaySameResult` 覆盖 canonical replay 同结果；`ClosedUnknownKeepIsolation` 覆盖 closed/unknown 下游零调用；`SingleRuntimeStorePool` 覆盖单一 runtime/store/pool；`NoSecondRuntimeRouterPoolAuthorityFallback` 覆盖无第二 router/pool/authority fallback。
- `production-chain.log:55` 明确 `adapter_scoped=simulated_journal_hmac_keyring; not real keyring files; reason=windows_unsupported`，说明测试没有冒充真实 keyring。

仍然存在的边界：

- `main.go:634-650` 没有注入 11 个 closed 端口，production `Enter`/`Install` 仍返回 `ErrAdmissionClosed`；这是 ADR-0030/0031 的设计边界，`s421-handoff.md:45` 明确不证明 production `Enter`/`Install` 可达或完整恢复成功。
- 测试使用 `newRecoveryRootAssembly` + 测试侧 software journal/commitments/sources/exclusive/lifecycle 等适配；它是 production runtime/root assembly 代码路径的测试装配，不是部署后的真实服务入口。
- Journal/HMAC/keyring 是 simulated；真实 keyring/HSM、POSIX journal/fsync/flock、真实 approved backup inventory 未验证。
- `s421-independent-review-r01/report.md:95` 明确其自身“未运行需要真实 PostgreSQL 的 S4.16/S4.20 生产链测试”；本轮复跑补上了这一项，但不改变其他边界。

### 3.6 F003 结论

**CLOSED（finding 级证据缺口已闭合；不等于 production Enter/Install 可达，也不等于 Case18 整体 PASS）。**

理由：

1. 原 finding 的第一条事实“S4.20 证据绑定旧哈希”仍然成立，但它是历史证据；S4.21 实施/独立评审已记录 before/after 并锁定当前哈希，本轮又在当前 revision 上重跑同一 production chain 测试。
2. 原 finding 的第二条事实“production Enter/Install 仍 closed”仍成立，但已在 ADR-0030/0031 下明确为设计边界，且 `s421-handoff.md:45` 没有把测试注入冒充为 production 可达。
3. 原 finding 的第三条事实“真实 PG S4.20 未重跑”已被本轮独立复跑消除：`EXIT=0`、包级 `ok 10.514s`、既有日志 8 个子测试 PASS。
4. 原 finding 的核心影响“当前 revision 无法核完整恢复后开流”不再成立：当前 revision + 真实 PostgreSQL + production root assembly 的 resume→publish→`public_ready` 链已可核；模拟 journal/keyring 与 production 端口 closed 作为保留限制记录，不扩张为 production readiness。

若 Lead 还要求把 S4.20 原始证据目录本身追加“superseded by S4.21”指针，那是证据整理动作，不是本 finding 的代码/测试阻断；本报告不修改既有证据。

## 4. C18-COV-F004 逐项核验

### 4.1 原 finding 的事实前提

原 finding 为 MEDIUM，文本为：

> Session/AccessGeneral 仅有 resolver/repository/handler 接缝；缺当前 revision 真实 `SessionStore` SQL/撤销证据；账号级 403 不等于全局 503。

拆分：

1. 此前只有 resolver/repository/handler 接缝；
2. 缺当前 revision 真实 `SessionStore` SQL/撤销证据；
3. 账号级 403 不能替代全局 503（这是一条覆盖边界警告）。

### 4.2 新增测试确实使用真实 PostgreSQL 的 `session.Store`

- `cmd/han-server/case18_session_store_integration_test.go`：
  - `:1` `//go:build integration`
  - `:13-18` 注释说明目标是关闭 `C18-COV-F004` 的 store 层证据缺口，使用真实 Postgres-backed `session.Store`，并明确不替代 dispatcher 层检查。
  - `:19` `TestCase18RealSessionStoreResolveAndRevoke`
  - `:20` `world := s420NewCase18World(t)`
  - `:23-26` `session.NewStore(session.StoreDependencies{Transactions: world.f.pool, PasswordResetGate: identity.PasswordResetStore{}})`，与生产 root 使用同一依赖形态。
  - `:31-42` 用 `Token`、`CSRFToken`、`RequireCSRF: true` 解析活动会话，并断言 `principal.UserID == world.userID`。
  - `:44-46` 调用 `store.Revoke(ctx, world.f.pool, world.sessionToken, world.f.now.Add(time.Minute), "case18-f004-revoke")`。
  - `:48-55` 用同一 token 再解析，若 `err == nil` 则 `t.Fatal("revoked session still resolves")`。
- `s420NewCase18World` 在 `recovery_production_chain_integration_test.go:1828`；其中 `:1840` 用 `session.NewStore` 创建真实 store，`:1849` 起用 `store.Create` 写入真实会话。
- `s420NewFixture` 在 `recovery_production_chain_integration_test.go:107`；`:111` 在 `HAN_TEST_POSTGRES_DSN` 为空时 `t.Fatal(... no Skip)`，证明该测试不会因缺 DSN 被静默跳过。
- `session.Store.Revoke` 在 `internal/session/store.go:235`；其 SQL 常量 `revokeSessionSQL` 在 `:534-537`：
  - `UPDATE sessions SET revoked_at = $2, revoke_reason = $3 WHERE token_digest = $1 AND revoked_at IS NULL`
  - 这说明撤销走真实 SQL，且不会持久化原始 token。

### 4.3 实际执行结果

本轮同一条命令（与 F003 复跑合并执行一次）：

```text
go test -tags=integration -timeout=20m ./cmd/han-server -run '^(TestCase18RealSessionStoreResolveAndRevoke|TestRecoveryProductionChainComplete|TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix)$' -count=1
```

结果：`EXIT=0`，墙钟 `12.292s`，包级 `ok han/cmd/han-server 10.514s`。由于 `-run` 正则包含 `TestCase18RealSessionStoreResolveAndRevoke`，且该测试无 `t.Skip` 路径，包级 PASS 即表示该测试执行并通过。

`go test -list` 结果（`EXIT=0`，`0.084s`）确认当前包中存在：

```text
TestCase18RealSessionStoreResolveAndRevoke
TestRecoveryIsolationMatrix
TestRecoveryMainBusinessDispatcher
TestRecoveryProductionChainComplete
```

现有日志 `f004-and-regression.log` 为 35 B、`ADD9EB…/35 B`，内容只有：

```text
ok  	han/cmd/han-server	8.672s
```

它只能证明包级通过，不能单独证明具体测试名或子测试；本轮复跑与 `-list` 补强了这一点。

### 4.4 与 summary 的证据指针差异

`case18-reconcile-r01/summary.md:39` 记录 F004 合并回归为 `8.503s`，日志哈希 `88A57C7B…`；但当前 `f004-and-regression.log` 为 `8.672s`、`ADD9EB0C…/35 B`。文件时间戳显示 summary 写于 21:04:26，日志写于 21:04:41，说明日志在 summary 之后被覆盖，summary 的指针已陈旧。

这不是 F004 代码/测试的阻断，但属于证据指针漂移；本轮不把 summary 的 `88A57C…` 作为 F004 通过证据，而以实际当前日志和本轮独立复跑为准。

### 4.5 保留限制

- 新测试只覆盖 `session.Store` 的 `ResolvePrincipal` / `Revoke` 行为，不覆盖 HTTP handler、`AccessGeneral`、账号级 403 或全局 503 矩阵；这些属于原 Case18 其他分项/`C18-COV-F002` 及范围决策，不在 F004 的 store 层证据缺口内。
- 第二次 `ResolvePrincipal` 只断言 `err != nil`，没有断言具体错误类型（如 `session.ErrSessionRequired`），也没有直接查询 `revoked_at` 行；它证明了“同一 token 撤销后不再解析”的行为，但错误类型断言强度可以更高。这是残余测试强度限制，不构成 F004 的阻断性缺陷。
- 测试使用的是 `s420NewCase18World` 的完整 fixture（含 runtime/handlers 等），不是最小 store-only fixture；这增加了执行成本，但不影响“真实 PG + 真实 store”的证据性质。

### 4.6 F004 结论

**CLOSED（仅限 store 层真实 SQL/撤销证据；不等同全局 503 或 AccessGeneral 全矩阵覆盖）。**

理由：

1. 原 finding 的核心缺口“缺当前 revision 真实 `SessionStore` SQL/撤销证据”已由新测试直接补齐：真实 PG、真实 `session.Store`、`Resolve`→`Revoke`→同 token 再解析失败。
2. 新测试无 skip 路径；本轮独立复跑 `EXIT=0`；`-list` 确认测试存在；`gofmt`/`vet` 均通过。
3. “账号级 403 不等于全局 503”仍然成立，但新测试没有宣称覆盖全局 503；该边界属于 F002/范围决策，不是 F004 store 层证据的缺陷。
4. `summary.md` 的旧日志指针漂移已记录；本轮以实际日志 + 独立复跑为准，不把它当作通过依据。

## 5. 逐项核对表

| finding | 原前提 | 独立核对结果 | 判定影响 |
|---|---|---|---|
| C18-COV-F003 | S4.20 d7 证据绑定旧 `recovery_runtime.go 900F31…/35832` | 属实；S4.20 `b13h-batch-gate-r02/report.md:28,59`；当前 `1C679…/37431`；S4.21 `s421-implementation-r01/report.md:13` 记录变更；`s421-independent-review-r01/report.md:13` 锁定当前哈希 | 历史事实成立，但已被 S4.21 证据链替代；不阻断 finding 闭合 |
| C18-COV-F003 | S4.21 production Enter/Install 仍 `ErrAdmissionClosed` | 属实；`s421-handoff.md:15-25,45`；`main.go:634-650` 无 11 端口；`recovery_runtime.go:456-497,530-531`；`s421-independent-review-r01/report.md:38,58` | 设计边界，保留限制；不把 CLOSED 解释为 production 可达 |
| C18-COV-F003 | 真实 PG S4.20 未重跑 | 已消除；`production-chain.log` 8 子测试 PASS；本轮 `EXIT=0`，包级 `10.514s` | 支持 CLOSED |
| C18-COV-F003 | 当前 revision 无法核完整恢复后开流 | 不再成立；当前 revision + 真实 PG + production root assembly 的 resume→publish→`public_ready` 可核；`recovery_production_chain_integration_test.go:2938,3011,3155-3158`；`production-chain.log:65-75` | 支持 CLOSED，保留 simulated journal/keyring 与 production 端口 closed 限制 |
| C18-COV-F004 | 仅有 resolver/repository/handler 接缝 | 新测试直接使用真实 `session.Store`；`case18_session_store_integration_test.go:23-46`；helper `recovery_production_chain_integration_test.go:1828,1840` | 缺口被补齐 |
| C18-COV-F004 | 缺真实 `SessionStore` SQL/撤销证据 | 已补齐；`store.go:235,534-537`；`case18_session_store_integration_test.go:31-55`；本轮 `EXIT=0`；`-list` 确认测试存在 | 支持 CLOSED |
| C18-COV-F004 | 账号级 403 不等于全局 503 | 仍成立；新测试不覆盖 HTTP/AccessGeneral/全局 503，也不宣称覆盖 | 范围边界，不作为 F004 阻断 |

## 6. 实际执行命令与结果

| # | 命令 | Exit | Duration | 结果 |
|---:|---|---:|---:|---|
| 1 | `git rev-parse --abbrev-ref HEAD` / `git rev-parse HEAD` | 0 | — | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 2 | `Get-FileHash` + `Get-Item`（对象与证据文件） | 0 | — | 见 §2 哈希/字节表 |
| 3 | `gofmt -l <cmd/han-server/*.go>` | 0 | — | 无输出 |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | 1.567s | 无输出 |
| 5 | `Test-NetConnection 127.0.0.1 5433` | — | — | `True`；`psql` 未安装，未重查版本 |
| 6 | `go test -tags=integration -timeout=20m ./cmd/han-server -run '^(TestCase18RealSessionStoreResolveAndRevoke|TestRecoveryProductionChainComplete|TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix)$' -count=1` | 0 | 12.292s（墙钟）/ 10.514s（包级） | `ok han/cmd/han-server 10.514s` |
| 7 | `go test -tags=integration -list '^(TestCase18RealSessionStoreResolveAndRevoke|TestRecoveryProductionChainComplete|TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix)$' ./cmd/han-server` | 0 | 0.084s | 列出 4 个测试名 |

NOT RUN（本轮未执行，不能据此声称通过）：

- `go build ./...`；
- `go test -race` 或全量测试套件；
- 真实部署 E2E / 双系统；
- 真实 `main` 入口上的 production `Enter`/`Install`；
- 真实 keyring/HSM、POSIX journal/fsync/flock、真实 approved backup inventory；
- 真实 PostgreSQL 版本 `18.6` 的独立 SQL 重查（`psql` 不可用）。

## 7. 保留限制与非声明

- **production Enter/Install**：`main.go:634-650` 未注入 11 个端口，实际 `main` 的 `Enter`/`Install` 仍按 ADR-0030/0031 返回 `ErrAdmissionClosed`。F003 的 CLOSED 只表示当前 revision 的恢复链证据缺口闭合，不表示 production 入口可达或可部署。
- **模拟边界**：`production-chain.log:55` 明确 `simulated_journal_hmac_keyring; not real keyring files; reason=windows_unsupported`；真实 keyring/HMAC/HSM、POSIX journal、approved backup inventory 未验证。
- **测试装配边界**：`TestRecoveryProductionChainComplete` 使用 `newRecoveryRootAssembly`（`recovery_production_chain_integration_test.go:3011`）与测试侧 software 端口；它是 production runtime/root assembly 代码路径，不是部署后的真实服务。
- **F004 边界**：只覆盖 `session.Store` 的 resolve/revoke；不覆盖 AccessGeneral/HTTP/全局 503，不把账号级 403 当作全局 503；第二次解析只断言非 nil error，未断言错误类型。
- **S4.20 历史证据**：`b13h-batch-gate-r02/report.md` 仍绑定旧 `900F31…/35832`，本轮未改写；当前 revision 的替代证据链是 S4.21 锁定 + 本轮 production chain 复跑。若 Lead 需要 S4.20 目录内的 superseded 指针，属于证据整理动作。
- **summary 指针漂移**：`summary.md:39` 的 `8.503s/88A57C…` 与当前 `f004-and-regression.log` 的 `8.672s/ADD9EB…` 不一致；以实际文件与本轮复跑为准。
- **未做 Git 写**：未提交、未暂存、未修改源码/测试/tasks/Plan/ADR/既有证据；仅写本目录 `report.md`、`verdict.md`。
- **非声明**：本报告不宣称 Case18 PASS、S8.2 PASS、S1.5 PASS、production readiness、人的业务接收、部署授权或 Lead Gate 通过。

---

AI 代理评审，非真人签字；不代替 Lead Gate、人的业务接收或部署授权。
