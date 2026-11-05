# article-management-S4.5 独立代码评审 R01

- Reviewer：Gauss（独立非作者 Reviewer）
- 实现作者：Backend-Dev Aristotle the 2nd / `01a17225-a3d0-7fd1-aeb9-622f7617f436`
- 评审日期：2026-10-25
- objectType：code / test artifacts
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **Verdict：CHANGES REQUIRED**
- **Code Review Gate：FAIL**

## 1. 结论摘要

四文件所有权、精确测试符号、Ready20/Unavailable6 authority 消费、固定 `sha256:bbbb` 跨引用边界、runbook 十个 H2 与主要 strict/mutation 检查均符合规划；专用普通、race 和 scoped vet 也真实执行且未出现 `[no tests to run]`。

但验证器仍有 3 个 MEDIUM 合同缺口：

1. 非法 UTF-8 字节可被 Go JSON decoder 替换后继续验证；
2. 敏感 canary 扫描只覆盖 `positive_cases[]`，未覆盖 inventory 与 keys 等非负例合成内容；
3. 共享 Ready20 canonical fixture 仅在可选 case ID 命中时校验，改名即可绕过固定 fixture。

另有 1 个 LOW 的无关旧 fixture 路径耦合。上述 MEDIUM 会使“不合格制品被专用 Gate 接受”，因此本修订不能 PASS，交回原 Backend-Dev 在原四文件边界内最小修复。

严重度统计：**0 HIGH、3 MEDIUM、1 LOW**。

## 2. 固定输入与对象

### 2.1 规划与设计输入

| 输入 | SHA256 | 开始核对 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `88EF11F034CC8C25919B01EBE62D58AD4387C1260DE1E58C8881676A5FCA52FC` | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `BAB8DA02745C25CF287EBE655123D6B2104A1560EF1B5A36D3F61C57550BB9AA` | MATCH |
| `.spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r02.md` | `A12062129944D8978D92A9732F255F7F1332DE7324AEA9BD3CF7185A7BC3693B` | MATCH |
| `.add/project/knowledge/design-workspace/article-management/design.md` | `AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5` | MATCH |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | `85D8A7CC6ABF01A88B24726EEE3957691AA0B8A7B6235FEB8A31677D77336DDF` | MATCH |

### 2.2 被评四文件

| 文件 | 开始 SHA256 |
|---|---|
| `docs/runbooks/article-deletion-recovery.md` | `41ECB0CF593511119A1BBC910C3B707D094282D1D06875134F531D58F6DD4A09` |
| `testdata/recovery/article-backup-inventory.json` | `1355094A5A6751E6E479F4769C9D6E41C993F2A054D23ACF948D3F7E5C4E9B8C` |
| `testdata/recovery/article-deletion-fixtures.json` | `E86F32A1731E04F3B1590A87282F71F4332B488F2B6040E74AAF7F5876CF7D81` |
| `test/integration/recovery/article_fixtures_test.go` | `C8C2A451410AC7B89FA84410ED9F4E8502CC11997D22415E1434B18FDE1C83A2` |

评审开始时报告路径不存在，分支/HEAD 匹配，staged 为空。四文件均为共享工作树中的未跟踪成果；未发现第五个 S4.5 实施文件，也未发现本卡修改生产代码、migration、公共合同、design 或 stories。

## 3. 两轮检查范围

### Round 1：固定四文件与机械 Gate

逐行检查：

- 两个 JSON 的单对象、字段集合/顺序/type/null/enum、重复键、trailing data、时间、排序与唯一性；
- inventory/key/tombstone/operation/evidence 的交叉引用；
- mutation 是否从正例 clone 生成、是否真正执行并核对预期错误码；
- runbook H1 起始、十个 H2 的集合/顺序/唯一性、unknown 与 fail-closed 声明；
- build tag、唯一顶层测试符号、精确 `-run`、普通/race/vet/gofmt；
- 第五文件、生产代码、migration、公共合同和设计制品越界。

### Round 2：authority、真实消费者与对抗性漂移

交叉核对：

