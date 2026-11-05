# identity-auth test-plan r14 独立评审 R01

- Reviewer：add-reviewer，非作者独立评审
- 日期：2026-10-24
- objectType：design（test-plan）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：**1 HIGH、3 MEDIUM；Test-plan Review Gate: CHANGES REQUIRED**

本报告只评价固定的 test-plan r14 是否能作为后续 Task-Planner R 对账和分层测试实施的稳定输入，不评价当前代码是否已修复。代码 finding `IDENTITY-S10.1-REV-R01-F002` 继续保持 Major OPEN；B6C/S10.1 继续 `[-] blocked-for-design-revision` / implementation stopped；B6D/S11.1 继续 todo、未认领、未启动。本评审不授权实现、测试执行、任务状态变更或真实环境验证。

## 1. 固定对象与写前 SHA256

| 对象 | revision / 状态 | 写前 SHA256 |
|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `identity-auth-test-plan-ir01-r14`，`planned-not-executed` | `5B2858DB38E72CC23D65AED29FBBE7A3E6E7A9DDB18083E691BDDD5A7A2B427E` |
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r27` | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r39`，draft | `71EEC596875EE9CBF44CF300C72A4203F5F2C8976522F478A1857D882AE86230` |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r15` | `715F5BEB95FC7AF141F7F9ED18D900979921C520D7E11A857AC017F65C481803` |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `identity-auth-boundary-ir01-r13` | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` |
| `.spec/wip/identity-auth/review-design-r39-s10.1-r03.md` | Design R03，ADR/Design Gate PASS | `CAE91411FD4D126BF033896186D761756C41AC1E7C8B6810FD36F34F8601B0B2` |

写前六个固定对象均与用户指定值一致；目标报告写前不存在。工作树存在大量既有合法改动，本轮未修改、格式化或回滚任何被审对象及其他文件。

## 2. 两轮实际检查范围

### Round 1：固定 test-plan 直接检查

完整读取 test-plan r14，并按 design 分支检查：

1. metadata、revision、source、review、status 与 R03 PASS 的使用边界；
2. I-PRD-01～20 的唯一性、前置、输入/动作、预期、证据、清理和自动/人工边界；
3. purpose 分流、successor generation、same-key replay、User-first 五对象原子性、六字段绑定、四个 commit 点、唯一发送权、provider evidence、old/new settlement；
4. Login、PasswordChange、Session resolve Gate、Session commit barrier、worker/租约/dead-letter/受控命令、双 worker lifecycle；
5. migration/preflight/混跑/不可回退、安全、审计、可观测、公共合同；
6. 规划、现有制品、历史证据、未执行/受阻、性能阈值 TBD 与任务建议边界。

I-PRD-01～20 在场景主表中各出现一次，共 20 个唯一 ID；各行形式上均提供了前置、动作、预期、证据及清理/自动边界。r14 也正确声明 R03 PASS 不等于实现或测试通过，并保留代码 F002、B6C、B6D 的状态边界。

### Round 2：有界上游依据与真实消费者深审

完整或定向核对：

- accepted ADR-0011、ADR-0015、ADR-0018、ADR-0023 及 requirements 中临时凭据、重置和重发合同；
- architecture r27、design r39、stories r15、boundary r13、Design R03；
- 现役 tasks/Plan 中 S1.5、S1.6、S2.2、S3.4、S5.2、S5.3、S10.1、S11.1、B6C/B6D 的实际状态和关闭边界；
- `internal/identity/password_reset.go`、Login、PasswordChange、Session/idempotency、现役 migrations 与 `cmd/han-server` composition/lifecycle，只用于确认真实改造面和“现有制品不等于 r39 已实现”；
- 当前 migration 仅到 `0007`，尚无 r39 operation/job/auth-generation schema，符合 r14 对 durable 实现未完成的总体判断。

未扩成全仓泛审，未运行 Go、PostgreSQL、SMTP、部署或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。

## 3. Findings

