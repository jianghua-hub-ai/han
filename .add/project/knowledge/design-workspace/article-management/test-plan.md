---
feature_id: article-management
artifact: test-plan
status: planned-not-executed
revision: article-management-test-plan-ir01-r01
created: 2026-10-20
last_updated: 2026-10-20
source_design_revision: article-management-design-ir01-r06
source_design_artifact: sha256:ed1a13b7f3d2e9f2c63b47d66d26469ea19315c42ace9cbe49db9f206880e4e0
requirements_checkpoint: decision-e3af3912166f7287e8324488cad1cf1b87ba1034fca3378d3e0467a23751592d
architecture_checkpoint: decision-0347e1026fd9148ad16f66750023cb50445a1d7c133db6d82669ea14865b318f
design_checkpoint: decision-b1f45c78289c05b46e3304495e6a5f5112bf47357615df6fce5650f63cf869f8
---

# article-management 风险驱动测试规划

本规划状态为 **planned / not executed**。它描述稳定设计切片需要哪些证据，不表示文章代码、数据库、Journal、SMTP、恢复环境、性能基准或 E2E 已执行或通过。

## 1. 对象、范围与来源

测试对象是 `article-management-design-ir01-r06` 及其 8 条主责 Story。设计已通过独立静态评审，但实施 Gate 按 `decision-b1f45c78289c05b46e3304495e6a5f5112bf47357615df6fce5650f63cf869f8` 处于 `deferred:reorder`。

| 来源 | 本规划使用的事实 |
|---|---|
| [requirements r16](../../requirements/requirements.md) | 4 条文章 US 与 4 条文章治理 US 的验收 |
| [architecture r12](../../design/architecture.md) | REST/JSON、PostgreSQL、模块边界、恢复与 readiness |
| [article boundary r12](../../design/article-management/boundary.md) | article 所有权、S1–S16、永久删除和通知边界 |
| [identity boundary r11](../../design/identity-auth/boundary.md) | Principal、作者状态、canonical 删除和恢复消费者合同 |
| [article design r06](design.md) / [stories r06](stories.md) | 16 个 REST endpoint、状态矩阵、ETag、cursor、通知、Journal 和 ReplayStatus producer |
| [identity design r09](../identity-auth/design.md) / [identity test plan](../identity-auth/test-plan.md) | 身份、停用/恢复、用户删除、Recovery Journal 与 ReplayStatus consumer |
| [article review r06](../../../../../.spec/wip/article-management/review-design-r06.md) | PASS 范围与未执行的 PostgreSQL、并发、Journal、恢复和通知验证 |
| [ADR 索引](../../decisions/README.md) | ADR-0001～ADR-0020 |
| [项目 Manifest](../../../../../add.manifest.yaml) | `go-service` 已启用；无项目检查命令 |

### 1.1 成功证据

1. 8 条主责 US 均映射到风险、场景、层次和证据；
2. 可见性状态组合、ETag、cursor、notification lease、Article Deletion Journal、ReplayStatus producer 和恢复重放都有正负向与故障场景；
3. 跨特性场景只在一个计划主责，另一计划引用；
4. 专项集成/安全/性能和真实 E2E 与开发局部测试分开；
5. 未决格式、容量、留存和部署参数保持 TBD；
6. test-plan 建议已由 task-planner 登记并在 §6、§7、§9、§11 回填实际 task 引用；任务状态仍只在 tasks SSOT。

## 2. 验收—风险地图

| 风险 | 优先级 | 影响 | 可能性依据 | 主要场景 |
|---|---|---|---|---|
| R-A01 可见性组合错误或治理信息泄露 | P0 | 违规、停用、无归属内容被普通成员看到 | publication、violation、author、author status、deletion 五维组合 | A-READ-01～06、A-MOD-01～04 |
| R-A02 永久删除不完整或旧备份复活 | P0 | 已删内容恢复、数据不可逆承诺失效 | PostgreSQL 与外部 Journal 跨介质、崩溃和备份恢复 | A-DEL-01～10、A-REC-01～08 |
| R-A03 ETag/并发覆盖 | P0 | 作者内容或治理事实被静默覆盖 | 编辑、治理、转移、删除共享 version | A-ETAG-01～04 |
| R-A04 canonical 删除跨模块部分成功 | P0 | 删除用户后文章仍有归属或错误公开 | identity 与 article 同库事务、零/多/违规/重放矩阵 | A-ORP-01～07 |
| R-A05 通知地址、用途或 TLS 越界 | P0 | PII 泄露、未经门禁发送、营销滥用 | 两收件人、地址快照、SMTP、lease 和处理方门禁 | A-NOT-01～09 |
| R-A06 notification lease/重试并发错误 | P0 | 重复投递、丢通知或 worker 互相覆盖 | 双 worker、租约过期、accepted 后崩溃 | A-NOT-03～08 |
| R-A07 cursor 漏重或隐藏密集区跳项 | P1 | 用户无法稳定浏览可见文章 | 分批扫描、作者状态批量读取、并发新增 | A-READ-01～04 |
| R-A08 状态/权限入口混用 | P0 | 管理员修改他人、错误入口删除违规/无归属文章 | 作者、成员、管理员三种投影和多条删除路径 | A-CREATE-01～03、A-MOD-01～04、A-MODR-01～06 |
| R-A09 审计、错误或 Journal 含正文/PII | P0 | 不可逆数据重新可恢复或二次泄露 | 删除、通知和恢复均产生内部证据 | A-CON-01～04、A-DEL-06、A-NOT-09 |
| R-A10 ReplayStatus schema/digest 漂移 | P0 | identity 错误开放 readiness | Ready20/Unavailable6 是封闭内部 schema | A-REC-01～08 |
| R-A11 内容输出执行 HTML/XSS | P1 | 客户端脚本执行与内容泄露 | 正文格式未拍板，服务端只承诺文本 JSON | A-CREATE-03、A-READ-06 |
| R-A12 容量与积压 | P1 | 列表退化、pending/通知/finalizer/replay 积压 | 文章规模、隐藏密度、并发和保留值均 TBD | A-PERF-01～05 |

