# B13J-S4.24a 架构侧独立评审报告（review-arch-r01）

## 0. 元信息与边界

- 角色：`add-reviewer`（架构侧），非被评审设计作者。
- 主方法：`.codex/skills/add-code-review/SKILL.md` 完整有效正文；规则：`.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`，并按对象加载 `coding-standards`（仅代码消费者核对）、`decisions-guide`、`design-doc-writing-style`。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（只读核对）。
- 授权与只读边界：本轮只评审、只读任意文件；只新建本报告；未修改 `design.md`、ADR、`cmd/**`、`internal/**`、`tasks.md`，未做任何 Git 写、删除或移动。报告目录 `review-arch-r01/` 由本评审新建（此前不存在）。
- 工作树状态：`git status --porcelain` 显示多个并行角色未提交改动（含 `design.md`、`framework.md`、`tasks.md`）。本评审只固定并核对下述文件版本/哈希；不据此评判其他改动，也不回退任何他人改动。

### 0.1 固定对象

| 项 | 实测值 |
|---|---|
| 对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| 字节 / 行 | 876660 B / 5109 行 |
| SHA256 | `EF55DC9272F24C7626F028F705B4322B4E33DCFA6D0197545015CA7DE4FE5811` |
| revision | `identity-auth-design-ir01-r61`（`:5`） |
| r60 改前基线 | `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D`（854991 B） |
| 评审增量 | 正文 `:3925`–`:4087`；一致性同步行 `:38`、`:83`、`:3611`、`:3878`、`:3882`、`:3915` |
| 作者证据 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r01/report.md`，18793 B / `B78CA378F6050B9494BEBE18ABC6C11171E6DB9ECCBB594206EE7C5AA7D4C6FA` |
| 上轮阻断证据 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/report.md`，10670 B / `815CC215DB615B77E176FBEA8CA45079235A0082854F500A50682B47C3443763`（B1–B6） |

未被 r61 修改的段落不作为本次对象；但凡 r61 与之矛盾，按 r61 的 finding 处理（见 F004、F008、F019）。

### 0.2 两轮实际检查范围

**第一轮（对象内一致性与可编码性）**：逐行读取 r61 正文 `:3925`–`:4087` 与 6 个同步行；核 strict JSON 键序/字段/映射、`approval_digest` 计算、`RestorePlan` 命中关系、读取时机、有效期/撤销、失败分类、`RestoreSourceClaim`/`RestoreSourceVerifier`、root 装配顺序、DSN 载体与 stdin 形状、平台文件边界、负向矩阵、退出码、影响面与不变量。另读 r60 的 11 端口表 `:3601`–`:3618` 与 restore 软件合同 `:3874`–`:3888` 作为一致性基线。

**第二轮（真实消费者与上游依据）**：ADR-0031 r07 `framework.md:1082` 起，重点 `:1113`–`:1169`（四项结论、不变量 1–6、约束 8）、`:1172`–`:1176`（前置节点、登记≠启动）；ADR-0030 约束 9/10 与 bundle 四端口；ADR-0027 的 `Prepare`/`Begin` 编排；ADR-0032 的 11 端口与 `exclusive` 来源；ADR-0019 的 closed/CAS 语义；`tasks.md:3184`–`:3213`（S4.24a 写集、验收命令、负向、关闭条件、停止条件）；代码消费者：`cmd/han-server/recovery_restore.go`、`recovery_restore_test.go`、`recovery_runtime.go`、`recovery_sources.go`、`recovery_config.go`、`main.go`、`lifecycle.go`、`internal/recovery/runtime.go`、`admission.go`、`replay_status_v1.go`、`installation.go`；作者证据与上轮阻断报告。

未做：`go test`/`go build`/`go vet`/lint、Linux 真实 host/ACL/mount/loader/fd-exec、真实批准记录/工具/备份/回执、安全侧报告正文。详见 §4。

## 1. 六问结论

### Q1：r61 是否真正闭合 B1／B2／B3／B5／B6

| 阻断项 | 结论 | 能否只依据 design 写出唯一实现 | 直接反例（缺哪个字段/签名/时机/失败分类） |
|---|---|---|---|
| B1 批准记录 | **未闭合** | **否** | strict JSON 只固定顶层/对象键序，未固定嵌套对象键序、数组顺序、HTML 转义、数字格式、时间严格度；`approval_digest` 与 `RestorePlan.SourceApprovalDigest` 的前缀要求互相矛盾；`source.source_schema_digest` 没有与 `RestorePlan` 的命中关系；Windows 上 owner/root/0700 与“真实可触发”冲突（F001/F002/F010/F011）。 |
| B2 来源核验 | **未闭合** | **否** | `RestoreSourceClaim` 只有 `ArtifactDigest`/`ArtifactBytes`，没有 `ArtifactPath`/fd/handle；接口没有调用时机、没有 plan→claim 映射、没有由谁在 `Prepare`/`Replace` 的哪一步调用；`newControlledRestoreDriver` 只加 verifier 形参，生产的 `authority`/`ownership` 无设置路径（F005/F006/F019）。 |
| B3 目标凭据 | **未闭合** | **否** | `dsn` 由 `string` 改为 `[]byte` 后 `json.Marshal` 默认输出 base64，与“线上形状仍是字符串”直接冲突；DSN 结构校验（scheme、query、空值/空白、非法 UTF-8）未定义（F007/F013）。 |
| B5 平台边界 | **部分闭合** | **否** | 平台文件表有方向，但 `tool` 记录没有 `ld.so.cache` 路径/摘要字段，strict JSON 又禁止额外键；`validateRestoreHost`/`execRestoreTool` 只给文件名与职责，无签名；`!linux` 的 exec 门禁会破坏现有 Windows 子进程测试；专用执行身份切换仍未定义（F003/F005/F014/F008）。 |
| B6 负向矩阵 | **部分闭合，卡不可关闭** | **否** | (d)(e)(f) 在本机 Windows 结构性不可达并标 NOT RUN；tasks 关闭条件 (4)(5) 无 NOT RUN 豁免；外部见证/验签的核验操作与审计字段未定义；(a)(b) 在 Windows 的“真实可触发”被 POSIX owner/root/0700 约束反证（F018/F009/F010）。 |

**总答**：不能。至少 digest canonicalization、verifier 对实际制品的访问路径、生产 `authority`/`ownership` 接线与执行身份、DSN `[]byte` 的 JSON 编码、外部回执验签五处必须由实施者自行发明；这不满足“唯一实现”的评审目标。

### Q2：与已接受口径是否冲突

| 依据 | 结论 | 说明 |
|---|---|---|
| ADR-0031 r07 四项结论 / 不变量 1–6 / 约束 8 | **closed/default 未放宽；但判据未完全编码** | `recoveryClosedRestoreDriver` 仍是默认（`recovery_runtime.go:596`–`:599`）；不变量 1 的“复核人独立于执行者”缺执行者身份字段；不变量 2 的工具记录缺 loader 字段与 tool digest 消费者；不变量 6 的验签/外部见证未编码（F018）。因此不是“与 ADR 冲突”，而是 r61 声称只补实现形态，却未覆盖这些 ADR 判据。 |
| ADR-0030 / anchor `sources` 端口 | **不冲突** | 未扩展 `recovery.InstallationSourceVerifier`，未改变 bundle 的 `Sources` 语义；方向正确（见 Q4）。 |
| ADR-0027 的 `Prepare`→`Begin` 编排 | **部分冲突** | `internal/recovery/runtime.go:1607` 的 `Prepare` 确在 `:1617` 的 `Journal.Begin` 前；但 r61 的“唯一预检入口/顺序”与现有 `planRecoveryStartup`/`main.go:629` 的晚置解析未对齐（F015）。 |
| ADR-0032 的 11 端口 / 唯一 root 装配 | **文字冲突** | `design.md:3603` 明文“未在本表内登记的依赖不得注入 production root”；r61.3 允许一个新 verifier 进入 root 装配链。实质可用“driver 构造期内部依赖≠端口”解释，但 r60 原文未同步（F004）。 |
| ADR-0019 | **未发现冲突** | r61 未改 CAS head anchor、keyring、closed 语义。 |
| r60 的 11 端口与唯一 production root | **实质可自洽、文本不自洽** | 新 verifier 不加入 `recoveryRootInputs`、不称第 12 端口，符合“端口清单不变”；但“依赖不得注入”的措辞需要一处限定，否则审计上可被判为越界。 |

