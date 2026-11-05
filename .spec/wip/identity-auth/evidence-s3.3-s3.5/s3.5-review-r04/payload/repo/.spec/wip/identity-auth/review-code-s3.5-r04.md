# identity-auth S3.5 独立评审 R04

- 日期：2026-10-28；活动：同一 `development-delivery` 的有界 review，结束状态 **stopped，交回 Lead**。
- 身份：本轮受派的非作者 Reviewer；未再委派，未实施或修复被审对象。本报告不是原作者自审，也不冒称本人是历史 R01～R03 Reviewer。
- 工作区：`D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- objectType：**code**（四份部署配置及 runbook）；**design/adr** 仅检查新增范围声明及直接消费者，不重审整个身份模块或全部 ADR。
- **Code Review Gate: PASS，仅限下列当前固定材料及新增范围声明的静态完整性、一致性和直接消费者合同。** 两轮完成，无新 CRITICAL/HIGH/MEDIUM/LOW finding，历史四项修复保留。
- **S3.5 平台/工具验证：授权豁免 / NOT RUN，不是 PASS。** 依据已接受 ADR-0025，不再把这些执行缺失作为本卡阻断。
- **S3.6：完全不变。** 本次无真实部署、采集、送达、演练或 production readiness 结论；不替人 accepted/checkpoint，不关闭 task，不启动下一工作。

## 1. 固定对象、来源与版本

固定基线：`C:/Users/J00010385/AppData/Local/Temp/han-s33-s35-lead-k2hnhmtu/baseline.json`。本次开始与交回核验覆盖 `materials5`、`source183` 和六份 `documents`，不是只用共同 HEAD 代表未提交实现。

| 当前对象 | SHA256 | 与历史 R03 的关系 |
|---|---|---|
| `deploy/observability/identity-alerts.yaml` | `A341B8651CF3A37DCDECEF58AA061D2F5AD243103D18C59CD2FBCBF336738422` | 相同 |
| `deploy/observability/identity-alerts.test.yaml` | `DCE00924EC9993CC5DB017D0B924242FEB87AC18F82C50DE77EBE3EF2CA7C213` | 相同 |
| `deploy/observability/identity-grafana-dashboard.json` | `4736FAF82DFE82B28C658BACDEE0C9A8EB1EA61CCE28DC982AF6FE336C665015` | 相同 |
| `deploy/observability/identity-alertmanager-routing.yaml` | `C2177891552E9797EDE767AA7AA420828C7F666C349951AD483CB36DE3B42EC2` | 相同 |
| `docs/runbooks/identity-observability.md` | `3DB9B5421E98040B194832377B960EED9C701DB05638AF8F7E70CB9346632887` | 新增第 23 行 ADR-0025 范围段及第 24 行空行 |

本 Reviewer 独立按原始字节移除 runbook 第 23～24 行，重构 SHA256 为 `390963951DB1A4225E957375F3C2A2790D64D09AC224423B8CE5D433C08FB0DF`，精确匹配 R03 固定对象。重构仅写个人 Temp；没有修改仓内 runbook。这证明其余旧正文保留，**不能据此把旧 Review 自动扩到新段**；新段在本轮另行审查。

| 直接文档输入 | 当前 SHA256 | 检查边界 |
|---|---|---|
| `.add/project/knowledge/decisions/framework.md` | `4FA647D562B869A4D5F09241244937F5E7066A6837965C766E39B6B7099315A2` | ADR-0022 §日志/指标/预算/告警/角色/lifecycle；ADR-0025 全节 |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `D57BB2152A93F6F651165EE7DBE1913B28E9E8719B6991B1FDD36E8B7DBAB7FE` | 第 65 行范围声明、生产观测直接合同 |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `0052894D3C4B58F644749F8337BB264FD2768DD26A31C7EA0A361F0F61D939BD` | 第 37 行范围声明、I-F005-13～22 相关依据、I-OBS-DEP-01～06 |

Tasks、Plan、Manifest 的基线与前后逐文件 SHA 留在 Temp，不复制为新任务台账。`tasks.md:362` 的 S3.2 为 `[x]`；S3.5 当前仍 `[-]`，S3.6 仍 `[ ]`。本次不更改它们。

## 2. 两轮实际检查

### Round 1：五对象全文与新增边界

完整审阅五文件，核对：四条表达式的分母/分子、最小样本、严格或包含阈值、classic histogram、向量聚合、job 隔离、severity、hold 与 `keep_firing_for`；九组测试的目标/噪声序列、反例、首次 firing 及精确恢复边界；Dashboard 六类面板、P50/P95/P99、datasource 输入及固定分类；Alertmanager 两条角色路由、root fallback、无真实 integration 的模板性质。

runbook 逐节检查必填配置与范围、无默认/fail-closed、JSON Lines/30 天查询、审计权威、0.0.4/独立 listener/网络边界、Dashboard/规则/角色链、四类即时止损/恢复/回滚、shutdown/退出码、演练证据索引及 §12 外部 owner/validator/前置。新增第 23 行单独核对其授权、适用任务、NOT RUN 标记、历史对象限制与 S3.6 保留条款。

### Round 2：直接生产者、消费者及修复回归

从规则/runbook 的实际引用有界追查：

- `internal/platform/metrics.go:17`、`:117`、`:160`、`:209`、`:248`、`:296`、`:341`：四指标名、固定标签、合法 buckets、`/metrics`、编码失败去敏与提交顺序。
- `cmd/han-server/main.go:403`、`:550`、`:615`、`:1772`、`:1788`、`:1829`：JSON stdout、双 listener、预算和地址读取、attempt/internal-failure/duration/Origin 采样适配；内部失败分子只随 internal-failure outcome 记录。
- `cmd/han-server/http_config.go:12`：单个 IPv4 或方括号 IPv6、端口 1..65535，拒绝 hostname/URL/空 host。
- `internal/identity/login.go:180`、`:293`、`:374`，`internal/identity/handler.go:1479`，`internal/platform/logging.go:50`，`internal/platform/http_security.go:82`：最终结果/时长、固定事件字段、日志与指标分界，未把 trace 或敏感输入变成应用指标标签。
- `cmd/han-server/lifecycle.go:202`、`:299`、`:363`、`:387`、`:694`：shared deadline、双 Shutdown/Close、等待 Serve loops、listener 和 pool 清理、信号与错误结果；只核 runbook 涉及的双 server 合同，不重审其他 worker。
- 只读配置/metrics/lifecycle 必要既有测试，含 `main_test.go:2932`、`metrics_test.go:118`、`lifecycle_test.go:21`，确认相关预期仍在；**没有运行这些测试**。
- 对照 ADR-0022、ADR-0025、design/test-plan、原 S3.5 卡、B5C Pack 及 S3.6 卡，逐项核上游与实际消费一致。S3.2 原十八文件清单纳入哈希保护；只精读本次真实引用所需片段，不声称重审十八文件全部实现或整个 identity 模块。

应用边界未见需要新增实现或改变核心合同的缺项；未扩到全库、生产平台、网络或外部资料。

## 3. 历史 findings 保留核对

历史作者为 Backend-Dev **Bohr** `01a1690c-8fcd-7cc3-ac1b-e089901eb47c`；原 Reviewer 为 **Arendt** `01a16912-e9a3-7371-a4d6-9b5fda7c29ec`，归属依据原卡及 R01～R03。本轮 Reviewer 只对当前对象独立复核，不改写原作者、原 Reviewer、原报告或当时平台结论。

| 稳定 finding ID / 原严重度 | 历史结果 | 本轮实际证据 | R04 结论 |
|---|---|---|---|
| `IA-S3.5-R01-F001` / MEDIUM | R02 CLOSED | rules `:5/:20/:24/:26/:42/:49/:66` 与 dashboard `:113/:200/:247/:258/:269/:316/:363` 全部按 `job="han-identity-auth"` 隔离；测试含 other-service 噪声，聚合/expected labels 未退化 | 修复保留，CLOSED |
| `IA-S3.5-R01-F002` / MEDIUM | R02 静态 CLOSED | 九组材料保留 sample=19、ratio=5%、低 P95、Origin=19 反例；hold 前/首次 firing 与四条 10m 恢复断言齐备；本地展开核样本长度、counter 单调性、累计 bucket 与 +Inf=count、expected annotations/labels | 静态修复保留，CLOSED；rule engine execution NOT RUN |
| `IA-S3.5-R01-F003` / MEDIUM | R02 CLOSED | runbook `:33` 的地址约束与 `validateHTTPAddress`、现有 hostname/URL 拒绝用例一致 | 修复保留，CLOSED |
| `IA-S3.5-R01-F004` / MEDIUM | R02 部分修复仍 OPEN；R03 CLOSED | runbook `:65` 保留 AuditEvent 权威；`:196` 保留隔离业务流量同时维持 scrape/rule 可见性；`:204` 维护例外保留 inventory/absence evidence；`:209` 退出须明确 up=1 连续健康 10m，absence/silence 不作恢复 | 修复保留，CLOSED；真实演练 NOT RUN |

F002 的静态时序审查：scrape 的空/首次 firing/保持/resolve 为 1/2/12/13m；failure ratio 与 Origin 为 19/20/30/31m；duration 为 24/25/35/36m。fixture 端点差值显示 failure 在 10m 为 10/20，21m 为 9/218；Origin 相应为 20、18；histogram 插值 P95 触发为 4.85s、恢复约 0.993s。它们只支持材料算术与断言意图核对，**不是自行实现平台 validator，也不等于真实 PromQL 执行通过**。

R01～R03 当时的 `blocked-for-execution / NOT PASS` 平台记录保留原时点含义；当前 S3.5 的处理由 ADR-0025 改为授权豁免 / NOT RUN，不倒改历史。

## 4. 新 findings

CRITICAL：无。HIGH：无。MEDIUM：无。LOW：无。

新增范围段未扩大免测范围、改变安全合同或预签关闭，design `:65`、test-plan `:37`、ADR-0025 `framework.md:744`～`:749` 与 runbook `:23` 的边界一致。ADR 锚点及 runbook 四文件链接可达。`review-validation-waivers-r01.json` 是 Lead 保存历史范围 Review 返回的记录，不能替代本轮检查；其当时 Tasks 摘要与当前基线不同不外推为当前 Tasks 哈希。

## 5. 待交接事项与限制（不列为新缺陷）

1. **Lead/Planner 收口**：Plan B5C `:696/:706/:709/:711` 保留历史 active、旧 runbook 摘要和 platform-blocked 前提；Tasks `:498` 仍列历史最终 runbook 摘要。ADR-0025 与本次用户指令已明确当前有界适用性，后续由 Lead/Planner 对账。本 Reviewer 不修改、不自行关闭，不把已授权豁免再交用户确认。
2. **S3.6 外部准备仍待落实**：Alertmanager 专用实例/整份配置/共享子路由及 unmatched-alert 策略尚须平台 owner 明确；当前 root 默认团队运行渠道是模板语义，不是已批准的共享平台全量配置。两个 receiver 只有角色名，不具真实投递能力。本次不选择 integration/provider/账号/secret/排班/费用。
3. **真实验证未做**：promtool check/test、任一平台 validator、Grafana import/query、Alertmanager config/route/silence、真实 receiver、日志留存查询、scrape/网络 allow-deny、四类触发/10m 恢复、非作者运行演练全部 NOT RUN。S3.5 的豁免不免除 S3.6 原真实证据、Issue 闭环和结果确认。
4. **应用检查未重跑**：七项历史 same-source 检查按 Lead 收据复用，记为 **本轮 NOT RERUN**。已核收据指向 `evidence-s9.2/lead-manifest-r02/results.json` 的字节 SHA256 `0DCF064C0982FADE0C5BF116303703C77E9533E87AFE785C93DBD9D723DF1B5D`；183 源文件与基线相同。此处不是本 Reviewer 新执行结果，也不证明浏览器、平台或专项覆盖。
5. **覆盖边界**：对固定五文件及边界声明给静态 Review 结论；不对全部身份认证、数据库/SMTP、其他 worker、容量、E2E 或 production readiness 给 PASS。无新增方向决策或必须由用户重新授权的事项。

## 6. 本轮验证与证据

| 本轮实际动作 | 结果与解释 |
|---|---|
| 分支/HEAD、哈希前后核对 | main/指定 HEAD；五材料、183 源文件、六关键文档全部匹配原基线且前后不变；细表留 Temp |
| 本地严格 JSON / PyYAML SafeLoader 解析 | PASS；拒绝重复键；4 rules / 9 cases / 6 panels / 2 role routes；仅语法/结构 |
| 本地 selector/datasource/标签/fixture 检查 | PASS；14 个应用/up selector 均含固定 job；序列长度、单调性、bucket/annotation/label 关系符合上述静态检查 |
| runbook 字节增量与链接 | PASS；23～24 行外精确匹配 R03；ADR 锚点和四 deploy 链接存在 |
| 五文件 scoped `git diff --check` | exit 0；文件当前为 untracked，故 Git diff 空结果不单独证明其内容，另以全文、解析和字节哈希核证 |
| `git diff --cached --name-only` | 空；没有 stage/commit/push 或其他 Git mutation |
| 应用七项检查 | 历史 same-source 证据；本轮 NOT RERUN |
| 平台校验/真实环境 | 授权豁免 / NOT RUN；无网络、安装、平台命令或部署 |

个人 Temp 证据目录：

`C:/Users/J00010385/AppData/Local/Temp/han-s35-review-r04-553241dc7d/`

- `payload-index.json`：可复制 payload 索引，含相对路径、来源、SHA256、尺寸；复制目录后可核验，不依赖本会话缓存。
- `reviewed-readset.json`：实际审阅范围；原始 `readset.jsonl` 另保留加载请求，失败输出/截断不冒充完整语义审阅。
- `before.json` / `after.json` / `protection-check.json`：5+183+6 的前后哈希与漂移结果。
- `before-read-dependencies.json` / `dependency-protection.json`：额外读取依赖（含历史报告/范围记录）的哈希保护。
- `static_review.py` / `static-results.json` / `fixture-arithmetic.json`：只读静态分析与算术证据；脚本使用 `python -B`，未生成 bytecode/cache。
- `runbook-r03-reconstructed.md`、Lead 两份收据与原始基线、历史报告及当前材料快照：仅个人 Temp 副本；原 R03 未覆写。
- `git-readonly-checks.json` / `status-before.txt` / `status-after.txt`：真实命令、退出码和共享树状态。并行 S3.3 报告不属于本 Reviewer 写集。

## 7. 交回与复审条件

当前静态 Code Review 无未关闭 finding；本轮新增段已独立审阅。平台豁免只以 NOT RUN 交回。Lead 仍须现场核报告、固定对象、只写本报告的所有权和证据，再按真实授权处理 Tasks/Plan 与接收；本报告不代替该动作。

若五材料或范围声明哈希改变，应只对改变的对象/受影响消费者重新固定修订并复审；若恢复/扩大豁免，须另有用户决定及 ADR/消费者同步。进入 S3.6 前仍按原卡核平台、版本、网络、真实 receiver、角色、窗口、Case/Run 与三层证据，不能使用本静态 PASS 代替。

**stopped：本轮唯一仓内产出为本 R04；交 Lead，不部署、不接下一工作。**
