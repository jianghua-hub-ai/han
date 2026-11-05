# SEC r09 独立复审报告 —— identity-auth 详细设计 r69

- 日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：`.add\project\knowledge\design-workspace\identity-auth\design.md`
  - 冻结值：**1,099,874 B / 5,476 行 / SHA256 `41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09`**
  - 版本：`identity-auth-design-ir01-r69`（`:5`、`:38`）
  - 冻结快照：`C:\Users\J00010385\AppData\Local\Temp\design-r69-freeze-41D708D9.md`，字节/行/摘要与上值一致（已核）
- 基准报告：`.spec\wip\recovery-runtime\evidence-b13j-s424a\review-sec-r08\report.md`
  - 15,389 B / `BD78FD4D9A020A9CD988D98C13099142518ABF0E4A61D23FDE59A471B3EF63A3`（Verdict: CHANGES REQUIRED；开放 `NEW-S08-02` LOW）
- 基线对象：`design-revision-r09\design.r68.baseline.md`，1,093,211 B / 5,476 行 / `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083`（已核）

## 0 方法、检查范围与只读声明

方法：按 `add-reviewer` / `add-code-review`（objectType=design）对冻结 r69 做独立复审；先逐行比对 r68 基线与 r69 快照定位变更，再对 6 处变更行全文判读，并沿真实消费者（`cmd/han-server/recovery_runtime.go`、`recovery_sources.go`、`internal/recovery/runtime.go`）核对设计引用；两条 golden vector 与三段后缀均在冻结快照上自行复算。全部数值本轮自算，不采信作者报告。

检查范围（判读过的文件，均为只读）：
- `.add\project\knowledge\design-workspace\identity-auth\design.md`（冻结快照 `design-r69-freeze-41D708D9.md`）
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r09\design.r68.baseline.md`
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\review-sec-r08\report.md`
- `cmd/han-server/recovery_runtime.go`（组合构造 `:54`–`:76`、生产默认 secret 源 `:631`–`:640`、`start`/`completeOnRequest`/`flowOnce` 既有实现）
- `cmd/han-server/recovery_sources.go`（`recoverySecretSource.ReadInitialRecoverySecret` `:415`–`:420`）
- `internal/recovery/runtime.go`（`RecoveryInput.Secret` `:41`–`:43`、`Enter` `:1517`–`:1534`）

只读声明：本轮仅写入本文件；未改 `design.md`、`tasks.md`、ADR/`framework.md`、`cmd/**`、`internal/**`、测试、既有 `review-*` 与 `design-revision-*`；未删除/移动/归档；未执行任何 Git 写操作。冻结对象现场哈希与快照一致，无漂移。

## 1 结论

**Verdict: CHANGES REQUIRED**

| 类别 | 计数 | 说明 |
|---|---|---|
| BLOCKER | 0 | — |
| MEDIUM | 0 | — |
| LOW | 1 | `NEW-S09-01`（`:4076` reaper 数量/双重关闭前后矛盾，直接削弱本轮声称的 join 屏障） |
| NOT RUN（不计入缺陷计数） | 1 | `SEC-F007` 及 Linux 侧能力（宿主隔离/进程树 reap/专用执行身份实测） |
| 范围外 BLOCKER（仅登记） | 1 | `N005`（ADR 状态与 `tasks.md` 冲突，不据此判 CHANGES REQUIRED，不代改） |

开放项清单：`NEW-S09-01`（LOW）；`SEC-F007`（NOT RUN）；`N005`（范围外，仅登记）。r08 的 `NEW-S08-02` 本轮判定为**意图已实质补齐、但因 `NEW-S09-01` 尚未完全闭合**。

## 2 r69 变更与字节账（独立复算）

逐行比对 `design.r68.baseline.md` 与 r69 快照，变更恰 6 行、0 行净增减（均 5,476 行），UTF-8 字节差求和 = 6,663 = 1,099,874 − 1,093,211，与作者声称一致：

