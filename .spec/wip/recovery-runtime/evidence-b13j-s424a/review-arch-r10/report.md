# 架构独立复审报告 — identity-auth 详细设计 r70（review-arch-r10）

- 日期：2026-11-02
- 角色：add-reviewer（架构轴）；主方法 `.codex/skills/add-code-review/SKILL.md`（objectType=design）；规则 `code-review-checklist`、`abstraction-principles`、`design-doc-writing-style`、`decisions-guide`
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
  - **1,103,152 B / 5,476 行 / CR=169 / LF=5,476 / SHA256 `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`**
  - 版本 `identity-auth-design-ir01-r70`（`:5`、`:38` 一致）
- 冻结快照：`C:\Users\J00010385\AppData\Local\Temp\design-r70-freeze-3A8A9E63.md`，1,103,152 B / 同 SHA256（与现场逐字节一致）
- 漂移核对：**无漂移**。现场 `design.md` 哈希等于冻结哈希；工作树中其他未提交改动（decisions/、其他特性设计、`tasks.md`、`go.mod` 等）属并行会话，不是本对象，未作为判读依据。
- 基准报告：`review-arch-r09/report.md` 20,732 B / 174 行 / `764406E81FEEA90A26B1C34C2DC81F3B5298C07C06F8CBE9A39366358B426C6D`（与派发一致）
- 作者线索（只作线索，数值全部自算）：`design-revision-r10/report.md` 13,214 B / 116 行 / `724130C867DA7A134DF964C033E731F118F75CCD100518399F2AB81BD52D98EE`；`hashes.txt` 4,729 B / 58 行 / `6635624ADDBEA8458206E6500128EE38198FEAA700C6BDA40258E3E96D16BB36`；`design.r69.baseline.md` 1,099,874 B / `41D708D9…`
- 方法：两轮。Round 1 对 5 行变更逐句核对，并对 `:4072`／`:4076` 做从句级增删比对（`。`／`；` 切分集合差）；Round 2 沿真实消费者（`cmd/han-server/recovery_runtime.go` 的 `completeOnRequest`／`setFlowResult`／`waitFlow`／`awaitControlReady`／assembly 字段与 `:617` 初始化）核 happens-before、call-site 与 teardown；另独立复算 golden vector、三段 raw-byte 后缀、括号与字节账，并核 `tasks.md`／ADR 未擅改。
- 检查范围：冻结对象全文（逐行 `-cne` 与 r69 基线比对）、`:5`、`:38`、`:4072` 全文、`:4076` 全文（15,028 字符）、`:4083`、`:4099`、`:4105`、`:4259`／`:4275`／`:4290`；r69 基线快照、r68／r67 基线（后缀对照）；作者 r10 `report.md`／`hashes.txt`；r09 `hashes.txt` 节 [5]；r08 `hashes.txt`（缺口核对）；r09 报告全文。
- 只读声明：本会话仅写本目录 `report.md` 与 `hashes.txt`。未修改 `design.md`、`tasks.md`、ADR（`framework.md`）、`cmd/**`、`internal/**`、测试、任何既有 `review-*/` 与 `design-revision-*`；未删除、移动、归档；未执行任何 Git 写操作。
- 判读过的文件：`design.md`（r70）、`%TEMP%\design-r70-freeze-3A8A9E63.md`、`design-revision-r10\design.r69.baseline.md`、`design-revision-r10\report.md`、`design-revision-r10\hashes.txt`、`design-revision-r09\design.r68.baseline.md`、`design-revision-r09\hashes.txt`、`design-revision-r08\design.r67.baseline.md`、`design-revision-r08\hashes.txt`、`review-arch-r09\report.md`、`tasks.md`、`decisions\framework.md`、`cmd\han-server\recovery_runtime.go`、`.codex\agents\add-reviewer.toml`、`.codex\skills\add-code-review\SKILL.md`、`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\guidance\design-doc-writing-style.md`、`.codex\guidance\decisions-guide.md`。

Verdict: PASS

