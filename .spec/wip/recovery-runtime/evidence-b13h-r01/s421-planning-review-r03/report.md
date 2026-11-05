# S4.21 规划评审报告（R03）

- 评审角色：add-reviewer（非作者独立 Planning Review；AI 代理核证，非真人签字）
- 作者：add-task-planner（任务给定 actualID `01a19293-a26b-7f23-8a0f-2aab006a63f2`）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审日期：2026-11-01（Asia/Shanghai）
- 对象：
  - `identity-auth-S4.21` 卡：`tasks.md:2961` 锚点、`:2963` 标题、`:2965-3027` 全部字段
  - Plan B13H 登记：`plans/ir-01-content-community-batches.md:1309`、`:1333`、`:1337`、`:1445-1450`、`:1611`、`:1624`
- 对象修订（字节级复核，与 Lead 给定一致）：
  - `tasks.md`：SHA256 `7EAB7278D70F6326DBC4B79E496D52AF4152D895A849F5AC2F9F2A57ED77C6CF` / 802158 B / 3027 行 / CRLF=3027 / BARE_LF=0 / 无 BOM（首字节 35,32,73）/ 末尾 CRLF；前 2959 行 782348 B / `DF91604B67CF690FE94F5B07ECC048B278FDC09BA275631265B12E03E1EC01FD`
  - `Plan`：SHA256 `F6C8B85AA86D80C083D3394117A3F877BA70D3C2ADEDE865FA89F3BFA0E9B1CF` / 496438 B / 1628 行 / CRLF=1628 / BARE_LF=0 / 无 BOM（首字节 35,32,230）/ 末尾 CRLF
- 结论：**PASS WITH MINOR**（无阻断项；1 条 LOW finding）

## 1. 方法

1. 第一轮：先读自身上轮 `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-planning-review-r02/report.md:40-86` 与 `verdict.md` 全文，取得 `S421-R02-F001`（MEDIUM）、`F002`/`F003`/`F004`（LOW）原文与复审条件，再逐字段读当前卡面与 Plan 登记。
2. 第二轮：沿真实消费者与代码事实核对（`cmd/han-server` 配置/命令解析、root assembly 构造顺序、`internal/recovery` 准入、既有测试与 `TestMain`），并机械复核计数、DAG、编码前缀。
3. 全部为只读检查；未运行任何构建、测试、vet、gofmt、PG、HTTP、service、restore、E2E；未做 Git 写、清理、删除、移动。本轮只写本目录三个证据文件。

## 2. R02 finding 状态

| R02 finding | 级别 | 本轮结论 | 证据 |
|---|---|---|---|
| `S421-R02-F001` `ConfigCommandMatrix` 无验收覆盖 | MEDIUM（阻断） | **CLOSED** | 新文件写集含第二个顶层测试 `TestRecoveryEnterInstallConfigCommandMatrix`（`tasks.md:2980`）；矩阵内容逐项映射（`:3016`）；验收命令普通与 `-race` 两条（`:3004-3005`）；关闭条件绑定该命令且禁止以解析通过替代执行可达（`:3021`）；风险叶与关闭条件一致（`:3016`/`:3021`） |
| `S421-R02-F002` “第五文件”与 3 文件写集错位 | LOW | **CLOSED** | `:2981`、`:2985`、`:3025` 均改为“任何清单外文件（第 4 个代码文件）”；`:2985` 保留“3 个候选（2改+1新增）”；全域扫描无“第五文件”残留 |
| `S421-R02-F003` handoff 未进写集归属/关闭条件 | LOW | **CLOSED** | `:2983` 明确其为非代码证据产物、不占 3 文件代码写集、不计入“第 4 个代码文件”、由实施者创建、Lead 复核；`:3021` 关闭条件含“存在且字段完整（由实施者创建、Lead 复核；无法创建或字段不完整则不得关闭）” |
| `S421-R02-F004` 并集口径 + 测试 (a) 未断言“全部” | LOW | **CLOSED（附 1 条非阻断观察）** | `:3012` 测试 (a) 改为“错误含全部缺失端口名并与 precheck 清单一致”，与 `:3021`“列明全部缺失端口”等强；并集口径见第 5 节观察 O2 |

