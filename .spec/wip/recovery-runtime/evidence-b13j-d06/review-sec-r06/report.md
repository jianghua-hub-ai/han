# ADR-0031「adr-0031-r12」安全独立复审报告（安全 r06）

## 0. 基本信息与只读声明

- 日期：2026-11-02（Asia/Shanghai）。
- 角色：add-reviewer（安全轴）｜objectType=adr｜判定层级：**决策接受级**（只判决策的安全充分性，不做实施验证、不判实施就绪）。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（`git rev-parse HEAD` 实测）。
- 只读声明：本轮**只**写入 `.spec\wip\recovery-runtime\evidence-b13j-d06\review-sec-r06\report.md` 与同目录 `hashes.txt`。未修改 `framework.md`、`design.md`、`tasks.md`、`decisions\README.md`、`internal\**`、测试或任何其他项目文件；未改动既有 `review-*`／`design-revision-*`／`adr-0031-r*` 目录；未删除、移动、归档任何文件；未执行任何 Git 写操作（仅 `git status`／`git ls-files`／`git rev-parse` 只读查询）。
- 判读依据清单（实际打开并逐行读取的）：见 §2.3。

## 1. 对象与冻结事实（自核，先于判读）

| 项 | 实测值 |
|---|---|
| 受审对象 | `.add\project\knowledge\decisions\framework.md` |
| 字节 / 行数 | **297,412 B / 1,275 行** |
| CR / LF / bare-LF | 1,275 / 1,275 / **0**（纯 CRLF） |
| BOM | 无（首 3 字节 `23 20 E6` = `# ` + U+4E2D 起首字节） |
| 末尾 | `0D 0A`（末尾 CRLF，无尾随空白行） |
| SHA256 | **`800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`** |
| 最后写入时间 | 2026-11-02T18:32:59.5015529+08:00 |
| 冻结快照 | `C:\Users\J00010385\AppData\Local\Temp\framework-adr0031-r12-freeze-800B5689.md`，297,412 B / **同哈希** → **无漂移** |
| r11 前像（用于 diff） | `%TEMP%\framework-adr0031-r11-freeze-9723F92F.md`，292,606 B / 1,274 行 / `9723F92FB53B076D228B5FA392CEC07CA0F74ED460524EBB7F00AC9202EB53CD`（实测一致） |
| ADR-0031 段范围 | `:1079`–`:1194`（`:1079` = `<a id=adr-0031></a>`；`:1078` 为空行；`<a id=adr-0032>` 起于 `:1198`） |

基准（本路上一次判定，必读全文）：

- `.spec\wip\recovery-runtime\evidence-b13j-d06\review-sec-r05\report.md`：**24,558 B / 188 行 / SHA256 `40F5C45DDCC77A1E786B9B1E1E64E03F7EA8DB1519CB22188E695658EB8DBF12`**；verdict = PASS，BLOCKER 0 / MEDIUM 0 / LOW 1（`ADR-S05-01`）。

交叉对象（旁证，非本对象写集）：

- `design.md`（`identity-auth-design-ir01-r70`）：1,103,152 B / 5,476 行 / `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`。
- `tasks.md`：904,142 B / 3,312 行 / `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C`。
- `decisions\README.md`：21,613 B / 110 行 / `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2`。
- `internal\recovery\runtime.go`：59,138 B / 1,969 行 / `4C0946AF7C29DE472CAE4848CE2F55A68288003071957FB62B66BFF3B59604C4`；**`git status --porcelain` = `?? internal/recovery/runtime.go`，`git ls-files --error-unmatch` 失败 → 该文件未被 HEAD 跟踪**，其行号仅作旁证，不构成本 ADR 的规范依据。
- 架构侧上轮：`review-arch-r05\report.md` 27,698 B / `98C3DD6F6670BB5F875B80AD084C9A0130F4956220206C38AE0B8DD938271FB7`（与 `:1194` 引述一致）。
- 作者证据（**只作线索，不作证据**）：`adr-0031-r12\report.md` 5,636 B / `654BBD6A08158C2228310F56EE02B58DD6388DECAC6A7A6C4DA2201691B28618`；`adr-0031-r12\hashes.txt` 2,180 B / `E7E541D2327654AF40CBE1F77F424E7B1FE25B8649C587C14231F236414EA01D`。

