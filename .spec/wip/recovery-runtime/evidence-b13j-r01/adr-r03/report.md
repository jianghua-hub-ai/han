# B13J ADR r03 返工报告（ADR-0030 / ADR-0031 / ADR-0032）

- 角色：add-arch-designer（作者），Agent 文档产出
- 工作流：development-delivery / 架构决策（ADR）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（全程未变）
- 记录时间：2026-11-02（Asia/Shanghai）
- 性质：作者侧文档处置。**不构成**独立评审 PASS、人的接纳、生产批准或实施授权。
- 触发：非作者规划评审 r02 = CHANGES REQUIRED，`.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r02/report.md`（18538 B / `0462EC08AD8BCFECC24AB0BDDC8A9627700DF8D72DEE0441781BF6CFD930C155`）与同目录 `index.json`（16508 B / `D325F724670C6238F724DB5B873AEF8CE6286F09AE8F3FB86738D32F47F8AE83`）。
- 本轮处置：R02-F001（HIGH）、R02-F002（HIGH）、R02-F005（LOW）。评审中 R02-F003/F004 属采用记录与授权原文，R02-F006/F009 属规划登记/`tasks.md` 0x0C，不在架构层写集内，本轮未动。

## 1. 输入与改前基线（实测）

| 对象 | SHA256 | 字节 | 行数 | 格式 |
|---|---|---:|---:|---|
| `.add/project/knowledge/decisions/framework.md`（改前） | `DA77EA1332DB69CB94B90960DEA58B35CABDA76C712EDAE23F2CCF6A1F45B5C1` | 175141 | 1135 | 纯 CRLF（1135/0）/ 无 BOM / 末尾单 CRLF / TAB 0 / 控制字符 0 |
| `adr-r03/snapshots/framework-before-r03.md` | `DA77EA1332DB69CB94B90960DEA58B35CABDA76C712EDAE23F2CCF6A1F45B5C1` | 175141 | 1135 | 逐字节副本 |
| `adr-r03/snapshots/gap-map-before-r03.md` | `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F` | 26452 | — | 只读备份；原件 `gap-map.md` 未改 |
| `planning-review-r02/report.md` | `0462EC08AD8BCFECC24AB0BDDC8A9627700DF8D72DEE0441781BF6CFD930C155` | 18538 | — | 评审输入 |
| `planning-review-r02/index.json` | `D325F724670C6238F724DB5B873AEF8CE6286F09AE8F3FB86738D32F47F8AE83` | 16508 | — | 评审输入 |
| ADR-0019 区间（字节 31305–126598）改前 | `607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6` | 95294 | — | 零改动不变量 |

本轮零改动（实测复核）：`decisions/README.md` `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` / 21472 B；`identity-auth/design.md` `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` / 854991 B；`article-management/design.md` `690015BB5C3B4800539F98C5B75115A5A20B5C161ECE7422F9F271840F7C55A7` / 377424 B。

说明：README 的索引行不含修订号字段，ADR-0030/0031/0032 的 `修订` 字段只存在于 `framework.md`，故本轮 README 无需改动（评审 r02 §4 亦确认）。

## 2. 变更总览

- 唯一改写的产品文件：`.add/project/knowledge/decisions/framework.md`。
- 原地替换 15 行、插入 4 行（`--numstat`：19 insertions / 15 deletions；`--unified=0`：13 个 hunk，相邻改行合并）：行数 1135 → 1139（净 +4）；字节 175141 → 179565（+4424）。
- 全部落在 ADR-0030（:915–:972）、ADR-0031（:974–:1061）、ADR-0032（:1062–:1139）范围内；ADR-0019/ADR-0027/ADR-0029 正文零改动（ADR-0019 区间哈希见 §6）。
- 未改：`decisions/README.md`、`identity-auth/design.md`、`article-management/design.md`、`gap-map.md`、`tasks.md`、`plans/`、源码与测试。
- 未新增/未登记任何 Task：候选 S4.22 / S4.23 / S4.24a / S4.24b 仍全部写作“候选/尚未登记为正式 Task”。
## 3. 逐行改动清单

行号均为实测：`旧` 指改前快照行号，`新` 指改后 framework.md 行号。文本按原样引用。

#### L918 → L918（ADR-0030 修订字段 adr-0030-r02 → adr-0030-r03）

改前：

