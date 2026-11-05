# B7 三卡 R 对账 Planning Review R01

- 日期：2026-10-25
- Reviewer：Gauss（独立非作者 Reviewer）
- Task-Planner 作者：Erdos the 2nd
- objectType：planning
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定对象：
  - `.add/project/knowledge/tasks/tasks.md`：SHA256 `6F0A3C126564655B29FA643691C203178F988CCDC8190BB4DCEE4EE3A5A2596B`
  - `.add/project/knowledge/plans/ir-01-content-community-batches.md`：SHA256 `0C11071F00DC1C7143BC5343DED94FCD158E425BFC8966FDCA04A9A13C0D7630`
- **Planning Review Gate：CHANGES REQUIRED**

## 1. 结论摘要

发现 **1 项 MEDIUM**。Tasks/Plan 对三卡状态、写集、依赖、命令、停止条件和未验证边界的主体登记一致；S9.1 没有获得可执行 START，S4.2/S4.5 也正确保持在 Planning Review 前暂停。

阻断 Gate 的问题是三卡的“R 对账四标签”不是符合现役 `add-task-decompose` 合同的 Story 级单标签 gap-map，并把候选文件、Review/设计证据或尚不存在制品的规划合同写成“已建成”。这会使后续实施者无法可靠区分“当前代码已初判符合设计”和“只有设计/规划依据或已知不符合的候选实现”。

该 finding 只需返回 Task-Planner 修订 Tasks/Plan；未发现需要修改 architecture、ADR、boundary、design 或 test-plan 的新方向缺口。

## 2. 两轮检查范围

### Round 1：固定 Tasks/Plan 直接检查

逐项核对：

- 57 / 29 completed / 6 active / 22 todo，重复 task 为 0；
- active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`identity-auth-S9.1`、`article-management-S4.2`、`article-management-S4.3`、`article-management-S4.5`；
- S9.1、Article S4.2、S4.5 卡片的 checkbox、日期、requires、写集、验收命令、阻塞和关闭条件；
- Plan 的 B7 Context Pack、DAG、文件所有权、批次 Gate、整体接纳和历史记录；
- `identity-auth-S10.5`、`identity-auth-S3.6`、B8 未启动，`article-management-S4.6` 与 B6A-W 保持 CLOSED。

### Round 2：上游合同与真实目录追踪

交叉核对：

- S9.1 Code Review R01 的四项 OPEN finding及“五文件内无法安全完成 authority locking 与 maintenance/readiness Gate”的结论；
- article design r14、Design Review R02 PASS、test-plan r12、Test-plan Review R02 PASS；
- S4.2 Code Review R01 的 3 HIGH + 2 MEDIUM；
- `add-task-decompose` 的 R 模式 Story 级 gap-map、四标签定义和“已建成＝代码在且初判符合新设计”要求；
- 当前 migration 只有 `0001`～`0008`，建议 `0009` 尚不存在；
- S4.2 的 production source/reader/PostgreSQL integration 新文件尚不存在，原五个候选文件存在；
- S4.5 四个登记文件均不存在；仓内只有现有 `TestRecoveryFixtures`，没有 `TestArticleRecoveryFixtures`；
- 三卡与其他 active lane 的登记写集无交叉；共享 `cmd/han-server` 和已关闭 S4.6 成果已有串行/重新认领约束。

## 3. Findings

### MEDIUM

#### `ARTICLE-B7-PLAN-R01-F001` — 三卡 gap-map 把规划/Review 证据或已知不符合的候选实现误标为“已建成”

- **状态**：OPEN
- **位置**：
  - `tasks.md:860`：S9.1 “已建成＝五文件候选与 R01 finding 证据”
  - `tasks.md:1318`：S4.2 “已建成＝五个候选文件、strict memory 雏形及 R01/design/test-plan 证据”
  - `tasks.md:1399`：S4.5 “已建成＝design r14/test-plan r12 与两份独立 PASS 所固定的测试专用制品合同”
  - 对照 `.codex/skills/add-task-decompose/SKILL.md:51-56,145-159,163`
