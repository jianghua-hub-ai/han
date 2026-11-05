# B13J-D01 设计增量独立评审报告（review-r01）

## 0. 角色、对象与写集

- 执行角色：`add-reviewer`（独立评审）。不是修订作者，未修改被评审产物。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮实测）。
- objectType：**design**。未套用提交/编译/覆盖率条款。
- 对象：`.add/project/knowledge/design-workspace/identity-auth/design.md` 本次增量修订（声明仅 `3989`、`4447`、`4449` 三行）。
- 基线：SHA256 `0FC3DE78C950661C2947680F678F92B7390843883FAD4F90362DB2C3F5A19277` / 842580 B。
- 目标：SHA256 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C` / 842966 B。
- 依据：`.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md`（实测 SHA256 `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F` / 26452 B）§2.3 `pending-B13J-D01`、§4 U1/U2/U9；作者证据 `design-revision-r01/report.md` 与 `index.json`（`state=applied-draft-unreviewed`，作者自述未获评审、未获设计 Gate）。
- 写集：仅本报告与同目录 `index.json`。未改 `design.md`、`tasks.md`、Plan、ADR、代码、测试；未做 Git 写；未删除/移动文件；未关闭任何既有 finding 或 Task。
- 边界：本评审不代替人的精确接纳、设计 Gate、Planner 登记或后续实施授权；结论只覆盖下文列出的实际检查范围。

## 1. 证据限制与范围

实际只读：

- `design.md`：frontmatter（`Get-Content -TotalCount 40`）、`78-91`（§决策追溯）、`3989`、`4430-4460`（§9.1）。
- `tasks.md`：`1872`、`2667`、`2669`、`2716-2731`、`2967`、`2975`、`3031`、`3034`。
- `gap-map.md`：`10-25`、`80-102`、`190-212`。
- `cmd/han-server/recovery_runtime.go:450-496`。
- 全仓只读检索 `0FC3DE78…` 与 `r57` 的消费者引用（`rg`）。

越界说明：为判定 SSOT 版本身份，额外只读同一对象的 frontmatter 与 §决策追溯（`78-91`）。这是对象元数据，不是新的外部依赖；未扩读其他章节或全库。允许范围中 `3570-3620`、`3850-3902`、`3975-3988`、`3990-3995` 未逐行读，其对本次 3 行 delta 的“未被改动”由第 2.3 节字节级反向重建排除；但其语义一致性未审（见 §8）。

未运行：Go、PostgreSQL、HTTP、restore、E2E、集成测试；未运行 markdown lint/渲染。`enter`/`install` 的生产可达性只引用关闭记录与源码，不是本轮实测。

## 2. Round 1：diff/格式/声明范围

### 2.1 哈希与字节

| 项 | 声明 | 本轮实测 | 结论 |
|---|---|---|---|
| 目标 SHA256 | `504966EA…5307C` | `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C` | 一致 |
| 目标字节 | 842966 | 842966 | 一致 |

### 2.2 格式不变量

逐字节扫描（PowerShell `ReadAllBytes`）：

- `bytes=842966`
- `bom=False`
- `LF_total=4924`
- `CRLF=169`
- `bareLF=4755`（= 4924 − 169）
- `bareCR=0`
- 末尾 6 字节 `66 20 2D 2D 3E 0A`，即 `f -->` 后单个 `0x0A`，无多余换行
- 以 UTF-8 解码成功（4925 段，末段为空 = 4924 行）

全部与任务书不变量一致。

### 2.3 精确 3 行 delta（反向重建）

做法：以 UTF-8 读目标，按 `\n` 切分；把三处“改后短语”唯一替换回“改前短语”，重新以 UTF-8 编码并计算哈希。

逐处替换对（改后短语均在该行仅出现 1 次；改前短语取自本报告独立复算，其中 3989 改前短语即 gap-map 验收命令点名的废弃句 最终 production 装配归 S4.16）：

**L3989**
- 改后：最终 production 装配所需端口的生产来源与构造尚无任务承接，须作为后续生产装配活动的责任经 Planner 对账后登记；已关闭的 S4.16 只交付其卡面范围内的同实例装配与 normal/resume/verify 开流顺序，不含这些生产端口。组件 C 在原 article 测试范围
- 改前：最终 production 装配归 S4.16，组件 C 在原 article 测试范围

**L4447**
- 改后：同实例装配与出口后最终开流顺序（限其卡面范围，不含 enter/install 生产端口与最终 production 装配）；root不写身份SQL
- 改前：同实例装配与出口后最终开流顺序；root不写身份SQL

**L4449**
- 改后：S4.16 原文未给上述精确新 port、其最终 production 装配也不属已闭范围，r52 的 S4.8 verifier亦需
- 改前：S4.16 原文未给上述精确新 port，r52 的 S4.8 verifier亦需

结果：`reconstructed_bytes=842580`，`reconstructed_sha256=0FC3DE78C950661C2947680F678F92B7390843883FAD4F90362DB2C3F5A19277`，与基线**精确相等**（`match=True`）。

结论：目标相对基线**只有** `3989`、`4447`、`4449` 三处字符改动，无任何未登记的字节（含换行）变化；+386 B 与三处替换长度一致。

### 2.4 废弃归属句已消失

```text
rg -n "production 装配归|装配归 S4\.16" .add/project/knowledge/design-workspace/identity-auth/design.md
```

真实结果：无输出，退出码 `1`（0 命中）。

`rg -n "最终 production 装配"` 仅命中 `3989`、`4447`、`4449` 三行，无第四处。

### 2.5 新表述不越权

- 未发明候选 Task 编号：`rg -n "task-identity-auth-s4-2[2-9]|identity-auth-S4\.2[2-9]"` 在 `design.md` 无命中，退出码 `1`。新句只写“后续生产装配活动……经 Planner 对账后登记”。
- 未宣称 `enter`/`install` 可达：三行内不含“可达/已交付/生产可用/已实现/已上线”等词（逐行检查为空）。
- 未改变 production 行为或已确认契约：改动是责任归属与措辞，未改 API/DDL/wire/错误语义；`tasks.md:3034` 与 `recovery_runtime.go:494-496` 仍为 `ErrAdmissionClosed`。

### 2.6 空白检查

`git diff --check -- .add/project/knowledge/design-workspace/identity-auth/design.md`：退出码 `0`，无 whitespace error；仅有仓库 `autocrlf` 提示（“LF will be replaced by CRLF the next time Git touches it”），非文件被改写。

未用 `git diff` 对照 HEAD 作为增量依据：HEAD 的 `design.md` 是旧版（作者证据记为 166893 B），842580 B 基线本身已是工作树既有未提交状态；增量以 §2.3 反向重建为准。

**Round 1 结论：PASS。** 声明范围、格式不变量、三行精确 delta 与“不再归属已闭 S4.16”均成立。

## 3. Round 2：消费者与一致性

### 3.1 与 S4.16 关闭记录一致

`tasks.md:2731` 明写：生产 CAS/anchor、installation commitment、source verifier、deployment lease、keyring loader、restore adapter 与真实 provider/ACL 未交付，recovery 仍显式 closed。新 `3989` 把这些“生产来源与构造”写成“尚无任务承接……经 Planner 对账后登记”，与关闭记录一致，未反向声称已交付。

### 3.2 与 S4.21 收窄记录一致

`tasks.md:2967` 只交显式 closed 端口清单、可注入装配 seam 与 secret candidate seam；`:3034` 明写 production `Enter`/`Install` 仍 `ErrAdmissionClosed`，并声明“不再声称 production `Enter`/`Install` 可达”。新措辞未与之矛盾。

### 3.3 与源码一致

`cmd/han-server/recovery_runtime.go:456-491` 的 `recoveryEnterInstallMissingPorts` 列出 11 个仍为 nil 的 seam（journal、articleSnapshots、restore、consumer、installation、articleAccount、commitments、sources、anchor、exclusive、secretCandidate）；`:494-496` 的 `recoveryEnterInstallClosedError` 包装 `recovery.ErrAdmissionClosed`。与“这些生产端口未交付”的表述一致。

### 3.4 与 gap-map D01 关闭条件一致

`gap-map.md:98` 要求：非作者设计 Review PASS；设计不再把未交付端口归给已闭 S4.16；明确后续 owner/活动类别；不写 tasks.md、Plan、TP；人的精确接纳仍标 unknown。

- 不再归属：§2.4 已证。
- 活动类别：写为“后续生产装配活动”，登记路径为“Planner 对账后登记”，未带编号，符合 `U1/U2` 建议值。
- 未写 tasks/Plan/TP：作者证据与 `git status` 一致（仅 `design.md` 为 `M`，证据目录为新增）。
- 人的接纳：frontmatter 仍 `draft`，未声称 Gate/接受。

### 3.5 作者自报残留张力：`4436`/`4438`

原文：

- `4436`：“……S4.16 继续承担原请求 lease、长期 control handle 和资源-before-pool 真 join。”
- `4438`：“……S4.16 仍 root/命令/单实例装配/实际维护 drain。”

**判定：属 S4.16 原卡面范围内的合法表述，与 `3989` 新表述不构成实质矛盾。**

依据：

- `tasks.md:2724` 的 `InstallRootDrainAndSingleBinding` 与 `:2725` 的 `LongLivedControlHandleTrueJoin` 明确把 root/命令/单实例装配/维护 drain、长期 control handle 与真 join 列为 S4.16 卡面交付。
- `3989` 排除的是“production 装配**所需端口的生产来源与构造**”，即 `2731` 列出的生产 CAS/anchor、commitment、verifier、lease、keyring loader、restore adapter 与真实 provider/ACL；不是 S4.16 对同一实例的运行时所有权与生命周期责任。
- 两者作用域不同：`4436`/`4438` 讲“持有并装配同一实例、root 生命周期与 drain”，`3989` 讲“为 enter/install 提供生产端口来源”。可同时为真。

残余只读风险（非阻断）：`4438` 的“单实例装配”与“最终 production 装配”字面接近，仍可能被快速读者误连。建议在**后续获授权轮次**加限定（例如“同实例装配，不含生产端口来源”）；本轮两行不在允许写集，未改，也不据此判 finding。

### 3.6 发现：SSOT 版本身份未随内容更新

见 §4 `B13J-D01-RV-01`（LOW）。

### 3.7 观察：`tasks.md:2669` 的原文宽于关闭记录

`2669` 仍写 S4.16“执行 normal 或 enter/resume/verify”，比 `2731`/`3034` 的“production Enter/Install 仍 closed”宽。该张力是**既有任务记录**，不由本次 design 修改引入；`4449` 已明确要求 Planner 在精确窗口同步 S4.8/10/16 措辞。故列为观察，不判 design finding，也不由本评审关闭或改写。

**Round 2 结论：PASS WITH MINOR。** 语义与消费者一致；唯一 finding 为 LOW 的版本身份/溯源问题。

## 4. Findings

### [LOW] 增量修订未更新 revision / last_updated / 决策追溯，SSOT 版本身份与下游哈希钉不一致

- finding ID：`B13J-D01-RV-01`
- 对象/修订：design；`design.md` 目标 `504966EA…5307C`（基线 `0FC3DE78…A19277` 之上 3 行增量）
- 位置：`.add/project/knowledge/design-workspace/identity-auth/design.md:5`（`revision: identity-auth-design-ir01-r57`）、`:7`（`last_updated: 2026-10-30`）、`:83`（§决策追溯仍只有 r57 行）
- 现状：内容已变为 `504966EA…` / 842966 B，但 frontmatter 仍标 r57，§决策追溯未追加本次增量行；全文检索 `B13J|r58|504966EA` 无命中（退出码 `1`）。文档自身无法说明“当前 r57”已被改写。
- 触发：任何以“design r57”或 `0FC3DE78…` 识别现役设计的消费者、评审或 Gate。已核到的钉住点包括 `tasks.md:2665`（“design r57 仍 draft”）、`.spec/wip/recovery-runtime/verification-b13g-r01.md:39`、`evidence-b13g-r01/s415-design-r57-r01/exact-edits.json`、`evidence-b13h-r01/s416-independent-review-r02/receipt.json`（`design_sha256=0FC3DE78…`）等 174 处引用（85 个文件）。
- 影响：同一 revision 标签对应两份不同字节；按哈希校验的工具会报不匹配，读者无法仅凭 SSOT 判断当前 r57 是否等于被 S4.15/S4.16 评审过的 r57；新旧内容边界只能靠 `.spec/wip/.../design-revision-r01/index.json` 解释。**不影响设计语义正确性、不改 production 行为，故为 LOW。**
- 建议：由 Lead 授权后，在最小后续修订中二选一：(a) 递增 `revision`（如 r58）+ 更新 `last_updated` + 在 §决策追溯追加一行（输入 r57 `0FC3DE78…`、输出 `504966EA…`、依据本评审）；或 (b) 明确记录“r57 内容已更新为 `504966EA…`、不递增编号”的接受决定，并对钉住 `0FC3DE78…` 的消费者做时点标注。评审方不代改，也不代选方案。
- 验证：frontmatter `revision`、`last_updated` 与 §决策追溯行一致且对应实测 SHA；`rg "0FC3DE78|504966EA"` 的消费者要么明确指向历史时点、要么显式标注新内容；`git diff --check` 仍退出 `0`，§2.2 格式不变量不变。

合计：1 条 LOW，0 条 MEDIUM/HIGH/CRITICAL，0 条阻断语义目标。

## 5. 待决定

| 编号 | 事项 | 缺什么/影响 | 决定负责人 |
|---|---|---|---|
| D-RV-01 | 是否接受“r57 标签不变、内容已更新”的现状，或授权 r58 递增+追溯行 | 见 `B13J-D01-RV-01`；不决定则 SSOT 版本身份持续歧义 | Lead（并发起人/用户确认） |
| D-RV-02 | `gap-map` U2“不重开 S4.16、另立后续范围”是否已由用户确认 | 本轮三处措辞按建议值书写；若改为重开，需回改 | 用户/Lead |
| D-RV-03 | 后续生产装配活动的登记批次/卡片与时机 | 设计只给活动类别与 Planner 对账路径，未给编号；何时登记属 Planner/Lead | Planner/Lead |
| D-RV-04 | 是否在后续授权轮次收敛 `4436`/`4438` 的“单实例装配”措辞 | 当前无实质矛盾，仅可读性风险 | Lead |

## 6. 建议（非阻断）

- 若 Lead 希望零歧义，后续授权轮次可把 `4438` 的“单实例装配”补成“同实例装配（不含生产端口来源）”；不属本轮缺陷。
- `4449` 已把 S4.8/10/16 的措辞同步交给 Planner；建议 Planner 同步时一并处理 `tasks.md:2669` 的宽表述（见 §3.7）。
- 不要在未获授权时顺手改 frontmatter；`B13J-D01-RV-01` 的修复应有明确写窗与复审。

## 7. 实际核验命令与结果

| # | 命令 | 真实结果 |
|---|---|---|
| 1 | `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 2 | `git status --porcelain -- <design.md> <design-revision-r01>` | ` M design.md`；`?? .../design-revision-r01/` |
| 3 | `Get-FileHash -Algorithm SHA256 design.md` + `.Length` | `504966EA…5307C`；842966 |
| 4 | 逐字节扫描 BOM/CRLF/bareLF/bareCR/尾部 | bom=False；CRLF=169；bareLF=4755；LF=4924；bareCR=0；tail=`66 20 2D 2D 3E 0A` |
| 5 | Python 反向替换重建三行后重算 | 842580 B；`0FC3DE78…A19277`；`match=True`；三处 after 各唯一出现 1 次 |
| 6 | `rg -n "production 装配归|装配归 S4\.16" design.md` | 无输出，exit `1` |
| 7 | `rg -n "最终 production 装配" design.md` | 仅 `3989`、`4447`、`4449` |
| 8 | `git diff --check -- design.md` | exit `0`；仅 autocrlf 提示 |
| 9 | `rg -n "task-identity-auth-s4-2[2-9]|identity-auth-S4\.2[2-9]" design.md` | 无输出，exit `1` |
| 10 | 三行逐行查“可达/已交付/生产可用/已实现/已上线” | 均为空 |
| 11 | `rg -n "B13J|r58|504966EA" design.md` | 无输出，exit `1` |
| 12 | `rg -n "0FC3DE78…" .add .spec` | 174 处 / 85 个文件（含 `tasks.md:2665`、`verification-b13g-r01.md:39`、`s415-design-r57-r01/exact-edits.json`、`s416-independent-review-r02/receipt.json`） |
| 13 | `Get-FileHash gap-map.md` + `.Length` | `3AA5B959…D10F`；26452 |
| 14 | 只读 `tasks.md` 指定行与 `recovery_runtime.go:450-496` | 与 §3.1–3.3 引文一致 |

