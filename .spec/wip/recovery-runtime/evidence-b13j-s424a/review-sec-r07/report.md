# SEC r07 独立复审报告（Han identity-auth design r67 冻结字节）

## 0 元信息与基线

- 日期：2026-11-02
- 评审角色：安全侧 Reviewer（独立复审，非作者、非 Lead）；按 `add-reviewer` Role 与 `add-code-review`（objectType = design）执行。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- **评审对象（冻结）**：`.add/project/knowledge/design-workspace/identity-auth/design.md`，**1,084,882 B / 5,476 行 / SHA256 `B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E`**，版本 `identity-auth-design-ir01-r67`（`:5`、`:38`）。
- **冻结快照**：`%TEMP%\design-r67-freeze-B12F1EF3.md`，本人实测与工作树逐字节相等（同 B／行数／SHA256）；评审判读以快照为准。
- **r66 基线快照**：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r07/design.r66.baseline.md`，1,079,559 B / 5,476 行 / `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298`（用于 r66→r67 逐行 diff）。
- **基准报告**：
  - `review-sec-r05/report.md`：39,093 B / 294 行 / `4A2292674E49EB4A7961CD1EDD0FA4BCF1560D2E911B74AE1E208AC3790DE0C1`（CHANGES REQUIRED）。
  - `review-sec-r06b/report.md`：19,005 B / 111 行 / `169C1116006AFF4526290F3BEC2046A3814B2A4A58C2EB7F0A41251385FF3A7F`（CHANGES REQUIRED；记录 3 项未闭合：`NEW-S05-05`／`NEW-S05-06`／`NEW-S05-07`）。
  - `review-sec-r06/report.md`：28,417 B / 232 行 / `31BAB8B0F67A2EC8706E52E8B897534E14B196B16FA4F84F1B7208A50F291B08`（并行会话因对象漂移自述中止；**仅作线索，不构成接受**）。
- **作者证据（只作线索）**：`design-revision-r07/report.md` 18,099 B / 203 行 / `B4AED3EDE54938FFE43687F6A052707D28AFDD567AB6CACBA04A7B26C879F4EB`；`hashes.txt` 1,847 B / 37 行 / `33EDAD36475A161C37C9DB8926916FED7FC2592F7F1AB0204A6DCE5DC943D604`。
- **任务卡**：`.add/project/knowledge/tasks/tasks.md`，898,536 B / 3,289 行 / `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`；`:3186` 仍为 `- [ ]`。
- **ADR 观察（并行改写中，只记不改）**：`.add/project/knowledge/decisions/framework.md`，285,639 B / 1,273 行 / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`（本轮开始与结束同值）；`:1082` 现行修订 `adr-0031-r10`。
- **方法**：读取 `add-reviewer.toml`、`add-code-review/SKILL.md`（objectType=design）、`code-review-checklist.md`、`abstraction-principles.md`、`decisions-guide.md`；代码提交／构建／编译条款按非代码对象裁掉，只保留可机核的契约、安全、证据链判据。
- **两轮实际检查范围**：
  - 第一轮（对象层）：对 r66 基线快照与 r67 冻结快照做逐行 diff，确认仅 12 行不同（`:5`、`:38`、`:3942`、`:3990`、`:4062`、`:4072`、`:4073`、`:4076`、`:4135`、`:4137`、`:4369`、`:4373`），合计 **+5,323 B**、行数不变；完整读取这 12 行及 r65/r66 锚点 `:1718`、`:3936`、`:3937`、`:3988`、`:3990`、`:4012`、`:4037`、`:4057`、`:4059`、`:4060`、`:4061`、`:4062`、`:4063`、`:4064`、`:4065`、`:4066`、`:4067`、`:4070`、`:4074`、`:4076`、`:4077`、`:4078`、`:4079`、`:4088`、`:4099`、`:4105`、`:4135`、`:4137`、`:4141`、`:4195`、`:4369`、`:4373`、`:4435`；读取 r05／r06b／r06／作者 r07 报告；独立复算两条 golden vector；统计信任根配置键短／长名；核对 r66→r67 字节账与 r65→r66 重建。
  - 第二轮（逐 finding 复核）：对 `NEW-S05-05`／`NEW-S05-06`／`NEW-S05-07` 回到原 finding 与 r67 落点；对并行 `NEW-S06-01`／`NEW-S06-02`／`NEW-S06-03` 逐条复核；对 13 项无回退项用 12 行 diff 加锚点原文核对；复核 `N005` 三处跨文件；复核 end-state 哈希（design／framework／tasks）；复核 V10 重验在 `:4062`／`:4135`／`:4137` 的可机核句、`generation` 三元定义与 ADR `:1118` (i)(ii)(iii) 的对应；复核 B1 重验枚举（`NEW-S07-01`）。
  - **限制**：未逐行通读全部 5,476 行业务章节；只覆盖 restore／ADR-0031 安全契约相关范围；非代码对象不要求编译／单测；Linux 实测仍 NOT RUN。
