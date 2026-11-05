# Architecture r32 / ADR-0027 r02 — F001 独立复审 R02

**结论：有界文档复审 PASS。ARCH-R32-R01-F001：CLOSED（设计/ADR/index 层）；新增 findings：0。**

原 MEDIUM finding 要求的锚定生产者、耐久确认、失败协调、可信迁移和后继责任已进入中央合同。此结论不表示实现、运行、测试、迁移或生产就绪通过，不关闭任务/Case18，不替代 Main 现场核收及人的接纳。未以风险豁免关闭 F001。

**归属与固定对象**

Reviewer=add-reviewer / Kuhn，原作者=Pascal。本角色由 Main 依用户 standing 授权及项目委派规则指定；实际 runtimeID `01a1835a-8010-7a73-a8dd-3192719b6965` 来源于 Main 的真实 spawn 返回，已在 R01 不可覆盖存证 index 中澄清。不是用户直接选择 Kuhn；本轮不改写 R01 报告中原有措辞。

固定 `D:\AID\VSCode\Codex\Han`、main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。ObjectType=design/architecture + ADR + index，只复审 F001 及其修订相关新风险，没有重新全库审查。共享仓库只读；仅在本独占 Temp 写报告、readset、verification 和 receipt。没有调用 Git 命令：通过只读 `.git/HEAD` 与 loose ref 文件核对分支/HEAD。

| 冻结对象 | 修订 | 审前/审后相同 SHA256 |
|---|---|---|
| `.add/project/knowledge/design/architecture.md` | r32 / F001 作者修订 | `F0D1334187A5E7533CF31C88EFCE4E3BB84F36EE484B01A1BDD9B7DB0D382CBA` |
| `.add/project/knowledge/decisions/framework.md` | ADR-0027 r02 | `2825D595FFC6C347DAC41142B23AB22DC29B543A1E2924D6C42744A76E7A7C71` |
| `.add/project/knowledge/decisions/README.md` | r02 影响面/场景索引 | `2F2F3B2C70D07218A06217AA8D1A15B3CC2C67C7F35BB99583F0A0E4EEC90C6E` |

原 R01 报告/receipt 在 `.spec/wip/recovery-runtime/evidence-architecture-r32-r01/review-r01/` 保持原样。实际核得报告 SHA256 `81BFE22BC1481630602176CDAD45AA19971BB7B3B328188CB91680DDDFDF0B53`，receipt SHA256 `5B56CE75144C159B6FB1D73375152DFFB90867265585492AA1333EAA64E384BB`。

修复交接包：`C:\Users\J00010385\AppData\Local\Temp\han-arch-r32-f001-40e41c45b6d74bdaa2b77c9803ae30bb`。完整读取三份精确 R01→R02 diff、handoff 与 verification；作者失败场景推演作为作者证据，以下判断来自本角色独立重核，不将作者声明当复审通过。

Role 与 add-code-review 在 R02 开始时重新完整读取；本会话 R01 已实际取得的必需 Rule 全文按同 SHA 复用，逐项记录于 readset，未加载不适用 profile 或未来实施方法。Main 已核的范围/links/编码/原 framework 前缀/ADR 一引用静态集不重复跑、不冒称本轮自跑。

**两轮实际复审范围**

Round 1：逐条对照原 R01 F001:44–53 的修订要求，检查完整三 diff 与当前 architecture §6.2.4/§6.2.5（591–636）、§7.1:1053、§8.3:1165、§11:1588/1594、§12.1:1676；ADR0027 r02:759–806 和索引:35/41/61。检查 owner、调用方向、协议、错误/取消、兼容/迁移、中央约束与下游实施责任是否同步。

Round 2：仅沿相关真实生产者/消费者核可实施性与新增失败风险：

- `internal/article/deletion_journal.go:84–100,345–421,542–610` 的 Options、锁、Append/幂等命中、persist；完整文件 SHA 与 R01 一致，R01 对其构造器及 readVerified 的其余相关实读证据仍有效。
- Main 追加疑点后，实读 `deletion_worker.go:121–195`、`deletion_finalizer.go:133–270,275–368`、`deletion_replay_source.go:98–147`，核 Reconcile→Converger→Append 及 ReplayAll→convergeAndCount→幂等 Append/operation 重建。
- `cmd/han-server/lifecycle.go:120–165,284–329,560–565` 的启动 Reconcile、join 与资源关闭；`internal/recovery/coordinator.go:58–94,129–158` 的恢复消费阶段；关联当前 architecture §9.1.1:1375–1394。
- 以修订合同独立逐状态推演文件与 Anchor 各崩溃窗口、ACK loss、ABA、正常读取零修复，以及临时屏障/故障关闭。没有运行代码、模型检查器或故障注入；下表是静态契约审查结果。

**F001 分项处置**

