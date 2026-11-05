# Han AM-Q2 R02 独立复审报告

> **非真人签字**：本报告由 `add-reviewer` 角色生成，是独立技术评审意见，不是人的接收、批准、签字或后续授权。Verdict 只代表评审结论，不代替 Lead/用户对风险与修复方向的决定。

## 0. 基线、对象与实测值

- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 角色：`add-reviewer`（非作者；与作者 Dewey/Aquinas 无关联）；评审日期 2026-11-02
- 对象类型：设计制品 + 测试计划制品（`objectType=design` 制品评审分支）
- 读取依据：`add-code-review` 完整正文、`code-review-checklist`、`abstraction-principles`、`design-doc-writing-style`、上一轮 R01 报告 `.spec/wip/article-management/review-am-q2-r01/report.md`

### 0.1 对象实测值（自行计算，未采信委派数值）

| 对象 | revision | 字节 | LF 行数 | SHA256 | 编码 |
|---|---|---:|---:|---|---|
| `.add/project/knowledge/design-workspace/article-management/design.md` | `article-management-design-ir01-r30` | 382472 | 3294 | `5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863` | UTF-8 无 BOM，纯 LF |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | `article-management-test-plan-ir01-r23` | 326550 | 1680 | `C1923CC43CD5A5E28A5C1512B640AB961F0AE25A155CEE34F0AFDA01BD192AB9` | UTF-8 无 BOM，纯 LF |

两者与委派声称的 revision 一致。行数以 `[IO.File]::ReadAllLines` 口径统计（PowerShell `Get-Content` 对含中文文件计数不可靠，本报告不采用其 `.Count`）。

### 0.2 改前/上下游快照链（用于差分与"原文保留"比对）

| 用途 | 文件 | revision | 字节 | SHA256 |
|---|---|---|---:|---|
| design 改前 | `.spec/wip/article-management/evidence-am-q2/design-r30/design-r29-before.md` | r29 | 381381 | `90AD2BC15E74726A06C9583BDE42C30B72A2B46E010F434EC9BE7C425D102A67` |
| design 改后副本 | `.spec/wip/article-management/evidence-am-q2/design-r30/design-r30-after.md` | r30 | 382472 | `5BAA8373…B1A863`（与现役对象同值） |
| test-plan 改前 | `.spec/wip/article-management/evidence-am-q2/test-plan-r23/test-plan-before.md` | r22 | 324805 | `76584E9278E0542151AE1DFADC61B785DFAF1160976DCC0CAC932775093BA217` |
| test-plan r21 终稿参照 | `.spec/wip/recovery-runtime/evidence-testplan-r01/planning-author-r02/after/.add/project/knowledge/design-workspace/article-management/test-plan.md` | r21 | — | `E2E7ADFD0581BCCE9CD2782F7181C12BA67A8496A1A321EF88006EA94D9FCA27` |

改前快照 SHA 与 R01 独立实测的 r29/r22 完全一致，差分链可追溯，未发现作者提供的"改前"被替换。

### 0.3 两轮检查范围

- 第一轮（findings 复核）：对 R01 的 9 项逐条回原文读取锚点段落，并对 design/test-plan 全文执行 `rg`（AM-Q2、P95、TBD、limit/批次/规模、`MaxLimit`、fixture tuple、`3.4.6`、旧锚 `90AD2BC1`/`76584E92`、`当前r21` 等）。
- 第二轮（差分与反向核验）：`r29→r30`（37 增 33 删）、`r22→r23`（15 增 15 删）逐 hunk 复核；再以 `r21 终稿→当前` 反向差分定位 r22/r23 的全部改动，用于验证 §11.2「原文保留」是否真正成立。
- 沿真实消费者/上游依据的有界检查：test-plan front matter 对 design 的 SHA/字节/行数锚点、test-plan 内对 design 章节的引用、§11/§12 状态继承、历史 `historical_r20_*`。
- 未逐行通读 3294/1680 行全文；未覆盖区域的结论只限于定点读取、全量 `rg` 与差分范围。

## 1. R01 九项 finding 逐条复核

