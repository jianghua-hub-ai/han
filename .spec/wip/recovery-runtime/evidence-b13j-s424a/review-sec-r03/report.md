# 安全侧独立复审 r03（对象：identity-auth design r63）

评审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`；976149 B；5416 行（`[IO.File]::ReadAllLines(...,UTF8).Count`）；复算 SHA256 `3B8BEA8C35A12F75B36091969DB15B9590FBDD16B0429D59328ABB9FA3BC027C`；`revision: identity-auth-design-ir01-r63`。作者报告：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r03/report.md`；17335 B；复算 SHA256 `DD63583CADF85B657A118C8DBEAC2DDB0105DA68133D917A71B1FFBF1C4A8CAE`。评审基线：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r02/report.md`；57982 B；SHA256 `9AD6C71CEFEEB06724DEC07D9CBF553EB787855BF804EFCC908870FD76CF9705`；基线 verdict `CHANGES REQUIRED`（BLOCKER 7／HIGH 6／LOW 2）。本次范围：只复核 r02 仍开放的 SEC-F001／F002／F003／F004／F005／F006／N001／N002／N003／N004／N006／N007／SEC-F008，并做 r63 回归检查；不重评架构侧、不评实现代码、不改任何被评对象。**结论：`CHANGES REQUIRED`；设计侧仍开放 BLOCKER 5／HIGH 4／LOW 3，另有范围外 BLOCKER 1（N005）与 NOT RUN 1（SEC-F007）。**

---

## 1. 复算、基线与读取范围

### 1.1 对象校验（全部一致，无偏差）

| 对象 | 字节 | 行数 | SHA256 | 与声明 |
|---|---:|---:|---|---|
| `design.md`（r63） | 976149 | 5416 | `3B8BEA8C…3BC027C` | 一致 |
| `design-revision-r03/report.md` | 17335 | — | `DD63583C…1C4A8CAE` | 一致 |
| `review-sec-r02/report.md`（基线） | 57982 | — | `9AD6C71C…76CF9705` | 一致 |

工作区 `main`／HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；`git status --porcelain` 仍为 295 条，未清理、未撤销。行数用 `ReadAllLines` 口径；`Get-Content | Measure-Object -Line` 会因末尾换行口径给出不同数字，本报告不采用该口径。

### 1.2 JCS golden vector 独立复算

复算只依赖 JSON 解析、UTF-16 code unit 键排序、紧凑序列化与 SHA-256，不依赖本仓库代码：

| 位置 | 字节 | 键序违规 | 复算 SHA-256 | 文中声明 | 与 Lead／r02 |
|---|---:|---:|---|---|---|
| `design.md:4039`（source） | 1115 | 0 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 同 | 一致 |
| `design.md:4045`（tool） | 2357 | 0 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 同 | 一致 |

结论：r62.1 的“非 canonical 字节 + 错摘要”缺陷已就地修正，`reviewer` < `revocation` 顺序正确，两条向量可作为验收基线。

### 1.3 实际读取范围

- `design.md:3924`–`:4060`（r63 全段 `:3925`–`:4022`，含 `:4039`／`:4045` 向量与 `:4028`–`:4057` r62.1／r62.2 被引用句）。
- 定向回读 r62／r61／r51 既有 restore 章节：`:3884`、`:1602`、`:1667`、`:4067`、`:4075`、`:4085`、`:4087`、`:4089`、`:4091`、`:4093`–`:4103`、`:4109`、`:4126`–`:4145`、`:4147`–`:4179`、`:4313`–`:4317`。
- `decisions/framework.md:1125`–`:1170`（ADR-0031 r07 状态、不变量 1–6、约束 8）。
- `tasks/tasks.md:3186`–`:3218`（只读）。
- r02 安全报告全文、r03 作者报告全文、`.codex/skills/add-code-review/SKILL.md`、`references/产品安全红线.md`、`guidance/design-doc-writing-style.md`、`guidance/decisions-guide.md`。

---

## 2. 逐条判定总表

| ID | 严重级 | 原 finding（摘要） | r63 落点 | 判定 |
|---|---|---|---|---|
| SEC-F001 | BLOCKER | 批准摘要自证、JCS 不可独立复算 | `:3928`–`:3931`；`:4039`／`:4045`；`:4053`；`:4055` | **已闭合（设计层）** |
| SEC-F002 | BLOCKER | 复核人独立性不可机核、一人两名 | `:3933`–`:3937` | **部分闭合（仍开放）** |
| SEC-F003 | BLOCKER | 撤销 TOCTOU、时钟回拨 | `:3939`–`:3944` | **部分闭合（含回归，仍开放）** |
| SEC-F004 | HIGH | DSN 副本／清零／oracle | `:3970`–`:3973` | **部分闭合（仍开放）** |
| SEC-F005 | BLOCKER | 宿主 loader 边界不完整 | `:3975`–`:3982`；`:4151`；`:4179` | **部分闭合（仍开放）** |
| SEC-F006 | HIGH | fd-exec 身份／权限链 | `:3983`–`:3985`；`:4153`–`:4157`；`:4145` | **部分闭合（仍开放）** |
| SEC-F007 | BLOCKER | NOT RUN 诚信、关闭条件未满足 | `:4020`(3)；作者报告 `:118`／`:132` | **NOT RUN（不是设计文本可闭合项）** |
| SEC-F008 | LOW | 规范化／措辞／示例精度 | `:3930`；`:4045`；`:4057`；`:4028` | 部分闭合（仍开放 LOW） |
| N001 | BLOCKER | 回执验证与审计信任链不闭合 | `:3934`；`:3937`；`:3987`–`:3992` | **未闭合** |
| N002 | HIGH | `audit_sink` 只读打开却要求追加 | `:3992` | **已闭合** |
| N003 | HIGH→**BLOCKER** | `tool.receipt` 混淆策略与实际回执 | `:3987`–`:3988`；`:3884`；`:1602`；`:1667`；`:4057` | **未闭合（建议升级）** |
| N004 | HIGH | artifact 共享 fd offset | `:3968` | **已闭合** |
| N005 | BLOCKER（范围外） | ADR r07 与 tasks 状态冲突 | `framework.md:1129`；`tasks.md:3197`；`design.md:4020`(1) | **未闭合（前置协调）** |
| N006 | HIGH | netns／seccomp 无 pre-exec 机制 | `:3980`；`:3983`–`:3985`；`:4157` | **部分闭合（仍开放）** |
| N007 | LOW | 信任根挂载／ACL 未与工具同级 | `:3982` | **已闭合** |
| N008 | HIGH（新增） | `audit_sink` 父目录／ACL 边界未定义 | `:3992`；`:4179` | **新增／开放** |
| N009 | LOW（新增） | 精度与陈旧交叉引用聚合 | `:3930`；`:4057`；`:4151`(8)；`:4028` | **新增／开放** |
| N010 | LOW（新增） | 错误文本携带 DSN 长度构成派生值 oracle | `:3973` | **新增／开放** |

---

## 3. 已闭合项（附证据，不因“写了签名／清零”而判过）

### SEC-F001 — 已闭合（设计层）

- **外部信任根 + pin**：`:4028` 要求信任根由 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH`／`_SHA256` 固定，且与批准文件位于不同受控只读挂载来源；`:4053` 把缺失／摘要不符归 `ErrAdmissionIncomplete`、验签失败归 `ErrAdmissionBinding`、撤销／过期归 `ErrAdmissionClosed`。
- **无隐式放行**：`:4053` 明文“没有『未签名也接受』的兼容分支；r61 的裸摘要批准文件在 r62 下一律 `ErrAdmissionBinding`”。全段检索未发现 `insecure`／“忽略签名”分支；`:3988` 明确“不接受回执自带公钥、环境变量覆盖或 `insecure` 分支”。
- **canonicalization 可编码性**：`:3930` 固定签名输入剔除 `approval_digest`／`signatures`、递归键序、转义集、整数十进制字符串、数组排序集、禁止 `null`／省略，并禁止用默认 `json.Marshal`；`:3931` 给出不依赖仓库代码的第三方复算步骤，且要求“任何签名、`approval_digest` 或回执摘要都必须由验证方按此规则重算，绝不信任记录内携带的摘要”。`:4048` 明确“若实现键序与向量不一致，必须拒绝该实现，不得反过来修改向量迎合实现”，排除了“多解可绕签名”的反向适配。
- **残余风险诚实登记**：`:4055` 明确“任何能同时改写信任根文件与 pin 的主体（root 或同权限部署方）仍可替换信任根”，登记为“批准抗伪造止于 root／部署方”，并声明生产 restore 继续 closed、不得以“有签名”暗示抗 root 伪造。任务所问“攻击者同时控制 root 与部署方能否同时替换信任根与 pin”的答案是**能**，且文本未否认。
- **攻击面复核**：签名对象不含 `organizations`／`appointments`（`:3930`(6) 排序集未列二者），因此 JCS 对该两数组的多解不会改变任何已签名载荷的字节；其对 `trust_anchor_sha256` 的影响仅在“单条 anchor 的 JCS”层面（`:4095`），不构成签名绕过。相关精度问题归入 SEC-F008。
- 判定依据是可编码性与攻击面，而非“新增了 Ed25519”。

