# 设计修订报告 r69 — identity-auth 详细设计（承接两路 r08 复审）

- 日期：2026-11-02
- 角色：Feature-Designer（`add-feature-designer`；主方法 `add-design-tdd`）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 修订对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 方法依据：`.codex/agents/add-feature-designer.toml`、`.codex/skills/add-design-tdd/SKILL.md`、`design-doc-writing-style`、`abstraction-principles`

## 1 对象、基线与本轮终值

| 时点 | 字节 | 行数 | 换行 | SHA256 | 版本 |
|---|---:|---:|---|---|---|
| 改前 r68（冻结） | 1,093,211 | 5,476 | CR=169 / LF=5,476 | `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083` | `identity-auth-design-ir01-r68` |
| 改后 r69（本轮终值） | 1,099,874 | 5,476 | CR=169 / LF=5,476 | `41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09` | `identity-auth-design-ir01-r69` |

- 改前快照：`design-revision-r09/design.r68.baseline.md`，1,093,211 B / 5,476 行 / `5E324AF9…`（与写盘前 `design.md` 逐字节相等，已核）；UTF-8 无 BOM（首 3 字节 45,45,45）。
- 版本号两处一致：`:5` = `revision: identity-auth-design-ir01-r69`；`:38` = **设计版本**：`identity-auth-design-ir01-r69`。
- 行数 0 增删；`:4072` / `:4076` / `:4083` / `:4099` / `:4105` 等锚点行号未移动。

### 1.1 判据报告（逐字读取，自行复算）

| 报告 | 字节 | 行数 | SHA256 |
|---|---:|---:|---|
| `review-arch-r08/report.md` | 26,972 | 257 | `80658FC1222BC1A42EBF0BB2C430AD3A61082F1AF94A7FBBE29416FBA61F0192` |
| `review-sec-r08/report.md` | 15,389 | 118 | `BD78FD4D9A020A9CD988D98C13099142518ABF0E4A61D23FDE59A471B3EF63A3` |

两份报告的开放项恰好对应本轮 4 项（其中 `NEW-R08-02` 与 `NEW-S08-02` 同根，合并闭合）。

### 1.2 只读与写入声明

本轮唯一写入：`design.md` 一次 + 本目录 `report.md`、`hashes.txt`、`design.r68.baseline.md`。未改 `tasks.md`（含 `:3186` 复选框）、ADR（`framework.md`，`adr-0031-r10` 并行会话改动未回退）、`cmd/**`、`internal/**`、测试、任何既有 `review-*/` 与 `design-revision-r01..r08/`；未删除、未移动、未归档；未执行任何 Git 写操作。

### 1.3 写盘方式（可复算）

读取 r68 为字符串 → 内存中替换 6 行 → 写工作区外临时预览 → 复核 `design.md` 仍为冻结值 `5E324AF9…` → `[IO.File]::Copy(preview, design.md, $true)` 一次覆盖。`design.md` 仅一次写盘，无中间态；报告中全部数值均为该终值实测。

## 2 逐项闭合

### 2.1 `NEW-R08-01`（MEDIUM，架构）— CLOSED