| R01 ID | 原严重度 | R02 判定 | 当前锚点与证据（回原文核实） |
|---|---|---|---|
| DR-29-01 | MEDIUM | **PASS** | `design.md:961`–`:984` 为 `##### r29 AM-Q2 容量参数确认`，层级正确（父级 `#### 3.3.1 共享基础` 在 `:803`，紧随 `##### 接口设计` `:986` 之前）；`§3.4.6` 现起于 `:2307`，r29 块已整体移除，`:2323` 新增外指语句"见 §3.3.1 的 `##### r29 AM-Q2 容量参数确认`（r30 迁入）"；`:1962` 的引用同步改为"见 §3.3.1 的 AM-Q2 确认记录"。不再出现"AM-Q2 块位于 L2282–L2463 之间"。 |
| TR-22-01 | MEDIUM | **PASS** | `test-plan.md:172` 已改为"AM-Q2 已确认分页 20/100/200、首期文章总量 100000、单用户上限 10000，P95 首期验收基线（开发机测量参考，非目标环境结论）…；目标环境真实 P95/并发/数据分布、告警阈值、retention、RPO/RTO 仍 TBD（AM-Q6/AM-Q8）"。原文"容量负载、时延、留存和部署参数仍保持 TBD"在全文 `rg` 中已零命中。 |
| TR-22-02 | MEDIUM | **PASS** | `test-plan.md:275` 性能层行已写"AM-Q2 分页 20/100/200、首期规模 100000/10000 与 P95 首期验收基线…已定；其他性能目标、目标环境真实 P95/并发/数据分布及告警阈值仍 TBD"。无条件"目标值 TBD"已零命中，与 `:883` A-PERF-02、`:1247` 一致。 |
| TR-22-06 | MEDIUM | **PASS** | `test-plan.md:1653` 已恢复为 `| AM-Q2 分页/规模/性能目标 | partial | 算法与正确性可测；limit、批次和 P95 目标被阻塞 |`。更强证据：`§11.2` 全文（现 `:1600`–`:1659`，标题 `:1600`、说明 `:1602`）与 r21 终稿快照对应段（`:1576`–`:1635`）**逐字节相同**（`-ceq` 判定 True），说明不是"看起来像原文"，而是整段原文保留已恢复。 |
| DR-29-02 | LOW | **PASS** | P95 全部现役表述改为"首期验收基线（开发机测量参考，非目标部署环境结论）"：`design.md:190`、`:963`、`:974`、`:980`、`:2841`、`:2966`、`:3180`、`:3264`；全文 `rg "P95 目标"` 仅剩 `:2841` 中带限定语的"P95 已有首期验收基线（开发机测量参考）"，无未限定的"验收目标"承诺。 |
| DR-29-03 | LOW | **PASS** | `design.md:3268` 起 `## 确认记录`；`:3270` 已改为"本段为 r22 时点的历史确认记录（当时版本 `article-management-design-ir01-r22` draft）；本文件现行版本以 front matter 的 r30 revision 为准…"。全文无未限定的"当前为 r22"。 |
| TR-22-03 | LOW | **PASS** | `test-plan.md:590` 已改为"…QueryConfig用显式测试值并标为非生产默认；AM-Q2 已确认 production 部署基线 DefaultLimit=20、MaxLimit=100、CandidateBatchSize=200，首期容量 100000/10000，P95 首期验收基线（开发机测量参考，非目标环境结论）…；目标环境真实 P95/并发/数据分布、告警阈值、retention、RPO/RTO 仍 TBD"。旧短语"生产参数、容量/P95仍TBD"零命中。 |
| TR-22-04 | LOW | **PASS** | `test-plan.md:29` `last_updated: 2026-11-02`，与 `design.md:7` 同日、与 `:5` r23 revision 一致。 |
| TR-22-05 | LOW | **PASS** | `test-plan.md:106` 已改为"本规划 r23 在 r21[…]基础上，同步 design r30 的 AM-Q2 容量与查询参数口径；r21 时点原增量…"；`:108` 改为"本规划状态为 planned；r23 作者侧同步 design r30…并完成 TR-22 文档 finding 的作者侧修订…"；`:1662` 改为"本规划 r23 已由作者同步 design r30…待非作者复审；…r21 时点仅完成恢复规划文档修订…"。全文 `rg "当前r21"` 零命中。 |

**小结：R01 9/9 全部 PASS，0 PARTIAL、0 FAIL。** 判定依据全部来自现役原文与差分，不以作者声明为准。

### 1.1 关于"闭合"的性质边界

上表 PASS 只表示"R01 指出的文档口径/结构缺陷已按原 finding 的验收方式消除"，不表示 AM-Q2 的容量、P95、并发或目标环境结论已取得；相关未验证项见 §5。

## 2. 额外独立检查（超出 R01 清单）

