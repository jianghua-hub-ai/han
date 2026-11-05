# identity-auth design r63 架构侧复审（第三轮：r02 开放项 + 回归）

## 0. 对象、基线与复算

| 项目 | 值 |
|---|---|
| 评审对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| 字节 / 行数 | 976149 B / 5416 行 |
| 复算 SHA-256 | `3B8BEA8C35A12F75B36091969DB15B9590FBDD16B0429D59328ABB9FA3BC027C` |
| revision | `identity-auth-design-ir01-r63` |
| r63 增量 | `:3925`–`:4022`；golden vector 正文 `:4039`、`:4045`，摘要 `:4042`、`:4048` |
| 作者报告 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r03/report.md`；17335 B / 159 行 / `DD63583CADF85B657A118C8DBEAC2DDB0105DA68133D917A71B1FFBF1C4A8CAE` |
| 本轮基线 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r02/report.md`；46877 B / 444 行 / `FEB4C89F6A3210D6BC33DFFB29B1BF3A3131360AB9C9BE9696BAE1220736C30C`；CHANGES REQUIRED（BLOCKER 4／HIGH 3／MEDIUM 3） |
| 固定工作区 / 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；复核时 `git status --porcelain` 仍为 295 条 |
| 本轮范围 | 只判 r02 开放项（F001、F004、F005、F006、F011、F018、NEW-R02-06/07/08）与 r63 回归；不重判 r02 已闭合项的全部细节，只做回归抽查 |

开工复算：`design.md` 与作者报告的字节数、行数、SHA-256 均与派发值一致，作者报告 17335 B 一致，无偏差，故进入实质评审。

**结论：CHANGES REQUIRED。** 仍开放：BLOCKER 1（F006）／HIGH 2（F004、F018）／MEDIUM 1（NEW-R03-03）。已闭合：F001、F005、F011、NEW-R02-06、NEW-R02-07、NEW-R02-08。范围外条件 1 项（F017，属 Task-Planner，不计开放计数）。F006 未闭合不是措辞问题：r63 的接线要求现有 `internal/recovery` 公开契约与四处 `Acquire`／两处 `Close` 生命周期同时改动，而文本、依赖表与建议写集都没有覆盖，按现文本实施只能二选一（改契约而不声明，或产生第二份 ownership）。

## 1. 方法与两轮实际检查范围

第一轮（对象层）：完整读取 r63 `:3925`–`:4022`；读取 golden vector `:4039`、`:4045` 并独立重排复算；读取 r62 保留段 `:4024`–`:4230` 中与取代关系相关的 r62.1／r62.3／r62.5／r62.6.x／r62.7.x；核对 r61 结构（`:4230` 标题与其后全文无 `r63` 引用）。

第二轮（依据层）：沿真实消费者反查 `internal/recovery/runtime.go`、`cmd/han-server/recovery_runtime.go`、`cmd/han-server/main.go`、`cmd/han-server/recovery_exclusive.go` 与既有测试构造点；核对 r60 平台矩阵与唯一 production root 清单（`:3603`、`:3616`）；核对 ADR-0031 不变量 6（`framework.md:1147`）、约束 8（`:1160`–`:1169`）与启动门禁（`:1174`）；核对 `tasks.md:3186`–`:3218` 的写集与状态（只读）。

## 2. 逐条处置总表

| ID | r02 严重级 | r63 落点 | 本轮判定 |
|---|---|---|---|
| F001 | BLOCKER | `:3928`–`:3931`、`:4039`、`:4042`、`:4045`、`:4048` | 已闭合 |
| F004 | HIGH | `:3995`（与 `:3997` 对读） | 部分闭合（HIGH） |
| F005 | BLOCKER | `:3945`–`:3968`、`:3942` R3／V2 | 已闭合 |
| F006 | BLOCKER | `:3997`、`:4013`、`:1619` | 未闭合（BLOCKER） |
| F011 | HIGH | `:3942` R6／V5、`:3944` | 已闭合 |
| F018 | BLOCKER | `:3987`–`:3992`、`:3933`–`:3937` | 部分闭合（HIGH） |
| NEW-R02-06 | MEDIUM | `:3937` | 已闭合 |
| NEW-R02-07 | MEDIUM | `:3928`–`:3931` | 已闭合 |
| NEW-R02-08 | MEDIUM | `:3936` 第（7）项 | 已闭合 |
| NEW-R03-03 | — | `:3934`、`:3990` | 新增（MEDIUM） |
| F017 | HIGH | `:4018`；`tasks.md:3191`–`:3195` | 范围外条件（不计开放数；实施前必须由 Task-Planner 更新写集） |
| F009／SEC-F007 | HIGH／BLOCKER | `:4020` 第（3）项、`:4022` | 设计侧处置正确，闭合条件仍 NOT RUN（不计缺陷） |
| F002／F003／F007／F008／F010／F012／F013／F014／F015／F016／F019／F020 | — | 未改 | 回归抽查无新问题 |

