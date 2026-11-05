# identity-auth design r27 + test-plan r09 Q1 增量独立评审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **对象类型**：design + test plan
- **结论**：**FAIL / Design Review Gate NOT PASS**
- **Finding 统计**：CRITICAL 0 / HIGH 1 / MEDIUM 2 / LOW 0

## 1. 固定对象与边界

| 项目 | 固定值 / 结果 |
|---|---|
| design | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| design revision | `identity-auth-design-ir01-r27` |
| design SHA256 | `5CB8A71618B9B1DB8FEC255D75EBE001F50B70D7BF5A77D3A6A04785A67D3E5A` |
| test plan | `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` |
| test-plan revision | `identity-auth-test-plan-ir01-r09` |
| test-plan SHA256 | `F996559607F78734F820A8D9620546FA4FC4F591D3AA70EA426909D89F1859BB` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |

本轮只评审上述固定对象的 Q1 增量。design、test-plan、代码、tasks、Plan、checkpoint 及他人文件均保持只读；唯一写入本报告。未运行实现测试。

## 2. 两轮实际检查范围

### Round 1：固定对象直接检查

检查了：

- Q1 建议值、范围排除、revision/status 与 planned-not-executed 声明；
- bootstrap、recovery bootstrap、后续正常改密三条路径；
- 强制 Gate、当前凭据证明、凭据状态、会话撤销/轮换和失败副作用；
- 唯一公共错误表、接口字段、权限矩阵；
- I-ADM-02/03、覆盖映射、执行 Gate 与未执行声明。

结果：密码规则和最小范围选择本身自洽，但后续正常改密的 Session 前置合同互相矛盾；失败公共错误未闭合；r09 缺少关键安全负例。

### Round 2：真实消费者与上游依据检查

有界反查了：

- requirements r16、identity boundary r11、architecture r23；
- ADR-0005、ADR-0007、ADR-0012；
- `internal/security/password.go`、`internal/session/store.go`、`internal/platform/errors.go`；
- 当前 identity 实现入口；
- `identity-auth-S4.1` 与 B5A/B5B identity lane。

确认：

- 现有密码策略确为至少 8 个字符、同时含字母和数字；
- ADR-0005 要求认证/强制改密等安全上下文变化后轮换并即时失效旧会话；
- 公共注册表已有 `auth.forbidden`、`auth.session_required`、`identity.invalid_state`、`identity.validation_failed` 等稳定 code；
- 当前实现尚无 `PUT /api/v1/users/me/password`，r27/r09 未伪称已经实现或执行；
- Q1 没有扩大到普通用户自助改密、周期轮换、定期过期、密码历史或新复杂度。

## 3. Findings

### [HIGH] 后续正常改密与“只接受当前受限 Session”前置互相矛盾

- **finding ID**：`IA-R27-Q1-R01-F001`
- **对象/修订**：design r27
- **位置**：`design.md:781`、`design.md:1073-1076`、`design.md:1470`、`design.md:2167`、`design.md:2185`
- **违反依据**：requirements 明确初始 `admin` 可通过正常改密流程修改自己的密码；r27 Q1 又明确首次及后续正常自改密复用同一接口。
- **现状**：接口表和状态说明把 `current_password`/改密限定为“当前受限 Session”，但同一修订又允许已完成首次 Gate、因此不再受限的初始 admin 后续正常自改密。
- **触发**：初始 admin 已完成首次改密并取得非受限 permanent Session，之后调用同一接口正常改密。
- **影响**：按“受限 Session”实现会错误拒绝 Q1 明确要求的后续改密；按后续改密段落实现则会违反接口前置和状态机文字。核心 Q1 行为无法得到唯一实现。
- **修复建议**：把合同统一为“当前 Session 记录的 credential kind/id”，并明确允许主体为：任何处于强制改密状态的有效账号，以及非强制状态下的初始 admin；其他非强制账号返回 `auth.forbidden`。同步修正 §3.2、请求字段、状态机与权限矩阵中的“受限 Session”表述。
- **复审证据**：三条路径分别具有唯一前置：bootstrap restricted、recovery-bootstrap restricted、initial-admin permanent unrestricted；普通成员/其他管理员 permanent unrestricted 均明确拒绝。

### [MEDIUM] 改密失败条件没有绑定唯一公共错误合同

