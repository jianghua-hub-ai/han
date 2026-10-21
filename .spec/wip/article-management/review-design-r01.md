# article-management 详细设计独立审查报告 r01

- 审查日期：2026-10-20
- objectType：design
- 主对象：`.add/project/knowledge/design-workspace/article-management/design.md`
- 主对象修订：`article-management-design-ir01-r01`
- 附属消费者：`.add/project/knowledge/design-workspace/article-management/stories.md`
- 附属对象修订：`article-management-stories-ir01-r01`
- 结论：**NOT PASS**
- 写入边界：本报告为唯一写入；未修改 design、stories 或任何上游制品。

## 1. 对象固定与摘要核对

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `8f0c9fc611e31be986f023efeb73379c4ac9bc587856690bee088eac04715467` | `8f0c9fc611e31be986f023efeb73379c4ac9bc587856690bee088eac04715467` | 一致 |
| `stories.md` | `6b4ba07dcf1b30b5f997bf16f3c4f06f237a8b05213b0369a9e02f547eb63df5` | `6b4ba07dcf1b30b5f997bf16f3c4f06f237a8b05213b0369a9e02f547eb63df5` | 一致 |

`design.md` 没有 `<!-- ADD:CHECKPOINTS -->` 标记；按 `## 确认记录` 作为确认段起点、排除确认段并把尾部规范化为单个 LF 后，artifact digest 为：

`sha256:3f5aee0385f67130aa846bae9bdc49d4c5cb803cdccda926f483522c1a79a845`

与 Lead 值一致。作者报告的 `3d32...` 不能由当前完整文件、确认段前原始前缀或规范化算法复现，属于摘要算法/旧内容摘要差异；本报告只对完整 SHA-256 固定对象下结论，未混用修订。

## 2. 两轮实际检查范围

### Round 1：固定对象全文

完整检查 design 1503 行和 stories 206 行，覆盖：

- 8 条主责 US 及每 Story 的实现、组件、接口、数据四块；
- 15 个 REST endpoint、权限、最小披露、错误与审计；
- cursor、ETag/If-Match、创建/修改/删除重试；
- 正交状态、物理删除、违规与无归属治理；
- canonical 删除、通知快照、TLS、ProcessingGate、隐私与 XSS 输出；
- TransferNotice dispatcher、事务和生命周期。

### Round 2：消费者与恢复回归

沿 requirements r16、architecture r08、两份 boundary r07、ADR-0001～0019、identity design r05 PASS 基线及 S1～S15 检查：

- identity Principal、作者状态、转移资格、墓碑联系和清理端口；
- canonical 删除及 `retained_email` 清理；
- Recovery Journal/readiness 与数据库恢复；
- Article、TransferNotice、AuditEvent 生命周期；
- stories 依赖及 task 可拆性。

## 3. Findings

### [HIGH] AM-DES-R01-F001：旧备份恢复会使已“永久删除”的文章重新出现，设计未提供满足需求的闭环

- 位置：`design.md:76-89`、`208-216`、`1220-1225`、`1252-1257`
- 违反依据：requirements r16 `US-ARTICLE-04.1`、`US-IDENTITY-03.2/03.8/03.12` 的“永久不可恢复”；article boundary S11；ADR-0010 的永久删除终态。
- 现状：当前业务库执行物理 DELETE，但备份包含删除前 Article。设计明确承认没有跨业务库删除事实日志，并把“如何证明恢复点之后的直接文章删除不被旧备份带回”留作 AM-Q8。现有 Recovery Journal 只保护恢复批次/身份信任根，不记录文章删除事实。
- 触发：文章在备份后被作者或管理员永久删除，随后 PostgreSQL 恢复到删除前备份并重新开放服务。
- 影响：已承诺不可恢复、对所有用户不可见的内容重新进入 Article 权威表，并可能再次被列表或详情读取。不能以“RPO 风险”反向放宽已接受业务需求。
- 最小修复：这是方向决策，不能由 Feature Design 局部假设。需在 architecture/ADR 固定可验证方案，例如业务数据库恢复域外的文章删除 Journal/不可变删除清单，或保证恢复后重放全部删除事实的等价机制；在删除事实不可证明时必须保持 readiness 关闭，并明确恢复责任和证据来源。
- 验证：真实演练“备份→永久删除普通/违规/无归属文章→恢复旧备份”，证明服务开放前删除事实被重放，正文、列表、详情和管理入口均无法恢复内容。

