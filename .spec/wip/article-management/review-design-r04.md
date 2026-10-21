# article-management design r04 独立回归复审报告

- 复审日期：2026-10-20
- objectType：design
- Reviewer：独立非作者 Reviewer
- 主对象：`article-management-design-ir01-r04`
- 附属对象：`article-management-stories-ir01-r04`
- 基线：article design r03 独立复审 PASS
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories、历史报告、identity design 或任何上游。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `c0a65f58e85bba89d01b7cdb1b599c5780ab4032637da2a73e195b87c4e40b8b` | `c0a65f58e85bba89d01b7cdb1b599c5780ab4032637da2a73e195b87c4e40b8b` | 一致；与委托相符 |
| `stories.md` | `85cacd742b946f1b82c6235419f22508374b56da3995923d90418fd67bee1a08` | `85cacd742b946f1b82c6235419f22508374b56da3995923d90418fd67bee1a08` | 一致；与委托相符 |

按 `## 确认记录` 前正文排除确认段，并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:0282a419b9e4ac2d9656df387ee87c695d67c8f1cd7d38a5f62b16d373455c07`

与 Lead artifact 一致。工作区未发现该摘要对应的独立文件，因此它只作为规范化摘要验证值，不作为额外评审对象。

### 1.2 上游与消费者快照

| 依据 | 修订/状态 | 当前读取 SHA-256 |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r10`；checkpoint `decision-a4868098c7e9483c946d3ea2e534abfb416356816981912d9be26b3e77fee1f9` accepted | `906d98afab853dd39904d5c0e25cfaa0f394713ed2a7fd73e0cf70035f1b989d` |
| article boundary | `article-management-boundary-ir01-r10` accepted | `fb26477a2f07f406ba978476394cf83eedce2d58e8a6360c49b784d5271f189d` |
| identity boundary | `identity-auth-boundary-ir01-r09` accepted | `42866dc11f41409c74846b54f96f5d579ae1ce9e141fc9446bbec4c7a054144a` |
| ADR-0020 | accepted，已修订 replay evidence 合同 | `content.md`：`4a448cfc9ebcd5b57427f51a2a3e6ce67c59e074619ff86c4c6d6f10fe83915b` |
| identity review | r06，`IA-DES-R06-N001` | `c045c7de45a4a27e832dd1367a46cba3f8b06aae57be9cec29ab3daac2a8ee50` |
| 当前真实消费者 | `identity-auth-design-ir01-r07`，待并行独立复审 | `0e6859cdf4c423065baf7af5dd046b30fcb52eaeda83fef21aeb67a583b644ac` |

identity r07 尚未被独立复审通过，不能反向冒充已接受上游；但它是当前实际消费者，因此本报告对其固定读取快照做有界兼容检查。

## 2. 两轮实际检查范围

### Round 1：r04 直接缺陷检查

定点检查 `design.md` 的 r04 修订记录、核心对象/框架/时序、S16 接口与数据结构、`ReplayAndVerify/ReadStatus`、恢复顺序、FMEA、STRIDE、可测试性和 stories S16 验收，重点范围：

- `design.md:40-106`
- `design.md:296-418`
- `design.md:473-609`
- `design.md:638-969`
- `design.md:1496-1652`
- `design.md:1681-1885`
- `stories.md:16-23`、`92-117`

检查 run+backup+namespace 绑定、canonical digest 字段、不可变持久化、幂等重算、输入漂移、unknown/failed、错 backup 和崩溃续跑。

### Round 2：消费者、上游与历史回归

沿 requirements r16、architecture r10、article boundary r10、identity boundary r09、ADR-0020、identity review r06、当前 identity design r07 及 article review r01-r03 检查：

- 当前 DTO 与真实消费者字段、digest 重算和二次读取合同；
- 8 条 article-owned US 和 S1-S16；
- r01 F001-F004、r02 N001 的关闭状态；
- 永久删除、通知状态机、cursor 分页和作者最小投影；
- 16 个 REST endpoint、数据所有权、审计、隐私和失败收敛。

## 3. Findings

### [HIGH] AM-DES-R04-N001：ArticleDeletionReplayStatus 与当前 identity r07 消费合同不兼容

- finding ID：`AM-DES-R04-N001`
- 对象/修订：article-management design r04
- 位置：`design.md:817-841`、`design.md:949-969`；消费者 `identity-auth/design.md:1475`、`1481-1487`、`1738-1739`
- 违反依据：
  - 评审清单要求跨模块接口字段、状态和消费者可实施且一致；
  - accepted S16 边界要求 identity 只消费 article-owned DTO，不能读取 article 内部表；
  - architecture r10、ADR-0020 要求 canonical digest 覆盖 Journal head、inventory、operation/audit、计数和时间。
