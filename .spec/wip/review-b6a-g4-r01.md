# B6A G4-batch 整体代码评审 r01

- Reviewer：Faraday（独立非作者）
- 日期：2026-10-23
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 批次：B6A「管理员创建与文章永久删除核心」
- **B6A G4-batch Gate：FAIL**

本结论不是把两个 task-level PASS 相加。两张任务卡的既有代码 finding 继续保持 CLOSED，但批次级消费者与后继责任核对发现 1 项 MEDIUM 未关闭 finding。

## 1. 固定对象与漂移核对

报告路径在开始评审及落报告前均不存在。开始与落报告前分别核对 branch、HEAD、git blob、SHA256；下列对象均与委托值一致，无混合版本：

### 1.1 权威输入

| 对象 | git blob | SHA256 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `f7385fae129aa7a40cfa288204e1f75c88eeeb53` | `5C54C8D9B4DFBD460A902BC3B389E2BE8882605A183062BF09959E59AD556B06` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `5164fdf79044280210057969e79cb1c3db19222c` | `7346408B79988EAF08EB4615EF3595331A5F2E14CF64D137AB0618C9895AA998` |
| `.spec/wip/identity-auth/review-code-s5.2-r04.md` | `e473153c726ee499254bd90c7f42df1cc82a8a97` | `B9C1598C2B8CAC42F055E355A6D95F28FBA5B8F459FB94495D72674F9945E075` |
| `.spec/wip/article-management/review-code-s4.1-r03.md` | `41e2c84b67804bf99acd622b876aa7464339575e` | `9FE6E887F2D90192CA20DC2FA602DFF1DCF5980EE9107DDE71430315D5401BBC` |

### 1.2 Identity S5.2 固定对象

| 文件 | git blob | SHA256 |
|---|---|---|
| `internal/identity/administrator_create.go` | `47fa2b8894dd588921fc9b5eacd49defe6de8221` | `DD1D1DA23CB9D9E13E3200E09867B2A2C2F6BCBF06894C46074F60AC94D75813` |
| `internal/identity/administrator_create_test.go` | `1ed2dc4abb6c583fe7bacc6ef66531586d2dd07b` | `26E059CD710160FFE7E6F6DCE793859DA478689D0038641EFDC2E5D335463B34` |
| `migrations/0007_idempotency_response_snapshot_v1.sql` | `92c5a163e84d58ec56f28082b86bbc4766309be1` | `9D3E642D4F5C5DE81113E23AEBD70E2E07BF583E140697D335DB4DDA35F01F9F` |
| `internal/idempotency/store.go` | `e221eaa45698480b670cca037a7e0d3b4cd2fdfc` | `E9AAE069942F310EB155FBCAE3E981B55C87ABCAFCF3ACA74730292483C57FA5` |
| `internal/idempotency/store_test.go` | `2ba775052044541569716efedb5c60fce4e4eec9` | `DB291F9AE7F501F60ABF71F715FC2DD2961067E8598CED64348A7A6AC77DAC5C` |
| `internal/idempotency/store_integration_test.go` | `813bf5700a792b28e5a60f2ec4a666f791d637d7` | `BAE146223ABADF5E3884962E2BDD119AE46964D694B3BE4A38B331DAA956CD5A` |
| `internal/identity/handler.go` | `685148bcafc803b701a083e15cb684f5a3f0b752` | `F21E1E9B824DADCD784AA89E4A9865CDD85BF17A372B4D9C2964A91250C456D9` |
| `internal/identity/handler_test.go` | `c33eec776c607cd1b9f1b78b13c0a0ee65471f6d` | `7A32FEF8C9C94489D8D8E055C6B5ACEB7E2EA0304BCF028BDE0CF1D0BCA6373F` |
| `cmd/han-server/main.go` | `afa2e52e924bf01c698a6a6510f44a59ec8cbdc0` | `FDED5C74F9BB06D3BACD6E3B349D05ABF59B75C93C74C4EDA59D28EE0E9ABA30` |
| `cmd/han-server/main_test.go` | `f16d31e4922b0ef8e25bb4f83df7301b214f4819` | `2B5336CF9994F795CB01928BB07CDAE3D47C0FED4E0D0AF07E66BAD9E3442884` |
| `internal/mailer/smtp.go` | `62c984be0efc708f2ee3578f93231ecf6cbbbf11` | `EA42BC63AFE1E8577E4F5C7C01FDCBF30C5A00496D8ED53051A199234CF633D8` |
| `internal/mailer/smtp_test.go` | `812ec8e465829b86a0f0c2f2b0bdeb6523b148b1` | `A4B7784F224800F6476C585526E7ACAB1D87E5882427799155BA2EE35B6D5364` |