### [HIGH] I-CRED-06 未按 purpose 限定，直接否定 ADR-0023 的 accepted settlement

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F001`
- 对象/修订：test-plan `identity-auth-test-plan-ir01-r14`
- 位置：
  - `test-plan.md:134`
  - `test-plan.md:297`
  - `test-plan.md:498`
  - 对照 `test-plan.md:377-380`
  - 对照 `design.md:2097-2099,2160`
  - 对照 `architecture.md:567-569`
  - 对照 ADR-0023 `identity.md:281-310`
- 现状：I-CRED-06 不限定 `new_administrator` 或 ADR-0011 路径，统一规定“SMTP accepted 后数据库提交失败/崩溃”时邮件中的值不可用；US-IDENTITY-03.11 的覆盖映射又同时引用 I-CRED-06 和 I-PRD-01～10，任务映射把 I-CRED-06 交给 S10.1/S11.1/S5.3。
- 违反依据：普通用户 `password_reset` 已由 ADR-0023 改为 SMTP 前持久 operation/job、accepted evidence 后由请求或 worker settlement。accepted/结果提交不确定时必须保持 unresolved Gate，最终可能由 succeeded settlement 激活该邮件中的 new credential；只有无 durable accepted evidence 的 stale candidate 才失效。ADR-0011 的“DB 提交失败则邮件值不可用”残余窗口只能继续适用于 `new_administrator` 路径。
- 触发：Task-Planner 或测试实现者按 I-CRED-06 对普通用户首次 reset/resend 注入 accepted-result commit-ack-loss，并把“邮件值永不可用”作为唯一通过条件。
- 影响：同一 test-plan 对普通 password-reset 同时要求“accepted 可由 worker 收敛成功”和“accepted 后 DB 失败时邮件值不可用”，形成不可同时满足的 P0 预期。后续可能错误拒绝合法 succeeded settlement、把 ADR-0023 降回 ADR-0011 residual-window，或让不同测试按相反结论通过。
- 修复建议：
  1. 将 I-CRED-06 明确限定为 `new_administrator` 和 ADR-0011 residual-window；
  2. 普通用户 `password_reset` 的 accepted/unknown/commit-ack-loss 只由 I-PRD-07～10 及对应 migration/Gate 场景定义；
  3. 同步 §3 风险映射、§5 覆盖映射、§9 S5.3/S10.1/S11.1 任务映射，避免同一 US 引用互斥预期。
- 证据：I-PRD-07 要求 accepted evidence 进入 settlement，I-PRD-09 要求 succeeded settlement 激活 new；而未限定 purpose 的 I-CRED-06 要求邮件值不可用。固定设计和 ADR-0023 明确 operation 是普通 password-reset 的安全权威。
- 复审条件：修订版须以两个 purpose 的独立场景和映射证明零串线；普通 password-reset 的 committed/not-committed/unknown 必须只有一套一致终态期望。

### [MEDIUM] stale/pre-acceptance 恢复缺少保护窗口边界及失败终结后的真实放行证明

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F002`
- 对象/修订：test-plan `identity-auth-test-plan-ir01-r14`
- 位置：
  - `test-plan.md:376-381`
  - `test-plan.md:527,551-554`
  - 对照 ADR-0023 `identity.md:303-309`
  - 对照 `design.md:2097-2099,2160,2337-2340`
  - 对照 `stories.md:349-350,384-385`
- 现状：I-PRD-06 覆盖 dispatch ownership，I-PRD-10 从“已经 stale”开始验证只失效 new，I-PRD-11 独立说明“Gate clear 才沿既有登录合同”。没有场景在同一恢复链中断言：
  1. initial job 的 `next_attempt_at` 必须晚于 SMTP 最迟结束时刻与安全余量，安全时点前 worker 不能领取、接管或终结 candidate；
  2. 明确未受理或 stale 无 accepted evidence 安全终结后，operation/job terminal 且无 hold，Gate 实际清除；
  3. old credential 随后能按原规则登录，既有有效 Session 能正常 resolve，而不是只检查数据库行“保持不变”。
