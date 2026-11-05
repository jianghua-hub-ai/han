# B13H 批次 Gate（G4-batch）独立评审报告

- 角色：非作者独立 Reviewer（`add-reviewer`；主方法 `add-code-review`，objectType=code；按 code 对象分支）
- 对象：B13H 批次 Gate，含三张已关闭卡 `identity-auth-S4.16`、`identity-auth-S4.19`、`identity-auth-S4.20`；17 个唯一文件 / 21 个声明槽
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审时间：2026-10-31（Asia/Shanghai）
- 结论：**FAIL**（详见同目录 `verdict.md`）
- 本报告与命令证据目录：`.spec/wip/recovery-runtime/evidence-b13h-r01/b13h-batch-gate-r01/`

> 边界说明：本记录为 AI 代理按 `add-code-review` 与 `task-execution-workflow` §7/§10 独立核证，非真人签字；不代替 Lead 的接受/拒绝决定、人的业务接收、部署授权或 Git 写授权。本轮只读，未修改任何源码/测试/设计/任务/Plan/既有证据。

## 1. 对象与固定修订

### 1.1 唯一文件指纹（本会话逐个复算）

`cmd/`、`test/` 尚未纳入 Git 版本控制，一律以 SHA256+字节+行数+内容核验，不以 `git diff`/`git status` 判定这些文件。

| # | 文件 | SHA256 | 字节 | 行 | mtime |
|---:|---|---|---:|---:|---|
| 1 | `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | 5044 | 2026-10-31 19:26:09 |
| 2 | `cmd/han-server/recovery_runtime.go` | `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` | 35832 | 1110 | 2026-10-31 19:25:50 |
| 3 | `cmd/han-server/recovery_runtime_integration_test.go` | `B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A` | 31638 | 769 | 2026-10-31 03:30:10 |
| 4 | `cmd/han-server/recovery_production_chain_integration_test.go` | `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` | 167339 | 3868 | 2026-10-31 20:55:11 |
| 5 | `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `B89DA3791616353ACBAC602B1F22AC98298CFE888FBD55E1D3A4686F7BA67A97` | 16500 | 466 | 2026-10-31 16:06:09 |
| 6 | `cmd/han-server/recovery_runtime_test.go` | `AB1782FF7F9AB398D3BFB1695D6FEBAD7517109A32B6D73688746D264AD56363` | 26928 | 770 | 2026-10-31 03:21:41 |
| 7 | `cmd/han-server/recovery_config.go` | `83FFC39F498468BB81F4ED092862B46D6EE80B45B1BE74D6F502941C2EE606B0` | 9842 | 280 | 2026-10-31 02:04:05 |
| 8 | `cmd/han-server/recovery_config_test.go` | `9071DCABB0619FD482656BA9FE8CBA9A19225048805FF44733472315A78A095E` | 2959 | 75 | 2026-10-31 02:04:05 |
| 9 | `cmd/han-server/recovery_command.go` | `4BCF661DFFD166554C1FF7210936C7B3EB9AF9CC31CE8B555B73E130052123F3` | 2110 | 65 | 2026-10-30 18:03:09 |
| 10 | `cmd/han-server/recovery_command_test.go` | `2FA4BFA09674BED725D4F02F0FDBFBB9EA07AF851D56E1D6DEBBA661054FB17A` | 2326 | 55 | 2026-10-30 18:02:47 |
| 11 | `cmd/han-server/recovery_sources.go` | `935DD7BDC3C3DF4B7FCF045622EEBC95D3BF30DEE313C93D19E1CC335B7DF90C` | 24136 | 662 | 2026-10-31 02:59:12 |
| 12 | `cmd/han-server/recovery_sources_test.go` | `BBCB47359E41A274FC5414FD2F5DFF35FBAE73FE816B76D241764E5A21188EF4` | 14797 | 390 | 2026-10-31 02:59:21 |
| 13 | `internal/identity/recovery_handler.go` | `4754B880E557F0B15BB1385CF13E2B37F08074E9C0F26F6194A4548755A00E52` | 25057 | 649 | 2026-10-31 02:51:03 |
| 14 | `internal/identity/recovery_handler_test.go` | `6643613FE9A9518DC9A242424529DA64CD08BDD2C0A65902E752EC3BEBEA2115` | 42139 | 925 | 2026-10-31 02:50:55 |
| 15 | `cmd/han-server/lifecycle.go` | `0CA9C3391382BDE21C2B71796FE0FC5494319C66697BD9F5C6FB65809988F663` | 21788 | 737 | 2026-10-30 17:59:01 |
| 16 | `cmd/han-server/lifecycle_test.go` | `33D641DF144BC9B39EF9F89A3E8AA310205C0A76C2137CB93522960EE71E4F05` | 45066 | 1364 | 2026-10-31 02:04:05 |
| 17 | `cmd/han-server/main_test.go` | `0E61F4A37D314D2D35B80EA8184BEF17EB6227FCAF1CA0F57C09736005185820` | 399522 | 11189 | 2026-10-31 02:03:49 |

