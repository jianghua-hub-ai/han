# identity-auth 详细设计 r07 独立回归复审报告

- 复审日期：2026-10-20
- objectType：design
- Reviewer：非作者独立 Reviewer
- 原报告：`.spec/wip/identity-auth/review-design-r06.md`
- 原开放项：`IA-DES-R06-N001`
- 主对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 主对象修订：`identity-auth-design-ir01-r07`
- 附属对象：`.add/project/knowledge/design-workspace/identity-auth/stories.md`
- 附属对象修订：`identity-auth-stories-ir01-r07`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories、历史报告或 upstream。

## 1. 固定对象与依据

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `0e6859cdf4c423065baf7af5dd046b30fcb52eaeda83fef21aeb67a583b644ac` | `0e6859cdf4c423065baf7af5dd046b30fcb52eaeda83fef21aeb67a583b644ac` | 一致；与委托相符 |
| `stories.md` | `9c0f378f7ea0d961e1cc6f54fa59a0699ef3f4c8218552b9b0a0b03983b82463` | `9c0f378f7ea0d961e1cc6f54fa59a0699ef3f4c8218552b9b0a0b03983b82463` | 一致；与委托相符 |

按 `<!-- ADD:CHECKPOINTS -->` 前正文排除确认段，并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:df9e649399bc676c4454eeb6ca71f83d863865d8d8215e64da801eff13402154`

与 Lead artifact 一致。

### 1.2 上游与生产者快照

| 依据 | 修订/状态 | 完整 SHA-256 或 checkpoint |
|---|---|---|
| requirements | `requirements-ir01-r16` | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r10` accepted | `906d98afab853dd39904d5c0e25cfaa0f394713ed2a7fd73e0cf70035f1b989d`；`decision-a4868098c7e9483c946d3ea2e534abfb416356816981912d9be26b3e77fee1f9` |
| identity boundary | `identity-auth-boundary-ir01-r09` accepted | `42866dc11f41409c74846b54f96f5d579ae1ce9e141fc9446bbec4c7a054144a` |
| article boundary | `article-management-boundary-ir01-r10` accepted | `fb26477a2f07f406ba978476394cf83eedce2d58e8a6360c49b784d5271f189d` |
| article provider design | `article-management-design-ir01-r04`，并行复审中 | `c0a65f58e85bba89d01b7cdb1b599c5780ab4032637da2a73e195b87c4e40b8b` |
| article stories | `article-management-stories-ir01-r04` | `85cacd742b946f1b82c6235419f22508374b56da3995923d90418fd67bee1a08` |
| ADR-0020 | 已接受修订 | `content.md` SHA-256 `4a448cfc9ebcd5b57427f51a2a3e6ce67c59e074619ff86c4c6d6f10fe83915b` |

完整复用同会话已读取且版本未变的 Reviewer Role、`add-code-review`、评审清单、抽象/设计/决策规则和产品安全红线；重新完整读取 r06 报告，核对 r01～r05 固定报告哈希及历史闭环，并读取 article review r03。r06 原始设计快照未单独保存，因此本次以 r06 固定哈希、r06 报告和 r07 全文整改记录固定变更依据。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

逐项检查 `IA-DES-R06-N001` 的关闭条件：

1. ReplayStatus 是否绑定当前 `run_id + backup_ref + namespace`；
2. evidence canonical 字段、版本、编码和 digest 是否具体且双方一致；
3. `ReplayAndVerify` / `ReadStatus` 是否均以 run+backup 为参数；
4. identity 是否锁定 RecoveryRun，原子比较 DTO 与 `RecoveryRun.backup_ref`；
5. backup A ready、旧 ready、未知版本、摘要损坏、输入变化和 evidence 不可读是否 fail closed；
6. ReplayAll、DeletionOperation/Audit 对账、evidence 和双 namespace 验证是否全部早于 Recovery Journal `MarkConsumed` / readiness；
7. identity 是否仍只消费 DTO，不读取 article Journal、inventory、DeletionOperation、AuditEvent、evidence 表或 Article。

### Round 2：历史回归与真实生产者对账

