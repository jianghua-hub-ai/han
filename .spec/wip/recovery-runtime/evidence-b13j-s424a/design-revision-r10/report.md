# 设计修订报告 — identity-auth 详细设计 r70（design-revision-r10）

- 日期：2026-11-02
- 角色：add-feature-designer（Feature-Designer；主方法 add-design-tdd）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 修订对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 版本：`identity-auth-design-ir01-r69` → `identity-auth-design-ir01-r70`（`:5`／`:38`）
- 判据报告（本轮输入，只读；哈希由本会话自算）：`review-arch-r09/report.md` 20,732 B / 174 行 / `764406E81FEEA90A26B1C34C2DC81F3B5298C07C06F8CBE9A39366358B426C6D`；`review-sec-r09/report.md` 13,848 B / 130 行 / `8D74FA3BC39AD6C8A0C99E7AB7F56614FACCFA8D19F6A40D5FFE11989CEB4A51`
- 冻结快照：`C:\Users\J00010385\AppData\Local\Temp\design-r69-freeze-41D708D9.md`
- 冻结核对：改前现场 `design.md` = 1,099,874 B / 5,476 行 / CR=169 / LF=5,476 / SHA256 `41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09`，与冻结快照逐字节相等（byteEqual=True），**无漂移**，未发生「漂移后改写」。

## 1 写盘方式与字节账

写盘方式：`[IO.File]::ReadAllText($p,[Text.Encoding]::UTF8)` → 对 10 处精确子串 `Replace`（每处在替换前断言 `count==1`，任一不唯一即中止且不写盘）→ 单次 `[IO.File]::WriteAllText($p,$txt,(New-Object Text.UTF8Encoding($false)))`。未使用 `Set-Content`／`Out-File`；无 BOM（首 3 字节 45,45,45 = `---`）；末字节 10（LF）；混合换行保持（CR=169、LF=5,476）。**一次写盘，无中间态**；`design.r69.baseline.md` 为写盘前的逐字节快照。

| 项 | 改前（r69） | 改后（r70） | Δ |
|---|---|---|---|
| 字节 | 1,099,874 | 1,103,152 | +3,278 |
| 行数 | 5,476 | 5,476 | 0 |
| CR | 169 | 169 | 0 |
| LF | 5,476 | 5,476 | 0 |
| SHA256 | `41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09` | `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254` | — |

逐行字节账（UTF-8 行字节，不含行尾；逐行 Ordinal 比较）：

| 行 | 改前 B | 改后 B | Δ | 内容 |
|---|---:|---:|---:|---|
| `:5` | 39 | 39 | 0 | `revision: …-r70` |
| `:38` | 50 | 50 | 0 | 设计版本 r70 |
| `:4072` | 6,806 | 8,022 | +1,216 | 优先级链扩 r70、r68 后缀引用改口、追加 r70 修订日志 |
| `:4076` | 19,973 | 21,950 | +1,977 | `waitFlow` 单点合同、joinWork 第③步改写、reaper 单点化与判据加强 |
| `:4083` | 511 | 596 | +85 | 追加 r70 沿用声明 |
| **合计** | | | **+3,278** | |

配平：1,103,152 − 1,099,874 = +3,278 = 逐行 Δ 之和；行数增量 0（0 行增删）；除上述 5 行外其余 5,471 行逐字节相同。

## 2 逐 finding

### 2.1 `NEW-R09-01`（MEDIUM，架构）— **CLOSED**