### 1.2 槽位数学与登记

- S4.16 15 文件 + S4.19 2 文件 + S4.20 4 文件 = **21 声明槽**。
- 共享：`cmd/han-server/main.go` ×3（S4.16/S4.19/S4.20）、`cmd/han-server/recovery_runtime.go` ×2（S4.16/S4.20）、`cmd/han-server/recovery_runtime_integration_test.go` ×2（S4.16/S4.20），共 4 个重复槽 → 21 − 4 = **17 唯一文件**。
- 17 文件均落在 Plan 外层 B13H 行（`.add/project/knowledge/plans/ir-01-content-community-batches.md:1309`）与三卡逐文件写集清单内；未发现清单外唯一文件。

### 1.3 Git 状态（仅参考）

- `git diff --cached --name-only` 为空（staged=0）。
- `git status --short` 288 行；`cmd/`、`test/` 为 `??` 未跟踪。本报告不据此判定 `cmd/` 内容。

## 2. 两轮实际检查范围

- **第一轮（全量）**：批次合同 `.add/project/knowledge/plans/ir-01-content-community-batches.md` 的 `<a id="b13h"></a>`（1587）至 `<a id="b13i"></a>`（1600）前全文及外层 B13H 行（1309）；`tasks.md` 三卡卡面与关闭记录（S4.16 约 2659 起、S4.19 约 2820 起、S4.20 约 2872 起，重点 2859/2862/2868/2932/2934/2941/2948）；17 文件全文/指纹；`main.go`/`recovery_runtime.go` 跨卡接口静态路径；`add-code-review`、`task-execution-workflow` §7/§10、`multi-agent-coordination`、`coding-standards`。
- **第二轮（定向复核）**：对第一轮识别的 F001/F002 逐项复算——R03 基线 dispatcher（`F761D7E4…`/20376）与当前 dispatcher（`B89DA379…`/16500）差异；被移除子测试原文（world-hold `.original`）；S4.20 链测试 transport 与 `production_dispatcher_coverage` 日志；`d7-main-dispatcher-r02.json` 子测试清单；S4.20 关闭记录措辞；S4.19 Gate 自身“两层缺一不得通过 B13H Gate”的原文。
- **沿消费者/上游有界检查**：`s419-independent-review-r03/01-objects.log`、`s419-independent-review-r03/report.md`、`s419-lead-gate-r03/report.md`、`s419-world-hold-r01/{README.md,.original}`、`s420-code-review-r01/{report-r02.md,verdict-r02.md,lead-recheck-r02.md}`、`s420-implementation-r01/{d7-*.json,d7-tooling-r02.log,progress.md}`。

## 3. 维度 1：跨卡接口一致性 — PASS（静态/指针层）

`cmd/han-server/main.go`（SHA `6D5373CA…`/173317）：

