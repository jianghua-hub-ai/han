# B13J D04 定向复审报告（review-r02）

- **评审角色**：add-reviewer（非作者独立评审；延续 D02/D03 的同一评审者身份）
- **复审对象**：B13J D04 第二轮修订（应答 `B13J-D04-RV-01`、`B13J-D04-RV-02` 与 Lead 追加 `RV-03`）
- **工作区**：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **复审时间**：2026-11-02（Asia/Shanghai）
- **本轮性质**：定向复核 + 回归检查；未重跑首轮 12 条全量，仅对受改动直接影响的证据重新实测
- **写入范围**：仅本目录两个文件

## 1. 复审范围与实测基线

被测对象为 4 个文件的当前工作区副本（非 Git 提交），基线取作者提供的 `design-d04/snapshots-r02/` 改前副本。以下数值均为本轮自查实测，未采信作者自述。

| 文件 | SHA256 | 字节 | 行数 | crlf:bareLF |
|---|---|---|---|---|
| `identity-auth/design.md`（r60） | `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` | 854991 | 4944 | 169:4775 |
| `article-management/design.md`（r28） | `690015BB5C3B4800539F98C5B75115A5A20B5C161ECE7422F9F271840F7C55A7` | 377424 | 3267 | 0:3267 |
| `decisions/framework.md`（ADR-0032） | `58C97FC06361022AA48EF737189EF3EB069A7AAD25772BCA4C80FE4D1D5E1F8E` | 169890 | 1118 | 1118:0 |
| `decisions/README.md` | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` | 21472 | 110 | 2:108 |

改前副本（`snapshots-r02/`）实测，与首轮实测值逐一相符：

| 副本 | SHA256 | 字节 |
|---|---|---|
| `identity-before-r02.md` | `4D2FC4AC638094BEB1564DF041A973ABA29FE3E43B5D585117BCC586777994AB` | 854160 |
| `framework-before-r02.md` | `C1E5F80354A442D446E988AC60A45806ACC73F401563F19A17063B7B74A8B25C` | 168354 |
| `article-before-r02.md` | `639D23262DE0D5D292B10FC29D371A9A908C33BD320E901E6BAE4BE9C5647F15` | 377416 |
| `readme-before-r02.md` | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` | 21472 |

因此 `snapshots-r02/` 确为上一轮验收基线的字节级副本，可用于本轮最小性判定。

## 2. RV 状态汇总

| RV | 上轮级别 | 上轮状态 | 本轮状态 | 一句话依据 |
|---|---|---|---|---|
| `B13J-D04-RV-01` | MEDIUM | OPEN（blocking） | **CLOSED** | `identity:3616` 已改为"已登记 anchor adapter bundle 四端口集合中的 `recovery.DeploymentExclusive`"，与代码 `recovery_sources.go:425-430/455-466/470-492/523-531`、`recovery_runtime.go:551/555/565-567` 逐项对得上 |
| `B13J-D04-RV-02` | LOW | OPEN（non-blocking） | **CLOSED** | `article:69` 已删除 ADR-0032 依据，改指同轮 11 端口清单（`identity:3615`）与 gap-map U6；全文已无 ADR-0032 引用 |
| `RV-03`（Lead 追加） | — | OPEN | **CLOSED** | `framework:1088`、`identity:3908`、`identity:3922` 三处同口径补入"本地 POSIX 文件系统 / 共享 NFS 未证明前不宣称支持"，与 ADR-0019 `:471` 原文同义，未新增 Windows 例外、未扩矩阵 |

## 3. 定向结论（6 项）

### 3.1 RV-01 是否真关闭

**结论：CLOSED。**

`identity-auth/design.md:3616` 现文：唯一生产来源 = "已登记 anchor adapter bundle 四端口集合中的 `recovery.DeploymentExclusive`（ADR-0030 registry；bundle 四项须齐备，缺一即 `ErrAdmissionBinding`；空、未登记或配置路径不可用即四项一起落 `recoveryClosedPort`）"；"root 侧唯一 `DeploymentExclusive` 适配器形态为 `restoreDeploymentExclusive`（ADR-0032），当前无生产构造点"；状态列写明"`newRecoveryAnchorRegistry(nil)` 使 registry 恒空，`exclusive` 只能是 `recoveryClosedPort`"、"`recoveryRootInputs.exclusive` 仅为合成／测试注入，生产不得用它替代 registry 来源"。

逐条对照代码，全部为真：

