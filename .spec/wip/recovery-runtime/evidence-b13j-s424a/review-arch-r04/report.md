# identity-auth 设计 r64 架构侧独立复审（review-arch-r04）

## 0. 对象与基线校验

**结论先行：CHANGES REQUIRED。** 设计侧仍有 BLOCKER 1（F006）、HIGH 2（F004、F018）、MEDIUM 4、LOW 2；另有条件项 1（F017，属 Task-Planner，不计入开放计数）。所有 NOT RUN 仍按 NOT RUN 登记，未计入任何通过判定。

| 对象 | 实测 | 与派发锚点 |
|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | 1,005,979 B / 5,449 行 / `FB84EC1E0B078AC99B09A48988F00257B77CA93102298A7DAD5004E0B8871E94`；`:5`、`:38` 均为 `identity-auth-design-ir01-r64` | 一致 |
| r64 区块 | `:4025`–`:4055`；`:4023` 为 r63 结束声明；`:4057` 为 r62 标题；两条向量在 `:4072`／`:4078` | 一致 |
| 基线 `review-arch-r03/report.md` | 31,164 B / 180 行 / `2649600B2BD063300B847746598EA257F5895FAB9EF99C3298B6B1EFCBBC7D6C`；CHANGES REQUIRED（1 BLOCKER／2 HIGH／1 MEDIUM + F017 条件项） | 一致 |
| `.add/project/knowledge/tasks/tasks.md` | 898,536 B / 3,289 行 / `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`；`:3186` 仍为 `- [ ]` | 一致，未改 |
| 工作区 | `main` @ `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；`git status --porcelain` 295 条 | 原样保留 |

**作者报告不作为证据。** 本轮派发未给出作者报告锚点；仓库内 `design-revision-r04/report.md` 实测 28,805 B / 153 行 / `452603BB6CC65E156F9363A03B877CB92B66287C9D8B51DF1C1A38651870C51F`。该文件只用于定位作者主张；本报告全部判定来自 `design.md` 原文、ADR-0031 原文与真实代码。另注：`design-revision-r03/report.md` 为 17,335 B / `DD63583CADF85B657A118C8DBEAC2DDB0105DA68133D917A71B1FFBF1C4A8CAE`，是上一轮锚点，不可与本轮混用。

**行号位移核对。** r63 为 5,416 行，r64 为 5,449 行，差值 33；r64 区块 `:4025`–`:4055` 为 31 行，加 `:4024`、`:4056` 两个分隔空行恰为 33。即除新增区块外没有增删行；就地改动只表现为行内替换或行尾追加标记。

## 1. 方法与读取范围

**读取范围（全量或定向）。**

- `design.md`：`:3860`–`:4110` 全量（含 r60 平台矩阵尾、r61 历史声明、r62 起点、r63 全段 `:3925`–`:4023`、r64 全段 `:4025`–`:4055`）；定向读取 `:1602`、`:1607`、`:1619`、`:1671`、`:3603`、`:3616`、`:3884`、`:4072`、`:4078`、`:4090`、`:4118`、`:4122`、`:4128`、`:4184`、`:4186`、`:4188`、`:4190`、`:4263`、`:4315`、`:4408`。
- ADR-0031 `framework.md`：`:1111`–`:1180` 全文（r05 四项结论、六条不变量、九条约束、后续阶段前置节点）。
- 真实代码：`internal/recovery/runtime.go`（`:146`–`:155` 端口接口、`:205`–`:216` `RuntimeDependencies`、`:476`–`:492`、`:729`–`:737`、`:760`–`:775`、`:1018`–`:1028`、`:1416`–`:1430`、`:1488`、`:1525`–`:1600`、`:1647`–`:1672`）；`cmd/han-server/recovery_runtime.go`（`:427`–`:451`、`:522`、`:555`–`:567`、`:641`–`:650`）；`cmd/han-server/main.go`（`:618`–`:690`）；`cmd/han-server/recovery_exclusive.go`（`:15`–`:42`）。

**golden vector 复算方法（自写脚本，内存内执行，未落盘、未写仓库）。** 用 `[IO.File]::ReadAllLines(..., UTF8)` 取 `:4072`／`:4078` 的精确行字符串，按 UTF-8 编码为字节（不含换行）后算 SHA-256；同时用自写递归下降 JSON 解析器建有序字典树，逐层比较原键序与 `StringComparer.Ordinal` 升序得 `orderViolations`；再用显式转义序列化器按键的 Ordinal 升序输出 canonical 字节，与原行字节逐字节比较。真实输出（原样）：

```
LINE=4072 bytes=1115 sha256=de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab orderViolations=0 canonicalByteEqual=True canonLen=1115 parsedEnd=1115/1115 trailingWs=False
  topKeys=approval_ref,artifact_bytes,artifact_digest,backup_ref,deployment_id,installation_id,issued_at,issuer,record_type,reviewer,revocation,source_installation_id,source_schema_digest,supplier_principal_ids,target_digest,valid_from,valid_until