- **唯一 root/handle**：`recoveryHandle func() recovery.Runtime` 字段 `:344`；晚绑定闭包 `:546-551`；注入 builder `:601`；启动 `:634` 调 `newRecoveryRootAssembly`；完成后回填 `:655-657`。
- **唯一 Runtime**：`recovery_runtime.go:475` 唯一 `newRecoveryRootAssembly`；`:588` 唯一 `recovery.NewRuntime`。全仓 `recovery.NewRuntime` 仅此一处；未发现第二 runtime/pool/authority/fallback。
- **同一 article evidence Store**：`main.go:338` 字段 `articleEvidence *article.PostgresReplayEvidenceStore`；`:568` 创建；`:588` 注入 composition；`:3299 newArticleReplayProvider` 内 `:3333 store := b.composition.articleEvidence`（仅在 nil 时创建），即 replay provider 复用同一 evidence store，未另建第二 Store。
- **唯一 router 与顺序**：`main.go:3542 newLoginBusinessRouterWithTraceAndRecovery`；exact path/method switch 最先（`/livez:3561`、`/readyz:3565`、其余精确 API 路径 `:3570-3660`）；未知路由 404 `:3660-3666`；方法 405 + `Allow` `:3668-3675`；随后 guard：probe 由同 handle 提供（`:3678-3684`，`Readiness`/`Liveness`），业务请求 `:3694-3696` 先 `guard.Protect(handler)`，再进 handler；handler 自身包 `origin.Protect(authorizer.Protect(session.AccessGeneral, …))`（如 `:3042`/`:3244-3250`），故 **guard 在 exact router 之后、Session/AccessGeneral 之前**；Trace 最外层 `:3707`。
- **`/livez` `/readyz` 同 handle**：`recoveryGuardFromRuntimeHandle(recoveryHandle)`（`:3678`、`:3694`）同一来源；probe 与业务共享同一 `recoveryHandle`。
- **不建第二 router/pool/authority/fallback**：`http.NewServeMux()` 仅 `main.go:4034`，位于 legacy `startupDependencies.serve`（`runLegacy` 默认路径）；生产 `newServer("business", businessHandler)`（`:705`、`:4050`）使用真实 handler，非第二公共面。
- **resume 完整完成→`public_ready`→同会话开流**：该语义在 S4.20 链测试中以真实 `RunNormal` 重建（`recovery_production_chain_integration_test.go:3814`），但使用 **测试 transport 适配器**（`:3767-3773`），并显式把 production dispatcher 覆盖委托给 `TestRecoveryMainBusinessDispatcher`（`:3867` 日志 `transport=test_adapter … production_dispatcher_coverage=TestRecoveryMainBusinessDispatcher`）。production dispatcher 侧的“真实 resume→同会话 200”缺失，见 F001。

结论：跨卡接口在指针/静态层一致，未发现第二 runtime/router/pool/authority/fallback；但“进程级全局唯一 runtime”仅指针/组装层证明，未做运行期隔离断言（列为未验证边界）。

## 4. 维度 2：迁移与回归 — FAIL（F001）

### 4.1 S4.16 既有 `TestRecoveryRuntimeProductionChain` 保持 — PASS

- `cmd/han-server/recovery_runtime_integration_test.go` SHA `B8DEBCA4…`/31638/769 行，与 `lead-gate-r05/files-sha256-r03.json` 登记逐项一致（S4.16 基线），即 S4.20 未改动该文件。
- `func TestRecoveryRuntimeProductionChain` 仍在 `:165`，原名；主体沿用 `recoveryChainOpen` / `recoveryTestPermitBinding` / `newRecoveryControlHandle` / `newRecoverySecretSource` / `newRecoveryRuntimeComposition` / `bridgeValue` / `platform.NewHTTPAuthorizer` 等原符号与断言（`:166-231` 抽样核对）。
- 该测试在 `127.0.0.1:53519`，本轮**未运行**（不在授权命令集内），保持 NOT RUN；S4.16 关闭记录 `tasks.md:2868` 亦声明其保持 NOT RUN。不得把“未跑”当“通过”。

### 4.2 S4.19 同修订双层 — FAIL

合同依据（实读原文）：

- `tasks.md:2859` leaf `PublicReadyAdmitsSameSession`：“两层证据同修订必跑：(a) `TestRecoveryRuntimePostgres`（真实 runtime + 真实 PostgreSQL）驱动 Begin→Consume→succeeded→Exit/join（三 worker，holders=0）→activate→`public_ready`…；(b) 本 leaf 证明同一 production handler、同一 Session 在 closed 下 503（acquire=1 release=0），共享 runtime 进入 `public_ready` 后同一请求 200，Trace 保留、Article 正常调用、acquire/release=2/1。无单测试跨层串联两层，不得声称单测试证明完整链…”。
- `tasks.md:2862` 关闭条件：`TestRecoveryMainBusinessDispatcher` 在真实 `cmd/han-server` production dispatcher 上普通与 `-race` 通过，并含 closed/unknown、livez/readyz、Origin/CSRF/Session/AccessGeneral、收窄后的两层证据。
- `tasks.md:2868` S4.19 Lead Gate R03 关闭记录：“B13H 批 Gate 仍须同修订同跑两层，缺一不通过”。
- `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-lead-gate-r03/report.md` §5：“层 a 真实 PG 全链与层 b production dispatcher 同 Session 200 均为必跑证据，两层缺一不得关闭或通过 B13H Gate”；“若两层未同修订同跑，定向测试无法发现 join/final proof/open 许可回归；由 B13H 批 Gate 强制同跑承担”。

