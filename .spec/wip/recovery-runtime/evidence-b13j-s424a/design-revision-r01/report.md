# B13J / `identity-auth-S4.24a` 设计修订 r01 报告

## 0. 元信息与边界

- 角色：`add-feature-designer`（Agent 产出，非人的签字）
- 主方法：`.codex/skills/add-design-tdd/SKILL.md`；规则：`design-doc-writing-style`、`abstraction-principles`、`decisions-guide`
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 日期：2026-11-02
- 授权依据：Lead 按用户 standing 授权取建议值，D1/D2/D3 为「是」，D4 不采用（不把 S4.24a 降级为恒拒绝门）
- 本报告只记录设计侧结论草案与写集建议；SSOT 是 `design.md` 的 `ir01-r61` 段。本报告不修改 `tasks.md`、ADR、Plan、test-plan、stories 与既有证据报告
- 设计状态仍为 `draft`；本报告不声称独立评审、人的接纳、隔离实测、原生回执或生产可用

## 1. 输入与基线

### 1.1 design.md 改前基线（实测）

| 指标 | 值 |
|---|---|
| 字节数 | 854991 |
| 行数 | 4944 |
| SHA256 | `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` |
| 总 LF | 4944 |
| CRLF | 169 |
| LF-only | 4775 |
| TAB 行 | 1 |
| FF | 0 |
| BOM | 无 |

### 1.2 证据报告哈希漂移登记（unknown，未修改）

派发说明给出的证据报告指纹与实测不符；本报告只登记，不声明一致，也不改动该文件：

| 来源 | 字节数 | SHA256 |
|---|---|---|
| 派发说明给定 | 10030 | `4CC0F4678373E0AEFB540EF54F97025BE70349970368731A998E0CB4F404B7C7` |
| 2026-11-02 实测 | 10670 | `815CC215DB615B77E176FBEA8CA45079235A0082854F500A50682B47C3443763` |

结论：证据报告内容已在本轮之前发生漂移，漂移原因与漂移前后正文差异为 **unknown**；本修订依据的是实测现行正文中 §2 B1–B6、§3 与 §5 D1–D4 的事实，若 Lead 需要以给定哈希为唯一基准，须先裁定以哪一份为准。本项不阻断设计闭合。

## 2. B1 批准记录：结论草案

**载体**：独立批准文件（不是配置文件扩展、不是 manifest 扩展、不是环境变量内联）。理由：ADR-0031 要求备份来源与可执行文件是「两条独立记录」且须有独立读取路径；内联进既有 manifest 会让被核验对象与核验依据同一份文件，等于用文件自带 hash 自证，与 `design.md:3878`「不能仅信文件内自带 hash」冲突。

**配置键**：`HAN_RECOVERY_RESTORE_APPROVAL_PATH`，值为绝对规范路径。

**文件约束**：字节级要求为绝对路径、父目录与文件 owner-only（`0700`／`0600` 等价）、no-follow 打开后的 regular file、受控只读挂载来源、不大于 65536 字节、UTF-8 无 BOM、无 NUL、只允许 LF。任一不满足即 closed。

**格式**：strict JSON，`version` 固定为 `han-restore-approval/v1`；顶层恰为 `version`、`source`、`tool` 三键，`source` 与 `tool` 是两条独立记录对象，不得互相代替、不得共用。

**B1 与 ADR-0031 r05 四项结论第 1 项最小字段的逐项映射**：

| ADR r05 第 1 项最小字段 | 载体字段 |
|---|---|
| 记录类型 | `record_type`（`restore_source` / `restore_tool`，与所在对象强一致） |
| 角色名与具名人员 | `issuer.role_name`、`issuer.person_name` |
| 任命依据引用 | `issuer.appointment_ref` |
| 签发时间（UTC） | `issued_at`（UTC 六位微秒） |
| 适用 installation 与 backup ref | `installation_id`、`backup_ref` |
| artifact digest（SHA256） | `artifact_digest`（仅 `source`） |
| source installation id | `source_installation_id`（仅 `source`） |
| 可执行文件绝对路径与 SHA256 | `executable_path`、`executable_sha256`（仅 `tool`） |
| 该文件 `st_dev`/`st_ino` | `st_dev`、`st_ino`（仅 `tool`） |
| owner/ACL | `owner_uid`、`owner_gid`、`mode`、`acl{kind,digest}`（仅 `tool`） |
| 受控只读挂载来源 | `mount{source,mount_point,options}`（仅 `tool`） |
| 复核人与复核时间 | `reviewer.person_name`、`reviewer.role_name`、`reviewer.reviewed_at`（+ `review_ref`） |
| 有效期与撤销状态 | `valid_from`、`valid_until`、`revocation{state,revoked_at,revocation_ref}` |