- **只读声明**：本轮只读核对；**未修改 `design.md`、`tasks.md`、ADR（`framework.md`）、`cmd/**`、`internal/**`、测试、既有 `review-*/`、`design-revision-*/`；无任何 Git 写操作（未 commit／branch／stash／reset）；未删除、移动或归档任何文件。** 唯一写入为本报告与同目录 `hashes.txt`。

## 1 结论

**Verdict: CHANGES REQUIRED**

| 计数项 | 数量 | 明细 |
|---|---:|---|
| 范围内 BLOCKER | 0 | — |
| 范围内 HIGH | 0 | — |
| 范围内 MEDIUM（开放） | 1 | `NEW-S05-05`（信任根配置键短／长名冲突未消除） |
| 范围内 LOW（开放） | 3 | `NEW-S05-06`（V 编号范围措辞）、`NEW-S05-07`（pin 轮换窗口）、`NEW-S07-01`（B1 重验枚举未随 r67.5 同步） |
| 范围内开放项合计 | 4 | 1 MEDIUM + 3 LOW |
| NOT RUN | 1 | `SEC-F007`（Linux 侧能力实测） |
| 范围外 BLOCKER（不计入主体计数，阻断启动） | 1 | `N005`（ADR／tasks 状态冲突，仍存在） |
| 已闭合（设计层；13 项无回退 + 2 项并行） | 15 | `SEC-F002`、`SEC-F003`、`N001`、`SEC-F006`、`N011`、`NEW-S05-01`、`NEW-S05-02`、`NEW-S05-04`、`N012`–`N016`、`NEW-S06-02`、`NEW-S06-03` |
| 已处置（作者证据链） | 1 | `NEW-S06-01`（r66→r67 一手配平；r65 列保留重建限制） |

**阻断计数**：范围内无 BLOCKER／HIGH；开放 MEDIUM 1 项、LOW 3 项，合计 4 项，故不能 PASS。`SEC-F007` 为 NOT RUN，不得改判 PASS。`N005` 为范围外 BLOCKER，阻断 `identity-auth-S4.24a` 启动，不计入主体计数。r67 已闭合 V10 重验与 `generation` 三元定义，并一手配平 r66→r67 字节账；但 `NEW-S05-05` 的配置键冲突仍在，且 r67.6 在 `:4373` 新增长名 SHA256 后，按 `:4072` 的优先级规则，`TRUST_ROOT_PATH` 的现行字面仍是短名、`TRUST_ROOT_SHA256` 的现行字面变为长名，注册清单只有长名——歧义没有被消除，反而按键分裂。

## 2 逐条判定（对照 r05／r06b 开放项）