- `internal/recovery/replay_status_v1.go` 的 Ready20/Unavailable6、`CanonicalJSON`、`CanonicalDigest`、`ReadyFixtureDigestV1` 唯一 authority；
- design r14 的 S4.2/S4.4/S4.5 分工和 S4.4 后续真实恢复边界；
- test-plan r12 §7、§9.2 的 UTF-8、synthetic-only、共享 canonical、精确四文件与防假绿合同；
- 现有 `test/integration/recovery/fixtures_test.go` 被复用 helper 的实际前置条件；
- 能否通过改字节、改 ID、同步交叉引用或植入 canary 使不合格制品仍通过当前专用测试。

## 4. Findings

### MEDIUM

#### `ARTICLE-S4.5-CR-R01-F001` — strict parser 未拒绝字符串中的非法 UTF-8

- **位置**：`test/integration/recovery/article_fixtures_test.go:136-146,149-176`
- **合同依据**：`test-plan.md:478` 要求两个 JSON 必须是 UTF-8；不合格输入必须 fail closed。
- **证据**：`parseArticleJSONDocument` 直接把原始字节交给 `json.Decoder`，没有 `utf8.Valid(raw)` 或等价前置校验，也没有非法 UTF-8 负例。Go `encoding/json` 对 quoted string 中的非法 UTF-8 不报错，而是替换为 U+FFFD；本轮 `go doc encoding/json.Unmarshal` 也明确给出该行为。
- **触发**：在任一 JSON 字符串值中写入非法 UTF-8 字节；decoder 将其规范化为 replacement character，后续 schema/语义检查仍可能全部通过。
- **影响**：专用测试会把违反 UTF-8 合同的制品登记为 PASS，strict parse 与实际仓内字节不一致。
- **最小修复方向**：在两个 JSON 进入 decoder 前显式校验原始字节为有效 UTF-8，并增加对 inventory、fixtures 各至少一个 quoted-string 非法字节负例；继续使用现有 `invalid_json` 分类即可，无需第五文件。
- **复审验证**：正例不变；两个非法 UTF-8 用例均确定性失败，专用普通/race 仍无零测试。

#### `ARTICLE-S4.5-CR-R01-F002` — synthetic/敏感 canary 防线没有覆盖 inventory 与 keys

- **位置**：`test/integration/recovery/article_fixtures_test.go:542-545,875-947,989-1011,1284-1321`
- **合同依据**：`test-plan.md:480,541,600` 要求所有 ID、引用、摘要和说明均为公开合成值，不得含真实凭据、DSN、生产路径或 PII，并要求 canary 防线。
- **证据**：`scanArticlePositiveCanaries` 只从每个 `positive_cases[]` 调用。inventory 的 `backup_ref`、`retirement_owner_ref`、`retirement_evidence_ref` 以及 `keys[].synthetic_material` 未进入扫描；key material 只校验 `SYNTHETIC-TEST-ONLY-` 前缀。
- **触发**：
  - 把已退役 backup 的 `retirement_owner_ref` 改为含 `postgres://...` 的非空字符串；或
  - 把 key material 改为 `SYNTHETIC-TEST-ONLY-password=...`。

  两者都满足当前 schema/状态检查，并可通过专用测试。
- **影响**：固定样本当前虽未发现 secret/PII，但后续误放入 DSN/credential 的制品不会被自动 Gate 拦截，违反测试专用公开合成数据边界。
- **最小修复方向**：在不扫描用于验证拒绝行为的 `negative_cases[].operand_json` 的前提下，把同一 canary 策略覆盖到 inventory 全部字符串及 fixtures 的 keys、顶层非负例内容；增加 inventory retirement ref 与 key material 的 mutation 回归。不得读取环境变量或引入生产 secret scanner/provider。
- **复审验证**：上述两个触发均返回 `pii_forbidden`，现有 intentional canary 负例仍能从正例副本执行而不会使基线文件自失败。

#### `ARTICLE-S4.5-CR-R01-F003` — 共享 Ready20 canonical fixture 可通过改名绕过