| 检查项 | 结论 | 证据 |
|---|---|---|
| 1. 迁块后悬挂交叉引用／层级错乱／锚点失效 | **PASS（2 处精度问题见 F001）** | 层级：`#### 3.3.1 共享基础`(:803) → `##### 生产配置…`(:877) → `##### r29 AM-Q2 容量参数确认`(:961) → `##### 接口设计`(:986)，顺序语义连贯（配置合同之后、接口之前）。`§3.4.6`(:2307) 的 AM-Q2 残留仅为 `:2323` 的外指语句。test-plan 中引用 design `§3.4.6` 的 `:749`、`:760`、`:1014`、`:1016` 均指 Journal/删除合同，不是 AM-Q2，合法。无 `](#…am-q2…)` 形式的锚点链接，故迁移未产生失效锚点。 |
| 2. 数值唯一性与约束未被软化 | **PASS** | 单一数值套系：`design.md:190`、`:883`–`:885`、`:921`、`:969`–`:971`、`:976`、`:2966`、`:3180`、`:3264`；`test-plan.md:172`、`:243`、`:275`、`:590`、`:883`、`:1210`、`:1247`。无第二套 `DefaultLimit/MaxLimit/CandidateBatchSize` 取值（除 `:2919` 明确标注为下界/上界 fixture 的 `1/1/2`、`1/2147483646/2147483647`）。`CandidateBatchSize > MaxLimit` 仍为 `>`：`:885`、`:921`、`:976`，且 `:2920` 仍以 `CandidateBatchSize == MaxLimit` 为失败用例；未见 `>=`。三键仍"环境；必填、无默认"：`:883`–`:885`、`:969`–`:971`；`:2074` 明确"使用同一 QueryConfig.DefaultLimit｜不添加数字默认"，未写成代码默认或 fixture 默认。 |
| 3. test-plan 是否只做文档口径修订 | **PASS** | r22→r23 差分仅 8 个 hunk、15 行替换，无新增/删除用例行；用例 ID 集合前后均为 110 个且完全相同（`Compare-Object` 无增无删）。`A-PERF-02` 仍 `planned`（`:883`），`A-PERF-01/03/04/05` 仍 `partial/blocked/partial/blocked`（`:882`、`:884`–`:886`）。`execution_status`(`:4`)、`status`(`:3`) 未改；未把 NOT RUN 写成 PASS（新增文本仅出现"不冒充…PASS""不证明新增Admin2已执行"等否定式）。`historical_r20_*`（`:8`–`:20`）与改前快照逐行相同。 |
| 4. 现役/历史分层是否真的恢复 | **PASS（§7.7 一处标记问题见 F002）** | 显式"原文保留"段 `§11.2`(:1600–1659) 与 r21 终稿逐字节相同（见 TR-22-06 证据）。reverse diff `r21 终稿→当前` 中不再出现 `§11.2` 任何 hunk，说明 r22 对该段的唯一改动已被精确回退，未留下第二处静默改写。 |
| 5. `test-plan.md:7` 锚点一致性 | **PASS** | `:7` `r23_source_design_artifact: "sha256:5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863; bytes:382472; lines:3294"`，与 design r30 实测 SHA/字节/行数三项全等；`:6` revision 亦为 `article-management-design-ir01-r30`。 |
| 6. 旧锚残留 | **PASS** | 现役 design/test-plan 中 `rg "90AD2BC1\|76584E92"` 零命中；两值仅存在于 evidence 快照与 R01 报告中。 |

## 3. 剩余 finding

### [LOW] AMQ2-R02-F001：迁块后两处"见 r29"未指向新落点

- **位置**：`design.md:1277`、`design.md:2292`
- **现状**：`:1277` "容量目标见 r29，目标环境执行计划实测仍 unknown / NOT RUN"；`:2292` "worker 的容量、吞吐和告警阈值仍为 TBD（QueryConfig 分页基线与首期规模目标见 r29）"。r30 已把该内容移入 `§3.3.1` 的 `##### r29 AM-Q2 容量参数确认`，但这两处仍以 revision 代号指向，未像 `:1962` 那样改为章节引用。
- **触发场景**：读者在 `§3.4`/`§5` 附近按"见 r29"查找分页基线与规模目标时，需自行推断 r29 对应现役 r30 文档中的哪一节；只读 front matter 的消费者可能去取 r29 历史快照。
- **影响**：AM-Q2 决策的可发现性仅部分恢复（DR-29-01 的原始诉求之一）。不影响任何合同、数值或状态，属文档追溯精度问题。
- **依据**：`design-doc-writing-style` 第 3 条"克制代号与链接、正文要能独立读懂"与"文档级硬约束 A 自包含"；R01 DR-29-01 的验证要求"引用落到新位置"。
- **修复建议**：两处改为"见 §3.3.1 的 r29 AM-Q2 容量参数确认"，与 `:1962` 写法一致。
- **验证**：`rg -n "见 r29" design.md` 应零命中，或每处均带 `§3.3.1` 定位。

### [LOW] AMQ2-R02-F002：§7.7（r16，NOT RUN）历史时点段内无标记承载 r30 现役值

