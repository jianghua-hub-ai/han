# identity-auth 详细设计 r71 作者修订报告（S4.24a / B13J）

- 角色：`add-feature-designer`；方法：`add-design-tdd`
- 日期：2026-11-02（Asia/Shanghai）
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 修订：`identity-auth-design-ir01-r70` → `identity-auth-design-ir01-r71`
- 写集：`design.md`、`design-revision-r11/design.r70.baseline.md`、`design-revision-r11/report.md`、`design-revision-r11/hashes.txt`
- 证据目录：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r11/`

## 1 结论

本轮完成 r71 就地修订，未新增、删除或重排行；四指定项均已在设计文本层闭合。第 4 项采用“以 `:4377` 的按原因分类为唯一 canonical”的分支，并在 `:3944` 写明取代关系；为使该唯一映射不再与 `:4377` 自身的旧句并存，额外修正了 `:4377` 一行（见 §3.3），故实际改动行为 10 行，超过派发默认的 9 行，理由与字节账已逐条列明。

本轮不构成设计评审通过，不构成生产 restore 解锁，不构成 `enter`／`resume` 解锁，不构成约束 8 满足，不构成 `identity-auth-S4.24a` 可关闭；未勾选 `tasks.md` 任何复选框。

## 2 对象与改前／改后终值

| 项目 | 改前 r70（基线快照） | 改后 r71（现行 design.md） | 差值 |
|---|---:|---:|---:|
| 字节 B | 1,103,152 | 1,106,644 | +3,492 |
| 行数（ReadAllLines） | 5,476 | 5,476 | 0 |
| CR | 169 | 169 | 0 |
| LF 总数 | 5,476 | 5,476 | 0 |
| CRLF | 169 | 169 | 0 |
| bare LF | 5,307 | 5,307 | 0 |
| BOM | 无（首字节 `2D 2D 2D`） | 无（首字节 `2D 2D 2D`） | — |
| 末字节 | `0A`（LF） | `0A`（LF） | — |
| SHA256 | `3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254` | `168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5` | 变化 |
| revision（`:5`／`:38`） | `identity-auth-design-ir01-r70` | `identity-auth-design-ir01-r71` | 升版 |

基线快照：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r11/design.r70.baseline.md`，B=1,103,152、SHA256=`3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`，与派发进入值及写前 `design.md` 逐字节相等（True）。

`design.md` 于 2026-11-02 18:49:53.775 +08:00 完成唯一一次写盘；本报告生成前复核其未再变化。全角括号 `（`／`）` 由 1,216／1,216 变为 1,237／1,237，保持平衡。

## 3 写盘方式与逐行字节账

### 3.1 写盘方式

`[IO.File]::ReadAllText($p,[Text.Encoding]::UTF8)` 读出 → 对精确子串执行 `Replace`，每处先断言命中次数 → `[IO.File]::WriteAllText($p,$txt,(New-Object Text.UTF8Encoding($false)))` 一次写回。未使用 `Set-Content`／`Out-File`，未重编码、未归一化换行、未加 BOM。写后 CRLF／bare LF／首末字节均与改前一致。

### 3.2 改动行集合与字节账

行内字节数按 UTF-8 编码、**不含行尾**计。

| 行号 | 改前 B | 改后 B | 增量 Δ | 一句摘要 |
|---:|---:|---:|---:|---|
| `:5` | 39 | 39 | 0 | revision 升为 `identity-auth-design-ir01-r71` |
| `:38` | 50 | 50 | 0 | 设计版本同步升为 `identity-auth-design-ir01-r71` |
| `:3944` | 545 | 1,370 | +825 | 不可读错误分类收敛为唯一 canonical 按原因映射，并写明取代关系 |
| `:4031` | 1,831 | 2,006 | +175 | policy 的 `tool.receipt.trust_anchor_ref` 改为必填 |
| `:4037` | 2,531 | 2,796 | +265 | V7 独立读取前、V8 验 producer 签名前各登记一次重验 |
| `:4062` | 2,234 | 2,655 | +421 | V7／V8 前置重验与 policy 必填同步进 V 编号语义 |
| `:4070` | 5,207 | 5,858 | +651 | B1 错误分类与 `Journal.Begin` 门禁按路径更正 |
| `:4072` | 8,022 | 8,952 | +930 | 登记 r71 就地修订、r71.1–r71.5 与优先级链 |
| `:4083` | 596 | 681 | +85 | 追加 r71 沿用声明，不改变任何 NOT RUN／unsupported／closed 状态 |
| `:4377` | 861 | 1,001 | +140 | 修正“预检后漂移→Binding”旧句为 Closed，并注明 r71.4 取代 |
| **合计** |  |  | **+3,492** | 等于全文 1,106,644 − 1,103,152 |

