# B13J ADR 前置节点返工 r02 证据报告（ADR-0030 / ADR-0031 / ADR-0032 → r02）

- 项目项：`pending-B13J-D04` 之后的 ADR 前置节点返工（承接非作者规划评审 `planning-review-r01` 的 CHANGES REQUIRED）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（全程未变）
- 角色与产出者：Arch-Designer（`add-arch-designer`，Agent 文档产出，不是人的签字、独立评审、安全评审或生产批准）
- 记录时间：2026-11-01（Asia/Shanghai）
- 评审来源：`.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r01/report.md`（SHA256 `6918A79C1E5A7426037A47E8DA5CEDC5E9F839B39BF0053B1DFE26712254D4AC` / 28901 B，实测）
- 结论状态：`verdict=applied-draft-unreviewed`（作者侧处置完成，等待非作者复审；本轮不声称评审 PASS、人的接纳或实施授权）

## 1. 结论

1. 评审 four findings 全部按作者侧处置落地，且只落在 ADR-0030 / ADR-0031 / ADR-0032 自身内（r01→r02），**未新开 ADR**，未改动 ADR-0019 / ADR-0027 / ADR-0029 正文。
2. ADR-0030、ADR-0031 的「首期不提供」结论与 ADR-0032 的 POSIX-only / Windows 生产 unsupported（退出码 2）/ 无 fallback 结论**逐字节未变**（决策行与解除条件行均不在差异集合内，见 §7）。
3. `decisions/framework.md`：169890 B / 1118 行 → 175141 B / 1135 行（净 +17 行）；保持**纯 CRLF**（crlf=1135、bareLF=0）、无 BOM、末尾单 CRLF、控制字符 0、TAB 0。
4. `decisions/README.md` **零改动**：`2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` / 21472 B（改前基线同值）；索引行不含修订号字段，故 r02 无需改索引（证据见 §10）。
5. F010 收口：`design-revision-r01` / `design-d02` / `design-d03` / `design-d04` 四个 `index.json` 的 state 更新为真实复审结果；`design-revision-r01/report.md` 追加 §7 记 `B13J-D01-RV-01` CLOSED（不改历史正文）。
6. 本轮**未**登记或启动任何实现卡：S4.22 / S4.23 / S4.24 一律写作「候选」「尚未登记为正式 Task」；生产继续 closed；未声称任何锁后端、ACL、双进程互斥、崩溃释放、restore 工具、备份、恢复演练已验证。

### 1.1 Finding → 落点（改后实测行号）

| Finding | 严重度 | 落点（`framework.md` 改后行号） | 处置摘要 |
|---|---|---|---|
| `B13J-PLAN-F004` | HIGH | `:960`、`:961`、`:969`；`:1046`-`:1048`、`:1057` | ADR-0030 新增约束 9（后续阶段实施门禁）与约束 10（`Exclusive` 与 CAS 头锚点同源）；ADR-0031 新增「后续阶段前置节点」小节 |
| `B13J-PLAN-F005` | HIGH | `:1115`-`:1122`、`:1135` | ADR-0032 解除条件新增第 8 条顺序链：ADR-0030 修订 → 候选 S4.23 bundle → POSIX 锁后端 → 候选 S4.24（requires 候选 S4.23）→ 候选 S4.22 → 正式登记与人的批准 |
| `B13J-PLAN-F007` | MEDIUM | `:1115`（规范性条件）、`:1122`（判定理由） | 把约束 9（改后 `:1104`）的顺序要求落成可勾选的规范性解除条件，不再挂空 |
| `B13J-PLAN-F010` | LOW | 四个 `index.json`；`design-revision-r01/report.md` §7 | 状态收口到真实复审结果；`B13J-D01-RV-01` 记 CLOSED |

## 2. 基线与改后实测

