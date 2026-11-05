# identity-auth S10.2 Tasks/Plan 增量 Planning Review R03

- 日期：2026-10-24
- Reviewer：Gibbs，原非作者独立 Reviewer
- 作者 / Task-Planner：Confucius
- objectType：design/planning 复合对象增量复审
- 工作区 / 分支 / HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 目标：核定 design r40 / stories r16 / test-plan r18 对既有 S10.1～S10.4、S11.1 的任务与批次影响同步是否完整、最小、无环且保持状态和专项边界

本报告只评价固定 Tasks/Plan 的本次规划增量。它不关闭代码 `IDENTITY-S10.2-REV-R01-F003` 或 `IDENTITY-S10.1-REV-R01-F002`，不关闭 S10.2，不恢复 S10.1，不授权或启动 S10.3、S10.4、S11.1，也不证明任何实现或测试已经通过。

## 1. 固定对象与完整性

| 对象 | 用途 | 固定 SHA256 | 开始核对 | 结束核对 |
|---|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | Tasks SSOT | `2FA2BCCE0A3755901AA91E6C0164A8923FA13AF1A57263DC4D3081C309CD8AB5` | MATCH | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | 批次 Plan | `D181BAFB35D6F1B16A3E11548E6C0F9DDD22AE2815BCA962F52298F040E771EC` | MATCH | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | design r40 | `B8D2171DEF5DF3E0B47B521C9898E9B19F77EFC44388A0B461ED34B774416336` | MATCH | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | stories r16 | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` | MATCH | MATCH |
| `.spec/wip/identity-auth/review-design-r40-s10.2-r01.md` | Design Review PASS | `EB1AB24C659CB8281FD4FC2058470C88ACA7FACFC33D1665900C091CAE9A46FB` | MATCH | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | test-plan r18 / planned-not-executed | `7A782D373684FD65114BBDA297FF4FF22A237F1868F5B1534E5FECEA2C9F88D4` | MATCH | MATCH |
| `.spec/wip/identity-auth/review-test-plan-r18-s10.2-r02.md` | Test-plan Review PASS | `3A241B10EFB64EFA7A09E452F20F22CB84051AAB7D3503E511C5298451AF794C` | MATCH | MATCH |

补充核对：

- architecture r27：`A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB`
- boundary r13：`0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3`
- 历史 Planning r02：`210AAED75029CB76DF30DA30F0D4DB165B555BF97D6F52CAA4CED63E17DC0118`；只作旧规划基线，不外推为本次增量 PASS。

目标报告写前不存在；分支、HEAD 匹配；暂存区为空。共享工作树中的 Tasks/Plan 修改属于固定被审现场，本 Reviewer 未修改它们。

## 2. 两轮实际检查

### Round 1：固定对象直接检查

1. 完整执行 Reviewer Role、`add-code-review` design 分支、评审清单、抽象原则，并按规划对象补读 `add-task-decompose`、`add-batch-plan`、`add-delivery-workflow` 和任务三态合同。
2. 检查 Tasks 的 Story-level gap-map、二级能力矩阵、S10.1～S10.4/S11.1 五张卡及状态、TDD、命令、写集、依赖、关闭条件和专项边界。
3. 检查 Plan 的批次外层表、B6C Context Pack、依赖图、文件所有权、批次 Gate、整体接纳和风险段落。
4. 核对本次增量没有新增 task，没有改 checkbox、日期、requires、DAG、owner 或写集。

直接检查结果：

- `tasks.md:94-122` 继续保持 S10=`需改`、S11=`待做` 的唯一 Story-level 结论；11 行 `1/4/5/1` 明确只是二级能力 traceability，不冒充 Story 分布。
- `tasks.md:896-922` 将 r40/r18 新增影响完整落入既有 S10.2 八文件、TDD 和关闭证据：A 后正常改密、A→B current permanent、A→B→temporary C、same-key 原终态 replay、new-key `legal_history_stale`、`CP-01/T01～T05/P01～P04/N01/F01`、合法路径零候选/能力/hold 写入及瞬时故障零新 hold。
- `tasks.md:924-995` 只让 S10.3、S10.1、S10.4、S11.1 消费同一 classifier 与状态语义，没有把 S10.2 的返工吞并进后继卡，也没有把真实环境专项并入开发卡关闭条件。
- `plan.md:58-76,647-659` 同步相同范围、状态和顺序，明确新增 task=0，并把 task-level close、B6C-3 batch Gate、代码 finding 复审和 B6D 启动条件分层。

### Round 2：上游、真实所有权与结构深审

1. 沿 design r40、stories r16、Design R01、test-plan r18 与 Test-plan R02 核对 current capability root / immutable legal history、合法 root replacement、真实 drift 和瞬时依赖故障的分类及写入后置。
2. 有界核对 architecture r27、boundary r13 与历史 Planning r02，确认公共 REST/错误/权限、ADR/boundary、专项任务和 B6D Gate 没有被规划增量改写。
3. 用静态脚本解析 Tasks/Plan：
   - checkbox 三态与 active 集合；
   - task ID 唯一性；
   - 外层批次唯一安排；
   - requires 未知目标与有向环；
   - S10.2/S10.3/S10.1/S10.4/S11.1 写集交叉。
4. 核对专项与 E2E 分界、显式运行输入及停止条件；未从当前代码实现反向降低设计或测试预期。

结构检查结果：

- 56 total / 22 completed / 4 active / 30 todo。
- active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`identity-auth-S10.1`、`identity-auth-S10.2`。
- 56 个 task ID 无重复。
- Plan 外层表共安排 56 次、56 个唯一 task；无遗漏、额外或重复。
- requires 图为 56 节点、237 条边；无未知目标、无环。
- S10.2/S10.3/S10.1/S10.4 写集两两互斥。
- S10.2 写集精确为固定八文件，无第九文件。
- S11.1 只与 S10.1 共享 `internal/identity/handler.go`，并保持 B6D 串行。
- `git diff --check -- <Tasks> <Plan>`：PASS。

## 3. 规划正确性结论

### 3.1 gap-map 与任务最小性

PASS。

- S10/S11 的 Story-level 标签和当前设计、代码、原 task 证据一致。
- 本轮只复用 S10.1～S10.4/S11.1 五张既有卡，新增 task=0；没有遗漏需要独立 owner 的增量，也没有把同一能力复制成新 task。
- 已关闭 S3.2/F005、S5.2、S8.1 只作为回归基线，没有重开其写集或状态。

### 3.2 S10.2 TDD、命令与关闭证据

PASS。

- S10.2 在唯一八文件中拥有 r40/r18 的 history/current classifier 返工及全部 CP 矩阵。
- 合法 A/B/C、same-key/new-key stale、真实逐字段 drift、去敏 Audit、零写入和 query/User lock/commit 瞬时故障均有可核证的 task-level 失败证据与关闭条件。
- 专用普通/race、scoped diff-check、Manifest 七项、原代码 Reviewer 关闭 F003 和 Lead §10 核证均被要求；旧局部 PASS 不可替代当前增量证据。

### 3.3 依赖、批次与所有权

PASS。

- 单向顺序保持 `S10.2 → S10.3 → S10.1 implementation slice → S10.4`。
- S10.1/S10.4 无相互 requires；两卡分别独立 task-level close 后才进入 B6C-3 integrated Review、Hubble 对代码 F002 的定点复审和 Lead batch 接纳。
- batch Gate 失败只阻止 B6D，不反向伪造或抹除已经取得的单卡局部 PASS。
- B6D/S11.1 仍须同时满足四卡 task-level 均关闭、B6C-3 batch Gate PASS、代码 F002 CLOSED。

### 3.4 专项、E2E 与运行输入

PASS。

- S1.5 继续承担公共合同，S1.6 承担真实 PostgreSQL 18、多连接、lease、commit-ack-loss、migration 混跑与回退阻断。
- S5.3 只承担 local real-protocol SMTP；S3.4 继续承担 approved-provider、external-receipt 和真实 Host E2E；S2.2 承担性能容量。
- lease duration、retry backoff、自动重试上限、批量领取上限和连接池预算仍为实施前显式输入；缺值时 writer/worker 不开放，规划没有硬编码默认。

## 4. Findings

新增 finding：无。

- CRITICAL：0
- HIGH：0
- MEDIUM：0
- LOW：0

未发现本次规划增量引入任务遗漏、所有权冲突、隐藏依赖环、错误关闭条件、专项/E2E 吞并、状态漂移或硬编码运行参数。

## 5. 待决定与非阻断建议

### 待决定

无。固定上游已经决定设计与测试语义；本轮没有发现需要新增 ADR、boundary、公共合同、task 或批次的方向性问题。

### 非阻断建议

无新增建议。恢复 S10.2 返工时直接按卡内列出的完整 CP 子例执行，避免用 `I-PRD-05.CP-*` 简写替代逐例证据即可；现有规划已经明确列出这些子例，不需要再改 Tasks/Plan。

## 6. 未验证风险与限制

1. 本轮是静态 Planning Review；未运行 Go、Manifest 七项、PostgreSQL、SMTP、部署、Host、浏览器或 E2E。
2. 未验证八文件实现是否已满足 r40/r18；代码 F003 仍须实现返工和原代码 Reviewer 定点复审。
3. 未验证真实事务锁、commit actual state、多连接、worker lease 竞争、migration 混跑或不可安全回退；这些仍归既定专项。
4. 未验证 local real-protocol SMTP、approved provider、外部收件、性能、容量或 production readiness。
5. 未读取、输出或使用敏感连接值。
6. 共享工作树其他合法未提交/未跟踪内容不属于本报告结论。

## 7. 状态边界

- `IDENTITY-S10.2-REV-R01-F003`：代码层 OPEN。
- `IDENTITY-S10.1-REV-R01-F002`：代码层 Major OPEN。
- S10.1：active，implementation stopped。
- S10.2：active，implementation paused / must not close。
- S10.3、S10.4、S11.1：todo / not started。
- 本 Planning PASS 不构成恢复实施、关闭任务、启动后继卡、执行测试或发布的授权。

## 8. Planning Review Gate

**Planning Review Gate: PASS**

理由：固定 Tasks/Plan 已把 r40/r18 的增量准确落入既有最小任务集合和唯一批次顺序；S10.2 的八文件、TDD、命令、关闭证据与后继消费者映射完整，状态、依赖、写集、专项/E2E 和显式运行输入均保持一致，静态结构检查无遗漏、重复、未知依赖或环。

该 PASS 仅供 Lead 现场复核后决定是否恢复 S10.2 返工；不关闭任何代码 finding，不关闭 S10.2，不恢复 S10.1，不启动 S10.3/S10.4/S11.1。

## 9. 实际命令

执行了：

- `git branch --show-current`
- `git rev-parse HEAD`
- `git diff --cached --name-only`
- `Test-Path`：确认目标报告写前不存在
- `Get-FileHash -Algorithm SHA256`：固定对象及上游开始/结束核对
- `Get-Content -Encoding utf8`、`rg -n --encoding utf-8`：固定对象与有界上游检查
- Node 静态解析：
  - task checkbox、三态、active 集合及重复 ID；
  - Plan 外层批次 task 引用；
  - requires 图、未知目标和 DFS 环检测；
  - B6C/S11.1 文件清单与交叉。
- `git diff --check -- .add/project/knowledge/tasks/tasks.md .add/project/knowledge/plans/ir-01-content-community-batches.md`
- `git diff --check -- .spec/wip/identity-auth/review-tasks-plan-s10.2-r03.md`

未运行实现、数据库、邮件、部署或 E2E 命令。

## 10. 唯一写入

仅新增 `.spec/wip/identity-auth/review-tasks-plan-s10.2-r03.md`。

未修改 Tasks、Plan、设计、stories、test-plan、代码、migration、Manifest、checkpoint、任务状态或其他报告；未 stage、commit、push、切分支、移动或清理。