### [HIGH] AM-DES-R01-F002：设计擅自禁止作者修改自己的违规文章，违反已确认修改需求

- 位置：`design.md:773-827`；`stories.md:65-84`
- 违反依据：requirements r16 `US-ARTICLE-03.1` 明确允许文章作者修改自己的现有、未按删除规则删除的文章；上游只禁止非作者修改，没有把违规状态设为禁止修改条件。
- 现状：设计要求文章处于“ordinary visible state”，并明确让违规文章通过作者 PUT 返回 404，声称新增此能力需回需求。
- 触发：active 作者的一篇文章被管理员标为 violating，作者随后提交满足规则的标题和正文修订。
- 影响：已确认 US 无法完成；作者不能修正违规内容。该限制不是最小披露所必需，因为修改可保持 `violation_state=violating`，文章仍可继续隐藏。
- 最小修复：允许作者修改自己所有未物理删除且仍有 `author_id` 的文章；修改只更新 title/body/version，不清除违规事实或其他隐藏约束。若作者需要取得当前 ETag/内容，定义受控“作者自己的隐藏文章”读取/编辑合同，不向其他成员披露。
- 验证：覆盖 active 作者修改普通与 violating 文章；违规文章修改后仍仅管理员可见、violation/category 不变；非作者和管理员非作者 PUT 继续拒绝。

### [MEDIUM] AM-DES-R01-F003：成员 cursor 分页没有固定“可见性过滤先于 limit”的可实施算法

- 位置：`design.md:520-534`、`717-771`
- 违反依据：requirements r16 要求稳定倒序分页；设计清单要求接口、状态和消费者可实施。
- 现状：时序先读取 Article candidates，再批量读取作者状态并应用 VisibilityPolicy；SQL 片段只定义 keyset 和 `LIMIT effective_limit+1`。设计没有说明成员可见性是在 SQL/limit 前过滤，还是对候选页事后过滤，也没有定义隐藏候选密集时的继续扫描与 cursor 取值。
- 触发：排序窗口内混有违规文章、停用作者文章或 held 无归属文章。
- 影响：事后过滤可能返回空/短页，错误生成 next_cursor，跳过后续可见文章或重复扫描；分页验收无法稳定证明“不漏项、不重复”。
- 最小修复：固定一种算法：在同一数据库查询中连接 User 并把完整可见性谓词放在 keyset/limit 之前；或定义有上限的迭代扫描，cursor 基于最后扫描项并继续直到取得有效页/耗尽。不能只写“QueryService 过滤”。
- 验证：可见/违规/停用/held 交错、同时间戳、并发新增、多页与连续隐藏区间测试，断言可见文章无重复或遗漏。

### [MEDIUM] AM-DES-R01-F004：TransferNotice 的 claim、自动重试和终态合同彼此不完整

- 位置：`design.md:501`、`1010-1082`、`645-662`、`1249`、`1268`
- 违反依据：ADR-0011 best-effort 失败收敛；可实施的数据生命周期、并发和恢复合同。
- 现状：
  - Dispatcher 声称通过“行锁或原子 claim”防并发重复，但表中没有 processing/lease/claim owner/next attempt 字段；
  - 状态定义 `pending → accepted|failed`，`failed` 看起来是终态；
  - FMEA 又承诺 SMTP 不可用后“后续自动处理”，attempt_count/自动重试参数仍存在。
- 触发：并发 dispatcher、SMTP 瞬时失败、进程在 SMTP accepted 后或保存结果前崩溃。
- 影响：实现者可能长期持有数据库事务跨 SMTP、重复投递、使通知永久停在 failed，或由多个 worker 同时发送；无法据当前 schema 安全拆分 dispatcher task。
- 最小修复：固定每个收件人的状态机和 claim 方案，例如 `pending/retry_wait/processing/accepted/failed_terminal`、`locked_until/claimed_by/next_attempt_at`，原子 claim、租约恢复、最大尝试和 Gate 重开语义。明确 accepted 后崩溃的重复投递是残余风险及其审计证据。
- 验证：双 worker 竞争、进程崩溃、租约到期、Gate closed→open、瞬时/永久 SMTP 失败和两收件人独立状态测试。

