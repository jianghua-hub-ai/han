# Case18 覆盖对账：非作者独立评审报告

> **AI 代理评审，非真人签字。** 本报告是只读取证与独立判断，不代替 Lead 的任务决定，不代替人的业务接收或部署授权，也不宣称 S8.2 / Case18 / S1.5 已 PASS 或已关闭。

## 0. 评审对象与边界

- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 现役代码基线：
  - `cmd/han-server/main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B
  - `cmd/han-server/recovery_runtime.go` `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` / 37431 B
  - `cmd/han-server/recovery_isolation_matrix_integration_test.go` `DA02943B9848C50E1960ECCDBF210C3E69D3AA8A64034F9A6321740E519E9115` / 8498 B
- 判定标准：
  - `.add/project/knowledge/design-workspace/article-management/test-plan.md` `24786C1A584ACF5277DD2DBD43E053CB7679F2867CD42937EDC7F722FD435036` / 324213 B，A-ADM2-18 在 `:348`，Case18 五分项表在 `:352-360`，整体阻塞在 `:362`。
  - `.add/project/knowledge/tasks/tasks.md` `87CB31CE7C379DE06AE8F128EBC11D6718E371ED5F8AB79DE1D2538768D071A4` / 813571 B，`article-management-S8.2` 在 `:1867-1873`，`identity-auth-S4.20` 在 `:2881` 与 `:2957-2958`，`identity-auth-S4.21` 在 `:2965-3032`。
- 授权：只读评审，仅允许写入本目录三文件；未做 Git 写、未改既有文件、未建/删数据库或 schema、未重跑会保留 schema 的集成测试。

## 1. 基线与证据哈希核对（实算）

| 文件 | 实算 SHA256 | 字节 | 与记录关系 |
|---|---|---:|---|
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | 等于给定现役基线 |
| `cmd/han-server/recovery_runtime.go` | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` | 37431 | 等于给定现役基线 |
| `cmd/han-server/recovery_isolation_matrix_integration_test.go` | `DA02943B9848C50E1960ECCDBF210C3E69D3AA8A64034F9A6321740E519E9115` | 8498 | 等于给定现役基线 |
| `cmd/han-server/main_test.go` | `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` | 400312 | 当前实际；**不等于** S8.2 root 证据绑定的 `B53DD34D…` |
| `cmd/han-server/recovery_production_chain_integration_test.go` | `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` | 167339 | 与 S4.20 记录一致 |
| `cmd/han-server/recovery_enter_install_integration_test.go` | `6BFA7D761432C960740F4E1FE91566317B5707B8481CE0D9FF1CF46DD4106C3A` | 20564 | 与 S4.21 关闭记录一致 |
| `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` | `2A0013FD4A92325DAD3BA35CF3444DBFA10B5A0884F17E335E26F591B8768019` | 3155 | 等于给定证据入口 |
| `.spec/wip/article-management/verification-s8.2.md` | `42AE7A0A7488EFFC28B7EF9060E94519826D9232FC7EFCEF46FAA0910B44A412` | 10530 | 等于 test-plan 引用 |
| `.spec/wip/article-management/review-code-s8.2-joint-r02.md` | `F1EE8BF2605C3755835E2A1C0F4DFDA354697BD4EEBE52FE8B8A80E0681B7808` | 20142 | 等于 test-plan 引用 |

S4.20 d7 证据（位于 `evidence-b13h-r01/s420-implementation-r01/`，非目录根）：

