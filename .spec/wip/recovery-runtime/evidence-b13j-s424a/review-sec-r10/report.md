# SEC r10 独立复审报告 —— identity-auth 详细设计 r70

- 日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：`.add\project\knowledge\design-workspace\identity-auth\design.md`
  - 冻结值：**1,103,152 B / 5,476 行 / CR=169 / LF=5,476 / SHA256 `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`**
  - 版本：`identity-auth-design-ir01-r70`（`:5`、`:38` 一致）
  - 冻结快照：`C:\Users\J00010385\AppData\Local\Temp\design-r70-freeze-3A8A9E63.md`，字节/行/CR/LF/摘要与上值完全一致（已核）；现场 `design.md` 与快照同哈希、同 mtime，无漂移
- 基准报告：`.spec\wip\recovery-runtime\evidence-b13j-s424a\review-sec-r09\report.md`
  - 13,848 B / `8D74FA3BC39AD6C8A0C99E7AB7F56614FACCFA8D19F6A40D5FFE11989CEB4A51`（Verdict: CHANGES REQUIRED；开放 `NEW-S09-01` LOW）
- 基线对象：`.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r10\design.r69.baseline.md`，1,099,874 B / 5,476 行 / `41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09`（已核，与派发值一致）
- 作者材料（只作线索，不采信数值）：`design-revision-r10\report.md`（13,214 B / `724130C867DA7A134DF964C033E731F118F75CCD100518399F2AB81BD52D98EE`）、`design-revision-r10\hashes.txt`（4,729 B / `6635624ADDBEA8458206E6500128EE38198FEAA700C6BDA40258E3E96D16BB36`）

## 0 方法、检查范围与只读声明

方法：按 `add-reviewer` / `add-code-review`（objectType=design）对冻结 r70 做独立安全侧复审。第一轮以 `[IO.File]::ReadAllBytes` 取基线（r69）与快照（r70）原始字节，按 LF 切行做 Ordinal 逐行比对定位变更，并对全文做字面扫描（`flowBothDone`／`flowWG`／`reaper`／`各起`／`waitFlow`／`closeRestoreLease`／`setFlowResult`／`flowFinished`），核实「reaper 唯一性」「关闭点唯一性」「Add 先于 Wait」；第二轮对 5 处变更行全文判读，沿契约链（`:4037`／`:4062`／`:4068`–`:4083`／`:1481`／`:1682`／`:3880`–`:3888`／`:4420`／`:4446`）复核 fail-closed、secret 面与无回退，并在快照上独立复算两条 golden vector、三段后缀与字节账。所有数值本轮自算；作者报告与 `hashes.txt` 只用于对照，不用于取值。