除上述 10 行外，其余 5,466 行按 Ordinal 逐行比较逐字节相同。改动行集合恰为 `:5`／`:38`／`:3944`／`:4031`／`:4037`／`:4062`／`:4070`／`:4072`／`:4083`／`:4377`。

### 3.3 为何第 10 行 `:4377` 是必要的

只改 `:3944` 会留下 `:4377` 的旧句“预检通过后字节漂移（含 mtime／size／inode 变化）→ `ErrAdmissionBinding`”，与 `:3944` 新映射的“预检通过后的字节／`generation` 漂移 → `ErrAdmissionClosed`”并存，仍违反“两处收敛为唯一 canonical、不得两种写法并存”。因此必须同步修正 `:4377` 该句，并加 r71.4 取代注。该行变更 +140 B；其位于 `:4259` 之后，直接解释三段后缀各 +140 B。

## 4 四项 finding 的闭合

### 4.1 policy 的 `anchor_ref` 必填化

- 落点：`:4031`、`:4062`
- 改法：原“批准记录（policy）中的 `anchor_ref`（若该键存在）”两处改为“批准记录（policy）**必填**的 `tool.receipt.trust_anchor_ref`”。两处均写明 policy 侧缺键、空串或字段名不符即不得放行；删除“若该键存在”的条件分支，不保留“policy 该键可缺”的读法。
- 未变：五处等式链其余四处、三签模型、`generation` 三元、启动前 `ErrAdmissionBinding`／已启动后 `restore_unknown` 的既有分类。
- 机核：旧条件短语全文命中 0；新 policy 必填短语命中 2。全文仅剩的“若该键存在”字样位于新增否定引述“删除「若该键存在」条件分支”内，不是保留条件分支。
- 判定：**CLOSED（设计文本层）**。

### 4.2 `:4070` B1 摘要的错误分类与 Begin 门禁按路径更正

- 落点：`:4070`
- 改法：删除“`P3`／`P6`／`R1`／`V1`／`R2`／`V2`／`R4`／`V4` 漂移 → `ErrAdmissionClosed`，且不进入 `Journal.Begin`”的混写，改为按路径：
  - 只有启动预检与 `P3`／`P6` 位于 `Journal.Begin` 之前；其漂移 → `ErrAdmissionClosed` 且不进入 `Journal.Begin`。
  - `R1`／`R2`／`R4` 属 `enter` 的 `Replace` 路径，位于同一 `Journal.Begin` 之后。
  - `V1`／`V2`／`V4` 属 `VerifyRestored`：`enter` 侧在 `Journal.Begin` 之后；`resume` 侧无新 `Journal.Begin` 且在 `Prepare` 之后。
  - 这些 R／V 点未越过工具启动点时按 `:4377` 原因分类；越过工具启动点后（`R6`／`V6` 以及 `V7`／`V8`／`V10` 前）一律 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown`。
- 保留：B1 只作摘要、非穷举的限定；并给出可机核完备集合“`P6` 与 `P6` 之后每个检查点”，指向 `:4037`／`:4062`／`:4135`／`:4137`。
- 机核：旧串“错误分类随检查点同步”命中 0；新完备集合指向存在。
- 判定：**CLOSED（设计文本层）**。

### 4.3 V7／V8 前重验登记进唯一调用图

- 落点：`:4037`、`:4062`
- 改法：在既有 V 编号语义内登记“`V7` 独立读取回执前”与“`V8` 验证 producer 签名前”各执行一次 `revalidateApproval` 类重验；明确只增加拒绝点、不放宽既有检查，失败 → `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`／`restore_unknown`。
- 未变：未新增、重排或复用 V 编号；`V1`–`V11` 既有语义不变；`V10` 推进见证 `sequence` 前的重验保留。
- 判定：**CLOSED（设计文本层）**。

### 4.4 不可读错误分类收敛

- 落点：`:3944`、`:4377`
- 选择分支：**以 `:4377` 的按原因分类为唯一 canonical**。
- 依据：
  1. ADR-0031 r12（`framework.md:1118`）已采用按原因分类，并明确 `P6` 与 `Prepare` 内重读按 design `:4377` 分类：缺键、文件缺失或不可读 → `ErrAdmissionIncomplete`；有效期、撤销、预检后字节／`generation` 漂移 → `ErrAdmissionClosed`。
  2. `:4377` 的分类集合更完整，且已有“同一快照内不一致才用 `ErrAdmissionBinding`”的限定；保留 `:3944` 旧“不可读 → Closed”一枝会与 ADR r12 冲突。
- 改法：
  - `:3944` 原地改写为唯一 canonical 映射：缺键、文件缺失或不可读（含批准文件、信任根、目标 DSN、回执独立读取路径、审计投影）→ `ErrAdmissionIncomplete`；同快照内格式／签名／字段不一致 → `ErrAdmissionBinding`；未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked`、预检通过后的字节／`generation` 漂移 → `ErrAdmissionClosed`；平台不支持 → `ErrAdmissionUnsupported`。
  - 在 `:3944` 写明旧“批准文件、信任根或目标 DSN 不可读 → `ErrAdmissionClosed`”作废、旧“回执独立读取路径或审计投影不可读 → `ErrAdmissionIncomplete`”并入本映射，并把“不得互换”限定到新的四类映射。
  - 保留 `:4133` 的 `revalidateApproval` 漂移／替换判定按该路径既有分类，不并入本映射。
  - `:4377` 同步修正旧“预检通过后字节漂移 → `ErrAdmissionBinding`”为“预检通过后的字节／`generation` 漂移 → `ErrAdmissionClosed`”，并注明 `:3944` 按本条读取。
