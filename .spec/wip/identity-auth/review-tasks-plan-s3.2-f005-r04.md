# identity Story S03 / F005 tasks 与 Batch Plan 独立评审 r04

## 0. 结论摘要

- **评审日期**：2026-10-23。
- **对象类型**：design；复合 planning 对象（tasks SSOT + Batches Plan）。
- **结论**：**不建议规划 Gate PASS**。
- **Findings**：HIGH 1、MEDIUM 2、LOW 1。
- **阻断原因**：
  1. 当前 `S3.2 → S3.5` 关闭/启动顺序与 design r21 的明确关闭合同形成不可同时满足的循环；
  2. S3.5 的文件与关闭合同没有完整承接上游要求的配置说明、日志采集、scrape/network 和进程退出验收材料；
  3. B5C Context Pack 的启动条件低于同一 Plan 已声明的 B4 G4-batch Gate。
- **保持未关闭**：`IDENTITY-S3.2-REV-F005`、`identity-auth-S3.2`、B4、`PLAN-R03-F001` 均不得因本评审关闭。
- **授权边界**：本结论不授权编码、测试执行、平台/provider 选择、账号或费用、自然人排班、部署、E2E、checkpoint、commit 或 push。

## 1. 固定对象与修订

工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD：

`6da8abbceb0fa197364810a92a8fa2d97b38ddca`

| 固定对象 | 开始/写报告前 SHA256 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `F86D9419F295007778CED5C38B45391FCFB53AC22657C6CEB03EAD7F61F07285` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `9B58CDA648F5F3362940764D13E68248AFEE239B2B77863F1CF6E79BCAB234F7` |

辅助固定证据：

| 证据 | SHA256 |
|---|---|
| identity test-plan r06 | `93E2F0E9B1F8BD397504470050A3722644AF01DB71D9597F2C4F5AB40559E67D` |
| observability decision review r03 | `5B8479ABC0A150491F72DB854A83D59F464FD2EA52498579BB18E4D9F7DB1F24` |

design r21 使用用户给定的 object digest
`sha256:d49ba5a13feceb4ffacfb3f776a6df6d6be8b68895d6672024b7ce4f2b5f1523`；
checkpoint `decision-1f1c1d250d737c03e387c8748a1bcab9fbb5111259e309c3e8f0ec21b1104d18`
为 `accepted:proceed`。本轮保留 `proceed-with-known-tool-limitation`，不声称 ADD CLI current。

## 2. 两轮实际检查范围

### Round 1：对象直接缺陷与同源一致性

实际检查：

1. 对照 tasks 顶部、S3.2 卡、S03 gap-map、S3.5/S3.6 卡，与 Plan 顶部、批次外层表、B4/B5C/B11B Context Pack、DAG、Gate 和阻塞表。
2. 只读核对现役代码/文件：
   - S3.2 十四个允许文件中十个已存在，`lifecycle.go/_test.go`、`metrics.go/_test.go` 不存在；
   - `cmd/han-server/main.go:44-119` 仍为单 listener、空 `ServeMux`，未装配 r21 composition；
   - S3.5 五个文件和 S3.6 两个 Case/Run 台账均不存在；
   - identity/platform 现有 coordinator、failure auditor、写后 observer 与低基数 metrics port 可定位。
3. 核 I-F005-13～22 的任务承接、十四文件边界、只读/禁止文件、开发/部署材料/真实验证三层分离。
4. 核 S3.5 的 provider-neutral 合同、平台 validator unknown 与 `blocked-for-execution` 表述。
5. 核 S3.6 的真实平台、Case/Run、三层证据、责任确认及其与 S3.4/Go 单测的隔离。
6. 核 F005/S3.2/B4 未关闭，以及规划没有授予实施、平台选择、部署或发布权限。

直接核对通过的部分：

- gap-map 为已建成 3 / 待做 2 / 需改 1 / 废弃 0；“已建成”没有被改成 `[x]`。
- I-F005-13～22 全部由唯一 active `identity-auth-S3.2` 承接，没有新增重复 application task。
- S3.2 十四文件、S3.5 五文件、S3.6 两台账三层零交叉。
- S3.6 明确禁止以 Go 测试、mock receiver 或 S3.4 用户旅程替代。
- S3.5/S3.6 均保留未实施、未认领和外部前置未知事实。

### Round 2：上游反查、消费者与结构解析

实际检查：

1. 反向核对 architecture r23、ADR-0022、design r21、stories S03、test-plan r06 §4.2.2/§9.1/§10～12，以及 observability review r03、F005 code review r03。
2. 检查真实消费者：
   - Backend-Dev 的 S3.2 application composition；
   - deployment/platform owner 的 S3.5 材料；
   - Test-Engineer/E2E 的 S3.6；
   - B12 / `identity-auth-S3.4` 的独立用户旅程。
3. 结构解析结果：
   - 51 个 task ID 唯一；
   - 12 completed / 1 active / 38 todo，唯一 active 为 `identity-auth-S3.2`；
   - 所有 `requires` 均指向存在的 task；
   - task requires 图无环；
   - 51 个 task 在批次外层表中各出现且仅出现一次；
   - S3.5/S3.6 目标文件未被其他 task 重复拥有。
