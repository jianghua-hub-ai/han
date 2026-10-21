# identity-auth 详细设计独立复审报告 r02

- 复审日期：2026-10-20
- objectType：design
- 原报告：`.spec/wip/identity-auth/review-design-r01.md`
- 原 findings：`IA-DES-R01-F001`～`IA-DES-R01-F010`
- 主对象：`identity-auth-design-ir01-r02`
- 附属对象：`identity-auth-stories-ir01-r02`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories、原报告或上游制品。

## 1. 固定对象与依据

### 1.1 开始/结束对象哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `178e25f312183bff1fd9d2a7b647b2b246145af72d929133224e2892249e8dc5` | `178e25f312183bff1fd9d2a7b647b2b246145af72d929133224e2892249e8dc5` | 一致 |
| `stories.md` | `1ca9f97dc865a081a18e7adfe6eb73b35e6cbd8e9796d576ab50362a022d4c31` | `1ca9f97dc865a081a18e7adfe6eb73b35e6cbd8e9796d576ab50362a022d4c31` | 一致 |

按确认段排除、尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:e377497d67645f07a0a9730eb585d0929ba48d2c7520c38efe8534680055c2a2`

与 Lead 提供值一致，无对象漂移。

### 1.2 上游与报告快照

| 依据 | 修订/状态 | 当前完整 SHA-256 |
|---|---|---|
| 原复审报告 | r01 | `83df815960a22073f2dc022afea5f8c94353d4040e0e7b2a80385f6cac17b914` |
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r05` accepted | `f16638d8c0698a5e85a5d0b29100bce7f473c653f4b64af462f4b7d27dba40ad` |
| identity boundary | `identity-auth-boundary-ir01-r04` accepted | `741740fcc98ec459824bbf5cb7454633f825f37216382702572a3ff60741199e` |
| article boundary | `article-management-boundary-ir01-r04` accepted | `8cd45916a2dc497f1eae1cd405f1d7f45d711c75a575c732c50dfdb34d51469f` |
| ADR index | ADR-0001～0018 | `492ecdfd3dfe8e37a4ee63cf45aac1dce43e780b0b551ef42c02ec911d27812e` |
| framework ADR | 含 ADR-0017/0018 | `78b77ceb50396801b64e9cc9e59a7ff3addaec640b6cb16dd4a3841d5d8356ae` |
| identity ADR | 修订 ADR-0005/0006 | `fc8e41583514785627e89ebdbefd0773ba6ed9ea517527b17180798cf6ce12c3` |
| content ADR | ADR-0010 | `f30ea02f15b313a95c928b9e278fc0b2567f6059de1b99f18fa8b36341840004` |

requirements r16 checkpoint 为 `decision-e3af3912166f7287e8324488cad1cf1b87ba1034fca3378d3e0467a23751592d`；architecture r05 checkpoint 为 `decision-cf4c96a92bdeb8c684308a6fe5432046b4115fdab5d0bd16ccaea7d5b829edbc`。

## 2. 两轮复审范围

### Round 1：原 finding 定点复审

逐项重新执行 F001～F010 的原触发场景，核对作者声明的修复位置、上游决定、错误/状态/数据合同和验证入口，不以整改摘要代替正文。

### Round 2：整改回归审查

沿 S1～S13、11 条 identity-owned US、全部 API、数据表、事务、恢复、隐私门禁、安全传输和 Story DAG 检查新增风险，重点覆盖：

- `credential_mode`/TemporaryCredential 状态机；
- 数据库恢复、幂等恢复与永久凭据；
- SMTP TLS、same-origin/login CSRF、防暴力；
- 公共错误、分页、零文章清理；
- PrivacyNotice/Acknowledgement 双 Gate 版本一致性；
- stories 共享内核和 DAG。

## 3. 原 findings 逐项复审

### IA-DES-R01-F001 — closed

- 修复证据：`design.md:643-648`、`687-756`、`914-989`、`1359`、`1699-1724`；requirements r16 和 ADR-0017 已接受版本化声明、管理员信息提供声明、双 Gate、事务性用途、ProcessingGate 与撤回复用删除。
- 原触发结果：未确认隐私声明不能注册；管理员未声明不能创建目标管理员；实际地区、控制者、处理依据或 SMTP 处理方缺失时相关能力 fail closed。
- 回归：未新增运营治理权限或营销邮件入口。
- 限制：实际 Q9 值、声明正文、隐私/法务评审和运行证据尚不存在，但设计缺陷本身已关闭。

### IA-DES-R01-F002 — closed

- 修复证据：`design.md:630-633`、`834-840`、`1238-1259`、`1303`、`1416-1430`。
- 原触发结果：`credential_mode=temporary` 时只走 TemporaryCredential；active 之外的 expired、superseded、recovery_invalidated 均失败，不回退永久密码。改密绑定当前 Session 的 credential kind/id。
- 回归：bootstrap 被单独建模，不与临时凭据回退混用。

### IA-DES-R01-F003 — still-open