- 落点：`:4076`（`enter` 的 secret 唯一取得路径）、`:1682`（Enter 责任列）、`:4072`（r69.1 登记）。
- 问题：r68 字面 `RecoveryInput{RunBinding: …}` 不传 `Secret`，而 `Enter` 在 `runtime.go:1522` 于 `beginAction`（`:1525`）之前返回 `ErrAdmissionBinding`；`:1682` 要求的「本次 secret」在设计内无 root 侧来源，与「Secret 仍走既有候选通路」自相矛盾。
- 修改摘要：
  - 唯一取得路径写死：`enter` goroutine 在 `bounded` 建立后、调用 `Enter` 之前，恰一次 `candidate, secretErr := a.composition.ports.Secret.ReadInitialRecoverySecret(bounded, binding)`（`binding = a.plan.config.command.binding`），并以显式 `recovery.RecoveryInput{RunBinding: binding, Secret: candidate}` 传入。
  - 绑定与零化：`candidate` 与 `Enter` 内 `in.Secret` 是同一 buffer，`Enter:1518` 的 `defer clear(in.Secret)` 负责零化，root 侧只可 `defer clear(candidate)` 幂等防御，不得复制、缓存或重读。
  - 失败分类：`secretErr != nil` → `setFlowResult(secretErr)` + `requestStop()` 后返回，**不调用 `Enter`**、不发生状态变更；生产默认候选源未交付为 `ErrAdmissionClosed`（`recovery_runtime.go:631`–`:633`），空 `ctx` / binding 为 `ErrAdmissionBinding`（`recovery_sources.go:416`–`:418`），`bounded` 到期 / 取消原样透传 `context.DeadlineExceeded` / `context.Canceled`，不包装、不降级。
  - 唯一空值判定点：空候选不在 root 侧另判，仍显式传入，由 `Enter:1522` 在 `beginAction:1525` 之前返回 `ErrAdmissionBinding`。
  - 删除矛盾表述：全文「候选通路」出现 **0** 次；不另设配置键、别名或 root secret 环境变量；`resume` 的 `prepareTrust` 惰性读取保持原样（`Resume` 不显式传 secret）；`install` 起点仍 closed。
- 闭合判定：**CLOSED**（设计文本层）。设计内现已存在可按字面实施的 secret 取得点、传递方式、nil 判定与失败分类，且不再与实现冲突。

### 2.2 `NEW-R08-02`（MEDIUM）＋ `NEW-S08-02`（LOW，同根）— CLOSED

- 落点：`:4076`（双路 join 屏障、`joinWork` 严格顺序、执行器不响应 ctx 分类、机核判据）、`:4072`（r69.2 登记）。
- 问题：`setFlowResult` 的 `sync.Once` 关闭 `flowDone` 后 `waitFlow` / `joinWork` 立即返回且不 `Stop`，另一条 goroutine 未被 join，`closeRestoreLease` 可能与在飞 `injectedExclusive` 使用并发；「两条 goroutine 都经 `flowDone` 被 join」不成立。
- 修改摘要：
  - 语义拆分：明写 `flowDone` **只是首个结果信号、不是 join 信号**；`flowFinished()` 不得作为 `closeRestoreLease()` 的前置短路；`sync.Once` 只提供幂等、不提供顺序。
  - 双路屏障：assembly 增 `flowWG sync.WaitGroup` 与 `flowBothDone chan struct{}`（与 `flowDone` 同在 `recovery_runtime.go:617` 初始化）；两条业务 goroutine 启动前 `flowWG.Add(2)` 恰一次，两条 goroutine 函数体首行各自 `defer a.flowWG.Done()`，一次性 reaper `go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()` 不接触 lease、不调 `setFlowResult`。
  - teardown 严格顺序（`case "enter"` / `"resume"` 共用模板）：(1) **无条件** `Stop(ctx)`（`flowFinished()` 在这两个 case 中出现 0 次）；(2) 在 `joinWork` 的 `shutdownCtx`（`lifecycle.go:330`）上等 `flowBothDone`；(3) `waitFlow(ctx)` 只做非阻塞结果读取，不再承担 join；(4) **最后** `closeRestoreLease()`——租约释放严格晚于两条路径退出。
  - 执行器不响应 ctx：屏障超时 / 取消时**不得**关闭租约（fail-closed 持有，宁可泄漏）；已启动 → `restore_unknown`（`errors.Join(recovery.ErrRecoveryWriteUnknown, err)`），未启动 → `ErrAdmissionIncomplete`；保留原错误、写日志、不降级为 warning。
  - 机核判据：`Add(2)` 恰 1 次、`Done` 恰 2 处、`close(flowBothDone)` 恰 1 次且只在 `Wait()` 之后；`case "enter"` / `"resume"` 内 `flowFinished()` = 0；同一 case 按序 `Stop(` → `flowBothDone` → `waitFlow(` → `closeRestoreLease(`；`closeRestoreLease(` 不早于 `flowBothDone` 接收。
  - 断言修正：「两侧都被 join」全文出现 **0** 次。
