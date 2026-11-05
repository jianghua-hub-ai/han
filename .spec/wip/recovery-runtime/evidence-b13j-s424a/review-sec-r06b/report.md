# SEC r06b 独立复审报告（Han identity-auth design r66 冻结字节）

## 0 元信息与基线

- 日期：2026-11-02
- 评审角色：安全侧 Reviewer（第二轮独立复审，冻结字节判读）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 唯一评审对象（冻结快照）：`%TEMP%\design-r66-freeze-9B193BDE.md`，**1,079,559 B / 5,476 行 / SHA256 `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298`**，版本 `identity-auth-design-ir01-r66`（与派发冻结值逐项相等，本人实测）
- 工作树 `design.md` 现行（漂移观察，不参与判读）：**1,084,882 B / 5,476 行 / SHA256 `B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E`**，版本已为 `identity-auth-design-ir01-r67`（`:5`／`:38`）；与冻结快照相差 **12 行**：`:5, :38, :3942, :3990, :4062, :4072, :4073, :4076, :4135, :4137, :4369, :4373`
- 上一轮安全报告（判据来源）：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r05/report.md`，39,093 B / 294 行 / `4A2292674E49EB4A7961CD1EDD0FA4BCF1560D2E911B74AE1E208AC3790DE0C1`（Verdict CHANGES REQUIRED）；更早基准 `review-sec-r04/report.md`，40,794 B / 292 行 / `F98B39D19083E7A010680465C40404AEEE23E95C65F48C103602F7758632E898`
- 作者修订证据：`design-revision-r06/report.md` 24,041 B / 197 行 / `CA5F28AC27DC4951AA6D4335F4DC2DE5CF0C714E3CCBF79C0E3EA93FF8A7C671`；`design-revision-r06/hashes.txt` 7,343 B / 117 行 / `ABF946B7B0ED4577703D3A468D93476C1A730AFFD2D2C8D5FB3987EDB255E594`
- 并行安全报告（仅对照，不作接受依据）：`review-sec-r06/report.md` 28,417 B / 232 行 / `31BAB8B0F67A2EC8706E52E8B897534E14B196B16FA4F84F1B7208A50F291B08`（自述中止）
- ADR 观察（并行会话改写中，只记不改）：`decisions/framework.md` 285,639 B / 1,273 行 / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`（本轮开始与结束同值），`:1082` = `adr-0031-r10`
- 任务卡：`tasks.md` 898,536 B / 3,289 行 / `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`；`:3186` 仍为 `- [ ]`
- 方法：按 `add-reviewer.toml` Role 全文与 `add-code-review/SKILL.md`（objectType = design）执行；加载 `code-review-checklist`、`abstraction-principles`、`decisions-guide`、`design-doc-writing-style`；代码提交／编译／构建条款按非代码对象裁掉，只保留可机核的契约、安全与证据链判据。
- 两轮实际检查范围：（第一轮）完整读取 r66 块 `:4072`–`:4083`、r65 锚点 `:3936`／`:3937`／`:3988`／`:3990`／`:4012`／`:4037`／`:4057`–`:4070`、`:1718`、`:4435`，读上一轮 r04／r05 报告、作者 r06 证据、并行 r06 报告；独立复算两条 golden vector 与六段块／段哈希。（第二轮）对冻结快照重跑 `V8.*r65.2` 机械判据（命中 0）、V8 出现行清单（`:3990/:4037/:4062/:4074`）、角色字面统计、冻结快照与工作树逐行差异、N005 三处跨文件对账，并核 `:4141` 的 wire 类型句与 `:4066` 历史行号映射。
- **声明：本轮只读，未修改任何被评审制品；未改 `design.md`、`tasks.md`、ADR、`cmd/**`、`internal/**`、测试、既有评审目录或作者证据；无任何 Git 写操作（未 commit／branch／stash／reset），未删除、移动或归档任何文件。**

## 1 关于并行 `review-sec-r06` 的「中止」处置