| 文件 | 时点 | SHA256 | 字节 | 行数 | crlf | bareLF | BOM | 末尾 | 控制字符（非 CR/LF/TAB） |
|---|---|---|---|---|---|---|---|---|---|
| `decisions/framework.md` | 改前（快照实测） | `58C97FC06361022AA48EF737189EF3EB069A7AAD25772BCA4C80FE4D1D5E1F8E` | 169890 | 1118 | 1118 | 0 | 无 | 单 CRLF | 0（TAB=0） |
| `decisions/framework.md` | 改后（实测） | `DA77EA1332DB69CB94B90960DEA58B35CABDA76C712EDAE23F2CCF6A1F45B5C1` | 175141 | 1135 | 1135 | 0 | 无 | 单 CRLF | 0（TAB=0） |
| `decisions/README.md` | 改前=改后（实测） | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` | 21472 | 110 | 2 | 108 | 无 | 单 LF | 0（TAB=0） |

- 行数口径：`[IO.File]::ReadAllLines($p).Count`。改后 1135 行 = 改前 1118 行 + 净增 17 行（+21 增 / -4 删，见 §6）。
- 两份改前副本：`snapshots/framework-before-r02.md`（`58C97FC0…1F8E` / 169890 B / 1118 行）与 `snapshots/readme-before-r02.md`（`2EE124AE…86B97` / 21472 B），逐字节核过。

## 3. 与任务书一处硬约束的算术冲突（需 Lead 确认口径）

任务书写「ADR-0019 前 155139 字节必须零改动（`D74F09A0…1C12`）」，同时又要求修订 ADR-0030/0031。实测：

- 改前文件前 155139 字节 = `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12`（与任务书一致，确认基线正确）；
- ADR-0030 锚点起始字节偏移 = **126599**（改前改后相同），即 ADR-0030/0031 的正文本身位于「前 155139 字节」区间内；
- 因此「前 155139 字节零改动」与「修订 ADR-0030/0031」在算术上互斥；改后前 155139 字节为 `F918A62D8AE0C6C73D08B0F883452904055C78CB33754248689C9A4262B77C03`（不同）。

本轮采用的**可辩护替代不变量**（三条同时成立）：

1. **ADR-0030 锚点之前的全部字节逐字节相同**：0–126598 共 126599 B，改前改后哈希均为 `8C93C47AE57BED1C8390C07D41210183814185A68B148DCEEF180BDB8AE1C9EB`。该区间包含 ADR-0019 到 ADR-0029 的全部正文。
2. **ADR-0019 独立区间相同**：从 `## ADR-0019` 标题（第 418 行，字节偏移 31305）到 ADR-0030 锚点之前共 95294 B，改前改后哈希均为 `607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6`。
3. **首个差异字节偏移 = 126770**，落在 ADR-0030 的 `修订:` 行内（`adr-0030-r01` → `adr-0030-r02`），即差异从 ADR-0030 元数据行才开始。

请 Lead 确认以上替代口径；若必须坚持「前 155139 字节零改动」的字面要求，则本任务的 ADR-0030/0031 修订无法同时成立，需要先改任务书边界。

## 4. 逐 finding 处置

### 4.1 `B13J-PLAN-F004`（HIGH）— 缺少 ADR 修订前置节点

**问题**：候选 S4.23/S4.24 的生产实现与 ADR-0030 `:926`/`:939`/`:951`、ADR-0031 `:981`（改后 `:984`）的「首期不提供」冲突。

**处置**（保持两份「首期不提供」结论不变，只加实施门禁）：

- ADR-0030 约束 9（`framework.md:960`）：任何把生产 CAS provider 或 anchor adapter bundle 付诸实施的卡片——包括尚未登记为正式 Task 的 S4.23 一类候选编号——不得登记或启动，除非本 ADR 已有新修订给出具体产品、凭据、独立故障域与部署/ACL 结论，并按约束 7（`:953`）解除 closed。
- ADR-0030 约束 10（`framework.md:961`）：`Exclusive` 端口与 CAS 头锚点同源。生产 `exclusive` 的唯一来源是已登记 anchor adapter bundle 的四端口集合（`Commitments`/`Sources`/`Anchor`/`Exclusive`，缺一即 `ErrAdmissionBinding`）；首期 bundle 不存在的直接原因是 production root 以空集合构造 registry（`newRecoveryAnchorRegistry(nil)`）、未登记解析结果四项一起落 `recoveryClosedPort`，且本 ADR 未提供生产 CAS provider；后续只能按「本 ADR 新修订解除 closed → 同一 bundle 四端口齐备并作为唯一登记来源 → `exclusive` 才可能离开 closed」获得；禁止 file/SQL/fake/环境探测自动 fallback，禁止以 `recoveryRootInputs.exclusive`（仅限合成/测试）或 registry 之外的第二来源替代。
- ADR-0031「后续阶段前置节点」（`framework.md:1046`-`:1048`）：任何把生产 restore 工具、生产批准主体、专用执行身份或宿主隔离付诸实施的卡片（含 S4.24 一类候选编号）不得登记或启动，除非本 ADR 已有新修订给出批准主体、执行主体、宿主隔离与原生耐久回执的可核结论，并按约束 8 通过架构与安全独立评审。
- 两份 ADR 的引用记录各加一条 r02 触发说明（`:969`、`:1057`）。

