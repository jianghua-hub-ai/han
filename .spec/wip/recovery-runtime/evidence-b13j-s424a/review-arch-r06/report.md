# identity-auth 详细设计 r66 架构侧独立复审（B13J／S4.24a，review-arch-r06）

- 角色：add-reviewer（架构侧独立评审；非作者、非实现者）
- 对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 修订：`identity-auth-design-ir01-r66`（`:5`、`:38`）
- 工作区：`D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审时点：2026-11-02 15:19–15:30（Asia/Shanghai）
- 上轮：`review-arch-r05`（CHANGES REQUIRED）、`review-sec-r05`（CHANGES REQUIRED）
- 作者证据：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r06/report.md`、`hashes.txt`（只作线索，数值全部自算）

**Verdict: CHANGES REQUIRED**

**是否构成 `design.md` r66 的架构侧接受：否，r66 尚不构成 design.md 的架构侧接受。**

三条独立理由：（1）`design.md` 在评审期间发生字节漂移（停止条件已触发）；（2）R05 结转 BLOCKER `F006` 仍未闭合——r66.4 规定的 `checkOwnership` 接入点会自锁，nil 检查在现有调用图上没有「任何状态变更之前」的落点；（3）作者证据链与 r65→r66 字节账仍不可复算，且属 `NEW-R05-02`／`NEW-S05-03` 同类复发。

## 计数

| 类别 | 数量 | 编号 |
|---|---:|---|
| 结转未闭合 BLOCKER | 1 | `F006`（含并入的 `NEW-R05-01`） |
| 结转未闭合 MEDIUM | 2 | `NEW-R05-02`、`NEW-R05-04` |
| 本轮新增 HIGH | 1 | `NEW-R06-01` |
| 本轮新增 MEDIUM | 5 | `NEW-R06-02`～`NEW-R06-06` |
| 本轮新增 LOW | 2 | `NEW-R06-07`、`NEW-R06-08`（后者已随作者改写消失，仅在 15:19 快照成立） |
| 本轮判闭合 | 3 | `F004`、`NEW-R05-03`（基本闭合，残 LOW）、`NEW-R05-05` |

## 0. 对象与实测基线

### 0.1 停止条件触发记录（先于一切实质结论）

| 时点 | 事件 | 证据 |
|---|---|---|
| 15:10:32 | 作者写 `hashes.txt`，记终值 1,078,886 B／`0BA1C6C1…` | 文件 mtime |
| 15:10:40 | 作者写 `report.md` | 文件 mtime |
| 15:18:54 | `design.md` 被再次写盘 | 文件 mtime |
| 15:19:22 | 本端实测 1,079,559 B／`9B193BDE…` | 本端 `[IO.File]::ReadAllBytes` |

本会话冻结基线为 **1,078,886 B／5,476 行／SHA256 `0BA1C6C1B2F4C21BA4BC3FA5F7E3A949040BC01BE00329D485D6CC4DC2FA1847`**（与作者 `hashes.txt` §1 一致，亦与本会话前段实测一致）。15:19:22 复测为 **1,079,559 B／5,476 行／SHA256 `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298`**，即 **+673 B**，行数与修订号未变。

按任务停止条件「若 `design.md` 在你评审期间被改动（B/SHA 变化），停止并报告」，本报告不把 15:18:54 之后的字节当作评审对象。下文 findings 均针对冻结快照判定；每条 finding 的触发句已在 15:19:22 的现行字节上复核仍然存在（见各条「现行复核」），但我未对新增的 +205 B 块内改动做实质评审。

漂移定位（口径：块含末行 LF）：r66 块 `:4072`–`:4083` 由 30,936 B／`6E25E9CD…` 变为 31,141 B／`587471F5…`（+205 B；去掉末行 LF 为 31,140 B／`38BDE4DE…`）；`:4259`、`:4275`、`:4290` 至 EOF 三段哈希与冻结值完全一致（`7D1DDC08…`、`5BD8E870…`、`E40EB7A2…`），证明尾部 r61 段未变；其余 +468 B 落在 `:1`–`:4071` 或 `:4084`–`:4258`，无冻结副本可逐行定位。

### 0.2 实测基线表

