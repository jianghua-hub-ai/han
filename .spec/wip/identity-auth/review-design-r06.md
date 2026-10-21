# identity-auth 详细设计 r06 独立回归复审报告

- 复审日期：2026-10-20
- objectType：design
- Reviewer：新的非作者 Reviewer；原 Reviewer 会话不可恢复
- 历史报告：`.spec/wip/identity-auth/review-design-r01.md`～`review-design-r05.md`
- 主对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 主对象修订：`identity-auth-design-ir01-r06`
- 附属对象：`.add/project/knowledge/design-workspace/identity-auth/stories.md`
- 附属对象修订：`identity-auth-stories-ir01-r06`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories、历史报告或上游。

## 1. 对象固定与依据

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `f9153dcc4a2d0c26390e7989923ebd993ad00edecda8b9eb72da4e7cf42fbddf` | `f9153dcc4a2d0c26390e7989923ebd993ad00edecda8b9eb72da4e7cf42fbddf` | 一致；与委托相符 |
| `stories.md` | `c94730148d80b46fcde731f9ef6ce4d8520fe47d79aa2b088babcec134ef804c` | `c94730148d80b46fcde731f9ef6ce4d8520fe47d79aa2b088babcec134ef804c` | 一致；与委托相符 |

按 `<!-- ADD:CHECKPOINTS -->` 前正文排除确认段，并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:8365393761c09dd842e44f923da259360460e7c697e8cc02b06e8bd88918a824`

与 Lead artifact 一致。

### 1.2 基线与上游快照

| 依据 | 修订/状态 | 完整 SHA-256 或 checkpoint |
|---|---|---|
| identity r05 基线 | 独立复审 PASS；设计基线接受；实施 Gate deferred | `design 73a6d7ed...b1525c`、`stories 2551f6c9...25d8b`；checkpoint `decision-76f8d41e...09563` |
| requirements | `requirements-ir01-r16` | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r09` accepted checkpoint | `f69dd966f36f4d268a775657ddfc46e1b076c64098fd998e254fed3cfa886072`；`decision-203d6b9bf6474bb8b07a04887f65549f306b4072e62e7b5657cef50174f94b39` |
| identity boundary | `identity-auth-boundary-ir01-r08` accepted | `c9ebf1660b06252d33db588f65133ef57e05ffaa5ede95b996c314e55ca12a54` |
| article boundary | `article-management-boundary-ir01-r09` accepted | `473cf725552ee52bd5a420ee2667c9e1ffd04b04fca1a21a2f54c3f194385614` |
| article design | `article-management-design-ir01-r03`；review r03 PASS | `69535a987a32ca5cb9a696ff74f1cfe808ca44351fc644968b98fbbd1dbe3155` |
| ADR | ADR-0001～0020 | index 与 `framework.md`、`identity.md`、`content.md` 全文读取 |

r05 原始文件快照未保存在 Git 历史或独立副本中，因此无法生成逐字节 r05→r06 diff；本次以 r05 固定哈希、checkpoint、五份历史报告和 r06 全文修订记录固定基线，再对 r06 全文及 S16 新增落点做独立审查。

## 2. 两轮实际检查范围

### Round 1：固定对象直接审查

完整检查 `design.md` 2152 行和 `stories.md` 395 行，重点核对：

- 11 条 identity-owned US 与 Story 四块、共享依赖和 41 条 DAG 边；
- S1～S16 映射、接口签名、RecoveryRun/RecoveryJournal 状态；
- identity 对 Article、DeletionOperation、Article Deletion Journal 的禁止所有权；
- `ArticleDeletionGate` / `ArticleDeletionReplayStatus` 消费合同；
- 恢复顺序：identity 终结与 canonical 删除 → article tombstone ReplayAll → DeletionOperation/Audit 对账 → 两个 namespace 分别验链 → Recovery Journal `MarkConsumed` → readiness；
- FMEA、安全红线、STRIDE、可测试性、可服务性和失败时 readiness=false。

### Round 2：历史回归与真实生产者检查

逐份完整读取 identity r01～r05 历史报告和 article review r03；沿 requirements r16、architecture r09、两份 accepted boundary、article design r03、ADR-0001～0020 检查：

- r01～r04 全部原 finding 的关闭条件是否在 r06 仍成立；
- S16 消费者与 article provider 的恢复、Journal、finalizer、inventory、审计合同是否相容；
- 数据和状态是否仍只有一个权威写入者；
- 恢复重启、状态重读、namespace 混淆、缺 key、损坏、operation/audit 不一致和旧备份复活场景是否 fail closed。

## 3. 历史 finding 回归状态

