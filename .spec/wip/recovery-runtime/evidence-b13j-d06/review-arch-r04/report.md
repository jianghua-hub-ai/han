# ADR-0031 `adr-0031-r10` 架构独立复审报告（r04）

**日期**：2026-11-02  
**评审角色**：add-reviewer（架构轴）  
**对象类型**：ADR（决策接受级）  
**工作区**：`D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`

## 1. 冻结对象与基准

| 项 | 路径/说明 | 字节 | 行数 | SHA256 |
|---|---|---:|---:|---|
| 被审对象 | `.add\project\knowledge\decisions\framework.md` | 285,639 | 1,273 | `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050` |
| 冻结快照 | `%TEMP%\framework-adr0031-r10-freeze-B432E44A.md`（同字节） | 285,639 | 1,273 | `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050` |
| ADR-0031 判读段 | `framework.md:1079`–`:1192`（含末尾 CRLF） | 59,129 | — | `DD1A61F8D76F06F9118CBAF8AF01FB94AF610E5D7EF2FDA19DDB45D96CCC0D48` |
| 前缀 | `framework.md:1`–`:1078`（含末尾 CRLF） | 205,336 | — | `7CFF3F32B15D79316D07BDE2BDE8A2C6A74C835B8DAE78158A247FC4E328F1F1` |
| 后缀 | `framework.md:1193`–`:1273` | 21,174 | — | `AA9E7E183B6D0A9301D736DEE890392CCAE4619C0B942216FBA03F32134FD599` |
| design 对照 | `.add\project\knowledge\design-workspace\identity-auth\design.md`（r70） | 1,103,152 | 5,476 | `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254` |
| 架构 r03 基准 | `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r03\report.md` | 30,450 | 231 | `15C193D62217ADD1E71D621805F1F88E1E1A84DD0DA8BFA295143D0C3B3FAF5B` |
| 安全 r03 基准 | `.spec\wip\recovery-runtime\evidence-b13j-d06\review-sec-r03\report.md` | 22,195 | 259 | `501B3102D61A74CBA775961ACC8A7C7ED2EA1254BDB7091C6C1CE2F68D180F9F` |
| 作者 r10 报告 | `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r10\report.md` | 14,800 | 116 | `D8A8C382AB27BFC2888AB8728CC17F19797910DFE96361072CDA78CC4C96F8F7` |
| 作者 r10 指纹 | `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r10\hashes.txt` | 3,487 | 50 | `6BE0058A03D0A2CF796BF20D0EC0E875D736C7C9F4F9D0D38CB3BAFC9BA614B7` |
| 范围外观察 | `.add\project\knowledge\tasks\tasks.md`（现场已漂移） | 904,142 | 3,312 | `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C` |
| 索引观察 | `.add\project\knowledge\decisions\README.md` | 21,613 | 110 | `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2` |
| 授权台账 | `.spec\wip\recovery-runtime\evidence-b13j-s424a\lead-authorization-r12.md` | 3,507 | 42 | `CC18D2622461391EACE5B523B5ABB113EEA8737D24A064D287AB46C6EA5EF3B9` |

现场 `framework.md` 与快照同字节同哈希，无对象漂移。ADR-0031 标题在 `:1080`，修订行 `:1082` 为 `adr-0031-r10`，判读范围到 `:1196`（含空行与分隔线）；ADR-0032 标题在 `:1197`。`tasks.md` 的漂移与 `README.md:39` 仍写 `adr-0031-r09` 均只作范围外观察，不改变本报告对冻结 ADR 的判读。

### 字节账自算

- `205,336 + 59,129 + 21,174 = 285,639`，与全文字节数一致。
- 作者记录的改前全文 `272,588 B / 1,271 行 / D8DBA64B…`，按其改前 ADR 段 `46,078 B` 计算：`205,336 + 46,078 + 21,174 = 272,588`；改后 `285,639 - 272,588 = +13,051 B`，行数 `+2`，算术自洽。
- 当前前缀、ADR 段、后缀三段哈希均由本评审按 raw bytes 独立复算；作者 `hashes.txt` 的「不含末尾 CRLF」口径未按同一约定独立复算，属于接受受限，不作为 finding。
- **无 r09 前像字节**：可独立复算的是 r10 的三段哈希、全文哈希和改前/改后算术；`r09→r10` 的逐行差异只能依据作者记录的 11 行清单与 `PREFIX_BYTE_EQUAL=True`／`SUFFIX_BYTE_EQUAL=True`，本评审不能从真实前像重算 diff。该限制在 F001 中明示。

