# Identity S5.2 Design r33 定点独立复审

- **Reviewer**：Faraday（非作者）
- **评审日期**：2026-10-23
- **objectType**：design
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **固定对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- **revision / status**：`identity-auth-design-ir01-r33` / `draft`
- **git blob**：`1c6ea6827a18158a9936a4612aac523b258e4e5d`
- **SHA-256**：`47FC3598CE2BCA7A1E80032FEC040BCF63344C869B4EF12F9BE5998589563513`
- **原报告**：`.spec/wip/identity-auth/review-design-r32-s5.2-r04.md`
- **原 finding**：`IDENTITY-DESIGN-R32-S5.2-R04-F001`
- **结论**：**PASS；原 finding CLOSED；无新阻断 finding**

本结论只绑定上述固定 r33 对象。Reviewer 未修改设计、代码、测试计划、任务/批次计划、ADR、boundary、architecture、migration、checkpoint 或旧报告；本轮仅新增本报告。

## 1. 两轮实际评审范围

### Round 1：r33 最小修订的直接正确性

逐项检查：

1. `effective_at` 是否只接受规范 UTC/Z、offset=0、非零、整秒 `time.RFC3339`。
2. 输入与 `parsed.UTC().Format(time.RFC3339)` 的逐字节校验是否明确拒绝 `.1Z`、`.123Z`、`.000Z` 等所有小数秒。
3. 配置表、值流说明、公共响应 fixture、测试矩阵与实施落点是否使用同一合同。
4. Handler 是否继续采用现役 `time.RFC3339`，且没有截断、舍入、修正输入或升级成 RFC3339Nano 的双重解释。

### Round 2：真实消费者接缝与 r32 合同回归

有界核对：

- `StaticAdministratorInformationNoticeReader` 的构造、校验和原值返回行为。
- current administrator information notice Handler 的 HTTP 序列化。
- 现役 identity 测试落点及 r33 指定的后续 fixture。
- r32 的 SMTP/keyring、startup lifecycle/secret clear/unwind、三条 production routes、D001、strict legacy/v1、当前请求 trace、`0007` 与 S5.2/S5.3 分界是否被 r33 直接回退。

未扩展为全仓审查，也未把尚待实施的 production composition/parser/fixture 当作设计缺陷。

## 2. 原 finding 复审

### `IDENTITY-DESIGN-R32-S5.2-R04-F001` — CLOSED

**原问题**：r32 允许 RFC3339Nano 小数秒，但现役 Handler 使用 `time.RFC3339` 输出，合法配置可能在 HTTP 响应中丢失小数秒。

**r33 关闭证据**：

1. `design.md:1108` 将配置合同收口为规范 UTC 整秒：先按 RFC3339 解析，要求 `Z`、offset=0、非零，并要求输入与 `parsed.UTC().Format(time.RFC3339)` 逐字节一致；明确拒绝包括 `.0Z`、`.000Z` 在内的所有 fractional seconds。
2. `design.md:1126` 固定唯一值流：配置字符串 → RFC3339 解析及 canonical 校验 → UTC 整秒 `time.Time` → `StaticAdministratorInformationNoticeReader` → Handler `time.RFC3339`；禁止任一层截断、舍入、补 `Z` 或修正小数秒输入。
3. `design.md:1650-1653` 给出 canonical HTTP fixture `2026-10-24T00:00:00Z`，并要求 `2026-10-24T00:00:00.123Z` 在启动配置解析阶段拒绝、不得进入 reader。
4. `design.md:2880` 要求正例从配置到 HTTP 响应逐字节保持；负例覆盖 `.1Z`、`.123Z`、`.000Z`、非 UTC offset、非 `Z`、零值及解析失败，并要求失败时 reader/Handler 调用数为 0。
5. `internal/identity/administrator_create.go:1306-1324` 的静态 reader 校验后直接返回 `r.Notice`，不改变时间精度。
6. `internal/identity/handler.go:672-674` 的现役 HTTP 输出使用 `notice.EffectiveAt.UTC().Format(time.RFC3339)`，与 r33 只允许规范 UTC 整秒的输入域一致。

**判定**：合法输入不再包含 Handler 无法保持的小数秒；非法小数秒在 reader/Handler 前 fail closed。配置、reader 与 HTTP formatter 之间不存在剩余精度歧义，原触发场景已被消除。

## 3. 新 findings

### CRITICAL / HIGH / MEDIUM / LOW

无。

## 4. r32 其余合同回归结果

| 检查项 | 结果 |
|---|---|
| SMTP / keyring | 未发现对 SMTP 必填配置、budget/deadline/cancel、TLS/auth/CA，或 current/retained keyring、lock/TTL 合同的直接回退 |
| lifecycle / ownership | 未发现对 `startupConfig → preparedProductionDependencies → businessComposition → newProductionBusinessHandler`、secret clear、失败 unwind、共享实例合同的直接回退 |
| production routes | 三条既有管理员路由、method/path、middleware 顺序、JSON 404/405 与 `Allow` 合同未被 r33 改写 |
| D001 | active initial-admin 授权优先，授权后稳定终态重放优先于动态 Gate/notice/password policy 的顺序未回退 |
| snapshot / migration | strict legacy/v1、当前请求 trace 物化和前向 `0007_idempotency_response_snapshot_v1.sql` 合同未回退 |
| S5.2 / S5.3 | S5.2 仍只证明配置、接线、deadline/cancel 与零网络副作用；真实 DNS/TCP/TLS/AUTH/SMTP 仍归 S5.3，未宣称 production readiness |

