# B13G / identity-auth-S4.15 独立 Code Review

**结论：两项 HIGH，均在最终冻结源码普通/race 中独立复现；CodeGate = CHANGES_REQUIRED；单任务 B13G whole G4 = CHANGES_REQUIRED。** 未修源码、未关闭 Task、未接受风险、未代签人的接纳。

评审者：Ptolemy/add-reviewer，实际 ID `01a18bbe-e72e-7d62-a3fc-8748bf262fb0`，依据 Main 实际 START grant；没有采用继承的 CODEX_SESSION_ID。原作者为 Sartre/add-backend-dev `01a18b91-4da7-72c3-92c7-0e2b5dbe0823`。本执行者为非作者；未子委派。

## 固定对象与授权

- objectType：code；原 development-delivery 的 review 环节。工作区 `D:/AID/VSCode/Codex/Han`，main，固定 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，分支/HEAD 来自用户及 grant，Git 命令为 0。
- 最终对象：`D:/Temp/han-b13g-lead-20261030-r01/s415-final-input-r01`。manifest SHA256 `587A4F63C10490A7BC2905563FE5984EC9F704FCC39EFEABCF205B7B2D1523A9`；full10.diff SHA256 `1D431EC7AD6F8AB98BDAB4D3E438974509BFF50000025DE64BF65BE1A5B7ED1D`；freeze46 SHA256 `737A216A0CAF5D166047BE7A6A7BF8E675CDE5F3C4210018859884784FCEA3E8`。
- 初审对象为同级 `s415-static-input-r01`，未把初审当最终。初审阶段没有读取/hash 活跃原10；收到最终 release 后才按新授权只读核现场46。
- 源文件所有权仍仅原10；前2是新增，其余8为既有 untracked。按 before 字节重建 full10.diff，最终9个变化块、lifecycle_test.go 未变；10份快照 SHA、release source10、现场冻结46一致。没有用 Gitdiff 推断 untracked 的作者贡献。
- START grant：`.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-final46-local-start-r01.json`，SHA256 `637F03E7A12CD024AEF9213BA441B061FFEF9C457DB9295D071646245176D84B`，已亲自全文读取。前期静态阶段 Go/cache/SQL=0；获得此 grant 后仅执行两个串行定向 Go job。
- Repo/source/meta 写、Git 命令、SQL、PG、DSN/秘密读取、服务、网络安装及子委派均为0。仅写本人唯一D，另使用本次明确排他租用的受控编译缓存；未修改生产源码或原断言。

## 两轮实际检查

**第一轮：直接实现、全部实际 test delta。** 亲自完整取得 add-reviewer.toml、add-code-review、已选 add-delivery-workflow，以及 task-execution、multi-agent、review checklist、abstraction、coding、no-hardcode、observability、安全红线、go-service README 及相关 coding/security/observability/no-hardcode 规则。未加载未启用 profile、无关前序或未来 Workflow，没有重新 Route/G0。

读取原 S4.15 整卡 Tasks:2584–2627、B13G Plan:1566–1574；核 identity r56 的固定 SHA/§4.2 和 article r27 的固定 SHA/§3.4.6，最低 U/C/race 及相关 TP。七前置的完成由本轮明确 Main 现役核证输入承接，未据旧 mapping/pending 重开 Planning Gate。design draft/人的接纳仍 unknown。

审完五个生产文件全文、full10.diff 全部变化及五个测试文件的全部实际增量；另读原 lifecycle/notice/worker 测试相关消费者与断言。对既有测试逐字前缀检查通过，没有改断言迎合实现。范围仅许可接入、started 观测、关闭收口；原业务 lease/retry/SQL/通知分类算法未见被本 diff 改动。

**最终4路径 delta 复核：** 见 `preliminary-to-final4.diff` 与 `static-checks-final.json`。

