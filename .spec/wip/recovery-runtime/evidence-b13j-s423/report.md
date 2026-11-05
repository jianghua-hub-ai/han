# identity-auth-S4.23 实施阻断报告（BLOCKED：未改代码、未交付、未关闭）

> 角色：`add-backend-dev`（Backend-Dev；本会话由主会话承载，无独立 Agent ID）。工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（只读核对，未变）。
> 日期：2026-11-03。结论：**BLOCKED**。命中任务卡停止条件第 2 条。本报告不是关闭证据，不替代独立评审、人的业务签收、生产批准或 Git 写授权。

## 0. 一句话结论

任务卡（`.add/project/knowledge/tasks/tasks.md:3152`–`:3182`）要求「实现并注册 anchor/CAS adapter」，且 adapter ID 必须是**已登记**值、witness 段必须**与批准记录比对**、`(w, r, token)` 状态机必须在 adapter 启动与每次推进前**真实核对**。核实结果：已登记 adapter ID 不存在、批准记录无任何已填值、anchor 见证的传输协议与存储 API 未定义；现行 design r72 与 ADR-0030 的冻结结论仍是「首期不提供生产 adapter、registry 恒空、四端口恒 closed」（`design.md:3612`–`:3616`、`:3910`；`framework.md:926`）。这些属决策级/公共契约级缺项，不是本卡 4 文件写集能补齐的实现细节。故按 `tasks.md:3182`「若 design/ADR 无法给出 adapter ID、配置路径、四端口合同或错误分类，停止并回报缺项」停止：未改代码、未创建新测试、未放宽任何断言、未虚构 adapter ID / 批准记录 / 见证端点。

依据原文（`tasks.md:3182`）：「若 ADR-0030 决策修订未被架构与安全独立评审接受，停止……；若 design/ADR 无法给出 adapter ID、配置路径、四端口合同或错误分类，停止并回报缺项；若需新增配置键、改 `internal/identity/**`/`internal/article/**`、建第二 registry/authority 或引入 fallback，停止并回设计。」

## 1. 基线与本轮真实执行的命令

工具链：`go version go1.27.0 windows/amd64`。`HAN_TEST_POSTGRES_DSN` 仅探测存在性（输出 `DSN=set`），**未打印其值**。基线在 HEAD 为绿，故下述阻断与既有失败无关。

| # | 命令（workdir 仓库根） | 退出码 | 真实输出摘要 |
|---|---|---|---|
| 0a | `git rev-parse HEAD` / `git branch --show-current` | 0 | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` / `main` |
| 0b | `git status --porcelain -- cmd internal migrations` | 0 | `?? cmd/` `?? internal/` `?? migrations/`（三目录在 HEAD 未跟踪） |
| 1 | `gofmt -l <4 文件>` | **2** | `GetFileAttributesEx cmd/han-server/recovery_anchor_production_integration_test.go: The system cannot find the file specified.` |
| 2 | `go build ./...` | 0 | 空 |
| 3 | `go vet ./cmd/han-server` | 0 | 空 |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | 空 |
| 5 | `go test ./cmd/han-server -run '^(TestRecoveryAnchorAdapterRegistry\|TestRecoveryRuntimeComposition)$' -count=1 -json` | 0 | `TestRecoveryAnchorAdapterRegistry` 5 子测试全 pass；`TestRecoveryRuntimeComposition` 8 子测试全 pass；包级 pass |
| 6 | 同 5 加 `-race` | 0 | 同上，包级 pass（1.22s 级） |
| 7 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryAnchorAdapterProductionBundle$' -count=1 -json` | 0 | **无任何 `"Test"` 事件**，仅包级 pass → 「无测试可跑」 |
| 8 | 同 7 加 `-race` | 0 | **无任何 `"Test"` 事件**，仅包级 pass → 「无测试可跑」 |
| 9 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryConfig$' -count=1 -json` | 0 | `TestRecoveryConfig PASS` |
| R | `go test -count=1 ./...`（回归） | 0 | 全绿：`cmd/han-server 24.343s`、`internal/article 2.647s`、`internal/identity 2.784s`、`internal/recovery 3.026s` 等；`internal/audit` 无测试文件 |

