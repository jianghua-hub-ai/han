# ADR-0031 `adr-0031-r11` 安全独立复审（r05，决策接受级）

- 日期：2026-11-02
- 角色：add-reviewer（安全轴）｜objectType=adr｜只判「决策接受级」（`framework.md:1122` 第 1 级）
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 被审对象：`.add\project\knowledge\decisions\framework.md`
  - 实测 **292,606 B / 1,274 行 / CR=1,274 / LF=1,274 / bare-LF=0 / 无 BOM / 首 3 字节 `23 20 E6` / 末 2 字节 `0D 0A` / SHA256 `9723F92FB53B076D228B5FA392CEC07CA0F74ED460524EBB7F00AC9202EB53CD`**
  - 修订：`adr-0031-r11`（`:1082`、`:1084`）
- 冻结快照：`%TEMP%\framework-adr0031-r11-freeze-9723F92F.md` = 292,606 B / 同哈希（实测 `C:\Users\J00010385\AppData\Local\Temp\framework-adr0031-r11-freeze-9723F92F.md`）
- 前像基线：`%TEMP%\framework-adr0031-r10-freeze-B432E44A.md` = 285,639 B / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`
- 判读段：`:1079`–`:1197`（ADR-0032 起于 `:1197` 的 `<a id=adr-0032></a>`）
- 基准报告（哈希均本轮复算）：
  - 安全 r04（我的上轮）：24,983 B / `C1AE35DF51F9D1CFAA14AA5B2E39EC33F9DD80CD428D507F0CE316FE1DABB9CE`
  - 架构 r04：22,635 B / `F17AC89A9970BC89FF5B96625531DD48BBA6ECA5ECB45B695D790465613A3DBC`
  - 安全 r03（原 finding 来源）：22,195 B / `501B3102D61A74CBA775961ACC8A7C7ED2EA1254BDB7091C6C1CE2F68D180F9F`
- 作者证据（**非证据**，只作线索）：`adr-0031-r11\report.md` 5,318 B / `7BEBB93235C6910C97F9E0D9902BEA7A921C019107C8EFFD663559D55F0E4171`；`hashes.txt` 1,391 B / `B055D80E7A954B03841BACCE94AAA4D881A72C5E4916EA19888C67BE104FA099`
- 交叉对象：`design.md`（`identity-auth-design-ir01-r70`）1,103,152 B / `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`（未漂移）；`tasks.md` 904,142 B / `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C`
- 方法：完整读 `.codex\skills\add-code-review\SKILL.md`（取 adr 分支），按加载规则读 `.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`，并读 `.codex\skills\add-code-review\references\产品安全红线.md`；两轮独立检查 + 独立复算；不采用作者声明或作者证据文件中的数字作为结论依据。
- 两轮实际检查范围：
  - 第一轮（对象完整性与 diff）：全文字节/行/CR-LF/BOM/首末字节与 SHA256；与 r10 冻结前像逐行比对（含行插入后的尾部平移比对）；段前/段后逐字节比对；`:1082`、`:1084`、`:1118`、`:1174`、`:1192` 与新增 `:1193` 逐句判读；不变量 1–6 与约束 1–9 段是否被改动。
  - 第二轮（上游/消费者交叉与独立复算）：design r70 的 `:1537`、`:3933`、`:3934`、`:3942`、`:3944`、`:3988`、`:4031`、`:4037`、`:4062`、`:4070`、`:4074`、`:4083`、`:4113`、`:4123`、`:4135`、`:4137`、`:4157`、`:4169`、`:4199`、`:4211`、`:4292`、`:4373`、`:4377`、`:4416`、`:4437`；design 全文 `Journal.Begin`／`r68.`／`r69.`／`r70.` 检索；`tasks.md:3186`／`:3197`／`:3220`；作者 report/hashes 的数值复算。
- 只读声明：本轮仅写本目录 `report.md` 与 `hashes.txt`；未修改、删除、移动、归档任何其他文件；未执行任何 Git 写操作。判读过的文件：`framework.md` 及其 r10/r11 快照、`design.md`、`tasks.md`、`.codex\skills\add-code-review\SKILL.md`、`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\skills\add-code-review\references\产品安全红线.md`、`review-sec-r04\report.md`、`review-arch-r04\report.md`、`review-sec-r03\report.md`、`adr-0031-r11\report.md`、`adr-0031-r11\hashes.txt`。

## 结论摘要

- r11 的改动范围、字节账与「段外不变」经独立复算成立：仅 `:1082`、`:1084`、`:1118`、`:1174`、`:1192` 五行变更并新增 `:1193`；净 **+1 行 / +6,967 B**（285,639→292,606）；段前（行 1–1078）与段后（旧 1193–1273 == 新 1194–1274）逐字节相同。
- 架构侧 `ADR-R04-01`（MEDIUM）的**位置**（P6/Prepare 与 R/V 的 Begin 前后关系）与 **P3** 两处已正确修复；R/V 的错误分类已按「原因 + 工具启动点」分层；「不得推进见证 `sequence`、不得当成功」保留。安全上正确，未削弱 fail-closed。
- 安全侧原 `SEC-R03-F001`–`SEC-R03-F007` 的闭合句在 r11 中逐字或等价保留，未发现判据删除；三签模型、`sequence` 单调、独立读取、验签前置、unknown 保守声明与「不声称抗 root」均无回退。
- 新增 1 条 LOW（`ADR-S05-01`）：`P6`/`Prepare` 的读取失败仍统一写 `ErrAdmissionClosed`，与 design `:4377`/`:4113`（缺键/不可读→`ErrAdmissionIncomplete`）冲突，且 design `:3944` 本身给出相反映射；所有分支均 fail-closed，无「失败当成功」读法，不阻断决策接受。
- 携带上轮 design 侧 LOW（`ADR-S04-01`：design `:4031`/`:4062` 仍写「（若该键存在）」）；`SEC-R03-F006`（`tasks.md:3220`）仅作范围外登记。
- 计数：**BLOCKER 0 / MEDIUM 0 / LOW 1**（ADR 侧）；另有 design 侧携带 LOW 1、范围外观察项 1，均不计入本对象阻断计数。

## 1. 改动范围与字节账（独立复算）

| 项 | 实测 | 判定 |
|---|---|---|
| 段前（行 1–1078，含 CRLF） | 205,336 B，新旧逐字节/哈希相同 | 段外不变 |
| 旧 ADR 段（1079–1192） / 新段（1079–1193） | 59,129 B / 66,096 B | +6,967 B |
| 段后（旧 1193–1273 / 新 1194–1274） | 21,174 B，新旧逐字节/哈希相同 | 段外不变 |
| 逐行差异 | 仅 `:1082`、`:1084`、`:1118`、`:1174`、`:1192` 变更；新增 `:1193`；旧 1193–1273 与 新 1194–1274 逐行 0 mismatch | 与作者声称一致 |
| 合计 | 205,336+66,096+21,174 = 292,606；旧 285,639 | +6,967 B / +1 行 |
| 行尾/编码 | CR=1,274、LF=1,274、bare-LF=0、无 BOM、首 3 字节 `23 20 E6`、末 2 字节 `0D 0A` | 与哈希一致 |

补充口径说明：作者 `hashes.txt` 的前缀 109,097 字符/205,357 B、后缀 11,152 字符/21,165 B 采用 `ReadAllText` + `IndexOf` 的**字符索引边界**（`## ADR-0031:` 标记行可能带前导锚点），与本报告按**整行边界**（205,336 B / 21,174 B）相差数十字节；两种口径的「段外不变」结论一致，我按整行口径独立复核为真。