LINE=4078 bytes=2357 sha256=8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508 orderViolations=0 canonicalByteEqual=True canonLen=2357 parsedEnd=2357/2357 trailingWs=False
  topKeys=acl,approval_ref,backup_ref,deployment_id,elf,environment,executable_path,executable_sha256,executor,installation_id,issued_at,issuer,mode,mount,network,owner_gid,owner_uid,receipt,record_type,reviewer,revocation,security,st_dev,st_ino,supplier_principal_ids,target_digest,valid_from,valid_until
```

**旧摘要残留检查。** 全文只在 `:3928`（r63.1 的历史反例说明）出现 `3c7b1cff…` 与 `9df40264…`；r64 区块内 `sha256:` 字面量只有 `:4033` 的两条，且与上述实测逐字符相等。不存在把旧摘要当现行向量的残留。

**回归扫描。** 在 `:4025`–`:4056` 内扫描 `已通过`／`已隔离`／`PASS`／`已实现`／`已验证`／`已满足`／`已闭合`／`已解决`：命中项全部处于否定语境（`:4027`「不得写成 PASS」、`:4035`「不声称已闭合／已解决」、`:4053`「不等于 PASS」「不声称已解决」）。`:3927`–`:4023` 与 `:4057`–`:4230` 未见新增肯定断言。结构核对：`:4263` 起的 r61 正文中仅 `:4408` 一处出现 `r64.` 引用（见 NEW-R04-02）；`:4229` 之后无任何 `r63.` 引用。

## 2. 逐条处置总表

| Finding | 级别 | 原级别 | r64 落点 | 结论 |
|---|---|---|---|---|
| F001 批准摘要自证 | — | BLOCKER(r01) | `:4061`、`:4067`、`:3928`、`:4049` | 已闭合（维持），未回归 |
| F002 `sha256:` 前缀唯一 | — | BLOCKER(r01) | `:4065`、`:4033` | 已闭合（维持），未回归 |
| F003 tool 字段/消费者/映射 | — | BLOCKER(r01) | `:4090`、`:4043` | 已闭合（维持），未回归 |
| F004 与 r60 冲突 | HIGH | HIGH | `:3995`（就地改写）、`:3997`（行尾标记）、r64.1 `:4029` | **部分闭合** |
| F005 claim/verifier 自证 | — | BLOCKER(r01) | `:3945`–`:3968`、`:4037` P4/P5 | 已闭合（维持），未回归 |
| F006 lease 类型/唯一取得点/生命周期/写集 | BLOCKER | BLOCKER | r64.1 `:4029`、r64.12 `:4051` | **未闭合** |
| F007–F010 DSN／`!linux`／NOT RUN | — | HIGH(r01) | `:4045`、`:4053`、`:4190` | 已闭合（维持），未回归 |
| F011 撤销/时间重取 | — | HIGH(r01) | `:4037` R2/R4/R6、V2/V4/V6、P6 | 已闭合（维持），未回归 |
| F012–F016 算法/文法/签名/文件表 | — | MEDIUM(r01) | `:4049`、`:4051`、`:4063` | 已闭合（维持），未回归 |
| F017 写集缺口 | 条件项 | 条件项 | `:4019`、`:4051` | 仍属 Task-Planner 范围外，不计开放计数 |
| F018 回执验签/见证 | HIGH | HIGH | r64.2 `:4031`、r64.3 `:4033`、r64.5 `:4037` | **部分闭合** |
| F019/F020 nil 语义与退出码 | — | MEDIUM(r01) | `:4014`、`:4051` | 已闭合（维持），未回归 |
| NEW-R02-06/07/08 | — | — | `:3928`、`:4067`、`:4090` | 已闭合（r03），未回归 |
| NEW-R03-03 见证双语义与改名 | MEDIUM | MEDIUM | r64.3 `:4033` | **部分闭合** |
| NEW-R04-01 放宽登记不完整 | MEDIUM | 新增 | `:4027` vs `:4039`、`:4049`、`:4408` | 新增 |
| NEW-R04-02 r61 正文被追加标记 | MEDIUM | 新增 | `:4408` | 新增 |
| NEW-R04-03 `baseline_sequence` 可降低 | MEDIUM | 新增 | `:4033` | 新增 |
| NEW-R04-04 witness 独立性未显式要求 | LOW | 新增 | `:4033`、`:4035` | 新增 |
| NEW-R04-05 `anchor_ref` 一致清单不完整 | LOW | 新增 | `:4031` | 新增 |

## 3. 已闭合项证据（回归确认）

- **F001（已闭合，未回归）**：`design.md:4061` 把真实性移到批准文件之外的信任根，并由 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH` 的绝对规范路径与 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256` 双固定——摘要来自外部 pin，不由被验对象自证，故不自我循环。`:4067` 给出六条 canonical 约束（键序、转义、整数十进制字符串、排序数组清单、禁用 `null`／省略），`:3928` 已修正旧串的键序错误，r64.11 `:4049` 再把 `organizations`／`appointments` 补入排序数组清单。两条向量实测 `orderViolations=0` 且 `canonicalByteEqual=True`，即“规则→字节”唯一，未发现第二解。
- **F002（已闭合，未回归）**：`:4065` 统一为 `sha256:` 加 64 位小写 hex；裸 hex 只在环境变量边界接受并立即规范化；批准记录、`RestorePlan`、`RestoreReceipt`、信任根、回执一律拒绝裸 hex／大写 hex。r64 区块未出现旧表述，`:4033` 的两条摘要与实测一致。
- **F003（已闭合，未回归）**：`:4090` 明确 `tool.approval_digest`→`RestoreSourceClaim.ToolApprovalDigest`→与批准快照逐字节比较→每次重校验确认未变；`source.artifact_bytes` 的三方相等（批准记录／`RestorePlan.ArtifactBytes`／实际字节数）已补齐；`:4043` 补上旧字段去取代处置。r64 未改动这三处语义。
- **F005（已闭合，未回归）**：`:3945`–`:3968` 的 claim 形状、`source.OpenApproved` 唯一 no-follow 句柄、`RestoreSourceVerifier` 恰好一次均保持；`:4037` 把 P4（`OpenApproved`）、P5（`RestoreSourceVerifier`，早于 `Journal.Begin`）、P6（`Journal.Begin` 前重读批准文件精确字节）排进唯一调用图，可实施性较 r63 更强，没有回退成自证。
- **F007（已闭合，未回归）**：`:4045` 把九处清零点改成代码级缓冲名，并固定 `privateRestoreInput` 与 `d.dsn` 为 `[]byte`、`appendJSONString` 直写字节；`recovery_restore.go:71`／`:78`／`:79`／`:342`／`:351`／`:392` 的点名与现状可对上（`:79` 的 `json.Marshal` 路径被明确要求改造）。“线上形状不变”与 `[]byte` 不再互斥：`[]byte` 是进程内载体，wire 仍是 JSON 字符串。`:4122` 已删除长度与前 8 位摘要，错误只留分类与随机关联 ID。
- **F008／F009／F010（已闭合，未回归）**：`:4053` 第（7）项保留 Linux host／ACL／mount／loader／fd-exec／maps／network／witness 全部 NOT RUN，并写明 `//go:build linux` 的 skip 与无匹配测试的 exit 0 不等于 PASS；`:4190` 已标记 netns 判据不成立；`!linux` 分流与 Windows 子进程测试的分流口径未被 r64 改写。`NoNewPrivs`／`CapBnd` 的降级被写成“观测值、不声称隔离”，没有越权成“已隔离”。
- **F011（已闭合，未回归）**：`:4037` 的 R2／R4／R6 与 V2／V4／V6 三次 `revalidateApproval`，加 P6 在 `Journal.Begin` 前重读批准文件精确字节，覆盖“准备后→启动前→子进程返回后”三个 TOCTOU 窗口；漂移一律 `restore_unknown` 或 closed，`R6`／`V6` 之后不得当成功。
- **F019／F020（已闭合，未回归）**：`:4014` 仍是“唯一预检调用、退出码 2”，`ErrAdmission*` 与现有 `run` 的映射未被 r64 改写；r64 只新增 `ErrAdmissionIncomplete` 作为关闭失败分类，未改既有分支。
- **NEW-R02-06／07／08（已闭合，未回归）**：向量修正、`artifact_bytes` 映射与 `sequence_floor` 单一语义在 r63 已闭合；r64 只做改名（见 NEW-R03-03）与清单补充，未回退。

