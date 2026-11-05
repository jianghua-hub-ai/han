# article-management test-plan r11 / S4.2、S4.5 Test-plan Review R01

- Reviewer：独立 Reviewer Gauss
- 被审作者：Test-Engineer Ohm
- 评审日期：2026-10-25
- objectType：test-plan
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定对象：`.add/project/knowledge/design-workspace/article-management/test-plan.md`
- revision：`article-management-test-plan-ir01-r11`
- status：`planned-not-executed`
- 固定 SHA256：`0ACE0A8A47DC4B2126F291F88B4C3397BBCE18B1FB1FC2C2DBA081014AD6EA20`
- design r14 SHA256：`AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`
- Design Review R02 SHA256：`85DAC7A66F9555E6DF711472E7F5A9A6448F281EC2496DFDDBD5B7DAC19DE143`，Gate `PASS`
- architecture r28 SHA256：`554686B77D0557F16A735F49F63626D922F41DAE8A8574B17980B7CB55CF1BCC`
- **Test-plan Review Gate：CHANGES REQUIRED**

## 1. 结论

两轮检查发现 **0 CRITICAL、0 HIGH、2 MEDIUM、0 LOW**。

S4.2 的精确 19 文件、前向 `0009`、production inventory、source/wiring、attempt CAS、UTC 微秒、单一 ReplayStatus authority、A-REC-04～07 真实 PostgreSQL证据路径和防零测试 Gate，与 design r14、Design Review R02、architecture r28/ADR-0020 一致且总体可实施。S4.5 的四文件写集、唯一测试符号和 `[no tests to run]` 防假绿合同也已形成。

但 r11 仍有两处当前合同不一致：其一把已关闭的 S4.6/B6A-W 写成 todo/open/not-started；其二同时把 S4.5 的 `inventory_digest` 定义为不可重算的固定合成引用，又要求验证器取得“实际重算值”。后者使四文件实现者无法确定应做固定跨引用检查还是另创摘要算法。故本 revision 不通过独立 Test-plan Gate。

## 2. 两轮检查范围

### Round 1：固定 test-plan 的直接完整性与可执行性

- 核 frontmatter revision/status、规划与执行边界；
- 将 §9.3 的 19 文件逐项与 design r14 建议写集比较；
- 核 `0009_article_replay_attempt.sql` 仅前向新增且禁止修改 `0001`～`0008`；
- 核 reader/source/wiring、ReplayStatus/evidence、`TestDeletionReplayPostgres` 的文件、符号、build tag、正则和 `[no tests to run]` Gate；
- 核 A-REC-04～07 的多连接/多事务、三类 Begin commit unknown、CAS、旧写晚到、UTC 微秒、TOCTOU/freshness、input drift 和唯一 authority；
- 核 S4.5 四文件、`TestArticleRecoveryFixtures`、strict fixture/runbook、负例矩阵及不冒充 production provider/真实恢复的边界。

### Round 2：上游、Tasks/Plan、公共消费者与真实目录交叉核对

- 定向核 architecture r28、ADR-0020、design r14 与 Design Review R02；
- 完整核公共 `internal/recovery/replay_status_v1.go` 的 Ready20/Unavailable6；
- 核 Tasks/Plan 中 S4.2、S4.5、S4.6/B6A-W 的当前状态和写集；
- 核当前五个 S4.2 候选文件及其测试符号、拟新增 source/reader/integration/0009 文件是否存在；
- 完整核 `test/integration/recovery/fixtures_test.go` 及 `testdata/recovery` 现状；
- 核 S4.5 四个建议文件当前均不存在，现有 `TestRecoveryFixtures` 不消费它们。

## 3. Findings

### `ARTICLE-TP-R11-S42-S45-R01-F001` — MEDIUM — 当前 S4.6 状态与 Tasks/Plan SSOT 冲突

- **位置**：test-plan `416-420`、`614`；正确边界已另见 `603`、`623-625`、`841`、`846`。
- **触发**：Task-Planner、实施者或 Reviewer 直接消费覆盖映射“登记状态”列或 §9 任务建议，而未先跳转到后文历史说明。
- **证据**：
  - `416-420` 仍写 S4.6/B6A-W 为 `todo`、planning findings `OPEN`、未认领/未启动；
  - `614` 仍称唯一 todo、planning-side findings OPEN、未启动；
  - 现役 Tasks 已将 `article-management-S4.6` 标为 `[x]`、完成于 2026-10-24；Plan 将 B6A-W 标为 `CLOSED`，并记录 Planning、Code Review 与 Lead Gate 均已关闭；
  - r11 自身 `603`、`623-625`、`846` 也承认 S4.6 task-level 已关闭、旧 handoff 只作历史追溯。