### Q3：是否引入第二 authority／fallback／隐式默认／把 closed 变 open

- **未发现新增 fallback 或隐式默认**：r61.4 明确禁止回退 `HAN_DATABASE_DSN`、禁止从 pool 反推、禁止从 manifest/批准文件读 DSN（`design.md:4049`）。`recoveryClosedRestoreDriver` 的定义、默认与回退逐字未变：`cmd/han-server/recovery_runtime.go:596`–`:599` 仍是 `restore := inputs.restore; if restore == nil { restore = recoveryClosedRestoreDriver{} }`；`cmd/han-server/recovery_sources.go:557`–`:574` 三方法仍返回 `ErrAdmissionClosed`。
- **未把 closed 改 open**：`nil` verifier → `ErrAdmissionUnsupported`（`design.md:4043`），没有“默认放行”的措辞。
- **第二 authority**：`RestoreSourceVerifier` 是 restore 来源的信任判定点，不是 phase/permit authority；`authority` 仍应是 `recovery.Runtime`（`recovery_restore.go:169`–`:172`、`:301`–`:308`）。但它是否“独立于执行者/root”无法由现有接口证明（F005）；因此“无第二 authority”目前只在命名与职责上成立，证据上未闭合。

### Q4：端口放置取舍

**成立**。`recovery.InstallationSourceVerifier` 属于 anchor adapter bundle 的四端口之一（`cmd/han-server/recovery_sources.go:427`、`:450`、`:529`；ADR-0030 约束 10 见 `framework.md:1057`），首期 registry 为空即 `recoveryClosedPort`；把 restore 批准塞进该接口会让 restore 可用性反向依赖 anchor registry，并把两种信任输入混成一个契约。新增 `internal/recovery/restore_source.go` 避免了这一耦合，且没有改变 anchor `sources` 的语义。

但两点保留：
- `RestoreSourceVerifier`/`RestoreSourceClaim` 是 `internal/recovery` 的新导出 API；r61.8 的“不改既有公开契约”只能读作“不改既有符号”，不能读作“无 API 增量”，实施者需知道这是新增面。
- 接口本身不足以核验实际制品（F005），所以“放置正确”不等于“接口闭合”。

### Q5：装配顺序是否唯一不可绕过

**方向正确，但不唯一、不可仅凭现有文本实施**：

1. **预检入口/顺序**：r61 要求 `runComposed` 在 `loadStartupConfig` 后、`parseDatabase` 前调用唯一预检（`design.md:4006`）；现有 `planRecoveryStartup` 在 `main.go:629`，晚于 `main.go:442` 的 `parseDatabase`，且 `recovery_config.go:58`–`:83` 的键集、`:222`–`:241` 的 `readAction` 都不含两个新键。新预检与既有 `planRecoveryStartup` 的职责边界未定义（F015）。
2. **构造**：r61 只要求 `newControlledRestoreDriver` 增加 `RestoreSourceVerifier` 形参（`design.md:4043`）；现有构造签名是 `(source, target, tool, dsn)`（`recovery_restore.go:203`），`authority`/`ownership` 是构造后由测试直接赋值（`recovery_restore_test.go:206`、`:229`）。生产 root 何时、由谁设置这两个字段未定义（F006）。
3. **失败语义**：r61 说“任一依赖为 nil，`inputs.restore` 保持 nil，现有 `nil → closed` 分支不变”（`design.md:4041`、`:3611`）；但 `enter`/`install` 在 `newRecoveryRootAssembly` 先经过 `recoveryEnterInstallMissingPorts`（`recovery_runtime.go:527`–`:532`、`:456`–`:492`），`restore == nil` 会先返回 `ErrAdmissionClosed`，不会进入 `:596`–`:599` 的 fallback。`resume` 才会走 fallback。描述与真实代码只在 `normal`/`verify`/`resume` 成立（F019）。
4. **DSN**：`privateRestoreInput(p, dsn string)` 用 `json.Marshal` 生成 `{"plan":...,"target_connection":{"dsn":string}}`（`recovery_restore.go:71`–`:80`）；改成 `[]byte` 会变成 base64（F007）。
5. **测试影响面**：`recovery_restore_test.go:201` 是唯一直接构造点；`:206` 设 `ownership`、`:229` 设 `authority`；`Prepare` 在 `:216` 检查 `ownership == nil`，`checkPermit` 在 `:302` 检查 `authority == nil`，`VerifyRestored` 在 `:378` 检查 `ownership == nil`。“只补测试替身实参”不足以覆盖这些后置赋值（F006）。

因此：**装配顺序的概念方向成立，但“唯一不可绕过”不成立**；至少需要补齐 F006、F015、F019，并给出 `authority`/`ownership` 的生产设置与执行身份。

### Q6：(d)(e)(f) 标 NOT RUN 是否构成 S4.24a 关闭缺口

**构成实质缺口，不能以 NOT RUN 关闭 S4.24a。明确结论：必须换 Linux 环境取证，或拆卡；设计层最多算“测试位置已定义”的部分闭合。**

依据：
- `tasks.md:3208` 的负向 (d) 要求证明无 shell、无附加 argv、无 stdout receipt/exit=0 自证；(e) 要求把外部见证/验签等价操作与审计字段纳入关闭证据；(f) 要求核对 `/etc/ld.so.preload`、`ld.so.cache` 与 RPATH/RUNPATH。r61.6 把这三项的真实触发标为 NOT RUN（`design.md:4079`–`:4081`）。
- `tasks.md:3209` 关闭条件 (4) 要求负向测试证明无 shell/无附加 argv/无自证成功；(5) 要求 9 条验收命令通过。`tasks.md:3196`–`:3206` 的 9 条命令全部是 Windows 本地 `gofmt`/`go build`/`go vet`/`go test`，没有 `GOOS=linux go build`、没有 Linux `go test`、没有 Linux host/fd-exec 测试；NOT RUN 不满足“通过”。
- 现有 `restoreProcessTool.imageConfiguration` 在 Windows 上只允许 `SystemRoot`/`WINDIR`（`recovery_restore.go:461`–`:466`），而 `recovery_restore_test.go:187` 正在 Windows 上真实运行子进程；r61.5 又要求 `!linux` 的 `execRestoreTool` 返回 `ErrAdmissionUnsupported`（`design.md:4062`、`:4070`）。两者需要在“测试替身/软件合同测试”与“生产 exec gate”之间分流，r61 未定义（F008）。

## 2. Findings

> 严重级定义：BLOCKER = 无法形成唯一可实施契约或必然违反既有消费者；HIGH = 会导致实施返工、验收不可达或交付范围不成立；MEDIUM = 可实现但存在多解/影响面遗漏/审计歧义。所有 finding 均针对 r61 增量或其同步行；未改段落仅在 r61 与其矛盾时引用。
### 2.1 清单