- **触发场景 / 证据**：
  1. R 模式要求每个 Story 取得一个标签和当前可定位证据；“已建成”只适用于代码/制品已经存在且初判符合当前设计。
  2. S9.1 五文件虽存在，但 R01 有 3 MAJOR + 1 MINOR OPEN，且 authority locking/Gate 还需上游设计修订，不能把“候选文件与 finding 证据”作为已建成能力。
  3. S4.2 原五文件存在，但 Code Review R01 的 3 HIGH + 2 MEDIUM 仍 OPEN；design/test-plan PASS 是返工依据，不是已建成代码。
  4. S4.5 四个目标文件全部不存在；design/test-plan 与 Review PASS 只能证明合同稳定，不能证明制品已建成。
  5. 当前三条内联文本同时列出四个类别，而不是 Story 级“一个标签 + 证据”的 gap-map，且没有把可保留的局部已建成切片与需改/待做切片确定性分开。
- **影响**：
  - 后续 Step 2～5 可能错误跳过应实施或应返工的能力；
  - “设计稳定”“存在候选代码”和“代码已初判符合设计”三个状态被混写，削弱 Tasks SSOT 的可审计性；
  - S4.5 尤其可能被误读为已有可验证制品，虽然四文件实际为 0。
  - 现有 paused/blocked Gate 降低了立即误实施风险，因此定为 MEDIUM，而非 HIGH。
- **最小修复**：
  1. 为 S9.1、S4.2、S4.5 各形成符合 R 模式的 Story 级单标签 gap-map，并给出当前代码/制品定位证据；
  2. 设计、test-plan 与 Review PASS 仅列为规划来源，不列入“已建成”；
  3. S4.5 在四文件均不存在时应归入“待做”，已建成能力为 0；
  4. S9.1/S4.2 若保留局部“已建成”切片，必须逐项给出当前代码位置并证明其初判符合现役设计；其余 OPEN finding 对应能力归入“需改/待做”，S9.1 继续标注上游设计阻塞；
  5. 不改 checkbox、日期、requires、DAG、批次范围或已关闭任务状态。
- **验证条件**：
  - 三卡均能得到唯一、可复核的 Story 级分类；
  - 任一“已建成”项都能追到实际存在且初判符合当前设计的代码/制品；
  - S4.5 不再以设计/Review PASS 充当制品存在证据；
  - Tasks 与 Plan 对修订后分类、暂停状态和后续 Gate 保持一致，再由非作者复审。

## 4. 已确认一致的关键范围

### S9.1

- 保持 `[-]`、`implementation paused / blocked-for-design-revision`；
- 明确五文件内无法安全实现 RecoveryRun authority locking 与 maintenance/readiness Gate；
- 不提供返工 START，不启动 S10.5、S3.6、B8或后继部署/E2E；
- direct requires 当前均为 CLOSED，未借此弱化设计阻塞；
- 没有扩写 article、schema/migration或建立第二 authority。

### Article S4.2

- Tasks 与 design/test-plan 的写集逐项一致，精确 19 文件；
- `migrations/0009_article_replay_attempt.sql` 是唯一新增 migration，且只允许前向增加 generation/token 与一致性约束；禁止修改 `0001`～`0008`；
- `internal/recovery/replay_status_v1.go` 与 identity consumer 保持只读，禁止第二 ReplayStatus authority；
- `cmd/han-server/main.go` / `main_test.go` 与已关闭 S4.6 的唯一 composition owner 串行，S4.6 不重开；
- 四个精确测试符号/tag 前置 Gate、普通/race、真实 PostgreSQL普通/race、vet/build/Manifest/diff-check均已登记；
- A-REC-04～07 明确要求多连接/多事务、三类 Begin commit unknown、generation/token CAS、旧 Fail/Ready 晚到、UTC 微秒、TOCTOU/freshness和 current/evidence 原子性；
- 第 2～5 项出现 `[no tests to run]` 必须 FAIL；memory、fixture、compile-only和 `t.Skip` 不得替代；
- design/test-plan PASS 未被外推为关闭 S4.2 Code Review R01。

