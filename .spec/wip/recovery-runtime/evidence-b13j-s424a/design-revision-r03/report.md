# identity-auth design r62 → r63 作者报告（r02 两路评审 BLOCKER／HIGH 最小修复）

## 1. 元数据与修订对象

| 项目 | 值 |
|---|---|
| 作者角色 | ADD Feature-Designer |
| 设计文件 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| 修订前 revision | `identity-auth-design-ir01-r62` |
| 修订后 revision | `identity-auth-design-ir01-r63`（frontmatter `revision:` 与第 38 行设计版本一致） |
| 修订前字节 / 行数 | 936613 B / 5317 行 |
| 修订后字节 / 行数 | 976149 B / 5416 行 |
| 修订前 SHA-256 | `EA880660B8D43AFD6FB227812C662BB5A4581B39F4235F3DAB8E221F5F9FDC9A` |
| 修订后 SHA-256 | `3B8BEA8C35A12F75B36091969DB15B9590FBDD16B0429D59328ABB9FA3BC027C` |
| 工作区 / 分支 / HEAD | 固定工作区 / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 未提交改动 | 修订前 `git status --porcelain` 295 条；本轮只新建本报告，且它位于已存在的未跟踪目录 `.spec/wip/recovery-runtime/` 内，故修订后仍为 295 条；未撤销、未覆盖任何既有改动 |
| 本报告性质 | 作者报告；不是评审报告，不构成评审通过或任务关闭 |

输入评审对象（本轮完整逐条读取，字节数与哈希复核一致）：

| 对象 | 字节 | SHA-256 | 结论 |
|---|---:|---|---|
| `review-arch-r02/report.md` | 46877 | `FEB4C89F6A3210D6BC33DFFB29B1BF3A3131360AB9C9BE9696BAE1220736C30C` | CHANGES REQUIRED；BLOCKER 4／HIGH 3／MEDIUM 3 |
| `review-sec-r02/report.md` | 57982 | `9AD6C71CEFEEB06724DEC07D9CBF553EB787855BF804EFCC908870FD76CF9705` | CHANGES REQUIRED；BLOCKER 7／HIGH 6／LOW 2 |
| `design-revision-r02/report.md` | 18821 | `A910B3DD42223025B7D7D7C1CE80CD1D366E582BDDA2E13E8A415E4E660A0549` | 上一轮作者报告 |

修订方式：r63 增量整段插在 r62 之前（`design.md:3925`–`:4022`），r62 正文从 `:4024` 起保留作历史；只就地替换 r62.1 的两条非 canonical golden vector 与两行摘要句（`:4039`–`:4048`），并更新 frontmatter `revision:`、`last_updated`（原值已为本日，未改）与第 38 行设计版本。r61 正文从 `:4230` 起未改。`tasks.md`、ADR 正文、`cmd/**`、`internal/**`、测试与任何 `.go` 均未修改。

## 2. JCS golden vector 复算（ARCH-F001／SEC-F001／NEW-R02-07）

### 2.1 缺陷与修正

r62.1 的两条 canonical 串声称按 UTF-16 code unit 升序，但顶层 `revocation` 排在 `reviewer` 之前：`reviewer` 第四字符为 `i`，`revocation` 第四字符为 `o`，故正确顺序必须是 `record_type` < `reviewer` < `revocation`。旧串因此不是合法 canonical bytes，旧摘要 `3c7b1cff…`（source）与 `9df40264…`（tool）只对应非 canonical 字节，已从全文删除（仅在本报告与 r63.1 中以截断形式作历史反例）。r63 已把两条向量就地递归重排，并写回摘要句；摘要句带 `（r63.1 修订）` 标记。

### 2.2 复算算法（第三方可复现）

本轮使用的脚本逻辑如下；它只依赖 JSON 解析、UTF-16 code unit 排序、紧凑序列化与 SHA-256，不依赖本仓库任何私有函数：

```python
def sort_obj(o):
    if isinstance(o, dict):
        return {k: sort_obj(o[k]) for k in sorted(o.keys(), key=lambda s: s.encode("utf-16-be"))}
    if isinstance(o, list):
        return [sort_obj(x) for x in o]
    return o

canonical = json.dumps(sort_obj(json.loads(line)), ensure_ascii=False, separators=(",", ":")).encode("utf-8")
digest = hashlib.sha256(canonical).hexdigest()
```

