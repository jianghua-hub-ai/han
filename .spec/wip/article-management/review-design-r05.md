# article-management design r05 独立回归复审报告

- 复审日期：2026-10-20
- objectType：design
- Reviewer：独立非作者 Reviewer
- 原报告：`.spec/wip/article-management/review-design-r04.md`
- 原 findings：`AM-DES-R04-N001`～`AM-DES-R04-N003`
- 主对象：`article-management-design-ir01-r05`
- 附属对象：`article-management-stories-ir01-r05`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories、identity r08、历史报告或 upstream。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `9991efd591f3ac37deb2ed596b218ddce08e74c6ad9a914ce77967215b0c9813` | `9991efd591f3ac37deb2ed596b218ddce08e74c6ad9a914ce77967215b0c9813` | 一致；与委托相符 |
| `stories.md` | `4504ba5a2904ef927f5cfbef89804878a59bfb9de94aa1eb37b815e2251503a7` | `4504ba5a2904ef927f5cfbef89804878a59bfb9de94aa1eb37b815e2251503a7` | 一致；与委托相符 |

按 `## 确认记录` 前正文排除确认段，并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:2ce9bda301a02738695126f5a1d05b2870bdf3c0303844a66ac3501e04a4936a`

与 Lead artifact 一致，对象无漂移。

### 1.2 上游与消费者快照

| 依据 | 修订/状态 | 当前读取 SHA-256 或 checkpoint |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r11`；accepted checkpoint | `decision-0485d8f9ac9ce04ce734dfc8bdc6ecf5b96b226f81fa21e781afb4d7717ea3a8`；当前载体 `6466f88b579b56b90e2aa7d658657a117337bf6a0057321b7f002b16b6d9edce` |
| article boundary | `article-management-boundary-ir01-r11` accepted | `ad94c1f54a192b22c8cd6ada49a21f7a3c471d6790b79e1d7113008ac7e702f7` |
| identity boundary | `identity-auth-boundary-ir01-r10` accepted | `33cf72545443130679167488c37b92dba74ef579c6829a841324b73b29c53fd1` |
| identity consumer | `identity-auth-design-ir01-r08` fixed | `6bdc960f323736e2397ea321e64b8718a483604aaeac5d3bfa59af08abb0c0ca` |
| identity stories | r08 当前固定附属对象 | `3d99ef5e8f7e2022e3348c82e73c0c66768da14883663fd27a51a576c3546508` |
| ADR-0020 | accepted，唯一 v1 schema 决定 | `content.md`：`b152d6ec2765a87ba144a474997ed5c82011b1b8af6c9b457e56675402617c15` |
| article 原报告 | r04 NOT PASS | `dc3de7b7d9035634810fd3804d068a1342e6b524912dd3ca20446c80cc3e9c7c` |
| identity 消费者前序复审 | r07 NOT PASS，DTO/canonical 冲突 | `review-design-r07.md` |

architecture r11 已存在明确 accepted checkpoint，但文件顶部 frontmatter 和确认段说明仍写 `pending-confirmation`。本报告按委托及 accepted checkpoint 消费 r11；该载体元数据漂移属于 upstream 风险，不计为 article r05 finding。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

逐项检查：

1. N001：20 字段名称、顺序、JSON 类型、canonical 前 16 字段、四个排除字段、计数名、fixture digest 及 identity r08 逐字段消费；
2. N002：`completed_at` 排除、稳定 `input_digest`、输入不变时 ref/digest/time 复用、并发和崩溃续跑；
3. N003：`DeletionReplayRun` 唯一 `(run_id,namespace)`、backup 首绑不可变、原子 insert/lock、并发错 backup。

重点检查 `design.md:852-916`、`1015-1050`、`1701-1734`、`1923-1935` 和 `stories.md:16-23`、`92-118`。

### Round 2：消费者与历史回归

沿 architecture r11、两份 accepted boundary、ADR-0020、固定 identity r08 和 identity review r07 核对生产者/消费者；并回归：

