# identity-auth-S4.19 独立 Code Review 报告（R02 / 复审）

**最终结论：不通过（FAIL / CHANGES REQUIRED）**

- 授权来源：用户 2026-10-31 项目级授权 + `ADR-0028`（`.add/project/knowledge/decisions/framework.md:815-856`）。
- 代理主体：`add-reviewer` 角色 / AI 代理 Reviewer（Host 未返回独立 Agent ID，记为 `unknown / 本会话`）。
- 代理范围：identity-auth-S4.19 R02 的只读代码复审、定向 build/vet/gofmt/test 执行与证据报告；不含实施、不含 Git 写、不含签收生效。
- 声明：**AI 代理评审，非真人签字**。本次结论不代替 Lead 的接受、风险承担或后续授权。

---

## 0. 复审摘要

- `S419-CR-F002`（无配置 `planRecoveryStartup present:false` 未真实覆盖）：**CLOSED**。测试已真实调用 `planRecoveryStartup(lookup, nil)`，断言 `present=false/action=""/verify=false/maintenance=false`，`present=false` 时 handle 保持 nil，并在无配置 dispatcher 上覆盖 `/livez` 200、`/readyz` 503、业务跳过 guard、404、405 + `Allow`。
- `S419-CR-F001`（完整软件恢复后同 Session 200 未真实执行）：**仍 OPEN（MEDIUM）**。作者改用“真实 runtime+真实 PG 全链（`TestRecoveryRuntimePostgres`）”+“生产 dispatcher 同 Session 分层”两层证据替代单测试跨层链，并声明按 ADR-0028 收窄；但 ADR-0028 未给“降低任务卡 leaf 验收标准”的依据，Lead Gate R02 无正式收窄决定，任务卡 `tasks.md:2841` 原文仍未变；测试自身注释（`recovery_main_dispatcher_integration_test.go:167-171`）承认“no single test links both layers”。故收窄不成立，F001 保留。
- N1（closed 下 `/livez` 200 后紧接业务 503 而非先改回 closed）：**CLOSED**。
- 作者 R02 证据引用修正（`present=false` 行由误引 `07-unit.log` 改引 `13-norecovery-verbose.log`）：**已核实，无新缺陷**，见 §7。

因为 `S419-CR-F001` 仍 OPEN，本卡不满足关闭条件，最终结论为 **FAIL / CHANGES REQUIRED**。

## 1. 固定对象与修订

| 项目 | 值 |
|---|---|
| 仓库 | `D:\AID\VSCode\Codex\Han` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| staged | 0 |
| 评审角色 | `add-reviewer`（非作者独立复审，未参与实施） |
| 评审时间 | 2026-10-31（Asia/Shanghai） |
| 唯一写入目录 | `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r02/` |

| 评审对象 | SHA256（本次实算） | 字节 | mtime | 本轮状态 |
|---|---|---:|---|---|
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `66D2AB7A582B78018146CF5E976FE77FD683B4C63FC0FC72EC470D0DC887DC46` | 20300 | 2026-10-31T14:20:36.8793948+08:00 | 本轮唯一修改的代码文件 |
| `cmd/han-server/main.go` | `B012EAE8814BD47B16797A277AAB2E0CDA7C1885C839A8EE9D5435D63E3F03B8` | 171424 | 2026-10-31T10:27:35.8807765+08:00 | 未修改，与 R01 一致 |

- 两个 SHA256 在评审开始（`01-objects.log`）与报告写入前（`15-final-objects.log`）两次复算一致，与委派输入完全一致，未触发“对象变化即停止”。
- `cmd/`、`internal/`、`test/` 整体未纳入当前 HEAD 的 Git 索引，无 commit diff 基线；写集归因依赖 SHA + mtime + 全目录清单交叉核对（§8）。

## 2. 检查范围

### Round 1：复审针对原 finding 的修复修订