```text
**目的**: 架构设计 | **仓库**: han | **修订**: adr-0030-r02
```

改后：

```text
**目的**: 架构设计 | **仓库**: han | **修订**: adr-0030-r03
```

#### L953 → L953（ADR-0030 约束 7 定为“生产解除级门禁”，并指向约束 9 的两类卡登记入口）

改前：

```text
7. **解除条件**：只有同时满足以下条件，才可通过新的 ADR 修订或新 ADR 解除“首期不提供”和生产 closed：
```

改后：

```text
7. **解除条件（生产解除级门禁）**：只有同时满足以下条件，才可通过新的 ADR 修订或新 ADR 解除“首期不提供”和生产 closed。本项只约束生产 closed 的解除与生产可用声明，不约束决策卡与实施/验证卡的登记；后两类卡的登记入口见约束 9：
```

#### L960 → L960（R02-F001 核心：约束 9 改为三级无环登记次序（决策卡 / 实施·验证卡 / 生产装配·解除卡））

改前：

```text
9. **后续阶段前置节点（实施门禁）**：任何把生产 CAS provider 或 anchor adapter bundle 付诸实施的卡片——包括规划中尚未登记为正式 Task 的 S4.23 一类候选编号——都不得登记或启动，除非本 ADR 已有新修订给出具体产品、凭据、独立故障域与部署/ACL 结论，并按约束 7 解除 closed。本 ADR 的 r02 只新增该门禁与下述来源关系，不改变“首期不提供”结论、候选取舍、约束 1–6 与解除条件实质。
```

改后：

```text
9. **后续阶段前置节点（三级登记次序，无环）**：按卡片类别区分三个入口，避免把“决策修订”与“约束 7 全项解除”绑成同一前置而形成登记死锁。(a) **决策卡（文档型）**：只产出 ADR 修订/新 ADR 与结论，不含生产实现代码、不改 `cmd/` 或 `internal/`；此类卡不受本门禁限制，可在任何时点登记。(b) **实施/验证卡（含尚未登记为正式 Task 的 S4.23 一类候选编号）**：登记入口是“本 ADR 的决策修订（给出具体产品、凭据来源、独立故障域论证、部署与 ACL 结论）已由架构与安全独立评审接受”，而不是“约束 7 全项解除”；该卡用于产出真实证据。(c) **生产装配/解除卡（含候选 S4.22 一类）**：仍由约束 7 全项（含 `:957` 的真实独立故障域验证与 `:958` 的正式 Task 登记）加人的明确生产批准共同约束；只有它才能解除“首期不提供”与生产 closed，也只有它能作生产可用声明。次序为：决策卡 → 决策修订被独立评审接受 → 实施/验证卡 → 真实证据 → 生产装配/解除卡 → 生产解除；每级只依赖其直接前驱，无回边。本 ADR 的 r03 只改门禁分级与候选编号口径，不改变“首期不提供”结论、候选取舍、约束 1–6 与解除条件实质。
```

#### L961 → L961（约束 10 末段改为与约束 9 同构的次序，来源唯一性与禁止 fallback 全文保留）

改前：

```text
10. **anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源**：生产 `exclusive` 的唯一来源是已登记 anchor adapter bundle 的四端口集合（`Commitments`/`Sources`/`Anchor`/`Exclusive`，缺一即 `ErrAdmissionBinding`）。首期 bundle 不存在，是因为 production root 以空集合构造 registry（`newRecoveryAnchorRegistry(nil)`），未登记解析结果四项一起落 `recoveryClosedPort`，且本 ADR 未提供生产 CAS provider。后续只能在“本 ADR 新修订解除 closed → 同一 bundle 四端口齐备并作为唯一登记来源 → `exclusive` 才可能离开 closed”的可核顺序中获得；不得以 file、SQL、fake 或环境探测自动 fallback，不得用合成/测试注入（`recoveryRootInputs.exclusive` 仅限合成/测试）或 registry 之外的第二来源替代。
```

改后：

