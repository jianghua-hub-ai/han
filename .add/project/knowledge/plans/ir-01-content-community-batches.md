# 批次 Plan: IR-01 内容发布社区完整实施地图

> 本 Plan 只消费 [tasks.md](../tasks/tasks.md) 中的 47 个原任务，其中 B0 为用户直接授权的项目启动初始任务，其他 46 个由 Story 派生。任务状态只在原 SSOT 维护；本文件不认领、不创建 task、不授予实施或发布权限。

## 背景

- **需求/任务来源**：[requirements r16](../requirements/requirements.md)、[tasks.md](../tasks/tasks.md)。
- **设计依据**：[architecture r13](../design/architecture.md)，accepted checkpoint `decision-5e55881bdc4a7e038b3764d91ff0dfd906a48fe121e93bbb0813ca58173cae40`；[identity boundary r11](../design/identity-auth/boundary.md)、[article boundary r12](../design/article-management/boundary.md)、[identity design r09](../design-workspace/identity-auth/design.md)、[article design r06](../design-workspace/article-management/design.md)、两份 stories/test-plan 与 [ADR-0001～0021](../decisions/README.md)。
- **当前活动**：development-delivery / plan；Step A G 模式 task-decompose 已登记原 46 个 Story task，B0 由用户直接授权登记；Step B 仅消费 47 个原 task。
- **涉及仓库/工作区**：`D:/AID/VSCode/Codex/Han`；固定 `main`。
- **工作分支/修订**：`main`，当前 HEAD `e553ee23d6500d5d5c568803dee8379c57a4203c`；初始提交外 ADD 成果为未跟踪内容，不要求全树干净。
- **预期收益**：1 个正式 B0 项目启动任务与 19 个 Story 的实现、专项验证和部署后 E2E 都有唯一 task、明确依赖与完整批次。
- **产出负责人 / 接收负责人**：Task-Planner 负责本次规划内容；Lead 现场复核。实际实施、独立 Reviewer、Test-Engineer、部署和业务验收责任均未分配。
- **确认来源**：需求与架构 checkpoint accepted；identity/article 最终设计 checkpoint 的设计负责人 accepted，但承接 Gate 均为 `deferred:reorder`。测试计划为 planned-not-executed，执行 Gate 存在 blocked/partial。
- **Git 约定**：本轮仅落未提交 diff；禁止 add/commit/push/PR/merge、切分支、清理或覆盖他人改动。

## 精确输入与读取依据

| 输入/引用 | 适用范围与修订 | 实际读取或有效复用 / 约束 / 缺项 |
| --- | --- | --- |
| requirements / architecture / boundaries / ADR | IR-01 最终稳定业务与跨模块合同；architecture r13 / ADR-0021 为 B0 新基线 | 已完整读取；architecture accepted checkpoint `decision-5e55881bdc4a7e038b3764d91ff0dfd906a48fe121e93bbb0813ca58173cae40` |
| identity design r09 / stories r09 / test-plan r01 | 11 Story、60 场景 | checkpoint object digest `sha256:df484400154f7b64c00c40842e02eb254408bdb1c764768c51bf9672dbd09c61`；实施 Gate deferred |
| article design r06 / stories r06 / test-plan r01 | 8 Story、66 场景 | checkpoint object digest `sha256:ed1a13b7f3d2e9f2c63b47d66d26469ea19315c42ace9cbe49db9f206880e4e0`；实施 Gate deferred |
| Task-Planner Role、task-decompose、batch-plan、模板、delivery plan 合同、Rule/Profile/Git | 本 plan 活动 | 已完整读取；Manifest `check_commands: []`，不得声称项目命令已登记 |
| 仓库事实 | G 模式 | 无业务代码、go.mod、schema、migration、测试框架；不做 gap-map |

## 正式 B0 项目启动 Gate

原外部别名已由 [tasks.md](../tasks/tasks.md) 中唯一原任务 `project-bootstrap-B0.1` 承接。该任务直接来自用户的项目启动授权，不经 task-decompose 从 Story 派生；它建立 Go 1.27/module `han`、已批准依赖锁定与审查、最小 `han-server`、ADR-0021 迁移执行器、测试标签/环境约定及 Manifest 检查命令登记。B0 完成前 B1～B12 全部暂停。

## 批次外层表

