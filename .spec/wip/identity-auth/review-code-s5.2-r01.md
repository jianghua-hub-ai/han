# Identity S5.2 独立代码评审 r01

- **评审日期**：2026-10-23
- **Reviewer**：独立非作者 Reviewer
- **objectType**：code
- **仓库 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **任务**：`identity-auth-S5.2`
- **结论**：**FAIL — 3 项 HIGH finding；另有 1 项合同方向待决定**

本结论只绑定第 1 节四个固定 SHA-256 对象。Reviewer 未修改被审代码、设计、任务、依赖、schema、migration 或其他报告；仅新建本报告。

## 1. 固定对象与修订

评审开始、报告写入前均复核分支、HEAD 与四个固定对象，结果与委托值一致：

| 文件 | SHA-256 |
|---|---|
| `internal/identity/handler.go` | `2CD9B251ECC6FEB6FF4806FA2A9F1E5D63D1A32EDBC4D7AF67598D525FF5C021` |
| `internal/identity/service.go` | `278F7DD18CF9974A6A1DAB5B4097BFBAEA74075B8F11217785CA7F1316416B59` |
| `internal/identity/administrator_create.go` | `0B93D2DD68D898C0280AD4C81998B55351E7C0DB78DEB56CD2A353C4F8194997` |
| `internal/identity/administrator_create_test.go` | `02AC8BE2347E36A770CD2D1E3AB04FA2F10D4402E7028870A7EA62BC95D9E6D0` |

共享工作树含大量其他人的未提交成果；本评审未回滚、覆盖、格式化或修改这些成果。

## 2. 目标、验收依据与边界

核对的主要合同：

1. 仅 initial admin 可创建其他管理员。
2. 信息提供声明必须使用 current 版本且 `declared=true`。
3. ProcessingGate 必须开放。
4. 创建操作满足作用域幂等；同键同摘要的 succeeded/failed 返回原结果。
5. SMTP 投递与数据库状态按 ADR-0011 fail closed；未受理不得留下可用管理员。
6. 新管理员必须完成改密和 current privacy acknowledgement 两个 Gate 才能解除受限状态。
7. 本卡不得修改 schema、migration、依赖或清单外实现文件。

使用的上游依据包括：

- `tasks.md:642-660`
- `design.md:1096-1106,1500-1527,1535-1560,2074-2088`
- `test-plan.md:270-280,378-380`
- `identity.md:175-191`
- `framework.md`
- 适用 Reviewer、code review、Go、安全、可观测性、追踪与硬编码规则。

允许依赖检查严格限定在 `internal/identity` 的直接消费者和 `idempotency`、`mailer`、`session`、`audit`、`security`、migration、production composition 的直接接缝；未扩展为全仓无界审查。

## 3. 两轮实际检查范围

### Round 1：固定四文件直接审查

逐路径检查：

- HTTP Handler 的主体、Trace、Idempotency-Key、严格 JSON、响应字段及双 Gate 入口；
- 创建协调器的授权、声明、ProcessingGate、作用域摘要、事务、SMTP、审计、失败/uncertain 与重放；
- PostgreSQL repository 的 pending/active 临时凭据、创建结果重建和 onboarding 行锁；
- privacy acknowledgement 与改密完成顺序；
- S5.2 测试是否以失败证据覆盖越权、声明过期、SMTP 未受理、提交不确定、幂等重放及双 Gate。

除 findings 外，确认：

- `declared=false`、过期声明、SMTP 未受理均不会提交可用管理员；
- SMTP accepted 后数据库最终提交失败进入 uncertain，事务中的 pending 用户/凭据不提交；
- 成功响应不返回临时密码或摘要；
- 新管理员初始保持 temporary credential、未完成 onboarding，两个 Gate 可按任一顺序完成，且 privacy Gate 要求 current notice；
- 创建作用域包含 actor、operation、归一化目标与 key digest，请求摘要覆盖结果相关字段。

### Round 2：直接消费者与上游契约审查

沿真实接缝检查：

`production router/composition → Handler → AdministratorCreateCoordinator → idempotency/mailer/repository/audit/session/security → migration`

