# identity-auth-S4.17 / B13I 独立复审报告（r02）

- review_id: `s417-independent-review-r02`
- 角色: add-reviewer（非作者独立评审；未修改被审对象、生产代码、任务卡/设计/测试计划）
- 工作区: `D:\AID\VSCode\Codex\Han`
- 分支 / HEAD: `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 对象类型: code（test-only 跨层软件专项）
- 评审方法: 完整读取 `.codex/skills/add-code-review/SKILL.md`、`code-review-checklist.md`、`abstraction-principles.md`；按代码对象加载 `coding-standards`、`no-hardcode-policy`；未启用前端 profile。
- 写入范围: 仅 `.spec/wip/recovery-runtime/evidence-b13i-r01/s417-independent-review-r02/`

## 1. 结论

**PASS（本卡测试对象通过独立复审）**

- r01 的 15 条 finding（H01–H05、M01–M06、L01–L04）经逐条复核，**15/15 CLOSED**。
- 本机独立复跑 7 条验收命令、`gofmt -l`、Case18 `-v`：全部 exit 0，JSON 无 fail、无 skip；5 个 leaf 全部 PASS。
- 该 PASS 仅代表“identity-auth-S4.17 / B13I 的被审测试对象通过独立复审”，**不代表 G4-batch 通过，也不代表人工签收或生产可用**。
- 仍有 2 条非阻断 LOW 观察（N01、N02）和若干残余限制，见第 6、9 节；它们不改变本卡结论，但应登记。

复核要点：ABA/fresh/stale/rollback 的新鲜度判定由真实 PG `article.PostgresReplayEvidenceStore` 产生；consumed 窗口由真实 `recovery.NewRuntime.Enter` → 真实 `identity.NewRecoveryIdentityExecution` → 真实 `recovery.Coordinator.Consume` → 生产 `coordinator.go:275 journal.MarkConsumed` 产生；maintenance gate 使用真实 `recoveryExecutionGate`；Case18 使用真实 `http.ServeMux` 方法路由且 ETag 取首次 GET 响应头；三文件均无 `t.Skip`、无 `DROP SCHEMA`、无硬编码 DSN/5433。

## 2. 对象与 SHA 复核

| 文件 | 字节 | 期望 SHA256 | 本机实测 SHA256 | 判定 |
|---|---:|---|---|---|
| `test/integration/recovery/runtime_admission_test.go` | 1021 | `AF87672131C50D6552C696F445C7C976B30173ABBE1DEDEC2F4151BC539BE158` | 同左 | 匹配 |
| `test/integration/recovery/runtime_recovery_test.go` | 58621 | `AE1A687D092C8B34F6B9C6EF4A63DE6DE183F86326FFF3F16D0DA88BF652C516` | 同左 | 匹配 |
| `test/integration/recovery/runtime_fixtures_test.go` | 105922 | `D27AD62408DEB4121C303296DAFF452A08C669A58934F01E9161D6E0CAFE5593` | 同左 | 匹配 |

三文件 SHA 与字节数均与任务给定值一致，评审期间未发生变化。评审前后均用 `Get-FileHash -Algorithm SHA256` 复核。

## 3. 两轮实际检查范围

**第一轮：对象正确性 / 契约符合性**
- 完整阅读三个被审文件（admission 1021B、recovery 1080 行、fixtures 2697 行）。
- 5 个 leaf 逐项对照任务卡最低事实：genesis/历史重启、跨命名空间 ABA + fresh/stale、F001/F002 幂等与回滚、consumed 窗口与 unknown 隔离、Case18 同版本 503/零下游/开流 200/404-405-Allow/Origin-CSRF-IfMatch。
- 逐 adapter 判定“是否只做介质读写”：`softwareJournal`、`softwareDeletionJournal`、`softwareExclusive`、`softwareLifecycle`、`softwareCommitments`、`softwareSourceVerifier`、`softwareInstallationWriter`、`s417SoftwareRestore`、计数 observer。
- 负路径逐条核对：拒绝、回滚、stale、unknown、503、零下游、重复幂等、未知 run、无 permit 零写入、重复 acquire。
- 检查 `t.Skip`、stub/fake 冒充、跨 leaf 共享可变状态、schema 清理、越界修改。
- 对照生产代码：`internal/recovery/coordinator.go`、`internal/identity/recovery_execution.go`、`internal/identity/deletion.go`、`internal/recovery/runtime.go`、`internal/recovery/filelock_unsupported.go`、`internal/recovery/journal_windows_test.go`。

**第二轮：独立复现 / 反证**
- 本机设置环境变量后独立执行 7 条验收命令 + `gofmt -l` + Case18 `-v`（未采信作者或 Lead 日志）。
- JSON 仅解析 trim 后以 `{` 起始的行；统计 pass/fail/skip 与 subtest。
- 核对 Case18 两次 503、trace 去重、零下游、开流 200、ETag、404/405/Allow、conditional 全链路。
- 核对 schema 保留登记与 DROP 扫描；核对 Lead Gate r02 证据齐备性；核对作者证据与真实代码的一致性。
- 沿真实消费者有界检查：`cmd/han-server` 生产链测试、`article.NewDeletionFinalizer`、`identity.NewRecoveryRunStore`、`identity.NewCanonicalDeleteService`、`article.NewPostgresReplayEvidenceStore`、`article.NewReplayPublicationPorts`。

## 4. D-B13I-02 判定

**前提成立；授权边界成立，但授权仅限“本层测试证明”，不升级介质能力。**

前提（真实、非为放行临时编造）：
- `internal/recovery/filelock_unsupported.go` 存在。
- `internal/recovery/journal_windows_test.go:14` 的 `TestFileRecoveryJournalWindowsUnsupportedHasZeroExternalChanges` 是既有契约。
- 因此 Windows 下真实文件 journal / 文件锁不可运行是既定事实，允许文件与介质端口使用受控软件 adapter。

授权边界逐点核对：
1. ABA/fresh/stale/rollback 由真实 `article.PostgresReplayEvidenceStore.ReadPublishedClosure` 判定：`runtime_recovery_test.go:269/282/293/297/301/306`；真实 `NewReplayPublicationPorts` 在真实 `Coordinator.Consume` 路径被使用（`internal/recovery/coordinator.go:399`）。
2. consumed 窗口走真实 `recovery.NewCoordinator` + 真实 `identity.NewRecoveryIdentityExecution`：`runtime_recovery_test.go:672`（`runtime.Enter`）。
3. maintenance gate 使用真实 `recoveryExecutionGate`：`internal/identity/recovery_execution.go:201`，经 `internal/identity/deletion.go:792` 调用。
4. Case18 使用真实 `http.NewServeMux` 方法路由：`runtime_fixtures_test.go:1995-2000`；无测试自造 Allow。
5. ETag 取首次 GET 响应头：`runtime_recovery_test.go:952`。
6. 禁止 `DROP SCHEMA`：三文件 `rg` 为 0；cleanup 只查 `AcquiredConns()==0` 后 `pool.Close()`。
7. 正向 published closure：A（历史前缀）与 B（精确 fresh）均经真实 store 接受。
8. 真实互斥：`softwareExclusive` 为真实进程内 mutex，第二次 Acquire 返回 `ErrAdmissionClosed`。
9. 真实安装绑定：真实 PG `recovery_installation` 行 + 真实 `recovery.NewInstallationExecutor`；仅文件写入介质为 counter。
10. F002 零写入 5 表 + ACK：users/articles/audit_events/recovery_runs/identity_identifiers + status/version + `AcquiredConns()==0`。
11. Leaf1 真实 `session.NewStore` + `platform.NewHTTPAuthorizer`。
12. trace 计数进入 `if` 断言：`runtime_recovery_test.go:967`。

结论：D-B13I-02 对当前修订可接受。其授权**不能**被表述为 POSIX journal / 真实 CAS / ACL / 批准备份 / E2E 已证明；也不能证明 `canonical_recovery_replay_same_result`（作者已在 `runtime_recovery_test.go:605` 明确登记为 unproven）。

### 4.1 “门禁通过”与“最低事实被真实覆盖”的区分

- 门禁通过：7 条命令 + gofmt 全绿，只说明可编译、可运行、无数据竞争、无 fail/skip。
- 最低事实被真实覆盖：
  - ABA/stale/rollback、namespace/epoch 错配由真实 PG store 返回 `ErrAnchorRollback`/`ErrAdmissionBinding`/`pgx.ErrNoRows`。
  - consumed 窗口唯一性与顺序由真实 runtime + 真实 coordinator + 真实 PG run store 产生；`consumed_marks=1` 且 caller 含 `coordinator.go`。
  - maintenance 窗口由真实 gate 决定；无 permit 时零写入，活 permit 时真实 canonical delete 落库。
  - Case18 的 503、零下游、200、404/405/Allow 由真实 guard/mux/handler/DB 产生。
- 未被真实覆盖（明确保留）：
  - `canonical_recovery_replay_same_result`。
  - 真实 POSIX/CAS/ACL/批准备份/E2E。
  - Case18 精确路由与生产 `main` dispatcher 的整链等价（仅验证方法路由与传输骨架）。

## 5. r01 15 条 finding 最新状态

| ID | r01 级别 | r02 状态 | 新证据 / 位置 | 判定摘要 |
|---|---|---|---|---|
| H01 | HIGH | **CLOSED** | `runtime_recovery_test.go:269/282/293/297/301/306`；adapter CAS 仅保留在 `:338-341` 并标注 `:226` `adapter_scoped=anchor_cas; not real_cas/journal` | fresh/stale/rollback/namespace/epoch/tampered 全部由真实 `PostgresReplayEvidenceStore` 决定；adapter CAS 明示非证据。 |
| H02 | HIGH | **CLOSED** | `runtime_recovery_test.go:672` 真实 `runtime.Enter`；`:682` `consumedMarkCount()==1`；`:685-686` caller 含 `coordinator.go`；`:742` 二次 Consume 失败；`:746` 未知 run 失败；生产 `coordinator.go:275` | consumed 窗口由真实 runtime→identity execution→coordinator→`journal.MarkConsumed` 产生；重复/未知负路径真实。 |
| H03 | HIGH | **CLOSED** | `runtime_fixtures_test.go:1995-2000` 真实 `http.NewServeMux` 方法模式；`:2016-2036` 仅重写 mux 文本 404/405 且保留 `Allow`；断言 `DELETE, GET, HEAD, PUT` | 405/Allow/404 由真实 mux 产生，不再手搭 switch。 |
| H04 | HIGH | **CLOSED** | `runtime_recovery_test.go:509-540` 无 permit 拒绝且零写入；`:541-612` 活 permit 经真实 gate 走真实 canonical delete；`recovery_execution.go:201`→`deletion.go:792` | 真实 `recoveryExecutionGate`；无 permit 零写入；活 permit 断言 status/deleted、decision、audit=1。 |
| H05 | HIGH | **CLOSED** | `runtime_fixtures_test.go:1242-1300` 单一 `deletionJournal` 同时供 finalizer/AnchorReader/Snapshots；`runtime_recovery_test.go:143-150` 直接核对 finalizer 写入 head 等于 Load head | 删除手工 `advance` 对齐，单权威 journal。 |
| M01 | MEDIUM | **CLOSED** | `runtime_recovery_test.go:269-290` 正向 closure A（历史前缀，CurrentHead=B）与 B（精确 fresh）经真实 store 接受 | 正向 published closure 已覆盖。 |
| M02 | MEDIUM | **CLOSED** | `runtime_recovery_test.go:204-215` 两命名空间 id/namespace 独立；`:296-299` 真实 store 拒绝跨 journal publication；`:314-322` 文章 ABA 后 recovery head 不变 | 增加可失败的真实 store 跨命名空间检查；残余：journal 字节仍为软件介质。 |
| M03 | MEDIUM | **CLOSED** | `runtime_fixtures_test.go:439` 真实 mutex；`runtime_recovery_test.go:770-783` 第二次 Acquire→`ErrAdmissionClosed` 且 rejection+1 | 独占语义与负路径已证；残余：进程内 mutex 非 OS 文件锁。 |
| M04 | MEDIUM | **CLOSED** | `runtime_recovery_test.go:797-840` 三 worker Stop 先阻塞、释放后 join、holders=0、lifecycle_stop=1；`runtime_fixtures_test.go:203-220` 真实 PG 安装行 + `:1270` 真实 executor | 真实 join；真实 PG 安装绑定；writer 仅文件介质 counter。 |
| M05 | MEDIUM | **CLOSED** | `runtime_recovery_test.go:381-436` 5 表 + status/version + `AcquiredConns()==0`；`:614-629` 篡改 tombstone ACK withheld、无部分提交 | F002 零写入与原子回滚已覆盖。 |
| M06 | MEDIUM | **CLOSED** | `runtime_recovery_test.go:161-198` 真实 session/authorizer（匿名 401 `auth.session_required`，认证 204）；`:32` 端口日志含 session/authorizer | Leaf1 身份链真实。 |
| L01 | LOW | **CLOSED** | 三文件 `rg 'DROP SCHEMA|DROP TABLE|TRUNCATE'` 为 0；`runtime_fixtures_test.go:163-171` 仅查连接数后 Close | 无自动 DROP，schema 保留登记。 |
| L02 | LOW | **CLOSED** | `lead-gate-r02/cmd-04.log`、`cmd-04b.log`、`gofmt.log` 均存在（0 字节） | Lead Gate 已补齐。 |
| L03 | LOW | **CLOSED** | `runtime_recovery_test.go:952` ETag 取首次 GET 响应头；Allow 由 mux 计算 | 不再测试自造 oracle。 |
| L04 | LOW | **CLOSED** | `runtime_recovery_test.go:967` `trace != 1` 进入 `if` 条件 | trace 计数已进入断言。 |

统计：**15 CLOSED / 0 PARTIAL / 0 OPEN**。

## 6. 新增非阻断观察（LOW，不改变 verdict）

- **N01（LOW，open，非阻断）**：`runtime_fixtures_test.go:138-166` 把保留 schema 登记写入硬编码路径 `../../../.spec/wip/recovery-runtime/evidence-b13i-r01/author-r02/retained-schemas.json`。
  - 影响：测试执行会写入作者证据目录（本次复审复跑使登记从 165 增至 182，均为追加去重，未 DROP）；若工作区缺少该目录或不可写，测试会失败；跨运行共享可变文件。
  - 修复方向：改为测试自有临时路径或环境变量指定，避免运行时写作者证据；或明确该登记由 Lead/作者在收口时统一生成。
- **N02（LOW，open，非阻断）**：`runtime_fixtures_test.go:1360-1364` 注释称 “runtime-driven consumed window remains unproven because RunNormal never invokes Consumer…”。
  - 事实：该注释位于 `s417FullWorld`（其 runtime 未注入 Consumer），对 full world 成立；但整体看已过时，Leaf4 的 `s417NewRecoveryWorld` 已证明 consumed 窗口。
  - 修复方向：把注释限定为 “s417FullWorld 范围”或删除过时表述，避免与 `runtime_recovery_test.go:605` 的 unproven 登记混淆。

## 7. 5 个 leaf 逐项判定

| Leaf | 判定 | 说明 |
|---|---|---|
| GenesisDeletionHistoricalRestart | **PASS** | 真实 install/finalizer/session/authorizer；同权威 journal 历史重启 `public_ready/verified`；无 recovery 端口 Resume 显式 `ErrAdmissionBinding`；Stop 后 holders=0。 |
| CrossNamespaceABAAndExactPublication | **PASS** | 真实 coordination 错配拒绝；真实 PG store 决定 fresh/stale/rollback/namespace/epoch/tamper；正向 A/B；真实 publication 行核对；adapter CAS 明示非证据。 |
| F001F002AndCanonicalIdempotencyReview | **PASS** | F001 非规范标识拒绝零写入；真实 canonical delete；日治理重复幂等拒绝；F002 缺能力/无 permit 零写入；活 permit 真实 gate 落库；篡改 tombstone 拒绝且不修复。 |
| CrashConsumedWindowAndDrain | **PASS** | 真实 `runtime.Enter` 产生唯一 consumed 窗口；`consumed_marks=1` 且 caller 含 `coordinator.go`；重复/未知拒绝；unknown 写故障隔离；三 worker join。 |
| Case18SameRevisionAndNoProductionClaim | **PASS** | 真实 mux/session/authorizer/recovery guard/origin guard/query handler；两次 503 且 trace 不同、零下游；开流 200、ETag 来自首次 GET；404/405/Allow；Origin/CSRF/If-Match/204。 |

## 8. 独立复跑与真实计数

环境（不记录 DSN 值）：`HAN_TEST_POSTGRES_DSN` / `HAN_B13H_ROOT_DSN` → `127.0.0.1:53519/han_b13h_root`（用户 `han_b13h`，trust）；`HAN_B13H_BACKUP_DSN` → `han_b13h_backup`。未占用 5433，未启停 PG。

| ID | 命令 | exit | 真实计数 |
|---|---|---:|---|
| cmd-01 | `go test -tags=integration ./test/integration/recovery -list '^TestRecoveryRuntimePostgres$'` | 0 | 列出 `TestRecoveryRuntimePostgres` |
| cmd-02 | `go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | 92 JSON 行；pass 7 / fail 0 / skip 0；5 个 subtest 全 PASS |
| cmd-03 | `go test -race -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | 92 JSON 行；pass 7 / fail 0 / skip 0；5 个 subtest 全 PASS |
| cmd-04 | `go vet -tags=integration ./test/integration/recovery` | 0 | 空输出 |
| cmd-05 | `go test -tags=integration ./cmd/han-server -list '^TestRecoveryRuntimeProductionChain$'` | 0 | 列出 `TestRecoveryRuntimeProductionChain` |
| cmd-06 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json` | 0 | 10 JSON 行；pass 2 / fail 0 / skip 0 |
| cmd-07 | `go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json` | 0 | 10 JSON 行；pass 2 / fail 0 / skip 0 |
| gofmt | `gofmt -l <三文件>` | 0 | 0 行输出 |
| cmd-08 | `go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$/^Case18SameRevisionAndNoProductionClaim$' -count=1 -v` | 0 | PASS |

合计：pass 事件 18 / fail 0 / skip 0。JSON 解析规则：仅解析 trim 后以 `{` 起始的行。

Case18 `-v` 实测（真实日志）：
- closed：`statuses=[503 503]`，两个不同 trace ID；`article_handler=0 article_service=0 article_repository=0 article_author_reader=0`。
- open：`status=200`，`etag="article-<uuid>-v1"`，`handler=1 trace=1 service=1 repository=1 author_read=1`。
- routing：`missing_article=404/article.unavailable`、`unknown_route=404/request.route_not_found`、`unsupported_method=405/request.method_not_allowed`、`allow="DELETE, GET, HEAD, PUT"`。
- conditional：`origin_missing=403/auth.origin_rejected`、`csrf_missing=403/auth.csrf_failed`、`if_match_missing=428/request.precondition_required`、`stale_if_match=412/article.precondition_failed`、`accepted=204`、`remaining_articles=0`、`completed_operations=1`、`audit_events=1`。

Schema 保留：复跑前登记 165 条，复跑后 182 条（+17，均由本次执行新建并追加登记）；182 条全部以 `han_b13i_s417_` 前缀、唯一、无重复；未执行任何 DROP/清理。作者 r03 记录 147、Lead r02 记录 164，差额来自各自复跑新建 schema，符合“保留不 DROP”口径。

## 9. 未验证风险与限制

- `canonical_recovery_replay_same_result` 仍未证明（作者已在 `runtime_recovery_test.go:605` 登记；本轮不将其计为已覆盖）。
- Windows 软件介质 adapter 不证明真实 POSIX 文件 journal / CAS / ACL / 批准备份 / E2E；本报告的 PASS 不得被引用为这些能力已证明。
- Case18 仅验证真实 `http.ServeMux` 方法路由与传输骨架；与生产 `main` dispatcher 的整链等价未证明。
- 归属 unknown：工作树 `go.mod` modified（+14 行），`test/`、`internal/`、`cmd/` 整体 untracked；无法用 git diff 证明无越界改生产代码，仅能用三文件 SHA 固定被审对象。
- 环境为单实例 local trust PostgreSQL（`127.0.0.1:53519`）；未验证多实例、网络分区、故障切换。
- 仅单次 `-race`，无 soak、无并发压力测试。
- 未运行 `goimports` / `golangci-lint`（本卡口径仅要求 `gofmt -l` 与 7 条命令）。
- 既有 `TestRecoveryRestoreTargetReadbackPostgres` 的环境性失败不在本卡范围，未作为本卡回归处理。
- 未评审本批次其他文件与历史修订；未评审生产实现正确性，仅评审测试是否真实覆盖最低事实。

## 10. 复审 / 关闭条件

- 本卡测试对象：无阻断项，可提交 Lead 做结果确认与人工签收。
- 若 Lead 要求关闭 N01/N02：建议下批把 schema 登记改为测试临时目录/环境变量，并清理 `s417FullWorld` 过时注释；不影响本卡 PASS。
- 若后续要升级结论到 G4-batch / 生产可用：必须另行提供真实介质（POSIX journal/CAS/ACL/批准备份）、`canonical_recovery_replay_same_result`、生产 dispatcher E2E 与 soak 证据；本报告不覆盖。
- 任何被审文件 SHA 变化都必须重新评审；本 PASS 绑定第 2 节 SHA。

## 11. 证据索引

- 本目录：`report.md`、`receipt.json`、`cmd-01.log`…`cmd-07.log`、`cmd-08-case18-verbose.log`、`gofmt.log`、`rerun-exit-codes.json`、`partial-codes-1.json`、`partial-codes-2.json`
- 作者证据：`.spec/wip/recovery-runtime/evidence-b13i-r01/author-r02/`
- Lead Gate：`.spec/wip/recovery-runtime/evidence-b13i-r01/lead-gate-r01/`、`lead-gate-r02/`
- 方法：`.codex/skills/add-code-review/SKILL.md`、`.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`、`.codex/guidance/coding-standards.md`、`.codex/guidance/no-hardcode-policy.md`