## 4. 仍开放的 BLOCKER

### F006 — BLOCKER — 未闭合（lease 所有权与取得点自相矛盾）

- **原 finding（r03）**：`:3997` 要求把 lease 作为“新的只读依赖”传给 `NewRuntime`，但未定类型、nil 检查、唯一取得点、生命周期与 `Close` 责任；r03 判定现有公开契约与四处 `Acquire`／两处 `Close` 必须同时改动，而 r63 未声明该契约变更。
- **r64 落点**：r64.1 `:4029`、r64.12 `:4051`；被认定的“非 restore 流”见 `:4029` 中点名的 `installationFlow.acquire`（`runtime.go:476`／`:486`）、`admissionRuntime`（`:870`／`:988`）与 **`recoveryFlow.acquire`（`:1554`／`:1562`）**。
- **判定：未闭合。** r64.1 确实补齐了类型（`recovery.ExclusiveLease`）、非可选参数、nil provider → `ErrAdmissionClosed`、注入 lease 不由 Runtime `Close`、关闭失败的错误分类。但以下三点使该接线仍不可实施：

1. **`recoveryFlow` 就是 restore 流，却被列为“非 restore 流”。** 代码核对：`Enter` 在 `runtime.go:1529` 构造 `&recoveryFlow{...}` 并在 `:1530` 调 `f.enter`；`Resume` 在 `:1546` 构造同一类型并在 `:1547` 调 `f.resume`；`enter` 在 `:1574` 先调 `f.acquire`，随后 `:1584` 调 `r.deps.Restore.Replace`；`resume` 在 `:1648` 调 `f.acquire`，随后 `:1666` 调 `r.deps.Restore.VerifyRestored`。即 `recoveryFlow.acquire` 是 restore 路径上唯一的取得点。r64.1 一方面说“r63.9.2 的『Runtime 不再调用 Acquire』只限定为 restore 相关流”，另一方面把唯一的 restore 流划到“非 restore 流、仍走 `deps.Exclusive`”。两者只能有一个成立：若按字面实现，restore 路径会第二次 `Acquire`；若承认 `recoveryFlow` 属 restore，则“只限定为 restore 相关流”变成空集断言，“不再 Acquire”失去内容。
2. **第二次 `Acquire` 不是无害重复。** `DeploymentExclusive` 是无状态的 `Acquire` 工厂（`runtime.go:146`–`:149`），真实适配器每次调用都新开一个句柄并重新加锁（`cmd/han-server/recovery_exclusive.go:20`–`:41`，`:30` 调 `e.acquire(ctx, e.path)`，`:34` 包装成新 lease）。装配阶段已持锁的前提下再调一次，要么阻塞，要么以 `ErrAdmissionClosed` 失败（`:31`–`:33`），不存在“幂等返回同一 lease”的路径。
3. **单一 `exclusive` 字段无法区分注入与自取，`Stop` 会关闭注入 lease。** `Runtime` 只有一个字段 `exclusive ExclusiveLease`（`runtime.go:768`）；`installationFlow.acquire`（`:488`）、`admissionRuntime`（`:988`）与 `recoveryFlow.acquire`（`:1564`）都写该字段；`Stop` 在 `:1419` 读出后在 `:1423`–`:1424` **无条件** `Close`。r64.1 要求“`closeObservationLease`（`:1022`–`:1023`）与 `:1424` 只处理 Runtime 内部取得的 lease，不得 `Close` 注入的 `restoreLease`”，但既未新增字段，也未给出可判据（borrow 标志、来源枚举等），因此在现行结构下不可判。另外 r64.1 说该参数“替代 `:768` 的 `exclusive ExclusiveLease` 字段在 restore 流中的取得职责”，而 `:768` 是三类流共用的唯一字段，替换职责无法只作用于其中一类。

