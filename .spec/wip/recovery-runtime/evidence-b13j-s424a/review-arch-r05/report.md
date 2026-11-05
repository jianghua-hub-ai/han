# ARCH r05 独立复审报告 — identity-auth design r65

日期：2026-11-02  
评审角色：add-reviewer（架构侧；非作者、非实现者）  
工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`  
评审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`（`identity-auth-design-ir01-r65`，`:5`、`:38`）  
上一轮架构报告：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r04/report.md`（29,426 B，`F2FB13FE3EF58AAF2EA5FCB64B32D7C3F15AEE5CBF99CF8F69986D520E98A6DB`，Verdict CHANGES REQUIRED）  
本轮方法：先查对象与基线，再完整读取 r04 的 finding，按 r65 正文逐条回核；随后沿真实消费者核对 `NewRuntime` 调用面、lease 读取点、restore driver、ADR 与 golden vector，并独立复算 r61 raw 段与作者备份差异。

**结论：Verdict: CHANGES REQUIRED**

## 0. 对象与实测基线

| 对象 | 本轮实测 |
|---|---|
| `design.md` | 1,045,111 B / 5,464 行 / SHA256 `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9` |
| `design.md` 修订 | `identity-auth-design-ir01-r65`（`:5`、`:38`） |
| `tasks.md` | 898,536 B / 3,289 行 / SHA256 `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`；`:3186` 的 S4.24a 仍为 `- [ ]` |
| `decisions/framework.md` | 272,588 B / 1,271 行 / SHA256 `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`；ADR-0031 为 `adr-0031-r09`（`:1082`），`:1084` 明示 r09 未经复评、决策接受级未达成 |
| 作者修订报告 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r05/report.md`，42,696 B / 239 行 / SHA256 `56FAD3F134487E3810286D4C3A183D2C29F2854355757250474D772CA2CF34EB` |
| 作者哈希清单 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r05/hashes.txt`，3,805 B / SHA256 `E5CDBBCFC6D1F7F9C921787F1F7CA607C27896E9B7BB6AB58ED02B7668655B05` |
| r65 区块 | `:4057`–`:4070`，15 行，对应 r65.0–r65.13 |
| golden vector | source `:4087`（1,115 B）、tool `:4093`（2,357 B） |
| r61 标题 | 现行 `:4278`；`:4247` 是 r62.9.2 表头，不是 r61 标题 |

### 0.1 作者记录的过时值（必须单独登记）

作者报告 §6.2 与 `hashes.txt` 记录 `design.md` 为 **1,045,042 B / SHA256 `D2A6B54B2153858DC6366995C4332A380D9DA0BEDE863BDDC80D51663A3D8BE2`**。该值不是本轮现行文件：

- 现行文件为 **1,045,111 B / SHA256 `87D70DA8…`**，比作者记录多 **+69 B**，哈希不同。
- 作者 §6.2 的 **“+15 行”只对行数成立**：r64 基线 5,449 行到现行 5,464 行确为 +15 行。
- 但现行文件相对 r64 基线（1,005,979 B）的字节净值为 **+39,132 B**，不是作者记录的 **+39,063 B**。
- 作者 §6.8 声称对 `:4057` 做了 **+69 B** 事实校正，改为“其外部见证与撤销判据由本轮同步”；该串在现行 design 中出现 **0 次**，现行 `:4057` 仍写“该版本由本轮之外的既有工作形成”“本轮不修改 ADR”。这与作者报告第五节“r09 同步”的陈述互斥。
- 将作者备份 `%TEMP%\design-r65-prefix.bak.md`（1,044,640 B / `2BBBDC8410BEE8681DA3F3B648693EADD6CDA1CEC8856B90222F943EDB72DE61`）与现行文件逐行比较，实际有 **13 行**发生内容变化、字节合计 **+471 B**、行数不变；不是作者所称“12 行”。其中 `:4063` 字节数相同但内容已变，作者漏记；`:4057` 相对备份为 **+257 B**，不是作者单笔描述的 +69 B。

该项独立登记为 **NEW-R05-02（MEDIUM）**。本报告所有设计结论只以现行 1,045,111 B 文件为准，不采信作者快照。

## 1. 方法与两轮实际检查范围

### 1.1 第一轮：对象结构与 r65 规范面

- 完整读取 r65 区块 `:4057`–`:4070`（15 行，r65.0–r65.13），逐行核对 F006、F004、F018 及 NEW-R03-03／NEW-R04-01～05 的落点。
- 读取 r64 与 r63 的上下文 `:3930`–`:4056`，核对 r65 所称“取代”“历史”“唯一编号”的边界，以及 r60 端口表 `:3601`–`:3610`。
- 读取 r61 历史段现行标题 `:4278` 至文件末，核对 `:4423` 原句、段内 `r64`／`r65` 标记和 raw 字节。
- 读取 ADR-0031 `:1082`–`:1190`，核对 8 字段载荷、五处 `anchor_ref`、`baseline_sequence` 单调性、见证独立性、撤销／TOCTOU 与 ADR 是否登记 `NewRuntime` 签名变化。

### 1.2 第二轮：逐 finding 回正文并与消费者对照

- 对 F006：读取 `internal/recovery/runtime.go` 的 `NewRuntime`、单一 `exclusive` 字段、`recoveryFlow.acquire`、`Enter`／`Resume`、`checkOwnership`、`readControlled`、`activate`、`beginObservation`、`replay`、`confirmSucceeded`、`Stop`；读取 `internal/recovery/admission.go` 的 `checkOwnership`／`checkExclusive`／`recheckProof`；读取 `cmd/han-server/recovery_runtime.go` 的 assembly、`NewRuntime` 调用与 `joinWork`；读取 `cmd/han-server/recovery_restore.go` 的 `controlledRestoreDriver` 与 ownership 字段。
- 对 F004：逐字核对 r60 `:3603` 是否被修订，以及 r65.1 `:4058` 所称“行号声明”是否等于对 r60 本节的修订；同时核对 ADR-0031 全文是否出现 `NewRuntime`。
- 对 F018：核对 `:3942` 与 `:3990` 是否明确标为历史、`:4062` 是否给出唯一 V 编号、`:4059` 是否为 8 字段载荷。
- 对 NEW-R03-03：核对 `:4012`、`:4060`、`:4033` 以及 `sequence_floor` 的残余出现位置。
- 对 NEW-R04-01：核对 `:4057` 的 3 项放宽 + 1 项容量变更，以及 `:4063`、`:4064`、`:4070` 的复述。
- 对 NEW-R04-02：以 raw 字节复算 r61 段；核对 `:4423` 是否恢复原句；统计 r61 段内 `r64`／`r65` 标记。
- 对 NEW-R04-03／04／05：分别核对 `:4060` 的单调性、独立性和 `:4062` 的五处等式链。
- 真实消费者与调用面：`rg` 统计 `NewRuntime(`（39 处、17 个文件）、`recovery.NewRuntime(` 的生产调用点；核对 `driver.Bind` 是否存在；核对生产 `controlledRestoreDriver.ownership` 是否有赋值。
- 独立复算：Python 3.12 按 `:4082` 的 JCS 规则解析 `:4087`／`:4093` 原始字节，重排键、紧凑编码并计算 SHA-256；PowerShell 5.1 按 raw 字节复算 `:4247`、`:4263`、`:4278` 起始的段。

### 1.3 检查限制

本轮是设计文本与静态代码事实复审；**没有运行 Go 构建、测试或 vet**。Linux host、ACL、mount、loader、fd-exec、maps、network、witness 的真实行为均为 **NOT RUN**；真实 Ed25519 签名、HSM、审计 sink、人的批准和任命也未验证。Windows 本机只覆盖解析、静态语义与字节复算，不能外推为 Linux 运行结论。

## 2. 结论与计数

| 项目 | 数量 | 明细 |
|---|---:|---|
| r04 开放项（不含 F017 条件项） | 9 | 7 PASS；2 CHANGES REQUIRED |
| 阻断项 | 2 | F006（BLOCKER）、F004（HIGH） |
| F017 条件项 | 1 | 仍开放，不计入阻断计数 |
| 新增独立开放 finding | 4 | NEW-R05-02（MEDIUM）、NEW-R05-04（MEDIUM）、NEW-R05-03（LOW）、NEW-R05-05（LOW） |
| 并入既有项的 finding | 1 | NEW-R05-01（HIGH）并入 F006，不重复计数 |

**Verdict 依据：** r65 确实修复了 lease 主体结构、8 字段载荷与 V7–V11 编号的多处落点，也补齐了放宽登记、单调性、独立性和五处等式链；但 F006 仍未给出“按动作选择 lease”的统一访问器，F004 仍未按 r60 `:3603` 的原文要求修订依赖集合，两项分别阻断实现接线与 Task-Planner 判断，因此不能判 PASS。

## 3. r04 开放项逐条判定

### 3.1 F006（BLOCKER）— CHANGES REQUIRED — lease 所有权与取得点仍未真正闭合

- **r04 原问题：** `recoveryFlow.acquire` 属于 restore 流，却被旧文本划为“非 restore 流”；装配预取与 restore 流内再次 `Acquire` 冲突；单一 `exclusive` 字段无法区分注入 lease 与 Runtime 自取 lease，`Stop` 可能关闭注入 lease。
- **r65 已修部分：**
  - `:4058` 明确 `restoreLease` 是 `exclusive` 端口一次 `Acquire` 的产出值，非可选参数传入 `NewRuntime`。
  - `:4068` 明确 `recoveryFlow.acquire` 属 restore 流，要求删除 `deps.Exclusive.Acquire` 与 `f.r.exclusive = l`，改为读 `injectedExclusive`。
  - `:4069` 要求 `ownedExclusive` 由 provider 取得、只由 Runtime 关闭；`injectedExclusive` 构造期赋值且永不关闭。
  - `:4070`(8) 仍登记 `NewRuntime` 签名波及调用点与测试替身，未把“已实现”写成事实。
- **未修部分（阻断）：**
  1. r65.11 只点名 `f.begin` 与 `f.refreshAfter` 等读取点“一律改读 `injectedExclusive`”，但 `f.refreshAfter` 在 `runtime.go:1734` 调用 `f.r.readControlled`，后者在 `admission.go:616`–`:648` 先调用 `checkOwnership`，而 `checkOwnership` 在 `runtime.go:1160`–`:1176` 读取的是单一 `r.exclusive`。`f.phase` 在 `runtime.go:1773`–`:1775` 又先调用 `f.refresh`；因此 `enter` 的 `f.enter`（`:1573`–`:1580`）和 `resume` 的 `f.acquire` 之后都会在 `checkOwnership` 命中旧字段。
  2. `CheckPhase`（`runtime.go:1192`／`:1196`）、`ReadPermitBinding`（`:1241`／`:1245`）、`activate`（`:917`／`:919`）、`beginObservation`（`:948`／`:966`）、`replay`→`activate`（`:1943`）、`confirmSucceeded`（`:1968`）、`admission.go:648` 仍读取单一 `r.exclusive`。这些不是 `f.begin`／`f.refreshAfter` 的子路径，不能由 r65.11 的点名覆盖。
  3. 若实施者按字面把 restore 流的所有读取点改成 `injectedExclusive`，则 `install`／`normal`／`verify` 这些 `injectedExclusive == nil` 的路径会在 `checkOwnership` 或其他 accessor 处失败；若保留 `ownedExclusive`，restore 路径又拿不到 lease。当前文本没有“按当前动作选择 lease”的统一契约。
  4. `Stop`（`runtime.go:1419`–`:1424`）现行无差别读取并 `Close` 单一字段；`closeObservationLease`（`:1022`–`:1023`）也需按来源区分。r65 只说“只关 owned”，但没有给出所有调用点如何取得 owned 的完整规则。
- **触发场景：** 生产 restore 动作 `enter` 或 `resume` 在装配阶段取得 `restoreLease`，构造 `Runtime` 后 `ownedExclusive == nil`；`enter` 进入 `f.phase` → `f.refresh` → `readControlled` → `checkOwnership`，因 lease 缺失返回 `ErrAdmissionBinding`，无法到达 `Restore.Replace`。若开发者为了让 restore 通过而把所有读取点改为 `injectedExclusive`，则 `install`／`normal`／`verify` 在 `injectedExclusive == nil` 时反而失败。
- **影响：** F006 的“唯一 lease 所有权”仍无法按文本实现；实现者只能自行发明动作判别和访问器，存在第二 ownership 或关闭注入 lease 的风险。属于 BLOCKER，阻断 S4.24a 的实现接线判断。
- **最小修复方向：** 在 `Runtime` 层面明确一个统一 lease 选择契约（例如按当前 flow/action 选择 `ownedExclusive` 或 `injectedExclusive`，或提供两个有明确调用约束的访问器），并逐项列出所有 lease 读取点与关闭点：`checkOwnership`、`CheckPhase`、`ReadPermitBinding`、`readControlled`／`recheckProof`、`activate`、`beginObservation`、`replay`／`confirmSucceeded`、`f.begin`、`f.refreshAfter`、`Stop`／`drain`、`closeObservationLease`。同时把该枚举写入 r65.11／r65.12 和 Task-Planner 写集。不能只靠“f.begin 与 f.refreshAfter 等”概括。

### 3.2 F004（HIGH）— CHANGES REQUIRED — `NewRuntime` 签名变更未满足 r60 的先行修订要求

- **r04 原问题：** r60 `:3603` 把 `recovery.NewRuntime` 的依赖集合纳入唯一 production root 清单，并要求“新增端口、替换来源或改变归属须先修订本节与对应 ADR”；r64 新增了 `NewRuntime` 参数，只用解释覆盖，没有修订本节或 ADR。
- **r65 已修部分：**
  - `:3995` 明确 `restoreLease` 是端口产出值，不进入 11 端口表、不注入 `recoveryRootInputs`。
  - `:3997` 行尾以 r65.1／r65.11 标记取代旧的“新的只读依赖”表述。
  - `:4058` 给出完整签名 `func NewRuntime(ctx context.Context, d RuntimeDependencies, restoreLease ExclusiveLease) (Runtime, error)`，并说明非可选、构造期赋值、不得改为可选参数／`Bind*`／全局变量／结构体字段注入。
  - 签名变更本身的方向可接受：lease 作为端口产出值由装配方持有，避免 Runtime 自取第二份 ownership；非可选参数也比可选参数或无锁默认更符合 fail-closed。
- **未修部分（HIGH）：**
  - r60 `:3603` 原文**一字节未改**，仍写“绑定 `cmd/han-server` 的 `recoveryRootInputs` 与 `recovery.NewRuntime` 依赖集合”“新增端口、替换来源或改变归属须先修订本节与对应 ADR”。新增一个**必需的构造参数**至少改变了 `NewRuntime` 的调用/依赖表面；` :4058` 声称“由本条的**行号声明**完成适用关系登记”，但这是对 r65.1 的声明，不是对 r60 本节的修订，也没有给出“原文继续有效”与“签名已变化”何者优先的规范句。
  - ADR-0031 全文 `NewRuntime` 出现 **0 次**，没有登记该签名变更；`:4057` 还写“本轮不修改 ADR”“该版本由本轮之外的既有工作形成”，与 `:4058` 试图登记新的构造契约相互矛盾。
  - 读者无法判定 r60 的“须先修订本节”是否已被满足；Task-Planner 也无法据此确认写集边界。
- **触发场景：** 实施者按 r60 `:3603` 检查依赖集合时，看到“改变 `NewRuntime` 依赖集合须先修订本节”，而现行 `:3603` 没有新增 `restoreLease` 条款；同时 ADR-0031 没有 `NewRuntime`，无法从决策层确认该参数属于“端口产出值”而非新增依赖。
- **影响：** 文档级 HIGH；阻断实现接线与任务拆分，且会让后续审查对“是否新增了未登记依赖”产生分歧。
- **最小修复方向：** 直接修订 r60 `:3603` 的规范句，明确“`restoreLease` 是已登记 `exclusive` 端口的一次 `Acquire` 产出值，不是端口／可替换依赖，不进入 11 端口表；构造函数新增该非可选参数不改变端口集合”。同时把该结论登记到 ADR-0031（或明确指向一个已接受的相关决策），或至少补上 ADR 的显式交叉引用与决策状态。仅靠 `:4058` 的行号声明不足以满足 `:3603` 的原文要求。

### 3.3 F018（HIGH）— PASS（附 LOW 残文）

- **已闭合的语义：**
  - `:4062` 固定 `V7` 独立读取、`V8` 验证 producer 签名、`V9` 读取见证 head 并比较 `max(baseline_sequence, head.sequence)`、`V10` 审计／`fsync`／验证方签名／原子推进、`V11` 返回副本。
  - `:4059` 固定验证方签名载荷恰为 8 个 string 字段，明确 `audit_record_ref` 与 `verifier_signature` 不进入载荷。
  - `:3990` 句首明确“旧文字仅作历史，已由 r65.5 整体取代、不再构成定义；现行 V1–V11、8 字段载荷与 head 形状只以 r65.5／r65.2 为准，V 编号全文唯一”。
  - `:3942` 行尾亦标明旧 P/R/V 编号仅作历史，不再构成定义。
  - `:4037` 的 r64.5 图与 `:4062` 的 V7–V11 一致；`V10` 仍唯一承担审计、验证方签名和见证推进。
- **判定口径：** 按“历史段已明确非定义、权威定义唯一”的语义口径，F018 已闭合，判 PASS。若 Lead 的验收口径要求“V7/V8/V9 字样在全文物理唯一”，则 `:3942`／`:3990` 仍保留旧编号，应降为 CHANGES；本报告不把该风格性物理唯一要求当作阻断，但将其登记为 **NEW-R05-03（LOW）**。
- **触发场景：** 实现者若只搜 `V7` 而不读历史标记，可能误用旧映射；当前文本已用“仅作历史”“不再构成定义”阻止该读法，但检索工具不会自动理解语义。
- **影响：** 语义上无阻断；可读性和机械检索仍有误读风险。
- **最小修复方向：** 删除 `:3990`／`:3942` 中的旧 `V7`／`V8`／`V9` 字样，改为“旧步骤一／二／三”；并把 `:3990` 的“以 r64.5／r64.2 为准”统一改为“以 r65.5／r65.2 为准”。不需要改现行 V7–V11 定义。

### 3.4 NEW-R03-03（MEDIUM）— PASS

- **证据：** `:4012` 已改为“`baseline_sequence` 基线与见证 head `sequence` 的原子推进”，并带 r65.3 标记；`:4060` 的 14 项 keyset 使用 `baseline_sequence`；`:4033` 的历史 r64.3 也已明确旧键 `sequence_floor` 出现即视为未知键并拒绝。剩余 `sequence_floor` 只出现在历史／改名叙述或已标历史段（约 `:3934`、`:3942`、`:4027`、`:4033`、`:4049`、`:4057`、`:4078`、`:4082`、`:4143`、`:4330`）。
- **触发场景：** 信任根若仍使用旧键 `sequence_floor`，按 `:4033` 应作为未知键返回 `ErrAdmissionBinding`；现行基线比较使用 `baseline_sequence`，不再把“初始基线”和“见证 head 当前值”混为一谈。
- **影响：** 无阻断；历史示例仍可能造成检索噪音，但已有明确的取代标记。
- **最小修复方向：** 无需新增设计条款；若追求机械整洁，可在历史示例旁统一加“仅历史”短标记。

### 3.5 NEW-R04-01（MEDIUM）— PASS

- **证据：** `:4057` 逐项登记三项放宽 `NoNewPrivs`／`CapBnd`／`PATH`，每项均含放宽内容、理由、影响、抵消措施和残余；另登记一项仅容量变更（信任根上限 `65536` → `1048576`，`:4049`）。`:4064` 明确 `CapInh=CapPrm=CapEff=CapAmb=0` 等仍是拒绝条件，`CapBnd`／`NoNewPrivs` 只作观测值；`:4063` 明确 `PATH` 的绝对规范路径与批准记录约束；`:4070`(4)(5) 复述残余与容量项；`:4027` 行尾已加 r65.0／r65.13 取代标记。
- **触发场景：** 若实现把 `NoNewPrivs=1` 或 `CapBnd=0` 继续当准入条件，会与 `:4064` 的“观测值”冲突；若允许相对 `PATH` 或未登记目录，`ErrAdmissionClosed` 应触发。
- **影响：** 设计侧的放宽登记已完整；真实隔离能力仍未获得，`NoNewPrivs`／`CapBnd` 的安全残余仍在 `:4070` 明示。
- **最小修复方向：** 无设计修复；实施时必须把三项放宽与抵消措施作为测试判据，且不能声称已获得 no-new-privs 或 bounding-set 隔离。

### 3.6 NEW-R04-02（MEDIUM）— PASS（历史哈希标签另见 NEW-R05-05）

- **证据：** r61 标题现行在 `:4278`；`:4423` 行尾已恢复为“子进程实际 `environ` 与白名单逐字节不等即 closed。”，不再带 r64.6 就地标记；从 `:4278` 到文件末的 `r64`／`r65` 出现次数均为 **0**。`:4063` 说明 r61 段及其前置 r62 尾段必须保持原字节，并明确 `:4423` 已恢复。
- **触发场景：** 若实施者或后续修订在 r61 段写入 `r64`／`r65` 标记，按 `:4063` 即应视为恢复失败；若 `:4423` 行尾仍残留旧标记，r61 原字节不成立。
- **影响：** 设计侧已恢复；作者报告将 `:4247` 标成 r61 段起点是标签错误，不影响 r61 正文恢复，但会影响旧段哈希复算，另记 LOW。
- **最小修复方向：** 保持 r61 原文不动；作者报告应把“r61 段”与“r62 尾段”分开标注。

### 3.7 NEW-R04-03（MEDIUM）— PASS

- **证据：** `:4060` 明确 `baseline_sequence` 只允许单调不减；人工 out-of-band 重钉必须新值 ≥ 旧值、由人的明确批准并留审计记录；降低尝试或无批准证据 → `ErrAdmissionBinding`（启动前）／`restore_unknown`（已启动）；本次回执 `sequence` 必须严格大于 `max(baseline_sequence, head.sequence)`。ADR-0031 `:1147` 和 `:1166` 同步了单调性与人的批准要求。
- **触发场景：** 部署方试图把 `baseline_sequence` 从较大值改小，或没有可核批准证据即重钉；按本条应拒绝。
- **影响：** 设计侧闭合；真实批准主体和审计证据仍为 NOT RUN。
- **最小修复方向：** 无设计修复；实施时需把“降低即拒绝”和“无批准证据即拒绝”做成负向测试。

### 3.8 NEW-R04-04（LOW）— PASS

- **证据：** `:4060` 要求 `witness_principal_id` 的 `org_id` 解析到 `organizations.kind = external_verifier`；`witness_principal_id`／`witness_key_id` 的 `org_id` 不得等于 root／executor／producer／issuer／reviewer，`witness_key_id` 不得等于上述任一 `key_id`，任一不满足 → `ErrAdmissionBinding`。ADR-0031 `:1147` 同步了见证方与 root／执行身份的独立性要求。
- **触发场景：** 见证方与 root／executor 共用组织或密钥，或以别名伪装独立；按本条应拒绝。
- **影响：** 设计侧已显式化；同一自然人使用不同 `person_ref` 的残余仍不可检测，`:4070`(1) 已登记。
- **最小修复方向：** 无设计修复；部署时需保留不同管理域这一前置，且不能被代码判定替代。

### 3.9 NEW-R04-05（LOW）— PASS

- **证据：** `:4062` 将 `anchor_ref` 等式链固定为五处：`receipt_anchors.anchor_ref` 自身、批准记录（policy）中的 `anchor_ref`（若存在）、回执 `trust_anchor_ref`、审计记录 `anchor_ref`、见证 head `anchor_ref`；任一不等 → `ErrAdmissionBinding`（启动前）／`restore_unknown`（已启动）。`:4031` 已标为五处取代。
- **触发场景：** 任一参与方写成别名或不同 anchor；按本条应拒绝。
- **影响：** 设计侧闭合；未验证真实回执与见证 head 的字段存在性和签名。
- **最小修复方向：** 无设计修复。

## 4. F017 条件项（Task-Planner 写集）

- **判定：仍开放，条件项，不计入阻断计数。**
- **证据：** `:4069` 和 `:4070`(7) 明确 `tasks.md` 写集更新属 Task-Planner；`:4069` 只列 `internal/recovery/runtime.go` 与 `cmd/han-server/recovery_runtime.go` 的建议改动，并明确“该表仍只作建议；不改 `tasks.md`、不改卡面、不勾选；写集更新前保持不可实施”。`tasks.md:3186` 仍为 `- [ ] identity-auth-S4.24a`。
- **影响：** 在 F006 修复前，`NewRuntime` 签名、lease 双字段、所有 lease 读取点与测试替身都没有任务卡承载；写集不更新则实现无处落卡。F017 不是本轮设计缺陷的替代判定，不能因任务卡未勾选而把 F006 算作已闭合。

## 5. 新问题

### 5.1 NEW-R05-01（HIGH，并入 F006，不重复计数）

lease 读取点缺口：`checkOwnership`、`CheckPhase`、`ReadPermitBinding`、`activate`、`beginObservation`、`replay`／`confirmSucceeded`、`admission.go:648` 仍读单一 `r.exclusive`。这是 F006 未闭合的直接证据，已并入 F006，不单独计入开放项或阻断计数。

### 5.2 NEW-R05-02（MEDIUM）— 作者证据链不可复算

- **证据：** 作者记录 1,045,042 B／`D2A6B54B…`；现行 1,045,111 B／`87D70DA8…`，差 +69 B。作者“+15 行”只对行数成立；现行相对 r64 为 +39,132 B，不是 +39,063 B。作者 §6.8 声称对 `:4057` 的 +69 B 校正未落盘：现行仍写“该版本由本轮之外的既有工作形成”“本轮不修改 ADR”；声称的“其外部见证与撤销判据由本轮同步”出现 0 次。备份 diff 实测为 13 行、+471 B，作者只列 12 行；`:4063` 内容变化被漏记，`:4057` 实际 +257 B。
- **触发场景：** 下游以作者 `hashes.txt` 或 §6.2 净值为基线复核时，无法对上现行文件；无法判断 +69 B 是校正、并发写入还是其他未登记改动。
- **影响：** 评审证据链可追溯性受损；可能导致未登记内容被误认为已审。设计语义判定不受影响，但发布前必须重新冻结。
- **最小修复方向：** 在文件冻结后重新生成作者报告与 `hashes.txt`；逐行列明 13 行差异、+471 B 与现行哈希；对 `:4057` 的实际文本二选一：保留现行“本轮之外形成”并修正作者报告，或恢复作者声称的校正文本并重新复算。

### 5.3 NEW-R05-03（LOW）— 历史段仍保留旧 V7／V8／V9 字面

- **证据：** `:3942` 与 `:3990` 仍出现旧 `V7`／`V8`／`V9`；两行都明确标为历史、不再构成定义。`:3990` 同时出现“以 r64.5／r64.2 为准”与后文“只以 r65.5／r65.2 为准”的表述并存。`:4062` 是现行唯一 V7–V11 定义。
- **触发场景：** 机械搜索 `V7` 的实现者可能先命中历史行，按旧步骤实现；或误读 `:3990` 的前半句为现行规范。
- **影响：** 低；语义标记已存在，但可读性和自动化检索仍有误读面。
- **最小修复方向：** 将历史行的 `V7`／`V8`／`V9` 改成“旧步骤一／二／三”或删除；统一 `:3990` 的取代指针为 r65.5／r65.2。

### 5.4 NEW-R05-04（MEDIUM）— 实现面映射不足

- **证据：**
  - `NewRuntime(` 全仓 Go 文件共 **39 处匹配、17 个文件**；生产调用点只有 `cmd/han-server/recovery_runtime.go:641`，其余大量为 `runtime_test.go`、`admission_test.go`、`installation_test.go`、`test/integration/recovery/runtime_fixtures_test.go` 等测试替身。r65.12 只列两个生产文件，未把测试调用面纳入写集。
  - `RestoreDriver` 接口（`internal/recovery/runtime.go:54`–`:59`）只有 `Prepare`／`Replace`／`VerifyRestored`，全仓没有 `driver.Bind` 方法或签名；但 `:3997`、`:4029`、`:4069` 反复要求把 lease “传给 `driver.Bind`”。
  - `newRecoveryRootAssembly`（`cmd/han-server/recovery_runtime.go:522`–`:650`）内没有 `Acquire`，也没有函数级 `defer Close`；`joinWork`（`:941`–`:964`）只覆盖 `normal`／`resume`，`enter`／`install` 在 `:957` 返回 `ErrAdmissionBinding`。r65.12 对 `enter` 只写“建议 Task-Planner 扩展”。
  - 生产 `controlledRestoreDriver.ownership` 字段（`cmd/han-server/recovery_restore.go:189`–`:192`）从未在非测试代码赋值；`Prepare` 在 `:216` 要求 `d.ownership != nil`，否则 `ErrAdmissionUnsupported`。仅测试 `recovery_restore_test.go:206`／`:230` 赋值。
- **触发场景：** 实施者按 r65.12 只改两个生产文件时，无法通过编译（测试调用点未补参数）；或实现了 `driver.Bind` 但代码中不存在该接口，必须自行发明方法名和所有权语义；或装配体未覆盖 `enter` 的 teardown，注入 lease 无法恰好关闭一次。
- **影响：** MEDIUM；不改变设计方向，但会让 F006 的修复在实现阶段再次产生未定义 API 和写集缺口。
- **最小修复方向：** 明确 `driver.Bind` 是否新增（给出完整签名、接口位置、调用者与 nil 语义），或删除该表述并改为既有构造路径；把 `NewRuntime` 全部调用点与测试替身纳入写集；给出 assembly 的 `Acquire` 与 `defer Close` 精确位置；明确 `ownership` 的生产赋值路径；把 `joinWork` 对 `enter` 的覆盖从“建议”提升为写集条件。

### 5.5 NEW-R05-05（LOW）— r61 段起点与旧哈希标签错误

- **现行事实：** r61 真标题在 `:4278`；`:4247` 是 r62.9.2 表头。按 raw 字节（含 LF/CR，至 EOF）复算：
  - 从 `:4247`：**282,599 B / `7D1DDC087A496FC2C02C26E5F04017F8C6900A19E20D596CF5A4F40E9150F05A`**。该值与旧记录一致，但起点是 r62.9.2 表头，不是 r61 标题。
  - 从 `:4263`：**280,319 B / `5BD8E87093A27AD40ED5A8DD056D56CE14160DB30A88DCEA26323DAC7E23B562`**。
  - 从 `:4278`（真正 r61 标题）：**277,334 B / `E40EB7A29AA909A1C8A2918F3A8FCF1C2013793A401C6046D8C276DE79A33254`**。
- **旧记录：** 282,706 B / `a1a256f5…` 在现行文件上不可复现，属历史快照；不能作为现行 r61 段基线。
- **触发场景：** 复审者按旧记录从 `:4247` 复算时，会得到 282,599 B 并误以为 r61 段从该处开始；若使用 282,706 B 记录则直接失配。
- **影响：** LOW；不影响设计正文语义，但影响历史证据的可追溯性与“r61 原字节恢复”的说明精度。
- **最小修复方向：** 作者报告把“r61 段”与“r62 尾段”分开写；若要报告 r61 段哈希，使用 `:4278` 起点并标明“至 EOF / raw 字节含 LF/CR”；若要报告设计 `:4063` 的合并范围，则明确写“r62 尾段 + r61 段，起点 `:4247`”。

## 6. 交叉核对与独立复算

### 6.1 ADR-0031

- `:1082` 为 `adr-0031-r09`；`:1084` 明确 r09 未经复评、决策接受级未达成、S4.24a 不得启动。
- `:1118` 同步 8 字段验证方载荷、5 键 witness head、五处 `anchor_ref`、`baseline_sequence` 只增不减、撤销／TOCTOU 每检查点重验。
- `:1147` 不变量 6 补见证独立性、单调性与人的批准、验签失败不推进。
- `:1166` 约束 8 第 6 条补 endpoint DNS／私网／TLS pin／query-fragment／独立性／单调性。
- `:1190` 有 r07 追溯行与 r09 跳号说明；ADR 全文 `NewRuntime` 为 **0 次**，故 F004 的“对应 ADR 未登记签名变更”成立。

### 6.2 golden vector（独立复算）

按 `:4082` 的 JCS 规则，取 `:4087`／`:4093` 原始行字节（去行尾 CR/LF），解析 JSON、递归排序键、紧凑 UTF-8 输出并计算 SHA-256：

| 向量 | 原始字节 | 原始 SHA256 | canonical == 原始行 | canonical SHA256 |
|---|---:|---|---|---|
| source `:4087` | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | True | 同左 |
| tool `:4093` | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | True | 同左 |

两条均与设计期望一致；原行本身即 canonical。此复算证明 canonical 字节和 SHA-256 一致，**不证明 Ed25519 签名、真实回执或生产链路可用**。

### 6.3 r61 raw 段

| 起点 | 字节 | SHA256 | 说明 |
|---|---:|---|---|
| `:4247` | 282,599 | `7D1DDC087A496FC2C02C26E5F04017F8C6900A19E20D596CF5A4F40E9150F05A` | r62.9.2 表头至 EOF；旧记录可复现但标签错误 |
| `:4263` | 280,319 | `5BD8E87093A27AD40ED5A8DD056D56CE14160DB30A88DCEA26323DAC7E23B562` | 中间边界，不是 r61 标题 |
| `:4278` | 277,334 | `E40EB7A29AA909A1C8A2918F3A8FCF1C2013793A401C6046D8C276DE79A33254` | 真正 r61 标题至 EOF；段内 `r64`／`r65` 均为 0 |

`:4423` 已恢复原句，结尾为“子进程实际 `environ` 与白名单逐字节不等即 closed。”。旧记录 282,706 B / `a1a256f5…` 在现行文件上不可复现，不能作为现行基线。

### 6.4 作者备份 diff

`%TEMP%\design-r65-prefix.bak.md` 为 1,044,640 B / `2BBBDC8410BEE8681DA3F3B648693EADD6CDA1CEC8856B90222F943EDB72DE61`；与现行文件逐行比较：

| 行 | 字节差 |
|---|---:|
| `:3990` | +255 |
| `:3997` | +59 |
| `:4013` | +22 |
| `:4017` | +41 |
| `:4029` | -36 |
| `:4051` | +64 |
| `:4057` | +257 |
| `:4058` | +154 |
| `:4060` | +133 |
| `:4063` | 0（内容变化，字节数不变） |
| `:4068` | -390 |
| `:4069` | -85 |
| `:4070` | -3 |
| **合计** | **+471（13 行）** |

该表用于证明作者“12 行／+69 B”的记录不完整，不作为设计正确性的替代证据。

## 7. NOT RUN 边界

以下均未实际验证，不得因文本合理、静态代码存在或 Windows 解析通过而写成 PASS：

- Linux host、POSIX ACL、mount 约束、loader／`ld.so` 闭包、fd-exec、`/proc/<pid>/maps`、netns／网络、external witness 的真实运行行为。
- 真实 Ed25519 签名、HSM、信任根、密钥轮换、审计 sink、人的批准与任命、真实旧备份与 restore 工具。
- 生产 restore E2E、`enter`／`resume` 解锁、生产可恢复性、约束 8 满足、`identity-auth-S4.24a` 已交付或已关闭。
- `go build`、`go test`、`go vet` 未运行；本轮代码核对仅限静态读取与 `rg`。
- 安全侧 finding（SEC-F002／F003／F006 等）未在本架构复审中重新判定；只在它们与 lease、wire、见证契约交叉处记录影响。
- ADR-0031 r09 未经本报告复评；ADR 状态段的“已接受但不构成 r09 接受”仍按原文保留。

## 8. 只读声明

本轮只读取对象与代码，未修改 `design.md`、`tasks.md`、ADR、`cmd/**`、`internal/**`、测试、`.add/project/**`，未改动既有 `review-arch-r04/`、`review-sec-r04/`、`design-revision-r*/`，未删除、移动或归档任何文件，未执行任何 Git 写操作。除本报告及同目录 `hashes.txt` 外，本角色未写入其他文件。

本报告不声称 S4.24a 已交付／已关闭，不声称生产可恢复，不声称 `enter`／`resume` 已解锁，不声称约束 8 已满足，也不代替 Lead 或人的接受、授权与风险决定。