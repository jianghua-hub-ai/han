# SEC r05 独立复审报告（Han design r65）

**日期**：2026-11-02

**评审角色**：安全侧 Reviewer（独立复审，非作者、非 Lead）。

**被评审对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`，现行实测 **1,045,111 B / 5,464 行（ReadAllLines）/ SHA256 `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9`**，版本 `identity-auth-design-ir01-r65`，`last_updated: 2026-11-02`。r65 区块 `:4057`–`:4070`；r61 段标题现行 `:4278`。

**基准报告**：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r04/report.md`，40,794 B / 292 行 / SHA256 `F98B39D19083E7A010680465C40404AEEE23E95C65F48C103602F7758632E898`，Verdict `CHANGES REQUIRED`。本报告以该报告的开放 finding 为判据，不以其结论代替本轮判断。

**作者修订证据（只作被核对材料）**：`design-revision-r05/report.md` 42,696 B / 239 行 / SHA256 `56FAD3F134487E3810286D4C3A183D2C29F2854355757250474D772CA2CF34EB`；`design-revision-r05/hashes.txt` 3,805 B / SHA256 `E5CDBBCFC6D1F7F9C921787F1F7CA607C27896E9B7BB6AB58ED02B7668655B05`。

**方法**：按 `add-reviewer` Role 全文、`add-code-review`（objectType = design）主方法、`code-review-checklist`、`abstraction-principles`、`decisions-guide`、`design-doc-writing-style` 执行；代码构建、单测、编译类条款按对象裁掉，只保留与设计契约可核性相关的部分。结论只覆盖 restore／ADR-0031 安全契约相关范围，未逐行通读全部业务章节。

**两轮实际检查范围**：

- 第一轮（对象层）：完整读取 r65 区块 `:4057`–`:4070`；读取 `:3933`–`:3942`（keyset／交叉约束）、`:3987`–`:3990`（回执 envelope／旧 VerifyRestored）、`:4012`、`:4031`–`:4039`、`:4049`、`:4053`、`:4055`、`:4203`、`:4247`、`:4278`、`:4423`；对 `:4087`／`:4093` 两条 golden vector 按"精确行字节、不含行尾 LF"独立复算；读取 `cmd/han-server/recovery_restore.go` `:36`–`:80`、`:247`–`:303`、`:343`–`:363`、`:366`–`:382`、`:392`；执行 `rg` 计数（`restoreReceiptWire`、`Target.DSN`、`appendJSONString`）。
- 第二轮（逐 finding 复核）：完整读取 r04 报告 292 行；逐条回到其原 finding、r65 声称落点与可机核判据；核对 ADR-0031 `:1082`／`:1084`／`:1111`–`:1180`（含 `:1117`／`:1144`／`:1147`／`:1164`／`:1166`）、`decisions/README.md:39`、`tasks.md:3186`／`:3197`；复算 design／tasks／framework／README／r04 报告／作者报告全部哈希；对冻结向量核对 `role_name` 字面与六值枚举的一致性。

**只读声明**：本轮只读核对，**未修改任何制品、无 Git 写操作**；唯一写入为本报告与同目录 `hashes.txt`。未删除、未移动、未归档任何文件；未触碰 `.spec/wip/article-management/` 下其他写入者内容。

**Verdict: CHANGES REQUIRED**

| 类别 | 计数 | 条目 |
|---|---:|---|
| BLOCKER（范围内） | 2 | SEC-F003、N001（同一根因，同一处修复） |
| HIGH | 2 | N011、NEW-S05-01 |
| MEDIUM | 1 | NEW-S05-02 |
| LOW | 2 | NEW-S05-03、NEW-S05-04 |
| 已闭合（设计层） | 7 | SEC-F002、SEC-F006、N012、N013、N014、N015、N016 |
| NOT RUN | 1 | SEC-F007（不得改判 PASS） |
| 范围外 BLOCKER | 1 | N005（不计入上表，但阻断启动） |

阻断项计数：**范围内 BLOCKER 2 项**；开放项（含 HIGH／MEDIUM／LOW）合计 **7 项**；另有 1 项 NOT RUN 与 1 项范围外 BLOCKER。任何 NOT RUN 仍是 NOT RUN。

## 1 复算、基线与证据不一致

### 1.1 对象与上游实测

| 对象 | 字节 | 行 | SHA256 |
|---|---:|---:|---|
| design.md（r65） | 1,045,111 | 5,464 | `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9` |
| tasks.md | 898,536 | 3,289 | `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` |
| framework.md（adr-0031-r09） | 272,588 | 1,271 | `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4` |
| decisions/README.md | 21,613 | 110 | `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2` |
| review-sec-r04/report.md | 40,794 | 292 | `F98B39D19083E7A010680465C40404AEEE23E95C65F48C103602F7758632E898` |

`git rev-parse HEAD` = `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，当前分支 `main`；`git status --porcelain` 显示 design/tasks/framework/README 等均为工作树未提交修改（与作者报告所述一致）。工作树另有与本对象无关的未跟踪文件（含 `$null`、`$prof`、`'){` 及 `.spec/wip/article-management/` 下内容），本轮一律未触碰。

### 1.2 golden vector 独立复算

方法：用 `[IO.File]::ReadAllLines(..., UTF8)` 取整行，再取该行 UTF-8 字节（不含行尾 LF），直接 SHA-256；不抄报告摘要，不经过 JCS 库。

| 位置 | 字节 | 复算 SHA256 | 设计期望 | 结果 |
|---|---:|---|---|---|
| `:4087` source | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 同左 | 一致 |
| `:4093` tool | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 同左 | 一致 |

