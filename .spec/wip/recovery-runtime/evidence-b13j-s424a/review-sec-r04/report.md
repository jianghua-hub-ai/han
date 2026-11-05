# 安全侧独立复审报告 r04（identity-auth design r64）

## 1 复算与读取范围

**评审对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 实测 **1005979 字节 / 5449 行 / SHA256 `FB84EC1E0B078AC99B09A48988F00257B77CA93102298A7DAD5004E0B8871E94`**，`revision: identity-auth-design-ir01-r64`，r64 增量区 `:4025`–`:4055`，两条 golden vector 在 `:4072`／`:4078`。与 Lead 给的锚点一致，无偏差。
- 作者报告（只用于定位，不作为证据）：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r04/report.md`，28805 字节 / SHA256 `452603BB6CC65E156F9363A03B877CB92B66287C9D8B51DF1C1A38651870C51F`。

**评审基线**：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r03/report.md`，39113 字节 / SHA256 `8CB71CA2997F552FC841C5D3DD941AA3AF81B5873ABD05AEEDCBA1F78C5C747F`，verdict `CHANGES REQUIRED`；开放 BLOCKER 5（SEC-F002／SEC-F003／SEC-F005／N001／N003）、HIGH 4（SEC-F004／SEC-F006／N006／N008）、LOW 3（SEC-F008／N009／N010），范围外 BLOCKER 1（N005），NOT RUN 1（SEC-F007）。

**本次范围**：只判 r03 开放项在 r64 的落点、r64 引入的回归，以及安全侧独立发现；不重判 r03 已闭合项，除其被 r64 改动处。

**实际读取**：`.codex/skills/add-code-review/SKILL.md`（设计对象分支）、`.codex/skills/add-code-review/references/产品安全红线.md`、`.codex/guidance/design-doc-writing-style.md`、`.codex/guidance/decisions-guide.md`；`design.md:3860`–`:4260` 全文，及点名区段 `:1602`–`:1718`、`:3884`、`:3970`–`:3990`、`:4012`、`:4015`、`:4061`–`:4128`、`:4142`、`:4184`–`:4190`、`:4406`–`:4410`；`.add/project/knowledge/decisions/framework.md:1111`–`:1180`（ADR-0031 r07）；`cmd/han-server/recovery_restore.go`（`:36`–`:82`、`:247`–`:276`、`:318`–`:397`）；`internal/recovery/runtime.go:729`／`:768`；`tasks.md:3186`–`:3197`；`review-sec-r03/report.md` 全文。

**golden vector 独立复算（自写脚本，不抄摘要）**

```text
python（cwd = 仓库根）:
import json,hashlib
p=r".add/project/knowledge/design-workspace/identity-auth/design.md"
raw=open(p,"rb").read(); lines=raw.split(b"\n")
def lb(n):
    b=lines[n-1]
    return b[:-1] if b.endswith(b"\r") else b
def an(n,label):
    b=lb(n); h=hashlib.sha256(b).hexdigest(); viol=[]
    def hook(pairs):
        ks=[k for k,_ in pairs]
        for i in range(1,len(ks)):
            if not (ks[i-1]<ks[i]): viol.append((ks[i-1],ks[i]))
        return dict(pairs)
    o=json.loads(b.decode("utf-8"),object_pairs_hook=hook)
    c=json.dumps(o,ensure_ascii=False,separators=(",",":"),sort_keys=True).encode()
    print(label,"bytes=",len(b),"sha256=",h,"key_order_violations=",len(viol),"canon_reserialize_equal=",c==b)
an(4072,"source")
an(4078,"tool")
```

真实输出：

```text
source bytes= 1115 sha256= de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab key_order_violations= 0 canon_reserialize_equal= True
tool bytes= 2357 sha256= 8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508 key_order_violations= 0 canon_reserialize_equal= True
```

判定：两条向量字节、摘要、递归键序（相邻键严格递增）、JCS 重序列化相等四项均成立；与 r64.3、r64.11 的声明、Lead 复算、r03 复算一致。**向量本身无缺陷**；缺陷在向量与 r64 角色约束的互斥（见 SEC-F002）。

**其他锚点实测**：`tasks.md`（`.add/project/knowledge/tasks/tasks.md`）3289 行 / SHA256 `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`，`:3186` 仍为 `- [ ] identity-auth-S4.24a …`，`:3197` 仍称 r07 门禁已解除。`cmd/han-server/recovery_restore.go:71` 仍为 `privateRestoreInput(p recovery.RestorePlan, dsn string) ([]byte, error)`、`:78` `input.Target.DSN = dsn`、`:79` `return json.Marshal(input)`、`:346` 已有 `defer clear(input)`、`:351` 已有 `defer clear(raw)`、`:352` 在 `tool.Run` 之后调用 `d.prepared(ctx)`、`:392` `d.dsn = ""`；`restorePlanWire`（`:36`–`:50`）无 DSN 字段，DSN 在匿名结构 `Target.DSN`（`:74`–`:76`）；另有 `restoreReceiptWire`（`:52`–`:65`，含 `receipt_digest,omitempty`）。`internal/recovery/runtime.go:729` `NewRuntime(ctx, d RuntimeDependencies)`、`:768` `exclusive ExclusiveLease` 与 r64.1 引用一致。

## 2 逐条判定总表

