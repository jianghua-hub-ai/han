# B13J Planning Review r01

- 评审角色：add-reviewer（非作者独立评审）
- 对象类型：planning（B13J 批次可登记性 + 候选 Task 卡可实施性）
- 被评对象：`pending-B13J`；`evidence-b13j-r01/gap-map.md`；D01–D04 设计修订链；当前落地设计/ADR；Plan 与 tasks 现状
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（评审期间未变）
- 评审日期：2026-11-02（Asia/Shanghai）
- 独立性：D01–D04 的设计作者为 Singer，D04 design-review 作者为 Sagan；本评审未参与上述产出，不默认接受其结论，独立核对原始制品与字节。
- 最终结论：**CHANGES REQUIRED**
- 一句话理由：D01–D04 的复审链和证据绑定基本成立，诚实性也未被破坏；但 U1–U9 仍无统一确认记录、候选 Task 卡未达到可登记/可实施粒度、ADR-0030/0031/0032 的前置修订节点缺失，且两条 D04 LOW 无承接处。现在登记 B13J Plan 批次会制造假引用，不能判 PASS。

> 边界声明：本评审只判“是否可登记 B13J 批次与候选 Task 卡”。不构成实施授权，不关闭 `pending-B13J-D04`，不解除 `blocked-upstream`，不视为人的签收，也不替代部署、Git 或生产批准。

## 1. 两轮实际检查范围

- 第一轮（对象与证据链）：逐份读取 `gap-map.md`、D01–D04 的作者报告、全部 review 报告及旁置 `index.json`；核对 SHA256/字节、JSON 合法性、verdict 行锚、历史 FAIL 保留、ADR 正文/索引、落地设计诚实性、Plan/tasks 当前状态。
- 第二轮（独立复核与规划判断）：沿真实消费者（identity-auth design、article-management design、`framework.md`、`decisions/README.md`、`tasks.md`、Plan）复核候选边界、U1–U9 前置、ADR 耦合链、D04 两条 OPEN LOW 的承接、批次/任务登记顺序、越界与格式不变量；对第一轮结论重新判断，不以作者或 D04 reviewer 的“有意显式写出”作为通过依据。
- 实际读取的是当前工作树字节，不是聊天摘要；未运行 Go、PostgreSQL、HTTP、服务、restore 或 E2E。

## 2. 基线与证据绑定

### 2.1 现场哈希/字节（与任务给定基线一致）

