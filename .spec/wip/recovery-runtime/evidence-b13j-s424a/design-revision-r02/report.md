# identity-auth r61 → r62 设计修订作者证据报告

## 0. 元信息与边界

- **设计版本**：`identity-auth-design-ir01-r62`
- **修订人角色**：Feature-Designer（ADD）
- **修订对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- **工作区基线**：分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **本轮性质**：仅设计正文修订与作者证据报告；不实施、不评审、不改代码、不改任务卡、不改 ADR。

**证据基线（两份 r01 独立评审，均为本轮修订前的 r61 评审）**：

| 评审 | 路径 | 字节 | SHA256 | verdict | 严重级计数 |
|---|---|---:|---|---|---|
| 架构侧 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r01/report.md` | 56625 | `78489013177ED00D460B3AA3A6E5EFD810CF2292CDE4C30B2E508D8F8C16C7DA` | **CHANGES REQUIRED** | BLOCKER 5 / HIGH 11 / MEDIUM 4 |
| 安全侧 | `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r01/report.md` | 40500 | `C5D5BE68CCEAFC349722D6677C40F6A5BB8C46CB58A8D6F4FC7F01D16ED3A637` | **CHANGES REQUIRED / BLOCKED** | BLOCKER 5 / HIGH 2 / LOW 1 |

本报告只证明「作者按 finding 做了哪些设计修订、落在哪些章节、哪些仍未闭合」。它不构成评审通过、不构成生产 restore 可用、不构成 `enter`／`resume` 解锁、不构成 ADR-0031 约束 8 满足，也不构成 `identity-auth-S4.24a` 可关闭。

## 1. 修改前后对象快照

| 项目 | 修改前（r61） | 修改后（r62） |
|---|---:|---:|
| 字节数 | 876660 | 936613 |
| 行数（`[IO.File]::ReadAllLines().Count`） | 5109 | 5317 |
| SHA256 | `EF55DC9272F24C7626F028F705B4322B4E33DCFA6D0197545015CA7DE4FE5811` | `EA880660B8D43AFD6FB227812C662BB5A4581B39F4235F3DAB8E221F5F9FDC9A` |
| `revision` | `identity-auth-design-ir01-r61` | `identity-auth-design-ir01-r62` |
| `last_updated` | `2026-11-02` | `2026-11-02` |

- 增量：`+59953` 字节、`+208` 行。
- 修订后作者自检补正两处（非评审提出）：其一，`r62.6.2` 原句「三次都发生在 `Journal.Begin` 之后」与其括注自相矛盾，改为明确 `Prepare` 早于 `Journal.Begin`、`Replace`／`VerifyRestored` 在 Begin 之后；其二，`r62.9.4`／§4／§5 的 r07 状态冲突出处按现行行号精确到 `tasks.md:3197`，并保留评审原文所引 `:3195`。
- 修改方式：r62 增量插在 r61 正文之前，保留 r61 历史正文；被 r62 取代的 r61 结论在原文内以 `（r62.x …）` 就地标注，不删除历史。
- `tasks.md` 基线（只读核对，未修改）：898536 字节 / SHA256 `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`；`tasks.md:3186` 仍为 `- [ ]`。
- 两份 r01 评审文件字节与 SHA256 与 §0 一致，未修改。
- 本轮复算的 JCS golden vectors：`source` canonical `1115` 字节 / `3c7b1cff97b230077c9d3d75085082d26bfe5e48b8355cb721e0a830f51059e9`；`tool` canonical `2357` 字节 / `9df40264e869c7ed9506b3ac5788f5878cf516c980082ffa64bedb1ffb205d68`。二者只证明文本向量可复算，不证明独立实现或生产工具可用。

## 2. 五个两路重合 BLOCKER 的落点

| # | BLOCKER | 落点章节 | 行号 | 唯一可编码答案摘要 |
|---|---|---|---:|---|
| 1 | 批准摘要自证（ARCH-F001／SEC-F001） | r62.1 | 3929 | 真实性移到独立信任根文件：`HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH` + `_SHA256` 固定 pin；信任根与批准文件不同挂载来源；Ed25519 签发／复核签名；RFC 8785 JCS 唯一定义（删 `approval_digest`／`signatures`、UTF-16 键序、控制字符转义、整数十进制字符串、数组排序、禁 `null`）；`signing_digest` 为 canonical bytes 的 SHA-256；`approval_digest` 为 `sha256:` + 小写 hex；信任根缺失／不可读／pin 不符 → `ErrAdmissionIncomplete`，格式／签名／principal／review 不符 → `ErrAdmissionBinding`，撤销／过期 → `ErrAdmissionClosed`，平台无法验签 → `ErrAdmissionUnsupported`；无「未签名也接受」分支。 |
| 2 | 复核人独立性不可机核（SEC-F002） | r62.2 | 3960 | `issuer`／`reviewer`／`executor` 各带稳定 `principal_id` 与 `appointment_ref`；三者必须登记在信任根 `principals`；`review_ref` 解析到唯一 `reviews` 记录，逐项匹配 reviewer／issuer／executor、artifact digest、tool sha256、backup ref、`decision=approved`、`reviewed_at`；principal／appointment 两两不同，`kind` 分别 approver／reviewer／executor，affiliations 与 supplier 交叉禁止；预检与每次 `revalidateApproval` 都重算；失败按 binding／closed／incomplete／unsupported 分类；时间关系 `issued_at <= reviewed_at <= now`，有效期 ≤366 天。 |
| 3 | 撤销 TOCTOU 未闭合（SEC-F003／ARCH-F011） | r62.3 | 3968 | 唯一解析与首次校验入口是 `recoveryRestoreStartupPreflight`；唯一重校验方法是 `revalidateApproval(ctx, now)`；读取点固定为预检、`Prepare` 返回 plan 前、`Replace` 启动子进程前、`VerifyRestored` 启动子进程前，以及子进程返回后解码回执前；每点现取一次 `clock.Now()` 并向内传递，跨点必须重取，时钟回拨 → `ErrAdmissionClosed`；字节／inode／`st_dev`／挂载来源或挂载点漂移、删除、截断、非 regular → `ErrAdmissionClosed`；撤销不晚于下一检查点生效，启动后撤销 → `errors.Join(ErrRecoveryWriteUnknown, err)`，调用方只能按 `restore_unknown`；撤销后新的 `Prepare`／`Replace`／`VerifyRestored` 一律 `ErrAdmissionClosed`。 |
| 4 | DSN `[]byte` 线上形状矛盾（ARCH-F007／SEC-F004） | r62.4 | 3980 | 作废 `[]byte` 字段方案；wire 字段仍为 `string`，`target_connection.dsn` 始终是 JSON 字符串；唯一编码路径 `privateRestoreInput` + `appendJSONString`，禁止对 `[]byte` 直接 `json.Marshal`、禁止 base64、禁止 `json.RawMessage` 与第二 DSN 字段；五处清零时点：`execute` defer、`runChild` 写后、`Close`、预检快照 `Clear()`、子进程解析后；残余风险明确登记为 GC／管道内核缓冲／子进程地址空间／core dump／调试器，只承诺本进程可见副本，工具侧 `RLIMIT_CORE=0`、`PR_SET_DUMPABLE=0`、`MADV_DONTDUMP` 属 unknown。 |
| 5 | 外部回执验签／见证未编码（ARCH-F018） | r62.5 | 3994 | `tool.receipt` exact keyset 固定为 13 字段；`signature_alg=ed25519`，签名载荷为回执正文 JCS + `{producer_identity, producer_key_id, trust_anchor_ref, sequence, issued_at}` 的 JCS，删除 `signature`／`audit_record_ref`；公钥只来自信任根 `receipt_anchors`；`verifier_principal_id` 必须 `kind=verifier` 且不等于 executor／issuer；`independent_read_path` 与 `audit_sink` 绝对规范、no-follow、独立挂载；`sequence > sequence_floor`；`issued_at` 在 `[now-5m, now]`；`VerifyRestored` 固定八步顺序（重校验 → source verify → execute → 独立读取 → 验签 → 序列 → 追加并 fsync 审计 → 回填 `audit_record_ref`）；缺字段／验签失败／anchor 不符／序列不单调 → binding 或 `restore_unknown`，缺失／不可读 → incomplete，撤销／过期 → closed，平台不支持 → unsupported。首期无真实 anchor／见证端点／审计 sink，生产 `VerifyRestored` 继续 closed。 |

架构侧另外两个 BLOCKER 也已闭合：ARCH-F005 落 `r62.6.2`（4010，`RestoreSourceClaim` 携带已核 artifact `*os.File`、三次调用时机、plan→claim 映射与错误分类）；ARCH-F006 落 `r62.6.3`（4028，两阶段构造 + `Bind(runtimeValue, anchors.Exclusive)`，`authority` 仍是同一个 `*recovery.Runtime`，`ownership` 仍是同一 `anchors.Exclusive`，closed port 只接线不解锁，无第二 authority）。

## 3. Finding → 章节映射（两路全覆盖）

### 3.1 架构侧（BLOCKER 5 / HIGH 11 / MEDIUM 4）

| Finding | 严重级 | 落点 | 本轮处置 |
|---|---|---|---|
| ARCH-F001 | BLOCKER | r62.1 | 本轮闭合契约：外部信任根、JCS、Ed25519、anchor 缺失／不可核 closed；真实信任根留实施 |
| ARCH-F002 | HIGH | r62.1／r62.6.11 | 本轮闭合：`approval_digest` 统一 `sha256:` + 64 位小写 hex，裸 hex／大写 hex 拒绝 |
| ARCH-F003 | HIGH | r62.1／r62.6.11／r62.7 | 本轮闭合：`tool` 增 `executor`／`elf`／`network`／`security`；`tool.approval_digest` 消费者明确；`artifact_bytes` 三方映射明确；loader schema 见 r62.7 |
| ARCH-F004 | HIGH | r62.6.1 | 本轮文本限定：构造期内部依赖不是端口，不注入 r60 production root；不推翻 r60 |
| ARCH-F005 | BLOCKER | r62.6.2 | 本轮闭合：claim 携带 artifact fd、三次调用时机、plan→claim 映射、错误分类 |
| ARCH-F006 | BLOCKER | r62.6.3 | 本轮闭合：两阶段装配与 `Bind`；同一 Runtime、同一 `anchors.Exclusive`；无第二 authority |
| ARCH-F007 | BLOCKER | r62.4 | 本轮闭合：wire `string` + `appendJSONString` + 五处清零；残余风险登记 |
| ARCH-F008 | HIGH | r62.6.4 | 本轮分流：既有 Windows 测试继续用 `restoreProcessTool`；新门禁只作用于生产 `restoreExecTool` |
| ARCH-F009 | HIGH | r62.6.5／r62.9.3 | 本轮登记关闭前置：本设计通过评审也不关闭 S4.24a；关闭需 Linux 目标／CI 真实证据 |
| ARCH-F010 | HIGH | r62.6.6 | 本轮分层：解析／语义层 Windows 可真实触发；POSIX host 层 Linux-only，Windows 只断言 unsupported |
| ARCH-F011 | HIGH | r62.3 | 本轮闭合：重读点、时钟点、撤销生效时限、in-flight 语义 |
| ARCH-F012 | MEDIUM | r62.7.4／r62.8 | 本轮闭合算法：mountinfo 选择、ACL 规范、父目录逐级 lstat；正负 fixture 留实施 |
| ARCH-F013 | MEDIUM | r62.4／r62.8 | 本轮闭合文法与错误映射；负向测试留实施 |
| ARCH-F014 | MEDIUM | r62.7.3／r62.8 | 本轮闭合端口签名、调用点与 fd 生命周期；artifact fd 由 driver 关闭 |
| ARCH-F015 | HIGH | r62.6.7 | 本轮闭合：四个新键、`actionRequiresRestore`、`readAction`、`runComposed` 唯一预检顺序、`planRecoveryStartup` 只消费快照 |
| ARCH-F016 | MEDIUM | r62.9.1 | 本轮闭合完整文件表；写集更新留 Task-Planner |
| ARCH-F017 | HIGH | r62.6.8／r62.9.2 | 本轮只给影响与建议写集；`tasks.md` 未改，留 Task-Planner |
| ARCH-F018 | BLOCKER | r62.5 | 本轮闭合契约：字段、签名算法、信任根、独立验证方、审计、缺失 closed；真实 anchor／见证／sink 留实施，生产继续 closed |
| ARCH-F019 | HIGH | r62.6.9 | 本轮按 action 固定 nil 语义：install 合法 nil；enter 装配期 closed；resume closed driver；normal／verify 不调用 |
| ARCH-F020 | HIGH | r62.6.10 | 本轮闭合：预检错误包装 `*recoveryRestorePreflightError` → exit 2；run 期错误 → exit 1；不作废现有 `run` 映射 |

### 3.2 安全侧（BLOCKER 5 / HIGH 2 / LOW 1）

| Finding | 严重级 | 落点 | 本轮处置 |
|---|---|---|---|
| SEC-F001 | BLOCKER | r62.1 | 本轮闭合契约：外部信任锚、pin、签名、JCS、anchor 缺失 closed；残余风险「root／部署方可同时替换信任根与 pin」明确登记 |
| SEC-F002 | BLOCKER | r62.2 | 本轮闭合：principal／appointment／review／角色交叉／校验时机／失败分类 |
| SEC-F003 | BLOCKER | r62.3 | 本轮闭合：`revalidateApproval`、每点重取时钟、撤销时限、in-flight 与 `restore_unknown` |
| SEC-F004 | HIGH | r62.4 | 本轮闭合 wire 与清零；GC／管道／子进程副本风险明确登记，不声称物理清零 |
| SEC-F005 | BLOCKER | r62.7／r62.7.1–r62.7.4 | 本轮闭合 schema 与算法：`PT_INTERP`、`DT_NEEDED`、`DT_AUDIT`／`DT_DEPAUDIT`、库内容／挂载、`AT_SECURE`／capabilities、网络隔离、CWD／fd 继承、`ld.so.cache` 路径／digest、ACL、父目录；Linux 实测 NOT RUN |
| SEC-F006 | HIGH | r62.7.2／r62.7.3 | 本轮闭合执行身份／权限链：root-owned `0755`、专用非 root uid/gid、空 `Groups`、`NoSetGroups=false`、`Credential` exec 前生效、禁 setuid／setgid／file capabilities、禁 `execveat(AT_EMPTY_PATH)`、`/proc` 不可用 unsupported 且无 fallback；Linux 实测 NOT RUN |
| SEC-F007 | BLOCKER | r62.6.5／r62.9.3 | 本轮诚实保留 NOT RUN；不把 skip／无匹配 exit 0 写成 PASS；关闭需 Linux 目标／CI |
| SEC-F008 | LOW | r62.1／r62.4／r62.7／r62.8 | 本轮闭合精度：JCS、ACL 编码、时间关系、mount option 排序去重、整数十进制字符串、`mode=0755`、receipt 字段 |

## 4. MEDIUM／LOW 处置登记

| Finding | 处置 | 落点 | 说明 |
|---|---|---|---|
| ARCH-F012 | 本轮闭合算法；fixture 留实施 | r62.7.4 | mountinfo 字段解析、最长前缀 + `mount_id` 决胜、ACL 规范文本、父目录逐级 lstat；未在 Linux 实测 |
| ARCH-F013 | 本轮闭合文法；负向测试留实施 | r62.4 | 只接受 `postgres://`／`postgresql://`，显式端口，三个固定参数，禁止 hostaddr／service／passfile／options／search_path |
| ARCH-F014 | 本轮闭合签名与生命周期 | r62.7.3 | `restoreHostLease` 只拥有 tool fd；artifact fd 由 `restoreSource.OpenApproved` 返回并由 driver 关闭；`Close` 只关 tool fd |
| ARCH-F016 | 本轮闭合文件表 | r62.9.1 | 11 个源文件 + 测试文件；写集更新不改 `tasks.md` |
| SEC-F008 | 本轮闭合精度 | r62.1／r62.4／r62.7／r62.8 | 不改变 finding 的 LOW 级；只消除示例与措辞歧义 |
| 真实信任根／密钥／批准记录／见证／审计 sink | 明确留待实施；生产继续 closed | r62.1／r62.5／r62.9.4 | 首期不提供生产批准主体与生产工具，契约形状可编码但无真实锚 |
| Linux host／loader／fd-exec／ACL／mount／network 实测 | 明确留待实施与独立验证（NOT RUN） | r62.6.5／r62.9.3 | Windows 结构性不可达；不得用「未 panic」或空断言替代 |
| ADR-0031 r07 状态冲突 | 需决策（范围外） | r62.9.4 | `framework.md:1129` 与 `tasks.md:3197`（评审原引 `:3195`）的状态陈述不一致；ADR 与任务卡均不在本轮可改范围 |

