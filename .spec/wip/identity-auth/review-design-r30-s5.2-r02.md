# Identity S5.2 Design r30 定点独立复审

- **Reviewer**：Faraday（非作者）
- **日期**：2026-10-23
- **objectType**：design
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **固定对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- **revision / status**：`identity-auth-design-ir01-r30` / `draft`
- **固定 SHA-256**：`0D3D23A02D52685FE87DFF50C02DA1BFC8BE54C8381BD550D881BC4497DAAD10`
- **原失败报告**：`.spec/wip/identity-auth/review-design-r29-s5.2-r01.md`，SHA-256 `49D6001BBBF5C6CB95038DBC1B984484C8B62422749D8AE3DC4A8DDB083E9495`
- **结论**：**FAIL — 原 F001/F002/F003 均已关闭，但发现 1 项新的阻断性 MEDIUM**

本结论只绑定上述固定对象。Reviewer 未修改设计、旧报告、test-plan、tasks/plan、代码、migration、ADR、boundary 或 architecture；仅新建本报告。

## 1. 两轮实际检查范围

### Round 1：r30 直接合同

1. 定点复审原 `IDENTITY-DESIGN-R29-S5.2-R01-F001/F002/F003`；
2. 核对 `0007` 的 legacy/v1 严格双读、升级、回退、owner、PostgreSQL 18 拒绝矩阵；
3. 核对 trace A→B 时 header/body/context/log 与稳定业务字段的来源；
4. 核对外层、success body、failed body 的字段、类型、必填性、映射和 canonical fixtures；
5. 核对既定 D001 顺序、SMTP budget、三条 production route 及 S5.2/S5.3 分界未被改写。

### Round 2：真实消费者与交接接缝

有界读取并核对：

- `migrations/0004_idempotency.sql` 与 migration runner；
- `internal/idempotency` 的现役 legacy snapshot、strict decoder 与测试；
- identity Handler、AdministratorCreateCoordinator 及其测试；
- platform trace/error/logging；
- SMTP adapter；
- `cmd/han-server` production composition/router 与测试；
- identity boundary、identity/framework ADR、现役 test-plan、tasks 与 batch plan。

未扩展为全仓审查。

## 2. 原 finding 复审结果

| 原 finding | r30 证据 | 复审状态 |
|---|---|---|
| `IDENTITY-DESIGN-R29-S5.2-R01-F001` | design:581、2058-2079、2578、2791、2945-2947：新增前向 `0007`，不改 `0004`/列/旧行；legacy/v1 互斥双读；管理员只写 v1；固定停旧进程→PG18 migration→双读新进程；固定首条 v1 前后回退边界、owner 与拒绝矩阵 | **CLOSED** |
| `IDENTITY-DESIGN-R29-S5.2-R01-F002` | design:866-867、1141-1142、1577、1581、2056、2790、2949-2950：快照排除 `trace_id`；首次与 replay 均用当前 request context；A→B 时 header/body/context/log 全为 B；稳定 status/code/message/field_errors 来自快照且不经当前 registry 重算 | **CLOSED** |
| `IDENTITY-DESIGN-R29-S5.2-R01-F003` | design:1605-1640、2058-2066、2789、2839：外层严格四字段，success 严格八字段，failed 严格三字段；字段类型与必填性明确；拒绝 extra/missing/null/type drift、未知版本、未知映射和 trace | **CLOSED（原“字段集合自相矛盾”已消除）** |

## 3. Findings

### CRITICAL / HIGH

无。

### MEDIUM

#### IDENTITY-DESIGN-R30-S5.2-R02-F001 — `identity.identifier_conflict` 没有唯一 canonical `field_errors`，严格快照合同仍不能单义实施

