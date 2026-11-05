# identity-auth-S10.4 Code Review R01

- 日期：2026-10-25
- Reviewer：非作者独立 Reviewer
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：**FAIL / CHANGES REQUIRED**

## 1. 固定对象

开始检查时以下 SHA256 均与委派输入一致：

| 文件 | SHA256 |
|---|---|
| `internal/identity/password_reset_worker.go` | `A07733F3770834C47687AB4421576C45B4A4433738AD4EC973D43FA3E6D63038` |
| `internal/identity/password_reset_worker_test.go` | `50B6C2BF43DF4514D32D4E5B2353F8D09D0AB81AB9F1D51EAAD7B13AEFF864DB` |
| `internal/identity/password_reset_operator.go` | `914CCDE8E59707188D815379E2A53429DA0CDCE2ECE8E4AC06888987997ABEFC` |
| `internal/identity/password_reset_operator_test.go` | `D8DE971FC79986EEBFF8F7BB9F55955D745B0F77092A7CBBC397D822E8B32910` |
| `cmd/han-server/main.go` | `A0190C6F29E8DDFD43DB2066DF477F35D12E0CFD95D9E81919CF16514F57B966` |
| `cmd/han-server/main_test.go` | `5C33E342A81AE7F19ACAAE7ABF9CCA36A61E99F2157FA173F1A8145EBAABD773` |
| `cmd/han-server/lifecycle.go` | `2BF4A480267265DDE84F49C9587EAE84201F62C5D37CDB764150C71710DA516D` |
| `cmd/han-server/lifecycle_test.go` | `D15AFCAA73DDC86ABEFDCFD2D706C0F60BCF7E15EEC8E80E4E426E0A45F674E4` |

评审依据：

