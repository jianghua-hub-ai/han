# identity-auth 详细设计修订报告 — r72（就地修订，零增删行）

- 日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 设计对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 修订号：`identity-auth-design-ir01-r71` → `identity-auth-design-ir01-r72`（`:5`、`:38` 同步）
- 改前快照：`design-revision-r12/design.r71.baseline.md`（与改前 `design.md` 逐字节相等）
- 写盘方式：`[IO.File]::ReadAllText($p,[Text.Encoding]::UTF8)` → 对精确子串 `Replace` → `[IO.File]::WriteAllText($p,$txt,(New-Object Text.UTF8Encoding($false)))`；未使用 `Set-Content`／`Out-File`，未整体重排或归一化换行。仅一次写盘。
- 读文件：`[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)`；哈希：`Get-FileHash -Algorithm SHA256 -LiteralPath`（PowerShell 5.1）。

> 本报告只完成设计文本层修订。**不构成评审通过，不构成生产 restore 解锁，不构成 `enter`／`resume` 解锁，不构成约束 8 满足，`identity-auth-S4.24a` 不可关闭**；任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。

## 1. 对象状态（改前／改后六项）

| 项 | 改前（r71） | 改后（r72） |
|---|---|---|
| 字节 | 1,106,644 | 1,110,477 |
| 行数 | 5,476 | 5,476 |
| CR 数 | 169 | 169 |
| LF 数 | 5,476 | 5,476 |
| BOM | 无 | 无 |
| 末字节 | `0A`（LF） | `0A`（LF） |
| SHA256 | `168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5` | `491A476E60636ADAA3F1610A07C78054ED4C9735F5AD42D60B76200A12C8414F` |

- 字节增量：**+3,833 B**；行数不变（零增删行）；换行结构保持 169 处 CRLF + 其余 bare LF，末尾 LF。
- 快照核验：`design.r71.baseline.md` = 1,106,644 B / SHA256 `168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5`，与改前对象逐字节相等（`equal=True`）。
- 全角括号平衡：改前 1,237/1,237 → 改后 1,258/1,258（本报告另记实测 1,258/1,258）。

## 2. 改动行与逐行字节账（合计 = 全文增量）

| 行 | 改前 B | 改后 B | Δ |
|---|---|---|---|
| `:5` | 39 | 39 | 0 |
| `:38` | 50 | 50 | 0 |
| `:3944` | 1,370 | 1,803 | +433 |
| `:4031` | 2,006 | 2,316 | +310 |
| `:4051` | 1,365 | 1,915 | +550 |
| `:4062` | 2,655 | 2,962 | +307 |
| `:4070` | 5,858 | 6,077 | +219 |
| `:4072` | 8,952 | 9,904 | +952 |
| `:4083` | 681 | 766 | +85 |
| `:4133` | 767 | 1,086 | +319 |
| `:4135` | 1,312 | 1,658 | +346 |
| `:4377` | 1,001 | 1,313 | +312 |
| **合计** | — | — | **+3,833** |

- 逐行 diff（`ReadAllLines` 对改前快照逐行 `-cne` 比较）实测改动集合：`5,38,3944,4031,4051,4062,4070,4072,4083,4133,4135,4377`，与预期集合完全一致，无第 13 行变化。
- 逐行字节差之和 3,833 B = 全文增量 3,833 B（可复算）。
- `:5`／`:38` 仅 `r71`→`r72`，同为 3 字节字符长度，Δ=0。

## 3. 逐 finding 闭合

### 3.1 `NEW-R11-01`（MEDIUM，架构）：policy `tool.receipt.trust_anchor_ref` 缺键／空串／字段名不符的错误类未写死

- 落点行号：`:3944`、`:4031`、`:4062`、`:4070`、`:4377`。
- 修改摘要：
  - `:3944` 末尾追加「**唯一 policy 专属例外（r72.1）**」：该字段缺键、空串或字段名不符**不在**本映射的「缺键 → `ErrAdmissionIncomplete`」之内，而按「启动前 `ErrAdmissionBinding`、工具已启动后 `restore_unknown`」分类；例外只限此字段，不得扩展到任何其他缺键，且不改变 fail-closed 与「任一非 nil 都在 Begin 前返回」。
  - `:4031` 把 policy 侧失败补成显式两类（启动前 `ErrAdmissionBinding`／已启动后 `restore_unknown`），并声明其为 `:3944`／`:4377` 通用「缺键／缺字段 → `ErrAdmissionIncomplete`」的**唯一 policy 专属例外**，只限 `tool.receipt.trust_anchor_ref`。
  - `:4062` 同一口径（五处等式链所在行），避免与 `:4031` 出现两套读法。
  - `:4070` B1 摘要的错误分类段追加「**唯一例外（r72.1）**」，明确其余检查点仍按 `:4377` 原因分类。
  - `:4377` canonical 映射尾部追加「r72.1 增补唯一例外」，说明该字段**不按**本段「缺键 → `ErrAdmissionIncomplete`」。
