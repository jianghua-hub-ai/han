---
name: e2e-verification
description: 真实环境 E2E 端到端验证闭环——起真实服务、走真实入口、做三层断言（交互/后端数据流/数据符合性）、人评审用例、按优先级+依赖编排执行、问题走 issues 闭环、台账回填完成。test-engineer 第二主 SKILL，活动 9 集成发布（部署）后的独立联通测试任务。区别于 test-design（规划测什么），本 SKILL 验"真实环境怎么跑通跑对"。
requires:
  - test-design
  - rule:tracing-standards
  - rule:observability-standards
  - rule:task-execution-workflow
  - rule:design-doc-writing-style
  - rule:abstraction-principles
  - code-review
  - issue-handling
  - systematic-debugging
  - knowledge-curate
produces:
  - .spec/e2e/<suite>/cases.md（用例台账 SSOT · 纯规格层：意图规格 + 覆盖派生表 + 物料汇总——不含执行状态）
  - .spec/wip/e2e-<suite>/run-*.md（执行台账 · 临时：一次执行一文件，状态 / 证据链接 / issue 落此；活动 10 归档）
  - .spec/e2e/env.md（环境与证据声明，跨套件共享，首次建）
  - .spec/e2e/<suite>/coverage.md / materials.md（可选 · 仅大套件拆分：>~50 用例 / 多人分工 / 物料跨套件复用）
---

# E2E Verification — 真实环境端到端验证

## 核心理念

按 OO §1.2 边界口诀——本 SKILL 干的是"真实环境 E2E 验证这件事"的流程：起真实服务、走真实用户入口、验后端真按设计做对了、跑出的问题闭环、完成的标记完成。

```text
特性 design + test-plan（test-design 产）
        ↓
e2e-verification SKILL（本 SKILL，test-engineer 主导，项目指定的自动化实现角色写代码）
  Step 0 重验触发判定（换包 / 重部署 / 不动 + 用例级 TIA 重跑范围）
  Step 1 用例拆解（意图 + 分级断言点 + 覆盖 SR 字段 + 优先级 / 正异常 / 前置，按套件组织；覆盖矩阵 + 物料从 case 字段派生）
  Step 2 专业评审（reviewer 预审用例意图与断言）
  Step 3 环境 + 物料就绪（核对站点连通 + 物料齐 / 校验对）
  Step 4 执行编排（前置当依赖项 + 优先级 tag + 依赖 DAG 排序）
  Step 5 执行 + 回填（项目指定的实现角色跑、test-engineer 抨结果 + 归档三层证据）
  Step 6 问题闭环 + 收尾（真问题走 issues、台账回填、终态对账）
        ↓
.spec/e2e/<suite>/cases.md（规格 SSOT · 唯一提交件）+ .spec/wip/e2e-<suite>/（run 记录 + 证据 · 临时）
```

**核心边界**：test-engineer 产"用例**意图规格**（意图 + 断言点「验什么」+ 覆盖 SR + 优先级 + 物料依赖）+ 执行编排 + 验收抨账，**不亲手写自动化实现代码**——项目指定的实现角色**从意图产代码**（handoff 传的是意图共识，不是待翻译的逐步操作脚本），代码由该角色写并跑（保住"规划与实施分离" SRP）。台账与代码靠 **CASE-ID 双向锚定**治 drift。

**cases.md 是规格层，项目选定的 E2E 自动化代码是实现层**（对齐 BDD 意图 / step-definition 实现分层）：
- cases.md = 「意图 + 三层断言点（验什么）+ 覆盖 SR + 优先级 + 物料依赖」——变化慢，规格层
- 详细操作步骤 / 选择器或调用参数 / 断言实现 = 自动化代码——变化快，是**步骤的 SoT**
- **生命周期**：规格先行期（还没写代码）步骤先写在 cases.md 作规格；**代码落地后代码是步骤 SoT，cases.md 保留意图 + 断言点**（步骤不再在两处重叠维护，drift 源消失）
- **执行状态不进 cases.md**——每次执行在 `.spec/wip/e2e-<suite>/` 新增一份 run 记录文件（Case / Run 分层，对齐 TestRail / Xray：结果回写 run，不改写 case）；cases.md 只在规格变更时才改，**其 diff 即重新评审信号**（见 Step 2）。run 记录是过程产物（controller actions），归 wip 约定、活动 10 归档——套件目录只有规格 SSOT

