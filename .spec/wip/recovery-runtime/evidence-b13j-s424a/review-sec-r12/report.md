# identity-auth 详细设计 r72 安全独立复审报告

- 日期：2026-11-02
- 角色：add-reviewer（安全轴，独立 Reviewer）；objectType = design
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 被审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 修订号：`identity-auth-design-ir01-r72`（`:5`、`:38`）

## 0. 冻结值与完整性（复审开始与结束各复算一次）

| 项 | 冻结值 | 本次实测 | 判定 |
|---|---|---|---|
| 字节 | 1,110,477 | 1,110,477 | 一致 |
| 行数 | 5,476 | 5,476 | 一致 |
| CR / LF | 169 / 5,476 | 169 / 5,476 | 一致 |
| BOM | 无 | 前 3 字节 `2D 2D 2D`（`---`），无 BOM | 一致 |
| 末字节 | `0x0A` | `0x0A` | 一致 |
| SHA256 | `491A476E60636ADAA3F1610A07C78054ED4C9735F5AD42D60B76200A12C8414F` | 同 | 一致 |

- 复审开始与结束两次 `Get-FileHash -Algorithm SHA256` 结果相同，现场无漂移；本轮所有判读基于现场对象（与给定冻结值逐字节一致），因此未另建 `%TEMP%` 快照，不存在“快照与现场不同”的判读分歧。
- r71 基线 `design-revision-r12\design.r71.baseline.md`：1,106,644 B / 5,476 行 / `168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5`（自算一致）。
- 作者证据（仅作线索，未采信）：`design-revision-r12\report.md` 13,295 B / `677854EE758D1AFD680E81753817E977BBFAA342A67E3277F7FE3E367029A65A`；`design-revision-r12\hashes.txt` 4,662 B / `F1A4112240FE22EE7684EE64CF1A7479945D90B93EDD3D57B543C560A231C146`。

## 1. 方法

- 完整读取 `.codex\skills\add-code-review\SKILL.md`，按 `objectType=design` 分支执行；按加载规则读取 `.codex\guidance\code-review-checklist.md`、`.codex\guidance\abstraction-principles.md`、`.codex\skills\add-code-review\references\产品安全红线.md`。
- 两轮检查：
  - 第一轮：r72 改动边界独立复算（12 行、0 增删、+3,833 B、行尾风格逐行一致），再逐行判读 12 行新文本的直接安全缺陷。
  - 第二轮：沿真实消费链核 fail-closed——批准记录读取 → 启动预检 → `P6`／`Prepare` → `R1`／`R2`／`R4`、`V1`／`V2`／`V4` → 工具启动点 → `R6`／`V6`／`V7`／`V8`／`V10` → 审计与见证 `sequence` 推进；并核 `anchor_ref` 五处等式、六值角色封闭枚举、错误分类唯一 canonical。
- 期间不启动其他 Agent，不执行 build/test/E2E，不做真实 restore。

## 2. 只读声明与判读文件

只读复审；除本目录 `report.md` 与 `hashes.txt` 外未写任何文件；未修改 `design.md`／`framework.md`／`tasks.md`／代码／测试／既有 `review-*`／`design-revision-*`；未删除、移动、归档；未执行任何 Git 写操作。

判读过的文件：

- `.add/project/knowledge/design-workspace/identity-auth/design.md`（全文按行读取 + 定点 regex，关键行 `:5`／`:38`／`:1481`／`:3936`／`:3944`／`:4031`／`:4037`／`:4051`／`:4061`／`:4062`／`:4070`／`:4072`／`:4076`／`:4079`／`:4083`／`:4099`／`:4105`／`:4133`／`:4135`／`:4377`）
- `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r12/design.r71.baseline.md`
- `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r12/report.md`、`hashes.txt`
- `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r11/report.md`（26,758 B / `D715E4116CAE3EDEAA6785A6E1FCF4BD5A15E07E036CA78923976A7B9D23FAFB`，Verdict PASS WITH CONDITIONS 0/0/1）
- `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r11/report.md`（21,990 B / `3D994EC464D5798592C95625C1B8E59C1FAA7338BC0B7C831C9BAF2A068262F2`，Verdict CHANGES REQUIRED 0/1/1）
- `.add/project/knowledge/decisions/framework.md`（297,412 B / 1,275 行 / `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`，`adr-0031-r12`；判读 `:1082`、`:1118`）
- `.add/project/knowledge/tasks/tasks.md`（904,893 B / 3,312 行 / `696A5B0EE9B2E4DA75D4C50EB82BCFC0E68D657500BDED346F4CA765822BD28E`；判读 `:3186`）
- 上述四份方法文件

