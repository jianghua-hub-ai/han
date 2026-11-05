# B13J D03 证据报告：restore 工具、批准者与宿主隔离（ADR-0031）

- 项目项：`pending-B13J-D03`（restore 工具、批准者、宿主隔离与原生耐久回执）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 角色与产出者：Arch-Designer（`add-arch-designer`，Agent，非人的签字）
- 记录时间：2026-11-01（Asia/Shanghai）
- 路由：本轮是同一 Workflow 的普通续接，未重新 Route、未新建批次、未改任何 Task 状态
- 设计版本：`identity-auth-design-ir01-r59`（draft；人的精确接纳 unknown）

## 1. 结论

1. 新增 **ADR-0031**（`decisions/framework.md:971-1049`，锚点 `:970`）：*首期不提供生产 restore 工具与生产批准主体*，同时固定未来任何生产形态必须满足的批准、执行身份、宿主隔离与原生耐久回执不变量。
2. 生产 `enter`/`resume` 的 **DB 替换路径**继续 closed；仅保留既有 `restore-v1`/`verify-v1` 软件合同、确定性测试替身与 `restore_unknown` 保守暂停语义。`genesis`/`install`/`verify` 的既有软件语义不因本 ADR 改变。
3. `identity-auth/design.md` 同步到 r59：版本行、设计版本行、决策追溯行，以及 restore driver 段 `:3858`、`:3862`、`:3870` 三行原地替换；行数保持 4924，未改 `:3874-3902` 命令/ENV 表。
4. `decisions/README.md` 三处索引同步 ADR-0031（全局索引、领域文件表、场景查询表）。
5. 本轮**不声称**任何 restore 工具、备份、执行身份、ACL、进程隔离或旧备份演练已验证；不声称生产装配、Case18、S8.2 或恢复 E2E 通过；未解除 `blocked-upstream`。
6. 本轮为独立评审 `review-r01`（`B13J-D03-RV-01`～`RV-05`）与 `review-r02`（`B13J-D03-RV2-01`～`RV2-03`）两轮返工的合并结果：RV 处置见第 12 节，RV2 处置见第 13 节。相对 r02 基线，`design.md` 只改 `:3862`（写入边界两层化与受保护对象核对项）与 `:83`（同长术语修正），`framework.md` 只改 ADR-0031 内 `:1007`、`:1034`、`:1036`、`:1045` 四行（无增删行），`README.md` 零改动。

## 2. 决策论证摘要

### 2.1 现状

设计已把恢复的**软件合同**写得很细：`restore-v1`/`verify-v1` 的固定 argv、私有 stdin 管道、no-follow 稳定 fd/handle、最小 ENV allowlist、严格 receipt 字段、Begin 前预检失败即拒绝、`restore_unknown` 保守暂停。但“**由谁批准、由谁执行、用什么隔离、原生回执从哪来**”这层组织与信任边界在改动前全部是 unknown：

- `RestorePlan.SourceApprovalRef/SourceApprovalDigest` 没有可核的签发主体；
- `HAN_RECOVERY_RESTORE_EXECUTABLE` 指向的二进制没有批准记录（SHA256/owner/ACL/只读挂载来源）；
- 执行替换的进程身份、root 与执行者之间的权限边界未定义；
- 宿主隔离（进程组/子进程树、fd no-follow、ENV allowlist、无 shell）没有实测证明；
- “该 artifact 已还原”没有工具或备份系统原生写入、可独立重读的耐久回执样本。

因此生产 root 无法构造可验收的 `RestoreDriver`，生产 DB 替换只能在 Begin 前拒绝。

### 2.2 痛点

- 若直接声明“由部署方批准、由 root 子进程执行”，等于把一个不存在的批准角色、未指定的执行身份和未验证的隔离机制写成公共合同。
- 若维持“可配置但未定义”的模糊状态，实现者容易用 shell 包装、附加 argv、环境探测、caller 自报 done、DB 可连接或自己刚写的 run 行填补空缺。
- 另一类风险是把 `restore_unknown` 当缺陷，用自动重放“修复”，从而绕开持久回执证明。
- 想“先给一个有产出”的外部产品选型，会把未经批准的公共依赖和未验证的故障域写进架构。

### 2.3 候选对比

| 方案 | 实现机制 | 优势 | 劣势与未满足条件 | 结论 |
|---|---|---|---|---|
| A. 首期不提供生产 restore 工具与批准主体（选定） | 不注册生产 restore 可执行文件，不任命生产批准角色；生产 `enter`/`resume` 的 DB 替换保持 closed；仅保留软件合同、确定性测试替身与 `restore_unknown` | 不引入未验证工具、批准角色或隔离声明；与 ADR-0019 的 fail-closed 及“测试替身不得作为生产证据”一致；解除路径唯一且可审计 | 首期生产恢复链不可用；旧备份只能离线处理；后续必须重新选型、验证并修订 ADR 才能解除 closed | 选定 |
| B. 部署方受控只读挂载已批准二进制 + 具名部署批准角色 + root 子进程执行 | 部署方把已批准二进制放入受控只读挂载，以批准记录绑定 SHA256/owner/ACL；具名恢复批准角色批准备份来源；root 以固定 argv 启动子进程并保留锁与验证 | 与既有软件合同最接近；工具可独立升级与替换；root 仍持部署锁、Journal/keyring 与最终验证 | 当前无具名批准角色、无已批准二进制、无专用执行身份与权限矩阵、无进程组/no-follow/ENV allowlist 实测、无工具原生回执；工具供应链、更新责任与真实旧备份演练均未定 | 不选 |
| C. 独立恢复运维单元执行替换，本服务只做 VerifyRestored 与绑定 | 外部团队或系统在其环境执行 restore，本服务只验证恢复结果并写绑定 | 职责分离强；本服务不承载替换执行面 | 关键替换动作移出本服务审计与进程隔离边界；如何证明执行的是同一 artifact/target、如何取得原生回执、如何阻止并发写与重放、责任与时限均未定；当前无该单元 | 不选 |
| D. 托管数据库平台原生 PITR/备份恢复 + 平台原生回执 | 由托管 PostgreSQL 平台执行按时间点恢复，平台提供恢复任务记录作为原生回执，本服务只做 VerifyRestored 与绑定 | 平台原生持久回执；无需自管工具链与执行身份 | 锁定具体云产品、区域与凭据；需证明平台恢复能精确绑定 artifact/target/source installation 与 schema digest，且不会把平台自身回退误判为恢复成功；当前无实例与验证证据 | 不选 |

### 2.4 选 A 的论证

B、C、D 都要求当前不存在的具名批准角色、已批准二进制或平台实例、专用执行身份、隔离实测与原生回执证据。缺少这些证据时选择任一形态，等于把“未验证”写成“已批准可执行”，并把后续实施锁进未经批准的公共依赖或外部单元。A 不新增外部依赖，不触发 shell/附加 argv/自动重放/自动 fallback，保留既有软件合同与 `restore_unknown`，把解除条件固定为可核清单，是当前唯一可审计且不伪造生产就绪的结论。

### 2.5 解除条件（ADR-0031 §8）

只有同时满足以下条件，才可通过新的 ADR 修订或新 ADR 解除“首期不提供”和生产 closed：

1. 部署方任命具名恢复批准角色，并留存备份来源与可执行文件两条独立批准记录（含角色、人员、时间、依据）；
2. 已批准二进制具备可核 SHA256、owner/ACL、受控只读挂载与更新/回滚责任；
3. 专用执行身份与权限矩阵经批准：不得写 Journal/keyring/锁/Store 绑定，不得启动 server/requeue/migration/worker；
4. POSIX 宿主隔离实测通过：固定 argv 无 shell、独立进程组与完整子进程树 join/reap、no-follow 稳定 fd、最小 ENV allowlist、目标连接外无网络；
5. 负向验证门禁通过：以执行身份凭据写入 Journal/keyring/部署锁/Store 绑定/审计，以及该身份的目标 DB 角色写入 identity/recovery 表，均必须被拒绝；
6. 取得工具或其备份系统原生写入、可独立重读的耐久回执真实样本，并能与 artifact digest、target、source installation 逐项核对；
7. 以真实旧备份完成 source installation/schema digest/`VerifyRestored` 一致性演练并保留原始证据，不以 synthetic fixture 代替；
8. 通过架构与安全独立评审，完成 ADR 修订/新增决策，并由正式 Task 规划登记实现与验证范围；
9. 取得人的明确生产批准；Agent 产出、评审结论或测试通过均不替代该批准。

## 3. 四项一句话结论

- **批准主体**：部署方具名“恢复批准角色”分别签发备份来源批准与可执行文件批准两条独立记录（绑定 artifact digest、source installation 与二进制 SHA256/owner/ACL/只读挂载），该角色不得自批、不得兼任执行者或本次 run 发起者；当前项目未任命该角色，故无任何可核批准记录。生产锚点/密钥类批准不属本 ADR，仍由 ADR-0019/ADR-0030 约束。
- **执行主体**：替换只能由 root 启动的独立非交互子进程执行，该子进程须以独立于 root 的专用执行身份运行；root 保留部署锁、Journal/keyring、恢复 secret、Begin/绑定/审计与最终 `VerifyRestored` 判定，执行身份只获得只读 artifact fd、经私有 stdin 的目标连接配置、最小 ENV allowlist 与目标 DB 网络。写入边界分两层（RV2-01）：替换窗口外，该身份及其独立 DB 角色不得写 Journal/keyring/部署锁/Store 权威绑定与审计；替换窗口内，对目标库的写入只能来自已批准整库 artifact 的还原本身，不得自行构造/注入/声明权威 binding，本次权威 binding 只由 root 经同一 Store 短事务创建/核，artifact 内历史行不算本次权威。
- **宿主隔离**：固定 argv 直接 exec 唯一批准程序（无 shell、无附加 argv）、独立进程组/会话与完整子进程树 join/reap、no-follow 稳定 fd/handle 且校验前后 fstat 与路径身份一致、最小 ENV allowlist 零继承业务 secret 与审批 key；任一预检失败在 Begin 前拒绝。POSIX 是唯一生产候选但首期仍 closed；Windows 生产 unsupported，在任何状态变更前返回 unsupported（退出码 2），仅允许测试替身与软件试验。
- **首期是否提供**：不提供。生产 `enter`/`resume` 的 DB 替换路径保持 closed；仅保留既有软件合同、确定性测试替身与 `restore_unknown`。

## 4. 改动清单（逐行前后原文）

### 4.1 `identity-auth/design.md`（6 行原地修改，行数 4924 不变）

| 行 | 类型 | 改前原文 | 改后原文 |
|---|---|---|---|
| `:5` | 整行 | `revision: identity-auth-design-ir01-r58` | `revision: identity-auth-design-ir01-r59` |
| `:7` | 未改（核对） | `last_updated: 2026-11-01` | `last_updated: 2026-11-01`（同日多增量保持不变） |
| `:38` | 整行 | 版本号为 `identity-auth-design-ir01-r58`（整行为 `**设计版本**：` + 反引号包裹的版本号） | 版本号改为 `identity-auth-design-ir01-r59`，其余字符不变 |
| `:83` | 同行末尾追加 + RV2-02 术语修正（整段改前文本保留；最长公共前缀 1184 字符，尾部追加 462 字符 / 760 字节；`进程组`→`进程树` 等长等字节，行仍 1646 字符 / 2508 字节） | 见下 | 见下 |
| `:3858` | 整行 | 见下 | 见下 |
| `:3862` | 整行（D03 首交付 + RV-01/RV-02/RV-05 + RV2-01 合并后的最终文本；369 → 1265 字符，615 → 2601 字节） | 见下 | 见下 |
| `:3870` | 整行 | 见下 | 见下 |

#### `:83` 改前全文（1184 字符）

```text
**r57 / B13G S4.15 独立评审返工**：输入 r56 SHA256 `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF`。依据 [S4.15 独立报告](../../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-review-r01/report.md) 的 `S415-CR-F001/F002` 与 Main [精确 metadata START](../../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-findings-scope-sync-metadata-start-r01.json)，按用户 standing 授权采用原作者精准建议：补内部 `Runtime.ShutdownContext()` 只读发布及 ordinary Acquire 健康竞争回原等待。实际产出者 Linnaeus／add-feature-designer／Agent ID `01a18be5-1d5a-7c13-89e6-bc2173e19f22`（来自 Main 授权，非人的签字）。本版仅同步同一设计；S4.15 原 SOURCE10 加 Runtime 两源后的总范围为12，本次修复预计4文件，原7前置/5风险不变，不新建 Story/Task或重开已闭前置。仍 draft、人的精确接纳 unknown；两 HIGH 待代码修复及原 Reviewer 复审，不预签 PASS、关闭、checkpoint 或源码窗口。 **r58 / B13J-D02 生产 CAS Anchor provider 选型边界（ADR-0030）**：输入 r57 SHA256 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`。依据 accepted ADR-0030，首期不提供生产 CAS Anchor provider，S4.5 维持未交付、生产 enter/install 保持 closed；仅保留 provider-neutral `JournalHeadAnchor` port 与确定性测试替身，`HAN_RECOVERY_ANCHOR_ADAPTER` 生产必须显式指定且无隐式默认，未配置即不启动 anchor 能力并保持 closed；同步开发视图与配置表。实际产出者 Arch-Designer（Agent，非人的签字）；仍 draft、人的精确接纳 unknown，不声明实现、独立故障域验证或生产可用。
```

#### `:83` 改后全文（1646 字符，尾部新增 462 字符）

```text
**r57 / B13G S4.15 独立评审返工**：输入 r56 SHA256 `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF`。依据 [S4.15 独立报告](../../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-review-r01/report.md) 的 `S415-CR-F001/F002` 与 Main [精确 metadata START](../../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-findings-scope-sync-metadata-start-r01.json)，按用户 standing 授权采用原作者精准建议：补内部 `Runtime.ShutdownContext()` 只读发布及 ordinary Acquire 健康竞争回原等待。实际产出者 Linnaeus／add-feature-designer／Agent ID `01a18be5-1d5a-7c13-89e6-bc2173e19f22`（来自 Main 授权，非人的签字）。本版仅同步同一设计；S4.15 原 SOURCE10 加 Runtime 两源后的总范围为12，本次修复预计4文件，原7前置/5风险不变，不新建 Story/Task或重开已闭前置。仍 draft、人的精确接纳 unknown；两 HIGH 待代码修复及原 Reviewer 复审，不预签 PASS、关闭、checkpoint 或源码窗口。 **r58 / B13J-D02 生产 CAS Anchor provider 选型边界（ADR-0030）**：输入 r57 SHA256 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`。依据 accepted ADR-0030，首期不提供生产 CAS Anchor provider，S4.5 维持未交付、生产 enter/install 保持 closed；仅保留 provider-neutral `JournalHeadAnchor` port 与确定性测试替身，`HAN_RECOVERY_ANCHOR_ADAPTER` 生产必须显式指定且无隐式默认，未配置即不启动 anchor 能力并保持 closed；同步开发视图与配置表。实际产出者 Arch-Designer（Agent，非人的签字）；仍 draft、人的精确接纳 unknown，不声明实现、独立故障域验证或生产可用。 **r59 / B13J-D03 restore 工具、批准主体与宿主隔离（ADR-0031）**：输入 r58 SHA256 `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC`。依据 accepted ADR-0031，首期不提供生产 restore 工具与生产批准主体，生产 enter/resume 的 DB 替换在 Begin 前保持 closed；仅保留既有 `restore-v1`/`verify-v1` 软件合同、确定性测试替身与 `restore_unknown`；同步 restore driver 段的批准记录、专用执行身份、root 权限边界、进程树/no-follow/ENV allowlist 隔离、原生耐久回执来源与 POSIX/Windows 矩阵。实际产出者 Arch-Designer（Agent，非人的签字）；仍 draft、人的精确接纳 unknown，不声明实现、工具/备份/演练验证或生产可用。
```