### N002 — 已闭合

`:3992` 固定：读／追加两个独立 fd；追加 `O_WRONLY|O_APPEND|O_CLOEXEC|O_NOFOLLOW`；读 `O_RDONLY|O_CLOEXEC|O_NOFOLLOW`；文件 root 属主、`0600`、执行身份不可写；追加前 `flock(LOCK_EX)` 且覆盖“读末条→算 `prev_record_sha256`→追加→`fsync`”整个区间；每条记录一次 `Write`；短写／`EINTR` 重试仍短写／`fsync` 失败／父目录 `fsync` 失败 → `restore_unknown` 或 closed，禁止返回成功；禁止 `O_RDONLY` 追加、禁止无锁读改写、禁止吞错。r02 的机械矛盾与原子性缺口已消除。文件级之外的目录边界另见 N008。

### N004 — 已闭合

`:3968` 唯一化 offset 规则：`ReadAt(header,0)`（等价 `pread`）；若走 `Read`／`Seek` 路径必须在头读取后与 `defer` 中复位，复位失败 → `ErrAdmissionBinding`；放入 `ExtraFiles` 前必须断言 `Seek(0,io.SeekStart)` 返回 0；禁止依赖调用顺序隐式留在 0。`ArtifactPath` 只作诊断、不得重开。

### N007 — 已闭合

`:3982` 要求信任根与批准文件位于 `ro,nodev,nosuid,noexec` 挂载、属主 root、group／other 无写；父目录逐级 `lstat` 无符号链接且属主 root；检测到 `system.posix_acl_access`／`system.posix_acl_default` 即 `ErrAdmissionClosed`。与 r02 要求的“与批准文件同等严格”一致，且给出可机核的拒绝条件。

---

## 4. 仍开放的 BLOCKER

### SEC-F002 — 复核人独立性（BLOCKER，部分闭合）