## 2. 逐条判定

### 2.1 `ADR-R04-01`（架构 r04 MEDIUM）修复的安全正确性 — **已闭合（残留 1 条 LOW，见 `ADR-S05-01`）**

- **P3 不再作 generation 漂移判定**：`:1118` 现写「`P3` 只读取批准文件、信任根与目标 DSN 并构造不可变快照、完成首次合取（无前像可比，故不作 generation 漂移判定，也不属重验点）」；与 design `:4037`（P3 构造快照并完成全部合取）、`:4373`（「其一，启动阶段…唯一预检…取得不可变快照」）一致。P3 的撤销/有效期检查仍以「首次合取」存在（`:4123` 启动预检一次性完成 principal 存在性、任命绑定、review 解析与角色交叉，失败分类含「principal 或任命已撤销…批准已过期→`ErrAdmissionClosed`」），未减少检查项；P3→P6 之间的文件变更由 P6 的前像比对捕获（`:4037`、`:4373`「`Prepare` … 在 `Journal.Begin` 之前重读文件精确字节与快照比对」），**无检测空窗**。
- **重读口径**：`:1118` 限定为「`P6` 及 `P6` 之后每个检查点」，与 `:4373` 的「其二…其三」一致。
- **位置（Begin 前/后）**：`:1118` 现写「`R1`–`R8` 属 `Replace` 路径、`V1`–`V11` 属 `VerifyRestored` 路径，`enter`／`resume` 中二者的调用均在 `Journal.Begin` 完成之后」。design **`:4169` 明写**「其中 `Replace` 与 `VerifyRestored` 两次发生在 `Journal.Begin` 之后；`Prepare` 那次早于 `Journal.Begin`」，`:4135` 亦写 `Prepare` 在 `Journal.Begin` 之前调用 `revalidateApproval`。故该句**事实成立**；`P6`/`Prepare` 的「Begin 前、不进入 `Journal.Begin`」子句与 `:4037`/`:4135` 一致。
- **R/V 失败分类**：按「先看原因（缺键/缺失/不可读→`ErrAdmissionIncomplete`；未到 `valid_from`/已过 `valid_until`/`revoked`→`ErrAdmissionClosed`；预检后漂移且工具未启动→`ErrAdmissionClosed`）→ 再看是否越过工具启动点（越过→`errors.Join(recovery.ErrRecoveryWriteUnknown, err)`→`restore_unknown`；`R6/V6`、`V7/V8/V10` 前均属此列）」分层。与 design `:4062`（`V10` 推进前失败→`errors.Join(...)` 且不得推进 head）、`:4037`（`R6/V6` 之后漂移必须 `restore_unknown` 或 closed）、`:4031`（`ErrAdmissionBinding` 启动前 / `restore_unknown` 已启动）一致。ADR 把 `R6/V6` 及 `V7/V8/V10` 前一律定为 `restore_unknown`，比 design 的「`restore_unknown` 或 closed」更严，属**收紧**，不是放宽。
- **不推进见证 `sequence`、不得当成功**：`:1118` 原句「任何情况下**不得推进见证 `sequence`**、不得当成功」保留。
- **fail-closed 是否被削弱 / 是否产生「失败被当成功」读法**：未削弱。三档分类全部是 error 返回；`P6`/`Prepare` 在 `Journal.Begin` 前阻断，R/V 越过工具启动点后落 `restore_unknown`；design `:3944` 明定「`restore_unknown` 只能写 `unknown`，**不得写 `verified`**」「任一检查点失败都在返回成功之前发生；不存在『先返回成功再异步核验』」。`errors.Is` 可区分集合（`:1537`）已含 `ErrAdmissionIncomplete`。未发现任何分支可读成成功。
- **残留**：见 `ADR-S05-01`（`P6`/`Prepare` 的读取失败仍统一 Closed，与 `:4377`/`:4113` 的 `ErrAdmissionIncomplete` 不一致；design `:3944` 又给出相反映射）。

