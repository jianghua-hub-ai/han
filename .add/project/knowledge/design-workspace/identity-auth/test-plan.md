---
feature_id: identity-auth
artifact: test-plan
status: planned-not-executed
revision: identity-auth-test-plan-ir01-r01
created: 2026-10-20
last_updated: 2026-10-20
source_design_revision: identity-auth-design-ir01-r09
source_design_artifact: sha256:df484400154f7b64c00c40842e02eb254408bdb1c764768c51bf9672dbd09c61
requirements_checkpoint: decision-e3af3912166f7287e8324488cad1cf1b87ba1034fca3378d3e0467a23751592d
architecture_checkpoint: decision-0347e1026fd9148ad16f66750023cb50445a1d7c133db6d82669ea14865b318f
design_checkpoint: decision-094b56eaec9148d2f91829ae0c3beb9820ce8ec27a347d9af0cc339caa073e02
---

# identity-auth 风险驱动测试规划

本规划状态为 **planned / not executed**。它定义稳定设计切片应如何验证，不表示代码、数据库、契约、环境、测试、恢复演练或 E2E 已存在或已通过。

## 1. 对象、范围与来源

测试对象是 `identity-auth-design-ir01-r09` 及其 11 条主责 Story。设计已通过独立静态评审，但实施 Gate 按 `decision-094b56eaec9148d2f91829ae0c3beb9820ce8ec27a347d9af0cc339caa073e02` 处于 `deferred:reorder`；期限、容量、隐私部署值和灾备运行参数仍未确定。

| 来源 | 本规划使用的事实 |
|---|---|
| [requirements r16](../../requirements/requirements.md) | 11 条 identity 主责 US、可观察验收、隐私与事务性邮件门禁 |
| [architecture r12](../../design/architecture.md) | 模块化单体、REST/JSON、PostgreSQL、恢复顺序与联合 readiness |
| [identity boundary r11](../../design/identity-auth/boundary.md) | identity 所有权、S1–S16 接缝、Recovery Journal 与 ReplayStatus 消费边界 |
| [article boundary r12](../../design/article-management/boundary.md) | article 提供的 canonical 删除、Article Deletion Journal 与 ReplayStatus 合同 |
| [identity design r09](design.md) / [stories r09](stories.md) | 状态机、接口、权限、数据、错误、安全、恢复和 11 条实现切片 |
| [article design r06](../article-management/design.md) / [article test plan](../article-management/test-plan.md) | article 主责的可见性、删除、通知、ReplayStatus 生产者验证 |
| [identity review r09](../../../../../.spec/wip/identity-auth/review-design-r09.md) | PASS 范围与尚未执行的真实 PostgreSQL、TLS、并发和恢复验证 |
| [ADR 索引](../../decisions/README.md) | ADR-0001～ADR-0020 的已接受约束 |
| [项目 Manifest](../../../../../add.manifest.yaml) | `go-service` 已启用；项目检查命令为空 |

### 1.1 成功证据

规划完成的证据是：

1. 11 条主责 US 都映射到场景、风险、层次和计划证据；
2. S1–S16 的 identity 责任与 article 主责引用无遗漏；
3. 高风险失败有负向、并发、事务和恢复场景；
4. 开发局部测试、专项测试和部署后真实 E2E 分开；
5. 未决输入只阻塞受影响场景，不被填入虚构数值；
6. test-plan 建议已由 task-planner 登记并在 §6、§7、§9、§11 回填实际 task 引用；任务状态仍只在 tasks SSOT。

## 2. 验收—风险地图

优先级由影响与发生可能性共同决定。影响依据是权限提升、凭据泄露、不可逆删除、隐私违规、跨备份恢复错误和跨模块不一致；可能性依据是外部输入、并发、跨介质事务、SMTP、崩溃续跑及人工部署配置，而不是代码行数。

| 风险 | 优先级 | 影响 | 可能性依据 | 主要场景 |
|---|---|---|---|---|
| R-I01 凭据、Session、CSRF 或恢复 secret 泄露/复用 | P0 | 账号接管、恢复信任根被重复建立 | 浏览器 Cookie、邮件明文、日志、备份恢复和多版本 keyring 均跨边界 | I-AUTH-01～07、I-CRED-01～06、I-REC-01～08 |
| R-I02 权限矩阵或目标资格旁路 | P0 | 普通用户或普通管理员获得高权限 | 多角色、多目标和恢复态组合多，每请求都需重判 | I-ADM-01～05、I-GOV-01～05 |
| R-I03 用户删除或 canonical 删除部分提交 | P0 | 已删账号仍可登录，文章仍有归属或错误公开 | User、Identifier、Session、Credential、Article、Audit 同一业务事务 | I-DEL-01～07 |
| R-I04 恢复误开放或 stale evidence 被接受 | P0 | 旧密码、旧权限或已永久删除文章复活 | 双 Journal、两个 namespace、run/backup、崩溃和输入漂移 | I-REC-01～10 |
| R-I05 隐私门禁或专用邮箱用途越界 | P0 | 未告知收集、未经登记外部处理或 PII 过度保留 | 注册、管理员创建、SMTP 和删除后通知均处理 PII | I-REG-01～06、I-DEL-05～07 |
| R-I06 SMTP/TLS 与数据库残余窗口 | P0 | 未交付账号、无效凭据邮件、明文或错误处理方发送 | SMTP 与数据库不能原子，证书和门禁可配置错误 | I-CRED-01～06 |
| R-I07 幂等、乱序或并发导致重复副作用 | P0 | 重复管理员、重复邮件、旧凭据复活 | 网络重试、同键异摘要、租约和恢复状态并发 | I-REG-04、I-CRED-02～06、I-GOV-04 |
| R-I08 登录枚举、暴力破解、CSRF/CORS 配置错误 | P0 | 账号枚举、跨站写入、资源耗尽 | 登录是公开入口且 Argon2id 成本高 | I-AUTH-01～08、I-PERF-01 |
| R-I09 审计缺失或含敏感数据 | P1 | 高风险动作不可追责或形成二次泄露 | 所有状态写与恢复都要求追加式审计 | I-AUD-01～03 |
| R-I10 标识保留/释放错误 | P1 | 用户名冒用、邮箱/手机号过早复用 | 删除墓碑与清理条件跨 identity/article | I-REG-03、I-DEL-05～07 |
| R-I11 游标、错误映射和最小披露漂移 | P1 | 列表漏重、消费者误判、敏感状态泄露 | 治理列表和统一错误均是公共契约 | I-LIST-01～03、I-CON-01～03 |
| R-I12 容量、期限和保留参数不合适 | P1 | DoS、积压、过早清理或恢复超时 | 当前目标值均为 TBD | I-PERF-01～04 |

