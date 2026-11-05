# B13J 规划评审 r05 — 登记轮 r01 定向评审

- 评审人：add-reviewer（非作者；本项未参与 tasks.md / Plan 编写，未参与 D01–D04 设计与 ADR 修订）
- 对象：B13J 登记轮 r01 的两处写入（tasks.md 六卡、Plan 六处 hunk）及其决策依据（framework.md ADR-0030/0031/0032、r04 评审次序）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮未变）
- 记录时间：2026-11-02（Asia/Shanghai）
- 性质：只判「六卡与 Plan 是否可作为 B13J 实施基线、能否进入 Task 卡认领/START」。不构成实施授权、不登记任何卡、不解除 `blocked-upstream`、不关闭 `pending-B13J-D04`、不是人的签收。

## 1. 结论（verdict）

**CHANGES REQUIRED**

- 六卡 12 项强制字段全部齐备、硬 requires 链与 r04 登记次序逐条一致、诚实性无扩张、mermaid 与槽位计数自洽、无裸 `S4.24` 与未登记编号引用；这些主干均成立。
- 阻断项一条：`S4.24b` 的负向测试 (e)「Windows 或非 POSIX 平台在任何副作用前退出码 2」在其实卡声明的唯一新增测试文件（POSIX-only 构建标签）内**不可执行**，且与本卡停止条件「改未列文件即 STOP」冲突，使关闭条件 (4)(5) 无法诚实满足（`B13J-R05-F002`，HIGH）。
- 另有一条需同批修正：`S4.22` 目标把「ADR-0032 r60」混写（`B13J-R05-F001`，MEDIUM）。
- 三条 LOW 不阻断（`F003` 槽位计数口径、`F004` 登记轮前像与作者证据缺项、`F005` ADR-0032 `:1121` 括注随本轮登记变为历史陈述），可与上述一并修正。
- 处置建议：一次**卡片文本级 r02 修订**（只改 `tasks.md` 六卡内两处 + 槽位/写集声明，不改 ADR、不改 Plan 结构），再做一轮定向复审；通过后才把六卡作为 START 基线。

## 2. 实测输入（本评审自算）

| 文件 | 字节 | 行 | SHA256 | 格式 |
|---|---:|---:|---|---|
| `.add/project/knowledge/tasks/tasks.md` | 855008 | 3222 | `A8205A2196AC77673C37C47545966C89B28AD9ADBBF643611CDD6A2125312852` | CRLF 3222 / bareLF 0 / 无 BOM / 末尾单 CRLF / 0x0C×2 + TAB×1（历史损伤，见 O01/F 注） |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | 504272 | 1651 | `D9A79C54612B90A69462FAA53067CB97B1DE109065633968369A586874CC936B` | CRLF 1651 / bareLF 0 / 无 BOM / 末尾单 CRLF |
| `.add/project/knowledge/decisions/framework.md` | 182488 | 1141 | `39BD3A797B0FA2F5CDF944ACCF4129BB0B7F7A23FABC743B7E5CCC9F7B9BD704` | 本轮未改（ADR-0030 r04 `:918`、ADR-0031 r04 `:978`、ADR-0032 r03 `:1068`） |
| `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r04/report.md` | 21187 | 157 | `38DE303B432921C9A369C08F7BF4DD1999C08EBA72CACF8A99BFDEBE9174E352` | 纯 LF |

登记前 `tasks.md` 基线**由本评审独立复算成立**：取现状前 3034 行、仅还原 4 处统计行（`:46` 76、`:47` 67/2/8、`:49` 49、`:51` 去「决策文档 2；」且「开发 58→54」）后实测 **823187 B / `F1D65B0D7016CDBF9C2F4E85B012B5B0B9F714B6DF0C7CDC9326C05676900E34`**，与登记前记录一致 → 3035–3222 之外（除 4 条统计行）零改动，既有卡的 checkbox/日期/requires/写集/关闭记录未被本次登记改写。

## 3. 必测项结果

