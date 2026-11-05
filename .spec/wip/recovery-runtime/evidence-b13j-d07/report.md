# B13J D07 作者证据 — ADR-0032 修订至 adr-0032-r04

- 作者角色：add-arch-designer（Arch-Designer，Singer）。文档型决策卡，不含生产实现。
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（改前/改后实核未变）。
- 日期：2026-11-02。
- 性质：**作者文档处置**。不是架构或安全独立评审 PASS，不是人的成果接收，不是生产批准，不是 Git 写授权。

## 1. 卡面与授权输入

- 卡面：`.add/project/knowledge/tasks/tasks.md:3089`–`:3115`（`identity-auth-D07`）。
- Lead 授权：`.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r09.md` §1「B」；`.spec/wip/recovery-runtime/evidence-b13j-d07/lead-authorization-r01.md` §2/§4/§6。
- 规划证据：`.spec/wip/recovery-runtime/evidence-b13j-d07/planning-report.md`。

## 2. 写集与改前快照

写集（严格 2 文件，均已核对）：

- `.add/project/knowledge/decisions/framework.md`——仅 ADR-0032 段。
- `.spec/wip/recovery-runtime/evidence-b13j-d07/report.md`——本文件（新建）。

改前快照：`%TEMP%\framework-r10-before-d07.md`（逐字节复制，235729 B / `B4FA4BA8F017732325E0A58BEA152EDDDEBB7788422FB578E6C122FEC99A2402`）。

## 3. 全文改前/改后实测

| 指标 | 改前 | 改后 |
|---|---|---|
| 字节 | 235729 | 237373 |
| 行数（含末尾空行前的物理行） | 1240 | 1241 |
| SHA256 | `B4FA4BA8F017732325E0A58BEA152EDDDEBB7788422FB578E6C122FEC99A2402` | `863E0A2F82427CDB604A91AA182A7B75651BB475ECAF769743E4FCC32C75F73C` |
| CRLF 数 | 1240 | 1241 |
| 裸 LF | 0 | 0 |
| 孤立 CR | 0 | 0 |
| BOM | 无 | 无 |
| TAB | 0 | 0 |
| 其它控制字符 | 0 | 0 |
| 末尾 | 单个 CRLF | 单个 CRLF |

段指纹格式为 `字节数:SHA256`，口径：按 UTF-8 无 BOM 取指定行区间、行间以 CRLF 连接、末尾单个 CRLF。

| 区段 | 改前行号 | 改前指纹 | 改后行号 | 改后指纹 |
|---|---|---|---|---|
| ADR-0019 区间（字节 31305–126598） | 固定字节区间 | `95294:607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6` | 同一字节区间 | `95294:607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6`（未变） |
| 前缀（ADR-0032 之前全部） | L1–1163 | `217469:2AA5D5FE49021CAF39D89507CCF50EA08500C59527E602001E24091EFE825763` | L1–1163 | 同值（未变） |
| ADR-0032 段 | L1164–1230 | `14938:E34FF7E61DD25BBE875F2572F1E040B551417941509588963EA2B35F13E250AA` | L1164–1230 | `15195:8F3009E60E7BD1883C5623B53C4377A3AC16466DCF2C78863224D1035304D564` |
| 解除顺序区 | L1218–1226 | `2047:E79D079A058CD3DEE7BEAACD5A808DC63E01CF7FD6A92AFB5E2035A2FB2C1223` | L1218–1226 | `2304:98A4EA9222950C145112222B59A4E15C444145AE45082E68D4F922C796345516` |
| 不变量 1–9 | L1199–1217 | `4157:4A757370D9365EA56ADD63C9F147158DB1B81893A499030EE1CE5C8ACA1FC41F` | L1199–1217 | 同值（逐字未变） |
| 解除条件 1–7 | L1211–1217 | `1111:DABED6F69BB2E4DD4F82C63F60EE5BF8A966F67CED23CB61A848D48ED55CD256` | L1211–1217 | 同值（逐字未变） |
| 引用记录区 | L1232–1240 | `3320:11155A848A3317B530F53CDEE96E5D7B72BF9205ECAF377F3FC1B1CAC49D4DA6` | L1232–1241 | `4707:484F30686DC7B13AC21A67ACDAE5170DC9256F2BFD7EB44668C1425B18986730` |

前缀 L1–1163 与 ADR-0019 区间指纹改前后完全相同，即 ADR-0019/0021/0029/0030/0031 全文与 ADR-0032 之外的一切逐字节未动。