| 文件 | 字节 | SHA256 |
|---|---:|---|
| `.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md` | 26452 | `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | 854991 | `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` |
| `.add/project/knowledge/design-workspace/article-management/design.md` | 377424 | `690015BB5C3B4800539F98C5B75115A5A20B5C161ECE7422F9F271840F7C55A7` |
| `.add/project/knowledge/decisions/framework.md` | 169890 | `58C97FC06361022AA48EF737189EF3EB069A7AAD25772BCA4C80FE4D1D5E1F8E` |
| `.add/project/knowledge/decisions/README.md` | 21472 | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | 497542 | `3FA19CDD4902F56B8C2F1DF59C570D96FAAE02D8AEE519B4B4165DCCA3637380` |
| `.add/project/knowledge/tasks/tasks.md` | 823187 | `F1D65B0D7016CDBF9C2F4E85B012B5B0B9F714B6DF0C7CDC9326C05676900E34` |
| `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d04/review-r02/report.md` | 19858 | `ECE706E0B30A3569B21B37C353F76A9A1546E5EC73841B029CCE31D9BF9BE876` |

### 2.2 复审链（verdict 行锚与报告哈希均现场核对）

| 修订 | review 报告 | verdict 行锚 | verdict | 哈希 / 字节 |
|---|---|---|---:|---|
| D01 | `design-revision-r01/review-r01/report.md` | `:214` | PASS WITH MINOR | `977022446B3B8ECD3DA03DC64C86A3FE911F16F8DF5AB78DB9F7F734E170DEFE` / 17113 |
| D02 | `design-d02/review-r01/report.md` | `:148` | CHANGES REQUIRED | `DAC90356F102F5E7AF18BFCDAFA153AD369D1F796923B5A42B8B6BC79606FF5E` / 18497 |
| D02 | `design-d02/review-r02/report.md` | `:82` | PASS | `957CACC5280E1287D519FB775A76CE602F98550B92DEB711023742651E84DC0B` / 8406 |
| D03 | `design-d03/review-r01/report.md` | `:147` | CHANGES REQUIRED | `EEC401E4613EFFC1AA4DD89639A32FA04C71C826F7874C32880224819655EBE3` / 25493 |
| D03 | `design-d03/review-r02/report.md` | `:147` | CHANGES REQUIRED | `7F357D6AE3C5E23FD9A19EFA808D1F86C6986668A34CC0E4B811A3A0742D18C6` / 15997 |
| D03 | `design-d03/review-r03/report.md` | `:112` | PASS | `37D4E811D252D8F584970F0B40B75403C0CF75015D67BBA66FFE89C14408A79A` / 16101 |
| D04 | `design-d04/review-r01/report.md` | `:149` | CHANGES REQUIRED | `51C91D48B55486375166267E0B1CED9CE2AFEB3878C94D79CB24427AF820751D` / 26602 |
| D04 | `design-d04/review-r02/report.md` | `:149`、`:153`、`:157` | PASS WITH MINOR | `ECE706E0B30A3569B21B37C353F76A9A1546E5EC73841B029CCE31D9BF9BE876` / 19858 |

- 13 个 `index.json` 以 UTF-8 显式读取后全部可解析（13/13）；8 个 review 报告的 artifact 哈希/字节与索引声明 8/8 一致；顶层 gap-map 绑定 1/1 一致。
- 作者报告绑定：D02 `report.md` `16A47B381CB5BD023EA931432CE9A3CEA622125CB4CCA9641DE9EDDDC293CE7F` / 20671；D03 `27AC38D34E4A814B97F9056FE203BEB40E7AD96DC1E874F87B960F5C2BA69685` / 108987；D04 `FDAEE534E9EFE65D659877984DDCF9F7AE22DE4BD3128A5257C2EF17CCAD3ACB` / 39426，均与各自索引一致。D01 作者报告未在 D01 索引的顶层 `artifact` 中绑定，但 `review-r01/index.json` 的 inputs 明确绑定 `565D1547F8B75A4E60EB8073AA52326E3C8ADB620F71E2D999E60C40506BE89A` / 10548，现场哈希一致。
- 历史 FAIL 均保留在原 review 报告中，后续 r02/r03 PASS 是新增报告而非改写旧报告；D02 r01、D03 r01/r02、D04 r01 的 CHANGES REQUIRED 行锚均真实存在。

## 3. Q1–Q8 逐项回答

### Q1. D01–D04 是否确已全部完成

- **复审链完成**：D01 为 PASS WITH MINOR；D02 为 CHANGES REQUIRED→PASS；D03 为 CHANGES REQUIRED→CHANGES REQUIRED→PASS；D04 为 CHANGES REQUIRED→PASS WITH MINOR。每轮 verdict 行锚真实存在，报告哈希/字节与索引一致，历史 FAIL 未被改写。
- **“全部完成”不等于“全 finding/门禁闭合”**：D04 r02 仍明确有 `B13J-D04-RV2-01`、`B13J-D04-RV2-02` 两条 LOW 为 OPEN；D01 的 `B13J-D01-RV-01` 没有正式 CLOSED 记录。D01 当前设计已到 r60，`:5` 为 `identity-auth-design-ir01-r60`、`:7` 为 `2026-11-01`、`:38` 同步 r60，`:83` 记录 r57→r58→r59→r60 链；当前版本歧义已消除，但原 finding 建议的“为 r57 内容 `504966EA…` 单独追加追溯行”没有以独立 CLOSED 条目收口。
- **结论**：评审链本身完成且证据可核；但“全 finding 闭合”与“作者元数据已反映最终状态”均不成立，见 F010。

### Q2. ADR 状态

- `framework.md` 中 ADR-0030 锚点 `:915`、标题 `:916`；ADR-0031 锚点 `:970`、标题 `:971`；ADR-0032 锚点 `:1053`、标题 `:1054`。每个 ADR 标题在 `framework.md` 中恰好一次。
- `decisions/README.md` 的全局索引 `:38-40`、域表 `:46`、场景查询 `:71-73` 均登记三者，状态均为“已接受”。`rg` 在 `decisions/` 的其他文档中未发现 ADR-0030/31/32 的第二个权威正文（exit 1）。
- 与 ADR-0019 `:471` 的本地 POSIX 锁语义、ADR-0027 `:760`、ADR-0029 `:862` 无直接冲突或相互覆盖；ADR-0032 明确以 ADR-0019/0027 为上游。未发现第二套权威。
- **但**：候选 S4.23/S4.24 的“生产实现”与 ADR-0030 `:926`/`:939`/`:951`（首期不提供生产 CAS provider、S4.5 未交付、生产 enter/install closed）及 ADR-0031 `:981`（首期不提供生产 restore 工具/批准主体）冲突，除非先完成 ADR 修订或新增 ADR。当前规划没有这个前置节点，见 F004。

### Q3. 诚实性

- identity-auth design `:3616` 明确生产 `exclusive` 只能是 `recoveryClosedPort`，`recoveryRootInputs.exclusive` 仅为合成/测试注入；`:3908` 固定锁合同；`:3910` 明确无自动 fake/file/SQL fallback；`:3916` 明确 recovery secret 只读且由 `Journal.VerifyBeginSecret` 验密；`:3922` 明确 Windows 生产 unsupported、当前仍是 closed 占位。
- `tasks.md:3034` 明确 production `Enter`/`Install` 仍为 `ErrAdmissionClosed`；`:1867` 的 `article-management-S8.2` 仍为 `[-]`；`:1872`、`:3027` 明确 Case18/ S8.2 仍 blocked-upstream。
- 未发现“生产可用”“生产开流已完成”等正向宣称；诚实性通过。本结论不证明生产链可达，也不解除任何 blocked 状态。

### Q4. B13J 批次是否可登记

- **新建 B13J 的方向成立**：B13H 已在 Plan `:1610`/`:1611` 关闭；B13I 是联合软件验证（`:1613-1620`）。生产装配不应挂已闭 B13H，也不应挂验证批 B13I。
- **登记顺序正确关系**：(1) 先取得 U1–U9 的明确确认（或明确缩小范围）；(2) 满足 ADR 前置（ADR-0030/0031/0032 的修订/新增决策）；(3) 登记真实 Task 卡（S4.22/S4.23/S4.24 及其依赖）；(4) 再做 Planning Review；(5) 最后才在 Plan 外层表登记 B13J 行。Plan 的“原任务引用”列只能引用已登记 Task。
- **“Plan 引用未登记候选”不可接受**：若 Plan 先登记 B13J 并在引用列写 S4.22/S4.23/S4.24，会形成死链接/假引用，并使 DAG、计数、所有权表看似已有任务。若必须提及候选，只能以“未登记候选，不进入 DAG/计数/引用列”的说明性文字出现，且不得使用可点击 Task 链接。见 F001。

### Q5. 候选边界是否可实施级清晰

- 不足。gap-map `:180-182` 只有一行候选形状：S4.22“唯一 root 装配 + secret candidate”、S4.23“anchor/CAS adapter 生产实现”、S4.24“restore source/target/tool + 部署独占 OS 后端”。这些描述不足以形成可复核 Task 卡。
- 缺少的必要字段至少包括：精确文件清单/写集（含新增文件、共享文件与串行约束）；可执行验收命令及期望退出码/输出；关闭条件；硬/软 `requires` 与 DAG 边；负向测试与失败 oracle；证据落点、责任人与“不宣称”边界；停止/回滚条件。
- S4.22 还需写死 11 端口装配清单、secret 只读/`Journal.VerifyBeginSecret` 验密 oracle，以及“生产不得设置 `recoveryRootInputs.exclusive`”的负向验收。
- S4.23 受 ADR-0030 直接约束；S4.24 同时跨 ADR-0031（restore）与 ADR-0032（锁）两个域，应拆分或至少给出独立子范围；若 S4.24 含 `exclusive`，必须 `requires S4.23`。见 F003、F005。

### Q6. 未闭合 LOW 的承接

- `B13J-D04-RV2-01`（D04 r02 `:133-138`；`recovery_runtime.go:565-567` 对 `inputs.exclusive` 的无条件覆盖、无来源/构建校验）**必须并入 S4.22 负向验收**：生产装配不得设置 `recoveryRootInputs.exclusive`，或用显式“仅测试构建”开关隔离覆盖路径，并给出可失败的负向测试。当前没有承接卡，构成遗漏。见 F006。
- `B13J-D04-RV2-02`（D04 r02 `:140-145`；`framework:1096` 顺序约束未进 `:1100-1106` 解除条件清单）**必须有明确收口点**：在 ADR-0032 修订中把“经 registry 的唯一来源接线并经批准记录可核”并入解除条件清单（或明确作为条件 5 的子项），或登记专门的 ADR 修订任务。当前没有收口任务/ADR 修订点，构成遗漏。见 F007。

### Q7. 耦合链可满足性

- 不是逻辑上的永久死锁，但按当前规划图是**不可满足**的：ADR-0030 `:926`/`:951` 首期 registry 恒空、生产 CAS provider 不提供；设计 `:3616` 因此生产 `exclusive` 只能是 `recoveryClosedPort`；ADR-0032 `:1096` 又要求先有已登记的 anchor adapter bundle，再实现 POSIX 锁后端并取得 ACL/负向证据。没有 ADR-0030 的修订/新增决策，S4.23 不能合法产出 bundle，S4.24 也就无法把 `exclusive` 接到非 closed 来源。
- 可核的独立满足顺序应是：ADR-0030 修订/新增 ADR（批准生产 CAS provider 与独立故障域）→ S4.23 adapter bundle（四端口齐备、注册）→ ADR-0032 POSIX 锁后端 + ACL/负向证据 → S4.24 锁接线 → S4.22 root 装配/secret candidate → 联合软件验证与真实 E2E（另立）。
- 当前 DAG 缺少 ADR 前置节点和 `S4.24 requires S4.23` 等边；因此“锁来源只能来自 adapter bundle”本身不是错，但在当前规划里构成不可达解除路径。见 F005。

### Q8. 越界与完整性

- `tasks.md` 与 Plan 的现场哈希/字节与任务给定基线完全一致，本轮未修改。
- `rg -n 'S4\.2[234]' tasks.md Plan` → exit 1，无命中；`rg -n 'B13J' tasks.md Plan` → exit 1，无命中。候选 Task 尚未登记，Plan 也尚无 B13J 行。
- 评审开始时 `planning-review-r01` 不存在；`evidence-b13j-r01` 下只有 gap-map、D01/D02/D03/D04 预期目录。`.spec` 下未发现本轮新增的越界产物；本轮只写入本目录的 `report.md` 与 `index.json`。

## 4. Findings（按严重度排序）

### B13J-PLAN-F001 — HIGH — OPEN

- **对象/修订**：planning；gap-map 当前修订 + Plan 当前修订。
- **位置**：`gap-map.md:178`、`:207`；Plan `:1300-1310`。
- **违反依据**：gap-map 自述 S4.22/23/24 只是候选编号形状，“不是已登记 Task，不进入当前 DAG、不预占写权”（`:178`）；U 表明确“未确认前不得据此登记 Task/Plan”（`:207`）。Plan 引用列当前全部指向已登记 Task。
- **触发场景**：现在新增 B13J 外层行，并在“原任务引用”列写 S4.22/S4.23/S4.24。
- **影响**：产生死链接/假引用；DAG、任务计数、所有权表看似已有任务，实际没有；下游可能据此派发或计算依赖。
- **修复建议**：按 Q4 的顺序先确认 U1–U9/ADR 前置，再登记真实 Task 卡，再做 Planning Review，最后登记 Plan 行。候选编号若必须出现，只能标注“未登记候选，不进入 DAG/计数/引用列”，且不得链接。
- **证据**：`rg` 无命中（exit 1）；gap-map 原文；Plan 外层表结构。
- **复审条件**：Plan B13J 行的引用列只含已登记 Task；候选说明有明确“未登记”标签且不参与计数/DAG。

### B13J-PLAN-F002 — HIGH — OPEN

- **对象/修订**：planning；gap-map §4 U1–U9。
- **位置**：`gap-map.md:191`、`:195-207`；`design-revision-r01/report.md:148`；`design-revision-r01/review-r01/report.md:169`。
- **违反依据**：gap-map 明确“以上均为建议值，不是已取得的用户确认；未确认前不得据此登记 Task/Plan”（`:207`）；D01 作者报告仍写“本轮按 gap-map 建议值书写”；D01 review 也记录 U2 未确认。
- **触发场景**：以 U1–U9 的建议值直接登记 B13J 或 Task 卡。
- **影响**：批次命名、是否停止拆批、是否重开 S4.16、root 装配/锁/secret 范围等关键选择被当作已确认；后续返工风险高，且会产生“用户已确认”的假声明。
- **修复建议**：建立一份 dated U1–U9 确认记录（逐项：决定、依据、责任人/来源、未确认项），或明确缩小本次只登记已确认子集；未确认项不得进入 Task/Plan。
- **证据**：gap-map `:207`；`rg` 未发现统一确认记录，只发现建议值与“按建议值书写”的说明。
- **复审条件**：U1–U9 每项有确认/明确延期记录；登记内容与确认范围逐项对应。

### B13J-PLAN-F003 — HIGH — OPEN

- **对象/修订**：planning；候选 S4.22/S4.23/S4.24。
- **位置**：`gap-map.md:178-182`；`tasks.md`/Plan（无登记，`rg` exit 1）。
- **违反依据**：候选仅一行形状描述，缺少可复核 Task 卡的必要字段；gap-map 自述“不分配 Task、不写所有权表”（`:191`）。
- **触发场景**：Planner 直接把候选形状转成 Task 卡，或实施者按一行描述开工。
- **影响**：无法确定写集/文件所有权、验收命令、关闭条件与依赖；容易出现共享文件冲突、漏掉负向测试、无法判断关闭。
- **修复建议**：为每个候选补：精确文件清单/写集、硬/软 `requires`、验收命令+期望退出码、负向测试/失败 oracle、关闭条件、证据落点/责任人/不宣称边界、停止条件。S4.24 至少拆成 restore 与部署锁两个子范围，或给出可独立验收的子项。
- **证据**：gap-map `:180-182`；Plan/tasks 无候选登记。
- **复审条件**：三张 Task 卡均具备上述字段，且能通过一次 Planning Review 的边界/验收检查。

### B13J-PLAN-F004 — HIGH — OPEN

- **对象/修订**：planning + ADR-0030/0031。
- **位置**：`framework.md:926`、`:939`、`:951`、`:981`；`gap-map.md:180-182`。
- **违反依据**：ADR-0030 首期不提供生产 CAS provider、S4.5 未交付、生产 enter/install closed；ADR-0031 首期不提供生产 restore 工具/批准主体、生产 enter/resume 的 DB 替换 closed。候选 S4.23/S4.24 是生产实现，当前规划没有 ADR 修订/新增决策前置。
- **触发场景**：把 S4.23/S4.24 作为普通实施卡登记并派发。
- **影响**：计划与已接受 ADR 冲突；实施者要么违反 ADR，要么在启动时停止；ADR 解除条件也无法被规划满足。
- **修复建议**：在 S4.23/S4.24 之前增加 ADR 前置节点：修订 ADR-0030（生产 CAS provider/独立故障域）与必要的 ADR-0031（restore 生产形态），或新增 ADR；Task 卡以 `requires` 指向该决策；决策未接受前不登记生产实现。
- **证据**：framework 原文；README `:38-40` 均标“已接受”；无 ADR 修订任务登记。
- **复审条件**：ADR 修订/新增决策已接受并与 Task 卡 `requires` 连接；S4.23/S4.24 不再与“首期不提供”冲突。

### B13J-PLAN-F005 — HIGH — OPEN

- **对象/修订**：planning；ADR-0032 + S4.24/S4.22 依赖图。
- **位置**：`framework.md:1096`、`:1100-1106`；identity design `:3616`；`gap-map.md:167-175`、`:182`。
- **违反依据**：ADR-0032 要求先有已登记 anchor adapter bundle，再实现 POSIX 锁后端并取得隔离/ACL 证据；设计明确生产 `exclusive` 只能来自 bundle，否则 `recoveryClosedPort`。当前候选 S4.24 没有 `requires S4.23`，DAG 也没有 ADR-0030 修订节点。
- **触发场景**：按当前 DAG 直接实施 S4.24 锁后端，或认为 ADR-0032 解除条件可在没有 bundle 的情况下满足。
- **影响**：`exclusive` 无法从 closed 变为非 closed；解除路径不可达；若以合成/测试注入替代，违反唯一来源约束。
- **修复建议**：补 Q7 的顺序与边：ADR-0030 修订 → S4.23 bundle → ADR-0032 锁后端/证据 → S4.24 锁接线 → S4.22 root 装配 → 联合验证/E2E；S4.24 明确 `requires S4.23` 与 ADR 修订；禁止 registry 之外的锁来源。
- **证据**：framework `:1096`；gap-map DAG `:167-175`；候选 `:182`；无 `requires` 登记。
- **复审条件**：DAG 含上述节点/边，Task 卡 `requires` 可核，且不存在独立锁来源。

### B13J-PLAN-F006 — MEDIUM — OPEN

- **对象/修订**：D04 r02 finding `B13J-D04-RV2-01` 的承接。
- **位置**：`design-d04/review-r02/report.md:133-138`；`cmd/han-server/recovery_runtime.go:565-567`；`gap-map.md:180`。
- **违反依据**：review 明确建议在 S4.22 验收中加负向要求；当前没有 S4.22 Task 卡，也没有其他承接处。
- **触发场景**：S4.22 实施时直接设置 `recoveryRootInputs.exclusive`，或没有负向测试。
- **影响**：绕过唯一 registry 来源；代码不会失败；设计约束只停留在文本。
- **修复建议**：S4.22 关闭条件加入：生产装配不得设置 `recoveryRootInputs.exclusive`，或以显式“仅测试构建”开关隔离；提供可失败的负向测试并记录证据。
- **证据**：D04 r02 `:133-138`；代码路径；`rg` 未见承接卡。
- **复审条件**：S4.22 卡面含该负向验收，测试可失败并在关闭时留存结果。

### B13J-PLAN-F007 — MEDIUM — OPEN

- **对象/修订**：D04 r02 finding `B13J-D04-RV2-02` 的承接。
- **位置**：`design-d04/review-r02/report.md:140-145`；`framework.md:1096`、`:1100-1106`。
- **违反依据**：顺序约束只在“不变量与约束”第 9 条，解除条件 1–7 未包含“经 registry 唯一来源接线并可核”；当前无 ADR 修订任务或收口条目。
- **触发场景**：后续按解除条件 1–4 判 ADR-0032 可解除，但 `exclusive` 仍未接线。
- **影响**：解除条件清单可被“满足”而生产仍 closed；规划漏项。
- **修复建议**：ADR-0032 修订把该条并入解除条件清单（或条件 5 子项），或登记专门 ADR 修订任务；与 S4.22/S4.24 关闭条件互链。
- **证据**：D04 r02 `:140-145`；framework `:1096`/`:1100-1106`。
- **复审条件**：ADR-0032 修订已接受或专门任务已登记；解除条件清单含该可勾选项。

### B13J-PLAN-F008 — LOW — OPEN

- **对象/修订**：`design-d04/review-r02/index.json` 的格式声明。
- **位置**：`design-d04/review-r02/index.json` 的 `artifact.trailing`；`design-d04/review-r02/report.md` 末尾字节。
- **违反依据**：索引声明 `"trailing": "single LF (0x0a)"`；实测报告最后 8 字节为 `89 B9 E5 87 86 E3 80 82`，末尾是“。”的 UTF-8 字节，不是换行；报告有 207 个 LF 终止符 / 208 逻辑行。
- **触发场景**：自动化按索引声明检查末尾换行。
- **影响**：证据元数据不准确；不影响报告语义或 verdict。
- **修复建议**：在获授权的元数据修订中改为“no trailing newline”或实际值；本评审不修改被评对象。
- **证据**：末尾字节实测；`index.json` artifact 字段。
- **复审条件**：索引与实际字节一致，或明确记录为已知差异。

### B13J-PLAN-F009 — LOW — OPEN

- **对象/修订**：`tasks.md` 既有格式。
- **位置**：`tasks.md:18`；字节偏移 `12676`、`12691`。
- **违反依据**：两个 `0x0C`（form feed）控制字符；从上下文看是把 `test/`、`fixtures`、`defaults` 等词切断的既有损坏，不是本轮引入。
- **触发场景**：文本处理、diff、检索或渲染工具按控制字符处理。
- **影响**：轻微可读性/工具兼容问题；不影响本规划结论。
- **修复建议**：另行取得授权后替换为原词并复核哈希；本评审不修改 `tasks.md`。
- **证据**：字节扫描 `controls=2`，偏移如上。
- **复审条件**：授权修复完成或明确接受为已知限制。

### B13J-PLAN-F010 — LOW — OPEN

- **对象/修订**：D01–D04 作者 `index.json` 的状态字段。
- **位置**：`design-revision-r01/index.json`（`verdict.state=applied-draft-unreviewed`）、`design-d02/index.json`（`verdict.state=rework-applied-pending-directed-re-review`）、`design-d03/index.json`（`verdict.state=decision-recorded-rework-pending-review-r03`）、`design-d04/index.json`（`verdict=delivered-for-independent-review`）。
- **违反依据**：最终复审链为 D01 PASS WITH MINOR、D02 PASS、D03 PASS、D04 PASS WITH MINOR；作者索引仍停在评审前/待复审状态。D01 的 `B13J-D01-RV-01` 也没有正式 CLOSED 记录。
- **触发场景**：下游把作者索引状态当作当前门禁状态。
- **影响**：元数据与最终复审链不一致；不影响技术 verdict，但会误导后续规划/交接。
- **修复建议**：在获授权的元数据修订中把作者索引更新到最终状态，并为 D01 LOW 增加 CLOSED/处置记录；不要改写历史 review 报告。
- **证据**：四个索引状态值；D01/D04 review 报告。
- **复审条件**：作者索引状态与最终复审链一致，D01 LOW 有明确处置。

## 5. 待决定与非阻断建议

以下不是本轮 findings 的替代，而是需要 Lead/用户/架构责任人决定的事项：

- **D-01（阻断登记）**：U1–U9 的逐项确认记录，或明确缩小登记范围。责任人：Lead/用户。
- **D-02（阻断登记）**：ADR-0030 生产 CAS provider 的修订/新增决策；如要生产 restore，还需 ADR-0031 的相应修订。责任人：Architect/Lead。
- **D-03（阻断登记）**：ADR-0032 解除条件清单是否补入“registry 唯一来源接线”条目，或以专门 ADR 修订任务承接。责任人：Architect/Lead。
- **D-04（阻断登记）**：S4.22/S4.23/S4.24 的 Task 卡字段与 S4.24 是否拆分；责任人：Planner/Architect。
- **D-05（非阻断）**：F008–F010 的元数据/格式修订是否单独授权；责任人：Lead/作者。不要在本评审中顺手修改。
- **D-06（非阻断）**：`tasks.md:18` 的既有控制字符是否另行修复；责任人：Lead/知识整理。

## 6. 未覆盖范围与限制

- 未运行 Go 测试、PostgreSQL、HTTP、服务、restore、E2E；未连接或创建数据库/schema。
- 未验证真实 POSIX 锁、ACL、双进程互斥、崩溃释放、Windows 拒绝、三入口共锁。
- 未验证真实 anchor/CAS provider、独立故障域、真实 restore 工具/批准主体/宿主隔离/原生回执。
- 未证明生产 `enter`/`install` 可达；本评审不把设计文本当作生产证据。
- 未重新水合全部 12 份历史 review 快照或逐条复核每个旧行锚；已核对复审链、当前关键行锚和报告字节。
- 未做全仓语义审计；未重新裁决 D01–D04 的技术结论，只核对其证据、verdict、边界与规划承接。
- 未做 Markdown lint/render；格式结论来自字节扫描。
- 未修改、删除、移动或归档任何既有文件；未执行任何 Git 写操作。

## 7. 格式实测

| 文件 | 行数/终止符 | CRLF | bare LF | BOM | 末尾换行 | 控制字符（不含 CR/LF/TAB） |
|---|---:|---:|---:|---|---|---:|
| identity-auth design | 4944 | 169 | 4775 | 无 | LF | 0；另有 1 个既有 TAB（byte 468004 / line 3303） |
| article-management design | 3267 | 0 | 3267 | 无 | LF | 0 |
| framework.md | 1118 | 1118 | 0 | 无 | CRLF | 0 |
| decisions/README.md | 110 | 2 | 108 | 无 | LF | 0 |
| Plan | 1628 | 1628 | 0 | 无 | CRLF | 0 |
| tasks.md | 3034 | 3034 | 0 | 无 | CRLF | 2 × `0x0C`（offset 12676、12691） |
| gap-map.md | 218 LF 终止符 / 219 逻辑行 | 0 | 218 | 无 | 无 | 0 |
| D04 review-r02 report | 207 LF 终止符 / 208 逻辑行 | 0 | 207 | 无 | 无 | 0 |

## 8. 实跑命令与真实结果

```text
git rev-parse HEAD
  -> 6da8abbceb0fa197364810a92a8fa2d97b38ddca