### 2.1 8 条 US 的关键验收与证据

| US | 关键可观察验收 | 至少一个计划证据 | 场景 |
|---|---|---|---|
| US-ARTICLE-01.1 创建文章 | active Principal 创建 published/clear 文章；无草稿 | HTTP 201/Location/ETag、Article 行、无 draft/offline 枚举 | A-CREATE-01～03 |
| US-ARTICLE-02.1 发现并阅读 | 倒序 cursor、当前可见性、不漏不重、受限统一 404 | 分页返回序列、作者状态批量读取、详情响应 | A-READ-01～06 |
| US-ARTICLE-03.1 修改文章 | 仅作者，强 If-Match，只改内容/version，违规事实保留 | 428/412/成功响应、前后 Article 快照、作者字段 allowlist | A-ETAG-01～04、A-MOD-01～04 |
| US-ARTICLE-04.1 删除文章 | tombstone 与物理 DELETE 双完成才 204，不可恢复 | DeletionOperation、Journal 记录、Article 缺失、AuditEvent | A-DEL-01～10 |
| US-IDENTITY-03.2 删除他人文章 | 管理员只删普通文章且不能修改他人 | 权限/状态矩阵响应、删除协议证据 | A-MODR-01～02 |
| US-IDENTITY-03.8 处置无归属文章 | 匿名公开、转移或永久删除；违规/目标资格规则正确 | Article/Notice/retained_email 前后事务快照 | A-ORP-01～07 |
| US-IDENTITY-03.9 转移通知 | 两收件人独立 best effort；失败不回滚转移 | Recipient 状态、SMTP 测试记录、Article 归属保持 | A-NOT-01～09 |
| US-IDENTITY-03.12 违规治理 | 判定、解除、删除只改变各自事实，其他隐藏约束保留 | Article 状态前后、成员 404、管理员详情、Audit | A-MODR-03～06 |

## 3. 测试分层

| 层次 | 证明内容 | 本特性重点 |
|---|---|---|
| 单元/组件 | 局部规则和纯函数 | validator、VisibilityPolicy、AuthorProjectionPolicy、状态转换、CursorCodec、强 ETag parser、canonical digest、通知错误分类 |
| 契约 | REST 与内部生产者/消费者一致 | 16 个 REST endpoint、公共错误、S1–S16、Ready20/Unavailable6、字段 allowlist |
| 集成 | 真实 PostgreSQL、SMTP 和持久卷边界 | 事务/约束/行锁、cursor 扫描、canonical 删除、通知 lease、Journal/finalizer/evidence |
| 安全/韧性 | 权限、隐私、TLS、崩溃与恢复 | 统一 404、XSS 输出、TLS 证书、双 worker、fsync、链损坏、旧备份恢复 |
| 性能 | 容量和积压维度 | 分页隐藏密度、正文大小、dispatcher/finalizer、replay；目标值 TBD |
| E2E | 少量真实业务旅程 | 创建浏览修改删除、停用/恢复可见性、删除用户后处置、旧备份重放；后续独立任务 |

## 4. 计划场景

### 4.1 创建、读取、cursor 与输出安全

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-CREATE-01 | P0 | active 已认证用户提交合法 title/body | 201、Location、强 ETag；Article 为 published/clear/active，author 只取 Principal | REST + PostgreSQL 集成；自动 |
| A-CREATE-02 | P0 | 未登录、受限 onboarding、suspended/recovery 状态、缺 CSRF | 拒绝且无 Article；请求 body 的 author_id 不被接受 | 权限/接入契约；自动 |
| A-CREATE-03 | P1 | 缺失、全空白、未知/重复字段、大于未来上限、包含 HTML 字面文本 | 422 或配置约束结果；数据库无行；正文作为 JSON 文本，不执行 HTML；日志无内容 | validator/契约；长度目标受 AM-Q1 阻塞 |
| A-READ-01 | P1 | 静态混合 published/violating/held/suspended/deletion_pending 集合分页 | 只返回成员可见项；按 `(created_at,id)` 倒序；达到 limit+1 或耗尽 | PostgreSQL 分页集成；自动 |
| A-READ-02 | P1 | 同时间戳、多批连续隐藏区、空结果、末页 | ID 打破平局；可见项不漏不重；不会空页后跳过稳定可见项 | 集成 + 属性测试建议；自动 |
| A-READ-03 | P1 | 遍历期间新增新文章、修改内容、作者状态变化 | 新增只在重开首页出现；修改不改顺序；状态按每请求当前事实计算 | 并发集成；自动 |
| A-READ-04 | P1 | cursor 缺字段、未知版本、错误 view、时间/UUID 越界、篡改 | `request.invalid_cursor`；无数据或授权泄露 | CursorCodec 单元/fuzz + HTTP 契约；自动 |
| A-READ-05 | P0 | identity 作者状态缺失/故障 | fail closed；文章不作为 active 作者内容返回 | 内部端口故障集成；自动 |
| A-READ-06 | P0 | 不存在、violating、held、author suspended、deletion pending、永久删除的成员详情 | 统一 404 `article.unavailable`，无标题/正文/隐藏原因泄露 | REST 契约；自动 |