| 原 finding 修订要求 | 当前依据 | R02 状态 |
|---|---|---|
| 消除 article“仅补历史 port”限制，明确锚定写者及独立 CAS namespace | architecture:608–612、1053、1588；ADR:770/778/800；README:35/61 | CLOSED：article 为 Anchor 推进/协调唯一领域写者，root 注入；不引入第二 run authority，不让恢复机制散写 article Anchor |
| 正常删除推进真实可信 head，确认耐久后才允许 finalizer/204 | architecture:614–616；ADR:800 | CLOSED：验 H0/r0→暂存 H1 并 sync 文件/目录→CAS pending 精确绑定→不覆盖发布/目录同步/清暂存别名→CAS committed H1→独立重读确认；原 HMAC/payload/operation 幂等/物理删除与审计顺序保留 |
| pending、两侧写 unknown 与崩溃有唯一受控处理 | architecture:617/621–630；ADR:801 | CLOSED：旧头可核暂存或撤销未确认 pending，新头只提交同一 pending，第三状态/回退拒绝；不盲重放，不向 finalizer 提供未 committed tombstone，不扩大 RecoveryRun 唯一补写窗口 |
| 可信一次性首装/存量迁移，不能 normal 首读认可合法前缀 | architecture:634–636/1165；ADR:802 | CLOSED：停旧写者并排空，独立完整链来源/期望 head，CAS 不存在前置、精确安装绑定；中断受控续接，已有 Anchor 不重置，新旧不混跑 |
| 正常运行中的 pending、fault、取消与恢复权限边界 | architecture:617–621；ADR:800/801 | CLOSED：pending 前同步停止新 permit/claim，不锁内等待自身排空；unknown 升级关闭、真实 join，迟到成功不能自动开放；健康提交仅解除自身原因，不清其他关闭原因、不重 Consume |
| 双 namespace/Store revision、ABA 及后继验证责任 | architecture:602/632/1594；ADR:796/801 | CLOSED：每 namespace 独立 CAS 键/revision，撤销回 H0 也不复用 revision；Store 完整集合/字段及独占单向转换证明，无证明则拒绝，不用文件 hash/MVCC 冒防回退 |
| 实施范围、ADR/index 与原任务门槛一致 | architecture:1053/1587–1596/1676；ADR:798–806；README:35/41/61 | CLOSED：纳入全局 runtime 新实施范围；测试卡无生产写权，不重开 S4.2/S10.5，不自动改 S4.5 范围/requires；真实 CAS/retirement 与环境门槛不减 |

F001 整体为 CLOSED，原因是设计责任与中央协议已补齐，不是实施完成或已有人接受风险。无 OPEN 子项；实现与真实层验证列入下文未验范围，不作为本次设计 finding 的 LIMITED 关闭。

**独立静态失败场景核查**

| 场景 | 依据及判断 |
|---|---|
| 首装 H0→正常删除 H1→normal 重启 | :614–616 为 H1 维护真实独立 committed/revision；:596/598 的 genesis 延伸/历史闭包允许合法 head 前进；不需要重新 Consume，也不能沿旧仅文件 sync 路径确认删除 |
| 合法旧前缀、缺失及无对应 pending 的新 tail | :630 明确拒绝，不重置 Anchor、不裁剪或认可无授权 tail；与 F001 原触发不再冲突 |
| 暂存后、CAS pending 前崩溃 | :625 将孤立暂存视为未确认候选，normal 不修/不当删除证明；只受控核来源后处置 |
| pending 成功、最终文件尚 H0 | :626 精确暂存才能完成原发布；无可验新记录只能撤销未确认 pending，并换 revision、不确认删除成功 |
| 已发布 H1、committed 前崩溃 | :627 校验完整链、operation/暂存绑定与目录耐久后只终结同一 pending；不追加第二 tombstone |
| CAS ACK loss、committed 后返回未知或取消 | :628/629 以独立 Anchor/文件精确回读分类；:617/619 保持故障关闭并真实 join，迟到成功不自行解锁；无“所有 unknown 均可重放”的扩张 |
| normal 构造器/读取遇 pending、暂存或未锚定 tail | :591/621 禁止初始化、CAS、修 pending/文件，现有 ensure-directory 和收敛 reader 不能直接用来宣称只读；不能提供未 committed 记录驱动删除 |
| 正常 append 的短暂屏障及并发关闭原因 | :617 在 pending 前同步禁止新工作，只确认屏障建立，不等自身；:619 健康后继须核连续 head/revisions、无 pending 且无其他关闭原因；不恢复故障 permit、不建立 SetReady 权威 |
| pending 撤销回 H0 的 ABA | :632 必须产生不可复用 provider revision，不能因 head 重复而沿用旧结果；两个 namespace 分别前后核，不跨 namespace 比 token 大小 |
| 迁移残缺/可信来源缺失 | :634–636 不接受当前目录自身作为完整来源；unknown 保持关闭，不靠 normal 自动补锚或假值通过 |

这些场景在中央合同层未发现新增可坐实缺陷；不把有意的短暂 503、CAS 故障拒绝或缺迁移来源拒启当作 bug。

