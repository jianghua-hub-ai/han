# identity-auth S10.1 Code Review R04

- 日期：2026-10-25
- objectType：code
- Reviewer：identity-auth-S10.1 非作者独立 Reviewer
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审来源：`.spec/wip/identity-auth/review-code-s10.1-r03.md`
- 来源报告 SHA256：`46732B803B37B8684EA23F8D8683559E27FAD3659423E3E5784244BA6A5D6933`
- 结论：R03 F001、F002、F003 均 **CLOSED**；发现 **2 个 Major** 新 finding。
- **Code Review Gate：CHANGES REQUIRED**

## 1. 固定对象与防漂移

三文件均为共享工作树中的 untracked 对象；本评审没有把空 `git diff` 当作代码证据，而是按固定 SHA256 全文检查并运行实际测试。

| 文件 | 固定 SHA256 |
|---|---|
| `internal/identity/handler.go` | `FCCF0290BEE93E7730AD8EE8B65A07B8BFC475D0BEA14C5A3AB2372D2C49C36C` |
| `internal/identity/password_reset.go` | `8B36788CE2095603AE202993E3CEC7E03EB38546E00688E46AC5A7A020AA470D` |
| `internal/identity/password_reset_test.go` | `7D448B256C330D45E72906CCDA739F0CFC19563A163D9FFBA04F2D94CF99AD34` |

落盘前再次核对：分支、HEAD、上述三文件哈希和 R03 哈希均未漂移，R04 路径此前不存在。

依据：

- identity design r44：`50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23`
- stories r16：`057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3`
- test-plan r20：`F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB`
- tasks/Plan 当前合同：S10.1 只拥有首次 reset Handler/coordinator 三文件返工，但目标同时要求保持公共 REST、权限及错误合同；若闭环必须越出三文件，应停止并回 Task-Planner/设计对齐。

## 2. 两轮实际检查范围

### Round 1：固定三文件直接审查

- 全文复核 Handler、coordinator、same-key 预分类、dispatch ownership、Mailer outcome、evidence 持久化、replay、错误 envelope 与 secret 清空。
- 全文复核 password-reset 测试入口、stub 行为以及 R03 三项 finding 的定点回归。
- 运行专项普通/race、测试枚举和 F001/F002/F003 精确测试。

### Round 2：真实消费者与上游边界

有界追踪：

- `password_reset_store.go`、model/SQL；
- `password_reset_operator.go`、`password_reset_worker.go`；
- Login、PasswordChange、Session Gate；
- Mailer 接口与 SMTP adapter；
- `cmd/han-server` 唯一 production composition/router；
- pgx v5.11.0 的 `pgxpool.Pool.BeginTx` / `pgx.Conn.BeginTx` context 行为；
- tasks/Plan 与 design r44 的 S10/I-PRD/I-CRED、production composition 相关合同。

没有扩展为无关全仓评审。

## 3. R03 Findings 定点复审

### `IDENTITY-S10.1-REV-R03-F001` — CLOSED

原问题是 same-key replay 在分类前生成秘密、ID、hash并读取 latest credential。

当前证据：

- `internal/identity/password_reset.go:202-219` 在 `Secrets.Generate`、密码策略、ID 分配、latest credential 和 hash 之前调用 `ClassifyExistingCandidate`。
- succeeded、failed、unresolved exact replay 均直接返回既有 operation/claim 分类。
- `internal/identity/password_reset_test.go:1114-1196` 对三类 replay 逐项断言 secret、ID、hash、verify、latest query 和 Mailer 调用均为零。
- 精确普通/race 测试通过。

### `IDENTITY-S10.1-REV-R03-F002` — CLOSED

原问题是把所有 Mailer error 持久化为明确 rejected，无法表达 provider unknown。

当前固定 revision 已修正该直接分支：

- `internal/identity/password_reset.go:310-316`：Mailer 任意 non-nil error 都进入 unknown 路径，不再推断 rejected。
- `internal/identity/password_reset.go:353-372`：目标写入 `state=commit_unknown`、`delivery_state=unknown` 并加速既有 recovery job；不直接激活/失效 credential，不写 terminal audit。
- `internal/identity/password_reset_test.go:1361-1450`：验证 unknown/commit-unknown 不写 rejected、不做有限 cleanup、不改变旧 credential/session 能力。

因此 R03 描述的“错误被直接归类为 rejected”已关闭。下文 R04-F001 是 request cancellation 使该新 unknown evidence 路径无法可靠执行的独立新缺陷，不重写本项历史结论。

### `IDENTITY-S10.1-REV-R03-F003` — CLOSED

- 顶层入口已恢复：
  - `TestPasswordResetAuthorization`：`password_reset_test.go:1082`
  - `TestPasswordResetIdempotency`：`:1090`
  - `TestPasswordResetPreparationFailures`：`:1098`
  - `TestPasswordResetHandler`：`:1106`
- `go test -list '^TestPasswordReset'` 可见上述入口及新增 replay/unknown 测试。
- 对应普通/race 测试实际通过。

## 4. 新 Findings

