# ADR-0031 r10 安全独立复审（r04；仅判决策接受级）

- 日期：2026-11-02
- 角色：add-reviewer｜安全轴｜objectType=adr
- 工作区：D:\AID\VSCode\Codex\Han（branch main，HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca）
- 判定级：仅 ADR-0031 `:1122` 第 1 级「决策接受级」；`:1124` 卡证据级与 `:1126` 生产解除级不在本轮判定范围
- 摘要：`adr-0031-r10` 在决策接受级通过；SEC-R03-F001–F005、F007 全部 CLOSED，F006 范围外登记；新 finding 1 项 LOW 且不阻断

## 1. 对象与冻结校验

**被审对象**：`.add\project\knowledge\decisions\framework.md`

- 派发声明：285,639 B / 1,273 行 / SHA256 `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`
- 本轮实测：**285,639 B / 1,273 行 / CR=1,273 / LF=1,273 / bare-LF=0 / BOM=no / 首 3 字节 `23 20 E6` / 末 2 字节 `0D 0A`** / SHA256 `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050` → 一致，无漂移
- 版本字面：`framework.md:1082` = `adr-0031-r10`；`:1080` 为标题（2026-11-01）
- 冻结快照：`%TEMP%\framework-adr0031-r10-freeze-B432E44A.md`，实际解析为 `C:\Users\J00010385\AppData\Local\Temp\framework-adr0031-r10-freeze-B432E44A.md`，285,639 B，同 SHA256 → 可读、同哈希；磁盘与快照一致，判读以快照为准
- 判读段：派发 `:1079`–`:1196`；实际 ADR-0031 正文为 `:1079`–`:1192`，`:1193`–`:1195` 为分隔行，`:1196` 为 ADR-0032 锚点 `<a id=adr-0032>`（`:1197` 为 ADR-0032 标题）→ 派发范围含尾部分隔，不影响判定

**基准报告（凭原 finding ID 逐条判闭合）**

| 基准 | 派发声明 | 本轮实测 |
|---|---|---|
| `review-sec-r03/report.md` | 22,195 B / `501B3102…0F9F` | 22,195 B / `501B3102D61A74CBA775961ACC8A7C7ED2EA1254BDB7091C6C1CE2F68D180F9F` |
| `review-arch-r03/report.md` | 30,450 B / `15C193D6…AF5B` | 30,450 B / `15C193D62217ADD1E71D621805F1F88E1E1A84DD0DA8BFA295143D0C3B3FAF5B` |
| `adr-0031-r10/report.md`（作者自述，只作线索） | 14,800 B / `D8A8C382…F8F7` | 14,800 B / `D8A8C382AB27BFC2888AB8728CC17F19797910DFE96361072CDA78CC4C96F8F7` |
| `adr-0031-r10/hashes.txt`（作者材料，逐值复算） | — | 3,487 B / `6BE0058A03D0A2CF796BF20D0EC0E875D736C7C9F4F9D0D38CB3BAFC9BA614B7` |

## 2. 方法

- `.codex\skills\add-code-review\SKILL.md`（按 objectType=adr 消费对应对象分支）
- `.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\guidance\decisions-guide.md`
- `.codex\skills\add-code-review\references\产品安全红线.md`
- 顺序：先核对象字节/行/哈希与快照 → 完整通读判读段 → 按原 finding ID 逐条判 → 向上游 design r70 与 `tasks.md` 做有界交叉核 → 独立复算字节账与作者 `hashes.txt`
- 版本优先级：被审对象为 ADR 正文；design 是声明不改写的软件合同（`framework.md:1086`）。design 内部字面冲突按 `design.md:4072` 的唯一优先级规则（`r70 > r69 > … > r66`）解析

## 3. 两轮实际检查范围

**第一轮（对象自身）**
- 完整性：字节、行数、CR/LF、BOM、首末字节、SHA256；快照同哈希核对
- ADR-0031 段 `:1079`–`:1196` 全文通读，含 r10 改写的 9 行（`:1082`、`:1084`、`:1118`、`:1129`、`:1147`、`:1166`、`:1174`、`:1176`、`:1190`）与新增 2 行（`:1191` r09 追溯、`:1192` r10 追溯）全文
- 不变量 1–6（`:1133`–`:1147`）、约束 1–9（`:1153`–`:1170`）、三级门禁（`:1122`/`:1124`/`:1126`）、状态段 `:1084`、历史接受 `:1129`、后续阶段前置节点 `:1174`/`:1176`

