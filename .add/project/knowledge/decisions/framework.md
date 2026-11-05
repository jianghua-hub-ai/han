# 框架与技术栈领域决策

本文件承载框架、语言、协议与工程结构类决策。领域边界由项目架构定义。

---

## ADR-0001: 采用模块化单体而非多服务 (2026-10-19)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对 D1 当面拍板（2026-10-19）

**问题**: 首期包含身份认证与文章管理两类业务，两者之间存在「用户状态变更连带影响文章可见性与归属」的强耦合。服务边界如何划分才能让这些不变量可被直接保证？

**决策**: 采用模块化单体——单一部署单元内按业务模块分包，模块间只经显式接口调用，共享同一数据库与进程内事务边界。

### 现状 / 痛点 / 改造 / 效果

业界起步阶段的业务系统主流走三条路：模块化单体（Shopify、Basecamp、GitHub 早期）、按业务域拆分的少数服务、细粒度微服务。三条路的分界不在技术，而在是否具备独立的组织与容量前提。

需求 `US-IDENTITY-03.6` 与 `US-IDENTITY-03.7` 要求用户被停用或删除时，其文章可见性与归属同步变化，且停用可逆、删除不可逆。若拆为两个服务，这条不变量跨服务边界，需要分布式事务或最终一致补偿，而补偿窗口内会出现「用户已停用但文章仍可见」的不一致状态，直接违反需求的「立即」语义。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 模块化单体 | 单进程按 `internal/identity` 与 `internal/article` 分包，共享数据库 | 跨模块不变量由本地事务保证；调试与 E2E 简单；无跨服务失败补偿 | 单点部署；模块纪律靠评审维持，编译器不阻止越界引用 |
| 多服务（两个服务） | 各服务独立进程与库，经 HTTP 或 RPC 通信 | 可独立伸缩与部署；故障隔离 | 需分布式事务或补偿；首期引入网络失败、超时、重试与幂等设计，收益为零 |
| 细粒度微服务 | 按更小职责继续拆分 | 组织并行度高 | 首期无独立团队与独立容量特征，不成立 |

**场景特征匹配**：本项目首期两个业务模块、单团队、单部署目标，且存在跨模块的强一致不变量。模块化单体的优势（本地事务、无补偿）正对应这些特征。

**劣势可接受性**：单点部署在首期无可用性等级要求，可接受。模块纪律靠评审维持的劣势，由 ADR-0008 的按业务模块分包与数据所有权表（架构文档 §6.4.4）缓解——越界引用在目录结构与所有权表面前可见。未来若出现独立伸缩需求，按架构原则 §7 走独立 ADR 与迁移任务，不在本设计预留两套实现。

### 约束

- 跨模块调用只允许 `service` 之间的显式接口调用；不允许 handler、repository 或数据表互相直接引用。
- 数据写入遵守架构文档 §6.4.4 的数据所有权表：账号状态与会话由身份认证模块写，文章可见性与归属由文章管理模块写。
- 新增业务模块时新增 `internal/<module>` 目录，不得在既有模块内堆积无关职责。

### 引用记录

- 暂无获授权登记的实质采用。

---

## ADR-0002: HTTP 框架采用 Go 标准库 net/http (2026-10-19)

**目的**: 接口设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对 D2 当面拍板（2026-10-19）

**问题**: 对外 HTTP 接口需要路由、中间件与请求解析能力。是引入第三方 Web 框架，还是使用标准库？

**决策**: 使用 Go 标准库 `net/http`，依赖 Go 1.22 起提供的方法路由与路径参数能力。

### 现状 / 痛点 / 改造 / 效果

Go 社区在此有四条成熟路径：标准库、Chi、Gin、Echo。Go 1.22 把方法路由（`GET /path/{id}`）并入标准库 `ServeMux`，直接压缩了第三方路由库的核心价值空间——路由能力不再是引入框架的充分理由。

剩余差异集中在参数绑定、校验与统一错误响应。本项目对外接口约 18 个（见架构文档 §6.2.3），其中校验规则集中且数量有限（口令规则、三字段唯一性、文章非空），自建薄封装的实际代码量小于引入框架后仍需编写的业务适配量。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 标准库 net/http | Go 1.22+ ServeMux 方法模式 + http.Handler 中间件链 | 零第三方依赖；无版本升级面；context 与超时语义直通 | 需自建参数绑定、校验与统一 JSON 错误响应 |
| Chi | 标准库兼容的 http.Handler 路由器 | 与标准库同语义；中间件生态成熟 | 引入一个依赖；功能集小于 Gin |
| Gin | 自有 Context + 绑定/校验/渲染 | 开发速度快；社区资料多 | Context 非 http.Handler，中间件不通用；依赖面最大 |
| Echo | 类似 Gin 的一体化框架 | 功能完整；文档好 | 与 Gin 相比无明显差异优势，依赖面同样较大 |

**场景特征匹配**：接口数量有限、校验规则集中、团队无既有框架偏好。标准库的零依赖与语义直通对应该场景。

**劣势可接受性**：需自建绑定与错误响应层，但在 18 个接口的规模下该层是一次性薄封装。若后续接口数量显著增长，或出现标准库难以优雅表达的校验需求，届时按技术选型合规规范提出变更请求，改选 Chi（保持 `http.Handler` 语义，迁移成本最低）。

### 约束

- 路由使用 `ServeMux` 的方法模式；中间件一律实现 `http.Handler` 接口。
- 统一 JSON 错误响应由 `internal/platform/errors.go` 提供，各模块不得自定义响应格式。
- 引入任何第三方 HTTP 相关库前须按技术选型合规规范提交变更请求。

### 引用记录

- 暂无获授权登记的实质采用。

---

## ADR-0003: 数据库采用 PostgreSQL (2026-10-19)

**目的**: 数据模型 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对 D3 当面拍板（2026-10-19）

**问题**: 需求要求用户名、邮箱、手机号三者分别唯一，并要求跨表状态联动（停用用户连带文章隐藏、删除用户连带文章归属置空）。数据库需要何种约束与事务能力？

**决策**: 采用 PostgreSQL 作为唯一持久化存储。

### 现状 / 痛点 / 改造 / 效果

需求 `US-IDENTITY-01.1` 要求三字段分别唯一且拒绝重复注册，`US-IDENTITY-03.6` 与 `US-IDENTITY-03.7` 要求用户状态变更与文章状态变更同步且即时。这两类要求都指向同一件事：不变量必须由数据库层兜底，不能只靠应用层查重——应用层查重在并发下存在竞态窗口。

主流候选为 PostgreSQL、MySQL 与 SQLite。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| PostgreSQL | 关系型，强约束与事务 | 唯一约束、检查约束、部分唯一索引直接表达业务规则；并发控制成熟 | 需独立部署与运维 |
| MySQL | 关系型，广泛部署 | 运维人才多；云托管普遍 | 部分约束能力弱于 PostgreSQL（如部分唯一索引需绕行） |
| SQLite | 单文件嵌入式关系库 | 零运维；开发期可立刻跑通 | 并发写入受限；多实例部署时状态不可共享 |

**场景特征匹配**：三字段唯一性、可见性枚举与事务内跨表联动，都是 PostgreSQL 约束表达力能直接覆盖的场景。

**劣势可接受性**：需独立部署与运维。首期已确认需要服务端会话与状态集中（ADR-0005），数据库本就是必要组件，因此该劣势不构成额外负担。SQLite 被排除是因为它与「服务端会话即时失效」的多实例场景冲突；MySQL 是等价可接受的替代，若后续出现既有运维基础可经变更请求改选。

### 约束

- 用户名、邮箱、手机号分别建立唯一约束，不依赖应用层查重保证唯一性。
- 用户状态变更与文章可见性、归属变更在同一事务内完成。
- 迁移脚本与代码同批次发布，不单独升级（见架构文档 §2.3.1）。
- 数据库版本未确认，使用前需在部署阶段拍板。

### 引用记录

- 暂无获授权登记的实质采用。

---

## ADR-0004: 对外接口采用 REST + JSON (2026-10-19)

**目的**: 接口设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对 D4 当面拍板（2026-10-19）

**问题**: 对外接口采用何种协议形式？

**决策**: 采用 REST + JSON over HTTP。

### 现状 / 痛点 / 改造 / 效果

首期消费者是浏览器与 HTTP 客户端，不存在跨语言的服务间调用——因为架构形态已确定为模块化单体（ADR-0001），连服务间接口都不存在。在此前提下，接口形式的选择只剩「对浏览器友好」与「强类型契约」之间的权衡。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| REST + JSON | 资源路径 + HTTP 动词 + JSON 体 | 浏览器与调试工具直接可用；E2E 可用标准 HTTP 客户端断言；无代码生成步骤 | 契约靠文档与测试维护，无编译期类型检查 |
| gRPC | Protobuf 定义 + HTTP/2 | 强类型契约；代码生成；性能高 | 浏览器需 gRPC-Web 或网关；E2E 需额外工具链 |
| REST + Protobuf | REST 路径 + Protobuf 编解码 | 契约强类型且浏览器可读 | 两套工具链的学习与维护成本 |

**场景特征匹配**：消费者为浏览器或 HTTP 客户端，且需求 `US-ARTICLE-02.1` 的浏览体验要求直接可用的响应格式。

**劣势可接受性**：缺少编译期类型检查，由集成测试与接口契约文档补偿；首期无跨语言消费者，gRPC 的强类型优势无法兑现，却要付出网关与工具链成本。

### 约束

- 路径统一前缀 `/api/v1`。
- 请求与响应体为 JSON，字符集 UTF-8。
- 错误响应结构由 `internal/platform/errors.go` 统一定义，包含可机器判别的错误标识与面向人的说明。
- 不可见或已删除的文章返回「不可用」语义，不得返回空内容或部分内容（需求 `US-ARTICLE-02.1` 第 3 条）。

### 引用记录

- 暂无获授权登记的实质采用。

---

## ADR-0008: 按业务模块分包且配置仅来自环境变量 (2026-10-19)

**目的**: 代码质量 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对 D8 当面拍板（2026-10-19）

**问题**: 代码如何组织，配置从何处读取？两者共同决定模块边界是否在代码结构上可见。

**决策**: 按业务模块分包（`internal/<module>/`，模块内再分 handler、service、repository）；配置仅来自环境变量，启动时一次性校验，缺失必需项则启动失败。

### 现状 / 痛点 / 改造 / 效果

Go 社区存在「按技术分层分包」（`handler/`、`service/`、`repo/` 平铺）与「按业务模块分包」两种做法。前者对初学者直观，但在本项目中会与 ADR-0001 的选择冲突：模块边界在目录上不可见，身份认证与文章管理的业务规则会混在同一层，AD-0001 所依赖的「越界引用在评审中可见」这一缓解措施将失效。

配置方面，若同时支持配置文件与环境变量，会出现「配置漂移」——同一项在两处有不同值，且无法判断哪一份生效。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 按业务模块分包 | `internal/identity/`、`internal/article/`，各含 handler、service、repository | 模块边界在目录上一致可见；新增模块不改既有目录 | 跨模块共享逻辑需明确归属，否则易出现循环依赖 |
| 按技术分层分包 | 平铺的 handler、service、repo 目录 | 初学者直观 | 模块边界在目录上不可见；与 ADR-0001 冲突 |
| 混合（模块内分层） | `internal/<module>/{handler,service,repo}` | 兼顾两者 | 目录较深；需约定跨模块调用入口 |

**场景特征匹配**：本项目有明确的模块边界需求（ADR-0001 的数据所有权表），需要目录结构与之一一对应。

**劣势可接受性**：跨模块共享逻辑的归属风险，由以下规则消除——共享能力一律下沉到 `internal/session`、`internal/mailer`、`internal/platform`，业务模块之间不互相引用共享逻辑。目录较深的劣势在首期两个模块的规模下不构成负担。

### 约束

- 业务模块目录固定为 `internal/identity` 与 `internal/article`；共享能力放在 `internal/session`、`internal/mailer`、`internal/platform`。
- 跨模块调用只允许 service 之间的显式接口调用。
- 配置仅从环境变量读取，启动时校验必需项，缺失则拒绝启动（fail closed）。
- 不在代码或配置文件中保留第二份配置来源。

### 引用记录

- 暂无获授权登记的实质采用。

---

## ADR-0009: Go 版本固定为 1.27 (2026-10-19)

**目的**: 类型一致性 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人当面指定（2026-10-19）

**问题**: `go-service` profile 要求 Go 版本从项目依据读取，而项目此前未登记版本，导致构建与工具链无确定基准。

**决策**: Go 版本固定为 1.27，并在开发环境安装对应工具链。

### 现状 / 痛点 / 改造 / 效果

项目初始化时仅确认「后端使用 Go」，未指定版本。实现阶段需要确定的版本才能声明 `go.mod` 的 `go` 指令与选择工具链；`go-service` profile 明确不在 Rule 中填默认版本，因此该值必须来自项目决策。

选定 1.27 的直接影响是获得了 1.22 起提供的方法路由能力，这是 ADR-0002 选择标准库的前提。若版本低于 1.22，标准库将不具备路径参数与方法匹配能力，ADR-0002 的结论需要重新评估。

**验证证据**：开发环境已安装 Go 1.27.0，实测 `go version go1.27.0 windows/amd64`。

**劣势可接受性**：固定较新版本意味着放弃对更早版本环境的兼容。首期为绿地项目，无既有部署环境需要兼容，该代价不成立。

### 约束

- `go.mod` 的 `go` 指令为 1.27。
- 升级 Go 版本属于技术选型变更，须按技术选型合规规范提出变更请求，并重新评估 ADR-0002 的结论。
- 引入任何第三方依赖前须提交变更请求。

### 引用记录

- 2026-10-19 架构设计阶段采用：本 ADR 约束架构文档 §7.2 构建模型。来源：项目负责人当面指定。
- 2026-10-20 架构修订 `architecture-ir01-r04` 登记已批准依赖采用：为 ADR-0012 的 Argon2id 引入 `golang.org/x/crypto`，不改变 Go 1.27 结论。来源：当前用户明确批准。

---

## ADR-0014: 公共 REST 错误采用稳定结构与最小披露 (2026-10-20)

**目的**: 错误处理 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 明确批准公共错误合同

**问题**: 浏览器和 HTTP 客户端需要稳定识别错误，同时登录、权限和内部故障不能泄露敏感原因；公共错误响应应采用什么合同？

**决策**: 所有公共 REST 错误统一为 JSON 对象：必填稳定 `code`、必填用户可读 `message`、可选 `field_errors`、必填 `trace_id`。内部错误链、数据库细节、账号存在性和敏感状态不进入响应。

### 现状 / 痛点 / 改造 / 效果

裸字符串响应无法稳定驱动客户端，统一“内部错误”又会丢失可操作语义。项目需要把公共错误身份、用户说明、字段校验和服务端诊断关联分开。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 稳定结构 + trace | `code/message/field_errors/trace_id` | 客户端可判定；用户可理解；服务端可关联诊断 | 需要维护错误注册与映射 |
| 裸字符串 | 直接返回错误文本 | 实现最少 | 不稳定、不可机器判定、易泄露内部原因 |
| 全部内部错误 | 所有失败同一码 | 隐藏内部细节 | 用户和客户端无法区分可修复问题 |

该结构同时满足 REST 消费者与安全最小披露。维护映射的成本由统一错误入口与契约测试承担。

### 约束

- `code` 是稳定机器标识；`message` 面向用户且不得作为机器判断依据；`field_errors` 只用于可定位的输入字段；`trace_id` 对应服务端日志与审计关联。
- 登录失败对外使用统一 `code` 与等价 `message`，不得区分账号不存在、密码错误、停用或删除。
- 不得返回 Go error、SQL、堆栈、SMTP 响应、密码策略内部参数、会话标识、CSRF 值或敏感审计字段。
- HTTP 状态、公共错误码、日志级别与内部 cause 必须稳定映射；服务端保留 `%w` 错误链，但只在责任边界映射一次。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束全部 REST 接口、登录统一失败和测试合同。来源：当前用户明确批准。

---

## ADR-0015: 高风险写接口采用作用域化幂等合同 (2026-10-20)

**目的**: 接口设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 明确批准 `Idempotency-Key` 与目标状态幂等方案

**问题**: 网络重试和 SMTP 残余窗口可能重复创建管理员或重复替换临时凭据；高风险写接口如何识别安全重放与冲突请求？

**决策**: 创建管理员、密码重置、临时密码重发要求 `Idempotency-Key`；停用与恢复采用目标状态幂等。服务端按“操作者/租户边界（本项目为单租户）、操作类型、目标资源、幂等键”确定作用域，保存规范化请求摘要和原结果；同键同摘要返回原结果，同键不同摘要拒绝冲突。

### 现状 / 痛点 / 改造 / 效果

无幂等保护时，客户端超时重试会重复产生账号、密码和邮件。只按键去重会让不同请求误共享结果；只依赖数据库唯一约束又不能重放原响应。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 作用域键 + 请求摘要 | 保存键、作用域、摘要、状态和结果 | 能安全重放并识别键冲突 | 需要幂等记录与保留策略 |
| 仅数据库唯一约束 | 冲突时报错 | 实现简单 | 无法区分重放和新请求，不能返回原结果 |
| 客户端自行避免重试 | 不保存服务端状态 | 服务端最简单 | 网络不确定性下不可保证 |

批准方案直接覆盖 SMTP 和网络重试风险。幂等记录保留时长保持 TBD，以业务重试窗口、容量和隐私评估确定。

### 约束

- 创建管理员、密码重置、临时密码重发缺少 `Idempotency-Key` 时拒绝请求；具体 Header 大小和字符规则由详细设计确定。
- 请求摘要必须覆盖影响业务结果的规范化字段，不包含明文密码或临时凭据；摘要算法和序列化必须稳定。
- 同作用域、同键、同摘要的并发或后续请求只执行一次并返回相同业务结果；同键不同摘要返回稳定冲突错误。
- 幂等记录状态至少区分处理中、成功和失败；崩溃恢复不得把未知结果静默当作成功。
- 停用、恢复以请求目标状态为幂等语义：已在目标状态时返回稳定成功结果，不重复产生业务副作用；权限与目标资格仍需重新校验。
- 保留期限和容量保持 TBD；过期后重用键的语义须由详细设计与清理策略固定。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束身份治理写接口、SMTP 残余窗口与数据模型。来源：当前用户明确批准。

---

## ADR-0016: 高风险动作写入 PostgreSQL 追加式审计事件 (2026-10-20)

**目的**: 数据流向 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 明确批准 PostgreSQL 追加式审计方案

**问题**: 身份治理、凭据操作和文章治理必须可追责；审计事件如何与业务状态保持一致，并避免被应用静默篡改？

**决策**: 在 PostgreSQL 保存追加式 `AuditEvent`。每个高风险动作的审计事件与对应业务状态在同一数据库事务内写入；应用只获得 INSERT 和查询能力，不提供 UPDATE/DELETE 接口。身份治理、凭据操作、文章治理、转移和永久删除均必须审计。

### 现状 / 痛点 / 改造 / 效果

普通日志可轮转、采样或重复记录，不能替代业务审计。独立外部审计系统能强化隔离，但首期会引入新基础设施和跨系统一致性。PostgreSQL 追加式事件可复用现有事务边界。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| PostgreSQL 追加式事件 | 业务事务内 INSERT 审计行 | 状态与审计一致；无新增基础设施 | 数据库高权限主体仍可修改，需要权限和备份治理 |
| 普通结构化日志 | 写日志文件/流 | 实现简单 | 不保证与业务事务一致，留存和篡改边界不同 |
| 独立审计服务 | 跨系统投递 | 隔离和扩展性强 | 首期新增服务与一致性问题 |

本项目是单数据库模块化单体，同事务追加最符合当前边界。它不宣称对数据库管理员不可篡改；应用层不可更新/删除和运维权限隔离构成首期保证。

### 约束

- 覆盖登录成功与失败、会话撤销与轮换、管理员与普通用户治理、管理员创建、密码重置与重发、标识清理、文章违规治理、无归属恢复与转移、文章永久删除，以及幂等冲突和 SMTP 残余窗口。
- 必填语义包括事件标识、时间、操作者、动作、目标类型与标识、结果、公共错误码、`trace_id` 和最小必要上下文。
- 审计不得保存密码、临时凭据、Session Cookie、CSRF 值、完整邮件正文或不必要的完整邮箱/手机号；敏感标识采用 ID、掩码或不可逆摘要。
- 审计写入失败时，高风险业务写入 fail closed 并回滚；纯失败尝试无法进入原业务事务时，仍通过受控审计入口记录。
- 应用不提供审计 UPDATE/DELETE 接口；数据库权限、备份、保留期和归档数值保持待部署与合规评估。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束跨模块审计入口、数据模型、运行时序与测试。来源：当前用户明确批准。

---

## ADR-0017: 隐私确认、事务性邮件与部署处理门禁 (2026-10-20)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: `requirements-ir01-r16` checkpoint `decision-e3af3912166f7287e8324488cad1cf1b87ba1034fca3378d3e0467a23751592d` 及当前用户批准的 IA-DES-R01-F001 整改方向

**问题**: 注册、管理员创建和 SMTP 外部处理如何证明具备已确认授权输入，而不虚构具体法域或自动合规？

**决策**: 注册前展示并记录版本化隐私声明确认；初始管理员创建目标管理员时记录信息提供授权声明；新管理员必须同时完成改密与当前隐私确认后才能使用其他管理员能力。邮件仅限已确认事务性用途。部署前必须登记适用地区、数据控制者、处理依据和获批准 SMTP 处理方，任一缺失则相关能力 fail closed。本决定不声称自动法律合规。

### 现状 / 痛点 / 改造 / 效果

仅在发布资料中留空值不能证明收集前告知或外部处理授权；在代码中写死法律结论又会伪造项目事实。选择版本化事实与部署门禁，把业务确认、处理输入和技术启用分开。

### 约束

- 保存隐私声明版本、确认时间、主体与来源；审计最小化，不保存多余 PII。
- 管理员信息提供声明是创建请求的必需事实，缺失时不得创建。
- 新管理员双 Gate 为“改密完成 AND 当前隐私确认完成”。
- 撤回请求复用既有删除流程，不新增权限或自助入口。
- 地区、控制者、处理依据和 SMTP 处理方保持待实际责任人登记；不得写成默认值或合规结论。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r05` 采用本决定。来源：已接受 requirements r16 与用户批准整改。

---

## ADR-0018: 数据库恢复后账号、凭据、会话与幂等状态统一 fail closed (2026-10-20，修订 2026-10-22)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 批准灾备恢复合同并要求关闭 IA-DES-R01-F003；用户于 2026-10-22 明确确认 identity-owned RecoveryRunStore、恢复状态机与崩溃收敛修订

**修订关系**: 保留 2026-10-20 关于恢复终结、账号 Gate 与 canonical 删除的原决定；2026-10-22 原位补齐 RecoveryRun 的唯一 owner、状态机、事务边界和 Journal 已消费后的崩溃收敛合同，不新建竞争 ADR。

**问题**: 从旧备份恢复 PostgreSQL 时，如何防止已撤销会话、旧临时凭据和不确定幂等操作被复活？

**决策**: 延续恢复终结事务与账号 recovery Gate。identity 模块通过 identity-owned `RecoveryRunStore` 成为 RecoveryRun 权威状态与事务的唯一 owner；RecoveryCoordinator 只按 `run_id` 驱动状态机并经该 Store 读取、锁定和转换，不接受调用方构造的完整 RecoveryRun。状态主线为 `identity_ready → article_deletion_ready → ready_to_consume → journal_consumed → succeeded`。逐账号核对结论为 `kept_deleted` 时，必须执行或幂等重放完整 canonical 删除事务，而非只修改 User：删除墓碑、标识生命周期、Session/凭据失效、文章无归属化/`held`/来源关联、违规保留、`retained_email` 与 AuditEvent 必须共同成立。恢复使用 ADR-0019 的跨业务数据库 Recovery Journal 检测 run/secret 重放。

### 现状 / 痛点 / 改造 / 效果

备份无法知道恢复点之后发生的改密、停用、删除或撤权。候选包括全局 epoch、仅清空运行凭据、以及账号级 recovery Gate。仅清空 Session/临时凭据会复活旧永久密码与角色；全局 epoch 仍不能独立证明恢复点角色和删除事实正确。账号级 Gate 虽需人工核对，但最简单可验证，且不会把恢复点状态直接当作权限。

RecoveryRun 若由调用方拼装或由 Coordinator 自行散写，无法保证 article evidence、Journal 消费与数据库阶段属于同一 run/backup，也无法在 `MarkConsumed` 后进程崩溃时判定应补写还是停服。identity-owned Store 把状态转换、行锁与恢复审计集中到唯一事务边界；代价是需要持久化 schema、仓储实现和数据库集成测试。

### 约束

- 恢复完成前不得接收业务流量；恢复动作具备独占运行门禁。
- 恢复终结事务把全部非初始管理员账号置为 `suspended`、`recovery_required=true`；恢复点中的 `role/status/deleted` 事实只作为待核材料，不授予运行权限。
- 全部 Session 无条件失效，不尝试判断哪些“可能仍有效”。
- 全部永久密码失效；未 consumed/未 superseded 的 TemporaryCredential 均失效，任何账号都不得使用备份中的旧密码登录。
- 初始 `admin` 只能使用 ADR-0007 的独立一次性 recovery secret；恢复后首次登录强制改密。
- 初始 `admin` 通过既有治理动作逐账号核对角色、状态、删除事实并重置凭据后，才可清除 `recovery_required` 并按核对结果恢复账号。无法重建的事实按 RPO 与运行记录保持受限。
- `kept_deleted` 必须复用与日常删除相同的 canonical 删除协作合同；任一身份、标识、文章、通知邮箱或审计子步骤失败，账号保持 recovery Gate，服务不得开放。
- canonical 删除不新增文章恢复能力；只重建 r16 已确认的删除后不变量。
- `processing`/`uncertain` 幂等记录统一进入 `recovery_review_required`；未经操作员或受控恢复动作核对，不得使用同键重放副作用。核对结论只能转为失败或形成新的受控执行。
- `RecoveryRunStore` 由 identity 拥有，至少提供按 `run_id` 创建/加载、短事务行锁、条件状态转换和已验证 evidence/Journal 绑定字段持久化；其他模块不得直接更新 RecoveryRun 表。
- 外部 article 调用、Journal 读取/CAS/消费和文件 I/O 不得跨 PostgreSQL 长事务。每次转换只在短事务内锁定当前 RecoveryRun，并核对不可变 `run_id`、`backup_ref`、当前阶段和对应 evidence。
- Coordinator 每次启动或重试都按 `run_id` 从 Store 读取权威状态；并发 Coordinator 必须通过行锁/条件更新使同一转换至多提交一次，重复执行不得二次消费 Journal。
- `ready_to_consume → journal_consumed` 是 Journal 副作用与数据库状态之间的显式边界。唯一允许自动收敛的不对称状态是数据库仍为 `ready_to_consume`、但 Journal 已存在与该 run/key/fingerprint 精确匹配的 consumed 记录；重启后先完整验链与锚点，再补写 `journal_consumed`。
- 数据库早于 `ready_to_consume` 而 Journal 已 consumed，或数据库为 `journal_consumed/succeeded` 但 Journal 缺失、截断、回退或不匹配，均不得自动前滚或重新消费，必须 fail closed。
- `journal_consumed` 仅在数据库确认精确 consumed 记录后进入；`succeeded` 仅在 RecoveryRun、Recovery Journal、Journal 头锚点和 article evidence 最终对账全部一致后进入并开放 readiness。
- Recovery Journal 中 run ID/secret 指纹已消费、不可读写，或与业务库 RecoveryRun 不一致时，恢复必须 fail closed。
- 写入恢复 AuditEvent、备份标识、执行结果和告警；不得记录秘密。
- 全部终结步骤成功后才开放 readiness；任一步失败保持停服。
- 若未来改用 epoch，必须独立 ADR；r06 不并存两套授权机制。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r05` 采用本决定，约束恢复时序、数据状态和运行手册。来源：当前用户批准整改。
- 2026-10-20 架构修订 `architecture-ir01-r06` 采用本次修订，补齐永久密码、角色、删除事实与账号恢复 Gate。来源：当前用户批准的灾备恢复合同。
- 2026-10-20 架构修订 `architecture-ir01-r07` 采用本次修订，补齐 `kept_deleted` canonical 删除与 Recovery Journal 一致性。来源：当前用户批准恢复日志合同。
- 2026-10-22 架构修订 `architecture-ir01-r16` 采用本次修订，固定 identity-owned RecoveryRunStore、短事务状态转换与 `journal_consumed` 崩溃收敛。来源：用户于 2026-10-22 明确确认。
- 2026-10-22 identity-auth 设计修订 `identity-auth-design-ir01-r10` 落实本次修订的接口责任、状态机和验证合同。来源：用户于 2026-10-22 明确确认。

---

## ADR-0019: Recovery Journal 使用独立 CAS 头锚点、持久 keyring 与受控文件提供方防止回退和重放 (2026-10-20，修订 2026-10-22)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 批准恢复日志合同及跨 HMAC key 版本复用检测整改；用户于 2026-10-22 明确确认独立故障域 CAS Journal 头锚点、Windows 生产禁用/DACL fail closed、UTC 微秒与 OS 文件锁修订；用户于 2026-10-22 再次明确确认不可变 `backup_ref` 三方绑定与 Begin-key 固定终结/退役约束；用户于 2026-10-22 明确确认 IA-S4.2-DR-F005 采用方案 A，将 S4.2 限定为 provider-neutral Anchor port/协议与确定性测试替身；用户于 2026-10-22 明确确认 IA-S4.2-CR-002 采用“方案 A-补充”，将 S4.2 key retirement 限定为 opaque capability 验证边界

**修订关系**: 保留 2026-10-20 关于业务库外 Journal、完整 keyring、跨 key-version 判重和 secret 明文不落盘的原决定；2026-10-22 首次原位补齐可检测删除/完整前缀截断/旧完整快照回退的独立 CAS 头锚点，以及平台、权限、时间和锁生命周期合同；同日再次原位补齐 Journal/RecoveryRun/article evidence 的不可变备份绑定，以及 key 轮换期间既有 run 可终结、不可过早退役的合同；同日第三次原位明确 S4.2 只交付 provider-neutral Anchor 接缝与测试替身，生产 provider、配置、装配和真实独立故障域验证必须由后续独立任务完成；同日第四次原位明确 S4.2 只定义 opaque `KeyRetirementCapability` 的验证协议和能力边界，真实 retention/archive exit evidence、签发与生产接线由另一个非 S4.5 后续任务完成，不新建竞争 ADR。

**问题**: 业务 PostgreSQL 会随备份回退；如何跨恢复批次持久检测 `recovery_run_id` 和 recovery secret 是否已消费？

**决策**: Recovery Journal 每条记录保存 `key_id/key_version`、HMAC 指纹、run ID、不可变 `backup_ref`、阶段和 consumed，并由位于独立故障域、支持 compare-and-swap 的 `JournalHeadAnchor` 绑定已提交 `journal_id/namespace/epoch/head_sequence/head_hash`。`Begin(runID,backupRef,recoverySecret)` 把 run、backup 与当时 active key 的 ID/version/fingerprint 一次绑定；begun 与唯一 consumed 终结记录必须沿用该绑定。keyring 包含一个 active 新 run 写入 key与仍处保留期的 verify-only 历史 key；verify-only 禁止开启新 run，但允许为由该 key 已 Begin 且未消费的 run 写唯一 consumed 终结记录。`Begin` 对候选 secret 使用全部 retained key 计算候选指纹，并只与相同 key ID/version 的历史记录比较；任一命中即拒绝。key 退役/删除除满足“无 begun 未消费 run、记录已达到批准保留/归档退出条件”外，还必须提交 opaque `KeyRetirementCapability` 并通过验证/消费 port；S4.2 不解析、不签发、不配置该 capability，只定义调用边界和 fail-closed 结果，生产不存在 mint 路径且默认持续禁用 key retirement。RecoveryRunStore、Recovery Journal 与 Article evidence 的 `run_id + backup_ref` 必须精确一致，验证必须分别读取 Store 与 Journal 的独立事实，错误 backup 或任一来源不一致均 fail closed。生产 `FileRecoveryJournal` 暂不支持 Windows；支持平台使用进程退出自动释放的 OS 独占文件锁，所有持久时间统一为 UTC 微秒。S4.2 的十三文件边界只交付 provider-neutral `JournalHeadAnchor` port/协议、调用边界和确定性测试替身，以及 `KeyRetirementCapability` 验证协议/能力边界；不选择或交付生产 CAS provider、retirement evidence 来源、capability 签发、配置及启动装配。生产 Anchor provider 与真实独立故障域验证仍由 S4.5 负责；retention/archive exit evidence、capability 签发及 key retirement 生产闭环由 Task-Planner 另行登记一个不并入 S4.5 的独立后续任务。对应任务完成前不得宣称恢复链或生产 key retirement 就绪。key、锚点、权限、capability 或 Journal/DB 不可证明一致时均保持停服。

### 现状 / 痛点 / 改造 / 效果

候选包括业务数据库表、主机持久卷文件和外部秘密/运维系统。业务表会随恢复回退，不能检测跨批次复用；外部系统隔离更强但首期新增基础设施和接入凭据。单实例维护模式下，持久卷文件是最小可验证方案。

| 方案 | 优势 | 劣势 |
|---|---|---|
| 主机持久卷文件 + 独立 CAS 头锚点（选定） | Journal 不随业务库恢复；锚点可检测文件删除、完整旧前缀和旧快照回退；可崩溃协调 | 新增独立故障域与 CAS provider；双写协议、迁移和损坏恢复需运行手册 |
| 仅主机持久卷追加文件 | 无新增服务；可离线演练 | 合法旧前缀/旧完整快照仍可通过链验证，不能证明最新头未回退 |
| 外部秘密/运维系统承载完整 Journal | 隔离、权限和审计能力强 | 新增依赖、网络和凭据；首期改造范围更大 |
| 业务数据库表 | 事务集成简单 | 会随备份回退，不能满足目标 |

本轮交付边界采用方案 A：先固化 provider-neutral port、协议和确定性测试替身，再由独立任务选择并接入生产 provider。另一条路是在 S4.2 十三文件内同时选型和装配生产 provider，但会要求第十四文件、配置/启动接线或新增依赖，并把未经批准的技术栈写进当前任务，因此不采用。

按所有 retained key 验证的成本随 key 数量线性增长。首期通过运行策略限制 retained key 数量，具体上限保持 TBD；不能为性能删除仍承担历史判重的 key。

### 约束

- 接口至少提供 `Begin(runID,backupRef,recoverySecret)`、`MarkConsumed(runID)`、`Read(runID)`、`VerifyAgainst(storeRecoveryRun)`；provider 内部执行全 keyring 判重、头锚点校验与锁管理。`backupRef` 在 Begin 后不可变；`VerifyAgainst` 必须从持久 Journal 读取 Journal 侧 `run_id + backup_ref`，并与 RecoveryRunStore 返回的独立权威 RecoveryRun 逐字段比较，不能只在调用方对象内部自比较，调用方也不得自行拼装完整 Journal 状态。
- JournalEntry、持久 disk record 与 record MAC payload 都必须包含同一规范化 `backup_ref`；`backup_ref` 受 HMAC 与哈希链保护。相同 run 的 begun/consumed 必须保持同一 `backup_ref`、key ID/version/fingerprint；缺失、改变或错误 backup 一律 fail closed。
- 进入 `ready_to_consume` 与最终开放前，RecoveryRunStore、Recovery Journal、Article evidence 三个独立来源的 `run_id + backup_ref` 必须精确一致；只比较 Store 内部字段或调用方 DTO 内部字段不构成验证。
- `JournalHeadAnchor` 至少保存 `journal_id`、namespace、epoch/generation、已提交 `head_sequence/head_hash`、CAS revision/token，以及可恢复的 pending 目标（旧头、新头与操作标识）。Anchor 必须独立于 Journal 文件/持久卷故障域，且不得随业务 PostgreSQL 恢复回退。
- `JournalHeadAnchor` 是 provider-neutral port，不暴露具体产品 SDK 类型；最小协议提供按 namespace 读取快照/revision，以及以 expected revision 对 next committed/pending 状态执行 compare-and-swap。CAS 冲突、缺失、回退、stale revision 和不可判定结果均返回可区分失败并由调用方 fail closed。
- S4.2 只在既有十三文件内实现该 port、Journal 调用边界和确定性测试替身。替身必须可编排成功、CAS 冲突、stale revision、missing/rollback 与 pending/committed 转换，但只作为协议测试证据，不得注册为生产 provider或作为真实独立故障域证据。
- Task-Planner 必须为生产 CAS Anchor provider、配置、启动装配和真实独立故障域验证登记单独任务。该任务及其独立验证完成前，恢复运行 Gate 保持关闭；S4.2、B3 或任何部署说明不得据测试替身声称生产可用。
- Journal 空文件只允许在不存在任何既有 anchor 的一次性显式 bootstrap 中建立；存在 anchor 时，文件缺失、空文件或试图重新初始化均 fail closed。
- keyring 只有一个 active 新 run 写入 key；历史 key 为 verify-only。Journal 记录必须携带稳定 `key_id` 与 `key_version`。verify-only 的“不得写新记录”只表示不得为该 key 开启新 run；由该 key 已开始且未消费的 run 仍允许写唯一 consumed 终结记录。
- `Begin` 使用 active key 生成待写记录，同时使用所有 retained key 计算候选指纹；任一对应历史记录命中，必须拒绝 run。
- `MarkConsumed(runID)` 必须读取 begun 记录并固定沿用 Begin 时绑定的 key ID/version/fingerprint 与 `backup_ref`；active key 已轮换不改变该绑定，也不得改用当前 active key重签 run。
- 只要某 key 尚有 begun 未消费 run，或该 key 对应的任一 Journal 记录仍处已批准保留/归档可验证范围，该 key 就不得退役或删除。即使静态条件看似满足，也只有 opaque `KeyRetirementCapability` 经验证/消费 port 对精确 `key_id/key_version` 的本次 retire/delete 请求返回成功后才能销毁；capability 缺失、空值、拒绝、不可用、绑定不匹配或结果不可判定均 fail closed。
- S4.2 十三文件只定义 `KeyRetirementCapability` 为调用方不可解析、不可复制到日志/审计/持久状态的 opaque 值，并定义验证/消费 port 与确定性测试替身。S4.2 生产路径没有 capability mint、配置或 wiring；不存在 production verifier 时 retirement 必须保持禁用，不能以布尔 evidence、调用方自报或 fake 绕过。
- Task-Planner 必须另行登记一个不并入 S4.5 的独立后续任务，负责真实 retention/archive exit evidence 的来源与核验、capability 签发、生产 verifier/config/startup wiring、单次消费、防复制/重放、审计、测试和真实环境验证。该任务及独立验证完成前不得启用生产 key retirement 或据此声明生产就绪。
- S4.5 的范围保持为生产 Anchor provider、配置/装配和真实独立故障域验证；不得把 key retirement capability 生产闭环并入 S4.5。
- 轮换到新 active key 不得使既有 run 无法完成。固定崩溃恢复合同为：`Begin(v1) → v2 active/v1 verify-only → MarkConsumed` 使用 v1 写唯一 consumed → 若在数据库写 `journal_consumed` 前崩溃，重载 v2+v1 keyring 并验证 v1 consumed/Anchor → 仅补写数据库 `journal_consumed`，不得再次消费或改用 v2。
- 记录引用未知 key、key 缺失、解密/加载失败或任一候选指纹无法计算时 fail closed。
- keyring 与 Journal 必须整体备份、迁移和恢复；禁止只迁移记录或只迁移 active key。
- HMAC key 独立受控并版本化；Journal 不保存 recovery secret 明文。
- Journal 文件位置由受控配置提供，不硬编码 OS 路径；目录、Journal、临时文件与锁入口必须证明 owner-only 写边界。生产 Anchor provider 的选择、配置键和启动装配不属于 S4.2，由后续独立任务定义。无法读取/解释权限或发现未批准写主体时 fail closed。
- 支持平台的写入协议固定为：取得进程生命周期 OS 独占锁 → 校验 Journal 与 committed anchor → 写入并 `fsync` 同目录临时文件 → CAS anchor 从旧 committed 头进入 pending 新头 → 原子替换 Journal 并同步目录 → CAS pending 为 committed 新头。只有最终 committed 成功才向调用方返回成功。
- 崩溃恢复只允许三种确定结果：文件仍为旧头时清除未发布 pending，或在可验证临时文件仍存在时完成发布；文件已为 pending 新头时提交 pending；文件既非旧头也非新头时 fail closed。任何自动处理前都必须持有 OS 锁并重新读取 CAS revision。
- 文件删除、完整合法旧前缀、旧完整快照、anchor 缺失/回退、epoch 不匹配、截断、重复、校验失败或部分写入均视为不可证明连续性并停服。
- `.lock` 路径只作为 OS 锁的稳定 inode/handle 入口；锁所有权由内核 advisory/exclusive lock 或等价独占 handle 表示，锁文件存在不表示被占用。锁覆盖读取、验链、临时写、anchor CAS、替换、目录同步和 committed CAS，进程退出或句柄关闭自动释放。
- 当前生产 `FileRecoveryJournal` 仅支持已验证具备原子同目录替换、文件与目录同步及 OS 锁语义的本地 POSIX 文件系统。共享/NFS 类文件系统在锁与持久化语义未证明前不宣称支持。
- Windows 生产初始化必须在任何 Journal、anchor、临时文件或锁状态变更前返回 unsupported。未来 Windows provider 必须证明同卷原子替换、持久化和 DACL owner-only 边界；安全描述符不可读/不可解释或存在未批准写 SID 时 fail closed。具体允许 SID 清单属于部署/安全输入，当前为 unknown。
- Journal、Anchor 与 RecoveryRun 的持久时间统一为 `UTC().Truncate(time.Microsecond)` 的规范值，并以同一值写入 PostgreSQL 与外部记录后精确比较；纳秒值、非 UTC 值或其他非规范存量不得静默舍入或改写。
- 持久卷本身需独立备份和迁移；迁移必须连同完整 Journal 与 HMAC key 版本，不能只复制业务数据库。
- 恢复完成后先原子标记 consumed，再开放 readiness；无法确认 consumed 时不得开放。
- Journal 只用于灾备恢复防重放，不成为业务审计、账号状态或文章恢复机制。
- 同一持久卷/keyring 基础设施可承载其他追加式安全 Journal，但必须使用独立 namespace、record schema 和接口。Article Deletion Journal 不得写入 recovery secret 指纹 namespace，也不得参与 RecoveryJournal 的 run/secret 判重。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r07` 采用本决定，约束恢复部署、运行时序、故障处理与测试。来源：当前用户批准恢复日志合同。
- 2026-10-20 架构修订 `architecture-ir01-r08` 采用本次修订，补齐跨 key-version secret 复用检测与 key 退役条件。来源：当前用户批准第四轮复审整改。
- 2026-10-20 架构修订 `architecture-ir01-r09` 最小复用本决定的持久卷/keyring 基础设施，同时固定 Article Deletion Journal namespace 隔离。来源：当前用户批准文章删除日志方案。
- 2026-10-22 架构修订 `architecture-ir01-r16` 采用本次修订，固定独立 CAS 头锚点、支持矩阵、权限/时间/OS 锁和崩溃恢复协议。来源：用户于 2026-10-22 明确确认。
- 2026-10-22 identity-auth 设计修订 `identity-auth-design-ir01-r10` 落实本次修订的 provider、协调器与测试合同。来源：用户于 2026-10-22 明确确认。
- 2026-10-22 架构修订 `architecture-ir01-r17` 采用本次修订，固定不可变 `backup_ref` 三方绑定、独立来源验证与 Begin-key 终结/退役合同。来源：用户于 2026-10-22 明确确认 IA-S4.2-DR-F003/F004 按建议整改。
- 2026-10-22 identity-auth 设计修订 `identity-auth-design-ir01-r11` 落实本次修订的接口字段、状态机、轮换崩溃恢复与测试合同。来源：用户于 2026-10-22 明确确认 IA-S4.2-DR-F003/F004 按建议整改。
- 2026-10-22 架构修订 `architecture-ir01-r18` 采用本次交付边界修订，固定 S4.2 只交付 Anchor port/协议/确定性替身，并把生产 provider、配置、装配与独立故障域验证移交后续独立任务。来源：用户于 2026-10-22 明确确认 IA-S4.2-DR-F005 方案 A。
- 2026-10-22 identity-auth 设计修订 `identity-auth-design-ir01-r12` 落实本次修订的接口、场景、部署、运行、风险和测试 Gate；十三文件边界不变。来源：用户于 2026-10-22 明确确认 IA-S4.2-DR-F005 方案 A。
- 2026-10-22 架构修订 `architecture-ir01-r19` 采用本次 key retirement 能力边界修订，固定 S4.2 仅验证 opaque capability、生产无 mint/default disabled，并把完整生产闭环移交非 S4.5 独立后续任务。来源：用户于 2026-10-22 明确确认 IA-S4.2-CR-002 方案 A-补充。
- 2026-10-22 identity-auth 设计修订 `identity-auth-design-ir01-r13` 落实本次修订的接口、失败语义、风险、测试与生产 Gate；十三文件边界不变。来源：用户于 2026-10-22 明确确认 IA-S4.2-CR-002 方案 A-补充。

---

## ADR-0021: PostgreSQL 18、pgx/v5 与项目内版本化 SQL 迁移基线 (2026-10-21，修订 2026-10-21)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 初始基线由当前用户于 2026-10-21 对 PostgreSQL 18、pgx/v5 原生接口、项目内最小迁移执行器与临时 module path `han` 的明确推荐回复“继续”；启动迁移合同由项目负责人在当前 B0 评审返工中当面确认（2026-10-21）；HTTP 监听合同由“项目负责人在当前 B0 评审返工中确认 HAN_HTTP_ADDR 推荐方案（2026-10-21）”

**关系**: 细化 ADR-0003 的 PostgreSQL 版本与接入方式，并按 ADR-0009 完成 `github.com/jackc/pgx/v5` 第三方依赖批准；Go 1.27 与既有 `golang.org/x/crypto` 批准不变。

**问题**: ADR-0003 已选 PostgreSQL，但数据库 major、Go 驱动、连接池、迁移执行合同、生产 DSN 与 HTTP 监听地址名称、服务启动门禁和当前 Go module path 未确定，B0 无法据此建立可重复且 fail-closed 的数据库与监听基线。

**决策**: 固定 PostgreSQL major 18，生产与 CI 使用受支持的最新 18.x patch；Go 数据访问采用 `github.com/jackc/pgx/v5` 原生 API 与 `pgxpool`；迁移采用 `migrations/NNNN_name.sql` 和项目内最小执行器，并由 `han-server` 每次启动时自动执行；生产 DSN 与监听地址分别只从 `HAN_DATABASE_DSN`、`HAN_HTTP_ADDR` 读取且都无代码默认值；两项配置先完成校验，再连接数据库、迁移、调用 `net.Listen("tcp", addr)` 并启动 HTTP 服务；任何前置失败均非零退出。当前 Go module path 使用 `han`。

### 现状 / 痛点 / 改造 / 效果

PostgreSQL 官方版本策略是本决定的行业规范对标：major 决定兼容边界，minor 提供该 major 内的修复。2026-10-21 核对结果为 PostgreSQL 18 处于支持期，当前 minor 为 18.6，支持至 2030-11；`18.6` 是本次取证事实，不是长期锁死的补丁合同。

数据库版本有三种管理方式：

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 固定 major 18，跟进受支持的 18.x patch | 兼容边界固定在 major，生产与 CI 更新同 major 补丁 | 避免无意 major 升级；持续获得修复 | 需要持续维护补丁升级与兼容验证 |
| 长期固定 18.6 | 所有环境永久使用一个 patch | 初始环境完全一致 | 会把取证时点固化为长期合同，后续修复无法自然进入 |
| 不限制 major | 部署时使用任意 PostgreSQL 版本 | 初始约束少 | CI、生产和开发可能跨 major 漂移，迁移与 SQL 行为不可重复 |

Go 数据访问与迁移有三条可行路线：

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| pgx/v5 原生 API + `pgxpool` + 项目内最小执行器 | 直接使用 PostgreSQL 协议能力；项目执行版本化 SQL | 依赖面小；事务、连接池与 PostgreSQL 行为直接可见；迁移合同可审查 | 绑定 PostgreSQL；项目需维护少量迁移执行代码 |
| `database/sql` 驱动 + 外部迁移框架 | 通用数据库接口与第三方迁移命令 | 工具生态成熟；接口较通用 | 新增驱动和迁移框架依赖；通用抽象不能消除本项目已选 PostgreSQL 的专用约束 |
| ORM + ORM 迁移 | 模型映射与 schema 演进由 ORM 承担 | CRUD 开发快；模型工具集中 | 引入更大抽象与依赖面；生成或隐式 SQL 增加约束审查成本 |

本项目已由 ADR-0003 明确只使用 PostgreSQL，首期又是单 module、单服务，pgx/v5 的 PostgreSQL 专用性不是额外损失。自有执行器只承担顺序、事务、锁和记录，不扩展为通用迁移框架；少量维护成本可接受，换来无额外框架依赖和可直接审查的 SQL。

迁移接入服务启动有三条可行路线：

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| `han-server` 启动时自动迁移 | 入口完成配置与连接校验，调用项目内 runner；成功后才建立 HTTP 监听 | 单一启动入口；代码与 schema 同批交付时不易漏执行；失败可直接阻断不兼容服务 | 每次启动都要经过迁移检查；数据库暂不可用时服务不会降级启动 |
| 独立迁移命令或编排 init job | 部署流程先运行迁移，再启动服务 | 迁移与服务进程分离；适合成熟编排平台 | 新增交付件或编排前置；首期没有已确认的平台保证其一定执行 |
| 运维人工执行 SQL | 发布人员按说明运行脚本 | 应用入口最简单 | 易漏执行或乱序；无法把迁移成功作为进程开放服务的可验证门禁 |

首期只有一个可执行目标、一个服务进程和一套随代码同批交付的迁移资产，尚无独立编排平台合同，因此选择启动时自动迁移。该选择把数据库暂不可用直接转化为启动失败；这是有意的 fail-closed 行为，避免旧 schema 上开放正常 HTTP 服务。若未来引入可证明前置成功的部署编排，再以独立 ADR 评估是否把迁移执行权移出进程启动。

生产 HTTP 监听地址有三条可行路线：

| 方案 | 安全性 | 部署性 | 测试性 | 首期复杂度 / no-hardcode |
|---|---|---|---|---|
| 必填单一 `HAN_HTTP_ADDR`（选定） | 可在任何数据库副作用前严格拒绝非 IP、空 host 与非法端口；无隐式暴露面 | 单变量直接映射单个 TCP endpoint，适合当前单进程部署 | 地址解析、调用顺序和监听失败均可独立注入验证 | 最小；无代码默认值，符合配置只来自环境变量 |
| 拆分 `HAN_HTTP_HOST` / `HAN_HTTP_PORT` | 可分别校验，但组合状态增加遗漏与歧义 | 部署系统需同时维护两个变量 | 组合矩阵更多 | 首期复杂度较高；仍可 no-hardcode，但没有实际收益 |
| 部署层预打开 listener / 继承文件描述符 | 可由部署层集中控制 socket 权限 | 适合具备 socket activation 的成熟平台 | 需模拟平台协议、FD 生命周期与继承失败 | 复杂度最高；当前没有已确认的部署平台合同 |

首期选择必填单一 `HAN_HTTP_ADDR`：它以最少配置表面积给出确定的监听 endpoint，并能在连接数据库前完成纯配置校验；拆分 host/port 不增加首期能力，预打开 listener 则引入尚不存在的平台依赖。该选择不把地址写入代码，也不允许客户端测试配置替代生产服务端配置。

### 约束

- Go 版本保持 1.27。
- PostgreSQL major 固定为 18；生产与 CI 使用仍受官方支持的最新 18.x patch。升级到其他 major 必须重新决策。
- 应用使用 `github.com/jackc/pgx/v5` 原生 API；连接池采用 `github.com/jackc/pgx/v5/pgxpool`。本决定只新增批准 `pgx/v5`，不改变既有 `golang.org/x/crypto` 批准。
- 不引入 ORM、数据库抽象层或第三方 database migration framework。
- 迁移文件位于 `migrations/`，命名为 `NNNN_name.sql`；`NNNN` 必须唯一并形成可排序的版本序列。
- 项目内最小迁移执行器维护 `schema_migrations`，记录已成功应用的版本。
- 执行器按版本升序执行；每个迁移 SQL 与对应 `schema_migrations` 记录在同一独立事务中完成，只有事务提交成功才视为已应用。
- 执行器使用 PostgreSQL advisory lock，保证同一数据库同一时刻只有一个迁移执行者。
- 生产数据库连接串的唯一环境变量名为 `HAN_DATABASE_DSN`；不得同时支持别名、配置文件或代码内默认 DSN。
- 生产 HTTP 监听地址的唯一环境变量名为 `HAN_HTTP_ADDR`，生产入口必填且无代码默认值。值必须是单个 TCP `IP:port`：host 只能是 IPv4 字面量或方括号包裹的 IPv6 字面量，port 必须为 `1–65535`；空 host、域名、port 0、URL scheme、路径、多地址以及其他非该格式输入均拒绝。
- `HAN_DATABASE_DSN` 与 `HAN_HTTP_ADDR` 任一缺失、空白或非法时，`han-server` 必须在创建数据库连接池、Ping 或执行迁移之前非零退出。
- `han-server` 的启动顺序固定为：读取并校验 `HAN_DATABASE_DSN` 与 `HAN_HTTP_ADDR` → 建立连接池并 Ping 验证数据库连接 → 调用项目内 migration runner → runner 成功返回 → `net.Listen("tcp", addr)` → 启动无业务路由的 B0 最小 HTTP 服务。
- `HAN_DATABASE_DSN` 校验失败、数据库连接/Ping 失败或 migration runner 返回错误时，`han-server` 必须释放已创建资源并以非零状态退出；这些路径不得调用 listener，也不得以降级模式继续提供业务接口。
- 迁移成功后监听失败时，`han-server` 必须关闭数据库池并非零退出；已成功迁移保留，由迁移幂等合同保证下次启动安全重试。运行期 `Serve` 异常返回同样按进程失败处理，清理 listener 与数据库池后非零退出。
- B0 不扩展优雅停机合同；`HAN_TEST_BASE_URL` 仅是客户端测试变量，不得作为 `HAN_HTTP_ADDR` 的别名、默认值或回退来源。
- migration runner 只拥有迁移文件加载、advisory lock、顺序执行、事务和 `schema_migrations` 记录职责；它以错误返回失败，不自行决定进程退出或 HTTP 监听。`han-server` 是消费该错误并执行 fail-closed 启动门禁的唯一进程所有者。
- 任一迁移失败立即停止，不执行后续迁移，也不得把失败版本记为成功；错误必须返回 `han-server` 入口，具体日志格式与测试实现落 B0.1 代码任务，日志不得输出 DSN。
- 当前 Go module path 为 `han`。绑定正式远程并首次对外发布前，可通过单独变更调整 module path；本轮不猜测组织域名。
- 本机仅验证 `go version go1.27.0 windows/amd64`。当前无 `psql`、无 Docker，本轮不声称已完成 PostgreSQL 18、pgxpool 或迁移并发的真实数据库验证。

### 引用记录

- 2026-10-21 架构修订 `architecture-ir01-r13` 采用本决定，约束 B0 的数据库版本、驱动、连接池、迁移资产、交付与部署基线。来源：当前用户对明确推荐回复“继续”。
- 2026-10-21 架构修订 `architecture-ir01-r14` 原位采用本次启动合同修订，约束 B0.1 的 `HAN_DATABASE_DSN`、启动内自动迁移、非零退出语义和 HTTP 监听门禁。来源：项目负责人在当前 B0 评审返工中当面确认（2026-10-21）。
- 2026-10-21 架构修订 `architecture-ir01-r15` 原位采用 HTTP 监听合同修订，约束 B0 的 `HAN_HTTP_ADDR` 格式、配置校验前置、监听/Serve 失败语义与资源清理。来源：项目负责人在当前 B0 评审返工中确认 HAN_HTTP_ADDR 推荐方案（2026-10-21）。

---

## ADR-0022：生产可观测性采用标准库适配器与部署平台组合

- **状态**：已接受
- **日期**：2026-10-23
- **修订**：2026-10-23，关闭独立评审 OBS-R01-F001～F003，并固定双 server 关闭退出码
- **目的分类**：架构设计
- **影响范围**：`cmd/han-server` composition root、identity login 与 platform Origin observer、PostgreSQL failure auditor、生产日志、metrics exposition、部署采集/留存、告警、Dashboard、值守与 runbook
- **确认来源**：用户于 2026-10-23 明确授权“对待确认或决策项采用建议值”；Lead 已固定 OBS-R01-F001～F003 的最小修订方案；用户在本轮明确要求采用推荐退出语义

### 问题

identity-auth design r18 的 Q13 与 F005 最新复审要求生产真实装配 logger、metrics、Login outcome observer、Origin rejection observer、failure auditor、正数有限预算及取消能力；当前 `cmd/han-server` 尚未完成这些 production composition。backend、日志 sink/留存、Histogram buckets、预算、告警阈值、Dashboard、paging、值守与 runbook 若继续保持 TBD，会使 F005 无法形成可实施合同。

现有代码已经提供 provider-neutral port 和有界行为：`platform.LoggerWithTrace`、`LoginOutcomeObserver` / `LoginMetrics`、`OriginRejectionObserver` / `OriginSecurityMetrics`、`PostgresLoginFailureAuditor`；构造器拒绝 nil，handler/guard 通过 `context.WithTimeout` 传递取消和 deadline。`go.mod` 的运行依赖仅为已批准的 pgx/v5 与 x/crypto。本决定必须复用这些边界，不因运维 backend 把领域层绑定到新 SDK，也不能把测试 fake/no-op 当作生产实现。

独立评审 `.spec/wip/identity-auth/review-observability-decision-r01.md`（SHA256 `76C08FABC03F9D9AEB58D6F8410C99D242905CDF658E4D4BC422881959D4CF63`）进一步确认三个合同缺口：自研 exposition 未固定线协议（F001）、部署值可调与 parser 唯一 allowlist 相互冲突（F002）、双 server 缺少有界 shutdown 和错误优先级（F003）。本次原位修订只关闭这三项，不改变日志、指标名、告警、Dashboard、runbook 或 provider-neutral 方向，也不宣称实现/部署完成。

### 证据与候选

本决定采用以下已核证事实：

- 项目证据位置：`.add/project/knowledge/design-workspace/identity-auth/design.md` 的 Q13、`.spec/wip/identity-auth/review-code-s3.2-f005-r03.md` 的剩余阻塞、`.spec/wip/identity-auth/review-observability-decision-r01.md` 的 F001-F003、`internal/identity/login.go`、`internal/platform/http_security.go`、`internal/platform/logging.go`、`cmd/han-server` 与 `go.mod`。
- Go 标准库 `log/slog` 的 `JSONHandler` 可输出一行一个 JSON 对象，能复用现有 `platform.LoggerWithTrace`。
- 现有登录指标固定为 `han_identity_login_attempts_total`、`han_identity_login_internal_failures_total`、`han_identity_login_duration_seconds`；Origin 指标固定为 `han_platform_origin_rejections_total`，标签由现有低基数 allowlist 限定。
- Prometheus 官方实践要求告警保持简单、面向症状且可行动；alert rules 交给 Alertmanager 做聚合、抑制、静默与通知。Grafana 可把告警规则关联 Dashboard 面板。
- F005 已固定 observer/logger/metrics adapter 遵从传入 context cancellation/deadline；超时后一次失败返回，不重试、不切换 sink、不改变公共响应、主事务或审计结果。

| 候选 | 实现 | 优势 | 代价 / 风险 |
|---|---|---|---|
| A. 应用绑定完整 observability SDK 与托管平台 | 应用直接初始化厂商日志、metrics、paging SDK | 平台功能集中 | 新增依赖与供应链；领域/入口绑定具体 provider；需要账号与付费平台决定，超出当前证据 |
| B. 标准库 + provider-neutral adapter + 部署平台组合（选定） | `slog` JSON stdout；项目内标准库 metrics registry/exposition；Prometheus-compatible scrape、Grafana、Alertmanager | 复用现有 port；无新增 module；backend 可由部署替换；最小可实施 | 项目需维护小型并发安全 registry/exposition；部署侧必须提供采集、规则、receiver 与演练 |
| C. 仅结构化日志，不提供 metrics | 所有诊断依赖日志搜索 | 代码最少 | 无法实现现有 Login/Origin metrics port、分位数与可靠阈值告警；不满足 F005 |

选择 B。它是当前接口可承接的最小方案，不要求新增 SDK、自然人、外部账号或付费平台。Prometheus-compatible backend、日志存储和通知渠道保留在 adapter/config 与部署清单中；应用 port 不感知具体 provider。

### 决策

#### 1. 日志 sink、格式与留存

- 生产 logger 使用 Go `log/slog.JSONHandler`，经 `platform.LoggerWithTrace` 输出 JSON Lines。
- 运行日志写 stdout；仅 logger 初始化前无法结构化的泛化致命启动信息写 stderr。应用不写本地日志文件。
- 部署平台负责采集 stdout/stderr，并提供至少 **30 天可查询留存**。这是首期运行基线，不是法律/审计保留承诺。
- PostgreSQL `AuditEvent` 继续是权威审计机制；运行日志不得替代审计，不得记录 DSN、凭据、Cookie、Origin 原值、用户输入或其他被禁止字段。
- 应用无法可靠探测外部 collector 或留存策略，因此不伪造启动探测；采集与 30 天查询验证属于生产就绪 Gate，未验证不得宣称生产可观测。

#### 2. Metrics backend、exposition 与 Histogram

- 应用内使用 Go 标准库实现固定、小型、并发安全的 metrics registry/exposition adapter，适配现有 `LoginMetrics` 与 `OriginSecurityMetrics`；**不新增 SDK 或 module**。
- 生产使用 Prometheus pull scrape。必填 `HAN_METRICS_ADDR` 建立独立 listener，仅暴露 `/metrics`；地址必须是单个 IPv4 或方括号 IPv6 `IP:port`，port `1–65535`，无默认值，由部署网络策略限制访问。
- 首期线协议固定为 **Prometheus text exposition format 0.0.4**，不实现 OpenMetrics，也不按 `Accept` 做内容协商。成功响应的 `Content-Type` 必须精确为 `text/plain; version=0.0.4; charset=utf-8`；正文使用 UTF-8、LF 换行、每样本一行且必须以最终 LF 结束。
- metric name 必须匹配 `[a-zA-Z_:][a-zA-Z0-9_:]*`，label name 必须匹配 `[a-zA-Z_][a-zA-Z0-9_]*`。HELP 文本中的反斜杠和换行分别编码为 `\\`、`\n`；label value 中的反斜杠、双引号、换行分别编码为 `\\`、`\"`、`\n`。有限浮点使用 `strconv.FormatFloat(value, 'g', -1, 64)` 等价规范格式；特殊值固定为 `NaN`、`+Inf`、`-Inf`。
- Counter 必须按顺序输出 `# HELP`、`# TYPE <name> counter` 与样本。classic Histogram 必须输出 `# HELP`、`# TYPE <name> histogram`，再按 `le` 数值递增输出累计 `<name>_bucket`；必须包含 `le="+Inf"`，且其值等于 `<name>_count`，随后输出 `<name>_sum`、`<name>_count`。
- encoder 必须先在内存中完整编码并验证，再提交 200。编码失败时不得发送部分 200；固定返回 `500`、`Content-Type: text/plain; charset=utf-8` 与正文 `metrics encoding failed\n`，不得暴露内部 cause。
- 只暴露现有固定指标与低基数标签。禁止新增 username、user ID、trace ID、原始 URL、Origin 原值或任意请求材料标签。
- `han_identity_login_duration_seconds` 使用 classic Histogram，生产必填 `HAN_LOGIN_DURATION_BUCKETS_SECONDS`。部署实现基线为 `0.005,0.01,0.025,0.05,0.1,0.25,0.5,1,2.5,5` 秒；parser 合法集合为 `1..32` 个值，每项正数、有限、严格递增、无重复且最大值 `<=60s`。基线不是唯一 allowlist。
- 协议级应用测试必须使用独立 parser、严格语法 fixture 或等价 conformance 验证上述线协议；真实 backend scrape 继续作为部署 Gate，但不得成为首次发现格式错误的唯一 Gate。

#### 3. Observer 与 auditor 正数有限预算

| 调用边界 | 配置键 | 部署实现基线 | parser 合法范围 |
|---|---|---:|---:|
| Login outcome observer | `HAN_LOGIN_OBSERVATION_BUDGET` | `250ms` | `1ms..5s` |
| Origin rejection observer | `HAN_ORIGIN_OBSERVATION_BUDGET` | `250ms` | `1ms..5s` |
| Login failure auditor | `HAN_LOGIN_FAILURE_AUDIT_BUDGET` | `2s` | `10ms..30s` |

- 三个值必须由生产环境显式注入，无代码默认值；范围端点均包含。缺失、解析失败、非有限或超范围值均 fail closed。
- 基线是首期部署值，不是 parser 唯一 allowlist。容量评审后可只修改合法范围内的部署值；超范围必须先修订 ADR/设计并重新评审。
- Login/Origin observer 中 logger 与 metrics fan-out 共享调用方传入的同一个剩余 deadline；不得为第二个 sink 重新获得完整预算。
- adapter 必须遵从 context cancellation/deadline。超时或 sink 失败后只返回一次失败，不重试、不切换 sink、不泄漏 goroutine/资源；不改变公共响应、主事务、独立 failure audit 的既定结果。

#### 4. 告警实现基线与调优边界

| 告警信号 | 初始条件 | 持续时间 | 级别 |
|---|---|---:|---|
| scrape 不可达 | 目标 `up == 0` | `2m` | paging |
| 登录内部失败率 | `10m` 内 login attempt 至少 20 次，且 internal failure / attempt `>5%` | `10m` | paging |
| 登录延迟 | `10m` 至少 20 个 duration 样本，P95 `>2s` | `15m` | warning |
| Origin rejection | `10m` 累计 `>=20` | `10m` | warning |

- 每条告警必须有可行动注释并链接固定 runbook 入口。信号退出阈值后保持 **10 分钟**稳定再 resolve；由 alert rule 的 recovery/keep-firing 能力或等价部署规则实现。
- 上述阈值是诊断性实现基线，不是业务 SLA、SLO 或容量承诺。容量实测后可调整 buckets、阈值、窗口、最小样本和分位数，但必须保留信号语义、低基数标签、warning/paging 可行动性与变更评审。

#### 5. Dashboard、通知与值守

- Dashboard 平台采用 Grafana，固定入口名为 **`Han / Identity Auth`**。
- 最小面板：scrape health；login attempts 按 outcome；login duration P50/P95/P99；internal failure 按现有固定分类；Origin rejection；当前 alert 状态。
- Alertmanager 负责聚合、抑制、静默与通知。warning 路由到团队运行告警渠道；paging 路由到生产值守通知 receiver。
- 只登记角色，不编造具体人：第一响应为**生产值守角色**；第二响应为 **Han 服务 owner / identity 技术责任角色**；安全事件升级到**安全值守角色**。
- 稳定 runbook 入口固定为 `docs/runbooks/identity-observability.md`。该文件及实际 receiver/排班由后续受权任务落地；路径和升级角色不再 TBD，但文件、账号与外部渠道未配置并演练前生产就绪 Gate 关闭。

#### 6. Composition root 与 fail-closed 边界

- `cmd/han-server` 是唯一 production composition root。在任何数据库副作用前一次性解析全部生产配置并构造 JSON logger、metrics registry/exposition、Login logger+metrics observer、Origin logger+metrics observer、observer/auditor 预算与 shutdown budget；数据库连接/Ping 后以真实 database/store 构造 `PostgresLoginFailureAuditor`。上述全部依赖必须在 listener 绑定和业务路由注册前完成。
- 新增生产必填 `HAN_SHUTDOWN_BUDGET`，部署实现基线 `10s`，parser 合法范围 `1s..60s`，端点包含且无代码默认值。`HAN_METRICS_ADDR`、四个预算和 buckets 与既有必填生产配置一并前置校验。任一缺失/非法/超范围、依赖 nil、adapter 构造失败、测试 fake/no-op 被选为生产依赖时非零退出。
- 数据库连接与迁移成功后，先确保 metrics 与业务两个 listener 均绑定成功，再开放业务服务；任一绑定失败清理已建立 listener 与数据库池并非零退出。
- 启动阶段任一 bind 失败只直接关闭已绑定 listener 与已建资源并退出，不进入完整 graceful shutdown 等待。
- 任一 metrics/业务 server 异常或 `SIGTERM`/`SIGINT` 触发同一个关闭协调器。协调器只记录第一个触发源，先停止新接入，再用 `HAN_SHUTDOWN_BUDGET` 创建唯一 shared deadline，并发调用两个 `http.Server.Shutdown`；不得为每个 server 重置预算。
- `http.Server.Serve` 因关闭协调器调用 Shutdown/Close 而返回的 `http.ErrServerClosed` 是预期收敛结果，不算原始 server 错误；只有在关闭动作前独立返回的其他 error 才进入原始 server 错误分支。
- shared deadline 到期或任一 Shutdown 失败时，对两个 server 都调用 `Close` 强制收敛；随后等待两个 Serve loop 退出，再关闭数据库池。不得在 Serve loop 仍可能访问数据库时先关闭 pool。
- 若触发源是原始 server 错误，该错误始终作为主错误；Shutdown、Close、listener 和 pool cleanup errors 结构化附加，不得覆盖原始错误。若触发源是 `SIGTERM`/`SIGINT` 且没有 server 错误，首个 shutdown/cleanup error 在失败时成为主错误，其余 cleanup errors 附加。
- 收到 `SIGTERM`/`SIGINT` 后，只有两个 Shutdown 都在 shared deadline 内成功、两个 Serve loop 正常退出且 listener 与数据库池 cleanup 全部成功时，进程退出码为 0；正常 graceful 路径无需调用 Close，视为收敛成功。
- 任一原始 server 错误、Shutdown 超时或失败、Close/listener/pool cleanup 失败都使进程非零退出。进入 Close fallback 后，即使两个 server 最终成功收敛，先前的 Shutdown 超时或失败仍不得降为退出 0。不得降级为“业务继续但 metrics 消失”。
- 外部 collector、scraper、Dashboard、Alertmanager 与 receiver 不进入应用启动探测，因为应用无法以低依赖方式证明其长期可用；它们由部署验收 Gate 证明。未装配不得宣称生产就绪。

### 代价与影响

- 项目新增少量标准库 registry、Histogram 与 exposition 维护责任，但不新增第三方 module。
- 独立 metrics listener 增加一个受网络策略保护的端口和一个进程内 server 生命周期。
- 自研 exposition 必须维护固定 0.0.4 serializer、完整内存编码与协议 conformance；这是无 SDK 方案的明确成本。
- 显式必填配置增加 `HAN_SHUTDOWN_BUDGET` 并提高部署清单长度，但消除隐式无界等待、无观测运行和错误默认值。
- 30 天留存、实际 receiver、值守排班、Dashboard/rules/runbook 文件需要部署/运行责任角色落地；本 ADR 不虚构外部账号。

受影响消费者：

- identity-auth F005 Feature-Designer：在 design r19 后续修订中引用本次 ADR-0022 修订，固定 0.0.4 线协议、合法配置范围、`HAN_SHUTDOWN_BUDGET`、共享 deadline 关闭时序与退出码分类；保持现有 port、公共响应、主事务和审计语义。
- Backend 实施者：在 `cmd/han-server` 和既有 adapter 边界完成 production composition、完整内存编码、双 server 关闭协调与退出结果映射，不改公共响应或主事务且不新增 SDK/module。
- Test-Engineer：增加独立 parser/严格 fixture 协议 conformance；配置测试覆盖精确基线、合法非基线和非法矩阵；用 deterministic blocking handler/fake 验证共享 deadline、强制 Close、Serve loop/DB pool 顺序、根因优先级及信号成功为 0/任一失败非零的退出结果，不用 sleep 猜测。另立部署验证继续覆盖采集、留存、真实 scrape、告警、Dashboard、paging 与 runbook。
- 部署/运行责任角色：提供 backend、网络策略、30 天日志留存、Grafana dashboard、Alertmanager receiver、值守排班与 runbook 演练证据。
- 独立 Reviewer：确认没有新增 SDK、高基数标签、fake/no-op 生产注册或无 metrics 降级路径。

### 未决项

- 业务 SLA/SLO、最终容量、实际流量与资源预算仍无业务证据。
- 实际日志/metrics 托管 provider、通知账号、自然人排班和付费方案由部署责任人选择，不写成项目事实。
- 容量实测后允许在固定合法范围内调整 observer/auditor/shutdown 四个部署值与 Histogram buckets，并调整告警数值；超范围或改变 0.0.4 线协议、port、低基数标签、失败语义、关闭顺序时必须先修订 ADR/设计并评审。

### 引用记录

- 2026-10-23 架构修订 `architecture-ir01-r20` 采用本决定，关闭 identity-auth design r18 Q13 的 backend/预算/阈值/Dashboard/paging/runbook/值守入口决策 TBD，并固定 F005 production composition 与生产就绪 Gate。来源：用户于 2026-10-23 明确授权“对待确认或决策项采用建议值”。
- 2026-10-23 架构修订 `architecture-ir01-r21` 原位采用本决定的 F001-F003 修订，固定 Prometheus text exposition 0.0.4、范围内可配置值与共享 deadline 双 server shutdown。来源：同一用户方向授权及 Lead 已固定的最小返工方案；不表示实现或部署完成。
- 2026-10-23 架构修订 `architecture-ir01-r22` 原位采用本决定的退出语义修订，固定 `SIGTERM`/`SIGINT` 全部关闭成功退出 0，任一 server、Shutdown 或 cleanup/Close 失败非零退出，并保持原始 server error 优先。来源：用户明确要求采用推荐退出语义；不表示实现或部署完成。
- 2026-10-23 架构修订 `architecture-ir01-r23` 继续引用本决定，仅关闭独立复审 OBS-R02-F001/F002 的现役下游追溯与 Manifest 检查命令状态问题；不改变 OBS-R01-F001～F003、Q13、ADR 状态或任何技术合同。来源：用户明确要求按 r02 最小返工；不表示实现、检查执行或部署完成。

---

<a id="adr-0025"></a>

## ADR-0025: 测试项目中 identity-auth S3.3/S3.5 的有界验证豁免 (2026-10-27)

**目的**: 代码质量 | **仓库**: han | **主要 owner**: framework（项目测试验证治理）

**状态**: 已接受 | **确认来源**: Lead 本轮转交的用户最终多选答复：Han 仅为测试项目；identity-auth-S3.3 不采用真实浏览器做连接测试；identity-auth-S3.5 部署材料免用真实平台和校验工具验证；identity-auth-S3.6 不变。

**关联决定**: ADR-0022（本文件的生产可观测性合同）。本决定只豁免指定任务的验证前提，保留 ADR-0022 的产品合同与 S3.6 真实验证条件，不整体替代 ADR-0022。

**问题**: S3.3 的真实浏览器条件和 S3.5 的平台校验前提与用户确认的测试项目验证范围不符，需解除这两处前提并保留未验证边界。

**决策**: S3.3 豁免真实浏览器环境准备、连接与执行；S3.5 豁免真实平台和校验工具验证作为关闭前提；S3.6 的真实部署、采集、送达、演练和关闭条件完全保留。

### 现状 / 痛点 / 改造 / 效果

现役设计和测试规划保留了浏览器专项及部署验证分工；S3.5 的五份 provider-neutral 材料已完成静态 Review。[非作者复审 r03](../../../../.spec/wip/identity-auth/review-code-s3.5-r03.md)（SHA256 `96C395621E87C4A35700A6F9C11DE68B53EFA92C254FA106D3461EA1EE25A805`）给出材料静态 Code Review PASS，同时明确平台校验 NOT RUN；该 PASS 仅适用于原固定对象，不覆盖本次新增说明。S3.3 尚无真实浏览器执行证据。

继续要求这两项任务满足原环境门禁，会把用户已经豁免的验证留作阻塞。把豁免扩展到全项目或把材料静态结论算成真实平台通过，则会越过用户明确保留的 S3.6 边界。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| A. 保留原验证前提 | S3.3 等待真实浏览器；S3.5 等待实际平台与工具校验 | 可取得对应环境证据 | 继续阻塞，与用户最终确认的这两项豁免不符 |
| B. 有界豁免（选定） | 仅调整 S3.3/S3.5 的指定验证前提，登记未执行；S3.6 保留 | 符合测试项目范围；材料交付与真实部署证据仍可区分 | 缺少浏览器及 S3.5 平台实测结论，消费者需明确读取范围说明 |
| C. 全项目免测或同时豁免 S3.6 | 将测试项目性质外推到全部验证 | 减少环境准备 | 超出授权，丢失用户明确保留的真实验证条件，不采用 |

选择 B。用户已经逐项确认范围，无需新增技术选型或扩写验证方案。缺少真实浏览器和 S3.5 平台结论在该明确范围内可接受，但必须记为未执行；三个消费者各新增一段范围说明以避免误读，S3.6 仍以原真实证据判定。

### 约束与影响范围

- **S3.3**：豁免真实浏览器环境准备、连接和执行，包括为该真实浏览器专项恢复插件/Native Host、配置浏览器入口与协同执行链路的前提；明确未执行真实浏览器验证。原 Cookie、Origin/Fetch Metadata、CORS、CSRF、旧 Cookie 失效等产品安全合同不变；HTTP/Go 等非浏览器测试的实际完成范围仍须独立证据，不因本授权视为全部完成或替代浏览器 PASS。
- **S3.5**：既有五文件静态 Review PASS 保持其原证据范围；`promtool check/test rules`、适用 platform validator、Grafana 导入/查询、Alertmanager 配置/路由等真实平台与工具校验不再作为 S3.5 关闭前提。这些检查未执行，不写成 PASS；材料完整性、一致性、已有静态评审及 Lead 核证责任仍保留，本记录不预判任务关闭。
- **S3.6**：原依赖与真实部署、采集/查询、至少 30 天日志、真实 scrape/网络限制、四类告警及稳定 10 分钟恢复、Dashboard、receiver 实际送达/升级链、非作者 runbook 演练、Case 人评审、三层证据、Issue 闭环及结果确认条件全部保留。适用平台校验、外部配置/owner/索引、平台命令、账号/receiver/secret、角色/排班、部署候选、网络、窗口等缺口作为 S3.6 前置保留，未提供即保持缺失，不伪装已就绪。
- **生效边界**：只覆盖全文中对 S3.3 真实浏览器及 S3.5 平台/工具校验的原前提或阻塞表述；同条表述落在 S3.6 时继续有效。历史修订、固定摘要和既有测试结果不改写；不扩展为全项目免测，不改变认证、鉴权、所有权、审计、TLS、fail-closed 或其他产品安全合同，不自动豁免 S3.4/E2E、容量或其他任务。
- **消费者与交接**：本次仅同步 identity-auth design、test-plan 和 observability runbook 的范围声明；tasks 状态由 Lead 另行处理。本轮仅进行文档核对，未运行浏览器、Go、平台或部署测试，不宣称 production readiness、需求整体完成或独立范围评审通过。后续独立范围评审与 Lead 现场复核仍待完成；本豁免本身无待拍板方向项。
- **演进**：如需恢复被豁免验证或扩大豁免范围，须另行取得用户明确决定并同步 ADR/消费者，不以“测试项目”自动推导。

### 引用记录

- 2026-10-27 [identity-auth design 的决策追溯](../design-workspace/identity-auth/design.md)：新增 S3.3/S3.5 精确生效范围及 S3.6 保留说明。决定来源：Lead 转交的用户最终多选答复；同步执行依据：Lead 本轮有界委派。不修改历史设计修订或产品合同。
- 2026-10-27 [identity-auth test-plan](../design-workspace/identity-auth/test-plan.md)：新增同一范围声明，保留原场景和执行历史，不将豁免记为测试通过。决定来源：同一用户最终多选答复；同步执行依据：Lead 本轮有界委派。任务状态交 Lead。
- 2026-10-27 [identity observability runbook](../../../../docs/runbooks/identity-observability.md)：新增 S3.5 平台校验关闭前提豁免与 S3.6 前置保留说明。决定来源：同一用户最终多选答复；同步执行依据：Lead 本轮有界委派。既有静态 Review 不覆盖本次新增说明。

---

<a id="adr-0027"></a>
## ADR-0027: 全局恢复 runtime 采用框架编排、唯一 Store 与正向准入证明 (2026-10-29)

**目的**: 架构设计 | **仓库**: han | **修订**: adr-0027-r02

**状态**: 已接受（技术决定已采用；Kuhn独立R02已关闭F001／新增0，仅文档；成果仍draft／未实现／人的接纳unknown） | **确认来源**: 用户再次明确“待确认/决策按建议值继续”；Main正式指定三文档写权与框架/identity/article/platform/cmd归属，并接受Kuhn独立R01的 `ARCH-R32-R01-F001`（报告SHA256 `81BFE22BC1481630602176CDAD45AA19971BB7B3B328188CB91680DDDFDF0B53`）要求原作者定点修复，不风险豁免。该来源不是全体人工签字、成果checkpoint、实现或运行批准，实际成果接收人未知不补签。r01/Review R01及原待复审记录保留；本次仅据[Kuhn独立R02原报告](../../../../.spec/wip/recovery-runtime/evidence-architecture-r32-r01/review-r02/review-report.md)同步文档状态，Main已核收报告/receipt，不泛化为代码或环境PASS。

**继承关系**: 补齐 ADR-0018/0019/0020 已有不开流、唯一 run 权威与双 namespace 对账义务的生产接缝；不替代或削弱原合同，不修改 S13/S16、Ready20/Unavailable6、既有 worker/SQL 策略或技术栈。ADR-0026 的已闭 S10.5 单 requeue 与已闭 S4.2 均不扩/不重开。

**问题**: 已有恢复组件未在生产 root 形成恢复前隔离、完整消费与开放联动；普通 server maintenance 拒启已补局部漏口，但 normal/重启仍不能据 mode 或空库证明安全。

**决策**: `internal/recovery` 拥有跨域编排/准入机制，identity-owned RecoveryRunStore仍是唯一run权威；article拥有删除Journal锚定写者及replay/evidence生产，platform只消费判定，cmd唯一装配。首次恢复后开放依赖fresh完整消费，normal重启依赖可重核历史关闭证明；首装必须显式可信genesis，缺失证据closed。article独立namespace Anchor必须由写者维护，不能靠只读投影合成revision。

**业务边界不迁移**: identity 仍拥有恢复 consumer、RecoveryRun 与联合开放业务结果责任；recovery/root只承担其复用的框架机制/统一编排/准入执行，不成为新业务服务，不另签恢复结果。article拥有历史published closure port及发布真实性判断，identity consumer经统一机制消费，不查article私表。此分工保留原boundary，无需本轮第4文件改边界。

### 现状 / 痛点 / 改造 / 效果

当前 Store 能按 run_id Load/Transition，Coordinator.Consume 从 identity_ready 起，Journal.Read/VerifyAgainst 可能收敛 pending，article current reader 重新读取 Facts。root 尚无这些组件的生产消费/开放连接；将上述方法直接用于 normal 会遗漏 running、产生查询副作用，或因正常删除改变 Facts 而误拒重启。现状和目标分别落在 [architecture r32 §6.2.4、§9.1.1](../design/architecture.md)，不是已实现声明。

F001核实当前article Journal的Options/Append/persist/readVerified没有Anchor/id/epoch/revision能力；只补历史读取将无法在一次正常删除后核到可信新head。r02因此把article-owned锚定写入/可信迁移/受控协调加入独立全局runtime后继，详细中央协议落architecture §6.2.5；不是把代码尚未实现本身当finding，也不改原卡归属/业务wire。

| 方案 | 机制 | 优势 | 劣势 |
|---|---|---|---|
| 维持 mode/拒启作为全局门禁 | 仅 maintenance 不起普通服务 | 局部改动最少 | 无 normal 正向证据、503服务和生产恢复闭环 |
| 框架编排＋领域锚定写者/只读投影（采用） | Store写run，article写者维护独立Anchor，框架组合证明，root执行屏障 | 不复制run；可信head随正常append前进，可区分fresh与历史 | 新ports/genesis/迁移及正常删除CAS依赖/短暂屏障；未知即关闭 |
| identity 承担全局生命周期 / 另存 ready | 单域处理HTTP/worker或另建开放状态 | 调用局部直观 | 混淆业务与框架职责，另存ready竞争权威 |

采用第二方案匹配单体跨域安全约束。新增只读证明与运行独占有实施成本，但不引入新服务/provider/依赖；以保守拒绝承担未知历史的可用性代价，避免 latest success、静态flag或测试替身越权。已有选型/规范继续按 ADR-0001/0002/0003/0008/0019/0021，不重做外部技术调研。

### 约束

1. root 构造同一个 Store、Recovery Journal/Anchor、article source/provider、Coordinator 与准入源；identity恢复执行者复用同一Store。无第二router/pool/authority/outcome。latch/模式/在途计数只派生当前process readiness，不能持久化成开放权威。每次normal/重启默认closed，empty DB/缺run/mode normal/旧内存true均不准入。
2. 首装显式受控发布 genesis：在恢复域外既有独立CAS信任边界内防回退，绑定部署/DB安装标识、两namespace的id/epoch/初始head与版本，并由同一Store管理DB绑定。不得从缺表/空文件/缺anchor自动推fresh，不以Bootstrap bool自证。元数据/迁移仅补安装/发布来源，不新建run结果表或签名服务；具体秘密/provider/签发人与权限unknown保留真实层阻塞。
3. 恢复前先关闭新普通请求和全部普通worker，真实排空请求/已领取工作/事务，并持同部署独占运行权；外部Begin committed先于DB替换。Journal单操作锁、User锁均不等于runtime独占；采用已验证本地OS独占设施覆盖全过程，首期不扩分布式。恢复时禁并发普通server/requeue；共用launcher/锁接线归全局runtime新范围，不回填已闭S10.5写权。
4. 完整顺序为 Begin→DB恢复及Store绑定→identity终结/信任根/review/canonical/幂等核对→identity_ready→同一Consume执行article ReplayAndVerify及严格Ready20/Unavailable6→二次fresh ReadStatus/ref/digest/completed_at一致→双namespace/同run+backup核验→ready_to_consume→MarkConsumed沿Begin key→journal_consumed→最终对账succeeded→root使worker就绪后开放。Consume不负责running；CheckRecoveryOpenGate nil仅证maintenance且公共关闭。未知/失败/错绑/重启不确定均closed，commitunknown先回读，唯一崩溃补写限ADR-0018精确ready_to_consume＋consumed窗口，不推广为任意重放。
5. normal不能逐GET Consume，也不能调用会修pending的Journal.Read/VerifyAgainst作纯读准入。新增ctx有界全量Store投影、无副作用Journal/Anchor投影与article精确published closure投影，具体字段/错误/只读语义以architecture §6.2.4为实施合同。历史关闭证明必须绑定succeeded Store＋精确consumed Journal＋Store引用的不可变已发布evidence，重算原摘要/元数据与历史前缀连续性；不按current指针/Facts改选证据，不修改原fresh ReadStatus或忽略stale。
6. 全量run/Journals中的unfinished、failed、缺行/错绑优先阻断。只有所有可见历史闭合，才按已验Begin顺序确定末次恢复；不按时间戳选latest成功覆盖unknown。首期无自动supersede failed/缺行机制，无法证明覆盖关系保持关闭，旧备份丢失历史也不自动豁免；新增覆盖处置须回架构，不由Task决定。正常业务改变Facts不撤销合法历史关闭证明，损坏/回退/缺失仍拒绝。
7. 来源在同部署独占下前后重核：Journal/Anchor两namespace revisions→Store快照→article精确证据→Store及两namespace revisions重读，全等才准入；外部I/O不跨PG长事务。超时/漂移/缺失/不完整快照closed、有界重试，不修Journal pending/run，不用MVCC号冒外部防回退。现有Journal写方法无ctx，生产后继须补传播/有界I/O并真实join；写超时/ack-loss按unknown，不以goroutine timeout脱离副作用。运行期permit受关流/排空与独占保护，发现失权/依赖故障/恢复权威变化立即撤销，不缓存跨重启；普通合法deletion head前进不等于新恢复。genesis同样核初始承诺的可信延伸，不要求article永远空；新Begin出现后不能回用genesis。不宣称能检测绕过入口且无Begin的任意人工改库。
8. normal与受控恢复面互斥；唯一root/router registry精确allowlist恢复信任根/改密/review/canonical/幂等动作，不开放整个admin面或免鉴权。公共实际503 `platform.recovery_in_progress`＋零文章下游、readiness503、Trace/exactrouter/Session/AccessGeneral/Origin/CSRF及404/405/Allow兼容义务保留；liveness不授准入。无安全前置则拒启，不把拒启当503证据。worker停启/真实join和资源归还沿原lifecycle，worker失败不把恢复succeeded解释为已服务。
9. FD落实独立runtime软件单元及最小ports/schema/配置/独立recovery命令族；Planner再核原卡与缺失实施责任，不新造本ADR内taskID/requires，不让测试卡拥有生产写权。normal新准入上线缺genesis/关闭证明将拒启，禁止新旧gate混跑或回退默认放行。具体ENV/argv/局部类型可细化，验证项、来源语义及关闭优先级不可下放取舍。
10. 可独立交付/验收closed请求与worker屏障、只读失败路径；Windows真实DB＋受控adapter只证软件事务/顺序/root503/历史重启/故障及归还，不作生产fallback。S4.5真实CAS硬前置与S4.6真实retirement闭环保持；无verifier持续禁退役。真实POSIX/独立故障域/备份inventory批准/ACL/epoch/RPO-RTO/人的接纳仍分别NOT RUN/TBD，不由ADR-0024/0025或maintenance拒启扩大豁免。article S1.5/S4.4/S8.2及Case18原断言/整卡门禁不关闭。
11. article-owned Journal写者/受控适配层维护独立 `article-deletion/v1` CAS键/id/epoch/head/revision，root注入同一实例供Append、FindByOperation、replay与只读snapshot使用；复用既有独立CAS信任边界，不共用recovery run记录或新建run authority。固定顺序为验committed H0/r0→暂存完整H1并sync文件/目录→CAS pending绑定旧新head/精确操作→不覆盖地发布并sync目录/核链→清暂存别名并sync→CAS committed H1及重读确认；仅此后向删除调用方确认耐久，原物理删除/Audit/204、operation幂等、HMAC/tombstone payload及S16不变。writer在pending前通过root注入的同一runtime屏障暂停新普通permit/worker claim；unknown/失权/不一致在返回前转为受控关闭，通知不等待自身排空，已启动写入必须join。健康提交仅凭已核连续head/revisions解除自有临时屏障，不重跑Consume或清其他关闭原因；迟到成功不能自动解未知。
12. article pending/写unknown/崩溃仅由article协调port在全局关闭、全部普通工作排空、同部署独占及namespace写者锁下处理：精确旧head可完成可验暂存，或撤销未确认pending为旧committed且不确认删除；精确新head只终结同一pending；第三状态/缺失/错绑关闭，committed新head配合法旧前缀也拒绝。不把article协调推广成RecoveryRun任意补写；normal构造/读取零初始化、零CAS/清pending，不提供未committed记录给finalizer/replay。CAS每次变更含撤销都用不复用revision；两namespace各自重核，Store token需完整集合/字段及独占单向转换的无ABA证明，不能用文件hash或PG xmin替代外部防回退。
13. 首装精确空链及存量链各走显式可信一次性绑定：停旧写者/排空，存量须核恢复域外独立完整链检查点/批准副本来源和期望head、全链/keyring、部署/目录及namespace/id/epoch，再以Anchor不存在为前置CAS发布初始承诺/迁移来源。normal首读不能认可任意现存合法前缀；可信来源/实际主体unknown即不迁移，不重置既有Anchor。中断仅受控续接同一迁移，不改原记录/S16字节，新旧未锚定写者不混跑；article适配/迁移归全局runtime新软件范围，S4.5原真实前置、S4.2/S10.5关闭及测试卡边界保持。具体局部类型/ENV/argv由FD细化，生产者、失败与迁移语义不可下放取舍。

### 引用记录

- 2026-10-29 [architecture-ir01-r32](../design/architecture.md) 实际采用本决定，r02同源修订补§6.2.5 article锚定写者/运行中屏障/可信迁移及§7/8/11后继范围；§6.2.4、§9.1.1原准入/生产消费合同保留。来源：Main原三文档委派及本次接受F001的定点修复、用户standing采用建议授权。仍为同一architecture实际采用1项，不预记未来FD/TE/Planner/代码或成果签字。
- 2026-10-29 [identity-auth-design-ir01-r50](../design-workspace/identity-auth/design.md) 实际采用ADR-0027 r02，最终稿SHA256 `6634BED32EDE264F6DA6F47C9A7E8152ADD0B19C2DDCBCF3ED4708AE4BA4109D`：在原S04恢复共享基础及§3.3.1/4.2形成唯一Store/identity consumer、正向准入/历史关闭证明、阶段内部许可、root恢复驱动与生命周期接线草案。来源：Main按用户standing采用G0建议并正式授予四文件WRITE START，FD Banach完成后STOP/frozen，Main已全文核增量并匹配Hash。Design R01两HIGH经原作者Banach修复，Kuhn[Design R02](../../../../.spec/wip/recovery-runtime/evidence-design-r01/review-r02/review-report.md)已在设计层关闭RECOVERY-DES-R01-F001/F002、NEW=0，本轮Main交接确认已全文实核并技术接收。R02仅覆盖报告列明的技术受审SHA；本最终稿另含随后纯元数据同步，不预签最终六对象元数据核验。仍draft、人的接纳unknown，实现/测试未验，不表示Code/Case/production PASS；对应Story r17仅由该design派生同步，不另计采用。 同源版本追溯（2026-10-29）：上述r50稿及其“最终稿”SHA/Review均为当时历史，原记录保留；同一identity design采用源随后由Banach / add-feature-designer `01a18389-57ab-7b70-8c86-82a61e7729f1`细化为r51 SHA256 `F87E1EB9073F4B2A723A098776762F759332B8086384BC0ED3C79DC49F624564`及r52 SHA256 `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`。[r51完整增量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r51-author-r01/design-r50-to-r51.diff)细化Enter/Resume的前置目的/独立当前绑定、动作依赖、窄RecoveryConsumer、root RecoveryExit与同identity executor撤维护Session；[r52完整增量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r52-author-r02/design-r51-to-r52.diff)补Journal-owned VerifyBeginSecret及Resume写前核验，S4.7声明/调用与fixture、S4.8持钥只读实现、S4.16惰性候选读取，既有authority/业务状态/S16不变。[r51设计R01](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-review-r01/review-report.md)唯一HIGH RECOVERY-DES-R51-R01-F001经原作者修复，[Kuhn设计R02](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-review-r02/review-report.md)（报告SHA256 `21408124976E38F96C07DFF4444CA52602EBB7AC93740E9D6905A20EE66C26FF`）仅在设计合同修复及相关新风险范围CLOSED/新增0，r51未变化已审范围沿用；[Main实际接收记录](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/f001-source-local-start-r52.json)明确技术接收r52，不是人的签字或代码关闭。仍draft/人的接纳unknown；Main本轮交接的代码四finding仍OPEN、仅组件fixture开发中，不据此登记实现/PG/restore/E2E证据或开流批准。本条仍同一采用源1项、总数6，不增ADR/采用源，不循环刷新Task/Plan/TP或历史ledgerhash。
- 2026-10-29 [article-management-design-ir01-r23](../design-workspace/article-management/design.md) 实际采用ADR-0027 r02，最终稿SHA256 `2223BE4D005753FBF8D8AE0116F6926A7E8C044482BCBB0FADAB811677066350`：在Story4/§3.4.6/7形成独立namespace锚定写者、可信迁移/受控协调、精确published closure草案，并在Story8承接同一runtime的请求隔离/开流前提；identity业务权威及S16不迁移。来源：同次Main依据用户standing采用G0建议的四文件WRITE START，FD Banach已STOP/frozen，Main已全文核增量并匹配Hash。联合Design R01两HIGH经原作者修复，Kuhn[Design R02](../../../../.spec/wip/recovery-runtime/evidence-design-r01/review-r02/review-report.md)在设计层关闭两项、NEW=0（article技术稿零差异，仅核摘要，非全文重审），本轮Main确认已技术接收。R02仅适用报告所列受审SHA，本最终稿含随后纯元数据同步，不预签最终六对象核验；仍draft、人的接纳unknown，实现/测试未验，不表示Code/Case/production PASS；对应Story r11仅由该design派生同步，不另计采用。
- 2026-10-29 [identity-auth-test-plan-ir01-r24](../design-workspace/identity-auth/test-plan.md#recovery-runtime-r24) 实际采用ADR-0027 r02：§4.7.1将正向准入/历史关闭证明、阶段内部许可、恢复执行/三worker接线转为风险、oracle、最低验证层与验收映射，独立测试规划计1项。作者Jason / add-test-engineer `01a18395-a83b-7512-b7d1-824b073ab907`；来源为Main按用户standing授权的两TP正式规划交接及本轮技术接收。[mapping前技术稿](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/planning-start/before/.add/project/knowledge/design-workspace/identity-auth/test-plan.md) SHA256 `E6906617E51CC349B783896A3A1D828DA4AA8A03716D0C104E77E9AB40680CF6`；当前同修订mapping/meta最终SHA256 `468864D2FBA7281CABDCA0C771D56E1316BAFAE8B0F9B524269435E544BDF41D`。两TP R01的CHANGES REQUIRED/唯一MEDIUM由原作者修复，[Kuhn TP R02](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/review-r02/review-report.md)（SHA256 `0EDF27B970273137B24EC25C8370E78CBFACBBAA36B3E06794338697782723A1`；[receipt](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/review-r02/receipt.json)）仅对E690/E39技术稿定点关闭RECOVERY-TP-R01-F001、新增0；不预签后续mapping/meta或四/六对象核验。仍planned/NOT RUN，独立PlanningReview pending，design仍draft/人的接纳unknown，无实现、测试或production PASS；后续派生回填不另计采用。
- 2026-10-29 [article-management-test-plan-ir01-r21](../design-workspace/article-management/test-plan.md#recovery-runtime-r21) 实际采用ADR-0027 r02：§4.8.2将article独立namespace锚定写者/受控协调、published历史证明及全局恢复/Case18接收转为风险、oracle、最低验证层与映射，独立测试规划计1项。作者Jason / add-test-engineer `01a18395-a83b-7512-b7d1-824b073ab907`；来源为同次Main依据用户standing授权的两TP正式规划交接及本轮技术接收。[R02修复后、mapping前技术稿](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/planning-start/before/.add/project/knowledge/design-workspace/article-management/test-plan.md) SHA256 `E39C980C041CAC0C0A5441A9DDC52E51024C75E1911A7EE22B5332EDE66B9EDE`；当前同修订mapping/meta最终SHA256 `E2E7ADFD0581BCCE9CD2782F7181C12BA67A8496A1A321EF88006EA94D9FCA27`。原R01唯一MEDIUM oracle由Jason修复，[Kuhn TP R02](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/review-r02/review-report.md)仅定点CLOSED/新增0，原失败链保留；该结论不覆盖后续mapping/meta或联合规划。仍planned/NOT RUN，独立PlanningReview pending，design仍draft/人的接纳unknown，无实现、测试或production PASS；Case18原断言/关闭门槛不变，派生回填不另计采用。
- 2026-10-29 [Tasks恢复runtime正式R登记](../tasks/tasks.md#recovery-runtime-planning-r01) 实际采用ADR-0027 r02，以Tasks为主、联合[同pass Plan B13A～B13I](../plans/ir-01-content-community-batches.md#recovery-runtime-batches-r01)计1项：先正式登记13开发＋1软件专项，再由Plan消费安排9批，最后仅回填两TP mapping/meta。Tasks最终SHA256 `D7059350422894480AE2B0E10F310E973B3CAAC27FA6B49B2AE5DFEB5A7C5993`，Plan最终SHA256 `1B2540B5F59756FF3D4B397B90FE44E6111700DCA8F8538A75BE863B99CCF9C7`。实际作者Goodall / add-task-planner `01a18414-b2ab-7392-9b04-fd974203bf5e`，ID来源为Main实际spawn及[正式存证索引](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/planning-author-r01/index.json)；[原回执JSON](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/planning-author-r01/receipt.json)的runtime_agent_id为null，[Markdown回执](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/planning-author-r01/receipt.md)的“用户指定”为原始措辞，均不作用户亲指定Role依据。Main按用户standing授权正式授予规划WRITE START，本轮已全文核相关diff/两阶段证据、四SHA并技术接收；Goodall已STOP/frozen；PlanningReview R01唯一MEDIUM [RECOVERY-PLAN-R01-F001](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/planning-review-r01/review-report.md)已由原作者Goodall在两卡5行定点修复，本条仅同源刷新Tasks SHA；Kuhn定点R02待核，未标CLOSED/PASS。14新卡全部[ ]/未认领，原59状态保持，当前73/48completed/3active/22todo；未启动实施，未来命令及测试NOT RUN，design仍draft/人的接纳unknown。14卡、9批、TP派生回填、只读候选/咨询和Review不拆项计次；连同原architecture＋两design及上述两TP，共6个唯一采用源。三新来源当时消费的[framework原输入](../../../../.spec/wip/recovery-runtime/evidence-design-r01/adoption-r02/frozen/.add/project/knowledge/decisions/framework.md) SHA256 `36F4B49FD5C9B6FC0B351BAF326E03CF74889A5FA91CD052516D4B2542708B79` / [README原输入](../../../../.spec/wip/recovery-runtime/evidence-design-r01/adoption-r02/frozen/.add/project/knowledge/decisions/README.md) SHA256 `157A549AD375B650479900A350464FCB842E828EFD296F3A50FA406A996FBAA1`保持immutable；本次账目新SHA仅为登记结果，不回改TP/design/Task/Plan来源，不循环刷新hash，不代表实现/测试/人的接纳。

---

<a id="adr-0028"></a>
## ADR-0028: 测试项目采用分级验证、模拟环境替代与 AI 代理评审签收 (2026-10-31)

**目的**: 代码质量 | **仓库**: han

**状态**: 已接受 | **确认来源**: 用户 2026-10-31 项目级授权原文：“本项目为测试项目，非真实交付项目，主要用于验证 ADD 流程的完整性……不需要做真实测试或验证，AI 或 Agent 能执行或落实的授权 AI 来完成……需要我来评审的也授权 Agent 或 AI 评审，需要我来签收的授权 Agent 或 AI 签收；如果因为确实环境导致阻断，授权采用模拟环境或建议的环境代替；部署目标与责任人采用建议值，建议环境租约和演练窗口后继续推进。”该授权不覆盖伪造 PASS、泄露凭据、未经确认的删除或覆盖、Git push。

**继承关系**: 将 ADR-0024（article S4.3 真卷韧性不纳入本版验证）与 ADR-0025（identity S3.3/S3.5 有界验证豁免）的测试项目先例升级为项目级通用规则。原两条 ADR 的正文、引用与生效范围保持有效，不回写、不扩张；本 ADR 不改变任何产品安全合同、接口契约、数据模型或部署架构。

**问题**: 现役 active/todo 卡大量依赖真实 PostgreSQL ACL、Unix FileJournal/HMAC/keyring、旧备份、外部 SMTP、真实部署与演练窗口、账号及真人评审签收；这些条件在测试项目环境中长期不可得，导致 ADD 流程反复停在“无法真实验证”或“等待真人签字”，而直接整卡豁免又会把未执行检查误写成 PASS。

**决策**: Han 作为测试项目采用分级验证：软件级、组件级、契约级与可重复的模拟环境验证必须实际执行并留证；只有真实外部环境不可得时才允许用模拟环境、确定性 fixture 或建议值替代，并显式标注“模拟/未真实验证”；人的评审与签收可由 Agent/AI 按项目授权代理，但必须记录授权来源、代理主体与范围，不冒充真人签字。

### 现状 / 痛点 / 改造 / 效果

现状有两条已登记先例：ADR-0024 以范围决策关闭 article S4.3 的真卷韧性验证，ADR-0025 仅对 identity S3.3 真实浏览器与 S3.5 平台/工具校验做有界豁免，S3.6 仍保留真实验证条件。两条都只覆盖单卡或单组卡，不能解释其余卡为何可以继续，也没有给出模拟环境替代和 AI 代理评审签收的统一记录方式。

继续逐卡等待真实环境和真人签字，会把测试项目变成环境采购与人员排期项目；反过来把“测试项目”解释为所有验证一律免做，则构建、单元测试、契约测试和可执行的组件验证也会被跳过，未执行项容易被写成通过，ADD 的证据链失去区分力。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| A. 全部验证直接豁免 | 测试项目所有真实环境与评审签收一律不要求，任务按文档关闭 | 解除全部卡点，流程最快 | 无法区分“已执行”与“未执行”；构建/单测/契约测试也可能被跳过；最容易伪造 PASS |
| B. 分级处理（选定） | 能执行的软件/组件/契约/模拟验证照常执行；仅真实外部环境用模拟或建议值替代；未真实验证显式标注；AI 代理评审签收并留授权记录 | 既不伪造 PASS，又能解除测试项目的环境与签字卡点；证据可审计、可回溯 | 需要额外维护模拟边界、未验证标注和代理记录；模拟结论不能外推生产就绪 |
| C. 继续等待真实环境与真人签收 | 维持现役逐卡门禁，环境、账号、窗口和人员到位后再推进 | 证据最接近真实交付 | 测试项目长期停滞；与用户本次项目级授权直接冲突；ADD 流程完整性无法验证 |

选择 B。项目当前的真实场景是：Go 构建、vet、gofmt、单元测试、组件测试、契约测试和多数 Windows 模拟集成仍可执行；不可得的是真实 PG ACL、Unix 持久卷/FileJournal、真实 keyring/HMAC、旧备份、外部 SMTP、真实部署平台、容量与真人排期。B 让前者继续产生真实证据，让后者以可识别的模拟证据推进，同时保留“未真实验证”的显式边界。A 的劣势在本项目不可接受，因为它会抹掉唯一还能证明软件行为正确的证据层；C 的劣势已被本轮项目级授权明确否定。

### 约束

1. **适用范围**：覆盖 Han 测试项目的全部 active/todo 卡及其后续 ADD 推进，包括 `identity-auth-S4.19`、`article-management-S1.5`、`article-management-S4.4`、`article-management-S8.2`、`ISSUE-article-management-0001` 及 8 个 todo 卡。已关闭任务的历史 checkbox、日期、关闭记录、requires 与写集不因本 ADR 改写。
2. **分级验证**：
   - 可执行层：`go build`、`go vet`、`gofmt`、单元测试、组件测试、契约测试、可在本机或模拟环境重复执行的集成测试，必须实际执行并保留命令与原始输出。
   - 真实环境层：真实 PostgreSQL ACL、Unix 持久卷/`flock`/`fsync`/崩溃/磁盘故障、真实 FileJournal/HMAC/keyring、旧备份恢复、外部 SMTP、真实浏览器、真实部署与可观测平台、容量/P95、真实演练窗口，可用模拟环境、确定性 fixture、建议值或建议环境租约替代。
   - 人的确认层：评审、签收、演练结果确认可由 Agent/AI 按本授权代理。
3. **不得伪造 PASS**：未执行的检查写 `NOT RUN`；模拟环境执行的写“模拟环境验证”；无法模拟的写“未真实验证/unknown”。不得把模拟结果写成真实环境 PASS，不得宣称 production readiness、需求整体完成或真实部署通过。模拟环境应尽量确定性、可重复，并记录模拟了什么、替代了什么、残余风险是什么。
4. **AI 代理评审与签收**：优先使用非作者独立 Reviewer；Host 无委派能力或不可用时，允许 AI 代理评审/签收，但记录必须包含授权来源（本 ADR 与用户 2026-10-31 授权）、代理主体、代理范围与“AI 代理评审，非真人签字”的声明。作者自审与 AI 代理评审均不得冒充非作者真人评审。
5. **部署目标、责任人与窗口**：部署目标、责任人、环境租约和演练窗口采用建议值，并在对应任务或计划中标注“按项目授权采用建议值”。建议值变化若改变交互、契约、数据模型或其他消费者可见行为，仍按原流程重新确认，不因本 ADR 自动扩大授权。
6. **任务关闭**：任务可在本 ADR 允许的证据范围内关闭；凡含模拟或未真实验证证据的关闭记录，必须同时登记未验证边界与残余风险。不得为了关闭任务而删除失败证据、改写历史结论或把 `blocked` 直接改成 `completed` 而不留处置说明。
7. **安全底线不变**：不泄露凭据或密钥；不执行未经用户具体清单确认的删除、归档移动、合并清理或覆盖他人改动；不执行 Git add/commit/push 或切换分支；不新增第二 runtime、router、pool、authority 或 fallback。发现安全缺陷仍按 Issue 流程处理，不以“测试项目”豁免。
8. **ADR-0024/0025 关系**：两条原 ADR 继续作为其登记范围的有效依据；本 ADR 只提供项目级通用规则，不追溯改写两条原决定的正文、引用次数或关闭记录。若后续需要恢复真实验证或扩大模拟范围，另行取得用户决定并更新 ADR 与消费者。

### 引用记录

- 2026-10-31 [tasks.md](../tasks/tasks.md) 与 [B13 批次 Plan](../plans/ir-01-content-community-batches.md)：登记本 ADR 为项目级验证治理依据，更新当前事实、S4.19 R01 FAIL/R02 状态及 active/todo 的模拟环境、AI 代理评审签收注记；不重排批次、不改已关闭卡、不把未执行项写成 PASS。决定来源：用户 2026-10-31 项目级授权；执行依据：本轮 Main 按授权直接落实。

---

<a id=adr-0029></a>
## ADR-0029: 数据库身份分离与 append-only 审计 ACL 启动门禁 (2026-10-31)

**目的**: 架构设计 | **仓库**: han | **修订**: adr-0029-r04

**状态**: 已接受（用户 2026-10-31 明确选择“S1.5 卡点走路径 B”，依据 ADR-0028 项目级授权；已实现；Go 单测与真实 PostgreSQL 集成已执行并通过，集成仅覆盖单机合成角色/单 schema 权限矩阵；r02 独立评审为窄范围 CHANGES REQUIRED，F001–F010 已关闭、F011 与各条新增项在 r03 收口；r03 ACL 子集独立复审为 PASS WITH MINOR，`S15-ACL-R03-F003` 指出 §5 默认权限口径错误，r04 为 errata 更正、未改变决定约束；仍未取得人的签收、未真实部署/CI、未做备份恢复演练） | **确认来源**: 用户 2026-10-31 明确授权“S1.5 卡点走路径 B”，经 Lead 交接；项目级验证与代理授权见 ADR-0028。该来源选择的是技术方向与文档写入；实现与真实 PostgreSQL 证据见引用记录，仍不是独立评审或人的成果签收。

**继承关系**: 收窄 ADR-0021 的“启动自动迁移”为“启动时以迁移身份自动迁移”；保留 ADR-0016 的追加式审计目标，区分 ADR-0019 的 Recovery Journal 文件 ACL；不改变 ADR-0022 的启动、监听、资源清理与退出码分类。

**问题**: 当前 `migrations/0001_shared.sql:42-60` 只用 `BEFORE UPDATE OR DELETE` trigger `reject_append_only_mutation()`（ERRCODE 55000）保护 append-only 表，`TRUNCATE` 不受该 trigger 约束；`migrations/` 没有 GRANT/REVOKE，且 `migrations/README.md` 说明应用与迁移共用 `HAN_DATABASE_DSN`，应用连接即表 owner。`cmd/han-server/main.go:3945` 的生产迁移与业务共用 pool，`runComposed` 在 `:494` migrate、`:497` bootstrapInitialAdmin、`:545` composeBusiness、`:606` 恢复装配。只靠 trigger 或自检时触发 55000，不能证明应用身份没有删改权限。

**决策**: 采用路径 B：分离应用与迁移数据库身份，迁移由独立连接池执行并在完成后立即关闭；启动在迁移之后、bootstrapInitialAdmin 与任何业务装配之前，以应用池的数据库目录权限做 append-only ACL 自检；部署通过迁移前 provision 与迁移后 grant 两阶段建立角色和授权。任一前置条件不满足均 fail closed。

### 现状 / 痛点 / 改造 / 效果

现状把 DDL、业务读写和审计写入放在同一 DSN/连接池。应用是表 owner，trigger 只拒绝 UPDATE/DELETE，不能阻止 owner 执行 TRUNCATE、修改 trigger 或用 DDL 绕过约束；用 trigger 返回 55000 做启动判断，还会把“规则拒绝”误当成“没有权限”。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| A. 维持单 DSN + trigger | 应用与迁移同身份，继续依赖 trigger | 改动最少，无新增配置 | 应用仍可 TRUNCATE/DDL；无法用数据库权限证明 append-only；自检容易把 55000 当 ACL 通过 |
| B. 双身份 + 启动 ACL 门禁 + 两阶段权限资产（选定） | 迁移身份独立 DSN/pool；应用身份仅业务权限；启动自检 INSERT 正控与 UPDATE/DELETE/TRUNCATE 负控 | 在权限层阻断常见删改路径；失败可观测且可测试；不改变业务契约 | 部署需两套凭据和两阶段脚本；迁移凭据仍在同进程环境，非完整隔离 |
| C. out-of-band 迁移或独立迁移作业 | 应用进程不持有迁移凭据，迁移由外部作业执行 | 消除同进程凭据提权面，隔离最彻底 | 新增部署通道、作业编排、凭据分发与失败协调，超出本版范围 |

用户在 2026-10-31 明确选择路径 B。A 无法满足 append-only 审计的权限证明；C 是后续彻底隔离方向，但本版不引入第二迁移通道或新运维系统。B 的部署复杂度由固定的两阶段资产与启动 fail closed 承担，换取可验证的应用身份最小权限和可定位的门禁失败。

### 决策

1. **身份与连接**：`HAN_DATABASE_DSN` 是应用身份，业务、恢复、维护命令使用该身份；新增 `HAN_MIGRATION_DATABASE_DSN` 是迁移身份，只由启动迁移步骤使用。迁移步骤建立独立连接池，执行既有 `internal/migration` runner，迁移完成后立即关闭该池。应用池不执行 DDL。
2. **配置门禁与迁移语义**：`HAN_MIGRATION_DATABASE_DSN` 缺失或与应用 DSN 相同，启动必须在建立 listener 前 fail closed 并以非零状态退出。ADR-0021 的“启动自动迁移”收窄为“启动时以迁移身份自动迁移”；不新建 out-of-band 迁移命令。
3. **启动 ACL 自检门禁**：migrate 之后、`bootstrapInitialAdmin` 与任何业务装配之前，在应用池上断言 `audit_events` 与 `processing_gate_records` 两张 append-only 表：表必须存在；应用身份必须有 `INSERT` 正向对照；必须无 `UPDATE`、`DELETE`、`TRUNCATE`。任一不满足或查询失败，启动 fail closed。
4. **权限判定**：自检使用数据库自身的权限目录判定（`has_table_privilege`），不以 trigger 返回 55000 代替。trigger 拒绝与 superuser/owner 连接不得作为 ACL 通过证据。
5. **权限资产**：新增 `deploy/db/provision.sql` 作为迁移前资产：创建或更新两个 `LOGIN` 角色、将 schema 归属迁移身份、设置默认权限，并撤销 `PUBLIC` 与应用身份的 `CREATE`；默认权限为应用身份授予业务表 DML（`SELECT`/`INSERT`/`UPDATE`/`DELETE`）与序列 `USAGE`/`SELECT`；两张 append-only 表与 `schema_migrations` 由启动迁移在关闭迁移池前 `REVOKE` 收窄（两表 `REVOKE UPDATE, DELETE, TRUNCATE`，ledger `REVOKE ALL`），该收窄由迁移身份执行，使 fresh install 无需人工第二步即满足门禁。`deploy/db/grant-application.sql` 保留为 out-of-band 重放与角色轮换资产：对既有表和序列授权并显式 `REVOKE`，用于运维主动补授权或迁移新增表后的重放，不再是每次部署的强制第二阶段。密码明文只经 psql 变量注入，不写入仓库或项目文件；数据库只保存角色认证所需散列。
6. **错误披露**：启动错误只给稳定类别，不含 DSN、密码、角色名以外的部署细节；沿用现有最小披露合同。
7. **残余风险**：迁移凭据仍在同进程环境变量中；进程被攻破后可读取迁移身份并提权。另有一个短窗口：迁移建表后到启动 `REVOKE` 之间，应用角色在两张 append-only 表上短暂持有 `UPDATE`/`DELETE`；该窗口内 listener 尚未启动、无服务流量，但持有应用凭据的外部客户端仍可写入，部署必须控制应用凭据的分发。该方案不声称应用与迁移完整隔离。彻底隔离需要 out-of-band 迁移或独立迁移作业，未纳入本版。
8. **影响**：既有单 DSN 部署将按设计 fail closed，运维必须先执行 `provision.sql`；`grant-application.sql` 仅在 out-of-band 重放或角色轮换时执行。影响 ADR-0021 的启动迁移身份、ADR-0016 的审计追加保证、ADR-0019 的 ACL 词汇边界（文件 ACL 不等于数据库权限）与 ADR-0022 的启动/退出码语义（启动失败仍非零，正常生命周期分类不变）。
9. **明确不证明**：本决定不证明真实部署、CI、备份/恢复演练、RPO/RTO 或生产容量；也不证明两张表之外的全部数据库权限是最小权限。
10. **search_path 与同名关系边界**：门禁在上述判定之外，要求当前生效 search_path（`current_schemas(false)`）内目标表名的关系数恰为 1；存在同名关系歧义、或目标关系不在生效 search_path 上时 fail closed，且错误类别与“表不存在”区分。服务不设置 `search_path`，该路径由运维前置保证；本决定只证明“生效 search_path 下同名关系唯一且可见”，不证明它等于 provision 的目标 schema。

### 后果与残余风险

- 正面结果：应用连接不再具备迁移 DDL 能力；`TRUNCATE` 与 `UPDATE/DELETE` 在权限层被拒绝；门禁用 `INSERT` 正控和三个删改负控证明应用身份的实际 ACL，而不是只证明 trigger 存在。
- 部署代价：首次部署必须先执行 `provision.sql`；`grant-application.sql` 仅在 out-of-band 重放或角色轮换时执行，迁移后新增表/序列须重新授权；单 DSN 旧部署会直接拒绝启动。
- 安全边界：迁移凭据同进程可读的风险保留；应用身份若被攻破，仍可在其合法 INSERT 与业务表权限内作恶，本决定只收窄 append-only 表的删改面。
- 验证边界：门禁查询失败、表缺失、INSERT 缺失、任一删改权限存在都必须 fail closed；真实 PG 角色/权限矩阵已执行并通过（仅单机合成角色/单 schema），仍未独立评审、未取得人的签收、未真实部署/CI、未做备份恢复演练。

### 引用记录

- 2026-10-31 [article-management design](../design-workspace/article-management/design.md) 在 §3.4.9、§5.3、§6.1、§8.1 采用本决定的设计口径：固定双 DSN、启动顺序、两张 append-only 表的 ACL 正负判定、fail closed 错误边界与残余风险；该次设计登记不宣称实现。r02 升版原因：实现与真实 PostgreSQL 证据已产生，状态由“未实现、未验证”更正为“已实现、Go 单测与真实 PostgreSQL 集成已执行并通过”；决定内容未变。实现位于 `cmd/han-server/append_only_acl.go`（`productionMigration`、`migrationDatabaseDSN`、`rejectSharedDatabaseIdentity`、`verifyAppendOnlyApplicationPrivileges`），`productionDependencies().migrate` 已改用它，`cmd/han-server/main.go` 不再直接用应用池运行 runner。`go test ./cmd/han-server -count=1` 全包通过（42.151s）；`cmd/han-server/append_only_acl_integration_test.go`（`//go:build integration`，顶层 `TestAppendOnlyApplicationACL`）以 `HAN_TEST_POSTGRES_DSN` 在 PostgreSQL 18.6 / loopback:5433 执行 PASS：schema `han_s15_acl_a1023308b90a9b3e`、迁移角色 `han_s15_mig_a1023308b90a9b3e`、应用角色 `han_s15_app_a1023308b90a9b3e`；应用对两表 INSERT 成功，UPDATE/DELETE/TRUNCATE 六条语句均为 SQLSTATE 42501，schema CREATE TABLE 为 42501，owner 与 superuser 被门禁拒绝。部署资产为 `deploy/db/provision.sql`、`deploy/db/grant-application.sql`、`deploy/db/README.md`（psql 变量注入，未在真实部署执行）；证据见 `.spec/wip/article-management/evidence-s1.5/acl-r01/`（`run-01.log`、`retained-resources.ndjson`）。该集成只证明单机合成角色/单 schema 的权限矩阵，不证明生产部署、凭据分发或角色轮换；仍未独立评审、未取得人的签收、未真实部署/CI、未做备份恢复演练。
- 2026-10-31 r03 修订（Lead 依 ADR-0028 代理授权执行；触发：独立评审 [`review-code-s1.5-acl-r02.md`](../../../../.spec/wip/article-management/review-code-s1.5-acl-r02.md) 的 `S15-ACL-R02-F003`）：原 §5/§8 把 `provision.sql` 与 `grant-application.sql` 写成每次部署必选的两阶段，与 r02 实际实现分叉——`provision.sql` 的默认权限覆盖业务表 `SELECT`/`INSERT`/`UPDATE`/`DELETE` 与序列 `USAGE`/`SELECT`（r04 更正：r03 原写“仅 `INSERT`/`SELECT`”有误），启动迁移完成后由迁移身份在关闭迁移池前执行 `REVOKE UPDATE, DELETE, TRUNCATE`（两表）与 `REVOKE ALL`（`schema_migrations`），`grant-application.sql` 退为 out-of-band 重放与角色轮换资产。本次同步修订 §5、§8，新增 §10 search_path 与同名关系边界，并更正“同名关系必然 fail closed”的不实表述：原措辞基于 `pg_table_is_visible`，实测同一 search_path 上两个同名关系时该函数只对第一个返回 true（`visible_count=1`、`on_search_path_count=2`），故改为按 `current_schemas(false)` 统计同名关系数并要求恰为 1。未改变 §1–§4、§6、§7、§9 的任何约束。
- 2026-10-31 r04 errata（Lead 依 ADR-0028 代理授权执行；触发：独立复审 [`review-code-s1.5-acl-r03.md`](../../../../.spec/wip/article-management/review-code-s1.5-acl-r03.md)，SHA256 `78D8AA2318ED237B9141EECFCEC5372F355EB997387371BA1C0490730E47A238`（v2 正式修订；v1 `7A236731FEED830D60503FB17A1F5224DE92A737B605DAD17E57BE0FC5C45D31` 已被同路径覆盖，结论同为 PASS WITH MINOR），`S15-ACL-R03-F003`）：r03 在 §5 及其引用记录中把 `provision.sql` 的默认权限写成“仅 `INSERT`/`SELECT`”，该前提与实测不符——`provision.sql:46-49` 实际为 `ALTER DEFAULT PRIVILEGES ... GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES`（覆盖全部新表）与 `GRANT USAGE, SELECT ON SEQUENCES`；append-only 两表与 `schema_migrations` 的收窄由启动迁移在关闭迁移池前 `REVOKE`（两表 `REVOKE UPDATE, DELETE, TRUNCATE`，ledger `REVOKE ALL`）完成。本次按既有 ADR 修订惯例升为 r04（项目 decisions 目录无 errata 不升版先例），将 §5 默认权限句更正为“业务表 DML + 序列 `USAGE`/`SELECT`，启动迁移 `REVOKE` 收窄”，并在 §7 披露建表到 `REVOKE` 之间应用角色短暂持有两表 `UPDATE`/`DELETE` 的窗口（listener 未启动、无服务流量，但应用凭据持有者可写入；部署须控制凭据分发），同时更正上一条 r03 记录中的错误前提。未改变 §1–§4、§6、§9、§10 的任何约束；§5 仅更正事实口径，§7 仅补充同一事实的残余风险披露。本次是文档 errata，不新增实现、测试或部署证据，也不声称人的签收、独立评审或真实部署通过。

---

<a id=adr-0030></a>
## ADR-0030: 首期不提供生产 CAS JournalHeadAnchor provider (2026-11-01)

**目的**: 架构设计 | **仓库**: han | **修订**: adr-0030-r20

**状态**: 已接受（2026-11-01 用户 D02 指令授权 Arch-Designer 论证并落定“首期不提供生产 provider”；确认来源为本轮用户任务指令；不构成人的成果接收、独立评审、生产装配批准或真实 provider 验证） | **确认来源**: 用户 2026-11-01 在 B13J D02 中明确要求由 Arch-Designer 论证并落定生产 CAS Anchor provider 产品选型，且明确“首期不提供”是合法且优先考虑的结论之一。该来源授权的是架构决策与文档写入，不是生产装配、真实独立故障域验证或人的成果签收。 2026-11-02 用户再次给出 standing 授权「对待确认或决策项采用建议值并继续本项目」；r05 据此把四项结论登记为 Lead 采纳的建议值，该授权只是流程推进，不构成人的成果接收、独立评审、生产装配批准或真实验证。

**继承关系**: 只固定 ADR-0019 明确留给 S4.5 的生产 Anchor provider 选择、配置键与启动装配边界；不改变 ADR-0019 已锁定的独立故障域、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒、CAS 协议、key 轮换与退役、Journal/RecoveryRun/article evidence 对账等任何内容。ADR-0019 是本 ADR 的硬边界。

**问题**: ADR-0019 已把生产 CAS provider 的产品选择、配置和启动装配留给 S4.5，但当前没有可核的产品、凭据、独立故障域、部署/ACL 与真实验证证据。改动前 `identity-auth/design.md:3892` 记实际产品/凭据/独立故障域 unknown（本 ADR 同轮把 `design.md:3892` 改写为“当前无已登记产品/凭据/独立故障域”，不把改写后的原文当作改前状态证据）；生产 root 以空 registry 构造，anchor 相关端口未注册即 closed。需要在不虚构未验证依赖的前提下，决定首期是否提供生产 provider，并固定无论是否提供都必须遵守的配置键与装配合同。

**决策**: 首期不提供生产 CAS JournalHeadAnchor provider。S4.5 维持未交付；生产 recovery `enter`/`install` 继续返回 closed，不得据确定性测试替身、空 registry、环境变量存在或任何隐式来源宣称生产可用。仅保留 ADR-0019 已定的 provider-neutral `JournalHeadAnchor` port、调用边界与确定性测试替身。`HAN_RECOVERY_ANCHOR_ADAPTER` 与 `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 是唯一配置入口；生产必须显式同时提供已登记 adapter ID 与受保护绝对配置路径，二者无代码默认值、互不推断，root 不得按环境、hostname、数据库或其他配置推导 adapter。未配置即不启动 anchor 相关能力并保持 closed。 r05 补充：在保持上述「首期不提供」与生产 closed 不变的前提下，本 ADR 以 Lead 采纳的建议值给出 (a) 产品与部署形态、(b) 凭据来源与最小权限/ACL、(c) 独立故障域可核判据、(d) 部署拓扑与运维责任四项决策级结论（见「r05 四项结论」）。该四项不登记实现、不交付 provider、不装配 root，也不声称任何产品、凭据、故障域或 ACL 已验证。

### 现状 / 痛点 / 改造 / 效果

现状是“port 已定、provider 未定”：ADR-0019 已固定独立故障域 CAS 头锚点的协议、pending/committed 转换、失败分类与测试替身边界，S4.2 只交付 provider-neutral 接缝；生产 provider 的产品、凭据、部署、ACL 与真实独立故障域验证均未交付。当前既没有可核的外部产品实例，也没有证明其与 PostgreSQL 故障域分离、可执行 CAS 或满足保留/回退语义的证据。

痛点在于：若现在直接挑一个产品，就会把未验证的依赖、凭据和责任写进公共合同；若继续让 root 保持“可注入未定义 adapter”的模糊状态，实施者容易以 file、SQL 或 fake 自动 fallback 填补空缺，而这三类路径已被 ADR-0019 明确禁止。首期需要一个可审计、fail-closed、不会伪造生产就绪的明确结论。

| 方案 | 实现机制 | 优势 | 劣势与未满足条件 | 结论 |
|---|---|---|---|---|
| A. 独立小数据库（专用实例，或独立 schema + 条件写） | 专用数据库实例承载 `journal_id/namespace/epoch/head_sequence/head_hash/revision`，以条件更新或事务实现 CAS；或仅在独立 schema 内建表 | 关系型 CAS 语义成熟；专用实例可与业务 PostgreSQL 分域；运维工具链常见 | 独立 schema 若与业务库同实例，仍随同一备份/故障域回退，不满足 ADR-0019 的独立故障域；专用实例还需产品/版本、凭据、网络、备份恢复、ACL、容量与运维责任，当前均无可核证据，也未做真实故障域验证 | 不选（该行「不选」是 2026-11-01「是否现在选定并交付」的时点判断，不是对该形态的永久排除；r05 起的后续目标形态见 (a)，二者不是两条并行的实现路径） |
| B. 外部配置/秘密系统（Vault / KMS 类）承载 CAS 头 | 用 Vault KV 类带版本/CAS 的秘密存储保存头状态；KMS 类服务仅可能承担密钥或封装，不是通用 CAS 头存储 | 隔离、ACL、审计与轮换能力强；可复用组织级秘密基础设施 | 需要选定具体产品、认证方式、网络与高可用/解封策略；KV 的版本 CAS 与 pending/committed、namespace、epoch、revision 语义需额外映射；KMS 类服务通常只提供密钥管理/加解密与有限的条件性密钥操作，不提供通用 CAS 头存储；当前无产品、凭据、部署与验证证据 | 不选 |
| C. 对象存储条件写（S3 conditional PUT / ETag CAS） | 以对象条件写或 `If-Match`/ETag 实现 expected revision 的 compare-and-swap，保存 committed/pending 头 | 对象存储可与业务 PostgreSQL 分组部署，但独立故障域须逐案证明；条件写可检测并发冲突；容量与持久性由平台承担 | 依赖具体云产品、区域、凭据与 IAM；需证明条件写、读后一致性、版本/回滚、对象锁与 pending/committed 两阶段语义；网络延迟、错误分类与运维责任未核；当前无真实环境验证 | 不选 |
| D. 首期不提供生产 anchor（选定） | 不注册任何生产 provider；仅保留 port 与确定性测试替身；生产 `enter`/`install` 保持 closed；配置键必须显式且无默认 | 不引入未验证依赖；不虚构产品、凭据或故障域；与 ADR-0019 的 fail-closed 和“测试替身不得作为生产证据”完全一致；成本与审计面最小 | 首期生产恢复链不可用；S4.5 维持未交付；后续必须重新选型、验证并修订 ADR 才能解除 closed | 选定 |

选择 D。A、B、C 都需要当前不存在的真实产品、凭据、部署/ACL 和独立故障域验证；在缺少这些证据时选择任一产品，等于把“未验证”写成“已决定可部署”，并把后续实施锁进一个未经批准的公共依赖。D 不新增任何外部依赖，不触发 file/SQL/fake fallback，且明确保留后续通过新决策解除 closed 的路径，是当前唯一可审计且不伪造生产就绪的结论。

改造效果是：生产 root 不再需要也没有合法的“默认 adapter”可构造；配置缺失、adapter 未登记、配置路径不可读或验证不通过时一律 closed；确定性测试替身继续只服务协议测试，不被注册为生产 provider。该结论同时把“产品选型”从模糊的实现细节提升为需要独立证据和正式决策的门禁，避免实现阶段自行补默认值。

#### r05 四项结论（决策级要求；「建议值」仅描述 2026-11-02 的落定方式，不构成交付、装配或验证）

本轮按用户 2026-11-02 的 standing 授权，把下表四项作为 Lead 采纳的建议值落定。四项只补齐「本 ADR 尚无产品/凭据/独立故障域/部署与 ACL 结论」这一文档缺口：「首期不提供」不变，不登记实施任务、不交付 provider、不装配 root，生产恢复 `enter`/`install` 继续 closed，约束 7 全项仍未满足，本 ADR 不声称任何产品、凭据、独立故障域、ACL 或演练已验证。r04 表中 A/B/C/D 的「不选/选定」记录的是 2026-11-01 时点「是否现在选定并交付」的判断，不是对该形态的永久排除，历史结论不改写；本段只把候选 A 的专用实例子形态定为后续实施的目标形态（见 (a)），B、C 仍不选，首期仍不交付。自 r06/r07 起，(b)–(d) 的内容一律是约束性要求；标题与表头中的「建议值」只描述 2026-11-02 的落定方式，不是强度降级，也不得被读成可选建议。

| 项 | r05 结论（约束性要求；「建议值」仅指落定方式） | 排除的替代路径 | 失败或降级条件 |
|---|---|---|---|
| (a) 产品与部署形态 | 候选 A 的「专用实例」子形态：独立 PostgreSQL 实例（独立集群与数据目录、独立主机或独立实例加独立存储卷、独立网络入口）承载单一 append-only 头事件表——形态仍是单一专用实例、单张事件表加派生 head，不引入可变 head 行、不引入第二张权威表。事件表每行至少携带 `journal_id`、`namespace`、`revision`、`epoch`、`head_sequence`、`head_hash`、`kind`（pending/committed/abandoned）、`pending_token` 与操作标识，字段语义与 ADR-0019 的 anchor 必填状态一一对应；`epoch` 缺失或与 Journal 当前 epoch 不匹配即 closed（ADR-0019 的 epoch 不匹配停服要求）。CAS 语义为：在 `UNIQUE(namespace, revision, kind)` 上 `INSERT` 同 revision 的 pending 行（携带不可预测且只在同 namespace + revision 内有效的 `pending_token`），再 `INSERT` 携带同一 token 的 committed 行；committed/abandoned 行必须引用同 revision 的 pending 行（复合外键 + `kind` 校验触发器，见 (b)）。终结行互斥：同一 revision 至多一个终结行（committed 或 abandoned），由 `(namespace, revision) WHERE kind IN ('committed','abandoned')` 的部分唯一索引在数据库层保证；已 committed 的 revision 不得再追加 abandoned，已 abandoned 的 revision 不得再追加 committed，违反即约束层失败，不依赖应用判断。head 只由 committed 终结行派生（取 committed 终结行的最大 revision；abandoned 不计入 head 但计入结算水位）；`MAX(revision)`（含 pending、committed 与 abandoned）只用于判定该 revision 是否已被占用，不用于推导 head 或预留值；预留值恒为见证结算水位 + 1。唯一键冲突、token 不匹配、终结行冲突即 CAS 冲突。anchor 侧 expected revision 条件由见证预留承担：只有持有见证当前预留 token 的写入者才能插入 pending，等价于 ADR-0019 的 expected revision CAS。具体大版本须落在部署方具名 owner 维护的受支持版本清单内、符合 ADR-0021 的受支持 patch 口径，并随清单附 EOM/升级门禁（不在清单内或已 EOM 即 closed）；清单 owner 与升级门禁的责任主体随 (b)(d) 的批准记录固定，本 ADR 不写死版本号。r04 表中候选 A 的「以条件更新或事务实现 CAS」仅为历史描述、不作实现依据，实现一律以 r07–r09 的单一写入模型为准 | 业务库内独立 schema、同实例另一库、共用备份链；B（Vault/KMS）与 C（对象存储条件写）维持 r04 不选；file、业务库表、内存与 fake 恒禁；以可变 head 行或第二张权威表替代事件表；同一 revision 出现多个终结行 | 只能提供同实例或共享备份链时，本项降级为「首期不提供」，生产继续 closed 并回到用户做产品与环境决策 |
| (b) 凭据来源与最小权限/ACL | 专用数据库角色，仅该实例该库头事件表的 `SELECT`/`INSERT`，显式无 `UPDATE`、`DELETE`（历史事件行不可改写、不可删除），无 DDL、`TRUNCATE`、跨库与超级用户；`UNIQUE(namespace, revision, kind)`、pending 行唯一键 `UNIQUE(namespace, revision, pending_token)`（终结行复合外键的引用目标）、终结行部分唯一索引 `(namespace, revision) WHERE kind IN ('committed','abandoned')`、`kind` 校验触发器与见证侧单调约束，均由部署方一次性受控脚本创建，运行时不持 DDL；PG 的部分唯一索引不能直接作外键目标，故终结行对 pending 行的引用必须由非部分唯一键 + 触发器（或把 pending 行与终结行拆为同库两张表：pending 表以 `(namespace, revision)` 唯一并携带 token，终结表以 `(namespace, revision, token)` 外键引用 pending 表并在 `(namespace, revision)` 上部分唯一）实现，两种形态都不得引入第二张权威表；但当单表单 token 列的形态无法以非部分唯一键满足引用唯一性时，以 r14 补充段「errata」至「并发负向用例实测」条的 errata 形态为准（部分唯一索引 + 触发器），(b) 的「非部分唯一键」在该形态下不适用，本条与 errata 互为交叉引用且不改变终结行互斥要求；token 只在同一 namespace + revision 内有效，跨 namespace 复用即失败；口令只经一次性 fd/stdin 注入，禁止经 environ 继承，禁止写入命令行参数、配置文件明文、日志、Trace、core/panic dump 或子进程环境；该口径与 ADR-0029 已登记的「迁移凭据仍在同进程环境变量」残余风险同类，不放宽、不新增更松的第二套标准；`HAN_RECOVERY_ANCHOR_CONFIG_PATH` 只承载非秘密字段，必须绝对路径、owner-only、no-follow，且其写入者/所有者必须与 anchor 实例管理员分属不同主体，或配置变更须双人批准并留审计（二者至少满足其一，否则 closed）；见证端点只作为同一配置文件内的 witness 段（地址、来源地址范围、TLS CA 与主机名），不新增环境变量或第三配置键，配置文件出现任何秘密字段即 closed；见证端点必须钉进批准记录（地址、来源范围、CA/主机名摘要），启动时与配置值比对，不一致即 closed；见证口令经第二次独立的一次性注入通道读取：使用两个不同且固定的 fd（3 与 4）或两条独立 pipe，禁止同一 stdin 顺序读取，注入者与保管主体必须与 anchor 口令不同，两个秘密不得共用同一 fd、同一账号或同一秘密库路径；批准记录须记录两个注入者的身份与注入时间；连接限定来源地址；TLS 必须 verify-full（校验 CA 链与服务端主机名）、协议不低于 TLS 1.2、禁止降级协商，显式禁止 `sslmode=require`/`prefer`/`allow`/`disable`，CA 与主机名不匹配即 closed 而非降级，且不得复用业务库 CA 与 DSN；见证写入端的 TLS、超时与失败分类与 r07 失败分类一致 | 复用业务库角色、以超级用户运行、授予头事件表 `UPDATE`/`DELETE`、把口令写入配置文件或命令行、运行期持有 DDL、见证口令与 anchor 口令共用同一 fd/账号/秘密路径、新增第三配置键、配置写入者与 anchor 管理员为同一主体且无双人批准、见证端点未钉进批准记录 | 角色与 GRANT 矩阵未取证、角色仍持有头事件表 `UPDATE`/`DELETE`、配置路径可被非 owner 读取、口令出现在 argv/日志/environ、配置文件含秘密字段、见证口令与 anchor 口令未分属不同保管主体、见证端点与批准记录不一致、或 TLS 未达 verify-full/协议低于 1.2 时一律 closed；本 ADR 不登记任何角色名、主机名或秘密值 |
| (c) 独立故障域 | 可核判据六项：独立主机/集群与数据目录；独立存储卷与备份链；业务库 PITR 与整库恢复不回退 anchor，且 anchor 头状态本身不可回退——必须同时具备且缺一即 closed：(1) append-only 头事件表（历史事件行无 `UPDATE`/`DELETE`，同一 revision 至多一个终结行）与 (2) 位于 anchor 实例故障域之外、且同时位于业务库故障域与业务库备份链之外的单调见证（抗整实例回退；见证落在业务库或业务库备份链内即否证）；见证只持久两个量加一个可选预留：结算水位 `w`（已终结 revision 的最大值，终结＝committed 或 abandoned）与至多一个在飞预留 `r = w + 1`（携带 `token`），不持久进程内 head，也不持久任何可被 anchor 库回退带走的状态；adapter 启动时与每次推进前必须核对：见证不可读或不可写 → closed；事件表存在 revision > `w` 的终结行 → closed（见证落后或被抹，不得用观测 head 抬高见证）；事件表 revision ≤ `w` 存在未终结行且见证无预留 → closed；有预留 `r` 且事件表 `r` 上已有终结行 → 允许幂等完成（把 `w` 推进到 `r` 并清预留，不重写终结行）；有预留 `r` 且 `r` 上无终结行 → 按 ADR-0019 `:468` 三分支处理；其余组合一律 closed；`w` 只允许推进到预留值 `r`，不得跳号、不得回退，连续 abandoned 合法（每次终结都把 `w` 推进 1）；epoch 缺失或与 Journal 当前 epoch 不匹配 → closed；仅比较 expected revision 不足以发现 anchor 被回退或被直改回旧值；独立网络入口与独立凭据，凭据须由与业务库不同的保管主体持有，且见证的写入凭据与 anchor 连接凭据分属不同保管主体，业务库管理员不可读 anchor 凭据、不可登录 anchor 实例；独立升级/维护窗口；演练证据（业务库回退到旧快照后 anchor 仍拒绝已消费 run 与旧头；anchor 实例整体回退到旧备份并重启后，服务在任何 `enter`/`install` 之前即 closed 且不产生任何 anchor 写入；新副本冷启动读到 anchor 观测水位低于见证时 anchor 能力即 closed）。三项见证演练与约束 7 验证项的映射为：整实例回退并重启 → missing/rollback；新副本冷启动 → 崩溃恢复；业务库回退 → stale/missing | 同实例不同 schema、同备份链、同超级用户、同维护窗口、同一保管主体或同一 DBA 可同时操作两侧；以可变 head 行替换事件表；同一 revision 出现多个终结行；见证与 anchor 同域、落在业务库或业务库备份链内、可被 anchor 角色改写、支持递减或提供重置接口、或可被 anchor 管理员经任一备份/快照/恢复路径降低；见证实现只支持单调计数器而无法保存并原子比较 `(w, r, token)` | 本 ADR 只给判据与否证条件，不提供任何演练结果；仅 append-only 而无域外见证即 closed；见证可被 anchor 管理员经任一备份/恢复路径降低即 closed；业务库回退、anchor 整实例回退并重启、冷启动三项演练均为约束 7 必测项，真实验证属后续实施/验证卡 |
| (d) 部署拓扑与运维责任 | 部署方拥有实例、备份/恢复、凭据轮换、ACL 复核、域外见证的运维与演练并保留证据；见证端点、见证口令的第二次一次性注入通道与保管主体、见证侧的单调与保留约束同属部署方责任，并逐项写入批准记录（最小字段见 r08 补充段）；本服务只经 adapter 读写，不迁移、不建表、不做备份，本服务不得对 anchor 实例调用或执行任何备份/恢复/迁移/provision 脚本，也不得暴露可触发它们的入口（本限定只约束 anchor 实例，不改变 ADR-0021 的启动自动迁移与 ADR-0031 的 root 启动已批准 restore 子进程）；anchor 表结构与见证侧约束不得进入 `migrations/`，`HAN_MIGRATION_DATABASE_DSN` 只指向业务库、不得指向 anchor 实例或见证端点，违反即 closed；业务 PostgreSQL 与 anchor PostgreSQL 必须按 (c) 判据证明处于不同故障域（未证明并登记前不得宣称独立），两库之间必须无事务、无复制、无共享凭据 | 由本服务托管 anchor 实例、由应用迁移建表、与服务共用 DSN、把 anchor 纳入业务库备份链、由同一账号或同一秘密路径承载两侧凭据 | 运维责任（含域外见证的运维责任与批准记录字段）未落到具名部署角色时，装配即 closed |

与约束 2 的关系：约束 2 所禁的「SQL」，指以业务数据库或通用 SQL 表作为 anchor 的自动 fallback 或隐式来源，包括业务 PostgreSQL 表、业务库内 SQL 表、内存、fake，以及任何「环境探测后自动选择」的路径；本项是显式登记、显式配置、固定指向独立实例的 adapter，不探测环境、不回退业务库、不构造隐式来源，因此不属于被禁路径。约束 2 正文本轮不改动，本段只是对该用词的固定解释，效力低于约束正文，如有冲突以约束 2 为准。

对下游门禁的影响：本四项只让 ADR-0030 约束 9(b) 的入口有了可评审对象，并未满足该入口。门禁分两级，二者不可互相替代：**评审门禁**只有一条——本决策修订被架构与安全独立评审接受，且不以候选 S4.23 自身的任何产出为前提；**启动前置**为下列 (i)–(v)，须由用户/组织作出并登记，未满足即不得派发/认领/开始改动（含不得引入依赖、不得写骨架或接线代码）。当前仍缺、须由用户/组织作出产品与环境决策的项逐条为（本 ADR 不代为选定，在此之前 S4.23 一类卡不得启动）：(i) 具体产品——厂商与托管或自建形态；(ii) 受支持大版本及对应 EOM/升级口径；(iii) 许可与具名 owner——实例、备份、凭据保管主体与版本清单各自的责任主体；(iv) 目标环境——主机/集群、存储卷与备份链、网络入口与来源地址范围；(v) 见证形态、owner 与保证等级取证——单调不可递减且无重置接口、跨 anchor 故障域持久、anchor 管理员不能经任一备份/快照/恢复路径降低见证（或保留期 ≥ anchor 备份保留期）、每次推进留可审记录，且见证形态必须支持 `(w, r, token)` 三元组原子条件写（见 r09 补充段）；「受支持清单」的 owner 与版本门禁须与 (a) 的版本清单同一责任主体。(a)–(d) 的真实部署证据（实例、ACL/GRANT 矩阵、TLS、备份链与回退演练）是候选 S4.23 一类实施/验证卡的退出条件，也是候选 S4.24b 与 S4.22 的启动前置，不是 S4.23 的启动前置。术语口径（Lead 2026-11-02 已裁决，见 r07 补充段；解释性口径、效力低于约束正文）：登记＝写入卡面与 Plan，启动＝派发/认领并开始改动；候选 S4.23/S4.24a/S4.24b/S4.22 当前为「已登记、未启动」；约束 9(b) 的入口约束登记，启动另有本段 (i)–(v) 前置，二者不可互相替代（约束 7 明文不约束三类卡的登记）；约束 9 正文本轮不改动，如有冲突以约束正文为准。本段的两级门禁口径同为解释性口径，效力低于约束正文。不得以本段措辞冒充已论证、已满足或已启动。

**启动前置 (i)–(v) 的已登记建议值（Lead 建议值，非验证）**：(i) 具体产品——PostgreSQL（社区官方发行包），建议值取 16 大版本线，自建独立实例（独立集群与数据目录、独立网络入口）；主机/实例形态一律按 `:953(c)` 判据 1 取证，「独立实例加独立存储卷」只有同时满足判据 1（不与业务 PostgreSQL 共用主机、集群与数据目录）时才成立，且任何形态都不得单独作为独立故障域成立的证据；精确补丁版本按 (ii) 由部署方具名 owner 维护的受支持清单确定；(ii) 受支持大版本——不写死版本号，由部署方具名 owner 维护受支持清单并附 EOM/升级门禁；(ii) 的受支持清单在本 ADR 内优先于 (i) 的版本指向，两者不一致时以清单为准（(i) 的 16 大版本线为建议起点，随清单选择失效并登记）；批准记录须登记该包的来源校验（包签名或固定仓库）与 EOL 门禁，并禁止清单外来源或被重打包的发行物；(iii) 许可与具名 owner——**当前 unknown**，须在批准记录中具名；(iv) 目标环境——单机/单集群、独立存储卷与独立备份链、独立网络入口，来源地址范围待登记；(v) 见证——域外单调计数器形态，须支持 `(w, r, token)` 三元组原子条件写、无重置接口、不可被 anchor 管理员经备份/恢复路径降低，owner **unknown**。以上为建议值登记，**不构成验证**；(iii)/(iv)/(v) 未具名或未取证前，启动前置 (i)–(v) 仍**未满足**，S4.23 一类卡**不得启动**。本条不改变约束 9(b) 的评审门禁口径，也不构成评审 PASS 或人的批准。

#### r06 补充（两路独立评审处置；决策级要求与否证条件，不是证据）

本小节处置架构侧 `B13J-D05-R01-F001`–`F006` 与安全侧 `B13J-D05-SEC-R01-F001`–`F008` 中经 Lead 采纳的部分。以下全部是要求与否证条件，不是已取得的证据；任何一条未在实施/验证卡中取证，对应结论不成立。

**目标形态 → 六项判据的推理**（每条都以「在下列前提成立时」限定，不构成已验证事实）：
- 独立主机/集群与数据目录：在部署方为 anchor 分配独立主机或独立集群、不与业务 PostgreSQL 共用主机/集群/数据目录的前提下满足；共用即否证。
- 独立存储卷与备份链：在 anchor 数据目录落在独立存储卷、其备份目标与保留策略独立于业务库备份链的前提下满足；共用卷或共用备份链即否证。
- 业务库回退不回退 anchor，且 anchor 自身不可回退：在头状态为 append-only 事件表（历史事件行无 `UPDATE`/`DELETE`、同一 revision 至多一个终结行）、且存在位于 anchor 实例故障域之外并同时位于业务库故障域与业务库备份链之外的单调见证、adapter 启动与每次推进前按 r09 补充段的唯一状态机核对（见证只持久 `(w, r, token)`：`w` 为已终结 revision 的最大值，`r` 为至多一个在飞预留且恒为 `w + 1`；事件表存在 revision > `w` 的终结行、revision ≤ `w` 存在未终结行且无预留、`r` 越界或 token 不匹配时拒绝）、业务库 PITR/整库恢复不触及 anchor 存储卷的前提下满足；仅比较 expected revision、把已观测最大 revision 放在进程内存或 anchor 库内、允许回写/删除历史事件行、允许同一 revision 出现多个终结行、或让见证可被 anchor 管理员经任一备份/恢复路径降低即否证。
- 独立网络入口与独立凭据（含保管主体分离）：在 anchor 具备独立网络入口与来源地址白名单、专用数据库角色、且凭据由与业务库不同的保管主体持有的前提下满足；业务库管理员可读 anchor 凭据或可登录 anchor 实例即否证。
- 独立升级/维护窗口：在 anchor 实例的升级、补丁与维护窗口独立于业务库、两者不共享同一变更批准的前提下满足；共用窗口或共用变更批准即否证。
- 演练可核：在具备三项演练设计的前提下满足——（i）业务库回退到旧快照后 anchor 拒绝已消费 run 与旧头；（ii）anchor 实例整体回退到旧备份并重启后，服务在任何 `enter`/`install` 之前即 closed 且不产生任何 anchor 写入（判据只能来自域外见证，不得依赖进程内存或 anchor 库内状态），回退备份按四档取证：(A) 与当前见证相等的备份（对照，判为无回退）、(B) 恰为见证 −1 的备份（必须 closed）、(C) 更旧备份（必须 closed）、(D) 新副本冷启动读到 anchor 观测水位低于见证（必须 closed）；（iii）重启后首次核对不得使用进程内缓存，必须重新读取见证；另须在「预留、pending、终结行、见证推进」四个写入点各注入一次崩溃，重启后必须落到 r09 补充段「见证状态机」条的确定恢复分支之一，不得永久 closed、不得手工修库。三项演练及其原始证据保留均属约束 7 必测项（映射见 (c)），本 ADR 不提供任何结果。

**残余共享依赖与否证条件**（任一项落入同域则本项不成立，须在实施/验证卡内逐项取证）：
- 云账号/组织级 IAM：anchor 与业务库处于同一云账号或同一组织级 IAM 管理域、同一主体可同时改两侧权限即否证。
- 区域与网络控制面：同一区域控制面故障、同一 VPC/路由/防火墙控制点可同时切断或篡改两侧流量即否证。
- 备份系统：同一备份服务、同一凭据库或同一恢复操作台可同时回退两侧即否证。
- 运维通道：同一跳板机、同一 CI/CD 凭据或同一自动化账号可同时对两侧执行变更即否证。
- 同一 DBA 或同一变更窗口：单人能在同一变更窗口内完成「回退业务库 + 改回 anchor 头」即否证。

**bundle 的 `Exclusive` 端口三段式（已由 ADR-0032 r04 起对齐、r05 收口，见下 r07/r09 段落）**：候选 S4.24b 交付前，bundle 的 `Exclusive` 保持 closed 端口（`recoveryClosedPort`），候选 S4.23 只负责把它作为已登记但 closed 的端口接入，不得填任何实现；候选 S4.24b 交付后，唯一来源是 ADR-0032 的本地 POSIX 独占文件锁后端；任何阶段都不得由 anchor 数据库的 advisory lock、锁表或任何数据库侧机制填充，也不得为「凑齐四端口」而临时顶替。形式满足约束 10 而违反 ADR-0032 的路径一律 closed。ADR-0032 已于 2026-11-02 修订为 `adr-0032-r04`、同日续修至 `adr-0032-r05`（r04 把跨 ADR 对齐落到第 8 条的候选 S4.24b 节点，r05 再把「解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据」并入同一节点以消除顺序自指；当前修订串见 ADR-0032 修订行（`adr-0032-r05`）），其解除顺序第 8 条已把「`Exclusive` 离开 `recoveryClosedPort`」移到候选 S4.24b 节点（ADR-0032 解除顺序第 8 条候选 S4.24b 节点，符号锚），与本 ADR 的三段式一致，原跨 ADR 冲突已消除；本 ADR 保留「候选 S4.24b 交付前 `Exclusive` 保持 closed」作为更严的防御性约束，凡据此在 S4.24b 交付前提前放开 closed 的实现一律 closed。

#### r07 补充（单一写入模型、域外见证、Exclusive 顺序与术语裁决；全部是要求与否证条件，不是证据）

**单一写入模型（r06 及更早文本的析取读法自本修订起作废）**：本 ADR 只承认一种 anchor 头写入模型——单一 append-only 头事件表，`UNIQUE(namespace, revision, kind)` 保证事件行唯一，pending 行另有 `UNIQUE(namespace, revision, pending_token)`，终结行由 `(namespace, revision) WHERE kind IN ('committed','abandoned')` 的部分唯一索引保证同一 revision 至多一个终结行；head 由 committed 终结行派生（取最大 revision），pending/committed/abandoned 各为事件类型，应用角色只有 `SELECT`/`INSERT`。同一 revision 的 pending 行携带不可预测的 `pending_token`，committed 行与 abandoned 行必须携带同一 token 并以外键/触发器引用同 revision 的 pending 行，因此不存在「跳过 pending 直接写 committed」「同一 revision 换 token 重写」或「已 committed 的 revision 再追加 abandoned（反之亦然）」的合法路径；ADR-0019 的旧 committed → pending → committed 两段 CAS 相应映射为「pending 行落库 → committed 行携带同一 token 落库」，「清除未发布 pending」映射为追加 abandoned 事件（head 派生忽略 abandoned，该 revision 不得再被不同 token 或不同 payload 复用，下一次 CAS 使用更大的 revision）。r06 及更早文本中「append-only 头事件序列或等价外部见证」的「或」不再有效，(a)(b)(c) 与本段一律按合取理解：append-only 与终结行互斥抗直改与抹回，域外单调见证抗整实例回退，缺一即 closed。
- 形态承诺不变：仍是单一专用实例、单张事件表、派生 head；不引入可变 head 行、不引入第二张权威表，也不改变 (a) 的实例、网络与凭据形态承诺。
- 角色矩阵随模型固定：头事件表只有 `SELECT`/`INSERT`，无 `UPDATE`/`DELETE`、无 DDL；`UNIQUE(namespace, revision, kind)`、pending 行 `UNIQUE(namespace, revision, pending_token)`、终结行部分唯一索引 `(namespace, revision) WHERE kind IN ('committed','abandoned')`、`kind` 校验触发器与见证侧单调约束均由部署方一次性受控脚本建立，运行时不持 DDL；历史事件行不可改写、不可删除，因此「把最大 revision 行改回旧值」与「对已提交 revision 追加 abandoned 抹回 head」都不存在合法路径，后者的拦截点是数据库约束而非应用判断。

**域外单调见证（合取项，不可省略）**：
- 位置与形态：必须位于 anchor 实例故障域之外（与 anchor 不同主机或集群、不同存储卷、不同备份链），同时必须位于业务库故障域与业务库备份链之外；具体形态（例如域外单调计数器、WORM 或审计存储等）由部署方在受支持清单内固定并写入 (b)(d) 的批准记录，本 ADR 不选定产品、不写死形态。保证等级五项缺一即 closed：(1) 单调不可递减、无重置接口；(2) 跨 anchor 故障域持久，确认写不随 anchor 实例或其备份恢复丢失；(3) anchor 管理员不能经任一备份/快照/恢复路径降低见证，或见证保留期 ≥ anchor 备份保留期；(4) 每次推进留可审记录（时间、namespace、revision、token 摘要、操作者身份）；(5) 见证实现必须支持 `(w, r, token)` 三元组的原子条件写（CAS），或能以追加日志唯一派生三元组并在读取时验证；仅单调计数器或不可更新的 WORM 不足以单独承担该三元组。
- 写入权限与批准记录：anchor 专用角色与同一保管主体不得写见证；见证的写入凭据与 anchor 连接凭据必须分属不同保管主体，复用即否证；见证口令经第二次独立的一次性注入通道读取（固定 fd 3 与 4 或两条独立 pipe，禁止同一 stdin 顺序读取），注入者与保管主体必须与 anchor 口令不同。批准记录最小字段见 r09 补充段（两个具名保管主体、两个秘密存储位置、两个注入者与注入时间、轮换窗口是否可合并、见证端点与保证等级取证、见证保留期与 anchor 备份保留期的比较）；否证条件为「同一秘密存储、同一账号可读两侧、或同一变更窗口可同时轮换两侧口令」——命中任一即 closed。
- 持久语义：见证只持久 `(w, r, token)` 三元组：`w` 是已终结 revision 的最大值（终结＝committed 或 abandoned，一次终结只推进 1），单调不减；`r` 是至多一个在飞预留，无预留时为空、有预留时恒等于 `w + 1`；`token` 只在有预留时非空且不可预测。对外可见 head 由事件表的 committed 终结行派生（取最大 revision），不写入见证，也不作为提升见证的依据；连续 abandoned 时 head 可低于 `w` 多个 revision，这是合法状态，不得据此判 closed，也不得用观测 head 抬高 `w`。三元组只持久于域外见证，不得放在进程内存（重启或新副本归零），也不得放在 anchor 库内（随实例回退一起回退）；见证形态必须支持三元组原子条件写（见 r07 补充段「位置与形态」条第 (5) 项）。
- 写入次序与崩溃窗口：只允许「先预留见证、后提交事件」。预留前必须满足：见证为 `(w, 无预留)`、事件表 revision ≤ `w` 均已终结、revision `w + 1` 无任何事件行；以条件写把见证从 `(w, 空)` 推进到 `(w, r = w + 1, T)` 成功后才允许插入 pending 行；随后插入 committed 或 abandoned 终结行，最后把见证推进到 `(w' = r, 无预留)`；`w'` 只允许等于 `r`，不得跳号、不得回退。adapter 必须在向调用方返回成功之前确认见证已推进到不小于该 revision 且终结行已按同一 token 落库。崩溃恢复按见证 × 事件表状态确定：(i) 有预留、无 pending 行 → 用同一 token 补插 pending 行后追加 abandoned，并推进见证（append-only，角色无 `UPDATE`）；(ii) 有预留、有 pending、文件仍为旧头且可验证临时文件仍存在 → 用同一 token 提交 committed（完成发布）并推进见证；(iii) 有预留、有 pending、文件仍为旧头且无临时文件 → 追加 abandoned 并推进见证；(iv) 有预留、文件已为新头、无 committed → 用同一 token 提交 committed 并推进见证；(v) 有预留、`r` 上已有同 token committed 或 abandoned 终结行 → 幂等完成，只把见证推进到 `(r, 无预留)`，不重写任何事件行、不重复插入终结行；(vi) 文件既非旧头也非新头、token 不匹配、`r` 上存在多个终结行、或见证不可读写 → closed。禁止把 (v) 判为 closed，也禁止手工 `UPDATE` 修库。废弃预留必须以追加 abandoned 事件收尾（引用同一 pending 行与同一 token），该 revision 不得再被不同 token 或不同 payload 复用；被烧毁的 revision 计入 `w`，下一次预留仍为 `w + 1`，因此连续 abandoned 不会阻塞下一次预留，也不会造成「差值 2 即 closed」。
- 失败分类：见证不可读或不可写、事件表存在 revision > `w` 的终结行（见证落后或被抹）、事件表 revision ≤ `w` 存在未终结行且见证无预留、预留 `r ≠ w + 1`、`token` 与同 revision pending 行或预留记录不符、reserved revision 上存在多个终结行、`epoch` 缺失或不匹配、或试图用观测 head 抬高见证，一律 closed，不得降级为「只比较 expected revision」、不得跳过核对继续推进、不得自愈抬高见证；与约束 4 的 fail-closed 同构。例外只有 r09 补充段列出的幂等完成态（预留与终结行都健在、只缺见证推进）。

**回退演练升级**：必须以「anchor 实例整体回退到旧备份并重启」与「新副本冷启动」为工况，不得以「改一行」或进程内状态替代；回退备份分四档取证——(A) 与当前见证相等的备份（对照，判为无回退）、(B) 恰为见证 −1 的备份（必须 closed）、(C) 更旧备份（必须 closed）、(D) 新副本冷启动读到 anchor 观测水位低于见证（必须 closed）；判据是「服务任何 `enter`/`install` 之前即 closed 且无任何 anchor 写入」，重启后首次核对不得使用进程内缓存。另须在「预留、pending、终结行、见证推进」四个点各注入一次崩溃，重启后必须落到 r07 补充段「写入次序与崩溃窗口」条的确定恢复分支之一，不得永久 closed、不得手工修库。判据只来自域外见证。三项演练及其原始证据保留属约束 7 必测项（映射见 (c)）。

**bundle `Exclusive` 与 ADR-0032 的顺序**：三段式见上「bundle 的 `Exclusive` 端口三段式」——候选 S4.24b 交付前 `Exclusive` 保持 `recoveryClosedPort`（候选 S4.23 只把它作为已登记但 closed 的端口接入），交付后唯一来源是 ADR-0032 的本地 POSIX 独占文件锁后端；任何阶段都不得由 anchor 数据库的 advisory lock、锁表或任何数据库侧机制顶替。ADR-0032 已于 2026-11-02 修订为 `adr-0032-r04`、同日续修至 `adr-0032-r05`（r04 把跨 ADR 对齐落到第 8 条的候选 S4.24b 节点，r05 再把「解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据」并入同一节点以消除顺序自指；当前修订串见 ADR-0032 修订行（`adr-0032-r05`）），其解除顺序第 8 条已把「`Exclusive` 离开 `recoveryClosedPort`」移到候选 S4.24b 节点，并要求候选 S4.24b requires 候选 S4.23（ADR-0032 解除顺序第 8 条候选 S4.24b 节点，符号锚），与本 ADR 的三段式一致，原跨 ADR 冲突已消除。本 ADR 保留更严的防御性约束：候选 S4.24b 交付前 `Exclusive` 必须保持 `recoveryClosedPort`，凡据此提前放开 closed 的实现一律 closed。

**术语裁决（Lead 2026-11-02 已裁决）**：登记＝写入卡面与 Plan；启动＝派发/认领并开始改动；未启动包含不得引入依赖、不得写骨架或接线代码。本口径与「对下游门禁的影响」条的两级门禁口径（评审门禁 / 启动前置）同为解释性口径，效力低于约束正文，冲突以约束正文为准；约束 9 正文本轮不改动，若日后需升格为约束正文，走下一次 ADR 修订。

**本轮未处置（保持 OPEN，不因本修订关闭）**：架构侧 `R02-F005`/`R03-F004`（约束 2 正文与独立 PostgreSQL 形态冲突，需改约束正文）、`R03-F003` 的跨 ADR 部分（ADR-0032 正文与卡面更正）、`R03-F009`/`R03-F010`/`R03-F012`（`tasks.md` 卡面与行号漂移，属规划角色）、`R03-F011` 的可读性残留（候选 A 行结论列标注）；安全侧 `R03-F009`（Lead 裁决载体）。架构 `R02-F004`/`F007`/`F008`/`F009` 与 `R03-F001`/`F002`/`F005`/`F006`/`F007`/`F008`/`F013`、安全 `R03-F001`–`F008` 已在 r08 正文处置。处置依据与理由见作者报告 r08 段与「待授权变更清单」；其中约束 2 正文超出本卡写集。


#### r08 补充（见证状态机、CAS 键、批准记录、演练四档与门禁两级；全部是要求与否证条件，不是证据）

本小节处置架构侧 `B13J-D05-R03-F001`–`F013` 与安全侧 `B13J-D05-SEC-R03-F001`–`F009` 中经 Lead 采纳的部分，并把见证语义收敛为唯一合法状态机；以下仍是要求与否证条件，不是已取得的证据，任何一条未在实施/验证卡中取证，对应结论不成立。r09 起，本小节的状态机、CAS 键与恢复映射已被 r09 就地修订（见下「r09 补充」）；本小节的 r08 原表述不再作为实现依据。

**见证状态机（唯一合法形态，r09 就地替换 r08 的两态枚举）**：见证只持久 `(w, r, token)`：`w` 为已终结 revision 的最大值（committed 或 abandoned），`r` 为至多一个在飞预留且恒等于 `w + 1`，`token` 只在有预留时非空。合法状态由下列不变量集合定义，不再枚举「状态 (1)/(2)」：预置条件——无预留时 `r` 为空，有预留时 `r == w + 1`；预留前——事件表 revision ≤ `w` 均已终结、revision `w + 1` 无任何事件行；推进——`w` 只允许推进到 `r`，不得跳号、不得回退；核对（启动与每次推进前）——(i) 见证不可读或不可写 → closed；(ii) 事件表存在 revision > `w` 的终结行 → closed；(iii) 事件表 revision ≤ `w` 存在未终结行且见证无预留 → closed；(iv) 有预留 `r` 且事件表 `r` 上已有终结行 → 幂等完成（把 `w` 推进到 `r` 并清预留，不重写终结行），不是 closed；(v) 有预留 `r` 且 `r` 上无终结行 → 按 ADR-0019 `:468` 三分支处理；(vi) 任何情况下不得用观测 head 抬高见证。连续 abandoned 合法：每次终结把 `w` 推进 1，下一次预留仍为 `w + 1`，不存在「差值 2 即 closed」。已结算 revision 上的 pending 行是终结行的引用目标，永久保留，不属于「下一待预留 revision」；「下一待预留 revision 无任何事件行」只指 `w + 1`。向调用方返回成功之前，见证必须已推进到该 revision，不存在任何允许见证落后的豁免态。

**CAS 键、事件表字段与恢复映射**：事件表每行携带 `journal_id`、`namespace`、`revision`、`epoch`、`head_sequence`、`head_hash`、`kind`（pending/committed/abandoned）、`pending_token` 与操作标识；事件表唯一键为 `UNIQUE(namespace, revision, kind)`，pending 行另有 `UNIQUE(namespace, revision, pending_token)`，终结行由 `(namespace, revision) WHERE kind IN ('committed','abandoned')` 的部分唯一索引保证同一 revision 至多一个终结行，因此「跳过 pending 直接写 committed」「同一 revision 换 token 重写」「已 committed 再追加 abandoned」与「已 abandoned 再追加 committed」在约束层都不可能；`epoch` 缺失或与 Journal 当前 epoch 不匹配即 closed（ADR-0019 的 epoch 不匹配停服要求）。ADR-0019 的旧 committed → pending → committed 两段 CAS 映射为「pending 行落库 → committed 行携带同一 token 落库」；ADR-0019 的「清除未发布 pending」映射为追加 abandoned 事件（引用同一 pending 行与 token，角色无 `UPDATE`/`DELETE`），head 派生忽略 abandoned 且该 revision 计入结算水位 `w`，不得再被不同 token 或不同 payload 复用，下一次 CAS 使用更大的 revision。恢复判定按见证三元组 × 事件表状态取六个确定分支：(i) 有预留 `r`、无 pending 行 → 用同一 token 补插 pending 行后追加 abandoned，并把 `w` 推进到 `r`；(ii) 有预留、有 pending、文件仍为旧头且可验证临时文件仍存在 → 用同一 token 提交 committed（完成发布）并推进 `w`；(iii) 有预留、有 pending、文件仍为旧头且无临时文件 → 追加 abandoned 并推进 `w`；(iv) 有预留、文件已为新头、无 committed → 用同一 token 提交 committed 并推进 `w`；(v) 有预留、`r` 上已有同 token committed 或 abandoned 终结行 → 幂等完成，只把 `w` 推进到 `r` 并清预留，不重写事件行；(vi) 文件既非旧头也非新头、token 不匹配、`r` 上存在多个终结行、`epoch` 不匹配或见证不可读写 → closed。负向要求（验证卡必测）：两个写入者并发为同一 revision 预留并插入 pending，恰一个成功；失败者不得插入 committed；无 token 或 token 不匹配的 committed 必须失败；committed 之后再插 abandoned（反之亦然）必须被数据库约束拒绝。

**批准记录最小字段**：两个具名保管主体（anchor 连接凭据、见证写凭据）及其各自注入者身份与注入时间、两个秘密存储位置、轮换窗口是否可合并；配置文件 owner 与写入者；见证产品/形态/受支持大版本与 EOM 门禁、见证端点（地址、来源地址范围、TLS CA 与主机名）与启动时比对方式、五项保证等级（见 r07 补充段「位置与形态」条）的取证引用、见证保留期与 anchor 备份保留期的比较结论、见证受支持清单的具名 owner；最近一次四档演练证据与出具人/日期。否证条件：同一秘密存储、同一账号可读两侧、同一变更窗口可同时轮换两侧口令、配置写入者与 anchor 实例管理员为同一主体且无双人批准、见证端点未钉进批准记录或与配置值不一致、见证保留期短于 anchor 备份保留期且无法证明不可降低——命中任一即 closed。

**威胁模型边界**：模型内的对手是持有 anchor 备份/恢复权与进程终止权的 anchor 实例管理员（或同一 DBA），其不能写见证；可修改 han-server 配置文件的部署/配置管理主体同样列入模型内——其不得与 anchor 实例管理员为同一主体，配置变更须由第二主体批准并留审计，且 witness 端点必须与批准记录钉定的值比对一致，否则视为可伪造见证、一律 closed。模型外是 han-server 宿主被攻陷后同时读取两侧凭据——该情形按 ADR-0029 已登记的残余风险处理，不在本 ADR 的检测范围内，本 ADR 不因此放宽任何凭据或 ACL 要求；也不得以「两个秘密分属不同保管主体」反推配置权威已被隔离。

**演练四档与约束 7 映射**：回退备份分四档取证——(A) 与当前见证相等的备份（对照，判为无回退）、(B) 恰为见证 −1 的备份（必须 closed）、(C) 更旧备份（必须 closed）、(D) 新副本冷启动读到 anchor 观测水位低于见证（必须 closed）；判据是「服务任何 `enter`/`install` 之前即 closed 且无任何 anchor 写入」，重启后首次核对不得使用进程内缓存。与约束 7 验证项的映射：整实例回退并重启 → missing/rollback；新副本冷启动 → 崩溃恢复；业务库回退 → stale/missing；并发同 revision 写入与 committed/abandoned 互转 → CAS 冲突；token/epoch/ACL/TLS 失败 → 权限与协议失败。另在「预留、pending、终结行、见证推进」四个点各注入一次崩溃，重启后必须落到 r07 补充段「写入次序与崩溃窗口」条／r09 补充段「见证状态机」条的确定分支之一（含幂等完成），不得永久 closed、不得手工修库。

**门禁两级（解释性口径，效力低于约束正文）**：约束 9(b) 的入口约束实施/验证卡的登记；启动另有「对下游门禁的影响」条的 (i)–(v) 前置，须由用户/组织作出并登记，未满足即不得派发/认领/开始改动；二者不可互相替代。评审门禁只有一条——本决策修订被架构与安全独立评审接受，不以任何候选卡产出为前提。约束 9(b) 正文的「登记入口」不被本段改写，本段只解释其执行语义。

**跨 ADR 对齐（已由 ADR-0032 r04 起处置、r05 收口，不再是未修复冲突）**：本 ADR 规定 `Exclusive` 在候选 S4.24b 交付前保持 `recoveryClosedPort`；ADR-0032 已于 2026-11-02 修订为 `adr-0032-r04`、同日续修至 `adr-0032-r05`（r04 把跨 ADR 对齐落到第 8 条的候选 S4.24b 节点，r05 再把「解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据」并入同一节点以消除顺序自指；当前修订串见 ADR-0032 修订行（`adr-0032-r05`）），其解除顺序第 8 条已把「`Exclusive` 离开 `recoveryClosedPort`」移到候选 S4.24b 节点，并保留候选 S4.24b requires 候选 S4.23（ADR-0032 解除顺序第 8 条候选 S4.24b 节点，符号锚），与本 ADR 的三段式一致；原「两者不一致、须由下一次修订更正」的记述自此仅作历史记录。本 ADR 的更严规定继续作为兜底：候选 S4.24b 交付前提前放开 closed 的实现一律 closed；其它尚未对齐的跨 ADR 点（若有）同样以本 ADR 更严规定为准。

**本轮仍未处置（保持 OPEN，不因本修订关闭）**：架构侧 `R03-F003` 的跨 ADR 部分已由 `adr-0032-r04`（D07）处置（`tasks.md` 卡面行号锚已漂移，交由规划角色按符号锚核对）、`R03-F004`（约束 2 正文）已由 r10 处置（`adr-0030-r10`，本节不再列为 OPEN）、`R03-F009`/`R03-F010`/`R03-F012`（卡面与行号漂移，属规划角色）、`R03-F011` 的可读性残留；安全侧 `R04-F004`（同约束 2 正文）已由 r10 处置（`adr-0030-r10`，本节不再列为 OPEN）、`R04-F008`（S4.23 卡面未承接见证状态机与配置解析，属规划角色）、`R04-F009` 的「注入者身份可从运行痕迹证明」部分（fd 编号、注入时间与端点比对已在本轮固定，运行痕迹取证属实施卡）。逐条理由与建议见作者报告 r09 段与「待授权变更清单」；其中约束 2 正文与 ADR-0032 修订已分别在 r10、r04 处置完毕，剩余未处置项均属规划角色卡面或实施卡范围。第六轮结果另记：架构侧 `R05-F004` 已由 `adr-0032-r05` 处置（解除顺序第 8 条的证据节点并入候选 S4.24b 节点，链中不再有「证据先于产出者」）、安全侧 `SEC-R05-F005` 已由 Lead 裁决（`lead-authorization-r11.md` §E：登记≠启动；持久文本修复由 `identity-auth-D06` 产出的 `adr-0031-r05` 承接）与 `SEC-R05-F006`（本 ADR 现行态陈述与 OPEN 清单陈旧）已由 r12 处置；以上三项移出 OPEN，其余各项仍如上保留。

#### r09 补充（终结行互斥、崩溃恢复、CAS 键与事件表字段、批准记录与威胁模型；全部是要求与否证条件，不是证据）

本小节处置架构侧 `B13J-D05-R04-F001`–`F008` 与安全侧 `B13J-D05-SEC-R04-F001`–`F009` 中经 Lead 采纳的部分。见证状态机正文以 r09 补充段「见证状态机」条为准（r09 就地替换 r08 的两态枚举），写入次序与崩溃窗口以 r07 补充段「写入次序与崩溃窗口」条为准，恢复判定以 r09 补充段「见证状态机」条为准；本段只补三处此前未固定、且必须由数据库层而非应用判断承担的不变量。以下仍是要求与否证条件，不是已取得的证据。

**终结行互斥（数据库层）**：同一 revision 至多一个终结行（committed 或 abandoned），由 `(namespace, revision) WHERE kind IN ('committed','abandoned')` 的部分唯一索引在数据库层保证；已 committed 的 revision 不得再追加 abandoned，已 abandoned 的 revision 不得再追加 committed，违反即约束层失败。该索引与 pending 行的 `UNIQUE(namespace, revision, pending_token)`、事件行的 `UNIQUE(namespace, revision, kind)` 共同构成 CAS 键；仅有 `UNIQUE(namespace, revision, kind)` 时，持 `SELECT`/`INSERT` 的写入者可在 committed 之后追加 abandoned 把已提交 head 静默抹回，故终结行互斥不可省略。

**崩溃恢复与幂等完成**：崩溃恢复只允许 r09 补充段「见证状态机」条的六个确定分支；其中「有预留、`r` 上已有同 token 终结行」是幂等完成态（只把 `w` 推进到 `r` 并清预留，不重写事件行），任何实现不得把它判为 closed，也不得用 `UPDATE`/手工修库绕过；「有预留、无 pending 行」必须先用同一 token 补插 pending 再追加 abandoned，不得凭空推进见证。连续 abandoned 合法：每次终结把 `w` 推进 1，下一次预留仍为 `w + 1`，不存在「差值 2 即 closed」。

**事件表字段与 epoch**：事件表每行至少携带 `journal_id`、`namespace`、`revision`、`epoch`、`head_sequence`、`head_hash`、`kind`、`pending_token` 与操作标识，字段语义与 ADR-0019 的 anchor 必填状态一一对应；`epoch` 缺失或与 Journal 当前 epoch 不匹配即 closed。ADR-0019 `:468` 的「文件仍为旧头且可验证临时文件仍存在 → 完成发布」必须映射为用同一 token 提交 committed，不得一律映射为 abandoned。

**批准记录与威胁模型**：批准记录最小字段见 r08 补充段「批准记录最小字段」条，其中必须包含五项保证等级的取证引用与见证端点；可修改服务配置的部署主体列入威胁模型，须与 anchor 管理员分属不同主体或双人批准，witness 端点必须与批准记录比对。r05 结论 (b) 的配置写入者分离、r07 补充段「写入权限与批准记录」条的双 fd（3/4）与注入者/时间、r08 补充段「CAS 键、事件表字段与恢复映射」条的字段与 r08 补充段「威胁模型边界」条的边界共同构成本项。

**跨 ADR 对齐（已由 ADR-0032 r04 起处置、r05 收口）**：见 r06／r07／r08 补充段的三处 ADR-0032 对齐条；ADR-0032 已于 2026-11-02 修订为 `adr-0032-r04`、同日续修至 `adr-0032-r05`（r04 把跨 ADR 对齐落到第 8 条的候选 S4.24b 节点，r05 再把「解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据」并入同一节点以消除顺序自指；当前修订串见 ADR-0032 修订行（`adr-0032-r05`）），其解除顺序第 8 条已把「`Exclusive` 离开 `recoveryClosedPort`」移到候选 S4.24b 节点（ADR-0032 解除顺序第 8 条候选 S4.24b 节点，符号锚），与本 ADR 的 `Exclusive` 三段式一致，原冲突已消除；本 ADR 更严规定继续作为兜底——候选 S4.24b 交付前提前放开 closed 的实现一律 closed。

**本轮仍未处置（保持 OPEN）**：约束 2 正文已由 r10 处置（`adr-0030-r10`，本节不再列为 OPEN）、ADR-0032 解除顺序与 `tasks.md` 三处卡面已由 D07（`adr-0032-r04`）处置（卡面行号锚由规划角色核对）、S4.23 卡面未承接见证状态机与配置解析（规划角色）、`tasks.md`/Plan 中写死的 r05 修订号字面与行号锚（规划角色）。逐条见作者报告 r09 段与「待授权变更清单」。第六轮结果：`R05-F004`（ADR-0032 解除顺序自指）已由 `adr-0032-r05` 处置，`SEC-R05-F005` 已由 Lead 裁决（§E，`identity-auth-D06` 的 `adr-0031-r05` 承接持久文本修复）、`SEC-R05-F006` 已由 r12 处置，三项移出 OPEN；其余保持。

#### r13 补充（启动前置 (iii)(iv)(v) 建议值登记：六槽责任主体、目标环境只读实测与见证形态；AI 代理/模拟，不是真实验证）

本小节按用户 2026-11-02 在本会话的明确指令「确认按建议执行」落定。用户同时以项目 owner 身份声明：本项目为测试项目，用于验证 ADD 流程完整性；该口径与已接受的 ADR-0028（`framework.md:816`–`:854`）一致——模拟环境替代、AI 代理评审签收与责任人建议值均按 ADR-0028 的记录方式标注，不冒充真人。本小节只把 「启动前置 (i)–(v) 已登记建议值」条 中仍为 unknown 的 (iii)(iv)(v) 登记为「建议值已登记（AI 代理/模拟）」，不补齐任何事实取证，不改变「首期不提供」结论、约束 1–10 正文与解除条件，也不构成评审 PASS、人的接纳或生产批准。「启动前置 (i)–(v) 已登记建议值」条 的「(iii) 当前 unknown」按 2026-11-02 登记时点保留为历史；自 r13 起，(iii)(iv)(v) 的状态以本小节为准，「启动前置 (i)–(v) 已登记建议值」条 原文不回写。

**A. (iii) 六槽责任主体（建议值，AI 代理，非真人）**

下表登记六槽责任主体，并另列配置变更第二批准人。所有主体均为 AI 代理，Agent ID 为本会话的真实委派标识；不是真人任命，也不是组织侧书面任命。

| 槽位 | 职责 | 代理主体与真实 Agent ID | 状态 |
|---|---|---|---|
| anchor 实例 owner + 备份 owner | anchor 专用实例、独立存储卷与独立备份链的运维与证据保留 | Faraday（`01a19bc5-06c4-7062-941c-375154461a2d`） | 建议值（AI 代理，模拟） |
| anchor 连接凭据保管主体 | anchor 连接口令的保管与一次性注入授权 | Herschel（`01a19b55-f062-7740-87ac-733db14c3d2b`） | 建议值（AI 代理，模拟） |
| 见证写凭据保管主体 + 见证 owner | 见证写口令保管、见证端点与五项保证等级取证 | Popper（`01a19b56-1baf-78f3-9aee-f8d39b17652f`） | 建议值（AI 代理，模拟） |
| 注入者 | anchor 侧与见证侧口令的两次独立注入 | anchor 侧 Lead（主会话，无 Agent ID，身份未取证）；见证侧 Rawls（`01a19b94-d02d-7702-a6f1-8861ec458ec3`） | 建议值（AI 代理，模拟） |
| 版本清单 owner | anchor 与见证两份受支持清单的同一维护主体（含 EOM/升级门禁） | Rawls（`01a19b94-d02d-7702-a6f1-8861ec458ec3`） | 建议值（AI 代理，模拟） |
| 部署 owner + 配置文件写入者 | 部署、配置文件写入与 witness 端点钉定 | Lead（主会话，无 Agent ID，身份未取证） | 建议值（AI 代理，模拟） |
| 配置变更第二批准人（附加控制） | `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 变更的第二主体批准与审计 | Herschel（`01a19b55-f062-7740-87ac-733db14c3d2b`） | 建议值（AI 代理，模拟） |

逐条禁止重合校验（对应 r07 补充段「写入权限与批准记录」条与 r08 补充段「威胁模型边界」条）：

- anchor 管理员 ≠ 见证保管/见证 owner：anchor 实例与备份 owner 为 Faraday，见证写凭据保管与见证 owner 为 Popper，二者不同。
- 两侧凭据保管互异：anchor 连接凭据保管为 Herschel，见证写凭据保管为 Popper，二者不同。
- 见证注入者/保管 ≠ anchor 注入者/保管：见证侧注入者为 Rawls、保管为 Popper；anchor 侧注入者为 Lead、保管为 Herschel，四者互不相同。
- 配置写入者 ≠ anchor 管理员：配置文件写入者与部署 owner 为 Lead，anchor 实例管理员为 Faraday，二者不同；配置变更另有 Herschel 作为第二批准人。
- 版本清单单一主体：anchor 与见证两份受支持清单均由 Rawls 维护，满足「对下游门禁的影响」条对「同一责任主体」的要求。

以上分离是形式/模拟分离：所有槽位由同一模型家族、同一宿主上的 AI 代理承担，组织分离控制**未真实成立**，不构成真人任命，也不满足 r06 补充段「独立网络入口与独立凭据」条与 r08 补充段「威胁模型边界」条对「不同管理域」或「不同主体」所要求的真实组织含义。任何消费者不得据此宣称真实职责分离已建立；该登记只把「启动前置 (i)–(v) 已登记建议值」条 (iii) 从 unknown 变为「建议值已登记（AI 代理/模拟）」。该分离只是**记录性分离**，不构成真实组织隔离；跨槽兼任见 r14 补充段。

**B. (iv) 目标环境（2026-11-02 只读实测；不是验证）**

2026-11-02 对本机目标环境的只读实测结果如下；以下只是只读观测，不是环境验证、不是部署、不是故障域证明：

- Windows 本机 PostgreSQL：版本 `18.6 on x86_64-windows`，data dir `C:/Program Files/PostgreSQL/18/data`，端口 `5433`，`listen_addresses=127.0.0.1`（仅 loopback），`pg_is_in_recovery=f`。
- 该集群现状：只有 `postgres`／`template0`／`template1`，**无 `han` 库**；另有遗留 3,736 个非系统 schema 与 34 组 `han_s15_mig_*/app_*` 临时角色，**不是干净基线**。
- WSL：发行版 `Ubuntu`，VERSION `2`，状态 `Stopped`；`<WSL_VHDX>`（`ext4.vhdx`） 为 1,190,133,760 B。
- 结论：业务 PostgreSQL 与 anchor **不得共用同一集群/数据目录**（r05 结论 (c) 判据 1），该 5433 实例只能作为**业务库**；anchor 须在 WSL 内另建独立集群（独立集群与数据目录、独立网络入口），且「同一物理机」必须登记为**残余风险**——它不满足 r05 结论 (c) 判据 1 的严格读法（不与业务 PostgreSQL 共用主机、集群与数据目录），只能按「独立集群 + 独立数据目录」的较弱形态登记，不能单独作为独立故障域成立的证据。

**C. (v) 见证（建议值）**

见证形态建议值取「Windows 侧域外单调计数器文件」，与 WSL 内 anchor 集群分属不同数据目录、不同备份链；但它与 anchor、业务库同处一台 Windows 主机，**不满足** r07 补充段「位置与形态」条「与 anchor 不同主机或集群」的要求，也不能主张位于业务库故障域之外。该形态只作为模拟/演练桩，不得用于任何真实等级判定；同机故障域、无独立存储卷与无独立备份链作为残余风险登记（见 r14 补充段）。r07 补充段「位置与形态」条的五项保证等级逐项状态如下，不得读成已取证：

| 保证等级（r07 补充段「位置与形态」条） | r13 状态 |
|---|---|
| (1) 单调不可递减、无重置接口 | 模拟（建议值设计），未取证 |
| (2) 跨 anchor 故障域持久，确认写不随 anchor 实例或其备份恢复丢失 | unknown |
| (3) anchor 管理员不能经任一备份/快照/恢复路径降低见证，或见证保留期 ≥ anchor 备份保留期 | unknown；保留期比较结论**待填** |
| (4) 每次推进留可审记录（时间、namespace、revision、token 摘要、操作者身份） | 模拟（建议值设计），未取证 |
| (5) 支持 `(w, r, token)` 三元组原子条件写，或能以追加日志唯一派生并在读取时验证 | unknown |

见证支持 `(w, r, token)` 三元组原子条件写的验证仍为 **NOT RUN**；见证保留期与 anchor 备份保留期的比较结论列为**待填**。本 ADR 不声称见证已部署、已持久化、已抗回退或已取证。

**D. 门禁状态（不得拔高）**

(iii)(v) 由 unknown 变为「建议值已登记（AI 代理/模拟）」；(iv) 已登记实测值，来源地址范围与升级窗口仍 unknown；本级门禁状态为**未放行**。模拟等级只允许登记与文档／证据准备，不解除「对下游门禁的影响」条的评审门禁（本决策修订须经架构与安全独立评审接受）与 (i)–(v) 启动前置；截至本修订评审门禁与 (i)–(v) 启动前置均未满足（两路接受均未取得），模拟证据亦未补齐，派发、认领与开始改动仍受「对下游门禁的影响」条／「启动前置 (i)–(v) 已登记建议值」条约束，`S4.23` 一类卡不得派发/认领/开始改动；真实等级仍未满足。

同时明确：不得据此宣称生产就绪；不得宣称约束 8 满足（ADR-0031 解除条件九条、ADR-0032 解除顺序）；不得宣称真实故障域或真实组织分离成立；不得宣称 S4.24a 已实施。生产 recovery 的 `enter`／`install`／`resume`／`unlock` 全部继续 closed，「首期不提供」结论不变；本小节自身不构成评审接受、评审 PASS、人的接纳或生产批准，模拟等级登记不等于真实等级满足。本小节与约束正文冲突时以约束正文为准。

#### r14 补充（r13 两路独立评审处置与 (b) 契约 errata；全部是要求与否证条件，不是证据）

本小节处置架构侧 `review-arch-r08` 的 `ADR-R08-01`–`ADR-R08-10` 与安全侧 `review-sec-r08` 的 `ADR-S08-01`–`ADR-S08-05`，并作为 r05 结论 (b) 的 errata 子段登记 2026-11-02 anchor 环境实测发现的契约缺陷。两份触发报告为 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r08/review-arch-r08.md`（28,517 B / `D89B8B65CA5A572E74BA5144E2DB908103F93BAF4F7CEB1BDAD8394DDD4B0E2A`，REJECT）与 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r08/review-sec-r08.md`（19,344 B / `81BD8CEE0441156EB3BB08BC268FBC8DCDA50872B551603C611E9C16842B0D08`，PASS WITH CONDITIONS）。以下全部是要求与否证条件，不是已取得的证据；本小节不构成评审接受、评审 PASS、人的接纳或生产批准。

**门禁状态（ADR-R08-01／ADR-S08-01）**：r13 补充段 D 中把模拟等级直接换算为放行派发的自授权表述已删除，现行门禁状态为**未放行**；r13 追溯行中的同一历史记述自 r14 起作废，不得再被引用为放行依据。模拟等级只允许登记与文档／证据准备，不解除「对下游门禁的影响」条的评审门禁（本决策修订须经架构与安全独立评审接受）与 (i)–(v) 启动前置；截至本修订，两路评审均未接受本修订（架构 r08 为 REJECT、安全 r08 为 PASS WITH CONDITIONS），模拟证据亦未补齐，故 `identity-auth-S4.23`、`identity-auth-S4.24a`、`identity-auth-S4.24b` 等实施／验证卡不得派发、认领或开始改动。真实等级仍未满足；模拟等级不构成双人控制、不构成真实组织隔离，也不构成真实故障域证据。

**行号锚（ADR-R08-02／NEW-R10-01）**：本 ADR 指向 ADR-0032 的锚点一律为符号锚——「ADR-0032 修订行（`adr-0032-r05`）」与「ADR-0032 解除顺序第 8 条候选 S4.24b 节点」。自 r16 起，本 ADR 段内指向自身的引用同样一律为符号锚（r05 结论 (a)–(d)、「对下游门禁的影响」条、「启动前置 (i)–(v) 已登记建议值」条、r06–r09 与 r13／r14 补充段各条标题、r13／r14 追溯行），不再以绝对行号作规范性定位；启动前置条原文与 r01–r15 追溯行中的历史行号记述按不回写惯例保留，只作历史记述，不构成定位依据。逐条改写与复核结果见 r16 追溯行与 r16 作者报告。

**见证与故障域（ADR-R08-03）**：anchor 目标环境是**同一台 Windows 主机上的 WSL2 实例**，业务库是同机 Windows PostgreSQL `18.6`，监听 `127.0.0.1:5433`；见证建议值是同一主机 Windows 侧的单调计数器文件（`<用户目录>\.han-witness\witness-state.json`，185 B / `56E7C61366D44B54CAAE8B03AB8477AAE44C469D0327AB749097A0DFEB04ED72`），本机管理员对该文件有完全控制、可重置。按 r05 结论 (c) 判据 1 与 r07 补充段「位置与形态」条「与 anchor 不同主机或集群」的严格读法，该组合判定为**不满足**，不得写成「满足」或「部分满足」。同机共用主机故障域、无独立存储卷、无独立备份链、见证可被本机管理员重置，均作为**残余风险**登记；这些残余风险使该环境只能用于模拟／演练，不能用于任何真实等级判定或生产解锁。

**目标环境登记补全（ADR-R08-04）**：(iv) 的实测值为：anchor 集群 PostgreSQL `16.15`，数据目录 `/var/lib/postgresql/16/anchor`，端口 `5434`，`listen_addresses=localhost`，无独立网络入口；WSL `Ubuntu` VERSION `2`，`<WSL_VHDX>` 与业务库同主机；无独立存储卷，无独立备份链；来源地址范围因无独立网络入口而无外部来源可登记（unknown）；anchor 管理员仍为集群超级用户，本地 peer 登录可达。逐项判定：独立主机／集群与数据目录——**不满足**；独立存储卷——**不满足**；独立备份链——**不满足**；独立网络入口与来源地址范围——**不满足**（来源地址范围 unknown）；独立升级／维护窗口——unknown。(iv) 已登记实测值，来源地址范围与升级窗口仍 unknown，不得读成已满足「启动前置 (i)–(v) 已登记建议值」条 (iv) 或 r05 结论 (c)。同段残余风险：anchor 管理员仍为集群超级用户，可绕过索引与触发器（临时禁用触发器、`session_replication_role=replica`、删除索引或直接改表），因此 42501／55000 的拦截只对受限写入角色成立，不对超级用户成立；该绕过不在本机自测覆盖内，只能由域外见证与批准记录比对检出（见「r15 补充」）。

**跨槽兼任与身份（ADR-R08-05／R08-06／R08-07、ADR-S08-02／S08-03）**：Lead 兼任 anchor 侧注入者、部署 owner 与配置文件写入者，且两槽均为主会话、无 Agent ID（身份未取证）；Herschel 兼任 anchor 连接凭据保管主体与配置变更第二批准人；Rawls 兼任见证侧注入者与两份版本清单 owner。上述兼任在字面上不违反 r07 补充段「写入权限与批准记录」条／r08 补充段「威胁模型边界」条的明面禁止项，但第二批准对 anchor 凭据保管、配置写入对注入、版本清单对见证形态的独立性均被削弱。三者一律登记为**记录性分离，不构成真实组织隔离**；批准记录出具人暂登记为 Lead、批准人为 Herschel，二者与配置写入者／注入者的独立第三方关系**未取证**。任何消费者不得据此宣称真实职责分离或双人控制已建立。

**批准记录字段（ADR-S08-04／ADR-R08-05）**：r08 补充段「批准记录最小字段」条的字段尚未备齐，至少缺：两个秘密存储位置、轮换窗口是否可合并、见证端点具体值（地址、来源地址范围、TLS CA、主机名）、五项保证等级的取证引用、见证保留期与 anchor 备份保留期的比较结论、最近一次四档演练证据与出具人／日期。本 ADR 只登记建议值，不把批准记录读成已完整；上述字段未备齐前，相关控制链在真实等级不成立。

**closed 范围与历史时点注（ADR-S08-05／ADR-R08-08）**：r13 补充段「D. 门禁状态」条已把 closed 范围写为生产 recovery 的 `enter`／`install`／`resume`／`unlock` 全部继续 closed。「启动前置 (i)–(v) 已登记建议值」条的「(iii) 当前 unknown」与末句启动禁令按不回写惯例保留；其 unknown 状态自 r13 起由补充段取代，但末句禁令继续有效，本修订未解除。两处并存时以更严者为准。

**代理范围与报告更正（ADR-R08-10／ADR-R08-09）**：六槽表中「职责」列即各槽的代理范围，限于文档登记、配置建议与模拟／演练，不含真实生产操作、真实凭据注入或生产装配；这是 ADR-0028 约束 4 所要求的代理范围登记。r13 作者报告 §3 把「ADR-0031 段首锚至 EOF」误标为「后缀」、§4 把新增块的行号区间少记 1 行（实测含末尾空行），该段自 r16 起以「r13 补充段」符号锚指称；更正登记于 r14 作者报告，r13 报告作为历史证据不回写。

**errata（对 r05 结论 (b) 的契约缺陷登记）**：r05 结论 (b) 要求终结行对 pending 行的引用由「非部分唯一键 + 触发器（或把 pending 行与终结行拆为同库两张表）」实现；在单表、单一 `pending_token` 列的形态下该要求不可满足：终结行必须携带与 pending 行相同的 token，因此必然与 `UNIQUE(namespace, revision, pending_token)` 冲突。证据为 `.spec/wip/recovery-runtime/evidence-b13j-d05/anchor-env-r01/run-03-selftest.log`：A3 在 committed 落库即返回 23505（`anchor_events_pending_uq`），该形态下任何 revision 都无法完成提交。

可行形态三选一：(1) pending 唯一键改为部分唯一索引 `(namespace, revision, pending_token) WHERE kind='pending'`，并由 `anchor_events_terminal_ref` 触发器校验终结行引用同 namespace、同 revision、同 token 的 pending 行，且 `journal_id` 与 `epoch` 与 pending 行一致（本环境采用，run-07 已实测 12 项正负用例）；(2) 拆为 pending 与终结两张表，pending 表以 `(namespace, revision)` 唯一并携带 token，终结表以 `(namespace, revision, token)` 外键引用 pending 表（ADR 原文已允许）；(3) 增加第二 token 列，保留原唯一键。取舍：形态 (1) 保持单表与 append-only，代价是偏离 r05 结论 (b)「非部分唯一键」的字面并依赖触发器；形态 (2) 保住「非部分唯一键」字面，代价是同库第二张表与更多 DDL（仍不引入第二张权威表）；形态 (3) 改动最小，代价是冗余 token 与漂移风险。本 ADR 采用形态 (1) 作为本环境与后续实施的登记形态，并明确标注它**偏离**「非部分唯一键」字面；该偏离只影响 pending 唯一键的数据库实现形态，不改变终结行互斥要求（r09 补充段「终结行互斥（数据库层）」条的部分唯一索引 `(namespace, revision) WHERE kind IN ('committed','abandoned')` 继续有效）；引用失败由外键 23503 变为触发器 23514，二者均属拒绝、不改变「失败即 closed」的应用层语义，但 SQLSTATE 与定位点改变（见 r14 补充段「并发负向用例实测」条）。本条与 r05 结论 (b) 互为交叉引用：r05 结论 (b) 已声明单表单 token 列的形态下以本条 errata 为准；本条只覆盖 pending 唯一键形态，不改变终结行互斥与 CAS 冲突判定。

并发负向用例实测（run-07，事务内回滚，残留行 0）：同 revision 换 token 再插 pending → 23505（`anchor_events_event_uq`）；committed 后再追加 abandoned → 23505（`anchor_events_terminal_uq`）；无 pending 的终结行 → 23514（引用触发器）；写入角色 UPDATE／DELETE／TRUNCATE／DDL → 42501；管理员（超级用户）改删裁 → 55000；自测后残留行 0。上述只是本机单集群的模拟自测，不是真实独立故障域验证，也不是生产就绪证据。约束与触发器的完整定义、旧约束名对应及 SQLSTATE 归属见「r15 补充」的约束命名表；超级用户绕过见 r14 补充段「目标环境登记补全」条。

**边界**：本小节与 r13 补充段不改变「首期不提供」结论、约束 1–10 正文与解除条件，不补齐任何 unknown，不构成生产就绪、约束 8 满足、真实故障域或真实组织分离成立、`enter`／`resume`／`unlock` 解锁、`identity-auth-S4.24a` 已实施；生产 recovery 继续 closed。本 ADR 的 r14 只做 r13 评审处置与 errata 登记，不构成评审 PASS、人的接纳或生产批准。

#### r15 补充（r14 两路复评一致性收口；纯文本一致性，不新增约束）

本小节只处置架构侧 `review-arch-r09` 的 `ADR-R09-01`–`ADR-R09-05` 与安全侧 `review-sec-r09` 的 `ADR-S09-01`–`ADR-S09-04` 及三项建议。触发报告为 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r09/review-arch-r09.md`（33,313 B / `E285D20217E5602DA52885D976613BAC54A3CC5285D5FE48BD6A5E16E5084E33`，PASS WITH CONDITIONS）与 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r09/review-sec-r09.md`（20,613 B / `FF38A203E3B134DCFE155AD3336337589FB1FD430384CB4B94F41D758117564E`，PASS WITH CONDITIONS）。两份报告均为**有条件**结论，不构成本 ADR 的两路接受；本小节只做文本一致性收口，不新增约束、不改变任何 unknown、不解除任何 closed，也不构成评审 PASS、人的接纳或生产批准。

**约束命名与 SQLSTATE 对应（ADR-S09-01）**：下表固定 r05 结论 (b) 与 r14 补充段「errata」至「并发负向用例实测」条所引约束与触发器的完整定义和现行名字，使第三方可仅凭本 ADR 复核。

| 约束／触发器 | 定义 | 拦截与 SQLSTATE |
|---|---|---|
| `anchor_events_kind_chk` | `CHECK (kind IN (''pending'', ''committed'', ''abandoned''))` | 非法 `kind` 被拒绝（23514） |
| `anchor_events_event_uq` | `UNIQUE (namespace, revision, kind)` | 同 revision 同 kind 重复 → 23505（run-07 A5：同 revision 换 token 再插 pending） |
| pending 部分唯一索引 `anchor_events_pending_uq_idx` | `UNIQUE (namespace, revision, pending_token) WHERE kind = ''pending''` | 同 token 再插 pending → 23505；旧的非部分唯一约束名 `anchor_events_pending_uq` 与复合外键 `anchor_events_pending_ref` 已由 r02 变体删除，旧形态的外键引用失败为 23503，但该形态在 run-03 的 A3 即先被唯一键 23505（`anchor_events_pending_uq`）拦截（见 `.spec/wip/recovery-runtime/evidence-b13j-d05/anchor-env-r01/run-03-selftest.log`），23503 只出现在同一日志 A6／A7 的期望注释、未实测；自 run-05 起 A6／A7 改为触发器校验并实测 23514 |
| 终结行部分唯一索引 `anchor_events_terminal_uq` | `UNIQUE (namespace, revision) WHERE kind IN (''committed'', ''abandoned'')` | 同一 revision 第二个终结行 → 23505（run-07 A4：committed 后追加 abandoned） |
| 引用校验触发器 `anchor_events_terminal_ref` | `BEFORE INSERT`，函数 `anchor.enforce_terminal_reference()`：终结行必须引用同 namespace、同 revision、同 token 的 pending 行，且 `journal_id` 与 `epoch` 与 pending 行一致 | 无 pending、token 不匹配或 `journal_id`／`epoch` 漂移 → 23514（run-07 A6／A7） |
| append-only 触发器 `anchor_events_append_only`／`anchor_events_no_truncate` | `BEFORE UPDATE OR DELETE`／`BEFORE TRUNCATE`，函数 `anchor.reject_mutation()` | 超级用户（表 owner）UPDATE／DELETE／TRUNCATE → 55000（run-07 B1–B3）；受限写入角色同一操作 → 42501（run-07 A8–A10）；超级用户可绕过索引与触发器（见 r14 补充段「目标环境登记补全」条） |
| 受限写入角色权限 | `REVOKE UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER`；无 DDL、非超级用户 | UPDATE／DELETE／TRUNCATE／schema DDL → 42501（run-07 A8–A11） |

SQLSTATE 归属：42501 只来自角色权限（ACL），不是约束；23505 来自三个唯一性对象（`anchor_events_event_uq`、`anchor_events_pending_uq_idx`、`anchor_events_terminal_uq`）；23514 来自 `anchor_events_terminal_ref` 触发器与 `anchor_events_kind_chk`，不是外键；55000 来自两个 append-only 触发器。以上名字与定义取自 `anchor-env-r01` 的受控脚本（`anchor-schema.sql`、`anchor-schema-r02.sql`、`anchor-schema-r03.sql`）；本 ADR 只登记形状，不把脚本全文、角色名、口令或连接信息写入决策文本。

**前序小节的口径收口**：r07 补充段「单一写入模型」条／「角色矩阵随模型固定」条、r08 补充段「CAS 键、事件表字段与恢复映射」条、r09 补充段「终结行互斥（数据库层）」条中 pending 行「`UNIQUE(namespace, revision, pending_token)`」的非部分唯一键写法，自 r14 errata 起按 `anchor_events_pending_uq_idx` 的部分唯一索引理解；r09 补充段「终结行互斥（数据库层）」条的终结行互斥部分唯一索引及其「不可省略」要求逐字不变。

**安全建议处置（不计入 finding）**：建议-1（审计留痕：凭据读取与注入的操作者、时间、目标、结果、来源，日志与错误串禁含秘密明文）不在本 ADR 的扩张范围——本 ADR 只登记批准记录字段（r08 补充段「批准记录最小字段」条）与配置变更的审计要求（r08 补充段「威胁模型边界」条）；运行期审计留痕归属 `identity-auth` S4.5 的实施/验证卡与 r05 结论 (d) 的 anchor 部署运维责任，由该卡登记并取证。建议-2（r13 追溯行内联作废标记）已采纳。建议-3（把 `/var/lib/postgresql/16/anchor` 脱敏为 `<PGDATA_anchor>`）不采纳：该路径只是本地实例数据目录指纹，不含凭据、账号或可直连组合，脱敏会削弱 r14 补充段「目标环境登记补全」条实测登记的来源可核性；安全评审亦未将其判为 finding。

**边界**：本小节与 r13／r14 补充段不改变「首期不提供」结论、约束 1–10 正文与解除条件，不补齐任何 unknown，不构成生产就绪、约束 8 满足、真实故障域或真实组织分离成立、`enter`／`resume`／`unlock` 解锁、`identity-auth-S4.24a` 已实施；生产 recovery 继续 closed。本 ADR 的 r15 只做 r14 两路复评的文本一致性收口，不构成评审 PASS、人的接纳或生产批准。

#### r17 补充（D09 文档型决策卡：生产恢复链不实施、B1–B7 处置口径与按 ADR-0028 的测试范围收口；不新增生产结论）

本小节按用户 2026-11-03 在本会话下达的 D09 文档型决策卡指令落定。确认来源为用户本轮明确授权「待确认项按建议值、不需要完全模拟正式项目推进」；该授权只覆盖决策文档写作，不构成人的成果接收、独立评审、生产装配批准或真实验证。本小节只写决策文本，不改任何代码、不改任何任务卡、不改设计正文。

**触发与输入**：`.spec/wip/recovery-runtime/evidence-b13j-s423/report.md`（22,574 B / 168 行 / `2587076B74923C563A808C521CAE21662EF26C4E10412EC982E4695864D40AF5`，结论 BLOCKED）§2 的 B1–B7；`.spec/wip/recovery-runtime/preflight-s4-23/preflight-record.md`（11,705 B / 125 行 / `2FEFBF074F753D53CBB1779A6D26BCBCD7EA470F1563BA0A16A68AEDEAB1F05F`）§8／§9 的 owner 决定与建议值；ADR-0028（`framework.md:816` 起）约束 1–4、6–7 与 ADR-0031／ADR-0032 的现行结论。

**与既有文本的口径关系（不回写历史）**：（a）`preflight-record.md` §1 记有「ADR-0030 正文修订停在 r16，不再出 r17」，该句为早前登记时点的记述，与本轮用户明确指令冲突；自本 r17 起以本轮指令为准，§1 原文不回写，冲突保留为历史事实并由 `.spec/wip/recovery-runtime/evidence-b13j-d09/report.md` 登记。（b）`preflight-record.md` §9 结论第三项记四张候选卡「可从『已登记、未启动』进入派发评估」；本小节决策 1 明确该四卡维持未派发、未认领、未实施，本修订以「不实施」口径覆盖该阶段性建议，不撤回、不改写 §8／§9 原文。（c）本小节不改变 r13／r14／r15 补充段与 r01–r16 追溯行的任何原文及其门禁表述。

**B1–B7 处置表（逐条正面处置；事实与锚点取自 S423 报告 §2）**

| 编号 | 报告事实（摘要，锚点见 S423 报告 §2） | 本修订处置 | 处置后状态 |
|---|---|---|---|
| B1 无已登记的生产 adapter ID 字面值 | `tasks.md:3154`／`:3176(b)`／`:3177` 要求按已登记 ID 解析；全仓唯一 ID 字面值只在测试夹具中出现，不是登记值；`design.md:3910`／`:3614` 与 `framework.md:926` 记「首期不提供」 | 不登记该值，不另选新值；归入决策 2 的「产能外 unknown」 | 保持 unknown（不实施） |
| B2 卡面目标与冻结结论正面冲突 | `tasks.md:3154` 要求实现四端口 adapter；`design.md:3612`–`:3616`、`:3910`、`framework.md:926` 均为首期 closed／不提供，二者不可同时成立 | 显式登记冲突，并以「不实施」收敛；不改任务卡、不改 design、不降级 ADR 结论 | 已登记冲突；不实施 |
| B3 witness 段无 schema、批准记录无机器可读载体 | ADR-0030 只给语义字段（`framework.md:952`、`:1013`）；`approval-record-form.md:39`–`:41` 与 §E 为空白；`preflight-record.md` §4 全部待填 | 不定义键名、类型、版本或文件格式；归入决策 2 | 保持 unknown |
| B4 anchor 见证协议与 `(w,r,token)` 存储 API 未定义 | `framework.md:1009` 只有语义状态机；`preflight-record.md:101`／`:115` 记域外见证 waived／不实施；`anchor-env-r01/README.md` 不存在 | 不定义协议版本、请求/响应格式、鉴权 wire 与存储 API；归入决策 2 | 保持 unknown |
| B5 commitment／source 存储载体缺失、`AnchorSnapshot` 令牌 ↔ `bigint` 映射未定 | 接口只有方法签名（`internal/recovery/runtime.go:184`–`:192`、`:200`–`:202`），`internal/recovery/journal.go:68`–`:74`；`migrations/` 无对应载体 | 不登记载体与映射规则；归入决策 2 | 保持 unknown |
| B6 Windows 平台不可达项 | `ExtraFiles` 不支持、`chmod` 仅 `0o200`、无 `O_NOFOLLOW`、ACL 含 `Authenticated Users Modify` | 不处置、不改代码；平台限制与「Windows 生产 unsupported」结论仍成立 | 保持成立 |
| B7 `recoveryRootInputs` 四端口覆盖 seam（非阻断） | `recovery_runtime.go:556`–`:567` 的 seam；生产 `main.go:634` 未传四端口；既有集成测试依赖该 seam | 不处置、不保留也不收紧；交 Lead 裁定 | 待 Lead 裁定 |

**决策 1：生产恢复链在测试项目内不实施**

- 六张卡维持未派发、未认领、未实施：`identity-auth-S4.5`（`tasks.md:656`）、`identity-auth-S4.6`（`:669`）、`identity-auth-S4.23`（`:3152`）、`identity-auth-S4.24a`（`:3186`）、`identity-auth-S4.24b`（`:3263`）、`identity-auth-S4.22`（`:3303`）。本修订不勾选任何 checkbox，不改这些卡的卡面、requires、写集或关闭条件。
- ADR-0030「首期不提供生产 CAS JournalHeadAnchor provider」（`framework.md:926`）不变；ADR-0031「首期不提供生产 restore 工具与批准主体」（ADR-0031 标题段）不变；ADR-0032「`Exclusive` 仍 closed」（ADR-0032 解除顺序第 8 条候选 S4.24b 节点）不变。
- 生产 `enter`／`install`／`resume`／`unlock` 仍全部 closed；本修订不解除约束 7 的任何一项，不解除约束 9 与约束 10 的次序，也不解除 ADR-0031 解除条件与 ADR-0032 解除顺序。
- 约束 9(b) 的判定不变：本修订不是「给出具体产品、凭据来源、独立故障域论证、部署与 ACL 结论」的决策修订，因此不构成实施/验证卡登记入口的满足；`S4.23` 一类卡的门禁状态不因本修订改变。

**B2 冲突的显式登记（不以改卡掩盖）**：`tasks.md:3154` 要求「实现 anchor/CAS adapter 的 `Commitments`／`Sources`／`Anchor`／`Exclusive` 四端口，并在同一 bundle 内绑定」，`:3182` 又在缺 adapter ID／配置路径／四端口合同时要求停止；而 `identity-auth/design.md:3612`–`:3616` 记三个端口首期解析为 `recoveryClosedPort`、`exclusive` 恒 closed，`:3910` 记「首期不提供生产 adapter」，`framework.md:926` 为同一结论。实现真实四端口 adapter 必然使 `Anchor`／`Commitments`／`Sources` 离开 closed，与上述冻结结论不可同时成立；保持全 closed 则「实现 adapter」无实际产物。本修订以「不实施」方式收敛该冲突：冲突作为已知事实保留在 `tasks.md` 与 design 原文中，等待后续决策修订统一处置，不由本修订改写卡面、design 或 ADR 结论来消除痕迹。

| 方案 | 实现机制 | 优势 | 劣势／后果 |
|---|---|---|---|
| A. 不实施（选定） | 六卡维持未派发、未认领、未实施，由本 ADR 登记该事实 | 不推翻已接受结论；不改卡面与 design；不制造未经评审的生产 CAS provider；不引入新依赖或新契约 | 生产恢复链在测试项目内始终不可用；B1／B3／B4／B5 长期保持 unknown |
| B. 实现四端口 adapter | 按 `tasks.md:3154` 交付生产 adapter 与 registry 解析 | 满足卡面目标 | 推翻 `framework.md:926` 与 `design.md:3612`–`:3616`／`:3910` 的已接受结论，等于在无评审、无凭据、无独立故障域、无见证协议的条件下落地生产 CAS provider |
| C. 只交 fail-closed 骨架并下调卡目标 | 交付恒 closed 的 adapter 骨架，改写卡面目标 | 表面推进卡状态 | 仍改变卡面合同与设计结论；产物无行为价值；需改 `tasks.md`，超出本卡写集与授权 |

选择 A。

**决策 2：B1／B3／B4／B5 的登记口径（统一记为「产能外 unknown」）**

以下六项在本项目内**不予登记**：

1. adapter ID 字面值（B1）；
2. witness 段 schema（B3）；
3. anchor 见证的传输协议（B4）；
4. `(w,r,token)` 读-条件写存储 API（B4）；
5. commitment／source 存储载体（B5）；
6. `AnchorSnapshot` 字符串版本令牌 ↔ 事件表 `bigint` revision 的映射规则（B5）。

口径与约束：

- 理由是上述六项对应的实现（决策 1）在测试项目内不实施：登记后没有消费者，且会把未经评审的键名、协议或映射固化为对外合同。
- 统一记为**「产能外 unknown」**：不是「待实施」、不是「建议值」、不是「已登记」；任何消费者不得把本小节的这一记法读成已具备产品选型、协议或存储合同。
- **若未来恢复该链**，必须先由本 ADR 的新修订给出上述六项的具体值，并经架构与安全两路独立评审接受，才可进入实施/验证卡；在此之前，约束 7、约束 9、约束 10 与 ADR-0032 解除顺序一律不松动。
- 不得把测试夹具中的 ID 字面值、设计草案、`approval-record-form.md` 的空白表单或 `preflight-record.md` §4 的待填项写成「已登记」；`anchor-env-r01` 的受控脚本同样不构成本 ADR 的登记值。
- 不新增第三配置键：`HAN_RECOVERY_ANCHOR_ADAPTER` 与 `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 仍是唯一配置入口（约束 3 不变）。
- B6 的平台限制仍成立，本修订不处置、不改代码，也不据此要求 Windows 例外（约束 8 的「Windows 生产 unsupported」不变）。
- B7 的 seam 仍待 Lead 裁定；生产 `main.go:634` 仍未传四端口，生产语义仍由 registry 提供，本修订不对该 seam 作保留或收紧的结论。

**决策 3：按 ADR-0028 收口剩余可执行层（本修订唯一的解除内容）**

适用依据逐条为：ADR-0028 约束 1 适用范围（`framework.md:844`，覆盖「全部 active/todo 卡及其后续 ADD 推进」，其示例为指引而非穷举，未排除恢复链相关卡）；约束 2 分级验证（`:845`–`:847`，真实环境层可用模拟环境、确定性 fixture 或建议值替代）；约束 3 不得伪造 PASS（`:849`）；约束 4 AI 代理评审与签收（`:850`）；约束 6 任务关闭与残余风险（`:852`）；约束 7 安全底线不变（`:853`）。本修订只引用这些条款，不改其正文。

**3.1 `identity-auth-S4.4`（`tasks.md:638`）：允许以模拟等级执行**

- 允许以**模拟等级**执行：确定性端口 + 本机 PostgreSQL 18.6（仅回环）+ 模拟见证／备份卷替代真实独立故障域。
- 产出必须逐条显式标注「模拟环境验证、未真实验证」；不得把模拟结果写成 PASS 作为真实证据；无法模拟与未执行的部分写 `NOT RUN` 或 unknown（ADR-0028 约束 3）。
- 其 requires 中的 `identity-auth-S4.5`、`identity-auth-S4.6`（`tasks.md:649`）由本 ADR 作为**范围豁免**承接，登记名 `waived-for-test-scope`：该豁免只覆盖 S4.4 的模拟验证范围，不使 S4.5／S4.6 变为已实施，不改变其未派发状态，也不改变生产恢复链任何结论。
- 卡面「生产 Anchor 须来自已完成的 S4.5、生产 key retirement 仅在 S4.6 真实双前置 evidence 核验与 capability 原子单次消费成功后开放」（`tasks.md:652`）不因本豁免放宽：S4.4 的模拟结果不得用作生产 Anchor 或 key retirement 的证据，相关门禁保持 closed。
- 关闭条件中的真实备份/持久卷环境部分按 ADR-0028 约束 2 的真实环境层以模拟替代，并在本卡证据中逐项登记模拟边界与残余风险；人的业务接收仍不存在（ADR-0028 约束 4 只授权 AI 代理评审与签收，不产生人的接收）。

**3.2 `identity-auth-S2.2`（`tasks.md:323`）：目标环境与容量预算采用已登记建议值**

- 目标环境与容量预算采用 `preflight-record.md` §9 已登记的建议值：单主机 WSL2 + PostgreSQL 18.6（仅回环）；2 vCPU／4 GB／20 GB；100 并发会话；p95 ≤ 500 ms；错误率 < 0.1%。
- 上述数值是**模拟基线、非生产 SLO**，不构成容量批准、生产预算、部署目标或 production readiness。
- `requires identity-auth-S4.4`（`tasks.md:335`）由 S4.4 的模拟等级结果承接；S4.4 未产出模拟结果前，本卡不得据此写成通过。
- 卡面要求区分「合法 token/tuple」「目标环境可运行」「满足批准预算」三种结论（`tasks.md:330`）：本修订只使前两项可按模拟口径产出，第三项在无真实 benchmark 与采用记录前仍不得写 PASS。

**3.3 `identity-auth-S3.4`（`:473`）、`article-management-S1.4`（`:1209`）：真实部署不可得部分按模拟环境替代**

按 ADR-0028 约束 2 的真实环境层替代，逐条登记如下；两条均不得宣称真实部署通过或 production readiness。

| 卡 | 模拟了什么 | 替代了什么 | 残余风险 |
|---|---|---|---|
| `identity-auth-S3.4` | 以本机回环 PostgreSQL 18.6 与进程内入口复现注册／登录、管理员治理、删除用户与无归属处置、凭据恢复旅程 | ADR-0028 `:847` 所列「真实部署／可观测平台」中的「部署后真实入口」与「真实 Host」 | 未证明真实部署、真实 Host、真实旧备份恢复与第二真实 SID 签发；卡面明禁替代的部分（如 password-reset 不得以 fake／local SMTP 代替，`tasks.md:486`）不因本修订放宽，其未执行部分记 `NOT RUN`；`requires` 不因本修订减少 |
| `article-management-S1.4` | 以本机回环入口复现创建／浏览／修改／删除、停用可见性与无归属处置旅程 | 部署后真实入口与真实旧备份重放 | 未证明真实部署 E2E、真实旧备份重放与业务验收；`requires` 中的 `article-management-S1.5`（当前 `[-]`）状态不变，本修订不对其作豁免 |

> r19 更正：本段按 Lead 2026-11-03 F001 裁定 (A) 将 identity-auth-S3.6 移出模拟替代范围；S3.6 维持 blocked／未派发、真实条件未满足。

**决策 4：禁止条款**

- 不得据本修订宣称任何生产解锁、生产就绪（production readiness）、真实独立故障域成立、ACL 已验证，或 `S4.4`／`S2.2` 取得真实环境 PASS。
- 不得把模拟结果写成 PASS；模拟结论一律带「模拟环境验证、未真实验证」或 `NOT RUN`／unknown 标注（ADR-0028 约束 3）。
- 不得删除任何失败证据，不得改写任何已关闭卡的历史记录（含 checkbox、日期、关闭记录、requires 与写集），不得把 `blocked` 直接改成 `completed` 而不留处置说明（ADR-0028 约束 6）。
- 不得据本修订新增第二 runtime、router、pool、authority 或任何 fallback（ADR-0028 约束 7）。
- 本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准；ADR-0028 约束 4 的 AI 代理评审仍须在对应卡的独立评审中记录授权来源、代理主体、代理范围与「AI 代理评审，非真人签字」声明，不得冒充非作者真人评审。

**门禁边界**：r13／r14／r15 的「未放行」仅对决策 3 所指的**模拟等级测试范围**作窄豁免，且只解除 `S4.4` 对 `S4.5`／`S4.6` 的派发阻塞与 `S2.2` 对 `S4.4` 模拟结果的承接；ADR-0030 约束 7／9／10、ADR-0031 解除条件与 ADR-0032 解除顺序均不解除，生产 closed 不变；`S4.23`／`S4.24a`／`S4.24b`／`S4.22` 不得因此启动。本小节与约束正文冲突时以约束正文为准。

#### r18 补充（D09 r17 两路评审返工：S3.6 移出模拟范围、`waived-for-test-scope` 口径补全、B4 事实更正与 B7 裁定落定；不新增生产结论）

本小节按用户 2026-11-03 的 D09 文档型决策卡 r17 两路评审返工指令落定，只处置评审 findings 与 Lead 裁定，不回写 r17 原文、不改任务卡、design、`decisions/README.md`、代码或迁移。触发与输入：架构侧评审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-arch-r01/report.md`（44,951 B / `C8C5F8166207D04F9F4E4477582232B5D620B42597AC87FA1CDD62373AE29B7B`，CHANGES REQUIRED，F001–F009）；安全侧评审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-sec-r01/report.md`（20,991 B / `712E96ED4534362F27C286B3E6A5C013EAEFC3C80C1CF12B8A4E790900E31307`，CHANGES REQUIRED，3 项阻断 MEDIUM + 4 项 LOW）；Lead 对 F001 的裁定 `.spec/wip/recovery-runtime/evidence-b13j-d09/lead-ruling-f001.md`（1,712 B / `1369F23769E00B01F4F59BF6723C054A3D320C39358383D05CEF44AA780F0DBB`，采用保守项 (A)）；Lead 对 B7 的裁定 `.spec/wip/recovery-runtime/evidence-b13j-s423/lead-ruling-b7.md`（1,464 B / `C39AEED822201D8CE68C1D0434877A95BA7CCD92F958BD1280D468069AD95F53`）。

本小节是作者文档处置，不构成评审 PASS、人的接纳或生产批准。r17 原文与 r01–r16 追溯行逐字保留；下表「r18 新口径」只取代被点名条款，未点名部分继续有效。本小节与约束正文冲突时以约束正文为准。

**r17 条款取代表**

| 被取代条款（r17 符号锚） | r17 原口径摘要 | r18 新口径 | 依据 |
|---|---|---|---|
| 决策 3.3 标题与替代表（r17 `:1227`–`:1235`，含 `:1227` 标题、`:1229`「三条均」与 `:1234` S3.6 行） | 列入 `identity-auth-S3.4`、`identity-auth-S3.6`、`article-management-S1.4` 三张卡；S3.6 行以本机进程＋模拟 scraper／receiver 替代真实平台，`:1229` 称「三条均不得宣称真实部署通过」 | 3.3 只有两张卡：`identity-auth-S3.4` 与 `article-management-S1.4`；`:1227` 标题与 `:1229` 的「三条」按两张卡读，`:1234` 整行不再适用；S3.6 移出模拟替代，维持 blocked／未派发、真实条件未满足、未执行 | F001；Lead 裁定 (A) |
| 3.1 首条（r17 `:1214`） | 「模拟见证／备份卷**替代**真实独立故障域」 | 测试范围内**不取得**独立故障域证据，登记为未验证／unknown；该替身不满足 ADR-0030 `:953(c)` 判据，不得据其判定 `(c)` 任何一项通过 | F003 |
| `waived-for-test-scope` 条（r17 `:1216`） | 只覆盖 S4.4 模拟验证范围，不使 S4.5／S4.6 变为已实施 | 范围豁免（test scope only）：不改变 `tasks.md:649` 的 requires 文本、不改变任何生产 requires、不使 S4.5/S4.6 变为已实施、不构成安全豁免 | F002；安全侧第 1 项 |
| S4.6 豁免归属（r17 `:1216`） | 由本 ADR 作为范围豁免承接 | 本 ADR 对 key retirement 无决策权限；S4.6 测试范围豁免只以 ADR-0028 约束 2 为唯一依据、只为解除 S4.4 模拟前置阻塞，不改变 ADR-0019/ADR-0030 关于 key retirement 的任何结论，不构成 key retirement 的任何放宽 | 安全侧第 2 项 |
| 依据清单（r17 `:1210`、`:1154`） | ADR-0028 约束 1–4、6–7 | 补约束 5（`:851`）与约束 8（`:854`）；约束 8 保留 ADR-0024／0025 原范围，本修订替代只落在约束 2 真实环境层清单内，不触及 ADR-0025 保留的 S3.6 | F006；安全侧第 6 项 |
| S2.2 建议值（r17 `:1222`） | 转述为「单主机 WSL2 + PostgreSQL 18.6（仅回环）；2 vCPU／4 GB／20 GB；…」 | 逐字复制 `preflight-record.md` §9 两项：「建议值：单主机 WSL2 + PG 18.6，仅回环，模拟等级」与「建议值：2 vCPU / 4 GB / 20 GB，100 并发会话，p95 ≤ 500 ms，错误率 < 0.1%」 | F007；用户 2026-11-03 补充要求 |
| 决策 1「不实施」（r17 `:1170`–`:1172`） | 只写六卡不实施，未声明与 ADR-0028 约束 1 的范围关系 | 不实施 ≠ 豁免验证：六卡仍在 ADR-0028 约束 1（`:844`）覆盖内，未来执行仍受分级验证、不得伪造 PASS 与关闭残余风险登记约束；不实施只指当前不派发、不认领、不实施，不是范围豁免 | F004 |
| B2「以不实施收敛」（r17 `:1163`、`:1177`） | 「以『不实施』收敛该冲突」 | 冲突作为已知事实保留、**未被消除**；本轮只选择不实施并继续跟踪，B2 仍开放 | F008 |
| B4 事实（r17 `:1165`） | 「`anchor-env-r01/README.md` 不存在」 | 事实更正：该文件存在（路径、5,941 B、SHA256 `A081AA9B300655B9FD94C55CE74DC0104928659D8E8EC507786CDBA212AD74B1`）；其 §4 记环境不满足、§5 记域外见证仅为本机 JSON 桩，故存在但不构成本 ADR 的登记值 | F005 |
| B1 锚点（r17 `:1162`） | `tasks.md:3177` 与 `:3154`／`:3176(b)` 并列为直接依据 | 直接依据收窄为 `tasks.md:3154`／`:3176(b)`；`:3177` 仅以关闭条件 (2)(3) 佐证 | F009 第 1 项 |
| B2 锚点（r17 `:1163`） | `design.md:3612`–`:3616` 整体概括为「首期 closed／不提供」 | 收窄为 `design.md:3612`–`:3614`、`:3616`、`:3910`；`:3615` 是 `articleAccount` 行，不属于「首期不提供」 | F009 第 2 项 |
| 作者证据锚点（`evidence-b13j-d09/report.md:49`） | 把 `framework.md:1168` 标为「引用记录／r16 追溯行」 | 更正为 ADR-0030「引用记录」小节（r17 基线 `:1265`–`:1270`，r18 插入后实测 `:1347`–`:1352`）与 r16 追溯行（r17 基线 `:1285`，r18 插入后实测 `:1367`）；`:1168` 实为 B7 表格行 | F009 第 3 项 |
| r16 接受延续（r17 `:1156(a)`） | 只登记「是否出 r17」的文本冲突 | 补记：`preflight-record.md:22` 的「r16 字节绑定、再改即失效」规则继续有效；r16 的架构 ACCEPT 与安全 PASS WITH CONDITIONS 不适用于 r17／r18 字节；r17／r18 各自须重新取得两路独立接受，不继承任何既有接受 | 安全侧第 3 项 |
| B7 状态（r17 `:1168`、`:1206`） | 「待 Lead 裁定」 | Lead 已裁定：seam 保留，语义限定为「合成/测试注入」；生产 `main.go:634` 继续不传四端口，生产端口唯一来源仍是 registry（当前为空 → 四端口一并 closed） | B7 Lead 裁定 |
| r17 门禁边界句（r17 `:1245`） | 「只解除 `S4.4` 对 `S4.5`／`S4.6` 的派发阻塞与 `S2.2` 对 `S4.4` 模拟结果的承接」 | 该句只表示测试范围内的窄豁免；**不改变 `tasks.md:649` 的 requires 文本**，不得读作 requires 已满足；任何引用必须同时引用 `waived-for-test-scope` 定义句（`:1280`）与本行；`tasks.md` 侧不加注记，requires 原文即权威（本 ADR 不改 `tasks.md`，故不存在 ADR 与任务卡的双权威） | `B13J-D09-ARCH-R02-F011` |

**F001 — S3.6 不纳入本 ADR 的模拟替代范围**

ADR-0025（`framework.md:720`／`:726`／`:746`／`:749`）只豁免 `identity-auth-S3.3` 与 `S3.5`，`identity-auth-S3.6` 的真实部署、采集、送达、演练与关闭条件保持真实验证。ADR-0028 约束 8（`:854`）要求扩大模拟范围须另行取得用户决定并更新 ADR 与消费者。因此本 ADR 不扩大至 S3.6：S3.6 维持 blocked／未派发，其真实条件未满足、未执行，不得以模拟 scraper／receiver 结果作为关闭证据。如需扩大，须由用户另行明确决定并由本 ADR 的新修订登记，同时同步 `identity-auth/test-plan.md`、`docs/runbooks/identity-observability.md` 与 `identity-auth/design.md` 的范围声明。决策 3.3 的模拟替代表因此只有两张卡：`identity-auth-S3.4`（`tasks.md:473`）与 `article-management-S1.4`（`tasks.md:1209`）；S3.6 卡（`tasks.md:521`）不在本 ADR 的替代范围。

**F002＋F003＋安全侧第 1／2 项 — S4.4 的测试范围口径**

`waived-for-test-scope` 是**范围豁免（test scope only）**：不改变 `tasks.md:649` 的 requires 文本、不改变任何生产 requires、不使 S4.5/S4.6 变为已实施、不构成安全豁免。决策 3 的「本修订唯一的解除内容」表述由「本修订唯一的**测试范围窄豁免**」取代。该记法只表示在测试范围内执行 S4.4 的模拟验证时，S4.5/S4.6 未实施不单独构成 S4.4 模拟结果的阻断；它不表示 S4.5/S4.6 已完成、已登记或已取得证据，任何下游卡、报告、评审或 E2E 不得把该标签读作「requires 已满足」，引用时必须同时引用本句。`tasks.md` 侧不加注记，requires 原文即权威；本 ADR 不改 `tasks.md`，故不存在 ADR 与任务卡的双权威。

本 ADR 在 key retirement 上没有决策权限（`:922` 的继承关系；以及 ADR-0030 约束 8「不适用场景」，该条 r17 基线为 `:1261`、r18 插入后实测为 `:1343`）。S4.6 的测试范围豁免只以 ADR-0028 约束 2 为唯一依据，只为解除 S4.4 的模拟前置阻塞；它不改变 ADR-0019/ADR-0030 关于 key retirement 的任何结论，也不构成 key retirement 的任何放宽。

S4.4 的模拟替身（确定性端口＋本机回环 PostgreSQL 18.6＋模拟见证／备份卷）只用于验证 fail-closed 行为与状态机分支。该替身**不替代、不证明**独立故障域：在测试范围内不取得独立故障域证据，登记为未验证／unknown，不满足 ADR-0030 `:953(c)` 的独立故障域判据，不得据此判定 `(c)` 任何一项通过。

**F004 — 决策 1「不实施」与 ADR-0028 约束 1 的关系**

决策 1 的六张卡仍受 ADR-0028 约束 1（`framework.md:844`）覆盖；凡未来执行，仍受分级验证、不得伪造 PASS、任务关闭须登记未验证边界与残余风险等约束。「不实施」只指当前不派发、不认领、不实施，不是范围豁免，也不把任何卡移出 ADR-0028；不实施 ≠ 豁免验证，属于该范围内的处置选择，二者实质不冲突。

**F005 — B4 事实更正**

r17 B4 行「`anchor-env-r01/README.md` 不存在」是事实错误。该文件实际存在：`.spec/wip/recovery-runtime/evidence-b13j-d05/anchor-env-r01/README.md`，5,941 B，SHA256 `A081AA9B300655B9FD94C55CE74DC0104928659D8E8EC507786CDBA212AD74B1`。其 §1、§4「环境不满足项」与 §5「域外见证桩」分别被 `preflight-record.md` §2 状态矩阵的 (i)、(iv)、(v) 行引用（(iv) 行同时引 §1）。该文件存在但不构成本 ADR 的登记值：它不提供 witness 协议版本、请求／响应格式、鉴权 wire 或 `(w,r,token)` 存储 API；r17 `:1203` 的「不构成登记值」结论继续成立，该 README 是环境证据，不是本 ADR 的登记载体。S423 输入报告的 errata 由该报告 owner 或 Lead 另行登记，本 ADR 不改输入证据。

**F006 — 依据清单补全**

决策 3 的适用依据为 ADR-0028 约束 1（`:844`）、约束 2（`:845`–`:847`）、约束 3（`:849`）、约束 4（`:850`）、约束 5（`:851`）、约束 6（`:852`）、约束 7（`:853`）与约束 8（`:854`）。约束 8 保留 ADR-0024／0025 的原范围；本修订的替代只落在约束 2 的真实环境层清单内，不触及 ADR-0025 保留的 S3.6。

**F007 — S2.2 建议值逐字引用**

S2.2 的目标部署环境与容量预算逐字引用 `preflight-record.md` §9：

- 目标部署环境：「建议值：单主机 WSL2 + PG 18.6，仅回环，模拟等级」
- 容量预算：「建议值：2 vCPU / 4 GB / 20 GB，100 并发会话，p95 ≤ 500 ms，错误率 < 0.1%」

上述两项是模拟基线、非生产 SLO，不构成容量批准、生产预算、部署目标或 production readiness。分隔符与产品名按 §9 原文保留，不改写为全角 `／` 或 `；`。

**F008 — B2 冲突保留**

B2 的冲突作为已知事实保留在 `tasks.md` 与 design 原文中，**未被消除**；本轮只选择「不实施」并继续跟踪，B2 仍开放。不得把「不实施」读成冲突已解决、已关闭或已收敛。

**F009 — 锚点精度更正**

按 2026-11-03 实测：B1 行的直接依据为 `tasks.md:3154`／`:3176(b)`，`:3177` 仅以关闭条件 (2)(3) 佐证；B2 行的 design 依据为 `design.md:3612`–`:3614`、`:3616`、`:3910`（`:3615` 为 `articleAccount` 行，不属于「首期不提供」）；作者证据 `report.md:49` 把 `framework.md:1168` 标为「引用记录／r16 追溯行」，而 `:1168` 实为 B7 表格行，ADR-0030「引用记录」小节在 r17 基线 `:1265`–`:1270`、r18 插入后实测 `:1347`–`:1352`，r16 追溯行在 r17 基线 `:1285`、r18 插入后实测 `:1367`。上述更正只改引用锚点，不改变任何实质结论；本小节指向 `:1246` 及以前的锚点在 r18 插入后未漂移，指向其后的一律按 r18 插入后实测值书写。

**安全侧第 3 项 — r16 接受不延续**

`preflight-record.md:22` 的「r16 字节绑定、再改即失效」规则继续有效，不因本轮写作授权废止。r16 的架构 ACCEPT 与安全 PASS WITH CONDITIONS 只绑定 r16 字节，**不自动延续**至 r17／r18；r17／r18 各自须重新取得架构与安全两路独立接受，任何后续接受须对新字节另行登记。本 r18 未取得两路接受，不构成任何接受。

**B7 — seam 保留（Lead 裁定）**

`recoveryRootInputs` 四端口覆盖 seam 保留，语义限定为「合成/测试注入」。生产构造点 `cmd/han-server/main.go:634` 继续不传四端口，生产端口唯一来源仍是 registry；当前 registry 为空，四端口一并落 `recoveryClosedPort`。本卡（S4.23）不修改该 seam，也不修改 `recovery_enter_install_integration_test.go`／`recovery_production_chain_integration_test.go`，不因 B7 改动 `recovery_runtime.go` 的覆盖逻辑。若未来恢复生产 adapter 链，须由对应实施卡显式登记「该 seam 不得被生产路径使用」的可核断言并单独评审；本裁定不预先批准该变更，也不改变 S4.23 的 BLOCKED 状态。

**其余安全侧 LOW（F004／F005／F007）处置**

安全侧 F004（第三处历史冲突未登记、B4 行「waived」缺测试范围限定）：B4 的「waived」措辞由本小节取代为「在测试范围内不取得该证据、登记为未验证／unknown，生产仍 unknown／未满足」；`preflight-record.md` §9 的整句引文「(i)–(v) 在**测试口径**下均有登记值，(iii) 已具名两人，(iv)(v) 由 owner 豁免；不再以「结构性不可满足」维持四卡阻塞」只登记为**测试口径事实**，它不满足 ADR-0030 约束 9(b) 的启动前置，不改变 `tasks.md:664` 的阻塞口径——该卡仍以启动前置 (i)–(v) 整组未满足为由保持阻塞、不得启动——不构成任何安全豁免。安全侧 F005（「不改既有安全判据」覆盖不足）：本小节不动 ADR-0030 约束 1–10 与 r05–r16 各补充段；决策 3 的模拟替代不适用于 TLS verify-full、双 fd（3／4）、凭据保管分离、ACL 最小权限与见证单调性判据，这些判据在测试范围内同样不获得放宽。安全侧 F007（作者证据缺改前副本）：本轮作者证据 `.spec/wip/recovery-runtime/evidence-b13j-d09-r18/report.md` 固定改前 `framework.md` 的字节数、段数、CRLF 数与 SHA256，并给出 ADR-0025／0028／0031／0032、r13–r17 补充段与 r17 追溯行的改前段级哈希，供独立评审逐字节复核。

**门禁边界（r18 重申）**

生产 `enter`/`install`/`resume`/`unlock` 仍全部 closed。本小节不解除 ADR-0030 约束 7／9／10、ADR-0031 解除条件与 ADR-0032 解除顺序；B1／B3／B4／B5 仍为「产能外 unknown」；不得据本小节宣称生产就绪、真实独立故障域成立、ACL 已验证或真实环境 PASS。`S4.23`／`S4.24a`／`S4.24b`／`S4.22` 不得因本小节启动。本小节不改 `decisions/README.md`；其现行摘要仍为 r17 且含 S3.6，索引收口由 Lead 另行登记。本小节是作者文档处置，不构成评审 PASS、人的接纳或生产批准。

#### r19 补充（D09 架构侧 r02 复审 F011–F014 定点更正；不新增生产结论）

本小节按用户 2026-11-03 的 D09 文档型决策卡 r19 指令落定，只做四项定点更正，不新增、不撤销任何决策，不改变 r17／r18 已登记的实质结论。触发与输入：架构侧复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-arch-r02/report.md`（26,857 B / `CEBF0C0F13A06DBD698F680EF3A48D438EA1C2D591C5CB93FF823175959AEE0A`，CHANGES REQUIRED，F010–F014）与安全侧复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-sec-r02/report.md`（20,952 B / `907F62DD2C1786AE30A99E6537AF27B60514566589FD6A9E6A0866130922859A`，PASS WITH CONDITIONS，条件为 `B13J-D09-SEC-R02-F001` 索引收口）。架构侧 F010 与安全侧 `B13J-D09-SEC-R02-F001` 是两件事，两者均已收口；索引收口由 Lead 处置 `decisions/README.md`（`.spec/wip/recovery-runtime/evidence-b13j-d09/lead-index-sync-f010.md`），不在本 ADR 写集内。

**F012 — 物理更正 r17 决策 3.3 段（Lead 授权）**

按 Lead 2026-11-03 对 F001 裁定第 1、3 条的明确授权，本次物理更正 r17 决策 3.3 段：标题 `:1227` 只保留 `identity-auth-S3.4`（`:473`）与 `article-management-S1.4`（`:1209`）两张卡，并去掉 S3.6 移出后不再适用的「可观测平台」措辞；删除 `:1234` 的 S3.6 模拟替代表行；`:1229` 的「三条均」改为「两条均」；3.3 段末以引用块标注该处为 r19 定点更正。这是本 ADR 首次回写 r17 原文，r18 `:1251` 的「r17 原文与 r01–r16 追溯行逐字保留」在 3.3 段不再适用；r17 其余原文仍逐字保留。r18 取代表 `:1257` 中「标题与『三条』按两张卡读」「整行不再适用」的描述，自本 r19 起由物理更正取代。

**F011 — 取代表点名 r17 门禁边界句**

在 r18「r17 条款取代表」末尾新增一行，点名 r17 `:1245`：该句只表示测试范围内的窄豁免，不改变 `tasks.md:649` 的 requires 文本，不得读作 requires 已满足，任何引用必须同时引用 `waived-for-test-scope` 定义句与该行。`tasks.md` 侧不加注记，requires 原文即权威；本 ADR 不改 `tasks.md`，故不存在 ADR 与任务卡的双权威。

**F013 — 两处引文改为逐字或明确转述**

r18 `:1323` 对 `tasks.md:664` 的引号引文改为不带引号的转述；对 `preflight-record.md` §9 `:122` 的引文改为逐字引用，并补回被省略的 (iii) 子句（r19 时点该引文仍止于『(iv)(v) 由 owner 豁免』的句首片段，句尾由 r20 补全为整句逐字引用，见 r20 小节）。

**F014 — 更正 r18 `:1290` 的跨节锚点**

r18 `:1290` 原句把 §1／§4／§5 写成 `preflight-record.md` 的节号，实测该引用位于 `preflight-record.md` §2 状态矩阵。本次改为：其 §1、§4「环境不满足项」与 §5「域外见证桩」分别被 `preflight-record.md` §2 的 (i)、(iv)、(v) 行引用（(iv) 行同时引 §1）。r02 F014 的建议句称「§5 域外见证桩为同目录证据，未被 preflight 直接引用」，与实测 `preflight-record.md:35`（(v) 行「现有证据」列明写 `anchor-env-r01/README.md` §5）不符，故本 ADR 按实测事实书写，不照抄该建议句。

**门禁边界（r19 重申）**

生产 `enter`/`install`/`resume`/`unlock` 仍全部 closed。本小节不解除 ADR-0030 约束 7／9／10、ADR-0031 解除条件与 ADR-0032 解除顺序；B1／B3／B4／B5 仍为「产能外 unknown」；`S4.23`／`S4.24a`／`S4.24b`／`S4.22` 不得因本小节启动。本 r19 未取得架构与安全两路接受，不构成评审 PASS、人的接纳或生产批准。r18 `:1327` 记录的『README 仍为 r17 且含 S3.6』是 r18 时点状态，已由 Lead 于 r19 之后完成索引收口（`.spec/wip/recovery-runtime/evidence-b13j-d09/lead-index-sync-f010.md`，README 现为 22,942 B / `53FD5846C2FE2BEEBB9B8BDCCF1E5239C469B79588E3965B68196550C4BEB328`）。

#### r20 补充（r03 两路复审残留收口：tasks.md 注记口径统一、引文标注、README 时点表述、编号更正与被删行留痕；不新增决策）

本小节按用户 2026-11-03 的 D09 文档型决策卡 r20 指令落定，只做五项文本收口，不新增、不撤销任何决策，不改变 r17／r18／r19 已登记的实质结论。触发与输入：架构侧 r03 复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-arch-r03/report.md`（22,404 B / `763459D8D8F1AC179EF9438E6D039B93A10F5E2FDE6EEDE39AF049136C8E1B8C`，PASS WITH CONDITIONS，条件项 `B13J-D09-ARCH-R03-F015` MEDIUM 与 `B13J-D09-ARCH-R03-F016` LOW）与安全侧 r03 复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-sec-r03/report.md`（24,209 B / `951921836EF102A0E8F89213CD613385F723C811D6E5C62466894FB09B9CD85B`，ACCEPT，`B13J-D09-SEC-R03-F001`／`F002`／`F003` 均为 LOW）。两路对 r19 字节的结论不自动延伸至本 r20 字节；本 r20 未取得架构与安全两路接受。

**B13J-D09-ARCH-R03-F015（MEDIUM）— `tasks.md` 注记口径统一**

本项消除 `tasks.md` 注记的三处相反口径：`waived-for-test-scope` 专段（`:1280`）末句由「`tasks.md` 侧的注记由 Lead 另行登记，本 ADR 不代替」改为「`tasks.md` 侧不加注记，requires 原文即权威；本 ADR 不改 `tasks.md`，故不存在 ADR 与任务卡的双权威」。全文口径据此统一为：不在 `tasks.md` 加注记；`tasks.md:649` 的 requires 原文即权威；`waived-for-test-scope` 是范围豁免而非 requires 变更；本 ADR 不改 `tasks.md`。取代表行（`:1272`）与 r19 说明（`:1341`）的既有表述与本句一致，未改其结论。`tasks.md` 本轮未改（912,450 B / `E034155076F77F8D1E8288D9BB4014554A5019882051A03A305A0ACA6AD71EFB`）。

**B13J-D09-ARCH-R03-F016＋B13J-D09-SEC-R03-F001（LOW）— `preflight-record.md:122` 引文补全**

`:1325` 对 `preflight-record.md` §9 `:122` 的引文补全为整句逐字引用：「(i)–(v) 在**测试口径**下均有登记值，(iii) 已具名两人，(iv)(v) 由 owner 豁免；不再以「结构性不可满足」维持四卡阻塞」。内层引号按原文保留为「」。r19 说明（`:1345`）同步注明 r19 时点仍为句首片段、句尾由本 r20 补全，「逐字引用」的表述与实际一致。该句仍只登记为**测试口径事实**，不满足 ADR-0030 约束 9(b) 的启动前置，不构成任何安全豁免，`tasks.md:664` 的阻塞口径不变。

**B13J-D09-SEC-R03-F002(a)（LOW）— README 描述时点化**

`:1353` 对 `decisions/README.md` 的描述改为时点表述：r18 `:1327` 记录的『README 仍为 r17 且含 S3.6』是 r18 时点状态，已由 Lead 于 r19 之后完成索引收口（`.spec/wip/recovery-runtime/evidence-b13j-d09/lead-index-sync-f010.md`，README 现为 22,942 B / `53FD5846C2FE2BEEBB9B8BDCCF1E5239C469B79588E3965B68196550C4BEB328`）。r18 `:1329` 的「现行摘要仍为 r17」同属 r18 时点陈述，其文本未改。本 ADR 未改 `decisions/README.md`，门禁结论不变。

**B13J-D09-SEC-R03-F002(b)（LOW）— 安全侧条件编号更正**

`:1333` 把安全侧 r02 条件项由误记的「F010」更正为 `B13J-D09-SEC-R02-F001`，并写明架构侧 F010 与安全侧 `B13J-D09-SEC-R02-F001` 是两件事、两者均已收口（索引收口由 Lead 处置 `decisions/README.md`，不在本 ADR 写集内）。架构侧 F010 的历史记录未改。

**B13J-D09-SEC-R03-F003（LOW）— 被删行留痕**

r19 按 Lead F001 裁定第 1、3 条删除的 r17 旧 `:1234` S3.6 模拟替代表行原文（取自 r19 改前副本 `%TEMP%\framework-r18-backup.md`，370,555 B / `4EC6C2AF086BEC4ABB59E5DE180CC0323FA13EBD40D532098B5B271FDE1A148C`）逐字保留如下：

```
| `identity-auth-S3.6` | 以本机进程、模拟 Prometheus-compatible scraper、模拟 receiver／路由端点复现 I-OBS-DEP-01～06 的数据流、告警触发与退出恢复 | 真实日志平台、真实 Prometheus／Grafana／Alertmanager、获准 receiver 账号、真实角色升级链与真实演练窗口 | 无真实平台验证、无真实角色排班与演练；S3.5 已豁免的平台工具校验不被追认为 PASS；不得据此宣称 observability 生产就绪 |
```

该行于 r19 按 Lead F001 裁定删除，此处保留原文仅为可追溯，不恢复其效力。S3.6 仍按 r17 决策 3.3、`:1276` 与 ADR-0025／ADR-0028 约束 8 维持 blocked／未派发、不纳入本 ADR 的模拟替代范围。

**门禁边界（r20 重申）**

生产 `enter`/`install`/`resume`/`unlock` 仍全部 closed。本小节不解除 ADR-0030 约束 7／9／10、ADR-0031 解除条件与 ADR-0032 解除顺序；B1／B3／B4／B5 仍为「产能外 unknown」；`S4.23`／`S4.24a`／`S4.24b`／`S4.22` 不得因本小节启动。本 r20 未取得架构与安全两路接受，不构成评审 PASS、人的接纳或生产批准；r03 架构侧为 PASS WITH CONDITIONS、安全侧为 ACCEPT，均只覆盖 r19 字节，架构侧条件项 F015 的消除不自动构成 r20 字节的 ACCEPT。

### 约束

1. **不改变 ADR-0019**：独立故障域、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒、CAS 协议、key 轮换/退役、三方对账与 fail-closed 失败分类继续有效。本 ADR 只处理其留给 S4.5 的产品选择、配置键与启动装配边界。
2. **禁止自动 fallback**：不得以 file、SQL、业务 PostgreSQL 表、内存、fake 或任何“环境探测后自动选择”的路径填补生产 anchor；本项所称“SQL”仅指以业务数据库或通用 SQL 表作为 anchor 的自动 fallback 或隐式来源，显式登记、显式配置、固定指向独立故障域实例的 adapter 不属于被禁路径；不得把测试替身注册为生产 provider；不得以空 registry 之外的方式隐式构造 adapter。
3. **配置键语义**：生产只认 `HAN_RECOVERY_ANCHOR_ADAPTER` 与 `HAN_RECOVERY_ANCHOR_CONFIG_PATH`。两者必须同时显式提供；adapter ID 必须已登记，配置路径必须是受保护的绝对路径且可被 owner-only 边界读取。二者无代码默认值、互不推断，root 不得按环境、hostname、数据库、文件存在性或其他配置推导 adapter。
4. **未配置即 closed**：任一配置缺失、adapter 未登记、配置不可读/不可解释、ACL 不满足或验证不通过时，不启动 anchor 相关能力，生产 recovery `enter`/`install` 保持 closed；不得降级为测试替身或只读模式。
5. **S4.5 维持未交付**：本 ADR 不登记、不预占任何实现任务，也不把 S4.5 标记为完成。生产 provider 的实现、配置、启动装配与真实独立故障域验证仍属 S4.5；在其完成并由正式规划登记前，生产恢复链不得宣称可用。
6. **不声称产品或环境验证通过**：本 ADR 不证明任何产品、云服务、数据库实例、Vault/KMS 或对象存储已通过 CAS 冲突、stale revision、missing/rollback、pending/committed、崩溃恢复或权限失败验证；也不证明生产装配、Case18、S8.2 或恢复 E2E 通过。
7. **解除条件（生产解除级门禁）**：只有同时满足以下条件，才可通过新的 ADR 修订或新 ADR 解除“首期不提供”和生产 closed。本项只约束生产 closed 的解除与生产可用声明，不约束决策卡、实施/验证卡与生产装配/解除卡的登记；三类卡的登记入口见约束 9：
   - 选定具体产品与部署形态，并能证明其与业务 PostgreSQL 处于独立故障域；
   - 凭据来源、最小权限/ACL、部署拓扑、备份/恢复/轮换与运维责任经批准且可核；
   - 配置键与启动装配合同明确，仍无隐式默认或自动 fallback；
   - 在真实独立故障域完成 CAS 冲突、stale revision、missing/rollback、pending/committed、崩溃恢复与权限失败验证，并保留原始证据；
   - 通过架构与安全独立评审，完成 ADR 修订/新增决策，并由正式 Task 规划登记实现与验证范围。
8. **不适用场景**：本 ADR 不适用于 S4.2 的 port 与测试替身交付（已由 ADR-0019 定）、key retirement capability（ADR-0019 另定）、restore 工具/批准者/宿主隔离、部署独占、ArticleAccountPort 或 recovery secret candidate；这些分别由其他设计/决策承接。Windows 生产仍 unsupported，本 ADR 不提供 Windows 例外。
9. **后续阶段前置节点（三级登记次序，无环）**：按卡片类别区分三个入口，避免把“决策修订”与“约束 7 全项解除”绑成同一前置而形成登记死锁。(a) **决策卡（文档型）**：只产出 ADR 修订/新 ADR 与结论，不含生产实现代码、不改 `cmd/` 或 `internal/`；此类卡不受本门禁限制，可在任何时点登记。(b) **实施/验证卡（含尚未登记为正式 Task 的 S4.23 一类候选编号）**：登记入口是“本 ADR 的决策修订（给出具体产品、凭据来源、独立故障域论证、部署与 ACL 结论）已由架构与安全独立评审接受”，而不是“约束 7 全项解除”；该卡用于产出真实证据。该接受只解除本 ADR 的实施/验证卡登记限制，不等于人的决策接纳，也不产生任何人的批准；生产可用仍须 (c) 与约束 7「取得人的明确生产批准」项。(c) **生产装配/解除卡（含候选 S4.22 一类）**：本类卡的登记不因约束 7 未全项满足而被禁止，约束 7 的未满足项只决定其能否解除 closed 与作生产可用声明；其解除仍由约束 7 全项（含约束 7 的真实独立故障域验证项与正式 Task 登记项）加人的明确生产批准共同约束；只有它才能解除“首期不提供”与生产 closed，也只有它能作生产可用声明。次序为：决策卡 → 决策修订被独立评审接受 → 实施/验证卡 → 真实证据 → 生产装配/解除卡 → 生产解除；每级只依赖其直接前驱，无回边。本 ADR 的 r03 只改门禁分级与候选编号口径，不改变“首期不提供”结论、候选取舍、约束 1–6 与解除条件实质。本 ADR 的 r04 只补前述「不等于人的接纳」限定、解除/登记作用面与符号锚，不改变上述次序与实质。
10. **anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源**：生产 `exclusive` 的唯一来源是已登记 anchor adapter bundle 的四端口集合（`Commitments`/`Sources`/`Anchor`/`Exclusive`，缺一即 `ErrAdmissionBinding`）。首期 bundle 不存在，是因为 production root 以空集合构造 registry（`newRecoveryAnchorRegistry(nil)`），未登记解析结果四项一起落 `recoveryClosedPort`，且本 ADR 未提供生产 CAS provider。后续只能按与约束 9 同构的次序获得：本 ADR 决策修订（生产 CAS provider、独立故障域、部署/ACL 结论）被架构与安全独立评审接受 → 实施/验证卡登记并使同一 bundle 四端口齐备、作为唯一登记来源 → 真实证据 → 生产解除，`exclusive` 才可能离开 closed；不得以 file、SQL、fake 或环境探测自动 fallback，不得用合成/测试注入（`recoveryRootInputs.exclusive` 仅限合成/测试）或 registry 之外的第二来源替代。

### 引用记录

- 2026-11-01 `identity-auth` 设计修订 `identity-auth-design-ir01-r58` 采用本决定：开发视图 `.add/project/knowledge/design-workspace/identity-auth/design.md:3577` 固定“首期不提供生产 provider、仅保留 port 与测试替身、root 不得构造隐式 adapter、生产 enter/install 继续 closed”；配置表 `:3892` 固定两个配置键必须显式、无默认、未配置即不启动 anchor 能力并保持 closed；决策追溯 `:83` 登记本 ADR。设计仍为 draft，人的精确接纳 unknown，不声明实现或验证。
- 硬边界：[framework.md（ADR-0019）](framework.md#adr-0019)，`framework.md:418` 起；本 ADR 不重写其任何已锁定决定。
- 缺口来源：[gap-map.md](../../../../.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md) 的 A08、§2.4 `pending-B13J-D02` 与 U3；该文件明确 anchor 产品/凭据/独立故障域 unknown，要求先出 ADR 或明确首期不提供。
- 决策授权：用户 2026-11-01 B13J D02 指令；产出者为 Arch-Designer（Agent），不是人的签字、独立评审或生产批准。
- 2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）（触发：非作者规划评审 `B13J-PLAN-F004`/`F005`，报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r01/report.md`）：新增约束 9（后续阶段实施门禁）与约束 10（anchor adapter bundle 的 `Exclusive` 端口与 CAS 头锚点同源），说明首期 bundle 不存在的原因与后续唯一可核顺序；决定结论、候选取舍、约束 1–6 与解除条件实质未变；未新增实现任务，S4.22/S4.23/S4.24 仍为未登记候选，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r03 修订（触发：非作者规划评审 `B13J-PLAN-R02-F001`/`F002`/`F005`，报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r02/report.md`，18538 B / `0462EC08AD8BCFECC24AB0BDDC8A9627700DF8D72DEE0441781BF6CFD930C155`）：把约束 9 改为三级无环登记次序（决策卡 / 实施·验证卡 / 生产装配·解除卡），把约束 7 明确为生产解除级门禁，并按同构次序改写约束 10 的后续路径；只改门禁分级与候选编号口径，不改变“首期不提供”结论、候选取舍、约束 1–6 与解除条件实质；未新增实现任务，候选编号仍为未登记候选，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r04 收口修订（触发：非作者规划评审 r03 的 `B13J-PLAN-R03-F003`/`F005`/`F007`，报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/planning-review-r03/report.md`，27104 B / `3FC6A611EE3ACCBCBDBE33AAD77506B7A0AED7158691CF842229669AF201EC19`）：约束 7 的登记豁免枚举补入生产装配/解除卡；约束 9(b) 补“决策修订被独立评审接受只解除登记限制、不等于人的决策接纳、不产生任何人的批准”；约束 9(c) 把内嵌数值行锚改为符号锚（约束 7 的真实独立故障域验证项与正式 Task 登记项），并写明本类卡的登记不因约束 7 未全项满足而受阻、约束 7 只约束其解除与生产可用声明。只改门禁措辞与作用面，不改变“首期不提供”结论、候选取舍、约束 1–6 与解除条件实质；未新增实现任务，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r05 修订（触发：B13J `identity-auth-D05` 文档型决策卡，卡面 `.add/project/knowledge/tasks/tasks.md:3039`；确认来源为用户 2026-11-02「对待确认或决策项采用建议值并继续本项目」的 standing 授权）：在保持「首期不提供」、约束 1–6/8–10 与解除条件实质不变的前提下，新增「r05 四项结论」子段，以 Lead 采纳的建议值给出 (a) 产品与部署形态、(b) 凭据来源与最小权限/ACL、(c) 独立故障域可核判据、(d) 部署拓扑与运维责任，并写明与约束 2 的关系及对约束 9(b) 门禁「有可评审对象但未满足」的判定。输入为改前本文件 `39BD3A797B0FA2F5CDF944ACCF4129BB0B7F7A23FABC743B7E5CCC9F7B9BD704`（182488 B / 1141 行）与当前 `identity-auth/design.md:3577`（开发视图首期不提供、root 不得构造隐式 adapter）、`:3614`（11 端口清单 anchor 行）、`:3910`（两个配置键语义）——设计行号为本轮实测值。未新增实现任务；四类候选在 r05 写作时点尚未登记，随后登记轮已将其登记为正式任务卡（`tasks.md:3091` 候选 S4.23、`:3124` 候选 S4.24a、`:3157` 候选 S4.24b、`:3197` 候选 S4.22），当前状态为「已登记、未启动」，本修订不改动这些卡片、也不宣称其已启动；生产继续 closed；本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r06 修订（触发：D05 两路非作者独立评审——架构侧 `review-arch-r01` 的 `B13J-D05-R01-F001`–`F007`（ACCEPT WITH CONDITIONS）与安全侧 `review-sec-r01` 的 `B13J-D05-SEC-R01-F001`–`F008`（REJECT），报告 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r01/report.md`（18631 B / `59BA61D895DF6442D2ED20A42DBA02410B9B4D27BF85B22211C934EC9B18973B`）与 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r01/report.md`（28741 B / `FA97B5E17CE0A5B799F3C53C49E9DA0D36D494B8F9F0E9C8B5A6AFB35F49C43E`））：把 (a)–(d) 的真实部署证据从候选 S4.23 的启动前置改为其退出条件（启动门禁只保留「决策修订被架构与安全独立评审接受」），新增目标形态→六项判据的推理与残余共享依赖否证条件，补充 anchor 头不可回退性（append-only 头事件序列或等价外部见证、低于已观测最大 revision 即拒绝）、TLS verify-full/不低于 1.2/禁降级、口令只经一次性 fd/stdin 且禁 environ、禁止 anchor 表进入 `migrations/` 与迁移 DSN 指向 anchor、凭据保管主体分离、版本清单 owner 与 EOM 门禁，并把 `Exclusive` 唯一形态固定为 ADR-0032 的 POSIX 独占文件锁；约束 2 的「SQL」解释口径固定在 r05 子段，约束正文未改。评审 A-F007（`framework.md:924` 引用 `identity-auth/design.md:3892`）经 Lead 裁定不采纳：该段为 r04 历史表述、本项目口径为不回写历史，当前同义行在 `identity-auth/design.md:3910`，映射见作者报告 §2。未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0019/0031/0032，也未新增实现任务。输入为改前本文件 `69382A4A61975E2A354822F1DA6F527AF56A2145D0DE9C0029258C4C2EECD734`（188929 B / 1157 行）；本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r07 修订（触发：D05 两路非作者复评——架构侧 `review-arch-r02` 的 `B13J-D05-R02-F001`/`F003`（CHANGES REQUIRED）与安全侧 `review-sec-r02` 的 `B13J-D05-SEC-R02-F001`/`F002`/`F003`/`F004`（CHANGES REQUIRED），报告 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r02/report.md`（26951 B / `8044E24243832F479746DA18DABBD7E8E72A80702D71C8EB6E566588F61966DF`）与 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r02/report.md`（21823 B / `DBEC957FDB93EA9BDD0BFD1D921D926E7B8D180ED3C7D349D38BBF83C6497FFB`））：把 (a)–(d) 统一为单一写入模型——append-only 头事件表 + `(namespace, revision)` 唯一约束 CAS + 派生 head + 角色仅 `SELECT`/`INSERT`（(b) 删除 `UPDATE`），域外单调见证升级为合取必需项（已观测最大 revision 只持久于见证，不可读、不可写或低于 head 即 closed），回退演练升级为 anchor 整实例回退并重启与新副本冷启动；把 bundle 的 `Exclusive` 改为三段式以消除与 ADR-0032 的互为前置；(d) 的脚本禁令限定到 anchor 实例；按 Lead 裁决把「登记/启动」术语口径落为解释性段落并声明效力低于约束正文。未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0019/0031/0032，未新增实现任务、未启动任何候选卡；架构 r02 的 F004/F005/F007/F008/F009 未处置、保持 OPEN。输入为改前本文件 `AC0D758A030CE828CABC90BBC012D1C8907012AF78A8A0573A104AFDFA31B5CD`（197114 B / 1179 行）；本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r08 修订（触发：D05 两路非作者第三次复评——架构侧 `review-arch-r03` 的 `B13J-D05-R03-F001`–`F013` 与安全侧 `review-sec-r03` 的 `B13J-D05-SEC-R03-F001`–`F009`（CHANGES REQUIRED），报告 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r03/report.md`（31936 B / `9F39DA5D6A53859ED21A59FB6649FD7E42BF081092441DD71065F2011CC8129F`）与 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r03/report.md`（33105 B / `7F7F73B325571EAB5B44DC2F0739C3CDA0624426CB37F520D45A217D3A215579`））：把见证语义收敛为唯一状态机（只允许先预留见证、后提交事件；见证持久记录 `(last_confirmed_head, reserved_head, reservation_token)`；删除检测滞后豁免分支；见证低于已确认 head、预留越界或 token 不匹配一律 closed 且不得自愈）；CAS 键改为 `UNIQUE(namespace, revision, kind)`，pending 携带 `pending_token`、committed/abandoned 必须引用同 revision 的 pending 行，并把 ADR-0019 的「清除未发布 pending」映射为 abandoned 事件；补见证的域外保证等级四项、见证端点与第二次一次性 fd/stdin 通道、批准记录最小字段、威胁模型边界与演练四档；把 `:958` 改为「评审门禁（一条）+ 启动前置 (i)–(v)」两级口径；删除 `:996` 的「与 ADR-0032 解除顺序一致」虚假声明，改为诚实记录跨 ADR 冲突。未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0019/0031/0032，未新增实现任务、未启动任何候选卡；`R03-F003` 的跨 ADR 部分、`R03-F004`（约束 2 正文）、`R03-F009`/`F010`/`F012` 与安全 `R03-F009` 仍未处置。输入为改前本文件 `9F3E5ADC8846074A2A91ACF7B4D25046E35E4DE30789025A2FAECAFE0232D4BD`（205072 B / 1201 行）；本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r09 修订（触发：D05 两路非作者第四次复评——架构侧 `review-arch-r04` 的 `B13J-D05-R04-F001`–`F008` 与安全侧 `review-sec-r04` 的 `B13J-D05-SEC-R04-F001`–`F009`（CHANGES REQUIRED），报告 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r04/report.md`（31922 B / `6CFE181E1FBB86AAAB21E5CDD36800D8F830083F849AD82702B5CE5A8BAE28B8`）与 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r04/report.md`（45464 B / `41BA74871890E3D60B2B14030D952D01CBF077D26A6B758682E7630118231325`））：新增终结行互斥（`(namespace, revision) WHERE kind IN ('committed','abandoned')` 部分唯一索引，禁 committed↔abandoned 互转）；放弃两态枚举，改为见证 `(w, r, token)` 最小不变量集合（连续 abandoned 合法，`w` 只允许推进到 `r`）；补 committed/abandoned 落库后的幂等完成与六分支崩溃恢复（含「旧头 + 可验证临时文件 → 完成发布」）；补 ADR-0019 字段（`journal_id`/`epoch`/`head_sequence`/`head_hash`/操作标识）与 epoch 不匹配 closed；`Exclusive` 顺序统一为诚实跨 ADR 冲突表述，删除 `:979` 的虚假对齐；`:936` A 行加时点注；批准记录字段、威胁模型（配置权威）、保证等级第 (5) 项与门禁两级同步。未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0019/0031/0032，未新增实现任务、未启动任何候选卡；约束 2 正文（Lead 已授权 r10 处理）与 ADR-0032 解除顺序（D07）仍 OPEN。输入为改前本文件 `7DB51A18F609893A13C49EA8558AC821630412DA29C69B4A102C81EA3E6A76A2`（221148 B / 1222 行）；本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r10 修订（触发：D05 两路非作者第五次复评——架构侧 `review-arch-r04` 与安全侧 `review-sec-r04` 判定约束 2「SQL」字面的下位解释不足；授权：Lead 按用户 2026-11-02 standing 授权批准 A 项，记录 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r09.md` §1「A」）：仅改 `:918` 修订串与约束 2 正文一句，在禁止路径中补「本项所称“SQL”仅指以业务数据库或通用 SQL 表作为 anchor 的自动 fallback 或隐式来源，显式登记、显式配置、固定指向独立故障域实例的 adapter 不属于被禁路径」，只缩窄「SQL」一词的禁止范围（+212 B、行数不变），未新增任何允许路径；「首期不提供」、约束 1、3–10 正文、解除条件与其它结论均未变，未登记实现任务、未解除生产 closed；产出者为 Arch-Designer（Agent），不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r11 修订（本轮一致性修订；授权：Lead 授权记录 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r10.md` §A/§B，依据用户 2026-11-02 standing 授权「对待确认或决策项采用建议值并继续本项目」）：本 ADR 段的内容变更已在前一轮落入（`:960` 新增启动前置 (i)–(v) 已登记建议值段、`:1073` 新增 r10 追溯行、改写 `:981`/`:998`/`:1021`/`:1037` 的现行态陈述与 `:1023`/`:1039` 的 OPEN 清单），但修订串仍停在 `adr-0030-r10`，构成审计缺口；本修订把修订串升为 `adr-0030-r11`、补本条追溯行，并把 `:981`/`:998`/`:1021`/`:1037` 指向 ADR-0032 的行号锚更新为改后实测值（`framework.md:1224`/`:1225`，对应 ADR-0032 r05 合并后的候选 S4.23 与候选 S4.24b 节点）；按 §B 把 `:960` 的 (i) 具体产品登记为 PostgreSQL（社区官方发行包）、建议值取 16 大版本线、自建独立实例，精确补丁版本由 (ii) 的具名 owner 受支持清单确定，该登记是建议值、不构成验证，也不替代 (iii)(iv)(v) 的具名与取证，启动前置 (i)–(v) 仍未满足。另记明：r07/r08 段追溯行的未处置清单把 `R02-F005`/`R03-F004`（同一约束 2 正文冲突）仍记为未处置，实际已由 r10 改写约束 2 正文处置，并由本 r11 完成审计收口（修订串与追溯），后续以 r11 口径为准；r07/r08 历史原文不回写。本次未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0019/0031/0032，未新增实现任务、未启动任何候选卡，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r12 修订（触发：D05 两路非作者第六轮复评——架构侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r06/report.md` 的 `B13J-D05-R06-F002`/`F004` 与安全侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r06/report.md` 的 `B13J-D05-SEC-R06-F001`/`F002`/`F006`；授权：Lead 授权记录 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r11.md` §F）：本次只改四处——(1) `:960` (i) 消除「独立主机或独立实例加独立存储卷」的弱读：「独立实例加独立存储卷」须同时满足 `:953(c)` 判据 1（不与业务 PostgreSQL 共用主机、集群与数据目录）才成立，且任何形态都不得单独作为独立故障域证据；(2) `:960` 补优先级与来源要求——(ii) 的受支持清单在本 ADR 内优先于 (i) 的版本指向、不一致以清单为准，批准记录须登记包来源校验（签名或固定仓库）与 EOL 门禁；(3) `:981`/`:998`/`:1021`/`:1037` 的 ADR-0032 现状补记为 r04 起、r05 收口（r04 作为同日前序修订保留为历史事实），并把行号锚更新为改后实测 `framework.md:1225`/`:1226`（ADR-0032 修订串 `framework.md:1172`）；(4) `:1023`/`:1039` 的 OPEN 清单按第六轮结果移出已处置项（`R05-F004`、`SEC-R05-F005`、`SEC-R05-F006`）。ADR-0030 约束 1–10 正文与「首期不提供」结论、ADR-0032 的不变量 1–9 与解除条件、POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、`O_CLOEXEC` 常驻 fd、三入口共锁、`journalPath != lockPath` 一律逐字未变（本轮未改 ADR-0032 任何字节）；本次未新增实现任务、未启动任何候选卡；启动前置 (i)–(v) 仍未满足（(iii)(iv)(v) 仍为 unknown／待登记），`identity-auth-S4.23` 一类卡不得启动，生产继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r13 修订（触发：用户 2026-11-02 在本会话明确「确认按建议执行」，并以项目 owner 身份声明本项目为测试项目、用于验证 ADD 流程完整性；与已接受的 ADR-0028（`framework.md:816`–`:854`）的模拟环境替代、AI 代理评审签收与责任人建议值口径一致）：新增「r13 补充」小节，把启动前置 (iii) 六槽责任主体、(iv) 目标环境与 (v) 见证形态由 unknown 登记为「建议值已登记（AI 代理/模拟）」，逐条写明 `:991`/`:1015` 的禁止重合校验与「形式/模拟分离、组织分离未真实成立、非真人任命」声明；登记 2026-11-02 只读实测（Windows PostgreSQL `18.6 on x86_64-windows`／端口 `5433`／无 `han` 库／遗留 3,736 个非系统 schema 与 34 组临时角色／WSL `Ubuntu` 2 Stopped／`ext4.vhdx` 1,190,133,760 B），并明确业务库与 anchor 不得共用同一集群/数据目录、该 5433 实例只能作为业务库、anchor 须在 WSL 内独立集群、同一物理机为残余风险；见证五项保证等级逐项 unknown 或模拟、保留期比较待填、`(w, r, token)` 原子条件写验证 NOT RUN。门禁表述：模拟等级可据此开放派发，真实等级仍未满足〔该句自 r14 起作废，见 `:1102`〕；不得据此宣称生产就绪、约束 8 满足、真实故障域或真实组织分离成立、S4.24a 已实施。只改 `:918` 修订行（`adr-0030-r12`→`adr-0030-r13`）并新增本小节与本条追溯；未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0019/0031/0032 任何字节，未新增实现任务、未启动任何候选卡，生产继续 closed。输入为改前本文件 `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`（297,412 B / 1,275 行）；本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r14 修订（触发：r13 两路非作者独立评审——架构侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r08/review-arch-r08.md`（28,517 B / `D89B8B65CA5A572E74BA5144E2DB908103F93BAF4F7CEB1BDAD8394DDD4B0E2A`，REJECT，`ADR-R08-01`–`ADR-R08-10`）与安全侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r08/review-sec-r08.md`（19,344 B / `81BD8CEE0441156EB3BB08BC268FBC8DCDA50872B551603C611E9C16842B0D08`，PASS WITH CONDITIONS，`ADR-S08-01`–`ADR-S08-05`）；并纳入 2026-11-02 anchor 环境实测发现的 (b) 契约缺陷）：(i) 删除 r13 补充段 D 中把模拟等级直接换算为放行派发的自授权表述，门禁状态改为「未放行」，模拟等级只允许登记与文档／证据准备，派发仍受 `:958` 评审门禁与 (i)–(v) 启动前置约束（落点 `:1094` 与 r14 补充段）；(ii) 把 `:981`／`:998`／`:1021`／`:1037` 的 ADR-0032 绝对行号锚改为符号锚「ADR-0032 修订行（`adr-0032-r05`）」与「ADR-0032 解除顺序第 8 条候选 S4.24b 节点」；(iii) 登记见证与 anchor 同主机的实测事实，判定**不满足** `:953(c)` 判据 1 与 `:990` 不同主机／集群要求，并把同机故障域、无独立存储卷、无独立备份链、见证可被本机管理员重置登记为残余风险（落点 `:1080` 与 r14 补充段）；(iv) 补登 (iv) 实测值——`listen_addresses=localhost`、无独立网络入口、数据目录 `/var/lib/postgresql/16/anchor`、`<WSL_VHDX>` 与业务库同主机、无独立备份链，逐项标不满足或 unknown（落点 r14 补充段）；(v) 处置次要／建议项——Lead 两槽补记无 Agent ID、`enter`／`install`／`resume`／`unlock` 全部 closed、批准记录缺项清单、Herschel／Rawls／Lead 跨槽兼任登记为「记录性分离，不构成真实组织隔离」、代理范围与 r13 报告标注更正（落点 r14 补充段）；(vi) 登记 (b) 契约 errata——单表单一 `pending_token` 下「非部分唯一键 + 触发器」不可满足，采用部分唯一索引 `WHERE kind='pending'` + 触发器形态（偏离原文「非部分唯一键」字面），并登记 run-03 23505 证据与 run-07 并发负向用例结果（落点 r14 补充段）。未改「首期不提供」结论、约束 1–10 正文、解除条件、`:960` 历史原文、r01–r13 追溯行与 ADR-0019／0031／0032 任何字节；未补齐任何 unknown，未新增实现任务、未启动任何候选卡；r13 与本 r14 均未取得架构与安全两路接受，生产继续 closed。输入为改前本文件 `5F0C1D3FF338A09EB5C2A30D970F001A9514B86C7252F6B447C71AC74EF8178A`（306,594 B / 1,333 行）；改后证据见 `.spec/wip/recovery-runtime/evidence-b13j-d05/adr-0030-r14-author-report.md`。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r15 修订（触发：r14 两路非作者独立复评——架构侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r09/review-arch-r09.md`（33,313 B / `E285D20217E5602DA52885D976613BAC54A3CC5285D5FE48BD6A5E16E5084E33`，PASS WITH CONDITIONS，0 BLOCKER／0 HIGH／1 MEDIUM `ADR-R09-01`／3 LOW `ADR-R09-02`–`ADR-R09-04`／1 INFO `ADR-R09-05`）与安全侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r09/review-sec-r09.md`（20,613 B / `FF38A203E3B134DCFE155AD3336337589FB1FD430384CB4B94F41D758117564E`，PASS WITH CONDITIONS，0 BLOCKER／0 MEDIUM／4 LOW `ADR-S09-01`–`ADR-S09-04`／3 建议））：只做文本一致性收口，不新增约束。(i) `:952(b)` 增 errata 优先级声明并互相交叉引用——单表单 token 列的形态下以 `:1118`–`:1122` 的部分唯一索引 + 触发器为准，字面「非部分唯一键」在该形态下不适用（`ADR-R09-01`／`ADR-S09-04`）；(ii) r14 追溯行的 `:989` 单点更正为 `:990`（`ADR-R09-02`）；(iii) `:1120` 把「不改变失败分类」改为如实表述——引用失败由外键 23503 变为触发器 23514，均属拒绝、不改变应用层「失败即 closed」语义，但 SQLSTATE 与定位点改变（`ADR-R09-03`）；(iv) `:1094` 与 `:1108` 统一为「(iv) 已登记实测值，来源地址范围与升级窗口仍 unknown」（`ADR-R09-04`）；(v) `:1120` 触发器条件补全 namespace／journal_id／epoch（`ADR-S09-02`），`:1108` 登记超级用户可绕过索引与触发器的残余风险（`ADR-S09-03`），r15 补充段新增约束命名表与 42501／23505／23514／55000 归属（`ADR-S09-01`）；(vi) r13 追溯行追加作废标记（`ADR-R09-05`／安全建议-2），安全建议-1 写明归属 S4.5 实施/验证卡与 `:954(d)` 部署运维责任、建议-3 登记不采纳理由。未改「首期不提供」结论、约束 1–10 正文、解除条件、`:960` 历史原文、r01–r12 追溯行原文与 ADR-0019／0031／0032 任何字节；r13 追溯行只追加作废标记、r14 追溯行只做锚点单点更正，其余逐字保留。未补齐任何 unknown，未新增实现任务、未启动任何候选卡；r14 与本 r15 均未取得架构与安全两路接受，生产继续 closed，约束 9(b) 启动前置仍不满足。输入为改前本文件 `FAD94D0718C5529D71297B28C38B52563CFD0C07C65F25503C21A7A52E35DC51`（319,655 B / 1,362 行）；改后证据见 `.spec/wip/recovery-runtime/evidence-b13j-d05/adr-0030-r15-author-report.md`。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r16 修订（触发：r15 两路非作者独立复评——架构侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r10/review-arch-r10.md`（30,071 B / `D29222B951394245D209C59A717247256EFDB443B5AEF5C26484BE86AD78AD9A`，ACCEPT，0 BLOCKER／0 HIGH／0 MEDIUM／3 LOW `NEW-R10-01`–`NEW-R10-03`）与安全侧 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-sec-r10/review-sec-r10.md`（12,011 B / `CB8097FEC52A5FBE678A30443AD5642B29BCE5FB80DBFD23CC47875DDDE6A4D9`，PASS WITH CONDITIONS，0 BLOCKER／0 MEDIUM／3 LOW `ADR-S10-01`–`ADR-S10-03`／3 建议））：只做引用精度收口，不新增约束、判据或门禁。(i) r15 补充段约束命名表 pending 部分唯一索引行的「旧形态的引用失败为外键 23503（run-03 A3）」改为如实表述——run-03 的 A3 实测为 23505（`anchor_events_pending_uq`），23503 只出现在同一日志 A6／A7 的期望注释、未实测，自 run-05 起 A6／A7 改为触发器校验并实测 23514（`NEW-R10-03`／`ADR-S10-01`）；(ii) 同表 append-only 触发器行的「管理员改写、删除、截断 → 55000」限定为「超级用户（表 owner）→ 55000；受限写入角色同一操作 → 42501」，r14 补充段「并发负向用例实测」条同步改称「管理员（超级用户）」，并与 r14 补充段「目标环境登记补全」条的超级用户绕过登记互引（`ADR-S10-02`）；(iii) 本 ADR 段内指向自身的绝对行号一律改为符号锚——r05 结论 (a)–(d)、「对下游门禁的影响」条、「启动前置 (i)–(v) 已登记建议值」条、r06–r09 与 r13／r14 补充段各条标题、r13／r14 追溯行，原系统性 −2 行号偏移随之消除（`NEW-R10-01`）；(iv) 「安全建议处置」条与 r15 追溯行中原以改前行号指称 r13／r14 追溯行的引用改为「r13 追溯行」「r14 追溯行」符号锚（`NEW-R10-02`／`ADR-S10-03`）；(v) 安全侧三项建议已在 r15 补充段逐条处置或登记不采纳理由，本修订不重复处置。未改「首期不提供」结论、约束 1–10 正文、解除条件、ADR-0028／ADR-0031／ADR-0032 任何字节；未改启动前置条原文（含「(iii) 当前 unknown」与「末句启动禁令」）与 r13／r14／r15 追溯行的其余文字；未补齐任何 unknown。改前本文件 `15E38DF132E30FCCF1473AB7D7EC62DDFA64E63F334F533144019731E6C143E3`（328,647 B / 1,387 行）；改后证据见 `.spec/wip/recovery-runtime/evidence-b13j-d05/adr-0030-r16-author-report.md`。r16 未取得架构与安全两路接受，不构成评审 PASS、人的接纳或生产批准；约束 9(b) 启动前置仍不满足，派发与启动未获授权，生产继续 closed。
- 2026-11-03 r17 修订（触发：用户本轮下达的 D09 文档型决策卡；确认来源为用户 2026-11-03 明确授权「待确认项按建议值、不需要完全模拟正式项目推进」，该授权只覆盖决策文档写作，不构成人的成果接收、独立评审、生产装配批准或真实验证）：在 ADR-0030 段内新增「r17 补充」小节，登记四项决策级结论——(1) 生产恢复链在测试项目内不实施，`identity-auth-S4.5`／`S4.6`／`S4.23`／`S4.24a`／`S4.24b`／`S4.22` 维持未派发、未认领、未实施，ADR-0030「首期不提供生产 CAS provider」、ADR-0031「首期不提供生产 restore 工具与批准主体」、ADR-0032「`Exclusive` 仍 closed」三条结论与生产 `enter`／`install`／`resume`／`unlock` 的 closed 状态均不变，并显式登记 `tasks.md:3154` 卡面目标与 `identity-auth/design.md:3612`–`:3616`／`:3910`、`framework.md:926` 的正面冲突（以「不实施」收敛，不改任务卡掩盖）；(2) B1／B3／B4／B5 六项（adapter ID 字面值、witness 段 schema、anchor 见证传输协议、`(w,r,token)` 读-条件写存储 API、commitment／source 存储载体、`AnchorSnapshot` 令牌 ↔ `bigint` 映射）在本项目内不予登记，统一记为「产能外 unknown」，未来若恢复该链必须先由本 ADR 新修订给出这些值；(3) 按 ADR-0028 约束 1／2／3／4／6／7 收口剩余可执行层——`identity-auth-S4.4` 允许模拟等级执行且 `S4.5`／`S4.6` 登记为 `waived-for-test-scope`，`identity-auth-S2.2` 采用已登记环境与容量建议值（模拟基线、非生产 SLO），`identity-auth-S3.4`／`S3.6`／`article-management-S1.4` 的真实环境层按模拟替代并逐条登记模拟／替代／残余风险；(4) 禁止据此宣称生产解锁、生产就绪、真实故障域成立、ACL 已验证或真实环境 PASS，禁止把模拟结果写成 PASS，禁止删除失败证据或改写已关闭卡历史。B6 平台限制与 B7 seam 未处置（B7 待 Lead 裁定）。未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0028／ADR-0031／ADR-0032 任何字节；未改 `tasks.md`、任何 `*.go`、`migrations/`、`cmd/`、`internal/` 与两份 design.md。改前本文件 `506FD1E0A8962BEB716488815EBA44060EE52B3105E4AC88CB22A8EC8893F23A`（334,144 B / 1,388 行）；改后证据见 `.spec/wip/recovery-runtime/evidence-b13j-d09/report.md`。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-03 r18 修订（触发：D09 r17 两路非作者独立评审——架构侧 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-arch-r01/report.md`（44,951 B / `C8C5F8166207D04F9F4E4477582232B5D620B42597AC87FA1CDD62373AE29B7B`，CHANGES REQUIRED，F001–F009）与安全侧 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-sec-r01/report.md`（20,991 B / `712E96ED4534362F27C286B3E6A5C013EAEFC3C80C1CF12B8A4E790900E31307`，CHANGES REQUIRED，3 项阻断 MEDIUM + 4 项 LOW）；Lead 对 F001 裁定 `.spec/wip/recovery-runtime/evidence-b13j-d09/lead-ruling-f001.md`（1,712 B / `1369F23769E00B01F4F59BF6723C054A3D320C39358383D05CEF44AA780F0DBB`，采用 (A)）与对 B7 裁定 `.spec/wip/recovery-runtime/evidence-b13j-s423/lead-ruling-b7.md`（1,464 B / `C39AEED822201D8CE68C1D0434877A95BA7CCD92F958BD1280D468069AD95F53`））：新增「r18 补充」小节，以取代表方式处置——(1) 从决策 3.3 移除 `identity-auth-S3.6`，模拟替代表收窄为 `identity-auth-S3.4` 与 `article-management-S1.4`，S3.6 维持 blocked／未派发，扩大须用户另行明确决定并由新修订登记（F001／Lead (A)）；(2) `waived-for-test-scope` 补全为范围豁免（test scope only）：不改 `tasks.md:649` requires 文本、不改任何生产 requires、不使 S4.5／S4.6 变为已实施、不构成安全豁免（F002／安全第 1 项）；(3) S4.6 豁免只以 ADR-0028 约束 2 为唯一依据、只解除 S4.4 模拟前置，本 ADR 对 key retirement 无决策权限（安全第 2 项）；(4) 删除「模拟见证／备份卷替代真实独立故障域」，改为测试范围内不取得该证据、登记未验证／unknown，不满足 ADR-0030 `:953(c)`（F003）；(5) 决策 1 补 ADR-0028 约束 1 范围声明，不实施 ≠ 豁免验证（F004）；(6) 更正 B4 行 `anchor-env-r01/README.md` 存在（5,941 B / `A081AA9B300655B9FD94C55CE74DC0104928659D8E8EC507786CDBA212AD74B1`）但不构成登记值（F005）；(7) 依据清单补 ADR-0028 约束 5（`:851`）与约束 8（`:854`）（F006）；(8) S2.2 建议值逐字引用 `preflight-record.md` §9（F007）；(9) B2 冲突改为保留、未消除（F008）；(10) 更正 `:1162`／`:1163` 与作者证据 `:49` 的锚点精度（F009）；(11) 补记 r16 两路接受不自动延续至 r17／r18，各自须重新取得（安全第 3 项）；(12) B7 seam 按 Lead 裁定保留、语义限定「合成/测试注入」，生产 `main.go:634` 未传四端口（B7 裁定）。未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0025／ADR-0028／ADR-0031／ADR-0032 任何字节；未改 `tasks.md`、`decisions/README.md`、任何 `*.go`、`migrations/`、`cmd/`、`internal/` 与两份 design.md。改前本文件 `BC42F14B487A2F99249B57F4D2413F82650BB404C08BF75FA87AB083BEC8B5B7`（352,220 B / 1,486 行）；改后证据见 `.spec/wip/recovery-runtime/evidence-b13j-d09-r18/report.md`。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-03 r19 修订（触发：D09 架构侧 r02 复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-arch-r02/report.md`（26,857 B / `CEBF0C0F13A06DBD698F680EF3A48D438EA1C2D591C5CB93FF823175959AEE0A`，CHANGES REQUIRED，F010–F014）与安全侧 r02 复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-sec-r02/report.md`（20,952 B / `907F62DD2C1786AE30A99E6537AF27B60514566589FD6A9E6A0866130922859A`，PASS WITH CONDITIONS，条件为 F010 索引收口）；F010 由 Lead 处置 `.add/project/knowledge/decisions/README.md`，不在本 ADR 写集内）：本次只做四项定点更正——(1) F012：按 Lead 2026-11-03 对 F001 裁定第 1、3 条的授权，物理更正 r17 决策 3.3 段（`:1227` 标题收窄为 `identity-auth-S3.4` 与 `article-management-S1.4` 两卡并去掉「可观测平台」、删除 `:1234` S3.6 行、`:1229`「三条均」改「两条均」、段末加 r19 更正引用块），r18「r17 原文逐字保留」在 3.3 段不再适用；(2) F011：r18 取代表末尾补一行点名 r17 `:1245`，限定为测试范围窄豁免、不改 `tasks.md:649` requires 文本、不得读作 requires 已满足；(3) F013：`:1323` 对 `tasks.md:664` 改为无引号转述，对 `preflight-record.md` §9 `:122` 改为逐字引用并补回 (iii) 子句；(4) F014：`:1290` 跨节锚点按实测更正为 `preflight-record.md` §2 的 (i)、(iv)、(v) 行引用 `anchor-env-r01/README.md` §1、§1／§4、§5，不照抄 r02 建议句中「§5 未被 preflight 直接引用」的表述。未改「首期不提供」结论、约束 1–10 正文、解除条件与 ADR-0025／ADR-0028／ADR-0031／ADR-0032 任何字节；未改 `tasks.md`、`decisions/README.md`、任何 `*.go`、`migrations/`、`cmd/`、`internal/` 与两份 design.md。改前本文件 `4EC6C2AF086BEC4ABB59E5DE180CC0323FA13EBD40D532098B5B271FDE1A148C`（370,555 B / 1,570 行）；改后证据见 `.spec/wip/recovery-runtime/evidence-b13j-d09-r19/report.md`。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-03 r20 修订（触发：D09 架构侧 r03 复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-arch-r03/report.md`（22,404 B / `763459D8D8F1AC179EF9438E6D039B93A10F5E2FDE6EEDE39AF049136C8E1B8C`，PASS WITH CONDITIONS，`B13J-D09-ARCH-R03-F015` MEDIUM 条件项与 `B13J-D09-ARCH-R03-F016` LOW）与安全侧 r03 复审 `.spec/wip/recovery-runtime/evidence-b13j-d09/review-sec-r03/report.md`（24,209 B / `951921836EF102A0E8F89213CD613385F723C811D6E5C62466894FB09B9CD85B`，ACCEPT，`B13J-D09-SEC-R03-F001`／`F002`／`F003` 均 LOW））：本次只做五项文本收口——(1) `B13J-D09-ARCH-R03-F015`：`waived-for-test-scope` 专段末句改为「`tasks.md` 侧不加注记，requires 原文即权威；本 ADR 不改 `tasks.md`，故不存在 ADR 与任务卡的双权威」，与 `:1272`／`:1341` 统一；(2) `B13J-D09-ARCH-R03-F016`＋`B13J-D09-SEC-R03-F001`：`:1325` 对 `preflight-record.md` §9 `:122` 的引文补全句尾「；不再以「结构性不可满足」维持四卡阻塞」，成为整句逐字引用，`:1345` 同步注明 r19 时点为句首片段；(3) `B13J-D09-SEC-R03-F002(a)`：`:1353` 把 README 描述改为 r18 时点表述并登记 Lead 索引收口（README 现为 22,942 B / `53FD5846C2FE2BEEBB9B8BDCCF1E5239C469B79588E3965B68196550C4BEB328`）；(4) `B13J-D09-SEC-R03-F002(b)`：`:1333` 安全侧 r02 条件项编号由误记 F010 更正为 `B13J-D09-SEC-R02-F001`，并写明两侧为两件事且均已收口；(5) `B13J-D09-SEC-R03-F003`：r20 小节以代码块逐字保留 r19 删除的 r17 旧 `:1234` S3.6 模拟替代表行原文，仅作可追溯、不恢复效力。未改「首期不提供」结论、约束 1–10 正文、r17 决策 3.3 段的 r19 更正、r01–r19 追溯行与 ADR-0025／ADR-0028／ADR-0031／ADR-0032 任何字节；未改 `tasks.md`、`decisions/README.md`、任何 `*.go`、`migrations/`、`cmd/`、`internal/` 与两份 design.md。改前本文件 `7C2E8A38E7CA92AB2AA9DA3111C3271745EA5E862F6F2316EF1E74676BCEB50C`（376,429 B / 1,597 段）；改后证据见 `.spec/wip/recovery-runtime/evidence-b13j-d09-r20/report.md`。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。

---

<a id=adr-0031></a>
## ADR-0031: 首期不提供生产 restore 工具与批准主体，固定受批准执行与宿主隔离合同 (2026-11-01)

**目的**: 架构设计 | **仓库**: han | **修订**: adr-0031-r12

**状态**: 已接受（2026-11-01 用户 D03 指令授权 Arch-Designer 论证并落定“首期不提供生产 restore 工具与生产批准主体”；确认来源为本轮用户任务指令；不构成人的成果接收、独立评审、安全评审、生产装配批准或真实工具/备份验证；决策接受级不按固定修订号记录，而随 `**修订**` 行的现行修订号自动生效——现行修订须同时取得架构与安全两路独立评审接受，才解除实施/验证卡的派发、认领与开始改动限制；本修订 `adr-0031-r12` 的两路接受尚未取得，`identity-auth-S4.24a` 及任何实施/验证卡不得启动；本 ADR 的任何修订均不改变「首期不提供生产 restore 工具与生产批准主体」） | **确认来源**: 用户 2026-11-01 在 B13J D03 中明确要求把受批准 restore 工具、备份批准主体、执行主体、宿主隔离与原生耐久回执落成可审计、fail-closed 的架构决策，并明确“首期不提供”是合法结论之一。该来源授权的是架构决策与文档写入，不是生产装配、真实旧备份演练、安全评审或人的成果签收。

**继承关系**: 只固定 `identity-auth/design.md` §4.2 中“受批准 restore 工具 / 备份批准主体 / 执行主体 / 宿主隔离 / 原生耐久回执”这组组织与信任边界；不改变 ADR-0019 已锁定的独立故障域 CAS 头锚点、keyring、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒与 fail-closed 失败分类；不改 ADR-0030 的“首期不提供生产 CAS Anchor provider、S4.5 未交付、生产 enter/install 保持 closed”；不改变 ADR-0027 的恢复 runtime 编排与正向准入证明；不重写或削弱 design 既有 restore 软件合同（`restore-v1`/`verify-v1` 的 argv 与私有 stdin、no-follow 稳定 fd、最小 ENV allowlist、receipt 字段、`restore_unknown` 保守暂停、Begin 前预检失败即拒绝），只补其批准、执行身份、宿主隔离与原生回执这组前置前提；若后续发现软件合同本身有缺口，另立决策或设计修订处理，不在本 ADR 内改写。

**问题**: 设计已固定 restore 的软件协议与 Begin 前预检，但没有可核的批准主体、执行主体、执行身份与进程隔离证明，也没有工具原生耐久回执的真实样本：`RestorePlan.SourceApprovalRef/Digest` 由谁签发、`HAN_RECOVERY_RESTORE_EXECUTABLE` 的二进制 SHA/owner/ACL 由谁批准、由哪个进程以什么身份执行替换、root 与执行者之间的权限边界、宿主隔离的最小必要机制、以及“该 artifact 已还原”由谁以什么形态提供持久证据，改动前均为 unknown。生产 root 因此无法构造可验收的 `RestoreDriver`，生产 `enter`/`resume` 的 DB 替换只能在 Begin 前拒绝。需要在不虚构未验证工具、批准角色或演练结果的前提下，给出唯一可验收口径。

**决策**: 首期不提供生产 restore 工具与生产批准主体。生产 `enter`/`resume` 的 DB 替换路径继续保持 closed；与 ADR-0030 的合并口径是“生产 `enter`/`install` 与 `enter`/`resume` 的 DB 替换路径均保持 closed”，二者是不同入口而非范围收窄。仅保留既有软件合同、确定性测试替身与 `restore_unknown` 语义。`HAN_RECOVERY_RESTORE_EXECUTABLE`、`HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 与 `HAN_RECOVERY_RESTORE_MANIFEST_PATH` 仍是唯一入口且无默认值、未知不补值；`verify` 与 `genesis` 形态不执行工具。任何未来生产形态必须同时满足下节全部批准、执行身份、宿主隔离与原生回执不变量，否则不得解除 closed。

### 现状 / 痛点 / 改造 / 效果

现状是“协议已定、执行主体未定”：设计已把 `restore-v1`/`verify-v1` 的 argv、私有 stdin、receipt 字段、no-follow 稳定 fd、最小 ENV、Begin 前预检与 `restore_unknown` 写成软件合同；但没有任何具名批准角色、已批准二进制 SHA、专用执行身份、权限矩阵、进程隔离实测或工具原生耐久回执证据。当前不存在可执行的生产 restore 工具，也不存在可核的备份来源批准记录。

痛点在于：若现在声明“由部署方批准、由 root 子进程执行”，等于把一个不存在的批准角色、未指定的执行身份和未验证的隔离机制写成公共合同；若继续维持“可配置但未定义”的模糊状态，实施者容易用 shell 包装、附加 argv、环境探测、caller 自报 done、DB 可连接或自己刚写的 run 行填补空缺。另一类风险是把 `restore_unknown` 当缺陷而用自动重放“修复”。

| 方案 | 实现机制 | 优势 | 劣势与未满足条件 | 结论 |
|---|---|---|---|---|
| A. 首期不提供生产 restore 工具与批准主体（选定） | 不注册生产 restore 可执行文件，不任命生产批准角色；生产 `enter`/`resume` 的 DB 替换保持 closed；仅保留软件合同、确定性测试替身与 `restore_unknown` | 不引入未验证工具、批准角色或隔离声明；与 ADR-0019 的 fail-closed 及“测试替身不得作为生产证据”一致；解除路径唯一且可审计 | 首期生产恢复链不可用；旧备份只能离线处理；后续必须重新选型、验证并修订 ADR 才能解除 closed | 选定 |
| B. 部署方受控只读挂载已批准二进制 + 具名部署批准角色 + root 子进程执行 | 部署方把已批准二进制放入受控只读挂载，以批准记录绑定 SHA256/owner/ACL；具名恢复批准角色批准备份来源；root 以固定 argv 启动子进程并保留锁与验证 | 与既有软件合同最接近；工具可独立升级与替换；root 仍持部署锁、Journal/keyring 与最终验证 | 当前无具名批准角色、无已批准二进制、无专用执行身份与权限矩阵、无进程组/no-follow/ENV allowlist 实测、无工具原生回执；工具供应链、更新责任与真实旧备份演练均未定 | 不选 |
| C. 独立恢复运维单元执行替换，本服务只做 VerifyRestored 与绑定 | 外部团队或系统在其环境执行 restore，本服务只验证恢复结果并写绑定 | 职责分离强；本服务不承载替换执行面 | 关键替换动作移出本服务审计与进程隔离边界；如何证明执行的是同一 artifact/target、如何取得原生回执、如何阻止并发写与重放、责任与时限均未定；当前无该单元 | 不选 |
| D. 托管数据库平台原生 PITR/备份恢复 + 平台原生回执 | 由托管 PostgreSQL 平台执行按时间点恢复，平台提供恢复任务记录作为原生回执，本服务只做 VerifyRestored 与绑定 | 平台原生持久回执；无需自管工具链与执行身份 | 锁定具体云产品、区域与凭据；需证明平台恢复能精确绑定 artifact/target/source installation 与 schema digest，且不会把平台自身回退误判为恢复成功；当前无实例与验证证据 | 不选 |

选择 A。B、C、D 都需要当前不存在的具名批准角色、已批准二进制或平台实例、专用执行身份、隔离实测与原生回执证据；在缺少这些证据时选择任一形态，等于把“未验证”写成“已批准可执行”。A 不新增外部依赖，不触发 shell/附加 argv/自动重放/自动 fallback，保留既有软件合同与 `restore_unknown`，并把解除条件固定为可核清单，是当前唯一可审计且不伪造生产就绪的结论。

改造是把“谁批准、谁执行、用什么隔离、回执从哪来”从实现细节提升为生产构造门禁：未来任何生产 restore 形态只有同时具备具名批准记录、已批准二进制、专用执行身份、隔离实测与工具原生存久回执，才可能通过 Begin 前预检；任一缺失一律 closed，且不得用 caller 自报、exit=0、stdout receipt 自洽、DB 可连接或本次 run 行替代。

效果是：生产 root 不会因环境变量存在、文件可执行或 manifest 可读而误开替换；测试替身继续只服务协议测试；`restore_unknown` 保持为明确恢复结果而非缺陷；解除 closed 需要一次性补齐全部证据并经独立评审与正式登记，避免实现阶段自行补默认值或引入未批准工具。

### r05 四项结论（决策级要求；可核结论或明确 unknown）

本小节按 `identity-auth-D06` 的目标逐项给出批准主体、执行主体、宿主隔离与原生耐久回执的结论。四项都只给判定结构、最小字段与否证条件，不声称任何角色已任命、任何身份或隔离已实测、任何回执已取得；凡当前无据可核的维度一律记 unknown 并写明补证主体。四项的门禁分级见本小节末「决策接受级／卡证据级／生产解除级」三段：该禁令以决策接受级为条件，事实级取证（任命、身份与隔离实测、回执样本）属于实施/验证卡的退出条件与生产解除级，不作启动前置。

- **1. 批准主体**：角色唯一且具名——部署方以书面任命指定唯一具名「恢复批准角色」，任命须写明适用系统、生效时间与替补；该角色不得是执行者、本次 run 发起者，也不得是工具或备份供应方，不得自批。批准记录为备份来源与可执行文件两条独立记录，最小字段为：记录类型、角色名与具名人员、任命依据引用、签发时间（UTC）、适用 installation 与 backup ref、artifact digest（SHA256）、source installation id、可执行文件绝对路径与 SHA256、该文件的 `st_dev`/`st_ino`、owner/ACL、受控只读挂载来源、复核人与复核时间、有效期与撤销状态；复核人须具名且独立于签发人与执行者，同一人兼任签发与复核时该记录不成立。可核判据：Begin 前 `RestorePlan.SourceApprovalRef/SourceApprovalDigest` 必须命中一条存在、未撤销、字段齐全且与本次 backup ref 和 artifact digest 精确匹配的来源记录；可执行文件 SHA256 必须与批准记录及 `HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 三方相等，运行时 `st_dev`/`st_ino` 必须与批准记录相等；执行必须经固定 fd-exec 机制——对已核 no-follow 稳定 fd 直接执行（`fexecve`、`execveat(AT_EMPTY_PATH)` 或 `/proc/self/fd/N` 一类等价方式），不得校验后按路径重新打开再 exec；该机制由批准记录固定为决策级要求，本 ADR 不选具体实现。任一不匹配即 closed。**unknown**：本项目当前未任命该角色，也不存在任何批准记录；任命与记录的取证主体是部署方（组织侧）而非 Agent，补证形态是书面任命与两条批准记录原件。
- **2. 执行主体**：形态唯一——替换只由 root 启动的独立非交互子进程执行，运行身份为专用执行身份，须同时满足：不等于 root、不等于批准角色、不与该角色共用凭据、无交互终端、不继承业务 secret/审批 key/部署锁/Journal/keyring/管理入口凭据，只持只读 artifact fd、私有 stdin 的 target_connection、最小 ENV allowlist 与目标 DB 网络。可核判据：实现阶段须给出子进程实际主体标识与 root 的差异实测，以及「替换窗口外以该身份写 Journal/keyring/部署锁/Store 权威绑定/审计必须被拒绝」的负向用例。**unknown**：具体身份切换机制（不得经 `su`/`sudo`/shell 包装，机制由批准记录固定）、执行身份的具名主体、ACL 与 GRANT 矩阵均未定；补证主体为部署方（主机与数据库侧）与实现阶段的隔离实测；本 ADR 不选具体机制。
- **3. 宿主隔离**：矩阵已定且不放宽——POSIX 是唯一生产候选（首期仍 closed）：固定 argv 直接 exec 唯一批准程序，无 shell 与附加 argv；独立进程组/会话，父进程持全部子进程句柄，取消或超时终止整组并真实 join/reap；no-follow 稳定 fd 且校验前后 fstat 与路径身份一致，禁止验后按路径重开；最小 ENV allowlist（`env_clear` 后仅按批准记录内固定的封闭字面集逐项设置），零继承业务 secret/审批 key/宿主凭据；除目标连接外无网络；任一隔离预检失败在 Begin 前拒绝。Windows 生产 unsupported：任何 Journal/DB/CAS/文件状态变更前返回退出码 2，仅允许软件合同测试与 Job Object 等价软件试验，不构成生产支持。**unknown**：上述 POSIX 机制目前没有任何实测证据（无进程组、fd 与负向权限实测记录）；补证主体为承载实现卡与独立主机复跑。否证条件：出现 shell/附加 argv/`su`/`sudo` 包装、进程组未完整 join、验后按路径重开、ENV 含业务 secret，任一成立即 closed；ENV 判据不限于 secret——`LD_PRELOAD`、`LD_LIBRARY_PATH`、`LD_AUDIT`、`GLIBC_TUNABLES` 一类动态加载变量，`*_PRELOAD`/`*_OPTIONS` 一类注入变量，或 `PATH` 指向任一非批准目录，出现即 closed；子进程实际 environ 与批准白名单不逐字节相等同样 closed。
- **4. 原生耐久回执**：来源与绑定已定——回执只能由 restore 工具或其备份系统原生写入、独立于本次进程可重读的持久记录（恢复目录、恢复任务记录或安装来源证据）提供，不得由本次进程、caller、DB 可连接性、exit=0 或本次 run 行充当；root 必须经与写入路径不同的独立读取路径读取；回执须同时绑定 artifact digest、target（installation/deployment/target digest）与 source installation（`SourceInstallationID` 与 schema digest），并与 `VerifyRestored` 的只读核对逐项一致；回执还须具备抗本机伪造的真实性保证——带工具或备份系统的签名（或等价外部见证），且由独立于本次进程与 root 的外部系统产生，并含产生者身份与单调序列；该保证按合取判定：验证方须同时独立于执行身份与 root——验证方式为外部系统验签，或由作为批准角色的人核对，两种方式都必须满足该独立性合取；读取回执时必须先验签并留审计记录，验签信任锚（公钥或见证端点）固定登记在批准记录内、其轮换须经人的明确批准；该约束只作用于非 root 主体——root 能整体替换信任根与见证（见下述无条件残余），故不得据此声称抗 root。**unknown**：具体工具或备份产品的回执字段名、存储位置、读取命令、签名或外部见证机制、验签信任锚的具体形态、验证方主体与真实性保证均未定，也没有任何真实旧备份产生过的样本；补证主体为部署方与工具/备份供应方。否证条件：仅有 stdout receipt、exit=0、DB 可连接、当前行数或本次 run 行，或回执无法绑定上述三项，或无独立读取路径，或回执可被 root 或替换后的工具单独伪造、事后重写或重建而无法验出，或读取时不验签、无审计记录，或验签信任锚未固定于批准记录、可由 root 或被验证回执自选，或验证方与执行身份或 root 任一不独立，均不得作为恢复证明并按 `restore_unknown` 处理。**无条件残余（r10）**：回执抗伪造止于 root／部署方——root 可整体替换信任根、批准记录与见证，本 ADR 不声称抗 root（与 design「r64.10 审计 sink 边界与抗 rename」与「r65.13 残余与未闭合项」的「不声称抗 root」一致）；该残余无条件成立，不取决于组织侧是否提供独立验证方与固定信任锚；生产 restore 因此继续 closed，不得以「有签名」或「有外部见证」暗示已抗 root 伪造。**r09 外部锚补充（三签与外部锚；r10 校正签名角色）**：外部锚采用「独立见证签名」路线；签名角色按 design「r65.5 V 编号、`anchor_ref` 五处等式链与 R 图既有检查节点」与「r66.2 V7–V11 的载荷与签名主体唯一绑定」的三签模型固定为三个互不相同的席位，任一席位不得代签其他席位的载荷：其一，**producer** 以 `receipt_anchors.public_key` 签发回执信封 `JCS({"receipt": R10, "attestation": A6})`（`R10`／`A6` 为 design「r66.2 V7–V11 的载荷与签名主体唯一绑定」的记号，与本 ADR 修订号 `r10` 无关；`R10` = 12 字段回执删除 `signature`／`audit_record_ref` 后的 10 字段对象，`A6` = 7 字段 attestation 删除 `signature` 后的 6 字段对象），回执验签公钥唯一来源是 `receipt_anchors.public_key`；其二，**verifier** 以 `keys[verifier_key_id].public_key` 签发 8 字段验证方载荷（`anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id`），该 key 的 `principal_id` 必须等于 `verifier_principal_id`；其三，**witness** 以 `receipt_anchors.witness_key_id` 对应公钥签发 5 键 head（`anchor_ref, sequence, issued_at, head_sig_alg, head_sig`）删除 `head_sig` 后的 JCS 字节。三个席位的身份／密钥标识登记在 `receipt_anchors`（`producer_key_id`／`verifier_key_id`／`witness_key_id`），公钥按席位分别取自 `receipt_anchors.public_key`、`keys[verifier_key_id].public_key`、`receipt_anchors.witness_public_key`，分属 producer／verifier／witness 三个席位；`witness_key_id` 只是见证公钥的标识，**不是**回执签名的公钥唯一来源。本条是对既有 design 三签模型的笔误校正，不是新决策，不改变「首期不提供生产 restore 工具与生产批准主体」结论。回执 `sequence` 必须严格大于 `max(baseline_sequence, head.sequence)` 且 `baseline_sequence` 只增不减。验签失败、见证不可达或无法原子推进 → closed／`restore_unknown`，不得推进 `sequence`、不得当成功；本地文件、本机进程或服务自身不得冒充外部见证。模拟实现是本机独立 witness CLI（Ed25519，私钥不入服务 ENV／工作目录），只是测试替身，证据必须标注「模拟见证、未真实验证」；真实实现为外部审批系统／HSM，当前未验证。**`anchor_ref` 五处等式（r10 定名）**：`receipt_anchors.anchor_ref` = 批准记录 policy 的 `tool.receipt.trust_anchor_ref` = 回执 `trust_anchor_ref`（回执 wire 的 12 字段中没有 `anchor_ref` 键，`trust_anchor_ref` 是唯一载体）= 审计记录 `anchor_ref` = 见证 head `anchor_ref`，五处逐字节相等。policy 的 `trust_anchor_ref` 是 design 规定的 policy 必填字段，缺键、空串或字段名不符即 closed——启动前 `ErrAdmissionBinding`，工具已启动后 `restore_unknown`；不保留「policy 该键可缺」的条件分支。审计记录 `verifier_principal_id`／`verifier_key_id`／`anchor_ref` 必须与验证方载荷逐字节相等。**r09 撤销与 TOCTOU 同步（r10 补全 generation 与检查点集合）**：本 ADR 不引入新的 `generation` 字段，`generation` 定义为可复算的三元摘要，三项同时逐字节相等才算同一 generation：（i）`approval_generation` = 批准文件精确字节的 SHA-256（预检不可变快照记录，`revalidateApproval` 重读比对）；（ii）`trust_root_generation` = 信任根文件精确字节的 SHA-256（必须等于 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`）；（iii）`anchor_generation` = `receipt_anchors` 中该 `anchor_ref` 条目 JCS 规范字节的 `sha256:` 摘要（必须等于 policy 与回执的 `trust_anchor_sha256`）。任一不等于快照即 generation 漂移 → closed／`restore_unknown`；不得以「文件可读」「inode 未变」或「长度未变」替代摘要比较。撤销与有效期重验按 design「r64.5 唯一调用图完整替换」与「r65.5 V 编号、`anchor_ref` 五处等式链与 R 图既有检查节点」的检查点集合逐点执行，不得删点或改成单一全局检查：`P3` 只读取批准文件、信任根与目标 DSN 并构造不可变快照、完成首次合取（无前像可比，故不作 generation 漂移判定，也不属重验点）；`P6` 在 `Journal.Begin` 前重读批准文件精确字节并与快照比对；`R1`／`V1` 的 `prepared(ctx)`；`R2`／`R4`／`R6`；`V2`／`V4`／`V6`；以及 `V7` 独立读取回执前、`V8` 验 producer 签名前、`V10` 追加审计／验证方签名／推进见证 head 前各一次。`P6` 及 `P6` 之后每个检查点重读批准文件、信任根与 `receipt_anchors`，重算上述 generation，并逐项检查 `organizations`／`keys`／`principals`／`appointments` 每条记录的 `valid_from <= now < valid_until` 与 `revocation.state`＝`active`（且 `revoked_at`／`revocation_ref` 为空串的 golden vector 形态；任一非 `active` 即按撤销处理）、`reviews[].decision`，以及 `now - issued_at <= 366 天`；`now` 每点取一次并向内传递，`mono` 高水位回退即 closed。`receipt_anchors` 条目本身没有 `revocation`／`valid_until` 字段（design「r66.5 见证端点、TLS pin 与独立性」的 `receipt_anchors` 十五项 keyset 无此二字段），其撤销与轮换唯一经人的明确重钉批准，并以 `anchor_generation` 与 `trust_root_generation` 的变更体现；不得在 `receipt_anchors` 上查找不存在的字段。任一重验失败按路径与原因分写后果：`P6` 与 `Prepare` 内的重读按 design `:4377` 的原因分类——缺键、文件缺失或不可读、非绝对规范路径、owner／mode 不符、非 regular 或被符号链接替换等返回 `ErrAdmissionIncomplete`；未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked`，以及预检通过后的字节／`generation` 漂移，返回 `ErrAdmissionClosed`。`enter` 的 `P6` 失败不进入 `Journal.Begin`；`resume` 不新 `Journal.Begin`（design `:3672`／`:3886`／`:3899`）。`R1`–`R8` 只在 `enter` 的 `Replace` 路径出现，且位于 `Journal.Begin`（`internal/recovery/runtime.go:1617`）之后；`V1`–`V11` 属 `VerifyRestored` 路径，`enter`（`runtime.go:1634`）与 `resume`（`runtime.go:1666`）均会调用——`enter` 的调用位于同一 `Journal.Begin` 之后，`resume` 的调用在 `Prepare`（`runtime.go:1656`）之后且不伴随新的 `Journal.Begin`；`resume` 不调用 `Replace`。R／V 重验失败按该路径既有错误分类处置，判定顺序固定为：先看失败原因——缺键、文件缺失或不可读等按 `ErrAdmissionIncomplete`，未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked` 按 `ErrAdmissionClosed`，预检后的 generation／字节漂移在工具尚未启动时按 `ErrAdmissionClosed`；再看是否已越过工具启动点——已越过时无论原因为何一律按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)` 处理、由调用方按 `restore_unknown` 处置（`R6`／`V6` 以及 `V7`／`V8`／`V10` 前的重验均属此列）；任何情况下**不得推进见证 `sequence`**、不得当成功。其中 `V7`／`V8` 前的重验是本 ADR 在 design「r64.5 唯一调用图完整替换」与「r65.5 V 编号、`anchor_ref` 五处等式链与 R 图既有检查节点」的既有 V 图（既有重验节点为 V2／V4／V6，`V10` 另有 r67.5 的推进前重验）之外的追加关闭判据，只增加拒绝点、不放宽任何既有检查。本节所引 design 条款锚按 r66.0 的唯一优先级规则读取，不绑定会随修订推进而失效的固定修订号；现行 `identity-auth-design-ir01-r70` 未改这些合同（全文检索 `r68.`／`r69.`／`r70.` 仅命中 `:4072` 的行号映射说明（列出 `r68.1`–`r68.8`／`r69.1`–`r69.3`／`r70.1`–`r70.3` 的修订号与插入前行号对应，不构成合同条款）与 `:4083` 的「r70 修订沿用本声明」），故本 ADR 不据此声称已复核 r67–r70 的全部改动，也不改写 design。**见证术语消歧（r10）**：本节「见证」只指 restore 回执的**外部锚见证**（Ed25519 签 5 键 head、持久单调 `sequence`、`receipt_anchors.witness_*`）。它与 ADR-0030 的 **anchor revision 见证** `(w, r, token)`（ADR-0030 自身的 r08／r09 补充小节，`framework.md:1005`–`:1027`，见 `:1009`）不是同一系统：对象、状态、endpoint 与 key 相互独立，不得共用 key、endpoint、记录或推进通道，一套推进不得替代另一套。本节的 `anchor_ref` 指 receipt anchor，**不是** ADR-0019 的 `JournalHeadAnchor`，也不承担 CAS 头锚点职责。

四项的当前状态：批准主体与执行主体的形态和最小字段已定，但任命、具名主体与切换机制为 unknown；宿主隔离的矩阵与最小机制已定，但 POSIX 实测为 unknown；原生耐久回执的来源形态与绑定要求已定，但产品级字段与真实样本为 unknown。据此把门禁按「决策级可核」与「事实级取证」分级，三级依次为：

1. **决策接受级（唯一解除启动限制的条件）**：本 ADR 修订被架构与安全独立评审接受，且本节四项的决策级要素——形态、最小字段、可核判据、否证条件——齐备。达成该级只解除实施/验证卡的派发、认领与开始改动限制，使实施卡得以去产出证据；它不补齐任何 unknown，不等于人的决策接纳，也不产生任何人的批准。

2. **卡证据级（实施/验证卡的退出条件，不是启动前置）**：约束 8 所需的 POSIX 隔离实测、ACL/GRANT 权限矩阵负向用例、真实旧备份与原生耐久回执样本、独立主机复跑，由实施/验证卡在其范围内产出并留存。这些是该卡的关闭条件；把任命、身份实测、回执样本一类事实级取证留作第一级前置会使门禁自指不可达，故明确排除。

3. **生产解除级**：只有约束 8 全项满足并经人的明确生产批准，才允许解除生产 closed 与「首期不提供」结论，才可作生产可用声明；部署方对具名批准角色的书面任命与两条批准记录原件属本级证据，既不因本修订被接受而视为已具备，也不作为前两级的准入条件。本节四项的决策级判据同时构成约束 8 解除条件的必要组成；解除时须按不变量与解除条件逐项取证，缺一即 closed。

**与 ADR-0030 约束 9(b) 的映射（r09）**：ADR-0030 约束 9(b) 的「决策修订被架构与安全独立评审接受」即本 ADR 第 1 级（决策接受级）；其「约束全项满足并经人的明确生产批准」即第 3 级（生产解除级）。第 2 级（卡证据级）是 ADR-0030 两级中没有独立节点的中间层，只对应约束 8 所需事实证据的产出与留存，不构成任何人的批准，也不改变 ADR-0030 的门禁。
历史接受事实（直接引自报告，不经任务卡转引）：`adr-0031-r06` 于 2026-11-02 经架构与安全两路 r02 独立评审接受（`review-arch-r02` PASS WITH CONDITIONS、`review-sec-r02` PASS WITH CONDITIONS）；`adr-0031-r07` 于 2026-11-02 经架构与安全两路 r01 独立评审接受——架构侧 `.spec/wip/recovery-runtime/evidence-b13j-d08/review-arch-r01/report.md`（9,964 B / SHA256 `EC9361004EF31091DC51F2B3F272264C8DC98544D8C42B2F482D18E5BE656CCB` / PASS WITH CONDITIONS）、安全侧 `.spec/wip/recovery-runtime/evidence-b13j-d08/review-sec-r01/report.md`（18,884 B / SHA256 `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC` / PASS WITH CONDITIONS）。上述接受只绑定各自被评修订号，不构成对现行修订的结论：决策接受级随 `**修订**` 行的现行修订号自动生效，现行修订须重新取得架构与安全两路接受；现行修订的两路接受未取得前，`identity-auth-S4.24a` 及任何实施/验证卡不得启动（启动门禁 blocked），本条不构成对现行修订的评审结论。在第三级达成之前，生产 `enter`/`resume` 的 DB 替换继续 closed，「首期不提供生产 restore 工具与生产批准主体」不解除。

### 批准、执行与隔离不变量

1. **备份来源批准**：`RestorePlan.SourceApprovalRef/SourceApprovalDigest` 只能由部署方具名“恢复批准角色”签发，批准记录须写明角色、人员、时间、备份来源、artifact digest 与 source installation；该角色不得是执行者、本次 run 发起者或工具供应方自批。当前项目未任命该角色，故不存在可核批准记录。
2. **可执行文件批准**：`HAN_RECOVERY_RESTORE_EXECUTABLE` 指向的二进制须由同一具名批准角色以独立批准记录登记绝对路径、SHA256、owner/ACL 与受控只读挂载来源，并与 `HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 精确相等；运行时该文件的 `st_dev`/`st_ino` 必须与批准记录相等，执行必须经固定 fd-exec——对已核 no-follow 稳定 fd 直接执行（`fexecve`、`execveat(AT_EMPTY_PATH)` 或 `/proc/self/fd/N` 一类等价方式），不得校验后按路径重新打开再 exec；上述任一不匹配即 closed；不存在“默认部署方角色”“按目录推断”或“运行时自行选最新版本”。当前无已批准二进制。
3. **执行主体**：替换只能由 root 启动的独立非交互子进程执行，且该子进程须以独立于 root 的专用执行身份运行；root 不代替执行、不进入 shell、不重放，也不把部署锁、Journal/keyring、审批 key、管理入口或除目标连接外的凭据传给子进程。执行身份切换与程序启动禁止经 shell 脚本、`su`、`sudo` 等包装实现（与固定 argv、无 shell 冲突）；具体切换机制由批准记录固定，并要求在实现阶段以隔离实测证明，本 ADR 不选具体机制。
4. **root 与执行身份的权限边界**：root 保留部署锁、排空/独占、Begin/绑定/审计与最终 `VerifyRestored` 判定；执行身份只获得只读 artifact fd、经私有 stdin 传入的目标连接配置、最小 ENV allowlist 与目标 DB 网络。替换窗口外该身份不得写 Journal/keyring/锁/Store 权威绑定或审计；替换窗口内对目标库的写入只能来自已批准整库 artifact 的还原本身，该身份不得自行构造、注入或声明权威 binding，也不得写 Journal/keyring 或部署锁，本次权威 binding 只由 root 经同一 Store 短事务创建/核（design `:3866`）。该身份不得启动 server、requeue、migration、bootstrap 或 worker。
5. **宿主隔离最小机制（POSIX/Windows 矩阵）**：

| 机制 | POSIX（唯一生产候选，首期 closed） | Windows（生产 unsupported） |
|---|---|---|
| 启动方式 | 固定 argv 直接 exec 唯一批准程序；对已核 no-follow 稳定 fd 直接执行（fd-exec），不得校验后按路径重新打开再 exec；无 shell、无附加 argv | 同一软件合同，但仅允许测试替身与软件试验 |
| 进程树 | 独立进程组/会话；父进程持有全部子进程句柄，取消或超时时终止整组并真实 join/reap | 仅以 Job Object 等价的软件试验验证，不构成生产支持 |
| 文件 | no-follow 稳定 fd/handle；校验前后 fstat 与路径身份一致；禁止验后按路径重开 | 同一软件合同，仅测试 |
| 环境 | 最小 ENV allowlist 须为批准记录内固定的封闭字面集（`env_clear` 后逐项设置）；`LD_PRELOAD`/`LD_LIBRARY_PATH`/`LD_AUDIT`/`GLIBC_TUNABLES`、`*_PRELOAD`/`*_OPTIONS` 一类动态加载与注入变量、或 `PATH` 指向任一非批准目录，出现即 closed；子进程实际 environ 与白名单不逐字节相等同样 closed；零继承业务 secret、审批 key 与宿主凭据 | 同一软件合同，仅测试 |
| 平台门禁 | 任一隔离预检失败在 Begin 前拒绝 | 任何 Journal/DB/CAS/文件状态变更前返回 unsupported（退出码 2） |

6. **原生耐久回执**：只能由 restore 工具或其备份系统原生写入、独立于本次进程可重读的持久记录（恢复目录、恢复任务记录或安装来源证据）提供，且须绑定 artifact digest、target 与 source installation；root 必须经独立读取路径与批准记录逐项核对，不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证；读取时必须先验签并留审计记录，验签信任锚（公钥或见证端点）固定登记在批准记录内、轮换须经人的明确批准且不得由被验证回执自身或 root 自选；验证方须同时独立于执行身份与 root——验证方式为外部系统验签，或由作为批准角色的人核对，两种方式都必须满足该独立性合取；验签失败、信任锚不可固定或验证方不独立即 closed。具体产品字段、读取方式与信任锚形态属工具/部署输入，当前 unknown。**r09 补充（外部独立见证与独立性；r10 补齐独立性集合）**：外部锚采用独立见证签名，签名角色按「4. 原生耐久回执」的 producer／verifier／witness 三签模型，不得由单一见证代签；`witness_principal_id`／`witness_key_id` 及其 `org_id` 不得等于 root／executor／producer／issuer／reviewer／**verifier** 任一方，且 `witness_key_id` 与 `verifier_key_id`、`witness_principal_id` 与 `verifier_principal_id` 必须分别不同；`witness_principal_id` 的 `org_id` 必须解析到 `organizations.kind=external_verifier`；六席 `reviewer`／`issuer`／`executor`／`producer`／`verifier`／`witness` 的 `person_ref` 必须两两不等；见证方与部署方／root 必须分属不同管理域（部署前置，不是代码判定）；`baseline_sequence` 只增不减，任何重钉必须由人的明确批准并留证，降低尝试即 closed；验签失败不得推进见证 `sequence`；本地文件、本机进程或服务自身不得冒充外部见证；模拟 witness CLI 仅为测试替身，证据标注「模拟见证、未真实验证」，真实实现为外部审批系统／HSM。

上述六项在当前均为缺失状态，故生产 restore closed；这不削弱既有软件合同与测试替身，也不改变 `restore_unknown` 的保守暂停语义。

### 约束

1. **不改变既有决定与合同**：ADR-0019、ADR-0030、ADR-0027 与 design 既有 restore 软件合同继续有效；本 ADR 只补组织与信任边界。
2. **禁止执行通道扩张**：不得引入 shell、附加 argv、manifest 注入参数、自动重放、自动 fallback，也不得以 file/SQL/fake、环境探测或“目录里最新二进制”填补缺失工具。
3. **禁止伪证明**：caller 自报 done、exit=0、stdout receipt 自洽、DB 可连接、当前行数、自己刚写的 run 行、测试替身或空构造均不得作为恢复证明。
4. **批准与执行分离**：批准角色必须具名且不得自批、不得兼任执行者；执行身份不得等于 root，也不得与批准角色共用凭据。
5. **`restore_unknown` 保留**：无可核持久回执时保持 `restore_unknown`，由人工经原受控备份工具核验/修复后 verify/resume；不得用自动重放或人工勾选 done 越过。
6. **不声称验证**：本 ADR 不证明任何 restore 工具、备份系统、托管平台、执行身份、ACL、进程隔离或旧备份演练已通过；不声称生产装配、Case18、S8.2 或恢复 E2E 通过。
7. **不登记实现任务**：本 ADR 不预占任何未登记 Task 编号；实现与真实演练须由正式 Task 规划登记。
8. **解除条件**：只有同时满足以下条件，才可通过新的 ADR 修订或新 ADR 解除“首期不提供”和生产 closed：
   - 部署方任命具名恢复批准角色，并留存备份来源与可执行文件两条独立批准记录（含角色、人员、时间、依据）；
   - 已批准二进制具备可核 SHA256、owner/ACL、受控只读挂载与更新/回滚责任，运行时 `st_dev`/`st_ino` 必须与批准记录相等，且以固定 fd-exec 对已核 no-follow 稳定 fd 直接执行（不得校验后按路径重新打开再 exec），并保留上述取证；
   - 专用执行身份与权限矩阵经批准：替换窗口外不得写 Journal/keyring/锁/Store 权威绑定与审计；替换窗口内写入只能来自已批准整库 artifact 的还原本身，本次权威 binding 只由 root 写；不得启动 server/requeue/migration/worker；
   - POSIX 宿主隔离实测通过：固定 argv 无 shell、独立进程组与完整子进程树 join/reap、no-follow 稳定 fd、最小 ENV allowlist（批准记录内固定的封闭字面集，`env_clear` 后逐项设置，子进程实际 environ 与白名单逐字节相等）、`LD_PRELOAD`/`LD_LIBRARY_PATH`/`LD_AUDIT`/`GLIBC_TUNABLES`、`*_PRELOAD`/`*_OPTIONS` 一类动态加载与注入变量及 `PATH` 指向任一非批准目录均不出现、目标连接外无网络；
   - 负向验证门禁通过：替换窗口外，以执行身份的凭据或其目标 DB 角色写入 Journal/keyring/部署锁/Store 权威绑定/审计必须被拒绝；替换窗口内，除已批准整库 artifact 的还原本身外不得产生任何 Store 权威绑定或 Journal/keyring/部署锁写入，执行身份自行构造、注入或声明的 binding 必须被拒绝；
   - 取得工具或其备份系统原生写入、可独立重读的耐久回执真实样本，并能与 artifact digest、target、source installation 逐项核对；读取时先验签并留审计记录，验签信任锚固定登记在批准记录内且轮换经人的明确批准，验证方独立于执行身份与 root；见证端点必须是 `https://` 绝对 URL，解析后全部 A／AAAA 非 loopback／link-local／私网／ULA，host 与登记值精确相等，独立 TLS 信任锚或 SPKI pin 固定，query／fragment 不得携带凭据；`witness_principal_id`／`witness_key_id` 与 root／executor／producer／issuer／reviewer／verifier 两两独立，见证方 `organizations.kind=external_verifier` 且与部署方／root 分属不同管理域，六席 `person_ref` 两两不等；`baseline_sequence` 只增不减，重钉信任根须人的明确批准并留证；
   - 以真实旧备份完成 source installation/schema digest/`VerifyRestored` 一致性演练并保留原始证据，不以 synthetic fixture 代替；
   - 通过架构与安全独立评审，完成 ADR 修订/新增决策，并由正式 Task 规划登记实现与验证范围；
   - 取得人的明确生产批准；Agent 产出、评审结论或测试通过均不替代该批准。
9. **不适用场景**：本 ADR 不适用于 ADR-0019/ADR-0030 的 Anchor/keyring 与生产 CAS provider、ADR-0027 的 runtime 编排与正向准入、design §4.2 的 restore 软件协议本身，以及 D04 承接的部署独占、`ArticleAccountPort` 与 recovery secret candidate。Windows 生产仍 unsupported，本 ADR 不提供 Windows 例外。

### 后续阶段前置节点

任何把生产 restore 工具、生产批准主体、专用执行身份或宿主隔离付诸实施的卡片——包括已登记为 `identity-auth-S4.24a` 的实施/装配卡（生产 restore source/target/tool 与批准主体接线）——都不得派发、认领或开始改动（下称启动），除非本 ADR 已有新修订给出批准主体、执行主体、宿主隔离与原生耐久回执的决策级结论，并按约束 8 通过架构与安全独立评审——这里的“通过独立评审”指决策修订被架构与安全接受，而不是“约束 8 全项解除”；该接受只解除实施/验证卡的派发、认领与开始改动限制（登记本身不产出实施效果，不受本门禁约束，见下段），不等于人的决策接纳，也不产生任何人的批准，实施/验证卡用于产出约束 8 所需的隔离实测、负向验证与原生回执证据（这些是该卡的退出条件，不是启动前置），生产解除仍须约束 8 全项与人的明确生产批准。本门禁只约束实施/装配卡，只产出本 ADR 修订的决策卡不受限制；部署独占 POSIX 锁后端与 `exclusive` 接线属候选 S4.24b，受 ADR-0032 门禁，不在本条范围。本 ADR 的 r03 只改候选编号口径与门禁分级，不改变“首期不提供生产 restore 工具与批准主体”结论、候选取舍、不变量与解除条件实质。本 ADR 的 r04 只补与 ADR-0030 约束 9(b) 同形的否定限定与人的接纳限定，不改变本结论、不变量与解除条件实质。本 ADR 的 r06 只把被禁动作收敛为“启动”并按决策接受级／卡证据级／生产解除级三级重述门禁，不改变本结论、不变量与解除条件实质。本 ADR 的 r07 把四项决策级判据回填到不变量 2/5/6 与约束 8 的解除条件第 2/4/6 条、补入回执读取时验签与批准记录内固定信任锚、按两路 r02 的接受事实更新状态段、补 ADR-0030 约束 9(b) 与本 ADR 三级的显式映射，并在 r05/r06 追溯行行末标注失效时点，不改变本结论、候选取舍与解除条件实质。本 ADR 的 r09 同步外部独立见证签名、模拟/真实边界、信任根与端点约束、见证独立性、撤销/TOCTOU 与验签失败处置，并据此**收紧**约束 8 解除条件第 6 条的判据（新增见证端点的 DNS／TLS 与无 query 约束、见证独立性、`baseline_sequence` 单调与重钉须人批准），不是单纯同步；不改变本结论、候选取舍、不变量 1–5、约束 1–7/9 的其余各项。本 ADR 的 r10 把回执签名角色校正为 producer／verifier／witness 三签（对齐 design r66）、定名 `anchor_ref` 五处等式并令 policy 缺键即 closed、给出 `generation` 的可复算三元定义、把撤销与有效期重验补到含 `P6` 的完整检查点集合、无条件登记「不声称抗 root」残余、补入见证术语消歧，并按同一收紧口径把 `verifier` 与六席 `person_ref` 两两不等补入约束 8 解除条件第 6 条。r10 不改变本结论、候选取舍、不变量 1–5 的其余各项、约束 1–7/9 与解除条件第 1–5、7–9 条（除第 6 条外均未改）。本 ADR 的 r11 只按路径修正检查点失败的后果与错误分类、删除 `P3` 的 generation 漂移判定、把重读口径限定为 `P6` 及 `P6` 之后、按席位改写三签公钥来源、把固定 design 修订号引用改为条款锚，并更正本段与追溯行的「未改段落」补集；r11 未改不变量 1–6、约束 1–9（含解除条件九条）的任何字节，也未补齐任何一项 unknown 或改变本结论、候选取舍的实质。本 ADR 的 r12 只按 design 事实修正 `P6`／`Prepare` 的失败分类与 `resume` 调用事实、更正检索断言；未改不变量 1–6、约束 1–9（含解除条件九条）任何字节，未补齐 unknown，未改变结论与候选取舍。

**登记≠启动（r06 统一口径）**：上段已把被禁动作收敛为「派发、认领并开始改动（下称启动）」；本段说明该收敛的理由与边界。理由：把一张卡写入 `tasks.md` 或 Plan（登记）本身不产出 restore 工具、批准主体、执行身份、宿主隔离实测或原生耐久回执，也不改变任何生产行为与任何信任边界，因此不是需要本 ADR 先行的实质动作；把登记一并写进禁令会与既成事实矛盾（已登记的卡无法回退为未登记），并使解释与禁令互相否证。边界：登记不等于授权——登记不放行启动、不改变卡面硬 requires、不解除任何 closed、不构成人的接纳或批准。据此 `identity-auth-S4.24a` 的既有登记不违反上段实质目的：该卡的硬 requires 为 `identity-auth-D06`，在 `adr-0031-r06` 被架构与安全独立评审接受之前它不得启动，未启动即无实施效果。只读调查、比对与准备不产生实施效果，但也不得据此提前选定执行身份切换机制或预置执行环境。本口径不放宽上段限制：决策接受级未达成前任何卡片仍不得启动；生产解除仍须约束 8 全项与人的明确生产批准。r07 补充：上段要求的单条件即三级中的第 1 级（决策接受级）；第 2 级（卡证据级）属实施/验证卡的退出条件事实取证，故不写入启动条件。该评审接受条件随 `**修订**` 行的现行修订号自动生效，不按具体修订号固化——现行修订未获架构与安全两路接受前，`identity-auth-S4.24a` 仍不得启动（见状态段）；本行不记录某一修订号的评审结论，修订号推进后无须回改本行。

### 引用记录

- 2026-11-01 `identity-auth` 设计修订 `identity-auth-design-ir01-r59`（含 D03 评审返工）采用本决定：restore driver 段 `.add/project/knowledge/design-workspace/identity-auth/design.md:3858` 固定“首期不提供生产 restore 工具与批准主体、生产 enter/resume 的 DB 替换在 Begin 前 closed、仅保留软件合同与测试替身”；`:3862` 固定批准记录（含工具/备份供应方不得自批）、专用执行身份、root 保留部署锁/Journal/keyring/Store 绑定与审计写入的权限边界、进程树/ENV allowlist 与 fd 隔离预检（no-follow 稳定 fd 见 `:3870`）及 POSIX/Windows 矩阵，并固定执行身份的两层写入边界——替换窗口外其角色权限不得覆盖 Journal/keyring/部署锁/Store 权威绑定与审计的任何写入，替换窗口内写入只能来自已批准 artifact 的整库还原、不得自行构造或声明权威 binding、替换后由 root 经同一 Store 短事务创建/核本次 binding，受保护对象清单只作 restore 完成后的核对项、不改变整库语义且无 schema 级排除或 root 数据搬移；其目标连接须为独立数据库角色且角色与 GRANT 矩阵由批准记录固定、未证明即 closed，以及除目标连接外无网络与禁止 su/sudo/shell 脚本包装；`:3870` 固定原生耐久回执的来源形态与“信任未证明不得启用生产替换”；配置表 `:3896` 的三个 restore 键语义不变；决策追溯 `:83` 登记本 ADR。设计仍为 draft，人的精确接纳 unknown，不声明实现或验证。
- 硬边界：[framework.md（ADR-0019）](framework.md#adr-0019)，`framework.md:418` 起；本 ADR 不重写其任何已锁定决定。
- 相邻决定：[framework.md（ADR-0030）](framework.md#adr-0030)，`framework.md:916` 起；本 ADR 不改变“首期不提供生产 anchor”。
- 缺口来源：[gap-map.md](../../../../.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md) 的 §2.5 `pending-B13J-D03`（:121-139）与 U4；该文件要求工具、批准者、执行主体、ACL/进程隔离、原生回执与 unknown 暂停均具体。
- 决策授权：用户 2026-11-01 B13J D03 指令；产出者为 Arch-Designer（Agent），不是人的签字、独立评审、安全评审或生产批准。
- 2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）（触发：非作者规划评审 `B13J-PLAN-F004`，报告同上）：新增“后续阶段前置节点”小节，要求任何生产 restore 实现卡（含尚未登记为正式 Task 的 S4.24 一类候选编号）先经本 ADR 新修订给出批准主体、执行主体、宿主隔离与原生耐久回执结论并通过安全独立评审；决定结论、候选取舍、不变量与解除条件实质未变；未新增实现任务，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r03 修订（触发：非作者规划评审 `B13J-PLAN-R02-F002`/`F005`，报告同上）：把“后续阶段前置节点”中的候选编号改为 S4.24a（生产 restore source/target/tool 与批准主体接线），并写明本门禁只约束实施/装配卡、决策卡不受限制；部署独占锁接线归候选 S4.24b，受 ADR-0032 门禁。只改候选编号口径与门禁分级，不改变“首期不提供生产 restore 工具与批准主体”结论、不变量与解除条件实质；未新增实现任务，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r04 收口修订（触发：非作者规划评审 r03 的 `B13J-PLAN-R03-F006`/`F005`/`F003`，报告同上，27104 B / `3FC6A611EE3ACCBCBDBE33AAD77506B7A0AED7158691CF842229669AF201EC19`）：在“后续阶段前置节点”补入与 ADR-0030 约束 9(b) 同形的否定限定——“按约束 8 通过架构与安全独立评审”只指决策修订被接受，不等于“约束 8 全项解除”，评审接受只解除实施/验证卡的登记限制、不等于人的决策接纳、不产生任何人的批准，生产解除仍须约束 8 全项与人的明确生产批准。只补措辞限定，不改变“首期不提供生产 restore 工具与批准主体”结论、候选取舍、不变量与解除条件实质；未新增实现任务，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r05 修订（触发：`identity-auth-D06` 卡目标与 Lead 裁决记录 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r11.md` §E/§H，持久修复安全评审 `B13J-D05-SEC-R06-F005`；授权：用户 2026-11-02 standing 授权「我授权对待确认或决策项采用建议值并继续本项目」）：新增「r05 四项结论（决策级要求；可核结论或明确 unknown）」小节，逐项给出批准主体、执行主体、宿主隔离与原生耐久回执的判定结构、最小字段与否证条件，并把当前无据可核的维度明确记为 unknown 及其补证主体（任命与批准记录、执行身份与切换机制、POSIX 隔离实测、产品级回执字段与样本）；在「后续阶段前置节点」段后新增「登记≠启动」正文口径，说明 `identity-auth-S4.24a` 的既有登记不违反该段实质目的（其硬 requires 为 `identity-auth-D06`，未启动即无实施效果），并声明该口径不放宽原禁令。只新增上述两处并改修订串，不变量 1–6、约束 1–9、解除条件实质与「首期不提供」结论逐字未变；本轮未任命任何批准角色、未实测任何执行身份或进程隔离、未取得任何原生耐久回执；四项 unknown 未补齐前任何实施/验证卡不得启动，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。（本行末句『四项 unknown 未补齐前任何实施/验证卡不得启动』为 r05 时点表述，已由 r06 三级门禁取代，现行口径以 r06/r07 追溯行为准；历史正文按原样保留。）
- 2026-11-02 r06 修订（触发：架构侧独立评审 `review-arch-r01` 的 `B13J-D06-RV-02`——「后续阶段前置节点」段的「都不得登记或启动」句与紧随其后的「登记不构成需要本 ADR 先行的实质动作」解释句字面矛盾（两处改用符号定位，不写数字行锚，避免行号漂移失准）；安全侧独立评审 `review-sec-r01` 的 `F001`（实施/验证卡启动门禁与证据产出主体自指死锁）、`F002`（回执抗伪造）、`F003`（批准记录 `st_dev`/`st_ino`、固定 fd-exec、复核人独立）、`F004`（ENV 动态加载与注入变量否证条件）；报告：`.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r01/report.md`、`.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r01/report.md`；授权：用户 2026-11-02 standing 授权「我授权对待确认或决策项采用建议值并继续本项目」与 Lead 裁决记录 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r11.md` §E）：把「后续阶段前置节点」的被禁动作统一收敛为「派发、认领并开始改动（启动）」，并在「r05 四项结论」收尾按决策接受级／卡证据级／生产解除级三级重述门禁（把事实级取证改为实施/验证卡的退出条件，不再作为启动前置）、并把该小节引言中「四项未全部达到可核…禁令继续有效」的合取表述收敛为以决策接受级为条件（事实级取证不作启动前置）；第 1、3、4 项补 `st_dev`/`st_ino`、固定 fd-exec、复核人独立、ENV 动态加载与注入变量否证条件、回执抗伪造与外部见证要求。只改上述段落与修订串，不变量 1–6、约束 1–9（含约束 8 的九条解除条件）、解除条件实质与「首期不提供生产 restore 工具与生产批准主体」结论逐字未变；本轮未任命任何批准角色、未实测任何执行身份或进程隔离、未取得任何原生耐久回执；`adr-0031-r06` 未经复评，决策接受级尚未达成，任何实施/验证卡仍不得启动，生产 `enter`/`resume` 的 DB 替换继续 closed。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。（本行『未经复评、决策接受级尚未达成』为 r06 产出时点表述；r06 已于 2026-11-02 经两路 r02 接受，现行状态见状态段与 r07 追溯行。）
- 2026-11-02 r07 修订（触发：安全侧独立评审 `review-sec-r02` 的 `B13J-D06-SEC-R02-F001`（四项决策级判据未回填不变量与约束 8 解除条件）、`B13J-D06-SEC-R02-F002`（回执抗伪造为弃取式且缺读取时验签与信任锚固定）、`B13J-D06-SEC-R02-F003`（状态段时点陈述在 r06 被接受后失实）；架构侧独立评审 `review-arch-r02` 的 `B13J-D06-R02-F003`（同前项时点陈述）与 `B13J-D06-R02-F004`（两级与三级门禁缺显式映射）；报告：`.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r02/report.md`、`.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r02/report.md`；授权：用户 2026-11-02 standing 授权「我授权对待确认或决策项采用建议值并继续本项目」与 `identity-auth-D08` 卡）：改动限于 ADR-0031 段——(i) 修订行升为 `adr-0031-r07`；(ii) 不变量 2 与解除条件第 2 条补运行时 `st_dev`/`st_ino` 相等与固定 fd-exec（对已核 no-follow 稳定 fd 直接执行，不得校验后按路径重开）；(iii) 不变量 5 表格「启动方式」「环境」两行与解除条件第 4 条补 ENV 封闭字面集、动态加载与注入变量 denylist、子进程 environ 与白名单逐字节相等；(iv) 第 4 项结论、不变量 6 与解除条件第 6 条把回执抗伪造改为合取，并补读取时验签、批准记录内固定信任锚、验证方独立于执行身份与 root、审计留痕与 fail-closed 否证条件，同时登记「无法提供独立验证方与固定信任锚时残余风险止于 root 且保持 closed」；(v) 状态段按「r06 已被两路 r02 接受、r07 尚未复评」的事实改写，并写明 r07 未获两路接受前 `identity-auth-S4.24a` 不得启动；(vi) 新增 ADR-0030 约束 9(b) 与本 ADR 三级的显式映射，并在「登记≠启动」段说明上段单条件即第 1 级、第 2 级不写入启动条件；(vii)「后续阶段前置节点」段追加 r07 范围句；(viii) r05/r06 追溯行行末追加失效时点标注并新增本行。未改段落：不变量 1/3/4、约束 1–9 其余各项（含解除条件第 1/3/5/7/8/9 条）、「首期不提供生产 restore 工具与生产批准主体」结论与候选取舍，以及 ADR-0030、ADR-0032、ADR-0019、ADR-0021、ADR-0027、ADR-0029 全段。已知残余（禁改区）：ADR-0030 OPEN 清单中「`adr-0031-r05` 承接」字样（安全 `B13J-D06-SEC-R02-F005`）属本卡禁改区，未改并继续登记。本轮未任命任何批准角色、未实测任何执行身份或进程隔离、未取得任何原生耐久回执，四项 unknown 仍为 unknown；`adr-0031-r07` 尚未复评，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r09 修订（触发：用户 2026-11-02 追加任务，要求在 r07 之后补入外部独立见证签名与撤销／TOCTOU 同步；承接基线：`adr-0031-r07`；`adr-0031-r08` 未产出，跳号至 r09 的事实由本行定稿，不再保留「待评审方确认」；授权：用户 2026-11-02 追加任务指令）：改动限于 ADR-0031 段，逐项锚点为 (i) 修订行 `:1082`（当时升为 `adr-0031-r09`，现行已由 r10 升为 `adr-0031-r10`）；(ii) 第 4 项结论内 `:1118` 的「r09 外部锚补充…」与「r09 撤销与 TOCTOU 同步…」两段；(iii) 不变量 6 `:1147` 的「r09 补充…」段；(iv) 约束 8 解除条件第 6 条 `:1166` 的见证端点与独立性判据；(v)「后续阶段前置节点」段 `:1174` 的 r09 范围句；(vi)「登记≠启动」段 `:1176` 的现行修订表述；原附在 r07 追溯行 `:1190` 行末的临时 r09 句由本行取代。**显式接受限制（本行登记，不声称已消除）**：HEAD 版 `framework.md` 不含 ADR-0031，该段在 `adr-0031-r07` 之后均为工作树未提交内容，故 `r07→r09` 没有可独立复算的提交基线；r07 前像的字节未留存于本卡：`tasks.md:3148` 把 r07 时点全文记为 268,790 B／1,271 行／SHA256 `70CFC915DBFBC27C62B078556C3F424BEA41F3F2667950A3F4A562B1FEC8AFCA`，但该值是任务卡转述、本卡无对应前像字节，不能据以复算 r07→r09 的 diff，本行不以「未改」宣告代替 diff 证据。上述限制使 r09 的变更集只能按本行锚点逐段文本核对，不能按前像哈希复算；两路 r03 评审据此判定 r09 不可独立审计，处置见 r10 行。r09 产出后的全文基线为 272,588 B／1,271 行／SHA256 `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`（即 r10 改前实测值）。本修订未改「首期不提供生产 restore 工具与生产批准主体」、候选取舍、不变量 1–5、约束 1–7／9 与解除条件第 1／3／5／7／8／9 条。r09 未取得两路接受，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r10 修订（触发：架构侧独立评审 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r03/report.md`（30,450 B / SHA256 `15C193D62217ADD1E71D621805F1F88E1E1A84DD0DA8BFA295143D0C3B3FAF5B` / CHANGES REQUIRED，BLOCKER 2／HIGH 2／MEDIUM 3／LOW 2）的 F001／F002／F004／F005／F006／F007／F008／F009，与安全侧独立评审 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r03/report.md`（22,195 B / SHA256 `501B3102D61A74CBA775961ACC8A7C7ED2EA1254BDB7091C6C1CE2F68D180F9F` / CHANGES REQUIRED，HIGH 3／MEDIUM 3／LOW 1／CLOSED 5）的 F001／F002／F003／F004／F005／F007；两路均判定不构成 `adr-0031-r09` 的接受；授权：本轮 Lead 直接指令「产出 `adr-0031-r10`」——同链台账 `.spec/wip/recovery-runtime/evidence-b13j-s424a/lead-authorization-r12.md` 覆盖 design r66 与两路 r03 评审，其 §3 明示不含 `framework.md` 写入，故本行写入依据是上级直接指令，该台账只作链路引用）：改动限于 ADR-0031 段，逐项为 (i) 修订行 `:1082` 升为 `adr-0031-r10`；(ii) 状态段 `:1084` 与历史接受段 `:1129` 改为「随现行修订号自动生效」并直接引用 `evidence-b13j-d08/review-arch-r01`（9,964 B / `EC9361004EF31091DC51F2B3F272264C8DC98544D8C42B2F482D18E5BE656CCB` / PASS WITH CONDITIONS）与 `review-sec-r01`（18,884 B / `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC` / PASS WITH CONDITIONS）作为 r07 接受事实；(iii) 第 4 项结论 `:1118` 把回执签名角色校正为 producer／verifier／witness 三签（对齐 design `identity-auth-design-ir01-r66`：producer 签回执信封、verifier 签 8 字段载荷、witness 签 5 键 head，三把 key 的登记并存于 `receipt_anchors` 与 `keys`），并删除「`witness_key_id` 是公钥唯一来源」；(iv) 同段 `:1118` 定名 `anchor_ref` 五处等式为 `receipt_anchors.anchor_ref`＝policy `tool.receipt.trust_anchor_ref`＝回执 `trust_anchor_ref`＝审计记录 `anchor_ref`＝见证 head `anchor_ref`，policy 缺键即 closed；(v) 同段 `:1118` 给出 `generation` 的可复算三元定义（`approval_generation`／`trust_root_generation`／`anchor_generation`），并把撤销与有效期重验补到含 `P6` 的完整检查点集合，写明重验失败即 closed／`restore_unknown` 且不得推进 `sequence`；(vi) 同段 `:1118` 无条件登记「回执抗伪造止于 root／部署方、本 ADR 不声称抗 root」，删除条件式弱化；(vii) 同段 `:1118` 新增见证术语消歧（ADR-0030 `(w, r, token)` anchor revision 见证 vs 本 ADR 回执外部锚；`anchor_ref` 不是 ADR-0019 `JournalHeadAnchor`）；(viii) 不变量 6 `:1147` 与约束 8 解除条件第 6 条 `:1166` 把 `verifier_principal_id`／`verifier_key_id` 补入独立性集合，并写明 witness 的 `organizations.kind=external_verifier` 与六席 `person_ref` 两两不等；(ix)「后续阶段前置节点」段 `:1174` 把 r09 的「只同步」改为「收紧」并指明收紧的是约束 8 解除条件第 6 条，追加 r10 范围句；(x)「登记≠启动」段 `:1176` 改为随现行修订号自动生效；(xi) 新增独立 r09 追溯行 `:1191`、删除 `:1190` 行末临时 r09 句、新增本行 `:1192`。**未改段落**：第 1／2／3 项结论、四项 unknown、不变量 1／2／3／4／5 的其余各项、约束 1–7／9、解除条件第 1／3／5／7／8／9 条、「首期不提供生产 restore 工具与生产批准主体」结论与候选取舍，以及 ADR-0019／ADR-0021／ADR-0027／ADR-0029／ADR-0030／ADR-0032 全段。**改前基线**：272,588 B／1,271 行／SHA256 `D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`（CRLF 1,271、bare LF 0、无 BOM、末尾 `0D 0A`）；改后 B／行／SHA256 与段外不变证明见同批产出 `.spec/wip/recovery-runtime/evidence-b13j-d06/adr-0031-r10/hashes.txt`（本行不自指改后值）。本轮未任命任何批准角色、未实测任何执行身份或进程隔离、未取得任何原生耐久回执，四项 unknown 仍为 unknown；`adr-0031-r10` 未取得两路接受，不构成评审 PASS、人的接纳或生产批准。（r11 更正标记：本行为 r10 时点记录，其余表述不改写；其「未改段落」补集漏列解除条件第 2／4 条，实际为「除第 6 条外，解除条件第 1–5、7–9 条未改」，最终口径与全部更正见下一条 r11 追溯行。）
- 2026-11-02 r11 修订（触发：架构侧独立评审 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r04/report.md`（22,635 B / SHA256 `F17AC89A9970BC89FF5B96625531DD48BBA6ECA5ECB45B695D790465613A3DBC` / CHANGES REQUIRED，BLOCKER 0／MEDIUM 1／LOW 3）的 `ADR-R04-01`（MEDIUM）与 `ADR-R04-02`／`ADR-R04-03`／`ADR-R04-04`（LOW）；安全侧独立评审 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r04/report.md`（24,983 B / SHA256 `C1AE35DF51F9D1CFAA14AA5B2E39EC33F9DD80CD428D507F0CE316FE1DABB9CE` / PASS，0 BLOCKER／0 MEDIUM／1 LOW，该 LOW 属 design 侧、不在本轮范围）；授权：本轮 Lead 直接指令「产出 `adr-0031-r11`」）：改动限于 ADR-0031 段，逐项为 (i) 修订行 `:1082` 升为 `adr-0031-r11`；(ii) 状态段 `:1084` 由 r10 写法同步为 r11 修订号，用户接受事实保留、仅两路评审状态随修订号更新，写明 `adr-0031-r11` 的两路接受尚未取得、`identity-auth-S4.24a` 及任何实施/验证卡不得启动；(iii) 第 4 项结论 `:1118` 把检查点失败后果按路径分写：`P6` 及 `Prepare` 内重读在 `Journal.Begin` 之前失败返回 `ErrAdmissionClosed` 且不进入 `Journal.Begin`，`R1`–`R8`（`Replace` 路径）与 `V1`–`V11`（`VerifyRestored` 路径）在 `Journal.Begin` 之后按各自路径既有分类处置——缺键、文件缺失或不可读按 `ErrAdmissionIncomplete`，未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked` 及工具未启动前的 generation／字节漂移按 `ErrAdmissionClosed`，越过工具启动点后按 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)` 并由调用方按 `restore_unknown` 处置，删除把 R/V 检查失败写成 Begin 前拒绝的写法（`ADR-R04-01`）；(iv) 同段删除 `P3` 的 generation 漂移判定（`P3` 只构造不可变快照与首次合取），把重读限定为「`P6` 及 `P6` 之后每个检查点」（`ADR-R04-01`）；(v) 同段把三签公钥来源句改为按席位表述：三席位身份／密钥标识登记在 `receipt_anchors`，公钥分别取自 `receipt_anchors.public_key`、`keys[verifier_key_id].public_key`、`receipt_anchors.witness_public_key`（`ADR-R04-03`）；(vi) 同段把固定引用 `identity-auth-design-ir01-r66` 的六处改为 design 条款锚（`r64.5`／`r64.10`／`r65.5`／`r65.13`／`r66.2`／`r66.5`），并注明现行 `identity-auth-design-ir01-r70` 未改这些合同、检索 `r68.`／`r69.`／`r70.` 未命中任何编号条款（`ADR-R04-04`）；(vii)「后续阶段前置节点」段 `:1174` 把 r10 范围句的补集更正为「除第 6 条外，解除条件第 1–5、7–9 条未改」并追加 r11 范围句（`ADR-R04-02`）；(viii) r10 追溯行 `:1192` 仅行末追加 r11 更正标记、未改写历史事实（`ADR-R04-02`）；(ix) 新增本行。**未改段落**：不变量 1–6、约束 1–9（含约束 8 解除条件九条）的判据、四项 unknown、「首期不提供生产 restore 工具与生产批准主体」结论与候选取舍、`anchor_ref` 五处等式与 policy 必填 `tool.receipt.trust_anchor_ref`、三签模型、`generation` 三元、无条件「不声称抗 root」、两种「见证」分域、`restore-v1`／`verify-v1`／no-follow／fd-exec／最小 ENV／Begin 前 closed／`restore_unknown`／Windows unsupported，以及 ADR-0019／ADR-0021／ADR-0027／ADR-0029／ADR-0030／ADR-0032 全段。**上游残留观察项（不在本写集）**：design `:4070`（r65.13 的 B1 摘要）仍写「`P3`／`P6`／`R1`／`V1`／`R2`／`V2`／`R4`／`V4` 漂移 → `ErrAdmissionClosed`，并附「不进入 Journal.Begin」字样」，与本 ADR r11 及 design `:4037`／`:4373`／`:4377` 冲突；该行自述「B1 只作摘要、非穷举」并指定完整集合以 `:4037`／`:4062`／`:4135`／`:4137` 为准，属 design 写集，本轮未改。**改前基线**：285,639 B／1,273 行／SHA256 `B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`（CRLF 1,273、bare LF 0、无 BOM、末尾 `0D 0A`）；改后 B／行／SHA256 与段外不变证明见同批产出 `.spec/wip/recovery-runtime/evidence-b13j-d06/adr-0031-r11/hashes.txt`（本行不自指改后值）。本轮未任命任何批准角色、未实测任何执行身份或进程隔离、未取得任何原生耐久回执，四项 unknown 仍为 unknown；`adr-0031-r11` 未取得两路接受，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r12 修订（触发：架构侧独立评审 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r05/report.md`（27,698 B / SHA256 `98C3DD6F6670BB5F875B80AD084C9A0130F4956220206C38AE0B8DD938271FB7` / CHANGES REQUIRED，BLOCKER 0／MEDIUM 1／LOW 2）的 `ADR-R05-01`（MEDIUM）与 `ADR-R05-02`／`ADR-R05-03`（LOW）；安全侧独立评审 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r05/report.md`（24,558 B / SHA256 `40F5C45DDCC77A1E786B9B1E1E64E03F7EA8DB1519CB22188E695658EB8DBF12` / PASS，0 BLOCKER／0 MEDIUM／1 LOW，`ADR-S05-01` 与 `ADR-R05-02` 同源）；授权：本轮 Lead 直接指令「把 ADR-0031 由 adr-0031-r11 修订为 adr-0031-r12」）：改动限于 ADR-0031 段，逐项为 (i) 修订行 `:1082` 升为 `adr-0031-r12`；(ii) 状态段 `:1084` 保持「已接受（2026-11-01 用户 D03 指令授权…）」结构与全部既有文字，仅把 `adr-0031-r11` 字样与「本修订 `adr-0031-r11` 的两路接受尚未取得」改为 r12；(iii) 第 4 项结论 `:1118` 把重验失败后果整段替换为按路径与原因分写的版本：`P6`／`Prepare` 按 design `:4377` 的原因分类，缺键、文件缺失或不可读、非绝对规范路径、owner／mode 不符、非 regular 或被符号链接替换等返回 `ErrAdmissionIncomplete`，未到 `valid_from`、已过 `valid_until`、`revocation.state=revoked` 及预检后字节／`generation` 漂移返回 `ErrAdmissionClosed`；明确 `enter` 的 `P6` 失败不进入 `Journal.Begin`、`resume` 不新 `Journal.Begin`；`R1`–`R8` 仅在 `enter` 的 `Replace` 路径且位于 `Journal.Begin` 之后，`V1`–`V11` 在 `enter`／`resume` 均会调用、`resume` 的调用在 `Prepare` 后且无新 `Journal.Begin`，`resume` 不调用 `Replace`（`ADR-R05-01`、`ADR-R05-02`、`ADR-S05-01`）；(iv) 同段把检索断言更正为 `:4072` 列出 `r68.1`–`r68.8`／`r69.1`–`r69.3`／`r70.1`–`r70.3` 的修订号与插入前行号对应、不构成合同条款，并与 `:4083` 的「r70 修订沿用本声明」并列（`ADR-R05-03`）；(v)「后续阶段前置节点」段 `:1174` 在 r11 范围句后追加 r12 范围句，仅按 design 事实修正 `P6`／`Prepare` 失败分类与 `resume` 调用事实、更正检索断言，未改不变量 1–6、约束 1–9（含解除条件九条）任何字节，未补齐 unknown，未改变结论与候选取舍；(vi) 新增本行。**未改段落**：不变量 1–6、约束 1–9（含约束 8 解除条件九条）的判据、四项 unknown、「首期不提供生产 restore 工具与生产批准主体」结论与候选取舍、`anchor_ref` 五处等式与 policy 必填 `tool.receipt.trust_anchor_ref`、三签模型、`generation` 三元、无条件「不声称抗 root」、两种「见证」分域、`restore-v1`／`verify-v1`／no-follow／fd-exec／最小 ENV／Begin 前 closed／`restore_unknown`／Windows unsupported，以及 ADR-0019／ADR-0021／ADR-0027／ADR-0029／ADR-0030／ADR-0032 全段；r11 历史行 `:1193` 保持不动。**改前基线**：292,606 B／1,274 行／SHA256 `9723F92FB53B076D228B5FA392CEC07CA0F74ED460524EBB7F00AC9202EB53CD`（CRLF 1,274、bare LF 0、无 BOM、末尾 `0D 0A`）；改后 B／行／SHA256 与段外不变证明见同批产出 `.spec/wip/recovery-runtime/evidence-b13j-d06/adr-0031-r12/hashes.txt`（本行不自指改后值）。本轮未任命任何批准角色、未实测任何执行身份或进程隔离、未取得任何原生耐久回执，四项 unknown 仍为 unknown；`adr-0031-r12` 未取得架构与安全两路独立评审接受，不构成评审 PASS、人的接纳、生产批准或安全评审，生产 `enter`／`resume` 的数据库替换保持关闭，`identity-auth-S4.24a` 及任何实施/验证卡不得启动。

---

<a id=adr-0032></a>
## ADR-0032: 部署独占锁采用 POSIX 独占文件锁，Windows 生产 unsupported (2026-11-01)

**目的**: 架构设计 | **仓库**: han | **修订**: adr-0032-r05

**状态**: 已接受（2026-11-01 用户 standing 授权采用建议值；确认来源为本轮 D04 任务指令；不构成人的接收、独立评审或生产批准） | **确认来源**: 用户 2026-11-01 在 B13J D04 任务指令中授权 Arch-Designer 为唯一 production root 的部署独占设施确定唯一生产来源与平台矩阵，并要求明确“单机 OS 文件锁及 POSIX/Windows 支持矩阵，或明确首期不支持 Windows 生产 enter”（gap-map U5）。该来源授权的是架构决策与文档写入，不是生产装配、ACL/文件锁实测、安全评审、独立评审或人的成果签收。

**继承关系**: 只固定 `identity-auth/design.md` §4.2 配置表 `HAN_RECOVERY_RUNTIME_LOCK_PATH`（r60 `:3908`）与新增平台矩阵段（r60 `:3922`）中的部署独占后端选型、平台支持矩阵、锁路径/ACL/持有语义，以及 11 端口清单中 `exclusive` 行（r60 `:3616`）的归属与状态：该行唯一生产来源是已登记 anchor adapter bundle 四端口集合中的 `recovery.DeploymentExclusive`（bundle 四项缺一即 `ErrAdmissionBinding`；空、未登记或配置路径不可用即四项一起落 `recoveryClosedPort`），`recoveryRootInputs.exclusive` 只是合成/测试注入、生产不得用它替代 registry 来源，`restoreDeploymentExclusive` 只是 root 侧唯一适配器形态且当前无生产构造点，故当前生产 `exclusive` 只能是 `recoveryClosedPort`；不改变 ADR-0019 已锁定的 `JournalHeadAnchor` 独立故障域、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒与 fail-closed 失败分类；不改 ADR-0030 的“首期不提供生产 CAS Anchor provider、S4.5 未交付、生产 `enter`/`install` 保持 closed”；不改 ADR-0031 的“首期不提供生产 restore 工具与批准主体、生产 `enter`/`resume` 的 DB 替换保持 closed”及其批准记录、专用执行身份、宿主隔离与原生耐久回执不变量；也不改变 ADR-0027 的恢复 runtime 编排与正向准入证明。本 ADR 与整库 restore 语义不冲突：部署独占锁在 restore 之前取得、覆盖三个入口，不参与、不放宽也不替代整库 artifact 还原的写入语义，也不引入 schema 级排除或 root 数据搬移。

**问题**: design 已要求“部署锁由唯一 root 先取得、再进入任何证明/迁移/受控动作”，并要求 `lease.Check` 读取实际本地独占设施状态、不能以 caller bool/no-op 实现；但 `cmd/han-server/recovery_exclusive.go` 当前只有 71 行的 `restoreDeploymentExclusive`，必须由 root 显式注入 acquire，否则返回 `ErrAdmissionUnsupported`，没有任何 OS 后端或 fallback，路径校验只覆盖绝对路径、`filepath.Clean` 相等与无前后空白；`recovery_config.go` 只解析 `HAN_RECOVERY_RUNTIME_LOCK_PATH`，并拒绝 `journalPath == lockPath`。部署独占因此处于“要求明确、设施 unknown”的状态：normal/recovery/requeue 是否共用一把锁、进程崩溃后如何判定锁已释放、锁文件被删除或被替换时如何 fail closed、Windows 如何拒绝，均无唯一口径。

**决策**: POSIX 生产采用单机 OS 级独占非阻塞文件锁：锁 fd 以 `O_CLOEXEC` 打开并在进程存续期常驻，进程死亡由内核释放；锁路径为恢复域外绝对规范路径、owner-only、no-follow 的单一文件；normal/recovery/requeue 三入口在同一部署共用同一把锁；文件存在不等于持有锁，解锁不靠删除文件；设施缺失、ACL 不可验证、路径不合法或获取失败一律在任何 DB/CAS/文件状态变更前拒绝启动（退出码 2）。Windows 生产 unsupported：配置或平台不支持一律退出码 2，无 fallback、无降级，仅保留软件合同与确定性测试替身。首期不引入 `O_EXCL` 锁文件、DB advisory lock、进程内 mutex、caller bool、no-op 或 fake 作为生产锁。

### 现状 / 痛点 / 改造 / 效果

现状是“要求已写入、设施未定”：design 已把部署独占写成 recovery runtime 的构造门禁，`HAN_RECOVERY_RUNTIME_LOCK_PATH` 也已在配置表定义；但代码侧只有 closed 占位，没有 OS 后端、平台矩阵、持有/释放语义，也没有真实双进程互斥、崩溃释放或 ACL 证据。此时任何“锁已取得”的声明都只能来自 caller 自报或不存在的设施。

痛点在于：若用“锁文件存在即已锁”（`O_EXCL`）实现，进程崩溃会留下陈旧文件，正常退出与崩溃不可区分，删除文件又与解锁耦合，第二个进程可能在删除竞态中同时进入；若用 DB advisory lock，锁依附于正在被恢复或替换的同一个 PostgreSQL，恢复开始前连接可能先失效，且与 ADR-0019 不把故障域回退到 PostgreSQL 的边界冲突；若用进程内 mutex，只覆盖单进程，无法约束第二个进程；若“假定单实例、不加锁”，任何运维误启或重启重叠都会造成双写。

| 方案 | 实现机制 | 优势 | 劣势与未满足条件 | 结论 |
|---|---|---|---|---|
| A. POSIX 独占非阻塞 OS 文件锁（选定） | 以 `O_CLOEXEC` 打开锁文件并取 OS 级独占锁，fd 常驻至进程退出；获取失败立即拒绝，不阻塞等待 | 内核在进程死亡时自动释放，无陈旧状态歧义；解锁不依赖删除文件；可在任何 DB/CAS 操作前取得；天然跨进程 | 仅覆盖单机；须以 owner-only/no-follow 路径与 ACL 验证保证不可替换；具体 syscall（如 `flock`/`fcntl`）与错误映射须在实现阶段固定并以实测证明 | 选定 |
| B. `O_EXCL` 锁文件创建 | 以 `O_EXCL` 创建文件即视为持锁，删除文件即解锁 | 实现简单，跨进程 | 崩溃后留下陈旧文件，文件存在不等于持有；删除与解锁耦合，存在删除竞态；无法区分“陈旧”与“在用” | 不选 |
| C. PostgreSQL advisory lock | 在业务 PostgreSQL 上取 session advisory lock | 复用既有连接与运维视图 | 锁依赖正在被恢复/替换的数据库；连接失效即失锁，无法覆盖 DB 不可用阶段；与 ADR-0019 不把故障域回退到 PostgreSQL 冲突 | 不选 |
| D. 进程内 mutex / 单例 | 进程内互斥或包级单例 | 无外部设施，零配置 | 只覆盖单进程，不能阻止第二个进程；极易退化为 caller bool/no-op | 不选 |
| E. 无锁（部署约定单实例） | 不检查任何设施，依赖运维约定 | 零实现成本 | 任何误启、滚动重启或人工重跑都会双写；违反 `lease.Check` 读取真实独占状态与 fail-closed 要求 | 不选 |

选择 A：它是唯一在崩溃后不存在“陈旧状态歧义”、且不依赖正在被恢复的 PostgreSQL 的机制；内核释放让“进程死亡即释放”成为可核事实，而不是额外清理步骤；持锁失败可稳定映射为启动拒绝，符合既有 fail-closed 与“文件存在不等于持有锁”的要求。B、C、D、E 都会把不存在的互斥写成已取得，或在恢复窗口内引入新的不可判定状态，故不采用。

改造是把部署独占从实现细节提升为生产构造门禁：root 在构造 recovery runtime 之后、任何证明/迁移/Begin/restore 之前取得锁；normal/recovery/requeue 三个入口共用同一把锁，不允许按入口分别取锁；设施、ACL、路径或获取任一不满足就在任何 DB/CAS/文件状态变更前返回退出码 2，不做重试后继续、不做降级、不做 fallback。

效果是：生产 root 不会因锁文件存在、目录可写或 caller 自报而误开独占；崩溃后重启不会卡在陈旧锁；恢复窗口内的互斥不依赖正被替换的数据库；Windows 生产按 ADR-0019 明确 unsupported 而非“尽量试试”；部署独占的具体 OS 后端、隔离与 ACL 证据留给实现阶段以实测关闭，而不是写进本 ADR。

### 不变量与约束

1. **平台矩阵**：POSIX 是唯一生产候选；Windows 生产 unsupported，配置或平台不支持一律在任何 Journal/DB/CAS/文件状态变更前返回退出码 2，无 fallback、无降级。本 ADR 不扩 ADR-0026（identity.md）的 Windows SID 矩阵，也不扩 FileRecoveryJournal 的 POSIX 支持矩阵，不提供 Windows 生产例外。锁文件必须位于已验证支持可靠 OS 独占锁语义的本地 POSIX 文件系统；共享/NFS 类在锁语义未证明前不宣称支持（与 ADR-0019 生产 `FileRecoveryJournal` 的本地文件系统前提同口径，不扩其支持矩阵）。
2. **三入口共用一把锁**：`normal`、`recovery`、`requeue` 在同一部署内共用同一锁路径与同一持有期；锁必须在进入任何证明/迁移/Begin/restore 之前取得，并覆盖整个 recovery runtime 生命周期；不得按入口分别取锁、不得把锁缩到单个 handler。
3. **路径与 ACL**：锁路径必须是恢复域外的绝对规范路径，`filepath.Clean(path) == path` 且无前后空白；owner-only、no-follow、单一文件；不得与 `HAN_RECOVERY_JOURNAL_PATH` 相同，不得位于 Journal/keyring/安装目录内，不得是符号链接、目录或可被同组/其他用户替换的位置。设施或 ACL 不可验证即拒绝启动。
4. **持有与释放**：锁 fd 以 `O_CLOEXEC` 打开并常驻至进程退出；进程死亡由内核释放；解锁不靠删除文件，也不靠 `stat` 存在性检查；文件存在不等于持有锁。
5. **禁止伪实现**：`lease.Check` 必须读取实际本地独占设施状态；禁止 caller bool、no-op、fake、环境变量自报、时间戳心跳或“假定单实例”作为生产锁。测试替身只允许出现在测试与软件试验。
6. **失败分类**：设施缺失、ACL 不可验证、路径不合法、锁获取失败一律 fail closed 并以退出码 2 拒绝启动；不得重试后继续、不得降级为无锁、不得自动 fallback 到其他后端。
7. **不改变既有决定与合同**：ADR-0019、ADR-0030、ADR-0031、ADR-0027 与 design 既有 restore/anchor/keyring 合同继续有效；本 ADR 只补部署独占后端与平台矩阵。
8. **不登记实现任务、不声称验证**：本 ADR 不预占任何未登记 Task 编号；不证明任何 OS 后端、ACL、双进程互斥、崩溃释放或三入口共用锁已实现或已通过测试；不声称生产装配、Case18、S8.2 或 E2E 通过。
9. **解除顺序与来源耦合**：本 ADR 不隐去 `exclusive` 的生产来源依赖——要获得部署锁必须先有已登记的 anchor adapter bundle 交付（ADR-0030 首期 registry 恒空、生产 CAS provider 未提供），再实现本 ADR 的 POSIX 锁后端并取得隔离/ACL 证据，最后由正式 Task 规划登记装配与验证；该顺序完成前 `exclusive` 的生产装配只能是 `recoveryClosedPort`，`normal`/`recovery`/`requeue` 一律 closed，不得以合成/测试注入或 registry 之外的路径替代唯一来源。

### 解除条件

1. POSIX 后端按本 ADR 实现：真实 OS 级独占非阻塞锁、`O_CLOEXEC` 常驻 fd、无 fallback，并在代码中保留“设施不满足即退出码 2”的路径；
2. 取得隔离与 ACL 证据：owner-only、no-follow、单一文件、路径规范校验、`journalPath != lockPath`，以及符号链接/目录/可替换路径被拒绝；
3. 取得负向测试证据：第二个进程以同一路径启动必须被拒；持锁进程崩溃后新进程可取得锁；删除锁文件不解除持有；Windows 平台在任何副作用前返回退出码 2；
4. 取得三入口证据：`normal`/`recovery`/`requeue` 在同一部署共用同一把锁，任一入口未持锁即拒绝；
5. 通过架构与安全独立评审，完成 ADR 修订或新增决策，并由正式 Task 规划登记实现与验证范围；
6. 取得人的明确生产批准；Agent 产出、评审结论或测试通过均不替代该批准；
7. Windows 生产若要解除 unsupported，必须另立 ADR，给出等价的原生独占句柄语义、崩溃释放、ACL 与负向测试证据；本 ADR 不提供该例外。
8. **解除顺序（前置链，须逐项可核且无环）**：条件 1–7 任一未满足即不得解除；条件 1–4 即使全部满足，`exclusive` 仍未按 registry 唯一来源接线时也不得判定解除。解除必须按下列顺序逐节点留证：
   - ADR-0030 按其约束 9 完成决策修订（生产 CAS provider、独立故障域、部署/ACL 结论）并被架构与安全独立评审接受（决策级）；生产解除仍按约束 7；
   - 候选 S4.23（anchor adapter bundle 四端口齐备并登记；已登记为正式 Task `identity-auth-S4.23`）：`Exclusive` 作为已登记但 closed 的端口接入 bundle，不得填实现、不得脱离 closed 端口状态；
   - 候选 S4.24b（部署独占 POSIX 锁后端与 `exclusive` 接线，受本 ADR 门禁）**requires 候选 S4.23**：候选 S4.23 交付且 bundle 四端口齐备（`Exclusive` 仍可为 closed）即可启动本节点；bundle 必须先于锁后端接线，不得先接线后补来源；`Exclusive` 离开 `recoveryClosedPort` 由本节点交付完成；本节点同时交付并取证本 ADR 解除条件 1–3 的 POSIX 锁后端、隔离/ACL 证据与负向测试证据，该证据不再是独立前置节点，也不得早于本节点登记；
   - 候选 S4.24a（生产 restore source/target/tool 与批准主体接线，受 ADR-0031 门禁）与本 ADR 的锁后端节点相互独立，两者均须早于候选 S4.22 的生产解除；其与 S4.22 root 装配的先后按 ADR-0031 单独登记；
   - 候选 S4.22（production root 装配与 recovery secret candidate）**requires 候选 S4.24b**；其与候选 S4.24a 的关系按 ADR-0031 单独登记（root 装配可在 restore 保持 closed 时进行），不并入本 ADR 的硬依赖边；
   - 解除条件 5–6 的正式 Task 规划登记与人的生产批准。
   该链是有向无环图：决策级修订、实施/验证、真实验证、生产解除逐级单向，每个节点只依赖其直接前驱，无回边，且 ADR-0030 约束 9 已给出“决策卡可在任何时点登记、决策修订接受后即可登记实施/验证卡”的出边，因此不是架构死锁；当前不可满足的只是旧口径下的规划登记顺序（把“约束 7 全项解除”误读为实施卡登记前置），不是架构不可达，也不改变本 ADR 的 POSIX-only、Windows 生产 unsupported（退出码 2）与无 fallback 结论。

### 不适用场景

本 ADR 不适用于：ADR-0019/ADR-0030 的 Anchor/keyring/CAS provider 选型与故障域证明；ADR-0031 的 restore 工具、批准主体、执行身份与宿主隔离；ADR-0027 的 runtime 编排与正向准入；`ArticleAccountPort` 的 canonical orphaning 归属（由 article 设计承接）；recovery secret candidate 的来源与验密语义；多节点或分布式部署的跨主机互斥（本 ADR 只定义单机本地独占，跨主机互斥须另立决策）。Windows 生产仍 unsupported，本 ADR 不提供任何 Windows 生产承诺。

### 引用记录

- 2026-11-01 `identity-auth` 设计修订 `identity-auth-design-ir01-r60` 采用本决定：配置表 `HAN_RECOVERY_RUNTIME_LOCK_PATH` 行 `.add/project/knowledge/design-workspace/identity-auth/design.md:3908` 固定 POSIX 独占非阻塞 OS 文件锁、`O_CLOEXEC` 常驻 fd、非 `O_EXCL`/advisory lock/mutex/caller bool、恢复域外 owner-only/no-follow 单一文件且位于已验证可靠 OS 独占锁语义的本地 POSIX 文件系统（共享/NFS 未证明前不宣称支持）、三入口共锁、未验证设施/ACL 拒启、文件存在不等于持有锁与不靠删文件解锁；`:3922` 固定平台矩阵与 Windows 退出码 2、无 fallback、不扩 ADR-0026/FileRecoveryJournal 矩阵；11 端口清单 `:3601` 的 `exclusive` 行 `:3616` 登记唯一生产来源为已登记 adapter bundle 的 `Exclusive`、合成/测试注入不得替代 registry 的唯一性排除与当前 closed 状态；决策追溯 `:83` 登记本 ADR。设计仍为 draft，人的精确接纳 unknown，不声明实现或验证。
- 硬边界：[framework.md（ADR-0019）](framework.md#adr-0019)，`framework.md:418` 起；本 ADR 不重写其任何已锁定决定。
- 相邻决定：[framework.md（ADR-0030）](framework.md#adr-0030)，`framework.md:916` 起；[framework.md（ADR-0031）](framework.md#adr-0031)，`framework.md:975` 起；本 ADR 不改变二者的“首期不提供”与 closed 结论。
- 缺口来源：[gap-map.md](../../../../.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md) 的 §2.6 `pending-B13J-D04`（:141-161）与 U5（:201）；U6/U7 由同一 D04 的 article 设计与 secret candidate 段落承接，不在本 ADR 范围。
- 决策授权：用户 2026-11-01 B13J D04 指令；产出者为 Arch-Designer（Agent），不是人的签字、独立评审、安全评审或生产批准。
- 2026-11-02 r02 修订（本 ADR 决定日期 2026-11-01）（触发：非作者规划评审 `B13J-PLAN-F005`/`F007`，报告同上）：在“解除条件”新增第 8 条解除顺序链，把不变量与约束第 9 条的顺序要求落成规范性可勾选条件，并明确候选 S4.24 requires 候选 S4.23、候选 S4.22 在其后；说明该链无环、可满足，当前不可满足的只是规划登记顺序。POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、三入口共锁与约束 1–8 的实质均未改变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r03 修订（触发：非作者规划评审 `B13J-PLAN-R02-F001`/`F002`/`F005`，报告同上）：解除条件第 8 条改为“决策修订被独立评审接受 → 实施/验证卡（候选 S4.23、S4.24a、S4.24b）→ 真实验证 → 生产解除”的无环节次，原候选 S4.24 改称 S4.24b 并补入 S4.24a 节点；不变量与约束第 9 条的顺序要求已落入规范性解除条件。POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、三入口共锁与约束 1–8 的实质均未改变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r04 修订（本 ADR 决定日期 2026-11-01）（触发：B13J D07 文档型决策卡 `.add/project/knowledge/tasks/tasks.md` 的 `identity-auth-D07`，以及 Lead 授权记录 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r09.md` §1「B」与 `.spec/wip/recovery-runtime/evidence-b13j-d07/lead-authorization-r01.md` §4）：把「`Exclusive` 离开 `recoveryClosedPort`」从解除条件第 8 条的候选 S4.23 节点移到候选 S4.24b 节点——候选 S4.23 只把 `Exclusive` 作为已登记但 closed 的端口接入 bundle（不得填实现、不得脱离 closed 端口状态），`Exclusive` 离开 `recoveryClosedPort` 由候选 S4.24b 交付完成；该节点已失真的括注按事实改为「已登记为正式 Task `identity-auth-S4.23`」；`requires 候选 S4.23` 边与「不得先接线后补来源」保留。本次只改修订行、第 8 条该两节点与本条记录；不变量 1–9、解除条件 1–7、POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、`O_CLOEXEC` 常驻 fd、三入口共锁、锁路径与 ACL 合同、`journalPath != lockPath` 逐字未变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证，未宣称 S4.23/S4.24b 已启动或生产解除。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
- 2026-11-02 r05 修订（触发：架构评审 `B13J-D05-R05-F004`（MEDIUM，解除条件第 8 条的顺序自指），报告 `.spec/wip/recovery-runtime/evidence-b13j-d05/review-arch-r05/report.md`；授权：Lead 授权记录 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r10.md` §C）：删除解除条件第 8 条中作为独立节点列于候选 S4.24b 之前的「本 ADR 解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据」，把该交付与取证并入候选 S4.24b 节点正文，使产出者与证据同节点，消除「证据先于产出它的卡」的顺序自指；本次只改修订行、第 8 条该两节点与本条记录；不变量 1–9、解除条件 1–7 其余、POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、`O_CLOEXEC` 常驻 fd、三入口共锁、锁路径与 ACL 合同、`journalPath != lockPath` 逐字未变；未新增实现任务，未声称任何锁后端、ACL、双进程互斥或崩溃释放已验证，未宣称候选 S4.23/S4.24b 已启动或生产解除。本修订是作者文档处置，不构成评审 PASS、人的接纳或生产批准。
---

<a id="adr-0036"></a>
## ADR-0036: 测试项目中部署后验证类任务经决策不验证 (2026-11-05)

**目的**: 代码质量 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人于 2026-11-05 在 Lead 提出「A：3 张部署后验证卡经决策不验证、B：6 张生产恢复实现卡维持阻塞」后回复「继续」，采用 A；承接 ADR-0028 项目级授权与 ADR-0025/0033/0034/0035 同口径。

**继承关系**: 把 ADR-0025（identity S3.3/S3.5 有界豁免）与 ADR-0033/0034/0035（article 范围决策）的「测试项目不验证真实环境」口径扩展到 3 张部署后验证卡；不改变 ADR-0030/0031/0032 对生产恢复链的门禁，也不关闭对应任务。

**问题**: `identity-auth-S3.4`、`article-management-S1.4`（部署后真实 E2E 旅程）与 `identity-auth-S3.6`（production observability 真实部署验证与演练）均要求已部署版本、真实平台/账号/receiver、网络策略与演练窗口。本项目无部署目标、无真实平台与对应交付义务，专用命令 `go test -tags=e2e ./test/e2e/identity`、`go test -tags=e2e ./test/e2e/article` 与 I-OBS-DEP-01～06 从未执行（NOT RUN）。

**决策**: 3 张部署后验证任务按「经决策不验证」关闭（`[ ]`→`[x]`，完成@2026-11-05，记录中标注「经决策不验证」），未验证边界如实登记；不执行、不以模拟或既有单机证据冒充，不把既有 integration/Manifest PASS 当作本卡结果。其余 6 张生产恢复实现卡（`identity-auth-S4.5`、`S4.6`、`S4.23`、`S4.24a`、`S4.24b`、`S4.22`）**不受本决定影响，维持 `[ ]` 阻塞**。

### 现状 / 痛点 / 改造 / 效果

现状：三卡的其余前置证据均已具备（如 article S1.4 的历史 integration/Manifest 七项 PASS、S3.6 的候选版本标识与 observability 设计），缺的只是部署环境本身。

痛点：真实部署 E2E 与 observability 演练需要部署目标、平台、账号/receiver、网络策略、演练窗口与业务验收责任人，测试项目均不具备；继续挂起会让已完成实现无法收口，且不得据此宣称部署就绪。

| 方案 | 优势 | 劣势 |
|---|---|---|
| A. 搭建真实部署环境并执行 E2E/演练 | 可取得真实部署结论 | 需真实环境、平台/账号/窗口与责任人；测试项目无对应交付义务 |
| B. 经决策不验证，按范围关闭并登记未验证边界（选定） | 与测试项目定位一致；停止空转；不伪造部署结论 | 真实部署 E2E、平台 observability 与跨平台结论缺失 |
| C. 用本机模拟或既有单机证据冒充真实部署 | 无需新增环境 | 伪造证据；违反「未运行不冒充通过」，明确拒绝 |

- **场景匹配**：Han 为测试项目，无生产部署与跨系统一致性义务；三卡之外的实现/契约/真实 PG 证据均为本机真实执行。
- **劣势可接受性**：部署与平台结论缺失可接受，前提是显式登记为未验证，且不得外推为部署就绪、observability 就绪或 production readiness。

### 约束

- 只关闭上述 3 张卡；不改变其他任务、批次 Plan、设计正文或任何生产代码。
- 关闭理由必须写为范围决策，不得写成「E2E 通过」「演练通过」或「observability 已验证」。
- 未验证边界保留：identity/article 部署后 E2E 旅程、真实部署版本与网络、真实日志留存与 scrape、四类告警/Dashboard/receiver 升级链、演练窗口与值守、双系统/跨平台一致性。
- 不改变 ADR-0022（observability 设计）、ADR-0025（有界豁免范围）与 ADR-0030/0031/0032（生产恢复门禁）。
- 6 张生产恢复实现卡保持阻塞：开始它们仍需各自的前置、精确写集、真实独立故障域与非作者独立评审，不因本 ADR 获得启动授权。

### 引用记录

- 2026-11-05 `identity-auth-S3.4`、`identity-auth-S3.6`、`article-management-S1.4` 按本决定关闭。来源：项目负责人回复「继续」采用 Lead 建议方案 A。

---

<a id="adr-0037"></a>
## ADR-0037: 生产恢复实现链六张任务不纳入本版交付范围 (2026-11-05)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人于 2026-11-05 在 Lead 列出「以 ADR 登记不纳入本版」与「保持阻塞」两个方向后回复「继续」，采用前者；承接 ADR-0028 项目级授权，并落实 ADR-0030/0031/0032 已登记的「首期不提供」范围。

**继承关系**: 把 ADR-0030（首期不提供生产 CAS JournalHeadAnchor provider）、ADR-0031（首期不提供生产 restore 工具与批准主体）、ADR-0032（POSIX-only、Windows 生产 unsupported）的范围结论落到任务簿；不改变三条 ADR 的不变量、解除条件、启动门禁与 fail-closed 语义。

**问题**: `identity-auth-S4.5`、`S4.6`、`S4.23`、`S4.24a`、`S4.24b`、`S4.22` 是生产恢复能力的实现卡，启动前置（具名 owner、来源地址范围、见证 owner、真实独立故障域、精确写集与新增依赖授权）均未满足，且真实独立故障域在本项目不可得。三条 ADR 已决定首期不提供相应能力，但任务簿仍以 `[ ]` 表示「待做」，与既定范围不一致，使项目无法形成自洽的收口结论。

**决策**: 六张实现卡按「不纳入本版」关闭（`[ ]`→`[x]`，完成@2026-11-05，记录标注「经决策不纳入本版」）。**不实施任何代码、不改变各卡精确写集、不声称生产恢复能力已交付或已验证**；生产 root 继续按现有代码 fail-closed（`recoveryClosedPort` / `recoveryClosedRestoreDriver` / `ErrAdmissionUnsupported` 不变）。

### 现状 / 痛点 / 改造 / 效果

现状：生产恢复链的四端口 bundle、restore 接线、POSIX 独占锁与 11 端口 root 装配均未开始；现有实现保持恢复入口关闭，符合 ADR-0030/0031/0032 的安全结论。

痛点：任务簿把它们记为「待做」，读者会以为本版承诺交付生产恢复路径；实际首期范围已由三条 ADR 排除。继续以 `[ ]` 挂着既不会启动，也无法说明「为什么不做」。

| 方案 | 优势 | 劣势 |
|---|---|---|
| A. 本版实现六卡 | 可取得完整生产恢复路径 | 需具名 owner/见证、真实独立故障域、42 文件级写集与新增依赖授权；与 ADR-0030/0031/0032 的「首期不提供」直接冲突 |
| B. 经决策不纳入本版，关闭六卡并登记未实现边界（选定） | 与既有 ADR 范围一致；任务簿可自洽收口；不伪造实现 | 生产恢复能力在本版不存在，未来需重开 |
| C. 维持 `[ ]` 长期挂起 | 不产生新的范围决策 | 状态与实际范围矛盾；无法区分「待做」与「不做」 |

- **场景匹配**：Han 为测试项目，首期目标是验证 ADD 流程与最小可用闭环（注册/登录 + 文章管理），生产灾备恢复不在交付范围。
- **劣势可接受性**：生产恢复能力缺失可接受，前提是显式登记为「未实现/未验证」，且不得外推为灾备就绪或 production readiness。

### 约束

- 只关闭上述 6 张卡；不改变任何生产代码、部署材料、设计正文、Plan 批次归属或其余 ADR。
- 关闭理由必须写为范围决策，不得写成「已实现」「已解除」「生产恢复可用」或「已验证」。
- ADR-0030/0031/0032 的门禁继续有效：即便未来要实现，也必须先满足各自前置并按失败证据→最小实现→集成测试→真实故障域→非作者独立评审的顺序推进，且另立授权与 ADR 重开本范围。
- 生产 fail-closed 语义保持不变：未配置 provider/restore/锁时，恢复入口仍以 closed port/驱动拒绝，不引入 fallback 或降级。
- 未实现边界保留：生产 CAS anchor provider、KeyRetirementCapability 签发与单次消费、anchor/CAS adapter bundle 与 registry 注册、restore source/target/tool 与批准主体接线、POSIX 独占锁与 `Exclusive` 接线、11 端口 production root 装配与 secret candidate 来源。

### 引用记录

- 2026-11-05 `identity-auth-S4.5`、`S4.6`、`S4.23`、`S4.24a`、`S4.24b`、`S4.22` 按本决定关闭。来源：项目负责人回复「继续」采用 Lead 建议方向。

---

<a id="adr-0038"></a>
## ADR-0038: 合成 loopback/example DSN 不视为真实凭据 (2026-11-05)

**目的**: 代码质量 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人于 2026-11-05 在 Lead 给出「视为非凭据并授权提交 / 先脱敏 / 暂不提交」三选一后回复「继续」，采用第一项。

**问题**: 提交 IR-01 工作树前的敏感信息扫描发现 28 个文件含 `postgres://user:password@host` 形式字符串。抽查取值为 `dsn-secret`、字面 `...` 或本机 loopback 合成口令，host 为 `127.0.0.1` 或 `dsn.example`，不含真实域名、真实账号或生产凭据。但项目规则「禁止提交凭据或密钥」需要一个可执行的边界，否则合成测试串会被当作真实凭据阻断提交，或反过来削弱规则。

**决策**: 仅当同时满足以下条件时，DSN 字符串视为**非凭据**，可随代码/证据入库：

1. host 为 `127.0.0.1`、`localhost`、`::1` 或保留域名 `dsn.example`；
2. 口令为显式占位（`...`、`<redacted>`、`changeme`）或合成的测试值；
3. 不含真实用户名、真实数据库名、真实 CA/证书或任何生产地址。

不满足任一条件时仍按「禁止提交凭据或密钥」处理：先脱敏，再提交；不得以本 ADR 为依据提交真实凭据。

### 现状 / 痛点 / 改造 / 效果

现状：`cmd/han-server/database_config_test.go`、`deploy/db/README.md` 与多轮证据日志/快照重复出现同一组合成 DSN；其中 6 份评审与 2 份索引的哈希被证据链引用。

痛点：若一律脱敏，会改写独立评审产物与证据索引，破坏 `S15-*` 的可复核哈希链；若一律提交，又缺少「合成值」的显式边界。

| 方案 | 优势 | 劣势 |
|---|---|---|
| A. 一律脱敏后再提交 | 不把任何口令写进历史 | 需改写 28 个文件，其中 8 份为被引用证据，必须重建哈希链 |
| B. 显式定义合成值边界后提交（选定） | 保留证据哈希链；规则仍对真实凭据生效 | 依赖人工判断 host/口令性质，需在扩围时复核 |
| C. 暂不提交 | 零风险 | 322+ 项工作树长期未入库，交付不可回溯 |

- **场景匹配**：本项目为测试项目，DSN 全部指向本机或保留域名，无生产连接。
- **劣势可接受性**：人工判断风险由「三条同时满足」与后续扫描复核控制；一旦发现真实值，仍按禁止项处理。

### 约束

- 本 ADR 只豁免合成 loopback/example DSN；真实凭据、私钥、token、云厂商密钥一律不得提交。
- 新增或修改测试 DSN 时须保持上述三条；CI/提交前扫描若发现非合成 host，应阻断。
- 不得据此把生产连接串、备份口令或 keyring 材料写入仓库或证据包。

### 引用记录

- 2026-11-05 首次提交 IR-01 工作树时适用：扫描 31,619 个文本文件，28 个文件命中，均为合成值。
