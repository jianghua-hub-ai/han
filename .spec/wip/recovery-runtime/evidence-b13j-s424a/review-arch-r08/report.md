# ARCH r08 独立架构复审报告 — identity-auth 详细设计 r68

- 日期：2026-11-02
- 角色：架构侧 Reviewer（`add-reviewer`；主方法 `add-code-review`，objectType=design）
- 会话性质：独立复审，无历史会话；本报告全部数值由本会话自行复算，作者报告只作线索
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 方法依据：`.codex/agents/add-reviewer.toml`、`.codex/skills/add-code-review/SKILL.md`（design 分支）、`code-review-checklist`、`abstraction-principles`、`design-doc-writing-style`、`decisions-guide`
- 前序线索（只作线索，不作接受依据）：`review-arch-r05/report.md`、`review-arch-r06/report.md`、`design-revision-r06/report.md`、`design-revision-r07/report.md`、`design-revision-r08/report.md` 与 `design-revision-r08/hashes.txt`

**Verdict: CHANGES REQUIRED**（开放 3 项：MEDIUM 2、LOW 1；阻断 0）

---

## 1. 冻结对象与基线

### 1.1 评审对象（冻结）

| 项 | 值 |
|---|---|
| 路径 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| 字节 | 1,093,211 B |
| 行数 | 5,476 |
| 换行 | CR=169、LF=5,476（混合换行，两版一致） |
| SHA256 | `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083` |
| 版本 | `identity-auth-design-ir01-r68`（`:5`、`:38`，两处一致，已核） |
| 冻结快照 | `%TEMP%\design-r68-freeze-5E324AF9.md`（自建；SHA256 与对象相同，字节一致） |

### 1.2 基准报告

| 项 | 字节 | 行数 | SHA256 |
|---|---:|---:|---|
| `review-arch-r07/report.md` | 35,228 | 325 | `3F27DAF27EA558CB83E733CB6F581A55CCBB69EFD151705710FFDDCC580824B5` |

### 1.3 作者侧线索（未采信为结论）

| 项 | 实测 | 说明 |
|---|---|---|
| `design-revision-r08/report.md` | 17,427 B／160 行／`5C0970F8B2B149141BAFA319FCC074A87F52A0A6C496D9197B42AFBA4130DB42` | 与派发值一致 |
| `design-revision-r08/hashes.txt` | 3,459 B／65 行／`4770136A02F3C706AB60170E18CB0E26CE7252A2E39175CFD0F85380A9CE9757` | 与派发值一致 |
| `design-revision-r08/design.r67.baseline.md` | 1,084,882 B／5,476 行／`B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E`（CR=169／LF=5,476） | 与派发 r67 哈希逐字节一致，判定为真实 r66→r67 终值，不是中间态 |

### 1.4 只读声明

本轮唯一写入为本报告及同目录 `hashes.txt`。未修改 `design.md`、`tasks.md`、ADR（`framework.md`）、`cmd/**`、`internal/**`、测试、任何既有 `review-*/` 与 `design-revision-*/`；未删除、未移动、未归档；未执行任何 Git 写操作（无 add／commit／checkout／stash／branch）。

本轮只完成设计文本评审，**不构成**评审通过、生产 restore 解锁、`enter`／`resume` 解锁、约束 8 满足或 `identity-auth-S4.24a` 可关闭的判断；任何 NOT RUN 仍为 NOT RUN。

---

## 2. 检查范围

### 2.1 第一轮（对象级字面，两遍）

- 版本与声明：`:5`、`:38`、`:4070`（r65.13 残余与声明）、`:4083`（r66.11 声明 + r68 沿用句）。
- 本轮变更行：`:4070`、`:4072`、`:4076`、`:4077`、`:4079`、`:4083`、`:4195`（逐行与 r67 基线比对，另有 `:5`／`:38` 两处仅版本号替换）。
- 关键引用点：`:3603`（production root 装配清单）、`:3942`、`:3990`、`:4037`、`:4062`、`:4074`（r66.2）、`:4135`、`:4137`、`:1682`、`:4257`、`:4259`、`:4290`、`:4369`、`:4373`。
- golden vector 两行：`:4099`、`:4105`。

