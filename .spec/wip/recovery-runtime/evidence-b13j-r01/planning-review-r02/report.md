# B13J Planning Review r02 — 定向复审

- 评审角色：add-reviewer（非作者独立评审）
- 对象类型：planning（B13J 批次可登记性 + 候选 Task 卡可实施性）
- 评审轮次：r02 定向复审（只判 r01 十条 finding 是否 CLOSED，及修订是否引入新问题；不重做整轮评审）
- 被评对象：`pending-B13J`；r01 十条 finding 的作者处置；ADR-0030/0031/0032 r02 修订；`u-adoption-record-r01.md`；`correction-r01.md`；四个作者 `index.json` 状态；`design-revision-r01/report.md` §7
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（复审期间未变）
- 评审日期：2026-11-02（Asia/Shanghai）
- 独立性：D01–D04 设计作者为 Singer，D04 design-review 作者为 Sagan；本评审未参与上述产出，不默认接受作者或 Sagan 的结论，独立核对原始制品与字节。
- 最终结论：**CHANGES REQUIRED**

## 1. 结论

本轮修订把 r01 的多数前置缺口落成文本，但**不能**据此登记 S4.22/S4.23/S4.24 实施卡：

- 可进入的是“**决策卡**”登记轮次（例如产出 ADR-0030 r03 的文档/决策 Task），且需先修正 ADR-0030 约束 9 的入口。
- **不可**登记“实施/验证卡”（S4.23/S4.24）与“生产解除/装配卡”（S4.22）：ADR-0030 约束 9 把“决策修订”与“约束 7 已解除（含真实独立故障域验证）”绑成同一入口，形成登记死锁（R02-F001）。
- 候选 S4.24 在 ADR-0031、ADR-0032 与 gap-map 中被赋予互相冲突的身份（restore 工具 vs `exclusive` 锁接线 vs 两者合并），任何单卡无法同时满足（R02-F002）。

r01 十条中：F001/F002/F004/F007/F008/F010 的**具体缺陷**已处置（其中 F002 为记录级采用，非人的业务签收）；F003/F005/F006/F009 仍 OPEN。另新增 R02-F001（HIGH）、R02-F002（HIGH）与 R02-F003/F004/F005（LOW）。

## 2. r01 finding 逐条判定

### 2.0 判定表

| 原 finding | r01 严重度 | r02 判定 | 依据（位置行锚） |
|---|---|---|---|
| F001 顺序结论 | HIGH | CLOSED（本轮无 Plan 违规；顺序结论继续适用并需增补“决策卡”步骤） | `tasks.md`/Plan 哈希未变；`rg 'S4\.2[234]' tasks.md Plan` exit 1；r01 report:77-78 顺序仍适用；顺序须按 R02-F001 在“ADR 修订”前插入“决策卡”。 |
| F002 U1–U9 记录 | HIGH | CLOSED（记录级采用，非人的业务签收） | `u-adoption-record-r01.md:5-7` 明确效力边界；`:12-22` 逐项 U1–U9 均“已采用”并给依据；满足 r01 report:124/126 的“统一逐项记录”闭口条件。人的业务签收仍 unknown。残余 R02-F003/F004。 |
| F003 候选卡粒度 | HIGH | OPEN | `tasks.md` 未改（F1D65B0D…/823187 B）、Plan 未改（3FA19CDD…/497542 B）；`rg 'S4\.2[234]'` 两文件 exit 1。仍待卡片登记轮。 |
| F004 ADR 前置节点 | HIGH | CLOSED（节点已存在） | ADR-0030 `:960`；ADR-0031 `:1046-1048`；ADR-0032 `:1115-1122`。节点内容问题另立 R02-F001。 |
| F005 锁来源链 | HIGH | OPEN | ADR-0032 `:1115-1122` 已形式化顺序，但入口继承 ADR-0030 约束 7（`:953-958`，尤 `:957` 真实独立故障域验证）与约束 9（`:960`）的“先解除后登记”要求，链仍无法启动。见 R02-F001。 |
| F006 RV2-01 承接 | MEDIUM | OPEN | D04 review-r02 `:133-138` 建议写入 S4.22 负向验收；`correction-r01.md:47` 明写“不足以关闭”；无 S4.22 卡。仍待登记轮。 |
| F007 RV2-02 收口 | MEDIUM | CLOSED | ADR-0032 解除条件新增第 8 条 `:1115-1122`，把“经 registry 唯一来源接线”纳入；与约束 9 `:1104` 咬合。r01 要求“并入解除条件清单或专门任务”已满足。 |
| F008 trailing 声明 | LOW | CLOSED（errata） | `correction-r01.md` 实测准确：report.md 19858 B、207 LF、0 CRLF、末字符 U+3002（末 3 字节 E3 80 82）、无 BOM；原 report.md `ECE706E0…`/19858 B 与 index.json `65E40D60…`/24007 B 零改动。机器消费者仍读 `review-r02/index.json:33` 的错误值，属不可变证据策略。 |
| F009 tasks.md 0x0C | LOW | OPEN（保持，未授权修复） | `tasks.md` 偏移 12676/12691 共 2 个 0x0C；文件哈希未变。 |
| F010 索引状态/D01 LOW | LOW | CLOSED | D01 `index.json:139` `state=pass-with-minor-reviewed-r01`；D02 `:306` `state=pass-reviewed-r02`；D03 `:601` `state=pass-reviewed-r03`；D04 `:596` `verdict=pass-with-minor-reviewed-r02-open-lows`。`design-revision-r01/report.md` 前 10548 B 哈希 565D1547…（=原报告），§7 为 append-only。 |