| 路径 | 初审到最终的实际变化及复核 |
|---|---|
| cmd/han-server/recovery_workers.go | 除格式化，新增冷启动 join 的 coldOnce/coldErr、限定三种 worker kind、checkWorkers 按 reset→article→notice 优先序、StopAndJoin 复用 lifecycle 错误优先级。逐项审读；两项 finding 所在取消/授予衔接仍存在。 |
| cmd/han-server/recovery_workers_test.go | 展开格式，保留原断言与风险组。 |
| internal/article/deletion_worker.go | 展开 stopping 判断格式，行为一致。 |
| internal/identity/password_reset_worker_test.go | 将原 false/true 表体提取 helper，保留全部原断言、调用和计数。 |

**第二轮：沿真实消费者至少两层。** 不是全库扫读，具体读取范围/哈希见 `readset.json`。

| 调用链 | 已核内容与边界 |
|---|---|
| new recoveryOrdinaryLifecycle → Runtime.RunNormal/activate/PhaseContext/Acquire/Stop/drain | 内部 startup capability、phase 生命周期/回调、旧代际撤销、异步取消、许可等待、实际 worker Started/Done/Err、StopAndJoin 外部调用均不在 Runtime mutex 内。Runtime:781–887、1015–1207、1209–1400 为关键读集。 |
| DeletionWorker discover/converge → Repository discovery / DeletionFinalizer → Journal Append、claim、DB completion | scan/每 operation 独立 Acquire，覆盖 Rows/Journal/claim/最终提交返回；没有嵌套 acquire 等自己。真实 Finalizer:133–172、249–273、388–451、464–508；Repository:18–58。健康 append 不取消已有 lease，故障撤代际后仍要 join。 |
| PasswordResetWorker scan/reconcileCandidate/settle → backend Claim/Renew/Fail → Operator.executeHeld / Store settlement/readback | Runtime 工作许可与连接 permit 分开；job 许可覆盖 Claim、Renew、settle、失败保存和 commit-unknown 回读。Operator:139–155、175–265、315–335；原 recovery_password_reset 显式 SettleRestored/claim/renew 路径仍独立，没有用 ordinary Start 替代受控恢复 driver。 |
| TransferNoticeDispatcher maintainWork/dispatchWork → claim/preflight/Send/resultContext/saveResult/readResult → Repository fenced SQL | 工作许可保持至结果保存/回读返回；核完整 fencing、claim owner/attempt/locked_until、旧发送与新 claim、unknown 分类、共享截止、停止与实际 Done。Repository:615–719；没有改 Mailer/SQL 算法。 |
| lifecycle → 原 main 的 coordinate/config 与实际三个 worker 构造 | main:604–619 仍是既有 root 传入；lifecycle 的初次调谐、等待、错误优先级、server shutdown、joinWork、Article资源→pool已核。唯一 production root 新三实例/第三surface接入仍归 S4.16/17，本报告不把它变成本卡前置。 |

数据/迁移方面，固定 diff 只有原10的9个变化块，没有 migration/DDL/Store/SQL/持久 wire 变更；未执行数据库迁移、读写或模拟生产回退。资源/性能方面核固定三 owner、每工作许可/取消回调的释放及无嵌套自等待；没有做容量/压力测试或声称吞吐达标。

## [HIGH] S415-CR-F001：worker 取消回调抢先绑定了另一个关闭截止