- 落点行号：`:4076`（`completeOnRequest` 段与 `joinWork` 第③步）。
- 修改摘要：
  - 在「两者成功后 `setFlowResult(a.waitFlow(ctx))` 再 `requestStop()`。」之后单点写出 **`waitFlow(ctx)` 单点合同（唯一实现，两个调用点共用）**：函数体为阻塞 `select`；`<-a.flowDone` 分支返回已记录的 `a.flowErr`（`flowErr` 由 `setFlowResult` 在 `close(flowDone)` 之前写入，关闭即 happens-before 可见）；`<-ctx.Done()` 分支到达时**先重查一次** `flowDone`，已关闭返回 `flowErr`，仍未关闭才返回可诊断的 ctx 错误（`context.DeadlineExceeded`／`context.Canceled`，允许包装但不改变分类）。
  - 明确禁止「未关闭即返回」：`flowDone` 未关闭前**不得**返回任何结果值（包括 nil），不得用 `select default` 返回哨兵，不得用 nil 表示「尚无结果」；唯一不以 `flowErr` 收尾的出口是 ctx 先到期且重查后仍未关闭。
  - 补充不变量：「两条业务路径的**每个**出口都先 `setFlowResult` 再返回，故 `flowBothDone` 关闭蕴含 `flowDone` 已关闭」；`waitFlow` 不承担 join。
  - 两种顺序的可机核判据：(a) `Enter` 先失败、`completeOnRequest` 后到——`flowDone` 已由该失败关闭，`waitFlow` 必须返回该失败（不得 nil），随后该调用点执行 `requestStop()`；(b) `completeOnRequest` 先到而 `Enter` 尚未落结果——`waitFlow` 阻塞于 `flowDone`／`ctx.Done()`，不得以 nil 关闭 `flowDone`，必须等到 `Enter` 成功或失败结果，或 ctx 到期后返回 ctx 错误，绝不以 nil 收敛。
  - `joinWork` 第③步改写为「只读已记录结果」：屏障通过后 `flowDone` 必然已关闭，按上述合同返回已记录的 `flowErr`；`flowDone` 已关闭时 `flowDone` 分支与「`ctx.Done()` 后重查」分支返回同一 `flowErr`，不存在被 `ctx.Err()` 随机替换的路径，也不再承担 join。
- 闭合判定：**CLOSED**。同一 helper 在两个调用点语义唯一；「非阻塞窥视返回零值 → nil 赢得 `sync.Once` → 吞掉 `Enter` 失败」的字面路径已被正文禁止，并给出 (a)(b) 两种顺序的判据。

### 2.2 `NEW-S09-01`（LOW，安全）— **CLOSED**

- 落点行号：`:4076`（`start` 分支 reaper 句、机核判据①）。
- 修改摘要：改为「两条**业务** goroutine 在函数体首行各自 `defer a.flowWG.Done()`、都**不**起 reaper，reaper 只由 `start` 在两条业务 goroutine 之前起 **1 个**」，删除「并各起一次性 reaper」；明写「本设计不为 `flowBothDone` 设 `sync.Once` 兜底，因为不存在第二个关闭点」；机核判据①加强为「两条业务 goroutine 均不起 reaper、`go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()` 在 `start` 内恰 1 次、`close(a.flowBothDone)` 全文恰 1 次且只在 `flowWG.Wait()` 之后」。
- 闭合判定：**CLOSED**。`:4076` 内 `reaper` 4 处（`start` 1 + 新句 3），无「各起」表述；`:4072` 的 r70.2 日志仅以「删除「各起一次性 reaper」」形式引用被删措辞，属变更记录、不是规范指令；`close(a.flowBothDone)` 单一关闭点，「恰 1 次」判据与正文一致，二次 close panic 的字面歧义消除。

### 2.3 `NEW-R09-02`（LOW，架构）— **CLOSED**