- 闭合判定：**CLOSED**（MEDIUM 与 LOW 同根，一并闭合；设计文本层）。租约释放与双路退出之间现有明确屏障与可机核判据。

### 2.3 `NEW-R08-03`（LOW，架构）— CLOSED

- 落点：`:4072`（后缀复算规则句）；本轮 `hashes.txt`（新增「三段后缀（r69 时点）」一节）。
- 问题：r68 `hashes.txt` 未记后缀取值与取得时点，而 `:4072` 称后缀「继续变化」；实测 r68 与 r67 三段后缀逐字节相同。
- 修改摘要：`:4072` 改为条件式——「只有当 `:4259`、`:4275`、`:4290` 各自的行首之后内容被改写时，对应后缀才变化；r68 与 r69 的全部改写均在 `:4259` 之前，故二者三段后缀与 r67 逐字节相同，r68 / r69 时点值只登记在各自修订的 `hashes.txt`（含取得时点）」；可复算规则（行首字节偏移→EOF、raw bytes、不归一）与 r66 历史值保留。
- 实测佐证（r69 终值，取得时点 2026-11-02 17:07:41 +08:00）：`:4259` 起点字节 816,394 / 283,480 B / `e8848838…cf0b6`；`:4275` 起点 818,674 / 281,200 B / `2f53882f…54fafb`；`:4290` 起点 821,659 / 278,215 B / `b0af3b04…e05f04`——与 r67 基线（`design-revision-r08/design.r67.baseline.md`）三段逐字节相同。
- 字节账交叉印证：`816,394 − 809,731（r68 起点）= 6,663`，恰等于本轮全文增量，说明变化全部落在 `:4259` 之前。
- 闭合判定：**CLOSED**（设计文本层 + 本轮 `hashes.txt` 已记值与取得时点）。

## 3 无回退确认

全文件逐行 `-cne` 比对：r68→r69 仅 **6 行**变化（`:5`、`:38`、`:1682`、`:4072`、`:4076`、`:4083`），其余 5,470 行逐字节相同，因此 `:4070`（B1 重验、长名、pin 轮换）、`:4077`、`:4079`、`:4195` 及其余全部已闭合条款在文本层自动保持。

被改写的两处长行（`:4072`、`:4076`）内锚点计数复核：

| 锚点 | 结果 |
|---|---|
| `activeExclusiveLocked()`（`:4076` 内） | 8 次；`:919` 禁自取锁句在 |
| `:892` / `:1943` / `:1968` / `recoveryPreflight` | 均在（2 / 2 / 1 / 1） |
| `ownershipLease` / `recheckProof(ctx, r.deps, lease, v.proof)` | 4 / 1，F006(b) 数据流在 |
| `restoreDriverFactory` / `newControlledRestoreDriver` / `recoveryClosedRestoreDriver` / `s414Canary` | 2 / 2 / 2 / 1，NEW-R05-04 / NEW-R06-05/06 在 |
| 长名 `…APPROVAL_TRUST_ROOT_PATH` / `…_SHA256`（全文） | 4 / 5，NEW-S05-05 唯一字面在 |
| `载荷绑定`（全文） | 2，NEW-R05-03 / NEW-S05-06 限定语在 |
| r66 / r67 历史后缀值与「不得标为现行」句（`:4072`） | 保留 |

结论：`F006`(a)(b)、`NEW-R07-01/02/03`、`NEW-R05-03`、`NEW-R07-04` 文本层、`F004` 文本层、`NEW-R05-04/05`、`NEW-R06-02/05/06/07/08`、`SEC-F002/F003`、`N001`、`SEC-F006`、`N011`、`NEW-S05-01/02/04/05/06/07`、`N012`–`N016`、`NEW-S06-02/03`、`NEW-S07-01` 均未回退。

