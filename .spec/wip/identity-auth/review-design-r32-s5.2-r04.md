# Identity S5.2 Design r32 独立评审

- **Reviewer**：Faraday（非作者）
- **评审日期**：2026-10-23
- **objectType**：design
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **固定对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- **revision / status**：`identity-auth-design-ir01-r32` / `draft`
- **git blob**：`630583c4b457440eb5e0f2e4383632afa6daba36`
- **SHA-256**：`DD808FC7CFEBF8AB7DE633F50797B38F3DDE2C83D5F1388C799846587FFB271D`
- **历史基线**：`.spec/wip/identity-auth/review-design-r31-s5.2-r03.md`（r31 PASS，不继承到 r32）
- **实现缺口来源**：`.spec/wip/identity-auth/review-code-s5.2-r01.md` 的 `IDENTITY-S5.2-R01-F003`
- **结论**：**FAIL — 1 项 MEDIUM finding OPEN**

本结论只绑定上述固定 r32 对象。Reviewer 未修改设计、代码、测试规划、任务/批次计划、migration、ADR、architecture、boundary、Manifest 或旧报告；仅新建本报告。

## 1. 两轮实际评审范围

### Round 1：r32 直接设计合同

逐项检查：

1. 管理员声明四键、幂等 current/retained keyring、lock/TTL、SMTP endpoint/from/auth/CA/TLS/operation budget 的必填性、无默认、语法范围、交叉约束和错误去敏。
2. `startupConfig → preparedProductionDependencies → businessComposition → newProductionBusinessHandler` 的构造阶段、唯一 owner、secret 引用/清理、失败 unwind、共享实例与数据库/listener 副作用时序。
3. current administrator notice、administrator create、privacy acknowledgement 三条路由的 method/path、Trace、Origin、Session、CSRF、Principal/Access、Handler/coordinator 顺序，以及 JSON 404/405 和 `Allow`。
4. 配置、CA/TLS/auth、共享实例、deadline/cancel、零副作用、factory/unwind、路由可达性测试矩阵。
5. S5.2 只证明配置/接线/deadline/cancel/零副作用，真实 DNS/TCP/TLS/AUTH 和 provider 受理留给 S5.3 的边界。
6. r31 已固定的 D001、strict legacy/v1、当前请求 trace、`0007` 和 identifier-conflict 合同是否被直接回退。

### Round 2：真实消费者与后续接缝

有界核对：

- `cmd/han-server` 当前 startup config、prepared dependency、production factory/local router 及测试替换点；
- `internal/platform` 的配置读取、OriginGuard、HTTPAuthorizer、Principal、JSON error/trace 接缝；
- `internal/mailer.SMTPConfig`、TLS 模式、caller deadline、TLS 后 AUTH 与 adapter 测试；
- `internal/idempotency.DigestKeyring`、lock/TTL 消费结构与 store 测试；
- `AdministratorCreateCoordinator`、`AdministratorPrivacyCoordinator`、`StaticAdministratorInformationNoticeReader`、Handler 响应序列化及相关测试；
- `session.AccessGeneral`、`session.AccessAcknowledgePrivacy`；
- identity boundary、architecture、identity/framework ADR、test-plan r12、tasks 与 B6A batch plan。

未扩展为全仓审查，未运行会连接数据库或发起真实 SMTP/DNS/TCP/TLS/AUTH 的命令。

## 2. Findings

### CRITICAL / HIGH

无。

### MEDIUM

#### `IDENTITY-DESIGN-R32-S5.2-R04-F001` — 管理员声明 effective-at 的合法精度与现役 HTTP 序列化不一致

- **状态**：OPEN
- **位置**
  - `design.md:1104`：配置值通过 `parsed.UTC().Format(time.RFC3339Nano)` 与输入逐字节比较，因此带小数秒的规范 UTC 值是合法输入。
  - `design.md:1641-1645`：current administrator notice 公共响应返回 `effective_at`。
  - `design.md:2872`：正确配置构造的 reader 必须精确保留 `EffectiveAt`。
  - `internal/identity/handler.go:672-674`：现役 Handler 使用 `notice.EffectiveAt.UTC().Format(time.RFC3339)`，该布局不输出小数秒。