### 2.1 11 条 US 的关键验收与证据

| US | 关键可观察验收 | 至少一个计划证据 | 场景 |
|---|---|---|---|
| US-IDENTITY-01.1 建立用户身份 | 当前隐私确认与三个标识原子创建；门禁缺失不创建 | HTTP 201/失败 envelope、User/Identifier/PrivacyAcknowledgement 同事务快照 | I-REG-01～05 |
| US-IDENTITY-01.2 设置注册密码 | 规则正确且仅存 Argon2id 自描述摘要 | validator 断言、数据库摘要格式、日志/审计敏感扫描 | I-REG-06、I-AUTH-06 |
| US-IDENTITY-02.1 登录社区 | 统一失败、受保护 Cookie、CSRF、旧会话轮换 | HTTP Header/Body、Session 摘要行、浏览器同源网络证据 | I-AUTH-01～08 |
| US-IDENTITY-03.1 初始管理员 | 唯一 bootstrap、永久保护、首次改密、恢复 secret 独立 | 启动并发记录、权限拒绝、受限会话、Journal 指纹 | I-ADM-01～03、I-REC-01～03 |
| US-IDENTITY-03.3 创建管理员 | 仅 initial admin；声明、SMTP、幂等、双 Gate 全成立 | IdempotencyRecord、SMTP 测试服务记录、User/TemporaryCredential/Audit | I-CRED-01～04 |
| US-IDENTITY-03.4 管理管理员 | 仅 initial admin；撤权/停用后旧会话失效 | 权限矩阵响应、User/Role/Session/Audit 事务证据 | I-ADM-04～05 |
| US-IDENTITY-03.5 查看用户 | 最小披露、稳定分页、不可枚举管理员 | JSON 字段 allowlist、cursor 遍历序列、拒绝响应 | I-LIST-01～03 |
| US-IDENTITY-03.6 停用与恢复 | 目标状态幂等；停用撤销会话；文章仅派生变化 | User/Session/Audit 事务、article 查询结果 | I-GOV-01～03 |
| US-IDENTITY-03.7 删除用户 | 不可恢复墓碑；文章无归属化；标识生命周期与清理 | 跨模块事务快照、权限矩阵、注册复用结果 | I-DEL-01～07 |
| US-IDENTITY-03.10 重置密码 | 最新临时凭据唯一有效；SMTP 未受理不改变旧状态 | TemporaryCredential generation、Session、SMTP 与幂等记录 | I-CRED-02、I-CRED-05 |
| US-IDENTITY-03.11 重发临时密码 | 授权按 purpose；成功替换、失败保留旧值 | generation/状态机、SMTP 记录、旧新凭据登录结果 | I-CRED-03、I-CRED-06 |

## 3. 测试分层

| 层次 | 证明内容 | 本特性重点 |
|---|---|---|
| 单元/组件 | 局部策略、不变量、状态机和错误映射 | 归一化、密码策略、Argon2 编码解析、TemporaryCredential、credential_mode、双 Gate、权限矩阵、cursor、请求摘要、公共错误 |
| 契约 | 生产者与消费者对 REST 和内部端口的共同理解 | REST schema/错误、Cookie/CSRF、S1–S16、Ready20/Unavailable6 消费者拒绝、审计 action |
| 集成 | 真实存储、锁、事务和基础设施边界 | PostgreSQL 约束/行锁/回滚、SMTP TLS 服务、跨模块 canonical 删除、持久卷 Recovery Journal |
| 安全/韧性 | 攻击、错误配置、崩溃和恢复时 fail closed | 暴力登录、CSRF/CORS、TLS 证书、隐私门禁、Journal 损坏、keyring 轮换、双 namespace readiness |
| 性能 | 目标环境资源与容量维度 | Argon2id 延迟/内存/并发、Session 查询、治理事务、SMTP、恢复批量；目标值 TBD |
| E2E | 少量真实业务旅程和真实部署联通 | 注册登录、管理员治理、删除用户、凭据恢复、旧备份恢复；后续独立任务 |