| 原 finding | r06 状态 | 回归证据摘要 |
|---|---|---|
| `IA-DES-R01-F001` 隐私告知/授权 | closed，无回归 | PrivacyNotice/Acknowledgement、管理员声明、ProcessingGate、撤回复用删除仍完整 |
| `IA-DES-R01-F002` 临时凭据回退永久密码 | closed，无回归 | `credential_mode` 唯一路径；expired/superseded/recovery_invalidated 不回退 |
| `IA-DES-R01-F003` 旧备份复活凭据/权限 | closed，无回归 | 全部永久密码、Session、活动临时凭据失效；账号 recovery Gate 和默认不授权仍成立 |
| `IA-DES-R01-F004` SMTP TLS | closed，无回归 | 仅隐式 TLS/强制 STARTTLS，证书链与主机名校验，禁止降级 |
| `IA-DES-R01-F005` 零文章 retained_email 清理 | closed，无回归 | canonical 删除仍覆盖零/多/违规文章和清理事实 |
| `IA-DES-R01-F006` login CSRF | closed，无回归 | same-origin、Origin/Fetch Metadata、拒绝 credentialed CORS、旧会话撤销仍成立 |
| `IA-DES-R01-F007` 防暴力合同 | closed，无回归 | 三类持久桶、代理信任、原子 upsert、退避与低基数指标仍成立 |
| `IA-DES-R01-F008` 公共错误注册 | closed，无回归 | 唯一公共错误表与 internal event 分离保持 |
| `IA-DES-R01-F009` 分页契约 | closed，无回归 | envelope、cursor、排序、过滤绑定和 null 语义保持 |
| `IA-DES-R01-F010` Story 依赖 | closed，无回归 | 41 条边无环，正文依赖未发现漂移 |
| `IA-DES-R02-N001` 双 Gate 换版 | closed，无回归 | onboarding version 与 current 原子比较保持 |
| `IA-DES-R02-N002` 三类限流桶 | closed，无回归 | `bucket_kind + bucket_key_digest` NOT NULL 唯一键保持 |
| `IA-DES-R02-N003` 恢复幂等状态 | closed，无回归 | `recovery_review_required`、默认 uncertain 和受控转换保持 |
| `IA-DES-R03-N001` kept_deleted 不完整 | closed，无回归 | 日常/恢复共用 canonical 删除；任一子步骤失败不完成 review/readiness |
| `IA-DES-R03-N002` recovery secret 跨 key 复用 | closed，无回归 | 一个 active + verify-only retained keyring，全 retained key 判重与退役前置保持 |

历史稳定 ID 是 `IA-DES-R03-N002`；r06 顶部整改表和附录使用了 `IA-DES-R04-N002/R04-N002` 别名。该问题只影响追溯可读性，本轮仍按原稳定 ID 完成回归，不作为阻断 finding。

## 4. 新 finding

### [MEDIUM] IA-DES-R06-N001：Article replay ready 证据没有绑定当前备份和已验证的重放快照

- finding ID：`IA-DES-R06-N001`
- 对象/修订：identity-auth design r06
- 位置：`design.md:1472-1478`、`design.md:1653-1654`、`design.md:1727-1729`、`design.md:2043-2045`
- 违反依据：
  - `design.md:1728` 自身要求验证 run ID 与备份引用一致；
  - ADR-0020 要求恢复当前旧备份后重放全部有效 tombstone，inventory/key 不可证明时不得开放；
  - article design r03 `§3.4.6～3.4.7` 要求 `ReplayAll(backup)`、`VerifyInventory`、DeletionOperation/Audit 对账和两个 namespace 分别通过；
  - 评审清单要求跨模块接口字段、状态和失败收敛可实施，状态权威唯一。
- 现状：identity 唯一允许消费的 `ArticleDeletionReplayStatus` 只含 `run_id,namespace,status`、计数、`evidence_ref/completed_at/status_digest`，没有 `backup_ref`，也没有定义 `status_digest/evidence_ref` 必须覆盖的 canonical 内容、已验证 Journal head/sequence 或 inventory 结论。`ArticleDeletionGate.ReadStatus` 又只接受 `run_id`。因此文档虽要求 identity 在 `MarkConsumed` 前确认“run ID/备份引用一致”和全部 tombstone/inventory 已覆盖，但允许的接口没有可核对该事实的明确字段或不可变证据合同。
- 触发：恢复流程在 article replay 后、Recovery Journal `MarkConsumed` 前崩溃并续跑；或同一未消费 recovery run 更换/误绑定 `backup_ref`，随后通过 `ReadStatus(run_id)` 取得先前的 ready 状态。实现若只按当前字段检查 run、namespace 和 `status=ready`，无法证明 ready 对应当前 RecoveryRun 的备份及同一份完整 tombstone/inventory 快照。
- 影响：stale 或误绑定的 ready 证据可能使 Recovery Journal 被消费并开放 readiness，而当前恢复出的旧备份尚未完成全部删除重放；已永久删除文章可能重新出现在成员、作者或管理员入口。
- 最小修复建议：保持 Article Deletion Journal、DeletionOperation 和 Article 全部归 article-management，不让 identity 读取其内部表；只收紧最小证据 DTO/接口：
  1. `ArticleDeletionReplayStatus` 明确携带并绑定 `backup_ref`；
  2. 定义不可变 replay evidence 的版本和 digest 覆盖范围，至少覆盖 `run_id,backup_ref,namespace`、已验证 Journal head/sequence、inventory 结论及 operation/audit 对账结论；
  3. `ReadStatus` 按 `run_id + backup_ref` 读取，或要求返回值的 `backup_ref` 与 RecoveryRun 原子比较；
  4. 明确状态证据的 article-owned 持久化/重算和幂等规则；任何缺失、旧版本、digest 不匹配或 evidence 不可读取都返回 failed/unknown 并保持 readiness=false。
