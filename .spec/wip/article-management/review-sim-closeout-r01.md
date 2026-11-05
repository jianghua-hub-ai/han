# S1.5/S4.4 模拟环境收口 独立评审 R01

**角色**：Confucius / add-reviewer（非作者独立评审）
**评审日期**：2026-10-31（Asia/Shanghai）
**评审对象**：当前工作树 revision 的 4 个测试文件（S1.5 恢复隔离矩阵 + S4.4 测试收口）
**结论**：**PASS WITH MINOR**（4 条 findings：1 MEDIUM + 3 LOW；无阻断项）

## 1. 授权与代理声明

- 授权来源：ADR-0028（`.add/project/knowledge/decisions/framework.md:815-857`，本节实测第 815 行为 `<a id="adr-0028"></a>`、第 852 行为约束 6「任务关闭」）+ 用户 2026-10-31 项目级授权原话：「作为项目责任人，我确认本项目为测试项目……需要我来评审的也授权Agent或AI评审……需要我来签收的授权Agent或AI签收；如果因为确实环境导致阻断，授权采用模拟环境或建议的环境代替」。
- 代理声明：**AI 代理评审，非真人签字**。本报告不冒充非作者真人评审，不代替人的结果接收或 Lead Gate 决定。
- 评审边界：只评 S1.5「恢复隔离矩阵」与 S4.4「测试收口」这 4 个测试文件及其证据；**不对 S1.5 整卡、S4.4 整卡下 PASS**，不宣称 production readiness。
- 只读约束遵守：未修改任何源码、测试、证据、tasks.md、ADR 或 design；未执行任何 Git 写操作；未删除/移动任何证据；未运行会追加保留资源的 ACL 测试。

## 2. 评审基线（实测）

| 项 | 值 |
|---|---|
| 工作区 | `D:\AID\VSCode\Codex\Han` |
| 分支 | `main` |
| HEAD | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `git status --porcelain` | 286 行（全仓库均未跟踪，无法用 Git 证明写集） |
| `git ls-files <相关目录>` | 0（同上） |

被评审 4 文件（SHA256 / 字节 / 最后写入）：

| 文件 | SHA256 | 字节 | 写入时间 |
|---|---|---|---|
| `cmd/han-server/recovery_isolation_matrix_integration_test.go` | `B11A93281FEE1697B2E2520390056DBB34B827944409FE387AF90FFC9659166E` | 5647 | 19:23:35 |
| `internal/article/deletion_journal_hash_test.go` | `10F940D8C18984BD5D806BB9D5D6379DC644B22ACE626EB2BF6E4A59AC33FB48` | 1349 | 19:22:52 |
| `test/integration/recovery/article_replay_test.go` | `1BD2B0ED9385973533C611EEEF9CD17A9B70149AFBD5339829B9EDEF9906B0CD` | 86718 | 19:22:52 |
| `test/integration/recovery/old_backup_restore_test.go` | `36B13366F968FF9BA1BD1AA00AF90458A7EE89DD3A7FCBDFE69F6FF07317E536` | 15482 | 19:22:52 |

相关生产/部署文件（用于判断是否被本轮触碰）：

| 文件 | SHA256 | 字节 | 说明 |
|---|---|---|---|
| `internal/article/deletion_journal.go` | `0A3F2AE9AEADDDB85C62A1F7CA1BD284AACF149DA0D5ED02184F1E30F0EB4333` | 25420 | 内容未变（与既有基线一致） |
| `internal/article/replay_publication.go` | `34B698A231B32101ACBF1B91AB7843FF1710D529C748072CFDF27BA1CF180B8E` | 37500 | 本轮未改 |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | 19:26:09 由并发 S4.20 写者修改，非本交付写集（见 F004） |
| `cmd/han-server/recovery_runtime.go` | `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` | 35832 | 19:25:50 由并发 S4.20 写者修改（同上） |
| `deploy/db/provision.sql` | `5252556099A335A480041A62ED30C7618384A5B4D038BFA29356423C446B8C23` | 2333 | 未变 |
| `deploy/db/grant-application.sql` | `FC0D229818D48407EF4CCBF97D0860E08E781FF8EF102201C551560741AF0AE8` | 1229 | 未变 |

