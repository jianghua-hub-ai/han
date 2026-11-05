# identity-auth-S4.21 规划评审报告（R01）

- 角色：add-reviewer（非作者独立 Planning Review）
- 对象类型：planning / task card + Plan registration
- 固定对象/修订：`.add/project/knowledge/tasks/tasks.md` 中 `identity-auth-S4.21`（`:2961` 锚点、`:2963` 标题、`:2965-3019` 全部字段）；Plan `.add/project/knowledge/plans/ir-01-content-community-batches.md` 的 B13H 登记
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 对象实测：tasks.md SHA256 `66AF5B88A3708FE4E7E67FB7761C1174063E9C46C52B478E14AA51947686942E` / 794742 B / 3019 行；前 2959 行 SHA256 `DF91604B67CF690FE94F5B07ECC048B278FDC09BA275631265B12E03E1EC01FD`（782348 B）；Plan SHA256 `F8AB5F05095CA99454EDB3C6D03737BE3E5F7026CC8A26525B310855C9385ADE` / 493270 B / 1628 行
- 评审日期：2026-11-01（Asia/Shanghai）
- 结论：**CHANGES REQUIRED**
- 评审方式：只读；未运行 go build/test/vet/gofmt、PG、HTTP、service、restore、E2E；未修改被评审制品；仅在本目录写入本报告、`verdict.md`、`commands.log`

## 1. 结论摘要

卡面的范围切分、依赖闭合、所有权与停止条件、Plan 登记计数总体成立；未发现把 Case18/S8.2 说成 PASS 或已解除的表述，也未发现重开 S4.16/S4.20 的措辞。

阻断问题是：卡面声称替换 `recovery_runtime.go` 三处 closed 占位后 `Enter`/`Install` 可达，但 production assembly 在 `Enter`/`Install` 主路径上还有多处未声明的 closed 端口；这些端口没有生产实现，且其实现必然落在声明写集之外。按卡面自己的“第五文件即 STOP”规则，实施要么立即 STOP，要么只能用测试 seam 注入模拟端口，从而把“production 可达”降级为测试内可达。该实质问题不能靠改名或措辞绕过。

## 2. Findings（按严重度排序）

### S421-PLAN-F001 — HIGH — `Enter`/`Install` 的“可达”声明超出声明写集与已知端口事实

- 位置：
  - 卡面：`tasks.md:2965`（目标：替换三处占位，承接 `Enter`/`Install`）、`:2971`（待做：接入同一 runtime 的 `Install`/`Enter` 运行形态）、`:3005`（`EnterInstallRuntimeSelectionOnSingleRoot`：真实或模拟边界下 `Enter`/`Install` 可达）、`:3013`（关闭条件：三处 closed 占位“被真实替换”）
  - 代码事实：`cmd/han-server/recovery_runtime.go:480-486`（enter/install 在 assembly 入口直接返回 `ErrAdmissionClosed`）、`:502-506`（`newRecoveryAnchorRegistry(nil)`）、`:543-550`（`articleSnapshots` 与 `restore` 默认 closed）、`:551-554`（`consumer` 默认 closed）、`:555-558`（`installation` 默认 closed）、`:559-562`（`articleAccount` 默认 closed）、`:582-584`（secret candidate 默认 closed）；`cmd/han-server/recovery_sources.go:470-492`（未注册 adapter 时四个 anchor 端口全部 closed）、`:540-574`（verified journal reader 与 restore driver 拒绝）、`:576-583`（consumer 拒绝）、`:618-637`（Journal head anchor 拒绝）、`:641-655`（Article account 拒绝）；`internal/recovery/runtime.go:1517-1535`（`Enter`）、`:1554-1571`（`acquire` 先 `Exclusive.Acquire`）、`:1573-1597`（`enter` 先 `Restore.Prepare`，后 `VerifyRestored`）、`:1600-1630`（`begin` 调 `Restore.Prepare` 与 `readProof`）、`:1793-1808`（`bind`）、`:1811-1851`（`identity` 依赖 `RecoverySecret`）；`internal/recovery/admission.go:364-375`（`readProof` 先读 `ArticleSnapshots`）；`internal/recovery/runtime.go:247-272`（`Install`）、`:322-335`（`installPreflight` 要求 `Installation`、`ArticleAnchorReader`、`Lifecycle`、`Commitments`、`InstallationSources`）；`internal/recovery/installation.go:17-39`（`InstallationDependencies` 还要求 `Anchor`、两个 `VerifiedJournalReader`、`Coordinate`）；`cmd/han-server/main.go:629-650`（production 调用 `newRecoveryRootAssembly` 时未传入任何 seam 端口）
  - 上游事实：`tasks.md:2727`（S4.16 关闭记录明确写“生产 CAS/anchor、installation commitment、source verifier、deployment lease、keyring loader、restore adapter 与真实 provider/ACL 未交付，recovery 仍显式 closed”）