- 逐条对照 R01 报告 `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r01/report.md` 的 `S419-CR-F001`、`S419-CR-F002` 与 N1–N7，核对作者 R02 修改的唯一代码文件 `recovery_main_dispatcher_integration_test.go` 是否真正关闭原问题。
- 全文阅读 554 行测试文件，重点核对：`testRecoveryMainLiveReadyProbes`（`:110-155`）、`testRecoveryMainOpenAfterCompleteSoftwareRecoverySameSession`（`:157-261`）、`testRecoveryMainNoRecoveryConfig`（`:302-338`）、fake runtime 与 `publishPublicReady`（`:497-515`）、`newS419ProductionDispatcher`（`:340-404`）。
- 对照作者 R02 证据包 `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-author-r02/`（`report.md`、`01`–`13` 日志/`.exit`、`10-writeset-inventory.txt`、`12-gate-inputs.txt`）。

### Round 2：沿真实消费者与上游依据的有界深度审查

- 上游验收依据：任务卡 `.add/project/knowledge/tasks/tasks.md`（S4.19 卡 `:2802` 起，`LiveReadyProbes` `:2840`、`OpenAfterCompleteSoftwareRecoverySameSession` `:2841`、R01 FAIL/R02 停止记录 `:2804`），ADR-0028 文本（`framework.md:815-856`）。
- 生产消费链：`main.go` 启动 wiring（`:538-544`、`:606-618`）→ `newProductionBusinessHandler`（`:2385-2425`）→ exact router / Trace / guard（`:3515-3662`）；probe 分支 `:3632-3640`、guard 晚绑定 `:3641-3652`、Trace 最外 `:3655-3662`、guard 构造失败 fail-closed `:3665-3680`。
- 只读依赖：`internal/platform/recovery_guard.go`（`Protect`/`Release`/`Liveness`/`Readiness` 语义）、`cmd/han-server/recovery_runtime.go`（单一 runtime 构造与装配）、`cmd/han-server/recovery_config.go`（`recoveryRootConfigurationPresent`/`planRecoveryStartup`，`:88-123`）、`cmd/han-server/recovery_runtime_integration_test.go`（`TestRecoveryRuntimeProductionChain` `:165`，硬绑 `127.0.0.1:53519` `:38-39`、`:62`）、`test/integration/recovery/runtime_admission_test.go`（`TestRecoveryRuntimePostgres` `:11`）、`test/integration/recovery/runtime_fixtures_test.go`（`HAN_TEST_POSTGRES_DSN` 必需，缺失即 `t.Fatal`，`:104-106`）。
- 测试真实性：是否有 `t.Skip`、`http.ServeMux` fixture 代证、只断言接口名；实际执行命令并以原始日志支撑（§6、§7）。

### 未执行的范围

- 未启动真实 HTTP 服务、部署、Restore、SMTP、任何 E2E 演练；未连接真实 provider/CAS/ACL/POSIX。
- `TestRecoveryRuntimeProductionChain`（需 `127.0.0.1:53519`）因端口不可达 **NOT RUN**，不作为证据（§8）。
- 未对被审对象做任何修改、格式化或 Git 写操作。

## 3. 逐 leaf 结论

| 必要风险 leaf | 任务卡行 | 结论 | 主要证据 |
|---|---|---|---|
| `ExactRouteBeforeGuard` | `:2839` | **PASS（复跑一致）** | 测试 `:34-69`；实现 `main.go:3515-3662` |
| `ClosedUnknownTwoAdminGets503` | （同段） | **PASS（带 N2 备注）** | 测试 `:71-108`；`recovery_guard.go:27-47` |
| `LiveReadyProbes` | `:2840` | **PASS** | 测试 `:110-155`；N1 已闭合；无配置子案例由 F002 关闭 |
| `OpenAfterCompleteSoftwareRecoverySameSession` | `:2841` | **不通过（F001 OPEN）** | 测试 `:157-261` 仅经 `publishPublicReady`→`setState(public_ready)` |
| `SingleRuntimeHandle` | （同段） | **PASS（静态+间接，N3）** | 测试 `:263-300`；`main.go:538-544/606-618`；`recovery_runtime.go:528-597` |
| `NoRecoveryConfig`（`LiveReadyProbes` 无配置子案例） | `:2840` | **PASS（F002 关闭）** | 测试 `:302-338` 真实 `planRecoveryStartup` + `present=false` |

