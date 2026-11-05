# S10.5 单 requeue 六文档独立评审 R01

- 日期：2026-10-28
- Reviewer：本会话非作者独立 Reviewer；未另行委派，不替作者整改。
- 工作区 / 分支 / HEAD：`D:/AID/VSCode/Codex/Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 结论：**CHANGES REQUIRED**。两轮有界审查完成，发现 **1 HIGH、2 MEDIUM**；未给本轮整体 PASS。
- 唯一正式报告：`.spec/wip/identity-auth/review-design-s10.5-r01.md`。
- 本结论仅评审下列固定设计、ADR 与测试规划，不是 Tasks R 规划、实施授权、代码 Review、部署验收或 production readiness。

## 1. 完整对象索引与版本

冻结来源为 Lead 的 `contracts-fixed-r01.json`、`contracts-fixed-r01/` 逐文件副本；原始基线为同一 Lead 目录的 `knowledge-before/` 原相对路径。Reviewer 独立核验六份工作树文件、冻结副本及清单 SHA 相同，并从原始基线自行计算 diff。

| 对象 | objectType / 实际范围 | 冻结 SHA256 |
|---|---|---|
| `.add/project/knowledge/decisions/identity.md` | adr；新增 ADR-0026 `adr-0026-r02`，333–495 行，含最终实际引用登记 | `E40E6567D2F1E8C0053659D4929660BB42B77D82B66D68A055BAF004D6D924D1` |
| `.add/project/knowledge/decisions/README.md` | adr 导航；ADR-0026 全局索引、领域表、场景表与来源/count | `0F75650B78DE4AC1DD308C25D02246E09FC6EAB9931EA33D279EFD320608C998` |
| `.add/project/knowledge/design/architecture.md` | design；`architecture-ir01-r29` draft 的本次有界增量 | `4AD3B200937C37D53C7773E90AEAAFA6031E3B84C23CAD4A243122E179C80C9E` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | design；r46 metadata/来源、§3.3.11 maintenance、§9.4.3 五文件映射 | `CF3131198E757D13C5433ED4F5D12A721C5DD67F271090243C2219266ACED8D2` |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | design/Story；保留 r16 头部的 S10 maintenance 增量，365–372 行 | `E8D5EAD1F12D6C34F01730EB3761E6F49DEC3ECEFA776560C222C506AACC737E` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | design/测试规划；r21 的 S10.5 I-S105-01～25、子矩阵、U/PG/B 命令与部署接力 | `013216306D4DEFF293D5D695181F845B0DAFDF8CA727D69688997EAE2705C8EC` |

ADR/README 相对阶段一 B99F/1B75 的变化已逐项核对：只补实际 FD 引用、派生 count=2 及历史时点说明；ADR r02 技术协议未变，architecture r29 字节未变。因此复用阶段一技术检查，再检查新的 FD/TE 消费者，没有重新评审无变化的全仓材料。TE 保留的 B99F 是其实际读取的同一 r02 来源快照；本报告评审对象始终采用上表最终 E40E 摘要，不混用对象版本。

证据副本：`.spec/wip/identity-auth/evidence-s10.5/design-r01/inputs.json` 及其 21 项 payload，Reviewer 逐项哈希均相符。临时审查材料位于 `C:/Users/J00010385/AppData/Local/Temp/han-s105-reviewer-prep-d3b855d173/merged-ad39a14b/`，包含 `frozen/`、独立 diff、`objects.json`、`review-evidence.json`。这两类材料是取证副本，不是新增业务 SSOT。

## 2. 两轮实际检查范围

### Round 1：逐对象直接检查

1. ADR 全部新增正文与最终引用登记：真实 Windows owner SID、独立 approver、部署 trust/allowlist/epoch、plain Ed25519、固定消息域、严格 presence/nullable/canonical 字节、时间窗/撤销、User-first、完整 before、one enqueue、有限 job 写集、Audit UUID 消费、重放与 commit unknown。
2. ADR 导航与 architecture 本次 metadata、§6.2.3、§6.3.4、§8.2/8.3、§9.1、Q23 的采用、作用域与证据声明。
3. FD 全部本次增量：CLI/模式、typed config/opaque request、pre-DSN 构造与 DB 注入、最小事务、结果/退出、stdin 所有权/EOF/取消回收、五文件与实际部署边界；对照 S10 Story 增量。
4. TE 全部本次增量：25 个 planned Case、FD API/DTO 映射、严格编码/数字精度/审计子矩阵、stdin 未 EOF 矩阵、未来 Root/Operator/PostgresCLI 普通与 race 命令、零测试/skip 边界、S3.4 待协调接力与 NOT IMPLEMENTED / NOT RUN。
5. 跨六对象核对唯一协议来源、配置必需项、签名/审计载体、模式与 pre-DSN 时序、one enqueue/current worker policy、五文件职责、授权/验收/运行事实的区分。

### Round 2：沿真实生产者与消费者核可实施性

实际检查或在本次哈希不变的前提下复用先前已读取正文：

- `cmd/han-server/main.go` 的 run/runComposed、productionDependencies、共享 Store/Gate 与 operator/worker builder；`main_test.go` 的配置与共享实例 seam。补读 `database_config.go`，核现役 DSN 解析与 ambient PG 配置拒绝接口；未读取任何实际环境值。
- `internal/identity/password_reset_operator.go` 的 Execute/permit、User-first、lease 检查、settlement/审计和 commit 分类；`password_reset_worker.go` 的 Scan/Claim/Renew/Fail、MaxRetries/RetryBackoff 与真实 lease；对应测试只作静态证据，不称已经执行。
- `password_reset_store.go` 的 authority/delivery/lineage 与 Gate、Login/PasswordChange/Session resolve 的直接消费点；只核 S10.5 不得改动或旁路的约束，没有扩大为全 identity Review。
- `internal/audit/store.go` 的 Event、Append、Get、decodeMetadata；`0001_shared.sql` 的全局 UUID PK、system actor/text/JSONB 与 append-only；`0008_password_reset_durable.sql` 的 job CHECK、operation/FK/唯一/终态不可逆 trigger。
- 本机 Go1.27.0 的 `os/user/lookup_windows.go`、`lookup.go`、`crypto/ed25519/ed25519.go`；本轮新增读 `os/file_windows.go`、`internal/poll/fd_windows.go`、`fd_poll_runtime.go`、`fd_mutex.go` 与相关 pipe/exec 测试，核实际继承 stdin 的阻塞读取及关闭语义。
- 官方一手交叉依据：[Go Ed25519](https://pkg.go.dev/crypto/ed25519)、[RFC 8032](https://www.rfc-editor.org/rfc/rfc8032.html)、[Microsoft SID](https://learn.microsoft.com/en-us/windows/win32/secauthz/sid-components)、[CancelIoEx](https://learn.microsoft.com/en-us/windows/win32/api/ioapiset/nf-ioapiset-cancelioex)、[Windows 取消 I/O 的并发注意事项](https://learn.microsoft.com/en-us/windows/win32/fileio/canceling-pending-i-o-operations)。版本性判断以本机源码为准，不把网页版本当作本机执行结果。

已读取并按对象适用自己的 Reviewer Role、add-code-review、delivery workflow、review checklist、abstraction、安全红线、design 写作、decisions、no-hardcode、observability 与 go-service 相关规则；SDK 事实补证使用 add-research。没有加载或强套未启用 profile。

## 3. Findings

### [HIGH] F001：Windows 阻塞 stdin 的一次 Close 存在漏取消窗口

- finding ID：`S105-DES-R01-F001`。
- 对象/修订：feature design r46，上表 CF313；Story 与 test-plan 的相同承诺属于直接消费者。
- 位置：`design.md:2350`、`:2354`、`:2356`；对应 `stories.md:367`、`test-plan.md:581`。
- 责任：**FD Ohm** 修复输入/取消设计；TE Russell 随修复同步 I-S105-06/11/22 的竞态验收。不重复登记同根因测试 finding。
- 违反依据：本设计要求 operationBudget 到期时由子进程自行中断输入并 join reader/callback，不依赖父端 EOF/Kill；ADR §2/§4 的有限预算与失败隔离；评审清单的取消/资源收敛。

**现状与触发**：设计把“确认为 pipe + context.AfterFunc + sync.Once 只 Close 一次”作为中断生产 stdin 的实现接缝。当前 Windows 标准库不能让这一次取消覆盖尚未提交的阻塞 ReadFile。一个允许的交错为：

1. reader 进入 `FD.Read`，已经取得 readLock/引用，但尚未提交内核 `ReadFile`。
2. deadline 回调调用 Close。Close 标 closed，调用一次 `CancelIoEx(handle,nil)`；此时没有待处理 I/O，可返回 ERROR_NOT_FOUND。该调用不使仍有引用的 OS handle 失效。
3. Close 等待在读引用释放上；阻塞 handle 无 poller，`pd.evict` 不负责阻止该尚未提交的 Read。
4. reader 恢复执行，`pd.prepare` 在 runtimeCtx=0 时返回 nil，然后提交阻塞 `ReadFile`。父端不 EOF/不再给数据时，该 Read 不返回；最后一次取消已经过去，close-once 又不允许后续取消。

于是 Close/callback 等 reader，reader 等输入；helper 等待两者收束，operationBudget 不能使子进程有界返回。没有 DB 写入不减轻此资源/退出合同缺陷。此处不是声称 CancelIoEx 从来不能取消 pipe，而是最后取消先于内核提交时会漏掉这次读取。

**证据**：Go1.27.0 `os/file_windows.go:87` 对实际 Stdin 不探测 nonblocking，按阻塞方式初始化；`fd_windows.go:453` 的 isBlocking、`:543` 的 Close（一次 CancelIoEx 后等待引用）、`:580` 的 readLock 与 `:596` 的 ReadFile 提交；`fd_poll_runtime.go:64` 在无 runtimeCtx 时 prepare 返回 nil；`fd_mutex.go:229`/`:238` 的读引用与关闭释放。Microsoft 明确 CancelIoEx 仅作用于 outstanding I/O，不改变 handle 状态，找不到请求返回 ERROR_NOT_FOUND。上述是源码可证的交错，**未运行 Go/真实 Windows pipe 复现**。

**影响**：半份 envelope、完整 envelope 无 EOF 或延迟输入在截止竞态下可使维护进程、读协程和取消回调无限等待，违反必测输入预算/回收与稳定非零退出要求。

**建议方向**：由 FD 明确实际 stdin 载体与取消/提交同步的无漏取消合同，覆盖最后取消与首次/下一次内核读取之间的窗口；仅以“pipe 类型”和单次 Close 不足。保持父端不 EOF/Kill、真实资源 join 的验收要求；如需改变已确认载体/公共边界，交 Lead 决定，不由 Reviewer 选择 OS 实现方案。

**复审条件**：冻结修订说明该交错为何不能发生或如何被有界收敛；增加确定性的“读已取得引用/即将提交时发生取消”分支证据及真实父端保持开启的 Windows pipe 未 EOF 验收计划。设计复审核机制与五文件映射，未来实现再取得真实运行证据；不能以正常阻塞后 Close 成功的单例替代竞态证明。

### [MEDIUM] F002：唯一 ADR 配置清单与 FD 必读配置发生分叉

- finding ID：`S105-DES-R01-F002`。
- 对象/修订：ADR-0026 r02 最终 E40E 与 feature design r46 CF313；test-plan r21 也消费了 FD 的额外必需项。
- 位置：`decisions/identity.md:377`–`:384`；`design.md:2309`–`:2318`；`test-plan.md:569`–`:570`。
- 责任：**Arch Epicurus / FD Ohm** 对齐唯一部署合同及本地消费；TE Russell 随最终合同更新配置矩阵。
- 违反依据：ADR 明确“配置只从以下 ENV 读取”，列出 trust、maxTTL、operationBudget 和既有 DSN；ADR/architecture 均规定公共合同唯一归 ADR，FD 只作 Go/CLI 映射。

**触发与影响**：按 ADR 四项清单注入合法配置、提供正确实际 SID/审批请求的调用，在 FD 路径仍会因为缺少必需 `HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET`、`HAN_SHUTDOWN_BUDGET`、显式 `HAN_RECOVERY_MODE=normal` 而 pre-DSN 拒绝。这些不是请求字段的内部重命名，而是 launcher 必须提供的外部输入与启动条件。反过来，部署方也无法仅据 ADR 判断它是否允许读取这些键；ADR 自身另要求 recovery 冲突检查，但没有在该唯一配置清单中说明这个既有输入。

**证据**：FD 配置表明确复用 connection budget 正 int32 parser、必填 shutdown budget，以及 recovery mode 缺失也拒绝；TE 映射表同步这组要求。现役 parser 可复用不等于它们已进入 ADR 的唯一维护配置合同。这里不质疑复用现役 parser 本身，也不要求新配置或默认值。

**建议方向**：原作者统一“维护专用配置”和确需消费的既有配置的完整必需清单、唯一来源、缺失/非法及 pre/post-DSN 失败时序；或让消费者遵守原清单。哪一项保留由既有授权和真实需求决定，Reviewer 不替作者选择。

**复审条件**：ADR、architecture 引用、FD config/构造顺序与 TE 的合法最小环境/缺键负例一致；能仅从唯一合同构造完整合法维护 invocation，不再依赖另一文档的隐含必读键。冻结修订后核相应 diff 与消费面。

### [MEDIUM] F003：审计 metadata 的上限检查晚于 Get 的读取与解码分配

- finding ID：`S105-DES-R01-F003`。
- 对象/修订：feature design r46 CF313；test-plan r21 602–604 行是对应资源/损坏事件验收消费者。
- 位置：`design.md:2320`、`:2333`；直接依赖 `internal/audit/store.go:107`–`:169`，`migrations/0001_shared.sql:28`。
- 责任：**FD Ohm** 修复读取接缝；TE Russell 补对应真实持久读取边界。无须 Reviewer 修改 audit 包或 schema。
- 违反依据：ADR §4 要求 metadata 最大 64 KiB、字段解码前核上限，不允许任意大输入触发分配；test-plan `:602` 明确“超限拒绝、不大分配”。同 ID 异绑定/损坏事件必须 fail closed。

**现状**：FD 明确直接用现役 `audit.Store.Get(ctx,tx,requestID)` 取得消费事件，随后 identity 才检查 metadata 大小、13 个 string 字段与各 carrier。现役 Get 先将整个 metadata 扫入 `[]byte`，再由 decodeMetadata 对全部内容 `json.Unmarshal` 成 `map[string]any`，调用返回后才可能执行 identity 的限制。共享 JSONB 列只限制为 object，没有 64 KiB 约束；Append 也没有该通用大小限制。

**触发**：合法已签 request UUID 碰到已有、异动作或损坏的超大 metadata 事件，例如含一个超长 string/巨大未知成员的 JSONB object。该状态可由现役共享审计模型表达，不需要 S10.5 增加列、关闭 trigger 或改变 DSN。预期结果应是有界冲突/失败；按现有消费接缝，整块 bytes、string/map 分配已经发生，之后才拒绝。

**影响**：读取历史/碰撞事件会绕过设计承诺的分配上限，占用与数据库字段大小成比例的内存和解码时间；仅约束本次新写 metadata 或解码后的 base64 字段，不能保护该入口。足够大的事件可导致超预算甚至进程资源耗尽，不能用“最后仍返回错误”证明有界 fail closed。

**建议方向**：在 S10.5 允许的 operator 接缝内明确原始审计内容在传输/物化/通用解码之前的有界读取与拒绝合同，再复用现役 Get/Append 的有效部分；不要只把上限检查放在已经 materialize 的 map 之后。具体读取机制由 FD 设计，保持 root 零 SQL、不扩 audit/schema/第六源码的既定边界；若确需越界再交 Lead。

**复审条件**：修订展示大小 guard 位于何处、何时拒绝、如何保证没有先分配全部 metadata；TE 增加新隔离 schema 中同 request UUID 的超限/巨大未知成员事件，并区分 U 解码边界与未来真实 PG 读取证据。损坏/超限事件不得被解释成未消费后重新排队。

## 4. 无新增缺陷的已核范围与待决定事项

除 F002 的配置一致性外，本次没有发现需改变 Windows owner SID + 独立单审批或 one enqueue 选择的新依据。以下检查未产生额外 finding，**不构成独立部署 PASS**：

- 受保护 launcher/固定 binary/DSN/schema/trust、普通调用方不能改信任锚或取得写库凭据的边界已写明；ENV role/proofref 不冒充认证，真实主体 resolver 不由请求选择，缓存仅作为短命 owner 快照。
- 公钥/签名长度在 Verify 前校验，固定 plain Ed25519 域与原 P；严格 nullable presence、canonical 重编码、int64 与 13 项 string carrier 避免 JSONB/float64 丢精度；receipt 的 applied 与本次 already_applied 分开。
- User-first、锁内及 commit 前时效重检、完整 before、job-only + Audit 同事务、全局 UUID 跨目标竞争、过期/撤销/损坏事件不借 replay 绕授权的方向一致。
- operation/hold/terminal/credential/Session/Idempotency 零写，保护窗不提前，原 worker 再拿真实 lease；MaxRetries 2→5 后可多次自动 attempt 不等于第二次 enqueue，未通过改 worker/重置 count 承诺 exactly-one attempt。
- pre-DSN opaque API 与 DB 依赖分开；同 binary/原 root、无参数 server/非 Windows server 兼容、零服务活动计数与五文件映射已给出。F001/F003 要求补足具体接缝，不据此擅定第六文件。
- U/PG/B/D 分层、未来命令的精确函数匹配/无 skip 假绿、子 binary race 证据分开、retain-only harness 和 S3.4 部署接力已明确。没有把 25 项 planned Case 当 Run，也没有把本地合成注册当生产签发。

本报告不另作架构采用决定。F002 的清单收口、F001 的输入取消实现机制及 F003 的有界读取机制由原作者在权限内形成修订；实质公共契约/文件边界变化交 Lead。Tasks/Plan 与 S3.4 接力仍是后续原责任方工作，不因本评审自行更新。

## 5. 分对象结论与复审要求

| 对象组 | 当前结论 | 后续归属 |
|---|---|---|
| ADR/导航/architecture | F002 跨文档配置合同须收口；没有另行否定已采用的 trust/签名/one enqueue 选择 | Arch Epicurus 协调 FD；如修改协议，同步导航/消费者 |
| feature design + stories | **CHANGES REQUIRED**：F001 HIGH、F002/F003 MEDIUM | FD Ohm 最小修订后由本 Reviewer 独立复审 |
| test-plan | 25 项与分层已检查；须随上述设计修订同步输入竞态、配置最小环境和审计读取边界，当前不予整体 PASS | TE Russell 消费最终修订，不代作者选实现；本报告不重复登记同根因 finding |
| 六文档整体 | **CHANGES REQUIRED** | Lead 恢复真实原 owner；冻结修订/精确 diff 后交本 Reviewer 核原 ID、直接消费和回归计划 |

复审须给修订后对象 SHA、精确 diff、原 finding 对应处理、作者静态分析/适用验证及限制。设计/规划修复不要求先有尚未授权的 Go/PG PASS；F001 的真实 Windows 运行、F003 的真实 PG 读取证据属于未来实施验收，设计先给可实施且可验证的合同。不得仅把问题改写为 NOT RUN 或删弱预算/上限验收以关闭 finding。原失败与本报告保留，不由 Reviewer 改被审产物。

## 6. 实际验证与未验证边界

本轮实际完成只读 PowerShell/Python/rg、SHA256、固定对象与原基线独立 diff、branch/HEAD 核对、SDK 源码读取与官方资料交叉核对。未运行 Go 编译/单测/race/vet、PG、CLI、SMTP、DSN/credential/环境读取、账户/GRANT、部署/E2E；未做 Git mutation、清理或委派。

报告写入前复核：六对象结束 SHA 与冻结值相同；**185 个源码文件无漂移，其他 27 份基线知识无漂移，21 份持久输入副本哈希一致**。只写自己的 Temp 与本报告；未改 Tasks/Plan、代码或六份被审文档。

局限包括：F001 为本机指定 Go 版本的静态交错证明，未动态复现；F003 为真实读取代码与可表达数据库输入的静态资源分析，未造库或测内存；其他将来实现仍须独立代码 Review。没有穷尽全仓/全部身份功能，也没有重开旧修订的历史状态或自审决定。

实际 launcher/ACL、生产 key 注册保管/签发、第二真实 SID、撤销排空、跨备份/新 epoch 操作、真实 ack-loss 注入及 S3.4 业务接收均未验证；不承诺即时在线撤销、自然人在场或跨备份 nonce。缺生产 key 不否定已选软件方案或禁止合成验证；合成验证也不证明物理隔离或生产签发已交付。人的精确 accepted/checkpoint 与后续动作授权继续分别处理。

**交回状态：stopped。** 本次两轮有界结果与三个 finding 交 Lead；不自行启动 Planner、实施或其他 phase。