### A. 六卡 12 项强制字段
脚本按卡区间逐个检查 12 个字段标签，六卡均 `missing=[]`：目标 / 编号·类别理由 / 类型·Story与范围 / 精确文件清单·写集 / 硬·软 requires / 验收命令 / 负向测试与失败 oracle / 关闭条件 / 证据落点 / 责任人 / 不宣称边界 / 停止条件。六卡均 `- [ ]`、无完成日期。

### B. requires 链与 r04 次序
| 卡 | 行区间 | 硬 requires | 与 r04 次序 |
|---|---|---|---|
| identity-auth-D05 | 3037–3061 | 无（约束 9(a)） | 一致 |
| identity-auth-D06 | 3063–3087 | 无 | 一致 |
| identity-auth-S4.23 | 3089–3120 | `D05` —— 决策修订被架构与安全独立评审接受（约束 9(b)），明示「不是约束 7 全项解除」 | 一致 |
| identity-auth-S4.24a | 3122–3153 | `D06` —— ADR-0031 `:1050`，明示「不是约束 8 全项解除」 | 一致 |
| identity-auth-S4.24b | 3155–3189 | `S4.23` —— bundle 交付且 `Exclusive` 离开 `recoveryClosedPort`（ADR-0032 `:1121`/`:1123`） | 一致 |
| identity-auth-S4.22 | 3191–3222 | `S4.24b`；登记按约束 9(c) 不受约束 7 未满足项阻挡，解除须约束 7 全项 + 人的明确生产批准 | 一致 |

D05/D06 不直连 S4.22，与 Plan `:1342`、ADR-0030 `:961`、ADR-0032 `:1125` 的「经 S4.23/S4.24b 传递」口径一致。

### C. 测试名与写集
- 验收命令引用的既有测试名实存：`TestRecoveryAnchorAdapterRegistry`、`TestRecoverySecretSourceIsLazyCandidate`（`cmd/han-server/recovery_sources_test.go`）、`TestRecoveryRuntimeComposition`（`recovery_runtime_test.go`）、`TestRecoveryRestoreDriver`（`recovery_restore_test.go`）、`TestRecoveryDeploymentExclusive`（`recovery_exclusive_test.go`，无构建标签）、`TestRecoveryConfig`（`recovery_config_test.go`）。
- 引用但尚不存在的 4 个顶层测试名（`TestRecoveryAnchorAdapterProductionBundle`、`TestRecoveryRestoreProductionWiring`、`TestRecoveryDeploymentExclusivePosix`、`TestRecoveryProductionRootAssembly`）全部落在各卡声明的新增测试文件上，属「未创建的新增文件」，不是假引用。
- 声明为修改的源码/测试文件实存：`recovery_sources.go`、`recovery_restore.go`、`recovery_exclusive.go`、`recovery_runtime.go`、`main.go`、`internal/recovery/filelock_flock.go`、`internal/recovery/filelock_unsupported.go`。
- 所有权索引（Plan `:1334`–`:1339`）与六卡写集逐项一致；`rg --pcre2 'S4\.24(?![ab])'` 在 tasks.md 与 Plan 均无命中；六卡引用的所有编号（D05/D06/S4.23/S4.24a/S4.24b/S4.22）均已在本轮登记为正式 Task，无未登记 Task 引用。
- 上游符号引用可核：`newRecoveryAnchorRegistry(nil)` 在 `cmd/han-server/recovery_runtime.go:551`；`inputs.exclusive` 无条件覆盖在 `:565-567`；`HAN_RECOVERY_ANCHOR_ADAPTER`/`..._CONFIG_PATH` 在 `recovery_config.go:67-68`；`HAN_RECOVERY_INITIAL_ADMIN_SECRET` 在 `recovery_config_test.go:46` / `recovery_config.go:159-160` 邻域。

### D. 负向 oracle 可失败性
- D05/D06：以 `Select-String` 断言「已解除/生产 restore 已可用」无命中 + `git status --porcelain -- cmd internal` 逐字节相等，可失败。
- S4.23/S4.24a/S4.22：断言落在本卡写集内文件（`recovery_sources.go`、`recovery_restore.go`、`recovery_runtime.go`、`main.go`）能产生的行为上，可失败。
- S4.24b (e)：**不可执行**（见 `F002`）。

