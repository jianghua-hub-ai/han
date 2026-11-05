# S4.20 独立代码评审报告（s420-code-review-r01）

- 评审角色：add-reviewer（非作者独立评审）
- 对象：工作卡 `identity-auth-S4.20` 本轮 5 个 D-6 叶 + 共享 helper `s420RunProductionResume`
- 固定工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审时间：2026-10-31（Asia/Shanghai）
- 报告位置：`.spec/wip/recovery-runtime/evidence-b13h-r01/s420-code-review-r01/report.md`
- 授权边界：只读被评对象；仅在本评审证据目录写报告与命令日志；未执行任何 Git 写、删除、移动或归档。

## 0. 基线与实际读取

被评文件（实测 SHA256 / 字节 / mtime）：

| 文件 | SHA256 | 字节 | mtime |
|---|---|---|---|
| `cmd/han-server/recovery_production_chain_integration_test.go` | `535D6A041F7A6131A1117A9266708178C2F1F2A35598B24120C01C4645A61D6A` | 162632 | 2026-10-31 20:26:53 |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | 2026-10-31 19:26:09 |
| `cmd/han-server/recovery_runtime.go` | `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` | 35832 | 2026-10-31 19:25:50 |
| `cmd/han-server/recovery_runtime_integration_test.go` | `B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A` | 31638 | 2026-10-31 03:30:10 |
| `.add/project/knowledge/tasks/tasks.md` | `351E5727BF4AA0C4FD58425C15BFBBE4007C8A04176E1D106C41F1C18D881C0C` | 742305 | 2026-10-31 19:42:34 |

- 三个关联生产/既有测试文件与本轮给定基线逐项一致（哈希、字节），本轮被评对象只有 `recovery_production_chain_integration_test.go` 变化。
- 变更前被评文件：`11766D09E75CD524D70341D8E48B270DEC63CB8DA9BAEB3A6D09338EC43E1D8A` / 136978 B / 3347 行；变更后 162632 B。行数按 `Get-Content` 末行索引为 3798 行（与卡面一致）。
- 实施证据已读：`s420-implementation-r01/baseline.md`、`progress.md`、6 个 `d7-*.json`。证据中记录的阶段结论与实测文件哈希一致；D-5 之后“本轮唯一改动文件为被评测试文件”的说法与实测一致。

## 1. 结论

**CHANGES REQUIRED**

阻断项：`S420-CR-F001`（HIGH）。`CanonicalReplaySameResult` 未能证明卡面要求的“同一 live permit / 同一 PermitBinding 的第二次 replay”；该叶在 permit 语义上是放宽断言，测试通过不能推出 oracle 成立。

5 个叶的实际执行结果均为 PASS（普通与 `-race`，见 §6），但 `CanonicalReplaySameResult` 的断言语义不足以支撑卡面验收语句，故整体不能给 PASS。

## 2. 两轮实际检查范围

第一轮（逐叶断言语义）：

- 逐行读取 5 个叶与 `s420RunProductionResume`（`:2938-3117`）、`s420WireReplay`（`:2551-2572`）、Case18 world/transport（`:1828-2010`）的断言与断言输入。
- 追踪每条断言背后的生产实现，而不是只看 t.Logf 文本：`internal/article/deletion_replay.go`、`internal/article/replay_evidence.go`、`internal/article/replay_publication.go`、`internal/article/deletion_replay_source.go`、`internal/recovery/runtime.go`、`internal/recovery/coordinator.go`。
- 判定恒真/同义反复/放宽/只对 fake 成立但声称真实链的断言。

第二轮（契约、边界、证据一致性）：

- 复核 `task.md` 卡面 `:2872-2945`（串行/停止条件、写集、验收命令、canonical oracle、5 叶最少层级）。
- 核唯一 Store/runtime/pool/authority/fallback 的指针关系与生产 wiring。
- 核 closed/unknown 是否真实经过 transport/guard、是否零 service/repository/author 调用、unknown 是否走真实 `PauseArticleAppend`→`FaultArticleAppend`。
- 核 `t.Skip`/`skipRecoveryChainLeaf`/未执行叶（`rg` 结果 0 命中）。
- 核写集越界（哈希 + mtime + `git status` 抽查）。
- 复跑 §6 命令并记录退出码与日志。
- 保密模式 grep 被评文件与证据目录。

有界上游/关联检查（只读）：`cmd/han-server/main.go`、`cmd/han-server/recovery_runtime.go`、`cmd/han-server/recovery_runtime_test.go`、`cmd/han-server/recovery_main_dispatcher_integration_test.go`、`internal/**` 相关实现、`deploy/**` 与 `test/integration/recovery/**` 的 mtime 抽查。

