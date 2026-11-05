# identity-auth design r28 / test-plan r11 Q1 独立复审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **objectType**：design；test-plan 与交付同步作为直接消费者检查
- **分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **复审输入**：`.spec/wip/identity-auth/review-design-r28-q1-r02.md`
- **结论**：**PASS / Design Review Gate PASS**
- **原 finding 结论**：`IA-R28-Q1-R02-F001` CLOSED；`IA-R28-Q1-R02-F002` CLOSED
- **新 findings**：无

## 1. 固定对象与开始哈希

| 对象 | revision / 状态 | 开始 SHA256 |
|---|---|---|
| `.add/project/knowledge/design/identity-auth/boundary.md` | r12 / draft | `335CC44131B542E5E3EEFCCBABBA443160AC8B0B240FE2BBAB0E849D50992591` |
| `.add/project/knowledge/design/architecture.md` | r24 | `FAC19BE91763855EC4CE91D97B8AC052A824AAC643794E4B0F45A01EE04C586B` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | r28 / draft | `977435C8391E4A21C070C359AFBB6F143EC84E4CA3EE19CBEC7FC5FCC9C0E6B0` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | r11 / planned-not-executed | `C49D87ED526D897CBF18CDE7E338F44A845E85CC1E0412A90E2A9169367B2D88` |
| `.add/project/knowledge/tasks/tasks.md` | tasks SSOT | `292DE0E5306A222D04E83D8510E24347B5FBB814AC96C62D2061690D38E20EF3` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | active Plan | `CD371BA3FC5824EA61676CA3F9F9A7872070B463ADD3BE333267D47E3F01E385` |
| `.spec/wip/identity-auth/review-design-r28-q1-r02.md` | 原复审报告 | `51163618F281FE40524CCB4A1938E7FBE01D8A806285D2C7BF80479B3FBF8CF1` |

开始复核时全部固定对象与指定 SHA256 一致。

## 2. 两轮实际检查范围

### Round 1：原 findings 的直接修订检查

围绕 r02 两项 finding 检查：

- S4.1 是否取得 Q1 production registration 所需的 `cmd/han-server/main.go`、`main_test.go`；
- production 顺序是否唯一固定为 Trace、exact path/method router、Origin、Session、CSRF、授权/Password Handler；
- JSON 404/405、405 `Allow`、trace 和错误分支零下游调用是否保持既有合同；
- closed S3.2 是否只顺序移交文件而不重开或继续占有；
- test-plan r11 是否把 architecture r24、boundary r12 的 revision、完整摘要及 Q1 追溯用途同步到现役来源；
- r10 的 Q1 表驱动矩阵和 `planned-not-executed` 状态是否保持不变。

### Round 2：直接消费者与交付同步检查

有界反查：

- boundary r12、architecture r24、design r28 的 Q1 主体、错误、凭据绑定与 Session 后置；
- tasks 的共享所有权、S4.1 文件清单、production registration、验收命令和关闭条件；
- Plan 的 B5A+B5B identity lane、production router owner、S3.2→S4.1 顺序移交与停止条件；
- 当前 `cmd/han-server/main.go` 的 production composition/local router，确认 S4.1 所获两文件正是新增 Q1 route 和 wiring 的真实接入点；
- 当前 Session、CSRF、公共错误和密码策略消费者能否承接已确认合同。

检查未扩展为全仓预审；历史修订中的 r10/r03 状态记录按历史事实保留，不作为当前漂移。

## 3. 原 finding 逐项结论

### IA-R28-Q1-R02-F001 — CLOSED

原问题是 S4.1 无权修改 production composition/router，导致改密实现即使落入 identity 包也无法从生产入口调用。

当前修订已闭合该断点：