## 3. 逐条详情

### F001 — BLOCKER — 已闭合

- **原 finding**：批准记录 digest 的 canonicalization 不唯一；golden vector 的键序违反同一节自定规则，摘要不可被独立 JCS 实现复算，等于自证。
- **r63 落点**：`:3928`（缺陷承认、旧摘要作废、新摘要）、`:3930`（七条 canonical 规则）、`:3931`（第三方复算步骤）、`:4039`／`:4045`（向量）、`:4042`／`:4048`（摘要句）。
- **证据（本轮独立复算）**：以自写递归 UTF-16／Ordinal 键排序 + 显式转义序列化重排两条向量，重排结果与文中字节**逐字节相等**，SHA-256 分别为 `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab`（1115 B）与 `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508`（2357 B），与 `:4042`／`:4048` 声明一致。全文检索旧完整摘要 0 次；截断旧摘要 `3c7b1cff`／`9df40264` 各 1 次，仅在 `:3928` 作历史反例，不构成 active 向量。
- **唯一性判断**：规则 1–7 固定了签名输入构造（删除 `approval_digest`／`signatures`）、键序、编码、转义、整数文法、数组排序白名单与空值形态，并禁止用 `encoding/json` 默认序列化；`source`／`tool` 两条记录不再存在第二条合法字节。`tool.receipt` 的 13 字段与占位值（`signature`／`issued_at`／`audit_record_ref` 空串、`sequence` 为 `"0"`）与 `:3987` 的 policy 说明一致，未出现“为了改摘要而改向量”的倒置。
- **信任根是否真 closed**：`:3933` 把信任根定义为批准文件之外的验签材料，`:3937` 要求 root 登记且不可由调用者自报，缺失／不可证明 → `ErrAdmissionIncomplete` 且保持 closed。批准 digest 不再是唯一证据链，摘要自证路径被切断。
- **残余（条件，不计 finding）**：本轮复算用的是自写序列化器，不是经过审计的 JCS 实现；本设计全部键为 ASCII，UTF-16 与 UTF-8 排序等价性、代理对与 U+2028／U+2029 分支未被本例覆盖。该点已在 §5 记为未验证边界，实施验收必须用独立 JCS 实现重放。

### F005 — BLOCKER — 已闭合

- **原 finding**：`RestoreSourceVerifier` 无法独立核验实际制品；无调用时机与 plan→claim 映射，且 claim 形状跨包不成立。
- **r63 落点**：`:3945`–`:3968`（导出 claim、删除私有快照、fd 所有权、offset 规则）、`:3942` R3／V2（调用时机）。
- **证据**：`:3948`–`:3965` 把 claim 固定为 `internal/recovery` 导出类型，字段含 `ArtifactDigest`、`ArtifactBytes int64`、`SourceApprovalDigest`、`ToolApprovalDigest`、`Artifact *os.File`；`:3946` 说明删除 `Approval`／`Tool` 两个私有快照，包边界矛盾消失。`:3968` 明确 verifier 只经 `Artifact` fd 读取（`ReadAt(header,0)`，或 `Read`／`Seek` 后立即复位并在 defer 再复位），`ArtifactPath` 只作诊断、不得重新打开；`execRestoreTool` 放入 `ExtraFiles` 前必须断言 offset=0。据此 verifier 核验的是**实际制品 fd**，不是路径或自报摘要；plan→claim 的映射由字段名逐项对应（backup/artifact/target/source installation/schema/两条批准），调用点固定在 `:3942` 的 R3 与 V2，失败分类在 `:3944`。fd 所有权唯一（driver 持有并关闭一次），不再有“谁 Close”的二义。
- **判定理由**：原先的“自证”来自 verifier 只能看到路径或摘要；现在绑定 fd 身份 + digest + offset 前置断言，且 claim 可被 `internal/recovery` 独立实现消费。

