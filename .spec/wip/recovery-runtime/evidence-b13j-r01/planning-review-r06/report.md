# B13J 规划评审 r06 — 登记轮 r02 定向复审

- 角色：add-reviewer（非作者）。未写 tasks.md、Plan、D01–D04 设计、ADR 修订；本目录外零写入。
- 对象：B13J 登记轮 r02 的两处写入（`tasks.md` 六卡区、`ir-01-content-community-batches.md` 三行）及其作者证据。
- 范围：只判 R05 的 F001–F005 是否 CLOSED，以及 r02 修订是否引入新问题。不重评 r01–r04 已 CLOSED 项。
- **verdict：`PASS WITH MINOR`**
- 边界：本评审不构成实施授权；不登记、不推进 S4.22/S4.23/S4.24a/S4.24b；不关闭 `pending-B13J-D04`；不解除 `blocked-upstream`；不是人的签收。

## 1. 结论摘要

R05 的 F002（HIGH）、F001（MEDIUM）、F003（LOW）、F004（LOW）四条均可 CLOSED。F005（LOW）仍 OPEN，属已明确转交 ADR-0032 下次修订的延后项，不阻塞登记轮，也未因本轮修订变差。新增 2 条 finding：1 条 MEDIUM（`B13J-R06-F002`，非 POSIX 负向命令未固定平台，可在 POSIX 上空跑通过）、1 条 LOW（`B13J-R06-F001`，gofmt 清单漏列第 7 文件）。两条均不影响「六卡 + Plan 文本作为登记基线」，但各有明确关闭条件。

因此：**可以进入「决策卡 + Task 卡（S4.22/S4.23/S4.24a/S4.24b）登记轮」**。登记不等于实施授权，也不解除任何生产 closed 状态。

## 2. 输入实测

| 文件 | 字节 | SHA256 | 格式实测 |
|---|---:|---|---|
| `.add/project/knowledge/tasks/tasks.md` | 856471 | `0BD07ADC2ECC8CD157DC0CD9F295BA6B60D7C2FB00148CF588C55BBEBED1FFE0` | CRLF 3225、bareLF 0、无 BOM、末尾单 CRLF；历史 TAB@12631、0x0C@12676、0x0C@12691（未改） |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | 504671 | `EC24892E46D5CB1C668F8F7E13443BC60F64F6A53A82812DBAEA30C79D9D37B9` | CRLF 1651、bareLF 0、无 BOM、末尾单 CRLF、控制字符 0、TAB 0 |
| `.add/project/knowledge/decisions/framework.md` | 182488 | `39BD3A797B0FA2F5CDF944ACCF4129BB0B7F7A23FABC743B7E5CCC9F7B9BD704` | 本轮未改（CRLF 1141、控制字符 0） |
| `registration-round-r01/author-report-r02.md` | 8466 | `42C9EF5AD8F6A7779D1C5755616A565A0B557D76872341BFC458303CC5D4516B` | 纯 LF、无 BOM、末尾单 LF |
| `registration-round-r01/author-index-r02.json` | 11092 | `8D3A39731E9DA33460B6A7F4BE48DD78E5096D7F83C1030EA4EA33B9F3793484` | 纯 LF 198、无 BOM、`ConvertFrom-Json` 合法、schema `add.evidence.registration-round/v1` |
| `planning-review-r04/report.md` | 21187 | `38DE303B432921C9A369C08F7BF4DD1999C08EBA72CACF8A99BFDEBE9174E352` | 未改（r04 次序依据） |
| `planning-review-r04/index.json` | 26320 | `95C2D39DDBC12FEA441B1AB083133A1636582CBF50C80CEDA7DB79F36D639D4E` | 未改 |
| `planning-review-r05/report.md` | 23571 | `FC5FACF409379149C1F0422515B32B6C1E1A2E1C90905688616496B58BB82558` | 未改（原 finding 基线） |
| `planning-review-r05/index.json` | 27756 | `544E6A99514E6D3D0DC6BF59820B7AB45AE75BD4E739E61C2B7CE8049E1EAC23` | 未改 |

工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（`git rev-parse HEAD` 与 `git branch --show-current` 实测一致，未变）。`git status --porcelain` 293 行。

## 3. R05 finding 逐条判定