## 3. r72 改动边界独立复算

方法：`[IO.File]::ReadAllLines` 逐行比对 r71 基线，按 UTF-8 字节数计算每行增量；再逐行比对行尾风格（CRLF/LF）与 CR/LF/BOM。

| 行 | r71 字节 | r72 字节 | 增量 |
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
| 合计 | — | — | **+3,833** |

核对结论：

- 恰 12 行内容不同，行数 5,476 → 5,476（零增删）；全文增量 1,110,477 − 1,106,644 = **+3,833 B**，与逐行增量之和相等。
- 其余 5,464 行内容与 r71 基线逐字节相同；逐行行尾风格（CRLF/LF）差异为 **0 行**（两份文件均 169 行 CRLF），故不存在“借换行变化制造隐藏差异”。
- 全角括号平衡 1,258 / 1,258；`:4072` 优先级链由 `r71 > r70 > …` 扩为 `r72 > r71 > … > r61`。
- 作者自述（仅线索）与本轮独立复算的 12 行集合完全一致；**结论不依赖作者声明**。

## 4. 原 finding 逐条判定

### 4.1 `NEW-S11-01`（LOW，安全侧）：CLOSED

原问题：`:4133` 仍把“批准文件被删除／截断／非 regular／不可读”统一写成 `ErrAdmissionClosed`，与 `:3944`／`:4377`／ADR `:1118` 的“缺键／不可读 → `ErrAdmissionIncomplete`”分叉。

r72 事实：

- `:4133` 已改为三分法：预检后**字节（含截断）**、`generation`、inode、`st_dev`、挂载来源／挂载点漂移、未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked` → `ErrAdmissionClosed`；**文件缺失、不可读、变为非 regular 或被符号链接替换** → `ErrAdmissionIncomplete`；同一快照内格式／签名／字段不一致 → `ErrAdmissionBinding`；policy `tool.receipt.trust_anchor_ref` 缺键／空串／名不符按 r72.1 例外。
- 全文 `同样 \`ErrAdmissionClosed\`` 出现 **0 次**（旧串已消失），不再存在“不可读 → Closed”的存活表述。
- 截断被判为 `Closed` 而非 `Incomplete`，与 `:4377`“预检通过后的字节／`generation` 漂移（含 mtime／size／inode 变化）→ `ErrAdmissionClosed`”同口径：同一 inode 上内容长度变化属“预检后字节漂移”，不是“读不到”。两处无矛盾。

判定：**CLOSED**。仅错误码收敛，不改变任何拒绝语义（见 §6 fail-closed 表）。

### 4.2 `NEW-R11-01`（MEDIUM，架构侧）：安全视角 CLOSED

原问题：`policy` 缺 `tool.receipt.trust_anchor_ref` 时，设计 canonical（`:3944`／`:4377`）判 `ErrAdmissionIncomplete`，ADR-0031 r12 `:1118` 判启动前 `ErrAdmissionBinding`，同一输入两类闭码。

r72 事实（五处同步，均可机核）：

- `:3944`：新增“**唯一 policy 专属例外（r72.1）**”，写明该字段缺键／空串／名不符不在“缺键 → `ErrAdmissionIncomplete`”映射内，按启动前 `ErrAdmissionBinding`、工具已启动后 `restore_unknown` 分类，**只限此字段**，不得扩展到任何其他缺键，且不改变本条 fail-closed 与“任一非 nil 都在 Begin 前返回”。
- `:4031`／`:4062`：同一例外句 + “其余四处任一不等 → 启动前 `ErrAdmissionBinding`／已启动后 `restore_unknown`”；并保留“删除「若该键存在」条件分支，不保留「policy 该键可缺」的读法”。
- `:4070`：B1 摘要同步写“**唯一例外（r72.1）**……只限该字段”。
- `:4133`／`:4377`：`revalidateApproval` 与 canonical 映射同步引用 r72.1。
- 全文 `若该键存在` = 2、`该键可缺` = 2，**全部位于 `:4031`／`:4062` 的否定引述**（“删除……条件分支”“不保留……读法”），不存在存活的条件分支。

