# B13G / identity-auth-S4.15 — READONLY G0

**结论：READY，仅表示 G0 已具备承接条件。** 原 10 文件内可实现并验证原五组 U/C/race 行为；未发现必须扩大文件清单、修改 Runtime/Store/SQL 或改变契约的证据。SOURCE、Go/test/gofmt、SQL/PG、服务均 CLOSED，全部验收 NOT RUN，无 CodeGate、Taskclose 或人的接纳结论。

正式作者：Host **Sartre / 01a18b91-4da7-72c3-92c7-0e2b5dbe0823**。继承环境 CODEX_SESSION_ID `01a17b82-d88a-7061-87c2-ca96fdd5b545` 不是本 Role 的实际 Agent ID。目录名 cedar 不代表 Host 昵称。本目录为本人本轮新建独有目录；只保存 G0 读集/报告及只读检查记录，没有 Temp 实现、源码候选或新前审 Gate。

固定 `D:/AID/VSCode/Codex/Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，未切换工作树/分支，未子委派、未作任何 Git 写操作。Main 最新告知：基线 15949/全树 guard PASS、Source 0；此为 Main 的核验结果，不冒称本人全树 guard。

交付前复核：原 10 路径及其余 37 个已读对象字节未变；仅 Tasks/Plan 随已告知的 Nash 纯 claim 更新，已本人重读原卡/B13G，现役 `[-]`、owner 为 Sartre，所有执行窗口仍 CLOSED。Tasks 新 SHA `051FD4DDB50572AFBC4F2AEF1CFB14420FED6E6EB4B53745C456622E83104446`；Plan 新 SHA `858D08074858A86CED6C1A87DCB0FE164D1DCCDF99742346492CF22D6C373D35`。本 receipt 保留旧读 SHA 与这次刷新，不将正常交接写入误报为未知源码漂移；没有自行验收 Nash 的全量 SSOT diff。Git staged 仍空；共享树新增两条 Main 交接证据路径，本作者项目写入为 0。

已本人完整读取 backend Role、task-implementation、delivery-workflow、route-work、git-workflow、coding/no-hardcode/review-checklist/task-execution、已启用 go-service README 与相关 coding/no-hardcode/security/observability rules、add-tdd-go。具体全文/片段及逐次实际 SHA 见 receipt；未以父会话读取代读。

业务依据已核：原 S4.15 整卡及同节通用合同、原 B13G；identity design r56 SHA `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF` 的 §4.2 内部准入/生命周期；article design r27 SHA `4D2321D4A2D9514EA1AC2A4812E11AA6A83CA8046202F596B2F99F1DE39C7CEE` 的 §3.4.6；identity TP r26 §4.7.1、article TP r21/r27 来源 §4.8.2。两 TP 当前 SHA 分别为 `34826759C79A58EB0B4A296D2DD5C9157014406859153DC6CA0FA6E2BD921DAB`、`24786C1A584ACF5277DD2DBD43E053CB7679F2867CD42937EDC7F722FD435036`。沿用已接收技术 mapping，不重开旧 pending；draft/人的接纳 unknown 保留。

七个 requires 已逐项从现役 Tasks 核得唯一 `[x]`：identity S4.7、S4.8、S4.12、S10.4，article S4.7、S7.1、S4.6。B13F 只消费指定 verification r06 FINAL 与 lead-resume-r06 的依赖事实：S4.12 技术关闭、当前测试 PG 已停止且数据保留；没有重跑或外推 B13F 结果。旧记录中的 ACTIVE 字段不覆盖 r06/current_PG stopped。

## 原 10 文件现况

八个既有文件在作者初始 Git status 中均为 `??`，是共享树既有内容，不能因 untracked 当作本作者新增。新两文件尚不存在。完整 SHA/字节数在 receipt.original10。

| 文件 | 现况 | SHA256 前 12 位 |
|---|---|---|
| cmd/han-server/recovery_workers.go | 不存在，未来新增 | — |
| cmd/han-server/recovery_workers_test.go | 不存在，未来新增 | — |
| cmd/han-server/lifecycle.go | 既有，699 行 | 8f1fe0efeef2 |
| cmd/han-server/lifecycle_test.go | 既有，1232 行 | 662a2693547a |
| internal/article/deletion_worker.go | 既有，494 行 | 9e6bdd23089b |
| internal/article/deletion_worker_test.go | 既有，744 行 | 444c304effe6 |
| internal/article/notice_dispatcher.go | 既有，1090 行 | c49cf9d7c78e |
| internal/article/notice_dispatcher_test.go | 既有，1537 行 | 7761947f2888 |
| internal/identity/password_reset_worker.go | 既有，399 行 | 9bf1a9b12300 |
| internal/identity/password_reset_worker_test.go | 既有，641 行 | dc6544c4390c |

行数按 Python splitlines（包含文件中原有非 LF 分隔符）；源码引用以函数名为准。作者未修改这 10 路径。

## 真实接缝与必要最小改动

- `recovery.Runtime` 已有 `Acquire/WorkLease`、`CheckPhase/PhaseContext` 和 `OrdinaryLifecycle.ReconcileAndStart/StopAndJoin`。Acquire 对未激活/临时 append 关闭的 worker 等待，对故障/排空拒绝；Pause 只禁新工作，Fault 撤旧代际。无需修改 Runtime 实现或新增 authority。
- `lifecycleRun.start` 现有流程先按 Article→PasswordReset→notice Reconcile，再同序 Start，但只凭 Start 返回置 started。新增 `recovery_workers.go` 承接同 Runtime、精确三实例注册及 startup PhaseContext；三个真实 run loop 增加一次性 started 观测，并核 Done/Err，内部 Reconcile 不等公共 Acquire。
- deletion 在 Discover 和每个 Converge 前取得对应许可；Converge 同步经过 Journal/绑定/Finalize 才结束。reset 的 Scan、Claim→Renew→settle/Fail 要分别守新工作边界，业务 lease 与 ConnectionPermit 不替代 Runtime WorkLease。notice 的维护扫描、claim→preflight→dispatch→fenced 保存/必要回读必须覆盖完整生命期；不得在 claim 返回或 SMTP 返回时提前 Release，也不得在已持工作中重复 Acquire 等自身屏障。
- 三 worker 原调度、lease/retry、SQL 和通知策略保持；只补依赖适配、真实 started 观测及取消/释放收口。四个既有测试文件保留原行为测试并补原五组对应断言，新 cmd 测试承载原 top。
- lifecycle 已有共享截止、notice BeginShutdown、Wait 后额外等待 Done、Article 资源→pool 和错误优先级。复用这些能力，补同一 Stop/Join owner、维护/事务/child 的实际 join 接缝及按实际注册容量收集结果；不以 Shutdown 返回或 timeout 当排空。
- **取消时序必须测实**：Runtime Stop/Fault 可能先撤代际、取消 work，再调用 StopAndJoin；notice 不能因此在共享截止绑定前丢失已有 accepted 结果的有界 fenced 保存机会。适配需在原 owner 的 shutdown 绑定/结果收口中处理该交错，不重置 deadline、不另发邮件。StopAndJoin 不同步反调 Runtime.Stop，避免等待包围自身的 normalDone/work。
- main.go 当前构造/注入路径只作只读消费者证据。唯一生产 root 注入、第三 server slot 与完整 R 仍交原 S4.16；本卡提供可消费组件及 U/C 接缝，不冒称生产 root 已接好。actor 的恢复 reset driver 不由 ordinary 循环替代。

## 原五组验收可行性（全部 NOT RUN）

| 原风险组 | 原最低层 | 本卡可验证的关键事实 |
|---|---|---|
| InternalStartupThreeActualHandshakes | U/C | 同 Runtime 真实 PhaseContext，公共关闭仍 Article→reset→notice 调谐；run 内 started，不借 Start 返回；提前 Done/Err 拒绝激活 |
| AllScanClaimDispatchLeases | U/race | 在实际 scan/claim/dispatch 入口放 barrier；临时关闭零新工作且不 terminal；lease 持有至保存/回读/事务真实结束 |
| AppendBarrierDoesNotWaitForSelf | U/race | 已持工作可跨健康 Pause 完成，新 claim 阻塞；Fault 撤代际并取消；不嵌套 Acquire 自等待 |
| SharedDeadlineNoticeFencedSave | U/race | 复用现有 noticeMemory/阻塞保存 fixture；覆盖上述撤代际交错、首次绝对截止不延长、单次保存、owner Stop 至多一次、deadline 后仍等 Done |
| RealJoinResourceOrderAndFailurePriority | U/C | 阻塞 HTTP/维护/Tx/child/三 Done 的真实调用返回；逐项释放前 Article/pool 关闭计数为零；metrics/earlyDone 拒绝开放；原主错误优先、cleanup 附加 |

`TestRecoveryOrdinaryWorkers` 当前文本搜索未发现定义；这是原卡待新增符号，未执行 `go test -list`。以后 Main 精确 START 后先取得可运行的行为失败证据，再实现；核普通/race JSON 的五组真实命中，不以零测试、Skip 或父 top 绿代替。原卡六条命令及 Manifest 七项均 NOT RUN。U/C 使用隔离 fixture；本批无默认 PG、无 ambient DSN，不启动旧 51936。

交回 Main：**G0 READY；无额外设计/Task/依赖/TP 决策请求。** 等待 Main 在原 claim 核收后交付原 10 文件 SOURCE/soleLocal 精确 grant；本次不提前实施或关闭任务。后续如证据要求清单外文件/契约变更或遇未知 writer/漂移，只停受影响部分并交具体最小影响。
