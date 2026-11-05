# article-management-S4.6 独立代码评审 r01

- Reviewer：Faraday（独立非作者 Reviewer）
- 评审日期：2026-10-23
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **Code Review Gate：FAIL**

## 1. 固定对象与漂移

评审开始时已核对分支、HEAD、报告路径不存在，以及下列 13 个未提交/未跟踪文件的 git blob 与 SHA256。落报告前已再次核对分支、HEAD 及全部固定对象，结果无漂移；本报告只绑定下列精确对象：

| 文件 | git blob | SHA256 |
|---|---|---|
| `cmd/han-server/main.go` | `2f58fc84f7ad3fa78c2c0806d719d7be6e62e38e` | `D56774321FB701B8D1D5113BDBF92D4F38985B1BBB1E72D031F0DA77C7E03468` |
| `cmd/han-server/main_test.go` | `ccdf2bea8e1233e3bc3e95e0fbf1da9070a9321f` | `354651A0CFEC2A7585C4C85E24B37AD66C1E1C96C1DD1E385DE868FB5DC7035C` |
| `cmd/han-server/lifecycle.go` | `9100ff4bba6ea1741b24754254e8b131fe1505c2` | `B6B246667F9279800B023EF67E8FB8D5EB8A2F4E56B6DA403C6D591E3B391816` |
| `cmd/han-server/lifecycle_test.go` | `36b1dcedb4a332c98a7db22d77f021fdaba9a773` | `9ABF3282664D63FA98C3303C31B8201330EE746266BEBEB3FCECF7A5598E81F6` |
| `internal/article/deletion_worker.go` | `a88339e76e14c80a1949ff205b11c1896227d6b5` | `CFACB6A8327A66BCCFD5E3050C0E4878D886DBC6DEC52317B543BC878C1CA90F` |
| `internal/article/deletion_worker_test.go` | `d04c28f331c2daf60b4d96a6924b36edc08b3eb0` | `3923C0FB4D8CF68A91ABA2F0335A90AB8EF7BA8F7ACC5D9D76B71243726E86E0` |
| `internal/article/deletion_finalizer.go` | `5e5f777ac7018ab7ffbfa9021090760bd04546c0` | `03C169DA70592F326A647746D269656ED3EC850E82B9D1C5916517CF90B73D53` |
| `internal/article/deletion_journal.go` | `ece5251f1fdb5df368077b3e1b3242156b57daa7` | `ECC805E82B842273E00B772647A9109466A8C4103DC484149CB4D22B0FC5B254` |
| `internal/article/deletion_test.go` | `4c3cc8689d314fbbb1bcf61e896b14ace76b064d` | `B1D59AC40D09A6426C25459F2966BB87349E29F9C9FF3A6454DA674BAD8FE2EA` |
| `internal/article/repository.go` | `41c18954de82ff9253f27ea0b2fffda08496263e` | `5A03F79D15706CB5CDD4F9A51129D388E192AF7967DD848C6585686FA511DDC7` |
| `internal/article/repository_integration_test.go` | `a5f44b8a9d98197729e7c2c6ab71ff533237cc76` | `EE9A68CCA141AC6F84C412C6BCE54457C543003ABB2812D4B8C3690D9B269355` |
| `internal/article/deletion_filelock_flock.go` | `15e3797683263735f187b3bb69b4892b89bafe99` | `5329CAB471A0516246118D2B47342C5F0F0252B567DED47C40F9BF1B17EC2A0E` |
| `internal/article/deletion_filelock_unsupported.go` | `ad26bfcdb0b422c8801774cfe8dbdcb5e9aa3b35` | `BF140294B11D068B2971DFEFF4D3598FFAEC591FCF1F5BE0435640370C5FEB8B` |

## 2. 实际读取与评审依据

完整读取并执行：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `code-review-checklist.md`、`abstraction-principles.md`、`coding-standards.md`、`no-hardcode-policy.md`
- 适用 observability、tracing、产品安全红线
- 已启用 `go-service` profile 的 README、coding、no-hardcode、security、observability、tracing 约束

上游与直接接缝读取范围：

- `tasks.md` 的完整 `article-management-S4.6` 条目、B6A-W Plan
- article design r10、stories r07、test-plan r08 的 §4.10.2、§9.1 与 A-WIRE-CFG/JRN/RTR/WKR/LIFE
- 13 个固定文件
- `deletion_coordinator.go`、`command_service.go`、`handler.go`、`model.go`
- migration `0003_article.sql`
- session authorizer、audit、platform trace/error、`internal/recovery` 私有锁实现的直接接缝