## 2. 方法、两轮检查范围

### 2.1 方法

- 完整读取 `.codex\skills\add-code-review\SKILL.md`，按 `objectType=adr` 分支消费：查对象 → 对照基准报告逐条判闭合 → 沿真实消费者与上游依据（design 条款、代码行号、作者证据）有界交叉。
- 必读规则：`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\skills\add-code-review\references\产品安全红线.md`；ADR 维度按 `decisions-guide.md` 口径（决策级、可复算、无自我解锁、残余显式登记）。
- 冲突消解口径：design 内以作者登记的修订优先级（r66.0 唯一优先级规则）为准；ADR 与 design 冲突时，先判「是否削弱 fail-closed 与拒绝强度」，再判「是否只是错误码/措辞分叉」。

### 2.2 两轮实际检查范围

- 第一轮（对象自身）：字节/行/CRLF/BOM/首末字节/哈希自核 → 与 r11 前像做逐行与逐字节 diff → 独立复算前缀/段/后缀字节账 → 定位 `:1082`／`:1084`／`:1118`／`:1174`／`:1194` 的实际差异文本。
- 第二轮（消费者与上游）：`design.md` 逐条核对（`:1537`／`:3672`／`:3886`／`:3899`／`:3942`／`:3944`／`:4037`／`:4062`／`:4113`／`:4123`／`:4135`／`:4373`／`:4377`）→ `internal\recovery\runtime.go` 调用顺序旁证（`:1607`／`:1617`／`:1634`／`:1656`／`:1666`，并核 `Restore.Replace` 全文仅 1 处）→ 检索断言复核（`r68.`／`r69.`／`r70.`／`未命中任何编号条款`／`若该键存在`）→ 无回退底线逐项计数与逐行比对 → 作者证据的数字/断言一致性核对。

### 2.3 判读过的文件

`framework.md`（`:1079`–`:1194` 全文逐行 + 段外哈希/前缀/后缀字节比对）、`framework` r11 前像、`design.md`（上列条款原文）、`tasks.md`（`:3186`、`:3220`–`:3222`）、`decisions\README.md`（`:39`）、`internal\recovery\runtime.go`（调用顺序与 `Replace` 调用点检索）、`review-sec-r05\report.md`（全文）、`review-arch-r05\report.md`（评审范围与 finding 标题）、`adr-0031-r12\report.md` 与 `hashes.txt`（线索核对）、`add-code-review/SKILL.md`、`code-review-checklist.md`、`abstraction-principles.md`、`产品安全红线.md`、`decisions-guide.md`。

## 3. 改动范围与字节账（独立复算）

### 3.1 段外不变

- 前缀（文件起至 `## ADR-0031` 之前）：r11 与 r12 均 **205,357 B**，逐字节相同（`-ceq` 比对 True）。
- 后缀（`<a id=adr-0032>` 起至 EOF）：r11 与 r12 均 **21,165 B**，逐字节相同（True）。
- 段内：r11 66,084 B → r12 70,890 B，净 **+4,806 B**；205,357 + 70,890 + 21,165 = 297,412 = 实测总字节，**账自洽**。

### 3.2 行级 diff（1–1,193 区段）

- 逐行 `-cne` 比对结果：仅 `:1082`、`:1084`、`:1118`、`:1174` 四行不同。
- `:1193`（r11 追溯行，2,845 字符）**逐字符未改**（True）。
- 新增 `:1194`（2,403 字符）。旧 `:1194`–`:1274` 与新 `:1195`–`:1275` 比对 **0 处不一致**。
- 结论：**内容改动 4 行 + 新增 1 行，0 行净增删外的其他改动**；与 Lead 复算及作者 `:1194` 自述一致（`+4,806 B`，非作者报告口径的其它数值）。

### 3.3 五行差异的实际语义