- **违反依据**
  - r32 的目标是让配置键表与现役消费者签名单义兼容，并给出可实施的 production composition。
  - 设计写作规则要求关键格式、边界和消费者行为没有两种合理解释。
- **触发场景**
  1. 部署配置 `HAN_ADMINISTRATOR_INFORMATION_NOTICE_EFFECTIVE_AT=2026-10-23T10:00:00.123Z`。
  2. 该值满足 r32 的 RFC3339Nano canonical round-trip 规则。
  3. `StaticAdministratorInformationNoticeReader` 保留对应 `time.Time`。
  4. current notice Handler 返回 `2026-10-23T10:00:00Z`，小数秒被丢弃。
- **影响**
  - 公共接口返回的声明生效时刻与获批准配置不再是同一时刻。
  - 实现者可能分别选择“非文档化拒绝小数秒”或“修改 Handler 为 RFC3339Nano”，当前设计不能唯一指导实现和契约测试。
  - S5.2 的配置到真实路由闭环无法证明逐值一致，故阻断 Design Review Gate。
- **最小修复**
  - 由设计 owner 明确选择并全文统一一种合同：
    1. 仅接受规范整秒 UTC `time.RFC3339`，配置 parser 明确拒绝任何小数秒；或
    2. 保留当前 RFC3339Nano 配置范围，并明确 current notice Handler 使用 `time.RFC3339Nano`。
  - 相应增加配置 → reader → HTTP 响应的 fractional-second fixture；若选择整秒合同，则该 fixture应证明小数秒在启动阶段被拒绝。
- **证据**
  - Go 标准库当前布局为 `RFC3339 = "2006-01-02T15:04:05Z07:00"`、`RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"`。
  - 现役 Handler 源码明确选择前者；r32 配置 canonical check 明确选择后者。

### LOW

无。

## 3. 其余重点核对结果

| 检查项 | 结果 |
|---|---|
| 配置完整性 | 除 F001 外，管理员声明、current/retained keyring、lock/TTL、SMTP host/port/server-name/from/auth/CA/TLS/budget 的键名、范围、无默认和交叉约束内部一致 |
| 现役签名兼容 | `mailer.SMTPConfig`、`idempotency.DigestKeyring`、`AdministratorCreateDependencies`、Session access 与设计注入形状兼容 |
| 启动时序 | 配置解析失败在 DSN parse/connect/Ping/migrate/prepare/factory/listener/route 前 fail closed；CA/adapter prepare 失败在 DB 副作用前停止 |
| ownership / unwind | startup 临时秘密、prepared 运行副本、pool/listener 与 factory 所有权、逆序清理和不重复关闭合同明确；未伪称 Go string 可物理清零 |
| SMTP 安全 | 端口不推导 TLS；auth 只允许 `plain/none`；可选 CA 追加系统根；坏文件/坏 PEM fail closed；TLS1.2、主机验证、TLS 后 AUTH、caller deadline 均有明确测试入口 |
| 三条 production route | method/path、Trace 最外层、Origin/Session/CSRF/Principal/Access/Handler 顺序、404/405/Allow 和共享实例要求明确 |
| deadline/cancel | 请求 deadline 与配置 budget 取较早者、及时 cancel、parent cancellation 与零副作用矩阵明确 |
| S5.2/S5.3 | S5.2 不发真实网络；真实 SMTP/TLS/AUTH/provider 证据归 S5.3，不宣称容量或 production readiness |
| r31 合同回归 | 未发现对 D001、legacy/v1、当前 trace、`0007`、identifier-conflict 空数组、稳定快照或三路由合同的直接回退 |
| 下游现状 | test-plan、tasks 与 batch plan 当前仍绑定 r31；它们是 r32 通过后的后续同步对象，不是 r32 已实施证据 |

`IDENTITY-S5.2-R01-F003` 是现役代码的 production composition/router 实现缺口；r32 已补足大部分设计输入，但本设计评审不替代后续代码修复和代码复审，也不把该 code finding 标为 CLOSED。

## 4. 待决策与非阻断建议

### 待决策