**第 7/8 条退出码 0 不构成通过**：本轮未创建 `recovery_anchor_production_integration_test.go`，`go test -run` 无匹配测试时返回空真结果。按卡面「不得以 exit 0 代替行为验证」，这两条登记为 **NOT RUN（空真）**，不计入验收。
## 2. 缺项（逐项带锚点与后果）

**B1 无已登记的生产 adapter ID 字面值。**
- 卡面要求「按已登记 adapter ID … 解析」（`tasks.md:3154`）、「adapter ID 非精确登记 → 四端口一起 closed」（`:3176(b)`）、关闭条件 (2)(3)（`:3177`）。
- 全仓检索：`newRecoveryAnchorRegistry(` 的调用只有 `recovery_runtime.go:551`（传 `nil`）与 `recovery_sources_test.go:260`/`:270`/`:282`/`:291`/`:302`（传测试夹具）；`resolve(` 的调用只有 `recovery_runtime.go:555` 与测试 `:264`/`:274`/`:286`/`:287`/`:295`/`:296`。
- 唯一出现的 ID 字面值 `product-cas` 仅存在于测试夹具（`recovery_sources_test.go:264`、`:270`、`:274`、`:275`、`:282`、`:291`、`:295`、`:302`），不是登记值；测试文件在本卡禁改清单内。
- `design.md:3910`：「首期不提供生产 adapter（ADR-0030），当前无已登记产品/凭据/独立故障域」；`design.md:3614` 同义。`framework.md:926`（ADR-0030 决策）：「首期不提供生产 CAS JournalHeadAnchor provider」。
- 后果：自选一个 ID（例如按 (a) 的 PostgreSQL 事件表命名）即新增对外配置契约并推翻「首期不提供」，属决策级修订，超出本卡授权。

**B2 卡面目标与已冻结设计结论正面冲突。**
- 卡面：「实现 anchor/CAS adapter 的 `Commitments`/`Sources`/`Anchor`/`Exclusive` 四端口，并在同一 bundle 内绑定」「`Exclusive` … 只作为『已登记但 closed』的端口接入，不得填实现」（`tasks.md:3154`）。
- 设计 11 端口清单中 `commitments`/`sources`/`anchor` 三行均写「首期 registry 为空，解析为 `recoveryClosedPort`；生产 closed」（`design.md:3612`–`:3614`），`exclusive` 行写「`newRecoveryAnchorRegistry(nil)` 使 registry 恒空，`exclusive` 只能是 `recoveryClosedPort`」（`:3616`）。
- 实现一个真实四端口 adapter 必然使 `Anchor`/`Commitments`/`Sources` 不再 closed（否则 adapter 无行为），与上述冻结结论冲突；保持全 closed 则「实现 adapter」无实际产物。二者不能同时成立，须先做决策级修订（`framework.md:1158` 约束 7、`:1165` 约束 9）。
- 后果：若擅自实现，等于在未经评审的 ADR 结论下落地生产 CAS provider，违反「设计与实现冲突即停止」。
**B3 witness 段无 schema，批准记录无机器可读载体。**
- ADR-0030 只给语义字段：「见证端点只作为同一配置文件内的 witness 段（地址、来源地址范围、TLS CA 与主机名）」「见证端点必须钉进批准记录（地址、来源范围、CA/主机名摘要），启动时与配置值比对，不一致即 closed」（`framework.md:952`）；「批准记录最小字段：…见证端点（地址、来源地址范围、TLS CA 与主机名）与启动时比对方式…」（`framework.md:1013`）。**未给键名、类型、版本或文件格式**。
- 唯一候选批准记录是空白表单 `approval-record-form.md`（96 行，A–H 组）：§C「允许来源地址范围」「TLS CA 与主机名」「启动时比对方式」三行「填写值」列全空（`:39`–`:41`），§E 五项保证等级全空；`preflight-record.md` §4 全部「待填」（`:51`–`:68`）。
- 卡面要求 (f3)/(f4) 断言「运行期见证端点与批准记录登记的端点不一致」「witness 段 CA/主机名与批准记录不一致」（`tasks.md:3176`）→ 没有可读的批准记录，任何比较都退化为「恒不一致 → 恒 closed」，不是可区分的行为验证，命中卡面 oracle「任一 (f) 用例仅以错误非 nil/exit 非 0 判定」。
- 后果：(f3)/(f4) 无法真实实现；自定义 witness 段键名与批准记录格式属新增公共契约（须决策级修订）。