**未改**：ADR-0030 决策 `:926`、解除条件 `:953`、候选表 D 行 `:939`；ADR-0031 决策 `:984`、解除条件 `:1034`。逐字节相同（§7）。

### 4.2 `B13J-PLAN-F005`（HIGH）— exclusive 依赖链不可满足、缺少可核顺序

**问题**：ADR-0030 首期 registry 恒空 ⇒ 无 adapter bundle ⇒ `exclusive` 只能 `recoveryClosedPort`；而 ADR-0032 要求 POSIX 锁后端，规划层缺 ADR 修订节点与「S4.24 requires S4.23」的可核顺序。

**处置**：ADR-0032 解除条件新增第 8 条（`framework.md:1115`-`:1122`），并在引用记录登记 r02 触发（`:1135`）：

- `:1115` 前置声明：条件 1–7 任一未满足即不得解除；条件 1–4 即使全部满足，`exclusive` 仍未按 registry 唯一来源接线时也不得判定解除。
- `:1116` ADR-0030 按其约束 7 与约束 9 完成新修订（生产 CAS provider、独立故障域、部署/ACL 结论），解除其 closed。
- `:1117` 候选 S4.23（anchor adapter bundle 四端口齐备并登记；尚未登记为正式 Task）完成，`Exclusive` 离开 `recoveryClosedPort`。
- `:1118` 本 ADR 解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据。
- `:1119` 候选 S4.24（`exclusive` 锁接线）**requires 候选 S4.23**：bundle 必须先于锁后端接线，不得先接线后补来源。
- `:1120` 候选 S4.22（production root 装配与 recovery secret candidate）必须在候选 S4.24 之后。
- `:1121` 解除条件 5–6 的正式 Task 规划登记与人的生产批准。
- `:1122` 判定理由：该链是线性前驱图，每个节点只依赖直接前驱、无回边，ADR-0030 自身已提供「经本 ADR 新修订解除 closed」的出边，因此**链无环、可满足**；当前不可满足的只是规划登记顺序（在 ADR-0030 修订前登记 S4.23/S4.24 一类实施卡），不是架构死锁；不改变 POSIX-only、Windows 生产 unsupported（退出码 2）与无 fallback 结论。

**与 F004 的一致性**：F004 的约束 9 是「实施登记门禁」，F005 的第 8 条是「解除条件顺序」；前者属于 ADR-0030 内部约束，后者属于 ADR-0032 解除条件，引用方向单一（ADR-0032 → ADR-0030），无循环。

### 4.3 `B13J-PLAN-F007`（MEDIUM）— 顺序约束未落入解除条件（规范性挂空）

**处置**：ADR-0032 约束 9（改后 `framework.md:1104`，改前 `:1096`，内容逐字节未变）原先只描述「解除顺序与来源耦合」；本轮在解除条件清单新增第 8 条把该顺序写成规范性可勾选条件，并由 `:1115` 明确「条件 1–4 全满足但 `exclusive` 未按 registry 唯一来源接线时也不得判定解除」，使约束 9 与解除条件互相咬合、不再挂空。
### 4.4 `B13J-PLAN-F010`（LOW）— 作者侧证据索引状态停在复审前；D01 LOW 无 CLOSED 记录

**处置**：

