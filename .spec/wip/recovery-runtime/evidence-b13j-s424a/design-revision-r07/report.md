# identity-auth 系统设计 r67 作者报告（B13J / S4.24a）

## 0. 对象与改前改后实测

- 对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`（系统设计，非特性设计）。
- 工作区：`D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abb`；本轮无任何 Git 写操作，未删除或移动既有文件。
- 写入次数：`design.md` 本轮**只写一次**；最后一次写盘后仅做只读测量与快照，之后未再修改。

| 节点 | B | 行 | SHA256 | CR | CR-LF | bare-LF | BOM |
|---|---:|---:|---|---:|---:|---:|---|
| r66（改前，= 快照源） | 1,079,559 | 5,476 | `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298` | 169 | 169 | 5,307 | 无（首字节 `2D 2D 2D`） |
| r67（改后，现行） | 1,084,882 | 5,476 | `B12F1EF3E555BF61AB52826B96B495E06535D195600429AB775351B2C34E064E` | 169 | 169 | 5,307 | 无（首字节 `2D 2D 2D`） |
| Δ | **+5,323** | **0** | — | 0 | 0 | 0 | — |

- 行数恒等：169（CR-LF）+ 5,307（bare-LF）= 5,476，与 r66 完全一致，**未新增或删除物理行**。
- 修订号双处登记：`:5` = `revision: identity-auth-design-ir01-r67`，`:38` = `**设计版本**：` + `` `identity-auth-design-ir01-r67` ``；两行 r66→r67 字节数不变（仅版本串升一位），故不计入下文字节表。
- 既有 169 个 CRLF 行逐字保留；本轮新增文本全部写入既有 LF 行内部，未做整篇归一化。

## 1. r66 基线快照（开工第一步）可复算证明

- 快照：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r07/design.r66.baseline.md`，按原始字节复制，非文本重写。

| 校验项 | 源 `design.md`（写前） | 快照副本 | 结果 |
|---|---:|---:|---|
| B | 1,079,559 | 1,079,559 | 相等 |
| 行（LF） | 5,476 | 5,476 | 相等 |
| CR | 169 | 169 | 相等 |
| SHA256 | `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298` | 同左 | 相等 |
| `SequenceEqual` | — | — | **True** |
| BOM | 无 | 无 | 相等 |

复算方式（只读，PowerShell，与本轮实测同法）：

```powershell
$a=[IO.File]::ReadAllBytes('…/design.md')
$b=[IO.File]::ReadAllBytes('…/design.r66.baseline.md')
[Convert]::ToBase64String($a) -eq [Convert]::ToBase64String($b)   # True 即逐字节相等
[BitConverter]::ToString([Security.Cryptography.SHA256]::Create().ComputeHash($b)).Replace('-','')
```

独立佐证：仓库外副本 `%TEMP%\design-r66-freeze-9B193BDE.md` 与上述字节相同（只读，未纳入写集）。快照在 `design.md` 首次写盘之前生成并核验，故可作为唯一可重放的比对基线；本报告第 3 节全部字节表均由 `design.r66.baseline.md` 与快照/备份逐行实测得出。

## 2. 逐项闭合（R06 两路复审）

### 2.1 `F006` BLOCKER（架构侧）——ADDRESSED

**问题**：r66.4 要求 `checkOwnership` 改用 `activeExclusive()`，但该函数 `:1161` 已持 `r.mu`、`:1162` 读 `r.exclusive`，直接调用会 `sync.Mutex` 自锁；`recoveryFlow.acquire`（`runtime.go:1554`）声明的 nil 检查位于 `beginAction` 之后，而 `:1556` 的 `StopAndJoin` 先于任何 lease 读取，检查无落点。

**只读代码事实**（`internal/recovery/runtime.go`）：`:1161` `r.mu.Lock()`；`:1162` 读 `r.exclusive`；`:1163` 构造 `generationCtx`；`:1164` `r.mu.Unlock()`。`recoveryPreflight` 位于 `:1497`–`:1514`，端口检查在 `:1504`–`:1512`，`:1513` 为 `return nil`。`Enter` 预检在 `:1519`、`beginAction` 在 `:1525`；`Resume` 在 `:1539`／`:1542`；`recoveryFlow.acquire` 在 `:1554`–`:1571`，`StopAndJoin` 在 `:1556`。

**唯一可实现合同**（已写入 `design.md:4076` 第(1)点与第(7)点，无两可表述）：