安全判定：该例外是**错误分类层面的窄化**,而非放行规则——启动前仍拒绝（`ErrAdmissionBinding` 非 nil，仍在 `Journal.Begin` 前返回），工具已启动后仍 `restore_unknown`，不产生成功路径，也未为其他缺键开口子。与 `framework.md:1118`（“policy 必填……缺键、空串或字段名不符即 closed——启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`”）一致。

判定：**CLOSED（安全视角）**。

### 4.3 `NEW-R11-02`（LOW，架构侧）：CLOSED（本轮已补）

`:4135` 新增“其七（r72.2）”：`V7` 独立读取回执前、`V8` 验证 producer 签名前各一次 `revalidateApproval` 类重验，明确“不新增、不重排、不复用 V 编号”，并写明该两点位于工具已启动后，失败按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown` 处置。`V12` 全文 0 次；`:4037` 的 V7/V8 重验措辞（“只增加拒绝点、不放宽既有检查”）未变。

判定：**CLOSED**，且只是登记补齐，不改变任何判据。

## 5. 必查项逐条核对

### 5.1 `:4133` 错误分类三分法（必查项）

| 触发条件 | 分类 | 是否在成功前拒绝 |
|---|---|---|
| 预检后字节（含截断）／`generation`／inode／`st_dev`／挂载来源或挂载点漂移 | `ErrAdmissionClosed` | 是（非 nil 返回） |
| 未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked` | `ErrAdmissionClosed` | 是 |
| 文件缺失、不可读、非 regular、被符号链接替换 | `ErrAdmissionIncomplete` | 是 |
| 同一快照内格式／签名／字段不一致 | `ErrAdmissionBinding` | 是 |
| policy `tool.receipt.trust_anchor_ref` 缺键／空串／名不符 | 启动前 `ErrAdmissionBinding`；已启动后 `restore_unknown` | 是（启动前）；已启动后不得当成功 |

无“忽略错误继续”的分支；`:4133` 保留“仅 `revalidateApproval` 与 `recoveryRestoreStartupPreflight` 两处允许读批准文件／目标 DSN／信任根”的读取面约束，未放宽。

### 5.2 门禁与 Begin 路径（`:4070`）

- 启动预检与 `P3`／`P6` 位于 `Journal.Begin` **之前**；其漂移 → `ErrAdmissionClosed` 且**不进入 `Journal.Begin`**（即 Begin 前不写 DB／CAS）。
- `R1`／`R2`／`R4` 属 `enter` 的 `Replace` 路径、位于同一 `Journal.Begin` **之后**；`V1`／`V2`／`V4` 属 `VerifyRestored`（`enter` 侧在 Begin 后；`resume` 侧**无新 `Journal.Begin`**、在 `Prepare` 之后）。
- 越过工具启动点（`R6`／`V6`、`V7`／`V8`、`V10`）→ 一律 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown`；**任何情况下不得推进 `sequence`、不得当成功**。
- r71 曾把 R／V 写成“不进入 `Journal.Begin`”，r72 `:4070` 已按路径更正；本轮全文复核未见该旧写法残留。

### 5.3 policy 例外是否只限该字段

`:3944`／`:4031`／`:4062`／`:4377` 均以“**只限此字段**、不得扩展到任何其他缺键”收束；`:4070` 写“只限该字段”。全文未出现第二个被豁免的字段名。判读：例外面 = 1 个字段 × 1 组码值，不扩大。

### 5.4 `sequence` 推进

`:4070`（V10 段）、`:4135` 其六（r67.5）、`:4062` V10 定义、`:4133` 例外句均要求失败不推进 `sequence`；`:4135` 其七同样把 V7／V8 失败归 `restore_unknown`。未发现“先推进后校验”或“失败仍推进”的表述。