#### `:3858` 改前

```text
`RestoreDriver` 是 root 中受信任备份工具的适配边界，不是 caller 自报恢复完成的 API；软件合同先固定，实际工具/备份批准者/执行主体 unknown 时生产 enter 在 Begin 前预检失败。recovery 只声明 `Prepare(ctx,RunBinding) (RestorePlan,error)`、`Replace(ctx,PhasePermit,RestorePlan) (RestoreReceipt,error)`、`VerifyRestored(ctx,RestorePlan) (RestoreReceipt,error)`；具体实现放 cmd，不能把测试 driver 注册进生产。
```

#### `:3858` 改后

```text
`RestoreDriver` 是 root 中受信任备份工具的适配边界，不是 caller 自报恢复完成的 API；软件合同先固定，首期不提供生产 restore 工具与生产批准主体（ADR-0031），生产 enter/resume 的 DB 替换在 Begin 前以 closed 拒绝，仅保留软件合同与确定性测试替身。recovery 只声明 `Prepare(ctx,RunBinding) (RestorePlan,error)`、`Replace(ctx,PhasePermit,RestorePlan) (RestoreReceipt,error)`、`VerifyRestored(ctx,RestorePlan) (RestoreReceipt,error)`；具体实现放 cmd，不能把测试 driver 注册进生产。
```

#### `:3862` 改前

```text
生产 adapter 执行配置中唯一批准程序，精确 argv 为 `<executable> restore-v1` 或只读 `<executable> verify-v1`，不用 shell、不允许 manifest 注入附加 argv。stdin 是私有管道的 strict v1 plan＋target connection 安全配置；不在 argv、stdout、日志或磁盘传 DSN/密码。传递环境采用最小 allowlist，零继承业务 secret/审批 key；root 保留部署锁，子进程不自行启动 server/requeue，不在替换期间运行 migration/bootstrap/worker。程序二进制 SHA、owner/ACL、受控 manifest 与 fd/进程树隔离预检失败均在 Begin 前拒绝。
```

#### `:3862` r02 返工后全文（1043 字符 / 2105 字节；本轮改前）

```text
生产 adapter 只允许执行配置中唯一批准程序：备份来源与可执行文件须由部署方具名恢复批准角色分别以批准记录登记，绑定 `SourceApprovalRef/Digest` 与二进制 SHA256、owner/ACL、受控只读挂载，批准者不得同时是执行者、本次 run 发起者或工具/备份供应方自批；执行主体是 root 启动的独立非交互子进程，须以独立于 root 的专用执行身份运行，root 不代替执行、不进入 shell、不重放，并保留部署锁、Journal/keyring、Store 绑定/审计写入与恢复 secret，不向子进程传递部署锁、Journal/keyring、审批 key、管理入口或除目标连接外的凭据。执行身份权限边界：只允许对目标 DB 执行 restore 与只读验证，不得写 Journal/keyring、部署锁、Store 绑定或审计，不得自行建立权威 binding；其目标连接凭据必须属于独立数据库角色，该角色权限不得覆盖 identity/recovery 表写入，数据库角色与 GRANT 矩阵由批准记录固定、未证明即保持 closed；整库 artifact 与这些受保护表的边界（排除相应 schema 或经隔离库后由 root 受控切换）同由批准记录固定。精确 argv 为 `<executable> restore-v1` 或只读 `<executable> verify-v1`，不用 shell、不允许 manifest 注入附加 argv；执行身份切换不得经 shell 脚本、su、sudo 等包装实现，具体机制由批准记录固定并在实现阶段以隔离实测证明。stdin 是私有管道的 strict v1 plan＋target connection 安全配置；不在 argv、stdout、日志或磁盘传 DSN/密码。传递环境采用最小 allowlist，零继承业务 secret/审批 key；执行身份除目标 DB 连接外无网络出口；子进程不自行启动 server/requeue，不在替换期间运行 migration/bootstrap/worker。程序二进制 SHA、owner/ACL、受控 manifest、执行身份与 fd/进程树隔离预检失败均在 Begin 前拒绝。POSIX 是唯一生产候选且首期仍 closed；Windows 生产 unsupported，须在任何状态变更前拒绝，仅允许软件合同测试。
```

#### `:3862` r03 改后全文（1265 字符 / 2601 字节）

```text
生产 adapter 只允许执行配置中唯一批准程序：备份来源与可执行文件须由部署方具名恢复批准角色分别以批准记录登记，绑定 `SourceApprovalRef/Digest` 与二进制 SHA256、owner/ACL、受控只读挂载，批准者不得同时是执行者、本次 run 发起者或工具/备份供应方自批；执行主体是 root 启动的独立非交互子进程，须以独立于 root 的专用执行身份运行，root 不代替执行、不进入 shell、不重放，并保留部署锁、Journal/keyring、Store 绑定/审计写入与恢复 secret，不向子进程传递部署锁、Journal/keyring、审批 key、管理入口或除目标连接外的凭据。执行身份权限边界：只允许对目标 DB 执行 restore 与只读验证；目标连接凭据必须属于独立数据库角色，数据库角色与 GRANT 矩阵由批准记录固定、未证明即保持 closed；替换窗口之外，该角色权限不得覆盖 Journal/keyring、部署锁、Store 权威绑定与审计记录的任何写入；替换窗口之内，对目标库的写入只能来自已批准 artifact 的整库还原本身，执行身份不得自行构造、注入或声明任何权威 binding，也不得写 Journal/keyring 或部署锁，替换完成后由 root 以独立读取路径与同一 Store 短事务创建/核本次 running 与不可变 binding（同下文 Replace 合同），artifact 内既有历史行不得当作本次权威；受保护对象以批准记录固定的可检清单（Journal、keyring、部署锁、Store 权威绑定与审计记录）作为 restore 完成后的核对项，不改变 restore 的整库语义，也不引入 schema 级排除或 root 数据搬移。精确 argv 为 `<executable> restore-v1` 或只读 `<executable> verify-v1`，不用 shell、不允许 manifest 注入附加 argv；执行身份切换不得经 shell 脚本、su、sudo 等包装实现，具体机制由批准记录固定并在实现阶段以隔离实测证明。stdin 是私有管道的 strict v1 plan＋target connection 安全配置；不在 argv、stdout、日志或磁盘传 DSN/密码。传递环境采用最小 allowlist，零继承业务 secret/审批 key；执行身份除目标 DB 连接外无网络出口；子进程不自行启动 server/requeue，不在替换期间运行 migration/bootstrap/worker。程序二进制 SHA、owner/ACL、受控 manifest、执行身份与 fd/进程树隔离预检失败均在 Begin 前拒绝。POSIX 是唯一生产候选且首期仍 closed；Windows 生产 unsupported，须在任何状态变更前拒绝，仅允许软件合同测试。
```

#### `:3870` 改前

```text
制品校验与实际 restore 必须使用同一 no-follow 打开的稳定 fd/handle（需要 seek 时仍用该 handle）；核读前后 fstat/路径身份、完整字节数与摘要，禁止验完 hash 再按可替换路径重新打开。工具/ACL/进程隔离的真实信任未证明时，这个软件协议仍不足以启用生产替换。
```

#### `:3870` 改后

```text
制品校验与实际 restore 必须使用同一 no-follow 打开的稳定 fd/handle（需要 seek 时仍用该 handle）；核读前后 fstat/路径身份、完整字节数与摘要，禁止验完 hash 再按可替换路径重新打开。原生耐久回执只能由 restore 工具或其备份系统原生写入、独立于本次进程可重读的持久记录（恢复目录、恢复任务记录或安装来源证据）提供，并须绑定 artifact digest、target 与 source installation；root 以独立读取路径与批准记录逐项核对，不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证。工具/批准记录/执行身份/ACL/进程隔离的真实信任未证明时，这个软件协议仍不足以启用生产替换；当前无批准工具、批准记录、隔离与原生回执证据，生产替换保持 closed。
```
### 4.2 `decisions/framework.md`（末尾追加 ADR-0031，83 行）

- 追加范围：`:967`（空行）、`:968`（`---`）、`:969`（空行）、`:970`（`<a id=adr-0031></a>`）、`:971`（标题）至 `:1049`（决策授权行）。
- 改前该位置不存在任何字节；前 137491 字节（原 966 行，含末尾 CRLF）逐字节不变。
- 分隔风格与 ADR-0029/ADR-0030 一致：空行 → `---` → 空行 → anchor → 标题。
- review-r01 返工后本段为 1049 行（`:967` 空行起）；review-r02 返工（`RV2-01`/`RV2-02`）只改其中 `:1007`、`:1034`、`:1036`、`:1045` 四行，无增删行，行数仍为 1049。以下为 review-r02 返工后的当前逐行原文。
- 全文如下（逐行原文）：

```text
---

<a id=adr-0031></a>
## ADR-0031: 首期不提供生产 restore 工具与批准主体，固定受批准执行与宿主隔离合同 (2026-11-01)

**目的**: 架构设计 | **仓库**: han | **修订**: adr-0031-r01

**状态**: 已接受（2026-11-01 用户 D03 指令授权 Arch-Designer 论证并落定“首期不提供生产 restore 工具与生产批准主体”；确认来源为本轮用户任务指令；不构成人的成果接收、独立评审、安全评审、生产装配批准或真实工具/备份验证） | **确认来源**: 用户 2026-11-01 在 B13J D03 中明确要求把受批准 restore 工具、备份批准主体、执行主体、宿主隔离与原生耐久回执落成可审计、fail-closed 的架构决策，并明确“首期不提供”是合法结论之一。该来源授权的是架构决策与文档写入，不是生产装配、真实旧备份演练、安全评审或人的成果签收。

**继承关系**: 只固定 `identity-auth/design.md` §4.2 中“受批准 restore 工具 / 备份批准主体 / 执行主体 / 宿主隔离 / 原生耐久回执”这组组织与信任边界；不改变 ADR-0019 已锁定的独立故障域 CAS 头锚点、keyring、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒与 fail-closed 失败分类；不改 ADR-0030 的“首期不提供生产 CAS Anchor provider、S4.5 未交付、生产 enter/install 保持 closed”；不改变 ADR-0027 的恢复 runtime 编排与正向准入证明；不重写或削弱 design 既有 restore 软件合同（`restore-v1`/`verify-v1` 的 argv 与私有 stdin、no-follow 稳定 fd、最小 ENV allowlist、receipt 字段、`restore_unknown` 保守暂停、Begin 前预检失败即拒绝），只补其批准、执行身份、宿主隔离与原生回执这组前置前提；若后续发现软件合同本身有缺口，另立决策或设计修订处理，不在本 ADR 内改写。

**问题**: 设计已固定 restore 的软件协议与 Begin 前预检，但没有可核的批准主体、执行主体、执行身份与进程隔离证明，也没有工具原生耐久回执的真实样本：`RestorePlan.SourceApprovalRef/Digest` 由谁签发、`HAN_RECOVERY_RESTORE_EXECUTABLE` 的二进制 SHA/owner/ACL 由谁批准、由哪个进程以什么身份执行替换、root 与执行者之间的权限边界、宿主隔离的最小必要机制、以及“该 artifact 已还原”由谁以什么形态提供持久证据，改动前均为 unknown。生产 root 因此无法构造可验收的 `RestoreDriver`，生产 `enter`/`resume` 的 DB 替换只能在 Begin 前拒绝。需要在不虚构未验证工具、批准角色或演练结果的前提下，给出唯一可验收口径。

**决策**: 首期不提供生产 restore 工具与生产批准主体。生产 `enter`/`resume` 的 DB 替换路径继续保持 closed；与 ADR-0030 的合并口径是“生产 `enter`/`install` 与 `enter`/`resume` 的 DB 替换路径均保持 closed”，二者是不同入口而非范围收窄。仅保留既有软件合同、确定性测试替身与 `restore_unknown` 语义。`HAN_RECOVERY_RESTORE_EXECUTABLE`、`HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 与 `HAN_RECOVERY_RESTORE_MANIFEST_PATH` 仍是唯一入口且无默认值、未知不补值；`verify` 与 `genesis` 形态不执行工具。任何未来生产形态必须同时满足下节全部批准、执行身份、宿主隔离与原生回执不变量，否则不得解除 closed。