| 证据文件 | 改后 state | SHA256（实测） | 字节 | 说明 |
|---|---|---|---|---|
| `.spec/wip/recovery-runtime/evidence-b13j-r01/design-revision-r01/index.json` | `pass-with-minor-reviewed-r01` | `218EF70836BE69EBE9D6A2C7485AB9D638E20F7070421DEB48B86265B785A9EF` | 8224 | summary 写明 review-r01 PASS WITH MINOR 且 `B13J-D01-RV-01` 已在 report §7 记 CLOSED |
| `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d02/index.json` | `pass-reviewed-r02` | `1F2A1D19ED1FA74CB987BDA38442D94DE2BD725315CA72A0F07DCC469D748476` | 20071 | review-r02 PASS；ADR-0030 后续升 r02 但决定不变 |
| `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d03/index.json` | `pass-reviewed-r03` | `DAB0013E113B8AE2417016AF93162E2CD51A6C94DF148E52616790B5FBBB790B` | 45798 | review r03 PASS（设计+安全，合同层） |
| `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d04/index.json` | `pass-with-minor-reviewed-r02-open-lows` | `AD4EA53C4C3FF6C6F6238320B4973F0D4D27259650D2F0B5DA638CC757AC346D` | 55974 | review r02 PASS WITH MINOR；`verdict_detail` 明记 2 条 LOW 仍 OPEN（RV2-01、RV2-02；后者由 ADR-0032 r02 承接为规范性解除条件） |

- `design-revision-r01/report.md` 追加 §7「后续处置：B13J-D01-RV-01 CLOSED」，只追加不改历史正文；改后 `6D8B28A47F12A51A483E9660D19BA8D4DF1564CC4DA3DC3A334E8BD1F36264E4` / 11998 B（末尾无换行，原文件即以「。」结束）。内容给出核证依据：`identity-auth/design.md` 已递增至 r60（`:5` 与 `:38`），`:7` 为 `2026-11-01`，`:83` 依次登记 r58/r59/r60；`framework.md` 的 r58/r59/r60 追溯行实测在 `:965`/`:1052`/`:1130`。
- 四个 `index.json` 均通过 `ConvertFrom-Json` 校验；哈希与字节为改后实测。

## 5. 改动清单（逐行改前/改后原文）

全部改动集中在 `decisions/framework.md`；行号为**改后**实测行号，括号内为改前行号。

### C1 `:918`（改前 `:918`，替换）— ADR-0030 修订号

- 改前：`**目的**: 架构设计 | **仓库**: han | **修订**: adr-0030-r01`
- 改后：`**目的**: 架构设计 | **仓库**: han | **修订**: adr-0030-r02`

### C2 `:960`-`:961`（新增 2 行）— ADR-0030 约束 9 / 约束 10

插入在改前 `:960`（空行）之后、改前 `:961`（`### 引用记录`）之前；原 `:961` 起内容整体后移 2 行。

- 新 `:960`：`9. **后续阶段前置节点（实施门禁）**：任何把生产 CAS provider 或 anchor adapter bundle 付诸实施的卡片——包括规划中尚未登记为正式 Task 的 S4.23 一类候选编号——都不得登记或启动，除非本 ADR 已有新修订给出具体产品、凭据、独立故障域与部署/ACL 结论，并按约束 7 解除 closed。本 ADR 的 r02 只新增该门禁与下述来源关系，不改变“首期不提供”结论、候选取舍、约束 1–6 与解除条件实质。`
- 新 `:961`：`10. **anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源**：生产 `exclusive` 的唯一来源是已登记 anchor adapter bundle 的四端口集合（`Commitments`/`Sources`/`Anchor`/`Exclusive`，缺一即 `ErrAdmissionBinding`）。首期 bundle 不存在，是因为 production root 以空集合构造 registry（`newRecoveryAnchorRegistry(nil)`），未登记解析结果四项一起落 `recoveryClosedPort`，且本 ADR 未提供生产 CAS provider。后续只能在“本 ADR 新修订解除 closed → 同一 bundle 四端口齐备并作为唯一登记来源 → `exclusive` 才可能离开 closed”的可核顺序中获得；不得以 file、SQL、fake 或环境探测自动 fallback，不得用合成/测试注入（`recoveryRootInputs.exclusive` 仅限合成/测试）或 registry 之外的第二来源替代。`

### C3 `:969`（新增 1 行）— ADR-0030 r02 引用记录

插入在改前 `:966`（D02 决策授权行）之后；原 `:967` 空行及之后内容后移 1 行。