r63.2 新增 `organizations`／`appointments`／公钥全局唯一／单 principal 单有效密钥／`affiliations` 受控，方向正确；但仍有三个可机核缺口：

1. **`role_name` 命名空间断裂，`:3936`(3) 不可实施**。`:3934` 的 `principals` exact keyset 是 `principal_id, display_name, kind, appointment_ref, org_id, affiliations, valid_from, valid_until, revocation`——**没有 `role_name`**；而 `:3936`(3) 要求“该记录的 `principal_id`、`org_id`、`role_name`、`valid_from`、`valid_until` 与 principal 侧一致”。principal 侧无该字段，`appointments.role_name`（如 `restore_approver`）与 `principals.kind`（如 `approver`）也不是同一命名空间，且无映射表。该合取项无法按文本机核。
2. **一人两名仍可构造**。`:3936`(1)(2) 只保证 `public_key` 全局唯一、同一 `principal_id` 同时最多一把有效密钥；攻击者可登记两个不同 `principal_id`、两把不同公钥、两个不同 `org_id`（`kind` 可同），指向同一自然人。`:3936`(8) 明确 `person_name`／`display_name` 不参与判定，设计无 enrollment／身份证明，也未把“信任根登记方可将同一人登记为两个 principal”登记为残余风险（`:4020` 的 (1)–(7) 未含此项）。`:3933` 声称“同一个人用两个 `person_name` 仍可伪装”已被修复，但实际只修复了同一 principal 内的别名，跨 principal 的同一人未被处理，且被当作已闭合。
3. **复核签名 key 未绑定 reviewer principal**。`:3934` 的 `reviews` 含 `reviewer_principal_id`、`reviewer_key_id`、`sig`；`:3936`(6) 只要求“复核人签名验证失败 → `ErrAdmissionBinding`”，**没有要求 `reviewer_key_id` 解析到 `keys.principal_id == reviewer_principal_id`**。可让 A 作为 reviewer principal、用 B 的已登记密钥签名：签名通过，但“复核人”身份与签名主体不一致。`:3936`(2) 的单密钥约束只保证 key→principal 唯一，不能替代这条显式交叉比对。

结论：任命／公钥唯一性可机核（部分缓解），但“两两不同 + 角色一致 + 签名者即复核人”的合取仍不可机核，一人两名仍有可行路径且未登记为残余风险。

### SEC-F003 — 撤销 TOCTOU 与唯一调用图（BLOCKER，部分闭合 + 回归）

**已闭合部分**：`:3941` 用不可变快照（`dsnLength`、`dsnSHA256`、`st_dev/st_ino/size/mode/uid/gid/mtime`）替代“逐字节比较 DSN”，并明确“`snapshot.Clear()` 清零 `dsnSHA256` 与身份元数据，不再与『逐字节比较 DSN』矛盾”，r02 指出的矛盾消除；`:3939` 明确高水位只在进程内单调、**不持久化、跨进程或重启后的回拨不保证被检测**，并在 `:4020`(5) 登记残余风险，未过度承诺；`:3942` 的 R6／V5 在 `execute` 返回后、解码前再次 `revalidateApproval`，并写明“若批准、信任根或 DSN 已漂移，必须 `restore_unknown` 或 closed，绝不允许当成成功”，r02 的 in-flight 窗口在本路径上闭合。

**回归／仍开放**：

1. **唯一调用图遗漏 Prepare 的批准重读**。`:3927`(3) 规定 r62.3 的“读取点／时钟段”由 r63.3 取代；`:3942` 声明“编号固定，实施不得增删或重排”，但其固定图只有 P1–P3、R1–R8、V1–V10。**Prepare 路径完全没有节点**，而 r61 `:4313` 仍要求“`Prepare` 用预检得到的同一不可变快照复核，并在 `Journal.Begin` 之前重读文件精确字节与快照比对，字节漂移即 closed”。`:3999` 的 r61 取代清单（1）–（7）**不含**这句。实施者按“不得增删”会跳过 Prepare 重校验，形成 P3 → `Journal.Begin` 之间的撤销窗口。
2. **VerifyRestored 路径缺 `prepared()`**。r62.6.2 `:4109` 仍写“`VerifyRestored` 在 `prepared()` 之后、启动子进程之前调用一次”；`:3942` 声称取代 r62.6.9 的 `VerifyRestored` 八步，而 `:3927`(4) 只取代 r62.6.2 的“claim 形状与句柄说明”（`:3945` 复述同范围），**未取代调用时机**。V1–V10 中无 `prepared()`，幂等／租约前置检查在 verify 路径消失，且与 `:4109` 直接冲突。
3. 若两者任一按字面实施，`revalidateApproval` 与 `prepared()` 的关系不再闭合：`prepared()` 与 `revalidateApproval` 不是同一检查（前者含幂等／租约，后者含批准／信任根／DSN 漂移），不能互相替代。

结论：撤销的“读取点”语义本身已改善，但唯一调用图把两条既有检查移出了覆盖范围，属新引入的回归，不能判闭合。

### SEC-F005 — 宿主 loader 边界（BLOCKER，部分闭合）