### 2.2 第二轮（真实消费者与上游依据，有界）

- `internal/recovery/runtime.go`：`:293`／`:404`／`:1123`／`:1160`–`:1177`／`:1196`／`:1245`／`:1497`–`:1514`／`:1517`–`:1530`／`:1539`–`:1542`／`:1554`–`:1570`／`:1854`–`:1858`／`:1943`、`:768`（字段）、`:892`、`:917`–`:921`。
- `internal/recovery/admission.go`：`:401`–`:412`（`checkExclusive`）、`:616`–`:650`（`readControlled` 与 `:648`）。
- `internal/recovery/coordinator.go`：`:96`。
- `cmd/han-server/recovery_runtime.go`：`:517`–`:519`、`:840`–`:850`、`:872`–`:896`、`:901`–`:921`、`:932`–`:964`。
- `cmd/han-server/recovery_sources.go`：`:165`–`:183`、`:314`–`:357`。
- `cmd/han-server/main.go`：`:770`–`:781`。
- `cmd/han-server/recovery_config.go`：`:58`–`:98`、`:222`–`:241`。
- `tasks.md`：`:3186`；ADR：`framework.md` 仅核版本行 `:1082` 与 `NewRuntime` 出现次数（0 次），未评审 ADR 内容、未改。

---

## 3. r07 开放项逐条判定

### 3.1 `F006`（BLOCKER，结转两轮）(a) — **CLOSED**

- `:4076` 第(3)点已固定：`activate` 在 `:918` 已持 `r.mu`，`:919` 固定为 `p, lease, generation := r.proof, r.activeExclusiveLocked(), r.generation`，并明写「禁止在 `:919` 调用 `activeExclusive()`（Go `sync.Mutex` 不可重入，否则自锁）」。
- 调用点点名与实测一致：`rg -n 'r\.activate\(' internal/recovery/runtime.go` 命中 `:892`（`RunNormal`）与 `:1943`（`replay`），正是设计点名的两处；无第三处。
- 机核判据可机核：`activate` 函数体内 `activeExclusive(` = 0、`activeExclusiveLocked(` = 1；`:4076` 全文出现 `activeExclusiveLocked(` 10 次、裸 `activeExclusive(` 7 次，全部位于「禁止／机核判据」语境（如「`activeExclusive(` 出现 0 次」），不存在把自取锁变体写成规定动作的残留。
- 现状代码 `:919` 是 `r.exclusive` 直读（`runtime.go:919`），改法不影响既有语义，且与 `:1162`（`checkOwnership`，已持锁）同一模式。
- teardown 覆盖：第(8)点 `Stop`／`drain` 只 `Close` `ownedExclusive`，绝不 `Close` `injectedExclusive`；第(10)点两处 owned 写入点保持原样。

判定：**闭合**，可按字面实施，无自锁。

### 3.2 `F006` (b) — **CLOSED**

