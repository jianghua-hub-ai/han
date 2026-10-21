# article-management 详细设计原 finding 独立复审报告 r02

- 复审日期：2026-10-20
- objectType：design
- 原报告：`.spec/wip/article-management/review-design-r01.md`
- 原 findings：`AM-DES-R01-F001`～`AM-DES-R01-F004`
- 主对象：`article-management-design-ir01-r02`
- 附属对象：`article-management-stories-ir01-r02`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories、原报告或上游。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `9233f797b26492c9a7aac77872fa67e78fa491422c8e772d58a5bb11c9afdb78` | `9233f797b26492c9a7aac77872fa67e78fa491422c8e772d58a5bb11c9afdb78` | 一致 |
| `stories.md` | `73889f245a08a0b4b9331df6d33d94ecf1ce2face27ec139a0888652f4c3b4a2` | `73889f245a08a0b4b9331df6d33d94ecf1ce2face27ec139a0888652f4c3b4a2` | 一致 |

按 `## 确认记录` 排除确认段并规范化尾部为单个 LF 后，artifact digest 为：

`sha256:c54867bb418fb627494cc573e68a063d2d7a85f8b71df0a6bdc6a28601e23516`

与 Lead 提供值一致，无对象漂移。

### 1.2 上游快照

| 依据 | 修订/状态 | 完整 SHA-256 |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r09` accepted | `f69dd966f36f4d268a775657ddfc46e1b076c64098fd998e254fed3cfa886072` |
| identity boundary | `identity-auth-boundary-ir01-r08` accepted | `c9ebf1660b06252d33db588f65133ef57e05ffaa5ede95b996c314e55ca12a54` |
| article boundary | `article-management-boundary-ir01-r08` accepted | `ac96e8dea88592c8fc7210b1cce2d810cabf8f9b72756046d6bcc7538972388d` |
| content ADR | 含 ADR-0020 | `edacccadb48c76e20addf684cf09ed372182b10efb3b4d734337cdc34acdc3e5` |
| 原报告 | r01 | `94dd46f29d935c7cac5a5e8c01fe7d13e5d68424f7a497e5b0bac4a701f84729` |
| identity review baseline | r05 PASS | `0b870ebb8cabd86cc887ff03f445831ccfc27bacdb1179171b503ee8c3209de7` |

architecture r09 checkpoint：
`decision-203d6b9bf6474bb8b07a04887f65549f306b4072e62e7b5657cef50174f94b39`。

## 2. 两轮复审范围

### Round 1：原 finding 定点复审

重新执行：

- F001：备份后永久删除、恢复旧备份、开放前删除重放；
- F002：active 作者修改自己的 violating 文章且保持隐藏；
- F003：隐藏候选密集时 cursor 分批扫描不漏不重；
- F004：recipient claim、lease、重试、终态和 accepted 后崩溃。

### Round 2：整改回归

沿 8 条 US、S1～S16、17 个当前 REST endpoint、Article/TransferNoticeRecipient/DeletionOperation、两个 Journal namespace、identity ports、恢复开放门禁和 stories 依赖检查新增阻断风险。

## 3. 原 findings 逐项状态

### AM-DES-R01-F001 — closed

- 修复证据：`design.md:233-244`、`796-823`、`995-1105`、`1417-1465`；ADR-0020；boundary r08 S11/S16。
- 删除协议：
  1. T1 原子写 `deletion_pending` 和 DeletionOperation，文章立即对所有消费者不可见；
  2. 业务数据库外 `article-deletion/v1` tombstone 原子追加并完成 file/directory fsync；
  3. finalizer 幂等物理 DELETE，并同事务写 AuditEvent 和 operation completed；
  4. tombstone 与物理删除双完成后才返回 204。
- 旧备份恢复：readiness 开放前 `ReplayAll` 全量验证并重放 tombstone；operation 缺失时可由最小 tombstone 重建，Article 版本异常或任何对账失败均停服。
- tombstone 不保存 title、body、内容摘要、邮箱或其他 PII；对应 key 保留到 backup inventory 证明所有更早可恢复备份已退役。
- 原“旧备份复活永久删除文章”的路径已被 accepted 架构和可实施合同关闭。

### AM-DES-R01-F002 — closed

- 修复证据：`design.md:583`、`691`、`935-993`；`stories.md:67-87`。
- active 作者可通过受控自有详情取得自己仍有归属、deletion active 的普通或 violating 文章和强 ETag。
- PUT 只要求 author_id 与 Principal 精确匹配；修改仅改变 title/body/updated_at/version。
- violating/category、publication、source、author 和 deletion 状态保持不变；成员公开详情仍返回 404。
- 非作者和管理员非作者继续拒绝。
- 回归发现：作者详情额外披露治理原因，作为新 finding `AM-DES-R02-N001` 报告，不重开原修改能力 finding。

### AM-DES-R01-F003 — closed

- 修复证据：`design.md:201-221`、`626-645`、`880-933`；`stories.md:44-65`。
- 成员列表按配置化候选批次扫描：
  - 内部 `scan_after` 以最后扫描候选推进；
  - 每批批量读取作者状态并保持排序过滤；
  - 持续扫描到 `limit+1` 个可见项或候选耗尽；
  - 公共 cursor 以最后返回的可见项生成；
  - 第 `limit+1` 个可见项不消费，下页重新扫描并返回。
- 静态集合、同时间戳及长隐藏区间的不漏不重语义已固定；并发变化明确不承诺跨请求快照。

### AM-DES-R01-F004 — closed

- 修复证据：`design.md:766-794`、`1189-1274`、`1467-1478`；`stories.md:156-180`。
- 每个收件人独立保存：
  - `pending/retry_wait/processing/accepted/failed_terminal`；
  - `claimed_by/locked_until/next_attempt_at/attempt_count/max_attempts`。
- claim 和结果写入均为短事务，SMTP 调用期间不持有数据库事务；原子 claim、lease 回收和 owner 条件更新阻止并发覆盖。
- Gate 关闭时不 claim、不增加 attempt、不连接 SMTP；瞬时/永久错误和最大尝试均有确定转换。
- SMTP accepted 后结果提交前崩溃的重复投递被明确为 at-least-once 残余风险，并要求审计/告警。

## 4. 新 findings

### [MEDIUM] AM-DES-R02-N001：作者受控详情提前披露了仍待拍板的治理原因

- 位置：`design.md:89-100`、`583`、`935-993`；`stories.md:67-78`
- 违反依据：design 自身 AM-Q4 明确“隐藏原因是否向作者或普通成员展示”仍待需求/隐私拍板；已接受 requirements r16 只要求作者能修改自己的文章，没有授权向作者披露违规类别或全部 hidden reasons。
- 现状：新增 `GET /api/v1/users/me/articles/{id}` 向作者返回 author detail，并明确可包含 `violation_state/category` 和 `hidden_reasons`。
- 触发：作者读取一篇被管理员判定 violating，或因其他治理原因隐藏的自有文章。
- 影响：在 AM-Q4 未决定前新增消费者可见的治理信息，可能披露管理员判定类别及其他账号治理状态；接口与待拍板表互相冲突。
- 最小修复：保留作者读取内容和 ETag 以支持 F002，但在 AM-Q4 拍板前只返回通用 `restricted/editable` 语义，不返回 `violation_category` 或完整 hidden reasons；或先把披露范围回 requirements/隐私确认后再固定 API。
- 验证：作者可读取并修改 violating 自有文章，但未获授权的治理类别/隐藏原因不出现；管理员详情仍获得治理所需机器状态；普通成员继续统一 404。

## 5. 待决定与限制

| 项 | 处理 |
|---|---|
| AM-Q1 标题/正文格式与长度 | 仍阻塞 validator 和最终渲染实现 |
| AM-Q2 分页/候选批次数值 | 机制已定；阻塞容量验收 |
| AM-Q3 删除二次确认 | 只影响 UX，API 不依赖 |
| AM-Q4 作者隐藏原因披露 | 当前新 finding 的决定来源；拍板前不得扩公开字段 |
| AM-Q5 人工重发/通知查询 | 当前排除 |
| AM-Q6/Q7 留存与处理输入 | 部署/隐私前必填 |
| AM-Q8 tombstone/inventory/finalizer/key 运行参数 | 机制已定，灾备演练前必填 |
| 创建超时重复 | S9/ADR-0015 仍不扩展文章 Idempotency-Key；该残余风险未因本轮整改变化 |

## 6. 回归与静态验证

- 8 条主责 US 映射完整。
- design 与两份 boundary r08 的 S1～S16 合同一致。
- identity ports、canonical 删除、retained_email 顺序和两个 Journal namespace 未发现回归。
- Article Deletion Journal 不保存内容/PII，AuditEvent 仍为最小证据。
- 强 If-Match 拒绝 weak、多值、`*` 和非法语法。
- 成功 JSON 明确 `application/json; charset=utf-8` 与 `nosniff`，正文不按 HTML 执行。
- design 26 个 PlantUML 块静态起止成对，未发现字面 `\n` 或 `linetype ortho`。
- stories 使用文本依赖图；人工核对未发现环或正文依赖漂移。
- design 中 17 个本地链接均存在。
- 本机无可用 PlantUML CLI/JAR，未执行 PNG 实渲染。
- 当前无代码、schema、迁移或测试；未运行不存在的构建、lint、单元、集成或 E2E。

## 7. 最终结论与后续条件

**最终结论：NOT PASS。**

- 原 findings：F001～F004 全部 closed。
- 新增：1 个 MEDIUM finding `AM-DES-R02-N001`。
- 未发现新增 CRITICAL/HIGH。

在 N001 关闭前，design 不应转为 `implementable`。复审需提供：

1. 新 design/stories 修订、完整 SHA-256 和规范 artifact digest；
2. AM-Q4 的真实决定，或收敛后的作者详情最小披露合同；
3. 作者修改 violating 文章、管理员详情和普通成员 404 的契约回归；
4. 若 requirements/boundary/ADR 发生修订，附 accepted checkpoint 及 S1～S16 回归。

本结论只针对固定 r02 对象，不代替用户/Lead 的成果接受、风险决定或实施授权。
