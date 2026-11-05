# ADR-0031 现行修订登记同步报告（tasks.md:3220 / decisions/README.md:39）

- 日期：2026-11-02（Asia/Shanghai）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 角色：`add-task-planner`（方法 `add-task-decompose`）；性质：**只同步登记**，未派发、未勾选、未改任务范围
- 对象：`.add/project/knowledge/tasks/tasks.md` 的 S4.24a 卡面 `:3220`；`.add/project/knowledge/decisions/README.md` 的 `:39`

## 1. 输入事实（本轮现场复核）

| 对象 | 期望值 | 实测 | 结论 |
|---|---|---|---|
| `framework.md` | 297,412 B / 1,275 行 / `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20` | 同值 | 一致 |
| `review-arch-r06/report.md` | 19,778 B / 171 行 / `705806BABF4C65A352E9442D43AD5D1D060A135CCA9C195A89C959D7650163C7` / PASS 0/0/0/0 | 同值 | 一致 |
| `review-sec-r06/report.md` | 25,883 B / 218 行 / `12D55EBCCA5282FF703BBAE10F42A5601429E290EF8B5F02C04368AB8BCDF08A` / PASS 0/0/0/1（`ADR-S06-01` OPEN） | 同值 | 一致 |
| `acceptance-adr-0031-r12.md` | 4,848 B / 46 行 / `F04E65668A80E9796FD7BEF057C2CBE92817FFC0D1CB9018D7E140A30368497D` | 同值 | 一致 |
| `design.md` | 1,103,152 B / 5,476 行 / `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`（r70） | 同值 | 一致 |

`framework.md` 锚点已读原文：`:1084`（状态段「本修订 `adr-0031-r12` 的两路接受尚未取得」）、`:1129`（决策接受级随 `**修订**` 行现行修订号自动生效）、`:1174`（接受只解除派发／认领／开始改动限制）、`:1194`（r12 追溯行，同含产出时点表述）。

## 2. 改前／改后

| 指标 | tasks.md 改前 | tasks.md 改后 | README.md 改前 | README.md 改后 |
|---|---:|---:|---:|---:|
| 字节 | 904,142 | 904,893 | 21,613 | 22,457 |
| 行数（`ReadAllLines`） | 3,312 | 3,312 | 110 | 110 |
| CR | 3,312 | 3,312 | 2 | 2 |
| LF | 3,312 | 3,312 | 110 | 110 |
| CRLF | 3,312 | 3,312 | 2 | 2 |
| bare LF | 0 | 0 | 108 | 108 |
| BOM | 无 | 无 | 无 | 无 |
| 末尾换行 | CRLF | CRLF | CRLF | CRLF |
| SHA256 | `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C` | `696A5B0EE9B2E4DA75D4C50EB82FCFC0E68D657500BDED346F4CA765822BD28E` | `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2` | `17F0C013810993F3CD751C08872FD43429C1BBA78FBEE3BE3EC44CB2861E2478` |

说明：`decisions/README.md` 基线本身即混合换行（2 个 CRLF＋108 个 bare LF），本次改写不触碰任何换行字节，改后计数与基线逐项相同；未新增或删除行。两文件均为 UTF-8 无 BOM。

## 3. 改动行与摘要

| 文件 | 行 | 摘要 |
|---|---|---|
| `tasks.md` | `:3220` | 硬/软 requires 段就地重写：清除「`adr-0031-r07` 为现行修订……故本卡启动门禁解除」的现行结论，改为 `adr-0031-r12` 现行＋两路 r06 PASS＋接受登记＋产出时点口径＋限定＋design r70／r71 现状；硬 requires 边仍为 `identity-auth-D06`，软关联 S4.22 表述不变。 |
| `decisions/README.md` | `:39` | ADR-0031 状态单元格：「现行修订 adr-0031-r09（未复评；……）」改为「现行修订 adr-0031-r12」＋两路 r06 PASS 事实＋接受登记＋仍 closed 限定；7 列表结构与管道符数量不变（8 个 `|`）。 |

## 4. 写入的事实（逐条）

1. ADR-0031 现行修订为 `adr-0031-r12`；`framework.md` 297,412 B / 1,275 行 / `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`。
2. 架构侧接受：`.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r06/report.md`，19,778 B / 171 行 / `705806BABF4C65A352E9442D43AD5D1D060A135CCA9C195A89C959D7650163C7`，PASS，BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 0，开放项 0。
3. 安全侧接受：`.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r06/report.md`，25,883 B / 218 行 / `12D55EBCCA5282FF703BBAE10F42A5601429E290EF8B5F02C04368AB8BCDF08A`，PASS，BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 1（`ADR-S06-01`，OPEN，属 design 侧错误码一致性，安全影响为零）。
4. 接受登记：`.spec/wip/recovery-runtime/evidence-b13j-d06/acceptance-adr-0031-r12.md`，4,848 B / 46 行 / `F04E65668A80E9796FD7BEF057C2CBE92817FFC0D1CB9018D7E140A30368497D`。
5. 时间口径：`framework.md:1084` 与 `:1194` 的「`adr-0031-r12` 的两路接受尚未取得」属产出时点表述；现行事实以接受登记与 `framework.md:1129`「随现行修订号自动生效」机制为准。
6. design 现状：现行 `identity-auth-design-ir01-r70`（1,103,152 B / 5,476 行 / `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`）；`r71` 正在修订 4 项（policy `tool.receipt.trust_anchor_ref` 必填化、`:4070` 的 Begin 门禁按路径更正、`V7`／`V8` 前重验登记、`:3944` 与 `:4377` 错误分类收敛），须经设计双复审后方可谈实施派发。