## 3. R03 findings

### S421-R03-F001 — LOW — Plan 所有权索引未登记新文件内的第二个顶层测试

- 位置：`.add/project/knowledge/plans/ir-01-content-community-batches.md:1333`（对比 `tasks.md:2980`、`:3002-3005`、`:3016`、`:3021`；Plan `:1611` R03 记录）
- 事实：`:1333` 的 S4.21 所有权行仍写“3 写（`recovery_runtime.go`、`main.go`、新增`recovery_enter_install_integration_test.go`顶层`TestRecoveryEnterInstallAssemblyWiring`）”，只登记一个顶层测试；而卡面 `:2980` 规定该新文件同时含 `TestRecoveryEnterInstallConfigCommandMatrix`，验收命令 `:3004-3005` 与关闭条件 `:3021` 都依赖它；Plan `:1611` 的 R03 记录也已写入该测试名。机械扫描：Plan 中 `TestRecoveryEnterInstallConfigCommandMatrix` 仅出现在 `:1611`，`TestRecoveryEnterInstallAssemblyWiring` 出现在 `:1333` 与 `:1611`。
- 影响：只读 Plan 所有权索引的读者会得到“新文件只声明一个顶层测试”的不完整印象，与卡面/验收命令不一致。该不一致不改变文件数（18 唯一文件/24 声明槽）、写权边界或 DAG，故不阻断。
- 修复建议：在 `:1333` 括号内补第二个顶层测试名（如“顶层`TestRecoveryEnterInstallAssemblyWiring`、`TestRecoveryEnterInstallConfigCommandMatrix`”），不改计数、不改 DAG、不改卡面。
- 复审条件：`:1333` 与 `:1611`、卡面 `:2980` 对同一文件内顶层测试集合的表述一致。

## 4. 逐项必查结论

