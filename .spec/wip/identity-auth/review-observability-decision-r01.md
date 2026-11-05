# identity-auth F005 生产可观测性决策独立整体评审 r01

## 1. 评审身份、对象与结论

- 角色：非作者独立 Reviewer
- Workflow：`development-delivery / identity-auth S3.2 / F005`
- 工作树：`D:\AID\VSCode\Codex\Han`，`main`
- 基线提交：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审日期：2026-10-23
- 总结论：**NOT PASS**
- 严重度统计：HIGH 0，MEDIUM 3，LOW 0
- Task-Planner R 对账准入：**当前不可交付为稳定实施输入**。须先关闭本报告 3 个 MEDIUM finding，形成新的精确修订并完成独立复审；随后用户仍须对通过评审的精确 identity-auth design revision/digest 做软确认，才能进入 R 对账。

本结论不否认用户已授权“对待确认或决策项采用建议值”，但该授权不能替代技术正确性、一致性和可实施性审查。现役 `cmd/han-server` 仍是空 mux、单业务 listener；production composition、部署材料、真实 scrape、Dashboard、告警路由、runbook、演练和 E2E 均未实现或未执行，本报告不将其写成完成。

### 1.1 固定对象

| 对象 | 固定修订 | 开始 SHA-256 | 结论 |
|---|---|---|---|
| ADR-0022 + 决策索引 | `.add/project/knowledge/decisions/framework.md` ADR-0022；`.add/project/knowledge/decisions/README.md` | `B1E7F119EED23375D03606B0A0665C8BB54B4A5CF3F2013C111C36E4A769FD5B`；`68FA8F98B6991334B207D0A4C1871CD57A1F92286D1F0B78F3FDB933EDD03896` | **NOT PASS**：F001 |
| Architecture | `architecture-ir01-r20` | `15C93DABA6FF0EA4174F918FAC3748E513469225332D86A7B3D4BD8BB55F287E` | **NOT PASS**：F001、F003 |
| Identity design | `identity-auth-design-ir01-r19` | `C319D9742E5E6EF5712D585D8453BABCF990E5CF43355C95E490231BE2AFF790` | **NOT PASS**：F001、F002、F003 |
| Test plan | `identity-auth-test-plan-ir01-r04` | `2103241AB0A40F9EC874672714851667AF6FD783AC67C8D2709C83021AE02B63` | **NOT PASS**：F001、F002、F003 |

决策索引本身的 ADR-0022 编号、状态、领域文件、场景查询和引用次数与 ADR 正文匹配；索引没有独立 finding。

## 2. 两轮实际检查范围

### 2.1 Round 1：对象直接审查

逐对象核查：

- 推荐值是否被表述为诊断性实现基线而非 SLA/SLO/容量承诺；
- 30 天可查询留存、`250ms/250ms/2s`、Histogram buckets、四类告警、稳定 10 分钟恢复、Dashboard、路由和 runbook 的触发、调整与失败边界；
- ADR 状态、替代方案、主要责任领域、受影响消费者、未决项、引用记录和索引；
- provider-neutral、无新增 SDK/module 与项目内标准库 registry/exposition 的可实施性；
- `HAN_METRICS_ADDR`、三项预算、buckets、双 listener、装配 fail-closed 与运行时 observer fail-open；
- 固定低基数指标、敏感 denylist、PostgreSQL `AuditEvent` 权威性；
- test plan 对应用 composition、部署 Gate、真实部署验证/E2E、既有证据、本轮未重跑和未执行状态的分离。

通过项：

- 四对象的主要数值、配置名、四类告警条件、`Han / Identity Auth`、角色升级链和 `docs/runbooks/identity-observability.md` 基本一致。
- 30 天留存被明确限定为运行基线和部署 Gate，没有伪造法律/审计承诺。
- 实际 provider/receiver 账号、自然人排班、付费方案和容量调优保持未决定，没有被伪造成已落实人员或账号。
- observer 运行时失败保持原响应，与 production composition 缺依赖时启动 fail closed 的分层没有直接矛盾。
- `AuditEvent` 仍是权威审计；日志和 metrics 只承担诊断，不替代审计。
- 测试规划明确区分 prior evidence、artifact-present-not-rerun、计划/未执行以及真实部署 Gate。

### 2.2 Round 2：消费者与上游有界核查

核查了：

