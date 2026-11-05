# SEC r08 独立复审报告 —— identity-auth 详细设计 r68

- 日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
  - 冻结值：**1,093,211 B / 5,476 行 / SHA256 `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083`**
  - 版本：`identity-auth-design-ir01-r68`（`:5`、`:38`）
  - 自建冻结快照：`%TEMP%\design-r68-freeze-5E324AF9.md`，字节/行/摘要与上值一致（已核）
- 上一轮对象（对照基线）：r67 `1,084,882 B / 5,476 行 / B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E`
- 基准报告：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r07/report.md`
  - 34,342 B / 233 行 / SHA256 `5F988533618E72D10712D8E52A2D7E6EAA62BC4FC83EFE19D83D4077D13D6D83`（Verdict: CHANGES REQUIRED）
- 前序参考：`review-sec-r05/report.md`（39,093 B / `4A229267…`）、`review-sec-r06b/report.md`（19,005 B / `169C1116…`，均 CHANGES REQUIRED）；r06 并行会话报告因对象漂移中止，**不构成接受**，只作线索
- 作者报告（只作线索）：`design-revision-r08/report.md` 17,427 B / 160 行 / `5C0970F8B2B149141BAFA319FCC074A87F52A0A6C496D9197B42AFBA4130DB42`；`hashes.txt` 3,459 B / 65 行 / `4770136A02F3C706AB60170E18CB0E26CE7252A2E39175CFD0F85380A9CE9757`
- 关联只读对象：`tasks.md` 898,536 B / 3,289 行 / `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA`；`framework.md` 285,639 B / 1,273 行 / `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`（`adr-0031-r10`，并行会话维护）

## 0 方法、两轮检查范围与只读声明

方法：按 `add-reviewer` / `add-code-review`（objectType=design）对冻结对象做独立两轮审查，并沿真实消费者（`cmd/han-server/recovery_runtime.go`、`recovery_sources.go`）有界核对；消费 `code-review-checklist`、`abstraction-principles`、`decisions-guide`。全部数值本轮自行复算，不转引作者报告。

- 第一轮：r67→r68 逐行差异（`[IO.File]::ReadAllLines` 逐行比对，UTF-8 字节差求和）、9 处变更行全文通读、句级替换/扩展检查、括号平衡复算。
- 第二轮：r07 四项开放项逐项闭合判定、15 项无回退锚点通读、两条 golden vector 字节/摘要独立复算、角色封闭枚举比对、信任根键字面出现面复算、`enter` join 语义与现役代码交叉核对、`SEC-F007`/`N005` 边界复核。

只读声明：本轮仅写入本文件（及同目录 `hashes.txt`）；未改 `design.md`、`tasks.md`、ADR/`framework.md`、`cmd/**`、`internal/**`、测试、既有 `review-*/`、`design-revision-*/`；未删除/移动/归档；未做任何 Git 写操作。

## 1 结论

**Verdict: CHANGES REQUIRED**

| 类别 | 计数 | 说明 |
|---|---|---|
| BLOCKER（本对象范围内） | 0 | — |
| HIGH（本对象范围内） | 0 | — |
| MEDIUM（本对象范围内） | 0 | — |
| LOW（本对象范围内） | 1 | `NEW-S08-02` |
| NOT RUN（不计入缺陷计数） | 1 | `SEC-F007`（Linux 实跑条件缺失，不改判 PASS） |
| 范围外 BLOCKER（只登记） | 1 | `N005`（ADR 状态段与 `tasks.md` 冲突） |

开放项清单：`NEW-S08-02`（LOW，本对象范围内唯一开放项）；`SEC-F007`（NOT RUN）；`N005`（范围外，仅登记）。

## 2 r07 四项开放项闭合判定

| 编号 | 等级 | 判定 | 证据 |
|---|---|---|---|
| `NEW-S05-05` | MEDIUM | **CLOSED** | `:4070`/`:4088`/`:4373`/`:4195` 只用长名；`:4195` 声明长名 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH`／`…_SHA256` 为**唯一合法字面**，短名 `HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH`／`…_SHA256` 出现即 `ErrAdmissionBinding`、不做别名/回退；两长名均已加入 `recoveryRootConfigurationKeys`，`readAction` 仅在 `actionRequiresRestore(action)` 为真时读取。复算出现面：长 PATH 共 4（`:4070`1/`:4088`1/`:4195`2），长 SHA256 共 5（`:4070`1/`:4088`1/`:4195`2/`:4373`1）；短名各仅 `:4195` 拒绝子句 1 次。无混合取值面。ADR `framework.md:1118` 亦用长名（并行维护，只记录） |
| `NEW-S05-06` | LOW | **CLOSED** | `:4079` 已限定为「唯一有效的**载荷绑定** V 编号是 `:4062` 的 V7–V11」，并明示 `V1`–`V6` 仍是有效前置检查点、完整集合以 `:4037` 的 V1–V11 与 `:4062` 为准、本条不缩减任何检查点；与 `:4037`/`:4062`/`:3990` 不再冲突 |
| `NEW-S05-07` | LOW | **CLOSED** | `:4077` 已补齐：单 active pin、无过渡期、请求接受窗口长度固定 **0**；判定时点 `effective_at`（UTC µs），`now < effective_at` 仅旧 pin、`now >= effective_at` 仅新 pin；四字段绑定 `old_spki_sha256`／`new_spki_sha256`／`effective_at`／(`approver_person_ref`+`audit_record_ref`)；审计记录切换前写入并 `fsync`，失败不切换；旧 pin 过期仍见 → `ErrAdmissionUnsupported`、未登记锚 → `ErrAdmissionBinding`；切换后 `trust_root_generation`／`anchor_generation` 必须变化。可判且不可绕过 |
| `NEW-S07-01` | LOW | **CLOSED** | `:4070` B1 重验点已标「只作摘要、非穷举」，补入 `P3`/`P6`（`Journal.Begin` 前）与 `V7`/`V8`/`V10`（推进 `sequence` 前）；错误分类与 r67.5 对齐：`P3`/`P6`/`R1`/`V1`/`R2`/`V2`/`R4`/`V4` 漂移 → `ErrAdmissionClosed` 且不进 `Journal.Begin`；`R6`/`V6` 与 `V7`/`V8`/`V10` 前漂移 → `restore_unknown`（`errors.Join(recovery.ErrRecoveryWriteUnknown, err)`），且与 `:4037`「`restore_unknown` 或 closed」一致 |