- 违反依据：ADR-0023 明确 job 在 SMTP 最迟结束时刻与安全余量前不得领取；design/stories 明确 failed/invalidated 安全终结后 old 恢复原规则。add-test-design 要求关键验收有可观察证据，不能以分离 fixture 的“Gate clear 时可登录”代替真实失败链收敛。
- 触发：worker 的扫描条件提前、时钟边界错误，或 failed/invalidated 已写入但 Gate 查询仍把该用户永久视为 unresolved。
- 影响：现有场景仍可能全部通过，却允许 worker 在 SMTP 尚可能完成时过早失效 candidate，或允许已安全失败的普通用户长期全局拒绝；这是现实并发/恢复条件下的测试盲区。
- 修复建议：扩充 I-PRD-06/10/11 或增加等价场景，使用可控 Clock 覆盖保护窗口前一刻、边界、边界后一刻；在同一 case 中断言 lease/claim 次数、operation/job 终态、hold、Gate 查询、old credential 登录及既有 Session resolve。不得用 sleep。
- 证据：固定设计明确 `next_attempt_at` 和失败终结后的 old 认证语义；r14 当前只断言 `old保持`、`Mailer=0` 和独立的“Gate clear”条件。
- 复审条件：修订后须能证明 worker 不会过早领取，并证明 failed/invalidated 的正常 clear 不形成永久拒绝。

### [MEDIUM] 未变 F005/S5.2/B6B 的当前事实仍停留在旧阶段，不能作为稳定 R 对账输入

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F003`
- 对象/修订：test-plan `identity-auth-test-plan-ir01-r14`
- 位置：
  - `test-plan.md:490,509-510`
  - `test-plan.md:596,601,603`
  - `test-plan.md:621-625`
  - `test-plan.md:689,711,718-724,736`
  - 对照 `tasks.md:307-309`（S3.2 CLOSED、F005 CLOSED）
  - 对照 `tasks.md:643-657,689-696`（S5.2/I-S52-01～09 CLOSED）
  - 对照 `tasks.md:779-799`（B6B/S8.1 CLOSED）
- 现状：r14 声明未变场景继续复用，但仍把 production composition 写成 fixed-fail、S3.2 写成 current active 且 F005 待复审、S5.2 的 F003/effective-at 写成未实施/blocked，并写“本轮不启动 B6B”。现役 tasks SSOT 已明确 S3.2/F005、S5.2/I-S52-01～09、B6B/S8.1 均已完成独立 Review 和 Lead Gate 后关闭。
- 违反依据：add-test-design 要求“场景 → 原任务/case → 实际证据”映射引用真实原账，计划、历史证据、当前状态和未执行项分开；用户也明确要求不得错误重写 Q1/F005/S5.2/observability 历史状态。
- 触发：Task-Planner 以 r14 的“current active / 未实施 / 待回填”文字进行 R 对账。
- 影响：可能重复规划或重开已关闭写集，混淆历史 review r03 的“当时 OPEN”和现役 task closure，且削弱 r14 作为 password-reset R 对账稳定输入的可信度。真实 SMTP、Host、部署、E2E 和 production readiness 仍未完成，但这不等于 S3.2/S5.2/B6B 未关闭。
- 修复建议：
  1. 将历史报告的当时状态明确标为 historical；
  2. 同步现役 tasks/Plan 的 S3.2、S5.2、B6B closure；
  3. 只保留其真实未验证边界（SMTP/TLS、部署、E2E、容量、production readiness）；
  4. 删除或改写会要求 Task-Planner 再次回填已关闭 application work 的当前态措辞。
- 证据：tasks SSOT 的关闭条目、最终 review/Lead Gate 和 r14 的 current-state 文字直接冲突。
- 复审条件：修订版逐项区分“历史报告当时状态”“现役 task closure”“仍未完成的独立真实环境 Gate”，且不重新占用已释放写集。

### [MEDIUM] “真实 provider/外部收件”被映射给无法产生该证据的 S5.3

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F004`
- 对象/修订：test-plan `identity-auth-test-plan-ir01-r14`
- 位置：
  - `test-plan.md:377,390`
  - `test-plan.md:528-529,554,568-573`
  - `test-plan.md:585,635,697,729`
  - 对照 `tasks.md:698-714`