### 4.2 强 ETag、作者修改与字段隔离

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-ETAG-01 | P0 | 缺 If-Match、weak、多值、`*`、格式错误 | 缺失 428；其他稳定拒绝；Article 不变 | parser 单元 + HTTP 契约；自动 |
| A-ETAG-02 | P0 | 两个 writer 使用同一 ETag 修改或治理 | 只有一个成功；另一方 412；不发生最后写入者覆盖 | PostgreSQL 行锁并发；自动 |
| A-ETAG-03 | P0 | 读取新 ETag 后重试合法动作 | 只对当前 version 成功，version 单调递增 | 集成；自动 |
| A-ETAG-04 | P0 | ETag 与数据库 version、响应 Header 和内部 DTO 对账 | 强 ETag 稳定且不依赖时钟；body 不泄露内部 version | 契约；自动 |
| A-MOD-01 | P0 | 作者读取普通与 violating 自有文章 | 返回 `title,body,moderation_status,editable`，ETag 在 Header；无类别/原因/账号状态 | 字段 allowlist 契约；自动 |
| A-MOD-02 | P0 | 作者修改普通/violating 文章 | 只改 title/body/updated_at/version；违规、归属、publication、source、deletion 不变 | PostgreSQL 前后快照；自动 |
| A-MOD-03 | P0 | 非作者普通用户和管理员非作者 PUT | 全部拒绝；管理员删除权不扩展为编辑权 | 权限矩阵；自动 |
| A-MOD-04 | P0 | violating 文章修改成功后从成员、作者、管理员三入口读取 | 成员仍 404；作者仍 restricted；管理员仍见完整类别/原因 | 多消费者契约集成；自动 |

### 4.3 管理员删除与违规治理

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-MODR-01 | P0 | 管理员删除他人普通文章 | 仅 active/published/clear/有作者路径进入永久删除；双完成才 204 | 权限/状态 + 删除集成；自动 |
| A-MODR-02 | P0 | 管理员用普通 DELETE 删除 violating/无归属，或非管理员删他人 | `article.invalid_state`/拒绝；必须使用专用治理路径 | REST 状态矩阵；自动 |
| A-MODR-03 | P0 | 管理员判定 category=violence/sexual、重复判定、非法 category | 合法写 violating/category/version；重复或非法稳定拒绝 | 状态机 + PostgreSQL；自动 |
| A-MODR-04 | P0 | 判定提交后成员、作者、管理员读取 | 成员立即 404；作者 restricted；管理员可见完整治理详情 | 多视图集成；自动 |
| A-MODR-05 | P0 | clear violation，分别叠加 active、author_suspended、orphan_hold | 只清 violation/category；其他约束继续隐藏 | 状态组合集成；自动 |
| A-MODR-06 | P0 | 管理员删除 violating 文章，Audit 失败或并发 version 变化 | 复用 Journal/finalizer；审计失败不完成物理事务；并发冲突 412 | 删除/审计故障注入；自动 |

### 4.4 Article Deletion Journal、finalizer 与不可恢复删除

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-DEL-01 | P0 | 作者删除自己的普通文章 | T1 写 deletion_pending + 唯一 DeletionOperation；提交后所有业务查询不可读 | PostgreSQL 集成；自动 |
| A-DEL-02 | P0 | Journal append + file/dir fsync 成功，finalizer 成功 | Article 物理 DELETE；Audit 与 operation completed 同事务；API 204 | PostgreSQL + 持久卷集成；自动 |
| A-DEL-03 | P0 | Journal append/锁/权限/空间/fsync/链/key 任一失败 | 不返回 204；Article 保持不可见 pending；不得恢复 active | 文件故障注入；自动 |
| A-DEL-04 | P0 | Journal 成功但记录 sequence 的 DB 更新失败，或 finalizer DB 失败 | tombstone 仍为权威；后台/授权重试继续收敛；无可见窗口 | 跨介质故障注入；自动 |
| A-DEL-05 | P0 | 同 Article 重复 DELETE、双 finalizer、lease 到期回收 | 不创建第二 operation/tombstone；最终仅一个完成结果 | PostgreSQL 并发 + Journal 对账；自动 |
| A-DEL-06 | P0 | 检查 tombstone、DeletionOperation、Audit、日志和错误 | tombstone 仅 allowlist；所有证据无 title/body/内容摘要/email/其他 PII | schema 契约 + 自动敏感扫描 |
| A-DEL-07 | P0 | 旧备份中 Article active/version≤tombstone、Article absent、version>tombstone | 前两类幂等删除/对账；ID 复用或高版本不一致保持停服 | 备份恢复集成；自动 |
| A-DEL-08 | P0 | 普通、violating、orphan 三类永久删除 | 都经相同 pending/tombstone/finalizer；delete_kind 和清理后果正确 | 参数化集成；自动 |
| A-DEL-09 | P0 | 物理删除完成后的所有成员/作者/admin/治理入口 | 全部 404/不可用；无恢复 endpoint 或内容副本 | REST 契约 + schema 检查；自动 |
| A-DEL-10 | P1 | tombstone/key 退役前检查 backup inventory | 早于 deleted_at 的可恢复备份未退役时禁止清理记录/key | 运行规则契约；自动结构检查 + 人工授权，留存值未定 |