检查范围（判读过的文件，均为只读）：
- `.add\project\knowledge\design-workspace\identity-auth\design.md`（冻结快照 `design-r70-freeze-3A8A9E63.md`；逐行比对 + `:5`／`:38`／`:4072`／`:4076`／`:4083` 全文 + 全文扫描）
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r10\design.r69.baseline.md`（基线比对；对照 `:4076` 原文以确认被删除的缺陷句）
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\review-sec-r09\report.md`（基准判据全文）
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\review-arch-r09\report.md`（20,732 B / `764406E81FEEA90A26B1C34C2DC81F3B5298C07C06F8CBE9A39366358B426C6D`；只读 `NEW-R09-01`／`NEW-R09-02` 条目与其复审条件）
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r10\hashes.txt`、`design-revision-r09\hashes.txt`（2,524 B / `92A23828A91EF94A4B6D971F138E6814E42EF5265C96904BAC7ABF51EE5BF2D2`）、`design-revision-r08\hashes.txt`（3,459 B / `4770136A02F3C706AB60170E18CB0E26CE7252A2E39175CFD0F85380A9CE9757`）
- `.add\project\knowledge\tasks\tasks.md`（898,536 B / `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`；只读 `:3186`、`:3197`）
- `.add\project\knowledge\decisions\framework.md`（285,639 B / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`；只读 `:1084`、`:1129` 与 `adr-0031-r10` 引用）

只读声明：本轮只写入本文件（含目录 `review-sec-r10`）；未改 `design.md`、`tasks.md`、ADR／`framework.md`、`cmd/**`、`internal/**`、测试、既有 `review-*`、`design-revision-*`；未删除、移动、归档任何文件；未执行任何 Git 写操作（仅 `git status`／`rev-parse` 只读命令）。冻结对象现场哈希与快照一致，无漂移。

## 1 结论

**Verdict: PASS**

| 类别 | 计数 | 说明 |
|---|---|---|
| BLOCKER | 0 | — |
| MEDIUM | 0 | — |
| LOW | 0 | `NEW-S09-01`（r09 唯一开放项）本轮判定**闭合** |
| 非阻断建议（不计入缺陷计数） | 1 | `NEW-S10-01`（`:4076` 机核判据的静态计数口径建议，无运行时安全后果） |
| NOT RUN（不计入缺陷计数） | 1 | `SEC-F007` 及 Linux 侧能力（POSIX 宿主隔离、进程树 reap、专用执行身份实测） |
| 范围外 BLOCKER（仅登记） | 1 | `N005`（ADR 状态段与 `tasks.md` 冲突；不据此判 CHANGES REQUIRED，不代改） |

开放项清单：**无阻断开放项**。本轮闭合：`NEW-S09-01`（LOW）、`NEW-S08-02`（r08 结转，随 `NEW-S09-01` 闭合）、`NEW-R09-01`（MEDIUM，跨轴）、`NEW-R09-02`（LOW，跨轴）。非阻断：`NEW-S10-01`（建议）、O1／O2／O4（沿用 r09 的非阻断残余）。仍存边界：`SEC-F007` = NOT RUN；`N005` = 范围外仅登记。

## 2 r70 变更与字节账（独立复算）

以原始字节按 LF 切行做 Ordinal 比对：两版均为 5,477 个元素（5,476 行 + 末尾 LF 后的空元素），**恰好 5 行不同**（`:5`、`:38`、`:4072`、`:4076`、`:4083`），其余 5,471 行逐字节相同；行数 5,476 → 5,476（0 行净增减），CR 169 → 169，LF 5,476 → 5,476。

| 行 | r69 字节 | r70 字节 | 增量 | 内容 |
|---|---|---|---|---|
| `:5` | 39 | 39 | 0 | `revision: identity-auth-design-ir01-r70` |
| `:38` | 50 | 50 | 0 | 设计版本升 r70 |
| `:4072` | 6,806 | 8,022 | +1,216 | r70.3：r68 后缀值改判为登记缺口，指向 r69 `hashes.txt` 节 [5]；新增 r70 修订行 |
| `:4076` | 19,973 | 21,950 | +1,977 | r70.1／r70.2：`waitFlow` 单点合同、reaper 唯一化与机核判据 |
| `:4083` | 511 | 596 | +85 | r70 沿用 r66.11 声明（不改 NOT RUN／unsupported／closed） |
| 合计 | — | — | **+3,278** | = 1,103,152 − 1,099,874 |

结论：**5 行、+3,278 B、0 行净增减**与作者声称一致；文件首 3 字节为 `---`、末字节 LF、无 BOM；5 处变更行均为完整句子（`:4076` 以「只允许命中上述两处工厂赋值。」收尾，`:4072` 以 r70.3 句收尾，`:4083` 以 r70 声明句收尾），无截断或括号失衡残留。写入方式（单次写盘/10 处子串替换）属作者过程自述，本轮只能核验终值，不采信过程描述。

## 3 逐条判定

### 3.1 `NEW-S09-01`（LOW）：reaper 数量与双重关闭 —— **CLOSED**

缺陷原文（r69 基线 `:4076`，已读原句）：「`enter`／`resume` 在起两条业务 goroutine **之前**各执行恰一次 `flowWG.Add(2)`，两条 goroutine 在函数体首行各自 `defer a.flowWG.Done()`，**并各起一次性 reaper** `go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()`」。该句若按字面实现，两条业务 goroutine 各起一个 reaper，两个 reaper 都会 `close(a.flowBothDone)`，二次关闭 panic。

r70 后的可核证据（`:4076`）：
- **reaper 唯一**：「两条业务 goroutine 在函数体首行各自 `defer a.flowWG.Done()`、都**不**起 reaper，reaper 只由 `start` 在两条业务 goroutine 之前起 **1 个**（`go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()`，不调用 `setFlowResult`、不接触 lease；本设计不为 `flowBothDone` 设 `sync.Once` 兜底，因为不存在第二个关闭点）」。
- **Add 先于 Wait**：`start` 段固定顺序「先 `a.flowWG.Add(2)`、再起一次性 reaper、最后起两条业务 goroutine」，故 `Wait()` 不可能先于 `Add(2)`，无 WaitGroup 误用竞态。
- **Done 两条**：两条业务 goroutine 函数体首行各 `defer a.flowWG.Done()`，且 `secretErr != nil` 的提前返回、`Enter` 返回、`completeOnRequest` 各出口都在 `defer` 覆盖内，不存在漏 Done 导致的永久阻塞。
- **关闭点唯一（全文扫描）**：`close(a.flowBothDone)` 仅出现在 `:4076`（4 次：`start` 段的启动规格 1 次、双路屏障段对同一启动的重述 1 次、机核判据引用 2 次）与 `:4072`（1 次，r70.2 历史说明）；**规范性关闭点只有 1 个**，且位于 `a.flowWG.Wait()` 之后。
- **无残留「各起」**：全文 `各起` 仅 1 处，位于 `:4072` 的 r70.2 说明「删除『各起一次性 reaper』」，属历史说明，不是规范句；`:4076` 已无「各起 reaper」表述。
- **机核判据同步**：①`flowWG.Add(2)` 在两条业务 goroutine 启动前恰 1 次、两条 goroutine 各 `defer a.flowWG.Done()` 恰 1 次且均不起 reaper、启动 `go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()` 恰 1 次、`close(a.flowBothDone)` 恰 1 次且只在 `flowWG.Wait()` 之后；②两个 case 内 `flowFinished()` 出现 0 次；③同一 case 内按序 `Stop(`、`flowBothDone`、`waitFlow(`、`closeRestoreLease(`；④`closeRestoreLease(` 不早于对 `flowBothDone` 的接收。

判定：`NEW-S09-01` 的 panic 成因（第二个关闭点）已消除，`flowBothDone` 的关闭由唯一的 `flowWG.Wait()` 后置点产生，**闭合**。

### 3.2 跨轴 `NEW-R09-01`（MEDIUM）：`waitFlow` 单点合同 —— **CLOSED**

缺陷原文（r69 基线 `:4076`）：「③`waitFlow(ctx)` **只读结果**：先非阻塞接收 `flowDone`……返回 `flowErr`」。非阻塞读取在 `flowDone` 未关闭时返回零值 nil，会把「`Enter` 失败尚未落结果」误报为成功，从而使 `joinWork` 以 nil 收敛并提前 `closeRestoreLease()`。

r70 后的可核证据（`:4076`）：
- **唯一实现、两点共用**：`waitFlow(ctx)` 单点合同（`NEW-R09-01`，唯一实现，两个调用点共用）；函数体是阻塞 `select`：`case <-a.flowDone:` 返回已记录 `a.flowErr`（`flowErr` 在 `close(flowDone)` 之前写入，关闭即 happens-before 可见）；`case <-ctx.Done():` 先**重查一次** `flowDone`，已关闭返回 `flowErr`，仍未关闭才返回可诊断的 `context.DeadlineExceeded`／`context.Canceled`（允许包装、不改分类）。
- **fail-closed 未被削弱**：`flowDone` 未关闭前不得返回任何结果值（包括 nil）、不得用 `select default` 返回哨兵、不得用 nil 表示「尚无结果」；唯一不以 `flowErr` 收尾的出口是 ctx 先到期且重查后仍未关闭。
- **顺序证据**：两条业务路径的每个出口都先 `setFlowResult` 再返回，故 `flowBothDone` 关闭蕴含 `flowDone` 已关闭；`waitFlow` 不承担 join，`joinWork` 第③步在 `flowBothDone` 之后调用它只读结果。
- **两种顺序的可机核判据**：(a) `Enter` 先失败、`completeOnRequest` 后到 —— `waitFlow` 必须返回该失败（不得返回 nil），随后 `requestStop()`；(b) `completeOnRequest` 先到而 `Enter` 未落结果 —— `waitFlow` 阻塞于 `flowDone`／`ctx.Done()`，不得以 nil 关闭 `flowDone`，必须等到 `Enter` 结果或 ctx 错误。
- **teardown 顺序未被改写**：`case "enter"`／`case "resume"` 共用模板 ①无条件 `Stop(ctx)`（`flowFinished()` 不得作提前返回）②在 `shutdownCtx` 上等 `flowBothDone` ③`waitFlow` 只读结果 ④最后 `closeRestoreLease()`；屏障超时分支**直接返回、不关闭租约**，按「已启动 → `restore_unknown`（`errors.Join(recovery.ErrRecoveryWriteUnknown, err)`）／尚未启动 → `ErrAdmissionIncomplete`」分类并写日志，不得降级为 warning。
- **执行器不响应 ctx 仍持租约**：若②在 ctx 到期/取消仍未通过，明令不得 `closeRestoreLease()`（宁可泄漏也不得在可能仍在飞的使用中关闭），未 join 的 goroutine 与租约由进程退出回收。

判定：`waitFlow` 的「未关闭前返回 nil」路径已被合同消除，nil 不再可能吞掉 `Enter` 失败；租约释放仍严格晚于两路退出；执行器不响应时 fail-closed 保持。**闭合**。

### 3.3 `NEW-S08-02`（r08 结转，LOW）：双路 join 与租约顺序 —— **CLOSED**

- 双路均存在真实 join 屏障：`enter` 与 `resume` 共用 `joinWork` 的 `Stop → flowBothDone → waitFlow → closeRestoreLease` 模板；`:4076` 明写「不存在只启动不 join 的 `enter` 路径」，机核判据 `rg -n 'case "enter"' cmd/han-server/recovery_runtime.go` = 1。
- 租约释放前不会出现两路并发持有：`closeRestoreLease()` 仅在两条路径都已从函数体退出（`flowBothDone` 关闭）之后调用，且 `sync.Once` 保证 `enter`／`resume`／错误路径／`joinWork` 合计恰好关闭一次；屏障超时分支不关闭租约。
- r09 将其保持开放的唯一原因（`NEW-S09-01` 的 reaper 矛盾）已闭合，故本轮判 **CLOSED**。

### 3.4 `NEW-R09-02`（LOW，跨轴）：r68 后缀值登记陈述 —— **CLOSED**

- r70 `:4072` 现文：「r68 时点值**未登记**（`design-revision-r08/hashes.txt` 无后缀节，属登记缺口，本句如实标注），r68 与 r69 的三段后缀逐字节相同由 r69 `hashes.txt` 节 [5] 的『对照 r68 基线……三段字节数与 SHA256 相同』佐证」。
- 独立核验：`design-revision-r08/hashes.txt`（3,459 B / `4770136A…`，65 行）确无 `:4259`／`:4275`／`:4290`／「后缀」条目；`design-revision-r09/hashes.txt` 节 [5] 确有「对照 r68 基线（design.r68.baseline.md）：startByte=809731 / 812011 / 814996，三段字节数与 SHA256 相同」与 r67 对照；我在 r67／r68／r69 三个基线上按 `:4259` 起点至 EOF 复算，均为 283,480 B / `e8848838…`，逐字节相同。**陈述属实，闭合**。

### 3.5 `enter` 的 secret 取得路径与泄露面

- 唯一取得路径未被 r70 改写破坏：`start` 的 `enter` goroutine 在 `bounded` 建立后、调用 `Enter` 前，经 `a.composition.ports.Secret.ReadInitialRecoverySecret(bounded, binding)` 取得候选，并以显式 `recovery.RecoveryInput{RunBinding: binding, Secret: candidate}` 传入；`binding = a.plan.config.command.binding`，无第二读取点、无别名、无新配置键、无 root secret 环境变量（`:4076`；`:1682` 同步）。
- 失败分类固定：`secretErr != nil` → `setFlowResult(secretErr)` + `requestStop()` 后返回，不调用 `Enter`、不发生状态变更；生产默认源返回 `ErrAdmissionClosed`（`recovery_runtime.go:631`–`:633`），空 ctx／binding 为 `ErrAdmissionBinding`，`bounded` 到期／取消原样上报、不包装、不降级。空候选不在 root 侧另判，仍由 `Enter:1522` 在 `beginAction`（`:1525`）前返回 `ErrAdmissionBinding`（唯一空值判定点）。
- 零化：`candidate` 与 `Enter` 内 `in.Secret` 为同一 buffer，`Enter` 的 `defer clear(in.Secret)`（`:1518`）负责零化，root 侧 `defer clear(candidate)` 作幂等防御（`:4076` 两处提及）；不得复制、缓存或重读。
- 可观测面：`:1481` 明写 `Secret []byte // root owns buffer; no String/JSON/log representation`；`RecoveryInput` 不进 `receipt`／`plan` wire（`:3882`／`:3884` 的 wire 为 `restorePlanWire`／`restoreReceiptWire` 与私有 stdin 对象，无 secret 字段）；无日志、无 stdout、无环境变量、无临时文件路径。
- r70 的 5 行变更未新增任何 secret 处理语句或日志点；本次改写只涉及 `waitFlow` 与 reaper 文本，**未引入新的凭据泄露面**。

### 3.6 无回退确认（r68／r69 已闭合安全项，锚点为 r70 快照原文）

| 项 | 锚点 | 现状（本轮读取） | 判定 |
|---|---|---|---|
| strict v1 合同 | `:3882`／`:3884` | 私有 stdin 恰为 `{"plan":…,"target_connection":…}`、plan 字段顺序固定、receipt 12 字段 strict JSON（无重复/额外/null） | 未回退 |
| 无 shell | `:3880`／`:3892`／`:4446` | root 不进入 shell、argv 不含 shell quoting、只允许固定 argv；真实 fd-exec 仍 NOT RUN | 未回退 |
| 私有 stdin | `:3882`／`:4420`／`:4070` | DSN 只经私有 stdin，`HAN_RECOVERY_RESTORE_TARGET_DSN_FILE` 单文件读取，禁止回退 `HAN_DATABASE_DSN` | 未回退 |
| 最小 ENV | `:4105`／`:4078` | tool golden vector 的 `environment` 逐项复算为 `LANG=C.UTF-8`、`LC_ALL=C.UTF-8`、`TZ=UTC` 三项；`PATH` 放宽抵消措施在 | 未回退 |
| Begin 前 closed | `:4037`／`:4062`／`:4076` | P6 在 `Journal.Begin` 前重读比对、漂移 `ErrAdmissionClosed`；空候选在 `beginAction` 前 `ErrAdmissionBinding` | 未回退 |
| 不得自证 | `:3884`／`:3888` | 不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证 | 未回退 |
| `injectedExclusive` 永不 Close | `:4068`／`:4082` | 只读 `injectedExclusive`、永不 `Close`；`Stop`／`drain` 只关 `ownedExclusive` | 未回退 |
| `Target.DSN` 删除与九处清零 | `:4066`／`:3971`／`:4045`／`:4420` | `Target.DSN` 字段删除（`rg Target\.DSN cmd` = 0）、DSN 九处清零与 redactor 在；stdin DSN 写后清零 | 未回退 |
| `SEC-F002`／`SEC-F006`／`N012`–`N016` | `:4082` | 明写「已闭合状态不回退」 | 未回退 |

## 4 golden vector 独立复算

方法：`[IO.File]::ReadAllBytes` 取快照，按 LF 建行首偏移表；第 n 行载荷 = `[start(n), start(n+1)-1)`（去掉行尾 LF）。两条向量行均不以 CR 结束（已核 `endsCR=False`），故载荷即行原文 UTF-8 字节；对该区间原始字节直接 SHA-256，不重新编码、不做换行归一。

| 向量 | 行 | 复算字节 | 复算 SHA-256 | 期望 | 一致性 |
|---|---|---|---|---|---|
| source | `:4099` | **1,115** | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 / `de914622…` | 一致 |
| tool | `:4105` | **2,357** | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 / `8c0c89…` | 一致 |

角色字面仍在六值封闭枚举内：`:4035` 定义 `restore_reviewer`／`restore_approver`／`restore_executor`／`restore_producer`／`restore_verifier`／`restore_witness`；source 向量的 `role_name` 复算为 `restore_approver,restore_reviewer`，tool 向量为 `restore_executor,restore_approver,restore_reviewer`，无枚举外值。

## 5 三段后缀独立复算与 r09 错值更正

方法（**raw bytes**）：`[IO.File]::ReadAllBytes`，扫描字节值 10 建 LF 行首偏移表；第 n 行起点 = 偏移表第 n-1 项；后缀 = `[start, EOF)` 原始字节，不重新编码、不做任何换行归一；字节数 = EOF − start，摘要 = 该区间原始字节 SHA-256。

| 行 | startByte | 后缀字节 | SHA-256 | 交叉校验 |
|---|---|---|---|---|
| `:4259` | 819,672 | **283,480** | `e8848838df83bc09bd8c94297219702cd86bd6629bd31fd8853f834f652cf0b6` | 1,103,152 − 819,672 = 283,480 |
| `:4275` | 821,952 | **281,200** | `2f53882f8774d30a51754872449ba29b4f257253c8a2f457ab6b9109d054fafb` | 1,103,152 − 821,952 = 281,200 |
| `:4290` | 824,937 | **278,215** | `b0af3b04c60584cf2618c83605e7fba88f65ca6b48f486463570268279e05f04` | 1,103,152 − 824,937 = 278,215 |

三项与派发预期逐位一致；与 r69／r68／r67 基线在各自 `:4259` 起点至 EOF 的复算逐字节相同（起点分别 816,394／809,731／801,402，差值 3,278 与本轮全文增量一致），支持 `:4072` 的条件式断言。

**r09 错值更正（必须记录）**：`review-sec-r09` 报告的 `283,444 B / 98d2a9a596960bf01f11edad434d74696c18b59f1b946b5469e68eea9ed9a807` **作废**，差 36 B，不再引用。原因经复算确认：该值来自 `ReadAllLines` + 以 `"\n"` 拼接 + 重新编码的路径，而后缀区间含 1,218 行、其中 35 行以 CRLF 结束，该方法丢弃 35 个 CR 字节，并因末尾 LF 不再作为分隔符而少 1 个 LF 字节，合计少 36 B（283,480 − 35 − 1 = 283,444）。本轮以 raw bytes 方法得 283,480 / `e8848838…`，与 `design-revision-r09/hashes.txt` 节 [5]（r69 时点值）及 r70 `hashes.txt` 节 [5] 一致。

## 6 `SEC-F007`、`N005` 与未验证边界

- `SEC-F007` 及全部 Linux 侧能力：**NOT RUN**。POSIX 宿主隔离、进程树 reap、专用执行身份实测、host／ACL／mount／loader／fd-exec／maps／network／witness 均未实跑；Windows 解析／语义层不等于 PASS；`//go:build linux` 的 skip 与无匹配测试 exit 0 不等于 PASS。本轮不因任何设计文字判 PASS。
- `N005`（范围外 BLOCKER，仅登记）：`decisions/framework.md`（285,639 B / `B432E44A…`）`:1084` 载明现行修订 `adr-0031-r10` 的两路接受尚未取得、「`identity-auth-S4.24a` 及任何实施/验证卡不得启动」；`tasks.md`（898,536 B / `4BD724A9…`）`:3197` 仍称 `adr-0031-r07` 为现行修订且门禁解除。两者相反，属架构决策方与 Task-Planner 范围；不据此判 CHANGES REQUIRED，不代改，不声称已解决。
- `tasks.md:3186` 实测仍为 `- [ ] identity-auth-S4.24a …`（未勾选）；`tasks.md` 哈希与派发值一致，未被我改动。

## 7 非阻断建议（不计入缺陷计数）

- `NEW-S10-01`（建议，非缺陷）：`:4076` 同时写「`enter`／`resume` 在起两条业务 goroutine 之前**各**执行恰一次 `flowWG.Add(2)`」与「`enter` 分支结构必须与 `resume` 同构」，而机核判据要求启动句「在 `start` 内恰 1 次」、`close(a.flowBothDone)`「全文恰 1 次」。若实现者把屏障启动内联进两个互斥分支，静态计数为 2；若抽成公共 helper，又不在 `start` 内。建议补一句明确「屏障启动（Add(2) + reaper）是分支之前的单一公共点，两条业务 goroutine 由分支分别启动」，或把判据写明「按每次执行动作计」。**无运行时安全后果**：两分支互斥，任一执行路径只有一个关闭点，不构成第二 closer，不改变第 3.1 节的闭合判定。
- O1（沿用 r09，非阻断）：`secretErr != nil` 分支未登记无条件 `defer clear(candidate)`；若某 provider 返回「非 nil 缓冲 + error」，该缓冲在错误路径不被零化。建议 provider 合同写明「error 时返回 nil」或无条件先登记 `defer`。
- O2（沿用 r09，非阻断）：provider 返回的 error 文本由 provider 控制，设计要求「原样上报」但未约束其不得回显 secret 字节；建议在 provider 合同注明。
- O4（沿用 r09，非阻断）：执行器不响应 ctx 时按设计持有租约至进程退出回收；属有意的 fail-closed 取舍，建议在部署／运维合同注明「teardown 后应结束进程」。

## 8 残余风险与复审条件

- 残余（非阻断）：见证与 root／executor／producer 独立性仅由组织登记与人工复核保证；同一自然人两个 `person_ref` 不可检测；DNS 投毒／同管理域冒充不因 TLS pin 消除；golden vector 仅两条；`bundle_digest` 仍显式 OPEN（首期不实现）。
- 后续任何对该对象的再修订，若触及 `:4259` 之后内容，必须重算三段后缀并登记取得时点；引用时必须同时给出「源文件 SHA-256 + 行号 + 取得时点」。
- 本判决只覆盖冻结 r70 快照；现场漂移时须按新哈希重判。

## 9 声明

本报告只完成对冻结 r70 的独立安全侧设计复审，不构成：生产解锁、`enter`／`resume` 解锁、约束 8 满足、`N005` 已解决、`identity-auth-S4.24a` 可关闭或已交付；`SEC-F007` 仍为 NOT RUN，不因本报告改判 PASS。任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。最终接受由 Lead／用户决定。