```text
10. **anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源**：生产 `exclusive` 的唯一来源是已登记 anchor adapter bundle 的四端口集合（`Commitments`/`Sources`/`Anchor`/`Exclusive`，缺一即 `ErrAdmissionBinding`）。首期 bundle 不存在，是因为 production root 以空集合构造 registry（`newRecoveryAnchorRegistry(nil)`），未登记解析结果四项一起落 `recoveryClosedPort`，且本 ADR 未提供生产 CAS provider。后续只能按与约束 9 同构的次序获得：本 ADR 决策修订（生产 CAS provider、独立故障域、部署/ACL 结论）被架构与安全独立评审接受 → 实施/验证卡登记并使同一 bundle 四端口齐备、作为唯一登记来源 → 真实证据 → 生产解除，`exclusive` 才可能离开 closed；不得以 file、SQL、fake 或环境探测自动 fallback，不得用合成/测试注入（`recoveryRootInputs.exclusive` 仅限合成/测试）或 registry 之外的第二来源替代。
```

#### L969 → L969（R02-F005：r02 追溯日期 2026-11-01 → 2026-11-02（注明决定日期 2026-11-01））

改前：

```text
- 2026-11-01 r02 修订（触发：非作者规划评审 `B13J-PLAN-F004`/`F005`，报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r01/report.md`）：新增约束 9（后续阶段实施门禁）与约束 10（anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源），说明首期 bundle 不存在的原因与后续唯一可核顺序；决定结论、候选取舍、约束 1–6 与解除条件实质未变；未新增实现任务，S4.22/S4.23/S4.24 仍为未登记候选，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

改后：

```text
- 2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）（触发：非作者规划评审 `B13J-PLAN-F004`/`F005`，报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r01/report.md`）：新增约束 9（后续阶段实施门禁）与约束 10（anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源），说明首期 bundle 不存在的原因与后续唯一可核顺序；决定结论、候选取舍、约束 1–6 与解除条件实质未变；未新增实现任务，S4.22/S4.23/S4.24 仍为未登记候选，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

#### 新增 L970（ADR-0030 新增 r03 追溯行）

新增文本：

```text
- 2026-11-02 r03 修订（触发：非作者规划评审 `B13J-PLAN-R02-F001`/`F002`/`F005`，报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r02/report.md`，18538 B / `0462EC08AD8BCFECC24AB0BDDC8A9627700DF8D72DEE0441781BF6CFD930C155`）：把约束 9 改为三级无环登记次序（决策卡 / 实施·验证卡 / 生产装配·解除卡），把约束 7 明确为生产解除级门禁，并按同构次序改写约束 10 的后续路径；只改门禁分级与候选编号口径，不改变“首期不提供”结论、候选取舍、约束 1–6 与解除条件实质；未新增实现任务，候选编号仍为未登记候选，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

#### L976 → L977（ADR-0031 修订字段 adr-0031-r02 → adr-0031-r03）

改前：

```text
**目的**: 架构设计 | **仓库**: han | **修订**: adr-0031-r02
```

改后：

```text
**目的**: 架构设计 | **仓库**: han | **修订**: adr-0031-r03
```

#### L1048 → L1049（R02-F002：ADR-0031 前置节点的候选编号 S4.24 → S4.24a，并写明决策卡不受该门禁限制）

改前：

```text
任何把生产 restore 工具、生产批准主体、专用执行身份或宿主隔离付诸实施的卡片——包括规划中尚未登记为正式 Task 的 S4.24 一类候选编号——都不得登记或启动，除非本 ADR 已有新修订给出批准主体、执行主体、宿主隔离与原生耐久回执的可核结论，并按约束 8 通过架构与安全独立评审。本 ADR 的 r02 只新增该门禁，不改变“首期不提供生产 restore 工具与批准主体”结论、候选取舍、不变量与解除条件实质。
```

改后：

```text
任何把生产 restore 工具、生产批准主体、专用执行身份或宿主隔离付诸实施的卡片——包括规划中尚未登记为正式 Task 的 S4.24a 一类候选编号（生产 restore source/target/tool 与批准主体接线）——都不得登记或启动，除非本 ADR 已有新修订给出批准主体、执行主体、宿主隔离与原生耐久回执的可核结论，并按约束 8 通过架构与安全独立评审。本门禁只约束实施/装配卡，只产出本 ADR 修订的决策卡不受限制；部署独占 POSIX 锁后端与 `exclusive` 接线属候选 S4.24b，受 ADR-0032 门禁，不在本条范围。本 ADR 的 r03 只改候选编号口径与门禁分级，不改变“首期不提供生产 restore 工具与批准主体”结论、候选取舍、不变量与解除条件实质。
```

#### L1057 → L1058（R02-F005：r02 追溯日期更正）

改前：