### E. 计数一致性
- mermaid 实测：**41 个节点定义 / 79 条边 / 节点 ID 无重复 / 边无重复**；新增边恰 `n35→n37`、`n36→n38`、`n37→n39`、`n39→n40` 四条，新节点 `n35`–`n40` 与 D05/D06/S4.23/S4.24a/S4.24b/S4.22 一一对应。
- 槽位：所有权索引 6 行反引号路径合计 **22 槽 / 14 唯一文件**（实测），与 Plan `:1311`、`:1641` 的「14 个唯一文件 / 22 个声明槽」一致。
- 增量：6 卡（6 条新节点）/ 4 边；18→24 卡、75→79 边、77→83 节点、329→333 边，与本轮 +6 节点/+4 边一致。全图 83 节点与 `- [` 行实测 83 一致；全图 333 条 requires 边未在本轮独立重算（见 §7）。
- 唯一残留：`tasks.md` `:46` 总数 82 与实测 83、`:49` identity-auth 55 与实测 56 差一 → 观察项 O01（登记前即存在，非本轮引入；`:47` 三态 67+2+14=83 与实测一致）。

### F. 诚实性与结论扩张
- 六卡「不宣称边界/停止条件/关闭条件」中的 `已解锁`/`生产可用`/`已实施`/`已通过评审` 命中**全部是否定式**（「不宣称……已解锁」「不宣称生产可用」），无一处把候选卡写成已通过评审、已实施、已解锁或生产可用。
- 生产 `enter`/`install` 仍 closed、无 file/SQL/fake 自动 fallback 的表述完整保留：`S4.23:3093`（缺一即四端口一起 closed、不得 fallback）、`S4.24b:3159`（无 fallback/无降级）、`S4.22:3195`/`:3218`（生产不得用 `recoveryRootInputs.exclusive` 替代 bundle）。
- `article-management-S8.2`、Case18 未被宣称 PASS（`S4.23:3119`、`S4.24a:3152`、`S4.24b:3188`、`S4.22:3218`/`:3221` 均为否定式）。
- ADR-0030/0031/0032 结论未被放宽：卡片把「登记入口」与「生产解除」分列，未把决策修订接受写成约束 7/8 全项解除。

## 4. Lead 预置问题处置

| 编号 | 判定 | 处置 |
|---|---|---|
| P1 `ADR-0032 r60` | **确认** | 记为 `F001`（MEDIUM）。ADR-0032 现为 r03（`framework.md:1068`），`r60` 是 `identity-auth-design-ir01-r60`（`design.md:5`/`:38`，其 `:3601`「r60 生产 root 装配合同与 11 端口清单」）；卡面把设计修订号挂在 ADR 号后。 |
| P2 Windows 退出码 2 | **确认** | 记为 `F002`（HIGH）。`main_test.go:650` 属 maintenance `extra_argv` 子测试（退出码 2 + `maintenance.invalid_request`），不是 recovery 平台门禁；`TestRecoveryRootStartup`（`main_test.go:11032`）只覆盖 recovery 非法参数退出码 2 与 verify 退出码 1；既有 `recovery_exclusive_test.go` 用 fake facility，覆盖 `ErrAdmissionUnsupported` 但不覆盖进程退出码。 |
| P3 22/14 与证据槽口径 | **确认但数值自洽** | 记为 `F003`（LOW）。22 槽去重确为 14，但四张实施卡的证据报告被排除在槽外，D05/D06 却把 `evidence-b13j-d05|d06/report.md` 计入「2 文件」。 |
| P4 `:46`/`:49` 差一 | **确认为历史残留** | **仅 observation（O01）**，不记 finding：登记前旧基线即为 76 vs 77、49 vs 50；本轮按旧口径同步为 82/55，三态行正确。建议由独立元数据更正记录处理，本轮无授权。 |
| P5 前像不可核 | **部分确认** | 记为 `F004`（LOW）。`tasks.md` 基线我独立复算成立（823187 B / `F1D65B0D…`）；`plan-before.md` 物理快照全仓不存在，Plan 基线字节不可复算；登记轮无作者 report/index。Plan 行数 1628→1651（+23）与「新增 25 行 / 替换 2 行」自洽，但不能替代前像。 |

