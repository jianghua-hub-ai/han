# identity-auth-S10.3 独立代码评审 R01

- Reviewer：Gibbs（非作者）
- 日期：2026-10-24
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 被审任务：`identity-auth-S10.3`
- 结论：**Code Review Gate: CHANGES REQUIRED**

## 1. 状态边界

本报告只评审 S10.3 的 13 文件实现，不修改被审代码、Tasks/Plan、design/test-plan 或其他报告。结论不关闭 S10.3，不启动 S10.4/S11.1/B6D，不改变 S10.1 `implementation stopped` 或代码 finding `IDENTITY-S10.1-REV-R01-F002` OPEN，也不表示部署、E2E 或 production readiness。

`han-server.exe` 是未获清理授权的未跟踪构建制品；开始核得 SHA256 `A92D062BC163A48277107F02D812F66AE77A213BBE707CB2CBFEB2F63EAF4145`、大小 18,137,600 bytes。本轮未修改或删除。

## 2. 固定输入与开始哈希

上游输入开始核对均匹配：

| 对象 | SHA256 |
|---|---|
| design r42 | `CEF17810A42DECF35F0B9A0585877B6FDFEEF77557E2A314D6BF536D2940D176` |
| test-plan r19 | `0381614EA06B34756C430989D04BAF0F8A55C5FAB24C713711535EB1D78AD0DE` |
| Tasks | `6E58D74AD88870A76E274F5ACC66CBB38240D1A8E73ADC28D99AD2BC154C5F97` |
| batch Plan | `AF5FCDF470168AFAD5B42F72522549FB7D5B550E79E414F1DC517512EA4A774D` |
| design Review R02 PASS | `94E8C180D4961C6EB141806DFA9662566C8BEC07CB968D3ACCD8F6EAE7D868A2` |

13 个固定代码对象开始哈希均匹配：

| 文件 | SHA256 |
|---|---|
| `internal/identity/password_reset_store.go` | `6C0136408EF62A7B5FC80C4C557AEBCB557457B772EB96E8DA90024C42D3315F` |
| `internal/identity/password_reset_store_test.go` | `62EAE06BD41E5F3CFA7AAB03B3721B8D3A964834A0E7C98B194181A2791827E6` |
| `internal/identity/login.go` | `1656AB4F808B3E5264D8BC040CD5162B5FCEB8BCE99808D4BAC2A7C6BB365B66` |
| `internal/identity/login_test.go` | `038C92750AF67D4C3A3E7699CBC3BA941CC8849ACAA029D83D50083076A7E4DF` |
| `internal/identity/login_integration_test.go` | `15C28D7BA7EE39499D970E8C7AE53F7CE28A1A450D2530FD7697BF8BE4D407BA` |
| `internal/identity/service.go` | `975A280A16385237C60BC5C98DAB648005E719ACDF81736EB1B4F46C1B0EDA19` |
| `internal/identity/bootstrap.go` | `AEF53946AB8C55A3CC03DAE03DC1CD0E42BE12553D9619D45C7F491F154CAAA5` |
| `internal/identity/bootstrap_test.go` | `CD4DD62DFB6768AB1ED00E649D55F71701ED5A3BFC8EEE5AA0F88031577F701D` |
| `internal/identity/administrator_create_test.go` | `AC9FE098FA3F1F09FA4944BB5BD7653BF375B23C08ED1E63F13DCEE45683EEAE` |
| `internal/session/store.go` | `D06E215FF2F065618B728B66A5899015D36604AE901DF7034F1AB1D061B199C9` |
| `internal/session/store_test.go` | `8B67AF590A168159A9518D65C361501CF32740F01E316BB2A000D0303630C79D` |
| `cmd/han-server/main.go` | `F4D82EB4BEC05FD44E9FC0CE70142C439FF2719EEC8FAD35915BF94387CD15BD` |
| `cmd/han-server/main_test.go` | `FF82B1AF356249FA0366BD60EC729A4C58D630EF3FD8343431D3090478AE27A9` |

这些文件在当前工作树均为未跟踪对象，因此评审读取全文，未用普通 `git diff` 代替对象审查。

## 3. 两轮实际检查

### Round 1：固定对象直接检查

1. 完整读取 13 文件，共 24,990 行、781,428 bytes，并核接口、constructor、调用点、错误路径、事务关闭和测试替身。
2. 全仓定向搜索 `PasswordResetGate`、`CheckLockedUser`、`NewStore`、`NewLoginService`、`NewPasswordChangeService`：
   - production `CheckLockedUser` 实现唯一位于 `identity.PasswordResetStore`；
   - test-only fake 仅声明于 `identity/login_test.go`、`session/store_test.go`、`cmd/han-server/main_test.go`；
   - `internal/session` 未反向导入 `internal/identity`；
   - production root 当前确实只构造一个 `*identity.PasswordResetStore`，并把同一变量传给 Session/Login/PasswordChange；
   - 未发现 legacy `NewStore(QueryRower)`、optional/no-op/allow-all/deny-all production Gate 或第二 classifier。