**第二轮（上游与证据）**
- design r70（1,103,152 B / `3A8A9E63…5254`）逐点交叉核：`:3988` producer envelope、`:4037` 唯一调用图与失败分类、`:4059` verifier 8 字段载荷、`:4033`/`:4060` `receipt_anchors` 14 项 keyset、`:4077` 扩为 15 项、`:4062`/`:4074` V7–V11 载荷与五处等式、`:4070` 残余、`:4073`/`:4153` `tool.receipt` 13 键集、`:4063`/`:4039` ENV 闭锁、`:4105` golden tool vector、`:4031`/`:4062` 的 `anchor_ref` 条件子句
- 字节账：`%TEMP%\framework.before.r10.md`（272,588 B / `D8DBA64B…7B4`）与新文件的前缀/后缀/ADR 段三段 raw-bytes 哈希与逐行差异
- 作者 `hashes.txt [1]`–`[5]` 逐值复算
- `tasks.md` 现状、`:3186`/`:3190`/`:3191`/`:3220` 冲突面与全文 `adr-0031-r07`/`adr-0031-r10` 命中扫描

## 4. 观察项（对象外，不中止、不代改）

- `tasks.md` 现状 **904,142 B / 3,312 行 / `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C`**，与派发声明 898,536 B / `4BD724A9…` 不一致 → 漂移，登记为观察项
- `design.md` 现状为 `identity-auth-design-ir01-r70`（1,103,152 B / `3A8A9E63…5254`）；作者 `hashes.txt [6]` 记录的是 r66（1,079,559 B / `9B193BDE…2298`）。ADR `:1118` 多处按 `identity-auth-design-ir01-r66` 引用；已按 r70 逐点复核，无实质分歧（见 §5、§7）
- HEAD `framework.md` 不含 ADR-0031（工作树三文件均为 `M`），故 r07→r09 无可提交前像 → 见 SEC-R03-F005
- `design.md:4033` 写 `receipt_anchors` 14 项、`:4077`（r66.5）扩为 15 项并新增 `witness_host`；按 `:4072` 优先级规则现行应为 15 项。ADR `:1118` 写「design r66 的 15 项 keyset」→ 与现行一致，无冲突

## 5. 逐条判定（引用原 ID）

### SEC-R03-F001（HIGH）→ CLOSED

