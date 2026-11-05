# Identity design r29 / S5.2 独立 Design Review

- **评审日期**：2026-10-23
- **Reviewer**：Faraday，独立非作者 Reviewer
- **objectType**：design
- **仓库 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **固定对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- **revision / status**：`identity-auth-design-ir01-r29` / `draft`
- **固定 SHA-256**：`F27FDEDBF488AAC58933BB88AC77B25633DC78282BB2ACCF0B4747EB5AF880CF`
- **结论**：**FAIL — 1 项 HIGH、2 项 MEDIUM**

本结论只绑定上述固定设计对象。Reviewer 未修改设计、test-plan、tasks/plan、代码、ADR、boundary、migration 或旧报告；仅新建本报告。

## 1. 目标、依据与边界

本轮核对 r29 是否完整、无矛盾、可实施，并逐项回应：

- `IDENTITY-S5.2-R01-F001`
- `IDENTITY-S5.2-R01-F002`
- `IDENTITY-S5.2-R01-F003`
- `IDENTITY-S5.2-R01-D001`

采用值按用户输入固定，不重新选择：active initial admin 授权优先；授权后的稳定终态重放优先于动态 Gate/notice/密码策略；快照版本为 `identity-administrator-create-response/v1`；SMTP budget 必填、无默认、部署基线 `10s`、合法闭区间 `1s..60s`；配置在 DB、listener、composition、route 前 fail closed；唯一 production composition 注册三条 S5.2 路由；S5.2 不宣称真实 SMTP/TLS、容量或 production readiness。

允许依赖检查限定在 identity boundary、identity/framework ADR、现役 test-plan、S5.2 tasks/plan、原 code review，以及 `internal/identity`、`internal/idempotency`、`internal/mailer`、`cmd/han-server`、平台错误/Trace 和直接 migration 接缝。未扩展为全仓无界审查。

## 2. 两轮实际检查范围

### Round 1：设计直接合同

检查了：

1. active initial admin 授权、只读幂等分类、摘要冲突、五种持久状态和并发 `Begin` 重分类的顺序；
2. succeeded/稳定 failed 的版本化公共响应快照、坏快照 fail closed、零副作用和兼容规则；
3. `HAN_SMTP_OPERATION_BUDGET` 的解析、范围、启动失败顺序、注入、较早 deadline、parent cancellation 与及时 cancel；
4. 三条 S5.2 路由、唯一 production composition、method/path、中间件、404/405 与资源 ownership；
5. S5.2/S5.3 验证分工、不宣称内容和 §9.3 实现影响面；
6. r29 对原 F001～F003/D001 的追溯表、验证落点和停止条件。

除下述 findings 外，r29 已明确：

- transport/Session/CSRF 后必须先重新读取主体并确认其仍是 active initial admin；授权失败不得读取幂等历史；
- 授权后先只读分类；同摘要 succeeded/稳定 failed 重放高于 ProcessingGate、current notice/声明版本和 PasswordPolicy；不同摘要保持冲突；
- processing 返回 in-progress，uncertain/recovery_review_required 返回 operation-uncertain；普通 HTTP 重试不得转换恢复状态；
- 重放不查询可变 User/TemporaryCredential，且不发送邮件、不创建用户/凭据/占位、不追加审计；
- SMTP budget 的必填、无默认、`1s..60s`、启动前 fail closed、较早 deadline、及时 cancel 和未受理零可用账号合同；
- 唯一 `cmd/han-server` composition 注册 current notice、administrator create、privacy acknowledgement 三条路由；
- S5.2 只验证配置、接线、deadline/cancel、稳定重放和零副作用，真实 SMTP/TLS 留给 S5.3。

### Round 2：真实消费者、配置、启动、测试与持久化接缝

沿以下链路有界核对：

`production startup/router → TraceMiddleware/platform response → Administrator Handler/Coordinator → idempotency Store/DB CHECK → SMTP adapter → tests`

重点事实：

- `internal/idempotency/store.go` 当前 `ResponseSnapshot` 只允许 `outcome/resource_id/error_code`，但代码层可按 r29 改造；
- `migrations/0004_idempotency.sql:31-53` 在数据库层再次封闭相同三个字段，并拒绝其他 JSON key；
- `platform.TraceMiddleware` 为每个进入请求生成当前 trace 并写 `X-Trace-ID`；`platform.WriteError` 从当前请求 context 取得同一 trace 写入 header/body；
- 管理员成功公共响应当前为八个字段；
- `SMTPMailer.Send` 已要求 context deadline，并消费 `DialContext`、连接 deadline 与 TLS handshake cancellation；
- `cmd/han-server` 已有可复用的 trim + Go duration parser、前置配置加载、唯一 factory、双 listener ownership 和 cleanup 骨架。