- **影响**：同一 test-plan 给出两套当前状态，可能导致错误重开 S4.6、重复分配 production wiring owner，或把 S4.2/S4.5 的后续串行关系建立在不存在的 OPEN planning Gate 上；也违反“实际状态只引用 Tasks/Plan”的自述。
- **最小修复**：把 `416-420` 和 `614` 的当前状态同步为 S4.6/B6A-W task-level CLOSED，并保留“真实 PostgreSQL、Unix 持久卷、Host/部署/E2E、production readiness 未验证”的独立边界。旧 todo/open/not-started 内容只保留在明确标记为历史的 §9.1。
- **验证**：复核所有非历史章节不再把 S4.6/B6A-W 描述为 todo/open/not-started；与 Tasks/Plan 的 `[x]`/CLOSED 及 r11 `846` 一致，且不把局部关闭外推为真实环境通过。

### `ARTICLE-TP-R11-S42-S45-R01-F002` — MEDIUM — S4.5 inventory digest 同时被定义为固定引用和未定义的重算值

- **位置**：test-plan `480`、`497`、`550`、`598`、`704`。
- **触发**：实现 `article_fixtures_test.go` 的合法样本、`digest_mismatch` 负例或“canonical/digest”自动检查。
- **证据**：
  - `480`、`497` 明确 `inventory_digest` 固定为合成跨引用 `sha256:bbbb`，并明确它不是 production digest 算法声明，验证器不得把本值或文件序列化方式实现成 production 公共合同；
  - `550` 又要求 Ready 的 `inventory_digest` 是 inventory“实际重算值”；
  - §7 未定义该测试包装 inventory 的 digest payload、canonical bytes 或摘要算法；`sha256:bbbb` 也不是完整 SHA-256；
  - `598`、`704` 仍把 S4.5 验证责任概括为 `canonical/digest`。
- **影响**：四文件实现者只能在“固定值/跨文件逐字一致”与“自行发明测试或生产摘要算法”之间猜测。前者不满足“实际重算”，后者缺合同且可能越过 S4.5 不定义 production provider/协议的边界，导致不可复核或假阳性的 digest 覆盖。
- **最小修复**：优先保持现有测试专用边界，将 `550` 改为“Ready 的 `inventory_digest` 必须与 inventory 文件中固定合成引用 `sha256:bbbb` 逐字一致”，并把 `598`、`704` 的 S4.5 `canonical/digest` 表述收窄为“Ready20 canonical/evidence digest + inventory 固定跨引用一致性”。若确需重算，则必须在本 test-plan 内另行定义纯测试专用 payload、canonical 和完整摘要值，并明确禁止生产复用。
- **验证**：正例和 `digest_mismatch` 负例可由四文件内唯一确定地实现；无需修改 design、production schema/API、现有 fixture、第五文件或 production provider；测试输出能区分 Ready evidence digest 重算与 inventory 固定引用检查。

## 4. 重点核对结果

| 核对项 | 结果 | 证据/限制 |
|---|---|---|
| S4.2 精确 19 文件 | 对齐 | §9.3 与 design r14 `2304-2322` 逐项一致；无第 20 文件条件扩权 |
| `0009` 与既有 migration | 对齐 | 仅新增 `migrations/0009_article_replay_attempt.sql`；禁止修改 `0001`～`0008`；当前仓库只有 `0001`～`0008` |
| 共享 `cmd/han-server` / S4.6 串行 | 合同对齐，但当前状态文本有 F001 | §9.3 禁止第二 root；S4.6 已关闭事实须统一 |
| reader/source/wiring 符号 Gate | 可执行 | 三个精确顶层测试符号均有存在检查；缺文件/符号即失败 |
| PostgreSQL 精确测试 Gate | 可执行 | integration 文件、首行 tag、`TestDeletionReplayPostgres` 均有前置检查；普通/race 命令精确锚定；`[no tests to run]` 为 FAIL |
| A-REC-04～07 | 覆盖充分 | 多连接/多事务、三类 Begin commit unknown、CAS、回滚、旧写晚到、UTC 微秒、input drift/current/evidence 原子性均有证据路径；不得由 memory/fixture/compile-only 替代 |
| production inventory strict v1 | 覆盖充分 | version/字段顺序/UTC 六位微秒/canonical SHA-256/1 MiB/no-symlink/regular/readonly/TOCTOU/freshness 及 fail-closed 矩阵齐全 |
| ReplayStatus authority | 对齐 | 公共代码保持 Ready20/Unavailable6，无 generation/token；test-plan 禁止第二 authority |
| S4.5 精确四文件 | 对齐 | runbook、两个 JSON、`article_fixtures_test.go`；第五文件触发停止 |
| S4.5 防假绿 | 对齐 | 精确符号、首行 integration tag、锚定命令、禁止 `t.Skip`、`[no tests to run]` FAIL |
| S4.5 digest 语义 | 不通过 | 见 F002 |
| `planned-not-executed` 与非外推 | 对齐 | frontmatter 正确；明确 design PASS、规划和未执行命令不等于 Code/Test/production PASS |
| Tasks/Plan 对账 | 尚未执行且已明确 | S4.2 仍登记旧五文件；S4.5 仍登记三文件，均待本评审后由 Task-Planner 实际对账 |