- `:1082`：修订号 `adr-0031-r11` → `adr-0031-r12`（单字符）。
- `:1084`：状态段仅把两处 `adr-0031-r11` 改为 `adr-0031-r12`，其余文字（含「已接受（2026-11-01 用户 D03 指令授权…）」「不构成人的成果接收、独立评审、安全评审、生产装配批准或真实工具/备份验证」「**本修订 `adr-0031-r12` 的两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动**」「本 ADR 的任何修订均不改变『首期不提供生产 restore 工具与生产批准主体』」）保留。
- `:1118`：中段整段替换（共同前缀 4,526 字符、共同后缀 442 字符；5,887 → 6,409 字符）。替换内容见 §4／§6。
- `:1174`：行末追加 r12 范围句（1,581 → 1,704 字符）。追加内容明写「未改不变量 1–6、约束 1–9（含解除条件九条）任何字节，未补齐 unknown，未改变结论与候选取舍」。
- `:1194`：r12 追溯行（触发来源、授权、逐项改动、未改段落、改前基线、未取得两路接受、四项 unknown 仍为 unknown、`enter`／`resume` 保持关闭、S4.24a 不得启动）。

## 4. `ADR-S05-01` 闭合判定：**CLOSED**

- 原 finding（r05）：`P6`／`Prepare` 的读取失败统一写 `ErrAdmissionClosed`，与 design `:4377`（缺键／不可读 → `ErrAdmissionIncomplete`）及 `:4113`／`:4123` 冲突；且 design `:3944` 本身给出相反映射。定级 LOW 的理由是「所有分支均 fail-closed，无『失败当成功』读法」。
- r12 现状（`:1118` 原文，节录）：「任一重验失败按路径与原因分写后果：`P6` 与 `Prepare` 内的重读按 design `:4377` 的原因分类——缺键、文件缺失或不可读、非绝对规范路径、owner／mode 不符、非 regular 或被符号链接替换等返回 `ErrAdmissionIncomplete`；未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked`，以及预检通过后的字节／`generation` 漂移，返回 `ErrAdmissionClosed`。」
- 对照上游：design `:4377`（r61.2 + 行内 r62.3 更正）确为同一分类（缺键/不可读/etc → Incomplete；`valid_from`/`valid_until`/`revoked` → Closed；预检后漂移 → Closed）；design `:4113`（r62.1）、`:4123`（r62.2）同向。
- 判定：原诉求「不再把 `P6`／`Prepare` 的读取失败一律写成 `ErrAdmissionClosed`」**已实现**，且新写法的失败集合是原写法的超集拆分，**拒绝强度未降低**（两条分支都是非 nil 拒绝）。→ **`ADR-S05-01` CLOSED**。
- 残余：r12 以 `:4377` 为唯一依据，未登记 design `:3944`（r63.3）的相反映射 → 新 LOW `ADR-S06-01`（§8），不改变本条闭合判定。

## 5. 携带项（不计入本对象阻断计数）

- `ADR-R05-01`（架构侧 MEDIUM；`resume` 无新 `Journal.Begin`、不调用 `Replace`，`R`／`V` 路径归属）：**CLOSED（安全侧独立确认）**。`:1118` 现文给出「`enter` 的 `P6` 失败不进入 `Journal.Begin`；`resume` 不新 `Journal.Begin`（design `:3672`／`:3886`／`:3899`）」，「`R1`–`R8` 只在 `enter` 的 `Replace` 路径出现，且位于 `Journal.Begin`（`internal/recovery/runtime.go:1617`）之后」，「`V1`–`V11` … `enter`（`runtime.go:1634`）与 `resume`（`runtime.go:1666`）均会调用——`resume` 的调用在 `Prepare`（`runtime.go:1656`）之后且不伴随新的 `Journal.Begin`」，并明写「`resume` 不调用 `Replace`」。旁证核验：`runtime.go` 中 `Prepare` 在 `:1607`、`Journal.Begin` 在 `:1617`、`resume` 的 `Prepare` 在 `:1656`、`VerifyRestored` 在 `:1666`；全文 `Restore.Replace` 仅 1 处（`enter` 路径）。design `:3672`／`:3886`／`:3899` 均写「不新 Begin／不再次 Replace」。安全后果：`resume` 的失败路径不产生新的替换写入，也不会把既有 `restore_unknown` 转成成功。
- `ADR-R05-02`（架构侧 LOW；`P6`／`Prepare` 错误码）：**CLOSED**（与 `ADR-S05-01` 同源、同一处修复）。
- `ADR-R05-03`（架构侧 LOW；检索断言不准确）：**CLOSED（独立复核）**。`:1118` 现文断言「全文检索 `r68.`／`r69.`／`r70.` 仅命中 `:4072` 的行号映射说明（列出 `r68.1`–`r68.8`／`r69.1`–`r69.3`／`r70.1`–`r70.3` …，不构成合同条款）与 `:4083` 的『r70 修订沿用本声明』」。实测：design 全文 `r68.`／`r69.`／`r70.` 命中**仅 `:4072` 一行**（各编号各 1 次）；`:4083` 只含 `r70 `（带空格）而无 `r70.` token → 新断言**准确**；旧断言「未命中任何编号条款」已被删除（仅 `:1193` 历史行保留 1 处，按口令约定不作现行判据）。
- `ADR-S04-01`（design 侧 LOW，携带）：`design.md` `:4031`／`:4062` 仍写「（若该键存在）」。实测 design 全文命中 `若该键存在` 2 处，正是 `:4031`、`:4062`；`framework.md` 全文命中 **0** 处（ADR 侧无此弱化分支）。design 哈希与 r70 一致（未变）→ **携带项仍成立**，属 design 写集，**不计入本对象阻断计数**。
- design 内部相反映射（观察项，非本对象写集）：`:3944`（r63.3）「批准文件、信任根或目标 DSN 不可读 → `ErrAdmissionClosed`；回执独立读取路径或审计投影不可读 → `ErrAdmissionIncomplete`；**两者的区别不得互换**」 vs `:4377`／`:4113`／`:4123` 的「不可读 → Incomplete」。该项落入 `ADR-S06-01` 的根因，建议由 design 侧消解。

## 6. fail-closed 逐路径复核（安全关键）

判定口径：任一路径的失败必须在「产生不可逆写入 / 推进见证 `sequence` / 返回成功」之前以非 nil 错误终止；「越过工具启动点」后必须是 `restore_unknown` 而不许降级。

| 路径 / 时点 | 依据（行号） | 失败分类 | Begin 前？ | 是否可能写入 | sequence | 结论 |
|---|---|---|---|---|---|---|
| `P3` 预检 | `:1118`（「`P3` 只读取…构造不可变快照、完成首次合取（无前像可比，故不作 generation 漂移判定，也不属重验点）」）；design `:3942` P3 | 按 design `:4377` 分类 | 是（解析入口） | 否（只读 + 快照） | 不推进 | fail-closed 成立 |
| `P6`（`enter`） | `:1118`；design `:4135` 其二（`Prepare` 在 `Journal.Begin` 之前调用 `revalidateApproval`）；`runtime.go:1607`→`:1617` | Incomplete／Closed | **是，且明写「不进入 `Journal.Begin`」** | 否 | 不推进 | fail-closed 成立 |
| `Prepare`（`enter`） | 同上 | Incomplete／Closed | 是（`:1617` 之前） | 否 | 不推进 | 成立（建议 S-1：显式写出） |
| `Prepare`（`resume`） | `:1118`（不新 `Journal.Begin`）；design `:3672`／`:3886`／`:3899`；`runtime.go:1656` | Incomplete／Closed | 无新 Begin（既有状态不推进） | 否 | 不推进 | 成立 |
| `R1`／`R2`／`R4`（`enter` 的 `Replace`） | `:1118`；`runtime.go:1584`（唯一 `Replace`）、`:1617` | 未启动子进程前：Incomplete／Closed | 在 Begin 后但**在工具启动点前** | 否（替换未开始） | 不推进 | 成立 |
| `V1`／`V2`／`V4` | `:1118`；`:1634`／`:1666` | 同上 | 同上 | 否 | 不推进 | 成立 |
| `R6`／`V6`／`V7`／`V8`／`V10`（越过工具启动点） | `:1118`（「已越过时无论原因为何一律按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)` 处理、由调用方按 `restore_unknown` 处置（`R6`／`V6` 以及 `V7`／`V8`／`V10` 前的重验均属此列）；任何情况下**不得推进见证 `sequence`**、不得当成功」） | `restore_unknown` | 否（已启动） | 可能已发生，故必须 unknown | **明写不得推进** | 成立且为强约束 |
| 全局 | `:1118`（「本节…只增加拒绝点、不放宽任何既有检查」）；design `:3944`（「不存在『先返回成功再异步核验』的分支」） | — | — | — | — | 无「先成功后异步核验」分支 |