- 落点行号：`:4072`。
- 修改摘要：将「r68／r69 时点值只登记在各自修订的 `hashes.txt`（含取得时点）」改为如实表述：r68 时点值**未登记**（`design-revision-r08/hashes.txt` 无后缀节，属登记缺口，本句如实标注）；r68 与 r69 的三段后缀逐字节相同，由 r69 `hashes.txt` 节 [5] 的「对照 r68 基线……三段字节数与 SHA256 相同」佐证；r69 时点值（取得时点 2026-11-02 17:07:41 +08:00）登记在 `design-revision-r09/hashes.txt` 节 [5]，是该时点的唯一登记；r70 时点值登记在 `design-revision-r10/hashes.txt`（含取得时点）。**未回填、未修改 r68 `hashes.txt`**（仍为 3,459 B / 65 行 / `4770136A02F3C706AB60170E18CB0E26CE7252A2E39175CFD0F85380A9CE9757`）。
- 闭合判定：**CLOSED**。设计断言与磁盘事实一致；本段自定的引用规则（源文件 SHA-256 + 行号 + 取得时点）不再自相矛盾。

### 2.4 版本、优先级与声明（配套变更，同行内）

- `:5`／`:38`：`identity-auth-design-ir01-r69` → `identity-auth-design-ir01-r70`（字节数不变）。
- `:4072`：`r66.0` 块内唯一优先级规则链 `r68 > r67 > r66 > r65 > r64 > r63 > r62 > r61` 扩为 `r70 > r69 > r68 > r67 > r66 > r65 > r64 > r63 > r62 > r61`；并在行内追加「**r70 就地修订（无新增／删除行）**」段，登记 r70.1（`waitFlow` 单点合同，`:4076`）、r70.2（仅 `start` 起 1 个 reaper，`:4076`）、r70.3（r68 后缀值改判登记缺口，`:4072` 本段）。
- `:4083`：追加「 r70 修订沿用本声明，不改变任何 NOT RUN／unsupported／closed 状态。」（+85 B）。

## 3 无回退核对