### 3.1 ExactRouteBeforeGuard（PASS，复跑一致）

- 实现顺序不变：exact path/method switch（`main.go:3515-3622`）→ 未知路径 `default` 404（`:3615-3621`）→ 已知路径错 method 405 + `Allow`（`:3623-3631`）→ probe（`:3632-3640`）→ 普通业务才进入 `guard.Protect`（`:3649-3652`）；`platform.TraceMiddleware` 最外（`:3662`）。
- 本轮修改未触及该 leaf 的实现与测试断言（测试 `:34-69` 未变），定向 integration 复跑全部通过（`07-integration.json`、`08-integration-race.json`）。

### 3.2 ClosedUnknownTwoAdminGets503（PASS，带 N2 备注）

- 测试对 `closed`、`unknown` 各请求两个 admin GET，断言 503、`platform.recovery_in_progress`、Trace 一致、无内部标记泄漏、下游零调用、事件仅 `trace`（`:71-108`）。R02 未改动此段，复跑通过。
- 非阻断：`unknown` 仍由 fake runtime 用 `ErrAdmissionClosed` 表示（`:497-507`），guard 对 Acquire 错误一律 503（`recovery_guard.go:33-39`），行为等价但未真实区分 unknown 分类。见 N2。

### 3.3 LiveReadyProbes（PASS；N1 已闭合）

- **N1 已修正**：closed 下 `/livez` 200（`:114`）后**紧接**同一 runtime 的业务 GET 仍 503（`:118`），并断言 `acquire=1`（`:119-121`）；不再“先改回 closed 再测”。因果链现为“liveness 200 未留下业务授权”。
- 覆盖：unknown/maintenance/verify 下 `/readyz` 503（`:122-125`）；`POST /livez`、`POST /readyz` 405 + `Allow: GET`（`:126-127`）；`/livezz`、`/livez/`、`/readyz/` 404（`:128-134`）；public_ready 下 `/readyz` 200 `ready` 后业务 200、acquire/release=2/1、readyz 确实调用共享 runtime（verifies>0）（`:136-144`）；最终 closed 业务 503 错误码校验（`:146-154`）。
- 无配置子案例由 `NoRecoveryConfig`（§3.6）覆盖并闭合。

### 3.4 OpenAfterCompleteSoftwareRecoverySameSession（不通过，F001 OPEN）

- 测试仍只有：closed 下同一 `*http.Request` 503（`:162`）→ `runtime.publishPublicReady(t)`（`:172`）→ 同一 Session 200（`:173`）。
- `publishPublicReady`（`:509-515`）只做“pre-open 计数校验（acquire=1、release=0）后 `setState("public_ready")`”；`setState`（`:497-507`）直接改写 `observation` 并把 `acquireErr` 置 nil。**没有**执行 `Begin → … → succeeded → 撤维护/真 join → 最终 proof → public open` 中任何一步真实状态机转换。
- 测试自身注释（`:167-171`）明确写：“TestRecoveryRuntimePostgres runs the real software recovery chain … no single test links both layers (see ADR-0028 narrowing)”。即作者自己也承认单测试跨层链未实现。
- 同一 Session 200、Trace、repository `lists=1`、Session resolver 调用、acquire/release=2/1、cookie 保持，以及 `AdmissionIsNotSticky`、`FailureNegativesNotMasked` 负控均成立；但这些只证明“runtime 已处于 public_ready 时 dispatcher 放行”，不能证明“完整软件恢复完成后同一 Session 放行”。见 `S419-CR-F001-R02`。

### 3.5 SingleRuntimeHandle（PASS，带证据限制）

- 生产装配只有一个 runtime 值：`recovery_runtime.go:528-547` 构造一次，同值交给 bridge/identity/review/composition，最后写入 `assembly.runtime`（`:596`）；`main.go:606-618` 仅在 `plan.present` 时把 `assembly.runtime` 写入晚绑定闭包，`:538-544` 业务 handler 读取闭包，属晚绑定而非第二 runtime。
- 测试 `:263-300` 证明晚绑定后同一 fake handle 被 guard 使用、lease 计数正确、panic 后释放（acquire/release=1/1）。
- 未发现第二 runtime/router/pool/authority/fallback。证据限制见 N3：fake provider 间接证明，`assembly.runtime` 与 `composition.runtime` 同一性由 `recovery_runtime_test.go:719-769` 与静态路径补强。