**断言分级**（不是每条都强制三层，按用例性质分）：
- **关键数据流用例**（提交 / 跨服务处理 / 状态流转 / 结果确认）→ **三层齐**：① UI ② trace 串后端调用链 ③ 数据符合性。**只过 ① UI 不算通过**——关键数据流需要验证后端确实按设计执行，因此关键数据流"后端真做对了"必须验。
- **UI 校验 / 交互类用例**（自引用被拒 / 表单校验 / 禁用态）→ ① UI（+ 必要时轻断言 API 状态码），**不强求 ③ DB + trace**。
- **③ 层断言对着设计契约**（业务状态流转 / 数据一致性 / 业务终态）**不对易变内部 schema**——防"断言内部实现"反模式。
- **金字塔提醒**：E2E 只覆盖关键旅程；后端边界穷举正确性归**集成测试**（test-design 规划），E2E 不背。

**白盒判据纪律（防同源污染 / "白盒全通"）**：白盒断言若"期望值"从被测代码读出 → 测试只验证"代码做了代码做的事"，同义反复，bug 被编码进断言全绿。把白盒拆两半：

| 半边 | 是什么 | 来源 |
|---|---|---|
| **探针** | 在哪取证（span 名 / 表名 / 端点 / 状态枚举） | 允许从实现代码取 |
| **判据** | 该看到什么（期望值 / 终态 / 顺序） | **必须溯 design 条款**（SR / US / 契约节），禁从实现现状读出 |

口诀：**代码告诉你去哪看，设计告诉你该看到什么**。

- **判据溯源硬要求**：每条 ②③ 层断言的期望值尾注设计条款引用（如 `←design.md §x.x / SR-id`）——无引用 = 判据来源不明，Step 2 评审打回
- **可失败性自检**：每条 P0 断言过变异思想实验——"若后端漏做 / 做错这步，本断言会红吗？"答不上 = 同义反复，重写
- **生成隔离**：拆用例时从 design 推导期望，查实现代码**只为定位探针**（span 名 / 表名 / 端点）；禁照前后端当前实现行为写期望——Step 1"禁基于前端当前实现预判"是本纪律的前端特例
- **组合判据缝隙**：期望值由 2+ 条款拼合推导时（如「A 说触发重下」×「B 说终态落库」），拼合语义（时机 / 上限 / 优先序）无条款 → 同样进设计 gap 清单，不作硬判据
- **半探针半判据拆分**：设计只给性质（"自然布局"）不给具体值（"`<org>/<name>` 层级"）时——具体值当探针，判据只断性质
- **永真断言排除**：可失败性自检加一问——"存在任何能让这条断言红的实现吗？"没有（「可保留 / 不要求 X」句式）= 删或降 informational 备注
- **引用悬空 = gap**：代码 / 注释引用的 ADR 或设计节**不存在**（决策未入档 / 悬空引用）→ 记设计 gap，不得把代码注释当判据来源
- **例外**：存量代码的特征测试（characterization，锁现状行为防改坏）允许照实现写判据，但必须显式标注"特征测试"，不冒充正确性验证

## 跟 test-design SKILL 的区别

| SKILL | 谁主导 | 范围 | 组织单位 | 时机 |
|---|---|---|---|---|
| **test-design** | test-engineer | 测试**规划**（金字塔全层 / 场景 / 覆盖率目标） | 按**特性** | 活动 5 末 / 6 前 |
| **e2e-verification**（本 SKILL） | test-engineer（项目指定的实现角色写代码） | 真实环境 **E2E 执行验证闭环** | 按**测试套件 / 用户旅程**（跨特性） | **活动 9 集成发布后**（独立任务） |

test-design 回答"该测什么"；e2e-verification 回答"真实环境怎么跑通跑对、问题怎么闭环"。两者 `requires` 链：design-tdd → test-design → e2e-verification。

## 何时启用 / 不启用

**启用**：

- 一个 batch 集成发布（活动 9）完成、代码已部署到真实环境后，跑独立的端到端联通验证（单独立任务、单独闭环）
- 已有真实环境（<form-a> / <form-b> 站点）+ 真实输入数据或测试物料，需端到端验"后端真按设计做对了"
- 跨特性用户旅程需要贯通验证（如"提交 → 处理 → 结果确认"横跨 `<feature-a>` + `<feature-b>`）
- 输入数据 / 配置 / 服务变更后的重验（按 Step 0 判定范围）
- **特性 design / ADR 条款修订后**——先刷新受影响套件的用例规格（判据随条款走，见 Step 0 设计变更联动），再按需重验

**不启用**：

- 单元 / 集成测试 → test-design 规划 + dev 按项目 profile 实施（本 SKILL 不重复）
- 测试场景规划（测什么 / 金字塔分层）→ test-design SKILL
- 纯 UI 组件测试 → 前端实现角色使用项目选定的组件测试工具
- 无真实环境 / 无真实物料时——先补环境与物料（Step 3 前置）

## 加载的规则