其余字段不是新增决策要素，只服务可编码性：`deployment_id`、`target_digest`（绑定本次目标）、`environment`（封闭 ENV 字面集）、`receipt{...}`（原生耐久回执与信任锚登记）、`approval_ref`、`approval_digest`（自引用摘要，排除自身后按固定字段顺序紧凑 UTF-8 重算 SHA256）。

**命中 `RestorePlan`**：`RestorePlan.SourceApprovalRef`/`SourceApprovalDigest` 必须与 `source` 记录的 `approval_ref`/`approval_digest` 逐字节相等；`SourceInstallationID`、`ArtifactDigest`、`BackupRef`、`InstallationID`、`DeploymentID`、`TargetDigest` 亦须逐项相等。任一不等即 `ErrAdmissionBinding`。

**读取时机**：`runComposed` 在 `loadStartupConfig` 之后、`parseDatabase` 之前调用唯一预检 `recoveryRestoreStartupPreflight(getenv, recoveryCommand, now)`；仅 `enter`／`resume` 要求批准文件存在。`Prepare` 再用同一已校验快照复核，并在 `Journal.Begin` 前重读精确字节比对；`Replace`／`VerifyRestored` 经 `prepared()` 复核。预检后字节漂移即 `ErrAdmissionBinding`。

**有效期与撤销**：`issued_at <= valid_from <= now < valid_until`，三者必填，不允许「永久有效」；`revocation.state` 必须为 `active` 且 `revoked_at`、`revocation_ref` 为空串（不用 `null`）。过期、未到生效、已撤销即 `ErrAdmissionClosed`。

**失败分类**：缺键／文件缺失／owner／mode／非 regular／超长／非 UTF-8／JSON 不合法 → `ErrAdmissionIncomplete`；字段缺失、摘要不符、自批（签发=复核或签发=执行者）、复核人非具名独立 → `ErrAdmissionBinding`；过期或撤销 → `ErrAdmissionClosed`；平台不支持 → `ErrAdmissionUnsupported`。全部在 Begin 前，无 fallback。

## 3. B2 恢复来源核验端口：结论草案

**结论**：新增独立端口，不扩展 `recovery.InstallationSourceVerifier`。理由：后者属 ADR-0030 anchor adapter registry 的安装来源核验语义，扩它会把 restore 批准记录耦合进 anchor bundle 解析，并与 `recoveryClosedPort` 的 closed 语义冲突；独立端口可用新增文件隔离，不动 `internal/recovery` 既有公开契约。

**契约**（新增 `internal/recovery/restore_source.go`）：

```go
package recovery

type RestoreSourceClaim struct {
    Version              string
    RunID                string
    BackupRef            string
    DeploymentID         string
    InstallationID       string
    TargetDigest         string
    ArtifactDigest       string
    ArtifactBytes        int64
    SourceApprovalRef    string
    SourceApprovalDigest string
    SourceInstallationID string
    SourceSchemaDigest   string
}

type RestoreSourceVerifier interface {
    VerifyRestoreSource(ctx context.Context, claim RestoreSourceClaim) error
}
```

**语义**：实现必须独立核验 `BackupRef`、`ArtifactDigest`、`ArtifactBytes`、`SourceSchemaDigest` 与已批准记录及实际制品一致，不得用 `RestorePlan` 自身字段互证；缺失、不匹配、无法核验分别返回 `ErrAdmissionIncomplete`／`ErrAdmissionBinding`／`ErrAdmissionUnsupported`。

**装配**：作为 `newControlledRestoreDriver` 的构造期内部依赖（不作为第 12 个 root 端口、不加入 r60 的 11 端口表、不加入 `RuntimeDependencies`）。唯一 production root 装配点仍在 `cmd/han-server/main.go:629`；预检通过后才构造 verifier/source/target/tool/driver，任一缺失即 `inputs.restore = nil`，`recovery_runtime.go:596` 继续落到 `recoveryClosedRestoreDriver`。`nil` verifier 传入 driver 构造即 `ErrAdmissionUnsupported`，不得默认放行。