- **对象/位置**：最终 `cmd/han-server/recovery_workers.go:195–200`、`:227–234`、`:252`。文件 SHA256 `432AD80CE31006CE6B4D0B36DFE15001D93B67D6B81DEB22DFEF4EF97036A4C4`。
- **依据**：原 S4.15 SharedDeadlineNoticeFencedSave/关闭条件；identity §4.2:3631–3635，TP:839。必须共用首次关闭的绝对截止，不能重置通知清算预算。
- **真实消费者两层**：Runtime.Stop:1371 先保存调用方 shutdownCtx；:1377–1385 先撤销 phase/代际/工作，再由 drain:1334 调本 adapter.StopAndJoin。新 forwardCancellation 的 AfterFunc 因而可能先于 StopAndJoin 执行；它用 worker caller ctx 调 bindShutdown。notice.BeginShutdown:263–268 又只接收第一次绑定的 context，resultContext:743–749 使用它进行清算。
- **触发交错**：Runtime 已持截止 D；有仍持有的 notice work，worker caller 没有 D；撤代际触发 callback，callback 先按 `time.Now()+l.budget` 锁定 shutdownOnce；随后 StopAndJoin(D) 无法再提供真正的首个 deadline。原始D已经存在，不是后续重复 Stop 合理地保持原截止。
- **实际影响**：通知 fenced 保存/根 owner join 可取得超过原共享截止的预算。超过截止继续等真实 Done 是正确行为，但该合同不允许继续授予新的清算时间；此处两种行为被混淆。
- **独立复现**：`TestS415Independent234247_SharedDeadlineOrigin` 保留真实 Runtime、adapter 和三个实际 worker，仅在已注入的 Lifecycle.StopAndJoin 入口加 barrier，让既已发出的撤权 callback 先完成；没有改 Runtime 的顺序。Stop调用方预算3秒、adapter显式预算1小时，普通/race 都观测通知 deadline 延长 **59m57s**。测试在释放 barrier/lease、Runtime.Stop 和三个 worker 全部实际 join 后才失败，没有等一小时、没有后台遗留。
- **证据**：普通 raw `raw/independent-final-normal/stdout.jsonl`；race raw `raw/independent-final-race/stdout.jsonl:72`，标识 `FIRST_SHARED_DEADLINE_LOST`；探针 `probes/s415_independent_test.go:84`，断言:101。两次 job exit1，失败来自真实断言。
- **最小修复边界/归属**：回 Sartre，优先在原 `recovery_workers.go` 及原 `_test.go` 内修复取消转发与共享关闭 context 的交接。让唯一关闭 owner 的既有截止在取消向 notice 传播前被绑定；不能再由普通 worker 的任意 caller ctx 抢先发明截止。可用显式交接/同步保障，仍保持 BeginShutdown 先于通知工作取消、callback/release 真 join、Stop owner 至多一次。不修改 main、Runtime、Store、SQL/DDL、Task/Plan/design/TP；若实现证明必须扩大，先向 Main 给出最小额外路径与理由，不由 Reviewer 扩权。
- **复审条件**：保留原探针与断言，在修订后的同 source10 普通/race 中变绿；覆盖 callback 先行与 StopAndJoin 先行、重复 Stop 不重置、超期仍等 Done、已开始保存不重启，原健康/故障关闭回归保持。

## [HIGH] S415-CR-F002：健康 append 与授予竞争时，ordinary Acquire 被误报为 terminal

