# identity-auth test-plan r16 最终定点复审 R03

- 日期：2026-10-24
- Reviewer：原非作者独立 Reviewer
- objectType：design（test-plan）
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审目标：仅复审 R02 保留的 F003/F004，并回归 R02 已关闭的 F001/F002、I-PRD-01～20 与状态边界
- 结论：**F003、F004 CLOSED；无新增 finding；Test-plan Review Gate: PASS**

本报告只评价固定 test-plan r16 是否满足 R02 的复审条件，且是否可交由 Lead 现场复核后决定后续动作。本报告不评价代码已修复，不执行测试，不修改任务状态，也不授权 Task-Planner 或实现。代码 finding `IDENTITY-S10.1-REV-R01-F002` 继续保持 Major OPEN；B6C/S10.1 继续 `[-] blocked-for-design-revision` / implementation stopped；B6D/S11.1 继续 todo、未认领、未启动。

## 1. 固定对象与写前 SHA256

| 对象 | Revision / 用途 | SHA256 | 写前结果 |
|---|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `identity-auth-test-plan-ir01-r16` | `FE8AB4B420D16C6E74F1BEB4AE7E4D41B03B5D0637CB95EC82B4818A550A6B8D` | MATCH |
| `.spec/wip/identity-auth/review-test-plan-r15-s10.1-r02.md` | 原 R02 | `F1EC9F96B677EC7EB100B85D95DAA2D3CE05BF2CA624168C5BC3158B492B586F` | MATCH |
| `.spec/wip/identity-auth/review-test-plan-r14-s10.1-r01.md` | 原 R01 | `C77AF5A60C2C35939ED5134E48F823AFCFDF7082F1D3D414D847F4569625CE39` | MATCH |
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r27` | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r39` | `71EEC596875EE9CBF44CF300C72A4203F5F2C8976522F478A1857D882AE86230` | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r15` | `715F5BEB95FC7AF141F7F9ED18D900979921C520D7E11A857AC017F65C481803` | MATCH |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `identity-auth-boundary-ir01-r13` | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` | MATCH |
| `.spec/wip/identity-auth/review-design-r39-s10.1-r03.md` | Design R03 | `CAE91411FD4D126BF033896186D761756C41AC1E7C8B6810FD36F34F8601B0B2` | MATCH |

固定对象无写前漂移；test-plan metadata 正确引用 r16、R02 与固定上游，状态保持 `planned-not-executed`。

## 2. 两轮实际检查范围

### Round 1：固定对象直接检查

1. 完整复核 R02 的 F003/F004 定位、影响、修复建议与复审条件。
2. 定点检查 r16 的 metadata、修订历史、§4.2.1、覆盖映射、数据/环境、任务边界、执行顺序、执行前 Gate 与阶段结论。
3. 检查 I-F005-13～29 是否统一为已 CLOSED 的 application historical/回归基线，并与真实容量、平台、部署、E2E 未验证状态分离。
4. 检查 S5.2 deterministic Mailer、S5.3 本地真实协议、S3.4 获批 provider/外部收件是否形成全文唯一且互不替代的证据所有权。
5. 回归检查 I-CRED-06、I-PRD-10/11、I-PRD-01～20 主表唯一性、性能 TBD、零新增 task/Case/Run，以及代码 F002、B6C、B6D 的状态边界。

### Round 2：有界上游与真实账本深审

1. 复核 architecture r27、design r39、stories r15、boundary r13 与 Design R03 的固定摘要未漂移；仅核 r16 未破坏 accepted ADR-0011/ADR-0023 的 purpose 分流与 durable 方向。
2. 有界读取现役 tasks/Plan，核实 `identity-auth-S3.2`/F005 与 `identity-auth-S5.2` 已关闭，`identity-auth-S5.3`、`identity-auth-S3.4` 未执行，S10.1/F002 与 B6C/B6D 状态和 r16 一致。
3. 未扩展为代码实现评审；未运行代码、数据库、SMTP、部署或 E2E。

## 3. R02 保留 findings 定点复审