| 批次 | 原任务 ID/引用 | 类型/形态收窄 | 启动与批间 Gate |
| --- | --- | --- | --- |
| B0 项目启动基线 | `project-bootstrap-B0.1` | 项目启动 | 完成 Go/module、依赖审查、最小入口、迁移执行器、测试约定与 Manifest 检查命令登记；未满足时所有 B1～B12 暂停。 |
| B1 共享平台与审计根 | `identity-auth-S1.1` | 开发（共享基础，identity-auth 所有） | 共享 REST/配置/隐私/审计与首个 migration 可复核；独立 Review 通过。 |
| B2 身份数据、运行输入与密码基础 | `identity-auth-S1.2`<br>`identity-auth-S1.4`<br>`identity-auth-S2.1` | 开发、运行输入准备 | 三项文件互斥，可并行；身份 schema、fail-closed 输入和 Argon2id 底座通过局部验证。 |
| B3 运行时共享基础与首批数据落点 | `identity-auth-S1.3`<br>`identity-auth-S3.1`<br>`identity-auth-S4.2`<br>`identity-auth-S5.1`<br>`article-management-S1.1` | 开发、开发（共享接入）、开发（共享恢复基础）、开发（共享凭据基础） | 文件互斥，可并行；注册、接入、恢复、凭据/邮件和 article schema 各自完成 Review。 |
| B4 登录、恢复样本与 article 公共策略 | `identity-auth-S3.2`<br>`identity-auth-S4.3`<br>`article-management-S1.2` | 开发、运行输入准备、开发（共享 article 基础） | 文件互斥，可并行；统一登录失败、合成恢复输入、文章状态/cursor/ETag 合同可消费。 |
| B5A 初始管理员、浏览器安全与文章创建 | `identity-auth-S3.3`<br>`identity-auth-S4.1`<br>`article-management-S1.3` | 专项安全测试、开发 | 文件互斥，可并行；Q1 只阻塞初始管理员具体改密断言，不阻塞浏览器安全与文章创建。 |
| B5B 查询与 canonical article 端口 | `identity-auth-S7.1`<br>`article-management-S6.1` | 开发、开发（article 共享接缝，Story 6 所有） | 文件互斥，可并行；用户查询与 OrphanArticlesCanonical 端口完成，为治理/删除解锁。 |
| B6A 管理员创建与文章永久删除核心 | `identity-auth-S5.2`<br>`article-management-S4.1` | 开发 | 跨模块文件互斥，可并行；SMTP 未受理不留管理员，删除双完成协议可收敛。 |
| B6B 身份停用/恢复 | `identity-auth-S8.1` | 开发 | 串行占有 identity handler/服务文件；完成作者状态 seam。 |
| B6C 密码重置 | `identity-auth-S10.1` | 开发 | 串行占有 identity handler；完成分级重置与幂等/SMTP 局部闭环。 |
| B6D 临时凭据重发 | `identity-auth-S11.1` | 开发 | 串行占有 identity handler；完成 generation 替换与 purpose 授权。 |
| B7 删除/恢复纵切与 article 读取 | `identity-auth-S5.3`<br>`identity-auth-S9.1`<br>`article-management-S2.1`<br>`article-management-S4.2`<br>`article-management-S4.3`<br>`article-management-S4.5` | 专项安全/集成测试、开发（跨模块协调）、开发、专项韧性测试、运行输入准备 | 文件互斥，可并行；canonical 用户删除、成员读取、replay producer、Deletion Journal 专项与运行输入具备。 |
| B8A 高级治理与恢复联调前置 | `identity-auth-S1.5`<br>`identity-auth-S6.1`<br>`article-management-S3.1`<br>`article-management-S4.4` | 专项契约测试、开发、专项契约/韧性测试 | 文件互斥，可并行；身份契约、管理员治理、作者修改与旧备份 replay 通过各自 Gate。 |
| B8B 无归属处置 | `article-management-S6.2` | 开发 | 串行占有 article command/handler；完成公开、转移、永久删除与清理事实。 |
| B9A 跨模块删除与通知主链 | `identity-auth-S1.6`<br>`identity-auth-S9.2`<br>`article-management-S5.1`<br>`article-management-S7.1` | 专项集成测试、专项集成测试（identity 主责）、开发 | 测试目录与 article 实现文件互斥，可并行；真实 PostgreSQL 删除矩阵、普通管理员删除和通知 dispatcher 完成。 |
| B9B 违规治理 | `article-management-S8.1` | 开发 | 串行占有 article command/handler/policy；完成违规判定、解除和删除。 |
| B10 契约、安全与韧性专项 | `identity-auth-S4.4`<br>`article-management-S1.5`<br>`article-management-S2.2`<br>`article-management-S7.2` | 专项韧性测试、专项契约测试、专项集成测试、专项集成/安全测试 | 专项测试文件互斥，可并行；真实 PostgreSQL/SMTP/持久卷/备份环境缺失则对应任务不关闭。 |
| B11 性能与容量基线 | `identity-auth-S2.2`<br>`article-management-S2.3` | 专项性能测试 | 可并行采集基线；批准目标、容量、RPO/RTO/retention 未确认时不宣称性能 Gate 通过。 |
| B12 部署后真实 E2E | `identity-auth-S3.4`<br>`article-management-S1.4` | E2E | 部署、环境、Case 与责任人就绪后执行；数据隔离时可并行，否则按共享环境串行。 |