证据目录（实测存在，未改动）：

| 文件 | SHA256 | 字节 |
|---|---|---|
| `.spec/wip/article-management/evidence-s1.5/sim-closeout-r01/index.json` | `6AAF8B0D8ACF7852071A4E993F3E9CC9C278748FF6BE29B1A0C0EDC9696BCE4E` | 6884 |
| 同上 `simulation-boundary.md` | `89A9890B6F24626482F7C2B0C87DBF7722FC7750AB1CF343F7CBEB180F83FD00` | 1494 |
| 同上 `change-hunks.md` | `3449A71896387E70B0F98F251B1ABA9BB2E1FAA0CB7494099235EA891C608559` | 956 |
| 同上 `commands.ndjson` | `70573AE07665E5D23203AC10CCED5EE2F4D8130279A726CB2EB9C987C61F5B2D` | 1505 |
| `.spec/wip/article-management/evidence-s4.4/sim-closeout-r01/index.json` | `D2A07736C33F874F5044E09825894CED64CCD30A7519AC2F34F890C9FCEF27C0` | 7205 |
| 同上 `simulation-boundary.md` | `D4AA9FAD44DC513514F587442ACBE25069DFA02622783C22BCA9629230F65B05` | 1562 |
| 同上 `change-hunks.md` | `4A75EDED78E5449C9AF1E9AAB32087A313BCBE7245DF1F4496192D4FE1D5310F` | 1366 |
| 同上 `commands.ndjson` | `33F2CA5310B0E3A54CA046306EE8B0AE8B514A99B1AF4C4EFE64E54589FEF5A1` | 1111 |

## 3. 逐对象核验

### 3.1 S1.5 恢复隔离矩阵（`recovery_isolation_matrix_integration_test.go`）

- **路由集合与生产一致**：测试 `recoveryIsolationBusinessRoutes()`（`:83-111`）手写 25 条 method+path；与 `cmd/han-server/main.go:3554-3659` 的生产 switch 逐条比对，25 条无遗漏、无多余（含 `/api/v1/articles` 的 GET/POST、`/api/v1/articles/{id}` 的 GET/PUT/DELETE、admin 与 orphan 各 GET/DELETE）。
- **空转排除**：未注册路由在 `main.go:3686-3692` 先判定并返回 404；测试 `:37-40` 明确把非 503（含 404）判失败，因此不存在「handler 未接线也通过」的空转。
- **guard 位置正确**：guard 包裹整个 router（`main.go:3694-3696`），业务 handler（含 login）在 guard 之后；`internal/platform/recovery_guard.go:33-46` 先 `Acquire` 再 `next.ServeHTTP`，closed/unknown 走 `writeRecoveryHTTPClosed` → 503 + `CodePlatformRecoveryInProgress`（`:77-83`）。因此从代码结构上，guard 确实先于 authorization/origin/session/repository。
- **逐路由断言**：每子测试断言 503（`:37`）、envelope code（`:45`）、envelope Trace 与 `X-Trace-ID` 一致（`:48-51`）、泄漏断言（`:52`）；矩阵结束断言 Article repository `lists+gets==0`、author reader `calls==0`、session resolver `inputs==0`（`:56-64`），即 session/repository/reader 均为零调用，属直接观测。
- **计数与探针**：`acquires==len(routes)`、`releases==0`（`:65-68`），`len(events)==routes+2` 且事件仅 `trace`（`:69-72`）；探针 `/livez` 200 alive、`/readyz` 503 unavailable（`:26-27`），因探针在 `main.go:3677-3685` 于 guard 之前分流，acquire 计数不含探针，测试成立。
- **未覆盖点（见 findings）**：authorization/origin 未被计数观测（fixture `configure=nil`，seam 未设置 `authorized`/`protectOrigin`）；矩阵本身无 guard-open 正控；泄漏断言含随机可命中的子串。

