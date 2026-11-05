# 架构独立复审报告 — identity-auth 详细设计 r69（review-arch-r09）

- 日期：2026-11-02
- 角色：add-reviewer（架构轴）；主方法 `.codex/skills/add-code-review/SKILL.md`（objectType=design）；规则 `code-review-checklist`、`abstraction-principles`、`design-doc-writing-style`、`decisions-guide`
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
  - **1,099,874 B / 5,476 行 / SHA256 `41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09`**
  - 版本 `identity-auth-design-ir01-r69`（`:5`、`:38` 一致）；CR=169 / LF=5,476
- 冻结快照：`C:\Users\J00010385\AppData\Local\Temp\design-r69-freeze-41D708D9.md`，1,099,874 B / 同 SHA256（与工作树逐字节一致）
- 漂移核对：**无漂移**。现场 `design.md` 哈希等于冻结哈希；`git status` 中其他未提交改动（decisions/、其他特性设计、`tasks.md`、`go.mod` 等）属并行会话，不是本对象，未作为判读依据。
- 基准报告：`review-arch-r08/report.md` 26,972 B / 257 行 / `80658FC1222BC1A42EBF0BB2C430AD3A61082F1AF94A7FBBE29416FBA61F0192`（与派发一致）
- 作者线索（只作线索，数值全部自算）：`design-revision-r09/report.md` 13,226 B / 143 行 / `7A9DE870758C22948FFAD15CA178CC83B6395BD0B956FC1B816B06F1338B8451`；`hashes.txt` 2,524 B / 37 行 / `92A23828A91EF94A4B6D971F138E6814E42EF5265C96904BAC7ABF51EE5BF2D2`；`design.r68.baseline.md` 1,093,211 B / `5E324AF9…`
- 方法：两轮。Round 1 对 6 行变更与两处被重写的长行逐句核对；Round 2 沿真实消费者（`cmd/han-server/recovery_runtime.go`、`recovery_sources.go`、`recovery_restore.go`、`lifecycle.go`、`main.go`、`internal/recovery/runtime.go`）与上游（`tasks.md`、`decisions/framework.md`）核契约、状态、teardown 与字节账；另独立复算 golden vector、三段后缀与括号平衡。
- 检查范围：冻结对象全文（逐行与 r68 基线 `-cne` 比对）、`:4072`／`:4076` 全文、`:1682`、`:4083`、`:4070`、`:4079`、`:4195`、`:3942`、`:3990`、`:3603`、`:4099`、`:4105`、`:4259`／`:4275`／`:4290`；r09 的作者报告与 `hashes.txt`；r08 与 r09 的基线快照；r08 报告中的原 finding 定义。
- 只读声明：本会话仅写本目录 `report.md` 与 `hashes.txt`。未修改 `design.md`、`tasks.md`、ADR（`framework.md`）、`cmd/**`、`internal/**`、测试、任何既有 `review-*/` 与 `design-revision-*`；未删除、移动、归档；未执行任何 Git 写操作。
- 判读过的文件：`design.md`（r69）、`%TEMP%\design-r69-freeze-41D708D9.md`、`design-revision-r09\design.r68.baseline.md`、`design-revision-r08\design.r67.baseline.md`、`design-revision-r09\report.md`、`design-revision-r09\hashes.txt`、`design-revision-r08\hashes.txt`、`review-arch-r08\report.md`、`review-arch-r07\report.md`（哈希核对）、`tasks.md`、`decisions\framework.md`、`cmd\han-server\recovery_runtime.go`、`cmd\han-server\recovery_sources.go`、`cmd\han-server\recovery_restore.go`（检索）、`cmd\han-server\lifecycle.go`、`cmd\han-server\main.go`（检索）、`cmd\han-server\recovery_runtime_test.go`（检索）、`internal\recovery\runtime.go`、`.codex\agents\add-reviewer.toml`、`.codex\skills\add-code-review\SKILL.md`、`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\guidance\design-doc-writing-style.md`、`.codex\guidance\decisions-guide.md`。

Verdict: CHANGES REQUIRED