## 4. 待决定与已接受风险

| 项 | 处理 |
|---|---|
| AM-Q1 标题/正文格式与长度 | 阻塞 validator 与前端渲染最终实现；不阻塞对象/接口结构评审 |
| AM-Q2 分页/容量数值 | 可拆配置和测量任务；阻塞容量验收 |
| AM-Q3 删除二次确认 | 只影响 UX；API 仍独立鉴权、If-Match 和审计 |
| AM-Q5 通知人工重发/查询 | 当前范围明确排除；新增时回需求与边界 |
| AM-Q6/Q7 留存与处理输入 | 部署/隐私前必填；Gate 缺失时邮件不发送 |
| 创建超时重复 | ADR-0015/S9 已接受不把 Idempotency-Key 扩展到文章写入。当前只能把重复创建作为显式残余风险；若要求安全自动重试，需回 ADR 选择客户端 request ID 或扩展幂等合同，Reviewer 不在本设计中代拍 |

## 5. 非阻断建议

1. 明确强 ETag 解析只接受单个精确标签，拒绝 weak ETag、非法多值和未支持的 `If-Match:*`，避免实现差异。
2. REST 文档补充 `Content-Type: application/json`、`X-Content-Type-Options: nosniff`；未来前端必须使用文本节点/上下文编码，不用 `innerHTML` 直接渲染 title/body。
3. cursor 虽不承担授权，仍建议集中 codec 并限制解码后时间/ID范围，避免不同列表实现漂移。

## 6. 已通过的重点检查

- 物理删除不保留软删正文、版本历史或恢复 API。
- AuditEvent 只保存主体、动作、目标 ID、结果、trace 和删除前 version；未保存 title/body、邮箱或内容摘要。
- 正交状态、数据库 CHECK、canonical 删除、违规保留和 `retained_email` 清理顺序一致。
- 15 个 endpoint 均有消费者、权限与主要错误；管理接口要求 Session、CSRF、管理员角色和目标状态。
- 写接口除创建外统一使用强 ETag/If-Match；内部 canonical 删除不伪装为 HTTP 条件请求。
- 标题/正文按 JSON 文本输出，服务端不解释 HTML；未引入富文本执行面。
- 转移先提交 Article/Notice/Audit，再异步 best effort 发送；通知失败不回滚转移。
- ProcessingGate 关闭时不连接 SMTP，通知用途固定且不营销。

## 7. 静态验证与限制

- design 共 25 个 PlantUML 块；每块有且仅有一个 `@start*`/`@end*`，未发现字面 `\n` 或 `linetype ortho`。
- stories 使用文本依赖图，无 PlantUML；人工核对未发现依赖环。
- design 中 15 个本地 Markdown 链接均存在。
- 本机无可用 PlantUML CLI/JAR，未做 `-checkonly` 或 PNG 实渲染。
- 当前无业务代码、数据库 schema、迁移、API schema 或测试；未运行不存在的构建、lint、单元、集成或 E2E。
- identity design r05 的独立 PASS 只作为消费基线；本报告未重新评审其内部实现。

## 8. 最终结论与复审条件

**最终结论：NOT PASS。**

- HIGH：2
- MEDIUM：2
- 未发现 CRITICAL。

在 AM-DES-R01-F001/F002 关闭前，design 不得转为 `implementable`。复审需提供：

1. 新 design/stories 精确 revision、完整 SHA-256 和规范 artifact digest；
2. F001 的 accepted architecture/ADR 决定及旧备份删除重放验证合同；
3. F002 的作者隐藏文章修改/ETag/最小披露回归；
4. F003 的可见性分页算法与组合状态测试；
5. F004 的通知 claim/lease/重试状态机和故障注入矩阵；
6. 若上游修订，附 requirements/architecture/boundary/ADR 新 checkpoint 及 S1～S15 回归。

独立结论不代替用户/Lead 的成果接受、风险决定或实施授权。
