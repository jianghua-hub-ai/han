# identity-auth-S4.1 独立代码评审 r01

- **评审日期**：2026-10-23
- **Reviewer**：非作者 Reviewer（与实现作者 Noether `01a16893-bc25-7a52-8532-4bd7e203565d` 职责隔离）
- **objectType**：code
- **仓库 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **任务**：`identity-auth-S4.1`
- **结论**：**FAIL；2 个 HIGH、1 个 MEDIUM finding；Code Review Gate FAIL。**

本结论只绑定下列六个固定代码对象，不关闭 S4.1、B5A/B5B，不表示 PostgreSQL、浏览器、部署、E2E 或 production readiness 已完成。

## 1. 固定对象与修订

评审开始、报告写入前均复核六文件 SHA-256，结果与委托值一致：

| 文件 | SHA-256 |
|---|---|
| `internal/identity/bootstrap.go` | `AC4B164F2BC0024E67E2805905C5BB982ECAABD9563A6BFF67CDF98C32F593B6` |
| `internal/identity/bootstrap_test.go` | `72F0F46892BB325F7C1818DF12BF5DAACEB60D20C0632DF002DC44E95ABEA80B` |
| `internal/identity/handler.go` | `5A073A613FC59B96E98F7172B35704C1E92E353E9F50BA232ECF3C6996C2D13F` |
| `internal/identity/service.go` | `ADC7202DBF5907A05C5B36F6CAD4C44CFF3393FD81D342E900EC4614884B66DA` |
| `cmd/han-server/main.go` | `5DBACCB74EF15F53BCDCC1845ADE8B66327163F8817A84C708489C0082E61988` |
| `cmd/han-server/main_test.go` | `D83244E9B2F8277FDE485D7FA2DCD14ADB570F9A3FD53361DB48550C903DDA77` |

共享工作树含其他人的未提交成果；本轮未回滚、覆盖、格式化或修改任何被审代码及范围外文件。

## 2. 评审依据

实际使用：

- Reviewer Role、`add-code-review`；
- code-review-checklist、abstraction、coding、no-hardcode；
- Go service coding/no-hardcode/security/observability/tracing profile，以及适用的核心 observability/tracing 规则；
- tasks S4.1、Plan B5A/B5B identity lane；
- identity boundary r12、architecture r24、design r28、test-plan r11；
- `.spec/wip/identity-auth/review-design-r28-q1-r03.md` 的 Design Review Gate PASS；
- 六个固定文件全文及直接消费者：
  `internal/session/store.go`、`internal/platform/http_security.go`、
  `internal/security/password.go`、`internal/audit/store.go`、
  `internal/identity/repository.go`、`internal/identity/model.go`、
  `migrations/0002_identity.sql`。

## 3. 两轮实际检查

### Round 1：固定六文件直接代码审查

检查了：

1. bootstrap advisory transaction lock、唯一性检查、环境密码读取时机、密码策略/摘要、User/Identifier/Audit 原子写入、commit/rollback；
2. 改密主体资格、失败顺序、credential kind/id、临时凭据状态、密码验证/策略、摘要更新、Gate、Session 撤销/轮换、Audit、事务边界；
3. Handler 的主体先行拒绝、严格 JSON、Cookie/trace、响应 Cookie 与敏感信息；
4. production composition、exact `PUT /api/v1/users/me/password`、Trace/router/Origin/Session/CSRF/授权顺序及 JSON 404/405/Allow；
5. 测试是否覆盖真实合同，而非只匹配实现。

除 findings 外，确认：

- bootstrap 密码只在锁内确认需要创建后读取；
- 两个并发 bootstrap 由事务级 advisory lock 串行，User、username Identifier 和成功 Audit 同事务；
- 成功改密的永久摘要、临时凭据消费、当前 Session 轮换、其他 Session 撤销和成功 Audit 在同一事务；
- 密码规则为至少 8 个 Unicode 字符且同时含字母和数字；
- router 顺序及 JSON 404/405、`Allow` 未回退。

### Round 2：直接消费者与上游契约审查

沿 `main/router → OriginGuard → HTTPAuthorizer → Handler → Service → repository/session/audit/security` 检查：

