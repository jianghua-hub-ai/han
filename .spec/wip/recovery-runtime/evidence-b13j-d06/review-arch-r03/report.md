# B13J D06 同链第三轮独立评审（架构视角）— `adr-0031-r09`

**结论：CHANGES REQUIRED**

**是否构成 `adr-0031-r09` 的接受：否。**

| BLOCKER | HIGH | MEDIUM | LOW |
|---|---|---|---|
| 2 | 2 | 3 | 2 |

判定句：`adr-0031-r09` 未达到 ADR-0031 第 1 级（决策接受级）——四项决策级要素中第 4 项「原生耐久回执」的签名角色形态与 design r65 实质冲突，且 r09 的变更范围缺少可独立复算的追溯登记；因此架构侧本次不接受该修订，`identity-auth-D06` 不得据此关闭，`identity-auth-S4.24a` 不得启动。

- 角色：add-reviewer（架构视角，非作者、非实现者）。被评对象作者：add-arch-designer。
- 本报告是评审意见：非真人签字，不构成人的成果接收、安全侧接受、实施授权或生产批准；不解锁任何卡，不关闭 `identity-auth-D06`，不解除生产 closed。
- 固定工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。全程未执行任何 Git 写操作，未删除或移动任何既有文件。
- objectType：adr（文档型决策卡 `identity-auth-D06` 的产出）。
- 多对象声明：评审对象只有 `framework.md` 的 ADR-0031 段（`:1079`–`:1190`，追溯行尾在 `:1190`）；`tasks.md`、`design.md`、ADR-0030/0032、Lead 授权记录只作为承接方、真实消费者与相邻依据被只读核对，不属于本对象；作者报告属证据不属对象。
- 写集：仅本文件。

## 1. 对象与实测基线（本人复算）

