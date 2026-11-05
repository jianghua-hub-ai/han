# identity-auth F005 生产可观测性决策独立复审 r02

## 1. 评审身份、对象与总体结论

- 角色：原 `review-observability-decision-r01.md` 的非作者独立 Reviewer；不是本轮 architecture/design/test-plan 作者
- Workflow：`development-delivery / identity-auth S3.2 / F005`
- 工作树：`D:\AID\VSCode\Codex\Han`，`main`
- 基线提交：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审日期：2026-10-23
- 总结论：**NOT PASS**
- 严重度统计：HIGH 0，MEDIUM 0，LOW 2
- 原 findings：`OBS-R01-F001`、`OBS-R01-F002`、`OBS-R01-F003` 均为 **resolved**，无 unresolved 或 regressed
- Task-Planner R 对账准入：**暂不可作为零 finding 的稳定输入交付**。三项原阻断已关闭，但 architecture r22 仍有两处现役追溯/验证状态陈述陈旧；按本轮“存在任一 HIGH/MEDIUM/LOW 即不得 PASS”的准入规则，须形成新的固定修订并复审。

本结论只评价固定文档合同，不表示 production composition、部署材料、真实 scrape、告警、Dashboard、receiver、runbook、演练或 E2E 已实现/通过。现役 `cmd/han-server` 仍为单业务 listener、空 mux、单 `Serve` 路径；这是文档已经明确标注的 planned/not implemented 事实，不作为文档缺陷。

### 1.1 固定对象与逐对象结论

| 对象 | 固定修订 / 开始 SHA-256 | 结论 |
|---|---|---|
| ADR-0022 | `.add/project/knowledge/decisions/framework.md`；`5F4A21D9354243258A06440896EBD2027F0F499B8E52B3E6F3129514F570A636` | **PASS**：F001～F003 已形成单一可实施合同 |
| 决策索引 | `.add/project/knowledge/decisions/README.md`；`95DECEA044382BE4CF0C85EB1212DAEC4A079BDED8272419D0EDA482071C958B` | **PASS**：状态、领域文件、查询入口、引用次数与 ADR 正文一致 |
| Architecture | `architecture-ir01-r22`；`A17F4A482FB5245FF6F3A9FE687F5EF51E4E54415B749F53928B14FAB29D82E0` | **NOT PASS**：新 LOW `OBS-R02-F001`、`OBS-R02-F002` |
| Identity design | `identity-auth-design-ir01-r20`；`2323EE274B343858DA4239F7D1E0CA15F0FB4388062851EB29F3575ABED3AF84` | **PASS（文档合同）**：技术复审通过；仍为 draft，且实现/部署未完成 |
| Test plan | `identity-auth-test-plan-ir01-r05`；`EBC4085ADDCC26FFFFA64EFCDF3466E492FCDC0262E9F4ABDCF9621C4341AC6C` | **PASS（规划对象）**：应用 conformance、composition、部署 Gate 和真实验证分层完整；新增场景均未执行 |
| 原评审基线 | `.spec/wip/identity-auth/review-observability-decision-r01.md`；`76C08FABC03F9D9AEB58D6F8410C99D242905CDF658E4D4BC422881959D4CF63` | **基线完整**：本轮逐项复核其 F001～F003 |

## 2. 两轮实际检查范围

### 2.1 Round 1：整改合同直接复审

逐对象核查：

- Prometheus text exposition `0.0.4` 的 HTTP、字符、转义、浮点、Counter/classic Histogram 和 encoder failure 原子合同；
- 四项预算的部署基线、parser 合法范围、端点、合法非基线值与非法矩阵；
- Histogram buckets 的数量、正数、有限、严格递增、无重复及最大值；
- 双 listener/server 的启动 Gate、统一 coordinator、首触发源、shared deadline、并发 Shutdown、双 Close fallback、Serve-loop/listener/pool 顺序、错误优先级和退出码；
- ADR 状态、候选、代价、owner/责任链、未决项、引用记录及索引；
- Q13 状态、历史修订与现役合同区分；
- 公共 REST、事务/审计、Origin 403、固定指标语义和 r18 cancellation 合同是否保持不变；
- test plan 是否区分应用 conformance、应用 composition、真实部署 Gate，以及已有证据/未重跑/未执行。