**B4 anchor 见证的传输协议与 `(w,r,token)` 存储 API 未定义，且 owner 已豁免域外见证。**
- 不变量本身已定义（`framework.md:1009`：(i)–(vi)、幂等完成、不得用观测 head 抬高见证），但**没有**：见证端点的协议版本与请求/响应格式、`(w,r,token)` 的读-条件写 API 形状、鉴权与双 fd 注入的具体 wire、事件表的读取路径与字段映射。
- 全仓 `han-restore-witness/v1` 在 `framework.md` 命中 0 次、在 `design.md` 命中 1 次且属 restore 段（restore head 的 5 键签名协议）；它不是 anchor 见证的 `(w,r,token)` 协议，不能移用。
- owner 登记：「(v) 域外见证：继续不构造域外持久、不设跨故障域单调计数器端点」→ 维持 waived（`preflight-record.md:115`）；§8 (v)「明确不满足且不实施；不是证据不足，属本项目范围外」（`:101`）。
- 后果：(e)(iv)「有预留 `r` 且 `r` 上已有终结行 → 只允许幂等完成（把 `w` 推进到 `r` 并清预留）」没有可执行载体；无协议前提下实现只能对接进程内实现或测试替身，而卡面禁止 fake 作为生产来源，且关闭条件 (8) 要求「实现…并由负向 (e) 覆盖并通过」（`tasks.md:3177`）。
- **证据链断**：`preflight-record.md:31`/`:34`/`:35`/`:43` 引用的 `anchor-env-r01/README.md` 在仓库中不存在（`preflight-s4-23` 目录递归只有 `approval-record-form.md`、`preflight-record.md`、`upstream-questionnaire.md`；`rg --files -g README.md` 无 anchor-env 命中）。环境实测记录无法核对。
**B5 commitment/source 存储载体缺失，`AnchorSnapshot` 字符串令牌与事件表 `bigint` 版本无映射定案。**
- 接口只有方法签名：`InstallationCommitmentStore.LoadInstallation/CompareAndSwapInstallation`（`internal/recovery/runtime.go:184`–`:187`）、`InstallationSourceVerifier.VerifyInstallationSource`（`:190`–`:192`）、`AnchorReader.Load(ctx, namespace) (AnchorSnapshot, string, error)`（`:200`–`:202`）。
- `internal/recovery/` 内没有任何基于 PostgreSQL 的独立安装 CAS 存储实现；`migrations/` 中也不存在 `InstallationCommitment` 对应的表（`migrations/0001`–`0004` 未见 installation commitment 表）。卡面禁止新增 migration、禁止 fake/file/SQL fallback 之外的来源，因此生产 adapter 的 commitment/source 端口**没有权威存储载体**。
- `AnchorSnapshot`（`internal/recovery/journal.go:68`–`:74`）携带 `JournalID/Namespace/Epoch/Committed JournalHead/Pending`，`Load` 返回的第二个值是 `string` 版本令牌；而事件/日志表版本是 `bigint`（`migrations/0001_shared.sql:2`、`0003_article.sql:301` 等）。二者如何映射（十进制字符串、比较规则、溢出边界）在设计/ADR 中无定案。
- 后果：四端口 bundle 中 `Commitments` 与 `Sources` 无法按"真实消费者/权威存储"实现，只能落 closed 或依赖测试替身；卡面禁止 fake 作生产来源，关闭条件 (7)(8) 无法满足。

**B6 Windows 平台不可达项（POSIX 前提在 Windows 上不能真实达成）。**
- `go doc os/exec.Cmd.ExtraFiles` 实测原文：`ExtraFiles is not supported on Windows.` → 卡中「fd 3/4 双通道」无法在 Windows 生产路径建立。
- `go doc os.Chmod` 实测原文：`On Windows, only the 0o200 bit (owner writable) of mode is used` → owner-only 权限判定无法在 Windows 上真实成立。
- `rg -n "O_NOFOLLOW" -g "*.go"` 命中 9 处，均为 Unix 条件分支；Windows 无等价 no-follow 打开原语。
- `Get-Acl go.mod` 实测工作区 ACL 含 `NT AUTHORITY\Authenticated Users Modify`、`BUILTIN\Users ReadAndExecute`、`BUILTIN\Administrators FullControl`、`DESKTOP-FGO5CAD\CodexSandboxUsers Modify` → 「仅 owner 可读写」不成立。
- 按平台现实条款：这些分支一律 fail-closed（落 closed 并记录原因），但 (f6)/(f7)/(f8) 的"真实达成 + 正向断言"在本机不能判 PASS，只能登记"平台不可达 + 实测错误/输出"。