| 项 | 实测 |
|---|---|
| `framework.md` 全文 | **272,588 B / 1,271 行 / SHA256 `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`**；`git status --porcelain` = `M`（未提交） |
| 格式 | CRLF 1,271、LF 1,271、CR 1,271、TAB 0；无 BOM（首字节 `23 20 E6`）；末尾字节 `0D 0A` |
| ADR-0031 段 `:1079`–`:1190` | **46,078 B / SHA256 `025CB8653E2EBE95FF5BC2575F4EC6CFB5B724EBCB2CE8624359FBD356DA539D`**（按 CRLF 原字节复算） |
| 修订行 / 状态行 | `:1082` = `**修订**: adr-0031-r09`；`:1084` 状态段含「现行修订 `adr-0031-r09` 未经复评」 |
| 设计文件实际路径 | `.add\project\knowledge\design-workspace\identity-auth\design.md`（**不是** `knowledge/design/...`） |
| 我读到的设计版本 | `revision: identity-auth-design-ir01-r65`（`:5`）；**1,045,111 B / 5,464 行（`ReadAllLines`）/ SHA256 `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9`**，与用户给定 r65 / `87D70DA8…` 一致 |
| HEAD | `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（只读核对，未变） |
| 上轮 r01 报告 | `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r01\report.md`；19,103 B；SHA256 `ADC42326583EBAB6ABB9D4B14B8A08C12661F95A5E53276E2E4034144938BDD2`；对象 r05；PASS WITH CONDITIONS |
| 上轮 r02 报告 | `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r02\report.md`；22,211 B；SHA256 `CE09A03B540F1258FCBCB1B83D768975DE112C49FF1CB10C1D9F1E01D48D5F45`；对象 r06；PASS WITH CONDITIONS |
| 安全侧本轮 r03 | **不存在**：`evidence-b13j-d06` 下只有 `review-sec-r01`、`review-sec-r02`，无 `review-sec-r03` |
| 作者报告 | `.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r05\report.md`；42,696 B；SHA256 `56FAD3F134487E3810286D4C3A183D2C29F2854355757250474D772CA2CF34EB` |
| `tasks.md` | 898,536 B；SHA256 `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` |
| `decisions/README.md` | 21,613 B；SHA256 `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2`；`:39` 记 `adr-0031-r09（未复评…）` |

设计版本口径说明：作者报告 `:17` 记录的 design 输出为 `D2A6B54B…`（1,045,042 B），与现行 `87D70DA8…`（1,045,111 B）相差 +69 B；作者报告 `:162` 记载其后对 `:4057` 做过一处事实校正。本报告一律以现行 `87D70DA8…` 文本为准，不混用作者快照。

停止条件核对：评审开始与写入报告前两次复算，`framework.md` 均为 272,588 B / SHA256 `D8DBA64B…`，对象未在评审期间变动。

## 2. 两轮实际检查范围

- 第一轮（对象层）：ADR-0031 段全文 `:1079`–`:1190`；重点逐字核对 `:1082`、`:1084`、`:1086`、`:1090`、`:1115`–`:1118`、`:1120`–`:1129`、`:1133`–`:1147`、`:1164`–`:1166`、`:1174`、`:1176`、`:1186`–`:1190`；追溯行区与正文增量的对应关系；四项决策级要素的形态／最小字段／可核判据／否证条件。
- 第二轮（真实消费者与上游依据）：design r65 `:3933`–`:3935`、`:3987`–`:3988`、`:4037`、`:4057`–`:4063`、`:4070`；ADR-0030 `:1005`–`:1027` 与约束 1–10 `:1043`–`:1057`；`tasks.md:3186`–`:3200`；`decisions/README.md:39`；Lead `lead-authorization-r11.md` §E、`lead-authorization-r12.md`；作者报告与上两轮架构侧报告；安全侧 r01/r02 结论。

两轮均只读；未对被评对象做任何修改。

## 3. 必核事实逐项结论

### 3.1 修订序列完整性（→ F001）

- 追溯行区 `:1186`–`:1190` 只有 r03、r04、r05、r06、r07 五条独立追溯行，**没有 r08、r09 的独立追溯行**。
- r09 的信息只以一句话追加在 `:1190`（r07 行）行末：「r09 追溯：本次修订基线为 `adr-0031-r07`；`adr-0031-r08` 未产出，按用户 2026-11-02 追加任务跳号使用 `adr-0031-r09`，跳号事实待评审方确认；r09 未经复评，未改…」。
- 但正文 `:1118` 已含「r09 外部锚补充（独立见证签名）」与「r09 撤销与 TOCTOU 同步」两段实质增量；`:1147`、`:1166`、`:1174`、`:1176` 亦含 r09 增量。
- 作者报告 `:18` 明写：HEAD 版 `framework.md` 仅 37,074 B 且不含 ADR-0031，工作树 ADR-0031 是他人未提交内容，「`adr-0031-r07 → r09` 没有可独立复算的提交基线，本报告只登记现行 r09 实测值，不声称复算 r07」。
- **判定：r09 的变更范围不可独立审计，构成接受阻断。** 与 r07 行相比，r09 追加句没有触发来源、授权记录引用、逐项改动清单与「未改段落」清单；也没有记录 r07 基线的可复算指纹。任何后续读者既无法从 git 重算 r07→r09，也无法从追溯行确认 r09 究竟改了哪些段。

### 3.2 状态自洽性与 `tasks.md` 冲突（→ F003）

- 状态侧 `:1084`：「现行修订 `adr-0031-r09` 未经复评，决策接受级对 r09 未达成，`identity-auth-S4.24a` 及任何实施/验证卡不得启动」。
- `:1129`：「但本 ADR 现行修订已是 `adr-0031-r09`（r07 的接受事实按 `tasks.md` 既有记录保留为历史），r09 尚未复评，故决策接受级对现行修订尚未达成——在 r09 被架构与安全两路接受之前…不得启动（启动门禁回到 blocked）」。
- `:1176`：「现行修订为 `adr-0031-r09`，该评审接受条件随修订号推进——r09 未获架构与安全两路接受前，`identity-auth-S4.24a` 仍不得启动」。
- **判定：ADR 内部 `:1084`／`:1129`／`:1176` 自洽，且方向保守正确。**
- 冲突在消费者侧：`tasks.md:3197` 仍称「`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受……决策接受级对 r07 成立，故本卡启动门禁解除」，并把 `framework.md:1129` 的「r07 尚未复评」解释为「产出时点表述」。该表述与现行 r09 状态冲突，且其「r61 设计修订已被判 CHANGES REQUIRED」也落后于 r65（现行 r65 已被 R05 两路判 CHANGES REQUIRED（r12 §4），已不是 r61）。
- `lead-authorization-r12.md:29` 写「不改 `framework.md`（ADR-0031 r09 的 N005 三处冲突待接受后由 Task-Planner 与 Arch-Designer 按各自授权处理）」。该顺序对 ADR 接受本身不构成循环（ADR 正文已是保守正确的一侧，修的是卡面），但它意味着**接受 r09 不会自动消除 N005**；卡面必须在任何依卡启动前另行对账。此点作为 F003 的上下文，不单列 finding。