### 现状 / 痛点 / 改造 / 效果

现状是“协议已定、执行主体未定”：设计已把 `restore-v1`/`verify-v1` 的 argv、私有 stdin、receipt 字段、no-follow 稳定 fd、最小 ENV、Begin 前预检与 `restore_unknown` 写成软件合同；但没有任何具名批准角色、已批准二进制 SHA、专用执行身份、权限矩阵、进程隔离实测或工具原生耐久回执证据。当前不存在可执行的生产 restore 工具，也不存在可核的备份来源批准记录。

痛点在于：若现在声明“由部署方批准、由 root 子进程执行”，等于把一个不存在的批准角色、未指定的执行身份和未验证的隔离机制写成公共合同；若继续维持“可配置但未定义”的模糊状态，实施者容易用 shell 包装、附加 argv、环境探测、caller 自报 done、DB 可连接或自己刚写的 run 行填补空缺。另一类风险是把 `restore_unknown` 当缺陷而用自动重放“修复”。

| 方案 | 实现机制 | 优势 | 劣势与未满足条件 | 结论 |
|---|---|---|---|---|
| A. 首期不提供生产 restore 工具与批准主体（选定） | 不注册生产 restore 可执行文件，不任命生产批准角色；生产 `enter`/`resume` 的 DB 替换保持 closed；仅保留软件合同、确定性测试替身与 `restore_unknown` | 不引入未验证工具、批准角色或隔离声明；与 ADR-0019 的 fail-closed 及“测试替身不得作为生产证据”一致；解除路径唯一且可审计 | 首期生产恢复链不可用；旧备份只能离线处理；后续必须重新选型、验证并修订 ADR 才能解除 closed | 选定 |
| B. 部署方受控只读挂载已批准二进制 + 具名部署批准角色 + root 子进程执行 | 部署方把已批准二进制放入受控只读挂载，以批准记录绑定 SHA256/owner/ACL；具名恢复批准角色批准备份来源；root 以固定 argv 启动子进程并保留锁与验证 | 与既有软件合同最接近；工具可独立升级与替换；root 仍持部署锁、Journal/keyring 与最终验证 | 当前无具名批准角色、无已批准二进制、无专用执行身份与权限矩阵、无进程组/no-follow/ENV allowlist 实测、无工具原生回执；工具供应链、更新责任与真实旧备份演练均未定 | 不选 |
| C. 独立恢复运维单元执行替换，本服务只做 VerifyRestored 与绑定 | 外部团队或系统在其环境执行 restore，本服务只验证恢复结果并写绑定 | 职责分离强；本服务不承载替换执行面 | 关键替换动作移出本服务审计与进程隔离边界；如何证明执行的是同一 artifact/target、如何取得原生回执、如何阻止并发写与重放、责任与时限均未定；当前无该单元 | 不选 |
| D. 托管数据库平台原生 PITR/备份恢复 + 平台原生回执 | 由托管 PostgreSQL 平台执行按时间点恢复，平台提供恢复任务记录作为原生回执，本服务只做 VerifyRestored 与绑定 | 平台原生持久回执；无需自管工具链与执行身份 | 锁定具体云产品、区域与凭据；需证明平台恢复能精确绑定 artifact/target/source installation 与 schema digest，且不会把平台自身回退误判为恢复成功；当前无实例与验证证据 | 不选 |

选择 A。B、C、D 都需要当前不存在的具名批准角色、已批准二进制或平台实例、专用执行身份、隔离实测与原生回执证据；在缺少这些证据时选择任一形态，等于把“未验证”写成“已批准可执行”。A 不新增外部依赖，不触发 shell/附加 argv/自动重放/自动 fallback，保留既有软件合同与 `restore_unknown`，并把解除条件固定为可核清单，是当前唯一可审计且不伪造生产就绪的结论。

改造是把“谁批准、谁执行、用什么隔离、回执从哪来”从实现细节提升为生产构造门禁：未来任何生产 restore 形态只有同时具备具名批准记录、已批准二进制、专用执行身份、隔离实测与工具原生存久回执，才可能通过 Begin 前预检；任一缺失一律 closed，且不得用 caller 自报、exit=0、stdout receipt 自洽、DB 可连接或本次 run 行替代。

效果是：生产 root 不会因环境变量存在、文件可执行或 manifest 可读而误开替换；测试替身继续只服务协议测试；`restore_unknown` 保持为明确恢复结果而非缺陷；解除 closed 需要一次性补齐全部证据并经独立评审与正式登记，避免实现阶段自行补默认值或引入未批准工具。

### 批准、执行与隔离不变量

1. **备份来源批准**：`RestorePlan.SourceApprovalRef/SourceApprovalDigest` 只能由部署方具名“恢复批准角色”签发，批准记录须写明角色、人员、时间、备份来源、artifact digest 与 source installation；该角色不得是执行者、本次 run 发起者或工具供应方自批。当前项目未任命该角色，故不存在可核批准记录。
2. **可执行文件批准**：`HAN_RECOVERY_RESTORE_EXECUTABLE` 指向的二进制须由同一具名批准角色以独立批准记录登记绝对路径、SHA256、owner/ACL 与受控只读挂载来源，并与 `HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 精确相等；不存在“默认部署方角色”“按目录推断”或“运行时自行选最新版本”。当前无已批准二进制。
3. **执行主体**：替换只能由 root 启动的独立非交互子进程执行，且该子进程须以独立于 root 的专用执行身份运行；root 不代替执行、不进入 shell、不重放，也不把部署锁、Journal/keyring、审批 key、管理入口或除目标连接外的凭据传给子进程。执行身份切换与程序启动禁止经 shell 脚本、`su`、`sudo` 等包装实现（与固定 argv、无 shell 冲突）；具体切换机制由批准记录固定，并要求在实现阶段以隔离实测证明，本 ADR 不选具体机制。
4. **root 与执行身份的权限边界**：root 保留部署锁、排空/独占、Begin/绑定/审计与最终 `VerifyRestored` 判定；执行身份只获得只读 artifact fd、经私有 stdin 传入的目标连接配置、最小 ENV allowlist 与目标 DB 网络。替换窗口外该身份不得写 Journal/keyring/锁/Store 权威绑定或审计；替换窗口内对目标库的写入只能来自已批准整库 artifact 的还原本身，该身份不得自行构造、注入或声明权威 binding，也不得写 Journal/keyring 或部署锁，本次权威 binding 只由 root 经同一 Store 短事务创建/核（design `:3866`）。该身份不得启动 server、requeue、migration、bootstrap 或 worker。
5. **宿主隔离最小机制（POSIX/Windows 矩阵）**：

| 机制 | POSIX（唯一生产候选，首期 closed） | Windows（生产 unsupported） |
|---|---|---|
| 启动方式 | 固定 argv 直接 exec 唯一批准程序；无 shell、无附加 argv | 同一软件合同，但仅允许测试替身与软件试验 |
| 进程树 | 独立进程组/会话；父进程持有全部子进程句柄，取消或超时时终止整组并真实 join/reap | 仅以 Job Object 等价的软件试验验证，不构成生产支持 |
| 文件 | no-follow 稳定 fd/handle；校验前后 fstat 与路径身份一致；禁止验后按路径重开 | 同一软件合同，仅测试 |
| 环境 | 最小 ENV allowlist；零继承业务 secret、审批 key 与宿主凭据 | 同一软件合同，仅测试 |
| 平台门禁 | 任一隔离预检失败在 Begin 前拒绝 | 任何 Journal/DB/CAS/文件状态变更前返回 unsupported（退出码 2） |

6. **原生耐久回执**：只能由 restore 工具或其备份系统原生写入、独立于本次进程可重读的持久记录（恢复目录、恢复任务记录或安装来源证据）提供，且须绑定 artifact digest、target 与 source installation；root 必须经独立读取路径与批准记录逐项核对，不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证。具体产品字段与读取方式属工具/部署输入，当前 unknown。

上述六项在当前均为缺失状态，故生产 restore closed；这不削弱既有软件合同与测试替身，也不改变 `restore_unknown` 的保守暂停语义。

### 约束

1. **不改变既有决定与合同**：ADR-0019、ADR-0030、ADR-0027 与 design 既有 restore 软件合同继续有效；本 ADR 只补组织与信任边界。
2. **禁止执行通道扩张**：不得引入 shell、附加 argv、manifest 注入参数、自动重放、自动 fallback，也不得以 file/SQL/fake、环境探测或“目录里最新二进制”填补缺失工具。
3. **禁止伪证明**：caller 自报 done、exit=0、stdout receipt 自洽、DB 可连接、当前行数、自己刚写的 run 行、测试替身或空构造均不得作为恢复证明。
4. **批准与执行分离**：批准角色必须具名且不得自批、不得兼任执行者；执行身份不得等于 root，也不得与批准角色共用凭据。
5. **`restore_unknown` 保留**：无可核持久回执时保持 `restore_unknown`，由人工经原受控备份工具核验/修复后 verify/resume；不得用自动重放或人工勾选 done 越过。
6. **不声称验证**：本 ADR 不证明任何 restore 工具、备份系统、托管平台、执行身份、ACL、进程隔离或旧备份演练已通过；不声称生产装配、Case18、S8.2 或恢复 E2E 通过。
7. **不登记实现任务**：本 ADR 不预占任何未登记 Task 编号；实现与真实演练须由正式 Task 规划登记。
8. **解除条件**：只有同时满足以下条件，才可通过新的 ADR 修订或新 ADR 解除“首期不提供”和生产 closed：
   - 部署方任命具名恢复批准角色，并留存备份来源与可执行文件两条独立批准记录（含角色、人员、时间、依据）；
   - 已批准二进制具备可核 SHA256、owner/ACL、受控只读挂载与更新/回滚责任；
   - 专用执行身份与权限矩阵经批准：替换窗口外不得写 Journal/keyring/锁/Store 权威绑定与审计；替换窗口内写入只能来自已批准整库 artifact 的还原本身，本次权威 binding 只由 root 写；不得启动 server/requeue/migration/worker；
   - POSIX 宿主隔离实测通过：固定 argv 无 shell、独立进程组与完整子进程树 join/reap、no-follow 稳定 fd、最小 ENV allowlist、目标连接外无网络；
   - 负向验证门禁通过：替换窗口外，以执行身份的凭据或其目标 DB 角色写入 Journal/keyring/部署锁/Store 权威绑定/审计必须被拒绝；替换窗口内，除已批准整库 artifact 的还原本身外不得产生任何 Store 权威绑定或 Journal/keyring/部署锁写入，执行身份自行构造、注入或声明的 binding 必须被拒绝；
   - 取得工具或其备份系统原生写入、可独立重读的耐久回执真实样本，并能与 artifact digest、target、source installation 逐项核对；
   - 以真实旧备份完成 source installation/schema digest/`VerifyRestored` 一致性演练并保留原始证据，不以 synthetic fixture 代替；
   - 通过架构与安全独立评审，完成 ADR 修订/新增决策，并由正式 Task 规划登记实现与验证范围；
   - 取得人的明确生产批准；Agent 产出、评审结论或测试通过均不替代该批准。
9. **不适用场景**：本 ADR 不适用于 ADR-0019/ADR-0030 的 Anchor/keyring 与生产 CAS provider、ADR-0027 的 runtime 编排与正向准入、design §4.2 的 restore 软件协议本身，以及 D04 承接的部署独占、`ArticleAccountPort` 与 recovery secret candidate。Windows 生产仍 unsupported，本 ADR 不提供 Windows 例外。

### 引用记录

- 2026-11-01 `identity-auth` 设计修订 `identity-auth-design-ir01-r59`（含 D03 评审返工）采用本决定：restore driver 段 `.add/project/knowledge/design-workspace/identity-auth/design.md:3858` 固定“首期不提供生产 restore 工具与批准主体、生产 enter/resume 的 DB 替换在 Begin 前 closed、仅保留软件合同与测试替身”；`:3862` 固定批准记录（含工具/备份供应方不得自批）、专用执行身份、root 保留部署锁/Journal/keyring/Store 绑定与审计写入的权限边界、进程树/ENV allowlist 与 fd 隔离预检（no-follow 稳定 fd 见 `:3870`）及 POSIX/Windows 矩阵，并固定执行身份的两层写入边界——替换窗口外其角色权限不得覆盖 Journal/keyring/部署锁/Store 权威绑定与审计的任何写入，替换窗口内写入只能来自已批准 artifact 的整库还原、不得自行构造或声明权威 binding、替换后由 root 经同一 Store 短事务创建/核本次 binding，受保护对象清单只作 restore 完成后的核对项、不改变整库语义且无 schema 级排除或 root 数据搬移；其目标连接须为独立数据库角色且角色与 GRANT 矩阵由批准记录固定、未证明即 closed，以及除目标连接外无网络与禁止 su/sudo/shell 脚本包装；`:3870` 固定原生耐久回执的来源形态与“信任未证明不得启用生产替换”；配置表 `:3896` 的三个 restore 键语义不变；决策追溯 `:83` 登记本 ADR。设计仍为 draft，人的精确接纳 unknown，不声明实现或验证。
- 硬边界：[framework.md（ADR-0019）](framework.md#adr-0019)，`framework.md:418` 起；本 ADR 不重写其任何已锁定决定。
- 相邻决定：[framework.md（ADR-0030）](framework.md#adr-0030)，`framework.md:916` 起；本 ADR 不改变“首期不提供生产 anchor”。
- 缺口来源：[gap-map.md](../../../../.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md) 的 §2.5 `pending-B13J-D03`（:121-139）与 U4；该文件要求工具、批准者、执行主体、ACL/进程隔离、原生回执与 unknown 暂停均具体。
- 决策授权：用户 2026-11-01 B13J D03 指令；产出者为 Arch-Designer（Agent），不是人的签字、独立评审、安全评审或生产批准。
```