| 行 | 字节差 | 内容 |
|---|---|---|
| `:5` | 0 | `revision: identity-auth-design-ir01-r69` |
| `:38` | 0 | 设计版本升 r69 |
| `:1682` | +257 | `Enter` 端口的 secret 取得路径（NEW-R08-01） |
| `:4072` | +1,060 | r69 修订登记（r69.1/r69.2/r69.3）＋三段后缀断言改为条件式（NEW-R08-03） |
| `:4076` | +5,261 | join 屏障与 teardown 顺序、enter secret 唯一取得路径（NEW-R08-01/02、NEW-S08-02） |
| `:4083` | +85 | r69 沿用 r66.11 声明（不改 NOT RUN/unsupported/closed） |

## 3 逐条判定

### 3.1 NEW-S08-02（r08 开放项）＋ NEW-R08-02 双路 join 与租约释放顺序

判定：**意图已实质补齐；因 `NEW-S09-01` 未完全闭合**。

成立的部分（`:4076` 证据）：
- 双路 join 屏障：assembly 增 `flowWG sync.WaitGroup` 与 `flowBothDone chan struct{}`；`enter`／`resume` 在起两条业务 goroutine 前各执行一次 `flowWG.Add(2)`，两条业务 goroutine 函数体首行各 `defer a.flowWG.Done()`；`flowBothDone` 只在两条路径都从函数体退出后关闭，是唯一 join 信号。**双路（enter/resume）都具备该屏障**，满足「双路都存在真实 join 屏障」。
- teardown 顺序：`joinWork` 的 `case "enter"`／`case "resume"` 共用模板 `Stop(ctx)` → 等 `flowBothDone` → `waitFlow`（仅读结果）→ `closeRestoreLease()`；`flowFinished()` 在两 case 中不得作提前返回；机核判据要求同一 case 内按序出现 `Stop(`、`flowBothDone`、`waitFlow(`、`closeRestoreLease(`，且 `closeRestoreLease(` 不早于对 `flowBothDone` 的接收。**租约释放严格晚于两条路径退出**。
- 执行器不响应 ctx 的 fail-closed：屏障在 `joinWork` 的 `shutdownCtx` 到期/取消仍未通过时，明令**不得**调用 `closeRestoreLease()`（持有租约，宁可泄漏也不在使用中关闭），并按 `restore_unknown`／`ErrAdmissionIncomplete` 分类。这正确回应了 r08 `NEW-S08-02` 指出的「提前释放独占租约」风险。

未闭合的原因（见 `NEW-S09-01`）：同一 `:4076` 内对 reaper 的数量与放置存在**自相矛盾**的两种表述，其中一种（两条业务 goroutine「各起一次性 reaper」）会导致 `close(flowBothDone)` 被调用两次而 panic，直接破坏 join 屏障。r08 要求的「双路 join 屏障 + 租约释放顺序」两件事在文本上均已出现，但屏障的构造指令尚不可无歧义实现，故 `NEW-S08-02` 不判完全闭合。

### 3.2 NEW-R08-01（`enter` 的 secret 取得路径）与凭据泄露面

判定：**路径闭合，未引入可观测的新泄露面**（附 2 条非阻断残余）。