`IDENTITY-S5.2-R01-F003` 仍是后续代码实施与代码复审对象；本次 Design PASS 不将该 code finding 标为关闭。

## 5. 待决定与非阻断建议

### 待决定

无。r33 使用的是已授权的整秒 RFC3339 最小修复，没有引入新的公共错误、schema/migration、依赖或架构方向。

### 非阻断建议

无独立建议。实施与测试应原样保留 r33 已指定的整秒正例和全部 fractional-second 启动拒绝 fixture，不以 formatter 隐式修正替代 parser 拒绝。

## 6. Design Review Gate 与下游承接

### Design Review Gate：PASS

通过依据：

- 原唯一 finding 已关闭；
- 未发现新阻断项或需重新决策的矛盾；
- config → parser → `StaticAdministratorInformationNoticeReader` → Handler HTTP fixture 已形成单义、可实施、与现役消费者兼容的合同；
- r32 其余已固定合同未见直接回退。

可承接范围：

- Test-Engineer 可据 r33 同步整秒正例、fractional seconds/offset/非规范/零值负例及零调用断言。
- Task-Planner 可登记 startup parser、静态 reader、Handler HTTP fixture 和现有 S5.2/F003 production composition 接力。
- Backend-Dev 可在取得对应任务与文件所有权后实施。

本 PASS 不表示代码、migration、PostgreSQL 18、真实 SMTP/TLS、部署或 E2E 已完成。

## 7. 未验证风险与限制

1. 本轮是定点 design review，未修改或运行 r33 描述的 startup parser、production composition、migration、路由或 fixture。
2. 未运行 Go build/test/race/vet；既有验证结果未冒充本轮执行证据。
3. 未连接数据库，未读取或打印 `HAN_TEST_POSTGRES_DSN`，未启动 Host，也未发起 DNS/TCP/TLS/AUTH/SMTP。
4. Round 2 只沿直接消费者与 r32 合同做回归核对；共享工作树其他未提交成果不在本结论范围。
5. 消费者哈希只表示本轮读取时快照；后续实现或消费者变更仍须独立代码评审。

## 8. 实际命令与结果

| 命令/检查 | 结果 |
|---|---|
| `git branch --show-current` | `main` |
| `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| revision/status 定点检查 | `identity-auth-design-ir01-r33` / `draft` |
| `git hash-object -- design.md` | `1c6ea6827a18158a9936a4612aac523b258e4e5d` |
| `Get-FileHash -Algorithm SHA256 design.md` | `47FC3598CE2BCA7A1E80032FEC040BCF63344C869B4EF12F9BE5998589563513` |
| 写入前 `Test-Path review-design-r33-s5.2-r05.md` | `False` |
| `rg -n` / UTF-8 定点读取 r33、原报告和直接消费者 | 完成 Round 1 / Round 2 |
| `git diff --check -- design.md` | PASS；仅 Git 的 LF→CRLF warning，无 whitespace error |

落报告后再次核对固定对象 revision/blob/SHA、限定 `git diff --check`、报告状态与报告 SHA-256。

## 9. 实际读取文件

### Role、Skill、Rule 与 Profile

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- add-code-review 引用的产品安全红线
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/decisions-guide.md`
- `.codex/guidance/observability-standards.md`
- `.codex/guidance/tracing-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- 已启用 `go-service` profile 的适用规则

### 对象、历史报告与直接消费者

- `.add/project/knowledge/design-workspace/identity-auth/design.md`
- `.spec/wip/identity-auth/review-design-r32-s5.2-r04.md`
- `internal/identity/handler.go`
- `internal/identity/handler_test.go`
- `internal/identity/administrator_create.go`
- `internal/identity/administrator_create_test.go`

直接消费者读取快照：

| 文件 | SHA-256 |
|---|---|
| `internal/identity/handler.go` | `F21E1E9B824DADCD784AA89E4A9865CDD85BF17A372B4D9C2964A91250C456D9` |
| `internal/identity/handler_test.go` | `7A32FEF8C9C94489D8D8E055C6B5ACEB7E2EA0304BCF028BDE0CF1D0BCA6373F` |
| `internal/identity/administrator_create.go` | `7A73F17BED8FD6EBE321AC5DAF3F02AE1EE7BBA8E4A32D06216608AA1A1E75D7` |
| `internal/identity/administrator_create_test.go` | `E94CC8EB8F3B4F0D2633C55354B350DE3CC65F6080A78F853C4F0D85C9C12552` |

本独立结论不替代 Lead/用户对修订采用、任务范围、实施授权或风险接受的决定。