仅 F001 的 effective-at 精度选择待设计 owner 明确。两种最小方向均不要求新增公共错误、schema/migration、依赖或结构性 ADR；Reviewer 不替 owner 选择。

### 非阻断建议

完成 F001 时，配置 canonical 校验与 HTTP 输出宜复用一个明确命名的时间格式合同或共享 fixture，避免 loader 与 Handler 再次漂移。除此之外无独立非阻断建议。

## 5. Design Review Gate 与下游承接

### Design Review Gate：FAIL

原因：`IDENTITY-DESIGN-R32-S5.2-R04-F001` 仍 OPEN。r31 PASS 不继承到新对象 r32。

在新的固定 revision/SHA 关闭 F001 前，不应把 r32 作为完整、单义、可实施的 S5.2/F003 设计 Gate 交给 Test-Engineer、Task-Planner 或 Backend-Dev。r31 已关闭且 r32 未回退的快照、trace、SMTP budget、production route 与 S5.2/S5.3 合同可继续作为历史依据，但不能替代 r32 的正式 PASS。

## 6. 未验证风险与限制

1. 本轮是 design review，未修改或执行 r32 所描述的 production composition、配置 parser、`0007`、strict codec/materializer 或三条路由。
2. 未运行 Go build/test/race/vet；未把 Lead 或历史测试结果冒充本轮执行证据。
3. 未连接 PostgreSQL 18，未读取或打印 User scope 的 `HAN_TEST_POSTGRES_DSN`。
4. 未启动真实 Host，未发起 DNS/TCP/TLS/AUTH/SMTP，未验证真实 CA/provider、邮件受理、容量、部署或 E2E。
5. test-plan r12、tasks 与 batch plan 尚绑定 r31；本轮只核对其后续接力合同，没有修改这些只读对象。
6. 共享工作树存在大量范围外未提交成果；消费者哈希只表示本轮读取时的接缝快照，结论不得外推为全仓通过或失败。

## 7. 复审条件

1. 提供新的固定 design revision、git blob 与 SHA-256。
2. 明确 effective-at 允许精度和唯一 HTTP formatter，并同步配置表、接口说明、构造责任、测试矩阵及影响面。
3. 提供配置 → `StaticAdministratorInformationNoticeReader` → current notice HTTP 响应的精度 fixture。
4. 原 Reviewer 对 F001 和最小修订回归定点复审；没有新阻断项后方可给出 Design Review Gate PASS。

## 8. 实际命令与结果

