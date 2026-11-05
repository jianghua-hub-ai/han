# Article Management test-plan r06 production wiring 定点复审 r02

- **复审日期**：2026-10-23
- **Reviewer 身份**：原独立 Reviewer，非 test-plan r05/r06 作者
- **objectType**：design（测试规划/验收对象）
- **固定分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **被审对象**：`.add/project/knowledge/design-workspace/article-management/test-plan.md`
- **revision**：`article-management-test-plan-ir01-r06`
- **git blob**：`41f0ec70de4076dc9c3114dc6153250a5351c299`
- **SHA256**：`9697316AF68848DE9898556DA673ECE531F4659AAB91A99ED7581B54E924F9EC`
- **状态**：`planned-not-executed`
- **原报告**：`.spec/wip/article-management/review-test-plan-r05-production-wiring-r01.md`
- **原报告 SHA256**：`2A080E134B57280D5629551A00B1DC2CB0F59008038B24A1CAC099C013B6BE07`
- **Test-plan Gate**：**PASS**

## 1. 定点结论

| 原 finding | 状态 | 结论 |
|---|---|---|
| `AM-TP-R05-R01-F001` | **CLOSED** | `A-WIRE-WKR-07` 已逐项固定 transient、no-op、terminal 的唯一映射及对应可观察结果，消除了 r05 的计数歧义。 |
| `AM-TP-R05-R01-F002` | **CLOSED** | §9.1 已冻结唯一 S4.6 的 owner、11 文件写集、只读/禁止范围、requires、批次、DAG、验证责任/命令及停止/关闭条件，足以直接交 Task-Planner 登记。 |

定点范围内未发现修订引入的新阻断 finding。r06 足以让 Task-Planner 无需新增架构或任务切分方案，创建且仅创建一个 `article-management-S4.6`。

本 PASS **只表示固定 r06 可交给 Task-Planner**。它不关闭 `B6A-G4-R01-F001`，不证明 S4.6 已登记、已安排、已授权实施、已实现或已测试，也不证明真实 PostgreSQL、Unix 持久卷、Host、部署、E2E、容量或 production readiness。

## 2. 固定对象与依据

| 对象 | 固定修订 / SHA256 | 用途 |
|---|---|---|
| `test-plan.md` | r06 / `9697316AF68848DE9898556DA673ECE531F4659AAB91A99ED7581B54E924F9EC` | 修复修订 |
| 原 r01 报告 | `2A080E134B57280D5629551A00B1DC2CB0F59008038B24A1CAC099C013B6BE07` | 原 finding、修复与复审条件 |
| `design.md` | r10 / `6785977683CF0F59E277667425F7FEE9714EBBD436FBDFE046E607F2E77CA67E` | 固定行为和实现影响面 |
| `stories.md` | r07 / `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05` | 验收边界 |
| production wiring design review | `1EB2D78179D40747904CCC8F5E975FAB4A5CD4205C754567F6B5C589BAFC001E` | Design-side Gate 与唯一后继条件 |
| `review-b6a-g4-r01.md` | `B6A-G4-R01-F001` | planning finding 及关闭链 |
| 当前 Tasks / Plan | 当前工作树定点读取 | S4.6 不存在、现有 owner/requires/DAG |

开始及落报告前均核 branch、HEAD、r06 revision/status/blob、固定 SHA256 和报告路径；未发现漂移或路径占用。

## 3. 两轮定点检查范围

### Round 1：原 finding 修复本身

仅检查：

- `A-WIRE-WKR-07` 的逐项分类、退避/no-op/全局 lifecycle 结果和敏感信息扫描；
- §9、§9.1、执行顺序及 Gate 中新增的唯一 S4.6 handoff；
- 修订文本是否仍保持 `planned-not-executed`，是否冒称 Tasks/Plan、实现或测试已完成。

未重开 r05 已通过且未变的配置、Journal、router、worker 其他场景、lifecycle、数据环境和清理范围。

### Round 2：紧邻消费者和交接风险

定点核对：