### Article S4.5

- 写集精确为四个文件，真实目录核对四者均不存在；
- `TestArticleRecoveryFixtures`、首行 integration tag、精确锚定普通/race、scoped vet、Manifest和零测试 FAIL 均已登记；
- Ready20 canonical/evidence digest 重算与 inventory 固定合成引用 `sha256:bbbb` 的逐字检查已分离；
- 禁止定义或重算 production inventory digest 算法，禁止第五文件、生产 provider、migration或公共合同扩权；
- 关闭边界未外推真实旧备份恢复、灾备、Host、部署、E2E或 production readiness。

### Tasks/Plan 与批次

- 三态、active 集合、checkbox、日期、requires、DAG和 B7 范围一致；
- 重复 task 为 0；
- S4.3 的真实 Unix 独立持久卷阻塞、S3.3/S3.5 的执行阻塞均未弱化；
- 未把真实 PostgreSQL、旧备份恢复、真实 provider、灾备、Host、部署/E2E、容量或 production readiness 写成已通过。

## 5. 验证

- 开始固定对象 SHA256：
  - Tasks：`6F0A3C126564655B29FA643691C203178F988CCDC8190BB4DCEE4EE3A5A2596B`
  - Plan：`0C11071F00DC1C7143BC5343DED94FCD158E425BFC8966FDCA04A9A13C0D7630`
- 结束固定对象 SHA256：
  - Tasks：`6F0A3C126564655B29FA643691C203178F988CCDC8190BB4DCEE4EE3A5A2596B`
  - Plan：`0C11071F00DC1C7143BC5343DED94FCD158E425BFC8966FDCA04A9A13C0D7630`
- 开始/结束哈希一致，未发生固定对象漂移。
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- `git diff --check -- .spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r01.md`：PASS；另以 no-index 检查未发现 whitespace error。
- staged：开始和结束均为空。
- 按限制未运行 Go 测试、PostgreSQL、网络、部署或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- `han-server.exe` 未触碰。

## 6. 实际读取

完整读取：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/task-execution-workflow.md`
- `.codex/skills/add-task-decompose/SKILL.md`
- `.codex/skills/add-batch-plan/SKILL.md`
- `.codex/guidance/abstraction-principles.md`
- 两个固定 Tasks/Plan 对象
- `.spec/wip/identity-auth/review-code-s9.1-r01.md`
- `.spec/wip/article-management/review-code-s4.2-r01.md`

定向读取：

- article design r14、test-plan r12；
- design review R02、test-plan review R02；
- 三卡及相关 active/closed task、B7 Context Pack、DAG、所有权、批次 Gate和整体接纳；
- S4.2/S4.5 真实目录、migration 列表、现有 recovery fixture/test 符号及相关文件状态。

## 7. 未验证边界

- 未执行任何计划中的测试命令，故不评价代码、migration或测试实际 PASS；
- 未连接真实 PostgreSQL，未验证 SQL、锁、CAS、回滚、时间精度或 commit outcome unknown；
- 未验证受支持 Unix 独立持久卷、真实旧备份、Journal/keyring/inventory、真实 provider、Host、部署或 E2E；
- 共享工作树含大量既有未提交/未跟踪成果，本报告只评价固定 Tasks/Plan 与上述有界消费者，不归属其他改动；
- S9.1 的上游设计阻塞是已知返回点，本报告不替作者选择 architecture/ADR/boundary 方向。

## 8. 最终 Gate

严重度统计：

- HIGH：0
- MEDIUM：1 OPEN
- LOW：0

**Planning Review Gate：CHANGES REQUIRED**

返回 Task-Planner 仅修复 `ARTICLE-B7-PLAN-R01-F001` 的 R gap-map 语义和证据表达；S9.1 继续 blocked-for-design-revision，S4.2/S4.5 在复审 PASS 与 Lead 明确 START 前继续暂停。
