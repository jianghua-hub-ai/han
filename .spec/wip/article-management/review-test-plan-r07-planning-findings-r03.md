# Article Management test-plan r07 planning findings 定点复审 r03

- **日期**：2026-10-23
- **Reviewer**：Rawls，原 Test-plan 非作者独立 Reviewer；本轮未继续委派
- **objectType**：design（测试规划/验收对象）
- **分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **被审文件**：`.add/project/knowledge/design-workspace/article-management/test-plan.md`
- **revision**：`article-management-test-plan-ir01-r07`
- **SHA256**：`57F5C5150057603BE8C0E3FA00C9DC7959FA4A719FEF6B74C94CB3E83F1C794A`
- **git blob**：`b9e00b4c37d91baf9d8cd4bb0e3cc8529b7f94c0`
- **状态**：`planned-not-executed`
- **Test-plan Gate**：**FAIL**
- **可交 Task-Planner**：**否；须先完成本报告修复并定点复审**

## 1. 结论摘要

| 对象 | 定点状态 | 结论 |
|---|---|---|
| `ARTICLE-S4.6-PLAN-R01-F001` | **STILL OPEN / 部分修复** | 精确写集已从 11 文件扩为 13 文件，Article 私有双 build-tag 适配路径可实施；但关闭条件要求“两条平台分支编译”，现有命令只编译执行 Host 对应的一条分支，不能取得所声明的所有平台编译证据。 |
| `ARTICLE-S4.6-PLAN-R01-F002` | **CLOSED（test-plan 侧）** | integration-tag compile-only 命令会编译 `repository_integration_test.go`，`-run '^$'` 不执行其中 PostgreSQL 测试函数；文本明确禁止把该结果冒充真实 PostgreSQL 或 S4.3 证据。 |

另发现：

- 1 个 MEDIUM 新 finding：覆盖映射仍把唯一 S4.6 写成“未登记/未安排”，与同一 r07 其他章节确认“已登记 todo / 已安排 B6A-W”直接冲突；
- 1 个 LOW 新 finding：source review 追溯仍停在 r05 FAIL，未登记 r06 定点复审 PASS，导致历史 finding 状态描述失真。

因此 r07 当前不能作为单义、关闭型输入交 Task-Planner 同步 Tasks/Plan。本报告不关闭任何 Tasks/Plan finding、`B6A-G4-R01-F001` 或 Planning Review Gate。

## 2. 固定依据

| 对象 | 固定值 | 用途 |
|---|---|---|
| test-plan r07 | SHA256 `57F5C5150057603BE8C0E3FA00C9DC7959FA4A719FEF6B74C94CB3E83F1C794A`；blob `b9e00b4c37d91baf9d8cd4bb0e3cc8529b7f94c0` | 被审修订 |
| design r10 | SHA256 `6785977683CF0F59E277667425F7FEE9714EBBD436FBDFE046E607F2E77CA67E` | writer lock、fail closed、S4.6/S4.3 边界 |
| stories r07 | SHA256 `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05` | 验收边界 |
| Tasks/Plan planning review r01 | SHA256 `0AF25106B09D3D5E2FB42E6BA04BDB85C7A92F1E7CF7AB8A3CF342CE9350F65B` | 原 F001/F002 |
| test-plan r05 review r01 | SHA256 `2A080E134B57280D5629551A00B1DC2CB0F59008038B24A1CAC099C013B6BE07` | 历史 finding |
| test-plan r06 review r02 | SHA256 `B26C9E88932925CEC7229DCAFD971E2A47514184967279488FF37CB7F0F391F2` | r05 finding 已关闭及 r06 Gate PASS |

开始评审及落报告前均核 branch、HEAD、r07 revision/status/hash/blob 和报告路径；未发现漂移或路径占用。

## 3. 两轮定点范围

### Round 1：r07 直接修订

检查：