### 2.2 `SEC-R03-F001`（HIGH，anchor_ref 五处 + policy 必填）— **仍闭合**

- `:1118` 逐字保留：「`receipt_anchors.anchor_ref` = 批准记录 policy 的 `tool.receipt.trust_anchor_ref` = 回执 `trust_anchor_ref` = 审计记录 `anchor_ref` = 见证 head `anchor_ref`，五处逐字节相等」。
- policy 字段为必填：「缺键、空串或字段名不符即 closed——启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`；**不保留「policy 该键可缺」的条件分支**」。
- 全文检索 `framework.md` 的「若该键存在」= **0 命中**，无「该键可缺」分支。
- 伴生观察：design `:4031`、`:4062` 仍写「（若该键存在）」，属 design 写集 → 携带 `ADR-S04-01`（design 侧 LOW，非本对象缺陷）。

### 2.3 `SEC-R03-F002`（HIGH，root 残余无条件）— **仍闭合**

- `:1118`「**无条件残余（r10）**」整句保留：「root 可整体替换信任根、批准记录与见证，本 ADR 不声称抗 root……该残余无条件成立，不取决于组织侧是否提供独立验证方与固定信任锚；生产 restore 因此继续 closed，不得以「有签名」或「有外部见证」暗示已抗 root 伪造」。
- 仅 design 引用由固定修订号改为条款锚（`r64.10`/`r65.13`），未改残余语义。

### 2.4 `SEC-R03-F003`（HIGH，generation 三元 + 检查点 + 不推进 sequence）— **仍闭合（含定义细化，实质未减）**

- 三元摘要：`approval_generation`（批准文件精确字节 SHA-256，预检快照记录、重读比对）、`trust_root_generation`（信任根精确字节 SHA-256，须等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`）、`anchor_generation`（`receipt_anchors` 对应条目 JCS 字节 `sha256:`，须等于 policy 与回执的 `trust_anchor_sha256`）——可复算，且「不得以文件可读/inode 未变/长度未变替代摘要比较」保留。
- 检查点集合：`P6`、`R1`/`V1`（`prepared`）、`R2`/`R4`/`R6`、`V2`/`V4`/`V6`、`V7` 前、`V8` 前、`V10` 前，全部保留；`P3` 由「重验点」改为「首次合取」——撤销/有效期在 P3 仍被检查，仅 generation 漂移判定后移到 P6 起的首个可比点；P3→P6 变更由 P6 捕获，检测无空窗。与 design `:4037`/`:4062`/`:4135`/`:4373` 一致。
- `V7`/`V8` 前追加关闭判据仍写「只增加拒绝点、不放宽任何既有检查」，并保留 `V10` 的 r67.5 推进前重验。
- 失败分类：见 2.1，`Begin` 前阻断、工具启动后 `restore_unknown`；「不得推进见证 `sequence`」保留。
- 结论：F003 的实质（三元可复算、检查点不删、任何情况下不推进 sequence、失败绝不当成功）未削弱；差异是把「P3 的撤销/有效期」正名为「首次合取」，并把漂移判定后移到有前像的 P6。