| ID | r03 级别 | r64 落点 | 本次判定 |
|---|---|---|---|
| SEC-F002 复核人独立性／一人两名 | BLOCKER | `:4035`（r64.4）、`:3934`、`:3936` | **部分闭合（仍开放）** |
| SEC-F003 撤销 TOCTOU／调用图 | BLOCKER | `:4037`（r64.5）、`:3942`、`:3990` | **部分闭合（仍开放）** |
| SEC-F005 宿主 loader 边界 | BLOCKER | `:4039`（r64.6）、`:3975`–`:3978`、`:4184` | **已闭合（设计层）** |
| N001 见证协议与权威推进 | BLOCKER | `:4033`（r64.3）、`:4031`、`:3990` | **部分闭合（仍开放）** |
| N003 回执 wire 唯一化 | HIGH（r03 建议升 BLOCKER） | `:4043`（r64.8）、`:3884`、`:1602`、`:1667`、`:1718` | **已闭合** |
| SEC-F004 DSN 明文清零点 | HIGH | `:4045`（r64.9）、`:3970`–`:3971`、`:4114`–`:4120` | **已闭合（LOW 精度另列）** |
| SEC-F006 能力断言／放宽登记 | HIGH | `:4041`（r64.7）、`:4027`（r64.0）、`:4188` | **未闭合** |
| N006 `none`／`self_attested` | HIGH | `:4041`、`:3980` | **已闭合** |
| N008 审计 sink 边界／抗 rename | HIGH | `:4047`（r64.10） | **已闭合** |
| SEC-F008／N009／N010 | LOW | `:4049`（r64.11）、`:3930`、`:4081`、`:4090`、`:4122`、`:4184` | **已闭合** |
| N011 witness_endpoint 独立性（新增） | — | `:4033`、`:4053` | **新增 HIGH（开放）** |
| N012 回执版本与旧类型漂移（新增） | — | `:1718`、`:4043`、代码 `:52` | **新增 LOW（开放）** |
| N013 DSN 类型与字段命名（新增） | — | `:4045` vs `:4114`／`:4116` | **新增 LOW（开放）** |
| N014 `static`／`dynamic` 措辞自否（新增） | — | `:4039` | **新增 LOW（开放）** |
| N015 R 图与现有代码节点差（新增） | — | `:4037` vs 代码 `:352` | **新增 LOW（开放）** |
| N016 旧字段名残留（新增） | — | `:4012` | **新增 LOW（开放）** |
| N005 ADR／tasks 状态冲突（范围外） | BLOCKER（范围外） | `framework.md:1129`、`tasks.md:3197`、`:4053(8)` | **未闭合（不计入本报告计数）** |
| SEC-F007 NOT RUN 诚实性 | NOT RUN | `:4053(7)`、`:4164`、`:4250`、`:4055` | **保持诚实，不构成通过** |

## 3 已闭合项证据

**SEC-F005（BLOCKER）→ 已闭合（设计层）**
- `:4039` 整句删除 r63.6 的「`LD_LIBRARY_PATH` 精确等于 dt_needed 父目录集合」及其替代路径，明写「不保留『登记后可放行』的例外」「任何 `LD_*`（含 `LD_LIBRARY_PATH`）… 出现即 `ErrAdmissionClosed`」「设计不得放宽」，并援引 ADR-0031 `:1117`／`:1144`／`:1164`。全文 `LD_LIBRARY_PATH` 仅 `:3975`／`:3976`／`:4039`／`:4408`，均为禁止或取代标记。
- 可机核判据齐备：`static` 要求无 `PT_INTERP` 且 `dt_needed` 空；`dynamic` 要求「无任何 `LD_*`」＋`pt_interp_path` 来自 `PT_INTERP`＋`ld_so_cache_path`／`ld_so_conf_paths` 闭包摘要逐项相等＋解释器自身 `DT_NEEDED` 并入，无法证明即 `ErrAdmissionUnsupported`，且禁止猜默认路径、禁止 `--inhibit-cache`（argv 恰两项）。
- 父目录算法覆盖 `pt_interp_path`、每个 `dt_needed.path`、`ld_so_cache_path`、每个 `ld_so_conf_paths.path`、`/etc/ld.so.preload`、工具与 artifact；`dt_needed` 挂载要求 `ro,nodev,nosuid` 且无 `noexec`。maps 时点统一为「exec 成功后一次 + 读回执前一次」（`:3978`），并明确取代 `:4184(8)` 的「exec 前与 exec 后」。cache 证明字段唯一为 `ld_so_cache_path` + `ld_so_cache_sha256` + 父目录／挂载（conf 逐项 `path` + `sha256`），并声明两套口径并存即未实现。
- 残余（不否定闭合）：`:4039` 同段先称 `static` 为「唯一完整可绑定形态」又允许 `dynamic`（N014）；`PATH` 相对 r61.5 重新允许属放宽且未登记（SEC-F006）。