- [`tracing-standards.md`](../../rules/tracing-standards.md)（trace_id 跨服务传递 → ② 后端数据流 trace-based 断言）
- [`observability-standards.md`](../../rules/observability-standards.md)（③ 日志断言 + 审计验证）
- [`task-execution-workflow.md`](../../rules/task-execution-workflow.md) §2.9 活动 9 部署后独立 E2E / §8 产物归属 / §10 状态标记
- [`design-doc-writing-style.md`](../../rules/design-doc-writing-style.md)（台账按写作准则 + §F 性能数值占位）
- [`abstraction-principles.md`](../../rules/abstraction-principles.md) §1.3（跨形态——<form-a> / <form-b> / <form-c> 验证场景识别）

## 业界对标

| 方向 | 对标 | 借鉴点 |
|---|---|---|
| 前置条件显式化 | Gherkin Given-When-Then / Playwright fixtures + setup-project | 前置建成可复用依赖项，不散写在用例里 |
| AI/人产可审用例 | Octomind（产标准 Playwright 代码、人可审） | AI 生成必须人可审，服务"人评审"卡点，不黑盒 |
| 执行编排 | Playwright `describe.serial` + project `dependencies` / pytest markers | 优先级 tag + 依赖 DAG，断链下游 skip |
| 数据流/日志验证 | Tracetest / Malabi（trace-based testing，OTel） | 用 trace 做后端链路深断言，远胜只断言 HTTP 200 |
| 用例库 vs 执行结果分层 | TestRail / Xray（Case 库 ↔ Run 记录） | cases.md = Case 库（纯规格）/ wip run 文件 = Run 记录（一次执行一份独立快照）——结果回写 run 不改写 case，完全对齐 |
| 意图 vs 实现分层 | BDD `.feature`（意图 Given-When-Then）/ step-definition（实现） | cases.md = 意图 + 断言点 / 项目选定的自动化代码 = 步骤实现，CASE-ID 锚定，步骤不双份维护 |
| trace 关联锚点 | Tracetest @playwright（页面注入 traceparent，前端点击与后端 trace 同链） | 触发段注入已知 trace 锚点，取证轮询、断言双层（定位+断值） |
| 判据独立 | Cucumber 官方（断言业务可观察结果）/ TDD 红灯先行 | 判据溯 design 条款不照实现写；断言必须"能红"过才可信 |

设计哲学：**真实环境验证 = 用例设计(SSOT) → 人审可审规格 → 前置依赖 + DAG 编排 → trace+截屏留痕 → 状态回写 + 问题走问题单**。

## 输入契约

调用方（test-engineer 主 Agent）必须提供：

| 字段 | 说明 |
|---|---|
| 🔒 **工作目录** | 规格 `.spec/e2e/<suite>/` + 执行期 `.spec/wip/e2e-<suite>/`（必钉死路径——防落错；SSOT 见 [`task-execution-workflow §8`](../../rules/task-execution-workflow.md) 产物归属表 + 本 SKILL 输出契约位置规则） |
| **套件标识** | suite / journey 名（如 `<journey-a>` / `<journey-b>`） |
| **交付对象/修订** | 实际制品版本/代码修订/摘要、部署状态及已确认验收范围；发布完成不等于 E2E 通过 |
| **当前 E2E** | 固定工作区/分支、Case／Run／Issue 项目制品引用、产出/接收负责人和已有决定来源；稳定套件重跑复用仍适用的用例评审与历史决定，不补造确认 |
| **覆盖范围** | 本套件覆盖的 feature / US / SR（每条 case 带「覆盖 SR」字段，顶部覆盖矩阵派生） |
| **特性 design 路径** | 各覆盖特性的 `design.md`（用例从设计推导，不重复设计） |
| **test-plan 路径** | 各特性 `test-plan.md`（E2E 场景来源，如有） |
| **重验触发** | 本次为啥而验（首次 / 被测内容变 / 服务变 / 配置变 / 仅台账改）——决定 Step 0 范围 + 用例级 TIA |
| **环境信息** | 站点 + 凭据指针 + 证据抓取入口（落 env.md） |
| **物料清单** | 本套件要的包 + 测试机路径 + 校验值（各 case 标物料依赖 → 汇总到 cases.md 顶部 §物料） |

核被测对象与环境是否匹配、可执行范围与环境修改授权。缺失项只限制依赖它的用例，不能猜实际部署版本或自动重建客户环境；不受影响且条件完整的用例可继续。部署/清数据等有副作用操作须有独立授权，测试方法不赋予权限。

## 输出契约

**位置规则** 🔒：