3. 核对三消费者顺序：
   - Login：真实 User `FOR UPDATE` 后 Gate，再 credential verify/rehash 与 Session 写入；missing User 仍走 fake credential 且不调用 Gate。
   - PasswordChange：caller-owned transaction 内锁 User 后 Gate，再读取/校验 credential、settlement、永久密码与 Session mutation。
   - Session：token 预读只定位 User；随后 begin transaction → User lock → Gate → Session/credential 重读 → 必要 revoke → commit → Principal；commit error/unknown 不返回 Principal。
4. 检查 Gate SQL 与 `0008_password_reset_durable.sql` 的 FK/UNIQUE/CHECK/terminal immutable 约束；当前实现没有复制 resend classifier，也没有写 hold。真实 SQL 行为仍须由 PostgreSQL 专项验证。
5. 以真实函数体边界检查 go-service 强制复杂度规则。

### Round 2：上游与直接消费者深审

1. 沿 design r42、test-plan r19、Tasks/Plan、accepted ADR-0023 和 S10.2 durable Store/schema 核对 owner、13 文件写集、公共错误、单向依赖及 S10.3→S10.4 分界。
2. 核对 production composition 在 listener/routes/Serve 前完成 factory；当前代码的单实例接线正确，构造失败由 `runComposed` 在开放 listener 前收敛。
3. 审阅 clear/closed/query/begin/User-lock/commit 测试及 singleton wiring 测试；Reviewer 独立执行定向普通、race 和三包 `go vet`，结果均 PASS。
4. 检查公共 REST/错误码、schema/migration、依赖未因本卡变化；未发现第 14 个必需文件。

## 4. Findings

### IDENTITY-S10.3-REV-R01-F001 — MEDIUM — mandatory interface 依赖的 typed-nil 可穿过 constructor 检查

**位置**

- `internal/session/store.go:148-155`
- `internal/identity/service.go:303-335`
- 对应测试仅覆盖 untyped `nil`：`internal/session/store_test.go:18-34`、`internal/identity/login_test.go:33-50`、`internal/identity/bootstrap_test.go:200-215`

**违反依据**

- design r42 要求唯一 mandatory **non-nil** Gate，缺 Gate/transaction boundary 必须在启动或构造阶段 fail fast。
- test-plan r19 I-PRD-11～13 要求 nil Gate/transaction boundary 的 constructor 负例。
- 安全边界不得把无效依赖延迟到认证请求期。

**触发**

Go interface 持有 typed-nil 指针时，interface 本身不等于 `nil`。例如把 `(*identity.PasswordResetStore)(nil)` 赋给 `session.PasswordResetGate`，或把 typed-nil transactor 赋给 `StoreTransactor`，当前 `== nil` 判断会接受该依赖。Login 和 PasswordChange constructor 对 Gate 有同样问题。

**影响**

无效安全依赖可被 constructor 当作成功装配；调用 Gate 时可能 panic 或发生实现相关运行期故障。当前 production root 使用明确的非 nil literal，静态检查未发现现成认证旁路，但 mandatory/fail-fast 合同没有被 constructor 自身守住，错误会从启动阶段拖到认证流量阶段，形成可避免的服务不可用面。

**修复建议**

在不改变公共 REST、包依赖方向或新增 authority 的前提下，集中拒绝 nil 与 typed-nil 的接口依赖；至少覆盖三处 Gate constructor 以及 Session transaction boundary。新增 typed-nil Gate/transactor 负例，并断言失败发生在 credential probe、DB 业务调用、route/listener/Serve 之前。

**复审条件**

1. 三个 constructor 对 typed-nil Gate 均 fail fast；`session.NewStore` 对 typed-nil transaction boundary 同样拒绝。
2. 测试验证零 verifier/DB/Session/listener/route 副作用。
3. S10.3 定向普通/race、适用 Manifest 与 scoped diff-check 不退化。

### IDENTITY-S10.3-REV-R01-F002 — MEDIUM — 三个本卡直接修改的安全关键函数仍超过强制 50 行

**位置与实测函数体**

- `internal/identity/login.go:1083-1381`，`Login`：299 行。
- `internal/identity/service.go:339-694`，`ChangePassword`：353 行。
- `cmd/han-server/main.go:1930-2296`，`newProductionBusinessHandler`：364 行。

**违反依据**

`.codex/profiles/go-service/rules/coding-standards.md` §1.2 是已启用 profile 的强制规则：函数最大 50 行、超出必须拆分；core coding rule 同样要求超过 profile 阈值先拆分再提交。

**触发与影响**

本卡把 User-lock→Gate→credential/Session 和 production singleton wiring 插入上述三个长流程。即使大部分体量来自既有代码，它们都是本卡直接修改并承担安全顺序证明的函数；当前结构把事务关闭、公共错误映射、审计、凭据状态和 composition 所有权集中在数百行流程中，增加漏掉回滚、错置 Gate、错误注入第二实例或后续回归绕过 Gate 的审查难度。

**修复建议**