- Session middleware 从当前 User/Session 构造 Principal，CSRF 在授权/Handler 前完成；
- Session 只保存 token/CSRF 摘要，改密依赖 caller-owned transaction；
- repository 对 User 与最新 TemporaryCredential 使用行锁；
- migration 的单 initial-admin、identifier、Session credential kind/id 约束；
- r28/r11 固定的 403/401/422、成功后置及 I-AUD-01 审计消费合同；
- production composition 使用真实 PostgreSQL repository/session/audit 与共享密码组件。

该轮发现锁等待后的权威状态与错误优先级冲突，以及 AuditEvent 合同未被真实消费者满足。

## 4. Findings

### CRITICAL

无。

### HIGH

#### IDENTITY-S4.1-R01-F001 — 并发成功改密后，等待锁的陈旧 credential 请求返回 403 且不撤销当前 Session

- **位置**：
  - `internal/identity/service.go:332-344`
  - `internal/identity/service.go:346-385`
  - `internal/identity/service.go:513-566`
  - 缺失回归：`internal/identity/bootstrap_test.go:185-217`
- **违反依据**：
  - design r28 `:1079`：主体检查后先重新核对当前 Session credential kind/id；
  - design r28 `:1088,2185`、boundary r12 `:73`：失配必须撤销当前 Session 并返回 `401 auth.session_required`；
  - test-plan r11 `:252-268`：credential 绑定失效矩阵；
  - tasks S4.1 `:487`：陈旧、失配或失效绑定固定 401 且只撤销当前 Session。
- **触发场景**：
  1. 非 initial admin 的 temporary/forced 用户用同一有效 Session 并发提交两个改密请求；
  2. 两请求都通过 Session middleware，Principal 均记录 forced/temporary；
  3. 请求 A 获得 User 行锁并成功把权威 `credential_mode` 改为 `permanent`；
  4. 请求 B 随后取得锁，读到 permanent 非 initial-admin；
  5. B 在调用 `passwordChangeCredential` 前被 `service.go:339-344` 判为 403。
- **影响**：陈旧 credential 绑定没有按安全合同返回 401，也没有执行 `service.go:360-380` 的“只撤销当前 Session”。调用方得到错误的授权结论，且该陈旧 Session 的失效后置不能由本事务证明。
- **证据**：当前 stale 测试只覆盖“权威账号仍为 bootstrap、Session kind 为 permanent”，未覆盖锁等待期间权威凭据由 temporary/forced 变为 permanent 的真实竞争。
- **最小修复建议**：在取得权威 User 行锁后，先区分“请求原本具备 forced 改密资格但绑定已因并发变旧”和“本来就是非强制越权主体”；前者必须进入 credential binding 的 401 + 当前 Session 撤销路径，后者才返回 403。不要放宽普通非强制账号权限。
- **复审验证**：使用确定性同步屏障/两个事务构造 A 先提交、B 后获锁；断言 B 为 401、verifier 0 次、只撤销 B 当前 Session、凭据/Gate/其他 Session 不变。不得用 sleep 证明顺序。

#### IDENTITY-S4.1-R01-F002 — 改密失败和陈旧绑定撤销没有写 AuditEvent

- **位置**：
  - `internal/identity/service.go:171-174`：端口被限定为“success evidence”；
  - `internal/identity/service.go:304-309,339-344`：403 无审计；
  - `internal/identity/service.go:360-384`：提交当前 Session 撤销但无审计；
  - `internal/identity/service.go:387-403`：当前密码/策略 422 无审计；
  - `internal/identity/service.go:480-499`：仅成功路径追加事件；
  - `internal/identity/handler.go:470-489`：Handler 前置拒绝也没有改密审计接缝。
- **违反依据**：
  - boundary r12 `:56,196`：凭据动作必须追加审计，高风险业务状态与审计同事务；
  - architecture r24 `:557`：高风险状态与 AuditEvent 同事务，审计失败使业务 fail closed；
  - design r28 `:1965`：正常成功/失败同事务 INSERT；
  - test-plan r11 `:301-307` 的 I-AUD-01：凭据成功/失败需具备 action、actor、target、result、code、trace。
- **触发场景**：
  - 非强制账号越权改密；
  - 当前密码错误或新密码策略失败；
  - Session credential kind/id 陈旧；
  - 尤其 `service.go:360-380` 会提交 `credential_binding_invalid` Session 撤销，却不在同一事务追加失败 AuditEvent。