重点核对：

- `idempotency.ResponseSnapshot` 的真实持久化 allowlist 与成功重放来源；
- `SMTPMailer.Send` 的 deadline 前置条件和 S5.2 调用上下文；
- production composition 是否构造新增 service/handler 并注册三条公共入口；
- `temporary_credentials`、双 Gate、幂等 JSON snapshot 的 schema/状态约束；
- 现有测试是否能发现上述消费者接缝问题。

## 4. Findings

### CRITICAL

无。

### HIGH

#### IDENTITY-S5.2-R01-F001 — 成功幂等记录未保存原响应快照，重放依赖可变业务状态

- **位置**
  - `internal/identity/administrator_create.go:434-445`
  - `internal/identity/administrator_create.go:909-955`
  - `internal/identity/administrator_create.go:1227-1256`
  - 直接接缝：`internal/idempotency/store.go:85-90`
  - 缺失回归：`internal/identity/administrator_create_test.go:60-157`
- **违反依据**
  - `design.md:1096-1103`：succeeded/failed 同键同摘要返回原快照。
  - `design.md:2074-2088`：成功/确定失败保存安全响应快照，重放返回原结果且不再执行副作用。
  - `identity.md:187`：相同作用域、相同摘要重放返回原结果。
  - `test-plan.md:275,379`：I-CRED-02 要求同摘要只执行一次并返回原结果，证据包含响应快照。
  - `internal/idempotency/store.go:85` 将 `ResponseSnapshot` 定义为可重放响应数据的完整 allowlist。
- **触发场景**
  1. initial admin 成功创建新管理员，幂等记录只保存 `Outcome` 和 `ResourceID`。
  2. 新管理员随后完成改密，临时凭据由 `active` 变为 `consumed`；或管理员的状态/字段发生后续合法变化。
  3. 客户端使用原 Idempotency-Key 和相同请求摘要重试。
  4. 重放代码不是返回原响应，而是重新查询当前 `users` 与 `temporary_credentials`；SQL 强制 `tc.state='active'`，并把两个 Gate 固定重建为 `true`。
  5. 改密后的合法状态会导致查询失败；其他后续状态变化会使响应偏离原 `201` 快照。
- **影响**
  - 已成功请求的安全重试可能变成持久化错误/503，而不是原成功结果。
  - 返回内容受请求完成后的可变状态影响，破坏作用域幂等和“原结果”合同。
  - 当前测试只证明归一化用户名可查询，没有覆盖成功后改密、privacy acknowledgement、停用等状态变化后的重放。
- **修复建议**
  - 成功完成幂等记录时保存完整、去敏、版本化的公共响应 allowlist；重放直接从该不可变快照恢复，不从可变业务表重建。
  - 若因数据治理不能保存完整响应，应设计独立不可变结果记录并明确保留期；不得继续把当前业务状态称为“原快照”。
  - 增加成功后改密/确认隐私/状态变化再重放的回归，逐字段断言与初次 `201` 相同且无新增邮件、用户、凭据或审计副作用。
- **范围风险**
  - 最小正确修复很可能需要修改 `internal/idempotency` 的 response allowlist 或增加受控结果结构，超出本卡四文件唯一写集。由 Lead 决定扩展授权或拆分任务；Reviewer 不替作者选择架构方案。

#### IDENTITY-S5.2-R01-F002 — 创建协调器未为真实 SMTP 调用建立明确 deadline

- **位置**
  - `internal/identity/administrator_create.go:387-392`
  - 直接接缝：`internal/mailer/smtp.go:78-89`
  - 直接证据：`internal/mailer/smtp_test.go:221-257`
  - 测试替身掩盖：`internal/identity/administrator_create_test.go:60-79,638-648`
- **违反依据**
  - `identity.md:191`：SMTP 投递必须设置明确超时，禁止无限等待。
  - `identity.md:180`：具体 SMTP 超时值保持 TBD，必须由目标环境测量后确定，不能编造。
  - no-hardcode 规则：环境相关超时不得在业务代码中拍脑袋写死。