- 现状：r14 一方面正确规定组件层只用本地替身，另一方面把“真实 DNS/TCP/TLS/AUTH/provider 受理、证书和外部收件”“真实 SMTP/provider”统一归给 S5.3。现役 S5.3 的固定文件是本地 `smtp_test_server.go` 与测试证书，关闭条件明确是可控 SMTP 且“不发外部邮件”；r14 自己的任务建议也以“可能发外部邮件”为停止条件。
- 违反依据：add-test-design 要求 mock/fake 不得冒充真实联通，任务建议必须有真实可执行的所有权、依赖和通过证据；尚未登记或无人承接的范围必须明确为待协调，不能称已安排。
- 触发：S5.3 使用本地 SMTP server 完成真实协议/TLS 故障注入后，被当作真实 provider acceptance 或外部收件已验证。
- 影响：会把生产 provider、账号/网络策略、真实受理语义和外部收件这一独立环境证据错误关闭；或者反过来要求 S5.3 越过既有“不发外部邮件”的安全边界。
- 修复建议：
  1. 明确 S5.3 只证明 production-equivalent SMTP/TLS 协议与可控 accepted/未受理/unknown；
  2. 将真实获批准 provider、真实账号/DNS/TCP/AUTH、外部收件与展示标为独立部署/E2E 证据，绑定真实责任人、Case/Run 和停止条件；若首期明确不要求真实外部发送，也应由有权责任人明确接受该限制，而不是称其已由 S5.3 覆盖。
- 证据：r14 的真实 provider 归属与 S5.3 现役文件、关闭条件和“不发外部邮件”边界不能同时成立。
- 复审条件：修订后本地 SMTP/TLS integration 与真实 provider/外部收件使用不同证据标签和 owner，任何一方不得替代另一方。

## 4. 已确认覆盖与未发现新增缺陷的部分

除上述 findings 外，本轮确认：

1. metadata、固定上游 SHA、`planned-not-executed` 和 Design R03 使用边界正确；
2. I-PRD-01～20 无 ID 冲突或重复主表定义；
3. operation 仍是普通 password-reset 的唯一安全权威，same-key replay、IdempotencyRecord、consumer 或 worker 均未在新增场景中获得绕过权；
4. 五对象原子性、六字段绑定、User-first 锁序、no-row barrier、caller-owned Session resolve transaction、commit 后 Principal 均有规划；
5. old active/new pending、succeeded settlement、generation/lineage/hold、consumed 竞态、dead-letter、受控命令、双 worker lifecycle、migration/preflight/混跑/不可回退均有场景和证据类型；
6. 规划、当前 durable 制品缺口、R03 历史设计 PASS、代码 F002 OPEN、真实环境未执行总体上有明确边界；
7. 性能阈值保持 TBD，没有编造容量或生产就绪结论；
8. r14 没有新增 task、改变 tasks/Plan 状态、关闭代码 F002、解除 B6C 或启动 B6D。

## 5. 待决定

1. **真实 SMTP provider / 外部收件是否为本阶段必需 Gate，以及由谁执行。** 当前资料只能支持本地可控 SMTP/TLS integration；真实 provider 账号、网络、处理方批准、外部收件和清理责任没有与 S5.3 相容的执行合同。责任方需选择：
   - 另设或绑定现有部署/E2E Case/Run 承接真实 provider；或
   - 明确首期只要求 production-equivalent 本地 SMTP/TLS，并把真实 provider 保持为未验证风险。