4. Manifest 七项与 S3.2 卡/Context Pack 一致，均明确“本轮未执行”；PostgreSQL 18 既有证据没有外推为 composition 或部署 Gate。
5. `git diff --check` 返回 0；只出现现有工作树的 LF/CRLF 提示。
6. 未发现晚于 2026-10-23 的事实性完成或验证声明。

## 3. Findings

### [HIGH] TP-S03-R04-F001：S3.2 关闭与 S3.5 启动条件和 design r21 形成不可满足循环

- **位置**：
  - `.add/project/knowledge/tasks/tasks.md:350`
  - `.add/project/knowledge/tasks/tasks.md:422-424`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:449`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:513-518`
  - 上游：`.add/project/knowledge/design-workspace/identity-auth/design.md:2495`
- **违反依据**：
  - design r21 明确规定：部署材料未落地并演练前，不得宣称 F005 production composition 或 S3.2/B4 完成。
  - 当前 tasks/Plan 却规定 S3.2 在 application composition、代码 Gate 和 Review 后关闭，S3.5 必须等 S3.2 关闭后才能启动。
  - `add-batch-plan` 要求 tasks 与批准设计冲突时停止并回设计 owner，不能在 Plan 中自行选择一种解释。
- **触发场景**：I-F005-13～22 application composition 已完成并通过代码 Review，但 S3.5/S3.6 尚未实施。
- **影响**：
  - 按 tasks/Plan 关闭 S3.2 会违反 design r21；
  - 按 design r21 保持 S3.2 未关闭，又无法满足 S3.5 的启动条件；
  - 因此没有一条执行路径能同时满足两个固定合同，规划不能作为可执行 Gate。
- **修复建议（方向决策，不由 Reviewer 代选）**：
  1. **建议方向**：由 design owner 明确把 §6.8 的“不得宣称”限定为 production observability readiness/生产就绪，不阻止 application task S3.2 按自身代码、测试和 Review 条件关闭；保留三层任务与 `S3.2 → S3.5 → S3.6`。
  2. 若原意确实是 S3.2/B4 必须等待部署材料和演练，则必须重做任务关闭条件与依赖，消除 S3.5 依赖 S3.2 关闭造成的循环，并重新核任务分层。
- **证据**：architecture r23 与 ADR-0022 只把外部 collector/scraper/Dashboard/receiver 定义为生产就绪 Gate；test-plan r06 也把 application 实现结论与 deployment readiness 分成两个 Gate。冲突集中在 design r21:2495 与本次 planning 的关闭语义。
- **复审条件**：提供修订后的权威合同与 tasks/Plan；证明 S3.2、S3.5、S3.6 的启动和关闭顺序可同时满足、requires 无环、F005/S3.2/B4 不会被提前关闭。

### [MEDIUM] TP-S03-R04-F002：S3.5 没有完整拥有上游要求的 production observability 部署材料

- **位置**：
  - `.add/project/knowledge/tasks/tasks.md:407-425`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:519-528`
  - 上游：`.add/project/knowledge/design/architecture.md:883-892`
  - 上游：`.add/project/knowledge/design-workspace/identity-auth/design.md:2495`
- **违反依据**：
  - architecture r23 的交付模型要求配置说明和 scrape 配置属于随仓交付元素。
  - design r21 要求部署材料包含全部环境变量/合法范围/基线/无默认、stdout JSON Lines 采集与 30 天查询、metrics 网络限制与真实 scrape、shutdown/进程退出验收，以及 Dashboard/rules/routing/runbook。
- **现状**：
  - S3.5 只固定 alert rules/tests、Grafana dashboard、Alertmanager routing 和 runbook 五文件；
  - 关闭条件只强制四类告警、Dashboard、角色路由、升级链和 runbook 处置内容；
  - Plan 进一步把本卡描述为主要覆盖 I-OBS-DEP-03～06；
  - S3.2 禁止修改 deploy/docs，S3.6 只写 Case/Run 并消费“已部署候选”，当前没有其他 task 拥有配置说明、日志采集、scrape/network 或进程退出验收材料。
- **触发场景**：五个 S3.5 文件通过各自 validator，但平台责任人准备部署 I-OBS-DEP-01/02 或核进程退出时。
- **影响**：S3.5 可以在上游要求的部署输入仍缺失时被关闭；S3.6 随后只能临时创造无 owner 的部署材料、越过文件边界，或继续阻塞。
- **修复建议**：
  - 明确 S3.5 的唯一材料 SSOT：可以扩展精确文件清单，也可以明确 `identity-observability.md` 必须完整承载配置/采集/scrape/network/退出验收合同；
  - 若真实平台配置不入仓，须明确外部配置由谁产生、仓内如何索引、何时评审，并在 S3.5 关闭条件中逐项覆盖 design r21:2495；
  - 保持 provider-neutral，不在规划中虚构 provider、账号、secret、自然人或付费方案。
- **验证**：复审时逐项映射 design r21:2495 的材料要求到唯一文件 owner、validator/人工核对与关闭条件；确认 S3.6 不需要越权补材料。

### [MEDIUM] TP-S03-R04-F003：B5C Context Pack 降低了同一 Plan 已声明的批间启动 Gate

- **位置**：
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:513-518`
  - 相关外层 Gate：`.add/project/knowledge/plans/ir-01-content-community-batches.md:61`
