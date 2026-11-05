# B6D / S11.1 六文件 R 对账独立 Planning Review R02

- **Reviewer**：Gibbs（非作者独立 Planning Reviewer）
- **日期**：2026-10-25
- **工作区**：`D:\AID\VSCode\Codex\Han`
- **对象类型**：Tasks / Plan 修订及其 Code Review 输入
- **动作边界**：只读评审；除本报告外未修改代码或规划，未执行真实 PostgreSQL、SMTP、部署或 E2E，未读取 DSN，未 commit/push。

## 1. 固定对象

| 对象 | SHA256 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `F0C07FC4CEDA2532470694FDE1EF7F077CAC9A6F6383896CA8F209C5500967C0` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `6BA503907E0BDB03228BE4F0D0D95C069DCCC248B92CD97F9A4F5AB89819B7F7` |
| `.spec/wip/identity-auth/review-code-s11.1-r01.md` | `BE60E271B0A294199E4D3F8DC378F3FD3E218F6BB8EE03A89636C38A89C12103` |

开始及落报告前均重新计算固定对象哈希且一致。本结论只适用于以上修订。

## 2. 两轮实际检查范围

### Round 1：Tasks/Plan 直接检查

- `tasks.md:1022-1048`：六文件所有权、F001～F003 责任、验收命令、requires、暂停及停止条件。
- `ir-01-content-community-batches.md:663-684`：B6D 六文件 Context Pack、finding 证据、关闭条件。
- `ir-01-content-community-batches.md:706-754,782-787,835-844`：DAG、共享文件顺序所有权、批次 Gate、三态与暂停状态。
- 机械核对三态：**57 total / 26 completed / 3 active / 28 todo**；active 精确为 S3.3、S3.5、S11.1。

### Round 2：原 findings 与真实接缝有界检查

- 对照 Code Review R01 的三个 MEDIUM finding：
  - F001：敏感准备早于 writer Gate/current predecessor/stale/hold 权威校验；
  - F002：首次响应与 same-key succeeded replay 的 expiry 事实不一致；
  - F003：production 测试绕过真实业务授权且缺 body purpose 串改。
- 有界检查当前真实消费者接缝：
  - `internal/identity/password_reset.go:172-288` 的 candidate 准备、same-key 分类和 replay expiry；
  - `internal/identity/temporary_credential_resend.go:53-240,551-766` 的 purpose router、两 coordinator 复用、两类 expiry 与快照；
  - `internal/identity/password_reset_store.go:53-91,241-272,621-708` 的既有唯一 Gate、candidate authority 与 lineage classifier；
  - identity design r44 `design.md:2381-2412` 与 test-plan r20 `I-CRED-03`、`I-PRD-01/02/18/19`。

当前代码仅作为判断规划充分性的消费者证据，不是本次固定修复 revision；本报告不关闭代码 findings。

## 3. Findings

| 严重度 | ID | 结论 |
|---|---|---|
| — | — | 未发现需要修改本次 Tasks/Plan 的 planning finding。 |

Code Review R01 的 `IDENTITY-S11.1-REV-R01-F001`～`F003` **仍为 OPEN**；它们是本次返工输入，不因 Planning Gate 通过而关闭。

## 4. 评审结论

### 4.1 新增 `password_reset.go` 是关闭 F001/F002 的最小且充分所有权

- F001 的真实顺序根因位于 `password_reset.go`：same-key 未命中后，当前实现先生成 secret/ID/hash，随后才由 resend authority/Store 校验 Gate 和 lineage。仅修改原五文件无法可靠改变这一共享准备顺序。
- F002 的普通 `password_reset` 首次结果和 replay 重建均位于 `password_reset.go`；要让首次响应使用与持久 claim/credential 一致的唯一 expiry 事实，该文件不可省略。
- 新管理员 purpose 的对应 expiry 修复仍落在既有 `temporary_credential_resend.go`；F001/F002 回归可由同 package 的 `temporary_credential_resend_test.go` 承载，不需要再占有 `password_reset_test.go`。
- 现有唯一 Store 已提供 Gate、candidate authority 和 lineage classifier。根据当前接缝，可以在六文件内通过 resend 专用前置流程复用这些权威事实并在最终写入时继续由同一 Store 复核；没有证据表明必须修改 Store、SQL、schema 或 migration。