### 4.5 无归属、canonical 删除与邮箱清理

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-ORP-01 | P0 | identity 删除有普通/违规文章的用户 | `author_id=NULL, publication=held, source=user`；violation 保留；每个新结果有 Audit | 跨模块 PostgreSQL；identity 计划端到端主责 |
| A-ORP-02 | P0 | canonical 删除零文章、已无归属、已匿名公开、已转移、已永久删除、deletion pending | 幂等 no-op/实际 pending 计数；不把已处置文章回退 held | 端口集成；自动 |
| A-ORP-03 | P0 | 管理员恢复 clear 无归属文章公开 | 变为 published、source 清除、author 仍 NULL；成员显示匿名 | PostgreSQL/REST 集成；自动 |
| A-ORP-04 | P0 | 直接恢复 violating 无归属文章 | 拒绝；仍 held+violating；必须先 clear | 状态机集成；自动 |
| A-ORP-05 | P0 | 转移给 active、suspended、deleted、缺失目标；转移仍有作者文章 | 仅 active 目标成功；其他拒绝且 Article 不变 | identity 端口 + 权限集成；自动 |
| A-ORP-06 | P0 | 转移 clear/violating 无归属文章 | 都写新 author/published/清 source；clear 对成员可见，violating 继续隐藏 | 状态组合集成；自动 |
| A-ORP-07 | P0 | 恢复、转移、删除逐步减少同墓碑集合；通知地址快照缺失/已留存 | 条件满足后才调用 identity 清理 retained_email；不足则保留；事务失败全回滚 | 跨模块集成；article 侧主验，identity 清理门禁引用 |

### 4.6 转移通知、lease、SMTP/TLS

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-NOT-01 | P0 | 成功转移 | 同事务创建 Notice 与两个独立 pending recipient 地址快照；响应/日志不披露邮箱 | PostgreSQL + 字段扫描；自动 |
| A-NOT-02 | P0 | ProcessingGate 缺地区/控制者/依据/SMTP 处理方任一输入 | 不 claim、不增加 attempt、不连接 SMTP；转移保持成功 | 门禁/SMTP 集成；自动 |
| A-NOT-03 | P0 | 双 worker 同时 claim 同 recipient | 只有一个 owner 获得有效 lease；SMTP 调用期间无 DB 长事务 | PostgreSQL 并发；自动 |
| A-NOT-04 | P0 | pending、retry_wait 到期、processing lease 到期、未到期 lease | 只有合法状态可 claim；attempt/owner/next time 转换正确 | 状态机 + 可控时钟集成；自动 |
| A-NOT-05 | P0 | SMTP accepted、瞬时错误、永久错误、达到 max_attempts | 分别 accepted/retry_wait/failed_terminal；两个 recipient 独立 | SMTP 故障分类集成；自动；次数值 TBD |
| A-NOT-06 | P0 | claim 后 Gate 关闭 | 释放为 retry_wait，不连接 SMTP；已 claim attempt 保守计数 | 竞态集成；自动 |
| A-NOT-07 | P0 | accepted 后进程在结果提交前崩溃 | lease 到期可重发；不回滚转移；重复投递风险有 trace/审计 | 崩溃注入；自动；明确 at-least-once |
| A-NOT-08 | P0 | 旧 worker 在 lease 被回收后提交结果 | 条件更新失败，不覆盖新 owner；产生低敏竞争告警 | 并发集成；自动 |
| A-NOT-09 | P0 | implicit TLS、required STARTTLS、明文/降级、过期/错主机证书、跳过验证 | 只允许前两种安全成功；其他 fail closed；purpose 固定事务性且无营销入口 | SMTP/TLS 安全专项；自动；不发外部真实邮件 |

### 4.7 REST、审计与 S1–S16 契约

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-CON-01 | P1 | 16 个 REST endpoint 的 method/path/content-type/schema/error | 与 design 接口表一致；响应含 trace_id/nosniff；未知/重复字段拒绝 | 契约；自动；当前无 schema |
| A-CON-02 | P0 | 成员、作者、管理员 serializer 交叉注入内部字段 | 三个投影字段集合隔离；正文、邮箱和内部删除字段不越界 | allowlist 契约；自动 |
| A-CON-03 | P0 | 高风险写 Audit INSERT 失败、应用尝试更新/删除 Audit | 业务回滚或 finalizer 不完成；应用无 UPDATE/DELETE 能力 | DB 权限/故障注入；自动 |
| A-CON-04 | P0 | S1–S16 内部端口和公共错误 fixture | Principal、作者状态、canonical 删除、清理、ProcessingGate、ReplayStatus 的方向/失败语义一致 | 内部 Go 契约；自动；当前无代码 |