只在既有 13 文件和已确认合同内拆出可命名阶段，保持调用顺序、事务所有权、错误 cause、公共错误、路由和实例 identity 不变。不要借复杂度修复重写 classifier、扩大 S10.4 或改变公共合同。

**复审条件**

1. 本卡直接修改的 production 安全关键函数均满足函数体不超过 50 行和嵌套上限。
2. 拆分后的 Login/PasswordChange/Gate/Session/composition 调用轨迹与现有测试一致。
3. 定向普通/race、适用 Manifest 与 scoped diff-check PASS。

## 5. 未形成 finding 的核对结果

- `PasswordResetStore.CheckLockedUser` 保持 identity-owned 唯一 SQL owner，函数本身只读；query/cancel 错误返回非 nil，未写新 hold。
- 当前 production root 的同一指针注入、`identity → session` 单向依赖及三消费者错误映射与 r42/r19 一致。
- Gate SQL 未要求 current root 永远等于旧历史 operation；未发现把 r40 合法 A→B 或 A→B→temporary C 历史重新判为 hold 的新增分支。
- Session commit 成功前不返回 Principal；commit error/unknown 返回空 Principal。
- 未发现新增公共 REST、公共错误码、schema/migration、第三方依赖或第 14 文件。

## 6. 待决定与非阻断建议

### 待决定

无新的架构或公共合同决定。F001/F002 均可在既有 r42、13 文件与单向依赖内修复；若实现者判断必须增加文件、改变接口或调整既定架构，应停止并交 Lead，而不是在代码评审中自行扩界。

### 非阻断建议

`cmd/han-server/main_test.go:2889-2918` 的 singleton 测试通过 production observation seam 比较指针；当前静态代码与观察值一致，但 observation 是 composition 手工上报。后续可让测试 seam 捕获各 constructor 实际收到的接口实例，减少“注入值改错但 observation 仍上报原变量”的同源断言风险。

## 7. Reviewer 实际命令与证据

Reviewer 自跑：

1. branch/HEAD、报告不存在、13 文件与五个上游 SHA256、未跟踪状态核对。
2. 13 文件全文读取及静态调用面/函数体分析。
3. `gofmt -d`（13 文件）：无输出。
4. 定向普通：
   `go test ./internal/identity ./internal/session ./cmd/han-server -run 'Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|SessionResolvePasswordReset|PasswordResetGate|ProductionPasswordResetWiring)' -count=1`
   — PASS。
5. 定向 race：
   `go test -race ./internal/identity ./internal/session ./cmd/han-server -run 'Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|SessionResolvePasswordReset|PasswordResetGate|ProductionPasswordResetWiring)' -count=1`
   — PASS。
6. `go vet ./internal/identity ./internal/session ./cmd/han-server` — PASS。
7. 全仓/固定范围 `rg` 核 constructor、Gate 实现、fake 与 import direction。

测试子进程显式移除敏感 DSN 环境变量；Reviewer 未读取或输出其值。

Lead 提供但 Reviewer 未冒充自跑的外部证据：13 文件 hash、scoped gofmt/diff-check、定向普通/race、三包普通/race及 Manifest 七项均 PASS。

## 8. 未验证风险

- 未运行真实 PostgreSQL 18、真实 SQL Gate fixture、多连接锁序、commit actual-state/unknown 或 migration 混跑；这些仍归 S1.6。
- 未运行 SMTP、部署、Host、浏览器或 E2E；未证明 production readiness。
- 未独立重跑 Lead 的全仓 Manifest 七项；本报告只记录 Lead 外部证据和 Reviewer 上述定向证据。
- 13 文件均为未跟踪对象，普通 Git diff 无法提供历史基线；本轮以固定 SHA、全文读取、上游合同和真实调用面审查。
- `han-server.exe` 仍是未跟踪交付残留，未获授权清理。

## 9. Gate

**Code Review Gate: CHANGES REQUIRED**

原因：`IDENTITY-S10.3-REV-R01-F001` 与 `IDENTITY-S10.3-REV-R01-F002` 均为未关闭 MEDIUM。定向普通/race/vet PASS 不能替代 mandatory constructor fail-fast 与强制复杂度规则。修订应回原作者，随后由非作者 Reviewer 定点复审；本结论不自动改变任务状态或后继启动条件。

## 10. 结束验证

- branch / HEAD 仍为 `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 13 个固定代码对象结束 SHA256 与 §2 逐项一致，均无漂移。
- `han-server.exe` 结束 SHA256 仍为 `A92D062BC163A48277107F02D812F66AE77A213BBE707CB2CBFEB2F63EAF4145`，大小仍为 18,137,600 bytes。
- 报告 scoped `git diff --no-index --check -- /dev/null .spec/wip/identity-auth/review-code-s10.3-r01.md` 无 whitespace error；退出码 1 仅表示新增文件与空文件存在预期差异，按 scoped diff-check 判定 PASS。
- 工作树状态确认本轮唯一写入为本报告；13 个被审对象和未授权二进制均未修改。