## 5. 未闭合项与理由

1. **ADR-0031 r07 状态冲突（需决策）**：`framework.md:1129` 与 `tasks.md:3197`（评审原引 `:3195`）对 r07 复评状态的描述不一致；需由架构决策与任务规划分别对账。本轮不改 ADR 正文与任务卡。
2. **真实信任根、签发／复核／执行／验证方密钥、两条批准记录、回执见证与审计 sink 均不存在**：r62 只固定契约形状；生产 restore 继续 closed，`recoveryClosedRestoreDriver` 仍是默认。
3. **首期不提供生产 restore 工具与生产批准主体**：结论不变；无真实工具、备份、见证与审计，生产 `VerifyRestored` 与 `enter`／`resume` 继续 closed，ADR-0031 约束 8 未满足。
4. **Linux 实测 NOT RUN**：host／ACL／mount／loader／fd-exec／netns／seccomp／`AT_SECURE` 均未在 Linux 上运行，未取得任何负向 (d)(e)(f) 证据；Windows 上的 skip／无匹配 exit 0 不是 PASS。
5. **`install` 排除与 `Bind` 接线需要改 `recovery_runtime.go`**：属实施任务，不在本轮；r62.9.1 已登记建议写集。
6. **写集不足**：`tasks.md` 当前 4 文件写集不足以实现 r62 新范围；本轮只给影响与建议，未改 `tasks.md`，由 Task-Planner 后续更新卡面。
7. **golden vectors 只由作者复算**：`source`／`tool` 两向量已在本地复算一致，但尚未由独立实现复现；验收必须由独立实现复算，不得把作者自算当 PASS。
8. **既有 Windows 子进程测试与 `restoreProcessTool` 本轮未运行**：本轮没有代码变更，也未执行测试；既有测试保持可跑是设计分流结论，不是本轮实测结论。