| 项 | 前轮级别 | r67 判定 | 证据与一句话依据 |
|---|---|---|---|
| `NEW-S05-05` | MEDIUM | **仍开放（MEDIUM）** | `:4070` 用短名 `HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH`／`..._SHA256`（各 1 次）；`:4088`／`:4195`／`:4373` 用长名 `..._APPROVAL_TRUST_ROOT_PATH`（2 次）／`..._APPROVAL_TRUST_ROOT_SHA256`（3 次）；`:4195` 注册清单只有长名；无别名拒绝句；r67 未改 `:4070`／`:4088`／`:4195`。详见 §3.1 |
| `NEW-S05-06` | LOW | **仍开放（LOW）** | `:4079` 仍写「唯一有效的 V 编号是 `:4062` 的 V7–V11」；`:4037` 列 V1–V11，`:4062` 写「`R1`–`R8` 与 `V1`–`V11` 只枚举契约检查点」，r67.2 改后的 `:3990` 也写「现行 V1–V11」。范围未限定。详见 §3.2 |
| `NEW-S05-07` | LOW | **仍开放（LOW）** | `:4077` 的 pin 轮换句仍缺窗口长度、判定时点、批准与审计字段绑定，以及窗口内旧 pin 是否可被接受。r67 未改该行。详见 §3.3 |
| `NEW-S06-01` | MEDIUM（并行） | **已处置（作者证据链）** | r66→r67 一手 diff 10 个语义行（另 2 行版本串）、+5,323 B，与作者表逐项相等；r65→r66 重建 16 行 +3,307 B、插入块 31,141 B（hash `587471F5…`）、合计 34,448 B，且备份 1,044,640 B + R05 13 行 +471 B = 1,045,111 B；内容级 23 行差异与作者一致。**限制：原始 r65 文件不可得，r65 列是重建值，非一手。** 详见 §4.1 |
| `NEW-S06-02` | LOW（并行） | **已闭合（设计层）** | `:4062`（V10）新增「验证方签名后、外部见证原子推进 `sequence` 前再调用一次 `revalidateApproval`」；`:4135` 其六、`:4137` 尾句给出同一判据；失败不得推进 head、按 `restore_unknown` 处置。跨文件 ADR 的 V7/V8 追加要求仍 OPEN，见 §4.2、§7 |
| `NEW-S06-03` | LOW（并行） | **已闭合（设计层）** | `:4369` 指向 `:4373` 的三元定义：`approval_generation`＝批准文件精确字节 SHA-256；`trust_root_generation`＝信任根精确字节 SHA-256 且等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`；`anchor_generation`＝`receipt_anchors` 该 `anchor_ref` 条目 JCS 的 `sha256:` 摘要且等于 policy／回执 `trust_anchor_sha256`。与 ADR `:1118` (i)(ii)(iii) 一致；长名引用反向加重 §3.1。详见 §4.3 |
| `SEC-F007` | NOT RUN | **NOT RUN（不得改判 PASS）** | `:4070(6)` 明列 Linux host／ACL／mount／loader／fd-exec／maps／network／witness 全部未运行；Windows 解析层不等于 Linux 实测。详见 §6 |
| `N005` | 范围外 BLOCKER | **仍存在（不计入主体）** | `framework.md:1084` 明写 `adr-0031-r10` 两路接受尚未取得、`identity-auth-S4.24a` 不得启动；`:1129` 明写现行修订须重新取得架构与安全两路接受；`tasks.md:3197` 仍称 `adr-0031-r07` 为现行修订、已接受、门禁解除；`:3186` 仍 `- [ ]`。只登记，不代改。详见 §7 |

## 3 新发现与仍开放项详情

### 3.1 `NEW-S05-05`（MEDIUM）信任根配置键短／长名冲突未消除，且 r67.6 使其按键分裂

- **位置**：`:4070`（r65.13 B1「批准记录载体与读取路径」）vs `:4088`（r62.1）、`:4195`（r62.6.7 注册清单）、`:4373`（r67.6 `generation` 定义）；ADR 侧 `framework.md:1118`。
- **事实（本人正则统计，全文）**：
  - `HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH`：**1 次**，仅 `:4070`。
  - `HAN_RECOVERY_RESTORE_TRUST_ROOT_SHA256`：**1 次**，仅 `:4070`。
  - `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH`：**2 次**，`:4088`／`:4195`。
  - `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`：**3 次**，`:4088`／`:4195`／`:4373`。
  - `:4070` 原文：「配置键 `HAN_RECOVERY_RESTORE_APPROVAL_PATH`、`HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH`、`HAN_RECOVERY_RESTORE_TRUST_ROOT_SHA256`」。
  - `:4195` 原文：「新增键 `HAN_RECOVERY_RESTORE_APPROVAL_PATH`、`HAN_RECOVERY_RESTORE_TARGET_DSN_FILE`、`HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH`、`HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`，全部加入 `recovery_config.go` 的 `recoveryRootConfigurationKeys`」。
  - 全文检索「别名／alias／未注册／短名」未发现针对这两个信任根键的别名拒绝条款；`:4079` 的「旧基线字段名出现即 `ErrAdmissionBinding`」是 JSON 基线字段规则，不是 env 配置键规则。
- **优先级复算**：`:4072`（r67.0）规定「修订号大者覆盖小者，`r67 > r66 > r65 > r64 > r63 > r62 > r61`；同一修订内小节编号大者覆盖小者；跨修订字面冲突一律按此判定，与条文出现位置、出现次数、行号先后无关」。据此：
  - **PATH 半边**：`:4070`（r65.13）短名 > `:4088`／`:4195`（r62.1／r62.6.7）长名 → **短名成为现行字面**；但注册清单只有长名。
  - **SHA256 半边**：`:4373`（r67.6）长名 > `:4070`（r65.13）短名、`:4088`／`:4195` 长名 → **长名成为现行字面**；但 B1 仍展示短名。
  - 结果：同一组信任根配置键按 PATH／SHA256 分裂成两种命名；B1 的两个短名与注册清单／ADR／r67.6 的长名不一致；没有「短名只作历史、出现即拒绝」的条款。
- **违反依据**：同一契约事实只能有一个权威字面（`design-doc-writing-style`／`decisions-guide`）；ADR-0031 要求信任根路径与摘要由批准记录固定；`review-sec-r04` 的 `N011` 信任根固定要求。
- **触发场景**：实现者按 B1 的短 PATH 名读取信任根，而 `readAction` 只从注册长名取值 → 信任根未被读取／校验（通常 fail-closed）；或实现者同时接受两套键名 → 接受未注册别名，信任根路径／摘要可由非预期键固定；SHA256 半边按优先级取长名，但 B1 仍写短名，易造成两套实现。
- **影响**：信任根路径／摘要可能未按批准记录固定，破坏「验签公钥只来自 `receipt_anchors.public_key`、信任根摘要固定」的准入前提。不是直接的准入绕过（预检通常 fail-closed），但属安全相关配置解析歧义，不能给 PASS。
- **修复建议（交作者／Lead 决定）**：把 `:4070`、`:4088`、`:4195`、`:4373` 与 ADR `:1118` 统一为同一组键名（建议采用已在注册清单和 ADR 中使用的长名 `..._APPROVAL_TRUST_ROOT_PATH`／`..._APPROVAL_TRUST_ROOT_SHA256`）；若保留短名，必须显式声明「短名只作历史、出现即 `ErrAdmissionBinding`」，并加入注册清单与可机核判据（例如 `rg 'HAN_RECOVERY_RESTORE_TRUST_ROOT_' design.md` 只允许命中历史／拒绝句）；同时同步 `:4373` 的 `trust_root_generation` 键名。ADR 属他人所有，本报告只记录，不代改。
- **证据**：上述行原文与计数；`framework.md:1118` 长名引用。
- **复审条件**：冻结后重新统计四个键名的命中；注册清单、B1、`generation` 定义与 ADR 使用同一字面；存在可机核的别名拒绝条款。

### 3.2 `NEW-S05-06`（LOW）`:4079` 的 V 编号范围措辞仍与 V1–V11 检查点集合冲突

- **位置**：`:4079`（r66.7）vs `:4037`（r64.5）、`:4062`（r65.5 + r67.5）、`:3990`（r67.2 改后）。
- **事实**：
  - `:4079`：「唯一有效的 V 编号是 `:4062` 的 V7–V11（载荷与签名主体按 r66.2 绑定）」。
  - `:4037`：「`V1 prepared(ctx)` … `V11` 返回带 `audit_record_ref` 的结果副本」，明确列 V1–V11。
  - `:4062`：「`R1`–`R8` 与 `V1`–`V11` 只枚举**契约检查点**，不得读成穷举代码中的所有检查」。
  - `:3990`（r67.2 改后）：「现行 V1–V11、8 字段验证方签名载荷与 head 形状只以 r65.5／r65.2 及 r66.2 为准」。
  - r67.2 只把「V7–V11 的载荷与签名主体」指向 `:4062` 与 `:4074`，没有限定 `:4079` 的「唯一有效的 V 编号」范围。
- **影响**：实现者可能把 `:4079` 读成「V1–V6 不再有效」，删掉或跳过 `prepared(ctx)`、`revalidateApproval`、`VerifyRestoreSource` 等前置检查点；不改变 V7–V11 的载荷唯一性，但造成契约检查点集合歧义。LOW。
- **修复建议**：把 `:4079` 改为「唯一有效的**载荷绑定** V 编号是 `:4062` 的 V7–V11；`V1`–`V6` 仍为前置检查点，完整集合见 `:4037`／`:4062`」，与 `:3990` 的「现行 V1–V11」对齐。
- **证据**：上述行原文；12 行 diff 显示 `:4079` 未被 r67 修改。
- **复审条件**：`:4079` 出现「载荷绑定」限定，或明确写出 V1–V6 仍有效；`rg` 对 V1–V11 的表述一致。

### 3.3 `NEW-S05-07`（LOW）pin 轮换窗口语义仍不闭合

- **位置**：`:4077`（r66.5「pin 轮换」句）。
- **事实**：同句写「任一时刻只有一个 active pin；轮换必须由人的明确批准并留审计记录，不做双 pin 同时接受；旧 pin 只作为部署回滚窗口保留，窗口外失败即 closed（`ErrAdmissionUnsupported`／`ErrAdmissionBinding`）」。
- **缺失**：窗口长度；判定时点（何时停止接受旧 pin）；窗口内旧 pin 是否可被请求路径接受（与「不做双 pin 同时接受」的关系）；轮换批准与审计记录必须绑定哪些字段（旧／新 SPKI 摘要、生效时间、批准人／审计引用）。
- **影响**：轮换期内可能出现「旧 pin 是否仍接受」的实现分歧；若窗口内双接受，则与同句前半冲突；若窗口内不接受，「回滚窗口」没有请求路径语义。属 LOW 契约歧义，不是当前 closed 状态的越权路径。
- **修复建议**：明确「任一时点只有一个 active pin；旧 pin 只保留为**离线回滚材料**，任何请求路径都不接受旧 pin；窗口长度与生效时点由批准记录固定；轮换批准须含旧／新 SPKI 摘要与生效时间并留审计」。
- **证据**：`:4077` 原文；12 行 diff 显示该行未被 r67 修改。
- **复审条件**：`:4077` 出现窗口长度、判定时点、旧 pin 请求路径处置、批准／审计字段绑定的可机核句。

### 3.4 `NEW-S07-01`（LOW，新发现）B1 重验点枚举与错误分类未随 r67.5 同步

- **位置**：`:4070`（r65.13 B1）vs `:4037`（r64.5 R6／V6 漂移分类）、`:4062`（r65.5 + r67.5）、`:4135` 其六、`:4137`。
- **事实**：
  - `:4070` B1：「启动预检构造不可变快照，R2／R4／R6、V2／V4／V6 与 `prepared(ctx)` 重读重验，任何漂移 → `ErrAdmissionClosed`。」
  - r67.5 在 `:4062` 新增 V10 重验：「在外部见证**原子推进** `sequence` 之前再调用一次 `revalidateApproval`……任一失败 → `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`，由调用方按 `restore_unknown` 处置，不得推进 head」；`:4135` 其六、`:4137` 尾句同义。
  - `:4037` 另写「`R6`／`V6` 之后的漂移必须 `restore_unknown` 或 closed，绝不允许当成成功」。
  - B1 的重验枚举未列 V10（也未列 P6）；其「任何漂移 → `ErrAdmissionClosed`」是预检／启动前分类，不能直接套到 V10 的 post-start `restore_unknown`。
- **影响**：实现者若把 B1 当作完整清单，可能漏掉 V10 推进前重验，或对 V10 失败返回错误的 `ErrAdmissionClosed`。规范句 `:4062`／`:4135`／`:4137` 已给出 V10 与 `restore_unknown`，且 r67 > r65 的优先级可解析行为，因此这是 LOW 精度／完整性缺陷，不是新的准入绕过；但它会削弱「B1 作为批准记录读取路径摘要」的可机核性。
- **修复建议**：把 V10 加入 B1 的重验枚举，并写明 post-start 失败按 `restore_unknown`（`errors.Join(recovery.ErrRecoveryWriteUnknown, err)`）；同时补 P6，或明确「B1 只是摘要，完整集合以 `:4037`／`:4062`／`:4135`／`:4137` 为准」。不要用 B1 的 `ErrAdmissionClosed` 覆盖 `:4037` 的 R6/V6「`restore_unknown` 或 closed」。
- **证据**：上述行原文；12 行 diff 显示 `:4070`、`:4037` 未被 r67 修改，`:4062`／`:4135`／`:4137` 被 r67.5 修改。
- **复审条件**：B1 与 `:4037`／`:4062`／`:4135`／`:4137` 的重验点集合、错误分类一致，或 B1 明确声明非穷举。

## 4 并行 findings 的闭合判定

### 4.1 `NEW-S06-01`（MEDIUM，作者证据链）——已处置，保留 r65 重建限制

- **r66→r67 一手配平（本人复算）**：`design.r66.baseline.md` 与 r67 冻结快照逐行比较，**仅 10 个语义行不同**（另 `:5`／`:38` 版本串等字节）：`:3942 +130`、`:3990 +130`、`:4062 +344`、`:4072 +1,125`、`:4073 +186`、`:4076 +1,992`、`:4135 +284`、`:4137 +251`、`:4369 +91`、`:4373 +790`，合计 **+5,323 B**，与 `1,084,882 − 1,079,559` 相等；行数均 5,476，CR 169／LF 5,476 不变。
- **r65→r66 重建（作者口径，本人复核）**：`%TEMP%\design-r65-prefix.bak.md` 实测 **1,044,640 B / 5,464 行 / `2BBBDC8410BEE8681DA3F3B648693EADD6CDA1CEC8856B90222F943EDB72DE61`**；备份 + R05 13 行 **+471 B** = **1,045,111 B**，与 r65 派发值相等。16 行 r66 表逐行实测合计 **+3,307 B**；r66 插入块 `:4072`–`:4083`（含末行 LF）**31,141 B / `587471F58DB925A17D4E3F06AFE981244E9D2120B969DB615AC5DE2BED749048`**；`3,307 + 31,141 = 34,448 = 1,079,559 − 1,045,111`。备份与 r66 基线按插入点 +12 对齐后，内容级差异 **23 行**（`:5`、`:38`、16 行 r66 行内、R05 的 `:4017`／`:4057`／`:4058`／`:4068`／`:4070`；其中 3 行在备份侧标签为 `bak:4078`／`:4143`／`:4175`，对应 `:4090`／`:4155`／`:4187`），与作者 §3.6 一致。
- **判定**：`NEW-S06-01` 的证据链自洽性与可机核性已恢复；1,555 B 残余按整行口径被吸收。**限制**：原始 r65 文件（1,045,111 B）在本机不可得，r65 列是「备份 + R05 13 行」重建值，不是一手实测；若日后取得原始 r65，应以一手值替换重建列。该限制不改变 r66→r67 的一手结论。
- **附注**：`NEW-S05-03`（作者报告终值／行号不一致）在本轮未复现：r07 作者报告与 `hashes.txt` 的终值一致，r66.0 已把旧 `1,045,042 B / D2A6…` 与 `:4247` 标签作废；因属作者证据而非设计语义，不计入闭合计数。

### 4.2 `NEW-S06-02`（LOW，V10 推进前重验）——设计层已闭合

- **设计层闭合**：`:4062` V10 明确「验证方签名后、外部见证原子推进 `sequence` 之前再调用一次 `revalidateApproval`（重读批准文件、信任根与 `receipt_anchors`，重算 `generation` 三元摘要并核对有效期与撤销；任一失败 → `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`，由调用方按 `restore_unknown` 处置，不得推进 head）」；`:4135` 其六与 `:4137` 尾句给出同一判据。该点已在检查点清单中可机核。
- **跨文件残余（不计入本对象计数）**：ADR r10 `framework.md:1118` 还要求 `V7` 独立读取回执前、`V8` 验 producer 签名前各一次重验，并把检查点集合扩到 `P3`／`P6`／`R1`／`V1`／`R2`／`R4`／`R6`／`V2`／`V4`／`V6`／`V7`／`V8`／`V10`；design r67 只补 V10，未镜像 V7／V8。作者报告 §4 已把 V7／V8 标为 **OPEN**。ADR 正被并行改写、其接受不在本对象内，本报告只记录，不代裁；若 ADR r10 冻结为现行契约，design 需另立修订对账（属方向决策，不是本报告代改项）。
- **时点解读项**：ADR `:1118` 的措辞「`V10` 追加审计／验证方签名／推进见证 head 前各一次」可被读成「V10 开始前」，而 design r67.5 把重验放在「追加审计、`fsync`、验证方签名之后，推进 head 之前」。若 ADR 冻结后采用「V10 开始前」的强读法，design 需对账重验时点与审计记录写入顺序；本轮只记录这一解读差异，不据此给 design 记 finding。

### 4.3 `NEW-S06-03`（LOW，`generation` 定义）——设计层已闭合

- **设计层闭合**：`:4369` 明确「`generation` 不是本行新增字段，按 r67.6 的三元摘要定义」；`:4373` 给出三元定义：
  - `approval_generation` = 批准文件精确字节的 SHA-256（预检不可变快照记录，`revalidateApproval` 重读时逐字节比对）；
  - `trust_root_generation` = 信任根文件精确字节的 SHA-256，必须等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`；
  - `anchor_generation` = `receipt_anchors` 中该 `anchor_ref` 条目 JCS 规范字节的 `sha256:` 摘要，必须等于 policy 与回执的 `trust_anchor_sha256`；
  - 三项逐字节相等才算同一 `generation`，不得以「文件可读」「inode 未变」「长度未变」替代摘要比较。