重点核对 r11→r12 的语义变化是否削弱 fail-closed：

- r11 曾写「`P6` 及 `Prepare` 内的重读在 `Journal.Begin` 之前失败返回 `ErrAdmissionClosed` 且不进入 `Journal.Begin`」。r12 把它拆为 Incomplete／Closed **两类**。两类均为非 nil 拒绝、均在 `Journal.Begin` 前返回、均不写入、均不推进 `sequence` → **拒绝强度不降低**，变化仅在错误码分叉。
- 错误码分叉是否产生外部可观测差异？design `:1537` 明写「依赖错误保留内部 cause，公共已识别业务只映射现有 `platform.recovery_in_progress`，不新增 REST 错误」→ **无外部可观测差异**，仅影响内部负向测试期望与审计 `result` 枚举（`verified`／`unknown`／`closed`／`incomplete`，design `:3944`）。这也是 `ADR-S06-01` 仅定 LOW 的依据。
- `resume` 是否可能把既有 `restore_unknown` 转成成功？`:1118` 要求 `resume` 在 `Prepare` 后调用 `VerifyRestored`，任何 R/V 失败按路径分类处置，越过启动点一律 `restore_unknown`；无「先置成功」路径 → **不会**。
- `restore_unknown` 的保守语义：约束 5（`:1157`）逐字节未改（§7）。