| ID | 严重级 | 主题 | 阻断实施 |
|---|---|---|---|
| `B13J-S424A-ARCH-R01-F001` | BLOCKER | 批准 JSON 的 digest canonicalization 不唯一 | 是 |
| `B13J-S424A-ARCH-R01-F002` | HIGH | `approval_digest` 裸 hex 与 `sha256:` 前缀契约冲突 | 是 |
| `B13J-S424A-ARCH-R01-F003` | HIGH | `tool` 记录缺 loader 字段；`tool.approval_digest` 无消费者；`artifact_bytes` 映射缺失 | 是 |
| `B13J-S424A-ARCH-R01-F004` | HIGH | r61.3 与 r60「未登记依赖不得注入」文字冲突 | 否（需文本限定） |
| `B13J-S424A-ARCH-R01-F005` | BLOCKER | `RestoreSourceVerifier` 无法独立核验实际制品；无调用时机与 plan→claim 映射 | 是 |
| `B13J-S424A-ARCH-R01-F006` | BLOCKER | 构造签名与生产 `authority`/`ownership` 接线缺失 | 是 |
| `B13J-S424A-ARCH-R01-F007` | BLOCKER | `dsn []byte` 经 `json.Marshal` 变 base64；清零时点不唯一 | 是 |
| `B13J-S424A-ARCH-R01-F008` | HIGH | `!linux` exec 门禁与现有 Windows 子进程测试冲突 | 否（需分流） |
| `B13J-S424A-ARCH-R01-F009` | HIGH | 负向 (d)(e)(f) 标 NOT RUN 不能关闭 S4.24a (4)(5) | 是（关闭条件） |
| `B13J-S424A-ARCH-R01-F010` | HIGH | Windows 上 owner/root/0700 不可达，与「真实可触发」冲突 | 否（需分层） |
| `B13J-S424A-ARCH-R01-F011` | HIGH | 批准快照/时间/撤销在 `Prepare`→`Replace` 间不闭合 | 是 |
| `B13J-S424A-ARCH-R01-F012` | MEDIUM | mountinfo/ACL/父目录/loader 算法未定义 | 否 |
| `B13J-S424A-ARCH-R01-F013` | MEDIUM | DSN 结构校验未定义 | 否 |
| `B13J-S424A-ARCH-R01-F014` | MEDIUM | `validateRestoreHost`/`execRestoreTool` 签名及与 `restoreProcessTool` 关系未给 | 否 |
| `B13J-S424A-ARCH-R01-F015` | HIGH | 新配置键与唯一预检入口未接入现有配置/启动顺序 | 是 |
| `B13J-S424A-ARCH-R01-F016` | MEDIUM | 文件/影响面清单不完整 | 否（与 F017 合并阻断） |
| `B13J-S424A-ARCH-R01-F017` | HIGH | tasks 4 文件写集无法实现 r61 新范围 | 是 |
| `B13J-S424A-ARCH-R01-F018` | BLOCKER | ADR-0031 不变量 6 的外部回执验签/见证未编码 | 是 |
| `B13J-S424A-ARCH-R01-F019` | HIGH | `inputs.restore == nil → closed` 对 `enter`/`install` 不成立 | 否（需分动作说明） |
| `B13J-S424A-ARCH-R01-F020` | HIGH | `ErrAdmission*` 一律退出码 2 与现有 `run` 错误映射不一致 | 否（需明确入口） |

### B13J-S424A-ARCH-R01-F001 — BLOCKER — 批准记录 digest 的 canonicalization 不唯一

- **定位**：`design.md:3931`–`:3984`（strict JSON 与 `approval_digest` 计算）、`design.md:4004`（与 `RestorePlan` 逐字节命中）。
- **问题**：只固定了顶层 `version`／`source`／`tool` 的键序，未定义 digest 规范化所需的全部字节规则：嵌套对象（`issuer`、`revocation`、`reviewer`、`acl`、`mount`、`receipt`）的键序是否同样固定；字符串采用 UTF-8 原文还是 `\uXXXX`、是否做 HTML 转义（Go `encoding/json` 默认转义 `<`、`>`、`&`）；`artifact_bytes`／`st_dev`／`st_ino`／`sequence_floor` 的整数文法（前导零、`-0`、指数）；`environment` 数组是否有序；未适用字段是空串、空数组还是省略；以及 `source`／`tool` 两个对象的精确键集（示例展示的并集不等于「无额外键」下每个对象必须有的键集）。
- **影响**：部署方生成 `approval_digest` 与独立 verifier 重算时可能得到不同字节；`RestorePlan.SourceApprovalDigest` 的「逐字节相等」无法稳定成立。实施者必须自行选择一种规范化算法，B1 仍是开放契约。
- **依据**：`design.md:3984` 只写「按上列固定键序序列化为紧凑 UTF-8 JSON」；`design.md:4004` 要求 `SourceApprovalDigest` 与 `source.approval_digest` 逐字节相等；未给出 golden bytes 或测试向量。
- **建议**：明确采用 RFC 8785（JCS）或给出等价的逐字节语法：对象键集与键序、字符串转义、整数文法、数组顺序、空值表示；为 `source` 与 `tool` 各给一条固定测试向量，并把向量纳入验收。
- **阻断实施**：是。没有唯一 canonical bytes，B1 的批准摘要无法由两个独立实现一致复算。

### B13J-S424A-ARCH-R01-F002 — HIGH — `approval_digest` 的裸 hex 与 `sha256:` 前缀契约冲突