| 对象 | 字节 | 行 | CR/LF | SHA256 | 备注 |
|---|---:|---:|---|---|---|
| `design.md`（冻结） | 1,078,886 | 5,476 | 169/5,476 | `0BA1C6C1B2F4C21BA4BC3FA5F7E3A949040BC01BE00329D485D6CC4DC2FA1847` | 本轮实质评审对象 |
| `design.md`（15:19:22 实测） | 1,079,559 | 5,476 | 169/5,476 | `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298` | mtime 15:18:54；**已漂移** |
| `.add/project/knowledge/decisions/framework.md` | 272,588 | 1,271 | 1,271/1,271 | `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4` | ADR-0031 现行 r09；r10 可能并行修改；只读参照 |
| `.add/project/knowledge/tasks/tasks.md` | 898,536 | 3,289 | 3,289/3,289 | `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` | `:3186` 仍 `- [ ]`；`:3197` 仍称 r07 已接受 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | 509,779 | — | — | — | mtime 08:48:33；无 `r66`／`restoreLease` 命中 |
| `review-arch-r05/report.md` | 32,489 | 281 | 0/281 | `DD2423B7BD574B5C61603F938B42856061EA3F7B4AA59BA4D033B04BE5AD93D1` | 上轮架构侧，CHANGES REQUIRED |
| `review-sec-r05/report.md` | 39,093 | 293 | 0/293 | `4A2292674E49EB4A7961CD1EDD0FA4BCF1560D2E911B74AE1E208AC3790DE0C1` | 上轮安全侧；判定不由本报告给出 |
| `design-revision-r06/report.md` | 30,201 | 327 | 0/327 | `4AC8FEDDCB92BFAC145476DEE11C5787A944E0370821B1C14C656B3ED1416AFE` | 作者报告（15:19 读取值）；其 `hashes.txt` 记 21,142 B／205 行／`A03CE62B…`，不符；15:20:55 已被改写，见 §0.3 |
| `design-revision-r06/hashes.txt` | 3,181 | 51 | 0/51 | `DE2D71BD48743583A4CEDF3712A26B16B581B38267A57517B6AFA6C5BF8954FD` | 15:19 读取值；已被作者改写，见 §0.3 |
| `design-revision-r06/report.md`（15:24 后复核） | 24,041 | 196 | 0/196 | `CA5F28AC27DC4951AA6D4335F4DC2DE5CF0C714E3CCBF79C0E3EA93FF8A7C671` | 作者重写版；见 §0.3 |
| `design-revision-r06/hashes.txt`（15:24 后复核） | 7,343 | 117 | 0/117 | `ABF946B7B0ED4577703D3A468D93476C1A730AFFD2D2C8D5FB3987EDB255E594` | 作者重写版；见 §0.3 |
| `framework.md`（15:23:48 复核） | 285,639 | 1,273 | 1,273/1,273 | `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050` | 并行 r10 写入；ADR 侧结论不在本对象内 |

### 0.3 证据侧与只读参照在评审读取之后的变化（15:23–15:24 复核）

- **作者证据被重写**：我在 15:19–15:23 读取的作者 `report.md` = 30,201 B／327 行／`4AC8FEDD…`，`hashes.txt` = 3,181 B／51 行／`DE2D71BD…`（上表值）。作者随后于 15:20:55、15:24:19 分别改写为 24,041 B／196 行／`CA5F28AC…` 与 7,343 B／117 行／`ABF946B7…`；新版把 1,078,886 标为「开工草稿态」、1,079,559 标为终值，与我 15:19:22 实测的现行文件一致。
- **因此**：`NEW-R06-02` 中「作者终值 ≠ 文件」的那一半已被作者事后修复；`NEW-R06-08` 的重复章节与 `:314` 旧值在新版中已不存在（`rg` 实测 `^## 5 ` = 1、`1,077,602` = 0）。
- **但 r65→r66 字节账仍不平（本端自算）**：
  - 总量：1,079,559 − 1,045,111 = **+34,448 B／+12 LF**。
  - 已登记：r66 块含末行 LF 31,141 B；`:3603` +468 B；旧版报告 §0 的「10 处行内替换」+1,284 B。合计 32,893 B。
  - 缺口：34,448 − 32,893 = **1,555 B／0 行**，作者未给逐行清单。
  - `design.md:4072`（r66.0）自述「对 `:3603`、`:3942`、`:3990`、`:4062`、`:4187` 等既有行追加 r66 就地指针」，说明确有未逐点列明的行内追加；1,555 B 与之同量级，但无前后字节数，不可核。
  - 新版 §2 的「382 → 850」「312 → 517」是句段级字节，不是整行字节（本端实测 `:3603` = 1,518 B、`:4072` = 3,155 B，均不含行尾），表头「行／UTF-8 字节增量」口径不清，不能按整行复核。
