# B13J D06 — `adr-0031-r10` 修订产出报告

**对象**：`.add/project/knowledge/decisions/framework.md` 的 ADR-0031 段（改前 `:1079`–`:1190`，追溯行尾 `:1190`；改后 `:1079`–`:1192`）。**只改该段**，ADR-0030（`:915`–`:1078`）、ADR-0032（`:1196`–`:1273`）及其它 ADR 零改动。

**改前实测**：272,588 B / 1,271 行 / SHA256 `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`；CRLF 1,271、bare-LF 0、bare-CR 0、无 BOM、末尾 `0D 0A`。
**改后实测**：285,639 B / 1,273 行 / SHA256 `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`；CRLF 1,273、bare-LF 0、bare-CR 0、无 BOM、末尾 `0D 0A`。
**增量**：+13,051 B / +2 行，全部为 CRLF 新增或原行替换，未做任何行尾归一化。全文指纹与段内外哈希见同目录 `hashes.txt`。

- 角色：add-arch-designer（ADR 作者）。
- 固定工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abb`。全程未执行任何 git 写操作，未删除或移动任何既有文件。
- 写集：`framework.md`（仅 ADR-0031 段）、本文件、`hashes.txt`。
- 本报告是作者自述的产出说明，**不是评审**：不含 PASS，不构成架构或安全侧接受、人的成果接收、真人签字或生产批准；不解锁任何卡，不解除生产 closed。

## 1. 改动清单（逐项行锚）

| 行锚 | 改动 | 对应 finding |
|---|---|---|
| `:1082` | 修订行升为 `adr-0031-r10` | 全量 |
| `:1084` | 状态段去时点化；写明现行修订随 `**修订**` 行自动生效、须两路接受，r10 未取得前 `identity-auth-S4.24a` 不得启动 | 架构 F003 上下文、安全 F005 |
| `:1118` | 第 4 项结论：三签校正、`anchor_ref` 五处等式、`generation` 三元、含 `P6` 的检查点集合、无条件 root 残余、见证术语消歧 | 架构 F002/F004/F005/F006/F007、安全 F001/F002/F003/F007 |
| `:1129` | r07 接受事实改为直接引用 d08 两报告；去时点化 | 架构 F009 |
| `:1147` | 不变量 6 独立性集合补 `verifier`、`external_verifier`、六席 `person_ref` | 安全 F004 |
| `:1166` | 约束 8 解除条件第 6 条同步收紧 | 安全 F004 |
| `:1174` | 「只同步」改为「收紧」并指明是解除条件第 6 条；追加 r10 范围句 | 架构 F008 |
| `:1176` | 「登记≠启动」段去时点化 | 架构 F003 上下文 |
| `:1190` | 删除行末临时 r09 句（内容并入独立 r09 行） | 架构 F001/F009、安全 F005 |
| `:1191`（新增） | 独立 r09 追溯行：跳号定稿、逐项锚点、显式接受限制、r09 改后基线 | 架构 F001/F009、安全 F005 |
| `:1192`（新增） | 独立 r10 追溯行：两份 r03 触发、授权披露、逐项 (i)–(xi)、未改段落、改前基线 | 架构 F001、安全 F005 |

段内逐行比对结果：命中上述 11 行，其余行（`1079`–`1081`、`1083`、`1085`–`1117`、`1119`–`1128`、`1130`–`1146`、`1148`–`1165`、`1167`–`1173`、`1175`、`1177`–`1189`、`1193`–`1195`）逐字节未变。

## 2. 两路 r03 评审 finding 处置

### 2.1 架构侧 `review-arch-r03`（CHANGES REQUIRED，BLOCKER 2 / HIGH 2 / MEDIUM 3 / LOW 2）

| Finding | 级别 | 状态 | 证据行锚 |
|---|---|---|---|
| F001 r09 无独立追溯行、变更范围不可审计 | BLOCKER | **ADDRESSED** | `:1190` 删临时句；`:1191` 新增独立 r09 行（触发、授权、逐项锚点、未改段落、「无提交基线／无 r07 前像」显式接受限制、r09 改后基线 272,588 B / `D8DBA64B…`）；`:1192` 新增独立 r10 行 |
| F002 第 4 项签名角色与 design 实质冲突 | BLOCKER | **ADDRESSED** | `:1118` 改为 producer 以 `receipt_anchors.public_key` 签 `JCS({"receipt": R10, "attestation": A6})`、verifier 以 `keys[verifier_key_id].public_key` 签 8 字段载荷、witness 以 `receipt_anchors.witness_key_id` 签 5 键 head；三把 key 并存；删除「`witness_key_id` 是公钥唯一来源」，并写明它不是回执签名的唯一来源。写作「对齐既有 design 的笔误校正，不是新决策」，不改「首期不提供」结论 |
| F003 `tasks.md:3197` 与现行门禁冲突 | HIGH | **DEFERRED + OPEN** | 不在本卡写集（禁改 `tasks.md`）。登记于 §5 与 §7 |
| F004 撤销/TOCTOU 漏 `P6` | HIGH | **ADDRESSED** | `:1118` 检查点集合补为 `P3`、`P6`（`Journal.Begin` 前精确字节重读与快照比对）、`R1`／`V1` prepared、`R2`／`R4`／`R6`、`V2`／`V4`／`V6`、`V7` 前、`V8` 前、`V10` 前 |
| F005 `generation` 无下位定义 | MEDIUM | **ADDRESSED**（取评审修复方向 b） | `:1118` 定义可复算三元：`approval_generation`／`trust_root_generation`／`anchor_generation`，三项逐字节相等才算同一 generation；撤销与有效期落在 design 既有字段 `organizations`／`keys`／`principals`／`appointments` 的 `revocation.state`、`valid_until`；显式写明 `receipt_anchors` 无 `revocation`／`valid_until` 字段，其撤销与轮换只经人的重钉批准并以 `anchor_generation`／`trust_root_generation` 体现 |
| F006 `anchor_ref` 五处等式字段名与条件不一致 | MEDIUM | **ADDRESSED** | `:1118` 写明 `receipt_anchors.anchor_ref` = policy `tool.receipt.trust_anchor_ref` = 回执 `trust_anchor_ref`（回执 wire 12 字段无 `anchor_ref`，`trust_anchor_ref` 是唯一载体）= 审计 `anchor_ref` = 见证 head `anchor_ref`；policy 缺键／空串／名不符即 closed，不保留条件分支 |
| F007 两种「见证」未分域 | MEDIUM | **ADDRESSED** | `:1118` 新增术语消歧段：本节见证 = restore 回执外部锚（Ed25519 5 键 head、`receipt_anchors.witness_*`）；ADR-0030 的 anchor revision 见证 `(w, r, token)`（`framework.md:1005`–`:1027`，见 `:1009`）是另一系统，key／endpoint／state／推进通道不共用；`anchor_ref` 不是 ADR-0019 `JournalHeadAnchor` |
| F008 「只同步」低估收紧 | LOW | **ADDRESSED** | `:1174` 改为「据此**收紧**约束 8 解除条件第 6 条的判据……不是单纯同步」，并追加 r10 范围句 |
| F009 遗留「待确认」语与 `tasks.md` 转引 | LOW | **ADDRESSED** | `:1191` 写明 r08 未产出、跳号至 r09 的事实已由本行定稿、不再「待评审方确认」；`:1129` 改为一律直接引用 `evidence-b13j-d08/review-arch-r01/report.md`（9,964 B / `EC9361004EF31091DC51F2B3F272264C8DC98544D8C42B2F482D18E5BE656CCB` / PASS WITH CONDITIONS）与 `review-sec-r01/report.md`（18,884 B / `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC` / PASS WITH CONDITIONS） |

### 2.2 安全侧 `review-sec-r03`（CHANGES REQUIRED，HIGH 3 / MEDIUM 3 / LOW 1 / CLOSED 5）

| Finding | 级别 | 状态 | 证据行锚 |
|---|---|---|---|
| SEC-R03-F001 policy 可缺键使五处等式不可判定 | HIGH | **ADDRESSED**（取修复方向 2） | `:1118` 把 policy 侧定为 `tool.receipt.trust_anchor_ref` 并声明其为 design 规定的必填字段；缺键、空串或字段名不符即 closed（启动前 `ErrAdmissionBinding`，已启动后 `restore_unknown`）；明确「不保留『policy 该键可缺』的条件分支」 |
| SEC-R03-F002 root 残余仅条件式登记 | HIGH | **ADDRESSED** | `:1118` 新增「**无条件残余（r10）**」：回执抗伪造止于 root／部署方，本 ADR 不声称抗 root，与 design r66 `:4070` 一致；该残余无条件成立、不取决于组织是否提供独立验证方；删除条件式弱化；仍以 `restore_unknown` 处置 |
| SEC-R03-F003 `generation` 未定义、重验点未落地 | HIGH | **ADDRESSED** | `:1118` 见架构 F004／F005 同一改点；并显式写明 `V7`／`V8` 前重验是本 ADR 在 design V 图之外的**追加关闭判据**，「只增加拒绝点、不放宽任何既有检查」；任一点失败即 closed／`restore_unknown`，**不得推进见证 `sequence`** |
| SEC-R03-F004 见证独立性集合漏 `verifier` | MEDIUM | **ADDRESSED** | `:1147` 与 `:1166` 均把 `verifier` 加入 `witness_principal_id`／`witness_key_id` 的不得相等集合，并补 `witness_key_id`≠`verifier_key_id`、`witness_principal_id`≠`verifier_principal_id`、`org_id` 解析到 `organizations.kind=external_verifier`、六席 `reviewer`／`issuer`／`executor`／`producer`／`verifier`／`witness` 的 `person_ref` 两两不等 |
| SEC-R03-F005 r09 不可独立复算 | MEDIUM | **ADDRESSED（按指令以显式接受限制登记）** | `:1191` 登记 r08 跳号、逐项锚点、r09 改后基线（272,588 B / 1,271 行 / `D8DBA64B…`）；同时把「HEAD `framework.md` 不含 ADR-0031，`r07→r09` 无提交基线」与「r07 前像字节未留存，`tasks.md:3148` 的 268,790 B／`70CFC915…` 只是任务卡转述、不可据以复算 diff」登记为**显式接受限制**，并声明不以「未改」宣告代替 diff 证据 |
| SEC-R03-F006 `tasks.md` 仍称门禁解除 | MEDIUM | **DEFERRED + OPEN** | 禁改区。登记于 §5 与 §7 |
| SEC-R03-F007 `(w,r,token)` 与外部见证缺消歧 | LOW | **ADDRESSED** | `:1118` 术语消歧段（同架构 F007） |

### 2.3 已闭合项未回归（安全侧 C01–C05）

- `C01` 签名载荷唯一性与单调性：5 键 head、8 字段验证方载荷、`sequence` 严格大于 `max(baseline_sequence, head.sequence)`、`baseline_sequence` 只增不减均保留（`:1118`）。
- `C02` V1–V11 编号：未触碰。
- `C03` 模拟见证边界：「模拟见证、未真实验证」「真实实现为外部审批系统／HSM，当前未验证」「只是测试替身」均保留（`:1118`、`:1147`）。
- `C04` 未把未登记 OPEN 写成已具备：`bundle_digest`、`nonce`、`2048`、`revocation_generation` 在全文中仍为 0 命中。
- `C05` 三级门禁：决策接受级／卡证据级／生产解除级与「登记≠启动」结构未改。

## 3. 段外不变证明

| 项 | 实测 |
|---|---|
| prefix `:1`–`:1078`（含末尾 CRLF） | 205,336 B / `7CFF3F32B15D79316D07BDE2BDE8A2C6A74C835B8DAE78158A247FC4E328F1F1`；改前改后逐字节相等 |
| suffix 改前 `:1191`–`:1271` ≡ 改后 `:1193`–`:1273` | 21,174 B / `AA9E7E183B6D0A9301D736DEE890392CCAE4619C0B942216FBA03F32134FD599`；逐字节相等 |
| 段边界 | 段起点字节偏移 205,336；改前段后 251,414；改后段后 264,465；205,336+46,078+21,174=272,588 与 205,336+59,129+21,174=285,639 双向自洽 |
| ADR-0030 | `:915`–`:1078`，整体落在 prefix 内，零改动 |
| ADR-0032 | `:1196`–`:1273`，整体落在 suffix 内，零改动 |

结论：ADR-0031 段以外没有任何字节变化；`framework.md` 未被整篇归一化（CRLF 1,273 / bare-LF 0 / bare-CR 0）。

## 4. 未闭合项与残余风险

1. **`tasks.md` 冲突未消**（架构 F003、安全 F006）：`:3197` 仍称 r07 为现行修订且门禁解除，`:3211`／`:3217` 亦过期。`adr-0031-r10` 的接受不会自动消除该冲突，卡面对账前仍不得依卡启动。
2. **design 侧字段名尚未字面同构**：design r66 `:4062`／`:4031` 仍写「批准记录（policy）中的 `anchor_ref`（若该键存在）」，而本 ADR 已按安全 F001 修复方向 2 定为必填 `tool.receipt.trust_anchor_ref`。语义等价、字面不同构；要字面一致需 design 修订（见 §7）。
3. **V7／V8 前重验是 design 外追加点**：design `:4037`／`:4062` 的 V 图只在 V2／V4／V6 有 `revalidateApproval`；本 ADR 在 V7、V8、V10 前追加拒绝点。方向是只增拒绝、不放宽，但 design 尚未登记这些点。
4. **`r07→r09` 仍不可复算**：只有任务卡转述的 268,790 B／`70CFC915…`，无前像字节，无法重算 diff；只能按 `:1191` 的锚点逐段文本核对。该项已按指令作为**显式接受限制**登记，不声称已消除。
5. **四项 unknown 未补齐**：具体工具/备份产品回执字段、存储位置、读取命令、签名或外部见证机制、验签信任锚形态、验证方主体等仍为 unknown，仍标补证主体为部署方与工具/备份供应方。
6. **r10 未获两路接受**：两份 r03 报告针对的是 `adr-0031-r09`；`adr-0031-r10` 需由**新的**架构侧与安全侧独立评审重跑各自清单。现行修订的决策接受级未达成，`identity-auth-S4.24a` 及任何实施/验证卡仍为 blocked。
7. **既有未登记 OPEN（本轮未处理，不属本次范围）**：`bundle_digest`、`nonce`、回执 2048 B 上限等仍未被显式登记为 OPEN（安全 C04 的提醒项）。
8. **索引未更新**：`decisions/README.md:39` 仍记 `adr-0031-r09（未复评…）`，与 `framework.md` 现行 `adr-0031-r10` 不一致；不在本卡写集。

## 5. 未验证声明

以下均**未运行**，不得读作已通过：

- 未运行 Go 构建、`go test`、`go vet`、`gofmt` 或任何单元/集成测试。
- 未启动服务、未连接 PostgreSQL、未做数据库替换或回退演练。
- 未接触真实 Linux 宿主、ACL/GRANT、mount、loader、fd-exec、`/proc/self/maps`、network namespace。
- 未接触真实旧备份、真实 restore 工具、真实批准记录、真实信任根、真实外部 witness／HSM／TLS 端点、审计 sink。
- 未运行 E2E；未验证 `enter`／`resume` 的任何生产路径。
- 未取得任何原生耐久回执样本；未验证模拟 witness CLI 行为。
- 未做撤销／过期／TOCTOU 的运行时实测。
- 未由真人签字、未做人的成果接收、未做安全评审、未做生产批准；本报告不含 PASS。

## 6. 越界需求（不在本卡写集，需相应角色处理）

1. `tasks.md:3197`／`:3211`／`:3217` 与 ADR 现行状态对账 → **Task-Planner**（架构 F003、安全 F006）。
2. design r66 `:4031`／`:4062` 的 policy `anchor_ref`（若该键存在）改为必填 `tool.receipt.trust_anchor_ref`，并把 V7／V8 前重验登记进唯一调用图 → **Feature-Designer**（需 design 修订，本卡不改 `design.md`）。
3. `decisions/README.md:39` 的现行修订条目更新为 r10 → **索引维护方**。
4. `lead-authorization-r12.md` §3 明示不含 `framework.md` 写入；本次写入依据为上级直接指令。若要求台账覆盖该写集，需授权方补记。

## 7. 授权与边界

- 授权：本轮 Lead 直接指令「产出 `adr-0031-r10`」。同链台账 `.spec/wip/recovery-runtime/evidence-b13j-s424a/lead-authorization-r12.md` 覆盖 design r66 与两路 r03 评审，其 §3 明示不含 `framework.md` 写入，故只作链路引用。
- 未改「首期不提供生产 restore 工具与生产批准主体」；未改候选取舍、不变量 1–5 的其余各项、约束 1–7／9、解除条件第 1／3／5／7／8／9 条。
- 未改 ADR-0030／ADR-0032 任何结论；未改 `tasks.md`、`plans/**`、`design.md` 或任何代码。
- 未补齐四项 unknown，未声称任何证据已具备。