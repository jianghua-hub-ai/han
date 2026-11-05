# identity-auth design r28 + test-plan r10 Q1 独立复审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **对象类型**：design + test-plan 复审；有界消费者/交付同步检查
- **分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **结论**：**FAIL / Design Review Gate NOT PASS**
- **原 finding 结论**：F001 `CLOSED`；F002 `CLOSED`；F003 `CLOSED`
- **新 finding 统计**：CRITICAL 0 / HIGH 1 / MEDIUM 0 / LOW 1

## 1. 固定对象与开始哈希

| 对象 | revision / 状态 | 开始 SHA256 |
|---|---|---|
| `.add/project/knowledge/design/identity-auth/boundary.md` | r12 / draft | `335CC44131B542E5E3EEFCCBABBA443160AC8B0B240FE2BBAB0E849D50992591` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | r28 / draft | `977435C8391E4A21C070C359AFBB6F143EC84E4CA3EE19CBEC7FC5FCC9C0E6B0` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | r10 / planned-not-executed | `3AD85455E8AFFD2A17A39E92C3F87D6A3327292463150D70753C2CD7D6C1BE87` |
| `.add/project/knowledge/design/architecture.md` | r24 | `FAC19BE91763855EC4CE91D97B8AC052A824AAC643794E4B0F45A01EE04C586B` |
| `.add/project/knowledge/tasks/tasks.md` | tasks SSOT | `CD94B016BF3904702B68B6EECF049500B0EA83452ACFE8CA068A06029F2BE0D4` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | active Plan | `DCCD3020A2F9C00AD9C3CC520C5739A5BF579C4D2C3AE4B2BECD5AC79401017F` |
| `.spec/wip/identity-auth/review-design-r27-q1-r01.md` | 原评审报告 | `E693DEC60DC1DED3314F49C678FB222DBE79B85E17AF105ACE081CD0A51B8736` |

开始复核时七个固定对象均与指定 SHA256 一致；未把工作树中其他人的未提交改动当作本轮授权，也未修改固定对象。

## 2. 两轮实际检查范围

### Round 1：原 finding 的直接修订检查

完整读取 r28、r10 和原报告，逐项检查：

- 三条允许路径与两条拒绝路径是否有唯一前置；
- 当前 Session credential kind/id、当前密码、新密码策略的处理顺序；
- 403/401/422、`field_errors`、凭据、Gate、当前 Session 和其他 Session 后置；
- 成功写永久密码、完成强制改密 Gate、撤销其他 Session、轮换当前 Session；
- 普通成员、其他非强制管理员、周期轮换、定期过期、密码历史和额外复杂度是否仍被排除；
- r10 是否形成可执行断言，且保持 `planned-not-executed`。

### Round 2：有界消费者与交付同步检查

完整读取 boundary r12、architecture r24、tasks、Plan，并按原报告允许范围反查：

- requirements r16、ADR-0005、ADR-0007、ADR-0012；
- `internal/security/password.go`；
- `internal/session/store.go`、identity Session 撤销消费者；
- `internal/platform/errors.go`；
- 当前 identity repository/service/handler；
- `cmd/han-server/main.go` 的现役 production composition 与 local router；
- `identity-auth-S4.1` 的文件写集、关闭条件及 B5A/B5B lane。

本轮没有扩成全库无界审查。

## 3. 原 finding 逐项结论

### IA-R27-Q1-R01-F001 — CLOSED

r28 已消除“只接受受限 Session”与初始 admin 后续正常改密之间的矛盾：

- `design.md:1076-1081` 固定只验证当前 Session 记录的 credential kind/id 所绑定权威凭据；
- 允许任何 `must_change_password=true` 的有效账号，以及非强制状态的初始 admin；
- 非强制普通成员或其他管理员在密码读取/验证前返回 `403 auth.forbidden`；
- boundary r12 `:71-73`、`:280-283` 与 architecture r24 `:497` 使用相同主体和后置合同。

bootstrap forced、recovery-bootstrap forced、initial-admin permanent non-forced 三条允许路径均能得到唯一实现；普通成员和其他管理员的非强制路径均被拒绝。

### IA-R27-Q1-R01-F002 — CLOSED

r28 `design.md:1083-1094` 已逐项固定唯一失败合同：

- 非强制普通成员或非初始管理员：`403 auth.forbidden`，无状态副作用；
- credential kind/id 陈旧、失配或临时凭据已失效：`401 auth.session_required`，只撤销当前 Session；
- 当前密码错误：`422 identity.validation_failed` + `current_password_mismatch`；
- 新密码过短、缺字母、缺数字：`422 identity.validation_failed`，按既有策略顺序只返回首个字段错误；
- 除明确的陈旧绑定 401 外，失败不写密码、不改变 Gate、不撤销 Session；
- 成功写 permanent、完成强制 Gate、撤销其他 Session并轮换当前 Session。

公共错误注册表已有 `auth.forbidden`、`auth.session_required`、`identity.validation_failed`；既有密码策略顺序为长度、字母、数字，与设计一致。

### IA-R27-Q1-R01-F003 — CLOSED

r10 `test-plan.md:220-264` 已把 I-ADM-02 展开为可执行表驱动矩阵：