- **跨文件一致性**：与 ADR `framework.md:1118` 的 (i)(ii)(iii) 逐项一致。**注意**：`trust_root_generation` 使用长名 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`，与 `:4070` B1 的短名冲突同源，反向加重 `NEW-S05-05`；这不是 `NEW-S06-03` 未闭合，而是 `NEW-S05-05` 的跨处证据。
- **判定**：`NEW-S06-03` 在设计层闭合；ADR 侧接受由架构侧独立给出，本报告不代判。

## 5 无回退确认（13 项）

**总判据**：r66→r67 仅 12 行变化，除 `:4062`（N015 锚点）与 `:4072`（优先级锚点）外，下列各项的锚点均不在这 12 行内，故与 r66 基线逐字节相同；对 `:4062`／`:4072` 的改动已逐句核对，未删除或弱化原闭合句。

| 项 | 锚点 | r67 证据（未回退） | 判定 |
|---|---|---|---|
| `SEC-F002` | `:3936(3)`、`:3937`、`:4061` | `:3936(3)` 已删 principal 侧 `role_name` 比较；`:4061` 六值封闭枚举（`restore_approver`／`restore_reviewer`／`restore_executor`／`restore_producer`／`restore_verifier`／`restore_witness`）、裸名显式映射、`person_ref` 全局唯一、六席两两不等；golden vector 的 5 个 `role_name` 全部在枚举内 | 无回退 |
| `SEC-F003` | `:3988`、`:4059`、`:4074`、`:4062`、`:3990` | `:3988` producer envelope 唯一；`:4059` 8 字段验证方载荷只服务 V10；`:4074` V7–V11 逐点绑定；`:4062` V8 仍引用 `:3988`；`:3990` 历史行仍只指向 r65.5／r65.2／r66.2，r67.2 只加确认指针 | 无回退 |
| `N001` | `:3988`、`:4059`、`:4062`、`:4074` | 与 `SEC-F003` 同根因；V8 producer 签名、V10 验证方签名、`anchor_ref` 五处等式链未变；`:4062` 仍写五处逐字节相等 | 无回退 |
| `SEC-F006` | `:4057`、`:4063`、`:4064`、`:4070(4)(5)` | 三项放宽（`NoNewPrivs`／`CapBnd`／`PATH`）登记与 1 项容量变更仍在；`CapInh/CapPrm/CapEff/CapAmb=0`、`Uid/Gid` 四元组、`Groups` 空、`AT_SECURE=0` 仍为拒绝条件；`CapBnd`／`NoNewPrivs` 仅观测 | 无回退 |
| `N011` | `:4077`、`:4060` | `witness_endpoint` 单次解析→固定 IP 集合→`DialContext` 只连集合→连接后 `RemoteAddr.Unmap()`∈集合；`witness_host` DNS 名约束、非规范 IPv4 拒绝、`ServerName` 绑定、pin 唯一路径、独立性链均保留 | 无回退 |
| `NEW-S05-01` | `:4078` | `environment_dirs` 绝对规范路径、`st_uid==0`、非 group／other／executor 可写、逐级 `lstat` 无符号链接、挂载 `ro,nodev,nosuid`；loader 父目录不得在目录集内且不得可写；禁 `exec.LookPath`／裸名；`PYTHONPATH`／`NODE_PATH`／`RUBYLIB`／`PERL5LIB` 拒绝；≤32 目录、`PATH` ≤4096 B 且逐字节等于 `:` 连接 | 无回退 |
| `NEW-S05-02` | `:4072`、`:4078`、`:4435` | `:4072` 优先级仍为 `r67 > r66 > r65 > r64 > r63 > r62 > r61`，同一修订小节号大者胜；`:4078` 仍明示 `:4435` 的「禁止 `PATH`」由 r66.6 取代；`:4435` 字节未变 | 无回退 |
| `NEW-S05-04` | `:4077`、`:4060`、`:4061` 尾 | `receipt_anchors` 15 项含 `witness_host`；`witness_endpoint` 2048 B 上限；`witness_key_id → keys.principal_id → principals.org_id` 链；pin 轮换句仍在；`root`／`supplier` 的 `appointment_ref` 只允许空串且不占六席 | 无回退（pin 轮换残余另见 `NEW-S05-07`） |
| `N012` | `:1718`、`:4065` | `recovery_restore_bindings.receipt_version` CHECK = `han-restore-receipt/v1`；`RestorePlan.Version` 仍为 `han-db-restore/v1`；第三别名 `restoreReceiptWire` 改造后 `rg` 必须为 0 | 无回退 |
| `N013` | `:4066`、`:4141` | `Target.DSN` **删除**（非改类型），DSN 由 `appendJSONString` 直写；`restorePlanWire`／`restoreReceiptWire` 仍无 DSN 字段且字段类型为 `string` | 无回退 |
| `N014` | `:4067` | `static` 为默认与首选；`dynamic` 仅在无 `LD_*`、`PT_INTERP` 来源、闭包摘要逐项相等、`DT_NEEDED` 并入、父目录与挂载规则覆盖等合取成立时允许；无法证明 → `ErrAdmissionUnsupported` | 无回退 |
| `N015` | `:4062` | `cmd/han-server/recovery_restore.go:352` 在 `tool.Run` 之后调用 `d.prepared(ctx)` 仍归入 R6 第二次同构复核；`execute` 之后必须保留该调用，不得删除；R1–R8 不是穷举 | 无回退 |
| `N016` | `:4012`、`:4060` | 基线字段统一 `baseline_sequence`；见证推进 head `sequence`；旧 `sequence_floor` 只作历史 | 无回退 |

## 6 Golden vector 独立复算

方法：用 `[IO.File]::ReadAllLines(..., UTF8)` 取整行，取该行 UTF-8 字节（不含行尾 LF），直接 SHA-256；不抄报告摘要，不经过 JCS 库。两条向量均为纯 ASCII（非 ASCII 字符数 0）。

| 位置 | 内容字节 | 复算 SHA256 | 设计期望 | 结果 |
|---|---:|---|---|---|
| `:4099` source | **1,115** | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 同左 | 一致 |
| `:4105` tool | **2,357** | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 同左 | 一致 |

角色字面：`:4099` 的 `role_name` 为 `restore_approver`、`restore_reviewer`；`:4105` 为 `restore_executor`、`restore_approver`、`restore_reviewer`；合计 5 个值，全部落在 `:4061` 的六值封闭枚举内。`restore_reviewer`／`restore_approver`／`restore_executor` 字面与六值枚举一致，没有裸名或第七值。

## 7 NOT RUN 与范围外观察

### 7.1 `SEC-F007`（NOT RUN，不得改判 PASS）

- `:4070(6)` 原文：「Linux host 层全部 NOT RUN：host／ACL／mount／loader／fd-exec／maps／network／witness 实测均未运行；Windows 只覆盖解析与语义层，`//go:build linux` 的 skip 与无匹配测试的 exit 0 都不等于 PASS。」
- 本轮未在 Linux 主机实跑；缺少 POSIX ACL／`st_uid/st_gid/st_mode`、`ro,nodev,nosuid` mount、loader（`pt_interp_path`／`ld_so_cache_path`／`ld_so_conf_paths`）、fd-exec（`O_CLOEXEC`／`O_NOFOLLOW`／`execve`）、`/proc/self/maps` 与 `AT_SECURE`、DNS／`DialContext`／`RemoteAddr`／TLS pin、外部 witness 等运行条件。
- 其他 NOT RUN／OPEN：`NoNewPrivs`／`CapBnd` 仅为事后观测值；`dynamic` 可证明形态、mount 父目录覆盖、`environment_dirs` 逐级 `lstat` 未实测；`bundle_digest` 仍为显式 OPEN（首期不实现）；Go `test`／`vet`／`build`／E2E 未运行。
- **本报告不声称**：生产 restore 解锁；`enter`／`resume` 解锁；ADR-0031 约束 8 满足；`N005` 已解决；`identity-auth-S4.24a` 可启动、可关闭或已交付。`SEC-F007` 仍是 **NOT RUN**，不得改判 PASS。

