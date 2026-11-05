# identity-auth 详细设计修订 r68 报告

角色：Feature-Designer（`add-feature-designer` / 主方法 `add-design-tdd`）。性质：对 `identity-auth` 详细设计做 r68 就地修订，闭合架构复审 r07（F006 等）与安全复审 r07（NEW-S05-05 等）的开放项。本报告只陈述设计层改动与可机核证据，不代替独立复审。

## 1 输入与基线实测

| 对象 | 实测值 |
|---|---|
| 工作区 / 分支 / HEAD | `D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 改前 `design.md` | 1,084,882 B / 5,476 行 / SHA-256 `B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E` / raw CR=169 / LF=5,476；版本行 `:5`／`:38` 均为 `identity-auth-design-ir01-r67` |
| 改前快照 | `design-revision-r08/design.r67.baseline.md`，1,084,882 B / 5,476 行 / SHA-256 同上（与派发声明值一致） |
| 改后 `design.md` | 1,093,211 B / 5,476 行 / SHA-256 `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083` / raw CR=169 / LF=5,476 |
| 架构复审 r07 | `review-arch-r07/report.md` 35,228 B / 325 行 / `3F27DAF27EA558CB83E733CB6F581A55CCBB69EFD151705710FFDDCC580824B5` |
| 安全复审 r07 | `review-sec-r07/report.md` 34,342 B / 233 行 / `5F988533618E72D10712D8E52A2D7E6EAA62BC4FC83EFE19D83D4077D13D6D83` |

开工实测与派发声明值一致，未出现基线差异。快照先于任何写入生成，未再覆盖。

## 2 变更总览与字节账

- 仅改 `.add/project/knowledge/design-workspace/identity-auth/design.md`；相对 r67 快照共 **9 行变化**（`:5`、`:38`、`:4070`、`:4072`、`:4076`、`:4077`、`:4079`、`:4083`、`:4195`），全部为行内替换/追加，**无新增行、无删除行**，行数保持 5,476，故所有 `:4xxx` 内文锚点仍有效。
- CR 计数保持 169（169 个 CRLF 行 + 5,307 个裸 LF 行），换行结构未被改写；文件末尾仍为单个 LF。

逐行字节账（Base = r67 快照该行 UTF-8 字节数，New = r68 该行 UTF-8 字节数）：

| 行 | 主题 | Base B | New B | Δ |
|---|---|---|---|---|
| `:5` | revision 号 | 39 | 39 | 0 |
| `:38` | 设计版本 | 50 | 50 | 0 |
| `:4070` | B1 重验枚举与错误分类 | 4,498 | 5,207 | +709 |
| `:4072` | 优先级、可复算规则、r68 清单 | 4,280 | 5,746 | +1,466 |
| `:4076` | `ownershipLease` 数据流、`activate`、`enter`／`completeOnRequest` | 10,088 | 14,712 | +4,624 |
| `:4077` | pin 轮换窗口与判定时点 | 3,877 | 4,700 | +823 |
| `:4079` | V 编号适用范围限定 | 762 | 986 | +224 |
| `:4083` | r68 沿用声明 | 341 | 426 | +85 |
| `:4195` | 信任根键名唯一字面与别名拒绝 | 1,728 | 2,126 | +398 |
| 合计 | | 1,084,882 | 1,093,211 | **+8,329** |

复算规则：对两文件同名行取 `UTF8.GetBytes(line).Length` 相减，逐行求和必须等于两文件总字节差；实测 `8,329` 与 `1,093,211 − 1,084,882` 相等。

## 3 逐项闭合

| # | Finding | 级别 | 落点 | 判定 |
|---|---|---|---|---|
| 1 | `F006`(a) / `NEW-R07-01`：`activate` 持锁区间访问器与 `:892` 调用点 | BLOCKER / MEDIUM | `:4076` 第(3)点、`:4072` r68.1 | 设计层闭合 |
| 2 | `F006`(b) / `NEW-R07-03`：`admission.go:648` 无数据流 | BLOCKER / MEDIUM | `:4076` 第(2)点、`:4072` r68.2 | 设计层闭合 |
| 3 | `NEW-R07-02`：`enter` 同步运行导致死锁、`completeOnRequest` 未定义 | BLOCKER | `:4076` `start` 段与 `completeOnRequest` 段、`:4072` r68.3 | 设计层闭合 |
| 4 | `NEW-R05-03` / `NEW-S05-06`：V7–V11 适用范围未限定 | LOW | `:4079`、`:4072` r68.4 | 设计层闭合 |
| 5 | `NEW-R07-04`：`:4072` 三段易腐后缀哈希陈旧 | LOW | `:4072`（可复算规则）、r68.5 | 设计层闭合 |
| 6 | `NEW-S05-05`：信任根短名/长名冲突与混合取值 | MEDIUM | `:4070`、`:4195`、`:4072` r68.6 | 设计层闭合 |
| 7 | `NEW-S05-07`：pin 轮换缺窗口、判定时点、批准与审计绑定 | LOW | `:4077`、`:4072` r68.7 | 设计层闭合 |
| 8 | `NEW-S07-01`：B1 重验枚举缺 V10/P6、错误分类未同步 | LOW | `:4070`、`:4072` r68.8 | 设计层闭合 |

「设计层闭合」= 该条可在文本内按字面实施、判据可机核；不等于独立复审通过（见 §8）。

### 3.1 F006(a) / NEW-R07-01（`activate` 持锁区间与 `:892`）

- 落点：`:4076` 第(3)点；`:4072` r68.1 登记。
- 修改：`activate` 在 `:918` 已持 `r.mu`，`:919` 内固定写作 `p, lease, generation := r.proof, r.activeExclusiveLocked(), r.generation`，并**禁止**在 `:919` 调用 `activeExclusive()`（`sync.Mutex` 不可重入）。`activate` 的调用点逐个点名：`RunNormal` 的 `:892`、`replay` 的 `:1943`，两处都只经 `activate` 读取 lease（即 `:919` 的 `activeExclusiveLocked()`），不另设读取点，也不在调用点前置取值。
- 机核：`activate` 函数体内 `activeExclusive(` = 0、`activeExclusiveLocked(` = 1；`rg -n 'r\.activate\(' internal/recovery/runtime.go（定义行不计入）` 命中恰为 `:892`／`:1943`（实测：`func (r *admissionRuntime) activate(` 不含 `r.activate(`，故定义行不被命中）。
- 判定：闭合；`activeExclusive()`／`activeExclusiveLocked()` 的二义性在本条内被消灭。

### 3.2 F006(b) / NEW-R07-03（`admission.go:648` 数据流）

- 落点：`:4076` 第(2)点；`:4072` r68.2 登记。
- 修改：新增包内 helper `ownershipLease(ctx) (ExclusiveLease, error)`，取值与失败分类固定为六步：①持 `r.mu` **一次**取 `lease := r.activeExclusiveLocked()`、`generationCtx`、`action` 后立即解锁；②`generationCtx == nil` → `ErrAdmissionClosed`；③`generationCtx.Err() != nil` → `r.fail(err)` + 原错误；④`missingRuntimePort(lease)` 的 nil/typed-nil 判定**先于** `checkExclusive`，injected 动作（`enter`／`resume`）→ `ErrAdmissionClosed`，owned 动作（`install`／`normal`／`verify`）→ `ErrAdmissionBinding`；⑤`checkExclusive` 其他失败 → `r.fail(err)` + 原错误；⑥返回同一 `lease` 与 `nil`。
- 数据流：`readControlled` 入口取一次 `lease`，`err != nil` 立即返回、不读 journal／store；`:648` 改为 `recheckProof(ctx, r.deps, lease, v.proof)`；`:617` 的 `checkOwnership` 调用删除。`checkOwnership(ctx) error` 变薄包装 `_, err := r.ownershipLease(ctx); return err`，六个调用者签名不变（`runtime.go:293`／`:404`／`:1123`／`:1196`／`:1245`、`coordinator.go:96`）。
- 机核：`readControlled` 函数体内 `r.exclusive` 直读 = 0、`checkOwnership(` = 0；`ownershipLease` 签名唯一。
- 判定：闭合；「谁取值、如何传、nil 何时判、失败分类」四问均有单一答案。

### 3.3 NEW-R07-02（`enter` 死锁与 `completeOnRequest`）

- 落点：`:4076` `start` 段与 `completeOnRequest` 段；`:4072` r68.3 登记。
- 修改：`start` 的 `enter` 分支改为 `bounded, cancel := context.WithTimeout(ctx, budget)` 后起**两条可 join 的 goroutine**（其一 `setFlowResult(a.runtime.Enter(bounded, ...))`，其二 `completeOnRequest(bounded)`，各自 `defer cancel()`），随后**立即 `return nil`**，使 control server 能启动并接受操作员确认；文本明写「不得在 `start` 内同步调用 `Enter`」，并给出死锁链 `Enter → f.enter → f.identity → CompleteIdentityReview`。
- `completeOnRequest`：签名固定 `func (a *recoveryRootAssembly) completeOnRequest(ctx context.Context)`；调用点恰为 `start` 的 `resume` 与 `enter` 分支各一次；HTTP 层只在接受响应 flush 之后 signal。流程：`waitCompleteRequest` 失败 → `setFlowResult(err)` + `requestStop()`；`completeIdentity` 失败 → 同；成功后 `setFlowResult(a.waitFlow(ctx))` 再 `requestStop()`。identity 完成时点由 `completeIdentity` 关闭 `completeDone` 决定（`recovery_sources.go:332`–`:357`），`CompleteIdentityReview`（`:314`–`:327`）只阻塞在 `completeDone` 或 `ctx.Done()`。
- 等待边界与超时：`enter` 的等待边界就是同一个 `bounded` ctx；超时／取消经 `waitCompleteRequest`、`ctx.Done()`、`waitFlow` 返回可诊断错误，随后 `requestStop()` 取消 generation 并交 `joinWork` 收敛，**不存在无界等待**。
- 收敛：`joinWork` 增 `case "enter"`（`Stop` → `waitFlow` → `closeRestoreLease`），`case "resume"` 同序补齐；`waitFlow` 等待的是两条 goroutine 汇集的 `flowDone`，故 `Enter` 与 `completeOnRequest` 两侧都被 join。
- 判定：闭合；同步死锁被消除，`completeOnRequest` 不再是未定义符号。

### 3.4 NEW-R05-03 / NEW-S05-06（V 编号适用范围）

- 落点：`:4079`；`:4072` r68.4 登记。
- 修改：唯一有效的 **载荷绑定** V 编号限定为 `:4062` 的 V7–V11；明确 `V1`–`V6` 仍是有效的前置检查点（`prepared(ctx)`、`revalidateApproval`、`VerifyRestoreSource` 等），完整集合以 `:4037` 的 V1–V11 与 `:4062` 为准，本条不缩减任何检查点。
- 判定：闭合；与 `:4037`／`:4062` 的检查点集合措辞不再冲突。

### 3.5 NEW-R07-04（易腐后缀哈希）

- 落点：`:4072`；r68.5 登记。
- 修改：删除「现行值」写死哈希，改为可复算规则：起点取该行行首在当次冻结副本中的字节偏移、终点取 EOF，不重新编码、不做换行归一；字节数 = 终点 − 起点，摘要 = 该区间原始字节 SHA-256；每次修订只对当次冻结副本复算并把结果与取得时点写入该次 `hashes.txt`；引用必须同时给出「源文件 SHA-256 + 行号 + 取得时点」。r66 时点值（`:4259` 282,599 B／`7D1DDC08…`、`:4275` 280,319 B／`5BD8E870…`、`:4290` 277,334 B／`E40EB7A2…`）显式标为历史，并说明 r67 两处就地修订（`:4369` +91 B、`:4373` +790 B）已使三段各 +881 B。
- 判定：闭合；正文不再固化易腐数值，规则可复算。

### 3.6 NEW-S05-05（信任根键名唯一字面）

- 落点：`:4070`（B1 载体句）、`:4195`（注册清单与读取点）；`:4072` r68.6 登记。
- 修改：唯一合法字面统一为长名 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH`／`HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`；短名 `HAN_RECOVERY_RESTORE_TRUST_ROOT_PATH`／`HAN_RECOVERY_RESTORE_TRUST_ROOT_SHA256` 未加入 `recoveryRootConfigurationKeys`，只作 r67 及以前历史，**出现即 `ErrAdmissionBinding`**，不做别名解析、不做回退。`readAction` 读取点与 `:4070` 的 B1 描述同步用长名。
- 机核（实测）：全文短名各仅 1 次且都在 `:4195` 的别名拒绝句；长名 PATH 3 处、SHA256 4 处。
- 判定：闭合；「PATH 取短名、SHA256 取长名」的混合取值路径在字面上不再存在。

### 3.7 NEW-S05-07（pin 轮换）

- 落点：`:4077`；r68.7 登记。
- 修改：任一时刻只有一个 active pin；轮换须人的明确批准并留审计，批准记录必须同时绑定 `old_spki_sha256`、`new_spki_sha256`、`effective_at`（UTC 微秒）、`approver_person_ref` + `audit_record_ref`，缺一或与 `receipt_anchors.witness_tls_spki_sha256` 不一致 → `ErrAdmissionBinding`。判定时点固定为 `effective_at`（`now < effective_at` 只用旧 pin，`now >= effective_at` 只用新 pin，无过渡期）；「部署回滚窗口」只指离线回滚材料，**请求接受窗口长度固定为 0**，`now >= effective_at` 后仍见旧 pin → 立即 closed（`ErrAdmissionUnsupported`；未登记条目 → `ErrAdmissionBinding`）；审计记录须在切换前写入并 `fsync`，失败不得切换；切换后 `trust_root_generation`／`anchor_generation` 随之变化。
- 判定：闭合；窗口长度、判定时点、批准与审计字段绑定四项齐备。

### 3.8 NEW-S07-01（B1 重验枚举与错误分类）

- 落点：`:4070`；r68.8 登记。
- 修改：B1 只作摘要、**非穷举**，完整集合以 `:4037`／`:4062`／`:4135`／`:4137` 为准；枚举补 `P3`、`P6`（`Journal.Begin` 前）、`prepared(ctx)`＝`R1`／`V1`、`R2`／`R4`／`V2`／`V4`、`R6`／`V6`、`V7`／`V8`／`V10`（验证方签名后、推进见证 `sequence` 前）。错误分类随检查点同步：启动预检与 `P3`／`P6`／`R1`／`V1`／`R2`／`V2`／`R4`／`V4` 漂移 → `ErrAdmissionClosed` 且不进 `Journal.Begin`；`R6`／`V6` 与 `V7`／`V8`／`V10` 前漂移 → `restore_unknown`（`errors.Join(recovery.ErrRecoveryWriteUnknown, err)`），与 `:4037` 的「`restore_unknown` 或 closed」一致。
- 判定：闭合；枚举与错误分类同 r67.5 对齐。

## 4 保持不回退项核对

以下条目本轮未改其承载行，逐行 diff 已确认不在改动集合内：`SEC-F002`、`SEC-F003`、`N001`、`SEC-F006`、`N011`、`NEW-S05-01`／`02`／`04`、`N012`–`N016`、`NEW-S06-02`／`03`、`F004` 文本层、`NEW-R05-04`、`NEW-R05-05` 行号部分、`NEW-R06-02`／`05`／`06`／`07`／`08`。

同时确认：`:4083` 的 r66.11 声明原文保留，仅追加「r68 沿用本声明，不改变任何 NOT RUN／unsupported／closed 状态」；`:4072` 的 r67 就地修订清单原文保留，仅追加 r68 清单与优先级扩展（`r68 > r67 > …`）；`bundle_digest` 仍为显式 OPEN、`nonce` 语义、`SEC-F007` 仍 NOT RUN 均未动。

## 5 机核判据与复算命令

| 判据 | 命令 | 实测结果 |
|---|---|---|
| 行数 | `[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8).Length` | 5,476（与 r67 相同） |
| raw CR / LF | 逐字节计数 `$_ -eq 13` / `$_ -eq 10` | CR=169 / LF=5,476（与 r67 相同） |
| 版本行 | `$lines[4]`、`$lines[37]` | 均为 `identity-auth-design-ir01-r68` |
| 改动行 | 逐行 `-cne` 比较快照 | 仅 `5,38,4070,4072,4076,4077,4079,4083,4195` |
| golden source | `:4099` 行字节 SHA-256 | 1,115 B / `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` |
| golden tool | `:4105` 行字节 SHA-256 | 2,357 B / `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` |
| 全角括号平衡（改动行） | 计 `（` 与 `）`，与快照同名行比对 | 8 行与快照同为平衡；`:4076` 82/82（快照 65/65） |
| 关键句存在性 | 子串检索 | `ownershipLease(ctx) (ExclusiveLease, error)`=1、`activeExclusiveLocked(), r.generation`=1、`请求接受窗口长度固定为 0`=1、`判定时点固定为`=1、`唯一有效的**载荷绑定** V 编号`=1、`正文不固化易腐数值`=1、`不做别名解析`=1 |
| 信任根键名 | 子串计数 | 短名各 1 次（仅 `:4195` 拒绝句）；长名 PATH 3、SHA256 4 |

## 6 升级待决（本会话只登记，不做）

1. `F004` 的 ADR 同步：设计文本层已闭合，但 ADR 侧尚未同步，需架构决策方处理。
2. `F017` 任务面对账：`tasks.md` 写集缺口未处理；本轮按硬约束不改 `tasks.md`（含 `:3186` 的 S4.24a 复选框），写集更新前相关任务保持不可实施。
3. 范围外 `N005`：`decisions/framework.md` 状态段与 `tasks.md:3197` 状态段冲突，属架构决策方与 Task-Planner 范围，本轮未改 ADR。

## 7 NOT RUN

- build / `go test` / `go vet` / `-race` / 任何单元或集成测试：未运行（本轮为纯设计文本修订）。
- Linux host、ACL、mount、loader、fd-exec、maps、network、witness 的真实环境验证：未运行（Windows 主机，`//go:build linux` skip 与无匹配测试的 exit 0 均不等于 PASS）。
- `SEC-F007`：仍为 NOT RUN，本体未改。
- ADR 复审、`framework.md` 同步、`tasks.md` 对账：未执行。
- 真实 restore／生产运行、`enter`／`resume` 实际进入：未执行。

## 8 边界与声明

- 本修订不构成独立评审通过，不构成生产 restore 解锁，不构成 `enter`／`resume` 解锁，不构成约束 8 满足，不构成 `identity-auth-S4.24a` 可关闭，不构成任何已交付结论；任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。
- 本轮未执行任何 Git 写操作（无 add／commit／branch／stash／reset／checkout）；`tasks.md`、`decisions/framework.md`（ADR-0031 r10）、`cmd/**`、`internal/**`、测试、既有 `review-*/`、`design-revision-r01..r07/` 均未写入；未删除、未移动、未归档任何文件。
- 并行会话此前对 ADR（`adr-0031-r10`）与 r67 的改动已保持原样，未回退。

## 9 写盘路径与中间态（可追溯）

`design.md` 共写盘 3 次，数值以最后一次为终值：

| 次序 | 字节 | SHA-256 | 说明 |
|---|---|---|---|
| 1 | 1,093,138 | `A95470BA8EFF3EEA246DB48D4B184D284AB2C1F51865DBB31298AB3CFAE051FD` | r68 主体 21 处替换（+8,256） |
| 2 | 1,093,208 | `43BFF3159C76616239752995A0A577D32CA3ECE11CE6C6D0ECDA822576773235` | 3 处机核/措辞润色（+70） |
| 3（终值） | 1,093,211 | `5E324AF9E5C6FA664093B8121E746F8DB2C83CE6C79E7C3318C7ADE8C0687083` | 修复第 2 次写入引入的 `:4076` 全角括号失衡（+3） |

第 2 次写入把「`之前）（`admission.go:405`–`:406`）`」改为「`之前，对应 `admission.go:405`–`:406``」时连带删掉了外层括号的闭括号；自测发现 `:4076` 全角括号 82/81 失衡（快照 65/65），第 3 次写入补回闭括号为「`之前），对应 …`」，复测 82/82 平衡。除该字符外，终值与第 2 次写入内容一致；所有中间值均为可复算的历史，**不得作为引用基线**。