- 套件规格落 `.spec/e2e/<suite>/cases.md`（独立的家，按套件组织，**不挂任何单特性目录**；套件目录**只有规格**，是唯一提交件）
- 执行期产物落 `.spec/wip/e2e-<suite>/`（run 记录 `run-<YYYYMMDD>-<n>.md` + 证据 `evidence/run-<id>/`——过程产物归 wip 约定，活动 10 归档 `.spec/archive/e2e-<suite>-<date>/` 或清理）
- 环境声明 `.spec/e2e/env.md`（跨套件共享一份）
- ❌ 禁落单特性 `design-workspace/<feature>/`（跨特性套件塞进单特性目录是反 pattern）
- ❌ 禁把 run 记录 / 证据提交进 `.spec/e2e/<suite>/`（规格与执行记录混目录 = 分层失效）
- ❌ 禁 `/tmp/`

完工时：

- **落盘（规格，默认）**：`cases.md`（纯规格——用例意图规格 + 顶部覆盖派生表 + 顶部物料汇总，**不含执行状态**）；首次建 `env.md` + `README.md`
- **落盘（可选拆分）**：`coverage.md` / `materials.md` 独立文件**仅在大套件才拆**（拆分阈值：>~50 用例 / 多人分工 / 物料跨套件复用）；默认它们的内容并入 `cases.md` 顶部小节，不单独产文件
- **落盘（执行期，临时）**：每次执行一份 run 记录 `.spec/wip/e2e-<suite>/run-<YYYYMMDD>-<n>.md`（触发原因 + TIA 范围 + 环境与被测版本 + 逐 case 结果表）
- **问题**：跑出的真问题在 `.spec/issues/issues.md` 台账登记 `ISSUE-<日期>-<序号>`（走 issue-handling，一 issue = 台账一行），run 记录行内链 issue 号
- **证据**：重证据（截图 / trace / 日志摘录）按 run 分目录归 `.spec/wip/e2e-<suite>/evidence/run-<id>/`；run 记录只放终态 + 证据链接，不当执行日志堆（对齐业界 Case / Run 分层——用例设计与执行结果分两层）
- **摘要返回主会话**：用例总数 / 按优先级分布 / 终态分布（通过 / 失败 / BLOCKED / 未执行或 skip）/ 开出 issue 数 / 覆盖矩阵（派生视图）是否无空白 / 残留 BLOCKED 清单

## 保留与归档（按 K8s 控制器思想）

| 文件 | 长期定位 | 何时回收 |
|---|---|---|
| `.spec/e2e/<suite>/cases.md` | 长期累积资产（套件规格 SSOT，可迭代重跑；默认自包含覆盖派生表 + 物料汇总；不含执行状态） | 不归档——只随规格变更而改 |
| `.spec/e2e/<suite>/coverage.md` / `materials.md`（**可选**，大套件才拆） | 长期（跟 cases.md 同生命周期；未拆时不存在，内容在 cases.md 顶部） | 不归档 |
| `.spec/e2e/env.md` | 长期（环境声明） | 环境下线时更新 / 退役 |
| `.spec/wip/e2e-<suite>/`（run 记录 + evidence/run-<id>/） | 过程记录，按材料约定处置 | 收尾/交接按 `skill:knowledge-curate` 核当前结论、Issue、验收与审计引用；唯一复现证据保留，已转 Issue 不等于可以删除。获授权归档到既有 archive 后核链接/摘要/恢复位置，再收口单处材料记录；引用或授权不明保持 pending/deferred |

E2E 套件规格跟特性级 test-plan.md 不同——**cases.md 是长期可迭代资产**（随规格演进重跑），不是一次性交接物；run 记录则是每次执行的临时快照。

---

## 流程（7 步）

### Step 0：重验触发判定

每次进 E2E 验证先问"这次为啥而验、要动环境吗、重跑多大范围"。判据口诀：**变的是"被测内容" → 换包；变的是"被测系统 / 数据" → 重部署**。

| 这次变了什么 | 环境动作 | 重验范围 |
|---|---|---|
| 仅输入数据 / 测试物料 / 配置包变化 | **替换输入**（服务不动） | 只重跑依赖该输入的套件用例 |
| 服务代码 / 运行配置 / 服务版本变化 | **重新部署**对应服务（或整环境） | 冒烟主线 + 受影响套件全量 |
| 数据结构 / 数据迁移变化 | **重建环境**（含数据） | 全量重跑 |
| 仅前端 / 配置改（无后端逻辑） | 不重部署，刷新前端 | 受影响 UI 用例 |
| 仅测试用例 / 台账改 | 环境不动 | 改动的用例 |

范围按"爆炸半径"取——能缩到受影响套件就不全量，但**改后端必带冒烟主线**（防牵一发动全身）。**首次验证 = 全量**。