## 4. 计划场景

### 4.1 注册、隐私与密码

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| I-REG-01 | P0 | ProcessingGate 完整；读取 current notice，确认后提交合法身份与密码 | 201；User、三个 Identifier、PrivacyAcknowledgement、AuditEvent 同事务；无 Session | REST 契约 + PostgreSQL 集成；自动 |
| I-REG-02 | P0 | 缺字段、未确认、旧 notice、重复/未知 JSON 字段 | 422/稳定错误；数据库无 User 或部分 Identifier | validator 单元 + HTTP/DB 集成；自动 |
| I-REG-03 | P0 | 现有账号、删除用户名、保留邮箱/手机号、已清理邮箱/手机号分别注册 | 前三类拒绝；已清理标识只重新进入唯一校验，不恢复墓碑 | PostgreSQL 约束/生命周期集成；自动 |
| I-REG-04 | P0 | 双事务并发提交相同 username/email/phone 或交叉冲突 | 最多一个账号成功；失败事务无部分数据；审计与业务一致 | 真实 PostgreSQL 并发；自动 |
| I-REG-05 | P0 | 地区、控制者、处理依据、SMTP 处理方逐项缺失 | 普通注册保持关闭；错误不声称特定法域结论 | 门禁组件 + HTTP 集成；自动；实际值内容由人工确认 |
| I-REG-06 | P0 | 密码长度、字母、数字边界；合法密码；哈希资源失败 | 无效拒绝；合法只保存 Argon2id 自描述编码；失败无账号；任何证据不含明文 | 单元 + PostgreSQL/日志扫描；自动；参数目标待基准 |

### 4.2 登录、Cookie、CSRF、CORS 与防暴力

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| I-AUTH-01 | P0 | active 用户从可信同源登录 | 统一成功响应；`__Host-han_session` 具 Secure/HttpOnly/SameSite=Lax/Path=/ 且无 Domain；返回 CSRF 明文，库中仅摘要 | HTTP/浏览器集成；自动 |
| I-AUTH-02 | P0 | 不存在、错密码、suspended、deleted、recovery_required、recovery_locked、过期临时凭据 | 均为相同 `401 auth.login_failed` 与等价 message；无新 Session；响应不泄露原因 | 契约 + 状态矩阵集成；自动 |
| I-AUTH-03 | P0 | 恶意/缺失 Origin、cross-site Fetch Metadata、credentialed CORS、伪造 CSRF | login 或写请求 fail closed；无状态改变；无 `Access-Control-Allow-Credentials` 跨源开放 | 浏览器/HTTP 安全集成；自动 |
| I-AUTH-04 | P0 | 已有有效会话再次登录成功与失败 | 成功时新建并轮换会话、旧 Cookie 下一请求失效；失败时旧会话保持 | Session/PostgreSQL 集成；自动 |
| I-AUTH-05 | P0 | 临时凭据 active/expired/superseded/consumed/recovery_invalidated 与旧永久密码组合 | 只接受 credential_mode 指向的唯一有效路径；绝不回退旧永久密码 | 状态机单元 + 登录集成；自动 |
| I-AUTH-06 | P0 | Argon2 参数缺失、旧参数、资源过载、并发信号量饱和 | 缺基线拒绝启动；旧参数成功后按策略重哈希；过载 429/503 且不泄露账号 | 组件 + 目标环境基准；部分，参数未定 |
| I-AUTH-07 | P0 | 单账号多来源、多账号单来源、账号+来源、代理 Header 伪造、重启 | 三类持久桶原子计数；非可信代理 Header 被忽略；账号维度仍统一失败 | PostgreSQL 并发 + 安全测试；自动，阈值未定 |
| I-AUTH-08 | P1 | 登录成功/失败、CSRF 拒绝、限流与内部故障 | 客户端错误不记 ERROR；每个动作有最小审计/指标，日志不含密码、Cookie、CSRF、原始来源 | 审计/日志契约；自动扫描 + 人工抽查 |

### 4.3 管理员与权限矩阵

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| I-ADM-01 | P0 | 空库双进程/双 goroutine 并发 bootstrap | 只有一个 `admin`；重复启动不覆盖凭据；配置值不进日志/审计 | PostgreSQL 并发 + 启动集成；自动 |
| I-ADM-02 | P0 | initial admin 首次登录、完成改密前调用管理能力 | 仅允许 me/改密/隐私接口；成功改密后轮换会话并开放允许能力 | HTTP/权限集成；自动；具体改密规则受 Q1 阻塞 |
| I-ADM-03 | P0 | 任意角色停用、删除、撤权或重置 initial admin | 全部拒绝且账号、角色、凭据和会话不变；正常自改密单独允许 | 权限矩阵 + DB 快照；自动 |
| I-ADM-04 | P0 | initial admin 与普通管理员分别查看/治理其他管理员、自己和普通用户 | 仅批准组合成功；普通管理员不可枚举管理员治理信息 | 表驱动权限单元 + HTTP 集成；自动 |
| I-ADM-05 | P0 | 撤权、停用或删除其他管理员后复用旧会话 | 下一请求立即失效；角色/状态/Session/Audit 同事务或全回滚 | PostgreSQL 集成；自动 |