**Main 追加 Reconcile/ReplayAll 疑点：不形成新 finding**

这条真实调用链已经逐段读到，不能仅凭“恢复 Finalizer”名称忽略 Append：

1. 普通启动/恢复完成后的 Reconcile：`DeletionWorker.Reconcile:183` 调 Converger；pending_journal 在 `ConvergeDeletionOperation:150` 调 Append。architecture:1390 明确先由 root 消费同一 runtime 的有效准入结论，再保持**请求屏障**关闭执行 Reconcile/Start。内部有效准入与公共 HTTP/readiness 开放不是同一条件，因此无需先开放公共业务才运行 startup Reconcile。FD 应用目的/阶段受限的内部许可表达这个差异，不能把 PublicReady 布尔值复用成唯一能力。
2. 恢复过程中 ReplayAll：`ProductionDeletionReplaySource:113` 读取已核 Journal，:121 交 Finalizer；`convergeAndCount:227` 仅在恢复库 operation 仍 pending_journal 时调用 Append。对于该输入 tombstone，正确锚定实现的这一调用是**精确已 committed operation 的幂等重读**，不是创建新的 H1；若 operation 不存在，`rebuildOperationFromTombstone:299–308` 重建为 journaled，跳过该 Append 分支，再由 `FindByOperation:256` 验已存在 tombstone。
3. 中央承接依据：§6.2.5:612 将 step1 的“有效业务 permit”置于“正常 Append 的固定协议”范围，:614 单独规定精确 committed 幂等命中，:621 允许无修复只读；§9.1.1:1392 明确恢复专用 Finalizer 仅由受控编排显式调用，:1385/1388 持续禁止普通 worker/request。这些合同允许 FD 在同一 Journal/受控调用链细化局部 capability 或区分只读命中与新追加，不需新建恢复权威、先开公共流量或改 operation 幂等。
4. 边界：上述判断**不授权恢复 caller 绕过校验追加任意新 tombstone**。精确 committed 命中必须核 request/operation/namespace/head 绑定；缺失、错绑、pending、未知均失败并保持公共关闭。真正修改 Journal/Anchor 的受控协调仍须 :621 的关闭、排空、独占和 namespace 锁前置。若 FD 把“有效业务 permit”实现成“PublicReady 必须 true”并在幂等判定前无条件拒绝，或把 normal 的屏障解除用于恢复阶段，即违背 :1390/1392，不能作为本报告通过的实现。

这是对现有合同作用域与实际消费者的解释，不添加一个中央例外或风险豁免。后继 FD 应明确记录 capability/调用分类并回归上述两条真实链，避免把阶段区别写丢；本次没有因尚未固定 Go 类型/函数名而新报 finding。

**新增 findings、待决定与未验证范围**

新增 findings：0；本轮范围内无未关阻断。未新增必须由人重新采用的方案决定；standing 授权来源沿 Main，实际人签字与接收仍 unknown，不补签。

实际做过：冻结三 SHA 核对、完整修复 diff 与定向正文读取、同版本规则复用校验、相关真实源码调用链检查、上述逐场景静态推演、实读路径 SHA 稳定性核查，以及 R01 原报告/receipt 哈希核对。详细路径、范围、哈希在 `readset.json`/`verification.json`；没有声称所有检索到的文件都全文审查，部分宽定位输出截断后只对关键相关段补读。

没有运行 Go/PG/构建/单元/集成/契约/race，未启动服务、worker、listener 或 HTTP；未读取 DSN、环境秘密、key、provider 凭据；无网络/安装/Git命令/清理/子委派/共享修改。Main 的现场静态检查及作者 verification 的大保护集/图检查分别归属原执行者，未重复运行、不作为 Reviewer 动态证据。

PlantUML 未安装的限制保留，语法检查与 PNG 渲染 NOT RUN，未给渲染 PASS。真实 POSIX/fsync/跨进程锁、独立 CAS 故障域/并发/ACK-loss、实际可信迁移来源和签发主体、真实备份/inventory、应用 ACL、epoch/retirement、RPO/RTO、部署 E2E 与人的接纳，均未由本轮验证。只有后继获准的软件/真实环境验证才能确认实现履行本合同。

article S1.5/S4.4/S8.2 仍 active、Case18 整项未通过；S4.5 真实 CAS 与 S4.6 retirement 门槛保留，不重开 S4.2/S10.5。本报告仅关闭原设计 finding，不是 Task 状态回填、人的签字、实现授权或成果采用。Main 现场核三 SHA、报告/readset/receipt 后方可接收。

**交回与后续复审条件**

若这三文档、中央权限/耐久协议或上述消费者解释被后续修订，应固定新 SHA、精确 diff 并独立检查受影响部分。实现评审须针对 staged→pending→published→committed、normal 零修复、startup/recovery 内部授权、真实 join/失败关闭和幂等命中留下实际证据；本轮不预签。R01 原样保留，本 R02 单独落档。报告/receipt 完成后 **STOP 所有写入**，不清理 Temp，不推进其他角色。
