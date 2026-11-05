# B7 三卡 R 对账 Planning Review R02

- 日期：2026-10-25
- Reviewer：Gauss（原独立 Reviewer，非 Task-Planner 作者）
- Task-Planner 作者：Erdos the 2nd
- objectType：非代码规划制品（Tasks/Plan，按 design/document review 维度）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r01.md`，SHA256 `1ED18D7DFF027DAE881B63ABABC90009A8B05566CD51A5745C6BB87FAB95D2BB`
- 固定复审对象：
  - `.add/project/knowledge/tasks/tasks.md`，SHA256 `E3BE2FC4DD7AE658BC20ACA47887198480CBCCE8F0AA650A84F400303F46806E`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md`，SHA256 `4F2CC8D15F6F79F15A78A98022C2D869ABB25A420DAC96F69D2975E6B3AA02B7`
- **Planning Review Gate：PASS**

## 1. 结论摘要

`ARTICLE-B7-PLAN-R01-F001` 已 **CLOSED**。Tasks 与 Plan 现在为三张 Story-level 卡各登记且仅登记一个现役四标签：

- S9.1：**需改（上游设计阻塞）**
- Article S4.2：**需改**
- Article S4.5：**待做**

三项均同时给出 desired design/story/test-plan 与 current code/制品状态的可定位证据；规划、Review、design/test-plan PASS、已知不合格候选实现以及不存在的目标制品均未再被称为“已建成”。本次修订未发现新增 HIGH 或 MEDIUM finding。

本 PASS 只关闭 Tasks/Plan 的规划对账 finding，不关闭 S9.1 或 S4.2 的代码 finding，也不构成实施 START。S9.1 继续 `blocked-for-design-revision`；S4.2/S4.5 仍须等待 Lead 明确 START。

## 2. 两轮检查范围

### Round 1：固定 Tasks/Plan 与原 finding 定点复审

逐项检查：

- R01 F001 的原始触发、最小修复与关闭条件；
- `tasks.md:860,1318,1399` 的三张 Story-level R gap-map；
- `Plan:697,699,701` 的对应标签、desired/current 证据与暂停状态；
- 三态统计、六项 active、重复 task、checkbox、日期、requires、DAG 与批次范围；
- S4.2 精确 19 文件、唯一前向 `0009`、命令和零测试防假绿；
- S4.5 精确四文件、精确测试符号、普通/race/scoped vet 与零测试防假绿。

### Round 2：上游合同、真实代码/目录与修订新增风险

交叉核对：

- identity S09 story/design 与 S9.1 Code Review R01 的四项 OPEN finding；
- article Story 4、design r14、test-plan r12 与 S4.2 Code Review R01；
- S9.1/S4.2 当前代码定位；
- S4.2 production source/inventory/PostgreSQL integration/`0009` 与 S4.5 四文件的真实存在性；
- 现有 recovery fixture 测试符号；
- Tasks/Plan 是否错误外推真实 PostgreSQL、Unix 持久卷、旧备份恢复、灾备、Host、部署、E2E 或 production readiness。

## 3. 原 finding 复审

### MEDIUM

#### `ARTICLE-B7-PLAN-R01-F001` — 三卡 gap-map 误用“已建成”并缺少 Story-level 单标签

- **状态**：CLOSED
- **原触发**：R01 中三卡同时混写多个分类，并将规划/Review 证据、已知不合格候选代码或尚不存在制品的合同标作“已建成”，无法可靠区分当前实现事实与 desired contract。
- **修订位置与证据**：
  1. `tasks.md:860` 将 S9.1 唯一标为 **需改（上游设计阻塞）**。该条引用 identity stories/design 的 desired authority 合同和 `internal/identity/deletion.go` 的 current 缺口，明确 R01 F001～F004 仍 OPEN，并明确候选文件、规划或 Review 不构成符合现役设计的能力。
  2. `tasks.md:1318` 将 Article S4.2 唯一标为 **需改**。该条分别引用 article stories/design/test-plan 与四个 current code 定位，明确 production source/inventory/wiring、PostgreSQL integration test 和 `0009` 不存在，Code Review R01 仍 FAIL；design/test-plan PASS 仅作为返工依据。
  3. `tasks.md:1399` 将 Article S4.5 唯一标为 **待做**。四个目标文件均明确为不存在；`test/integration/recovery/fixtures_test.go:23` 的 `TestRecoveryFixtures` 被明确排除为目标 `TestArticleRecoveryFixtures` 的替代品；design/test-plan 与 Review PASS 仅证明合同稳定。
  4. `Plan:697,699,701` 与上述三项标签、desired/current 定位、OPEN/FAIL 状态和制品存在性逐项一致。
- **真实目录复核**：
  - S4.2 新增 source/reader/integration 五个目标 Go 文件及 `migrations/0009_article_replay_attempt.sql` 均不存在；
  - S4.5 四个目标文件均不存在；
  - migration 目录当前只有 `0001`～`0008`；
  - 现有 recovery fixture 只有 `TestRecoveryFixtures`。
- **关闭判断**：每张卡已形成“一个 Story-level 标签 + desired 依据 + current 证据”的可审计表达；没有保留误导性的现役“已建成”声明，满足 R01 的最小修复和验证条件。

## 4. 新 finding 检查

- HIGH：0
- MEDIUM：0

未发现本次修订引入新的阻断问题。尤其未发现：

- 将 design/test-plan/Review PASS 外推为代码、测试或 production PASS；
- 将 S9.1 的不合格五文件候选或 S4.2 原五文件候选改称为合格实现；
- 将 S4.5 不存在的四文件改称为已有制品；
- 通过第 20 个 S4.2 文件、S4.5 第五文件、第二 authority/root/provider 或修改 `0001`～`0008` 扩权；
- 弱化暂停、依赖、环境或后继 Gate。