### 2.5 `SEC-R03-F004`（MEDIUM，见证独立性含 verifier、六席 person_ref）— **仍闭合**

- `:1147`（不变量 6，与 r10 逐字节相同）：「`witness_principal_id`／`witness_key_id` 及其 `org_id` 不得等于 root／executor／producer／issuer／reviewer／**verifier** 任一方，且 `witness_key_id` 与 `verifier_key_id`、`witness_principal_id` 与 `verifier_principal_id` 必须分别不同；…六席 `reviewer`／`issuer`／`executor`／`producer`／`verifier`／`witness` 的 `person_ref` 必须两两不等」。
- `:1166`（约束 8 解除条件第 6 条，未改）同口径保留。
- `:1118` 三签句明确「三个互不相同的席位，任一席位不得代签其他席位的载荷」，`witness_key_id` 只是见证公钥标识、**不是**回执签名公钥唯一来源。

### 2.6 `SEC-R03-F005`（MEDIUM，可复算性）— **已闭合（受限：作者证据文本有 1 处笔误）**

- ADR 自身给出改前基线（285,639 B / 1,273 行 / `B432E44A...`）与改后值出处；我独立复算：对象 292,606 B / 1,274 行 / `9723F92F...`，前像 285,639 B / `B432E44A...`，逐行差异与段外不变均成立（见 §1）。第三方可复算。
- 受限：作者 `adr-0031-r11\report.md` 写「首字节为 `35 32 E6`」，实测为 **`23 20 E6`**（`#`+空格+UTF-8 首字节）。该笔误不涉及 bytes/行数/SHA256 等可复算数字，不影响 F005 的闭合结论；已列建议 S-2。