- **触发场景**
  1. 未来 production composition 把真实 `SMTPMailer` 注入创建协调器。
  2. Handler 将通常不带 deadline 的 HTTP request context 原样传入 service。
  3. 协调器把同一 context 直接传给 `Mailer.Send`。
  4. 真实 `SMTPMailer.Send` 在拨号前检查 `ctx.Deadline()`；无 deadline 时立即返回 `ErrDeadlineRequired`，不会联系 SMTP。
  5. S5.2 mailer stub 丢弃 context，因此当前成功测试使用 `context.Background()` 仍通过。
- **影响**
  - 一旦使用真实 SMTP adapter，创建管理员会稳定失败为 `identity.email_delivery_failed`，主流程不可用。
  - 如果换用不强制 deadline 的 adapter，则违反 ADR 的“明确超时”要求并可能长期占用事务/连接。
- **修复建议**
  - 在 composition/config 中提供经过范围校验的 SMTP 操作预算，由创建协调器在发送前使用 `context.WithTimeout` 派生上下文并及时取消。
  - 不得在本文件硬编码未经采用的数值；测试同时覆盖无配置 fail closed、deadline 实际传入、取消/超时以及 SMTP 未受理零可用管理员。
- **范围风险**
  - 超时具体值仍为设计 TBD，且 production config/composition 不在本卡写集。需要 Lead 决定先补设计/采用值，还是授权独立配置与接线任务。

#### IDENTITY-S5.2-R01-F003 — 新增三个管理员入口未接入真实 production composition/router

- **位置**
  - 新入口：`internal/identity/handler.go:349-369,618-783`
  - 真实消费者：`cmd/han-server/main.go:846-946,949-988`
- **违反依据**
  - `design.md:785-786,1111-1112,1535-1560` 要求：
    - `GET /api/v1/admin/administrator-information-provision-notices/current`
    - `POST /api/v1/admin/administrators`
    - `POST /api/v1/users/me/privacy-acknowledgements`
  - `tasks.md:642-660` 的目标是实现创建其他管理员与双 Gate，并要求独立 Review 通过后才可关闭。
  - Closed-loop delivery 要求真实消费者可达，不能只证明孤立 Handler 方法存在。
- **触发场景**
  1. 启动当前 production business handler。
  2. 请求任一上述新增路径。
  3. `newProductionBusinessHandler` 只构造 login/password service 与 handler。
  4. `newLoginBusinessRouter` 只注册 `POST /api/v1/sessions` 和 `PUT /api/v1/users/me/password`；新增路径进入默认 JSON 404。
  5. 全仓 Go 消费者搜索没有发现固定文件之外对 `NewAdministratorHandler`、`NewAdministratorCreateService` 或 `NewAdministratorPrivacyService` 的生产调用；只有 S5.2 测试直接构造 service。
- **影响**
  - 用户无法从真实 HTTP 入口创建管理员、读取 current 信息提供声明或完成 privacy Gate。
  - 即使内部状态机单测通过，S5.2 的创建和双 Gate 闭环仍不可交付。
- **修复建议**
  - 在唯一 production composition root 构造并注入真实 Gate、notice、idempotency、mailer、audit、session、source、repository 与配置；按设计的 method/path、Origin/Session/CSRF/授权顺序注册三条路由。
  - 增加 production router/component tests：正确 method 可达；错误 method 返回稳定 405/Allow；未知路径稳定 404；授权、CSRF、Gate 及零副作用顺序明确。
- **范围风险**
  - `cmd/han-server/main.go` 不在 S5.2 四文件唯一写集，任务又明确遇到清单外修改需求应停止交 Lead。该 finding 不能由作者越权“顺手接线”；Lead 必须决定扩展本卡、建立独立接线任务或明确已有 owner。

### MEDIUM

无独立 MEDIUM finding。

### LOW

无独立 LOW finding。

## 5. 待决定

### IDENTITY-S5.2-R01-D001 — 动态前置条件与“稳定幂等重放”合同存在冲突

- **位置**
  - `internal/identity/administrator_create.go:281-337`
  - `internal/identity/administrator_create_test.go:287-344`