- **附带的第二处不可实施：非可选参数与不需要 restore 的动作互斥。** `newRecoveryRootAssembly` 只在 `recoveryPlan.present` 时构造（`cmd/han-server/main.go:629`–`:634`），该分支同时覆盖 `install`／`enter`／`resume`／`verify`；`install` 走 `installationFlow.acquire`（`:486`）、`verify`／`resume`／`enter` 走 `recoveryFlow.acquire`（`:1562`／`:1648`）。若装配阶段对所有 `present` 动作都预取 lease，则四个动作都与内部 `Acquire` 冲突（同上第 2 点）；若只对 `actionRequiresRestore` 预取（`:4015` 的配置口径），则 `install` 拿不到 lease，而 r64.1 又规定参数非可选且 nil 不可用——只能构造失败或引入隐式默认，后者被 `:4027` 不变量（5）禁止。
- **与 r60 的关系**：`exclusive` provider 仍留在 `RuntimeDependencies`（`:647` 传 `anchors.Exclusive`），这一点与 r60 `:3616` 一致，没有问题；问题只在“装配预取 + restore 流内再取”的双重所有权。**因此不是 r60 登记问题，而是 r64.1 自身的接线不可实施。**
- **修复方向（不代作者改）**：把 `recoveryFlow.acquire` 明确划入 restore 流并改为读注入 lease（或把注入点下移到 `recoveryFlow` 而不改 `NewRuntime` 签名），同时为注入 lease 指定独立存储与不 `Close` 的可判据；`install`／`admission` 的 provider 路径保持原样；若坚持改 `NewRuntime` 签名，须在 r60 `:3603`「绑定 `recovery.NewRuntime` 依赖集合」处给出显式修订，而不是只用适用关系解释。

