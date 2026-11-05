# identity-auth-S8.1 Code Review R02

- 日期：2026-10-24
- Reviewer：add-reviewer（当前会话直接执行，非实现作者）
- 实现作者：Backend-Dev James `01a16dc2-80e8-7603-b685-ea20cfe4887d`
- objectType：code / 定点复审
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 批次 / 任务：B6B / `identity-auth-S8.1`
- 原报告：`.spec/wip/identity-auth/review-code-s8.1-r01.md`
- 原报告 SHA256：`3E597ED60AA7523EF62B20B2FFE32586726E607C578108EE7762E1C5F4303109`
- 结论：**Code Review Gate: PASS**

## 1. 固定修订

| 文件 | R02 SHA256 | 相对 R01 |
|---|---|---|
| `internal/identity/handler.go` | `40E4D8D6D4BE5497B3DDD7C513DA144AB105EDC0A5DADF068BDFDF06020450C6` | 未变化 |
| `internal/identity/account_status.go` | `C03B8D155580CECE26707863D24062FC54B6C3707F2C006F38C3DAB408C00FA0` | 已返工 |
| `internal/identity/article_reader.go` | `D9315AFA411CB8C4612459C8E97979BD2274780A7B7B32D78F3AB735C3F82B6B` | 已返工 |
| `internal/identity/account_status_test.go` | `5E349230E58FE6F8EF1C899559B10DB9AD47050D9BD54660EA31B65599F26CCA` | 已返工 |
| `internal/identity/article_reader_test.go` | `C6D2B246236D24FC27C9E2C66413DE03999EA98D61CD61B2054D0152206C28AC` | 已返工 |

开始复审时实际哈希与 Lead 固定值完全一致。工作树其余大量合法既有内容保持不动。

## 2. 复审范围

本轮只复核 R01 的 F001～F004、对应回归及返工是否引入新 blocker/major/minor：

1. 先逐项核对原 finding 的根因、修复位置和回归断言；
2. 再沿受影响的事务顺序、公共错误、测试替身和 Article 只读接缝做有界回归检查；
3. 未重新扩展为全仓或生产接线评审。

## 3. 原 Findings 复审

### IDENTITY-S8.1-REV-R01-F001 — CLOSED

**原问题**：stale unrestricted Principal 对应的当前 actor 已为 `recovery_locked` 时仍可能通过事务内授权。

**修复证据**：

- `internal/identity/account_status.go:295-325`
  - 事务内锁定最新 actor 后调用 `accountStatusCredentialModeAllowed`；
  - allowlist 仅允许 `CredentialModePermanent`，因此 `recovery_locked`、bootstrap、temporary、recovery_bootstrap 均 fail closed。
- `internal/identity/account_status_test.go:127-205`
  - `CurrentActorEligibilityMatrix` 覆盖最新 role、status、initial-admin fact、四类非 permanent credential、`recovery_required`、onboarding 和 deleted marker；
  - `recovery locked credential with stale unrestricted principal` 明确使用 stale unrestricted initial-admin Principal；
  - 断言只锁 actor 一次，target 不再读取，update/revoke/audit 均为 0，commit 为 0、rollback 为 1。

**复审结论**：根因已消除，安全失败路径与零越界副作用已有针对性回归。

### IDENTITY-S8.1-REV-R01-F002 — CLOSED

**原问题**：initial admin 作为治理目标时错误返回通用 `auth.forbidden`。

**修复证据**：

- `internal/identity/account_status.go:199-204,384-389`
  - `target.IsInitialAdmin` 返回独立的 `accountStatusInitialAdminProtectedError`；
  - PublicError 使用 canonical `platform.CodeIdentityInitialAdminProtected`。
- `internal/identity/account_status_test.go:207-282`
  - Service 权限矩阵断言 initial admin 目标返回 `identity.initial_admin_protected`，且 update/revoke/audit 为 0并 rollback。
- `internal/identity/account_status_test.go:559-653`
  - HTTP 回归断言 403、canonical code、canonical 中文 message 和当前 trace ID。

**复审结论**：Service 与 HTTP 两层稳定公共错误合同均已恢复。

### IDENTITY-S8.1-REV-R01-F003 — CLOSED

**原问题**：最新资格、关键事务失败、HTTP 非法目标和 Article rows 行为的自动化矩阵不足。

**修复证据**：

- 最新 actor 资格：
  - `internal/identity/account_status_test.go:127-205` 覆盖 role/status/initial-admin/credential/recovery/onboarding/deleted 的权威状态漂移及零后续副作用。
- 事务与依赖失败：
  - `internal/identity/account_status_test.go:284-464` 覆盖 Begin 失败、nil Tx、actor/target missing、actor/target lock error、零时钟、ID error/empty、update、session revoke、audit 和 commit 失败；
  - 每例断言公开 code、lock/update/revoke/audit/ID 调用数以及 commit/rollback，证明失败后不继续执行后续副作用。