### 4.3 `decisions/README.md`（2 行新增 + 1 行原地修改）

| 行（改后） | 类型 | 改前原文 | 改后原文 |
|---|---|---|---|
| `:39` | 新增全局索引行 | （无） | `| ADR-0031 | [首期不提供生产 restore 工具与批准主体，固定受批准执行与宿主隔离合同](framework.md#adr-0031) | 已接受（2026-11-01 D03 架构决策授权；首期不提供生产 restore 工具与生产批准主体，生产 enter/resume 的 DB 替换保持 closed；固定批准记录、专用执行身份、root 权限边界、宿主隔离与原生回执不变量；无真实工具/备份/演练验证） | 架构设计 | 框架 | 生产 restore 工具与可执行文件批准、备份来源批准、执行主体与 root 权限边界、进程组/no-follow/ENV allowlist 宿主隔离、原生耐久回执、POSIX/Windows 矩阵、首期不提供与解除条件 | 1 |` |
| `:45` | 原地修改 framework.md 领域行 | `| framework.md | 框架与技术栈 | ADR-0001、ADR-0002、ADR-0003、ADR-0004、ADR-0008、ADR-0009、ADR-0014、ADR-0015、ADR-0016、ADR-0017、ADR-0018、ADR-0019、ADR-0021、ADR-0022、ADR-0025、ADR-0027、ADR-0028、ADR-0029、ADR-0030 |` | 同行末尾追加 `、ADR-0031`，其余字符不变 |
| `:71` | 新增场景查询行 | （无） | `| 选择或调整生产 restore 工具、备份来源与可执行文件批准主体、执行主体与 root 权限边界、宿主隔离（进程组/no-follow/ENV allowlist/无 shell）、原生耐久回执、首期不提供或解除条件 | [framework.md（ADR-0031）](framework.md#adr-0031)、framework.md（ADR-0027、ADR-0018） |` |

两处新增行均为 LF 行尾；原有两个 CRLF 行（改后 `:36`、`:59`）未变。

## 5. 改前改后实测值

测量方法：`[IO.File]::ReadAllBytes` → SHA256；`[IO.File]::ReadAllText(..., UTF8)` → 按 `` `n `` 切分计行数、统计 CRLF 与裸 LF；BOM 检查前三字节；末尾行尾取最后两字节；控制字符按 `<32` 且非 `13/10` 计数。

| 文件 | 阶段 | SHA256 | 字节 | 行数 | crlf / bareLF | BOM | 末尾 | 控制字符 |
|---|---|---|---|---|---|---|---|---|
| `identity-auth/design.md` | D03 初始基线 | `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC` | 844091 | 4924 | 169 / 4755 | false | 单 LF（`3E-0A`） | 1（既有 TAB） |
| `identity-auth/design.md` | 评审返工前（本轮基线） | `79F226D5BB152C7A2DF359548CBFE24C306DF54A1B2EFD8BED71A80F488B2037` | 846117 | 4924 | 169 / 4755 | false | 单 LF（`3E-0A`） | 1（既有 TAB） |
| `identity-auth/design.md` | review-r01 返工后（review-r02 基线） | `B452E38E7074B1D483CB8A0EB511A7EF858B3F35C13CC0841A75118E4B7F013C` | 846941 | 4924 | 169 / 4755 | false | 单 LF（`3E-0A`） | 1（既有 TAB） |
| `identity-auth/design.md` | review-r02 返工后（最终） | `B07BAFC7CE1265A42B4F9A79C0F1489D3FCAAB50F47F1C02F0A1FCDA3C23AF99` | 847437 | 4924 | 169 / 4755 | false | 单 LF（`3E-0A`） | 1（既有 TAB） |
| `decisions/framework.md` | D03 初始基线 | `33281CB5CF6807A9B8B57825CD8A7D9A2AFBED52CC899D9F95D64196DD1CF532` | 137491 | 966 | 966 / 0 | false | 单 CRLF（`0D-0A`） | 0 |
| `decisions/framework.md` | 评审返工前（本轮基线） | `A80F5DD3F542AD15EC9BB717BAF20C4CA43EC83EB1B215171E1DA41516EB0589` | 153033 | 1048 | 1048 / 0 | false | 单 CRLF（`0D-0A`） | 0 |
| `decisions/framework.md` | review-r01 返工后（review-r02 基线） | `3A3B2B214D9732BB61A0C8768CEF701B08F65ED3D8161C85F254A0F4B37275B8` | 154074 | 1049 | 1049 / 0 | false | 单 CRLF（`0D-0A`） | 0 |
| `decisions/framework.md` | review-r02 返工后（最终） | `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12` | 155139 | 1049 | 1049 / 0 | false | 单 CRLF（`0D-0A`） | 0 |
| `decisions/README.md` | D03 初始基线 | `5075879AC7F3E31D83A06E843BCCE80CAA8DBA27F588E39D700E7F6C3765C10D` | 19724 | 106 | 2 / 104 | false | 单 LF（`82-0A`） | 0 |
| `decisions/README.md` | D03 全程零改动（review-r02 复测相同） | `A67B00D5BF050E100EA5AEC9DEBBFB8E5DABCE4463448658D68A5B4792E0F706` | 20764 | 108 | 2 / 106 | false | 单 LF（`82-0A`） | 0 |

补充说明：

- `design.md` review-r01 返工只改 `:3862` 一行（整行原地替换），字节 846117 → 846941；review-r02 返工改 `:83`（术语 `进程组`→`进程树`，等长等字节，行仍 1646 字符 / 2508 字节）与 `:3862`（1043 → 1265 字符，2105 → 2601 字节），字节 846941 → 847437（+496）。行数 4924 与 crlf 169 / bareLF 4755 全程不变。唯一控制字符仍是既有 TAB（0x09），位于 UTF-8 字节偏移 466829 的 Go 代码块内（行 3303，早于本轮所有改动点）；两轮返工引入 0 个控制字符。
- `framework.md` 全程纯 CRLF：review-r02 返工后 1049 行，crlf 计数等于行数，bareLF 恒为 0；末尾保持单 CRLF。review-r01 返工修正 `:981`、`:1006`、`:1045`（原 `:1044`，因新增行后移一位）并在 `:1036` 新增 1 行（153033 → 154074）；review-r02 返工只改 `:1007`、`:1034`、`:1036`、`:1045` 四行、无增删行（154074 → 155139，+1065）。
- `framework.md` 前 137491 字节与 D03 初始基线逐字节相同（SHA256 `33281CB5…1CF532`，逐字节 `SequenceEqual=True`），ADR-0019、ADR-0030、ADR-0027 正文未变。
- `README.md` D03 全程零改动：review-r02 复测与 review-r01 时的状态逐字节相同（`A67B00D5…`/20764/108 行，crlf 2 / bareLF 106）。

## 6. 反向还原验证

改前副本（`RV2-03`）：本轮动笔前把 `framework.md` 与 `design.md` 各原样复制一份到本证据目录 `snapshots/`（只复制、不修改原文件），实测等于本轮基线：

- `snapshots/design-before-r03.md`：SHA256 `B452E38E7074B1D483CB8A0EB511A7EF858B3F35C13CC0841A75118E4B7F013C`，846941 字节。
- `snapshots/framework-before-r03.md`：SHA256 `3A3B2B214D9732BB61A0C8768CEF701B08F65ED3D8161C85F254A0F4B37275B8`，154074 字节。
- `decisions/README.md` 本轮零改动，无需快照（D03 全程未改）。

| 文件 | 撤销方式 | 重建结果 | 结论 |
|---|---|---|---|
| `design.md` | 用快照 `:83`、`:3862` 两行原文整行替换当前对应行，其余字节不动 | SHA256 `B452E38E…7F013C`，846941 字节，与 `snapshots/design-before-r03.md` 逐字节 `SequenceEqual=True` | REVERSE-OK，精确回到本轮基线 |
| `framework.md` | 用快照恢复 `:1007`、`:1034`、`:1036`、`:1045` 四行原文（无增删行） | SHA256 `3A3B2B21…7275B8`，154074 字节，与 `snapshots/framework-before-r03.md` 逐字节 `SequenceEqual=True` | REVERSE-OK，精确回到本轮基线 |
| `README.md` | 无需撤销（本轮零改动） | SHA256 `A67B00D5BF050E100EA5AEC9DEBBFB8E5DABCE4463448658D68A5B4792E0F706`，20764 字节，与基线逐字节相同 | UNCHANGED-OK |

逐行差异核对（与 `snapshots/` 改前副本按行逐元素比较；行数不变，无需重新对齐）：

- `design.md` 内容差异行号恰为 `83`（等长术语替换）与 `3862`（整行替换），其余 4922 行逐字符相同。
- `framework.md` 内容差异行号恰为 `1007`、`1034`、`1036`、`1045`，其余 1045 行逐字符相同；其前 137491 字节仍与 D03 初始基线逐字节相同。
- 更早两轮的基线分别为：D03 首交付 design `30B122A0…`/844091、framework `33281CB5…`/137491、README `5075879A…`/19724；review-r01 返工后 design `79F226D5…`/846117、framework `A80F5DD3…`/153033。本轮以 review-r02 返工后为基线，不改变更早结论。

## 7. 验收命令输出

### 7.1 design 关键字检索

命令：

```text
rg -n "restore-v1|verify-v1|批准者|批准主体|执行主体|进程树|no-follow|restore_unknown" .add/project/knowledge/design-workspace/identity-auth/design.md
```

退出码：`0`；命中 **13** 行。

输出：

```text
83:**r57 / B13G S4.15 独立评审返工**：输入 r56 SHA256 `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF`。依据 [S4.15 独立报告](../../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-review-r01/report.md) 的 `S415-CR-F001/F002` 与 Main [精确 metadata START](../../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-findings-scope-sync-metadata-start-r01.json)，按用户 standing 授权采用原作者精准建议：补内部 `Runtime.ShutdownContext()` 只读发布及 ordinary Acquire 健康竞争回原等待。实际产出者 Linnaeus／add-feature-designer／Agent ID `01a18be5-1d5a-7c13-89e6-bc2173e19f22`（来自 Main 授权，非人的签字）。本版仅同步同一设计；S4.15 原 SOURCE10 加 Runtime 两源后的总范围为12，本次修复预计4文件，原7前置/5风险不变，不新建 Story/Task或重开已闭前置。仍 draft、人的精确接纳 unknown；两 HIGH 待代码修复及原 Reviewer 复审，不预签 PASS、关闭、checkpoint 或源码窗口。 **r58 / B13J-D02 生产 CAS Anchor provider 选型边界（ADR-0030）**：输入 r57 SHA256 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`。依据 accepted ADR-0030，首期不提供生产 CAS Anchor provider，S4.5 维持未交付、生产 enter/install 保持 closed；仅保留 provider-neutral `JournalHeadAnchor` port 与确定性测试替身，`HAN_RECOVERY_ANCHOR_ADAPTER` 生产必须显式指定且无隐式默认，未配置即不启动 anchor 能力并保持 closed；同步开发视图与配置表。实际产出者 Arch-Designer（Agent，非人的签字）；仍 draft、人的精确接纳 unknown，不声明实现、独立故障域验证或生产可用。 **r59 / B13J-D03 restore 工具、批准主体与宿主隔离（ADR-0031）**：输入 r58 SHA256 `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC`。依据 accepted ADR-0031，首期不提供生产 restore 工具与生产批准主体，生产 enter/resume 的 DB 替换在 Begin 前保持 closed；仅保留既有 `restore-v1`/`verify-v1` 软件合同、确定性测试替身与 `restore_unknown`；同步 restore driver 段的批准记录、专用执行身份、root 权限边界、进程树/no-follow/ENV allowlist 隔离、原生耐久回执来源与 POSIX/Windows 矩阵。实际产出者 Arch-Designer（Agent，非人的签字）；仍 draft、人的精确接纳 unknown，不声明实现、工具/备份/演练验证或生产可用。
263:| Q10 | Recovery Journal 持久卷、生产 CAS Anchor provider/config/wiring/独立故障域验证、目标 POSIX 文件系统证明、周期、未来 Windows 允许 SID 清单及历史非规范时间迁移方案是什么？ | §3.3.1、§3.4.2、§4.2、§6.3/8 | S4.2 只交付 Anchor port/协议/fake；生产 provider、配置、装配与独立故障域验证由 S4.5 承担；其余介质、周期、批准主体与历史迁移保持 unknown | 灾备演练前必填 |
1535:所有读取（含构造器）只打开已安装的 regular files/锁入口，验证 no-follow、owner-only、完整 retained keyring、绑定和支持矩阵，不 mkdir/create/chmod、bootstrap、fsync、CAS、清 pending 或修文件。采用独立 `OpenVerifiedRecoveryJournal(ctx,ReadOptions)` / article 对应只读入口，共用无副作用 parser；现有 NewFileRecoveryJournal/loadRecords 只能经受控 writer 路径调用。锁等待可取消，读取大小/记录数预算显式配置，超限返回 closed，禁止静默只读前 N 条。normal 遇 pending/孤立暂存/未锚定 tail 只报关闭；维护协调必须另持目的许可。
1685:共同 Genesis 是只读安装承诺验证器，不是 InstallManifest/签发秘密；Journal 的 contextual 方法由 S4.8 串行迁移，S4.7 只声明/调用中性端口，不能用 Background 兼容旧写方法。配置/工具/秘密/批准者 unknown 仍按对应真实动作拒绝，Normal/Verify 不因不适用 restore 字段缺失而失败。已构造对象的其他方法只消费本实例当前有效 grant/lease；没有执行入口完成核证，不得通过 CheckPhase/PhaseContext/绑定观察值自行获许可。
3654:Resume 不新 Begin，不盲 Replace：重读现有 Begin/完整 retained key、两 namespace/安装及同一 plan 的耐久 receipt/独立目标证明，按持久阶段续接。若替换已可证但 Store 绑定 ACK未知/缺失，只能在原设计已允许的精确回执路径用 recovery_bind 核同一绑定；无耐久可核来源仍 restore_unknown，不能靠 caller done 填 run。已有 running 直接按其真实绑定产 identity 许可；identity_ready 及后继仅进入原 Consume 允许的向前核证，succeeded 的恢复尾部只核关闭证明/退出，不再执行 Consume 或重置凭据。普通历史证明始终不放宽 failed/unfinished/缺行优先；受控当前 run 的预期中间态不是忽略其他历史/自动 supersede 的例外，更不扩大唯一 ready_to_consume＋exact consumed 窗口。
3828:Reader 由 identity consumer 定义、root 注入经批准来源的只读适配；缺依赖构造失败，不能用 body/备份行/刚写的 approval 自证。返回 Version 精确 `han-recovery-review/v1`，全部字段非空，role/status 枚举合法，ActorID 为本 run 已强改密初始 admin；EvidenceDigest 为可信来源不可变批准原件的 SHA256，不是 caller 自报摘要。来源记录须逐项绑定上述安装/run/target/actor/role/status/ref，适配器核既有来源授权后才返回；实际提供方、批准主体/介质/ACL仍unknown，不另建签名系统。读取在事务外，identity把已验证不可变记录绑定当前许可；事务内再次核 run/安装/target/actor/提议一致，保存 digest。结算按同一记录精确重读核验后再入事务，来源不明/更改/拒绝均零结算。只有完整批准记录可用的受控 fixture 可证明软件正向，不能当真实批准。
3858:`RestoreDriver` 是 root 中受信任备份工具的适配边界，不是 caller 自报恢复完成的 API；软件合同先固定，首期不提供生产 restore 工具与生产批准主体（ADR-0031），生产 enter/resume 的 DB 替换在 Begin 前以 closed 拒绝，仅保留软件合同与确定性测试替身。recovery 只声明 `Prepare(ctx,RunBinding) (RestorePlan,error)`、`Replace(ctx,PhasePermit,RestorePlan) (RestoreReceipt,error)`、`VerifyRestored(ctx,RestorePlan) (RestoreReceipt,error)`；具体实现放 cmd，不能把测试 driver 注册进生产。
3860:`RestorePlan` 固定 `Version='han-db-restore/v1',RunID,BackupRef,DeploymentID,InstallationID,TargetDigest,ArtifactDigest,SourceApprovalRef,SourceApprovalDigest,SourceInstallationID,SourceSchemaDigest string; ArtifactBytes int64; ArtifactPath string`。来自独立批准、只读挂载的 restore manifest；其批准摘要须由既有 CAS 信任边界的受控安装/备份来源记录核验，不能仅信文件内自带 hash。artifact 为 no-follow regular file，父目录和工具路径均受保护；Prepare 验 target 非秘密描述符摘要、精确 backup/安装/schema、全文件 SHA256/长度及实际 adapter 格式支持，不用 inventory 的 backup 名称替代实际制品证明。现有 inventory wire 不变，manifest 是恢复执行输入；没有批准来源不得临时填当前文件 hash 自证。
3862:生产 adapter 只允许执行配置中唯一批准程序：备份来源与可执行文件须由部署方具名恢复批准角色分别以批准记录登记，绑定 `SourceApprovalRef/Digest` 与二进制 SHA256、owner/ACL、受控只读挂载，批准者不得同时是执行者、本次 run 发起者或工具/备份供应方自批；执行主体是 root 启动的独立非交互子进程，须以独立于 root 的专用执行身份运行，root 不代替执行、不进入 shell、不重放，并保留部署锁、Journal/keyring、Store 绑定/审计写入与恢复 secret，不向子进程传递部署锁、Journal/keyring、审批 key、管理入口或除目标连接外的凭据。执行身份权限边界：只允许对目标 DB 执行 restore 与只读验证；目标连接凭据必须属于独立数据库角色，数据库角色与 GRANT 矩阵由批准记录固定、未证明即保持 closed；替换窗口之外，该角色权限不得覆盖 Journal/keyring、部署锁、Store 权威绑定与审计记录的任何写入；替换窗口之内，对目标库的写入只能来自已批准 artifact 的整库还原本身，执行身份不得自行构造、注入或声明任何权威 binding，也不得写 Journal/keyring 或部署锁，替换完成后由 root 以独立读取路径与同一 Store 短事务创建/核本次 running 与不可变 binding（同下文 Replace 合同），artifact 内既有历史行不得当作本次权威；受保护对象以批准记录固定的可检清单（Journal、keyring、部署锁、Store 权威绑定与审计记录）作为 restore 完成后的核对项，不改变 restore 的整库语义，也不引入 schema 级排除或 root 数据搬移。精确 argv 为 `<executable> restore-v1` 或只读 `<executable> verify-v1`，不用 shell、不允许 manifest 注入附加 argv；执行身份切换不得经 shell 脚本、su、sudo 等包装实现，具体机制由批准记录固定并在实现阶段以隔离实测证明。stdin 是私有管道的 strict v1 plan＋target connection 安全配置；不在 argv、stdout、日志或磁盘传 DSN/密码。传递环境采用最小 allowlist，零继承业务 secret/审批 key；执行身份除目标 DB 连接外无网络出口；子进程不自行启动 server/requeue，不在替换期间运行 migration/bootstrap/worker。程序二进制 SHA、owner/ACL、受控 manifest、执行身份与 fd/进程树隔离预检失败均在 Begin 前拒绝。POSIX 是唯一生产候选且首期仍 closed；Windows 生产 unsupported，须在任何状态变更前拒绝，仅允许软件合同测试。
3868:Replace/提交 ack 丢失、进程崩溃或取消均保持关闭，回读 Store/两 Journal/目标和 adapter 的耐久恢复回执；同一精确 binding/receipt 全部可证才 resume，不重新 Begin/恢复 secret 判重或再次 Replace。无持久可核来源（包括子程序完成但 Store 绑定前崩溃且无可核回执）即 `restore_unknown`，人工通过原受控备份工具核验/修复后再 verify/resume；不自动重放 DB 替换、不接受人工勾选 done。该保守暂停是明确恢复结果，不阻断其余软件设计；若要自动越过此未知窗口，须回架构定义新的证明，不能在 Task 增加 bool。任何外部工具调用均沿 io ctx、停止并真实 join 后才归还 target/pool/锁。
3870:制品校验与实际 restore 必须使用同一 no-follow 打开的稳定 fd/handle（需要 seek 时仍用该 handle）；核读前后 fstat/路径身份、完整字节数与摘要，禁止验完 hash 再按可替换路径重新打开。原生耐久回执只能由 restore 工具或其备份系统原生写入、独立于本次进程可重读的持久记录（恢复目录、恢复任务记录或安装来源证据）提供，并须绑定 artifact digest、target 与 source installation；root 以独立读取路径与批准记录逐项核对，不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证。工具/批准记录/执行身份/ACL/进程隔离的真实信任未证明时，这个软件协议仍不足以启用生产替换；当前无批准工具、批准记录、隔离与原生回执证据，生产替换保持 closed。
3957:Audit沿原ID owner一次分配，ctx供Trace；既有Metadata绑定installation/run/backup、原六字段固定hex/版本/generation、failed意图、原用途、来源Version/ref/digest/ApprovedAt及实际actor/提交时间。本增量cohort/resolution的generation及集合计数等int64在Metadata用规范十进制字符串存取并按int64严格解析，来源/提交时间用UTC RFC3339Nano微秒值；不经JSON float64丢失精度，不改其他既有审计元数据格式。完整结果由该不可变原件绑定并在回读时逐字段比较，不把raw Scope/原请求/快照全文/秘密复制进审计，也不能只写verified布尔。所需reader依赖在原executor私有装配首次使用前固定；S4.16后续接真实provider，无第二pool/Store/Runtime。未配置只令受影响resolve及有该未决集合的Complete闭，不凭空要求normal/verify配置未用的真实源，不以no-op或fixture作生产fallback；实际provider/批准主体/介质/ACL仍unknown，只阻对应真实层。
4203:r50 只新增恢复信任边界：受控 listener/root surface→阶段能力；restore 私有管道→受信任执行程序/目标DB；安装批准/CAS→本地安装行；article publication→identity精确消费。伪造surface/阶段能力在Session/写入前拒绝；manifest/回执须核独立来源、工具/目标及精确摘要，程序exit0不等于恢复证明；argv不含secret、禁shell拼接、stdout严格有界，stderr不转发原文；元数据不可变/原子约束与双namespace revision检测回退/篡改，unknown默认关闭；外部I/O超时只触发取消/真实join，不把未结束副作用藏后台。公共probe/错误不泄run/账号/secret，维护动作仍initial-admin/CSRF/审计，普通Principal不获能力。原Argon2id、HMAC/retained keys、凭据/Session、最小PII、审计不可变与安全红线合同精确复用；实际ACL/可信工具/批准主体未证不启真实层，不新增签名服务或固定秘密。
```

