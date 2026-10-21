# article-management design r06 独立回归复审报告

- 复审日期：2026-10-20
- objectType：design
- Reviewer：独立非作者 Reviewer
- 原报告：`.spec/wip/article-management/review-design-r05.md`
- 原开放项：`AM-DES-R04-N001`
- 回归项：`AM-DES-R04-N002`、`AM-DES-R04-N003`
- 主对象：`article-management-design-ir01-r06`
- 附属对象：`article-management-stories-ir01-r06`
- 结论：**PASS**
- 写入边界：仅新建本报告；未修改 design、stories、identity r09、历史报告或 upstream。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `71b8953790b828d67373cae1eb020c26d5e4d6c10b434c057e582fdf74d17a51` | `71b8953790b828d67373cae1eb020c26d5e4d6c10b434c057e582fdf74d17a51` | 一致；与委托相符 |
| `stories.md` | `44bf8bb237ec4ba7cd79e0237ccac92fefb1d7fdc2313f1e961f1fef5ec7b307` | `44bf8bb237ec4ba7cd79e0237ccac92fefb1d7fdc2313f1e961f1fef5ec7b307` | 一致；与委托相符 |

按 `## 确认记录` 前正文排除确认段，并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:ed1a13b7f3d2e9f2c63b47d66d26469ea19315c42ace9cbe49db9f206880e4e0`

与 Lead artifact 一致，对象无漂移。

### 1.2 上游与消费者快照

| 依据 | 修订/状态 | 当前读取 SHA-256 或 checkpoint |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r12` accepted | checkpoint `decision-0347e1026fd9148ad16f66750023cb50445a1d7c133db6d82669ea14865b318f`；当前载体 `5cac5b922fbe976640a5617a9a24136abb4283ad7ad866a6ab933498b5bd1dce` |
| article boundary | `article-management-boundary-ir01-r12` accepted | `a4d70c060a7d3d8fe986bc32b7047b28c57e2fe8b19733d423f5a56814ede5e4` |
| identity boundary | `identity-auth-boundary-ir01-r11` accepted | `fb5197625eafc07bf0a98e5f09abd7c60e018ee6507bc1d47b88dc40b651767c` |
| identity consumer | `identity-auth-design-ir01-r09` fixed | `d05132a4bc4ecacf0243059d53eafe8b963c79383d9a07209dd903350b823d85` |
| identity stories | r09 固定附属对象 | `d8ac23a5eb7bf535b2111bc3411b6874e455065b237695470b3addefdaab4ad5` |
| ADR-0020 | accepted，最终 tagged-union schema | `content.md`：`a0594229fcfda0b8f243a476024bf51825f8dfb12671b41675fc270fa12dbf88` |
| article 原报告 | r05 NOT PASS | `40e8a864fcb58ee6ceeacd5d91c3ed4e5aa3ccd848e4aab67f9d9d50f810f6a5` |
| identity 前序复审 | r08 NOT PASS | `5292bb4341b16f7f42e0ab1e92b7d54dafba3c0f2dcd7136d8fdb3e194acd127` |

architecture r12 已存在明确 accepted checkpoint，但当前载体顶部仍保留 `pending-confirmation` 说明。本报告按委托及 accepted checkpoint 消费 r12；该 upstream 元数据漂移不计为 article r06 finding。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

检查 `design.md:855-948`、`1046-1081`、`1732-1765`、`1958-1969` 及 stories S16：

- strict tagged union 是否保证每次恰好一个 variant；
- Ready 是否固定 20 字段、全部非 null；
- `inventory_scope` 是否为 string；
- canonical 是否仅含前 16 字段；
- fixture/canonical bytes/digest 是否固定；
- Unavailable 是否严格只有 6 字段且无伪 evidence 字段；
- N002 completed_at/input_digest 幂等；
- N003 run→backup 唯一及并发约束。

### Round 2：真实消费者与历史回归

沿 architecture r12、两份 accepted boundary、ADR-0020 和固定 identity r09 逐字段对账；回归：

- 8 条 article-owned US 与 S1-S16；
- r01 F001-F004、r02 N001；
- 永久删除、canonical 删除、通知状态机；
- cursor 分页、作者最小投影；
- 16 个 REST endpoint、审计、安全、隐私和恢复失败收敛。

## 3. 原 findings 复审

### AM-DES-R04-N001 — closed

- 修复证据：
  - article：`design.md:855-948`、`1734-1751`、`1962-1969`；
  - identity r09：`identity-auth/design.md:1485-1575`、`1825-1827`、`2075-2078`；
  - architecture r12、article boundary r12、identity boundary r11、ADR-0020。
- Ready variant：
  - article 与 identity 的字段名称、顺序均为相同 20 项；
  - `status` 固定 `ready`，全部字段必填且非 null；
  - `inventory_scope` 均为 JSON string；
  - canonical 均严格使用前 16 字段，排除 `status/evidence_digest/evidence_ref/completed_at`。