### F011 — HIGH — 已闭合

- **原 finding**：撤销与时间重取在子进程 in-flight 段未闭合（八步流程缺 execute 后、解码前的 revalidate）。
- **r63 落点**：`:3942`（R6／V5 固定为“子进程返回后、解码回执前再次 `revalidateApproval`”，并明确失败必须 `restore_unknown` 或 closed）、`:3944`（“任一检查点失败都在返回成功之前发生；不存在先返回成功再异步核验”）、`:3939`（每个检查点一次 `(wall, mono)` 样本，进程内 `mono` 高水位）。
- **证据**：R6／V5 的位置语义唯一，且 `:3941` 把重校验内容固定为 DSN 摘要 + 文件身份（`st_dev`／`st_ino`／`size`／`mode`／`uid`／`gid`／`mtime`）与批准／信任根漂移，避免了“只比长度／只比 inode”的弱化实现。in-flight 段在 execute 返回与解码之间关闭，剩余窗口不含“把旧批准当成功”的分支。
- **残余（非缺陷）**：跨进程／重启的时钟回拨不保证检测，已在 `:3939` 与 `:4020` 第（5）项自认，属登记风险而非隐藏缺陷。

### NEW-R02-06 — MEDIUM — 已闭合

- **原 finding**：验证方与 root 的独立性缺少可机核字段。
- **证据**：`:3937` 要求信任根登记至少一个 `principals.kind=root` 及其 `key_id`；producer／executor／issuer／reviewer／verifier 的 `principal_id`、`key_id`、`org_id` 均不得等于 root，且必须从信任根字段机核；`receipt_anchors` 的 `producer_principal_id`／`verifier_principal_id` 与 root、executor、issuer、reviewer 两两分离；`producer_key_id` 与 `verifier_key_id` 必须不同且都登记在 `keys`。任一不满足 → `ErrAdmissionBinding`，无 root 登记 → `ErrAdmissionIncomplete` 并 closed。

### NEW-R02-07 — MEDIUM — 已闭合

- **原 finding**：JCS 与 Go `encoding/json` 的转义差异未约束。
- **证据**：`:3930` 明确禁止 `encoding/json` 默认 `json.Marshal`（HTML 转义与 U+2028／U+2029 处理改变字节），要求显式键排序与显式转义编码器，并规定“实现键序与向量不一致必须拒绝实现，不得改向量迎合实现”。与 §F001 的复算结果一致。

### NEW-R02-08 — MEDIUM — 已闭合

- **原 finding**：`reviews.tool_sha256` 与 `tool.executable_sha256` 的映射未明说。
- **证据**：`:3936` 第（7）项要求 `reviews.tool_sha256` 与 `tool.executable_sha256` **逐字节相等**（同为 `sha256:` 小写形式），不一致 → `ErrAdmissionBinding`，并明确不得把该字段解释成 `tool.approval_digest` 或 `tool.receipt` 的其他摘要。`reviews` keyset（`:3934`）含 `tool_sha256`，消费者唯一。

### F004 — HIGH — 部分闭合

- **原 finding**：r61.3 允许新 verifier 进入 root 装配链，与 r60 `:3603`“未在本表内登记的依赖不得注入 production root”冲突，无优先句。
- **r63 落点**：`:3995`（优先句）。
- **已补齐**：`:3995` 用加粗优先句声明：r63 所指的是 `controlledRestoreDriver` 构造期内部依赖（`RestoreSourceVerifier`、批准快照、工具句柄、目标 DSN、独占 lease），不构成 r60 表格登记项，不注入 `recoveryRootInputs`、不加入 `RuntimeDependencies`；r60 原文继续适用于“所有端口与可独立配置／替换的依赖”，r60 原文不修改，r63 不是其例外；同时取代 r62.6.1 较弱的文本限定。就 r02 所问的“verifier／快照／工具句柄／DSN 是否被允许进入 root”而言，这一句给出可判定答案，方向正确。
- **未闭合的证据**：
  1. `:3995` 把“独占 lease”列入**不加入 `RuntimeDependencies`** 的内部依赖，但 `:3997` 要求“把该 lease 作为新的只读依赖传给 `recovery.NewRuntime`”。`recovery.NewRuntime` 的唯一依赖载体就是 `RuntimeDependencies`（`internal/recovery/runtime.go:205`–`:216`、`:729`–`:735`），因此两句互斥：要么 lease 进入 `RuntimeDependencies`（与 `:3995` 冲突），要么 `NewRuntime` 新增参数（即依赖集合变更，未声明）。
  2. r60 `:3603` 不只约束 `recoveryRootInputs`，原文是“绑定 `cmd/han-server` 的 `recoveryRootInputs` 与 `recovery.NewRuntime` 依赖集合……未在本表内登记的依赖不得注入 production root”，并规定“新增端口、替换来源或改变归属须先修订本节与对应 ADR”。`:3995` 的优先句只覆盖“端口／可独立配置替换的依赖”，未覆盖“派生 lease 进入 runtime 依赖集合”这一新情况，r63 也未提出修订 `:3603` 或 ADR-0032。
  3. r60 `:3616` 把 bundle 中的 `exclusive` 固定为 `recovery.DeploymentExclusive`，且 bundle 四端口须齐备；`recoveryRootInputs.exclusive` 的静态类型同为 provider（`cmd/han-server/recovery_runtime.go:445`），`:565`–`:567` 直接赋给 bundle。若装配体改为向 Runtime 传 lease，bundle／inputs 的类型与“四项齐备”语义都要改，r63 未声明。
