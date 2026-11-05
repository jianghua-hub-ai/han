# Han B13D / identity-auth-S4.14 / Backend-Dev 只读 G0

时间：2026-10-29T12:36:46.639Z。本包是当前 Role 的实施前核读与建议，不是新设计、Code Review、测试结果、任务认领或完成证明。完成本包后 STOP；SOURCE / Go / PG / process 全部 CLOSED。

## 结论

可以在原五文件内准备并随后实现现设计的中性 RestoreDriver / DeploymentExclusive 软件组件与局部验证；生产 OS 锁后端、受批准工具、稳定 handle 传递、原生耐久回执和实际来源验证未固定，真实启用必须在 Begin 前失败关闭。本 G0 不选择 SDK/平台后端、不发明签名/receipt 协议、不增加第六源或额外 build-tag 载体、不注册 production fallback。

## 当前对象与真实读取

- 固定 D:/AID/VSCode/Codex/Han，main，HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca，从 .git/HEAD 与 refs/heads/main 只读核实。共享大脏树来自交接；本次零仓内写入，不要求整树干净。
- 自身完整 Role、add-task-implementation、development-delivery、add-tdd-go 已全文读取。已读通用 coding/no-hardcode/code-review/task-execution/multi-agent/observability/tracing；已启用 go-service README 及 coding/no-hardcode/security/observability/tracing 规则。完整 hash/实际行段见 readset.json；不把搜索命中或摘要检查当全文读取。
- 完整原任务卡 tasks.md:2517–2561、通用关闭合同:2000–2010、稳定输入:1930–1949、Plan B13D:1529–1534；S4.7/S4.9 已闭任务头已核。
- identity design 当前 r54（仍 draft），SHA256 E823D25D9CDDFD4F2D7E3A869FBB6B15EAF5275E55147A21546D4C2836472522；已核 §3.3.1:1438–1662 与 §4.2:3463–3897，重点 RestoreDriver:3814–3828、部署独占:3598/3848、Enter/Resume:3628–3644。按本轮明确预期/范围和 standing 推荐采用授权承接，不把 draft 元数据擅改为人的 accepted。
- TP 原引用“登记前826–828,830”已按语义定位现件 §4.7.1:832–834,836；当前826是 restored 终态 ACK，不能机械按旧行号验错场景。当前实际工具/访问面段为828起；精确映射见 risk-scenarios.md。
- B13C 19源全部当前 SHA 与 Main closure receipt 匹配（scope.json）；不修改。Tasks 当前 FEC43089DF8F54DD3BAFAE0D87D01037FDFFB31ADB5984D9E51C60BE03D3F734、Plan EE1097D9FA41656709EA4B6BFCBC5EE37462C7B946841C12E0EC612E29807438，匹配交接 74/54/3/17 的记录。此为读取时快照；Planner 后续认领不由本包代写或冻结。
- 五目标文件实际均不存在。main.go/main_test.go、go.mod/go.sum 摘要已记录；Zeno identity 执行三路径、Noether publication 十路径与 Planner Tasks/Plan 保留各自所有权。

## 已存在的真实消费接口

- internal/recovery/runtime.go:46–59：RestorePlan 与 Prepare(ctx,RunBinding)、Replace(ctx,PhasePermit,RestorePlan)、VerifyRestored(ctx,RestorePlan)；admission.go:55–61 定义 RestoreReceipt。DTO 无 JSON tags，不能直接 Marshal 当既定 snake_case wire。
- runtime.go:146–155：DeploymentExclusive.Acquire(ctx) -> ExclusiveLease.Check(ctx)/Close()。锁域在恢复域外，不能借 Journal/User/PG 行锁、文件存在或 caller bool 充当部署独占。
- runtime.go:1431–1544：真实 Enter 先 Lifecycle.StopAndJoin，取得真实 lease，然后 readProof、Prepare、重核、Journal.Begin、readControlled 独立核 committed/精确增量，再经私有 phase producer 把 recovery_restore permit/PhaseContext 交 Replace。Prepare 失败或 Begin ACK/回读不可证必须 Replace=0。
- runtime.go:1547–1597：Replace 后重新核来源、VerifyRestored 且 receipt 逐值相等才 Bind；Resume 只 Prepare/Verify/readback，零 Begin/Replace。driver 不代 Runtime 执行 Begin、不返回 Ready、不设置 phase。
- runtime.go:1122–1180：Replace 消费实际 Runtime.CheckPhase(ctx,permit,PhaseRecoveryRestore) 与 ReadPermitBinding(ctx)，比较当前 run/backup/installation；PermitBinding 只是观察值，不是授权。拒绝零值、旧代、错实例/目的、失锁与取消，IO 前后重核；不 unsafe/reflection/linkname/导出 mint。
- runtime.go:1328–1399：Stop 等 action normalDone 与所有 work/operations，再 Close deployment。driver 的 Replace/Verify 不能在 child/pipe/target tx 尚活时返回；否则该真实 drain 会被提前释放。
- identity RecoveryRunStore.ReadAdmissionSnapshot（recovery_admission_store.go:18–94）可在新连接/短 RR 只读事务重核安装/全 run/restore binding；0010:1–64 已有 installation 与 immutable recovery_restore_bindings。该 Store 不等于“这个 artifact 确实还原”的原生证明，不能用刚写的新 run 自证。