- 新 `:969`：`- 2026-11-01 r02 修订（触发：非作者规划评审 `B13J-PLAN-F004`/`F005`，报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r01/report.md`）：新增约束 9（后续阶段实施门禁）与约束 10（anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源），说明首期 bundle 不存在的原因与后续唯一可核顺序；决定结论、候选取舍、约束 1–6 与解除条件实质未变；未新增实现任务，S4.22/S4.23/S4.24 仍为未登记候选，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。`

### C4 `:976`（改前 `:973`，替换）— ADR-0031 修订号

- 改前：`**目的**: 架构设计 | **仓库**: han | **修订**: adr-0031-r01`
- 改后：`**目的**: 架构设计 | **仓库**: han | **修订**: adr-0031-r02`

### C5 `:1046`-`:1049`（新增 4 行）— ADR-0031「后续阶段前置节点」

插入在改前 `:1041`（约束 9 不适用场景行）与其后空行之后；原 `:1042` 空行、`:1043` `### 引用记录` 等整体后移 4 行。

- 新 `:1046`：`### 后续阶段前置节点`
- 新 `:1047`：空行
- 新 `:1048`：`任何把生产 restore 工具、生产批准主体、专用执行身份或宿主隔离付诸实施的卡片——包括规划中尚未登记为正式 Task 的 S4.24 一类候选编号——都不得登记或启动，除非本 ADR 已有新修订给出批准主体、执行主体、宿主隔离与原生耐久回执的可核结论，并按约束 8 通过架构与安全独立评审。本 ADR 的 r02 只新增该门禁，不改变“首期不提供生产 restore 工具与批准主体”结论、候选取舍、不变量与解除条件实质。`
- 新 `:1049`：空行

### C6 `:1057`（新增 1 行）— ADR-0031 r02 引用记录

- 新 `:1057`：`- 2026-11-01 r02 修订（触发：非作者规划评审 `B13J-PLAN-F004`，报告同上）：新增“后续阶段前置节点”小节，要求任何生产 restore 实现卡（含尚未登记为正式 Task 的 S4.24 一类候选编号）先经本 ADR 新修订给出批准主体、执行主体、宿主隔离与原生耐久回执结论并通过安全独立评审；决定结论、候选取舍、不变量与解除条件实质未变；未新增实现任务，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。`

### C7 `:1064`（改前 `:1056`，替换）— ADR-0032 修订号

- 改前：`**目的**: 架构设计 | **仓库**: han | **修订**: adr-0032-r01`
- 改后：`**目的**: 架构设计 | **仓库**: han | **修订**: adr-0032-r02`

### C8 `:1115`-`:1122`（新增 8 行）— ADR-0032 解除条件第 8 条

插入在改前 `:1106`（解除条件第 7 条）之后、空行之前；原 `:1107` 空行及之后内容后移 8 行。

- 新 `:1115`：`8. **解除顺序（前置链，须逐项可核且无环）**：条件 1–7 任一未满足即不得解除；条件 1–4 即使全部满足，`exclusive` 仍未按 registry 唯一来源接线时也不得判定解除。解除必须按下列顺序逐节点留证：`
- 新 `:1116`：`   - ADR-0030 按其约束 7 与约束 9 完成新修订（生产 CAS provider、独立故障域、部署/ACL 结论），解除其 closed；`
- 新 `:1117`：`   - 候选 S4.23（anchor adapter bundle 四端口齐备并登记；尚未登记为正式 Task）完成，`Exclusive` 离开 `recoveryClosedPort`；`
- 新 `:1118`：`   - 本 ADR 解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据；`
- 新 `:1119`：`   - 候选 S4.24（`exclusive` 锁接线）**requires 候选 S4.23**：bundle 必须先于锁后端接线，不得先接线后补来源；`
- 新 `:1120`：`   - 候选 S4.22（production root 装配与 recovery secret candidate）必须在候选 S4.24 之后；`
- 新 `:1121`：`   - 解除条件 5–6 的正式 Task 规划登记与人的生产批准。`
- 新 `:1122`：`   该链是线性前驱图：每个节点只依赖其直接前驱，无回边，且 ADR-0030 自身已提供“经本 ADR 新修订解除 closed”的出边，因此不是架构死锁；当前不可满足的只是规划登记顺序（在 ADR-0030 修订前登记 S4.23/S4.24 一类实施卡），不是架构不可达，也不改变本 ADR 的 POSIX-only、Windows 生产 unsupported（退出码 2）与无 fallback 结论。`