层 (a) 真实 PG 全链 — PASS：

- `TestRecoveryRuntimePostgres` 定义于 `test/integration/recovery/runtime_admission_test.go:11`（**不在 `cmd/han-server` 包内**）。授权命令 `./cmd/han-server -run '^(…|TestRecoveryRuntimePostgres|…)$'` 实际不会匹配到它；真实 PG 由 `./test/integration/recovery` 单独执行（见 §8 r2-07/r2-08）。
- 真实 PG 日志含 `observation=public_ready`、`three worker join complete: workers=3 … holders=0`、`acquire_after=ErrAdmissionClosed`、`Case18SameRevisionAndNoProductionClaim`；普通与 `-race` 均 fail=0/skip=0/DATA RACE=0。

层 (b) production dispatcher 同 Session 200 — **FAIL**：

1. **R03 基线确有层 (b)**：`s419-independent-review-r03/01-objects.log` 记录 dispatcher SHA `F761D7E4FE4FCE03C50A3916228EA97243C7730BD2C1BE6EAEA958C8B73AAFD7` / 20376 B / mtime 14:35:59；`s419-independent-review-r03/report.md:92-94` 记录 closed 503、`public_ready` 200、Trace、`repository.lists==1`、resolver 1 次、`acquire/release==2/1`、同 Session cookie、`FailureNegativesNotMasked`、`AdmissionIsNotSticky`。其原函数 `testRecoveryMainOpenAfterCompleteSoftwareRecovery`（world-hold `.original:166-305`）用**真实** `recovery.NewRuntime(ctx, world.dependencies())`（`:169`）+ `runtime.Enter(...)`（`:201`）完成恢复→`public_ready`，再以同一请求 200（`:224`）、同 cookie（`:234`）。
2. **该层在 gate 修订已不存在**：当前 dispatcher SHA `B89DA379…`/16500/466 行（mtime 16:06:09），`s419-world-hold-r01/README.md` 说明原 subtest 因 `undefined: newS419RecoveryWorld` 阻断 `go vet -tags=integration` 被移除，原文完整保留为 `.original`（SHA `36DA30ABD77EAC4EE8E6098F588677B2A20BB4A854B6FEA8F161168D2D634391`/22209，本会话复算一致）。当前 `TestRecoveryMainBusinessDispatcher` 仅 5 个子测试（`:27-31`：`ExactRouteBeforeGuard`、`ClosedUnknownTwoAdminGets503`、`LiveReadyProbes`、`SingleRuntimeHandle`、`NoRecoveryConfig`）；全文 `recovery.NewRuntime`=0、`runtime.Enter`=0、`PublicReadyAdmitsSameSession`/`FailureNegativesNotMasked`/`AdmissionIsNotSticky`/`OpenAfterComplete` 均 0 命中。
3. **替代测试是弱代理**：`LiveReadyProbes`（`:117-162`）确实覆盖 production handler、同 cookie、closed 503 acquire=1、`setState("public_ready")`（`:143`）→ `/readyz` 200 → 业务 200、acquire/release=2/1，且 `fixture.serve` 会断言 Trace；但它在 `:143` **直接 `setState("public_ready")`**，运行时是测试 stub `s419RecoveryRuntime`（`progress.md:44` 自述“closed/unknown 用，禁止成功链”），并非真实 `recovery.NewRuntime` 经 `Enter` 到达 `public_ready`；也未显式断言同 Session 解析次数与 Article 下游计数。
4. **S4.20 未在 production dispatcher 上重建**：`ClosedUnknownKeepIsolation`（`recovery_production_chain_integration_test.go:3759-3867`）用真实 `RunNormal`（`:3814`）→ 同会话 200（`:3815-3822`），但 routing 是**测试适配器** `s420NewCase18Transport`（`:1982-1990`；`:3767-3773` 明确“TEST adapter”），且 `:3867` 日志把 `production_dispatcher_coverage` 显式指向 `TestRecoveryMainBusinessDispatcher`——即又把层 (b) 交回给缺少真实 resume 的测试。
5. **S4.20 关闭条件明确禁止此替代**：`tasks.md:2934`：“不得用直接`setState`或build成功代替 resume 恢复运行完整完成、测试、评审或E2E。” 当前 production dispatcher 的 closed→`public_ready` 恰是直接 `setState`。
6. **关闭记录证据不含层 (b)**：`d7-main-dispatcher-r02.json`（SHA `A673F00E92532CFCC1FDBEADA5FF17EC468F2DF0E0734DC139568514503A79F7`/9596）与 `-race-r02`（`08407B0F…`/9595）子测试清单仅为上述 5 项（含其子分组），无 `PublicReadyAdmitsSameSession`/真实 resume 层 (b)。

