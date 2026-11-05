# Article Management test-plan r05 production wiring 独立评审 r01

- **评审日期**：2026-10-23
- **Reviewer 身份**：独立 Reviewer，非 test-plan r05 作者
- **objectType**：design（测试规划/验收对象）
- **固定分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **被审对象**：`.add/project/knowledge/design-workspace/article-management/test-plan.md`
- **revision**：`article-management-test-plan-ir01-r05`
- **git blob**：`4a329a4f37b0437d9e49c04c11b439b204229065`
- **SHA256**：`C7875359D8269E8CFC93D59F179D9A7B8D38C1A835E4DF3F9D6D321EABF49F72`
- **状态声明**：`planned-not-executed`
- **Test-plan Gate**：**FAIL**

## 1. 结论摘要

r05 正确保持了规划、任务登记、实现、自动化测试、真实环境专项和 E2E 之间的证据边界；配置、Journal、唯一 router、worker、lifecycle、数据环境、清理责任以及后续真实验证的主要风险维度已形成较完整覆盖，也没有冒称 `article-management-S4.6` 已登记、已安排或已完成。

但固定 r05 仍有两个 MEDIUM finding：

1. `A-WIRE-WKR-07` 对固定设计中的 transient / no-op / terminal 分类使用矛盾的计数代称，不能唯一映射全部输入；
2. 唯一 S4.6 建议尚未提供足以直接冻结精确写集、task requires、验证命令/责任和批次依赖的交接合同。

因此 r05 当前不足以让 Task-Planner 无额外方案选择地创建“一个且仅一个”精确、可执行的 `article-management-S4.6`。原 `B6A-G4-R01-F001` 保持 OPEN，不能因本次 test-plan 更新自动关闭。

## 2. 固定依据

| 对象 | 固定修订 / SHA256 | 本次用途 |
|---|---|---|
| `test-plan.md` | r05 / `C7875359D8269E8CFC93D59F179D9A7B8D38C1A835E4DF3F9D6D321EABF49F72` | 直接被审对象 |
| `design.md` | r10 / `6785977683CF0F59E277667425F7FEE9714EBBD436FBDFE046E607F2E77CA67E` | 固定 production wiring 行为合同 |
| `stories.md` | r07 / `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05` | 验收与消费者边界 |
| `review-design-r10-production-wiring-r01.md` | `1EB2D78179D40747904CCC8F5E975FAB4A5CD4205C754567F6B5C589BAFC001E` | design-side Gate 与后继条件 |
| `review-b6a-g4-r01.md` | `B6A-G4-R01-F001` | 原 planning finding 与关闭条件 |
| `tasks.md`、`ir-01-content-community-batches.md` | 当前工作树固定读取 | 核 S4.6 未登记、既有任务边界和批次/DAG |

开始评审及落报告前均重新核对 branch、HEAD、revision、test-plan blob、上述四个固定 SHA256 和报告路径占用状态；均匹配，未发生对象漂移。未读取允许范围外的额外直接引用文件。

## 3. 两轮实际检查范围

### Round 1：r05 直接审查

逐项检查：

- source revision/hash、`planned-not-executed` 及证据状态；
- 配置 missing/empty/format/range/cross-validation、retained 显式空/缺失和敏感去除；
- Journal root、权限、symlink/逃逸、writer lock、keyring/key ID、namespace/domain；
- 唯一 `cmd/han-server`、三条 exact DELETE、404/405/`Allow`、Trace 与安全链顺序、Identity route 回归；
- `pending_journal` 补写、discovery/claim/lease、即时首轮、ticker+wake、batch/backoff 和错误分类；
- worker terminal failure 到全局 lifecycle，cancel/drain/shared deadline、shutdown timeout、reverse unwind 与 DB 最后关闭；
- 各场景最低充分层级、断言/证据、数据/环境和清理责任；
- S4.6 局部开发闭环与 PostgreSQL、Unix 持久卷、部署及真实 E2E 的分离；
- 任务建议、执行顺序、Gate 和未执行声明的一致性。

除 findings 所列问题外，未发现 r05 冒称任务登记、实现、自动化运行、真实 PostgreSQL、Unix 持久卷、Host、部署、E2E、容量或 production readiness 已完成。

### Round 2：消费者、上游与交接审查

沿下列真实依据有界核对：

- design r10 的候选实现影响面、配置/router、worker 错误分类和 lifecycle 所有权；
- stories r07 的验收及非目标边界；
- production wiring design review 的 Design-side Gate、唯一 S4.6 后继条件；
- 原 `B6A-G4-R01-F001` 对精确 owner、写集与 requires 的关闭要求；
- Tasks 中 S4.1～S4.5 的现有边界，以及 `article-management-S4.6` 当前不存在的事实；
- Plan 的 B6A、B7、B12、DAG 和“未安排范围：无”的当前登记；
- `go-service` profile 中对测试规划直接适用的测试、安全、并发、错误和资源释放要求。