- 未变：fail-closed；任一非 nil 都在 Begin 前返回；“不得先恢复后补批准”。
- 判定：**CLOSED（设计文本层）**。本轮未发现足以保留 `ErrAdmissionClosed` 不可读一枝的反证。

### 4.5 版本与登记

- `:5`／`:38`：`identity-auth-design-ir01-r71`。
- `:4072`：加入“**r71 就地修订（无新增／删除行）**”，列明 r71.1（`:4031`／`:4062`）、r71.2（`:4070`）、r71.3（`:4037`／`:4062`）、r71.4（`:3944`／`:4377`）、r71.5（`:5`／`:38`／`:4072`／`:4083`）；唯一优先级规则扩为 `r71 > r70 > r69 > r68 > r67 > r66 > r65 > r64 > r63 > r62 > r61`。因未增删行，原“插入后 = 插入前 + 12”口径保持有效且未改。
- `:4083`：追加“r71 修订沿用本声明，不改变任何 NOT RUN／unsupported／closed 状态。”

## 5 验证与未回退

- 行数：写后 `[IO.File]::ReadAllLines(...,UTF8)` = 5,476，零增删。
- 行号：未发生移动；被引用行 `:4113`、`:4037` 等位置不变；`:4377` 位置不变但按第 4 项要求改写了内容。
- golden source（`:4099`）：1,115 B／`de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab`，PASS。
- golden tool（`:4105`）：2,357 B／`8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508`，PASS。
- 保留的既有关闭项：strict v1、无 shell、私有 stdin、最小 ENV、Begin 前 closed、不得以 stdout／exit=0／DB 可连接自证、`injectedExclusive` 永不 Close、`Stop → flowBothDone → waitFlow → closeRestoreLease` 顺序、执行器不响应 ctx 时持租约 fail-closed；r67–r70 的 `F006`、`NEW-R07-01/02/03`、`NEW-R05-03`、`NEW-R07-04` 文本层、`F004` 文本层、`NEW-R05-04/05`、`NEW-R06-02/05/06/07/08`、`SEC-F002/F003`、`N001`、`SEC-F006`、`N011`、`NEW-S05-01/02/04/05/06/07`、`N012`–`N016`、`NEW-S06-02/03`、`NEW-S07-01`、r69 双路 join、r70 `waitFlow` 单点合同与单 reaper 均未回退。
- 未修改：`tasks.md`、`framework.md`、`decisions/README.md`、`cmd/**`、`internal/**`、测试、任何 `review-*/`、`design-revision-r01..r10/**`；未执行 Git 写、未删除／移动／归档。

## 6 三段后缀复算