- **影响**：r02 的原始冲突在“端口”层面已答复，但在“lease 是否属于 runtime 依赖集合”这一 r63 新引入的接线上仍无优先关系，且与 r63 自身 `:3995` 相矛盾。读者无法判断哪句是现行规范。
- **修复方向**：明确写出“lease 是已登记 `exclusive` 端口的派生句柄，不是独立依赖”，并二选一：(a) lease 只留在 driver（Runtime 继续持有 provider，唯一 `Acquire` 在 Runtime 内，driver 通过只读接口取 `Check` 能力）；或 (b) 明确变更 `RuntimeDependencies` 契约，同时按 `:3603` 自身规则修订该表与 ADR-0032，并在建议写集加入 `internal/recovery/runtime.go`。这属方向决策，由 Lead 选择，不在本报告代选。

### F006 — BLOCKER — 未闭合

- **原 finding**：构造签名与生产 `authority`／`ownership` 接线不成立；`Bind` 收到 provider 而非已取得 lease，类型不成立；若无解则实施者只能类型断言或自行 `Acquire`，产生第二 ownership。
- **r63 落点**：`:3997`（r63.9.2，取代 r62.6.3）、`:4013`（建议写集）、`:1619`（设计自身依赖表）。
- **已补齐**：`:3997` 正确区分 `anchors.Exclusive` 是 **provider** 而不是 lease，禁止把 provider 传给 `Bind`，禁止 `Bind` 做类型断言／包装，要求 `inputs.exclusive == nil` 时由既有 `recoveryClosedPort` 的 `Acquire` 失败并返回 `ErrAdmissionClosed`，且 `authority` 仍是同一个 `*recovery.Runtime`。方向与 r02 的修复建议一致。
- **未闭合的证据（按现有 HEAD 代码逐项核对）**：
  1. **依赖类型与 nil 检查未定**。`RuntimeDependencies.Exclusive` 仍是 `DeploymentExclusive`（`internal/recovery/runtime.go:211`），`NewRuntime` 的缺端口检查把它当端口（`:729`–`:735`）；设计 `:1619` 也仍写 `Exclusive DeploymentExclusive`。`:3997` 说“把 lease 作为新的只读依赖传给 `NewRuntime`”，但未说字段改名／改类型／新增字段，也未说 `NewRuntime` 的 nil 检查如何对待 lease。若替换类型，这是 `internal/recovery` 公开契约变更，现有测试世界都以 provider 形态构造（`cmd/han-server/recovery_restore_test.go:225`，其中 `:527` 的 `Acquire` 返回 lease；`cmd/han-server/recovery_production_chain_integration_test.go:906`、`:1333`、`:2455`），而建议写集 `:4000`–`:4017` **不含** `internal/recovery/runtime.go`。
  2. **“Runtime 不再 `Acquire`”与代码不符且无替代说明**。现有 `admissionRuntime` 在 `RunNormal`（`internal/recovery/runtime.go:870`）与 `recoveryFlow.acquire`（`:1562`，尚有 `:486` 同类调用）多处 `Acquire`；`Verify` 在无 live lease 时**自行**取得临时 observation lease 并在返回前关闭（`:980`、`:986`–`:1000`、`:1022`–`:1026`）。r63 未给出按 action（normal／enter／install／resume／verify）的 Acquire 矩阵，也未说明 `Verify` 的临时 lease 由谁提供、是否还允许存在。
  3. **生命周期与 `Close` 冲突**。`:3997` 要求 lease“活到进程退出”、Runtime“不 `Close`”；但 `Stop` 目前在 join 成功后 `Close` 该 lease（`:1418`–`:1425`），且 `Verify` 会关闭临时 lease（`:1022`）。若按 r63 改，部署锁释放时点、`Stop` 的 `cleanupErr` 语义、`RunNormal`／`Stop` 重入行为都变化；r63 未声明这一契约变更，也未列受影响的测试。
  4. **唯一 `Acquire` 的位置自相矛盾**。`:3997` 说在“装配阶段”且“唯一持有者是 `newRecoveryRootAssembly` 的调用方”，即 `cmd/han-server/main.go:629`–`:634` 的 `runComposed`；但建议写集 `:4013` 写 `recovery_runtime.go`“接收已 `Acquire` 的 lease 并传给 `NewRuntime`”，`:4014` 的 `main.go` 行只写“唯一预检调用、退出码 2”，没有 `Acquire`。另外 `:3997` 的“`Acquire` 在 driver 构造前失败”依赖 driver 构造点（预检内还是 assembly 内），r63 未固定。
  5. **closed 端口路径的错误分类未固定**。`:3997` 只说失败“原样返回 `ErrAdmissionClosed`”，未说明脏装配顺序下与 `ErrAdmissionBinding`（缺端口）／`ErrAdmissionUnsupported`（平台）的优先次序，也未说明与既有“未绑定 driver 一律 `ErrAdmissionClosed`”规则（`:4127` 历史段）如何合并。
  6. **visible impact 未登记**。若 lease 进入 `RuntimeDependencies`，r60 `:3616` 的 bundle 四端口齐备语义、`recoveryRootInputs.exclusive` 类型与 ADR-0032 的继承说明都要改；r63 未声明。