**结论：fail-closed 语义未被 r12 削弱，逐路径均有明确拒绝点与写入边界。**

## 7. 无回退底线逐项核对

行级依据：§3.2 证明 `:1079`–`:1193` 内仅 `:1082`／`:1084`／`:1118`／`:1174` 变化，因此其余各行（含不变量、约束、解除条件）**逐字节等于 r11**。

| 底线项 | 位置 | 核对结果 |
|---|---|---|
| 不变量 1–6 | `:1133`–`:1149`（含 `:1139`–`:1145` POSIX/Windows 矩阵、`:1149`「生产 restore closed」） | 未改（diff 证明） |
| 约束 1–9（含解除条件九条 `:1161`–`:1169`） | `:1153`–`:1170` | 未改（diff 证明） |
| 「首期不提供生产 restore 工具与生产批准主体」结论 | 段内 11 处；`:1084` 末句明写任何修订均不改变 | 保留 |
| `anchor_ref` 五处等式 + policy 必填 `tool.receipt.trust_anchor_ref`（缺键／空串／名不符即 closed） | `:1118` 同段「`anchor_ref` 五处等式（r10 定名）」 | 未改；「五处逐字节相等」1 处、键名 4 处、明写「不保留『policy 该键可缺』的条件分支」；`framework.md` 全文 `若该键存在` = **0** |
| 三签模型与三席位密钥来源分属 | `:1118` 三签段（producer／verifier／witness，公钥分别取 `receipt_anchors.public_key`／`keys[verifier_key_id].public_key`／`receipt_anchors.witness_public_key`；`witness_key_id` 非回执签名公钥唯一来源） | 未改 |
| `generation` 三元摘要定义 | `:1118`「三项同时逐字节相等」 | 未改 |
| 无条件「不声称抗 root」 | `:1118`「无条件残余（r10）」；段内 6 处 | 未改 |
| 两种「见证」分域（vs ADR-0030 `(w, r, token)`） | `:1118` 末段「见证术语消歧（r10）」 | 未改 |
| 验签信任锚固定于批准记录 | `:1118`「验签信任锚…固定登记在批准记录内」 | 未改 |
| `restore-v1`／`verify-v1` | 段内各 4 处 | 保留 |
| no-follow／fd-exec／最小 ENV（含禁 `LD_*`／`*_PRELOAD` 等） | `:1141`–`:1144`；段内 14／9／9 处 | 未改 |
| Begin 前 closed | `:1118`（`enter` 的 `P6` 失败不进入 `Journal.Begin`）；`:1145`（「任一隔离预检失败在 Begin 前拒绝」） | 保留且更明确 |
| `restore_unknown` | 段内 19 处 | 保留 |
| Windows 生产 unsupported | `:1139`–`:1145`；段内 9 处 | 保留 |
| 四项 unknown 仍为 unknown | `:1084`、`:1194` 均明写 | 保留 |
| 无自我解锁 | `:1084`「本修订 `adr-0031-r12` 的两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动」；`:1194`「未取得架构与安全两路独立评审接受…生产 `enter`／`resume` 的数据库替换保持关闭」 | 保留；r12 未声称任何解锁 |