- **定位**：`design.md:3984`、`design.md:4004`；`internal/recovery/admission.go:765`–`:768`；`internal/recovery/replay_status_v1.go:33`、`:311`–`:313`；`cmd/han-server/recovery_config.go:235`–`:237`；`cmd/han-server/recovery_restore.go:492`。
- **问题**：r61 定义 `approval_digest` 为「SHA256 小写十六进制」（裸 64 hex），并要求它逐字节等于 `RestorePlan.SourceApprovalDigest`；但 `validateRestorePlan` 用 `sha256DigestPattern`（`^sha256:[0-9a-f]{64}$`）校验 `SourceApprovalDigest`，裸 hex 会被判 `ErrAdmissionBinding`。同一项目内 `HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 用裸 hex（`recovery_config.go:235`–`:237`），而运行时比较用 `"sha256:"+hex`（`recovery_restore.go:492`），说明两种表示都真实存在，r61 未指定批准摘要采用哪一种。
- **影响**：按字面实现时，任何批准文件都无法通过 `RestorePlan` 校验；实施者若自行补前缀，又违反「逐字节等于」的文本。B1 必然返工。
- **依据**：上述四处的正则/拼接方向相反；`design.md:3984` 未写 `sha256:` 前缀。
- **建议**：统一为 `sha256:<64 lowercase hex>`（与 `RestorePlan` 和 `RestoreReceipt` 一致），或明确在解析边界把裸 hex 规范化为带前缀并同步修改「逐字节相等」表述；增加裸 hex 被拒/被规范化的单测。
- **阻断实施**：是（B1 无法同时满足两处契约）。

### B13J-S424A-ARCH-R01-F003 — HIGH — `tool` 记录缺 loader 字段，`tool.approval_digest` 无消费者，`artifact_bytes` 映射缺失

- **定位**：`design.md:3955`–`:3979`（`tool` 对象字段）、`design.md:4064`（loader 检查）、`design.md:3995`（字段映射表）。
- **问题**：r61.5 要求「动态 ELF 的 `ld.so.cache`（路径登记在工具记录）SHA256 与记录相等，静态 ELF 要求记录中该摘要为空」，但 `tool` 对象没有 `ld_so_cache_path`／`ld_so_cache_sha256`（或等价）字段；strict JSON 又规定「无额外键」，实施者无法在不违反格式的情况下登记该路径与摘要。其次，`tool.approval_digest`（`:3979`）被计算但没有任何消费者：`RestorePlan` 只有 `SourceApprovalDigest`，`RestoreSourceClaim` 也只有 `SourceApprovalDigest`，工具记录摘要是否被核验未定义。再次，`source.artifact_bytes` 出现在示例（`:3949`）且 verifier 要求与批准记录一致（`:4039`），但字段映射表只在 `:3995` 映射 `artifact_digest`，没有 `artifact_bytes` 行。
- **影响**：B5 的 loader 检查不可编码；工具记录可被替换而不被摘要核验；`ArtifactBytes` 的批准来源不唯一，实施者需自行决定读哪个字段。
- **依据**：`design.md:3955`–`:3979` 的 `tool` 键集无 loader 字段；`design.md:4064` 却要求「记录中该摘要」；`design.md:3995` 无 `artifact_bytes` 映射。
- **建议**：在 `tool` 记录增加 `ld_so_cache_path`、`ld_so_cache_sha256`（静态 ELF 为空串）并加入 strict 键集；定义 `tool.approval_digest` 的消费者（例如新增 `RestorePlan.ToolApprovalDigest` 或 verifier claim 字段）；补 `artifact_bytes` 映射行与相等校验。
- **阻断实施**：是（B5 与 B1 的部分字段无法落地）。

### B13J-S424A-ARCH-R01-F004 — HIGH — r61.3 与 r60「未登记依赖不得注入 production root」文字冲突

- **定位**：`design.md:3603`（r60 唯一装配清单约束）、`design.md:4041`（r61.3 新 verifier 进入 root 装配）、`design.md:4043`、`design.md:4087`。
- **问题**：r60 原文说「本表是唯一 production root 装配清单……未在本表内登记的依赖不得注入 production root」。r61.3 让新的 `RestoreSourceVerifier` 进入 root 的唯一装配链，并称其为「driver 构造期内部依赖：不登记进 r60 的 11 端口清单、不加入 `RuntimeDependencies`、不称第 12 端口」。把「端口」与「构造期内部依赖」区分开在工程上可以成立，但 r60 用的是「依赖」，且未说明 `recoveryRootInputs` 与 `RuntimeDependencies` 的边界；审计者可以据此判定 r61 违反已接受的 ADR-0032／r60 合同。
- **影响**：同一设计出现两种可辩论的合规解释；实施者可能把 verifier 放进 `recoveryRootInputs`（越界），也可能因不敢注入而无法构造 driver。Q2 的「与 r60 口径是否自洽」不能给出无保留的「是」。
- **依据**：`design.md:3603` 与 `design.md:4041` 的原文直接对撞；`design.md:4087` 又声明「不变：r60 的 11 端口清单」，但没有同步限定「依赖」一词。
- **建议**：在 r60 `:3603` 增加一句窄例外：`driver` 构造期内部依赖不是端口、不得进入 `recoveryRootInputs`／`RuntimeDependencies`、不得独立替换或单独配置；或把该 verifier 显式登记为第 12 个端口并走 ADR 修订。二者择一，不能保留现状。
- **阻断实施**：否（文本限定即可），但在限定前属 HIGH 合同歧义。

### B13J-S424A-ARCH-R01-F005 — BLOCKER — `RestoreSourceVerifier` 无法独立核验实际制品，且无调用时机与 plan→claim 映射

- **定位**：`design.md:4015`–`:4032`（接口定义）、`design.md:4037`–`:4039`（语义与四字段）、`design.md:4041`（装配）。
- **问题**：`VerifyRestoreSource(ctx, claim RestoreSourceClaim)` 只收到摘要、字节数、引用等声明值；`RestoreSourceClaim` 没有制品路径、已打开 fd/handle、批准记录快照或 source installation 证据。r61 要求「与批准记录及实际制品一致」并「不得用 `RestorePlan` 自身字段互证」，但未定义 verifier 如何取得实际制品：是由构造函数预先持有 no-follow fd，还是在调用时传入路径，还是复用 `restoreSource.OpenApproved`？也未定义由 `Prepare`、`Replace` 还是 `VerifyRestored` 调用、调用发生在 `OpenApproved` 之前还是之后、claim 各字段从 `RestorePlan`／`RunBinding` 如何映射，以及 verifier 错误如何映射到 `ErrAdmission*`。`SourceSchemaDigest` 还要求读取「备份制品内 source installation 的 schema 摘要」，接口没有制品句柄，无法完成。
- **影响**：实施者必须自行发明 B2 的核心信任边界；最省事的实现会只比较 claim 与 plan，构成 r61 自己禁止的「plan 自身字段互证」。B2 未闭合。
- **依据**：`design.md:4015`–`:4028` 字段集无制品句柄；`design.md:4030`–`:4032` 只有 claim 参数；`design.md:4039` 明确要求实际制品与 schema 证据；全文无调用点。
- **建议**：定义 verifier 的构造形态（注入已核 no-follow fd／不可变快照）、调用点（建议在 `OpenApproved` 成功后、`Prepare` 返回前，且早于 `Journal.Begin`）、claim 的逐字段来源、错误映射，以及 `SourceSchemaDigest` 的读取路径；或者把接口改为显式接收 `*os.File`／`RestorePlan`＋handle。
- **阻断实施**：是。没有这些，B2 无法写出唯一实现。

### B13J-S424A-ARCH-R01-F006 — BLOCKER — 构造签名与生产 `authority`/`ownership` 接线缺失

- **定位**：`design.md:4041`–`:4043`；`cmd/han-server/recovery_restore.go:203`–`:208`、`:216`、`:301`–`:308`、`:378`；`cmd/han-server/recovery_restore_test.go:201`、`:206`、`:229`。
- **问题**：现有构造函数是 `newControlledRestoreDriver(source, target, tool, dsn)`；r61 只说「增加一个 `RestoreSourceVerifier` 形参」。但 driver 的 `Prepare` 在 `d.ownership == nil` 时返回 `ErrAdmissionUnsupported`（`:216`），`checkPermit` 在 `d.authority == nil` 时返回 `ErrAdmissionClosed`（`:302`），`VerifyRestored` 又在 `d.ownership == nil` 时拒绝（`:378`）。现有测试在构造后直接赋值 `f.driver.ownership = f.world.lease`（`:206`）和 `f.driver.authority = r`（`:229`）；生产 root 没有对应设置路径。r61 声称「任一依赖为 nil → `inputs.restore` 保持 nil → 走 closed fallback」，但实际上构造函数只检查 source/target/tool/dsn（以及新增 verifier），不检查 `authority`/`ownership`；它们为 nil 时 driver 仍被构造并赋给 `inputs.restore`，随后在 `Prepare` 失败，语义与声明不同。更麻烦的是 `authority` 接口由 `recovery.Runtime` 实现，而 Runtime 又由 `newRecoveryRootAssembly` 用 `Restore: inputs.restore` 构造，形成「driver 需要 runtime，runtime 需要 driver」的循环。
- **影响**：生产 driver 要么不可用（`ownership`/`authority` 永远 nil），要么需要未声明的 setter／late binding；测试通过不等于生产接线成立。实施者必须自行发明生命周期，B2／装配合同不闭合。
- **依据**：上述构造签名、三处 nil 检查和测试后置赋值；`design.md:4043` 只提「增加一个形参」与「只改测试装配」。
- **建议**：明确 constructor 的完整形参（是否包含 `authority`/`ownership` 或工厂），或定义 root 在 Runtime 构造后如何安全注入二者；为每个 nil 依赖指定错误分类；补充生产装配单测，证明 `inputs.restore` 非 nil 时 `Prepare` 不会因 `ownership`/`authority` 为 nil 而失败。
- **阻断实施**：是。生产接线无法仅依据 design 唯一写出。

### B13J-S424A-ARCH-R01-F007 — BLOCKER — `dsn []byte` 经 `json.Marshal` 变 base64，清零时点不唯一

- **定位**：`design.md:4053`；`cmd/han-server/recovery_restore.go:71`–`:80`、`:341`–`:364`、`:649`–`:697`。
- **问题**：`privateRestoreInput` 的 `Target` 结构字段是 `DSN string`（`:71`–`:80`）。r61 要求「把 `target_connection` 的 `dsn` 由 `string` 改为写后可清零的 `[]byte`」，同时声称「不改变 stdin JSON 的线上形状（仍是字符串）」。Go 的 `encoding/json` 对 `[]byte` 默认输出 base64 字符串，直接改字段类型会得到 `{"dsn":"cG9zdGdyZXM6..."}`，与子进程期望的原始 DSN 字符串不符；若在 marshal 前转回 `string`，又产生不可清零的副本。r61 没有指定自定义 `MarshalJSON`／`json.RawMessage` 方案。清零方面：`execute` 用 `defer clear(input)`，实际在 `tool.Run` 返回后才清零；`runChild` 在 goroutine 中写管道（`:675`–`:682`），设计说「写入管道后」清零，但没有定义由谁、在哪个同步点清零；`d.dsn` 若改为 `[]byte`，`Close` 清零与并发 `Run` 的互斥也未说明。
- **影响**：要么线上协议被破坏（子进程收到 base64，连接失败），要么秘密在内存中多存活一份不可清零的字符串；「不进日志/不落盘」的保证无法验证。
- **依据**：`privateRestoreInput` 的字段类型与 `json.Marshal` 行为；`execute` 的 `defer clear(input)`；`runChild` 的异步写管道；`design.md:4053` 未给自定义编码。
- **建议**：定义专用类型（实现 `MarshalJSON` 输出原始字符串，或直接构造 `json.RawMessage` 的安全片段），明确唯一所有者与清零同步点（建议 pipe 写完成后立即清零，再由 `Close` 兜底），并增加 wire golden test 与「base64 形状被拒」的负向测试。
- **阻断实施**：是。线协议与秘密生命周期均不唯一。

### B13J-S424A-ARCH-R01-F008 — HIGH — `!linux` exec 门禁与现有 Windows 子进程测试冲突

- **定位**：`design.md:4059`–`:4062`、`:4070`；`cmd/han-server/recovery_restore.go:451`–`:467`；`cmd/han-server/recovery_restore_test.go:187`。
- **问题**：r61.5 规定 `!linux` 的 `execRestoreTool` 一律返回 `ErrAdmissionUnsupported`，r61.8 又说 `recovery_restore.go` 的「宿主与 exec 委托到 build-tag 文件」。现有 `recovery_restore_test.go:187` 在 Windows 上构造 `restoreProcessTool`，ENV 仅 `SystemRoot=...`，并真实启动子进程；`imageConfiguration`（`:461`–`:466`）明确允许 `SystemRoot`／`WINDIR`。如果 `restoreProcessTool.Run` 改走 `execRestoreTool`，该测试在 Windows 必然失败；如果保留原路径，则生产 exec 门禁并未覆盖 `restoreProcessTool`，`design.md:4070` 的「不提供 Windows 例外」不成立。
- **影响**：现有测试被破坏，或生产路径与测试路径分叉；实施者需自行决定「软件合同测试」与「生产 exec」的分流方式。
- **依据**：`recovery_restore.go:451`–`:467` 的 Windows ENV 允许列表；`recovery_restore_test.go:187` 的真实子进程；`design.md:4062`、`:4070` 的 `!linux` 拒绝。
- **建议**：明确 `restoreProcessTool` 是测试专用替身还是生产实现；若为生产实现，给出测试 seam（注入 exec 函数）与 Windows 期望；若为测试专用，新增生产 `execRestoreTool` 并禁止其进入测试构造。同步更新 `recovery_restore_test.go` 的 build tag/平台断言。
- **阻断实施**：否（可通过分流解决），但当前描述会直接导致测试失败或门禁绕过。

### B13J-S424A-ARCH-R01-F009 — HIGH — 负向 (d)(e)(f) 标 NOT RUN 不能关闭 S4.24a 的关闭条件 (4)(5)

- **定位**：`design.md:4079`–`:4081`、`:4083`；`.add/project/knowledge/tasks/tasks.md:3196`–`:3209`。
- **问题**：r61.6 明确把 (d) shell／附加 argv／stdout receipt／exit=0、(e) 外部见证／签名、(f) `ld.so.preload`／`ld.so.cache`／RPATH／RUNPATH 的真实触发标为 NOT RUN（`:4079`–`:4081`），并要求测试用 `//go:build linux` 或 skip 标注。tasks 关闭条件 (4) 要求「负向测试证明无 shell、无附加 argv、无自证成功」，(5) 要求 9 条验收命令通过；9 条命令全部是 Windows 本地 `gofmt`／`go build`／`go vet`／`go test`（`:3198`–`:3206`），没有任何 Linux 构建、Linux 测试或 Linux 集成命令。被 skip 的测试不是「通过」，NOT RUN 也不满足 (4)(5)。
- **影响**：S4.24a 无法在现有任务合同下关闭；若以 NOT RUN 签收，会把 B6 的未验证范围伪装成已闭合。生产 restore 继续 closed 不能替代卡关闭条件的证据。
- **依据**：`design.md:4079`–`:4081` 的 NOT RUN 标注与 `:4083` 的 skip 要求；`tasks.md:3208`–`:3209` 的负向与关闭条件；`tasks.md:3198`–`:3206` 的命令集。
- **建议**：把 Linux host／fd-exec／loader 的真实执行列为独立关闭条件并给出环境、命令与证据落点；或拆出一张 Linux 验证卡，S4.24a 只保留可在 Windows 验证的解析/拒绝层，并删除「闭合 B6」表述。不得以 NOT RUN 关闭 (4)(5)。
- **阻断实施**：是（阻断关闭，不阻断设计文本定稿）。

