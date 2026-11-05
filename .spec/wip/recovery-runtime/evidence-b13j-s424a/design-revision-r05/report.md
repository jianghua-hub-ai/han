# identity-auth 设计 r64 → r65 修订报告（B13J／S4.24a，第三轮最小修复）

**本修订未经 R05 独立评审、未实施、未闭合、非真人签字。** 本文件是契约文本修订的证据记录，不是独立评审结论，不构成 PASS、不构成生产 restore 可用、不构成 `identity-auth-S4.24a` 可关闭、不构成 ADR-0031 约束 8 满足，也不构成任何人的批准或接纳。

## 一、修订对象、基线、实测口径与不变量

| 项目 | 值 |
|---|---|
| 工作区 | `main`／HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；未切分支、未 checkout／stash／reset／commit／push（本轮无任何 Git 写操作） |
| 修订对象 1 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| 输入基线（design） | `identity-auth-design-ir01-r64`；1,005,979 B／5,449 行／SHA256 `FB84EC1E0B078AC99B09A48988F00257B77CA93102298A7DAD5004E0B8871E94`（本轮开工前实测，与 `design-revision-r04/report.md` 第 9 行记录逐项一致） |
| 输出（design） | `identity-auth-design-ir01-r65`；1,045,042 B／5,464 行（`ReadAllLines`）／LF 5,464／CR 169／无 BOM／SHA256 `D2A6B54B2153858DC6366995C4332A380D9DA0BEDE863BDDC80D51663A3D8BE2`；12:38:45 写入后到本报告定稿前复核未再变化（复核窗口见 6.8） |
| 净值（r65 相对 r64） | **+39,063 B／+15 行**。终值相对 r65 初稿（1,044,640 B）另有四笔追加：并发写入者三次改写同一 12 行（+57 B、+103 B、+173 B）与本轮对 r65.0 ADR 表述的事实校正（+69 B），均不增行；本轮未回退并发改动，逐行登记见 6.8。 |
| 版本号行 | `:5` `revision:` 与 `:38` 设计版本号均为 `identity-auth-design-ir01-r65`；`last_updated: 2026-11-02` 未变 |
| r65 区块 | `:4057`–`:4070`（r65.0–r65.13）；`:4025`–`:4055` 为 r64 区块；`:4056`／`:4071` 为空行；`:4072` 起为 r62 历史段 |
| 修订对象 2 | `.add/project/knowledge/decisions/framework.md`（ADR-0031） |
| ADR 输出 | `adr-0031-r09`；272,588 B／1,271 行／LF 1,271／CR 1,271／无 BOM／SHA256 `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`；修订行 `:1082`、状态行 `:1084` |
| ADR 基线限制 | HEAD 版 `framework.md` 为 37,074 B 且不含 `ADR-0031`（检索计数 0），工作树中的 ADR-0031 是他人未提交内容；因此 `adr-0031-r07 → r09` 没有可独立复算的提交基线，本报告只登记现行 r09 的实测值与落点，**不声称复算 r07** |
| 修订对象 3 | `.add/project/knowledge/decisions/README.md`；`:39` 索引行为 `adr-0031-r09（未复评；外部独立见证签名；模拟见证未真实验证）`；21,613 B／110 行／LF 110／CR 2／无 BOM／SHA256 `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2` |
| 证据目录 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r05/report.md`（本文件）与 `hashes.txt`（哈希清单）；本轮以最终报告整体替换目录内旧草稿，未保留其增量内容 |
| 写集合规 | 只改上述三个文件与 `design-revision-r05/` 目录（含 `report.md` 与 `hashes.txt`）；未改 `tasks.md`（任何一行，含勾选）、`cmd/**`、`internal/**`、`test/**`、requirements、test-plan、其他设计文件；用户任务所称 r64 行号 `:4072`／`:4078` 的两条 golden vector（r65 中因 r65 区块插入 15 行后移至 `:4087`／`:4093`）字节未动（见第六节第 4 项） |

口径：字节 = `[IO.File]::ReadAllBytes`；行 = `[IO.File]::ReadAllLines`（UTF-8）；SHA-256 = 对整文件字节直接计算；行号 = 当前工作树文件行号（1 基）；检索区分大小写。基线来源说明：HEAD 的 `design.md` 仍是 `identity-auth-design-ir01-r09`（110,040 B），r61–r65 全部是工作树未提交内容，因此「改动前」取 r64 终态而非 HEAD，且该终态有 `design-revision-r04/report.md` 的独立记录可对账。

不变量（`r65.0` `:4057` 逐条判定）：（1）「首期不提供生产 restore 工具与生产批准主体」不变；（2）生产 restore 继续 closed，`recoveryClosedRestoreDriver` 仍是默认；（3）`enter`／`resume` 不因本修订解锁；（4）任何 NOT RUN 仍是 NOT RUN，不得写成 PASS；（5）不新增第二 authority、fallback、隐式默认或「本次忽略」分支；（6）r60 的 11 端口清单、唯一 production root 与 ADR-0031 的解除条件不被改写；（7）不勾选 `tasks.md` 的 `identity-auth-S4.24a`，不构成约束 8 满足。`tasks.md:3186` 实测仍为 `- [ ]`（见 6.5）。

r65 对既有正文的处置：只做 `:3934`／`:3936`／`:1718`／`:3995`／`:3997`／`:3990`／`:4012`／`:4039`／`:4045`／`:4129`／`:4203`／`:4423` 等被评审逐条点名的就地改写或句尾取代标记，其余 r61／r62／r63／r64 字节不动；r61 段（现行 `:4247`–文件末）的 `r64`／`r65` 就地标记实测为 **0**。lease 相关的 12 行并发改写见 6.8。

## 二、逐条 finding 落点表（Finding → r65 子节 → 行号 → 可机核判据）

| 编号 | 级别（r04 原文） | Finding 摘要 | r65 落点 | 行号 | 可机核判据（在当前 `design.md` 上检索／复算） |
|---|---|---|---|---|---|
| ARCH-F006 | BLOCKER | `recoveryFlow.acquire` 是 restore 唯一取得点却被划为「非 restore 流」；第二次 `Acquire` 阻塞或 `ErrAdmissionClosed`；`Runtime` 单一 `exclusive` 字段与 `Stop` 无条件 `Close` 无法区分注入 lease | r65.1；r65.11；r65.12；r65.0 取代清单（6） | `:4068`；`:4069`；`:4058`；`:3997`；`:4029` | 现行设计把 `recoveryFlow.acquire`（定义 `internal/recovery/runtime.go:1554`；用户评审引用的 `:1574`／`:1648` 是 `f.acquire(ctx)` 在 `enter`／`resume` 的调用点）明确归入 restore 流：删除 `deps.Exclusive.Acquire`（`:1562`）与 `f.r.exclusive = l`（`:1564`），只读 `f.r.injectedExclusive` 并做 `checkExclusive` 等价持锁校验；`Runtime` 在 `:768` 的单一 `exclusive` 拆为 `ownedExclusive`（provider 路径自取，Runtime 唯一关闭）与 `injectedExclusive`（构造期一次赋值，永不 `Close`）；装配阶段 `Acquire(ctx)` 恰好一次，以非可选参数 `restoreLease` 传入 `recovery.NewRuntime(ctx, deps, restoreLease)`，同一 lease 传 `driver.Bind`，装配体是唯一 `Close` 方（`defer`，恰好一次）；`install`／`admission` 的 provider 路径保持原样。可机核判据（设计文本）：`:4068` 含「属于 restore 流」「只读 `injectedExclusive`」「`injectedExclusive` **永不 `Close`**」「只 `Close` `ownedExclusive`」，并给出 `rg 'func NewRuntime\(ctx context.Context, d RuntimeDependencies, restoreLease ExclusiveLease\) \(Runtime, error\)' internal/recovery` = 1、`rg 'recovery.NewRuntime\(ctx, deps, restoreLease\)' cmd` = 1、`recoveryFlow.acquire` 内 `rg '\.Acquire\('` = 0 等实现级判据；`:4069` 登记 `joinWork`（`recovery_runtime.go:941`–`:964`）只覆盖 `normal`／`resume`、`enter` 仍走 `ErrAdmissionBinding` 的实施缺口，并要求 Task-Planner 扩展 teardown。当前 `design.md` 全文实测：`BindRestoreLease` = **0**、`NewRuntime(ctx, deps, restoreLease)` = **5**、`ownedExclusive` = **14**、`injectedExclusive` = **29**。注意：该方案改变了 `NewRuntime` 签名，与用户 BLOCKER 1 的括号替代路线（把注入点下移到 `recoveryFlow`、不改签名）不同；是否满足 r60 `:3603` 的先行修订要求待 R05 判定，本报告不称其已闭合。 |
| SEC-F002(a)(b)(c) | BLOCKER | (a) `:3936` 交叉约束(3) 比较 principal 侧不存在的 `role_name`；(b) `role_name` 无封闭枚举，裸角色名与冻结向量字面互斥；(c) witness 未纳入 `person_ref` 互斥 | r65.4；就地改写 `:3936`；`:3934` 标记；残余 `:4070`(1) | `:4061`；`:3936`；`:3934`；`:4070` | `:3936` 约束(3) 已无 principal 侧 `role_name` 比较（句含「`principals` 没有 `role_name` 字段，不存在该比较」）；`:4061` 含 `appointments.role_name` 六值封闭枚举 `restore_approver`／`restore_reviewer`／`restore_executor`／`restore_producer`／`restore_verifier`／`restore_witness`、裸名显式映射、`person_ref` 全局唯一、六席（reviewer／issuer／executor／producer／verifier／witness）两两不等；冻结向量 `restore_reviewer`／`restore_approver`／`restore_executor` 未改（见第六节第 3 项） |
| N011 | HIGH（安全 r04 新增；用户追加任务要求按 BLOCKER 级处置） | `witness_endpoint` 可被本机冒充：无 loopback／私网／link-local／ULA 限制，无 host 精确匹配、TLS 锚与凭据位置约束 | r65.3；残余 `:4070`(3)；ADR `:1147`／`:1166` | `:4060`；`:4070` | `:4060` 含六项合取：(a) `https://` 绝对 URL；(b) 解析后**全部** A／AAAA 非 `127.0.0.0/8`／`::1`／`169.254.0.0/16`／`fe80::/10`／`10.0.0.0/8`／`172.16.0.0/12`／`192.168.0.0/16`／`fc00::/7`／`0.0.0.0`／`::`；(c) host 与登记值精确相等；(d) 禁止重定向；(e) 禁止 userinfo／fragment／任何 query；(f) 只信任 `witness_tls_spki_sha256` 且强制 hostname 校验；判据句写明预检必须 `ErrAdmissionBinding` 或 `ErrAdmissionUnsupported`；不同管理域为部署前置并登记残余 |
| SEC-F003／N001（同源一半） | BLOCKER | `:3990` 仍是 5 字段验证方载荷与旧 V7／V8／V9；与 r64.5 的 V7–V11、r64.2 的 8 字段冲突；撤销／TOCTOU 未在每检查点重验 | r65.5；r65.2；r65.3；`:3990` 标记；ADR `:1118` 撤销同步 | `:4062`；`:4059`；`:4060`；`:3990`；`:4037` | V 编号全文唯一：`:4062` 固定 `V7` 独立读取→`V8` producer 验签→`V9` 见证 head→`V10` 审计／验证方签名／原子推进→`V11` 返回副本；`:4059` 固定 8 字段载荷（`anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id`）；`:3990` 句首已标「旧文字仅作历史，已由 r65.5 整体取代」；撤销／过期在每个检查点重读并以 closed／`restore_unknown` 收口（ADR `:1118`） |
| ARCH-F004 | HIGH | `:3997`「新的只读依赖」与 `:3995`／`:4029`「不是新依赖项」字面互斥；r60 `:3603` 的优先关系未处理 | r65.1；r65.0 取代清单（2） | `:4058`；`:3995`；`:3997`；`:4029` | 唯一现行表述：`restoreLease` 是 `exclusive` 端口一次 `Acquire` 的**产出值**，不是端口／不是可独立配置或替换的依赖／不是新依赖项；以**非可选参数**进入 `NewRuntime(ctx, deps, restoreLease)`，构造期赋给 `injectedExclusive`；不进入 `RuntimeDependencies`、`recoveryRootInputs` 与 r60 的 11 端口表。`:4058` 明确「与 r63.9.1／r63.9.2／r64.1 的字面冲突一律以本条为准」；r60 `:3603` 原文未改，其「改变 `NewRuntime` 依赖集合须先修订本节」由 r65.1 的行号声明处理。因签名确实新增参数，是否构成 r60 要求的「先修订」需 R05 判定；`RuntimeDependencies` 端口集合未变。 |
| ARCH-F018 | HIGH | `:3990` 定义另一套 V7／V8／V9 与 5 字段载荷，与 r64.5 的 V7–V11、r64.2 的 8 字段冲突 | r65.5；r65.2 | `:4062`；`:4059`；`:3990` | `:4059` 的 8 字段 exact keyset 逐字出现且称「exact keyset 恰为 8 项」；`:4062` 的 `V7`–`V11` 与之一一对应；全文不再有第二套现行 V 编号（`:3990` 旧段仅历史并被标记） |
| SEC-F006 | HIGH | 放宽登记自称「唯一一项」，实际至少 `NoNewPrivs`／`CapBnd`／`PATH` 三项（另信任根上限 1..65536→1..1048576） | r65.0（3+1 项逐项登记）；r65.7；残余 r65.13(4)(5) | `:4057`；`:4064`；`:4070`；`:4027` | `:4057` 逐项给出放宽内容／理由／影响／抵消措施／残余（`NoNewPrivs`、`CapBnd`、`PATH`），并把信任根上限单列为「非安全放宽、仅容量变更」；`:4064` 重申 `CapInh=CapPrm=CapEff=CapAmb=0`、uid／gid 四元组、`Groups` 为空、`AT_SECURE=0` 仍是拒绝条件；`:4070`(4)(5) 复述残余；`:4027` 行尾已加「（r65.0／r65.13 取代）」 |
| NEW-R04-01 | MEDIUM | 放宽登记完整性（与 SEC-F006 合并处理） | r65.0；r65.7 | `:4057`；`:4064` | 同上；`PATH` 的抵消措施含「每个目录必须是绝对规范路径并列入批准记录」与 `LD_*`／`*_PRELOAD`／`*_OPTIONS`／`GLIBC_*`／`MALLOC_*` 一律 `ErrAdmissionClosed`；容量项的判定位置、失败分类 `ErrAdmissionIncomplete`、不改准入判定集合三项齐备 |
| NEW-R04-02 | MEDIUM | `:4423` 在 r61 正文行尾追加 r62.4 取代标记，违反「不改 r61 字节」不变量 | r65.6 | `:4063`；`:4423` | `:4423` 已恢复为以「子进程实际 `environ` 与白名单逐字节不等即 closed。」结尾；r61 段（`:4247`–文件末）`r64`／`r65` 标记实测 **0**；`:3927`／`:3999` 的「r61／r62 原文保留作历史」不变量未被改写 |
| NEW-R04-03 | MEDIUM | `baseline_sequence` 可被重钉降低且无人的批准要求 | r65.3；ADR `:1147`／`:1166` | `:4060`；`:4070` | `:4060` 含「只允许**单调不减**」「新值必须 ≥ 旧值」「必须由**人的明确批准**并留审计记录」「降低尝试、无批准证据的重钉 → `ErrAdmissionBinding`（启动前）／`restore_unknown`（已启动）」；ADR 不变量 6 与约束 8 第 6 条同步 |
| NEW-R03-03／N016 | MEDIUM／LOW | `:4012`、`:3990` 仍用 `sequence_floor`，与 `baseline_sequence` 并存 | r65.3 | `:4060`；`:4012`；`:3990` | `:4012` 行含 `baseline_sequence` 且行尾含「（r65.3 取代…）」；建议写集表 `:4000`–`:4019` 内 `sequence_floor` 实测 **0**；`:3990` 现行文字已用 `baseline_sequence` |
| NEW-R04-04 | LOW | witness 与 root／executor／producer 的独立性未显式要求 | r65.3；残余 `:4070`(3) | `:4060`；`:4070` | `:4060` 独立性合取：`witness_principal_id` 的 `org_id` 必须解析到 `organizations.kind = external_verifier`；其 `org_id` 与 `witness_key_id` 均不得等于 root／executor／producer／issuer／reviewer 任一方；违反 → `ErrAdmissionBinding` |
| NEW-R04-05 | LOW | `anchor_ref` 相等清单未含 `receipt_anchors.anchor_ref` 自身 | r65.5 | `:4062`；`:4031` 标记 | `:4062` 明确「从四处扩为**五处**」并逐项列出，首项即 `receipt_anchors.anchor_ref` 自身；`:4031` 行尾已加「（r65.5 取代：`anchor_ref` 等式链由四处扩为五处。）」 |
| N012 | LOW | `:1718` 的 `receipt_version CHECK='han-db-restore/v1'` 与唯一 wire `han-restore-receipt/v1` 不一致；`restoreReceiptWire` 未点名 | r65.8 | `:4065`；`:1718` | `:1718` 就地改为 `CHECK = 'han-restore-receipt/v1'` 并加 r65.8 标记；`RestorePlan.Version` 的 `han-db-restore/v1`（`:3878`）保留；`:4065` 点名 `restoreReceiptWire`（`cmd/han-server/recovery_restore.go:52`–`:65`）并给出目标判据 `rg restoreReceiptWire cmd` = **0**（当前实测 3，未实施） |
| N013 | LOW | `:4045` 指向的 `restorePlanWire` 无 DSN 字段；`:78` 改 `[]byte` 与 `:4114`「wire 仍 string」未消解 | r65.9 | `:4066`；`:4129`；`:4045` | 唯一改法：`privateRestoreInput` 匿名 wire 的 `Target.DSN` 字段**删除**（不是改类型），DSN 由 `appendJSONString` 直写；`restorePlanWire`／`restoreReceiptWire` 保持 `string` 且无 DSN 字段；`:4129` 已加 r65.9 消解句；目标判据 `rg Target\.DSN cmd` = **0**（当前实测 2，未实施） |
| N014 | LOW | `:4039`「static 是唯一完整可绑定形态」与同段允许 dynamic 自否 | r65.10 | `:4067`；`:4039` | `:4039` 已改为「`static`（无 `PT_INTERP`、`dt_needed` 为空）是**默认与首选**形态，不再称『唯一完整可绑定形态』」并加 r65.10 标记；`:4067` 给出 `dynamic` 的四项合取与不可证明时 `ErrAdmissionUnsupported` |
| N015 | LOW | 代码 `recovery_restore.go:352` 在 `tool.Run` 后有 `d.prepared(ctx)`，不在 R1–R8 内 | r65.5 | `:4062`；`:4037` | `:4062` 明确该调用「归入 **R6 的第二次同构复核**，不新增编号」，并要求 `execute` 之后保留、不得删除；`R1`–`R8` 与 `V1`–`V11` 未增点、未重排 |

说明：用户追加任务把 N011 列在 BLOCKER 组，`review-sec-r04` 将其登记为新增 HIGH；本报告按用户要求的严重级别处置，但评审级别以 `review-sec-r04` 原文为准，不对其改判。`review-sec-r04` 的 N005（ADR／`tasks.md` 状态冲突）属范围外，本轮只登记（见第八节）。
## 三、五个重合 BLOCKER 的现行处置

以下只说明设计文本落点与可机核判据，不主张评审闭合；R05 独立评审尚未发生。

| # | 原始 BLOCKER（用户 2026-11-02 授权口径） | 现行落点 | 处置要点 | 可机核判据 |
|---|---|---|---|---|
| 1 | 批准摘要自证（无可信外部锚） | r62.1 `:4076`；r65.3 `:4060`；残余 `:4070`(2)(3)；ADR `:1118` | 批准真实性来自批准文件之外的信任根（Ed25519、独立挂载来源、`HAN_RECOVERY_RESTORE_TRUST_ROOT_SHA256` 固定摘要），并由外部见证签名承担 head 权威；摘要只作完整性值 | 信任根 exact keyset 与 `version=han-restore-approval-trust-root/v1` 未变；`:4060` 的 `witness_tls_spki_sha256` 与六项 endpoint 约束齐备；`:4070`(2)(3) 明确「root 可整体替换信任根」「DNS 投毒／同管理域冒充」为残余 |
| 2 | 复核人独立性不可机核 | r65.4 `:4061`；就地改写 `:3936`；`:3934` 标记 | 独立性按 `principal_id`／`appointment_ref`／`org_id`／`person_ref` 机核，不按姓名；`role_name` 封闭枚举 + 裸名显式映射；六席 `person_ref` 两两不等（含 witness） | `:3936` 无 principal 侧 `role_name` 比较；`:4061` 六值枚举与映射表逐字出现；`:4035` 已加「（r65.4 取代…）」 |
| 3 | 撤销 TOCTOU | r64.5 `:4037`（P6、R2／R4／R6、V2／V4／V6）；ADR `:1118`；r65.0 `:4057` | 启动预检快照只是起点；每个检查点按 generation／expiry 重读重验；任一撤销、过期、字节漂移或读取失败 → closed／`restore_unknown`，不得推进 sequence | `:4037` 含 `P6` 重读批准文件精确字节、`R2`／`R4`／`R6`、`V2`／`V4`／`V6`、`V1 prepared(ctx)`；ADR `:1118` 含「R2／R4／R6、V2／V4／V6 与 `prepared(ctx)` 各重验一次」 |
| 4 | `dsn []byte` 经 `json.Marshal` 变 base64 与「线上形状不变」矛盾 | r65.9 `:4066`；`:4129` 消解；`:4045` 就地改写 | 明确 DSN 唯一 wire 形态：只存在于 `privateRestoreInput` 匿名 `Target`，`Target.DSN` 字段删除、由 `appendJSONString` 直写 JSON 字符串；`restorePlanWire`／`restoreReceiptWire` 保持 `string` 且无 DSN 字段；禁止 base64、`json.RawMessage` 与第二 DSN 字段 | `:4129` 含「禁止对 `[]byte` 直接 `json.Marshal`；禁止 base64；禁止 `json.RawMessage` 承载 DSN；禁止第二个 DSN 字段」；`:4066` 含「删除（不是改类型）」 |
| 5 | 外部回执验签／见证未编码 | r65.2 `:4059`；r65.3 `:4060`；r65.5 `:4062`；ADR `:1118` | 回执 producer 签名、验证方 8 字段签名、见证 head 5 键签名、`producer_key_id`／`verifier_key_id`／`witness_key_id` 三处 key_id 载体、`receipt_sha256`／`prev_record_sha256`／`verified_at`、见证原子推进与验签失败处置齐备；`sequence` 严格大于 `max(baseline_sequence, head.sequence)` | `:4059` 8 字段；`:4060` 14 项 keyset 与 head 5 键；`:4062` `V7`–`V11`；验签失败「不得推进见证 sequence」；缺失／不可达／不可验签／无法原子推进 → `ErrAdmissionUnsupported` 并保持 closed |

## 四、B1–B6 设计级处置（引用 `:4070`）

| 项 | 处置 | 落点 | 边界 |
|---|---|---|---|
| B1 批准记录载体与读取路径 | 批准文件与信任根是两个独立文件、不同只读挂载来源；配置键 `HAN_RECOVERY_RESTORE_APPROVAL_PATH`／`HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH`／`HAN_RECOVERY_RESTORE_TRUST_ROOT_SHA256`；两者以 `O_RDONLY|O_CLOEXEC|O_NOFOLLOW` 打开 regular file、owner-only、fstat 与路径身份一致；approval `1..65536`、trust root `1..1048576`；启动预检构造不可变快照，R2／R4／R6、V2／V4／V6 与 `prepared(ctx)` 重读重验，漂移 → `ErrAdmissionClosed` | `:4070` B1 段；`:4076`；`:4037` | 真实信任根与批准记录不存在；读取路径的 POSIX 行为 NOT RUN |
| B2 `RestoreSourceVerifier`／`InstallationSourceVerifier` 入参扩展 | `RestoreSourceClaim` 是唯一入参；扩展只允许新增 claim 字段或版本化接口；`InstallationSourceVerifier` 若扩展必须 nil → closed；不新增端口、不改 `RuntimeDependencies` | `:4070` B2 段；`:4016` 写集行 | 属实施写集，本轮不改代码 |
| B3 专用 restore DB 角色目标凭据载体 | 只经 `HAN_RECOVERY_RESTORE_TARGET_DSN_FILE` 单文件读取（owner-only、no-follow、绝对规范路径），**不得回退** `HAN_DATABASE_DSN`；`[]byte` 承接、九处清零；DSN 只经私有 stdin；快照只留摘要／身份 | `:4070` B3 段；`:4045`；`:4066` | GC／管道／子进程副本不可清零，登记为残余 |
| B5 POSIX 专用实现文件与 build-tag 边界 | `recovery_restore_host_linux.go`（`linux`）、`recovery_restore_host_other.go`（`!linux` → `ErrAdmissionUnsupported`）、`recovery_restore_exec_linux.go`（`linux`）、`recovery_restore_exec_other.go`（`!linux`，零状态变更）；`!linux` 一律 closed | `:4070` B5 段；`:4007`–`:4010`；`:4013`；`:4017` | 写集扩展只是建议，交 Task-Planner；本轮不改 `tasks.md`、不改卡面、不勾选 |
| B6 平台门前置／负向矩阵 | 门顺序：配置存在性 → 批准／凭据解析 → 路径／SHA → 平台 host gate；Windows 生产 unsupported 退出码 2；负向矩阵至少覆盖非 Linux、ACL 存在、mount 不满足、路径非规范、fd-exec 不可用、loader 不可证明、witness 不可达／验签失败、`restoreLease` 未传参／nil（restore 动作在任何状态变更前 `ErrAdmissionClosed`）、`joinWork` 未覆盖 `enter` 的 teardown 缺口、`Target.DSN` 残留、`restoreReceiptWire` 残留；无 fallback、无 Windows 例外 | `:4070` B6 段；`:4014` | 矩阵的 Linux／POSIX 分支全部 NOT RUN；Windows 侧只有语义层与 fail-closed |

补充：`r65.13` 末尾同时写明「模拟见证边界」——本机独立 witness CLI（Ed25519，私钥不入服务 ENV／工作目录）只是测试替身，其证据必须标注「模拟见证、未真实验证」；真实实现是外部审批系统／HSM，未验证。

## 五、ADR-0031 r09 与索引同步

| 项目 | 落点 | 内容 |
|---|---|---|
| 修订行 | `:1082` | `**修订**: adr-0031-r09` |
| 状态行 | `:1084` | 已接受（2026-11-01 D03 授权）；现行修订 `adr-0031-r09` 未经复评，决策接受级对 r09 未达成，`identity-auth-S4.24a` 及任何实施／验证卡不得启动；r09 不改变「首期不提供生产 restore 工具与生产批准主体」 |
| 第 4 项（原生耐久回执 + 外部锚） | `:1118` | 保留原生耐久回执的不变量；r09 补「独立见证签名」路线：见证 head 5 键签名载荷、验证方 8 字段载荷、`anchor_ref` 五处逐字节相等、`sequence` 严格大于 `max(baseline_sequence, head.sequence)`、`baseline_sequence` 只增不减；验签失败／见证不可达／无法原子推进 → closed／`restore_unknown`；模拟 witness CLI 仅为测试替身、证据标注「模拟见证、未真实验证」，真实实现为外部审批系统／HSM 未验证；新增「r09 撤销与 TOCTOU 同步」句 |
| 门禁分级与映射 | `:1126`；`:1128`；`:1129` | 决策接受级／卡证据级／生产解除级三级不变；ADR-0030 约束 9(b) 映射不变；r06 的两路 r02 接受事实保留为历史，r09 未复评前启动门禁回到 blocked |
| 不变量 6 | `:1147` | 补：`witness_principal_id`／`witness_key_id` 及其 `org_id` 不得等于 root／executor／producer／issuer／reviewer 任一方；见证方与部署方／root 分属不同管理域（部署前置，不是代码判定）；`baseline_sequence` 只增不减，重钉须人的明确批准并留证，降低即 closed；验签失败不得推进见证 `sequence` |
| 约束 8 第 6 条 | `:1166` | 解除条件补 witness endpoint 的 DNS 解析约束、host 精确相等、独立 TLS 信任锚或 SPKI pin、query／fragment 不带凭据、见证独立性、`baseline_sequence` 单调与重钉须人批准 |
| 范围句与门禁 | `:1174`；`:1176` | r09 只同步外部独立见证签名、模拟／真实边界、信任根与端点约束、见证独立性、撤销／TOCTOU 与验签失败处置；不改结论、候选取舍、不变量 1–5、约束 1–7／9 与解除条件实质；登记≠启动不变 |
| 追溯行 | `:1190` | r07 追溯行末补 r09 追溯：基线为 `adr-0031-r07`；`adr-0031-r08` 未产出，按用户 2026-11-02 追加任务跳号使用 `adr-0031-r09`，跳号事实待评审方确认；r09 未经复评 |
| 索引 | `README.md:39` | 现行修订 `adr-0031-r09（未复评；外部独立见证签名；模拟见证未真实验证）`；范围列增加「见证信任根／端点约束／独立性」 |

决策边界：r09 未引入新产品、新依赖或新不变量，只把 design r62–r65 已定的外部锚与撤销判据同步进 ADR 口径；「首期不提供生产 restore 工具与生产批准主体」与不变量 1–5、约束 1–7／9、解除条件实质均未改写；未修改 ADR-0030、ADR-0032 或其他 ADR。

## 六、自检证据（本轮实测，可复算）

### 6.1 三份修订文件终态

| 文件 | 字节 | 行（ReadAllLines／LF） | CR | BOM | SHA256 |
|---|---:|---:|---:|---|---|
| `design.md` | 1,045,042 | 5,464 | 169 | 无 | `D2A6B54B2153858DC6366995C4332A380D9DA0BEDE863BDDC80D51663A3D8BE2` |
| `framework.md` | 272,588 | 1,271 | 1,271 | 无 | `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4` |
| `README.md` | 21,613 | 110 | 2 | 无 | `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2` |
| 本文件 | 追加后终值见同目录 `hashes.txt` | — | 0 | 无 | 同上 |

口径：字节 = `[IO.File]::ReadAllBytes`；行 = `[IO.File]::ReadAllLines`（UTF-8）；SHA-256 对整文件字节直接计算。本文件为最终版本，终值字节数与 SHA256 见同目录 `hashes.txt`。`design.md` 在 12:38:45 后到本报告定稿前复核两次（间隔约 45 s），字节数与 SHA256 均未变化。

### 6.2 r65 相对 r64 的净值

| 版本 | 字节 | 行 | SHA256 |
|---|---:|---:|---|
| r64（输入基线） | 1,005,979 | 5,449 | `FB84EC1E0B078AC99B09A48988F00257B77CA93102298A7DAD5004E0B8871E94` |
| r65（本轮输出） | 1,045,042 | 5,464 | `D2A6B54B2153858DC6366995C4332A380D9DA0BEDE863BDDC80D51663A3D8BE2` |
| 净值 | **+39,063** | **+15** | — |

+15 行全部来自 r65 区块及其紧邻空行；后续追加（并发三次 +57／+103／+173 B，本轮 ADR 表述校正 +69 B）都发生在既有 12 行内，不增行。

### 6.3 冻结向量与 r61 字节（对应第二节引用的「第六节第 3 项」）

- `:4087`（source golden vector）：1,115 B（不含行尾 LF），SHA256 `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab`；与 `design-revision-r04/report.md` 第 106 行记录一致。
- `:4093`（tool golden vector）：2,357 B（不含行尾 LF），SHA256 `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508`；与 `design-revision-r04/report.md` 第 107 行记录一致。
- 两条向量中的 `restore_reviewer`／`restore_approver`／`restore_executor` 三个字面未改；r65.4 的角色封闭枚举与显式映射以这三个字面为输入，不改向量（SEC-F002(b)）。
- r61 段（`:4247`–文件末，1,218 行）内 `r64` 出现 **0** 次、`r65` 出现 **0** 次；`:4423` 已恢复原句（NEW-R04-02）。

### 6.4 golden vector 行号偏移说明（对应第一节「第六节第 4 项」）

用户追加任务所称 `:4072`／`:4078` 是 r64 文件中的行号。r65 在 `:4057`–`:4070` 插入 15 行后，两条向量整体后移至 `:4087`／`:4093`；字节未变。本报告一律以 r65 现行行号引用。

### 6.5 `tasks.md` 与代码侧未实施证据

- `tasks.md:3186` 实测为 `- [ ] identity-auth-S4.24a 接入生产 restore source/target/tool 与批准主体接线；受 ADR-0031 门禁，与锁后端节点独立。`；`tasks.md` 整文件 898,536 B／SHA256 `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`，与本轮开工前一致（本轮未写该文件）。
- `rg restoreReceiptWire cmd` = **3**（均在 `cmd/han-server/recovery_restore.go`），设计目标为 0；未实施。
- `rg Target\.DSN cmd` = **2**，设计目标为 0；未实施。
- `rg NewRuntime\(ctx, deps, restoreLease\)`（`design.md` 全文，按出现次数）= **5**；`BindRestoreLease` = **0**；`restoreLease` = 29 次；`ownedExclusive` = 14 次；`injectedExclusive` = 29 次。代码侧 `internal/recovery/runtime.go:729` 仍是 `func NewRuntime(ctx context.Context, d RuntimeDependencies) (Runtime, error)`，`:768` 仍是单一 `exclusive`，`:1554`／`:1562` 的 `recoveryFlow.acquire` 仍自行 `Acquire`，`:1574`／`:1648` 是其调用点；未实施。

### 6.6 未改文件哈希与工作树状态

| 文件 | 字节 | SHA256 |
|---|---:|---|
| `.add/project/knowledge/tasks/tasks.md` | 898,536 | `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` |
| `review-arch-r04/report.md` | 29,426 | `F2FB13FE3EF58AAF2EA5FCB64B32D7C3F15AEE5CBF99CF8F69986D520E98A6DB` |
| `review-sec-r04/report.md` | 40,794 | `F98B39D19083E7A010680465C40404AEEE23E95C65F48C103602F7758632E898` |
| `design-revision-r04/report.md` | 28,805 | `452603BB6CC65E156F9363A03B877CB92B66287C9D8B51DF1C1A38651870C51F` |

`git status --porcelain` 当前 297 项（17 项 `M`、280 项 `??`）。其中 `??` 包含本轮新建的 `design-revision-r05/` 与并发产生的 `.spec/wip/article-management/review-am-q2-r01/report.md`（12:28:55，非本轮写集）；扣除这两项后为开工前基线 295 项（17 `M`／278 `??`）。17 项 `M` 都是本轮开工前已存在的未提交改动（含 `design.md`、`framework.md`、`README.md`、`tasks.md`）；本轮只写三个允许文件和 r05 目录，`tasks.md` 及其余文件未写，未 `git add`／`commit`／`push`／切分支／`stash`／`reset`／`checkout`。

### 6.7 字段与编号唯一性计数（`design.md` 全文）

| 检索项 | 次数 | 判据 |
|---|---:|---|
| `baseline_sequence` | 24 | 基线字段唯一名 |
| `sequence_floor` | 19 | 仅存于历史／改名叙述与 r62 正文（`:3934`、`:3942`、`:4027`、`:4033`、`:4049`、`:4057`、`:4078`、`:4082`、`:4143`、`:4330`）；建议写集表 `:4000`–`:4019` 内为 **0**；`:3934`／`:4078`／`:4082`／`:4143` 行尾已有改名或取代标记；其余行由 r64.3 的整体改名条款覆盖，未逐行改写（见 8.3）。 |
| `witness_tls_spki_sha256` | 6 | endpoint 信任锚唯一形态 |
| `restoreReceiptWire` | 13 | 回执 wire 唯一点名 |
| `han-restore-receipt/v1` | 7 | 回执版本唯一字面 |
| `han-db-restore/v1` | 6 | 仅 `RestorePlan.Version` 等计划侧使用 |
| `Target\.DSN` | 8 | 全部出现在「删除该字段」或「残留检查」的规则句，不是保留字段 |
| `nonce` / `bundle_digest` | 0 / 0 | 现行设计**无定义**，见 8.3 第 5 项 |

### 6.8 并发改写登记与待评审状态

- 并发改写发生在 r65 初稿落盘之后：`design.md` 的改动始终落在同一 12 行（`:3990`、`:3997`、`:4013`、`:4017`、`:4029`、`:4051`、`:4057`、`:4058`、`:4060`、`:4068`、`:4069`、`:4070`）。相对初稿备份 1,044,640 B 共三笔：+57 B（本报告首次落盘前）、+103 B（12:36:27）、+173 B（12:37:55），行数始终 5,464；以初稿备份与当前文件逐行 diff，差异仍只在这 12 行。改写前备份为 `%TEMP%\design-r65-prefix.bak.md`（SHA256 `2BBBDC8410BEE8681DA3F3B648693EADD6CDA1CEC8856B90222F943EDB72DE61`，1,044,640 B），仅用于 diff。
- 本轮未回退或覆盖该并发改写；它把 lease 方案固定为「`NewRuntime(ctx, deps, restoreLease)` 非可选参数 + `ownedExclusive`／`injectedExclusive` 两互斥字段 + `recoveryFlow.acquire` 只读 `injectedExclusive` + 只关 `ownedExclusive` + 装配体唯一 `Close`」，并在 `:4069` 登记 `joinWork` 未覆盖 `enter` 的实施缺口。
- 该方案改变了 `NewRuntime` 签名。用户 BLOCKER 1 给出的括号替代路线是「把注入点下移到 `recoveryFlow`，不改 `NewRuntime` 签名」；现行文本是否被 R05 接受、是否满足 r60 `:3603` 的先行修订要求，均待 R05 判定，本报告不把 BLOCKER 1 写成已闭合或已通过。
- 本轮对 `:4057`（r65.0）做了一处事实校正（+69 B）：原文写「ADR-0031 …由本轮之外的既有改动形成，本轮不修改 ADR」，与本报告第五节及 `framework.md:1082`／`README.md:39` 的 r09 落点互斥；已改为「现行 `adr-0031-r09`（`decisions/framework.md:1082`；其外部见证与撤销判据由本轮同步，见 `:1118`／`:1147`／`:1166`／`:1190`），本条不主张其评审状态」。该校正只改事实归属，不改任何判据、字段、错误分类或 ADR 内容。
- 本节只登记事实与待判定项，不构成对并发写入者改动的评审结论，也不表示本轮重新实现了这些行。

## 七、影响面与建议写集（只作建议，不改 `tasks.md`）

以下只用于说明设计会落到哪些文件，不是任务卡、不是排期、不是实施授权。实际写集由 Task-Planner 在任务卡上更新，供 R05 评审核对；本轮未改 `tasks.md`、未改任何代码。

| 文件 | 变更 | 设计依据 |
|---|---|---|
| `internal/recovery/runtime.go` | `Runtime` 在 `:768` 的单一 `exclusive` 拆为 `ownedExclusive`／`injectedExclusive` 两个互斥字段；`NewRuntime` 增非可选参数 `restoreLease`（`:729`）；`recoveryFlow.acquire`（`:1554`；调用点 `:1574`／`:1648`）只读 `injectedExclusive`、不再 `Acquire`；`closeObservationLease`／`Stop`（`:1423`–`:1424`）只关 `ownedExclusive` | `:4017`；r65.11 `:4068`；r65.12 `:4069` |
| `cmd/han-server/recovery_runtime.go` | 装配阶段 `anchors.Exclusive.Acquire(ctx)` 恰好一次，把同一 lease 作为 `restoreLease` 传给 `recovery.NewRuntime` 与 `driver.Bind`；assembly 生命周期 `defer Close` 恰好一次；建议 Task-Planner 扩展 `joinWork`／teardown 覆盖 `enter`（当前 `:941`–`:964` 只覆盖 `normal`／`resume`） | r65.11／r65.12；`:4069` |
| `cmd/han-server/recovery_restore.go` | 删除匿名 `Target.DSN` 字段、改 `appendJSONString` 直写；`restoreReceiptWire` 版本字面改 `han-restore-receipt/v1`；保留 `d.prepared(ctx)`（R6） | r65.8 `:4065`；r65.9 `:4066`；r65.5 `:4062` |
| `cmd/han-server/recovery_restore_approval.go`（新增） | 批准文件／信任根读取、owner-only／no-follow／fstat、不可变快照与每检查点重读 | B1；r64.5 `:4037` |
| `cmd/han-server/recovery_restore_target.go`（新增） | `HAN_RECOVERY_RESTORE_TARGET_DSN_FILE` 单文件读取、九处清零、私有 stdin | B3；r64.9 `:4045` |
| `cmd/han-server/recovery_restore_receipt.go`（新增） | producer／verifier 签名、8 字段载荷、V7–V11 | r65.2 `:4059`；r65.5 `:4062` |
| `cmd/han-server/recovery_restore_witness.go`（新增） | endpoint 预检（DNS／网段／host／TLS pin）、head 5 键、`baseline_sequence` 原子推进 | r65.3 `:4060` |
| `cmd/han-server/recovery_restore_host_linux.go`／`_other.go`（新增） | `//go:build linux`／`!linux`；`!linux` 一律 `ErrAdmissionUnsupported` | B5；`:4007`–`:4010`、`:4013` |
| `cmd/han-server/recovery_restore_exec_linux.go`／`_other.go`（新增） | POSIX fd-exec 与 `/proc` 观测；非 Linux 零状态变更 | B5；`:4203` |
| 对应 `*_test.go` | Linux-only 测试一律 `//go:build linux`；Windows 只保留语义层 | 第六／八节 |

影响面提示：`NewRuntime` 新增非可选 `restoreLease` 参数会波及全部调用点与 `recovery.Runtime` 测试替身；`Runtime` 由单一 `exclusive` 拆为两个互斥字段；`joinWork` 当前只覆盖 `normal`／`resume`，`enter` 的 teardown 缺口未实施；`install`／`admission` 的 provider 路径保持原样（r65.12）。

## 八、未闭合项、NOT RUN 边界与停止条件

### 8.1 NOT RUN / unknown（本机不可执行）

| 类别 | 状态 | 原因 |
|---|---|---|
| Linux fd-exec、`flock`、loader 预检（`PT_INTERP`／`dt_needed`／`AT_SECURE`） | NOT RUN／unknown | 本机 Windows，无 WSL 发行版 |
| `/proc/<pid>/status`、`/proc/<pid>/auxv`、maps／cache 观测 | NOT RUN／unknown | 同上 |
| POSIX ACL（`posix_access`）、mount 选项（`nodev`／`nosuid`／`ro`） | NOT RUN／unknown | 同上 |
| witness endpoint DNS 解析、私网／loopback 拒绝、TLS SPKI pin | NOT RUN／unknown | 同上，且无真实 witness |
| 真实 Ed25519 密钥、真实见证签名、真实回执验签、原子推进 | NOT RUN／unknown | 无外部审批系统／HSM，无私钥 |
| 生产 restore 端到端 | NOT RUN／unknown | 生产 restore 继续 closed，`recoveryClosedRestoreDriver` 默认 |

Windows 上可执行的只有解析／语义层：JSON keyset、封闭枚举、字段名、门顺序、错误分类的字面判定。`//go:build linux` 文件在 Windows 被编译排除；Windows 生产路径保持 fail-closed（非 Linux host gate → `ErrAdmissionUnsupported`，退出码 2）；「没有可跑的测试而命令 exit 0」不是 PASS。所有 Linux 专用测试必须显式标 NOT RUN。

### 8.2 生产与治理边界

- 生产 restore 继续 closed；`recoveryClosedRestoreDriver` 仍是默认。
- `enter`／`resume` 不因 r65 解锁。
- ADR-0031 现行 `adr-0031-r09` 未经复评，决策接受级对 r09 未达成；`identity-auth-S4.24a` 不得启动，`tasks.md:3186` 保持 `- [ ]`。
- 本修订不构成约束 8 满足、不构成任务关闭、不构成任何人批准。

### 8.3 残留 unknown（已登记，不隐藏）

1. `person_ref` 自证：同一自然人可用两个不同 `person_ref` 分别登记 reviewer 与 witness，设计只能机核字段不等，不能证明自然人不同。
2. root 可整体替换信任根（含 witness 公钥），设计不能证明「外部见证未被替换」；只能要求见证方与部署方／root 分属不同管理域（部署前置）。
3. DNS 投毒／同管理域冒充：endpoint 约束把可解析地址限制在公网，但无法证明解析结果对应的主机就是登记见证方。
4. GC／管道／子进程 DSN 副本不可清零；DSN 明文在子进程生命周期内存在。
5. `nonce`／`bundle_digest`：用户原任务点名要求，但现行 `design.md` 与 `framework.md` 全文均为 **0** 次出现，**无定义、未验证**；新增会改变 wire 形状，超出「线上形状不变」约束，需先由用户拍板扩展点。
6. `sequence_floor` 仍出现在 r62 正文 `:3934`／`:3942`／`:4027`／`:4033`／`:4049`／`:4057`／`:4078`／`:4082`／`:4143`／`:4330`；`design.md` 的 `sequence_floor` 总出现次数为 19，`baseline_sequence` 为 24。语义由 r64.3 改名条款与相关行尾标记整体取代，未逐行改写，登记为文本残留。
7. F017（`NewRuntime` 签名变更与 `Runtime` 字段拆分的波及面）写集待 Task-Planner；`review-sec-r04` 的 N005（ADR／`tasks.md` 状态冲突）在写集外，未处理。
8. 模拟 witness CLI（本机独立进程、Ed25519、私钥不入服务 ENV／工作目录）只是测试替身；其证据必须标注「模拟见证、未真实验证」。真实实现是外部审批系统／HSM，未验证。
9. `adr-0031-r08` 未产出、按用户追加任务直接使用 r09 的跳号事实，待 R05 评审方确认；`adr-0031-r09` 未经复评，决策接受级未达成。
10. `design.md` 的 lease 相关 12 行在 r65 初稿落盘后被并发改写（+57／+103／+173 B，行数不变；另有本轮对 r65.0 ADR 表述的事实校正 +69 B），本轮按「保留他人改动」未回退；该并发改写未经 R05 评审，不能当作本轮作者已评审或已闭合的内容。
11. `NewRuntime` 签名已改为 `NewRuntime(ctx, deps, restoreLease)`，会波及全部调用点与 `recovery.Runtime` 测试替身；用户 BLOCKER 1 的括号替代路线是「把注入点下移到 `recoveryFlow`、不改签名」，现行路线是否被 R05 接受、是否满足 r60 `:3603` 的先行修订要求，均待判定。
12. `joinWork`（`cmd/han-server/recovery_runtime.go:941`–`:964`）只覆盖 `normal`／`resume`，`enter`／`install` 走 `ErrAdmissionBinding` 且没有现成 teardown 位置；设计要求 assembly 生命周期覆盖全部 restore 取 lease 动作，但代码侧覆盖仍是实施缺口，交 Task-Planner。

### 8.4 停止条件核对

| 停止条件 | 是否触发 | 依据 |
|---|---|---|
| 决策超出 ADR-0031 授权范围（新产品／新依赖／改不变量） | 未触发 | r09 只同步 design r62–r65 已定内容；不变量 1–5、约束 1–7／9、解除条件实质未改 |
| 需要修改写集外文件 | 未触发 | 只改 3 个允许文件＋新建 r05 目录 |
| 无法在不改变线上形状下定义 wire | 未触发 | DSN 走 `appendJSONString` 直写字符串、删除匿名 `Target.DSN` 字段；`restorePlanWire`／`restoreReceiptWire` 保持 `string` 且无 DSN 字段（`:4129`／`:4066`） |

因此本轮未中止；但 8.1–8.3 的全部 unknown 保留，未以任何方式升级为 PASS。

## 九、声明

**本修订未经 R05 独立评审、未实施、未闭合、非真人签字。**

- 不构成 PASS、不构成生产可用、不构成 `identity-auth-S4.24a` 可关闭、不构成 ADR-0031 约束 8 满足、不构成任何人的批准或接纳。
- 未执行任何 Git 写操作（`add`／`commit`／`push`／切分支／`stash`／`reset`／`checkout`）。
- 只修改 `.add/project/knowledge/design-workspace/identity-auth/design.md`、`.add/project/knowledge/decisions/framework.md`、`.add/project/knowledge/decisions/README.md`，并在 `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r05/` 新建本报告与哈希清单。
- r65 相对 r64 的实测：+39,063 B／+15 行，SHA256 见 6.2；两条 golden vector 复算见 6.3／6.4；lease 相关 12 行并发改写与 r65.0 ADR 表述校正见 6.8。
- 本轮为契约文本修订；任何实现、真实见证、Linux／POSIX 验证均未发生。`design.md` 的 lease 相关 12 行在 r65 初稿落盘后被并发改写并由本轮保留；其评审状态、`NewRuntime` 签名路线与 `joinWork`／`enter` 缺口均未闭合。