git rev-parse --abbrev-ref HEAD
  -> main

git status --porcelain | Measure-Object -Line
  -> 293

Get-FileHash -Algorithm SHA256 / Get-Item Length on 8 baseline files
  -> 全部与任务给定哈希/字节一致（见 2.1）

[IO.File]::ReadAllText(path, UTF8) | ConvertFrom-Json on 13 evidence index.json
  -> 13/13 OK

review index artifact bindings (8 review reports)
  -> 8/8 match (hash + bytes)

top-level gap-map binding
  -> 1/1 match

author report bindings
  -> D02/D03/D04 index match; D01 author report bound in review-r01 inputs and hash matches

rg -n 'S4\.2[234]' tasks.md Plan
  -> exit 1, no hit

rg -n 'B13J' tasks.md Plan
  -> exit 1, no hit

rg -n '^## ADR-003[012]' framework.md
  -> 916 / 971 / 1054, one heading each

rg -n 'ADR-003[012]' decisions -g '!framework.md' -g '!README.md'
  -> exit 1, no second authority

rg -n 'B13J-D01-RV-01|B13J-D04-RV2-0[12]' evidence + knowledge
  -> only original review reports/indexes; no closure record

byte scan: line endings / BOM / trailing / control chars
  -> 见第 7 节

D04 review-r02 report last 8 bytes
  -> 89 B9 E5 87 86 E3 80 82 (no trailing newline)

