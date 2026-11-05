# article-management-S4.6 独立代码复审 r03

- Reviewer：Faraday（独立非作者 Reviewer）
- 评审日期：2026-10-23
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：
  - `.spec/wip/article-management/review-code-s4.6-r01.md`
  - `.spec/wip/article-management/review-code-s4.6-r02.md`
- **Code Review Gate：PASS**

## 1. 固定对象与漂移核对

评审开始时确认 r03 报告路径不存在，分支与 HEAD 匹配；r01 SHA256 为
`778D8704132991844C4201A34C9EA4F583372AECA6A50B716BF1F1C312923C99`，
r02 SHA256 为
`DE157C9DFCA29AF15F8984FC40BFD8647BF3740F76C8BD1B2F52E9223D94CBD8`。

开始及落报告后均逐项核对以下 13 个固定对象；两次均为 0 mismatch：

| 文件 | git blob | SHA256 |
|---|---|---|
| `cmd/han-server/main.go` | `a61811669db6205d4cd7d75a3d82e70d81c74066` | `371738A08A847E67D971C998BDFF44B36A8DF36E7C5EA411864BC26FB954E449` |
| `cmd/han-server/main_test.go` | `5d213abf995857192eb4cc6e678017c5e8696dfc` | `514CB7B96F761CC83D0800B6B22C511497A7A090F511C522648B063010457BD1` |
| `cmd/han-server/lifecycle.go` | `9100ff4bba6ea1741b24754254e8b131fe1505c2` | `B6B246667F9279800B023EF67E8FB8D5EB8A2F4E56B6DA403C6D591E3B391816` |
| `cmd/han-server/lifecycle_test.go` | `b79ba7e06262b60ec04c4c416728b232933c748d` | `8C59F61FAD9A65E637EB9991C2951293ECDC7417418B38044D3243EBDF32E66B` |
| `internal/article/deletion_worker.go` | `911657f38a44f915f3ff3948830cb59a0e4609f7` | `9E6BDD23089B41A3870DB0A35420589DD2210972C0610A43184D6E7C60F6612D` |
| `internal/article/deletion_worker_test.go` | `03b0a60777bc0489dfc13982599b50eec78972ee` | `444C304EFFE624B65345C138C9BF75964AEFA9C52E6293341961DAAF59E107CA` |
| `internal/article/deletion_finalizer.go` | `66e7a8d47eaa390d9243db824afa10f05d9273e9` | `73E0E967EA9A3AA85F610E2146B1A03B9CA96DAB001E8B0F071524BA5FB470E0` |
| `internal/article/deletion_journal.go` | `810a8017dbecf7ee71b386802f49a1f830d6803c` | `B70E65B56AEAE06E465A52D96C4CC2CDC1AAB021A66C5C9700261EA448317B4F` |
| `internal/article/deletion_test.go` | `a3ff6ce95c34478754e13c1b540f457fb360a576` | `89457FEEDE0803ED1A12942A81CAED590132ED1F02B357EF991C322CBFC698B2` |
| `internal/article/repository.go` | `41c18954de82ff9253f27ea0b2fffda08496263e` | `5A03F79D15706CB5CDD4F9A51129D388E192AF7967DD848C6585686FA511DDC7` |
| `internal/article/repository_integration_test.go` | `a5f44b8a9d98197729e7c2c6ab71ff533237cc76` | `EE9A68CCA141AC6F84C412C6BCE54457C543003ABB2812D4B8C3690D9B269355` |
| `internal/article/deletion_filelock_flock.go` | `b3d33952515d1eb67dd59c89fa843c6349d454a1` | `C72AC42EAF78B279CB774C79B9024A7E330C1FCCE2D40D912782E5B0B50A5C96` |
| `internal/article/deletion_filelock_unsupported.go` | `ad26bfcdb0b422c8801774cfe8dbdcb5e9aa3b35` | `BF140294B11D068B2971DFEFF4D3598FFAEC591FCF1F5BE0435640370C5FEB8B` |