- **不以其为接受依据**：该报告明确自述「本次评审未完成，不能作为 r66 的安全侧接受」，本报告独立判读，未引用其结论作为通过或失败的依据。
- **对其「中止」处置的独立判定：在其自身派发语境下可接受，但**不**适用于本任务。** 该报告记录的派发基线是 `0BA1C6C1…`（1,078,886 B），随后观察到对象变为 `9B193BDE…`，属于「派发值与现实不符」的漂移；在缺少冻结快照时中止并等待重新冻结，是稳妥处置。本任务（r06b）已显式给出与冻结值逐字节相等的只读快照 `%TEMP%\design-r66-freeze-9B193BDE.md`，评审期间快照不变、仅工作树漂移，故按派发要求「记录漂移、以冻结快照继续判读」推进，不中止全部评审。
- **其 provisional findings 的复现情况**：
  - `NEW-S06-01`（作者证据链不自洽，MEDIUM）：**不复现**。作者 `hashes.txt` 现记录 r06 终值 `1,079,559 B / 5,476 行 / 9B193BDE…`，与 `report.md`／`hashes.txt` 互值一致，且等于本任务冻结值；其旧 `1,045,042 B / D2A6B54B…` 已被显式标为历史作废值（`:4072`、`:4080`）。
  - `NEW-S06-02`（V10 推进前无显式撤销／有效期重验，LOW provisional）：在 design r66 冻结字节内**不构成 design finding**。`:4135` 的五点清单确实只在预检、`Prepare`、`Replace`（启动前两次）、`VerifyRestored`（`prepared()` 与启动前）重验，`:4137` 只补「子进程返回后、解码回执前」；ADR r10 在 `framework.md:1118`／`:1174` 把撤销与有效期重验扩到「含 `P6` 的完整检查点集合」，属 ADR 侧追加的关闭判据，超出 r66 V 图。记作**跨文件分层观察**，不计 design finding。
  - `NEW-S06-03`（批准记录缺 `generation` 可核定义，LOW provisional）：**不复现**。ADR r10 `:1118` 已给出 `generation` 的可复算三元定义（`approval_generation`＝批准文件精确字节 SHA-256；`trust_root_generation`＝信任根文件精确字节 SHA-256，必须等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`；`anchor_generation`＝`receipt_anchors` 该 `anchor_ref` 条目 JCS 字节的 `sha256:` 摘要）。design `:4369` 无该字段但无相反定义，属**跨文件观察**，不计 design finding。
- 结论：并行 r06 的「0 findings ≠ 通过」态度正确；其 `NEW-S06-01／03` 不复发，`NEW-S06-02` 只作分层观察。其报告不作为本报告任何 PASS 或接受依据。

## 2 结论

**Verdict: CHANGES REQUIRED**

| 计数项 | 数量 | 明细 |
|---|---:|---|
| 范围内 BLOCKER | 0 | — |
| 范围内 HIGH | 0 | — |
| 范围内 MEDIUM（开放） | 1 | `NEW-S05-05`（信任根配置键名冲突） |
| 范围内 LOW（开放） | 2 | `NEW-S05-06`（V 编号措辞过窄）、`NEW-S05-07`（pin 轮换回滚窗口语义） |
| NOT RUN | 1 | `SEC-F007`（Linux 侧能力实测） |
| 范围外 BLOCKER（不计入上表，阻断启动） | 1 | `N005`（ADR／tasks 状态冲突，仍存在） |
| 已闭合（设计层，无回退） | 13 | `SEC-F003`、`N001`、`N011`、`NEW-S05-01`、`NEW-S05-02`、`NEW-S05-04`、`SEC-F002`、`SEC-F006`、`N012`、`N013`、`N014`、`N015`、`N016` |

判据：上一轮全部 BLOCKER／HIGH 在设计层已闭合，本轮无回退；但出现 1 项 MEDIUM 跨文件配置键名冲突，可导致信任根读取／校验落空或接受未注册别名，故不能给 PASS。

## 3 逐项判定（对照 review-sec-r05 开放项）

| 项 | 上一轮级别 | 本轮判定 | 冻结快照证据（行号） |
|---|---|---|---|
| `SEC-F003`＋`N001` | BLOCKER | **已闭合（设计层）** | `:4074` 五检查点逐项绑定唯一载荷／签名主体；V8 唯一引用 `:3988` producer envelope（`JCS({"receipt":R10,"attestation":A6})`）；8 字段载荷只在 `:4059` 定义且 `:4074` 明示「只服务 V10」；`:4062` V8 子句内「8 字段只用于 V10、定义以 r66.2 为准」；`:3990` 旧步骤整段标历史。独立机械判据 `V8.*r65.2` = **0**；V8 仅在 `:3990`（历史）／`:4037`（图）／`:4062`／`:4074` 出现。残余 LOW 见 `NEW-S05-06` |
| `N011` | HIGH | **已闭合（设计层），残余 LOW** | `:4077` 单次解析→固定 IP 集合→`DialContext` 只连集合→连接后 `RemoteAddr.Unmap()`∈集合；`witness_host` 必须 DNS 名并拒 `127.1`／`0x7f…`／`0177…`／`::ffff:127.0.0.1`；`ServerName=witness_host` 使 SNI／pin／IP 同连接绑定；pin 唯一路径（`InsecureSkipVerify`＋`VerifyConnection` 三项）；轮换须人批准留审计；独立性链 `witness_key_id→keys.principal_id→principals.org_id` 且 org 为 `external_verifier`。残余：DNS 投毒／同管理域冒充为部署前置（`r65.13(3)`），轮换窗口语义见 `NEW-S05-07` |
| `NEW-S05-01` | HIGH | **已闭合（设计层）** | `:4078` `environment_dirs` 绝对规范路径、`st_uid==0`、非 group／other／executor 可写、逐级 `lstat` 无符号链接、挂载至少 `ro,nodev,nosuid`；loader 父目录（`pt_interp_path`／`ld_so_cache_path`／`ld_so_conf_paths`）不得在目录集内且不得可写；禁 `exec.LookPath`／裸名；拒 `PYTHONPATH`／`NODE_PATH`／`RUBYLIB`／`PERL5LIB`；≤32 目录、`PATH` 拼接 ≤4096 B 且逐字节等于目录以 `:` 连接。Linux 权限实测 NOT RUN |
| `NEW-S05-02` | MEDIUM | **已闭合** | `:4072` 唯一优先级 `r66>r65>r64>r63>r62>r61`，同修订小节号大者胜；`:4078` 明示 `:4435`（r61.5）「禁止 `PATH`」由 r66.6 取代；`:4435` 字节不变（364 B / `429E633A…`），仅语义被覆盖，无并存歧义 |
| `NEW-S05-04` | LOW | **已闭合（设计层），残余 LOW** | `:4077` `witness_endpoint` ≤2048 B、独立 `witness_host`、`witness_key_id→…→org_id` 链、pin 轮换、`root`／`supplier` 的 `appointment_ref` 只允许空串且不占六席；`:4061` 尾 r66.5 指针落点一致。残余见 `NEW-S05-07` |
| `SEC-F002` | BLOCKER | **无回退** | `:3936(3)` 已删 principal 侧 `role_name` 比较；`:4061` 六值封闭枚举（`restore_approver/reviewer/executor/producer/verifier/witness`）、裸名须显式映射、`person_ref` 全局唯一、六席 `person_ref` 两两不等；`:3937` 明示可从信任根机核、不依赖调用方自报；principal 只能来自已登记信任根，无调用方自报旁路 |
| `SEC-F006` | HIGH | **无回退** | `:4057` 登记 3 项放宽（`NoNewPrivs`／`CapBnd`／`PATH`）与 1 项容量变更（信任根上限）；`:4064` `CapInh/CapPrm/CapEff/CapAmb=0`、`Uid/Gid` 四元组、`Groups` 空、`AT_SECURE=0` 仍为拒绝条件，`CapBnd`／`NoNewPrivs` 仅观测；`:4063` ENV 白名单 `LANG/LC_ALL/TZ/PATH`，`LD_*`／`*_PRELOAD`／`*_OPTIONS`／`GLIBC_*`／`MALLOC_*` 即 closed；`PATH` 与 `LD_*`／解释器变量路径由 `:4078` 收口 |
| `N012` | LOW | **无回退** | `:1718` CHECK = `han-restore-receipt/v1`；`:4065` 点名第三别名 `restoreReceiptWire` 改造后 `rg` 必须为 0 |
| `N013` | LOW | **无回退** | `:4066` 删除 `Target.DSN` 改 `appendJSONString` 直写，`rg Target\.DSN` 改造后为 0；`:4141`（r62.4 历史 `:4129` +12）仅约束 `restorePlanWire`／`restoreReceiptWire` 字段类型仍 `string`，二者无 DSN 字段，不冲突 |
| `N014` | LOW | **无回退** | `:4067` `static` 为默认与首选，`dynamic` 仅在无 `LD_*`、`PT_INTERP` 来源、闭包摘要逐项相等、`DT_NEEDED` 并入、父目录与挂载规则覆盖等合取成立时允许，无法证明即 `ErrAdmissionUnsupported` |
| `N015` | LOW | **无回退** | `:4062` `d.prepared(ctx)` 归 R6 第二次同构复核，`execute` 后必须保留 |
| `N016` | LOW | **无回退** | `:4012` 基线字段统一 `baseline_sequence`，见证推进 head `sequence` |
| `SEC-F007` | NOT RUN | **NOT RUN（不得改判 PASS）** | 见第 5 节 |

## 4 新发现详情

### `NEW-S05-05`（MEDIUM）信任根配置键名冲突：r65.13 短名覆盖 r62 注册长名

- 位置：冻结快照 `:4070`（r65.13，B1「设计级处置」）vs `:4088`（r62.1）与 `:4195`（r62.6.7 注册清单）。
- 事实：`:4070` 的规范配置键列表写 `HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH`／`HAN_RECOVERY_RESTORE_TRUST_ROOT_SHA256`；`:4088` 与 `:4195` 写 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH`／`HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`，且 `:4195` 明示四个键「全部加入 `recovery_config.go` 的 `recoveryRootConfigurationKeys`」。ADR r10 `framework.md:1118` 亦引用长名 `…APPROVAL_TRUST_ROOT_SHA256`。全文件仅 `:4070` 出现短名，`:4088`／`:4195` 出现长名（本人正则统计）。
- 违反依据：`design-doc-writing-style`／`decisions-guide` 的「同一契约事实只能有一个权威字面」；`review-sec-r04` 的 `N011`／信任根固定要求。按 `:4072` 的唯一优先级 `r65 > r62`，短名在语义上成为现行字面，但注册清单只有长名。
- 触发场景：实现者按优先级取短名读取信任根，而 `readAction` 只从注册长名取值 → 短名未被读取／校验，或实现者同时接受两套键名 → 接受未注册别名，信任根固定摘要落空。
- 影响：信任根路径／摘要可能未按批准记录固定，破坏「验签公钥只来自 `receipt_anchors.public_key`、信任根摘要固定」的准入前提；属配置解析层的安全相关歧义。
- 修复建议（交作者／Lead 决定）：把 `:4070` 的短名改为与 `:4088`／`:4195`／ADR r10 一致的长名，或显式声明「短名只作历史、出现即 `ErrAdmissionBinding`」并同步注册清单；在 r66.12 增加「短名不得接受」的可机核判据（如 `rg HAN_RECOVERY_RESTORE_TRUST_ROOT_ design.md` 命中只能出现在历史声明）。
- 证据：上述行原文与正则命中；`framework.md:1118` 长名引用。

