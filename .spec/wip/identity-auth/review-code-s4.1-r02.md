# identity-auth-S4.1 定点代码复审 r02

- **复审日期**：2026-10-23
- **Reviewer**：Raman（原非作者 Reviewer；与实现作者 Noether `01a16893-bc25-7a52-8532-4bd7e203565d` 职责隔离）
- **objectType**：code
- **仓库 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **任务**：`identity-auth-S4.1`
- **原报告**：`.spec/wip/identity-auth/review-code-s4.1-r01.md`，SHA-256 `4C51ADAF6AF86E89F3F77CC5E9848B58FFBA9F2FDF7A466A3B8AF1664EE54A0D`
- **结论**：原 `F001`～`F003` 全部 **CLOSED**；未发现相关新增缺陷；**Code Review Gate PASS**。

本结论只绑定本报告列出的六个固定代码对象，用于原 findings 的定点复审；不替代 Lead 的任务关闭决定，也不表示真实 PostgreSQL、浏览器、部署、E2E 或 production readiness 已完成。

## 1. 固定对象与漂移核验

复审开始及报告完成前均复核下列 SHA-256，结果与委托值一致：

| 文件 | SHA-256 |
|---|---|
| `internal/identity/bootstrap.go` | `AC4B164F2BC0024E67E2805905C5BB982ECAABD9563A6BFF67CDF98C32F593B6` |
| `internal/identity/bootstrap_test.go` | `915A0A9E4E0C71134212464194B9D609A385F14AF3F527BBA9F722B6A05051F4` |
| `internal/identity/handler.go` | `1E7200AF5A46509FB1CB31E58EAE598EC27C6F5A53E43BA2F06306924449E076` |
| `internal/identity/service.go` | `2A7BD8D786862B6DD61C424DE9747E12856E6DC8CDCC27F9E070CC5724BC414B` |
| `cmd/han-server/main.go` | `5DBACCB74EF15F53BCDCC1845ADE8B66327163F8817A84C708489C0082E61988` |
| `cmd/han-server/main_test.go` | `D83244E9B2F8277FDE485D7FA2DCD14ADB570F9A3FD53361DB48550C903DDA77` |

共享工作树含其他人的未提交成果；本轮未回滚、覆盖、格式化或修改任何代码及范围外文件，只新增本报告。

## 2. 复审依据与两轮范围

实际读取并应用：

- Reviewer Role、`add-code-review`；
- code-review-checklist、abstraction、coding、no-hardcode；
- Go service coding/no-hardcode/security/observability/tracing profile，以及适用的核心 observability/tracing 规则；
- tasks S4.1、Plan B5A/B5B、identity boundary r12、architecture r24、design r28、test-plan r11、design Review PASS；
- 原报告 r01、新六文件全文及必要直接消费者 `internal/audit/store.go`、`internal/session/store.go` 和 production composition。

### Round 1：原 findings 与固定代码定点复审

1. 核对锁等待后的权威 credential 重判、错误优先级、密码 verifier 调用和当前 Session 撤销；
2. 核对 403、credential-binding 401、当前密码 422、新密码策略 422、持久化失败和成功的 AuditEvent；
3. 核对 Audit Append 失败时的 rollback、成功状态与 Audit 的 caller-owned transaction；
4. 核对 Handler 前置失败审计接缝与 Service 审计责任互斥；
5. 核对 canonical action、敏感字段 denylist 及确定性并发回归。

### Round 2：有界消费者与相关回归复审

沿 `router → OriginGuard → HTTPAuthorizer → Handler → Service → repository/session/audit` 核对：

- exact route、Trace/Origin/Session/CSRF/授权顺序未被本次修订改变；
- `session.Store.Revoke/Rotate` 与其他 Session 撤销继续使用调用方事务；
- `audit.Store.Append` 把事件写入调用方事务；
- 持久化失败先回滚业务事务，再用独立事务追加失败事件；
- production composition 仍装配真实 password repository、session store 和 audit store；
- 未发现旧 action `identity.password_change` 的 Go 代码引用。

## 3. 原 findings 处置

### IDENTITY-S4.1-R01-F001 — CLOSED

**原问题**：forced/temporary 请求等待 User 行锁后，权威 credential 已变为 permanent 时错误返回 403，且不撤销当前 Session。

**关闭证据**：

- `internal/identity/service.go:409-425`：锁后读到 permanent 非 initial-admin 时，只有原请求本就不具 forced/initial-admin 资格才是 403；已通过主体检查的 forced 请求进入 stale rejection。
- `internal/identity/service.go:619-687`：stale rejection 固定返回 `auth.session_required`，只用当前 token 调用 `Revoke`，并把失败审计与撤销一并提交。
- `internal/identity/bootstrap_test.go:528-586`：两个事务通过 channel 和互斥锁确定顺序，无 sleep；A 先提交 permanent，B 后获锁并断言 401、总 verifier 仅 A 的 1 次、只撤销 `token-b`、不撤销 `token-a`，成功轮换/其他 Session 撤销仅发生 1 次。
- `internal/identity/bootstrap_test.go:150-197`：普通非强制 permanent 主体仍为 403，verifier 为 0。