### 4.8 ReplayStatus producer、证据与联合恢复

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| A-REC-01 | P0 | 生成固定 Ready fixture | 严格 20 字段、全部非 null、`inventory_scope` 为 string；canonical 前 16 字段摘要为 `169e0f…926a7` | producer 契约；自动 |
| A-REC-02 | P0 | Ready 缺失/额外/null/错误类型/换名/乱序输入，或添加 reason_code | serializer/parser 按封闭 schema 拒绝；不发布 ready | schema 负向契约；自动 |
| A-REC-03 | P0 | 生成 pending/unknown/failed Unavailable；附加 canonical/digest/ref/time/null/ready 状态 | 只允许精确 6 字段与三种状态；禁止伪 evidence | producer 契约；自动 |
| A-REC-04 | P0 | 同 run 同 backup 连续/并发/响应丢失重试且 input 不变 | 复用同一 evidence_ref/digest/completed_at；Evidence 行数不增加 | PostgreSQL 并发；自动 |
| A-REC-05 | P0 | 分别改变 Journal head、inventory、operation、audit 输入 | input_digest 变化；追加/复用唯一新 evidence，原子更新 current；旧 ready 失效 | 集成；自动 |
| A-REC-06 | P0 | 同 run 的 backup A/B 并发，及相同 backup 并发 | `(run_id,namespace)` 只有一行；首绑 backup 不变；失败方稳定 Unavailable failed | PostgreSQL 双事务；自动 |
| A-REC-07 | P0 | 崩溃于 run 插入后、replay 后、evidence/current 提交前、提交后未响应 | 续跑不产生第二 run、语义重复 evidence 或伪 ready | 崩溃恢复专项；自动 |
| A-REC-08 | P0 | 备份→普通/违规/无归属删除→恢复旧备份→ReplayAll→operation/audit 对账→双次 ReadStatus | 三类已删文章所有入口不可读；两个 namespace 独立；identity 二次同 ref/digest 后才可 MarkConsumed/open | 双模块真实恢复专项；联合开放主责在 identity 计划 |

### 4.9 性能与容量维度

| ID | 状态 | 测量维度 | 当前结论 |
|---|---|---|---|
| A-PERF-01 | blocked | 标题/正文大小、创建/修改延迟 | AM-Q1 与目标值未定 |
| A-PERF-02 | blocked | 文章规模、分页 limit/候选批次、隐藏扫描比、P95 | AM-Q2 与工作负载未定 |
| A-PERF-03 | blocked | dispatcher 吞吐、lease、SMTP 延迟、重试积压 | 并发、超时、max_attempts TBD |
| A-PERF-04 | blocked | deletion_pending 年龄、Journal fsync、finalizer 吞吐 | 容量和告警阈值 TBD |
| A-PERF-05 | blocked | tombstone inventory、ReplayAll、evidence 重算与 RTO | 备份规模、retention、RPO/RTO TBD |

## 5. 跨特性场景主责

| 跨特性场景 | 主责计划 | 本计划范围 |
|---|---|---|
| 停用/恢复作者后文章可见性 | [identity test plan](../identity-auth/test-plan.md) | 提供 VisibilityPolicy/查询断言，不重复建立端到端任务 |
| 删除用户→无归属/邮箱清理 | [identity test plan](../identity-auth/test-plan.md) | 验证 `OrphanArticlesCanonical`、待处置统计和清理完成事实 |
| canonical 删除与 kept_deleted | [identity test plan](../identity-auth/test-plan.md) | 验证 article 状态、违规保留、pending/finalizer 分支 |
| Article Deletion Journal 与永久删除 | 本计划 | identity 只引用恢复 Gate 结果 |
| 双 Journal、双 namespace、联合 readiness | [identity test plan](../identity-auth/test-plan.md) | 生产并验证 article namespace 和 Ready/Unavailable |
| ReplayStatus tagged union/fixture | 本计划 | identity 计划验证严格消费者和 MarkConsumed 顺序 |
| notification lease/SMTP | 本计划 | identity 只提供 ProcessingGate/联系/清理端口 |

### 5.1 S1–S16 合同责任