- 开放项与阻断计数：**BLOCKER 0 / MEDIUM 0 / LOW 0**。r09 结转 2 项全部闭合：`NEW-R09-01`（MEDIUM，CLOSED）、`NEW-R09-02`（LOW，CLOSED）；安全侧 `NEW-S09-01` 跨轴复核 CLOSED。范围外登记 `N005` 1 项，不计入阻断、不据此改变判定。

## 1. 开放项清单

| ID | 严重度 | 主题 | 位置 | 本轮判定 |
|---|---|---|---|---|
| `N005` | 范围外 | ADR `framework.md` 状态段与 `tasks.md` 的口径冲突 | `decisions/framework.md` / `tasks.md` | 只登记，不参与本判定，不代改 |

已闭合：`NEW-R09-01`（CLOSED）、`NEW-R09-02`（CLOSED）、`NEW-S09-01`（CLOSED，安全侧跨轴复核）。本轮无新 finding。

## 2. 逐项判定（对照 r09 开放项）

### 2.1 `NEW-R09-01`（MEDIUM，`waitFlow` 无单点合同）— **CLOSED**

- 设计侧（`:4076`）：新增「`waitFlow(ctx)` 单点合同（NEW-R09-01，唯一实现，两个调用点共用）」，函数体固定为阻塞 `select`：
  - `case <-a.flowDone:` 返回已记录的 `a.flowErr`；`flowErr` 由 `setFlowResult` 在 `close(flowDone)` 之前写入，关闭即 happens-before 可见。
  - `case <-ctx.Done():` 到达时**先重查一次** `flowDone`，已关闭返回 `flowErr`，仍未关闭才返回可诊断的 ctx 错误（`DeadlineExceeded`／`Canceled`，允许包装不改分类）。
  - 明禁「`flowDone` 未关闭前返回任何结果值（含 nil）」「`select default` 返回哨兵」「nil 表示尚无结果」；唯一不以 `flowErr` 收尾的出口是 ctx 先到期且重查后仍未关闭。
  - 两条业务路径每个出口都先 `setFlowResult` 再返回，故 `flowBothDone` 关闭蕴含 `flowDone` 已关闭；`waitFlow` 不承担 join。
- 两处调用已统一：`completeOnRequest` 的 `setFlowResult(a.waitFlow(ctx))` 与 `joinWork` 第③步「只读已记录结果」共用同一合同；屏障通过后 `flowDone` 必已关闭，两分支返回同一 `flowErr`，不再有「碰运气非阻塞」路径。r69 的「先非阻塞接收」与「只读结果」措辞在 r70 全文命中 **0**。
- 两种顺序判据可机核：(a) `Enter` 先失败、`completeOnRequest` 后到 → 必须返回该失败、不得返回 nil，随后 `requestStop()`；(b) `completeOnRequest` 先到而 `Enter` 未落结果 → 阻塞于 `flowDone`／`ctx.Done()`，不得以 nil 关闭 `flowDone`，只能等 `Enter` 结果或 ctx 到期返回错误。原「非阻塞窥视返回零值 → nil 赢得 `sync.Once` → 吞掉 `Enter` 失败」的字面路径已被正文禁止。
- 消费者核对（自行读代码）：`recovery_runtime.go:916`–`:921` 的 `setFlowResult` 确为「`a.flowErr = err` 后 `close(a.flowDone)`」，与 happens-before 表述一致；`:517`–`:519` 存在 `flowOnce`／`flowDone`／`flowErr` 字段；`:617` 确有 `flowDone: make(chan struct{})` 初始化；`:901`–`:914` 的 `completeOnRequest` 与设计锚点吻合；现役 `:932`–`:938` 的 `waitFlow` 尚无 ctx 分支重查，属目标态实现差量（设计已把它固定为唯一实现），不是设计缺口。
- 残余观察（非阻断、不计 finding）：`:4076` 的「两个调用点共用」按函数计为 `completeOnRequest` 与 `joinWork`；`case "install"` 的 `waitFlow` 位于 `joinWork` 内，合同同样覆盖，不构成第三种语义。