- **影响**：按现文本实施者无法在不改公开契约、不产生第二份 ownership、不破坏既有测试世界的前提下写出可编译接线。r63 只把“传给 `Bind` 的对象”从 provider 换成 lease，没有解决 lease 的来源、所有权、关闭与 Runtime 内部既有 `Acquire` 的并存问题，因此原 BLOCKER 的核心（同一 lease、唯一 ownership、可编译）仍未闭合。
- **修复方向**：在设计中固定：(a) `RuntimeDependencies` 中 lease 的字段名／类型，或改为 `NewRuntime` 新增显式参数；(b) 五类 action 的 Acquire／Check／Close 责任表；(c) `Stop` 是否释放部署锁（若不释放，说明进程退出释放即可，并给出 `Stop` 的成功语义）；(d) `Verify` 临时 lease 的存废；(e) 唯一 `Acquire` 的确切函数与在 driver 构造前后的顺序；(f) closed 端口失败的分类与优先级；(g) 写集与测试影响（含 `internal/recovery/runtime.go` 与上述测试文件）。在此之前不得判 F006 闭合。

### F018 — BLOCKER — 部分闭合

- **原 finding**：r61 的 `tool.receipt` 只有展示字段，未编码 ADR-0031 不变量 6（验签、信任锚、验证方独立、单调序列、独立读取、审计）。
- **r63 落点**：`:3987`–`:3992`（r63.8）、`:3933`–`:3937`（信任根 v2 与 root／verifier 分离）。
- **已补齐（对照 r02 六项残余逐条）**：
  1. 回执 policy 与真实回执拆分：`tool.receipt` 是 policy（13 字段，`signature`／`issued_at`／`sequence`／`audit_record_ref` 为占位），真实回执是 12 字段 strict v1，字段映射唯一且 `signature`／`issued_at`／`sequence`／`audit_record_ref` 绝不取自 policy（`:3987`）。
  2. 签名载荷外层 schema 固定为 `{"receipt": R, "attestation": A}`，`R` 删除 `signature`／`audit_record_ref` 后 10 字段，`A` 删除 `signature` 后 6 字段，`audit_record_ref` 不进入载荷且不是空串占位（`:3988`），r02 的“空串 vs 删除”二义消失。
  3. 回填与摘要：`receipt_sha256 = sha256(JCS(12 字段回执删除 audit_record_ref))`，回填不回写、不触发重算（`:3990`）。
  4. sequence 基准：本次 `sequence` 必须严格大于信任根初始基线与外部见证当前 head 两者，持久化责任方是外部见证（`:3990`）。
  5. 审计 sink：读／追加两个独立 fd，`O_WRONLY|O_APPEND|O_CLOEXEC|O_NOFOLLOW` 与 `O_RDONLY|O_CLOEXEC|O_NOFOLLOW`，root 属主、`0600`、执行身份不可写，`flock(LOCK_EX)` 覆盖“读尾—算 prev—追加—fsync”，单条一次 `Write`，短写／EINTR／fsync／父目录 fsync 失败一律不得成功（`:3992`）。
  6. 验证方独立：`:3937` 机核 verifier 与 producer／executor／issuer／reviewer／root 的 principal、key、org 分离；外部见证不存在／不可验签／无法推进 → `ErrAdmissionUnsupported` 并 closed（`:3992`）。与不变量 6（`framework.md:1147`）的“先验签、独立读取、信任锚固定、验证方独立、审计”逐项对应。