- path/UUID 与公开 envelope：
  - `internal/identity/account_status_test.go:559-653` 覆盖多余路径段、非法 UUID 和 initial-admin 专用错误；
  - 非法路径在 Handler 前置拒绝；非法 UUID 经真实 Service 验证后稳定返回 404，而测试 transactor 无可用事务，若错误进入 Begin 将无法得到该结果。
- Article 接缝：
  - `internal/identity/article_reader_test.go:16-64` 断言输入去重后的 UUID 数组、单次只读查询和成功 Close；
  - `internal/identity/article_reader_test.go:82-138` 覆盖非法输入、Query error、非法状态、Scan error、`rows.Err()`、冲突重复行及失败 Close。

这些用例仍是本卡范围内的确定性单元/HTTP 测试；真实 PostgreSQL 锁竞争和物理回滚继续由已登记的 `identity-auth-S1.6` 承担，未发生任务越界。

**复审结论**：R01 指定的测试缺口均已有可执行行为断言，不再只验证 mock 内部状态。

### IDENTITY-S8.1-REV-R01-F004 — CLOSED

**原问题**：`SetAccountStatus` 与 `GetAuthorStates` 超过 go-service 强制 50 行上限。

**修复证据**：

- `internal/identity/account_status.go:84-115`：`SetAccountStatus` 为 33 行（含签名和空行）。
- `internal/identity/article_reader.go:39-56`：`GetAuthorStates` 为 19 行（含签名和空行）。
- 新增 helper 均为同文件内的单一事务步骤、授权 allowlist、ID 规范化或 rows 扫描，没有新增公共接口、依赖或跨模块抽象。
- 状态机、事务顺序、只 suspend 撤销 Session、no-change Audit、Article 只读 SQL和公开响应均保持原行为，并由原有及新增测试覆盖。

**复审结论**：两个公开核心函数均低于 50 行；拆分局部、职责明确，未扩大架构边界。

## 4. 新问题检查

未发现由本次返工引入的新 blocker、major 或 minor。

重点确认：

- actor 授权仍在 target 锁定和任何业务写入之前完成；
- self、target role mismatch、initial admin、deleted 的判定顺序未产生信息泄露或行为回退；
- 状态更新、仅 suspend 的 Session 撤销和 Audit 仍使用同一 Tx；
- commit 失败不继续产生业务调用；其他事务内失败由 defer rollback 收敛；
- Article 拆分仍只执行一次批量 SELECT，并在成功与失败路径 Close rows；
- Handler 文件未变化，既有 users/administrators、suspend/restore 路径合同未漂移。

## 5. Reviewer 实际命令

```text
git status --short --branch
git rev-parse HEAD
git branch --show-current
Get-FileHash -Algorithm SHA256 <五个固定文件和 R01 报告>
go test ./internal/identity -run 'TestAccountStatus|TestGetAuthorStates' -count=3
go test -race ./internal/identity -run 'TestAccountStatus|TestGetAuthorStates' -count=1
go test ./internal/identity -run '^(TestAccountStatus_CurrentActorEligibilityMatrix|TestAccountStatus_PermissionAndTerminalStateFailures|TestAccountStatus_DependencyFailureMatrix|TestAccountStatusHandler_RejectsMalformedTargetsAndWritesProtectedEnvelope|TestGetAuthorStates_.*)$' -count=10
go vet ./internal/identity
gofmt -l -- <五个固定文件>
git diff --check -- <五个固定文件>
PowerShell 函数起止行统计（account_status.go、article_reader.go）
```

结果：

- 定向测试连续 3 次通过；
- 定向 race 通过；
- 修复重点用例连续 10 次通过；
- `go vet ./internal/identity` 通过；
- `gofmt -l` 无输出；
- scoped `git diff --check` 无输出；
- 分支和 HEAD 符合固定对象。

Reviewer 未读取、打印或使用 `HAN_TEST_POSTGRES_DSN`。

## 6. 限制与剩余风险

- 未连接真实 PostgreSQL，未在本轮复验真实 `FOR UPDATE` 竞争、物理事务回滚或旧 Session 下一请求失效；这些仍由 `identity-auth-S1.6` 闭环。
- 未重新运行 Lead 已完成的全仓检查，也未执行浏览器、部署或 E2E。
- production composition 接线仍属于既定后续任务，不因本次 R02 PASS 被视为已完成。
- 结论仅绑定本报告列出的五文件 SHA256、R01 报告哈希及当前有界消费者状态。

## 7. Gate

| 项目 | 状态 |
|---|---|
| F001 | CLOSED |
| F002 | CLOSED |
| F003 | CLOSED |
| F004 | CLOSED |
| 新 blocker | 0 |
| 新 major | 0 |
| 新 minor | 0 |

**Code Review Gate: PASS**

该独立结论不代替 Lead 对实际工作树、交付范围和后续任务的最终确认。