- 新增 `ownershipLease(ctx) (ExclusiveLease, error)` 六步与 `checkOwnership`（`runtime.go:1161`–`:1176`）逐步对应：①一次持锁取 `activeExclusiveLocked()`／`generationCtx`／`action`（对应 `:1161`–`:1164`）；②`generationCtx == nil` → `ErrAdmissionClosed`（对应 `:1165`–`:1167`）；③`generationCtx.Err()` → `r.fail` 后返回原错误（对应 `:1168`–`:1171`）；④nil／typed-nil 用 `missingRuntimePort` 判于 `checkExclusive` **之前**：injected → `ErrAdmissionClosed`、owned → `ErrAdmissionBinding`（与 `checkExclusive` 的 `:405`–`:406` 同值，判定提前）；⑤`checkExclusive` 其他失败 `r.fail` 后返回（对应 `:1172`–`:1175`）；⑥返回同一 `lease`。
- 值确实贯穿到 `:648`：`readControlled`（`admission.go:616`）入口一次取得 `lease`，`:648` 由 `recheckProof(ctx, r.deps, r.exclusive, v.proof)` 改为 `recheckProof(ctx, r.deps, lease, v.proof)`；`:617` 的 `r.checkOwnership(ctx)` 删除。改后 `readControlled` 内 `r.exclusive` 直读 = 0、`checkOwnership(` = 0。
- 6 个调用者签名不变：实测改前 `checkOwnership(` 调用点为 `runtime.go:293`／`:404`／`:1123`／`:1196`／`:1245`、`coordinator.go:96`、`admission.go:617`（共 7 处）；删除 `:617` 后恰为设计所列 6 处，签名保持 `func (r *admissionRuntime) checkOwnership(ctx context.Context) error` 薄包装，行为不变。
- `ctx` 已取消分支差异已被第⑤步覆盖：`checkExclusive` 首行 `checkedContext(ctx)` 返回的错误会经第⑤步 `r.fail` + 返回原错误，与现状 `:1172`–`:1174` 等价。
- nil 窗口：`enter` 缺失 lease 的唯一可能是装配未注入，`recoveryPreflight`（`:1497`–`:1514`）在 `:1513` `return nil` 之前判 `injectedExclusive`（含 typed nil），该点严格早于 `beginAction`（`Enter` `:1525`、`Resume` `:1542`）与 `recoveryFlow.acquire` 的 `StopAndJoin`（`:1556`）；实测 `Enter` 在 `:1519` 调 preflight、`:1525` 才 beginAction，顺序成立。`recoveryFlow.acquire` 删除 `Acquire`／`f.r.exclusive = l`（`:1562`／`:1564`）后不再承担 nil 判定。
- 结论：6 个调用者不用改，`admission.go:648` 取到同一 `lease`，不存在「取一次后又被重取」的窗口。

判定：**闭合**，无自锁、无 nil 窗口。

### 3.3 `NEW-R07-02`（BLOCKER）— 死锁机制 **CLOSED**；但引入两处新缺陷

- `enter` 已改为可 join 的 goroutine：`:4076` 规定 `start` 增 `enter` 分支，`bounded, cancel := context.WithTimeout(ctx, budget)` 后起两条 goroutine（`Enter` 与 `completeOnRequest`），随后**立即 `return nil`**，使 `main.go:770` 返回后 `:781` 的 control server 得以启动。实测 `start`（`recovery_runtime.go:872`–`:896`）现有 `resume` 分支正是该形态（两条 goroutine + `return nil`），且 `main.go:770`→`:773`–`:777` 的 `awaitControlReady` 与 `:781` 的 `coordinate` 顺序证实确认面在 `start` 返回后才起。
- `completeOnRequest(ctx context.Context)` 定义在 `recovery_runtime.go:901`–`:914`，签名与设计一致；调用点恰为 `start` 的 `resume`（`:891`）与新增 `enter` 分支各一次；identity 完成时点由 `completeIdentity` 关闭 `completeDone` 决定（`recovery_sources.go:356`），`CompleteIdentityReview`（`:314`–`:327`）只阻塞在 `completeDone` 或 `ctx.Done()`。
- `bounded` 边界闭合：同一 `budget` ctx 传入 `waitCompleteRequest`、`CompleteIdentityReview`、`waitFlow`，超时／取消分别经 `ctx.Done()` 返回可诊断错误，随后 `requestStop()`（`:840`–`:850`，取 `a.stop` 后调用）取消 generation，交 `joinWork` 收敛；不存在无界等待，也不存在「`start` 内同步调用 `Enter` 导致确认面未起而死锁」的原始缺陷。

判定：原始死锁缺陷**闭合**。但同一处改写引入 `NEW-R08-01` 与 `NEW-R08-02`（见 §4），后者使「两侧都被 join」的收尾承诺不成立。

### 3.4 `NEW-R07-01`／`NEW-R07-03`（MEDIUM）— **CLOSED**