未做：未修改任何被评产物；未执行卡面以外的部署/restore/全量 E2E；未对真实 keyring/HSM、真实生产 dispatcher 全路由做运行验证（见 §7）。

## 3. Findings

### S420-CR-F001 — HIGH — 阻断

- 位置：`cmd/han-server/recovery_production_chain_integration_test.go:3372`、`:3393`（关键断言在 `:3425`）
- 违反依据：卡面 `tasks.md:2917`“第二次 replay 触发方式：同 run、同 live permit、同 PermitBinding（由 ReadPermitBinding 读取）；不得新建 run、permit、binding 或第二 runtime”；`:2927` 叶验收要求同 live permit/同 binding 的第二次 replay。
- 现状：`:3333` 取得 `lease := runtime.Acquire(ctx, WorkRequest)`，`:3338`/`:3400` 用 `lease.Context()` 读取 binding；但真正执行 replay 的 `:3372` 与 `:3393` 传入的是普通 `ctx`。`bindingBefore == bindingAfter` 只证明 lease 的 binding 在两次读取间没有漂移，不证明 replay 在 live permit/binding 下执行。
- 生产路径核实：`internal/article/deletion_replay.go:40-52` 的 `ReplayAndVerify` 只做 nil/空校验与 `tryReuseReady`；`tryReuseReady:275-291` → `PostgresReplayEvidenceStore.ReadCurrent`（`internal/article/replay_evidence.go:433-476`）；`ReadCurrent` 不调用 `checkReplay`（`replay_publication.go:132-147` 仅用于 Begin/Publish/Fail 路径），`validateRequest`（`replay_evidence.go:589-602`）也不校验 permit。`recovery.ReadPermitBinding`（`internal/recovery/runtime.go:1241-1251`）只是从 ctx 读取 runtime 上下文键。
- 触发场景：若 replay 读路径的 permit/binding 校验被移除、被绕过，或 replay 在另一 permit 下执行，本叶仍会通过（reuse-ready 分支不触碰 permit）。测试无法区分“live permit 下的 replay”和“不受 binding 约束的 replay”。
- 影响：卡面核心 oracle 之一未被证明；把未验证行为当作已验证，可能让后续消费者（article-management-S8.2）误以为同 permit replay 已闭环。
- 修复建议（由作者/Lead 决定方向）：
  1. 至少把 `:3372`、`:3393` 改为 `lease.Context()`；但这仍只证明“租约未漂移”，因为 `ReplayAndVerify` 本身不读取 binding。
  2. 增加负向对照：不带 permit/binding 的上下文调用 replay 必须被明确拒绝（`ErrAdmissionBinding` 或等价错误），再在 live permit 下成功；若当前生产实现不拒绝，则属于生产契约缺口，需要按“方向决策”回 Lead，不在本叶内私自改 `internal/article`。
  3. 或改为经生产 coordinator/consumer 的 permit 路径触发第二次 replay，而不是直接调用 provider。
- 证据：静态追踪上述文件行；普通与 `-race` 运行日志显示 `binding_equal=true`，但该值来自两次 `ReadPermitBinding(lease.Context())`，与 replay 调用上下文无关。
- 是否阻断：是。

### S420-CR-F002 — MEDIUM — 非阻断（需 Lead 决策）

- 位置：`cmd/han-server/recovery_production_chain_integration_test.go:3048-3056`
- 现状：helper 手工组合 `runtime.Resume(flowCtx, ...)` + `composition.requestComplete()` + `composition.completeIdentity(ctx)`，未走生产 `assembly.start`（`recovery_runtime.go:819-843`）与 `completeOnRequest`（`:848-861`）。
- 影响：resume 的生产编排（含 `completeOnRequest` 的完成时序、`requestStop` 触发）未被本 helper 覆盖；虽然 resume 本身是真实 runtime 调用，但“生产 root 的完整 resume 编排”只被部分验证。实施 `progress.md` 已声明该选择（为避开 `completeOnRequest` 提前 `requestStop`），属于已知偏差而非隐瞒。
- 建议：在报告中显式标注“编排层为等价手工组合，不是 `assembly.start` 原路径”，或补一个走 `assembly.start` 的对照叶（若卡面允许）。
- 是否阻断：否。

### S420-CR-F003 — LOW — 非阻断