结论：在 gate 修订上，**不存在**“真实恢复完成→`public_ready`→同会话 200 经 production dispatcher”的证据；R03 认证的层 (b) 被移除后未在 production dispatcher 上重建。违反 S4.19 “两层缺一不通过”与 S4.20 “不得用 setState 代替 resume 恢复运行完整完成”。单卡 R03 PASS 不能替代本批复跑 → 批次 Gate 不成立。

## 5. 维度 3：所有权与串行 — PASS

- 21 槽/17 文件数学与 Plan `:1309` 一致；17 文件全部在登记清单内。
- 串行交接按 mtime 可核：S4.16（`recovery_runtime_integration_test.go` 03:30:10、其余 02:00–03:21）→ S4.19（`main.go` 10:27 基线、dispatcher 16:06:09）→ S4.20（`recovery_runtime.go` 19:25:50、`main.go` 19:26:09、链测试 20:55:11）。共享文件无并行覆盖迹象。
- 无活动写者：任务记录 `tasks.md:2945` 披露的另一宿主 `xDesk.exe`（PID 27012）在本轮复核时已不存在（`Get-Process -Id 27012` → alive=false）；复核前后 17 文件 mtime 未变。
- 未发现清单外越界写入、未发现删除/移动/归档；`git diff --cached` 为空；本轮未做任何 Git 写。
- 结论：所有权与串行 PASS；残余风险为“仅凭 mtime/SHA 判定，非文件锁或进程级写者证明”。

## 6. 维度 4：批次级集成/覆盖 — 不足（受 F001 阻断）

- 授权命令本轮全部通过：`go build ./...`、`go vet`（普通与 `-tags=integration`）、17 文件 `gofmt -l`、dispatcher+chain（普通与 `-race`）、真实 PG `TestRecoveryRuntimePostgres`（普通与 `-race`），均 exit 0，fail=0、skip=0、DATA RACE=0（见 §8）。
- 但“单卡 PASS 不能相加成 G4-batch”（Plan `:1598`）：三卡各自关闭记录不能替代批次级同修订复跑。层 (b) 在 gate 修订缺失（F001），故本批**未达到可交回的批次级覆盖**。
- 已覆盖：跨卡单 handle/store/router/order/probe 静态一致；层 (a) 真实 PG 全链；S4.16 迁移保持；S4.20 链测试真实 `RunNormal`→200（测试 transport）；production dispatcher 的 closed/unknown 503、livez/readyz、405/404/Allow、无配置跳过 guard、单 handle 晚绑定、panic 释放。
- 未覆盖：见 §9。

## 7. Findings

### B13H-GATE-F001（HIGH，阻断）