### 3.6 NoRecoveryConfig（PASS；F002 关闭）

- `testRecoveryMainNoRecoveryConfig`（`:302-338`）：
  1. `validCompositionEnvironment()` 构造不含 recovery data key 的环境（与 `main_test.go` 既有无配置用例同源，`main_test.go:8134`）。
  2. `:305` 真实调用 `planRecoveryStartup(lookup, nil)`，`:306-308` 断言 `err==nil`。
  3. `:309-311` 断言 `!present && !verify && !maintenance && action=="" && !recoveryRootConfigurationPresent(lookup)`。
  4. `:312-316` 仅当 `present` 为真才创建 runtime；实际 `present=false`，handle 保持 nil，dispatcher 通过 `func() recovery.Runtime { return handle }` 拿到 nil。
  5. `:318-319` `/livez` 200 `alive`、`/readyz` 503 `unavailable`；`:321-324` 业务 `GET /api/v1/admin/articles` 200 且不含 `platform.recovery_in_progress`（无 runtime 跳过 guard）；`:326-330` 未知路径 404；`:332-336` `POST` 405 + `Allow: GET`。
  6. `:337` 打印 `planRecoveryStartup present=false action="" verify=false maintenance=false handle_nil=true`。
- 独立复跑 `09-norecovery-verbose.log`（exit 0）输出该行，支撑 F002 关闭。对照 `recovery_config.go:110-122`：`getenv==nil` 返回 `ErrAdmissionBinding`，无配置返回零值 plan，与测试一致。
- 该 leaf 现同时满足 R01 要求的“真实调用 `planRecoveryStartup` / 断言 `present=false` / 无配置 dispatcher 行为”，**F002 CLOSED**。

## 4. Findings

### [CLOSED] S419-CR-F002 — NoRecoveryConfig 未覆盖 planRecoveryStartup 的 present:false

- **Finding ID**：`S419-CR-F002`（本轮复审编号 `S419-CR-F002-R02`）
- **位置**：`cmd/han-server/recovery_main_dispatcher_integration_test.go:302-338`
- **原现状**：直接注入 nil provider，未调用 `planRecoveryStartup`、未断言 `present:false`。
- **现现状**：真实调用 `planRecoveryStartup(lookup, nil)`（`:305`），断言 `present/action/verify/maintenance` 为空/false（`:309-311`），`present=false` 时 handle 保持 nil（`:312-316`），并在无配置 production dispatcher 上覆盖 `/livez` 200、`/readyz` 503、业务跳过 guard、404、405 + `Allow`。
- **证据**：`09-norecovery-verbose.log`、`13-norecovery-verbose.log`（作者，exit 0）均打印 `present=false ... handle_nil=true`。
- **结论**：**CLOSED**。原 finding 指出的“空环境误判 present:true / 仍写 handle 不会被发现”场景现由 `present=false` 与 `handle_nil=true` 直接拦截。
- **复审条件**：若后续改动 `planRecoveryStartup` 或该测试，需重新核对 `present=false` 与 handle 为 nil 的断言。

### [OPEN · MEDIUM] S419-CR-F001 — OpenAfterCompleteSoftwareRecoverySameSession 未执行完整软件恢复