- `tasks.md:27` 明确 `cmd/han-server/main.go`、`main_test.go` 从 closed S3.2 顺序移交给 active S4.1；S3.2 不重开、不再占有或写入；
- `tasks.md:472-479` 把两文件纳入 S4.1 写集，并把 S4.1 定为 exact `PUT /api/v1/users/me/password` production registration 的 owner；
- `tasks.md:479` 固定唯一顺序：`TraceMiddleware` 最外层 → local router exact path/method → Origin → Session → CSRF → 主体授权/Password Handler；
- 同一条款固定未知 path、错误 method 使用 `platform.WriteError` JSON 404/405，405 `Allow` 与登记 method 一致，既有登录的 JSON fallback、`Allow: POST`、trace 和错误分支零下游调用不得回退；
- `tasks.md:480-489` 增加 `cmd/han-server` 回归命令，并继续绑定 r28/r11 的 Q1 主体、403/401/422、Gate 与 Session 后置；
- Plan `:522-525`、`:680`、`:732` 使用相同 owner、顺序、fallback 和串行交接合同；S4.1 完成开发、Review、Lead 交接后，S7.1 才可占用共享 `handler.go`。

当前 production router 仍只登记登录入口，这是 S4.1 尚待实现的基线事实，不再是任务写集缺口。S4.1 已获得完成 Q1 接线所需的真实 composition root，且未借此重开 S3.2。

### IA-R28-Q1-R02-F002 — CLOSED

test-plan r11 已完成上游来源同步：

- frontmatter `test-plan.md:13-16` 固定 architecture r24、boundary r12 及其完整 SHA256；
- `test-plan.md:27-41` 明确 r11 只修复该追溯 finding，并说明两份上游用于 Q1 主体资格、当前 Session credential 绑定、403/401/422 和成功/失败 Session 后置；
- r11 修订记录明确 r10 矩阵、其他测试合同和执行状态不变；
- `test-plan.md:214-268` 的主体准入、三条允许路径、字段失败、credential kind/id 与临时凭据失效、永久密码回退禁止、成功轮换及全部失败后置保持可执行；
- 状态仍为 `planned-not-executed`，没有把来源修订伪称为测试执行。

architecture r24、boundary r12、design r28 与该矩阵对 Q1 业务值和失败后置一致。

## 4. 第二轮合同结论

- 密码规则保持至少 8 位并同时含字母和数字。
- 改密只复用 `PUT /api/v1/users/me/password`。
- 允许主体仍为任何强制改密状态的有效账号，以及非强制状态的初始 admin。
- 其他非强制普通成员或管理员在密码验证前返回 `403 auth.forbidden`。
- 当前密码错误和新密码策略失败返回 `422 identity.validation_failed` 及既定字段错误。
- 陈旧、失配或失效的 Session credential 绑定返回 `401 auth.session_required`，只撤销当前 Session。
- 成功写入永久密码、完成适用强制 Gate、撤销其他 Session 并轮换当前 Session。
- 未扩大普通用户自助改密，也未引入周期轮换、定期过期、密码历史或额外复杂度。
- tasks/Plan 只补足真实接入点的所有权、middleware 顺序和回归责任，没有改写上述业务合同。

未发现新的 CRITICAL、HIGH、MEDIUM 或 LOW finding。

## 5. Gate 结论

### Design Review Gate：PASS

`IA-R28-Q1-R02-F001` 与 `IA-R28-Q1-R02-F002` 均已 CLOSED，且两轮检查未发现新的阻断 finding。

该 PASS 仅适用于本报告固定的 boundary r12、architecture r24、design r28、test-plan r11、tasks、Plan 及 Identity Q1 增量实施 Gate。它只说明设计、测试规划和交付承接已形成无歧义合同，不表示 S4.1 已实现、测试已执行、代码 Review 已通过、已部署、production readiness 或 E2E 已完成。

## 6. 限制与未验证项

- 未运行 Go 单元、集成、race、contract、vet 或 build；
- 未运行 PostgreSQL、浏览器、部署或 E2E；
- 未验证尚未实现的 Q1 endpoint 行为；
- test-plan r11 仍为 `planned-not-executed`；
- 未修改任何被审对象、代码、tasks、Plan 或 Git 状态；
- 未写 checkpoint，未执行 Git add/commit/push；
- 本报告不替代 S4.1 实施后的独立代码 Review、Lead Gate 或真实验证。

## 7. 结束验证

结束时重新计算全部固定对象 SHA256，并对本报告及固定文档执行 scoped `git diff --check`。具体结果和本报告物理文件 SHA256在最终交付回报中给出；物理文件摘要不自嵌回本文件，以免摘要字段改变被摘要对象。