**SEC-F004（HIGH）→ 已闭合，附 LOW 精度项**
- 九处清零点为代码级符号且都能落到可清零的 `[]byte`：`input`（`recovery_restore.go:342`，`execute` 内 `defer clear(input)`，代码 `:346` 已存在）、`execute` 引用、`runChild` 写后本地副本、`raw`（`:351`，已有 `defer clear(raw)`）、`revalidateApproval` 读缓冲、百分号解码缓冲、`snapshot.dsnSHA256` 与身份元数据、子进程解析器缓冲、driver 关闭（`:392`）。
- 类型与编码路径可行：`:4045` 要求 `privateRestoreInput` 参数／内部字段／`d.dsn` 一律 `[]byte` 并点名 `:71`／`:78`／`:392`；`:4114`／`:4116` 已固定唯一编码路径（`json.Marshal(planWire(p))` + 项目内 `appendJSONString(dst, src []byte)`，`appendJSONString` 只转义 `"`、`\`、U+0000–U+001F，不做 HTML 转义，输出解码后逐字节等于输入），并禁止 `string(dsn)` 中间副本、base64、`json.RawMessage`、第二个 DSN 字段。
- `net/url` 字符串副本问题已由 `:3970` 关闭：「不使用 `net/url` 解析带密码的 DSN」；因此百分号解码只能落在 `[]byte` 上，与清零点 (6) 自洽。
- 错误文本：`:4045` 与 `:4122` 一致要求只保留分类 + 进程内随机关联 ID，禁止 DSN 与任何可离线比对派生值（摘要、指纹、前缀、host、用户名）。
- 残余诚实登记：`:4120`（GC 复制、管道内核缓冲、子进程地址空间、core dump、调试器；`RLIMIT_CORE=0`／`PR_SET_DUMPABLE=0`／`madvise(MADV_DONTDUMP)` 属 unknown，不得作为已满足证明）与 `:4053(2)`。未发现「已清零＝绝不泄露」的过度承诺。
- 未闭合 LOW：N013。

**N003（原 HIGH）→ 已闭合**
- `:4043` 唯一 wire 固定为 12 字段 `han-restore-receipt/v1`，并逐一点名取代 `:3884`（r61 driver 12 字段表）、`:1602`（Go `RestoreReceipt`）、`:1667`（§4.2 字段说明）；`:3884` 与 `:1667` 有就地取代标记。
- 旧字段处置完整：`run_id`／`backup_ref`／`deployment_id`／`installation_id`／`target_digest`／`artifact_digest`／`artifact_bytes`／`source_approval_digest` 留在 `RestorePlan`／批准记录由 `VerifyRestored` 只读核对；`receipt_digest` 不再作回执字段，Store 同名列只存 `receipt_sha256` 且不得当 wire（`:1718` 就地标记）；`restored_schema_digest` 只读核对；`restored_at` 不作权威时间。
- `attestation` 与回执 7 个同名字段（`producer_identity`、`producer_key_id`、`trust_anchor_ref`、`sequence`、`issued_at`、`signature_alg`、`signature`）逐字节相等，任一不等 → `ErrAdmissionBinding`／`restore_unknown`；`audit_record_ref` 不入 producer 载荷与 attestation。此三项均为可机核判据。
- 残余（LOW）：`:1602` 类型块本身无就地标记，只靠 `:4043` 的行号引用定位；`:1718` 的 `receipt_version` CHECK 值仍为 `han-db-restore/v1`；代码侧第三个别名 `restoreReceiptWire` 未被点名 → 合并为 N012。

**N006（HIGH）→ 已闭合**
- `:4041`：`none` 与 `self_attested` 统一为「工具自证 + 事后审计」，删除「`none` = 不联网」断言，明写「本设计不提供网络隔离能力」，`netns_inode`／`netns_path` 两模式下必须为空串，`netns` 仅作 fixture 且运行时 `ErrAdmissionUnsupported`，无法绑定即 `ErrAdmissionBinding`。`:3980` 有「r64.7 取代」标记，r64.13(4) 登记残余。

**N008（HIGH）→ 已闭合**
- `:4047` 把父目录 root／`0700`／逐级 `lstat`／禁 POSIX ACL、`O_NOFOLLOW|O_CLOEXEC` 双 fd、文件 root／`0600`、`(st_dev, st_ino)` 每次追加前复核、`flock(LOCK_EX)` 覆盖「读末条 → 算 `prev_record_sha256` → 追加 → `fsync` 文件 → `fsync` 父目录」写成**现行判据**，并明写「不得把父目录边界、ACL、inode 复核或锁区间写进残余风险段而弱化为建议」。残余仅保留「root／父目录写者可整体替换，抗回滚只到外部见证 head 可验，不声称抗 root」——与本设计能力一致。

**SEC-F008／N009／N010（LOW）→ 已闭合**
- `:3930` canonical 排序数组清单补 `organizations`／`appointments`（就地标记），`:3934` 说明两集合按各自键升序去重参与 JCS。
- `:4049`：信任根上限 `1..1048576` 且给出理由（四集合叠加），判定位置在信任根读取边界、早于任何签名校验，超限失败分类保持 `ErrAdmissionIncomplete`；approval 仍 `1..65536`。
- `:4090`／`:4122`／`:4184` 均有取代标记；`:4122` 删掉「长度」，只留分类 + 进程内随机关联 ID。
- `:4081` 紧邻 `:4078` 写明该向量 `network.mode=netns` 只是 canonical fixture、不构成运行时可接受批准记录、不得复制为示例，向量字节不变。
- `:4049` 明确 r61 段 `tool.receipt` 示例的 `sequence_floor` 只是历史、现行一律 `baseline_sequence`（对 `:4315` 适用）。

**SEC-F001（r03 已闭合，本轮只核残余是否仍诚实）**：`:4088` 仍明确「任何能同时改写信任根文件与 pin 的主体（root 或同权限部署方）仍可替换信任根… 不得以『有签名』暗示已抗 root 伪造」，生产 restore 继续 closed；`:4061` 的信任根与批准文件「不同受控只读挂载来源」、`O_RDONLY|O_CLOEXEC|O_NOFOLLOW`、fstat 与路径身份一致均保留；`:4067` 的 JCS 约束与「禁止 `encoding/json` 默认 `json.Marshal` 生成 canonical bytes」保留。r64 未削弱该结论。

## 4 仍开放 BLOCKER

### SEC-F002（BLOCKER）复核人独立性／一人两名 —— 部分闭合

**原 finding**：`principal_id`／`appointment_ref`／`review_ref`／`kind`／affiliations 交叉禁止／两两不同不足以机核地阻止一人两名或同组织自审，校验点与时间关系不可实施。

**r64 落点**：`:4035`（r64.4）、`:3934`（keyset）、`:3936`（交叉约束 (1)–(9)）。

**已闭合部分**：`principals` 增 `person_ref`；角色只来自 `appointments.role_name`；`reviewer_key_id`／`producer_key_id`／`verifier_key_id`／`witness_key_id` 的 `keys.principal_id` 与对应 principal 逐项绑定（r64.4(4)）；「同一 `person_ref` 不得在同一 review 的 `reviewer`／`issuer`／`executor` 两席」（r64.4(2)）。这四项均为可机核合取。

**仍开放三条（每条都有可机核判据）**：

(a) **`:3936` 交叉约束 (3) 未被就地取代，且比较对象不存在**。该行无任何 r64 标记（`:3934` 有标记，`:3936` 没有），仍要求 appointments 记录的 `role_name` 与「principal 侧一致」；但 `principals` 的 exact keyset（`:3934`，r64.4 增补后）为 `principal_id, person_ref, display_name, kind, appointment_ref, org_id, affiliations, valid_from, valid_until, revocation`——**没有 `role_name`**。该比较无右侧操作数，实施者只能自选语义（与 display_name 比？忽略？拒绝？）。r64.4(1)(3) 说的是「角色取自 appointment」，并未在 (3) 的位置作废该比较。
判据：`rg 'role_name.*与 principal 侧一致'` 在 r64 后应为 0 命中，或 `principals` keyset 增加 `role_name`。

(b) **`role_name` 值域未枚举，且 r64.4(3) 的字面值与冻结向量互斥**。全文无「`role_name` 只允许 …」的封闭枚举（`appointments` 项仅写 keyset 与唯一性）。r64.4(3) 规定：`reviewer_principal_id` 的 appointment `role_name` 必须为 `reviewer`、issuer 为 `approver`、executor 为 `executor`、producer 为 `producer`、verifier 为 `verifier`、witness 为 `witness`。而两条 golden vector 中同一批身份写作 `restore_reviewer`（`:4072` 的 reviewer 块）、`restore_approver`（`:4072` 的 issuer 块）、`restore_executor`（`:4078` 的 executor 块），且 r64.11 明确「向量字节不变」。于是：按 r64.4(3) 实现的预检会拒绝本设计自己的验收向量；按向量实现则违反 r64.4(3)。两者不可能同时满足，且文档未给 `restore_*` 与裸角色的映射规则。
判据：存在唯一的 `role_name` 封闭枚举，且对 `:4072`／`:4078` 逐项求值全部通过。

(c) **`(role_name, org_id)` 作用域留下跨角色一人两名，且残余登记不完整**。r64.4(1) 只要求「同一 `(person_ref, role_name, org_id)` 至多一个 active principal」，因此同一自然人可用**同一 `person_ref`** 登记 `reviewer` 与 `witness` 两个 principal（`role_name` 不同即不同桶）；`:3936(4)` 的「两两不同」只覆盖 issuer／reviewer／executor，不含 witness。全文无 `person_ref(witness) ≠ person_ref(reviewer|issuer|verifier)` 约束。结果是：复核人与「外部见证」可由同一自然人控制，而 r64.13(1) 只登记了「同一自然人使用两个不同 `person_ref` 不可检测」——那不是同一风险，读者会误以为跨角色已被防住。
判据：`person_ref` 唯一性改为对 active principal 全局唯一，或增加 witness 与 reviewer／issuer／verifier 的 `person_ref` 互斥；残余段补登该条。

**影响**：同组织／同人自审可通过角色分桶绕过；外部见证的独立性在文本层不成立；验收向量与角色约束互斥，实施无法同时满足。

**修复方向（不代作者改）**：在 `:3936` 就地重写 (3)（删掉 principal 侧 `role_name` 比较）；给出 `role_name` 封闭枚举并显式吸收向量中的 `restore_*` 字面值（向量已冻结，只能靠枚举／映射消化）；把 witness 纳入独立性交叉约束；同步 r64.13。

### SEC-F003（BLOCKER）撤销 TOCTOU／调用图 —— 部分闭合（含一处未标记的旧编号冲突）

**原 finding**：五个读取点与每次 `clock.Now()` 重取、时钟回拨 → closed、in-flight → `restore_unknown` 的链路是否闭合，`revalidateApproval` 与 `prepared()` 关系是否闭合。

**r64 落点**：`:4037`（r64.5）、`:3942`（r63.3，已标「r64.5 取代」）、`:3990`（r63.8，**无标记**）。

**已闭合部分**：
- `:4037` 补齐 P4 `source.OpenApproved`（唯一 artifact 句柄）、P5 `RestoreSourceVerifier` 恰好一次且早于 `Journal.Begin`、P6 `Journal.Begin` 前重读批准文件精确字节与快照比对（漂移 → `ErrAdmissionClosed`）；R1 `prepared(ctx)` 保留；R2／R4／R6 三次 `revalidateApproval`（启动前、子进程前、子进程返回后解码回执前）；V1 `prepared(ctx)` 恢复。r03 指出的两个具体缺口（`Prepare` 缺重读、`VerifyRestored` 缺 `prepared`）均已补，且与代码现状不矛盾：`recovery_restore.go:375` 已在 `execute` 前调 `prepared`，`:366`–`:382` 与 V1 一致。
- `snapshot.Clear()` 与 DSN 比较的矛盾已消解：`:3942` 规定快照只保留摘要与身份元数据、`snapshot.Clear()` 在 driver／assembly 退出路径执行；`:4045(7)` 把 `dsnSHA256` 归入 `Clear()`；任何 `Clear()` 之后的再进入都必须先过 `prepared(ctx)`（R1／V1）的 closed 与幂等检查（`:4037` 明写「`V1` 的 `prepared(ctx)` 不得省略或下移」）。三者合取后不存在「已清零的摘要仍被用于比较」的窗口。
- 时钟：`:4037` 保留「每个检查点只取一次 `(wall, mono)` 样本并内传」，`:4031` 规定 `verified_at` 在 V10 一次性取样、不得二次取钟、与签名载荷逐字节相等；`:4110` 的「撤销自下一次 `revalidateApproval` 起生效、最长延迟等于相邻检查点间隔」与 r64.5 的检查点集合一致。**未发现跨重启或墙钟回拨下的保证声明**（r03 要求的「不得声称跨重启保证」仍成立）。

**仍开放一条（BLOCKER）**：`:3990` 属 r63.8（段首 `:3987`），**无任何 r64 标记**，仍以旧编号与旧载荷定义 `VerifyRestored`：
- `V7` = 重算并验证 producer 签名；`V8` = 读 `receipt_anchors.sequence_floor` 与见证 head；`V9` = 审计 + 验证方签名 + 推进 `sequence_floor`；
- `V9` 的验证方签名载荷为 **5 字段** `{receipt_sha256, sequence, verified_at, result, prev_record_sha256}`。

这与 r64.5 的 `V7`=独立读取、`V8`=producer 签名、`V9`=见证 head、`V10`=审计/签名/推进直接冲突，也与 r64.2 的 **8 字段**载荷（含 `anchor_ref`、`verifier_principal_id`、`verifier_key_id`）冲突。r64.5 自己声明「编号唯一，不得增删、重排或复用」；r64.0(3) 只作废 **r63.3** 的编号图，`:3990` 不属 r63.3；r64.0(7) 只把「验证方签名载荷」指向 r64.2，未处置 `:3990` 的 V 编号与 `sequence_floor` 旧名。结果是同一文档对 `V8` 给出两种互斥语义，且按 `:3990` 会签出不绑定 `anchor_ref` 与验证方身份的载荷——恰好重开 N001 的绑定缺口。

判据：`rg 'V7／V8／V9'` 只应命中 r64.5 及其取代标记；`:3990` 应带「r64.5／r64.2 取代」标记或被改写。

**影响**：实施者按 `:3990` 实现即违反 r64.2/r64.5，按 r64.5 实现则与文档现存规范句冲突；两种实现都能在文本中找到依据，属于不可判定的契约冲突，且失败方向是安全侧的（验证方签名不绑定锚与身份）。

### N001（BLOCKER）见证协议与权威推进 —— 部分闭合

**原 finding**：`receipt_anchors` schema 不足以支撑「权威推进 sequence」，见证协议、head、`audit_sink` 相等要求、签名覆盖面与验证方身份绑定不可实施。

**已闭合部分**：
- `:4033` `receipt_anchors` exact keyset 扩为 13 项（增 `witness_principal_id`／`witness_key_id`／`witness_public_key`／`witness_endpoint`／`witness_protocol`），`sequence_floor`→`baseline_sequence`（正则 `^(0|[1-9][0-9]*)$`），旧键出现即未知键 → `ErrAdmissionBinding`；
- head exact keyset 5 项 `anchor_ref, sequence, issued_at, head_sig_alg, head_sig`，`head_sig_alg` 只 `ed25519`，载荷 = 删 `head_sig` 后的 JCS，签名公钥 = `receipt_anchors.witness_key_id`；
- 判决合取：head 可验签、`head.sequence >= baseline_sequence`、本次 `sequence` 严格大于 `baseline_sequence` 与 `head.sequence`、`anchor_ref` 四处逐字节相等；policy 的 `audit_sink` 必须与 `receipt_anchors.audit_sink` 逐字节相等；
- `witness_principal_id` 要求 `kind=verifier` 且 appointment `role_name=witness`，与 `:3934` 的 `kind` 枚举（`root`／`approver`／`reviewer`／`executor`／`supplier`／`verifier`／`producer`）**不冲突**——`witness` 是 appointment 的角色而非 principal 的 `kind`，r64.4(3) 的同一写法与之自洽；`witness_public_key` 必须与 `keys[witness_key_id].public_key` 逐字节相等；`witness_principal_id` 不得等于 `verifier_principal_id`；
- 验签与推进的唯一性：`:4031` 固定 8 字段载荷与唯一公钥来源（`keys[verifier_key_id].public_key`，且该 key 的 `principal_id` 必须等于 `verifier_principal_id`），禁止回执自带公钥、ENV 覆盖、`insecure` 分支；签名失败不得推进 sequence；`:3990` 保留「见证先以 `verifier_key_id` 验证，失败不得推进；推进失败 → `restore_unknown`」，`:4033` 加「见证不可用即 `ErrAdmissionUnsupported` 并保持 closed，不得用本地文件替代」。这使 r03 的「权威推进不可实施」在 schema 与代码可写性层面**已可实施**。

**仍开放**：
- (a) 与 SEC-F003 同源：`:3990` 未标记的 5 字段载荷与旧 `sequence_floor` 会把外部见证收到的验证方签名载荷退回旧形状；
- (b) `witness_endpoint` 的独立性与传输信任未约束 → 独立列为 N011（HIGH）。

**影响**：`sequence` 单调推进的协议骨架已具备，但「外部见证不可由本机冒充」这一前提不成立，且签名载荷存在两套互斥定义，故不能判 N001 闭合。

## 5 仍开放 HIGH／LOW

### SEC-F006（HIGH）能力断言与放宽登记 —— 未闭合

**原 finding**：pre-exec 能力（`CapBnd` drop、`NoNewPrivs`、`setns`、seccomp）以 Go 不可实现为由降级时，是否诚实登记为放宽、是否残留「用 Go 可实现」的暗示。

**r64 落点**：`:4041`（r64.7）、`:4027`（r64.0 放宽登记）、`:4188`（r62.7.2 原文，已标取代）。

**已闭合部分**：
- `:4041` 明确 `seccomp`／`netns`／`setns` 一律 unsupported，需要时 `ErrAdmissionUnsupported` 并保持 closed，「不得先 exec 再补设置」；明写「bounding set 保留但不在 exec 时约束子进程」「本设计**不声称**」；`CapInh=CapPrm=CapEff=CapAmb=0`、Uid/Gid 四元组、`Groups` 为空仍是拒绝条件；`:3983` 的「Go 无 pre-exec 钩子」理由保留且没有「用 Go 可实现」的反向暗示；`:4188` 的旧断言已带「r64.7／r64.0 取代」标记。r03 指出的「残留暗示」已消除。

**未闭合理由（可机核）**：
- `:4188`（r62.7.2）的拒绝条件是 `CapInh=CapPrm=CapEff=CapBnd=CapAmb=0`、`NoNewPrivs=1` **全部**；r64.7 只保留 `CapInh/CapPrm/CapEff/CapAmb` 与身份四元组、`Groups` 为拒绝条件，把 `CapBnd` 与 `NoNewPrivs` 同时降为观测值。这是**两项**放宽。
- `:4027` 的放宽登记标题写「**放宽登记（唯一一项）**：`NoNewPrivs` …」，正文只在「抵消措施」里顺带提到「`CapBnd` 只作观测」，未作为放宽项列出，也未写 `CapBnd` 放宽的理由与影响。于是同一节自相矛盾：登记声称 1 项，实际 2 项。
- 第三项未登记放宽：`:4039` 重新允许 `PATH`（`:3976` 起即如此），而 r61.5 `:4408` 原文是「禁止 `PATH`」。ADR-0031 `:1144` 允许「`PATH` 指向任一非批准目录…出现即 closed」，即允许批准目录的 `PATH`，故**不违反 ADR**；但相对 r61.5 是放宽且从未在任何放宽登记中出现。

**判据**：`rg '放宽登记（唯一一项）'` 与拒绝条件集合的差集必须为空；充分条件是放宽登记逐项列出 `NoNewPrivs`、`CapBnd`、`PATH` 三项，各含放宽内容／理由／抵消措施／残余。

**影响**：读者（含部署方与后续评审）会把「本设计只放弃了 no-new-privs」当成事实，而实际还放弃了 bounding-set 的准入约束与 ENV 收紧；这直接影响对提权面的判断，属安全契约的静默放宽。**不因写了 unsupported 判过。**

### N011（HIGH，新增）`witness_endpoint` 可被本机冒充，见证独立性不可机核

**位置**：`:4033`（字段约束）、`:4053`（残余清单未收录）。

**事实**：`witness_endpoint` 的全部约束是「`https://` 绝对 URL、≤2048 字节、无 userinfo、无 fragment」。没有：host 允许集；loopback／link-local／私网／ULA 排除；TLS 信任锚或证书 pin；DNS 解析结果约束；query 是否可携带凭据的禁止；见证方与部署方／root 的管理域分离要求。