## 4. 逐项改动对照（行号 → 改前 → 改后）

| 行号 | 改动类别 | 改前 | 改后 |
|---|---|---|---|
| `:1167` | 修订串（等长） | `**目的**: 架构设计 \| **仓库**: han \| **修订**: adr-0032-r03` | `**目的**: 架构设计 \| **仓库**: han \| **修订**: adr-0032-r04` |
| `:1220` | 候选 S4.23 节点 | `   - 候选 S4.23（anchor adapter bundle 四端口齐备并登记；尚未登记为正式 Task）完成，`Exclusive` 离开 `recoveryClosedPort`；` | `   - 候选 S4.23（anchor adapter bundle 四端口齐备并登记；已登记为正式 Task `identity-auth-S4.23`）：`Exclusive` 作为已登记但 closed 的端口接入 bundle，不得填实现、不得脱离 closed 端口状态；` |
| `:1222` | 候选 S4.24b 节点 | `   - 候选 S4.24b（部署独占 POSIX 锁后端与 `exclusive` 接线，受本 ADR 门禁）**requires 候选 S4.23**：bundle 必须先于锁后端接线，不得先接线后补来源；` | `   - 候选 S4.24b（部署独占 POSIX 锁后端与 `exclusive` 接线，受本 ADR 门禁）**requires 候选 S4.23**：候选 S4.23 交付且 bundle 四端口齐备（`Exclusive` 仍可为 closed）即可启动本节点；bundle 必须先于锁后端接线，不得先接线后补来源；`Exclusive` 离开 `recoveryClosedPort` 由本节点交付完成；` |
| `:1241` | 引用记录追加 1 行 | （无，原文件止于 `:1240`） | r04 修订记录：日期 2026-11-02、触发 B13J D07 与两份 Lead 授权记录、范围仅修订行与第 8 条两节点、声明不构成评审 PASS/人的接纳/生产批准 |

`git diff --no-index --unified=0`（改前快照 vs 改后文件）实跑只产生 4 个 hunk：`@@ -1167 +1167 @@`、`@@ -1220 +1220 @@`、`@@ -1222 +1222 @@`、`@@ -1240,0 +1241 @@`，无其它 hunk。

## 5. 增量账目（闭合）

| hunk | 改前行字节 | 改后行字节 | 增量 |
|---|---|---|---|
| `:1167` | 71 | 71 | +0 |
| `:1220` | 151 | 238 | +87 |
| `:1222` | 191 | 361 | +170 |
| `:1241`（追加） | 0 | 1387 | +1387 |
| 合计 | — | — | **+1644** |

`235729 + 1644 = 237373`，与改后实测字节一致，账目闭合。

## 6. 反向还原验证

方法：在改后文本上逐项反向执行第 4 节的三处替换，并删除末尾追加的第 1241 行，重新以 UTF-8 无 BOM、CRLF 连接、末尾单个 CRLF 写出 `%TEMP%\framework-reverse-r10-d07.md`，然后比对字节、行数与 SHA256，并复算三段指纹。

实跑结果：

- 还原后字节 235729、行数 1240、SHA256 `B4FA4BA8F017732325E0A58BEA152EDDDEBB7788422FB578E6C122FEC99A2402`。
- `match_r10 = True`（字节、行数、哈希三项同时命中）。
- 还原后 ADR-0032 段 L1164–1230 = `14938:E34FF7E61DD25BBE875F2572F1E040B551417941509588963EA2B35F13E250AA`（命中基线）。
- 还原后解除顺序区 L1218–1226 = `2047:E79D079A058CD3DEE7BEAACD5A808DC63E01CF7FD6A92AFB5E2035A2FB2C1223`（命中基线）。
- 还原后引用记录区 L1232–1240 = `3320:11155A848A3317B530F53CDEE96E5D7B72BF9205ECAF377F3FC1B1CAC49D4DA6`（命中基线）。

结论：改后文本相对 r03 只包含第 4 节列出的 4 处差异，写集外零差异。

## 7. 负向 oracle 与验收命令实跑