与 §3.1／§3.2 同源：自锁改为 `Locked` 变体、nil 判定提前到 `checkExclusive` 之前且在任何状态变更之前，均已随 `:4076` 第(1)(2)(3)(7)点闭合；`:4076` 第(2)点的机核判据（`readControlled` 内 `r.exclusive` 直读 = 0、`checkOwnership(` = 0）与第(3)点（`activate` 内 `activeExclusive(` = 0、`activeExclusiveLocked(` = 1）都可直接机核。

### 3.5 `NEW-R05-03`（LOW）— **CLOSED**

`:4079` 已把范围限定为「唯一有效的**载荷绑定** V 编号是 `:4062` 的 V7–V11」，并明写「`V1`–`V6` 仍是有效的前置检查点……完整集合以 `:4037` 的 V1–V11 与 `:4062` 为准，本条不缩减任何检查点」。与 `:4037`（P1–P3／R1–R8／V1–V10 完整替换，V 编号唯一）、`:4062`（V7–V11 载荷绑定）无措辞冲突；`:3942`／`:3990` 的旧步骤名与「以 r64.5／r64.2 为准」一律标为历史、不参与验证。r06b 提出的 `:4079` 与 `:4037`／`:4062` 的 V1–V11 措辞冲突已由「载荷绑定」限定语消解，表述唯一可判。

### 3.6 `NEW-R07-04`（LOW）— 文本层 **CLOSED**（残余 LOW 见 `NEW-R08-03`）

`:4072` 已删除易腐数值的「现行」地位，改为可复算规则：起点取该行行首在当次冻结副本中的字节偏移、终点 EOF、不重新编码、不做换行归一，字节数 = 终点 − 起点，摘要 = 该区间原始字节 SHA-256；并要求「把结果与取得时点写入该次修订的 `hashes.txt`；引用时必须同时给出源文件 SHA-256 + 行号 + 取得时点」。r66 值已明确标为「r66 时点值（r67 前）只作历史」。文本层闭合。

### 3.7 无回退确认

| 项 | 判定 | 证据 |
|---|---|---|
| `F004` 文本层 | 保持 | `:3603` 仍为唯一 production root 装配清单并绑定 `recoveryRootInputs` 与 `NewRuntime` 依赖集合；`:4073` 仍为 r66.1 的 `restoreLease` 非新端口、11 项端口集合不变、唯一签名句 |
| ADR 同步 | 仍为升级待决、不阻断 | `framework.md` 285,639 B／`B432E44A…`／`:1082` `adr-0031-r10`；全文 `NewRuntime` 命中 **0**。ADR 由并行会话改写，本轮不改、不作为停止条件 |
| `NEW-R05-04` | 保持 | `:4076` driver 段：`RestoreDriver` 三方法、全仓禁止新增任何 `Bind`（含 `driver.` 前缀）、`newControlledRestoreDriver` 第五参数与构造期写入、生产只经构造器 |
| `NEW-R05-05` 行号部分 | 准确 | `:4257` =「**r62.9.1 完整文件表**」，`:4259` = 表头 `| 文件 | build tag | 变更 | 职责 |`，`:4290` =「**r61 生产 restore 接线的实现形态（B13J／S4.24a）**」，与 `:4072` 的更正一致 |
| `NEW-R06-02` 字节账 | 自洽、r65 列标「重建」 | `design-revision-r07/report.md` 16 行表（`:100`–`:118`），列名 `r65 B（重建）`；合计 `+412`／`+3,307`，说明行给出 5 行补充 `+59`；与本次独立复算的 r65→r67 账一致 |
| `NEW-R06-05`／`NEW-R06-06` | 保持 | `:4076` 测试面：先 `f.world = newS414World(p)`、再 `newControlledRestoreDriver(..., f.world.lease)`，删除字段赋值；`restore` 改工厂后两处测试字面量与 nil 工厂返回 `recoveryClosedRestoreDriver` 均已写明 |
| `NEW-R06-07` | 保持 | `:3990` 明写「现行 V1–V11……只以 r65.5／r65.2 及 r66.2 为准（r67.2 再确认……）」；`:3942` 实测含 `r66.2`×2、`r67.2`×1、`:4062`×1、`V1–V11`×1，与 `:4079` 不矛盾 |
| `NEW-R06-08` | 保持 | `design-revision-r08/report.md`：`##` 9 个、`###` 8 个，无重复标题；全文无 `1,077,602`／`0BA1`／`4AB7CFC6` 等旧值残留 |