### 7.2 framework 关键字检索

命令：

```text
rg -n "restore|恢复工具|批准|进程隔离|no-follow" .add/project/knowledge/decisions/framework.md
```

退出码：`0`；命中 **72** 行（review-r01 时为 70 行，新增命中来自 `RV2-01` 改写后的 ADR-0031 行）。

输出：

```text
228:- 2026-10-20 架构修订 `architecture-ir01-r04` 登记已批准依赖采用：为 ADR-0012 的 Argon2id 引入 `golang.org/x/crypto`，不改变 Go 1.27 结论。来源：当前用户明确批准。
236:**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 明确批准公共错误合同
263:- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束全部 REST 接口、登录统一失败和测试合同。来源：当前用户明确批准。
271:**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 明确批准 `Idempotency-Key` 与目标状态幂等方案
287:批准方案直接覆盖 SMTP 和网络重试风险。幂等记录保留时长保持 TBD，以业务重试窗口、容量和隐私评估确定。
300:- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束身份治理写接口、SMTP 残余窗口与数据模型。来源：当前用户明确批准。
308:**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 明确批准 PostgreSQL 追加式审计方案
336:- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束跨模块审计入口、数据模型、运行时序与测试。来源：当前用户明确批准。
344:**状态**: 已接受 | **确认来源**: `requirements-ir01-r16` checkpoint `decision-e3af3912166f7287e8324488cad1cf1b87ba1034fca3378d3e0467a23751592d` 及当前用户批准的 IA-DES-R01-F001 整改方向
348:**决策**: 注册前展示并记录版本化隐私声明确认；初始管理员创建目标管理员时记录信息提供授权声明；新管理员必须同时完成改密与当前隐私确认后才能使用其他管理员能力。邮件仅限已确认事务性用途。部署前必须登记适用地区、数据控制者、处理依据和获批准 SMTP 处理方，任一缺失则相关能力 fail closed。本决定不声称自动法律合规。
364:- 2026-10-20 架构修订 `architecture-ir01-r05` 采用本决定。来源：已接受 requirements r16 与用户批准整改。
372:**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 批准灾备恢复合同并要求关闭 IA-DES-R01-F003；用户于 2026-10-22 明确确认 identity-owned RecoveryRunStore、恢复状态机与崩溃收敛修订
410:- 2026-10-20 架构修订 `architecture-ir01-r05` 采用本决定，约束恢复时序、数据状态和运行手册。来源：当前用户批准整改。
411:- 2026-10-20 架构修订 `architecture-ir01-r06` 采用本次修订，补齐永久密码、角色、删除事实与账号恢复 Gate。来源：当前用户批准的灾备恢复合同。
412:- 2026-10-20 架构修订 `architecture-ir01-r07` 采用本次修订，补齐 `kept_deleted` canonical 删除与 Recovery Journal 一致性。来源：当前用户批准恢复日志合同。
422:**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 批准恢复日志合同及跨 HMAC key 版本复用检测整改；用户于 2026-10-22 明确确认独立故障域 CAS Journal 头锚点、Windows 生产禁用/DACL fail closed、UTC 微秒与 OS 文件锁修订；用户于 2026-10-22 再次明确确认不可变 `backup_ref` 三方绑定与 Begin-key 固定终结/退役约束；用户于 2026-10-22 明确确认 IA-S4.2-DR-F005 采用方案 A，将 S4.2 限定为 provider-neutral Anchor port/协议与确定性测试替身；用户于 2026-10-22 明确确认 IA-S4.2-CR-002 采用“方案 A-补充”，将 S4.2 key retirement 限定为 opaque capability 验证边界
428:**决策**: Recovery Journal 每条记录保存 `key_id/key_version`、HMAC 指纹、run ID、不可变 `backup_ref`、阶段和 consumed，并由位于独立故障域、支持 compare-and-swap 的 `JournalHeadAnchor` 绑定已提交 `journal_id/namespace/epoch/head_sequence/head_hash`。`Begin(runID,backupRef,recoverySecret)` 把 run、backup 与当时 active key 的 ID/version/fingerprint 一次绑定；begun 与唯一 consumed 终结记录必须沿用该绑定。keyring 包含一个 active 新 run 写入 key与仍处保留期的 verify-only 历史 key；verify-only 禁止开启新 run，但允许为由该 key 已 Begin 且未消费的 run 写唯一 consumed 终结记录。`Begin` 对候选 secret 使用全部 retained key 计算候选指纹，并只与相同 key ID/version 的历史记录比较；任一命中即拒绝。key 退役/删除除满足“无 begun 未消费 run、记录已达到批准保留/归档退出条件”外，还必须提交 opaque `KeyRetirementCapability` 并通过验证/消费 port；S4.2 不解析、不签发、不配置该 capability，只定义调用边界和 fail-closed 结果，生产不存在 mint 路径且默认持续禁用 key retirement。RecoveryRunStore、Recovery Journal 与 Article evidence 的 `run_id + backup_ref` 必须精确一致，验证必须分别读取 Store 与 Journal 的独立事实，错误 backup 或任一来源不一致均 fail closed。生产 `FileRecoveryJournal` 暂不支持 Windows；支持平台使用进程退出自动释放的 OS 独占文件锁，所有持久时间统一为 UTC 微秒。S4.2 的十三文件边界只交付 provider-neutral `JournalHeadAnchor` port/协议、调用边界和确定性测试替身，以及 `KeyRetirementCapability` 验证协议/能力边界；不选择或交付生产 CAS provider、retirement evidence 来源、capability 签发、配置及启动装配。生产 Anchor provider 与真实独立故障域验证仍由 S4.5 负责；retention/archive exit evidence、capability 签发及 key retirement 生产闭环由 Task-Planner 另行登记一个不并入 S4.5 的独立后续任务。对应任务完成前不得宣称恢复链或生产 key retirement 就绪。key、锚点、权限、capability 或 Journal/DB 不可证明一致时均保持停服。
441:本轮交付边界采用方案 A：先固化 provider-neutral port、协议和确定性测试替身，再由独立任务选择并接入生产 provider。另一条路是在 S4.2 十三文件内同时选型和装配生产 provider，但会要求第十四文件、配置/启动接线或新增依赖，并把未经批准的技术栈写进当前任务，因此不采用。
458:- 只要某 key 尚有 begun 未消费 run，或该 key 对应的任一 Journal 记录仍处已批准保留/归档可验证范围，该 key 就不得退役或删除。即使静态条件看似满足，也只有 opaque `KeyRetirementCapability` 经验证/消费 port 对精确 `key_id/key_version` 的本次 retire/delete 请求返回成功后才能销毁；capability 缺失、空值、拒绝、不可用、绑定不匹配或结果不可判定均 fail closed。
466:- Journal 文件位置由受控配置提供，不硬编码 OS 路径；目录、Journal、临时文件与锁入口必须证明 owner-only 写边界。生产 Anchor provider 的选择、配置键和启动装配不属于 S4.2，由后续独立任务定义。无法读取/解释权限或发现未批准写主体时 fail closed。
472:- Windows 生产初始化必须在任何 Journal、anchor、临时文件或锁状态变更前返回 unsupported。未来 Windows provider 必须证明同卷原子替换、持久化和 DACL owner-only 边界；安全描述符不可读/不可解释或存在未批准写 SID 时 fail closed。具体允许 SID 清单属于部署/安全输入，当前为 unknown。
481:- 2026-10-20 架构修订 `architecture-ir01-r07` 采用本决定，约束恢复部署、运行时序、故障处理与测试。来源：当前用户批准恢复日志合同。
482:- 2026-10-20 架构修订 `architecture-ir01-r08` 采用本次修订，补齐跨 key-version secret 复用检测与 key 退役条件。来源：当前用户批准第四轮复审整改。
483:- 2026-10-20 架构修订 `architecture-ir01-r09` 最小复用本决定的持久卷/keyring 基础设施，同时固定 Article Deletion Journal namespace 隔离。来源：当前用户批准文章删除日志方案。
501:**关系**: 细化 ADR-0003 的 PostgreSQL 版本与接入方式，并按 ADR-0009 完成 `github.com/jackc/pgx/v5` 第三方依赖批准；Go 1.27 与既有 `golang.org/x/crypto` 批准不变。
553:- 应用使用 `github.com/jackc/pgx/v5` 原生 API；连接池采用 `github.com/jackc/pgx/v5/pgxpool`。本决定只新增批准 `pgx/v5`，不改变既有 `golang.org/x/crypto` 批准。
592:现有代码已经提供 provider-neutral port 和有界行为：`platform.LoggerWithTrace`、`LoginOutcomeObserver` / `LoginMetrics`、`OriginRejectionObserver` / `OriginSecurityMetrics`、`PostgresLoginFailureAuditor`；构造器拒绝 nil，handler/guard 通过 `context.WithTimeout` 传递取消和 deadline。`go.mod` 的运行依赖仅为已批准的 pgx/v5 与 x/crypto。本决定必须复用这些边界，不因运维 backend 把领域层绑定到新 SDK，也不能把测试 fake/no-op 当作生产实现。
764:**状态**: 已接受（技术决定已采用；Kuhn独立R02已关闭F001／新增0，仅文档；成果仍draft／未实现／人的接纳unknown） | **确认来源**: 用户再次明确“待确认/决策按建议值继续”；Main正式指定三文档写权与框架/identity/article/platform/cmd归属，并接受Kuhn独立R01的 `ARCH-R32-R01-F001`（报告SHA256 `81BFE22BC1481630602176CDAD45AA19971BB7B3B328188CB91680DDDFDF0B53`）要求原作者定点修复，不风险豁免。该来源不是全体人工签字、成果checkpoint、实现或运行批准，实际成果接收人未知不补签。r01/Review R01及原待复审记录保留；本次仅据[Kuhn独立R02原报告](../../../../.spec/wip/recovery-runtime/evidence-architecture-r32-r01/review-r02/review-report.md)同步文档状态，Main已核收报告/receipt，不泛化为代码或环境PASS。
799:10. 可独立交付/验收closed请求与worker屏障、只读失败路径；Windows真实DB＋受控adapter只证软件事务/顺序/root503/历史重启/故障及归还，不作生产fallback。S4.5真实CAS硬前置与S4.6真实retirement闭环保持；无verifier持续禁退役。真实POSIX/独立故障域/备份inventory批准/ACL/epoch/RPO-RTO/人的接纳仍分别NOT RUN/TBD，不由ADR-0024/0025或maintenance拒启扩大豁免。article S1.5/S4.4/S8.2及Case18原断言/整卡门禁不关闭。
802:13. 首装精确空链及存量链各走显式可信一次性绑定：停旧写者/排空，存量须核恢复域外独立完整链检查点/批准副本来源和期望head、全链/keyring、部署/目录及namespace/id/epoch，再以Anchor不存在为前置CAS发布初始承诺/迁移来源。normal首读不能认可任意现存合法前缀；可信来源/实际主体unknown即不迁移，不重置既有Anchor。中断仅受控续接同一迁移，不改原记录/S16字节，新旧未锚定写者不混跑；article适配/迁移归全局runtime新软件范围，S4.5原真实前置、S4.2/S10.5关闭及测试卡边界保持。具体局部类型/ENV/argv由FD细化，生产者、失败与迁移语义不可下放取舍。
807:- 2026-10-29 [identity-auth-design-ir01-r50](../design-workspace/identity-auth/design.md) 实际采用ADR-0027 r02，最终稿SHA256 `6634BED32EDE264F6DA6F47C9A7E8152ADD0B19C2DDCBCF3ED4708AE4BA4109D`：在原S04恢复共享基础及§3.3.1/4.2形成唯一Store/identity consumer、正向准入/历史关闭证明、阶段内部许可、root恢复驱动与生命周期接线草案。来源：Main按用户standing采用G0建议并正式授予四文件WRITE START，FD Banach完成后STOP/frozen，Main已全文核增量并匹配Hash。Design R01两HIGH经原作者Banach修复，Kuhn[Design R02](../../../../.spec/wip/recovery-runtime/evidence-design-r01/review-r02/review-report.md)已在设计层关闭RECOVERY-DES-R01-F001/F002、NEW=0，本轮Main交接确认已全文实核并技术接收。R02仅覆盖报告列明的技术受审SHA；本最终稿另含随后纯元数据同步，不预签最终六对象元数据核验。仍draft、人的接纳unknown，实现/测试未验，不表示Code/Case/production PASS；对应Story r17仅由该design派生同步，不另计采用。 同源版本追溯（2026-10-29）：上述r50稿及其“最终稿”SHA/Review均为当时历史，原记录保留；同一identity design采用源随后由Banach / add-feature-designer `01a18389-57ab-7b70-8c86-82a61e7729f1`细化为r51 SHA256 `F87E1EB9073F4B2A723A098776762F759332B8086384BC0ED3C79DC49F624564`及r52 SHA256 `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`。[r51完整增量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r51-author-r01/design-r50-to-r51.diff)细化Enter/Resume的前置目的/独立当前绑定、动作依赖、窄RecoveryConsumer、root RecoveryExit与同identity executor撤维护Session；[r52完整增量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r52-author-r02/design-r51-to-r52.diff)补Journal-owned VerifyBeginSecret及Resume写前核验，S4.7声明/调用与fixture、S4.8持钥只读实现、S4.16惰性候选读取，既有authority/业务状态/S16不变。[r51设计R01](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-review-r01/review-report.md)唯一HIGH RECOVERY-DES-R51-R01-F001经原作者修复，[Kuhn设计R02](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-review-r02/review-report.md)（报告SHA256 `21408124976E38F96C07DFF4444CA52602EBB7AC93740E9D6905A20EE66C26FF`）仅在设计合同修复及相关新风险范围CLOSED/新增0，r51未变化已审范围沿用；[Main实际接收记录](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/f001-source-local-start-r52.json)明确技术接收r52，不是人的签字或代码关闭。仍draft/人的接纳unknown；Main本轮交接的代码四finding仍OPEN、仅组件fixture开发中，不据此登记实现/PG/restore/E2E证据或开流批准。本条仍同一采用源1项、总数6，不增ADR/采用源，不循环刷新Task/Plan/TP或历史ledgerhash。
920:**状态**: 已接受（2026-11-01 用户 D02 指令授权 Arch-Designer 论证并落定“首期不提供生产 provider”；确认来源为本轮用户任务指令；不构成人的成果接收、独立评审、生产装配批准或真实 provider 验证） | **确认来源**: 用户 2026-11-01 在 B13J D02 中明确要求由 Arch-Designer 论证并落定生产 CAS Anchor provider 产品选型，且明确“首期不提供”是合法且优先考虑的结论之一。该来源授权的是架构决策与文档写入，不是生产装配、真实独立故障域验证或人的成果签收。
941:选择 D。A、B、C 都需要当前不存在的真实产品、凭据、部署/ACL 和独立故障域验证；在缺少这些证据时选择任一产品，等于把“未验证”写成“已决定可部署”，并把后续实施锁进一个未经批准的公共依赖。D 不新增任何外部依赖，不触发 file/SQL/fake fallback，且明确保留后续通过新决策解除 closed 的路径，是当前唯一可审计且不伪造生产就绪的结论。
955:   - 凭据来源、最小权限/ACL、部署拓扑、备份/恢复/轮换与运维责任经批准且可核；
959:8. **不适用场景**：本 ADR 不适用于 S4.2 的 port 与测试替身交付（已由 ADR-0019 定）、key retirement capability（ADR-0019 另定）、restore 工具/批准者/宿主隔离、部署独占、ArticleAccountPort 或 recovery secret candidate；这些分别由其他设计/决策承接。Windows 生产仍 unsupported，本 ADR 不提供 Windows 例外。
966:- 决策授权：用户 2026-11-01 B13J D02 指令；产出者为 Arch-Designer（Agent），不是人的签字、独立评审或生产批准。
971:## ADR-0031: 首期不提供生产 restore 工具与批准主体，固定受批准执行与宿主隔离合同 (2026-11-01)
975:**状态**: 已接受（2026-11-01 用户 D03 指令授权 Arch-Designer 论证并落定“首期不提供生产 restore 工具与生产批准主体”；确认来源为本轮用户任务指令；不构成人的成果接收、独立评审、安全评审、生产装配批准或真实工具/备份验证） | **确认来源**: 用户 2026-11-01 在 B13J D03 中明确要求把受批准 restore 工具、备份批准主体、执行主体、宿主隔离与原生耐久回执落成可审计、fail-closed 的架构决策，并明确“首期不提供”是合法结论之一。该来源授权的是架构决策与文档写入，不是生产装配、真实旧备份演练、安全评审或人的成果签收。
977:**继承关系**: 只固定 `identity-auth/design.md` §4.2 中“受批准 restore 工具 / 备份批准主体 / 执行主体 / 宿主隔离 / 原生耐久回执”这组组织与信任边界；不改变 ADR-0019 已锁定的独立故障域 CAS 头锚点、keyring、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒与 fail-closed 失败分类；不改 ADR-0030 的“首期不提供生产 CAS Anchor provider、S4.5 未交付、生产 enter/install 保持 closed”；不改变 ADR-0027 的恢复 runtime 编排与正向准入证明；不重写或削弱 design 既有 restore 软件合同（`restore-v1`/`verify-v1` 的 argv 与私有 stdin、no-follow 稳定 fd、最小 ENV allowlist、receipt 字段、`restore_unknown` 保守暂停、Begin 前预检失败即拒绝），只补其批准、执行身份、宿主隔离与原生回执这组前置前提；若后续发现软件合同本身有缺口，另立决策或设计修订处理，不在本 ADR 内改写。
979:**问题**: 设计已固定 restore 的软件协议与 Begin 前预检，但没有可核的批准主体、执行主体、执行身份与进程隔离证明，也没有工具原生耐久回执的真实样本：`RestorePlan.SourceApprovalRef/Digest` 由谁签发、`HAN_RECOVERY_RESTORE_EXECUTABLE` 的二进制 SHA/owner/ACL 由谁批准、由哪个进程以什么身份执行替换、root 与执行者之间的权限边界、宿主隔离的最小必要机制、以及“该 artifact 已还原”由谁以什么形态提供持久证据，改动前均为 unknown。生产 root 因此无法构造可验收的 `RestoreDriver`，生产 `enter`/`resume` 的 DB 替换只能在 Begin 前拒绝。需要在不虚构未验证工具、批准角色或演练结果的前提下，给出唯一可验收口径。
981:**决策**: 首期不提供生产 restore 工具与生产批准主体。生产 `enter`/`resume` 的 DB 替换路径继续保持 closed；与 ADR-0030 的合并口径是“生产 `enter`/`install` 与 `enter`/`resume` 的 DB 替换路径均保持 closed”，二者是不同入口而非范围收窄。仅保留既有软件合同、确定性测试替身与 `restore_unknown` 语义。`HAN_RECOVERY_RESTORE_EXECUTABLE`、`HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 与 `HAN_RECOVERY_RESTORE_MANIFEST_PATH` 仍是唯一入口且无默认值、未知不补值；`verify` 与 `genesis` 形态不执行工具。任何未来生产形态必须同时满足下节全部批准、执行身份、宿主隔离与原生回执不变量，否则不得解除 closed。
985:现状是“协议已定、执行主体未定”：设计已把 `restore-v1`/`verify-v1` 的 argv、私有 stdin、receipt 字段、no-follow 稳定 fd、最小 ENV、Begin 前预检与 `restore_unknown` 写成软件合同；但没有任何具名批准角色、已批准二进制 SHA、专用执行身份、权限矩阵、进程隔离实测或工具原生耐久回执证据。当前不存在可执行的生产 restore 工具，也不存在可核的备份来源批准记录。
987:痛点在于：若现在声明“由部署方批准、由 root 子进程执行”，等于把一个不存在的批准角色、未指定的执行身份和未验证的隔离机制写成公共合同；若继续维持“可配置但未定义”的模糊状态，实施者容易用 shell 包装、附加 argv、环境探测、caller 自报 done、DB 可连接或自己刚写的 run 行填补空缺。另一类风险是把 `restore_unknown` 当缺陷而用自动重放“修复”。
991:| A. 首期不提供生产 restore 工具与批准主体（选定） | 不注册生产 restore 可执行文件，不任命生产批准角色；生产 `enter`/`resume` 的 DB 替换保持 closed；仅保留软件合同、确定性测试替身与 `restore_unknown` | 不引入未验证工具、批准角色或隔离声明；与 ADR-0019 的 fail-closed 及“测试替身不得作为生产证据”一致；解除路径唯一且可审计 | 首期生产恢复链不可用；旧备份只能离线处理；后续必须重新选型、验证并修订 ADR 才能解除 closed | 选定 |
992:| B. 部署方受控只读挂载已批准二进制 + 具名部署批准角色 + root 子进程执行 | 部署方把已批准二进制放入受控只读挂载，以批准记录绑定 SHA256/owner/ACL；具名恢复批准角色批准备份来源；root 以固定 argv 启动子进程并保留锁与验证 | 与既有软件合同最接近；工具可独立升级与替换；root 仍持部署锁、Journal/keyring 与最终验证 | 当前无具名批准角色、无已批准二进制、无专用执行身份与权限矩阵、无进程组/no-follow/ENV allowlist 实测、无工具原生回执；工具供应链、更新责任与真实旧备份演练均未定 | 不选 |
993:| C. 独立恢复运维单元执行替换，本服务只做 VerifyRestored 与绑定 | 外部团队或系统在其环境执行 restore，本服务只验证恢复结果并写绑定 | 职责分离强；本服务不承载替换执行面 | 关键替换动作移出本服务审计与进程隔离边界；如何证明执行的是同一 artifact/target、如何取得原生回执、如何阻止并发写与重放、责任与时限均未定；当前无该单元 | 不选 |
996:选择 A。B、C、D 都需要当前不存在的具名批准角色、已批准二进制或平台实例、专用执行身份、隔离实测与原生回执证据；在缺少这些证据时选择任一形态，等于把“未验证”写成“已批准可执行”。A 不新增外部依赖，不触发 shell/附加 argv/自动重放/自动 fallback，保留既有软件合同与 `restore_unknown`，并把解除条件固定为可核清单，是当前唯一可审计且不伪造生产就绪的结论。
998:改造是把“谁批准、谁执行、用什么隔离、回执从哪来”从实现细节提升为生产构造门禁：未来任何生产 restore 形态只有同时具备具名批准记录、已批准二进制、专用执行身份、隔离实测与工具原生存久回执，才可能通过 Begin 前预检；任一缺失一律 closed，且不得用 caller 自报、exit=0、stdout receipt 自洽、DB 可连接或本次 run 行替代。
1000:效果是：生产 root 不会因环境变量存在、文件可执行或 manifest 可读而误开替换；测试替身继续只服务协议测试；`restore_unknown` 保持为明确恢复结果而非缺陷；解除 closed 需要一次性补齐全部证据并经独立评审与正式登记，避免实现阶段自行补默认值或引入未批准工具。
1002:### 批准、执行与隔离不变量
1004:1. **备份来源批准**：`RestorePlan.SourceApprovalRef/SourceApprovalDigest` 只能由部署方具名“恢复批准角色”签发，批准记录须写明角色、人员、时间、备份来源、artifact digest 与 source installation；该角色不得是执行者、本次 run 发起者或工具供应方自批。当前项目未任命该角色，故不存在可核批准记录。
1005:2. **可执行文件批准**：`HAN_RECOVERY_RESTORE_EXECUTABLE` 指向的二进制须由同一具名批准角色以独立批准记录登记绝对路径、SHA256、owner/ACL 与受控只读挂载来源，并与 `HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 精确相等；不存在“默认部署方角色”“按目录推断”或“运行时自行选最新版本”。当前无已批准二进制。
1006:3. **执行主体**：替换只能由 root 启动的独立非交互子进程执行，且该子进程须以独立于 root 的专用执行身份运行；root 不代替执行、不进入 shell、不重放，也不把部署锁、Journal/keyring、审批 key、管理入口或除目标连接外的凭据传给子进程。执行身份切换与程序启动禁止经 shell 脚本、`su`、`sudo` 等包装实现（与固定 argv、无 shell 冲突）；具体切换机制由批准记录固定，并要求在实现阶段以隔离实测证明，本 ADR 不选具体机制。
1007:4. **root 与执行身份的权限边界**：root 保留部署锁、排空/独占、Begin/绑定/审计与最终 `VerifyRestored` 判定；执行身份只获得只读 artifact fd、经私有 stdin 传入的目标连接配置、最小 ENV allowlist 与目标 DB 网络。替换窗口外该身份不得写 Journal/keyring/锁/Store 权威绑定或审计；替换窗口内对目标库的写入只能来自已批准整库 artifact 的还原本身，该身份不得自行构造、注入或声明权威 binding，也不得写 Journal/keyring 或部署锁，本次权威 binding 只由 root 经同一 Store 短事务创建/核（design `:3866`）。该身份不得启动 server、requeue、migration、bootstrap 或 worker。
1012:| 启动方式 | 固定 argv 直接 exec 唯一批准程序；无 shell、无附加 argv | 同一软件合同，但仅允许测试替身与软件试验 |
1014:| 文件 | no-follow 稳定 fd/handle；校验前后 fstat 与路径身份一致；禁止验后按路径重开 | 同一软件合同，仅测试 |
1018:6. **原生耐久回执**：只能由 restore 工具或其备份系统原生写入、独立于本次进程可重读的持久记录（恢复目录、恢复任务记录或安装来源证据）提供，且须绑定 artifact digest、target 与 source installation；root 必须经独立读取路径与批准记录逐项核对，不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证。具体产品字段与读取方式属工具/部署输入，当前 unknown。
1020:上述六项在当前均为缺失状态，故生产 restore closed；这不削弱既有软件合同与测试替身，也不改变 `restore_unknown` 的保守暂停语义。
1024:1. **不改变既有决定与合同**：ADR-0019、ADR-0030、ADR-0027 与 design 既有 restore 软件合同继续有效；本 ADR 只补组织与信任边界。
1027:4. **批准与执行分离**：批准角色必须具名且不得自批、不得兼任执行者；执行身份不得等于 root，也不得与批准角色共用凭据。
1028:5. **`restore_unknown` 保留**：无可核持久回执时保持 `restore_unknown`，由人工经原受控备份工具核验/修复后 verify/resume；不得用自动重放或人工勾选 done 越过。
1029:6. **不声称验证**：本 ADR 不证明任何 restore 工具、备份系统、托管平台、执行身份、ACL、进程隔离或旧备份演练已通过；不声称生产装配、Case18、S8.2 或恢复 E2E 通过。
1032:   - 部署方任命具名恢复批准角色，并留存备份来源与可执行文件两条独立批准记录（含角色、人员、时间、依据）；
1033:   - 已批准二进制具备可核 SHA256、owner/ACL、受控只读挂载与更新/回滚责任；
1034:   - 专用执行身份与权限矩阵经批准：替换窗口外不得写 Journal/keyring/锁/Store 权威绑定与审计；替换窗口内写入只能来自已批准整库 artifact 的还原本身，本次权威 binding 只由 root 写；不得启动 server/requeue/migration/worker；
1035:   - POSIX 宿主隔离实测通过：固定 argv 无 shell、独立进程组与完整子进程树 join/reap、no-follow 稳定 fd、最小 ENV allowlist、目标连接外无网络；
1036:   - 负向验证门禁通过：替换窗口外，以执行身份的凭据或其目标 DB 角色写入 Journal/keyring/部署锁/Store 权威绑定/审计必须被拒绝；替换窗口内，除已批准整库 artifact 的还原本身外不得产生任何 Store 权威绑定或 Journal/keyring/部署锁写入，执行身份自行构造、注入或声明的 binding 必须被拒绝；
1040:   - 取得人的明确生产批准；Agent 产出、评审结论或测试通过均不替代该批准。
1041:9. **不适用场景**：本 ADR 不适用于 ADR-0019/ADR-0030 的 Anchor/keyring 与生产 CAS provider、ADR-0027 的 runtime 编排与正向准入、design §4.2 的 restore 软件协议本身，以及 D04 承接的部署独占、`ArticleAccountPort` 与 recovery secret candidate。Windows 生产仍 unsupported，本 ADR 不提供 Windows 例外。
1045:- 2026-11-01 `identity-auth` 设计修订 `identity-auth-design-ir01-r59`（含 D03 评审返工）采用本决定：restore driver 段 `.add/project/knowledge/design-workspace/identity-auth/design.md:3858` 固定“首期不提供生产 restore 工具与批准主体、生产 enter/resume 的 DB 替换在 Begin 前 closed、仅保留软件合同与测试替身”；`:3862` 固定批准记录（含工具/备份供应方不得自批）、专用执行身份、root 保留部署锁/Journal/keyring/Store 绑定与审计写入的权限边界、进程树/ENV allowlist 与 fd 隔离预检（no-follow 稳定 fd 见 `:3870`）及 POSIX/Windows 矩阵，并固定执行身份的两层写入边界——替换窗口外其角色权限不得覆盖 Journal/keyring/部署锁/Store 权威绑定与审计的任何写入，替换窗口内写入只能来自已批准 artifact 的整库还原、不得自行构造或声明权威 binding、替换后由 root 经同一 Store 短事务创建/核本次 binding，受保护对象清单只作 restore 完成后的核对项、不改变整库语义且无 schema 级排除或 root 数据搬移；其目标连接须为独立数据库角色且角色与 GRANT 矩阵由批准记录固定、未证明即 closed，以及除目标连接外无网络与禁止 su/sudo/shell 脚本包装；`:3870` 固定原生耐久回执的来源形态与“信任未证明不得启用生产替换”；配置表 `:3896` 的三个 restore 键语义不变；决策追溯 `:83` 登记本 ADR。设计仍为 draft，人的精确接纳 unknown，不声明实现或验证。
1048:- 缺口来源：[gap-map.md](../../../../.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md) 的 §2.5 `pending-B13J-D03`（:121-139）与 U4；该文件要求工具、批准者、执行主体、ACL/进程隔离、原生回执与 unknown 暂停均具体。
1049:- 决策授权：用户 2026-11-01 B13J D03 指令；产出者为 Arch-Designer（Agent），不是人的签字、独立评审、安全评审或生产批准。
```