## 3 新问题

### NEW-S08-02（LOW）：`:4076` r68「两侧都被 join」的机制断言不成立

- 位置：`design.md:4076`（r68 追加句）；关联现役代码 `cmd/han-server/recovery_runtime.go:516-520`、`:901-914`、`:916-921`、`:932-939`、`:942-964`，`recovery_sources.go:314-327`、`:332-357`。
- 原文断言：「`enter`／`resume` 的 `waitFlow` 等待的是 `start` 起的两条 goroutine 所汇集的 `flowDone`，因此 `Enter` 与 `completeOnRequest` 两侧都被 join。」同段并称 `waitFlow`「等待并 reap 已启动的 restore 工作（…避免孤儿进程／goroutine）」。
- 机制事实：`flowDone` 由 `setFlowResult` 经 `flowOnce.Do` 在**第一条**结果上关闭（`recovery_runtime.go:916-921`），`waitFlow` 只有在 `flowDone` 关闭或 ctx 结束时返回（`:932-939`）；它是「首个结果信号」，不是 WaitGroup，也不追踪第二条 goroutine 的函数体是否退出。`completeOnRequest` 在 `completeIdentity` 中执行真实 executor 调用（`recovery_sources.go:346-351`）。
- 触发场景：`enter`/`resume` 的 `bounded` ctx 到期时，`Enter` 侧经 `CompleteIdentityReview` 的 `ctx.Done()` 分支（`recovery_sources.go:324-325`）返回 → 第一侧 `setFlowResult` 关闭 `flowDone` → `joinWork` 依 `Stop → waitFlow → closeRestoreLease` 执行 `closeRestoreLease`；此时 `completeOnRequest` 侧可能仍停在 `b.executor.CompleteIdentityReview`（若真实执行器不配合 ctx 取消，则继续运行）。即 `restoreLease`（exclusive 独占租约）可能在 executor 仍在执行时被 `Close`。
- 影响：独占互斥在 restore 仍在进行时被提前释放，理论上允许第二个 exclusive 持有者进入；这与设计「恰好一个 exclusive 持有者」的不变量冲突。设计已有 teardown 结果分类（已启动 → `restore_unknown`），但该分类约束的是**结果**，不约束 `closeRestoreLease` 相对第二条 goroutine 退出的**顺序**。
- 为何是缺陷：r68 把 `:4076` 改写为「可 join 的 goroutine」并给出「两侧都被 join / 避免孤儿 goroutine」的断言，但所引机制（`flowOnce`/`flowDone`）不能支撑该断言；实现者若据此认为租约关闭前两条 goroutine 必已退出，会漏掉明确的 join 屏障。属对安全判据支撑的过度声明，非纯风格。
- 修复建议（任选其一，落到设计文本）：① 在 assembly 增 `sync.WaitGroup`（或第二条 done channel），两条 goroutine 各自在退出前 `Done`，`waitFlow`/`joinWork` 在 `closeRestoreLease` 之前等待**两条**均已退出；② 或显式规定 `closeRestoreLease` 只在「executor goroutine 已退出」的屏障之后执行，并把该屏障列为可机核判据；③ 若维持现机制，则把 `:4076` 断言收窄为「首个结果收敛到 `flowDone`」，删除「两侧都被 join / 避免孤儿 goroutine」的强断言，并单列 teardown 顺序要求。
- 严重度说明：定 LOW（需 budget 到期且执行器不配合 ctx 取消才显现，且已有 `restore_unknown` 结果分类）。**若真实执行器不保证响应 ctx 取消，则应按 MEDIUM 处理**；本报告的 LOW 定级以「执行器最终可被取消/回收」为前提。