与 r02 固定对象相比，变化仅位于已授权的
`main.go`、`main_test.go`、`deletion_worker.go`、`deletion_worker_test.go`
和 `deletion_test.go`；其余八个对象保持 r02 摘要。

## 2. 实际读取与评审依据

完整读取并执行：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- 适用的 observability、tracing、产品安全红线
- 已启用 `go-service` profile 的 README、coding、no-hardcode、security、
  observability、tracing 规则

有界读取：

- r01、r02 报告及 `ARTICLE-S4.6-R01-F002` 的关闭条件
- article design r10、test-plan r08 的 A-WIRE-CFG/RTR/WKR 合同
- `tasks.md` 的完整 `article-management-S4.6` 条目及 B6A-W Plan
- 13 个固定对象
- 唯一 production composition/router、Article 删除授权、repository
  discovery、Journal/finalizer、worker/lifecycle 直接接缝与相关测试

未扩展为全仓审计；未读取、打印或持久化
`HAN_TEST_POSTGRES_DSN`。

## 3. 两轮复审范围

### Round 1：F002 定点关闭核对

逐项核对 r02 剩余的 CFG-04/06、RTR-03/05/06、WKR-03/07：

1. 配置上下界与越界是否逐项 fail closed；
2. 各配置失败类别是否在 DB、migration、composition、listener 前归零；
3. 路由安全链是否完整有序且逐层拒绝后零后续调用；
4. 三条 DELETE 的路由、角色与状态组合是否由 composition 和既有业务授权共同证明；
5. Identity 与 Article 是否引用同一组共享实例；
6. fake repository 是否证明 discovery 状态、排序、limit 与 worker convergence；
7. transient/no-op/terminal 分类、退避和低敏 bucket 是否可确定性观察。

### Round 2：消费者与修订新增风险

沿 `newProductionBusinessHandler`、Article command/deletion 链、
PostgreSQL discoverer、Journal/finalizer、worker 与 lifecycle 做有界深审；
同时检查：

- 私有测试 seam 在生产为 nil 时不改变生产分支；
- observer 不进入公共 API，不记录 operation ID、路径、key、record body 或底层错误；
- callback 不创建后台任务；当前生产构造不注入 observer，因此不引入生产阻塞或竞态；
- F001/F003/F004 的路径安全、lease competition/no-op 与去敏边界无回退；
- 未新增第 14 文件、migration、依赖、第二 composition/router 或 Recovery
  私有原语变更。

## 4. 原 finding 状态

### `ARTICLE-S4.6-R01-F002` — **CLOSED**

#### CFG-04

`cmd/han-server/main_test.go:2250-2371` 对 poll、batch、lease、
operation budget、initial/max backoff、max attempts、shutdown budget
逐项覆盖合法下界、合法上界、下越界与上越界。r02 明列的八个缺口均已有
直接拒绝断言；测试调用真实 `loadStartupConfig`，不是源码字符串检查。

#### CFG-06

`cmd/han-server/main_test.go:2401-2555` 覆盖 root、file mode、
current/retained version 与 key、worker ID、全部数值配置以及四项交叉约束。
每个 case 均经真实 `run` 启动入口证明 DB parse/connect/ping/migrate、
composition、listener 六类调用全部为 0，并扫描 DSN、绝对路径、key、
正文与 PII canary。配置错误在副作用前返回稳定错误。

#### RTR-03 / RTR-05

- `cmd/han-server/main_test.go:1846-1956` 证明
  `Trace → Origin → Session → CSRF → Principal → authorization → Handler`
  的完整顺序，并对 Origin、Session、CSRF、Principal/授权逐层拒绝验证后续零调用。
- `cmd/han-server/main_test.go:1958-2055` 证明三条 exact DELETE 分别进入
  ordinary、violating、orphan handler，并覆盖 active/disabled/restricted
  HTTP principal 状态。
