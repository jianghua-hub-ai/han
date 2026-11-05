# 身份、会话与凭据领域决策

本文件承载身份认证、登录态、凭据交付与邮件依赖类决策。领域边界由项目架构定义。

---

## ADR-0005: 认证采用服务端会话与受保护的不透明 Session Cookie (2026-10-19，修订 2026-10-20)

**目的**: 接口设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对 D5 当面拍板采用服务端会话（2026-10-19）；当前用户于 2026-10-20 明确批准不透明随机 Session Cookie、Cookie 安全属性、会话轮换与会话绑定 CSRF Header

**修订关系**: 本次在同一编号内细化既有服务端会话决定，补齐浏览器载体、跨站请求防护和会话固定攻击防护；不改变“会话权威状态存 PostgreSQL、可即时撤销”的原结论。

**问题**: 「已登录」状态保存在哪里，才能满足需求要求的管理员停用或删除用户后该用户**立即**退出登录？

**决策**: 采用服务端会话——会话记录存于 PostgreSQL，浏览器使用不可预测的不透明随机 Session Cookie。浏览器 API 默认 same-origin，禁止跨源 credentialed CORS。登录请求必须校验可信 `Origin` 与 Fetch Metadata（或等价 fail-closed 信号）；已有会话再次登录时创建新会话并撤销旧会话。Cookie 启用 `Secure`、`HttpOnly` 与已批准的 `SameSite` 策略；认证成功以及角色、权限、强制改密状态等安全上下文变化时轮换会话 ID。写请求继续校验会话绑定 CSRF Header。

### 现状 / 痛点 / 改造 / 效果

服务端会话与无状态令牌是两条成熟路线。OWASP Session Management 与 Django、Rails 的默认模型都属于前者，其核心设计哲学是「会话状态由服务端持有，失效即删除记录」。JWT 体系的核心哲学是「状态自包含于令牌，校验无需 I/O」，代价是撤销必须另建机制。

需求 `US-IDENTITY-03.6` 第 3 条与 `US-IDENTITY-03.7` 第 2 条都使用「立即结束该用户当前登录状态」的表述，且 `US-IDENTITY-03.10` 要求旧临时密码「立即失效」。这些要求与无状态令牌的撤销需求正面冲突——JWT 要满足它们必须额外实现黑名单或版本号，等于把会话状态又存回服务端，却多承担一套机制。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 服务端会话（存 PostgreSQL） | 登录写会话表，请求携带会话标识，服务端查表校验 | 撤销是删除记录而非补偿逻辑；无新增基础设施；与数据库事务天然一致 | 每次请求一次会话查询；会话表需清理策略 |
| 服务端会话（存 Redis） | 同上，会话存 Redis | 校验更快；天然支持过期 | 新增基础设施与部署依赖；首期容量证据不足 |
| JWT（无状态） | 签发自包含令牌，服务端不存状态 | 校验无 I/O；水平扩展简单 | 立即登出必须额外实现黑名单或版本号，否则须等令牌自然过期 |
| JWT + 撤销黑名单 | JWT + 服务端撤销列表 | 保持无状态校验优势 | 同时承担两套机制的复杂度；黑名单与令牌生命周期需额外一致性设计 |

**场景特征匹配**：本项目最关键的会话需求是**可即时撤销**，而非无状态扩展。服务端会话使「停用用户」与「结束其登录状态」落在同一事务内，不存在不一致窗口。

**劣势可接受性**：每次请求一次会话查询，在首期规模下由主键索引承担，且该查询与业务查询共用同一连接池。Cookie 会话需要 CSRF 合同与会话轮换，但这是浏览器凭据自动携带所必需的防护，不引入第二套认证机制。会话绝对期限、空闲期限、容量和清理周期均保持 TBD，不能由实现填入未经容量评估的默认值。

### 约束

- 会话记录唯一存放于 PostgreSQL `Session` 表；不引入第二处会话状态。
- 接入层是唯一的会话校验位置；业务模块从请求上下文取得已确认身份，不各自解析凭据。
- Session Cookie 只保存不透明随机会话标识，不承载用户、角色、权限或其他业务声明；Cookie 必须启用 `Secure`、`HttpOnly` 与已批准的 `SameSite` 策略。
- 认证成功和任何权限安全上下文变化后必须轮换会话 ID，旧会话立即失效，防止会话固定与旧权限继续生效。
- 使用 Cookie 会话的状态写请求必须携带与当前会话绑定的 CSRF Header；缺失、不匹配或无法校验时 fail closed。Cookie 名称和 Header 名称由详细设计确定，不得改变该机制。
- 浏览器 API 默认同源；不得配置跨源 credentialed CORS。登录端点必须拒绝不可信或缺失必要同源证据的浏览器请求，不以“登录前无会话”为由免除 login CSRF 防护。
- 已有有效会话再次登录时必须轮换会话并撤销旧会话，不能覆盖 Cookie 后让旧服务端会话继续有效。
- 用户停用或删除的同一事务内必须删除该用户全部会话记录。
- 登录失败对外统一，不区分账号不存在、密码错误、停用或删除；内部真实原因只进入最小化审计与受控可观测链路。
- 会话绝对期限、空闲期限、容量与清理周期保持 TBD；未经目标环境容量与安全评估不得硬编码默认值。

### 引用记录

- 2026-10-19 架构设计阶段采用：本 ADR 约束架构文档 §5.2.2、§6.2.2、§6.3.4、§9.1 的会话校验与时序设计。来源：项目负责人对 D5 当面拍板。
- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本次同编号修订，补齐 Session Cookie、CSRF 与会话轮换合同。来源：当前用户明确批准的详细设计安全方案。
- 2026-10-20 架构修订 `architecture-ir01-r05` 继续修订本决定，补齐 same-origin、login CSRF 与再次登录撤销旧会话。来源：当前用户批准 IA-DES-R01-F006 整改方向。

---

## ADR-0006: 邮件采用强制 TLS 的 SMTP 直连并经适配器调用 (2026-10-19，修订 2026-10-20)

**目的**: 数据流向 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对 D6 当面拍板 SMTP 直连（2026-10-19）；当前用户于 2026-10-20 批准强制 TLS、证书验证与启动 fail-closed

**修订关系**: 本次在同一编号补齐 SMTP 传输安全与处理方部署门禁，不改变邮件适配器和调用方决定失败语义的原结论。

**问题**: 系统需要发送临时密码邮件与文章转移通知，而这两类邮件在需求中的失败语义完全相反——密码邮件失败必须导致整体失败，转移通知失败不得阻断主流程。邮件发送如何组织才能表达这两种语义？

**决策**: 定义邮件适配器接口，生产实现为 SMTP 直连且必须使用隐式 TLS 或强制 STARTTLS；验证证书链与主机名，禁止明文、机会式降级和 `InsecureSkipVerify`。安全配置或获批准 SMTP 处理方登记缺失时服务启动或相关能力 fail closed。失败语义仍由调用方决定。

### 现状 / 痛点 / 改造 / 效果

传输层有两条主流路径：SMTP 直连（RFC 5321）与第三方事务性邮件服务（Amazon SES、Postmark、SendGrid）。差异在于投递率、退信处理与运维负担的转移。

调用契约层面的问题更关键。需求给出两条相反规则：`US-IDENTITY-03.10` 第 2 条要求「临时密码邮件发送失败时应把本次密码重置视为整体失败」；`US-IDENTITY-03.9` 第 4 条要求「新作者或原作者通知发送失败时应保持文章转移结果有效，不得回滚」。若邮件层自行决定失败处理，这两条规则必然有一条无法满足。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| SMTP 直连 | 应用直连 SMTP 服务器同步提交 | 无第三方账号依赖；离线环境可用；发送结果直接可得 | 需自建投递可靠性与退信处理；易进垃圾箱 |
| 第三方事务性邮件服务 | 调用 HTTP API | 投递率高；退信与统计完善 | 供应商依赖；需外网与凭据；离线环境不可用 |
| 邮件适配器 + 开发期内存实现 | 定义接口，生产接 SMTP，开发与测试用内存实现 | 两类失败语义都可在接口层表达；E2E 可断言投递内容而不真发 | 多一层抽象 |

**场景特征匹配**：项目同时存在两种相反的失败语义，接口层必须把失败处理留给调用方；同时需要可离线运行以支撑集成测试与 E2E。

**劣势可接受性**：多一层抽象的成本，由已存在的两个真实消费者（临时密码、转移通知）与两种相反语义共同构成引入依据，符合抽象原则 §4「第二个真实消费者出现且差异可命名时再提炼共同契约」。SMTP 投递率低于第三方服务的劣势，在首期无对外服务质量承诺的前提下可接受；若后续投递率成为问题，替换为第三方服务只需新增一个适配器实现，不改调用方。

### 约束

- 邮件调用一律经 `internal/mailer` 定义的接口；业务模块不得直接构造 SMTP 连接。
- 邮件层不决定失败处理方式，只返回投递受理结果；由调用方决定回滚或记录。
- SMTP 出口地址、端口与凭据由环境变量注入，不写入代码或配置文件。
- 只允许隐式 TLS 或强制 STARTTLS；STARTTLS 未提供、协商失败、证书链/主机名无效时必须失败，禁止降级为明文。
- 禁止 `InsecureSkipVerify` 或等价跳过验证；明文临时凭据和邮箱不得经不安全通道发送。
- 部署必须登记获批准的 SMTP 数据处理方；安全参数或处理方门禁缺失时不得启用邮件能力。
- 临时密码邮件与转移通知的失败语义分别为 fail closed 与 best effort，见 ADR-0011。
- 邮件正文文案与发件人名称属待确认项（架构文档 Q13）。