- 现状：
  - article r04 DTO 只输出 run、backup、namespace、status、version/digest、三项 tombstone 计数和 `completed_at`；
  - DTO 不输出 `evidence_ref`、Journal head、inventory scope/cutoff/digest、operation/audit digest；
  - article canonical 字段没有 identity r07 要求的 `deletion_operations_reconciled_count`、`audit_events_verified_count`，且使用 `replayed`，消费者使用 `tombstones_replayed`；
  - 当前 identity r07 要求从 DTO 重算 digest，保存 version/digest/ref，并在 MarkConsumed 前再次读取相同 version/digest/ref。
- 触发：article r04 与当前 identity r07 集成恢复流程。
- 影响：identity 无法按自身合同重算 canonical digest 或完成 evidence ref 二次一致性比较；实现只能停在恢复 Gate，或越界读取 article 内部表/削弱验证。若通过削弱验证绕过，错 backup 或 stale ready 可能开放旧备份，重新暴露已永久删除文章。
- 修复建议：这是跨设计方向对齐，不由 Reviewer 代拍。固定一份精确、版本化 DTO/canonical schema，并同步生产者和消费者：
  1. 若 identity 必须自验证 digest，则 article DTO 输出相同 canonical 字段、统一计数名称和稳定的 opaque evidence ref；
  2. 若 digest 对 identity 是 article-owned opaque attestation，则 identity r07 删除重算和 evidence ref 强依赖，只比较 accepted 边界允许的最小 DTO；
  3. 任一路线都不得让 identity 直接读取 Journal、inventory、DeletionOperation、AuditEvent 或 evidence 表。
- 验证：生产者/消费者契约测试使用同一 schema fixture；缺字段、字段改名、未知版本、digest 不匹配、错 backup 和二次读取变化均 fail closed。

### [MEDIUM] AM-DES-R04-N002：`completed_at` 参与 digest，但幂等复用算法形成循环依赖

- finding ID：`AM-DES-R04-N002`
- 对象/修订：article-management design r04
- 位置：`design.md:949-969`、`design.md:1622-1631`、`design.md:1844-1845`
- 违反依据：article boundary r10、identity boundary r09 和 ADR-0020 均要求同一 run+backup 在输入不变时幂等读取或重算同一 evidence。
- 现状：
  - `completed_at` 明确参与 `evidence_digest`；
  - Step 6 要求先形成 snapshot，并在“digest 完全相同”时才复用旧 `completed_at`；
  - 新一次重算若先生成当前完成时间，digest 必然变化；而要得到旧 digest 又必须先知道并使用旧 `completed_at`；
  - 唯一约束包含 `evidence_digest`，因此新时间产生的新 digest 可以合法追加另一行。
- 触发：同一 run+backup 在首个 ready 后重复调用 `ReplayAndVerify`，或响应丢失后重试。
- 影响：输入未变化也可能产生新的 digest/evidence；identity 的二次重读可能把它视为变化并拒绝 MarkConsumed，破坏幂等续跑并造成恢复无法开放。
- 修复建议：先以不含 `completed_at` 的稳定输入快照定位当前 evidence；全部输入完全一致时复用首份 evidence 的 `completed_at` 和 digest。只有 covered input 变化时才生成新的 `completed_at`。可增加稳定 input fingerprint/唯一约束，避免并发重算重复插入。
- 验证：同 run+backup 连续、并发和崩溃后重试 `ReplayAndVerify`，在输入不变时必须返回同一 evidence/digest/completed_at；任一 covered input 变化时才生成新 evidence。

### [MEDIUM] AM-DES-R04-N003：同一 run 的 backup 绑定缺少并发安全约束

- finding ID：`AM-DES-R04-N003`
- 对象/修订：article-management design r04
- 位置：`design.md:939-947`、`design.md:967`、`design.md:1622-1624`
- 违反依据：accepted S16 合同要求同一 run 绑定唯一 backup，backup A 的 ready 不得用于 backup B；状态所有权和失败收敛必须可由数据约束实施。
- 现状：
  - `DeletionReplayRun` 的主键是 `(run_id,backup_ref,namespace)`，该键允许同一 run 同时存在多个 backup；
  - Step 1 按完整复合键锁定或创建，再检查“同 run 已绑定其他 backup”，但没有 `(run_id,namespace)` 唯一约束、按 run 的锁或等价串行化机制；
  - 两个并发请求可同时看不到对方并分别插入 backup A/B。
- 触发：同一 `run_id` 并发调用 `ReplayAndVerify(run,backupA)` 与 `ReplayAndVerify(run,backupB)`。
- 影响：同一恢复 run 可出现多个 running/ready 行，直接破坏 run→backup 不可变绑定；错 backup 拒绝和状态读取结果依赖竞态，恢复证据不再是唯一真相。
- 修复建议：让 run 维度成为唯一所有权键，例如对 `(run_id,namespace)` 建唯一约束并把 `backup_ref` 设为首次绑定后不可变字段，或在创建前取得按 run 的数据库/advisory lock；冲突事务必须返回 failed，不能创建第二行。
- 验证：并发双 backup 压测只能有一个绑定成功；失败方稳定返回 failed；同 backup 并发调用继续满足 N002 的同证据幂等要求。