- **全套批次与 DAG**：`B0 → B1 → B2 → B3 → B4 → {B5A ∥ B5B} → B6A → B6B → B6C → B6D → B7 → B8A → B8B → B9A → B9B → B10 → B11 → B12`。大括号内是批间可并行分支；批内并行仍以文件零交叉、前置完成和独立环境为准。
- **R 待验对账**：不适用；当前为 G 模式且无业务代码。
- **未安排范围**：无。tasks.md 中 47 个原 task 各安排一次：`project-bootstrap-B0.1` 进入 B0，原 46 个 Story task 进入 B1～B12。
- **范围确认**：规划完成不等于实施授权、独立评审通过、测试执行、E2E 接纳或发布许可。

## 依赖图与关键链

```text
project-bootstrap-B0.1
  → identity-auth-S1.1
  → {identity-auth-S1.2 | identity-auth-S1.4 | identity-auth-S2.1}
  → {identity-auth-S3.1 | identity-auth-S4.2 | identity-auth-S5.1 | article-management-S1.1}
  → identity-auth-S3.2 → identity-auth-S7.1 → identity-auth-S8.1
  → article-management-S2.1 → article-management-S3.1

article-management-S1.1 → article-management-S1.2 → article-management-S1.3 → article-management-S4.1
article-management-S6.1 → identity-auth-S9.1 → article-management-S6.2 → identity-auth-S9.2
article-management-S4.1 → article-management-S4.2 → article-management-S4.4 → identity-auth-S4.4
article-management-S6.2 → article-management-S7.1 → article-management-S7.2

{全部开发/契约/集成/安全/韧性专项}
  → {identity-auth-S2.2 | article-management-S2.3}
  → {identity-auth-S3.4 | article-management-S1.4}
```

## 文件所有权与并行边界

| 所有权/冲突面 | 规则 |
| --- | --- |
| `internal/platform`、`internal/privacy`、`internal/audit` | `identity-auth-S1.1` 唯一共享 owner；article 只消费 |
| `internal/identity/handler.go`、`service.go` | identity 纵切任务必须按 B4→B5A/B5B→B6A→B6B→B6C→B6D→B7→B8A 串行，不并发写 |
| `internal/article/handler.go`、`command_service.go`、`policy.go` | article 创建/修改/删除/无归属/违规任务按批次串行；同层冲突已拆批 |
| canonical 用户删除 | identity 编排文件归 `identity-auth-S9.1`；article 端口归 `article-management-S6.1`；真实事务专项归 `identity-auth-S9.2` |
| ReplayStatus/恢复 | identity consumer/Recovery Journal 与 article producer/Deletion Journal 分属不同目录；通过 fixture/端口依赖串行对齐，不共享实现文件 |
| 专项测试与 E2E | 各自独立测试目录；若共用同一数据库、SMTP、持久卷或部署环境，执行时按环境租约串行 |

规划只能降低当前可见冲突，不能充当跨会话实时锁。发现未知写者或文件范围变化时暂停受影响批次并协调，不覆盖。

## 开发闭环与 E2E 分界

- 每个开发 task 在本卡内先建立适用失败证据、完成最小实现、运行局部 Go 测试并取得非作者独立 Review；满足自身条件即可由指定关闭责任更新原 task，不等待需求整体 E2E。
- PostgreSQL、SMTP/TLS、浏览器、双 worker、Recovery Journal、Article Deletion Journal、旧备份恢复、安全和性能由独立专项 task 承接。
- B12 只在可用部署版本、真实环境、Case 审核、数据与责任人齐备后执行；部署成功不能代替 E2E。

## 批次 Gate

