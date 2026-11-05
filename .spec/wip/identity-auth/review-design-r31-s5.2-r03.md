# Identity S5.2 Design r31 定点独立复审

- **Reviewer**：Faraday（非作者）
- **日期**：2026-10-23
- **objectType**：design
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **固定对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- **revision / status**：`identity-auth-design-ir01-r31` / `draft`
- **固定 SHA-256**：`CBF29B80A33C08455151D8E7621C1A115F3D16185F090A5371ACF1E29D6A55FC`
- **原报告**：`.spec/wip/identity-auth/review-design-r30-s5.2-r02.md`
- **原 finding**：`IDENTITY-DESIGN-R30-S5.2-R02-F001`
- **结论**：原 finding **CLOSED**；无新增阻断 finding；**Design Review Gate PASS**

本结论仅绑定上述固定设计对象。Reviewer 未修改设计、代码、migration、test-plan、tasks/plan、ADR、boundary、旧报告或其他文件；仅新建本报告。

## 1. 两轮实际复审范围

### Round 1：r31 最小修订直接合同

1. 核对 identifier-conflict canonical fixture 的 HTTP status、result code、message 和 `field_errors` 精确值。
2. 核对外层四字段与 failed body 三字段的封闭集合、必填性、JSON 类型及映射一致性。
3. 核对非空、缺失、null、类型漂移、冲突字段披露的拒绝规则是否在正文、schema、migration、codec、fixtures 和测试矩阵中一致。
4. 全文检索 r31 的追溯、默认假设、兼容性、影响面及附录响应，检查同一规则是否出现相互矛盾表述。

### Round 2：真实消费者与后续接缝

有界核对：

- 现役管理员创建 unique-violation 映射和 platform error registry；
- `migrations/0004_idempotency.sql`、migration runner、现役 legacy idempotency store 及测试；
- identity Handler、AdministratorCreateCoordinator 及测试；
- platform trace/error/logging、SMTP adapter、production composition/router 及测试；
- identity boundary、identity/framework ADR、现役 test-plan、tasks 与 batch plan。

同时检查 r31 是否直接回退已关闭的 trace A→B、legacy/v1 双读、SMTP operation budget、三条 production route 及 S5.2/S5.3 分界。未扩展为全仓审查。

## 2. 原 finding 复审

### `IDENTITY-DESIGN-R30-S5.2-R02-F001` — CLOSED

**原问题**：`identity.identifier_conflict` 缺少唯一 canonical `field_errors`，导致数据库 CHECK、Go codec/writer 和 fixtures 无法单义实施。

**r31 关闭证据**：

1. design:1629-1644 给出完整 canonical snapshot，精确固定：
   - `http_status = 409`
   - `result_code/body.code = identity.identifier_conflict`
   - `message = 身份标识已被使用`
   - `field_errors = []`
2. design:1644、1661、2085-2087 明确 `field_errors` 必填且只能为空数组；非空、缺失、null、object/string 等类型漂移均拒绝，并禁止披露 username、email、phone 或任何冲突字段。
3. design:2089-2100 将同一规则同步到拟新增 `0007` CHECK、strict legacy/v1 reader、Go decoder/writer、升级/回退约束及 PostgreSQL 18 拒绝矩阵。
4. design:2810-2812、2966-2971 将同一 canonical fixture 和拒绝矩阵落实到 Go fixtures、PG18 fixtures、idempotency/Handler 测试与后续 owner。
5. design:2860、3100、3167 的兼容性、追溯和整改响应与正文一致，未发现第二种合法表示。
6. 现役 `internal/identity/administrator_create.go:1122-1126` 将所有 unique violation 统一映射为 `identity.identifier_conflict`；`internal/platform/errors.go:99` 固定 409 与“身份标识已被使用”，均不提供冲突字段，与空数组和最小披露合同相容。

**结论**：触发场景、唯一 wire/persistence 值、拒绝边界、实现 owner 和测试入口均已单义，原 finding 关闭。

## 3. 新 Findings

### CRITICAL / HIGH / MEDIUM / LOW

**无。**

r31 是对原唯一 finding 的局部收口，未发现修订引入新的合同矛盾、安全披露、兼容、owner 或实施边界问题。原 r29 F001/F002/F003 保持 **CLOSED**，无直接回归证据，不重开。

## 4. 最小修订回归检查

| 检查项 | 结果 |
|---|---|
| trace A→重放 trace B | 未回退：快照继续禁止 `trace_id`；header/body/context/log 使用本次请求 trace；稳定业务字段不经当前 registry 重算 |
| legacy/v1 | 未回退：`0004` 不改，`0007` 前向扩宽；legacy/v1 严格互斥双读；管理员新写只写 v1 |
| SMTP budget | 未回退：必填、无默认、部署基线 `10s`、合法闭区间 `1s..60s`、较早 deadline 与及时 cancel |
| production routes | 未回退：唯一 production composition 注册 current notice、administrator create、privacy acknowledgement 三条路由 |
| S5.2/S5.3 | 未回退：S5.2 只证明 migration/config/wiring/deadline/cancel/稳定重放/零副作用；真实 SMTP/TLS 留给 S5.3 |
| 公共合同与范围 | 未新增公共错误、依赖、列、ADR、boundary 或 architecture 变更；不启动 B6B/S3.6/S2.2，不宣称 production readiness |