1. **`LD_LIBRARY_PATH` 与 ADR-0031 直接冲突**。`design.md:3975` 允许“如需抑制 cache，只能让 `LD_LIBRARY_PATH` 精确等于 `dt_needed` 父目录集合且列入 `tool.environment` 白名单”；`:3976` 的 ENV 白名单键集含 `LD_LIBRARY_PATH`。ADR-0031 r07 `framework.md:1144` 明确“`LD_PRELOAD`/`LD_LIBRARY_PATH`/`LD_AUDIT`/`GLIBC_TUNABLES`、`*_PRELOAD`/`*_OPTIONS` 一类动态加载与注入变量……**出现即 closed**”，约束 8 `:1164` 同义重复。设计单方面放宽已接受的 ADR，不能以“精确等于依赖父目录”自证等价；要么删除该路径、要么先走 ADR 修订。这是设计契约与上游权威的正面冲突，不是精度问题。
2. **`dt_needed`／cache／conf 的父目录未被父目录算法覆盖**。`:4151`(7) 对 `dt_needed.path` 只要求文件 root-owned、无 group/other 写、所在挂载 `ro,nodev,nosuid`、摘要一致；`:4151`(6) 对 cache／conf 文件同。`:4179` 的逐级 `lstat` 算法只写“artifact 与工具的父目录规则相同”，**未覆盖 `dt_needed`、`ld_so_cache_path`、`ld_so_conf_paths`**。可写父目录可在“摘要校验”与“exec／dlopen”之间 rename 替换同 soname 文件，构成“校验了 A、实际加载 B”的 TOCTOU；`:3978` 的 maps 只是检测，且文本自认“不是阻止”。
3. **`LD_LIBRARY_PATH` 目录本身无属性要求**。`:3975` 未要求这些父目录 root-owned、无 group/other 写、同一 `ro` 挂载；glibc 会优先搜索该变量，即使解决了第 1 条的 ADR 冲突，仍存在同 soname 替换窗口。
4. **cache／conf 的“实际读取路径”缺证明字段**。`:3975` 方向正确（无法证明即 `ErrAdmissionUnsupported`），但未给可核字段：解释器编译期默认路径、loader 版本、`--inhibit-cache`／`--library-path` 行为、hwcap 目录的处理结果。实施者只能一律 unsupported（fail-closed，安全方向可接受，但契约不可实施为“支持”）。
5. **陈旧 maps 句未删除**。`:4151`(8) 仍写“exec 前与 exec 后各扫描一次……两次扫描集合必须相等”，与 `:3978` 的“exec 成功后立即 + 读取回执前”不一致；`:3927`(6) 虽声明 maps 部分由 r63.6 取代，但两处时点口径不同会让“校验／执行”时点产生歧义。
6. `/etc/ld.so.preload` 已按 root-owned、no-follow、`ro,nodev,nosuid`、父目录受控、非空即 closed 处理（`:3975`），该项可接受。

### N001 — 回执验证与审计信任链（BLOCKER，未闭合）

1. **`receipt_anchors` schema 不足以承载外部见证**。`:3934` 的 exact keyset 只有 `anchor_ref, producer_principal_id, producer_key_id, public_key, verifier_principal_id, verifier_key_id, audit_sink, sequence_floor`；**没有 witness 端点、witness principal／公钥、协议与版本、TLS／认证材料**。而 `:3990` 要求“从**外部见证**读取当前 `sequence_floor` 与见证签名，要求本次 `sequence` 严格大于两者且见证 head 可验证”，`:3992` 要求“外部见证必须先以 `receipt_anchors.verifier_key_id` 验证验证方签名……请求外部见证原子推进”。无端点与见证公钥，V8／V9 不可实施、不可验证；`:3937` 只说“外部见证系统提供时只校验其签名与登记身份”，未说明见证自身如何登记、公钥放哪、签名算法是什么。
2. **验证方签名不覆盖 anchor 与验证方身份**。`:3990` 的签名载荷是 `{receipt_sha256, sequence, verified_at, result, prev_record_sha256}`；而审计记录 keyset（同段）含 `anchor_ref, verifier_principal_id, verifier_key_id`。这些字段被写入记录、被 `audit_record_ref` 覆盖，却不被验证方签名覆盖；篡改后重算 `audit_record_ref` 与下一条 `prev_record_sha256` 可自洽，外部见证无法据签名分辨记录属于哪个 anchor／哪个验证方。这是“签名了 A、却用 B 做决定”的绑定缺口。
3. **policy `audit_sink` 与 anchor `audit_sink` 无逐字节相等要求**。`:3987` 把 `audit_sink`／`independent_read_path` 定义为 policy 位置字段，`:3934` 的 anchor 也有 `audit_sink`；两处并存但未规定必须相等，且 `independent_read_path` 仅存在于 policy、anchor 无对应字段。可“校验 policy sink、实际写 anchor sink”。
4. `sequence_floor` 的持久化责任改到外部见证、本地仅缓存、回滚风险登记于 `:4020`(5)，方向正确；但受第 1 条限制，仍无法给出可实施的原子推进契约。

### N003 — 三套 `RestoreReceipt` 与重复字段（未闭合，建议由 HIGH 升 BLOCKER）

1. **三套同名不同形的回执定义并存，取代清单不全**：
   - `:3987` 实际回执为 `version, producer_identity, producer_key_id, trust_anchor_ref, trust_anchor_sha256, verifier_identity, verifier_principal_id, signature_alg, signature, issued_at, sequence, audit_record_ref`（`han-restore-receipt/v1`），落地在 `:4011` 的 `cmd/han-server/recovery_restore_receipt.go`。
   - `:3884`（r61 受控 driver 段）仍规定子程序输出的 strict JSON 回执字段为 `version,run_id,backup_ref,deployment_id,installation_id,target_digest,artifact_digest,artifact_bytes,source_approval_digest,restored_schema_digest,restored_at,receipt_digest`。
   - `:1602`（`internal/recovery.RestoreReceipt`）与 `:1667`（r51 “string/int64/time.Time 字段与 §4.2 精确 manifest/receipt 键一一对应”）仍是 run/backup 语义。
   `:3927`(8) 只取代 r62.5／r62.6.3；`:3999` 的 r61 取代清单（1）–（7）未含 `:3884`；r51 的 `:1602`／`:1667` 完全不在任何取代清单内。同一 wire 有两套互斥字段表，实施者无法确定 stdout 必须输出哪一种。