### 2.7 `SEC-R03-F006`（MEDIUM，tasks.md 冲突）— **范围外登记（不据此判 CHANGES REQUIRED）**

- `tasks.md`（904,142 B / `CA20577B...`，与 r04 审查时相同）`:3220` 仍写「`adr-0031-r07` 为现行修订……决策接受级对 r07 成立，故本卡启动门禁解除」，与 `framework.md:1084`（现行 `adr-0031-r11`、两路接受尚未取得、`identity-auth-S4.24a` 及任何实施/验证卡不得启动）冲突；`:3186` 仍未勾选（`- [ ]`），与 ADR 的 blocked 一致。
- 影响：若有人以 `tasks.md:3220` 为依据派发/认领 S4.24a，将绕过现行修订的启动门禁。该冲突属 `tasks.md` 写集，本轮不改、不代改、不据此判本对象不通过；交 Task-Planner 对账。

### 2.8 `SEC-R03-F007`（LOW，两种见证分域）— **仍闭合**

- `:1118`「**见证术语消歧（r10）**」整段保留：本节「见证」只指 restore 回执的外部锚见证（Ed25519 签 5 键 head、持久单调 `sequence`、`receipt_anchors.witness_*`），与 ADR-0030 的 anchor revision 见证 `(w, r, token)`「对象、状态、endpoint 与 key 相互独立，不得共用 key、endpoint、记录或推进通道」；`anchor_ref` 不是 ADR-0019 的 `JournalHeadAnchor`。

### 2.9 新引入安全缺陷核查（三签 / sequence / 独立读取 / unknown / 抗 root）— **未发现**

- **三签载荷与 design r70 实质一致**：
  - producer：`receipt_anchors.public_key` 签 `JCS({"receipt": R10, "attestation": A6})`，`R10`=12 字段回执删 `signature`/`audit_record_ref`，`A6`=7 字段 attestation 删 `signature` → 与 design `:3988`、`:4074`（V8 唯一权威 `:3988`）一致。
  - verifier：`keys[verifier_key_id].public_key` 签 8 字段（`anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id`），`principal_id == verifier_principal_id` → 与 `:4031`、`:4074` 一致。
  - witness：`receipt_anchors.witness_key_id` 对应公钥签 5 键 head（`anchor_ref, sequence, issued_at, head_sig_alg, head_sig`）删 `head_sig` 后 JCS；公钥取 `receipt_anchors.witness_public_key`（design `:4033`/`:4035`/`:4060`）→ 与 `:4074` 一致。
  - 公钥按席位分取（`receipt_anchors.public_key` / `keys[verifier_key_id].public_key` / `receipt_anchors.witness_public_key`）落实了 `ADR-R04-03`，无「在 `receipt_anchors` 找 verifier 公钥」的误导。
- **`sequence > max(baseline_sequence, head.sequence)` 且 baseline 只增不减**：`:1118` 与 `:1147` 均保留；`:1147` 另写「任何重钉必须由人的明确批准并留证，降低尝试即 closed」。
- **独立读取路径与验签前置**：`:1118` 保留「root 必须经与写入路径不同的独立读取路径读取」「读取回执时必须先验签并留审计记录」；V7 独立读→V8 producer 验签→V10 验证方签名后推进，顺序未变。
- **不得把 unknown 写成已具备**：`:1193` 末句「四项 unknown 仍为 unknown；`adr-0031-r11` 未取得两路接受，不构成评审 PASS、人的接纳或生产批准」；`:1118` 的 unknown 段与「生产 restore 因此继续 closed」保留。
- **不暗示抗 root**：见 2.3；`:1118` 明确「不得以『有签名』或『有外部见证』暗示已抗 root 伪造」。
- **模拟见证边界**：`:1118` 保留「本机独立 witness CLI…只是测试替身，证据必须标注「模拟见证、未真实验证」」。

### 2.10 无回退底线（字节级核对）

