# B5C / identity-auth-S3.5 独立代码评审 r01

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **作者**：Backend-Dev Bohr `01a1690c-8fcd-7cc3-ac1b-e089901eb47c`
- **objectType**：code（provider-neutral deployment configuration + operational runbook）
- **工作区**：`D:\AID\VSCode\Codex\Han`
- **分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **结论**：**Code Review Gate: NOT PASS**
- **平台结论**：**S3.5 Platform Gate: BLOCKED-FOR-EXECUTION / NOT PASS**。本报告不得用于关闭平台 validator Gate、S3.6 或 production readiness。

## 1. 固定对象

| 文件 | SHA256 |
|---|---|
| `deploy/observability/identity-alerts.yaml` | `0FD66E8AFE7822472D2C8C06576C937FCF337C319AA80DF2EA9ADC78AADE6912` |
| `deploy/observability/identity-alerts.test.yaml` | `5298580E2EB0F16A4D41A85E6350FC5A27BC0835C8FD87387E870DD82DF40B77` |
| `deploy/observability/identity-grafana-dashboard.json` | `69A25E9FDFC7BCCD35842073099E50E08E98E7E751A6C9CEA118E65D5E63D518` |
| `deploy/observability/identity-alertmanager-routing.yaml` | `C2177891552E9797EDE767AA7AA420828C7F666C349951AD483CB36DE3B42EC2` |
| `docs/runbooks/identity-observability.md` | `D3AAF347E015CCDD64C87F179C752CB22BC484BE48BEAC514B8C020251E8B33A` |

开始检查、形成 findings 前以及报告落盘前复核均匹配，无对象漂移。

## 2. 两轮实际检查范围

### Round 1：直接对象

完整检查五个固定文件：

- 四条 PromQL 的阈值、分子/分母、classic histogram、样本门槛、标签和向量匹配；
- `for`、`keep_firing_for` 与测试序列/评估时刻；
- Grafana schema 39、`${DS_PROMETHEUS}` 输入、六类面板及查询；
- Alertmanager warning/paging matcher、角色 receiver、默认路由、secret/provider 边界；
- runbook 的配置范围、JSON Lines/30 天、0.0.4、网络、shutdown/退出码、处置、回滚、演练、证据索引和 S3.6 边界。

### Round 2：真实消费者与上游合同

有界核对：

- `internal/platform/metrics.go`
- `cmd/han-server/main.go`
- `cmd/han-server/http_config.go`
- `cmd/han-server/lifecycle.go`
- 上述文件的必要测试
- ADR-0022、identity design S03/F005、test-plan I-OBS-DEP-01～06
- tasks 中 `identity-auth-S3.5`
- Plan 中 B5C/S3.5 Pack/Gate

确认了指标名和固定标签、内部失败只随 internal-failure outcome 记录、独立 `/metrics`、配置读取、单一 shared shutdown deadline、双 server shutdown/Close/Serve-loop/pool 顺序，以及 S3.5/S3.6 所有权分界。未把现有实现反写成新合同。

## 3. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

#### IA-S3.5-R01-F001：应用指标查询未按部署 `job` 隔离，会混入同 backend 的其他目标

- **位置**：
  - `deploy/observability/identity-alerts.yaml:17-27,39-52,64-67`
  - `deploy/observability/identity-grafana-dashboard.json:200,247,258,269,316,363`
  - 对照 `docs/runbooks/identity-observability.md:78-81`
- **违反依据**：runbook 要求真实目标使用 `job="han-identity-auth"`，并将该标签作为部署元数据供规则/面板消费；S3.5 材料必须在共享真实 backend 中保持身份服务范围一致。
- **触发场景**：同一 Prometheus-compatible backend 中存在另一个 job、环境或历史部署，暴露相同 `han_identity_*` 或平台级 `han_platform_origin_rejections_total` 名称。
- **影响**：
  - 其他目标的 attempts/internal failures/duration/Origin rejection 被求和；
  - 可能产生 identity-auth 假告警、掩盖真实故障或显示错误趋势；
  - P95 还可能聚合不同 bucket 配置的 classic histograms，失去可解释性。
- **证据**：只有 scrape 查询使用了 `job="han-identity-auth"`；其余三条应用告警和四组应用 Dashboard 查询均没有 job matcher。规则测试序列也没有 job 标签，无法证明隔离。
- **最小修复**：为全部应用指标 selector 增加 `job="han-identity-auth"`；Dashboard 同步修改。测试序列补 job 标签，并加入另一个 job 的高值噪声，证明不影响本服务结果。
- **复验**：静态核所有 selector；运行 `promtool test rules`；在目标 Grafana 对照同时间窗原始查询。