2. **`attestation` 与 `receipt` 重复字段无逐字节相等要求**。`:3988` 的 envelope 同时含 12 字段 `receipt` 与 7 字段 `attestation`，两者重复 `producer_identity, producer_key_id, trust_anchor_ref, sequence, issued_at, signature_alg, signature`；`:3987` 只规定这些字段“必须与 policy 逐字节相等”，未规定 attestation 与 receipt 之间相等。签名覆盖两者，但“验签用 receipt、取值用 attestation”（尤其在 V8 判定 `sequence` 单调时）没有唯一取值来源；若二者不一致，验证方应先拒绝而非任选其一。
3. **陈旧指针**：`:4057` 仍写“`tool.receipt` 的字段与验签见 r62.5”，而 `:3992` 声明 r63.8 取代 r62.5 中冲突句。读者按 `:4057` 会读到已被取代的策略／回执混用文本。

判定说明：r02 指出的“策略与实际回执混用”在概念上已被 `:3987` 拆分（该点已改善），但 wire 字段表冲突与重复字段无相等要求使“执行时用哪份字段”仍不确定，属执行绑定缺口，故建议按 BLOCKER 处理。

### N005 — ADR r07 与 tasks 状态冲突（范围外 BLOCKER，未闭合）

`framework.md:1129` 明确“本 ADR 现行修订已是 `adr-0031-r07`，r07 尚未复评……在 r07 被架构与安全两路接受之前，`identity-auth-S4.24a` 及任何实施/验证卡不得启动（启动门禁回到 blocked）”。`tasks.md:3197` 则写“`adr-0031-r07`……已于 2026-11-02 被两路独立复评接受……决策接受级对 r07 成立，故本卡启动门禁解除”。两处权威直接冲突；`tasks.md:3197` 同时又称“r61 设计修订已被两路 r01 判 CHANGES REQUIRED……本卡仍不可进入实施”，`tasks.md:3217` 实施状态为 BLOCKED。`design.md:4020`(1) 只登记该冲突，未裁定。本轮不改 ADR／tasks，判定保持“未闭合（前置协调）”，不计入设计侧计数，但在对账完成前不得以任一方主张启动。

---

## 5. 仍开放的 HIGH／LOW

### SEC-F004 — DSN wire、清零点与错误脱敏（HIGH，部分闭合）

**已改善**：`:3970` 禁止用 `net/url` 解析带密码的 DSN、禁止默认 `encoding/json` 把 DSN 解到不可变 `string`，要求自定义流式字节解析器直接写 `[]byte`；`:3971`(3) 补齐 `revalidateApproval` 读缓冲清零、(4) 百分号解码缓冲清零；`:3973` 禁止错误携带 SHA-256 前缀／任何摘要／任何可离线比对指纹，并给出 argv／ENV／日志／Trace／Journal／审计／stdout／stderr／panic 的 canary 负向测试。r02 的三项具体缺口（`revalidateApproval` 副本、百分号解码缓冲、`net/url` 字符串、摘要 oracle）均已处理。

**仍开放**：

1. **`privateRestoreInput` 的输出缓冲未列入清零点**。`:3971`(5) 只写“`privateRestoreInput` 的**输入**缓冲，在写入子进程 stdin 后清零”；但该函数的返回值才是含 JSON 转义 DSN 的 wire 载荷，(5)(6) 都没有覆盖它。(6) 是“`runChild` 在管道写入完成后清零**本地副本**”，若 `execute` 持有该返回值，其副本不在清单内。
2. **旧清单中 `execute` 的 defer 被删除且无等价项**。r62.4 `:4085`(1) 原为“`execute` 在 `privateRestoreInput` 返回后立即 `defer clear(input)`，覆盖成功与所有错误路径”；`design.md:3927`(5) 以 r63.5 取代该清单后，九处清零点无对应条目。若 `privateRestoreInput` 成功、`runChild` 之前出错（如 ENV 校验失败、`execRestoreTool` 提前返回），输出缓冲可能不被清零。
3. `:3971` 结尾的“任何 `[]byte` 副本、临时 `string` 转换、错误包装或 debug 输出都不得延长明文生命周期”是原则句，不能替代可核时点；验收无法据此判定“未实现”。
4. 六类副本覆盖情况：argv／ENV／日志／Trace／Journal／审计／stdout／stderr／panic 均由 `:3973` 覆盖；**管道内核缓冲、GC 复制、core dump、调试器**由 `:4087` 诚实登记为“只保证本进程内可见副本按上列时点清零，不保证 DSN 绝不泄露”，并列出 `RLIMIT_CORE=0`／`PR_SET_DUMPABLE=0`／`MADV_DONTDUMP` 为 unknown。该项不存在过度承诺，可接受。

### SEC-F006 — fd-exec 执行身份／权限链（HIGH，部分闭合）