**触发场景（可复现）**：部署方（或 root）在 `127.0.0.1:443` 起一个 HTTPS 服务，把 `witness_endpoint` 指向它并把自己的公钥登记为 `witness_public_key`。预检与运行期全部通过：head 可验签、`sequence` 严格递增、`anchor_ref` 四处相等。于是「外部见证」= root 自己，`baseline_sequence` 与 `sequence` 的单调性、抗回滚的最后一层全部失效；root 同时还能借该 URL 对任意外部主机发起 TLS 连接（受控主机上的出网通道）。

**违反依据**：ADR-0031 `:1147`「验证方须同时独立于执行身份与 root…验签失败、信任锚不可固定或验证方不独立即 closed」；`:1166`「验签信任锚固定登记在批准记录内且轮换经人的明确批准」。r64.3 的 `witness_public_key` + `witness_endpoint` 只满足「可固定」，不满足「独立于 root」。

**影响**：N001 想闭合的「权威推进不可本地伪造」在传输与部署层不成立；且新增了一条由特权进程发起的、目标由信任根任意指定的出网通道（新攻击面）。

**修复方向**：`witness_endpoint` 解析后所有 A/AAAA 必须落在非 loopback／非 link-local／非私网／非 ULA，且与登记 host 精确匹配；要求独立的 TLS 信任锚（pin 字段或系统根 + 强制 hostname 校验）；禁止 query／fragment 携带任何凭据；把「见证方与部署方／root 分属不同管理域」写成部署前置并在 r64.13 登记残余。判据：给定 `127.0.0.1`、`::1`、`10/8`、`192.168/16`、`fc00::/7` 的 endpoint，预检必须 `ErrAdmissionBinding` 或 `ErrAdmissionUnsupported`。