```text
- 2026-11-01 r02 修订（触发：非作者规划评审 `B13J-PLAN-F004`，报告同上）：新增“后续阶段前置节点”小节，要求任何生产 restore 实现卡（含尚未登记为正式 Task 的 S4.24 一类候选编号）先经本 ADR 新修订给出批准主体、执行主体、宿主隔离与原生耐久回执结论并通过安全独立评审；决定结论、候选取舍、不变量与解除条件实质未变；未新增实现任务，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

改后：

```text
- 2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）（触发：非作者规划评审 `B13J-PLAN-F004`，报告同上）：新增“后续阶段前置节点”小节，要求任何生产 restore 实现卡（含尚未登记为正式 Task 的 S4.24 一类候选编号）先经本 ADR 新修订给出批准主体、执行主体、宿主隔离与原生耐久回执结论并通过安全独立评审；决定结论、候选取舍、不变量与解除条件实质未变；未新增实现任务，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

#### 新增 L1059（ADR-0031 新增 r03 追溯行）

新增文本：

```text
- 2026-11-02 r03 修订（触发：非作者规划评审 `B13J-PLAN-R02-F002`/`F005`，报告同上）：把“后续阶段前置节点”中的候选编号改为 S4.24a（生产 restore source/target/tool 与批准主体接线），并写明本门禁只约束实施/装配卡、决策卡不受限制；部署独占锁接线归候选 S4.24b，受 ADR-0032 门禁。只改候选编号口径与门禁分级，不改变“首期不提供生产 restore 工具与批准主体”结论、不变量与解除条件实质；未新增实现任务，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

#### L1064 → L1066（ADR-0032 修订字段 adr-0032-r02 → adr-0032-r03）

改前：

```text
**目的**: 架构设计 | **仓库**: han | **修订**: adr-0032-r02
```

改后：

```text
**目的**: 架构设计 | **仓库**: han | **修订**: adr-0032-r03
```

#### L1116 → L1118（R02-F001/F002：ADR-0032 解除条件第 8 条首节点改为“ADR-0030 按其约束 9 完成决策修订并被独立评审接受（决策级）”，生产解除仍按约束 7）

改前：

```text
   - ADR-0030 按其约束 7 与约束 9 完成新修订（生产 CAS provider、独立故障域、部署/ACL 结论），解除其 closed；
```

改后：

```text
   - ADR-0030 按其约束 9 完成决策修订（生产 CAS provider、独立故障域、部署/ACL 结论）并被架构与安全独立评审接受（决策级）；生产解除仍按约束 7；
```

#### L1119 → L1121（R02-F002：候选 S4.24 → S4.24b（部署独占 POSIX 锁后端与 exclusive 接线））

改前：

```text
   - 候选 S4.24（`exclusive` 锁接线）**requires 候选 S4.23**：bundle 必须先于锁后端接线，不得先接线后补来源；
```

改后：

```text
   - 候选 S4.24b（部署独占 POSIX 锁后端与 `exclusive` 接线，受本 ADR 门禁）**requires 候选 S4.23**：bundle 必须先于锁后端接线，不得先接线后补来源；
```

#### 新增 L1122（R02-F002：链中补入候选 S4.24a 节点（受 ADR-0031 门禁，与锁后端独立））

新增文本：

```text
   - 候选 S4.24a（生产 restore source/target/tool 与批准主体接线，受 ADR-0031 门禁）与本 ADR 的锁后端节点相互独立，两者均须早于候选 S4.22 的生产解除；其与 S4.22 root 装配的先后按 ADR-0031 单独登记；
```

#### L1120 → L1123（R02-F002：候选 S4.22 requires 候选 S4.24b；与 S4.24a 的关系按 ADR-0031 单独登记）

改前：

```text
   - 候选 S4.22（production root 装配与 recovery secret candidate）必须在候选 S4.24 之后；
```

改后：

```text
   - 候选 S4.22（production root 装配与 recovery secret candidate）**requires 候选 S4.24b**；其与候选 S4.24a 的关系按 ADR-0031 单独登记（root 装配可在 restore 保持 closed 时进行），不并入本 ADR 的硬依赖边；