未扩展为全仓审计；未读取 `HAN_TEST_POSTGRES_DSN`。

## 3. 两轮范围

### Round 1：固定对象直接审查

核对配置解析与副作用时序、keyring/secret 清理、Journal 路径与锁、worker 状态机、repository discovery、三条 DELETE 路由、共享 composition、lifecycle ownership/unwind，以及固定测试是否真实约束上述行为。

### Round 2：有界消费者深审

沿唯一 `newProductionBusinessHandler` / `newLoginBusinessRouter`、Article Handler → CommandService → Coordinator → Journal/Finalizer、PostgreSQL discovery、session/audit/platform error/trace、worker result → lifecycle shutdown，以及 Recovery 锁隔离进行核对。未发现第二 production composition/router、migration、新依赖或第 14 个 S4.6 写文件。

## 4. Findings

### HIGH — `ARTICLE-S4.6-R01-F001`：Journal root 未逐路径分量拒绝 symlink

**触发**

将 `HAN_ARTICLE_DELETION_JOURNAL_ROOT` 配为绝对且 clean 的路径；最终分量是 owner-only 的真实目录，但任一祖先分量是 symlink。`ValidateArticleDeletionJournalRoot` 只对完整 root 执行一次 `os.Lstat`，随后 `ensureArticleDeletionDirectory` 只检查应用创建的三个后代；祖先 symlink 已在路径解析时被跟随。

**证据**

- design r10 要求 root 为 owner-controlled/owner-only，并“以 `Lstat` 逐段拒绝 symlink”（`design.md:757,772-783`）。
- A-WIRE-JRN-02 要求 root 或任一既存路径对象的 symlink/逃逸输入 fail closed（`test-plan.md:285`）。
- 实现仅检查最终 root（`deletion_journal.go:186-201`），后续直接以该字符串拼接并创建固定后代（`:204-222`）。
- 现有 provider 测试只使用 `t.TempDir()` 的普通路径，没有祖先 symlink/路径逃逸负例（`deletion_test.go:445-496`）。

**影响**

生产 Journal、writer lock 和 tombstone 可落入配置文本所指持久卷之外的位置，破坏 Article/Recovery 隔离、耐久性与永久删除证据的 fail-closed 假设。该问题发生在受支持 Unix 路径上，Windows fail-closed 与双平台 compile-only 不能证明此行为安全。

**最小复审条件**

在既定 13 文件内对 root 到固定后代的每个既存路径分量执行不可跟随的校验并拒绝 symlink/逃逸，补确定性祖先 symlink、直接 symlink、错误对象及权限负例；不得改公共契约、引入第 14 文件或复制 Recovery 私有锁原语。

### HIGH — `ARTICLE-S4.6-R01-F002`：任务关闭所需的 P0 确定性测试矩阵未完成

**触发**

当前测试全绿，但没有实现 task 明确要求的 A-WIRE-CFG-01～06、JRN-01～07、RTR-01～06、WKR-01～10、LIFE-01～08 的局部自动化闭环。

**证据**

- task 关闭条件明确要求上述矩阵全部完成（`tasks.md:1185`）；test-plan r08 在 `:271-328` 给出逐项可观察断言。
- `deletion_worker_test.go` 只有 4 个顶层测试（`:12,65,140,207`），未覆盖 discovery 状态/排序/limit、即时调谐 transient/terminal、full-jitter/max batches、逐类 transient/no-op/terminal、NewTraceID 失败、operation budget、lease competition、runtime terminal → 双 server shutdown 等合同。
- `lifecycle_test.go` 只有 7 个顶层测试，未覆盖每个构造点的故障注入、所有资源严格 reverse unwind、多个 trigger 竞争/one-shot/逐资源恰好关闭一次及 worker/global 较早 deadline 的双向组合。
- Article router 测试仅对一条路由抽样错误方法，并只证明正确 DELETE 先到 Origin；未对三条路由的 GET/POST/PUT/PATCH、每层拒绝后的零后续调用、完整 Trace→Origin→Session→CSRF→Principal/授权→Handler 和共享实例做矩阵证明（`main_test.go:1652-1798`）。
- Article 配置测试只抽样 7 个非法 case；没有逐键 missing/empty/blank、每项上下界/越界、每类配置失败的零 DB/listener/composition 调用与 secret/path/DSN/PII canary 扫描（`main_test.go:1800-1857`）。
- integration-tag discovery 测试只校验 SQL 字符串片段和 nil pool，没有执行确定性 repository contract；真实 PostgreSQL 留给后继专项是正确边界，但不能替代本卡要求的 fake/局部证明（`repository_integration_test.go:31-45`）。

