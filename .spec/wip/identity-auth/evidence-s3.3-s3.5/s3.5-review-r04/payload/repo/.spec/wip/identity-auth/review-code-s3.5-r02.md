# B5C / identity-auth-S3.5 独立代码复审 r02

- **复审日期**：2026-10-23
- **Reviewer**：r01 原非作者独立 Reviewer
- **作者**：Backend-Dev Bohr `01a1690c-8fcd-7cc3-ac1b-e089901eb47c`
- **objectType**：code（provider-neutral deployment configuration + operational runbook）
- **工作区**：`D:\AID\VSCode\Codex\Han`
- **分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **原报告**：`.spec/wip/identity-auth/review-code-s3.5-r01.md`
- **原报告 SHA256**：`E9D0822E20E9A55C012F58B7A0D681F895058BA9BD6BAB1536EAAFEBBD7CF241`
- **材料静态结论**：**Code Review Gate: NOT PASS**
- **平台结论**：**S3.5 Platform Gate: BLOCKED-FOR-EXECUTION / NOT PASS**

本次只复审 r01 的 `IA-S3.5-R01-F001`～`F004` 及相关回归。未修改 r01、五个被审对象、tasks、Plan 或其他文件。

## 1. 固定修订

| 文件 | 本次 SHA256 | 与 r01 相比 |
|---|---|---|
| `deploy/observability/identity-alerts.yaml` | `A341B8651CF3A37DCDECEF58AA061D2F5AD243103D18C59CD2FBCBF336738422` | 已修订 |
| `deploy/observability/identity-alerts.test.yaml` | `DCE00924EC9993CC5DB017D0B924242FEB87AC18F82C50DE77EBE3EF2CA7C213` | 已修订 |
| `deploy/observability/identity-grafana-dashboard.json` | `4736FAF82DFE82B28C658BACDEE0C9A8EB1EA61CCE28DC982AF6FE336C665015` | 已修订 |
| `deploy/observability/identity-alertmanager-routing.yaml` | `C2177891552E9797EDE767AA7AA420828C7F666C349951AD483CB36DE3B42EC2` | 未修改 |
| `docs/runbooks/identity-observability.md` | `767BED93C0220564BDA7D79AC62864A1D2A08857B9BAF4F7E8FF7281DE01472E` | 已修订 |

开始复审时全部哈希和原报告哈希匹配，无对象漂移。

## 2. 两轮复审范围

### Round 1：原 finding 定点复审

- F001：逐个核 alert rule 与 Grafana 应用指标 selector 的 `job="han-identity-auth"`；核 test target/noise job 和 expected labels。
- F002：完整展开 9 个 rule-test case 的 promtool 数列记法，核样本数量、counter 单调性、10m `increase`、classic histogram 累计 bucket、P95、`for` 首次 firing 及 `keep_firing_for` 精确边界。
- F003：核 runbook 地址语法与 ADR-0022、`validateHTTPAddress`、既有配置测试一致。
- F004：核 `AuditEvent`/JSON Lines 权威边界，以及四类告警的即时止损、责任、禁止动作和退出/回滚。

### Round 2：相关回归与消费者核对

- 核 job matcher 没有改变两个无标签聚合向量的 `and` 匹配；
- 核聚合 alert expected labels 不错误期待 job，scrape alert 仍保留 job/instance；
- 核 Dashboard 六类面板、`${DS_PROMETHEUS}` 和查询结构未退化；
- 核指标名/标签、`HAN_METRICS_ADDR` validator、`/metrics`、shared-deadline shutdown 和 S3.5/S3.6 分界仍与 r01 已读真实消费者及上游合同一致；
- 扫描修订引入的新安全/运维风险。

## 3. 原 findings 逐项复审

### IA-S3.5-R01-F001 — CLOSED

**原问题**：应用告警和 Dashboard 查询缺少部署 job 隔离。

**修复证据**：

- `identity-alerts.yaml:20,24,26,42,49,66` 的全部应用指标 selector 均含 `job="han-identity-auth"`。
- Dashboard attempts、三个 duration quantile、internal failures、Origin rejections 查询均含同一 job matcher。
- 9 个 test case 的目标序列均带 `job="han-identity-auth"`，并加入 `job="other-service"` 高值噪声。
- 内部失败规则的两侧 `sum(...)` 仍都生成无标签 instant vector，`and` 匹配不受 job matcher 破坏。
- 聚合类告警在 `sum` 后不保留 job，因此 expected labels 只含静态 `service/severity/signal` 是正确的；scrape alert 未聚合，expected labels 正确保留 `job/instance`。

