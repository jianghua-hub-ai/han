# article-management-S4.5 独立代码评审 R02

- Reviewer：Gauss（原独立非作者 Reviewer）
- 实现作者：Backend-Dev Aristotle the 2nd / `01a17225-a3d0-7fd1-aeb9-622f7617f436`
- 评审日期：2026-10-25
- objectType：code / test artifacts
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/article-management/review-code-s4.5-r01.md`
- 原报告 SHA256：`44DE54697DFEA8964E62A9AE2B8196C24E3233A44E177E04410F331AB64E5FF8`
- **Verdict：PASS**
- **Code Review Gate：PASS**

## 1. 结论

R01 四项 finding 的复审结果：

| Finding | 原严重度 | R02 状态 |
|---|---:|---|
| `ARTICLE-S4.5-CR-R01-F001` | MEDIUM | **CLOSED** |
| `ARTICLE-S4.5-CR-R01-F002` | MEDIUM | **CLOSED** |
| `ARTICLE-S4.5-CR-R01-F003` | MEDIUM | **CLOSED** |
| `ARTICLE-S4.5-CR-R01-F004` | LOW | **CLOSED** |

返工仍严格位于原四文件边界，实际只改变 `test/integration/recovery/article_fixtures_test.go`。未发现返工引入的新 HIGH 或 MEDIUM finding，也未发现需要保留的新 LOW finding。

本 PASS 仅表示 S4.5 四个仓内 synthetic fixture/runbook 制品及其机械验证通过本轮独立代码复审；不证明真实 inventory、真实旧备份恢复、retirement/cleanup 授权、灾备、Host、部署、E2E 或 production readiness。

## 2. 固定对象与开始核对

### 2.1 固定输入

| 输入 | 开始 SHA256 | 结果 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `88EF11F034CC8C25919B01EBE62D58AD4387C1260DE1E58C8881676A5FCA52FC` | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `BAB8DA02745C25CF287EBE655123D6B2104A1560EF1B5A36D3F61C57550BB9AA` | MATCH |
| `.spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r02.md` | `A12062129944D8978D92A9732F255F7F1332DE7324AEA9BD3CF7185A7BC3693B` | MATCH |
| `.add/project/knowledge/design-workspace/article-management/design.md` | `AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5` | MATCH |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | `85D8A7CC6ABF01A88B24726EEE3957691AA0B8A7B6235FEB8A31677D77336DDF` | MATCH |
| R01 报告 | `44DE54697DFEA8964E62A9AE2B8196C24E3233A44E177E04410F331AB64E5FF8` | MATCH |

### 2.2 返工四文件

| 文件 | 开始 SHA256 | 与 R01 revision |
|---|---|---|
| `docs/runbooks/article-deletion-recovery.md` | `41ECB0CF593511119A1BBC910C3B707D094282D1D06875134F531D58F6DD4A09` | 未变 |
| `testdata/recovery/article-backup-inventory.json` | `1355094A5A6751E6E479F4769C9D6E41C993F2A054D23ACF948D3F7E5C4E9B8C` | 未变 |
| `testdata/recovery/article-deletion-fixtures.json` | `E86F32A1731E04F3B1590A87282F71F4332B488F2B6040E74AAF7F5876CF7D81` | 未变 |
| `test/integration/recovery/article_fixtures_test.go` | `88FD408A466CAF208E780112F82B6EB82BCCDB58D5CB1E848C07B52DFDC3C4F6` | 已返工 |

开始时 R02 报告不存在；branch/HEAD 匹配；staged 为空。四个被评文件仍为共享工作树中的未跟踪成果，未发现 S4.5 第五文件或生产代码、migration、公共合同、Tasks/Plan、design/stories 的本卡扩权。

## 3. 两轮复审范围

### Round 1：R01 F001～F004 定点关闭

- 逐项读取 R01 的触发、影响、最小修复与复审条件；
- 检查 revised test 对原始字节、canary 扫描、共享 canonical 和 module root 的实际控制流；
- 确认新增回归通过 `validateArticleArtifactPair` 等真实入口，而非只测试孤立 helper；
- 从仓库根和 `test/integration/recovery` package 目录分别执行精确目标测试。

### Round 2：返工回归与新风险扫描

- 复核首行 build tag、唯一顶层测试符号、无 `t.Skip`、精确锚定及零测试防假绿；
- 复核原 strict schema、mutation、交叉引用、Ready20/Unavailable6 authority、runbook 十个 H2 和 fixed `sha256:bbbb` 边界未弱化；
- 检查新 root discovery 的路径来源、向上遍历、regular-file marker 与无关旧 fixture 解耦；
- 对 revised file 执行普通、race、scoped vet、gofmt 和四文件 diff-check，并扫描新 HIGH/MEDIUM。

## 4. R01 findings 复审

### `ARTICLE-S4.5-CR-R01-F001` — MEDIUM — CLOSED

- **原问题**：Go JSON decoder 会把 quoted string 中的非法 UTF-8 替换为 U+FFFD，原 validator 没有原始字节前置拒绝。
- **修订证据**：
  - `article_fixtures_test.go:212-215` 在任何 token decode 前对完整 raw document 执行 `utf8.Valid(raw)`，失败统一返回 `invalid_json`；
  - inventory 与 fixtures 均由 `validateArticleArtifactPair:395-403` 先进入该 parser，因此前置校验覆盖两个真实文件入口；
  - `:1738-1774` 分别在 inventory retirement owner quoted string 和 fixtures description quoted string 中注入原始 `0xff`，随后调用完整 `validateArticleArtifactPair` 并精确断言 `invalid_json`。
- **复验结果**：`invalid_UTF-8/inventory_quoted_string` 与 `fixtures_quoted_string` 均真实执行并 PASS；若 `bytes.Replace` 未命中，基线验证将返回 nil，断言会失败，不存在只测 helper 或 mutation 未生效的假绿。
- **关闭理由**：原始字节、两个文档与确定性错误分类均满足 R01 关闭条件。

### `ARTICLE-S4.5-CR-R01-F002` — MEDIUM — CLOSED

- **原问题**：canary 扫描只覆盖 `positive_cases[]`，inventory retirement refs 和 key material 可夹带 DSN/credential。
- **修订证据**：
  - `article_fixtures_test.go:404-408` 对完整 inventory node 调用 `scanArticleCanaries`，并对完整 fixtures node 调用 `scanArticleFixtureCanaries`；
  - `:1401-1435` 扫描 fixtures 的所有非 `negative_cases` 顶层字段；对 `negative_cases[]` 继续扫描 case metadata，但显式跳过用于构造 intentional 拒绝用例的 `operand_json`；
  - `:1437-1467` 递归覆盖 object/array/string，沿用 email、PostgreSQL DSN、password、credential 与 private-key canary；
  - `:1776-1812` 通过真实 artifact-pair 入口分别把 retirement owner ref 改为 PostgreSQL DSN、把 key material 改为带允许前缀的 `credential=` 字符串，并精确断言 `pii_forbidden`。
- **复验结果**：两个 canary 子测试均执行并 PASS；基线 fixtures 中 intentional email/credential operand 仍能通过正例加载并由 mutation 负例单独使用，证明 skip 仅限 `negative_cases[].operand_json`，没有把整个 negative catalog 漏扫。
- **关闭理由**：inventory、keys、其他非负例内容与 intentional operand 的边界已确定，原两个绕过触发均关闭。

### `ARTICLE-S4.5-CR-R01-F003` — MEDIUM — CLOSED

- **原问题**：固定 shared Ready20 只在一个可选 case ID 命中时检查，改名并重算 digest 可绕过。
- **修订证据**：
  - `article_fixtures_test.go:1185-1200,1215-1221` 登记 `sharedCanonical` 计数，并要求最终精确等于 1；case ID 仍受全局唯一检查；
  - `:1277-1305` 要求 shared ID 必须是 Ready，并在该 ID 命中时无条件进入 shared validator；
  - `:1329-1348` 同时核对现役 `CanonicalJSON` 结果、固定前 16 bytes、`recovery.ReadyFixtureDigestV1`、DTO evidence digest、`status=ready`、固定 backup ref、evidence ref 与 completed_at；
  - generic reference 校验 `:1255-1274,1307-1326` 继续把 fixed Ready backup ref/evidence ref 与 case references、inventory 及固定 `sha256:bbbb` 逐字绑定；
  - `:1814-1874` 覆盖 shared ID 改名/缺失、重复，以及修改 `run_id` 后使用现役 authority 重算自洽 digest 的漂移。
- **复验结果**：`renamed_is_missing`、`duplicate`、`self_consistent_canonical_drift` 均通过真实 artifact-pair 入口确定性返回 `invalid_state`。
- **边界确认**：返工只重算 Ready 前 16 字段的 evidence digest；inventory digest 仍只与固定合成 `sha256:bbbb` 比较，没有定义 production inventory digest payload、canonical 或算法。
- **关闭理由**：shared ID 的存在/唯一性、固定 canonical/authority digest、后四字段与 canonical backup ref 均已锁定，自洽漂移不再可通过。

### `ARTICLE-S4.5-CR-R01-F004` — LOW — CLOSED

- **原问题**：新测试复用旧 `moduleRoot`，被无关 `keyring-fixtures.json` 和 `backup-manifest.json` 绑定。
- **修订证据**：
  - `article_fixtures_test.go:83-165` 使用本文件私有 `articleModuleRoot/findArticleModuleRoot/isArticleModuleRoot`；
  - root marker 仅要求 `go.mod` 和三个 S4.5 目标产物为 regular file，不再读取或要求旧 keyring/backup manifest；
  - 搜索 revised file 不再调用旧 `moduleRoot`；
  - `:1906-1934` 在临时层级中只创建上述四个 marker，从 root 与嵌套 package 路径分别向上定位并要求精确返回同一 root。
- **独立运行证据**：
  - 从仓库根执行精确目标：PASS；
  - 从 `test/integration/recovery` 目录执行 `go test -tags=integration . -run '^TestArticleRecoveryFixtures$' -count=1`：PASS。
- **路径判断**：搜索起点仅来自 cwd、当前测试源绝对路径或测试 executable 目录；解析只做 `filepath.Abs` 后逐级取 parent，不拼接外部输入或跟随用户提供的相对 target。marker 同时要求 module marker 与三个精确目标 regular file，未发现可由本返工引入的路径遍历或单一 `go.mod` 假阳性。
- **关闭理由**：无关旧 fixture 依赖已移除，仓库根/package 目录两种执行方式均成立。

## 5. 新 finding 与守护项

### 新 finding

- HIGH：0
- MEDIUM：0
- LOW：0

### 未弱化的原 Gate

- 首行仍精确为 `//go:build integration`。
- 精确 `func TestArticleRecoveryFixtures(t *testing.T) {` 数量为 1；顶层 Go Test 总数为 1；`t.Skip` 数量为 0。
- 专用普通与 race 均真实列出并执行全部子测试，无 `[no tests to run]`。
- 原 table-driven mutation、duplicate key、trailing data、strict field order/type/null/enum、UTC 微秒、排序/唯一性、cross-reference、Ready20/Unavailable6 与 runbook 检查继续执行。
- Ready20 继续直接消费 `internal/recovery` authority；没有复制第二 canonical encoder。
- runbook、inventory 和 fixtures 三个数据文件内容及 SHA 均未改变，retention/RPO/RTO 仍为 unknown，未外推真实恢复或 readiness。
- 精确四文件边界保持；没有第五文件、production provider、migration、公共 DTO 或 inventory digest 算法。