- 验证：
  - backup A 产生的 ready 证据不能用于 backup B；
  - replay 后崩溃重启，重读证据仍绑定同一 run/backup/Journal head/inventory；
  - tombstone head、inventory、key、DeletionOperation 或 Audit 任一变化/缺失时旧 ready 失效；
  - 只有精确匹配的 ready 证据才能进入 `ready_to_consume` 和 `MarkConsumed`。

## 5. S16 与所有权专项结论

除 N001 外，重点合同符合预期：

1. **所有权正确**：identity 只通过 `ArticleDeletionGate` 消费 `ArticleDeletionReplayStatus`；`design.md:1480-1485` 明确 Article Deletion Journal、DeletionOperation、Article 删除状态归 article-management，identity 无直接读写权。
2. **恢复顺序正确**：`design.md:1726-1729` 和时序图明确 article ReplayAndVerify、DeletionOperation/Audit 对账和双 namespace 验证均先于 Recovery Journal `MarkConsumed` 与开放。
3. **失败收敛正确**：`design.md:1704`、`1729-1731`、`1971`、`1997` 明确任一 namespace、key、Journal、finalizer、operation、audit 或重放失败时 readiness=false。
4. **不新增文章恢复**：未发现 identity 读取 tombstone 内容、重建 Article 或定义文章恢复入口。
5. **Story 共享依赖**：stories 公共验收已加入 identity_ready 后等待 ArticleDeletionGate、双 namespace 验证和只消费 ReplayStatus；41 条既有依赖边无环，未破坏 11 个业务 Story。

## 6. 非目标 finding、待纠偏与限制

### 上游风险（不计为本对象 finding）

`architecture.md:875` 的安全部署运行手册仍写成“Journal/DB 对账 → MarkConsumed → 开放”，没有插入 r09 已新增的 article tombstone ReplayAll、DeletionOperation/Audit 对账和双 namespace 验证。architecture 同文 §9、ADR-0020、两份 boundary、article design 和本轮明确验收依据采用更严格顺序，r06 也采用了正确顺序；但该旧句可能误导只消费架构运行手册的实施者，应由架构 owner 单独修订，Reviewer 未修改 upstream。

### 静态验证

- design PlantUML 22 块、stories 1 块；起止标记成对。
- 未发现字面 `\n` 或 `skinparam linetype ortho`。
- design 17 个、stories 12 个本地 Markdown 链接均存在。
- Story DAG 41 条边，未发现环。
- 11 条 identity-owned US 与 S1～S16 均可定位。
- 本机无 `plantuml` 命令；按委托仅做静态检查，未实渲染，不能证明用户侧无空框或布局问题。
- 当前无业务代码、schema、迁移、API schema 或测试；未运行不存在的构建、lint、单元、集成或 E2E。
- 项目仍未登记检查命令；Q1、容量/期限、隐私部署输入和灾备运行参数保持既有 deferred/TBD，不在本轮伪判通过。

## 7. 结束漂移复核

结束时已再次计算完整 SHA-256：

- `design.md`：`f9153dcc4a2d0c26390e7989923ebd993ad00edecda8b9eb72da4e7cf42fbddf`
- `stories.md`：`c94730148d80b46fcde731f9ef6ce4d8520fe47d79aa2b088babcec134ef804c`

两者均与开始值一致；审查期间固定对象无漂移。

## 8. 最终结论与复审条件

**最终结论：NOT PASS。**

- r01～r05 的全部历史 findings：closed，未发现行为回归。
- 新 finding：1 个 MEDIUM（`IA-DES-R06-N001`）。
- 未发现新增 CRITICAL 或 HIGH。
- r06 的所有权、恢复主顺序、fail-closed 和 stories 公共依赖方向正确，但 Article replay ready 证据合同尚不足以证明其绑定当前备份与完整重放快照。

复审条件：

1. 提供新的 design/stories 精确 revision、完整 SHA-256 和规范 artifact digest；
2. 修订 `ArticleDeletionReplayStatus`/`ArticleDeletionGate` 的 run+backup+不可变 evidence 绑定合同，并同步 article provider 设计或取得其 owner 的明确兼容确认；
3. 增加恢复中断续跑、错 backup、旧 ready、Journal head/inventory 变化和 evidence digest 不匹配的 fail-closed 验证设计；
4. 继续保持 identity 不拥有 Article Deletion Journal、DeletionOperation、Article 或文章恢复能力；
5. 上游 architecture §8.3 的旧顺序由架构 owner 另行纠偏，不由本 Reviewer 修改。

本结论只针对固定 r06 对象，不代替用户/Lead 的风险接受、设计确认或实施授权；既有实施 Gate deferred 状态不因本报告自动改变。