| 接缝 | 本计划验证 | identity 计划验证 |
|---|---|---|
| S1 同源与身份 | 文章 handler 只消费已确认 Principal；写请求缺 Session/CSRF 被接入层拒绝 | Cookie、Origin、Fetch Metadata、CSRF 和会话权威状态 |
| S2 作者状态与可见性 | A-READ-01～05 的批量作者状态与 VisibilityPolicy | `GetAuthorStates` 的身份侧只读输出 |
| S3 停用/恢复 | article 零写入，查询按当前作者状态派生 | 停用/恢复、Session 撤销与端到端主责 |
| S4 canonical 删除 | A-ORP-01～02 的 Article NULL/held/source/违规保留与计数 | User 墓碑、凭据/会话/标识与同事务编排 |
| S5 无归属处置/清理 | A-ORP-03～07 的三种处置、通知快照和完成事实 | 目标资格、墓碑联系和 retained_email 清理 |
| S6 邮件/TLS | A-NOT-01～09 的转移通知 best effort、TLS 和 lease | 凭据邮件 fail-closed 与共享 ProcessingGate |
| S7 分级治理 | A-MOD/A-MODR/A-ORP 的动作与目标矩阵 | Principal 角色、initial admin 与账号目标资格 |
| S8 公共错误 | article code、统一 404 与 REST envelope | 共享错误注册和 identity code |
| S9 幂等 | 文章写不接受/不扩展 Idempotency-Key；ETag 与 finalizer 幂等保持分离 | 三类身份写的作用域化键式幂等 |
| S10 审计 | 文章治理、转移、永久删除的 Audit 同事务及最小字段 | 身份、凭据、恢复和标识清理审计 |
| S11 Article Deletion Journal | A-DEL-01～10 主责 | 只消费恢复 Gate，不读取删除内部表 |
| S12 隐私/部署门禁 | A-NOT-02/09 的通知用途、地址快照和 Gate | 注册、管理员声明和凭据邮件门禁 |
| S13 恢复信任根 | readiness 关闭时全部文章 API 503，不改变身份状态 | 恢复信任根和账号 fail-closed 主责 |
| S14 kept_deleted/RPO | A-ORP-01～02 幂等重放 article 侧 canonical 结果 | 逐账号核对和完整删除协调 |
| S15 Recovery Journal | 只在恢复开放结果成立后接受业务流量，不读写 `recovery/*` | Recovery Journal/keyring 防重放主责 |
| S16 replay evidence | A-REC-01～08 的 producer、EvidenceStore、run/backup、fixture 和 replay | strict consumer、二次读取、MarkConsumed 和联合开放 |

## 6. 覆盖映射

场景范围表示该范围内每个场景 ID 都继承本行映射；场景的逐项前置、动作和预期见 §4。

| 场景 ID | US/SR | 设计章节/风险 | 层次 | 计划证据 | 任务建议 | 登记状态 |
|---|---|---|---|---|---|---|
| A-CREATE-01～03 | US-ARTICLE-01.1 / SR-ARTICLE-01 | design §3.3.2；R-A08/R-A11 | 单元、REST、PostgreSQL | 201/Location/ETag、Article 行、无草稿状态、内容不进日志 | `article-management-S1.3`、`article-management-S1.5` | 已登记 |
| A-READ-01～06 | US-ARTICLE-02.1 / SR-ARTICLE-02 | design §2.4.1、§3.3.3、§3.4.1；R-A01/R-A07 | 单元、契约、PostgreSQL 并发 | 可见序列、cursor、作者状态读取、统一 404 | `article-management-S2.1`、`article-management-S2.2` | 已登记 |
| A-ETAG-01～04 | US-ARTICLE-03.1、04.1；US-IDENTITY-03.2、03.8、03.12 | design §2.4.2、§3.3.1；R-A03 | 单元、HTTP、PostgreSQL 锁 | 428/412、单一成功 writer、version/ETag 对账 | `article-management-S1.2`、`article-management-S3.1`、`article-management-S2.2` | 已登记 |
| A-MOD-01～04 | US-ARTICLE-03.1 / SR-ARTICLE-03 | design §3.3.4、§3.4.1；R-A01/R-A08 | 权限、契约、PostgreSQL | 作者 allowlist、Article 前后快照、三视图结果 | `article-management-S3.1`、`article-management-S1.5`、`article-management-S2.2` | 已登记 |
| A-MODR-01～02 | US-IDENTITY-03.2 / SR-IDENTITY-03 | design §3.3.6；R-A03/R-A08 | 权限、状态、删除集成 | 普通文章入口、invalid_state、永久删除证据 | `article-management-S5.1`、`article-management-S4.1`、`article-management-S4.3` | 已登记 |
| A-MODR-03～06 | US-IDENTITY-03.12 / SR-IDENTITY-03 | design §3.3.9、§3.4.1；R-A01/R-A03/R-A08 | 状态机、权限、审计 | violation/category/version、三视图、Audit | `article-management-S8.1`、`article-management-S2.2`、`article-management-S4.3` | 已登记 |
| A-DEL-01～10 | US-ARTICLE-04.1 / SR-ARTICLE-04；删除类治理 US | design §2.4.3、§3.4.5/6；R-A02/R-A09 | PostgreSQL、持久卷、并发、恢复 | DeletionOperation、tombstone/fsync、物理 DELETE、Audit、inventory | `article-management-S4.1`、`article-management-S4.3` | 已登记 |
| A-ORP-01～02 | US-IDENTITY-03.7 / SR-IDENTITY-03；S4/S14 | design §3.4.3；R-A04 | 跨模块 PostgreSQL、恢复 | canonical 计数、Article 状态、违规保留、Audit | `article-management-S6.1`、`identity-auth-S9.2` | 已登记 |
| A-ORP-03～07 | US-IDENTITY-03.8 / SR-IDENTITY-03；S5 | design §3.3.7、§3.4.4；R-A01/R-A04/R-A05 | 状态机、跨模块事务 | 匿名公开/转移/删除、Notice、retained_email 门禁 | `article-management-S6.2`、`article-management-S7.1`、`article-management-S7.2`、`identity-auth-S9.2` | 已登记 |
| A-NOT-01～09 | US-IDENTITY-03.9 / SR-IDENTITY-03 | design §3.3.8、§3.4.8；R-A05/R-A06/R-A09 | PostgreSQL 并发、SMTP/TLS、崩溃 | Recipient 五态、claim/lease、SMTP 记录、低敏审计 | `article-management-S7.1`、`article-management-S7.2` | 已登记 |
| A-CON-01～04 | 8 条 article 主责 US / SR-ARTICLE-01～04、SR-IDENTITY-03 | design §3.3.1、§3.4.2；R-A01/R-A08/R-A09 | REST、内部契约、DB 权限 | 16 endpoint schema、三投影 allowlist、S1–S16 fixture、Audit 失败 | `article-management-S1.5` | 已登记 |
| A-REC-01～03 | US-ARTICLE-04.1 / SR-ARTICLE-04；S16 | design §3.4.7、§6.1；R-A10 | producer schema/canonical 契约 | Ready20/Unavailable6、固定 digest、负向 fixture | `article-management-S4.2`、`article-management-S1.5` | 已登记 |
| A-REC-04～07 | US-ARTICLE-04.1 / SR-ARTICLE-04；S16 | design §3.4.7；R-A02/R-A10 | PostgreSQL 并发、崩溃恢复 | run/backup 唯一绑定、input_digest、Evidence/current 指针 | `article-management-S4.2`、`article-management-S4.4` | 已登记 |
| A-REC-08 | US-ARTICLE-04.1、US-IDENTITY-03.2/03.8/03.12；S11/S16 | design §3.4.5～7、§4.2；R-A02/R-A10 | 真实备份恢复、双模块 | 三类 tombstone 重放、operation/audit 对账、双次 ReadStatus | `article-management-S4.4`、`identity-auth-S4.4` | 已登记 |
| A-PERF-01～05 | 8 条 article 主责 US | design §4.4、§6.6；R-A12 | 性能/容量 | 基准报告、积压/资源指标、批准阈值 | `article-management-S2.3` | 已登记 |