### 2.2 Round 2：真实消费者与可验证性复审

沿以下消费者和现状核查：

- `go.mod`：仍无 observability SDK/module；
- `cmd/han-server`：现役 production composition 尚未交付，但现有入口、HTTP 地址校验和资源清理边界可承接目标结构；
- `deploy/` 与 `docs/runbooks/`：现只有 processing gate 与 database recovery 等材料，identity observability rules/dashboard/routing/runbook 尚不存在；
- 当前 `add.manifest.yaml`：七项项目检查已经登记；
- ADR、architecture、design、test plan 的 revision、digest、数值、配置名、告警、Dashboard/runbook 和责任链；
- Markdown 相对引用、PlantUML 块及禁用项、`git diff --check`。

## 3. 原 findings 逐项复审

### 3.1 OBS-R01-F001：resolved

**结论**

F001 已关闭，未回归。

**证据**

- ADR 固定 `0.0.4`、无内容协商、成功 Content-Type、UTF-8/LF/最终 LF、name/label/HELP escaping、浮点、Counter/classic Histogram、`+Inf == _count`、`_sum/_count`、完整内存编码和固定去敏 500：`.add/project/knowledge/decisions/framework.md:624-634`。
- Architecture 同步同一线协议及独立 conformance，真实 scrape 保留为部署 Gate：`.add/project/knowledge/design/architecture.md:561`、`:1248`。
- Design 固定相同 serializer/HTTP/failure 合同：`.add/project/knowledge/design-workspace/identity-auth/design.md:862-870`。
- Test plan 的 I-F005-19/20 使用独立 parser、严格 fixture 或等价 conformance，禁止自产 encoder 自证；I-OBS-DEP-02 明确真实 scrape 不反向替代应用协议验证：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md:166-167`、`:175`、`:180`。

该方案保持项目内标准库实现和 no-new-SDK/module，不依赖外部账号或 provider 才能完成应用协议验证。

### 3.2 OBS-R01-F002：resolved

**结论**

F002 已关闭，未回归。

**证据**

- ADR 将基线与合法集合分开：Login/Origin `1ms..5s`、failure auditor `10ms..30s`，基线 `250ms/250ms/2s`；shutdown 基线 `10s`、合法范围 `1s..60s`；合法非基线不得被拒：`.add/project/knowledge/decisions/framework.md:633`、`:636-647`、`:672`。
- Design 配置表一致：buckets `1..32` 项、正、有限、严格递增、无重复、最大 `<=60s`；四预算范围含端点且无默认：`.add/project/knowledge/design-workspace/identity-auth/design.md:1912-1917`。
- Test plan I-F005-13/I-F005-15 明确覆盖每项基线、合法非基线、两端点及缺失/解析/非有限/越界/结构非法矩阵：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md:160`、`:162`。

当前四对象对“部署基线不是 parser 唯一 allowlist”只有一个解释。

### 3.3 OBS-R01-F003：resolved

**结论**

F003 已关闭，未回归。

**证据**