**结论**：原静态材料缺陷已修复。真实 Prometheus/Grafana 运行验证仍属平台未验证风险，不保持本 finding OPEN。

### IA-S3.5-R01-F002 — CLOSED（静态逻辑闭环；实际 promtool execution 未运行）

**原问题**：rule tests 未锁定阈值反例、hold 和精确 10m resolve 边界。

**test data notation 与时序复核**：

1. **Scrape**
   - 目标 `up` 在 `t=0..2` 为 0，表达式从 `t=0` 成立；
   - `for: 2m`：`t=1` 不 firing，`t=2` 首次 firing；
   - `t=3` 恢复为 1；`t=12` 仍在 10m keep-firing 窗口，`t=13` resolve；
   - other-service 的持续 `up==0` 被 job matcher 排除。
2. **Internal failure ratio**
   - `t=10` 的 10m 窗口 attempts 增量 20、failures 增量 10，表达式首次成立；
   - `for: 10m`：`t=19` 不 firing，`t=20` 首次 firing；
   - `t=21` 窗口 attempts/failures 增量为 `218/9`，比例约 `4.13%`，表达式转 false；
   - `t=30` 仍 firing，`t=31` resolve；
   - 独立反例分别证明 attempts 增量 19 不满足门槛、比例恰为 `0.05` 不满足严格 `>0.05`。
3. **Login duration P95**
   - 主触发窗口的 count 增量 20，`le=2` 增量 0、`le=5/+Inf` 增量 20，P95 为 4.85 秒，表达式从 `t=10` 成立；
   - `for: 15m`：`t=24` 不 firing，`t=25` 首次 firing；
   - `t=26` 的 10m 窗口累计 bucket/count 增量为 `le=1:400, le=2:400, le=5:418, +Inf/count:418`，P95 约 0.993 秒，表达式转 false；
   - `t=35` 仍 firing，`t=36` resolve；
   - 独立反例证明 count 增量 19 不满足门槛，以及全部样本落在 `le=1` 时 P95 `<=2s`。
4. **Origin rejection**
   - `t=10` 首次达到 10m 增量 20；
   - `for: 10m`：`t=19` 不 firing，`t=20` 首次 firing；
   - `t=21` 的 10m 增量降为 18，表达式转 false；
   - `t=30` 仍 firing，`t=31` resolve；
   - 独立反例证明增量 19 不告警。

**结构回归**：

- 多段 expanded notation 的每个 case 都提供到最后 eval time 的样本；
- 所有 counter/count/bucket 序列单调不减，无伪造 counter 回退；
- 每个时刻 classic histogram buckets 累计单调，`+Inf == count`；
- other-service 噪声足以在缺 job matcher 时改变结果，但当前被排除。

**结论**：原测试材料缺口已静态关闭。由于本机没有 promtool，本结论不声明 YAML 被真实 Prometheus rule-test engine 执行过；该执行限制单列于平台风险，不保持原 finding OPEN。

### IA-S3.5-R01-F003 — CLOSED

**修复证据**：

- `identity-observability.md:31` 现在明确仅接受一个 IPv4 `IP:port` 或方括号 IPv6 `[IP]:port`，port `1..65535`。
- 同行明确拒绝 hostname、URL、path、multiple addresses 和 empty host。
- 该文字与 ADR-0022、`cmd/han-server/http_config.go:12-43` 以及 `main_test.go` 的 hostname/URL 拒绝测试一致。

**结论**：runbook 已恢复为准确配置 SSOT，原 finding CLOSED。

### IA-S3.5-R01-F004 — 仍存在（部分修复，MEDIUM）

**已完成部分**：

- `identity-observability.md:63-68` 已明确 PostgreSQL `AuditEvent` 是权威业务审计，JSON Lines 仅用于诊断/关联，禁止据其补写、改写或在 persistence unknown 时盲重试。
- 四类告警均新增了责任角色、禁止动作、退出/回滚条件；internal failure、P95、Origin 的止损内容与合同一致。

**剩余缺陷位置**：`identity-observability.md:194-206`，尤其 `196-197`。

**触发场景**：scrape-down 告警触发后，平台/网络 owner 按 runbook 将坏 target “isolate ... from rule evaluation”。

**问题与影响**：

- 当前 scrape 表达式只有 `up{job="han-identity-auth"} == 0`，没有 `absent()` 或独立 inventory/expected-target 信号；
- 从抓取/规则评估中移除 target 会使该 `up` series 消失，表达式变为空而非继续为 true；
- 10 分钟 `keep_firing_for` 后告警会 resolve，即使应用仍未恢复，从而把移除监控对象变成清除告警的手段；
- 同段虽写有“不得把 missing telemetry 当 healthy”，但实际操作与恢复信号相冲突，不能形成安全即时止损。