| R05 编号 | r05 级别 | r06 判定 | 一句话依据 |
|---|---|---|---|
| `B13J-R05-F002` | HIGH | **CLOSED** | 新增第 7 文件与严格互补构建标签齐备；写集 7/槽 23/唯一 15 三处自洽；12 条命令与 12 行一致；(e) 已绑定非 POSIX 测试且要求 `ErrUnsupportedPlatform` 分类。残留验证强度问题另立 `R06-F002`。 |
| `B13J-R05-F001` | MEDIUM | **CLOSED** | `tasks.md:3198` 已改为设计修订 `identity-auth-design-ir01-r60` 的 11 端口清单，并以 ADR-0032 r03 约束部署独占与解除边界；全文无「ADR-0032 r60」。 |
| `B13J-R05-F003` | LOW | **CLOSED** | Plan `:1311`、`:1641` 明写槽位口径（D05/D06 的 2 文件含各自作者证据；实施卡 `.spec` 证据报告不计槽）；15/23 与卡面 2/2/4/4/7/4 及所有权 `:1334–:1339` 一致。 |
| `B13J-R05-F004` | LOW | **CLOSED** | `author-report-r02.md` + `author-index-r02.json` 存在且自洽，并如实写明 Plan 旧基线不可复算；改前/改后哈希、行距、单元位移与我的独立复核一致。 |
| `B13J-R05-F005` | LOW | **OPEN（延后）** | `framework.md` 本轮零改动（`39BD3A79…9704`），ADR-0032 `:1121`/`:1123–:1125` 仍写「尚未登记为正式 Task」「候选」；须随 ADR-0032 下次修订收口。未变差。 |

## 4. 关闭依据

### 4.1 `B13J-R05-F002`（S4.24b 负向 oracle 平台不可达）

- 写集第 7 文件已落地于 `tasks.md:3169`：`cmd/han-server/recovery_exclusive_unsupported_integration_test.go`，标签 `//go:build integration && !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)`。
- 标签补集严格：与 `tasks.md:3168` 的 POSIX 标签 `integration && (aix || darwin || … || solaris)` 逐项取反，且与仓内既有 `internal/recovery/filelock_unsupported.go:1` 的 `//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)` **逐字节相同**（`rg -n "go:build" internal/recovery/filelock_unsupported.go` 实测）。
- Windows 可编译：该文件仅在非 POSIX 平台参与构建，不引用 `unix` 包；断言对象是 `cmd/han-server` 的退出码路径（`main.go` 在写集内），不依赖 POSIX 系统调用。
- 断言可失败且非「只断错误非 nil」：`tasks.md:3187` 要求退出码 2 且以 `ErrUnsupportedPlatform` 为分类依据；`tasks.md:3188` 关闭条件 (4) 把 (a)–(d) 归 POSIX 测试、(5) 把 (e) 归非 POSIX 测试并重申不得只断非 nil；`tasks.md:3192` 停止条件禁止移除 `!(…)` 取反。既有 `cmd/han-server/main_test.go:650` 有子进程退出码断言先例，路径可实现。
- 计数自洽：`tasks.md:3172` 标「12 条」，实际命令行 `:3174–:3185` 恰 12 行；`gofmt` 之外 11 条与写集匹配。
- 三处数字自洽（我独立解析六卡写集得出）：槽位 2+2+4+4+7+4 = **23**，去重 = **15**；Plan `:1311`、`:1641`、所有权行 `:1338` 均写 15/23，S4.24b 行写 7 个文件并列出新文件名与顶层测试名。
- 残留（不否定本条 CLOSED，另立 `R06-F002`）：`:3182`/`:3183` 两条命令未固定平台；在 POSIX 主机上该文件被构建标签排除，`go test -run '^TestRecoveryDeploymentExclusiveUnsupportedPlatform$'` 会「无测试可跑」而仍 exit 0。

### 4.2 `B13J-R05-F001`（S4.22 目标把 ADR 与设计版本混写）

- `tasks.md:3198` 现文：「按 design `identity-auth-design-ir01-r60` 的 11 端口清单完成装配（部署独占与解除边界受 ADR-0032 r03 约束）」。
- 锚点真实：`design.md:5` 为 `revision: identity-auth-design-ir01-r60`；`design.md:3601` 为 11 端口清单小节标题。`framework.md:1068` 修订行为 `adr-0032-r03`。
- 残留扫描：对 `tasks.md` 与 Plan 执行 `rg -n "ADR-0032 r60|adr-0032-r60"` → 退出码 1（零命中）。

### 4.3 `B13J-R05-F003`（15/23 槽位口径）

