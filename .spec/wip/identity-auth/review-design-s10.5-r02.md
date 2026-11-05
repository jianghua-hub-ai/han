# S10.5 单 requeue 六文档独立复审 R02

- 日期：2026-10-28。
- 实际 Reviewer Role：**Dewey / add-reviewer**；真实 Agent ID：**`01a180fe-c3bb-7661-86fb-06af4f76ba1b`**。
- 归属：Lead 实际委派的非作者独立 Reviewer；本 Reviewer 未再委派子任务，未替原作者修改被审产物。R01 的“本会话……未另行委派”指没有继续分派子任务，不是否认 Lead 的真实委派。
- 固定工作区 / 分支 / HEAD：`D:/AID/VSCode/Codex/Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- **结论：PASS，限下表冻结六对象的 S10.5 设计、ADR 与测试规划。** 两轮有界复审完成；R01 三项及 R02 Phase1 新增一项 finding 均 **CLOSED（设计及规划层面）**，新增 finding **0**，本 Gate 未关闭阻断项 **0**。
- 本结论不构成 Tasks R 规划通过、实施授权、代码 Review、测试运行、E2E、部署验收、生产安全隔离交付或人的 accepted/checkpoint。各未来实现与用例仍为 **NOT IMPLEMENTED / NOT RUN**。
- 唯一正式输出：`.spec/wip/identity-auth/review-design-s10.5-r02.md`。R01 和 Phase1 原始证据保持不变。

## 1. 完整固定对象索引

冻结来源：`C:/Users/J00010385/AppData/Local/Temp/han-s105-lead-_eqdfl5h/contracts-fixed-r02.json` 的 `objects`，及同目录 `contracts-fixed-r02/` 逐文件原字节副本。Reviewer 已核工作树、清单、Lead 副本和持久副本 SHA256 一致；复审期间未把在途 TE 草稿当固定对象。

| 对象路径 | objectType / 修订与本次范围 | 最终 SHA256 |
|---|---|---|
| `.add/project/knowledge/decisions/identity.md` | **adr**；ADR-0026 `adr-0026-r03`，含完整七 ENV 修复及实际引用登记 | `BCB5707C90E008D4200087750361A6A6C648BCF0822691A3BD8BDBF8EB0F6442` |
| `.add/project/knowledge/decisions/README.md` | **adr 导航/索引**；ADR-0026 r03 索引、来源与派生 count=2 | `A0D02A02ED4EC5A32FC7A2AF38BCE193D287A2BFD25240A3F2EE4A8A77086CC6` |
| `.add/project/knowledge/design/architecture.md` | **design**；`architecture-ir01-r30` draft 的 S10.5 有界增量 | `7854417E2FD2090FE0EB3EAE1BF54A2502F3393764D3CE77462331F33D07C5D5` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | **design**；`identity-auth-design-ir01-r48` draft，maintenance §3.3.11、§9.4.3 文件/API 映射、相关元数据与修复说明 | `69309241DFAEB448383EE2C63CA494B4ABAD8CE0AC5DE29A1555F7D54A5CCEB5` |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | **design / Story**；保留历史 r16 头部的 S10 maintenance r48 修复增量，特别是 365–374 行 | `5DAE6F90D143FB02324107AB782694EF15CB293ABB24AD6CE4D3D0F216D6FC52` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | **design / 测试规划**；`identity-auth-test-plan-ir01-r22`，S10.5 §4.4.3、API/子矩阵、未来命令与 S3.4 接力；含 Lead 两处计数措辞修正 | `E431119C5A0EC2F884A00DF85070212994C3A0C3B078CD00F6BFFD351F3F0082` |

最终持久取证索引：`.spec/wip/identity-auth/evidence-s10.5/design-r02/inputs-final.json`，SHA256 **`411BD4A8B5ED9895CEEAFA4E9C487F195BC471706554FC23CC6AD764015AC300`**。其 **14** 项 payload 已逐项核验，包括最终六对象、TE diff/readset/验证材料、Lead copyedit 证明与原始 TE 快照。它们是证据副本，不新建业务真相源。

### 最终 test-plan 作者与字节来源

TE Russell（`01a180fb-ea7c-7331-8ebf-b9855dfdd634`）原始 r22 冻结 SHA256 为 **`DE9E4DF6B8C7EE8360B65B99095AE5C3B9A3BA24D1A6C05E15C05E6A4FC940E5`**。TE 写者关闭后，Lead 仅作以下两处措辞修正：

1. `或加第六文件来证明错误承诺` → `或扩出七路径来证明错误承诺`。
2. `第七文件必须独立fixture` → `集成测试载体必须独立fixture`。

Reviewer 对最终 E431 原字节反向替换，结果与原始 TE 快照逐字节及 DE9E SHA 完全相同，并核 `test-plan-r22-lead-copyedits.json`。因此最终来源是 **TE 正式 r22 + Lead 两处措辞修正**；未伪称最终 E431 等于 TE 原始冻结摘要。25 个 CaseID、命令与其余验收要求没有借 copyedit 改写。

## 2. 两轮实际复审

### Round 1：固定对象及修复直接检查

复审沿原 finding ID 与原对象差异进行：核 ADR r03/索引/architecture r30 的七 ENV 收口，FD r47 输入与审计读取修复，r47→r48 SDK 关闭与构造所有权修复，再核最终 TE r22 的真实消费。未重新审查无变化的全部 identity 历史设计，也未把历史 S10.1 OPEN/未启动语句当作当前 S10.5 缺陷。

实际检查包括：ADR 配置唯一来源与确认/部署边界；FD 的 root 模式与 pre-DSN、typed opaque request/构造顺序、User-first/完整 before/事务、stdin 正常与取消/EOF、审计读取与重放、SDK 生命周期及七路径映射；Story 对应验收；最终 test-plan §4.4.3 全部 25 项及输入/资源/七 ENV/F003 子矩阵、API/DTO、U/PG/B 命令、S3.4 与 NOT RUN 声明。已审且哈希不变的协议与生产代码读集复用实际既有检查，不以作者“已修复”代替内容核对。

### Round 2：直接生产者/消费者及实际接口

沿本次契约实际依赖核对，未扩成全仓 Review：

- root 的 `run`/`runComposed`、`productionDependencies`、operator/worker builder 与现有测试 seam；必要的 `database_config.go` 只核 parser/ambient PG 拒绝代码，未读取环境或实际 DSN。
- `password_reset_operator.go`/测试、worker/store 及其直接 authority/capability/Gate 消费点；核现役仅 `Execute(ctx, leased handle)`、真实 lease 与 current worker policy，不能把未来 Requeue API 写成现有功能。
- `internal/audit/store.go` 的 Event/Append/Get/decodeMetadata；`0001_shared.sql` 全局 UUID PK、JSONB/text、append-only；`0008_password_reset_durable.sql` job CHECK、operation/FK、终态与不可逆保护。未更改 audit/store/worker/schema。
- 本机 Go **1.27.0** 的 Windows SID、Ed25519 长度/算法、Windows file/poll/关闭及系统 DLL 加载源码。公钥/签名长度在 Verify 前检查；真实进程 owner SID 不由请求字段或 ENV role 自证。
- 本机 **pgx v5.11.0 / puddle v2.2.2**：BeforeConnect/ConnectTracer/AfterConnect 的真实调用顺序，QueryTracer 类型要求，Lookup 先于 Dial/ConnectTimeout，构造与 Acquire 取消分离，pool stats、健康 Release、BeforeClose、Conn/PgConn.Close、CleanupDone，及 QueryResultFormatsByOID 的真实参数消费。没有把 Go 接口设想当成 SDK 已支持的事实。
- 官方原始资料与本机源码交叉核对：[Microsoft PeekNamedPipe](https://learn.microsoft.com/en-us/windows/win32/api/namedpipeapi/nf-namedpipeapi-peeknamedpipe)、[GetNamedPipeInfo](https://learn.microsoft.com/en-us/windows/win32/api/namedpipeapi/nf-namedpipeapi-getnamedpipeinfo)、[pgxpool Config](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool#Config)、[PostgreSQL Read Committed](https://www.postgresql.org/docs/18/transaction-iso.html)。具体 SDK 版本行为以本机所读源码为准。

上述两轮及下列关闭结论均为独立静态设计审查；作者 Backend G0/Lead 现场检查只作辅助证据，未替代 Reviewer 判断。

## 3. Finding 闭环与原始失败链

| 稳定 finding ID | 原严重度 / 真正修复责任 | 本次状态 |
|---|---|---|
| `S105-DES-R01-F001` | HIGH；FD Ohm，TE Russell 同步消费 | **CLOSED — 设计及最终计划覆盖** |
| `S105-DES-R01-F002` | MEDIUM；Arch Epicurus / FD Ohm，TE Russell 同步消费 | **CLOSED — 公共配置合同与消费一致** |
| `S105-DES-R01-F003` | MEDIUM；FD Ohm，TE Russell 同步消费 | **CLOSED — 读取前边界与并发合同** |
| `S105-DES-R02-F001` | HIGH；FD Ohm，TE Russell 同步消费；起于 r47 Phase1 | **CLOSED — r48 主动 SDK Close 顺序与最终计划覆盖** |

### R01-F001：阻塞 stdin 最后一次取消早于 ReadFile 提交

原对象为 design r46，SHA256 `CF3131198E757D13C5433ED4F5D12A721C5DD67F271090243C2219266ACED8D2`，原位置 `design.md:2350`、`:2354`、`:2356`。原方案可在读引用已取得、内核读尚未提交时执行唯一 CancelIoEx/Close，随后新提交的阻塞读无人取消；父端保持开启时无法有界退出。原 finding 是标准库可证交错，未声称动态复现。

**修复与独立复核：** r48 `design.md:2365`–`:2377` 给出同一 goroutine 串行 Peek 正可读量→不超过已观察字节的原生 ReadFile；无数据时只用可取消单次 timer。读前/读后/接受 EOF 前核原 D，取消与 EOF 竞争时取消优先；仍要求实际 EOF，不能把完整 JSON 当 EOF。取消不再通过唯一 Close 回调追赶尚未提交的阻塞读；最后一次原生调用结束后才唯一 Close。Windows/!windows 两个编译选择适配文件、固定 System32 系统 DLL 接缝已映射。

该机制明确依赖受信 launcher 提供本地匿名 byte pipe、跨进程/线程/所有 duplicate handle 的唯一读取者、无同一 file object 的并发 I/O、关闭父端多余 read handle；OS byte/type 属性不认证来源或唯一消费者。Microsoft 对多线程同步 Peek 可能阻塞的说明没有被省略，也未把 Go 进程说成单线程；正可用量后再读成立于上述部署前提，不能任意套用未知 pipe。

最终 test-plan `:590`–`:598` 与 I-S105-06/11/22、D 层 Case24 对应半包/完整无 EOF、父端不关闭、正可用量后读前取消、实际回收与 launcher 前提。旧普通 close probe 的六次观察不作为新机制或竞态证明。原缺陷路径在设计中已替换且未来验证明确，故关闭；实际 Windows 运行仍 NOT RUN。

### R01-F002：ADR 四 ENV 与 FD 必需七 ENV 分叉

原位置为 ADR r02 `decisions/identity.md:377`–`:384` 与 FD r46 配置段；按 ADR 四项部署会被 FD 另外三项 pre-DSN 要求拒绝。

**修复与独立复核：** ADR r03 `:379`–`:397`、architecture r30、FD `:2308`–`:2325` 与最终 TE `:626`–`:640` 现一致限定完整七键：

`HAN_PASSWORD_RESET_MAINTENANCE_TRUST`、`HAN_PASSWORD_RESET_MAINTENANCE_MAX_TTL`、`HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET`、`HAN_DATABASE_DSN`、`HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET`、`HAN_SHUTDOWN_BUDGET`、`HAN_RECOVERY_MODE`。

六项非 DSN 配置及主体/签名/时效通过才读取 DSN；connection budget 的规范正 int32 校验先于 DSN，只有与解析后 MaxConns 的关系检查属于 parse 后、connect 前。shutdown 显式 1s–60s，经共同 parser 注入四字段 config 的 `ShutdownBudget`，无默认；recovery 只接受显式 normal 以排冲突，不承担认证或维护模式选择。DSN 缺失/空白与 DSN 语法错误的 lookup/parse/connect 计数分别表达；其他业务 loader/worker 全配置不被顺带读取，ambient PG guard 只拒绝输入，不形成第八配置源。Case02/05 及配置矩阵已消费这一时序，故关闭。

### R01-F003：上限检查晚于 Get 物化/解码

原位置 `design.md:2320`、`:2333`；真实 `audit.Store.Get` 先 Scan 全部 metadata，再 json.Unmarshal，事后限长不能保护同 UUID 的巨型历史事件。

**修复与独立复核：** r48 `:2325`、`:2337`–`:2348` 明确 ReadCommitted 事务；User-first/operation/job 锁与当前授权/请求验证后，在 Get 前按全局 `id=$1` 执行仅返回 **int64 字节长 + binding bool** 的 scalar guard。长度是 `octet_length(convert_to(metadata::text,'UTF8'))::bigint`；其余 Event 的无界 text 在 SQL 内比较固定预期值及 nullable，未传回任意长字段。false/null/异常拒绝，只有 ErrNoRows 表示 absent。

连接 UTF8 检查及同事务私有 QueryRower 的 `pgx.QueryResultFormatsByOID{pgtype.JSONBOID: pgx.TextFormatCode}` 让现有 Get 的 `[]byte` 接收有界的相同 JSONB UTF8 text；该选项及真实 codec 消费已核，不要求修改 audit 包。已存在行依靠 append-only/UUID PK 不变性保护 guard→Get；absent 分支 **不调用 Get**。其他事务此后抢占 UUID，仅使本次 Append PK 冲突、回滚非零，不取 winner、不重试或开新事务。Append 前限序列化大小，Append 后 commit 前再核实际 text 长度/binding；回滚/commit unknown 不冒充成功。

最终 TE `:653`–`:665`、I-S105-15/16/17/20 覆盖超大 metadata/其他 text、absent→巨型 UUID 竞争、UTF8/text-format、未知与实际写入上限，保留全部 string carrier/int64 精度要求。该边界保护客户端传输、物化与解码；未承诺数据库 detoast/render 恒定 CPU/内存。未在本轮造库或测分配，所需真实 PG 载体已经规划，故在设计层关闭。

### R02-F001：r47 健康 idle 连接等待 CleanupDone 却未先触发 SDK Close

**finding 原始出处保持：** r47 design SHA256 **`E8F67BF4CAA1800B0F9D76D773A46F230BF6BCBE92B398BF7E0DC94CB84FC553`**，原位置 `design.md:2383`–`:2385`。成功 commit/replay 后连接健康 Release 成 idle，仅关闭 raw transport 不会完成 SDK CleanupDone；若先等 Done，再到 Pool.Close/BeforeClose 才触发 SDK Close，形成等待环，合法请求也会超时/失败。

Phase1 原始临时报告 SHA256 **`05EBB643B01D3424C71162BBD511AC7068F1DFABD340F15D3B115C7F09D07D9C`**；持久来源 `.spec/wip/identity-auth/evidence-s10.5/design-r02/phase1-result.json` SHA256 **`183DC0CFA92BDA285BDE3A1217F33621EB6CC90B5D0B202813FC980314112DE1`**。该阶段报告是在 r47 发现新 HIGH 的记录，不是六对象整体 PASS；本 R02 关闭不回写抹除该失败。

**r48 修复与真实 SDK 复核：** 当前 design SHA693092（完整值见索引），`:2381`–`:2405` 对健康连接先由共享 `closeMaintenanceConnection` 主动调用 `pgx.Conn.Close(cleanupCtx)`，再等 CleanupDone，最后同步 Pool.Close。root 与 BeforeClose 共用每 entry mutex/once/result/latch；BeforeClose 是共享兜底，不再是等待 Done 之后才会发生的唯一触发者。已 asyncClosing 分支不中途重入 SDK Close，只截断受登记 transport 并等待原 Done。真实 Conn.Close→PgConn.Close、健康 Release 不自动 Close、Pool 析构触发顺序均已核源码。

同时核了修复所依赖的构造/收束全路径：

- private opaque D 与 root/operator 共同 cleanup cutoff 不重置，Lookup/Dial/成功 SDK 对象均登记；quiescing 禁新 Acquire/网络阶段，登记与关闭竞态用同一 registry 锁，但锁内不做 I/O/Close/join。
- BeforeConnect 在 config 副本安装 Lookup/Dial/tracer。ConnConfig.Tracer 真实要求 QueryTracer；设计的同一私有对象另实现 ConnectTracer 与两个无日志 noop QueryTracer 方法，接口成立。pgx 真正采用 TraceConnectStart 返回的 ctx 包住包括 DNS/TLS/auth 的 connect，不误称 BeforeConnect 可返回替代 ctx。
- TraceConnectEnd 先于 pool.AfterConnect，成功 Conn 先登记为 constructor-owned；AfterConnect 发布或在 quiescing 时在构造调用栈有界 close-late。Lookup 位于 Dial/ConnectTimeout 前，因此原 D 不是只给 Dial timeout。puddle 构造不随 Acquire 返回而当然结束；设计未作该假设。
- MinConns/MinIdleConns=0、无业务 AfterRelease 异步钩子，禁新 Acquire 且 root 调用返回后，ConstructingConns/AcquiredConns 与 Lookup/Dial/tracer/AfterConnect stageRefs 共同形成 barrier；没有拿单次旧 Stat/空 socket 表证明全部构造完成。失败 attempt 没有成功 SDK 对象时只核 transport/构造完成，不伪造健康 CleanupDone。
- 活跃 SDK 操作结束后才统一 SDK Close；成功对象 Done、失败 transport、构造/阶段全部收束后再 Pool.Close。不使用 SDK 内部 2m/15s 当本任务默认，不遗弃关闭 goroutine；明确提交与 cleanup 失败仍分别分类。

最终 TE `:600`–`:615`、I-S105-15/22 及对应 PG/B 子例已要求正常 apply/replay 的 **Close-trigger→CleanupDone→Pool.Close** 正序、raw close 不代替 Done、root/BeforeClose 只一次关闭、asyncClosing 不重入、Lookup/迟到构造/失败无回调/统计与阶段 barrier、1s cleanup 和共同 cutoff。r48 机制及最终消费已实际复审，故关闭。未将静态接口充分性写成连接生命周期动态测试已通过。

### 历史报告保全

R01 正式报告 `.spec/wip/identity-auth/review-design-s10.5-r01.md` 保持 **CHANGES REQUIRED / 三项 OPEN 的历史原文**，SHA256 **`FAA1EAA42C2CF3AD8495611E23EF24BD731B5641D51574FE5713FE6D817EBA01`**；R01 失败证据包 `evidence-s10.5/design-r01/index.json` SHA256 **`706862D5947AC8BAFE11599950A2F26CA8FECE9BCB7AF348CD7A370E52DEE0EB`**。本报告是独立修复复审后的新结论，不覆盖旧报告、原严重度、发现修订或原作者归属。

## 4. 六对象合并消费与规划充分性

在上述修复之外，已核下列原安全合同仍一致，无新增 finding：

- 同一 binary/root、maintenance/server 互斥，模式选择在 server effects 前；维护路径所有 listener/Serve/worker 活动为零，root 不直接 UPDATE、不新增第二 classifier、不伪造 lease。无参数 server 与非 Windows server 回归边界保留。
- Windows 真实进程 owner SID、部署受控公钥 trust/allowlist/epoch 与独立 approver plain Ed25519 绑定 action/target/before/expiry/requestID；env 自称 role/proofref 不作认证。签名长度、canonical 原字节、presence/nullable、时效与撤销 fail closed；审批理由/evidence/ref/完整 before-after/receipt 与审计消费对应。
- pre-DSN typed opaque 授权产物先于 DB constructor，锁内及 commit 前复核；User-first、完整陈旧 before 并发拒绝；job+Audit 原子，operation/hold/terminal/credential/session 不改，保护窗不提前。当前维护事务不发送邮件；后续原 worker 取得真实 lease 并按当前策略执行。
- 只承诺 **one enqueue / current worker policy**，不承诺 one attempt；MaxRetries 2→5 反例在设计/Story/Case19 和 PG/B 子例一致。全局 Audit UUID 单次消费及跨 target 竞争、完整 string carrier（含 int64）精度、重放仍须当前授权/时效/撤销/完整匹配、commit unknown 保持 hold 均一致。
- 输出、exit code、敏感信息抑制、失败隔离、既有 shared audit 契约与 schema/worker 不改；保护窗与当前 Gate 的生产消费者没有发现新增冲突。

### 未来验证：25 Case、11 PG/B 子例与命令

Reviewer 实读最终 r22 §4.4.3，并核 **I-S105-01～25** 唯一有序且均 **NOT IMPLEMENTED / NOT RUN**。U/PG/B/D 分层、API/DTO、根因子矩阵和拒绝/成功对照已覆盖本轮四个 finding。三个顶层未来测试名保持：

- `TestPasswordResetMaintenanceRoot`
- `TestPasswordResetMaintenanceOperator`
- `TestPasswordResetMaintenancePostgresCLI`

未来五条命令与原 r21 逐项一致（见最终 test-plan `:1011`–`:1026`）：Root、Operator 普通单测；两包 race；integration PostgresCLI 普通及 race。它们是未来验收入口，本 Reviewer 未执行。零测试/skip 不能算通过；真实子 binary 的 race 证据须单独确认，不能只凭父测试进程带 race 标志。

额外 **11** 个真实 PG/B 计划子例为：`I-S105-06_half_no_eof`、`I-S105-06_complete_no_eof`、`I-S105-06_blocked_producer`、`I-S105-06_positive_available_cancel_before_read`、`I-S105-16_oversize_uuid_guard`、`I-S105-17_absent_guard_uuid_race`、`I-S105-22_shutdown_1s_socket_cleanup`、`I-S105-22_applied_idle_sdk_close`、`I-S105-22_replay_idle_sdk_close`、`I-S105-22_constructor_lookup_late`、`I-S105-19_one_enqueue_max_retries_2_to_5`。命令/名字、子矩阵和 Story 的要求已合并核对，不以用例标题存在替代未来运行证据。

### 七路径充分性与后续接力

静态设计范围现为以下 **七路径**，未来 root 与 identity 两位实施作者分别 4+3 所有权；属于设计方向，尚非 Tasks 已登记或 CODE 已完成：

1. `cmd/han-server/main.go`
2. `cmd/han-server/main_test.go`
3. `internal/identity/password_reset_operator.go`
4. `internal/identity/password_reset_operator_test.go`
5. `test/integration/identity/password_reset_maintenance_test.go`（新独立 PG carrier）
6. `cmd/han-server/maintenance_stdin_windows.go`（新 Windows 适配）
7. `cmd/han-server/maintenance_stdin_other.go`（新显式 !windows 适配）

在已核 SDK 接缝和行为下，没有发现必须扩出第八源码、新依赖、go.mod、迁移、store/worker/audit 包的证据。集成 carrier 的独立完整 migration/constraint fixture 与 retain-only 边界已规划；不据此进行自动 DROP/cleanup。后续实际实施若出现新必需范围，应由原责任方另行对齐，不由本报告预授权。

S3.4 的实际 launcher/ACL、固定部署、trust/DSN/key 注册与保管签发、人员接收、第二真实 SID、pipe/duplicate handle 唯一消费者、撤销排空及恢复 epoch 属于待 Planner/部署方协调的真实接力，未伪装成已登记依赖或验收。设计机制已经选定，不因没有生产 key 而误写“软件方案未选”；本地合成 trust/签名测试也不证明生产签发已批准、自然人在场或物理隔离完成。现有 S3.3/S3.5 的窄未验/豁免边界及 S3.6 状态保持，不重开无关历史事项。

## 5. 实际验证、限制与分对象结论

实际完成：只读文件/SDK/官方资料、原字节 SHA256、独立 diff、Lead 两处 copyedit 的字节逆验证、25 Case 与 11 子例/五命令核对。结束核验六对象仍等于固定 SHA；**185 个源码文件、其他 27 份基线知识均无漂移**，R01 与 Phase1 来源摘要保持。未运行任何项目检查命令：没有 Go 编译/单测/race/vet、PG、Han CLI、SMTP、部署/E2E；未读实际 DSN/环境/凭据/key，未作账户/GRANT、Git mutation、清理或再次委派。

保留的实际限制：stdin 方案依赖受保护 launcher 的唯一消费者前提，其真实部署尚未验证；审计 guard 限制客户端大内容读取，不能约束后台 JSONB detoast/render 的恒定开销；SDK 资源机制尚待实现及真实故障/竞态验证。旧六次普通 stdin probe 不是新 F001 证明；只读 local PG preflight（180006）不是 Han Case、DDL 或新设计 PG PASS。真实 ack-loss、第二真实 SID、生产签发、跨备份/新 epoch、即时撤销不在本轮验证结果内。

| 对象组 | 两轮独立复审结果 | 精确边界 |
|---|---|---|
| ADR 本体 / 导航 | **PASS**；F002 关闭 | r03 七 ENV 与单一公共协议及消费引用一致 |
| architecture / feature design / stories | **PASS**；R01-F001/F002/F003、r47 起源 R02-F001 均关闭 | 限索引冻结修订的 S10.5 增量、直接消费者及七路径可实施性 |
| test-plan | **PASS（规划）** | 最终 TE r22 + Lead 两处措辞修正；25 Case/子矩阵/11 PG/B 子例与设计对齐；均未实施/未运行 |
| 六对象合并 Gate | **PASS（Design / ADR / TestPlan Review）** | 无未关闭阻断项、新 finding 0；不外推 Planner/代码/E2E/部署/人的接纳 |

Reviewer 个人取证目录：`C:/Users/J00010385/AppData/Local/Temp/han-s105-reviewer-prep-d3b855d173/`。本轮最终 `r02-final-75c1a7a9/` 保存冻结副本、与 R01 的独立差异、`objects.json`、`review-evidence.json`；实际复用的既有读集为 `merged-ad39a14b/review-evidence.json`、`r02-phase1-5fbc8843/phase1-evidence.json`、`r48-recheck-9ae24f5e/recheck-evidence.json`。正式报告写入后另在个人 Temp 保存最终摘要/不变性核验，避免报告自包含哈希循环。

**交回状态：stopped。** 本 R02 交 Lead 作本次有界设计复审结果；不自行启动 Planner、实现、测试运行或后续 phase，不代替人的结果接收或下一动作授权。