### B13J-S424A-ARCH-R01-F010 — HIGH — Windows 上 owner/root/0700 不可达，与「真实可触发」冲突

- **定位**：`design.md:3929`（批准文件 owner/root/0700/只读挂载）、`design.md:4076`–`:4077`（(a)(b) 标「真实可触发」）；`tasks.md:3208` 的 (a)(b)。
- **问题**：批准文件要求父目录 `0700`、文件 `0600` 等价 owner-only、属主 root、位于登记过的受控只读挂载来源；`tool` 记录还要求 `owner_uid`／`owner_gid`／`mode`／`st_dev`／`st_ino`／`mount`。Windows 没有 root、POSIX uid/gid/mode 语义，也没有 `/proc/self/mountinfo`；r61.6 却把 (a) 相对路径／SHA 不符／manifest 缺失和 (b) backup ref／digest 缺失／角色自批标为 Windows「真实可触发」。若这些用例走完整文件边界，会先得到 `ErrAdmissionUnsupported` 或 `ErrAdmissionIncomplete`，而不是 (a)(b) 期望的分类；若绕过文件边界，则测试的不是生产路径。
- **影响**：负向覆盖是「空真」或错层断言，B6 的可达性声明不成立。
- **依据**：`design.md:3929` 的 POSIX 文件边界；`:4076`–`:4077` 的 Windows 标注；`tasks.md:3208` 的 oracle。
- **建议**：把 (a)(b) 拆成「平台无关的解析/语义层」与「POSIX host 层」两组；Windows 只跑前者并明确不执行 owner/mount 检查，后者标 Linux-only；或在设计里把 (a)(b) 改为 Linux 环境验证。
- **阻断实施**：否（需分层说明），但当前 B6 可达性结论不可接受。

### B13J-S424A-ARCH-R01-F011 — HIGH — 批准快照、时间与撤销在 `Prepare`→`Replace` 间不闭合