### 2.1 被审对象哈希/字节（本次实核）

| 对象 | 字节 | SHA256 |
|---|---:|---|
| `planning-review-r01/report.md` | 28901 | `6918A79C1E5A7426037A47E8DA5CEDC5E9F839B39BF0053B1DFE26712254D4AC` |
| `planning-review-r01/index.json` | 28780 | `5825DDD2C9483034015BB948036763156CC3C7ED9BCD3EDD30680792667597E0` |
| `u-adoption-record-r01.md` | 4665 | `1DD59523B3FB0FF336E7204F3EAC764F29CE7D0DDAF0F1BE633F88B2F646FDBD` |
| `decisions/framework.md`（r02 后） | 175141 | `DA77EA1332DB69CB94B90960DEA58B35CABDA76C712EDAE23F2CCF6A1F45B5C1` |
| `adr-r02/report.md` | 29335 | `D66E898C5DEF97767FA735390D95558923F941F633CD2A23009051A8FFE8315C` |
| `adr-r02/index.json` | 24032 | `CA19DD735D4A0B0F5F7EAE3109388A8F3C8C03C6B841DF6A9414882E8A8248AA` |
| `adr-r02/snapshots/framework-before-r02.md` | 169890 | `58C97FC06361022AA48EF737189EF3EB069A7AAD25772BCA4C80FE4D1D5E1F8E` |
| `adr-r02/snapshots/readme-before-r02.md` | 21472 | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` |
| `design-d04/review-r02/correction-r01.md` | 6204 | `F806AA28F9F51B782313C0C8CE119776F1AC1661B719571816FAF70633950CD8` |
| `design-revision-r01/report.md` | 11998 | `6D8B28A47F12A51A483E9660D19BA8D4DF1564CC4DA3DC3A334E8BD1F36264E4` |
| `design-revision-r01/index.json` | 8224 | `218EF70836BE69EBE9D6A2C7485AB9D638E20F7070421DEB48B86265B785A9EF` |
| `design-d02/index.json` | 20071 | `1F2A1D19ED1FA74CB987BDA38442D94DE2BD725315CA72A0F07DCC469D748476` |
| `design-d03/index.json` | 45798 | `DAB0013E113B8AE2417016AF93162E2CD51A6C94DF148E52616790B5FBBB790B` |
| `design-d04/index.json` | 55974 | `AD4EA53C4C3FF6C6F6238320B4973F0D4D27259650D2F0B5DA638CC757AC346D` |
| `design-d04/review-r02/report.md` | 19858 | `ECE706E0B30A3569B21B37C353F76A9A1546E5EC73841B029CCE31D9BF9BE876` |
| `design-d04/review-r02/index.json` | 24007 | `65E40D60D09BA36DD2532173266E36DD6A4A15327A079125B0B3CDBA5F5626FF` |
| `decisions/README.md` | 21472 | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` |
| `tasks.md` | 823187 | `F1D65B0D7016CDBF9C2F4E85B012B5B0B9F714B6DF0C7CDC9326C05676900E34` |
| Plan | 497542 | `3FA19CDD4902F56B8C2F1DF59C570D96FAAE02D8AEE519B4B4165DCCA3637380` |
| `gap-map.md` | 26452 | `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F` |
| identity design.md | 854991 | `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` |
| article design.md | 377424 | `690015BB5C3B4800539F98C5B75115A5A20B5C161ECE7422F9F271840F7C55A7` |