- **Finding ID**：`S419-CR-F001`（本轮复审编号 `S419-CR-F001-R02`）
- **位置**：`cmd/han-server/recovery_main_dispatcher_integration_test.go:157-261`（关键 `:167-172`）；helper `:509-515`；fake 状态写入 `:497-507`。
- **现状**：测试在 closed 得到 503 后经 `publishPublicReady`（`:509-515`）直接 `setState("public_ready")`（`:514`），随即同 Session 200；fake 绕过真实状态机、join、最终对账与 open 许可。测试自身注释（`:167-171`）承认“no single test links both layers”。
- **收窄评估（逐项）**：
  - (a) **收窄是否有 ADR-0028 明确依据**：ADR-0028 授权“真实外部环境不可得时可用模拟并显式标注、AI 可代理评审/签收”，但**未授权降低任务卡 leaf 验收标准**，并明确禁止“把未执行/模拟写成 PASS”。任务卡 `tasks.md:2841` 的 `R/P` 措辞（完整链 `Begin→…→succeeded→撤维护/真 join→最终 proof→public open` 后同 Session 200）未被修改；Lead Gate R02 目录（`s419-lead-gate-r02/`）仅有身份/SHA/mtime/git-status 三份，无正式收窄决定或 leaf 措辞变更记录。→ **收窄无明确依据**。
  - (b) **分层证据是否真实执行且差距是否如实披露**：`TestRecoveryRuntimePostgres`（真实 runtime + 真实 PG `127.0.0.1:5433`）本轮独立复跑通过（`11-postgres-real-runtime.json`，pass=7/fail=0/skip=0），含 `real runtime.Enter consumed window … observation=public_ready` 与 `three worker join complete: workers=3 holders=0 … acquire_after=ErrAdmissionClosed`；production dispatcher 同 Session 分层测试亦通过。差距**已如实披露**（测试注释 + 作者报告 §3.2/§3.3）。→ 分层证据真实、披露到位。
  - (c) **是否仍有“把未验证写成 PASS”或测试名夸大**：**仍存在**。测试函数名 `testRecoveryMainOpenAfterCompleteSoftwareRecoverySameSession` 与子测试汇总名 `OpenAfterCompleteSoftwareRecoverySameSession` 主张“完整软件恢复后同 Session”，但该单测试从未执行完整恢复；若在 B13H Gate 之外单独跑该定向测试，会得到“完整链 PASS”的错误印象。
  - (d) **残余风险是否充分登记**：作者报告 §3.3 已登记“若真实恢复链在 join/最终对账/open 许可回归而分层测试未同跑，本卡定向测试不会失败”，并要求由 B13H Gate + `TestRecoveryRuntimePostgres` 独立执行承担。→ 登记较充分，但**不能替代 leaf 验收或正式收窄决定**。
- **触发场景**：完整恢复链在 join、最终对账、open 许可或状态持久化上回归，而 runtime 的 `Verify` 仍返回 `public_ready`（或测试直接伪造该 observation）时，本测试仍 PASS。
- **影响**：任务卡 `tasks.md:2841` 的 `R/P` leaf 未被证明；本卡关闭条件中的“完整软件恢复后同一 Session 200”不成立。失败负控（session/origin/csrf）虽覆盖，但不能替代正向完整链。
- **最小修复（二选一）**：
  1. **补链（推荐）**：在 `cmd/han-server` 同 package 内用真实 `recovery.Runtime` + 真实 PG 驱动完整恢复链（`Begin→…→succeeded→Exit/join→activate→public_ready`），再调用 production dispatcher，用同一 Session 断言 200，并保留现有 session/origin/csrf 负控；若因写集限制需新增第三写文件（测试夹具），须先由 Lead 扩展写集授权。
  2. **正式收窄（需 Lead/设计 owner 落权）**：修改任务卡 leaf `:2841` 措辞，重命名或明确标注该测试，登记残余风险，并把 `TestRecoveryRuntimePostgres` 明确为独立必跑的真实 PG 证据；收窄决定须写入 Lead Gate/任务卡，不能仅凭作者报告关闭。
- **验证**：普通与 `-race` 定向 integration 通过；补链方案的新日志需显示完整状态序列与 join/final proof 实际调用；收窄方案需有任务卡/Lead Gate 的书面收窄记录。
- **复审条件**：产生新 SHA 后复核原 finding、完整状态序列断言或正式收窄记录、负控是否仍成立。
- **结论**：**OPEN**。收窄不成立，最小修复见上。

### [CLOSED] N1 — LiveReadyProbes 断言因果偏弱