### 7.2 `N005`（范围外 BLOCKER，只登记不代改）

- `framework.md:1084`：「本修订 `adr-0031-r10` 的两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动」。
- `framework.md:1129`：「现行修订须同时取得架构与安全两路独立评审接受，才解除实施/验证卡的派发、认领与开始改动限制」；历史接受只绑定各自修订号。
- `tasks.md:3197` 仍称「`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受……决策接受级对 r07 成立，故本卡启动门禁解除」；`tasks.md:3186` 仍为 `- [ ]`。
- 三处对「现行修订是哪一版、门禁是否解除」给出相反事实；该冲突在 design 修订范围之外，本轮不改 ADR、不改 `tasks.md`，但 `N005` **仍是范围外 BLOCKER，阻断 `identity-auth-S4.24a` 启动**；须由架构决策方与 Task-Planner 对账。该 1 项**不计入**本报告主体计数。

### 7.3 ADR 并行改写观察（只记不改）

- `framework.md` 本轮开始与结束均为 **285,639 B / 1,273 行 / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`**；`:1082` 为 `adr-0031-r10`。ADR 侧接受不在本对象内，本报告不代裁。
- `framework.md` 仍有 3 处指向 `identity-auth-design-ir01-r66`（`:1118` 两处、`:1192` 一处），r67 升级后成为陈旧指针；作者报告 §4 已标 **OPEN（跨文件陈旧指针）**，ADR 属他人所有，本轮未改。
- ADR r10 的 V7／V8 追加重验与完整检查点集合未被 design r67 镜像（作者报告 §4 标 OPEN）；若 ADR r10 冻结为现行契约，需另立 design 修订对账。该事项属跨文件契约方向决策，不在本报告计数主体内。