- **位置**：`test/integration/recovery/article_fixtures_test.go:1081-1108,1211-1239`
- **合同依据**：`test-plan.md:482,504,551-557,600` 固定共享 canonical 样本的前 16 字段、digest、后四字段与 backup ref，并要求重算校验；`internal/recovery/replay_status_v1.go:23-24` 提供共享 `ReadyFixtureDigestV1` authority。
- **证据**：覆盖状态只要求“存在任意 Ready”；固定 canonical bytes 与 `ReadyFixtureDigestV1` 仅在 `case_id == "article-recovery-ready-canonical"` 时校验，但代码没有要求该 case ID 必须存在或只能存在一次。
- **触发**：把 canonical case 改名并同步 `negative_cases[].base_case_id`，再修改例如 `run_id` 并用现役 authority 重算自洽 digest。当前校验仍有一个合法 Ready，条件分支不再执行，专用测试可通过。
- **影响**：S4.5 Gate 无法持续锁定跨模块共享 fixture；测试数据可以漂移成另一个自洽 Ready20，而不再证明固定 canonical/evidence authority 一致。
- **最小修复方向**：显式要求且仅要求一个共享 canonical case，并无条件核其固定前 16 bytes、固定 digest、status/evidence_ref/completed_at 与 inventory canonical backup ref；增加“共享 case 缺失/改名”和“固定字段漂移但 digest 自洽”的负例。无需定义 inventory digest 算法。
- **复审验证**：改名、缺失、重复或自洽重算后的共享 case 均 fail closed；合法固定样本继续直接消费 `internal/recovery` authority。

### LOW

#### `ARTICLE-S4.5-CR-R01-F004` — 专用测试定位根目录时依赖两个无关旧 fixture

- **位置**：`test/integration/recovery/article_fixtures_test.go:61-64`；`test/integration/recovery/fixtures_test.go:268-318`
- **合同依据**：`test-plan.md:598` 要求唯一专用测试独立读取三个 S4.5 产物，只复用现有通用 strict JSON/重复键辅助逻辑。
- **证据**：新测试复用 `moduleRoot`；其 `fixtureModuleRoot` 强制要求无关的 `keyring-fixtures.json` 与 `backup-manifest.json` 为 regular file，之后才允许读取 S4.5 三产物。
- **触发**：旧 recovery fixture 被移动、重命名或单独损坏，即使三个 S4.5 产物和 `go.mod` 完整，`TestArticleRecoveryFixtures` 也会在定位根目录时失败。
- **影响**：形成跨任务隐藏耦合和无关假失败，专用 Gate 不能只反映 S4.5 制品状态。
- **最小修复方向**：在允许的测试文件内使用只依赖 module marker 与三个 S4.5 目标路径的定位方式；不要修改旧 `fixtures_test.go`，不要增加第五文件。
- **复审验证**：确认专用测试的 root discovery 不再把两个旧 fixture 当作前置，同时从仓库根和 package 目录执行均能读取目标三产物。

## 5. 已确认正确的范围

- 精确四文件，无第五文件、生产代码、migration、公共 DTO、design/stories 改动。
- 测试文件首行精确为 `//go:build integration`；唯一顶层测试函数为 `TestArticleRecoveryFixtures`。`testArticleFixtureMutations`、`testArticleFixtureTrailingData` 是小写 helper，不改变精确锚点。
- 专用普通/race 输出真实 `ok`，未出现 `[no tests to run]`；scoped vet 通过。
- Ready20/Unavailable6 直接消费 `internal/recovery.DecodeReplayStatusV1`、`CanonicalJSON`、`CanonicalDigest` 与 `ReadyFixtureDigestV1`，未复制第二套 canonical encoder。
- `inventory_digest` 只与固定合成值 `sha256:bbbb` 逐字比较；未定义或重算 production inventory digest。
- 当前正例覆盖 Ready、pending、unknown、failed，inventory 三类时间/状态样本及 retirement true/false；主要字段顺序、类型/null、重复键、trailing data、UTC 微秒、排序/唯一性和跨引用均有实际检查。
- mutation 从解析后的正例文档 clone 后生成，当前目录中的各 mutation 会实际执行并核对错误码。
- runbook 当前精确包含十个要求的 H2，顺序和唯一性正确；retention/RPO/RTO 均为 unknown，且明确不构成真实恢复、灾备演练、授权或 production readiness。
- 当前四文件未发现真实 secret、DSN 或 PII；命中的 email/password 文本均为测试 canary/拒绝逻辑。
- 制品的结构可作为后续 S4.4 合成输入与机械 Gate 基线；但 S4.4 按 design r14 仍必须消费真实 inventory、真实旧备份与 Journal/keyring 并独立验证，不能从本评审外推。