- revision、status、日期、source/review 追溯；
- §1.3 对 planning F001/F002 的整改记录；
- §4.10.2 的 Article 私有锁接口、Unix `flock`、unsupported fail closed 和 S4.6/S4.3 分界；
- §9.1 的 13 文件写集、只读/禁止范围、第 14 文件停止条件；
- integration-tag compile-only Gate；
- §10～§12 的顺序、Gate、未执行声明和 finding 状态；
- 由 11 文件扩为 13 文件后，任务建议、覆盖映射及关闭条件是否同步一致。

### Round 2：可实施性与消费者边界

有界读取：

- `internal/recovery/filelock_flock.go`：确认现役可靠锁依赖 Unix build tags、`unix.Flock`、owner-only regular file 和稳定 inode；
- `internal/recovery/filelock_unsupported.go`：确认其他平台 fail closed；
- `internal/article/deletion_journal.go`：确认当前仅有进程内 mutex、尚无跨进程 writer lock owner/Close；
- `internal/article/repository_integration_test.go`：确认 `//go:build integration`，且 PostgreSQL DSN 只在测试函数执行路径内读取。

据此核对：

- 13 文件路径能否承载实现；
- Host 原生命令是否真的覆盖两条互斥 build-tag 分支；
- compile-only Gate 是否会冒充 PostgreSQL 执行；
- S4.6 组件/fake/编译责任与 S4.3 真实 Unix/PostgreSQL 责任是否混淆；
- 是否新增第二 wiring task、改变 requires/B6A-W/后继 DAG，或冒称实现与真实验证完成。

## 4. 原 planning findings 状态

### `ARTICLE-S4.6-PLAN-R01-F001` — STILL OPEN / 部分修复

#### 已充分修复部分

- **位置**：`test-plan.md:264, 435, 442-490`
- 原 11 文件只新增：
  - `internal/article/deletion_filelock_flock.go`
  - `internal/article/deletion_filelock_unsupported.go`
- 两文件被限定为 Article 模块私有适配，不修改或导出 `internal/recovery` 原语；
- 支持 Unix 的文件负责可靠排他 `flock`，其他平台文件负责可编译和 fail closed；
- 需要 migration、新依赖、公共合同变更、Recovery 修改或第 14 个文件时必须停止；
- 该方向与 planning review 给出的“Article 成对平台 lock 文件”修复方向一致，现有 `golang.org/x/sys/unix` 使用模式也证明实现路径客观存在。

#### 仍未关闭部分

- **位置**
  - `test-plan.md:264, 271, 274`
  - `test-plan.md:474, 494-503, 509`
- **依据**
  - r07 明确要求“两条 build-tag 分支编译”“所有平台编译部分”；
  - flock 与 unsupported 文件使用互斥 GOOS build tags，同一次 Host 原生命令只会选中其中一条。
- **触发**
  - 在 Windows/其他 unsupported Host 执行 r07 命令时，Unix flock 文件不会编译；
  - 在受支持 Unix Host 执行时，unsupported 文件不会编译。
- **证据**
  - 普通 `go test`、race、vet、build 和新增 integration-tag 命令均未定义跨 GOOS 编译矩阵；
  - `-tags=integration` 只改变 integration tag，不改变 GOOS 选择，不能覆盖另一条平台文件。
- **影响**
  - 未被当前 Host 选择的第 12 或第 13 文件可存在类型/API/build-tag 错误，而全部列明命令仍通过；
  - Task 可能在未取得 r07 自己要求的“两条平台分支编译证据”时被错误关闭。
- **最小修复**
  - 在 §9.1 冻结明确的双分支编译矩阵：至少一个受支持 Unix 目标和一个 unsupported 目标；
  - 使用项目确认的 compile-only 方式，明确不执行测试、不访问 DSN，并负责清理任何编译输出；
  - 将该矩阵纳入 Backend-Dev、Reviewer 和 Lead 的关闭证据，而不是仅写抽象的“所有平台编译”。
- **验证**
  - 两个目标分别证明 flock/unsupported 文件被 Go 文件选择器实际纳入编译；
  - 任一分支故意制造类型错误时，对应 compile Gate 必须失败；
  - 不运行 PostgreSQL、Unix 持久卷行为或 E2E。