- 事实：
  1. production 路径下 `newRecoveryAnchorRegistry(nil)` 返回四个 closed 端口，`Exclusive.Acquire` 即失败，`Enter` 连 `acquire` 都过不去；即使仅注入 Exclusive，`begin` 仍会撞上 closed `Restore.Prepare` 与 closed `ArticleSnapshots`；`identity` 阶段还会撞上 closed `RecoverySecret`（卡面第三处占位）与 closed `ArticleAccount`。
  2. `Install` 的 `installPreflight` 要求 `Commitments`/`InstallationSources`/`ArticleAnchorReader`，三者均来自上述 closed anchor 端口；`NewInstallationExecutor` 还要求 `Anchor`/`Recovery`/`Article` reader，均无生产实现。
  3. 生产 `restoreSource`/`restoreTarget`/`restoreTool` 实现不存在：`rg` 仅命中接口与 `newControlledRestoreDriver` 的定义（`cmd/han-server/recovery_restore.go:174-208`），唯一调用点在测试 `cmd/han-server/recovery_restore_test.go:201`。要交付生产 restore driver 至少需要改动 `recovery_restore.go` 或新增文件，属于卡面声明的“第四/第五文件”，按 `tasks.md:2981`/`:2984`/`:3017` 必须 STOP。
  4. 卡面只声明了三处占位（`:482-483`、`:556-557`、`:582-583`），未声明 restore driver、verified journal reader、journal head anchor、consumer、article account 以及四个 anchor 端口。实施者若按卡面字面执行，会在替换三处后仍无法让 production `Enter`/`Install` 走通；若改用 `recoveryRootInputs` 的测试 seam 注入模拟端口，则得到的是测试内可达，而 production 默认路径仍 fail-closed，与卡面“唯一 production root 的接线与可达性”“被真实替换”不一致。
- 影响：本卡的中央验收（`Enter`/`Install` 可达）在声明写集内不可达。实施会二选一：按 STOP 规则退回（卡面无法关闭），或以 seam 模拟测试通过并对外宣称可达（把测试证据冒充 production 可达，直接污染 Case18 “完整恢复后生产开流”和 S8.2 的承接前提）。两种情况都会使后续规划与消费者判断建立在错误前提上。
- 修复建议（二选一，不得用措辞绕过）：
  1. 收窄：把目标、`待做`、leaf 与关闭条件改为只承诺本写集能交付的内容（action gate 分流、installation executor 接线、secret candidate 来源），并明确列出仍 closed、导致 production `Enter`/`Install` 不可达的端口；leaf 改为“在测试 seam 注入模拟端口时可走到 X；production 默认路径仍在 Y 处 fail-closed”，关闭条件删除“被真实替换”“可达”或限定为“测试边界可达”。
  2. 扩围：由 Main 授权新增 restore source/target/tool 与 anchor adapter 等生产实现文件，更新设计/写集/停止条件，并为每个端口给出最小正控与 fail-closed 负控；同时与 S4.16 关闭记录 `:2727` 的“未交付”清单对齐，避免把组件层关闭表述为 production 可用。