两条向量均为纯 ASCII（非 ASCII 字符数 0），键序检查未发现逆序。角色字面核对：source 的 `issuer=restore_approver`、`reviewer=restore_reviewer`；tool 的 `executor=restore_executor`、`issuer=restore_approver`、`reviewer=restore_reviewer`。三者全部落在 r65.4 的六值封闭枚举内，向量与角色枚举**不互斥**。tool 的 `environment=["LANG=C.UTF-8","LC_ALL=C.UTF-8","TZ=UTC"]`，不含 `PATH`；tool 记录无独立"批准目录"字段。

### 1.3 作者报告与现行对象不一致（必须记录）

- `design-revision-r05/report.md` 第一节"输出（design）"行与 `hashes.txt` 的 `design.md` 行记录 **1,045,042 B / SHA256 `D2A6B54B2153858DC6366995C4332A380D9DA0BEDE863BDDC80D51663A3D8BE2`**；现行实测为 **1,045,111 B / `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9`**，差 **+69 B**。作者报告"净值"行自己提到最后一次对 r65.0 ADR 表述的事实校正 `+69 B`，但未把终值回写到表格与哈希清单。因此该报告与 hashes.txt **不能绑定本轮被评审对象**，复审只能以本报告复算值为准。
- 作者报告称"r61 段（现行 `:4247`–文件末）"；现行 r61 段标题在 **`:4278`**。`:4247` 实际是 r62.9.1 文件表的表头行（r62 尾段），不是 r61 段起点。r65.6 的原文"r61 段及其前置的 r62 尾段（现行 `:4247` 至文件末）"是准确的；作者报告把它简写成"r61 段 `:4247`"造成误导。
- 冻结向量行号从 r04 的 `:4072`／`:4078` 移到现行 `:4087`／`:4093`，与 r65.0 的行号换算规则（≥`:4057` 的历史行号 +15）一致；向量字节未变。

## 2 逐条判定总表

| ID | r04 级别 | 本轮判定 | 一句话依据 |
|---|---|---|---|
| SEC-F002 复核人独立性／一人两名 | BLOCKER | **已闭合（设计层）** | `:3936(3)` 已删 principal 侧 `role_name` 比较；`:4061` 六值封闭枚举＋显式映射＋`person_ref` 全局唯一＋六席两两不等；`:3937` 要求机核、不依赖调用者自报；向量角色在枚举内 |
| SEC-F003 撤销 TOCTOU／调用图 | BLOCKER | **仍开放（BLOCKER）** | V 编号已唯一，但 `:4062` 的 `V8` 把 **producer 签名**指向 r65.2 的 **8 字段验证方载荷**；该载荷含 `verified_at`／`verifier_*`，V8 时尚不存在，producer 载荷实际在 `:3988` |
| N001 见证协议与权威推进 | BLOCKER | **仍开放（BLOCKER）** | 与 SEC-F003 同根因（N001(a) 载荷半边）；见证端点/独立性另有 N011 |
| N011 `witness_endpoint` 独立性 | HIGH | **部分闭合，仍开放（HIGH）** | 六项约束已加，但 DNS rebinding TOCTOU、host 精确匹配无独立登记字段、IPv4-mapped IPv6 未覆盖、pin 与"禁止 InsecureSkipVerify"组合无实现路径、pin 轮换未定义、独立性清单漏 verifier |
| SEC-F006 能力断言／放宽登记 | HIGH | **已闭合（登记层）** | `:4057` 逐项登记 `NoNewPrivs`／`CapBnd`／`PATH` 三项（内容/理由/影响/抵消/残余）＋信任根上限 1 项容量变更；`:4064` 复述拒绝条件；`:4070(4)(5)` 复述残余。PATH 抵消措施不足另列 NEW-S05-01 |
| N012 回执版本与第三别名 | LOW | **已闭合（设计层）** | `:1718` CHECK 改为 `han-restore-receipt/v1` 且加 r65.8 标记；`:4065` 点名 `restoreReceiptWire`；当前代码 3 处命中已声明为未实施 |
| N013 DSN 字段／类型 | LOW | **已闭合（设计层）** | `:4066` 明确删除 `Target.DSN`（非改类型）；`:4129` 加消解句；代码 `:36`–`:50`／`:52`–`:65` 无 DSN 字段，`:78` 为 `Target.DSN`（未实施） |
| N014 `static`／`dynamic` 措辞 | LOW | **已闭合** | `:4067` 改为 `static` 默认与首选＋`dynamic` 四合取；`:4039` 已就地改写并加 r65.10 标记 |
| N015 R 图与代码节点 | LOW | **已闭合（设计层）** | `:4062` 把代码 `:352 d.prepared(ctx)` 归入 R6 第二次同构复核并要求保留；代码 `:347 tool.Run`→`:352 prepared`，`:346`／`:351` defer clear，`:392 d.dsn=""` |
| N016 旧字段名残留 | LOW | **已闭合** | `:4012` 改为 `baseline_sequence` 并加 r65.3 标记；其余 `sequence_floor` 均在历史/取代说明中 |
| SEC-F007 真实环境验证 | NOT RUN | **NOT RUN** | `:4070(6)` 仍写 Linux host 层全部未跑；不得改判 PASS |
| N005 ADR／tasks 状态冲突 | 范围外 BLOCKER | **仍存在（不计入计数）** | `framework.md:1084`／`:1129` 称 r09 未复评、S4.24a 不得启动；`tasks.md:3197` 称 r07 已接受、门禁解除；`:3186` 仍 `- [ ]` |
| NEW-S05-01 `PATH` 放宽限定不足 | 新增 HIGH | **开放** | 仅"绝对规范路径＋列入批准记录"，无属主/可写性/loader 父目录覆盖/禁 LookPath；批准记录无目录载体 |
| NEW-S05-02 `:4423` 与 r65.6 字面冲突 | 新增 MEDIUM | **开放** | r61 原字节 `:4423` 仍写"禁止 `PATH`"，r65.6 允许 `PATH`；取代链只在 `:4039`／`:4063` 元声明，无冲突位置的可判定优先级 |
| NEW-S05-03 作者报告终值/行号不一致 | 新增 LOW | **开放** | 1,045,042/`D2A6…` vs 1,045,111/`87D70…`；r61 起点写作 `:4247`（实际 `:4278`） |
| NEW-S05-04 精度残余 | 新增 LOW | **开放** | r65.3 未保留 r64.3 的 2048 B 上限；host 精确匹配无独立字段；`witness_key_id` 无 `org_id` 字段；pin 轮换未定义；root/supplier 任命值域未定义 |