| 底线 | 证据（行号） | 结论 |
|---|---|---|
| 不变量 1–6 | `:1131`–`:1149`，r11 未改任何字节（逐行差异为 0） | 未削弱 |
| 约束 1–9（含解除条件九条） | `:1151`–`:1170`，r11 未改任何字节 | 未削弱 |
| 首期不提供生产 restore 工具与生产批准主体 | `:1084`、`:1090`、`:1176`（仅 `:1084` 改修订号/评审状态，结论句保留） | 未削弱 |
| `restore-v1`/`verify-v1`、私有 stdin | `:1086`（未改） | 未削弱 |
| no-follow 稳定 fd 与 fd-exec、无 shell | `:1086`、`:1115`、`:1117`、`:1141`（未改） | 未削弱 |
| 最小 ENV allowlist（禁 `LD_*`/`*_PRELOAD`/`*_OPTIONS`/`GLIBC_*` 口径） | `:1117`、`:1144`（未改） | 未削弱 |
| Begin 前 closed | `:1086`、`:1090`、`:1118` | `:1118` 改写后仍保留 P6/Prepare 的 Begin 前阻断；R/V 按工具启动点分类，越过即 `restore_unknown`；未放宽 |
| `restore_unknown` 保守暂停 | `:1086`、`:1090`、`:1118`、`:1149` | 未削弱 |
| Windows 生产 unsupported | `:1117` 及 design `:4437`（引用未改） | 未削弱 |
| 模拟见证必须标注「未真实验证」 | `:1118` | 未削弱 |
| 决策接受级随现行修订号自动生效 | `:1084`、`:1129`、`:1176` | 未削弱（`adr-0031-r11` 明确未取得两路接受） |

## 3. Findings（新编号 `ADR-S05-nn`）

### [LOW] `ADR-S05-01` — `P6`/`Prepare` 的读取失败仍统一写 `ErrAdmissionClosed`，与 design 的 `ErrAdmissionIncomplete` 口径冲突

- finding ID：`ADR-S05-01`
- 对象/修订：adr，`adr-0031-r11`，`framework.md:1118`
- 位置：`framework.md:1118`（对照 `design.md:4377`、`:4113`、`:4123`、`:3944`）
- 现状：`:1118` 对 `P6` 及 `Prepare` 内重读写「失败返回 `ErrAdmissionClosed`，且不进入 `Journal.Begin`」，只对 R/V 按原因分层（缺键/文件缺失/不可读→`ErrAdmissionIncomplete`）。但 design `:4377` 明定「缺键、文件缺失或不可读…→ `ErrAdmissionIncomplete`；…任一非 nil 都在 Begin 前返回」，`:4113`（信任根缺失/不可读→`ErrAdmissionIncomplete`）、`:4123`（`Prepare`/`Replace`/`VerifyRestored` 每次 `revalidateApproval` 重算同一合取）同口径；反之 design `:3944` 写「批准文件、信任根或目标 DSN 不可读 → `ErrAdmissionClosed`」，design 自身亦不自洽。
- 触发：`Prepare`/`P6` 重读时批准文件或信任根缺失/不可读（非漂移）。
- 影响：按 ADR 字面实现会返回 `ErrAdmissionClosed`，按 design B1 段（`:4377`，更具体）应返回 `ErrAdmissionIncomplete`；两者只在错误码与负向测试期望上分叉。**安全影响为零**：两分支都在 `Journal.Begin` 前阻断，均为 fail-closed，外部只映射 `platform.recovery_in_progress`（design `:1537`），不产生成功读法、不写 DB/CAS、不建子进程、不推进见证 `sequence`。
- 建议：把 `P6`/`Prepare` 子句改为「漂移 → `ErrAdmissionClosed`；缺键/缺失/不可读 → `ErrAdmissionIncomplete`」，或在 `:1193` 的「上游残留观察项」中与 `:4070` 并列登记 design `:3944` 的相反映射并声明以 `:4377` 为准。
- 验证：逐点列出 (`P3`/`P6`/`Prepare`/`R*`/`V*`, Begin 前/后, 原因, 错误码)，与 design `:4037`/`:4113`/`:4377` 逐项比对无第二种读法。

## 4. 开放项 / 待决定 / 建议

