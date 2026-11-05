# identity-auth-S4.21 规划评审报告（R02）

- 角色：add-reviewer（非作者独立 Planning Review）。作者 add-task-planner，actualID `01a19293-a26b-7f23-8a0f-2aab006a63f2`。
- 对象类型：planning / task card + Plan registration。
- 固定对象/修订：
  - `.add/project/knowledge/tasks/tasks.md` 中 `identity-auth-S4.21`：`:2961` 锚点、`:2963` 标题、`:2965-3024` 全部字段。
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md` B13H 登记：`:1309` 外层表、`:1333` 所有权索引、`:1337` 计数、`:1445-1450` DAG `n34` 五边、`:1611` 登记记录、`:1624` 交回段。
- 工作区/分支/HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 对象实测（本端只读复核，与 Lead 现场值一致）：
  - tasks.md：SHA256 `49A4F7D07D213465D03E827E754D932EBC96F96B53F7E456DDBB52CD5684F6A0` / 800323 B / 3024 行 / CRLF=3024 / BARE_LF=0 / 无 BOM（首字节 35,32,73）/ 末尾 CRLF；前 2959 行 782348 B / SHA256 `DF91604B67CF690FE94F5B07ECC048B278FDC09BA275631265B12E03E1EC01FD`。
  - Plan：SHA256 `13B21F56728996DB9E3D29842AE016D9F9BC1DDA5598EF894DA2BBDAB9C8FFC1` / 495502 B / 1628 行 / CRLF=1628 / BARE_LF=0 / 无 BOM / 末尾 CRLF。
- 评审日期：2026-11-01（Asia/Shanghai）。
- 结论：**CHANGES REQUIRED**。
- 阻断项：`S421-R02-F001`（MEDIUM）。
- 非阻断项：`S421-R02-F002`/`F003`/`F004`（LOW）。
- 评审方式：只读。未修改 tasks.md/Plan/设计/测试计划/代码；未运行 go build/test/vet/gofmt、PG、HTTP、service、restore、E2E；未做 Git 写；未清理/删除/移动。仅写本目录 `report.md`、`verdict.md`、`commands.log`。

## 1. 结论摘要

R01 的 HIGH 阻断 `S421-PLAN-F001` 已实质关闭。当前卡面已把交付收窄为“唯一 production root 的显式 closed 端口清单 + 可注入装配 seam + secret candidate seam”，并反复明确 production 默认仍 closed、模拟 seam 不等于 production 可达；`:482-483` 的 precheck 位置在构造 `runs`/`closures`/anchor registry/keyring/journal 之前；`:556-557` 复用既有 `inputs.installation` seam；`:582-583` 新增未导出 `secretCandidate` seam；production `main.go:634-650` 不注入任何 seam。

仍有一个 MEDIUM 证据覆盖缺口：卡面 `:3013` 的 `ConfigCommandMatrix` 与 `:3018` 的“配置/命令旁路运行矩阵逐项正控/负控闭合”没有被 `:2992-3003` 的任何验收命令覆盖。`TestRecoveryRuntimeComposition` 与 `TestRecoveryMainBusinessDispatcher` 均不解析 config/argv；现有 `TestRecoveryConfig`、`TestRecoveryCommand` 未列入命令清单。因此本卡在“当前版本证据”上不能自证该关闭项。

其余为 LOW：`第五文件` STOP 与 3 文件写集的 off-by-one；`s421-handoff.md` 不在 3 文件写集、未进入关闭条件、责任人未指定；11 项 precheck 是 Enter∪Install 并集且测试 (a) 只要求“含缺失端口名”而非“列明全部”。

未发现：把 production `Enter`/`Install` 说成可达；把 Case18/S8.2 说成 PASS 或 blocked-upstream 解除；重开 S4.16/S4.20；requires 不闭合/成环；S8.2 进入 hard requires；Plan 计数或 DAG 五边与卡面不一致；编码/前缀完整性异常。

## 2. R01 finding 状态

| R01 finding | 原级别 | R02 状态 | 依据 |
|---|---|---|---|
| `S421-PLAN-F001` | HIGH | **CLOSED（实质）** | 卡面 `:2963`/`:2965`/`:2967`/`:2971`/`:3009`/`:3012`/`:3018`/`:3020`/`:3024` 统一收窄为“production 默认仍 closed、模拟 seam 不等于 production 可达”；`:482-483` precheck 位于 `:488` `runs`、`:495` `closures`、`:502` anchor registry、`:525` keyring、`:529` journal 之前；`:556-557` 仅复用既有 installation seam；`:582-583` 仅新增命运为 closed 的 secret candidate seam；`main.go:634-650` 未注入 journal/articleSnapshots/restore/consumer/installation/articleAccount/exclusive/commitments/sources/anchor/secretCandidate。 |
| `S421-PLAN-F002` | MEDIUM | **CLOSED（原要求）**；残余另立 `S421-R02-F001` | `:2997-3000` 已补 S4.16 `TestRecoveryRuntimeComposition` 与 S4.19 `TestRecoveryMainBusinessDispatcher` 普通/`-race`；但 `ConfigCommandMatrix` 仍无执行命令覆盖，见 F001。 |
| `S421-PLAN-F003` | LOW | **CLOSED** | Plan `:1624` 已改为“18个新Task（截至 S4.21 登记）”；`:1337` 当前 18 卡/75 边、77 节点/329 边。 |
| `S421-PLAN-F004` | LOW | **CLOSED** | `:3022` 已要求 `main.go` 与 `recovery_runtime.go` 各自记录 SHA256/字节数/mtime，任一变化即 STOP。 |
| `S421-PLAN-F005` | LOW | **CLOSED（原要求）**；残余另立 `S421-R02-F003` | `:3016` 已指定 `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 及字段；但该产物不在 3 文件写集、未进 `:3018` 关闭条件、责任人未指定，见 F003。 |