- 已修部分：`design.md:1534-1573` 和 ADR-0018 已固定停服、删除全部 Session、失效当前 TemporaryCredential、收敛不确定幂等并追加恢复审计。
- 未关闭触发：从早于一次永久密码改密、账号停用/删除或角色撤销的备份恢复时，快照中的 `users.status/role/permanent_password_digest` 会恢复为旧值。RecoveryGate 只处理 Session、TemporaryCredential 和部分 IdempotencyRecord，没有使恢复点中的永久密码或恢复后的旧账号/角色失效。
- 现实场景：用户在备份后把永久密码 A 改为 B，随后恢复旧备份；恢复流程删除 Session，但 User 仍为 `credential_mode=permanent` 且摘要为 A，密码 A 可重新登录。被删除账号同理可能以恢复点中的 active 状态重新出现。
- 影响：原 finding 中“旧密码、已删除/撤权身份状态复活”的 HIGH 风险仍存在；`design.md:1530` 声称“旧权限/凭据”已统一规避，证据不足。
- 复审要求：不要求重开“批量失效 vs epoch”的已接受选择，但 ADR-0018/设计必须明确处理永久凭据及恢复点中的账号/角色回退，例如恢复后强制所有账号进入不可登录的 recovery-reset Gate，或给出经责任人明确接受的剩余 HIGH 风险及其安全边界。

### IA-DES-R01-F004 — closed

- 修复证据：`design.md:659-681`、`1368-1376`、`1661-1667`、`1697`；修订 ADR-0006。
- 原触发结果：仅允许 implicit TLS 或 required STARTTLS；证书链、有效期和主机名必须校验；禁止明文、机会式降级和 `InsecureSkipVerify`；处理方或安全配置缺失时 fail closed。
- 回归：测试矩阵包含无 TLS、降级、错误证书/主机名及处理方缺失。

### IA-DES-R01-F005 — closed

- 修复证据：`design.md:1157-1208`、`1482-1483`。
- 原触发结果：`OrphanArticles` 返回 pending 数与清理事实；零文章时在删除事务内同步清理 `retained_email` 和 email Identifier，并追加审计。
- 回归：多文章场景仍由最后处置事实触发清理，phone 清理继续保持 Q2，不被越权补定。

### IA-DES-R01-F006 — closed

- 修复证据：`design.md:539`、`611-615`、`793-866`；修订 ADR-0005。
- 原触发结果：浏览器 API same-origin；禁止跨源 credentialed CORS；登录校验 canonical Origin/Fetch Metadata；已有会话再次登录成功时撤销旧会话。
- 回归：登录失败保持旧会话，未引入失败即登出的额外副作用。

### IA-DES-R01-F007 — closed

- 修复证据：`design.md:650-657`、`866`、`1338`、`1583-1595`。
- 原触发结果：账号摘要、可信来源及组合三维均有持久窗口和退避；代理信任、统一登录失败、429 边界、审计和低基数指标均已定义，数值继续配置化 TBD。
- 回归发现：具体持久化结构仍有新问题，见新 finding `IA-DES-R02-N002`；该问题不改变原 finding 的机制缺失已被补齐。

### IA-DES-R01-F008 — still-open

- 已修部分：`design.md:575-605` 已补齐保护、限流、依赖、门禁和恢复 code，正文未发现未注册的公开错误 code。
- 未关闭点：表内 `platform.processing_gate_closed` 的日志级别为“ERROR 启动或 WARN 请求”，`identity.email_delivery_failed` 为“ERROR/WARN”，`platform.recovery_in_progress` 为“INFO/WARN”；但 `design.md:607` 同时声明同一 code 只有一个日志级别。
- 影响：实现者仍需自行决定最终失败的责任层日志级别，违反原 finding 要求的稳定映射，可能产生漏告警或重复告警。
- 最小修复：把“请求公共错误映射”和“启动/中间重试事件日志”分开；每个最终公共 code 固定一个责任边界日志级别，启动或重试过程使用独立事件名而不是斜杠备选。

### IA-DES-R01-F009 — closed

- 修复证据：`design.md:1064-1101`。
- 原触发结果：列表 envelope、`next_cursor`、`has_more`、排序、版本化 HMAC 游标、过滤/角色/limit 绑定、非法游标、末页/空页及 null/省略语义均已固定。
- 回归：明确不承诺跨请求快照，未伪造一致性保证。

### IA-DES-R01-F010 — closed

- 修复证据：`stories.md:13-80` 及各 Story `依赖`。
- 原触发结果：K1/K2/K3 表达共享实现前置；S06、S10 等依赖图与正文一致。
- 静态验证：41 条边，未检测到环。

## 4. 整改新增 findings

### [MEDIUM] IA-DES-R02-N001：新管理员先完成隐私 Gate 后，声明换版可绕过“当前版本”要求