## 5. 当前目录与规划/执行边界

- 五个既有 S4.2 候选文件存在，SHA256 与 design/Tasks 固定值一致；新的 source、reader、PostgreSQL integration 和 `0009` 均不存在。
- S4.5 建议的四个文件均不存在。
- `test/integration/recovery/fixtures_test.go` 只有 `TestRecoveryFixtures`，只消费 `keyring-fixtures.json` 与 `backup-manifest.json`；没有 `TestArticleRecoveryFixtures`。
- 上述现状只证明“尚待实施/对账”，不构成额外 finding；r11 已把它们标为 blocked/planned-not-executed。
- 本轮按授权未运行任何 Go 测试、PostgreSQL、网络、部署或 E2E；未把规划命令登记成执行 PASS。

## 6. 实际读取

### 完整读取

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.add/project/knowledge/design-workspace/article-management/test-plan.md`
- `.spec/wip/article-management/review-design-r14-s4.2-r02.md`
- `internal/recovery/replay_status_v1.go`
- `test/integration/recovery/fixtures_test.go`

### 定向读取、哈希与检索

- `.add/project/knowledge/design-workspace/article-management/design.md`
- `.add/project/knowledge/design/architecture.md`
- `.add/project/knowledge/decisions/content.md`（ADR-0020）
- `.add/project/knowledge/tasks/tasks.md`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`
- `internal/article/deletion_replay.go`
- `internal/article/replay_evidence.go`
- `internal/article/replay_status_v1.go`
- `internal/article/deletion_replay_test.go`
- `internal/article/replay_status_v1_test.go`
- `cmd/han-server/main_test.go`
- `internal/article`、`test/integration/recovery`、`testdata/recovery` 的相关文件/测试符号现状
- `migrations/0001`～`0008` 及建议 `0009` 的存在性

## 7. 验证与限制

- 评审开始时固定对象 SHA256 为 `0ACE0A8A47DC4B2126F291F88B4C3397BBCE18B1FB1FC2C2DBA081014AD6EA20`。
- 分支/HEAD 为 `main@6da8abbceb0fa197364810a92a8fa2d97b38ddca`；开始时 staged 为空。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未运行测试、真实 PostgreSQL、migration、网络、外部服务、部署或 E2E，因此不验证命令实际结果、数据库锁/CAS/回滚、文件系统 no-follow/ACL/原子替换或真实恢复。
- 未验证未来四文件/19 文件实现质量；本结论只评审固定 r11 的测试规划合同。
- 共享工作树有大量他人未提交成果；除本报告外未修改任何文件，也未 stage/commit/push/切分支、删除、移动或清理。

## 8. 最终 Gate

**CHANGES REQUIRED**

关闭 F001、F002 后，应由原 Test-Engineer 提交新 revision，再由独立 Reviewer 复审；随后才能把该 test-plan 交给 Task-Planner 对账 S4.2 的 19 文件/`0009` 与 S4.5 的四文件写集。本报告不修改 test-plan、Tasks/Plan、代码或 migration，也不替用户接受风险。

固定对象开始 SHA256：`0ACE0A8A47DC4B2126F291F88B4C3397BBCE18B1FB1FC2C2DBA081014AD6EA20`。  
固定对象结束 SHA256：`0ACE0A8A47DC4B2126F291F88B4C3397BBCE18B1FB1FC2C2DBA081014AD6EA20`。
