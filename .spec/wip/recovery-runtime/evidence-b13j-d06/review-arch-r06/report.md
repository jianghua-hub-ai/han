# ADR-0031 `adr-0031-r12` 架构独立复审报告（r06）

- 日期：2026-11-02
- 角色：add-reviewer（架构轴 / 架构侧 Reviewer）；objectType：adr；只判「决策接受级」，不做实施验证、不替作者修复、不代替人的接受。
- 复审对象：`.add/project/knowledge/decisions/framework.md`
  - 实测：297,412 B / 1,275 行 / SHA256 `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`；CRLF 1,275、bare LF 0、无 BOM、末尾 CRLF。
  - 冻结快照：`C:\Users\J00010385\AppData\Local\Temp\framework-adr0031-r12-freeze-800B5689.md`，297,412 B / 1,275 行 / 同哈希。**现场与快照逐字节一致，无漂移。**
  - 判读段：`:1079`–`:1194`（116 行）；`:1195` 空行、`:1196` `---`、`:1197` 空行、`:1198` `<a id=adr-0032></a>`，ADR-0032 自此起。
  - 修订版本：`adr-0031-r12`（`:1082` 修订行、`:1084` 状态段、`:1194` 追溯行）。
- 改前基线（r11）：`C:\Users\J00010385\AppData\Local\Temp\framework-adr0031-r11-freeze-9723F92F.md`，292,606 B / 1,274 行 / `9723F92FB53B076D228B5FA392CEC07CA0F74ED460524EBB7F00AC9202EB53CD`。
- 基准报告（本路上次判定）：`.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r05/report.md`，27,698 B / SHA256 `98C3DD6F6670BB5F875B80AD084C9A0130F4956220206C38AE0B8DD938271FB7` / CHANGES REQUIRED，BLOCKER 0／MEDIUM 1／LOW 2（`ADR-R05-01`／`ADR-R05-02`／`ADR-R05-03`）。
- 安全侧对照（不作本路接受依据）：`review-sec-r05/report.md`，24,558 B / `40F5C45DDCC77A1E786B9B1E1E64E03F7EA8DB1519CB22188E695658EB8DBF12` / PASS，0/0/1（`ADR-S05-01` 与 `ADR-R05-02` 同源，r12 一并修正）。
- 作者证据（只作线索，数值均自算）：`adr-0031-r12/report.md` 5,636 B / 84 行 / `654BBD6A08158C2228310F56EE02B58DD6388DECAC6A7A6C4DA2201691B28618`；`adr-0031-r12/hashes.txt` 2,180 B / 77 行 / `E7E541D2327654AF40CBE1F77F424E7B1FE25B8649C587C14231F236414EA01D`。
- 上游只读对象：`design.md`（r70）1,103,152 B / 5,476 行 / `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`；`internal/recovery/runtime.go` 59,138 B / 1,969 行 / `4C0946AF7C29DE472CAE4848CE2F55A68288003071957FB62B66BFF3B59604C4`；`tasks.md` 904,142 B / 3,312 行 / `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C`；`decisions/README.md` 21,613 B / 110 行 / `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2`。

## 0. 方法与两轮检查范围

### 0.1 方法
- 完整读取 `.codex/skills/add-code-review/SKILL.md`；按其中加载规则读 `.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`；按 adr 维度另读 `.codex/guidance/decisions-guide.md`。
- adr 维度逐项：实际问题／约束、候选与否决理由、选定方案代价、适用范围、确认状态、消费者影响与迁移/退出条件；偏好不冒充事实，未定不写已接受。
- 全部数值自算；作者报告与 `hashes.txt` 只作线索，逐项复算后才引用。

### 0.2 Round 1（diff 快速审查）
- 以 r11 冻结副本为基线，按行逐字节比对 r12 现行文件：同索引变化仅 `:1082`／`:1084`／`:1118`／`:1174`，插入 `:1194`；其余行仅行号 +1、逐字节相同。
- 逐行读改动前后文本，核改动是否只落在声称范围、是否夹带实质决策变化。