### 3.3 四项决策级要素

| # | 要素 | 形态 | 最小字段 | 可核判据 | 否证条件 | 本次判定 |
|---|---|---|---|---|---|---|
| 1 | 批准主体 | 有（具名唯一角色、独立复核） | 有（记录类型、角色/人员、依据、时间、installation/backup、digest、路径/SHA、`st_dev`/`st_ino`、owner/ACL、挂载来源、复核人、有效期、撤销） | 有（Begin 前三方 SHA 相等、`st_dev`/`st_ino` 相等、固定 fd-exec） | 有（任一不匹配即 closed） | **PASS**；unknown（未任命、无记录）已如实标注 |
| 2 | 执行主体 | 有（root 启动的独立非交互子进程、专用身份） | 有（不等于 root/批准角色、不共享凭据、无终端、不继承 secret、只读 fd、私有 stdin、最小 ENV、目标 DB 网络） | 有（主体差异实测 + 窗口外负向用例） | 有（`su`/`sudo`/shell 包装、共用凭据、越权写入） | **PASS**；unknown（切换机制、具名主体、ACL/GRANT）已如实标注 |
| 3 | 宿主隔离 | 有（POSIX 唯一候选、Windows unsupported 矩阵） | 有（固定 argv、进程组、no-follow fd、ENV allowlist、无网络） | 有（预检失败 Begin 前拒绝；Windows 退出码 2） | 有（shell/附加 argv、进程组未 join、验后重开、ENV 含 secret 或动态加载变量） | **PASS**；POSIX 实测 unknown 已如实标注 |
| 4 | 原生耐久回执 | **冲突**：签名角色写成「回执与见证 head 均由外部见证签名、`witness_key_id` 是公钥唯一来源」，与 design r65 的 producer/verifier/witness 三签三 key 模型不一致 | 部分：绑定三项与序列有，但未给回执字段名，且与 design 字段名不一致（`trust_anchor_ref`） | 部分：验签、独立读取、审计有；`anchor_ref` 五处等式与 `generation` 判据不可按字面机核 | 有（stdout/exit=0/DB 可连接/可伪造/未验签/信任锚未固定） | **FAIL**（F002、F005、F006） |

结论：第 1–3 项达到决策级；第 4 项未达到，故「四项决策级要素齐备」不成立。

### 3.4 外部锚与 design r65、ADR-0030 的一致性

先更正一处口径：用户所述「ADR-0030 见证状态机 `:1005`–`:1027` 及不变量 1–6」中，**`不变量 1–6` 是 ADR-0031 自己的不变量（`:1133`–`:1147`）**；ADR-0030 的对应物是约束 1–10（`:1043`–`:1057`）。ADR-0030 `:1005`–`:1027` 是其 r08/r09 补充小节，描述 anchor revision 见证只持久 `(w, r, token)`。本报告按此实际结构核对，不虚构 ADR-0030 存在「不变量 1–6」。

