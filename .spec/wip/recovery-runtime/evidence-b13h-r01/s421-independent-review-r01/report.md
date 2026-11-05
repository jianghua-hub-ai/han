# S4.21 独立评审报告（s421-independent-review-r01）

- 结论：**PASS WITH MINOR**（1 个 LOW 非阻断 finding；无代码/测试缺陷）
- 评审角色：非作者独立评审（Reviewer / add-reviewer），不是 S4.21 实施者
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 日期：2026-11-01
- 声明：**AI 代理评审，非真人签字。** 本报告不代替 Lead/Main 的接受、任务关闭、部署授权或 Git 写授权；不宣称 Case18 或 S8.2 PASS，不解除 `blocked-upstream`。

## 1. 评审对象与修订

| # | 对象 | 实测 SHA256 | 字节 | 判定 |
|---|---|---|---|---|
| 1 | `cmd/han-server/recovery_runtime.go` | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` | 37431 | 与锁定一致 |
| 2 | `cmd/han-server/recovery_enter_install_integration_test.go` | `6BFA7D761432C960740F4E1FE91566317B5707B8481CE0D9FF1CF46DD4106C3A` | 20564 | 与锁定一致 |
| 3 | `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` | `2A0013FD4A92325DAD3BA35CF3444DBFA10B5A0884F17E335E26F591B8768019` | 3155 | 与锁定一致 |
| 4 | `cmd/han-server/main.go`（对照组） | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | 与基线一致，零改动 |

测试全部完成后复算四对象哈希仍一致，未发现评审期间对象漂移。所有测试命令均在上述 HEAD/哈希修订上执行。

## 2. 两轮实际检查范围

**第一轮：对象与静态结构（只读）**

- 完整读取 `add-code-review` 合同、`task-execution-workflow`、`coding-standards`、`no-hardcode-policy`、`code-review-checklist`、`abstraction-principles` 与产品安全红线。
- 逐段读取 `recovery_runtime.go`：`recoveryRootInputs`（`:432-451`）、11 端口缺失清单（`:456-492`）、错误拼接（`:494-496`）、`newRecoveryRootAssembly` 的 precheck 与资源构造顺序（`:522-737`）、secret candidate seam（`:631-637`）。
- 逐段读取新测试：生产 closed 形态（`:273-295`）、install/enter seam（`:297-335`）、单 runtime/store（`:337-361`）、默认 secret closed（`:363-373`）、配置/命令矩阵正负控（`:381-521`）、临时目录零写入断言（`:513-521`）。
- 读取 `main.go:618-680` 生产 root 接线，确认未注入 11 端口；读取 `internal/recovery/runtime.go` 的 `Install`（`:248-272`）、`VerifyInstallationSource`（`:411`）、`Enter`（`:1517-1535`）、`StopAndJoin`（`:1556`）、`Exclusive.Acquire`（`:1562`）。
- 读取任务卡 `tasks.md:2961-3029`、identity 设计 `:1458-1529`、identity test-plan `:831-842`、article test-plan `:353-362`、handoff 全文、实施侧 `report.md/red.log` 与 Lead 复跑 `report.md/commands.log`。

**第二轮：独立执行与证据核对**

- 独立执行 12 条卡面验收命令（见 `commands.log`），另做 4 对象哈希复核、`-race` 插桩探针、源码越界扫描、单实例 grep 计数、Lead JSON 测试名分组核对。
- 核对每条 test2json 的 `Action:fail`、`DATA RACE`，并统计 pass/fail/skip。
- 核对行号：卡面 `:482-483`、`:556-557`、`:582-583` 均明确标注“修订前代码事实”，属陈旧引用而非错误断言；修订后对应位置为 `:527-534`、`:572-591`、`:631-636`。

**有界追查（真实消费者/上游依据）**

- `main.go` 生产组装 `:634-650` 的 `recoveryRootInputs` 仅含 `pool/plan/prepared/closures/seal`，无任何 11 端口，确认 production `enter/install` 在 precheck 处即 closed。
- 新测试“注入全端口”确实进入真实 runtime 的 sentinel 分支：Install 先到 `InstallationSources.VerifyInstallationSource`（`:411`），Enter 先经 `Lifecycle.StopAndJoin` 再到 `Exclusive.Acquire`（`:1556`、`:1562`），不是用 fixture 绕过真实分支。
- Lead 证据 `c11/c12` JSON 同时含 `TestRecoveryConfig`、`TestRecoveryCommand` 与矩阵测试事件，证明其实际命令为卡面三重正则；但 Lead `commands.log` 行文只写矩阵单测，见 F001。
## 3. 关闭条件逐条判定（`tasks.md:3021`）

判定口径：每条以本仓库同一 HEAD、同一对象哈希上的只读执行或静态核对为依据。`PASS` 只表示该条关闭条件在本修订可核成立，不代替 Lead/Main 的业务接受、任务关闭或部署授权。

| # | 关闭条件 | 判定 | 证据 |
|---|---|---|---|
| 1 | `go build ./...` 通过 | PASS | `commands.log` 命令 2，EXIT=0，0 行输出 |
| 2 | `TestRecoveryEnterInstallAssemblyWiring` 普通与 `-race`（`-tags=integration`）通过 | PASS | `c09`/`c10` 各 9 个 `Action:pass` = 8 个测试事件（父测试 + 7 个子测试）+ 1 个包级聚合事件；fail=0、skip=0、无 `DATA RACE` |
| 3 | S4.16 `TestRecoveryRuntimeComposition` 同修订普通与 `-race` 回归通过 | PASS | `c05`/`c06` 各 10 = 9 个测试事件 + 1 包级；fail=0、无 `DATA RACE`；与目标对象同一 HEAD/哈希 |
| 4 | S4.19 `TestRecoveryMainBusinessDispatcher` 同修订普通与 `-race` 回归通过 | PASS | `c07`/`c08` 各 12 = 11 个测试事件 + 1 包级；fail=0、无 `DATA RACE` |
| 5 | `TestRecoveryEnterInstallConfigCommandMatrix` 普通与 `-race` 覆盖正控/负控并闭合 | PASS | `c11`/`c12` 各 27 = 26 个测试事件 + 1 包级：配置正控 5 形态、配置负控 9 例、命令正控 5 形态、命令负控 11 例；负控断言 `err != nil` 且解析前后临时目录零写入（`recovery_enter_install_integration_test.go:443`、`:454`） |
| 6 | `go vet`（普通与 `-tags=integration`）与 changed-file `gofmt -l` 通过 | PASS | 命令 1/3/4，均 EXIT=0；`gofmt -l` 输出 0 行 |
| 7 | `recovery_runtime.go` 原 `:482-483` 前置整路径 closed 已被显式 precheck 替换 | PASS | `recovery_runtime.go:527-534`：`enter`/`install` 分支调用 `recoveryEnterInstallMissingPorts`，非空即返回 closed 错误；`normal`/`verify`/`resume` 与未知 action 行为未变 |
| 8 | precheck 位于**任何资源构造之前** | PASS | precheck `:530-532`；首个资源构造是 `:537` 的 `identity.NewRecoveryRunStore`（紧随其后 `closures`、anchor registry、journal、runtime）；`:523-535` 仅做 ctx/pool/prepared/plan 校验与 `inputs.plan.config` 值拷贝，无分配、无 Store/pool/router 构造 |
| 9 | 缺任一必填端口返回 `recovery.ErrAdmissionClosed` 且错误文本列明**全部**缺失端口名 | PASS | `:494-496` 用 `%w` 包装 `recovery.ErrAdmissionClosed` 并以 `strings.Join(missing, ", ")` 拼接整份清单（固定顺序、无提前返回）；测试 `:271` 构造完整期望文本并 `:285` 做精确相等断言，`:288-292` 再逐项 `strings.Contains` |
| 10 | 11 个必填端口清单与卡面 `:3012` 完全一致 | PASS | `:456-491` 顺序为 `journal, articleSnapshots, restore, consumer, installation, articleAccount, commitments, sources, anchor, exclusive, secretCandidate`，与 `tasks.md:3012` 逐项一致；测试 `:38-50` 镜像同一顺序，顺序漂移会使精确文本断言失败 |
| 11 | `secretCandidate` seam：非 nil 时使用、nil 时保持 closed | PASS | `:631-637`：nil 时保留原 closed 闭包（“the recovery secret candidate source is not delivered”），非 nil 时替换为注入函数；测试 `:316-319` 断言注入候选被读取，`:363-373` 断言未注入（verify 形态）仍 `ErrAdmissionClosed` |
| 12 | production `main.go` 未注入，故 production `Enter`/`Install` 仍 closed | PASS | `main.go:634-650` 的 `recoveryRootInputs` 仅含 `pool/plan/prepared/closures/seal`，11 端口与 `secretCandidate` 全为 nil；测试 `:273-295` 以同形态输入断言 assembly 为 nil 且错误文本等于完整 closed 清单 |
| 13 | 无第二 runtime/Store/pool/router/authority/fallback；不新建 authority、不扩 S10.5 | PASS | `recovery_runtime.go` 中 `recovery.NewRuntime(`=1、`identity.NewRecoveryRunStore(`=1、`article.NewPostgresReplayEvidenceStore(`=1、`newRecoveryAnchorRegistry(`=1、`pgxpool.New(`=0、`http.NewServeMux(`=0；`newRecoveryRootAssembly` 定义 1 处；测试 `:337-361` 断言 `assembly.runtime`、`composition.runtime()`、`ports.Runtime`、`bridge.boundRuntime()` 同一值且 Store/Closures 指针等于注入值 |
| 14 | `s421-handoff.md` 字段完整 | PASS | 分项 `:3-5`、当前状态 `blocked-upstream` `:6`、承接卡 S4.21 `:7`、剩余 closed 端口清单 `:11-25`（11 项齐全）、消费者 S8.2 `:8`、证据指针 `:33-41`、责任人/日期 `:9`，另有边界声明 `:43-47` |
| 15 | 无越界：无第 4 个代码文件、`internal/**` 零改动、未改既有测试文件、未改其他任务卡 | PASS | 当日（2026-11-01）修改的 `.go` 仅 `recovery_runtime.go` 与 `recovery_enter_install_integration_test.go`；`internal/` 当日修改文件数=0；既有测试文件 mtime 未变；`tasks.md` 哈希 `B739369773DAB6BD041AC6AFC7912379642F5ECD4A223D71670A2B1B775EC9AF` / 806010 B / 3029 行，S4.21 仍 `[-]`、S8.2 仍 `[-]`；本次评审未写评审目录外任何文件 |
| 16 | 不宣称 Case18/S8.2 PASS、不解除 `blocked-upstream` | PASS | handoff `:6`、`:45-47`；实施 `report.md:85-87`；新测试文件头注释 `:24-27`、函数注释 `:265-268`、`:376-380` 均显式否认 production 可达/完整成功 |

合计：16 条中 16 条在本次修订可核成立，无 FAIL 项。第 5 条的“逐项正控/负控闭合”依据测试源码的表驱动逐例断言，而非仅统计 pass 数量。

## 4. Findings

### S421-CR-F001 — LOW（非阻断）：Lead 复跑日志的命令文本与卡面验收命令不一致

- **位置**：`.spec/wip/recovery-runtime/evidence-b13h-r01/s421-lead-verify-r01/commands.log:21-22`
- **违反依据**：`tasks.md:3004-3005` 要求第 11/12 条为 `-run '^(TestRecoveryConfig|TestRecoveryCommand|TestRecoveryEnterInstallConfigCommandMatrix)$'`；Lead 日志只写 `-run TestRecoveryEnterInstallConfigCommandMatrix`。
- **触发场景**：后续读者按该日志文本复现第 11/12 条命令，或在审计中比对日志与卡面命令清单。
- **影响**：仅影响证据可追溯性与可复现性，不影响验收结论。Lead 的 `c11`/`c12` JSON 同时含 `TestRecoveryConfig`（1）、`TestRecoveryCommand`（1）与矩阵（24）测试事件，共 26 个测试级 pass 事件 + 1 个包级 = 27，说明其实际执行的是卡面三重正则；本次独立复跑同一三重正则在普通与 `-race` 下均 27 pass、0 fail。
- **修复建议**：把 `commands.log:21-22` 的命令文本改为卡面三重正则，日志文件名（`c11`/`c12`）不变。不需要改代码或测试。
- **证据**：本目录 `c11-config-command.json`、`c12-config-command-race.json` 的测试名分组统计；本文件 `commands.log` 第 3 节。
- **复审条件**：Lead 日志文本与 `c11`/`c12` 实际测试名集合一致即关闭；如需重出证据，重跑同一三重正则覆盖原文件即可。

无 HIGH / MEDIUM finding，无阻断项。待决定事项：无（F001 的修复方式由 Lead 决定，不影响本卡正确性判断）。

## 5. 已核实事实与非阻断观察

- **precheck 位置**：`:527-534` 的选择分支先判定 action，`enter`/`install` 立即调用 `recoveryEnterInstallMissingPorts` 并在缺项时返回；`:537` 才有第一个构造点（`identity.NewRecoveryRunStore`）。因此不是“仅在 `recovery.NewRuntime` 前”拦截，而是覆盖 runs/closures/anchor registry/journal/secret source 全部构造。
- **错误文本确定性**：清单按固定顺序追加，`strings.Join` 一次性拼接；不存在“只报第一项”；顺序漂移会被测试 `:271`/`:285` 的精确相等断言捕获。
- **测试未冒充 production 可达**：production 形态用 `&pgxpool.Pool{}` + `s421Prepared()` + enter/install plan 构造，与 `main.go:634-650` 的输入形态一致（无任何 11 端口）；注入形态用 sentinel 端口证明 Install 到达 `VerifyInstallationSource`、Enter 到达 `Exclusive.Acquire` 后即失败，断言均为 `errors.Is(err, errS421Sentinel)`，未断言成功/ready。新测试全文无 `setState`、`public_ready`、`t.Skip`、`t.Parallel`。
- **reflect 读取私有字段可接受**：`s421RuntimePortPointer`（`:248-262`）在 `deps` 字段不存在或目标依赖非非 nil 接口时 `t.Fatalf` 并提示需更新断言，重复字段重命名会显式失败而非静默通过。属“防止第二实例”的必要断言，代价可接受。
- **负控不是仅 `err != nil`**：配置负控 `:443`/`:454` 在解析前后两次 `s421AssertEmptyDir`，`s421AssertEmptyDir`（`:513-521`）对目录条目数做断言；正控 `:417` 亦断言零写入。
- **`-race` 证据真实**：`go test -race -tags=integration -c` 产物 44,960,214 B，`go tool nm` 命中 361 处 tsan/race 符号（含 `runtime.RaceRead`/`runtime.RaceWrite`），探针 `s421-race-probe.test.exe` 留在本目录；JSON 无 `DATA RACE` 文本属无竞争时的正常表现。
- **pass 计数口径差异（非缺陷）**：handoff `:36-37` 的 8/26 与 Lead/本次的 9/27 差异来自“是否计入包级聚合 pass 事件”与“是否计入父测试”：wiring 为 7 子测试 + 父测试 = 8，加包级 = 9；矩阵组为 26 个测试事件，加包级 = 27。实施侧 `report.md:73-76` 记为 8/26，Lead 与本次记为 9/27，两者同源、不矛盾。
- **行号陈旧引用可接受**：卡面 `:482-483`、`:556-557`、`:582-583` 已在 `tasks.md:2971` 明确标注“修订前代码事实”，属可追溯的历史锚点；修订后对应位置为 `:527-534`、`:631-637`（installation seam 仍存在于输入结构 `:443` 与其使用点）。不构成本次实现的错误断言。
- **Red 证据有效**：`red.log` 记录两段 red——先是 `unknown field secretCandidate`（build-fail，EXIT=1），再是仅加字段未改 precheck 时的行为失败（旧文本 + 全注入仍被拒，EXIT=1）；符合“先失败、后实现”的最小证据链，且失败文本与本次修订的实际旧行为一致。
- **实施侧命令日志正确**：`s421-implementation-r01/commands.log` 第 11/12 条使用卡面三重正则，与本次找到的 Lead 日志偏差不同源，故 F001 仅限 Lead 证据文件。

## 6. 未验证边界与未验证风险

- 未运行需要真实 PostgreSQL 的 S4.16/S4.20 生产链测试（如 `recovery_production_chain_integration_test.go`）；仅以 `go vet -tags=integration` 与 `-race -tags=integration -c` 编译证明其可编译、未被本次改动破坏签名。
- 未验证真实部署、E2E、POSIX/ACL、真实 provider、真实备份、secret 轮换/复用行为；这些不属于本卡关闭条件，但也未取得证据。
- 本仓库 `cmd/` 与 `internal/` 整体未纳入 Git（`git status` 显示 `?? cmd/`、`?? internal/`），因此 `git diff` 不能证明“既有测试文件未被修改”或“internal 零改动”。本报告对越界的结论依据是当日 mtime 扫描（仅两个目标文件）与 `tasks.md` 哈希/状态，属较弱但可复核的证据形态。
- 无法独立确认 Lead `c11`/`c12` JSON 与其实时执行一一对应，只能由 JSON 内容（测试名集合、时间戳）推断其命令形态；F001 的定位也受此限制。
- 本次未做人工业务语义签收，未评估 11 端口未来真实实现的接口合理性（超出 S4.21 范围）。

## 7. 结论

**PASS WITH MINOR。** 16 条关闭条件在本修订全部可核成立；唯一 finding `S421-CR-F001`（LOW，非阻断）为 Lead 证据日志的命令文本书写偏差，代码与测试无缺陷，不需要返工。

- 不宣称 Case18 或 `article-management-S8.2` PASS，不解除 `blocked-upstream`，不重开 S4.13/S4.16/S4.18/S4.19/S4.20/S8.2。
- 本结论不代替 Lead/Main 的任务关闭、业务接收、部署授权或 Git 写授权。
- **AI 代理评审，非真人签字。**
