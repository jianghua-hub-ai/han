# identity-auth 详细设计 r67 架构侧独立复审（B13J／S4.24a，review-arch-r07）

- **日期**：2026-11-02
- **角色与方法**：架构侧 Reviewer（`add-reviewer`），objectType=design；主方法 `.codex/skills/add-code-review/SKILL.md`（Round 1 对象面、Round 2 消费者面），必读 `.codex/guidance/code-review-checklist.md`、`abstraction-principles.md`，按对象加载 `design-doc-writing-style.md`、`decisions-guide.md`；未启用 profile 不作为要求（项目启用 `go-service`，本对象为设计，不套代码提交/构建条款）。
- **固定工作区**：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本次实测）。
- **只读声明**：本次唯一写入为本报告及同目录可选 `hashes.txt`。未修改 `design.md`、`tasks.md`、ADR（`framework.md`）、`cmd/**`、`internal/**`、测试、既有 `review-*/`、`design-revision-*/`；未删除/移动/归档；未执行 Git 写操作（未切分支、未 stash、未 commit）。

**Verdict: CHANGES REQUIRED**

- 阻断项 **2**：`F006`（BLOCKER，结转）、`NEW-R07-02`（BLOCKER，新）。
- 开放项 **6**：`F006`、`NEW-R07-01`（MEDIUM）、`NEW-R07-02`（BLOCKER）、`NEW-R07-03`（MEDIUM）、`NEW-R05-03`（LOW）、`NEW-R07-04`（LOW）。
- 本轮判闭合 **8**：`F004`（文本层）、`NEW-R05-04`、`NEW-R05-05`（行号半边）、`NEW-R06-02`、`NEW-R06-05`、`NEW-R06-06`、`NEW-R06-07`、`NEW-R06-08`。
- 条件项非阻断 **1**：`F017`。
- 待决（非 finding）**1**：ADR-0031 是否登记 `NewRuntime` 构造契约（`F004` 升级待决）。

## 0. 对象、冻结与基准

### 0.1 被评对象（冻结）