## 3 仍开放 BLOCKER（SEC-F003／N001，同一根因）

### 3.1 事实与位置

- `:4062`（r65.5）现行规范句：`V8` 验证 **producer 签名**（**8 字段载荷见 r65.2**）。
- `:4059`（r65.2）：该 8 字段是**验证方**签名载荷，exact keyset 为 `anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id`；并明确 `verified_at` **只在 V10 检查点一次性取样**。
- `:4031`（r64.2）：同一 8 字段验证方载荷；`verified_at` 是 V10 检查点取样。
- `:3988`（r63.8）：**producer 签名载荷**的唯一现行定义是 `JCS({"receipt": R, "attestation": A})`，其中 `R` 为 12 字段回执删除 `signature` 与 `audit_record_ref` 后的 **10 字段**对象，`A` 为 attestation 删除 `signature` 后的 **6 字段**对象；`audit_record_ref` 不进入签名载荷。
- `:3990`（历史映射）：旧步骤二＝`V8` producer 签名，并写"按上段（r65.2 的 8 字段载荷）重算签名载荷并验证 Ed25519"；同一错误指向在历史行里重复出现。
- 规范化定义中，8 项 keyset 只见于 `:4031`（r64.2，原文作「8 项」）与 `:4059`（r65.2），两者均为**验证方**载荷；`:4062` 把它错指给 V8，`:3990` 历史行重复同一错指；全文不存在「producer 8 字段载荷」的定义。

### 3.2 违反依据与触发场景

违反依据：V8 的语义是"验证 producer 签名"，其签名对象必须来自 `:3988` 的 producer envelope；把验证方载荷指给 V8，等于让验证方用含 `verifier_principal_id`／`verifier_key_id`／`verified_at`／`prev_record_sha256` 的对象去验 producer 的签名。V8 在 V10 之前执行，`verified_at` 尚未取样，`prev_record_sha256` 也尚未产生（审计记录尚未追加）。

触发场景（可实现性推理，非运行复现）：

1. 实现者严格按 `:4062` 实现 V8 → 用验证方 8 字段对象重算 producer 签名 → 字段不存在或值不匹配 → 验签恒失败；生产 restore 恒 `restore_unknown`／`ErrAdmissionBinding`（fail-closed，但契约不可用）。
2. 实现者为了让 V8 可运行，自行把 producer 载荷"统一"为 8 字段（例如删掉 `anchor_ref`／验证方身份，或把 attestation 字段裁到 8 个）→ **验证方签名不再绑定 `anchor_ref` 与 `verifier_principal_id`／`verifier_key_id`**，直接重开 N001 的绑定缺口；这是安全失败方向。
3. 两个实现者分别按 `:4062` 与 `:3988` 实现 → 产出方与验证方对"签名载荷"取值不同，互操作失败或接受不同字节，形成签名覆盖面的实现分歧。

### 3.3 影响

- BLOCKER：r04 要求"验证方与产出方取值不得歧义"未达成；本轮只闭合了 **V 编号唯一**，没有闭合 **载荷指向唯一**。
- `:3990` 的"仅历史"标记并不能消除 `:4062` 现行规范句的错误；历史行里重复同一错误会强化误解。
- 失败方向是安全侧的：最省事的"统一"改法会削弱验证方签名对锚与身份的绑定。

### 3.4 修复建议（不代作者修改）

- 把 `:4062` 的 V8 改为引用 `:3988` 的 producer envelope，并写明"签名对象 = `JCS({"receipt": R10, "attestation": A6})`；`R10` 删除 `signature`／`audit_record_ref`，`A6` 删除 `signature`"；删除"8 字段载荷见 r65.2"。
- 同步修正 `:3990` 的历史映射句，避免历史行再引用 r65.2 的 8 字段载荷。
- 增加可机核判据：`rg 'V8.*r65\.2'` = 0；`rg '8 字段'` 只命中验证方载荷与 V10 语境；producer 载荷在全文只有 `:3988` 一处定义。

### 3.5 证据、未验证风险与复审条件

- 证据：上述行原文；`rg '8 字段|八字段|8 项'` 命中集合；两条 golden vector 与角色枚举核对（与本 finding 无直接关系，但排除了"向量导致角色冲突"的旧解释）。
- 未验证风险：未做实现级验签实验；本 finding 是文本契约与调用顺序的可实现性判断。若作者能给出"r65.2 同时定义 producer 与 verifier 两种 8 字段载荷"的正文，需重新判定；当前全文不存在该正文。
- 复审条件：V8 只引用 producer 载荷；`:3990` 历史映射不再把 8 字段验证方载荷指给 V8；产出方/验证方载荷各只有一处定义且互不混用。