**影响**

配置 fail-closed、路由安全链、worker 分类与收敛、资源回收等 P0 合同可在当前 suite 全绿时回归；因此现有测试结果不能满足 S4.6 自身关闭条件，也不能支持代码 Review PASS。

**最小复审条件**

只在固定 13 文件内补齐 test-plan 指定的确定性部分，并明确把真实 PostgreSQL、真实 Unix 持久卷/flock/fsync/崩溃/跨进程竞争、Host/部署/E2E 留给后继任务。测试需能使相应旧/错误实现失败，而非只断言构造成功或源码包含字符串。

### MEDIUM — `ARTICLE-S4.6-R01-F003`：活动 lease 竞争被当作 no-op，未进入规定的 transient 退避

**触发**

operation 已被另一调用方持有未过期 lease 时，`ClaimDeletionOperation` 返回 `claimed=false`；`Finalize` 将其统一映射为 `ErrDeletionInProgress`。worker 对该错误显式不设置 `transient`。当发现批次小于 batch size 时，本次 wake 立即返回，不执行 full-jitter backoff，只能等待后续 ticker/wake。

**证据**

- 活动 lease 返回未 claim（`deletion_finalizer.go:470-475`），随后变为 `ErrDeletionInProgress`（`:133-139`）。
- worker 将 `ErrDeletionInProgress` 排除在 transient 之外（`deletion_worker.go:148-153`），并可在短批次直接结束（`:158-166`）。
- design r10 明确把 lease competition 归为 transient 并要求低敏 bucket + backoff；completed/not-claimed 才是 no-op（`design.md:1821-1823`）。
- 当前 worker 测试没有 lease competition/no-op 分类用例。

**影响**

请求路径与后台 worker 竞争同一 operation 时，后台收敛时延和退避行为偏离固定合同；分类证据也不可观察。数据不会因此恢复公开，但故障恢复可能被无声推迟到下一 poll。

**最小复审条件**

在不改变公共错误合同的前提下区分活动 lease competition 与真正 completed/not-claimed，并为两类结果分别证明 transient backoff 与 no-op 行为。

### MEDIUM — `ARTICLE-S4.6-R01-F004`：Journal 文件系统失败可把绝对路径带入错误链

**触发**

合法 root 在创建固定后代、同步目录、chmod、创建临时记录或清理文件时发生失败。多个分支直接 `%w` 包装 `*os.PathError` 或直接返回文件错误；该错误继续由 `runComposed` 返回。

**证据**

- 路径可从目录创建/同步错误进入错误文本（`deletion_journal.go:204-220`），运行期的创建/同步/删除也直接包装底层文件错误（`:420-476`）。
- flock chmod/close 错误也未做稳定类别映射（`deletion_filelock_flock.go:31-48,53-60`）。
- provider 错误由 startup 直接向上传播（`main.go:395-404`）。
- design 与 A-WIRE-CFG-06/JRN-07 禁止错误、日志或 lifecycle failure 输出 Journal 绝对路径（`design.md:753`；`test-plan.md:276,290`）。
- 当前 `main()` 只输出稳定的 `han-server startup failed`，降低了现入口的直接暴露面，但 `run` 返回值和内部错误链仍违反固定合同，且没有 canary 扫描测试。

**影响**

调用方、测试框架或后续结构化日志若记录错误链，可泄露部署挂载路径；同时错误输出不满足既定安全验收。

**最小复审条件**

在 Article Journal/production composition 边界将文件系统失败映射为不含路径、key、record body 的稳定类别，并增加 startup、operation、close failure 的 canary 扫描；保留内部可判定的 sentinel/类别，不打印敏感原值。

## 5. 待决定与非阻断建议

### 待决定

无。上述问题均可在已确认设计、固定 13 文件和既有公共契约内修复；本轮没有提出新架构、migration、依赖或公共 REST 决策。

### 非阻断建议

- 测试名或子测试名直接带 A-WIRE ID，便于 Reviewer 逐项核验而不依赖源码推断。
- 将 worker 的低敏分类观察收敛为一个私有 observer/recording seam，避免在错误或日志中携带 operation ID、路径或原始依赖错误。

## 6. 测试计划风险映射