**已闭合部分**：`:3983` 明确 Go `os/exec.SysProcAttr` 无 fork 后／exec 前钩子，`CapBnd` 清零、`PR_SET_NO_NEW_PRIVS`、`setns`、seccomp 一律 **unsupported**，需要时 `ErrAdmissionUnsupported` 并保持 closed，且“不得实现成先 exec 再补设置，也不得声称已达成隔离”；`:3985` 固定 Go 能保证的部分：专用非 root uid/gid、`Groups` 空切片、`NoSetGroups=false`、`Pdeathsig`／`Noctty`／`Setpgid`、fd 3／4、`Path="/proc/self/fd/3"`，并固定“清除 supplementary groups → `setgid` → `setuid`，由 `Credential` 在 child `execve` 前生效”；`:4145` 复述 root-owned `0755`、禁 setuid／setgid／file capabilities、禁 `execveat(AT_EMPTY_PATH)`、`/proc` 不可用即 unsupported 且无 fallback。r02 要求的降权顺序、`Args[0]` 语义（`:4159` 固定 `Args` 恰为 `{executable_path, verb}`，工具合同禁止用 `argv[0]` 做资源定位或安全决策）均已明确。

**仍开放**：

1. **旧能力断言未删除，仍在正文可读位置**。`:4153` 仍以规范语气要求 `mode` 只允许 `netns`、`netns_inode` 必须等于 `stat(netns_path).st_ino`；`:4155` 仍要求 exec 后 `CapBnd=CapAmb=0`、`NoNewPrivs=1`；`:4157` 仍写“netns 内只允许到 `target_addr` 的连接”“seccomp 只允许 `socket` 的 `AF_INET`…”。`design.md:3927`(6)(7) 只做“由 r63.6／r63.7 取代”的元声明，`:3983` 要求“删除 r62 中任何『已通过……达成隔离』的暗示”，但文本并未删除。与 `:4039`／`:4045` 向量用小括号标注“r63.1 修订”的做法相比，`:4153`–`:4157` 无任何 inline 取代标记，实施者按顺序阅读会把 unsupported 能力当成待实现要求。
2. **`NoNewPrivs` 由拒绝条件降级为观测值，未登记为契约放宽**。r62 `:4155` 是“任一不符 → `ErrAdmissionClosed`”；`:3985` 改为“`NoNewPrivs` 只能作为事后观测值登记为 unknown／NOT RUN，不能作为准入前提”。这在效果上把“NNP=0 即拒绝”变成“NNP=0 仍可执行”，是对既有契约的放宽（并非 closed→open 的生产解锁，因为生产仍 closed），但 `:4020`(6) 只写“unsupported／不声称已隔离”，未登记“准入拒绝条件被移除”。
3. **保留的 bounding set 未被约束**。`:3985` 要求工具文件无 setuid／setgid／file capabilities，但未要求“工具及其任何子进程不得 exec 具有 setuid 位或 `security.capability` 的程序”。在 `CapBnd` 未清零且无 NNP 的前提下，工具若 exec 此类程序即可重新获得权限；该残余风险未在 `:4020` 登记。

### N006 — 网络模式（HIGH，部分闭合）

`:3980` 收窄为只允许 `none`／`self_attested`，`netns` 仅作 canonical vector 的语法 fixture、运行时 `ErrAdmissionUnsupported`，并明确“本设计**不声称**已建立 netns、setns 或任何网络隔离”“`self_attested` 时……事后审计只核对声明，不阻止连接”“不得用『已隔离』措辞掩盖仅自证”。`netns_inode`／`netns_path` 在 `none`／`self_attested` 下必须为空串，收窄自洽。

仍开放：**`none`（不联网）无任何执行机制**。在 seccomp／netns／防火墙均 unsupported 的前提下，`none` 只能靠工具合同自证，与 `self_attested` 在可验证性上没有区别；`:3980` 却把 `none` 标注为“不联网”，且未把它并入“自证／仅审计”的措辞。ADR-0031 `framework.md:1164` 的解除条件要求“目标连接外无网络”，本设计明确无法提供，故生产保持 closed 是正确的，但 `none` 的枚举值会被误读为已满足该条。建议：把 `none` 定义为 `self_attested` 的别名，或声明 `none` 同样不可机核并登记残余风险。

（上句 `framework.md` 前的字符为笔误，正确引用为 `framework.md:1164`。）

### N008 — `audit_sink` 父目录与 ACL（HIGH，新增）

`:3992` 只定义审计文件本身：root 属主、`0600`、执行身份不可写、`O_NOFOLLOW`、双 fd、`flock`、单次 `Write`、`fsync` 文件与父目录；`:3992` 末尾另要求 `independent_read_path` 与 `audit_sink` 不同文件、不同挂载来源。全文检索未发现对 `audit_sink` **父目录**的 `lstat`／属主／group-other 写／挂载受控要求；`:4179` 的父目录算法只覆盖 artifact 与工具，`:3982` 只覆盖信任根与批准文件。缺口后果：

- 父目录可写时，攻击者可在两次追加之间把审计文件 rename 换成旧副本（回滚），或换成另一 root-owned 文件（重定向）；`flock` 锁的是 inode，换文件后新 fd 在另一 inode 上，单写者保证失效。
- 目录级 ACL 未禁止，与 `:3982` 对信任根侧“检测到 POSIX ACL 即 closed”的最简策略不一致。

建议：把 `:4179` 的父目录算法扩展到 `audit_sink` 与 `independent_read_path`，并要求二者父目录 root 属主、无 group/other 写、位于受控 `ro` 或明确受控挂载、拒绝 POSIX ACL；同时对“审计文件被替换／截断”给出检测（如记录文件身份 `st_dev`／`st_ino` 并在锁内复核）。

### SEC-F008 — 规范化／措辞／示例精度（LOW，部分闭合）