**叠加用例级 TIA（Test Impact Analysis，比部署类型更细粒度）**：借每条 case 的「覆盖 SR」字段——某 SR 的设计 / 代码变更 → **精确选跑覆盖该 SR 的 case 集**，不只靠上表部署类型粗粒度筛。部署类型定"要不要动环境 + 冒烟范围"，覆盖 SR 字段定"哪几条 case 精确受影响"，两者叠加取并集。

**设计变更联动（规格刷新，先于重验）🔒**：特性 design / ADR **条款本身修订**时，不是"重跑"是"先改用例"——按「覆盖 SR」字段 + 判据尾注（`←条款`）用 Host 可用的文本/路径检索能力定位受影响用例 → 刷新其判据 / 断言（Step 1 增量）→ cases.md 产生 diff → 按 Step 2 重新评审 → 才进重验。判据引用的条款变了、用例照旧 = 拿过期判据验新设计。全局触发入口挂 [`task-execution-workflow §2.11 变更完整性联动面清单`](../../rules/task-execution-workflow.md)（活动 5 修订设计时过联动面必查 E2E 套件）。

**判定结果记录化**：每次重验 = `.spec/wip/e2e-<suite>/` 新建一份 run 记录文件 `run-<YYYYMMDD>-<n>.md`（模板见 [`references/runs-template.md`](references/runs-template.md)），开头即写下本次判定——触发原因 + 环境动作 + 本 run 范围（TIA 选出的 case 集）。范围外的 case 不出现在该 run 文件（不重跑 ≠ 失效，上次结论仍有效）。

### Step 1：用例拆解（基于需求 + 特性设计）

从覆盖范围内各特性 design + test-plan 推导 E2E 用例**意图规格**（**禁基于前端当前实现预判**——某用例若执行时前端确无对应页面，记 `失败`/`BLOCKED` + 注明"前端未实现"作为 gap 暴露，不提前不测）。

按**套件**组织（一个套件 = 一条用户旅程，可跨特性）。每条用例填固定结构（模板见 [`references/cases-template.md`](references/cases-template.md)）：

| 字段 | 内容 |
|---|---|
| CASE-ID | 稳定锚点——与自动化代码双向锚定（代码用例名包含 `CASE-xxx` / cases.md CASE-xxx） |
| 优先级 | P0 / P1 / P2 |
| 类型 | 正向流 / 异常流 |
| **覆盖 SR / US** | 本条 case 覆盖的 SR / US（**顶部覆盖矩阵从这些字段派生**，见下）+ TIA 选跑锚点 |
| 前置条件 | 站点 + 物料 + 数据态（建成可复用前置依赖项，见 Step 4） |
| **物料依赖** | 本条用例所需测试资产标识（路径 / 版本 / 校验值；**顶部物料清单从这些字段 bottom-up 汇总**，见下） |
| 操作步骤 | 意图级步骤（规格先行期作规格；代码落地后步骤 SoT 移到自动化代码，此处保留意图 + 断言点） |
| 断言点（分级） | **关键数据流用例**填 ①UI + ②trace + ③数据符合性三层；**UI/交互类用例**填 ①UI（+ 必要时轻断言 API 状态码）。③ 对着设计契约不对易变 schema（分级判据见「核心理念 · 断言分级」）。每层期望值尾注 design 条款引用（判据溯源，见核心理念 · 白盒判据纪律）|
| 风险备注（可选） | 规格期已知的依赖风险 / BLOCKED 预判（如"依赖 X 未实现 → BLOCKED 风险"）——规格级验收口径，跟 §已知降级清单互指 |

**执行期字段不进本表**：状态 / 证据 / issue 号是执行产物，落 `.spec/wip/e2e-<suite>/` 当次 run 记录（Step 5），cases.md 不承载。规格期已知的依赖风险 / BLOCKED 预判**保留在 cases.md §已知降级清单**（+ 用例卡片「风险备注」行）——那是规格级验收口径（"验到哪算通过"），不是执行状态。

**覆盖矩阵 = 派生视图（不手工独立维护）**：每条 case 带「覆盖 SR / US」字段是覆盖度的 **SoT**；cases.md 顶部覆盖矩阵是从各 case 该字段**汇总的派生表**（将来可脚本生成）+ 一张"哪些 SR 未被覆盖"的反查表。默认并入 cases.md 顶部，**不产独立 coverage.md**（除非大套件按输出契约拆分）。

**物料清单 = bottom-up 汇总（不顶层手工列）**：先写 case 标各自「物料依赖」→ 汇总到 cases.md 顶部 §物料清单（路径 / 校验值 / 缺失补法）。默认并入 cases.md 顶部，**不产独立 materials.md**（除非大套件 / 物料跨套件复用才拆）。

**拆解姿势三则**（实战回灌）：