## 2. 方法、检查范围与只读声明

### 方法

- 已完整读取 `.codex\skills\add-code-review\SKILL.md`，按 `objectType=adr` 维度执行；已读取 `.codex\guidance\code-review-checklist.md` 与 `.codex\guidance\abstraction-principles.md`。
- 以决策接受级判据审查：形态、最小字段、可核判据、否证条件是否唯一可实施；不以“文档存在”或作者声明代替证据。
- 两轮检查：
  1. **第一轮**：复核冻结对象、快照、字节账、行尾/编码；逐行读取 ADR-0031 `:1079`–`:1196`；读取架构 r03 `F001`–`F009`、安全 r03 基准、作者 r10 报告与 `hashes.txt`；核对引用真实性与范围外观察。
  2. **第二轮**：对照 design r70 的 `:3984`–`:3991`、`:4028`–`:4045`、`:4150`–`:4164`、`:4368`–`:4380`；对照 `internal/recovery/runtime.go:1570`–`:1668` 的实际调用时序；复核 ADR-0030 `:1005`–`:1027`、`:1009` 的见证语义；核对门禁分级、无回退底线和引用哈希。

### 只读声明

本评审未修改 `framework.md`、`design.md`、`tasks.md`、`decisions/README.md`、ADR、代码、测试或既有评审/修订目录；未删除、移动、归档任何文件；未执行任何 Git 写操作。实际判读过的文件包括：