### 引用记录

- 2026-10-19 架构设计阶段采用：本 ADR 约束架构文档 §6.2.2、§6.3.3、§9.2 的邮件调用设计。来源：项目负责人对 D6 当面拍板并指定 SMTP 直连。
- 2026-10-20 架构修订 `architecture-ir01-r05` 采用本次同编号修订，补齐 TLS、证书与部署门禁。来源：当前用户批准 IA-DES-R01-F004 整改方向。

---

## ADR-0007: 初始管理员 bootstrap 与灾备恢复凭据经独立环境变量注入 (2026-10-19，修订 2026-10-20)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 项目负责人对首次 bootstrap 凭据当面拍板（2026-10-19）；当前用户于 2026-10-20 批准独立一次性恢复凭据

**修订关系**: 保留“首次建库 bootstrap 仅首次生效”的原决定；新增的 recovery secret 只用于受控灾备恢复，不替代、不复用 bootstrap secret，也不决定日常轮换 Q1。

**问题**: 首期须自动创建用户名 `admin` 的管理员账号，其初始密码已由用户提供。需求要求该凭据「安全注入且不得进入项目文件、示例、日志或版本库」，注入方式如何选择？

**决策**: bootstrap secret 与 recovery secret 使用不同环境变量和运行门禁。每次恢复输入唯一 `recovery_run_id` 与 recovery secret；应用只计算并使用 recovery secret 的 HMAC 指纹，明文不持久化。指纹、run ID 与 consumed 状态由 ADR-0019 的 Recovery Journal 保存；已消费、Journal 不可用或 Journal/业务库不一致时保持停服。

### 现状 / 痛点 / 改造 / 效果

业界主流有四种注入路径：环境变量（12-Factor App 的 Config 原则）、密钥管理服务（Vault、K8s Secret）、配置文件、首次启动交互式输入。

需求只提出三条硬要求：安全注入、不落盘、首次登录必须修改密码。配置文件与交互式输入直接违反前两条或不适用于容器化部署（需求 `US-IDENTITY-03.1` 第 4、5 条）。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 环境变量 + 仅首次生效 | 启动时读取，仅当 `admin` 不存在时使用，之后忽略该变量 | 实现面最小；与 12-Factor 一致；离线可用 | 环境变量可能被进程检视读取；无轮换能力 |
| 环境变量（每次启动都生效） | 每次启动重设密码 | 简单 | 会导致密码被反复重置，与首次修改后持久化的预期冲突 |
| 密钥管理服务 | 启动时从密钥服务拉取 | 轮换与审计完善 | 首期需新增运维组件与凭据依赖 |
| 首次启动交互式注入 | 启动时交互输入 | 不落任何配置面 | 无法自动化部署与集成测试 |

**场景特征匹配**：需求只要求一次性安全注入且首次登录强制改密，后续轮换属待确认项（需求 Q1）。

**劣势可接受性**：环境变量可能被进程检视读到——该风险通过「仅首次启动生效」收敛到首次启动窗口，且首次登录强制改密使初始凭据是一次性的。无轮换能力的劣势由需求 Q1 覆盖；Q1 澄清后若要求轮换，届时按变更请求评估密钥管理服务。

### 约束

- 初始凭据仅从环境变量读取，仅当 `admin` 账号不存在时生效。
- recovery secret 必须与 bootstrap secret 使用不同配置键，只在独占恢复终结步骤读取；缺失、复用或恢复流程校验失败时保持停服。
- recovery secret 明文不得进入业务数据库、Recovery Journal、日志、审计或运行记录；只保存带版本的 HMAC 指纹。
- `recovery_run_id` 在每次恢复中全局唯一；同一 run ID 或指纹一经 consumed，任何后续恢复都必须拒绝。
- recovery secret 的复用检测必须委托 ADR-0019 的完整 keyring；不得仅用 active key 计算单一指纹后判重。keyring 缺失或历史 key 不可验证时保持停服。
- recovery secret 只重置初始 `admin` 的恢复凭据状态；恢复后首次登录必须强制改密，不能直接开放其他管理员能力。
- 不得把凭据写入项目文件、示例、日志、错误信息或版本库。
- 初始 `admin` 首次登录必须修改密码，未完成前阻止使用其他管理员能力（需求 `US-IDENTITY-03.1` 第 2 条）。
- 初始 `admin` 的管理员权限永久不可撤销（需求 `US-IDENTITY-03.4` 第 5 条）。
- 凭据轮换方式与首次修改后的口令规则是否沿用现有规则属待确认项（需求 Q1）。

### 引用记录

- 2026-10-19 架构设计阶段采用：本 ADR 约束架构文档 §4.4、§8.3 与需求 `US-IDENTITY-03.1` 的实现路径。来源：项目负责人对 D7 当面拍板。
- 2026-10-20 架构修订 `architecture-ir01-r06` 采用本次同编号修订，区分 bootstrap secret 与一次性 recovery secret。来源：当前用户批准灾备恢复合同。
- 2026-10-20 架构修订 `architecture-ir01-r07` 补齐 run ID、HMAC 指纹和跨恢复批次消费检测。来源：当前用户批准恢复日志合同。
- 2026-10-20 架构修订 `architecture-ir01-r08` 补齐跨 HMAC key 版本的 secret 复用检测调用约束。来源：当前用户批准第四轮复审整改。

---

## ADR-0011: 临时凭据交付 fail closed、SMTP 残余窗口治理与转移通知 best effort (2026-10-19，修订 2026-10-20)

**目的**: 错误处理 | **仓库**: han

**状态**: 已接受 | **确认来源**: `requirements-ir01-r15` checkpoint `decision-09cd3493c4eced164f39341156ecd422a1eec46d1837f115da8d55bc013ca532`（确认对象摘要 `sha256:dc60ddb68bc7ee73ad24382d7f88a6c7eb1a9919bf54bda3c632b0c45c154a1c`）延续临时凭据和转移通知业务语义；当前用户于 2026-10-20 明确接受 SMTP 与数据库不能真正原子及其残余窗口治理方案

**修订关系**: 2026-10-20 首次修订把新管理员创建纳入原“密码重置与重发 fail closed、转移通知 best effort”结论；本次继续原位修订，明确 SMTP 受理与 PostgreSQL 提交之间无法真正原子，不再把旧表述解释为零风险。

**问题**: 新管理员创建、密码重置、临时密码重发和无归属文章转移都包含数据库写入与邮件投递，但成功判据不同；写入与投递之间的失败组合如何处理？

**决策**: 新管理员创建、密码重置与临时密码重发仍采用应用层 fail closed：SMTP 未受理时不得提交相关账号或凭据写入；SMTP 已受理后再提交 PostgreSQL。首期明确接受“SMTP 已受理但数据库最终提交失败”的残余窗口，通过幂等键、重发、最新凭据语义、追加式审计、告警与 trace 关联处置。无归属文章转移仍先提交业务事实，再以 best effort 通知新作者与已删除原作者。

### 现状 / 痛点 / 改造 / 效果

四类调用都有“数据库写入 + 网络投递”结构，但没有共同成功判据。新管理员创建、密码重置和临时密码重发的成功判据都是“收件人获得已受理且可使用的临时凭据”，因此相关账号或凭据写入必须与投递同生共死；无归属文章转移的成功判据是“文章归属已变更”，邮件只是告知，因此投递不得影响结论。

写入与投递的顺序有两种可能，各自的失败残留不同。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 事务内暂存、SMTP 受理后提交 | 暂存账号或凭据，SMTP 未受理则回滚，受理后提交数据库 | 业务成功必须经过邮件受理；失败请求不主动替换旧凭据 | SMTP 与数据库不能原子，存在“邮件已受理但提交失败”窗口；事务跨网络调用 |
| 先投递后写入 | 先发邮件再创建账号或写临时密码 | 无长事务 | 写入失败时收件人得到无法使用的密码，与三类凭据交付需求冲突 |
| 先写入后投递，失败不回滚 | 先提交账号或凭据，投递失败仅记录 | 数据库事务短 | 留下未交付凭据的管理员账号，或使用户拿不到已经生效的新密码 |

**场景特征匹配**：三类临时凭据交付都要求“邮件成功才算业务成功”，而无归属文章转移要求“归属改变后再通知”。同一邮件适配器只返回投递受理结果，由调用方按这两类成功判据决定提交或回滚。

**劣势可接受性**：SMTP 超时保持 TBD，按目标环境测量后确定。残余窗口不会被文档伪装为已消除：邮件中的临时凭据在数据库提交失败时不可用；客户端以幂等键安全重试，后续成功重发或重置产生最新凭据并使旧值失效；审计事件、trace 与告警用于识别并处置。若未来业务要求“受理邮件与数据库提交零窗口”，必须修订协议和交付机制，而不是继续沿用本决定。

### 约束