### 1.3 Article S4.1 固定对象

| 文件 | git blob | SHA256 |
|---|---|---|
| `internal/article/deletion_coordinator.go` | `fe3d2b2280590da5304a9977af3f08835ecfebcf` | `51B720A6ACFBDE81966AA959BF38898C68EB61622D7D428D3344DEE2D4CA0D1D` |
| `internal/article/deletion_finalizer.go` | `3e618fbeda01a894acf3db1b2164ae140da2ef66` | `9ABEEE4D1314F52E44B25553C19B3E9989FFC396C738ED43375C3AA5CC319B17` |
| `internal/article/deletion_journal.go` | `3489d2f2c594af2b257eda19a04f5db3610f91a9` | `E2539017B9C88EC6C60E3BBC0DA6DFF581EE885AE337E7B17E368129F7193343` |
| `internal/article/command_service.go` | `9097af708b06196f9c31c04e643ccb4c7a054191` | `58A1FFDFD19C7C7214C07C28F90CF99F50A172B9AD054787A16C90BF08CC0603` |
| `internal/article/handler.go` | `2dc8122cc33b8be12270fe733922a5f543b8a412` | `874A2BAE6AE69C1C96A888E80AC1E35FE94452398B2CABE5F38C81696CE60F0A` |
| `internal/article/deletion_test.go` | `1637fe9b46e1c523edf0dd923f7ebe6a5d6fbd04` | `7D8841D39647EB0AE22E051B89FDE8C30917CF1732EEC6B7F337E5D4517F297B` |

## 2. 两轮实际评审范围

### Round 1：两张固定任务对象

- Identity：复核 strict legacy/v1、`0007` CHECK、稳定成功/失败快照、当前请求 trace、D001 顺序、SMTP deadline/cancel/失败收敛、启动配置与三条 production route。
- Article：复核 T1 隐藏、Journal 发布/验链/fsync、T2 finalizer、lease/并发收敛、三态物理删除、事务内审计、公共错误映射与固定测试。
- 两卡固定写集互斥；未发现一方覆盖另一方代码或公共契约。
- Identity r04 的 `IDENTITY-S5.2-R01-F001/F002/F003/D001`、`R02-F001`、`R03-F001` 保持 **CLOSED**。
- Article r03 的 `ARTICLE-S4.1-R01-F001`～`F005` 保持 **CLOSED**。

### Round 2：批次级接缝

- Migration runner 按数值升序加载；article schema `0003`、idempotency `0004`、recovery `0005`、user indexes `0006`、snapshot `0007` 顺序一致，`0007` 未修改 `0004`。
- Identity production composition 继续复用单一 `newProductionBusinessHandler`、Session、Audit、ProcessingGate、Mailer、keyring 和 lifecycle；没有 Article 代码反向污染该接线。
- Article 删除审计继续使用共享 `audit.Event`/`audit.Store`，action、actor、target、trace 与最小 metadata 未和 Identity 新增审计事件冲突。
- 检查 Article Journal/Coordinator/Finalizer 的真实生产消费者、配置、router、worker 和后继任务所有权；形成下列批次级 finding。

## 3. Findings

### MEDIUM

#### `B6A-G4-R01-F001` — Article 永久删除 production composition/router/worker 没有已登记后继 owner