**违反依据**：I-OBS-DEP-06 要求可执行且安全的即时止损；ADR-0022 的 scrape-down 信号和 10m stable resolve 要求恢复应由健康信号证明，而不是通过移除 series 达成。

**最小修复**：

- 将动作改为隔离候选应用流量或停止 promotion，同时保留目标的 scrape/rule 可见性；
- 若目标必须退出发现/抓取，要求使用经批准、可追溯的 maintenance/silence 流程并维持独立的 expected-target/absence 证据，且不得据此声明恢复；
- 只有 approved scraper path 恢复并连续 10m 健康后才退出 containment。

**复验条件**：非作者桌面演练“target 仍 up=0”和“target series 消失”两种情况，确认 runbook 不会把 series absence 当作恢复；真实执行仍归 S3.6。

## 4. 新 finding 扫描

除 F004 修订中仍存在的 scrape containment 缺陷外，未发现独立的新 CRITICAL/HIGH/MEDIUM/LOW finding。

## 5. 待决定

Alertmanager 集成形态仍未决定：当前顶层 route 的默认 receiver 为 `role-team-operations-alerting`。平台 owner 仍需决定它是专用实例配置、整份配置还是合并子路由，并明确 unmatched-alert 策略。本复审不替平台 owner 选择，也不把该待决定项误报为已验证 route 行为。

## 6. 命令与证据

| 检查 | 结果 |
|---|---|
| 五对象与 r01 `Get-FileHash -Algorithm SHA256` | 全部匹配用户固定值 |
| `git branch --show-current` / `git rev-parse HEAD` | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `git diff --cached --name-only` | 暂存区为空 |
| `git diff --check -- <五文件>` | PASS |
| Python JSON/PyYAML 结构读取与 selector/job/test 摘要 | PASS；4 rules、9 cases、6 panels |
| 自建只读 expanded-notation 检查 | 所有 case 样本长度覆盖 eval time；counter 无回退 |
| 10m delta/P95 静态计算 | 与本报告 F002 时序证据一致 |
| `Get-Command promtool,amtool,grafana-server,docker,podman,yq,jq` | 全部 NOT FOUND |
| Go 命令 | NOT RUN；无 Go 修订，本次仅复用 r01 的真实消费者核对 |

Lead 提供的严格解析、重复键、最终 LF/尾空白、selector job、9-case 结构、止损段和 diff-check PASS 仅作为已有证据；本 Reviewer 独立复核了与四个 findings 直接相关的内容，没有将静态检查外推为平台 PASS。

## 7. 未验证风险

- `promtool check rules`：**NOT RUN / blocked-for-execution**
- `promtool test rules`：**NOT RUN / blocked-for-execution**
- 目标 Grafana import/query：**NOT RUN / blocked-for-execution**
- 目标 Alertmanager config/route test：**NOT RUN / blocked-for-execution**
- 真实 receiver、scrape、网络策略、30 天日志查询、四类演练和 runbook 执行：归 S3.6，**NOT RUN**

上述限制不阻止 F001～F003 按静态材料缺陷 CLOSED，但禁止据此给平台 validator、S3.6 或 production readiness PASS。

## 8. Gate 结论

- `IA-S3.5-R01-F001`：**CLOSED**
- `IA-S3.5-R01-F002`：**CLOSED（静态逻辑闭环；actual promtool execution NOT RUN）**
- `IA-S3.5-R01-F003`：**CLOSED**
- `IA-S3.5-R01-F004`：**MEDIUM / OPEN（部分修复）**
- **Code Review Gate: NOT PASS**：原因仅为 F004 剩余的 unsafe scrape containment，不是因为平台工具缺失。
- **S3.5 Platform Gate: BLOCKED-FOR-EXECUTION / NOT PASS**：promtool、Grafana、Alertmanager 等实际 validator 未执行，严禁关闭。
- **S3.6 / production readiness**：未启动或未通过，不受本报告改变。

## 9. 下次复审条件

1. 提供修订后五对象的新固定 SHA256；未改文件也须列明。
2. 定点移除或替换 F004 的“从 rule evaluation 隔离坏 target”动作，并保留健康恢复的 10m 证明。
3. 原 Reviewer 复审 F004 和相关 scrape/absence 回归。
4. 平台工具可用后另行执行并留存 promtool、Grafana import/query、Alertmanager route test；这些执行结果属于 Platform Gate，不与材料 finding 状态混写。