- **对象/位置**：最终 `cmd/han-server/recovery_workers.go:174–176`，以及其接入的 `internal/identity/password_reset_worker.go:254–259`。后者 SHA256 `6F1908205EC356E1943C028BB57034F061A5BE551ABD71085867E1952D3A8A3C`。
- **依据**：原 S4.15 AllScanClaimDispatchLeases/AppendBarrierDoesNotWaitForSelf，identity §4.2:3621、3631、3635。健康临时屏障应等待，不得成为业务 terminal。
- **真实消费者两层**：现役 Runtime.Acquire:1027–1032 在锁内观察 ready 后释放锁，直接返回 acquireReady 的结果；acquireReady:1062 先调用实际 Exclusive.Check，:1065–1068 再取锁核 causes。Article writer 的 PauseArticleAppend:1223–1225 可恰在两者间加入健康 cause。此时返回 ErrAdmissionClosed，没有重新进入等待。新 adapter 直接上抛；PasswordResetWorker.scan→Reconcile 返回错误，run:254–259 在并未 stopping 时加入 ErrPasswordResetWorkerTerminal 并退出；lifecycle:199–201/441–442 消费其 Done 会触发整体关闭。
- **触发**：正常开放、一个真实持有 request lease 的 writer 准备健康追加；ordinary PasswordReset Acquire 已观察 ready，但最终 grant 尚未执行；writer Pause 成功，然后授予复核发现 cause。
- **实际影响**：正常合法 append 的短暂拒新工作屏障可以停止 password-reset recovery worker，并沿既有生命周期触发全局关闭。无需任何 DB/Journal 损坏、丢锁或真正 fault。该问题在新 adapter 与既有 Runtime/worker 的组合边界成立；没有把既有未修改 Runtime 冒称本作者新写。
- **独立复现**：`TestS415Independent234247_HealthyPauseDuringAcquire` 只在已注入 Exclusive.Check 对带私有测试标记的一次普通 Acquire 设置 barrier；真实 Runtime 的首次 ready 观察已完成，测试用另一个真实 request lease 调 Pause。释放 Check 后，在 caller 500ms预算到期前立即收到 `recovery admission closed`；普通/race 都满足 errors.Is(ErrAdmissionClosed)=true。健康 Complete 之后按真实 Runtime 合同完成，acquire goroutine 与所有 worker 均真 join。期望是保持屏障等待至 caller deadline，而不是提前 terminal。没有通过修改生产 Runtime/断言制造结果。
- **证据**：同两份 raw，race:77 标识 `HEALTHY_PAUSE_BECAME_TERMINAL`；探针 `probes/s415_independent_test.go:108`，断言:130。真实 worker 的 terminal 后果由上述源码链证明，探针直接复现的是其输入边界 ErrAdmissionClosed；不冒称此探针额外执行了真实业务 DB 或 production 全局停服。
- **最小修复边界/归属**：回 Sartre，首先核原10内的普通许可适配/错误分类。保证这种健康竞争重新进入现有等待路径，真实 fault/失权/caller取消仍能停止；不得用无限重试全部 ErrAdmissionClosed、吞掉故障或绕过 Runtime 自行授许可。如果现役 Runtime 接口确实无法可靠区分，向 Main 精确提出 `internal/recovery/runtime.go` 最终授予竞争的最小原子性修复及回归边界；它在原10之外，本报告仅给 finding 依据，不授权直接改它，也不重开全部已闭前置。
- **复审条件**：原探针普通/race 变绿，新增健康屏障完成后可继续的正控、真实 fault 撤代际负控，原三 worker 暂关等待/已有 work 继续/结果保存许可不变。

## 独立执行与真实退出

实际只运行两个串行 Go job，均 `GOMAXPROCS=1`、`-p=1`、`-parallel=1`、`-count=1`、offline/local/modreadonly。sharedwarmcache 本轮明确排他租用；profile/TEMP/GOPATH/work/raw 在本人D，未继承 DSN/业务秘密。调用前写 command/source header/heartbeat；CreateProcess suspended→AssignJob→Resume，双管道并行 pump，父进程实际 Wait、kernel active=0、双 EOF、pump线程 join 后才算结束。

| 模式 | 原5风险组 | 模块新leaf | 必要旧回归leaf | 新独立探针 | job |
|---|---:|---:|---:|---:|---|
| 普通 | 7 PASS | 4 PASS | 8 PASS | 2 FAIL | exit1；Wait/Job0/双EOF完成 |
| race | 7 PASS | 4 PASS | 8 PASS | 2 FAIL | exit1；Wait/Job0/双EOF完成 |

每次总计19 PASS、2 FAIL、0 SKIP，普通/race 同名；不能把整个 job 记为 PASS。race 无 DATA RACE 报告，这不消除上述逻辑竞争。详见 `independent-leaves.json`，精确 argv 及运行环境在各 job 的 `command.json`，结果与46前后SHA在 `result.json`。