### 2.2 `NEW-R09-02`（LOW，`:4072` 误称 r68 后缀值登记）— **CLOSED**

- 设计侧：`:4072` 已改为「r68 时点值**未登记**（`design-revision-r08/hashes.txt` 无后缀节，属登记缺口，本句如实标注）」；r68 与 r69 后缀相同改由 r69 `hashes.txt` 节 [5] 的「对照 r68 基线……三段字节数与 SHA256 相同」佐证；r69 值标明取得时点 `2026-11-02 17:07:41 +08:00` 与出处；r70 值声明登记在 `design-revision-r10/hashes.txt`（含取得时点）。
- 我独立核对：r08 `hashes.txt`（3,459 B / 65 行 / `4770136A…9757`）对 `4259`／`4275`／`4290`／`816394`／`818674`／`821659` 命中 **0**，缺口标注属实且未回填；r09 `hashes.txt` 节 [5] 确含 816,394／818,674／821,659 三值与 17:07:41 时点；r10 `hashes.txt` 节 [5] 确含 r70 三值与 `2026-11-02 17:19:54 +08:00`。引用链自洽。
- 交叉引用复算：我对 `design.r68.baseline.md`（1,093,211 B / `5E324AF9…`）按 raw bytes 复算三段后缀，起点 809,731／812,011／814,996，字节数 283,480／281,200／278,215，SHA 与 r70 完全相同，佐证句成立。

### 2.3 `NEW-S09-01`（安全侧，跨轴复核一句）— **CLOSED**

- `:4076` 已固定：两条**业务** goroutine 首行各自 `defer a.flowWG.Done()`、均**不**起 reaper；reaper 只由 `start` 在两条业务 goroutine 之前起 **1 个**（`go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()`）；启动顺序为 `Add(2)` → reaper → 两条业务 goroutine；不为 `flowBothDone` 设 `sync.Once` 兜底，因为不存在第二个关闭点；机核判据要求 `close(a.flowBothDone)` 在代码中恰 1 次且只在 `flowWG.Wait()` 之后。「各起」全文仅剩 `:4072` 变更日志中的删除记录，不是规范指令。二次 close panic 与多 reaper 歧义已消除。

### 2.4 无回退确认

- 逐行 `-cne` 比对 r69 基线：仅 `:5`、`:38`、`:4072`、`:4076`、`:4083` 五行不同，其余 **5,471 行逐字节相同**。因此 `:3603`（F004 文本层）、`:4070`（B1／长名／pin）、`:4077`、`:4079`、`:4195`、`:3942`、`:3990` 等已闭合条款在文本层不可能回退。
- `:4076` 从句级集合差：删除恰 3 条，全部是本轮应替换的旧义务——「各起一次性 reaper」句、`joinWork` ③「先非阻塞接收」句、旧机核判据①；新增 13 条为新合同与判据。无其他义务被删除。
- `:4072` 从句级集合差：删除 2 条（旧优先级链 `r68 > …`、旧「r68／r69 时点值只登记在各自 hashes.txt」句），新增 9 条（r70 优先级链、缺口如实标注、r69／r70 出处、r70 修订日志）。历史值 `282,599`／`7D1DDC08`、`280,319`／`5BD8E870`、`277,334`／`E40EB7A2`、`+881`、「只作历史」「不得再标作 r61 段基线」、`282,706`／`a1a256f5`、`1,045,111`／`87D70DA8` 全部仍在。
- 关键义务仍在（`:4076`）：`activeExclusiveLocked()` 8 次、禁 `activeExclusive()` 句、`:892`／`:1943`／`:1968`、`ownershipLease` 4、`recoveryPreflight`、`restoreDriverFactory` 2、`newControlledRestoreDriver` 2、`recoveryClosedRestoreDriver` 2、`s414Canary`、`admission.go:648`、`beginObservation` 2、`confirmSucceeded`；「绝不 `Close` `injectedExclusive`」、「`Stop` → 等 `flowBothDone` → `waitFlow` → `closeRestoreLease`」、`restore_unknown`／`ErrAdmissionIncomplete` fail-closed 分类均保留。
- 变更行内 `ADR-0031` 命中 0；`:4083` 的「约束 8」只出现在 r66.11 声明式否定（「不构成约束 8 满足」），不构成满足声明。ADR-0031 不变量 1–6 的设计表述未被破坏。
- 未变更行抽查（5,471 行逐字节相同）：strict v1、不经 shell（`:3880`／`:3892`／`:4446`／`:4755`）、私有 stdin（`:3882`）、ENV allowlist（`:83`）、Begin 前 closed、禁止以 stdout receipt／exit=0／DB 可连接自证（`:3888`）、`injectedExclusive` 永不 Close、执行器不响应 ctx 时持租约 fail-closed、r65.11／r65.12 写集建议与分流判据（全文各 9 处）均在。
- `F004` 文本层：`:3603` 未变；ADR 仍 `adr-0031-r10`，`NewRuntime` 命中 0（`framework.md` 285,639 B / `B432E44A…B050`），维持升级待决。
- `F017`：`tasks.md` 898,536 B / `4BD724A9…9FDA`，`:3186` 仍为 `- [ ]`，未被擅改。