| 文件 | 实算 SHA256 | 字节 | 记录核对 |
|---|---|---:|---|
| `d7-chain-complete-r02.json` | `225F78BDF3BC67A453B83018E1FC1D2A8E1A0548EC44703C42D7AAC933343761` | 27057 | 一致 |
| `d7-chain-complete-race-r02.json` | `72002FB661EF69588684F2E7B4CB239F4D8F96672C3F927F8AB1C248E43BCC56` | 27055 | 实物存在；`tasks.md:2957` 记录为 `72002FB661EF695886F2E7B4CB239F4D8F96672C3F927F8AB1C248E43BCC56`（62 位，缺 `84`） |
| `d7-five-leaves-r02.log` | `2236B5068B5F538F29F2DFD0F7D92B1E165F1656604217AF958758C5CCD1DA51` | 12482 | 一致 |
| `d7-five-leaves-r02-race.log` | `36D49464FF311FB813F76725DF4098A3AE09FA7E03B1C9F24A2E5B3BFC037FD9` | 12434 | 一致 |
| `d7-main-dispatcher-r02.json` | `A673F00E92532CFCC1FDBEADA5FF17EC468F2DF0E0734DC139568514503A79F7` | 9596 | 一致 |

S1.5 producer-real：

| 文件 | 声明值 | 实算值 | 结论 |
|---|---|---|---|
| `evidence-s1.5/producer-real-r01/index.json` | `F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B` / 19567 | 相同 | index 自身一致 |
| 其 `artifacts[]` 声明的 `report.md` | `A9A4ECBB9DEF6C1862ECE1D151F085F179DE43A6A489FD94411B017978150BFF` / 8023 | `1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749` / 8267 | **不一致**（见 F005） |

## 2. 逐项判定

### 项 1：Case18「真实factory/共享实例」— PARTIAL

**结论：源码结构支持唯一共享实例，但 S8.2 的 root 执行证据绑定旧 revision，当前 revision 没有可核执行证据。**

源码事实（当前 revision）：

- `cmd/han-server/main.go:566-572`：唯一 `article.PostgresReplayEvidenceStore`，只由 `pool` 构造一次。
- `cmd/han-server/main.go:573-602`：同一 `pool`、同一 `articleEvidence`、同一 `runtime`、同一 `recoveryHandle` 注入 `composeBusiness`，无第二套 composition。
- `cmd/han-server/main.go:2476`：唯一 `session.NewStore(...)`。
- `cmd/han-server/main.go:3111-3126`：`NewPostgresArticleQueryRepository(pool)`、`NewArticleIdentityReader(pool)`、`NewArticleQueryService(...)` 各构造一次。
- `cmd/han-server/main.go:3143-3160`：`authorHandler` 与 `adminHandler` 共用同一 `service`，并通过 `observeArticleSharedDependency` 记录共享关系。

测试事实：

- `cmd/han-server/main_test.go:1199-1201`：`TestProductionAdminArticleQuery` / `ProductionSharedChain`。
- `cmd/han-server/main_test.go:1511-1542`：`adminRootActualFactory` 断言 admin/member/author 同 `ArticleQueryService`、同 pool/repository/reader、同 config、同 `SessionStore`。
- `cmd/han-server/main_test.go:8406-8447`：helper 调用真实 `newProductionBusinessHandler`，但传入 `new(pgxpool.Pool)`，是装配接缝而非真实 PostgreSQL。

缺口：`evidence-s8.2/final-r01/review-root-r02/stage-readset-binding.json` 的 `currentRootSHA256` 仍为 `main.go=6B45B5509A2D01C9F5DF0393CA00CAD5C8430353B5F849BF6E71BA3A47C52AC4`、`main_test.go=B53DD34D57AD6A2797E9FC080FD90034FC52CB26675686CB075D3153F193A9BC`，且 `PExecuted:false`、`UCReexecuted:false`、`Case18Overall:"BLOCKED upstream"`、`TaskClosed:false`。当前实际为 `6D5373…` / `372065…`。

### 项 2：Case18「Trace/exactrouter」— PARTIAL

**结论：当前源码的 exact router 行为可静态核对，但缺少绑定当前 revision 的真实执行证据。**

源码事实：

- `cmd/han-server/main.go:3626-3627`：`/api/v1/admin/articles` 仅允许 `GET`，`Allow=GET`。
- `cmd/han-server/main.go:3644-3651`：`/api/v1/admin/articles/{id}` 允许 `GET, DELETE`，分别路由到 admin query 与 delete handler。
- `cmd/han-server/main.go:3694-3698`：exact 路由选定 handler 后，再由 recovery guard 包裹，Trace 在外层。
- `cmd/han-server/main.go:3758-3764`：`exactArticleDeletePath` 只接受前缀后非空且不含 `/` 的单段 ID；空 ID、额外段、`/articlesx` 不匹配。