- `:1118` 已把键名定死并删条件分支：「**`anchor_ref` 五处等式（r10 定名）**：`receipt_anchors.anchor_ref` = 批准记录 policy 的 `tool.receipt.trust_anchor_ref` = 回执 `trust_anchor_ref`（回执 wire 的 12 字段中没有 `anchor_ref` 键，`trust_anchor_ref` 是唯一载体）= 审计记录 `anchor_ref` = 见证 head `anchor_ref`，五处逐字节相等。policy 的 `trust_anchor_ref` 是 design 规定的 policy 必填字段，缺键、空串或字段名不符即 closed——启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`；不保留「policy 该键可缺」的条件分支。」
- 五处齐备（`receipt_anchors.anchor_ref`、policy `tool.receipt.trust_anchor_ref`、回执 `trust_anchor_ref`、审计 `anchor_ref`、见证 head `anchor_ref`），并要求逐字节相等
- 对象内 `若该键存在` 命中数 = **0**（framework.md 全文），无「该键可缺」分支
- 与 design 数据面一致：`design.md:4153` 的 `tool.receipt` 13 键集含 `trust_anchor_ref`、不含 `anchor_ref`；`:4105` golden tool vector 含 `"trust_anchor_ref":"anchor-01"`；`:3988`/`:4043` 的 12 字段回执 wire 无 `anchor_ref`、有 `trust_anchor_ref` → 「唯一载体」陈述准确
- 残余：design `:4031`、`:4062` 仍写「批准记录（policy）中的 `anchor_ref`（若该键存在）」→ 记 ADR-S04-01（LOW，不阻断）

### SEC-R03-F002（HIGH）→ CLOSED

- `:1118`「**无条件残余（r10）**：回执抗伪造止于 root／部署方——root 可整体替换信任根、批准记录与见证，本 ADR 不声称抗 root（与 design `identity-auth-design-ir01-r66` 的「不声称抗 root」一致）；该残余无条件成立，不取决于组织侧是否提供独立验证方与固定信任锚；生产 restore 因此继续 closed，不得以「有签名」或「有外部见证」暗示已抗 root 伪造。」
- 无条件性成立：文字明确「不取决于组织侧是否提供独立验证方与固定信任锚」，未残留条件式弱化
- 前置限定同向：`:1118` 前段「该约束只作用于非 root 主体——root 能整体替换信任根与见证（见下述无条件残余），故不得据此声称抗 root」
- 上游一致：`design.md:4070`（2）「root 可整体替换信任根与见证：本设计不声称抗 root」
- 无「有签名/有见证即抗 root」的暗示：`:1147`/`:1166` 仅把见证定位为外部锚，并保留 root 残余与验签失败即 closed

### SEC-R03-F003（HIGH）→ CLOSED

- 三元定义可复算（`:1118`）：(i) `approval_generation` = 批准文件精确字节的 SHA-256（预检不可变快照记录，`revalidateApproval` 重读比对）；(ii) `trust_root_generation` = 信任根文件精确字节的 SHA-256（必须等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`）；(iii) `anchor_generation` = `receipt_anchors` 中该 `anchor_ref` 条目 JCS 规范字节的 `sha256:` 摘要（必须等于 policy 与回执的 `trust_anchor_sha256`）；三项同时逐字节相等才算同一 generation，且明写「不得以『文件可读』『inode 未变』或『长度未变』替代摘要比较」
- 检查点覆盖：`P3`；`P6`（`Journal.Begin` 前）；`R1`/`V1` 的 `prepared(ctx)`；`R2`/`R4`/`R6`；`V2`/`V4`/`V6`；`V7` 独立读取回执前、`V8` 验 producer 签名前、`V10` 追加审计／验证方签名／推进见证 head 前 → 要求全项齐备
- 每点动作可判：重读批准文件、信任根与 `receipt_anchors`，重算三元，检查 `valid_from <= now < valid_until`、`revocation.state`＝`active`（`revoked_at`/`revocation_ref` 为空串）、`reviews[].decision`、`now - issued_at <= 366 天`；`now` 每点取一次并内传，`mono` 高水位回退即 closed
- 失败分类：`P3/P6/R1/R2/R4/V1/V2/V4` → `ErrAdmissionClosed` 且不进入 `Journal.Begin`；`R6/V6` 与 `V7/V8/V10` 前 → `errors.Join(recovery.ErrRecoveryWriteUnknown, err)` → `restore_unknown`；「任何情况下**不得推进见证 `sequence`**」→ 与 `design.md:4037`（`P6` → `ErrAdmissionClosed`；「`R6`／`V6` 之后的漂移必须 `restore_unknown` 或 closed」）、`:4062`（V10 前 `revalidateApproval` 失败 → `ErrRecoveryWriteUnknown` → `restore_unknown`，不得推进 head）一致
- 只增拒绝点：V7/V8 前重验被声明为「本 ADR 在 design r66 既有 V 图（重验节点为 V2／V4／V6）之外的追加关闭判据，只增加拒绝点、不放宽任何既有检查」
- 无自指：`receipt_anchors` 条目无 `revocation`/`valid_until`，ADR 明写「不得在 `receipt_anchors` 上查找不存在的字段」，撤销与轮换经人的明确重钉批准并以 `anchor_generation`/`trust_root_generation` 变更体现 → 与 `design.md:4077` 的 15 项 keyset 一致
- 引用版本注记：ADR 称「按 design r66 的检查点集合」，而 `P6` 的文本落在 `design.md` r67.5（`:4062`）——内容与现行 r70 要求一致，仅出处标注不精确，按观察处理（不记 finding，不影响可判性）

### SEC-R03-F004（MEDIUM）→ CLOSED