- 位置：`design.md:646`、`985-989`、`1335-1340`、`1432-1439`
- 违反依据：requirements r16 要求新管理员完成“当前隐私声明确认”；ADR-0017 双 Gate。
- 现状：确认 current 版本后立即把 `onboarding_privacy_required` 永久清为 false。若新管理员先确认 v1，尚未改密时发布 v2，之后完成改密即可开放管理员能力，不会重新校验其确认版本仍是 current。
- 触发：隐私 Gate 先完成，两个 Gate 之间 PrivacyNotice 换版。
- 影响：新管理员未确认开放权限时的当前声明，双 Gate 证据与已接受需求不一致。
- 最小修复：Gate 不只看布尔值；保存/派生 onboarding acknowledged version，并在最终开放时要求其等于 current version。版本变化时只重新锁定仍处于 onboarding 的账号，不重开已完成 onboarding 用户的已接受权衡。
- 验证：覆盖“隐私先完成→声明换版→改密”和“改密先完成→声明换版→确认”两种顺序。

### [MEDIUM] IA-DES-R02-N002：LoginAttemptBucket 表结构不能明确表达三类限流桶

- 位置：`design.md:652-657`、`1338`、`1350`、`1587-1595`
- 违反依据：F007 修复合同要求账号、来源、账号+来源三个独立维度，并发下状态唯一。
- 现状：表只定义 `subject_digest,source_digest` 及二元唯一约束，没有 `bucket_kind` 或明确非空哨兵规则。账号-only/source-only 桶若使用 NULL，PostgreSQL 普通唯一约束允许多条含 NULL 记录；若不使用 NULL，则无法从当前合同判断如何编码独立维度。
- 触发：并发失败尝试创建账号-only 或来源-only 桶。
- 影响：可能产生重复桶、计数分裂或维度互相覆盖，使防暴力策略可被绕过。
- 最小修复：采用 `bucket_kind + bucket_key_digest` 的单一唯一键，或明确所有字段 NOT NULL、各维度规范编码和 `NULLS NOT DISTINCT` 约束；同时固定原子 upsert/锁语义。
- 验证：并发创建三类桶、重复失败累加、进程重启和清理/衰减测试。

### [MEDIUM] IA-DES-R02-N003：恢复后幂等新状态没有请求重放合同

- 位置：`design.md:640`、`1357`、`1540`、`1563`
- 违反依据：ADR-0015 同键重放合同；ADR-0018 要求恢复后由安全重试产生新结论。
- 现状：正常状态只列 `processing/succeeded/failed/uncertain`，恢复又引入 `review_required/failed_recovery`，但数据约束、同键请求遇到这些状态的返回/抢占/重执行规则和公开错误均未定义。
- 触发：备份恢复后，客户端使用原 Idempotency-Key 重试一条原 processing/uncertain 请求。
- 影响：实现可能永久拒绝、错误重放旧结果，或无原子状态转换地重复发送临时凭据邮件。
- 最小修复：把恢复状态加入正式状态机和数据库检查约束；固定同键同摘要时由谁、在什么租约/审核条件下转回 processing，是否返回 `identity.operation_uncertain`，以及同键不同摘要仍如何冲突。
- 验证：恢复前 processing/uncertain、恢复后同键同/不同摘要、并发重试和 SMTP 残余窗口回归。

## 5. 静态验证与限制

- design artifact digest 与 Lead 值一致。
- PlantUML：design 22 块、stories 1 块；每块均有一个 `@start*`/`@end*`，未发现禁用 `\n` 或 `linetype ortho`。
- design 中 12 个本地链接均存在。
- stories DAG 共 41 条边，未发现环。
- 未发现唯一公共错误表之外的公开错误 code；审计 action 已与公共错误区分。
- 本机仍无 PlantUML CLI/JAR，未做 `-checkonly` 或 PNG 实渲染。
- 当前仍无业务代码、数据库 schema、迁移、API schema 或测试；没有运行不存在的构建、lint、单元、集成或 E2E。
- Q1、Q2、Q3/Q4 数值、Q8/Q9 实际部署输入和项目检查命令仍按上游保持待决定/待登记；本复审不将已接受 TBD 重新判为缺陷。

## 6. 最终结论与后续条件

**最终结论：NOT PASS。**

- 原 findings：8 closed，2 still-open；已关闭项的整改范围内另发现独立新增回归风险。
- 未关闭原项：`IA-DES-R01-F003`（HIGH）、`IA-DES-R01-F008`（MEDIUM）。
- 新增风险：3 个 MEDIUM（`IA-DES-R02-N001`～`N003`）。
- 未发现新增 CRITICAL。

design 保持 `draft` 正确；在 F003 关闭前不得转为 `implementable`。复审条件：

1. 提供新的 design/stories 精确 revision、完整 SHA-256 和规范 artifact digest；
2. 逐项回应 F003、F008、N001～N003；
3. 对恢复后的永久凭据/账号状态、双 Gate 换版、三维限流桶和恢复幂等状态提供更新后的验证设计；
4. 若修订 ADR-0018 或其他已接受上游，附新的确认对象和 S1～S13 消费者回归。

本结论只针对上述固定 r02 对象，不代替用户/Lead 的风险接受、设计确认或后续实施授权。