测试事实：

- `cmd/han-server/main_test.go:1206-1224`：使用真实 production dispatcher 注册的 exact route，不是 spy router。
- `cmd/han-server/main_test.go:1637-1660`：`adminRootRouteMatrix` 覆盖 list/detail 的 method、`Allow`、404 与旧路由共存。

缺口：当前 revision 没有新的 root 执行证据；可引用的是旧 revision 的 93/76 leaf 结果或仅哈希/日志。静态正确不等于当前版本已执行。

### 项 3：Case18「Session/AccessGeneral」— PARTIAL

**结论：可信 Principal、GET 无 CSRF、账号级拒绝零文章下游有装配/接缝证据；真实 `SessionStore` SQL 与会话撤销在本 revision 未证。**

证据：

- `cmd/han-server/main.go:3244-3250`：7 个受保护查询/治理路由全部 `authorizer.Protect(session.AccessGeneral, ...)`。
- `cmd/han-server/main_test.go:1398-1439` `adminRootPrincipalCase`：
  - 传入可信 `session.Principal`；
  - `:1434` 断言 GET 时 `RequireCSRF=false`、`CSRFToken` 为空、`Token` 正确；
  - 对 member/suspended/deleted/onboarding 等拒绝路径断言 article list/get 与 reader 零调用；
  - 普通管理员请求可进入真实 query handler。
- `cmd/han-server/main_test.go:1825` 起 `articleAuthorRootResolver` 为测试 resolver spy。

边界：

- resolver spy 只证明“拒绝/传递”接缝，不证明真实 `SessionStore` SQL、会话撤销、事务与数据库行为。
- 账号级 403 不能当作全局恢复 503；`test-plan.md:358` 明确要求两者分开。
- 缺当前 revision 的真实 SessionStore SQL/会话失效证据。

### 项 4：Case18「全局恢复请求隔离（blocked-upstream）」— PARTIAL

**结论：`closed`/`unknown` 两态的两个 admin GET 503 与零文章下游在 production dispatcher 测试中成立；`failed`/`expired`/`undetermined` 未覆盖，且矩阵使用 fake runtime，当前 revision 无执行证据。**

源码/测试事实：

- `cmd/han-server/recovery_isolation_matrix_integration_test.go:22` `TestRecoveryIsolationMatrix`；`:24-25` 只循环 `closed`、`unknown`；`:30` 使用 `newS419RecoveryRuntime(state)`（fake）；`:31` 使用 production dispatcher helper。
- 同文件 `:52-67` 断言 503、`platform.CodePlatformRecoveryInProgress`、Trace/header 一致；`:71-86` 断言 authorization/origin/session/repository/reader 零调用、`acquires=路由数`、`releases=0`。
- 同文件 `:173` 与 `:178` 明确有 `admin_article_list`、`admin_article_detail` 两个叶子；共 25 条受保护业务路由。
- `cmd/han-server/recovery_main_dispatcher_integration_test.go:72-94` `testRecoveryMainClosedUnknownTwoAdminGets503`：明确对 `/api/v1/admin/articles` 与 `/api/v1/admin/articles/{id}` 断言 503 + `platform.recovery_in_progress` + 零 repository/reader/session 调用。
- `internal/platform/recovery_guard.go:30,43,82`：acquire 失败/缺 lease 即 `writeRecoveryHTTPClosed`，写 `CodePlatformRecoveryInProgress`，不调用下游。

缺口：

- 只覆盖 `closed`/`unknown`；`failed`/`expired`/`undetermined` 未在矩阵中出现。
- isolation matrix 使用 fake `newS419RecoveryRuntime`，非真实 recovery runtime。
- 无当前 revision 的实际执行证据，也无真实 PostgreSQL/双系统隔离证据。