- **评审对象本身**：15:19:22 与 15:24 后两次实测同为 1,079,559 B／5,476 行／`9B193BDE…`，mtime 仍为 15:18:54，即 `design.md` 在 15:18:54 之后未再变化；本报告仍未对该次写入做实质评审，故 `NEW-R06-01` 维持。
- **只读参照 `framework.md`**：我读取时 272,588 B／1,271 行／`D8DBA64B…`；15:23:48 复测为 285,639 B／1,273 行／`B432E44A…`（并行 r10 修订）。ADR-0031 的结论不在本对象内，本报告不因 ADR 侧变化改判；`tasks.md`／`plans/**` 读取前后同值。

## 1. 两轮实际检查范围

**第一轮（对象结构与规范面，只读）**：版本头 `:5`／`:38`；全文 B/行/CR/BOM；r66 块 `:4072`–`:4083` 全文；`:3603`、`:3942`、`:3990`、`:4062`、`:4187` 的就地指针；r65 基线算术；段哈希 `:4072`–`:4083`、`:4259`／`:4275`／`:4290` 至 EOF、`:4435`、golden `:4099`／`:4105`；作者 `report.md`／`hashes.txt` 的结构与数值；磁盘 R05 报告的第 3、5 节编号与判语。

**第二轮（消费者与实现面，只读）**：`rg` 39 点／17 文件逐文件计数；`internal/recovery/runtime.go` 定义 `:729`、字段 `:768`、`checkOwnership` `:1160`–`:1177`、`Enter`／`Resume` `:1517`–`:1552`、`recoveryFlow.acquire` `:1554`–`:1571` 与 `enter`／`resume` 调用链；`admission.go` 的 `checkOwnership` 调用者；`cmd/han-server/recovery_runtime.go` `:441`／`:596`–`:599`／`:641`；`recovery_restore.go` `:203`；`recovery_restore_test.go` `:204`／`:208`／`:209`／`:228`／`:233`；`recoveryRootInputs` 字面量；`main.go:634`；`framework.md`／`tasks.md`／`plans`／Go 代码越界扫描；ADR-0031 只读核对（r09、门禁 blocked、`NewRuntime` 出现 0 次）。

**检查限制**：r65 完整字节不可得（HEAD 版本为 2026-10-21 的旧快照，工作区为未提交修改）；`%TEMP%` 仅有前缀备份与中间快照；因此 r65→r66 全量 diff 只能做算术核对，不能独立重放。见 `NEW-R06-02`。

## 2. R05 开放项逐条闭合矩阵（按磁盘 R05 报告自身编号）

| R05 编号（磁盘） | 级别 | 本轮判定 | 依据 |
|---|---|---|---|
| `F006` | BLOCKER | **未闭合** | 访问器模型与 teardown 已给出，但 `checkOwnership` 接入点自锁（`NEW-R06-03`）、nil 检查无落点（`NEW-R06-04`）；见 `:4076` |
| `F004` | HIGH | **闭合（文本层）** | 唯一签名、非可选参数语义、`:3603` 就地修订、39／17 点清单均在 `:3603`／`:4073`；实现未做 |
| `NEW-R05-01` | HIGH（并入 F006） | **未闭合** | 随 `F006` |
| `NEW-R05-02` | MEDIUM | **未闭合，同类复发** | `NEW-R06-02` |
| `NEW-R05-03` | LOW | **基本闭合（残 LOW）** | `rg 'V8.*r65[.]2' design.md` = 0；r66.0／r66.7 已声明历史；残 `NEW-R06-07` |
| `NEW-R05-04` | MEDIUM | **未闭合，同类复发** | 39 点清单仍不覆盖 `recoveryRootInputs` 字面量，见 `NEW-R06-06` |
| `NEW-R05-05` | LOW | **闭合** | `:4290` 起点与段哈希正确、旧 `:4247` 标签作废已在 `:4080` 登记 |
| `F018` | HIGH | 维持 R05 的 PASS | 本轮未见回退；不重复计为闭合项 |