## 5. Findings

### B13J-R05-F002（HIGH，OPEN）— S4.24b 的 Windows 退出码 2 负向断言在声明写集内不可执行
- 位置：`tasks.md:3184`（负向测试 (e)）、`:3168`（唯一声明新增测试文件的构建标签）、`:3189`（停止条件）、`:3185`（关闭条件 (4)(5)）。
- 证据：
  - `:3168` 声明 `cmd/han-server/recovery_exclusive_posix_integration_test.go` 带 `//go:build integration && (aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)` → 在 Windows 及非 POSIX 平台该文件**根本不参与编译**，(e) 无法在该测试内实现或失败。
  - 仓内现有唯一「退出码 2」进程级断言是 `cmd/han-server/main_test.go:650`，对象是 maintenance `password-reset retry extra`（`maintenance.invalid_request`），与 recovery 部署锁平台门禁无关；`TestRecoveryRootStartup`（`:11032`–`:11072`）只断言 recovery 非法参数退出码 2 / verify 退出码 1。
  - 既有 `cmd/han-server/recovery_exclusive_test.go` 无构建标签但全部用注入的 fake facility，验证 `ErrAdmissionUnsupported` 错误值，不验证进程退出码。
  - 本卡写集（`:3162`–`:3169`）不含任何 Windows 可编译测试文件；停止条件 `:3189` 明写「改 `internal/recovery` 之外未列文件…停止」。因此实施者要么触发 STOP，要么在关闭时给出无法成立的「(e) 通过」。
- 影响：关闭条件 (4)「上述负向测试全部通过」与 (5)「退出码 2 且在任何状态变更前拒绝」缺可失败 oracle；同时 ADR-0032 解除条件 3（`framework.md:1114`「Windows 平台在任何副作用前返回退出码 2」）在 B13J 内**没有承接节点**。
- 建议（二选一，均在卡片文本层）：
  1. **首选**：写集增列一个 Windows/非 POSIX 专用测试文件（如 `cmd/han-server/recovery_exclusive_unsupported_integration_test.go`，标签 `integration && !(aix||darwin||…)`），(e) 指向它，并把声明槽 `6→7`、唯一文件 `14→15`、B13J 槽位计数同步（Plan `:1311`/`:1641` 与所有权索引一并更新）；或
  2. 把该断言从本卡负向 oracle 移出，改列为 ADR-0032 解除条件 3 的证据，并在 ADR-0032 下次修订或另立节点中指定承接卡；此时本卡 (e) 必须删除而不是保留为不可执行项。
- 关闭条件：卡片文本与写集/槽位计数一致，且 (e) 指向的测试在对应平台可编译、可构造「退出码不为 2 即失败」的阴性用例。

### B13J-R05-F001（MEDIUM，OPEN）— S4.22 目标混写 ADR 修订号与设计修订号
- 位置：`tasks.md:3195`。
- 证据：卡面写「完成 ADR-0032 r60 的 11 端口装配」。实测 `framework.md:1068` 的 ADR-0032 修订为 `adr-0032-r03`；不存在 ADR-0032 r60。`r60` 属设计：`design.md:5` `revision: identity-auth-design-ir01-r60`，`:3601`「r60 生产 root 装配合同与 11 端口清单」。11 端口清单本身出自设计 r60，ADR-0032 只给部署独占/解除链边界。
- 影响：可追溯性错位——实施者若按「ADR-0032 r60」检索会落空；也使「11 端口清单是唯一装配清单」的权威归属（设计 §r60）被误记为 ADR。
- 建议：改为「按 design `identity-auth-design-ir01-r60` §r60 的 11 端口清单完成装配（边界受 ADR-0032 r03 约束）」，并在关闭条件 (2) 保持对设计清单的引用。
- 关闭条件：卡面不再出现不存在的 ADR 修订号，且设计/ADR 两个引用各自指向真实存在的制品与章节。

