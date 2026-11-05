# Identity Story S03 / F005 tasks 与 batch-plan r05 独立复审

## 0. 结论摘要

- **评审日期**：2026-10-23。
- **对象类型**：design；复合 planning 对象（tasks SSOT + Batches Plan）。
- **复审结论**：**PASS，建议规划 Gate PASS**。
- **原 findings disposition**：
  - `TP-S03-R04-F001`：**CLOSED**（原 HIGH）
  - `TP-S03-R04-F002`：**CLOSED**（原 MEDIUM）
  - `TP-S03-R04-F003`：**CLOSED**（原 MEDIUM）
  - `TP-S03-R04-F004`：**CLOSED**（原 LOW；对应 `PLAN-R03-F001` 可关闭）
- **新 findings**：0（CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0）。
- **状态边界**：本 PASS 只表示本次 tasks/Plan 规划合同通过复审；`IDENTITY-S3.2-REV-F005`、`identity-auth-S3.2`、B4、`identity-auth-S3.5`、`identity-auth-S3.6` 均未因此关闭，production observability readiness 也未通过。
- **授权边界**：不授权编码、测试执行、平台/provider 选择、账号或费用、自然人排班、部署、E2E、Git、发布或任务关闭。

## 1. 固定对象、修订与哈希

工作区：`D:\AID\VSCode\Codex\Han`  
分支：`main`  
HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`

| 固定对象 | 开始 SHA256 | 写报告前 SHA256 | 结束 SHA256 |
|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `E19C35F99648044B932881C48B680A1E2DF48878E6DA9A914DA66E780D746235` | `E19C35F99648044B932881C48B680A1E2DF48878E6DA9A914DA66E780D746235` | `E19C35F99648044B932881C48B680A1E2DF48878E6DA9A914DA66E780D746235` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `9D015D17F33310F169D9121A618040305A2B7C06FF3079976BE6EC405C22CDFD` | `9D015D17F33310F169D9121A618040305A2B7C06FF3079976BE6EC405C22CDFD` | `9D015D17F33310F169D9121A618040305A2B7C06FF3079976BE6EC405C22CDFD` |

评审期间未发现固定对象漂移。scoped `git diff --check` 返回 0。

本次采用并核对：

- identity design r22：revision `identity-auth-design-ir01-r22`，object digest `sha256:8eae5505954933826b8405c239d17c41cbea624501eb33d3ace2bfff8feee5d8`。
- accepted checkpoint：`decision-4530446ed148585f8b015eec3d4b09c461b021007cfcacb8273306e09e9136d4`，record SHA256 `408F8048144BF08D42B6671860E2A6F8885430C8AD7736151B17CA60B62EE9C4`；其作用仅为接受规划继续。
- r22 独立复审：`.spec/wip/identity-auth/review-design-r22-tp-s03-f001.md`，SHA256 `D110175D5C5FF5546C1980B78EA8F92570DE4A1A8589AB28C50385F29395639F`，PASS / 原设计整改 F001 CLOSED。
- test-plan r06：SHA256 `93E2F0E9B1F8BD397504470050A3722644AF01DB71D9597F2C4F5AB40559E67D`。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审与对象直接一致性

1. 全文检查 tasks 顶部、S3.2 卡、S03 gap-map、S3.5/S3.6 卡，以及 Plan 顶部、批次外层表、B4/B5C/B11B Context Pack、DAG、批次 Gate、阻塞表和变更记录。
2. 针对 `TP-S03-R04-F001～F004` 逐项核修订位置、触发路径、关闭条件及相反残留。
3. 核 S3.2 application composition、S3.5 provider-neutral 部署材料、S3.6 真实部署验证三层文件所有权与关闭声明。
4. 只读核现役相关路径：四个 lifecycle/metrics 文件、S3.5 五文件、S3.6 两台账均不存在；`cmd/han-server/main.go` 仍为单 listener/空 `ServeMux` 基线。因此规划没有把未实施内容误写为已完成。
5. 全文反查 F005、S3.2、B4、S3.5、S3.6、production observability readiness 的虚假关闭/通过声明；未发现错误外推。

### Round 2：上游、消费者与结构反向检查

1. 反向核对 architecture r23、ADR-0022、design r22、test-plan r06 的 application readiness、部署材料和 production observability readiness 合同。
2. 沿真实消费者检查 Backend-Dev S3.2、部署材料 owner、Test-Engineer/B11B、B12/S3.4；确认 application、材料、真实验证和用户旅程互不替代。
3. 独立解析 tasks/Plan：
   - 51 tasks：12 completed / 1 active / 38 todo；
   - 唯一 active：`identity-auth-S3.2`；
   - task ID 重复 0；
   - requires 引用缺失 0；
   - 拓扑遍历 51/51，无环；
   - 批次外层表分配 51 次，missing 0 / extra 0 / nonunique 0；
   - Plan 中 task checkbox 状态副本 0。
4. 独立解析文件集合：
   - S3.2：14 文件；
   - S3.5：5 文件；
   - S3.6：2 文件；
   - 三组两两交集均为 0。
5. 核 Manifest 现役七项、PostgreSQL 18 prior evidence/本轮未复跑、平台 validator unknown 与 `blocked-for-execution` 的分层表述。
6. 日期事实未晚于当前日期 2026-10-23。

## 3. 原 findings disposition

### `TP-S03-R04-F001`（原 HIGH）— CLOSED

- **原问题**：S3.2 关闭必须等待 S3.5/S3.6，而 S3.5 又 requires S3.2，形成不可满足循环。
- **修订证据**：
  - `tasks.md:302,350`：S3.2 按自身代码、适用测试、非作者 Review 与 Lead Gate 独立关闭；明确不等待 S3.5/S3.6，也不据此宣称 production observability readiness。
  - `tasks.md:424-425`：S3.5 requires S3.2，并在 S3.2 关闭后独立启动。
  - `tasks.md:448-451`：S3.6 requires S3.2 + S3.5，且须真实环境、三层证据与责任确认。
  - `Plan:449,590-593`：保持 `S3.2 → S3.5 → S3.6`，三个闭环互不替代。
  - `Plan:437,599,621,645`：S3.2 仍为唯一 active，B4 明确未关闭。
  - design r22 `§6.8` 对应正文约在 `design.md:2497-2499`：同样允许 S3.2/B4 按自身 Gate 关闭，同时把 S3.6 保留为 production observability readiness 必经 Gate。
- **复审判断**：关闭循环与相反残留均已消除；requires 解析无环；没有提前关闭 F005/S3.2/B4。

### `TP-S03-R04-F002`（原 MEDIUM）— CLOSED

- **原问题**：S3.5 没有完整拥有上游要求的配置、日志采集、scrape/network、shutdown/退出验收及外部配置索引材料。
- **修订证据**：
  - `tasks.md:409-414`：S3.5 固定且仅固定五文件。
  - `tasks.md:415`：`docs/runbooks/identity-observability.md` 被明确设为 provider-neutral 统一 SSOT，覆盖环境变量/合法范围/无默认、JSON Lines 与 30 天查询、metrics 网络与真实 scrape、shutdown/资源顺序/退出结果、Dashboard/alerts/routing/处置和证据索引。
  - `tasks.md:416`：rules/tests、Dashboard、routing、runbook 对 I-OBS-DEP-01～06 的材料映射明确，真实执行结果仍只属于 S3.6。
  - `tasks.md:422,426-427`：平台 validator 未确认时保持 `blocked-for-execution`；仓外配置必须登记 owner 类型、仓内索引、validator/评审责任和 S3.6 前置；关闭条件逐项覆盖 design r22 §6.8。
  - `tasks.md:433,451`：S3.6 不得补造或改写 S3.5 材料，且必须基于真实平台执行。
  - `Plan:520-531,535-542`：Context Pack 与 tasks 同源，未扩张五文件，也未把 provider、账号、secret、费用或自然人排班写成硬合同。
- **复审判断**：材料 owner、统一 SSOT、外部配置索引、validator/blocked 语义及 S3.6 消费闭环完整；S3.6 无需越权补材料。

### `TP-S03-R04-F003`（原 MEDIUM）— CLOSED

- **原问题**：B5C Context Pack 的启动条件低于外层 B4 G4-batch Gate。
- **修订证据**：
  - `Plan:513-514`：B4 三卡关闭后仍须独立批次整体 Review；只有 `B4 G4-batch Gate: PASS` 且用户另行授权才可启动 B5A/B5B/B5C。
  - `Plan:518`：B5C 启动同时要求 S3.2 关闭、B4 三卡关闭、B4 G4-batch PASS、用户另行授权。
  - `Plan:519`：B5C Pack 明确不得覆盖或弱化外层完整 DAG/批次 Gate。
  - `Plan:535`：B11B Pack 同样不得弱化外层 Gate，并保留全部真实环境前置。
  - `Plan:544-570,595-608`：关键链、批次 Gate 与 Pack 口径一致，没有 task-level 与 batch-level 混用。
- **复审判断**：外层表、DAG、Gate、B5C/B11B Pack 已统一；单独摘取 Pack 不再允许提前启动。

### `TP-S03-R04-F004`（原 LOW；对应 `PLAN-R03-F001`）— CLOSED

- **原问题**：tasks/Plan 仍以当前时态声称 Manifest 未登记/未配置项目检查命令。
- **修订证据**：
  - `tasks.md:13` 及各未执行任务卡已统一引用现役 Manifest 七项，并明确“本轮未执行”。
  - `Plan:26` 明确七项已登记，PostgreSQL 18 专用命令与平台专用 validator 分开治理。
  - 全文反查中，“未登记/未配置”只剩：
    1. `Plan:624` 的 Grafana/Alertmanager/provider-specific 平台命令未登记——这是事实上的独立平台缺口，不是 Manifest 七项漂移；
    2. `Plan:626` 对旧 `PLAN-R03-F001` 问题文本的历史引用及“待原 Reviewer 复审”状态——不是当前事实声明。
  - `add.manifest.yaml` 现役七项与 tasks/Plan 引用逐项一致。
- **复审判断**：原当前时态漂移已实质清除。`Plan:626` 有意保留 OPEN 等待本报告，不构成未修复残留；本次独立复审判定 `PLAN-R03-F001` 可关闭。

## 4. 新 findings

**无。**

未发现会阻断本次规划 Gate 的新增 CRITICAL、HIGH、MEDIUM 或 LOW finding。

## 5. 待决定与非阻断建议

### 待决定

无本次规划 Gate 前必须新增的方向决定。实际平台/provider、validator 命令、receiver/账号、费用、角色映射、自然人排班、演练窗口和结果确认责任仍应在对应任务启动前由用户/实际责任方决定；它们不应由 Task-Planner虚构。

### 非阻断建议

1. 后续如获授权同步状态，可在 Plan 的问题台账中把 `PLAN-R03-F001` 标为 CLOSED，并引用本报告；本次不要求为此再次修改固定对象。
2. 后续交接继续把 B5C/B11B Pack 与外层 Gate 一并提供，避免脱离上下文摘取消费。

## 6. 限制与剩余风险

- 按授权未运行 Go、PostgreSQL、Manifest 七项、部署、Prometheus/Grafana/Alertmanager validator、E2E 或真实告警演练。
- 未验证任何 provider-specific 查询、scrape/import/route-test 命令、30 天真实留存、网络策略、receiver 或升级链。
- 未判断外部账号、费用、自然人排班、演练窗口和实际责任人是否已经可用。
- r22 设计复审和 checkpoint PASS 仅作为规划依据；不等于 application composition、部署材料或真实验证 PASS。
- 工作树包含他人未提交内容；本结论只针对两个固定哈希对象及声明的有界消费者，不声称全仓不存在其他历史漂移。

## 7. Gate 与下一步

**规划 Gate：PASS。**

- 原 `TP-S03-R04-F001～F004` 全部 CLOSED；`PLAN-R03-F001` 可由本报告作为独立关闭证据。
- 无需再次退回 Task-Planner 修订本次固定对象；可交 Lead 接纳本次 planning Gate。
- 任何后续 S3.2 编码/测试、B4 关闭、B5C/S3.5、B11B/S3.6、部署或 E2E，仍须分别满足其现有前置并取得用户另行授权。
- S3.6 真实平台 Gate 通过前，不得宣称 production observability readiness、真实部署/采集/告警演练通过或需求整体完成。