### 5.5 作者主动澄清的 `:4051`

新增 r72.4：明确 r63.9.4／r64.12 建议写集为**累积口径**，r65.12／r66.4 只局部取代所列职责写法，不废止其余增补项；点名六个生产文件与配套测试，供 `F017-R1` 写集扩写；并写“是否并入本卡仍由 Task-Planner 决定”。

安全判定：该段是写集口径说明，**不构成授权、不扩大实施面、不改变任何安全判据**；`:4051` 原句“该表仍只作建议，不改 `tasks.md`、不改卡面、不勾选”未删。`tasks.md:3186` 仍为 `- [ ]`（实测未勾选）。因此不构成本轮安全 finding。

## 6. fail-closed 逐路径表

| 检查点 / 路径 | 相对 Begin 与工具启动点 | 失败分类（r72） | 拒绝语义 | `sequence` |
|---|---|---|---|---|
| 启动预检 `recoveryRestoreStartupPreflight` | Begin 前 | `:4377` 按原因；预检后漂移 → `ErrAdmissionClosed` | 返回非 nil，不进入后续动作 | 不涉及 |
| `P3` | Begin 前 | 同上 | 不进入 `Journal.Begin`，不写 DB／CAS | 不涉及 |
| `P6`／`Prepare` | Begin 前（`enter` 的 `Journal.Begin` 之前） | 缺键／缺失／不可读 → `Incomplete`；同快照不一致 → `Binding`；有效期／撤销／漂移 → `Closed`；policy 该字段 → 启动前 `Binding` | `enter` 不进入 `Journal.Begin`；`resume` 不新 Begin、不 `Replace` | 不推进 |
| `R1`／`R2`／`R4` | `enter` 的 `Journal.Begin` 之后、工具启动前 | `:4377` 按原因 | 返回非 nil，不产生成功 | 不推进 |
| `V1`／`V2`／`V4` | `enter`：Begin 后、工具启动前；`resume`：无新 Begin、`Prepare` 之后 | `:4377` 按原因 | 同上 | 不推进 |
| `R6`／`V6` | 工具启动后 | `errors.Join(ErrRecoveryWriteUnknown, err)`／`restore_unknown` | 不当成功；审计只能写 `unknown` | 不推进 |
| `V7`／`V8` | 工具启动后 | 同上（r72.2 明确） | 同上 | 不推进 |
| `V10` | 工具启动后、推进见证 head `sequence` 之前 | 同上 | 失败不得推进 `sequence` | 不推进 |
| policy `trust_anchor_ref` 例外 | 启动前 / 已启动后 | 启动前 `Binding` / 已启动后 `restore_unknown` | 窄例外，不扩字段 | 不推进 |
| 审计投影 `result` | 全程 | `verified`／`unknown`／`closed`／`incomplete` 封闭枚举；`restore_unknown` 只能写 `unknown` | 不得写 `verified`；审计写失败不得把 unknown 改写成成功 | — |

结论：所有路径都在“返回成功之前”拒绝；Begin 前不写 DB／CAS；越过工具启动点后统一 `restore_unknown`；不存在“不可读／缺键被当成已授权”的读法。

## 7. 与 `framework.md:1118`（ADR-0031 r12）对齐

| 条款 | ADR `:1118` | design r72 | 判定 |
|---|---|---|---|
| policy `tool.receipt.trust_anchor_ref` 必填 | 必填，缺键／空串／名不符即 closed | `:4031`／`:4062` 必填 + 删除条件分支；`:3944`／`:4377` canonical | 一致 |
| 缺键的闭类 | 启动前 `ErrAdmissionBinding`、已启动后 `restore_unknown` | r72.1 同一口径（五处） | 一致 |
| `anchor_ref` 五处逐字节相等 | `receipt_anchors.anchor_ref` = policy = 回执 `trust_anchor_ref` = 审计 = 见证 head | `:4031`／`:4062` 保留五处等式与“其余四处任一不等 → Binding” | 一致 |
| `generation` 三元 | 未变 | `:4062`／`:4135` 未变 | 一致 |
| 三签与三席位密钥来源 | 未变 | `:4031` 八字段 verifier 载荷、producer／witness 签名来源未变 | 一致 |
| 见证分域与“模拟见证、未真实验证” | 未变 | 全文“模拟见证”2 处，标注保留 | 一致 |
| 不声称抗 root | 无条件成立 | “不声称抗 root”3 处，未削弱 | 一致 |