- **影响**：高风险凭据失败及安全性 Session 撤销没有权威追加式证据，无法按 trace/action/code 追查尝试；审计消费者和 I-AUD-01 无法区分“没有发生”与“发生但未记录”。
- **最小修复建议**：为每个稳定公共失败结果生成最小、去敏的 `identity.password.change` 失败事件，包含 actor/target/outcome/error_code/trace；凡有 Session 或凭据状态变化，事件必须与该变化同事务提交，Audit Append 失败则整体回滚。Handler 前置失败应使用受控的改密审计接缝，不得记录密码、Cookie、CSRF、摘要或原始错误。
- **复审验证**：表驱动覆盖 403、401、两类 422、内部持久化失败和成功；逐项断言事件字段、敏感 denylist、调用次数、事务提交/回滚，以及 Audit Append 失败时 Session/凭据/Gate 无部分状态。

### MEDIUM

#### IDENTITY-S4.1-R01-F003 — 成功审计使用了非合同 action `identity.password_change`

- **位置**：
  - `internal/identity/service.go:489-493`
  - 缺失断言：`internal/identity/bootstrap_test.go:219-251`
- **违反依据**：design r28 `:2129-2137` 固定改密 action 为 `identity.password.change`。
- **触发场景**：任一成功强制改密或 initial admin 正常改密。
- **影响**：成功事件被写入另一个未登记 action namespace；按合同 action 查询、告警、审计集成或恢复核对会漏掉实际改密成功事件。现有成功测试未断言 action，因此全部定向测试仍可通过。
- **最小修复建议**：使用单一常量 `identity.password.change`，成功和失败共用该 action；测试精确断言 action、actor、target、outcome、error_code、trace。
- **复审验证**：成功事件只出现 canonical action；仓内不再出现 `identity.password_change`；审计消费测试按 canonical action 能读到事件。

### LOW

无独立 LOW finding。

## 5. Code Review Gate

### Code Review Gate：FAIL

F001 破坏已确认的陈旧 credential 401 + 当前 Session 撤销合同；F002/F003 破坏高风险凭据动作的追加式审计及稳定 action 合同。S4.1 不应关闭，也不应交接共享 `handler.go` 给 S7.1。

复审条件：

1. 原作者固定新的六文件修订并逐项回应 F001～F003；
2. 增加确定性并发回归和成功/失败审计矩阵；
3. 运行 S4.1 定向 identity/server、race、Manifest 项目检查及 scoped diff-check；
4. 由非作者 Reviewer 对原 findings、修复修订和相关回归定点复审。

## 6. Reviewer 实际验证

```text
go test ./internal/identity -run "TestBootstrap|TestInitialAdmin|TestOnboarding" -count=1
go test ./cmd/han-server -run "Test.*(Password|Router|Bootstrap|ProductionBusiness)" -count=1
```

两条命令均 PASS。PASS 只证明现有测试通过；F001 的锁等待竞争、F002 的失败审计和 F003 的 canonical action 均未被现有测试覆盖。

Lead 已报告 identity/server 定向、gofmt、`go mod verify`、vet、build、contract、race、全量测试及 owner diff check PASS；本报告将其作为外部已有证据，不冒充 Reviewer 独立复跑。

## 7. 未验证风险与限制

1. 未连接真实 PostgreSQL 18，未执行真实 advisory lock、行锁等待、commit/rollback 故障注入或 Audit Append 回滚验证。
2. 未运行浏览器、部署、真实 Host、E2E 或 production readiness 验证。
3. 未复跑 Lead 已完成的全量/race/contract/vet/build，只运行上述两个有界定向命令。
4. bootstrap 已确认环境密码读取时机、事务 lock 和原子写入方向正确；但本轮未以真实 PostgreSQL 验证并发唯一性和启动中断恢复。
5. 共享工作树含范围外未提交成果；结论仅绑定本文六个 SHA-256。

## 8. 交回

由 Lead/原作者决定修复方向与是否接受风险；本独立结论不替代人的采用、关闭或下一步授权。报告只写入
`.spec/wip/identity-auth/review-code-s4.1-r01.md`，未修改代码、design、tasks、Plan、manifest 或 Git 状态。