### 0.3 Round 2（全段＋真实消费者）
- 全读 ADR-0031 段 `:1079`–`:1194`（不变量、约束、三级门禁、前置节点、引用记录）。
- 对照 design r70 的条款锚与事实行：`:3672`（resume 不新 Begin、不盲 Replace）、`:3886`（不再 Begin/Replace、restore_unknown）、`:3899`（零新 Begin）、`:4037`（r64.5 唯一调用图）、`:4047`（r64.10 审计 sink）、`:4062`（r65.5 V 编号/五处等式）、`:4074`（r66.2 V7–V11）、`:4077`（r66.5 见证端点/pin）、`:4079`（V 编号范围限定）、`:4083`（r66.11 声明）、`:4373`（generation 三元）、`:4377`（失败分类）。
- 对照实现时序：`internal/recovery/runtime.go` `:1573`–`:1684`（`enter`／`begin`／`verifyReplacement`／`resume`），并对 `Journal.Begin(`／`.Replace(`／`.VerifyRestored(`／`.Prepare(` 做全文计数与行定位。
- 消费者／索引有界检查：`tasks.md`（`:3186`、`:3220`）、`decisions/README.md:39`（仅登记，不作判据）。

### 0.4 只读声明（实际判读过的文件）
`framework.md`（对象）及其 r12 快照、r11 冻结副本；`review-arch-r05/report.md`；`review-sec-r05/report.md`；`adr-0031-r12/report.md`、`adr-0031-r12/hashes.txt`（线索）；`design.md` r70 快照；`internal/recovery/runtime.go`；`tasks.md`；`decisions/README.md`；`.codex/skills/add-code-review/SKILL.md`；`.codex/guidance/code-review-checklist.md`、`abstraction-principles.md`、`decisions-guide.md`。除本报告与同目录 `hashes.txt` 外未写入任何文件；未执行 Git 写、删除、移动、归档。

## 1. 对象真实性与字节账（全部自算）

| 项 | 值 |
|---|---|
| 现行 framework.md | 297,412 B / 1,275 行 / `800B5689…D31D20` |
| r12 快照 | 297,412 B / 1,275 行 / `800B5689…D31D20`（一致） |
| r11 基线 | 292,606 B / 1,274 行 / `9723F92F…EB53CD` |
| 字节增量 | +4,806 B（= 772 + 255 + 3,779） |

- 逐行 diff（同索引）：`:1082` 69→69 B；`:1084` 1,164→1,164 B（仅 `adr-0031-r11`→`r12`）；`:1118` 10,625→11,397 B（+772）；`:1174` 3,671→3,926 B（+255）；插入 `:1194`（2,403 字符 / 3,777 B + CRLF 2 = 3,779 B）。
- 配平：772 + 255 + 3,779 = 4,806；292,606 + 4,806 = 297,412 ✓，与实测增量一致。
- `:1193`（r11 历史追溯行）与基线 `:1193` 逐字符相同；基线 `:1193`–`:1273`（81 行）≡ 现行 `:1194`–`:1274`（81 行），逐字节相等。
- 段外不变量（raw bytes，行首偏移→EOF，不重编码、不归一）：前缀（`## ADR-0031:` 之前）= 205,357 B / 109,097 字符 / `594FCB9DEF2F9D9FED11952D94FF75CA536D43B9C2FD937B74EC262C3839E1DB`；后缀（`<a id=adr-0032>` 起至 EOF）= 21,165 B / 11,152 字符 / `F921F81BC589C51FE104DB62F6CB42065C6E417B70F54EEFF7222400B88B64A5`。r11 与 r12 两段逐字节相等。
- ADR-0031 段 raw bytes（口径：第 1079 行首字节至第 N+1 行首字节，含各行行尾 CRLF、不重编码）：r12 `:1079`–`:1194` = 70,902 B / `32495F7E7A7D291B22F5E2EA9A902C7674F9573BC6335193783AC442FD2B3EB5`；r11 `:1079`–`:1193` = 66,096 B / `91C1150A48183302520ABADFCB3D81BFBFE70A2AC12410FE3CA7EC9EAADA10CA`；差 +4,806 B，与全文增量一致。
- 全角括号：全文 559/559；ADR 段 181/181；改动行逐行平衡（`:1084` 1/1；`:1118` 38/38、ASCII 5/5；`:1174` 9/9、2/2；`:1194` 12/12、6/6）。

