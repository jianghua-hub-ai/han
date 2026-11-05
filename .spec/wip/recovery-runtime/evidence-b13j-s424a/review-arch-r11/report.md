# identity-auth 详细设计 r71 架构独立复审（review-arch-r11）

- 日期：2026-11-02
- objectType：design（非 ADR；与同会话的 ADR-0031 r06 复审是不同制品，未混用）
- 受审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
  - 实测：1,106,644 B / 5,476 行 / CR=169 / LF=5,476 / 无 BOM / 末尾 LF / SHA256 `168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5`
  - 修订号 `identity-auth-design-ir01-r71`（`:5`、`:38`），CR=169
- 我的冻结副本：`%TEMP%\design-r71-freeze-168B614D.md`（1,106,644 B / 同 SHA256，逐字节相同；现场对象与副本一致，**无漂移**）
- 写前基线：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r11/design.r70.baseline.md`，1,103,152 B / 5,476 行 / `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`（自算，与 Lead 声明一致；该值等于 r10 轮冻结的 r70 哈希，基线真实性成立）
- 基准复审报告（本路上一次判定）：`review-arch-r10/report.md`，16,624 B / `3E6639A5B566E3FAC644D4368F4CC33526163EB63DFF0F49343978413A30455A`，PASS（0 BLOCKER / 0 MEDIUM / 0 LOW，范围外 N005 1 项）
- 安全侧对照（不作本路依据）：`review-sec-r10/report.md`，23,423 B / `4D8625127035B84EEAF595BAFABC2666B9BFA64382FE43B45B1FF0D4DB9EF094`
- 作者证据（只作线索）：`design-revision-r11/report.md` 15,016 B / `8A3AEACB0A475875E57EC2EDDC81A801BCD6AB8DE5E78F5F5F26E45BCE57CF54`；`design-revision-r11/hashes.txt` 4,942 B / `5A493A856EEACF004D51D6494E30A183004C63EAD45577B0C8FF974F35E1AC55`
- 对齐基准：`.add/project/knowledge/decisions/framework.md`（ADR-0031 `adr-0031-r12`）297,412 B / 1,275 行 / `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`，只读判读 `:1118`

## 方法

按 `add-code-review`（objectType=design）两轮执行：Round 1 固定对象、复算字节账与逐行 diff、核对四项修正的落点事实；Round 2 沿真实依据（ADR-0031 r12 `framework.md:1118`、r70 基线、runtime.go 契约线、golden vector、后缀登记）核契约一致性、无回退与跨文档冲突。清单维度按 `code-review-checklist` 的“正确性与契约／安全与 fail-closed／架构与抽象／可观测与配置”，抽象判断按 `abstraction-principles`（所有权唯一、失败收敛、无重复真相源）。代码提交/构建类条款不强套设计文本。

两轮实际检查范围：
- Round 1：对象身份与编码（字节/行/CR/LF/BOM/末尾）、r70 基线真实性、全文逐行 diff（10 行）、逐行字节增量、四项修正的落点原文（`:3944`、`:4031`、`:4037`、`:4062`、`:4070`、`:4072`、`:4083`、`:4377`、`:5`、`:38`）。
- Round 2：`framework.md:1118`（r12）逐项对齐；无回退底线逐项（三签与密钥来源、五处等式、generation 三元、`receipt_anchors` keyset、restore-v1/verify-v1、no-follow/fd-exec、最小 ENV、Begin 前 closed、`restore_unknown`、Windows unsupported、首期不提供生产 restore 工具与批准主体、r70 已闭合项）；两条 golden vector 独立复算；三段后缀 raw bytes 复算并与作者 `hashes.txt` 交叉核；全角括号全文平衡；范围外登记核对（`tasks.md:3186`）。

只读声明：本轮未修改 `design.md`、`framework.md`、`tasks.md`、`decisions/README.md`、`cmd/**`、`internal/**`、测试、任何 `review-*` 与 `design-revision-*`；未删除/移动/归档；无任何 Git 写操作。判读过的文件：`design.md`（全文按行）、`design.r70.baseline.md`（全文按行 diff）、`framework.md`（`:1118` 与相关检索）、`tasks.md`（`:3186` 与元数据）、`review-arch-r10/report.md`（头部与结论）、`review-sec-r10/report.md`（仅取哈希）、`design-revision-r11/report.md` 与 `hashes.txt`、`.codex/skills/add-code-review/SKILL.md`、`.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`。唯一写入为本目录 `report.md` 与 `hashes.txt`。

## 一、对象固定与字节账（全部自算）

| 项 | 结果 |
|---|---|
| 行数 | 新旧均 5,476；**0 行净增减** |
| 内容变更行 | 恰 10 行：`:5`、`:38`、`:3944`、`:4031`、`:4037`、`:4062`、`:4070`、`:4072`、`:4083`、`:4377`；其余行逐字节相同 |
| 逐行 UTF-8 字节增量 | `:5` 0、`:38` 0、`:3944` +825、`:4031` +175、`:4037` +265、`:4062` +421、`:4070` +651、`:4072` +930、`:4083` +85、`:4377` +140 |
| 合计 | +3,492 B = 全文 1,106,644 − 1,103,152 = +3,492 B，**自洽** |
| 编码 | CR=169、LF=5,476、无 BOM、末字节 0x0A；与 r70 持平 |
| 三段后缀（raw bytes：行首偏移→EOF，不重编码、不归一） | `:4259` start=823,024 / 283,620 B / `3798fb8f7cc457b8743a9674ccebe6506c563f35c57975f7654eefd429c16358`；`:4275` start=825,304 / 281,340 B / `dd8f8d1642d562046186c17ec7810c1e8ed1f4709a86c1ef47f86a948e2f10ce`；`:4290` start=828,289 / 278,355 B / `32bdd8d002fefa0ee04ac02c443bd060430699f7ac992c448216d8d141354b41` |
| 与 r70 对比 | 三后缀各 **+140 B**（283,480→283,620；281,200→281,340；278,215→278,355），start 各 +3,352（= 3,492 − 140），与 `:4377` 位于三者之后且 +140 B 完全吻合 |
| 与作者 `hashes.txt` 节 [5] 交叉核 | 六项（before/after 三后缀）逐项一致；作者登记正确 |

## 二、四项修正逐条判定

### 1. policy `anchor_ref` 必填化（`:4031`、`:4062`）——**部分闭合（必填本体 CLOSED；错误类 OPEN）**
- 事实：两处均已由“批准记录（policy）中的 `anchor_ref`（若该键存在）”改为“批准记录（policy）**必填**的 `tool.receipt.trust_anchor_ref`”，并写明“policy 侧缺键、空串或字段名不符即不得放行（删除‘若该键存在’条件分支，不保留‘policy 该键可缺’的读法）”，`:4072` r71.1 同步登记。语义上已不可读成“可选”。
- **未闭合点**：两处只写“不得放行”，**未给出错误类**；同句的“其余四处任一不等，均 → 启动前 `ErrAdmissionBinding`”在语法上只约束“其余四处”。而本文件 canonical 映射 `:3944`／`:4377` 把“缺键／缺字段”判为 `ErrAdmissionIncomplete`，ADR-0031 r12 `framework.md:1118` 则明确“缺键、空串或字段名不符即 closed——启动前 **`ErrAdmissionBinding`**，工具已启动后 `restore_unknown`”。同一输入（policy 缺 `tool.receipt.trust_anchor_ref`）在两份现行文本下给出不同闭类，实现与测试无法唯一判定 → **NEW-R11-01（MEDIUM）**。
- 依据：`design.md:4031`、`:4062`、`:3944`、`:4377`；`framework.md:1118`。

### 2. `:4070`（r65.13 B1 摘要）按路径更正——**CLOSED**
- 原文已不再把 R/V 写成“不进入 `Journal.Begin`”。现文分层：只有启动预检与 `P3`／`P6` 在 `Journal.Begin` 之前；`R1`／`R2`／`R4` 属 `enter` 的 `Replace` 路径、位于同一 `Journal.Begin` 之后；`V1`／`V2`／`V4` 属 `VerifyRestored`（`enter` 侧在 Begin 之后，`resume` 侧无新 Begin 且在 `Prepare` 之后）；这些点按 `:4377` 原因分类（缺键或不可读 → Incomplete，同快照内不一致 → Binding，有效期/撤销/预检后漂移 → Closed）；越过工具启动点后（`R6`／`V6`／`V7`／`V8`／`V10`）→ `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown`；“任何情况下不得推进 `sequence`、不得当成功”保留。
- 与 `framework.md:1118` 的同段表述逐项一致（含 `enter` 的 `Replace` 在 Begin 之后、`resume` 无新 Begin）。
- 依据：`design.md:4070`、`:4037`、`:4135`、`:4137`；`framework.md:1118`。

### 3. `V7`／`V8` 前重验登记进唯一调用图（`:4037`、`:4062`）——**CLOSED（附 LOW：`:4135` 枚举未同步）**
- `:4037`：`V7` 从 `independent_read_path` 独立读取回执字节（读取前一次 `revalidateApproval` 类重验）→ `V8` 验 producer 签名（验签前再一次；“两次重验只增加拒绝点、不放宽既有检查，失败 → `errors.Join`／`restore_unknown`”）。
- `:4062`：同步登记，保留 `V10` 的 r67.5 推进前重验与“不得推进 head”约束；V1–V11 编号无增、无删、无重排、无复用，`V10` 语义未变。
- 失败类与 ADR 一致：V7/V8 在工具已启动之后，按“已越过启动点 → 一律 `errors.Join`／`restore_unknown`”。
- **残留（LOW）**：`:4135`“读取点与时钟点”仍只列 `runComposed`／`Prepare`／`Replace` 两次／`VerifyRestored` 两次／`check`／`V10`，未含 V7/V8 两次重验；而 `:4070` 恰把 `:4135` 列为“重验点完整集合”的组成之一。四行并读仍完整（`:4037`／`:4062` 已含 V7/V8），但只读 `:4135` 的实施者会漏两处 → **NEW-R11-02（LOW）**。
- 依据：`design.md:4037`、`:4062`、`:4070`、`:4135`、`:4137`、`:4074`。

### 4. 不可读错误分类收敛（`:3944`、`:4377`）——**CLOSED**
- `:4377` 为唯一 canonical：缺键、文件缺失或不可读、非规范路径、owner/mode 不符、非 regular/符号链接替换、超长、非 UTF-8、JSON 不合法、缺字段 → `ErrAdmissionIncomplete`；同快照内字段值/摘要/签名不一致 → `ErrAdmissionBinding`；未到 `valid_from`、已过 `valid_until`、`revoked` → `ErrAdmissionClosed`；平台不支持该文件边界 → `ErrAdmissionUnsupported`；**预检通过后的字节／`generation` 漂移（含 mtime／size／inode 变化）→ `ErrAdmissionClosed`**（旧 `ErrAdmissionBinding` 句已删，并注明 `:3944` 按本条读取）；“任一非 nil 都在 Begin 前返回，不降级为‘先恢复后补批准’”保留。
- `:3944` 声明旧“批准/信任根/DSN 不可读 → Closed”作废、旧“回执独立读取/审计投影不可读 → Incomplete”并入，按 `:4377` 唯一 canonical；“本条不另立分类”“映射保持 fail-closed 不变”；`result` 枚举与“`restore_unknown` 只能 `unknown`、不得写 `verified`”保留；`:4133` 的 `revalidateApproval` 漂移判定被显式划出本映射、仍按 ADR 路径分类保留。
- 同域检查未发现同一作用域内第二套映射：`:4113`／`:4123`／`:4161`／`:4185` 的“不可读→Incomplete、同快照不一致→Binding、撤销/过期→Closed”与 canonical 一致；`:4133` 的“删除/截断/非 regular/不可读→Closed”属“预检后漂移/替换”分支，与 `:4377` 的“预检后漂移→Closed”一致（口径张力见观察项，不影响 fail-closed）。
- 依据：`design.md:3944`、`:4377`、`:4133`、`:4113`、`:4123`、`:4161`、`:4185`；`framework.md:1118`。

## 三、10 行逐行核对（变更摘要）

| 行 | 旧→新 | 核对结论 |
|---|---|---|
| `:5` | `r70`→`r71` | 与 `:38` 一致；仅版本号 |
| `:38` | `r70`→`r71` | 同上 |
| `:3944` | +825 B | 改写为“以 `:4377` 为唯一 canonical”的取代说明；保留 result 枚举与 `restore_unknown` 限定 |
| `:4031` | +175 B | policy 改必填 `tool.receipt.trust_anchor_ref`；删除条件分支；**未写错误类**（NEW-R11-01） |
| `:4037` | +265 B | V7/V8 各加一次重验；编号未动 |
| `:4062` | +421 B | 同步 policy 必填与 V7/V8 重验；第五处等式与 V10 重验保留 |
| `:4070` | +651 B | B1 摘要按路径更正 Begin 门禁与错误分类；未再写 R/V“不进入 Begin” |
| `:4072` | +930 B | 优先级扩为 `r71 > r70 > …`；追加 r71.1–r71.5 登记；r66–r70 历史登记逐字保留 |
| `:4083` | +85 B | 追加“r71 修订沿用本声明”，NOT RUN／unsupported／closed 状态不变 |
| `:4377` | +140 B | 预检后漂移由 Binding 改 Closed；注明 r71.4 与 `:3944` 读取关系 |

零增删证明：两文件行数均 5,476；逐行比较除上述 10 行外全部 `-ceq` 相等；字节差 +3,492 等于 10 行 UTF-8 增量之和。

## 四、“若该键存在”语义判定

全文命中**恰 2 处**，分别在 `:4031` 与 `:4062`，且都位于变更说明括注“删除「若该键存在」条件分支，不保留「policy 该键可缺」的读法”内——是**否定引述**，不是存活条件；旧条件短语“policy 中的 `anchor_ref`（若该键存在）”命中 0。判定：不构成未闭合项。

## 五、与 `framework.md:1118`（ADR-0031 r12）对齐核对

| ADR 要素 | design r71 | 结论 |
|---|---|---|
| policy 必填 `tool.receipt.trust_anchor_ref`，缺键/空串/名不符即 closed，不保留可缺分支 | `:4031`、`:4062` | 一致 |
| 缺键/空串/名不符的**错误类**（启动前 `ErrAdmissionBinding`） | 未写死（canonical 反而给 Incomplete） | **不一致 → NEW-R11-01** |
| 五处逐字节相等（含 `receipt_anchors.anchor_ref` 自身） | `:4031`、`:4062` | 一致 |
| 三签三席位与公钥来源（producer `receipt_anchors.public_key`；verifier `keys[verifier_key_id].public_key` 且 principal 相等；witness `witness_key_id` 对应公钥） | `:3988`、`:4031`、`:4059`、`:4060`、`:4074` | 一致 |
| `generation` 三元（approval/trust_root/anchor） | `:4373` | 一致 |
| `receipt_anchors` 无 `revocation`／`valid_until`，撤销/轮换只经重钉与 generation 体现 | `:4060`（14 项闭合 keyset）、`:4077`（扩 15 项仅增 `witness_host`）、`:4090`（15 项指针） | 一致（无额外键即无法承载该二字段） |
| P6 与 Prepare 重读按 `:4377` 原因分类 | `:4377`、`:3944`、`:4070` | 一致 |
| R1–R8 在 `enter` 的 Replace、Begin 之后；V1–V11 在 enter/resume 均调用；resume 无新 Begin、不调 Replace | `:4037`、`:4070`、`:4135`、`:4137` | 一致 |
| V7/V8 前各一次重验、只增拒绝点 | `:4037`、`:4062` | 一致 |
| 越过启动点 → `errors.Join`／`restore_unknown`，不得推进 sequence | `:4070`、`:4137`、`:4062` | 一致 |
| 两种“见证”分域 | 见 `:4077` 与 ADR 见证消歧段；design 侧 `receipt_anchors.witness_*` 与 ADR-0030 `(w,r,token)` 无共用 | 一致（`framework.md` 侧已判） |

## 六、无回退底线逐项（依据：10 行外逐字节相同 + 变更行未削弱）

| 底线 | 证据 | 结论 |
|---|---|---|
| fail-closed 全路径 | `:3944`／`:4377` 映射保持 closed；`:4070` “不得当成功” | 未削弱 |
| 任一非 nil 都在 Begin 前返回 | `:4377` 原句保留 | 未削弱 |
| 三签模型与三席位密钥来源 | `:3988`、`:4031`、`:4059`、`:4060`、`:4074` | 未削弱 |
| `anchor_ref` 五处逐字节相等 | `:4031`、`:4062` | 未削弱（必填化加强） |
| `generation` 三元 | `:4373` | 未削弱 |
| `receipt_anchors` 无 `revocation`／`valid_until` | `:4060`／`:4077` keyset 闭合 | 未削弱 |
| no-follow／fd-exec／最小 ENV | `:3992`、`:4047`、`:4088`（`O_NOFOLLOW`）；`:4063`（ENV 白名单）；fd-exec 多处未变 | 未削弱 |
| `restore-v1`／`verify-v1` | `:4037` R5／V5；`:4043` 12 字段 wire | 未削弱 |
| `restore_unknown` 只能写 `unknown` | `:3944` 保留 | 未削弱 |
| Windows 生产 unsupported | `:4070` B6“退出码 2”；`:4010`／`:4021` NOT RUN 声明 | 未削弱 |
| 首期不提供生产 restore 工具与批准主体 | `:83`、`:3876`、`:3925`、`:4027`、`:4057`、`:4084`、`:4283`、`:4290`（8 处） | 未削弱 |
| r70 已闭合项（NEW-R09-01／NEW-R09-02／NEW-S09-01） | `:4076`（waitFlow 单点合同、reaper 单点、teardown 顺序）与 `:4079` 未在变更集内，逐字节同 r70；`:4072` r70.1／r70.2／r70.3 登记逐字保留（含“r68 时点值未登记、属登记缺口”） | 未回退 |
| golden vector 不变 | `:4099`／`:4105` 与 r70 基线 `-ceq` 相等 | 未回退 |
| 四类 unknown 仍为 unknown | `:4021`／`:4053`／`:4070` 未变 | 未削弱 |

## 七、两条 golden vector 独立复算

方法：取冻结副本中该行行内字节（UTF-8，不含行终止符），直接对字节串做 SHA-256；同时 `ConvertFrom-Json` 验证可解析。

| 向量 | 行 | 字节 | SHA256 | 结果 |
|---|---|---|---|---|
| source | `:4099` | **1,115** | **`de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab`** | 与文档声明及预期一致 |
| tool | `:4105` | **2,357** | **`8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508`** | 同上 |

两行均与 r70 基线逐字节相同，向量未被 r71 触碰。

## 八、括号平衡与后缀登记

- 全角括号：全文 `（` = **1,237**、`）` = **1,237**，平衡（含新增段）；r71 五行新增文本未引入失衡。
- 后缀：见第一节；三项与作者 `hashes.txt` 节 [5] 六项值全部吻合，`startByte` 增量关系（+3,352）自洽。

## 九、Findings

### [MEDIUM] policy `trust_anchor_ref` 缺失/空串/名不符的错误类未写死，与 ADR r12 及本文件 canonical 冲突

- finding ID：`NEW-R11-01`
- 对象/修订：design `identity-auth-design-ir01-r71`，`168B614D…`
- 位置：`design.md:4031`、`design.md:4062`（对照 `design.md:3944`、`design.md:4377`、`framework.md:1118`）
- 现状：两处只写“policy 侧缺键、空串或字段名不符即不得放行”，未绑定错误类；“其余四处任一不等，均 → 启动前 `ErrAdmissionBinding`”在句法上只约束另外四处比较。本文件 canonical 映射把“缺键／缺字段”判为 `ErrAdmissionIncomplete`；ADR-0031 r12 明确为“启动前 `ErrAdmissionBinding`，已启动后 `restore_unknown`”。
- 触发：批准记录 `tool.receipt` 缺少 `trust_anchor_ref`（或为空串、或字段名写成近似名），在启动预检/`Prepare` 读取阶段被拒。
- 影响：同一输入可按现行文本判为 Incomplete 或 Binding；按 ADR 编写的负向测试与按设计实现的错误码不一致，契约不可唯一机核。fail-closed 未被破坏，但闭类不确定。
- 建议：在 `:4031`／`:4062` 明写“缺键、空串、字段名不符 → 启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`”，或在 `:4377`／`:3944` 增加 policy 侧例外句；两处措辞与 `framework.md:1118` 逐字一致。
- 验证：`rg` 核对两处含 `ErrAdmissionBinding`；构造缺键/空串/近似名三类样例的判定表，与 ADR 文本逐项比对。

### [LOW] `:4135` 读取点枚举未随 r71.3 同步 V7/V8

- finding ID：`NEW-R11-02`
- 对象/修订：同上
- 位置：`design.md:4135`（对照 `design.md:4037`、`:4062`、`:4070`）
- 现状：`:4135` 的“读取点与时钟点”仍只列 `runComposed`／`Prepare`／`Replace` 两次／`VerifyRestored` 两次／`check`／`V10`，无 V7/V8 两次重验；而 `:4070` 把 `:4135` 列为“重验点完整集合”的组成之一。
- 触发：实现者或评审者只按 `:4135` 建立重验点清单。
- 影响：漏实现 V7/V8 两次 `revalidateApproval`；四行并读可纠正，故不构成 fail-open，属维护与测试盲区。
- 建议：在 `:4135` 追加一句“`V7`／`V8` 前的重验见 `:4037`／`:4062`（r71.3）”，或就地加 r71.3 指针并同步“`VerifyRestored` 两次”的表述。
- 验证：`rg -n 'V7|V8'` 与 `:4135` 的枚举逐项对照，确认重验点集合与 `:4070` 声明一致。

## 十、观察项（不计 finding）

1. `:4133`（`revalidateApproval`）把“删除/截断/非 regular/不可读”判 `ErrAdmissionClosed`，`:4377` 把“不可读”判 `ErrAdmissionIncomplete`。`:3944` 已把 `:4133` 显式划出本映射，且 `:4133` 的语义是“预检后漂移/替换”，与 `:4377` 的“预检后漂移 → Closed”一致；建议后续在这两处各加半句互指，减少读者歧义。
2. `:4113`／`:4123` 的“平台无法读取/解析信任根 → `ErrAdmissionUnsupported`”与同行的“缺失、不可读 → `ErrAdmissionIncomplete`”并存；按“平台能力”与“文件可读性”分域可解释，但字面易混。
3. `:4074`（r66.2）称 V7“只做长度与 JSON 语法校验”；r71.3 在 V7 前加了重验。因 `:4037`／`:4062` 明写“读取前”，且优先级 `r71 > r66`，不构成冲突，建议在 `:4074` 加 r71.3 指针。
4. `tasks.md` 现行 904,893 B / 3,312 行 / `696A5B0EE9B2E4DA75D4C50EB82BCFC0E68D657500BDED346F4CA765822BD28E`，与旧轮登记的 904,142 B / `CA20577B…` 不同——说明该文件已由其他会话改动；本轮只核 `:3186`，不做判读。

## 十一、范围外登记

- `tasks.md:3186` 仍为 `- [ ]`（未勾选）；受 ADR-0031 门禁，与锁后端节点独立。未改、未勾选。
- `N005`（ADR-0031 状态与 `tasks.md` 状态段冲突）：范围外，交架构决策方与 Task-Planner；不据此判 CHANGES REQUIRED。
- `F017`（`tasks.md` 写集缺口）：待 Task-Planner 对账；本轮不改卡面。
- `decisions/README.md`：未读取、未修改；不参与判定。

## 十二、NOT RUN

build、`go test`、E2E、Linux host／ACL／mount／loader／fd-exec／maps／network／witness、真实 restore、SEC-F007、真实审批主体与生产工具：全部 **NOT RUN**。本报告不构成生产 restore 解锁、不构成 `enter`／`resume` 解锁、不构成约束 8 满足、不构成 `identity-auth-S4.24a` 可关闭或可派发。

## 十三、实际执行命令（PowerShell 5.1，节选）

- `Get-FileHash -Algorithm SHA256 -LiteralPath`（design.md／快照／r70 基线／framework.md／tasks.md／r10 报告／作者证据）
- `[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)` + 逐行 `-cne` 比较与 `[Text.Encoding]::UTF8.GetByteCount`
- `[IO.File]::ReadAllBytes` 扫描 0x0A 求行首偏移，计算三段后缀字节数与 SHA-256
- `[Security.Cryptography.SHA256]::Create()` 对 `:4099`／`:4105` 行内字节做摘要；`ConvertFrom-Json` 校验
- `[regex]::Matches` 统计全角括号与“若该键存在”命中

## 计数

- BLOCKER：0
- MEDIUM：1（`NEW-R11-01`，开放）
- LOW：1（`NEW-R11-02`，开放）
- 范围外：1（`N005`）

报告自身终值（自引用哈希无法内嵌）：字节 21,990／行 194／SHA256 见同目录 `hashes.txt` 首行登记。

Verdict: CHANGES REQUIRED