## 6. 写集影响（仅报告，不改 `tasks.md`）

`tasks.md` 当前 S4.24a 写集为 4 文件：`cmd/han-server/recovery_restore.go`、`cmd/han-server/recovery_runtime.go`、`cmd/han-server/main.go`、`cmd/han-server/recovery_restore_production_integration_test.go`。

按 r62 契约，建议写集至少扩展为：

- `cmd/han-server/recovery_restore.go`（修改）
- `cmd/han-server/recovery_restore_approval.go`（新增）
- `cmd/han-server/recovery_restore_target.go`（新增）
- `cmd/han-server/recovery_restore_host_linux.go`（新增，`linux`）
- `cmd/han-server/recovery_restore_host_other.go`（新增，`!linux`）
- `cmd/han-server/recovery_restore_exec_linux.go`（新增，`linux`）
- `cmd/han-server/recovery_restore_exec_other.go`（新增，`!linux`）
- `cmd/han-server/recovery_runtime.go`（修改）
- `cmd/han-server/main.go`（修改）
- `cmd/han-server/recovery_config.go`（修改）
- `internal/recovery/restore_source.go`（新增）
- 对应测试：`recovery_restore_test.go`（修改）、`recovery_restore_approval_test.go`、`recovery_restore_target_test.go`、`recovery_restore_host_linux_test.go`、`recovery_restore_host_other_test.go`、`recovery_restore_exec_linux_test.go`、`recovery_restore_exec_other_test.go`、`recovery_config_test.go`（修改）、`recovery_restore_production_integration_test.go`、`internal/recovery/restore_source_test.go`。