### Major — `IDENTITY-S10.1-REV-R04-F001`：Mailer 返回后复用已取消的请求 context，accepted/unknown evidence 可能完全丢失

- 位置：
  - `internal/identity/password_reset.go:310-320`
  - `internal/identity/password_reset.go:326-372`
  - `internal/identity/password_reset_operator.go:186-189`
  - `internal/identity/password_reset_operator.go:522-575`
  - `internal/identity/password_reset_test.go:501`
  - `internal/identity/password_reset_test.go:801`
- 触发：
  1. `prepared → dispatching` 已提交，唯一 SMTP 发送权已经消耗；
  2. Mailer 在 parent request cancellation/deadline 后返回 error，或返回 accepted 后请求 context 随即被取消；
  3. `recordUnknownPasswordReset` / `recordAcceptedPasswordReset` 继续使用原 request `ctx` 开启 evidence 事务。
- 直接证据：
  - Mailer 使用 `context.WithTimeout(ctx, SMTPBudget)`，但 evidence 方法接收的仍是原始 `ctx`。
  - production transactor 是 `*pgxpool.Pool`；pgx v5.11.0 `pgxpool/pool.go:815-829` 的 `BeginTx` 先以传入 context `Acquire`，`tx.go:100-103` 又用该 context 执行 `BEGIN`。已取消 context 下不能依赖 evidence 事务成功。
  - 当前 transactor stub 在 `password_reset_test.go:501` 忽略 context，Mailer stub在 `:801` 也忽略 context，因此现有绿色测试掩盖了该 production 路径。
  - evidence 丢失后 durable operation 保持 `dispatching/delivery=dispatching`，job 未加速；operator 后续把“delivery != accepted 且 operation=dispatching”的记录走 unaccepted settlement并失效候选，而不是保持 unknown unresolved Gate。
- 影响：
  - provider unknown 不能可靠持久化为 `commit_unknown/unknown`，随后可能被错误终结为 invalidated并提前解除 Gate，违反 I-PRD-07/design r44 的 unknown fail-closed 收敛合同。
  - Mailer 已明确 accepted、但请求恰在返回后取消时，accepted evidence同样可能丢失；收件人收到的 temporary credential随后可能被 worker 当成未受理候选失效。
  - HTTP 仍返回 uncertain/error不能补偿 durable authority 的错误分类；不重发约束使该证据无法从请求线程重建。
- 建议：
  - delivery attempt 已发生后，使用与 parent cancellation 隔离、但重新施加有限预算的 reconciliation context 写 accepted/unknown evidence；需保留允许的 context values，不能直接使用无 deadline 的后台 context。
  - 若当前合同没有可复用的 evidence DB budget，应先由设计/Task-Planner明确配置与所有权，不能在本三文件中硬编码。
  - 增加 context-aware Mailer/transactor/tx fake：Mailer 内取消 parent，分别覆盖 non-nil error和 accepted返回；断言 evidence事务实际执行、unknown持续 Gate、accepted最终由 operator激活，且超时有界、无重发。
- 复审条件：
  - 上述两个 cancellation 矩阵普通/race 通过；
  - 证明 production pgx 路径不再使用已取消 context；
  - unknown 不进入 invalidated/terminal audit，accepted 不被错误按 unaccepted settlement。

### Major — `IDENTITY-S10.1-REV-R04-F002`：唯一 production composition 未构造或注册 password-reset Handler，公共 REST 实际不可达

- 位置：
  - `internal/identity/handler.go:369-374`、`:890-966`
  - `cmd/han-server/main.go:2024-2069`
  - `cmd/han-server/main.go:2170-2204`
  - `cmd/han-server/main.go:2404-2429`
  - `cmd/han-server/main.go:2613-2686`
- 触发：通过真实 `han-server` 请求：
  - `POST /api/v1/admin/users/{id}/password-reset`
  - `POST /api/v1/admin/administrators/{id}/password-reset`
- 证据：
  - `productionIdentityHandlers` 没有 password-reset handler 字段。
  - `buildIdentityHandlers` 未调用 `NewPasswordResetService` / `NewPasswordResetHandler`。
  - `protectIdentityHandlers` 未给 reset Handler 注入 OriginGuard/authorizer。
  - production router 只分派登录、改密、三条管理员/隐私路由及三条 article DELETE；两条 reset path 均进入稳定 `request.route_not_found`。
  - 全仓 Go 调用搜索显示 `NewPasswordResetService` / `NewPasswordResetHandler` 只在 `password_reset_test.go` 使用。
  - `go test ./cmd/han-server -list 'PasswordReset'` 只有 worker/lifecycle/Store共享/config 测试，没有 reset coordinator/Handler/route 测试。
- 违反依据：
  - S10.1 task 目标明确要求保持公共 REST/权限/错误合同不变；
  - design r44 列出两条首次 reset 公共路由，并要求 production composition 缺 atomic/dispatch/Gate/worker 时 fail closed，而不是永久不注册正常能力；
  - 本轮目标同样要求公共 REST 不变。
- 影响：
  - 三文件单元测试中的 reset 能力在 production server 中不可调用；实际客户端稳定得到 route-not-found。
  - 当前测试可证明 Handler 直接调用，但不能证明唯一 composition root 的真实消费者链。