### B13J-R05-F003（LOW，OPEN）— 声明槽计数的口径在 D 卡与实施卡之间不一致
- 位置：`plans/ir-01-content-community-batches.md:1311`、`:1641`；`tasks.md:3044`/`:3070`（D05/D06 计 2 文件，含作者证据）与 `tasks.md:3101`/`:3134`/`:3169`/`:3203`（实施卡声明证据报告「不占写集」）。
- 证据：22 槽 = 2+2+4+4+6+4，去重恰 14（实测），数值自洽；但四张实施卡的 `evidence-b13j-s423|s424a|s424b|s422/report.md` 被排除，D05/D06 的 `evidence-b13j-d05|d06/report.md` 被计入。若统一计入，槽位应为 26、唯一文件 18。
- 影响：跨批比较与「声明槽 = 实际写入文件」的 Gate 核对会出现口径分歧；不影响实现安全。
- 建议：统一为「证据报告一律不计入代码写集，但在证据落点单独声明」，或「一律计入」；在 Plan `:1311` 明写该口径。

### B13J-R05-F004（LOW，OPEN）— 登记轮缺作者证据包与前后像不可复算
- 位置：`.spec/wip/recovery-runtime/evidence-b13j-r01/registration-round-r01/`（仅 `lead-r04-low-triage.md` 4052 B、`lead-verification-r05.md` 5242 B 两份 Lead 自证文件）。
- 证据：目录内无作者（Carson）`report.md`/`index.json`；作者声称的 `tasks-before.md` / `plan-before.md` 快照全仓不存在（按文件名与 `*before*` 检索均未命中）。`tasks.md` 基线可由现状重建并命中 `F1D65B0D…`/823187 B（本评审独立复现）；Plan 的 497542 B 基线**不可复算**，只剩「1628→1651（+23）＝新增 25 行/替换 2 行」的行数自洽，不能证明六处 hunk 之外零漂移。
- 影响：登记轮的「改了哪些字节」缺少作者侧可核对证据；不改变本次六卡与 Plan 内容的静态可核性。
- 建议：登记轮收口时补一份作者证据 `index.json`（改前/改后哈希+字节+行数、每处 hunk 的行区间），前像不再要求物理快照、但必须给出可复算的哈希与还原方法（如 `tasks.md` 本次做法）；Plan 侧建议保存 `plan-before.md` 或记录 git blob 标识。

### B13J-R05-F005（LOW，OPEN）— ADR-0032 `:1121` 的「尚未登记为正式 Task」括注已被本轮登记变为历史陈述
- 位置：`framework.md:1121`（ADR-0032 解除顺序第 2 节点）；对照 `tasks.md:3089`（S4.23 已登记）。
- 证据：`:1121` 写「候选 S4.23（anchor adapter bundle 四端口齐备并登记；尚未登记为正式 Task）完成」。本轮六卡登记后 S4.23 已是正式 Task（`tasks.md:3089`、Plan `:1311`/`:1336`），括注与现状不符；`:1124`/`:1125` 的「候选 S4.24a/S4.22」是类别名，不构成事实冲突。
- 影响：解除顺序文本的可追溯性；不改门禁实质（`:1121` 的实质要求是 bundle 交付且 `Exclusive` 离开 `recoveryClosedPort`），不阻断登记或实施。
- 建议：并入 ADR-0032 下次修订（可与 `B13J-PLAN-R04-F002` 的 `:1108` 限定语一并处理），删除该括注或改为「（规划编号，登记状态见 Tasks）」的时点无关表述。
- 关闭条件：ADR-0032 下次修订不再含与 Tasks 现状冲突的登记状态括注。