| 范围 | 当前静态/执行结论 |
|---|---|
| A-WIRE-CFG-01～06 | **部分覆盖**；基础成功配置和少数非法/交叉项存在，逐键/全边界/零副作用/canary 不足 |
| A-WIRE-JRN-01～07 | **部分覆盖且有 F001/F004**；durability、HMAC、并发 writer、Windows fail-closed 有既有测试，生产 root 逐段 symlink 与错误去敏未满足 |
| A-WIRE-RTR-01～06 | **部分覆盖**；三条成功路由存在，完整错误方法矩阵、安全链逐层拒绝与共享实例证明不足 |
| A-WIRE-WKR-01～10 | **明显不足且有 F003**；基础批次、wake、drain、global cancel 存在，大部分 discovery/分类/backoff/lifecycle 联动缺失 |
| A-WIRE-LIFE-01～08 | **部分覆盖**；双 server shutdown、部分顺序和初始 reconcile failure 存在，逐构造点 unwind、竞争 one-shot、完整 deadline 矩阵不足 |

## 7. Reviewer 独立命令与结果

环境：`go1.27.0 windows/amd64`，`CGO_ENABLED=1`。

```text
git branch --show-current
git rev-parse HEAD
Test-Path .spec/wip/article-management/review-code-s4.6-r01.md
Get-FileHash -Algorithm SHA256 <13 files>
git hash-object -- <13 files>
git status --short -- <13 files> <report>
```

结果：branch/HEAD 匹配；报告创建前不存在；13 个对象开始哈希全部匹配；13 个对象均为未跟踪文件。

```text
go test ./internal/article -run 'TestDeletionWorker|TestDeletionJournal|TestDeletionFinalizer' -count=1
go test ./cmd/han-server -run 'TestArticle|TestLoadStartupConfig_Article|TestRunLifecycle' -count=1
go test ./internal/article ./cmd/han-server -count=1
go test -race ./internal/article ./cmd/han-server -count=1
go vet ./internal/article ./cmd/han-server
go build -o NUL ./cmd/han-server
go test -tags=integration ./internal/article -run '^$' -count=1
go mod verify
```

结果：全部 PASS；integration-tag 命令显示 `[no tests to run]`，只作为编译证据，不冒充 PostgreSQL 集成测试。

```text
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -o NUL ./internal/article
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go test -c -o NUL ./internal/article
```

结果：linux/amd64 与 windows/amd64 compile-only 均 PASS，未执行生成物。先尝试的临时目录/`finally` 清理脚本在命令创建前被本地策略拒绝，未产生文件；随后使用 Lead 已验证的 `-o NUL` 等价 compile-only 方法。

```text
gofmt -l <13 fixed Go files>
git diff --check -- <13 fixed files> .spec/wip/article-management/review-code-s4.6-r01.md
PowerShell trailing-whitespace scan <13 fixed files> <report>
```

结果：`gofmt -l` 无输出；包含新报告的限定 `git diff --check` 无输出；因固定文件与报告均未跟踪，另行扫描确认 trailing whitespace 为 0。

Lead 提供的 Manifest、PG18 及其他执行结果只作为外部证据，未冒充 Reviewer 本轮独立命令。

## 8. 未验证限制

- 未连接真实 PostgreSQL；未验证真实事务、锁、lease、commit unknown 或双连接竞争。
- 未在受支持 Unix 目标持久卷执行真实 `flock`、file/directory fsync、崩溃、断电、磁盘/权限故障或跨进程竞争；Windows 只证明 production fail closed 与 compile-only。
- 未运行真实 Host、部署、signal/exit E2E、浏览器或外部服务。
- 未执行容量、吞吐、积压、目标环境 benchmark；不得外推 production readiness。
- 未读取、打印或持久化 `HAN_TEST_POSTGRES_DSN`。

## 9. Gate 与复审条件

存在 2 个 HIGH 与 2 个 MEDIUM 未关闭 finding：

- `ARTICLE-S4.6-R01-F001` OPEN
- `ARTICLE-S4.6-R01-F002` OPEN
- `ARTICLE-S4.6-R01-F003` OPEN
- `ARTICLE-S4.6-R01-F004` OPEN

**Code Review Gate：FAIL**

复审必须固定修订后的 13 文件，逐项关闭 F001～F004，重跑对应定向测试、普通测试、race、vet、双 GOOS compile-only、integration-tag compile-only、gofmt 与限定 whitespace/diff 检查。若修复需要第 14 文件、migration、新依赖、第二 composition/router、Recovery 私有原语变更或公共契约变化，应停止并返回上游对齐。

本报告不修改 task 状态，不表示真实 PostgreSQL、Unix 持久卷、Host、部署、E2E、容量或 production readiness 已完成。