## 3. R02 findings

### S421-R02-F001 — MEDIUM（阻断）— `ConfigCommandMatrix` 未被执行命令覆盖

- 位置：`tasks.md:3013`（leaf）、`:3018`（关闭条件）、`:2992-3003`（验收命令）；对照 `cmd/han-server/recovery_config_test.go:34/40/69`、`cmd/han-server/recovery_command_test.go:8/14/21/25/39-49`。
- 事实：
  - `:2997-3002` 只运行 `TestRecoveryRuntimeComposition`（`recovery_runtime_test.go:229`，composition ports/exit）与 `TestRecoveryMainBusinessDispatcher`（`recovery_main_dispatcher_integration_test.go:26`，dispatcher 路由/probes），以及新增 `TestRecoveryEnterInstallAssemblyWiring`（`:2980` 定位为装配 seam/secret candidate seam）。三者都不解析 recovery config/argv。
  - 现有 `TestRecoveryConfig` 仅循环 `normal`/`verify` 两个 action（`recovery_config_test.go:40`），非法用例只对 `normal`（`:69`）；`TestRecoveryCommand` 覆盖 enter/resume、enter 安装形态（`--installation-manifest`）、verify、无效/重复/未知 flag（`recovery_command_test.go:14-49`）。这两个正是 `ConfigCommandMatrix` 的正/负控来源，但都不在命令清单。
  - 因此 `:3018` 的“配置/命令旁路运行矩阵逐项正控/负控闭合”与 `:2971` 的“在同修订上回归复跑配置/命令旁路矩阵”没有可由验收命令产生的当前版本证据。
  - 更进一步：即使把上述两个测试加入命令，`TestRecoveryConfig` 也不会覆盖 `enter`/`resume` 的 config 分支——`recovery_config.go:46-50` 只在 enter/resume 时读取 restore/control 键，而 `recovery_config_test.go:40` 只循环 normal/verify。当前 config 侧 enter/resume 仅由 `TestRecoveryProductionChainComplete` 的 `s420ResumeStartupPlan`（`recovery_production_chain_integration_test.go:2870-2893`，需真实 PG）间接覆盖，它同样不在命令清单。