- 唯一取得路径：`start` 的 `enter` goroutine 在 `bounded` 内经 `a.composition.ports.Secret.ReadInitialRecoverySecret(bounded, binding)` 取得候选，并以显式 `recovery.RecoveryInput{RunBinding: binding, Secret: candidate}` 传入 `Enter`；`:1682` 同步。`binding` 同源 `a.plan.config.command.binding`，无二次读取、无别名、无配置键、无 root secret 环境变量。
- 引用准确（已核现役代码）：`newRecoveryRuntimeComposition` 对缺失/typed-nil `ports.Secret` 返回 `ErrAdmissionBinding`（`recovery_runtime.go:54`–`:58`）；`ReadInitialRecoverySecret` 对 nil 接收者/nil ctx/空 `RunID`/`BackupRef` 返回 `ErrAdmissionBinding`（`recovery_sources.go:415`–`:420`）；生产默认源返回 `ErrAdmissionClosed`（`recovery_runtime.go:631`–`:633`）；`Enter` 的 `defer clear(in.Secret)`（`runtime.go:1518`）、空值判定 `len(in.Secret)==0 → ErrAdmissionBinding`（`:1522`）、`beginAction`（`:1525`）、`f.enter(..., in.Secret)`（`:1530`）均存在。
- 泄露面核查：
  - 日志/错误串：设计不记录 secret；`secretErr` 仅来自上述已定义分类（binding/closed/ctx），不含 secret 值；`bounded` 到期/取消「原样上报、不包装、不降级」。
  - stdout：不经 stdout（receipt 明确排除 stdout 自证，`:3888`）。
  - 环境变量：显式禁止另设配置键/别名/root secret 环境变量。
  - 临时文件：secret 由 provider 读取，无落盘路径。
  - receipt/plan wire：`RecoveryInput.Secret` 带 `json:"-"`（`internal/recovery/runtime.go:43`），不参与任何 JSON 序列化；secret「只用于本 run Begin/信任根」，不进 receipt/plan。
- 非阻断残余 O1：`secretErr != nil` 分支未登记 `defer clear(candidate)`；生产默认源返回 nil 故当前无实害，但若某 provider 返回「非 nil 缓冲 + error」，该缓冲在错误路径不被零化。建议：无条件先登记 `defer clear(candidate)`，或把「error 时返回 nil」写入 provider 合同。
- 非阻断残余 O2：provider 返回的 error 文本由 provider 控制；设计要求「原样上报」，未约束其不得回显 secret。建议在 provider 合同注明错误串不得包含候选字节。

### 3.3 NEW-R08-03（`:4072` 三段后缀条件化）

判定：**成立**。`:4072` 已把后缀断言改为条件式（「只有当 `:4259`/`:4275`/`:4290` 各自的行首之后内容被改写时，对应后缀才会变化」），并声明 r68/r69 全部改写均在 `:4259` 之前、其三段后缀与 r67 逐字节相同、时点值只登记在各自 `hashes.txt`。独立复算：`design.r68.baseline.md` 与 r69 快照自 `:4259` 至 EOF 的原始字节均为 **283,444 B / `98d2a9a596960bf01f11edad434d74696c18b59f1b946b5469e68eea9ed9a807`**，逐字节相同；r69 六处变更行均 `< :4259`，故条件成立，未见易腐的固化数值。

## 4 新问题

### NEW-S09-01（LOW）：`:4076` reaper 数量与双重关闭自相矛盾

- 位置：`design.md:4076`（r69 新增「双路 join 屏障」段与 `start` 段）。
- 矛盾：`start` 段写「先 `a.flowWG.Add(2)`、再起**一次性 reaper** `go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()`、最后起两条业务 goroutine」——即 `start` 起 **1 个** reaper。但「双路 join 屏障」段写「两条 goroutine 在函数体首行各自 `defer a.flowWG.Done()`，**并各起一次性 reaper** `go func(){ a.flowWG.Wait(); close(a.flowBothDone) }()`」——若「各」指两条 goroutine，则为 **2 个** reaper，二者都会执行 `close(a.flowBothDone)`，Go 中对同一 channel 二次 `close` 会 panic。同段机核判据又要求「`close(a.flowBothDone)` **恰 1 次**」，与前句直接冲突。
- 影响：若实现者按「各起」字面实现（两条 goroutine 各起一个 reaper），teardown 收敛时 `close(flowBothDone)` 二次关闭 panic，恰好破坏本轮新增的 join 屏障，使 `closeRestoreLease` 的严格顺序保证落空；该 panic 发生在恢复 teardown 路径。
- 修复建议：明确「仅由 `start` 起 1 个 reaper」（删除「两条 goroutine…各起」表述），或对 `close(a.flowBothDone)` 加 `sync.Once`/专用 `flowBothOnce`；并保持机核判据「恰 1 次」。
- 定级：LOW（机核判据与 `start` 段已给出正确读法，正确实现可得；但规范句存在可导致 panic 的歧义，若按「各起」实现则应按 MEDIUM 处理）。