- 与 ADR-0030 见证状态机：**无字面冲突**。ADR-0030 的见证是 anchor revision 单调计数器 `(w, r, token)`；ADR-0031 r09 的见证是 restore 回执外部锚（signed head 5 键、`sequence` 严格大于 `max(baseline_sequence, head.sequence)`）。两者是不同对象。但 ADR-0031 未写一句分域说明，存在实现混淆风险（F007）。
- 与 design r65 逐项：见证 head 5 键（`anchor_ref, sequence, issued_at, head_sig_alg, head_sig`，删 `head_sig` 后 JCS）与 design `:4060`/`:4062` 一致；验证方 8 字段载荷与 design `:4059` 一致；`sequence` 严格大于 `max(baseline_sequence, head.sequence)` 与 design `:4060`/`:4062` 一致；`baseline_sequence` 单调不减与 design `:4060` 一致；`witness_endpoint` 的 TLS/DNS/无 query 约束与 design `:4060` 一致；见证独立性（`org_id=external_verifier`、不得等于 root/executor/producer/issuer/reviewer）与 design `:4060` 一致；模拟见证仅测试替身、证据标注「模拟见证、未真实验证」与 design `:4070` 一致。
- 不一致点：签名角色与公钥唯一来源（F002）；`generation`／`revocation.state`／`valid_until` 在信任根与 `receipt_anchors` 上无对应字段（F005）；`anchor_ref` 五处等式的字段名与条件（F006）；撤销重验检查点漏 `P6`（F004）。

### 3.5 授权、unknown 与模拟证据

- r09 **未改变**「首期不提供生产 restore 工具与生产批准主体」：`:1084`、`:1090`、`:1118`、`:1129`、`:1149`、`:1174` 均重申。
- **未发现**把 unknown 写成已具备：四项均显式记 unknown 并写补证主体；`:1118` 明确「模拟见证、未真实验证」「真实实现为外部审批系统／HSM，当前未验证」；`:1149` 明确六项当前均缺失、生产 restore closed。
- **未发现**把模拟见证当真实证据：`:1118`、`:1147`、`:1166` 均要求真实样本与真实外部见证，且把模拟标注为测试替身。
- 范围表述问题：`:1174` 称「只同步」，但 `:1166` 实际向约束 8 解除条件第 6 条新增了 endpoint TLS/DNS、独立性、单调性判据，属于收紧而非单纯同步（F008）。
- 遗留待确认语：「跳号事实待评审方确认」未落定；`:1129` 把 r07 接受事实指向 `tasks.md` 而非直接引用 `evidence-b13j-d08` 的两份报告（F009）。

## 4. Findings（按严重度排序）

### F001 — BLOCKER — r09 无独立追溯行，变更范围不可审计

- 行锚：`framework.md:1190`（r09 追加句）；对照 `:1186`–`:1190` 追溯行区、`:1118`、`:1147`、`:1166`、`:1174`、`:1176`。
- 原句摘录：「r09 追溯：本次修订基线为 `adr-0031-r07`；`adr-0031-r08` 未产出，按用户 2026-11-02 追加任务跳号使用 `adr-0031-r09`，跳号事实待评审方确认；r09 未经复评，未改「首期不提供生产 restore 工具与生产批准主体」、候选取舍、不变量 1–5、约束 1–7/9 与解除条件实质」。
- 判据：`decisions-guide` 与 `code-review-checklist` 要求修订可追溯、变更范围可界定；决策接受级要求评审对象可被完整限定。现有追溯区对 r03–r07 均给出触发、报告/授权与逐项改动，r09 只有一句范围宣告。
- 触发场景：后续评审、实施或回归需要确认「r09 到底改了什么、r07 基线是什么」；因 HEAD `framework.md` 不含 ADR-0031，无法用 git 重算 r07→r09。
- 影响：接受结论无法被限定到具体变更集；`:1190` 声称「未改不变量 1–5、约束 1–7/9」也无法被独立验证；后续若发现 r09 漏改或多改，无法归因。
- 修复方向：新增独立 r09 追溯行，至少含（a）触发与用户 2026-11-02 追加任务；（b）授权记录引用（`lead-authorization-r11.md` §E、`lead-authorization-r12.md`）；（c）逐项改动锚点（`:1082`、`:1118` 两段、`:1147`、`:1166`、`:1174`、`:1176`、`:1190`）；（d）未改段落清单；（e）r07 基线的可复算指纹或前像保存位置。若无法提供 r07 前像，应显式登记「无提交基线」为接受限制，而不是以一句话代替追溯。
- 验证方式：按追溯行列出的锚点逐段比对，确认改动集与声明一致；用记录的 r07 前像哈希复算 diff。

### F002 — BLOCKER — 第 4 项签名角色与 design r65 实质冲突