| 命令 | 期望 | 实测 | 结果 |
|---|---|---|---|
| `rg -n '^\s+- 候选 S4\.23.*离开 `recoveryClosedPort`' framework.md` | exit 1 | exit 1（0 命中） | 通过：S4.23 节点不再携带「`Exclusive` 离开 `recoveryClosedPort`」 |
| `rg -n '^\s+- 候选 S4\.24b' framework.md` | exit 0 | exit 0（`:1222`） | 通过 |
| `rg -n 'adr-0032-r03' framework.md` | 0 命中 | exit 1（0 命中） | 通过 |
| `rg -n 'adr-0032-r0[0-9]' framework.md` | 仅修订行 | 仅 `:1167` | 通过 |
| `rg -n '已解除.*首期不提供\|首期不提供.*已解除' framework.md` | 无命中 | exit 1 | 通过 |
| `rg -n '至多 1 个 revision\|或等价受控通道\|与 ADR-0032 的解除顺序(一致\|对齐)' framework.md` | 无命中 | exit 1 | 通过 |
| `git diff --check -- .add/project/knowledge/decisions/framework.md` | exit 0 | exit 0 | 通过 |
| `git status --porcelain -- cmd internal` | 改前改后逐字节相等 | 改前 `?? cmd/` + `?? internal/`（2 行）；改后完全相同（2 行） | 相等；但**不空**（见 §10-2） |
| `Test-Path .spec/wip/recovery-runtime/evidence-b13j-d07/report.md` | exit 0 | 本文件存在 | 通过 |

### 7.1 卡面验收命令 1 的 CRLF 陷阱（如实报告）

卡面 `tasks.md:3104` 写的是：

`rg -n '\*\*修订\*\*: adr-0032-r04$' .add/project/knowledge/decisions/framework.md  # exit 0`

实跑 **exit 1**。原因不是内容不符，而是 ripgrep 在未加 `--crlf` 时不剥离行尾 `\r`，`r04$` 无法匹配以 CRLF 结尾的行。同一文件的三项独立核证均通过：

- `rg --crlf -n '\*\*修订\*\*: adr-0032-r04$' framework.md` → exit 0，命中 `:1167`；
- `rg -n '\*\*修订\*\*: adr-0032-r04' framework.md`（去掉尾锚）→ exit 0，命中 `:1167`；
- `Select-String -Pattern '\*\*修订\*\*: adr-0032-r04$'` → 1 命中，`:1167`。

处置建议（超出本卡写集，需规划角色）：把该行改为 `rg --crlf` 或改用 `Select-String`。本卡不改 `tasks.md`。

## 8. 改后实测行号表（供后续引用）

| 内容 | 改后行号 |
|---|---|
| `<a id=adr-0032></a>` | `:1164` |
| 标题行 | `:1165` |
| 修订行（`adr-0032-r04`） | `:1167` |
| 状态行 | `:1169` |
| 解除条件第 8 条标题 | `:1218` |
| ADR-0030 决策修订节点 | `:1219` |
| 候选 S4.23 节点 | `:1220` |
| POSIX 锁后端与 ACL/负向证据节点 | `:1221` |
| 候选 S4.24b 节点 | `:1222` |
| 候选 S4.24a 节点 | `:1223` |
| 候选 S4.22 节点 | `:1224` |
| 解除条件 5–6 节点 | `:1225` |
| 无环说明段 | `:1226` |
| 不适用场景段 | `:1230` |
| `### 引用记录` | `:1232` |
| 既有记录 1–7 | `:1234`–`:1240` |
| 新增 r04 记录 | `:1241` |

**`tasks.md` 锚点结论**：本次三处改动均为同号替换，追加行只落在文件末尾，故 `:1164`–`:1240` 行号全部未位移。因此 `tasks.md` 卡面中「登记时实测 `framework.md:1218` 起 / `:1220` / `:1222` / `:1226`」在本次改动后**仍然成立**（卡面 `:3093`、`:3121`、`:3122`、`:3189`、`:3228` 的引用无需同步行号）。

## 9. 逐字不变项核对

- 不变量 1–9（`:1199`–`:1207`）与解除条件 1–7（`:1211`–`:1217`）：段指纹改前后完全相同（见 §3）。
- POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、`O_CLOEXEC` 常驻 fd、三入口共锁、锁路径与 ACL 合同、`journalPath != lockPath`：均位于上述逐字不变区间内，未改一字。
- `:1165` 标题行、`:1169` 状态行、`:1230` 不适用场景段、`:1234`–`:1240` 既有引用记录：未改（前缀/段指纹与反向还原共同证明）。

## 10. 偏差、未覆盖范围与残留（如实登记）