## 5 无回退确认（r68 已闭合安全项，锚点为 r69 快照原文）

| 项 | 锚点 | 现状 | 判定 |
|---|---|---|---|
| strict v1 合同 | `:3882`/`:3884` | 私有 stdin 形状、v1 plan 字段顺序、strict JSON receipt（无重复/额外/null）在 | 未回退 |
| 无 shell | `:3880`/`:3892`/`:4446` | root 不进入 shell、argv 不含 shell quoting、只允许固定 argv；(d) 项真实 fd-exec NOT RUN | 未回退 |
| 私有 stdin | `:3882`/`:4420`/`:4070` | DSN 仅经私有 stdin、`HAN_RECOVERY_RESTORE_TARGET_DSN_FILE` 读取、无回退业务 DSN | 未回退 |
| 最小 ENV | `:4105`/`:4078` | golden tool ENV 仅 `LANG/LC_ALL/TZ`；`PATH` 放宽抵消措施在 | 未回退 |
| Begin 前 closed | `:4037`/`:4062`/`:4076` | P6 在 `Journal.Begin` 前重读比对、漂移 `ErrAdmissionClosed`；`enter` 空候选在 `beginAction` 前 `ErrAdmissionBinding` | 未回退 |
| 不得自证 | `:3884`/`:3888` | 不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证 | 未回退 |

## 6 golden vector 独立复算

在 r69 冻结快照上对 canonical 行取 `UTF8.GetBytes(line)`（不含围栏/换行）计字节与 SHA-256：

| 向量 | 行 | 复算字节 | 复算 SHA-256 | 一致性 |
|---|---|---|---|---|
| source | `:4099` | **1,115** | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 与期望一致 |
| tool | `:4105` | **2,357** | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 与期望一致 |

（r69 六处变更行均不在 `:4099`/`:4105`，两条向量字节未受影响。）

## 7 SEC-F007、N005 与未验证边界

- `SEC-F007` 及 Linux 侧能力：**一律 NOT RUN**。POSIX 宿主隔离、进程树 reap、专用执行身份实测、host/ACL/mount/loader/fd-exec/maps/network/witness 均未实跑；Windows 解析/语义层不等于 PASS；`//go:build linux` 的 skip 与无匹配测试 exit 0 不等于 PASS。本轮不因任何设计文字判 PASS。
- `N005`（范围外阻断，仅登记）：ADR 状态段与 `tasks.md` 冲突属架构决策方与 Task-Planner 范围；不据此判 CHANGES REQUIRED，不代改。
- 残余（非阻断）：执行器不响应 ctx 时按设计**持有租约至进程退出回收**，若进程在 teardown 后继续运行，后续恢复可能因租约未释放而受阻；这是有意的 fail-closed 取舍，建议在部署/运维合同注明「teardown 后应结束进程」。

## 8 复审条件

闭合 `NEW-S09-01` 后本对象可再判：① `:4076` 明确「仅 `start` 起 1 个 reaper」或对 `close(flowBothDone)` 加 `sync.Once`，并使机核判据「恰 1 次」无歧义；② 保留第 3.1 节的双路屏障与 `Stop → flowBothDone → waitFlow → closeRestoreLease` 顺序及执行器不响应时的持租约 fail-closed；③ 保留第 5 节无回退证据与两条 golden vector；④ 不改动 `SEC-F007` 与 `N005` 边界。

## 9 声明

本报告只完成对冻结 r69 的独立安全侧设计复审，不构成：生产解锁、`enter`/`resume` 解锁、约束 8 满足、`N005` 已解决、`identity-auth-S4.24a` 可关闭或已交付；`SEC-F007` 仍为 NOT RUN，不因本报告改判 PASS。任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。最终接受由 Lead/用户决定。