## 5. 守护项复核

### 任务状态与批次

- 程序化统计仍为 57 个任务：29 completed / 6 active / 22 todo；重复 task 为 0。
- 六项 active 精确为：
  - `identity-auth-S3.3`
  - `identity-auth-S3.5`
  - `identity-auth-S9.1`
  - `article-management-S4.2`
  - `article-management-S4.3`
  - `article-management-S4.5`
- `Plan:906` 与 Tasks 一致；S10.5、S3.6、B8 及后继部署/E2E 未启动。
- S9.1 保持 `implementation paused / blocked-for-design-revision`；现有五文件内不能安全解决 authority locking 与 maintenance/readiness Gate。
- S4.2 保持 `implementation paused pending Planning Review`；S4.5 保持 `planning/implementation paused pending Planning Review`。本报告 PASS 后仍缺 Lead 明确 START。

### Article S4.2

- Tasks 与 Plan 均保持精确 19 文件；`cmd/han-server/main.go` / `main_test.go` 与已关闭 S4.6 的唯一 composition owner 串行，不重开 S4.6。
- `migrations/0009_article_replay_attempt.sql` 是唯一规划新增 migration，仅允许前向增加 generation/token 与一致性约束；禁止修改 `0001`～`0008`。
- 四个精确测试符号/build tag 前置 Gate、article+server 普通/race、真实 PostgreSQL `TestDeletionReplayPostgres` 普通/race、vet/build/verify/Manifest/diff-check 均保持登记。
- A-REC-04～07 仍要求多连接/多事务、三类 Begin commit unknown、generation/token CAS、UTC 微秒、旧 Fail/Ready 晚到、TOCTOU/freshness 与 current/evidence 原子性。
- 第 2～5 项出现 `[no tests to run]` 必须 FAIL；memory、fixture、compile-only 或 `t.Skip` 不得替代真实 PostgreSQL 证据。

### Article S4.5

- Tasks 与 Plan 均保持精确四文件，且四文件当前均不存在。
- 前置 Gate 精确检查 integration build tag 与 `TestArticleRecoveryFixtures`；唯一专用普通命令、race、scoped vet、Manifest 和零测试 FAIL 规则保持不变。
- Ready20 canonical/evidence digest 重算与 inventory 固定合成引用 `sha256:bbbb` 的逐字跨引用检查保持分离；未定义 production inventory digest 算法，未引入第五文件或 production provider。

### 依赖、DAG 与未外推边界

- 三卡 requires、DAG、checkbox、日期、批次范围与文件所有权未因标签修复而改变。
- S4.3 的受支持 Unix 独立持久卷阻塞、S3.3/S3.5 的执行阻塞均未弱化。
- Tasks/Plan 未宣称真实 PostgreSQL、真实旧备份恢复、灾备、外部 SMTP、Browser、Host、部署、E2E、真实 provider、容量或 production readiness 已通过。

## 6. 实际读取

完整读取：

- `.add/project/knowledge/tasks/tasks.md`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`
- `.spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r01.md`

同一连续评审上下文复用此前已完整读取且版本未变的：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/task-execution-workflow.md`
- `.codex/skills/add-task-decompose/SKILL.md`
- `.codex/skills/add-batch-plan/SKILL.md`

定向读取/核对：

- identity-auth 的 `stories.md`、`design.md` 与 `.spec/wip/identity-auth/review-code-s9.1-r01.md`
- article-management 的 `stories.md`、design r14、test-plan r12 与 `.spec/wip/article-management/review-code-s4.2-r01.md`
- `internal/identity/deletion.go`
- `internal/article/replay_evidence.go`
- `internal/article/deletion_replay.go`
- `internal/article/replay_status_v1.go`
- `internal/article/deletion_replay_test.go`
- `internal/recovery/replay_status_v1.go`
- `test/integration/recovery/fixtures_test.go`
- S4.2/S4.5 目标路径存在性与 `migrations/` 当前文件列表

## 7. 验证与未验证边界

- 开始固定对象 SHA256：
  - Tasks：`E3BE2FC4DD7AE658BC20ACA47887198480CBCCE8F0AA650A84F400303F46806E`
  - Plan：`4F2CC8D15F6F79F15A78A98022C2D869ABB25A420DAC96F69D2975E6B3AA02B7`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 开始 staged：空。
- 结束固定对象 SHA256：
  - Tasks：`E3BE2FC4DD7AE658BC20ACA47887198480CBCCE8F0AA650A84F400303F46806E`
  - Plan：`4F2CC8D15F6F79F15A78A98022C2D869ABB25A420DAC96F69D2975E6B3AA02B7`
- 开始/结束哈希一致，固定对象未漂移。
- `git diff --check -- .spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r02.md`：PASS。
- 结束 staged：空。
- 按授权未运行 Go 测试、PostgreSQL、网络、部署或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`；未触碰 `han-server.exe`。
- 本报告只评审固定 Tasks/Plan 的规划表达及必要直接证据，不评价尚未执行的实现、migration、测试命令或真实环境结果。
- 共享工作树含大量他人未提交成果；本报告不归属、不清理也不评价无关改动。

## 8. 最终 Gate

`ARTICLE-B7-PLAN-R01-F001`：**CLOSED**

新 HIGH/MEDIUM finding：**无**

**Planning Review Gate：PASS**

返回点不变：S9.1 先完成上游设计修订与后续 R 对账；S4.2/S4.5 虽已通过本次 Planning Review，仍须 Lead 明确 START 后方可实施。