核对结果：

- Tasks/Plan 当前均没有 `article-management-S4.6`；
- r05 正确写为“建议且仅建议创建”“未登记/未安排”；
- B6A G4 Gate 仍 FAIL，原 finding 未被本规划自动关闭；
- 真实 PostgreSQL、受支持 Unix 持久卷、部署和 E2E 没有被错误设为 S4.6 局部开发 task 的关闭前置；
- 但 S4.6 的精确规划交接信息仍不完整，见 F002。

## 4. Findings

### MEDIUM

#### `AM-TP-R05-R01-F001` — `A-WIRE-WKR-07` 的错误分类计数与固定设计不一致

- **位置**
  - `test-plan.md:271`
  - 对照 `design.md:1818-1824`，尤其 §3.4.5 第 5 点。
- **违反依据**
  - design r10 明确规定：
    - transient：DB 暂不可用、Journal 暂不可用、lease 竞争、operation budget deadline；
    - no-op：completed、not-claimed；
    - terminal：缺 key、Journal 损坏，以及 namespace/domain/tombstone/Article/operation 绑定不一致。
  - `add-test-design` 要求场景预期和可观察证据明确，不能由实施者猜测分类。
- **触发**
  - S4.6 按 `A-WIRE-WKR-07` 建表驱动用例时，需要把该行列出的每种错误映射到唯一分类和 lifecycle 结果。
- **证据**
  - r05 写为“前四类中前三类为 transient、completed/not-claimed 为 no-op、后四类为 terminal”。
  - 该表述既把固定设计中的第 4 个 transient（operation deadline）排除在“前三类”之外，又使“后四类”与 completed/not-claimed、缺 key、链损坏、绑定不一致的分组发生重叠或计数歧义。
- **影响**
  - operation deadline 可能被实现或测试误归为 terminal；
  - no-op 或 terminal 安全错误可能被错误重试；
  - 应退避的瞬时故障可能触发全局关闭，或应 fail closed 的安全/不变量错误不能触发全局 lifecycle failure。
- **最小修复**
  - 不再使用“前三类/后四类”计数代称；
  - 在该场景逐项显式列出每个输入及唯一分类，并分别声明退避、no-op 和全局失败的可观察结果；
  - 保留低敏 bucket、canary secret/PII 扫描和禁止输出 fixture 原值的要求。
- **验证**
  - 与 design r10 §3.4.5 第 5 点逐项对账；
  - 确认每个表驱动 fixture 只映射一个分类，operation deadline 明确为 transient；
  - 独立复审修订后的场景文本及覆盖映射。

#### `AM-TP-R05-R01-F002` — S4.6 任务建议缺少可直接冻结的精确规划交接合同

- **位置**
  - `test-plan.md:345-349`
  - `test-plan.md:396-426`，尤其 S4.6 建议行 `:409`、后续专项 `:410-414` 和执行顺序 `:418-426`
  - 对照 `design.md:523-533`、`review-b6a-g4-r01.md:75-89`。
- **违反依据**
  - `add-test-design` Step 5 要求每项任务建议写明精确所有权、依赖、命令/操作、通过证据和停止条件；
  - 原 `B6A-G4-R01-F001` 的最小关闭方向要求后继 task 登记精确 owner、写集与 requires；
  - 本轮验收要求 r05 能直接支持 Task-Planner 定义精确 owner、写集、requires、验证责任和批次依赖。
- **触发**
  - Task-Planner 尝试在当前 52 个 task 和现有 DAG 中登记唯一 `article-management-S4.6`。
- **证据**
  - r05 已给出角色级 owner、场景关闭范围和停止条件，但没有给出精确文件写集、task requires、局部测试命令/责任，以及 S4.6 应插入的批次和显式前后依赖。
  - design r10 `:523-533` 明确称文件表只是“实现影响面，不是 task 清单”，且仍含“可新增”“或现有删除 repository 文件”等未冻结选择，不能直接替代 task 写集。
  - 当前 Plan 没有 S4.6，仍声明“未安排范围：无”；B7 已包含 S4.3/S4.5，B12 已包含 S1.4，但 r05 没有把 S4.6 与这些既有条目的 batch/requires 关系冻结为唯一解释。
- **影响**
  - Task-Planner 必须自行补做方案选择，才可决定写哪些文件、依赖哪些 task、在哪个批次插入及运行哪些局部验证；
  - 可能与已关闭 S4.1 或后续 S4.3/S4.5/S1.4 发生写集/责任重叠，或再创建第二个 wiring task；
  - 不能仅凭 r05 证明“一个且仅一个最小 S4.6”已经具备可登记性。
