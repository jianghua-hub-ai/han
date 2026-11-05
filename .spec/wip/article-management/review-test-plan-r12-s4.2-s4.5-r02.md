# article-management test-plan r12 / S4.2、S4.5 Test-plan Review R02

- Reviewer：原独立 Reviewer Gauss
- 被审作者：Test-Engineer Ohm
- 评审日期：2026-10-25
- objectType：test-plan
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定对象：`.add/project/knowledge/design-workspace/article-management/test-plan.md`
- revision：`article-management-test-plan-ir01-r12`
- status：`planned-not-executed`
- 固定 SHA256：`85D8A7CC6ABF01A88B24726EEE3957691AA0B8A7B6235FEB8A31677D77336DDF`
- 原 R01：`.spec/wip/article-management/review-test-plan-r11-s4.2-s4.5-r01.md`
- 原 R01 SHA256：`52DF371E3E07EC88D6DA8046B1ED19A519D5FCBB882F30B3336C10B1D0A1910B`
- **Test-plan Review Gate：PASS**

## 1. 结论

两轮复审确认：

- `ARTICLE-TP-R11-S42-S45-R01-F001`：**CLOSED**
- `ARTICLE-TP-R11-S42-S45-R01-F002`：**CLOSED**
- 新增 finding：**0**
- 严重度统计：**0 CRITICAL、0 HIGH、0 MEDIUM、0 LOW**

r12 已把所有非历史 S4.6/B6A-W 当前态统一为 `article-management-S4.6 [x]`、`B6A-W CLOSED`，并继续明确真实 PostgreSQL、受支持 Unix 持久卷、Host、部署/E2E、容量与 production readiness 未验证。S4.5 也已确定性地区分 Ready20 canonical/evidence digest 重算与 inventory 固定合成引用 `sha256:bbbb` 的逐字跨引用检查；`digest_mismatch` 不再暗示存在未定义的 inventory 摘要算法。

本 PASS 只表示固定 r12 的测试规划合同通过独立复审，不表示 Tasks/Plan 已完成 r14 对账，不表示 S4.2/S4.5 已实施或测试通过，也不外推真实恢复、部署、E2E 或 production readiness。

## 2. 两轮复审范围

### Round 1：原 finding 定点复审

- 完整读取固定 test-plan r12 与原 R01；
- 检索全部 S4.6/B6A-W 当前态及 `todo/open/未认领/未启动` 表述，区分当前合同与 §9.1 历史记录；
- 逐项核 S4.5 的 Ready20 canonical/evidence digest、inventory 固定引用、`digest_mismatch`、四文件所有权和停止条件；
- 核 frontmatter 与正文继续保持 `planned-not-executed`，且未把作者修订写成独立 Review 或测试执行 PASS。

### Round 2：SSOT、设计、公共消费者与守护合同交叉核对

- 对照 Tasks/Plan 的 S4.6 `[x]`、B6A-W CLOSED 及真实环境未验证边界；
- 对照 design r14 的共享 Ready20 fixture、S4.2/S4.4/S4.5 分工与 production inventory 合同；
- 对照 `internal/recovery/replay_status_v1.go`：公共 DTO 仍为 Ready20/Unavailable6，且只对 Ready 前 16 字段 canonical 重算 `evidence_digest`；
- 守护复核 S4.2 精确 19 文件、前向 `0009`、A-REC-04～07、测试符号/命令/零测试 Gate，以及 S4.5 精确四文件、唯一测试符号和 `[no tests to run]` FAIL；
- 只读核当前目录：S4.2 新 source/reader/PostgreSQL integration/`0009` 与 S4.5 四文件仍不存在，符合“规划未执行”，不作为本轮新增 finding。

## 3. 原 finding 复审

### `ARTICLE-TP-R11-S42-S45-R01-F001` — MEDIUM — CLOSED

- **原问题**：r11 的非历史章节把已关闭的 S4.6/B6A-W 写成 todo/open/not-started，与 Tasks/Plan SSOT 冲突。
- **r12 证据**：
  - `test-plan.md:85,96,105,115` 均明确现役 S4.6 为 `[x]`、B6A-W 为 CLOSED，并限制该关闭只覆盖 task-level 局部闭环；
  - `test-plan.md:433-437` 的覆盖映射已逐行使用 S4.6 `[x]` / B6A-W CLOSED，同时保留真实 PostgreSQL、Unix 持久卷、Host、signal/exit、部署/E2E 与 production readiness 未验证；
  - `test-plan.md:621-633` 的当前任务建议把 S4.6 写为已关闭开发任务，并把 S4.2 与已关闭 composition owner 的关系写为串行；
  - `test-plan.md:641-711` 明确标为“r08 历史 handoff”，旧规划过程未再充当当前状态；
  - `test-plan.md:841,865,880` 继续保留 task-level closed 与真实环境 blocked 的分离；
  - Tasks `tasks.md:1369` 为 `[x] article-management-S4.6`，Plan `ir-01-content-community-batches.md:57` 为 `B6A-W ... CLOSED`。
- **触发复验**：直接消费覆盖映射、任务建议或 Gate 表，无需跳转历史章节纠正当前状态。
- **影响复验**：不存在第二套当前状态，不会因 test-plan 文本错误重开 S4.6 或重复分配 production wiring owner。
- **关闭理由**：原最小修复和验证条件均满足；历史过程与当前状态已清楚分层，且没有把局部关闭外推为真实环境通过。

### `ARTICLE-TP-R11-S42-S45-R01-F002` — MEDIUM — CLOSED

