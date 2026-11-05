# AM-Q2 test-plan r23 作者侧证据补录

> **性质：事后核验补录（post-hoc evidence supplement），非原作者产出。**
> 原作者 `report.md` 未落盘，原作者会话不可恢复（unknown）。本报告不声称复现原作者当时的命令、环境、意图或执行结果；只记录对现存制品的只读核验。

- 核验日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`
- 唯一写入：`.spec\wip\article-management\evidence-am-q2\test-plan-r23\report.md`
- 未执行：任何测试、构建、服务启动、E2E、性能测量或 git 写操作
- 总体结论：六条 TR-22 finding 均为 `ADDRESSED`；用例定义集前后不变；test-plan r23 声明与其初次核验时的 design r30 主锚点一致。核验期间 `design.md` 被外部更新为 r31、`test-plan.md` 被外部更新为 r24；本报告对象仍为 r23，r23 全文已由后续 r24 改前副本复核，详见 §1.1 与 §3.1。真实环境性能、容量和独立评审仍未验证。

## 1. 对象与改前/改后

被测对象是 test-plan r23。初次核验时它位于 `.add/project/knowledge/design-workspace/article-management/test-plan.md`；核验期间该路径被外部更新为 r24，r23 全文随后由 `.spec/wip/article-management/evidence-am-q2/test-plan-r24/before.md` 复核，字节数和 SHA256 与初次核验完全一致。改前副本是 `.spec/wip/article-management/evidence-am-q2/test-plan-r23/test-plan-before.md`。以下字节数、LF 行数和 SHA256 均由本机文件实测，不是抄录任务描述。

| 对象 | revision | 字节数 | LF 行数 | SHA256 |
|---|---|---:|---:|---|
| `test-plan-before.md`（改前） | `article-management-test-plan-ir01-r22` | 324805 | 1680 | `76584E9278E0542151AE1DFADC61B785DFAF1160976DCC0CAC932775093BA217` |
| `test-plan.md`（改后 r23；初次核验时现行） | `article-management-test-plan-ir01-r23` | 326550 | 1680 | `C1923CC43CD5A5E28A5C1512B640AB961F0AE25A155CEE34F0AFDA01BD192AB9` |
| `design.md`（锚点对象；初次实测 r30） | `article-management-design-ir01-r30` | 382472 | 3294 | `5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863` |

r22 与 r23 两份 test-plan 均为 UTF-8 无 BOM、CR=0、LF=1680、末尾 LF。r23 相对 r22 增加 1745 字节，行数不变。


原目录中的作者侧残留证据为 `apply-r23.sh`（290 B，SHA256 `FE97BFA8A16F39162AF4EA0BBB9C85F1945ED3F7C436FCDF01680ADF9D1BF372`）和 `r23.patch`（9412 B，SHA256 `ABAB1CA9E357CC0A0C61B2BBA75C21958D24329E162FAB5593BEA31E5D6D4BF4`）。patch 有 13 个 hunk、15 个 `-` 行和 15 个 `+` 行；逐行比较只发现 15 个变化行，未发现 patch 之外的其他文本变化。

变化行为：`5, 6, 7, 29, 83, 106, 108, 172, 275, 590, 883, 1210, 1247, 1653, 1662`。按用途分为：

- 元数据/评审标签：L5、L6、L7、L29、L83。
- 现役叙述：L106、L108、L1662（TR-22-05）。
- finding 落点：L172（TR-22-01）、L275（TR-22-02）、L590（TR-22-03）、L1653（TR-22-06）。
- AM-Q2 同步：L883、L1210、L1247。

L83 的 `source_test_plan_review: pending-independent-review-r23` 仍是待独立评审状态，不能把本次补录当作评审结论。

### 1.1 核验期间的并发变化

核验期间检测到两条外部并发变化，均非本会话写入：

- `test-plan.md`：初次核验时为 r23（326550 B、1680 LF、SHA256 `C1923CC43CD5A5E28A5C1512B640AB961F0AE25A155CEE34F0AFDA01BD192AB9`），之后于 2026-11-02 14:03:11 变为 r24（326640 B、1680 LF、SHA256 `4C517E90B105D3C37C70F3430A71262AF9088B9D18FEE442640908740F39A34F`）。
- `design.md`：初次核验时为 r30（382472 B、3294 LF、SHA256 `5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863`），之后于 2026-11-02 13:59:31 变为 r31（382562 B、3294 LF、SHA256 `63FD0691E1D66326B8EED7CFDFD8017417C60BB628CB961359087242D5D58B64`）。

r23 全文已由并发 r24 改前副本 `.spec/wip/article-management/evidence-am-q2/test-plan-r24/before.md` 独立复核：326550 B、1680 LF、SHA256 `C1923CC43CD5A5E28A5C1512B640AB961F0AE25A155CEE34F0AFDA01BD192AB9`，与初次核验完全一致。本报告所有 r23 行号和原文均在该副本上复核；本报告不评价 r24。

## 2. 六条 finding 逐条核验

判定口径：`ADDRESSED` 只表示 finding 所指的作者侧文档缺口在 r23 文本中有直接、可复核的落点；不表示测试执行、性能达成、独立评审通过或人的成果接纳。下列摘录均为 `test-plan-r24/before.md`（r23 全文副本）连续原文节选，均不超过 120 字符。

### TR-22-01 — ADDRESSED

- 原锚点：`test-plan.md:172`；现落点：`test-plan.md:172`。
- 现文本（94 字符）：

> AM-Q2 已确认分页 20/100/200、首期文章总量 100000、单用户上限 10000，P95 首期验收基线（开发机测量参考，非目标环境结论）为列表≤300ms、详情≤150ms

- 改前同位置实测：`容量负载、时延、留存和部署参数仍保持 TBD`。
- 结论：AM-Q2 的分页、总量、单用户上限和 P95 开发机基线已进入成功证据，同时保留目标环境 TBD；原缺口已覆盖。

### TR-22-02 — ADDRESSED

- 原锚点：`test-plan.md:275`；现落点：`test-plan.md:275`。
- 现文本（同一行两段，85 字符和 35 字符）：

> AM-Q2 分页 20/100/200、首期规模 100000/10000 与 P95 首期验收基线（开发机参考，非目标环境结论）列表≤300ms/详情≤150ms 已定

> 其他性能目标、目标环境真实 P95/并发/数据分布及告警阈值仍 TBD

- 改前同位置实测：`目标值 TBD`。
- 结论：性能层已写入 AM-Q2 已定值和仍未定的目标，覆盖了原 finding 的“目标值 TBD”缺口。

### TR-22-03 — ADDRESSED

- 原锚点：`test-plan.md:590`；现落点：`test-plan.md:590`。
- 现文本（103 字符）：

> QueryConfig用显式测试值并标为非生产默认；AM-Q2 已确认 production 部署基线 DefaultLimit=20、MaxLimit=100、CandidateBatchSize=200

- 改前同位置实测：`生产参数、容量/P95仍TBD`。
- 结论：production 部署基线、首期容量和 P95 基线已写入，并明确测试值不是生产默认；原缺口已覆盖。

### TR-22-04 — ADDRESSED

- 原锚点：`test-plan.md:29` / `test-plan.md:5`；现落点：`test-plan.md:29` / `test-plan.md:5`。
- 现文本：

> revision: article-management-test-plan-ir01-r23

> last_updated: 2026-11-02

- 改前同位置实测：`revision: article-management-test-plan-ir01-r22`；`last_updated: 2026-10-29`。
- 结论：revision 和最后更新时间均已更新到 r23 时点；原缺口已覆盖。

### TR-22-05 — ADDRESSED

- 原锚点：`test-plan.md:106` / `test-plan.md:108` / `test-plan.md:1662`；现落点同三行。
- 现文本（三处，82、82、74 字符）：

> 本规划 r23 在 r21[恢复锚定与发布规划](#recovery-runtime-r21)基础上，同步 design r30 的 AM-Q2 容量与查询参数口径

> 本规划状态为 **planned；r23 作者侧同步 design r30 的 AM-Q2 容量与查询参数口径并完成 TR-22 文档 finding 的作者侧修订

> 本规划 r23 已由作者同步 design r30 的 AM-Q2 容量与查询参数口径，并按 TR-22 finding 修订现役表述，待非作者复审

- 改前同位置实测：L106 以 `本规划 r21 仅新增` 开头；L108 仍以 `r20仅补` 叙述；L1662 仍写 `当前r21只完成恢复规划文档修订`。
- 结论：三处现役落点均已改为 r23/design r30；r21/r20 只保留为历史/时点记录，不再作为这三处的现役结论。

### TR-22-06 — ADDRESSED

- 原锚点：`test-plan.md:1653`；现落点：`test-plan.md:1653`。
- 现文本（61 字符）：

> | AM-Q2 分页/规模/性能目标 | partial | 算法与正确性可测；limit、批次和 P95 目标被阻塞 |

- 改前同位置实测：同一行状态为 `planned`，并写 `目标环境未实测`。
- 结论：状态改为 `partial`，且明确 limit、批次和 P95 目标被阻塞；与未实测事实一致。

六条汇总：TR-22-01、TR-22-02、TR-22-03、TR-22-04、TR-22-05、TR-22-06 全部为 `ADDRESSED`；本次核验没有判定为 `PARTIAL` 或 `NOT ADDRESSED` 的 finding。

## 3. 设计锚点核验

r23 文本 L7（初次核验时为 `test-plan.md:7`，现由 `test-plan-r24/before.md:7` 复核）的原文是：

> r23_source_design_artifact: "sha256:5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863; bytes:382472; lines:3294"

初次核验时 `design.md` 实测为 revision `article-management-design-ir01-r30`、382472 字节、3294 LF 行、SHA256 `5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863`；最终复核时该文件已被外部更新为 r31，见 §3.1。r23 L6 的 `r23_source_design_revision` 也是 `article-management-design-ir01-r30`，与初次核验时的 `design.md:5` 一致。

| 核验项 | test-plan 声明 | design 初次实测（r30） | 结果 |
|---|---|---|---|
| SHA256 | `5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863` | 同左 | 一致 |
| 字节数 | 382472 | 382472 | 一致 |
| LF 行数 | 3294 | 3294 | 一致 |
| revision | `article-management-design-ir01-r30` | `article-management-design-ir01-r30` | 一致 |

### 3.1 核验期间 design 并发变化

| 时点 | revision | 字节数 | LF 行数 | SHA256 |
|---|---|---:|---:|---|
| 初次核验 | `article-management-design-ir01-r30` | 382472 | 3294 | `5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863` |
| 最终复核 | `article-management-design-ir01-r31` | 382562 | 3294 | `63FD0691E1D66326B8EED7CFDFD8017417C60BB628CB961359087242D5D58B64` |

`design.md` 最终已是 r31，最后一次写入时间为 2026-11-02 13:59:31；该变化非本会话写入，本会话未修改 `design.md`。因此 r23 L7 只与初次实测的 design r30 一致，不再与最终复核时的当前 design r31 一致。design r30→r31 是否要求 test-plan 从 r23 升版，属于后续判断，本次补录状态为 `unknown`，不作结论。

对 r23 全文做大小写不敏感检索：`90AD2BC1` 命中 0 次，`r29` 命中 0 次，`381381` 命中 0 次，`3290` 命中 0 次。因此未发现 r29 旧 SHA 或旧尺寸锚残留。改前 L7 的 r22 锚为 `90AD2BC1...`，patch 已将其替换为 r30 锚。

仍存在但不属于 `90AD2BC1`/r29 旧锚的内容：

- L8–L20 的 `historical_r20_*` 指向 design r22，是历史时点记录。
- L30–L31 的通用 `source_design_revision: article-management-design-ir01-r23` 与 SHA `2223BE4D...` 是恢复切片锚；L106 已声明这些通用字段只指新恢复切片。
- L37–L41 的 `source_design_r21_*` 指向 design r21，是 r21 历史锚。

本报告只回答 r23 中 `90AD2BC1`/r29 旧锚是否残留；不把上述历史/切片锚重新解释为 r30，也不据此宣称所有 source_design 字段都已统一。

## 4. 用例集合未变的证据

计数均在 UTF-8 原文上、区分大小写执行。用例定义行使用 `(?m)^\|\s*(A-(?:[A-Z]+-)*\d+)\s*\|`，全文引用使用 `\bA-(?:[A-Z]+-)*\d+\b`，Case 使用 `\bCase\d+\b`；状态词使用 `\b` 词边界匹配。

| 指标 | 改前 r22 | 改后 r23 | 差值 | 说明 |
|---|---:|---:|---:|---|
| 用例定义行（表首列 A-*） | 112 次 / 110 唯一 | 112 次 / 110 唯一 | 0 / 0 | ID 集合对称差为空 |
| 全文 A-* 数字引用 | 244 次 / 110 唯一 | 244 次 / 110 唯一 | 0 / 0 | 含覆盖映射和叙述引用 |
| `CaseNN` | 7 次 / 1 唯一 | 7 次 / 1 唯一 | 0 / 0 | 唯一值为 `Case18` |
| `planned` | 28 | 28 | 0 | L108 词边界 +1，L1653 状态 -1 |
| `blocked` | 18 | 18 | 0 | 无变化 |
| `PASS` | 71 | 71 | 0 | 无变化 |
| `partial` | 7 | 8 | +1 | 仅 L1653 历史 Gate 行由 `planned` 改为 `partial` |
| `NOT RUN` | 9 | 10 | +1 | 仅词边界/空格计入差异；原始子串均为 24 次 |
| `not-executed` | 25 | 25 | 0 | 无变化 |
| `unknown` | 40 | 40 | 0 | 无变化 |

状态词差异逐条说明：

1. `partial +1`：L1653 的 AM-Q2 历史 Gate 行由 `planned` 改为 `partial`。该行是 Gate 记录，不是新增或删除测试用例；对应 TR-22-06 的修订。
2. `NOT RUN +1`：L1662 改前为 `新增验证NOT RUN`，中文紧邻导致 `\bNOT RUN\b` 不成立；改后为 `新增与性能验证仍 NOT RUN`，空格分隔后计入。原始子串 `NOT RUN` 前后均为 24 次，说明没有新增该状态文本。
3. `planned` 总数不变：L1653 的 `planned` 减 1；L108 由 `全部planned-not-executed` 改为 `全部 planned-not-executed`，词边界使 `planned` 加 1。两者都不是测试用例状态变化。

辅助广义 ID-like 扫描 `\b[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+\b` 得到 574→588 次、212→215 个唯一值；新增唯一值只有 `AM-Q6`、`AM-Q8`、`TR-22`，均为需求/finding 引用，不是测试用例 ID；没有删除唯一值。

A-PERF 五项的前后状态均为：

| ID | 改前 | 改后 |
|---|---|---|
| A-PERF-01 | partial | partial |
| A-PERF-02 | planned | planned |
| A-PERF-03 | blocked | blocked |
| A-PERF-04 | partial | partial |
| A-PERF-05 | blocked | blocked |

因此，本次 r23 文字修订没有增删测试用例 ID，也没有改变 A-PERF 用例状态；`partial` 和 `NOT RUN` 的计数变化来自 Gate/词边界，不是用例集合变化。

## 5. 未验证项与阻塞

- 目标环境真实 P95、并发、数据分布、告警阈值、retention、RPO/RTO：仍是 `TBD` / `NOT RUN` / `unknown`。
- 100000 篇文章、单用户 10000 篇的首期容量：只有规划和开发机基线，没有目标环境容量实测。
- 单元、集成、契约、性能、安全和 E2E 执行：本次补录未运行任何测试、服务或真实依赖，不能宣称通过。
- 原作者的实际命令、执行环境、运行输出和原始报告：`unknown`；本报告只能核验残留 patch、改前副本、初次核验时的现行文件与后续稳定副本。
- 独立评审：L83 仍为 `pending-independent-review-r23`，未取得独立评审结论。
- 对象漂移：核验期间 `test-plan.md` 已由 r23 外部变为 r24、`design.md` 已由 r30 外部变为 r31；test-plan r23 仍锚 r30。r31 是否要求 test-plan 升版、r24 是否完整继承并保持 r23 的 TR-22 结论，均超出本次补录范围，状态为 `unknown`。
- 人的成果接纳、产品验收和签字：`unknown` / 未提供。
- 本次未新增或变更 tasks、plans 或其他任务登记；没有新的测试/E2E 任务登记引用。

## 6. 非真人签字与边界

**非真人签字。** 本报告由 Codex 作为事后核验补录生成，不是原作者、不是独立评审人、不是测试执行人或产品验收人；不构成人的成果接纳、评审通过、发布批准或真实环境验证通过。

本次只读核验了 `test-plan-before.md`、r23 初次核验态 `test-plan.md` 及其后并发 r24 的改前副本 `test-plan-r24/before.md`、`design.md` 的 r30/r31 两态、`r23.patch` 和 `apply-r23.sh`；唯一写入为本 `report.md`。未修改 `test-plan.md`、`design.md`、`tasks.md`、`plans/*` 或其他文件，未执行 git 写操作。