- **定位**：`design.md:4004`–`:4008`；`cmd/han-server/recovery_restore.go:222`–`:255`、`:375`。
- **问题**：r61 说 `Prepare` 用预检快照复核并在 `Journal.Begin` 前重读精确字节；`Replace`／`VerifyRestored`「经 `prepared()` 再次校验有效期与撤销状态」。但现有 `prepared()` 只做 `source.CheckApproved` 与 `checkRestoreArtifact`（`:247`–`:255`），driver 没有批准快照字段、没有注入时钟、没有批准文件路径；`prepared()` 如何取得这些输入未定义。若只使用预检快照，则 `Prepare` 之后、`Replace` 之前的撤销（原子替换文件）不可见；若在 `Replace` 重读文件，又与「Begin 之后不再接受新的批准或凭据来源」（`:3927`）以及「`now` 取预检注入的时钟，不在校验中途重取」（`:4008`）冲突。长时间运行的 `Replace` 在 `valid_until` 之后完成时，是否仍算有效也未定义。
- **影响**：已撤销或已过期的批准可能被用于实际替换；或者实现选择重读，导致时钟/字节语义与设计声明不一致。B1 的「未撤销」判据不可验证。
- **依据**：`prepared()` 的现有调用与字段；`design.md:4006`–`:4008` 的读取时机与时钟；`design.md:3927` 的 Begin 后封闭声明。
- **建议**：定义 driver 持有的 `approvalSnapshot`（含原始字节、解析结果、预检 `now`）与 `prepared()` 的扩展；明确 `Replace` 前是否重读、重读失败/漂移的分类、`valid_until` 与运行时钟的关系；补「Prepare 后撤销」「Replace 跨过 valid_until」两个负向测试。
- **阻断实施**：是（B1 核心判据）。

### B13J-S424A-ARCH-R01-F012 — MEDIUM — mountinfo/ACL/父目录/loader 算法未定义

- **定位**：`design.md:4064`。
- **问题**：只给检查项名称，没有给可编码算法：`/proc/self/mountinfo` 的字段转义（`\040` 等）、嵌套挂载/绑定挂载/overlay 的「当前挂载点」选择、同一 mount point 多行的处理；ACL「条目排序加十六进制小写」未给条目文本格式（tag、qualifier、permissions、mask、默认 ACL 是否纳入）；「artifact 父目录受保护」未定义属主/模式/sticky/no-follow 的具体判据；`ld.so.cache` 的「路径登记在工具记录」因 F003 缺字段而无法落地。
- **影响**：不同实现会接受不同宿主状态，隔离检查可能被绕过或误拒。
- **依据**：`design.md:4064` 的短语式描述；无伪代码、无测试向量。
- **建议**：给出 mountinfo 解析与匹配规则、ACL 规范文本与摘要算法、父目录检查清单、loader 字段 schema；为每项提供正/负 fixture。
- **阻断实施**：否（MEDIUM）。

### B13J-S424A-ARCH-R01-F013 — MEDIUM — DSN 结构校验未定义

- **定位**：`design.md:4045`–`:4051`。
- **问题**：只规定文件路径/权限、UTF-8、长度 1..4096、无 BOM/NUL/CR/LF/首尾空白，没有规定 DSN 语法：`postgres://` URL 还是 keyword=value；允许哪些参数；是否要求 `sslmode`、目标 dbname、角色名与 `installation_id`／`deployment_id` 绑定；重复参数、空 host、非法百分号编码如何处理。r61 说「缺失或不可核即 `ErrAdmissionIncomplete`，在 `Journal.Begin` 前 closed」，但「不可核」没有可执行定义。
- **影响**：语法错误的 DSN 可能通过预检，直到 `Replace` 子进程连接时才失败（此时已过 Begin），违反预检 closed 的承诺；两个实现接受范围不同。
- **依据**：`design.md:4047`–`:4049` 只列字节级约束。
- **建议**：指定解析器与错误映射（复用 `pgx` 的 `ParseConfig` 或自定义严格文法），在预检阶段完成解析并核对目标绑定；补非法 scheme/空 host/重复参数等负向测试。
- **阻断实施**：否（MEDIUM，但影响 B3 的早拒保证）。

### B13J-S424A-ARCH-R01-F014 — MEDIUM — `validateRestoreHost`/`execRestoreTool` 签名及与 `restoreProcessTool` 的关系未给

- **定位**：`design.md:4057`–`:4062`、`:4087`；`cmd/han-server/recovery_restore.go:451`–`:500`、`:649`–`:697`。
- **问题**：文件表只给文件名、build tag 与职责，没有给 `validateRestoreHost`／`execRestoreTool` 的 Go 签名、返回值（fd/lease/cleanup）、调用点，也没有说明现有 `restoreProcessTool.imageConfiguration`／`openImage`／`runChild` 是保留、包装还是替换。fd 所有权（谁关闭、取消时谁回收）未定义。
- **影响**：实施者可写出多种调用图，测试替身与生产路径可能分叉；平台门禁是否覆盖现有 `restoreProcessTool` 不可判定（并触发 F008）。
- **依据**：`design.md:4057`–`:4062` 只有职责描述；`recovery_restore.go` 现有方法签名未被引用。
- **建议**：给出精确签名、返回的 fd/lease 生命周期、调用点与错误映射；明确现有方法的去留和测试 seam。
- **阻断实施**：否（MEDIUM）。

### B13J-S424A-ARCH-R01-F015 — HIGH — 新配置键与唯一预检入口未接入现有配置/启动顺序

- **定位**：`design.md:4006`、`:4085`、`:4087`；`cmd/han-server/recovery_config.go:58`–`:83`、`:222`–`:241`；`cmd/han-server/main.go:400`、`:442`、`:629`。
- **问题**：r61 新增 `HAN_RECOVERY_RESTORE_APPROVAL_PATH` 与 `HAN_RECOVERY_RESTORE_TARGET_DSN_FILE`，但：`recoveryRootConfigurationKeys`（`:58`–`:83`）不含这两个键；`readAction`（`:222`–`:241`）只读 executable/manifest/SHA/control；`recoveryRootConfigurationPresent` 因而可能在只配置新键时不认为 recovery 存在。r61 要求 `runComposed` 在 `loadStartupConfig` 后、`parseDatabase` 前调用唯一预检，但现有 `planRecoveryStartup` 在 `main.go:629`、晚于 `:442` 的 `parseDatabase`；两者的职责边界、是否重复读取、谁拥有错误分类均未定义。`verify`／`genesis`／普通启动不读新文件，但 `install` 是否读也未说明（与 F019 相关）。
- **影响**：新键可能被忽略、recovery 模式可能不触发，或预检顺序无法满足「Begin 前 closed」；实施者必须自行改配置层和启动链，范围超出 r61.8 的描述。
- **依据**：上述键集与调用点；`design.md:4087` 只说 `recovery_config.go`「两个新键」，未说如何加入 key set/readAction。
- **建议**：把两个键加入 `recoveryRootConfigurationKeys`，定义 `readAction` 或独立 preflight 的读取与错误映射，明确 `planRecoveryStartup` 与新预检的合并/替换关系及调用顺序；补配置存在性/缺失/部分配置测试。
- **阻断实施**：是（B1/B3 的早拒保证无法落地）。

### B13J-S424A-ARCH-R01-F016 — MEDIUM — 文件/影响面清单不完整

- **定位**：`design.md:4055`–`:4062`、`:4074`–`:4081`、`:4087`；作者报告 §8；`tasks.md:3189`–`:3194`。
- **问题**：r61.8 的影响面只列 `recovery_restore.go`、`recovery_runtime.go`、`main.go`、`recovery_config.go` 及对应测试，新增只列 `internal/recovery/restore_source.go` 与 r61.5 的四个平台文件及测试。但 r61.6 表明确提到 `recovery_restore_approval.go`、`recovery_restore_approval_test.go`、`recovery_restore_platform_other_test.go`；作者报告 §8 还列出 `recovery_restore_target.go`、`recovery_restore_host_linux_test.go`、`recovery_restore_exec_linux_test.go`、`recovery_restore_production_integration_test.go`。r61.8 未给完整文件表，也未区分生产文件与测试文件。
- **影响**：实施者无法确定完整写集；审计无法核对是否有未声明文件；与 tasks 4 文件写集直接冲突（F017）。
- **依据**：上述三处清单的并集大于 r61.8；`design.md:4087` 未提 approval/target 解析文件。
- **建议**：补一张完整文件表（路径、build tag、生产/测试、职责），与 tasks 写集同步；删除「仅四个平台文件」的误导性表述。
- **阻断实施**：否（MEDIUM，但与 F017 合并后阻断）。