### 3.2 S4.4 固定摘要（`deletion_journal_hash_test.go`）

- 测试构造 tombstone（`:14-26`），期望 `sha256:4ce06ce79ad8639d2468f0774d6d2d2097909bab6203a2dd5f63701d2f4bfacf`（`:27`），断言 `deletionRecordHash`（`:28`）。
- **独立复算（未调用 Go 实现）**：按 `deletionDiskRecord` 字段序（`internal/article/deletion_journal.go:730-742`）构造 JSON：`namespace=article-deletion/v1, sequence=7, article_id=10000000-…-0007, article_version=3, deletion_operation_id=20000000-…-0007, deleted_at=2026-10-31T12:34:56.123456789Z, key_id=key-7, key_version=1, stage=tombstoned, previous_record_hash=sha256:aa…(64), record_hmac=hmac-sha256:bb…(64)`，UTF-8 475 B；实测 SHA256 = `4ce06ce79ad8639d2468f0774d6d2d2097909bab6203a2dd5f63701d2f4bfacf`，与期望一致；JSON 中无 `&`/`<`/`>`，不存在 HTML 转义差异（Go `encoding/json` 仅对这三类转义）。
- **与生产投影自洽**：`deletionRecordHash`（`:690-694`）序列化的是 `deletionDiskRecordFromModel`；`DeletedAt` 用 `UTC().Format(time.RFC3339Nano)`（`:773`）；`deletionTombstoneStage`/`ArticleDeletionNamespace` 与测试取值一致。该期望值确为按文档化字段序独立计算，不是照抄实现输出。
- **钉住效果**：字段序、时间格式、命名空间/阶段任一漂移都会改变摘要；`rg` 确认快照写入（`deletion_journal_anchor.go:792/798`、`deletion_journal_snapshot.go:273/335`、`deletion_journal.go:545`）共用同一投影，因此该单测对磁盘投影有真实防护价值。

### 3.3 CheckPhase（`article_replay_test.go`、`old_backup_restore_test.go`）

- **purpose 集合与生产一致**：生产 replay 用 `recovery.PhaseRecoveryReplay`（`internal/recovery/runtime.go:19`，`replay_publication.go:663/709`），install 用未导出常量 `replayPublicationInstall="install"`（`replay_publication.go:25`，调用点 `:762`）；测试常量 `s44PublicationInstallPhase="install"`（`article_replay_test.go:48`）并注释说明来源（`:515-517`）。合成 `CheckPhase` 接受两者、其余返回 `ErrAdmissionBinding`，nil ctx 返回 `ErrAdmissionBinding`，ctx 取消返回 `ctx.Err()`（`article_replay_test.go:518-535`；`old_backup_restore_test.go:368-385`），语义正确。
- **但未被任何被评审测试执行（F003）**：`test/integration/recovery` 内没有任何 `EnsureCurrentPublication` / `ImportLegacyPublication` 调用（`rg` 全目录 0 命中，仅有构造与 `Publish/Begin`），也没有通过 `recovery.Coordinator` 走 publication 端口（该包内 Coordinator 使用的是真实 admission runtime，`runtime_fixtures_test.go:1126-1129` 传入 `rt`）。因此 `CheckPhase` 当前是「可编译的镜像」，不构成对 purpose 绑定的证明。
- **panic 边界**：两个合成 Runtime 都嵌入 nil `recovery.Runtime`（`article_replay_test.go:494-497`、`old_backup_restore_test.go:346-348`），`internal/recovery/runtime.go:227-245` 共 16 个方法，除已实现的 `ReadPermitBinding`/`CheckPhase` 外调用即 nil panic。当前路径不可达（`checkReplayImport` 先 `CheckPhase` 再 `ReadPermitBinding` 再 `CheckArticleCoordination`，但入口方法从未被调用），S4.4 `simulation-boundary.md` 已披露该已知边界，不影响结论；若将来接入真实调用则需补实现。