**B7（非阻断，需 Lead 裁定）`recoveryRootInputs` 四端口覆盖 seam 与既有集成测试耦合。**
- `recovery_runtime.go:556`–`:567` 允许 `inputs.commitments/sources/anchor/exclusive` 覆盖 registry 解析结果；`missingRecoveryRootInputs`（`:476`–`:491`）在 nil 时拒绝。
- 生产构造点仅 `main.go:634` 一处（`rg "recoveryRootInputs{" cmd/han-server` 命中 `main.go:634` 与两处集成测试），且**未传**这四个字段 → 生产语义仍由 registry 提供，满足"生产不得离开 closed"。
- 但既有集成测试依赖该 seam 注入端口：`recovery_enter_install_integration_test.go:225/276/364`、`recovery_production_chain_integration_test.go:2991`。这些测试不在本卡"不得修改"清单内、亦不在 4 文件写集内，本卡不改。需 Lead 确认：是否保留该 seam 作为测试注入点（生产不传），还是后续卡收紧。
## 3. 九条验收命令逐条状态

> 说明：以下为基线上真实执行的输出。因实施前已命中停止条件未改代码，第 1/7/8 条无本卡产出，其余为基线结果，不能代表本卡实现验收。

| # | 命令 | exit | 状态 | 真实输出摘要 |
|---|---|---|---|---|
| 1 | `gofmt -l <4 文件>` | 2 | **NOT PASS** | `GetFileAttributesEx ...recovery_anchor_production_integration_test.go: The system cannot find the file specified.`（测试文件未创建，gofmt 无法列空） |
| 2 | `go build ./...` | 0 | 基线 PASS | 空输出 |
| 3 | `go vet ./cmd/han-server` | 0 | 基线 PASS | 空输出 |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | 基线 PASS | 空输出 |
| 5 | `go test ./cmd/han-server -run '^(TestRecoveryAnchorAdapterRegistry\|TestRecoveryRuntimeComposition)$' -count=1 -json` | 0 | 基线 PASS | Anchor 5 子测试（empty_registry_is_legal_and_closes / registered_adapter_resolves_exactly / unregistered_id_closes / unavailable_configuration_closes / incomplete_registration_is_rejected）+ Runtime 8 子测试全 pass |
| 6 | 同 5 加 `-race` | 0 | 基线 PASS | 同上全 pass，无 race 报告 |
| 7 | `go test -tags=integration ... -run '^TestRecoveryAnchorAdapterProductionBundle$' -count=1 -json` | 0 | **NOT RUN（空真）** | JSON 事件无任何 `"Test"` 项，仅包级 pass → 测试不存在，exit 0 是空真 |
| 8 | 同 7 加 `-race` | 0 | **NOT RUN（空真）** | 同上 |
| 9 | `go test -tags=integration ... -run '^TestRecoveryConfig$' -count=1 -json` | 0 | 基线 PASS | `TestRecoveryConfig PASS` |
| R | `go test -count=1 ./...` | 0 | 基线全绿 | `cmd/han-server 24.343s`、`internal/article 2.647s`、`internal/identity 2.784s`、`internal/recovery 3.026s`；`internal/audit` 无测试文件 |

结论：无一条可代表本卡实现的验收；第 1 条明确失败，第 7/8 条不得记为 PASS。

复跑核对（2026-11-03，同一 HEAD）：9 条命令退出码与上表一致（1=2，2–9=0），第 7/8 条仍为 0 个 "Test" 事件；`go test -count=1 ./...` 复跑 exit 0（`cmd/han-server 23.818s`、`internal/recovery 2.994s`）。

## 4. 负向用例矩阵（按卡面 (a)–(f) 原编号）

全部 **NOT RUN**：无本卡实现，测试文件未创建。逐条依赖如下。