## 3. Findings

### CRITICAL

无。

### HIGH

#### IDENTITY-DESIGN-R29-S5.2-R01-F001 — 新快照结构被现役数据库 CHECK 拒绝，但 r29 同时禁止 migration

- **位置**
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:1133,2006-2011,2136-2137,2718,2872-2879`
  - `migrations/0004_idempotency.sql:31-53`
  - `internal/idempotency/store.go:85-90,623-677`
- **现状**
  - r29 要求 v1 快照包含 `schema_version,http_status,result_code,body`，并把它作为 succeeded/稳定 failed 重放的唯一真相源。
  - 现役数据库约束要求 `response_snapshot` 必有字符串 `outcome`，只允许可选字符串 `resource_id/error_code`，且 `response_snapshot - ARRAY['outcome','resource_id','error_code']` 必须为空对象。
  - r29 §9.3 又断言继续使用现有 JSONB、不需要 schema/migration，并未把 migration 纳入返工影响面。
- **违反依据**
  - add-code-review 的 design 分支要求字段、持久化、兼容与实现映射可落地。
  - design-writing-style §J 要求结构设计给出必要字段、状态约束和可实施合同，不能让实现者猜测。
  - r29 自身要求坏/未知/额外/缺失字段 fail closed，不能通过弱化数据库约束绕过。
- **触发场景**
  1. 实现按 r29 把首次成功或稳定失败编码为包含 `schema_version/http_status/result_code/body` 的 JSON。
  2. `IdempotencyStore.Complete` 更新 `response_snapshot`。
  3. PostgreSQL 的既有 CHECK 因缺少 `outcome` 或出现新 key 而拒绝该行。
- **影响**
  - `IDENTITY-S5.2-R01-F001` 在声明的无 migration 范围内不可实施。
  - 成功路径可能在 SMTP 已受理后因快照持久化失败而进入回滚/不确定窗口；稳定失败也无法按新 v1 结构持久化和重放。
  - 仅修改 Go allowlist 或测试不能修复数据库真实消费者。
- **建议**
  - 由 Lead/设计责任人明确扩大范围并提供前向 migration，更新现有 CHECK 以精确承载 v1；或提出另一条能真实通过现役数据库约束且仍保存完整公共响应的持久化方案。
  - 不得把 JSONB 列存在等同于其约束允许任意 JSON，也不得要求 Backend-Dev 越权修改 migration。
- **验证**
  - 修订设计必须列出 migration/兼容/回退/旧记录读取策略和文件 owner。
  - PostgreSQL 18 集成测试应证明成功与稳定失败 v1 可写、可读、额外/缺失/错误类型被拒绝，旧快照的处理策略明确。

### MEDIUM

#### IDENTITY-DESIGN-R29-S5.2-R01-F002 — 稳定失败“原 trace_id 重放”与每请求 Trace 合同相互冲突

- **位置**
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:920,1133-1134,2716,2718,2750,2766`
  - `internal/platform/http.go:18-42`
  - `internal/platform/errors.go:207-241`
  - `internal/platform/errors_test.go:590-605`
- **现状**
  - r29 要求稳定失败快照保存并逐字段重放首次 body 的原 `trace_id`，且保持首次 body 字段和值。
  - 同一设计和真实平台接缝要求每个请求由最外层 TraceMiddleware 建立可信 trace，并让响应 header/body 与当前请求 trace 一致。
- **触发场景**
  1. 首次稳定失败响应使用 trace A 并写入快照。
  2. 重放请求进入 TraceMiddleware，生成当前 trace B 并写入 context/header。
  3. 若原样返回快照，body 为 A、header/context/log 为 B；若改写 body 为 B，则不再是逐字段原响应；若把 header/context 改成 A，则复用历史 trace 并破坏当前请求关联。
- **影响**
  - 现有合同没有一种实现能同时满足“不可变原 body”和“当前请求 trace header/body/context 一致”。
  - 重放请求可能无法通过 trace 关联到当前日志，或迫使实现者自行改变平台追踪语义。