- **未闭合的证据（残余，均为可实施性缺口）**：
  1. **验证方签名载荷的编码规则未固定**。`:3990` 只枚举“对 `{receipt_sha256, sequence, verified_at, result, prev_record_sha256}` 签名”，未说明该对象是否按 `:3930` 的 JCS 规则序列化、是否恰为这 5 个键（无外层包装）、字段类型（`sequence` 十进制字符串？`verified_at` UTC 微秒？`result` 取 `:3944` 四值？）。`:3930` 的签名输入规则原文只覆盖“记录对象删除 `approval_digest` 与 `signatures`”，不能自动套用到此对象。独立验证方仍无法唯一重建待签字节。
  2. **验证方公钥来源未写**。`:3988` 只把 producer 验签公钥绑定到 `receipt_anchors.public_key`；`:3934` 的 `receipt_anchors` 只有一个 `public_key`，且 `:3937` 规定它等于 `producer_key_id` 对应公钥。V9 要求外部见证以 `receipt_anchors.verifier_key_id` 验证验证方签名（`:3990`），但没有说验证方公钥取自全局 `keys[verifier_key_id].public_key`。实施者可能误用 producer 公钥，导致验签恒失败，或被迫自选来源（形成隐式默认）。
  3. **`verified_at` 的取值点未绑定**。`:3939` 规定每个检查点取一次 `(wall, mono)` 样本；`:3990` 未说明审计记录的 `verified_at` 必须取 V9 的哪一次样本，也未说明与 `issued_at`／`reviewed_at` 的时间窗关系。序列推进与审计时间因此不唯一。
  4. **未签名字段的完整性未声明**。签名载荷只含上述 5 个字段，不含 `anchor_ref`、`verifier_principal_id`、`verifier_key_id`；`receipt_sha256` 绑定回执（回执内含 `trust_anchor_ref`／`trust_anchor_sha256`），`prev_record_sha256` 绑定链，但 `anchor_ref` 本身未被签。r63 应写明“审计记录 `anchor_ref` 必须等于回执 `trust_anchor_ref`，否则 `restore_unknown`”，否则审计记录被换锚后签名仍有效。
- **影响**：不变量 6 的骨架已编码，但“独立验证方能否重算同一字节、用哪把公钥、取哪个时间”仍有三个未定义分支；按 r02 的验收口径（独立实现可唯一复算／执行），F018 未达到闭合。
- **修复方向**：在 `:3990` 增补“验证方签名载荷固定为 `{...}` 的 5 键对象，按 `:3930` 的 JCS 规则序列化，`sequence` 为十进制字符串，`verified_at` 为 UTC 六位微秒 `Z` 结尾，`result` 取 `:3944` 四值之一”；写明验证方公钥 = `keys[verifier_key_id].public_key` 并纳入登记／轮换；把 `verified_at` 绑定到 V9 的样本；补 `anchor_ref` 一致性要求。

### NEW-R03-03 — MEDIUM — 新增（`sequence_floor` 双语义与见证 head 形状）