### 3.4 old backup 去重（`old_backup_restore_test.go`）

- `s44OldBackupRecordHash` 已改为 `return s44PublicationRecordHash(record)`（`:338-342`），`rg s44OldBackupDiskRecord` 在仓库内 0 命中，重复的磁盘投影已删除；`s44PublicationRecordHash`（`article_replay_test.go:472-490`）的匿名结构与生产 `deletionDiskRecord` 字段序完全一致。
- import 清理后 `go vet -tags=integration` 通过（未使用 import 会编译失败），可反证 `encoding/json`、`time` 已从该文件移除且无残留引用。

## 4. Findings

| 编号 | 级别 | 证据 | 影响 | 建议 |
|---|---|---|---|---|
| `S15-SIM-F001` | MEDIUM | `recovery_main_dispatcher_integration_test.go:325-333` 的禁词表含 `"run"`、`"ref"`，而 `s419AssertNoRecoveryLeak` 扫描整个响应体，响应体含 `trace_id`（`internal/platform/errors.go:163-168`）；trace ID 为 26 位 Crockford base32（`internal/platform/logging.go:15-17`，字母表含 `R/E/F`）。实测 50,000 个随机 trace 中 38 个含 `ref`（0.076%）；每轮矩阵产生 54 个响应（2 探针 + 25×2 路由），整轮至少一次误报概率 ≈ 4.0% | 集成测试随机失败（约每 25 次运行 1 次），会污染 CI 与核收证据，且失败信息误导为「响应泄漏内部标记」 | 只扫描结构化字段（`envelope.Code`/`envelope.Message`），或在扫描前剔除 `trace_id`；禁词改为具体内部标记值（如 `backupRef`、`runID` 的实际字面量），不要用 `ref`/`run` 这类短子串 |
| `S15-SIM-F002` | LOW | 矩阵调用 `newS419ProductionDispatcher(t, runtime.handle, nil)`（`recovery_isolation_matrix_integration_test.go:46`），`configure=nil`；fixture seam 只设置 `principalResolver`/`shared`/`trace`（`recovery_main_dispatcher_integration_test.go:256-266`），未设置 `authorized`/`protectOrigin`，`main.go:3732-3735` 遂直接返回 `next`、`main.go:3509-3513` 用生产 originGuard | 「guard 先于 authorization/origin」缺少直接观测，只能由 503+code 与 session/repository/reader 零调用间接推断；矩阵自身也没有 guard-open 正控（仅 `TestRecoveryMainBusinessDispatcher` 覆盖 2 条 admin 路由），若 guard 退化为「无条件 503」，矩阵仍全绿 | 在 fixture 的 `configure` 中设置 `authorized`/`protectOrigin` 计数并断言均为 0；再补一个 `public_ready` 对照组，断言 25 条路由不再返回 `CodePlatformRecoveryInProgress`（可为 401/403/400 等业务错误），以排除「恒 503」自证 |
| `S15-SIM-F003` | LOW | `s44PublicationRuntime.CheckPhase`（`article_replay_test.go:518-535`）与 `s44OldBackupRuntime.CheckPhase`（`old_backup_restore_test.go:368-385`）在被评审测试中不可达：`test/integration/recovery` 无 `EnsureCurrentPublication`/`ImportLegacyPublication` 调用（rg 0 命中），`checkReplayImport`（`replay_publication.go:761-776`）与 `EnsureCurrentPublication`（`:655-719`）均未执行 | purpose 镜像正确但未被验证；测试通过不能证明该实现，未来若有人接入调用还可能在 `CheckArticleCoordination` 处 nil panic | 增加一个直接调用 `CheckPhase` 的单元测试（覆盖 replay/install/其他 purpose、nil ctx、取消 ctx），或在注释中标注「本方法当前未被测试执行」；如需真正接入，应显式实现剩余 Runtime 方法返回 `ErrAdmissionBinding` 而非嵌入 nil |
| `S15-SIM-F004` | LOW | 证据命令 `ran_at_utc` 为 11:25:21Z–11:28:05Z（本地 19:25–19:28，`.spec/wip/article-management/evidence-s1.5/sim-closeout-r01/commands.ndjson`），而 `main.go` 19:26:09、`recovery_runtime.go` 19:25:50 被并发 S4.20 写者修改（`.spec/wip/recovery-runtime/evidence-b13h-r01/s420-implementation-r01/progress.md:22/34` 记录 `1D47CC94→7E2F94DF→6D5373CA` 与 `47CCE68A→900F31F5`）；工作树全部未跟踪，`git ls-files` 0，无法用 Git 绑定写集 | 部分证据命令在漂移前执行，「`production_code_changed:false`」只能读作「本交付写集未改生产代码」，不能读作「工作树生产文件未变」；证据与 revision 的可审计绑定偏弱 | Lead 冻结 revision 后重跑一次关键命令并登记 revision 与哈希；closeout 文案改为「本交付未改生产代码；评审窗口内 `main.go`/`recovery_runtime.go` 存在 S4.20 并发改动」。本轮已在漂移后 revision 复跑，结果仍 PASS（见 §6） |