- 新管理员创建：仅初始 `admin` 可发起；投递未获受理时，管理员账号与临时密码写入必须一起回滚，不得留下未交付凭据的管理员账号。
- 密码重置：任一当前有效管理员可重置普通用户密码，只有初始 `admin` 可重置其他管理员密码；投递未获受理时，本次临时密码写入必须回滚，原凭据状态保持不变。
- 临时密码重发：新管理员临时密码邮件仅初始 `admin` 可重发，普通用户密码重置邮件可由任一当前有效管理员重发；投递成功才使新值生效并使旧值失效，投递失败时本次新值回滚，旧值不因失败请求失效。
- 三类临时凭据写操作必须消费 `Idempotency-Key`；相同作用域、相同请求摘要重放返回原结果，摘要冲突必须拒绝，防止残余窗口后的盲目重复创建或替换。
- SMTP 已受理但数据库最终提交失败时，对外返回失败，不声称凭据可用；记录失败阶段、幂等作用域、错误码和 `trace_id`，触发可观测告警。日志和审计不得保存明文凭据。
- 后续成功重置或重发生成最新凭据并使此前值失效；收到残余窗口邮件的用户不得因此获得可用旧凭据。
- 文章转移：只允许无归属文章；转移提交后通知新作者与已删除原作者。任一通知失败不得回滚新归属、无归属约束移除、发布意图或违规状态，投递结果写入 `TransferNotice` 供核对。
- SMTP 投递必须设置明确超时，禁止无限等待。
- 转移通知的人工重发能力是否提供属待确认项（架构文档 Q10）。

### 引用记录

- 2026-10-19 架构设计阶段采用：本 ADR 约束架构文档 §6.3.3、§9.2 与需求 `US-IDENTITY-03.10`、`US-IDENTITY-03.9` 的实现路径。来源：需求已确认条款，架构阶段据此确定实现顺序。
- 2026-10-20 架构修订 `architecture-ir01-r03` 采用本次同编号修订：把新管理员创建纳入临时凭据 fail-closed 合同，补齐两类重发授权、无归属转移范围及已删除原作者通知。来源：已接受的 `requirements-ir01-r14` checkpoint `decision-f167ac06ba4e751c2fa8681835c58c6dba2d8fea758ba86e44a9ce63d0f25159`。
- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本次同编号修订：显式接受并治理 SMTP 已受理而数据库提交失败的残余窗口。来源：当前用户批准的详细设计安全方案与 `requirements-ir01-r15`。

---

## ADR-0012: 密码与临时凭据采用 Argon2id 哈希 (2026-10-20)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-20 明确批准 Argon2id 与 `golang.org/x/crypto` 依赖

**问题**: 用户密码和临时凭据应使用什么哈希机制，才能提供内存硬化保护并允许参数随目标环境演进？

**决策**: 使用 `golang.org/x/crypto/argon2` 实现 Argon2id。持久化编码必须包含算法版本、盐和实际参数，以支持验证与后续重哈希；内存、迭代和并行参数在目标部署环境基准测试后确定，不在架构中硬编数值。

### 现状 / 痛点 / 改造 / 效果

常见候选包括 bcrypt、scrypt、Argon2id 与通用快速哈希。通用快速哈希不适合密码；bcrypt 成熟但内存硬化能力有限；scrypt 内存硬化成熟；Argon2id 是面向密码存储的现代内存硬化方案。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| Argon2id | 内存硬化的混合数据相关/无关模式 | 抵抗 GPU 并行破解；参数可调；编码可自描述 | 需要第三方 Go 依赖与目标环境基准测试 |
| bcrypt | 自适应计算成本 | 生态成熟；使用简单 | 内存成本低，面对并行硬件的保护弱于 Argon2id |
| scrypt | 内存硬化 KDF | 成熟且抗并行 | 参数调优与资源占用同样需要基准测试 |
| SHA-2 等快速哈希 | 快速摘要 | 标准库直接可用 | 不适合密码，低成本暴力破解 |

本项目处理用户密码和可登录的临时凭据，Argon2id 的内存硬化与可演进编码符合安全目标。新增依赖已获批准；参数未知的劣势通过目标环境基准测试解决，不以示例值变成生产承诺。

### 约束

- 用户密码、初始管理员首次改密后的密码和临时凭据摘要统一使用 Argon2id；明文只允许存在于受控输入和邮件构造的瞬时内存。
- 每条摘要使用独立随机盐；持久化编码包含算法标识、版本、参数、盐与摘要。
- 登录验证发现旧参数时，可在认证成功后按明确迁移策略重哈希；不得降低参数。
- 参数必须通过目标环境的安全与延迟基准测试确定，具体数值保持 TBD。
- `golang.org/x/crypto` 是本决定唯一新增的密码哈希依赖；版本由实现批次锁定并接受依赖审查。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束密码数据模型、技术选型、部署基准与测试策略。来源：当前用户明确批准。

---

## ADR-0013: 删除账号标识保留与专用通知邮箱清理生命周期 (2026-10-20)

**目的**: 数据模型 | **仓库**: han

**状态**: 已接受 | **确认来源**: `requirements-ir01-r15` checkpoint `decision-09cd3493c4eced164f39341156ecd422a1eec46d1837f115da8d55bc013ca532`（确认对象摘要 `sha256:dc60ddb68bc7ee73ad24382d7f88a6c7eb1a9919bf54bda3c632b0c45c154a1c`）

**问题**: 删除账号后，用户名、邮箱、手机号和转移通知邮箱如何保留、限制复用并安全清理？

**决策**: 删除账号以不可恢复墓碑保存标识生命周期。用户名永久保留且永不复用；邮箱和手机号在对应保留数据存在期间继续参与唯一性冲突检查，清理后仅恢复参加新账号唯一性校验的资格。专用 `retained_email` 仅用于无归属文章转移通知，并在全部关联无归属文章已处置且转移通知地址已独立留存后清理。

### 现状 / 痛点 / 改造 / 效果

物理删除全部标识会导致用户名被冒用，并破坏已删除原作者通知；永久保留全部个人标识又违反最小必要原则。项目需要按标识用途区分永久占用与条件保留。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| 分类生命周期 | 用户名永久墓碑；邮箱/手机号条件保留；通知邮箱按处置条件清理 | 同时保护身份连续性、唯一性和数据最小化 | 清理条件跨账号墓碑、无归属文章与通知快照 |
| 全部永久保留 | 所有删除标识永不清理 | 规则简单；不会复用 | 过度保留个人数据 |
| 全部立即清理 | 删除即释放全部标识 | 数据最少 | 用户名可被冒用；转移通知无法完成 |

分类生命周期直接对应 r15 已确认行为。跨对象清理判断的复杂度由同一 PostgreSQL 事务、明确所有权、审计事件和集成测试控制。

### 约束

- `username` 在删除墓碑上永久保留并永久参与唯一性冲突检查。
- `email`、`phone` 及其保留表示在存在期间参与唯一性冲突检查；清理后不恢复旧账号，也不保证新账号创建成功。
- `retained_email` 只供无归属文章转移通知使用，不得用于登录、恢复、资料展示或其他邮件。
- 只有在该墓碑关联的无归属文章全部完成恢复公开、转移或永久删除，且已发生转移的通知地址已进入独立通知快照后，身份认证模块才可清理 `retained_email`。
- 其他删除后身份信息的具体清理条件仍为 r15 Q2，不由本 ADR补齐。
- 标识保留、清理和复用拒绝均写追加式审计事件；审计不保存完整敏感标识值。

### 引用记录

- 2026-10-20 架构修订 `architecture-ir01-r04` 采用本决定，约束 User 墓碑、唯一性、邮箱清理与跨模块查询。来源：已接受的 `requirements-ir01-r15`。
---

## ADR-0023: Password reset 采用持久操作屏障、恢复 worker 与认证消费者 fail-closed Gate (2026-10-24)

**目的**: 架构设计 | **仓库**: han

**状态**: 已接受 | **确认来源**: 当前用户于 2026-10-24 明确授权“对待确认或决策项采用建议值并继续”，并在本轮指定采用方案 A

**问题**: Password reset 在 SMTP 已受理、PostgreSQL 提交结果不确定、同步核验与 cleanup 又都失败时，如何跨请求、进程重启和 worker 接管持续阻止旧认证能力被误放行，并最终收敛凭据、会话、幂等和审计状态？

**决策**: 采用方案 A。identity 领域以 PostgreSQL 持久 `PasswordResetOperation` 作为 password-reset 安全结果权威，在同一事务创建 recovery job/outbox；由数据库租约 worker 恢复并收敛。Login、PasswordChange 与 Session resolve 在 operation 非终态、未知、非法、不可读或关系不一致时统一 fail closed。`IdempotencyRecord` 只保存重放合同，不得单独证明认证能力安全或放行消费者。

### 现状 / 痛点 / 改造 / 效果

现状是 ADR-0011 已接受 SMTP 与 PostgreSQL 无法真正原子，并用幂等、最新凭据、审计、trace 与告警治理残余窗口；ADR-0015 定义作用域化幂等；ADR-0018 定义恢复后的认证状态统一 fail closed。identity design r34 进一步证明：password-reset 的 commit-ack-loss、同步核验失败和 cleanup 双失败会留下跨请求安全不确定性，有限请求线程 cleanup 不能建立 durable invariant。

痛点在于 `IdempotencyRecord` 只能表达请求重放结果，不能单独证明 exact TemporaryCredential、其派生 Session、临时凭据 consumed 后形成的永久密码能力以及 AuditEvent 已一致收敛。若认证消费者继续只读凭据或幂等状态，未知 operation 可能被错误解释为可登录、可改密或可恢复 Session。

| 方案 | 实现机制 | 优势 | 劣势 | 结论 |
|---|---|---|---|---|
| A：持久 operation + recovery job/outbox + 三消费者 Gate | SMTP 前持久化 operation、pending credential 与同事务 job；数据库租约 worker 按固定锁顺序收敛；Login、PasswordChange、Session resolve 查询同一 Gate | 跨请求、重启和租约接管仍保留安全屏障；operation 与能力状态可原子核对；可覆盖 commit-ack-loss、consumed 和同键重放 | 增加表、约束、索引、Gate 查询、后台 worker、运行手册和部署/回退约束 | **采用** |
| B：延长有限同步 cleanup | 增加请求内核验、重试次数或超时，并继续以幂等状态推断 | 局部改动较少；请求路径直观 | 数据库/网络不可用或双 cleanup 失败时仍无持久屏障；有限尝试无法证明 durable invariant | 拒绝 |
| C：长期用户级维护锁 | 对目标用户保持认证锁，人工核对后解除 | 初始拒绝语义直接 | 可用性差；长期锁本身仍需持久状态、恢复、审计和解锁模型，最终退化为方案 A 的不完整版本 | 拒绝 |