- **违反依据**：
  - Plan:514 明确要求三张 B4 task 关闭后还要完成独立 B4 整体 Review，只有 `B4 G4-batch Gate: PASS` 且用户另行授权才可启动 B5C。
  - B5C Context Pack:518 却只要求 S3.2 与 “B4 task-level Gate” 关闭并取得独立授权，没有保留 G4-batch PASS。
  - `add-batch-plan` 明确区分 task-level 关闭和 G4-batch 整体评审，二者不能互相替代。
- **触发场景**：B4 三卡均关闭，但 B4 整体 Review 尚未完成或尚有跨任务 finding；调用方按 B5C Pack 请求启动。
- **影响**：同一 Plan 给出两个不同的启动口径，可能在批次整体 Gate 前启动 B5C。
- **修复建议**：把 B5C Context Pack 的启动条件与 Plan:514 完全统一，明确列出 `B4 G4-batch Gate: PASS`、S3.2 关闭及用户另行授权；B11B Pack 也宜明确引用其受控的外层 DAG/Gate，避免 Pack 被单独消费时弱化前置。
- **验证**：逐项核外层表、完整 DAG、Gate 章节和各 Context Pack 的启动条件一致，没有 task-level 与 batch-level 混用。

### [LOW] TP-S03-R04-F004：现役 Manifest 事实漂移仍在固定对象中保留

- **原 finding 对应**：`PLAN-R03-F001`，仍为 OPEN；本报告不借旧 PASS 关闭。
- **位置示例**：
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md:416`
  - `.add/project/knowledge/tasks/tasks.md:380`
  - `.add/project/knowledge/tasks/tasks.md:397`
  - `.add/project/knowledge/tasks/tasks.md:464`
  - Plan 自身登记：`.add/project/knowledge/plans/ir-01-content-community-batches.md:623`
- **现状**：这些旧卡/历史 Context 仍以当前时态写“Manifest 未登记/未配置项目检查命令”，但现役 Manifest 已登记七项。
- **触发场景**：未来实施者从旧 task 卡或 B3 Context Pack 复制项目 Gate。
- **影响**：可能把现役七项误当临时候选命令；不改变本次 S3.2/B5C/B11B 的直接命令和授权，因此为 LOW。
- **修复建议**：另行授权有界事实同步；历史段改成带日期的历史事实或直接引用 Manifest，不顺带重写任务状态。
- **验证**：全 tasks/Plan 搜索同类当前时态陈述，与 Manifest 七项逐项一致。

## 4. 待决定与非阻断建议

### 待决定

1. **必须决定** design r21:2495 的关闭语义：它是 production readiness 限制，还是 S3.2/B4 task closure 的硬前置。该决定直接影响 F001 的修复方向。
2. S3.5 的环境配置说明、日志采集、scrape/network 和退出验收材料采用新增独立文件，还是由现有 runbook 作为完整 SSOT；必须在实施授权前固定所有权。

### 非阻断建议

1. B5C/B11B Context Pack 各自增加一句“本 Pack 不覆盖外层批次 Gate；启动仍须满足 §批次 Gate/完整 DAG”，减少被单独摘取时的歧义。
2. S3.5 的 `blocked-for-execution` 语义总体与 test-plan r06 一致；修订材料范围时仍应保持平台 validator、真实 receiver/账号和责任未确认不得宣称可执行或通过。
3. PG18 的 prior evidence、当前未复跑和 DSN/major 18 缺失时 `blocked-for-execution` 的三态表达正确，建议保持。

## 5. 限制与剩余风险

- 按授权未运行 Go、PostgreSQL、部署、平台 validator、E2E 或真实告警演练。
- 未验证任何 provider-specific 查询、Grafana import、Alertmanager route test、日志留存、网络策略或 receiver。
- 未判断外部账号、费用、自然人排班和实际责任人是否可用。
- 工作树含大量他人未提交改动；本评审只针对固定哈希对象和有界消费者，不声称全仓规划无其他历史漂移。
- `git diff --check` 通过不替代设计一致性、测试、部署或 E2E。

## 6. Gate 结论与复审条件

**规划 Gate：FAIL / 不建议 PASS。**

复审至少需要：

1. 针对 `TP-S03-R04-F001` 提供 design/规划 owner 的明确方向决定及一致修订；
2. 补齐 `TP-S03-R04-F002` 的材料所有权和关闭映射；
3. 统一 `TP-S03-R04-F003` 的 B5C 启动 Gate；
4. 保持两个固定对象可识别的新哈希，重新核 51 task 唯一性、requires 存在性/无环、每 task 批次一次、三层文件零交叉；
5. `PLAN-R03-F001` 可继续作为 LOW 非阻断项，但不得被本次修订静默标 CLOSED；
6. 修订和复审均不得借机关闭 F005、S3.2、B4，或扩成编码、测试、部署、E2E、Git/发布授权。