- `cmd/han-server/recovery_sources.go:425-430`：`recoveryAnchorAdapter` 确为四字段（`Commitments`/`Sources`/`Anchor`/`Exclusive`）。
- `:455-466`：四项任一为 nil 即返回 `recovery.ErrAdmissionBinding`，即"缺一即拒绝登记"。
- `:470-492`：`resolve` 在 adapterID 为空/未登记、configPath 非绝对路径时，用同一 `recoveryClosedPort` 同时填四个端口——"四项一起落 closed"为真。
- `:523-531`：`recoveryClosedPort` 实现 `Acquire` 并断言 `recovery.DeploymentExclusive`。
- `recovery_runtime.go:551`：生产路径固定 `newRecoveryAnchorRegistry(nil)`，registry 恒空。
- `:555` 解析后 `:565-567` 才允许 `inputs.exclusive` 覆盖。
- `cmd/han-server/recovery_exclusive.go`：仅定义 `restoreDeploymentExclusive`，`Acquire` 无 acquire 即 `ErrAdmissionUnsupported`；`rg` 全仓非 md 命中仅定义与 `recovery_exclusive_test.go`，无生产构造点——"当前无生产构造点"为真。

**关于"把部署锁来源耦合到 anchor adapter bundle 是否形成不可满足的解除路径"的判定：不构成死锁，可接受。** 理由与可核性如下：

1. **无环**：ADR-0030 只规定"首期"不提供生产 CAS provider，registry 是被登记驱动的有界集合而非永久空集；gap-map `:119` 已把 `S4.23`（anchor/CAS adapter 生产实现）登记为后续候选。依赖链为 `S4.23 bundle → ADR-0032 锁后端与 ACL 证据 → 正式 Task 登记 → 人的生产批准`，有向无环。
2. **不新增关键路径依赖**：生产恢复今天已整体 closed，因为 `recovery_runtime.go:586` 在生产构造时恒传 `recoveryClosedJournalAnchor`；任何恢复状态变更本就排在 ADR-0030 交付之后。把锁挂在同一 bundle 上不改变先后次序，只让"一个 adapter = 一组自洽 provider"这一全有全无性质成立。
3. **与 D04 关闭条件 1 不冲突**：该条件要求"11 端口的唯一生产来源可逐项列出"，不要求"本轮可交付"。修订后 `exclusive` 已能指出唯一来源与唯一当前落点。
4. **顺序可核**：`framework.md:1096` 新增第 9 条把顺序写成可检验的三段，并明确"该顺序完成前 `exclusive` 的生产装配只能是 `recoveryClosedPort`"。

残余问题（不阻断，见 `B13J-D04-RV2-01`）：该"生产不得用 `recoveryRootInputs.exclusive` 替代 registry"目前只是文本约束，代码侧 `recovery_runtime.go:565-567` 的覆盖是无条件、无来源校验的；今天无生产调用方设置它（`rg "exclusive:" --glob '!*_test.go' cmd internal` exit 1），但未来误用没有被机制挡住。

### 3.2 RV-02 是否真关闭

**结论：CLOSED。**

- `article-management/design.md:69` 现文改为"依据同轮 D04 的 11 端口清单（identity `:3615` 的 `articleAccount` 行）与 gap-map U6"。
- 实测 `identity:3615` 确为 `articleAccount` 行，指向正确。
- `rg -n "ADR-0032" article-management/design.md` 无命中（exit 1），全文已无对 ADR-0032 的错误归属。
- 该行其余内容（语义 owner、唯一装配点、同事务、禁止 identity 复制 SQL、未装配 closed、draft/非人签字声明）逐字未变，未借机扩张结论。

### 3.3 RV-03 是否真关闭

**结论：CLOSED。**

- `framework.md:1088` 增补："锁文件必须位于已验证支持可靠 OS 独占锁语义的本地 POSIX 文件系统；共享/NFS 类在锁语义未证明前不宣称支持（与 ADR-0019 生产 `FileRecoveryJournal` 的本地文件系统前提同口径，不扩其支持矩阵）。"
- `identity-auth/design.md:3908` 配置表行、`:3922` 平台矩阵段同口径增补，措辞一致。
- 与 ADR-0019 `framework.md:471` 原文"当前生产 `FileRecoveryJournal` 仅支持已验证具备原子同目录替换、文件与目录同步及 OS 锁语义的本地 POSIX 文件系统。共享/NFS 类文件系统在锁与持久化语义未证明前不宣称支持"同义，属**沿用既有口径**而非新决定。
- 三处均未新增 Windows 例外，未扩 ADR-0026 的 Windows SID 矩阵，未扩 `FileRecoveryJournal` 的 POSIX 支持矩阵；`framework.md:1088` 与 `identity:3922` 均保留"不提供 Windows 生产例外 / Windows 生产 unsupported、退出码 2"。

