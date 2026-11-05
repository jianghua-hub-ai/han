# identity-auth-S3.2 / F005 修复复审 r02

- Reviewer：Ampere（非作者）
- 日期：2026-10-22
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 复审对象：`IDENTITY-S3.2-F005-R01-F001`
- 结论：**F001 CLOSED；未发现本次修复引入的新 finding。**
- Task Gate：**`IDENTITY-S3.2-REV-F005` 仍为 OPEN / 受限；S3.2 与 B4 不得关闭。**
- 操作边界：被审代码全程只读；仅新增本报告。未修改 r01、代码、design、tasks 或 Plan，未执行任何 Git 操作。

## 1. 固定 revision

原报告：

| 文件 | SHA-256 |
|---|---|
| `.spec/wip/identity-auth/review-code-s3.2-f005-r01.md` | `23D9F391CAE0465F6ED2CBACE69D0CE7B1D84E0D41C4A655957FD58077FA5628` |

本次修复对象：

| 文件 | SHA-256 |
|---|---|
| `internal/identity/login.go` | `89B5537BC8A4D0446DD55655C87492E35787EFEC8DD923A031F0C7B930237871` |
| `internal/identity/login_test.go` | `17EB9D870700548464EE5D410ACBF861668DB9CEF7003AC6F331590B7349C7DD` |
| `internal/identity/login_integration_test.go` | `B76ADABD7F7A6146FDC9C666699714CF53EA1485C9E916D4F74E61D46C8337FC` |

保持 r01 不变的直接对象：

| 文件 | SHA-256 |
|---|---|
| `internal/identity/handler.go` | `DDCAC63DAF56BDFDA2BA296C6A6B9BA5FF04AC83D690FC452CA89CBAC6B4AA3E` |
| `internal/identity/service.go` | `8DC48D994C7B967FC586B4F7C4B70C6A99B9C3EF9DFD05BA018D278B086AA28F` |
| `internal/platform/http_security.go` | `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` |
| `internal/platform/http_security_test.go` | `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` |
| `internal/audit/store_integration_test.go` | `4A3CC84CE6B620724692D696B705D74BC39C8E1EB28F044EDD136E166FF6138C` |
| `cmd/han-server/main.go` | `5C7ADD02E1921447555441CDBDA13A247284409EBE58065FFB03F642D5EFF08B` |
| `cmd/han-server/main_test.go` | `85326107F9605F3353AEFEE983221345CCAB18CCBE0A844821E7B8DAE037B8C8` |

## 2. 两轮复审范围

### Round 1：F001 根因与最小修复

检查：

- `internal/identity/login.go:333-360` 的 attempt 时间生成与复用边界。
- `internal/identity/login.go:1954-1986` 的 prepared event 只读核对。
- `internal/identity/login_test.go:778-802` 的非 UTC、亚微秒输入规范化。
- `internal/identity/login_test.go:1068-1143` 的完整不可变字段、canonical metadata 和时间精确比较。

结果：

- `login.go:341` 在时间首次生成处执行 `clock.Now().UTC().Truncate(time.Microsecond)`；修复位于持久化事实的唯一来源边界，没有在比较阶段引入容差。
- `sameAuditEvent` 在 `login.go:1968-1986` 仍用 `time.Time.Equal` 精确比较；没有把相差不足一微秒或相差一微秒的事件放宽为相同。
- `login_test.go:778-802` 使用非 UTC 且纳秒尾数非零的输入，同时断言 UTC location、期望瞬间和整微秒。
- `login_test.go:1098-1106` 明确接受同一规范微秒值，并拒绝相差一微秒的值；其余 13 个字段及 metadata 差异仍逐项拒绝。

### Round 2：PostgreSQL 18 测试定义与剩余 F005 Gate

检查：

- `internal/identity/login_integration_test.go:22-145`：commit unknown 实际提交/实际回滚，fixture 使用非 UTC 亚微秒原始时间。
- `internal/identity/login_integration_test.go:147-270`：rollback unknown 且 Append 后的匹配行、无行、字段不一致三种实际状态。
- `internal/identity/login_integration_test.go:272-342`：完整 14 字段、nil/空可选字段、规范时间和 canonical metadata 的 PostgreSQL round-trip。
- `internal/identity/login_integration_test.go:618-670`：实际 commit/rollback fault adapter、DSN 和 PostgreSQL major 18 门禁。
- `internal/testenv/env.go:30-55`、`internal/identity/repository_integration_test.go:774-843`：缺 DSN 或非 PostgreSQL 18 时明确 `Fatal`，不是 `Skip`。
- r01 中 production composition 结论对应的未变 `cmd/han-server/main.go` / `main_test.go`。