```

#### L1122 → L1125（无环性说明改为决策级/实施级/生产解除级三段单向，并点明旧口径才是登记顺序不可满足的原因）

改前：

```text
   该链是线性前驱图：每个节点只依赖其直接前驱，无回边，且 ADR-0030 自身已提供“经本 ADR 新修订解除 closed”的出边，因此不是架构死锁；当前不可满足的只是规划登记顺序（在 ADR-0030 修订前登记 S4.23/S4.24 一类实施卡），不是架构不可达，也不改变本 ADR 的 POSIX-only、Windows 生产 unsupported（退出码 2）与无 fallback 结论。
```

改后：

```text
   该链是有向无环图：决策级修订、实施/验证、真实验证、生产解除逐级单向，每个节点只依赖其直接前驱，无回边，且 ADR-0030 约束 9 已给出“决策卡可在任何时点登记、决策修订接受后即可登记实施/验证卡”的出边，因此不是架构死锁；当前不可满足的只是旧口径下的规划登记顺序（把“约束 7 全项解除”误读为实施卡登记前置），不是架构不可达，也不改变本 ADR 的 POSIX-only、Windows 生产 unsupported（退出码 2）与无 fallback 结论。
```

#### L1132 → L1135（行号正确性：相邻决定引用 ADR-0031 的 framework.md 行号 :974 → :975（新实测））

改前：

```text
- 相邻决定：[framework.md（ADR-0030）](framework.md#adr-0030)，`framework.md:916` 起；[framework.md（ADR-0031）](framework.md#adr-0031)，`framework.md:974` 起；本 ADR 不改变二者的“首期不提供”与 closed 结论。
```

改后：

```text
- 相邻决定：[framework.md（ADR-0030）](framework.md#adr-0030)，`framework.md:916` 起；[framework.md（ADR-0031）](framework.md#adr-0031)，`framework.md:975` 起；本 ADR 不改变二者的“首期不提供”与 closed 结论。
```

#### L1135 → L1138（R02-F005：r02 追溯日期更正）

改前：

```text
- 2026-11-01 r02 修订（触发：非作者规划评审 `B13J-PLAN-F005`/`F007`，报告同上）：在“解除条件”新增第 8 条解除顺序链，把不变量与约束第 9 条的顺序要求落成规范性可勾选条件，并明确候选 S4.24 requires 候选 S4.23、候选 S4.22 在其后；说明该链无环、可满足，当前不可满足的只是规划登记顺序。POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、三入口共锁与约束 1–8 的实质均未改变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

改后：

```text
- 2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）（触发：非作者规划评审 `B13J-PLAN-F005`/`F007`，报告同上）：在“解除条件”新增第 8 条解除顺序链，把不变量与约束第 9 条的顺序要求落成规范性可勾选条件，并明确候选 S4.24 requires 候选 S4.23、候选 S4.22 在其后；说明该链无环、可满足，当前不可满足的只是规划登记顺序。POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、三入口共锁与约束 1–8 的实质均未改变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

#### 新增 L1139（ADR-0032 新增 r03 追溯行）

新增文本：

```text
- 2026-11-02 r03 修订（触发：非作者规划评审 `B13J-PLAN-R02-F001`/`F002`/`F005`，报告同上）：解除条件第 8 条改为“决策修订被独立评审接受 → 实施/验证卡（候选 S4.23、S4.24a、S4.24b）→ 真实验证 → 生产解除”的无环节次，原候选 S4.24 改称 S4.24b 并补入 S4.24a 节点；不变量与约束第 9 条的顺序要求已落入规范性解除条件。POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、三入口共锁与约束 1–8 的实质均未改变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
```

## 4. 语义论证

### 4.1 R02-F001（HIGH）：约束 9 的登记死锁

旧文本（改前 `:960`）把两件事写成同一前置：「本 ADR 已有新修订给出具体产品、凭据、独立故障域与部署/ACL 结论」**并且**「按约束 7 解除 closed」。而约束 7 的 `:957` 要求“在真实独立故障域完成 CAS 冲突、stale revision、missing/rollback、pending/committed、崩溃恢复与权限失败验证”，`:958` 要求“由正式 Task 规划登记实现与验证范围”；两者都需要先有实现。实施卡又因旧约束 9 不得先登记，于是形成「卡不能登记 ⇒ 无人实现 ⇒ 永远拿不到证据 ⇒ 永远无法解除」的闭环。这不是“先决策后实施”的正当门禁，而是自锁。

新文本按卡片类别拆分三个入口：

1. **决策卡（文档型）**：只产出 ADR 修订/新 ADR 与结论，不含生产实现代码、不改 `cmd/` 或 `internal/`。此类卡不受本门禁限制，可在任何时点登记。
2. **实施/验证卡（含候选 S4.23 一类）**：登记入口是「本 ADR 的决策修订（具体产品、凭据来源、独立故障域论证、部署与 ACL 结论）已由架构与安全独立评审接受」，**不是**「约束 7 全项解除」。该卡用于产出真实证据。
3. **生产装配/解除卡（含候选 S4.22 一类）**：仍由约束 7 全项（含 `:957` 真实验证与 `:958` 正式 Task 登记）加人的明确生产批准共同约束。只有它能解除「首期不提供」与生产 closed，也只有它能作生产可用声明。

并在约束 7 开头（`:953`）加注：它是**生产解除级**门禁，只约束生产 closed 的解除与生产可用声明，不约束决策卡与实施/验证卡的登记，登记入口见约束 9。这一句是消除误读的关键——旧文本被读成“所有卡都要先解除 closed”，正是死锁的直接来源。

约束 7 的两个子项行号未变：`:957` 仍是真实独立故障域验证、`:958` 仍是正式 Task 登记（约束 7 只在原有行内改写，未增删行）。因此约束 9 中对 `:957`/`:958` 的引用在新文本中仍然成立。

### 4.2 约束 10 一致性（B）

改前 `:961` 的末段写「本 ADR 新修订解除 closed → 同一 bundle 四端口齐备 → `exclusive` 才可能离开 closed」，把“解除 closed”置于四端口齐备之前，与 4.1 的旧死锁同构。改后按同一三级次序表述：决策修订被独立评审接受 → 实施/验证卡登记并使同一 bundle 四端口齐备、作为唯一登记来源 → 真实证据 → 生产解除。

来源唯一性与禁止项**全文保留**：四端口集合 `Commitments`/`Sources`/`Anchor`/`Exclusive` 缺一即 `ErrAdmissionBinding`；registry 是唯一登记来源；首期因 `newRecoveryAnchorRegistry(nil)` 空集合构造、四项一起落 `recoveryClosedPort`；禁止 file/SQL/fake/环境探测自动 fallback；禁止合成/测试注入（`recoveryRootInputs.exclusive` 仅限合成/测试）与 registry 之外的第二来源。本轮未削弱任何一条。

### 4.3 R02-F002（HIGH）：候选编号 canonical 化

评审指出的三方冲突：ADR-0031 `:1048` 把 S4.24 当 restore 实施卡；ADR-0032 `:1119`–`:1120` 把 S4.24 当 `exclusive` 锁接线（requires S4.23）且置于 S4.22 之前；`gap-map.md:182` 把两者合并。同一编号无法同时满足两套门禁。

本轮按任务书给定的 canonical 编号统一（`gap-map.md` 由规划角色另行同步，本报告不写其行号结论、也未改该文件）：

- `S4.24a`：生产 restore source/target/tool 与批准主体接线，受 ADR-0031 门禁，与锁后端节点独立。
- `S4.24b`：部署独占 POSIX 锁后端与 `exclusive` 接线，受 ADR-0032 门禁，requires 候选 S4.23。
- `S4.22`：production root 装配 + recovery secret candidate，requires 候选 S4.24b；与 S4.24a 的先后按 ADR-0031 单独登记（root 装配可在 restore 保持 closed 时进行），不并入 ADR-0032 的硬依赖边。

落点：ADR-0031 `:1049` 改为 S4.24a 并写明“本门禁只约束实施/装配卡，只产出本 ADR 修订的决策卡不受限制”；ADR-0032 解除条件第 8 条 `:1121`–`:1123` 改为 S4.24b + 新增 S4.24a 节点 + S4.22 requires S4.24b；`:1118` 的首节点由“按其约束 7 与约束 9 完成新修订…解除其 closed”改为“按其约束 9 完成决策修订并被架构与安全独立评审接受（决策级）；生产解除仍按约束 7”。

未削弱：ADR-0032 的 POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、三入口共锁、约束 1–8 全部保持逐字节不变（仅 `:1118`/`:1121`/`:1122`/`:1123`/`:1125` 属声明改动，约束 1–7 与不变量正文未动）。

### 4.4 R02-F005（LOW）：修订日期 errata

三处 r02 追溯行（改前 `:969`、`:1057`、`:1135`）由「2026-11-01 r02 修订」改为「2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）」，保留决定日期以免与 ADR 标题的 `(2026-11-01)` 冲突。历史的 `adr-r02/report.md` 与 `adr-r02/index.json` **未改**（不可变证据），errata 记录在本报告 §9。

## 5. 无环性论证

改后节点与边（全部为“候选”，未登记为正式 Task）：

| 节点 | 含义 | 直接前驱 |
|---|---|---|
| D0 | 决策卡（文档型，任何时点可登记） | 无 |
| D1 | ADR-0030 决策修订被架构与安全独立评审接受（决策级） | D0 |
| I1 | 候选 S4.23：anchor adapter bundle 四端口齐备并登记 | D1 |
| I2 | 候选 S4.24b：POSIX 锁后端与 `exclusive` 接线 | I1（requires S4.23） |
| I3 | 候选 S4.24a：restore source/target/tool 与批准主体接线 | ADR-0031 门禁（决策级修订被接受）；与 I2 无边 |
| P1 | 候选 S4.22：production root 装配 + recovery secret candidate | I2（requires S4.24b）；与 I3 的先后按 ADR-0031 单独登记 |
| R | 生产解除（约束 7 全项 + 人的明确生产批准） | I1、I2、I3、P1 与正式登记 |

- **无回边**：所有边由“决策级”指向“实施级”、再由“实施级”指向“生产解除级”，不存在从生产解除反向指回决策或实施登记的边。
- **旧文本的环**：I1 的登记需要 R 的前置（约束 7 全项），而 R 需要 I1 的证据。新文本把 I1/I2/I3 的入口改为 D1，该环被断开。
- **I3 与 I2 独立**：两者都受各自 ADR 的门禁，互不 requires；I3 与 P1 的先后由 ADR-0031 单独登记（root 装配可在 restore 保持 closed 时进行），因此不引入新的交叉硬边。
- 结论：链是**有向无环图**，架构上可达；当前不可满足的只是旧口径下的规划登记顺序（把“约束 7 全项解除”误读为实施卡登记前置），不是架构死锁。ADR-0032 的 POSIX-only / Windows 生产 unsupported（退出码 2）/ 无 fallback 结论不受影响。

## 6. 格式与哈希实测（改后）

| 对象 | SHA256 | 字节 | 行数 | crlf / bareLF | 其它 |
|---|---|---:|---:|---|---|
| `framework.md`（改后） | `9B9A0D7A62DAE9B62F4CF292E9B85888903352E39D06C70B04D303C40D6D53B8` | 179565 | 1139 | 1139 / 0 | 无 BOM；末尾单 CRLF；bareCR 0；TAB 0；控制字符（<0x20 且非 CR/LF/TAB）0 |

- 公共前缀：改前/改后首异字节偏移 = **126770**；`[0..126769]`（126770 B）改前改后 SHA256 均为 `0D80C9D34B5E16735AA9E6B0BBB8406398B37DBF8A11A99557173E6F32DA940F`。
- ADR-0019 区间（字节 31305–126598，95294 B）：改前/改后均为 `607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6`。
- 说明：不使用被评审标记为历史错误口径的“前 155139 字节”表述；本轮以前缀哈希 + ADR-0019 区间哈希 + ADR-0019/0027/0029 正文未改作为等价证明。

## 7. 反向还原验证

方法：把改后文件读入行数组，依次

1. 按精确文本删除 4 条新增行（新 `:970`、`:1059`、`:1122`、`:1139`）；
2. 把三处「2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）」还原为「2026-11-01 r02 修订」；
3. 把三处 `**修订**: adr-00xx-r03` 还原为 `-r02`；
4. 与快照按同索引逐行比对，实测剩余差异行恰为 `953,960,961,1048,1116,1119,1120,1122,1132`（即 §3 中剩余 9 条原地替换），逐行回填快照原文；
5. 以 CRLF 重连并补末尾单 CRLF，写入临时文件测量。

结果：`DA77EA1332DB69CB94B90960DEA58B35CABDA76C712EDAE23F2CCF6A1F45B5C1` / 175141 B / 1135 行，与改前快照逐字节一致。

## 8. 验收命令与结果（实跑）

```text
rg -n "部署锁|文件锁|Windows|POSIX|ADR-003" .add/project/knowledge/decisions/framework.md   -> exit 0（命中既有行，如 ADR-0019 :471/:472）
rg -n "^## ADR-003[012]" .add/project/knowledge/decisions/framework.md                        -> exit 0；命中 916 / 975 / 1064
rg -n "adr-003[012]-r03" .add/project/knowledge/decisions/framework.md                        -> exit 0；命中 918 / 977 / 1066
rg -n "S4\.24[ab]?|S4\.2[23]" .add/project/knowledge/decisions/framework.md                   -> exit 0；命中 960 / 969 / 1049 / 1058 / 1059 / 1119 / 1121 / 1122 / 1123 / 1138 / 1139
git diff --check -- .add/project/knowledge/decisions/framework.md                             -> exit 0（无 whitespace 错误）
git diff --no-index --unified=0 <snapshot> <framework.md>                                     -> exit 1（有差异，预期）；13 个 hunk 覆盖 §3 的 19 条改动（相邻改行合并），无其他 hunk
git status --porcelain                                                                        -> M framework.md（README.md 的 M 属前几轮既有改动，本轮未触碰，哈希仍 2EE124AE…/21472）
git rev-parse HEAD                                                                            -> 6da8abbceb0fa197364810a92a8fa2d97b38ddca
ConvertFrom-Json adr-r03/index.json                                                           -> OK
```

`git diff --no-index --numstat` 实测 **19 insertions / 15 deletions**，与 §2 的“15 行原地替换 + 4 行插入”一致（每处替换计 1 删除 + 1 插入）。`--unified=0` 实测 **13 个 hunk**：git 会把相邻改行并入同一 hunk，故 960/961 与 1119/1120 各合并为一个 hunk。hunk 头（旧→新）为：918→918、953→953、960–961→960–961、969→969–970、976→977、1048→1049、1057→1057–1058、1064→1066、1116→1118、1119–1120→1121–1123、1122→1125、1132→1135、1135→1135–1138。逐行粒度仍是 §3 的 19 条（15 改 + 4 增），无其他 hunk。

关于 `:1132 → :1135` 一行：因 ADR-0031 标题随插入前移到 `:975`，该行引用 `framework.md:974` 已失真，故按实测改为 `framework.md:975`。这属“引用记录必须写改动后的真实行号”的必要修正，不改变任何决定语义。

## 9. R02-F005 errata 记录

- 事实：`adr-r02/report.md` 与 `adr-r02/index.json` 记录时间为 2026-11-01，而触发它们的规划评审 r01 报告日期为 2026-11-02；`framework.md` 内三处 r02 追溯行同样写 2026-11-01。
- 处置：仅在 `framework.md` 内把三处 r02 追溯行改为「2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）」；`adr-r02/` 下的报告与索引保持原样（不可变证据，不追改历史）。
- 影响：读者可从 r03 追溯行与本 errata 还原先后关系；ADR 决定日期仍为 2026-11-01，与标题一致。

## 10. 未覆盖边界与声明

- 本报告是**作者侧文档处置**：不构成非作者评审 PASS、人的接纳、安全评审通过、生产批准或实施授权；不得据此宣称 ADR-0030/0031/0032 已通过 r03 复审。
- 未登记任何 Task：S4.22 / S4.23 / S4.24a / S4.24b 仍为候选，`tasks.md` 与 Plan 未改。
- 未运行 Go 测试、PostgreSQL、服务、E2E、restore、锁后端或任何真实环境验证；未证明任何 provider、工具、锁后端、ACL、双进程互斥、崩溃释放或独立故障域。
- 未修改 `gap-map.md`（S4.24a/b 的 gap-map 同步由规划角色执行）；本轮未改 `decisions/README.md`（其索引行不含修订号字段）。
- 未处理评审 r02 的 R02-F003/F004（采用记录行锚与 standing 授权原文）与 R02-F006/F009（规划登记与 `tasks.md` 0x0C），均不在架构层写集内；评审判定 F003/F006/F009 仍 OPEN，本轮不构成其关闭证据。
- 生产仍 closed：ADR-0030「首期不提供生产 CAS provider」与 ADR-0031「首期不提供生产 restore 工具与批准主体」结论逐字节未变；ADR-0032 POSIX-only / Windows 生产 unsupported（退出码 2）/ 无 fallback 未变；`blocked-upstream` 未解除；Case18 与 S8.2 未声称 PASS。
