# SEC r11 独立复审报告 —— identity-auth 详细设计 r71

- 日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：`.add\project\knowledge\design-workspace\identity-auth\design.md`
  - 实测：**1,106,644 B / 5,476 行 / CR=169 / LF=5,476 / 无 BOM / 末字节 `0A`**
  - SHA256：**`168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5`**
  - 版本：`identity-auth-design-ir01-r71`（`:5` 与 `:38` 一致）
  - 冻结快照：`C:\Users\J00010~1\AppData\Local\Temp\design-r71-freeze-168B614D.md`，同 1,106,644 B / 同哈希（已核；现场与快照一致，无漂移）
- 写前基线：`.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r11\design.r70.baseline.md`
  - 1,103,152 B / 5,476 行 / SHA256 `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`（已核）
- 基准报告：`.spec\wip\recovery-runtime\evidence-b13j-s424a\review-sec-r10\report.md`
  - 23,423 B / 168 行 / SHA256 `4D8625127035B84EEAF595BAFABC2666B9BFA64382FE43B45B1FF0D4DB9EF094`（Verdict: PASS；BLOCKER 0 / MEDIUM 0 / LOW 0）
- 架构基线：`.spec\wip\recovery-runtime\evidence-b13j-s424a\review-arch-r10\report.md`
  - 16,624 B / 150 行 / SHA256 `3E6639A5B566E3FAC644D4368F4CC33526163EB63DFF0F49343978413A30455A`（PASS 0/0/0；范围外 `N005`）
- 作者材料（只作线索，不采信数值）：`design-revision-r11\report.md` 15,016 B / `8A3AEACB0A475875E57EC2EDDC81A801BCD6AB8DE5E78F5F5F26E45BCE57CF54`；`design-revision-r11\hashes.txt` 4,942 B / `5A493A856EEACF004D51D6494E30A183004C63EAD45577B0C8FF974F35E1AC55`

## 0 方法、检查范围与只读声明

方法：按 `add-reviewer` / `add-code-review`（objectType=design）对冻结 r71 做安全侧独立复审。第一轮用原始字节（`[IO.File]::ReadAllBytes`）冻结对象与 r70 基线，按 LF 切行做 Ordinal 比较，确认内容改动边界、CR/LF、BOM、末字节和字节账；第二轮判读变更行及相邻合同，沿 `:3944`／`:4031`／`:4037`／`:4062`／`:4070`／`:4072`／`:4083`／`:4133`／`:4377` 与 ADR-0031 r12 `framework.md:1118` 交叉核对，并独立复算 `:4099`／`:4105` 两条 golden vector、六值角色枚举和关键安全字面。作者报告、作者 `hashes.txt` 与既有报告只用作线索，不用于取值。

两轮实际检查范围：

1. 冻结与字节账：比对 `design.md` 与 `design.r70.baseline.md` 的全部行；确认恰 10 行不同且零增删；计算每行 UTF-8 字节增量、全文哈希、行数、CR/LF、BOM、首末字节；建立并核对冻结快照。
2. 设计与 ADR 判读：读取 `:5`、`:38`、`:3944`、`:4031`、`:4037`、`:4062`、`:4070`、`:4072`、`:4083`、`:4099`、`:4105`、`:4133`、`:4377`；交叉读取 `framework.md:1082`／`:1084`／`:1118`／`:1174`／`:1192`／`:1193`／`:1194`；只读 `tasks.md:3186`／`:3220` 与 `decisions\README.md:39`。未把 ADR 复审（`review-sec-r06`）混同为本设计的 r71 复审：`.spec\wip\recovery-runtime\evidence-b13j-s424a\review-arch-r06`／`review-sec-r06` 属 design r66 旧评审；ADR 接受报告在 `.spec\wip\recovery-runtime\evidence-b13j-d06\review-arch-r06`／`review-sec-r06`。
3. 安全面扫描：`若该键存在`、`tool.receipt.trust_anchor_ref`、`V12`、`close(a.flowBothDone)`、`各起`、`restore_unknown`、`no-follow`、`fd-exec`、`首期不提供生产 restore 工具与生产批准主体`、`模拟见证`、`Windows 生产 unsupported`，以及 `:4061` 六值角色枚举、`:4076` 双路 join 屏障、`:4195` 信任根长名唯一字面、`:4373` generation 三元。
4. 独立复算：`:4099` source vector 与 `:4105` tool vector 的字节数和 SHA256；本轮未运行任何构建、测试、E2E、Linux host、ACL、mount、loader、fd-exec、maps、network、witness 或真实 restore。