## 4 仍开放 HIGH

### 4.1 N011（HIGH）`witness_endpoint` 独立性——部分闭合，仍开放

**已闭合部分**：`:4060` 已加六项合取：(a) `https://` 绝对 URL；(b) 解析后全部 A/AAAA 非 loopback／link-local／私网／ULA／未指定地址，并给出至少包含 `127.0.0.1`、`::1`、`10.0.0.1`、`192.168.1.1`、`fc00::1`、`0.0.0.0` 的拒绝清单；(c) host 精确匹配；(d) 禁重定向；(e) 禁 userinfo／fragment／任何 query；(f) 只信任 `witness_tls_spki_sha256` 的 SPKI 且强制 hostname 校验、禁 `InsecureSkipVerify`、禁系统默认根替代 pin。判定句给出 `ErrAdmissionBinding`／`ErrAdmissionUnsupported` 且保持 closed。部署前置"见证方与部署方／root 分属不同管理域"已写，`r65.13(3)` 已登记残余。相对 r04 的"本机冒充"路径，直接写 `127.0.0.1`／`::1` 已被拒绝。

**仍开放的子项**：

1. **DNS rebinding TOCTOU 未关闭**。设计只要求"解析后"检查 A/AAAA，未要求把预检解析出的 IP 集合固定给连接（如自定义 `DialContext` 连到已核 IP），也未要求连接后复核 `RemoteAddr` 或"连接时解析结果与预检集合相等"。触发场景：部署方控制域名，预检时返回公网 A，连接时返回 `127.0.0.1`；其在本机运行持有该域名证书与登记 SPKI 私钥的 HTTPS 服务，即可让"外部见证"重新等于本机。r04 用该场景判 N011 未闭合，r65.3 只加了预检解析约束，未加"检查—使用"绑定，因此同一攻击仍成立。
2. **host 精确匹配近乎同义反复**。`receipt_anchors` 的 14 项 keyset 没有独立的 witness host 登记字段；"host 与登记值精确相等"中的"登记值"只能是 `witness_endpoint` 自身的 host，比较对象与操作数同源，不能防止别名/解析替换。真正起作用的是 (b) 的 IP 区间检查与 (f) 的 pin；而 (b) 有第 1 项的 TOCTOU。
3. **IPv4-mapped IPv6 与非规范 IPv4 字面值未覆盖**。拒绝清单只有 `127.0.0.1`、`::1` 等；未写 `::ffff:127.0.0.1` 的 `Unmap`，也未写 `127.1`、`0x7f.0.0.1`、`0177.0.0.1` 一类非规范写法。若实现按字面清单逐项比较而未先做 `Unmap`／规范化，映射回环地址可通过。
4. **pin 与"不得 `InsecureSkipVerify`"的组合未给出实现路径**。Go 标准库下，若只信任 SPKI pin 而不信任系统根，通常需要 `InsecureSkipVerify=true` 加自定义 `VerifyPeerCertificate`／`VerifyConnection`（手动校验 SPKI、hostname、有效期）；本条同时禁止 `InsecureSkipVerify`，又要求"只信任 pin"，未说明如何同时满足。实现者可能改为"系统根 + 额外 pin"（与"不得以系统默认根替代登记 pin"冲突），或违反禁令使用 `InsecureSkipVerify`（与 (f) 冲突）。这是可实现性与安全语义的双重歧义。
5. **pin 失效／轮换／回滚路径未定义**。`witness_tls_spki_sha256` 在信任根内，但设计未写轮换需人的明确批准、旧 pin 的保留窗口、轮换失败时的 closed 行为；ADR-0031 `:1166` 只对回执验签信任锚写了"轮换经人的明确批准"。当前失败方向是 fail-closed（pin 不匹配 → `ErrAdmissionUnsupported`），属可用性风险，不是绕过。
6. **独立性清单漏 verifier**。`:4060` 的独立性合取要求 `witness_principal_id`／`witness_key_id` 的 `org_id` 不得等于 root／executor／producer／issuer／reviewer，**不含 verifier**；`witness_key_id` 在 `keys` 记录中没有 `org_id` 字段（`:3934` 的 keys keyset 只有 `key_id, principal_id, alg, public_key, valid_from, valid_until, revocation`），必须经 `keys.principal_id → principals.org_id` 解析，设计未写该解析步骤。`witness_principal_id` 与 verifier 的同一人风险由六席 `person_ref` 两两不等覆盖，但同组织风险未覆盖。

**影响**：N011 想关闭的"外部见证可由本机／同域冒充"在 DNS rebinding 路径下仍成立；该路径同时是特权进程对任意主机的出网通道。pin 组合歧义会让实现者在两个都违反某条约束的方案中二选一。故 N011 不能判闭合。

**修复方向**：预检解析后把 IP 集合固定到连接层（连接已核 IP，或以 SNI/Host 保持域名但 `DialContext` 只连已核 IP），连接后复核 `RemoteAddr` 属于该集合；拒绝清单补 `::ffff:127.0.0.1`（先 `Unmap`）与非规范 IPv4；显式允许"仅见证客户端"使用受控的自定义校验（或改用专用 RootCAs + pin），并写清 hostname 校验与有效期；补 pin 轮换/回滚的人工批准与 closed 语义；把 verifier 纳入组织独立性或写明排除理由，并给出 `witness_key_id → principal → org_id` 的解析链。

**未验证风险**：无 DNS/TLS 实验环境；上述为文本与协议层推理，未做 rebinding 复现，也未验证真实 witness 服务。复审条件：`witness_endpoint` 约束段出现"连接使用已核 IP／连接后复核"和 `Unmap`；pin 组合有唯一实现路径；轮换路径与独立性链补齐。

