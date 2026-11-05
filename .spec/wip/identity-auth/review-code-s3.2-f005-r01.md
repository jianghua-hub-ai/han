# identity-auth-S3.2 / `IDENTITY-S3.2-REV-F005` 独立代码复审

- Reviewer：Ampere（非作者）
- 日期：2026-10-22
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：**`IDENTITY-S3.2-REV-F005` OPEN / 受限；G4-task FAIL，不得关闭 S3.2 或 B4。**
- 操作边界：除本报告外全程只读；未修改、还原、格式化代码/设计/tasks/Plan，未 commit/push/PR/merge。

## 1. 固定对象

评审开始时以下 SHA-256 与交接一致：

| 文件 | SHA-256 |
|---|---|
| `internal/identity/handler.go` | `DDCAC63DAF56BDFDA2BA296C6A6B9BA5FF04AC83D690FC452CA89CBAC6B4AA3E` |
| `internal/identity/service.go` | `8DC48D994C7B967FC586B4F7C4B70C6A99B9C3EF9DFD05BA018D278B086AA28F` |
| `internal/identity/login.go` | `69DA80346DF0AB32A180B168067A84B77152535D26D333ED0F7DC4500A348369` |
| `internal/identity/login_test.go` | `B981BDACC4FE9467BE71B5FA9BE48C813F5F9CAF5E9896B94F16741FE57C19D1` |
| `internal/identity/login_integration_test.go` | `BB24F41782FC03A83CFD78B894319D61A127AFF26F2D997778B594A106F69B6D` |
| `internal/platform/http_security.go` | `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` |
| `internal/platform/http_security_test.go` | `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` |
| `internal/audit/store_integration_test.go` | `4A3CC84CE6B620724692D696B705D74BC39C8E1EB28F044EDD136E166FF6138C` |
| `cmd/han-server/main.go`（授权但未改） | `5C7ADD02E1921447555441CDBDA13A247284409EBE58065FFB03F642D5EFF08B` |
| `cmd/han-server/main_test.go`（授权但未改） | `85326107F9605F3353AEFEE983221345CCAB18CCBE0A844821E7B8DAE037B8C8` |

只读消费者：

- `internal/platform/logging.go`：`F362D36954DF0119D5DC25040640C796069CC8E178A1D92B828C6D6149CF30A9`
- `internal/audit/store.go`：`091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F`
- 禁止文件 `internal/identity/repository_integration_test.go`：`FE6F63FF1C89E68CD61EA24B94D51D7FDED8947BEDA6FDB3CCEB7837F3A07448`

已知 F005 前置快照来自本会话交接：`handler.go` `9543F87D...270714`、`service.go` `69E0AA8A...8A2D45`、`login.go` `FE0B90A7...CB345F2`、`login_test.go` `C2814154...B4AD2`、`login_integration_test.go` `DFC2FC13...A8D78C`。当前共享工作树中的这些对象均为 untracked，普通 `git diff` 不能重建内容差异；platform/audit 三个 F005 前置文件的完整 G0 SHA 未在可读交接制品中找到，因此不臆造。

## 2. 两轮实际检查范围

### Round 1：直接实现、事务、安全与测试

检查了：

- `handler.go` 的 source 前 ID/time/guard、Handler/Service 审计责任转移、五 outcome、invalid JSON、唯一响应写入后 finalizer、Cookie/response-write 分类和 observation budget。
- `login.go` 的 `insert_failure/verify_prepared`、rollback unknown Append 前/后、commit unknown 只读核对、完整 AuditEvent 比较、事务关闭、Session/Audit 顺序、logger/metrics allowlist。
- `http_security.go` 的四种 Origin 原因、唯一 403 写后 observer、WARN→counter fan-out、request context/有限预算、失败后不重试和不二次响应。
- 三个测试文件中的确定性顺序、单次 guard、阻塞至取消、敏感字段、事务不确定性及 PostgreSQL fault adapter。

直接路径整体符合项：

- ID/time/guard 在 source provider 前创建；Handler/Service 共享一次性 guard。
- rollback unknown 在 Append 前不调用 auditor；Append 后及 commit unknown 只使用 `verify_prepared`。
- Handler 与 OriginGuard 都在唯一响应写入尝试后同步观察，不重写第二响应。
- 登录日志/指标及 Origin 日志/指标均使用固定低基数字段；未发现用户名、密码、Cookie/CSRF、Origin/Header/IP 原值进入普通日志或指标。
- 构造器拒绝 nil observer/coordinator/logger/metrics 和非正预算；timer 在同步调用后 cancel。