## 3. 新 finding

无。`NEW-R10-nn` 本轮未开。

非阻断观察（不计入开放项）：`:4076` 的 `waitFlow` 现役实现（`recovery_runtime.go:932`–`:938`）尚未加入 ctx 分支的 `flowDone` 重查，属设计已固定目标态后的实现差量；实施时必须按合同改写。

## 4. 独立复算

### 4.1 冻结与基线真实性

| 项 | 实测 |
|---|---|
| 现场 `design.md` | 1,103,152 B / 5,476 行 / CR=169 / LF=5,476 / `3A8A9E63…D5254`（等于冻结值） |
| 冻结快照 | 1,103,152 B / 同哈希（逐字节一致） |
| r69 基线 | 1,099,874 B / 5,476 行 / `41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09`（等于 r69 对象哈希，是真实改前字节） |
| r68 基线 | 1,093,211 B / `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083` |
| 中间态 | r10 目录只有 `design.r69.baseline.md`／`report.md`／`hashes.txt`；基线即改前终值，未被中间态污染；作者声明一次写盘，实测终值即对象哈希 |

### 4.2 字节账（5 行 / +3,278 B / 0 行净增减）

| 行 | r69 B | r70 B | Δ |
|---|---:|---:|---:|
| `:5` | 39 | 39 | 0 |
| `:38` | 50 | 50 | 0 |
| `:4072` | 6,806 | 8,022 | +1,216 |
| `:4076` | 19,973 | 21,950 | +1,977 |
| `:4083` | 511 | 596 | +85 |
| 合计 | | | **+3,278** |

文件级：1,103,152 − 1,099,874 = **+3,278**，配平；行数 5,476 → 5,476（0 增删）；CR 169 → 169；LF 5,476 → 5,476。作者声称成立。

### 4.3 golden vector（`:4099`／`:4105`）

方法：`[IO.File]::ReadAllLines(p, UTF8)[n-1]` 取单行原文 → `[Text.Encoding]::UTF8.GetBytes(line)`（不含行尾 CR/LF）→ SHA-256。

| vector | 行 | 字节 | 实测 SHA256 | 期望 | 判定 |
|---|---|---:|---|---|---|
| source | `:4099` | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 / `de914622…` | PASS |
| tool | `:4105` | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 / `8c0c89…` | PASS |

### 4.4 三段后缀（raw bytes：LF 行首偏移 → EOF，不重编码、不归一）

扫描原始字节按 `0x0A` 建立行首偏移表（LF=5,476 → 5,477 个行首）：