- **现状**
  - ProcessingGate、调用者资格、声明、current notice 和密码策略均在 `Idempotency.Begin` 之前检查。
  - 测试明确固化 Gate/声明失败不得触达幂等记录。
- **冲突依据**
  - `design.md:1511-1514,1527` 支持“先 Gate/声明、后幂等占位”的当前序列。
  - `design.md:1102,2081` 与 `identity.md:187` 又要求 succeeded/failed 同键同摘要返回原结果。
- **需要决定的场景**
  - 初次请求已经成功或稳定失败；随后 ProcessingGate 关闭、声明 current 版本变化、操作者资格变化或密码策略变化；客户端用原键和原摘要重试。
  - 当前代码返回新的前置条件错误，不能到达已保存的原结果。
- **方向选择**
  1. 稳定重放优先：先识别已存在的同键同摘要终态，再决定哪些动态 Gate 不重复执行；或
  2. 当前前置条件优先：明确修改“返回原结果”合同，列出可覆盖历史终态的动态条件和错误优先级。

这是契约/时序方向决定，不由 Reviewer 代替 Lead 或设计责任人拍板。决定后必须同步 design、测试矩阵、实现及重放安全分析。

## 6. 非阻断建议

1. 增加 service 级越权回归，而不只直接测试 `validateCreateAdministratorRequest`：断言非 initial admin 固定返回 403，并明确是否允许其先观察 ProcessingGate 状态。当前实现先调用 Gate，再验证 principal；设计未明确该错误优先级，若要求授权先行则需另行登记修复。
2. 对 F001/F002/F003 增加能在真实消费者边界失败的测试，避免 stub 或孤立构造器只匹配当前实现。

## 7. Reviewer 实际验证命令

以下命令由 Reviewer 在固定工作树执行：

| 命令 | 结果 | 说明 |
|---|---|---|
| `go test ./internal/identity -run "TestCreateAdministrator|TestAdministratorOnboarding" -count=1` | PASS | S5.2 定向测试 |
| `go test ./internal/identity -count=1` | PASS | identity 包测试 |
| `go test ./internal/idempotency -count=1` | PASS | 幂等内核测试 |
| `go test ./internal/mailer -run "TestSMTPMailerRequiresDeadline" -count=1` | PASS，但 `[no tests to run]` | 首次正则未命中，不能作为 deadline 验证证据 |
| `go test ./internal/mailer -run "Deadline" -count=1 -v` | PASS | 实际执行 missing-deadline 测试 |
| `go test ./cmd/han-server -run "Test.*Router|Test.*Production" -count=1` | PASS | 现有 router/production 测试；没有管理员路由用例，不能否定 F003 |
| `go vet ./internal/identity ./internal/idempotency ./internal/mailer ./internal/session` | PASS | 有界静态检查 |
| `gofmt -l internal/identity/handler.go internal/identity/service.go internal/identity/administrator_create.go internal/identity/administrator_create_test.go` | PASS（无输出） | 仅检查，没有格式化文件 |
| `rg -n --glob '*.go' 'NewAdministratorHandler\|NewAdministratorCreateService\|NewAdministratorPrivacyService\|HandleCreateAdministrator\|HandleAdministratorPrivacy' .` | 完成 | 生产消费者静态核对；构造调用只出现在 S5.2 测试 |
| `git branch --show-current; git rev-parse HEAD; Get-FileHash ...; git status --short --branch` | 完成 | 固定对象、HEAD、共享工作树与越界检查 |

Lead 已提供全仓 `gofmt`、`go mod verify`、`go vet`、`go build`、contract、race、test 通过证据；本报告将其记为外部已有证据，不冒充 Reviewer 独立复跑。

## 8. 未验证风险与限制

1. 未连接真实 PostgreSQL 18，未执行真实并发锁、幂等终态持久化、SMTP accepted 后 commit failure 或崩溃恢复。
2. 未连接真实 SMTP/TLS 服务；F002 由真实 adapter 合同及其单元测试证明，未发外部邮件。
3. 未启动真实 HTTP Host、浏览器或部署环境，未执行 CSRF、Origin、Cookie 和三条新增路由的真实 E2E。
4. 未独立复跑 Lead 已执行的全仓 build/contract/race/test；本轮只执行第 7 节有界验证。
5. 未审查 S5.3、S10.1、S11.1 等后续凭据专项，也未扩展到无关模块。
6. 工作树存在大量范围外未提交成果；结论只绑定第 1 节四个固定哈希及第 10 节读取依据，不能外推为整个仓库通过或失败。