- 位置：`cmd/han-server/recovery_production_chain_integration_test.go:3076`；`cmd/han-server/recovery_runtime.go:774-797`
- 现状：`t.Cleanup(assembly.requestStop)` 期望停止 control surface；但 `bindStop` 在全仓无调用者（`rg` 仅定义处命中），`a.stop == nil`，`requestStop` 为 no-op。`recovery_runtime_test.go:572-586` 的 `Serve` 阻塞在 `<-s.stop`，只有 `Shutdown/Close` 会关闭；`listener.Close` 不会解除该阻塞。`:3044-3046` 注释“release it in cleanup order after the runtime stop”与事实不符。
- 影响：测试 control server goroutine 与 joined wrapper 的 done 通道可能存活到测试进程结束（测试级泄漏），不会造成数据竞争或掩盖生产缺陷（`-race` 未报 DATA RACE）。注释会误导后续维护者。
- 建议：修正注释；若确需停止，走 `control.Shutdown`/`joined` 的关闭路径或显式 `bindStop`。
- 是否阻断：否。

### S420-CR-F004 — LOW — 非阻断

- 位置：`cmd/han-server/recovery_production_chain_integration_test.go:3540-3542`、`:3633-3635`
- 现状：叶名 `SingleRuntimeStorePool` 声称“单一 runtime”，但断言本身要求 `world.runtime != assembly.runtime`，即 harness 实际持有两个 runtime 实例（world 的 bootstrap runtime + production root runtime）。
- 影响：断言是“production root 内唯一且 bootstrap runtime 未被复用”，不是“进程内只存在一个 runtime 对象”。语义正确但名称与日志（`bootstrap_runtime=` 打印不同指针）容易被读成“只有一个 runtime”。
- 建议：在叶内注释/日志明确“production root 唯一；测试专用 bootstrap runtime 在 resume 前已退出且不参与 resume”，或改名以匹配实际断言。
- 是否阻断：否。

### S420-CR-F005 — LOW — 非阻断

- 位置：`cmd/han-server/recovery_production_chain_integration_test.go:3604-3635`
- 现状：`NoSecondRuntimeRouterPoolAuthorityFallback` 主要是指针相等/不等断言（composition runtime、assembly runtime、chain runtime、run store tap、evidence store、article handle），没有直接构造或断言“不存在第二 public router / fallback authority”。`:3612-3614` 的说明是注释而非可执行证据。
- 影响：对“无第二 public surface/fallback”的证明依赖“测试未构建”，属于结构性弱证明；不能从断言失败中检测“有人新增了第二 router 但未被引用”。
- 建议：补一条可执行证据（例如断言 `main` 组合只暴露一个 handler/dispatcher，或对 production root 的 public surface 计数），或显式声明本叶只覆盖指针层面的唯一性。
- 是否阻断：否。

### S420-CR-F006 — LOW — 非阻断

- 位置：`cmd/han-server/recovery_production_chain_integration_test.go:3695`、`:3705`、`:3744`
- 现状：`ClosedUnknownKeepIsolation` 使用 `s420NewCase18World(t)`（= `s420NewFullWorld` + world runtime）与测试专用 `http.ServeMux` 适配器（`:1982-1990`），不是 production dispatcher/router；成功分支由 `world.full.s420RunNormal(t)` 驱动，不是 production resume root。guard/origin/session/authorizer/service/repository/author 组件为真实组件（`:1866-1943`）。
- 影响：closed/unknown 的 503 + Trace + 零下游调用在真实组件上成立，但不是“production dispatcher + production resume root”上的端到端证据；卡面允许 R/P 层级，但读者不应把它当作 production router 验证。
- 建议：在叶内标注 transport 为测试适配器、成功分支为 `RunNormal` 而非 resume；或另补 production dispatcher 版本（卡面若允许）。
- 是否阻断：否。

### S420-CR-F007 — LOW — 非阻断

- 位置：`cmd/han-server/recovery_production_chain_integration_test.go:3554-3598`
- 现状：`SimulatedJournalKeyringBoundary` 明确打印 `adapter_scoped=simulated_journal_hmac_keyring`，并用测试私有 key 复算 begin/consumed HMAC、fingerprint，再与真实 PG `recovery_runs` 的 key id/version/fingerprint 对照；真实 journal snapshot entries=2/prefixes=3。
- 影响：边界标注真实，未声称 keyring/HSM 已验证；但“keyring 边界可复现”只到软件模拟层。符合 ADR-0028 的模拟环境标注要求。
- 建议：保持现状即可；如后续有真实 keyring 环境，另立验证项。
- 是否阻断：否。

### S420-CR-F008 — MEDIUM — 非阻断（写集/交付完整性，需 Lead 归属与处置）