### 3.8 `F017`（条件项）

`tasks.md` 898,536 B／3,289 行／`4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`，`:3186` 仍为 `- [ ] identity-auth-S4.24a 接入生产 restore source/target/tool 与批准主体接线……`。未被擅改，不计阻断。

---

## 4. 新 Findings

### `NEW-R08-01`（MEDIUM）— `enter` 字面不传 `Secret`，`Enter` 在任何动作前即 `ErrAdmissionBinding`；secret 来源在设计内无定义

- 位置：`design.md:4076`（`start` 的 `enter` 分支字面）、`design.md:1682`、`design.md:4195`。
- 违反依据：`internal/recovery/runtime.go:1517`–`:1530`——`Enter` 在 `:1522` `if len(in.Secret) == 0 { … ErrAdmissionBinding }` 早退，早于 `:1525` `beginAction`；`prepareTrust` 的惰性读取在 `:1854`–`:1858`（`if secret == nil` → `deps.RecoverySecret.ReadInitialRecoverySecret`），只有经 `:1530` `f.enter(r.generationCtx, in.Secret)` 进入流程后才可达。
- 触发场景：按设计字面接线 `a.setFlowResult(a.runtime.Enter(bounded, recovery.RecoveryInput{RunBinding: a.plan.config.command.binding}))`（`RecoveryInput` 只有 `RunBinding`，无 `Secret`）。无论 `deps.RecoverySecret` 是否装配，`Enter` 都在 `beginAction` 前返回 `ErrAdmissionBinding`；所谓「`Secret` 仍走既有候选通路」在代码中不存在可提供 secret 的调用点。另：`:1682` 的 `Enter` 行要求「`RecoveryInput` 的精确 `RunBinding` 与**本次 secret**」，但 `recoveryRootConfigurationKeys`（`cmd/han-server/recovery_config.go:58`–`:83`）没有任何 root 侧 recovery secret 键，`:4195` 新增的四个键也不含 secret，设计未定义 `enter` 的 secret 取得点。
- 影响：结果仍是 fail-closed（不构成解锁或安全放宽），但设计与实现字面不一致，按字面实施时 `Enter` 的 secret 分支不可达，「候选通路」无法被实现者定位；若后续按 `:1682` 补 root 键，会改变配置面而缺少对应设计条文与 `recoveryRootConfigurationKeys` 变更判据。
- 修复建议：二选一并写死——(i) `enter` goroutine 先从 `deps.RecoverySecret` 读取候选并显式传入 `RecoveryInput.Secret`（nil／空仍 closed），或 (ii) 删除 `runtime.go:1522` 的早退、把 nil 判定下移到 `prepareTrust`，使候选通路真实可达；同时修订 `:1682` 或注明 `enter` 的 secret 来源未实现、`RecoveryInput.Secret` 必为空。
- 证据：`runtime.go:1517`／`:1522`／`:1525`／`:1530`／`:1854`–`:1858`；`design.md:1682`／`:4076`／`:4195`；`recovery_config.go:58`–`:83`。

### `NEW-R08-02`（MEDIUM）— 「`Enter` 与 `completeOnRequest` 两侧都被 join」不成立；`flowDone` 先关使另一 goroutine 失去 join，`closeRestoreLease` 可与在飞 lease 使用并发