- `internal/article/deletion_test.go:328-419` 补充真实删除授权矩阵：
  ordinary author/admin/other member、violating admin/member、orphan
  admin/member 及 disabled/restricted 状态；因此 HTTP 路由 seam 未替代业务授权证明。
- `articleProductionTestSeam`、授权观察与 handler 替换均为 `main` 包私有；
  production composition 不设置该 seam，nil 分支保持现役行为。

#### RTR-06

`cmd/han-server/main_test.go:2727-2887` 同时保留 Identity route、
JSON 404/405/Allow/trace 回归，并从真实 composition 构造点观察 pool、
Session store、OriginGuard、AuditStore、logger、clock、ID provider。
每类引用均断言实例 identity 一致并具有预期的多个消费者；没有构造第二组
Article 专用共享资源。

#### WKR-03

`internal/article/deletion_worker_test.go:319-381,647-693` 的 fake repository
以混合 `pending_journal`、`journaled`、过期/未过期 `finalizing` 和
`completed` 数据证明：

- 只返回允许状态；
- `locked_until == now` 可重领，未来 lease 被排除；
- 按 `started_at,id` 稳定排序；
- limit 精确传递和截断；
- worker 按 discovery 顺序调用 convergence。

该层级符合 test-plan 明定的 “repository contract fake”；真实 SQL、索引和锁
仍正确留给后续 PostgreSQL 专项，不以源码字符串冒充本卡行为证据。

#### WKR-07

`internal/article/deletion_worker.go:16-36,145-215,429-468` 将分类限制为私有、
稳定的 category/reason allowlist。`deletion_worker_test.go:252-315,383-474,
476-598` 共同证明：

- DB discovery/convergence、Journal filesystem/busy、lease competition、
  trace/operation 类失败走 transient；competition 和 trace 分支实际进入
  有界 jitter/backoff；
- 裸 `ErrDeletionInProgress` 是 completed/not-claimed no-op，不退避；
- Journal corrupt、tombstone missing、Article mismatch 和 invariant
  均返回 terminal，既有 WKR-08/lifecycle 测试继续证明 terminal 停止并触发
  全局失败；
- full-jitter、批次数上限、operation budget cancel 由 WKR-06/WKR-10
  的确定性测试继续约束；
- observer 只接收固定 `Category`/`Reason`，canary 与 operation ID 不进入观察值。

observer 是同包私有白盒 seam；公共 `NewDeletionWorker` 和 production
composition 的公共接口、错误语义及资源所有权均未改变。

**结论**：r02 列出的全部剩余关闭条件均已由行为测试和直接实现证据覆盖，
F002 关闭。

## 5. 其他历史 finding 回归

| Finding | r03 状态 | 证据 |
|---|---|---|
| `ARTICLE-S4.6-R01-F001` | **CLOSED / NO REGRESSION** | Journal 路径实现摘要保持 r02；祖先 symlink、非目录、权限、child traversal 定向测试 PASS。 |
| `ARTICLE-S4.6-R01-F003` | **CLOSED / NO REGRESSION** | finalizer/flock 实现摘要保持 r02；competition 与裸 no-op 分类测试 PASS。 |
| `ARTICLE-S4.6-R01-F004` | **CLOSED / NO REGRESSION** | Journal/flock 去敏实现摘要保持 r02；startup/operation/close canary 回归 PASS。 |

## 6. 新 findings、待决定与建议

### 新 findings

无。

### 待决定

无。未发现需要改变已确认架构、公共 REST、migration、依赖、Recovery
私有原语或 13 文件边界的问题。

### 非阻断建议

无。本轮不把真实环境专项或测试风格偏好升级为代码 finding。

## 7. Reviewer 独立命令与结果

以下均由 Reviewer 本轮独立运行；Lead 证据仅作为外部输入，未冒充本轮执行。

```text
git branch --show-current
git rev-parse HEAD
Test-Path .spec/wip/article-management/review-code-s4.6-r03.md
Get-FileHash -Algorithm SHA256 <r01/r02/13 fixed files>
git hash-object -- <13 fixed files>
```