| # | 检查项 | 结论 | 依据摘要 |
|---|---|---|---|
| 1 | 范围对应 S4.20 排除句与 Case18 两个 blocked-upstream | 通过 | `:2975` 原文引用 `tasks.md:2879`（S4.20“仅 resume，不含 enter/install 与 recovery secret candidate”）与 `:2881`；引用 `article-management/test-plan.md:359`/`:360`，并注明 `:360` 的“本卡”指 S8.2；`:3019` 只要求 handoff 承接记录；`:3023` 保持 blocked-upstream、不声称开流。全文无助记解除或 PASS 表述 |
| 2 | 是否误重开已闭卡 | 通过 | `:2963`“不重做 S4.20 已闭的 resume 主线”；`:2967` 明确 S4.16 的 `ConfigCommandAndLazySecret`/`InstallRootDrainAndSingleBinding` 已在组件层闭合（实测 `tasks.md:2713`/`:2720`），本卡只补唯一 production root 的端口清单/装配/接线；`:3016`“不重开 S4.16”；`:3025` 禁止改写 S4.13/S4.16/S4.18/S4.19/S4.20/S8.2 的 checkbox/日期/requires/写集 |
| 3 | 所有权精度、3 文件无重叠、串行写窗、STOP | 通过 | `:2977-2980` 三文件（`recovery_runtime.go` 改、`main.go` 改、新增 integration test），职责不重叠；`:2989` 与 `:3025` 明确 `main.go`/`recovery_runtime.go` 同属唯一 root 串行写窗，实施前记录两者各自 SHA256/字节数/mtime、任一变化即 STOP；“清单外文件（第 4 个代码文件）/新 authority 即 STOP”已统一（`:2981`/`:2985`/`:3025`） |
| 4 | requires 闭合无环、S8.2 仅消费者、DAG 五边一致 | 通过 | 实测 S4.13 `tasks.md:2476 [x]`、S4.16 `:2665 [x]`、S4.18 `:2775 [x]`、S4.19 `:2826 [x]`、S4.20 `:2879 [x]`；S8.2 `:1865 [-]` 仅作后续验证关联、未进 hard requires（`:2987`）；Plan `n34` 五条入边 `n20/n23/n31/n32/n33` 分别对应 S4.13/S4.16/S4.18/S4.19/S4.20（`:1445-1450`） |
| 5 | 验收命令可执行、普通与 `-race`、全部 NOT RUN、无以 build 替代 | 通过 | `:2994-3006` 列出 gofmt/build/vet/五组测试（S4.16、S4.19、装配、配置命令矩阵，各含普通与 `-race`）；`:3004-3005` 的 `-run '^(TestRecoveryConfig|TestRecoveryCommand|TestRecoveryEnterInstallConfigCommandMatrix)$'` 在 `-tags=integration` 下可选中（前两者无 build tag，集成构建仍编译执行；包内唯一 `TestMain` 在 `recovery_restore_test.go:28`，只做 `GOMAXPROCS` 与子进程分派，不连 PG）；`:3021` 明确不得用 build 成功或解析通过替代；`:3027` 标注全部 NOT RUN、两个测试当前不存在 |
| 6 | 关闭条件满足 §10.3 | 通过 | `:3021` 覆盖成果与验收、handoff 存在且字段完整（消费者/交接同步）、同修订证据与未验证标注、独立非作者 CodeReview、Lead 同 revision 核 status/diff/所有权；`:3025` 标注 Main 接收/实施 START/NOT RUN 证据 pending，无 PASS 预填、`[ ]` 无完成日期 |
| 7 | 卡内 file:line 指针有效 | 通过 | 逐条实测：`recovery_runtime.go:482-483`（enter/install 返回 `ErrAdmissionClosed`）、`:502-506`（anchor registry 四端口）、`:537`（closed journal anchor）、`:543-562`（verified reader/restore/consumer/installation/articleAccount 默认）、`:582-583`（secret 闭包占位）；`recovery_sources.go:470-492`（四端口全 closed）；`recovery_restore.go:174-205`（仅接口/构造，唯一调用在 `recovery_restore_test.go:201`）；`internal/recovery/runtime.go:322-335`（installPreflight）、`:1554-1571`（acquire）、`:1573-1597`（enter）；`internal/recovery/admission.go:364-378`（verified snapshot 读取）；`design.md:1463`（`Install`）、`:1520`（三入口互斥）、`:1524`（`InstallationBinding`）；`identity-auth/test-plan.md:837`（配置/命令旁路矩阵条目，与 `:3016` 五项正控一致）；`article-management/test-plan.md:359`/`:360`；`tasks.md:2879`/`:2881`/`:2727` |
| 8 | Plan 计数与 DAG 与卡面一致 | 通过 | `:1309` 18 唯一文件/24 声明槽；`:1337` 18 卡/75 边、全图 77 节点/329 边；机械复核：卡状态 66 `[x]`/9 `[ ]`/2 `[-]`=77、依赖行 77、依赖行内任务 ID 提法 336、非硬引用 7（`:2845` S4.17+S8.2、`:2902` S4.17+article-S4.4+article-S1.5+S8.2、`:2987` S8.2）→ 硬边 329；mermaid 节点 35、边 75、`n34` 入边 5；`:1611` 的 R03 记录哈希/字节/行数与当前 `tasks.md` 一致；`:1624` 为“18个新Task（截至 S4.21 登记）” |
| 9 | 与 S8.2 `[-]`、S1.5 `[-]` 无冲突 | 通过 | `:3023` 明示 S8.2 保持 `[-]`、不改其九文件写集、不自动关闭；卡面无对 S1.5 状态的改写或依赖；`:3027` 只声明“设计仍 draft、人的精确接纳 unknown” |

## 5. 非阻断观察（不计入 finding）