- **位置**：`cmd/han-server/recovery_main_dispatcher_integration_test.go:110-155`
- **原现状**：public_ready 检查后先改回 closed 再测业务 503，不能证明“此前 `/livez` 200 未留下授权”。
- **现现状**：closed 下 `/livez` 200（`:114`）后紧接业务 GET 503（`:118`），断言 `acquire=1`（`:119-121`）。
- **结论**：**CLOSED**。

## 5. 非阻断观察

- **N2（unknown 保真度）**：`s419RecoveryRuntime.setState` 对所有非 public_ready 状态统一设 `ErrAdmissionClosed`（`:497-507`），包括 `unknown`。`RecoveryGuard.Protect` 对任何 Acquire 错误一律 503（`recovery_guard.go:33-39`），行为等价；但真实 runtime 的 unknown 若在 Trace/错误码上有差异，本测试无法区分。非阻断。R02 未回归。
- **N3（SingleRuntimeHandle 测试间接）**：`SingleRuntimeHandle` 仍以 fake provider 间接证明 dispatcher 使用同一 handle；`assembly.runtime`/`composition.runtime` 同一性由 `recovery_runtime_test.go:719-769` 与静态路径补强。非阻断，无回归。
- **N4（旧 helper 行为变化）**：旧 `newLoginBusinessRouterWithTrace` 委托后非 probe 路由 404/405/Allow/Trace 不变；`/livez`、`/readyz` 按设计变为 200/503。既有结论维持，无回归。
- **N5（Lead Gate 证据）**：R01 N5 关于 Lead Gate 缺 01–04 build/vet/gofmt 原始输出的问题，本轮已由独立复跑 `02-gofmt`、`03-build`、`04-vet`、`05-vet-integration`、`09-norecovery-verbose` 替代，**视为足够**；建议 Lead Gate 后续补齐或注明由独立复跑替代。非阻断。
- **N6（probe 忽略 bool）**：`main.go` probe 分支忽略 `recoveryGuardFromRuntimeHandle` 第二返回值安全（零值 guard `/livez` 恒 200、`/readyz` 恒 503；业务分支 fail-closed）。无回归。
- **N7（写集可归属性）**：两对象未纳入 Git，无法用 diff 证明只改两文件；mtime + 全目录清单显示 `cmd/han-server` 内仅两对象位于本轮窗口（测试 `14:20:36`、`main.go` `10:27:35`），`internal/platform` 停留 10-29，无第三写文件。与任务卡写集一致，仍属弱证据。见 §8。
- **观察（历史遗留文件，非本轮产物）**：仓库根存在 `$null`、`(Join-Path`、`.red-a.patch`、`han-server.exe` 等早于本轮 mtime 的历史文件，非本轮写集，仅记录，不清理、不移动（受只读边界约束）。

## 6. 命令与退出码

所有命令在 `D:\AID\VSCode\Codex\Han` 根目录执行（PowerShell），原始日志位于本报告同目录。