**结论：无回退。**

## 8. 新 findings

### `ADR-S06-01`（LOW，ADR 侧）— `:1118` 对「不可读」的错误码引用与 design `:3944` 相反且未登记

- 位置：`framework.md:1118`（「`P6` 与 `Prepare` 内的重读按 design `:4377` 的原因分类——缺键、文件缺失或不可读…返回 `ErrAdmissionIncomplete`」）。
- 违反依据（冲突原文）：
  - design `:3944`（r63.3）：「读取失败分类固定为：批准文件、信任根或目标 DSN 不可读 → `ErrAdmissionClosed`；回执独立读取路径或审计投影不可读 → `ErrAdmissionIncomplete`；**两者的区别不得互换**。」
  - design `:4377`（r61.2，含 r62.3 行内更正）：「缺键、文件缺失或不可读…→ `ErrAdmissionIncomplete`」。
  - design `:4113`（r62.1）、`:4123`（r62.2）：同向写 Incomplete。
  - design `:3942` 明示该段「唯一调用图（取代 r62.3 的读取点段与 r62.6.9 的 `VerifyRestored` 八步）」，即 `:3944` 属较晚的规范段；按 r66.0 优先级，冲突时以较新修订为准 → 现行字面应为 `:3944` 的 Closed，而 r12 以 `:4377` 为唯一依据。
- 触发场景：实现方按 ADR `:1118` 实现，把「批准文件不可读」映射为 `ErrAdmissionIncomplete`，与 `:3944` 的 Closed 相反；后果表现为负向测试期望与审计 `result`（`incomplete` vs `closed`）不同。
- 影响评估（为何是 LOW 而非 MEDIUM/HIGH）：两条分支都是非 nil 拒绝、都在 `Journal.Begin` 之前、都不写入、都不推进见证 `sequence`；公共面只映射 `platform.recovery_in_progress`（design `:1537`），外部不可观测；不存在「失败当成功」或「先成功再核验」读法。→ **安全影响为零，仅错误码/审计枚举/测试期望分叉**。
- 修复建议（供作者决定，不代改）：在 `:1118` 增加一句登记 design `:3944` 的相反映射，并声明「以 `:4377` 为准、由 design 侧在后续修订消除 r63.3 与 r61.2／r62.1／r62.2 的内部冲突」；或直接把「不可读」按 `:3944` 写作 `ErrAdmissionClosed`。
- 证据：上述五处原文、design 哈希（r70，未变）、`framework.md` r12 快照 `:1118` 原文。

**本轮未发现 BLOCKER／HIGH／MEDIUM。**

## 9. 建议（非阻断，与缺陷分开）

- S-1：`:1118` 现只明写「`enter` 的 `P6` 失败不进入 `Journal.Begin`」，`enter` 的 `Prepare` 失败同样在 `Journal.Begin` 之前（design `:4135`；`runtime.go:1607`→`:1617`），建议并列写出，减少读者自行推断。
- S-2：design `:3944` 与 `:4377`／`:4113`／`:4123` 的「不可读」分类冲突应由 design 侧单点消解（design 写集，本轮不动）；ADR 侧只需登记取舍。
- S-3：`:1118` 引用的 `internal/recovery/runtime.go` 行号为**未跟踪工作树文件**（不在 HEAD）。建议注明「旁证、非规范依据」，避免后续被当作已提交合同引用。

## 10. 开放项、待决定与范围外登记

### 10.1 开放项（本对象）

| ID | 严重级 | 状态 | 一句话 |
|---|---|---|---|
| `ADR-S06-01` | LOW | OPEN | `:1118` 把「不可读」定为 `ErrAdmissionIncomplete` 与 design `:3944`「→ Closed、不得互换」相反且未登记；两分支均 fail-closed，安全影响为零。 |

### 10.2 携带项（不计入本对象计数）