## 4 字节账与校验

### 4.1 逐行字节账（r68→r69，0 行增删）

| 行 | r68 B | r69 B | Δ |
|---|---:|---:|---:|
| `:5` | 39 | 39 | 0 |
| `:38` | 50 | 50 | 0 |
| `:1682` | 342 | 599 | +257 |
| `:4072` | 5,746 | 6,806 | +1,060 |
| `:4076` | 14,712 | 19,973 | +5,261 |
| `:4083` | 426 | 511 | +85 |
| **合计** | | | **+6,663** |

文件级：1,099,874 − 1,093,211 = **+6,663**，配平；CR 169→169、LF 5,476→5,476、行数 5,476→5,476。

### 4.2 golden vector（`:4099` / `:4105`，无尾随换行）

| vector | 字节 | 实测 SHA256 | 期望 | 判定 |
|---|---:|---|---|---|
| source | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 / `de914622…` | PASS |
| tool | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 / `8c0c89…` | PASS |

### 4.3 文本卫生

- 全角括号：全文 **1,205 / 1,205**；`:4076` **107 / 107**。
- 陈旧串消除：`两侧都被 join`、`继续变化`、`候选通路` 全文均 **0** 次。
- 修订登记：`:4072` 含 `r69.1` / `r69.2` / `r69.3`，优先级规则扩为 `r69 > r68 > …`；`:4083` 追加 r69 沿用声明（不改变任何 NOT RUN / unsupported / closed 状态）。

## 5 升级待决（只登记，不做）

1. `F004`：`:3603` 的 production root 装配清单与 ADR-0031（`framework.md`，现 `adr-0031-r10`）的同步——ADR 由并行会话维护，本轮不改、不代表已同步。
2. `F017`：任务面对账（`tasks.md` 写集与 `:3603` / `:4072` 实现面的差异）——属 Task-Planner 范围，本轮不改 `tasks.md`。
3. 范围外 `N005`：`framework.md` 状态段与 `tasks.md:3197` 的冲突——只登记，不代改。

## 6 NOT RUN

- `go build` / `go vet` / `go test` / `-race`：**NOT RUN**（本轮为设计文本修订；`flowWG` / `flowBothDone`、`enter` secret 取得点等均为目标态，当前代码不存在）。
- 真实 restore、真实备份 / 旧库、E2E：**NOT RUN**。
- Linux host 层（host / ACL / mount / loader / fd-exec / maps / network / witness）：**NOT RUN**；`SEC-F007` 仍为 NOT RUN，不因本轮改判 PASS。
- `enter` / `resume` 实际进入与解锁：**NOT RUN**；当前生产候选源未交付，`enter` 实际进入即 `ErrAdmissionClosed`。
- ADR-0031 `adr-0031-r10` 全文复审：**NOT REVIEWED**（并行会话维护）。

## 7 复审条件

1. `NEW-R08-01`：按 `:4076` 字面复核 `enter` goroutine 的读取点、`RecoveryInput.Secret` 绑定、空值判定唯一性与三条失败分类。
2. `NEW-R08-02` / `NEW-S08-02`：按 `:4076` 机核判据逐条核对 `flowWG` / `flowBothDone` 与 `joinWork` 顺序；确认 `flowFinished()` 不在 `enter` / `resume` case 内出现。
3. `NEW-R08-03`：核对 `:4072` 条件式表述与本轮 `hashes.txt` 的后缀值＋取得时点。
4. 保留第 3 节无回退证据；不改动 `SEC-F007` 与 N005 边界。

## 8 声明

本轮只完成对 r68 的设计修订，不构成独立评审通过，不构成生产 restore 解锁，不构成 `enter` / `resume` 解锁，不构成约束 8 满足，不构成 `identity-auth-S4.24a` 可关闭或已交付，不勾选 `tasks.md` 任何复选框。最终接受由两路独立复审与 Lead / 用户决定；任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。