- design r10 的 transient/no-op/terminal 固定分类；
- design r10 的唯一 composition root 和候选实现影响面；
- 原 r01 报告及 `B6A-G4-R01-F001` 对精确 owner、写集、requires 的要求；
- Tasks 中两项直接 requires 的关闭状态、11 文件历史 owner 是否已释放、当前 active task 是否发生写集交叉；
- Plan 中现有 B6A、B7、B12 和 DAG；
- S4.3、S4.5、S1.4 的现有依赖与 r06 新增后继方向是否成环；
- Tasks/Plan 当前是否已出现 S4.6 或 `B6A-W`。

## 4. 原 finding 逐项复审

### `AM-TP-R05-R01-F001` — CLOSED

**修复位置**：`test-plan.md:283`

r06 已明确：

- transient：DB 暂不可用、Journal 暂不可用、lease 竞争、operation budget deadline；
- no-op：completed、not-claimed；
- terminal：缺 key、Journal 链/记录损坏、namespace/domain/tombstone/Article/operation 绑定不一致。

每个输入只映射一个分类，并分别给出：

- transient：有界 full-jitter 退避，不触发全局 failure；
- no-op：不重试、不升级告警、不改变完成事实；
- terminal：停止 discovery/新 claim，将首个失败送入全局 lifecycle failure；
- 所有分类只记录低敏 category/reason bucket，并扫描 key、路径、DSN、正文及 PII 泄漏。

该映射与 design r10 §3.4.5 一致；原“前三类/后四类”的重叠和漏项已消除，未发现紧邻新矛盾。

### `AM-TP-R05-R01-F002` — CLOSED

**修复位置**：`test-plan.md:421-487`，重点 §9.1。

#### 任务唯一性与所有权

- 唯一 task ID：`article-management-S4.6`；
- 唯一实现 owner：Article Backend-Dev；
- 非作者 Reviewer 和 Lead 的核证责任分开；
- 明确禁止第二 composition/router、第二 wiring task、公共合同扩张或条件性越界。

#### 精确写集

已冻结 11 个文件，其中现存 9 个、计划新增 2 个 `deletion_worker.go/_test.go`；同时列出只读复用和禁止修改范围。历史上涉及这些文件的 S5.2、S4.1 等任务均已 task-level 关闭并释放独占窗口；当前仅有的两个 active task 是 identity 浏览器/observability 范围，与该 11 文件无交叉。

若实施发现需要 migration、新第三方依赖、第二 composition/router、公共合同变更或第 12 个文件，必须停止并回规划/设计，不允许在 S4.6 内扩权。

#### requires 与 DAG

- 直接 requires 仅为已关闭的 `identity-auth-S5.2` 和 `article-management-S4.1`；
- 入边不依赖任何后继 task；
- 出边指向尚未执行的 S4.3、S4.5 和 S1.4 的 production wiring/真实边界部分；
- 当前 S4.3、S4.5、S1.4 均不存在返回 S4.6 或其直接前置的依赖，因此未形成环；
- `B6A-W Article production wiring` 位于 B6A 后，并在 B7 的 S4.3/S4.5 及 B12 的 S1.4 前形成后继约束；B6B/B6C/B6D 未被虚构为业务 requires。

Task-Planner 可据此把当前 Plan 展开为 B6A 后的独立 `B6A-W` 分支，并在进入相关 B7/B12 消费者前汇合，无需选择第二种任务切分或 wiring owner。

#### 验证和关闭边界

- 已列出 gofmt、article/han-server 普通测试、race、vet、build、module verify 和 scoped diff-check；
- Backend-Dev、非作者 Reviewer、Lead 的证据责任明确；
- 关闭只覆盖 fake/组件和确定性局部自动化；
- 真实 DSN、真实 PostgreSQL、受支持 Unix 持久卷、部署、Host、signal/exit E2E、容量和 production readiness 明确不属于 S4.6 关闭前置。

#### 状态真实性

Tasks/Plan 当前均不存在 `article-management-S4.6` 或 `B6A-W`。r06 保持“未登记/未安排”和 `planned-not-executed`，没有冒称实现、测试或环境验证完成。

## 5. 新 findings

无。

CRITICAL / HIGH / MEDIUM / LOW：均无新增 finding。

## 6. 待决定与非阻断交接说明