结果：

- 新矩阵静态覆盖 r01 指出的 commit unknown、rollback unknown Append 后三态、完整事件 round-trip、nil/空、规范时间和 canonical metadata 缺口。
- 匹配行/无行/差异行分别核行数，并对存在的行再次完整读回比较；当前验证路径未补写事件，也未把差异行改写为 prepared event。
- integration 测试带 `integration` build tag；compile-only 明确显示 `[no tests to run]`，没有冒充 PostgreSQL 运行 PASS。
- 真正执行时，`RequirePostgresDSN` 缺值会 `t.Fatal`，数据库 major 非 18 也会 `t.Fatal`；没有 `Skip` 或静默降级。

## 3. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

无新增 MEDIUM finding。

### LOW

无。

## 4. 原 finding 处置

### `IDENTITY-S3.2-F005-R01-F001` — CLOSED

- **原触发**：attempt 保留亚微秒，而 PostgreSQL `timestamptz` round-trip 后只有微秒精度，导致实际已提交事件可能被精确比较误判为 unknown。
- **关闭证据**：
  - `internal/identity/login.go:341` 在首次生成处统一 UTC 微秒精度。
  - `internal/identity/login_test.go:778-802` 覆盖非 UTC、亚微秒输入。
  - `internal/identity/login_test.go:1068-1143` 保持严格比较，并证明一微秒差异仍被拒绝。
  - `internal/identity/login_integration_test.go:54-88` 将亚微秒 fixture 经 coordinator 规范化后用于 commit-unknown prepared event。
- **判定**：根因在正确边界以最小改动修复；测试没有通过放宽比较掩盖精度差异。

## 5. `IDENTITY-S3.2-REV-F005` 状态

**OPEN / 受限。**

分项状态：

1. **F001 本地实现与单元证据：完成。**
2. **r01 指出的 PostgreSQL 矩阵代码定义：已补齐且 integration tag 编译成立。**
3. **PostgreSQL 18 真实执行：仍 blocked-for-execution。** 本机 `HAN_TEST_POSTGRES_DSN` 未设置；本轮没有真实数据库提交状态证据，因此不得把 compile-only 记为 integration PASS。
4. **production composition：仍未完成。** `cmd/han-server/main.go` 与 r01 哈希一致，仍没有登录/受保护路由及 production logger、metrics、observer、failure auditor、正数有限预算和取消能力装配。
5. **Q13 与具体预算：仍为 design 已接受的上游剩余阻塞。** Reviewer 不替项目选择 backend、预算、阈值、dashboard、paging/runbook 或值守入口。

因此，本次关闭 F001 和补齐 PostgreSQL 测试定义，不足以关闭总 finding、S3.2 或 B4。

## 6. 实际命令与结果

未执行 Git 命令。

```text
Get-FileHash -Algorithm SHA256 <r01、修复对象、保持不变对象>
=> 全部与交接哈希一致

Get-Content / Select-String
=> 只读检查实现、单元测试、integration 矩阵、环境门禁和直接消费者

HAN_TEST_POSTGRES_DSN presence check（只输出 SET/UNSET）
=> UNSET

go test ./internal/identity -run \
  'TestLoginAttemptCoordinator_NormalizesOccurredAtToUTCMicroseconds|TestSameAuditEvent_RequiresAllImmutableFieldsAndCanonicalMetadata' \
  -count=1
=> PASS

go test -tags=integration ./internal/identity -run '^$' -count=1
=> PASS [no tests to run]；仅证明 integration tag 编译成立

gofmt -l internal/identity/login.go \
  internal/identity/login_test.go \
  internal/identity/login_integration_test.go
=> clean
```

Lead 提供的更广泛 test/race/vet/build/contract 结果作为交接证据记录，但不替代本 Reviewer 对上述对象的独立核查；本报告不把未实际执行的 PostgreSQL 18 范围记为 PASS。

## 7. 限制与 Gate

- PostgreSQL 18 commit/rollback 实际状态矩阵尚未运行，真实连接、事务和资源释放行为仍未验证。
- production composition 与真实 backend 未装配，无法验证生产 adapter 的取消遵从、有界返回和资源回收。
- `IDENTITY-S3.2-F005-R01-F001`：**CLOSED**
- 本次新增 finding：**0**
- `IDENTITY-S3.2-REV-F005`：**OPEN / 受限**
- S3.2 G4-task：**FAIL**
- B4：**不得关闭**
- 不更新 tasks/Plan，不关闭任务，不 commit/push。