- 行锚：`framework.md:1118`（r09 外部锚补充）；关联 `:1086`（继承关系「不重写或削弱 design 既有 restore 软件合同……receipt 字段」）、`:1147`。
- 原句摘录：「回执与见证 head 均由外部见证以 Ed25519 签名，`witness_key_id` 是公钥唯一来源」。
- 判据：决策级要素 4 的「形态」必须唯一可实现，且不得与其声明继承的 design 软件合同冲突。
- 反证（design r65，现行 `87D70DA8…`）：
  - `:3987`–`:3988`：实际回执是 12 字段 `RestoreReceipt`，producer envelope 为 `{receipt, attestation}`，签名对象为二者 JCS；**验签公钥只来自 `receipt_anchors.public_key`**（producer 公钥）。
  - `:4059`：验证方签名载荷恰为 8 字段，验签公钥唯一来源是 `keys[verifier_key_id].public_key`。
  - `:4060`：`receipt_anchors` 14 项同时含 `public_key`（producer）、`verifier_key_id`、`witness_key_id`、`witness_public_key`；见证 head 由 `witness_key_id` 对应公钥验签。
  - `:4061`：六席 `reviewer/issuer/executor/producer/verifier/witness` 的 `person_ref` 两两不等。
  - `:4062`：`V8` 验证 **producer 签名**；`V10` 验证方签名并让外部见证原子推进 `sequence`。
- 触发场景：实施者按 ADR 让「外部见证」签回执、并用 `witness_key_id` 作为回执验签唯一公钥；或按 design 让 producer 签回执、verifier 签审计、witness 签 head。两种读法互斥，且无法同时满足设计 `V8`/`V10`。
- 影响：第 4 项决策级形态不唯一，回执与见证的 key/principal 绑定不可机核；同时使作者报告 `:90`「只把 design r62–r65 已定的外部锚同步进 ADR 口径」失实——同步错了签名人。安全侧既有报告也已把 design `:4062` 的 V8 载荷指向判为 BLOCKER（`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r05/report.md:72`、`:88`–`:117`），独立印证该区域不可接受。
- 修复方向（二选一，需架构方向决策）：
  1. 对齐 design r65：改为「回执由 producer 以 `receipt_anchors.public_key` 签名；验证方以 `verifier_key_id` 对 8 字段载荷签名；见证以 `witness_key_id` 对 5 键 head 签名；三把 key 分属不同席位，均登记在信任根」，并删除「`witness_key_id` 是公钥唯一来源」。
  2. 若确要让见证签回执，则必须显式声明本 ADR 取代 design 的对应条款，并同步触发 design 修订；不得继续声称「不重写 design 软件合同」。
- 验证方式：逐个签名对象写出「签名人 principal／key 来源／载荷 keyset」三元组，与 design `:3987`、`:4059`、`:4060` 逐项比对；不得存在两种可读法。

### F003 — HIGH — 卡面 `tasks.md:3197` 与现行 r09 门禁冲突