另记（不影响本对象判定）：`framework.md:1084`／`:1129` 判 r09 未复评、门禁 blocked，`tasks.md:3197` 称 r07 已接受、门禁解除；r66.9 对该冲突的登记与实测一致，且作者未改 ADR／tasks。该冲突属架构决策方与 Task-Planner，不由本报告改判。

## 3. 本轮 findings

### NEW-R06-01（HIGH）— 评审对象在评审期间漂移，冻结前提失效

- **行锚**：`design.md` 全文 B/SHA；`:4072`–`:4083`。
- **事实/原句**：冻结基线 1,078,886 B／`0BA1C6C1…`；15:19:22 实测 1,079,559 B／`9B193BDE…`；mtime 15:18:54。r66 块（含末行 LF）30,936 B／`6E25E9CD…` → 31,141 B／`587471F5…`（+205 B）。
- **判据**：任务停止条件（对象 B/SHA 变化即停止）；r66.8「报告与 hashes 必须在最后一次写盘之后生成」。
- **触发场景**：任何按 r06 证据复算、按行锚实施或按本报告接受的人。
- **影响**：作者 15:10 生成的证据已不覆盖现行文件；段哈希与行锚可能再次失效；本轮不能对现行字节给出接受结论。
- **修复方向**：冻结对象、停止并行写盘；由作者说明 15:18:54 这次写入的范围与原因，按终值重算 `report.md`／`hashes.txt`；若引入了新修订内容，升级修订号并重新走两路评审。
- **验证方式**：交付前后 `Get-FileHash`、B、行数一致；评审窗口内多次采样无变化。
- **现行复核**：漂移已确认；尾段三段哈希未变，漂移集中在 r66 块（+205 B）与 `:1`–`:4071`／`:4084`–`:4258` 区域（其中 +468 B 已由作者新版报告归到 `:3603`，见 §0.3）。

### NEW-R06-02（MEDIUM）— 字节账与作者证据链不可复算，`NEW-R05-02`／`NEW-S05-03` 同类复发

- **行锚**：作者 `report.md:15`、`:16`、`:18`、`:47`、`:314`；`hashes.txt` §1／§2／§3；`design.md:4072`–`:4083`。
- **原句**：`report.md:18`「终值 − 本段首测 = **+1,284 B**……因此除这 10 处外没有其它字节被改动」；`:47`「r66 块插入占 12 行」；`:314`「`design.md`：1,077,602 B／5,476 行」（把首测值用作现行值）；`hashes.txt` §3「21,142 B／205 行／`A03CE62B…`」。（以上引文均为 15:19 读取的版本；作者已于 15:20:55／15:24:19 重写，见 §0.3。）
- **判据/复算**：
  - r65→首测：1,077,602 − 1,045,111 = **+32,491 B**；12 行块 30,936 B 只解释其中 30,936 B，**余 1,555 B 未登记**。
  - 「12 行块 + 10 处替换」合计 30,936 + 1,284 = 32,220 B ≠ 32,491 B，**缺 1,555 B** ⇒ 不构成 r65→r66 的全部变更。
  - r05 进入值 1,065,941 不可重放：1,045,111 + 30,936 = 1,076,047，与 1,065,941、1,077,602 均不等。
  - 作者 `hashes.txt` 自记 `report.md` = 21,142 B／205 行／`A03CE62B…`；实际 30,201 B／327 行／`4AC8FEDD…`（报告在哈希生成后又被追加）。
  - 现存临时快照 `%TEMP%\design-pre-fd-r06.md` = 1,074,911 B／5,476 行／`58EC7D8D…`（15:00:51），与「首测」1,077,602／`EFBFDDF0…` 不符 ⇒ 首测值没有任何可复现快照。
  - 作者终值 1,078,886 ≠ 评审时点文件 1,079,559（`NEW-R06-01`）。
