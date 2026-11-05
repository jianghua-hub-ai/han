# ADR-0031 `adr-0031-r11` 架构独立复审报告（r05）

**日期**：2026-11-02
**评审角色**：add-reviewer（架构轴）
**对象类型**：ADR（只判“决策接受级”，不判实施级、不判生产解除级）
**工作区**：`D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
**本轮性质**：对 r04（CHANGES REQUIRED，0 BLOCKER／1 MEDIUM／3 LOW）四项原 finding 的修复复审；作者 r11 自述只修这四项、未改决策实质，本报告逐条独立复核该自述。

## 1. 冻结对象、快照与基准

| 项 | 路径/说明 | 字节 | 行数 | SHA256 |
|---|---|---:|---:|---|
| 被审对象 | `.add\project\knowledge\decisions\framework.md` | 292,606 | 1,274 | `9723F92FB53B076D228B5FA392CEC07CA0F74ED460524EBB7F00AC9202EB53CD` |
| 冻结快照 | `%TEMP%\framework-adr0031-r11-freeze-9723F92F.md`（同字节） | 292,606 | 1,274 | `9723F92FB53B076D228B5FA392CEC07CA0F74ED460524EBB7F00AC9202EB53CD` |
| 改前基线 | `%TEMP%\framework-adr0031-r10-freeze-B432E44A.md`（r10 冻结） | 285,639 | 1,273 | `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050` |
| 架构 r04 基准（本评审上一轮） | `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r04\report.md` | 22,635 | 225 | `F17AC89A9970BC89FF5B96625531DD48BBA6ECA5ECB45B695D790465613A3DBC` |
| 安全 r04 基准 | `.spec\wip\recovery-runtime\evidence-b13j-d06\review-sec-r04\report.md` | 24,983 | 226 | `C1AE35DF51F9D1CFAA14AA5B2E39EC33F9DD80CD428D507F0CE316FE1DABB9CE` |
| 作者 r11 报告（线索） | `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r11\report.md` | 5,318 | 71 | `7BEBB93235C6910C97F9E0D9902BEA7A921C019107C8EFFD663559D55F0E4171` |
| 作者 r11 指纹（线索） | `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r11\hashes.txt` | 1,391 | 51 | `B055D80E7A954B03841BACCE94AAA4D881A72C5E4916EA19888C67BE104FA099` |
| design 对照 | `.add\project\knowledge\design-workspace\identity-auth\design.md`（r70） | 1,103,152 | 5,476 | `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254` |
| 代码时序对照 | `internal\recovery\runtime.go` | 59,138 | 1,969 | `4C0946AF7C29DE472CAE4848CE2F55A68288003071957FB62B66BFF3B59604C4` |
| 范围外观察 | `.add\project\knowledge\tasks\tasks.md`（现场已漂移） | 904,142 | 3,312 | `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C` |
| 索引观察 | `.add\project\knowledge\decisions\README.md` | 21,613 | 110 | `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2` |

现场 `framework.md` 与快照同字节同哈希，判读期间无对象漂移。ADR-0031 标题 `:1080`、修订行 `:1082` 为 `adr-0031-r11`、状态段 `:1084`、追溯行 `:1192`（r10）与 `:1193`（r11）；ADR-0032 起于 `:1197`，判读段取 `:1079`–`:1196`。`tasks.md` 与 `README.md` 只作范围外观察，不改变本报告对冻结 ADR 的判读，也不作为结论依据。

## 2. 字节账与变更集（全部自算，不采信作者数值）

### 2.1 编码与行尾

- 现行全文：292,606 B／1,274 行；CRLF 1,274、bare LF 0、无 BOM、末尾 `0D 0A`。
- 改前基线：285,639 B／1,273 行；CRLF 1,273、bare LF 0、无 BOM、末尾 `0D 0A`。其 SHA256 与我 r04 现场复算的 r10 对象一致，可认定为真实前像（不是中间态或重编码产物）。
- 差值：`+6,967 B`、`+1` 行，行尾形态与 BOM 均未变。

### 2.2 逐行 diff 与配平

以 `ReadAllLines` 对基线与现行逐行精确比较（含行内空格与全角字符），首差异行 `:1082`、公共尾 81 行（基线 `:1193`–`:1273` ≡ 现行 `:1194`–`:1274`）：

| 行 | 性质 | 字节增量 | 内容要点 |
|---|---|---:|---|
| `:1082` | 改 | 0（`r10`→`r11` 单字符等长） | 修订行升为 `adr-0031-r11` |
| `:1084` | 改 | 0（同上，单字符） | 状态段同步为 r11；保留用户 D03 接受事实；写明 r11 两路接受未取得、S4.24a 及任何实施/验证卡不得启动 |
| `:1118` | 改 | +1,740 | 检查点失败后果按路径分写；删除 `P3` 的 generation 漂移判定；重读口径限定 `P6` 及之后；三签公钥来源按席位；固定 r66 引用改条款锚 |
| `:1174` | 改 | +504 | r10 范围句补集更正为“除第 6 条外，解除条件第 1–5、7–9 条未改”，追加 r11 范围句 |
| `:1192` | 改 | +270 | 仅行末追加 r11 更正标记（前 2,790 字符与基线逐字符相同，append-only） |
| `:1193` | 新增 | +4,451（另加 CRLF +2） | r11 独立追溯行 (i)–(ix)、未改段落、改前基线、上游残留观察项 |

合计 `1,740 + 504 + 270 + 4,451 + 2 = 6,967 B`，与全文差一致；无其他行变化（其余 1,267 行逐字符相同）。作者的 `change lines=1082,1084,1118,1174,1192`、`added_line=1193` 与本评审独立 diff 一致。

### 2.3 段外不变与前缀/段/尾哈希（raw bytes，行首偏移→EOF，不重编码、不归一）

| 区段 | 字节 | SHA256 |
|---|---:|---|
| 前缀 `:1`–`:1078`（含末尾 CRLF） | 205,336 | `7CFF3F32B15D79316D07BDE2BDE8A2C6A74C835B8DAE78158A247FC4E328F1F1` |
| ADR-0031 正文 `:1079`–`:1193`（含末尾 CRLF） | 66,096 | `91C1150A48183302520ABADFCB3D81BFBFE70A2AC12410FE3CA7EC9EAADA10CA` |
| 尾段 `:1194`–`:1274` | 21,174 | `AA9E7E183B6D0A9301D736DEE890392CCAE4619C0B942216FBA03F32134FD599` |
| （判读段整体 `:1079`–`:1196`，含分隔空行与 `---`） | 66,105 | `97C58E35018A470443C532F2774E8FEB5A438288AB4296272B223644BB4EE83C` |

- `205,336 + 66,096 + 21,174 = 292,606` ✓。
- 前缀与尾段分别与基线的 `:1`–`:1078`、`:1193`–`:1273` 逐字节相等（前缀哈希与 r04 报告同值；尾段哈希与 r04 报告同值，仅行号顺延）。
- 作者 `hashes.txt` 的段外口径（prefix 109,097 chars／205,357 B、suffix 11,152 chars／21,165 B）与“`:1`–`:1079` + 行尾 CRLF”“`:1197` 起至 EOF”两种切法一致，可复算；其 `closeout_byte_delta=+15`（收口前中间态）无留存字节，本评审**不可复算**，只记为接受受限，不据此判 finding。

### 2.4 括号平衡

全角括号全文 540/540；本次改动六行逐行平衡：`:1084` 1/1、`:1118` 32/32、`:1174` 8/8、`:1192` 11/11、`:1193` 20/20（`:1082` 无括号）。ASCII 括号在各行亦成对。

## 3. 方法、两轮检查范围与只读声明

### 3.1 方法

- 完整读取 `.codex\skills\add-code-review\SKILL.md`（120 行），按 `objectType=adr` 维度执行“实际问题/约束、候选与不选理由、代价、适用范围、确认状态、影响消费者与迁移/退出条件”检查，并按“复审引用原 finding ID、逐项给关闭/仍存在/受限”的闭环要求组织本报告。
- 按加载规则读取 `.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\guidance\decisions-guide.md`；代码提交/构建/覆盖率条款不强套本 ADR 对象。
- 只判决策接受级：形态、最小字段、可核判据、否证条件是否唯一可实施；不以“文档存在”“作者声明”代替证据；不替作者修复、不采用决定、不改任务状态。

### 3.2 两轮检查范围

1. **第一轮（对象与变更集）**：复算全文/快照/基线哈希、字节数、行数、CRLF/bare LF/BOM/末尾字节；逐行 diff 确认改动集与段外不变；核对 `:1082`、`:1084`、`:1118`、`:1174`、`:1192`、`:1193` 全文；核对全角括号平衡；核对作者 `hashes.txt` 各口径。
2. **第二轮（契约与消费者）**：把 `:1118` 的检查点/错误分类逐项对照 design r70 `:4037`（P1–P6／R1–R8／V1–V11）、`:4373`（读取时机与 Prepare 重读）、`:4377`（失败分类）、`:4137`（工具启动后 `errors.Join`→`restore_unknown`）、`:3672`／`:3886`／`:3899`（resume 零新 Begin／不再 Replace）；对照 `internal\recovery\runtime.go:1516`–`:1684`（`Enter`/`Resume`/`enter`/`begin`/`resume` 实际调用时序，`Journal.Begin` 唯一调用点 `:1617`）；回归复判 r03 F001/F002/F004–F009 与无回退底线；扫描新缺陷（unknown 被写成已具备、条件式“若该键存在”、门禁自指、越界声称）。

### 3.3 只读声明

本评审未修改 `framework.md`、`design.md`、`tasks.md`、`decisions/README.md`、ADR、`cmd/**`、`internal/**`、测试或任何既有 `review-*`／`design-revision-*`；未删除、移动、归档任何文件；未执行任何 Git 写操作。唯一写入为本目录 `report.md` 与 `hashes.txt`。实际判读过的文件：

- `.add\project\knowledge\decisions\framework.md`（ADR-0031 全段 `:1079`–`:1196`，含 r10/r11 追溯行）
- 冻结快照 `%TEMP%\framework-adr0031-r11-freeze-9723F92F.md`、基线 `%TEMP%\framework-adr0031-r10-freeze-B432E44A.md`
- `.add\project\knowledge\design-workspace\identity-auth\design.md`（`:3672`、`:3886`、`:3899`、`:4037`、`:4070`、`:4137`、`:4373`、`:4377`）
- `internal\recovery\runtime.go`（`:1516`–`:1684`）
- `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r04\report.md`
- `.spec\wip\recovery-runtime\evidence-b13j-d06\review-sec-r04\report.md`（只作基准哈希，不引用其结论代替本轮判断）
- `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r11\report.md`、`hashes.txt`（只作线索）
- `.add\project\knowledge\tasks\tasks.md`（仅 `:3186` 范围外观察）、`.add\project\knowledge\decisions\README.md`（仅索引观察）
- `.codex\skills\add-code-review\SKILL.md`、`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\guidance\decisions-guide.md`

## 4. 原 finding 逐条判定

### ADR-R04-01（MEDIUM）— **仍存在（部分闭合）**

**已闭合的部分**（逐项对 design `:4037`／`:4373`／`:4377` 复核）：

- `P3` 的 generation 漂移判定已删除：`:1118` 现写“`P3` 只读取批准文件、信任根与目标 DSN 并构造不可变快照、完成首次合取（无前像可比，故不作 generation 漂移判定，也不属重验点）”，与 design `:4037`（P3 为首次预检、P6 才做重读比对）一致。
- 重读口径已限定为“`P6` 及 `P6` 之后每个检查点”，不再要求 P3 重算 generation。
- 失败后果已按路径分写：`P6` 及 `Prepare` 内重读在 `Journal.Begin` 之前失败 → `ErrAdmissionClosed` 且不进入 `Journal.Begin`（对应 design `:4037` P6“漂移 → Closed”、`:4373` Prepare 在 Begin 前重读）；`R1`–`R8`（Replace）与 `V1`–`V11`（VerifyRestored）按路径既有分类，判定顺序为“先看原因（缺键/文件缺失/不可读 → `ErrAdmissionIncomplete`；未到 `valid_from`／已过 `valid_until`／`revoked` → `ErrAdmissionClosed`；预检后漂移且工具未启动 → `ErrAdmissionClosed`），再看是否越过工具启动点（越过则 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)` → `restore_unknown`）”，与 design `:4377`、`:4137` 一致；`R6`／`V6`／`V7`／`V8`／`V10` 被点名归入“已越过工具启动点”一列，正确。
- r10 的“R/V 失败不进入 Journal.Begin”错误写法已被删除（`:1118` 现文无该句式）。

**仍存在的部分（一条）**：`:1118` 写“`R1`–`R8` 属 `Replace` 路径、`V1`–`V11` 属 `VerifyRestored` 路径，**`enter`／`resume` 中二者的调用均在 `Journal.Begin` 完成之后**”。该句把 R 与 V 的调用归属同时套到 `enter` 与 `resume` 两条路径，但：

- `resume` 零新 Begin：design `:3672`“Resume 不新 Begin，不盲 Replace”、`:3886`“同一精确 binding/receipt 全部可证才 resume，不重新 Begin／…或再次 Replace”、`:3899`“零新 Begin”；`runtime.go:1537` 注释“never Begin/Replace”，`resume`（`:1647`–`:1683`）依次为 `acquire`→`readControlled`→`Prepare`→`validateRestorePlan`→`refresh`→`VerifyRestored`，无 `Journal.Begin`；全文 `Journal.Begin(` 唯一调用点 `:1617` 只在 `begin` 内，而 `begin` 只由 `enter` `:1577` 调用。
- `resume` 也不调用 `Replace`，因此“二者（R 与 V）的调用”在 `resume` 中不成立。
- 该句最宽松的读法（“R/V 在时间上晚于该 run 既有 Begin 的完成”）可自洽，但另一种读法（“`resume` 会经过 `Journal.Begin`，或在 Begin 之后调 R/V”）与 design 的“零新 Begin／不再 Replace”直接冲突，属 r04 要求的“不得有可作两种读法的句子”未达成。实施者若按后者在 `resume` 补 Begin/Replace，会破坏 restore_unknown 保守处置与 design 的写路径约束。

详见本轮新 finding `ADR-R05-01`（MEDIUM，与本题同源）。另 `ADR-R05-02`（LOW）为该段“P6/Prepare 一律 `ErrAdmissionClosed`”的精度残余。

### ADR-R04-02（LOW）— **已闭合**

- `:1174`：补集已改为“除第 6 条外，解除条件第 1–5、7–9 条未改”，并追加 r11 范围句（“r11 未改不变量 1–6、约束 1–9（含解除条件九条）的任何字节”）。逐行 diff 证明 `:1174` 之外的不变量/约束行确未变，补集与事实一致。
- `:1192`：严格 append-only——前 2,790 字符与基线 r10 行逐字符相同，行末追加 270 B 的更正标记（“本行为 r10 时点记录，其余表述不改写；其‘未改段落’补集漏列解除条件第 2／4 条……见下一条 r11 追溯行”），未改写历史事实。
- `:1193` 追溯行同时载明同一补集口径，三处一致。

### ADR-R04-03（LOW）— **已闭合**

`:1118` 已改为按席位表述：“三个席位的身份／密钥标识登记在 `receipt_anchors`（`producer_key_id`／`verifier_key_id`／`witness_key_id`），公钥按席位分别取自 `receipt_anchors.public_key`、`keys[verifier_key_id].public_key`、`receipt_anchors.witness_public_key`”，并明确 `witness_key_id` 只是见证公钥标识、**不是**回执签名的公钥唯一来源。原“三把 key 的登记并存于 `receipt_anchors` 与 `keys`”的误导性首句已不存在；verifier 公钥只在 `keys` 这一点已清楚。

### ADR-R04-04（LOW）— **已闭合（附 `ADR-R05-03` 观察）**

- `:1118` 中 `identity-auth-design-ir01-r66` 出现次数为 **0**；全文检索该串仅命中历史追溯行 `:1192`、`:1193`（作为“r10 当时引用了 r66”的历史描述，属不可改写的历史记录），不构成仍以固定修订号作现行引用。
- 现文改用条款锚 `r64.5`／`r64.10`／`r65.5`／`r65.13`／`r66.2`／`r66.5`（基线 `:1118` 原有 6 处 r66 引用：2 处 `identity-auth-design-ir01-r66`、3 处 `design r66`、1 处 `design r66.2`，与 `:1193` 的“六处”表述相符）。
- 现文明写“现行 `identity-auth-design-ir01-r70` 未改这些合同……故本 ADR 不据此声称已复核 r67–r70 的全部改动”，未虚称已复核；本评审独立检索 design r70：`r68.`／`r69.`／`r70.` 仅命中 `:4072`，与所述一致。
- 残余：支持句“仅命中 `:4072` 的行号映射说明，未命中任何编号条款”不精确——`:4072` 正文含 `r68.1`–`r68.8`、`r69.1`–`r69.3`、`r70.1`–`r70.3`。结论（六条条款锚未被改写）实质成立，故只记 LOW `ADR-R05-03`。

## 5. 回归复判（r04 已闭合项是否仍闭合）

字节级证据：除 `:1082`/`:1084`/`:1118`/`:1174`/`:1192` 五行改写与 `:1193` 新增外，全文其余 1,267 行逐字符相同，前缀与尾段逐字节相等。因此 r04 已闭合项在字节层不可能回退；逐项复核其语义亦未变：

| 原项 | 结论 | 证据 |
|---|---|---|
| F001 独立追溯行与可审计变更范围 | 仍闭合 | `:1193` (i)–(ix) 与独立 diff 完全对应；改前基线 285,639 B／1,273 行／`B432E44A…` 真实可复算；改后值外置 `hashes.txt` 避免自指 |
| F002 三签角色与 design r70 实质一致 | 仍闭合 | producer `receipt_anchors.public_key` 签 `JCS({"receipt": R10, "attestation": A6})`；verifier `keys[verifier_key_id].public_key` 签 8 字段且 `principal_id=verifier_principal_id`；witness `receipt_anchors.witness_public_key`（经 `witness_key_id`→`keys` 绑定）签 5 键 head 去 `head_sig` 的 JCS；三席位互不代签 |
| F004 撤销/TOCTOU 检查点集合（P6 补齐、无删点、无单一全局替代） | 仍闭合（另见 ADR-R04-01 残余） | `:1118` 仍列 `P6`、`R1`/`V1` 的 `prepared(ctx)`、`R2`/`R4`/`R6`、`V2`/`V4`/`V6`、`V7`/`V8`/`V10` 前；`V7`/`V8` 明写“只增加拒绝点、不放宽任何既有检查” |
| F005 generation 可复算三元 | 仍闭合 | `approval_generation`＝批准文件精确字节 SHA-256；`trust_root_generation`＝信任根精确字节 SHA-256 且等于 ENV；`anchor_generation`＝该 `anchor_ref` 条目 JCS 字节 `sha256:` 且等于 policy/回执 `trust_anchor_sha256`；并明写 `receipt_anchors` 无 `revocation`/`valid_until` 字段、不得查找不存在字段 |
| F006 `anchor_ref` 五处等式 | 仍闭合 | 五处逐字节相等；policy `tool.receipt.trust_anchor_ref` 必填，缺键/空串/字段名不符即 closed（启动前 `ErrAdmissionBinding`，工具启动后 `restore_unknown`）；全文 ADR 段无“若该键存在” |
| F007 两种“见证”分域 | 仍闭合 | 回执外部锚见证（Ed25519 5 键 head、`receipt_anchors.witness_*`）vs ADR-0030 `(w, r, token)`（`framework.md:1005`–`:1027`，见 `:1009`）；不得共用 key/endpoint/记录/推进通道；`anchor_ref` 非 ADR-0019 `JournalHeadAnchor` |
| F008 “收紧”表述 | 仍闭合 | `:1174` 仍写“据此**收紧**约束 8 解除条件第 6 条……不是单纯同步” |
| F009 跳号定稿 | 仍闭合 | `:1191` 仍写“`adr-0031-r08` 未产出，跳号至 r09 的事实由本行定稿，不再保留‘待评审方确认’” |

## 6. 状态段（`:1084`）与追溯行（`:1193`）准确性

- `:1084`：与基线 r10 仅差 1 个字符（`r10`→`r11`）。内容保留“已接受（2026-11-01 用户 D03 指令授权 …）”、保留“不构成人的成果接收、独立评审、安全评审、生产装配批准或真实工具/备份验证”，并写明“决策接受级……随 `**修订**` 行的现行修订号自动生效”“本修订 `adr-0031-r11` 的两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动”。与事实一致，未把 unknown 写成已具备、未声称 PASS。
- `:1193`：(i)–(ix) 与独立 diff 完全对应（改动行恰为 `:1082`/`:1084`/`:1118`/`:1174`/`:1192` + 新增 `:1193`）；“未改段落”与本评审字节证据一致；“改前基线”值与真实快照一致；“上游残留观察项”描述 design `:4070` 的内容经本评审核对属实（该行确有“`P3`／`P6`／`R1`／`V1`／`R2`／`V2`／`R4`／`V4` 漂移 → `ErrAdmissionClosed`，且不进入 `Journal.Begin`”，并自述“B1 只作摘要、非穷举”）。该残留属 design 写集，不据此判 ADR 不通过，也不代改。

## 7. 新 finding

### ADR-R05-01（MEDIUM）— `:1118` 把 R/V 调用位置错误地合并写为“`enter`／`resume` 均在 `Journal.Begin` 完成之后”

- **位置**：`framework.md:1118`（`R1`–`R8`／`V1`–`V11` 后果段）。
- **现状**：原文“`R1`–`R8` 属 `Replace` 路径、`V1`–`V11` 属 `VerifyRestored` 路径，`enter`／`resume` 中二者的调用均在 `Journal.Begin` 完成之后，故其重验失败按该路径既有错误分类处置……”。
- **反证**：design `:3672`“Resume 不新 Begin，不盲 Replace”、`:3886`“不再重新 Begin／…或再次 Replace”、`:3899`“零新 Begin”；`runtime.go:1537` 注释“never Begin/Replace”，`resume`（`:1647`–`:1683`）无 `Journal.Begin`、无 `Replace`，`Journal.Begin` 全文唯一调用点 `:1617` 只在 `enter` 路径的 `begin`（`:1577`）内。
- **触发**：实施者按 `:1118` 实现 `resume` 的重验后果，或据此推断 `resume` 存在“Begin 前/后”分区。
- **影响**：决策文档与 design/代码事实冲突；若按字面在 `resume` 补 `Begin`/`Replace`，会违反“零新 Begin／不再 Replace”与 `restore_unknown` 保守处置（写路径风险）；若按宽松读法则该句仍可作两种读法，未满足 r04 的“无二义”要求。
- **建议**：改为“`R1`–`R8` 只属 `Replace` 路径（仅 `enter`）；`V1`–`V11` 在 `enter` 中位于 `Journal.Begin` 之后，在 `resume` 中零新 `Begin`、零 `Replace`，其失败只按‘失败原因 + 是否越过工具启动点’分类”。
- **验证**：逐路径表 `(检查点, 是否经 Begin, 工具启动点前/后, 失败错误, 是否推进 sequence)` 与 design `:4037`/`:4373`/`:4377`、`runtime.go:1573`–`:1683` 逐项对齐，`resume` 行的 “Begin” 列必须为“无（沿用既有记录）”、“Replace” 列必须为“不调用”。

### ADR-R05-02（LOW）— `:1118` 将 `P6`/`Prepare` 重读的“失败”一律写成 `ErrAdmissionClosed`

- **位置**：`framework.md:1118`。
- **现状**：“`P6` 及 `Prepare` 内的重读发生在 `Journal.Begin` 之前，失败返回 `ErrAdmissionClosed`，且不进入 `Journal.Begin`”。
- **依据**：design `:4377` 按原因分类——缺键、文件缺失或不可读 → `ErrAdmissionIncomplete`；未到/已过有效期、`revoked` → `ErrAdmissionClosed`；预检后字节漂移 → `ErrAdmissionClosed`（r62.3 修正）。design `:4037` 只把 `P6` 的**漂移**写成 Closed。
- **影响**：错误码契约粒度不一致（不影响 fail-closed，但影响调用方观测与设计一致性）。
- **建议**：限定为“字节漂移/快照不一致 → `ErrAdmissionClosed`，其余按 design `:4377` 分类”。
- **说明**：该粗粒度源自 r04 的建议措辞，本轮按 LOW 记录，不追认为新 HIGH。

### ADR-R05-03（LOW）— `:1118` 的检索断言“未命中任何编号条款”不精确

- **位置**：`framework.md:1118` 末段。
- **现状**：“全文检索 `r68.`／`r69.`／`r70.` 仅命中 `:4072` 的行号映射说明，未命中任何编号条款”。
- **事实**：检索确只命中 `:4072` 一行，但该行正文含 `r68.1`–`r68.8`、`r69.1`–`r69.3`、`r70.1`–`r70.3` 的编号条款（修订登记）。
- **影响**：结论（`r64.5`/`r64.10`/`r65.5`/`r65.13`/`r66.2`/`r66.5` 未被 r68–r70 改写）实质成立，但该支持句作为“未改”证据不严谨。
- **建议**：改为“仅命中 `:4072` 的修订登记段，未对上述条款锚作改写”。

## 8. 无回退底线核查

| 底线 | 证据 | 结论 |
|---|---|---|
| 不变量 1–6 | `:1131`–`:1149`（不在改动行内，逐字节未变） | 未削弱 |
| 约束 1–9（含解除条件九条） | `:1153`–`:1170`（未变） | 未削弱 |
| “首期不提供生产 restore 工具与生产批准主体” | `:1084`、`:1090`、`:1176` | 未削弱 |
| restore-v1／verify-v1 严格合同 | `:1086` | 未削弱 |
| no-follow 稳定 fd 与 fd-exec | `:1115`、`:1117`、`:1141` | 未削弱 |
| 最小 ENV allowlist | `:1117`、`:1144` | 未削弱 |
| Begin 前 closed | `:1086`、`:1090`、`:1118` | 未削弱（但 `P6`/`Prepare` 失败码见 ADR-R05-02；R/V 位置见 ADR-R05-01） |
| restore_unknown 保守处置 | `:1118`、`:1162` | 未削弱 |
| Windows 生产 unsupported | `:1115`、`:1164` | 未削弱 |
| 五处等式／三签／generation 三元／不声称抗 root／两种见证分域 | `:1118` | 未削弱 |
| 门禁分级与 unknown | `:1122`/`:1124`/`:1126` | 未削弱；未见门禁自指或二级/三级越界声称；`unknown` 仍如实保留 |

本报告不声称生产已解锁、`enter`／`resume` 已解锁、约束 8 已满足、`identity-auth-S4.24a` 可关闭或可派发实施，也不声称 `adr-0031-r11` 已取得两路接受。

## 9. 开放项、待决定与建议

### 开放项（本轮架构结论）

- `ADR-R04-01`（MEDIUM，仍存在／部分闭合）→ 具体为 `ADR-R05-01`：`:1118` 的 resume/Begin/Replace 归属句。
- `ADR-R05-02`（LOW）：`P6`/`Prepare` 失败码未按 design `:4377` 区分。
- `ADR-R05-03`（LOW）：`:1118` 检索断言措辞不精确。

### 待决定（非本报告架构接受的阻断依据）

- `:1192`（r10）与 `:1193`（r11）的写入依据均为 Lead 直接指令，而 `lead-authorization-r12.md` §3 明示不含 `framework.md` 写入；是否补记授权台账由 Lead 决定。
- `decisions/README.md:39` 仍写“现行修订 adr-0031-r09”，与现行 r11 不一致，属索引维护事项（decisions-guide 要求修订同步索引），不影响本报告对象判读。
- `tasks.md:3186` 仍为未勾选、`tasks.md` 现场哈希已漂移为 `CA20577B…`（非任务书给定 `4BD724A9…`），属范围外观察（N005），不据此判 CHANGES REQUIRED，不代改。

### 建议（非缺陷）

- `ADR-R04-01` 修复时把 `:1118` 的路径归属改成“逐路径一行”，并同时处理 `ADR-R05-02`，避免再产生位置/错误码的二义。
- design `:4070`（r65.13 B1 摘要）与 `:4037`/`:4377` 的冲突属设计侧写集，建议 Feature-Designer 在下一设计修订中收敛（本轮不代改，也不据此判 ADR 不通过）。

## 10. NOT RUN

以下均未实测，不得因设计/ADR 文字存在而推断通过：

- `SEC-F007` 安全专项复核：NOT RUN（本轮为架构轴；安全 r04 PASS 只作基准，不代替本轮判断）。
- Linux/POSIX 隔离、`fd-exec`、no-follow 稳定 fd 的真实系统调用行为：NOT RUN。
- 真实 PostgreSQL、外部见证、HSM、审计 sink、信任根轮换、真实旧备份演练：NOT RUN。
- Go build／vet／unit／integration／E2E、gofmt：NOT RUN。
- 生产 restore 解锁、`enter`／`resume` 解锁、约束 8 满足、S4.24a 关闭或派发：NOT RUN，且本报告不作此类声称。

## 11. 实际执行命令

- `Get-FileHash -Algorithm SHA256 -LiteralPath`：对 `framework.md`、两份 `%TEMP%` 快照、r04 两报告、作者 r11 报告与 `hashes.txt`、design r70、`runtime.go`、`tasks.md`、`README.md` 计算 SHA256。
- `[IO.File]::ReadAllBytes` + 逐字节扫描：统计 CRLF／LF／bare LF、BOM、末尾两字节。
- `[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)`：读取 ADR 段、design、runtime、评审与作者报告；逐行精确比较基线与现行（首/尾对齐 + 逐行 diff + 单行 append-only 判定 + 单行字节增量）。
- 行首偏移切片 + `SHA256`：按 raw bytes 复算前缀 `:1`–`:1078`、正文 `:1079`–`:1193`、判读段 `:1079`–`:1196`、尾段 `:1194`–`:1274`。
- `[regex]::Matches`：全角/ASCII 括号计数；`identity-auth-design-ir01-r66`、`r68.`/`r69.`/`r70.`、`若该键`、`已具备`/`已实测`/`已取得`/`PASS` 等关键词定位。
- `rg -n`／`Select-String` 等价检索：`Journal.Begin(`、`resume`、`r66.` 等交叉引用。

## 12. 计数与结论

- 原 finding 判定：`ADR-R04-01` 仍存在（部分闭合，MEDIUM）；`ADR-R04-02` 已闭合；`ADR-R04-03` 已闭合；`ADR-R04-04` 已闭合（附 `ADR-R05-03` LOW）。
- 回归：r03 F001／F002／F004／F005／F006／F007／F008／F009 仍闭合；不变量与无回退底线未削弱。
- 本轮新增开放：**BLOCKER 0 / MEDIUM 1 / LOW 2**，开放项合计 **3**（`ADR-R05-01`、`ADR-R05-02`、`ADR-R05-03`）。
- `ADR-R05-01` 使 `:1118` 仍存在与 design/代码冲突且可作两种读法的句子，故不能判决策接受级通过；`tasks.md`（N005）未被用作判据。

Verdict: CHANGES REQUIRED