### N012（LOW，新增）回执版本与旧类型的同版漂移

- `:1718` 的 Store DDL 仍为 `receipt_version text NOT NULL CHECK = 'han-db-restore/v1'`，而唯一回执 wire 是 `han-restore-receipt/v1`（`:4043`）。该行的 r64.8 标记只处置了 `receipt_digest` 列，未处置 CHECK 字面值。判据：receipt 语境下 `han-db-restore/v1` 应为 0 命中（该值仍可合法用于 `RestorePlan.Version`，见 `:3878`）。
- 代码侧第三个别名 `restoreReceiptWire`（`cmd/han-server/recovery_restore.go:52`–`:65`，含 `receipt_digest,omitempty`）未被 `:4043` 点名；`:4043` 只点名 `internal/recovery.RestoreReceipt` 与 design `:1602`。判据：`rg 'restoreReceiptWire'` 在改造后为 0 命中，或 `:4043` 明确点名该结构。

### N013（LOW，新增）DSN 写入路径的字段／类型命名

- `:4045` 写「`restorePlanWire` 的 DSN 写入必须用 `appendJSONString` 直写字节」，但 `restorePlanWire`（代码 `:36`–`:50`）没有 DSN 字段，DSN 在匿名结构 `Target.DSN`（`:74`–`:76`）。所指类型不准。
- `:4045`「`:78` 内部字段改 `[]byte`」与 `:4114`「wire 结构体字段类型仍是 `string`，`restorePlanWire`／`restoreReceiptWire` 不变」并列，且无消解句；`:4116` 的实际做法是「只 `json.Marshal(planWire(p))`，DSN 单独追加」，即该 string 字段应被删除而非改类型。三者共存时实施者会得到两种改法。
- 判据：改造后 `rg 'Target\.DSN' cmd/han-server` 为 0 命中，且不存在 `string` 型 DSN 变量；`:4045` 指向「`privateRestoreInput` 的匿名 wire 结构」。

