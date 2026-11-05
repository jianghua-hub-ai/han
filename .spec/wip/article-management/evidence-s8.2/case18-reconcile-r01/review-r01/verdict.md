# Case18 覆盖对账 finding 级复审结论（r01）

> **AI 代理评审，非真人签字。** 本结论不代替 Lead Gate、人的业务接收或部署授权，不宣称 Case18 / S8.2 / S1.5 已 PASS 或已关闭。

## 结论

| finding | 原级别 | 本轮判定 | 一句话依据 |
|---|---|---|---|
| `C18-COV-F003` | HIGH | **CLOSED** | 当前 revision（`recovery_runtime.go` `1C679377…/37431`）+ 真实 PostgreSQL 上 `TestRecoveryProductionChainComplete` 独立复跑 `EXIT=0`（既有日志 8 个子测试全 PASS）；旧 S4.20 哈希由 S4.21 变更证据替代。 |
| `C18-COV-F004` | MEDIUM | **CLOSED** | 新增 `TestCase18RealSessionStoreResolveAndRevoke` 使用真实 PG `session.Store` 完成 `Resolve→Revoke→同 token 再解析失败`；本轮独立复跑 `EXIT=0`。 |

> 两条 CLOSED 均为 **finding 级证据缺口闭合**，不是 Case18 整体 PASS；production `Enter`/`Install` 仍按 ADR-0030/0031 为 closed，模拟 journal/keyring 等边界保留在 §4。

## F003 关键证据

- 原 finding：`.spec/wip/article-management/evidence-case18-coverage-r01/verdict.md`，SHA256 `C75CFC5662E537C0D40F6A3B19DA80C8A84EFA93E8E5942A65DF28E33BD14C97`，3464 B。
- 现役 `cmd/han-server/recovery_runtime.go`：`1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` / 37,431 B。
- 现役 `cmd/han-server/main.go`：`6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173,317 B。
- S4.20 关闭证据仍绑定旧哈希：`b13h-batch-gate-r02/report.md:28`、`:59` 记录 `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35,832 B；`report.md` 本身 `7893B430…/18,851 B`。
- S4.21 变更/锁定证据：`s421-implementation-r01/report.md:13`（`900F31…/35832` → `1C679…/37431`，`5B00BF77…/7,612 B`）；`s421-independent-review-r01/report.md:13`（锁定 `1C679…/37431`，`0B27AF34…/15,873 B`）；`s421-independent-review-r01/verdict.md:4`（`C36A9D4C…/1,581 B`）。
- 测试对象：`cmd/han-server/recovery_production_chain_integration_test.go`，`65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` / 167,339 B；`TestRecoveryProductionChainComplete` 入口 `:2747`，8 个子测试 `:2748-2755`；`s420RunProductionResume` `:2938`；`newRecoveryRootAssembly` `:3011`；`public_ready` 断言 `:3155-3158`；`MarkConsumed`/二次读取 `:3192` 起；canonical replay `:3569`；`public_stream_open` `:3827`；closed/unknown 隔离 `:3867`。
- 现有日志：`production-chain.log`，`5C5604603DEF586660D314E6361B36173056F98A9861111656517AD22E6392A9` / 9,746 B；`:65` 父测试 PASS 10.87s；`:66-73` 8 个子测试逐项 PASS；`:75` `ok han/cmd/han-server 10.981s`；日志中的源码行号与当前测试文件一致（如 `:2818`、`:3156`、`:3206`、`:3569`、`:3616`、`:3827`、`:3867`）。
- 本轮独立复跑（仅一次）：`go test -tags=integration -timeout=20m ./cmd/han-server -run '^(TestCase18RealSessionStoreResolveAndRevoke|TestRecoveryProductionChainComplete|TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix)$' -count=1` → `EXIT=0`，墙钟 `12.292s`，包级 `ok han/cmd/han-server 10.514s`。
- `go test -list` 同正则 → `EXIT=0`，`0.084s`，确认 4 个测试名均存在。
- 保留边界：`main.go:634-650` 的 `recoveryRootInputs` 仍只有 `pool/plan/prepared/closures/seal`，11 个端口未注入；`recovery_runtime.go:456-497,530-531` 逐项返回 `ErrAdmissionClosed`；`s421-handoff.md:15-25,45` 明确 production `Enter/Install` 仍 closed 且“模拟端口注入不证明 production 可达”；`production-chain.log:55` 明确 `simulated_journal_hmac_keyring … windows_unsupported`。

**F003 判定：CLOSED。** 原 finding 的“当前 revision 无法核完整恢复后开流”已被当前 revision + 真实 PG + production root assembly 的 resume→publish→`public_ready` 证据消除。旧 S4.20 哈希未被改写，但已由 S4.21 的 before/after + 独立锁定证据替代；production `Enter/Install` 仍 closed 是 ADR-0030/0031 设计边界，不因本 CLOSED 被解释为可达。

## F004 关键证据