1. **数组排序清单不全**。`:3930`(6) 的有限清单未列 `organizations`、`appointments`，而 `:3934` 末句要求二者“按各自键升序去重后参与 JCS”。同一文档两种读法。当前签名对象（批准记录）不含这两数组，故不影响 SEC-F001 的签名字节，但仍应补齐清单。
2. **tool 向量使用运行时拒绝的 `netns`**。`:4045` 含 `"network":{"mode":"netns",...}`。`:3980` 已声明 netns 仅作“canonical vector 的语法 fixture”，因此该项**已在 r63 得到解释**（r02 时不存在该句），不构成缺陷；但向量本身无 inline 标注，建议加一行“仅 canonical fixture，不通过运行时网络判定”，避免被当作可批准记录示例复制。
3. **陈旧交叉引用**：`:4057` 的“见 r62.5”与 `:3992` 的“r63.8 取代 r62.5”冲突（详见 N003）。
4. **信任根长度上限**。`:4028` 仍为 `1..65536`；r63.2 新增 `organizations`／`appointments`／`keys`／`principals`／`reviews` 后，多组织部署可能超限。超限的失败分类是 `ErrAdmissionIncomplete`（fail-closed，安全方向可接受），但未说明“超出即 closed”的判定位置，建议明确或评估上限。

### N009 — 精度与陈旧句聚合（LOW，新增）

- `:4089` 仍写“错误只允许携带分类、长度与 SHA-256 前 8 字节十六进制”，与 `:3973` 的“不得携带 SHA-256 前缀、任何摘要”直接冲突。`design.md:3927`(5) 已声明该错误文本由 r63.5 取代，但旧句无 inline 标记，实施者可能按旧句泄漏 oracle。建议删除或就地标注。
- `:4151`(8) 的“exec 前与 exec 后各扫描一次、两次集合相等”与 `:3978` 的“exec 成功后立即 + 读取回执前”时点不同（详见 SEC-F005 第 5 点）。
- `:4151`(6) 的 `ld_so_cache_path` 与 `:3975` 的“解释器实际读取路径”未给同一套证明字段（详见 SEC-F005 第 4 点）。
- `:4020` 的未闭合项清单未含 SEC-F002 的“同人两 principal”与 SEC-F004 的输出缓冲，读者会误以为这些已闭合。

### N010 — 错误文本中的 DSN 长度（LOW，新增）

`:3973` 规定错误“只允许携带分类与 DSN 长度”，示例 `restore target DSN rejected: class=binding length=137`；同段又要求“若实现需要关联多次错误，只能使用进程内随机关联 ID，**不得使用 DSN 的任何派生值**”。DSN 长度本身即 DSN 的派生值，且会通过日志／stderr 暴露口令长度。威胁有限（需要错误可见性），但与同段原则冲突，且未在 `:4087` 的残余风险段登记。建议改为固定桶（如 `<128`／`<256`／`>=256`）或仅保留分类，或将长度列入已登记残余风险。

---

## 6. 额外必查项（安全侧独立发现）

| 必查项 | 结论 | 证据 |
|---|---|---|
| 新增配置键是否引入秘密泄漏面 | **未新增秘密**。四个新键在 r62.6.7 `:4135` 已逐一列举：`APPROVAL_PATH`、`TARGET_DSN_FILE`、`APPROVAL_TRUST_ROOT_PATH`、`APPROVAL_TRUST_ROOT_SHA256`；r63 未再加键，`:4015` 的“四个新键”指向该处，可核。这些是路径与摘要，不是凭据；同 uid 可读 `/proc/<pid>/environ` 得到路径，但不泄漏 DSN 或私钥。错误侧 `:3973` 只带分类与长度，不回声路径。 | `:4015`、`:4135`、`:3970`–`:3973` |
| 信任根文件本身的权限／属主／挂载 | **已与批准文件同级**：`ro,nodev,nosuid,noexec`、root 属主、group/other 无写、父目录逐级 `lstat`、禁 POSIX ACL。 | `:3982`（N007 闭合） |
| 回执 `audit_sink` 追加 + fsync 的截断／重放 | 文件级已闭合（双 fd、`O_APPEND`、`flock`、`0600`、单次 `Write`、文件与父目录 `fsync`）；**父目录与 ACL 未闭合（N008）**；**外部见证 schema 不足（N001）**，`sequence_floor` 的服务端原子推进无法实施。 | `:3992`、`:3990`、`:3934`、`:4179` |
| `sequence_floor` 持久化与回滚语义 | 责任方改为外部见证、本地只作缓存、回滚风险登记于 `:4020`(5)，方向正确；但 `receipt_anchors` 无 witness 端点／公钥／协议字段，权威推进不可实施。 | `:3990`、`:3992`、`:4020`(5) |
| 是否引入第二 authority／fallback／隐式默认／closed→open | **未新增第二 authority、未新增 fallback、未新增隐式默认**；`:3997` 固定唯一 `Acquire` 后传 lease，`Bind` 不再收 provider；`:4020`(4) 声明生产仍 closed。**但** `NoNewPrivs` 由拒绝条件降为观测值属契约放宽（未登记，见 SEC-F006 第 2 点）；`network.mode=none` 的“不联网”为无法机核的声明（N006）。 | `:3997`、`:3985`、`:3980`、`:4020`(4) |
| 是否把生产批准主体或生产工具事实上解锁 | **设计文本未解锁**：`:4022` 明确“不构成生产 restore 可用、不构成 `enter`／`resume` 解锁、不构成约束 8 满足、不构成 S4.24a 可关闭”。**但** `tasks.md:3197` 声称启动门禁已解除，与 `framework.md:1129` 冲突（N005），对账前不得据任何一方主张已启动。 | `:4022`、`framework.md:1129`、`tasks.md:3197` |