1. **B0→B1**：原任务 `project-bootstrap-B0.1` 已按自身关闭条件完成；Go 1.27/module `han`、依赖锁定与审查、迁移执行器、测试标签/环境约定及 Manifest 检查命令可核。
2. **每个开发批次**：批内 task 各自局部测试通过、独立 Review/复审完成，Lead 核固定工作区的 status/diff/所有权；不得用整体验收替代。
3. **跨模块 Gate**：S1–S16 端口 fixture、canonical 删除事务、ReplayStatus strict variant 与共享 digest 无漂移。
4. **专项 Gate**：真实 PostgreSQL/SMTP/浏览器/持久卷/备份环境存在；未运行不算通过。
5. **性能 Gate**：先采集基线；只有用户/责任人补齐容量、目标值、RPO/RTO 与 retention 后才能判目标通过。
6. **E2E Gate**：部署版本、Case、人工/业务验收责任人与真实输入齐备；失败进入 Issue，不自动把全部开发 task 回退。

## 阻塞与待协调

| 范围 | 当前影响 | 继续条件 |
| --- | --- | --- |
| 项目启动骨架 | 阻塞所有实施/测试批次 | 完成原任务 `project-bootstrap-B0.1`；无真实 PostgreSQL 18 集成证据时不得关闭 |
| 设计 checkpoint `deferred:reorder` | 规划可完成，不构成实施承接授权 | 用户/Lead 明确实施批次授权 |
| Q1 初始 admin 改密/轮换 | 阻塞 `identity-auth-S4.1` 受影响断言 | 产品/安全责任人拍板规则 |
| AM-Q1/Q2、会话/限流/SMTP/连接池等 | 不阻塞结构与正确性；阻塞最终 validator/容量/性能结论 | 提供工作负载与批准目标 |
| 隐私部署值/SMTP 处理方 | 缺值行为可测；真实能力和 E2E fail closed | 实际责任人登记地区、控制者、依据、origin、代理与处理方 |
| RPO/RTO、key/Journal/tombstone/evidence retention | fail-closed 合同可实现；真实恢复、退役和性能结论阻塞 | 灾备/运行责任人提供批准值与证据 |
| PostgreSQL/SMTP/浏览器/双 worker/持久卷/备份环境 | 专项任务 blocked | 环境、合成数据、证书、inventory 与清理责任就绪 |
| 实施/Review/Test/E2E 责任 | 尚未分配 | 每批启动前真实分配，不能从 Role 名推断 |

## 批次整体核对与交回

- **跨任务依赖/契约**：以 tasks.md 的 requires 和本 Plan DAG 为准；批次发现漏项必须回 Step A 补 task，不在 Plan 新建。
- **适用集成/性能/覆盖测试**：当前均未执行；命令只在原 task 中记录为实现后命令。
- **G4-batch 整体评审**：尚未安排；不能以多个 task Review 通过代替。
- **整体接纳与未完事项**：Task-Planner 只负责规划内容；Lead 现场复核后仍需完成 B0、取得后续实施授权并分配实际责任。
- **真实 E2E**：引用 `identity-auth-S3.4`、`article-management-S1.4`；当前未部署、未执行。
- **任务/日期对账**：以原 tasks SSOT 为准；本 Plan 不复制状态或日期。
- **集成/发布交接**：未授权 Git add/commit/push/PR/merge/发布。

## 风险与注意事项

- migration 路径已在原 Story task 中固定为 `migrations/0001～0003`；执行工具与 PostgreSQL major 18 由 `project-bootstrap-B0.1`、architecture r13 和 ADR-0021 统一约束，不得由实现者另造第二套迁移机制。
- 共享 `handler.go`/`service.go` 文件是主要并行冲突面；批次已串行化，执行者不得跨卡顺手重构。
- 设计 checkpoint 的对象摘要与当前整文件 SHA-256 语义不同；复核时以 checkpoint `objectVersion.digest` 识别接受对象，以当前文件 hash 识别交付载体。
- 凭据、PII、真实邮件和生产数据不得进入仓库、测试 fixture、日志或审计。

## 变更记录

- 2026-10-20：Task-Planner 以 G 模式登记 46 个 task，完成全套批次、DAG、Gate 与 test-plan task 引用回填；未实施、未认领、未执行测试。
- 2026-10-21：按用户项目启动授权新增唯一原任务 `project-bootstrap-B0.1`，总数更新为 47；原 46 个 Story task 仅将外部前置别名对账为 B0，批次状态仍只在 tasks SSOT。