- 可机核判据（实测，全文计数）：`**唯一 policy 专属例外（r72.1）**` = 3（`:3944`／`:4031`／`:4062`）；`**唯一例外（r72.1）**` = 1（`:4070`）；`r72.1 增补唯一例外` = 1（`:4377`）。三者互不冲突，且均限定「只限 `tool.receipt.trust_anchor_ref`／不得扩展到任何其他缺键」。
- 对齐依据：ADR-0031 `adr-0031-r12`，`framework.md:1118` 实读为「policy 的 `trust_anchor_ref` 是 design 规定的 policy 必填字段，缺键、空串或字段名不符即 closed——启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`」（`framework.md` 只读引用，本轮未改：297,412 B / 1,275 行 / `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`）。
- 闭合判定：**CLOSED（设计文本层）**。三义（`:3944` 通用缺键、`:4031`／`:4062` 只写「不得放行」、`:4377` 缺键归 Incomplete）已收敛为唯一读法；最终判定仍待两路独立复审。

### 3.2 `NEW-R11-02`（LOW，架构）：`:4135`「读取点与时钟点」未同步 V7／V8 前两次重验

- 落点行号：`:4135`。
- 修改摘要：在「其六（r67.5）」之后新增「**其七（r72.2）**」——`V7` 独立读取回执前与 `V8` 验证 producer 签名前各再一次 `revalidateApproval` 类重验；标注与 `:4037`／`:4062`／`:4070` 同登记，**不新增、不重排、不复用 V 编号**；并写明该两点位于工具已启动后，失败按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown` 处置。
- 可机核判据（实测）：`其七（r72.2）` = 1；`V12` = 0；`V1`–`V11` 编号集合未变、未重排。`:4037`／`:4062`（V7／V8 语义定义处）本轮**逐字节未改**。
- 闭合判定：**CLOSED（设计文本层）**。`:4135` 枚举现与 `:4037`／`:4062`／`:4070` 一致，四行并读不再依赖外部补丁。

### 3.3 `NEW-S11-01`（LOW，安全）：`:4133`「不可读 → `ErrAdmissionClosed`」与 canonical／ADR 未收敛

