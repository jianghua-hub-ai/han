# B5C / identity-auth-S3.5 独立代码复审 r03

- **复审日期**：2026-10-23
- **Reviewer**：r01/r02 原非作者独立 Reviewer
- **作者**：Backend-Dev Bohr `01a1690c-8fcd-7cc3-ac1b-e089901eb47c`
- **objectType**：code（provider-neutral deployment configuration + operational runbook）
- **工作区**：`D:\AID\VSCode\Codex\Han`
- **分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **定点范围**：仅复审 r02 仍 OPEN 的 `IA-S3.5-R01-F004` 及 scrape/absence 相关回归
- **材料静态结论**：**Code Review Gate: PASS**
- **平台结论**：**S3.5 Platform Gate: BLOCKED-FOR-EXECUTION / NOT PASS**

本次未复审或重新判定 F001～F003，也未修改 r01、r02、五个被审对象、tasks、Plan 或其他文件。

## 1. 固定对象与修订

| 文件 | SHA256 |
|---|---|
| `deploy/observability/identity-alerts.yaml` | `A341B8651CF3A37DCDECEF58AA061D2F5AD243103D18C59CD2FBCBF336738422` |
| `deploy/observability/identity-alerts.test.yaml` | `DCE00924EC9993CC5DB017D0B924242FEB87AC18F82C50DE77EBE3EF2CA7C213` |
| `deploy/observability/identity-grafana-dashboard.json` | `4736FAF82DFE82B28C658BACDEE0C9A8EB1EA61CCE28DC982AF6FE336C665015` |
| `deploy/observability/identity-alertmanager-routing.yaml` | `C2177891552E9797EDE767AA7AA420828C7F666C349951AD483CB36DE3B42EC2` |
| `docs/runbooks/identity-observability.md` | `390963951DB1A4225E957375F3C2A2790D64D09AC224423B8CE5D433C08FB0DF` |
| `.spec/wip/identity-auth/review-code-s3.5-r01.md` | `E9D0822E20E9A55C012F58B7A0D681F895058BA9BD6BAB1536EAAFEBBD7CF241` |
| `.spec/wip/identity-auth/review-code-s3.5-r02.md` | `43C0B18850540CC405F269B26E19A032D4BBA478FFF32CEDDAC6EB0E1DACF9F5` |

复审开始时全部哈希匹配固定值，无对象漂移。

## 2. 两轮定点复审范围

### Round 1：F004 修复核对

- 核对 r02 指出的 unsafe scrape containment 是否已移除；
- 核对正常止损是否隔离候选业务流量而非隐藏监控对象；
- 核对批准维护导致 target 退出 discovery/scrape 时，absence、silence、expected-target evidence 与恢复判定的关系；
- 核对退出 containment 是否必须由明确 `up == 1` 和连续健康 10 分钟证明；
- 确认 F004 已修复的 AuditEvent/JSON Lines 权威边界仍保留。

### Round 2：相关回归与新 finding 扫描

- 对照实际规则 `up{job="han-identity-auth"} == 0`、`for: 2m`、`keep_firing_for: 10m` 检查 runbook 不再把 series 消失当作健康恢复；
- 检查修订没有引入 provider、产品、账号、receiver endpoint 或 secret；
- 检查 production on-call、Han service owner / identity 技术责任角色以及平台/网络外部所有权没有冲突；
- 检查修订没有改变告警表达式、测试、Dashboard 或 routing 对象。

## 3. 原 finding 复审

### IA-S3.5-R01-F004 — CLOSED

**r02 剩余问题**：旧版 runbook 允许把坏 target 从 rule evaluation 隔离；由于规则只有 `up{job="han-identity-auth"} == 0` 而没有 expected-target/absence 告警，series 消失会在 `keep_firing_for: 10m` 后使告警 resolve，可能把移除监控对象误当成恢复。

**修复证据**：

- `identity-observability.md:194-201` 现在要求 production on-call 停止 promotion，并通过批准的部署控制隔离候选应用业务流量，同时保持 target 对 service discovery、批准的 scraper 和 rule evaluation 可见；
- 同段明确禁止把 target absence 隐藏或把 missing telemetry 当作 healthy；
- `identity-observability.md:202-206` 将退出 discovery/scrape 限定为批准的维护例外，要求可追溯 maintenance/silence，并保留独立 expected-target inventory 与 absence evidence；
- maintenance silence 或 absent series 被明确排除为恢复证据，不得据此关闭处置；
- `identity-observability.md:207-217` 要求 target 先恢复到批准的 scraper path，明确产生 `up{job="han-identity-auth"} == 1`，再连续健康 10 分钟，才可恢复 promotion 或移除 containment；若维护移除了 target，必须先恢复可见性，absence 不能满足退出条件；
- 回滚责任仍由 Han service owner / identity 技术责任角色承担，并要求恢复 scrape/rule visibility 后重复同一健康验证；
- `identity-observability.md:63-68` 继续保留 PostgreSQL `AuditEvent` 为权威审计、JSON Lines 仅为诊断/关联证据且不得据其补写、改写或盲重试的边界。