| 对象 | 路径 | 标识 | 字节 | LF 行 | SHA256 |
|---|---|---|---:|---:|---|
| 设计 | `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r67`（`:5`、`:38`） | 1,084,882 | 5,476 | `B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E` |
| 冻结快照 | `%TEMP%\design-r67-freeze-B12F1EF3.md` | 同字节 | 1,084,882 | — | `B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E` |
| r66 基线 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r07/design.r66.baseline.md` | r66 终值 | 1,079,559 | 5,476 | `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298` |
| r66 冻结快照 | `%TEMP%\design-r66-freeze-9B193BDE.md` | 同字节 | 1,079,559 | — | `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298` |
| 任务卡 | `.add/project/knowledge/tasks/tasks.md` | `:3186` 仍 `- [ ]` | 898,536 | 3,289 | `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` |
| ADR | `.add/project/knowledge/decisions/framework.md` | 现行 `adr-0031-r10`；`NewRuntime` 0 次 | 285,639 | 1,273 | `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050` |

- 版本行实测：`:5` = `revision: identity-auth-design-ir01-r67`；`:38` 为 `**设计版本**：` 后接代码字体版本号 `identity-auth-design-ir01-r67`。
- r66 块为 `:4072`–`:4083`（12 行）；r62 历史段自 `:4084` 起（`:4084` 为 r62 段标题，`:4255` 为 r62.9），与设计声明一致。
- r67 为就地修订、无增删行：内容变化恰 **10 行**（`:3942`、`:3990`、`:4062`、`:4072`、`:4073`、`:4076`、`:4135`、`:4137`、`:4369`、`:4373`），Δ=**+5,323 B**；`1,079,559 + 5,323 = 1,084,882`；行数不变。
- 评审期间对象未漂移：冻结快照与工作树两次 `Get-FileHash` 同字节；`git status --porcelain` 中 `design.md` 为既有未提交修改（`M`），298 条既有变更未由本报告触碰。

### 0.2 基准报告与哈希

| 报告 | 字节 | LF 行 | SHA256 | 说明 |
|---|---:|---:|---|---|
| `review-arch-r05/report.md` | 32,489 | 281 | `DD2423B7BD574B5C61603F938B42856061EA3F7B4AA59BA4D033B04BE5AD93D1` | CHANGES REQUIRED；`F006`/`F004` 来源 |
| `review-arch-r06/report.md` | 28,491 | 247 | `C66F9BBEB2C04103A9947078A8B86A807037C52ED890773696B2E7E30B53B47E` | 仅作线索；派发声明 24,561 B／`4AB7CFC6…`，已漂移（OBS-R07-02） |
| `design-revision-r06/report.md` | 24,041 | 196 | `CA5F28AC27DC4951AA6D4335F4DC2DE5CF0C714E3CCBF79C0E3EA93FF8A7C671` | 作者 r06 报告 |
| `design-revision-r07/report.md` | 18,099 | 202 | `B4AED3EDE54938FFE43687F6A052707D28AFDD567AB6CACBA04A7B26C879F4EB` | 作者 r07 报告（只作线索） |
| `design-revision-r07/hashes.txt` | 1,847 | 36 | `33EDAD36475A161C37C9DB8926916FED7FC2592F7F1AB0204A6DCE5DC943D604` | 作者哈希清单 |

所有数值均由本报告自算；作者报告的结论与数值不替代复核。

### 0.3 检查限制

- r65 原始文件不可得（作者报告 §3.1 已登记），r65 列为「备份 + R05 已登记 13 行」的**重建值**；本报告按作者明确标注判定，不以重建值冒充一手实测。
- ADR-0031 正由并行会话改写为 `adr-0031-r10`；本报告只读现时内容，不评价并行改写结果。
- 未编译、未运行测试、未启动服务；实现对照只证明“设计落到现役代码是否可实施”。

## 1. 方法与两轮实际检查范围

### 1.1 Round 1：对象面

- 固定对象字节/行/哈希，并核对冻结快照；
- 逐行读取 r67 变更 10 行及相邻 r66 块（`:4072`–`:4083`）、r62 起点（`:4084`）、r61 标题（`:4290`）；
- 复算 r66→r67 10 行字节表、r65→r66 16 行字节表与四条整文件恒等式；
- 独立复算两条 golden vector（`:4099`、`:4105`）；
- 核对作者报告结构重复、旧值残留、自述哈希；机械检索 P/R/V 编号与行号指向。

### 1.2 Round 2：消费者面（只读真实实现）

- `internal/recovery/runtime.go`：`NewRuntime`（`:729`）、单一 `exclusive` 字段（`:768`）、`checkOwnership`（`:1160`–`:1177`）、`activate`（`:917`–`:921`，调用点 `:892`／`:1943`）、`beginObservation`（`:948`–`:966`）、`drain`（`:1418`–`:1425`）、`recoveryPreflight`（`:1497`–`:1513`）、`recoveryFlow.acquire`（`:1554`–`:1571`）、`f.enter`（`:1573`–`:1597`）、`f.identity`（`:1836`–`:1839`）、`confirmSucceeded`（`:1968`）、`closeObservationLease`（`:1022`–`:1023`）、`installationFlow.acquire`（`:486`–`:488`）、admission 写入（`:870`–`:876`）、`Enter`（`:1517`–`:1535`）；
- `internal/recovery/admission.go`：`readControlled`（`:616`–`:652`，重点 `:617`／`:648`）；
- `cmd/han-server/recovery_runtime.go`：`start`（`:872`–`:896`）、`completeOnRequest`（`:901`–`:914`）、`joinWork`（`:942`–`:964`）；
- `cmd/han-server/recovery_sources.go`：`CompleteIdentityReview`（`:314`–`:327`）、`completeIdentity`（`:332`–`:357`）；
- `cmd/han-server/main.go`：`start` 与 control server 启动顺序（`:770`／`:781`）；
- 装配/测试：`NewRuntime(` 39 行／17 文件；`RestoreDriver`（`:55`–`:59`）无 `Bind`；`recovery_restore.go:203` 四参构造、`:216`／`:378`–`:381` 的 ownership 非 nil 要求；`recovery_restore_test.go:204`／`:208`／`:209`／`:228`／`:233`；`restore:` 字面量（`recovery_enter_install_integration_test.go:232`、`recovery_production_chain_integration_test.go:2999`）；`recoveryRootInputs.restore`（`recovery_runtime.go:441`、`:596`–`:599`）。

### 1.3 限制

- 只读静态对照，未执行改造后代码；因此“可实施”不等于“已工作”。
- 未覆盖 Linux host/ACL/fd-exec/见证等运行面；见 §7 NOT RUN。

## 2. 结论与计数

| 类别 | 数量 | 项 |
|---|---:|---|
| 阻断 | 2 | `F006`、`NEW-R07-02` |
| 开放 MEDIUM | 2 | `NEW-R07-01`、`NEW-R07-03` |
| 开放 LOW | 2 | `NEW-R05-03`、`NEW-R07-04` |
| **开放合计** | **6** | 上述 6 项 |
| 本轮闭合 | 8 | `F004`（文本层）、`NEW-R05-04`、`NEW-R05-05`（行号）、`NEW-R06-02`、`NEW-R06-05`、`NEW-R06-06`、`NEW-R06-07`、`NEW-R06-08` |
| 条件项非阻断 | 1 | `F017` |
| 待决 | 1 | ADR-0031 的 `NewRuntime` 构造契约登记（`F004` 升级待决） |

**Verdict 依据**：r67 已修掉 r06 指出的 `checkOwnership` 自锁与 nil 检查无落点，并补齐字节账、测试字面量与编号指向；但 `F006` 仍有三处未闭合（`activate` 访问器二义、`admission.go:648` 数据流缺口、`enter` 生命周期死锁），其中 `NEW-R07-02` 使 `enter` 分支按字面不可实现，故不给 PASS。

## 3. 逐条判定（派发项 1–8）

### 3.1 任务 1 — `F006`：**未闭合（BLOCKER）**

| 子项 | 判定 | 证据 |
|---|---|---|
| `checkOwnership` 自锁修复 | 闭合（文本） | `:4076` 第(1)点要求 `:1162` 改 `activeExclusiveLocked()` 且禁止 `activeExclusive()`；现役 `runtime.go:1161` 持锁、`:1162` 读 `r.exclusive`、`:1164` 解锁，落点存在；机核判据明确 |
| `CheckPhase`／`ReadPermitBinding` 继承 | 闭合（文本） | 二者经 `checkOwnership`（`:1196`、`:1245`），访问器修复自动覆盖 |
| `activate` | **未闭合** | 见 `NEW-R07-01`：`:918` 已持锁、`:919` 读 `r.exclusive`；设计只说“改用访问器”，未指明 `activeExclusiveLocked()`，且只点名 `:1943`，漏 `:892` |
| `beginObservation` | 闭合（文本） | `:948` 持锁到 `:966`；设计要求 `activeExclusiveLocked()` + 仅 owned，可落 |
| `replay`／`confirmSucceeded` | 部分闭合 | `replay` 经 `:1943` 已点名，但同函数 `:892` 未点名（并入 `NEW-R07-01`）；`:1968` 读 `r.exclusive`，设计要求改用访问器，落点存在 |
| `admission.go:648` | **未闭合** | 见 `NEW-R07-03`：`checkOwnership` 仅返回 error，入口无 lease 值可“贯穿” |
| `enter` 缺失 lease | 访问器侧闭合 | injected 动作只读 `injectedExclusive`；nil → `ErrAdmissionClosed`；“缺失 owned lease”分支不会命中；生命周期侧另见 `NEW-R07-02` |
| nil 检查“状态变更之前” | 闭合（文本） | `recoveryPreflight`（`:1497`–`:1513`）早于 `beginAction`（`:1525`／`:1542`）与 `StopAndJoin`（`:1556`）；检查点插在 `:1513` 前，严格前移 |
| teardown 只关 `ownedExclusive` | 闭合（文本） | `drain`（`:1418`–`:1425`）与 `closeObservationLease`（`:1022`–`:1023`）只关 owned；设计第(8)(9)点明确 |
| 生产 `ownership` 赋值点 | 闭合（文本） | `installationFlow.acquire`（`:486`–`:488`）与 admission（`:870`–`:876`）写入；设计第(10)点保持并改 `ownedExclusive` |

**`F006` 结论**：访问器方向与 nil/teardown 落点已对，但 `activate` 二义、`admission.go:648` 数据流缺口、`enter` 生命周期死锁三项使其仍不可按字面实施；BLOCKER 结转。

### 3.2 任务 2 — `F004`：**文本层闭合；ADR 同步升级待决（非阻断）**

- `:3603` 已就地写入 r66.1 句（`restoreLease` 是既有 `exclusive` 端口一次 `Acquire` 的产出，不新增端口，不进 11 项表）；`:4073` 固定唯一签名 `NewRuntime(ctx context.Context, d RuntimeDependencies, restoreLease ExclusiveLease)`，第三参数**非可选**、允许 nil（nil = restore 动作保持 closed），禁止变参/`Bind*`/全局/字段注入。
- 独立实测：`internal/recovery/runtime.go:729` 仍为两参（待实施）；`rg -n 'NewRuntime\('` = 39 行／17 文件；生产唯一调用 `cmd/han-server/recovery_runtime.go:641`；ADR 全文 `NewRuntime` = **0** 次。
- **ADR 是否需要同步**：现 ADR 无该构造契约文字，故无直接条文冲突；但该签名把“restore 动作可用性”编码为构造期非可选参数，属于对 ADR-0031 ownership/装配语义的显式化。**升级待决**：由 Lead/架构决策方决定是否在 ADR-0031（现由并行会话改写至 `r10`）登记；本报告不改 ADR、不计 finding、不阻断。若登记，须在 r10 落定后复核一致性。

### 3.3 任务 3 — `NEW-R05-04`／`NEW-R06-05`／`NEW-R06-06`：**闭合**

- `RestoreDriver.Bind`：现役 `runtime.go:55`–`:59` 为 `Prepare`/`ValidateRestorePlan`/`Execute` 三方法，无 `Bind`；全仓 `func .*Bind\(` 6 处均为测试函数名（如 `TestRecoveryRuntimeExitRouterRefusesBeforeBind`）。设计明确“不得新增”，并把 r65 旧接线句标为历史 → 映射闭合。
- `newRecoveryRootAssembly` 的 `Acquire`／`defer Close`：设计固定 `anchors.Exclusive.Acquire(ctx)` 恰好一次（仅 enter/resume）、同一 lease 交工厂、`closeRestoreLease()` + `sync.Once` 恰好一次；现役 `recoveryRootInputs.restore` 类型为 `recovery.RestoreDriver`（`recovery_runtime.go:441`），`:596`–`:599` 为 nil→closed 占位 → 工厂改型可落。
- `joinWork` 覆盖 `enter`：设计给出 `case "enter"`（`Stop`→`waitFlow`→`closeRestoreLease`）；现役 `joinWork`（`:942`–`:964`）只有 normal/resume，缺 enter 属待实施；**但其“先 Stop 再 waitFlow”的 in-flight 语义与 `NEW-R07-02` 的同步 enter 冲突**（见 §4.2）。
- 生产 `ownership` 赋值点：`recovery_restore.go:203` 四参构造、`:216` 与 `:378`–`:381` 要求 `d.ownership != nil`；设计要求增第五参并在构造期赋值 → 映射闭合。
- 测试面：`recovery_restore_test.go:204` 构造 driver、`:208` `f.world = newS414World(p)`、`:209`／`:233` 赋 ownership；设计与现役一致（重排 + 删两处赋值）；`:228` `NewRuntime` 补第三参。`restore:` 字面量恰 `recovery_enter_install_integration_test.go:232` 与 `recovery_production_chain_integration_test.go:2999`（`:2464` 是 `RuntimeDependencies.Restore`，不属该字段）→ 清单闭合，漏项风险已消除。
- `recoveryRootInputs.restore` 现类型与 nil 工厂占位支持工厂改型；`main.go:634`、`recovery_runtime_test.go:731` 未设该字段，依赖 nil 工厂 → 与设计一致。

### 3.4 任务 4 — `NEW-R06-02`：**闭合（字节账自洽）**

- 作者 16 行表逐列复加（本报告独立计算，见 §5.3）：
  - bak 列和 **28,976**；R05 Δ 和 **+412**；r65 重建列和 **29,388**；r66 列和 **32,695**；Δ 和 **+3,307**。
  - 逐行校验 `r65 = bak + R05Δ`：16/16 行相等（0 处不符）。
  - 整文件恒等式：`1,044,640 + 471 = 1,045,111`；`1,079,559 − 1,045,111 = 34,448`；`3,307 + 31,141 = 34,448`；`34,448 + 5,323 = 39,771 = 1,084,882 − 1,045,111`。
  - 残余 1,555：`3,307 − 468 − 1,284 = 1,555`，由 16 行整行口径吸收；旧 468/1,284 为片段级历史，不再参与配平。
- `design.r66.baseline.md` 实测 **1,079,559 B / 5,476 LF / `9B193BDE…`**，与派发声明一致；`%TEMP%\design-r66-freeze-9B193BDE.md` 同字节。
- r65 列在作者报告 §3.1/§3.2 明确标注「重建」，并给出备份哈希与 +471 B 依据 → 未把重建值冒充一手实测。**闭合**（残余风险：原始 r65 不可得；若日后取得应替换重建列，不阻断）。

### 3.5 任务 5 — `NEW-R06-07`：**闭合**

- `:3942` 与 `:3990` 均已含 `r66.2`，并指向 `:4062` 与 `:4074`；`:4074` 确为 `**r66.2 V7–V11 的载荷与签名主体唯一绑定…**`；`:4062` 为 V7–V11 现行定义。两处不再只列 r64.5／r65.5，指向不再自相矛盾（V1–V11 口径的残余矛盾另记 `NEW-R05-03`）。

### 3.6 任务 6 — `NEW-R06-08`：**闭合**

- 作者 r07 报告：`## ` 8 个、`### ` 14 个，重复标题 0；旧值 `1,077,602` 命中 0；自述终值与磁盘一致（18,099 B／202 LF／`B4AED3ED…`）。结构与旧值残留已消除。

### 3.7 任务 7 — `NEW-R05-03`／`NEW-R05-05`：**`NEW-R05-03` 未闭合（LOW）；`NEW-R05-05` 行号闭合、哈希半边转 `NEW-R07-04`**

- `NEW-R05-03`：`:4079` 声明“唯一有效的 V 编号是 `:4062` 的 V7–V11”；但 `:4037`（r64.5）完整定义 **V1–V11**，`:4062` 自身又写“`R1`–`R8` 与 `V1`–`V11` 只枚举契约检查点”，`:3942` 也写“现行 P1–P6／R1–R8／V1–V11 只以 r64.5／r65.5 及 r66.2 为准”。三处对“V1–V11 是否有效”给出不同答案；虽 r66.0 的优先级规则可裁定 `r66.7 > r66.2`，但 `:4079` 对 `:4062` 的描述本身不成立（`:4062` 含 V1–V11，不只 V7–V11）。**仍未唯一可判**（LOW）。
- `NEW-R05-05`：r61 真标题 `:4290` = `**r61 生产 restore 接线的实现形态（B13J／S4.24a）`；`:4259` 为 r62.9.1 表头；`:4257` 为 r62.9.1 标题。**行号更正准确**。
- 但 `:4072` 所称“现行值”三段 raw 后缀哈希（`:4259` 282,599 B／`7D1DDC08…`、`:4275` 280,319 B／`5BD8E870…`、`:4290` 277,334 B／`E40EB7A2…`）实测只匹配 **r66 baseline**；r67 现行分别为 **283,480 B／`E8848838…`**、**281,200 B／`2F53882F…`**、**278,215 B／`B0AF3B04…`**，每条恰 +881 B（`:4369` +91 + `:4373` +790）。→ **`NEW-R07-04`（LOW）**。

### 3.8 任务 8 — `F017`：**非阻断，未擅改**

- `tasks.md` 实测 898,536 B／3,289 LF／`4BD724A9…`，与派发声明一致；`:3186` 仍为 `- [ ] identity-auth-S4.24a 接入生产 restore source/target/tool 与批准主体接线；受 ADR-0031 门禁，与锁后端节点独立。`
- 未见本轮对 `tasks.md` 的擅改；`F017` 不阻断。任务卡仍未覆盖 r66/r67 的写集（既有冲突），交 Task-Planner 处理。
## 4. Findings（按严重度排序）

`F006` 与 `NEW-R07-02` 同处 r66.4 块但根因不同：前者是 lease 访问器/所有权契约，后者是 `enter` 生命周期与完成握手；不合并。

### [BLOCKER] F006 — 按动作选择 lease 与生命周期仍不可按字面实施（结转）

- **finding ID**：`F006`（复审引用原 ID）
- **对象/修订**：design／`identity-auth-design-ir01-r67`（`B12F1EF3…`）
- **位置**：`design.md:4076`；消费者 `internal/recovery/runtime.go:1160`–`:1177`、`:917`–`:921`、`internal/recovery/admission.go:616`–`:652`、`cmd/han-server/recovery_runtime.go:872`–`:896`
- **现状**：访问器模型、nil 检查点、teardown 已给出；但 `activate` 未强制持锁变体、`readControlled` 无入口 lease 数据流、`enter` 同步执行与既有 identity 确认握手冲突（三个根因分别见 `NEW-R07-01`／`NEW-R07-03`／`NEW-R07-02`）。
- **依据**：`code-review-checklist` §2「资源/事务/状态所有权唯一，失败路径能收敛」；`abstraction-principles` §1（所有权、状态不变量）与 §5（失败/回退边界）；`add-code-review` design 分支「各视图能映射到实现」。
- **触发**：实现者按 `:4076` 逐字落地；或启用 `enter` 的 secret 通路。
- **影响**：`activate` 自锁（RunNormal/replay 卡死）；`admission.go:648` 读取错误 lease 或无法“同一次”；`enter` 在 identity 审核处等待永不到来的完成信号，受 `ioBudget` 超时后失败。
- **建议**：分别按 `NEW-R07-01`／`NEW-R07-02`／`NEW-R07-03` 的最小修复方向修订 `:4076`；不改变访问器总体方向。
- **验证**：三项机核判据 + `-race` 并发测试 + `enter` 完成握手集成测试（见各 finding）。

### [BLOCKER] NEW-R07-02 — `enter` 分支“同步运行并缓存”与 identity 完成握手冲突，且 `completeOnRequest` 未在设计中出现

- **finding ID**：`NEW-R07-02`
- **对象/修订**：design r67
- **位置**：`design.md:4076`（`start` 的 `enter` 分支）；消费者 `cmd/han-server/recovery_runtime.go:872`–`:896`、`:901`–`:914`；`cmd/han-server/main.go:770`／`:781`；`internal/recovery/runtime.go:1517`–`:1535`、`:1573`–`:1597`、`:1836`–`:1839`；`cmd/han-server/recovery_sources.go:314`–`:327`、`:332`–`:357`
- **现状**：设计要求 `start` 的 `enter` 分支“**同步运行并缓存** `Enter(...)` 返回值”，**禁止 fire-and-forget**，并称 `resume` 保持现有 `setFlowResult` 结构。现役 `start` 的 `resume` 分支起两个 goroutine：`Resume`（`:885`）与 `completeOnRequest`（`:889`）；`enter` 无分支。`Enter` → `f.enter` → `f.identity` → `CompleteIdentityReview`（`recovery_sources.go:314`）在 `:319`–`:326` 阻塞等待 `completeDone`；该通道只由 `completeIdentity`（`:356`）关闭，而 `completeIdentity` 只由 `completeOnRequest`（`recovery_runtime.go:901`）驱动。设计全文 `completeOnRequest`／`waitCompleteRequest` 命中 **0**。`main.go:770` 调 `start`，`:781` 才启动 control HTTP；`joinWork`（`:942`–`:964`）无 `enter` 分支。
- **依据**：`code-review-checklist` §2「主路径、边界、异常、并发、取消、重试和幂等与设计一致」；`add-code-review` design 分支「各视图能映射到实现」；`abstraction-principles` §5「失败、回退、兼容和迁移边界是否明确」。
- **触发**：按设计实现 `enter` 分支并启用生产 secret 通路（设计称当前生产 secret 通路不存在，故现状被“进入即 closed”掩盖；一旦接通即触发）。执行顺序：`main` → `start`（同步 `Enter`）→ `f.identity` → `CompleteIdentityReview` 等待 → control server 尚未启动 → 操作员无法确认 → 等待 `ioBudget` 超时。
- **影响**：`enter` 永远无法完成 identity 审核；启动返回错误，恢复入口不可用；与 `joinWork` 的“先 Stop 再 waitFlow reap 已启动工作”自相矛盾（同步调用没有在飞工作可 reap）；“resume 保持现有结构”的对照也说明 enter 缺少完成驱动。
- **建议**：把 `enter` 改为与 `resume` 同构的**可 join 的 goroutine**（`Enter` 在 goroutine 内运行并 `setFlowResult`，另起 `completeOnRequest` 驱动 identity 完成），或明确证明 `enter` 不经 identity 审核；删除“同步运行并缓存/禁止 fire-and-forget”的绝对措辞，并让 `joinWork` 的 enter 分支与该结构一致。不得把操作员确认步骤移出 control 面。
- **验证**：机核 `rg -n 'case "enter"' cmd/han-server/recovery_runtime.go` = 1，且分支内启动 `Enter` 与 `completeOnRequest`；集成测试：`enter` 启动后 control server 可服务、确认后 `CompleteIdentityReview` 返回、`joinWork` 恰好 join 两个 goroutine、无泄漏；超时路径返回可诊断错误。

### [MEDIUM] NEW-R07-01 — `activate` 的访问器二义：持锁区间内“改用访问器”未指定 `activeExclusiveLocked()`，且调用点清单漏 `:892`

- **finding ID**：`NEW-R07-01`
- **对象/修订**：design r67
- **位置**：`design.md:4076` 第(3)点；`internal/recovery/runtime.go:917`–`:921`、`:892`、`:1943`
- **现状**：`:918` 持 `r.mu`，`:919` 在同一临界区读 `p, lease, generation := r.proof, r.exclusive, r.generation`。设计只说“`activate`（`:917`／`:919`）改用访问器”，未写“`activeExclusiveLocked()`”；而访问器定义中 `activeExclusive()` 会自取 `r.mu`。`activate` 的调用点是 `:892`（RunNormal）与 `:1943`（replay），设计只点名 `:1943`。
- **依据**：`code-review-checklist` §2（并发与状态所有权）；`abstraction-principles` §1（动作/状态不变量）；`add-code-review`「每个疑点必须定位到最小行范围并说明触发条件」。
- **触发**：实现者按“改用访问器”字面在 `:919` 调用 `activeExclusive()`。
- **影响**：Go `sync.Mutex` 不可重入 → 自锁；RunNormal 与 replay 卡死。与 `:4076`“全部读取点，无遗漏”的机核声明不符。
- **建议**：把第(3)点改写为“`:919` 改为 `p, lease, generation := r.proof, r.activeExclusiveLocked(), r.generation`（持锁区间，禁 `activeExclusive()`）”，并补列 `:892` 调用点；机核判据增加 `activate` 函数体内 `activeExclusive(` = 0。
- **验证**：静态检查 + RunNormal/replay 的 `-race` 测试通过；`rg` 机核。

### [MEDIUM] NEW-R07-03 — `admission.go:648` 的“入口同一次访问器结果”没有数据流，`checkOwnership` 只返回 error

- **finding ID**：`NEW-R07-03`
- **对象/修订**：design r67
- **位置**：`design.md:4076` 第(2)点；`internal/recovery/admission.go:616`–`:652`（`:617`、`:648`）；`internal/recovery/runtime.go:1160`–`:1177`
- **现状**：`readControlled` 在 `:617` 调 `checkOwnership(ctx)`，该函数只返回 `error`；`:648` 直接读 `r.exclusive` 传给 `recheckProof`。设计要求“改为使用入口处取得的同一次访问器结果，并贯穿到 `recheckProof`”，但未给出任何数据流：要么改 `checkOwnership` 返回 lease（影响 7 个调用者），要么在 `readControlled` 入口另取访问器（与 `checkOwnership` 内部快照不是“同一次”）。
- **依据**：`code-review-checklist` §2「资源/事务/状态所有权唯一，失败路径能收敛」；`abstraction-principles` §1（所有权唯一）。
- **触发**：实现者按字面在 `:648` 调 `activeExclusive()`；或改签名但未覆盖全部调用者。
- **影响**：`:617` 与 `:648` 之间 ownership 可能被替换/关闭，`recheckProof` 用错 lease；或产生“看似同一次、实际两次快照”的假安全。属并发正确性与所有权契约风险。
- **建议**：明确数据流，二选一并写入设计：(a) 在 `readControlled` 入口取一次 `activeExclusive()`，把该值传给 `checkExclusive` 与 `:648` 的 `recheckProof`，并删除 `:617` 的重复读；(b) 让 `checkOwnership` 返回 lease，逐点更新调用者（`:293`、`:404`、`:1123`、`:1196`、`:1245`、`admission.go:617`）。
- **验证**：静态检查 `readControlled` 内 `r.exclusive` 直读 = 0；并发测试在 ownership 替换窗口内验证 `recheckProof` 使用同一 lease；`-race`。

### [LOW] NEW-R05-03 — “唯一有效 V 编号”与 `:4037`/`:4062`/`:3942` 的 V1–V11 表述冲突，仍非唯一可判（结转）

- **finding ID**：`NEW-R05-03`
- **对象/修订**：design r67
- **位置**：`design.md:4079`、`:4037`、`:4062`、`:3942`
- **现状**：`:4079` 称唯一有效 V 编号是 `:4062` 的 V7–V11；`:4037` 定义 V1–V11；`:4062` 写“R1–R8 与 V1–V11 只枚举契约检查点”；`:3942` 写“现行 P1–P6／R1–R8／V1–V11 只以 r64.5／r65.5 及 r66.2 为准”。r66.0 的优先级规则虽可裁定修订号，但 `:4079` 对 `:4062` 的描述与 `:4062` 自身文本不符。
- **依据**：`design-doc-writing-style` 三条最高准则（克制代号、正文可独立读懂）；`code-review-checklist` §7（只报告可操作问题并引用具体行）。
- **触发**：实现/评审按 `:4079` 检索 V 编号。
- **影响**：读者无法仅凭一条规则判断 V1–V6 是否有效；可能误删/误改检查点。局部文档一致性问题，不影响正确性判定。
- **建议**：把 `:4079` 改为“V1–V11 均为有效契约检查点；V7–V11 的载荷与签名主体以 `:4062`/`:4074` 为准”，或明确“V1–V6 已并入历史、仅 V7–V11 有效”并同步 `:4037`/`:4062`/`:3942` 的措辞。
- **验证**：`rg -n 'V1–V11|V7–V11|唯一有效' design.md` 的命中在语义上一致；人工复核三处。

### [LOW] NEW-R07-04 — `:4072` 的“现行值”三段 raw 后缀哈希在 r67 后已陈旧（每条 +881 B）

- **finding ID**：`NEW-R07-04`
- **对象/修订**：design r67
- **位置**：`design.md:4072`（声明值）；实测对象 `:4259`、`:4275`、`:4290`
- **现状**：`:4072` 写“按 raw 字节（含 LF/CR）自该行首至 EOF 复算，现行值：`:4259` = 282,599 B／`7D1DDC08…`；`:4275` = 280,319 B／`5BD8E870…`；`:4290` = 277,334 B／`E40EB7A2…`”。实测这三值精确匹配 **r66 baseline**；r67 现行（`:4369` +91、`:4373` +790，均位于 `:4290` 之后）为 283,480 B／`E8848838…`、281,200 B／`2F53882F…`、278,215 B／`B0AF3B04…`。
- **依据**：`code-review-checklist` §1（验证证据真实、可复核）；`design-doc-writing-style` §2（正文只写事实）。
- **触发**：读者按“现行值”直接比对 r67 文件。
- **影响**：证据链再次出现“旧值标为现行”的同类问题（`NEW-R05-02`/`NEW-R06-02` 同簇）；不影响设计行为判断，但降低可复核性。
- **建议**：把标签改为“r66 时点值（r67 前）”，并补 r67 现行三值与 +881 B 的来源（`:4369` +91、`:4373` +790）；或在 r67 定稿后重算替换。
- **验证**：`Get-FileHash`/raw 后缀复算三行，与文中值逐一相等。

### 观察项（非阻断，不计开放项）

- **OBS-R07-01**：`:4072` 称“中间边界 `:4263`（插入前；插入后 `:4275`）”，但实测 r66 baseline `:4263` 是 r62.9.1 文件表的一行（`| cmd/han-server/recovery_restore_target.go ...`），`:4275` 是 `**r62.9.2 写集影响说明**` 标题；+12 的算术成立，但“中间边界”的命名不准确。建议改为具体锚点（如 r62.9.1 表末行）或删除该短语。
- **OBS-R07-02**：`review-arch-r06/report.md` 派发声明 24,561 B／`4AB7CFC6…`，磁盘实测 28,491 B／247 LF／`C66F9BBE…`；r06 仅作线索，其数值不作为本轮接受依据。
- **OBS-R07-03**：ADR-0031 正由并行会话改写为 `adr-0031-r10`；本报告只读现时内容，不评价并行改写结果（见 §3.2 升级待决）。
- **OBS-R07-04**：`tasks.md` 与 ADR-0031 门禁的既有冲突未变（`:3186` 仍 `- [ ]`），交 Task-Planner，不计本轮 finding。
## 5. 独立复算证据

### 5.1 两条 golden vector

| 位置 | 期望 | 实测 | 判定 |
|---|---|---|---|
| `design.md:4099`（source） | 1,115 B／`de914622…` | **1,115 B／`DE914622D3F8E90724D6C6FA83E0BD40D98E6510E2BBCF3BD592077C2E7B0CAB`** | 一致 |
| `design.md:4105`（tool） | 2,357 B／`8c0c89…` | **2,357 B／`8C0C89E3DAF0E024F266FF99443A6B54A2393D177D6CFAED27D21B894A326508`** | 一致 |

方法：取整行（不含行尾 LF）的 UTF-8 字节后计算 SHA-256；结果与派发期望逐字节一致。

### 5.2 r66→r67 10 行变更（自算）

| 行 | r66 B | r67 B | Δ |
|---|---:|---:|---:|
| `:3942` | 2,214 | 2,344 | +130 |
| `:3990` | 3,371 | 3,501 | +130 |
| `:4062` | 1,890 | 2,234 | +344 |
| `:4072` | 3,155 | 4,280 | +1,125 |
| `:4073` | 4,837 | 5,023 | +186 |
| `:4076` | 8,096 | 10,088 | +1,992 |
| `:4135` | 1,028 | 1,312 | +284 |
| `:4137` | 914 | 1,165 | +251 |
| `:4369` | 157 | 248 | +91 |
| `:4373` | 665 | 1,455 | +790 |
| **合计** | | | **+5,323** |

`1,079,559 + 5,323 = 1,084,882`；行数 5,476 不变；与作者表一致。

### 5.3 r65→r66 16 行表复加（独立计算）

| 行 | bak B | R05 Δ | r65 B（重建） | r66 B | Δ（r65→r66） |
|---|---:|---:|---:|---:|---:|
| `:3603` | 668 | 0 | 668 | 1,518 | +850 |
| `:3942` | 1,874 | 0 | 1,874 | 2,214 | +340 |
| `:3973` | 748 | 0 | 748 | 893 | +145 |
| `:3990` | 2,725 | +255 | 2,980 | 3,371 | +391 |
| `:3997` | 1,882 | +59 | 1,941 | 2,163 | +222 |
| `:4013` | 305 | +22 | 327 | 478 | +151 |
| `:4029` | 3,784 | −36 | 3,748 | 3,924 | +176 |
| `:4051` | 1,151 | +64 | 1,215 | 1,365 | +150 |
| `:4060` | 3,164 | +133 | 3,297 | 3,536 | +239 |
| `:4061` | 2,510 | 0 | 2,510 | 2,678 | +168 |
| `:4062` | 1,606 | 0 | 1,606 | 1,890 | +284 |
| `:4063` | 814 | 0 | 814 | 941 | +127 |
| `:4069` | 3,518 | −85 | 3,433 | 3,616 | +183 |
| `:4090` | 1,298 | 0 | 1,298 | 1,392 | +94 |
| `:4155` | 1,114 | 0 | 1,114 | 1,228 | +114 |
| `:4187` | 1,815 | 0 | 1,815 | 1,488 | −327 |
| **合计** | **28,976** | **+412** | **29,388** | **32,695** | **+3,307** |

- 逐行 `bak + R05Δ = r65`：**16/16 相等**。
- 整文件：`1,044,640 + 471 = 1,045,111`；`1,079,559 − 1,045,111 = 34,448`；`3,307 + 31,141 = 34,448`。
- r66 插入块 `:4072`–`:4083`（含末行 LF）= 31,141 B，与作者值一致。
- **这是对作者表的独立复加**；r65 源文件不可得，故未对重建列做一手实测（NOT RUN）。

### 5.4 其他机核

- `NewRuntime(` = **39 行／17 文件**（`rg -n` / `rg -l` / `rg -c`）；生产唯一调用 `cmd/han-server/recovery_runtime.go:641`。
- `RestoreDriver`：`runtime.go:55`–`:59` 三方法，无 `Bind`；全仓 `func .*Bind\(` 6 处均为测试函数名。
- `restore:` 字面量：`recovery_enter_install_integration_test.go:232`、`recovery_production_chain_integration_test.go:2999`。
- 单一 `exclusive` 字段的读取/写入点：`:486`–`:488`、`:870`–`:876`、`:917`–`:921`、`:948`–`:966`、`:1022`–`:1023`、`:1160`–`:1177`、`:1418`–`:1425`、`:1554`–`:1571`、`:1601`／`:1614`、`:1968`；全仓无 `ownedExclusive`／`injectedExclusive`／`r.action`（尚未实施，符合“待实施”）。
- `completeOnRequest`／`waitCompleteRequest` 在 `design.md` 命中 **0**。
- `:4074` = r66.2 标题；`:4084` = r62 段标题；`:4257` = r62.9.1 标题；`:4259` = r62.9.1 表头；`:4275` = r62.9.2 标题；`:4290` = r61 真标题。
- 作者 r07 报告：`## ` 8、`### ` 14、重复 0、`1,077,602` 命中 0。

## 6. 对并行 `review-arch-r06` 的同意/不同意

- **同意**其方向性结论：`NEW-R06-03`（`checkOwnership` 按 r66.4 字面接入会自锁）、`NEW-R06-04`（nil 检查在现调用图无“状态变更之前”落点）、`NEW-R06-05`（`f.world.lease` 早于 `f.world`）、`NEW-R06-06`（`recoveryRootInputs.restore` 测试字面量漏列）、`NEW-R06-02`（字节账不可复算）以及 `F006`／`F004` 的判定。r67 已修掉其中多数；`F006` 因 `NEW-R07-01/02/03` 仍未闭。
- **不同意**把 r06 报告的 24,561 B／`4AB7CFC6…` 当作现行基线：磁盘实测 28,491 B／247 LF／`C66F9BBE…`，r06 自身已漂移（OBS-R07-02）。
- r06 的 `NEW-R06-01`（对象漂移）不适用于本轮：本轮冻结哈希与磁盘一致，评审期间未漂移。
- 上述同意不作为本轮接受依据；所有判定均以本报告自算为准。

## 7. 未验证风险（NOT RUN）

**NOT RUN**：`go build`、`go test`、`go vet`、`-race`、E2E、真实服务/DB、Linux host/ACL/mount/loader/fd-exec/maps/network/witness、SEC-F007、除两条 golden vector 外的任何向量、r65 原始字节、ADR 并行改写的最终结果。实现对照只证明“设计到现役代码的映射”，不证明改造后可工作。

**不得据此声称**：生产解锁、`enter`/`resume` 解锁、约束 8 满足、`identity-auth-S4.24a` 可关闭或已交付。`F004` 文本闭合不等于实现正确；`NEW-R06-05/06` 闭合不等于编译通过；本报告不构成独立评审通过以外的任何授权。

## 8. 复审条件

1. 对象修订 ≥ r68，提供新哈希与冻结快照；若 `design.md` 再次漂移，按新快照重判，不混用版本。
2. 关闭 `NEW-R07-02`（enter 生命周期）与 `NEW-R07-01`／`NEW-R07-03`（`F006` 残余）；给出可机核判据与必要的并发/集成测试证据。
3. 处理 `NEW-R05-03`（V 编号措辞）与 `NEW-R07-04`（陈旧哈希）。
4. ADR-0031 的 `NewRuntime` 构造契约由决策方拍板；若登记，附 `r10` 落定后的 ADR 哈希与一致性说明。
5. `F017` 的写集更新由 Task-Planner 处理；复审只确认未被擅改。

## 9. 写入清单与只读声明

- 本次写入：本报告 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r07/report.md`；同目录 `hashes.txt`（记录本报告与基准哈希）。
- 未修改任何被评对象、既有报告、任务卡、ADR、代码、测试；未删除/移动/归档；未执行 Git 写操作。
- 所有结论基于 2026-11-02 工作树实测；未实测项已标 **NOT RUN**。