- ADR 固定必填 `HAN_SHUTDOWN_BUDGET`、统一 coordinator、首触发源、唯一 shared deadline、两个 Shutdown 并发、失败/超时后双 Close、等待 Serve loops 后关闭资源、原始 server error 优先、cleanup error 附加、`http.ErrServerClosed` 正常收敛、bind 失败路径和退出码：`.add/project/knowledge/decisions/framework.md:669-680`。
- Architecture 同步相同生命周期和 r21/r22 强制验证：`.add/project/knowledge/design/architecture.md:563`、`:1044`、`:1248-1250`。
- Design 给出相同启动与关闭顺序，并明确当前尚未实现：`.add/project/knowledge/design-workspace/identity-auth/design.md:595-600`、`:860-862`。
- Test plan I-F005-17/18/21/22 覆盖 bind Gate、近同时触发、首触发 CAS、shared deadline、双 Close、Serve-loop/listener/pool 顺序、主/附加错误和 SIGTERM/SIGINT 退出矩阵；全部使用屏障、通道、fake deadline，禁止 sleep：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md:164-169`。

合同已经足以进行确定性实现与测试；本轮未把“已规划”写成“已实现或已执行”。

## 4. 新 Findings

### LOW OBS-R02-F001：architecture 的现役同步表仍把 identity design r18 当作待同步对象

**位置**

- `.add/project/knowledge/design/architecture.md:1281-1288`，尤其 `:1287`

**违反依据**

- 本轮明确要求检查 revision/digest/source 引用，并禁止历史修订冒充现役合同。
- `code-review-checklist` 的跨对象一致性与消费者影响检查。
- `design-doc-writing-style` 的版本、状态和引用可追溯要求。

**触发场景**

Task-Planner 或后续 Reviewer 从“后续 boundary 与 designs 同步影响”表取当前下游对象时，会得到 `identity-auth-design-ir01-r18`，且该行仍只描述旧的 Prometheus-compatible exposition 与 `250ms/250ms/2s`，没有反映现役 r20 的 `0.0.4`、合法范围、`HAN_SHUTDOWN_BUDGET`、关闭顺序和退出码。

**影响**

- 历史 r18 在当前同步表中表现为现役目标，削弱 architecture r22 → design r20 的精确追溯。
- 后续 R 对账可能引用错误 revision，遗漏 r20 新增的实现/测试输入。
- 不改变已写明的技术合同本身，因此定为 LOW，而非重新打开 Q13 或 F001～F003。

**建议**

由 architecture owner 在新的固定修订中把该行更新为现役 design r20 及其完整影响，或把 r18 明确移入历史追溯并另列现役 r20。若因此产生新的 architecture revision/digest，应同步所有精确 source digest、ADR 引用记录/索引计数及 test-plan 来源元数据。

**复审条件**

当前同步表不再把 r18 表示成现役目标，且下游精确 revision/digest 可闭环。

### LOW OBS-R02-F002：architecture 仍把已登记的项目检查命令写成待确认

**位置**

- `.add/project/knowledge/design/architecture.md:1252`
- `.add/project/knowledge/design/architecture.md:1269`
- 当前事实：`add.manifest.yaml:42-48`
- 已同步消费者：`.add/project/knowledge/design-workspace/identity-auth/design.md:71,115`；`.add/project/knowledge/design-workspace/identity-auth/test-plan.md:41,400,413`

**违反依据**

- `code-review-checklist` 的验证入口真实性与跨对象一致性要求。
- 当前架构文档已把 `add.manifest.yaml` 列为来源，不能同时把已存在的七项检查写成未登记/TBD。

**触发场景**

实施者按 architecture §11/§12 读取验证入口时，会认为格式、静态检查和测试命令尚未登记；按 design/test-plan/Manifest 读取时，又会得到已登记的七项命令。

**影响**

- Task-Planner 无法从 architecture 获得唯一的当前验证状态，可能重复打开已关闭的检查命令决策，或漏排现役项目 Gate。
- 不影响 F001～F003 技术方案，但属于当前验证闭环的直接状态错误，因此定为 LOW。

**建议**

由 architecture owner 在新的固定修订中同步 Manifest 七项现役检查事实，并关闭/改写 Q12；不要把这些命令写成已执行证据。

**复审条件**

architecture、design、test plan 与 Manifest 对“命令已登记、当前文档复审未执行”保持一致。

## 5. 跨合同结论

- Q13 技术基线已关闭；实际 provider/receiver 账号、自然人排班、付费方案和容量调优仍是部署/容量输入，不因本轮 findings 重新打开。
- observer runtime fail-open 与 production composition 缺依赖/非法配置 fail-closed 分层一致。
- 固定指标名、低基数标签、敏感 denylist和 PostgreSQL `AuditEvent` 权威性未回归。
- 公共 REST、登录事务、失败审计、Origin 403、指标语义和 r18 cancellation/deadline 合同未被改变。
- Application conformance、production composition、部署 Gate、真实环境验证/E2E 分层明确。
- 已有 PostgreSQL 18 证据、存在但本轮未重跑的制品、I-F005-13～22 未执行计划和 I-OBS-DEP-01～06 未执行部署验证没有混写。
- 未发现新增外部 provider、账号、自然人排班、付费方案、SLA/SLO 或容量承诺的虚构事实。

## 6. 实际检查与限制

### 6.1 已执行

- 开始阶段核对六个固定文件 SHA-256，全部匹配用户给定值。
- 核对分支 `main` 与 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 完整读取 Reviewer Role、`add-code-review` 及适用 review/design/observability/security/go-service 规则。
- 只读核对 ADR、索引、architecture、design、test plan、原 r01 finding 及真实消费者。
- Markdown 相对引用检查：无缺失。
- PlantUML 文本静态检查：architecture `11/11`、design `21/21` 的 start/end 成对；被审对象无 `linetype ortho`，UML 块内无字面 `\n`。
- `git diff --check`：退出码 0；仅报告工作树 LF/CRLF 提示，无 whitespace error。

### 6.2 未验证

- 未运行 Go test、race、vet、build 或七项项目 Gate；本轮是文档复审，且新 production composition 尚未实现。
- 未运行 PlantUML CLI 或进行图片渲染。
- 未复跑 PostgreSQL 18 事务矩阵，只保留原 r03 明确限定的历史证据。
- 未验证真实日志采集与 30 天查询、真实 scrape、Grafana、Alertmanager、receiver、排班、runbook 演练或网络策略。
- 未执行编码、部署、E2E、stage、commit、push、PR 或 merge。
- 共享未提交树中其他文件由其他工作者所有；本 Reviewer 未回滚、覆盖或格式化其改动。

## 7. 下一动作

1. 由原 architecture owner 关闭 `OBS-R02-F001/F002`，形成新的精确修订；同步受影响的 source digest、ADR 引用记录/索引和 test-plan 来源元数据，不改变已通过复审的 F001～F003 技术合同。
2. 对新的固定对象和哈希执行一次独立定点复审；本 Reviewer不自行启动修复角色。
3. 只有复审达到 HIGH/MEDIUM/LOW 均为 0，才可把该组文档作为稳定输入交 Task-Planner 做 R 对账。
4. 即使后续文档复审 PASS，用户仍需对届时精确的 identity-auth design revision/digest 做软确认；当前技术复审通过的是 `identity-auth-design-ir01-r20` / `2323EE274B343858DA4239F7D1E0CA15F0FB4388062851EB29F3575ABED3AF84`，该确认不等于实现、部署或 E2E 授权。

## 8. 固定对象最终哈希

结束阶段须再次核对以下六个固定文件；报告自身 SHA-256 在交付消息中提供，避免自包含哈希悖论。

| 对象 | 预期最终 SHA-256 |
|---|---|
| `.add/project/knowledge/design/architecture.md` | `A17F4A482FB5245FF6F3A9FE687F5EF51E4E54415B749F53928B14FAB29D82E0` |
| `.add/project/knowledge/decisions/framework.md` | `5F4A21D9354243258A06440896EBD2027F0F499B8E52B3E6F3129514F570A636` |
| `.add/project/knowledge/decisions/README.md` | `95DECEA044382BE4CF0C85EB1212DAEC4A079BDED8272419D0EDA482071C958B` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `2323EE274B343858DA4239F7D1E0CA15F0FB4388062851EB29F3575ABED3AF84` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `EBC4085ADDCC26FFFFA64EFCDF3466E492FCDC0262E9F4ABDCF9621C4341AC6C` |
| `.spec/wip/identity-auth/review-observability-decision-r01.md` | `76C08FABC03F9D9AEB58D6F8410C99D242905CDF658E4D4BC422881959D4CF63` |