### 4.4 临时凭据、SMTP、TLS 与幂等

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| I-CRED-01 | P0 | initial admin 以 current 声明和 Idempotency-Key 创建管理员 | SMTP 受理后才提交账号/临时凭据/声明/审计/幂等结果；双 Gate 未完成时受限 | HTTP + PostgreSQL + SMTP 集成；自动 |
| I-CRED-02 | P0 | 同键同摘要并发/重试、同键异摘要、processing、uncertain、recovery_review_required | 同摘要只执行一次并返回原结果；异摘要冲突；不确定状态不自动重放 | PostgreSQL 锁/幂等集成；自动 |
| I-CRED-03 | P0 | 创建管理员、重置、重发的角色/purpose 组合 | 新管理员重发仅 initial admin；普通用户重置/重发由有权管理员；无上下文拒绝 | 权限 + purpose 契约；自动 |
| I-CRED-04 | P0 | 新管理员按两种顺序完成改密和隐私；中途 notice 换版 | 两 Gate 都完成且版本为 current 才开放；已完成 onboarding 的账号换版不重锁 | 状态机 + PostgreSQL 集成；自动 |
| I-CRED-05 | P0 | SMTP 未受理、TLS 不可用、STARTTLS 降级、过期/错主机证书、跳过验证、处理方门禁缺失 | fail closed；旧凭据/Session 保持；无账号或新凭据生效；不使用明文 SMTP | 故障注入 SMTP/TLS 集成；自动 |
| I-CRED-06 | P0 | SMTP accepted 后数据库提交失败/进程崩溃；随后同键重试 | 返回 uncertain/503；邮件中的值不可用；不重复产生有效副作用；有 trace、告警和审计 | SMTP + DB 故障注入；自动；外部真实邮件不在本任务执行 |

### 4.5 查询、治理、删除与标识生命周期

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| I-LIST-01 | P1 | 普通管理员列普通用户；initial admin 列管理员；空/单/多页、同时间戳 | 稳定 `(created_at,id)` 顺序；不漏不重；末页 `next_cursor:null/has_more:false` | cursor 单元 + PostgreSQL 集成；自动 |
| I-LIST-02 | P1 | 非法、未知版本、过期或过滤不匹配 cursor | `request.invalid_cursor`；无数据披露 | 契约 + fuzz/属性测试建议；自动 |
| I-LIST-03 | P0 | Summary、Detail、管理员详情序列化 | Summary 无邮箱/手机号；任何响应无密码、临时凭据、Session、retained_email、审计 metadata | 字段 allowlist 契约；自动 |
| I-GOV-01 | P0 | active→suspended、重复 suspend、suspended→active、重复 restore | 目标状态幂等，`changed` 稳定；每次重新鉴权；Audit 与状态同事务 | 状态机 + PostgreSQL 集成；自动 |
| I-GOV-02 | P0 | 停用有多个 Session 的用户 | 全部会话立即撤销；旧 Cookie 下一请求失败 | Session/DB 集成；自动 |
| I-GOV-03 | P0 | 停用/恢复作者后读取普通和违规文章 | 不批量写 Article；普通文章按作者状态隐藏/恢复，违规仍隐藏 | 跨模块集成；主责在本计划，article 计划引用 |
| I-GOV-04 | P0 | 治理时目标并发被删除/撤权；审计写失败 | 带旧状态条件的更新只成功一次；审计失败全回滚 | PostgreSQL 锁/故障注入；自动 |
| I-GOV-05 | P0 | 非管理员、普通管理员治理自己/管理员/initial admin | 拒绝，目标状态和会话不变；不披露受限制详情 | 权限矩阵集成；自动 |
| I-DEL-01 | P0 | 删除普通用户，含普通文章、违规文章和多个 Session/凭据 | User 墓碑、Identifier、Session/凭据失效、Article NULL/held/source、违规保留、Audit 同事务 | 跨模块真实 PostgreSQL；主责在本计划 |
| I-DEL-02 | P0 | ArticleAccountPort、标识写入或 Audit 任一步失败 | 整体回滚；不得出现半墓碑或半无归属文章 | 事务故障注入；自动 |
| I-DEL-03 | P0 | 删除零文章用户 | pending=0；同事务清理 retained_email/email Identifier；用户名永久保留 | PostgreSQL 跨模块集成；自动 |
| I-DEL-04 | P0 | 删除不存在、受保护或越权目标；撤回请求由运营转交 | 稳定拒绝；不新增运营权限或自助入口；其他用户无变化 | HTTP/权限契约；自动 + 人工流程确认 |
| I-DEL-05 | P0 | 删除后以 username/email/phone 注册；完成允许清理后重试 | username 永久拒绝；保留期内 email/phone 拒绝；清理后只重新唯一校验 | 生命周期集成；自动；phone 清理条件阻塞 |
| I-DEL-06 | P0 | 多篇无归属文章逐步恢复/转移/删除，通知快照已/未留存 | 条件不足保留 retained_email；最后完成事实成立后才清理 | 跨模块集成；article 处置主责，identity 清理主责 |
| I-DEL-07 | P0 | 恢复核对结论为 kept_deleted，覆盖零/多/违规/部分已完成矩阵 | 幂等重放与日常删除相同 canonical 结果；任一步失败不完成 review/readiness | 恢复专项集成；主责在本计划 |