- Plan `:1311` 与 `:1641` 两处都显式写明口径：D05/D06 的 2 文件含各自作者证据（`.spec/wip/recovery-runtime/evidence-b13j-d05/report.md`、`…-d06/report.md`），四张实施卡的 `.spec` 证据报告不计入声明槽、另在证据落点声明。口径唯一，无第二种读法。
- 数字与卡面枚举一致：我按六卡「精确文件清单/写集」逐条解析得 23 槽 / 15 唯一（含 `framework.md` 与两份 D05/D06 证据），与 Plan 及所有权行一致。
- 实施卡证据报告确实不在写集内（各卡以「证据报告 … 为非代码产物，不占 N 文件写集」单列），与口径自洽。

### 4.4 `B13J-R05-F004`（作者证据）

- 两份新证据齐全、哈希与 Lead 一致；`author-index-r02.json` 的 `changed_files` 给出改前/改后字节、行数与哈希，`format_after` 给出 CRLF/bareLF/BOM/末尾/TAB/0x0C 明细，`prior_image` 明确 `plan.recomputable=false`、`physical_snapshot_in_repo=false`，并在 `note` 中说明仅存于仓库外的 `%TEMP%` 副本（497542 B / `3FA19CDD…637380`）。
- 我独立复核的位移证据：
  - `tasks.md` 1–3154 行 842938 B，SHA256 `31B0BB3EDFD584FECD683B84758748D0BF6588CB2BFF728353FF0E8580C05152`（该卡之前零改动）。
  - `tasks.md` 3193–3225 行 6224 B，SHA256 `A2F5EB15BFF6D2C19A29410DDCDF49D5AB9D680D187A4EB7CA26F3408E4EB351`；S4.24b 卡体 3155–3192 行 7309 B，SHA256 `BA1AD85BA864C7D3B64B3E4F2810E14C83AA44A69C97F4A658693A8CE81A78CE`。
  - Plan 四段指纹：1–1310 `5EEE2810D75CC7C714992272B7A3F6EF6B405BC0D4B86D92BC6AE82AE19E98D0`；1312–1337 `7EEAFF3AC79E4BD55811E3271C4DA9FE887A109DC4F4DBFDD5B940DDE5AAFAF3`；1339–1640 `D2D767025EA1345AC4F8C0D2DB18404774AB3F8F16109BB54C5D39DCED9D96A0`；1642–1651 `1B6F191178733DD164B0746C7396C24CC0D8996141108B4E600DD1D878010241`；均与 Lead 实测一致，变化仅 `:1311`、`:1338`、`:1641` 三行。
  - 增量算术：`tasks.md` +1463 B / +3 行（写集 1 行 + 命令 2 行），Plan +399 B / 0 行（三处单行替换），与作者声明一致。
- 结论：证据足以支撑「六处 hunk 之外零漂移」的**行级**判断。不可核项保留为 `O03`（Plan r01 旧像仓库内不可复算），作者已如实披露，不构成隐瞒。

### 4.5 `B13J-R05-F005`（ADR-0032 候选措辞）

- `framework.md` 哈希与 r05 相同，本轮无 ADR 改动，故该 finding 必须保持 OPEN。
- 现文仍为：`:1121`「候选 S4.23（…；尚未登记为正式 Task）」、`:1123–:1125` 三处「候选」。
- 处置成立：转交 ADR-0032 下次修订，在登记完成后把这些「候选/尚未登记」改为指向已登记 Task。级别 LOW，不阻塞登记轮。**未因本轮修订变差**（无新增同类措辞）。

## 5. 新 finding

### `B13J-R06-F001`（LOW）S4.24b 的 gofmt 清单漏列第 7 个文件

- 位置：`tasks.md:3174`。
- 证据：该行 `gofmt -l` 只列 6 个路径，缺 `cmd/han-server/recovery_exclusive_unsupported_integration_test.go`；而 `:3162` 写集标题与 `:3169` 已把它列为第 7 文件，Plan `:1338` 也按 7 文件登记。
- 影响：格式门禁不覆盖新增测试文件，`gofmt -l` 仍会 exit 0，属验证覆盖缺口，不影响任务可实施性。
- 建议：把该路径补入 `:3174`（命令仍为 12 条，仅改一行内容）。
- 关闭条件：`tasks.md:3174` 含 7 个路径且与 `:3162–:3169` 写集一一对应。

### `B13J-R06-F002`（MEDIUM）非 POSIX 负向命令未固定平台，可在 POSIX 上空跑通过