- 建议：
  - 该修复需要修改 `cmd/han-server` composition/router及相应测试，超出本轮三文件写集。按 S10.1 停止条件，应先由 Lead/Task-Planner/设计明确顺序所有权，不得在三文件内建立第二 composition root、隐藏全局或第二 Store/Gate。
  - 获得合法写集后，在唯一 root 构造 coordinator/Handler，复用 S10.3/S10.4 已有的同一 `PasswordResetStore`、Gate、operator/worker与现有安全依赖，并为两条 path 增加 exact method、Origin/authorization、404/405和零下游调用测试。
- 复审条件：
  - 明确 production wiring 所属 task及写集；
  - 两条 route 在唯一 production router 可达且权限、Idempotency-Key、envelope、错误和 secret safety合同不变；
  - 缺任一必需 durable dependency时启动/路由 fail closed；
  - 证明零第二 Store/Gate/operator/composition root。

## 5. 实际命令与结果

以下均在仓库根目录实际执行：

| 命令 | 结果 |
|---|---|
| `go test ./internal/identity -run '^TestPasswordReset' -count=1` | PASS |
| `go test -race ./internal/identity -run '^TestPasswordReset' -count=1` | PASS |
| `go test ./internal/identity -list '^TestPasswordReset'` | PASS；四个恢复入口及 replay/unknown 测试可见 |
| `go test ./internal/identity -run '^(TestPasswordResetSameKeyReplayPrecedesSensitivePreparation|TestPasswordResetUnknownDeliveryPreservesGateAndOldCapability|TestPasswordResetUnknownDeliveryCommitUnknownDoesNotUseFiniteCleanup|TestPasswordResetAuthorization|TestPasswordResetIdempotency|TestPasswordResetPreparationFailures|TestPasswordResetHandler)$' -count=1` | PASS |
| 上述精确集合加 `-race` | PASS |
| `go test ./internal/identity -run '^(TestLoginPasswordResetGate|TestOnboardingPasswordChangePasswordResetGate|TestPasswordResetOperator|TestPasswordResetWorker)' -count=1` | PASS |
| `go test ./internal/session -run '^(TestPasswordResetGateConstruction|TestSessionResolvePasswordResetGate)' -count=1` | PASS |
| `go test ./cmd/han-server -run 'PasswordReset' -count=1` | PASS；覆盖 worker/composition Store 共享，不覆盖 reset route |
| `go test ./cmd/han-server -list 'PasswordReset'` | PASS；确认不存在 reset Handler/route 顶层测试 |
| `go vet ./internal/identity ./internal/session ./internal/mailer ./cmd/han-server` | PASS |
| `git diff --check -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go` | PASS；因三文件 untracked，此命令只作补充，非全文审查证据 |
| `gofmt -l -- internal/identity/handler.go internal/identity/password_reset.go internal/identity/password_reset_test.go` | PASS；无输出 |
| `rg` 定点搜索 `NewPasswordResetService` / `NewPasswordResetHandler` / production routes | 仅测试构造；production composition 无调用/路由 |
| `Get-FileHash -Algorithm SHA256`、`git branch --show-current`、`git rev-parse HEAD` | 固定哈希、`main`、固定 HEAD 均匹配 |

## 6. 未验证风险与限制

- 按禁止项未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`，未连接真实 PostgreSQL/SMTP，未部署、未执行 E2E。
- 未验证真实 provider 对取消、超时与受理 acknowledgement 的具体行为；R04-F001 基于固定代码路径、production pgx 实现及 worker状态机成立。
- R04-F001 没有新增动态测试复现：唯一允许写入的是本报告，而现有 fake 明确忽略 context；这也是 finding 所述测试缺口。
- 未执行真实多连接、commit-ack-loss、migration 混跑、worker crash/restart、容量和 production readiness 验证。
- Lead 报告的全量 Manifest 结果没有被本 Reviewer 冒充为本轮独立执行；本轮独立重跑的是上表专项、消费者、vet、格式与 scoped 检查。
- 共享工作树的其他大量修改/未跟踪文件不在本结论范围，也未被回滚、覆盖、格式化、stage、commit或清理。

## 7. Gate

| 项目 | 结论 |
|---|---|
| `IDENTITY-S10.1-REV-R03-F001` | **CLOSED** |
| `IDENTITY-S10.1-REV-R03-F002` | **CLOSED** |
| `IDENTITY-S10.1-REV-R03-F003` | **CLOSED** |
| `IDENTITY-S10.1-REV-R04-F001` | **OPEN — Major** |
| `IDENTITY-S10.1-REV-R04-F002` | **OPEN — Major / 需先对齐跨文件所有权** |

**Code Review Gate：CHANGES REQUIRED**

阻断项是 R04-F001 与 R04-F002。R03 三项修复本身均已闭合，但当前 revision 仍不能宣称 S10.1 的 durable outcome 和公共 REST 消费链完整。是否接受风险、如何分配跨文件修复所有权及后续实施授权，由 Lead/用户决定；本评审不自行修改架构、公共契约或被审代码。
