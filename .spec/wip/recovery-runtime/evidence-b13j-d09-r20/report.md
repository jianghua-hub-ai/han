# ADR-0030 r20 作者证据（r03 两路复审 6 项残留文本收口）

## 1. 角色、授权与边界

- 角色：Arch-Designer（作者角色本体），对既有 ADR-0030 做 r20 修订；工作性质为既有 ADR 的作者返工，不新建工作、不重新路由。
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abb`。本轮未执行任何 Git 写操作。
- 写集：仅 `framework.md`（ADR-0030 正文）与本报告；未改 `decisions/README.md`、`tasks.md`、两份 design.md、任何 `*.go`／`migrations/`／`cmd/`／`internal/` 或其它代码。
- 未回显、未落盘任何凭据。
- 本修订为作者文档处置，不构成评审 PASS、人的接纳或生产就绪声明。

## 2. 输入与基线

| 输入 | 字节 | SHA256 |
|---|---:|---|
| 架构 r03 `review-arch-r03/report.md` | 22,404 | `763459D8D8F1AC179EF9438E6D039B93A10F5E2FDE6EEDE39AF049136C8E1B8C` |
| 安全 r03 `review-sec-r03/report.md` | 24,209 | `951921836EF102A0E8F89213CD613385F723C811D6E5C62466894FB09B9CD85B` |
| Lead 索引登记 `lead-index-sync-f010.md` | 2,503 | `AE6892F482FB269FE73D597E3CA2CD88E08E75B567B88CBB22DE3115BF616718` |
| r19 改前副本 `%TEMP%\framework-r18-backup.md` | 370,555 | `4EC6C2AF086BEC4ABB59E5DE180CC0323FA13EBD40D532098B5B271FDE1A148C` |
| r19 作者证据 `evidence-b13j-d09-r19/report.md` | 10,906 | `A15326D60B780D9E46E06C5099EF67906A3D898CDF086D60692919C702FD1604` |

改前 `framework.md`：376,429 B / 1,597 段 / CRLF 1,596 / 独立 LF 0 / UTF-8 无 BOM / SHA256 `7C2E8A38E7CA92AB2AA9DA3111C3271745EA5E862F6F2316EF1E74676BCEB50C`。

## 3. 本轮改动与落点（改后行号）

| 项 | finding | 落点 | 内容 |
|---|---|---|---|
| 1 | `B13J-D09-ARCH-R03-F015`（MEDIUM） | `:1280` | `waived-for-test-scope` 专段末句改为「`tasks.md` 侧不加注记，requires 原文即权威；本 ADR 不改 `tasks.md`，故不存在 ADR 与任务卡的双权威。」`tasks.md` 注记口径统一为：不加注记、`:649` requires 原文即权威、`waived-for-test-scope` 是范围豁免而非 requires 变更、本 ADR 不改 `tasks.md`。`:1272`／`:1341` 既有表述未改结论。 |
| 2 | `B13J-D09-ARCH-R03-F016`＋`B13J-D09-SEC-R03-F001`（LOW） | `:1325`、`:1345` | `:1325` 对 `preflight-record.md` §9 `:122` 的引文补全句尾「；不再以「结构性不可满足」维持四卡阻塞」，成为整句逐字引用（内层引号按原文保留为「」）；`:1345` 同步注明 r19 时点仍为句首片段、句尾由 r20 补全。 |
| 3 | `B13J-D09-SEC-R03-F002(a)`（LOW） | `:1353` | README 描述改为时点表述：r18 `:1327` 所述为 r18 时点状态，已由 Lead 于 r19 之后完成索引收口（`lead-index-sync-f010.md`，README 现为 22,942 B / `53FD5846C2FE2BEEBB9B8BDCCF1E5239C469B79588E3965B68196550C4BEB328`）。 |
| 4 | `B13J-D09-SEC-R03-F002(b)`（LOW） | `:1333` | 安全侧 r02 条件项由误记的「F010」更正为 `B13J-D09-SEC-R02-F001`，并写明架构侧 F010 与安全侧 `B13J-D09-SEC-R02-F001` 是两件事、两者均已收口；架构侧 F010 历史记录未改。 |
| 5 | `B13J-D09-SEC-R03-F003`（LOW） | `:1377`–`:1381`（代码块原文在 `:1380`） | 在 r20 小节内以代码块逐字保留 r19 删除的 r17 旧 `:1234` S3.6 模拟替代表行原文，并注明「该行于 r19 按 Lead F001 裁定删除，此处保留原文仅为可追溯，不恢复其效力」。 |

- 修订串 `:918`：`adr-0030-r19` → `adr-0030-r20`。
- r20 小节：`:1355`–`:1387`（标题行 `:1355`；`:1388` 为小节后的空行；`:1389` 为 `### 约束`）。
- r20 追溯行：`:1431`（r19 追溯行仍为 `:1430`，r01–r19 追溯行未改）。
- 被删行逐字核对：与 `%TEMP%\framework-r18-backup.md` 第 `:1234` 行逐字符相等（262 字符），未改写。

## 4. 改后格式与哈希

- 字节：384,383 B
- 段数：1,632（CRLF 1,631 + 1）
- CRLF：1,631；独立 LF：0；独立 CR：0
- BOM：无；U+FFFD：0
- SHA256：`F0655B2A1D5F8F56297DB0FA7A26FE3070F3D6B0478F15CD730C27EBB3F55E80`

## 5. 受保护内容核对

- `decisions/README.md`：22,942 B / `53FD5846C2FE2BEEBB9B8BDCCF1E5239C469B79588E3965B68196550C4BEB328`（未写）。
- `tasks.md`：912,450 B / `E034155076F77F8D1E8288D9BB4014554A5019882051A03A305A0ACA6AD71EFB`（未写；`:649` requires 仍含 `identity-auth-S4.5`、`identity-auth-S4.6`）。
- `preflight-record.md`：11,705 B / `2FEFBF074F753D53CBB1779A6D26BCBCD7EA470F1563BA0A16A68AEDEAB1F05F`（未写）。
- ADR-0025／0028／0031／0032、约束 1–10 正文、r17 决策 3.3 段的 r19 更正、r01–r19 追溯行：逐字节未变（由下节逆变换整体证明）。
- 两份 design.md 与全部代码未写。

## 6. 逆变换验证

对改后文件执行逆向操作：删除 r20 小节（`:1355`–`:1388`）与 r20 追溯行，按行内定点还原 6 处文本与修订串（`:918`、`:1280`、`:1325`、`:1333`、`:1345`、`:1353`）。

- 逆变换结果：376,429 B / 201,782 字符 / SHA256 `7C2E8A38E7CA92AB2AA9DA3111C3271745EA5E862F6F2316EF1E74676BCEB50C`
- 与改前基线一致：是。
- 说明：逆变换按行定位，避免误伤 `:1341` 中原本就存在的同一口径句。

## 7. 未完成与未宣称事项

- 本 r20 未取得架构侧与安全侧两路复审接受；r03 架构侧 PASS WITH CONDITIONS（F015 条件项）与安全侧 ACCEPT 均只覆盖 r19 字节。
- 生产 `enter`／`install`／`resume`／`unlock` 仍全部 closed；约束 7／9／10 与 ADR-0031／0032 不解除；B1／B3／B4／B5 仍为产能外 unknown。
- 不宣称评审 PASS、人的接纳、生产就绪或真实环境验证通过。