**相关回归结论**：

- 规则仍是 `up{job="han-identity-auth"} == 0`、`for: 2m`、`keep_firing_for: 10m`；runbook 已显式处理该规则无法自行识别 expected-target absence 的边界，没有虚构 `absent()` 能力；
- 正常 containment 保留 scrape/rule 可见性，因此 `up == 0` 会继续驱动告警；
- 批准维护例外即使导致 Prometheus series 消失，也不能满足业务恢复/退出条件，处置状态由可追溯维护记录、expected-target inventory、absence evidence 和 incident 继续承载；
- 文本只使用 provider-neutral 的 deployment control、service discovery、scraper、rule evaluation、maintenance/silence 与角色名称，没有绑定具体平台或虚构 provider；
- production on-call 负责即时止损，Han service owner / identity 技术责任角色负责应用或外部可观测性 revision 回滚，§12 继续由平台/网络 owner 负责 metrics target/network；未发现角色责任冲突。

**结论**：r02 的最小修复条件已逐项满足，原 finding 静态关闭。真实维护、silence、expected-target inventory、absence evidence 和 10 分钟恢复流程仍须在 S3.6 演练，不影响材料 finding CLOSED。

## 4. 新 finding

未发现新的 CRITICAL/HIGH/MEDIUM/LOW finding。

## 5. 命令与证据

| 检查 | 结果 |
|---|---|
| 七个固定对象 `Get-FileHash -Algorithm SHA256` | 全部匹配用户固定值 |
| `git branch --show-current` / `git rev-parse HEAD` | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `git diff --cached --name-only` | 暂存区为空 |
| runbook 8.1 与 §12 逐行读取 | 修复证据与角色所有权如上 |
| alert rule 与 scrape test 定点读取 | 规则仍为 `up == 0`、`for: 2m`、`keep_firing_for: 10m`；既有边界测试对象未改 |
| 旧危险语句检索 | `isolate a bad scrape target from rule evaluation` 不存在 |
| 恢复条件检索 | 明确存在 `up{job="han-identity-auth"} == 1` 和 `absence never satisfies the exit condition` |
| runbook 尾空白/最终 LF | PASS |
| `Get-Command promtool,amtool,grafana-server,docker,podman,yq,jq` | 全部 NOT FOUND |

Lead 提供的五文件 diff-check PASS 与暂存区为空作为已有证据；本 Reviewer 独立复核了固定哈希、F004 修订正文、规则语义和相关所有权。

## 6. 未验证限制

- `promtool check rules` / `promtool test rules`：**NOT RUN / blocked-for-execution**
- 目标 Grafana import/query：**NOT RUN / blocked-for-execution**
- 目标 Alertmanager config/route/silence 行为：**NOT RUN / blocked-for-execution**
- 真实 service discovery、scraper、expected-target inventory、maintenance/silence 与 absence evidence 联动：**NOT RUN**
- 真实 `up == 0`、target disappearance、恢复 `up == 1` 及连续 10 分钟退出演练：**NOT RUN**
- I-OBS-DEP-01～06、真实 receiver、网络策略与 runbook 四类演练仍归 S3.6。

静态材料结论不得外推为平台 validator PASS、真实部署通过或 production readiness。

## 7. Gate 结论

- `IA-S3.5-R01-F004`：**CLOSED**
- 新 finding：**无**
- **Code Review Gate: PASS**：仅表示本次固定五文件的材料静态质量及 r01 F001～F004 均已关闭；不表示平台执行通过。
- **S3.5 Platform Gate: BLOCKED-FOR-EXECUTION / NOT PASS**：promtool、Grafana、Alertmanager 及真实 scrape/maintenance/absence 流程未执行，不得关闭。
- **S3.6 / production readiness**：未通过，不受本报告改变。

## 8. 平台复验条件

1. 在目标平台加载并执行规则测试，保留 validator 输出；
2. 演练 target 持续 `up == 0`、批准维护后 series 消失、target 恢复可见并变为 `up == 1` 三条路径；
3. 证明 silence/absence 不会被人员或自动流程当作恢复，expected-target inventory 与 incident evidence 可追溯；
4. 证明仅在明确 `up == 1` 连续健康 10 分钟后恢复 promotion/移除 containment；
5. 完成 Grafana import/query、Alertmanager route/silence、真实 receiver 和 I-OBS-DEP-01～06 证据。