### 待决定

无新的架构、契约或任务切分决定。

### 非阻断交接说明

1. §10 第 2 步中的“关闭 planning finding”只能解释为未来登记、Plan 同步和相应 planning review 完成后的独立动作；本报告不授权 Task-Planner 以登记动作自行关闭 Reviewer finding。
2. Task-Planner 落账时应显式修正当前 Plan 的“未安排范围：无”，并保留其他 52 个任务的既有历史，不覆盖或重写其完成证据。
3. 共享工作树在实际认领前仍须重新核文件 owner 和在途修改；本次只证明当前登记不存在写集冲突。

## 7. 静态命令与结果

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current`、`git rev-parse HEAD` | PASS；`main` / 固定 HEAD |
| `Get-FileHash -Algorithm SHA256` 核 r06、原 r01、design r10、stories r07、source design review | PASS；全部匹配 |
| `git hash-object -- <test-plan>` | PASS；匹配固定 r06 blob |
| `Select-String` 核 revision/status | PASS；r06 / `planned-not-executed` |
| `Test-Path` 核报告路径 | PASS；写入前不存在 |
| `Get-Content -Encoding utf8` 定点读取 F001 修复、§9.1、命令、关闭条件和 Gate | 完成 |
| `Select-String` 核 Tasks/Plan 中 S4.6、`B6A-W` | PASS；均无匹配 |
| `Test-Path` 与 `Select-String` 核 11 文件存在性和历史 task owner | PASS；9 个现存、2 个明确新增；当前 active task 无交叉 |
| 定点核 S4.3、S4.5、S1.4、S5.2、S4.1 requires/status 与现有批次 DAG | PASS；未发现依赖环或未释放 owner |
| 尝试以 r05 blob 值执行 `git diff` | 未采用；该 hash 仅为计算值、不是本地 Git object，命令返回 `bad object`，不影响基于固定 r06、原报告和定点章节的复审 |
| Go、DB、E2E、外部状态命令 | NOT RUN；按授权边界禁止 |
| `git diff --check -- .spec/wip/article-management/review-test-plan-r06-production-wiring-r02.md` | PASS |

## 8. 未验证范围与限制

- 未运行 Go test、race、vet、build、数据库或 E2E。
- 未读取、打印或使用任何 DSN/凭据。
- 未验证 11 文件实现是否存在未来代码级缺陷；实现尚未发生。
- 未验证真实 PostgreSQL、Unix 持久卷、Host、部署、signal/exit code、容量或 production readiness。
- 未对 r06 未变且前轮已通过的全量 test-plan 重新审查。
- 未修改 Tasks/Plan，因此 S4.6、B6A-W 和新增 DAG 当前仍只是已通过复审的规划 handoff。

## 9. Gate 与 Task-Planner 交回条件

### Test-plan Gate：PASS

Task-Planner 可接收固定 r06，并必须：

1. 创建且仅创建一个 `article-management-S4.6`；
2. 原样采用 §9.1 的 Article Backend-Dev owner、11 文件写集、只读/禁止范围和停止条件；
3. 登记直接 requires `identity-auth-S5.2`、`article-management-S4.1`；
4. 新增且仅新增 `B6A-W Article production wiring` 承载该 task，并同步其与 B6A、B7/S4.3/S4.5、B12/S1.4 的 DAG；
5. 修正 Plan 的“未安排范围：无”，但不得写成任务已启动、实现或测试完成；
6. 不创建第二个 production wiring task，不把真实 PostgreSQL、Unix 持久卷、部署或 E2E 并入 S4.6 局部关闭 Gate；
7. 完成 Tasks/Plan 登记及独立 planning review 后，再由有权角色判断 `B6A-G4-R01-F001` 是否关闭；本 PASS 本身不关闭该 finding；
8. 另行取得开发授权后才能启动实施。

## 10. 实际改动

- 仅新增本报告：`.spec/wip/article-management/review-test-plan-r06-production-wiring-r02.md`
- 未修改 test-plan、Tasks、Plan、design、stories、代码或原报告。
- 未执行 stage、commit、push、切分支、clean、reset、checkout、部署或清理。