- **位置**：`cmd/han-server/recovery_main_dispatcher_integration_test.go`（SHA `B89DA379…`/16500）`LiveReadyProbes:143`；对照 `s419-world-hold-r01/recovery_main_dispatcher_integration_test.go.original:166-305`；`cmd/han-server/recovery_production_chain_integration_test.go:3767-3773,3814,3867`；`.spec/wip/recovery-runtime/evidence-b13h-r01/s420-implementation-r01/d7-main-dispatcher-r02.json`。
- **违反依据**：`tasks.md:2859`（layer b）、`:2862`、`:2868`（“两层缺一不通过”）、`:2934`（“不得用直接 setState 代替 resume 恢复运行完整完成”）；`s419-lead-gate-r03/report.md` §5（“两层缺一不得…通过 B13H Gate”）；Plan `:1597`/`:1598`。
- **触发场景**：按 B13H 批 Gate 合同，在 gate 修订上执行 S4.19 两层同跑，并尝试核对“真实恢复完成→`public_ready`→同会话 200 经 production dispatcher”。
- **影响**：R03 认证的层 (b)（真实 `recovery.NewRuntime` + `Enter` → 同 Session 200）在 gate 修订已被移除；现 production dispatcher 仅以 stub `setState("public_ready")` 到达开流，S4.20 链测试又以测试 transport 承载并显式把 dispatcher 覆盖交回该不完整测试。故“两层缺一不通过”被违反，批次 Gate 不能判定通过；join/final proof/open 许可回归缺少合同要求的同修订证据。
- **修复建议（不在本轮执行）**：由作者在 production dispatcher 测试中以真实 `recovery.NewRuntime` + 真实恢复完成→`public_ready` 重建同 Session 200（含 Trace、Article 下游、`acquire/release=2/1`、失败负控），或在 Lead 正式收窄合同后以书面决定替换该层并同步卡面/Plan/Gate 口径；不得用 `setState` 或测试 transport 冒充。
- **证据**：§4.2 全部条目；§8 命令结果；`01-objects.log`（R03 基线 SHA）；world-hold README/.original；`d7-main-dispatcher-r02.json` 子测试清单。

### B13H-GATE-F002（MEDIUM，非阻断于代码但影响交付完整性）

- **位置**：`tasks.md:2932`（`MainWriteWindowFrozenAndS419Reverified`：“关闭前在当前 SHA 上重跑 S4.19 dispatcher/probe 两层并记录”）、`tasks.md:2948`（S4.20 关闭记录：“…`TestRecoveryMainBusinessDispatcher` 普通与 `-race` 均 EXIT=0…；closed/unknown…成功后进入 `public_ready` 开流均按卡面 oracle 闭合”）；`s420-implementation-r01/d7-main-dispatcher-r02.json`、`-race-r02`。
- **违反依据**：`tasks.md:2859`/`:2868` 与 `s419-lead-gate-r03/report.md` §5 的“两层”要求；`task-execution-workflow` §10 的“成果/证据与验收一致”原则。
- **触发场景**：核对 S4.20 关闭记录所称“在当前冻结 `main.go` SHA 上重跑 S4.19 dispatcher/probe 两层”是否真的覆盖层 (b)。
- **影响**：关闭记录把 dispatcher 复跑表述为满足“两层/按卡面 oracle 闭合”，但实际 `d7-main-dispatcher-r02.json` 仅含 5 个子测试，不含真实 resume 的层 (b)；证据表述强于实际覆盖，可能使 Lead/后续消费者误判 S4.19 门禁已在本批复跑。
- **修复建议（不在本轮执行）**：在关闭记录中如实限定 dispatcher 复跑覆盖范围（层 a 由 PG 测试、层 b 待重建），或补齐层 (b) 后再声明“两层闭合”。
- **证据**：§4.2 第 6 条；`d7-main-dispatcher-r02.json`（`A673F00E…`/9596）子测试清单；`tasks.md:2932`/`:2948` 原文。

### 非阻断观察 N01（LOW，建议）

- `cmd/han-server/recovery_main_dispatcher_integration_test.go:421 publishPublicReady` 定义后全文件无调用（count=1）；同时 `s420-implementation-r01/progress.md:44` 自述 `s419RecoveryRuntime`“closed/unknown 用，禁止成功链”，但 `LiveReadyProbes:143` 用它走 `setState("public_ready")` 成功链，披露与实现不一致。属测试卫生/披露问题，不单独阻断；与 F001 同源。

## 8. 命令与结果（本会话第一手复跑）

环境：`D:\AID\VSCode\Codex\Han`；PowerShell；`HAN_TEST_POSTGRES_DSN` 已存在（本报告不记录其值）。原始输出：`r2-01…r2-08`（另保留前序 `01…08`，未覆盖）。