| 命令/检查 | 结果 |
|---|---|
| `git branch --show-current` | `main` |
| `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| revision/status 定点检索 | `identity-auth-design-ir01-r32` / `draft` |
| `git hash-object -- design.md` | `630583c4b457440eb5e0f2e4383632afa6daba36` |
| `Get-FileHash -Algorithm SHA256 design.md` | `DD808FC7CFEBF8AB7DE633F50797B38F3DDE2C83D5F1388C799846587FFB271D` |
| 写入前 `Test-Path review-design-r32-s5.2-r04.md` | `False` |
| `rg` / UTF-8 定点读取设计、历史报告、规划、代码与测试接缝 | 完成 Round 1 / Round 2 |
| `git diff --check -- design.md` | PASS；仅 Git LF→CRLF 提示，无 whitespace error |
| `go doc time.RFC3339 time.RFC3339Nano` | 命令形式错误，返回 `doc: no such package: time.RFC3339`；不作为证据 |
| `go doc time RFC3339` / `go doc time RFC3339Nano` | PASS；确认两个标准布局的秒精度差异 |
| 写入后固定对象 revision/blob/SHA 复核 | PASS；仍为 r32 / `630583c4...` / `DD808FC7...` |
| `git diff --check -- design.md report.md` | PASS；仅 design 的 Git LF→CRLF 提示，无 whitespace error |
| `git status --short -- report.md` | `??`；本报告为本轮唯一新增文件 |

## 9. 实际读取文件与 SHA-256

### 9.1 Role、Skill、Rule 与 Profile

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

### 9.2 设计、历史评审与规划依据

| 文件 | SHA-256 |
|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `DD808FC7CFEBF8AB7DE633F50797B38F3DDE2C83D5F1388C799846587FFB271D` |
| `.spec/wip/identity-auth/review-design-r31-s5.2-r03.md` | `08753F743B85CE599368D45FD74023EA093EF9B786A2B5F4C46A8299B100B133` |
| `.spec/wip/identity-auth/review-code-s5.2-r01.md` | `69645C38F55584FB8ABEB96E6CB8813F9E9357E10066D76CFF04A898D162D63E` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `B8612EDF6C43234CC1AB3017F3714C8DAAA6F42B601E099C71469EE1ED1E8E8C` |
| `.add/project/knowledge/tasks/tasks.md` | `E33456CE746FC8473449324FAC5EB42CB12EA4CCC7AE475F8841BD9EF3835796` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `A53621669134A37A22EB5A79E79E3FF82FE717BF119C55764880AC045B47E473` |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `335CC44131B542E5E3EEFCCBABBA443160AC8B0B240FE2BBAB0E849D50992591` |
| `.add/project/knowledge/design/architecture.md` | `FAC19BE91763855EC4CE91D97B8AC052A824AAC643794E4B0F45A01EE04C586B` |
| `.add/project/knowledge/decisions/identity.md` | `4A1141B55D882A76DB332DD5E75CCA640A2C0CA7121980910477BC408A42BC45` |
| `.add/project/knowledge/decisions/framework.md` | `8D381C9D753242B47E1B6CF0C269228AD4ED591A4605CA4F119CAF03DB4E0AC7` |

### 9.3 真实消费者与测试接缝

| 文件 | SHA-256 |
|---|---|
| `cmd/han-server/main.go` | `0AFBBAA9A38D91B295D9EB7192F085302741334E3DB466BEEE2860ED686E5B90` |
| `cmd/han-server/main_test.go` | `ADDE9CA171BBC48CE6451DACB2E5CF3E6EA697609AB1B8DADB2B3DD472F8E63D` |
| `internal/platform/config.go` | `3F5E28FE3404DC970374F39F16556C91E28AA9DBEC30E662109EFD58B8F866B0` |
| `internal/platform/config_test.go` | `3A1DD70ECC3209A782CDD01C8F61E7AC941D15555450332B37298C9421950A46` |
| `internal/platform/http.go` | `0890069CB559C7B8454A242B46736FC23D542B0D45E350B0D035C7012BB710C4` |
| `internal/platform/http_security.go` | `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` |
| `internal/platform/http_security_test.go` | `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` |
| `internal/mailer/smtp.go` | `3B66245D0CDF4A7FB1E878EF68CE04342A32DABF0214AD1F5B8871F32C444F4F` |
| `internal/mailer/smtp_test.go` | `08FC9DF16CEBC7A1A47E293647EA8B6E2F0B5A2EED0154791B019F7FCEA9412E` |
| `internal/idempotency/store.go` | `E9AAE069942F310EB155FBCAE3E981B55C87ABCAFCF3ACA74730292483C57FA5` |
| `internal/idempotency/store_test.go` | `DB291F9AE7F501F60ABF71F715FC2DD2961067E8598CED64348A7A6AC77DAC5C` |
| `internal/identity/handler.go` | `F21E1E9B824DADCD784AA89E4A9865CDD85BF17A372B4D9C2964A91250C456D9` |
| `internal/identity/handler_test.go` | `7A32FEF8C9C94489D8D8E055C6B5ACEB7E2EA0304BCF028BDE0CF1D0BCA6373F` |
| `internal/identity/administrator_create.go` | `7A73F17BED8FD6EBE321AC5DAF3F02AE1EE7BBA8E4A32D06216608AA1A1E75D7` |
| `internal/identity/administrator_create_test.go` | `E94CC8EB8F3B4F0D2633C55354B350DE3CC65F6080A78F853C4F0D85C9C12552` |
| `internal/session/store.go` | `0E7D65194ABE283E0E01BE67C35C170CFD41EA07A38AA70A4C45EBA5FD482347` |
| `internal/session/store_test.go` | `CB33A14D2D241FB3179B1715EFE583DC13898963747D20D5199615DDA3C92EA3` |

本独立结论不替代 Lead/用户对修订采用、风险接受、任务范围或后续实施的授权。