- `.add\project\knowledge\decisions\framework.md`（含 ADR-0031 与 ADR-0030 对照段）
- `.add\project\knowledge\design-workspace\identity-auth\design.md`
- `internal\recovery\runtime.go`
- `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r03\report.md`
- `.spec\wip\recovery-runtime\evidence-b13j-d06\review-sec-r03\report.md`
- `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r10\report.md`
- `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r10\hashes.txt`
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\lead-authorization-r12.md`
- `.add\project\knowledge\tasks\tasks.md`（仅范围外观察）
- `.add\project\knowledge\decisions\README.md`（仅索引观察）
- `.codex\skills\add-code-review\SKILL.md`、`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`

## 3. 原 finding 逐条判定

### F001（原 BLOCKER）— 已闭合（附接受受限）

**结论**：已具备独立追溯行与可审计变更范围；不是仅改号。

- `:1191` 是 r09 独立追溯行，登记触发、授权、`r08` 未产出跳号事实、逐项锚点 (i)–(vi)、显式接受限制与 r09 改后基线 `272,588 B / 1,271 行 / D8DBA64B…`。
- `:1192` 是 r10 独立追溯行，登记两份 r03 报告及其哈希、授权来源、逐项 (i)–(xi)、未改段落清单、改前基线，并把改后值外置到 `hashes.txt`，避免自指。
- 作者 `hashes.txt` 的 11 行清单与 r10 追溯行 (i)–(xi) 对应，含 `:1082`、`:1084`、`:1118`、`:1129`、`:1147`、`:1166`、`:1174`、`:1176`、`:1190`、`:1191`、`:1192`；其余段内行按作者记录逐字节相同。
- 本评审独立复算：全文 `285,639 B / 1,273 行`、ADR 段 `59,129 B / DD1A…`、前缀 `205,336 B / 7CFF…`、后缀 `21,174 B / AA9E…`；改前/改后算术 `+13,051 B / +2 行` 自洽。
- **接受受限**：HEAD 不含 r09 前像，无法独立复算 r09→r10 的逐行 diff；作者 `PREFIX_BYTE_EQUAL=True`／`SUFFIX_BYTE_EQUAL=True` 只能作为作者记录引用。该限制如实保留，不判 F001 仍开放，但不得把它写成“已独立复算全部变更”。
- 另见 LOW `ADR-R04-02`：未改段落补集漏列解除条件第 2/4 条，影响审计清单完整性。

### F002（原 BLOCKER）— 已闭合（附 LOW `ADR-R04-03`）

**结论**：三签角色与 design r70 实质一致，三席位互不代签，`witness_key_id` 不再被写成回执签名公钥唯一来源。

- **producer**：`:1118` 明定以 `receipt_anchors.public_key` 签发 `JCS({"receipt": R10, "attestation": A6})`；`R10` 为 12 字段回执删除 `signature`／`audit_record_ref` 后的 10 字段对象，`A6` 为 7 字段 attestation 删除 `signature` 后的 6 字段对象。design `:3988` 的 envelope、字段删除规则与验签公钥来源一致。
- **verifier**：`:1118` 明定以 `keys[verifier_key_id].public_key` 签发 8 字段载荷（`anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id`），且该 key 的 `principal_id` 必须等于 `verifier_principal_id`；design `:4059` 与 r66.2 同形。
- **witness**：`:1118` 明定以 `receipt_anchors.witness_key_id` 对应公钥签发 5 键 head（`anchor_ref, sequence, issued_at, head_sig_alg, head_sig`）删除 `head_sig` 后的 JCS；design `:4033`／`:4060` 与 r66.2 规定 `witness_public_key` 必须等于 `keys[witness_key_id].public_key`，且 `witness_key_id` 不是回执签名公钥唯一来源；`:4062` 的 V10/anchor_ref 链支持 V10 前验证方签名与 anchor_ref，`:4063` 是 r65.6 ENV 段，不承担 witness 公钥来源（任务书给出的邻近行号只作定位参照，不据此改判）。
- 三席位互不代签、`witness_key_id` 非唯一来源均已明文；未发现回退。
- LOW `ADR-R04-03`：`:1118` 首句“三把 key 的登记并存于 `receipt_anchors` 与 `keys`”对 verifier 不逐字成立（verifier 公钥只在 `keys`），紧随括号已澄清，但建议改写以防实现者误找。

### F003（原 HIGH）— 范围外观察项，不判 CHANGES REQUIRED

- 现场 `tasks.md:3186` 仍为 `- [ ] identity-auth-S4.24a ...`；`:3220` 仍称 `adr-0031-r07` 为现行修订并称“门禁解除”，与 ADR `:1084`／`:1129`／`:1176` 的“现行 r10 两路接受未取得、S4.24a 不得启动”冲突。
- `tasks.md` 现场哈希已漂移为 `CA20577B…`，不是任务书给定的 `4BD724A9…`；该漂移由另一会话造成，本评审只登记。
- 按用户边界，F003 只登记为范围外观察项，不据此判 CHANGES REQUIRED，不代改 `tasks.md`，不更新任务状态。

### F004（原 HIGH）— 检查点集合已闭合；同段新增 MEDIUM `ADR-R04-01`

**结论**：撤销/TOCTOU 检查点集合已补齐 P6，且与 design 的检查点集合一致；但“失败位置与错误分类”句存在新的决策级矛盾，故另立 `ADR-R04-01`。

- `:1118` 明列 `P3`、`P6`、`R1`／`V1` 的 `prepared(ctx)`、`R2`／`R4`／`R6`、`V2`／`V4`／`V6`、`V7`／`V8`／`V10` 前各一次。
- design `:4037` 的 P1–P6、R1–R8、V1–V11 与之一致；`P6` 明确在 `Journal.Begin` 前重读批准文件精确字节并与快照比对；`R1`／`V1` 的 `prepared(ctx)` 保留；`V7`／`V8` 前是追加拒绝点，明写“只增加拒绝点、不放宽任何既有检查”。
- 未发现删点或“单一全局替代”。但 `:1118` 的后果句把 `R1/R2/R4/V1/V2/V4` 失败写成“不进入 Journal.Begin”，与 design/代码时序冲突，详见 `ADR-R04-01`。

### F005（原 MEDIUM）— 已闭合

- `:1118` 定义 `generation` 为可复算三元：`approval_generation` = 批准文件精确字节 SHA-256（预检快照记录，`revalidateApproval` 重读比对）；`trust_root_generation` = 信任根文件精确字节 SHA-256，必须等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`；`anchor_generation` = `receipt_anchors` 中该 `anchor_ref` 条目 JCS 规范字节的 `sha256:` 摘要，必须等于 policy 与回执的 `trust_anchor_sha256`。
- design `:4373` 的 r67.6 三元定义与 ADR 同文；`receipt_anchors` 的 15 项 keyset 无 `revocation`／`valid_until`，ADR 明写不得查找不存在字段，正确。
- 可读性建议（不编号）：把“三项同时逐字节相等”改写为“三项各自与对应快照/ENV/policy 值逐字节相等”，避免歧义；当前定义已足以实施，故不判 finding。