改造后，operation 成为认证安全的单一结果权威；请求线程的 cleanup 只可 enqueue 或加速恢复，不承担安全证明。worker 在一个受控事务中核对并原子收敛 exact credential、派生或目标 Session、consumed 后永久密码能力、IdempotencyRecord 稳定快照和 AuditEvent。无法证明安全时维持 Gate；只有权威状态与能力后置一致时才进入终态并允许稳定重放。

效果是 password-reset 的不确定结果不会因请求结束、进程重启、同键重试或 worker 换租约而丢失安全屏障；代价是认证读路径增加 Gate 查询，production composition、生命周期、迁移、告警和运行手册必须共同演进。本决定只固定架构合同，不表示实现、测试、部署完成，也不关闭 `IDENTITY-S10.1-REV-R01-F002`。

### 约束

- `PasswordResetOperation`、exact pending TemporaryCredential 与 recovery job/outbox 必须在 SMTP 调用前同一事务建立；job 在 SMTP 最迟结束时刻与显式安全余量前不得领取。
- operation 是安全结果权威。非终态、未知、非法、不可读、缺失但关联证据表明本应存在，或 operation/job/credential/idempotency 关系不一致时，Gate 必须 fail closed；从未发生 password-reset 且无关联证据的用户不得被误锁。
- Login、PasswordChange 与 Session resolve 是强制消费者：Gate 未 clear 时不得验证或接受凭据、consume TemporaryCredential、写永久密码、创建/轮换 Session 或返回 Principal。
- `IdempotencyRecord` 不得单独放行。只有 terminal operation 与 exact credential、能力后置及稳定响应快照一致时，才允许 replay 对应成功或失败结果。
- worker 使用 PostgreSQL 数据库租约、过期接管、显式退避、自动重试上限与批量领取上限；这些数值必须由部署配置和容量证据确定，不得以代码默认值替代。`dead_letter` 不解除 Gate。
- worker 必须按 identity design r34 固定锁顺序，在同一事务原子收敛 exact credential、派生或目标 Session、consumed 后永久密码能力、幂等快照和 AuditEvent；请求线程有限 cleanup 只能 enqueue 或加速，不得直接绕过 operation 判定。
- SMTP 未受理时原凭据和 Session 状态保持不变；到期仍为 `prepared` 的 job 只能安全失效该值并收敛失败，不得由 worker 重发邮件。
- ADR-0011 的 SMTP 受理成功判据与残余窗口、ADR-0015 的作用域化幂等、ADR-0016 的事务性追加审计、ADR-0018 的恢复后 fail-closed 基线继续有效；本 ADR 只补充 password-reset 的 durable operation/Gate/worker 合同。

### 影响

- **长期 owner**：identity 领域拥有 operation、job、Gate 语义和恢复事务；framework/architecture 与 production composition 只消费启动、生命周期、迁移和关闭合同，不复制领域正文。
- **数据与查询**：新增 operation/job 持久模型、状态约束、每用户 unresolved 唯一约束、credential/幂等 generation 绑定及 Gate/worker 索引；AuditEvent 继续使用既有 PostgreSQL 权威存储。
- **认证消费者**：Login、PasswordChange、Session resolve 必须在凭据或 Session 能力判定前消费 Gate；任何旁路 reader 都属于安全违约。
- **运行与部署**：同一 `han-server` 内增加受控后台 worker；worker 启动和健康属于 password-reset writer/认证路由开放前置，关闭时必须先停止领取并收敛 worker，再关闭数据库连接池。
- **流程状态**：B6C/S10.1 继续 `[-] blocked-for-design-revision`，B6D 不得启动；F002 继续 Major OPEN，等待后续重新规划、独立评审和 Lead Gate。

### 升级与回退

1. 只允许前向迁移：先创建 operation/job 表、CHECK/FK、部分唯一约束及 Gate/worker 索引，并使新 reader 能识别全部合法状态。
2. 部署升级时先停止旧 password-reset writer；禁止旧 writer 与新 Gate/worker 混跑。
3. migration 完成且 production composition 已同时装配三个 Gate、租约 worker、配置、告警与运行手册后，才可开放 password-reset writer 和受影响认证路由。
4. 关闭顺序固定为停止新接入、停止 worker 领取并等待其 loop 收敛、再关闭 PostgreSQL pool；不得先关 pool 后遗留无法持久化状态的 worker。
5. 只要存在任何 `PasswordResetOperation` 或 recovery job 记录，就禁止回退到不理解该 Gate 的旧版本。回退只能在证明表为空、无旧 writer、无未决能力且回退版本不会绕过 Gate 时评估；否则必须前向修复。

### 引用记录

- 2026-10-24 架构修订 `architecture-ir01-r25` 采用本决定，补充 operation/job/worker 的系统位置、三类认证消费者及升级、关闭与禁止回退合同。来源：本轮用户明确授权并指定方案 A。
- 2026-10-24 identity 详细设计修订 `identity-auth-design-ir01-r35` 引用本决定；保留 r34 机制、F002 OPEN 与实施 Gate，仅把 ADR 候选同步为已接受决定。来源：本轮用户明确授权并指定方案 A。
- 2026-10-24 identity Story 修订 `identity-auth-stories-ir01-r11` 引用本决定；同步 S10/K3/公共验收的架构来源，不创建任务或解除 B6C/B6D。来源：本轮用户明确授权并指定方案 A。

---

## ADR-0026: Password-reset dead-letter requeue 采用 Windows 实际进程 SID 与离线审批签名 (2026-10-28)

**目的**: 接口设计 | **仓库**: han

**修订**: adr-0026-r04

**r04 当前采用来源（2026-10-28）**：Lead 在既有用户明确“采用建议/继续”授权下，于 `2026-10-28T18:42:13.964885+08:00` 实际采纳 FD Phase A 排名1整包，含数值 primary/全部 fallback 限制。永久证据为 `.spec/wip/identity-auth/evidence-s10.5/cleanup-contract-adoption-r01/lead-adoption.json`（SHA256 `3ED915618A722E910835FB39D990DAF932786A4131EE6E87B131F8238ED5B84A`）、同目录 `index.json`（SHA256 `4174DEBFE8CF9C34BFB2ACBE58A10E1AD0E40B6CCF6C85E99C7091B0D2F246F6`）及 `fd-phase-a/consult.md`（SHA256 `0591453C50F13F4161155E6745C7B377D064965995F99DF0B278E5172697570F`）。Phase A 文中“未采用”是拟稿时历史状态；当前以 Lead 采纳记录为准。此次明确改变维护公开预算语义和 DSN 支持面，替代原先到 cutoff 所有 SDK/进程必须完成的承诺；不是说旧 r48 已满足原承诺。原CODE两项HIGH及Root gap、历史Design R02/E431输入证据保持原件，未关闭finding或预签Review/人的accepted/checkpoint。下列r03/r02来源是当时历史，当前语义以r04为准。

**历史 r03 / F002 作者响应（2026-10-28，当时修订范围）**：依据正式 Composite Review R01 的 `S105-DES-R01-F002`（`.spec/wip/identity-auth/review-design-s10.5-r01.md`，SHA256 `FAA1EAA42C2CF3AD8495611E23EF24BD731B5641D51574FE5713FE6D817EBA01`），Lead 在既有用户采用建议继续授权下明确保留七项必需输入。此次仅补齐唯一配置来源/缺失非法/DSN前后校验清单，复用既有 connection/shutdown/recovery 语义，不重新选择信任方案或 one enqueue。F001/F003 仍由 FD 处理；本修订不改变其输入取消、有界读取或预算合同，不以作者整改声称独立 Review PASS。

**状态**: 已接受 | **确认来源**: 本会话用户已明确授权“待确认/决策采用建议继续”，Lead 于 2026-10-28 读完最终咨询并明确选用 Windows 实际 owner SID + 单独 approver Ed25519 签名、仅一 requeue；随后明确授权本角色落此永久决定。此状态仅表示决定采用，不是 architecture/feature 成果 accepted/checkpoint、独立 Review 或生产运行批准。

**问题**: 既有 S10.5 如何以最低充分、可验证的执行与审批权限，仅把 dead-letter job 重新排队，且不伪造 lease、不改变 ADR-0023 的安全结果权威？

**决策**: identity 专用维护入口以 Windows 实际进程 owner SID 识别 executor，以部署登记的 Ed25519 公钥验证另一主体的离线审批；签名绑定执行主体、动作、目标、完整 before、时效和 requestID；一份审批/同 requestID 最多一次已提交重排（one enqueue）。User-first 事务仅重排 job 并追加真实 AuditEvent，原 worker 再领取合法 lease，后续自动尝试遵循各运行 worker 的现役显式 MaxRetries/退避配置。协议公共合同以本 ADR 为唯一 authority；FD 只做局部 Go/CLI 映射，不维护第二份可分叉的字段、版本或签名字节定义。

### 现状 / 痛点 / 改造 / 效果