沿 11 条 identity-owned US、S1～S16、RecoveryRun/RecoveryJournal、FMEA、安全、测试和 stories 共享依赖检查整改回归；再与 architecture r10、两份 accepted boundary、修订 ADR-0020 和 article design r04 的 provider 接口、DTO、evidence schema、崩溃续跑及失效算法逐字段对账。

## 3. 原 finding 复审

### [MEDIUM] IA-DES-R06-N001 — still-open

- 原 finding：Article replay ready 证据没有绑定当前备份和已验证的重放快照。
- 对象/修订：identity-auth design r07。
- 位置：
  - identity consumer：`design.md:1475`、`1481-1494`、`1642`、`1662-1663`、`1737-1740`；
  - article provider：`article-management/design.md:817-841`、`949-969`、`1622-1650`。
- 已修复部分：
  - `ReplayAndVerify` 与 `ReadStatus` 均改为 `run_id,backup_ref`；
  - RecoveryRun 保存 article evidence version/digest/ref/namespace；
  - identity 在锁定 RecoveryRun 的事务中比较 run、backup、namespace、版本和 digest；
  - `ready_to_consume` 前二次 `ReadStatus`，旧 ready、错 backup、未知版本、不可读和 input 变化均阻止 `MarkConsumed/open`；
  - 恢复顺序明确为 ReplayAll → operation/audit 对账 → evidence → 双 namespace → MarkConsumed → readiness；
  - identity 明确禁止读取 article 内部表。
- 仍未关闭的合同冲突：
  1. identity 声明 DTO 包含 `evidence_ref`、Journal head/hash、inventory scope/cutoff/digest、operation/audit digest，以及 operation/audit 核对计数；article r04 却明确 DTO“只包含”run、backup、namespace、status、version/digest、三个 tombstone 计数和 completed_at。
  2. identity 要求用 DTO canonical 字段重新计算 `evidence_digest`；article provider 不输出这些 canonical 字段。identity 若遵守 DTO-only 禁令，就没有输入可以重算；若读取 evidence/Journal/inventory/operation/audit 表补齐，又违反所有权边界。
  3. 字段名不一致：identity 使用 `tombstones_replayed`，article DTO/evidence 使用 `replayed`。
  4. identity canonical 范围包含 `deletion_operations_reconciled_count`、`audit_events_verified_count`，article canonical v1 不包含这两个字段；identity 所写 `audit_events_verified=true` 又与其 DTO 的 count 字段不一致。
  5. identity 的二次比较要求 `version/digest/ref` 完全相同，但 article DTO 不返回 `evidence_ref`。
- 触发：article r04 按其接口返回正常 ready DTO，identity r07 尝试按自身合同核验并写 `article_deletion_ready`。
- 影响：
  - 严格实现 identity 合同会因缺字段永久 fail closed，恢复无法开放；
  - 若实现者为可用性跳过 digest 重算、evidence_ref 或 input 摘要比较，则原 finding 的 stale/错 backup ready 风险重新出现，可能开放尚未完成永久删除重放的旧备份。
- 最小修复建议：由两个设计共同引用一个精确、版本化的 DTO/canonical schema，不允许各自维护近似字段表。至少统一：
  - DTO 是否携带全部 canonical 字段；
  - `evidence_ref` 是否存在及其语义；
  - `replayed` 的唯一字段名；
  - operation/audit 使用 digest、计数还是布尔结论；
  - v1 canonical 字段顺序和排除字段。
  
  若坚持 identity 重算 digest，article DTO 必须返回完整 canonical 输入摘要字段；这些是恢复证据摘要，不是内部表读取。若坚持 article 的最小 DTO，则 identity 不能声称可独立重算 digest，必须另行固定可验证机制并经 architecture/boundary 接受，不能只比较 provider 自报的摘要字符串。
- 验证：
  1. 用同一契约样本在 article 生成 DTO，并在 identity 不访问任何 article 内部表的情况下重算出相同 digest；
  2. JSON/schema 契约测试逐字段验证名称、类型、必填、canonical 顺序和版本；
  3. backup A/B、旧 ready、head/inventory/operation/audit 变化、evidence_ref 缺失和崩溃续跑全部保持既定 fail-closed 行为。