方法：对每个冻结副本取 raw bytes；以 LF 计行首偏移（CRLF 的 CR 计入前一行尾，不单独计行），从目标行行首字节到 EOF 取原始区间；不重新编码、不做换行归一。复算时点：2026-11-02 18:51:08 +08:00。

| 行 | 改前 startByte | 改前 B | 改前 SHA256 | 改后 startByte | 改后 B | 改后 SHA256 |
|---:|---:|---:|---|---:|---:|---|
| `:4259` | 819,672 | 283,480 | `e8848838df83bc09bd8c94297219702cd86bd6629bd31fd8853f834f652cf0b6` | 823,024 | 283,620 | `3798fb8f7cc457b8743a9674ccebe6506c563f35c57975f7654eefd429c16358` |
| `:4275` | 821,952 | 281,200 | `2f53882f8774d30a51754872449ba29b4f257253c8a2f457ab6b9109d054fafb` | 825,304 | 281,340 | `dd8f8d1642d562046186c17ec7810c1e8ed1f4709a86c1ef47f86a948e2f10ce` |
| `:4290` | 824,937 | 278,215 | `b0af3b04c60584cf2618c83605e7fba88f65ca6b48f486463570268279e05f04` | 828,289 | 278,355 | `32bdd8d002fefa0ee04ac02c443bd060430699f7ac992c448216d8d141354b41` |

解释：改前值与 `design-revision-r10/hashes.txt` 节 [5] 逐项相同；本轮唯一位于 `:4259` 之后的写盘改动是 `:4377`（+140 B），故三段后缀字节数各 +140 B，三段 startByte 各 +3,352 B（= 全文 +3,492 − 后缀内 +140）。不是“后缀不变”。

## 7 输入核对差异（升级待决）

派发文本称 ADR-0031 `adr-0031-r12` 已由 `review-arch-r06`（PASS 0/0/0/0）与 `review-sec-r06`（PASS 0/0/0/1）接受。按指定路径只读核验，实际内容是：

- `review-arch-r06/report.md`：B=28,491、248 行、SHA256=`C66F9BBEB2C04103A9947078A8B86A807037C52ED890773696B2E7E30B53B47E`；对象为 design r66；Verdict **CHANGES REQUIRED**。
- `review-sec-r06/report.md`：B=28,417、232 行、SHA256=`31BAB8B0F67A2EC8706E52E8B897534E14B196B16FA4F84F1B7208A50F291B08`；对象为 design r66；因对象漂移**评审中止**，明确“不构成安全侧接受”。

因此本报告只声称“按 ADR-0031 r12 在 `framework.md:1118` 已载明的口径落回设计”，不声称两路 r06 已接受 ADR，也不声称本轮已获评审通过。所指 PASS 报告未在指定路径找到，交 Lead 核对。

## 8 未闭合／残余／升级待决

- `F004` 的 ADR 同步：ADR-0031 r12 内仍以 design r70 为现行引用；本轮 r71 已按该 ADR 口径落回，ADR 侧版本引用同步不在本轮写集。
- `F017` 任务面对账：`tasks.md` 的 S4.24a 写集仍待 Task-Planner 对账；本轮不改 `tasks.md`、不勾选、不改卡面。
- 范围外 `N005`：ADR 状态段与 `tasks.md` 状态段的冲突未处理。
- 输入核对差异：见 §7；两路 r06 的 PASS 声称与磁盘对象不一致，需 Lead 核对。
- 本轮不新增任何 `unsupported` 降级、不改变任何既有 NOT RUN，也不把软件合同测试写成实测。

## 9 NOT RUN

- 设计评审：两路独立复审 NOT RUN（本轮为作者修订）。
- build／compile／unit test／integration test／race test：NOT RUN。
- E2E／真实 restore：NOT RUN。
- Linux host／ACL／mount／loader／fd-exec／maps／network／witness 实测：NOT RUN。
- 真实审批角色任命、真实旧备份样本、真实外部见证：NOT RUN。
- Git 写操作：未执行（按边界禁止）。

## 10 声明

本轮不构成评审通过／不构成生产 restore 解锁／不构成 `enter`、`resume` 解锁／不构成约束 8 满足／`identity-auth-S4.24a` 不可关闭。任何 NOT RUN 仍是 NOT RUN，任何 unsupported 仍是 closed。最终是否接受由两路独立复审与人的成果确认另行判定。