| 依据 | SHA256 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `750AD8D02DCCBF0223DED37A337006D6C38FB830CAF85964BAEA8A8E8F56FFFB` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `4661456F2870FF66D512641DC6BCBBB53AAA27140FCAAE84512F713AEE66AADA` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` r44 | `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` r20 | `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB` |
| `.spec/wip/identity-auth/review-design-r44-s10.4-r02.md` | `9A76D7E3C5BC29855043784057C99383844A0B3CC7F14E32F96EDBB5290F0434` |
| `.spec/wip/identity-auth/review-tasks-plan-s10.4-s10.5-r02.md` | `B0C727A361A87BEE69F0669C1606AC346705BCEBB4DD513F96A572D07E27A7E5` |

## 2. 两轮实际检查

### Round 1：固定八文件直接检查

- 逐文件检查 operator、worker、五项配置、production wiring、双 worker lifecycle 及其测试。
- 逐条检查 SQL 占位符和调用参数、User-first 锁序、事务 commit/rollback、`RowsAffected`、lease owner/attempt/expiry、retry/dead-letter、permit acquire/release、取消与 panic、启动和关闭顺序。
- 检查精确八文件边界、敏感信息、硬编码、隐藏生产入口、邮件发送、第二 Store/Gate/root/global/service locator。
- 对照 r44、r20 强制测试矩阵及任务卡判断交付完整性。

### Round 2：消费者、上游依据与资源所有权反查

- 反查 migration `0008_password_reset_durable.sql` 的 operation/job/credential/user/session/idempotency schema 和约束。
- 反查 S10.3 `PasswordResetStore`/Gate 及 Login、PasswordChange、Session 的注入和生产复用关系。
- 反查 Article deletion worker/lifecycle 的 Stop/Wait/取消实现及双 worker 共享关闭时序。
- 反查数据库配置解析、同一 pool/Store/Gate/permit owner 的 production construction。
- 沿设计的 commit-unknown、单有效租约、过期接管、终态重读、worker-before-pool 和测试责任检查真实失败场景。

确认项：

- 五项配置均为必填且无代码默认；内在错误在 DSN parse 前失败。
- 正数 connection budget 大于 `MaxConns` 时在 parse 后、connect 前失败。
- production root 复用同一 `*identity.PasswordResetStore`/Gate；未发现第二 Store/Gate/root 或 global/service locator。
- operator、worker、permit owner 均只在 production root 构造一次。
- 未发现 worker/operator 发邮件、maintenance command、runbook 或隐藏生产入口。
- dead-letter 路径未解除 Gate/hold。
- 启动顺序为 `Article Reconcile → PasswordReset Reconcile → Article Start → PasswordReset Start → Serve`。
- direct operator 与 worker-held permit 分层、panic defer release 的基本结构正确。

## 3. Findings

### HIGH — IDENTITY-S10.4-REV-R01-F001：accepted settlement 的 SQL 参数数量错误，正常成功收敛必然失败

- 位置：`internal/identity/password_reset_operator.go:259`，SQL 定义 `internal/identity/password_reset_operator.go:328`
- 现状：`passwordResetSettlementActivateSQL` 只有 `$1`、`$2` 两个参数，但调用传入 `now, operation.credentialID, lease.UserID` 三个参数。结果不仅数量不匹配，前两个绑定值的语义和类型也与 `id/user_id` 不符。
- 触发：任意 `delivery_state=accepted` 且进入 `settleAcceptedPasswordReset` 的正常恢复。
- 影响：驱动/服务端在 bind/execute 阶段返回参数错误，事务回滚；新 temporary credential 无法激活，operation/job 无法成功终结，三消费者 Gate 持续关闭，并会继续 retry 直至 dead-letter。
- 违反依据：
  - design r44 `:2402`：accepted settlement 必须在同一事务激活 new、推进能力、撤销 Session、更新 Idempotency/Audit 并终结 succeeded。
  - test-plan r20 `I-PRD-09` 及 `I-PRD-14.S104-04/05`。
  - `code-review-checklist.md`：主路径、事务状态及测试须与设计一致。
- 最小修复：按 SQL 合同只传 credential ID 和 User ID；不要把时间参数加入没有对应列/占位符的语句。
- 修复验证：
  1. 增加执行真实 settlement SQL 的事务替身或 PostgreSQL 协议级 harness，逐条断言 SQL 参数数量、顺序和类型。
  2. 覆盖 accepted 主路径完整 transition，断言 credential、user generation、sessions、operation、idempotency、audit 和 job 在同一事务收敛。
  3. 重跑定向普通/race、contract 和全量测试。

### HIGH — IDENTITY-S10.4-REV-R01-F002：过期 lease 仍可被旧 owner 续约、失败结算或用于 settlement

- 位置：
  - `internal/identity/password_reset_worker.go:93,110,119-123,143-146`
  - `internal/identity/password_reset_worker.go:281-315,318-334,348-352`
  - `internal/identity/password_reset_operator.go:161,165-168,213-228`
- 现状：
  1. `Reconcile` 在扫描前只采样一次 `now`，后续多个 claim 继续复用；锁等待后创建的 lease 可能已经过期，甚至 `expires=now+duration` 已落在当前时间之前。
  2. Renew/Fail 在取得 User 锁前采样时间，SQL 只校验 status、owner、attempt，不校验当前 lease 在写入时仍未过期。
  3. settlement 在等待 User 锁前采样 `now`；若锁等待跨过 expiry，`expires.After(now)` 仍会错误接受已经过期的租约。
- 触发：扫描、permit 等待、User 锁等待、事务暂停或调度暂停跨越 `lease_expires_at`，且旧 owner 尚未观察到新 owner 的 CAS；Renew/Fail/settlement 均可触发。
- 影响：
  - 旧 owner 可在授权已过期后“复活”租约或写入 retry/dead-letter。
  - 过期租约可能继续执行凭据、Session、幂等和审计状态迁移。
  - 破坏“单有效租约”和“仅过期后恰一次接管”，并扩大多 worker 竞争下的双重结算窗口。
- 违反依据：
  - design r44 `:2171`：单 operation 最多一个未过期租约，漂移只在过期后接管。
  - design r44 `:2973`：lease owner/expiry/attempt 条件转换、过期接管、幂等终态。
  - test-plan r20 `I-PRD-14.S104-05`。
- 最小修复：
  - 每次 claim/renew/fail/settlement 在取得必要锁后使用数据库当前时间，或在锁后重新采样同一权威时钟。
  - Renew、Fail 和 settlement CAS 必须明确要求现有 `lease_expires_at` 在写入时仍未过期；claim expiry 必须基于实际 claim 时刻，而不是扫描时刻。
  - 保持 User-first 锁序，不通过放宽 owner/attempt 条件解决。
- 修复验证：
  1. 用可控 Clock 与同步锁屏障覆盖 expiry 前、等于 expiry、expiry 后。
  2. 覆盖扫描后延迟、claim 锁等待、renew 锁等待、settlement 锁等待跨 expiry。
  3. 两 worker 竞争下断言只有一个有效 owner、只有一次 takeover/settlement，旧 owner 的 renew/fail/settlement 全部为零写入。

### HIGH — IDENTITY-S10.4-REV-R01-F003：共享 shutdown deadline 到期后会在 PasswordReset worker 仍运行时关闭数据库池

- 位置：
  - `internal/identity/password_reset_worker.go:174-190,211-235`
  - `cmd/han-server/lifecycle.go:97-168,317-320`
- 现状：
  - worker 的 active `Reconcile` 使用 `context.WithoutCancel(parent)`。
  - `Stop` 只关闭 stop channel，不能取消当前正在执行的 query、锁等待或事务。
  - `Wait` 在 shutdown context 到期时立即返回，而不等待 `done`。
  - lifecycle 在所有 Wait goroutine 返回后继续关闭 Article 资源、listener，最终关闭 pool；此时 PasswordReset worker 仍可能持有 permit、连接或事务。
- 触发：worker 在 query、User 锁、事务、permit 或 operator settlement 中阻塞，shutdown shared deadline 先到期。
- 影响：pool/连接可与仍运行的 settlement 并发关闭，产生 goroutine/事务泄漏、回滚或 commit 结果不确定；还违反“两个 worker 全部停止后才能关闭 Article 资源和 pool”的明确顺序。
- 违反依据：
  - design r44 `:850,3097`。
  - test-plan r20 `I-PRD-15.S104-03`：worker 仍可能提交时 `pool close=0`。
  - Article deletion worker 已提供可取消 active operation 并在 deadline 分支等待 `done` 的直接消费者先例。
- 最小修复：让 Start/run 持有可由 Stop 取消的运行 context；Stop 必须取消 active reconcile；Wait/lifecycle 在关闭 Article 资源和 pool 前必须取得 worker `done` 的确定屏障。错误仍可记录 shared deadline，但不能把 Wait context 到期误当作 worker 已停止。
- 修复验证：
  1. 用同步屏障阻塞 PasswordReset query/transaction，触发 shutdown deadline。
  2. 断言取消已传递到 active operation，且 `done` 前 `closeArticle`、listener close、pool close 调用数均为 0。
  3. 解除阻塞后断言 worker、Serve、listener、Article、pool 的完整顺序和无 goroutine/channel 泄漏。

### HIGH — IDENTITY-S10.4-REV-R01-F004：settlement commit acknowledgement 丢失会把已成功收敛误报为 worker terminal 并触发全局停服

- 位置：
  - `internal/identity/password_reset_operator.go:200-209`
  - `internal/identity/password_reset_worker.go:137-149,233-235`
- 现状：operator 的 Commit 返回任何错误都直接返回；worker 随即调用 Fail。若数据库实际已经提交，只是 acknowledgement 丢失，则 job 已是 `succeeded`，Fail 的 leased CAS 为 0 行并返回冲突；worker把 settlement error 与 Fail error 合并为 terminal，`run` 关闭 `Done`，lifecycle 将其作为全局终止触发。
- 触发：settlement Commit 在服务端实际成功后连接中断、超时或响应丢失。
- 影响：凭据和 operation 已成功收敛，却被误判为 worker terminal，导致整个 server shutdown；没有执行设计要求的权威终态重读，也无法把重试安全地归并为幂等成功。
- 违反依据：
  - design r44 `:2403`：settlement commit unknown 保持 unresolved 并由 worker 按租约收敛。
  - design r44 `:2970,2973`：联合核对权威状态、终态重读、幂等终态和 worker 持续收敛。
  - `code-review-checklist.md`：commit unknown、重试和幂等失败路径必须收敛。
- 最小修复：Commit unknown 后按既定 User-first 锁序开启只读/受控核对，重读 operation、job、exact credential、user/session/idempotency/audit 的权威终态；已完整提交则归并为幂等成功，明确未提交才进入 retry，无法判定则保持 fail-closed/可恢复状态，不能盲写 Fail 或把已成功状态升级为 worker terminal。
- 修复验证：
  1. 故障注入“Commit 返回错误但 actual state=committed”和“actual state=rolled back”两类。
  2. committed 分支断言 worker 继续运行、无 Fail 写入、无重复 Audit/能力变更。
  3. rolled-back 分支断言安全 retry；无法判定分支保持 Gate，且不伪造 succeeded/failed。

### MEDIUM — IDENTITY-S10.4-REV-R01-F005：强制测试矩阵没有执行真实状态机、租约和资源关闭风险

- 位置：
  - `internal/identity/password_reset_worker_test.go:11-149`
  - `internal/identity/password_reset_operator_test.go:10-140`
  - `cmd/han-server/main_test.go:4904-4945`
  - `cmd/han-server/lifecycle_test.go:473-508,534-583`
- 现状：
  - worker 测试主要验证 fake 调用顺序/计数，没有验证 lease expiry CAS、User 锁等待、Rows/Tx、commit/rollback 或终态重读。
  - operator 测试只覆盖 permit 包装，实际 settlement SQL、accepted/invalidated transition、参数绑定、`RowsAffected`、审计、幂等和 commit unknown 均未执行。
  - 配置 overflow 用例对所有键统一使用带 `h` 的值；对整数键只证明语法 parse error，未证明目标整数类型 overflow。
  - lifecycle fake 的 Stop 会立即关闭 done，无法模拟 active operation 跨 deadline，因此未验证“worker 仍可提交时 pool close=0”。
- 触发：当前定向普通/race 或 Lead 全量门禁执行。
- 影响：测试全部通过仍未发现 F001–F004；r20 标为强制的交付证据没有形成，不能以绿灯证明 S10.4 的状态机和 production lifecycle 正确。
- 违反依据：
  - design r44 `:3586-3593` 的八文件测试责任。
  - test-plan r20 `I-PRD-14.S104-01/04/05`、`I-PRD-15.S104-03`。
- 最小修复：增加事务/Rows/锁/commit harness、可控 Clock、多 worker lease 边界和 blocked-shutdown 同步屏障；整数配置使用超过目标整数上限但语法合法的十进制输入。
- 修复验证：先证明新增回归在当前 revision 上失败，再随 F001–F004 修复转绿；普通与 race 都必须执行。

## 4. 待决定与非阻断建议

- 待决定：无。上述问题均可在已确认 r44/r20 契约内修复；若修复必须改变 schema、公共契约、锁序或 shutdown 合同，应停止并升级上游方向决策。
- 非阻断建议：无。本轮仅报告会影响正确性、安全收敛或强制交付证据的问题。

## 5. Reviewer 运行命令与结果

```text
go test ./internal/identity ./cmd/han-server -run 'TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle' -count=1
```

结果：PASS。

```text
go test -race ./internal/identity ./cmd/han-server -run 'TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle' -count=1
```

结果：PASS。

Lead 已提供并声明当前固定 revision 现场通过：定向普通测试、定向 race、`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`、scoped `git diff --check`。本 Reviewer 未重复执行除上述两条外的全量命令；绿灯不消除本报告的静态和故障时序 findings。

## 6. 未验证范围与剩余风险

- 按授权未运行真实 PostgreSQL、SMTP/TLS、部署、性能或 E2E，也未读取或输出测试 DSN。
- 未对真实 PostgreSQL 驱动执行 accepted settlement，因此 F001 依据为 SQL 占位符与调用参数的确定性静态不匹配。
- 未进行真实连接中断/commit acknowledgement 丢失注入；F004 由当前 Commit/Fail/terminal 控制流与持久化 CAS 合同推导。
- 未做全仓泛查；扩展范围限于 migration 0008、S10.3 Store/Gate 及三消费者、Article worker/lifecycle、数据库配置解析。
- S10.4 在 F001–F005 关闭并完成独立复审前，不具备 Code Review Gate PASS，也不能据此声明 production readiness。

## 7. Code Review Gate

**FAIL / CHANGES REQUIRED**

阻断原因：4 个 HIGH 正确性/并发/生命周期缺陷，以及 1 个 MEDIUM 强制测试证据缺口。复审条件：

1. 提供固定修复 revision、八文件新 SHA256 和与原 revision 的有界 diff。
2. 逐条提供 F001–F005 的回归测试及普通/race 结果。
3. F001 需证明 accepted settlement 的真实参数和完整事务状态迁移。
4. F002 需证明锁等待跨 expiry 时旧 owner 无权 renew/fail/settle，且 takeover 恰一次。
5. F003 需证明 worker `done` 前不会关闭 Article 资源或 pool。
6. F004 需覆盖 committed、rolled-back、unknown 三种 commit acknowledgement 故障结果。
7. 复审前后再次校验固定修订哈希；若修复改变已确认契约，先返回上游决策。

## 8. 结束哈希

结束复核时分支 / HEAD 仍为 `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。固定八文件与全部评审依据均未漂移：