- **证据**：`:3934` 在每条 `receipt_anchors` 里定义 `sequence_floor`（信任根初始基线）；`:3990` 又用同名 `sequence_floor` 指“外部见证读取的当前 head”，并强调“运行时不回写信任根”。语义靠上下文区分，字段名相同；`:4011`／`:4012` 要新增回执与见证文件，但外部见证返回的 head 记录 keyset、签名算法、端点身份与鉴权、原子推进失败分类都未定义（只有“见证签名”与“推进失败 → `restore_unknown`”）。
- **影响**：实施者可能把见证 head 写回信任根（违反“不回写”），或把基线当 head 比较；见证 head 的 wire 无形状则 `:4012` 不可实施。
- **修复方向**：给两个概念不同名字（如信任根 `initial_sequence_floor`／见证 `head_sequence`），并固定 head 记录 keyset 与签名算法；若坚持把见证形态留作部署输入，则明确“无法给出该形状即 `ErrAdmissionUnsupported` 且 closed”，与 `:3992` 一致。

### F017 — 范围外条件（不计开放数）

- **原 finding**：r62 建议写集未落回任务卡，按现任务卡实施必越界。
- **现状**：`tasks.md:3191`–`:3195` 仍是 4 文件写集（`recovery_restore.go`、`recovery_runtime.go`、`main.go`、`recovery_restore_production_integration_test.go`），`:3213`／`:3217` 仍是 BLOCKED 且未勾选。r63 `:4018` 明确“该缺口属 Task-Planner 后续动作，本轮不改 `tasks.md`…写集更新前保持不可实施”。
- **判定**：属 Task-Planner 范围外事项，**不因此判设计未闭合**，也不计入本轮开放计数；但必须在实施前由 Task-Planner 更新写集。按本轮 F006／F018 结论，更新写集时至少要补 `internal/recovery/runtime.go`（lease 契约）与验证方／见证相关文件，并把 `main.go` 的 `Acquire` 责任写清；否则即使设计补齐，任务卡仍不可实施。

### F009／SEC-F007 — 闭合条件仍 NOT RUN（不计缺陷）

- **证据**：`:4020` 第（3）项列明 Linux 目标／CI 上的 host／ACL／mount／loader／fd-exec／maps／network／witness 实测均未运行，Windows 只覆盖解析与语义层，`//go:build linux` 的 skip 与无匹配测试 exit 0 都不等于 PASS；`:4022` 明确不构成 S4.24a 可关闭、不构成约束 8 满足。`tasks.md:3217` 记录的负向 (d)(e)(f) 在本机结构性不可达，与本设计自述一致。
- **判定**：诚实且不越权；不暗示可关闭 S4.24a。设计层无新增缺陷，但任何“r63 通过即卡可关闭”的推断都不成立。

## 4. 回归与范围完整性

1. **范围**：r63 只新增／修改 `:3925`–`:4022` 与两条向量及两行摘要；r62 正文自 `:4024` 起保留作历史，取代关系在 `:3927`（八项）与 `:3999`（r61 七项）显式列出；`:3919`／`:3921` 行号与内容未变；后段整体位移 +99（5317→5416）。未发现 r63 静默删除 r61/r62 句子。
2. **r61 完整性**：`:4230` 标题内容未变；`:4230` 之后 `r63` 引用计数为 0；`revision:` 行唯一切换为 `identity-auth-design-ir01-r63`。
3. **未被写成 PASS／未把 closed 变 open**：`:3925`、`:4022` 明确“不新增第二 authority、fallback 或隐式默认，不把 closed 改为 open，不把 NOT RUN 当作 PASS”；`:3997` 保持 `recoveryClosedPort` 路径；`:3983` 把 CapBnd／NNP／setns／seccomp 明确降为 unsupported；`:3992` 见证缺失 → closed。这些词只出现在否定语境（`:3983` 标题“unsupported，不是已实现”、`:4020`“不等于 PASS”），未发现越权使用（`:3973` 的“该测试在 Windows 语义层可真实执行”是对未来验收可行性的判断，`:4018` 仍列为测试建议，`:4020` 声明 NOT RUN，不作为证据；此为观察项，不计 finding）。
4. **与 r60 的冲突**：`verifier／快照／工具句柄／DSN` 已有优先句（`:3995`）；但 lease 进入 `recovery.NewRuntime` 依赖集合这一点仍与 `:3603`／`:3616` 无优先关系，并与 `:3995` 自相矛盾（计入 F004）。
5. **旧摘要残留**：完整旧摘要 0 次；截断形式仅 `:3928` 历史反例。新摘要分别在 `:3928`／`:4042`／`:4048` 出现，未发现第三套值。
6. **向量与 policy 一致性**：`:4045` 的 `tool.receipt` 恰为 13 键，占位值与 `:3987` 一致；`:4039`／`:4045` 顶层均不含 `approval_digest`／`signatures`（签名输入删除项），`source` 向量不含 `version`，与 `:3987` 的“policy 不是实际回执”不矛盾。
7. **未发现**：第二 authority、fallback、隐式默认、把 closed 改 open、把 NOT RUN 写成 PASS、把 unsupported 写成隔离已达成。