| 序号 | 命令 | exit | 结果摘要 | 日志 |
|---|---|---:|---|---|
| 01 | `Get-FileHash` 两对象 + 字节/mtime + 分支/HEAD/staged/status | 0 | 两 SHA 与输入一致 | `01-objects.log` / `.exit` |
| 02 | `gofmt -l cmd/han-server/main.go cmd/han-server/recovery_main_dispatcher_integration_test.go` | 0 | 无输出 | `02-gofmt.log` / `.exit` |
| 03 | `go build ./...` | 0 | 无输出 | `03-build.log` / `.exit` |
| 04 | `go vet ./cmd/han-server` | 0 | 通过 | `04-vet.log` / `.exit` |
| 05 | `go vet -tags=integration ./cmd/han-server` | 0 | 通过 | `05-vet-integration.log` / `.exit` |
| 06 | `go test -tags=integration ./cmd/han-server -list '^TestRecoveryMainBusinessDispatcher$'` | 0 | 列出目标测试 | `06-list.log` / `.exit` |
| 07 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | pass=17 fail=0 skip=0 | `07-integration.json` / `.exit` |
| 08 | 同 07 + `-race` | 0 | pass=17 fail=0 skip=0，无 DATA RACE | `08-integration-race.json` / `.exit` |
| 09 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$/^NoRecoveryConfig$' -count=1 -v` | 0 | 打印 `present=false ... handle_nil=true` | `09-norecovery-verbose.log` / `.exit` |
| 10 | `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 42.368s` | `10-unit.log` / `.exit` |
| 11 | `HAN_TEST_POSTGRES_DSN=postgres://postgres:<redacted>@127.0.0.1:5433/postgres?sslmode=disable go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json` | 0 | pass=7 fail=0 skip=0（真实 PG 全链） | `11-postgres-real-runtime.json` / `.exit` |
| 12 | 写集清单（mtime/目录/`internal/platform`） | 0 | 仅两对象在本轮窗口 | `12-writeset-inventory.log` / `.exit` |
| 13 | 静态审计（`t.Skip`/`ServeMux`/凭据/`NewRuntime`/`plan` 源） | 0 | 无 `t.Skip`、无测试内 `ServeMux`；单一生产 `recovery.NewRuntime` | `13-static-audit.log` / `.exit` |
| 14 | B13H 拓扑端口探测（5433/53519） | 0 | 5433=True；53519=False | `14-b13h-topology.log` / `.exit` |
| 15 | 最终 SHA/HEAD/staged/status 复算 | 0 | 对象未变 | `15-final-objects.log` / `.exit` |

测试真实性核对：`13-static-audit.log` 显示被审测试文件无 `t.Skip`、无 `http.ServeMux` fixture；`newS419ProductionDispatcher`（`:340-404`）真实调用 `prepareProductionLoginDependencies`、`productionBusinessComposition`、`newProductionBusinessHandler`（`:398`），不是 ServeMux fixture；断言均为实际状态码/计数/响应体，非只断言接口名。生产中唯一 `http.NewServeMux()` 位于 `main.go:3986`（metrics 服务，非本卡 business dispatcher）。

## 7. 作者 R02 证据引用修正核实

- 作者补充说明：`s419-author-r02/report.md` 做了一处引用修正，并新增 `13-norecovery-verbose.log` / `.exit`（exit 0）；`present=false` 行原误引 `07-unit.log`（非 verbose，不含该行），现改正为 `13-norecovery-verbose.log`。
- 本轮核实：
  - `s419-author-r02/13-norecovery-verbose.log`（422 字节）实际含 `recovery_main_dispatcher_integration_test.go:337: planRecoveryStartup present=false action="" verify=false maintenance=false handle_nil=true`，且 `--- PASS`、`ok han/cmd/han-server`；`.exit` 为 `0`。
  - `s419-author-r02/07-unit.log`（68 字节）为非 verbose 全量单元输出（`ok han/cmd/han-server 37.683s`），不含该 verbose 行。
  - 作者 `report.md` 第 38、44 行现正确引用 `13-norecovery-verbose.log`，并另以 `07-unit.log` 指代全量单元套件。
- **结论：修正属实且已生效，无新缺陷，不作为 finding。** 本评审的 F002 结论不依赖作者该行，另有独立复跑 `09-norecovery-verbose.log` 支撑。

## 8. 写集与安全审计

- **SHA 冻结**：评审开始与报告写入前两次复算一致；未修改被审对象。
- **写集**：本轮唯一修改的代码文件为 `cmd/han-server/recovery_main_dispatcher_integration_test.go`（SHA `66D2…DC46`，14:20:36）；`main.go` 未改（SHA `B012…03B8`，10:27:35，与 R01 一致）；`cmd/` 内其余文件与 `internal/platform` mtime 均早于本轮窗口；无第三写文件。
- **越界/删除/Git**：未写 `cmd/`、`internal/`、`test/`、`.add/`、`.codex/`；未 Git add/commit/push/切分支/回滚；未删除/移动/归档任何文件；`staged=0`，HEAD 未变（`15-final-objects.log`）。
- **本评审写入**：仅向本报告目录写 `report.md` 与 01–15 原始日志。
- **敏感信息**：测试使用 `admin-session-token` 等合成值；断言覆盖 run/account/ref/secret 不泄漏；`13-static-audit.log` “credential-like literals” 段无命中；未发现真实凭据/API Key。
- **硬编码**：`/livez`、`/readyz` 为设计固定 exact route 字面量（合同常量）；未发现可配置值被硬编码。
- **第二 authority/runtime/fallback**：未发现第二 runtime、router、pool、authority 或 production fallback；`13-static-audit.log` 显示生产 `recovery.NewRuntime` 仅 `recovery_runtime.go:528` 一处（其余均为 `_test.go`）。
- **Git 状态**：`git status --porcelain` 282 行（281 基线 + 本评审目录），与 R01 基线一致；`cmd/`、`internal/` 整体未跟踪。