- `go.mod`：当前没有 observability SDK/module；
- `cmd/han-server`：现役 composition root 只有单业务 listener、空 mux、单 `Serve` 路径；
- `internal/platform/logging.go`、`http_security.go`；
- identity login outcome observer、metrics、failure auditor 接口及其写后 observation、低基数标签和取消合同；
- identity boundary、requirements 中登录、安全、审计、可服务性和容量边界；
- F005 code review r03 的既有证据与限制；
- tasks/Plan 中原 F005 合同和仍停留在 r18/r03/Q13 TBD 的陈旧映射；
- `deploy/processing-gate.env.example`、现有 `docs/runbooks/database-recovery.md`、`docs/runbooks/processing-gate.md`、`docs/testing.md`，确认 observability rules/dashboard/routing/runbook 尚不存在；
- 新增 application composition、部署材料和真实部署验证任务的建议范围；
- Markdown 引用、revision/source 字段、PlantUML 块成对、禁用 `linetype ortho`/字面 `\n` 的静态检查。

消费者核查确认现有 ports、`go.mod` 和 composition root 可以承接“标准库 adapter + 双 listener”的方向，但以下 3 个合同缺口会让不同实现产生互不兼容或不可验收的结果。

## 3. Findings

### MEDIUM F001：自研 Prometheus-compatible exposition 没有固定线协议，应用 Gate 可通过而真实 scrape 才失败

**位置**

- `.add/project/knowledge/decisions/framework.md:621-626`（ADR-0022 §2）
- `.add/project/knowledge/design/architecture.md:532-559,1211`
- `.add/project/knowledge/design-workspace/identity-auth/design.md:469,2232-2266,2437`
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:160,174`

**违反依据**

- `code-review-checklist`：公共数据格式和真实消费者兼容性必须有证据。
- `design-doc-writing-style` §J：接口必须给出与实现相称的输入、输出和异常合同，不能让实现者另猜。
- ADR-0022 选择“不引入 SDK/module、项目内自行实现 exposition”，因此线协议不是可留给第三方库隐含完成的细节。

**触发场景**

实现者按“Prometheus-compatible”自行输出文本，但没有共同约束具体格式版本、HTTP `Content-Type`、转义规则、样本终止、经典 Histogram 的累计 `_bucket{le=...}`、`+Inf`、`_sum`、`_count` 等。I-F005-15 只比较 bucket boundary 文本/快照，I-F005-16 只要求固定名称/类型/标签；真正的 parser/scrape 兼容性被推迟到 I-OBS-DEP-02。

**影响**

- application composition 和本地测试可能全部通过，部署后真实 backend 才拒绝 scrape 或无法计算 P95。
- 四类告警、Dashboard 和 production readiness 链条会在最晚阶段失败。
- 两个不同实现都可能自称满足“Prometheus-compatible”，但输出并不互操作。

Prometheus 官方文档区分 Prometheus text exposition 0.0.4、OpenMetrics 等受支持格式，并对 classic Histogram 的 `_bucket`、`le`、`+Inf`、`_sum`、`_count` 有具体语义；仅写“compatible”不足以替代这些实现合同。

**修复建议**

这是方向性合同选择，不由 Reviewer 代选：

1. 在 ADR 中明确选择一种首期线协议；最小方案可固定 Prometheus text exposition 0.0.4 且不做内容协商，或明确选择 OpenMetrics 及协商规则。
2. 同步 architecture/design：固定 HTTP content type、编码/换行/结束规则、名称与 label escaping、Counter/Histogram 序列化、经典 Histogram 的累计 bucket、`+Inf`、`_sum`、`_count` 和错误响应。
3. 同步 test plan：增加官方语法/parser 或等价协议级契约验证；真实 backend scrape 继续保留为部署 Gate，但不能成为首次发现格式错误的唯一验证。

**复审条件**

ADR、architecture、design、test plan 对同一线协议及协议级测试完全一致，且不引入未经批准的 SDK/module。

---

### MEDIUM F002：配置被同时定义为“容量评审后可调”和“只有精确基线值合法”

**位置**

- `.add/project/knowledge/decisions/framework.md:626,650,687`
- `.add/project/knowledge/design/architecture.md:146,557,1240`
- `.add/project/knowledge/design-workspace/identity-auth/design.md:859,1901-1904`
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:158,160`

**违反依据**

- `code-review-checklist`：设计/ADR/验收必须一致，配置缺失策略和消费者语义必须清楚。
- `abstraction-principles`：变化点应由一个清晰边界承接，不能同时存在互斥解释。

**触发场景**

容量实测后经评审决定把 Login observer 调为 `300ms`、failure auditor 调为 `3s`，或把 buckets 改为另一组正数、有限、严格递增值。ADR 与 architecture 明确允许评审后调优这些配置值；design §3.4.8 同样说可以调优。但 design 配置表要求 buckets 必须“显式等于”当前基线，test-plan I-F005-13/I-F005-15 又要求任何偏离精确值均启动失败。