- **建议**
  - 优先把 `trace_id` 定义为当前请求元数据，不纳入稳定业务快照；快照保存稳定 status/code/message/field_errors，Handler 使用当前 context trace 生成本次 envelope。
  - 若坚持重放原 trace，必须明确 header/context/log/body 的统一规则及其对 tracing 合同的影响，不能留给实现者选择。
- **验证**
  - 增加首次失败 trace A、重放 trace B 的测试；明确断言重放 header、body、context 和日志应使用哪个值，并证明四者一致。

#### IDENTITY-DESIGN-R29-S5.2-R01-F003 — 封闭 v1 schema 的精确字段集合仍自相矛盾

- **位置**
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:1133,1591-1595,2011,2136-2137,2718,2766`
  - `internal/identity/handler.go:1020-1029`
- **现状**
  - r29 把 v1 定义为拒绝额外字段、缺失字段和类型漂移的封闭 schema。
  - 同一段落却称外层“至少保存”四个字段，没有固定是否允许其他字段；成功 body 又称“九个字段”，而 §3.3.6 示例和真实响应结构均只有八个字段：
    `id,username,role,status,must_change_password,must_acknowledge_privacy,temporary_credential_expires_at,delivery_status`。
- **触发场景**
  - Test-Engineer 编写 strict decoder/fixture，或 Backend-Dev 实现 exact allowlist 时，无法确定第九个字段以及外层额外字段是否合法。
- **影响**
  - 不同实现/测试可能产生不兼容的同名 v1；坏快照 fail-closed 测试没有唯一预期。
- **建议**
  - 给出 v1 外层和 success/failed body 的完整 JSON schema 或等价字段表，逐项列字段名、类型、必填/可选；删除“至少”和错误字段数量。
  - 与 F002 一并明确失败 body 的 `trace_id` 归属。
- **验证**
  - 固定 canonical success/failed fixtures；测试精确接受合法对象并拒绝每一种 extra/missing/type drift。

### LOW

无独立 LOW finding。

## 4. 原 F001～F003/D001 映射

| 原项 | r29 覆盖结论 | 本轮结果 |
|---|---|---|
| `IDENTITY-S5.2-R01-F001` | 已补授权后稳定终态重放、不可变版本化快照、坏快照 fail closed、零副作用、状态变化后逐字段回归 | **未关闭**：数据库 CHECK 与“无 migration”冲突；失败 trace 语义和封闭 schema 仍不唯一 |
| `IDENTITY-S5.2-R01-F002` | 已固定配置键、必填无默认、`10s` 基线、`1s..60s`、启动前 fail closed、composition 注入、较早 deadline、parent cancel 和 S5.2/S5.3 分界 | **设计覆盖充分**，未发现独立阻断项；仍待实现与测试证据 |
| `IDENTITY-S5.2-R01-F003` | 已固定唯一 production composition、三条路径、正确 method、JSON 404/405、middleware/资源 ownership 和 production consumer 测试 | **设计覆盖充分**，未发现独立阻断项；仍待实现与测试证据 |
| `IDENTITY-S5.2-R01-D001` | 已固定 active initial admin 授权 → 只读分类 → 冲突/稳定重放 → 仅新执行动态 Gate/notice/policy → 并发 Begin 重分类 | **方向已关闭**；其 F001 快照持久化与 trace 细节仍受本轮 findings 阻断 |

## 5. 待决定与非阻断建议

### 待决定

1. **持久化范围**：现役 DB CHECK 无法承载 r29 v1。Lead/设计责任人需决定并授权前向 migration 及相应 task/write set；在此之前不得把 F001 交 Backend-Dev 实施。
2. **失败重放 trace 语义**：决定 `trace_id` 是历史业务快照字段还是当前请求关联元数据。Reviewer 建议使用当前请求 trace，并从稳定快照排除它。

### 非阻断建议

1. 幂等记录 retention 与旧 HMAC key 保留期仍为 TBD。它不阻断本次局部设计方向，但阻断生产治理、密钥退役和 production readiness。
2. 现役 test-plan/tasks/plan 仍主要引用 r28/旧写集。Design finding 关闭后，由 Test-Engineer/Task-Planner 按 r29 新 revision 重新对账，不应把旧范围直接视为已同步。
3. `internal/identity/handler_test.go` 当前不存在；若下游采用 r29 §9.3 的新增测试文件，Task-Planner 应显式登记所有权和新增文件，而不是把它描述为既有文件。

## 6. Design Review Gate 与下游可承接范围

### Design Review Gate：FAIL

阻断理由：

- F001 的新快照结构不能通过现役数据库约束，而 r29 明确排除 schema/migration；
- 稳定失败重放的 trace 合同互相冲突；
- 封闭 v1 的精确字段集合尚未唯一。

**下游范围**：

- Test-Engineer/Task-Planner 可把 F002、F003 和 D001 的顺序合同作为非最终规划输入，但不得据此关闭或批准整张 S5.2 返工计划。
- Backend-Dev 不应开始 F001 快照持久化，也不应以本报告替代写范围授权；整合后的 S5.2 implementation handoff 暂不成立。
- 本结论不否定 r29 已正确补齐的 SMTP、production composition 和动态 Gate 顺序，只表示它们尚不能与未解决的 F001 形成完整可实施设计。

## 7. 复审条件

1. 提供新的固定 design revision/SHA，并逐项回应本报告 F001～F003。
2. 对数据库 CHECK 给出获授权的可实施方案；若需要 migration，先同步 tasks/plan、测试范围、兼容与回退合同。
3. 明确失败 replay 的 trace header/body/context/log 规则。
4. 固定 v1 外层、成功 body、失败 body 的精确字段、类型、必填/可选和 canonical fixtures。
5. Test-Engineer/Task-Planner 更新 r29 后续 revision 的测试与任务映射后，由独立非作者 Reviewer 定点复审原 code review F001～F003/D001 及本报告 findings。

## 8. Reviewer 实际命令与结果

| 命令/检查 | 结果 |
|---|---|
| `git rev-parse --abbrev-ref HEAD; git rev-parse HEAD` | `main` / 固定 HEAD 一致 |
| `Get-FileHash -Algorithm SHA256 design.md` | 固定 SHA 一致；写报告前再次复核无漂移 |
| `Test-Path review-design-r29-s5.2-r01.md` | 写入前为 `False`，无报告冲突 |
| `Get-Content -Encoding utf8` / `rg -n -C ...` | 完成设计、原 review、上游依据和直接消费者的两轮有界检查 |
| `Get-FileHash` | 记录第 10 节实际读取对象 |
| `git diff --check -- design.md review-design-r29-s5.2-r01.md` | 写入前无 whitespace error；Git 仅提示 design 工作副本未来可能 LF→CRLF，本轮未触碰 design |
| `git status --short --branch` | 确认共享工作树有大量他人成果；本轮未回滚、覆盖或格式化 |

本轮是 design review，未运行 Go test/race/vet/build、PostgreSQL 或 SMTP/TLS；Lead 的既有验证不冒充 Reviewer 本轮执行证据。

## 9. 未验证风险与限制

1. 未执行 PostgreSQL 18 写入实验；F001 由现役 migration CHECK 与拟议 JSON 字段的静态直接冲突判定。
2. 未连接真实 SMTP/TLS；S5.2/S5.3 分界仅核设计与 adapter 合同。
3. 未启动真实 HTTP Host，未执行三条路由、Session、CSRF、404/405 或资源关闭 E2E。
4. 未审查 S5.2 允许范围外的全仓消费者，不对整个 identity r29 或整个仓库给 PASS/FAIL。
5. 共享工作树含大量未提交成果；结论只绑定固定 design SHA 与第 10 节读取依据。

## 10. 实际读取文件与 SHA-256

### 10.1 入口、Role、Skill、Rule、Profile

| 文件 | SHA-256 |
|---|---|
| `AGENTS.md` | `BD5360EBBEC516AC635094637FC51B4FFB647D02B42982010EF84D3531D31164` |
| `.codex/skills/add-route-work/SKILL.md` | `14ED7A82A4DEA00B92E549840AB0A563A728D657CB60C2745248B31663076DF7` |
| `.codex/agents/add-reviewer.toml` | `549202AE65D455056279D40D55C539DCCB2FA4F51AB060D7BB77891C5E1C7054` |
| `.codex/skills/add-code-review/SKILL.md` | `5D9DF5E173CFD523A444B0DEDDDB141D7DDD6C6E1EB2130D1AA6A89F9F8E79F4` |
| `.codex/guidance/code-review-checklist.md` | `192FA2344908E4A96D16DAC065479CC8F20C00F6EF4DC399FFE5A22470EBA72F` |
| `.codex/guidance/design-doc-writing-style.md` | `1F182C1C8FACA6EA48FF4C827E1C0EAA953D6A1BF3D65F15D4222FE88E12F23B` |
| `.codex/guidance/abstraction-principles.md` | `3FF17453E0D8A8E8B36834AFB2588C258A2F7694F3921E84C87E44B7E96F7465` |
| `.codex/guidance/decisions-guide.md` | `0E6532AE32A13F92CD295CA26515B023C9BC8A94B4FF279EFD94EA81F75D41B5` |
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

### 10.2 固定对象与上游依据

| 文件 | SHA-256 |
|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `F27FDEDBF488AAC58933BB88AC77B25633DC78282BB2ACCF0B4747EB5AF880CF` |
| `.spec/wip/identity-auth/review-code-s5.2-r01.md` | `69645C38F55584FB8ABEB96E6CB8813F9E9357E10066D76CFF04A898D162D63E` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `C49D87ED526D897CBF18CDE7E338F44A845E85CC1E0412A90E2A9169367B2D88` |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `335CC44131B542E5E3EEFCCBABBA443160AC8B0B240FE2BBAB0E849D50992591` |
| `.add/project/knowledge/decisions/identity.md` | `4A1141B55D882A76DB332DD5E75CCA640A2C0CA7121980910477BC408A42BC45` |
| `.add/project/knowledge/decisions/framework.md` | `8D381C9D753242B47E1B6CF0C269228AD4ED591A4605CA4F119CAF03DB4E0AC7` |
| `.add/project/knowledge/tasks/tasks.md` | `E64F0B17EB4F07C7633ACA5A6EFB27A6C2F4CC2DCBBD5EE365D482722840E275` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `1E66CA07C446B0293F1C5A33F8FC583772B8EC7C3A815921BA05F6948A2B1715` |

### 10.3 有界真实消费者与直接接缝

| 文件 | SHA-256 |
|---|---|
| `internal/identity/administrator_create.go` | `0B93D2DD68D898C0280AD4C81998B55351E7C0DB78DEB56CD2A353C4F8194997` |
| `internal/identity/administrator_create_test.go` | `02AC8BE2347E36A770CD2D1E3AB04FA2F10D4402E7028870A7EA62BC95D9E6D0` |
| `internal/identity/handler.go` | `2CD9B251ECC6FEB6FF4806FA2A9F1E5D63D1A32EDBC4D7AF67598D525FF5C021` |
| `internal/idempotency/store.go` | `65F888E7DD7A73301C6EC7F626E57EA4C4922A1411EB6359B8D8240B9E2FFF80` |
| `internal/idempotency/store_test.go` | `739238655091FB540AB31DE34B10D124DE9A2E1C58FB6069817EDD4386BFB96A` |
| `migrations/0004_idempotency.sql` | `4794816AC2FDA4AD12522F2F324448A6F9EFB2E9E39677A595400E68148424A9` |
| `internal/mailer/smtp.go` | `3B66245D0CDF4A7FB1E878EF68CE04342A32DABF0214AD1F5B8871F32C444F4F` |
| `internal/mailer/smtp_test.go` | `08FC9DF16CEBC7A1A47E293647EA8B6E2F0B5A2EED0154791B019F7FCEA9412E` |
| `cmd/han-server/main.go` | `5DBACCB74EF15F53BCDCC1845ADE8B66327163F8817A84C708489C0082E61988` |
| `cmd/han-server/main_test.go` | `D83244E9B2F8277FDE485D7FA2DCD14ADB570F9A3FD53361DB48550C903DDA77` |
| `internal/platform/http.go` | `0890069CB559C7B8454A242B46736FC23D542B0D45E350B0D035C7012BB710C4` |
| `internal/platform/errors.go` | `99B1094A6C15A6E86E45B43928F8AC1B4F3A4EE810857BE598E9AA4336D968E3` |
| `internal/platform/errors_test.go` | `5C8CEA67B738BE1AA11FB4AE206907604BEE7CC2CC0C646891CF4EF35A6D9237` |
| `internal/platform/logging.go` | `F362D36954DF0119D5DC25040640C796069CC8E178A1D92B828C6D6149CF30A9` |

本独立结论不替代 Lead/用户对 migration 范围、风险接受、任务授权或后续实现的决定。