- 8 条 article-owned US 与 S1-S16；
- r01 F001-F004、r02 N001；
- 永久删除、canonical 删除、通知状态机；
- cursor 分页、作者最小投影；
- 16 个 REST endpoint、审计、安全、隐私和失败收敛。

## 3. 原 findings 复审

### [HIGH] AM-DES-R04-N001 — still-open

- 原 finding：ArticleDeletionReplayStatus 与 identity 消费合同不兼容。
- 对象/修订：article-management design r05。
- 位置：
  - article provider：`design.md:859-916`、`1033-1050`、`1714-1720`；
  - identity r08 consumer：`identity-auth/design.md:1483-1508`、`1510-1539`。
- 已修复部分：
  - 两侧字段名称、顺序和数量均固定为 20；
  - canonical 固定为前 16 字段；
  - `status/evidence_digest/evidence_ref/completed_at` 均排除；
  - 重放计数统一为 `tombstones_replayed`；
  - operation/audit count 与 `evidence_ref` 已补齐；
  - article fixture canonical bytes 的 SHA-256 经 Reviewer 独立计算，确为 `169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`。
- 仍未关闭的逐字段冲突：
  1. article 将 `inventory_scope` 固定为 JSON **string**，fixture 值为 `"recoverable-backups-before-cutoff"`；
  2. identity r08 将同一字段固定为非 null JSON **object**，并要求递归 canonical object 编码；
  3. 因此 article 的 `169e0f...` canonical bytes 无法被 identity r08 按其固定 schema 接受或重算；
  4. article 对 unknown/failed DTO 要求 evidence 不可用字段使用零值/空字符串，而 identity r08 对 `evidence_digest`、`evidence_ref`、`completed_at` 分别要求合法摘要、非空字符串和 RFC 3339 时间，非 ready DTO 的合法编码仍不一致。
- 触发：
  - identity r08 严格解析 article r05 的 ready fixture；
  - 或 article 返回 unknown/failed 的完整 20 字段 DTO。
- 影响：
  - 严格实现时 identity 会把 article DTO 判为类型错误，恢复永久停在 Gate；
  - 若实现者通过 string↔object 强制转换、忽略类型或放宽 canonical 规则，会产生不同 canonical bytes/digest，破坏“双方仅凭 DTO 重算同一摘要”的安全合同；
  - unknown/failed 只能以“格式错误”而非已定义状态被消费，接口语义不再统一。
- 违反依据：
  - ADR-0020 要求固定 JSON 类型和共享 schema fixture；
  - architecture r11 与两份 boundary 要求生产者/消费者共用唯一 v1 schema；
  - 评审清单要求跨模块字段、类型、失败语义和真实消费者完全一致。
- 修复建议：这是跨设计 schema 决策，不由 Reviewer 选择哪侧为准。生产者、identity r08 和 ADR/boundary 必须引用同一份实际 fixture：
  1. 明确 `inventory_scope` 唯一 JSON 类型及 canonical 编码；
  2. 用同一 ready fixture 在两侧重算同一 digest；
  3. 明确 unknown/failed 时 20 个字段的合法值约束，避免一侧要求空值、另一侧禁止空值；
  4. 若改变 article 当前 fixture，必须重新固定 fixture digest；不得继续沿用不对应新 bytes 的 `169e0f...`。
- 验证：
  - article 生成的同一 JSON fixture 可被 identity r08 严格解析，无类型转换；
  - 双方从前 16 字段得到相同 digest；
  - ready、unknown、failed 各有完整 schema fixture；
  - 缺字段、额外字段、null、错误类型和字段别名均 fail closed。

原 N001 的字段集和 digest 范围已大幅收敛，但“逐字段兼容”仍不成立，因此保持 open。

### AM-DES-R04-N002 — closed

- 修复证据：`design.md:1021-1050`、`1707-1712`、`1929-1934`。
- `completed_at` 已从 canonical 和 `input_digest` 排除，只在首份 evidence INSERT 时生成。
- 输入不变时，run 的 `input_digest/current_evidence_ref` 直接复用同一 ref/digest/completed_at，不读取新时钟。
- 输入变化时才生成或复用新 digest 对应 evidence，并原子更新 current 指针。
- Evidence 唯一约束与 run 行锁覆盖连续、并发和响应丢失重试。