## 8. golden vector 独立复算

方法：`[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8)` 取整行 → `UTF8.GetBytes`（不含行终止符）→ `SHA256`。

| 行 | 角色 | 实测字节 | 实测 SHA256 | 期望 | 判定 |
|---|---|---|---|---|---|
| `:4099` source | `restore_reviewer`／`restore_approver` | **1,115** | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 1,115 / `de914622…7b0cab` | 一致 |
| `:4105` tool | `restore_executor`／`restore_approver`／`restore_reviewer` | **2,357** | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 2,357 / `8c0c89…326508` | 一致 |

角色字面：`:4061` 六值封闭枚举未改动（`restore_approver`／`restore_reviewer`／`restore_executor`／`restore_producer`／`restore_verifier`／`restore_witness`）；两条向量出现的角色字面均在该枚举内（`restore_source`／`restore_tool` 是文件／概念名，不是角色）。

## 9. 三段后缀复算（raw bytes，LF 行首偏移 → EOF）

方法：`ReadAllBytes` → 记录每行行首偏移（`0` 与每个 `LF+1`）→ 从目标行行首切到 EOF → `SHA256`。不使用字符串拼接，不受行终止符归一化影响。

| 行 | startByte | 长度 | SHA256 |
|---|---|---|---|
| `:4259` | 826,545 | 283,932 | `b8135070edecb97470255d0dd505e93fd05e8d5197f8475dbd302832582ff4af` |
| `:4275` | 828,825 | 281,652 | `26d4f46c6aa3c4a3d79bf8e472ce5bf95a4b80c596cff876fad776c0552d220c` |
| `:4290` | 831,810 | 278,667 | `58b55f08acf1af5fba85f38a1d65f6cf6399616dc989bfaf49a3d1d31d6a9b6a` |

说明：r72 在 `:4135` 之前新增约 3.8 KB 文本，故三个 startByte 相对 r69／r70 整体后移；行数不变、`:4259`／`:4275`／`:4290` 的行号映射仍成立。本节值仅作**观测登记**：设计中该三段为“条件式断言”，不作为放行判据，也不是本轮 verdict 依据。

## 10. 无回退底线逐项

| 底线 | 现状（r72） | 判定 |
|---|---|---|
| `restore-v1`／`verify-v1` | 未在改动行内；向量字节与哈希不变 | 未回退 |
| 无 shell | 改动行无 shell 化表述 | 未回退 |
| 私有 stdin 传 secret／DSN | 改动行未触及；`:1481` `Secret []byte // root owns buffer; no String/JSON/log representation` 保留 | 未回退 |
| 最小 ENV allowlist | 未触及 | 未回退 |
| Begin 前拒绝（任一非 nil 在 Begin 前返回） | `:3944`／`:4377` 明写；`:4070` 路径化 | 未回退 |
| 不得以 stdout／exit=0／DB 可连接自证 | 未触及 | 未回退 |
| `injectedExclusive` 永不 `Close` | `:4076` 未改动（10 处 `injectedExclusive` 表述保留） | 未回退 |
| teardown 顺序 `Stop → flowBothDone → waitFlow → closeRestoreLease` | `:4076` 未改动；“`close(a.flowBothDone)`”5 处、`各起` 仅见于 r70.2 的否定历史登记 | 未回退 |
| r70 已闭合项（`NEW-R09-01`／`NEW-R09-02`／`NEW-S09-01`） | `:4076`／`:4079` 不在 12 行内 → 逐字节未变 | 未回退 |
| golden vector `:4099`／`:4105` | 复算一致 | 未回退 |
| 审计 `result` 枚举与“`restore_unknown` 只能写 `unknown`” | `:3944` 保留 | 未回退 |
| 错误／日志／receipt／plan wire 不含 raw secret | 12 行改动无新增 secret 路径、无日志回显、无 wire 字段 | 未回退 |
| 六值角色封闭枚举 | `:4061` 未改动 | 未回退 |
| `anchor_ref` 五处等式 | `:4031`／`:4062` 保留 | 未回退 |
| `generation` 三元 | 未改动 | 未回退 |
| `receipt_anchors` 无 `revocation`／`valid_until` | 未改动 | 未回退 |
| 两种“见证”分域 | 未改动 | 未回退 |
| Windows 生产 unsupported | 5 处保留 | 未回退 |
| “首期不提供生产 restore 工具与生产批准主体” | 8 处保留 | 未回退 |
| 声明段（未解锁） | `:4083` 追加“r72 修订沿用本声明，不改变任何 NOT RUN／unsupported／closed 状态” | 未回退 |

