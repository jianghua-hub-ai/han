# identity-auth-S10.4 Code Review R02

- 日期：2026-10-25
- Reviewer：R01 原非作者独立 Reviewer
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：当前未提交、固定八文件修复 revision
- 结论：**FAIL / CHANGES REQUIRED**

## 1. 固定对象与依据

开始检查时八文件 SHA256 均与委派值一致：

| 文件 | SHA256 |
|---|---|
| `internal/identity/password_reset_worker.go` | `9BF1A9B1230004422E9ABC563329C9C7E44BB48CC1E1452CB5DF18E20D704C90` |
| `internal/identity/password_reset_worker_test.go` | `DC6544C4390CC36FC93ED4B66D530E2E0FBC067A7C733FFF6692944CBC4801CD` |
| `internal/identity/password_reset_operator.go` | `C6B462EDE88A862655B59A3D4F698EF76BBD199FA157E5F0318DD8106B2BA1A6` |
| `internal/identity/password_reset_operator_test.go` | `5FB037438DFC6818AEB04F92B241421D227B5520E5578ED9D37FB92788B61D13` |
| `cmd/han-server/main.go` | `A0190C6F29E8DDFD43DB2066DF477F35D12E0CFD95D9E81919CF16514F57B966` |
| `cmd/han-server/main_test.go` | `7990D7A8CAFCABA5CB50CD601417C33E5DE60F12E71D532E3A98CC2EB113E884` |
| `cmd/han-server/lifecycle.go` | `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888` |
| `cmd/han-server/lifecycle_test.go` | `25DCD9F790D1C0E6462AA7F62592B8E51C941327C19452E80C18CEA111C01CA9` |

主要依据：