**首次r01设计时点**：当时的 `PasswordResetOperator.Execute` 接收worker lease，Claim排除dead_letter；S10.5仍是维护入口候选，尚未交付可信运维/审批链。该段保留首次问题来源，不描述当前部分实现是否完成。ADR-0023 继续以 operation 为能力安全结果权威，dead-letter 不解除 Gate/hold；ADR-0008 要求业务模块归属及 ENV 单一配置源。

直接传 role/proofref、借业务 admin、修改 SQL 或构造 lease 都不能补足真实授权。这里仅借鉴 RFC 8032 的 Ed25519 签名和 Windows OS token 的主体语义，不采用完整 IAM、CA、在线审批服务或新的恢复 authority。

| 方案 | 实现机制 | 优势 | 劣势 |
|---|---|---|---|
| Windows SID + 单审批签名（采用） | 实际进程 token 对应 SID；受控部署授予执行用途；另一主体签完整动作 | 无 executor key；可在 DSN 前验证；标准库足够且实际执行主体可核 | 维护入口先限 Windows；依赖受控部署隔离；不提供即时在线撤销或自然人在场证明 |
| 双主体离线签名 | 注册 key 分别签审批与执行意图 | 平台无关，无 OS 身份解析 | 多一套 key/签署步骤；不能证明实际 CLI 进程 owner；已有 SID 接缝后非最低方案 |
| 单主体自批签名 | 同主体兼审批和执行 | 软件最少 | 无审批执行分离；本次未采用自批制度 |
| HMAC 共享秘密 | 部署共享 key 验 MAC | 标准库可实现、机制短 | verifier 同时持有签发能力，运行时新增签发秘密，审批边界较弱 |

本项目当前固定 Windows 工作区且已有可核的标准库 SID 接缝，因此选择第一项。它不新增联网 authority、运行时私钥、第三方依赖、公共 REST 或通用审批框架；代价通过Windows-only、按次受控进程、显式配置及真实在途排空合同约束；按次不承诺硬进程寿命。软件设计与隔离合成验证不以已有生产 key 为前置；实际部署与人员/key 归属仍须真实交付。

**r02 纠正来源**：TE r02 提供 MaxRetries 从 2 升为 5 的真实代码反例，Lead 已核 worker.settle/Fail 并明确采用 one enqueue，而非 exactly-one attempt。原“一审批最多额外一次尝试”不成立；本修订不改变既有 worker 自动策略，也不承诺恢复或冻结其重试预算。

### 1. 身份、用途与部署信任边界

维护入口仅在 `runtime.GOOS=windows` 时启用，实际 executor 为 `"windows-sid:" + os/user.Current().Uid`，按原始规范 SID 字串精确比较。禁止用 Username、Name、USER/USERNAME、请求 subject、业务 admin 或 ENV role 代替 OS 身份；不支持其他平台 UID/用户名 fallback。Current 失败、Uid 空或非规范 SID、实际主体未获该动作授权时 pre-DSN 拒绝。其他平台普通 server 不受此维护能力限制。

本机标准库 `lookup_windows.go` 的 current() 经 runAsProcessOwner、OpenCurrentProcessToken、GetTokenUser、Sid.String 取得进程 owner；它排除线程 impersonation。`os/user.Current` 首次调用会缓存，本合同只承诺身份固定的按次进程 owner 快照，不把再次调用称为刷新 token；本 binary 不自行改变或模拟执行身份。

主体 namespace 固定 `windows-sid:<SID>`。SID 为 `S-1-A-B...`：十进制 authority A 在 0..2^48-1，1..15 个 subauthority 各在 0..2^32-1，数字除 0 外无前导零、无空白或大小写替代。请求的 executor_subject 必须同时匹配实际 OS 主体与受控执行 allowlist。approver_subject 由受控 key 记录推导，必须不同于 executor；不允许同主体兼任两用途。审批证明 key 持有人意图，不证明自然人不同、MFA 或人在终端。launcher 若以 service SID 运行，审计 executor 就是该 service，不冒称提交请求的人。

可信部署渠道须用受保护的一次性 launcher/job 启动固定 binary、固定进程 SID，注入固定 trust/DSN 环境，仅转交 stdin 请求；不透传调用方 ENV、任意命令、可执行路径或配置路径。普通调用方不得修改 binary/launcher/trust、覆盖真实 DSN/schema/search_path，也不得取得目标库直接写凭据自行启动替代进程。ENV 自身不提供真实性；有权替换部署或直接管理数据库的人处于信任边界内，本决定不声称防御其恶意替换。该物理隔离是必须实际交付的部署合同，不是已经存在的事实。

### 2. 唯一配置位置与严格编码

维护路径的完整必需输入恰为下表七项 ENV：三项维护专用值加四项既有值，全部由同一可信部署渠道注入，没有 argv/stdin/请求/文件 fallback 或代码默认。普通 server 不因本能力新增三项维护专用要求，仍消费其原有完整配置。下表不表示提前读取 DSN；六项非 DSN 输入先完成内在校验，DSN 延后到全部 pre-DSN 授权检查成功后才访问。缺失、空白、非法、溢出或冲突均 fail closed，精确时序见表后。

| ENV | 唯一含义与语法 |
|---|---|
| `HAN_PASSWORD_RESET_MAINTENANCE_TRUST` | 最多 16 KiB 的严格 UTF-8 JSON 对象，字段恰为 `version,deployment_id,epoch,approver_keys,executors`；version 固定 `han-password-reset-maintenance-trust/v1` |
| `HAN_PASSWORD_RESET_MAINTENANCE_MAX_TTL` | 正的 Go duration 字符串，可由 time.ParseDuration 精确解析且不溢出；部署建议显式 `15m`，没有默认。请求实际有效窗不得超过它 |
| `HAN_PASSWORD_RESET_MAINTENANCE_OPERATION_BUDGET` | 正 duration，部署建议显式 `30s`，无默认。确定本invocation不可变业务/合作取消绝对deadline D，锁等待/多层helper不重启D；到D停止新业务，不重置签名TTL。不是全部SDK/CPU/native完成或进程返回的硬上限，实际join可能超过D及清理截止 |
| 既有 `HAN_DATABASE_DSN` | 必填、非空白；全部pre-DSN检查成功后读取并由现役解析器解析，再在connect前校验 primary及全部fallback的host均为数值IPv4/IPv6字面量。hostname、隐式localhost或其他非数值host一律 maintenance.invalid_config/2，OS lookup/connect=0；任何fallback非法都拒绝整个配置，不等它被选中。保留现役ambient PG/外部来源拒绝；普通server DSN支持及TLS验证/SNI不改 |
| 既有 `HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET` | 必填规范十进制正 int32，范围 `1..2147483647`；复用现役正整数 parser，拒绝空白、符号、前导零、非数字、零/负数及溢出，无默认。仅作为本维护进程独立 pool/permit 的连接预算，不构造 worker；正值与解析后 `pgxpool.Config.MaxConns` 的关系另做交叉校验 |
| 既有 `HAN_SHUTDOWN_BUDGET` | 必填Go duration，无前后空白，包含端点的 `1s..60s`，解析/溢出/越界拒绝，无默认。维持K=D+ShutdownBudget及每次cleanup的较早deadline，用于合作取消/中断可取消I/O；不是全部SDK/CPU/native完成或进程返回保证。非取消阶段真实join可越期甚至一直等；不延长D/TTL，不触发server worker coordinator |
| 既有 `HAN_RECOVERY_MODE` | 必填且原文精确等于 `normal`；缺失、空白、大小写/前后空白变体、`maintenance` 或任何其他值均 pre-DSN 拒绝，不默认 normal。这里只排除全局恢复冲突，不授予维护权限 |

maxTTL/operationBudget 只在各自专用 ENV，连接预算、shutdownBudget、recoveryMode 也只在表列既有 ENV；它们不进入 trust JSON、签名请求或第二配置位置。15m/30s 仍为显式部署建议值，无代码默认、隐藏上限或生产容量承诺。

**失败顺序**：先在触及 DSN 前校验三项维护专用值、connectionBudget 的正 int32 内在有效性、shutdownBudget 的 `1s..60s` 及 recoveryMode 精确 normal，再完成实际 SID/trust/请求/签名/时效授权。前述任一失败，DSN getenv/parse、connect/ping/业务访问均为 0。之后才读取 DSN：缺失/全空白时 parse=0；DSN语法、ambient PG或外部源拒绝由现役解析器报告，parse=1、connect及以后=0。解析成功后，primary及全部fallback的数值地址支持检查在任何Lookup/Dial/connect前完成；存在hostname或隐式localhost则invalid_config/2、parse=1、OS lookup/connect及以后=0。唯一依赖 `MaxConns` 才能判定的连接预算错误仍是正 connectionBudget > 解析后 MaxConns，此时 parse=1、connect/ping/业务访问=0；内在非法预算绝不为取得 MaxConns 而提前 parse DSN。合法值通过交叉检查后方可 connect/ping。

**其他配置零加载**：维护配置读取器只消费上述七个 `HAN_*` 输入，不调用普通 server 的全量配置或整个 worker 配置 loader；SMTP/Article/Argon2、bootstrap/Session/Journal 及 worker lease/retry/MaxRetries/batch 等无关配置和秘密均不读取。DSN 解析内部现役 ambient `PG*` 拒绝 guard 保持原合同，只用于拒绝外部来源，不把其值采纳为第八项输入；不得借此访问其他 server 业务配置。七ENV名称/来源、数值合法范围及15m/30s显式建议不变；本r04调整operation/cleanup截止与完成保证的含义，不延长D或重置签名TTL，也不以放大budget消除原反例。