## 9. 未验证边界与剩余风险

- `TestRecoveryRuntimeProductionChain`（`cmd/han-server/recovery_runtime_integration_test.go:165`，硬绑 `127.0.0.1:53519`，`:38-39`/`:62`）因 `53519 tcpTestSucceeded=False`（`14-b13h-topology.log`）**NOT RUN**，不得作为 S4.19 证据。
- `TestRecoveryRuntimePostgres`（`test/integration/recovery/runtime_admission_test.go:11`）本轮以 5433 真实 PG 通过，属真实 runtime+PG 全链，但**不接** `cmd/han-server` production dispatcher、不做同 Session 200；它只是 F001 分层证据的一半，不能写成“S4.19 完整链 PASS”。
- 未运行真实 HTTP 服务、部署、Restore、SMTP、任何 E2E 演练；未验证真实 POSIX/FileJournal、CAS、ACL、备份、keyring/HMAC。
- 未证明 `canonical_recovery_replay_same_result`。
- 两对象未纳入 Git，变更归因依赖 SHA + mtime + 清单，属弱证据。
- 本次 build/vet/gofmt/定向测试通过只证明编译、静态检查与定向测试通过，不等于部署 E2E、人工签收或 PlanningReview 通过。

## 10. 待 Lead 决定

1. **F001 处置**：建议按 §4 最小修复二选一——补真实完整链，或由 Lead/设计 owner 对任务卡 `:2841` 正式收窄并落书面记录。在此之前本卡不得关闭。
2. **是否要求把 `TestRecoveryRuntimePostgres` 纳入本卡/批次必跑集**：若采用收窄方案，应明确其为独立必跑的真实 PG 证据。
3. **N2/N3 是否随批和修正**：非阻断，成本低，建议随 F001 一并处理。

## 11. 最终结论

**不通过（FAIL / CHANGES REQUIRED）。**

- `S419-CR-F002`：**CLOSED**（真实 `planRecoveryStartup`、`present=false`、handle 为 nil、无配置 dispatcher 行为全覆盖）。
- N1：**CLOSED**（closed 下 `/livez` 200 后紧接业务 503）。
- `S419-CR-F001`：**仍 OPEN（MEDIUM）**。作者分层证据真实、差距已披露、残余风险已登记，但 ADR-0028 未授权降低任务卡 leaf 验收，Lead Gate R02 无正式收窄决定，测试仍以 `setState(public_ready)` 冒充“完整软件恢复”，测试名存在夸大风险。收窄不成立。
- 生产 dispatcher 接线方向正确（exact route 先行、Trace 最外、guard 晚绑定 fail-closed、单一 runtime、lease 正确释放），本轮无新增生产缺陷回归。

在 `S419-CR-F001` 按上列最小修复关闭、或 Lead/设计 owner 对任务卡 leaf 正式收窄并落书面记录之前，identity-auth-S4.19 不满足关闭条件，不能声称通过。以上为 **AI 代理评审，非真人签字**，结论不代替 Lead 的接受与后续授权。

---

**2026-10-31 脱敏说明（Lead，按用户确认执行）**：按用户 2026-10-31 确认，本文件中原引用的本机 loopback 合成 DSN 口令已替换为 `<redacted>`（共 1 处）；其余内容与结论未变。脱敏前 SHA256 为 `E3540E9504D05C3791C936E841EA1AA7CC79A86371F861268CC20C3A529841A6`，脱敏后哈希见 tasks.md 同日记录。