**影响面**：`recovery.InstallationSourceVerifier` 与 `recoveryClosedRestoreDriver` 定义不变；`recovery_restore_test.go` 既有直接构造点需补 verifier 实参（测试替身）。无生产语义变更。

## 4. B3 restore 目标连接凭据：结论草案

**配置键**：`HAN_RECOVERY_RESTORE_TARGET_DSN_FILE`，值为绝对规范路径，指向单凭据文件。**不采用** `HAN_DATABASE_DSN` 作为回退：ADR-0031 不变量 3 要求执行身份使用独立数据库角色、与业务库角色分离，回退会直接把业务 DSN 交给 root 启动的替换子进程。

**文件约束**：绝对路径、owner-only、no-follow regular、受控只读挂载；内容为单个 UTF-8 DSN，无 BOM、无 NUL、无 CR、无 LF、无尾随空白，长度 1 至 4096。

**读取时机**：在创建目标 pool／发起连接**之前**读取并校验；缺失、不可读、权限不符、格式不符一律 `ErrAdmissionIncomplete` 并 closed，不做环境变量回退、不从既有 pgxpool 反推、不「先连上再补批准」。

**传递**：只经私有 stdin，仍为 `{"plan":...,"target_connection":{"dsn":string}}`；不进 argv、不进环境变量、不进日志／Trace／stdout、不落盘、不复用 root 的 pool 配置。建议实现把 `dsn` 由 `string` 改为写后可清零的 `[]byte`，在写入管道后与 driver `Close` 时清零；该改动只在 cmd 内部，不改 `internal/recovery` 公开契约。

## 5. B5 POSIX/Windows 平台文件边界：结论草案

**文件划分**：

| 文件 | build tag | 内容 |
|---|---|---|
| `cmd/han-server/recovery_restore_host_linux.go` | `//go:build linux` | host 侧文件边界检查 |
| `cmd/han-server/recovery_restore_host_other.go` | `//go:build !linux` | 返回 `ErrAdmissionUnsupported` |
| `cmd/han-server/recovery_restore_exec_linux.go` | `//go:build linux` | 固定 fd-exec、进程组、ENV |
| `cmd/han-server/recovery_restore_exec_other.go` | `//go:build !linux` | 返回 `ErrAdmissionUnsupported`，零状态变更 |

**host 检查项（Linux）**：`st_dev`/`st_ino` 与批准记录相等；owner/ACL（`system.posix_acl_access` 规范摘要）与记录相等；挂载来源经 `/proc/self/mountinfo` 核为受控只读且含 `ro`、`nodev`、`nosuid`、不含 `noexec`；`/etc/ld.so.preload` 不存在或为空；动态 ELF 的 `ld.so.cache` 摘要与批准记录一致，静态 ELF 要求该摘要为空；ELF 的 `DT_RPATH`/`DT_RUNPATH` 非空即拒；打开方式固定 `O_RDONLY|O_CLOEXEC|O_NOFOLLOW`。

**exec 机制（Linux）**：对同一已核 no-follow fd 直接执行——经 `exec.Cmd.ExtraFiles` 把该 fd 放到子进程 fd 3，`Path` 取 `/proc/self/fd/3`，`Args[0]` 为批准的绝对路径；或 `execveat(fd, "", ..., AT_EMPTY_PATH)`。禁止验后按路径重开。独立进程组／会话，取消或超时终止整组并真实 `Wait`/reap。`env_clear` 后只设批准记录内封闭字面集（建议 `LANG=C.UTF-8`、`LC_ALL=C.UTF-8`、`TZ=UTC`），禁 `PATH`、`LD_PRELOAD`、`LD_LIBRARY_PATH`、`LD_AUDIT`、`GLIBC_TUNABLES`、`*_PRELOAD`、`*_OPTIONS`，禁任何 DSN/secret；子进程实际 environ 与白名单不逐字节相等即 closed。

**Windows 显式门禁**：`!linux` 两个文件一律返回 `ErrAdmissionUnsupported`，在 Begin 前、任何 DB/CAS/文件状态变更前拒绝，退出码 2；不建子进程、不静默放行、不把「无效」当「通过」。

## 6. B6 负向矩阵可达性

预检顺序必须为：配置路径校验 → 批准/DSN 解析 → 路径与 SHA 核验 → 平台 host gate。平台门不得前置，否则卡面负向 (a)(b) 在 Windows 不可达。