### 项 5：Case18「完整恢复后生产开流（blocked-upstream）」— NOT COVERED

**结论：S4.20 只证明旧 runtime 上的 resume 链；S4.21 只交付 seam，production Enter/Install 仍 closed。当前 revision 的“完整恢复 + S16 对账/二次核验 + MarkConsumed + 真实开流”不可核。**

事实：

- `cmd/han-server/recovery_production_chain_integration_test.go:2747-2756`：`TestRecoveryProductionChainComplete` 八叶，含 `ResumeCoordinatorRecoveryCompleteAndOpen`、`SecondReadStatusAndMarkConsumed`、`CanonicalReplaySameResult`、`ClosedUnknownKeepIsolation`、`SingleRuntimeStorePool`、`SimulatedJournalKeyringBoundary`、`NoSecondRuntimeRouterPoolAuthorityFallback`。
- 同文件 `:3192-3206`：MarkConsumed 恰一次，caller 为 `coordinator.go`。
- 同文件 `:3759-3773`：明确注释 `s420NewCase18Transport` 是 **TEST adapter**，不是 production dispatcher；该叶覆盖 `/api/v1/articles/{id}` 与 `/api/v1/articles`，不是两个 admin GET。
- S4.20 d7 证据绑定 `recovery_runtime.go` `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35832 B（`tasks.md:2957`），当前为 `1C679377…` / 37431 B。
- `cmd/han-server/recovery_runtime.go:456-492`：11 个必填 enter/install 端口；`:527-535`：`enter`/`install` 缺端口直接返回 `ErrAdmissionClosed`。
- `cmd/han-server/recovery_enter_install_integration_test.go:269-373`：`production shape stays closed`（`:273`）、注入 seam 只到 sentinel failure、one runtime/store、`default secret source stays closed`（`:363`）。
- `tasks.md:3032` S4.21 关闭记录明写：production `Enter`/`Install` 仍 `ErrAdmissionClosed`；真实 PostgreSQL 下 S4.20 链本轮未重跑；未做 E2E/真实备份/部署；不解除 blocked-upstream。

结论：S4.20 的证据链是 resume 主线（且绑定旧 runtime），S4.21 是装配 seam，二者都不等于当前 revision 的“完整恢复后生产开流”。因此该项为 NOT COVERED，而不是 PARTIAL。

## 3. 证据分层

**组件/装配层（in-process `httptest` / assembly seam）**

- `cmd/han-server/main_test.go:1199-1660`（factory/shared/exact route/method matrix）
- `cmd/han-server/recovery_main_dispatcher_integration_test.go:72-250`（closed/unknown 两个 admin GET 503；public_ready 同 session 放行）
- `cmd/han-server/recovery_isolation_matrix_integration_test.go:22-130`（25 路由 × closed/unknown）
- `cmd/han-server/recovery_enter_install_integration_test.go:269-373`（closed 端口清单与注入 seam）

**真实 PostgreSQL 证据**

- S4.20 `d7-*.json`：真实 PostgreSQL 环境上的 resume 链，但绑定旧 `recovery_runtime.go 900F31F5…/35832`。
- S1.5 `producer-real-r01`：真实 PostgreSQL 18.6 双轮 58 pass / 0 fail / 0 skip，但仅 article `Publish/ReadCurrent`，不是 Case18 两个 admin GET 链；且索引 `report.md` 哈希/字节不一致（F005）。
- 当前 `1C679377…/37431` 上没有真实 PG S4.20 重跑。

**真实部署 E2E**

- 无。`evidence-s1.5/sim-closeout-r01/index.json:22` 明示“Case18 真实双系统全局恢复隔离：NOT RUN / unknown（需要两套真实系统）”。
- 现有均为 in-process、test adapter、模拟 keyring/backup。

## 4. Findings

### C18-COV-F001（HIGH）S8.2 root 证据绑定旧 revision，当前 revision 无执行证据

- 位置：`.spec/wip/article-management/evidence-s8.2/final-r01/review-root-r02/stage-readset-binding.json`
- 依据：`currentRootSHA256.main.go=6B45B5509A2D01C9F5DF0393CA00CAD5C8430353B5F849BF6E71BA3A47C52AC4`、`main_test.go=B53DD34D57AD6A2797E9FC080FD90034FC52CB26675686CB075D3153F193A9BC`；`PExecuted:false`、`UCReexecuted:false`、`Case18Overall:"BLOCKED upstream"`、`TaskClosed:false`。
- 影响：factory/共享实例、Trace/exactrouter 等项的“已验”只对旧 revision 成立；当前 `main.go 6D5373…/173317` 与 `main_test.go 372065…/400312` 没有对应的 root 执行结果。
- 修复建议：在绑定的当前 revision 上补 root 证据窗口，覆盖真实 production factory/shared instance、Trace、exact method/path/Allow、旧路由共存；无需重写实现。

### C18-COV-F002（HIGH）全局恢复隔离覆盖不全

- 位置：`cmd/han-server/recovery_isolation_matrix_integration_test.go:24-25,30`；`cmd/han-server/recovery_main_dispatcher_integration_test.go:72-94`
- 依据：只覆盖 `closed`/`unknown`；`newS419RecoveryRuntime` 为 fake；`failed`/`expired`/`undetermined` 未覆盖。
- 影响：`test-plan.md:359` 要求的“未完成、失败、失效或不可判定”未完整取证；当前 revision 无执行证据；无真实 runtime/PG/双系统。
- 修复建议：由上游恢复主线卡补齐全状态矩阵，使用真实 recovery runtime 与真实 PostgreSQL；两个 admin GET 必须真实 503 且 article service/repository/reader 零调用。

### C18-COV-F003（HIGH）完整恢复后生产开流证据绑定旧 runtime，且 production 仍 closed

- 位置：`tasks.md:2957`（S4.20 d7 绑定 `recovery_runtime.go 900F31F5…/35832`）；`tasks.md:3032`（S4.21 production Enter/Install 仍 `ErrAdmissionClosed`，真实 PG S4.20 未重跑）；`cmd/han-server/recovery_runtime.go:456-492,527-535`；`cmd/han-server/recovery_enter_install_integration_test.go:273,363`
- 依据：当前 runtime 为 `1C679377…/37431`，与 S4.20 证据绑定不一致；S4.21 明写未做 E2E/真实备份/部署。
- 影响：`test-plan.md:360` 的“完整恢复 + S16 对账/二次核验 + MarkConsumed + 真实开流”在当前 revision 不可核；失败/不可判定继续隔离也未在当前 runtime 证明。
- 修复建议：后继恢复生产实现卡在当前 runtime 上重跑完整链，并使用真实 PostgreSQL/Journal/keyring/backup 环境。

### C18-COV-F004（MEDIUM）Session/AccessGeneral 仅接缝证据

- 位置：`cmd/han-server/main_test.go:1398-1439,1825`；`main.go:3244-3250`
- 依据：可信 Principal、GET 无 CSRF、账号级拒绝零文章下游由 resolver spy 和 handler 装配证明；无真实 `SessionStore` SQL、会话撤销与数据库事务证据。
- 影响：`test-plan.md:358` 要求“实际 Session SQL 证据按相应身份层保留，不把账号 403 当全局 503”，当前 revision 缺该层。
- 修复建议：由 identity-auth 会话卡/真实 PostgreSQL 测试补齐 SessionStore SQL 与撤销路径；不得用 resolver spy 替代。

### C18-COV-F005（MEDIUM）S1.5 producer-real 索引与实物不一致

- 位置：`.spec/wip/article-management/evidence-s1.5/producer-real-r01/index.json` 的 `artifacts[]`
- 依据：声明 `report.md` 为 `A9A4ECBB9DEF6C1862ECE1D151F085F179DE43A6A489FD94411B017978150BFF` / 8023 B；实算为 `1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749` / 8267 B。
- 影响：producer 证据本身（真实 PG 18.6 双轮 58 pass）可用，但目录完整性受损，不能作为整体引用基线。
- 修复建议：由证据 owner 修正索引声明或补 superseded 指针；属文档纠正，不改变行为结论。

### C18-COV-F006（LOW）S4.20 race JSON 哈希记录少 2 位

- 位置：`tasks.md:2957`
- 依据：记录 `72002FB661EF695886F2E7B4CB239F4D8F96672C3F927F8AB1C248E43BCC56`（62 位）；实物 `72002FB661EF69588684F2E7B4CB239F4D8F96672C3F927F8AB1C248E43BCC56` / 27055 B（64 位，缺 `84`）。
- 影响：只是记录瑕疵，实物存在且字节数一致；不影响 S4.20 行为结论。
- 修复建议：更正记录文本。

### C18-COV-F007（LOW）无真实部署 E2E / 双系统

- 位置：`.spec/wip/article-management/evidence-s1.5/sim-closeout-r01/index.json:22`
- 依据：明示“Case18 真实双系统全局恢复隔离：NOT RUN / unknown（需要两套真实系统）”；现有为 in-process `httptest`、test adapter、模拟 keyring/backup。
- 影响：不能证明真实部署、进程级隔离、双系统恢复边界。
- 修复建议：由部署 E2E 或项目真实环境验证补齐。

## 5. S8.2 整卡缺口清单（是否具备进入关闭评估条件）

**S8.2 整卡不具备进入关闭评估的条件。** 精确缺口：

1. 当前 revision 的 root/装配执行证据：需绑定 `main.go 6D5373…/173317`、`main_test.go 372065…/400312`，覆盖真实 production factory/shared instance、Trace、exact method/path/Allow、旧路由共存。可由 S8.2 当前 root 验证窗口或既有 root 回归卡补齐，无需新写实现。
2. 当前 revision 的 Session/AccessGeneral 真实证据：需实际 `SessionStore` SQL、失效/撤销会话与拒绝路径零文章下游。属 identity-auth SessionStore/真实 PostgreSQL 层，resolver spy 不可替代。
3. 全局恢复隔离全状态证据：`closed`/`unknown` 之外还需 `failed`/`expired`/`undetermined`；两个 admin GET 必须真实 503 `platform.recovery_in_progress` 且 article service/repository/reader 零调用；需真实 recovery runtime、当前 root、真实 PostgreSQL/双系统。需上游恢复主线卡或后继恢复实现卡补齐。
4. 完整恢复后生产开流：需当前 runtime 上完成完整恢复 + S16 对账/二次核验 + MarkConsumed + canonical replay，再真实 `public_ready` 开流；失败/不可判定继续隔离。S4.20 旧 runtime 证据不可直接沿用；S4.21 production Enter/Install 仍 closed。需后继恢复生产实现卡 + 真实 PostgreSQL/Journal/keyring/backup 环境。
5. 真实部署 E2E/双系统：当前无；需真实部署环境/双系统恢复演练。
6. 证据目录修正：S1.5 producer index 修正 `report.md` 哈希/字节；S4.20 race hash 记录补 2 位。属证据整理，不改变行为结论。

## 6. 未验证边界

- 未运行任何 `go test`：集成测试会创建并保留 schema，超出只读授权，统一记 `NOT RUN`。
- 未连接或写入真实数据库；未创建/删除 schema；未重跑 S4.20 真实 PG 链。
- 未做真实部署 E2E、双系统恢复、真实备份恢复、真实 keyring/HSM。
- 未核对全部 291 行既有 worktree 改动与 30902 行 untracked 的逐项归属，只核对本评审相关文件。
- 证据绑定判断以实算 SHA256/字节为准；HEAD 不含 Go 源码（`git ls-files '*.go'` 为 0），不能仅靠 commit 绑定。

## 7. 复跑与限制

- 已执行 `go build ./...`（EXIT=0）与 `go vet ./cmd/han-server`（EXIT=0）。
- 未执行任何测试；`go test` 记 `NOT RUN`，原因同上。

---

AI 代理评审，非真人签字。