## 3. 新 findings

### R02-F001 — HIGH — OPEN — ADR-0030 约束 9 形成登记死锁

- **位置**：`framework.md:960`（约束 9）；`framework.md:953-958`（约束 7，尤 `:957`）；`framework.md:1116`（ADR-0032 第 8 条继承）。
- **现状**：约束 9 要求任何实施卡登记前“本 ADR 已有新修订…**并按约束 7 解除 closed**”。约束 7 的 `:957` 要求“在真实独立故障域完成 CAS 冲突…验证并保留原始证据”，`:958` 又要求“由正式 Task 规划登记实现与验证范围”。真实验证需要先实现 adapter（S4.23），而 S4.23 卡又不得先登记。
- **影响**：S4.23/S4.24 永久不可登记；ADR-0032 `:1122` 的“不是架构死锁”只覆盖“ADR 修订前”，未覆盖“修订后仍需约束 7 解除”，故 F005 实质未解；ADR-0032 第 8 条链无法启动。
- **建议**：在 ADR/卡片粒度区分三类——（a）决策卡：产出 ADR-0030 r03（产品、凭据、独立故障域、部署/ACL 结论），仅文档/决策，不需真实验证即可登记；（b）实施/验证卡：r03 接受后即可登记，用于产生真实证据；（c）生产解除卡：最后满足约束 7 全部证据 + 人的批准。约束 9 的入口应改为“决策修订已接受”，把“约束 7 已解除”只作为生产解除卡的前置。
- **验证（解除条件）**：修订后文本能给出无环、每步有可核前置的登记序列，且实施/验证卡在 r03 接受后即可登记。

### R02-F002 — HIGH — OPEN — 候选 S4.24 身份冲突

- **位置**：`framework.md:1048`（ADR-0031 称 S4.24 为 restore 生产实施卡）；`framework.md:1119-1120`（ADR-0032 称 S4.24 为 `exclusive` 锁接线，requires S4.23，并在 S4.22 之前）；`gap-map.md:182`（S4.24 = restore source/target/tool + 部署独占 OS 后端）。
- **现状**：同一候选编号被三处赋予不同实现范围与门禁；ADR-0032 的解除链（`:1116-1121`）完全不含 restore 半区。
- **影响**：任何单张 S4.24 卡无法同时满足 ADR-0031（restore 批准/隔离/回执）与 ADR-0032（锁接线 requires S4.23）门禁；登记即产生契约冲突，且 restore 半区在解除链中无承接。
- **建议**：拆分为 S4.24a（restore source/target/tool）与 S4.24b（部署独占锁接线），在 ADR-0031/0032 与 gap-map 同步改名；S4.24b requires S4.23；S4.22 在两者之后。
- **验证（解除条件）**：三处编号定义一致，解除链覆盖 restore 与锁接线两个节点。

### R02-F003 — LOW — OPEN — 采用记录行锚互换

- **位置**：`u-adoption-record-r01.md:7`。
- **现状**：写作“`gap-map.md` §4 `:191`（‘以上均为建议值…’）与 §3.2 `:207`（登记时机…）”。实测 gap-map §3.2 起于 `:185`（`:191` 属 §3.2），§4 起于 `:193`（`:207` 属 §4）；两处行号与章节/引文错配。
- **影响**：读者按锚点查证会落到错误章节；不改变采用结论。
- **建议**：改为 §3.2 `:191`、§4 `:207`。

### R02-F004 — LOW — OPEN — standing 授权来源不可核

- **位置**：`u-adoption-record-r01.md:6`。
- **现状**：声称“用户自 2026-11-01 起…（本记录以 2026-11-02 会话内的最后一次重复为准）”，但证据目录内无该原话的日期/位置记录；仓库仅能核到 2026-10-31 的同类实例（`.spec/wip/recovery-runtime/evidence-b13h-r01/s420-code-review-r01/lead-recheck-r02.md:7`）。
- **影响**：采用记录的授权来源无法独立复核；不影响 F002 的记录级闭口，但削弱可审计性。
- **建议**：补记具体会话/消息定位，或改引可核的 2026-10-31 实例。

### R02-F005 — LOW — OPEN — ADR r02 修订日期早于触发报告