### 3.4 回归检查

**结论：自洽，无新的前后矛盾。**

- `framework.md:1060`（继承关系）：新增的唯一来源说明与 `:1096`、`:1114` 表述一致；对 ADR-0019/0030/0031/0027 的"不改变"声明保留，且实测这些正文（位于第 1060 行之前与 ADR-0032 之外）字节未变。
- `framework.md:1096`（新增约束 9）：位于"不变量与约束"第 9 项，紧邻"解除条件"，内容为解除顺序与来源耦合，未把未完成事项写成已完成。
- `framework.md:1114`（引用记录）：引用行号 `:3908`/`:3922`/`:3601`/`:3616`/`:83` 与本轮实测逐项一致；`:1116` 的 `framework.md:916`/`:971` 分别对应 ADR-0030/ADR-0031 标题行，口径与既有条目相同。
- `identity:83` 追溯仍登记 r60、输入哈希 `B07BAFC7…`（即 r59 实际输出），并如实标注"Agent 产出、非人的签字、draft、人的精确接纳 unknown"。
- `identity:1055` 仍指向 `recovery_sources.go:641-662`；实测该区间正是 `recoveryClosedArticleAccount` 定义与接口断言。
- `identity:3601-3618` 除 `:3616` 外零改动；`:3612`/`:3613`（commitments/sources，归 ADR-0030 边界）与 `:3616`（同一 bundle，归属标 cmd root/platform）在"来源"上同为该 registry，在"决策归属"上分属 ADR-0030/ADR-0032，属既有归属口径的延续，不构成矛盾。
- `identity:3617` 指向 `recovery_runtime.go:631-640`；实测为 closed reader 兜底与 `newRecoverySecretSource` 构造，引用为真。
- `identity:3916`/`:3918` 未改动，仍为"唯一来源 `HAN_RECOVERY_INITIAL_ADMIN_SECRET`、无默认、无替代来源、读取不等于验证、经 `Journal.VerifyBeginSecret` 验密、不持久化/清零"。

### 3.5 格式与越界

**结论：全部不变，无越界。**

- 四文件均无 BOM；`identity`/`article`/`README` 末尾单 LF，`framework` 末尾单 CRLF。
- 行数/换行：`identity` 4944（169:4775）、`article` 3267（0:3267）、`framework` 1118（1118:0）、`README` 110（2:108），与基线一致。
- 控制字符：四文件除 CR/LF/TAB 外为 0；`identity` 唯一 TAB 位于字节偏移 468004（第 3303 行），与改前副本同偏移同数量，确认为既有 Go 代码块 TAB，非本轮引入。
- 越界：`tasks.md` 实测 `F1D65B0D7016CDBF9C2F4E85B012B5B0B9F714B6DF0C7CDC9326C05676900E34`、`plans/ir-01-content-community-batches.md` 实测 `3FA19CDD4902F56B8C2F1DF59C570D96FAAE02D8AEE519B4B4165DCCA3637380`，均与 D04 首轮基线相同，未改。
- 工作区扫描：23:59 之后被修改的文件只有 3 个被测文件（00:00:09/00:00:16/00:00:27）与 `design-d04/report.md`、`design-d04/index.json`（00:09）；`README.md` 未进入该集合，与其哈希不变一致；`design-d04/**` 之外无新增 `.spec` 文件。
- `git diff --check` exit 0。

### 3.6 诚实性

**结论：未发现新的不实断言。**

- ADR-0032 `:1095` 仍明确"不证明任何 OS 后端、ACL、双进程互斥、崩溃释放或三入口共用锁已实现或已通过测试""不声称生产装配、Case18、S8.2 或 E2E 通过"；`:1105` 明确人的生产批准不可被 Agent 产出、评审结论或测试通过替代。
- 全文出现的"已验证"仅作为限定语（"已验证支持可靠 OS 独占锁语义的本地 POSIX 文件系统"），是对文件系统前提的描述，不是对本轮成果的验证声明，与 ADR-0019 `:471` 用法一致。
- `identity:3922` 仍写"`recovery_exclusive.go` 当前仍是……closed 占位，具体 OS 后端实现属未来实施责任，本设计不声称已接线或验证"。
- production 仍 closed；未登记任何 Task 编号（ADR-0032 正文内无 `S4.x` 编号，`tasks.md` 无 `S4.22/23/24` 行）；未声称 Case18/S8.2 通过。

## 4. 关闭条件复核（gap-map `:160` 六项）