### 7.3 whitespace 检查

命令：

```text
git diff --check -- .add/project/knowledge/design-workspace/identity-auth/design.md .add/project/knowledge/decisions/framework.md
```

退出码：`0`

输出：

```text
warning: in the working copy of '.add/project/knowledge/design-workspace/identity-auth/design.md', LF will be replaced by CRLF the next time Git touches it
```

说明：只有 Git `autocrlf` 的 advisory warning，没有 whitespace error；`design.md` 与 `README.md` 的 LF 风格是既有状态，不属于本轮引入。

## 8. 继承与边界

- 不改变 **ADR-0019**：`JournalHeadAnchor` 独立故障域、keyring、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒、fail-closed 分类均保持原样；本轮未改 `framework.md` 前 966 行任何字节；review-r01 返工只动 ADR-0031 范围内的 `:981`、`:1006`、`:1045` 三行并新增 `:1036` 一行，review-r02 返工只动 `:1007`、`:1034`、`:1036`、`:1045` 四行（无增删行），前 137491 字节（原 966 行，含末尾 CRLF）始终与 D03 初始基线逐字节相同。
- 不改变 **ADR-0030**：首期不提供生产 CAS Anchor provider、S4.5 维持未交付、生产 `enter`/`install` 保持 closed 的结论原样保留；本 ADR 只处理 restore 的执行与信任边界。
- 不改变 **ADR-0027** 的 runtime 编排、唯一 Store 与正向准入证明；本 ADR 不新增 runtime 分支、不放宽准入。
- 不重写或削弱 design 既有 restore 软件合同：`restore-v1`/`verify-v1` argv、私有 stdin、no-follow 稳定 fd、最小 ENV allowlist、receipt 字段、`restore_unknown`、Begin 前预检失败即拒绝均原样保留；只在 `:3858`、`:3862`、`:3870` 补组织与信任前提。
- 未改 `:3874-3902` 命令/ENV 表；`HAN_RECOVERY_RESTORE_EXECUTABLE`、`HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256`、`HAN_RECOVERY_RESTORE_MANIFEST_PATH` 仍无默认值、未知不补值。
- 未写任何未登记 Task 编号（无 S4.22/S4.23/S4.24 等）；未来实现与真实演练须由正式 Task 规划登记。