## 5. 仍开放的 HIGH／MEDIUM／LOW

### F004 — HIGH — 部分闭合

- **已补齐**：`:3995` 已就地改写为“`restoreLease` 是已登记 `exclusive` 端口一次 `Acquire` 的产出值，不是新端口或新依赖项，不进入 r60 的 11 端口表、不注入 `recoveryRootInputs`”，并声明 r60 原文不被修改；`:3997` 行尾加了 r64.1 取代标记；r64.1 `:4029` 又用一段专门说明 r60 `:3603` 的适用关系。
- **仍开放**：`:3997` 正文仍是“把该 lease 作为**新的只读依赖**传给 `recovery.NewRuntime`”，与 `:3995`／`:4029` 的“不是新依赖项”字面互斥；该句的行尾标记只取代了“取得点、参数、Close 责任与 nil 分类”，没有取代“新的只读依赖”这个定性。且 r60 `:3603` 原文不只约束 `recoveryRootInputs`，而是“绑定 `cmd/han-server` 的 `recoveryRootInputs` 与 `recovery.NewRuntime` 依赖集合……新增端口、替换来源或改变归属须先修订本节与对应 ADR”；r64 新增了一个 `NewRuntime` 参数，属于对该依赖集合的改变，却没有按 `:3603` 自身规则修订该节或 ADR，只用“端口产出值不适用”的解释来覆盖。解释可以成立，但与“须先修订本节”的原文规则并存，读者仍无法确定哪句是现行规范。
- **判定理由**：原冲突在“端口”层面已答复，在“`NewRuntime` 依赖集合”层面仍未给出优先关系；属文档级但阻断 Task-Planner 判断，维持 HIGH。

### F018 — HIGH — 部分闭合