| 编号 | 卡面要求 | 状态 | 依赖缺失 / 现有覆盖 |
|---|---|---|---|
| (a) | 四端口缺任一 → `ErrAdmissionBinding`，四端口一起 closed | NOT RUN | `newRecoveryAnchorRegistry` 拒绝不完整注册（`recovery_sources.go:455`–`:466`），但无生产 bundle 与子测试 |
| (b) | adapter ID 非精确登记、路径相对/不存在/不可读 → 四端口一起 closed | NOT RUN | 现有 `unregistered_id_closes`、`unavailable_configuration_closes` 仅覆盖 registry 层；生产路径解析与"不可读"未实现 |
| (c) | registry 外无第二来源（file/SQL/fake/env）可打开 `Exclusive` | NOT RUN | 现有默认满足；`recovery_runtime.go:565`–`:567` 的 `inputs.exclusive` seam 见 B7，无本卡测试 |
| (d) | 未配置两个 env 键时生产 root 仍 closed | NOT RUN | 现有默认满足；无本卡测试 |
| (e) | 见证不变量：不可读/不可写、`revision > w` 有终结行、`revision <= w` 有未终结行且无预留 → 启动与每次推进前 closed；有预留 `r` 且 `r` 上有终结行 → 仅幂等完成（`w` 推进到 `r`、清预留），不得重写终结行、不得用观测 head 抬高 | NOT RUN | **B4 阻断**：无 witness 协议与 `(w,r,token)` 存储载体（ADR-0030 `framework.md:1009` 有语义、无 API） |
| (f1) | 见证端点凭据缺失/过期/主体错误 → closed 且拒绝推进 | NOT RUN | B3/B4（无端点与凭据合同） |
| (f2) | TLS 校验失败（自签 CA/主机名不匹配/证书过期）→ closed，不降级明文、不跳校验、不接受 `sslmode=require\|prefer\|allow\|disable` | NOT RUN | B3/B4 |
| (f3) `WitnessEndpointMismatch` | 运行期端点与批准记录不一致 → closed | NOT RUN | B3（无机器可读批准记录） |
| (f4) `WitnessConfigMissingOrMismatch` | witness 段缺失，或 CA/主机名与批准记录不一致 → closed | NOT RUN | B3（witness 段无 schema） |
| (f5) `WitnessSecretLeak` | 口令出现在 environ/argv/配置文件明文/日志 → closed | NOT RUN | B3（无 schema 即无法判定秘密字段） |
| (f6) `WitnessChannelReuse` | 双通道互异（两个一次性 fd、不同保管主体、不同秘密库路径），复用即 closed | NOT RUN | B6（Windows 无 `ExtraFiles`） |
| (f7) `WitnessCredentialChainReuse` | 复用业务 PostgreSQL 的 CA/DSN/用户名/凭据链 → `ErrAdmissionBinding` 且 closed | NOT RUN | B3/B4 |
| (f8) `WitnessConfigPathInsecure` | 配置路径 owner-only + no-follow；软链、组/其他可读、相对路径、父目录不满足 → closed | NOT RUN | B6（Windows ACL 不满足、无 O_NOFOLLOW） |

另：卡面要求 (f6)–(f8) 与 ACL/GRANT 矩阵以 `has_table_privilege` 等查询断言 `UPDATE/DELETE/DDL/TRUNCATE` 被拒、无跨库与超级用户路径；本机无 anchor 实例与可核环境记录（B4），**NOT RUN**。

## 5. 平台不可达项（实测）

| 项 | 实测命令/输出 | 结论 |
|---|---|---|
| `exec.Cmd.ExtraFiles` | `go doc os/exec.Cmd.ExtraFiles` → `ExtraFiles is not supported on Windows.` | fd 3/4 双通道无法在 Windows 建立；实现按 fail-closed |
| owner-only 权限 | `go doc os.Chmod` → `On Windows, only the 0o200 bit (owner writable) of mode is used` | 无法用 chmod 达成 owner-only |
| no-follow | `rg -n "O_NOFOLLOW" -g "*.go"` → 9 处，均为 Unix 条件分支 | Windows 无等价原语 |
| ACL | `Get-Acl go.mod` → 含 `Authenticated Users Modify`、`Users ReadAndExecute`、`Administrators FullControl`、`CodexSandboxUsers Modify` | owner-only 不成立 |
| `has_table_privilege` ACL 矩阵 | 无 anchor 实例；`preflight-record.md` 引用的 `anchor-env-r01/README.md` 不存在 | NOT RUN，无可核证据 |
## 6. 文件 SHA256（本轮未改动，均为基线值）