- 落点行号：`:4133`。
- 修改摘要：把旧句「文件被删除、截断、变为非 regular 或不可读，同样 `ErrAdmissionClosed`」替换为三分法：
  - 预检通过后的字节（含截断）／`generation`／inode／`st_dev`／挂载来源或挂载点漂移，未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked` → `ErrAdmissionClosed`；
  - 文件缺失、不可读、变为非 regular 或被符号链接替换 → `ErrAdmissionIncomplete`；
  - 同一快照内格式／签名／字段不一致 → `ErrAdmissionBinding`；
  - 并保留 policy `tool.receipt.trust_anchor_ref` 的 r72.1 唯一例外。
- 可机核判据（实测）：旧串 `同样 `ErrAdmissionClosed`` = 0；`文件缺失、不可读、变为非 regular 或被符号链接替换，返回 `ErrAdmissionIncomplete`` = 1。与 `:3944`／`:4377` 的 canonical 及 `framework.md:1118` 对齐。
- 闭合判定：**CLOSED（设计文本层）**。`:3944`／`:4133`／`:4377` 现为同一映射，`:3944` 的「唯一 canonical」声明不再有反例。跨制品 `ADR-S06-01` 仍由 ADR 写集处置（见第 8 节）。

### 3.4 r72.4（作者主动澄清，非 finding）：`:4051` 建议写集的累积口径

- 落点行号：`:4051`。
- 理由：`F017-R1` 裁决（只读，未落盘）结论为「仍开放、需本卡扩写集」，其依据是 r63.9.4／r64.12 建议表仍列 6 个生产文件与配套测试（`design.md:4000`／`:4005`／`:4006`／`:4011`／`:4012`／`:4015`／`:4016`／`:4019`／`:4275`），而 `:4051`／`:4069` 的「在…基础上」与 `:4072` 优先级规则只裁字面冲突、不废止增补项。为免后续误读「r65.12／r66.4 已整体取代 r63.9.4 表」，就地澄清为**累积口径**。
- 修改摘要：在既有两条「取代」括注后追加「（r72.4 澄清：…）」：r65.12／r66.4 只局部取代所列职责写法，不废止其余增补项——含 `recovery_restore_approval.go`、`recovery_restore_target.go`、`recovery_restore_receipt.go`、`recovery_restore_witness.go`、`recovery_config.go`、`internal/recovery/restore_source.go` 六个生产文件及配套 host／exec／approval／target／receipt／witness／config／restore_source 测试；是否并入本卡仍由 Task-Planner 决定。
- 边界：**不扩权**。未改 `tasks.md`、未改卡面、未勾选、未声称写集已扩展；`F017` 仍为 Task-Planner 对账项。
- 闭合判定：**登记完成（澄清性）**；不改变任何授权或任务状态。

## 4. 无回退核对

- 逐行 diff：仅第 2 节 12 行变化，其余 5,464 行逐字节同 r71 快照。
- 关键行未动（字节级）：`:4037`（唯一调用图）、`:4070` 除 B1 分类句外的既有内容、`:4076`（`waitFlow` 单点合同／reaper 单点／teardown 顺序）、`:4081`、`:4099`／`:4105`（golden vector）、`:4113`、`:4137`、`:4195`、`:4373`。
- 编号与语义：`V1`–`V11` 未新增、未删除、未重排（`V12` 计数 = 0）；R1–R8 未改。
- 已闭合项保持：strict v1／verify-v1、无 shell、私有 stdin、最小 ENV、Begin 前 closed、不得以 stdout／exit=0／DB 可连接自证、`injectedExclusive` 永不 `Close`、`Stop → flowBothDone → waitFlow → closeRestoreLease` 顺序、执行器不响应 ctx 时持租约 fail-closed、`NEW-R09-01`／`NEW-R09-02`／`NEW-S09-01`（`:4076`／`:4079` 未在变更集内）。
- 安全扫描计数（改前 → 改后）：`O_NOFOLLOW` 10 → 10；`fd-exec` 23 → 23；`restore_unknown` 38 → 45（+7，全部来自本轮新增的分类句，位置见第 2 节六行：`:3944`・`:4031`・`:4062`・`:4070`・`:4133`・`:4135`・`:4377` 各 +1）。
- 版本串：`identity-auth-design-ir01-r71` 残留 1 处，位于 `:4072` 的 r71 历史登记句，属历史事实，不回改（`:4072` 为登记行，本轮按规则追加 r72 段）；`identity-auth-design-ir01-r72` = 3（`:5`、`:38`、`:4072` r72 登记段）。

## 5. golden vector 复算（字节与 SHA256 均未变）

| 行 | 期望字节 | 实测字节 | 期望 SHA256 | 实测 SHA256 | 结论 |
|---|---|---|---|---|---|
| `:4099`（source） | 1,115 | 1,115 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 同 | 不变 |
| `:4105`（tool） | 2,357 | 2,357 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 同 | 不变 |

复算方法：取该行文本的 UTF-8 字节（不含行终止符），`SHA256` 小写十六进制。

## 6. 验证命令与结果

1. `[IO.File]::ReadAllBytes` → `B=1110477`；`ReadAllLines` → `Lines=5476`；`Get-FileHash` → `491A476E60636ADAA3F1610A07C78054ED4C9735F5AD42D60B76200A12C8414F`。
2. `CR=169`／`LF=5476`／`BOM=False`／`lastByte=10`（`[IO.File]::ReadAllBytes` 首三字节非 `EF BB BF`）。
3. 逐行 `-cne` 对快照：changed = `5,38,3944,4031,4051,4062,4070,4072,4083,4133,4135,4377`。
4. 逐行字节差合计 3,833 = `1110477 − 1106644`。
5. 全角括号 `U+FF08`/`U+FF09` = 1,258/1,258。
6. golden vector 字节 1,115／2,357，SHA256 与期望逐字节相等。
7. 断言的替换均为精确子串、写前自检「每锚点全文恰 1 次」；`r72 > r71 > …` 链 = 3（`:4072` 既有两处 + r72 登记段一处），旧链 `r71 > r70 > …` 加反引号形式 = 0。

## 7. NOT RUN（未执行，不得当 PASS）

- `go build`／`go vet`／单元测试／集成测试／`go test` 任何包：NOT RUN（本轮为设计文本修订）。
- 真实 restore、真实审批主体、真实外部见证（HSM／审批系统）：NOT RUN。
- Linux host／ACL／mount／loader／fd-exec／maps／network／witness 实跑：NOT RUN（`//go:build linux` skip 与无匹配测试 exit 0 均不等于 PASS）。
- Windows 运行时行为与平台测试：NOT RUN（仅设计文本层）。
- E2E、性能、故障注入：NOT RUN。
- ADR、`tasks.md`、代码、测试：本轮**未读改写**（ADR 仅只读引用 `:1118` 口径）。

## 8. 升级待决（不自行处置）

1. `F004`：ADR 同步事项仍待架构决策方；本轮只读引用 `framework.md:1118`，未改 ADR。
2. `F017`／`F017-R1`：任务面写集对账仍待 Task-Planner。r72.4 只澄清「建议写集为累积口径」，**不扩权**、不改 `tasks.md`（含 `:3235`／`:3241`–`:3245`）、不改卡面、不勾选。
3. `N005`（ADR-0031 状态段 vs `tasks.md` 状态段）：范围外，交架构决策方与 Task-Planner。
4. `ADR-S06-01`（LOW，ADR 写集）：`:4133` 的设计侧已按 r12 收敛，ADR 侧对应措辞仍由 ADR 修订处置，本轮不改 ADR。

## 9. 声明

本节只完成设计修订：不构成独立评审通过，不构成生产 restore 解锁，不构成 `enter`／`resume` 解锁，不构成约束 8 满足，不构成 `identity-auth-S4.24a` 可关闭；不勾选 `tasks.md` 任何复选框。`NEW-R11-01`／`NEW-R11-02`／`NEW-S11-01` 的「CLOSED」仅指**设计文本层已按建议值收敛、可唯一机核**，最终判定必须由两路独立复审重新给出。