- 覆盖三条允许路径和两条非强制拒绝路径；
- 覆盖错误当前密码、三类新密码策略错误；
- 覆盖 kind mismatch、非 temporary ID 非空、temporary ID 缺失/非当前值；
- 覆盖 superseded、expired、recovery_invalidated、consumed；
- 明确禁止回退验证永久密码；
- 每例断言 HTTP envelope、verifier 调用次数、凭据/Gate 快照、当前及其他 Session 后置；
- 成功例断言旧 Cookie 失效、新 Cookie/CSRF 可用、其他 Session 全部撤销；
- `test-plan.md:264` 明确这些自动化尚未实现或执行，未伪称执行结果。

## 4. 新 findings

### [HIGH] S4.1 的授权写集不能把 Q1 改密入口接入现役 production router

- **finding ID**：`IA-R28-Q1-R02-F001`
- **位置**：`tasks.md:467-484`；Plan `:522-524`、`:677`；`cmd/han-server/main.go:836-922`
- **现状**：
  - S4.1 的文件清单只有 `internal/identity/bootstrap.go`、`bootstrap_test.go`、`handler.go`、`service.go`；
  - S4.1 关闭条件却要求可调用并复用 `PUT /api/v1/users/me/password`；
  - 现役 `newProductionBusinessHandler` 只装配登录 Handler，`newLoginBusinessRouter` 只接受精确 `POST /api/v1/sessions`，其他路径返回 JSON 404；
  - 当前 Go 实现中不存在改密 endpoint/service；
  - Plan 要求发现卡外文件需求立即停止，并明确已关闭 S3.2 不得继续占用其十八文件；`cmd/han-server/main.go` 正属于该已关闭写集。
- **影响**：实施者无法在 S4.1 当前授权写集内交付可从生产入口调用的 Q1 endpoint。继续实施只能越过文件所有权、留下不可达实现，或借已关闭 S3.2 修改 production composition；因此修订集尚未形成无歧义、可执行的 Identity Q1 实施 Gate。
- **关闭标准**：
  1. 由 Task-Planner/Lead 明确唯一 owner，并把现役 production composition/router 接线所需文件纳入有效任务写集；
  2. 固定 `PUT /api/v1/users/me/password` 的 exact method/path、Session/CSRF/Origin middleware 顺序和 endpoint registration 责任；
  3. 同步 tasks/Plan 的 lane、文件所有权与串行交接，避免与已关闭 S3.2 或已 active S7.1 冲突；
  4. 不改变 r28/r10 已确认的 Q1 行为，也不扩大普通用户自助改密等范围。
- **状态**：`STILL OPEN`。

### [LOW] r10 的上游来源元数据仍停留在 architecture r23 / boundary r11

- **finding ID**：`IA-R28-Q1-R02-F002`
- **位置**：`test-plan.md:13-15`、`:31-38`
- **现状**：r10 正确绑定 design r28 和原报告，但 frontmatter/source table 仍声明 architecture r23 与 identity boundary r11；本轮固定且已同步 Q1 的上游对象是 architecture r24、boundary r12。
- **影响**：不改变 §4.3.1 的测试断言，因此不重新打开 F003；但会使后续消费者无法仅凭 r10 的来源元数据确认其已核对本次 Q1 上游同步，存在追溯漂移。
- **关闭标准**：把 r10 的 architecture/boundary 来源 revision、摘要和用途更新到本轮固定对象；不得重写其他已确认合同或把 planned test 写成已执行。
- **状态**：`STILL OPEN`，非阻断。

## 5. 范围与合同核对

- Q1 建议值保持最小且自洽：至少 8 位并同时含字母/数字，复用一个改密接口。
- 首次 bootstrap、恢复 bootstrap、后续初始 admin 改密、强制 Gate、当前凭据绑定、成功撤销/轮换和失败副作用在 design/test-plan/boundary/architecture 中闭合。
- 未扩大为普通成员或其他非强制管理员自助改密。
- 未增加周期性轮换、定期过期、密码历史或额外复杂度。
- 未发现 r28/r10 对其他已确认技术合同作无关重述性改写；LOW finding 仅涉及来源元数据未跟随固定上游修订。
- 当前 Session Store 的 `Rotate`、identity repository 的按用户撤销 Session 能力和公共错误 code 可承接合同；这不等于 endpoint 已实现或生产接线已完成。

## 6. Gate 结论

### Design Review Gate：NOT PASS

原报告 F001、F002、F003 均已 `CLOSED`，但新增 `IA-R28-Q1-R02-F001` 为未关闭 HIGH finding：S4.1 当前文件所有权与现役 production router 之间存在不可交付断点。

本结论仅针对上述固定对象及 Identity Q1 增量的实施 Gate。不得外推为实现、测试执行、部署、production readiness 或 E2E 完成。

## 7. 限制与未验证项

- 未运行 Go 单元、集成、race、contract、vet 或 build；
- 未运行 PostgreSQL、浏览器、部署或 E2E；
- r10 仍是 `planned-not-executed`，本报告只评审测试设计的可执行性；
- 未修改 design、test-plan、boundary、architecture、requirements、ADR、tasks、Plan 或代码；
- 未写 checkpoint，未执行 Git add/commit/push；
- 未评审 Q1 以外的新能力，也未替代实施后的代码 Review。

## 8. 结束验证

结束时重新计算七个固定对象 SHA256；结果须与 §1 完全一致。对本报告及六个固定交付文档执行 scoped `git diff --check`。具体命令结果和本报告物理文件 SHA256 在完成写入后记录于最终交付回报；物理文件摘要不自嵌回本文件，以避免摘要字段改变被摘要对象。