## 5. 待决定与非阻断建议

### 待决定

无新的架构、边界或 ADR 决定。

### 非阻断实施注意

现役 `platform.Envelope.FieldErrors` 使用 `json:"field_errors,omitempty"`，现役 `platform.WriteError` 会省略空数组。r31 已要求 `AdministratorCreateResponseMaterializer` 使用稳定快照直接物化首次/重放失败响应，而不通过当前 registry/通用 writer 重算；后续实现和测试应保持该专用路径，确保 wire body 实际包含 `"field_errors":[]`。这是已明确合同的实现注意，不是新的设计 finding。

## 6. Design Review Gate 与下游承接

### Design Review Gate：PASS

`IDENTITY-DESIGN-R30-S5.2-R02-F001` 已关闭，且本轮没有未关闭阻断项。

**S5.2 下游可承接范围：**

1. Test-Engineer 可把 test-plan 的 source revision/digest 更新为 r31，并加入 `0007` PG18、strict legacy/v1、三个 canonical fixtures、identifier-conflict 空数组拒绝矩阵、当前 trace、SMTP budget/deadline 与三条 production route 场景。
2. Task-Planner 可登记 `0007`、共享 idempotency persistence/migration owner、拟新增测试落点及实施顺序。
3. 在 test-plan/tasks/plan 所有权同步后，Backend-Dev 可按 design §9.3 的限定 write set 实施。

PASS 只表示 r31 设计对本轮范围完整、无矛盾且可交接；不表示 `0007`、strict codec/materializer、SMTP wiring、production routes、PostgreSQL 18、真实 SMTP/TLS、E2E、容量或 production readiness 已实现或验证。

## 7. 实际命令与结果