### Observation O01（非 finding）— `tasks.md` 统计行差一为登记前既有
- `:46` 总数 82 而实测 `- [` 83；`:49` identity-auth 55 而实测 56（56+26+1=83）。登记前旧值为 76 vs 77、49 vs 50，本轮按同一（错误的）口径同步为 82/55。`:47` 三态 67/2/14 与实测一致。
- 判定：非本轮引入、非本轮授权修复范围，本轮不改；建议另立元数据更正记录（不得夹带在 ADR 或卡片修订中）。
- 附：`tasks.md` 的 0x0C×2 + TAB×1 位于偏移 12631/12676/12691（`test/`→TAB`est`、`fixtures.sql`/`faults_test.go`→0x0C`ixtures...`），属 r01 `F009` 已登记的历史损伤，本轮未改、未授权修复，保持 OPEN。

### 5.1 承接 r04 未闭合 LOW（本轮未处置，保持 OPEN）
- `B13J-PLAN-R04-F001`（LOW，OPEN）：`framework.md:960(b)` 把「人的明确生产批准」引作约束 7 既有子项，但约束 7（`:954`–`:958`）无此项，人的批准实见 `:960(c)` 与 ADR-0032 `:1117`。六卡沿用「约束 7 全项加人的明确生产批准」属加严引用，不产生绕过；本轮未改 ADR，保持 OPEN。
- `B13J-PLAN-R04-F002`（LOW，OPEN）：`framework.md:1108` 末句「最后由正式 Task 规划登记装配与验证」仍位于不变量章，可被读成 S4.22/S4.24b 的登记前置。本轮 Plan `:1642` 已明确「登记不受约束 7 未满足项阻挡」，运行层风险被卡片文本抵消；ADR 措辞仍待下次修订。
- `B13J-PLAN-R04-F003`（LOW，OPEN）：`adr-r04/report.md:19` 与 `adr-r04/index.json` 把 r03 报告记为 259 行（实测 258），属作者证据元数据；本轮未改作者证据，保持 OPEN。

## 6. 实跑命令与真实结果

| # | 命令（要点） | 真实结果 |
|---|---|---|
| 1 | `[IO.File]::ReadAllBytes` + SHA256 三文件 | tasks.md 855008/`A8205A21…12852`；Plan 504272/`D9A79C54…C936B`；framework 182488/`39BD3A79…BD704` |
| 2 | CRLF/bareLF/BOM/tail 扫描 | tasks CRLF 3222、bareLF 0、无 BOM、tail `0d 0a`；Plan CRLF 1651、bareLF 0、无 BOM、tail `0d 0a`；framework CRLF 1141、bareLF 0、tail `0d 0a` |
| 3 | 前缀重建：前 3034 行 + 还原 4 统计行 → SHA256 | **823187 B / `F1D65B0D7016CDBF9C2F4E85B012B5B0B9F714B6DF0C7CDC9326C05676900E34`**（与登记前记录一致） |
| 4 | `- [` 计数与三态 | BOXCOUNT=83；TODO=14 / ACTIVE=2 / DONE=67 |
| 5 | ID 归属计数 | identity-auth 56 / article-management 26 / bootstrap 1 |
| 6 | 0x0C/TAB 定位 | 0x0C=2（@12676,12691）、TAB=1（@12631），均在文件前部历史区 |
| 7 | 六卡 12 字段存在性 | 六卡 `missing=[]` |
| 8 | 硬 requires 行抽取（3037–3222） | D05/D06 无；S4.23←D05；S4.24a←D06；S4.24b←S4.23；S4.22←S4.24b |
| 9 | mermaid 节点/边解析（1344–1465） | NODE_DECL=41、EDGE_LINES=79、UNIQ_NODE_IDS=41、UNIQ_EDGES=79 |
| 10 | 所有权索引槽位抽取（Plan 1334–1339） | SLOT_TOTAL=22、UNIQUE=14（清单见 §3E） |
| 11 | `rg -n 'func TestRecovery…'` | 实存 6 个既有测试名 + 4 个未创建的新增测试名（落在声明新增文件） |
| 12 | `rg --pcre2 'S4\.24(?![ab])'` tasks.md + Plan | 无命中（exit 1） |
| 13 | `rg 'ErrUnsupportedPlatform|退出码 2'` cmd/internal | 只有 `internal/recovery/*` 的错误值与 `internal/article`/`journal_windows_test`；`cmd/han-server` 仅 `main_test.go:650`（maintenance） |
| 14 | 读 `main_test.go:640–670`、`11032–11072` | 均为 maintenance 额外 argv / recovery 非法参数退出码 2，**不含**平台不支持门禁 |
| 15 | 读 `recovery_exclusive_test.go` | 无构建标签，全部 fake facility，断言错误值不断言进程退出码 |
| 16 | 读 `recovery_runtime.go:545–570` | `:551` `newRecoveryAnchorRegistry(nil)`；`:565-567` `inputs.exclusive` 无条件覆盖（与 S4.22 引用一致） |
| 17 | `rg 'adr-003[012]-r0[0-9]'` framework | ADR-0030 r04（`:918`）、ADR-0031 r04（`:978`）、ADR-0032 r03（`:1068`）→ 无 r60 |
| 18 | `rg '11 端口|ir01-r60'` design | `design.md:5`/`:38` r60、`:3601` 11 端口清单 |
| 19 | `git status --porcelain` / `git rev-parse` | 293 行未变；HEAD `6da8abb…`、分支 `main`；三目标文件 `M` |
| 20 | `git diff --numstat`（仅参考） | HEAD 版 tasks.md 仅 69445 B、Plan 为早期少 830 行版本 → **git 不能充当登记前基线**，故用 #3 重建 |
| 21 | `rg -n '尚未登记为正式 Task' framework.md` | 4 命中：`:960`/`:1050` 为类别表述、`:1059` 为历史追溯、`:1121` 为与现状冲突的登记状态括注（`F005`） |