### B13J-S424A-ARCH-R01-F017 — HIGH — tasks 4 文件写集无法实现 r61 新范围

- **定位**：`tasks.md:3189`–`:3194`；`design.md:4087`；作者报告 §8。
- **问题**：任务卡只授权 4 个文件：`cmd/han-server/recovery_restore.go`、`recovery_runtime.go`、`main.go`、新增 `recovery_restore_production_integration_test.go`。r61 需要新增 `internal/recovery/restore_source.go`、批准文件解析与测试、目标凭据解析与测试、四个平台文件与测试，并修改 `recovery_config.go`（不在写集）。按现有任务合同，实施者要么越界写文件，要么无法实现 r61。
- **影响**：S4.24a 在任务层不可执行；任何「已实现」声明都会伴随未授权写集。
- **依据**：`tasks.md:3189`–`:3194` 的 4 文件清单与 `design.md:4087`／作者报告 §8 的并集。
- **建议**：由 Task-Planner 修订卡面写集与范围（本评审不改 `tasks.md`），或把新增文件拆成独立任务并明确依赖；在写集更新前不得启动实施。
- **阻断实施**：是（任务合同层）。

### B13J-S424A-ARCH-R01-F018 — BLOCKER — ADR-0031 不变量 6 的外部回执验签/见证未编码

- **定位**：`design.md:3955`–`:3979`（`tool.receipt` 字段）、`:4037`–`:4039`（verifier 只核来源四字段）、`:4079`–`:4081`（(e) NOT RUN）；`framework.md:1147`（不变量 6）；`cmd/han-server/recovery_restore.go:359`–`:362`。
- **问题**：ADR-0031 不变量 6 要求回执由工具/备份系统原生写入、经独立读取路径核对、读取时先验签并留审计记录、信任锚固定登记在批准记录内且轮换经人批准、验证方独立于执行身份与 root、含产生者身份与单调序列。r61 只在 `tool.receipt` 里放了 `kind`、`producer_identity`、`sequence_floor`、`independent_read_path`、`verifier_identity`、`trust_anchor_ref` 六个字符串/数字字段：没有签名或见证回执字段，没有信任锚材料/格式，没有验签算法，没有审计记录字段，没有序列单调性检查，也没有任何调用点；`RestoreSourceVerifier` 只核 `BackupRef`／`ArtifactDigest`／`ArtifactBytes`／`SourceSchemaDigest`，不核回执。现有 `execute` 仅比较 `target.ReadRestored` 与子进程 receipt 相等（`:359`–`:362`），不具备外部验签能力。r61.6(e) 又把真实签名/见证标为 NOT RUN。
- **影响**：不变量 6 未被编码，B6(e) 未闭合；实施者无法实现「读取时先验签」；若未来据此解除 closed，会缺少 ADR 要求的信任锚与独立性证据。r61 的「只补 B1–B6 可实施合同」声明不成立。
- **依据**：`framework.md:1147` 的合取要求；`design.md:3955`–`:3979` 的字段集；`design.md:4037`–`:4039` 的 verifier 范围；`:4079`–`:4081` 的 NOT RUN。
- **建议**：定义回执签名/外部见证的完整契约（字段、canonical bytes、信任锚登记与轮换、验签实现位置、审计字段、失败映射、序列检查），或明确把不变量 6 排除出本卡并停止声称 B6 闭合；与 ADR-0031 的 owner 对齐后再进入实施。
- **阻断实施**：是（不变量 6 无法实现）。

### B13J-S424A-ARCH-R01-F019 — HIGH — `inputs.restore == nil → recoveryClosedRestoreDriver` 对 `enter`/`install` 不成立

- **定位**：`design.md:3611`、`:4041`；`cmd/han-server/recovery_runtime.go:456`–`:492`、`:527`–`:532`、`:596`–`:599`。
- **问题**：r61 说「任一步失败或任一依赖为 nil，`inputs.restore` 保持 nil，`recovery_runtime.go` 现有 `nil → recoveryClosedRestoreDriver` 分支不变」。但 `newRecoveryRootAssembly` 对 `enter`／`install` 先调用 `recoveryEnterInstallMissingPorts`（`:527`–`:532`），该函数把 `inputs.restore == nil` 列入缺失端口（`:464`–`:466`），直接返回 `ErrAdmissionClosed`，不会到达 `:596`–`:599` 的 fallback；只有 `normal`／`verify`／`resume` 会走 fallback。并且 r61.2 只说 `enter`／`resume` 要求批准文件，`install` 不读批准/凭据却仍被要求提供 restore 端口，语义不一致。
- **影响**：`enter`／`install` 的失败分类与设计声明不同；测试若按「closed driver」断言会失败；`install` 是否需要 restore 能力不明确。
- **依据**：上述 `recoveryEnterInstallMissingPorts` 与 switch 分支；`design.md:4006` 只列 `enter`／`resume`。
- **建议**：按 action 逐一写清 nil 依赖的结果与错误分类；若 `install` 不需要 restore，则从 `recoveryEnterInstallMissingPorts` 排除或明确 unsupported；若需要，则把 `install` 纳入预检。同步修正 `:3611`／`:4041` 的「现有分支不变」表述。
- **阻断实施**：否（需澄清），但当前描述与代码不符，属 HIGH。

### B13J-S424A-ARCH-R01-F020 — HIGH — `ErrAdmission*` 一律退出码 2 与现有 `run` 错误映射不一致

- **定位**：`design.md:4085`、`:4070`；`cmd/han-server/main.go:171`–`:199`；`cmd/han-server/lifecycle.go:707`–`:713`。
- **问题**：r61.7 说四类 `ErrAdmission*`「一律退出码 2（与既有 invalid_config／closed 一致）」；r61.5 也说 `!linux` 的 `ErrAdmissionUnsupported` 退出码 2。但现有 `main` 只在 `selectRecoveryCommand` 解析失败时 `os.Exit(2)`（`:176`）；`run` 返回任何非 nil 错误都经 `lifecycleExitCode` 得到 1，然后 `os.Exit(1)`（`:181`–`:183`、`:196`–`:198`），`lifecycleExitCode` 对所有非 nil 恒返回 1（`lifecycle.go:707`–`:713`）。若预检在 `runComposed` 内返回 `ErrAdmission*`，实际退出码是 1，不是 2。
- **影响**：负向测试/运维脚本按 2 断言会失败；若为实现退出码 2 在 `main` 增加特殊分支，又超出 r61.8 的影响面说明。
- **依据**：上述 `main` 与 `lifecycleExitCode` 的实际映射；`design.md:4085` 的声明。
- **建议**：明确退出码映射发生在哪一层（例如 preflight 在 `main` 调用并在 `run` 之前映射，或引入可识别的错误类型由 `main` 转 2），并同步 `recovery_config`/lifecycle 的既有语义；补退出码集成测试。
- **阻断实施**：否（需明确），但当前声明与代码不一致，属 HIGH。
## 3. Verdict

**CHANGES REQUIRED**

### 3.1 结论

r61 的方向与既有边界基本一致：独立批准文件、独立 `RestoreSourceVerifier` 端口、独立目标凭据文件、Linux/非 Linux build-tag 文件、负向 (a)–(f) 分层，均没有把 `recoveryClosedRestoreDriver` 改成默认放行，也没有新增 `HAN_DATABASE_DSN` 回退或从 pool 反推 DSN。`cmd/han-server/recovery_runtime.go:596`–`:599` 与 `cmd/han-server/recovery_sources.go:557`–`:574` 的 closed 默认与回退逐字未变（Q3 通过）。

但 r61 仍不能作为 S4.24a 的唯一实施合同，原因不是「实现细节待补」，而是五处核心契约仍要求实施者自行发明：

