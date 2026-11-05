# identity-auth test-plan r15 定点复审 R02

- Reviewer：add-reviewer，原非作者独立 Reviewer
- 日期：2026-10-24
- objectType：design（test-plan）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审目标：仅复审 R01 F001～F004，并检查相关覆盖和状态边界是否回归
- 结论：**F001、F002 CLOSED；F003、F004 OPEN；无新增 finding；Test-plan Review Gate: CHANGES REQUIRED**

本报告只评价固定 test-plan r15 是否已完整关闭 R01 finding，且能否作为后续 Task-Planner R 对账的稳定输入。本报告不评价代码是否修复，不执行测试，也不授权实现或任务状态变化。代码 finding `IDENTITY-S10.1-REV-R01-F002` 继续保持 Major OPEN；B6C/S10.1 继续 `[-] blocked-for-design-revision` / implementation stopped；B6D/S11.1 继续 todo、未认领、未启动。

## 1. 固定对象与写前 SHA256

| 对象 | revision / 状态 | 写前 SHA256 |
|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `identity-auth-test-plan-ir01-r15`，`planned-not-executed` | `63E3D3807F5F09998C05574882BF9D84DE5ABC541CB2E04F72A61F5BD30A9C76` |
| `.spec/wip/identity-auth/review-test-plan-r14-s10.1-r01.md` | test-plan R01，CHANGES REQUIRED | `C77AF5A60C2C35939ED5134E48F823AFCFDF7082F1D3D414D847F4569625CE39` |
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r27` | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r39`，draft | `71EEC596875EE9CBF44CF300C72A4203F5F2C8976522F478A1857D882AE86230` |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r15` | `715F5BEB95FC7AF141F7F9ED18D900979921C520D7E11A857AC017F65C481803` |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `identity-auth-boundary-ir01-r13` | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` |
| `.spec/wip/identity-auth/review-design-r39-s10.1-r03.md` | Design R03，ADR/Design Gate PASS | `CAE91411FD4D126BF033896186D761756C41AC1E7C8B6810FD36F34F8601B0B2` |

写前七个固定对象、分支和 HEAD 均与用户指定值一致；目标报告写前不存在。

## 2. 两轮实际检查范围

### Round 1：固定对象直接检查

完整读取 test-plan r15 与原 R01，并检查：

1. revision、source、review、`planned-not-executed` 和 R03 PASS 的使用边界；
2. R01 F001～F004 的全部原定位、修复建议和复审条件；
3. I-CRED-06、I-PRD-06～11、风险地图、US 映射、覆盖映射、数据环境、自动/人工边界、任务建议、执行顺序和 Gate；
4. I-PRD-01～20 主表定义的唯一性与其前置、动作、预期、证据、清理边界；
5. S3.2/F005、S5.2/I-S52-01～09、B6B/S8.1、S5.3、S3.4 的所有现役/历史/未验证状态文字；
6. 代码 F002、B6C、B6D、性能 TBD、Case/Run 和任务新增边界。

I-PRD-01～20 在主表中仍各定义一次，共 20 个唯一 ID；除本报告指出的两处残留冲突外，r14 已确认的 durable 场景覆盖没有退化。

### Round 2：有界上游与真实账本深审

定向核对：

- architecture r27、design r39、stories r15、boundary r13、Design R03 和 accepted ADR-0011/0023 的 purpose、保护窗口、settlement 与证据分层合同；
- tasks/Plan 中 S3.2/F005、S5.2/I-S52-01～09、B6B/S8.1 的最终独立 Review + Lead Gate closure；
- 既有 S1.5、S1.6、S2.2、S3.4、S5.3、S10.1、S11.1 的任务边界和当前状态；
- S5.3 本地 SMTP/TLS/AUTH 测试边界与 S3.4 部署后真实 provider / external-receipt E2E 的所有权。

本轮未扩成全仓审查，未运行 Go、PostgreSQL、SMTP、部署或 E2E，也未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。

## 3. R01 findings 定点复审