- 位置：`tasks.md:3182`、`:3183`（并影响 `:3188` 关闭条件 (5) 的取证强度）。
- 证据：两条命令为 `go test [-race] -tags=integration ./cmd/han-server -run '^TestRecoveryDeploymentExclusiveUnsupportedPlatform$'`，未限定 `GOOS`。该测试文件带 `!(aix||darwin||…||solaris)` 标签，在 Linux/macOS 上被构建排除；Go 在过滤后无测试匹配时仍以 exit 0 结束，命令表面「通过」但未执行任何断言。卡面 `:3187` 要求 (e) 必须由该测试承接，`(5)` 也以它为唯一取证手段。
- 影响：在 POSIX CI 上，(5) 可被空跑满足，形成假通过；不会造成计划层错误，但会让 S4.24b 的负向门禁失效。**不阻塞登记轮**（卡可登记），但必须在 S4.24b 关闭前消除。
- 建议：命令显式固定非 POSIX 平台（例如 `GOOS=windows go test -tags=integration …`，或在 Windows runner 上执行并记录 `GOOS`）；并在 `:3188` (5) 增补「须证明测试实际执行（`-json` 中出现该测试的 run/pass 事件，而非 no tests to run）」。
- 关闭条件：`:3182`/`:3183` 绑定平台，且 `:3188` (5) 含「测试实际执行」的反空跑要求。

## 6. Observations（不单独编 finding）

- `O01` 统计行历史差一：`:46` 总数 82、`:47` 67+2+14=83、`:49` identity-auth 55，而实际 `- [` 行 83、identity-auth 归属 56。登记前即存在，本轮未触碰、未授权修复。
- `O02` 非 POSIX 补集标签同时覆盖 js/wasip1/plan9/android/illumos/ios 等平台；在 Windows 之外的这些平台运行该测试时退出码路径可能不同，建议实施时以 Windows 为准（与 `R06-F002` 的建议一致）。
- `O03` Plan r01 旧像（497542 B / `3FA19CDD…637380`）仓库内无物理副本，仅存于仓库外 `%TEMP%`；作者已如实标注不可复算，评审侧同样无法复算，仅能核行数算术与行级位移。
- `O04` `tasks.md` 历史 TAB@12631 与 0x0C@12676/12691 未改（r01 `F009` 保持 OPEN，无修复授权）。
- `O05` `author-index-r02.json` 的 `boundary.changed_existing_cards: false` 措辞不精确：本轮确实改了既有卡文本（S4.24b 卡体、S4.22 目标行），但 `changes[]` 与 report §2 已逐条披露，不构成隐瞒；建议下轮改为「未改既有卡状态字段」。
- `O06` `framework.md:1121`/`:1123–:1125` 的「候选/尚未登记」在 r01 登记后已成历史措辞，与 `R05-F005` 同一收口点。
- `O07` 我按 Lead 给出的重建方法复算 `tasks.md` 旧像未收敛（得 823204 B，非 823187 B），原因是对 `:51` 原文字（`开发 54` 等）无权威来源；因此旧像哈希 `F1D65B0D…` 我未能独立复现，仅记录为限制，不作为对作者的质疑。

## 7. 新不一致扫描（r02 修订是否引入新问题）

- 命令条数：`:3172` 标 12，实际 `:3174–:3185` 12 行，一致。
- 槽位数字：23/15 在 `:1311`、`:1338`、`:1641` 与卡面枚举四处一致（我独立解析确认）。
- 卡面与 Plan：六卡名称、编号、依赖方向与 Plan `:1311`、`:1334–:1339`、`:1456–:1465` 一致；硬依赖仍为 4 条边（D05→S4.23、D06→S4.24a、S4.23→S4.24b、S4.24b→S4.22）。
- mermaid 与图计数：Plan `:1342` 仍为「24 卡/79 边」「77 节点/329 边→83 节点/333 边」，mermaid 实测 41 节点 / 79 条边（runtime 切片视图，与 r01 相同，未被本轮误改）；`:1647` 交回句未变。
- 停止条件与写集：`:3192` 的「不得放宽标签」只约束新增测试文件的构建标签，不要求改其他文件，与 7 文件写集无冲突。
- 宣称词扫描：六卡内「已通过评审/已实施/已解锁/生产可用」仅以「不宣称…」否定式出现；未出现正向宣称。
- 引用完整性：对 `tasks.md` 与 Plan 执行 `rg --pcre2 "S4\.24(?!a|b)"` → 退出码 1（无裸 `S4.24`）；卡内引用均为已登记编号（D05/D06/S4.22/S4.23/S4.24a/S4.24b）。
- 结论：未发现新的阻断性不一致；两条新 finding 均为可局部修补的验证覆盖问题。

## 8. 实跑命令与真实结果