## 9. 遗留风险与本轮未覆盖项

1. **无真实证据**：不存在已批准的 restore 工具、已批准二进制、具名批准记录、专用执行身份、ACL/进程隔离实测或工具原生耐久回执样本；本轮没有运行任何 restore、备份或演练。
2. **执行身份未落地**：ADR-0031 要求“独立于 root 的专用执行身份”，但具体 POSIX 用户/组、capability、Windows 等价机制均属未来实现与安全评审输入，当前 unknown。
3. **批准角色未任命**：`SourceApprovalRef/Digest` 与可执行文件批准记录没有真实承载主体；“批准者不得兼任执行者/发起者”目前只是合同约束。
4. **口径已合并（RV-03 处置）**：ADR-0031 `:981` 已写明与 ADR-0030 的合并口径：生产 `enter`/`install`（anchor 未交付）与 `enter`/`resume` 的 DB 替换路径均保持 closed，二者是不同入口而非范围收窄。`design.md:3858` 仍用“生产 enter/resume 的 DB 替换在 Begin 前 closed”，是为保留 `genesis`/`install`/`verify` 的既有软件语义；与 ADR-0030 叠加后所有入口仍闭，不构成范围收窄。
5. **软件合同缺口未处理**：若既有 `restore-v1`/`verify-v1` 合同本身有缺口，本轮按任务要求只在报告提出，不在 ADR 或 design 中改写。
6. **未验证声明**：不声称任何工具、备份、托管平台、执行身份、ACL、进程隔离或旧备份演练已通过；不声称生产装配、Case18、S8.2 或恢复 E2E 通过；`blocked-upstream` 未解除。
7. **设计状态**：`identity-auth` 设计仍为 draft，人的精确接纳 unknown；本 ADR 的“已接受”只表示用户 D03 指令授权的架构决策，不是人的成果签收或生产批准。
8. **Windows**：生产仍 unsupported，仅允许测试替身与软件试验；本轮没有 Windows 隔离实测。
9. **无代码/测试变更**：未修改 Go 代码、迁移、部署文件、测试文件、`tasks.md`、`plans/`、`stories.md`、`test-plan.md` 或 `decisions/identity.md`；未执行 Git 写操作。
10. **索引一致性**：`README.md` 三处已补 ADR-0031；未做全库 ADR 引用审计。
11. **权限矩阵未实测（RV-01 / RV2-01 剩余）**：ADR 与设计已把执行身份的写入边界固定为两层可满足形式（替换窗口外角色权限不覆盖受保护对象的任何写入；替换窗口内写入只能来自已批准整库 artifact 的还原本身，本次权威 binding 只由 root 经同一 Store 短事务创建/核），`framework.md:1036` 的负向验证门禁亦据此改写；但角色名、具体 GRANT 语句、“窗口外写入被拒绝”与“窗口内不产生额外权威 binding”的实测均未在真实环境完成，属实现阶段与安全评审输入，当前 unknown。
12. **身份切换机制未选（RV-05 剩余）**：RV-05 已禁止经 shell 脚本、`su`、`sudo` 等包装实现身份切换，并要求实现阶段以隔离实测证明；具体 POSIX 机制（如 `fork` + `setgroups/setgid/setuid` + `exec` 或经批准的单一 launcher）未在本轮选定。
13. **整库语义专项风险（RV2-01 衍生）**：若未来实现把“受保护对象可检清单”当作 schema 级排除或不还原范围的依据，就会与 `:3864`（只传现役已校验 DSN）、`:3866`（同一 artifact/target 出 receipt）与 `:3870`（同一 no-follow 稳定 fd）冲突；本轮已在 ADR 与设计中明确该清单只作 restore 完成后的核对项、不改变整库语义，实现阶段须以 GRANT 与隔离实测证明不存在隐藏例外路径。