### F006（原 MEDIUM）— ADR 层已闭合；design 侧陈旧残留为观察项

- `:1118` 五处等式字段名与条件一致且逐字节相等：`receipt_anchors.anchor_ref` = 批准记录 policy 的 `tool.receipt.trust_anchor_ref` = 回执 `trust_anchor_ref` = 审计记录 `anchor_ref` = 见证 head `anchor_ref`。
- policy 的 `trust_anchor_ref` 明定为必填，缺键、空串或字段名不符即 closed（启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`），不保留“该键可缺”分支。
- design `:4153` 的 policy 13 项 keyset 含 `trust_anchor_ref`，`:4161` 缺字段返回 `ErrAdmissionBinding`，支持 ADR 的必填语义。
- **观察项**：design `:4031`／`:4062` 仍写“policy 中的 `anchor_ref`（若该键存在）”，与本 ADR 和 design policy keyset 不一致；作者已在 r10 报告中披露，属 design 写集，不在本报告修改范围。不据此判 ADR F006 仍开放。

### F007（原 MEDIUM）— 已闭合

- `:1118` 新增“见证术语消歧”：本 ADR 的“见证”只指 restore 回执的**外部锚见证**（Ed25519 签 5 键 head、持久单调 `sequence`、`receipt_anchors.witness_*`）；ADR-0030 的 **anchor revision 见证**是 `(w, r, token)` 状态机（ADR-0030 `:1005`–`:1027`，`:1009`）。
- 两者对象、状态、endpoint、key 相互独立，明写不得共用 key／endpoint／记录／推进通道，一套推进不得替代另一套；并明确 `anchor_ref` 不是 ADR-0019 `JournalHeadAnchor`，不承担 CAS 头锚点职责。分域清楚。

### F008（原 LOW）— 已消除

- `:1174` 已改为“据此**收紧**约束 8 解除条件第 6 条的判据……不是单纯同步”，并列出 r10 对 verifier 与六席 `person_ref` 的收紧。原“只同步”表述不再存在。
- 关联的清单补集问题另见 `ADR-R04-02`，不影响 F008 的语义闭合。

### F009（原 LOW）— 已消除

- `:1191` 已将“`adr-0031-r08` 未产出，跳号至 r09”定稿，不再保留“待评审方确认”。
- `:1129` 直接引用 d08 的 `review-arch-r01` 与 `review-sec-r01`；本评审复算两份报告分别为 `9,964 B / EC936100…` 与 `18,884 B / DA757AD1…`，与 ADR 引用一致。

## 4. 新 finding

### ADR-R04-01（MEDIUM）— 检查点失败的“位置/错误分类”句与 design 时序冲突

**现状**：`:1118` 写“任一重验失败（撤销、过期、generation 漂移或读取失败）即 closed／`restore_unknown`：`P3`／`P6`／`R1`／`R2`／`R4`／`V1`／`V2`／`V4` 失败返回 `ErrAdmissionClosed` 且**不进入 `Journal.Begin`**；`R6`／`V6` 及 `V7`／`V8`／`V10` 前失败按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)` 处理、由调用方按 `restore_unknown` 处置”。

**反证一（位置）**：design `:4037` 明定 `R1`–`R8` 属 `Replace` 路径、`V1`–`V11` 属 `VerifyRestored` 路径；`internal/recovery/runtime.go` 的 `:1617` `Journal.Begin` 在 `:1584` `Restore.Replace` 与 `:1634` `Restore.VerifyRestored` 之前。`R1/R2/R4/V1/V2/V4` 失败时 Begin 已经发生（或根本不经过 Begin 路径），不可能满足“且不进入 Journal.Begin”。

**反证二（P3）**：design `:4037` 的 `P3` 是首次预检，读取批准文件、信任根、目标 DSN 并构造不可变快照；P3 没有前快照，因此“generation 漂移”在 P3 不是可判条件。要求“每个重验点重读批准文件、信任根与 `receipt_anchors`，重算 generation”也不能适用于 P3，否则会把首次预检写成重验。