- **位置**：`test-plan.md:1210`；所在小节标题 `test-plan.md:1202` `### 7.7 Admin2 合成数据与执行载体（r16，NOT RUN）`
- **现状**：r23 把该段原句"生产limit/批次/环境值和P95容量仍为既有TBD"替换为"AM-Q2 已确认生产 limit/批次 DefaultLimit=20、MaxLimit=100、CandidateBatchSize=200 与首期规模 100000/10000；P95 首期验收基线（开发机测量参考，非目标环境结论）…"。该小节按标题仍是 r16 时点规划文本，替换后未加"r23 现役注记（非 r16 原文）"之类标记。
- **触发场景**：读者按标题把 `§7.7` 当作 r16 时点原文引用时，会把 r30 的确认值当作 r16 既有事实；或按现役状态引用时又发现其位于 r16 段。
- **影响**：与刚恢复的现役/历史分层（TR-22-06）口径不一致，削弱历史时点可还原性；R01 §7 曾把该句明确称为"历史时点文本"并建议加标记。不改用例集合、`A-PERF-02` 状态、`NOT RUN` 语义，也不构成伪造 PASS。
- **依据**：`test-plan.md:1602` 的分层契约与 R01 TR-22-06 的同一判据；R01 §7 对该段"可加 historical as of r16"的非阻断建议。
- **修复建议**：在 `:1210` 句首加"r23 现役注记（非 r16 原文）："，或把该 AM-Q2 句移至 `§1.1`/`§4.9` 等现役段、`§7.7` 保留 r16 原 TBD 表述。
- **验证**：`rg -n "AM-Q2 已确认生产 limit/批次" test-plan.md` 只应出现在现役段或带历史标记的段落；`§7.7` 不再无标记承载现役数值。

**计数：0 BLOCKER / 0 HIGH / 0 MEDIUM / 2 LOW（均为非阻断文档追溯项）。**

### 3.1 非阻断观察（不计 finding）

- `design.md:965`、`:2323` 各有一句迁移来历（"r30 依据独立评审 DR-29-01 …迁入""（r30 迁入）"）。`design-doc-writing-style` 准则 K 反对过程叙述，但此处一句话即闭合了评审追溯，信息量正向，故不判缺陷。
- `design.md:963`–`:984` 块内"部署基线、容量规划输入与首期验收基线"与"不是代码默认、测试 fixture 或目标环境实测结论"并存，区分清楚，未产生"实现基线 / 目标规格"混淆（准则 F 满足）。

## 4. Verdict

**PASS WITH CONDITIONS**

- BLOCKER：0
- HIGH：0
- MEDIUM：0
- LOW：2（AMQ2-R02-F001、AMQ2-R02-F002）
- R01 复核：9/9 PASS

**条件**：F001、F002 作为非阻断文档追溯项修正或显式登记后即可。两项均不改变任何合同、数值、用例集合或执行状态，故不阻断 AM-Q2 文档侧结论。

**AM-Q2 文档侧关闭建议**：可以关闭（文档侧）。前提与边界：

- 关闭范围仅限"design r30 与 test-plan r23 的 AM-Q2 口径同步与 R01 finding 整改"。
- 不得据此宣称测试已执行、P95/容量已达标、PostgreSQL/SMTP/E2E 已通过，或人的成果已接收。

## 5. 未验证 / NOT RUN / unknown

- 未运行 Go 测试、`go vet`、构建、性能压测；未连接 PostgreSQL、SMTP；未做真实部署、E2E 或容量测量。目标环境真实 P95、并发、数据分布、告警阈值、retention/RPO/RTO 仍为 unknown / NOT RUN。
- 未核验 design/test-plan 中全部历史 PASS/已关闭声明是否与其原始证据一致（仅复核本次改动涉及的历史段与 `historical_r20_*` 锚点）。
- 未逐行通读 3294/1680 行全文；未覆盖区域只按差分、定点读取与全量 `rg` 判断，可能存在未触及区域的既有问题。
- 未评审工作树中与 AM-Q2 对象无关的其它 modified/untracked 文件，未对其作任何判断。
- 未核验作者 Agent 身份、工具执行记录与 evidence 报告中的过程陈述；作者报告仅用于定位，未作为证据。
- 不代替人的成果接收、批准或后续授权；本报告不构成签字。

## 6. 只读声明

- 本次评审只写本文件 `.spec/wip/article-management/review-am-q2-r02/report.md`（另仅创建其所在目录），未修改 design.md、test-plan.md 或任何其它被评对象/制品。
- 未执行 `git add/commit/push/branch/stash/checkout/restore/reset/clean`；未删除或移动任何文件。
- 评审结束后重新计算两份被审对象 SHA256，仍为 §0.1 表中的值（`5BAA8373…A863` / `C1923CC4…2AB9`），被审对象未被改动。