- 逐行 Ordinal 比对 `design.r69.baseline.md`：仅 `:5`、`:38`、`:4072`、`:4076`、`:4083` 五行不同，其余 5,471 行逐字节相同；已闭合条款在文本层不可能回退。
- `:4076` 内 F006 锚点保持：`activeExclusiveLocked()` 8、`:892` 2、`:1943` 2、`:1968` 1、`ownershipLease` 4；全文 `recoveryPreflight` 2、`restoreDriverFactory` 2、`recoveryClosedRestoreDriver` 17、`s414Canary` 2。
- 保留项逐条保持：teardown 顺序句「`Stop` → 等 `flowBothDone` → `waitFlow` → `closeRestoreLease`」（2 处）；执行器不响应 ctx 时持租约 fail-closed（`执行器不响应 ctx` 2 处）；`injectedExclusive` 永不 Close（`永不关闭` 4 处）；strict v1、无 shell、私有 stdin、最小 ENV、Begin 前 closed、不得以 stdout／exit=0／DB 可连接自证均在未变更行。
- 逐项未触碰清单：`F006`(a)(b)、`NEW-R07-01/02/03`、`NEW-R05-03`、`NEW-R07-04` 文本层、`F004` 文本层、`NEW-R05-04/05`、`NEW-R06-02/05/06/07/08`、`SEC-F002/F003`、`N001`、`SEC-F006`、`N011`、`NEW-S05-01/02/04/05/06/07`、`N012`–`N016`、`NEW-S06-02/03`、`NEW-S07-01`、`NEW-R08-01/02/03`、`NEW-S08-02`。
- 零写入核对（只读取值，取报告生成时点）：`tasks.md` 898,536 B / 3,289 行 / `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`，mtime 2026-11-02T09:21:19（早于本轮），`:3186` 仍为 `- [ ]`；`decisions/framework.md` 285,639 B / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`，mtime 2026-11-02T15:23:48（早于本轮）。两者未改、未勾选。

## 4 golden vector 复算（改后终值）

方法：`[IO.File]::ReadAllLines(design.md, UTF8)[n-1]` 取单行原文 → `[Text.Encoding]::UTF8.GetBytes(line)`（不含围栏与行尾）→ SHA-256。0 行增删，行号不变。

| vector | 行 | 字节 | SHA256 | 期望 | 判定 |
|---|---|---:|---|---|---|
| source | `:4099` | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 / `de914622…` | PASS |
| tool | `:4105` | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 / `8c0c89…` | PASS |

## 5 三段后缀复算

方法：raw bytes；按 `0x0A` 扫描建立行首偏移表（5,477 个行首，末行首 = EOF）；取该行行首字节偏移 → EOF；不重新编码、不做换行归一；SHA-256 对区间原始字节计算。取得时点：**2026-11-02 17:19:54 +08:00**；对象 SHA256 = `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`（r70 终值）。

| 行 | 改前 start（r69） | 改后 start（r70） | 字节 | SHA256 |
|---|---:|---:|---:|---|
| `:4259` | 816,394 | 819,672 | 283,480 | `e8848838df83bc09bd8c94297219702cd86bd6629bd31fd8853f834f652cf0b6` |
| `:4275` | 818,674 | 821,952 | 281,200 | `2f53882f8774d30a51754872449ba29b4f257253c8a2f457ab6b9109d054fafb` |
| `:4290` | 821,659 | 824,937 | 278,215 | `b0af3b04c60584cf2618c83605e7fba88f65ca6b48f486463570268279e05f04` |

- 起点差 3,278 = 全文增量（819,672 − 816,394），字节数与 SHA256 与 r69／r68／r67 完全相同，证明全部改写都在 `:4259` 之前。
- 与派发给定基线逐值一致（816,394／818,674／821,659；283,480／281,200／278,215；三条 SHA256）。
- `review-sec-r09` 报的 283,444 B 值经本会话独立复算为算术错误：`1,103,152 − 819,672 = 283,480`；该错值未写入设计，也未据此改 r69 `hashes.txt`。

## 6 括号平衡

全文全角括号：改前 1,205 / 1,205 → 改后 **1,216 / 1,216**（平衡）；新增与改写片段内部括号均成对。

## 7 未闭合／残余、升级待决与 NOT RUN

- 本轮 3 项：`NEW-R09-01` CLOSED、`NEW-S09-01` CLOSED、`NEW-R09-02` CLOSED；无未闭合项。
- 范围外残余（r09 安全复审的非阻断项，本轮未处理、不声称闭合）：provider 错误串不得回显候选字节的合同未单列；`secretErr != nil` 分支未登记 `defer clear(candidate)`（生产默认源返回 nil，当前无实害）。
- 升级待决（只登记，不做）：`F004` 的 ADR 同步；`F017` 任务面对账；范围外 `N005`（`framework.md` 状态段与 `tasks.md:3197` 口径冲突）。
- NOT RUN：`go build`／`go vet`／`go test`／`-race`；真实 restore、真实备份／旧库、E2E；`enter`／`resume` 实际进入与解锁（生产候选源未交付，按字面进入即 `ErrAdmissionClosed`）；`SEC-F007` 与 Linux 侧能力（host／ACL／mount／loader／fd-exec／maps／network／witness）；ADR-0031 全文复审；`tasks.md` 卡面对账。全部未运行、不因本报告改判。

## 8 声明

本报告只完成设计文本修订，**不构成**独立评审通过，**不构成**生产 restore 解锁，**不构成** `enter`／`resume` 解锁，**不构成**约束 8 满足，**不构成** `identity-auth-S4.24a` 可关闭或已交付；未勾选 `tasks.md` 任何复选框。最终接受由两路独立复审与 Lead／用户决定。

## 9 写入范围

本会话写入：`design.md`（一次写盘）、`design-revision-r10/design.r69.baseline.md`、`design-revision-r10/report.md`、`design-revision-r10/hashes.txt`。此外零写入：未改 `tasks.md`、`decisions/framework.md`（ADR-0031）、`cmd/**`、`internal/**`、测试、任何既有 `review-*/` 与 `design-revision-r0[1-8]/`；未删除、移动、归档；未执行任何 Git 写操作。