## 7. 数据与环境计划

| 条件 | 计划 |
|---|---|
| PII 与内容 | 合成用户、墓碑、邮箱、标题和正文；包含 HTML 字面量、Unicode、空白边界；不得使用生产数据 |
| PostgreSQL | 隔离数据库/schema；至少双连接验证行锁、唯一约束、SKIP/条件更新和事务回滚 |
| 可控时钟 | ETag 不依赖时钟；cursor 时间、recipient lease/backoff、tombstone deleted_at、completed_at 可冻结 |
| identity fixture | active/suspended/deleted/recovery 状态 Principal 与作者批量状态；真实跨模块集成时不用 mock 冒充 |
| SMTP | 支持 TLS/STARTTLS、证书故障、受理/瞬断/超时/永久错误；记录连接次数但不对外发送 |
| 双 worker | dispatcher、finalizer、ReplayAndVerify 并发各至少两个 worker/事务 |
| Article Deletion Journal | 独立持久卷；可注入部分写、重复 sequence、链/HMAC/key/namespace/fsync/空间故障 |
| Recovery Journal | 由 identity 恢复专项提供；两个 namespace 使用不同链和 HMAC domain |
| keyring | 一个 active + 多个 verify-only；覆盖缺 key、错误 key、完整迁移和保留条件 |
| 备份样本 | 删除前数据库备份、删除后三类 tombstone、operation/audit/inventory 快照 |
| tagged union | Ready20、Unavailable6、混合/缺失/额外/null/类型错误 fixture；固定 canonical bytes/digest |

环境、fixture、证书、备份和清理准备已登记为 `identity-auth-S1.4`、`identity-auth-S4.3`、`article-management-S4.5`；实际运行与清理责任仍未分配。

## 8. 自动化与人工边界

| 自动化 | 必须人工或独立责任确认 |
|---|---|
| validator、状态矩阵、cursor、ETag、serializer、canonical digest | AM-Q1 的格式/长度与是否支持 Markdown |
| PostgreSQL 事务、并发、canonical 删除、lease/finalizer | UI 永久删除二次确认与最终交互体验 |
| SMTP TLS、ProcessingGate、重试和 accepted 崩溃 | 实际地区、控制者、处理依据和 SMTP 处理方登记 |
| Journal、inventory、evidence、旧备份恢复 | tombstone/key/evidence/通知/Audit 的运行留存和退役授权 |
| HTTP/内部端口契约和敏感字段扫描 | 真实恢复运行、RPO/RTO 和 readiness 开放签核 |

## 9. 任务建议

以下建议已由 task-planner 登记；实际状态只在 [tasks.md](../../tasks/tasks.md) 维护。

