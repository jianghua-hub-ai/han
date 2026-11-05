# Article S4.6 Tasks/Plan 定点复审 r02

- 日期：2026-10-23
- Reviewer：Hypatia / add-reviewer；原 r01 Reviewer，非本次 planning 修订作者；未委派
- 原作者：Task-Planner Hume `01a16a7b-9353-7141-866b-a0020768aa5b`
- objectType：design（planning 复合对象）
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`

## 1. 固定对象与复审依据

| 对象 | SHA256 | blob |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `C23A94E3BEF36C657D5473ECA91FBEC6199A18161FDB1CF9D0AD26252D83C981` | `51afea72e4303c0aa7189da986880aaecd3c023b` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `D8E1AE078A891267F38DE662DC2613828FCE675EE2E94998F92EC2F16E077A25` | `2e2a84fd48ac10fe6984be638a85fed85c3fcc55` |

复审依据：

- 原 r01：`.spec/wip/article-management/review-tasks-plan-s4.6-r01.md`，SHA256 `0AF25106B09D3D5E2FB42E6BA04BDB85C7A92F1E7CF7AB8A3CF342CE9350F65B`
- Article test-plan r08：SHA256 `4608502D4B4E9078509651A52589245D6726634053FF8A5E787569C55776EA7E`
- Rawls r04：`.spec/wip/article-management/review-test-plan-r08-planning-findings-r04.md`，SHA256 `832ED2262DBF443CA3A172C88E5A904A206296771CA309A44EE68D111909F6EA`，Test-plan Gate PASS
- Article design r10、stories r07 与 `.spec/wip/review-b6a-g4-r01.md` 中 `B6A-G4-R01-F001`

评审开始及落报告前，HEAD、两份固定对象的 SHA256/blob 与上述三份复审依据哈希均匹配委托值；报告路径此前不存在。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

- 核对 Tasks `article-management-S4.6` 与 Plan `B6A-W` 是否同步为原 11 文件加两个 Article 私有 build-tag 文件的精确 13 文件合同。
- 核对第 14 个文件、migration、新依赖、第二 composition/router、Recovery 私有锁原语及公共合同的禁止边界。
- 将 Tasks/Plan 的双 GOOS PowerShell 命令与 test-plan r08 逐字符对照，并使用 PowerShell AST 静态解析。
- 核对 integration-tag Gate 是否编译 `repository_integration_test.go` 且不宣称真实 PostgreSQL 已验证。
- 核对 S4.6 状态、owner、requires、关闭条件、后继责任及旧 11 文件历史记录。

### Round 2：跨 Tasks/Plan、设计/测试、消费者与 DAG

- 静态解析全部 53 个 task 的 checkbox、模块、requires 与批次外层表。
- 核对 53 个 task 各安排一次、requires 全部指向现存 task、DAG 无环及 B6A-W 仅含 S4.6。
- 对读 design r10、test-plan r08、Rawls r04 与原 `B6A-G4-R01-F001`，检查 13 文件是否覆盖唯一 production composition/router、Article Journal/keyring/目录/lease、worker lifecycle、失败 unwind、资源关闭和测试责任。
- 有界核实现事实：现有唯一 `cmd/han-server` composition 入口、Article deletion schema/lease、现有依赖、integration build tag，以及 13 文件当前存在性；未扩展为代码质量评审。
- 核对 S4.6 组件/fake/编译与 S4.3/S4.5/S1.4 的真实 PostgreSQL、Unix 持久卷、运行输入和部署后 E2E 分界。

## 3. 原 findings 复审

| Finding | 结论 | 证据 |
|---|---|---|
| `ARTICLE-S4.6-PLAN-R01-F001` | **CLOSED** | Tasks `1156–1185` 与 Plan `602–630` 均冻结精确 13 文件；新增且仅新增 `deletion_filelock_flock.go`、`deletion_filelock_unsupported.go`，并禁止第 14 文件、migration、新依赖、第二 composition/router、修改/导出/复制 Recovery 私有锁原语或公共合同。双 GOOS Gate 与 r08 完全一致，AST 0 错误，分别编译 linux/amd64 flock 与 windows/amd64 unsupported 分支，具备失败非零、唯一临时目录、`finally` 清理、父环境隔离且不执行生成物。 |
| `ARTICLE-S4.6-PLAN-R01-F002` | **CLOSED** | Tasks `1178–1179`、Plan `626–627` 与 test-plan r08 §9.1 均固定 `go test -tags=integration ./internal/article -run '^$' -count=1`；现有 `repository_integration_test.go` 带 `//go:build integration`，当前 article 包无 `TestMain`/`init` 启动数据库。文档明确该 Gate 只提供 integration-tag owner 文件编译覆盖，不代表真实 PostgreSQL、S4.3 或 production readiness。 |

### F001 详细判断

13 文件的责任映射足以承接原 `B6A-G4-R01-F001` 要求：