所选旧回归是：Article GlobalDeadlineCancelsAndJoins、PasswordReset CommitUnknownDoesNotFailOrStop、lifecycle PasswordResetDeadlineWaitsForDoneBeforeClosingResources、CompetingFailures_ErrorPriorityAndCloseOnce 的5叶。没有重复全量3167。

**实际释放**：`actual-release.json` 的 UTC `2026-10-30T08:05:53.749403+00:00`，SOURCE=0、LOCAL/cachelease RELEASED、pending=0、两个 job 已全 join，前后 frozen46 精确一致。外层工具 session 也实际 exit0；此 exit0 只表示 runner 正确保留预期失败并收口，不是 CodeGate PASS。释放后只进行 D-only、cache-free 材料生成，无后续 Go/缓存访问。

## 作者/Main 证据与未验证范围

- 亲读 Main 作者接收记录：26 primary jobs 已真 Wait/Job0/双 EOF、required10 same final。其排他窗口是 Main 现场接收事实，本 Reviewer 没有逐个重跑26 job。
- 另直接解析作者两个精确当前三包 stdout：普通/race 各3165 PASS、2 SKIP、0 FAIL，终端 leaf 名称逐一相同；见 `author-final-raw-summary.json`。跳过项均为 `han/internal/article/TestFileBackupInventoryReader` 的 `symlink_rejected`（当前客户端无符号链接特权）与 `world_writable_rejected`（本平台无 POSIX write-mode 语义）。二者仍未验证，不计3167全PASS，不继承旧waiver。
- 作者 topRace/vet 历史重叠1.451s，两旧结果已排除合规验收；kernel byte lock 后仅重跑两条且成功。保留 incident，不声称历史全程串行，不把调度 incident 当产品 RED。
- Main 8 commonchecks 及 own40fmt0为 Main 独立现场证据，非本 Reviewer 重跑；rootfmt原23 FAIL2保留，goimports/golangci缺失仍NOTPASS。没有继承旧30全局豁免。
- 暂未验证真实 SMTP/provider、native平台语义/ACL、生产 root/第三surface、部署/E2E、真实 HTTP/维护/Tx/child 的生产接线和人的接纳。U/C 的真实 Runtime/三worker循环与测试屏障，不冒充 production root、真实服务或 E2E。原 S4.16/17 的后继责任不新增为本卡前置。
- additionalServers 本次没有真实 root 消费者；其 shutdown 结果汇总仍只按 business/metrics 分支处理，后继接线必须有真实消费者证据；此处保留范围限制，没有用尚未接入的第三surface造当前产品阻断项。

## Gate 因果与交回

**CodeGate：CHANGES_REQUIRED。** 当前最终源码的两个直接可操作契约缺陷，已有非作者普通/race失败证据。不是因为 WIP未跑宽回归、工具缺失或未来 production/E2E 尚未完成而给产品 finding。修复回原作者，原 Reviewer 按稳定 finding ID 复审。

**单任务 B13G whole G4：CHANGES_REQUIRED。** B13G 虽只有 S4.15，仍需组合关系成立：F001 破坏 Runtime撤权→notice清算→全部owner join 的共同截止；F002 破坏 Article健康屏障→Runtime许可→PasswordReset循环→lifecycle整体停止的跨组件行为。这是本批内部真实接口/资源的独立因果判断，不把“单卡 CodeGate不通过”机械复制成整批结论，不评判未来 B13H/整个需求的通过。

没有待 Reviewer 自动采用的风险决定；两个 HIGH 均交 Main 安排 Sartre 修复。原10之外若确需 Runtime修改，由 Main 精确限定额外路径与 owner。当前报告不要求新G0、设计审批或用户重复许可；也不自行改 source、Task/Plan 或把 standing 建议授权当人的接纳。

精确复现文件/命令与后续复审要求见 `reproduce.md`。本报告与原失败 raw/probe 应保留，修复后追加新 revision 的结果，不覆盖本次失败链。