## 2. 原 finding 逐条闭合（沿用原 ID）

| 原 ID | 严重度 | 判定 | 关键依据 |
|---|---|---|---|
| `ADR-R05-01` | MEDIUM | **CLOSED** | `:1118` 已逐路径分写；`runtime.go` 调用点逐项核实，无二义 |
| `ADR-R05-02` | LOW | **CLOSED** | `:1118` 已按 design `:4377` 原因分类，含 r62.3 漂移改 Closed 覆盖 |
| `ADR-R05-03` | LOW | **CLOSED** | `:1118` 已把 `:4072` 定性为行号映射说明、不构成合同条款，并与 `:4083` 并列 |

### `ADR-R05-01`（MEDIUM）— CLOSED
- 原缺陷：`:1118` 把 `enter`／`resume` 中 R/V 调用合并写成「均在 `Journal.Begin` 完成之后」。
- r12 现文（`:1118`）：`R1`–`R8` 只在 `enter` 的 `Replace` 路径出现且位于 `Journal.Begin`（`runtime.go:1617`）之后；`V1`–`V11` 属 `VerifyRestored` 路径，`enter`（`runtime.go:1634`）与 `resume`（`runtime.go:1666`）均会调用；`enter` 侧在同一 `Journal.Begin` 之后，`resume` 侧在 `Prepare`（`runtime.go:1656`）之后且无新 Begin；`resume` 不调用 `Replace`。
- 独立核实：`Journal.Begin(` 全文 1 次 @`:1617`（在 `begin` `:1600` 内，由 `enter` `:1577` 调用）；`.Replace(` 全文 1 次 @`:1584`（`enter` 内，Begin 之后）；`.VerifyRestored(` 2 次 @`:1634`（`verifyReplacement`，`enter` `:1591` 调用）与 @`:1666`（`resume`）；`.Prepare(` 2 次 @`:1607`（`begin`）与 @`:1656`（`resume`）；`resume` `:1647`–`:1684` 无 `Journal.Begin`、无 `Replace`。design `:3672`／`:3886`／`:3899` 与之一致。
- 结论：逐路径可唯一判读，与 design/实现事实一致，原文二义已消除 → CLOSED。