- 复审条件：卡面逐端口与代码事实对照；若仍声称 production 可达，必须给出每个阻断端口的真实实现与运行证据；若改为模拟边界，必须删除 production 可达/真实替换表述，并明确未验证边界。

### S421-PLAN-F002 — MEDIUM — 验收命令未覆盖卡面自称“同修订回归复跑”的既有矩阵与隔离证据

- 位置：`tasks.md:3008`（`ConfigCommandMatrix`：同修订回归，引用 S4.16 已闭解析）、`:3009`（`ClosedUnknownKeepIsolation`：同修订回归复跑并覆盖 enter/install 形态）、`:2992-2998`（验收命令）
- 事实：验收命令只包含 `gofmt`、`go build ./...`、两个 `go vet`，以及新测试 `TestRecoveryEnterInstallProductionChain` 的普通与 `-race`。没有任何命令运行 S4.16 的 `TestRecoveryRuntimeComposition`（S4.16 卡 `:2694-2696`）或 S4.19 的 `TestRecoveryMainBusinessDispatcher`（S4.19 卡 `:2850-2852`）；后者承载 closed/unknown 503+Trace 与零下游隔离。卡面没有说明新测试的哪些子测试对应这些 leaf，也没有把它们列入关闭证据。
- 影响：关闭条件中的“配置/命令旁路运行矩阵逐项正控/负控闭合”和 `ClosedUnknownKeepIsolation` 可以在没有运行既有回归测试的情况下被声称满足；S4.16/S4.19 的行为回归可能漏检。
- 修复建议：把既有回归命令加入验收（普通与 `-race` 按原卡口径），或明确声明新测试的子测试覆盖映射（子测试名 + 对应 leaf），并要求关闭证据逐项列出实际运行结果。不得只保留 `go build`/`go vet`。
- 复审条件：验收命令或子测试映射能逐项对应 `:3008`/`:3009` 的验收事实，且全部标 NOT RUN。

### S421-PLAN-F003 — LOW — Plan `:1624` 的“17个新Task”与当前 18 张新增卡不一致

- 位置：Plan `:1624`；对照 `:1337`（当前 18 卡/75 边）与 `:1611`（S4.21 登记后 18 卡）
- 事实：`:1624` 仍写“17个新Task全部唯一安排到9批”，且未像 `:1603`/`:1607`/`:1610` 那样标注为历史时点；S4.21 登记后当前新增 runtime 卡为 18 张。
- 影响：当前计数陈述与权威登记不一致，可能误导批次覆盖与后续盘点；不影响 DAG/卡面本身。
- 修复建议：更新为 18，或明确标注该段为 S4.21 登记前的历史记录。

### S421-PLAN-F004 — LOW — 串行写窗只要求记录 `main.go` 的 SHA/字节/mtime，未覆盖同样共享且修改的 `recovery_runtime.go`

- 位置：`tasks.md:3017`；文件所有权 `:2977-2979`
- 事实：卡面把 `main.go` 与 `recovery_runtime.go` 都称为唯一 root 共享文件，且两者都在写集内；但只要求实施前记录 `main.go` 的 SHA256/字节数/mtime，并在变化时 STOP。
- 影响：若另一 writer 在实施窗口内改动 `recovery_runtime.go`，现有停止条件不会触发 drift 复核；串行保证不对称。
- 修复建议：对两个共享文件都记录并在实施前后复核 SHA256/字节数/mtime；任一变化即 STOP 重核。

### S421-PLAN-F005 — LOW — `Case18BlockedUpstreamHandoff` 的验收事实不可执行/不可核