- **位置/证据**
  - `cmd/han-server/main.go:1444-1629` 的唯一 production composition 只构造 Identity 服务并返回 `newLoginBusinessRouter`；该文件没有 `internal/article` import 或 Article 构造/路由引用。
  - `tasks.md:1045-1067` 明确把 Article production composition/router 与后台 worker 列为 S4.1 未验证、未接线范围。
  - 后继 `article-management-S4.3` 仅拥有三个 `test/integration/article` 测试文件；`S5.1`、`S6.2`、`S8.1` 只拥有 `internal/article` 实现/测试文件；`S4.5` 只拥有 runbook/testdata。它们均不拥有 `cmd/han-server`、Article 启动配置或 worker lifecycle。
  - Plan `:68` 同时声明“未安排范围：无”，与上述生产接线没有任务 owner 的事实不一致。
- **违反依据**：`add-batch-plan` Step 5 要求批次整体核对跨任务依赖、接口、适用集成/覆盖，并把测试缺口交给明确责任；不能只保留“后续验证”而没有可实施 owner。
- **触发场景**：后续 S4.3、S5.1/S6.2/S8.1 或最终 E2E 尝试从真实 server 调用永久 DELETE、启动 finalizer worker，或注入 Journal directory/keyring/lease 配置。
- **影响**
  1. 当前 Article 永久删除核心在生产进程不可达，且没有登记任务能合法完成接线。
  2. S4.3 的真实 PostgreSQL/持久卷/双 worker 验证缺少与生产装配一致的被测入口。
  3. 未来临时接线容易另建 router/composition 或绕开现有 Session/Origin/CSRF/Principal/Trace/lifecycle，形成跨模块契约回归。
- **最小修复方向**：由 Task-Planner/Lead 明确一个现有或新增后继 task，登记唯一 production composition/router、Article Journal/keyring/目录/lease 配置、finalizer worker lifecycle、失败 unwind、资源关闭及对应测试的精确 owner、写集与 requires；不得由 Reviewer替用户选择任务或修改已确认公共契约。
- **状态**：**OPEN**。这在 S4.1 task-level 可作为已知限制，但因没有后继 owner，不能在 G4-batch 层仅作为“后续会验证”的非阻断限制。

CRITICAL / HIGH / LOW：无新增 finding。

## 4. 待决定与非阻断建议

### 待决定

1. 由 Lead/Task-Planner 决定把 `B6A-G4-R01-F001` 的 production wiring/worker 责任并入哪个既有后继 task，或登记新的有界 task。Reviewer 不替代该规划决定。

### 非阻断建议

1. 后继测试应从唯一 server composition 证明 Article 路由与现有 Identity 路由共存，覆盖 exact path/method、JSON 404/405/`Allow`、Trace → router → Origin → Session → CSRF → Principal → Handler 顺序、共享实例、worker 正常关闭与构造失败逆序 unwind。

## 5. Reviewer 独立验证

| 命令 / 检查 | 结果 |
|---|---|
| branch、HEAD、22 个固定权威/代码对象的 `git hash-object` 与 SHA256，开始及落报告前核对 | PASS；全部匹配 |
| `go test ./internal/identity ./internal/idempotency ./internal/mailer ./internal/article ./cmd/han-server -count=1` | PASS |
| `go test -race ./internal/identity ./internal/idempotency ./internal/mailer ./internal/article ./cmd/han-server -count=1` | PASS |
| `go vet ./internal/identity ./internal/idempotency ./internal/mailer ./internal/article ./cmd/han-server` | PASS |
| `go test ./internal/article -run 'TestPermanentDeletion\|TestDeletionJournal\|TestDeletionFinalizer' -count=1` | PASS |
| `go test ./internal/article -run '^TestDeletionJournal_DirectoryFsyncFailureBlocksIndependentFinalizerUntilProof$' -count=100` | PASS |
| `go test ./internal/article -run '^TestDeletionFinalizer_ConcurrentPendingReconciliationCompletesOnce$' -count=1000` | PASS |
| Identity stable replay/deadline/cancel/reconciliation targeted tests | PASS |
| Mailer cancellation watcher targeted tests | PASS |
| `cmd/han-server` administrator config/router/effective-at/lifecycle targeted tests | PASS |
| `gofmt -l` 对 17 个固定 Go 文件 | PASS；无输出 |
| tasks/Plan `git diff --check`；18 个固定代码/SQL对象逐文件 no-index whitespace check | PASS；无诊断 |
| `rg` 检查 `cmd/han-server` 的 Article import/构造/route | 无匹配；确认当前无生产接线 |