## 11. 新 findings

**无。** `NEW-S12-nn` 未分配：12 行改动均为声明／错误分类／登记层面的收敛，未削弱任何安全判据，也未引入新的凭据面或可观测泄漏面。

## 12. 范围外观察（不计入本轮 finding 计数）

- `N005`：`framework.md` 状态段与 `tasks.md` 的冲突仍属范围外阻断，本轮只登记、不据此判 CHANGES REQUIRED、不代改。
- `tasks.md:3186`：实测仍为 `- [ ] identity-auth-S4.24a …`（未勾选），符合只读预期。
- `F017-R1` 写集：`:4051` 明确“是否并入本卡仍由 Task-Planner 决定”，属任务规划面，不在本轮设计安全判定内。
- ADR 侧 `ADR-S06-01`：位于 `.spec/wip/recovery-runtime/evidence-b13j-d06/`，不属本设计对象的修复责任。

## 13. NOT RUN

以下一律 **NOT RUN**，不得因设计文字存在而判 PASS：

- `SEC-F007` 及 Linux 侧能力：POSIX 宿主隔离、进程树 reap、专用执行身份、ACL、mount、loader、fd-exec、`/proc` maps、network、witness 实跑。
- 真实 restore 工具、真实审批主体、外部见证／HSM 的真实签名验证（设计内为“模拟见证、未真实验证”）。
- build / test / E2E / 静态分析 / 竞态检测。

由此：本报告**不**声称生产 restore 解锁，**不**声称 `enter`／`resume` 解锁，**不**声称约束 8 满足，**不**声称 `identity-auth-S4.24a` 可关闭或可派发，**不**声称任何安全红线解除。

## 14. 实际执行命令（代表性）

```powershell
# 对象与基线
$b=[IO.File]::ReadAllBytes($p); (Get-FileHash -Algorithm SHA256 -LiteralPath $p).Hash
# 行/CR/LF/BOM/末字节
[IO.File]::ReadAllLines($p,[Text.Encoding]::UTF8) ; foreach($x in $b){ if($x -eq 13){$cr++}; if($x -eq 10){$lf++} }
# 12 行边界：逐行内容 + UTF8 字节增量 + 行尾风格
for($i=0;$i -lt $cur.Count;$i++){ if($cur[$i] -ne $old[$i]){ ... } }
# golden vector
$by=[Text.Encoding]::UTF8.GetBytes($L[4098]); [Security.Cryptography.SHA256]::Create().ComputeHash($by)
# 三段后缀（raw bytes）
$starts[0]=0; for(...) if($b[$i] -eq 10){$starts.Add($i+1)}; [Array]::Copy($b,$starts[4258],$seg,0,$len)
# 全局计数
[regex]::Matches($t,[regex]::Escape('若该键存在')).Count
```

## 15. 计数

| 严重级 | 数量 |
|---|---|
| BLOCKER | 0 |
| MEDIUM | 0 |
| LOW | 0 |

- 已闭合（本轮判定）：`NEW-S11-01`（LOW）、`NEW-R11-01`（MEDIUM，安全视角）、`NEW-R11-02`（LOW）。
- 开放项：无。
- 范围外登记：`N005`、`tasks.md:3186` 未勾选、`F017-R1` 写集待 Task-Planner、ADR 侧 `ADR-S06-01`。

Verdict: PASS