#### IA-S3.5-R01-F002：rule tests 只证明明显超阈值后的正例，未锁定阈值、hold 和精确恢复边界

- **位置**：`deploy/observability/identity-alerts.test.yaml:7-112`
- **违反依据**：
  - test-plan `I-OBS-DEP-03` 要求四类触发持续时间及退出阈值后稳定 10 分钟才 resolve；
  - tasks S3.5 将四类规则、持续窗口和稳定 10m resolve 交给 rules/tests 材料；
  - 本轮特别验收要求核查反例与恢复边界。
- **触发场景**：阈值比较符、最小样本、`for` 或 `keep_firing_for` 被错误缩短/放宽，但现有测试仍在较晚时刻观察明显超阈值数据。
- **影响**：测试可能继续通过，却允许低流量误报、边界值误报、过早 firing 或错误 resolve；当前材料不能充分证明配置的精确时序合同。
- **证据**：
  - attempts、P95 和 Origin 仅在 `21m/26m/21m` 检查正例；
  - 没有 attempts/sample `<20`、ratio `==5%`、P95 `<=2s`、Origin `19` 或 `for` 尚未满足的空告警断言；
  - recovery 仅对 scrape 在 `12m` 断言仍 firing、`14m` 断言已消失，没有检查表达式自 `3m` 为 false 后的精确 10m 边界 `13m`，其他三条无恢复测试。
- **最小修复**：为四类规则增加 hold 前反例和首次 firing 边界；补上述阈值边界；至少精确断言 10m resolve 边界，并证明四条规则均实际应用 `keep_firing_for: 10m`。
- **复验**：目标 Prometheus 版本执行 `promtool test rules deploy/observability/identity-alerts.test.yaml`，保留版本和完整输出。

#### IA-S3.5-R01-F003：runbook 把 `HAN_METRICS_ADDR` 描述为宽泛“valid listen address”，与应用的严格 IP 合同不一致

- **位置**：
  - `docs/runbooks/identity-observability.md:29-36`
  - 对照 `cmd/han-server/http_config.go:12-43`
  - 对照 `cmd/han-server/main_test.go:462-467`
  - ADR-0022 `framework.md:627`
- **违反依据**：ADR-0022 与 I-F005-14 固定为单个 IPv4 或方括号 IPv6 `IP:port`，端口 `1..65535`；域名、URL、空 host 等必须拒绝。
- **触发场景**：部署人员按 runbook 使用通常也被视为合法 listen address 的 `localhost:9090` 或其他 hostname。
- **影响**：应用在启动阶段 fail closed，造成可避免的部署失败；runbook 不能作为准确配置 SSOT。
- **证据**：实际 validator 使用 `net.ParseIP` 并拒绝非 IP；现有测试明确要求 `localhost:9090` 和 URL 失败。
- **最小修复**：将接受范围写为“单个 IPv4 `IP:port` 或方括号 IPv6 `[IP]:port`，port `1..65535`；禁止 hostname、URL、路径和多地址”，并保留网络审批说明。
- **复验**：逐项对照 ADR-0022、`validateHTTPAddress` 和配置测试矩阵。

#### IA-S3.5-R01-F004：runbook 未完整提供 I-OBS-DEP-06 要求的即时止损，以及权威审计与诊断日志区分

- **位置**：
  - `docs/runbooks/identity-observability.md:42-61,162-217`
  - 对照 test-plan `I-OBS-DEP-06`
  - 对照 ADR-0022 `framework.md:620-622`
- **违反依据**：I-OBS-DEP-06 明确要求 runbook 包含“即时止损、审计与日志区分”；ADR-0022 明确 PostgreSQL `AuditEvent` 是权威审计，运行日志不得替代审计。
- **触发场景**：真实告警处理中，值守人员需要在尚未完成根因定位时决定立即停止 promotion、隔离错误版本/目标或采取其他安全缓解；或仅凭 JSON Lines 推断审计事实并执行重试/修复。
- **影响**：
  - 现有 8.1～8.4 主要提供查询、诊断、升级和禁止动作，没有为四类信号给出明确的安全即时缓解、授权 owner 和停止条件；
  - 文档没有明确声明 `AuditEvent` 才是权威审计、JSON Lines 仅是诊断证据且不得替代审计，存在错误处置、重复副作用或错误合规结论风险。
- **证据**：日志章节定义格式、字段和 denylist，但缺少上述权威性声明；triage 章节没有完整的即时止损步骤。第 10 节回滚仅覆盖规则/Dashboard/routing 平台校验失败，不能替代运行中四类告警的即时处置。
- **最小修复**：
  1. 明确 `AuditEvent` 为权威业务审计，JSON Lines 只用于诊断/关联，未知审计状态不得仅凭日志改写、补写或重试；
  2. 为四类信号补 provider-neutral 的即时止损决策点、执行责任角色、禁止项和退出/回滚条件，例如停止 promotion、恢复最后验证版本、保持 Origin 强校验、避免扩大 metrics 暴露。