- 开放项与阻断计数：BLOCKER 0 / MEDIUM 1 / LOW 1（`NEW-R09-01` MEDIUM、`NEW-R09-02` LOW）。r08 结转 3 项：`NEW-R08-01` 闭合、`NEW-R08-02`＋`NEW-S08-02` 闭合、`NEW-R08-03` 主体闭合后残余降级为 `NEW-R09-02`。范围外登记 `N005` 1 项，不计入阻断。

## 1. 开放项清单

| ID | 严重度 | 主题 | 位置 | 本轮判定 |
|---|---|---|---|---|
| `NEW-R09-01` | MEDIUM | `waitFlow` 没有单点合同，`:4076` 两处规范互相拉扯；按字面把结果读取做成非阻塞会把恢复失败报成 nil 成功 | `design.md:4076` | 新开 |
| `NEW-R09-02` | LOW | `:4072` 断言 r68 后缀值「只登记在 r68 `hashes.txt`（含取得时点）」不实，违反本段自定的引用规则 | `design.md:4072` | 新开（`NEW-R08-03` 残余） |
| `N005` | 范围外 | ADR `framework.md` 状态段与 `tasks.md` 的口径冲突 | `decisions/framework.md` / `tasks.md` | 只登记，不参与本判定，不代改 |

已闭合：`NEW-R08-01`（CLOSED）、`NEW-R08-02`＋`NEW-S08-02`（CLOSED，原缺陷）、`NEW-R08-03`（主体 CLOSED，残余转 `NEW-R09-02`）。

## 2. 逐项判定（对照 r08 开放项）

### 2.1 `NEW-R08-01`（MEDIUM，`enter` 不传 secret）— **CLOSED**

- 设计侧：`:4076` 现规定 `enter` goroutine 在 `bounded` 建立后、调用 `Enter` 之前恰一次 `candidate, secretErr := a.composition.ports.Secret.ReadInitialRecoverySecret(bounded, binding)`，`binding = a.plan.config.command.binding`，并以 `recovery.RecoveryInput{RunBinding: binding, Secret: candidate}` 传入；失败分支 `setFlowResult(secretErr)` ＋ `requestStop()` 后返回且不调用 `Enter`；空候选不在 root 侧另判，由 `Enter:1522` 在 `beginAction:1525` 之前返回 `ErrAdmissionBinding`。`:1682` 责任列已同步为「由 root 在 `start` 的 `enter` goroutine 内经 `ports.Secret.ReadInitialRecoverySecret` 一次取得」。
- 消费者核对（自行读代码，不采信作者）：
  - `recovery_runtime.go:43`–`:49` `recoveryRuntimeComposition.ports` 字段存在，`:26`–`:36` `recoveryRuntimePorts.Secret` 类型为 `recovery.RecoverySecretSource`；`:54`–`:58` 构造器对缺失／typed-nil `Secret` 返回 `ErrAdmissionBinding`。故 `a.composition.ports.Secret` 是可达字段，设计未编造路径。
  - `recovery_sources.go:415`–`:419` 对 nil 接收者、nil `ctx`、空 `RunID`／`BackupRef` 返回 `ErrAdmissionBinding`；`recovery_runtime.go:631`–`:633` 生产默认候选源返回 `ErrAdmissionClosed`——与设计的三条分类一致。
  - `internal/recovery/runtime.go:40`–`:43` `RecoveryInput` 含 `Secret []byte`；`:1517` `defer clear(in.Secret)`、`:1522` 空值检查、`:1525` `beginAction`、`:1530` `f.enter`，与设计的时点与唯一空值判定一致。
- 结论：设计文本层存在可按字面实施的取得点、传递方式、nil 判定与失败分类；原「字面不传 Secret 必失败」缺陷闭合。`flowWG`／`flowBothDone` 等目标态在当前代码不存在属实现差量，不是本项缺陷。

### 2.2 `NEW-R08-02`＋`NEW-S08-02`（MEDIUM／LOW，双路 join 与 teardown 顺序）— **CLOSED（原缺陷）**