**F002 对齐验证要求**：完整合法最小环境须显式提供七项；逐项缺失/空白/非法以及 connectionBudget 的零/负/溢出须按上述计数拒绝，shutdownBudget 覆盖端点与越界，recoveryMode 仅 normal 允许。正 connectionBudget > 解析后 MaxConns 单独验证 parse=1/connect=0；无关业务键读取计数为0，并保留现役DSN外部来源拒绝。维护配置还须分别覆盖primary/fallback数值IPv4/IPv6、primary hostname、隐式localhost与仅fallback hostname：非法支持面必须parse=1、connect=0/OS解析=0且退出2。此处是待执行设计要求，不是Go/PG或DNS运行证据。

**维护数值端点支持面**：这是相对原hostname支持的显式兼容性缩减。解析后primary与全部fallback逐项在本地校验为数值IPv4/IPv6，禁止维护路径调用net.DefaultResolver或底层OS GetAddrInfoW；私有LookupFunc只校验并返回已验证字面量，私有DialFunc仅使用已验证数值host/port，不能用wrapper结束冒称OS DNS线程已join。不得增加公共配置字段、ENV、依赖或新resolver provider；不把支持边界缩减扩散到普通server。TLS证书验证及既有ServerName/SNI规则保持，不自动关闭验证、切换认证或改证书匹配策略；部署是否能提供匹配数值端点与证书当前unknown，投用前需真实证据。

deployment_id、epoch、key_id 为 1..64 字节 ASCII token，语法 `[A-Za-z0-9][A-Za-z0-9._-]*`。epoch 是精确匹配的部署信任版本，不从时间推算；部署必须确保撤销/灾备后的新 epoch 不复用旧值。

approver_keys 是非空数组，每项字段恰为 `key_id,public_key,subject,action,not_before,expires_at`；public_key 是无 padding 的规范 base64url，解码恰好 32 字节；subject 为上节 SID namespace；action 固定 `identity.password_reset.requeue`，所在集合授予 approve 用途。executors 是非空数组，每项恰为 `subject,action,not_before,expires_at`，集合授予 execute 用途，action 同上。时间均非负 int64 UTC Unix 秒，expires_at 大于 not_before。禁止重复 key_id、公钥、同集合重复 subject，以及两个集合重叠 subject；不提供通配符、业务角色继承、allow-all、自选 verifier 或请求声明的 policy。

配置严格解析必须检查全部字段 presence、类型、重复/未知字段及 UTF-8；拒绝 null 必需字段与非规范公钥/SID/token，不把宽松 json.Unmarshal 成功当作配置有效。ENV 的键顺序和无害空白不参与签名；不可因此允许重复字段、别名或多个生效来源。

### 3. 请求、before 与签名字节（公共 v1 合同）

全部授权材料仅经有界 stdin 输入一个 envelope，总 UTF-8 字节数不超过 16 KiB，读到 EOF；字段按顺序恰为 `payload,approval_signature`，两者都是无 padding 的规范 base64url 字符串。payload 解码后的 P 不超过 12 KiB；signature 解码恰好 64 字节。不接受请求公钥、trust、role、DSN 或 alg。CLI 只选择非授权性的模式/动作；目标、主体、理由、before 全来自签名 P。既有候选 `expectedBeforeDigest` 参数不得作为独立 authority、替代完整 signed before 或覆盖它；FD须据此收敛 CLI，不从 argv 读取授权材料。

P 的字段声明顺序固定如下；每项都必须出现且大小写精确，禁止未知字段：

| 顺序 | 字段 | 类型及语义 |
|---|---|---|
| 1 | version | string，固定 `han-password-reset-requeue/v1` |
| 2–3 | deployment_id, epoch | string，精确匹配本次受控配置 |
| 4 | request_id | string，规范非零小写 UUID（8-4-4-4-12 十六进制），每次审批独立，不得复用 operation_id |
| 5 | action | string，固定 `identity.password_reset.requeue` |
| 6–7 | user_id, operation_id | string，规范非零小写 UUID，唯一目标，二者不得被 CLI 覆盖 |
| 8 | approver_key_id | string，定位受控 key；不接受请求自报 approver role/subject |
| 9 | executor_subject | string，必须匹配实际进程 SID 与已授权执行记录 |
| 10–11 | issued_at, expires_at | 非负 JSON int64 UTC Unix 秒，正有效窗且 <= maxTTL |
| 12 | reason | 非空且非全空白 string，<=512 UTF-8 字节，无控制字符；签名后不 trim/正规化 |
| 13 | evidence_ref | 同上，<=256 字节；是被审批方背书的审计引用，不读取/联网/执行该路径或 URL，不声称软件已核外部材料真伪 |
| 14 | before | 下表完整对象，禁止用摘要替代 |

before 字段顺序固定；N 表示 JSON null 合法，所有字段仍须 presence 校验。所有 UUID/SID 非 null 时使用上述规范；状态/模式引用现役 identity 枚举，不扩展状态空间。DB 时间使用 UTC RFC3339 固定 6 位小数及字母 Z（例如 `2006-01-02T15:04:05.000000Z`），无 offset/变长小数替代；非负整数均为 JSON int64。

| 顺序 | before 字段 | 类型 |
|---|---|---|
| 1 | user_status | string |
| 2–3 | user_auth_generation, user_execution_generation | int64 |
| 4 | user_current_operation_id | UUID string / N |
| 5–6 | credential_mode, integrity_hold | string, bool |
| 7–9 | integrity_hold_operation_id, integrity_hold_reason, integrity_hold_at | UUID string / N，string / N，DB time string / N |
| 10 | user_updated_at | DB time string |
| 11–13 | operation_state, delivery_state, settlement_state | string |
| 14–16 | operation_execution_generation, temporary_credential_id, operation_updated_at | int64，UUID string，DB time string |
| 17–19 | job_status, attempt_count, next_attempt_at | string，int64，DB time string |
| 20–22 | lease_owner, lease_expires_at, last_failure_code | string / N，DB time string / N，string / N |
| 23–25 | job_created_at, job_updated_at, completed_at | DB time string，DB time string，DB time string / N |

P 的唯一规范编码是按上述顺序的固定结构 `encoding/json.Marshal` 默认字符串转义、无 omitempty、无缩进、无尾随换行；不用 map/float。envelope 同样使用固定结构与此规范。接收者先做 UTF-8、精确字段名、presence、类型及字节上限校验，再重编码并逐字节比较原文；重复字段、大小写别名、缺失 nullable 字段、额外字段、null 必需字段、空白/键重排、非规范数字/转义和尾随内容均拒绝。nullable 字段缺失不能当成 null。canonical 检查必须覆盖 before；仅 DisallowUnknownFields 不足以拒绝重复字段。签名前后不做 Unicode 正规化或静默修剪。

精确签名消息 `M = ASCII("Han/identity-auth/S10.5/requeue-approval/v1") || 0x00 || P`，0x00 为实际单字节。仅使用 plain Ed25519 `ed25519.Verify(pub, M, sig)`；不使用 Ed25519ctx/ph、Options.Context、调用方 alg 或预散列替代 M。调用 Verify 前必须检查 public key=32 字节、signature=64 字节，避免非法公钥长度 panic。SHA-256 仅用于审计摘要。

授权检查为 `issued_at <= now < expires_at`，请求正窗口 <= maxTTL，且 approver key 与 executor 授权各自 `not_before <= now < expires_at`；按 UTC 真实时钟、不额外放宽 clock skew，时间/窗口/预算计算溢出即拒绝。当前部署 ID/epoch、动作、实际 SID、用途与独立主体均须满足。错误/过期/撤销不因已存在消费记录获得例外放行。

### 4. 事务、消费与审计的精确表示

maintenance 先选择模式，按 §2 七项清单及失败时序完成六项非 DSN 配置、Windows/真实 SID、全部请求结构/签名/用途/时效与主体分离检查，之后才读取/parse DSN、核primary及全部fallback数值host和正连接预算/MaxConns，再允许连接/Ping。早期失败的 DSN parse/connect/ping/业务写均为 0。不进入普通 server 的 config/prepare 链；不跑 migration/bootstrap、Journal、SMTP、Mailer、两 worker、业务 handler、listener、Serve、Reconcile 或 Start。新模式不等于 `HAN_RECOVERY_MODE`；灾备恢复模式冲突 pre-DSN 拒绝。

同一 identity operator 的 Requeue 路径取得一次本进程唯一 permit，开始事务后锁 User，再锁属于目标的 operation/job；内部直接调用也须完成相同验证。main 只读配置、处理 stdin 并装配，不承载领域 SQL。生产主体 resolver 不能由请求选择、不能只信传入 actor；私有测试 seam 不作为真实 OS 验证证据。

锁内先复核当前时间、不可变请求/受控 key/epoch/execute 授权快照，再用既有 `audit.Store.Get(ctx, tx, request_id)` 读取消费事实。不存在消费事件才逐字段比较全部实际 before，要求 operation/User/job 关系正确、job=dead_letter、lease_owner/expiry/completed_at=null、operation 非终态且 settlement=pending、hold=false、attempt_count<int64 最大值。缺行/陈旧/非法组合拒绝；只做排队 guard，不复制 worker 的完整 lineage/settlement classifier。