- **触发场景**：任何人按报告复算、按报告行锚实施、或以报告数值作为接受依据。
- **影响**：「除这 10 处外没有其它字节被改动」只在「首测→终值」小段成立，被报告表述成整段变更叙事；1,555 B 的未登记改动使 r65→r66 的变更集不完整；作者当时提供的中间/终值数值不可作为接受依据（终值一致性已在作者 15:24 版修复）。
- **修复方向**：保留 r65／首测／终值三份可复现快照；给出逐行 diff（含新增块每行字节数）与增删字节合计；报告、hashes、文件三方引用同一终值；`§8.4` 改终值；删除重复章节（15:24 版已完成这两项）。
- **验证方式**：逐行 diff 的增删行与字节差合计等于 B 差；`report.md`、`hashes.txt`、`design.md` 三方 SHA 一致。
- **现行复核（15:24 后）**：作者已重写 `report.md`／`hashes.txt`，重复章节与 `:314` 旧值消失，终值改为 1,079,559／`9B193BDE…`，与现行文件一致；但按 §0.3 复算，r65→r66 仍有 1,555 B 未登记，§2 增量为句段级口径，账仍不平 ⇒ 维持 MEDIUM。

### NEW-R06-03（MEDIUM）— `checkOwnership` 按 r66.4 字面接入会自锁

- **行锚**：`design.md:4076`（r66.4 第 (1) 点）；`internal/recovery/runtime.go:1160`–`:1177`（`:1161` `r.mu.Lock()`、`:1162` 读 `r.exclusive`、`:1164` `Unlock`）。
- **原句**：「`checkOwnership`（`:1160`–`:1177`，现读 `:1162`）改用 `activeExclusive()`」。
- **判据**：同一节把 `activeExclusive()` 定义为「自取 `r.mu`」的访问器；`:1162` 处于 `r.mu` 持锁区间；Go `sync.Mutex` 不可重入。
- **触发场景**：`CheckPhase`、`ReadPermitBinding`、`acquireReady`、`CheckArticleCoordination`、`readControlled` 任一调用 `checkOwnership`。
- **影响**：按字面落地即全局死锁（测试超时）；若实施者自行改用 `activeExclusiveLocked()`，则偏离设计文本，属未定义选择，评审无法据文本判定实现正确。
- **修复方向**：明确 `:1162` 使用 `activeExclusiveLocked()`（调用方已持锁），或把读取移到 `:1164` 解锁后使用 `activeExclusive()`；补机核判据「`checkOwnership` 函数体内不出现自取锁访问器」。
- **验证方式**：实施后 `go test ./internal/recovery -run 'Admission|Phase|Permit' -timeout 60s`；静态检查该函数体的访问器选择。
- **现行复核**：`:4076` 该句与 `runtime.go:1160`–`:1177` 结构均未变。

### NEW-R06-04（MEDIUM）— `recoveryFlow.acquire` 的 nil 检查在现有调用图上没有「状态变更之前」的落点

- **行锚**：`design.md:4076` 第 (7) 点；`runtime.go:1519`／`:1525`／`:1530`（Enter）、`:1539`／`:1542`／`:1547`（Resume）、`:1554`–`:1571`（`acquire`，`:1556` `StopAndJoin`）。
- **原句**：「`recoveryFlow.acquire`……nil → 在 `beginAction`／任何状态变更之前返回 `ErrAdmissionClosed`」。
- **判据**：`f.acquire` 只在 `Enter`／`Resume` 的 `beginAction`（`:1525`／`:1542`）之后被 `f.enter`／`f.resume` 调用；`acquire` 首步是 `Lifecycle.StopAndJoin`（`:1556`，已有副作用）。设计未给出 entry 级预检位置。
- **触发场景**：装配未注入 lease（typed nil）时的 `enter`／`resume`。
- **影响**：要求不可实现——检查若留在 `:1562` 附近，既晚于 `beginAction` 又晚于 `StopAndJoin`；实现者只能自行选择落点，评审无法据文本判定「任何状态变更之前」是否满足。
- **修复方向**：把 closed/nil 检查上移到 `recoveryPreflight`（`:1519`／`:1539`）或 entry 级预检，并写明 `StopAndJoin` 必须在检查通过之后；给机核判据（nil 注入下 `beginAction` 未被调用、`StopAndJoin` 计数为 0）。
- **验证方式**：注入 typed nil 的单测断言返回 `ErrAdmissionClosed`，且状态/计数无变化。
- **现行复核**：`:4076` 句与 `runtime.go:1517`–`:1571` 调用顺序均未变。

### NEW-R06-05（MEDIUM）— `recovery_restore_test.go:204` 的 `f.world.lease` 早于 `f.world` 初始化