- 设计侧：`flowDone` 明写为「首个结果信号、不是 join 信号」；新增 `flowWG`／`flowBothDone`，`Add(2)` 在两条业务 goroutine 之前恰一次，两条各自 `defer a.flowWG.Done()`，一次性 reaper `Wait()` 后 `close(a.flowBothDone)`；`joinWork` 的 `enter`／`resume` 共用顺序 `Stop` → 等 `flowBothDone` → `waitFlow` → `closeRestoreLease`；屏障超时则 fail-closed 不关租约，已启动报 `restore_unknown`、未启动报 `ErrAdmissionIncomplete`；机核判据齐全。全文「两侧都被 join」= 0。
- 消费者核对：`lifecycle.go:330` 的 `joinWork(run.shutdownCtx)` 带 `shutdownBudget` 期限（`lifecycle.go:216`／`:218`），屏障等待有界；`start` 在生产只由 `main.go:770` 调用一次，一次性 channel 不会被二次关闭；`RestoreDriver.Replace` 为同步方法（`internal/recovery/runtime.go:55`–`:59` 注释「joins all tool work」），等待 `Enter` goroutine 退出即可回收工具工作，r68 旧句「有结果通道时必须读空」的意图被屏障覆盖。
- 结论：use-after-close 窗口已被屏障关闭；原 MEDIUM 与同根 LOW 闭合。**但同一段的 `waitFlow` 用法引入新问题，见 `NEW-R09-01`。**

### 2.3 `NEW-R08-03`（LOW，后缀复算账）— **主体 CLOSED，残余转 `NEW-R09-02`**

- `:4072` 已改为条件式：只有 `:4259`／`:4275`／`:4290` 行首之后的内容被改写时后缀才变化；r68 与 r69 的改写都在 `:4259` 之前，故与 r67 逐字节相同；可复算规则（行首字节偏移→EOF、raw bytes、不归一）保留。
- 我独立复算：`:4259` 起点 816,394 / 283,480 B / `e8848838…cf0b6`；`:4275` 818,674 / 281,200 / `2f53882f…54fafb`；`:4290` 821,659 / 278,215 / `b0af3b04…e05f04`。r68 基线三段起点 809,731／812,011／814,996，字节数与 SHA 相同；r67 基线 801,402／803,682／806,667，亦相同。`816,394 − 809,731 = 6,663`，等于全文增量。
- 残余：`:4072` 声称 r68 的时点值登记在 r68 的 `hashes.txt`，实测该文件没有后缀节（见 `NEW-R09-02`）。

### 2.4 无回退确认