### Round 2：真实消费者与装配

沿以下链路检查：

- `handler → service/login → audit.Store.Get/Append → session store`
- `OriginGuard → platform.LoggerWithTrace`
- `NewHandler`、`NewLoginHandler`、`NewLoginOutcomeObserver`、`NewOriginGuard`、`NewOriginRejectionObserver`、`NewPostgresLoginFailureAuditor` 的全部 Go 消费者
- `cmd/han-server/main.go` / `main_test.go` production composition

结论：

- `NewHandler` 注册构造器保持兼容；登录使用独立 `NewLoginHandler`。当前生产代码没有这些登录/Origin observer 构造器的消费者。
- `audit.Store.Get` 读取全部 14 个不可变字段；metadata 经 JSON 解码后参与 canonical 比较。
- `main.go:110-114` 仍只启动空 `http.ServeMux`，没有登录路由、Origin 受保护路由、production logger/metrics、failure auditor、observer 或预算装配。

## 3. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

#### `IDENTITY-S3.2-F005-R01-F001` — `occurred_at` 未规范到 PostgreSQL 微秒精度，实际已提交事件会被误判为 unknown

- **位置**：
  - `internal/identity/login.go:337-348`
  - `internal/identity/login.go:1968-1986`
  - 隐藏该问题的 fixture：`internal/identity/login_integration_test.go:428-453`、`internal/identity/login_test.go:1042-1106`
- **触发**：`LoginAttemptCoordinator.Begin` 仅执行 `clock.Now().UTC()`，保留 Go 纳秒；`audit_events.occurred_at` 经 pgx/PostgreSQL 以微秒精度持久化。commit 返回 unknown、数据库实际提交后，读回时间会丢弃亚微秒部分，而 `sameAuditEvent` 使用 `time.Time.Equal` 精确比较。
- **影响**：真实 AuditEvent 已提交且其他 13 个字段完全相同，仍可能得到 `audit_persisted=unknown`，违反 I-F005-02/06 的“实际已提交且完整全等为 yes”。这是 fail-closed 误判，不会产生盲写，但会丢失应可证明的审计状态并制造错误内部告警/处置歧义。
- **最小修复**：在 attempt 时间首次生成处统一形成一个 `UTC().Truncate(time.Microsecond)` 值，并让原事务、prepared event 与核对都复用该值；不要在比较时用宽松时间容差掩盖非规范输入。
- **验证**：
  1. 单元用亚微秒时间（例如纳秒尾数非 000）证明 coordinator 产出 UTC 微秒值。
  2. PostgreSQL 18 commit-unknown/实际提交测试使用亚微秒输入，断言读回完整事件为 `yes`、总行数 1。
  3. I-F005-06 增加规范时间同值与 1 微秒异值用例。

#### `IDENTITY-S3.2-REV-F005` — 原 finding 仍 OPEN / 受限：PostgreSQL 18 证据矩阵与 production composition 未完成

- **位置**：
  - `internal/identity/login_integration_test.go:22-114`
  - `internal/audit/store_integration_test.go:24-149`
  - `cmd/han-server/main.go:89-119`
  - `cmd/han-server/main_test.go:20-325`
- **触发**：
  1. 当前 identity integration 只提供 commit unknown 的“实际提交/实际回滚”两例；没有任务卡要求的 rollback unknown Append 后 PostgreSQL 行/无行/字段不一致矩阵，也没有 14 字段 nil/空值/规范时间的 PostgreSQL 矩阵及 I-F005-08/12 状态快照。audit integration 仍是通用 append-only round-trip，不替代这些登录合同。
  2. production main 使用空 mux，未注册登录或受保护路由，也没有 logger/metrics backend、observer、failure auditor、正数有限预算或 adapter 取消保证的 composition test。
  3. `HAN_TEST_POSTGRES_DSN` 未设置，PostgreSQL 18 命令本轮严格 blocked-for-execution。