- 影响：关闭时可以在未运行矩阵测试的情况下声称该条闭合；S4.16 `ConfigCommandAndLazySecret` 的 normal/enter 两形态/resume/verify 解析回归可能漏检。该缺口不改变生产仍 closed 的结论，但使“同修订回归”只成立一半。
- 修复建议：把 `go test ./cmd/han-server -run '^TestRecoveryConfig$' -count=1 -json` 与 `go test ./cmd/han-server -run '^TestRecoveryCommand$' -count=1 -json`（各自补 `-race`）加入验收命令；或在可写的 `cmd/han-server/recovery_enter_install_integration_test.go` 内补 config/command 矩阵子测试（无需 PG，函数名可显式列出），并在卡面把 `ConfigCommandMatrix` 五个正控与三个负控逐项映射到测试名/子测试名，在关闭条件列出对应证据。全部保持 NOT RUN。
- 复审条件：命令清单或显式映射覆盖 leaf 的每一项正控/负控，且不以 build/解析通过替代执行结果。

### S421-R02-F002 — LOW — “第五文件” STOP 与 3 文件写集错位

- 位置：`tasks.md:2981`、`:2984`、`:3022`；写集 `:2977-2980`（`recovery_runtime.go` 修改、`main.go` 修改、新增 integration test，共 3 个）。
- 事实：三处 STOP 都写“第五文件/第五写文件”。当前写集是 3 个文件，清单外的下一个文件是第 4 个；`:2984` 虽另有“改动上述以外的包”，但同包 `cmd/han-server` 的第 4 个文件不属于“以外的包”，字面不触发 STOP。
- 影响：实施者新增第 4 个 `cmd/han-server` 文件（例如拆分 seam helper）时，所有权/串行写窗的 STOP 规则不会按字面触发，削弱“清单外写入即停”的控制。
- 修复建议：统一改为“任何清单外文件（第 4 个代码文件）/新 authority/改 identity|article 包即 STOP”，保留“3 个候选（2 改+1 新增）”计数。
- 复审条件：STOP 措辞覆盖任何清单外文件，且与 `:2984` 的 3 文件计数一致。

### S421-R02-F003 — LOW — handoff 产物不在写集、未进关闭条件、责任人未指定