- **行锚**：`design.md:4073`；`cmd/han-server/recovery_restore_test.go:204`、`:208`、`:209`。
- **原句**：「`:204` 的构造器调用增第五实参 `f.world.lease`」。
- **判据**：`:208` 才执行 `f.world = newS414World(p)`；`:204` 时 `f.world` 仍为 nil。
- **触发场景**：按设计实施 `s414Fixture` 构造。
- **影响**：按字面实施即在 fixture 构造期 nil deref（panic），测试面不可运行；「39 点清单 + 构造器第五参数」的测试面闭合因此不成立。
- **修复方向**：把 `f.world = newS414World(p)` 移到构造 driver 之前，或先取 `lease := ...` 再构造；同步更新 `:209`／`:233` 的说明与行锚。
- **验证方式**：`go test ./cmd/han-server -run 'S414|Restore' -timeout 120s`。
- **现行复核**：`:4073` 句与 `:204`／`:208` 顺序均未变。

### NEW-R06-06（MEDIUM）— `recoveryRootInputs.restore` 工厂改造漏列测试字面量（`NEW-R05-04` 同类）

- **行锚**：`design.md:4076`（工厂句与「测试面」）；`cmd/han-server/recovery_runtime.go:441`；`recovery_enter_install_integration_test.go:232`；`recovery_production_chain_integration_test.go:2999`。
- **原句**：「`recoveryRootInputs.restore`（`recovery_runtime.go:441`）改为工厂 `restoreDriverFactory func(recovery.ExclusiveLease) (recovery.RestoreDriver, error)`」；「**测试面**：全仓 `NewRuntime(` 39 处／17 文件全部补第三实参……」（未列 `recoveryRootInputs.restore` 字面量）。
- **判据**：现有字面量直接赋 driver：`restore: s421Restore{}`（`:232`）、`restore: world.restore`（`:2999`）；字段类型改为工厂后这两处编译失败。`main.go:634` 与 `recovery_runtime_test.go:731` 不设该字段，依赖 `:596`–`:599` 的 nil 占位，工厂化后该路径需保留 closed 语义。
- **触发场景**：按设计实施 `recoveryRootInputs` 字段类型变更。
- **影响**：`cmd/han-server` 测试包编译失败；39 点清单只覆盖 `NewRuntime` 调用，不覆盖编译面，与 `NEW-R05-04` 同类。
- **修复方向**：写集/测试面补列这两处字面量（改为工厂闭包）、`main.go`／`recovery_runtime_test.go` 的 nil 工厂语义，以及 `:596`–`:599` 的 closed 占位改法；给出 `rg -n 'restore:\s' cmd` 的期望命中。
- **验证方式**：`go build ./...`、`go vet ./cmd/han-server`。
- **现行复核**：`:4076` 文本与两处字面量均未变。

### NEW-R06-07（LOW）— `:3942`／`:3990` 末句仍只列 r64.5／r65.5，未含 r66.2

- **行锚**：`design.md:3942`、`:3990`。
- **原句**：`:3942`「现行 P1–P6／R1–R8／V1–V11 只以 r64.5／r65.5 为准」；`:3990`「现行 V1–V11、8 字段验证方签名载荷与 head 形状只以 r65.5／r65.2 为准」。
- **判据**：r66.2 已重绑 V7–V11 的载荷与签名主体；r66.0 给优先级可判，但同句前半已写 `r66.2`、末句仍是旧口径，句内自相矛盾。
- **触发场景**：实施者按末句读取 V 编号权威来源。
- **影响**：可能忽略 r66.2 的绑定，回到 r65 载荷/签名主体。
- **修复方向**：末句改为 `r65.5／r66.2`（`:3942` 保留 r64.5 作为 P/R 编号来源）。
- **验证方式**：`rg -n '只以 r64.5／r65.5 为准' design.md` = 0。
- **现行复核**：两处末句均未变。

### NEW-R06-08（LOW）— 作者报告结构重复与旧值残留