- 逐行 `-cne` 比对 r68 基线：仅 `:5`、`:38`、`:1682`、`:4072`、`:4076`、`:4083` 六行不同，其余 **5,470 行逐字节相同**。因此 `:4070`（B1 重验／长名／pin 轮换）、`:4077`、`:4079`、`:4195`、`:3942`、`:3990`、`:3603` 等已闭合条款在文本层不可能回退。
- `:4072` 重写保留：`282,599`／`7D1DDC08`、`280,319`／`5BD8E870`、`277,334`／`E40EB7A2`、`+881`、「只作历史」、「不得再标作 r61 段基线」、`282,706`／`a1a256f5`、`1,045,111`／`87D70DA8` 全部仍在（逐串核对）。
- `:4076` 重写保留：F006 第(1)–(10)点、`:892`／`:1943`／`:1968`／`recoveryPreflight`、`ownershipLease`、`restoreDriverFactory`、`newControlledRestoreDriver`、`recoveryClosedRestoreDriver`、`s414Canary`。被替换的 5 个旧片段正是 r08 判为不成立的「两侧都被 join」「候选通路」、旧 `joinWork` 顺序、旧 teardown 判据与旧 `start` 字面，替换是修复本身，不是删义务。
- ADR-0031 不变量 1–6 的设计表述未被破坏：`:4060`、`:4153` 的「ADR-0031 不变量 6」句不在变更行；六行变更内 `ADR-0031` 命中 0；`:4076` 的「不变量」是本文的「互斥不变量」，`:4083` 的「约束 8」是否定声明（不构成满足）。
- `F004` 文本层：`:3603` 逐字节未变；ADR 仍 `adr-0031-r10`、`NewRuntime` 0 命中（`framework.md` 285,639 B / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`），维持升级待决，不据此判阻断。
- `F017`：`tasks.md` 898,536 B / `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`，`:3186` 仍为 `- [ ]`，未被擅改。

### 2.5 `N005`（范围外）

`framework.md` 状态段与 `tasks.md` 的口径冲突仍存在，属另一会话／Task-Planner 范围。本报告只登记为范围外阻断，不据此判 `CHANGES REQUIRED`，不代改 ADR 或 `tasks.md`。

## 3. 新 finding

### 3.1 [MEDIUM] `waitFlow` 无单点合同；按字面实现会把恢复失败报成 nil 成功

- finding ID：`NEW-R09-01`
- 对象/修订：design，`identity-auth-design-ir01-r69`（`41D708D9…`）
- 位置：`design.md:4076`（`completeOnRequest` 段与 `joinWork` 第③步）
- 现状：同一 helper `waitFlow` 在同一行内被赋予两种读法。`completeOnRequest` 段要求「两者成功后 `setFlowResult(a.waitFlow(ctx))` 再 `requestStop()`」，即该调用必须等到 `Enter` 结果或可诊断的 ctx 错误；`joinWork` ③ 则写「`waitFlow(ctx)` **只读结果**：先非阻塞接收 `flowDone`（…屏障通过后必已关闭）并返回 `flowErr`，不得以 `select` 的随机分支把结果替换成 `ctx.Err()`」。r69 没有在任何一处定义 `waitFlow` 的共享合同（未关闭时是否阻塞、ctx 到期时如何重查），现役实现 `recovery_runtime.go:932`–`:939` 是阻塞 `select`，设计未确认沿用。
- 触发：若实现者按「只读结果／先非阻塞接收」把 `waitFlow` 写成非阻塞读取：`completeOnRequest` 在 `completeIdentity` 返回后、`Enter` goroutine 尚未 `setFlowResult` 时调用 `waitFlow`，得到零值 `nil`；`setFlowResult(nil)` 赢得 `flowOnce` 并关闭 `flowDone`；`Enter` 随后的失败被 `sync.Once` 丢弃。`joinWork` ③ 随屏障返回 `nil`，④ 关闭租约；`flowFinished()` 也在 `Enter` 未退出时为真。
- 影响：恢复失败被上报为成功（安全相关路径的错误结论），与本段「任何失败不得降级」「不得替换成 ctx.Err()」的意图相反。屏障仍保证租约不早于两条 goroutine 退出而关闭，故 `NEW-R08-02` 的 use-after-close 不复发；本项是结果传播的新缺口。
- 建议：在 `:4076` 单点写出合同，例如「`waitFlow(ctx)` 先非阻塞接收 `flowDone`；未关闭则在 `flowDone`／`ctx.Done()` 上等待；`ctx.Done()` 就绪时重查 `flowDone`，已关闭返回 `flowErr`，否则返回包装后的 `ctx.Err()`；本函数不承担 join」；或把 `joinWork` 的结果读取改为不接收 ctx 的独立 `flowResult()`，避免两个调用点共享含义不清的函数。
- 验证：单测中让 `completeIdentity` 返回后 `Enter` goroutine 延迟再 `setFlowResult(失败)`，断言 `joinWork` 返回该失败且租约在 `Enter` 退出前未关闭；反向用例断言成功路径仍返回 `nil` 且只关一次。

### 3.2 [LOW] `:4072` 对 r68 `hashes.txt` 的交叉引用不实，违反本段自定引用规则

- finding ID：`NEW-R09-02`
- 对象/修订：design，`identity-auth-design-ir01-r69`（`41D708D9…`）
- 位置：`design.md:4072`
- 现状：该段断言「r68 与 r69 的全部改写均在 `:4259` 之前，故二者三段后缀与 r67 逐字节相同，r68／r69 时点值只登记在各自修订的 `hashes.txt`（含取得时点）」。实测：r69 `hashes.txt` 第 [5] 节有值与取得时点（2026-11-02 17:07:41 +08:00）；r68 `hashes.txt`（3,459 B / 65 行 / `4770136A02F3C706AB60170E18CB0E26CE7252A2E39175CFD0F85380A9CE9757`，未被改动）没有后缀节，全文对 `4259`／`4275`／`4290`／`816394`／`818674`／`821659` 命中 0。同段又规定「引用时必须同时给出『源文件 SHA-256 + 行号 + 取得时点』」，该引用自相矛盾。
- 触发：读者或后续审计按 `:4072` 到 r68 `hashes.txt` 查后缀时点，查不到，无法区分「未登记」与「值不同」。
- 影响：审计链在这一项上不可核；后缀值本身无碍（我已独立复算，r68／r69 与 r67 逐字节相同）。作者 r09 报告 §2.3 自己也写「r68 `hashes.txt` 未记后缀取值与取得时点」，与设计正文的断言直接冲突。
- 建议：改为「r69 时点值登记在本轮 `hashes.txt`；r68 未登记后缀值，其值与 r67 相同由不变条件推得」，或删除对 r68 `hashes.txt` 的引用，保留条件式规则与 r67 基线。
- 验证：`design.md:4072` 不再出现「r68 … `hashes.txt`（含取得时点）」的断言；引用 r68 时只使用 r67 基线与条件式规则。

## 4. 独立复算

### 4.1 冻结、基线与中间态

| 项 | 实测 |
|---|---|
| 现场 `design.md` | 1,099,874 B / 5,476 行 / `41D708D9…BAB09`（等于冻结值） |
| 冻结快照 | 1,099,874 B / 同哈希（逐字节一致） |
| r68 基线 | 1,093,211 B / 5,476 行 / `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083` |
| r67 基线 | 1,084,882 B / `B12F1EF3…E064E`（用于后缀对照） |

r68 基线真实性：其哈希等于 r08 `hashes.txt` 的 `after.sha` 与 `write#3` 终值，也等于 r08 报告 `after` 值，是真实 r68 终值。r68 中间态 `write#1 A95470BA…`（1,093,138 B）与 `write#2 43BFF315…`（1,093,208 B）在 r09 目录命中 0（只出现在 `review-arch-r08/hashes.txt` 的历史记录），**未被误用为基线**。r69 作者声明一次写盘，实测文件哈希即终值，目录内无第二份设计副本。