### 4.6 审计、公共错误与内部合同

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| I-AUD-01 | P0 | 注册、登录、会话、治理、凭据、标识清理、恢复的成功/失败 | action、actor_kind、目标、结果、code、trace 齐全；高风险成功与业务同事务 | 审计集成；自动 |
| I-AUD-02 | P0 | Audit INSERT 失败或尝试 UPDATE/DELETE | 高风险业务 fail closed；应用无更新删除能力 | DB 权限/故障注入；自动 |
| I-AUD-03 | P0 | 对日志、错误、审计、幂等快照做敏感字段扫描 | 无密码、临时密码、Cookie、CSRF、secret、完整邮件正文或不必要 PII | 自动扫描 + 人工抽查 |
| I-CON-01 | P1 | 公共错误注册逐项核 code/HTTP/message/retry/log/cause | 一个 code 只有稳定映射；内部 cause 只保留在服务端 | 单元/契约；自动 |
| I-CON-02 | P0 | REST 严格 JSON、未知/重复字段、Content-Type、trace_id | 非法请求稳定拒绝；响应符合 `{code,message,field_errors?,trace_id}` | REST schema 契约；自动 |
| I-CON-03 | P0 | S1–S16 内部端口编译/契约 fixture | 调用方向、输入输出、失败语义和所有权与双边设计一致 | 内部 Go 契约；自动；当前无代码/schema |

### 4.7 Recovery Journal、ReplayStatus 与 readiness

| ID | 优先级 | 前置与动作 | 预期与计划证据 | 层次/自动边界 |
|---|---|---|---|---|
| I-REC-01 | P0 | 正常恢复：maintenance、Begin、DB restore、身份终结、initial admin recovery、逐账号核对 | 全部永久密码/Session/current 临时凭据失效；非 initial 账号默认不授权 | 真实备份恢复集成；自动断言 + 人工运行签核 |
| I-REC-02 | P0 | 同 run、同 secret 跨 active v1/v2/v3 重放；不同 secret | retained key 任一命中即拒绝；不同 secret 只用 active key 写新 begun | 多版本 keyring 集成；自动 |
| I-REC-03 | P0 | 缺历史 key、未知 key、只迁 Journal、只迁 active key、verify-only 不完整 | 全部保持停服；不得重建信任根 | 持久卷/keyring 故障注入；自动 |
| I-REC-04 | P0 | Journal 部分写、HMAC/链损坏、重复 sequence、只读、空间不足、fsync 失败 | Begin/Read/MarkConsumed fail closed；RecoveryRun 不前进 | 文件系统故障注入；自动 |
| I-REC-05 | P0 | RecoveryRun 与 Journal 的 run/key/fingerprint/stage 不一致 | VerifyAgainst 失败，readiness=false | PostgreSQL + 持久卷集成；自动 |
| I-REC-06 | P0 | 接收 Ready20 fixture、Unavailable6 三状态及混合/缺失/额外/null/类型错误/未知版本 | 仅完整 Ready 执行 canonical 重算；Unavailable 立即停止且证明未调用 digest 计算器 | consumer 契约；自动 |
| I-REC-07 | P0 | Ready canonical 前 16 字段、固定 fixture、四个排除字段分别变化 | fixture 摘要为 `169e0f…926a7`；排除字段不影响摘要；前 16 任一变化影响摘要 | canonical digest 单元/契约；自动 |
| I-REC-08 | P0 | backup A ready 用于 B、错 namespace、二次 ref/digest 变化、旧 ready/input 漂移 | 全部 fail closed；不 MarkConsumed；清除已缓存 ready 结论 | 跨模块契约/恢复集成；自动 |
| I-REC-09 | P0 | `recovery/v1` 与 `article-deletion/v1` 独立通过及任一混用/缺失 | 两个 namespace 分别验链；混用视为损坏；联合 readiness 不开放 | 双 Journal 恢复专项；自动 |
| I-REC-10 | P0 | 崩溃发生在 Begin、身份终结、article replay、二次 ReadStatus、MarkConsumed 前后 | 续跑收敛；只有二次同 ref/digest + MarkConsumed 原子落盘后才开放 | 崩溃恢复专项；自动 + 人工运行签核 |

### 4.8 性能和容量维度

| ID | 状态 | 测量维度 | 当前结论 |
|---|---|---|---|
| I-PERF-01 | blocked | Argon2id 单次延迟、内存、并发、拒绝行为 | 参数和目标值待安全/容量基准 |
| I-PERF-02 | blocked | Session 查询、登录桶、治理列表和跨模块删除事务 P95/吞吐 | 用户/会话/文章规模与目标值 TBD |
| I-PERF-03 | blocked | SMTP 超时/并发、幂等记录与审计增长 | 超时、并发、保留期 TBD |
| I-PERF-04 | blocked | 恢复批量账号、全 keyring 判重、双 Journal replay 与 RTO | RPO/RTO、批量规模、retained key 数量 TBD |

## 5. S1–S16 合同责任