- **finding ID**：`IA-R27-Q1-R01-F002`
- **对象/修订**：design r27
- **位置**：`design.md:781`、`design.md:841-876`、`design.md:1069-1076`、`design.md:2167`
- **违反依据**：design 声明公共错误表是本特性的唯一注册表，正文不得使用表外 code；公共接口必须给消费者稳定、唯一的失败语义。
- **现状**：普通成员非强制改密已明确为 `auth.forbidden`，但错误的 `current_password`、Session credential kind/id 已失配或失效、以及 `new_password` 策略失败没有逐项绑定 HTTP status、稳定 code、`field_errors` 和失败后的 Session 处理。
- **触发**：当前密码错误；临时凭据登录后被重发、supersede、过期或恢复失效；新密码少于 8 位、缺字母或缺数字。
- **影响**：实现可能在 `auth.login_failed`、`auth.session_required`、`identity.invalid_state`、`identity.validation_failed` 间分叉，客户端和测试无法形成唯一断言，并可能错误保留可继续使用的陈旧受限 Session。
- **修复建议**：不新增范围外能力，逐项选择现有注册表中的唯一 code/status/field error，并明确失败时凭据、Gate、当前及其他 Session 是否保持或撤销；credential kind/id 失配且要求重新登录的路径还应明确当前 Session 的失效语义。
- **复审证据**：每个失败原因在接口章节、公共错误表和状态机中一一对应，且不存在同一原因多个公共 code。

### [MEDIUM] r09 未验证当前凭据证明、陈旧凭据绑定和范围拒绝

- **finding ID**：`IA-R27-Q1-R01-F003`
- **对象/修订**：test-plan r09
- **位置**：`test-plan.md:213-214`、`test-plan.md:324`、`test-plan.md:470-480`
- **违反依据**：R-I01/R-I02 为 P0；design 要求只验证当前 Session 记录的 credential kind/id、不回退其他凭据，并拒绝普通成员非强制自助改密。
- **现状**：I-ADM-02 覆盖新密码复杂度、Gate 与成功会话轮换，但没有动作/断言覆盖错误 `current_password`、credential kind/id 失配、重发/supersede/过期/恢复失效后的旧受限 Session、永久密码回退禁止，以及普通成员和其他管理员非强制状态调用接口的 `auth.forbidden`。
- **触发**：实现漏验当前密码；使用陈旧临时凭据 Session 改密；普通成员通过当前 `AccessChangePassword` 接入能力调用接口。
- **影响**：即使按 r09 全部通过，仍可能存在无需当前密码即可改密、陈旧凭据完成改密、或意外扩大普通用户自助改密的安全回归。
- **修复建议**：把 I-ADM-02 拆成可执行表驱动矩阵，分别覆盖 bootstrap、recovery bootstrap、initial-admin permanent 后续改密；增加错误当前密码、kind/id 失配及各失效状态、无回退、普通成员/其他管理员非强制拒绝，并对每个失败断言 DB/Gate/当前及其他 Session 快照不变或按 F002 的合同失效。成功路径继续断言其他会话撤销、旧当前 Cookie 失效、新 Cookie/CSRF 可用。
- **复审证据**：测试计划写出精确前置、动作、公共 code/status、持久化与 Session 后置；仍标记 planned/not executed。

## 4. 非 Finding 的对账与交付风险

- boundary r11、architecture r23、ADR-0007 仍把 Q1 标为待确认；tasks 的 active `identity-auth-S4.1` 仍写 Q1 未确认，Plan 仍引用旧的非 S03 identity 基线。r09 已明确写明“待 R 对账”，因此本轮不把该事实误报为 r27/r09 已同步；但在修复并复审通过后，必须先完成上游与 tasks/Plan 对账，不能直接用当前 active 卡关闭 Q1。
- 当前实现只有既有密码策略、Session Gate/Rotate 和公共错误基础，尚无改密 endpoint/service；这与 r27/r09 的“待实施、未执行”声明一致。
- Q1 的建议值没有引入普通用户自助改密、周期轮换、定期过期、密码历史或额外复杂度，最小范围判断通过。

## 5. Gate 结论

### Design Review Gate：NOT PASS

存在 1 个 HIGH 和 2 个 MEDIUM 开放 finding，不能给出 `Design Review Gate PASS（仅针对固定对象及 Q1 增量）`。

## 6. 未验证风险与复审条件

- 按授权未运行 Go test、race、vet、build、PostgreSQL、浏览器、部署或 E2E；
- 未验证尚不存在的改密实现，仅检查当前消费者能否承接设计；
- 未写 checkpoint，未修改 design、test-plan、tasks 或 Plan；
- 复审须固定修订后 design/test-plan 完整 SHA256，逐项回应 F001～F003，并提供 Q1 上游与 `identity-auth-S4.1` R 对账状态；仍不得把 planned test 写成已执行。