### 4.2 字节账（6 行 / +6,663 B / 0 行净增减）

| 行 | r68 B | r69 B | Δ |
|---|---:|---:|---:|
| `:5` | 39 | 39 | 0 |
| `:38` | 50 | 50 | 0 |
| `:1682` | 342 | 599 | +257 |
| `:4072` | 5,746 | 6,806 | +1,060 |
| `:4076` | 14,712 | 19,973 | +5,261 |
| `:4083` | 426 | 511 | +85 |
| 合计 | | | **+6,663** |

文件级：1,099,874 − 1,093,211 = **+6,663**，配平；行数 5,476 → 5,476（0 增删）；CR 169 → 169；LF 5,476 → 5,476。作者声称成立。

### 4.3 golden vector（自行定位为 `:4099` / `:4105`）

方法：`[IO.File]::ReadAllLines(p, UTF8)[n-1]` 取单行原文 → `[Text.Encoding]::UTF8.GetBytes(line)`（不含行尾 LF/CR）→ SHA-256。

| vector | 行 | 字节 | 实测 SHA256 | 期望 | 判定 |
|---|---|---:|---|---|---|
| source | `:4099` | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 / `de914622…` | PASS |
| tool | `:4105` | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 / `8c0c89…` | PASS |

### 4.4 三段后缀（raw bytes：行首字节偏移→EOF，不重新编码、不归一）

扫描原始字节按 `0x0A` 建立行首偏移表（LF=5,476 → 5,477 个行首）：