### `ADR-R05-02`（LOW）— CLOSED
- 原缺陷：`:1118` 把 `P6`／`Prepare` 重读失败一律写成 `ErrAdmissionClosed`。
- r12 现文：`P6`／`Prepare` 按 design `:4377` 的原因分类——缺键、文件缺失或不可读、非绝对规范路径、owner／mode 不符、非 regular 或被符号链接替换「等」→ `ErrAdmissionIncomplete`；未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked`、预检通过后的字节／`generation` 漂移 → `ErrAdmissionClosed`；并明确 `enter` 的 `P6` 失败不进入 `Journal.Begin`。
- 独立核实 design r70 `:4377` 原文：`ErrAdmissionIncomplete` 原因集合与 ADR 枚举一致（且 ADR 用「等」保留超长/非 UTF-8/JSON 不合法/缺字段等）；`ErrAdmissionClosed` 对应未到/已过/revoked；同段 r62.3 覆盖句「预检后字节／inode／挂载漂移改 `ErrAdmissionClosed`」支持 ADR 的漂移归类；`generation` 漂移按 `:4373`（r67.6）「任一漂移即 closed／`restore_unknown`」，预检阶段为 closed。
- 结论：错误分类与 design 一致，且 fail-closed 语义未变 → CLOSED。

### `ADR-R05-03`（LOW）— CLOSED
- 原缺陷：`:1118` 检索断言「未命中任何编号条款」不精确（`:4072` 实含 r68.1–r68.8／r69.1–r69.3／r70.1–r70.3）。
- r12 现文：明确 `:4072` 是「行号映射说明（列出 r68.1–r68.8／r69.1–r69.3／r70.1–r70.3 的修订号与插入前行号对应，不构成合同条款）」，并与 `:4083` 的「r70 修订沿用本声明」并列；同时保留「不据此声称已复核 r67–r70 的全部改动」。
- 独立核实：design r70 全文检索 `r68.`＝8、`r69.`＝3、`r70.`＝3，合计 14 次，全部落 `:4072`；`:4072` 确为修订范围/行号口径/优先级与 r67–r70 就地修订登记块，其点号编号是对条款的登记引用，不构成新合同条款；`:4083` 为 r66.11 声明（含「r70 修订沿用本声明」）。
- 结论：支持句已改为如实登记，结论（六条条款锚未被改写）成立 → CLOSED。`:1193` 历史行仍含旧措辞，按既定口径属历史保留，不据此判 CHANGES REQUIRED。

## 3. 独立事实核对（实现与 design）

### 3.1 `internal/recovery/runtime.go` 调用点（自算，全文计数）
| 调用 | 次数 | 行 | 上下文 |
|---|---:|---|---|
| `Journal.Begin(` | 1 | `:1617` | `begin`（`:1600`）内，`enter` `:1577` 调用 |
| `.Replace(` | 1 | `:1584` | `enter` 内，`begin` 之后 |
| `.VerifyRestored(` | 2 | `:1634`、`:1666` | `verifyReplacement`（`enter` 侧）／`resume` |
| `.Prepare(` | 2 | `:1607`、`:1656` | `begin` 内／`resume` |

- `resume`（`:1647`–`:1684`）序列：`acquire`→`readControlled`→`seed`→`Prepare`（`:1656`）→`validateRestorePlan`→`refresh`→`VerifyRestored`（`:1666`）→ …；**无** `Journal.Begin`、**无** `Replace`，与 design `:3672`／`:3886`／`:3899` 一致。
- `:1537` 注释「Resume continues only independently verified persistent progress, never Begin/Replace」与实现相符。

### 3.2 design `:4377` 失败分类（r70 原文）
- `ErrAdmissionIncomplete`：缺键、文件缺失或不可读、非绝对规范路径、owner／mode 不符、非 regular 或被符号链接替换、超长、非 UTF-8、JSON 不合法、缺字段。
- `ErrAdmissionBinding`：字段值与 `RestorePlan`／实测不符、`approval_digest` 不符、签发=复核、复核非具名、两条记录互替；**同一快照内**不一致。
- `ErrAdmissionClosed`：未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked`；r62.3 覆盖：预检后字节／inode／挂载漂移改 Closed。
- `ErrAdmissionUnsupported`：平台不支持该文件边界。任一非 nil 均在 Begin 前返回、不降级为「先恢复后补批准」。
- ADR `:1118` 的第一句已写「按 design `:4377` 的原因分类」，枚举与覆盖句一致；另见 §6 观察项（Binding／Unsupported 未逐项列出，属可读性残余，不改变分类委派）。

### 3.3 design 检索断言
- `r68.`／`r69.`／`r70.`（带点）共 14 次，全部在 `:4072`；`r70`（不带点）另有 12 次，其中包含 `:4083` 的「r70 修订沿用本声明」。
- r12 的「不构成合同条款」与「不据此声称已复核 r67–r70 的全部改动」同时成立。

## 4. 无回退底线逐项核对（r11→r12，改动行仅 5 行）

| 底线 | 证据位置 | 结论 |
|---|---|---|
| 不变量 1–6 | `:1131`–`:1149` | 不在改动行，逐字节未变 |
| 约束 1–9（含解除条件九条） | `:1151`–`:1170`（解除条件 `:1160`–`:1169`） | 逐字节未变 |
| 「首期不提供生产 restore 工具与生产批准主体」结论与候选取舍 | `:1084`／`:1090`／`:1100`／`:1105`／`:1129`／`:1174` | 保留；`:1084` 仅改修订号，原「已接受（2026-11-01 用户 D03…）」文字保留 |
| `anchor_ref` 五处等式 + policy 必填 `tool.receipt.trust_anchor_ref`（缺键/空串/名不符即 closed） | `:1118`（共同前缀，未改） | 未变 |
| producer／verifier／witness 三签模型，三席互不代签，`witness_key_id` 非回执签名公钥唯一来源 | `:1118`（共同前缀，未改） | 未变 |
| `generation` 三元 | `:1118`（未改段）＋ design `:4373` | 未变 |
| 无条件「不声称抗 root」 | `:1118`（r10 无条件残余段，未改） | 未变 |
| 两种「见证」分域（ADR-0030 `(w,r,token)` vs 回执外部锚） | `:1118` 末段「见证术语消歧（r10）」（共同后缀，未改） | 未变 |
| `restore-v1`／`verify-v1` 严格合同 | `:1086`／`:1094`／`:1116` | 未变 |
| no-follow 稳定 fd 与 fd-exec | `:1117`／`:1141`／`:1143` | 未变 |
| 最小 ENV allowlist | `:1117`／`:1144` | 未变 |
| Begin 前 closed | `:1086`／`:1090`／`:1118` | 未削弱；r12 明确 `P6` 失败不进入 `Journal.Begin`（更明确） |
| `restore_unknown` 保守处置 | `:1118`／`:1157`／`:1166` | 未变 |
| Windows 生产 unsupported | `:1086`／`:1139`／`:1145` | 未变 |
| 四项 unknown 仍为 unknown | `:1113`／`:1120`／`:1122` | 未变；未出现「已具备/已任命/已实测」的肯定写法（检索命中的均为否定句） |
| 三级门禁（决策接受／卡证据／生产解除） | `:1122`／`:1124`／`:1126` | 未变；未见门禁自指、二级/三级越界声称 |
| 状态段与追溯行准确 | `:1084`／`:1194` | `:1084` 写明 r12 两路接受尚未取得、S4.24a 及任何实施/验证卡不得启动；`:1194` (i)–(vi) 与实际 diff 逐项对应，引用 arch r05（27,698 B / `98C3DD6F…` / 0/1/2）与 sec r05（24,558 B / `40F5C45D…` / PASS 0/0/1）均属实 |

- 结论：r12 未回退 r11 已闭合项（`ADR-R04-01`–`ADR-R04-04` 的修复文本仍在），未削弱任何无回退底线。

## 5. 范围外登记与待决定

- `tasks.md`（904,142 B / 3,312 行 / `CA20577B…`）：`:3186` 仍为 `- [ ]`；`:3220` 仍写「`adr-0031-r07` 为现行修订……决策接受级对 r07 成立，故本卡启动门禁解除」，与现行 r12 冲突。**范围外观察项，不作本报告判据、不代改。**
- `decisions/README.md`（21,613 B / 110 行 / `1FB7B05A…`）：`:39` 仍写「现行修订 adr-0031-r09（未复评…）」，与 r12 不一致。**范围外观察项，登记给 Lead／知识整理，不代改。**
- 待决定（本对象层面）：无。范围外两项是否由 Lead 安排同步，属另行授权。

## 6. 观察与建议（非 finding，不计数、不改变 PASS）

- `OBS-R06-01`：`:1118` 的并列句式「全文检索……仅命中 `:4072` 的……与 `:4083` 的……」可被读作 `:4083` 也是 `r68.`／`r69.`／`r70.` 的命中；实际带点 token 只命中 `:4072`，`:4083` 是「r70 修订沿用本声明」（无点）。建议后续拆成两个分句。
- `OBS-R06-02`：`:1193`（r11 历史行）仍含「未命中任何编号条款」；按既定口径属历史事实保留，现行规范口径以 `:1118` 为准。
- `OBS-R06-03`：`:1118` 的 `P6`／`Prepare` 枚举只列 `ErrAdmissionIncomplete`／`ErrAdmissionClosed`，未列 `ErrAdmissionBinding`／`ErrAdmissionUnsupported`；前句「按 design `:4377` 的原因分类」已作委派，决策级可判，建议后续补全。
- `OBS-R06-04`：r12 新增固定行号引用（`:4377` 等）在 r70 可核，但设计插入行后会漂移；建议后续优先条款锚，或在引用处注明取得时点。
- `OBS-R06-05`：`V1`–`V11` 的「`enter`／`resume` 均会调用」一句，其对象是 `VerifyRestored` 路径（`:1634`／`:1666`）而非逐个检查点直接调用；建议加「（经 `VerifyRestored`）」以免误读。

## 7. NOT RUN

- build／test／E2E／Linux 侧能力／`SEC-F007`：**NOT RUN**；本轮不运行任何生产入口，不做实现验证。
- 不声称：生产已解锁、`enter`／`resume` 已解锁、约束 8 已满足、`identity-auth-S4.24a` 可关闭或可派发、`adr-0031-r12` 已取得两路接受。
- 不声称任一 unknown 已补齐；不声称任何角色已任命、任何隔离已实测、任何原生回执已取得。

## 8. 实际执行命令（PowerShell 5.1，代表性）

- `Get-FileHash -Algorithm SHA256 -LiteralPath <path>`：复算 framework.md、r11/r12 快照、r05 报告、作者两件证据、design r70、runtime.go、tasks.md、README.md。
- `[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)`：逐行读取与 diff、结构标题、括号计数、行号定位。
- `[IO.File]::ReadAllBytes($p)` + 行首偏移切片 + `MemoryStream` + `Get-FileHash`：按 raw bytes 复算前缀／ADR 段／后缀的长度与 SHA256（不重编码、不归一）。
- `[regex]::Matches(...)`：`Journal.Begin(`／`.Replace(`／`.VerifyRestored(`／`.Prepare(` 全文计数；`r68.`／`r69.`／`r70.` 检索；全角括号计数。
- `Get-ChildItem`／`Test-Path`：目录创建前检查与快照存在性核对。
- `git rev-parse --short HEAD`：确认 HEAD `6da8abb`（只读）。

## 9. 计数

- 原 finding 判定：`ADR-R05-01` CLOSED、`ADR-R05-02` CLOSED、`ADR-R05-03` CLOSED。
- 回归：r11 已闭合项与全部无回退底线未削弱。
- 新增 finding：**BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 0**；开放项 **0**。
- 观察项 5 条（OBS-R06-01 至 05，不计数、不阻断）；范围外登记 2 项（tasks `:3186`／`:3220`、README `:39`）。

## 10. 自身指纹（正文前缀，写盘后复算）
- 口径：本文件第 1 字节起、至本节标题（`## 10.`）首字节前一字节止（即 §0–§9 正文及其后的分隔空行，共 161 行），对 raw bytes 复算 SHA256。
- 正文字节：18903 B；正文行数：161；正文 SHA256：`2532A3310AC9BC5D03A1262578F488C17024A4EE9DA650925273FDDE5DAC1FA4`。
- 完整文件（含本节与末行 Verdict）的最终字节／行／SHA256 见同目录 hashes.txt（hashes.txt 不自指），并由本轮回报给出。

## 11. 结论
- 三项原 finding 全部 CLOSED；无回退；新增 BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 0，开放项 0。
- 本结论只针对 adr-0031-r12 的**决策接受级**充分性；不构成生产解锁、enter/resume 解锁、约束 8 满足、S4.24a 可关闭或可派发，也不代替人的接受。

Verdict: PASS