- 位置：`design.md:4076`（`joinWork` 段与 r68 括注「因此 `Enter` 与 `completeOnRequest` 两侧都被 join」）。
- 违反依据：`cmd/han-server/recovery_runtime.go:916`–`:921` `setFlowResult` 用 `sync.Once`，**第一个**完成者关闭 `flowDone`；`:932`–`:939` `waitFlow` 在 `flowDone` 关闭后立即返回；`:942`–`:964` `joinWork` 在 `flowFinished()` 为真时直接 `return a.waitFlow(ctx)`（**不调用 `Stop`**）。因此两条 goroutine 中任一先 `setFlowResult`，`joinWork` 的 `waitFlow` 立即返回，另一条仍在运行（仅受 `bounded`／`ioBudget` 约束），随后设计新增的 `closeRestoreLease()` 关闭 `restoreLease`。
- 触发场景（两条具体序列）：
  (a) `completeOnRequest` 在 `waitCompleteRequest`（`recovery_sources.go:165`–`:183`）或 `completeIdentity` 失败（含 `bounded` 超时／取消）→ `setFlowResult(err)` 关闭 `flowDone` → `joinWork` 立即返回并 `closeRestoreLease()`，而此时 `Enter` 可能仍在 `ownershipLease`／`activate`／`recheckProof` 中读取 `injectedExclusive` → 「lease 已关闭、仍被使用」的窗口，最终是 `l.Check` 的 closed 包装还是其他错误取决于时序。
  (b) `Enter` 先返回（按 `NEW-R08-01`，几乎必然先返回 `ErrAdmissionBinding`）→ `flowDone` 关闭 → `joinWork` 无 `Stop` 直接返回 → `completeOnRequest` 仍阻塞在 `waitCompleteRequest` 直到 `bounded` 到期；该 goroutine 与 `requestStop()` 都不在本轮收敛。
- 影响：不是死锁（有界 ctx 兜底），但「`joinWork` never leaves a flow, worker or call behind」与「关闭 lease 恰好一次」不能同时由 `sync.Once` + `waitFlow` 保证；teardown 的 use-after-close 窗口未在设计内排除，且可机核判据（「同一 case 内按序出现 `Stop`、`waitFlow`、`closeRestoreLease`」）会被 `flowFinished()` 短路绕过。
- 修复建议：把「结果收敛」与「join」分开——为两条 goroutine 各设独立 done（或 `sync.WaitGroup`），`joinWork` 的 `enter`／`resume` 分支先 `Stop`，再分别等待两个 done（`waitFlow` 只承担结果读取），最后 `closeRestoreLease()`；或至少规定 `flowFinished()` 为真时也必须 `Stop` 并等待第二个 done 后再关闭 lease。删除或改写「两侧都被 join」为上述可机核判据。
- 证据：`recovery_runtime.go:901`–`:921`、`:932`–`:964`；`recovery_sources.go:165`–`:183`、`:332`–`:357`；`design.md:4076`。

### `NEW-R08-03`（LOW）— r68 后缀复算账与 `:4072` 自身规则不一致

- 位置：`design.md:4072`；`design-revision-r08/hashes.txt`。
- 事实：`:4072` 规定「每次修订只对当次冻结副本复算，并把结果与取得时点写入该次修订的 `hashes.txt`」，但 r68 的 `hashes.txt`（65 行）没有三段后缀值、也没有取得时点；同时 `:4072` 称「r68 起还会随行的改写继续变化」，而按同一规则（行首字节偏移→EOF，raw bytes，不归一）复算，r68 与 r67 的三段后缀**完全相同**：`:4259` 283,480 B／`e8848838df83bc09bd8c94297219702cd86bd6629bd31fd8853f834f652cf0b6`、`:4275` 281,200 B／`2f53882f8774d30a51754872449ba29b4f257253c8a2f457ab6b9109d054fafb`、`:4290` 278,215 B／`b0af3b04c60584cf2618c83605e7fba88f65ca6b48f486463570268279e05f04`。原因是 r68 的 9 处改写全部位于 `:4259` 之前，不影响「该行行首→EOF」区间内容。
- 影响：不改变任何判定；但「继续变化」的断言与实测相反，读者会预期 r68 已有新值从而认为 `hashes.txt` 缺记；按「未引用即不需记」也可自洽，条文存在两种读法。
- 修复建议：`hashes.txt` 增加「后缀复算（r68 时点）」一节记录三个值与取得时点，或明写「r68 未改写 `:4259` 之后的行，后缀与 r67 相同，故不重复登记、以 r67 时点值为准」；把 `:4072` 的断言改为条件式（「当 `:4259` 之后的行被改写时才变化」）。
- 证据：`design.md:4072`；`design-revision-r08/hashes.txt`（全文 65 行无后缀值）；本报告 §5.4 复算表。