**Reviewer 验证**：并发回归单独连续运行 50 次通过；相关 race 定向通过。

### IDENTITY-S4.1-R01-F002 — CLOSED

**原问题**：403、credential-binding 401、两类 422 和持久化失败缺少 AuditEvent；Session/credential 状态变化与 Audit 原子性及 Handler 接缝未满足。

**关闭证据**：

- `internal/identity/service.go:319-329`、`576-617`：Service 前置 403 通过受控失败审计事务记录一次事件。
- `internal/identity/handler.go:489-520`、`553-574`：Handler 在读取非法 body 或缺 Cookie 等 Service 前置失败时调用唯一审计接缝；一旦进入 Service，Handler 只写 Service 结果，不二次审计。
- `internal/identity/service.go:352-379`：当前密码和新密码策略等稳定失败在原事务追加失败事件后提交。
- `internal/identity/service.go:619-687`：credential-binding 401 的当前 Session 撤销与失败 Audit 在同一事务；Append/commit 失败不返回原 401 成功后置。
- `internal/identity/service.go:380-400`：业务持久化失败先确认 rollback，再使用独立事务记录 `platform.database_unavailable`；不在失败事务中追加。
- `internal/identity/service.go:478-569`：密码摘要、TemporaryCredential 消费、当前 Session 轮换、其他 Session 撤销及成功 Audit 共用同一事务；Audit Append 失败由 defer rollback，不能 commit 部分状态。
- `internal/identity/bootstrap_test.go:150-197,199-244,246-380,383-473,475-526`：覆盖 403、401、当前密码 422、新密码策略 422、成功、Audit Append 失败 rollback、持久化失败 rollback 后独立审计和 Handler 前置接缝。
- `internal/identity/bootstrap_test.go:650-691`：精确断言 action、actor kind/user/role、target、outcome、error code、trace，并扫描密码、Cookie、CSRF 和 digest 标记；生产事件未设置 metadata/source 等额外载荷。
- `internal/audit/store.go:50-94` 与 `internal/session/store.go:160-211`：消费者分别使用传入 `pgx.Tx`/executor，支持上述同事务保证。

**验证结果**：相关测试及 race 均通过。未发现双审计路径或修订引入的部分状态提交路径。

### IDENTITY-S4.1-R01-F003 — CLOSED

**原问题**：成功事件使用非合同 action `identity.password_change`。

**关闭证据**：

- `internal/identity/service.go:176` 定义唯一常量 `identity.password.change`；
- `internal/identity/service.go:690-717` 的成功和失败事件共用该常量；
- `internal/identity/bootstrap_test.go:650-675` 精确断言 canonical action 及完整关键字段；
- Go 代码范围搜索未发现 `identity.password_change`。

## 4. 新增 findings

### CRITICAL / HIGH / MEDIUM / LOW

无相关新增 finding。

## 5. Reviewer 实际验证

```text
go test ./internal/identity -run "TestBootstrap|TestInitialAdmin|TestOnboarding" -count=1 -timeout=60s
go test ./cmd/han-server -run "Test.*(Password|Router|Bootstrap|ProductionBusiness)" -count=1 -timeout=60s
go test -race ./internal/identity -run "TestOnboardingConcurrentPasswordChangeStaleWaiterReturnsSessionRequired|TestOnboardingPasswordChange(AuditAppendFailureRollsBack|PersistenceFailureAuditsAfterRollback|StaleCredentialRevokesOnlyCurrentSession|SuccessPersistsPermanentAndRotatesSessions|RejectsUnauthorizedPrincipalBeforeVerifier)|TestOnboardingPasswordValidationReturnsExactFieldError" -count=1 -timeout=90s
go test ./internal/identity -run "^TestOnboardingConcurrentPasswordChangeStaleWaiterReturnsSessionRequired$" -count=50 -timeout=60s
```

四条命令均 PASS。

Lead 报告的 identity/server/article 定向、gofmt、`go mod verify`、vet、build、contract、race、全量测试和 scoped diff-check PASS 仅作为外部已有证据；本报告不冒充 Reviewer 独立复跑这些全量检查。

## 6. 限制与未验证风险

1. 未连接真实 PostgreSQL，未验证真实 `FOR UPDATE` 等待、事务 commit/rollback 故障注入及数据库层 Audit Append rollback；本轮以生产事务调用链、测试替身和 race 回归为证据。
2. 未运行浏览器、部署、真实 Host 或 E2E。
3. 未复跑 Lead 已完成的 article、全量 contract/vet/build/race/all tests。
4. 共享工作树存在范围外未提交成果；结论严格绑定本报告六个 SHA-256，不评价范围外改动。

## 7. Code Review Gate

### Code Review Gate：PASS

`IDENTITY-S4.1-R01-F001`、`F002`、`F003` 均已按原复审条件关闭，相关定向与 race 回归通过，未发现相关新增缺陷。

该 PASS 只表示固定修订通过本次代码复审 Gate；任务关闭、后续交接、真实 PostgreSQL/E2E 和 production readiness 仍由 Lead/相应责任人依据独立证据决定。