| 接缝 | 本计划责任 | 另一计划责任 |
|---|---|---|
| S1 同源、Session、CSRF | 主责：I-AUTH-01～04 | article 验证只消费 Principal |
| S2 作者状态 | 提供方：I-GOV-03、I-CON-03 | article 主验可见性组合 |
| S3 停用/恢复 | 主责：I-GOV-01～03 | article 引用派生结果 |
| S4 canonical 删除 | 端到端主责：I-DEL-01/02/07 | article 验证 `OrphanArticlesCanonical` 内部结果 |
| S5 处置与清理 | identity 主验清理门禁：I-DEL-06 | article 主验三种处置与通知快照 |
| S6 邮件/TLS | 凭据邮件主责：I-CRED-01～06 | 转移通知主责在 article |
| S7 权限 | identity 角色主责：I-ADM/I-GOV | article 动作矩阵主责在 article |
| S8 公共错误 | 主责共享契约：I-CON-01/02 | article 验证自身 code |
| S9 幂等 | identity 键式幂等主责：I-CRED-02 | article 只验证未擅自接受键式幂等 |
| S10 审计 | identity action 主责：I-AUD | article action 主责在 article |
| S11 Article Deletion Journal | 引用，不重复验证删除生产者 | article 主责 |
| S12 隐私/部署门禁 | 注册/凭据主责：I-REG-05/I-CRED-05 | 转移通知门禁主责在 article |
| S13 恢复信任根 | 主责：I-REC-01～05 | article 验证恢复期间 API 关闭 |
| S14 逐账号核对 | 主责：I-DEL-07/I-REC-01 | article 验证 canonical article 侧 |
| S15 Recovery Journal | 主责：I-REC-02～05 | article 只消费恢复开放结果 |
| S16 ReplayStatus Gate | consumer 与联合开放主责：I-REC-06～10 | producer/schema/evidence 主责在 article |

## 6. 覆盖映射

场景范围表示该范围内每个场景 ID 都继承本行映射；场景的逐项前置、动作和预期见 §4。

| 场景 ID | US/SR | 设计章节/风险 | 层次 | 计划证据 | 任务建议 | 登记状态 |
|---|---|---|---|---|---|---|
| I-REG-01～05 | US-IDENTITY-01.1 / SR-IDENTITY-01 | design §3.3.2、§3.4.1；R-I05/R-I10 | 单元、REST、PostgreSQL、隐私门禁 | HTTP、User/Identifier/PrivacyAcknowledgement/Audit 事务快照 | `identity-auth-S1.3`、`identity-auth-S1.6` | 已登记 |
| I-REG-06 | US-IDENTITY-01.2 / SR-IDENTITY-01 | design §3.3.3；R-I01/R-I08 | 单元、集成、安全基准 | validator、Argon2 编码、无明文扫描 | `identity-auth-S2.1`、`identity-auth-S2.2` | 已登记 |
| I-AUTH-01～05 | US-IDENTITY-02.1 / SR-IDENTITY-02 | design §3.3.1、§3.3.4、§3.4.4；R-I01/R-I08 | REST、浏览器、PostgreSQL | Cookie/CSRF Header、Session 摘要、统一失败 | `identity-auth-S3.1`、`identity-auth-S3.2`、`identity-auth-S3.3`、`identity-auth-S1.5` | 已登记 |
| I-AUTH-06～08 | US-IDENTITY-02.1 / SR-IDENTITY-02 | design §4.1、§4.4、§5.3；R-I08/R-I09/R-I12 | 安全、韧性、性能 | Argon2/桶指标、审计、日志级别与代理解析 | `identity-auth-S3.2`、`identity-auth-S3.3`、`identity-auth-S2.2` | 已登记 |
| I-ADM-01～03 | US-IDENTITY-03.1 / SR-IDENTITY-03 | design §3.3.5、§3.4.6；R-I01/R-I02 | 组件、PostgreSQL、权限 | 唯一 admin、受限会话、拒绝后的 DB 快照 | `identity-auth-S4.1`、`identity-auth-S1.6` | 已登记 |
| I-ADM-04～05 | US-IDENTITY-03.4 / SR-IDENTITY-03 | design §3.3.7、§3.4.6；R-I02/R-I09 | 权限、事务、契约 | 角色/目标矩阵、Session 撤销、Audit | `identity-auth-S6.1`、`identity-auth-S1.5` | 已登记 |
| I-CRED-01、I-CRED-04 | US-IDENTITY-03.3 / SR-IDENTITY-03 | design §3.3.6、§3.4.5；R-I05/R-I06 | REST、PostgreSQL、SMTP | 声明、双 Gate、User/TemporaryCredential/Audit | `identity-auth-S5.1`、`identity-auth-S5.2`、`identity-auth-S5.3` | 已登记 |
| I-CRED-02 | US-IDENTITY-03.3、03.10、03.11 / SR-IDENTITY-03 | design §3.4.2；R-I07 | 单元、PostgreSQL 并发 | IdempotencyRecord 状态/锁/响应快照 | `identity-auth-S5.1`、`identity-auth-S1.6` | 已登记 |
| I-CRED-03、I-CRED-05～06 | US-IDENTITY-03.10、03.11 / SR-IDENTITY-03 | design §3.3.11～12、§4.1；R-I01/R-I06/R-I07 | 权限、SMTP/TLS、故障注入 | generation、旧新凭据登录、TLS/uncertain 证据 | `identity-auth-S10.1`、`identity-auth-S11.1`、`identity-auth-S5.3` | 已登记 |
| I-LIST-01～03 | US-IDENTITY-03.5 / SR-IDENTITY-03 | design §3.3.8；R-I11 | 单元、契约、PostgreSQL | cursor 遍历、字段 allowlist、拒绝响应 | `identity-auth-S7.1`、`identity-auth-S1.5` | 已登记 |
| I-GOV-01～05 | US-IDENTITY-03.6 / SR-IDENTITY-03 | design §3.3.9、§3.4.3/6；R-I02/R-I03/R-I07 | 状态机、事务、跨模块 | User/Session/Audit 与 article 可见性结果 | `identity-auth-S8.1`、`identity-auth-S1.6` | 已登记 |
| I-DEL-01～07 | US-IDENTITY-03.7 / SR-IDENTITY-03 | design §3.3.10、§3.4.7/8；R-I03/R-I05/R-I10 | 跨模块集成、恢复、故障注入 | 墓碑/Identifier/Session/Credential/Article/Audit 快照 | `identity-auth-S9.1`、`identity-auth-S9.2`、`identity-auth-S4.4` | 已登记 |
| I-AUD-01～03 | 11 条 identity US / SR-IDENTITY-01～03 | design §3.4.2、§4.3、§5；R-I01/R-I09 | 契约、DB 权限、安全扫描 | AuditEvent、失败回滚、敏感字段扫描 | `identity-auth-S1.1`、`identity-auth-S1.5`、`identity-auth-S1.6` | 已登记 |
| I-CON-01～03 | 11 条 identity US / SR-IDENTITY-01～03 | design §3.3.1、§3.4.7/8；R-I11 | 单元、REST、内部契约 | error registry、JSON schema、S1–S16 fixture | `identity-auth-S1.5` | 已登记 |
| I-REC-01～05 | US-IDENTITY-03.1、03.7 / SR-IDENTITY-03；S13～S15 | design §4.2、§5.6、§6.1；R-I01/R-I04 | 真实恢复、持久卷、keyring | RecoveryRun/Journal/账号状态/readiness | `identity-auth-S4.2`、`identity-auth-S4.3`、`identity-auth-S4.4` | 已登记 |
| I-REC-06～10 | US-IDENTITY-03.7 / SR-IDENTITY-03；S16 | design §3.4.2/7/8、§4.2；R-I04 | 契约、跨模块恢复、崩溃 | Ready20/Unavailable6、digest、双 namespace、MarkConsumed | `identity-auth-S1.5`、`identity-auth-S4.2`、`identity-auth-S4.4` | 已登记 |
| I-PERF-01～04 | 11 条 identity US / SR-IDENTITY-01～03 | design §2.3、§4.4、§6.6；R-I12 | 性能/容量 | 基准报告、资源指标、批准阈值 | `identity-auth-S2.2` | 已登记 |