| # | 命令 | exit | run | pass | fail | skip | DATA RACE | 证据文件 |
|---|---|---:|---:|---:|---:|---:|---:|---|
| 1 | `go build ./...` | 0 | - | - | - | - | - | `r2-01-build.log`（空） |
| 2 | `go vet ./cmd/han-server` | 0 | - | - | - | - | - | `r2-02-vet.log`（空） |
| 3 | `go vet -tags=integration ./cmd/han-server` | 0 | - | - | - | - | - | `r2-03-vet-integration.log`（空） |
| 4 | `gofmt -l <17 files>` | 0 | - | - | - | - | - | `r2-04-gofmt.log`（空） |
| 5 | `go test -tags=integration ./cmd/han-server -run '^(TestRecoveryProductionChainComplete\|TestRecoveryRuntimePostgres\|TestRecoveryMainBusinessDispatcher)$' -count=1 -json` | 0 | 19 | 20 | 0 | 0 | 0 | `r2-05-cmd-integration.json` |
| 6 | 同上 + `-race` | 0 | 19 | 20 | 0 | 0 | 0 | `r2-06-cmd-integration-race.json` |
| 7 | `go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | 6 | 7 | 0 | 0 | 0 | `r2-07-postgres.json` |
| 8 | 同上 + `-race` | 0 | 6 | 7 | 0 | 0 | 0 | `r2-08-postgres-race.json` |

补充说明：

- 第 5/6 条的 `-run` 虽列出 `TestRecoveryRuntimePostgres`，但该测试不在 `cmd/han-server` 包（定义于 `test/integration/recovery/runtime_admission_test.go:11`），故第 5/6 条实际只执行 `TestRecoveryProductionChainComplete` + `TestRecoveryMainBusinessDispatcher`；层 (a) 真实 PG 由第 7/8 条单独执行。
- 第 5/6 条 `pass=20` 含 package 级 pass 事件；测试级为 `TestRecoveryProductionChainComplete`（8 叶）+ `TestRecoveryMainBusinessDispatcher`（5 子测试及子分组）。
- 第 7/8 条含真实 PG 标记：`observation=public_ready`、`workers=3`、`holders=0`、`acquire_after=ErrAdmissionClosed`、`Case18SameRevisionAndNoProductionClaim`。
- DSN 泄漏检查：`r2-07`/`r2-08` 中 `postgres://`/`postgresql://` 命中 0；报告与日志未写入 DSN 明文。

## 9. 未验证边界（全部列出，未跑≠通过）

- 性能/压力/容量与延迟预算。
- 真实部署环境、真实 Host E2E、真实 HTTP 服务端到端（本轮仅 httptest/测试装配）。
- POSIX journal/fsync/flock/CAS 语义。
- 真实 keyring/HSM/密钥托管（S4.20 使用模拟边界，已披露）。
- approved backup inventory 与真实 restore adapter。
- production dispatcher **全路由** closed/unknown 矩阵（当前仅选定路由；S1.5 卡范围）。
- 进程级全球唯一 runtime 的运行期隔离断言（仅指针/组装层证明）。
- 真实 provider/ACL/CAS/anchor/lease/restore 行为。
- `enter`/`install` 与 recovery secret candidate 的承接任务（仍未登记，unknown）。
- `TestRecoveryRuntimeProductionChain`（`127.0.0.1:53519`）本轮 NOT RUN。
- `goimports`/`golangci` 未运行；非全仓格式检查。
- 人的业务接收、部署授权、Git 写授权。
- S4.20 链测试的 `R02-F004` 残余：closed/unknown 非 production dispatcher E2E（已被 R02 列为 LOW 并披露）。

## 10. 证据文件与本轮边界

- 本目录原始输出：`r2-01-build.log`…`r2-08-postgres-race.json`；前序 `01-build.log`…`08-postgres-race.json` 保留未覆盖。
- 本轮仅新建 `report.md`、`verdict.md`、`commands.log` 及上述 `r2-*` 原始文件；未修改源码/测试/设计/任务/Plan/既有证据；未做 Git 写；未删除/移动/归档。
- 本报告不代替 Lead 现场决定、人的签收与后续授权；FAIL/HIGH 已按要求停止，不修复代码。