- **已补齐**：`:4031` 的 8 键签名载荷（`anchor_ref`、`prev_record_sha256`、`receipt_sha256`、`result`、`sequence`、`verified_at`、`verifier_principal_id`、`verifier_key_id`，全 string）、JCS 字节、只允许 Ed25519、公钥唯一来源 `keys[verifier_key_id].public_key` 且 `keys.principal_id == verifier_principal_id`、`verified_at` 在 V10 一次性取样并逐字节传递、`anchor_ref` 在回执 `trust_anchor_ref`／policy／审计记录／外部见证 head 四处逐字节相等、验签失败不得推进见证 `sequence`——这些均可机核，且与 `:4037` 的 V1–V11 图一致（V10 唯一：追加审计、`fsync`、验证方签名、见证原子推进）。
- **仍开放**：`:3990`（r63.8）仍以未被取代的措辞给出**另一套** `VerifyRestored` 的 V7／V8／V9 细节：V7 = 重算并验签、V8 = 取 `sequence_floor` 基线并读见证、V9 = 算 `receipt_sha256`＋追加审计＋验证方签名＋推进。它与 `:4037` 的 V7 = 独立读取回执、V8 = 验证 producer 签名、V9 = 读见证 head 整体错位一格。r64.0 取代清单第（3）项只声明“r63.3 的 P1–P3／R1–R8／V1–V10 编号图”被替换，第（7）项只声明“验证方签名载荷由 r64.2 固定”，都没有声明 r63.8 的 V 编号失效；而 r64.5 自己写“编号唯一，不得增删、重排或复用”。因此“编号唯一”目前不成立。
- **判定理由**：载荷、公钥来源、`verified_at`、`anchor_ref` 四处一致性均已闭合；残留是两套 V 编号并存，属机械修正但会直接导致实现按错编号落点，维持 HIGH。

### NEW-R03-03 — MEDIUM — 部分闭合

- **已补齐**：`:4033` 把 `receipt_anchors` 的基线字段由 `sequence_floor` 改名为 `baseline_sequence`（十进制字符串、运行时只读），旧键出现在信任根即未知键 → `ErrAdmissionBinding`；`receipt_anchors` 固定 13 键；外部见证 head 固定 5 键（`anchor_ref, sequence, issued_at, head_sig_alg, head_sig`）并给出签名载荷与验签公钥来源；判决用“严格大于 `baseline_sequence` 与 `head.sequence`”把“初始基线”与“活水位”拆成两个变量，双语义已分离。两条向量不含这些字段，改名未破坏向量。
- **仍开放**：`:4012`（r63.9.4 建议写集表）仍写“外部见证客户端、`sequence_floor` 原子推进”，没有 r64 标记；`:3990` 又用 `sequence_floor` 同时表示信任根基线与见证 head 的当前值。r64.0 取代清单第（2）项只覆盖“r63.2 的 `sequence_floor`”，覆盖不到 r63.8／r63.9.4。于是现行文本里存在两套见证 head 模式：`:4033` 的 `sequence`＋`head_sig_alg`／`head_sig`，与 `:3990`／`:4012` 的 `sequence_floor`。
- **判定理由**：权威定义（`:4033` 明确“exact keyset 固定”）可判定，实现照 `:4033` 不会错；残留属取代标记缺口，维持 MEDIUM。

### F017 — 条件项（Task-Planner 范围外，不计开放计数）

- `:4019`、`:4051` 均明确“只作建议，不是任务卡；本轮不改 `tasks.md` 任何一行；写集更新前保持不可实施”。`tasks.md:3186` 仍为 `- [ ]`，哈希与派发一致。
- **判定**：设计侧未闭合项与 Task-Planner 写集缺口是两件事；`tasks.md` 未改是预期，不因此判设计未闭合。但**实施前必须由 Task-Planner 更新写集**（至少 `internal/recovery/runtime.go` 与 `cmd/han-server/recovery_runtime.go` 的职责描述），否则 F006 的修复无处落卡。

## 6. 新增风险

### NEW-R04-01 — MEDIUM — 放宽登记不完整（PATH 与信任根上限）