1. **`Exclusive` 措辞偏离 Lead 原始建议值**：Lead 建议 S4.23 节点写「不得使其离开 `recoveryClosedPort`」，但卡面负向 oracle 的正则是 `^\s+- 候选 S4\.23.*离开 `recoveryClosedPort``，若照抄会使该 oracle exit 0（与期望 exit 1 相反）。因此按卡面 `:3093` 的目标文本改写为「`Exclusive` 作为已登记但 closed 的端口接入 bundle，不得填实现、不得脱离 closed 端口状态」，语义与 Lead §4.2 完全一致（S4.23 不得使该端口离开 closed），只是换用「脱离 closed 端口状态」表述以避开 oracle 子串。`Exclusive` 离开 `recoveryClosedPort` 的时点已明确落在 S4.24b 节点（`:1222`）。
2. **`git status --porcelain -- cmd internal` 不为空**：本工作树中 `cmd/` 与 `internal/` 整体未被 Git 跟踪，改前改后恒为 `?? cmd/`、`?? internal/` 两行。该 oracle 的「为空」前提在本树中不成立；可核事实是「改前改后逐字节相同且我未写入这两处」，不以 git diff 作为越界结论。
3. **`:1226` 无环说明未改写**：Lead 指示「若因 2/3 改动而失真则最小改写，若仍成立则不改」。改动后链仍为 S4.23 → S4.24b（`requires`）→ S4.22，S4.24a 独立，无回边；且 S4.23 节点的前置只是变弱（不再要求 `Exclusive` 已交付），因此「有向无环、决策级修订先于实施/验证、当前不可满足的只是旧口径下的规划登记顺序」全部仍成立，故不动该行。卡面 `:3093` 的「同步改写」字样属目标摘要，实际规范约束（`:3111` 关闭条件 6 项）不要求改写；若复评坚持要字面改写，属零语义增量的一行编辑，可在下一轮执行。
4. **ADR-0030 侧残留（写集外，未改）**：`framework.md:979`、`:996`、`:1019`、`:1021`、`:1037` 仍以「ADR-0032 现行解除条件把 `Exclusive` 离开 `recoveryClosedPort` 列为候选 S4.23 完成的一部分」描述跨 ADR 冲突，经本次 r04 后该描述已过时，需由 ADR-0030 的下一次修订更正。`framework.md:1054`、`:1063` 仍含「尚未登记为正式 Task 的 S4.23 一类候选编号」「S4.23/S4.24/S4.22 仍为未登记候选」等历史表述。二者均超出本卡写集（仅 ADR-0032 段），本次未改，也不据此宣称已处置。
5. **未做**：未启动、未认领 S4.23/S4.24a/S4.24b/S4.22；未实现任何锁后端；未取得 ACL/POSIX/双进程互斥/崩溃释放证据；未改 `tasks.md`、Plan、`cmd/**`、`internal/**`、`design.md`、README、ADR-0019/0021/0029/0030/0031；未执行任何 Git 写操作。
6. **写集外关键制品实测（未被我修改的证明）**：`identity-auth/design.md` = 854991 B / `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D`；`decisions/README.md` = 21472 B / `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97`；`article-management/design.md` = 377424 B / `690015BB5C3B4800539F98C5B75115A5A20B5C161ECE7422F9F271840F7C55A7`（三者与 D05 r10 轮 Lead 记录值一致）。`tasks.md` = 865138 B / `54455ED278B7848BDD8530E06DF2E92B1DC7E75BE2F7B37A58B6D8ADCC3D8104`、Plan = 505880 B / `2FF40F396A81F7BA802984D84309FDC7289E8CE64C336BE744A2010DBA809B9F`（均为规划角色 D07 登记后的状态，本卡未写）。`git status --porcelain` 行数改前/改后同为 293，HEAD 未变。

## 11. 结论与不宣称边界

- 关闭条件自查：(1) 修订串已为 `adr-0032-r04`；(2) 第 8 条两节点口径已按卡面改写（S4.23 只作已登记但 closed 端口接入，`Exclusive` 离开 closed 归 S4.24b）；(3) POSIX-only / Windows 退出码 2 / 无 fallback / `O_CLOEXEC` / 三入口共锁 / 路径与 ACL 合同逐字未变；(4) 作者证据含哈希、字节、行数与反向还原命中（`match_r10 = True`）；(5) 写集外零差异（反向还原 + 前缀/ADR-0019 指纹证明）；(6) 未宣称评审 PASS 或人的接收。
- **不宣称**：ADR-0032 r04 未经过架构或安全独立评审，不是评审 PASS；不宣称 S4.23/S4.24a/S4.24b/S4.22 已启动、已解锁、已实施；不宣称 POSIX 锁后端已实现、ACL 已实测、生产 `enter`/`install`/`normal`/`recovery`/`requeue` 可用；不解除任何「首期不提供」与 `blocked-upstream`；不代替人的决策接纳、实施授权或 Git 写授权。