### F001 — CLOSED

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F001`
- 证据：
  - `test-plan.md:304` 将 I-CRED-06 明确限定为 `new_administrator` / ADR-0011 residual-window，并明确禁止用于普通 `password_reset`；
  - `test-plan.md:107-108,139,470,506,731` 在风险、US、接缝、覆盖和结论中保持同一 purpose 限定；
  - `test-plan.md:383-387` 仅以 I-PRD-06～10 定义普通 password-reset 的 dispatch、accepted/not-accepted/unknown、commit-ack-loss、settlement 与 stale 收敛；
  - `test-plan.md:384,386` 明确 durable accepted evidence 可由 succeeded settlement 激活邮件中的 new credential。
- 结论：两个 purpose 已有独立且不冲突的安全预期；普通 password-reset 不再被 I-CRED-06 降级为 residual-window。**F001 CLOSED。**

### F002 — CLOSED

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F002`
- 证据：
  - `test-plan.md:383` 要求 initial job 的 `next_attempt_at` 严格晚于 SMTP 最迟结束时刻加安全余量；
  - `test-plan.md:387` 使用可控 Clock、禁止 sleep，覆盖保护窗口前、精确边界、边界后以及 lease 过期接管；
  - 同一场景明确断言 lease/claim/takeover 次数、operation/job terminal、无 integrity hold、Gate clear、old credential 登录、既有 Session resolve、Mailer=0 和终态 replay；
  - `test-plan.md:388` 复用 I-PRD-10 收敛后的同一 User 验证 Login 正常放行；
  - `test-plan.md:560-561` 固定 barrier/channel、边界 Clock 和 worker 计数，禁止 wall-clock sleep。
- 结论：已形成“保护窗口不早领 + failed/invalidated 正常 clear 不永久拒绝”的同链可观察证据。**F002 CLOSED。**

### F003 — OPEN（MEDIUM）

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F003`
- 已正确修订：
  - `test-plan.md:57,587,606-610,628-630,703-713` 已把 S3.2/F005、S5.2/I-S52-01～09、B6B/S8.1 写为最终独立 Review + Lead Gate CLOSED，并把真实环境未验证单列；
  - `test-plan.md:494-520` 的覆盖映射也主要使用现役 closure 状态。
- 未关闭残留：
  - `test-plan.md:185` 仍把 I-F005-13～22 的当前“状态”写为“计划/未执行”，并将 I-F005-23～29 表述为新增切片，未在该场景入口同步其已随 S3.2 最终 closure 闭环的现役事实。
- 触发：Task-Planner 或后续读者从 §4.2.1 场景入口读取 F005 状态，而非继续读到后部 closure 表。
- 影响：同一 r15 同时给出“计划/未执行”和“application CLOSED”两个当前状态，仍可能造成重复对账、重开已释放写集或误判 F005 尚待实施。
- 修复建议：将 `test-plan.md:185` 改为明确的历史/现役分栏：场景最初是规划，但 I-F005-13～29 现已由 S3.2 最终 Review + Lead Gate CLOSED；本 r15 未复跑，真实容量、平台和部署 Gate 仍独立未验证。
- 复审条件：全文不再把 I-F005-13～29 的 application 当前状态写为 planned/unexecuted，且继续保留“本 r15 未复跑”和真实环境未验证边界。**F003 OPEN。**

### F004 — OPEN（MEDIUM）

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F004`
- 已正确修订：
  - `test-plan.md:155,563-564,595,600,629,640-641,680-681,689,691` 已建立稳定证据标签；
  - S5.3 被限定为 `local real-protocol integration` 且不发外部邮件；
  - S3.4 承接 `approved-provider deployment` / `external-receipt E2E`，并明确部署版本、审批、责任人、Case/Run、数据与清理前置以及 `pending coordination`。
- 未关闭残留：
  - `test-plan.md:537` 仍写“真实 SMTP/TLS、provider 证书和网络行为……以 S5.3 为唯一后续验证”，与上述 S5.3/S3.4 分层直接冲突。
- 触发：测试实施者按数据与环境计划把真实 provider 证书/网络行为全部归入 S5.3。
- 影响：可能要求 S5.3 越过“不发外部邮件”的边界，或用本地协议证据替代获批 provider deployment / external receipt。
- 修复建议：把 `test-plan.md:537` 限定为 S5.2 deterministic Mailer 不覆盖本地真实协议；本地 SMTP/TLS/AUTH/测试证书归 S5.3，真实 provider 账号、证书链/网络、受理和外部收件归 S3.4。
- 复审条件：全文对 provider/网络证据只保留一套所有权，S5.3 与 S3.4 的证据不得互相替代。**F004 OPEN。**

## 4. 新增 findings 与回归结论

### 新增 findings

无。上述两项均是原 F003/F004 的残留，未另建 finding ID。

### 已确认未退化