- **位置**：`framework.md:969`、`:1057`、`:1135`（均写“2026-11-01 r02 修订”）；`adr-r02/report.md:6`（“记录时间：2026-11-01”）。
- **现状**：触发它的 `planning-review-r01/report.md:8` 明写评审日期 2026-11-02；`framework.md` mtime 2026-11-02 00:50:59，`adr-r02/report.md` mtime 2026-11-02 00:55:17。
- **影响**：修订追溯时间早于触发事件，审计时易误判先后。
- **建议**：改为 2026-11-02，或明确“决策日期 2026-11-01 / 修订日期 2026-11-02”。

## 4. 关键问答

- **ADR 状态（Q2）**：ADR-0030/0031/0032 在 `framework.md` 正文（`:916`/`:974`/`:1062` 起）与 `decisions/README.md:38-40` 全局索引均登记为“已接受”，无第二套权威。ADR-0019 `:471`（本地 POSIX 文件系统 + OS 锁语义）、ADR-0027（全局恢复 runtime）、ADR-0029（DB 身份分离/ACL）未被覆盖，也未相互冲突；ADR-0032 的 POSIX-only 与 ADR-0019 同口径。README 无修订号字段（末列为“引用次数”），故 r02 不需改 README；但其状态栏未反映 r02 追溯，仅属可读性限制（见 §8）。
- **诚实性（Q3）**：identity design `:3616` 生产 `exclusive` 仍 `recoveryClosedPort`、`:3617` secret 仅只读候选；`tasks.md:1873` Case18 仍 CHANGES REQUIRED、S8.2 保持 `[-]`；`tasks.md:3034` production `Enter`/`Install` 仍 `ErrAdmissionClosed`。framework 中“生产可用/生产就绪”均为否定式（如 `:926` “不得…宣称生产可用”），未发现“生产可用”肯定表述；未发现 Case18/S8.2 被宣称 PASS。
- **F002（Q1）**：`u-adoption-record-r01.md` 满足 r01 的“统一逐项确认记录”记录要求（逐项 U1–U9 + 依据 + 效力边界），故 F002 记录级 CLOSED。“采用建议值”与“人确认”的实质差距如实保留为“人的业务签收 unknown”，不阻断规划；残余以 R02-F003（行锚）与 R02-F004（授权原文不可核）记录。
- **F003/F006**：本轮无 Task 登记（`tasks.md`/Plan 哈希未变），二者保持 OPEN，等待卡片登记轮承接，未被本轮悄悄关闭。
- **F008**：CLOSED（errata 准确、原证据零改动）。原 `index.json:33` 错误值保留属不可变证据策略，机器消费者应以 `correction-r01.md` 为准。
- **F010**：CLOSED（四个索引状态真实；§7 append-only 已由前缀哈希 = 原报告哈希证明）。
- **约束 9 与 F001 顺序（Q 关键）**：原则相容“先决策后实施”，但**当前文本不相容**——它把“决策修订”与“约束 7 解除”绑成同一入口，形成死锁。这是新的登记死锁，不是正当的“先决策后实施”。解除须在 ADR/卡片粒度显式区分“决策卡”与“实施/验证卡”（见 R02-F001）。
- **耦合链可满足性（Q7）**：不采信“有意显式写出”。ADR-0030 首期 registry 恒空 ⇒ 无 adapter bundle ⇒ `exclusive` 只能 `recoveryClosedPort`；ADR-0032 要求部署锁后端。当前“锁来源只能来自 adapter bundle”的路径被约束 9 与约束 7 的耦合封死，属**登记不可达**（非架构不可达）。给出无环可核顺序（§5）后即可满足，但需先改约束 9。

## 5. 最终登记顺序（r02 判定）

1. **决策卡**：登记一张仅产出 ADR-0030 r03 的文档/决策 Task（产品、凭据、独立故障域、部署/ACL 结论）。它不“付诸实施” provider，故不受约束 9 的“实施卡”限制；但约束 9 入口须先改为“决策修订已接受”。
2. **决策修订**：ADR-0030 r03 接受；ADR-0031/0032 按需同步。
3. **实施/验证卡**：登记 S4.23（anchor adapter bundle）与 S4.24a（restore）、S4.24b（锁接线，requires S4.23）；用于产生真实独立故障域证据。
4. **生产解除/装配卡**：在约束 7 全部证据 + 人的批准后，登记 S4.22（root 装配 + secret candidate）。
5. **Plan 行**：最后才在 Plan 外层表登记 B13J 行，引用列只含已登记 Task。