### 4.2 NEW-S05-01（HIGH）`PATH` 放宽的抵消措施不足以维持最小权限

**位置**：`:4057`（r65.0 放宽登记第 3 项）、`:4063`（r65.6）、`:3976`（r63.6）、`:4039`（r64.6）、`:4423`（r61.5 原字节）。

**事实**：`PATH` 的抵消措施只有两句："每个目录必须是绝对规范路径并列入批准记录"；`LD_*`／`*_PRELOAD`／`*_OPTIONS`／`GLIBC_*`／`MALLOC_*` 出现即 `ErrAdmissionClosed`。没有：

- 目录属主必须 root、group/other 无写、executor 不可写；
- 目录路径逐级无符号链接、与 loader 父目录/挂载检查同一覆盖；
- 禁止用 `exec.LookPath`／裸名解析启动任何子进程（或明确子进程 exec 不在本合同内）；
- 批准记录中 `PATH` 目录的独立字段与 schema（当前 `tool.environment` 只是 `KEY=VALUE` 字符串数组，`hashes` 中 tool 向量甚至不含 `PATH`）；
- 对 `PYTHONPATH`／`NODE_PATH`／`RUBYLIB`／`PERL5LIB` 一类解释器路径变量的显式拒绝（虽然 `env_clear` + 封闭 allowlist 已使它们不能进入子进程，但设计未把这层理由写成可核判据）。

**触发场景**：批准记录把 `PATH` 指向 `/usr/local/bin`（root 拥有但 group 可写）或 executor 拥有的目录；同机低权用户或 executor 自身在该目录放置与工具后续调用的 helper 同名的可执行文件；工具以 executor 身份调用裸名 helper → 执行攻击者代码。设计未禁止工具调用子进程，也未要求这些目录不可写，故该面存在。

**违反依据**：最小权限与"批准目录内的封闭字面集"要求（ADR-0031 `:1144`／`:1164`）；`:1144` 只要求"`PATH` 指向任一非批准目录即 closed"，未要求目录不可写，因此本项不是 ADR 违反，而是设计抵消措施不足。

**影响**：`PATH` 从"禁止"放宽为"允许"后，可执行文件搜索面扩大；若目录可写，形成二进制种植面。r65.13(4) 只登记"搜索面扩大"，未登记"目录可写导致种植"这一具体残余。

**修复方向**：为 `PATH` 目录增加 root 属主、group/other 与 executor 不可写、逐级无符号链接、纳入 loader/挂载父目录检查；在批准记录 schema 中给出目录数组载体；明确工具不得用 `PATH` 解析自身或子进程可执行文件（或把子进程 exec 明确排除并 closed）；残余段补登二进制种植面。

**未验证风险**：未在 Linux 上验证目录权限与 helper 解析；本项为设计文本的最小权限判断。复审条件：上述约束进入 `:4057`／`:4063` 与批准记录 schema，并有可机核判据。

## 5 仍开放 MEDIUM

### 5.1 NEW-S05-02（MEDIUM）`PATH` 的"允许"与 r61.5 `:4423` 的"禁止"两套规范句并存

**事实**：`:4423`（r61.5，现行原字节）写"`env_clear` 后只设置批准记录 `environment` 数组内逐项固定的字面集（建议 `LANG=C.UTF-8`、`LC_ALL=C.UTF-8`、`TZ=UTC`）；**禁止 `PATH`**、`LD_PRELOAD`……"。`:4063`（r65.6）写"ENV 允许键为 `LANG`、`LC_ALL`、`TZ`、`PATH`"，并要求 r61 段保持原字节、不得留下 r64／r65 就地标记；`:4057(15)` 也要求恢复 r61 原字节。

**问题**：取代关系只由 `:4039`（r64.6）与 `:4063` 的元声明承担；`:4423` 本身没有标记，也没有"本条已被 r65.6 取代"的就地指针。r65.0 的取代清单是列举式，没有一条全局"r65 与 r61／r62 字面冲突一律以 r65 为准"的优先级句（只有 `:4058` 对 r63.9.1／r63.9.2／r64.1 的局部优先级）。实施者读到 r61 的"唯一可编码形态"章节时，会得到"禁止 `PATH`"的直接规范句；读到 r65 时得到"允许 `PATH`"。

**影响**：两种实现都能在文本中找到依据：按 `:4423` 实现会拒绝含 `PATH` 的合法批准记录（fail-closed，但与冻结 tool 向量/批准 schema 不一致）；按 `:4063` 实现则启用 NEW-S05-01 的放宽面。该歧义与 SEC-F003 的"两套规范句"同类，只是失败方向偏向可用性，故定 MEDIUM 而非 BLOCKER。

**修复方向**：在不改 r61 字节的前提下，于 r65.0 增加一条明确的全局优先级句（"r65 与 r61／r62 正文的字面冲突一律以 r65 为准"），并在 `:4063` 列出被取代的 r61 行号与冲突字段；或把 `PATH` 的允许写成"仅在本节定义的批准记录 schema 扩展生效时允许"，使两套文本可判定。

**复审条件**：存在唯一可判定的优先级规则，且 `:4423` 与 `:4063` 的冲突有明确指向。

## 6 仍开放 LOW

### 6.1 NEW-S05-03（LOW）作者报告终值与 r61 起点行号不一致