1. 两条访问器路径：`activeExclusiveLocked() ExclusiveLease` **不自取锁**，调用方必须已持 `r.mu`；`activeExclusive()` 自取 `r.mu` 后**委派**给 `activeExclusiveLocked()`。二者都只按 `r.action` 选择 `ownedExclusive`／`injectedExclusive`，返回 nil 一律表示 closed。
2. 调用点与持锁前提：`checkOwnership` 在 `:1161` 已持锁的区间内，`:1162` 必须写为 `lease := r.activeExclusiveLocked()`，与 `:1163` 的 `generationCtx` 属**同一次持锁快照**，`:1164` 解锁；**禁止在该函数体内调用 `activeExclusive()`**。
3. 解锁后的判定：injected 动作且 `lease == nil` → `ErrAdmissionClosed`；owned 动作且 `lease == nil` → 沿用 `checkExclusive` 的 `ErrAdmissionBinding`（`admission.go:405`–`:406`）。因 `checkOwnership` 的调用者（`:1192`／`:1196`、`:1241`／`:1245`、`:1123`、`:293`、`admission.go:617`）都经它，无需逐点补锁。
4. nil lease 的唯一落点：`recoveryPreflight`（`:1497`–`:1514`），在 `checkedContext`、binding 文本、`:1504`–`:1512` 全部端口检查**之后**、`:1513` `return nil` **之前**检查 `injectedExclusive`（含 typed nil，复用 `missingRuntimePort`）→ `ErrAdmissionClosed`。该点严格早于 `Enter`／`Resume` 的 `beginAction`（`:1525`／`:1542`）与 `recoveryFlow.acquire` 的 `Lifecycle.StopAndJoin`（`:1556`）。`recoveryFlow.acquire`（`:1562`／`:1564`／`:1570`）删除 `deps.Exclusive.Acquire` 与 `f.r.exclusive = l`，只读 `injectedExclusive` 做与 `checkExclusive` 等价校验，**不再承担 nil 判定**；`StopAndJoin` 只在 preflight 通过后执行。
5. 机核判据：`checkOwnership` 函数体内 `activeExclusive(` 出现 0 次、`activeExclusiveLocked(` 出现 1 次；typed nil 注入下 `beginAction` 未调用、`StopAndJoin` 计数为 0、返回 `ErrAdmissionClosed`。

未采用「降级为 fail-closed」作为替代：自锁可通过不自取锁的访问器消除，且 nil 判定已有更早的合法落点；fail-closed 仅作为上述判定失败时的结果语义保留。

### 2.2 `NEW-R06-02` MEDIUM（字节账）——ADDRESSED

见第 3 节：r65→r67 全量按行配平，**1,555 B 残余已被 16 行整行表逐行吸收**（`3,307 − 468 − 1,284 = 1,555`），不再存在未解释字节；`:4072` 自述的就地指针（`:3942`／`:3990`／`:4062`／`:4187`）已给出逐行整行字节数。

### 2.3 `NEW-R06-05` MEDIUM ——ADDRESSED

`design.md:4073` 与 `:4076` 测试面均改为固定顺序：先 `f.world = newS414World(p)`（`recovery_restore_test.go:208`），再 `newControlledRestoreDriver(f.source, f.target, f.tool, s414Canary, f.world.lease)`（原 `:204`），并删除 `:209`／`:233` 的 `ownership` 字段赋值；`NewRuntime` 仍为 `:228`。文中明写「`:204` 不得早于 `:208`」，第五实参取自已初始化的 `f.world` 且非 nil（`recovery_restore.go:216` 要求 `d.ownership` 非 nil）。清单与顺序已一致。

### 2.4 `NEW-R06-06` MEDIUM ——ADDRESSED

`design.md:4076` 测试面补齐：`recoveryRootInputs.restore` 字段名不变、类型改 `restoreDriverFactory`；**`recovery_enter_install_integration_test.go:232`（`restore: s421Restore{}`）与 `recovery_production_chain_integration_test.go:2999`（`restore: world.restore`）必须改为工厂闭包**；`main.go:634` 与 `recovery_runtime_test.go:731` 依赖 `:596`–`:599` 的 nil 工厂 → `recoveryClosedRestoreDriver`。机核：改后 `rg -n 'restore:\s' cmd/han-server` 只命中上述两处工厂赋值（本轮只读实测：当前恰好为这两处）。