以上哈希、行文本与退出码均为本会话独立重算/重读，不是转抄作者报告。

## 8. 未验证与剩余风险

- 未运行任何 Go/PG/HTTP/restore/E2E；`enter`/`install` 仍 `ErrAdmissionClosed` 的结论来自 `tasks.md` 关闭记录与源码，不是本轮运行证据。
- 未做全文 4924 行一致性审计。`3570-3620`、`3850-3902`、`3975-3988`、`3990-3995` 未逐行读；这些区域未被本次 delta 改动（字节级已证），但其中是否仍有“把未交付端口归给 S4.16”的措辞未审，不能声称全篇已对齐。
- 未审 `tasks.md`、Plan、ADR、test-plan 的完整消费者；只核了任务书列出的行与 `rg` 命中面。
- 未运行 markdown lint/渲染；`git diff --check` 仅覆盖空白类问题。
- 人（用户）的精确接纳 unknown；frontmatter 仍 `draft`，无设计 Gate。
- `B13J-D01-RV-01` 未决期间，SSOT 版本身份保持歧义；钉住 `0FC3DE78…` 的消费者未更新（不在本写集）。
- 本评审不关闭任何既有 finding/Task，也不把 Round 1/2 的 PASS 外推为 D01 的正式关闭或后续实施授权。

## 9. 结论

- **Round 1：PASS** —— 目标哈希/字节、格式不变量、精确三行 delta、废弃归属句消失、无新 Task 编号、未宣称 enter/install 可达，均成立。
- **Round 2：PASS WITH MINOR** —— 与 `tasks.md:2731`/`:3034`、源码 11 端口 closed、gap-map D01 条件一致；`4436`/`4438` 判定为卡面内合法表述、与 `3989` 无实质矛盾。
- **总体：PASS WITH MINOR** —— 1 条 LOW（`B13J-D01-RV-01`），0 条 MEDIUM/HIGH/CRITICAL，无语义阻断。
- 是否据此关闭 `pending-B13J-D01`、是否递增 revision、是否登记后续生产装配活动，均由 Lead/用户决定；本报告不代替设计 Gate 或人的接纳。