原“completed_at 参与 digest 导致幂等循环”的路径已关闭。

### AM-DES-R04-N003 — closed

- 修复证据：`design.md:1015-1027`、`1703-1706`、`1933-1934`。
- `DeletionReplayRun` 唯一键已改为 `(run_id,namespace)`；`backup_ref` 不属于唯一键且首次绑定后不可修改。
- 创建使用 `INSERT ... ON CONFLICT DO NOTHING`，随后按唯一键 `SELECT ... FOR UPDATE`。
- 并发不同 backup 只能由首次插入方绑定成功；失败方锁内比较后稳定返回 failed，不能创建第二行或改写 run。
- 相同 backup 并发通过 run 锁、evidence 唯一约束和 current 指针收敛到同一 evidence。

原“同 run 并发绑定多个 backup”的路径已关闭。

## 4. 新 findings

未发现独立于 `AM-DES-R04-N001` 的新增 CRITICAL、HIGH 或 MEDIUM finding。

## 5. 历史回归

- r01 F001-F004 与 r02 N001 保持 closed。
- 8 条主责 US 均保留 Story 四块和分配表。
- S1-S15 的身份、状态、canonical 删除、通知、审计、恢复信任根和 keyring 合同未发现回归。
- 永久删除继续要求 pending 隐藏、外部 tombstone 与物理 DELETE 双完成；旧备份开放前 ReplayAll；无文章恢复入口。
- 通知继续使用两 recipient 五态、短事务 claim/lease、ProcessingGate、强制 TLS 和 at-least-once 残余风险。
- 分页继续按 `(created_at,id)` 倒序，候选分批扫描至 `limit+1` 可见项或耗尽。
- 作者响应仍严格限制为 `title,body,moderation_status,editable`，ETag 仅在 Header；管理员完整治理投影和成员 404 保持分离。
- REST 接口总表仍为 16 个 endpoint。

## 6. 静态验证与限制

- design 共 2096 行，stories 共 221 行。
- design 有 27 个 PlantUML 块，`@start*`/`@end*` 均为 27；未发现字面 `\n` 或 `linetype ortho`。
- design 中 25 个本地 Markdown 文件链接均存在。
- stories 文本依赖图人工核对未发现环。
- Reviewer 独立计算 article fixture canonical bytes 长度为 546 bytes，SHA-256 为 `169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`。
- 本机无 `plantuml` 命令或 PlantUML JAR，未执行语法检查或 PNG 实渲染。
- 当前无业务代码、schema、迁移、API schema 或测试；项目未登记检查命令，未运行 build/lint/unit/integration/E2E。
- 本报告是静态设计复审，不能替代真实 PostgreSQL 并发、Journal/inventory、崩溃续跑或恢复演练。
- architecture r11 顶部状态/说明与 accepted checkpoint 不一致，可能误导只读取 frontmatter 的消费者；需由 upstream owner 纠偏，不计入本对象 finding。

## 7. 最终结论与复审条件

**最终结论：NOT PASS。**

- `AM-DES-R04-N001`：still-open（HIGH）
- `AM-DES-R04-N002`：closed
- `AM-DES-R04-N003`：closed
- 新增独立阻断 finding：无
- 历史业务 findings：未发现回归

复审条件：

1. 提供 article 新 revision、design/stories 完整 SHA-256 和规范 artifact digest；
2. 固定 article 与 identity 的同一份 20 字段 JSON schema，尤其统一 `inventory_scope` 类型；
3. 提供双方完全相同的 ready、unknown、failed fixtures 及对应 canonical bytes/digest；
4. 保持已关闭的 completed_at/input_digest 幂等和 run→backup 并发绑定合同；
5. 重新执行 identity r08 逐字段兼容、8 US、S1-S16 和历史 finding 回归。

该结论只针对固定 r05 对象与本报告记录的 fixed identity r08/upstream 快照，不替代 Lead/用户的 schema 决策、风险接受、成果确认或实施授权。