**影响**

- Task-Planner/实现者无法判断 parser 应接受所有满足结构约束的值，还是把当前基线编译成唯一 allowlist。
- 若按 test plan 实现，环境变量不再是真正可调配置；任何已批准容量调优仍需代码修改。
- 若按 ADR/architecture 实现，现有测试验收会错误拒绝合法配置。

**修复建议**

须明确选择并全对象同步：

- 方案 A：配置 parser 接受任意正数有限预算及正、有限、严格递增且不重复的 buckets；发布/部署清单固定首期基线，容量评审后只改部署值。测试应包含精确基线、至少一个合法非基线值和非法矩阵。
- 方案 B：当前版本只允许精确值，未来调优必须先修订 ADR/设计并发布新代码；相应删除“配置值可经容量评审调整”的当前时态表述，说明环境变量仅用于显式注入而非运行调优。

**复审条件**

ADR、architecture、design 配置表和 I-F005-13/I-F005-15 对“合法值集合”只有一个解释。

---

### MEDIUM F003：双 server 的“有界 shutdown”没有预算、强制关闭和错误优先级合同

**位置**

- `.add/project/knowledge/decisions/framework.md` ADR-0022 §6
- `.add/project/knowledge/design/architecture.md:559,956-1011,1045`
- `.add/project/knowledge/design-workspace/identity-auth/design.md:2467,2500`
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md:163,322,324`

**违反依据**

- `code-review-checklist`：资源所有权唯一，失败路径必须收敛且不留下半完成状态。
- `design-doc-writing-style` §J：失败与恢复要说明谁清理、如何得知结果及可实施边界。
- no-hardcode/可服务性原则：运行超时不能成为来源不明的隐藏常量。

**触发场景**

metrics server 异常返回，而业务 server 正在处理长期或阻塞请求。test plan 要求“等待有界 shutdown”，但 ADR/architecture/design 只要求清理两个 listener 和 DB pool，没有定义：

- 使用立即 `Close` 还是优雅 `Shutdown`；
- shutdown 上界来自哪个配置或批准常量；
- 超时后是否强制 `Close`；
- 先停止接入、等 handler、关 listener、关 DB pool 的顺序；
- 原 server 错误与 shutdown/listener/pool 清理错误的保留优先级。

**影响**

- 实现可能无限等待，违反“有界”；
- 也可能直接中断请求，却仍被测试错误称为 graceful shutdown；
- deterministic test 无法根据当前合同判定正确上界和强制收敛行为；
- 任一 server 异常时的进程退出可靠性仍依赖实现者临场选择。

**修复建议**

这是生命周期方向选择，不由 Reviewer 代选：

1. 明确首期采用立即关闭，或采用带明确预算的 graceful shutdown。
2. 若 graceful：登记预算来源、取值/合法范围、超时后的强制 `Close`、两个 server 与 DB pool 的关闭顺序，以及根因和 cleanup errors 的组合规则。
3. test plan 按选定策略验证正常收敛、handler 阻塞、预算超时、强制关闭和多重清理错误；不得用 sleep 推断。

**复审条件**

architecture/design 给出单一可执行生命周期，test plan 可在 deterministic fake 上验证同一上界和错误语义。

## 4. 逐对象两轮结论

### 4.1 ADR-0022 与决策索引：NOT PASS

- Round 1：状态、日期、目的分类、主要责任领域、影响范围、确认来源、问题、三类候选、选择理由、代价、消费者、未决项和引用记录完整；索引同步正确。
- Round 2：沿 architecture/design/test plan、`go.mod` 和真实 metrics 消费者核查后，发现 ADR 自行实现 exposition 却没有固定线协议（F001）。
- 结论：方向合理，索引/责任完整，但在 F001 关闭前不能作为无歧义的实施契约。

### 4.2 Architecture r20：NOT PASS

- Round 1：实现基线与 SLA/容量承诺分离正确；配置名、数值、告警、Dashboard、角色链、runbook 和部署 Gate 基本完整。
- Round 2：`cmd/han-server` 可承接双 listener；但 exposition 线协议缺失（F001），server 异常后的“清理”不足以支撑 test plan 的“有界 shutdown”（F003）。
- 结论：不能直接交实施。

### 4.3 Identity design r19：NOT PASS

- Round 1：observer 运行 fail-open、装配 fail-closed、低基数、敏感 denylist、AuditEvent 权威性和部署 Gate 分离均成立。
- Round 2：现役 ports 和 composition root 能承接目标结构；但缺 exposition 线协议（F001），同一文档内“容量后可调”与 buckets 精确等值要求冲突（F002），双 server 生命周期不完整（F003）。
- 结论：当前精确 revision 不建议用户软确认；修订并复审通过后，仍须用户对新的精确 revision/digest 做软确认。

### 4.4 Test plan r04：NOT PASS

- Round 1：application、部署 Gate、真实 E2E/演练，以及既有证据/本轮未重跑/未执行的分离清楚。
- Round 2：I-F005-13/I-F005-15 把精确基线误写成唯一合法值（F002）；I-F005-18 引入没有上游预算/策略的“有界 shutdown”（F003）；应用层没有协议级 exposition conformance，真实 scrape 成为首次兼容验证（F001）。
- 结论：不能作为当前 F005 production composition 的稳定验收输入。

## 5. 待决定、非阻断建议与已确认限制

### 5.1 待决定（对应阻断 findings）

1. 首期 metrics exposition 的精确协议版本和 HTTP 协商/响应合同。
2. 三项预算与 Histogram buckets 的合法值集合：结构合法即可，还是当前版本只接受精确基线。
3. 双 server 异常后的立即关闭或 graceful shutdown 策略、预算来源、强制关闭和错误组合。

这些均属于方向性合同，Reviewer 不替 Lead/用户选择，也不启动修复角色。

### 5.2 非阻断建议

- R 对账时复用现有 `validateHTTPAddress` 的语义或等价共享 helper，但不得为了复用把业务/metrics 错误标识混成不可定位的公共错误。
- 部署材料任务应在项目现有 `deploy/` 布局中绑定实际 rules/dashboard/routing 文件，并新增固定 runbook；具体 provider 路径和 validator 命令由 Task-Planner 根据实际平台登记，不在设计中伪造。
- tasks/Plan 仍引用 r18/r03 和 Q13 TBD；这是必须由后续 R 对账修正的陈旧消费者，不应在本轮直接修改，也不能据此宣称新设计已同步。

### 5.3 未验证风险

- 未运行 Go 测试、race、vet、build 或项目 Gate：本轮对象是 ADR/设计/测试规划，且 production composition 尚未实现。
- 未运行 PlantUML CLI 或 PNG 实渲染；仅做文本级块成对和禁用项静态检查。
- 未验证真实 PostgreSQL 18；仅沿用 F005 code review r03 明确限定的既有证据，本轮未重跑。
- 未验证真实日志采集、30 天查询、Prometheus-compatible scrape、Grafana 导入、Alertmanager receiver、排班/升级链或 runbook 演练。
- 未验证实际 provider/receiver 账号、自然人责任、付费方案、生产流量或容量调优。
- 未执行 E2E、部署、Git 集成或发布。

## 6. 交回条件

1. 修订 ADR/architecture/design/test plan，关闭 F001-F003。
2. 对修订后的固定对象和哈希进行独立复审。
3. 复审 PASS 后，由用户对精确 identity-auth design revision/digest 做软确认。
4. 再交 Task-Planner 执行 R 对账，至少区分：
   - 既有 review r03 PostgreSQL 18 执行证据；
   - 已存在但本轮未重跑的测试制品；
   - 尚未实现/未执行的 application composition；
   - 独立部署材料任务；
   - I-OBS-DEP-01～06 真实部署验证与演练任务。
5. 不得因任务登记、应用测试通过或配置文件存在而关闭 F005、S3.2、B4 或 production readiness。

## 7. 最终哈希

报告落盘后已执行只读复核，分支仍为 `main`，HEAD 仍为
`6da8abbceb0fa197364810a92a8fa2d97b38ddca`。五个固定对象的最终哈希均与开始值一致：

| 对象 | 最终 SHA-256 |
|---|---|
| `.add/project/knowledge/decisions/framework.md` | `B1E7F119EED23375D03606B0A0665C8BB54B4A5CF3F2013C111C36E4A769FD5B` |
| `.add/project/knowledge/decisions/README.md` | `68FA8F98B6991334B207D0A4C1871CD57A1F92286D1F0B78F3FDB933EDD03896` |
| `.add/project/knowledge/design/architecture.md` | `15C93DABA6FF0EA4174F918FAC3748E513469225332D86A7B3D4BD8BB55F287E` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `C319D9742E5E6EF5712D585D8453BABCF990E5CF43355C95E490231BE2AFF790` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `2103241AB0A40F9EC874672714851667AF6FD783AC67C8D2709C83021AE02B63` |

报告文件自身 SHA-256 不能自包含而保持不变，最终值在交付消息中提供。