tasks.md control scan
  -> controls=2 at offsets 12676, 12691 (0x0C)

Test-Path planning-review-r01 (before write)
  -> False
```

## 9. 复审条件

本轮 finding 的复审应在以下前置完成后进行：

1. U1–U9 有逐项确认/延期记录（F002）。
2. ADR-0030/0031/0032 的修订/新增决策已接受，并与 Task `requires` 连接（F004/F005/F007）。
3. S4.22/S4.23/S4.24 已有真实 Task 卡，含文件清单/写集、验收命令+退出码、负向测试、关闭条件、requires、证据落点（F003）。
4. Plan B13J 行只引用已登记 Task；候选说明明确“未登记”（F001）。
5. S4.22 已承接 D04 RV2-01 负向验收；ADR-0032 已承接 RV2-02 条件清单（F006/F007）。
6. F008–F010 的元数据/格式问题已修复或明确接受；不要求修改被评对象来“顺手”关闭。

复审通过的最低标准：上述 1–5 全部可核，且没有新增 HIGH/CRITICAL；否则仍为 CHANGES REQUIRED。

## 10. 边界声明（重复）

本报告是 AI 代理的非作者独立评审产出，不是人的签字，不构成实施授权，不关闭 `pending-B13J-D04`，不解除 `blocked-upstream`，不授权 Git 写、部署或生产操作。此结论只覆盖本轮规划就绪性，不覆盖真实代码、数据库、服务、restore、E2E 或生产可用性。