- `main.go` / `main_test.go`：唯一 production composition/router、Article 配置、Journal/keyring provider、三条 DELETE 与失败矩阵；
- `lifecycle.go` / `lifecycle_test.go`：worker、server、Journal/writer lock、key material 与 DB 的关闭顺序、构造失败逆序 unwind；
- `deletion_worker.go` / `_test.go`：discovery、claim/lease、即时首轮、ticker/wake、退避与 transient/no-op/terminal 收敛；
- `deletion_journal.go` 与两个 Article 私有 build-tag 文件：独立目录、writer lock、受支持 Unix flock 与 unsupported fail-closed；
- finalizer、repository 及对应测试：现有 operation/lease 数据模型上的收敛与局部回归。

现有 `go.mod` 已包含 `golang.org/x/sys`，现有 Article migration 已有 deletion operation、claim/lease 字段与索引；当前规划无需 migration 或新依赖。四个声明为新增的 worker/lock 文件目前确实不存在，不能据此误称已经实施。

S4.6 只关闭组件/fake、所有平台编译与局部自动化；真实 PostgreSQL discovery/claim/lease/事务/锁、真实 Unix 持久卷 flock/file/directory fsync/崩溃/跨进程竞争、运行输入、Host、部署、signal/exit E2E、容量和 production readiness 仍分别由 S4.3、S4.5、S1.4 等后继独立关闭。未发现遗漏或重复所有权。

### F002 详细判断

integration-tag 命令会选择并编译 `repository_integration_test.go`，`-run '^$'` 不运行现有 PostgreSQL 测试。当前包无 `TestMain` 或 `init` 数据库启动路径；因此本次 planning 对现状成立。后续实施仍须由代码 Reviewer/Lead 核对该命令没有被新增启动副作用破坏，且不得把它记成真实 PostgreSQL PASS。

## 4. 新 findings

未发现新的 CRITICAL、HIGH、MEDIUM 或 LOW finding。

## 5. 待决定与非阻断建议

- 待决定：无。
- 非阻断建议：无。实施阶段按已登记代码 Review/Lead Gate 核实实际 build tags、无测试启动副作用、临时生成物已删除及 13 文件无越界即可。

## 6. 独立验证

| 检查 | 结果 |
|---|---|
| task 实际统计 | 53 total / 20 completed / 2 active / 31 todo；article-management 21 |
| active 集合 | 仅 `identity-auth-S3.3`、`identity-auth-S3.5` |
| S4.6 状态 | `[ ]` todo；未认领、未启动 |
| S4.6 直接 requires | 仅 `identity-auth-S5.2`、`article-management-S4.1` |
| 三个后继 requires | S4.3、S4.5、S1.4 均新增 S4.6，同时保留各自其他真实 requires |
| 批次分配 | 53 个 task 各出现一次；无遗漏、无重复；B6A-W 仅含 S4.6 |
| DAG | requires 全部指向现存 task；0 cycle |
| B6B/B6C/B6D | 未建立对 S4.6 的业务硬依赖；仅共享当前 planning 复审流程 Gate |
| 13 文件一致性 | Tasks、Plan、test-plan r08 的顺序和内容一致 |
| PowerShell AST | 三处脚本完全一致；0 parse error；2 个 `go test -c`、2 个失败 `throw`、1 个 `try/finally`、1 次创建与 1 次递归清理；环境赋值为 CGO=0、linux/amd64、windows/amd64 |
| integration-tag Gate | Tasks/Plan/r08 命令一致；`repository_integration_test.go` build tag 为 integration |
| 旧 11 文件记录 | 仅 Plan 变更历史第 853 行保留，并明确“已由下一条 r08 同步取代，不作为当前写集、Gate 或阻塞合同”；第 854 行及当前合同均为 13 文件 |
| 状态与完成声明 | 未宣称 S4.6、production wiring、真实 PostgreSQL、Unix 持久卷、Host、部署、E2E、容量或 production readiness 已完成 |
| `git diff --check` | 两份固定对象 PASS，无输出 |

未读取或输出 `HAN_TEST_POSTGRES_DSN`。

## 7. 未验证限制

- 按授权未运行 Go、PostgreSQL、跨 GOOS 编译、Unix 持久卷、部署、Host、浏览器、E2E 或容量测试。
- PowerShell 与 integration-tag Gate 仅做静态解析和现状副作用检查；实际命令结果必须在 S4.6 实施后取得。
- 本报告评审 planning 可实施性，不证明 13 文件实现正确，也不替代后续代码 Review、Lead §10 核证或真实专项验证。
- 本报告不关闭 `B6A-G4-R01-F001`，不关闭 B6A，不启动 S4.6、B6A-W 或任何后续任务。

## 8. Gate 与交回

**Planning Review Gate: PASS**

`ARTICLE-S4.6-PLAN-R01-F001` 与 `ARTICLE-S4.6-PLAN-R01-F002` 在本次固定 Tasks/Plan 修订上均 **CLOSED**，无新增 finding。

该 planning 修订**可以交原 Reviewer Faraday**对 `B6A-G4-R01-F001` 做定点复审。Faraday/Lead 仍须独立决定该 finding 与 B6A Gate；本报告不代为关闭，也不构成 S4.6 启动授权。

## 9. 实际改动

仅新增本独立报告：

- `.spec/wip/article-management/review-tasks-plan-s4.6-r02.md`

未修改 Tasks、Plan、test-plan、design、stories、代码、ADR 或其他报告；未 stage/commit/push/切分支/清理。