- 见 1.3：`1,045,042 B / D2A6…` 为过时值；现行 `1,045,111 B / 87D70…`。作者报告"净值"行已提到 `+69 B`，但表格与 hashes.txt 未回写，导致报告不能绑定本轮对象。
- 作者报告把 r61 段起点写作 `:4247`；现行 r61 标题在 `:4278`，`:4247` 是 r62.9.1 文件表表头。
- 影响：复审/实施若按作者报告取哈希或行号，会核对到错误对象或错误段落。该报告只能作过程材料，不能作本轮对象的固定证据。
- 修复：回写终值哈希与行数；把 `:4247` 改为"r62 尾段起点"，r61 段起点写 `:4278`；重新生成 hashes.txt。

### 6.2 NEW-S05-04（LOW）精度残余（逐项）

1. **2048 B 上限丢失**：`:4033`（r64.3）原有"`witness_endpoint` 长度上限 2048 字节"；`:4060`（r65.3）重列 endpoint 约束 (a)–(f) 时未保留该上限，且标记"endpoint 约束以 r65.3 为准"。这是未登记的约束删除（容量类，非提权），建议补回或明确作废理由。
2. **host 精确匹配无独立字段**：已在 N011 子项 2 说明；作为 LOW 精度项记录在 `receipt_anchors` 14 项 keyset 缺少 witness host 字段。
3. **`witness_key_id` 无 `org_id` 字段**：`:4060` 写"`witness_key_id` 的 `org_id`"，但 `keys` keyset（`:3934`）没有 `org_id`；必须经 `keys.principal_id → principals.org_id` 解析，设计未写解析步骤。
4. **pin 轮换/回滚未定义**：见 N011 子项 5；建议补"轮换需人的明确批准、旧 pin 保留窗口、失败 closed"。
5. **root／supplier 任命值域未定义**：`:3936(3)` 要求**每个** `principal.appointment_ref` 解析到 `appointments` 唯一记录，而 `:4061` 的 `appointments.role_name` 封闭枚举只有六个 `restore_*` 值；`principals.kind` 仍含 `root`／`supplier`，设计未给 root／supplier 的豁免或对应 role 值。按字面实现，root principal 无法登记（信任根恒被拒，fail-closed）；若实施者自行加"root 免 appointment 解析"的豁免，则该豁免未在契约中声明，需重新评审。建议显式写"root／supplier 的 appointment_ref 为可空且不参与角色判定"或给它们独立 role 值。
6. **`sequence_floor` 历史残留**：`:4078` 的历史 keyset 仍含 `sequence_floor`，但已有 r64.3 标记；`:4330` 的 r61 示例同样属历史，`:4049` 有全局说明。这些不构成现行规范冲突，仅记录为上下文。

## 7 已闭合项的证据（设计层，不等于实现或运行通过）

### 7.1 SEC-F002（BLOCKER）→ 已闭合（设计层）

- `:3936(3)` 已重写：要求 `appointment_ref` 解析到唯一 appointment，且 `principal_id`／`org_id`／有效期逐项相等；明确"`principals` 没有 `role_name` 字段，不存在该比较"；`appointment_ref` 不得绑定多个 principal。全文 `rg 'role_name.*与 principal 侧一致'` = 0。
- `:4061`（r65.4）：`appointments.role_name` 六值封闭枚举 `restore_approver`／`restore_reviewer`／`restore_executor`／`restore_producer`／`restore_verifier`／`restore_witness`，枚举外 → `ErrAdmissionBinding`；裸名到 `restore_*` 的显式映射；`person_ref` 在全部 active principal 中全局唯一；六席 `person_ref` 两两不等；`witness` 是角色不是 kind（`kind=verifier` + `role_name=restore_witness`）。
- `:3937`：root 独立性检查"必须能从信任根字段机核，不依赖调用者自报"；无 root 登记／root 公钥未登记／无法证明调用进程不是 root → `ErrAdmissionClosed`。
- `:4037`：P3 预检"构造不可变快照并完成全部合取"，为上述约束提供了统一检查点；`prepared(ctx)` 与 R2／R4／R6、V2／V4／V6 的重验保持。
- 冻结向量角色字面全部在六值枚举内（1.2 已复算）。
- 残余已诚实登记于 `:4070(1)`：`person_ref` 是自证标识，同一自然人用两个 `person_ref` 不可检测；本报告接受该残余为已声明风险，不把它当作未闭合项。SEC-F002 的三个 r04 子项 (a)(b)(c) 均有可机核判据，判**已闭合（设计层）**。

### 7.2 SEC-F006（HIGH）→ 已闭合（登记层）

- `:4057` 的"放宽登记（3 项）"逐项给出放宽内容／理由／影响／抵消措施／残余：`NoNewPrivs` 降为事后观测值；`CapBnd` 降为事后观测值；`PATH` 相对 r61.5 重新允许。另有"非安全放宽、仅容量变更（1 项）"：信任根上限 `65536` → `1048576` 字节（`:4049`），判定位置、失败分类 `ErrAdmissionIncomplete`、不改准入判定集合三点齐备。
- `:4064`（r65.7）复述 `CapInh=CapPrm=CapEff=CapAmb=0`、`Uid`／`Gid` 四元组、`Groups` 为空、`AT_SECURE=0` 仍是拒绝条件；`CapBnd`／`NoNewPrivs` 只允许记 `unknown`／NOT RUN；seccomp／netns／setns 一律 unsupported。
- `:4203` 已加"`CapBnd`、`NoNewPrivs` 降为观测值"标记；`AT_SECURE=0` 的"要求"与 `:4064` 的"仍是拒绝条件"方向一致（满足才通过，不满足才拒绝），未发现方向反转。
- `:4070(4)(5)` 复述三项放宽的残余与容量变更。
- r04 的判据"放宽登记逐项列出三项，各含内容/理由/抵消/残余"已满足，故原 SEC-F006 **已闭合**。但 `PATH` 抵消措施的实质强度不足，另立 NEW-S05-01；这不否定登记完整性，也不把 SEC-F006 重新计为开放。