## 9. Code Review Gate 与复审条件

### Code Review Gate：FAIL

F001 破坏成功终态的原结果重放；F002 使真实 SMTP adapter 无 deadline 时必然拒绝调用；F003 使新增能力在 production router 中不可达。三项均影响核心验收或真实交付，S5.2 不应关闭。

复审条件：

1. Lead 先对 F001/F002/F003 的清单外修改和 D001 合同方向作出明确授权/决定。
2. 原作者提供固定的新修订、逐项 finding 回应与相关回归：
   - F001：不可变安全响应快照及状态变化后重放；
   - F002：经采用的 SMTP budget、deadline 传递和超时/取消；
   - F003：production composition/router 可达性及 404/405/授权/CSRF/Gate 顺序。
3. 重新运行 S5.2 定向测试、涉及的直接包、race/vet/build/contract/全仓测试及 owner diff-check。
4. 由独立非作者 Reviewer 对原 F001～F003、D001 决定、修复修订和相关消费者回归定点复审。

本独立结论不替代人的风险接受、架构选择、任务范围扩展或后续授权。

## 10. 实际读取文件与 SHA-256

### 10.1 入口、Role、Skill、Rule、Profile

| 文件 | SHA-256 |
|---|---|
| `AGENTS.md` | `BD5360EBBEC516AC635094637FC51B4FFB647D02B42982010EF84D3531D31164` |
| `.codex/skills/add-route-work/SKILL.md` | `14ED7A82A4DEA00B92E549840AB0A563A728D657CB60C2745248B31663076DF7` |
| `.codex/agents/add-reviewer.toml` | `549202AE65D455056279D40D55C539DCCB2FA4F51AB060D7BB77891C5E1C7054` |
| `.codex/skills/add-code-review/SKILL.md` | `5D9DF5E173CFD523A444B0DEDDDB141D7DDD6C6E1EB2130D1AA6A89F9F8E79F4` |
| `.codex/skills/add-code-review/references/产品安全红线.md` | `994EABF6BD9076EBA67F497768AD298D7AEFEDA90D901DB39B6AABC263B405FA` |
| `.codex/guidance/code-review-checklist.md` | `192FA2344908E4A96D16DAC065479CC8F20C00F6EF4DC399FFE5A22470EBA72F` |
| `.codex/guidance/abstraction-principles.md` | `3FF17453E0D8A8E8B36834AFB2588C258A2F7694F3921E84C87E44B7E96F7465` |
| `.codex/guidance/coding-standards.md` | `D6B610323FDC793A6AA0A873B3CF760B7DA5A260FAEF2874D6599882D3371142` |
| `.codex/guidance/no-hardcode-policy.md` | `B52137A2AED299748D76B55A184C5EE60ADE7E0B540588E0F5A69028F3F4AB6F` |
| `.codex/guidance/observability-standards.md` | `D88E1CA42C2BF3B46FD4CEEC41772257686E5A6CB319652ED79EB7C4C861B8FD` |
| `.codex/guidance/tracing-standards.md` | `86002304BDF8AA58B44235EBD3A35520CF915A428ACB02B64512E6B0725AF9A6` |
| `.codex/profiles/go-service/README.md` | `4D221CAD44FC8CF225A47116D9CA1993405DFA018BC3314DAAF6C98AB71A6D88` |
| `.codex/profiles/go-service/rules/coding-standards.md` | `804541ECEB08D727E96217D2D38BE1B4D2A5E4D55905799790A21963F8BA83E3` |
| `.codex/profiles/go-service/rules/no-hardcode-policy.md` | `7A0F2268440F520D06C95EDF94C3E3F0DDED63731A8F582FCF495F2A200CC4A3` |
| `.codex/profiles/go-service/rules/observability-standards.md` | `F0C8BCE5FC61E274EE4833DD268E605C32516D2A4C578CD34AD793918A60025B` |
| `.codex/profiles/go-service/rules/security-coding-standards.md` | `8D5621678A5798E53D52CF0EA5D96E285A21477DFCC3FEC3E5CEDEDFC20C97A1` |
| `.codex/profiles/go-service/rules/tracing-standards.md` | `B9D7EC43EED4634EBBB3CC0378768E84BC1EB60765BA4B2BD14B959194A05338` |