---

## 7. Verdict 与计数

**Verdict：`CHANGES REQUIRED`**（实施与 `identity-auth-S4.24a` 关闭仍 BLOCKED；生产 restore 仍 closed）。

**设计侧计数**

- **BLOCKER 5**：SEC-F002、SEC-F003（含回归）、SEC-F005、N001、N003（由 HIGH 升级）。
- **HIGH 4**：SEC-F004、SEC-F006、N006、N008（新增）。
- **LOW 3**：SEC-F008、N009（新增）、N010（新增）。

**设计侧已闭合 4**：SEC-F001、N002、N004、N007。

**设计侧之外**

- **BLOCKER（范围外）1**：N005 —— ADR-0031 r07 状态与 `tasks.md:3197` 冲突，需 ADR 维护者与 Task-Planner 对账；不计入设计侧计数，但对账完成前阻断启动。
- **NOT RUN 1**：SEC-F007 —— 关闭条件 (4)(5) 未满足；skip 与无匹配 `exit 0` 不等于 PASS，不得据此关闭 S4.24a。作者报告 `:118`／`:132` 与 `design.md:4020`(3) 的诚实性维持，未把 NOT RUN 写成 PASS，也未暗示可关闭。

**仍阻断实施的 finding（一句话）**

- `SEC-F002`：`appointments.role_name` 与 `principals` 无对应字段、`reviewer_key_id` 未绑定 `reviewer_principal_id`，且同一自然人可登记为两个 principal 却未登记为残余风险。
- `SEC-F003`：唯一调用图遗漏 Prepare 的字节重读与 VerifyRestored 的 `prepared()`，与 r61 `:4313`、r62.6.2 `:4109` 冲突，撤销窗口与幂等／租约检查回退。
- `SEC-F005`：`LD_LIBRARY_PATH` 与 ADR-0031 `:1144`／`:1164`“出现即 closed”正面冲突，且 `dt_needed`／cache／conf 父目录不在 `:4179` 算法内，存在“校验 A、加载 B”窗口。
- `N001`：`receipt_anchors` 无 witness 端点／公钥／协议字段，验证方签名不覆盖 `anchor_ref` 与验证方身份，policy 与 anchor 的 `audit_sink` 无相等要求。
- `N003`：三套互斥的 `RestoreReceipt` wire 字段表并存且取代清单不全，`attestation` 与 `receipt` 重复字段无逐字节相等要求。
- `SEC-F004`：九处清零漏 `privateRestoreInput` 的输出缓冲与 `execute` 持有的副本，旧 `defer clear(input)` 被取代后无等价项。
- `SEC-F006`：r62 的 `netns`／`CapBnd`／`NoNewPrivs`／seccomp 断言仍留在 `:4153`–`:4157` 且无 inline 取代标记；NNP 由拒绝条件降为观测值未登记；bounding set 保留未约束子进程 exec。
- `N006`：`network.mode=none` 标注“不联网”但无 setns／seccomp／防火墙机制，不可机核。
- `N008`：`audit_sink` 的父目录、属主、group/other 写与 ACL 边界未定义，可 rename 回滚／重定向并使 `flock` 失效。
- `SEC-F008`／`N009`／`N010`：JCS 数组清单漏 `organizations`／`appointments`，`:4057`／`:4089`／`:4151` 陈旧句，信任根 64 KiB 上限，错误文本携带 DSN 长度构成派生值 oracle。
- `N005`（范围外）：ADR r07 blocked 与 `tasks.md`“门禁解除”冲突，需前置对账。

---

## 8. 未验证边界

- 未在 Linux 上运行任何实测：`PT_INTERP`／`DT_NEEDED` 解析、`ld.so.cache`／`ld.so.conf` 实际读取路径、`/proc/self/maps` 时点、fd-exec、`Credential` 降权顺序、`AT_SECURE`、`/proc/<pid>/status`／`auxv`、mountinfo、ACL xattr、`flock`、父目录 `lstat` 全部为设计文本核对，结论不代表实现正确。
- 外部见证、真实信任根、Ed25519 密钥、组织／任命／复核记录、两条批准记录、审计 sink、生产工具均不存在，回执链与 `sequence_floor` 原子推进无法端到端验证。
- 未评审实现代码；未跑 `go build`／`go vet`／`go test`／lint；未验证 `SysProcAttr` 与 Go 版本行为（`CapBnd`／NNP／setns／seccomp 的 unsupported 结论基于标准库无 pre-exec 钩子的事实）。
- 未验证 JCS 实现在 Go 侧是否与 Python 复算逐字节一致（仅验证了向量本身合法且摘要正确）。
- 未复核架构侧 r02/r03 全文；只读与安全结论直接相关的段落。
- `tasks.md` 与 ADR 状态冲突只登记、未裁定；本轮无权更改。

---

## 9. 声明

本报告是安全侧独立复审结论，不是作者自评、不是人的接纳、不是生产批准、不是 `S4.24a` 关闭证据。评审期间未修改被评对象 `design.md`、未修改代码、未修改 `tasks.md`、未修改 ADR、未修改 `.add/**`；未修改 `review-sec-r01/`、`review-sec-r02/`、`review-arch-r01/`、`design-revision-r01/`、`design-revision-r02/`、`design-revision-r03/`；未做 Git 写、删除、移动、归档或清理；工作区原有 295 条未提交改动原样保留。本次只新建本报告一个文件。报告内只使用仓库相对路径。