| # | 命令 | 结果 |
|---|---|---|
| 1 | `git rev-parse HEAD; git branch --show-current` | `6da8abb…38ddca`；`main` |
| 2 | `Get-FileHash` + `[IO.File]::ReadAllBytes` 于 tasks/Plan/framework | 856471/`0BD07ADC…1FFE0`；504671/`EC24892E…D37B9`；182488/`39BD3A79…9704` |
| 3 | 列 `registration-round-r01` 目录 + 两份新证据哈希 | 报告 8466/`42C9EF5A…4516B`；索引 11092/`8D3A3973…3484` |
| 4 | `ConvertFrom-Json` 于 `author-index-r02.json` | 合法；schema `add.evidence.registration-round/v1` |
| 5 | 打印 `tasks.md` 3156–3192、3193–3225 行 | S4.24b 卡 7 文件、12 命令、(4)/(5) 分离、停止条件含取反标签 |
| 6 | 打印 Plan `:1311`、`:1338`、`:1641` | 15/23 口径 + 7 文件所有权行 + 槽位说明 |
| 7 | 打印 `framework.md` `:960`、`:961`、`:1050`、`:1068`、`:1119–:1126` | 约束 9 三级次、约束 10、ADR-0031 门禁、ADR-0032 修订头与解除顺序链 |
| 8 | `rg -n "ADR-0032 r60\|adr-0032-r60"`（tasks+Plan） | 退出码 1，零命中 |
| 9 | `rg -n --pcre2 "S4\.24(?!a\|b)"`（tasks+Plan） | 退出码 1，零命中 |
| 10 | `rg -n "go:build" internal/recovery/filelock_unsupported.go` | `:1` 补集标签与 `tasks.md:3169` 逐字节相同 |
| 11 | `Test-Path` 两个新测试文件 | 均 False（尚未创建，符合规划态） |
| 12 | 卡计数：`^- \[` / `- [ ]` / `- [x]` / `- [-]` | 83 / 14 / 67 / 2；identity-auth 卡 56（统计行写 55，见 O01） |
| 13 | 逐卡解析「精确文件清单/写集」 | 2/2/4/4/7/4，合计 23 槽，去重 15 |
| 14 | Plan mermaid 节点/边计数 | 41 节点 / 79 边；`:1342` 与 `:1647` 未变 |
| 15 | 字节分段指纹（tasks 1–3154、3155–3192、3193–3225） | `31B0BB3E…C05152`、`BA1AD85B…1A78CE`、`A2F5EB15…4EB351` |
| 16 | 字节分段指纹（Plan 四段） | `5EEE2810…E98D0`、`7EEAFF3A…AAFAF3`、`D2D76702…D96A0`、`1B6F1911…10241` |
| 17 | 增量算术 | tasks +1463 B / +3 行；Plan +399 B / 0 行 |
| 18 | r04/r05 证据哈希复核 | 与 r05 记录一致（见 §2） |
| 19 | `Get-ChildItem -Recurse -Include 'tasks-before.md','plan-before.md'` | 零命中（见 O03） |
| 20 | `git status --porcelain` | 293 行（无新增对本轮被评对象的写入） |

未运行：Go 编译/测试、PG、任何服务、任何 Git 写操作。

## 9. 未覆盖范围

- 未编译或执行新增测试（含 `GOOS=windows` 交叉编译验证），故「Windows 可编译」是静态判断（标签互补 + 无 `unix` 依赖 + 退出码路径在写集内），不是构建证据。
- 未重算全仓 requires 图（333 边），只核了 6 卡/4 边与 `:1342` 文本、mermaid 计数的一致性。
- 未重评 r01–r04 已 CLOSED 的 ADR 语义与设计结论；`framework.md` 本轮零改动，相关结论按其时状态沿用。
- 未核 `article-management-S8.2`、Case18 现状卡面（本轮两处写入未触及）。
- 未核 Plan 的 `:1311` 之前/`:1641` 之后各批次正文与 DAG 全量一致；仅核本轮 hunk 邻域。
- 未复现 tasks 旧像哈希 `F1D65B0D…`（见 O07）。

## 10. 本报告自身格式实测

- 字节：18374；LF：162；CRLF：0；BOM：false；末尾：单 LF；控制字符：0；TAB：0。

## 11. 边界复述

本评审只回答「六卡 + Plan 文本是否可作为登记轮基线」。`PASS WITH MINOR` 不构成实施授权，不登记或推进 S4.22/S4.23/S4.24a/S4.24b 的实施，不关闭 `pending-B13J-D04`，不解除 `blocked-upstream`，不代表人的业务签收。`R05-F005` 需随 ADR-0032 下次修订收口；`R06-F001`、`R06-F002` 应在 S4.24b 实施前/关闭前修补。