- 共享 fixture：
  - article 与 identity 的 canonical bytes 逐字节相同，均为 546 UTF-8 bytes；
  - Reviewer 独立复算 SHA-256 为  
    `169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`；
  - 与两侧声明及 ADR-0020 一致。
- Unavailable variant：
  - 两侧字段名称和顺序均严格为 `evidence_version,run_id,backup_ref,namespace,status,reason_code`；
  - status 只允许 `pending|unknown|failed`；
  - 禁止 canonical、digest、ref、time、null、空字符串或零值伪装 Ready；
  - identity 收到后立即 fail closed，不调用 canonical digest 计算器。
- `ReplayAndVerify` 与 `ReadStatus` 每次只能构造一个 variant；独立 serializer 阻止字段混合。

原“producer/consumer 字段类型、fixture 和非 ready DTO 不兼容”的触发路径已消除。

### AM-DES-R04-N002 — closed，无回归

- `completed_at` 继续排除于 canonical 和 `input_digest`，只在首份 Evidence 创建时生成。
- 输入不变时复用同一 `evidence_ref/evidence_digest/completed_at`；输入变化才追加 Evidence 并原子更新 current。
- Evidence 唯一约束、run 行锁、连续/并发/响应丢失和崩溃续跑验证合同保持。

### AM-DES-R04-N003 — closed，无回归

- `DeletionReplayRun` 唯一 `(run_id,namespace)`，`backup_ref` 首次绑定后不可变。
- 原子 `INSERT ... ON CONFLICT` 加行锁保证并发双 backup 只能一方绑定，失败方只返回 Unavailable。
- 相同 backup 并发继续收敛到同一 Evidence。

## 4. 新 findings

未发现新增 CRITICAL、HIGH、MEDIUM 或 LOW finding。

## 5. 历史业务回归

- r01 F001-F004 与 r02 N001 保持 closed。
- 8 条主责 US 均保留 Story 四块和需求分配表。
- S1-S16 均有明确落点；S16 所有权继续归 article，identity 只消费 tagged-union DTO。
- 永久删除继续要求 `deletion_pending` 隐藏、外部 tombstone 与物理 DELETE 双完成；旧备份开放前 ReplayAll；无文章恢复入口。
- canonical 删除继续覆盖零文章、多文章、违规文章和 `kept_deleted`，并保持 retained_email 清理顺序。
- 通知继续使用两 recipient 五态、短事务 claim/lease、ProcessingGate、强制 TLS 和 at-least-once 残余风险。
- 分页继续按 `(created_at,id)` 倒序，候选分批扫描至 `limit+1` 可见项或耗尽。
- 作者响应仍严格限制为 `title,body,moderation_status,editable`，ETag 仅在 Header；管理员完整治理投影和成员统一 404 保持分离。
- REST 接口总表仍为 16 个 endpoint。

## 6. 静态验证与限制

- design 共 2132 行，stories 共 222 行。
- design 有 27 个 PlantUML 块，`@start*`/`@end*` 均为 27；未发现字面 `\n` 或 `linetype ortho`。
- design 中 27 个本地 Markdown 文件链接均存在。
- stories 文本依赖图人工核对未发现环。
- Ready 20 字段和 Unavailable 6 字段名称/顺序已脚本比较，article 与 identity 完全一致。
- 两侧 canonical bytes 已逐字节比较，一致；独立 digest 复算通过。
- 本机无 `plantuml` 命令或 PlantUML JAR，未执行语法检查或 PNG 实渲染。
- 当前无业务代码、schema、迁移、API schema 或测试；项目未登记检查命令，未运行 build/lint/unit/integration/E2E。
- 本报告是静态设计复审，不替代真实 PostgreSQL 并发、Journal/inventory、崩溃续跑或恢复演练。
- architecture r12 顶部状态说明与 accepted checkpoint 不一致，需由 upstream owner 纠偏；不影响本次按精确 checkpoint/artifact 取证。

## 7. 最终结论

**最终结论：PASS。**

- `AM-DES-R04-N001`：closed
- `AM-DES-R04-N002`：closed，无回归
- `AM-DES-R04-N003`：closed，无回归
- 历史业务 findings：保持 closed
- 新 findings：无

该 PASS 仅表示固定 article r06 设计对象通过本次独立静态回归复审，不表示：

- 代码、schema、迁移、契约生成物、并发测试或真实恢复演练已经完成；
- PlantUML 已实渲染；
- AM-Q1/Q2、容量、留存和部署运行输入已经拍板；
- 用户/Lead 已完成成果接受、实施 Gate 或后续授权。

后续由 Lead 核对固定对象、identity r09 的独立复审结果、实际 Gate 与未决运行输入后决定下一步。