- 位置：`tasks.md:3011`（“获得明确承接条目”）、`:3019`；Case18 `design-workspace/article-management/test-plan.md:359`/`:360`
- 事实：卡面禁止修改 test-plan 等被评审制品，也未指定“承接条目”写入哪个制品/字段；而卡面在规划时已经点名两个 blocked-upstream 分项，实施不会新增可核产物。该 leaf 的验收事实无法与实施动作对应。
- 影响：关闭时可仅指向卡面自身即声称满足，缺少真实交接记录；也可能被误读为 Case18 已被处理。
- 修复建议：明确交接产物与责任人（例如由 Lead/planner 在 tasks.md/Plan 关闭记录中写明“Case18 两项仍 blocked，承接卡为 S4.21，未改 test-plan”），或把该 leaf 从实现验收移到规划/关闭记录，并保留“不声称 PASS/解除”的限定。

## 3. 逐项检查结论

| # | 检查项 | 结论 | 依据摘要 |
|---|---|---|---|
| 1 | 范围对应 S4.20 排除句与 Case18 两个 blocked-upstream | 通过（范围表述）；“可达”另见 F001 | `tasks.md:2879` 明确“不含 enter/install 与 recovery secret candidate”；`:2881` 明确“后续承接任务未登记（unknown）”；`:2975`/`:3011`/`:3019` 正确引用 `article-management/test-plan.md:359`/`:360` 的 blocked-upstream，未称 PASS 或解除 |
| 2 | 是否误重开 S4.16 | 通过（无“重做”措辞）；自洽性见 F001 | 卡面 `:2967`/`:2973`/`:3008` 明确“不重开 S4.16”“不重做 S4.20 resume”；未改 S4.16/S4.20 的 checkbox、日期、requires、写集。但“组件层已闭”不能推出 production 可达（S4.16 记录 `:2727` 自述 restore adapter/provider 未交付） |
| 3 | 文件所有权精确、无重叠、串行写窗、第五文件 STOP | 通过；F004 为 LOW 缺口 | `:2977-2984` 三文件（2 改 + 1 新增），`:2981`/`:2984`/`:3017` 明确第五文件/新 authority/改 identity|article 包即 STOP；`:3017` 明确两共享文件与 main.go 写窗；`main.go` 与 `recovery_runtime.go` 只与已闭 S4.16/S4.19/S4.20 共享 |
| 4 | requires 闭合、无环、S8.2 仅消费者、DAG 5 边 | 通过 | S4.13/S4.16/S4.18/S4.19/S4.20 均 `[x]`；`article-management-S8.2` 只在“后续验证关联/消费者”，未进 hard requires；Plan `:1446-1450` 五条边与实际 requires 一致；机械重算全图 77 节点/329 边、0 无效引用、DFS 无环 |
| 5 | 验收命令可执行、普通与 `-race`、全部 NOT RUN、无 build 替代 | 通过（覆盖问题见 F002） | `:2992-2998` 含 gofmt、build、两条 vet、新测试普通与 `-race`；`:2990`/`:3019` 标待实施/NOT RUN；`:3013` 明确“不得用 build 成功或直接 setState 代替”；F002 指出未跑既有回归 |
| 6 | 关闭条件满足 §10.3 | 基本满足；F005 指出交接不可核 | `:3013` 覆盖成果、当前版本证据、独立 CodeReview、Lead 同 revision 核 status/diff/所有权、未真实验证标注；§10.3 第 2 项的“消费者/交接引用同步”未显式写入（与 F005 相关）；第 6 项以“不代替人的业务签收”明确不适用；卡面 `[ ]`、无完成日期、无 PASS 预填 |
| 7 | 卡内 file:line 指针真实有效 | 通过 | 逐条核对：`design.md:1463`（`Install(...)`）、`:1520`（RunNormal/Enter/Resume 同一 root）、`:1524`（InstallationBinding）；`identity-auth/test-plan.md:837`（配置/命令矩阵）；`article-management/test-plan.md:359`/`:360`（blocked-upstream）；`recovery_runtime.go:482-483`/`:556-557`/`:582-583`（三处占位）；`stories.md:176-212`（S04 范围/验收）；`tasks.md:2879`/`:2881`（S4.20 排除与 unknown） |
| 8 | Plan 计数与 DAG 与卡面一致 | 通过；F003 为陈旧句 | B13H 18 唯一文件/24 声明槽：S4.16 15 + S4.19 新增 1 + S4.20 新增 1 + S4.21 新增 1 = 18；槽位 15+2+4+3=24。runtime 切片 18 卡/75 边；全图 77 节点/329 边（tasks 依赖行原始 ID 提及 336，扣除 S4.19/S4.20/S4.21 行内 7 个非 requires 提及后为 329）；mermaid 35 节点/75 边为 runtime 切片图 |
| 9 | 与 S8.2 `[-]`、S1.5 `[-]` 状态冲突 | 通过 | 卡面 `:3015` 明确 S8.2 保持 `[-]`、不改九文件写集；未声称 S1.5 已完成或整卡 PASS；S1.5 只在上游历史记录中出现，未进入本卡 requires |