Reviewer 不代替责任方作该采用决定。

## 6. 非阻断建议

1. 修订时保留 I-CRED-06 作为 ADR-0011 历史场景，不必删除；通过 purpose 标签、场景名称和覆盖映射消除串线，避免破坏既有新管理员回归。
2. 对所有“真实”证据使用稳定标签，例如 `deterministic fake`、`local real-protocol integration`、`real PostgreSQL 18`、`approved-provider deployment`、`external-receipt E2E`，减少后续把不同层次合并为 PASS 的风险。

## 7. ADR 一致性与 Test-plan Review Gate

### ADR 一致性

accepted ADR-0023 的方向未被 test-plan 整体替换，I-PRD-01～20 的主体仍按方案 A 规划 durable operation/job/Gate/worker。问题是 I-CRED-06 的旧无 purpose 合同与该方向发生直接冲突。

**ADR 一致性结论：受 F001 限制；ADR 本身不重开，需修正 test-plan 对 ADR-0011/ADR-0023 的消费。**

### Test-plan Review Gate

- HIGH：1 OPEN
- MEDIUM：3 OPEN
- CRITICAL / LOW：0

**Test-plan Review Gate: CHANGES REQUIRED**

理由：F001 使同一 P0 password-reset accepted/commit-unknown 路径出现互斥预期；F002 留下保护窗口和正常 clear 的关键盲区；F003/F004 使 Task-Planner 的现役状态与真实环境证据所有权不稳定。当前 r14 不足以直接交给 Task-Planner 作最终 R 对账。

该 Gate 不改变：

- `IDENTITY-S10.1-REV-R01-F002`：Major OPEN；
- B6C/S10.1：`[-] blocked-for-design-revision` / implementation stopped；
- B6D/S11.1：todo、未认领、未启动。

## 8. 未验证风险与限制

- 未运行任何 Go test、race、vet、build 或 Manifest 检查；本对象是文档设计制品，不以代码 build 条款替代设计评审。
- 未连接 PostgreSQL，未验证真实约束、锁序、commit actual state、租约、多 worker、migration、混跑或回退。
- 未连接 SMTP/TLS/provider，未验证证书、AUTH、受理、外部收件或保护窗口。
- 未启动服务、浏览器、部署、E2E、告警平台或性能环境。
- 未执行 PlantUML 渲染。
- 当前代码和 migrations 只作为真实消费者/改造面证据；不能据此降低已确认设计预期，也不表示 r39 已实现。
- 工作树有大量既有未提交内容；本轮只新增本报告，未评价无关改动。

## 9. 复审条件与 Lead 下一步

复审需固定新的 test-plan revision/SHA，并至少证明：

1. F001：I-CRED-06 已限定 `new_administrator`，普通 password-reset accepted/unknown 只有一套 ADR-0023 预期；
2. F002：保护窗口前/边界/后及 failed/invalidated 后 Gate clear、old login、Session resolve 已形成同链证据；
3. F003：S3.2/F005、S5.2、B6B 的现役 closure 与剩余真实环境 Gate 分离准确；
4. F004：本地 SMTP/TLS 与真实 provider/外部收件的 owner、Case/Run 和证据标签不再混用；
5. 修订未弱化 I-PRD-01～20 其他已确认覆盖，未关闭代码 F002、解除 B6C 或启动 B6D。

Lead 下一步应先现场核本报告、固定对象和唯一写入，再将 findings 交 test-plan 原作者返工；返工后由非作者 Reviewer 定点复审。Test-plan Gate 未 PASS 前，不应启动 Task-Planner R 对账。

## 10. 写后验证

写后将执行：

- 六个固定对象 SHA256 漂移复核；
- 本报告定向 `diff --check`；
- 唯一写入归属核对；
- 本报告 SHA256 计算。

最终结果与报告 SHA256在完成返回中登记；本文件不嵌入自身最终哈希，避免自指修改导致哈希再次变化。