规则声明：对象键递归按 UTF-16 code unit 升序；数组保持原顺序；输出无空格、无缩进、无尾随换行；UTF-8；只对 `"`、`\` 与 U+0000–U+001F 做 JSON 转义（短转义优先，其余 `\u00xx` 小写）；不做 HTML 转义；非 ASCII 原样输出；整数型字段已是十进制字符串。任何使用 Go 默认 `encoding/json`／`json.Marshal` 生成 canonical bytes 的实现都违反本规则（HTML 转义与 U+2028／U+2029 处理会改变字节），必须拒绝。

### 2.3 复算结果

| 向量 | canonical 字节数 | SHA-256 | 首个差异偏移（相对旧串） | `reviewer` < `revocation` |
|---|---:|---|---:|---|
| `source` | 1115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 454 | 是 |
| `tool` | 2357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 1738 | 是 |

本轮按规则重新解析 `design.md:4039`／`:4045` 的字节：递归键序检查返回空集（即全部对象键已升序），重排序列化结果与文中字节逐字节相等（`canonical_ok = True`），故两个摘要对应的是合法 canonical bytes。上述两个摘要与 Lead 独立复算、以及两路 r02 评审给出的值一致。**作者复算不等于独立验证**；独立验证必须由未参与本设计的实现或评审方按 2.2 的规则重新执行，并得到同一字节与同一摘要。

### 2.4 完整 canonical 字节串（source，`design.md:4039`）

```text
{"approval_ref":"APR-SRC-0001","artifact_bytes":"1048576","artifact_digest":"sha256:1111111111111111111111111111111111111111111111111111111111111111","backup_ref":"BK-2026-11-02","deployment_id":"dep-han-prod","installation_id":"inst-han-prod","issued_at":"2026-11-02T00:00:00.000000Z","issuer":{"appointment_ref":"APT-ISS-01","person_name":"Approver One","principal_id":"prn-issuer-01","role_name":"restore_approver"},"record_type":"restore_source","reviewer":{"appointment_ref":"APT-REV-01","person_name":"Reviewer One","principal_id":"prn-reviewer-01","review_ref":"REV-2026-11-02-01","reviewed_at":"2026-11-02T01:00:00.000000Z","role_name":"restore_reviewer"},"revocation":{"revocation_ref":"","revoked_at":"","state":"active"},"source_installation_id":"inst-han-old","source_schema_digest":"sha256:2222222222222222222222222222222222222222222222222222222222222222","supplier_principal_ids":["prn-backup-supplier","prn-tool-supplier"],"target_digest":"sha256:3333333333333333333333333333333333333333333333333333333333333333","valid_from":"2026-11-02T00:00:00.000000Z","valid_until":"2027-11-02T00:00:00.000000Z"}
```

### 2.5 完整 canonical 字节串（tool，`design.md:4045`）

```text
{"acl":{"digest":"sha256:4444444444444444444444444444444444444444444444444444444444444444","kind":"posix_access"},"approval_ref":"APR-TOOL-0001","backup_ref":"BK-2026-11-02","deployment_id":"dep-han-prod","elf":{"dt_audit":[],"dt_depaudit":[],"dt_needed":[],"elf_type":"static","ld_so_cache_path":"","ld_so_cache_sha256":"","ld_so_conf_paths":[],"pt_interp_path":"","pt_interp_sha256":"","rpath":[],"runpath":[]},"environment":["LANG=C.UTF-8","LC_ALL=C.UTF-8","TZ=UTC"],"executable_path":"/opt/han/restore-tool","executable_sha256":"sha256:5555555555555555555555555555555555555555555555555555555555555555","executor":{"appointment_ref":"APT-EXE-01","gid":"65532","principal_id":"prn-executor-01","role_name":"restore_executor","uid":"65532"},"installation_id":"inst-han-prod","issued_at":"2026-11-02T00:00:00.000000Z","issuer":{"appointment_ref":"APT-ISS-01","person_name":"Approver One","principal_id":"prn-issuer-01","role_name":"restore_approver"},"mode":"0755","mount":{"mount_point":"/opt/han","options":["nodev","nosuid","ro"],"source":"/dev/sdb1"},"network":{"mode":"netns","netns_inode":"4026532500","netns_path":"/run/han/restore-netns","target_addr":"db.internal:5432"},"owner_gid":"0","owner_uid":"0","receipt":{"audit_record_ref":"","audit_sink":"/var/lib/han/audit/restore.jsonl","independent_read_path":"/var/lib/han/restore/receipt.json","issued_at":"","producer_identity":"backup-system","producer_key_id":"key-receipt-01","sequence":"0","signature":"","signature_alg":"ed25519","trust_anchor_ref":"anchor-01","trust_anchor_sha256":"sha256:6666666666666666666666666666666666666666666666666666666666666666","verifier_identity":"external-verifier","verifier_principal_id":"prn-verifier-01"},"record_type":"restore_tool","reviewer":{"appointment_ref":"APT-REV-01","person_name":"Reviewer One","principal_id":"prn-reviewer-01","review_ref":"REV-2026-11-02-01","reviewed_at":"2026-11-02T01:00:00.000000Z","role_name":"restore_reviewer"},"revocation":{"revocation_ref":"","revoked_at":"","state":"active"},"security":{"capabilities":"","setgid":false,"setuid":false},"st_dev":"2049","st_ino":"123456","supplier_principal_ids":["prn-backup-supplier","prn-tool-supplier"],"target_digest":"sha256:3333333333333333333333333333333333333333333333333333333333333333","valid_from":"2026-11-02T00:00:00.000000Z","valid_until":"2027-11-02T00:00:00.000000Z"}
```

### 2.6 负向反例与验收方向

旧串（`revocation` 在 `reviewer` 之前）的摘要 `3c7b1cff…`／`9df40264…` 只对应违反规则的字节，任何合规实现都不会产出；它是本轮的负向反例。r63.1 已写明判定方向：若实现的键序与向量不一致，必须拒绝该实现，不得反过来修改向量迎合实现。

## 3. 架构侧 r02 finding → r63 落点 → 判定

严重级沿用 `review-arch-r02`。判定只表示"设计层是否已给出唯一、可编码、可核的契约"，不表示评审通过或实现完成。

| ID | 等级 | r63 章节／行号 | 判定 |
|---|---|---|---|
| F001 | BLOCKER | r63.1 `:3928`–`:3931`；向量 `:4039`–`:4048` | 闭合（设计层）；独立 JCS 复算仍属复审职责 |
| F002 | HIGH | 保持 r62 结论；r63.1 `:3928`、`:3931` 维持 `sha256:` 唯一表示 | 保持闭合 |
| F003 | HIGH | 保持 r62 结论；r63.2 `:3933`–`:3937` 固定 tool keyset 与 `reviews.tool_sha256` 映射 | 保持闭合 |
| F004 | HIGH | r63.9.1 `:3995`（与 r60 的适用关系与优先句，不改 r60 原文） | 闭合 |
| F005 | BLOCKER | r63.4 `:3945`–`:3968`（导出 claim、fd 唯一持有、`ReadAt`／`Seek(0,0)`） | 闭合 |
| F006 | BLOCKER | r63.9.2 `:3997`（唯一 `Acquire` 后传 lease，`Bind` 不再收 provider） | 闭合 |
| F007 | BLOCKER | 保持 r62 结论；r63.0 `:3927` 第（5）项只取代清零清单与错误文本 | 保持闭合 |
| F008 | HIGH | 保持 r62 分流；r63.9.4 `:4009`–`:4010` 明确 exec 平台文件职责 | 保持闭合 |
| F009 | HIGH | 设计侧已处置；r63.9.5 `:4020` 第（3）项 | 设计层已处置；卡关闭条件仍 NOT RUN |
| F010 | HIGH | 保持 r62 分层；r63.9.5 `:4020` 第（3）项 | 保持闭合（host 层 NOT RUN） |
| F011 | HIGH | r63.3 `:3942`（R6／V5：execute 返回后、解码前 `revalidateApproval`） | 闭合 |
| F012 | MEDIUM | 保持 r62 结论；r63.6 `:3982` 只收紧信任根侧 ACL 策略 | 保持闭合 |
| F013 | MEDIUM | 保持 r62 结论；r63.5 `:3970`–`:3973` 补 redactor 与 canary | 保持闭合 |
| F014 | MEDIUM | 保持 r62 结论；r63.6 `:3975`–`:3982`、r63.9.2 `:3997` | 保持闭合 |
| F015 | HIGH | 保持 r62 结论；r63.3 `:3942` 固定 P1–P3 启动顺序 | 保持闭合 |
| F016 | MEDIUM | r63.9.4 `:4000`–`:4018`（13 行生产写集 + 测试建议） | 保持闭合 |
| F017 | HIGH | r63.9.4 `:4000`、`:4018`；r63.9.5 `:4020` 第（2）项 | 范围外（Task-Planner）；`tasks.md` 一行未改 |
| F018 | BLOCKER | r63.8 `:3987`–`:3992`（签名载荷、审计 sink 模式、sequence 基准、回填规则） | 闭合 |
| F019 | HIGH | 保持 r62 结论；r63.9.2 `:3997` 与 r63.9.5 `:4020` 第（4）项 | 保持闭合 |
| F020 | HIGH | 保持 r62 结论；r63.9.4 `:4014` 维持预检错误 → 退出码 2 | 保持闭合 |
| NEW-R02-06 | MEDIUM | r63.2 `:3937`（root principal 登记 + verifier／producer 与 root 机核分离） | 闭合 |
| NEW-R02-07 | MEDIUM | r63.1 `:3930`（禁止默认 `encoding/json` 承担 canonicalization） | 闭合 |
| NEW-R02-08 | MEDIUM | r63.2 `:3936` 第（7）项（逐字节等于 `tool.executable_sha256`） | 闭合 |

## 4. 安全侧 r02 finding → r63 落点 → 判定

| ID | 等级 | r63 章节／行号 | 判定 |
|---|---|---|---|
| SEC-F001 | BLOCKER | r63.1 `:3928`–`:3931`；向量 `:4039`–`:4048` | 闭合（设计层）；独立复算仍属复审职责 |
| SEC-F002 | BLOCKER | r63.2 `:3933`–`:3937`（organizations／appointments／公钥唯一／单 principal 单密钥／`affiliations` 受控）；r63.9.3 `:3999`（6） | 闭合 |
| SEC-F003 | BLOCKER | r63.3 `:3939`–`:3941`（进程内 mono 高水位 + 摘要／身份快照） | 闭合；跨重启回拨不保证，残余风险登记 `:4020` 第（5）项 |
| SEC-F004 | HIGH | r63.5 `:3970`–`:3973`（九处清零、错误只带分类与长度、禁 `net/url` 解析带密码 DSN、canary） | 闭合 |
| SEC-F005 | BLOCKER | r63.6 `:3975`–`:3980`（`PT_INTERP` 绑定的 cache／conf 闭包、ENV 白名单、maps 复核、network 绑定） | 闭合（设计层）；Linux 实测 NOT RUN |
| SEC-F006 | HIGH | r63.7 `:3983`–`:3985`（CapBnd／NNP／setns／seccomp = unsupported → closed） | 闭合（能力降级为 unsupported，不声称隔离） |
| SEC-F007 | BLOCKER | r63.9.5 `:4020` 第（3）项 | 仍开放：关闭条件需 Linux 证据，保持 NOT RUN |
| SEC-F008 | LOW | r63.1 `:3928`、r63.6 `:3975`、r63.8 `:3987` | 闭合（精度与示例） |
| N001 | BLOCKER | r63.8 `:3987`–`:3992`（验证方签名、外部见证推进 `sequence_floor`、producer 独立、`independent_read_path` 边界）；r63.2 `:3937` | 闭合 |
| N002 | HIGH | r63.8 `:3992`（`O_WRONLY`、`O_APPEND`、`O_CLOEXEC`、`O_NOFOLLOW`，flock、0600、fsync、短写） | 闭合 |
| N003 | HIGH | r63.8 `:3987`（`tool.receipt` 仅为策略，实际回执为 12 字段 wire） | 闭合 |
| N004 | HIGH | r63.4 `:3968`（`ReadAt`／读后 `Seek(0,0)`／传递前断言 offset=0） | 闭合 |
| N005 | BLOCKER | r63.9.5 `:4020` 第（1）项 | 范围外（ADR／任务卡决策）；本轮只登记 |
| N006 | HIGH | r63.7 `:3983`–`:3985`、r63.6 `:3980`（`network.mode` 只允许 `none`／`self_attested`） | 闭合（unsupported／自证，不声称已隔离） |
| N007 | LOW | r63.6 `:3982`（信任根与批准文件挂载属性、逐级 lstat、ACL 拒绝条件） | 闭合 |

## 5. 未闭合项、范围外项与 NOT RUN 边界

- **N005（范围外）**：`decisions/framework.md:1129` 的 r07 blocked 陈述与 `tasks.md:3197` 的"门禁解除"互相冲突。建议架构决策方先统一 r07 状态，再由 Task-Planner 对账任务卡；本设计不代改 ADR 或任务卡，对账前不得以任一方文字主张 S4.24a 已解锁。
- **F017（范围外）**：`tasks.md:3191`–`:3195` 仍是 4 文件写集，与 r63.9.4 的 13 行建议写集不一致；写集更新属 Task-Planner。本轮未改 `tasks.md` 任何一行，也未勾选任何任务。
- **SEC-F007（NOT RUN）**：Linux 目标／CI 上的 host、ACL、mount、loader、fd-exec、maps、network 与 witness 实测均未运行；Windows 只覆盖解析与语义层。`//go:build linux` 的 skip 与无匹配测试的 exit 0 都不等于 PASS。
- **真实输入全部缺失**：真实信任根、Ed25519 密钥、组织／任命记录、两条批准记录、外部见证、审计 sink 与生产工具均不存在。
- **时钟**：高水位只在进程内单调，跨进程／重启后的回拨不保证被检测，属已登记残余风险。
- **见证与审计**：本地 `witness_path` 与审计投影可被回滚，只有外部见证才能验证 head 完整；外部见证缺失时 `VerifyRestored` 返回 `ErrAdmissionUnsupported` 并保持 closed。
- **pre-exec 能力**：CapBnd／NoNewPrivs／setns／seccomp 在本设计范围内 unsupported，`ErrAdmissionUnsupported` → closed；不声称已达成任何隔离。
- **默认仍是 closed**：`recoveryClosedRestoreDriver` 仍是默认，生产 `VerifyRestored` 与 `enter`／`resume` 未解锁，生产 restore 不可用。