## 8 复审条件

1. `design.md` 冻结：给出只读窗口与冻结前后 B／LF／CR／SHA256；窗口内再次变化则重新冻结、重新判读。
2. `NEW-S05-05`：统一 `:4070`／`:4088`／`:4195`／`:4373` 与 ADR `:1118` 的信任根键名；加入可机核的别名拒绝条款；确保 `recoveryRootConfigurationKeys`、B1 与 `generation` 定义使用同一字面。
3. `NEW-S05-06`：把 `:4079` 限定为「载荷绑定 V 编号 V7–V11」，并明确 V1–V6 仍为前置检查点，与 `:4037`／`:4062`／`:3990` 对齐。
4. `NEW-S05-07`：补齐 pin 轮换窗口长度、判定时点、旧 pin 请求路径处置、批准／审计字段绑定与失败分类。
5. `NEW-S07-01`：把 V10（及 P6）加入 B1 重验枚举并写明 `restore_unknown`，或明确 B1 非穷举、完整集合以 `:4037`／`:4062`／`:4135`／`:4137` 为准。
6. ADR r10 冻结后，由架构侧对账 V7／V8 追加重验、完整检查点集合与 r66→r67 陈旧指针；`N005` 由架构决策方与 Task-Planner 对账；本报告不代改。
7. 复跑：两条 golden vector、r66→r67 字节账、13 项无回退锚点、`SEC-F007` 的真实 Linux 条件；未取得 Linux 实跑前，`SEC-F007` 仍为 NOT RUN。