## 6. Reviewer 独立命令与结果

```text
git branch --show-current
git rev-parse HEAD
Get-FileHash -Algorithm SHA256 <R01、五个固定输入、四个被评文件>
git diff --cached --name-only
git status --short -- <四文件、R01、R02>
```

结果：开始 branch/HEAD 与全部 SHA 匹配，0 mismatch；R02 开始时不存在；staged 为空。

```text
# build tag / symbol / skip Gate
Get-Content test/integration/recovery/article_fixtures_test.go -TotalCount 1
Select-String ... '^func TestArticleRecoveryFixtures\(t \*testing\.T\) \{$'
Select-String ... '^func Test[A-Za-z0-9_]*\(t \*testing\.T\) \{$'
Select-String ... '\bt\.Skip(f|Now)?\('
```

结果：首行匹配；精确符号 1；顶层 Test 1；`t.Skip` 0。

```text
$env:GOPROXY='off'
$env:GOSUMDB='off'
go test -v -tags=integration ./test/integration/recovery -run '^TestArticleRecoveryFixtures$' -count=1
```

结果：PASS，`ok han/test/integration/recovery 0.179s`；F001～F004 新增子测试及原 mutation/runbook 全部实际列出，无 skip、无 `[no tests to run]`。

```text
# workdir: test/integration/recovery
go test -v -tags=integration . -run '^TestArticleRecoveryFixtures$' -count=1
```