- **最小修复**
  - 为唯一 S4.6 增加单一、无备选分支的 Task-Planner handoff：
    1. 冻结精确 task owner 和逐文件写集，明确只读复用及禁止修改范围；
    2. 列出 task-level requires，并明确与 S4.1、S4.3、S4.5、S1.4 的依赖方向；
    3. 指定局部自动化验证责任和可执行命令/操作，覆盖 CFG/JRN 组件部分/RTR/WKR 确定性部分/LIFE；
    4. 指定唯一批次位置和 DAG 边，消除 Plan“未安排范围：无”与新增任务的冲突；
    5. 明确不得另建第二 composition root、第二 wiring task，且不得把真实 PostgreSQL、Unix 持久卷、部署或 E2E 并入本 task 关闭 Gate。
- **验证**
  - Task-Planner 能只依据固定 r10/r07、修订后的 test-plan 及当前 Tasks/Plan，生成一个无占位、无二选一的 S4.6 条目；
  - 写集与现有 owner 对账无冲突，每个场景有唯一关闭责任；
  - requires 和批次 DAG 可拓扑排序，S4.3/S4.5/S1.4 后继责任仍独立；
  - 规划复审确认未创建重复 wiring task、未改变固定行为合同。

CRITICAL / HIGH / LOW：无。

## 5. 待决定与非阻断建议

### 待决定

无新的产品或架构方向决定。F002 要求 Task-Planner 交接合同明确化，不授权 Reviewer 选择具体文件或批次。

### 非阻断建议

1. 修订 F001 时可为每个错误分类固定低敏 bucket 名称，便于后续安全扫描和 lifecycle 断言保持稳定；不得记录对象内容、路径、DSN、key 或 PII。
2. S4.6 的后续规划应复用 r05 已明确的局部/专项/E2E 分层，不因补齐任务精度而扩大成 production readiness 任务。

## 6. 未验证范围与证据限制

- 未运行 Go test、race、vet、build 或数据库测试；本次对象为测试规划文档。
- 未读取、打印或使用 `HAN_TEST_POSTGRES_DSN` 或其他凭据。
- 未验证真实 PostgreSQL transaction/lock/commit outcome。
- 未验证受支持 Unix 持久卷的 owner-only、flock、file/directory fsync、崩溃、断电或磁盘故障。
- 未启动真实 Host、监听端口、后台进程、部署、signal/exit-code 测试或 E2E。
- 未验证容量、吞吐、P95、告警阈值、运行留存或 production readiness。
- 未对允许依赖范围外的代码或全仓进行无界审查；设计 r10 的既定行为不在本次重新设计。
- 本结论只评价固定 r05 的规划正确性和可交接性，不等同于人的采用、任务创建授权、实现授权或风险接受。

## 7. 执行的静态命令与结果

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current`、`git rev-parse HEAD` | PASS；`main` / 固定 HEAD |
| `Get-FileHash -Algorithm SHA256` 核 test-plan、design、stories、design review | PASS；均匹配固定 SHA256 |
| `git hash-object -- <test-plan>` | PASS；匹配固定 blob |
| `Select-String` 核 test-plan revision | PASS；匹配 r05 |
| `Test-Path` 核报告路径（写入前） | PASS；路径未被占用 |
| `Get-Content -Encoding utf8`、`Select-String` 定点读取规则、profile、r05、r10/r07、design review、B6A finding、Tasks/Plan | 完成 Round 1 / Round 2 |
| 真实测试、数据库、外部状态操作 | NOT RUN；按授权边界不执行 |
| `git diff --check -- .spec/wip/article-management/review-test-plan-r05-production-wiring-r01.md` | PASS |

## 8. Gate 与 Task-Planner 交回条件

### Test-plan Gate：FAIL

复审前置：

1. Test-Engineer 仅修订 test-plan，关闭 F001 的逐项错误分类歧义；
2. 补齐 F002 的唯一 S4.6 精确规划交接合同，不修改 design r10 固定行为；
3. 修订对象重新固定 revision、SHA256/blob，并由非作者 Reviewer 对 F001/F002 定点复审；
4. 复审 PASS 后，Task-Planner 才可创建且仅创建一个 `article-management-S4.6`；
5. Task-Planner 登记时必须给出精确 owner、写集、requires、验证责任/命令、批次位置与 DAG，并同步 Tasks/Plan，不能写成已实现或已测试；
6. S4.6 局部开发闭环不得以真实 PostgreSQL、Unix 持久卷、部署或 E2E 为关闭前置；这些继续由既有/后续专项和 `article-management-S1.4` 独立关闭；
7. 完成 task 登记和 planning review 后，才可回到 `B6A-G4-R01-F001` 判断是否满足 planning finding 的关闭条件。

## 9. 实际改动

- 仅新增本独立报告：`.spec/wip/article-management/review-test-plan-r05-production-wiring-r01.md`
- 未修改 test-plan、design、stories、Tasks、Plan、代码或任何既有报告。
- 未执行 stage、commit、push、PR、部署、clean、reset、checkout 或分支切换。