| 建议项 | 类型 | 建议所有权 | 依赖 | 关闭条件 | 停止条件 | 实际 task 引用 |
|---|---|---|---|---|---|---|
| 文章策略、状态机、cursor、ETag 局部测试 | 开发 task 内 | article 开发 | 对应实现切片 | 纯策略、validator、parser 和错误映射随实现通过 | 需求/设计合同不明确 | `article-management-S1.2`、`article-management-S1.3`、`article-management-S2.1`、`article-management-S3.1`、`article-management-S5.1`、`article-management-S6.1`、`article-management-S6.2`、`article-management-S8.1` |
| 文章 REST 与内部端口契约测试 | 开发 task 内/共享专项 | article + platform | schema、S1–S16 | 16 endpoint、三种投影、公共错误和内部端口 fixture 通过 | schema/消费者未形成 | `article-management-S1.5` |
| 文章 PostgreSQL 状态与分页专项 | 专项集成 | article 测试/开发 | schema、migration、隔离库 | 状态组合、cursor、多批扫描、ETag 并发通过 | 无真实 PostgreSQL | `article-management-S2.2` |
| canonical 删除跨模块专项 | 专项集成 | identity 主责、article 协作 | 双模块 schema/端口 | article 侧零/多/违规/pending/重放矩阵通过 | 任一模块未实现 | `identity-auth-S9.2` |
| 通知 lease、双 worker 与 SMTP/TLS 专项 | 专项集成/安全 | article 测试 | SMTP、证书、可控时钟 | 五态、Gate、TLS、崩溃和两收件人证据齐全 | 可能发送外部邮件或无法控制 TLS | `article-management-S7.2` |
| Article Deletion Journal/finalizer 专项 | 专项韧性 | article 测试 | 持久卷、keyring、DB | 三类删除、fsync、崩溃、并发、无 PII 通过 | Journal provider/环境未形成 | `article-management-S4.3` |
| ReplayStatus producer 与旧备份恢复专项 | 专项契约/韧性 | article 主责、identity 协作 | inventory、备份、双 Journal | Ready/Unavailable、并发绑定、输入漂移、联合恢复通过 | retention/RPO/RTO 或运行责任缺失影响真实演练 | `article-management-S4.4` |
| article 性能基准 | 专项性能 | 性能测试 | 目标环境、容量模型 | 分页、dispatcher、finalizer、replay 有基线和批准目标 | AM-Q1/Q2、容量目标未确认 | `article-management-S2.3` |
| article 真实 E2E 旅程 | 独立 E2E | Test-Engineer + 业务验收人 | 部署、Case、数据和责任人 | 真实入口/后端数据流/数据符合性三层证据齐全 | 环境、责任人或真实邮件安排未确认 | `article-management-S1.4` |

## 10. 执行顺序

1. validator、VisibilityPolicy、状态机、cursor、ETag、canonical digest 和 REST/S1–S16 契约；
2. PostgreSQL schema/约束/行锁、分页、权限和审计；
3. canonical 删除、无归属处置和邮箱清理协作；
4. notification lease、双 worker、SMTP/TLS 和隐私故障；
5. Article Deletion Journal、finalizer、ReplayStatus、旧备份恢复；
6. 容量参数确定后的性能基准；
7. 部署后少量真实 E2E。

## 11. 执行前 Gate

| Gate | 状态 | 依据与继续条件 |
|---|---|---|
| 8 US 与关键验收可观察 | ready | requirements、design、stories 有明确响应/状态/数据证据 |
| 状态组合、权限、删除、通知和恢复高风险场景 | ready | §4 已覆盖主路径、负向、并发、事务和崩溃 |
| 跨计划主责与 S1–S16 | ready | §5 已避免重复和遗漏 |
| Ready20/Unavailable6、fixture/digest | ready | 已接受封闭 schema |
| 代码、schema、migration、REST schema | blocked | 当前不存在 |
| 测试框架与项目检查命令 | blocked | Manifest 无命令；不得虚构 |
| PostgreSQL、SMTP、双 worker、持久卷和备份环境 | blocked | 未提供 |
| 合成数据、证书、inventory 与清理责任 | blocked | 已登记 `identity-auth-S1.4`、`identity-auth-S4.3`、`article-management-S4.5`；实际责任仍待分配 |
| AM-Q1 内容格式/长度 | partial | 非空、文本 JSON 与安全输出可测；最终 validator/容量被阻塞 |
| AM-Q2 分页/规模/性能目标 | partial | 算法与正确性可测；limit、批次和 P95 目标被阻塞 |
| AM-Q3 UI 二次确认 | partial | 后端鉴权/ETag/审计可测；最终 UX 被阻塞 |
| ProcessingGate 实际值 | partial | 缺值 fail-closed 可测；通知能力真实开放被阻塞 |
| SMTP/lease/retry/finalizer 容量值 | partial | 状态机可测；性能和告警验收被阻塞 |
| tombstone/key/inventory/evidence retention、RPO/RTO | partial | fail-closed/结构可测；退役和真实恢复结论被阻塞 |
| 人工验收与 E2E 责任人 | blocked | E2E task 已登记为 `article-management-S1.4`；实际 Test-Engineer 与业务验收人仍未分配 |

## 12. 当前阶段结论

article-management 的稳定设计切片已形成风险驱动规划，覆盖 8 条主责 US、S1–S16 的 article 责任、状态组合、cursor/ETag、notification lease、Article Deletion Journal、ReplayStatus tagged union 和旧备份恢复。当前结论仅为 **test plan complete as planned, tests not executed**；缺少代码、schema、命令、环境、数据和责任人，不能宣称可执行、通过或已完成 E2E。