- **证据**：`:4027` 写“**放宽登记（唯一一项）**：`NoNewPrivs` 由 r62.7.2 的『拒绝条件』降为事后观测值”。但 `:4039`（r64.6）把 `PATH` 列入 `tool.environment` 允许键（要求每个目录为绝对规范路径并列入批准记录），而 r61.5 原文 `:4408` 是“禁止 `PATH`、`LD_PRELOAD`…”。相对 r61.5，这是放宽；`:4049`（r64.11）又把信任根大小上限从 `1..65536` 提到 `1..1048576` 字节，同属放宽。两者都不在“唯一一项”的登记内。
- **性质澄清**：`PATH` 本身与 ADR-0031 相容——`framework.md:1117` 与 `:1144` 的 closed 条件是“`PATH` 指向任一非批准目录”，并非一律禁止 `PATH`；故不是“未登记的第二 authority”，而是“登记声明与正文不一致”。语义变化实际发生在 r63.6 `:3976`，r64.6 只是重申并删除 `LD_LIBRARY_PATH` 的替代路径。
- **影响**：`r64.0` 的“唯一一项”是可被批准方直接引用的完整性声明；不完整会使后续读者漏审其余放宽。

### NEW-R04-02 — MEDIUM — r61 正文被追加取代标记，违反自身不变量

- **证据**：`:4408`（位于 r61.5，`:4395`–`:4411`）行尾新增“（r64.6 取代：允许键只剩 `LANG`／`LC_ALL`／`TZ`／`PATH`；任何 `LD_*` 含 `LD_LIBRARY_PATH` 一律 closed。）”。这是 r61 正文的字节改动。
- **冲突对象**：`:3927`（r63.0）写“r61 正文（本文件 r61 标题之后的全部内容）**不被修改**”；`:3999`（r63.9.3）标题写“r61 历史正文的取代清单（**不改 r61 字节**）”。r64.0 的取代清单里没有任何一条声明解除该不变量，`:3999` 的七条取代项也不含 r61.5 的 ENV／PATH 句。
- **影响**：不是语义静默改写（原句完整保留、只加标记），但使“r61 字节不变”这条可机核承诺失真；若后续以字节比对做追溯，会误判 r61 被动过。建议要么撤回该标记、改用 r64.0 的取代清单声明，要么显式修订 `:3927`／`:3999` 的不变量。

### NEW-R04-03 — MEDIUM — `baseline_sequence` 可被重钉降低，且未要求人的批准