允许写集只有本 job 的 status=ready、next_attempt_at=max(旧值,now)、updated_at=max(旧值,now)，保留 attempt_count、last_failure_code、created_at；lease/completed 保持 null。不提前保护窗，不改 User/operation/credential/Session/Idempotency/hold/终态、不发送邮件、不伪造 lease。attempt_count 不减、不重置。原 worker 之后正常 Claim/Renew 再 Execute，自动尝试次数与失败后的 retry_wait/dead_letter、退避严格遵循各运行 worker 的现役显式 MaxRetries/RetryBackoff；不声称 exactly-one attempt，不恢复或冻结预算。例如旧 MaxRetries=2 已使 count=2 的 job dead_letter，后以 MaxRetries=5 的 worker 领取会变成 count=3，失败时仍可 retry_wait 并继续尝试，不能由“保留 count”推导只再尝试一次。再次 dead_letter 后若要重新排队，须新合法审批、新 requestID 及匹配新 before；旧 requestID 只能返回原消费事实，不重新排队。

同一事务调用现役 `audit.Store.Append`：ID=request_id（复用全局 UUID PK），OccurredAt=本次 UTC clock，ActorKind=system，ActorUserID=null，ActorSystem=实际完整 `windows-sid:<SID>`，ActorRole 固定 `password_reset_requeue_executor`，Action=`identity.password_reset.requeued`，TargetType=`password_reset_operation`，TargetID=operation_id，Outcome=`succeeded`，TraceID=request_id，ErrorCode/SourceFingerprint=null。现役 `0001_shared.sql` 的 actor_system/actor_role 为 text，无长度上限/业务 role 枚举；system CHECK 仅要求非空 system 且 user_id=null。`audit.Store.Append` 没有额外截断或角色校验，SID可直接保存，不改 schema/audit 包。

专用 Metadata 所有值均为 string，字段恰为下列 13 项；不得放解析后的 before/after 数字对象：

| 字段 | 精确含义 |
|---|---|
| schema_version | `identity-password-reset-requeue-audit/v1` |
| deployment_id, epoch | 原受控部署绑定 |
| approver_key_id, approver_subject, executor_subject | 受控 key 定位及推导的两主体，executor须等于ActorSystem |
| payload_b64, approval_signature | 原规范 P 与64字节审批签名的规范 base64url |
| payload_sha256 | 对原 P 的 SHA-256，64位小写hex |
| after_b64, after_sha256 | 实际写后完整 snapshot 的规范 JSON 字节（结构/字段顺序同 before，上限12KiB）的 base64url与SHA-256 |
| result_b64, result_sha256 | 下述规范 receipt 字节（上限1KiB）的 base64url与SHA-256 |

receipt 字段顺序恰为 `version,request_id,outcome,operation_id,job_status,audit_id`，所有值都是 string；version=`han-password-reset-requeue-result/v1`，outcome=`applied`，job_status=`ready`，audit_id=request_id，其余精确匹配本请求。编码同 P 的固定结构 Marshal。after 除 job_status、next_attempt_at、job_updated_at 外必须逐字段等于 P.before；这三项分别为 ready、max(before.next_attempt_at,OccurredAt)、max(before.job_updated_at,OccurredAt)，使用与 job 写入相同的 UTC 微秒时刻 OccurredAt，不能在审计时另取一个不匹配时刻。metadata 序列化后最多64KiB，解码各字段前同时检查自身上限、base64规范和字段 presence/类型，不允许任意大的输入触发分配。

`audit.Store.Get` 现役 decodeMetadata 将 JSON 数字反序列化为 float64；本合同通过字符串 carrier 保留原精确字节。identity 读取时要求 metadata 恰为上述字符串字段，strict 解码 payload/after/result、校验规范字节及摘要，并对照 Event 全部绑定；禁止从 lossy map、float64 或重新拼出的数字恢复 P、int64 snapshot 或 receipt。reason/evidence_ref/before只从保留的 P 解码，不另存竞争对象。

已有同ID事件只有在动作/target/actor/metadata v1、原 P/签名、两主体、部署/epoch、摘要及规范 receipt/after 绑定全部一致时，返回 `already_applied` 的调用处置及原 receipt/after，零重复 UPDATE/Append；原 receipt.outcome 仍是 applied，不改历史为 already_applied。重复读取可以复核原事务提交，不表示 job 目前仍ready（worker可能已领取/完成），也不重新证明认证能力安全。同ID异payload/target/action、事件损坏/无法严格解析均冲突或 fail closed，不当作“未执行”。UUID全局PK裁决跨目标ID竞争，目标User锁串行同目标竞争；事件只证明此次动作已消费，operation仍是唯一安全结果权威。

提交前的审计/条件UPDATE非1行/ID冲突/时间或context失败，按原事务规则尝试回滚并真实完成资源处理；commit前再检时效。明确已提交的事实和原AuditEvent不能因随后D到期或cleanup越期改称回滚。任何commit错误仍按maintenance.outcome_unknown/5，不猜回滚、不清hold、不自动补查/再写、不调用或复制现役settlement commit classifier。后续有效同请求可按原审计记录重放。CLI只输出稳定结果类别/请求与审计引用；错误分类固定为输入/配置、授权拒绝、状态冲突、操作失败、结果未知，Go局部映射及其余既有CLI错误码不变，不改公共REST错误体系；不输出原请求/签名、before/after、密码/摘要、完整邮箱、Session、DSN、SQL或内部cause。只有实际资源/阶段join结束后才作最终输出裁决；不得先输出applied或already_applied再发现清理越期。维护失败不启动server，完整资源/退出规则如下。

**D、K与真实完成**：D为原不可变operation绝对deadline。到D不再发起认证、Acquire、Ping/Requeue、业务SQL、提交核验或任何新业务；不重置审批TTL/D，不因清理再开业务窗口。已经发往DB的在途事务可能提交，保留原unknown处理。清理令K=D+ShutdownBudget，每次实际cleanup取消deadline C=min(cleanup_start+ShutdownBudget,K)，时间加法溢出拒绝，不因多层helper、错误、SDK关闭或pool-close重置C/K。context.WithoutCancel仅隔离原取消以便终结资源，必须带上述C；它不是无界I/O许可，也不是硬进程完成计时器。

C是对可取消I/O实施合作取消/中断的截止，不是所有SDK、CPU、native调用或整个进程在C/K前完成的保证。已开始而不响应取消的SDK/native阶段必须实际结束并join；不可用脱离goroutine、超时提前return、parent Kill或假造统计归零替代。真实join可能越过C/K；下层若永不返回，本进程也可能一直等，没有硬进程寿命上限。这是本次公开接受的可用性代价，不把原r48的严格完成承诺说成已满足。不得擅自关闭SCRAM、改认证方式/迭代数、借SDK 2m/15s重新计时或引入不存在的可取消KDF API。

**资源完成与TLS独立义务**：root/operator消费同一D/K/C，停止新阶段后仍须实际等待constructor及stage引用退出、Rows/tx终结、借用连接与permit真实归还、自有callback/timer停止并join，健康成功SDK连接按原互斥顺序触发SDK Close→CleanupDone，最后pool.Close实际完成；raw socket.Close、Release、Acquire调用者已取消或空socket表都不能冒充这些完成事实，不与正在构造/使用连接的阶段并发Close。cleanup只终结已有资源/回滚，不新增DB业务写或timeout AuditEvent。

CODE cleanup R01-F002的可避免网络/TLS越期仍须在现有transport owner内修复：cleanup期对SetDeadline/SetReadDeadline/SetWriteDeadline的写入含清空/放宽均在同一owner锁下裁至不晚于C，TLS内部now+5s不能覆盖共同截止；如安装同C的raw-interrupt回调，也须真实join。健康路径仍先SDK Close，raw断开不能伪造CleanupDone。允许不可取消CPU/native阶段越期不豁免网络5s窗口，不把一般ctx存在当成完成证明。本ADR记录义务，不宣称Root实现或CODE finding已关闭。

**粘性越期与退出**：一旦C到期/观察到cleanup overrun即粘性保存，实际join结束时还要用保留单调时钟分量的最终完成时刻再次核对C，不能因计数最终归零或timer回调较晚而清除越期。原本apply/replay成功若最终清理越期，join之后返回maintenance.failed/6、stdout空；原maintenance.outcome_unknown/5保持优先，其余已有非零结果保持原主错误及清理错误附加规则，不统一覆盖成6。明确commit与原审计仍是已提交事实；绝不为超时增加业务UPDATE、补查提交、重排或写timeout审计。原operationBudget取消与成功清理是否符合新合同，必须在同一invocation按真实D/C/end事实分类。

Lead已采用的预算/端点方案对照如下；不重新开放选择：

| 方案 | 机制 | 优势 | 劣势 |
|---|---|---|---|
| 1（采用）合作截止 + 真实join + 数值端点 | D停止新业务，C取消可取消I/O，所有已开始阶段实际结束 | 现有七路径/依赖可承载；不遗弃SDK阶段，不启动OS DNS | 放弃硬退出上限，可能无限等待；维护hostname不兼容，部署数值端点/证书待证 |
| 2（未采用）合作截止但保留hostname | 对底层OS resolver完成明确免责 | 保持hostname配置兼容 | 返回后可能仍有本次OS DNS工作，降低无遗留保证 |
| 3（未采用）受监督进程隔离 | 另定强制终止/reap/unknown生命周期 | 可另设计进程寿命边界 | 新终止/资源语义；不等于SDK优雅join，超出当前采用范围 |

选择1保留当前真实SDK能力和无脱离工作的责任；用数值地址避免本invocation启动无法跟踪完成的OS resolver。无限等待及数值地址兼容性损失是公开接受的代价，不能暗增ShutdownBudget、改变SCRAM或以通常4096轮/单次benchmark恢复硬完成承诺。

### 5. 撤销、恢复与验证边界