| 文件 | SHA256 | 字节 | 行数 |
|---|---|---|---|
| `cmd/han-server/recovery_sources.go` | `935DD7BDC3C3DF4B7FCF045622EEBC95D3BF30DEE313C93D19E1CC335B7DF90C` | 24136 | 662 |
| `cmd/han-server/recovery_runtime.go` | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` | 37431 | 1163 |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | 5044 |
| `cmd/han-server/recovery_config.go`（只读，零改动） | `83FFC39F498468BB81F4ED092862B46D6EE80B45B1BE74D6F502941C2EE606B0` | 9842 | 280 |
| `cmd/han-server/recovery_sources_test.go`（不得修改） | `BBCB47359E41A274FC5414FD2F5DFF35FBAE73FE816B76D241764E5A21188EF4` | 14797 | 390 |
| `cmd/han-server/recovery_anchor_production_integration_test.go` | **不存在（未创建）** | — | — |

输入文档基线（只读核对用）：

| 文件 | SHA256 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `E034155076F77F8D1E8288D9BB4014554A5019882051A03A305A0ACA6AD71EFB` |
| `.add/project/knowledge/decisions/framework.md` | `506FD1E0A8962BEB716488815EBA44060EE52B3105E4AC88CB22A8EC8893F23A` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `491A476E60636ADAA3F1610A07C78054ED4C9735F5AD42D60B76200A12C8414F` |
| `.spec/wip/recovery-runtime/preflight-s4-23/preflight-record.md` | `2FEFBF074F753D53CBB1779A6D26BCBCD7EA470F1563BA0A16A68AEDEAB1F05F` |
| `.spec/wip/recovery-runtime/preflight-s4-23/approval-record-form.md` | `7EDF6E870CDFF2DAA574E8CFBA7C79FC340A3C9C452B500730C88BF85C58E41A` |

## 7. 工作树与证据边界

- 未执行任何 Git 写操作（无 add/commit/checkout/switch/stash/restore/reset/push/建分支）。
- 本轮只写本报告；4 个代码文件零改动，哈希与基线一致；测试文件未创建。
- 未触碰他人未提交改动与工作区既有未跟踪文件；未清理、未删除任何内容。
- 未改 `tasks.md` 任何 checkbox（3152 行仍为 `- [ ]`）；未改 `framework.md`/`design.md`/`recovery_config.go`。
- 未回显或落盘任何凭据；`HAN_TEST_POSTGRES_DSN` 只核存在性（未打印值）；未创建/未清理任何 schema，未连接数据库执行 schema 操作。
- 基线各命令均在 HEAD `6da8abb` 上真实执行，输出已摘录；未伪造任何 PASS。

## 8. 需 Lead / Arch 决策事项

1. **adapter ID 字面值**：给出唯一权威登记值，并决策级修订 `design.md:3612`–`:3616`、`:3910` 与 `framework.md:926` 的"首期不提供生产 adapter"结论。
2. **witness 段 schema 与批准记录**：定义键名/类型/版本与机器可读批准记录（现有 `approval-record-form.md` 为空白表单，`preflight-record.md` §4 全待填）。
3. **anchor witness 实现合同**：给出传输协议、`(w,r,token)` 读-条件写 API、鉴权/双 fd wire、事件表读取路径；或明确本卡只交 fail-closed 骨架、把 (e) 幂等完成移到后续卡（与关闭条件 (8) 冲突，需决定级处理）。
4. **commitment/source 存储合同**：给出 `InstallationCommitmentStore`/`InstallationSourceVerifier` 的权威存储载体，及 `AnchorSnapshot` 字符串令牌 ↔ `bigint` revision 的映射规则。
5. **`recoveryRootInputs` seam 口径**：确认保留测试注入（生产 `main.go` 不传）还是后续卡收紧。
6. **平台口径**：Windows 上 (f6)–(f8) 与 ACL/GRANT 矩阵不能真实达成；是否改 POSIX 主机执行，或由 owner 登记豁免并接受本卡相应关闭条件未满足。

## 9. 声明

本报告是**停止回报**，不是评审 PASS、不是任务关闭、不是生产批准、不是 Git 写授权。任务卡 checkbox 由 Lead 决定，本角色未勾选、未宣称关闭。