## 5. 未验证边界

- **JCS 独立性**：本轮复算为自写序列化器（`ConvertFrom-Json` + Ordinal 键排序 + 显式转义），不是经过审计的 JCS 库；未覆盖非 ASCII 键、代理对、U+2028／U+2029、浮点／负数拒绝分支；未用 Go 实现或第二个独立语言实现交叉验证。作者复算与 Lead 复算均不能替代第三方 JCS 一致性证据。
- **密码学**：未生成或验证任何 Ed25519 签名，未验证 base64url 无填充解码、密钥登记与轮换；信任根 v2 的 JSON 是否满足 `:3930` 全部规则未逐字节验证（只验证两条 golden vector）。
- **环境**：无 Linux 宿主、无真实 POSIX ACL／mount／loader／fd-exec／flock／fsync／外部见证／审计 sink／生产工具／真实信任根与任命记录；上述均为 NOT RUN。
- **代码**：只读静态核对 HEAD `6da8abb`；未运行 `go build`／`go vet`／`go test`／`gofmt`；F006 的结论基于现有公开契约与调用点，若实施阶段先改契约再实现，需按新 revision 重新核。
- **任务与 ADR**：`tasks.md` 只读核对 `:3186`–`:3218`；ADR-0031 只核对不变量 6、约束 8 与门禁段（`framework.md:1147`、`:1160`–`:1169`、`:1174`），未全文重读 ADR-0019／0027／0030／0032，r63 对这些 ADR 的转述只在被引用处抽查。
- **工作区**：只确认 `git status --porcelain` 仍为 295 条、`design.md` 哈希与派发值一致；未逐条审计其余未提交改动是否与未来实施路径重叠（不属本轮范围）。

## 6. 复审条件

1. **F006（阻断）**：补齐 lease 契约（字段／类型或 `NewRuntime` 新参数）、五类 action 的 `Acquire`／`Check`／`Close` 责任表、`Stop` 与 `Verify` 临时 lease 语义、唯一 `Acquire` 函数与顺序、closed 端口失败分类，并同步 `:1619`、`:3603`／`:3616`、ADR-0032 与写集／测试影响；否则维持未闭合。
2. **F004（阻断）**：消除 `:3995` 与 `:3997` 的互斥，并对“lease 是否属于 `recovery.NewRuntime` 依赖集合”给出 r60 优先关系；若选择变更依赖集合，须按 `:3603` 自身规则先修订该表与 ADR。
3. **F018（阻断）**：固定验证方签名载荷的 JCS／keyset／类型、验证方公钥来源与轮换、`verified_at` 取值点、`anchor_ref` 一致性。
4. **NEW-R03-03（建议）**：区分 `sequence_floor` 双语义并固定外部见证 head 形状，或声明其为部署输入且缺失即 closed。
5. **F017（Task-Planner）**：实施前必须由 Task-Planner 更新写集与验收命令；本轮 `tasks.md` 未改是预期，不据此判设计未闭合。
6. **回归**：任何一项修订后，须重跑两条 golden vector 的独立复算与 r61 结构核对（`:4230` 之后不得出现新修订引用）。

## 7. 只读声明

本报告为独立架构侧复审，只新建本文件；未修改 `design.md`、`tasks.md`、任何 ADR、任何代码或测试，未修改或覆盖 `review-arch-r01/`、`review-arch-r02/`、`review-sec-r01/`、`review-sec-r02/`、`design-revision-r01/`、`design-revision-r02/`、`design-revision-r03/report.md`；未执行任何 Git 写操作（add／commit／branch／stash／checkout／restore／reset／clean），未删除、移动或归档任何文件；工作区 295 条未提交改动原样保留。

作者报告中的 JCS golden vector 复算只证明作者本地脚本与文本一致，**不构成独立证据**；本报告的复算同样只是评审侧独立脚本，不构成第三方 JCS 一致性认证。