### N014（LOW，新增）`static`／`dynamic` 措辞自否

`:4039` 同段先写「`static`（无 `PT_INTERP`、`dt_needed` 为空）是唯一**完整可绑定**形态」，紧接着写「`dynamic` 必须在无任何 `LD_*` 的前提下…若无法证明 → `ErrAdmissionUnsupported`」。后者意味着 dynamic 在可证时是允许的，与「唯一」互斥。判据：同一段内「唯一可绑定形态」与「dynamic 允许路径」只能保留一个，或改写为「static 是默认形态；dynamic 仅在下列合取成立时允许」。

### N015（LOW，新增）R 图与现有代码存在未列节点

r64.5 声明 R1–R8「不得增删」，但代码 `recovery_restore.go:352` 在 `tool.Run` 之后调用 `d.prepared(ctx)`；`prepared` = `CheckApproved` + `checkRestoreArtifact`（重算 artifact 摘要并 `Seek(0)`），**不是** `revalidateApproval`，且该节点不在 R1–R8 中（R6 要求的是 `revalidateApproval`）。实施者要么保留一个未列节点（违反「不得增删」），要么删除现有检查（改变行为）。判据：`execute` 之后不得出现 `prepared(ctx)`，或 r64.5 显式声明 `:352` 由 R6 取代。