- **正向断言可失败性答不上时**（如"好文件下抽查 sha256 验不出漏校验"）→ 标注「由哪条负向用例兜住」，不硬凑、不误删
- **设计超前于实现的用例**：按「阶段性负向契约」写（验未实现期显式报错不静默）+ 备注转正条件（实现落地后翻转为正向用例）——不按当前实现预判成"不测"
- **回归类用例（验产品修复）**：前置 / 可失败性描述以 **ADR 选定方案**为准（重读 ADR「改造」段），issue 台账记载只作历史锚——防照 issue 临时记载把被否决方案当修复机制

### Step 2：专业评审（reviewer 预审）—— 用例设计的一次性 Gate

用例**意图规格**拆完，先评审再执行。**这是用例设计的一次性 Gate，不是 E2E 结果的业务确认，也不是每次执行都重过评审**：

- **reviewer agent 预审**——用例完整性 + 断言分级是否得当（关键数据流够"验对"不只"验通"、UI 类不过度堆三层）+ 覆盖矩阵（派生视图）无空白 + 异常流是否覆盖（不只正向流）
- finding 修完才进执行——**未通过专业评审的用例不准跑**；本步骤不代替 Step 6 的 E2E 软确认
- **重跑不重审（机械判据）**——用例设计通过评审后，同一套件后续重跑（Step 0 触发）不重走评审。判据机械化：**cases.md 有 diff（规格新增 / 实质改动）→ 重过本 Gate；仅 wip 新增 run 记录（执行）→ 不重审**。Case / Run 分层保证规格变更与状态刷新分属两处，不会混在同一文件 diff 里
- **方向**：稳定套件长期目标进 CI 自动跑（已评审用例 + 项目选定的 E2E 自动化代码纳入流水线，专业评审前移到设计期，执行自动化）

### Step 3：环境 + 物料就绪

执行前核对（物料声明是 SSOT，字节在测试机）：

- **站点连通**——按 `env.md` 站点表逐站验 Web / API 入口 + 项目声明的证据抓取入口可达
- **物料齐 + 校验对**——按 cases.md 顶部 §物料清单（大套件拆分时按独立 `materials.md`）逐包核对测试机路径存在 + sha256 / 大小匹配；**缺则按"缺失补法"先补再跑**
- 环境动作按 Step 0 判定执行（换包 / 重部署 / 重建 / 刷新前端 / 不动）

env.md 模板见 [`references/env-template.md`](references/env-template.md)；物料清单模板见 cases.md 顶部小节（大套件拆分用 [`references/materials-template.md`](references/materials-template.md)）。

### Step 4：执行编排

把用例排成可执行顺序：

- **前置条件建成"前置依赖项"**（如 setup project / fixture）——"起服务、备数据、登录态"是显式依赖项，不隐式假设
- **优先级 tag**——P0/P1/P2 打 tag，P0 先跑
- **依赖 DAG 拓扑排序**——用例间依赖（如"先提交才能处理"）建 DAG，拓扑排执行顺序
- **断链下游 skip**——依赖链断了的下游用例自动 skip（serial 语义），按未执行记录；skip 不等于 PASS

产出"执行顺序清单"交项目指定的自动化实现角色。

### Step 5：执行 + 回填

**白盒取证三段式（自动化实现角色实现白盒断言的工程骨架）**：

- **触发**：让"这一次操作"携带已知 trace 锚点——发请求带 `x-trace-id` 头 / 前端场景从响应头或页面注入取回 trace_id（对齐 [`tracing-standards §2.1`](../../rules/tracing-standards.md)）。trace_id 是 ② 层断言的唯一关联锚点
- **取证**：**轮询 + 超时内建**——trace / DB 落库有秒级延迟，②③ 层断言一律 retry+timeout（如 poll 每 2s / 上限 30s）；无轮询直查是白盒 E2E 第一 flaky 源
- **断言**：先定位（按 service / span 名选调用链段、按业务键选数据记录）再断值（终态 / 数量 / 一致性标识）
- **helper 三层组织**：连接层（一处建立 UI / API / 数据存储访问上下文）→ 领域动词 helper（如 `getBusinessRecord(id)` / `getProcessState(id)`，测试体不裸写存储查询）→ 测试体（UI 动作 + 深断言交替）
- **探索与固化分离**：agent 真跑探索（项目允许的交互工具 / 手驱界面）只用于**用例生成期与失败诊断期**；回归期一律跑固化的确定性自动化代码
- **P0 探针实核**：P0 用例的 ② 层至少给一个实核过的探针锚点（file:line 或已确认存在的日志行）——判据溯设计、探针溯代码分工不变，但零探针的断言可实现性无保障；给一个即可，其余留自动化实现角色