- **证据**：`:4033` 只写“`baseline_sequence` 只由部署方 out-of-band 更新”，未要求“只增不减”，也未要求更新信任根（等价于重钉 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`）必须经人的明确批准并留证。ADR-0031 `framework.md:1147`（不变量 6）与 `:1166`（约束 8 第 6 条）要求“验签信任锚固定登记在批准记录内、**轮换须经人的明确批准**且不得由被验证回执自身或 root 自选”。
- **技术边界**：实际回滚仍被“严格大于 `head.sequence`”与见证不可用即 `ErrAdmissionUnsupported` 挡住，所以不是直接绕过；缺口在治理与证据：降低基线本身无判据、重钉无批准记录要求。
- **影响**：见证被替换或暂不可达时，基线成为唯一剩余水位，其可信度取决于重钉过程是否有人的批准证据。

### NEW-R04-04 — LOW — 见证方与 root／执行身份的独立性未显式要求

- **证据**：`:4033` 只要求 `witness_principal_id` 为已登记 principal、`kind=verifier`、appointment `role_name=witness`、且不等于 `verifier_principal_id`；`anchor_ref` 四处一致；未要求它与 root、执行身份或 producer 在组织／信任面上独立。对比 `:3937`（r63.4）对 `verifier_principal_id` 明确写了“不得等于 executor 或 issuer”。
- **说明**：本条不是“kind 与 role 自相矛盾”——`principals.kind` 枚举（`:4063`）本就没有 `witness`，`role_name` 是区分 `verifier` 与 `witness` 的唯一维度，二者组合是自洽的；缺的是独立性约束。

### NEW-R04-05 — LOW — `anchor_ref` 一致清单未显式列出信任根自身

- **证据**：`:4031` 列“回执 `trust_anchor_ref`、policy、审计记录、外部见证 head 四处必须逐字节相等”，未把 `receipt_anchors.anchor_ref`（被查找的那条锚记录本身）写进等式；`:4033` 的 head 段也只写“head 的 `anchor_ref` 必须与回执／policy／审计逐字节相等”。按上下文可推出“相等”是通过该锚记录的 `anchor_ref` 索引，但等式链少一环，机核实现需要自行补。

## 7. Verdict 与计数

**Verdict：CHANGES REQUIRED。**

| 级别 | 计数 | 条目 |
|---|---|---|
| BLOCKER | 1 | F006 |
| HIGH | 2 | F004、F018 |
| MEDIUM | 4 | NEW-R03-03、NEW-R04-01、NEW-R04-02、NEW-R04-03 |
| LOW | 2 | NEW-R04-04、NEW-R04-05 |
| 条件项（不计入） | 1 | F017（Task-Planner 写集） |

**判据说明**：PASS 需要每个 BLOCKER 真闭合且无未登记放宽。本轮 F006 未闭合（接线不可实施），NEW-R04-01 又证明放宽登记声明不完整，两项各自都足以阻止 PASS 类结论。NOT RUN 项（Linux host／ACL／mount／loader／fd-exec／maps／network／witness，以及真实信任根、密钥、任命、批准、见证、审计 sink、生产工具）全部未计入任何通过判定，`identity-auth-S4.24a` 仍不可关闭。

**已确认无回归的三项**：① 未新增第二 authority／fallback／隐式默认（`:4027` 不变量 5 有明文，正文未出现“本次忽略”类分支）；② 未把 NOT RUN 写成 PASS（`:4027`、`:4053`、`:4055` 三处否定表述齐全）；③ 未与 r60 `:3603`／`:3616` 的端口清单和唯一 production root 产生新的未声明冲突（残留的是 F004 的“依赖集合”层面）。

## 8. 未验证边界

- **JCS 实现**：向量复算用的是本机自写解析／序列化器，不是经审计的 RFC 8785 独立第二实现；`canonicalByteEqual=True` 只证明“本文规则下的序列化可还原原文”，不证明跨语言实现的等价性。未执行 JCS 官方测试向量（如 UTF-16 排序边角、浮点用例）；本次两条向量全为 ASCII 键与字符串值，未覆盖数字、代理对与转义边角。
- **密码学执行**：未执行 Ed25519 验签、base64url 解码或 `head_sig` 长度校验；`verified_at` 的精度与 `Z` 结尾只作文本核对。
- **Linux 侧**：未运行任何 Linux 目标测试；`/proc` 扫描、`flock`、`fsync`、mount 属性、ACL、fd-exec、`PT_INTERP`／`DT_NEEDED` 解析、`NoNewPrivs`／`CapBnd` 观测全部为 NOT RUN。
- **代码可实施性**：`revalidateApproval`、`RestoreSourceVerifier`、`recovery_restore_witness.go`、`recovery_restore_receipt.go` 等在设计中被点名的符号在当前代码中**尚不存在**，只能核对其与既有符号（`NewRuntime`、`RuntimeDependencies`、`recoveryFlow`、`DeploymentExclusive`、`newRecoveryRootAssembly`）的衔接是否自洽；不能证明实现后可通过编译或测试。
- **未构建／未测试**：本轮未执行 `go build`、`go test`、`go vet`，未改任何代码；F006 的“第二次 `Acquire` 会阻塞或失败”是基于 `restoreDeploymentExclusive.Acquire` 每次新开句柄的静态阅读，未用真实 `flock` 复现。
- **ADR 范围**：ADR-0031 只读 `:1111`–`:1180`，未读全文；ADR-0019／0027／0030／0032 未在本轮重读，其约束通过 design 的引用间接使用。
- **作者证据**：`design-revision-r04/report.md` 的作者自述与任何本地复算结果均未采信为证据；报告中的向量数值由本评审独立复算。

## 9. 只读声明

本轮为只读评审。未修改 `.add/**` 下任何文件（含 `design.md`、`tasks.md`、ADR），未修改 `cmd/**`、`internal/**` 或任何 `.go` 文件，未修改或覆盖 `review-arch-r01`–`r03`、`review-sec-*`、`design-revision-*` 的既有报告；未删除、移动、归档任何文件；未执行任何 Git 写操作（无 add／commit／branch／stash／checkout／restore／reset／clean）。工作区 `main` @ `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 的 295 条未提交改动原样保留。唯一写入为本报告文件。
> 脚注：本行之前的正文实测 29236 字节 / 174 行 / SHA256 `2CBFBE80C0EEC7DAD32B9355751A7EF19A20FB6B04E4D2DB507BDB7E3FE3FB36`；含本脚注的整文件摘要见交付消息。