### C9 `:1132`（改前 `:1116`，替换）— ADR-0032 相邻决定行号引用

- 改前：`- 相邻决定：[framework.md（ADR-0030）](framework.md#adr-0030)，`framework.md:916` 起；[framework.md（ADR-0031）](framework.md#adr-0031)，`framework.md:971` 起；本 ADR 不改变二者的“首期不提供”与 closed 结论。`
- 改后：`- 相邻决定：[framework.md（ADR-0030）](framework.md#adr-0030)，`framework.md:916` 起；[framework.md（ADR-0031）](framework.md#adr-0031)，`framework.md:974` 起；本 ADR 不改变二者的“首期不提供”与 closed 结论。`

（ADR-0031 标题由改前 `:971` 位移到改后 `:974`，引用随实测改写。）

### C10 `:1135`（新增 1 行）— ADR-0032 r02 引用记录

- 新 `:1135`：`- 2026-11-01 r02 修订（触发：非作者规划评审 `B13J-PLAN-F005`/`F007`，报告同上）：在“解除条件”新增第 8 条解除顺序链，把不变量与约束第 9 条的顺序要求落成规范性可勾选条件，并明确候选 S4.24 requires 候选 S4.23、候选 S4.22 在其后；说明该链无环、可满足，当前不可满足的只是规划登记顺序。POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、三入口共锁与约束 1–8 的实质均未改变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。`

## 6. 差异行集合

`git diff --no-index -U0 snapshots/framework-before-r02.md framework.md` 实测（旧→新）：

| # | 旧行 | 新行 | 类型 | 内容 |
|---|---|---|---|---|
| 1 | `-918` | `+918` | 替换 | `adr-0030-r01` → `adr-0030-r02` |
| 2 | `-959,0` | `+960,2` | 新增 | ADR-0030 约束 9、约束 10 |
| 3 | `-966,0` | `+969` | 新增 | ADR-0030 r02 引用记录 |
| 4 | `-973` | `+976` | 替换 | `adr-0031-r01` → `adr-0031-r02` |
| 5 | `-1042,0` | `+1046,4` | 新增 | ADR-0031「后续阶段前置节点」（标题+空行+正文+空行） |
| 6 | `-1049,0` | `+1057` | 新增 | ADR-0031 r02 引用记录 |
| 7 | `-1056` | `+1064` | 替换 | `adr-0032-r01` → `adr-0032-r02` |
| 8 | `-1106,0` | `+1115,8` | 新增 | ADR-0032 解除条件第 8 条（主句+6 子项+判定理由） |
| 9 | `-1116` | `+1132` | 替换 | ADR-0031 行号引用 `:971` → `:974` |
| 10 | `-1118,0` | `+1135` | 新增 | ADR-0032 r02 引用记录 |

- `git diff --no-index --numstat`：`21 4`（21 增 / 4 删，净 +17 行）。
- 差异行集合之外的所有行逐字节相同（见 §7）。

### 6.1 关键锚点行号位移对照（改前 → 改后）

| 锚点 | 改前 | 改后 |
|---|---|---|
| ADR-0030 锚点 / 标题 / 修订 | `915` / `916` / `918` | `915` / `916` / `918` |
| ADR-0030 决策 / 约束 7（解除条件） | `926` / `953` | `926` / `953` |
| ADR-0030 约束 8（不适用场景） | `959` | `959` |
| ADR-0030 约束 9 / 约束 10 | —（新增） | `960` / `961` |
| ADR-0030 引用记录标题 / r58 记录 / r02 记录 | `961` / `963` / —（新增） | `963` / `965` / `969` |
| ADR-0031 锚点 / 标题 / 修订 | `970` / `971` / `973` | `973` / `974` / `976` |
| ADR-0031 决策 | `981` | `984` |
| ADR-0031 约束 8（解除条件） | `1031` | `1034` |
| ADR-0031 约束 9（不适用场景） | `1041` | `1044` |
| ADR-0031 后续阶段前置节点 / r02 记录 | —（新增） | `1046`-`1048` / `1057` |
| ADR-0032 锚点 / 标题 / 修订 | `1053` / `1054` / `1056` | `1061` / `1062` / `1064` |
| ADR-0032 约束 9（解除顺序与来源耦合） | `1096` | `1104` |
| ADR-0032 解除条件标题 / 第 7 条 / 第 8 条 | `1098` / `1106` / —（新增） | `1106` / `1114` / `1115`-`1122` |
| ADR-0032 不适用场景 / 引用记录标题 | `1108` / `1112` | `1124` / `1128` |
| ADR-0032 相邻决定（引用记录内） | `1116` | `1132` |
| ADR-0032 r02 记录 | —（新增） | `1135` |