## 12. r05 证据段（`adr-0032-r05` 承接；补正 R06-F001 证据落点）

本段追加于既有 §1–§11 之后，不修改 r04 段任何文字、标题或既有自查行。目的：使本卡声明的证据落点与其真实承载一致——`adr-0032-r05` 由本卡（identity-auth-D07）承接完成，其完整过程证据写在 `.spec/wip/recovery-runtime/evidence-b13j-d05/report.md` §22（「r05/r11 收口」），本段给出可直接复核的摘要与指纹，不复制两份互相漂移的全文。

### 12.1 触发与授权

- 触发：架构侧第五轮复评 `B13J-D05-R05-F004`（MEDIUM，解除条件第 8 条的顺序自指），报告 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r05/report.md`。
- 授权：`.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r10.md` §C。

### 12.2 改动内容（`adr-0032-r04` → `adr-0032-r05`）

1. `framework.md:1171`（r05 完成时实测行号）修订串 `adr-0032-r04` → `adr-0032-r05`。
2. 删除原独立的第 8 条节点「本 ADR 解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据；」（−1 行），并把该证据并入候选 S4.24b 节点正文，使「产出者」与「证据」同节点：本节点同时交付并取证解除条件 1–3 的 POSIX 锁后端、隔离/ACL 证据与负向测试证据，该证据不再是独立前置节点、也不得早于本节点登记。候选 S4.24b 原有的 requires 候选 S4.23、bundle 先于接线、不得先接线后补来源、`Exclusive` 离开 `recoveryClosedPort` 由本节点交付等语义逐字保留。
3. 在 `### 引用记录` 末尾追加 1 条 r05 追溯行（r05 完成时为 `:1245`），声明不变量 1–9、解除条件 1–7 其余、POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、`O_CLOEXEC` 常驻 fd、三入口共锁、`journalPath != lockPath` 逐字未变，且不构成评审 PASS 或人的批准。

### 12.3 实测

| 项 | 值 |
|---|---|
| 改前 `framework.md` | 240301 B / 1244 行 / `B778BD3B79C4F6809E094B73A2B000C4D58EBC59C6EAA395DA956CC71555E839`（`adr-0032-r04`） |
| 改后 `framework.md` | 243427 B / 1245 行 / `F3FAE58F332D7AF6A3C35A5417564B1416219D3065A4FD9B66ADB266B930C6D8`（`adr-0032-r05`） |
| 节点行号（r05 完成、全文 1245 行时实测） | `:1171` 修订串、`:1222` 解除条件第 8 条标题、`:1224` 候选 S4.23、`:1225` 候选 S4.24b（含并入的证据句）、`:1226` 候选 S4.24a、`:1227` 候选 S4.22、`:1229` DAG 说明、`:1235` `### 引用记录`、`:1245` r05 追溯行 |
| 反向还原 | 按 hunk 机械逆替换重建 r04 文本，命中 240301 B / 1244 行 / `B778BD3B…E839`，`byte_equal_snapshot=True` |
| 不变段 | 不变量 1–9 = `3024:677949AB…9A66`、解除条件 1–7 = `1111:DABED6F6…D256`、ADR-0019 区间 95294 B = `607827B08F64D2ADD88DCB815F428575BA214EED3780D67C28DE7DEA6C18D2B6`，改前改后同值 |

后续 ADR-0030 r12 修订在 ADR-0030 引用记录末尾新增 1 行，使 ADR-0032 段整体下移 1 行（现为 `:1172`/`:1225`/`:1226`/`:1236`/`:1246`）；ADR-0032 段字节未变（`21165:F921F81BC589C51FE104DB62F6CB42065C6E417B70F54EEFF7222400B88B64A5`）。

### 12.4 证据落点声明

`adr-0032-r05` 的证据唯一落点为：`.spec/wip/recovery-runtime/evidence-b13j-d05/report.md` §22（全过程、逐 hunk 与反向还原）＋本报告 §12（摘要与指纹，供本卡按自身关闭条件自证）。两处数值同源；若后续出现不一致，以 d05 §22 的逐 hunk 记录为准并需同步本段。

### 12.5 不宣称

不宣称 `adr-0032-r05` 已构成评审 PASS 或人的接纳；不宣称 S4.23/S4.24a/S4.24b/S4.22 已启动或已解锁；不宣称 POSIX 锁后端已实现、ACL 已实测或生产可用；不解除任何「首期不提供」与 `blocked-upstream`。