影响：按现 4 文件写集实施只能越界或无法实现；建议 Task-Planner 以 r62.9.1 为基线重划写集，或在写集更新前保持不可实施。本轮未改 `tasks.md` 任何一行，未勾选任何 checkbox。

## 7. NOT RUN 边界声明

- 本机是 Windows；负向 (a)(b) 的解析／语义层可真实触发，POSIX host 层（owner／root／`0700`、ACL、mountinfo、loader、fd-exec、netns／seccomp）为 Linux-only，Windows 只断言 `ErrAdmissionUnsupported` 且零状态变更。
- (c) 可真实触发；(d)(e)(f) 在 Windows 结构性不可达，标 **NOT RUN**。
- `tasks.md` 的 9 条验收命令是 Windows 本地 `gofmt`／`go build`／`go vet`／`go test`；`//go:build linux` 导致的 skip、无匹配测试的 exit 0 都不等于 PASS。
- Linux 目标或 CI 上的真实 host／ACL／mount／loader／fd-exec／network 实测未运行，属 **NOT RUN**。
- 本设计通过评审也不构成 `identity-auth-S4.24a` 关闭；关闭条件 (4)(5) 需要 Linux 目标／CI 上真实负向 (d)(e)(f) 与 9 条验收命令的通过证据，本轮不提供。
- 本轮未执行任何代码构建、测试、lint 或运行时验证；所有结论均为设计文本与只读核对结论。

## 8. 最终声明

**未写代码、未改 `tasks.md`／ADR、未勾选任务、未声称 PASS。**

本报告不构成评审通过、生产 restore 可用、`enter`／`resume` 解锁、ADR-0031 约束 8 满足或 `identity-auth-S4.24a` 可关闭。生产仍以 `recoveryClosedRestoreDriver` 为默认；任何真实信任根、批准记录、工具、备份、见证、审计 sink 与 Linux 实测仍为 unknown／NOT RUN。