- **行锚**：作者 `report.md:179`／`:191`／`:198` 与 `:207`／`:216`／`:260`／`:279`；`:314`。
- **事实**：同一报告出现两套 §5–§8；`§8.4` 用 1,077,602 作为 `design.md` 当前值，与 `§0:16` 的 1,078,886 矛盾。
- **判据**：r66.8 要求报告与 hashes 在最后写盘后生成且数值一致。
- **影响**：读者可能采信旧值/旧结论；证据链可读性下降（与 `NEW-R06-02` 同簇）。
- **修复方向**：删除重复段；`§8.4` 改用与 `§0` 一致的终值。
- **验证方式**：`rg -c '^## 5 ' report.md` = 1；`1,077,602` 只出现在「首测」行。
- **现行复核（15:24 后）**：作者重写版（24,041 B／`CA5F28AC…`）无重复 §5–§8、无 `1,077,602` 残留（`rg` 实测 0）；本条只在 15:19 的快照版本上成立，现记为**已消失**，不要求作者再修；编号仅保留以追踪「证据在评审期间被重写」这一事实（并入 `NEW-R06-01`）。

## 4. 已闭合／基本闭合项与复算命令

### 4.1 `F004`（HIGH）— 闭合（文本层）

- `:3603` 已就地写入 r66.1：`restoreLease` 是 `exclusive` 一次 `Acquire` 的产出值、非可选入参、不进 11 端口表；新增端口/替换来源仍须先改本节与 ADR。
- 唯一签名与 39 点清单见 `:4073`；逐文件计数与作者清单一致（`runtime_test.go` 12 = 列 11 + `:1658`）。
- 复算：`rg -n 'NewRuntime\(' --glob '*.go'` = **39 行**；`rg -l 'NewRuntime\(' --glob '*.go'` = **17 文件**；`rg -c 'NewRuntime\(' --glob '*.go'` 逐文件为 3/1/1/2/3/1/1/1/1/1/1/1/3/3/1/12/3。
- 定义与生产：`rg -n 'func NewRuntime\(' internal/recovery/runtime.go` = `:729`（仍为 2 参数，待改）；`rg -n 'NewRuntime\(' cmd/han-server/recovery_runtime.go` = `:641`（生产唯一）；`rg -n 'NewRuntime\(ctx, deps, restoreLease\)' cmd` = 0（代码未实施，符合「设计层闭合」）。
- 注意：`F004` 闭合不等于实现正确；`NEW-R06-05`／`NEW-R06-06` 仍使测试/编译面不闭合。

### 4.2 `NEW-R05-03`（LOW）— 基本闭合

- `rg -n 'V8.*r65[.]2' design.md` = **0**；`:3942` 已把旧 V1–V10 改为「旧步骤一…十」并标历史；`:3990` 标历史并指回 V10；`:4062` 的 V7–V11 引用 `:3988` producer envelope。
- `driver.Bind`：`rg -n 'driver[.]Bind' design.md` = **5 行／6 次**（`:3997`、`:4013`、`:4029`、`:4051`、`:4069`），r66.4 声明全部历史、语义无效；`rg -n 'func .*Bind\(' --glob '*.go'` = 6 处，全部为测试函数名，无方法定义。
- `sequence_floor`：`rg -c 'sequence_floor' design.md` = **10 行**；`rg -o 'sequence_floor' design.md | Measure-Object` = **19 次**；命中落在历史段/旧键拒绝语境，r66.7 有统一历史声明。
- 残 LOW：`NEW-R06-07`。

### 4.3 `NEW-R05-05`（LOW）— 闭合

- 段哈希（raw，含 LF/CR，至 EOF）：`:4259` = 282,599 B／`7D1DDC087A496FC2C02C26E5F04017F8C6900A19E20D596CF5A4F40E9150F05A`；`:4275` = 280,319 B／`5BD8E87093A27AD40ED5A8DD056D56CE14160DB30A88DCEA26323DAC7E23B562`；`:4290` = 277,334 B／`E40EB7A29AA909A1C8A2918F3A8FCF1C2013793A401C6046D8C276DE79A33254`；`:4435` 单行 = 364 B／`429E633A6B13D5B26E851FF4F3459E7A1A33246A706F2CF30431235207FFE1A5`。
- golden：`:4099` = 1,115 B／`DE914622D3F8E90724D6C6FA83E0BD40D98E6510E2BBCF3BD592077C2E7B0CAB`；`:4105` = 2,357 B／`8C0C89E3DAF0E024F266FF99443A6B54A2393D177D6CFAED27D21B894A326508`。
- `:4080` 已把旧 `:4247` 标签作废、确认 r61 真标题为 `:4290`；本轮复算一致。

### 4.4 必核项 C 的边界（未发现把 unknown 写成已具备）