### `NEW-S05-06`（LOW）`:4079` 的 V 编号范围措辞过窄

- 位置：`:4079`（r66.7）vs `:4037`（r64.5）／`:4062`（r65.5）／`:4074`（r66.2）。
- 事实：`:4079` 写「唯一有效的 V 编号是 `:4062` 的 V7–V11」；但 `:4037` 的 R／V 图明确列 `V1`–`V11`（`V1 prepared`…`V11`），`:4062` 亦写「`R1`–`R8` 与 `V1`–`V11` 只枚举契约检查点」，`:4074` 只覆盖 V7–V11 的载荷绑定。三处并存可被读成「V1–V6 不再有效」。
- 影响：验证方实现者可能误删或跳过 V1–V6 前置检查点；不改变载荷唯一性，但造成契约检查点集合歧义。
- 修复建议：把 `:4079` 改为「唯一有效的**载荷绑定** V 编号是 V7–V11；V1–V6 仍为前置检查点」，与 `:4037`／`:4062` 对齐。
- 证据：`:4037`、`:4062`、`:4079` 原文。

### `NEW-S05-07`（LOW）pin 轮换回滚窗口语义不闭合

- 位置：`:4077`（r66.5）「pin 轮换」句。
- 事实：同句同时写「任一时刻只有一个 active pin」「不做双 pin 同时接受」「旧 pin 只作为部署回滚窗口保留，窗口外失败即 closed（`ErrAdmissionUnsupported`／`ErrAdmissionBinding`）」。缺：窗口长度、判定时点、窗口内旧 pin 是否可被接受（与「不做双 pin 同时接受」的关系）、轮换批准与审计记录的字段／绑定。
- 影响：轮换期内可能出现「旧 pin 是否仍接受」的实现分歧；若实现为窗口内双接受，则与同句前半冲突；若窗口内不接受，则「回滚窗口」无实际语义。属 LOW 契约歧义，不是当前 closed 状态的越权路径。
- 修复建议：明确「任一时点只有一个 active pin；旧 pin 只保留为**离线回滚材料**，任何请求路径都不接受旧 pin；窗口长度与生效时点由批准记录固定，轮换批准须含旧／新 SPKI 摘要与生效时间并留审计」。
- 证据：`:4077` 原文。