## 6. 验证、证据与声明

### 6.1 修订轨迹与哈希

| 阶段 | 字节 | 行数 | SHA-256 |
|---|---:|---:|---|
| r62（修订前） | 936613 | 5317 | `EA880660B8D43AFD6FB227812C662BB5A4581B39F4235F3DAB8E221F5F9FDC9A` |
| r63 首轮（插入 r63 增量 + 就地替换向量） | 975049 | 5416 | `48AA096C16D4941FF8D22C95D2F0448579B3C840D3144C79CE20C2E9B62D9914` |
| r63 本轮补充（r63.8 取代范围句、r63.9.3 第（6）（7）项） | 976149 | 5416 | `3B8BEA8C35A12F75B36091969DB15B9590FBDD16B0429D59328ABB9FA3BC027C` |

最后一次补充只做两处精确子串替换，分别落在 `:3992` 与 `:3999`，不增删行，故行数保持 5416、r61 起点保持 `:4230`；`design.md:4230` 之后的字节因此未被触碰。r61 标题行与总行数在替换后已重新断言。

### 6.2 复算方法

字节数用 `[IO.File]::ReadAllBytes` 长度、行数用 `[IO.File]::ReadAllLines(..., UTF8).Count`、摘要用 `Get-FileHash -Algorithm SHA256`。两个 golden vector 用 §2.2 的 Python 逻辑重新解析、递归排序、紧凑序列化、UTF-8 编码并取 SHA-256，同时断言重排后字节与文中字节逐字节相等、递归键序检查为空集。

### 6.3 范围与只读声明

本轮只写两个文件：`design.md`（r62 → r63）与本报告。未写代码，未改 `tasks.md`、ADR 正文、`cmd/**`、`internal/**`、测试或任何 `.go`；未执行 Git 写、删除、移动或归档；工作区既有 295 条未提交改动原样保留。

**作者复算不等于独立验证。本报告不构成评审通过或任务关闭。未写代码、未改 `tasks.md`/ADR、未勾选任务、未声称 PASS。**