结果：开始时 branch/HEAD、r01/r02 与 13 对象匹配；报告创建前不存在。

```text
go test ./cmd/han-server -run \
  'TestLoadStartupConfig_ArticleDeletionMatrix|TestLoadStartupConfig_ArticleDeletion_AWIRECFG06HasZeroSideEffects|TestProductionBusinessHandler_AWIRERTR03RTR05OrdersAndStopsEverySecurityLayer|TestProductionBusinessHandler_AWIRERTR05ThreeDeleteRoutesCarryRoleAndStatus|TestProductionBusinessHandler_AWIRERTR06UsesOneSharedInstancePerDependency' \
  -count=1

go test ./internal/article -run \
  'TestDeletionWorker_AWIREWKR03DiscoveryStateOrderAndLimitMatrix|TestDeletionWorker_AWIREWKR07TraceFailureIsTransient|TestDeletionWorker_AWIREWKR07CategoryReasonBucketsAreCompleteAndLowSensitivity' \
  -count=1

go test ./internal/article -run \
  'TestDeletionJournal_AWIREJRN02RejectsUnsafePathComponents|TestDeletionFinalizer_AWIREWKR07ClassifiesActiveLeaseCompetition|TestDeletionWorker_AWIREWKR07DistinguishesCompetitionFromNoOp|TestDeletionJournal_AWIREJRN07SanitizesOperationAndCloseFailures' \
  -count=1
```

结果：全部 PASS。

```text
go test ./internal/article ./cmd/han-server -count=1
go test -race ./internal/article ./cmd/han-server -count=1
go vet ./internal/article ./cmd/han-server
go build -o NUL ./cmd/han-server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -c -o NUL ./internal/article
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go test -c -o NUL ./internal/article
go test -tags=integration ./internal/article -run '^$' -count=1
go mod verify
gofmt -l <13 fixed Go files>
```

结果：全部 PASS；`gofmt -l` 无输出；integration-tag 命令显示
`[no tests to run]`，仅作为编译证据；双 GOOS 命令仅编译、不执行目标平台代码。

```text
git diff --check -- <13 fixed files> \
  .spec/wip/article-management/review-code-s4.6-r03.md
PowerShell trailing-whitespace scan <13 fixed files> <r03>
```

结果：限定 `git diff --check` PASS；因当前固定代码为未跟踪工作树对象，另做
逐行 trailing-whitespace 扫描，计数为 0；未留下 `han-server.exe` 或 `NUL`
普通文件。

落报告后最终复核结果：branch/HEAD、r01/r02 SHA256 与 13 个固定对象
SHA256/git blob 均保持匹配，mismatch 为 0。

## 8. 未验证限制

- 未连接真实 PostgreSQL；未验证真实事务、锁、lease、commit unknown 或双连接竞争。
- 未在受支持 Unix 持久卷执行真实 `flock`、file/directory `fsync`、崩溃、
  断电、磁盘故障或跨进程竞争；双 GOOS 仅 compile-only。
- 未运行真实 Host、部署、signal/exit E2E、浏览器或外部服务。
- 未执行容量、吞吐、积压或目标环境 benchmark。
- 未读取或使用 `HAN_TEST_POSTGRES_DSN`。
- 本 PASS 不表示上述专项、E2E 或 production readiness 已通过。

## 9. 最终 Gate

| Finding | 最终状态 |
|---|---|
| `ARTICLE-S4.6-R01-F001` | CLOSED / NO REGRESSION |
| `ARTICLE-S4.6-R01-F002` | **CLOSED** |
| `ARTICLE-S4.6-R01-F003` | CLOSED / NO REGRESSION |
| `ARTICLE-S4.6-R01-F004` | CLOSED / NO REGRESSION |

新增 finding：无。

**Code Review Gate：PASS**

无需代码复审返工。后续仍须由既定专项 owner 分别关闭真实 PostgreSQL、
受支持 Unix 持久卷、Host/部署/E2E 与容量边界，不得从本报告外推
production readiness。