新问题检查结论：未发现 REVOKE/DSN/`aclexplode`/fallback 遍历等生产路径被本交付改动（本轮只动测试）；`s44PublicationRecordHash` 委派未改变旧备份路径的摘要投影；未发现连接泄漏或资源新增（测试使用 `httptest` 与既有 fixture，cleanup 已由 fixture 注册）。

## 5. 本次证明

- 在 HEAD `6da8abb` 的当前工作树 revision 上，`go build ./...`、`gofmt -l`、`go vet`（article / recovery integration / cmd integration）全部退出码 0。
- `go test ./internal/article -count=1` 通过；固定摘要测试实际执行并通过，且期望摘要经独立 PowerShell/.NET 复算一致（475 B，无 HTML 转义）。
- `go test -tags=integration ./test/integration/recovery -run "TestArticleReplayPostgres|TestOldBackupRestore|TestArticleReplayEvidencePostgres"` 在本机真实 PostgreSQL 上通过（20.054s）；old backup 去重后两条路径仍通过。
- `go test -tags=integration ./cmd/han-server -run "TestRecoveryMainBusinessDispatcher|TestRecoveryIsolationMatrix"` 通过：25 条业务路由 × closed/unknown 全部 503 + `CodePlatformRecoveryInProgress` + Trace 一致；session/repository/reader 零调用；`acquires==25`、`releases==0`；探针不进 guard。
- 4 个被评审文件格式正确、vet 通过；S8.2 九文件写集、`cmd/han-server/main.go`（本交付）、`deploy/db/*` 未被本交付改动（哈希与既有基线一致）。

## 6. 本次不证明

- 真实双系统全局恢复隔离 / Case18：`NOT RUN / unknown`（需两套真实系统）。
- 真实 POSIX Journal `fsync`/`flock`、真实 keyring/HMAC runtime、真实旧备份恢复与 RPO/RTO。
- identity 侧双 namespace、`MarkConsumed`、联合流行为。
- 真实部署/CI、多节点生产 superuser 绕过、凭据分发/轮换、备份恢复演练。
- 「生产代码本轮未改」的 Git 级证明（工作树未跟踪；见 F004，仅由 S4.20 进度记录与 `change-hunks.md` 交叉印证）。
- S1.5 ACL 子集未在本轮独立复跑（按任务要求禁止追加保留资源），其结论只引用既有 `acl-r03` 证据，不构成本报告的新证明。
- 不宣称 production readiness；不代表 S1.5 / S4.4 整卡关闭，不代表 Lead Gate 通过。