### `ARTICLE-S4.6-PLAN-R01-F002` — CLOSED（test-plan 侧）

- **位置**：`test-plan.md:501`
- 新增命令：
  - `go test -tags=integration ./internal/article -run '^$' -count=1`
- `repository_integration_test.go` 确有 `//go:build integration`；
- `-run '^$'` 不匹配现有测试函数，PostgreSQL pool/DSN 调用位于测试函数调用路径，不因该命令作为正常 test case 执行；
- r07 明确称其为 compile-only Gate，并声明不得作为真实 PostgreSQL、S4.3 或 production readiness 证据。

本结论只关闭 test-plan 对“第 11 个 owner 文件未编译”的文档缺口；Tasks/Plan 中的 planning finding 仍须由 Task-Planner 同步并交其原复审链处理。

## 5. 新 findings

### MEDIUM — `AM-TP-R07-R03-F001`：覆盖映射仍声称 S4.6 未登记，与 r07 当前事实冲突

- **位置**
  - 当前事实：`test-plan.md:31, 58, 424, 435, 444, 533-534, 553`
  - 冲突位置：`test-plan.md:371-375`
- **违反依据**
  - 测试规划的“场景 → 实际 task → 登记状态”映射必须引用同一 Tasks/Plan 事实；
  - 未登记、已登记 todo、已启动和已完成必须分开。
- **触发**
  - Task-Planner 按 §6 覆盖映射核场景所有权，同时按 §9.1 同步既有 S4.6/B6A-W。
- **证据**
  - r07 多处明确 planning review 已核得唯一 todo `article-management-S4.6` 和专属 `B6A-W`；
  - A-WIRE-CFG/JRN/RTR/WKR/LIFE 五行仍分别写“未登记/未安排”或“S4.6 未登记”。
- **影响**
  - 同一文档给出两种互斥任务状态；
  - 消费者可能重复创建 S4.6，或错误地把“同步既有任务”处理为“创建新任务”；
  - 无法满足“单义同步且不增加第二 wiring task”的交接条件。
- **最小修复**
  - 将五行登记状态统一为：唯一 S4.6/B6A-W 已登记/已安排、todo、findings OPEN、未启动；
  - 保留 S4.3/S4.5/S1.4 的独立后继和未执行边界；
  - 不修改 Tasks/Plan 状态本身。
- **验证**
  - 全文搜索 `S4.6 未登记`、`未登记/未安排`，不得再与当前登记事实冲突；
  - 全文仍只有一个 S4.6 和一个 B6A-W，不出现实现、测试或启动完成声明。

### LOW — `AM-TP-R07-R03-F002`：source review 追溯遗漏 r06 的独立 PASS

- **位置**
  - `test-plan.md:16-17, 43, 60-67, 532`
- **违反依据**
  - 修订来源和原 finding 状态必须保留最近一次独立复审证据，不能把已复审 PASS 写回“仍待首次复审”的历史状态。
- **触发**
  - 后续 Reviewer/Task-Planner 追溯 r05 finding 是否已在 r06 被独立关闭。
- **证据**
  - frontmatter 的 `source_test_plan_review` 仍指向 r05 FAIL 报告；
  - r07 未登记 `review-test-plan-r06-production-wiring-r02.md` 及其 SHA256；
  - r06 review r02 已明确 `AM-TP-R05-R01-F001/F002` CLOSED、Test-plan Gate PASS。
- **影响**
  - 历史审查链不完整，容易把“r07 新修订需要复审”误写成“r06 从未独立复审”；
  - 不直接改变 F001/F002 技术方案，但降低 finding 状态和来源的可审计性。
- **最小修复**
  - 在 metadata/source 表登记 r06 review r02 及固定 SHA256；
  - 明确 r06 PASS 只绑定 r06，不能自动外推到 r07；r07 仍需本轮独立复审。
- **验证**
  - source 元数据、来源表、§1.2 和 §11 对 r05/r06/r07 三个对象的 Gate 状态无矛盾。