部署移除/禁用 key、executor SID 或改变 epoch，使使用新配置的后续 invocation 拒绝；历史 verify-only key 不具有继续执行授权。每个维护进程使用启动时冻结的受控配置，不假装实时获知之后的撤销。已通过最终检查或发往DB的事务仍可能提交；显式30s建议值只确定业务/合作取消D，不保证数据库提交、全部SDK/native完成或进程返回时限。撤销排空必须以真实事务/阶段完成为准，可能越过K；下层不返回时无法宣告排空完成。

要宣告撤销彻底生效，可信部署先关闭新启动，终止或等待旧快照进程及其DB事务确认结束，再更新/启用新配置；在途unknown未查明时保留未决。没有在线撤销源，就不承诺无竞态即时撤销。

requestID消费依赖现役数据库审计事实，不承诺跨备份回滚仍防重放。灾备恢复期间部署禁用该入口并与恢复独占流程互斥，恢复后换不复用的epoch再开放。此维护模式仅同一次进程与server互斥，不要求停掉其他正常server的worker；它不实现或冒称canonical deletion的RecoveryOpenGate，不改变RecoveryRun/Journal/CAS或全局恢复。签名不提供机密性，stdin、传递材料及审计读取须限制权限。

软件实现及隔离合成验证不要求已有生产issuer/key。实际launcher/ACL、主体与key登记保管/签发、部署、撤销排空演练、第二真实SID运行验证均NOT RUN；仅用ENV换用户名不是第二主体测试，私有seam不是OS运行证明，合成签名不是自然人真实运维批准。

### 6. 消费者、七路径与来源

当前沿用Lead已采用的恰七个源码路径，r01–r03的“五文件足够”仅为当时历史，不再作为当前充分性结论；不新增第八文件/依赖：

| 路径 | 本轮合同消费者责任 |
|---|---|
| `cmd/han-server/main.go` | 同一root/七ENV/模式/数值primary及全部fallback校验、私有literal Lookup/numeric Dial、登记/transport截止、SDK/pool真实join、粘性overrun与原退出优先级；零领域SQL |
| `cmd/han-server/main_test.go` | root私有seam与真实SDK/协议peer等可授权层次核D/C/end、数值支持及TLS截止；不以普通mock或parent Kill冒真实完成 |
| `cmd/han-server/maintenance_stdin_windows.go` | 既有Windows原生输入adapter/所有权与取消合同保持，native调用需实际完成，不创造硬完成保证 |
| `cmd/han-server/maintenance_stdin_other.go` | 既有非Windows维护拒绝，普通server不变 |
| `internal/identity/password_reset_operator.go` | 专用协议/同类Requeue与原事务，消费相同D/K/C及错误优先级，Rows/tx/连接/permit真实收束；Execute/安全结果权威不改 |
| `internal/identity/password_reset_operator_test.go` | 原协议/one enqueue和合作cleanup/unknown/越期结果分支，不冒真实PG或生产主体 |
| `test/integration/identity/password_reset_maintenance_test.go` | 既有retain-only carrier；未来真实CLI/PG事实与observer，不修改共享harness或autoDROP |

公开Go Config仍仅TrustJSON、MaxTTL、OperationBudget、ShutdownBudget四字段，Validate/Requeue/opaque D及既有CLI、codes、wire/trust/签名/审计schema不变；本修订明确改变预算语义和维护DSN支持面，不能称“配置合同完全未变”。root零领域SQL，不改worker/store/audit包/迁移/公共REST/ADR0023核心，不增第二classifier。七路径边界不等于本次源码写权；本角色仅写三文档，FD负责其两文档、TE待稳定输入后规划r23，不预记其未来采用。

本轮验证接力保留原cutoff失败原件/原时间采样：真实SDK SCRAM反例应在C/K仍记录活跃constructor/stage、最后实际join并以单调end证明越期非零，不能删除失败断言改成“提前return即绿”；TLS背压必须证明内部5s不能放宽C，健康SDK Close→CleanupDone→pool及callback/timer均实际完成。数值IPv4/IPv6 primary/全部fallback正反例要证明post-parse拒绝与OS lookup/connect=0；受控lookup gate不是Windows DNS实测。原Case ID及分层由TE修订，不新增假Run。新实现/合成验证/真实PG-CLI/部署证书、第二SID、撤销排空及E2E均未由本次作者静态修订验证；CODE finding及Root-ready/combined CLI-PG的实际状态以[正式cleanup复审R02](../../../../.spec/wip/identity-auth/review-code-s10.5-cleanup-r02.md)、后续正式复审及[Tasks](../tasks/tasks.md)为准；本作者文档修订不关闭finding或替代代码复审。

最低验证须包含 MaxRetries 2→5 反例：先在原配置下得到 dead_letter/count=2；一份审批只提交一次 requeue 且 count 仍2；运行配置变为5后，由合法 worker Claim 取得 count=3 的 lease，受控 settlement 失败可进入 retry_wait，并按现役退避继续4/5等后续尝试。这不是第二次 requeue，不得把后续多次自动尝试判为审批幂等失败。任意这些阶段重放旧 requestID 都只返回原 receipt/after、零 UPDATE/Append；再次 dead_letter 后以新审批/新ID/新before重排，旧请求仍不写。以上为待实现/执行的验证要求，NOT RUN，不通过改worker或伪造lease证明。

FD消费本协议并收敛现有CLI；TE区分合成材料下真实实现验证、OS真实主体矩阵与部署/签发未验；部署负责物理信任隔离和撤销/恢复窗口。未来若需其他平台、实时撤销、自然人在场或跨备份nonce，必须重新确认边界，不在本次造通用authority。

可复核代码：`password_reset_operator.go` 的 Execute、User-first/审计和lease校验；`password_reset_worker.go` 的 Claim与MaxRetries；`migrations/0008_password_reset_durable.sql` job CHECK及operation绑定/终态trigger；`audit/store.go` 的 Append/Get/decodeMetadata、`0001_shared.sql` 的UUID PK/system CHECK；`main.go` 唯一root。业务依据为既有S10.5、identity r45 Story重置/重发与r44维护候选、identity boundary；首次落档时r46为FD在途局部设计，当时未作为AUTH已内化证据；本次固定r46的实际采用见下方引用记录。

标准库主证据：`C:/Program Files/Go/src/crypto/ed25519/ed25519.go` 的32/64长度、Verify plain/ctx/ph及非法公钥panic；`os/user/lookup_windows.go` current/runAsProcessOwner，`os/user/lookup.go` Current缓存，`os/user/user.go` Uid平台定义。本轮只读本机源码，未运行Go；官方算法依据为 [RFC 8032](https://www.rfc-editor.org/rfc/rfc8032.html) 与 [Go crypto/ed25519](https://pkg.go.dev/crypto/ed25519)，链接由Lead提供，不声称本轮独立浏览了网页全文。

**产出归属**：Arch-Designer Epicurus，Role ID `01a18106-6fce-7e60-8345-dd51ba9db6ff`，在本会话明确三文档授权内落档；是架构专业产出，不是独立Review。已有作者/评审/确认历史保持原来源，本次不添加成果checkpoint或生产批准。

### 引用记录

- 2026-10-28 `.add/project/knowledge/design/architecture.md` 修订 `architecture-ir01-r29` 实际采用本决定，仅同步维护入口/信任边界/短命部署运行及NOT RUN项。来源：本会话用户允许待确认/决策采用建议继续，Lead明确选用最终Windows SID+单审批签名并授权三文档落档；Epicurus按该授权写入；本轮又按TE r02反例及Lead明确采用的one enqueue纠正同步r02，不另计一条同来源引用。首次登记时FD尚未内化AUTH，因此当时未预记design/stories引用；其后实际采用另见下一条记录。随后同一architecture消费者的F002返工同步 `architecture-ir01-r30`，只补七项配置。2026-10-28本轮再按cleanup-contract-adoption-r01/lead-adoption.json已采用整包同步 `architecture-ir01-r31`，明确D/K合作语义、真实join/粘性越期与数值端点支持；以上均为同一实际来源的修订追踪，不新增引用次数。FD r49/TE r23尚未在此登记，不预记未来引用。
- 2026-10-28 Feature-Designer Ohm（`add-feature-designer`，Agent ID `01a180f6-2c3a-77e3-a286-d9bb373a62b7`）已在 `.add/project/knowledge/design-workspace/identity-auth/design.md` 修订 `identity-auth-design-ir01-r46`（SHA256 `CF3131198E757D13C5433ED4F5D12A721C5DD67F271090243C2219266ACED8D2`）及对应 `.add/project/knowledge/design-workspace/identity-auth/stories.md` 的 S10 maintenance 增量（SHA256 `E8D5EAD1F12D6C34F01730EB3761E6F49DEC3ECEFA776560C222C506AACC737E`；stories头部r16保留历史基线，本增量实际消费design r46）内化本ADR。来源：用户明确授权“待确认/决策采用建议继续”，Lead已明确采纳最终Windows实际owner SID+单独approver Ed25519、one enqueue方案；本会话Lead报告Ohm completed后已close writer，实读新节全文并按上述实际hash复制至 `C:/Users/J00010385/AppData/Local/Temp/han-s105-lead-_eqdfl5h/feature-fixed-r46`。本次登记现场核原文件及Lead固定副本hash一致；采用时点为该completed/close及Lead固定r46之后，登记时刻 `2026-10-28T06:32:07.934398+00:00`，不推定未提供的Agent完成精确时分秒。Epicurus仅据本轮两文件引用登记授权追加此实际来源；不改变协议/修订号，不记人的成果accepted/checkpoint、独立Review或生产批准，也不预记TE未来引用。