## 五路径内实施分配（仅建议，当前禁止写）

| 精确 path | 将来本卡责任 |
|---|---|
| cmd/han-server/recovery_restore.go | 私有配置/依赖校验，现有 RestoreDriver 实现；批准来源及目标验证，中性 no-follow 稳定文件 owner、固定 argv/private stdin/严格 receipt codec、进程/pipe/target 终结管理、独立目标观察与 VerifyRestored 比对。所有类型保持 cmd 私有，不新增 recovery 公共契约。 |
| cmd/han-server/recovery_restore_test.go | 原 top TestRecoveryRestoreDriver 与对应风险 leaf；仅公开 NewRuntime/Enter/Resume 的 pubRuntime fixture，严格编解码/canary/handle 变换/unknown；真实子进程 fixture 的唯一 TestMain/helper 也留在该文件，不增加 helper 源。 |
| cmd/han-server/recovery_exclusive.go | 实现现有 DeploymentExclusive/ExclusiveLease 的中性设施适配：校验已有恢复域外路径与实际 facility lease，Check 始终到真实 owner，Close 单 owner/幂等；缺设施拒绝，不选定生产 OS 锁/SQL/file-existence fallback。 |
| cmd/han-server/recovery_exclusive_test.go | 原 top TestRecoveryDeploymentExclusive；三入口共用设施的配对竞争、失权/过代/取消、部分 acquire owned-handle 错误、Close 至多一次、真实 join 前 Close=0；跨进程 fixture 仅软件证据。 |
| cmd/han-server/recovery_restore_integration_test.go | 原 top TestRecoveryRestoreTargetReadbackPostgres；仅本既定 integration 载体内 P harness、独立目标连接/只读回读、错误目标/来源/schema/receipt 与 ACK/崩溃 unknown。允许沿项目惯例给这一个既定文件 integration tag；不增第六个或额外平台 tag 文件。 |

生产 assembly（命令分流、配置注入、normal/recovery/requeue 同 launcher、长生命周期 owner 清算）仍在后续 S4.16；本卡可以提供未注册的私有组件与 fail-closed 构造，不改 main.go/main_test.go。root 之外不创建第二 pool/RecoveryRun authority，也不复制 identity 终结/绑定/审计 SQL；本卡 target inspection 的只读边界单独注入。

## 软件实现约束与资源所有权