- **F001**：批准记录 digest 的 canonical bytes 未定义，两个独立实现无法得到同一 `approval_digest`。
- **F005**：`RestoreSourceVerifier` 拿不到实际制品句柄，也没有调用时机与 plan→claim 映射，B2 的独立核验不可实现。
- **F006**：`newControlledRestoreDriver` 只加 verifier 形参，但生产 `authority`/`ownership` 无设置路径；driver 需要 Runtime、Runtime 又需要 driver，形成未解循环。
- **F007**：`dsn []byte` 经 `json.Marshal` 变 base64，直接违反「stdin 仍是字符串」；清零同步点也未定义。
- **F018**：ADR-0031 不变量 6 的外部回执验签/见证/信任锚/审计字段未编码，B6(e) 仍为 NOT RUN。

因此 verdict 是 **CHANGES REQUIRED**，不是 PASS WITH CONDITIONS：上述五项需要作者重新定义契约或调整范围，不能由实施者在卡内自行选择；其中 F009/F017 还分别阻断关闭条件与任务写集。

### 3.2 Finding 数量

| 严重级 | 数量 | ID |
|---|---|---|
| BLOCKER | 5 | F001、F005、F006、F007、F018 |
| HIGH | 11 | F002、F003、F004、F008、F009、F010、F011、F015、F017、F019、F020 |
| MEDIUM | 4 | F012、F013、F014、F016 |
| LOW | 0 | — |

### 3.3 六问简答

1. **B1／B2／B3／B5／B6 是否闭合**：B1 未闭合（F001/F002/F010/F011）；B2 未闭合（F005/F006/F019）；B3 未闭合（F007/F013）；B5 部分闭合但 loader 字段与 exec 分流未闭合（F003/F008/F014）；B6 部分闭合且卡不可关闭（F009/F018/F020）。结论：不能只依据 design 写出唯一实现。
2. **与已接受口径是否冲突**：ADR-0031 的 closed/default 未放宽，但不变量 6 未编码（F018）；ADR-0030／anchor `sources` 语义未被改变（Q4 成立）；ADR-0027 的 `Prepare` 早于 `Begin` 方向正确，但唯一预检与 `planRecoveryStartup` 未对齐（F015）；ADR-0032／r60 的「未登记依赖不得注入」与 r61.3 文字冲突（F004）；ADR-0019 未发现冲突。r61.3「不登记 11 端口、不称第 12 端口」在「driver 构造期内部依赖≠端口」的解释下可以自洽，但 r60 必须补这一限定，现状文本不自洽。
3. **第二 authority／fallback／隐式默认／closed 变 open**：未发现新增 fallback、隐式默认或 closed→open；`recoveryClosedRestoreDriver` 默认与回退逐字未变。第二 authority 目前只在命名/职责上不成立，`RestoreSourceVerifier` 的独立性因缺制品句柄（F005）无法证明。
4. **端口放置取舍**：新增 `internal/recovery/restore_source.go` 而不扩展 `recovery.InstallationSourceVerifier` 的理由成立；后者属于 anchor adapter bundle 的 `sources` 端口，首期 registry 为空即 closed，复用会让 restore 反向依赖 anchor registry 并混用两种信任输入。新接口的放置正确，但接口本身不闭合（F005）。
5. **装配顺序是否唯一不可绕过**：概念方向正确，但「唯一不可绕过」不成立：预检入口/顺序与现有 `main.go:629` 冲突（F015）；`authority`/`ownership` 无生产接线（F006）；`enter`/`install` 不会走 `nil → closed` fallback（F019）；DSN 线上形状会被 `[]byte` 改变（F007）。现有测试 `recovery_restore_test.go:201`、`:206`、`:229` 的后置赋值也证明「只补测试替身实参」不足。
6. **NOT RUN 是否构成 S4.24a 关闭缺口**：是。`tasks.md:3208`–`:3209` 的关闭条件 (4)(5) 要求负向证明与 9 条验收命令通过，r61.6 自标 (d)(e)(f) NOT RUN（`design.md:4079`–`:4081`），而 9 条命令全是 Windows 本地命令，没有 Linux 构建/测试。明确结论：**必须换 Linux 环境取证，或拆卡；不能以 NOT RUN 关闭 S4.24a**。设计层最多算「测试位置已定义」的部分闭合。

## 4. 未验证边界

以下内容本轮未取得证据，不得被读作已通过；它们只降低本报告结论的覆盖范围，不改变上述 finding 的成立：

- **未运行代码验证**：未执行 `go test`、`go build`、`go vet`、`go test -race`、`gofmt`、lint，也未执行 `GOOS=linux go build ./...`；未在 Windows 上实际运行退出码用例。代码结论均来自静态阅读。
- **工作树含并行改动**：`git status --porcelain` 显示 `design.md`、`framework.md`、`tasks.md` 等存在其他角色的未提交改动；本报告只固定并核对了给定哈希的 `design.md` r61，不评判其他改动，也未回退任何内容。
- **无 Linux 真实环境证据**：未验证 POSIX owner/ACL、`/proc/self/mountinfo`、`ld.so.preload`、`ld.so.cache`、RPATH/RUNPATH、fd-exec、进程组 join/reap、ENV allowlist 的真实行为；这些正是 r61.5/r61.6 的 NOT RUN 范围（F009/F012/F018）。
- **无真实信任输入**：未取得真实批准记录、真实 restore 工具、真实旧备份、原生耐久回执、签名/外部见证、信任锚或验签审计样本；F018 的结论只针对「设计与接口是否编码这些要求」，不评价真实产品能力。
- **未读安全侧报告正文**：未读取 `review-sec-r01` 的完整正文，未核对安全侧 findings；本报告只代表架构侧独立结论，不代替安全侧评审或人的接纳。
- **ADR 状态存在项目内不一致**：`framework.md:1129` 写「r07 尚未复评，决策接受级对现行修订尚未达成」，而 `tasks.md:3195` 写「r07 已于 2026-11-02 被两路独立复评接受」并给出报告哈希。两者未被 r61 修改，本报告只登记为未验证边界，不作为 r61 finding；但它直接影响「S4.24a 能否启动」，需由 Lead 核对以哪一份状态为准。
- **ADR 未全文重读**：ADR-0030／0027／0032／0019 只核对了与 11 端口、`sources`、`Prepare`/`Begin`、closed/CAS 相关的锚点，未逐字全文重读；若全文另有与 r61 冲突的条款，本报告未覆盖。
- **待实现文件不存在**：`cmd/han-server/recovery_restore_approval.go`、`recovery_restore_target.go`、`recovery_restore_host_linux.go` 等尚不存在，未做编译或行为验证；本报告只判断设计文本是否给出可实施契约。
- **对象稳定性**：本报告只对 `design.md` SHA256 `EF55DC9272F24C7626F028F705B4322B4E33DCFA6D0197545015CA7DE4FE5811` 负责；若对象在复审前变化，本 verdict 不自动适用，须按新哈希重做差异核对。

## 5. 复审条件

1. 作者提交新的 design revision（建议 r62），固定新哈希，并逐项回应 F001–F020；不得只写「后续修复」。
2. F001/F002/F003 需给 canonical JSON 规则与 golden vector、统一 digest 表示、补齐 tool loader/`tool.approval_digest`/`artifact_bytes` 映射。
3. F005/F006/F011 需给 verifier 的制品句柄/构造形态、调用点、claim 映射、`authority`/`ownership` 的生产接线与批准快照/时钟/撤销语义。
4. F007 需给自定义 JSON 编码与清零同步点，并附 wire golden test。
5. F018 需给回执签名/外部见证/信任锚/审计契约，或明确把不变量 6 排除出本卡并停止声称 B6 闭合。
6. F009 需给出 Linux 取证环境与关闭条件，或拆卡；F017 需由 Task-Planner 更新写集后才能启动实施。
7. 复审将对照本报告原 finding 与本轮修复修订，并重新核对 `tasks.md` 写集、`cmd/**` 消费者与 ADR-0031 状态；复审不替代人的接纳或生产批准。