| 行 | 起点字节 | 字节数 | SHA256 |
|---|---:|---:|---|
| `:4259` | 816,394 | 283,480 | `e8848838df83bc09bd8c94297219702cd86bd6629bd31fd8853f834f652cf0b6` |
| `:4275` | 818,674 | 281,200 | `2f53882f8774d30a51754872449ba29b4f257253c8a2f457ab6b9109d054fafb` |
| `:4290` | 821,659 | 278,215 | `b0af3b04c60584cf2618c83605e7fba88f65ca6b48f486463570268279e05f04` |

r68 基线同三项字节数与 SHA 完全相同（起点 809,731／812,011／814,996）；r67 基线亦相同（801,402／803,682／806,667）。起点差 6,663 = 本轮增量，证明全部改写都在 `:4259` 之前。

### 4.5 括号平衡

全文全角括号 1,205 / 1,205；新增重写行 `:4076` 107 / 107。终值平衡。r69 只有一次写盘，不存在被引用的中间失衡态。

### 4.6 作者报告声称抽样核对（只作线索，数值自算）

- 6 行字节表与我的结果逐格一致；+6,663 配平；0 行增删。
- 锚点计数一致：`:4076` 内 `activeExclusiveLocked()` 8；`:892` 2、`:1943` 2、`:1968` 1、`recoveryPreflight` 1；`ownershipLease` 4；`restoreDriverFactory` 2、`newControlledRestoreDriver` 2、`recoveryClosedRestoreDriver` 2、`s414Canary` 1。
- 长名 4 / 5 一致（`APPROVAL_TRUST_ROOT_PATH` 4、`APPROVAL_TRUST_ROOT_SHA256` 5）；`载荷绑定` 2；「两侧都被 join」「继续变化」「候选通路」全文 0。
- 括号 1,205 / 1,205 与 `:4076` 107 / 107 一致；golden vector 一致。
- 唯一不一致：作者报告 §2.3 与设计 `:4072` 对 r68 `hashes.txt` 是否登记后缀值的陈述相反（见 `NEW-R09-02`）。

## 5. 与并行结论的关系

上轮 `review-arch-r08` 的 2 MEDIUM＋1 LOW 我按其原文重核：`NEW-R08-01` 与 `NEW-R08-02` 的机制描述在当前代码中成立（`setFlowResult` 的 `sync.Once`、`waitFlow` 阻塞读、`joinWork` 旧分支），其关闭判定我独立复算后同意。更早的 `review-arch-r06` 只作线索；本轮未以其结论作为接受依据。`review-sec-r08` 属安全轴，本轮 **NOT REVIEWED**，其哈希仅由作者报告转述，未用于本判定。

## 6. NOT RUN

- `go build` / `go vet` / `go test` / `-race`：**NOT RUN**（设计文本复审；`flowWG`／`flowBothDone` 等为目标态，当前代码不存在）。
- 真实 restore、真实备份／旧库、E2E：**NOT RUN**。
- `enter`／`resume` 实际进入与解锁：**NOT RUN**；生产候选源未交付，按设计字面进入即 `ErrAdmissionClosed`。本报告不声称 `enter`／`resume` 解锁。
- 生产 restore 解锁、约束 8 满足、`identity-auth-S4.24a` 可关闭或已交付：**均不成立**，本报告不作此声称。
- `SEC-F007`：**NOT RUN**，不因设计文字改判 PASS。
- Linux 侧能力（host／ACL／mount／loader／fd-exec／maps／network／witness）：**NOT RUN**。
- ADR-0031 `adr-0031-r10` 全文复审：**NOT REVIEWED**（并行会话维护）。

## 7. 复审条件

1. `NEW-R09-01`：在 `:4076` 给出 `waitFlow` 的单点合同（或拆分结果读取函数），使 `completeOnRequest` 与 `joinWork` 两处调用语义唯一；给出「先失败后成功」与「先成功后失败」两种顺序的可机核判据。
2. `NEW-R09-02`：修正 `:4072` 对 r68 `hashes.txt` 的引用；保留条件式规则与 r67 基线对照。
3. 保留本轮 6 行字节账、golden vector、后缀与括号的复算证据；修订后重跑同一组复算，任何新增行不得引入新的未闭合断言。

## 8. 写入

本会话写入：`review-arch-r09/report.md`、`review-arch-r09/hashes.txt`。此外零写入。