Lead 提供的 Manifest 7/7、Identity PostgreSQL 18 integration、Article 专用/高重复验证作为外部证据引用；未冒充 Reviewer 独立执行。

## 6. 实际读取范围

- Role / 方法 /规则：
  - `.codex/agents/add-reviewer.toml`
  - `.codex/skills/add-code-review/SKILL.md`
  - `.codex/skills/add-code-review/references/产品安全红线.md`
  - `.codex/skills/add-batch-plan/SKILL.md` Step 5
  - `.codex/guidance/code-review-checklist.md`
  - `.codex/guidance/abstraction-principles.md`
  - `.codex/guidance/coding-standards.md`
  - `.codex/guidance/no-hardcode-policy.md`
  - `.codex/guidance/observability-standards.md`
  - `.codex/guidance/tracing-standards.md`
  - `.codex/profiles/go-service/README.md` 及其 coding/no-hardcode/security/observability/tracing rules
- 权威与历史：
  - 本报告第 1.1 节四个固定输入
  - Identity design r33、test-plan r13、design review r05
  - Article design/test-plan、ADR-0020
- 固定代码：
  - 本报告第 1.2、1.3 节全部文件
- 有界直接接缝：
  - `migrations/0003_article.sql`、`0004_idempotency.sql`、`0005_recovery_runs.sql`、`0006_user_governance_indexes.sql`
  - `internal/migration/runner.go`
  - `internal/audit/store.go`
  - Article model/repository/policy/article port
  - Identity production composition/router、platform error/trace/config/lifecycle、session/security/privacy 直接消费者
  - tasks/Plan 中 S4.3、S4.5、S5.1、S6.2、S7.1/S7.2、S8.1 与最终 E2E 的后继边界

未扩成无界全仓审查。

## 7. 未验证范围与限制

1. 未读取、打印或持久化 User scope `HAN_TEST_POSTGRES_DSN`；未独立运行需该 DSN 的 integration。
2. 未连接真实 SMTP/TLS，未执行真实 DNS/TCP/TLS/AUTH 或外部邮件。
3. 未验证 Article 真实 PostgreSQL 事务/锁/lease/回滚/commit outcome unknown。
4. 未在受支持 Unix 目标持久卷验证 directory fsync、进程崩溃、断电、磁盘满或磁盘故障；Windows 结果只证明 fail closed。
5. Article production composition/router/worker 尚不存在，无法进行真实 server/worker 集成验证。
6. 未执行真实 Host、部署、浏览器/E2E、容量或目标环境 benchmark；本报告不宣称 production readiness。

## 8. Gate 与复审条件

### B6A G4-batch Gate：FAIL

两张 task 的代码级 finding 均保持 CLOSED，固定代码未发现新的实现缺陷；但 `B6A-G4-R01-F001` 仍 OPEN。关闭条件：

1. Task-Planner/Lead 为 Article production composition/router/config/worker 明确后继 owner、写集、requires 与验证责任；
2. 若只修 planning，定点复审该 planning 修订能否消除所有权/依赖矛盾；
3. 后续实施完成时，对唯一 production composition、route/middleware、Journal/worker lifecycle 与失败收敛另做代码独立评审和适用集成验证。

该 FAIL 不回退 S5.2 与 Article S4.1 已完成的 task-level CLOSED；它阻止的是把 B6A 作为完成了跨任务交接的整体批次关闭。

## 9. 实际改动

仅新增本报告：`.spec/wip/review-b6a-g4-r01.md`。未修改被审代码、tasks、Plan、设计、ADR、测试规划或其他报告；未 stage/commit/push/切分支/清理。