- **原问题**：r11 同时把 S4.5 `inventory_digest` 定义为固定合成引用，又要求取得未定义的“实际重算值”。
- **r12 证据**：
  - `test-plan.md:482` 明确 Ready20 中的 inventory digest 是业务设计共享 fixture 的 opaque string，保留 `sha256:bbbb`；
  - `test-plan.md:497,514` 明确 inventory 顶层值固定为 `sha256:bbbb`，只作跨引用，且不是 production inventory digest 算法声明；
  - `test-plan.md:551-557` 只对 Ready20 前 16 字段形成 canonical bytes，并用固定完整 SHA-256 校验 `evidence_digest`；
  - `test-plan.md:561-567` 要求 Ready `inventory_digest` 与 inventory 顶层固定引用逐字一致，并明确禁止为测试包装发明或重算 inventory digest；
  - `test-plan.md:574` 把 `digest_mismatch` 限定为上述固定跨引用不一致；
  - `test-plan.md:600,616,722,893` 在场景、自动化责任、四文件所有权和结论处均使用“Ready20 canonical/evidence digest 重算 + inventory 固定跨引用一致性”的同一表述；
  - 公共消费者 `internal/recovery/replay_status_v1.go:94-130,275-307` 只重算 Ready canonical 的 `evidence_digest`；Ready20/Unavailable6 字段保持不变，未引入 inventory 算法或 generation/token。
- **触发复验**：实现正例、`digest_mismatch` 负例与四文件 validator 时，可唯一确定应重算什么、只比较什么。
- **影响复验**：实现者无需猜测或发明 inventory payload/canonical/摘要算法，也无需增加第五文件、production provider、协议或公共 DTO 字段。
- **关闭理由**：原歧义已在所有相关责任点一致消除，且与 design r14 共享 fixture和公共 DTO 的实际行为一致。

## 4. 修订守护与新增风险检查

| 核对项 | 结果 | 证据与边界 |
|---|---|---|
| S4.2 精确 19 文件 | 保持 | `test-plan.md:774-796` 恰好列出 19 文件；第 20 文件触发停止 |
| `0009` / 既有 migration | 保持 | 只新增 `migrations/0009_article_replay_attempt.sql`，禁止修改 `0001`～`0008`；当前目录仍只有 `0001`～`0008` |
| A-REC-04～07 | 保持 | `292-295,453-464,798-833` 继续要求真实 PostgreSQL多连接/多事务、三类 Begin commit unknown、CAS、旧写晚到、UTC 微秒、TOCTOU/freshness 与唯一 authority |
| S4.2 命令与防假绿 | 保持 | 精确测试符号/build tag 前置检查；定向普通/race/真实 PostgreSQL命令均禁止 `[no tests to run]` |
| S4.5 精确四文件 | 保持 | `715-722` 恰好四文件；第五文件或 production 合同变化触发停止 |
| S4.5 专用测试 | 保持 | `TestArticleRecoveryFixtures`、精确锚定正则、integration 首行 tag、禁止 `t.Skip`，`[no tests to run]` 必须 FAIL |
| production 边界 | 保持 | S4.5 只拥有 synthetic fixture/runbook，不得成为 production provider 或真实恢复证据 |
| status / 执行边界 | 保持 | frontmatter 为 `planned-not-executed`；正文持续声明未执行测试、PostgreSQL、部署或 E2E |

未发现因 F001/F002 修订引入的新阻断风险。

## 5. 实际读取

### 完整读取

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.spec/wip/article-management/review-test-plan-r11-s4.2-s4.5-r01.md`
- `.add/project/knowledge/design-workspace/article-management/test-plan.md`
- `internal/recovery/replay_status_v1.go`

### 定向读取、哈希与检索

- `.add/project/knowledge/tasks/tasks.md`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`
- `.add/project/knowledge/design-workspace/article-management/design.md`
- `.add/project/knowledge/design/architecture.md`
- `test/integration/recovery/fixtures_test.go`
- `internal/article`、`test/integration/recovery`、`testdata/recovery` 的相关文件/符号存在性
- `migrations/0001`～`0008` 与建议 `0009` 的存在性

## 6. 验证与未验证边界

- 评审开始时固定对象 SHA256：`85D8A7CC6ABF01A88B24726EEE3957691AA0B8A7B6235FEB8A31677D77336DDF`。
- 原 R01 SHA256：`52DF371E3E07EC88D6DA8046B1ED19A519D5FCBB882F30B3336C10B1D0A1910B`。
- design r14 SHA256：`AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`。
- architecture r28 SHA256：`554686B77D0557F16A735F49F63626D922F41DAE8A8574B17980B7CB55CF1BCC`。
- 开始时分支/HEAD 为 `main@6da8abbceb0fa197364810a92a8fa2d97b38ddca`，staged 为空。
- 结束时固定对象 SHA256 仍为 `85D8A7CC6ABF01A88B24726EEE3957691AA0B8A7B6235FEB8A31677D77336DDF`，未发生对象漂移。
- `git diff --check -- .spec/wip/article-management/review-test-plan-r12-s4.2-s4.5-r02.md` 退出码为 0；结束时 staged 仍为空。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未运行测试、PostgreSQL、migration、网络、部署或 E2E；因此未验证未来实现的 SQL/锁/CAS/回滚、文件系统 no-follow/ACL/TOCTOU、真实恢复和命令实际输出。
- 未修改 test-plan、design、Tasks、Plan、代码、migration 或测试；共享工作树中的他人改动保持原状。

## 7. 最终 Gate

**PASS**

F001/F002 均已关闭，无新增 finding。该 test-plan 可作为后续 Task-Planner 对账 S4.2 精确 19 文件/`0009` 与 S4.5 精确四文件的稳定输入；后续仍须分别完成 Planning Review、实施、非作者代码/制品 Review、Lead 核证及适用真实环境验证。