- **实现角色从意图产自动化用例并真跑**（test-engineer 不写代码，交接的是"用例意图规格 + 执行顺序清单"——传意图共识，非逐步操作脚本；实现角色据意图 + 断言点自主实现步骤 / 选择器或调用参数 / 断言）
- **CASE-ID 双向锚定**——自动化用例名带 CASE-ID，与 cases.md 条目一一对应；台账 ↔ 代码靠 CASE-ID 互查治 drift（代码是步骤 SoT，cases.md 是意图 + 断言点 SoT）
- **test-engineer 看抨结果**——按用例断言分级逐条核：关键数据流用例核三层（① UI 截图对 ② trace 串后端调用链对 ③ `<data-store>` 数据跟设计契约对）；UI / 交互类核 ① UI（+ 轻断言 API 状态码）
- **归档重证据**到 `.spec/wip/e2e-<suite>/evidence/run-<id>/`（按 run 分目录；失败自动留图 + trace + 日志摘录）——执行日志进 evidence，不堆 run 结果表
- **结果回填当次 run 记录**（`.spec/wip/e2e-<suite>/run-<id>.md` 逐 case 结果表：通过 / 失败 / BLOCKED / 未执行或 skip + 证据链接 + issue 号）= 完成标记，**不动 cases.md**（Case / Run 分层：cases.md 是 Case 库，run 文件是 Run 记录——结果回写 run 不改写 case）

### Step 6：问题闭环 + 收尾

- **失败进 Issue**——跑出的前端 gap / 后端 bug 在 `.spec/issues/issues.md` 登记并走 [`issue-handling SKILL`](../issue-handling/SKILL.md)。需要跨角色跟踪时由 Issue 项目制品引用原问题、失败 Case/Run 及修订，不另抄问题状态；run 行内链 issue 号。仅定位交回不标已修/已闭，不清除同一 E2E 的其他阻塞。
- **修复后重跑**——根据实际修复修订和已批准预期，在新的 run 保存相关用例与冒烟结果、关联旧失败和 Issue，不覆写旧 run。Issue 独立复核与两方接纳按原 owner；修复声明不是验收通过，拒绝或复发时重开原 Issue。E2E 套件逐项核剩余失败、环境和确认，不能因一个 Issue 关闭就自动通过套件。
- **终态对账**（数据来源 = 当次 run 记录逐 case 结果表）返回主会话：
  - 用例总数 / 按优先级分布 + 本 run 范围（TIA 选出的 case 集）
  - 终态分布（通过 / 失败 / BLOCKED / 未执行或 skip）
  - 开出 issue 数 + 残留未闭环 issue 清单
  - 覆盖矩阵（从各 case「覆盖 SR」字段派生的视图）是否无空白
  - 残留 BLOCKED 清单（环境受限项，注明原因，不算失败）
- **E2E 软确认**：以精确 Run、被测版本、环境和逐 Case 结果为对象，在本次交接时明确实际产出人与接收/验收人；由 Host 展示表单，只将实际业务责任人已表达的 accept、reject、defer、skip 或 proceed-with-risk 交 `add checkpoint` 记录一次。无业务确认不得代签或把测试结果写成已验收；用户对精确对象、理由、范围、风险和下一步明确决定带风险继续时，Host 直接记录并续接，不二次询问。skip 不等于 PASS，失败、阻塞和未执行仍按真实结果保留并关联 Issue。
- **确认记录**：Run 模板预置 `## 确认记录`。人对精确真实 E2E 成果作出真实决定后，Host 才调用 `add checkpoint`；任一落点失败都不声称完成。完整语义由 Workflow 的“三处框架级软确认”统一定义，本 Skill 不复制第二套合同。
- **持久留痕与交回**：失败关联原 Issue；当前 E2E 交回逐 case 结果、被测/环境修订、未决及真实确认；套件 README 只摘要引用实际 run，不作为第二结果账。run 与证据按上表及 `skill:knowledge-curate` 保留/归档；摘要或已转 Issue 不能替代唯一原始证据。需要当前任务/批次接纳时分别交回对应负责人，不自动启动发布或宣布整个交付完成。

---

## 失败模式速查

