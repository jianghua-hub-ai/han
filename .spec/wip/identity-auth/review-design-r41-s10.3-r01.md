# identity-auth design r41 / S10.3 独立设计评审 R01

- 日期：2026-10-24
- Reviewer：Gibbs（非作者独立 Reviewer）
- objectType：design
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：**Design Review Gate: CHANGES REQUIRED**

本报告只评价 r41 对 S10.3 三消费者 Gate 的设计增量是否稳定、可实施并可交 Task-Planner 对账。它不评价代码已经实现，不关闭 S10.3，不启动 S10.4/S11.1，不关闭代码 `IDENTITY-S10.1-REV-R01-F002`，也不表示 production readiness。

## 1. 固定对象与输入

| 对象 | revision / 用途 | SHA256 | 开始检查 |
|---|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r41`，被审对象 | `6EBA81AD36D0E86A7FEEB65CF598E376D78F3D3298E18C2A865F3A0E3AA94FDA` | MATCH |
| r40 design 基线 | `identity-auth-design-ir01-r40` | `B8D2171DEF5DF3E0B47B521C9898E9B19F77EFC44388A0B461ED34B774416336` | 固定输入 |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r16`，本轮未改 | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` | MATCH |
| `.add/project/knowledge/design/identity-auth/boundary.md` | r13 | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` | MATCH |
| `.add/project/knowledge/design/architecture.md` | r27 | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` | MATCH |
| `.add/project/knowledge/decisions/README.md` | ADR 索引 | `B51C6265BB2493BE3AA2EE0C32425F29974B38A177018DC5D75F72A12437E14E` | MATCH |
| `.add/project/knowledge/decisions/identity.md` | accepted ADR-0023 | `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978` | MATCH |
| `.spec/wip/identity-auth/review-design-r40-s10.2-r01.md` | r40 Design PASS | `EB1AB24C659CB8281FD4FC2058470C88ACA7FACFC33D1665900C091CAE9A46FB` | MATCH |
| `.spec/wip/identity-auth/review-test-plan-r18-s10.2-r02.md` | Test-plan PASS | `3A241B10EFB64EFA7A09E452F20F22CB84051AAB7D3503E511C5298451AF794C` | MATCH |
| `.spec/wip/identity-auth/review-tasks-plan-s10.2-r03.md` | 历史增量 Planning PASS | `B0839C80C59B1761D33F362207D496F0E8E7D419D36CD0EA28FDAA4DC1E981E1` | MATCH |

适用日期按上级运行指令及用户更正固定为 2026-10-24；r41 的 2026-10-24 frontmatter/追溯日期正确，不构成 finding。

## 2. 两轮实际检查

### Round 1：固定设计对象直接检查

1. 核对 revision、draft 状态、r41 增量摘要、变更记录、状态边界和未实现声明。
2. 检查 `session.PasswordResetGate.CheckLockedUser(context.Context, pgx.Tx, string) error` 的包归属、调用前置、只读语义、nil/non-nil 决策、错误保留和 hold 写入边界。
3. 检查 Login、PasswordChange、Session resolve 的锁序、调用时点、rollback/commit barrier、Principal 物化时点及缺失 User 的登录抗枚举路径。
4. 检查 §9.4.1 九文件影响面、S10.3/S10.4 所有权分界、停止条件，以及 r40/F006/公共 REST、错误、权限、schema 和 ADR 不变声明。
5. 核对 stories r16 行为合同未被 r41 改写；其历史状态文字需由后续影响同步处理，但不是本轮日期 finding。

### Round 2：上游决定与真实消费者有界深审

1. 沿 accepted ADR-0023、ADR-0005、ADR-0014/0018、boundary r13 与 architecture r27 核对唯一 Gate owner、三消费者强制消费、旁路 reader 禁止、升级/混跑和 production composition Gate。
2. 检查现役：
   - `internal/session/store.go` / `_test.go`
   - `internal/identity/login.go` / `login_test.go` / `login_integration_test.go`
   - `internal/identity/service.go`
   - `internal/identity/bootstrap.go` / `_test.go`
   - `internal/identity/password_reset_store.go` / `_test.go`
   - `cmd/han-server/main.go` / `main_test.go`
3. 核对当前 production root 仍以 `session.NewStore(pool)` 构造 Session store，并在未提供 PasswordReset Gate 的情况下构造 Login 与 PasswordChange Service。
4. 核对范围外但会被严格构造合同直接影响的真实测试调用点，包括 `internal/identity/login_integration_test.go`、`internal/identity/administrator_create_test.go` 与 `cmd/han-server/main_test.go`。
5. 检查现役 Tasks/Plan 的 S10.3 七文件写集、Manifest 全量检查、S10.4 composition 所有权和串行关闭条件。

## 3. Findings

### `IDENTITY-DESIGN-R41-S10.3-R01-F001` — HIGH — S10.3/S10.4 过渡合同无法同时满足“缺 Gate 构造失败”、九文件独立关闭与无旁路 authority

**定位**

- `design.md:2810-2813`
- `design.md:3485-3501`
- 真实消费者：
  - `cmd/han-server/main.go:1955,1974-2005`
  - `internal/session/store.go:119-130,214-227`
  - `internal/identity/login_integration_test.go:368,467-485`
  - `internal/identity/administrator_create_test.go:1464`
  - `cmd/han-server/main_test.go:3528`

**违反依据**

1. ADR-0023 要求 Login、PasswordChange、Session resolve 都是强制 Gate 消费者，任何旁路 reader 都是安全违约（ADR-0023 约束与影响）。
2. r41 要求 Gate-aware 路径缺 Gate/事务边界时构造失败，不得注入 allow-all/no-op；同时又要求 S10.3 不修改 `cmd/han-server`，保留当前 `session.NewStore(pool)` production 调用，并由 S10.4 才完成共享实例注入。
3. S10.3 当前任务要求本卡可独立通过 Manifest 全量检查并 task-level 关闭，然后才进入 S10.1/S10.4。

**触发**

按 r41 在九文件内给 `LoginDependencies`、`PasswordChangeDependencies` 和 Session Gate-aware constructor 增加必需 Gate：

- 若严格拒绝缺 Gate，现役 `cmd/han-server` production composition 及其测试、登录集成测试和其他直接构造点会构造失败；要恢复 Manifest，必须修改 r41 九文件外且当前归 S10.4/其他已关闭范围的文件。
- 若为了保持现役 composition/tests 成功而让 Gate 可选、保留旧 constructor 的可授权 `ResolvePrincipal`，或另增只供新测试使用的 Gate-aware constructor，则旧 Login/PasswordChange/Session 路径仍可在不消费 Gate 时认证并返回 Principal，形成 r41 明令禁止的 parallel authority。

**影响**

r41 目前不能作为 Backend-Dev 的唯一可执行输入：实现者必须在“越出写集/提前修改 S10.4 composition”“让全量 Gate 失败”或“保留认证旁路”之间自行选择。前两项破坏规划所有权和独立关闭条件，后一项直接破坏 durable fail-closed 安全目标。该歧义也使“至少九文件”不是可核证的最小完整影响面。

**修复建议**

在不改变已采用的 `identity → session` 单向依赖和 S10.4 production composition owner 的前提下，明确一个可核证的安全过渡方案，并同步精确写集/任务关闭条件：

1. 明确 S10.3 结束时旧 constructor、旧 `ResolvePrincipal`、Login 与 PasswordChange 缺 Gate 的精确行为，证明其不能继续形成可授权旁路；以及
2. 列出为保持编译、全量测试和 fail-closed 所需的全部真实调用点/测试文件，由 Task-Planner 明确归 S10.3、S10.4 或不可单独部署的联合 Gate；不得只写“至少九文件”；以及
3. 增加自动化证明：缺 Gate/事务边界的所有 constructor 或旧入口不能产生 credential verify、Session mutation、Principal 或成功 production composition；新 Gate-aware 路径是唯一可授权路径；以及
4. 若仍坚持 S10.3 task-level 独立关闭且 `cmd/han-server` 完全留给 S10.4，需明确中间 revision 的非部署约束如何由可执行 Gate 阻止，而不是只靠“不宣称 production 接线”的文档声明。

**复审条件**

- 修订 design，给出唯一过渡状态和精确 constructor/call-site 行为；
- Task-Planner 对账精确写集、Manifest 与 S10.3/S10.4 交接；
- test-plan 覆盖旧入口不可授权、缺依赖 fail closed、全量构造调用点及 production root 交接；
- 原 Reviewer 对修订设计与真实消费者重新定点复审。

## 4. 其他重点结论

除 F001 外，未发现新的 HIGH/MEDIUM/LOW finding：

1. **端口与唯一 owner**：把中性端口放在 `internal/session`，由 `identity.PasswordResetStore` 唯一实现 durable classifier/Gate SQL，可保持 `identity → session` 单向依赖；接口未泄露 identity DTO。
2. **事务顺序**：Login/PasswordChange 的 User-first 调用点及 Session 的 token 预读→事务→User lock→Gate→Session/credential 重读→commit→Principal 主序自洽；预读不授权，commit error/unknown 零 Principal。
3. **登录抗枚举**：missing User 继续走 fake credential，Gate 只在确有并锁定 User 后调用，未引入用户名存在性旁路。
4. **唯一 classifier**：r41 明确消费者不得复制 operation 状态、lineage/settlement 或 SQL；瞬时错误只 fail closed、不写新 hold，与 r40 history/current 分权一致。
5. **公共合同**：未发现新增 migration/schema/依赖、公共 REST、公共错误码、权限或 boundary/ADR 核心变化。
6. **F006/r40 回归**：F006 保持 CLOSED；r40 current capability root / immutable legal history 分权未被重开。

## 5. 待决定与非阻断建议

### 待决定

需由设计作者、Task-Planner 与 Lead 在既有方向内确定 F001 的安全过渡所有权：是扩大/重排 S10.3 精确写集，还是把不能安全独立落地的构造切换纳入与 S10.4 相连的不可部署 Gate。Reviewer 不代替作者选择，也不直接修改 Tasks/Plan。

### 非阻断建议

后续 test-plan 同步时，把 Gate 的内部错误分类与三消费者既有公共错误映射逐项写明，尤其区分“确定 closed”与 PostgreSQL/query/commit 不可判定错误；不新增公共 code，但避免各消费者自行选择不一致映射。

## 6. Gate

### ADR consistency Gate

**PASS（未重开 ADR）**。

r41 的中性端口、唯一 identity classifier、三消费者 User-first/commit barrier 与 S10.4 production owner 均未改变 ADR-0023 已接受方向。F001 是该方向下的阶段切分与可实施性缺口，不需要 Reviewer 擅自新增 ADR。

### Design Review Gate

**CHANGES REQUIRED**

理由：F001 为阻断性设计 finding。当前 r41 不能证明 S10.3 在九文件内既可独立通过项目 Gate，又不会保留可授权的旧认证旁路；因此尚不足以直接交 Backend-Dev 实施或作为 Task-Planner 最终对账输入。

## 7. 任务与测试影响

1. Task-Planner 应在设计修订通过后重新核对 S10.3/S10.4 写集、直接 constructor 调用点、全量 Manifest 和 task-level close 语义。
2. test-plan 应补充 Gate-aware/legacy constructor、生产 root 未接线、旧入口不可授权、缺依赖 fail-closed 和跨卡交接矩阵。
3. S10.3 保持 active，但实现应继续停在 G0；不得因本报告的 ADR consistency PASS 继续写代码。
4. S10.4、S11.1/B6D 不启动；S10.1 继续 implementation stopped；代码 F002 保持 OPEN。

## 8. 实际命令与静态证据

执行了以下只读检查：

- `git branch --show-current`
- `git rev-parse HEAD`
- `Get-FileHash -Algorithm SHA256`（固定 design/stories、ADR、boundary、architecture、Tasks/Plan 与前置 Review）
- `Test-Path`（确认报告路径开始时不存在）
- `Get-Content -Encoding UTF8` 分段读取设计、ADR、boundary、Tasks/Plan 与直接消费者
- `rg -n` 检查 Gate、constructor、`NewStore`、`NewLoginService`、`NewPasswordChangeService`、`ResolvePrincipal`、任务写集和生产 composition 调用点

本轮未运行 Go、PostgreSQL、SMTP、部署或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。

## 9. 未验证风险与限制

1. 未实现或执行 r41 对应代码，不能证明具体 SQL、pgx transaction、rollback/commit unknown 或并发锁序行为。
2. 未运行真实 PostgreSQL 18、多连接、migration 混跑、Session/token 竞态或 commit-ack-loss。
3. 未验证 production composition、Host、部署、SMTP/TLS、浏览器/E2E、容量或性能。
4. 当前工作树含大量合法并行改动；本报告只对固定 design hash 和列明的有界消费者作结论。
5. stories r16 与 test-plan r18 的历史任务状态文字早于当前 S10.2 CLOSED / S10.3 active 现场；后续影响同步不得把旧状态当作当前事实。

## 10. 复审条件与状态边界

复审至少需要：

1. 固定修订后的 design hash；
2. F001 安全过渡、精确调用点和写集合同；
3. 更新后的 Tasks/Plan 与 test-plan 影响映射；
4. 证明不存在 Gate-optional/旧 constructor 可授权旁路的静态设计证据。

在上述条件满足前：

- S10.3：active / G0 blocked；
- S10.4、S11.1/B6D：不得启动；
- S10.1：implementation stopped；
- `IDENTITY-S10.1-REV-R01-F002`：OPEN；
- 本报告不授权实现、任务关闭、部署或发布。