## 4. 历史 finding 与业务回归

### 4.1 历史 finding

| finding | r04 回归状态 | 证据摘要 |
|---|---|---|
| `AM-DES-R01-F001` 永久删除跨备份恢复 | closed，未因业务路径回归 | pending、DeletionOperation、外部 tombstone、finalizer、ReplayAll、inventory/key 留存仍在；S16 新证据层存在本报告 N001-N003 |
| `AM-DES-R01-F002` 作者修改 violating 文章 | closed，无回归 | 作者受控读取/修改仍允许；治理字段不变 |
| `AM-DES-R01-F003` cursor 可见性分页 | closed，无回归 | 候选分批扫描到 `limit+1` 可见项或耗尽；公共 cursor 仍基于最后返回项 |
| `AM-DES-R01-F004` 通知 claim/lease/重试 | closed，无回归 | 五态 recipient、短事务 claim、Gate、lease 和 at-least-once 风险仍完整 |
| `AM-DES-R02-N001` 作者治理信息过度披露 | closed，无回归 | AuthorEditView allowlist、ETag Header、管理员完整投影和成员统一 404 仍分离 |

### 4.2 8 US、S1-S16 与关键能力

- 8 条主责 US 在 design 和 stories 中均可定位，Story 四块结构和依赖未发现新增漂移。
- S1-S15 的身份、可见性、canonical 删除、邮件、审计、维护模式与 keyring 方向未发现本轮回归。
- S16 的所有权方向正确：evidence、Journal、operation/audit 对账归 article；identity 只应消费 DTO。阻断项是 schema、幂等和并发绑定的可实施性。
- 永久删除仍要求 tombstone 与物理 DELETE 双完成才返回 204；pending 对成员、作者和管理员均不可见；无文章恢复入口。
- 通知仍为两收件人独立五态、Gate 关闭不 claim、SMTP 事务外调用、失败不回滚转移。
- 分页仍按 `(created_at,id)` 稳定倒序并处理长隐藏区间。
- 作者投影仍只返回 `title,body,moderation_status,editable`，ETag 仅在 Header；管理员治理字段没有串入。
- REST 接口总表为 16 个 endpoint，与 r03 回归基线一致。

## 5. 静态验证与未验证风险

- design 有 27 个 PlantUML 块，`@start*`/`@end*` 数量均为 27；未发现字面 `\n` 或 `linetype ortho`。
- design 中 23 个本地 Markdown 文件链接均存在。
- stories 依赖图为文本图，人工核对未发现环。
- 本机无 `plantuml` 命令或 PlantUML JAR；虽有 Java 21，但未执行语法检查或 PNG 实渲染，不能证明渲染器无空框或布局问题。
- 当前无业务代码、schema、迁移、契约生成物或测试；项目未登记检查命令，未运行不存在的 build/lint/unit/integration/E2E。
- r03 原始文件快照未独立保留，无法重建逐字节 r03→r04 diff；本次依赖 r03 PASS 报告、checkpoint digest、r04 修订记录，并对 S16 新落点和既有关键路径做回归。
- identity r07 仍待并行复审；其固定消费者快照若变化，N001 必须针对新 revision 重新核对，不能把本报告自动套用到漂移对象。
- 跨 Journal、inventory 与 PostgreSQL 的真实并发/崩溃行为尚未演练；静态设计不能代替恢复演练。

## 6. 最终结论与复审条件

**最终结论：NOT PASS。**

- CRITICAL：0
- HIGH：1
- MEDIUM：2
- r01/r02 历史 findings：保持 closed
- 新阻断风险：DTO/canonical schema 消费冲突、`completed_at` 幂等循环、同 run 错 backup 并发绑定

复审条件：

1. 提供新的 article design/stories 精确 revision、完整 SHA-256 和规范 artifact digest；
2. 固定 article 与 identity 的同一份 DTO/canonical schema；若 identity r07 修订，附其固定 revision/hash 和并行复审结果；
3. 明确输入不变时复用原 `completed_at`/digest 的幂等算法及数据约束；
4. 为同一 run 的 backup 首次绑定增加并发安全的唯一约束或锁合同；
5. 增加生产者/消费者契约、重复/并发 ReplayAndVerify、错 backup 和三个崩溃点的验证设计；
6. 回归 8 US、S1-S16、永久删除、通知、分页和作者投影。

该结论只针对固定 article r04 对象，不替代 Lead/用户的方向决定、风险接受、成果确认或实施授权；既有实施 Gate deferred 状态不因本报告改变。