### N016（LOW，新增）旧字段名在规范表中残留

`:4012`（r63.9.4 建议写集的基线表，r64.12 在其上增补）仍写「外部见证客户端、`sequence_floor` 原子推进」，无 r64 标记；而 `:4049` 明确「现行基线一律 `baseline_sequence`」。`:4315` 的 `sequence_floor` 已由 `:4049` 的全局说明覆盖，`:4012` 未覆盖。判据：建议写集表中 `sequence_floor` 为 0 命中。

### 范围外 N005（不计入本报告计数，但阻断启动）

`framework.md:1129` 仍写「r07 尚未复评，故决策接受级对现行修订尚未达成…`identity-auth-S4.24a` 及任何实施/验证卡不得启动」，`tasks.md:3197` 仍写「本卡启动门禁解除」。r64.13(8) 只登记、不裁定。设计侧不能也不应代改；在 ADR 与任务卡对账前，S4.24a 的启动状态不可判定。

## 6 额外必查项

**新增配置键与秘密面**：r64 未新增任何配置键——恢复相关仍为 `HAN_RECOVERY_RESTORE_APPROVAL_PATH`、`…_TRUST_ROOT_PATH`、`…_TRUST_ROOT_SHA256`、`…_TARGET_DSN_FILE`、`…_EXECUTABLE`／`_SHA256`、`…_MANIFEST_PATH`（`:3914`–`:3916`、`:4061`），与 r03 相同。全文 `HAN_` 键清单无新增项，故无新的 ENV 可见性／错误回显面。唯一新增的外部输入面是信任根内的 `witness_endpoint` → N011。

**信任根文件的权限／属主／挂载**：与批准文件同等严格——`:3982` `ro,nodev,nosuid,noexec`、属主 root、group/other 无写、父目录逐级 `lstat` 无符号链接、不允许 POSIX ACL；`:4061` `O_RDONLY|O_CLOEXEC|O_NOFOLLOW`、fstat 与路径身份一致、UTF-8 无 BOM/NUL；`:4061` 与批准文件必须来自不同受控只读挂载来源（`mount.source` 或 `mount_point` 至少一项不同）。上限提高到 1 MiB 的理由（四集合叠加）与判定位置（读取边界、早于签名校验）已写明。

**`audit_sink` 追加／fsync 的截断与重放**：`:4047` 的 `flock` 区间覆盖「读末条 → 算 `prev_record_sha256` → 追加 → `fsync` 文件 → `fsync` 父目录」，`(st_dev, st_ino)` 每次追加前复核；`:3990` 规定 `prev_record_sha256` 必须等于最后一条已 `fsync` 记录的 `audit_record_ref`，空文件时为空串，否则 `restore_unknown`。截断到见证 head 以上可检测；head 以下不可检测，这一点已在 `:4047` 明确登记（「抗回滚只到外部见证 head，不声称抗 root」）。`audit_sink` 路径不进入错误文本（`:4122` 只留分类 + 进程内 ID），未发现路径回显面。

**`sequence` 持久化与回滚语义**：持久化责任方为外部见证，producer／本地审计文件／信任根均不承担；`baseline_sequence` 只由部署方 out-of-band 更新、运行时不回写信任根（`:4033`、`:4066` 一并说明）；旧键 `sequence_floor` 出现即 `ErrAdmissionBinding`。语义足够——除 N011 的冒充通道外。

**第二 authority／fallback／隐式默认／把 closed 变 open**：未发现。`:4027` 的不变量 (1)–(6) 逐条为判定；`:4029` 规定 `inputs.exclusive == nil` 时 `Acquire` 必须在 driver 构造前失败并原样返回 `ErrAdmissionClosed`，不得把 nil 当无锁运行、不得把 `Bind` 失败降为 warning；`:4033` 见证不可用 → `ErrAdmissionUnsupported` 并保持 closed，不得用本地文件替代；`:4041` 需要真实隔离 → `ErrAdmissionUnsupported`。未发现新增「本次忽略」「按目录找最新」类分支（`:4088`、`:4136` 仍明令禁止）。

**是否事实上解锁生产批准主体／生产工具**：未解锁。`:4027(1)(2)(3)`、`:4053(6)`、`:4055` 均声明生产 restore 继续 closed、`recoveryClosedRestoreDriver` 仍是默认、`enter`／`resume` 不解锁；`tasks.md:3186` 仍 `- [ ]`；r64.12 明写「只作建议，不改 `tasks.md`」。

