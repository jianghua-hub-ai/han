# identity-auth-S10.4 Code Review R03

- 日期：2026-10-25
- Reviewer：R01/R02 原非作者独立 Reviewer
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：当前未提交、固定八文件修复 revision
- 结论：**PASS**

## 1. 固定对象

开始检查时八文件 SHA256 均与委派值一致：

| 文件 | SHA256 |
|---|---|
| `internal/identity/password_reset_worker.go` | `9BF1A9B1230004422E9ABC563329C9C7E44BB48CC1E1452CB5DF18E20D704C90` |
| `internal/identity/password_reset_worker_test.go` | `DC6544C4390CC36FC93ED4B66D530E2E0FBC067A7C733FFF6692944CBC4801CD` |
| `internal/identity/password_reset_operator.go` | `4DC973ECB2272E07083884224A898D0C7C38E4CFEE8B03E44DA14E9D9F951F42` |
| `internal/identity/password_reset_operator_test.go` | `9463D7B8B098A21BA3F68BDB190CFCF51C6FA47038BA0BD8A7E7505ADD210E02` |
| `cmd/han-server/main.go` | `A0190C6F29E8DDFD43DB2066DF477F35D12E0CFD95D9E81919CF16514F57B966` |
| `cmd/han-server/main_test.go` | `7990D7A8CAFCABA5CB50CD601417C33E5DE60F12E71D532E3A98CC2EB113E884` |
| `cmd/han-server/lifecycle.go` | `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888` |
| `cmd/han-server/lifecycle_test.go` | `B3AA659B1A98E0AE8A3EF09678309866DC3D46345B2A6C29A5A220D700B36DDF` |

复审依据：

| 依据 | SHA256 |
|---|---|
| R01 `.spec/wip/identity-auth/review-code-s10.4-r01.md` | `DC1E66FBEB9B2D37BC52BC1E5F55BA87A89EF079823377DD6A702075C86B755C` |
| R02 `.spec/wip/identity-auth/review-code-s10.4-r02.md` | `BC642532262E18272697038DD613A2AE5529CEEBB4EE9BEF35DF651BD601E852` |
| design r44 | `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23` |
| test-plan r20 | `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB` |
| tasks | `750AD8D02DCCBF0223DED37A337006D6C38FB830CAF85964BAEA8A8E8F56FFFB` |
| plan | `4661456F2870FF66D512641DC6BCBBB53AAA27140FCAAE84512F713AEE66AADA` |

## 2. R02 OPEN findings 定点复审

### `IDENTITY-S10.4-REV-R01-F002` — **CLOSED**

- `settleAcceptedPasswordReset` 与 `settleUnacceptedPasswordReset` 不再提前完成 recovery job；二者只完成业务状态和 Idempotency 写入并返回（`password_reset_operator.go:474-535`）。
- Audit 写入完成后重新调用权威 Clock，并把新采样的 `completionNow` 传给 lease completion CAS（`:196-205`）。
- `completePasswordResetRecoveryJob` 是非终态 settlement 事务的最后一个状态写；SQL 同时检查 leased status、owner、attempt 和 `lease_expires_at > completionNow`（`:551-577`）。零行会返回冲突并由事务 defer 完整 rollback。
- 已终态 operation 分支同样在 completion CAS 前重新采样 Clock，且该 CAS 是该分支唯一及最后状态写（`:178-183`）。
- `TestPasswordResetOperatorSettlementRechecksLeaseBeforeFinalCompletion` 使用 Audit 同步屏障把时间推进到 expiry 前、等于、之后：expiry 前仅一次 commit；等于/之后均零 commit、一次 rollback。之后以第二 owner、attempt=2 执行 settlement，两个事务合计仅一次 commit。
- unchanged worker 回归 `TestPasswordResetWorkerTwoWorkerTakeoverSettlesExactlyOnce` 继续验证旧 owner renew/fail/settle 全拒绝、第二 owner恰一次接管和 settlement。
- 旧逻辑会因 SQL 顺序仍为 job-before-Audit、completion 参数仍为旧时间而稳定使新增 scripted harness 失败；断言不是只迎合返回值。

### `IDENTITY-S10.4-REV-R01-F005` — **CLOSED**

- blocked-shutdown 核心负断言已完全移除 `time.Sleep`。
- `blockedLifecycleWorker.Done()` 在 lifecycle 的 Wait 已返回后关闭 `doneWaitEntered` 屏障（`lifecycle_test.go:713-745`）。
- 测试先等待 `waitReturned` 和 `doneWaitEntered`，此时 lifecycle 已调用 `Done()` 并将阻塞于未关闭的 done channel；随后确定性断言 Article、两个 listener、pool 的关闭计数均为 0（`:687-695`）。
- 只有测试显式关闭 worker done 后 lifecycle 才可继续，并断言最终关闭计数为 `1/2/1`（`:696-710`）。保留的 `time.After` 仅是释放 done 后防止测试永久挂起的 watchdog，不参与核心负断言。
- F002 的最终 completion CAS 边界、第二 owner/attempt、commit/rollback 已由新增 operator harness 覆盖；commit 三态和错误链回归也仍执行真实 settlement scripted transaction。
- 删除 Done 屏障或恢复旧 lifecycle 行为时，资源关闭顺序/屏障断言会失败；恢复旧 settlement 顺序或旧时间参数时 operator harness 会失败。

### `IDENTITY-S10.4-REV-R02-F001` — **CLOSED**