> 该顺序需先落实 R02-F001（改约束 9 入口）与 R02-F002（拆 S4.24）；在此之前不得登记 S4.22/S4.23/S4.24。

## 6. 实跑命令与真实结果

```text
git rev-parse --abbrev-ref HEAD            -> main
git rev-parse HEAD                         -> 6da8abbceb0fa197364810a92a8fa2d97b38ddca
git status --porcelain | Measure-Object    -> 293
rg 'S4\.2[234]' tasks.md Plan              -> exit 1（无命中）
rg 'B13J' tasks.md Plan                    -> exit 1
rg 'S4\.2[234]' framework.md               -> exit 0（:960/:1048/:1117/:1119/:1120/:1122/:1135 等，均为“未登记候选”散文）
rg 'S4\.2[234]' identity/article design    -> exit 1
framework.md SHA256                        -> DA77EA1332DB69CB94B90960DEA58B35CABDA76C712EDAE23F2CCF6A1F45B5C1 / 175141 B
framework 0..126598（126599 B）前缀哈希     -> 8C93C47AE57BED1C8390C07D41210183814185A68B148DCEEF180BDB8AE1C9EB
framework ADR-0019 区间 31305..126598     -> 607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6
framework 控制字符/换行                    -> ctrl=0, TAB=0, crlf=1135, bareLF=0, bareCR=0, BOM=False, endCRLF=True
tasks.md 0x0C                              -> 2 个，偏移 12676, 12691
design-revision-r01/report.md 前缀 0..10547 -> 565D1547F8B75A4E60EB8073AA52326E3C8ADB620F71E2D999E60C40506BE89A（=原报告，append-only）
D04 review-r02 report 末 3 字节             -> E3 80 82（U+3002），207 LF，0 CR，19858 B
D04 review-r02 index artifact.trailing     -> "single LF (0x0a)"（仍为错误值，errata 已更正）
JSON 解析（UTF-8 显式读取）               -> adr-r02 / planning-review-r01 / review-r02 / D01-D04 index 全部 OK
```

## 7. 格式实测

本报告自身：UTF-8 无 BOM、LF-only、末尾单 LF、控制字符 0、TAB 0；行数 = 164；CRLF = 0；bareLF = 行数；bareCR = 0；字节数与 SHA256 见 `index.json` 的 `report` 字段（数字回填只改字节数，不改行数/换行符）。

被审对象实测：`framework.md` 175141 B / 1135 行 / CRLF 1135 / bareLF 0 / bareCR 0 / 无 BOM / 末尾单 CRLF / 控制字符 0 / TAB 0；`adr-r02/report.md` 29335 B；`planning-review-r01/report.md` 28901 B（LF, 末 LF）；`u-adoption-record-r01.md` 4665 B / 37 行；`correction-r01.md` 6204 B / 61 行 / LF / 末 LF；`design-revision-r01/report.md` 11998 B / 159 行 / 无末尾换行（原文件即以“。”结束，§7 追加后仍无末尾换行）。

## 8. 未覆盖范围

- 未重做 r01 整轮评审；本轮只定向复核 r01 十条 finding 与 r02 修订。未逐行重读 D01–D04 全部 review 报告正文（仅核哈希/字节/关键行锚）。
- 未运行 Go 测试、PostgreSQL、服务、E2E、restore 或任何真实部署；未验证 `exclusive` 运行时行为、真实独立故障域、ACL、OS 锁。
- 未取得人的业务签收或 standing 授权原文的日期/位置证据（R02-F004）。
- 未核对 `.spec` 下“新增产物”的绝对基线（`.spec` 不被 Git 跟踪）；以目录树列举确认预期产物集存在，未发现 `design-d04/**` 之外的异常新增，但无法用 Git 证明历史时点唯一性。
- README 状态栏未反映 r02 追溯，属可读性限制；因无修订号字段且结论未变，未列为缺陷。

## 9. 边界声明

本评审只判“是否可进入 Task 卡登记轮次”。**不构成实施授权**；**不关闭 `pending-B13J-D04`**；**不解除 `blocked-upstream`**；**不视为人的签收**；**不授权登记 S4.22/S4.23/S4.24**。本轮只读被审对象，只新增本证据目录；未做 Git 写操作，未删除或移动任何文件，未运行 Go/PG/服务。
