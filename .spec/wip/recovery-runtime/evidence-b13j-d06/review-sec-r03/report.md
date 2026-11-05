# B13J D06 第三轮安全侧独立评审 — `adr-0031-r09`

**结论：CHANGES REQUIRED**

> 评审者：add-reviewer（安全视角，非作者、非实现者）。被评对象作者：add-arch-designer（据 ADR-0031 追溯行与 `tasks.md` 记录）。
> 判断方式：读文判断 + 文本/哈希实测。未运行 Go 测试/构建、未连接 PostgreSQL、未起服务、未接触任何真实主机、备份系统、凭据、ACL 或外部见证。
> 本报告是评审意见：不构成安全认证、人的成果接纳、生产批准或实施授权；不关闭 `identity-auth-D06`，不解锁任何卡，不解除生产 closed。

## 0. 结论与判定

| 级别 | 数量 |
|---|---|
| HIGH | 3 |
| MEDIUM | 3 |
| LOW | 1 |
| CLOSED | 5 |

**明确判定：`adr-0031-r09` 是否构成决策接受级（第 1 级门禁）的「接受」——否。**

一句话理由：r09 新增的外部锚与撤销/TOCTOU 同步，在四项决策级要素上未齐备——「最小字段／可核判据」在 policy `anchor_ref`（F001）与 `generation`（F003）两处不可判定，「形态」的见证独立性集合漏掉 verifier（F004），「否证条件」对 root 整体替换只作条件式残余而非无条件登记（F002）；同时 r09 增量不可独立复算（F005）。安全侧因而不接受；即使架构侧接受，合取门禁仍不成立，`identity-auth-S4.24a` 不得启动。

## 1. 对象与实测基线（本人复算）