---

## 5. 独立复算证据

### 5.1 r67→r68 逐行字节账（9 行，0 行增减）

取法：`[IO.File]::ReadAllLines(path, UTF8)[n-1]` → `UTF8.GetBytes(line).Length`，两版逐行 `-cne` 比较。

| 行 | r67 B | r68 B | Δ |
|---|---:|---:|---:|
| `:5` | 39 | 39 | 0 |
| `:38` | 50 | 50 | 0 |
| `:4070` | 4,498 | 5,207 | +709 |
| `:4072` | 4,280 | 5,746 | +1,466 |
| `:4076` | 10,088 | 14,712 | +4,624 |
| `:4077` | 3,877 | 4,700 | +823 |
| `:4079` | 762 | 986 | +224 |
| `:4083` | 341 | 426 | +85 |
| `:4195` | 1,728 | 2,126 | +398 |
| **合计** | | | **+8,329** |

文件级：1,093,211 − 1,084,882 = **+8,329**，行数 5,476 → 5,476（0 增减），CR 169 → 169、LF 5,476 → 5,476。除上述 9 行外无差异，与作者 9 行表逐项一致。

### 5.2 golden vector（两条，独立复算）

对 `:4099`／`:4105` 行原文取 `UTF8.GetBytes`（无尾随换行）后 SHA256：

| vector | 行 | 字节 | 实测 SHA256 | 期望 | 判定 |
|---|---|---:|---|---|---|
| source | `:4099` | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 B／`de914622…` | PASS |
| tool | `:4105` | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 B／`8c0c89e3…` | PASS |

附注：带尾随 LF 时分别为 1,116 B／`8debd528…` 与 2,358 B／`abb4373c…`，与设计「无尾随换行」口径不符，故采用无 LF 值。

### 5.3 括号平衡与中间态

- 全角括号（正文嵌套）：全文件 1,173／1,173；`:4076` 82／82（r67 同行为 65／65）。
- 半角括号在行内代码之外（正文语义括号）：全文件 281／281；`:4076` 代码之外 1／1。行内代码内 413／406，差额来自 `activeExclusive(`、`r\.activate\(`、`checkOwnership(` 等有意不配对的字面片段，不是正文失衡。
- 中间态：write#1 `A95470BA…`（1,093,138 B）与 write#2 `43BFF315…`（1,093,208 B）在 `design.md` 中出现 **0** 次；终值 1,093,211 B／`5E324AF9…` 与 `:5`／`:38` 一致。判定：中间失衡态未被引用为基线，终值括号平衡。

### 5.4 三段后缀 raw 复算（`:4072` 规则）

取法：行首字节偏移→EOF，raw bytes，不归一。

| 起点 | r68 | r67 | 判定 |
|---|---|---|---|
| `:4259` | 283,480 B／`e8848838…cf0b6` | 同 | 未变化 |
| `:4275` | 281,200 B／`2f53882f…54fafb` | 同 | 未变化 |
| `:4290` | 278,215 B／`b0af3b04…e05f04` | 同 | 未变化 |

---

## 6. 观察项（OBS，不计入开放项／阻断计数）