- 新增 `cmd/han-server/case18_session_store_integration_test.go`：`2E29CBFD252B9281D56882F4C8AE8D9868036274E29B70AE8F115F161B183172` / 1,728 B。
- 测试使用真实 PG `session.Store`：`:1` `//go:build integration`；`:19` `TestCase18RealSessionStoreResolveAndRevoke`；`:20` `s420NewCase18World`；`:23-26` `session.NewStore`（`Transactions: world.f.pool`）；`:31-42` 解析活动会话并校验 `principal.UserID`；`:44-46` `store.Revoke(...)`；`:48-55` 同 token 再解析，`err == nil` 即失败。
- 真实 PG 与真实 SQL：fixture `s420NewFixture` 在 `recovery_production_chain_integration_test.go:107`，`:111` 在缺 `HAN_TEST_POSTGRES_DSN` 时 `t.Fatal(... no Skip)`；helper `s420NewCase18World` `:1828`，真实 `session.NewStore` `:1840`，`store.Create` `:1849`；`internal/session/store.go:235` 的 `Store.Revoke` 使用 `:534-537` 的 `revokeSessionSQL`（`UPDATE sessions SET revoked_at = $2, revoke_reason = $3 WHERE token_digest = $1 AND revoked_at IS NULL`）。
- 现有日志 `f004-and-regression.log`：`ADD9EB0CE4AD9389E95878B2C3F0E1CC09EDD9FE195CCC406723749C8D018F41` / 35 B，内容仅 `ok han/cmd/han-server 8.672s`（包级，不含测试名）。
- 本轮独立复跑同上：`EXIT=0`，包级 `10.514s`；`go test -list` 确认 `TestCase18RealSessionStoreResolveAndRevoke` 存在；该测试无 `t.Skip`，包级 PASS 即该测试执行并通过。
- 证据指针漂移：`case18-reconcile-r01/summary.md:39` 记 `8.503s/88A57C7B…`，当前日志为 `8.672s/ADD9EB0C…/35 B`；日志时间戳晚于 summary，故本轮不以 summary 的旧指针作为通过依据。
- 保留边界：新测试只覆盖 store 层 `Resolve`/`Revoke`，不覆盖 HTTP、`AccessGeneral`、账号级 403 或全局 503；第二次解析只断言 `err != nil`，未断言具体错误类型。

**F004 判定：CLOSED。** 原 finding 的核心缺口“缺当前 revision 真实 `SessionStore` SQL/撤销证据”已由真实 PG store 的行为测试补齐；账号级 403 与全局 503 的区分仍成立，但属于 F002/范围决策，不是 F004 store 层证据的缺陷。

## 实际执行命令

| # | 命令 | Exit | Duration | 结果 |
|---:|---|---:|---:|---|
| 1 | `git rev-parse --abbrev-ref HEAD` / `git rev-parse HEAD` | 0 | — | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 2 | SHA256/字节核验（对象与证据文件） | 0 | — | 见 §F003/F004 |
| 3 | `gofmt -l <cmd/han-server/*.go>` | 0 | — | 无输出 |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | 1.567s | 无输出 |
| 5 | `Test-NetConnection 127.0.0.1 5433` | — | — | `True`；`psql` 未安装，未重查 PG 版本 |
| 6 | `go test -tags=integration -timeout=20m ./cmd/han-server -run '^(TestCase18RealSessionStoreResolveAndRevoke|TestRecoveryProductionChainComplete|TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix)$' -count=1` | 0 | 12.292s（墙钟）/ 10.514s（包级） | `ok han/cmd/han-server 10.514s` |
| 7 | `go test -tags=integration -list '^(TestCase18RealSessionStoreResolveAndRevoke|TestRecoveryProductionChainComplete|TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix)$' ./cmd/han-server` | 0 | 0.084s | 4 个测试名均列出 |

NOT RUN：`go build ./...`、`go test -race`/全量套件、真实部署 E2E/双系统、真实 `main` 入口 production `Enter`/`Install`、真实 keyring/HSM/POSIX journal/approved backup inventory、PG `18.6` 的独立 SQL 版本重查。

## 保留限制

1. **production 入口**：`main.go:634-650` 未注入 11 个 closed 端口，实际 `main` 的 `Enter`/`Install` 仍返回 `ErrAdmissionClosed`；F003 CLOSED 不等于 production readiness 或部署可用。
2. **模拟边界**：`production-chain.log:55` 明示 `simulated_journal_hmac_keyring`、`windows_unsupported`；真实 keyring/HSM、POSIX journal/fsync/flock、approved backup inventory 未验证。
3. **测试装配**：`TestRecoveryProductionChainComplete` 通过 `newRecoveryRootAssembly`（`:3011`）+ 测试侧 software 端口驱动，不是部署后的真实服务入口。
4. **F004 范围**：只证 store 层 resolve/revoke；不证 AccessGeneral/HTTP/全局 503；第二次解析未断言错误类型。
5. **S4.20 历史证据**：`b13h-batch-gate-r02/report.md` 仍绑定旧 `900F31…/35832`，本轮未改写；如 Lead 需要该目录内的 superseded 指针，属证据整理动作，不是代码/测试阻断。
6. **证据指针**：`summary.md:39` 的 `8.503s/88A57C…` 与当前 `8.672s/ADD9EB…/35 B` 不一致；以实际当前日志与本轮复跑为准。
7. **未做 Git 写**：未改源码、测试、tasks、Plan、ADR 或既有证据；仅写本目录 `report.md`、`verdict.md`。

---

AI 代理评审，非真人签字；不代替 Lead Gate、人的业务接收或部署授权。