因此六文件相对五文件只增加根因文件，未加入无关文件，也未遗漏关闭 F001/F002 所需的实现或测试承载点。

### 4.2 边界限制足以防止重开 S10.1 或越权修改持久权威

- Tasks/Plan 明确把 `password_reset.go` 授权限制在 **resend 前置权威校验与可持久重放 expiry 事实**。
- S10.1 保持历史 CLOSED；本次是后继 S11.1 对共享文件的顺序、有界消费，不改变 S10.1 task 状态或责任。
- 首次 password-reset 公共合同、worker/operator/Store、schema、migration、composition 均明确禁止改变。
- 精确六文件写集排除了 `password_reset_store.go`、migration 和公共 contract/registry 文件；需要第七文件、公共合同/architecture/ADR 变化或第二 authority/coordinator/Store/Gate/root/global 时必须停止并交回。
- F001 的修复不得复制或替代 Store/lineage 权威；最终实现仍须由代码复审验证其只是 resend 专用前置与同一权威复用，而不是建立并行 classifier。

### 4.3 F003 责任完整

Tasks/Plan 已要求：

- 保留 router/wiring seam 测试，但不得把 seam 当作业务授权证据；
- 经过真实 `TemporaryCredentialResendService` 与既有 coordinator；
- 覆盖 member、普通 admin、initial admin及当前有效管理员的 purpose/角色组合；
- 验证 `new_administrator` 仅 initial admin 可执行，普通用户 password-reset resend 仅有权管理员可执行；
- 分别覆盖 body/header/query 无法串改 route-derived purpose；
- 对拒绝路径断言稳定 403、公共错误 envelope及零 secret/ID/hash/mail/持久业务写入；
- 保留 exact route、middleware、Idempotency、404/405 和共享依赖回归。

这些责任与 design r44 及 I-CRED-03、I-PRD-01/18/19 对齐，足以定点复审 F003。

### 4.4 requires、DAG、三态和暂停状态一致

- S11.1 原 requires 未变化且均已满足；Plan DAG 未新增边或新 task。
- S11.1 保持 `[-]`，表示已启动/已有 owner；Code Review 为 `CHANGES REQUIRED` 后又明确 `implementation paused pending Planning Review`，没有误标 completed 或 todo。
- Planning Review PASS 后仍须 Lead 现场复核并明确恢复 Curie START；Copernicus 随后定点复审 F001～F003。
- S10.1/S10.4 保持 CLOSED，S10.5 保持未启动且不是 S11.1 硬依赖。

## 5. 非阻断说明与未验证边界

- 本 PASS 只确认六文件返工合同最小、充分且边界一致，不批准具体实现方案。
- 未验证返工代码是否真正做到敏感准备前拒绝、expiry 逐字节稳定或真实授权矩阵通过。
- 未运行 Go 专项、race、Manifest 七项、build、gofmt 或 diff-check。
- 未验证真实 PostgreSQL 18、多连接/commit-ack-loss、SMTP/TLS、Browser、Host、部署、E2E、容量或 production readiness。
- 若实现触及 initial reset 行为、复制 Store SQL/classifier、修改 Store/schema/migration/public contract，或需要第七文件，本 Gate 立即失效。

## 6. 复审条件

当前没有 planning finding 需要复审。以下任一发生时须重新固定对象：

- Tasks、Plan 或 Code Review R01 输入哈希变化；
- 六文件写集、requires/DAG、暂停状态或停止条件变化；
- 修复方案需要 Store/schema/migration/public contract 或其他第七文件；
- 上游 design/test-plan/ADR 权限、expiry 或 lineage 合同变化。

代码返工完成后仍必须由 Copernicus 针对 F001～F003 复审固定代码 revision。

## 7. Gate

**Planning Review Gate: PASS**