## 4. 两轮实际检查范围

### 第一轮：对象、契约与结构

- 逐字段读取 `tasks.md:2961-3019`：锚点、标题、目标、类型/Story 与范围、R 对账、设计/测试依据、文件所有权、文件数、依赖、批次关联、验收命令、风险 leaf、关闭条件、后续验证关联、串行/停止条件、登记依据/未验证边界。
- 对照 S4.20 卡 `:2879-2959`（排除句 `:2879`、unknown `:2881`）、S4.16 卡 `:2665-2740`（关闭记录 `:2727`）、S4.19 卡 `:2826-2875`、S4.18 卡 `:2775`、S4.13 卡 `:2476`。
- 对照 Plan：B13H 外层表 `:1309`、所有权索引 `:1333`、计数 `:1337`、DAG `:1445-1450`、登记记录 `:1611`、交回段 `:1624`。
- 对照 §10.3 六项关闭条件（`.codex/guidance/task-execution-workflow.md:151-158`）。
- 对照必读 Skill/Rule（见第 6 节）。

### 第二轮：独立事实重导与有界消费者检查

- 从代码重导 production `Enter`/`Install` 主路径：`cmd/han-server/recovery_runtime.go:421-620`、`cmd/han-server/recovery_sources.go:436-661`、`cmd/han-server/main.go:576-695`、`internal/recovery/runtime.go:230-360/1505-1700/1786-1905`、`internal/recovery/admission.go:341-400`、`internal/recovery/installation.go:13-70`。
- 核对 restore driver 是否有生产实现：`rg` 仅命中 `cmd/han-server/recovery_restore.go:174-208` 的接口/构造与 `cmd/han-server/recovery_restore_test.go:201` 的测试调用。
- 核对 configuration/command 是否已支持 enter/install：`cmd/han-server/recovery_command.go:29-37`、`cmd/han-server/recovery_config.go:27/227-236`。
- 机械重算依赖图：77 卡、77 条依赖行、原始 ID 提及 336；排除 `S4.19`（2 个非 requires：S4.17/S8.2）、`S4.20`（4 个非 requires：S4.17/S4.4/S1.5/S8.2）、`S4.21`（1 个非 requires：S8.2）后为 329 边；0 无效引用；DFS 无环。
- 机械核对 mermaid：35 个 node 定义、75 条 edge；`n34` 五条入边与 requires 一致。
- 复核对象哈希/字节/行数与前 2959 行基线哈希；确认 `cmd/han-server/recovery_enter_install_integration_test.go` 不存在；只读核对 HEAD/分支/status。
- 有界消费者/上游：Case18 `article-management/test-plan.md:359/360`、S8.2 `[-]`、S1.5 `[-]`、S4.16 关闭记录 `:2727`、S4.20 排除句 `:2879`/`:2881`、Plan 登记 `:1333`/`:1611`。