| 卡面负向 | 层／文件 | 测试类型 | 本机 Windows |
|---|---|---|---|
| (a) 相对路径、SHA 不符、manifest 缺失/不可读 | `recovery_restore_approval_test.go`、`recovery_config_test.go`、`recovery_restore_test.go` | 单元 | 真实可触发 |
| (b) backup ref/digest 缺失、角色自批 | `recovery_restore_approval_test.go` | 单元 | 真实可触发 |
| (c) 平台不支持 | `recovery_restore_platform_other_test.go` | 单元/integration | 真实可触发；断言 `ErrAdmissionUnsupported` 且零 DB/CAS/文件写 |
| (d) shell／附加 argv／stdout receipt／exit=0 | 构造与解析单测 + `recovery_restore_exec_linux_test.go` | 单元 + Linux integration | 构造/解析可跑；真实 fd-exec 与子进程 **NOT RUN** |
| (e) 外部见证／批准人独立性 | `recovery_restore_approval_test.go` + Linux integration | 单元 + Linux | 拒绝分支可跑；真实签名/见证 **NOT RUN** |
| (f) `ld.so.preload`／cache／RPATH／RUNPATH | `recovery_restore_host_linux_test.go` | Linux integration | **NOT RUN**／平台不支持；不得写空真断言 |

判定 oracle：任一负向若进入 `Begin`/`Replace`、写入 DB/CAS、或返回成功即判定失败。

## 7. design.md 落点锚点

| 项 | 锚点 | 内容 |
|---|---|---|
| 头部修订号 | `design.md:5` | `revision: identity-auth-design-ir01-r61` |
| 头部日期 | `design.md:7` | `last_updated: 2026-11-02` |
| 头部设计版本 | `design.md:38` | `**设计版本**：identity-auth-design-ir01-r61` |
| 头部 r61 修订摘要 | `design.md:83`（行尾追加） | 输入 r60 SHA256、D1/D2/D3 依据、五项范围、不变项 |
| B1／B2 落地改写 | `design.md:3878` | manifest 批准摘要改由 r61 独立批准文件经 `RestoreSourceVerifier` 核验 |
| B3 落地改写 | `design.md:3882` | target DSN 改由 `HAN_RECOVERY_RESTORE_TARGET_DSN_FILE`，禁回退 `HAN_DATABASE_DSN` |
| 11 端口表 restore 行 | `design.md:3611` | 生产 driver 仅预检全通过后构造，否则 nil → closed |
| ENV 表新增行 | `design.md:3915` | `HAN_RECOVERY_RESTORE_APPROVAL_PATH,HAN_RECOVERY_RESTORE_TARGET_DSN_FILE` |
| r61 段起始 | `design.md:3925` | 目的与边界总述 |
| B1 段 | `design.md:3929` | 载体、文件字节要求、JSON 格式、字段映射表 `:3988`、读取时机、有效期/撤销、失败分类 |
| B2 段 | `design.md:4012` | `RestoreSourceVerifier` Go 契约（`:4014` 起）、错误语义、装配顺序、影响面 |
| B3 段 | `design.md:4045` | 配置键、文件要求、禁止回退、读取时机、私有 stdin 传递与清零 |
| B5 段 | `design.md:4055` | 文件划分表 `:4057`、Linux host 检查、fd-exec、ENV、Windows 门禁 |
| B6 段 | `design.md:4072` | 负向矩阵表 `:4074`、前置顺序、oracle、NOT RUN 标注 |
| 预检与退出码 | `design.md:4085` | 唯一预检入口与四类错误/退出码 2 |
| 影响面与不变量 | `design.md:4087` | 受影响/新增/不变/仍 closed/unknown |

插入位置说明：按 Lead 指定锚点置于 r60 段（`:3923`）之后、既有 `唯一 exact router registry` 段落（`:4089`）之前；采用与 r58–r60 一致的粗体段落体例而未使用 `#####` 标题，以免把后续既有段落误归入 r61 小节。r61.1、r61.7、r61.8 分别位于 `:3927`、`:4085`、`:4087`。

## 8. `identity-auth-S4.24a` 写集建议（供 `add-task-planner` 改卡面，本报告不改 tasks.md）

改：