### 7.3 N012（LOW）→ 已闭合（设计层）

- `:1718` 的 `receipt_version` CHECK 已改为 `'han-restore-receipt/v1'`，并带 r65.8 取代标记；`RestorePlan.Version` 的 `han-db-restore/v1`（`:3878`）保留，两个版本域已分开。
- `:4065`（r65.8）点名第三别名 `restoreReceiptWire`（代码 `:52`–`:65`，含 `receipt_digest,omitempty`），要求改造后 `rg restoreReceiptWire cmd` = 0。
- 当前代码实测 `rg restoreReceiptWire cmd` = **3**（`recovery_restore.go:52`、`:129`、`:147`），与设计"未实施"声明一致；不作为设计层未闭合。

### 7.4 N013（LOW）→ 已闭合（设计层）

- `:4066`（r65.9）明确：DSN 进入子进程的唯一路径是 `privateRestoreInput` 的匿名 `Target` wire（代码 `:74`–`:76` 的 `Target.DSN`）；该字段**删除**（不是改类型），DSN 由 `appendJSONString` 直接追加；`restorePlanWire`／`restoreReceiptWire` 保持 `string` 且无 DSN 字段。
- `:4129` 已加消解句：`:4114` 的"wire 结构体字段类型仍是 `string`"只针对 `restorePlanWire`／`restoreReceiptWire`，与匿名 `Target` 删除 `DSN` 不冲突。
- 代码实测：`restorePlanWire`（`:36`–`:50`）与 `restoreReceiptWire`（`:52`–`:65`）无 DSN 字段；`Target.DSN` 在 `:78` 与测试 `:45` 共 2 处；`appendJSONString` 当前 0 处。均与"设计要求、未实施"声明一致。

### 7.5 N014（LOW）→ 已闭合

- `:4067`（r65.10）：`static` 是默认与首选，不再称"唯一完整可绑定形态"；`dynamic` 仅在无任何 `LD_*`、`PT_INTERP` 来源、`ld_so_cache_path`／`ld_so_conf_paths` 闭包逐项相等、解释器自身 `DT_NEEDED` 并入、父目录与挂载规则覆盖全部路径的合取下允许；无法证明 → `ErrAdmissionUnsupported`（closed）。
- `:4039`（r64.6）已就地改写为"默认与首选"并加 r65.10 标记，两段不再互斥。

### 7.6 N015（LOW）→ 已闭合（设计层）

- `:4062` 明确：代码 `recovery_restore.go:352` 在 `tool.Run` 之后调用 `d.prepared(ctx)`（含 `CheckApproved`、`checkRestoreArtifact`、`Seek(0)`）是既有实现，归入 **R6 的第二次同构复核**，不新增编号；`execute` 之后必须保留，不得删除；R1–R8／V1–V11 只枚举契约检查点，不是代码检查点的穷举。
- 代码核对：`:347 d.tool.Run` → `:352 d.prepared`；`:346`／`:351` 两个 `defer clear`；`:392 d.dsn=""`；`prepared` 在 `:247`–`:255` 调用 `CheckApproved` 与 `checkRestoreArtifact`，后者在 `:269`／`:285` 两次 `Seek(0)`。设计描述与代码一致。

### 7.7 N016（LOW）→ 已闭合

- `:4012` 已改为"外部见证客户端、`baseline_sequence` 基线与见证 head `sequence` 的原子推进"，并加 r65.3 标记。
- 其余 `sequence_floor` 命中：`:3934`（带 r64.3／r64.4 取代标记）、`:3942`（带 r65.5／r64.5 历史标记）、`:4027`（修订史）、`:4033`（改名说明与旧键拒绝）、`:4049`（r61 历史说明）、`:4057`（取代清单）、`:4078`（历史 keyset，带 r64.3 标记）、`:4082`（历史字段清单，带 r64.3 标记）、`:4143`（带 r64.3 标记）、`:4330`（r61 示例，`:4049` 全局覆盖）。无未标记的现行规范命中。

## 8 范围外观察：N005（不计入本报告计数，但阻断启动）

- `framework.md:1084`（状态行）：现行修订 `adr-0031-r09` 未经复评，决策接受级对 r09 未达成，`identity-auth-S4.24a` 及任何实施/验证卡不得启动。
- `framework.md:1129`：`adr-0031-r06` 已被两路 r02 接受，但现行修订已是 r09，r09 尚未复评；在 r09 被架构与安全两路接受之前，S4.24a 及任何实施/验证卡不得启动（启动门禁回到 blocked）。
- `decisions/README.md:39`：索引行写 `adr-0031-r09（未复评；外部独立见证签名；模拟见证未真实验证）`。
- `tasks.md:3197`：仍写"`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受……决策接受级对 r07 成立，故本卡启动门禁解除"；同一段又承认 r61 设计修订被判 CHANGES REQUIRED、本卡仍不可进入实施。
- `tasks.md:3186`：`identity-auth-S4.24a` 仍为 `- [ ]`。

**性质**：这与设计 r65 的文本修订不是同一对象；设计侧只能登记，不能裁定。N005 未解决，且其当前形态是 **ADR 方（r09 未复评 → blocked）与任务卡方（r07 已接受 → 门禁解除）的事实冲突**，属启动门禁阻断，不是本报告计数的设计 finding。作者 r65.0 已声明"本轮不修改 ADR、不主张其评审状态"，该声明正确，但不能替代裁定。