1. **OBS-1**：`:4076` 第(7)点引用 `recoveryPreflight` `:1497`–`:1514`，实际 `return nil` 在 `:1513`（函数体止于 `:1514` 的 `}`）；范围多收一行，不影响「nil 判定在任何状态变更之前」的结论。
2. **OBS-2**：作者 r08 报告的长名计数（「`PATH` 3／`SHA` 4」）按行计；实测出现次数为 longPATH 4（`:4070`、`:4088`、`:4195`×2）、longSHA 5（另加 `:4373`）。属计数口径差异，不改变结论。
3. **OBS-3**：`:4195`「短名出现即 `ErrAdmissionBinding`」在代码中没有专门拒绝点——`recoveryRootConfigurationKeys` 只用于存在性判定（`recovery_config.go:88`–`:98`），`readAction`（`:222`–`:241`）也不做短名检查。`enter`／`resume` 仅设短名时恰好经 `readAction` → `recoveryConfigError`（= `ErrAdmissionBinding`）失败，其他动作无对应拒绝路径；若设计要求全动作拒绝，需写明判定点。
4. **OBS-4**：工作树 `git status` 有 298 条（含 `design.md`、`framework.md` 已修改），但 `design.md` 现哈希仍等于冻结值 `5E324AF9…`，无漂移影响；本轮按冻结快照判读。
5. **OBS-5**：派发称 `review-arch-r06/report.md` 为 24,561 B／`4AB7CFC6…`，盘上实测为 28,491 B／248 行／`C66F9BBE…`（自述哈希漂移）。该报告只作线索，未作接受依据。

---

## 7. 与并行 `review-arch-r06` 关键结论的一致性

- **同意**其方向：`F006` 自锁与 nil 检查落点（`NEW-R06-03`／`NEW-R06-04`）、`NEW-R06-02` 字节配平与 r65 列「重建」标注、`NEW-R06-05` 的 `f.world.lease` 初始化顺序、`NEW-R06-06` 的 `restore` 工厂化测试字面量清单、`NEW-R06-07` 的 `:3942`／`:3990` 权威指向、`NEW-R06-08` 的报告重复与旧值残留。这些在 r67／r68 中均已闭合，与本次独立复算一致。
- **不引用其为接受依据**：其盘上字节／哈希与派发声明不一致（OBS-5），且本轮每条结论均由本会话重新取证。
- 分歧仅在其未覆盖 r68 新增的 `enter` 字面问题与 join 收尾问题（`NEW-R08-01`／`NEW-R08-02`），二者是 r68 新增文本引入，不是对 r06 结论的否定。

---

## 8. NOT RUN

- `go build`／`go vet`／`go test`／`-race`：**NOT RUN**（本轮为设计文本评审；设计描述的 `ownershipLease`、`restoreLease` 字段与 `enter` 分支均为目标态，当前代码不存在，无法编译验证）。
- 真实 restore、真实备份／旧库、E2E、Linux host 层（host／ACL／mount／loader／fd-exec／maps／network／witness）：**NOT RUN**。
- 真实见证方／HSM／外部审批系统：**NOT RUN**；模拟见证仍只能标注「模拟见证、未真实验证」。
- `enter`／`resume` 实际进入与解锁：**NOT RUN**；且按 `NEW-R08-01` 的字面不可能进入。本轮不声称 `enter`／`resume` 解锁。
- 生产 restore 可用、约束 8 满足、`identity-auth-S4.24a` 可关闭：均**不成立**，本轮不作此声称。
- ADR-0031 `adr-0031-r10` 全文同步：**NOT REVIEWED**（由并行会话维护，`NewRuntime` 仍 0 次）；如需同步，属升级待决，不作为本轮阻断。

---

## 9. 复审条件

1. `NEW-R08-01`：修订后须给出 `enter` secret 的可实施取得点（或明确删除 `:1682` 的本次 secret 要求），并使 `Enter` 的 nil 判定与设计一致。
2. `NEW-R08-02`：修订后须给出两条 goroutine 的独立 join 与 `closeRestoreLease()` 先后的可机核判据，删除「两侧都被 join」的当前表述。
3. `NEW-R08-03`：`hashes.txt` 记录后缀值与取得时点（或明确说明 r68 未变化），`:4072` 断言改为条件式。
4. 若上述修订触及 `:1682`、`:4195` 或配置键集合，需一并复核 `:4070`／`:4077` 的联动表述与无回退项。

---

## 10. 写入清单

- `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r08/report.md`（本文件）
- `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r08/hashes.txt`（对象／基线／golden vector／后缀／本报告哈希）