| 文件 | 结束 SHA256 |
|---|---|
| `internal/identity/password_reset_worker.go` | `A07733F3770834C47687AB4421576C45B4A4433738AD4EC973D43FA3E6D63038` |
| `internal/identity/password_reset_worker_test.go` | `50B6C2BF43DF4514D32D4E5B2353F8D09D0AB81AB9F1D51EAAD7B13AEFF864DB` |
| `internal/identity/password_reset_operator.go` | `914CCDE8E59707188D815379E2A53429DA0CDCE2ECE8E4AC06888987997ABEFC` |
| `internal/identity/password_reset_operator_test.go` | `D8DE971FC79986EEBFF8F7BB9F55955D745B0F77092A7CBBC397D822E8B32910` |
| `cmd/han-server/main.go` | `A0190C6F29E8DDFD43DB2066DF477F35D12E0CFD95D9E81919CF16514F57B966` |
| `cmd/han-server/main_test.go` | `5C33E342A81AE7F19ACAAE7ABF9CCA36A61E99F2157FA173F1A8145EBAABD773` |
| `cmd/han-server/lifecycle.go` | `2BF4A480267265DDE84F49C9587EAE84201F62C5D37CDB764150C71710DA516D` |
| `cmd/han-server/lifecycle_test.go` | `D15AFCAA73DDC86ABEFDCFD2D706C0F60BCF7E15EEC8E80E4E426E0A45F674E4` |
| `.add/project/knowledge/tasks/tasks.md` | `750AD8D02DCCBF0223DED37A337006D6C38FB830CAF85964BAEA8A8E8F56FFFB` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `4661456F2870FF66D512641DC6BCBBB53AAA27140FCAAE84512F713AEE66AADA` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB` |
| `.spec/wip/identity-auth/review-design-r44-s10.4-r02.md` | `9A76D7E3C5BC29855043784057C99383844A0B3CC7F14E32F96EDBB5290F0434` |
| `.spec/wip/identity-auth/review-tasks-plan-s10.4-s10.5-r02.md` | `B0C727A361A87BEE69F0669C1606AC346705BCEBB4DD513F96A572D07E27A7E5` |