- 行锚：`framework.md:1084`／`:1129`／`:1176`；`tasks.md:3197`；`lead-authorization-r12.md:29`。
- 原句摘录（`tasks.md:3197`）：「`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受……决策接受级对 r07 成立，故本卡启动门禁解除」。
- 判据：真实消费者不得与上游决策的现行状态冲突；`framework.md:1176` 明写「该评审接受条件随修订号推进」。
- 触发场景：任何人按卡面字面读取「门禁解除」，即可派发/认领 `identity-auth-S4.24a`，与 ADR 现行状态相反。
- 影响：D06 关闭条件「门禁解除」与 S4.24a 依卡面启动均不可按字面成立；形成 N005 状态冲突，且 `tasks.md` 同段「r61 设计修订 CHANGES REQUIRED」也落后于 r65。
- 修复方向：由 Task-Planner 在 `tasks.md` 对账（不在本评审写集）：把 3197 改为「r09 为现行修订、未复评、门禁 blocked」，并删除或标注 r07 接受为历史；保留硬 requires 边。`r12:29` 的「接受后处理」应同时写明：接受 r09 不消除 N005，卡面未对账前不得依卡启动。
- 验证方式：`tasks.md:3197` 与 `framework.md:1084`/`:1129`/`:1176` 语义一致；`identity-auth-S4.24a` 仍为 `- [ ]` 且门禁 blocked。

### F004 — HIGH — 撤销/TOCTOU 检查点漏 `P6`，与 design 不一致

- 行锚：`framework.md:1118`（r09 撤销与 TOCTOU 同步段）。
- 原句摘录：「批准记录、信任根与见证信任锚的 `revocation.state`／`valid_until` 必须在每个检查点按 generation 与 expiry 重读重验……R2／R4／R6、V2／V4／V6 与 `prepared(ctx)` 各重验一次，读取回执、验签与推进见证 head 前再重验一次」。
- 判据：决策级「可核判据」必须覆盖 design 实际定义的检查点。
- 反证（design `:4037`）：`P6` 在 `Journal.Begin` 之前**重读批准文件精确字节并与不可变快照比对**，漂移 → `ErrAdmissionClosed`；另有 `R1 prepared(ctx)`、`V1 prepared(ctx)`。
- 触发场景：实施者按 ADR 列举的 `R2/R4/R6、V2/V4/V6、prepared(ctx)` 实现，漏掉 `P6`；批准文件在预检快照与 `Journal.Begin` 之间漂移时不被发现，恰好留下 r09 声称要关闭的 TOCTOU 窗口。
- 影响：r09 的「撤销与 TOCTOU 同步」判据不完整，决策级可核性不足。
- 修复方向：在枚举中加入 `P6`（Begin 前精确字节重读与快照比对），并明确 `R1`/`V1` 由 `prepared(ctx)` 覆盖或单独列出；保留「读取回执、验签、推进 head 前再重验」。
- 验证方式：把 design `:4037` 的 `P1`–`P6`、`R1`–`R8`、`V1`–`V11` 逐点映射到 ADR 文本，确认无未列出的重验点。

### F005 — MEDIUM — `generation` 与信任根/见证锚的 `revocation.state`／`valid_until` 无下位定义

- 行锚：`framework.md:1118`（同上段）。
- 原句摘录：同 F004 引句中的「批准记录、信任根与见证信任锚的 `revocation.state`／`valid_until` 必须在每个检查点按 generation 与 expiry 重读重验」。
- 判据：可机核判据引用的字段必须在被引 schema 中存在并已定义。
- 反证（design r65）：
  - `:3933` 信任根顶层 exact keyset 为 `version, issued_at, anchor_ref, organizations, keys, principals, appointments, reviews, receipt_anchors`——**没有顶层 `revocation.state`／`valid_until`**。
  - `:3934` 的 `organizations`／`keys`／`principals`／`appointments` 每项有 `valid_from`／`valid_until`／`revocation`，但**没有 `generation`**。
  - `:4060` `receipt_anchors` 14 项**没有** `revocation`／`valid_until`／`generation`，只有 `baseline_sequence`。
  - 全文 `generation` 只出现在 identity-auth 的 auth/operation 语境，不用于 restore 批准记录或信任根。
- 触发场景：实施者为满足 ADR 自行发明 `generation` 字段，或在信任根顶层查找不存在的 `revocation.state`，导致不同实现互不兼容，或把「无法读取」误判为 closed/成功。
- 影响：该判据不可按字面实现，撤销重验的「每个检查点」无法机核。
- 修复方向：二选一——（a）在 restore 批准记录/信任根 schema 中显式定义 `generation` 并触发 design 修订；（b）改用既有字段，明确写成「`organizations`／`keys`／`principals`／`appointments` 各项的 `revocation.state` 与 `valid_until`，加上信任根/批准文件的精确字节摘要」，并说明见证信任锚以 `receipt_anchors` 登记值的摘要与人的重钉批准为准。
- 验证方式：ADR 文本中每个字段名都能在 design keyset 中定位；否则登记为设计修订项。

### F006 — MEDIUM — `anchor_ref` 五处等式的字段名与条件不一致

- 行锚：`framework.md:1118`。
- 原句摘录：「`anchor_ref` 在 `receipt_anchors`、policy、回执、审计记录与见证 head 五处逐字节相等」。
- 判据：五处等式必须给出可定位的精确字段名与适用条件。
- 反证（design `:4062`）：等式链为 `receipt_anchors.anchor_ref`、批准记录（policy）中的 `anchor_ref`（**若该键存在**）、回执 **`trust_anchor_ref`**（回执中的 `anchor_ref`）、审计记录 `anchor_ref`、见证 head `anchor_ref`。
- 触发场景：实施者在回执对象中查找 `anchor_ref` 键而失败（实际键名是 `trust_anchor_ref`）；或把 policy 的 `anchor_ref` 当作必存在，与 design 的条件语义冲突。
- 影响：五处等式不可按字面机核，可能出现「字段缺失即拒绝」与「字段缺失合法」两种实现。
- 修复方向：按 design 原样写全字段名与条件，例如「`receipt_anchors.anchor_ref` = policy `anchor_ref`（若存在）= 回执 `trust_anchor_ref` = 审计记录 `anchor_ref` = 见证 head `anchor_ref`，任一不等 → 启动前 `ErrAdmissionBinding`／启动后 `restore_unknown`」。
- 验证方式：逐键在 design keyset 中定位；构造「policy 无该键」与「回执仅 `trust_anchor_ref`」两个用例，确认判定一致。

### F007 — MEDIUM — 两种「见证」未分域，存在实现混淆

- 行锚：`framework.md:1118`、`:1147`；对照 ADR-0030 `:1005`–`:1027`（尤其 `:1009`）。
- 原句摘录（ADR-0030 `:1009`）：「见证只持久 `(w, r, token)`」；（ADR-0031 `:1118`）「外部锚采用「独立见证签名」路线；回执与见证 head 均由外部见证以 Ed25519 签名……」。
- 判据：同一文档内同名术语指向不同系统时，必须显式分域。
- 触发场景：读者把 ADR-0030 的 anchor revision 见证（`(w,r,token)`、无签名 head）与 ADR-0031 的 restore 回执外部见证（signed head 5 键、Ed25519、`sequence`）当成同一系统，复用错误的 key、endpoint 或记录。
- 影响：跨 ADR 实施时可能把两套见证混接；虽然当前无字面冲突，但缺少分域句使「无冲突」依赖读者自行推断。
- 修复方向：在 `:1118` 或 `:1147` 加一句范围说明：「本节『见证』指 restore 回执外部锚见证，与 ADR-0030 的 anchor revision 见证 `(w, r, token)` 不是同一系统，key／endpoint／record 不共用。」
- 验证方式：全文检索「见证」，确认每处均能唯一归属到一套系统。

### F008 — LOW — 「只同步」范围句低估了实际收紧

- 行锚：`framework.md:1174`；对照 `:1166`。
- 原句摘录：「本 ADR 的 r09 只同步外部独立见证签名、模拟/真实边界、信任根与端点约束、见证独立性、撤销/TOCTOU 与验签失败处置，不改变本结论、候选取舍、不变量 1–5、约束 1–7/9 与解除条件实质」。
- 判据：范围句应准确描述改动性质。
- 反证：`:1166` 在约束 8 解除条件第 6 条新增 witness endpoint 的 DNS/TLS/无 query 约束、见证独立性、`baseline_sequence` 单调与重钉须人批准——这些是**解除条件判据的收紧/补充**，不是单纯「同步」。
- 触发场景：读者据「只同步」低估变更，认为解除条件未变，从而在解除评估时漏掉新增判据。
- 影响：范围披露不足；与 F001 的追溯缺口叠加，进一步削弱可审计性。
- 修复方向：改为「只同步……并据此收紧约束 8 解除条件第 6 条的判据；不改变本结论、候选取舍、不变量 1–5、约束 1–7/9 的其余各项」。
- 验证方式：比对 `:1166` 与 r07 前像，确认新增判据均被范围句覆盖。

### F009 — LOW — 遗留「待确认」语与 r07 接受事实的引用方式

- 行锚：`framework.md:1190`、`:1129`。
- 原句摘录：「跳号事实待评审方确认」；「r07 的接受事实按 `tasks.md` 既有记录保留为历史」。
- 判据：决策记录不应把可由本次评审落定的事实继续留为待确认；接受事实宜直接引用评审报告，而非经任务卡转引。
- 触发场景：后续读者必须翻 `tasks.md` 才能找到 r07 接受的原始报告，且不知道跳号是否已被确认。
- 影响：追溯链多一跳、可核性下降；但不影响 r09 的实质判据。
- 修复方向：记录「评审方 r03 已确认 r08 未产出、r09 为按用户 2026-11-02 追加任务的跳号使用」，并直接引用 `evidence-b13j-d08/review-arch-r01`（9,964 B / `EC936100…`）与 `review-sec-r01`（18,884 B / `DA757AD1…`）。
- 验证方式：ADR-0031 段内不再出现「待…确认」；r07 接受事实有直接报告引用。

## 5. 未运行项（不得留空冒充已核）

- 未运行 Go 构建、`go test`、`go vet`、`gofmt`；未运行任何单元/集成测试。
- 未运行真实 PostgreSQL、真实数据库替换、真实旧备份 restore、`VerifyRestored` 演练。
- 未运行 E2E；未起真实服务；未验证 `enter`/`resume` 的任何生产路径。
- 未运行 POSIX 宿主隔离实测：进程组/会话、完整 join/reap、no-follow fd、fd-exec、ENV 实际 `environ` 逐字节比对、`LD_*`/`GLIBC_*`/`*_PRELOAD` 负向用例。
- 未运行 ACL/GRANT 权限矩阵负向用例；未验证 root 与执行身份的差异实测。
- 未验证真实 HSM/外部审批系统/外部见证端点；未验证 TLS SPKI pin、DNS 解析、A/AAAA 过滤、无重定向/无 query。
- 未取得真实原生耐久回执样本；未验证模拟 witness CLI 的任何行为（本报告不把模拟当证据）。
- 未运行安全侧本轮 r03：`evidence-b13j-d06/review-sec-r03/report.md` 不存在，两路接受合取未满足。
- 未做 r07→r09 的提交基线 diff：HEAD `framework.md` 为 37,074 B 且不含 ADR-0031，无可独立复算的提交基线（F001）。
- 未做人的成果接收、未做真人签字、未做生产批准。

## 6. 门禁判定与边界

- 架构侧：本报告 verdict 为 **CHANGES REQUIRED**，故架构侧**不接受** `adr-0031-r09`。
- 安全侧：本轮 `review-sec-r03` 未见，安全侧接受未取得。
- 因此 ADR-0031 第 1 级「决策接受级」的合取条件（架构与安全两路接受 + 四项决策级要素齐备）**未满足**：两路接受缺一路，四项要素第 4 项亦不成立。
- 直接后果：`identity-auth-D06` 不得以「门禁解除」关闭；`identity-auth-S4.24a` 及任何实施/验证卡不得启动；生产 `enter`/`resume` 的 DB 替换继续 closed；「首期不提供生产 restore 工具与生产批准主体」不解除。
- 本报告非真人签字，不构成人的成果接收；不构成安全侧结论；不解锁任何卡；不解除生产 closed；不声称 Case18/S8.2/E2E 通过。

## 7. 实际执行命令与结果（审计）

1. `[IO.File]::ReadAllBytes` + `SHA256` 复算 `framework.md`：272,588 B / `D8DBA64B…`；`ReadAllLines`：1,271 行；`git rev-parse HEAD`：`6da8abb…`；`git status --porcelain -- framework.md`：` M`。
2. 复算 design：1,045,111 B / `87D70DA8…` / `ReadAllLines` 5,464 行；`rg "revision: identity-auth-design"` → `identity-auth-design-ir01-r65`。
3. 逐行读取 `framework.md:1079`–`:1190`、`:1003`–`:1058`；逐行读取 design `:3932`–`:3935`、`:3985`–`:3990`、`:4035`–`:4038`、`:4057`–`:4063`、`:4070`。
4. 复算 ADR-0031 段：46,078 B / `025CB865…`；格式：CRLF 1,271 / LF 1,271 / TAB 0 / 无 BOM。
5. 读取 `tasks.md:3188`–`:3200`、`decisions/README.md:36`–`:41`、`lead-authorization-r11.md` §E、`lead-authorization-r12.md`、作者报告与上两轮报告；`Get-ChildItem` 确认 `review-sec-r03` 不存在，且写入前 `review-arch-r03/report.md` 不存在（本报告为新建）。
6. 写入本报告后再次复算 `framework.md`，确认仍为 272,588 B / `D8DBA64B…`（对象未在评审期间变动）。