## 7. 数据与环境计划

| 条件 | 计划 |
|---|---|
| PII | 只使用合成用户名、邮箱、手机号、IP 和通知地址；禁止复制真实用户或生产凭据 |
| PostgreSQL | 每次测试使用隔离数据库或独立 schema；并发用至少两个独立连接/事务；清理由后续任务明确责任 |
| 时钟与随机 | 可控时钟覆盖 7 天临时凭据、Session、lease、退避和 RFC3339；随机源可注入但不得降低生产熵 |
| SMTP | 本地可控测试服务支持 implicit TLS、required STARTTLS、证书过期/错主机/不可信 CA、受理后断链和延迟故障注入 |
| 浏览器/同源 | 可观察 Cookie、Origin、Fetch Metadata、CORS、CSRF 和旧 Cookie 失效；当前无前端 profile，不虚构 UI 自动化框架 |
| 并发 | 至少双 worker/双请求方，覆盖唯一性、幂等、治理行锁、bootstrap 和会话轮换 |
| Recovery Journal | 独立持久卷；可注入只读、空间不足、部分写、fsync、链损坏和 namespace 混用 |
| keyring | active + 多个 verify-only 版本；覆盖完整迁移、缺 key、退役前引用和跨版本 secret 判重 |
| 备份恢复 | 含账号、角色、删除墓碑、Session、临时凭据、幂等记录和 article 删除前数据的合成样本 |
| tagged union | 固定 Ready20/Unavailable6 fixture，独立 canonical bytes 与 SHA-256 复算 |

数据、证书、备份、持久卷和清理脚本的准备工作已登记为 `identity-auth-S1.4`、`identity-auth-S4.3`、`article-management-S4.5`；实际运行与清理责任仍未分配。任何清理动作不得使用真实生产数据或未经确认的目录。

## 8. 自动化与人工边界

| 自动化 | 必须人工或独立责任确认 |
|---|---|
| validator、状态机、权限矩阵、错误映射、canonical digest、REST/internal contract | 实际地区、数据控制者、处理依据和 SMTP 处理方值是否由责任人登记 |
| PostgreSQL 事务、约束、行锁、并发和回滚 | 隐私声明文本、管理员声明文本和邮件正文是否满足产品/法律责任人的要求 |
| SMTP TLS 握手、证书失败、处理方门禁与残余窗口 | UI 是否需要二次确认及最终交互可用性 |
| 浏览器同源、Cookie、CSRF、CORS | 真实恢复运行的操作授权、RPO/RTO 接受和开放签核 |
| Journal/keyring 损坏、崩溃续跑、双 namespace readiness | 外部真实邮件收件与展示；本任务不执行 |

## 9. 任务建议

以下建议已由 task-planner 登记；实际状态只在 [tasks.md](../../tasks/tasks.md) 维护。