## 5. 实际读取的文件与区间

- `.add/project/knowledge/tasks/tasks.md`：全文哈希/字节/行数；前 2959 行哈希；`:1237`、`:1865`、`:2476`、`:2665-2740`、`:2775`、`:2826-2875`、`:2879-2959`、`:2961-3019`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`：`:1305-1311`、`:1316-1333`、`:1337-1451`、`:1606-1611`、`:1624`、`:1628`
- `.codex/agents/add-reviewer.toml`（完整）
- `.codex/skills/add-code-review/SKILL.md`（完整）
- `.codex/skills/add-task-decompose/SKILL.md`（完整）
- `.codex/guidance/task-execution-workflow.md`：§10 `:128-163`（§5/§7 在本评审前段已读）
- `.codex/guidance/multi-agent-coordination.md`（完整）
- `.codex/guidance/design-doc-writing-style.md`（完整）
- `.codex/guidance/code-review-checklist.md`、`.codex/guidance/abstraction-principles.md`（完整）
- `cmd/han-server/recovery_runtime.go:421-620`
- `cmd/han-server/recovery_sources.go:436-661`（关键段：`:470-492`、`:540-583`、`:618-655`）
- `cmd/han-server/main.go:576-695`
- `cmd/han-server/recovery_restore.go:151-220`
- `cmd/han-server/recovery_command.go:29-37`
- `cmd/han-server/recovery_config.go:27`、`:227-236`
- `internal/recovery/runtime.go:230-360`、`:1505-1700`、`:1786-1905`
- `internal/recovery/admission.go:341-400`
- `internal/recovery/installation.go:13-70`
- `.add/project/knowledge/design-workspace/identity-auth/design.md:1463`、`:1520`、`:1524`
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:837`
- `.add/project/knowledge/design-workspace/article-management/test-plan.md:359`、`:360`
- `.add/project/knowledge/design-workspace/identity-auth/stories.md:176-212`
- `cmd/han-server/recovery_restore_test.go:201`、`cmd/han-server/recovery_production_chain_integration_test.go:1297`、`:2426`（仅 rg 命中确认调用/构造点）

## 6. 未验证边界

- 未运行任何验收命令；`gofmt -l`、`go build ./...`、`go vet`（普通与 `-tags=integration`）、`TestRecoveryEnterInstallProductionChain` 普通与 `-race` 全部 NOT RUN，不能据此声称构建、测试或竞态通过。
- `cmd/han-server/recovery_enter_install_integration_test.go` 当前不存在；未验证其未来实现能否覆盖卡面 leaf。
- 未验证真实 provider/ACL/真实备份/部署/E2E/POSIX/进程级唯一 runtime；这些按卡面仍为 unknown。
- 未审查 S4.13/S4.18/S4.19/S4.20 的实现细节与关闭证据充分性，仅把它们作为上游契约与已闭状态消费。
- 未验证 production 是否可通过配置注册 anchor adapter（只读代码显示 `newRecoveryAnchorRegistry(nil)`；未运行）。
- 未对 Plan 全量 329 边之外的叙述、其他批次或其他卡做完整一致性审查；本次仅限 S4.21 与 B13H 相关登记。
- 工作树 `git status --short` 存在多项修改与未跟踪项；本评审不判断其归属，也不将其计入本对象证据。

## 7. 身份与责任说明

- 本会话未提供可读取的 Agent actualID；不编造。作者 add-task-planner 的 actualID 按任务给定为 `01a19293-a26b-7f23-8a0f-2aab006a63f2`。
- 本报告是 AI 代理的独立评审结论，不代替人的业务接收、后续授权、实施许可或 Git 写授权。
- 本评审未修改被评审制品；未替作者修复。