**回归与诚信扫描（全文，区分大小写）**
- `已隔离`／`不联网`：仅出现在被删除或否定的句子与历史段（`:3980`、`:3983`、`:4021`、`:4041`、`:4053`、`:4057`、`:4164`、`:4250`、`:4410`），无肯定句，未发现「已隔离」宣称。
- `PASS`（大写）：65 处命中全部属 identity-auth 历史评审／checkpoint／否定句（如 `:3738`「不由作者记 CLOSED/PASS」、`:4055`「不构成评审通过」、`:4164`、`:4250`），恢复区无新增肯定句，NOT RUN 未被写成 PASS。
- `LD_LIBRARY_PATH`：`:3975`／`:3976`／`:4039`／`:4408`，全部为禁止或取代标记，无允许路径残留。
- `CapBnd`／`NoNewPrivs`：`:4188` 已标取代；`:4027`／`:4041`／`:4053` 为降级说明。**但 `:4027` 的「唯一一项」与事实不符**（SEC-F006）。
- `sequence_floor`：`:3934`／`:3942`／`:4027`／`:4033`／`:4049`／`:4063`／`:4067`／`:4128` 均有改名或取代标记；**`:3990` 与 `:4012` 无标记**（SEC-F003／N016）。
- `baseline_sequence`：`:3934`／`:4033`／`:4049`／`:4063`／`:4067`／`:4128`，与旧名并存但方向一致。
- `receipt_digest`：`:1671`／`:1718`／`:3884`／`:4043`，均已标 Store 绑定边界；`:1718` 的 CHECK 字面值另见 N012。

**NOT RUN（SEC-F007）**：`:4053(7)`「Linux 目标／CI 的 host／ACL／mount／loader／fd-exec／maps／network／witness 实测均未运行；Windows 只覆盖解析与语义层，`//go:build linux` 的 skip 与无匹配测试的 exit 0 都不等于 PASS」、`:4164`、`:4250`、`:4055`「任何 NOT RUN 仍是 NOT RUN」均未被改写，未暗示可关闭 S4.24a。**不计入通过。**

## 7 Verdict 与计数

**Verdict：`CHANGES REQUIRED`**

| 级别 | 计数 | 条目 |
|---|---|---|
| BLOCKER（设计侧） | 3 | SEC-F002、SEC-F003、N001 |
| HIGH | 2 | SEC-F006、N011（新增） |
| LOW | 5 | N012、N013、N014、N015、N016（均为新增） |
| 已闭合 | 8 | SEC-F005、N003、SEC-F004、N006、N008、SEC-F008、N009、N010 |
| 范围外 BLOCKER | 1 | N005（不计入上表） |
| NOT RUN | 1 | SEC-F007（不构成通过） |

**阻断实施的最关键一条**：**SEC-F006**——`:4027` 声称「放宽登记（唯一一项）」只有 `NoNewPrivs`，实际同时把 `CapBnd` 从拒绝条件降为观测值、并保留 `PATH` 的重新允许。这是安全契约的静默放宽：后续实现者与部署方会按「只放弃 no-new-privs」评估提权面，而实际放弃了 bounding-set 的准入约束。修复成本低（逐项登记即可），但必须先改，否则任何基于该段的隔离结论都建立在错误的自我描述上。

**最危险的实现歧义（并列说明）**：SEC-F003 的 `:3990` 未标记 5 字段验证方签名载荷＋旧 V 编号，会让实现者签出不绑定 `anchor_ref` 与验证方身份的载荷，直接重开 N001 的绑定缺口。

## 8 未验证边界

- **Linux host 层全部 NOT RUN**：owner/root、`0700`、ACL、mountinfo、loader（`PT_INTERP`／`DT_NEEDED`／`DT_AUDIT`／`DT_DEPAUDIT`／cache／conf／preload）、fd-exec、`/proc` maps／status／auxv、netns、seccomp 均未在任何 Linux 主机或 CI 上运行；本报告只判设计文本的可机核性，不判实现是否正确。
- **见证与审计 sink 无实物**：`witness_endpoint`、`witness_public_key`、`audit_sink`、`independent_read_path`、真实信任根、真实批准记录均不存在，N011 的冒充路径是**文本级推理**，未做端到端复现。
- **代码未构建、未运行**：本报告只做静态阅读（`recovery_restore.go`、`runtime.go`），未执行 `go build`／`go vet`／`go test`，也未验证 r64.9 的改造在 Go 类型系统下可直接编译（例如删除 `Target.DSN` 后 `planWire` 与 `json.Marshal` 的配合）。
- **DSN 清零的运行时效果未验证**：GC 复制、管道缓冲、子进程地址空间、core dump、调试器残留均为设计声明的残余，无实测。
- **向量只覆盖两条**：`source`／`tool`；`receipt_anchors`、`audit` 记录、见证 head、`baseline_sequence` 均无 golden vector，其 JCS 字节无法独立复算。
- **`person_ref` 与角色约束的判定**：本报告给出的是文本可满足性，未在真实信任根 fixture 上求值；(c) 的绕过路径依赖「同一自然人可控两个 principal」，属组织事实，设计层只能约束声明。
- **N005 未裁定**：ADR-0031 r07 与 `tasks.md` 的启动状态冲突不在设计侧权限内，本报告不代替架构决策方或 Task-Planner。

## 9 声明

- 本次为只读独立评审：未修改 `design.md`、`tasks.md`、ADR、`cmd/**`、`internal/**`、任何 `.go` 文件，未修改或覆盖 `review-sec-r01`／`r02`／`r03`、`review-arch-r01`、`design-revision-r01`／`r02`／`r03`／`r04` 的任何报告。
- 未做 Git 写操作（无 add／commit／branch／stash／checkout／restore／reset／clean），未删除、移动或归档任何文件；工作区未提交改动原样保留。
- 唯一写入为本文件：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r04/report.md`（新建）。
- 本报告的结论不代替人的接受、不构成生产可用、不构成 `identity-auth-S4.24a` 可启动或可关闭、不解除任何 closed。
- 自校验：除本行外正文 **40516 字节 / SHA256 `84f14cb99438e7c5119673a3e987ed419764ca60b7a62c1c948541da5a4c96a9`**；含本行全文的字节数与 SHA256 由评审方在写入后复算，记录于本次交付回复，与本行共同构成可核对的完整性锚。