判读过的文件（均为只读）：

- `.add\project\knowledge\design-workspace\identity-auth\design.md`（冻结快照 `design-r71-freeze-168B614D.md`）
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r11\design.r70.baseline.md`
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\review-sec-r10\report.md`
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\review-arch-r10\report.md`
- `.spec\wip\recovery-runtime\evidence-b13j-s424a\design-revision-r11\report.md`、`hashes.txt`（仅作线索）
- `.add\project\knowledge\decisions\framework.md`（ADR-0031 r12 判读段）
- `.add\project\knowledge\tasks\tasks.md`（`:3186`／`:3220`）
- `.add\project\knowledge\decisions\README.md`（`:39`，仅范围外观察）
- `.codex\skills\add-code-review\SKILL.md`、`.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\skills\add-code-review\references\产品安全红线.md`（方法依据）

只读声明：本轮唯一写入是本目录的 `report.md`（以及写后生成的 `hashes.txt`）；未改 `design.md`、`tasks.md`、`framework.md`、`decisions\README.md`、`cmd/**`、`internal/**`、测试、既有 `review-*`／`design-revision-*`；未删除、移动、归档任何文件；未执行任何 Git 写操作。现场对象与自建快照同哈希，无漂移。

## 1 结论

**Verdict: PASS WITH CONDITIONS**

| 类别 | 计数 | 说明 |
|---|---:|---|
| BLOCKER | 0 | — |
| MEDIUM | 0 | — |
| LOW | 1 | `NEW-S11-01`：`:4133` 与 `:3944`／`:4377`／ADR `:1118` 的不可读错误码仍未完全收敛；仅错误码／审计枚举分叉，不产生成功路径 |
| 非阻断观察（不计入计数） | 2 | `tasks.md` 状态登记漂移；ADR 状态段与登记文书的时点差异 |
| NOT RUN（不计入计数） | 1 | `SEC-F007` 及全部 Linux／真实 restore 能力 |

开放项清单：`NEW-S11-01`（LOW）。本轮已闭合的四项中，前三项（policy 必填、`:4070` 门禁与路径分类、V7/V8 重验登记）判定 **CLOSED**；第四项（不可读分类收敛）判定 **PARTIAL**，故本报告不把“唯一 canonical、全文无两套映射”写成已完全满足。该残留不影响 fail-closed、Begin 前拒绝、工具启动点后的 `restore_unknown`，也不构成生产解锁或任何实施授权。

## 2 r71 变更边界与字节账（独立复算）

以 r70 基线和新快照原始字节按 LF 切行比较：

- 行数：基线 5,476 行，新对象 5,476 行；**零增删**。
- 内容不同的行：恰 **10 行**——`:5`、`:38`、`:3944`、`:4031`、`:4037`、`:4062`、`:4070`、`:4072`、`:4083`、`:4377`；其余行逐字节相同。
- 全文增量：**+3,492 B**；1,103,152 + 3,492 = **1,106,644 B**，账自洽。

| 行 | r70 行字节 | r71 行字节 | 增量 |
|---|---:|---:|---:|
| `:5` | 39 | 39 | 0 |
| `:38` | 50 | 50 | 0 |
| `:3944` | 545 | 1,370 | +825 |
| `:4031` | 1,831 | 2,006 | +175 |
| `:4037` | 2,531 | 2,796 | +265 |
| `:4062` | 2,234 | 2,655 | +421 |
| `:4070` | 5,207 | 5,858 | +651 |
| `:4072` | 8,022 | 8,952 | +930 |
| `:4083` | 596 | 681 | +85 |
| `:4377` | 861 | 1,001 | +140 |
| **合计** |  |  | **+3,492** |

结论：r71 的改动边界与 Lead 给出的 10 行、零增删、+3,492 B 一致；没有隐藏的全文重写、行插入或行删除。

## 3 四项修正逐条判定

### 3.1 policy `anchor_ref` 必填化（`:4031`、`:4062`）—— CLOSED

- `:4031` 明写批准记录 policy **必填**的 `tool.receipt.trust_anchor_ref`，并列出五处逐字节等式：`receipt_anchors.anchor_ref`、policy 的 `tool.receipt.trust_anchor_ref`、回执 `trust_anchor_ref`、审计记录 `anchor_ref`、见证 head `anchor_ref`。
- `:4031` 与 `:4062` 均写明：policy 侧缺键、空串或字段名不符即不得放行；启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`。原先“若该键存在”的条件分支已删除。
- 全文 `若该键存在` 仅 **2 处**，分别在 `:4031` 与 `:4062` 的变更说明括注中，语义是“删除旧条件分支”的历史说明，不是存活的分支条件；不是可选的 policy 字段。
- `tool.receipt.trust_anchor_ref` 全文 **3 处**，均为必填长名；`:4195` 的信任根配置长名仍是唯一合法字面，短名出现即 `ErrAdmissionBinding`，不作别名。policy 字段与配置键不是同一个命名空间，但本项要求的“policy 侧唯一可接受字段名”已明确。
- 与 `framework.md:1118` 的 policy 必填、五处等式、缺键／空串／名不符即 closed 对齐。

判定：**CLOSED**；不存在“policy 该键可缺”的存活读法。

### 3.2 `:4070` Begin 门禁与错误分类按路径更正 —— CLOSED

- 现文只把启动预检与 `P3`／`P6` 放在 `Journal.Begin` 之前：`P3`／`P6` 漂移 → `ErrAdmissionClosed` 且不进入 `Journal.Begin`。
- `R1`／`R2`／`R4` 明确属于 `enter` 的 `Replace` 路径，位于同一 `Journal.Begin` 之后；`V1`／`V2`／`V4` 属于 `VerifyRestored`，`enter` 侧在 Begin 之后，`resume` 侧没有新 Begin 且在 `Prepare` 之后。上述点失败按 `:4377` 的原因分类，不再统一写成 Begin 前 `Closed`。
- 越过工具启动点后的 `R6`／`V6`／`V7`／`V8`／`V10` 一律按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown` 处置；任何情况下不得推进 `sequence`、不得当成功。
- 全文扫描未发现旧错写“R/V 漂移 → `ErrAdmissionClosed` 且不进入 `Journal.Begin`”；“不进入 `Journal.Begin`”只保留给 `P6` 前路径。

判定：**CLOSED**；路径顺序与错误分类已按设计事实更正，未把 R/V 写成 Begin 前拒绝。

### 3.3 `V7`／`V8` 前重验登记（`:4037`、`:4062`）—— CLOSED

- `:4037` 在完整 V 图中登记：`V7` 从 `independent_read_path` 独立读取回执字节之前执行一次 `revalidateApproval` 类重验；`V8` 验证 producer 签名之前再执行一次；两次重验“只增加拒绝点、不放宽既有检查”，失败按 `errors.Join(ErrRecoveryWriteUnknown, err)`／`restore_unknown`。
- `:4062` 对 `V7`／`V8` 作同样登记，并保留 `V9` 的 `sequence` 严格比较和 `V10` 的推进前重验；`V1`–`V11` 的编号集合未变。
- 全文 `V12` = **0**，没有新增、重排或复用 V 编号；`R1`–`R8` 未变。新增两次重验只会增加拒绝点，没有删除既有检查。

判定：**CLOSED**；只增加拒绝点，不放宽既有检查。

### 3.4 不可读错误分类收敛（`:3944`、`:4377`；与 `:4133`）—— PARTIAL

- `:4377` 已把旧“预检后字节／generation 漂移 → `ErrAdmissionBinding`”改为 `ErrAdmissionClosed`，并声明本条是唯一 canonical、`:3944` 按本条读取。
- `:3944` 已把旧“批准文件、信任根或目标 DSN 不可读 → `ErrAdmissionClosed`”作废，改为按原因映射：缺键、文件缺失或不可读 → `ErrAdmissionIncomplete`；同一快照内格式／签名／字段不一致 → `ErrAdmissionBinding`；有效期、撤销、预检后漂移 → `ErrAdmissionClosed`；平台不支持 → `ErrAdmissionUnsupported`。同时保留“任一失败在返回成功前发生”“`restore_unknown` 只能写 `unknown`”。
- **但仍有残余字面分叉**：`:3944` 同时写“`revalidateApproval` 对预检快照的漂移／替换判定（`:4133`）……不属本映射”；而未改的 `:4133` 仍写“文件被删除、截断、变为非 regular 或不可读，同样 `ErrAdmissionClosed`”。`Prepare`／`P6` 正是通过 `revalidateApproval` 读取（`:4135` 第二项；`:4070` 的 `P6`），因此 `Prepare` 路径的“不可读”在 `:3944`（`ErrAdmissionIncomplete`）与 `:4133`（`ErrAdmissionClosed`）之间仍有两套字面。ADR `framework.md:1118` 对 `P6`／`Prepare` 明确写缺键、文件缺失或不可读 → `ErrAdmissionIncomplete`，因此这也是设计—ADR 的对齐缺口。
- 这是**收敛不完整**，不是放宽：两个错误都非 nil；`P6`／`Prepare` 都在 `Journal.Begin` 前；不写 DB／CAS，不启动工具，不推进 `sequence`；公共面只在 `:1537` 映射为现有 `platform.recovery_in_progress`。没有“不可读被当成已授权”的路径。影响限于内部错误码、审计枚举和负向测试期望的差异。
- 修复建议：把 `:4133` 的“删除／截断／不可读 → `ErrAdmissionClosed`”拆开：缺键／缺失／不可读 → `ErrAdmissionIncomplete`，仅把字节／generation／inode／挂载漂移与撤销／有效期越界 → `ErrAdmissionClosed`；或者删除 `:3944` 的“唯一 canonical”表述并明确路径例外。前者更能保持 ADR r12 的单一映射。

判定：**PARTIAL**；见 `NEW-S11-01`（LOW）。因此本报告不声称“全文不存在两套并存映射”。
## 4 fail-closed 逐路径复核

| 路径／检查点 | `Journal.Begin` 位置 | 失败分类（现行文字） | 结果约束 | 判定 |
|---|---|---|---|---|
| 启动预检、`P3` | Begin 前 | 缺键／缺失／不可读 → `ErrAdmissionIncomplete`；同快照不一致 → `ErrAdmissionBinding`；有效期／撤销／漂移 → `ErrAdmissionClosed`；平台不支持 → `ErrAdmissionUnsupported` | 不生成可执行计划，不进入 Begin，不写 DB／CAS | fail-closed 保持 |
| `P6`／`Prepare` | Begin 前 | `:3944`／`:4377` canonical：缺键／缺失／不可读 → `ErrAdmissionIncomplete`；`:4133` 旧路径文字仍把不可读写为 `ErrAdmissionClosed` | 不进入 Begin，不写 DB；无成功返回 | **错误码残余分叉（NEW-S11-01）**，但无 fail-open |
| `enter` 的 `R1`／`R2`／`R4` | Begin 后 | 按 `:4377` 原因分类：缺键／不可读 → Incomplete；同快照不一致 → Binding；有效期／撤销／未越过工具启动点的漂移 → Closed | 不把失败当成功；未越过工具启动点时不得执行后续阶段 | fail-closed 保持 |
| `enter`／`resume` 的 `V1`／`V2`／`V4` | `enter` 在 Begin 后；`resume` 无新 Begin，在 `Prepare` 后 | 同上，按路径既有原因分类 | 不返回成功；不跳过 `prepared`／重验 | fail-closed 保持 |
| 越过工具启动点：`R6`／`V6`／`V7`／`V8`／`V10` | 已在启动点之后 | 一律 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`，调用方按 `restore_unknown` 处置 | 不得推进见证 `sequence`；不得当成功；审计结果只能是 `unknown`（`restore_unknown` 不得写 `verified`） | fail-closed 保持 |
| `resume` 调用 `VerifyRestored` | 无新 Begin；`Prepare` 之后 | `V7`／`V8` 前新增重验；失败按已启动路径 `restore_unknown` | 不调用 `Replace`；不新 Begin；不返回成功 | fail-closed 保持 |
| `V10` 推进见证前 | 工具已启动 | `revalidateApproval` 失败 → `errors.Join(ErrRecoveryWriteUnknown, err)` | **不得推进 head `sequence`**；不得把 unknown 写成成功 | fail-closed 保持 |

结论：除 `NEW-S11-01` 的内部错误码分叉外，所有路径仍在返回成功之前拒绝；`P6` 失败不进入 Begin；越过工具启动点后的失败保留 unknown；没有任何“不可读即已授权”的读法。

## 5 与 `framework.md:1118`（ADR-0031 r12）对齐核对

ADR 对象实测：`.add\project\knowledge\decisions\framework.md`，297,412 B / 1,275 行 / SHA256 `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`；`:1082` 为 `adr-0031-r12`，`:1118` 是本项判读段。

| 安全判据 | design r71 | ADR r12 `:1118` | 对齐 |
|---|---|---|---|
| policy 必填 `tool.receipt.trust_anchor_ref` | `:4031`／`:4062` 明确必填；缺键／空串／名不符不得放行 | 明写 policy 必填，缺键／空串／名不符即 closed | 一致 |
| `anchor_ref` 五处逐字节相等 | `:4031`／`:4062` 列出五处 | 同列五处 | 一致 |
| `P6`／`Prepare` 失败分类 | `:4377` canonical + `:3944` 取代说明；但 `:4133` 残留“不可读 → Closed” | 缺键／文件缺失／不可读 → `ErrAdmissionIncomplete`；有效期／撤销／漂移 → `ErrAdmissionClosed` | **不一致（NEW-S11-01）** |
| `enter` 的 `P6` 不进入 Begin | `:4070`／`:4037` | 明写不进入 `Journal.Begin` | 一致 |
| `resume` 不新 Begin、不调 `Replace` | `:4070` | 明写 `resume` 无新 Begin、不调 `Replace` | 一致 |
| 越过工具启动点后 `restore_unknown` | `:4070`、`:4062`、`:4037` | 明写 `errors.Join`／`restore_unknown` | 一致 |
| 不得推进见证 `sequence` | `:3944`、`:4031`、`:4062`、`:4377` | 明写任何情况下不得推进 | 一致 |
| 三签席位与密钥来源 | `:3988`／`:4031`／`:4061` | producer／verifier／witness 三签、三处公钥来源 | 一致（设计行未被 r71 改写） |
| `generation` 三元 | `:4373` | `approval_generation`／`trust_root_generation`／`anchor_generation` | 一致（设计行未被 r71 改写） |
| 不声称抗 root | `:4070` 第(2)项 | 无条件残余、不得以签名／见证暗示抗 root | 一致 |
| 两种见证分域 | design 相关历史条款未被改 | ADR `:1118` 末段明确分域 | 无新冲突 |

说明：`framework.md:1084` 仍写“`adr-0031-r12` 的两路接受尚未取得”，而 `tasks.md:3220`／`decisions\README.md:39` 记录 d06 两路接受已登记；这是状态段与登记文书的时点／维护差异，属范围外观察，不作为本设计 r71 的通过依据，也不声称 `N005` 已解决。

## 6 无回退底线逐项核对

r71 只有 10 行不同、其余全文逐字节等于 r70，因此“未被修改的行”没有回退；对变更行也逐项核对了以下底线：

- **fail-closed 全路径**：`P3`／`P6` 在 Begin 前拒绝；`R`／`V` 失败不返回成功；启动点后 `restore_unknown`。保持。
- **Begin 前非 nil 返回**：`:3944`／`:4377` 明写任一非 nil 在 Begin 前返回；`:4070` 明写 `P6` 不进入 Begin。保持。
- **三签与三席位密钥来源**：producer 用 `receipt_anchors.public_key`；verifier 用 `keys[verifier_key_id].public_key` 且 `principal_id` 相等；witness 用 `receipt_anchors.witness_public_key`／`witness_key_id` 仅作标识。相关行未被 r71 改动。保持。
- **`anchor_ref` 五处逐字节相等**：`:4031`／`:4062` 现文明确；policy 缺键不是可缺分支。保持并加强。
- **`generation` 三元**：`:4373` 的 approval／trust_root／anchor 三元摘要定义与旧句“漂移即 closed”保持；r71 未改该行。保持。
- **`receipt_anchors` 无 `revocation`／`valid_until`**：相关 keyset 条款未被 r71 改写，未引入用不存在字段判断撤销的路径。保持。
- **no-follow／fd-exec／最小 ENV／`restore-v1`／`verify-v1`／`restore_unknown`／Windows unsupported**：这些合同不在 10 个变更行内，按逐行比较保持 r70 字节；关键扫描仍可见 `no-follow` 18 处、`fd-exec` 23 处、`restore_unknown` 38 处、`Windows 生产 unsupported` 5 处。保持。
- **“首期不提供生产 restore 工具与生产批准主体”**：`:4083` 原声明未被改写（只追加 r71 沿用句），全文该结论 8 处；r71 未解锁生产工具或批准主体。保持。
- **r70 已闭合项**：`:4076` 未被 r71 改动；reaper 仍只由 `start` 起 1 个、两条业务 goroutine 只 `defer a.flowWG.Done()`、`close(a.flowBothDone)` 的规范关闭点仍唯一且在 `flowWG.Wait()` 之后。保持。
- **golden vector**：`:4099`／`:4105` 未被改动，独立复算仍为规定值（见第 7 节）。保持。
- **审计 `result` 枚举与 unknown**：`:3944` 保留 `verified`／`unknown`／`closed`／`incomplete`，明写 `restore_unknown` 只能写 `unknown`；无回退。
- **错误／日志不含 raw secret**：r71 的 10 个变更行没有新增 secret 读取、日志、错误回显或 wire 字段；`enter` 唯一 secret 取得路径位于未改的 `:4076`，`RecoveryInput.Secret` 与 `defer clear` 合同未被本轮改写。保持。
- **六值角色封闭枚举**：`:4061` 未被 r71 改写，仍只允许 `restore_approver`／`restore_reviewer`／`restore_executor`／`restore_producer`／`restore_verifier`／`restore_witness`；裸名映射仍仅经显式映射写入 `appointments`。保持。

## 7 golden vector 与角色字面独立复算

方法：在冻结快照上取 `:4099` 与 `:4105` 的完整行文本，以 UTF-8 编码为字节，计算字节数和 SHA-256；不重新序列化 JSON，不依赖作者给出的值。

| 向量 | 行 | 字节 | SHA256 | 结果 |
|---|---|---:|---|---|
| source | `:4099` | **1,115** | **`de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab`** | 与冻结期望一致 |
| tool | `:4105` | **2,357** | **`8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508`** | 与冻结期望一致 |

两条向量均为 ASCII 文本，字节数与哈希独立复算通过。`:4061` 的六值封闭枚举未被改；source 向量使用 `restore_approver`／`restore_reviewer`，tool 向量使用 `restore_executor`／`restore_approver`，均在六值枚举内；未出现裸角色名替代 `appointments.role_name` 的写法。

## 8 Finding

### `NEW-S11-01`（LOW）：`:4133` 的不可读错误码仍与 canonical／ADR 不一致

- **位置**：`design.md:4133`；相邻 canonical 为 `:3944`、`:4377`；路径依据为 `:4135` 的 `Prepare` 读取点和 `:4070` 的 `P6` 门禁；ADR 对齐依据为 `framework.md:1118`。
- **事实**：`:3944` 说“启动预检与 `Prepare` 内的读取”按唯一 canonical 映射，缺键、文件缺失或不可读 → `ErrAdmissionIncomplete`；但同一句又把 `:4133` 的漂移／替换判定排除在映射外。`:4133` 未改，仍写“文件被删除、截断、变为非 regular 或不可读，同样 `ErrAdmissionClosed`”。`Prepare`／`P6` 通过 `revalidateApproval` 读取该文件，因此同一输入存在 Incomplete／Closed 两种文字结论。ADR `:1118` 对同一路径要求 Incomplete／Closed 按原因区分，其中不可读为 Incomplete。
- **触发场景**：`Prepare` 或 `P6` 时批准文件、信任根或目标 DSN 被删除、截断、变为非 regular 或不可读；实现者按 `:4133` 返回 Closed，或按 `:3944` 返回 Incomplete，负向测试与审计枚举可能分叉。
- **影响**：不产生成功路径；两个错误都非 nil，均发生在 Begin 前（或作为 `Prepare` 拒绝），不写 DB／CAS、不启动工具、不推进 `sequence`。公共错误面只在 `:1537` 映射为 `platform.recovery_in_progress`。因此这是错误码与“唯一 canonical”声明的一致性问题，不是 fail-open，也不构成 BLOCKER／MEDIUM。
- **修复建议**：把 `:4133` 改为“缺键／缺失／不可读 → `ErrAdmissionIncomplete`；字节／generation／inode／挂载漂移及撤销／有效期越界 → `ErrAdmissionClosed`”，或者删去 `:3944` 的“唯一 canonical”绝对措辞并显式登记路径例外。建议采用前者，使设计与 ADR r12 单一映射一致。
- **复审条件**：`:4133` 与 `:3944`／`:4377`／ADR `:1118` 的文字一致后方可关闭；不得以本 finding 为理由修改 ADR 或 `tasks.md`。

## 9 携带项、范围外登记与观察

- **携带项**：ADR 侧 `ADR-S06-01`（LOW，ADR 写集）在 ADR 接受报告中标为 OPEN；其根因是错误码一致性。本设计 r71 已在 `:4377`／`:3944` 建立 canonical，但 `:4133` 仍使其未完全消除；本报告只登记该跨制品关联，不据此判 ADR 或本设计为 BLOCKER，也不代改 ADR。
- **范围外观察 1**：`.add\project\knowledge\tasks\tasks.md` 实测 904,893 B / SHA256 `696A5B0EE9B2E4DA75D4C50EB82BCFC0E68D657500BDED346F4CA765822BD28E`；`:3186` 仍为 `- [ ] identity-auth-S4.24a ...`，**未勾选**，符合边界。该文件的本次哈希与旧记录 904,142 B／`CA20577B…` 不同，按并发编辑观察登记；本轮未写该文件。
- **范围外观察 2**：`tasks.md:3220` 已登记 ADR-0031 现行 r12 与 d06 两路接受，并标注 r71 正在修订四项；它不替代本设计复审，也不构成实施派发。
- **范围外观察 3**：`decisions\README.md:39` 已登记 r12 两路接受；与 `framework.md:1084` 的“尚未取得”存在维护时点差异，属架构／登记范围；不据此改变本设计 verdict，不声称 `N005` 已解决。
- **范围外观察 4**：`framework.md` 中 ADR r12 的两路接受事实与 `tasks.md:3220` 的接受登记一致；本报告只读，不修改 ADR、`tasks.md` 或 `README.md`。

## 10 NOT RUN 与边界

- `SEC-F007` 仍 **NOT RUN**：Linux host／ACL／mount／loader／fd-exec／maps／network／witness 实跑条件未具备，不因设计文字存在改判 PASS。
- 构建、单元测试、集成测试、E2E、真实 restore、真实备份、真实批准主体、专用执行身份、POSIX 宿主隔离、进程树 reap、独立 witness、OpenPGP／Ed25519 实机验签均 **NOT RUN**。
- 本轮只判设计文本层安全充分性；不声称生产 restore 解锁、`enter`／`resume` 解锁、约束 8 满足、`identity-auth-S4.24a` 可关闭或可派发、任何安全红线解除、抗 root 成立或四项 unknown 已补齐。
- `N005` 只作范围外登记，不据此判 CHANGES REQUIRED，不代改任何文件。

## 11 实际执行命令（代表性）

```powershell
$p='.add\project\knowledge\design-workspace\identity-auth\design.md'
[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)
[IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $p))
Get-FileHash -Algorithm SHA256 -LiteralPath $p
[Text.Encoding]::UTF8.GetBytes($line)
[Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
```

另以 `[IO.File]::ReadAllBytes` 对 r70 基线／r71 快照做 LF 切行、Ordinal 比较和每行 UTF-8 字节计数；以 `Get-FileHash` 核对 `design.md`、基线、`review-sec-r10`、`review-arch-r10`、`framework.md`、`tasks.md` 和作者材料。未运行任何写入被审对象的命令。

## 12 自哈希记录与声明

本报告是自指对象，无法在正文内嵌入最终自身的 SHA256（写入该值会改变文件内容并改变哈希）。写后实测的最终字节数、行数和 SHA256 已固化在同目录 `hashes.txt` 的 `[report.md]` 节，并在本轮最终回报中给出；`hashes.txt` 同时记录被审对象、快照、基线、基准报告和 ADR 的字节与摘要。

本报告只完成对冻结 `identity-auth-design-ir01-r71` 的独立安全侧设计复审，不构成生产解锁、`enter`／`resume` 解锁、约束 8 满足、`N005` 已解决、`identity-auth-S4.24a` 可关闭或已交付；`SEC-F007` 及全部 Linux／真实能力仍为 NOT RUN。任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。最终接受由 Lead／用户决定。
（本文件为自指对象：最终字节数、行数与 SHA256 已实测并记录在同目录 `hashes.txt` 的 `[report.md]` 节；正文无法内嵌最终自身摘要。）

Verdict: PASS WITH CONDITIONS