- rolled-back 分支使用 `errors.Join` 同时保留 `ErrPasswordResetSettlementRolledBack` 和 `%w` 包装的原始 `commitErr`（`password_reset_operator.go:237-241`）。
- unknown 分支同时保留 `ErrPasswordResetSettlementCommitUnknown`、原始 `commitErr` 与 `verifyErr`（`:242-247`）。
- `TestPasswordResetOperatorCommitAcknowledgementClassification` 对 rolled-back/unknown 均断言分类 sentinel 和 `commitErr` 可被 `errors.Is` 识别；unknown 还断言 `verifyErr` 保留（`password_reset_operator_test.go:392-446`）。
- committed 分支仍按既定合同归并为成功，不向 worker返回伪失败。

## 3. 两轮实际检查

### Round 1：三个 OPEN finding 与直接回归

- 逐语句检查 settlement 的 User/job 锁后 Clock 采样、业务写、Audit、最终 lease completion CAS、RowsAffected、rollback/commit 顺序。
- 检查 expiry 前/等于/之后、第二 owner/attempt、两个事务合计 commit 数和 scripted SQL 顺序。
- 检查 lifecycle Wait→Done 屏障的因果顺序、核心负断言是否仍依赖 wall-clock sleep、资源释放前后计数。
- 检查 commit committed/rolled-back/unknown 三态及 sentinel、commitErr、verifyErr 的完整错误链。
- 检查新增测试在恢复旧 SQL 顺序、旧时间采样、缺 Done 屏障或 `%v` 断链时是否会稳定失败。

### Round 2：关联回归与新增风险

- 有效复用并复核 R01/R02、design r44、test-plan r20、Tasks/Plan，以及 Reviewer Role、add-code-review Skill、checklist、abstraction principles 和适用 go-service profile。
- 反查 unchanged worker 的 Claim/Renew/Fail/takeover、permit 和 Stop/Wait 行为；反查 migration 0008 的 lease/job 约束。
- 反查 production Store/Gate、operator、worker、permit owner和双 worker lifecycle：本轮未改变 production wiring、schema、锁序、公共合同或 shutdown 合同。
- 检查固定八文件边界、staged 状态、格式、diff whitespace、遗留二进制及 S10.5 禁止边界。
- 未发现新增 correctness、安全、资源生命周期、架构边界或强制测试证据 finding。

## 4. Findings

**无新增 finding。**

R01/R02 状态链：

- R01 F001：CLOSED（R02）
- R01 F002：CLOSED（本 R03）
- R01 F003：CLOSED（R02）
- R01 F004：CLOSED（R02）
- R01 F005：CLOSED（本 R03）
- R02 F001：CLOSED（本 R03）

## 5. Reviewer 实际运行命令

专项普通：

```text
go test ./internal/identity ./cmd/han-server -run 'TestPasswordResetOperator(SettlementRechecksLeaseBeforeFinalCompletion|CommitAcknowledgementClassification)|TestRunLifecyclePasswordResetDeadlineWaitsForDoneBeforeClosingResources' -count=1
```

结果：PASS。

专项 race：

```text
go test -race ./internal/identity ./cmd/han-server -run 'TestPasswordResetOperator(SettlementRechecksLeaseBeforeFinalCompletion|CommitAcknowledgementClassification)|TestRunLifecyclePasswordResetDeadlineWaitsForDoneBeforeClosingResources' -count=1
```

结果：PASS。

专项重复：

```text
go test ./internal/identity ./cmd/han-server -run 'TestPasswordResetOperator(SettlementRechecksLeaseBeforeFinalCompletion|CommitAcknowledgementClassification)|TestRunLifecyclePasswordResetDeadlineWaitsForDoneBeforeClosingResources' -count=20
```

结果：PASS。

整卡定向普通：

```text
go test ./internal/identity ./cmd/han-server -run 'TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle' -count=1
```

结果：PASS。

整卡定向 race：

```text
go test -race ./internal/identity ./cmd/han-server -run 'TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle' -count=1
```

结果：PASS。

其他检查：

- 三个修复写文件 `gofmt -l`：无输出。
- 固定八文件 scoped `git diff --check`：PASS。
- 固定八文件 staged diff：无输出。
- 开始、写报告前固定八文件 SHA256：MATCH。
- 根目录 `han-server.exe`：不存在。

Lead 提供的同 revision 专项、整卡定向、gofmt、diff-check、哈希、staged 和二进制检查结果与 Reviewer 现场结果一致。

## 6. 未验证范围与剩余风险

- 按授权未运行真实 PostgreSQL、SMTP/TLS、部署、Host 或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- PostgreSQL 18 的真实多连接锁等待、真实 takeover 和 commit acknowledgement 丢失仍由既定专项验证，不由 scripted harness 冒充。
- 未作性能、容量或 production readiness 结论；五项生产实际值仍为 TBD。
- 本 PASS 仅适用于本报告固定八文件哈希及其有界直接消费者，不外推共享工作树的其他未提交内容，也不启动 S10.5。

## 7. 待决定、建议与 Gate

- 待 Lead 决定项：无新的产品、架构、schema、锁序或 shutdown 合同决定。
- 非阻断建议：无。
- 三个 R02 OPEN finding 全部 **CLOSED**，未发现新增 finding。
- **Code Review Gate：PASS**。

## 8. 结束哈希

报告完成后重新校验：固定八文件 SHA256 与第 1 节相同；分支仍为 `main`，HEAD 仍为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，根目录无 `han-server.exe`。