| # | 条件 | 本轮判定 | 依据 |
|---|---|---|---|
| 1 | 11 端口唯一生产来源可逐项列出 | **PASS** | `identity:3607-3617` 11 行；`exclusive` 行修订后与代码一致（RV-01 关闭） |
| 2 | 部署锁 OS/平台与拒绝路径明确 | **PASS（合同层）** | ADR-0032 `:1064`/`:1088`/`:1093`；`identity:3908`/`:3922`；无运行时证据，ADR 自陈 |
| 3 | article 侧拥有 canonical orphaning | **PASS** | `article:69`、`article:2432-2439`、`identity:1055`、`recovery_sources.go:641-662` |
| 4 | secret 只读候选且由 Journal 验密 | **PASS（合同层）** | `identity:3916`/`:3918`，本轮未改动 |
| 5 | 设计 Review PASS | **PASS WITH MINOR** | 本报告第 6 节 |
| 6 | production 仍 closed | **PASS** | `newRecoveryAnchorRegistry(nil)`、`recoveryClosedPort`、`recoveryClosedArticleAccount`、`recoveryClosedJournalAnchor` |

## 5. Findings

本轮新增 2 条 LOW，均不阻断；`B13J-D04-RV-01`/`RV-02`/`RV-03` 已关闭，无新增 HIGH/MEDIUM。

### B13J-D04-RV2-01（LOW，OPEN，non-blocking）

- **位置**：`identity-auth/design.md:3616`、`decisions/framework.md:1060`；代码 `cmd/han-server/recovery_runtime.go:565-567`
- **事实**：设计把 `recoveryRootInputs.exclusive` 限定为"仅为合成／测试注入、生产不得用它替代 registry 来源"，但代码中的覆盖是无条件的：`if inputs.exclusive != nil { anchors.Exclusive = inputs.exclusive }`，既不看来源也不看是否已从 registry 解析成功。实测当前无生产调用方设置该字段（`rg "exclusive:" --glob '!*_test.go' cmd internal` exit 1），所以文本描述与现状不矛盾。
- **影响**：约束只存在于文档；若未来 S4.22 装配时有人以 `recoveryRootInputs.exclusive` 直接接线，将绕过本轮确立的唯一来源，且不会有任何测试失败。属于"留给实施阶段关闭"的残余风险，不影响本轮合同层结论。
- **建议**：在 S4.22 的验收项中加一条负向要求——生产装配不得设置 `recoveryRootInputs.exclusive`，或以显式"仅测试构建"开关隔离该覆盖路径，并给出可失败的负向测试。不在本轮改稿。

### B13J-D04-RV2-02（LOW，OPEN，non-blocking）

- **位置**：`decisions/framework.md:1096`（约束 9）与 `:1100-1106`（解除条件 1-7）
- **事实**：顺序依赖写在"不变量与约束"第 9 条，而"解除条件"1-7 中没有一条要求"`exclusive` 经 registry 接入 production root 且可核"。解除条件 1-4 描述的是锁后端实现与证据，理论上可在没有任何注册路径的情况下全部满足，此时"解除"仍无法落地。
- **影响**：不会造成不可满足（第 5 条允许"完成 ADR 修订或新增决策"，且 `:1096` 已紧邻说明），但解除条件清单单独阅读时缺少一条可勾选的门禁，容易被后续 Task 规划漏掉。
- **建议**：在后续 ADR 修订或 Task 登记时，把"经 registry 的唯一来源接线并经批准记录可核"并入解除条件清单（或明确标注为条件 5 的子项）。属文档结构收口，不阻断。

## 6. 评审结论

### 设计评审结论：**PASS WITH MINOR**

RV-01/RV-02/RV-03 三项均已实质关闭；11 端口唯一来源、ADR-0032 内部一致性、article 归属、secret candidate 合同、追溯与格式均成立。残余两条 LOW 属实施阶段门禁与文档结构收口。

### 安全评审结论：**PASS WITH MINOR（不构成安全签收）**

从安全视角复核：唯一来源收敛后，生产 `exclusive` 只剩 `recoveryClosedPort` 一条路径，未新增绕过 fail-closed 的入口；"无自动 fallback、无环境推导、无 caller bool/no-op"三类禁止仍完整保留；POSIX/Windows 矩阵与 ADR-0019 同口径，未产生 Windows 生产例外；"本地 POSIX 文件系统 / 共享 NFS 未证明前不宣称支持"是收紧而非放宽。残余风险为 `recoveryRootInputs.exclusive` 覆盖路径缺乏机制约束（RV2-01），当前无生产调用方，故不作为阻断项。

### 总判定：**PASS WITH MINOR**