- **复验**：按 I-OBS-DEP-06 做非作者桌面演练；真实执行与证据仍归 S3.6。

### LOW

无。

## 4. 待决定（不是已接受风险）

1. **Alertmanager 集成形态**：当前文件是完整顶层 route 模板，root receiver 为 `role-team-operations-alerting`。若目标平台把它作为共享 Alertmanager 的整份配置，所有未匹配告警都会落到该 receiver；若作为 identity-auth 专用实例或合并片段，语义不同。平台 owner 必须在 S3.6 前明确“整份配置/专用实例/合并子路由”及未匹配告警策略，不能由本 Reviewer替其选择。
2. **receiver 可接受性**：两个无 integration 的角色 receiver 静态上未发现 secret 或虚构 provider，但本机没有目标 Alertmanager validator，不能确认目标版本是否接受以及 route test/真实投递是否成功。

## 5. 非阻断建议

- 在 runbook 的验证记录模板中固定记录 Prometheus、Grafana、Alertmanager 版本，避免未来 `keep_firing_for` 或 dashboard schema 兼容性结果不可复核。

## 6. 命令与证据

| 检查 | 结果 |
|---|---|
| `git branch --show-current` / `git rev-parse HEAD` | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 五文件 `Get-FileHash -Algorithm SHA256` | 全部匹配固定哈希 |
| `git diff --check -- <五文件>` | PASS |
| `git diff --cached --name-only` | 暂存区为空 |
| Python `json` + PyYAML 加载及规则/测试/面板/路由结构摘要 | PASS；4 rules、5 test cases、6 panels、2 role routes |
| Dashboard 查询和 datasource 静态枚举 | `${DS_PROMETHEUS}` 与六类面板存在 |
| `Get-Command promtool,amtool,grafana-server,docker,podman,yq,jq` | 全部 NOT FOUND |
| Go 命令 | NOT RUN；本对象无 Go 改动，Go 消费者仅只读核对 |

Lead 已提供的严格 JSON/PyYAML parse、重复键、空白/最终 LF、核心静态合同与 diff-check PASS 作为已有证据记录；本评审没有把这些结果外推为平台 validator PASS。

## 7. 已确认的正确范围

- 四条规则的静态阈值、severity、`for` 和 `keep_firing_for: 10m` 与 ADR 基线一致。
- internal-failure 分子只由 internal-failure outcome 记录，分母为全部 Handler-scope attempts；无标签聚合后的 `and` 向量语义可成立。
- P95 使用 classic histogram `_bucket` 和 `sum by (le)`；样本门槛使用 `_count`。
- Dashboard 有 `${DS_PROMETHEUS}` 输入和六类面板。
- routing 没有凭据、endpoint、账号或自然人信息；warning/paging 子路由角色与合同一致。
- runbook 已覆盖 JSON Lines、至少 30 天、0.0.4、网络限制、shared-deadline shutdown、退出码、回滚、证据索引、外部 owner/validator 与 S3.6 分界。

这些确认不抵消 findings，也不证明真实导入、规则执行、通知投递或生产可用。

## 8. 未验证风险

- `promtool check rules`：**NOT RUN / blocked-for-execution**
- `promtool test rules`：**NOT RUN / blocked-for-execution**
- 目标 Grafana import/query：**NOT RUN / blocked-for-execution**
- 目标 Alertmanager config/route test：**NOT RUN / blocked-for-execution**
- warning/paging 真实 receiver、聚合、静默/抑制和升级链：**NOT RUN**
- 真实 scrape、网络 allow/deny、30 天日志查询、四类触发与恢复、runbook 演练：归 S3.6，**NOT RUN**

因此 YAML/JSON 可解析、文件存在、静态 Review 或本报告均不能冒充平台 Gate PASS。

## 9. 复审条件

1. 作者提交五文件的新固定 SHA256，并逐项映射 F001～F004 的修订位置。
2. 原 Reviewer 对修订对象重新执行两轮有界复审，核原 findings 和相关回归，不混用本次哈希结论。
3. 至少取得目标版本的 `promtool check rules` 与 `promtool test rules` PASS。
4. Grafana import/query 与 Alertmanager config/route test 必须由目标 validator 执行并留证。
5. 平台 owner 明确 Alertmanager 集成形态和默认路由策略。
6. 即使材料修订复审 PASS，S3.5 平台 Gate 仍只能在 tasks/Plan 规定的实际 validator 全部 PASS 后由 Lead 决定是否关闭；S3.6 和 production readiness 继续独立。