## 10. 建议另行处理（不在本轮改动）

- `design.md:3828`（reader/approval 来源）与 `:3957`（S4.16 provider 层）仍写“实际提供方/批准主体/介质/ACL 仍 unknown”。它们属于 S4.16 的真实 provider 承接面，不在 D03 的 restore 工具/宿主隔离范围；建议由对应任务的设计修订处理。
- `design.md:263` Q10 中“批准主体”仍指 Recovery Journal 持久卷/CAS 介质，属 ADR-0019/ADR-0030 范围，本轮不扩大解释。
- `tasks.md` 的 S4.5 真实登记行为 `:656`（Lead 实测）；本轮未改 `tasks.md`。

## 11. 证据自身

- 本报告与 `index.json` 位于 `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d03/`。
- 报告中的哈希/字节/行数均为本轮写后即时实测；`index.json` 记录同一组最终值与改动清单。
- 本目录不包含任何原始备份、工具输出文件或测试日志；所有结论都可从三个目标文件的当前字节与改前副本重建验证。
- 最终哈希与字节记录在 `index.json` 的 `artifacts` 与 `evidence`：三个目标文件的评审返工后值，以及本报告写完后的实测值。报告与索引都不内嵌自身哈希（会自指），两者由 Lead 在目录级复测。
- 改前快照（`RV2-03`）：`snapshots/design-before-r03.md`（SHA256 `B452E38E7074B1D483CB8A0EB511A7EF858B3F35C13CC0841A75118E4B7F013C`，846941 字节）与 `snapshots/framework-before-r03.md`（SHA256 `3A3B2B214D9732BB61A0C8768CEF701B08F65ED3D8161C85F254A0F4B37275B8`，154074 字节）；两者是改前文件的逐字节副本，只读复制、未作任何修改。
- 评审基线：`design-d03/review-r01/report.md`（设计评审 CHANGES REQUIRED、安全评审 CHANGES REQUIRED）与 `design-d03/review-r02/report.md`（RV-01～RV-05 已 CLOSED，`RV2-01` 阻断）；均为 Agent 评审，不是人的接收或安全签收。
- 证据文件自检：`index.json` 以 `ConvertFrom-Json` 解析通过（`schema=add.evidence.design-decision/v1`、`item=pending-B13J-D03`），两个证据文件均为 UTF-8 无 BOM、纯 LF、末尾单 LF、零控制字符；本报告 4.1 节逐行引文（`:83`、`:3858`、`:3862` r02/r03、`:3870`）与 4.2 节 ADR-0031 全文块均与当前文件按 ordinal 逐字符比较相同。

## 12. 评审返工处置（B13J-D03-RV-01～RV-05）

评审输入：`.spec/wip/recovery-runtime/evidence-b13j-r01/design-d03/review-r01/report.md`。设计评审 CHANGES REQUIRED、安全评审 CHANGES REQUIRED；逐条 14 项中 12 项 PASS，阻断项是权限矩阵不变量 `RV-01`。

| 编号 | 级别 | 评审问题 | 本轮修法 | 落点 | 结论 |
|---|---|---|---|---|---|
| `RV-01` | MEDIUM | `design.md:3862` 只有“root 不向子进程传递”，缺执行身份的写入侧否定；ADR 无强制机制；`framework.md:1044` 追溯失真 | 在 `:3862` 补“只允许对目标 DB 执行 restore 与只读验证，不得写 Journal/keyring、部署锁、Store 绑定或审计，不得自行建立权威 binding”，并新增独立数据库角色（权限不得覆盖 identity/recovery 表写入）与“角色和 GRANT 矩阵由批准记录固定、未证明即 closed”的强制机制；ADR 解除条件新增负向验证门禁；重写追溯句 | `design.md:3862`；`framework.md:1036`（新增）、`:1045` | CLOSED（合同层；GRANT 实测留实现阶段） |
| `RV-02` | LOW | `:3862` 未同步“工具/备份供应方不得自批”、“root 不代替执行、不进 shell、不重放”、“除目标连接外无网络” | 在同一次 `:3862` 重写内补齐三项 | `design.md:3862` | CLOSED |
| `RV-03` | LOW | ADR-0030 `:926` 与 ADR-0031 `:981` 的 closed 口径宽窄不同 | `:981` 增加合并口径：生产 `enter`/`install` 与 `enter`/`resume` 的 DB 替换路径均保持 closed，二者是不同入口而非范围收窄 | `framework.md:981` | CLOSED |
| `RV-04` | LOW | 报告称 `:83` “前 744 字符不变”，与实测不符 | 改为“最长公共前缀 1184 字符，尾部追加 462 字符 / 760 字节”，并写明测量方法 | 本报告 §4.1 表与 §6 | CLOSED |
| `RV-05` | LOW | 未禁止 `su`/`sudo`/脚本包装的身份切换 | `:1006` 与 `:3862` 明确禁止经 shell 脚本、`su`、`sudo` 等包装实现身份切换，要求由批准记录固定并在实现阶段以隔离实测证明 | `framework.md:1006`；`design.md:3862` | CLOSED（机制未选，留实现阶段） |

`RV-04` 测量方法：对 `design.md:83` 的改前/改后整行按 UTF-16 码点逐字符比较，最长公共前缀 1184 字符；改后整行 1646 字符，尾部追加 462 字符，按 UTF-8 编码为 760 字节。

边界说明：

- 本轮只改 `design.md:3862` 一行；`framework.md` 只改 ADR-0031 内 `:981`、`:1006`、`:1045` 并新增 `:1036`；`README.md` 零改动。
- 不声称任何工具、备份、执行身份、ACL、进程隔离或恢复演练已验证；不声称 Case18/S8.2 PASS，未解除 `blocked-upstream`。
- 复审由独立评审执行；本报告不预判 PASS。

## 13. review-r02 返工处置（B13J-D03-RV2-01～RV2-03）

评审输入：`.spec/wip/recovery-runtime/evidence-b13j-r01/design-d03/review-r02/report.md`（设计评审 CHANGES REQUIRED、安全评审 CHANGES REQUIRED；`RV-01`～`RV-05` 已全部 CLOSED，问题集中在 review-r01 返工新引入的内容）。

| 编号 | 级别 | 评审问题 | 本轮修法 | 落点 | 结论 |
|---|---|---|---|---|---|
| `RV2-01` | MEDIUM | `design.md:3862` 的“整库 artifact 与受保护表边界（排除相应 schema 或经隔离库后由 root 受控切换）”与 `:3864`/`:3866`/`:3870` 冲突；`framework.md:1036` 的门禁与整库还原不可同时满足 | 删除该边界句；写入边界改为两层（替换窗口外该角色权限不得覆盖受保护对象的任何写入；替换窗口内对目标库的写入只能来自已批准 artifact 的整库还原本身，本次权威 binding 只由 root 经同一 Store 短事务创建/核）；受保护对象清单降为 restore 完成后的核对项，明确不改变整库语义、不引入 schema 级排除或 root 数据搬移；`framework.md:1036` 门禁改写为可满足形式 | `design.md:3862`；`framework.md:1007`、`:1034`、`:1036`、`:1045` | CLOSED（合同层；GRANT/隔离实测留实现阶段） |
| `RV2-02` | LOW | `framework.md:1045` 把“进程组/no-follow”归给 `:3862`，而 `:3862` 写“进程树”，no-follow 在 `:3870` | `design.md:83` 术语 `进程组`→`进程树`（等长等字节，行仍 1646 字符 / 2508 字节）；`framework.md:1045` 追溯改为“进程树/ENV allowlist 与 fd 隔离预检（no-follow 稳定 fd 见 `:3870`）” | `design.md:83`；`framework.md:1045` | CLOSED |
| `RV2-03` | LOW | 上轮缺少可留存的改前副本，导致“除声明行外零改动”无法字节级证明 | 动笔前把 `design.md`、`framework.md` 各复制一份到 `snapshots/`，本轮以快照逐行/逐字节比对 | `snapshots/design-before-r03.md`、`snapshots/framework-before-r03.md` | CLOSED |

整库语义自查（评审要求的描述约束）：

- 每条写入不变量均在“整库 restore”语义下能同时成立；本轮没有留下任何“只在部分还原下才成立”的条款，也没有为它新增例外路径。
- “执行身份不得写 Store”与“整库还原必然包含 Store 表”的处理方式：artifact 内的既有历史行（含历史 run/binding 行）只作为被还原的数据，不构成本次权威；本次 running 与不可变 binding 只能由 root 在替换完成后以独立读取路径与同一 Store 短事务创建或核（沿用 `design.md:3866`），`VerifyRestored` 仍须对应同一 artifact/target 的 no-follow 稳定 fd（`:3870`）。
- 这不是隐蔽放宽：执行身份仍不得自行构造、注入或声明任何权威 binding，不得写 Journal/keyring 或部署锁；替换窗口外其数据库角色权限不得覆盖受保护对象的任何写入；受保护对象（Journal、keyring、部署锁、Store 权威绑定与审计记录）以批准记录固定的可检清单作为 restore 完成后的核对项。
- `framework.md:1036` 的负向验证门禁据此改写为：替换窗口外以执行身份凭据或其目标 DB 角色写入 Journal/keyring/部署锁/Store 权威绑定/审计必须被拒绝；替换窗口内除已批准整库 artifact 的还原本身外不得产生任何 Store 权威绑定或 Journal/keyring/部署锁写入，执行身份自行构造、注入或声明的 binding 必须被拒绝——不再保留与整库还原矛盾的“角色写入 identity/recovery 表必须被拒绝”。

边界说明：

- 本轮只改 `design.md` 的 `:83`（等长术语）与 `:3862`（整行替换），行数仍 4924、crlf 169 / bareLF 4755；`framework.md` 只改 ADR-0031 内 `:1007`、`:1034`、`:1036`、`:1045` 四行、无增删行（行数仍 1049，纯 CRLF）；`README.md` 零改动。
- 不声称任何工具、备份、执行身份、ACL、GRANT、进程隔离或恢复演练已验证；不声称 Case18/S8.2 PASS，未解除 `blocked-upstream`。
- 复审由独立评审执行；本报告不预判 PASS。