| ID | 侧 | 状态 | 一句话 |
|---|---|---|---|
| `ADR-S04-01` | design | OPEN（携带） | design `:4031`／`:4062` 仍含「（若该键存在）」；design 哈希未变；属 design 写集。 |

### 10.3 待决定（交 Lead／作者，不代改）

- 是否在 `adr-0031-r13` 采纳 `ADR-S06-01` 的登记写法定稿，或改为按 `:3944` 写 Closed。
- 是否要求 design 侧新增修订消解 `:3944` 与 `:4377`／`:4113`／`:4123` 的冲突（该冲突影响负向测试与审计枚举口径，不影响 fail-closed）。

### 10.4 范围外登记（**不**据此判 CHANGES REQUIRED，不代改）

- `tasks.md:3220`：仍写「`adr-0031-r07` 为现行修订…决策接受级对 r07 成立，故本卡启动门禁解除」。与 `:1084`「本修订 `adr-0031-r12` 的两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动」直接冲突。后果：读者可能据此认为 S4.24a 门禁已解除——**该读法错误**，以 `:1084`／`:1174` 现行口径为准。
- `tasks.md:3186`：`- [ ] identity-auth-S4.24a …` 仍未勾选（与「不得启动」一致）。
- `decisions\README.md:39`：仍写「现行修订 adr-0031-r09（未复评…）」，落后于 r12。

## 11. 未运行项（NOT RUN）

- Linux 侧全部实测能力：POSIX 宿主隔离、专用执行身份、ACL/mount/loader、no-follow 稳定 fd 与 fd-exec、`/proc/*/maps` 校验、网络隔离、真实见证端点（DNS/TLS/pin）、进程树 join/reap 实测。
- 构建、单测、E2E、真实工具/旧备份演练、原生耐久回执真实样本。
- `internal/recovery/runtime.go` 的可编译性/可执行性验证（该文件未跟踪，仅按文本读取行号作旁证）。
- 结论：**凡涉及上述能力的安全结论一律 NOT RUN，不以文档文字判 PASS。**

## 12. 实际执行的命令（摘要，PowerShell 5.1，`;` 分隔）

- `Get-FileHash -Algorithm SHA256 -LiteralPath` + `Get-Item.Length` + `[IO.File]::ReadAllLines` 于 `framework.md`、r11/r12 快照、`design.md`、`tasks.md`、`decisions\README.md`、`internal\recovery\runtime.go`、基线报告与作者证据。
- `[IO.File]::ReadAllText` + `[Text.Encoding]::UTF8.GetByteCount(Substring(...))` 复算前缀/段/后缀字节；`IndexOf('## ADR-0031')`、`IndexOf('<a id=adr-0032>')` 定界。
- 逐行 `-cne` 比对 r11 与 r12 全文件（含 `1..1078` 前缀、`1194..EOF` 尾段、`1193` 行），统计差异行号。
- 正则/`Select-String -SimpleMatch` 统计段内关键字次数与 design／framework 全文 `r68.`／`r69.`／`r70.`／`未命中任何编号条款`／`若该键存在` 命中行。
- `git status --porcelain -- internal/recovery/runtime.go`、`git ls-files --error-unmatch`、`git rev-parse HEAD`（只读）。
- `New-Item -ItemType Directory` 创建本轮输出目录；`[IO.File]::WriteAllText`（UTF8 无 BOM）写本报告与 `hashes.txt`。

## 13. 计数与结论

- **BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 1**（`ADR-S06-01`，OPEN）。
- 携带项：design 侧 LOW 1（`ADR-S04-01`，不计入本对象计数）。
- 范围外登记：3 处（`tasks.md:3186`、`tasks.md:3220`、`decisions\README.md:39`）。
- 已判闭合：`ADR-S05-01`（CLOSED）、`ADR-R05-01`／`ADR-R05-02`／`ADR-R05-03`（CLOSED）。
- 本报告自述：自身字节数/行数/SHA256 无法在文件内固定（自指），由同目录 `hashes.txt` 记录写后实测值。

**边界声明**：本报告不声称生产已解锁、不声称 `enter`／`resume` 已解锁、不声称约束 8 已满足、不声称 `identity-auth-S4.24a` 可关闭或可派发实施、不声称抗 root 成立。本轮仅对 `adr-0031-r12` 给出决策接受级安全判断。

Verdict: PASS