| 命令/检查 | 结果 |
|---|---|
| `git branch --show-current` | `main` |
| `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `Get-FileHash -Algorithm SHA256 design.md` | 与固定 SHA 一致 |
| revision/status 检查 | `identity-auth-design-ir01-r31` / `draft` |
| `Test-Path review-design-r31-s5.2-r03.md`（写入前） | `False` |
| UTF-8 定点读取与 `rg` | 完成 Round 1 / Round 2 |
| `git diff --check -- design.md` | PASS；仅 Git LF→CRLF 提示，无 whitespace error |

## 8. 未验证风险与限制

1. 未创建或执行 `0007`，未连接 PostgreSQL 18；CHECK、升级、双读、拒绝和回退矩阵仍待实施验证。
2. 未运行 Go test/build/race/vet；本轮是设计定点复审，不把既有 Lead 结果冒充本轮执行。
3. 未启动真实 Host、未连接真实 SMTP/TLS、未执行三条路由或 deadline/cancel E2E。
4. 现役 test-plan、tasks/plan 尚未同步 r31；这是本次 PASS 后的下游接力，不是已完成事实。
5. 本轮只复审原唯一 finding 和最小修订回归，不对完整 identity 设计或全仓消费者作总体结论。
6. 共享工作树持续变化；结论仅绑定固定 design SHA 与下列实际读取依据。

## 9. 复审条件

本设计 finding 无需再次复审。若后续修改 canonical fixture、空数组规则、`0007` 双格式 CHECK、trace 物化、公共错误语义、schema/column、owner 或 S5.2/S5.3 边界，则须提供新的固定 revision/SHA 并重新独立评审受影响范围。

## 10. 实际读取文件与 SHA-256

### 10.1 Role、Skill、Rule 与 Profile

| 文件 | SHA-256 |
|---|---|
| `.codex/agents/add-reviewer.toml` | `549202AE65D455056279D40D55C539DCCB2FA4F51AB060D7BB77891C5E1C7054` |
| `.codex/skills/add-code-review/SKILL.md` | `5D9DF5E173CFD523A444B0DEDDDB141D7DDD6C6E1EB2130D1AA6A89F9F8E79F4` |
| `.codex/skills/add-code-review/references/产品安全红线.md` | `994EABF6BD9076EBA67F497768AD298D7AEFEDA90D901DB39B6AABC263B405FA` |
| `.codex/guidance/code-review-checklist.md` | `192FA2344908E4A96D16DAC065479CC8F20C00F6EF4DC399FFE5A22470EBA72F` |
| `.codex/guidance/abstraction-principles.md` | `3FF17453E0D8A8E8B36834AFB2588C258A2F7694F3921E84C87E44B7E96F7465` |
| `.codex/guidance/design-doc-writing-style.md` | `1F182C1C8FACA6EA48FF4C827E1C0EAA953D6A1BF3D65F15D4222FE88E12F23B` |
| `.codex/guidance/decisions-guide.md` | `0E6532AE32A13F92CD295CA26515B023C9BC8A94B4FF279EFD94EA81F75D41B5` |
| `.codex/guidance/observability-standards.md` | `D88E1CA42C2BF3B46FD4CEEC41772257686E5A6CB319652ED79EB7C4C861B8FD` |
| `.codex/guidance/tracing-standards.md` | `86002304BDF8AA58B44235EBD3A35520CF915A428ACB02B64512E6B0725AF9A6` |
| `.codex/guidance/no-hardcode-policy.md` | `B52137A2AED299748D76B55A184C5EE60ADE7E0B540588E0F5A69028F3F4AB6F` |
| `.codex/profiles/go-service/README.md` | `4D221CAD44FC8CF225A47116D9CA1993405DFA018BC3314DAAF6C98AB71A6D88` |
| `.codex/profiles/go-service/rules/coding-standards.md` | `804541ECEB08D727E96217D2D38BE1B4D2A5E4D55905799790A21963F8BA83E3` |
| `.codex/profiles/go-service/rules/no-hardcode-policy.md` | `7A0F2268440F520D06C95EDF94C3E3F0DDED63731A8F582FCF495F2A200CC4A3` |
| `.codex/profiles/go-service/rules/observability-standards.md` | `F0C8BCE5FC61E274EE4833DD268E605C32516D2A4C578CD34AD793918A60025B` |
| `.codex/profiles/go-service/rules/security-coding-standards.md` | `8D5621678A5798E53D52CF0EA5D96E285A21477DFCC3FEC3E5CEDEDFC20C97A1` |
| `.codex/profiles/go-service/rules/tracing-standards.md` | `B9D7EC43EED4634EBBB3CC0378768E84BC1EB60765BA4B2BD14B959194A05338` |

### 10.2 设计、上游与规划

| 文件 | SHA-256 |
|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `CBF29B80A33C08455151D8E7621C1A115F3D16185F090A5371ACF1E29D6A55FC` |
| `.spec/wip/identity-auth/review-design-r30-s5.2-r02.md` | `B89D9F716759780FA60A1998E81CFCBBD5FF9A3CA22B0EA99F7EFACA92886670` |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `335CC44131B542E5E3EEFCCBABBA443160AC8B0B240FE2BBAB0E849D50992591` |
| `.add/project/knowledge/decisions/identity.md` | `4A1141B55D882A76DB332DD5E75CCA640A2C0CA7121980910477BC408A42BC45` |
| `.add/project/knowledge/decisions/framework.md` | `8D381C9D753242B47E1B6CF0C269228AD4ED591A4605CA4F119CAF03DB4E0AC7` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `C49D87ED526D897CBF18CDE7E338F44A845E85CC1E0412A90E2A9169367B2D88` |
| `.add/project/knowledge/tasks/tasks.md` | `E64F0B17EB4F07C7633ACA5A6EFB27A6C2F4CC2DCBBD5EE365D482722840E275` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `1E66CA07C446B0293F1C5A33F8FC583772B8EC7C3A815921BA05F6948A2B1715` |

### 10.3 真实消费者与测试接缝

| 文件 | SHA-256 |
|---|---|
| `migrations/0004_idempotency.sql` | `4794816AC2FDA4AD12522F2F324448A6F9EFB2E9E39677A595400E68148424A9` |
| `internal/migration/runner.go` | `DD473A94D645EAAFA04FC2E812EF1462B1E92CD0CC431E52F48C6F6EBB87924D` |
| `internal/idempotency/store.go` | `65F888E7DD7A73301C6EC7F626E57EA4C4922A1411EB6359B8D8240B9E2FFF80` |
| `internal/idempotency/store_test.go` | `739238655091FB540AB31DE34B10D124DE9A2E1C58FB6069817EDD4386BFB96A` |
| `internal/identity/handler.go` | `2CD9B251ECC6FEB6FF4806FA2A9F1E5D63D1A32EDBC4D7AF67598D525FF5C021` |
| `internal/identity/administrator_create.go` | `0B93D2DD68D898C0280AD4C81998B55351E7C0DB78DEB56CD2A353C4F8194997` |
| `internal/identity/administrator_create_test.go` | `02AC8BE2347E36A770CD2D1E3AB04FA2F10D4402E7028870A7EA62BC95D9E6D0` |
| `internal/platform/http.go` | `0890069CB559C7B8454A242B46736FC23D542B0D45E350B0D035C7012BB710C4` |
| `internal/platform/errors.go` | `99B1094A6C15A6E86E45B43928F8AC1B4F3A4EE810857BE598E9AA4336D968E3` |
| `internal/platform/logging.go` | `F362D36954DF0119D5DC25040640C796069CC8E178A1D92B828C6D6149CF30A9` |
| `internal/mailer/smtp.go` | `3B66245D0CDF4A7FB1E878EF68CE04342A32DABF0214AD1F5B8871F32C444F4F` |
| `cmd/han-server/main.go` | `5DBACCB74EF15F53BCDCC1845ADE8B66327163F8817A84C708489C0082E61988` |
| `cmd/han-server/main_test.go` | `D83244E9B2F8277FDE485D7FA2DCD14ADB570F9A3FD53361DB48550C903DDA77` |

本复审不替代 Lead/用户对修订采用、任务所有权、风险接受或后续实施的授权。