- 位置：`tasks.md:3016`（`Case18BlockedUpstreamHandoff` 指定 `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 及字段）、`:3018`（关闭条件未提 handoff）、`:2977-2984`（3 文件写集/STOP）、`:3020`（S8.2 消费者说明）、`:3022`（初始 `[ ]`）；对照 `.codex/guidance/task-execution-workflow.md:151-158` §10.3（尤其第 1、2 项）。
- 事实：
  - `:3016` 要求留下承接记录，字段包括分项、blocked-upstream、承接卡、剩余 closed 端口、消费者 S8.2、证据指针、责任人/日期。
  - 该文件位于 `.spec/wip/...`，不属于 `:2977-2980` 的 3 文件写集；卡面未说明由 Implementer 还是 Lead/planner 创建，也未说明它是否受 `:2981`/`:2984`/`:3022` 的“第五文件”STOP 约束。
  - `:3018` 的关闭条件没有要求该 handoff 存在或可核；§10.3 第 2 项“实际触达的消费者、设计、任务和交接引用已同步或明确不适用”只由 `:3020` 的 S8.2 状态说明部分覆盖。
  - 本端 `Test-Path` 复核：该 handoff 当前不存在（False）。
- 影响：实施者可能为遵守 3 文件写集而跳过 handoff，或创建它但无所有权记录；关闭时可遗漏 Case18 承接证据，导致“blocked-upstream 有可核承接”只停留在卡面。
- 修复建议：把 handoff 明确为“证据产物，不计入 3 个代码写文件”，指定创建责任（建议 Implementer 创建、Lead/planner 在关闭记录中核证），并把 handoff 或等价的“消费者/交接同步记录”写入 `:3018`；若项目约定下不适用，明确写出理由。
- 复审条件：handoff 的路径、字段、责任人、写集归属和关闭条件四处一致。

### S421-R02-F004 — LOW — 11 项 precheck 是 Enter∪Install 并集，且测试 (a) 未断言“全部”缺失名

- 位置：`tasks.md:3009`（11 项清单与测试 (a)）、`:3018`（“列明全部缺失端口”）、`:2965`/`:2971`；代码：`cmd/han-server/recovery_runtime.go:431-449`（`recoveryRootInputs` 字段）、`:475-486`（precheck 位置）、`:488`/`:495`/`:502-518`/`:523-541`/`:543-562`/`:582-584`（默认/覆盖点）；`internal/recovery/runtime.go:322-335`（`installPreflight`）、`:1497-1513`（`recoveryPreflight`）、`:1522`（`Enter` 显式 secret）、`:1554-1571`（`acquire`）、`:1573-1597`（`enter`）、`:729-736`（`NewRuntime` 必填 Store/Journal/ArticleSnapshots/Closures/Genesis/Exclusive）；`cmd/han-server/recovery_sources.go:470-492`（四个 anchor 端口全 closed）。
- 事实：
  - 11 项与 `recoveryRootInputs` 中“nil 即落入 closed 默认”的可选 seam 一一对应，没有多列/漏列：`runs`/`closures` 在 nil 时由真实 Store 构造（不是 closed 端口），`seal` 是可选钩子，`pool`/`plan`/`prepared` 是前置输入，均不应进入该清单。
  - 但这 11 项是 Enter∪Install 的并集，不是逐 action 实际依赖：`installPreflight` 要求 `Installation`/`ArticleAnchorReader`/`Lifecycle`/`Commitments`/`InstallationSources`；`Enter` 走 `Runs`/`Identity`/`Restore`/`Consumer`/`Exit`/`Lifecycle` + `Exclusive`/`ArticleSnapshots`/secret。Install 不需要 `restore`/`secretCandidate`；Enter 不需要 `installation`/`commitments`/`sources`/`anchor`。统一要求全部非 nil 是保守 fail-closed，方向正确，但“必填输入”措辞没有区分并集与逐 action 条件。
  - `:3009` 测试 (a) 只要求“错误含缺失端口名”，`:3018` 要求“列明全部缺失端口”。测试若只断言一个名字即可通过，不能证明“全部”。
- 影响：保守并集不会放开生产（production 仍 closed），但可能让未来合法的逐 action 接线被过度阻断；同时“全部缺失端口名”的关闭断言缺少等强测试。
- 修复建议：在卡面注明“precheck 为 Enter∪Install 装配 seam 并集（保守 fail-closed）；按 action 的运行时条件依赖另列”；把测试 (a) 改为断言 production 默认输入下错误包含全部 11 个端口名（或列出期望集合），与 `:3018` 对齐。
- 复审条件：并集/逐 action 措辞清楚，测试 (a) 与 `:3018` 的“全部”等强。

## 4. 逐项检查结论

| # | 检查项 | 结论 | 依据摘要 |
|---|---|---|---|
| 1 | 范围对应 S4.20 排除句与 Case18 两个 blocked-upstream | 通过 | `tasks.md:2879` 明写“本卡范围为 resume，不含 enter/install 与 recovery secret candidate 关闭点”；`:2881` 明写“enter/install/secret 的后续承接任务未登记（unknown）”。`:2975`/`:3016`/`:3019`/`:3020` 正确引用 `design-workspace/article-management/test-plan.md:359`（全局恢复请求隔离 blocked-upstream）与 `:360`（完整恢复后生产开流 blocked-upstream），并明确“不声称 Case18 PASS 或整体解除阻塞”“本卡不解除 blocked-upstream、不声称 production 开流”。 |
| 2 | 是否误重开已闭 S4.16 | 通过 | S4.16 `[x]`@2665，关闭记录 `:2727` 自述“生产 CAS/anchor、installation commitment、source verifier、deployment lease、keyring loader、restore adapter 与真实 provider/ACL 未交付，recovery 仍显式 closed”。S4.21 `:2967` 只补“显式 closed 端口清单、可注入装配 seam 与接线，不声称 production 可达”；`:2971` 明写“S4.16 已闭的 normal/resume/verify 解析不重做，不重开 S4.16”；`:3013` 再次写“不重开 S4.16”；`:2973` “不重开任何已关闭卡”。未发现残留“重做”措辞。 |
| 3 | 文件所有权精确、3 文件无重叠冲突、串行写窗、第五文件 STOP | 通过（F002 LOW 措辞） | `:2977-2980` 三文件（`recovery_runtime.go` 修改、`main.go` 修改、新增 `recovery_enter_install_integration_test.go`），`:2984` “3 个候选（2改+1新增）”；`main.go` 与 `recovery_runtime.go` 为唯一 root 共享文件，`:3022` 要求实施前记录两者各自 SHA256/字节数/mtime，任一变化即 STOP；`:2981`/`:2984`/`:3022` 有 identity|article 包/新 authority/第五文件 STOP。三文件与 S4.16/S4.19/S4.20 的重叠均按同一 root 串行写窗处理。F002 只指出“第五文件”与 3 文件写集的 off-by-one。 |
| 4 | requires 闭合且无环、S8.2 仅消费者、DAG 5 边 | 通过 | S4.13 `[x]`@2476、S4.16 `[x]`@2665、S4.18 `[x]`@2775、S4.19 `[x]`@2826、S4.20 `[x]`@2879；`:2986` requires 五项且明写 `article-management-S8.2` 仅后续验证关联/消费者、不写入 hard requires、不形成环。Plan `:1446-1450` 的 `n34` 五条入边为 `n20`/`n23`/`n31`/`n32`/`n33`，节点定义分别对应 S4.13/S4.16/S4.18/S4.19/S4.20（Plan `:1361`/`:1364`/`:1434`/`:1438`/`:1441`），与卡面 requires 完全一致。 |
| 5 | 验收命令可执行、普通与 `-race`、全部 NOT RUN、无以 build 替代验收 | 部分通过（F001） | `:2992-3003` 含 `gofmt -l`、`go build ./...`、两条 `go vet`、三组测试普通/`-race`（runtime composition、main dispatcher、新 assembly wiring）；`:2990` “待实施/NOT RUN”，`:3024` “所有验收命令 NOT RUN”；`:3018` 明写“不得用 build 成功、测试 seam 或直接 `setState` 代替 production 可达、测试、评审或 E2E”。F001 指出 `ConfigCommandMatrix` 无对应执行命令。 |
| 6 | 关闭条件满足 §10.3、无 PASS 预填/日期误写 | 部分通过（F003） | `:3018` 覆盖成果、当前版本证据、独立非作者 CodeReview、Lead 同 revision 核 status/diff/所有权、未真实验证标注；`:3022` 本卡初始 `[ ]`、无完成日期；`:3024` 明确不代替独立评审/人的业务签收/实施授权；无 PASS 预填。§10.3 第 2 项“消费者、设计、任务和交接引用已同步或明确不适用”仅由 `:3020` 的 S8.2 状态说明部分覆盖，handoff 未进关闭条件（F003）。 |
| 7 | 卡内 file:line 指针真实有效 | 通过 | 逐条核对：`design.md:1463` 为 `Install(context.Context, InstallationManifest) error`；`:1520` 为 `RunNormal/Enter/Resume` 同一 root 选择其一；`:1524` 为 `InstallationBinding`；`identity-auth/test-plan.md:837` 为配置/命令旁路矩阵；`article-management/test-plan.md:359`/`:360` 为两个 blocked-upstream；`recovery_runtime.go:482-483` 为 `case "enter","install"` 整路径 closed、`:556-557` 为 installation nil/closed、`:582-583` 为 secret candidate closed；`:502-506` 为 `newRecoveryAnchorRegistry(nil)`、`:537` 为 closed journal anchor、`:543-562` 为 articleSnapshots/restore/consumer/installation/articleAccount；`recovery_restore.go:174-205` 仅接口与构造，`rg` 仅命中测试调用 `recovery_restore_test.go:201`，无生产 source/target/tool 实现；`internal/recovery/runtime.go:1554-1571` 为 `acquire`、`:1573-1597` 为 `enter`、`:322-335` 为 `installPreflight`；`internal/recovery/admission.go:364-375` 为 `readProofAttempt` 读 Journal + ArticleSnapshots；`stories.md:176-212` 为 S04 范围/验收；`tasks.md:2727` 为 S4.16 关闭记录。S4.20 `:2881` 内含旧版代码行号 `recovery_runtime.go:465`/`:523`，但 S4.21 引用的是该句内容（`tasks.md:2879`/`:2881`），不是这些旧行号；属 S4.20 既有文本，不计入 S4.21 指针缺陷。 |
| 8 | Plan 计数/DAG 与卡面一致 | 通过 | Plan `:1309` 外层表 18 唯一文件/24 声明槽；`:1333` 所有权索引标题统一为“enter/install 显式 closed 端口清单与装配 seam（production 仍 closed；不含 resume 主线）”，3 写；`:1337` 当前 18 卡/75 边、全图 77 节点/329 条 requires 边；mermaid 实测 35 节点/75 边，`n34` 5 条入边；tasks.md 共 77 卡（66 `[x]`、9 `[ ]`、2 `[-]`，含 `project-bootstrap-B0.1`@55），依赖行原始 ID 提及 336，扣除 7 个非 hard-requires 提及（S4.19：S4.17/S8.2；S4.20：S4.17/S4.4/S1.5/S8.2；S4.21：S8.2）后为 329；`:1624` 为“18个新Task（截至 S4.21 登记）”；`:1611` 已追加 R02 修订记录。未新增文件、未改 DAG。 |
| 9 | 与 S8.2 `[-]`、S1.5 `[-]` 状态冲突 | 通过 | `article-management-S8.2` `[-]`@1865；`article-management-S1.5` `[-]`@1226；`identity-auth-S1.5` `[x]`@251（澄清：口语“S1.5 [-]”指 article-management-S1.5，不是 identity-auth-S1.5）。S4.21 `:3020` 明写 S8.2 为后续消费者、保持 `[-]`、待本卡与应用独立验证后由 Main 评估；`:2986` 不把 S8.2 写入 hard requires；卡面未声称 S8.2/S1.5 完成或整卡 PASS。 |
## 5. 两轮实际检查范围

### 第一轮：对象、契约与结构

- 重读 R01 `report.md`（全文 140 行）与 `verdict.md`（全文），确认原 findings、判定与复审条件。
- 逐字段读取 `tasks.md:2961-3024`：锚点、标题、目标、类型/Story 与范围、R 对账四标签、设计/测试依据、文件所有权、文件数、依赖、批次关联、验收命令、必要风险 leaf 表、关闭条件、后续验证关联、串行/停止条件、登记依据/未验证边界。
- 对照 S4.20 卡 `:2879-2883`（排除句 `:2879`、unknown `:2881`）、S4.16 卡 `:2665` 与关闭记录 `:2727`、S4.19 卡 `:2826` 与依赖/命令 `:2845-2852`、S4.13 `[x]`@2476、S4.18 `[x]`@2775。
- 对照 Plan：B13H 外层表 `:1309`、所有权索引 `:1316-1333`、计数/DAG 说明 `:1337`、mermaid `:1339-1451`、`n34` 五边 `:1445-1450`、登记记录 `:1611`、交回段 `:1624`。
- 对照 §10.3 六项关闭条件（`.codex/guidance/task-execution-workflow.md:151-158`）与 §10.1/§10.2 三态/写入责任（`:132-149`）。

### 第二轮：独立事实重导与有界消费者检查

- 从代码重导 production `Enter`/`Install` 主路径与 precheck 位置：`cmd/han-server/recovery_runtime.go:420-620`（`recoveryRootInputs` 字段 `:431-449`；入口检查 `:475-486`；`runs` `:488`；`closures` `:495`；anchor registry `:502-506`；覆盖 `:507-518`；genesis `:519`；journal/keyring `:523-541`；closed anchor `:537`；articleSnapshots `:543-546`；restore `:547-550`；consumer `:551-554`；installation `:555-558`；articleAccount `:559-562`；secret `:582-584`；`recovery.NewRuntime` `:588-607`；seal `:611-615`）。
- 核对 anchor registry 四端口 closed：`cmd/han-server/recovery_sources.go:466-496`（`:473` 返回 `Commitments`/`Sources`/`Anchor`/`Exclusive` 四个 closed port）。
- 核对 production 接线点：`cmd/han-server/main.go:628-652`；`newRecoveryRootAssembly` 调用 `:634`，只传 `pool`/`plan`/`prepared`/`closures`/`seal`，未注入 journal/articleSnapshots/restore/consumer/installation/articleAccount/exclusive/commitments/sources/anchor/secretCandidate。
- 核对 runtime 依赖：`internal/recovery/runtime.go:318-340`（`installPreflight` 要求 Installation/ArticleAnchorReader/Lifecycle/Commitments/InstallationSources）、`:729-740`（`NewRuntime` 必填 Store/Journal/ArticleSnapshots/Closures/Genesis/Exclusive）、`:1495-1530`（`recoveryPreflight`；resume 额外 RecoverySecret；Enter 显式 secret）、`:1550-1600`（`acquire` 先 Lifecycle.StopAndJoin 再 Exclusive.Acquire；`enter` 先 acquire 再 begin/Restore.Replace/verifyReplacement/bind/identity）；`internal/recovery/admission.go:25-35`（`ErrAdmissionClosed` 定义）、`:360-380`（`readProofAttempt` 读 Journal + ArticleSnapshots）。
- 核对 restore 生产实现缺失：`cmd/han-server/recovery_restore.go:170-210`（restoreSource/restoreTarget/restoreTool 仅接口；`newControlledRestoreDriver` `:203`）；`rg` 仅命中定义与测试调用 `recovery_restore_test.go:201`，非测试代码只有接口方法声明/驱动内部调用，无生产 source/target/tool 实现。
- 核对测试覆盖：`recovery_config_test.go:34-80`（`TestRecoveryConfig` 仅 normal/verify；非法用例只对 normal）、`recovery_command_test.go:1-55`（`TestRecoveryCommand` 覆盖 enter/resume、enter 安装形态、verify、无效/重复/未知 flag）、`recovery_runtime_test.go:229`（`TestRecoveryRuntimeComposition`）、`:719-770`（`TestRecoveryRootAssemblyWiresRuntime` 用 `&pgxpool.Pool{}` 零值指针构造，说明装配测试可不需要真实 PG）、`recovery_main_dispatcher_integration_test.go:26`（`TestRecoveryMainBusinessDispatcher`）、`recovery_production_chain_integration_test.go:2975-3020`（S4.20 以真实 pool/world 构造 `recoveryRootInputs`）、`recovery_runtime_integration_test.go:1-90`（integration harness 需真实 PG 的 `recoveryChainOpen`）。
- 机械重算：tasks.md 77 卡（66 `[x]`、9 `[ ]`、2 `[-]`；identity/article 76 卡 + `project-bootstrap-B0.1`@55）；77 条依赖行、原始 ID 提及 336，扣除 7 个非 hard-requires 提及后 329；Plan mermaid 35 节点/75 边，`n34` 5 条入边；节点标签 `n20`=S4.13、`n23`=S4.16、`n31`=S4.18、`n32`=S4.19、`n33`=S4.20。
- 复核对象哈希/字节/行数/编码与前 2959 行基线哈希；`Test-Path` 确认新增 integration test `cmd/han-server/recovery_enter_install_integration_test.go` 不存在、handoff `s421-handoff.md` 不存在；确认 HEAD/分支。
- 有界消费者/上游：Case18 `article-management/test-plan.md:352-362`（`:359`/`:360` blocked-upstream）、S8.2 `[-]`@1865、article-management-S1.5 `[-]`@1226、S4.16 关闭记录 `:2727`、S4.20 排除句 `:2879`/`:2881`、Plan 登记 `:1309`/`:1333`/`:1337`/`:1611`/`:1624`。

## 6. 实际读取的文件与区间

- `.add/project/knowledge/tasks/tasks.md`：全文 SHA256/字节/行数/编码；前 2959 行哈希；`:55`（B0 卡）、`:1226`（article-management-S1.5）、`:1865`（S8.2）、`:2476`（S4.13）、`:2665`（S4.16）、`:2725-2729`（S4.16 关闭记录）、`:2775`（S4.18）、`:2826`（S4.19）、`:2845-2852`（S4.19 依赖/命令）、`:2879-2886`（S4.20 范围/依赖）、`:2961-3024`（S4.21 全字段）；机械扫描全部 77 条依赖行。
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`：`:1305-1311`、`:1316-1340`、`:1339-1451`（mermaid 计数）、`:1443-1452`、`:1608-1628`；`n20/n23/n31/n32/n33` 节点定义；全文 SHA256/字节/行数/编码。
- `.codex/guidance/task-execution-workflow.md`：§10 `:128-162`（三态、写入责任、关闭条件）。
- `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-planning-review-r01/report.md`（全文 140 行）、`verdict.md`（全文）。
- `.add/project/knowledge/design-workspace/identity-auth/design.md:1463`、`:1520`、`:1524`。
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:837`。
- `.add/project/knowledge/design-workspace/article-management/test-plan.md:352-362`。
- `.add/project/knowledge/design-workspace/identity-auth/stories.md:176-182`、`:205-212`。
- `cmd/han-server/recovery_runtime.go:420-620`。
- `cmd/han-server/recovery_sources.go:466-496`。
- `cmd/han-server/main.go:628-652`。
- `cmd/han-server/recovery_restore.go:170-210`。
- `cmd/han-server/recovery_config_test.go:34-80`。
- `cmd/han-server/recovery_command_test.go:1-55`。
- `cmd/han-server/recovery_runtime_test.go:1-12`、`:700-770`。
- `cmd/han-server/recovery_main_dispatcher_integration_test.go:26`（函数定位）。
- `cmd/han-server/recovery_production_chain_integration_test.go:2975-3020`。
- `cmd/han-server/recovery_runtime_integration_test.go:1-90`。
- `internal/recovery/runtime.go:318-340`、`:729-740`、`:1495-1530`、`:1550-1600`。
- `internal/recovery/admission.go:25-35`、`:360-380`。
- R01 已完整读取、R02 复用同版本未重读：`.codex/agents/add-reviewer.toml`、`.codex/skills/add-code-review/SKILL.md`、`.codex/skills/add-task-decompose/SKILL.md`、`.codex/guidance/multi-agent-coordination.md`、`.codex/guidance/design-doc-writing-style.md`、`.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`。R02 重读了 `task-execution-workflow.md` §10。

## 7. 未验证边界

- 未运行任何 `gofmt -l`、`go build ./...`、`go vet`（普通与 `-tags=integration`）、`go test`（普通与 `-race`）、PG、HTTP、service、restore、E2E；全部 NOT RUN，不能据此声称构建、测试、竞态或真实恢复通过。
- `cmd/han-server/recovery_enter_install_integration_test.go` 与 `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 当前均不存在；未验证其未来实现能否覆盖卡面 leaf，也未验证 handoff 字段是否会按 `:3016` 填写。
- 未验证真实 provider/ACL/真实备份/部署/E2E/POSIX/进程级唯一 runtime/anchor adapter 注册；这些按卡面仍为 unknown。
- 未审查 S4.13/S4.18/S4.19/S4.20 的实现细节与关闭证据充分性，仅把它们作为上游契约与已闭状态消费。
- 未对全图 329 条边做逐边语义审查；仅机械计数并与 S4.21/B13H 相关边核对。
- 未运行 restore 以证明生产 source/target/tool 缺失；仅以 `rg` 调用点/接口声明作静态证据。
- 工作树 `git status --short` 的既有修改/未跟踪项未归因于本对象，也未清理；本评审未读取其内容作为 S4.21 证据。
- 未验证 Plan 中与本卡无关的批次叙述或其他卡的一致性。

