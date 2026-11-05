# Article S4.6 Tasks/Plan 独立评审 r01

- 日期：2026-10-23
- Reviewer：add-reviewer，非原作者独立评审；未继续委派
- 原作者：Task-Planner Hume `01a16a7b-9353-7141-866b-a0020768aa5b`
- objectType：design（planning 复合对象）
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`

## 1. 固定对象

| 对象 | SHA256 | blob |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `3045DC84FC33255C7D17A145E29D81F15A0F8FA41DC6EB5B47CD4A44F9701CD4` | `190775e9905aeb350ea13c39cd10ce0952d6b51d` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `059CD958187A4962175CC3CE7CD9205311A148487454F709AD908E3EB5171BF9` | `4ad80b8b9bc32990fa8c641aa01cdf052fe59d4f` |

评审开始及落报告前，HEAD、两对象 SHA256/blob 均与委托值一致；报告路径此前不存在。

## 2. 两轮实际检查范围

### Round 1：直接修订检查

- 全文检查 Tasks/Plan 中 `article-management-S4.6`、`B6A-W`、统计、当前事实、变更记录、关闭边界及未验证声明。
- 核对 S4.6 为唯一 todo 开发卡，owner 为 Article Backend-Dev，直接 requires 仅 `identity-auth-S5.2`、`article-management-S4.1`。
- 核对 11 文件、验收命令、停止条件、后继关联以及 S4.3/S4.5/S1.4 的 requires。
- 核对既有 checkbox、active 集合、日期和完成声明未被本次规划错误改写。

### Round 2：跨对象、消费者与 DAG 检查

- 对读 design r10、stories r07、test-plan r06、test-plan r06 定点复审及原 `B6A-G4-R01-F001`。
- 静态解析全部 53 个 task、requires 和批次外层表：53 个 task 各安排一次，无未知、遗漏、重复或环；统计为 20 completed / 2 active / 31 todo，active 仅 S3.3/S3.5。
- 核对 B6A-W 仅含 S4.6；B6B/B6C/B6D 未被建立业务硬依赖。
- 有界核对 11 文件现场：9 个既有文件存在，两个 worker 文件尚不存在；确认唯一 production composition/router 位于 `cmd/han-server/main.go`，现有 deletion schema/index 已在 `migrations/0003_article.sql`，无需 migration。
- 定点检查 Article Journal、Recovery file-lock 实现、integration build tag、lifecycle/production composition 的消费者约束。

## 3. Findings

### HIGH — `ARTICLE-S4.6-PLAN-R01-F001`：冻结的 11 文件没有正常、可维护的可靠 Unix writer-lock 实现路径

- **位置**
  - Tasks：`article-management-S4.6` 的精确 11 文件、禁止第 12 文件/修改共享实现的停止条件。
  - Plan：`B6A-W` 的 11 文件与只读/禁止范围。
  - design r10：`.writer.lock`、进程级排他锁、owner 关闭释放；不支持可靠 file/directory fsync/lock 时启动 fail closed。
  - test-plan r06：A-WIRE-JRN-04/07；后继 S4.3 必须验证受支持 Unix 上的真实 `flock`。
- **违反依据**：规划必须给实施者提供能满足上游合同的明确写集和所有权边界；不能依赖越界修改、弱化安全合同或把无关文件重排为平台适配器。
- **证据**
  - 现有可靠实现分布在 `internal/recovery/filelock_flock.go` 与 `filelock_unsupported.go`，通过 build tags 和 `golang.org/x/sys/unix.Flock` 实现；helper 未导出且 `internal/recovery/**` 不在 S4.6 写集。
  - Article 当前跨平台 `deletion_journal.go` 只有 runtime 平台白名单和进程内 mutex，没有跨进程 writer lock。
  - 在单一跨平台文件直接引入 Unix API 会破坏不支持平台的编译；仅用 `O_EXCL` 锁文件又不能提供 test-plan 指定的真实 Unix `flock` 证据。
- **触发场景**：Backend-Dev 实现 production Journal provider 时，需要加入支持 Unix 的真实锁，同时保持 Windows/不支持平台可编译并 fail closed。
- **影响**：实施者只能越出 11 文件、修改共享 Recovery 原语、采用低于设计合同的锁，或按任务停止条件中止；当前规划不能稳定完成 finding 要求的 production Journal owner/lifecycle。
- **修复建议**：由 Lead/设计/Task-Planner 决定并同步写集：允许 Article 增加成对的平台 lock 文件，或授权受控抽取/导出共享 file-safety primitive。Reviewer 不替代该架构与所有权决定。

### MEDIUM — `ARTICLE-S4.6-PLAN-R01-F002`：验收命令不编译第 11 个 owner 文件

- **位置**
  - `internal/article/repository_integration_test.go:1`
  - Tasks S4.6 验收命令；Plan `B6A-W` 验证命令；test-plan r06 §9.1。
- **违反依据**：任务关闭命令必须覆盖精确写集的可编译性；格式检查不能替代带 build tag 测试文件的类型/API 检查。
- **证据**：该文件声明 `//go:build integration`。当前普通 `go test`、race、vet 和 build 均不包含它；`gofmt` 只能发现语法/格式问题。因此该文件即使存在类型错误或与 `repository.go` API 漂移，现有关闭命令仍可能全部通过。
- **触发场景**：S4.6 修改 repository API 与 integration fixture，但不提供真实 PostgreSQL DSN。
- **影响**：任务可能在一个 owner 文件未编译的情况下被错误关闭，缺陷推迟到 S4.3 或后续真实环境才暴露。
- **修复建议**：增加不运行真实数据库用例的 integration-tag 编译门禁，例如经项目确认的 `go test -tags=integration ./internal/article -run '^$' -count=1` 或等价 compile-only 检查；真实 PostgreSQL执行仍归 S4.3。

CRITICAL：无。除上述两项外，未发现新的直接状态、统计、批次分配或 DAG 缺陷。

## 4. 待决定与非阻断建议

### 待决定

1. 选择 Article 平台锁文件扩界，或共享 file-safety primitive 的受控复用方向，并更新精确写集、所有权及停止条件。

### 非阻断建议

1. 整改时明确区分：S4.6 负责所有平台编译、组件/fake 闭环；S4.3 负责受支持 Unix 持久卷上的真实 `flock`/fsync/崩溃证据。

## 5. 独立验证

- HEAD、固定对象 SHA256/blob：开始及落报告前一致。
- 静态解析：53 task；20/2/31；模块计数 1/31/21；requires 全部指向现存 task；DAG 无环。
- 批次外层表：53 个 task 各出现一次；S4.6 仅位于 B6A-W。
- S4.6 requires：仅 S5.2、S4.1；S4.3/S4.5/S1.4 均保留原 requires 并新增 S4.6。
- `git diff --check --` 两固定对象：PASS，无输出。
- 未读取或输出 `HAN_TEST_POSTGRES_DSN`。

## 6. 未验证限制

- 按授权未运行 Go、PostgreSQL、部署、浏览器、Host、E2E、容量或真实 Unix 持久卷测试。
- 本报告不证明 S4.6 已实施，也不评价尚不存在的 worker 代码质量。
- 未关闭 `B6A-G4-R01-F001`，未关闭 B6A，未启动 S4.6、B6A-W 或任何后继任务。

## 7. Gate 与复审交接

**Planning Review Gate: FAIL**

当前 planning **不足以作为关闭型输入交原 Reviewer Faraday 对 `B6A-G4-R01-F001` 做定点复审**。应先整改 F001/F002，固定新修订及相关回归证据，再交 Faraday 核对原 finding；是否接受风险、关闭 finding 或关闭 B6A 仍由 Lead/用户决定。

## 8. 实际改动

仅新增本独立报告：`.spec/wip/article-management/review-tasks-plan-s4.6-r01.md`。未修改被审 Tasks/Plan、设计、test-plan、代码、ADR 或其他报告；未 stage/commit/push/切分支/清理。