1. Prepare 从独立批准来源取得精确 plan，核 source approval/ref/digest、source installation/schema、目标非秘密 descriptor、approved binary SHA/格式支持/权限/进程树隔离前置。来源缺失与不支持平台在 Begin 前拒绝；文件自带 hash/inventory backup 名称不自证批准。source 与 target installation 严格分开。
2. stable artifact owner 只接受提供方已 no-follow 打开的 regular handle；父目录/路径身份、owner/ACL 与保护必须可核。完整 hash/长度、前后 fstat、restore/verify 读取均沿同 handle，seek 也不换 handle；路径/size/摘要变化拒绝。现有 inventory 的 Lstat+os.Open+SameFile 只能作模式参考，不是原生 no-follow 保证；recovery 私有 Unix opener 不能从 cmd 直接复用，也不扩大 Windows 支持矩阵。
3. artifact_path 是现有 plan 引用，绝不授 child 按可替换原路径重新打开。中性 runner 可消费保留的已核 handle；如何无额外 argv/JSON/secret ENV 将该 handle 安全交给实际批准程序尚未固定，保持 native adapter Gate。仅父进程持 handle、child 又重开 path 的 fixture 不可记同 handle 通过。生产工具 SHA 校验后路径替换风险也由真实工具/ACL/执行身份适配负责，不能散落 os/exec 路径调用绕过。
4. 唯一 argv 为批准 executable + restore-v1 或 verify-v1；不 shell，不附 caller flag。ENV 显式最小 allowlist，绝不 os.Environ 继承；OS 必需项/Trace 的确切注入须沿批准平台适配，不从 profile 占位符发明生产变量。ctx 保留原 Trace，按 root io 预算/父截止收紧。
5. 私有 stdin 恰 plan + target_connection；plan 精确13键固定顺序，target_connection 恰 dsn。用私有 wire DTO 显式 tag/序列化，不 Marshal 公共 Go DTO；DSN 仅接收 root 已校验值、不新设 TLS/default。复用 parseDatabasePoolConfig 的隔离/去敏意图，不能调用真实 getenv 或因 parse 错误包进 raw DSN。原 maintenance stdin 为 requeue 的 Windows 读入设施，不复用成 restore 传输或重开其 SID/签名/epoch设计。
6. stdout 计整个字节流，上限65536，至多再读1字节用于拒绝超限；先验证原始 UTF-8，再 token 级拒绝重复键（含转义等价键）、额外/缺失/null/错误类型、第二 JSON/尾随非空白。校验版本、正 int64 ArtifactBytes、逐值 binding、精确 UTC 六位微秒与固定有序 canonical receipt_digest。encoding/json 单独 DisallowUnknownFields 不足以拒绝 duplicate/null/无效UTF8。
7. stderr 原文直接丢弃，不累计/回显；保留低敏退出分类与父 Trace。持续消费但不留 raw，无限输出/堵塞用已批准预算取消并真 join。canary 覆盖 DSN/密码/审批材料样本，不能把canary写进留存 raw 来“证明去敏”。只记是否泄露/计数/分类，不持真实 secret。
8. 进程 owner 唯一负责 Start/Cancel/Wait 与 stdin writer/stdout reader/stderr drainer 的真实完成；Wait 至多一次，Start 后任意失败亦 join，不以 timeout/ctx/Done bool 或 WaitDelay 代所有 owner 已终结。超过共享关闭绝对截止记失败、停新业务，但继续等实际 owner；不能 detach 后归还锁/pool。直接 child 退出不证明继承 pipe 的后代已终结，native 进程树隔离不可证保持不启用。
9. Replace 至多一次且仅当前许可；其未知结果保留 ErrRecoveryWriteUnknown 与 ctx 可分类性，外部文本仅低敏分类，不把可能含 DSN 的底层 error 打入日志。VerifyRestored 是原工具的只读 verify-v1 + 新开目标独立读回 + 独立耐久来源逐值核；任一缺失为 restore_unknown/closed，不新 Begin/Replace、不用人工 done 补证明。
10. 仅 driver 内必要的私有 resource owner/cleanup，不修改 recovery.Runtime 接口。Prepare 成功后保留 handle 的失败/取消/动作退出所有权必须明确：本卡组件 fixture 显式关闭并真 join，未来 S4.16 root 持有同一 owner 至所有使用者结束。若完成所需行为实际要求新公共 close/mint/receipt 接缝、额外 native 文件或 root 提前接线，停止该部分，给 Main 精确最小扩边界建议。

## 验证与交付条件（全部未执行）

风险/层次/真 child 机制及 oracle 见 risk-scenarios.md；命令以原卡2535–2545和通用合同2006为唯一来源，不添加默认工具/依赖。go.mod 当前 Go1.27.0/pgx5.11.0，仅文件事实；gopter/testify未列入依赖，不为遵循可选技能示例改 modules/安装，先用现有 testing 表驱动和确定性生成覆盖风险；若必要方法确需新库另停相关部分。

未来 SOURCE 开启后先补可编译的行为 FAIL，再最小实现并留同次 RED/GREEN；LOCAL、PG、process 分别取得 Main 精确窗口才运行对应层。普通/race -json 要逐个 top+必要 risk leaf 实际命中且非Skip，-list/exit0不能替行为；七项 Manifest检查按原合同顺序且只读格式检查，不格式化共享树。实际 format/lint 环境缺口如实交 Main，不安装工具或修改他人文件。