## 4 无回退确认（逐项，锚点为冻结 r68 原文）

| 编号 | 锚点 | 证据 / 现状 | 判定 |
|---|---|---|---|
| `SEC-F002` | `:4061` | 六值封闭枚举 `restore_approver`／`restore_reviewer`／`restore_executor`／`restore_producer`／`restore_verifier`／`restore_witness` 原样；裸名映射、`person_ref` 全局唯一、六席两两不同约束在 | 未回退 |
| `SEC-F003` | `:3939`/`:4037`/`:4057`/`:4059`/`:4062`/`:4068`/`:4074`/`:4127` | 唯一调用图 P1–P6／R1–R8／V1–V11 与 V 编号语义在；未删检查点 | 未回退 |
| `N001` | `:3987`/`:4033`/`:4057`/`:4059`/`:4068`/`:4074` | `receipt_anchors` 14 项 exact keyset、`baseline_sequence` 在；lease 注入半边在 | 未回退 |
| `SEC-F006` | `:3983`/`:4041`/`:4057`/`:4064`/`:4082`/`:4205`/`:4207` | 能力断言、`CapBnd`/`NoNewPrivs` 观测化、seccomp/netns unsupported 原样 | 未回退 |
| `N011` | `:4057`/`:4060`/`:4077` | 见证端点、TLS pin、独立性、`witness_host`/地址判定在 | 未回退 |
| `NEW-S05-01` | `:4078` | `PATH` 放宽的抵消措施（目录约束、禁 `LookPath`、上限）在 | 未回退 |
| `NEW-S05-02` | `:4072`/`:4077`/`:4078` | 唯一优先级规则与 `InsecureSkipVerify` 受控例外边界在 | 未回退 |
| `NEW-S05-04` | `:4077` | `root`/`supplier` `appointment_ref` 只允许空串、不入六值枚举在 | 未回退 |
| `N012` | `:4057`/`:4065`/`:4082` | `receipt_version` 字面、第三别名移除要求在 | 未回退 |
| `N013` | `:4057`/`:4066` | `Target.DSN` 删除而非改类型在 | 未回退 |
| `N014` | `:4057`/`:4067` | `static`/`dynamic` 分档合取在 | 未回退 |
| `N015` | `:4057`/`:4062` | V 编号与 `anchor_ref` 等式链在 | 未回退 |
| `N016` | `:4057`/`:4082` | 已闭环状态不回退声明在 | 未回退 |
| `NEW-S06-02` | `:4062`/`:4135`/`:4137` | `V10` 验证方签名后、推进见证 head `sequence` 前再 `revalidateApproval`；失败不推进、按 `restore_unknown` 在 | 未回退（ADR 侧 V7/V8 跨文件项另计，不在本对象体） |
| `NEW-S06-03` | `:4369`/`:4373` | `generation` 三元摘要（`approval_generation`＝批准文件字节 SHA-256、`trust_root_generation`＝信任根文件字节 SHA-256 且等于长名 ENV、`anchor_generation`＝`receipt_anchors` 条目 JCS SHA-256）在，逐字节相等、无 inode/长度替代 | 未回退 |

`NEW-S06-01`（字节账）：r68 9 行字节差求和 `+8,329` 恰等于 `1,093,211 − 1,084,882`，字节账配平；不作为开放安全项。