- O1（矩阵配置口径）：`:3016` 把“enter 安装形态（action install）”列为配置正控。代码事实为 `recovery_config.go:31-52` 只在 `enter`/`resume` 调用 `readAction`，`install` 走 `maintenance` 模式且不读 restore/control 键；卡面未声明 install 的配置集与 enter 不同，但也没有相反断言，实施者按代码推导即可，不构成缺陷。建议实施时在测试用例注释中区分 install 与 enter 的配置键集合。
- O2（并集口径）：`:3012` 的 11 项必填输入是 Enter∪Install 装配 seam 并集（`recoveryRootInputs:431-449` 中 nil 即落 closed 的 10 个可选字段 + 新增 `secretCandidate`），`internal/recovery/runtime.go:322-335` 的 `installPreflight` 与 `:1573-1597` 的 `enter` 实际依赖不同。卡面 `:2971` 已写“precheck 对 enter/install 要求这些输入非 nil”，行为无歧义且为保守 fail-closed；仅缺“并集/逐 action 条件另列”的显式标注，建议后续补充而不阻断。
- O3（测试 helper 命名）：`:2980` 要求新文件自包含 env helper（参考 `recoveryConfigFixture` 风格）。同包内若沿用同名会编译失败；实施时使用不同函数名即可，非规划缺陷。

## 6. 实际读取的文件与区间

- 评审依据：`.spec/wip/recovery-runtime/evidence-b13h-r01/s421-planning-review-r02/report.md:40-86`、同目录 `verdict.md` 全文；`.codex/guidance/task-execution-workflow.md:140-175`（§10.3）
- 被评审制品：`tasks.md:2959-3027` 全文；`tasks.md` 机械扫描（状态计数、77 条依赖行、ID 统计、`第五文件|第 4 个代码文件|清单外文件` 与 `可达|PASS|解除|开流|production` 关键字扫描、L1315/L2845/L2902 非硬引用核对、L2665-2740 风险叶名核对）；Plan `:1309`、`:1333`、`:1337`、`:1445-1450`、`:1611`、`:1624` 全文
- 代码事实：`cmd/han-server/recovery_command.go:1-65`；`cmd/han-server/recovery_config_test.go:1-75`；`cmd/han-server/recovery_command_test.go:1-55`；`cmd/han-server/recovery_runtime.go:425-500`、`:500-586`；`cmd/han-server/recovery_sources.go:468-494`；`cmd/han-server/recovery_restore.go:172-207`；`cmd/han-server/recovery_restore_test.go:1-70`、`:198-204`；`cmd/han-server/recovery_runtime_test.go:200-300`、`:700-770`；`internal/recovery/runtime.go:320-337`、`:725-745`、`:1550-1600`；`internal/recovery/admission.go:360-378`
- 上游依据：`.add/project/knowledge/design-workspace/identity-auth/design.md:1461-1465`、`:1518-1526`；`identity-auth/test-plan.md:835-839`；`article-management/test-plan.md:356-362`；`tasks.md:2879`、`:2881`、`:2727`
- 身份与存在性：`Test-Path cmd/han-server/recovery_enter_install_integration_test.go` = False；`Test-Path .spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` = False（与卡面“当前不存在、NOT RUN”一致）

## 7. 未验证边界

- 受禁限制，未运行 `go build`/`go test`（含 `-race`、`-tags=integration`）/`go vet`/`gofmt`/PG/HTTP/service/restore/E2E；命令可执行性为静态核对（`-run` 正则与 `-tags` 组合、包内唯一 `TestMain` 无 PG 依赖）。
- 新测试文件与 handoff 尚不存在，实施后的真实覆盖强度、矩阵子用例是否逐项落到五个正控/三个负控，均未实测。
- 未复核 R01/R02 之外的历史证据链，也未验证 Plan 其它批次的内部一致性（仅按机械计数核 77 节点/329 边）。
- 未验证真实 provider/ACL/备份/部署/POSIX/E2E，亦未验证“模拟 seam 装配”在 `-race` 下的实际行为。
- 本报告为 AI 代理评审，不代替人的接收、实施授权、测试/PG/API 运行授权或 Git 写授权。

## 8. 身份

- 本会话 add-reviewer 的 actualID 不可得（未由 Host 提供），如实说明、不编造；作者 add-task-planner 的 actualID 按任务给定为 `01a19293-a26b-7f23-8a0f-2aab006a63f2`。