| 建议项 | 类型 | 建议所有权 | 依赖 | 关闭条件 | 停止条件 | 实际 task 引用 |
|---|---|---|---|---|---|---|
| 身份策略与状态机局部测试 | 开发 task 内 | identity 开发 | 对应实现切片 | validator、权限、状态机、错误映射随实现通过 | 合同与设计不一致时回设计 | `identity-auth-S1.3`、`identity-auth-S2.1`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S5.2`、`identity-auth-S6.1`、`identity-auth-S7.1`、`identity-auth-S8.1`、`identity-auth-S9.1`、`identity-auth-S10.1`、`identity-auth-S11.1` |
| 身份 REST 与内部端口契约测试 | 开发 task 内/共享专项 | identity + platform | REST schema、S1–S16 接口 | 11 US 的公共契约和内部端口 fixture 可复核 | schema 未形成或消费者未对齐 | `identity-auth-S1.5` |
| 身份 PostgreSQL 事务与并发专项 | 专项集成 | identity 测试/开发 | schema、migration、隔离库 | 唯一性、锁、审计、幂等、删除事务故障矩阵通过 | 无真实 PostgreSQL 或迁移 | `identity-auth-S1.6` |
| SMTP/TLS 与凭据残余窗口专项 | 专项安全/集成 | identity 测试 | 可控 SMTP、证书、故障注入 | 三类凭据动作的成功/失败/uncertain 证据齐全 | 不能控制 SMTP/TLS 或可能发外部邮件 | `identity-auth-S5.3` |
| 浏览器认证安全专项 | 专项安全 | 安全测试 | 可部署 HTTP 入口 | Cookie/CSRF/CORS/login CSRF/暴力矩阵通过 | canonical origin、代理信任或阈值缺失影响目标验收 | `identity-auth-S3.3` |
| canonical 用户删除跨模块专项 | 专项集成 | identity 主责、article 协作 | 双模块 schema/端口 | 零/多/违规/失败/kept_deleted 全矩阵通过 | 任一模块合同未实现 | `identity-auth-S9.2` |
| Recovery Journal/keyring 与联合恢复专项 | 专项韧性 | recovery/identity 主责 | 持久卷、备份、双 Journal、keyring | 崩溃/损坏/重放/双 namespace/MarkConsumed 顺序通过 | RPO/RTO、运行责任或真实备份环境未确认 | `identity-auth-S4.4` |
| identity 性能基准 | 专项性能 | 性能测试 | 目标环境与容量模型 | Argon2、Session、事务、恢复指标有基线和批准目标 | 目标值/工作负载未确认 | `identity-auth-S2.2` |
| identity 真实 E2E 旅程 | 独立 E2E | Test-Engineer + 业务验收人 | 已部署版本、环境、Case 审核 | 真实入口、后端数据流、数据符合性三层证据齐全 | 未部署、责任人/数据/邮件安排未确认 | `identity-auth-S3.4` |

## 10. 执行顺序

1. 公共错误、validator、状态机、权限矩阵、cursor、canonical digest 和 S1–S16 契约；
2. PostgreSQL schema/约束/事务/锁、Session、幂等和审计；
3. 跨模块停用/恢复、canonical 删除与 retained_email 生命周期；
4. SMTP/TLS、浏览器安全、暴力登录和故障注入；
5. Recovery Journal、keyring、ReplayStatus consumer、双 namespace 与备份恢复；
6. 容量参数明确后的性能基准；
7. 部署完成后的少量真实 E2E。

## 11. 执行前 Gate

| Gate | 状态 | 依据与继续条件 |
|---|---|---|
| 11 US 与关键验收可观察 | ready | requirements、design、stories 均有明确响应/状态/数据证据 |
| 高风险与安全红线有场景 | ready | 凭据、权限、隐私、删除、恢复、并发和审计均有 P0 场景 |
| S1–S16 与跨计划主责 | ready | §5 已明确主责和引用 |
| Ready20/Unavailable6 fixture | ready | schema、canonical 与固定 digest 已接受 |
| 代码、schema、migration | blocked | 当前不存在；形成后才能实施自动化 |
| 测试框架与项目检查命令 | blocked | Manifest `check_commands: []`；不得虚构命令 |
| PostgreSQL/SMTP/浏览器/持久卷环境 | blocked | 尚未提供或登记 |
| 合成数据、证书、备份样本和清理责任 | blocked | 已登记 `identity-auth-S1.4`、`identity-auth-S4.3`、`article-management-S4.5`；实际责任仍待分配 |
| 隐私部署实际值 | partial | 门禁行为可测；能力开放和真实 E2E 被阻塞 |
| 密码/Session/限流/SMTP/保留期目标值 | partial | 结构和失败语义可测；目标数值与性能验收被阻塞 |
| 初始管理员后续轮换与改密规则 | partial | 保护和首次 Gate 可规划；具体规则验收被阻塞 |
| RPO/RTO、key 数、Journal/tombstone/evidence retention | partial | fail-closed 合同可测；灾备容量和退役结论被阻塞 |
| 人工验收与 E2E 责任人 | blocked | E2E task 已登记为 `identity-auth-S3.4`；实际 Test-Engineer 与业务验收人仍未分配，不宣称可执行 |

## 12. 当前阶段结论

identity-auth 的稳定设计切片已形成风险驱动规划，覆盖 11 条主责 US、S1–S16、凭据/隐私/权限/删除/恢复高风险与跨模块责任。当前结论仅为 **test plan complete as planned, tests not executed**。由于代码、schema、测试命令、环境、数据和责任人均缺失，不能宣称可执行、通过或可进入真实 E2E。