- `nonce`：`rg -n '\bnonce\b' design.md` = **2**（`:3973`、`:4075`）；定性为进程内错误关联标识，`crypto/rand` 16 B、单次失败复用、成功不生成；不落盘、不进 Journal/审计/DB/成功回执/`restoreReceiptWire`/producer envelope/DSN/argv/ENV；未改变 wire 形状。**未发现冲突**。
- `bundle_digest`：`rg -n 'bundle_digest' design.md` = **2**（`:4075`、`:4082`）；显式 OPEN、首期不实现、不定义字段/算法。**未发现写成已闭合**。
- `SEC-F007`：`rg -n 'SEC-F007' design.md` = **3**（`:4021`、`:4082`、`:4191`），均为 NOT RUN，无 PASS。**保持 NOT RUN**。
- 访问器计数（现行字节）：`activeExclusive` 5、`activeExclusiveLocked` 2、`ownedExclusive` 19、`injectedExclusive` 38。
- `recoveryFlow.acquire` 的 `enter` teardown 顺序（`Stop` → `waitFlow` → `closeRestoreLease`、`sync.Once`、`injectedExclusive` 永不关闭）与 r66.10／r66.11 的不变量声明**语义上不冲突**；缺陷在落地位置，见 `NEW-R06-04`。

## 5. 未运行项

- 未运行 `go build ./...`、`go test ./...`、`go vet`：作者未改 Go 代码，本评审为设计对象只读评审。
- 未编译/运行任何按 r66 改造后的代码；`F004` 的「实施后」机核判据全部未执行。
- 无 E2E、无 Linux host／ACL／mount／loader／fd-exec／maps／network／witness 实测；`SEC-F007` 维持 NOT RUN；`NoNewPrivs`／`CapBnd` 仍只是观测值。
- 未重放 r65→r66 全量 diff：无 r65 完整快照（HEAD 为 2026-10-21 旧版；`%TEMP%` 仅有前缀备份与中间快照）。
- ADR-0031 侧不在本对象内：`framework.md` 读取时为 `adr-0031-r09`、门禁 blocked；并行 r10 可能修改；本报告不因 ADR 变化改判，也不对 ADR 给结论。
- 未运行任何 git 写操作；未修改 `design.md`、`tasks.md`、`framework.md`、`plans/**`、Go 代码或既有报告。

## 6. 越界检查与只读声明

- 越界内容扫描：`rg -n 'r66|restoreLease|activeExclusive|ownedExclusive|injectedExclusive|closeRestoreLease|bundle_digest|restore_nonce' framework.md tasks.md plans --glob '*.md'` = 0；Go 代码同组标识 = 0；`framework.md` 的 `NewRuntime` = 0。mtime：framework 12:19:02、tasks 09:21:19、plan 08:48:33，均早于作者证据 15:10，但 mtime 不能单独证明未改。15:23:48 复测 framework 已变为 285,639 B／`B432E44A…`（并行 r10 写入，非本轮作者证据）；ADR 侧结论不在本对象内，本报告不因其改判；tasks／plans 复测同值。
- 作者报告（15:19 版）§8「越界需求（本轮未处理）」／（15:24 版）§4「未闭合／升级待决」与 r66.9「不改 ADR、不改 tasks」一致；**未发现把 ADR 层面待办当作本轮已闭合**。
- 本报告非真人签字，不构成人的成果接收或批准。
- 不解锁任何卡；未勾选 `tasks.md` 任何复选框。
- 不解除生产 closed；不构成 `enter`／`resume` 解锁；不构成 `identity-auth-S4.24a` 可启动或可关闭；不构成约束 8 满足。
- 安全侧 `SEC-F003`／`N001`／`N011`／`NEW-S05-*` 的判定不由本报告给出或改判。
- 本评审只写 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r06/report.md` 与同目录 `hashes.txt`。

## 7. 复审条件

1. 冻结 `design.md`，由作者说明 15:18:54 写入范围；作者已按终值重算 `report.md`／`hashes.txt`（15:24），仍欠逐行 diff 与 1,555 B 缺口的解释。
2. 修复 `NEW-R06-03`／`NEW-R06-04`（F006 残余）与 `NEW-R06-05`／`NEW-R06-06`（测试/编译面）。
3. 处理 `NEW-R06-07`／`NEW-R06-08`。
4. 由架构与安全两路独立复审重新判定；本报告不替代安全侧结论，也不替代人的接受。