| 依据 | SHA256 |
|---|---|
| R01 `.spec/wip/identity-auth/review-code-s10.4-r01.md` | `DC1E66FBEB9B2D37BC52BC1E5F55BA87A89EF079823377DD6A702075C86B755C` |
| tasks `.add/project/knowledge/tasks/tasks.md` | `750AD8D02DCCBF0223DED37A337006D6C38FB830CAF85964BAEA8A8E8F56FFFB` |
| plan `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `4661456F2870FF66D512641DC6BCBBB53AAA27140FCAAE84512F713AEE66AADA` |
| design r44 | `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23` |
| test-plan r20 | `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB` |

另完整读取并应用了 `add-reviewer` Role、`add-code-review`、code review checklist、abstraction principles，以及 `go-service` 的 coding、security、no-hardcode、observability、tracing 适用规则。

## 2. R01 findings 逐项复审

| R01 finding | 状态 | 复审结论与证据 |
|---|---|---|
| `IDENTITY-S10.4-REV-R01-F001` | **CLOSED** | accepted settlement 已按 SQL 合同仅传 credential ID、User ID（`password_reset_operator.go:495`）。`TestPasswordResetOperatorAcceptedSettlementExecutesCompleteTransaction` 使用事务 harness 逐步核对参数、RowsAffected、commit/rollback，并断言 credential、User generation、Session、operation、Idempotency、job、Audit 同事务收敛。 |
| `IDENTITY-S10.4-REV-R01-F002` | **OPEN** | Claim、Renew、Fail 已在 User 锁后重采样，Renew/Fail 增加 expiry CAS；settlement 也在 User 锁后首次采样。但 settlement 后续所有业务写仍使用 `password_reset_operator.go:173` 的旧 `now`，job completion 在 `:545-546` 把该值传给 `:571` 的 expiry CAS。若事务在首次 lease 校验后、job completion 前跨过 expiry，旧 owner 仍可用旧时间通过 CAS 并提交。详见 Finding 1。 |
| `IDENTITY-S10.4-REV-R01-F003` | **CLOSED** | PasswordReset worker 的运行 context 可由 Stop/Wait 取消（`password_reset_worker.go:171-213,276-282`），active Reconcile 不再使用不可取消 context；lifecycle 在 Wait 返回后仍等待 `Done`（`lifecycle.go:317-321`），因此关闭 Article 资源、listener、pool 前存在确定性 worker 完成屏障。 |
| `IDENTITY-S10.4-REV-R01-F004` | **CLOSED** | settlement Commit 出错后执行 User-first authority 重读，区分 committed、rolled-back、unknown；committed 归并成功，rolled-back 才允许 worker Fail，unknown 不写 Fail且不使 worker terminal。三态 operator 回归和 worker unknown 回归均已增加并通过。新发现的 commit cause 错误链中断单列为 R02 finding，不重开本项的状态机结论。 |
| `IDENTITY-S10.4-REV-R01-F005` | **OPEN** | 已补 SQL/事务 harness、Rows/Tx 释放、lease 边界、双 worker takeover、commit 三态、目标整数 overflow、permit peak 和 blocked shutdown 场景；但 blocked-shutdown 的核心负断言仍依赖 `time.Sleep`，且没有覆盖 settlement 在首次校验后跨 expiry 的最终 CAS。详见 Finding 2。 |

## 3. 两轮实际检查

### Round 1：修复点与固定八文件直接复审

- 逐项追踪 F001～F005 的修复代码、对应回归和失败分类。
- 检查 settlement SQL 参数、事务 defer、commit/rollback、RowsAffected、lease owner/attempt/expiry CAS、终态重读和 commit acknowledgement 三态。
- 检查 worker Scan→Claim→Renew→settle/Fail、重试/dead-letter、Stop/Wait/Done/Err、permit 顶层所有权和 panic/cancel 归还。
- 检查五项配置的 missing/blank/parse/overflow/zero/negative、pre-DSN 失败以及 budget/MaxConns 的 parse 后、connect 前失败。
- 检查双 worker Reconcile/Start/Serve 顺序、运行错误优先级、shared deadline、Stop/Wait/Shutdown/Close、worker 完成屏障和 pool 最后关闭。
- 检查八文件中敏感信息、生产值硬编码、maintenance command/runbook、worker/operator Mailer 权限及隐藏入口。

### Round 2：消费者、上游定义与新增风险反查

- 对照 design r44、test-plan r20、tasks S10.4、plan B6C-3/S10.4 和 R01 的精确修复条件。
- 反查 migration `0008_password_reset_durable.sql` 的 job 状态、attempt、owner/expiry、completed_at 约束。
- 反查 S10.3 `PasswordResetStore`/Gate、Login、PasswordChange、Session：production 仍只构造一个 Store/Gate，并把同一指针复用给三消费者、operator 和 worker。
- 反查 Article deletion worker 的 Stop/Wait/cancel 语义及 lifecycle 的双 worker 关闭关系。
- 静态反查生产 constructor：未发现第二 Store/Gate、第二 composition root、包级 service locator、lifecycle 构造依赖或新增 maintenance 入口；S10.5 仍为 `[ ]` 且不可执行。
- 复查新增测试是否能在修复回退时确定性失败，并检查错误链、日志/审计和 fail-closed 行为。

## 4. Findings（按严重度）

### HIGH — `IDENTITY-S10.4-REV-R01-F002`：settlement 跨过 lease expiry 后仍可由旧 owner 提交

- 位置：
  - `internal/identity/password_reset_operator.go:173-180`
  - `internal/identity/password_reset_operator.go:468-529`
  - `internal/identity/password_reset_operator.go:545-546`
  - `internal/identity/password_reset_operator.go:571`
- 触发：旧 owner 在取得 User/job 锁并通过 `expires.After(now)` 后，credential/User/Session/operation/Idempotency/Audit 任一步阻塞或调度暂停，直到真实时间到达或越过 `lease_expires_at`；随后 job completion 仍使用首次校验时的旧 `now`。
- 影响：`lease_expires_at > $1` 对旧采样值仍成立，整个 settlement 可在授权实际过期后提交。等待同一行锁的接管 worker随后只能看到旧 owner 已提交，违反“单有效租约、仅过期后恰一次接管、旧 owner 过期后零写入”的 r44/r20 合同。
- 违反依据：design r44 §4.1 worker 崩溃/重复执行不变量；test-plan r20 `I-PRD-14.S104-05`；R01 F002 明确要求 settlement 写入时 expiry CAS。
- 最小修复：
  1. 把 lease completion CAS 作为事务内最后一个状态写；
  2. 在该 CAS 前、User/job 锁已持有时重新采样同一权威时钟，或直接使用数据库当前时间；
  3. owner/attempt/status/expiry 任一不满足时返回冲突并整体回滚，不放宽 User-first 锁序。
- 验证要求：用同步屏障在首次 lease 校验后、最终 CAS 前把时钟推进到 expiry 前/等于/之后；等于和之后必须零 commit、完整 rollback，随后第二 worker只接管并成功 settlement 一次。

### MEDIUM — `IDENTITY-S10.4-REV-R01-F005`：强制回归仍未形成确定性的 expiry/关闭屏障证据

- 位置：
  - `cmd/han-server/lifecycle_test.go:640-711`，尤其 `:689`
  - `internal/identity/password_reset_operator_test.go:242-279`
- 触发：
  1. lifecycle 回归在 `waitReturned` 后用 `time.Sleep(20ms)` 推断资源尚未关闭；
  2. operator expiry 回归只覆盖“取得 User 锁后立即发现已到 expiry”，没有把时间推进点放在首次 lease 校验与最终 completion CAS之间。
- 影响：调度较慢时，即使 lifecycle 错误地在 worker Done 前关闭资源，20ms 内尚未运行到关闭点也可能假通过；同时 F002 的真实剩余窗口不会被现有绿色测试发现。该证据不满足 r20 明确的“同步屏障，不用 sleep”。
- 违反依据：test-plan r20 `I-PRD-15.S104-03`、`password-reset deterministic harness`、`S10.4 双worker lifecycle harness`；R01 F005 修复条件。
- 最小修复：
  - blocked worker fake 的 `Done()` 方法增加“lifecycle 已调用并即将阻塞”的 channel/barrier；测试等待该屏障后立即断言 Article/listener/pool close=0，再关闭真实 done channel并断言完整顺序，删除 `time.Sleep`。
  - 增加 Finding 1 所述 settlement 中段跨 expiry 回归。
- 验证要求：新回归普通与 race 均通过，并能在移除 lifecycle Done 屏障或恢复旧 settlement 时间采样时稳定失败。

### MEDIUM — `IDENTITY-S10.4-REV-R02-F001`：commit acknowledgement 错误的原始 cause 未保留在错误链

- 位置：`internal/identity/password_reset_operator.go:236,239`
- 现状：rolled-back 与 unknown 分支使用 `fmt.Errorf("%w: %v", sentinel, commitErr)`；只有分类 sentinel 被 `%w` 包装，原始 `commitErr` 仅格式化为文本。
- 触发：Commit 返回可由 `errors.Is`/`errors.As` 识别的驱动、网络、context 或测试注入错误。
- 影响：上层只能识别 settlement 分类，不能通过错误链定位原始 commit 故障；这削弱 commit-ack-loss 的诊断、观测和根因关联，也违反已启用 Go profile 的强制错误链规则。
- 违反依据：`go-service/rules/coding-standards.md`（用 `%w` 保留错误链）、`observability-standards.md`（错误链不能断）、code-review checklist 的“错误保留根因”。
- 最小修复：使用 `errors.Join` 或等价多重 unwrap，同时保留稳定分类 sentinel 与带上下文 `%w` 包装的 `commitErr`；unknown 分支继续合并 `verifyErr`。
- 验证要求：rolled-back、unknown 两类测试同时断言 `errors.Is(err, classificationSentinel)` 和 `errors.Is(err, commitErr)`；验证失败时还应保留 `verifyErr`。

## 5. 已确认通过的其他风险点

- 五项 PasswordReset worker 配置均显式必填、无默认；目标整数 overflow 已使用合法十进制超界输入覆盖。
- 仅正数 connection budget 大于 `MaxConns` 时允许数据库配置 parse=1，并在 connect 前失败。
- production root 仍复用 S10.3 的同一 `*identity.PasswordResetStore`/Gate；operator、worker、permit owner 各构造一次。
- worker/operator 没有 Mailer 权限；没有新增 maintenance command、runbook或生产隐藏入口。
- dead-letter 仅更新 job，不解除 Gate/hold。
- permit 的 direct path、worker-held path、无嵌套 acquire、error/cancel/panic 释放及并发 peak 回归已覆盖。
- commit committed/rolled-back/unknown 的安全控制流已建立；unknown 不 Fail、不 terminal。

## 6. Reviewer 实际运行命令

```text
go test ./internal/identity ./cmd/han-server -run 'TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle' -count=1
```

结果：PASS。

```text
go test -race ./internal/identity ./cmd/han-server -run 'TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle' -count=1
```

结果：PASS。

另执行并通过：

- 固定八文件开始/写报告前 SHA256 校验；
- R01、tasks、plan、design r44、test-plan r20 SHA256 校验；
- scoped `git diff --check`；
- production constructor/Store/Gate/permit/worker/operator 引用与禁止入口静态检索；
- 根目录 `han-server.exe` 不存在检查。

Lead 已提供同一固定 revision 的全量普通/race、contract、vet、build、mod verify、gofmt 与 scoped diff-check PASS 证据；本 Reviewer 未重复运行这些全量命令。绿色测试不消除上述静态时序与证据缺口。

## 7. 未验证范围与剩余风险

- 按授权未运行真实 PostgreSQL、SMTP/TLS、部署、Host 或 E2E；未读取、打印或使用 `HAN_TEST_POSTGRES_DSN`。
- 未验证 PostgreSQL 18 的真实多连接锁等待、lease 接管和 commit acknowledgement 丢失；这些仍由既定专项承接。
- 未做性能/容量结论；五项生产实际值仍为 TBD。
- 共享工作树存在其他人的大量改动；本结论只针对固定八文件哈希和有界直接消费者，不外推全仓其他未提交内容。
- F002/F005 修复后须重新固定八文件哈希并定点复审；若修复需要改 schema/migration、公共 REST、锁序、shutdown 合同或第九 production 文件，应停止并交由 Lead 做上游决定。

## 8. 待决定、建议与 Gate

- 待 Lead 决定项：无新的产品或架构方向决定；三个 OPEN finding 均可在已确认 r44/r20 与精确八文件内修复。
- 非阻断建议：无；本报告只记录合同、正确性与强制证据问题。
- R01 状态汇总：F001 **CLOSED**、F002 **OPEN**、F003 **CLOSED**、F004 **CLOSED**、F005 **OPEN**。
- 新 finding：`IDENTITY-S10.4-REV-R02-F001` **OPEN**。
- **Code Review Gate：FAIL / CHANGES REQUIRED**。

## 9. 结束哈希

报告完成后重新校验，固定八文件 SHA256 与本报告第 1 节开始哈希相同；未发生对象漂移。分支仍为 `main`，HEAD 仍为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