| 错误 | 修正 |
|---|---|
| 把跨特性套件塞进单特性 design-workspace 目录 | 必落 `.spec/e2e/<suite>/`——E2E 按套件组织，跨特性是常态 |
| 关键数据流用例只验 ① UI"点了不报错"就标通过 | 关键数据流用例（提交 / 跨服务处理 / 状态流转 / 结果确认）三层缺一不可——必验 ② 后端数据流（trace）+ ③ 数据符合性 |
| UI / 交互类用例硬凑三层断言 | 断言分级——UI 校验 / 表单 / 禁用态类只需 ①UI（+ 轻断言 API 状态码），不强求 ③DB+trace（过度 = 负担 + 断言易变实现）|
| ③ 层断言写死易变内部 schema | ③ 对着设计契约（业务状态流转 / 数据一致性 / 业务终态）——不断言易变表结构（防"断言内部实现"反模式）|
| 判据从实现代码读出（照现状写期望值） | 同源污染 = "白盒全通"——判据必溯 design 条款（探针才允许来自代码，见核心理念 · 白盒判据纪律）|
| ②③ 层断言不带轮询直查 trace/DB | 落库秒级延迟必 flaky——取证一律 retry+timeout（见 Step 5 三段式）|
| 断言从未红过就上回归 | 可失败性存疑——P0 断言过变异思想实验（后端漏做这步会红吗），必要时故意断链验一次红 |
| 组合判据把拼合缝当已定义 | 2+ 条款拼合的语义缝隙（时机 / 上限 / 优先序）无条款 → 进设计 gap 清单，不作硬判据 |
| 回归用例照 issue 记载写修复机制 | 以 ADR 选定方案为准（重读「改造」段）——issue 临时记载可能是被否决方案，只作历史锚 |
| design / ADR 条款修订后套件用例照旧执行 | 判据已过期——先按「覆盖 SR」+ 判据尾注定位受影响用例刷新规格（Step 2 重新评审），再重验（见 Step 0 设计变更联动） |
| 想在 E2E 里穷举后端边界正确性 | 金字塔——E2E 只覆盖关键旅程，边界穷举归集成测试（test-design 规划），E2E 不背 |
| **把执行状态写进 cases.md** | Case / Run 分层——状态 / 证据 / issue 归 `.spec/wip/e2e-<suite>/` 当次 run 记录；cases.md 只随规格变更而改（diff 即重新评审信号） |
| run 记录 / 证据提交进套件目录 | 过程产物归 `.spec/wip/e2e-<suite>/`（活动 10 归档）——套件目录只有 cases.md 规格 |
| 手工维护独立 coverage 矩阵 / 顶层物料清单 | 覆盖矩阵从各 case「覆盖 SR」字段派生；物料从各 case「物料依赖」bottom-up 汇总——不手工独立维护 |
| cases.md 步骤与自动化代码两处重叠维护 | 代码落地后代码是步骤 SoT，cases.md 保留意图 + 断言点；靠 CASE-ID 锚定，不双份步骤 |
| 用例没人评审直接跑 | Step 2 reviewer 专业预审是用例设计 Gate，未评审不准跑；E2E 结果的业务确认在 Step 6 单独处理 |
| 依赖断链的 skip 被记为通过 | skip 是未执行状态，必须单列；skip 不等于 PASS |
| 物料缺 / 校验不对就开跑 | Step 3 必核对 materials.md，缺则按"缺失补法"先补 |
| 后端只改一点就只跑受影响用例、不跑冒烟主线 | 改后端必带冒烟主线（防牵一发动全身） |
| 跑出的真 bug 只记台账不开 issue | 真问题必走 `.spec/issues/` issue-handling 闭环，台账只是链接 |
| 修复后不重跑就标通过 | 修复后必重跑对应用例 + 冒烟（按评审跟修验证铁律） |
| 把环境受限 BLOCKED 当失败 | BLOCKED（如外部依赖凭据暂不可用）注明原因、不算失败，单列清单 |
| test-engineer 自己写自动化实现代码 | 越界——代码交项目指定的实现角色，test-engineer 只产规格 + 编排 + 抨账 |
| 前端没实现就提前标"不测 / 跳过" | 按设计驱动生成用例，前端无对应页面记"失败 + 前端未实现"暴露 gap |

## 与其他规则 / SKILL 的关系

- **前置**：[`test-design`](../test-design/SKILL.md)（规划测什么——本 SKILL `requires: [test-design]`）
- **执行实施**：项目指定的实现角色按"用例规格 + 执行顺序清单"写自动化用例（test-engineer 不写代码）
- **问题闭环**：[`issue-handling`](../issue-handling/SKILL.md)（跑出的真问题走第二入口）
- **评审**：[`code-review`](../code-review/SKILL.md)（Step 2 reviewer 预审用例意图规格）；活动 9 部署后 E2E 独立任务的结果评审按 cases.md 顶部覆盖派生表核验覆盖度
- **调试**：[`systematic-debugging`](../systematic-debugging/SKILL.md)（定位跑出的后端 bug）
- **写作准则**：[`design-doc-writing-style`](../../rules/design-doc-writing-style.md)（台账按写作准则）
- **形态隔离**：[`abstraction-principles §1.3`](../../rules/abstraction-principles.md)（跨形态 <form-a> / <form-b> / <form-c> 验证场景识别）