本判定仅覆盖合同层。它不构成人的接收、设计 Gate、安全签收，不关闭 `pending-B13J-D04`，不解除 `blocked-upstream`，不解锁 S4.22/S4.23/S4.24 中任何候选 Task。

## 7. 实测附录

### 7.1 差异行集合（`git diff --no-index --unified=0`，1-based 新行号）

| 文件 | 实测差异行 | 作者声明 | 一致 |
|---|---|---|---|
| `identity-auth/design.md` | `3616, 3908, 3922` | `{3616, 3908, 3922}` | 是 |
| `decisions/framework.md` | `1060, 1088, +1096（新增）, 1114` | `{1060, 1088, +1096, 1114}` | 是 |
| `article-management/design.md` | `69` | `{69}` | 是 |
| `decisions/README.md` | 无 | 无改动 | 是 |

字节级佐证（非行级）：

- `framework.md` 与副本首个不同字节偏移 156298（正落在第 1060 行），此前字节逐字节相同；公共后缀 943 字节（第 1114 行之后未动）；总增量 1536 B。
- `identity-auth/design.md` 首个不同字节偏移 529505（第 3616 行），此前字节逐字节相同（含 `:83`、`:1055`、`:3601-3615`）；公共后缀 256127 字节；增量 831 B。
- `article-management/design.md` 首个不同字节偏移 7593（第 69 行），增量 8 B。
- `README.md` 首个不同字节偏移不存在（逐字节相同）。

### 7.2 实测命令与结果

| # | 命令 | exit | 结果 |
|---|---|---|---|
| 1 | `rg -n "HAN_RECOVERY_RUNTIME_LOCK_PATH\|ArticleAccountPort\|HAN_RECOVERY_INITIAL_ADMIN_SECRET\|生产 root 装配" .add/project/knowledge/design-workspace/identity-auth/design.md` | 0 | 14 行命中，含 `:83`、`:1055`、`:3601`、`:3615`、`:3616`、`:3617`、`:3908`、`:3916`、`:3918` |
| 2 | `rg -n "ArticleAccountPort\|OrphanArticlesCanonical\|canonical" .add/project/knowledge/design-workspace/article-management/design.md` | 0 | 91 行命中 |
| 3 | `rg -n "部署锁\|文件锁\|Windows\|POSIX\|ADR-" .add/project/knowledge/decisions/framework.md` | 0 | 103 行命中，含 `:1054`、`:1060`、`:1062`、`:1064`、`:1074`、`:1088`、`:1114` |
| 4 | `git diff --check -- <identity, article, framework>` | 0 | 无空白错误，仅 CRLF 工作区提示性 warning |
| 5 | `rg -n "exclusive:" --glob '!*_test.go' cmd internal` | 1 | 无命中：生产 `recoveryRootInputs` 未设置 `exclusive` |
| 6 | `rg -n "restoreDeploymentExclusive" --glob '!*.md' .` | 0 | 仅 `recovery_exclusive.go` 定义与 `recovery_exclusive_test.go`，无生产构造点 |
| 7 | `rg -n "ADR-0032" .add/project/knowledge/design-workspace/article-management/design.md` | 1 | 无命中，RV-02 的依据错配已清除 |

### 7.3 诚实性扫描

- 未发现"已实现/已接线/已验证/测试通过"用于描述本轮成果的断言；`framework.md:1095` 保持否定式声明，`:1105` 保留"人的生产批准不可被替代"。
- 未出现未登记 Task 编号；`tasks.md` 无 `S4.22/S4.23/S4.24` 行。
- 未声称 Case18 或 S8.2 通过。

## 8. 未覆盖范围与复审条件

- 未运行任何 Go 测试、集成测试或真实服务；未验证 OS 锁后端、ACL、双进程互斥、崩溃释放、三入口共锁（ADR-0032 解除条件 1-4 全部仍为未满足）。
- 未验证 `ArticleAccountPort` 的真实接线、GRANT 矩阵或恢复 secret 读取行为。
- 未执行作者的反向还原程序（不写被评对象）；以字节级 diff 作为等价证据。
- 未复核本轮未改动的 ADR-0019/0030/0031/0027 正文语义（仅以字节未变证明其未被改）。
- 未运行完整 `git diff --check` 全仓（仅针对 4 个被测文件），其余工作区既有改动非本轮范围。
- 复审条件：若上述 4 文件中任一文件在本次报告后再次变更，或 `recoveryRuntime` 装配路径改变，需重新定向复核。

## 9. 声明

本报告由非作者评审者独立产出，评审期间未修改任何被评审文件，未执行 Git 写操作，未删除或移动文件。判定为合同层结论，不代替人的接受、设计 Gate、安全签收或生产批准。