全部命令为只读文本/哈希检索；未运行 Go 测试、PG、服务，未做任何 Git 写操作。

## 7. 未覆盖范围

- 未独立重算全图 333 条 requires 边（只核 +4 边与 79 边切片、83 节点）；未做全图无环/引用有效性复算。
- 未验证 11 端口清单与 `cmd/han-server/main.go` 现有装配点的逐项可实现性（设计合同层，属实施前设计核对）。
- 未验证 S4.23/S4.24a/S4.24b/S4.22 各卡新增测试文件在真实 Linux/Windows 上的可编译性（文件尚不存在）。
- 未核 ADR-0019/0027/0029 与 ADR-0030/0031/0032 的全文冲突（沿用 r01–r04 已 CLOSED 结论，本轮只核与本登记直接相关的 `:953`–`:961`、`:1050`、`:1110`–`:1127`）。
- 未核 `article-management-S8.2`、Case18 的当前卡面（只核六卡中是否有对其 PASS 的宣称）。
- 未核 `.spec` 全树新增产物的完整性（仅登记轮目录与本评审目录）。

## 8. 格式实测（被审对象）

| 文件 | 行数 | CRLF | bareLF | BOM | 末尾 | 控制字符 |
|---|---:|---:|---:|---|---|---|
| tasks.md | 3222 | 3222 | 0 | 无 | 单 CRLF | 0x0C×2 + 0x09×1（历史区） |
| ir-01-content-community-batches.md | 1651 | 1651 | 0 | 无 | 单 CRLF | 0 |
| framework.md | 1141 | 1141 | 0 | 无 | 单 CRLF | 0 |

本报告与 `index.json` 自身的格式见 `index.json` 的 `files` 字段（`self_hash_note` 说明为何不自嵌自身哈希）。

## 9. 边界

- 本评审只读被评审制品，未修改 `tasks.md`、Plan、`framework.md`、`decisions/README.md`、源码、测试、设计或任何既有评审证据；唯一写入为本目录 `report.md` 与 `index.json`。
- 本结论不构成实施授权、不登记 S4.22/S4.23/S4.24a/S4.24b（六卡已由登记轮登记，本评审不新增/不推进状态）、不解除 `blocked-upstream`、不关闭 `pending-B13J-D04`、不宣称 Case18/S8.2 或生产可用，不是人的业务签收或生产批准。
- 本评审为 AI 代理核证；`actualID` 本会话不可得，不编造。