- 位置与实测：
  - `internal/article/deletion_journal_hash_test.go`（1349 B，2026-10-31 19:22:52）
  - `test/integration/recovery/article_replay_test.go`（86718 B，2026-10-31 19:22:52）
  - `test/integration/recovery/old_backup_restore_test.go`（15482 B，2026-10-31 19:22:52）
  - `cmd/han-server/recovery_isolation_matrix_integration_test.go`（8498 B，2026-10-31 19:39:57）
- 现状：这些文件不在卡面写集（卡面只允许 `main.go`、`recovery_runtime.go`、`recovery_runtime_integration_test.go` 仅测试承载迁移、新文件 `recovery_production_chain_integration_test.go`）；其中 `internal/article/*` 被卡面明确列为“不默认修改”。mtime 落在 S4.20 实施窗口（18:59–20:27）内，但 `progress.md` 未记录这些写入，`s420-*` 证据也未声明。
- 影响：无法从现有证据证明归属（可能是本会话额外工作，也可能是并行会话），但交付层面存在“未声明的写集外文件”。内容抽查：`internal/article/deletion_journal_hash_test.go` 为纯内存 hash pin 测试；`recovery_isolation_matrix_integration_test.go` 复用 S4.19 dispatcher 测试的 `newS419ProductionDispatcher`，属 S4.19 覆盖扩展；未见危险操作或生产行为改动。全量 `git status` 中 `internal/`、`deploy/`、`test/integration/recovery/` 整体为未跟踪目录，无法用 git 精确归属。
- 建议：由 Lead 核对这些文件的真实写入者与所属卡；若属 S4.20，需要按卡面“第五文件/写集外即停止”的要求补记录或回退处置（处置动作不由评审执行）。
- 是否阻断：否（但未归属前不得当作 S4.20 交付物）。

## 4. 逐叶判定

| 叶 | 判定 | 要点 |
|---|---|---|
| `CanonicalReplaySameResult` | **CHANGES REQUIRED**（F001） | 持久字段逐项相等、零额外写入（行数/ transitions / marks）证据充分；但 permit/binding 证明放宽，replay 用普通 ctx，oracle 未闭环。 |
| `SingleRuntimeStorePool` | PASS WITH MINOR（F004） | `chain.store==chain.ports==world.closures`、`chain.rt==assembly.runtime`、`runStoreTap.inner==inputs.runs`、`composition.runtime()==assembly.runtime` 均成立；但 harness 存在两个 runtime 对象，命名/日志需限定为“production root 唯一”。 |
| `SimulatedJournalKeyringBoundary` | PASS WITH MINOR（F007） | 边界明确标注为模拟；真实 journal + PG key 绑定校验成立；无独立 keyring/HSM。 |
| `NoSecondRuntimeRouterPoolAuthorityFallback` | PASS WITH MINOR（F005） | 指针层面唯一性成立；没有直接证明“无第二 public router/fallback authority”。 |
| `ClosedUnknownKeepIsolation` | PASS WITH MINOR（F006） | closed/unknown 真实经过 guard/session/authorizer/service/repository/author（零调用），unknown 走真实 `PauseArticleAppend`→`FaultArticleAppend`→`ErrRecoveryWriteUnknown`，durable writes=0；但 transport 为测试 mux、成功分支由 `RunNormal` 驱动，不在 production resume root 上。 |

补充：`CanonicalReplaySameResult` 的持久结果比较不是同义反复——`runBefore/runAfter` 来自真实 `world.store.Load`，journal snapshot 来自 `ReadVerifiedSnapshot`，admission snapshot 经 `CanonicalAdmissionSnapshot`，行数来自真实 PG 表；`transitionsBefore/After` 经只读 tap 计数，`marks` 经只读计数器。这些断言有效。问题只在 permit 语义。

## 5. 共享 helper `s420RunProductionResume` 结论

- `t.Cleanup(flowCancel)`（`:3047`）修复本身正当：`runtime.Resume` 从 `flowCtx` 派生 generation context，helper 返回后若立即 cancel 会释放 live permit，后续叶无法在“同一 permit”下读取状态。Cleanup LIFO 顺序为 `requestStop`（`:3076`）→ `flowCancel`（`:3047`）→ `cancel`（`:2941`）→ `listener.Close`（`:3034`），顺序合理。
- 但 `requestStop` 实际是 no-op（F003），因此“release after the runtime stop”的注释不成立；control server 的 goroutine 可能存活到进程结束。未发现数据竞争（`-race` 通过），未发现该问题掩盖 canonical 缺陷。
- helper 的根级指针断言（`:3019-3027`）有效；`world.runtime` 只在 `:2968` bootstrap `Enter` 使用，`s420WaitLeaseReleased`（`:2971`）确认其租约释放，代码与证据支持“resume 前 bootstrap 专用”的说法。
- helper 未走 `assembly.start`/`completeOnRequest`（F002），属已声明的等价手工编排。