### 开放项
1. `ADR-S05-01`（LOW，本对象）：见 §3。非阻断。
2. `ADR-S04-01`（LOW，**design 侧**，上轮携带）：design `:4031`、`:4062` 仍写「（若该键存在）」，与 ADR/F001 的「policy 必填、无缺键分支」在字面上不一致；r11 未改 design（设计写集），ADR 侧已无该分支。不阻断决策接受。
3. `SEC-R03-F006`（范围外）：`tasks.md:3220` 仍以 `adr-0031-r07` 为现行并称门禁解除，与 `framework.md:1084` 冲突；`tasks.md:3186` 未勾选。只登记，不据此判 CHANGES REQUIRED，不代改。

### 待决定
1. `ADR-S05-01` 是否需要在正式接受前由作者补一句（把 `P6`/`Prepare` 的原因分类对齐 `:4377`），还是由 Lead 以「design `:4377` 更具体、ADR 子句从简」为由显式接受。安全侧意见：两分支均 fail-closed，接受该 LOW 不影响决策接受级与任何安全结论；若要求决策级判据零歧义，可回退作者一次，成本约一行。

### 建议（非阻断）
- S-1：`:1193` 的「上游残留观察项」可更精确——`:4070` 中 `R1/V1/R2/V2/R4/V4` 漂移→`ErrAdmissionClosed` 与 ADR r11 实际一致（都在工具启动点前）；真正的残留是 `P3` 仍被列为漂移判定点，以及 R/V 的「不进入 `Journal.Begin`」措辞。另 design `:4137` 也有同类「撤销后…不进入 Begin」措辞（针对 `Replace`/`VerifyRestored`），可一并登记。
- S-2：作者 `adr-0031-r11\report.md` 的「首字节 `35 32 E6`」应更正为 `23 20 E6`；不影响 bytes/行数/SHA256 复算。
- S-3：作者 `hashes.txt` 的前缀/后缀字节采用字符索引边界（205,357 / 21,165），与整行边界（205,336 / 21,174）不同；建议注明边界口径，避免第三方复算时误判。
- S-4：design `:4157` 的 producer 载荷仍按 r61 旧形态描述（未含 attestation/`signature_alg`），与 `:3988`/`:4074` 的「唯一权威」定义字面不同；属 design 写集，ADR 引用的是 `:3988`，无冲突，仅登记。

## 5. NOT RUN（不得据文档存在或作者声明判 PASS）

- Linux 宿主隔离实测（POSIX 进程组、`reap`、`setns`/`seccomp`）
- ACL/GRANT、mount、loader、fd-exec、`/proc/<pid>/maps`、network、witness 实测
- 真实原生耐久回执样本、真实 HSM/外部见证、专用执行身份实测
- build / vet / gofmt / test / E2E
- 任何生产装配或生产 restore 演练
- 任何人的成果接收、独立评审接受或生产批准

## 6. 实际执行命令（PowerShell 5.1，节选）

```powershell
Get-FileHash -Algorithm SHA256 -LiteralPath '.add\project\knowledge\decisions\framework.md'
[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)
[IO.File]::ReadAllBytes($p)   # 首/末字节、CR/LF/bare-LF/BOM
# 逐行 diff：old vs new；尾部平移比对 old[1192..] vs new[1193..]
# 段前/段后：按行切片后 Get-FileHash / 逐字节比对
# design 交叉：for($i=0;...){ if($L[$i] -match 'Journal\.Begin'){...} }
#   以及 :4070/:4377/:3944/:4113/:4169/:4137/:4157 逐行读取
# framework 全文检索：'若该键存在'、'adr-0031-r\d'、'trust_anchor_ref'
# tasks.md:3186/:3197/:3220 读取与哈希
```

## 7. 计数

- BLOCKER：0
- MEDIUM：0
- LOW：1（`ADR-S05-01`，本对象；非阻断）
- 携带：design 侧 LOW 1（`ADR-S04-01`）；范围外观察项 1（`SEC-R03-F006`）

**审查限制**：本轮只判决策接受级；不判卡证据级与生产解除级；未运行任何 Linux/build/test/E2E；不以作者证据文件中的数字或本 ADR 的文字存在替代实测。

Verdict: PASS