- **影响**：即使现有非 PG 单元、race、vet、build 全通过，仍不能证明 I-F005-02/03/05/06/08/12，也不能证明生产路由在完整依赖下可装配和有界返回；原 F005 不满足关闭条件。
- **最小修复**：
  - 在已授权 integration 文件内补齐任务卡登记的 PostgreSQL 18 矩阵与状态/零盲写断言。
  - 待 Q13 backend、预算输入及生产 wiring 获批准后，在唯一 production composition root 显式装配两类 observer，并用启动测试证明缺依赖/无取消保证/无有效预算时不注册相应路由。
  - Reviewer 不替项目选择 backend、预算、阈值、dashboard、paging/runbook 或值守入口。
- **验证**：
  - 设置但不输出 `HAN_TEST_POSTGRES_DSN`，先证明数据库 major=18，再执行任务卡两条 integration 命令及 integration race。
  - composition 启动测试覆盖完整装配与每个缺项；阻塞 sink 必须由被测 deadline/cancel 自行收敛，并核 goroutine/timer/连接/临时资源回到基线。

### LOW

无新增 LOW finding。

## 4. Production fail-closed 判定

1. **当前不存在“已完整装配但缺依赖仍开放”的生产登录路由。** `main.go:110-114` 使用空 mux，因此登录和受保护路由根本未注册；按设计“缺依赖时不注册对应路由”的负向安全结果看，它是 fail closed。
2. **这不等于 production composition 已完成。** 当前只新增了可测试端口/构造器；没有真实登录路由、OriginGuard、failure auditor、logger/metrics、预算及取消能力装配，不能宣称生产就绪或关闭 F005。
3. **分类**：production backend/预算/Q13 是 design 已接受的剩余上游阻塞；main 未接线和 composition test 未交付是原 F005 尚未完成的代码任务状态，不是 Reviewer 应替项目选型的新需求。

## 5. 待决定、非阻断建议与限制

### 待决定 / 上游 blocked

- Q13：production structured-log sink/留存、metrics backend/Histogram buckets、告警阈值、dashboard、paging/runbook、值守人与处置入口。
- 两类 observer 的具体正数有限预算；Reviewer 仅确认必须显式、正数、有限，不提供数值。

### 非阻断建议

- production composition test 应使用明确具备 context cancellation 合同的 adapter，而不是仅凭任意接口值非 nil 推断“可取消”。
- integration 失败信息继续只打印字段名/稳定状态，不打印 metadata、DSN 或原始 cause。

### 未验证风险

- `HAN_TEST_POSTGRES_DSN` 未设置；未执行 PostgreSQL 18 的 F005 integration/integration-race，不能给该范围 PASS。
- 未启动真实 production logger/metrics backend；阻塞/取消与连接资源只能由 deterministic fake 证明，真实 backend 行为未知。
- 当前对象是 untracked 精确快照，普通 Git diff 无法证明内容前后差异；本报告依赖固定 SHA、全文读取和消费者搜索。

## 6. 实际命令与结果

```text
git branch --show-current
git rev-parse HEAD
Get-FileHash -Algorithm SHA256 <固定对象与消费者>
git status --short --branch -- <固定对象>
rg -n ... <构造器、消费者、设计、任务、测试索引>
Get-Content -Encoding UTF8 ... <实现/测试/上游全文与精确范围>

HAN_TEST_POSTGRES_DSN presence check（仅输出 SET/UNSET，不输出值）
=> UNSET

go test ./internal/identity -run 'TestLogin|TestCredentialMode|TestLoginAbuse' -count=1
=> PASS

go test ./internal/identity -count=1
=> PASS

go test ./internal/identity ./internal/platform ./cmd/han-server -count=1
=> PASS

go test -race ./internal/identity ./internal/platform ./cmd/han-server -count=1
=> PASS

gofmt -l <十个范围文件>
=> clean

go mod verify
=> PASS

go vet ./internal/identity ./internal/platform ./cmd/han-server
=> PASS

go build ./...
=> PASS
```

PostgreSQL 18 两条任务命令未运行，状态为 **blocked-for-execution**，不得记为 Skip 或 PASS。

## 7. Gate

- 原 finding：`IDENTITY-S3.2-REV-F005` = **OPEN / 受限**
- 新增 HIGH：0
- 新增 MEDIUM：1（`IDENTITY-S3.2-F005-R01-F001`）
- S3.2 G4-task：**FAIL**
- B4：**不得关闭**
- 不更新 tasks/Plan，不关闭任务，不 commit/push。