## 9 额外必查项（本轮实际核对）

- **ENV 注入面**：允许键只有 `LANG`／`LC_ALL`／`TZ`／`PATH`；`env_clear` 后逐项设置；`LD_*`（含 `LD_LIBRARY_PATH`）、`*_PRELOAD`、`*_OPTIONS`、`GLIBC_*`、`MALLOC_*` 出现即 `ErrAdmissionClosed`。由于未列出的键不进入子进程，`PYTHONPATH`／`NODE_PATH`／`RUBYLIB`／`PERL5LIB` 一类变量不能经 ENV 注入；这一点成立。剩余风险集中在 `PATH` 的目录权限与搜索面（NEW-S05-01）以及 `:4423` 的字面冲突（NEW-S05-02）。
- **endpoint／DNS／TLS**：六项约束已加；直接回环/私网/链路本地/ULA/未指定地址的预检拒绝成立；DNS rebinding、IPv4-mapped IPv6、host 同义反复、pin 实现路径、pin 轮换仍未闭合（N011）。
- **audit sink**：`:4033` 仍要求 policy 的 `audit_sink` 与 `receipt_anchors.audit_sink` 逐字节相等；r65 未回退 N008 的结论。
- **sequence**：`:4060` 的 `baseline_sequence` 单调不减、重钉需人的明确批准并留审计、本次 `sequence` 严格大于 `max(baseline_sequence, head.sequence)`；`sequence_floor` 旧键出现即未知键。语义未回退；外部见证的传输/冒充面见 N011。
- **第二 authority／fallback／隐式默认**：r65.0(5) 明确不新增；`:4058` 保持同一 `*recovery.Runtime`；`:4068` 的 `injectedExclusive` 与 `ownedExclusive` 两字段互斥、`injectedExclusive` 永不 `Close`、装配体唯一关闭方；未发现新增第二 authority 或 fallback 分支。
- **冻结向量**：两条字节与摘要独立复算一致；角色字面在六值枚举内；tool 向量不含 `PATH`。
- **代码基线**：`restoreReceiptWire` 3 处、`Target.DSN` 2 处、`appendJSONString` 0 处、`prepared` 节点与设计描述一致；这些是"未实施"证据，不是实现通过证据。

## 10 NOT RUN 边界（SEC-F007）

本轮**没有**运行任何真实环境验证。以下全部为 **NOT RUN**，不得写成 PASS，也不得由本轮复审改判：

- Linux host 层的 host／ACL／mount／loader／fd-exec／maps／network／witness 实测；
- 真实 Ed25519 密钥／HSM／密钥轮换；
- 真实信任根、真实批准记录、真实工具与真实备份的 restore E2E；
- 生产 restore 的 `enter`／`resume` 路径；
- netns／seccomp／setns 能力（设计已声明 unsupported，需要即 `ErrAdmissionUnsupported`）；
- `AT_SECURE`、`CapBnd`、`NoNewPrivs`、`/proc/<pid>/maps` 的 host 观测；
- DNS rebinding、TLS SPKI pin、真实 witness 服务的协议联调。

缺少的运行条件：具备 root／`CAP_SYS_ADMIN` 的 Linux 宿主；POSIX ACL 工具与 `ro,nodev,nosuid,noexec` 挂载；glibc loader 与 `PT_INTERP`／`DT_NEEDED` 样本；fd-exec 与 `/proc` 可读（非 hidepid 阻断）；独立网络命名空间与外部 witness 服务；真实密钥材料与批准记录。Windows 侧只覆盖解析与语义层；`//go:build linux` 的 skip 与无匹配测试的 exit 0 都不等于 PASS。

## 11 结论与边界声明

**Verdict: CHANGES REQUIRED**。范围内 BLOCKER 2 项（SEC-F003、N001，同一根因）；HIGH 2 项（N011、NEW-S05-01）；MEDIUM 1 项（NEW-S05-02）；LOW 2 项（NEW-S05-03、NEW-S05-04）；NOT RUN 1 项（SEC-F007）；范围外 BLOCKER 1 项（N005，不计入上表但阻断启动）。

本报告**不构成**：生产 restore 解锁；`enter`／`resume` 解锁；ADR-0031 约束 8 满足；N005 已解决；`identity-auth-S4.24a` 可启动或可关闭；任何人的成果接收或批准。SEC-F007 仍是 NOT RUN。SEC-F002／SEC-F006／N012–N016 的"已闭合"仅指**设计文本层**的可核判据成立，不代表代码实现、真实环境、E2E 或生产可用。

## 12 复审条件

- SEC-F003／N001：V8 只引用 `:3988` 的 producer 载荷；`:3990` 历史映射不再把 8 字段验证方载荷指给 V8；全文 producer／verifier 载荷各只有一处定义。
- N011：补"连接使用已核 IP／连接后复核"、`Unmap` 与非规范 IPv4、pin 的实现路径与轮换、verifier 的组织独立性或排除理由、`witness_key_id → principal → org_id` 解析链。
- NEW-S05-01／02：`PATH` 目录的属主/可写性/父目录覆盖/子进程 exec 约束；`:4423` 与 `:4063` 的冲突有唯一可判定优先级。
- NEW-S05-03：作者报告与 hashes.txt 回写终值（1,045,111 B / `87D70…`），r61 段起点改为 `:4278`。
- 复审时须给出新的对象修订号与实测哈希；不得把本报告的哈希复用于后续修订。若对象变化，须先核对 diff 并明确本次修订，再逐条复核原 finding。