| 项 | 实测 |
|---|---|
| 被审对象 | `.add/project/knowledge/decisions/framework.md` 的 ADR-0031 段 `:1079`–`:1191`，现行 `adr-0031-r09`；「r05 四项结论」段 `:1111`–`:1129`；状态段 `:1082`/`:1084`；不变量 6 `:1147`；约束 8 `:1160`–`:1169`；追溯行 `:1188`–`:1190` |
| 对象全文 | **272,588 B / 1,271 行 / SHA256 `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`**，工作树 `M` |
| `identity-auth/design.md`（实际路径 `.add/project/knowledge/design-workspace/identity-auth/design.md`） | **1,045,111 B / 5,464 行 / `revision: identity-auth-design-ir01-r65` / SHA256 `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9`**（读到的是 r65，不是并行中的 r66；本报告结论只绑定 r65） |
| `tasks.md`（仅对照，未修改） | 898,536 B / 3,289 行 / SHA256 `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` |
| HEAD / 分支 | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` / `main` |
| 上轮 `review-sec-r01` | `.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r01/report.md` 19,208 B / 165 行 / `A4FFA36DF701CC9705563B3A1A415FE53C36A5857B6C7603E354BB2D90DDAABA` / CHANGES REQUIRED（对象 r05） |
| 上轮 `review-sec-r02` | `.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r02/report.md` 18,286 B / 143 行 / `1C373557F597422127527E8544307E3FB4862E81F192A6F6DA54D6EEE72914BF` / PASS WITH CONDITIONS（对象 r06） |
| 本报告 | 新建，对象 r09（同链 r03）；唯一写集 |

基线停止条件：评审开始与写入前各实测一次，B/行/SHA 未变；无停止触发。

## 2. 七个指定安全要点的裁决索引

| # | 安全要点 | 裁决 | 归入 |
|---|---|---|---|
| 1 | 外部锚可信性（公钥唯一来源、org 绑定、轮换人批、撤销重验、root/回执自选） | **不通过** | F002、F003、F004；F001 涉及锚引用 |
| 2 | 签名载荷唯一性（5 键 head / 8 字段验证方 / 与 ADR-0030 不冲突 / sequence 单调） | 通过（术语见 F007） | C01 |
| 3 | `anchor_ref` 五处逐字节等式与缺失 fail-closed | **不通过** | F001 |
| 4 | 模拟见证边界 | 通过 | C03 |
| 5 | 三级门禁消除自指死锁 | 通过 | C05 |
| 6 | 修订可审计性（r08/r09 追溯行缺失） | **不通过** | F005 |
| 7 | 未登记 OPEN 被写成已具备 | 通过（r09 未 overclaim） | C04 |

## 3. 逐条 finding

### `SEC-R03-F001` — HIGH — `anchor_ref` 五处等式在 policy 可缺键时不可判定

- 行锚：`framework.md:1118`；`design.md:4031`、`:4062`、`:4093`、`:4101`
- 原句：
  - `framework.md:1118`：「`anchor_ref` 在 `receipt_anchors`、policy、回执、审计记录与见证 head 五处逐字节相等」
  - `design.md:4031` / `:4062`：「批准记录（policy）中的 `anchor_ref`（若该键存在）」
- 判据：
  - policy 侧被明写为「若该键存在」，五处等式可合法退化为四处；缺键没有独立 fail-closed 分支。
  - `design.md:4093` 的 tool golden vector，`receipt` 对象只有 `"trust_anchor_ref":"anchor-01"`，**没有 `anchor_ref` 键**；批准记录顶层也没有 `anchor_ref`。设计正文用「回执 `trust_anchor_ref`（回执中的 `anchor_ref`）」把两者等同，但没有任何唯一映射规则（何时、由谁、按什么规则把 `trust_anchor_ref` 映射为 `anchor_ref`）。
  - `design.md:4101` 的 fail-closed 只覆盖**信任根文件**的 `anchor_ref` 缺失（`ErrAdmissionIncomplete`），不覆盖 policy `anchor_ref` 缺失。
  - 结果：作为决策级「可核判据」，这五处等式当前无法机核，也无法判定缺失时应 closed。
- 影响：实现可只比较容易拿到的四处；policy 侧锚被替换或缺失时无判据拦截，外部锚引用链出现断点。
- 修复方向（择一，二选一并同步 golden vector）：
  1. 把 policy exact keyset 强制包含 `anchor_ref`，缺失 → `ErrAdmissionBinding`／closed；
  2. 或把五处等式改为 policy 的 `trust_anchor_ref`，并给出它与其余四处逐字节相等的唯一映射，删除「若该键存在」。
- 证据：上述行原文；`design.md:4093` 的 receipt 键集；`design.md:4101` 只列信任根 `anchor_ref` 缺失。
- 复算：
  ```powershell
  cd 'D:\AID\VSCode\Codex\Han'
  $p='.add/project/knowledge/decisions/framework.md'
  $d='.add/project/knowledge/design-workspace/identity-auth/design.md'
  rg -n '若该键存在|trust_anchor_ref|五处' $p $d
  ```

### `SEC-R03-F002` — HIGH — 外部锚仍不能抵抗 root 整体替换，ADR 与 design 的残余表述强度不一致

- 行锚：`framework.md:1118`；`design.md:4070`、`:4103`
- 原句：
  - `framework.md:1118`：「验签信任锚（公钥或见证端点）固定登记在批准记录内、其轮换须经人的明确批准，且信任锚不得由被验证回执自身或 root 自选」；「残余风险：若组织侧无法提供独立验证方与固定信任锚，本项须显式登记残余风险『回执抗伪造止于 root』并保持 closed」
  - `design.md:4070`：「root 可整体替换信任根与见证：本设计不声称抗 root，审计与见证链的权威止于外部见证 head」
  - `design.md:4103`：「批准抗伪造止于 root／部署方」
- 判据：
  - 批准记录本身位于信任根体系内；design 明说 root 可整体替换信任根与见证。既然 root 能重写批准记录与信任锚，`framework.md:1118` 的「不得由 root 自选」在机制上不成立，只能作为对非 root 主体的约束。
  - ADR 的残余登记是**条件式**（「若组织侧无法提供……」），而 design 是**无条件**残余登记。决策级对象应无条件登记当前残余，否则读者可把外部见证签名误读为「已抗 root」。
  - 这不是要求 r09 解决 root 威胁，而是要求其「否证条件／残余」与 design 的事实一致且无条件。
- 影响：把「外部见证」读成「抗 root 伪造」的口子仍在；生产 closed 的依据在 ADR 层不完整。
- 修复方向：在 `:1118`（及不变量 6）无条件登记「回执抗伪造止于 root／部署方，生产保持 closed」；若要声称抗 root，须把信任根/批准记录放到 root 写域之外（外部方签名或不可变存储）并写成可核判据。
- 证据：上述三段原文；`design.md:4070` 明确「不声称抗 root」。
- 复算：
  ```powershell
  rg -n '不声称抗 root|批准抗伪造止于 root|抗伪造止于 root' $p $d
  ```

### `SEC-R03-F003` — HIGH — `generation` 未定义，读取/验签/推进 head 前的重验点未落到 design/code

- 行锚：`framework.md:1118`；`design.md:4037`、`:4098`、`:4125`；`cmd/han-server/recovery_restore.go:247`–`:255`
- 原句：
  - `framework.md:1118`：「批准记录、信任根与见证信任锚的 `revocation.state`／`valid_until` 必须在每个检查点按 generation 与 expiry 重读重验（启动预检快照只是起点，不得沿用）：R2／R4／R6、V2／V4／V6 与 `prepared(ctx)` 各重验一次，读取回执、验签与推进见证 head 前再重验一次」
  - `design.md:4098`：「`revocation` 恰为 `revocation_ref, revoked_at, state`」
- 判据：
  - `generation` 在 `framework.md` 只有 2 次命中：`:450`（ADR-0019 的 epoch/generation）与本行 `:1118`；ADR-0031 未定义恢复批准/信任根/见证锚的 generation 字段。
  - design 的批准记录、信任根、`receipt_anchors`、`revocation` keyset 均无 `generation`/`revocation_generation` 字段（`design.md:3934`、`:4098`）；design 中所有 `generation` 均为 `auth_generation`／`execution_generation`／Journal `epoch/generation`，与 restore 撤销无关。
  - `design.md:4037` 的 V 图：`V7` 独立读取 → `V8` 验 producer 签名 → `V9` 读见证 head → `V10` 审计/验证方签名/推进 `sequence`，节点之间没有再调用 `revalidateApproval`；`design.md:4125` 的「Prepare/Replace/VerifyRestored 三点重读」是 r63 期表述，未覆盖「读取回执、验签、推进 head 前」。
  - `recovery_restore.go:247`–`:255` 的 `prepared(ctx)` 只调 `CheckApproved` 与 `checkRestoreArtifact`；`CheckApproved` 是接口，代码中无生产实现可直接核对是否重读批准文件/信任根/撤销状态；`:352` 只是 `tool.Run` 后再调一次 `prepared`。
- 影响：`generation` 不可比较导致「按 generation 重验」为空操作；V7/V8/V10 与推进 head 前没有明确重验点，撤销在子进程启动后到推进见证 head 之间的 TOCTOU 窗口未闭合。
- 修复方向：
  1. 删除 `generation`，或在其参与的 schema 增加单调 `revocation_generation` 并写比较规则；
  2. 把「读取回执、验签、推进 head 前」的重验落到 `V7`/`V8`/`V10`（或明确由哪个既有检查点承担），不得只写 `V2/V4/V6`；
  3. design 指定生产 `CheckApproved` 的重读职责，或在 `prepared(ctx)` 增加明确重读并写测试。
- 证据：上述行原文；`generation` 命中仅 `:450`/`:1118`；design 撤销 keyset 仅三字段；代码 `prepared` 两调用。
- 复算：
  ```powershell
  rg -n 'generation' $p
  rg -n 'revocation' $d | Select-Object -First 6
  $c='cmd/han-server/recovery_restore.go'; $l=Get-Content -Encoding UTF8 -LiteralPath $c; $l[246..254]
  ```

### `SEC-R03-F004` — MEDIUM — 见证独立性集合漏掉 verifier

- 行锚：`framework.md:1147`；`design.md:4033`、`:4060`、`:4061`
- 原句：
  - `framework.md:1147`：「`witness_principal_id`／`witness_key_id` 及其 `org_id` 不得等于 root／executor／producer／issuer／reviewer 任一方」
  - `design.md:4060`：「`witness_principal_id` 的 `org_id` 必须解析到 `organizations.kind = external_verifier`」
  - `design.md:4061`：六席 `reviewer`／`issuer`／`executor`／`producer`／`verifier`／`witness` 的 `person_ref` 两两不等
- 判据：ADR 作为决策级对象，其独立性集合缺 `verifier_principal_id`／`verifier_key_id`；实现若只按 ADR 检查，可让同一主体兼任 witness 与 verifier，破坏「见证独立于验证方」的双人分离。design 侧另有 `:4033`「`witness_principal_id` 不得等于 `verifier_principal_id`」与 `:4061` 六席 `person_ref` 两两不等，说明设计意图是分离，但 ADR 未落。
- 影响：决策级契约与 design 不一致；仅按 ADR 实现时见证与验证方可能同体，外部锚的独立性被削弱。
- 修复方向：`:1147` 加入 `verifier_principal_id`／`verifier_key_id`，并写明 witness 的 `organizations.kind=external_verifier` 与六席 `person_ref` 两两不等。
- 证据：上述行原文。
- 复算：
  ```powershell
  rg -n 'root／executor／producer／issuer／reviewer|witness_principal_id.*verifier_principal_id|六席' $p $d
  ```

### `SEC-R03-F005` — MEDIUM — r09 修订不可独立复算

- 行锚：`framework.md:1190`；`tasks.md:3148`
- 原句：
  - `framework.md:1190`（r07 追溯行末）：「r09 追溯：本次修订基线为 `adr-0031-r07`；`adr-0031-r08` 未产出，按用户 2026-11-02 追加任务跳号使用 `adr-0031-r09`，跳号事实待评审方确认」
  - `tasks.md:3148`：「`framework.md` 268790 B / 1271 行 / SHA256 `70CFC915…AFCA`」（r07 时点）
- 判据：
  - ADR-0031 没有独立的 r09 追溯行；r09 说明附在 r07 行末。追溯区 `:1188`–`:1190` 无 r08/r09 独立条目（`:1071`/`:1072` 是 ADR-0030 的 r08/r09 追溯，不属 ADR-0031）。
  - 当前全文 272,588 B 与 `tasks.md:3148` 的 r07 基线 268,790 B 相差 +3,798 B；没有 r09 后的 B/行数/SHA 日账，也没有逐项变更清单。复算者只能看到「附在 r07 行末的一段话」，无法独立核对 r09 到底改了哪些字节、是否越界改动禁改区。
  - 这是安全侧实质问题：决策级对象在复审时要求变更可复算；当前只能依赖作者叙述。
- 影响：无法排除 r09 在外部锚/撤销同步之外夹带未声明改动；复审与后续追溯缺可靠基线。
- 修复方向：新增独立 r09 追溯行，登记 r07 基线（`268790 B / 70CFC915…`）、r09 后 B/行/SHA（`272588 B / 1271 行 / D8DBA64B…`）、逐项变更清单、r08 跳号原因；后续修订同样登记改前/改后基线。
- 证据：`framework.md:1188`–`:1190` 追溯区；`tasks.md:3148` r07 基线。
- 复算：
  ```powershell
  rg -n 'r07 修订|r09 追溯|adr-0031-r08|r08 修订' $p
  rg -n '268790|70CFC915' .add/project/knowledge/tasks/tasks.md
  ```

### `SEC-R03-F006` — MEDIUM — `tasks.md` 仍指向 r07 并声称 S4.24a 门禁解除

- 行锚：`tasks.md:3197`、`:3211`、`:3217`；对照 `framework.md:1129`、`:1176`
- 原句：
  - `tasks.md:3197`：「`adr-0031-r07` 为现行修订……决策接受级对 r07 成立，故本卡启动门禁解除」
  - `tasks.md:3211`：关闭条件 (1) 仍指向 r07 两路报告
  - `tasks.md:3217`：仍引 design r61 / `EF55DC92…`，并称 r61 被两路 r01 判 CHANGES REQUIRED
  - `framework.md:1129`：「r09 尚未复评……`identity-auth-S4.24a` 及任何实施/验证卡不得启动（启动门禁回到 blocked）」
  - `framework.md:1176`：「现行修订为 `adr-0031-r09`……r09 未获架构与安全两路接受前，`identity-auth-S4.24a` 仍不得启动」
- 判据：`tasks.md` 的「门禁解除」与 ADR 现行状态直接冲突；按 `tasks.md` 阅读的执行者可能据此启动 S4.24a，绕过 r09 未复评的事实。该文件不在本报告写集内，只能由 Task-Planner 对账。
- 影响：交付完整性与门禁一致性风险；不是 ADR 文本缺陷，但会实际放行错误动作。
- 修复方向：由 Task-Planner 仅对账 `tasks.md`：把现行修订/门禁/关闭条件/设计 revision 改为 r09 现状与最新 design 修订；本报告不修改 `tasks.md`。
- 证据：上述行原文。
- 复算：
  ```powershell
  rg -n 'adr-0031-r07|adr-0031-r09|门禁解除|r09 未获' .add/project/knowledge/tasks/tasks.md $p
  ```

### `SEC-R03-F007` — LOW — ADR-0030 `(w,r,token)` 与 ADR-0031 外部见证缺术语消歧

- 行锚：`framework.md:1009`（ADR-0030 见证状态机）、`:1118`、`:1147`；另见 `:450`（ADR-0019 `JournalHeadAnchor` epoch/generation）
- 原句：
  - `framework.md:1009`：「见证只持久 `(w, r, token)`：`w` 为已终结 revision 的最大值……」
  - `framework.md:1118`：「回执与见证 head 均由外部见证以 Ed25519 签名……`anchor_ref` 在 … 五处逐字节相等」
- 判据：两套「见证」共用术语但对象不同：ADR-0030 的 `(w,r,token)` 是 anchor 事件 revision 状态机；ADR-0031 的见证是 receipt `anchor_ref`/head `sequence` 签名者。未发现直接数值不变量冲突，但 `anchor_ref` 易与 ADR-0019 `JournalHeadAnchor` 混淆；也未声明两套 witness endpoint/key/state 必须独立。
- 影响：实现/评审时可能把两套见证状态混用，误以为一套推进可替代另一套。
- 修复方向：加消歧句：`anchor_ref` 指 receipt anchor，不是 ADR-0019 `JournalHeadAnchor`；ADR-0030 `(w,r,token)` 仅属 anchor 事件状态机；两套 witness endpoint/key/state 不得共享。
- 证据：上述行原文。
- 复算：
  ```powershell
  rg -n 'w, r, token|anchor_ref' $p | Select-Object -First 8
  ```

## 4. 已闭合项（含复算命令）

### `SEC-R03-C01` — CLOSED — 签名载荷唯一性与 sequence 单调性

- `framework.md:1118` 的 5 键 head 载荷（`anchor_ref, sequence, issued_at, head_sig_alg, head_sig` 删 `head_sig` 后 JCS）与 `design.md:4033` 一致；8 字段验证方载荷（含 `verified_at, verifier_principal_id, verifier_key_id`）与 `design.md:4031` 一致。
- `sequence` 严格大于 `max(baseline_sequence, head.sequence)`，`baseline_sequence` 只增不减；`design.md:4060` 给出正则 `^(0|[1-9][0-9]*)$`、单调比较与「降低尝试 → `ErrAdmissionBinding`/`restore_unknown`」。
- 与 ADR-0030 `(w,r,token)` 无直接数值冲突；仅保留 F007 的术语消歧建议。
- 复算：
  ```powershell
  rg -n '5 键|8 字段|严格大于|max\(baseline_sequence|只增不减' $p $d
  ```

### `SEC-R03-C02` — CLOSED — V 编号正确（V1–V11）

- `design.md:4037`（r64.5，`:4062` r65.5 复述）现行图：V1 `prepared` → V2 `revalidateApproval` → V3 source.Verify → V4 revalidate → V5 execute → V6 revalidate → V7 独立读取 → V8 验 producer → V9 witness head → V10 审计/验证方签名/推进 → V11 返回。旧 `:3990` 的 V 编号已被 r65.5 就地标记取代。
- 复算：
  ```powershell
  rg -n 'V7.*独立读取|V8.*producer 签名|V9.*见证 head|V10.*推进|r65.5' $d
  ```

### `SEC-R03-C03` — CLOSED — 模拟见证边界明确

- `framework.md:1118`、`:1147`、`design.md:4070` 均写明本机 witness CLI 是测试替身、证据必须标注「模拟见证、未真实验证」，真实实现为外部审批系统／HSM 且当前未验证；未发现把模拟见证当真实证据的放行句。
- 复算：
  ```powershell
  rg -n '模拟见证、未真实验证|外部审批系统／HSM|只是测试替身' $p $d
  ```

### `SEC-R03-C04` — CLOSED — r09 未把未登记 OPEN 写成已具备

- `framework.md` 中 `bundle_digest`、`nonce`、`2048`、`revocation_generation` 均 0 命中；`2048` 只在 `design.md:4033`/`:4060` 作为 `witness_endpoint` 长度上限，不是回执大小判据。r09 本身没有 overclaim。
- 注意（不阻断本项）：若 `bundle_digest`、回执 2048 B 上限、`nonce` 仍是项目所需判据，它们仍是未登记 OPEN，须由后续修订显式登记；本项只判「r09 没把它们写成已具备」。
- 复算：
  ```powershell
  foreach ($k in 'bundle_digest','nonce','2048','revocation_generation') { "$k=" + (rg -c -- $k $p 2>$null) }
  ```

### `SEC-R03-C05` — CLOSED — 三级门禁消除自指死锁

- `framework.md:1122` 决策接受级 = 两路独立评审接受 + 四项决策级要素齐备，只解除派发/认领/开始改动；`:1124` 卡证据级是退出条件、不是启动前置；`:1126` 生产解除级需约束 8 全项 + 人的明确生产批准；`:1176`「登记≠启动」。
- 结构上消除「实施卡等 ADR、ADR 等实施卡产证据」的自指。**但** r09 本身未达第 1 级（见 F001–F005），S4.24a 仍 blocked；本项只判门禁结构，不判 r09 是否被接受。
- 复算：
  ```powershell
  rg -n '决策接受级|卡证据级|生产解除级|登记≠启动' $p
  ```

## 5. 未运行项（单列，任何 NOT RUN 仍是 NOT RUN）

- 未运行 `go test ./...`、`go build`、`go vet` 或任何 Go 测试。
- 未启动服务、未连接 PostgreSQL、未做数据库替换或回退演练。
- 未接触真实 Linux host、ACL/GRANT、mount、loader、fd-exec、`/proc/self/maps`、network namespace。
- 未接触真实旧备份、真实 restore 工具、真实批准记录、真实信任根、真实外部 witness／HSM／TLS 端点、审计 sink。
- 未做撤销/过期/TOCTOU 的运行时实测；F003 的窗口只做文本与代码路径核对。
- 未做真实 E2E；未由真人签字。
- design 可能正被 r66 并行修订；本报告只对 r65（`87D70DA8…`）负责，若 r66 改变被引段落须重审。

## 6. 未验证风险

- F001/F003 的不可判定性来自文本与 schema 核对，未用实现/测试证明某条具体路径会被绕过；但「不可判定」本身已足以阻断决策接受级。
- F002 的 root 威胁是设计自述的残余；本报告未做攻击演示。
- F006 的实际放行后果依赖执行者是否先读 `tasks.md`；未做组织流程验证。
- 未验证 design r65 与 `cmd/**` 未跟踪改动之间是否还有其它实现偏差。

## 7. 复审条件

1. `framework.md` 在 r09 基线（`272588 B / 1,271 行 / D8DBA64B…`）上产出 r10 修订，且 F001–F004 在 ADR 与 design 同步修复并附复算命令。
2. F005 增独立 r09/r10 追溯行，含改前/改后 B/行/SHA 与逐项变更清单。
3. F006 由 Task-Planner 对账 `tasks.md` 至 r09/r10 现状（本报告不改该文件）。
4. 修复后由**新的**安全侧独立评审（非本报告作者）重跑本报告清单；架构侧接受为合取条件，单独安全侧通过不解锁。
5. 复审时须重新实测对象 B/行/SHA 与 design revision；若 design 已到 r66，须重新核对被引段落。

## 8. 声明

本报告由 AI 评审角色（add-reviewer，安全视角）产出，**非真人签字，不构成安全签收、人的成果接纳或生产批准**；**不解锁任何卡**（`identity-auth-S4.24a` 仍不得启动）；**不解除生产 closed**；不关闭 `identity-auth-D06`。是否接受 r09、是否承担上述残余风险，由 Lead/用户决定。