原 finding 的风险方向已被正确识别，但 consumer/provider 具体合同尚未形成同一个可实现事实，因此不能关闭。

## 4. 历史 finding 回归状态

r01～r05 已关闭的 15 个稳定 finding 均未发现回归：

- 隐私确认/处理门禁、临时凭据不回退永久密码、备份恢复凭据与账号 Gate；
- SMTP 强制 TLS、零文章 retained_email 清理、login CSRF、防暴力；
- 公共错误、分页、Story DAG；
- onboarding 换版、三类限流桶、`recovery_review_required`；
- kept_deleted canonical 删除；
- Recovery Journal 全 retained key 跨版本判重。

当前 r07 仍保持：

- 11 条 identity-owned US 全部映射；
- S1～S16 均有落点；
- identity 不拥有 Article、DeletionOperation、Article Deletion Journal、DeletionReplayEvidence 内部表或文章恢复能力；
- stories 公共验收已加入 run+backup、digest、旧 ready/input 变化和 `MarkConsumed/open` 前置；
- 41 条依赖边无环，未发现正文依赖漂移。

## 5. 新 findings

未发现独立于 `IA-DES-R06-N001` 的新增 CRITICAL、HIGH 或 MEDIUM finding。

## 6. 静态验证与限制

- design 2171 行，stories 397 行。
- PlantUML：design 22 块、stories 1 块；起止标记成对。
- 未发现字面 `\n` 或 `skinparam linetype ortho`。
- design 18 个、stories 12 个本地 Markdown 链接均存在。
- Story DAG 41 条边，未发现环。
- 11 条 identity-owned US 与 S1～S16 均可定位。
- 本机无 `plantuml` 命令；仅做静态检查，未实渲染。
- 当前无业务代码、schema、迁移、API schema 或测试；未运行不存在的构建、lint、单元、集成或 E2E。
- article design r04 正在并行复审，尚无独立 PASS；本报告的生产者对账只适用于 §1.2 固定 SHA-256。该对象若变化，N001 必须按新 provider 修订重新核对。
- 初始管理员轮换、容量/期限、隐私部署输入、tombstone/evidence 保留与版本退役仍为既有 deferred/TBD；本轮不将其伪判为已完成。

## 7. 结束漂移复核

结束时再次计算：

- identity design：`0e6859cdf4c423065baf7af5dd046b30fcb52eaeda83fef21aeb67a583b644ac`
- identity stories：`9c0f378f7ea0d961e1cc6f54fa59a0699ef3f4c8218552b9b0a0b03983b82463`
- article provider design r04：`c0a65f58e85bba89d01b7cdb1b599c5780ab4032637da2a73e195b87c4e40b8b`
- article provider stories r04：`85cacd742b946f1b82c6235419f22508374b56da3995923d90418fd67bee1a08`

identity 固定对象与开始值一致；并行 provider 快照也未在复审期间漂移。

## 8. 最终结论与复审条件

**最终结论：NOT PASS。**

- `IA-DES-R06-N001`：**still-open**。
- 历史已关闭 findings：未发现回归。
- 新增独立阻断 finding：无。

r07 已正确补上 run+backup 参数、恢复顺序、RecoveryRun 比较、旧 ready/input 变化 fail closed 和 DTO-only 所有权；但 identity consumer 与 article provider 的实际 DTO/canonical digest schema 不一致，当前仍无法据此实现并验证安全开放。

复审条件：

1. 固定 identity 与 article 两侧同一版 DTO/canonical schema，并给出双方精确 revision/SHA；
2. 消除 `evidence_ref`、canonical 字段集、`replayed` 命名、operation/audit digest/计数/布尔语义差异；
3. 提供无需 identity 读取 article 内部表即可重算相同 digest 的契约样本或经批准的替代验证机制；
4. 保留当前 run+backup、旧 ready/input 变化 fail closed、MarkConsumed 前顺序和 DTO-only 边界；
5. 由独立 Reviewer 对原 `IA-DES-R06-N001` 和 article 并行复审结果做联合回归。

本结论只针对固定 r07 对象和 §1.2 的 provider/upstream 快照，不代替用户/Lead 的风险接受、设计确认或实施授权。
