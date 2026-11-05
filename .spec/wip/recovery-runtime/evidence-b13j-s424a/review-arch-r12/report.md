# identity-auth 详细设计 r72 架构独立复审（review-arch-r12）

- 日期：2026-11-02
- objectType：design（identity-auth 详细设计；非 ADR）
- 受审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
  - 开始与结束两次实测：1,110,477 B / 5,476 行 / CR=169 / LF=5,476 / 无 BOM / 末字节 `0A` / SHA256 `491A476E60636ADAA3F1610A07C78054ED4C9735F5AD42D60B76200A12C8414F`，revision `identity-auth-design-ir01-r72`（`:5`、`:38`）
  - 我的冻结副本：`%TEMP%\design-r72-freeze-491A476E.md`（同字节／同哈希；首末复算一致，**未见漂移**）
- r71 基线：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r12/design.r71.baseline.md`，1,106,644 B / 5,476 行 / `168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5`（自算；该值等于 r11 轮冻结的 r71 终值，真实性成立）
- 基准复审报告（本路上一次判定）：`review-arch-r11/report.md`，21,990 B / 194 行 / `3D994EC464D5798592C95625C1B8E59C1FAA7338BC0B7C831C9BAF2A068262F2`（Verdict: CHANGES REQUIRED，BLOCKER 0 / MEDIUM 1 / LOW 1 / 范围外 1）
- 安全侧对照（不作本路依据）：`review-sec-r11/report.md`，26,758 B / 228 行 / `D715E4116CAE3EDEAA6785A6E1FCF4BD5A15E07E036CA78923976A7B9D23FAFB`（PASS WITH CONDITIONS，1 LOW `NEW-S11-01`）
- 作者证据（只作线索，数值全部自算）：`design-revision-r12/report.md` 13,295 B / `677854EE758D1AFD680E81753817E977BBFAA342A67E3277F7FE3E367029A65A`；`design-revision-r12/hashes.txt` 4,662 B / `F1A4112240FE22EE7684EE64CF1A7479945D90B93EDD3D57B543C560A231C146`
- ADR 对齐基准（只读）：`.add/project/knowledge/decisions/framework.md`（ADR-0031 `adr-0031-r12`）297,412 B / 1,275 行 / `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`，判读 `:1118`

## 方法

按 `add-code-review`（objectType=design）两轮执行；维度按 `code-review-checklist`（正确性与契约／安全与 fail-closed／架构与抽象／可观测与配置），抽象判断按 `abstraction-principles`（所有权唯一、失败收敛、单一真相源）。代码提交／构建类条款不强套设计文本。

两轮实际检查范围：

- Round 1（改动边界与直接缺陷）：对象身份与编码（字节／行／CR／LF／BOM／末字节）、r71 基线真实性、全文逐行 `-cne` diff、逐行 UTF-8 字节账、12 行改动逐行原文通读（`:5`、`:38`、`:3944`、`:4031`、`:4051`、`:4062`、`:4070`、`:4072`、`:4083`、`:4133`、`:4135`、`:4377`）、四项改动（r72.1–r72.5）的落点事实、全角括号平衡、`若该键存在` 命中语义、版本串与优先级链计数。
- Round 2（契约、状态、分类与回归）：`framework.md:1118`（ADR-0031 r12）逐项对齐；`NEW-R11-01`／`NEW-R11-02` 闭合判定；`:4133` 与 canonical `:3944`／`:4377` 的一致性；V／R 编号不变性；r71 与更早已闭合项无回退逐项核对；golden vector `:4099`／`:4105` 独立复算；三段后缀 raw bytes 复算；`internal/recovery` 侧现状（消费者可行性，只读）；`tasks.md:3186` 状态核对。

只读声明：本轮未修改 `design.md`、`framework.md`、`tasks.md`、`decisions/README.md`、`cmd/**`、`internal/**`、测试、任何 `review-*` 与 `design-revision-*`；未删除／移动／归档；无任何 Git 写操作；未 build／test／E2E；未启动其他 Agent。判读过的文件：`design.md`（全文按行 diff + 指定行通读）、`design.r71.baseline.md`（逐行比较）、`framework.md`（`:1118`）、`tasks.md`（`:3186`）、`review-arch-r11/report.md`、`review-sec-r11/report.md`、`design-revision-r12/report.md` 与 `hashes.txt`、`design-revision-r11/hashes.txt`（后缀登记对照）、`internal/recovery/admission.go`（`:200`–`:262`，只读）、`.codex/skills/add-code-review/SKILL.md`、`.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`。唯一写入为本目录 `report.md` 与 `hashes.txt`。

## 一、对象与字节账（全部自算）

| 项 | 结果 |
|---|---|
| 字节／行 | 1,110,477 B / 5,476 行；r71 为 1,106,644 B / 5,476 行 → **0 行净增减**，Δ = **+3,833 B** |
| 编码 | CR=169、LF=5,476、无 BOM（首三字节 `2D 2D 2D`）、末字节 `0A`；与 r71 持平 |
| 内容变更行 | 恰 12 行：`:5`、`:38`、`:3944`、`:4031`、`:4051`、`:4062`、`:4070`、`:4072`、`:4083`、`:4133`、`:4135`、`:4377`；其余 5,464 行逐字节相同（无第 13 行） |
| 逐行字节账 | `:5` 39→39（0）、`:38` 50→50（0）、`:3944` 1,370→1,803（+433）、`:4031` 2,006→2,316（+310）、`:4051` 1,365→1,915（+550）、`:4062` 2,655→2,962（+307）、`:4070` 5,858→6,077（+219）、`:4072` 8,952→9,904（+952）、`:4083` 681→766（+85）、`:4133` 767→1,086（+319）、`:4135` 1,312→1,658（+346）、`:4377` 1,001→1,313（+312） |
| 合计 | 逐行之和 **+3,833 B** = 全文增量 1,110,477 − 1,106,644，**自洽**；与作者第 2 节表逐格一致 |
| 全角括号 | `（` = 1,258、`）` = 1,258，**平衡**（r71 为 1,237／1,237；r72 新增 21 对） |
| 版本串 | `identity-auth-design-ir01-r72` = 3（`:5`／`:38`／`:4072`）；`identity-auth-design-ir01-r71` = 1（`:4072` r71 历史登记，属历史事实） |
| 优先级链 | `r72 > r71 > … > r61` = 3（`:4072` 既有两处 + r72 登记段一处）；反引号旧链 `` `r71 > r70 `` = 0 |

## 二、原 finding 逐条判定

### `NEW-R11-01`（MEDIUM）：policy `tool.receipt.trust_anchor_ref` 缺键／空串／字段名不符的错误类未写死 —— **CLOSED**

判据（逐点核对原文，非按作者落点表）：

| 落点 | 现文事实 | 判定 |
|---|---|---|
| `:3944` | canonical 映射尾部增「**唯一 policy 专属例外（r72.1）**」：该字段缺键／空串／名不符**不在**本映射「缺键 → `ErrAdmissionIncomplete`」之内，按启动前 `ErrAdmissionBinding`、已启动后 `restore_unknown`；**只限此字段**、不得扩展到任何其他缺键；并声明不改变 fail-closed 与「任一非 nil 都在 Begin 前返回」 | 一致 |
| `:4031` | policy 侧失败显式给出两类（启动前 `ErrAdmissionBinding`／已启动后 `restore_unknown`），并声明其为 `:3944`／`:4377` 通用「缺键／缺字段 → `ErrAdmissionIncomplete`」的**唯一 policy 专属例外**，只限 `tool.receipt.trust_anchor_ref` | 一致 |
| `:4062` | 同一口径（五处等式链行），无第二套读法 | 一致 |
| `:4070` | B1 摘要错误分类段追加「**唯一例外（r72.1）**」，其余检查点仍按 `:4377` 原因分类 | 一致 |
| `:4377` | canonical 尾部增「r72.1 增补唯一例外」，明确该字段**不按**本段「缺键 → `ErrAdmissionIncomplete`」 | 一致 |
| `:4133` | `revalidateApproval` 侧同步引用 r72.1 唯一例外（启动前 `ErrAdmissionBinding`／已启动后 `restore_unknown`） | 一致（额外收敛，见下） |

- 与 ADR 对齐：`framework.md:1118` 实读为「policy 的 `trust_anchor_ref` 是 design 规定的 policy 必填字段，缺键、空串或字段名不符即 closed——启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`；不保留「policy 该键可缺」的条件分支」。design 现文与该句逐要素一致（含「不保留可缺分支」与此前那句的唯一例外范围）。
- 机核性：全文对 `tool.receipt.trust_anchor_ref` 的 7 处命中（`:3944`／`:4031`／`:4062`／`:4070`／`:4072`／`:4133`／`:4377`）均指向同一分类，没有一处把它写成 `ErrAdmissionIncomplete`；`缺键` 全文其余命中（`:1259`／`:2628`／`:4856`／`:4858`／`:5179`／`:5180`）分属秘密键／ENV／建议值域，与恢复批准链无关。
- 结论：**CLOSED**（设计文本层，可唯一机核）。原 r11 指出的「同一输入三种闭类」已消失。

### `NEW-R11-02`（LOW）：`:4135` 读取点枚举未随 r71.3 同步 V7／V8 —— **CLOSED**

- `:4135` 现新增「其七（r72.2）」：`V7` 独立读取回执前与 `V8` 验证 producer 签名前各再一次 `revalidateApproval` 类重验；标注与 `:4037`／`:4062`／`:4070` **同登记**、「不新增、不重排、不复用 V 编号」；并写明两点位于工具已启动后，失败按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown` 处置。
- 一致性：`:4070` 声明「重验点完整集合以 `:4037`／`:4062`／`:4135`／`:4137` 为准」，`:4135` 现与 `:4037`／`:4062` 的 V7／V8 重验语义逐项一致；四行并读不再依赖外部补丁。
- 编号不变性：`V12` 全文 0 次、`R9` 0 次；`:4037`（V1–V11 定义行）本轮**逐字节未改**。
- 结论：**CLOSED**。

### 跨轴 `NEW-S11-01`（LOW，安全侧，仅文字层复核）

- `:4133` 已三分：预检后字节（含截断）／`generation`／inode／`st_dev`／挂载漂移、未到 `valid_from`、已过 `valid_until`、`revoked` → `ErrAdmissionClosed`；**文件缺失、不可读、变为非 regular 或被符号链接替换 → `ErrAdmissionIncomplete`**；同一快照内格式／签名／字段不一致 → `ErrAdmissionBinding`。
- 与 `:3944`／`:4377` 及 ADR `:1118` 的「缺键、文件缺失或不可读 → `ErrAdmissionIncomplete`；漂移 → `ErrAdmissionClosed`」一致；旧句「同样 `ErrAdmissionClosed`」已不可检索。
- 判定：**文字层 CLOSED**；是否接受由安全侧给出（本路不作安全结论，不以本次复核替代）。

## 三、`:4051`（r72.4）累积写集澄清的越权评估

- 现文性质未变：`:4051` 仍自述「**只作建议，不是任务卡；本轮不改 `tasks.md` 任何一行**」「该表仍只作建议，不改 `tasks.md`、不改卡面、不勾选；写集更新属 Task-Planner，写集更新前保持不可实施」。
- r72.4 追加的只是一句口径澄清：「r65.12／r66.4 只局部取代其所列职责写法，不废止其余增补项」，并列出 r63.9.4 表内 6 个生产文件与配套测试「供 `F017-R1` 写集扩写使用；**是否并入本卡仍由 Task-Planner 决定**」。
- 事实核对：所列 6 个文件确实在 r63.9.4 表内（`recovery_restore_approval.go` `:4005`、`recovery_restore_target.go` `:4006`、`recovery_restore_receipt.go` `:4011`、`recovery_restore_witness.go` `:4012`、`recovery_config.go` `:4015`、`internal/recovery/restore_source.go` `:4016`）；无编造文件。
- 越权判定：**不越权**。它不授予实施、不改卡面、不勾选、不改变任何门禁；只是对 r65.12「写集以 r65.12 为准」的范围作解释性限定（在该句只描述局部职责改写的语境下），并受 `r72 > r65` 优先级支持，自洽。
- F017 状态：**未改变**。`:4070`(7)「F017（`tasks.md` 写集缺口）**待 Task-Planner 对账**」、`:4021`(2)、`:4053`(8)、`:4081`（r66.9）均未变；`tasks.md` 全文未被改动（904,893 B / 3,312 行 / `696A5B0EE9B2E4DA75D4C50EB82BCFC0E68D657500BDED346F4CA765822BD28E`，与本路上轮实测同值），`:3186` 仍为 `- [ ]`。

## 四、无回退核对（依据：12 行之外逐字节相同 + 变更行未削弱）

| 底线 | 证据 | 结论 |
|---|---|---|
| fail-closed 全路径 | `:3944`／`:4377` 映射保持 closed；`:4070`「不得当成功」 | 未削弱 |
| 任一非 nil 都在 Begin 前返回 | `:4377` 原句保留（全文 2 处，另一处 `:4021` 未变） | 未削弱 |
| 三签模型与三席位密钥来源 | `:3988`／`:4031`／`:4059`／`:4060`／`:4074` 未在变更集内 | 未回退 |
| `anchor_ref` 五处逐字节相等 | `:4031`／`:4062`（必填化加强，未放宽） | 未削弱 |
| `generation` 三元 | `:4373` 未在变更集内 | 未回退 |
| `receipt_anchors` 无 `revocation`／`valid_until` | 15 项 keyset 行（`:4060` 区段）未变 | 未回退 |
| no-follow／fd-exec | `O_NOFOLLOW` 10 次、`fd-exec` 23 次（与 r71 同） | 未回退 |
| `restore-v1`／`verify-v1` | `:4037` R5／V5 未变 | 未回退 |
| Begin 前 closed | `:4070` 路径句维持「只有启动预检与 `P3`／`P6` 在 `Journal.Begin` 之前」 | 未削弱 |
| `restore_unknown` 保守处置 | 全文 45 次（r71 为 38，+7 全部来自本轮新增分类句） | 未削弱 |
| Windows 生产 unsupported | `:4070` B6「退出码 2」未变 | 未回退 |
| 首期不提供生产 restore 工具与批准主体 | 全文 8 处，均未在变更集内 | 未回退 |
| 不声称抗 root | 3 处（含 `:4070`(2)）未变 | 未回退 |
| V1–V11 未增删重排 | `V12` = 0、`R9` = 0；`:4037` 逐字节未改；`:4135` 其七显式声明不新增／不重排／不复用 | 未回退 |
| r70 已闭合项（`NEW-R09-01`／`NEW-R09-02`／`NEW-S09-01`） | `:4076`／`:4079` 未在变更集内，逐字节同 r71；`:4072` 的 r70.1–r70.3 登记逐字保留 | 未回退 |
| r71 已闭合项 | `:4037` 未变；`:3944`／`:4377`／`:4070` 在原文基础上只作「加例外／加分类」，未删除 r71 的取代说明与路径更正 | 未回退 |
| golden vector 不变 | `:4099`／`:4105` 未在变更集内，独立复算值一致（见第五节） | 未回退 |
| 四类 unknown 仍为 unknown | `:4021`／`:4053`／`:4070` 相关段未削弱 | 未削弱 |
| `若该键存在` 语义 | 全文恰 2 处（`:4031`／`:4062`），均为「删除『若该键存在』条件分支，不保留『policy 该键可缺』的读法」的**否定引述**；旧条件短语「policy 中的 `anchor_ref`（若该键存在）」命中 0 | 不构成未闭合项 |

## 五、独立复算（golden vector 与三段后缀，raw bytes）

golden vector（取该行 UTF-8 字节、不含行终止符，SHA-256 小写）：

| 向量 | 行 | 实测字节 | 实测 SHA256 | 期望 | 结果 |
|---|---|---|---|---|---|
| source | `:4099` | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 同 | 一致（未变） |
| tool | `:4105` | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 同 | 一致（未变） |

三段后缀（方法：`[IO.File]::ReadAllBytes` 扫描 `0x0A` 求各行行首字节偏移，取该偏移至 EOF 的**原始字节**区间，不重编码、不归一换行；长度 = EOF − 起点，摘要 = 该区间 SHA-256）：

| 行 | startByte（r72） | 字节 | SHA256 | 与 r71 对比 |
|---|---|---|---|---|
| `:4259` | 826,545 | 283,932 | `b8135070edecb97470255d0dd505e93fd05e8d5197f8475dbd302832582ff4af` | r71：823,024 / 283,620 / `3798fb8f…` → 长度 **+312** |
| `:4275` | 828,825 | 281,652 | `26d4f46c6aa3c4a3d79bf8e472ce5bf95a4b80c596cff876fad776c0552d220c` | r71：825,304 / 281,340 / `dd8f8d16…` → **+312** |
| `:4290` | 831,810 | 278,667 | `58b55f08acf1af5fba85f38a1d65f6cf6399616dc989bfaf49a3d1d31d6a9b6a` | r71：828,289 / 278,355 / `32bdd8d0…` → **+312** |

- 自洽性：三段起点各 +3,521 = 全文 Δ 3,833 − 312，恰好由 `:4377`（位于三段区间内，+312 B）造成；其余 11 行改动均在 `:4259` 之前，只推起点、不改区间内容。与 design `:4072` 自述的条件式规则（「只有 `:4259`／`:4275`／`:4290` 行首之后被改写时后缀才变」）一致。
- **登记缺口**：r12 `hashes.txt` 无后缀节（检索 `4259`／`4275`／`4290`／`suffix`／`startByte` 均 0 命中），而 r71 `hashes.txt` 有 before／after 六项与 startByte。见 `NEW-R12-01`。

## 六、Findings

### [LOW] r72 改动了三段后缀内的字节，但 `design-revision-r12/hashes.txt` 未登记 r72 时点后缀值

- finding ID：`NEW-R12-01`
- 对象／修订：design `identity-auth-design-ir01-r72`，`491A476E…`；作者证据 `design-revision-r12/hashes.txt`
- 位置：`design.md:4072`（规则句「每次修订只对当次冻结副本复算，并把结果与取得时点写入该次修订的 `hashes.txt`；引用时必须同时给出「源文件 SHA-256 + 行号 + 取得时点」，不得把上一修订的复算值标为『现行』」）对照 `design-revision-r12/hashes.txt`
- 现状：r72 的 `:4377` 位于 `:4259`／`:4275`／`:4290` 三个后缀区间内（+312 B），故三段后缀现值已从 r71 登记的 283,620／281,340／278,355 变为 283,932／281,652／278,667（哈希全变）；但 r12 `hashes.txt` 只有 [1]–[9] 节，没有后缀节，也未记取得时点。
- 触发：后续评审或实施者按 `:4072` 规则引用「现行」后缀值时，只能取到 r71 的登记（`design-revision-r11/hashes.txt` 节 [5]），与 r72 实际值差 312 B 且摘要不符；若照抄即把上一修订值标成了「现行」。
- 影响：证据可追溯性缺口；不改变任何合同、错误分类或 fail-closed 行为，无安全影响。
- 建议：在 `design-revision-r12/hashes.txt` 增补后缀节（三行：`startByte`／字节／SHA256），并写明取得时点（2026-11-02，对象 SHA256 `491A476E…`）；或在 `:4072` 明示「本次修订未改后缀区间内容」——但本事实为**已改**，故只能补登记。
- 验证：按第五节方法对 r72 冻结副本复算，三项应与本报告值逐字相等。
- 类别参考：与 r68 的 `NEW-R09-02`（登记缺口）同类，严重级同为 LOW。

## 七、观察项（不计 finding）

1. `:3944` 同句并存「本条不另立分类」与「**唯一 policy 专属例外（r72.1）**」。按语义，例外被显式声明为「不在本映射之内」的范围剔除，不是第二套映射，故不构成矛盾；但字面并列易被误读为「既说不另立、又立了一类」，建议后续加半句「例外不在本映射范围」的括注。
2. policy 的 `trust_anchor_ref` **有值但与其他四处不等**这一情形，未在 `:4031`／`:4062` 的 policy 侧子句显式枚举（该句只列缺键／空串／字段名不符）。其闭类可由「五处必须逐字节相等」＋`:3944`／`:4377`「同一快照内字段不一致 → `ErrAdmissionBinding`」唯一推出（启动前 Binding／已启动后 unknown），与 policy 侧子句结果相同，故不构成分叉；建议补一句使分类出处显式。
3. `:4113`／`:4123`「平台无法读取／解析信任根 → `ErrAdmissionUnsupported`」与同行「缺失、不可读 → `ErrAdmissionIncomplete`」并存（沿用 r11 观察）：可按「平台能力」与「文件可读性」分域解释，字面仍易混；本轮未改。
4. `:4074`（r66.2）称 V7「只做长度与 JSON 语法校验」，r71.3／r72.2 在 V7 前加了重验（沿用 r11 观察）：因 `:4037`／`:4062`／`:4135` 均写「读取前」且优先级 `r72 > r66`，不构成冲突；建议加指针。
5. ADR 侧时点滞后（范围外）：`framework.md:1118` 仍写「现行 `identity-auth-design-ir01-r70` 未改这些合同……故本 ADR 不据此声称已复核 r67–r70 的全部改动」。设计已到 r72，且 r71／r72 改的正是这些合同（方向与 ADR r12 一致）；该句属 ADR 侧维护，不在本路写作范围，仅登记。

## 八、范围外登记（不纳入本轮 finding，不据此判 verdict）

- `N005`（ADR-0031 状态段与 `tasks.md` 状态段冲突）：`design.md:4070`(8)／`:4081` 仍登记为范围外，交架构决策方与 Task-Planner；本轮不改 ADR、不改 `tasks.md`。
- `F017`／`F017-R1`（写集缺口）：仍为 Task-Planner 对账项；`tasks.md` 未被改动、`:3186` 仍未勾选。
- `ADR-S06-01`（ADR 写集，安全侧 LOW）：设计侧 `:4133` 已收敛，ADR 侧措辞由 ADR 修订处置。
- `decisions/README.md`：未读取、未修改，不参与判定。

## 九、NOT RUN

- `go build`／`go vet`／`go test`／集成测试／E2E／性能／故障注入：**NOT RUN**。
- Linux host／ACL／mount／loader／fd-exec／maps／network／witness 实跑：**NOT RUN**；`//go:build linux` 的 skip 与无匹配测试 exit 0 都不等于 PASS。
- 真实 restore、真实审批主体、真实外部见证（HSM／审批系统）：**NOT RUN**；模拟见证仍是测试替身。
- 设计文本对实现的验证：`internal/recovery` 与 `cmd/**` 中检索 `trust_anchor_ref`／`TrustAnchorRef` **0 命中**，即 policy 必填字段与该例外在代码中尚不存在（与 `F017` 写集缺口一致），故 r72.1 的落地一致性与错误码传播属实现阶段待验，本路不能据设计文字推断 PASS。
- 本报告**不**构成生产 restore 解锁、不构成 `enter`／`resume` 解锁、不构成约束 8 满足、不构成 `identity-auth-S4.24a` 可关闭或可派发；任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。

## 十、实际执行命令（PowerShell 5.1，节选）

- `Get-FileHash -Algorithm SHA256 -LiteralPath`（design.md 首末各一次、快照、r71 基线、framework.md、tasks.md、两份基准报告、作者 report／hashes、r11 hashes）
- `[IO.File]::ReadAllBytes`（字节数、首三字节 BOM 判定、末字节、CR／LF 扫描、行首偏移与后缀区间）
- `[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)` + 逐行 `-cne` 比较 + `[Text.Encoding]::UTF8.GetByteCount`
- `[Security.Cryptography.SHA256]::Create().ComputeHash`（golden vector 行内字节、三段后缀区间）
- `Select-String`／`[regex]::Matches`（`若该键存在`、`tool.receipt.trust_anchor_ref`、`policy 专属例外`、`V12`、优先级链、全角括号）
- `rg -n "trust_anchor_ref|ErrAdmissionBinding|ErrAdmissionIncomplete" internal cmd`

## 计数

- BLOCKER：0
- MEDIUM：0
- LOW：1（`NEW-R12-01`，开放）
- 观察项（不计）：5（第 1–5 项）
- 范围外登记：`N005`、`F017`、`ADR-S06-01` 共 3 项

报告自身终值（自引用哈希无法内嵌）：字节 21,717／行 178／SHA256 见同目录 hashes.txt 首行登记。

Verdict: PASS WITH CONDITIONS