这些正确项不关闭 F001～F003。

## 6. Reviewer 命令与结果

```text
git branch --show-current
git rev-parse HEAD
Get-FileHash -Algorithm SHA256 <固定输入与四文件>
git diff --cached --name-only
git status --short -- <四文件与报告>
```

结果：`main@6da8abbceb0fa197364810a92a8fa2d97b38ddca`；全部开始 SHA 匹配；开始 staged 为空；四个被评文件为未跟踪成果，报告开始时不存在。

```text
# 精确符号前置 Gate
Test-Path test/integration/recovery/article_fixtures_test.go
Get-Content ... -TotalCount 1
Select-String ... '^func TestArticleRecoveryFixtures\(t \*testing\.T\) \{$'
# 另计顶层 Test 函数
```

结果：PASS；首行、精确符号均匹配，顶层 Test 数量为 1。

```text
$env:GOPROXY='off'
$env:GOSUMDB='off'
go test -tags=integration ./test/integration/recovery -run '^TestArticleRecoveryFixtures$' -count=1
go test -race -tags=integration ./test/integration/recovery -run '^TestArticleRecoveryFixtures$' -count=1
go vet -tags=integration ./test/integration/recovery
gofmt -l test/integration/recovery/article_fixtures_test.go
```

结果：

- 专用普通：PASS，`ok han/test/integration/recovery 0.108s`，无 `[no tests to run]`；
- 专用 race：PASS，`ok han/test/integration/recovery 1.413s`，无 `[no tests to run]`；
- scoped vet：PASS；
- `gofmt -l`：无输出。

```text
go doc encoding/json.Unmarshal
```

结果：本机 Go 文档确认 quoted string 的非法 UTF-8 不作为错误，而会替换为 U+FFFD，支持 F001 的静态判断。

## 7. 实际读取

完整读取：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.codex/skills/add-delivery-workflow/SKILL.md`
- `.codex/guidance/task-execution-workflow.md`
- `.codex/profiles/go-service/README.md` 及适用 coding/no-hardcode/security rules
- 四个被评文件
- `internal/recovery/replay_status_v1.go`

定向读取/检索：

- 固定 Tasks、Plan、Planning Review R02、design r14、test-plan r12
- `test/integration/recovery/fixtures_test.go` 的共享 helper 与 module root 定位
- S4.4 后续消费者、S4.2/S4.4/S4.5 分工和真实恢复边界
- 四文件相关符号、现状目录、写集和 staged 状态

## 8. 未验证边界

- 未运行 PostgreSQL、网络、部署、Host、E2E 或真实旧备份恢复。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未独立重跑 Manifest 七项全仓检查；Lead 已报告其为 exit 0，但本结论不以该转述代替 Reviewer 对合同的判断。
- 未验证真实 inventory、真实 keyring/Journal、retirement evidence、cleanup 授权、retention/RPO/RTO、灾备窗口或 production readiness。
- 未对共享工作树其他大量未提交成果作正确性结论。

## 9. 结束验证与 Gate

结束核对结果：

- 分支/HEAD 仍为 `main@6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 四个被评文件及五个固定输入的结束 SHA256 均与 §2 开始值相同，0 mismatch。
- `git diff --check -- .spec/wip/article-management/review-code-s4.5-r01.md`：PASS；另以 no-index 方式核未跟踪报告，无 whitespace diagnostic。
- staged 为空。
- scoped status 只显示四个原被评未跟踪文件与本报告；Reviewer 除本报告外未写入任何文件，未 stage/commit/push/切分支、未清理/移动/回滚他人内容，未触碰 `han-server.exe`。

**Code Review Gate：FAIL / CHANGES REQUIRED。**

复审返回点：原 Backend-Dev 在原四文件内修复 F001～F003，并说明 F004 的处理；固定新 SHA 后由非作者 Reviewer 定点复审。专用命令或 Manifest PASS 不能替代这些合同缺口的关闭。