- `:1147`：「`witness_principal_id`／`witness_key_id` 及其 `org_id` 不得等于 root／executor／producer／issuer／reviewer／**verifier** 任一方，且 `witness_key_id` 与 `verifier_key_id`、`witness_principal_id` 与 `verifier_principal_id` 必须分别不同」；同段「`witness_principal_id` 的 `org_id` 必须解析到 `organizations.kind=external_verifier`」「六席 `reviewer`／`issuer`／`executor`／`producer`／`verifier`／`witness` 的 `person_ref` 必须两两不等」
- `:1166` 同步收紧：「`witness_principal_id`／`witness_key_id` 与 root／executor／producer／issuer／reviewer／verifier 两两独立，见证方 `organizations.kind=external_verifier` 且与部署方／root 分属不同管理域，六席 `person_ref` 两两不等」
- `witness_key_id` 定位正确：`:1118`「`witness_key_id` 只是见证公钥的标识，**不是**回执签名的公钥唯一来源」；producer 验签公钥唯一来源 `receipt_anchors.public_key`，verifier 为 `keys[verifier_key_id].public_key`（`principal_id` 必须等于 `verifier_principal_id`），witness 为 `receipt_anchors.witness_key_id` 对应公钥
- 与 design r70 一致：`:4059`（8 字段 + principal 相等）、`:4074`（head 5 键、签名主体唯一 `witness_key_id`）、`:4077`（见证独立性）

### SEC-R03-F005（MEDIUM）→ CLOSED（附显式登记的限制）

- `:1191` 显式登记 r07→r09 不可独立复算（HEAD 无 ADR-0031、无 r07 前像、`tasks.md` 卡面只是转述），并把跳号（r08 未产出）定稿
- `:1192` 给出改前基线 `D8DBA64B…7B4`／272,588 B／1,271 行与 (i)–(xi) 变更清单，改后值外置到同批 `hashes.txt`（不自指）
- 本轮独立复算 `hashes.txt [1]`–`[5]` 全部命中（见 §6）→ **r10 之后可独立复核成立**
- 限制（不得越过）：r07→r09 区间属已披露、不可消除的缺口（无前像），不得据此宣称该区间已被审计

### SEC-R03-F006（MEDIUM）→ 范围外观察项（不判 CHANGES REQUIRED，不代改）