## 9 复算命令（只读）

```powershell
# 对象与冻结快照
Get-FileHash -Algorithm SHA256 -LiteralPath '.add/project/knowledge/design-workspace/identity-auth/design.md'
Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $env:TEMP 'design-r67-freeze-B12F1EF3.md')

# r66→r67 逐行 diff 与字节差
$r66='.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r07/design.r66.baseline.md'
$r67='.add/project/knowledge/design-workspace/identity-auth/design.md'
$a=[IO.File]::ReadAllLines($r66,[Text.Encoding]::UTF8); $b=[IO.File]::ReadAllLines($r67,[Text.Encoding]::UTF8)
for($i=0;$i -lt $a.Count;$i++){ if($a[$i] -cne $b[$i]){ '{0}: {1} -> {2}' -f ($i+1),[Text.Encoding]::UTF8.GetByteCount($a[$i]),[Text.Encoding]::UTF8.GetByteCount($b[$i]) } }

# golden vector（内容字节，不含 LF）
$t=[IO.File]::ReadAllLines($r67,[Text.Encoding]::UTF8)
foreach($n in @(4099,4105)){ $bytes=[Text.Encoding]::UTF8.GetBytes($t[$n-1]); '{0}: {1} B / {2}' -f $n,$bytes.Length,([BitConverter]::ToString([Security.Cryptography.SHA256]::Create().ComputeHash($bytes)).Replace('-','').ToLower()) }

# 配置键计数
rg -c 'HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH' $r67
rg -c 'HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH' $r67
rg -c 'HAN_RECOVERY_RESTORE_TRUST_ROOT_SHA256' $r67
rg -c 'HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256' $r67
```

## 10 声明

本报告是安全侧独立复审意见，**不构成**人的成果接收、独立评审通过、安全签收或生产装配批准；**不构成**生产 restore 解锁、`enter`／`resume` 解锁、ADR-0031 约束 8 满足、`N005` 已解决、`identity-auth-S4.24a` 可启动／可关闭／已交付。`SEC-F007` 仍为 **NOT RUN**，不得改判 PASS。

本端未修改 `design.md`、`tasks.md`、ADR（`framework.md`）、`cmd/**`、`internal/**`、测试、既有 `review-*/`、`design-revision-*/`；未执行任何 Git 写操作；未删除、移动或归档任何文件。写集仅为本报告与同目录 `hashes.txt`。