自身最低 U/local/P与真实 child/pipe join均满足、精确源读集/工作树/无越界与非作者 Review/复审及 Main 同版核证之后，才可讨论本卡软件关闭。native锁/ACL/实际批准工具与原生durable receipt/真实restore/完整root/E2E不能借软件fixture记PASS，也不由本G0豁免；任务现在未完成。本包没有代码 diff、commit或测试结果；仅 Temp 文件及读取摘要。

## 当前停止边界

按本轮明确指令在本包完成后 STOP，不请求 routine 批准，不推进 SOURCE/LOCAL/PG/process。生产设施/工具/来源未固定的相关 adapter 继续在真实启用 Gate 停止；本轮没有为此启动新设计或改 SSOT。旧自有 PG stopped/保留数据来自当前交接与仓内 closure guard，不做进程/DSN探测、连接或重启，不复用旧 DSN。当前所有 child/PG/Go结果均 NOT RUN；未作整树 Git status/diff/测试、未声称人的接纳或 Task完成。


## 封包时并行元数据更新（只读续核）

初始摘要核对后，Tasks/Plan由其他写者发生本轮预期的认领更新；本会话没有写入。当前实读S4.14:2519为[-]、开始2026-10-29；2523登记Chandrasekhar/add-backend-dev、runtime Agent ID 01a187a1-c022-71f3-92f5-6a9dc032e32b，仅G0。Plan B13D:1532–1533同步三owner及所有运行窗closed。新Tasks SHA256 ED7B2E51DAF509B3292BC57347373A7708C8E28D2DAC940F5DEC608F2793BE25，新Plan FDBF3E26BD9B8063C3158299EF8056E3AE60427D44D46D11D670E2DEC8E71736；本轮不重算全仓任务总数，当前统计以Tasks SSOT为准。本卡五文件/依赖/验收与风险leaf原样。已读新增认领正文，与交接所述Planner元数据工作一致；未对整份metadata作独立diff或替Planner/Main接收签字。B13C19源、冻结main.go/main_test.go、go.mod/go.sum仍逐项原hash，五新文件仍不存在。最终本Role交付后STOP，SOURCE/LOCAL/Go/PG/process仍CLOSED。


## Main更新与跨包编译边界（本轮消息来源）

Main确认Goodall已STOP/Hostclosed，三卡认领已核收；当前Tasks ED7B2E51DAF509B3292BC57347373A7708C8E28D2DAC940F5DEC608F2793BE25、Plan FDBF3E26BD9B8063C3158299EF8056E3AE60427D44D46D11D670E2DEC8E71736，74/54完成/6active/14todo。本包前次观察摘要一致；meta变化为已知writer更新，不是新冲突，不触发routine批准。Main guard4842及“技术/files/commands/requires/risk/layers保持”作为Main提供事实记录，不冒本Role整树独立核证。

将来测试package仍为main。recovery_restore_test.go / recovery_exclusive_test.go直接跨包消费han/internal/recovery（真实公开Runtime producer）和han/internal/platform（真实Trace），不新增identity/article导入；integration文件计划额外消费han/internal/identity的真实Store、han/internal/migration、han/internal/testenv及现役pgx/pgxpool，不新增Article import。生产driver/独占私有组件只消费现有recovery中性ports与platform Trace；不增加identity→article或article→identity依赖，不导入对方私有test helper。上述为五文件内拟实现import集合，代码尚未创建、未编译。

即使单元测试不直接导入identity/article，cmd/han-server既有production root已导入两者，因此cmd普通或integration Go测试均会编译两域传递依赖；Main发Go窗口时仍须与Zeno/Noether源码写窗安全串行，不能因direct test import少就当独立编译。没有运行go list/test/build做依赖确认，也不复跑Main已完成静态检查。

Main新自有PG ready只作为未来环境preflight：本Role不读取postgres-resource.json/其私有DSN、不检查PID/端口、不连接SQL/PG、不启动child，不复用B13C旧DSN/autoDROP/global integration。所有权限仍CLOSED；本包完成后STOP，等待Main据原scope发精确窗口，不发起新的用户routine批准。