1. I-PRD-01～20 的其他 purpose、generation/replay、五对象原子性、六字段绑定、Gate、settlement、worker、migration、混跑、回退、安全、审计与生命周期覆盖保持完整；
2. test-plan 仍为 `planned-not-executed`，没有把 Design R03 PASS 写成实现或测试通过；
3. 性能和容量数值继续为 TBD；
4. 没有新增或修改 task、Case/Run、Plan 或任务状态；
5. 代码 F002 保持 Major OPEN，B6C 未解除，B6D 和 Task-Planner 均未启动；
6. Q1、S5.2 与 observability 的独立真实环境边界未因 application closure 被错误关闭。

## 5. 待决定与非阻断建议

### 待决定

无。r15 已采用“本地真实协议归 S5.3、获批 provider/外部收件归 S3.4”的方向；当前问题是全文残留冲突，不需要 Reviewer 代替责任方选择新方案。

### 非阻断建议

1. 返工后对 `planned`、`未执行`、`provider`、`S5.3`、`S3.4` 做一次全文一致性扫描，避免前部场景说明和后部 Gate 表再次漂移。
2. 继续使用 `deterministic fake`、`local real-protocol integration`、`real PostgreSQL 18`、`approved-provider deployment`、`external-receipt E2E` 五类稳定标签。

## 6. ADR 一致性与 Test-plan Review Gate

### ADR 一致性

F001 已关闭，r15 对 accepted ADR-0011/ADR-0023 的 purpose 分流和 durable settlement 方向正确；本轮没有重开或破坏 ADR-0023。F003/F004 是测试计划内部的现役状态与证据所有权残留，不改变 ADR 决定本身。

### Test-plan Review Gate

- MEDIUM：2 OPEN（原 F003、F004）
- CRITICAL / HIGH / LOW：0 OPEN
- 新增 finding：0

**Test-plan Review Gate: CHANGES REQUIRED**

理由：r15 的主体修订已关闭 F001/F002，也在后部多数映射中正确处理 F003/F004；但 §4.2.1 仍把已关闭 F005 application 场景写成计划/未执行，§7 仍把 provider 证书和网络行为指向 S5.3。两处都位于后续执行者会直接消费的场景/环境合同中，使同一文档保留互斥当前状态或 owner，尚不足以作为稳定的 Task-Planner R 对账输入。

该 Gate 不改变：

- `IDENTITY-S10.1-REV-R01-F002`：Major OPEN；
- B6C/S10.1：`[-] blocked-for-design-revision` / implementation stopped；
- B6D/S11.1：todo、未认领、未启动；
- Task-Planner：本报告不启动、不授权。

## 7. 未验证风险与限制

- 未运行代码 build、test、race、vet 或 Manifest 检查；本对象是 test-plan 设计制品。
- 未连接 PostgreSQL，未验证 durable schema、事务、锁、commit actual state、worker lease、migration、混跑或不可回退。
- 未连接本地或外部 SMTP/TLS/provider，未验证证书、AUTH、受理、保护窗口或外部收件。
- 未启动服务、浏览器、部署、E2E、告警平台或性能环境。
- 未创建或执行 Case/Run，也未验证清理责任。
- 当前代码和 migration 仅作为真实消费者/缺口事实，不表示 r39/r15 已实现。
- 工作树存在大量合法未提交改动；本轮不评价无关内容。

## 8. 复审条件与 Lead 下一步

下一次定点复审只需固定新 revision/SHA，并证明：

1. `test-plan.md:185` 不再把 I-F005-13～29 的现役 application 状态写为计划/未执行；
2. `test-plan.md:537` 不再把真实 provider 证书/网络/外部证据归给 S5.3；
3. 其余 F001/F002 修订、I-PRD-01～20、`planned-not-executed`、性能 TBD、代码 F002/B6C/B6D 状态和零新增 task/Case/Run 均未退化。

Lead 下一步应现场核本报告与固定对象，将 F003/F004 的两处残留交原作者最小返工，再由原非作者 Reviewer 定点复审。Gate PASS 前不得启动 Task-Planner、继续 B6C 实现或启动 B6D。

## 9. 写后验证

- 七个固定对象写后 SHA256：与第 1 节一致，无漂移；
- 本报告定向 `diff --check`：PASS；
- 唯一写入归属：仅新增本报告，未修改被审对象或其他文件；
- 本报告最终 SHA256：在完成返回中登记，避免文件自指哈希。
