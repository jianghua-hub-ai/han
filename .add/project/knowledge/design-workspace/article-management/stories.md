---
feature_id: article-management
artifact: design-stories
status: draft
revision: article-management-stories-ir01-r06
created: 2026-10-20
last_updated: 2026-10-20
source_design: design.md
source_design_revision: article-management-design-ir01-r06
---

# article-management 设计 Story 切片

本文件只列 8 个与已确认用户故事一一对应的设计切片，供后续 task-decompose 使用。它不创建 task、批次、认领或完成状态；永久设计合同以 [design.md](design.md) 为准。

## 共享依赖

- 所有 Story 依赖 [共享基础](design.md#331-共享基础)：Principal、Session/CSRF、公共错误、集中 CursorCodec、强 ETag/version、VisibilityPolicy、AuditStore。
- 所有写 Story 依赖 PostgreSQL 事务和 Article 检查约束。
- 管理 Story 依赖 identity-auth 的管理员 Principal；转移依赖目标资格、墓碑联系和 retained_email 清理端口。
- 所有永久删除 Story 依赖 ADR-0020 的 `deletion_pending`、DeletionOperation、独立 `article-deletion/*` tombstone、finalizer 与旧备份重放。
- S16 provider 依赖唯一 20 字段 v1 fixture；canonical 固定前 16 字段，identity 只消费完整 DTO并重算摘要。
- 恢复期间 readiness 关闭；S1–S16 任一门禁失败都不得开放，任何 Story 都不提供文章恢复。

## Story 1：创建文章（US-ARTICLE-01.1）

**切片目标**：已认证 active 用户提交标题和正文后直接得到 published/clear 的文章。

**验收**

1. POST 只从 Principal 取得作者，不接受客户端 author_id。
2. 标题或正文缺失/全空白/违反待拍板规则时返回 422，数据库无新行。
3. 成功返回 201、Location、ETag；不存在草稿或下线状态。
4. 数据库失败不得显示创建成功；日志不含标题和正文。

**依赖**

- 身份 Session/CSRF 接入。
- 标题/正文格式与长度 AM-Q1 在实现校验器前拍板。

**设计引用**

- [§3.3.2 创建文章](design.md#332-story创建文章us-article-011)
- [Article 数据结构](design.md#数据结构设计)

## Story 2：发现并阅读文章（US-ARTICLE-02.1）

**切片目标**：已认证用户通过稳定倒序 cursor 浏览成员可见文章，并读取详情。

**验收**

1. 列表按 `created_at,id` 倒序；候选分批扫描，批量读取作者状态并持续过滤到 `limit+1` 可见项或耗尽。
2. 列表不返回 body；详情返回标题和正文。
3. Cursor 基于最后返回可见项；静态混合状态集合中可见文章不漏不重，同时间戳稳定。
4. 不存在、deletion pending、永久删除或任一隐藏约束成立时统一返回 404，不泄露原因或内容。
5. 浏览无写副作用；identity 作者状态缺失时 fail closed。

**依赖**

- VisibilityPolicy 与 identity `GetAuthorStates`。
- 分页默认/上限和容量 AM-Q2。

**设计引用**

- [§3.3.3 发现并阅读](design.md#333-story发现并阅读文章us-article-021)
- [分页选择](design.md#241-分页选择)
- [正交状态矩阵](design.md#341-正交状态与有效可见性)

## Story 3：修改现有文章（US-ARTICLE-03.1）

**切片目标**：作者基于受控自有详情的当前 ETag，完整替换自己仍有归属、未进入永久删除协议的文章内容，包括 violating 文章。

**验收**

1. `GET /api/v1/users/me/articles/{id}` 只向当前作者返回 `title,body,moderation_status,editable` 和强 ETag；违规文章为 `restricted/true`，deletion pending 不返回。
2. PUT 要求单个强 If-Match；拒绝 weak、多值和 `*`，缺失 428，不匹配 412。
3. 只有作者可修改；普通非作者和管理员非作者继续拒绝。
4. 修改只改变 title/body/updated_at/version，不改变归属、publication、violation/category、source 或 deletion 状态。
5. 作者响应不得出现 `violation_state`、`violation_category`、`hidden_reasons` 或账号治理状态；管理员治理详情保持完整。
6. violating 文章修改后仍仅管理员和作者受控详情可见，成员详情继续 404。

**依赖**

- ArticleStatePolicy、行锁、ETag middleware。
- Story 2 提供当前资源和 ETag。

**设计引用**

- [§3.3.4 修改文章](design.md#334-story修改现有文章us-article-031)
- [并发更新选择](design.md#242-并发更新选择)

## Story 4：作者删除文章（US-ARTICLE-04.1）

**切片目标**：作者永久删除自己的普通文章，不留下可恢复正文。

**验收**

1. DELETE 要求作者身份和单个强 If-Match。
2. T1 将 Article 置 deletion_pending 并创建 DeletionOperation，立即对所有业务查询隐藏。
3. `article-deletion/*` tombstone 原子追加并 fsync 后，finalizer 物理 DELETE、AuditEvent 和 operation 完成同事务。
4. 只有 tombstone 与物理删除双完成返回 204；Journal 成功 DB 失败时 503 且继续收敛。
5. 旧备份开放前全量重放 tombstone；Journal 不保存内容、摘要、邮箱或其他 PII。
6. 无软删除正文、版本历史或恢复入口。
7. Ready variant 固定完整 20 字段、全部非 null，`inventory_scope` 为 string；canonical 仅为前 16 字段，fixture digest 固定 `169e0f...926a7`。
8. DeletionReplayRun 唯一 `(run_id,namespace)`，backup 首绑不可变；并发双 backup 只能一方成功，另一方稳定 failed。
9. stable input_digest 不变时复用同一 evidence_ref/evidence_digest/completed_at；covered input 变化才追加 evidence 并更新 current。
10. Unavailable variant 仅含 version/run/backup/namespace/status/reason_code，status 限 pending/unknown/failed，禁止 canonical/digest/ref/time。
11. ReplayAndVerify/ReadStatus 每次只返回一个 variant；共享 fixture 与负向 variant 测试必须通过。

**依赖**

- Article Deletion Journal、完整 keyring、backup inventory、DeletionReplayRun/EvidenceStore、共享 v1 fixture 与 AuditStore。
- UI 二次确认 AM-Q3 不影响 API 安全。

**设计引用**

- [§3.3.5 作者删除](design.md#335-story作者删除现有文章us-article-041)
- [永久删除选择](design.md#243-永久删除选择)
- [S16 replay evidence](design.md#347-deletionreplayevidence-与恢复开放门禁)

## Story 5：管理员删除他人普通文章（US-IDENTITY-03.2）

**切片目标**：管理员可永久删除他人普通文章，但不能修改他人文章或绕过专用治理路径。

**验收**

1. 管理员使用普通 DELETE 路径并提供 If-Match。
2. 普通文章必须 deletion active、有作者、published、clear；违规或无归属返回 invalid_state。
3. 管理员调用 PUT 修改他人文章必须拒绝。
4. 删除复用 pending/tombstone/finalizer；双完成才 204，最小审计不保存正文。

**依赖**

- identity 管理员 Principal。
- Story 4 的永久删除基础。

**设计引用**

- [§3.3.6 管理员删除](design.md#336-story管理员删除他人普通文章us-identity-032)
- [权限与 endpoint 总表](design.md#接口设计)

## Story 6：处置无归属文章（US-IDENTITY-03.8）

**切片目标**：管理员对待处置无归属文章执行匿名公开、转移或永久删除，并维护 retained_email 清理条件。

**验收**

1. 三种写操作只接受 `author_id=NULL` 且 `orphaned_from_user_id` 非空的文章。
2. 违规文章不能直接恢复公开；转移后继续隐藏且违规事实不变。
3. 目标用户必须 active；仍有归属的文章拒绝转移。
4. 公开/转移事务完成后退出集合；无归属删除由 finalizer 物理删除后统计并清理 retained_email。
5. 零文章、多文章、违规文章、deletion pending 和 kept_deleted 重放矩阵全部可验证。

**依赖**

- identity 转移目标/墓碑联系与清理端口。
- AuditStore、TransferNotice/Recipient、canonical 删除端口和 Article Deletion Journal。

**设计引用**

- [§3.3.7 无归属处置](design.md#337-story处置无归属文章us-identity-038)
- [canonical 删除端口](design.md#343-canonical-删除端口)
- [地址清除顺序](design.md#344-转移与通知地址清除顺序)

## Story 7：发送文章转移通知（US-IDENTITY-03.9）

**切片目标**：已提交转移通过持久通知记录 best effort 告知新作者和已删除原作者。

**验收**

1. 两路地址在转移事务中写入两个独立 pending recipient，公共响应和日志不披露邮箱。
2. 通知 purpose 固定为事务性转移通知，禁止营销。
3. ProcessingGate 缺实际输入时不 claim、不增加 attempt_count、不连接 SMTP，转移不回滚。
4. 每收件人状态固定为 `pending/retry_wait/processing/accepted/failed_terminal`，含 claim owner、lease、next attempt、attempt/max。
5. claim 与结果保存使用短事务，SMTP 调用期间不持有数据库事务；双 worker 和 lease 到期可恢复。
6. 瞬时错误重试，永久错误或最大尝试进入 failed_terminal；两个收件人独立。
7. SMTP accepted 后崩溃可能重复投递，作为 at-least-once 残余风险审计。
8. 生产 SMTP 只允许隐式 TLS 或强制 STARTTLS，并校验证书链与主机名；首期无人工重发 endpoint。

**依赖**

- Story 6 的转移事务和快照。
- ProcessingGate、Mailer、SMTP TLS 合同。

**设计引用**

- [§3.3.8 转移通知](design.md#338-story发送文章转移通知us-identity-039)
- [TransferNotice 数据结构](design.md#数据结构设计)
- [隐私分析](design.md#54-隐私风险分析)

## Story 8：违规文章治理（US-IDENTITY-03.12）

**切片目标**：管理员判定暴力/色情违规、解除违规或永久删除，且不覆盖其他隐藏约束。

**验收**

1. 判定只允许 category `violence|sexual`，写 violating/category/version。
2. 成员在判定提交后立即得到统一 404；管理员仍可查看治理详情。
3. restore 只清除 violation/category；author_suspended 或 orphan_hold 仍保持隐藏。
4. 违规删除复用 pending/tombstone/finalizer，双完成才 204；finalizer 物理删除与 AuditEvent 同事务。
5. 重复判定、clear 状态 restore 和错误删除入口返回 invalid_state；并发冲突返回 412。

**依赖**

- identity 管理员 Principal。
- VisibilityPolicy、AuditStore、ETag/version 和 Article Deletion Journal/finalizer。

**设计引用**

- [§3.3.9 违规治理](design.md#339-story隐藏解除判定与删除违规文章us-identity-0312)
- [正交状态矩阵](design.md#341-正交状态与有效可见性)

## Story 依赖顺序

```text
Story 1 创建 ─┬─> Story 2 浏览 ─> Story 3 修改
              └─> Story 4 作者删除 ─> Story 5 管理员删除
共享正交状态 ─> Story 8 违规治理
canonical 删除 ─> Story 6 无归属处置 ─> Story 7 转移通知
```

该顺序只表达设计依赖，不是实施批次。task-planner 应结合测试、迁移和共享基础再拆任务。