## 7. 字节级差异与「其余字节不变」证明

改前快照 vs 改后：

- 首异字节偏移：**126770**（0-based），位于 ADR-0030 修订行内；
- 公共前缀长度：126770 B，改前改后前缀哈希均为 `0D80C9D34B5E16735AA9E6B0BBB8406398B37DBF8A11A99557173E6F32DA940F`；
- 公共后缀长度：**20 B**，内容 `E6 88 96 E7 94 9F E4 BA A7 E6 89 B9 E5 87 86 E3 80 82 0D 0A`（即「或生产批准。\r\n」）；
- 增量：**+5251 B**（169890 → 175141）。

不变行逐字节核验（改前行号与改后行号指向同一文本，`-ceq` 比较）：

| 语义 | 改前行 | 改后行 | 相同 |
|---|---|---|---|
| ADR-0030 决策（「首期不提供」） | `926` | `926` | 是 |
| ADR-0030 约束 7（解除条件） | `953` | `953` | 是 |
| ADR-0031 决策（「首期不提供」） | `981` | `984` | 是 |
| ADR-0031 约束 8（解除条件） | `1031` | `1034` | 是 |
| ADR-0031 约束 9（不适用场景） | `1041` | `1044` | 是 |
| ADR-0032 决策（POSIX-only） | `1064` | `1072` | 是 |
| ADR-0032 约束 9（改前已存在） | `1096` | `1104` | 是 |
| ADR-0032 解除条件第 7 条（Windows） | `1106` | `1114` | 是 |

前缀与独立区间（对应 §3 的替代不变量）：

| 区间 | 字节 | 改前哈希 | 改后哈希 | 相同 |
|---|---|---|---|---|
| 0 → ADR-0030 锚点前（覆盖 ADR-0019…ADR-0029） | 126599 | `8C93C47AE57BED1C8390C07D41210183814185A68B148DCEEF180BDB8AE1C9EB` | 同左 | 是 |
| ADR-0019 标题 → ADR-0030 锚点前 | 95294 | `607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6` | 同左 | 是 |
| 0 → 155139（任务书字面区间） | 155139 | `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12` | `F918A62D8AE0C6C73D08B0F883452904055C78CB33754248689C9A4262B77C03` | 否（见 §3） |

## 8. 反向还原验证

方法：以改后文件为输入，在内存中按 §6 的 10 处改动逐项撤销（先恢复 4 条替换行原文，再按降序删除 6 组新增行 —— 1135、1115–1122、1057、1046–1049、969、960–961），以 `\r\n` 连接并补末尾 CRLF，重新编码为 UTF-8（无 BOM）。

结果：

- 还原字节数 **169890**，行数 **1118**；
- 还原 SHA256 `58C97FC06361022AA48EF737189EF3EB069A7AAD25772BCA4C80FE4D1D5E1F8E`；
- 与 `snapshots/framework-before-r02.md` **逐字节相同**（布尔比较 `True`）。

即：撤销本轮全部改动可精确回到改前基线，无隐藏改动。

## 9. 格式实测

| 文件 | SHA256 | 字节 | 行数 | crlf | bareLF | BOM（前 3 字节） | 末尾 | 控制字符 <0x20 且非 CR/LF/TAB |
|---|---|---|---|---|---|---|---|---|
| `framework.md` 改前 | `58C97FC0…1F8E` | 169890 | 1118 | 1118 | 0 | 无（`23 20 E6`） | 单 CRLF | 0 |
| `framework.md` 改后 | `DA77EA13…B5C1` | 175141 | 1135 | 1135 | 0 | 无（`23 20 E6`） | 单 CRLF（`0D 0A`） | 0 |
| `README.md`（未变） | `2EE124AE…86B97` | 21472 | 110 | 2 | 108 | 无（`23 20 E6`） | 单 LF（`0A`） | 0 |