### 10.2 上游任务、设计、测试与决策

| 文件 | SHA-256 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `A3DBE8625EA1879711740F7DE9E63C423C9B597CA163EF0B63A86F4914789400` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `977435C8391E4A21C070C359AFBB6F143EC84E4CA3EE19CBEC7FC5FCC9C0E6B0` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `C49D87ED526D897CBF18CDE7E338F44A845E85CC1E0412A90E2A9169367B2D88` |
| `.add/project/knowledge/decisions/identity.md` | `4A1141B55D882A76DB332DD5E75CCA640A2C0CA7121980910477BC408A42BC45` |
| `.add/project/knowledge/decisions/framework.md` | `8D381C9D753242B47E1B6CF0C269228AD4ED591A4605CA4F119CAF03DB4E0AC7` |

### 10.3 固定代码与有界直接接缝

| 文件 | SHA-256 |
|---|---|
| `internal/identity/handler.go` | `2CD9B251ECC6FEB6FF4806FA2A9F1E5D63D1A32EDBC4D7AF67598D525FF5C021` |
| `internal/identity/service.go` | `278F7DD18CF9974A6A1DAB5B4097BFBAEA74075B8F11217785CA7F1316416B59` |
| `internal/identity/administrator_create.go` | `0B93D2DD68D898C0280AD4C81998B55351E7C0DB78DEB56CD2A353C4F8194997` |
| `internal/identity/administrator_create_test.go` | `02AC8BE2347E36A770CD2D1E3AB04FA2F10D4402E7028870A7EA62BC95D9E6D0` |
| `internal/identity/model.go` | `72BAE00D2B6862D5CA154F7BFB7DB8AF7FA84A7A60F86852699F0C94DAA3BE07` |
| `internal/identity/repository.go` | `9FC7F99D58E432E1807DE06C6B705A00CC736F99141EC3F95AF234EBA44DACE3` |
| `internal/idempotency/store.go` | `65F888E7DD7A73301C6EC7F626E57EA4C4922A1411EB6359B8D8240B9E2FFF80` |
| `internal/idempotency/store_test.go` | `739238655091FB540AB31DE34B10D124DE9A2E1C58FB6069817EDD4386BFB96A` |
| `internal/mailer/smtp.go` | `3B66245D0CDF4A7FB1E878EF68CE04342A32DABF0214AD1F5B8871F32C444F4F` |
| `internal/mailer/smtp_test.go` | `08FC9DF16CEBC7A1A47E293647EA8B6E2F0B5A2EED0154791B019F7FCEA9412E` |
| `internal/session/store.go` | `0E7D65194ABE283E0E01BE67C35C170CFD41EA07A38AA70A4C45EBA5FD482347` |
| `internal/audit/store.go` | `091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F` |
| `internal/security/password.go` | `59F3E886F7D8DD7409DEF141EB00415E6976B72C3F4450685E2F786E87259507` |
| `migrations/0002_identity.sql` | `F4EC1005EA72B1774DF5E370C5ABA57CA3ED7A0B13099DBBE2D16AA66AEB9D50` |
| `migrations/0004_idempotency.sql` | `4794816AC2FDA4AD12522F2F324448A6F9EFB2E9E39677A595400E68148424A9` |
| `cmd/han-server/main.go` | `5DBACCB74EF15F53BCDCC1845ADE8B66327163F8817A84C708489C0082E61988` |
| `cmd/han-server/main_test.go` | `D83244E9B2F8277FDE485D7FA2DCD14ADB570F9A3FD53361DB48550C903DDA77` |