CRITICAL：无。除上述项目外，未发现第二 wiring task、requires/B6A-W/后继 DAG 变化、真实环境并入局部 Gate或实现/测试完成冒称。

## 6. 待决定与非阻断建议

### 待决定

无新的架构方向决定。Article 私有双平台文件方向可实施；剩余问题是验证矩阵和文档一致性。

### 非阻断建议

1. 双 GOOS compile-only Gate 应只验证文件选择、类型/API 和依赖闭合，不替代 S4.3 的真实 Unix `flock`、fsync、崩溃或跨进程竞争测试。
2. 对“不得复制 Recovery 锁原语”的实现约束，后续代码 Review 应核其未导入/修改 Recovery 私有实现，同时允许 Article 通过自己的私有接口实现同等安全语义。

## 7. 静态验证

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current`、`git rev-parse HEAD` | PASS；固定 branch/HEAD |
| `Get-FileHash -Algorithm SHA256` 核 r07、design、stories、planning review、r05/r06 review | PASS |
| `git hash-object -- <test-plan>` | PASS；匹配固定 blob |
| `Select-String` 核 revision/status/date/source | revision/status/date 匹配；source review 存在 F002 所述追溯缺口 |
| 定点读取 §1.3、§4.10.2、§6、§9.1、§10～§12 | 完成 |
| 读取四个允许代码文件，核 build tags、锁实现路径、当前 Journal owner 和 integration test 执行路径 | 完成；未扩成代码质量评审 |
| 搜索全文 11/12/13 文件、S4.6 状态、compile-only、S4.3 边界 | 发现 F001 及新 finding |
| Go/PostgreSQL/Unix/部署/E2E | NOT RUN |
| `git diff --check -- .spec/wip/article-management/review-test-plan-r07-planning-findings-r03.md` | PASS |

## 8. 未验证范围与限制

- 未运行 Go、PostgreSQL、Unix 持久卷、部署、Host 或 E2E。
- 未读取或输出 `HAN_TEST_POSTGRES_DSN`。
- 未验证尚不存在的 Article file-lock/worker 文件实现质量。
- 未修改或复审 Tasks/Plan 当前内容；本报告无权关闭其 finding。
- 未重开 r07 未变且与原 planning F001/F002 无关的测试规划范围。
- 未验证真实 `flock`、file/directory fsync、崩溃、磁盘故障、跨进程竞争、容量或 production readiness。

## 9. Gate 与交回条件

### Test-plan Gate：FAIL

r07 当前**不可作为关闭型输入交 Task-Planner**。重新交回前须：

1. 补齐受支持 Unix 与 unsupported 两条 build-tag 分支的明确 compile-only 矩阵，关闭 `ARTICLE-S4.6-PLAN-R01-F001` 的剩余测试规划缺口；
2. 统一 §6 五条 A-WIRE 覆盖映射的 S4.6 登记状态，关闭 `AM-TP-R07-R03-F001`；
3. 补齐 r06 review PASS 的 source 追溯，关闭 `AM-TP-R07-R03-F002`；
4. 固定新 revision/hash/blob，由 Rawls 对上述项目及紧邻回归定点复审；
5. PASS 后才可由 Task-Planner 把 13 文件、双平台 compile-only Gate 和 S4.6/S4.3 边界同步到既有唯一 S4.6/B6A-W；
6. Task-Planner 不得自行关闭 planning review finding；Tasks/Plan 同步后仍须回原 planning Reviewer 复审；
7. 不得新增第二 wiring task、改变直接 requires/B6A-W/后继 DAG，或把真实 PostgreSQL、Unix、部署、E2E并入 S4.6 局部关闭 Gate。

本 Test-plan Gate FAIL 不回退 design r10，也不表示应修改 Tasks/Plan、代码或已关闭的历史 task。

## 10. 实际改动

- 仅新增本报告：`.spec/wip/article-management/review-test-plan-r07-planning-findings-r03.md`
- 未修改 test-plan、design、stories、Tasks、Plan、代码或既有报告。
- 未 stage、commit、push、切分支、clean、reset、checkout 或清理任何他人改动。