### F003 — CLOSED

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F003`
- R02 问题：§4.2.1 曾把已由 S3.2/F005 最终非作者 Review + Lead Gate 关闭的 I-F005-13～29 application 场景继续写成当前 planned/unexecuted，和文档后部现役 closure 冲突。
- r16 证据：
  - `test-plan.md:187` 明确 I-F005-13～29 已随 S3.2/F005 最终非作者 Review + Lead Gate CLOSED；r16 未复跑，场景只作 historical/回归基线，不重开任务或占用已释放写集。
  - `test-plan.md:496-501` 将 I-F005-13～29 分组标为现役 closure/application CLOSED，并把真实 scrape、容量和部署证据独立保留。
  - `test-plan.md:608-610,668-675,707-712` 在账本、执行顺序和 Gate 中一致区分 application closure 与 S2.2/S3.5/S3.6、真实平台、容量、部署/E2E 未验证。
  - tasks/Plan 真实账本确认 S3.2/F005 task-level CLOSED，`IDENTITY-S3.2-REV-F005` CLOSED；相关真实环境 Gate 仍独立未完成。
- 结论：全文不再把 I-F005-13～29 application 当前状态写成 planned/unexecuted，且保留“r16 未复跑”和真实环境未验证边界。**F003 CLOSED。**

### F004 — CLOSED

- finding ID：`IDENTITY-TESTPLAN-R14-S10.1-R01-F004`
- R02 问题：数据/环境章节曾残留把 provider 证书与网络行为归给 S5.3 的文字，和 S3.4 的 approved-provider/external-receipt 所有权冲突。
- r16 证据：
  - `test-plan.md:539` 建立三层唯一所有权：S5.2 仅负责 deterministic Mailer 的 deadline/cancel、调用次数和资源收敛；S5.3 仅负责本地 production-equivalent SMTP/TLS/AUTH、测试证书和可控 provider-result，严格不发外部邮件；S3.4 独占真实 provider、真实网络、受理与外部收件。
  - `test-plan.md:566,582-585,597,602` 在未验证矩阵、自动/人工边界和任务表中保持同一分层，S3.4 执行前明确要求已部署版本、审批、责任人、Case/Run、数据与清理方案。
  - `test-plan.md:631,642-643,682-693,714,720,737` 在现役边界、R 对账建议、执行顺序、Gate 与阶段结论中重复约束三层证据不得互相替代。
  - tasks/Plan 真实账本确认 S5.2 已 CLOSED；S5.3 与 S3.4 均为既有未执行任务，r16 未新增任务或改状态。
- 结论：原冲突句已移除，全文只保留一套邮件证据所有权。**F004 CLOSED。**

## 4. 回归结果

### R02 已关闭 findings

- **F001 继续 CLOSED**：`test-plan.md:306,508` 的 I-CRED-06 仍仅适用于 `new_administrator` / ADR-0011 residual-window；普通 `password_reset` 的 accepted/not-accepted/unknown/commit-ack-loss 继续由 ADR-0023 durable settlement 场景承接。
- **F002 继续 CLOSED**：`test-plan.md:389-390` 的 I-PRD-10/11 仍使用可控 Clock 覆盖保护窗口前/边界/后，禁止 sleep，记录 lease/claim/takeover，并在 failed/invalidated 后同链证明 operation/job terminal、无 hold、Gate clear、old login 与既有 Session resolve。

### 其他回归

1. I-PRD-01～20 主场景表各出现且仅出现一次，无缺号或重复主定义。
2. r16 status 继续为 `planned-not-executed`，没有把 Design R03 PASS 外推成实现、测试、部署或 E2E 通过。
3. 性能阈值继续为 TBD；真实容量、平台、PostgreSQL、SMTP/provider、Host、部署和 E2E 分别保持未验证。
4. `test-plan.md:717-720,735-737` 保持代码 F002 Major OPEN、B6C blocked/implementation stopped、B6D 未启动。
5. r16 明确没有新增或改状态的 task，也没有创建或执行 Case/Run；tasks/Plan 定向核对未发现与该声明冲突。
6. 未发现本轮修订引入的新架构、安全、迁移、恢复、状态所有权、并发或消费者契约风险。

## 5. 新增 findings

无。CRITICAL/HIGH/MEDIUM/LOW 新增 finding 均为 0。

## 6. 待决定与非阻断建议

### 待决定

无新的产品或架构决定。accepted ADR-0011/ADR-0023 未被重开或破坏。

### 非阻断建议

Lead 可在现场复核固定哈希、报告与工作树边界后，决定是否将 r16 作为后续 Task-Planner R 对账输入；Reviewer 不启动 Task-Planner，也不授权继续实现。

## 7. Test-plan Review Gate

- R02 F003：CLOSED
- R02 F004：CLOSED
- R02 F001/F002 回归：保持 CLOSED
- 新增 finding：0

**Test-plan Review Gate: PASS**

理由：r16 已消除 F005 application 当前状态冲突，并把三层邮件证据所有权统一到全文；其余 durable 场景、状态边界和未验证层次未退化。该 PASS 仅表示固定 test-plan r16 可交 Lead 现场复核并作为后续规划输入，不表示代码 F002 已关闭、B6C 已解除、B6D 可启动、任何 task/Case/Run 已授权或生产就绪。

## 8. 未验证风险与限制

- 未运行任何代码、构建、测试或静态检查；文档复审不以代码 build 为前置。
- 未连接真实 PostgreSQL，未验证 durable schema、事务、锁、commit actual state、worker lease、migration、混跑或不可安全回退。
- 未连接本地 SMTP/TLS 或外部 provider，未验证 AUTH、证书、受理、保护窗口或外部收件。
- 未运行真实 Host、部署、浏览器、性能、容量、观测平台或 E2E。
- 未创建或执行 Case/Run，未验证审批、责任人、真实数据与清理责任。
- PlantUML、部署制品和运行手册未在本轮复验。

## 9. 写后验证

- 固定对象写后 SHA256：应与 §1 完全一致；最终命令核对后无漂移。
- 定向 whitespace/diff check：对本新增报告执行，要求无 whitespace error。
- 唯一写入边界：仅允许新增本报告；写前工作树已有大量合法改动，均不得覆盖、回滚或格式化。
- 报告最终 SHA256 不嵌入正文以避免自引用改变摘要；以本轮完成返回中的最终计算值为准。

## 10. Lead 下一步

Lead 应在真实工作树复核固定对象写后哈希、报告 SHA、定向 diff-check 与唯一写入边界。复核通过后，可自行决定是否进入 Task-Planner R 对账；本 Review PASS 不直接启动 Task-Planner，不解除 B6C，不关闭代码 F002，也不启动 B6D。