### 2.5 `NEW-R06-07` LOW ——ADDRESSED

`:3942` 与 `:3990` 的权威指向改为含 r67 的唯一表述：「只以 r64.5／r65.5 及 r66.2 为准（r67.2 再确认：V7–V11 的载荷与签名主体唯一权威为 `:4062` 与 `:4074`；r67 不改变本条指向）」。同句前部已含 r66.2，故不再与「只以 r64.5／r65.5 为准」自相矛盾；`:3990` 同构（`r65.5／r65.2` 及 `r66.2`）。

### 2.6 `NEW-S06-02` LOW ——ADDRESSED

`V10` 推进见证 head 前显式重验：`:4062`（V10 定义）写明「在外部见证**原子推进** `sequence` 之前再调用一次 `revalidateApproval`（r67.5：重读批准文件、信任根与 `receipt_anchors`，重算 `generation` 三元摘要并核对有效期与撤销；任一失败 → `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`，由调用方按 `restore_unknown` 处置，**不得推进 head**）」；`:4135` 其六、`:4137` 尾句给出同一判据（含撤销／`valid_until`／`generation` 漂移与「见证 head 不得推进」）。

### 2.7 `NEW-S06-03` LOW ——ADDRESSED

`:4369` 与 `:4373` 给出 `generation` 的可核定义（非新增字段，而是三元摘要）：`approval_generation` = 批准文件精确字节 SHA-256（预检快照记录、`revalidateApproval` 重读逐字节比对）；`trust_root_generation` = 信任根文件精确字节 SHA-256，须等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`；`anchor_generation` = `receipt_anchors` 该 `anchor_ref` 条目 JCS 规范字节的 `sha256:` 摘要，须等于 policy 与回执的 `trust_anchor_sha256`；三项逐字节相等，不得以可读／inode／长度替代。与 `framework.md:1118`（ADR-0031 r10，(i)(ii)(iii)）逐项一致；本轮**只读引用，未修改 `framework.md`**（当前实测 B=285,639，SHA256 `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`）。

### 2.8 `NEW-S06-01` MEDIUM ——ADDRESSED

与 `NEW-R06-02` 同源（字节账不可配平即无法核对安全侧「对象被改写」的边界），随第 3 节配平一并闭合；本轮未改变任何安全语义结论。

## 3. r65→r67 全量字节配平

### 3.1 前置事实与口径

- r65 派发声明的进入值：1,045,111 B／5,464 行／`87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9`。**该文件在本机不可得**（全盘无 1,045,111 B 的 `design.md` 历史副本），故 r65 逐行字节为**重建值**，非一手实测。
- 重建依据（两件均可复核）：`%TEMP%\design-r65-prefix.bak.md` = 1,044,640 B／5,464 行／`2BBBDC8410BEE8681DA3F3B648693EADD6CDA1CEC8856B90222F943EDB72DE61`（实测）；`review-arch-r05/report.md` §6.4 记录该备份与 r65 现行文件差 **13 行、+471 B**，逐行为 `:3990` +255、`:3997` +59、`:4013` +22、`:4017` +41、`:4029` −36、`:4051` +64、`:4057` +257、`:4058` +154、`:4060` +133、`:4063` 0（内容变化）、`:4068` −390、`:4069` −85、`:4070` −3。
- 一致性：1,044,640 + 471 = **1,045,111**，与派发值逐字节相等；行数同为 5,464。故「备份 + 13 行调整」可作为 r65 的可复算重建。
- 下文所有 r65 列均标注「重建」；r66／r67 列为 `design.r66.baseline.md` 与现行 `design.md` 的**整行实测**。

### 3.2 r65→r66 行内变更（16 行，整行字节）

| 现行行 | bak 行 | bak B | R05 Δ（bak→r65） | r65 B（重建） | r66 B | Δ（r65→r66） |
|---|---:|---:|---:|---:|---:|---:|
| `:3603` | 3603 | 668 | 0 | 668 | 1,518 | +850 |
| `:3942` | 3942 | 1,874 | 0 | 1,874 | 2,214 | +340 |
| `:3973` | 3973 | 748 | 0 | 748 | 893 | +145 |
| `:3990` | 3990 | 2,725 | +255 | 2,980 | 3,371 | +391 |
| `:3997` | 3997 | 1,882 | +59 | 1,941 | 2,163 | +222 |
| `:4013` | 4013 | 305 | +22 | 327 | 478 | +151 |
| `:4029` | 4029 | 3,784 | −36 | 3,748 | 3,924 | +176 |
| `:4051` | 4051 | 1,151 | +64 | 1,215 | 1,365 | +150 |
| `:4060` | 4060 | 3,164 | +133 | 3,297 | 3,536 | +239 |
| `:4061` | 4061 | 2,510 | 0 | 2,510 | 2,678 | +168 |
| `:4062` | 4062 | 1,606 | 0 | 1,606 | 1,890 | +284 |
| `:4063` | 4063 | 814 | 0（内容变，字节同） | 814 | 941 | +127 |
| `:4069` | 4069 | 3,518 | −85 | 3,433 | 3,616 | +183 |
| `:4090` | 4078 | 1,298 | 0 | 1,298 | 1,392 | +94 |
| `:4155` | 4143 | 1,114 | 0 | 1,114 | 1,228 | +114 |
| `:4187` | 4175 | 1,815 | 0 | 1,815 | 1,488 | **−327** |
| **合计** | | | **+412** | | | **+3,307** |

说明：`+412` 只是 R05 的 13 行中落在上表 16 行内的部分；其余 5 行（`:4017` +41、`:4057` +257、`:4058` +154、`:4068` −390、`:4070` −3 = +59）不在 r66 的就地修订范围内。`:4187` 为**负增长**，故旧口径把 r66 变更一律记作「插入字节」本身不成立。

### 3.3 r66 插入块（12 行）

| 范围 | 行数 | B | SHA256 |
|---|---:|---:|---|
| `:4072`–`:4083`（含末行 LF） | 12 | **31,141** | `587471F58DB925A17D4E3F06AFE981244E9D2120B969DB615AC5DE2BED749048` |
| 同范围（不含末行 LF） | 12 | 31,140 | `38BDE4DE601D068C26C45C0F276CE949BB5E2CBB5F8C86B7C7AB25EF49DCB74A` |

该块使 `:4072` 之后全部行号 +12，因此上表三行（`bak:4078/4143/4175` → `:4090/4155/4187`）才有位移。

### 3.4 r66→r67 行内变更（10 行，整行字节）

| 行 | r66 B | r67 B | Δ |
|---|---:|---:|---:|
| `:3942` | 2,214 | 2,344 | +130 |
| `:3990` | 3,371 | 3,501 | +130 |
| `:4062` | 1,890 | 2,234 | +344 |
| `:4072` | 3,155 | 4,280 | +1,125 |
| `:4073` | 4,837 | 5,023 | +186 |
| `:4076` | 8,096 | 10,088 | +1,992 |
| `:4135` | 1,028 | 1,312 | +284 |
| `:4137` | 914 | 1,165 | +251 |
| `:4369` | 157 | 248 | +91 |
| `:4373` | 665 | 1,455 | +790 |
| **合计** | | | **+5,323** |

机核方式：对两份文件按 `\n` 切段后逐段比较字节长度，r66→r67 **恰好 10 段**不等，与上表一一对应，无其他段变化；段数均为 5,477（= 5,476 行 + 尾段），故行数零漂移。

### 3.5 恒等式与残余字节

```
r65→r66：3,307（16 行行内） + 31,141（插入块，含 12 个 LF） = 34,448 = 1,079,559 − 1,045,111   ✓
r66→r67：5,323（10 行行内，Δ0 行）                                              = 1,084,882 − 1,079,559   ✓
r65→r67：34,448 + 5,323 = 39,771 = 1,084,882 − 1,045,111（Δ 行 +12）                              ✓
备份口径：471（R05 13 行） + 34,448 = 34,919 = 1,079,559 − 1,044,640                             ✓
```

**1,555 B 残余的处置**：r65→r66 旧登记为 `31,141 + 468 + 1,284 = 32,893`，与 34,448 差 **1,555**。原因不是「漏了某段字节」，而是**计量粒度不一致**：468 只计 `:3603` 中 r66.1 那一句的增量、1,284 只计 10 处就地指针的**片段**增量，二者都不是整行字节；而 `:4187` 为 −327、`:4069` 为 +183 等负向/整行效应无法由「片段插入」口径表达。改用整行口径后 `3,307 − 468 − 1,284 = 1,555` 被 16 行表逐行吸收，**残余为 0（按整行）**。旧 `468`／`1,284` 数字仅作片段级历史，不再参与配平；也不再用「其余字节未动」替代配平。

### 3.6 完整性检验（无其他漂移）

对 `design.r66.baseline.md` 与 `%TEMP%\design-r65-prefix.bak.md` 做**内容级**逐行比对（插入点之后按 +12 位移对齐）：

- 插入点之前：20 行内容不同 = `:5`、`:38`（版本串 r65→r66，等字节）+ 上述 16 行 + R05 的 `:4017`／`:4057`／`:4058`／`:4068`／`:4070`。
- 插入点之后：3 行内容不同 = `bak:4078`／`4143`／`4175`（对应 `:4090`／`:4155`／`:4187`）。
- 合计 23 行，恰为「2 行版本串 + 16 行 r66 行内 + 13 行 R05」去重后的并集；**除此之外无等长内容漂移**。

## 4. 未闭合项与残余风险

| 项 | 状态 | 说明 |
|---|---|---|
| ADR-0031 r10 要求 `V7`／`V8` 前各一次重验 | **OPEN** | `framework.md:1118` 明写 `V7`／`V8` 前的重验是 design r66 既有 V 图（重验节点 V2／V4／V6）之外的**追加关闭判据**。r67 只补 `V10`（r67.5），未在 design 侧镜像 `V7`／`V8`；补它会改动 V 编号契约与 ADR↔design 对账，超出本轮写集，故保留 OPEN，**不声称已具备**。 |
| ADR 内指向 `identity-auth-design-ir01-r66` | **OPEN（跨文件陈旧指针）** | `framework.md:1118` 三处仍写 r66；r67 升级后成为陈旧指针。ADR 属他人所有（只读），本轮未改。 |
| r65 一手文件不可得 | **残余风险** | 第 3.2 节 r65 列为重建值（备份 + R05 已登记 13 行）；若日后取得原始 r65，应以其复算替换重建列。 |
| r67 子编号定义位置 | 已缓解 | 子编号定义写在 `:4072`（r67.0），目标行仍保留 r66.x 标题，由 `r67 > r66 > …` 优先级规则覆盖；物理无新增节，行号零漂移。 |
| `F017`／`N005` | **OPEN（范围外）** | 仍按 r66.9 登记，未在本轮处理。 |
| `SEC-F007` | **NOT RUN** | 安全侧 R06 因对象被改写中止，无安全语义否决，也无安全通过。 |
| `bundle_digest` | **OPEN** | 保持「首期不实现 / 显式 OPEN」，未因 r67 改变。 |
| `nonce` | 不变 | 仍为进程内错误关联标识，未升级为跨进程唯一键。 |

## 5. 未验证声明

本报告不构成任何通过结论：

- 未运行 `go build`／`go test`／`go vet`／任何静态检查；第 2 节中的机核判据均为**待实施后执行的判据**，非已执行结果。
- 未做 E2E、未起真实服务、未接真实数据库或外部见证；Linux 侧能力 **NOT RUN**。
- 无 `PASS`、无评审通过、无真人签字、无生产批准、无解锁声明；`enter` 生产 secret 通路仍不存在，实际进入即 closed。
- 全部结论限于「设计文本与字节账」层面；代码事实仅来自本轮只读核对（`runtime.go`、`admission.go`、`recovery_restore*.go`、两个集成测试与 `rg` 计数）。

## 6. 越界需求

无。本轮未修改 `framework.md`、`tasks.md`、`plans/**`、既有 `review-*/`、`design-revision-r01..r06/`，也未改任何 Go 代码。第 4 节的 ADR↔design 对账（`V7`／`V8` 镜像与 r66 指针更新）若要进行，需由 ADR/架构决策方另立授权，不在本次写集内。

## 7. 写入清单

| 文件 | 动作 | 说明 |
|---|---|---|
| `.spec/…/design-revision-r07/design.r66.baseline.md` | 新建 | r66 快照，字节级原样（见第 1 节） |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | 修改 | r66 → r67，10 行行内修订，Δ0 行（见第 0、3.4 节） |
| `.spec/…/design-revision-r07/report.md` | 新建 | 本报告 |
| `.spec/…/design-revision-r07/hashes.txt` | 新建 | 快照／r67／本报告的 B 与 SHA256；**不含 `hashes.txt` 自身哈希** |

本报告自身 B／SHA256 由 `hashes.txt` 记录（写入后实测，报告未再改写）。