**反证三（错误分类）**：design `:4377` 将“缺键、文件缺失或不可读”归 `ErrAdmissionIncomplete`，只有未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked` 才归 `ErrAdmissionClosed`。`ADR` 把“读取失败”统一并入 Closed，和 design 的分类冲突；实现者按 ADR 写会把不可读/缺键错误错误降级为 Closed。

**影响**：决策级“可核判据”自相矛盾。实施者若按字面把 R/V 检查移到 Begin 前，会改变 design 的检查点拓扑；若保留原位，该后果声明永远不成立；读取失败分类会导致实现与设计的错误处理分叉。

**触发场景**：按 ADR 枚举实现 revalidate 后果；或对 P3 做“generation 漂移”判定；或让 `R1/R2/V1/V2` 在 Replace/VerifyRestored 路径中失败。

**建议修复**：按路径分写后果：`P6`（及 Prepare 内重读）失败 → Begin 前 `ErrAdmissionClosed`；`R`/`V` 检查失败 → 按其路径既有错误分类（缺键/不可读 `ErrAdmissionIncomplete`，撤销/过期 `ErrAdmissionClosed`，启动后按 `restore_unknown`）；删除 P3 的“generation 漂移”表述（P3 只构造快照与首次合取）；把“每个重验点重读”改为“P6 及 P6 之后每个检查点”。

**验证**：逐路径列出 `(检查点, Begin 前/后, 失败错误, 是否推进 sequence)`，与 design `:4037`／`:4373`／`:4377` 及 `runtime.go` 时序逐项对照，确保没有一句可作两种读法。

### ADR-R04-02（LOW）— “未改段落”补集漏解除条件第 2/4 条

- `:1174` 与 `:1192` 均写“不改变……解除条件第 1/3/5/7/8/9 条”；r10 实际只改第 6 条，未改第 2/4 条，补集应为第 1/2/3/4/5/7/8/9 条。
- 来源是沿用 r07 行模板（r07 时第 2/4 条被改），不影响 (i)–(xi) 锚点，但削弱“未改段落”作为审计补集的完整性。
- 建议：两处改为“除第 6 条外，解除条件第 1–5、7–9 条未改”。

### ADR-R04-03（LOW）— 首句“三把 key 的登记并存于 receipt_anchors 与 keys”对 verifier 不逐字成立

- `receipt_anchors` 15 项只有 `verifier_principal_id`／`verifier_key_id`，没有 verifier 公钥；公钥只在 `keys[verifier_key_id].public_key`。紧随括号已逐席位澄清，但首句可误导实现者在 `receipt_anchors` 查找 verifier 公钥。
- 建议：改为“三个席位的身份/密钥标识登记在 `receipt_anchors`，公钥按席位分别取自 `receipt_anchors.public_key`、`keys[verifier_key_id].public_key`、`receipt_anchors.witness_public_key`”。

### ADR-R04-04（LOW）— 三签/keyset/检查点集合固定引用 `identity-auth-design-ir01-r66`

- `:1118` 的三签模型、15 项 keyset、检查点集合均固定引用 design r66；现行 design 是 r70。相关合同在 r67–r70 未改，但固定修订号引用易腐。
- 建议：改为 design 条款锚（如 r66.2／r65.5／r66.5 段名）并注明“现行 r70 未改”，或在每次 design 修订时复核该引用。

## 5. 无回退底线核查

| 底线 | 证据 | 结论 |
|---|---|---|
| 不变量 1–6 | `:1131`–`:1149`；r10 只在第 6 条补 verifier／external_verifier／六席 `person_ref` 独立性 | 未削弱，第 6 条为收紧 |
| 约束 1–9 | `:1153`–`:1170`；第 6 条为收紧，其余未改 | 未削弱 |
| 首期不提供生产 restore 工具与批准主体 | `:1084`、`:1090`、`:1176` | 未削弱 |
| restore-v1／verify-v1 严格合同 | `:1086` | 未削弱 |
| no-follow 稳定 fd 与 fd-exec | `:1115`、`:1117`、`:1141` | 未削弱 |
| 最小 ENV allowlist | `:1117`、`:1144` | 未削弱 |
| Begin 前 closed | `:1086`、`:1090`、`:1118` | 未削弱；但 `ADR-R04-01` 的后果句需按路径修正 |
| restore_unknown 保守处置 | `:1118`、`:1162` | 未削弱 |
| Windows 生产 unsupported | `:1115`、`:1164` | 未削弱 |

门禁分级核查：`:1122` 决策接受级、`:1124` 卡证据级、`:1126` 生产解除级分开表述，未把第二级事实取证写成启动前置；`:1084` 与 `:1192` 明写 r10 两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动。未发现门禁自指、二级/三级越界声称，或把 unknown 写成已具备。本报告不声称生产已解锁、`enter`／`resume` 已解锁、约束 8 已满足、S4.24a 可关闭或可派发实施。

## 6. 开放项、待决定与建议

### 开放项（本轮架构结论）

- **ADR-R04-01（MEDIUM）**：检查点失败位置与错误分类句需按路径修正；是 `Verdict: CHANGES REQUIRED` 的唯一直接依据。
- **ADR-R04-02（LOW）**：未改段落补集漏第 2/4 条。
- **ADR-R04-03（LOW）**：三把 key 登记句对 verifier 不逐字准确。
- **ADR-R04-04（LOW）**：固定引用 design r66，建议改条款锚。

### 待决定（非本报告架构接受的阻断依据）

- `:1192` 的 r10 写入依据是 Lead 直接指令，而 `lead-authorization-r12.md` §3 明示不含 `framework.md` 写入；建议 Lead 补记授权台账。只作链路待决定，不据此判架构 CHANGES REQUIRED。
- design `:4031`／`:4062` 的 policy “若该键存在”陈旧表述，由 Feature-Designer 清理；作者已披露，不在本写集。
- `decisions/README.md:39` 仍写 `adr-0031-r09`，属索引维护项，不改本报告对象。
- `tasks.md:3220`／`:3186` 与现行 ADR 冲突，属范围外 Task-Planner 事项；只登记，不代改。

### 建议（非缺陷或已并入 finding）

- F005 三元句建议写成“三项各自与对应快照/ENV/policy 值逐字节相等”。
- `ADR-R04-01` 修复时同步检查 P3 的首次预检措辞与 R/V 路径的错误分类表。
- 三签句按 `ADR-R04-03` 改写，避免在 `receipt_anchors` 查找 verifier 公钥。

## 7. NOT RUN

以下均未实测，不得因设计文字存在而推断 PASS：

- `SEC-F007` 安全专项复核：NOT RUN（本轮为架构轴）。
- Linux/POSIX 隔离、`fd-exec`、no-follow 稳定 fd 的真实系统调用行为：NOT RUN。
- 真实 PostgreSQL、外部见证、HSM、审计 sink、信任根轮换与真实旧备份演练：NOT RUN。
- Go build／unit test／integration test／E2E：NOT RUN。
- 生产 restore 解锁、`enter`／`resume` 解锁、约束 8 满足、S4.24a 关闭或派发：NOT RUN，且本报告不作此类声称。

## 8. 实际执行命令

- `Get-FileHash -Algorithm SHA256 -LiteralPath`：对 `framework.md`、快照、r03 两报告、作者报告/指纹、design r70、`tasks.md`、`README.md`、授权台账计算 SHA256。
- `[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)`：读取 ADR、design、runtime、评审报告与基线文本。
- `[IO.File]::ReadAllBytes($p)` + 行首偏移切片 + `MemoryStream` + `SHA256`：复算前缀、ADR 段、后缀的 raw bytes 长度与哈希。
- `Get-Content -Encoding UTF8`：读取作者 `hashes.txt` 的改前/改后字节账。
- `rg`／`Select-String`：定位 ADR、design、runtime 与评审报告中的交叉引用（含 `NewRuntime`、`P6`、`anchor_ref`、`witness_key_id`、`Journal.Begin` 等）。

## 9. 计数与结论

- 原 finding：F001 已闭合（附接受受限）；F002 已闭合；F003 为范围外观察项；F004 检查点集合已闭合但新增 `ADR-R04-01`；F005、F006、F007 已闭合；F008、F009 已消除。
- 本轮新增开放：**BLOCKER 0 / MEDIUM 1 / LOW 3**。
- 开放项总数：**4**（`ADR-R04-01` 至 `ADR-R04-04`）。
- 由于 `ADR-R04-01` 使决策级检查点后果在位置与错误分类上不可唯一实施，不能判决策接受级通过；F003 未被用作判据。

Verdict: CHANGES REQUIRED