| 行 | 起点字节 | 字节数 | SHA256 |
|---|---:|---:|---|
| `:4259` | 819,672 | 283,480 | `e8848838df83bc09bd8c94297219702cd86bd6629bd31fd8853f834f652cf0b6` |
| `:4275` | 821,952 | 281,200 | `2f53882f8774d30a51754872449ba29b4f257253c8a2f457ab6b9109d054fafb` |
| `:4290` | 824,937 | 278,215 | `b0af3b04c60584cf2618c83605e7fba88f65ca6b48f486463570268279e05f04` |

与派发期望逐项一致。r69 基线起点 816,394／818,674／821,659，r68 基线 809,731／812,011／814,996，r67 基线 801,402／803,682／806,667，三段字节数与 SHA 均相同；`819,672 − 816,394 = 3,278` 等于本轮增量，证明全部改写都在 `:4259` 之前。`review-sec-r09` 的 283,444 B／`98d2a9a5…` 未采用（其 283,444 = 1,103,152 − 819,708，起始偏移与预期差 36 B）。

### 4.5 括号平衡

全文全角括号 **1,216 / 1,216**；`:4076` 与 `:4072` 无未配对括号。终值平衡，与作者声称一致。

### 4.6 作者声称抽样核对（只作线索，数值自算）

- 5 行字节表与我的结果逐格一致；+3,278 配平；0 行增删；r69 基线与改前对象同哈希。
- golden vector、三段后缀、括号、`close(a.flowBothDone)` 单点判据均一致；r68 `hashes.txt` 缺口与 r09／r10 节 [5] 登记均属实。
- 未发现作者报告与设计正文的相反陈述（r09 的 `NEW-R09-02` 类问题本轮未复现）。

## 5. 与并行结论的关系

`review-arch-r09` 的 2 项开放项我按其原文重核并独立复算：`NEW-R09-01` 的机制描述对应现役 `setFlowResult`／`waitFlow` 代码锚点成立，r70 修复后关闭；`NEW-R09-02` 的 r68 缺口描述属实，r70 改口后可核。更早的 `review-arch-r06`／`r08` 只作线索，未以其结论作为接受依据。`review-sec-r09` 属安全轴，本轮仅对其 `NEW-S09-01` 做一句跨轴复核，其余 **NOT REVIEWED**，未以其数值（含错值 283,444）为判据。

## 6. NOT RUN

- `go build` / `go vet` / `go test` / `-race`：**NOT RUN**（设计文本复审；`flowWG`／`flowBothDone` 为目标态，当前代码不存在）。
- 真实 restore、真实备份／旧库、E2E：**NOT RUN**。
- `enter`／`resume` 实际进入与解锁：**NOT RUN**；生产候选源未交付，按设计字面进入即 `ErrAdmissionClosed`。本报告不声称 `enter`／`resume` 解锁。
- 生产 restore 解锁、约束 8 满足、`identity-auth-S4.24a` 可关闭或已交付：**均不成立**，本报告不作此声称。
- `SEC-F007`：**NOT RUN**，不因设计文字改判 PASS。
- Linux 侧能力（host／ACL／mount／loader／fd-exec／maps／network／witness）：**NOT RUN**。
- ADR-0031 `adr-0031-r10` 全文复审：**NOT REVIEWED**（并行会话维护）。

## 7. 复审条件与结论

- `NEW-R09-01` 已按唯一合同关闭；实施时按 `:4076` 改写 `waitFlow`（ctx 分支重查）并保留 (a)(b) 两顺序测试。
- `NEW-R09-02` 已按缺口如实登记关闭；后续修订只需在各自 `hashes.txt` 登记当次后缀值，不得回填 r68。
- `NEW-S09-01` 跨轴复核关闭；reaper 单点与 `close(flowBothDone)` 单次判据保留。
- 本轮判定 **PASS**，开放项 BLOCKER 0 / MEDIUM 0 / LOW 0；`N005` 仅登记为范围外。

## 8. 写入

本会话写入：`review-arch-r10/report.md`、`review-arch-r10/hashes.txt`。此外零写入。