## 6. 实跑命令与退出码

全部在固定工作区、HEAD `6da8abb` 上执行；日志见同目录 `commands.log`、`rerun-plain.log`、`rerun-race.log`、`rerun-full8.log`、`rerun-dispatcher.log`、`rerun-recovery-postgres.log`。

| 命令 | 退出码 | 结果 |
|---|---|---|
| `gofmt -l cmd/han-server/recovery_production_chain_integration_test.go` | 0 | 空输出 |
| `gofmt -l` 四文件（main/recovery_runtime/recovery_runtime_integration_test/被评文件） | 0 | 空输出 |
| `go build ./...` | 0 | 通过 |
| `go vet ./cmd/han-server` | 0 | 通过 |
| `go vet -tags=integration ./cmd/han-server` | 0 | 通过 |
| `go test -tags=integration ./cmd/han-server -run '^TestRecoveryProductionChainComplete$/(CanonicalReplaySameResult|SingleRuntimeStorePool|SimulatedJournalKeyringBoundary|NoSecondRuntimeRouterPoolAuthorityFallback|ClosedUnknownKeepIsolation)$' -count=1 -v -timeout 400s` | 0 | 5/5 PASS，包 ok 4.945s |
| 同上 + `-race`，`-timeout 500s` | 0 | 5/5 PASS，无 DATA RACE，包 ok 6.53s |
| `go test -tags=integration ./cmd/han-server -run '^TestRecoveryProductionChainComplete$' -count=1 -v -timeout 400s` | 0 | 8/8 PASS，包 ok 7.230s |
| `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -timeout 300s` | 0 | 通过，ok 0.418s |
| `go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -timeout 300s` | 0 | 通过，ok 6.506s |

`rg -n 't\.Skip|skipRecoveryChainLeaf|SkipNow|Skipf'` 在被评文件：0 命中（exit 1，无输出）；`t.Run` 8 个（`:2748-2755`），5 个目标叶均实际执行。

关键日志摘录（不含凭据）：

- `canonical_replay_same_result ... binding_equal=true run_equal=true snapshot_equal=true admission_equal=true revision_equal=true rows=1/1/1/1 transitions=4 marks=1 head_seq=2`
- `single_runtime_store_pool runtime=<A> ... bootstrap_runtime=<B>`（A != B）
- `public_stream_open handler=2 trace=2 service=2 repository=2 author=2`
- `unknown_fault cause=recovery write result unknown observation="closed" verify_err=recovery admission closed`
- `closed_unknown_isolation service_zero=true repository_zero=true author_zero=true durable_writes=0`

## 7. 未验证风险与限制

- 本报告不声称 production dispatcher/router 或 production resume root 全链 PASS：closed/unknown 叶使用测试 mux 与 `RunNormal` 成功分支（F006）；resume 编排为等价手工组合（F002）。
- 未验证真实 keyring/HSM（F007 明确模拟）；未验证真实部署 E2E、POSIX/ACL、真实备份 provider。
- `CanonicalReplaySameResult` 的 permit/binding 语义缺口无法通过“测试通过”自证，属静态语义缺陷（F001）。
- 写集外文件归属未知（F008），在本轮证据内无法判定写入者。
- 本轮未对 `test/integration/recovery` 的两个 19:22:52 文件做内容级逐行审查（仅抽查 + mtime/hash 定位），其质量结论不在本报告范围。

## 8. 保密

- 被评文件与实施证据未发现 DSN、口令、API Key 或真实 endpoint 明文；命中的仅为字段名与测试生成的合成值（会话/csrf token 由测试 UUID 生成，origin 为 `.invalid` 合成域）。
- 本报告未复述任何凭据或真实 endpoint。

## 9. 报告元数据

- 评审者：add-reviewer（非作者）；`actualID`：不可用（本端未返回真实 Agent ID，未编造）。
- 关联日志：`commands.log`、`rerun-plain.log`、`rerun-race.log`、`rerun-full8.log`、`rerun-dispatcher.log`、`rerun-recovery-postgres.log`。
- 本报告只读评审，不代替 Lead 的复核、人的业务签收或后续授权。