- **严重度 / Gate 属性**：MEDIUM / 阻断
- **位置**
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:1305`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:1605-1640`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:2058-2079`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:2789-2791`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:2945-2947`
  - `internal/identity/administrator_create.go:1122-1126`
- **违反依据**
  - r30 自身要求 strict fixture 精确接受 success、identifier conflict、email delivery failed，并要求应用 decoder、数据库 CHECK 与测试一致。
  - 设计只给出了 email-delivery failed 的完整 JSON fixture；identifier conflict 仅固定 `409/code/message`。
  - `field_errors` 的通用结构虽已固定，但 identifier conflict 可使用空数组、单个 username/email/phone、多个字段及不同顺序；design:1305 还只表述为“可通过 field_errors 指明字段”。
- **触发场景**
  1. 当前实现把任意 unique violation 统一映射为 `identity.identifier_conflict`，没有提供唯一冲突字段；
  2. 一个 writer 保存 `"field_errors":[]`，另一个实现保存 username/email/phone 项或多个项；
  3. 这些结果都可能满足当前文字定义的通用 schema，却不能同时成为同一版本的唯一 canonical fixture。
- **影响**
  - migration CHECK、Go allowlist/decoder、writer 和 strict fixture 可作出不同但各自“看似合规”的选择；
  - 同一 `identity-administrator-create-response/v1` 不能保证唯一稳定响应，后续重放与兼容测试存在实现漂移；
  - Task-Planner/Backend-Dev 无法从设计唯一确定 identifier-conflict 的持久化值和拒绝边界。
- **修复建议**
  - 在设计中增加 identifier-conflict 的完整 canonical JSON fixture，并固定 `field_errors` 的精确值与顺序。
  - 最小方案可固定为空数组；若必须披露冲突字段，则需同时固定允许字段、field-error code、单项/多项规则、排序及冲突字段来源。
  - 同步要求 `0007` CHECK、应用 decoder/writer 和 PostgreSQL 18/Go fixture 测试采用同一规则。
- **复审证据**
  - email-delivery failed 已有完整 fixture（design:1625-1637）；
  - identifier conflict 只有映射 allowlist（design:1640、2066），没有完整 fixture；
  - 验证条目却要求 strict identifier-conflict fixture（design:2789、2791）。

### LOW

无。

## 4. 原 code review F001～F003/D001 映射

| 原项 | r30 设计覆盖 | 本轮结论 |
|---|---|---|
| `IDENTITY-S5.2-R01-F001` | 授权后稳定终态重放、版本化安全快照、strict legacy/v1、坏快照 fail closed、零副作用、当前 trace 物化均已明确 | **主体方向充分，但新 F001 阻断 identifier-conflict 快照的最终实施交接** |
| `IDENTITY-S5.2-R01-F002` | SMTP budget 必填无默认、`10s` 基线、`1s..60s`、启动前 fail closed、较早 deadline、parent cancellation、及时 cancel 与零副作用已明确 | **设计覆盖充分；待实现与验证** |
| `IDENTITY-S5.2-R01-F003` | 唯一 production composition、三条 route、method/path、中间件、JSON 404/405 与资源 ownership 已明确 | **设计覆盖充分；待实现与验证** |
| `IDENTITY-S5.2-R01-D001` | active initial-admin 授权 → 幂等只读分类 → 摘要冲突/稳定重放 → 仅新执行进入动态 Gate/notice/policy 的顺序未回退 | **CLOSED（设计方向）** |

## 5. 待决定与非阻断建议

### 待决定

无新的架构或 ADR 方向决定。只需由设计责任人把 identifier-conflict 的既有 v1 variant 收口为唯一 fixture；不得借此新增公共错误或重开已拍板值。

### 非阻断建议

1. 现役 test-plan、tasks 与 batch plan 仍主要绑定 r28/r11 和旧 S5.2 写集，尚未登记 `0007`、strict 双读、materializer 与拟新增 Handler 测试。它们是后续 Test-Engineer/Task-Planner 的同步工作，不是本轮设计对象的历史漂移 finding。
2. 幂等 retention、真实 SMTP/TLS、容量与 production readiness 仍不属于 S5.2 已证明内容。

## 6. Design Review Gate 与下游可承接范围

### Design Review Gate：FAIL

原 r29 review 的三项 finding 已关闭，但新 `IDENTITY-DESIGN-R30-S5.2-R02-F001` 使 v1 identifier-conflict variant 仍非单义，因而不能对完整 S5.2 设计给 PASS。

**下游可承接范围：**

- Test-Engineer/Task-Planner 可先消费已关闭的 `0007`、trace A→B、严格字段集合、SMTP budget、production routes 与 S5.2/S5.3 分界，准备同步草案；
- 在新 F001 关闭前，不得把 identifier-conflict fixture、DB CHECK 或整张 S5.2 计划标记为最终批准；
- Backend-Dev 不应据 r30 开始完整 v1 snapshot/migration 实施；正式 handoff 仍需设计复审 PASS、test-plan 与 tasks/plan 所有权同步。

## 7. 复审条件

1. 提供新的固定 design revision/SHA；
2. 给出 identifier conflict 的完整 canonical JSON fixture；
3. 固定其 `field_errors` 精确值、允许集合和顺序，并让 DB CHECK、应用 decoder/writer、Go fixture 与 PG18 fixture 一致；
4. 再由原非作者 Reviewer 定点复审新 F001，不重开已关闭的 r29 F001/F002/F003。

## 8. 实际命令与结果

| 命令/检查 | 结果 |
|---|---|
| `git branch --show-current` | `main` |
| `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `Get-FileHash -Algorithm SHA256 design.md` | 与固定 SHA 一致 |
| `Test-Path review-design-r30-s5.2-r02.md`（写入前） | `False` |
| `git status --short --branch` | 确认共享工作树存在大量他人未提交成果；未回滚、覆盖或格式化 |
| `rg` / UTF-8 定点读取 design、旧报告、规则、消费者、migration、测试与规划接缝 | 完成 Round 1 / Round 2 |
| `git diff --check -- design.md` | PASS；仅出现 Git 的 LF→CRLF 提示，无 whitespace error |

## 9. 未验证风险与限制

1. 未创建或执行 `0007`，未连接 PostgreSQL 18；migration 可实施性为设计与现役约束的静态核对。
2. 未运行 Go test/build/race/vet；本轮是 design 复审，不把既有 Lead 结果冒充本轮执行。
3. 未连接真实 SMTP/TLS，未启动真实 Host，未执行三条路由、deadline/cancel、404/405 或资源关闭 E2E。
4. 未审查允许范围外的全仓消费者，不对整个 identity r30 或整个仓库作总体结论。
5. 共享工作树持续变化；结论仅绑定固定 design SHA 与下列读取依据。

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
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `0D3D23A02D52685FE87DFF50C02DA1BFC8BE54C8381BD550D881BC4497DAAD10` |
| `.spec/wip/identity-auth/review-design-r29-s5.2-r01.md` | `49D6001BBBF5C6CB95038DBC1B984484C8B62422749D8AE3DC4A8DDB083E9495` |
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

本复审不替代 Lead/用户对设计修订采用、任务所有权、风险接受或后续实现的授权。