- `cmd/han-server/recovery_restore.go`
- `cmd/han-server/recovery_runtime.go`（仅装配需要）
- `cmd/han-server/main.go`
- `cmd/han-server/recovery_config.go`
- `cmd/han-server/recovery_config_test.go`
- `cmd/han-server/recovery_restore_test.go`

新增：

- `internal/recovery/restore_source.go`
- `cmd/han-server/recovery_restore_approval.go`
- `cmd/han-server/recovery_restore_approval_test.go`
- `cmd/han-server/recovery_restore_target.go`
- `cmd/han-server/recovery_restore_target_test.go`
- `cmd/han-server/recovery_restore_host_linux.go`
- `cmd/han-server/recovery_restore_host_other.go`
- `cmd/han-server/recovery_restore_host_linux_test.go`
- `cmd/han-server/recovery_restore_exec_linux.go`
- `cmd/han-server/recovery_restore_exec_other.go`
- `cmd/han-server/recovery_restore_exec_linux_test.go`
- `cmd/han-server/recovery_restore_platform_other_test.go`
- `cmd/han-server/recovery_restore_production_integration_test.go`

不得修改：`recovery_sources.go` 的 `recoveryClosedRestoreDriver`、`internal/recovery/runtime.go` 既有公开契约、ADR、`tasks.md`、Plan、test-plan、stories、既有证据报告。

## 9. unknown 登记

- 真实「恢复批准角色」的任命、具名人员、两条批准记录原件：不存在，属部署方组织侧取证
- 真实 restore 工具二进制、其 SHA256、owner/ACL、受控只读挂载、`st_dev`/`st_ino` 实际值
- 真实旧备份与其 source installation／schema digest，以及 `VerifyRestored` 对真实备份的一致性
- POSIX ACL 摘要、mountinfo 形态、`ld.so.cache` 摘要、`/etc/ld.so.preload` 现状
- `fexecve`／`execveat` 在本机与目标主机的可用性与实测
- 原生耐久回执产品字段、独立读取路径、签名/外部见证机制、验签信任锚形态
- 执行身份切换机制与现实主体（ADR 明确「本 ADR 不选具体实现」，设计同样只固定判据）
- 证据报告哈希漂移（§1.2）
- Windows 上 (d)(e)(f) 的真实触发：结构性不可达

## 10. 未验证边界

- 未写任何 `.go` 文件，未编译、未跑 `go test`、未跑 lint
- 未做任何隔离实测、未取得任何原生回执、未做任何真实旧备份演练
- 未取得设计独立评审结论，未取得人的接纳；设计仍为 `draft`
- 未修改 ADR-0031，未修改 `tasks.md`／Plan／test-plan／stories

## 11. 停止条件核对

| 停止条件 | 结论 |
|---|---|
| 需改 ADR-0031 不变量/约束/解除条件/结论 | 未触发；r61 只补实现机制，未放宽任何不变量 |
| 需新增第二 authority／fallback／让 `recoveryClosedRestoreDriver` 不再是默认 | 未触发；缺失即 nil → closed |
| 需改 `internal/**` 既有公开契约且无法用新增端口隔离 | 未触发；新增 `restore_source.go` 隔离 |
| 契约只能靠「实现者自行决定」落地 | 未触发；五项均有唯一可编码形态 |
| 需超出写集 | 未触发 |

## 12. 改后实测与段落锚点

| 指标 | 改前 | 改后 |
|---|---|---|
| 字节数 | 854991 | 876660 |
| 行数 | 4944 | 5109 |
| SHA256 | `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` | `EF55DC9272F24C7626F028F705B4322B4E33DCFA6D0197545015CA7DE4FE5811` |
| 总 LF | 4944 | 5109 |
| CRLF | 169 | 169（未变） |
| LF-only 行 | 4775 | 4940（+165） |
| TAB | 1 | 1（未变） |
| FF | 0 | 0 |
| BOM | 无 | 无 |
| 孤立 CR | 0 | 0 |

净增 21669 字节：小范围精确替换 1705 字节（头部 3 处、restore 端口行、`:3878`、`:3882`、ENV 表新增 1 行）＋ r61 段 19964 字节。插入后抽检结构：`design.md:3923` r60 段、`:3925`–`:4087` r61 段、`:4089` 起既有段落逐字节保留；未改动区域的 169 处 CRLF 原样保留，新增内容全部为 LF。

本报告自身（`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r01/report.md`）为纯 LF、无 BOM 的新建文件。