## 5 NOT RUN（不得改判 PASS）

- `SEC-F007`：**仍为 NOT RUN**。本轮未在 Linux host 实跑，缺以下运行条件：Linux 主机（非 Windows 解析层）、POSIX ACL 与 `st_uid/st_gid/st_mode` 实测、mount 选项（`ro,nodev,nosuid`）实测、loader（`pt_interp_path`／`ld_so_cache_path`／`ld_so_conf_paths`）解析实测、fd-exec（`O_CLOEXEC`／`O_NOFOLLOW`／`execve`）实测、`/proc/self/maps` 与 `AT_SECURE` 实测、network（DNS 解析／`DialContext`／`RemoteAddr`／TLS pin）实测、外部 witness 实测。
- 其余 NOT RUN：`NoNewPrivs`／`CapBnd` 仅为事后观测值，不构成隔离已获得；`dynamic` 可证明形态、mount 父目录覆盖、`environment_dirs` 逐级 `lstat` 均未实测；`bundle_digest` 仍为显式 OPEN（首期不实现）。
- 因此本报告不声称：生产 restore 解锁、`enter`／`resume` 解锁、约束 8 满足、`N005` 已解决、`identity-auth-S4.24a` 可启动或可关闭。

## 6 范围外观察（不计入计数）

- **`N005`（范围外 BLOCKER，仍存在，阻断启动）**：`decisions/framework.md:1084` 写明「本修订 `adr-0031-r10` 的两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动」；`:1129` 写明「现行修订须同时取得架构与安全两路独立评审接受，才解除实施/验证卡的派发、认领与开始改动限制」；而 `tasks.md:3197` 仍称「`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受……本卡启动门禁解除」。三者对「现行修订是哪一版、门禁是否解除」给出相反事实；`tasks.md:3186` 仍为 `- [ ]`。该冲突在 design 修订范围之外，本轮不改 ADR、不改 `tasks.md`，但它**阻断 `identity-auth-S4.24a` 启动**的性质未变；须由架构决策方与 Task-Planner 对账。
- **对象漂移**：工作树 `design.md` 在评审期间已变为 r67（12 行差异，含 `:4135`／`:4137`／`:4369`／`:4373` 的疑似 V10 前重验与 `generation` 相关改动）。本报告**只**对冻结快照 r66 判读；r67 的上述改动**未被**本报告视为已闭合，须在下一轮按新冻结值重新判定。
- **ADR 分层观察**：ADR r10 在 `framework.md:1118` 把撤销／有效期重验扩到含 `P6` 的检查点集合、给出 `generation` 三元定义，均超出 design r66 的 V 图；这是 ADR 侧追加关闭判据，不是 design r66 缺陷，也不构成 design r66 已满足该追加判据。ADR 的接受由架构侧独立给出，本报告不代判。

## 7 方法与限制

- 只读核对：所有行号、字节、行数、SHA256 均由本人对冻结快照与工作树复算；golden vector source（`:4099`，1,115 B，`de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab`）与 tool（`:4105`，2,357 B，`8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508`）均独立复算一致，非 ASCII 字节 0；向量内 `role_name` 仅 `restore_approver`×2、`restore_reviewer`×2、`restore_executor`×1，全部落在 `:4061` 六值封闭枚举内。
- 机械判据：`V8.*r65.2` 命中 0；V8 出现行 `:3990/:4037/:4062/:4074`；六个块／段哈希与 r66.0 声明逐项相等（见 `hashes.txt`）。
- 限制：非代码对象不要求编译／单测；设计层闭合不等于实现层正确，也不等于 Linux 实测通过。残余风险以第 5 节 NOT RUN 与第 6 节范围外观察为准。

## 8 结论声明

本报告是安全侧独立复审意见，**不构成**人的成果接收、独立评审通过、安全签收或生产装配批准；**不构成**生产 restore 解锁、`enter`／`resume` 解锁、约束 8 满足、`N005` 已解决、`identity-auth-S4.24a` 可启动或可关闭。`SEC-F007` 仍为 **NOT RUN**，不得改判 PASS。