结果：PASS，`ok han/test/integration/recovery 0.175s`；证明 package 目录执行可定位三个 S4.5 产物。

```text
go test -race -tags=integration ./test/integration/recovery -run '^TestArticleRecoveryFixtures$' -count=1
go vet -tags=integration ./test/integration/recovery
gofmt -l test/integration/recovery/article_fixtures_test.go
git diff --check -- <四个被评文件>
```

结果：race PASS，`ok han/test/integration/recovery 1.646s`；vet PASS；gofmt 无输出；四文件 diff-check PASS。

本轮没有重复运行 Lead 已完成的 `go mod verify`、全仓 vet/build/contract/race/all；这些全仓 Gate 仍由 Lead 的同 revision 现场证据承载，不替代本 Reviewer 对原 finding closure 的独立判断。

## 7. 实际读取与未验证边界

### 实际读取

- 完整读取 R01 与 revised `test/integration/recovery/article_fixtures_test.go`；
- 复用同一连续评审上下文中已完整读取且版本未变的 Reviewer Role、`add-code-review`、development-delivery Review Gate、task-execution、适用 Go profile、coding/no-hardcode/security rules；
- 定向复核固定 Tasks/Plan、Planning Review R02、design r14、test-plan r12、`internal/recovery/replay_status_v1.go`、旧 `fixtures_test.go` helper、三个未变数据制品及 S4.4 后续消费者边界。

### 未验证边界

- 未运行 PostgreSQL、网络、部署、Host、E2E 或真实旧备份恢复。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未验证真实 inventory、Journal/keyring、retirement evidence、cleanup 授权、retention/RPO/RTO 决策、灾备窗口或 production readiness。
- 未评价共享工作树中与本返工无关的大量未提交成果。

## 8. 结束验证与最终 Gate

结束核对结果：

- branch/HEAD 仍为 `main@6da8abbceb0fa197364810a92a8fa2d97b38ddca`；
- R01、五个固定输入及四个被评文件结束 SHA 与 §2 开始值一致，0 mismatch；
- `git diff --check -- .spec/wip/article-management/review-code-s4.5-r02.md`：PASS；
- staged 为空；
- scoped status 只显示四个原被评未跟踪文件、R01 与本 R02；Reviewer 除本 R02 报告外未写入任何文件，未 stage/commit/push/切分支、未删除/移动/清理或触碰 `han-server.exe`。

**最终 Verdict：PASS。**

**Code Review Gate：PASS。**

F001～F004 全部 CLOSED；无新增 HIGH/MEDIUM finding。后续仍需 Lead 在相同 revision 核固定哈希、工作树边界与全量 Gate 后，才可决定是否关闭 task。