## 8. 身份与责任说明

- 本会话未提供可读取的 Agent actualID；不编造。作者 add-task-planner 的 actualID 按任务给定为 `01a19293-a26b-7f23-8a0f-2aab006a63f2`。
- 本报告是 AI 代理的独立评审结论，不代替人的业务接收、后续授权、实施许可、PG/API 运行授权、清理授权或 Git 写授权。
- 本评审未修改被评审制品；未替作者修复；未声称 PASS 或实施授权。

## 9. 非缺陷观察（不计 finding）

- O01：Plan `:1611` 的登记句仍使用登记时标题“enter/install 运行形态与 recovery secret candidate 来源”，同段后文才记录统一后的标题；并有两个“R02”标签（“R01 规划评审 CHANGES REQUIRED→R02 收窄修订”与“R02 精度修订”）。同段还保留了首次 R02 收窄修订的中间哈希 `A9AE65ABA5919CC4E7A55C8FE7D71BB59913EA4C92116DE6D9661514FC7651DA`/798638 B，随后“R02 精度修订”写明当前 `49A4F7D0…`/800323 B；这是历史登记记录，且同段已写明当前标题与当前哈希，不构成本轮一致性缺陷；若后续整理，可加“登记时标题（历史）”与日期/序号以提升审计可读性。
- O02：tasks.md 共 77 卡，其中 identity/article 76 卡、`project-bootstrap-B0.1` 1 卡；这是 Plan `:1337` 全图 77 节点/329 边的机械基础，不是 S4.21 卡面计数错误。
- O03：口语“S1.5 `[-]`”对应 `article-management-S1.5`@1226；`identity-auth-S1.5`@251 是 `[x]`。S4.21 的 requires 不含任一 S1.5，卡面也未声称其完成，无状态冲突。