- 事实：`tasks.md:3220` 仍写「`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受……决策接受级对 r07 成立，故本卡启动门禁解除」；`tasks.md` 全文 **无** `adr-0031-r10` 命中；`:3186` 仍未勾选；`:3190` 范围仍为「生产 restore source/target/tool 与批准主体接线」；`:3191` 已「2026-11-02 对账至 design r70」
- 冲突：与 `framework.md:1084`（现行修订须两路接受才解除；r10 两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动）、`:1129`、`:1174`、`:1176` 直接冲突
- 后果：只读卡面者可误判门禁已解除并按 r07 口径派发/认领；ADR 侧已无此口径，两处结论相反
- 处置：仅登记，不在本报告计数主体内；交 Lead/Task-Planner 对账

### SEC-R03-F007（LOW）→ CLOSED

- `:1118`「**见证术语消歧（r10）**：本节「见证」只指 restore 回执的**外部锚见证**（Ed25519 签 5 键 head、持久单调 `sequence`、`receipt_anchors.witness_*`）。它与 ADR-0030 的 **anchor revision 见证** `(w, r, token)`（`framework.md:1005`–`:1027`，见 `:1009`）不是同一系统：对象、状态、endpoint 与 key 相互独立，不得共用 key、endpoint、记录或推进通道，一套推进不得替代另一套。本节的 `anchor_ref` 指 receipt anchor，**不是** ADR-0019 的 `JournalHeadAnchor`」
- 引用锚正确：`:1009` 确为 ADR-0030 的 `(w, r, token)` 见证状态机
- 分域四项俱全（对象、状态、endpoint、key），并禁止共用记录与推进通道

## 6. 字节账独立复算（r09→r10）

改前基线：`%TEMP%\framework.before.r10.md` = **272,588 B / 1,271 行 / `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`**（与 `:1192` 自述一致）。

| 区段 | 字节 | SHA256 |
|---|---|---|
| prefix 行 1–1078 含末尾 CRLF（新旧相同，raw bytes） | 205,336 | `7CFF3F32B15D79316D07BDE2BDE8A2C6A74C835B8DAE78158A247FC4E328F1F1` |
| ADR 段 BEFORE 行 1079–1190 含末尾 CRLF | 46,078 | `025CB8653E2EBE95FF5BC2575F4EC6CFB5B724EBCB2CE8624359FBD356DA539D` |
| ADR 段 AFTER 行 1079–1192 含末尾 CRLF | 59,129 | `DD1A61F8D76F06F9118CBAF8AF01FB94AF610E5D7EF2FDA19DDB45D96CCC0D48` |
| suffix 末 21,174 B（新 264,465 起／旧 251,414 起，raw bytes 相同） | 21,174 | `AA9E7E183B6D0A9301D736DEE890392CCAE4619C0B942216FBA03F32134FD599` |

- 算术自洽：旧 205,336 + 46,078 + 21,174 = 272,588；新 205,336 + 59,129 + 21,174 = 285,639 → **+13,051 B / +2 行**，与 `hashes.txt [1]` 一致
- 逐行比对（1079–1192）：改写 9 行 `:1082/:1084/:1118/:1129/:1147/:1166/:1174/:1176/:1190`，新增 2 行 `:1191`/`:1192`；旧 1191–1271 与新 1193–1273 逐行比对 **0 mismatch**；合计「命中 11 行」与 `hashes.txt [5]` 一致
- 结论：段外零改动、增量为纯新增与改写，作者字节账真实

## 7. r10 新引入安全面核查

- **producer**：`:1118` 以 `receipt_anchors.public_key` 签 `JCS({"receipt": R10, "attestation": A6})`；`R10` = 12 字段回执删 `signature`/`audit_record_ref`（10 字段），`A6` = 7 字段 attestation 删 `signature`（6 字段）→ 与 `design.md:3988`、`:4074` 一致
- **verifier**：`keys[verifier_key_id].public_key` 签 8 字段（`anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id`），key 的 `principal_id` 必须等于 `verifier_principal_id` → 与 `design.md:4059` 一致
- **witness**：`receipt_anchors.witness_key_id` 对应公钥签 5 键 head 去 `head_sig` → 与 `design.md:4074` 一致
- **三签不代签**：任一席位不得代签其他席位载荷；三把 key 登记并存于 `receipt_anchors` 与 `keys` 且分属三席
- **sequence**：严格大于 `max(baseline_sequence, head.sequence)`；`baseline_sequence` 只增不减；重钉须人的明确批准并留证，降低尝试即 closed；验签失败不得推进 `sequence` → 与 `design.md:4060`/`:4074`、`:4033` 一致
- **独立读取与验签前置**：`:1118`「root 必须经与写入路径不同的独立读取路径读取」「读取回执时必须先验签并留审计记录」「验签信任锚固定登记在批准记录内、其轮换须经人的明确批准」；`:1147`「不得由被验证回执自身或 root 自选」→ 与 `design.md:4153` 一致
- **unknown 未被写成已具备**：`:1118` 第 4 项 unknown 块、`:1120`「任命、具名主体与切换机制为 unknown」、`:1149`「上述六项在当前均为缺失状态」、`:1126` 生产解除级不受本修订影响、`:1174`「不产生任何人的批准」、`:1192` 末句「四项 unknown 仍为 unknown」
- **模拟见证**：`:1118`/`:1147`/`:1166` 均要求标注「模拟见证、未真实验证」，并禁止本地文件、本机进程或服务自身冒充外部见证

## 8. 无回退确认（逐项在文）

- 不变量 1–6：`:1133`–`:1147` 齐备（批准来源、可执行文件批准含 `st_dev`/`st_ino` 与 fd-exec、专用执行身份、root/执行身份权限边界、POSIX/Windows 矩阵、原生耐久回执）
- 约束 1–9：`:1153`–`:1170` 齐备（不改变既有决定、禁止执行通道扩张、禁止伪证明、批准与执行分离、`restore_unknown` 保留、不声称验证、不登记实现任务、解除条件九条、不适用场景）
- 「首期不提供生产 restore 工具与生产批准主体」：`:1084`/`:1090`/`:1100`/`:1105`/`:1129`/`:1149`/`:1174` 一致，无收窄
- `restore-v1`/`verify-v1`：`:1086`/`:1090` 保留；未新增第二 authority、fallback 或隐式默认
- no-follow 稳定 fd 与 fd-exec：`:1115`/`:1134`/`:1141`/`:1143`/`:1162`
- 最小 ENV allowlist：`:1144`/`:1164`（`env_clear` 后封闭字面集、实际 environ 与白名单逐字节相等、`LD_PRELOAD`/`LD_LIBRARY_PATH`/`LD_AUDIT`/`GLIBC_TUNABLES`、`*_PRELOAD`/`*_OPTIONS`、`PATH` 越界即 closed）。ADR 未逐一枚举 `GLIBC_*`/`MALLOC_*` 家族名，但 `design.md:4063`/`:4039` 明确要求并反向引用 ADR `:1117`/`:1144`/`:1164`，且 allowlist 语义本身排除未登记键 → 不构成回退（仅 ADR 字面枚举面窄于 design，ADR 已声明不改写 design 软件合同 `:1086`）
- Begin 前 closed：`:1107`/`:1115`/`:1122`/`:1126`/`:1145`/`:1174`
- `restore_unknown`：`:1100`/`:1105`/`:1118`/`:1149`/`:1157`
- Windows 生产 unsupported（退出码 2）：`:1117`/`:1139`–`:1145`/`:1170`
- 不得以 stdout／exit=0／DB 可连接／本次 run 行自证：`:1107`/`:1118`/`:1147`/`:1155`
- 私有 stdin、无 shell、固定 argv：`:1116`/`:1117`/`:1135`/`:1141`/`:1154`
- 凭据泄露面：本轮 r10 未新增任何 secret 取得、传递、记录或 env 路径（改动集中在声明、等式、检查点与追溯行），`design.md:3866`/`:3942` 的 secret 清理与快照口径未被 ADR 改写

## 9. 新 finding

### ADR-S04-01（LOW；不阻断决策接受）

- 位置：`framework.md:1118`（新口径）对 `design.md:4031`、`design.md:4062`（旧口径）
- 事实：ADR 已把 policy 信任锚键定名为 `tool.receipt.trust_anchor_ref` 并删除「若该键存在」；design 两处仍写「批准记录（policy）中的 `anchor_ref`（若该键存在）」
- 违反依据：ADR `:1086` 声明不重写 design 软件合同，但被引用上游仍保留条件子句 → 可追溯性缺口
- 触发场景：实现者只读 `design.md:4031`/`:4062` 而不读 `framework.md:1118`，实现出「policy 无该键时跳过等式」的分支
- 影响：安全方向由 ADR 收紧（缺键即 closed），design 字面更宽；不影响 ADR 自身判据的可判性与严格性，故 LOW
- 修复方向（不代改）：由 design 修订删除两处条件子句，并把键名统一为 `tool.receipt.trust_anchor_ref`
- 为何不阻断：被验收对象是 ADR，其判据唯一且无「可缺」分支；`design.md:4153` 的 policy 13 键集已含 `trust_anchor_ref`、`:4105` golden tool vector 含 `"trust_anchor_ref":"anchor-01"`，说明 design 现行数据面与 ADR 定名一致，分歧仅在两处残留说明文字

## 10. 开放项 / 待决定 / 建议（分开）

**开放项（计数主体）**
- 无 BLOCKER、无 MEDIUM；LOW 1 项：ADR-S04-01（见 §9，不阻断）

**开放项（计数主体之外，登记）**
- SEC-R03-F006：`tasks.md:3220` 与 `:3186` 冲突及 `tasks.md` 漂移（§4、§5），范围外，交 Lead/Task-Planner
- SEC-F007 等价项：Linux 侧与真实验证全部 NOT RUN（§11），不得因文档存在判 PASS
- r07→r09 无前像：已披露、不可消除（§5 F005）

**待决定**
- design 侧是否排期删除 `:4031`/`:4062` 的「若该键存在」（ADR-S04-01 的收口方式与时机）
- `tasks.md` 卡面对账与 `:3186` 勾选（Lead/Task-Planner；本轮不改）
- `adr-0031-r10` 的两路接受须架构侧 r04 与本报告同时成立；任一路未接受时 S4.24a 仍不得启动（`:1084`/`:1174`）

**建议（非阻断）**
- ADR 引用 design 时补一句版本适用声明（`r66` 引用在 `r70` 仍成立，依据 `design.md:4072` 优先级规则），减少后续版本引用歧义
- 后续修订可把 `GLIBC_*`/`MALLOC_*` 家族名直接写入 `:1144`/`:1164`，与 `design.md:4063` 字面对齐（当前不构成回退）

## 11. 未运行项（NOT RUN，不得判 PASS）

- Linux 宿主隔离实测：POSIX ACL/GRANT 负向用例、mount/loader、fd-exec 实跑、`/proc/self/maps`、网络隔离、进程树 join/reap
- 真实外部 witness（审批系统/HSM）与任何工具原生耐久回执样本
- 专用执行身份与身份切换机制实测；批准角色任命与两条批准记录原件
- build / test / E2E；`cmd/**`、`internal/**` 任何代码与测试
- golden vector SHA256 复算（`design.md:4099` 行长 1,115、`:4105` 行长 2,357，两行均为纯 ASCII，字符数即字节数，与声明吻合；本轮对象为 ADR，未复算其哈希）

## 12. 实际执行命令（PowerShell 5.1，节选）

```
$fw='.add\project\knowledge\decisions\framework.md'
(Get-Item -LiteralPath $fw).Length; (Get-FileHash -Algorithm SHA256 -LiteralPath $fw).Hash
$sn=Join-Path $env:TEMP 'framework-adr0031-r10-freeze-B432E44A.md'; (Get-FileHash -Algorithm SHA256 -LiteralPath $sn).Hash
$L=[IO.File]::ReadAllLines($fw,[Text.Encoding]::UTF8); $L.Count; $L[1078..1195]
$b=[IO.File]::ReadAllBytes($fw); 逐字节计 CR/LF、取首末字节
$b0=Join-Path $env:TEMP 'framework.before.r10.md'; 分段 SHA256（prefix 1-1078 / ADR 段 / 末 21174 B）
逐行差异循环 for($k=0;...) { $N[$ni] -ne $O[$oi] }
[regex]::Matches 统计「若该键存在」（framework=0；design=:4031、:4062）
-like 扫描 tasks.md 的 adr-0031-r07 / adr-0031-r10 命中行并读取 :3186/:3190/:3191/:3220
```

## 13. 只读声明

本轮只读判读下列文件；未修改、移动、删除任何被审或其他制品；未执行任何 Git 写操作；唯一写入为本目录 `report.md` 与 `hashes.txt`。

- `.add\project\knowledge\decisions\framework.md`（被审对象，未改）
- `%TEMP%\framework-adr0031-r10-freeze-B432E44A.md`（冻结快照，未改）
- `%TEMP%\framework.before.r10.md`（r10 改前基线，未改）
- `.add\project\knowledge\design-workspace\identity-auth\design.md`（上游，未改）
- `.add\project\knowledge\tasks\tasks.md`（范围外观察，未改、未勾选）
- `.spec\wip\recovery-runtime\evidence-b13j-d06\review-sec-r03\report.md`（基准）
- `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r03\report.md`（基准，仅交叉核对）
- `.spec\wip\recovery-runtime\evidence-b13j-d06\adr-0031-r10\report.md`、`hashes.txt`（作者材料，只作线索）
- `.codex\skills\add-code-review\SKILL.md`、`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\guidance\decisions-guide.md`、`.codex\skills\add-code-review\references\产品安全红线.md`

## 14. 结论与计数

- 决策接受级：SEC-R03-F001–F005、F007 全部 CLOSED；SEC-R03-F006 范围外登记；新 finding 1 项 LOW 且不阻断
- 计数：**BLOCKER 0 / MEDIUM 0 / LOW 1**
- 本结论不构成：生产已解锁、`enter`/`resume` 已解锁、约束 8 已满足、S4.24a 可关闭或可派发实施、抗 root 成立；亦不替代架构侧接受与人的决策接纳
- 未验证风险：§11 全部 NOT RUN 项；`tasks.md` 卡面冲突的传播风险

Verdict: PASS