## 5 golden vector 独立复算与角色枚举

在冻结快照上对 canonical 行取 `UTF8.GetBytes(line)` 复算（不含围栏、不含换行）：

| 向量 | 行 | 复算字节 | 复算 SHA-256 | 设计声明 | 一致 |
|---|---|---|---|---|---|
| source | `:4099` | **1,115** | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 / `de914622…`（`:4102`） | 是 |
| tool | `:4105` | **2,357** | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 / `8c0c89…`（`:4108`） | 是 |

角色字面：source 含 `restore_approver`／`restore_reviewer`，tool 含 `restore_executor`／`restore_approver`／`restore_reviewer`；`restore_source`／`restore_tool` 是 `record_type` 值，不是角色名。全部角色字面均落在 `:4061` 六值封闭枚举内（`restore_approver`、`restore_reviewer`、`restore_executor`、`restore_producer`、`restore_verifier`、`restore_witness`），无越界。

## 6 r68 改动、字节账与写盘历史

- 变更行恰 9 处：`:5`(0)、`:38`(0)、`:4070`(+709)、`:4072`(+1,466)、`:4076`(+4,624)、`:4077`(+823)、`:4079`(+224)、`:4083`(+85)、`:4195`(+398)；求和 **+8,329** = `1,093,211 − 1,084,882`；行数 5,476 不变（无增删行）。
- 语义性质：均为行内替换/扩展（声明、映射、键字面、错误分类、V 编号限定、pin 轮换、join 断言），未删除任何安全判据；`r68` 修订登记于 `:4072`（`r68.1`–`r68.8`）与 `:4083`（沿用声明，不改 NOT RUN/unsupported/closed）。
- 作者 3 次写盘（只作线索，已交叉核）：① 1,093,138 / `A95470BA…`；② 1,093,208 / `43BFF315…`（自述 `:4076` 全角括号失衡）；③ 终值 1,093,211 / `5E324AF9…`。独立复算 `:4076` 全角括号 **82/82** 平衡（`（`=82、`）`=82），终值语义完整，第 2 次写入的失衡已被第 3 次修复；中间值不得作为引用基线。
- 但终值内仍含 `NEW-S08-02` 所述 join 断言（见第 3 节），属 r68 终值内容，未因括号修复而消除。

## 7 SEC-F007、N005 与 ADR 观察

- `SEC-F007`：**仍为 NOT RUN**。`:4070`(6) 明示 host／ACL／mount／loader／fd-exec／maps／network／witness 实测未运行；Windows 只覆盖解析与语义层，`//go:build linux` 的 skip 与无匹配测试 exit 0 都不等于 PASS。本轮不因任何结论改判 PASS。
- `N005`（范围外阻断，仅登记不代改）：`framework.md:1084`／`:1129` 记 `adr-0031-r10` 两路接受尚未取得、`identity-auth-S4.24a` 不得启动；`tasks.md:3197` 写 r07 已接受、启动门禁解除；`tasks.md:3186` 为 `- [ ] identity-auth-S4.24a …`。二者冲突属架构决策方与 Task-Planner 范围，不计入本报告主体计数。
- ADR 观察：`framework.md` 正被并行会话改写（现 `adr-0031-r10`），其 `:1118` 已用长名 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`；本轮只记录、不改。

## 8 未验证风险与复审条件

未验证风险：
- `SEC-F007` 全部 Linux 实跑项 NOT RUN，本轮结论不覆盖 host/ACL/mount/loader/fd-exec/maps/network/witness 的真实行为。
- `NEW-S08-02` 的运行时显现依赖执行器对 ctx 取消的响应性；本轮只做设计与现役代码的机制核对，未做并发压测或故障注入。

复审条件（闭合 `NEW-S08-02` 后）：① `:4076` 明确「`closeRestoreLease` 之前两条 goroutine 均已退出」的屏障（WaitGroup/第二 done/可机核判据）；或②收窄 join 断言并单列 teardown 顺序要求；③ 保留第 4 节无回退证据；④ 不改动 `SEC-F007` 与 N005 边界。

## 9 声明

本报告只完成对冻结 r68 的独立安全侧设计复审，不构成：生产解锁、`enter`/`resume` 解锁、约束 8 满足、`N005` 已解决、`identity-auth-S4.24a` 可关闭或已交付；`SEC-F007` 仍为 NOT RUN，不因本报告改判 PASS。任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。最终接受由 Lead/用户决定。