- 测量方法：`[IO.File]::ReadAllBytes` + `[Text.Encoding]::UTF8.GetString`；crlf 用 `` `r`n `` 正则计数，bareLF = 总 LF − crlf；行数用 `[IO.File]::ReadAllLines().Count`；控制字符逐字节扫描。
- `framework.md` 新增行全部使用 CRLF，未引入任何 bareLF；TAB 计数 0（本文件原本无 TAB）。
- `README.md` 的 2 CRLF / 108 bareLF 混合未被扩散（本轮零改动）。

## 10. `decisions/README.md` 零改动证据

- 哈希与字节：改前基线 `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` / 21472 B；本轮结束后实测同值。
- 索引行无修订号字段：`rg -n "adr-003[012]-r0" .add/project/knowledge/decisions/README.md` → **exit 1（0 命中）**，故 r02 不需要改索引；`rg -n "ADR-003[012]" README.md` 的既有条目仍指向 `#adr-0030`/`#adr-0031`/`#adr-0032`，锚点未变。
- `git diff --check` 对 `README.md` 无输出（仅 Git 的 autocrlf 提示），exit 0。

## 11. 验收命令与结果（实跑）

| 命令 | 结果 |
|---|---|
| `rg -n "后续阶段前置节点\|解除顺序\|adr-0030-r02\|adr-0031-r02\|adr-0032-r02" .add/project/knowledge/decisions/framework.md` | exit 0；命中 `918`、`960`、`976`、`1046`、`1057`、`1064`、`1104`、`1115`、`1135`（9 行） |
| `rg -n "^## ADR-003[012]" .add/project/knowledge/decisions/framework.md` | exit 0；命中 `916`、`974`、`1062` |
| `rg -n 'requires 候选 S4\.23\|S4\.22.*S4\.24\|候选 S4\.2[234]' framework.md` | exit 0；命中 `969`、`1117`、`1119`、`1120`、`1135`（5 行） |
| `rg -n "adr-003[012]-r0" .add/project/knowledge/decisions/README.md` | exit 1；0 命中（证明索引无修订号字段、README 未变） |
| `git diff --check -- .add/project/knowledge/decisions/framework.md .add/project/knowledge/decisions/README.md` | exit 0（仅 autocrlf 警告，无 whitespace error） |
| 控制字符扫描（framework.md 改后） | 非 CR/LF/TAB 控制字符 = 0；TAB = 0 |
| `ConvertFrom-Json`（4 个作者侧 index.json） | 全部合法 |
| `ConvertFrom-Json`（本目录 index.json） | 合法（复校通过，`verdict=applied-draft-unreviewed`、`findings=4`） |

## 12. 未覆盖范围

- **未**做独立评审：本报告是作者侧处置，`verdict=applied-draft-unreviewed`；不声称 review PASS、planning review PASS、人的接纳或生产批准。
- **未**登记或启动任何实现卡：S4.22 / S4.23 / S4.24 仍为候选、尚未登记为正式 Task；`tasks.md`、`plans/`、`design.md`、测试与源码零改动。
- **未**引入或验证任何锁后端、ACL、双进程互斥、崩溃释放、restore 工具、备份、恢复演练、CAS 产品、独立故障域；所有「已实现/已验证」均未声称。
- **未**改 ADR-0019 / ADR-0027 / ADR-0029 正文；未新开竞争 ADR；未放宽 ADR-0030/0031 的「首期不提供」与 ADR-0032 的 POSIX-only / Windows 退出码 2 / 无 fallback 结论。
- **未**消除任务书「ADR-0019 前 155139 字节零改动」与「修订 ADR-0030/0031」的算术冲突；本轮以 §3 的三条替代不变量证明 ADR-0019…ADR-0029 区间逐字节未变，**需 Lead 确认该口径**。
- **未**处理评审报告中的其他非架构层 finding（若有），它们不在本角色写集内。