## 7. ADR-0028 约束 6 一致性

- 两份 `simulation-boundary.md` 均显式区分「实际操作 / 模拟部分 / 未真实验证/unknown / 不宣称 production readiness」，S1.5 记录 Case18 为 `NOT RUN`，S4.4 记录合成端口未实现方法会 panic 的已知边界，与约束 3「不得伪造 PASS」、约束 6「登记未验证边界与残余风险」一致。
- 证据目录只新增文件；未发现删除失败证据、改写历史结论或把 `blocked` 直接改成 `completed` 的痕迹（`commands.ndjson` 与日志文件名/时间连续，`retained-resources-snapshot.md` 仍保留）。
- S1.5 ACL 子集此前 R03 的 PASS WITH MINOR 与本轮结论不冲突；本报告未改动该结论。

## 8. 实际执行命令与退出码

| 命令 | 退出码 | 结果 |
|---|---|---|
| `git rev-parse HEAD` | 0 | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `git status --porcelain` | 0 | 286 行（全未跟踪） |
| `git ls-files cmd/han-server internal/article test/integration/recovery deploy/db` | 0 | 0 行 |
| `go build ./...` | 0 | 无输出 |
| `gofmt -l <4 文件>` | 0 | 无输出 |
| `go vet ./internal/article` | 0 | 无输出 |
| `go vet -tags=integration ./test/integration/recovery` | 0 | 无输出 |
| `go vet -tags=integration ./cmd/han-server` | 0 | 无输出 |
| `go test ./internal/article -count=1` | 0 | `ok han/internal/article 1.491s` |
| `go test -tags=integration ./test/integration/recovery -run "TestArticleReplayPostgres\|TestOldBackupRestore\|TestArticleReplayEvidencePostgres" -count=1` | 0 | `ok han/test/integration/recovery 20.054s` |
| `go test -tags=integration ./cmd/han-server -run "^(TestRecoveryMainBusinessDispatcher\|TestRecoveryIsolationMatrix)$" -count=1` | 0 | `ok han/cmd/han-server 0.547s`；两测试 PASS，25×2 子测试全绿 |
| 摘要独立复算（PowerShell/.NET SHA256） | 0 | 475 B → `4ce06ce7…4bfacf`，与期望一致，无 HTML 特殊字符 |
| trace 误报模拟（50,000 样本） | 0 | 38 个含 `ref`（0.076%），整轮 54 响应约 4.0% 误报概率 |
| ACL 测试 | 未运行 | 按任务要求禁止追加保留资源 |

注：一次带 `Select-String | Select-Object -First` 的管道截断曾产生虚假 `EXIT=-1`；改为完整捕获后重跑退出码为 0，表中采用 0。本轮未使用真实 PG DSN 之外的敏感信息，报告中不含口令。

## 9. 可核收判断

- **S1.5 恢复隔离矩阵子集：可核收**（附 F001 修复建议）。矩阵对 25 条受保护路由完成了 closed/unknown 的 503 fail-closed、错误码与 Trace 一致性、下游零调用、acquire/release 计数与探针豁免的实测，404 误通过已被显式排除。
- **S4.4 测试收口子集：可核收**（附 F003 建议）。摘要钉住值独立可信，old backup 去重未改变投影，CheckPhase 语义正确但当前未被执行。
- **整卡关闭 / Lead Gate：不由本报告决定**。建议 Lead 在冻结 revision 后登记：F001 的 flake 风险、F002 的正控缺口、F004 的并发写者事实，并按 ADR-0028 约束 6 在关闭记录中保留「模拟/未真实验证」边界。

## 10. 报告元信息

- 报告路径：`.spec/wip/article-management/review-sim-closeout-r01.md`
- 编码：UTF-8（无 BOM）
- 评审性质：**AI 代理评审，非真人签字**
- 本报告不构成对 S1.5/S4.4 整卡的关闭结论，也不代表 Lead Gate 通过。