## 5. 删除的作废表述

- `tasks.md:3220` 原「`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受」——改后 0 命中。
- `tasks.md:3220` 原「故本卡启动门禁解除」及「r07／r01 报告」作为现行依据的整段——改后 0 命中；仅保留「r06／r07 的接受事实只作历史，不再作为『现行修订已接受』的依据」。
- `decisions/README.md:39` 原「现行修订 adr-0031-r09（未复评；外部独立见证签名；模拟见证未真实验证）」——改后 `adr-0031-r09`、`未复评` 均 0 命中。

## 6. 保留的限定（逐条）

- 该接受只解除实施/验证卡的**派发、认领与开始改动**限制（`framework.md:1174`）。
- 不构成人的接纳；不补齐四项 unknown。
- 不解除生产 `enter`／`resume` closed；生产 DB 替换继续关闭。
- 不表示 ADR-0031 约束 8 全项满足；不使 `identity-auth-S4.24a` 可关闭。
- 本卡仍未派发、未认领、未启动；`tasks.md:3186` 仍为 `- [ ]`。

## 7. 只读与边界声明

- 仅修改上述两行：`tasks.md:3220`、`decisions/README.md:39`。逐行比对证明 `tasks.md` 3,312 行中唯一差异行为 `:3220`，`README.md` 110 行中唯一差异行为 `:39`；两文件的目标行前后字节段 SHA256 与基线**完全相同**（`tasks.md` 前缀 874,710 B / `8F54AF681943FDACFB3A7BED168F93D4473461A0634D203808CBE7E7A58F71CA`，后缀 27,699 B / `48EEA6B42275F56666CA6B9501F63902EEDA91ED2FB3DE32CBF85B6BF0EF72AF`；`README.md` 前缀 8,306 B / `5903E1322CE75529055A6BCF62F8B06F67451670EB7AE27AC76C33ADA8FBEA2C`，后缀 12,469 B / `69EA429BDBE068C76D659C754D3D494B737B9B4F2C6440A6345BBC726CD1490A`）。
- 未勾选 `tasks.md:3186` 或任何复选框；未改任务数量、编号、范围、关闭条件、负向 oracle、停止条件；硬 requires 边仍为 `identity-auth-D06`。
- 未修改 `framework.md`、`design.md` 或任何其他行；未执行任何 Git 写操作；未删除、移动、归档任何文件；未触碰 `.spec/wip/article-management/`、`.spec/wip/identity-auth/`。
- 本轮新建：本报告、`hashes.txt`、`tasks.md.baseline`、`decisions-README.md.baseline`（均在 `task-sync-r11/` 下）。

## 8. NOT RUN

- 未运行 `gofmt`、`go build`、`go vet`、`go test`、`-race`、integration tag 或任何 E2E／Linux 实测。
- 未派发任何 Agent，未启动任何实施或验证活动；未任命批准角色、未实测执行身份或宿主隔离、未取得原生耐久回执。
- 本报告只做文本改写与只读核验（`Get-FileHash`、`ReadAllLines`、逐行比对、`rg`）。

## 9. 基线快照

| 文件 | 大小 | SHA256 |
|---|---:|---|
| `task-sync-r11/tasks.md.baseline` | 904,142 | `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C` |
| `task-sync-r11/decisions-README.md.baseline` | 21,613 | `1FB7B05A6673043B3C959ADC9E43BA8F58240921AE9AC8DB77D0C09F30681DD2` |

## 10. 未决项

1. design `r71` 的 4 项修订尚未完成双复审，完成前不谈实施派发。
2. `ADR-S06-01`（LOW，OPEN）：`:3944` 与 `:4377` 的不可读错误分类相反，须由 design 写集裁决。
3. `framework.md:1084`／`:1194` 仍为产出时点表述（本轮范围外未改）；是否闭合 `N005` 由架构决策方与 Lead 判定。
4. 生产 `enter`／`resume` 仍 closed；ADR-0031 约束 8 未满足；四项 unknown 仍为 unknown。
5. 携带项 `ADR-S04-01`（design `:4031`／`:4062` 的「若该键存在」）随 `r71` 一并处理。

## 11. 复核命令与结果

| 检查 | 结果 |
|---|---|
| `tasks.md` 字节／行 | 904,142 → 904,893；3,312 → 3,312 |
| `tasks.md` SHA256 | `CA20577B…` → `696A5B0E…` |
| `decisions/README.md` 字节／行 | 21,613 → 22,457；110 → 110 |
| `decisions/README.md` SHA256 | `1FB7B05A…` → `17F0C013…` |
| 逐行差异 | 仅 `tasks.md:3220`、`README.md:39` |
| 目标行前后字节段 | 前缀／后缀 SHA256 与基线相同 |
| 换行／BOM | `tasks.md` 全 CRLF、无 BOM、末尾 CRLF；`README.md` 混合换行计数与基线相同、无 BOM、末尾 CRLF |
| `tasks.md:3186` | `- [ ] identity-auth-S4.24a …`（未勾选） |
| 作废表述 | `adr-0031-r07` 现行结论、`故本卡启动门禁解除`、`adr-0031-r09`、`未复评` 均 0 命中 |
