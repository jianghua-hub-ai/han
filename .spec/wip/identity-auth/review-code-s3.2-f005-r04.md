# identity-auth S3.2 / F005 独立代码评审 r04

- Reviewer：Ampere（非作者）
- 阶段：development-delivery / code review
- 日期：2026-10-23
- 仓库：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 对象：accepted design r26 / test-plan r08 的 I-F005-13～29 实现 revision
- 总结：**FAIL；4 个 MEDIUM finding；`IDENTITY-S3.2-REV-F005` 保持 OPEN。**

本结论只评价固定代码对象。它不关闭 `identity-auth-S3.2`、B4，不代表 target-profile、production observability、部署或 E2E ready。

## 1. 固定对象

评审开始与报告写入前均以 SHA256 固定以下十八文件；两次结果一致：

| 文件 | SHA256 | 状态 |
|---|---|---|
| `internal/identity/handler.go` | `5B8B012A3F855C98DBBB10AA34050E15320615F3B36321A84B850B87990B2D36` | changed |
| `internal/identity/service.go` | `8DC48D994C7B967FC586B4F7C4B70C6A99B9C3EF9DFD05BA018D278B086AA28F` | baseline |
| `internal/identity/login.go` | `E9DD10FC26227F59976ECBB57FBA95692FD3DA3C43BA8305A6054024BD212881` | changed |
| `internal/identity/login_test.go` | `941C792EAF7C5B2635864461BAAB62AF6B9155DB873EE780F9094EE1DE89D3A1` | changed |
| `internal/identity/login_integration_test.go` | `1D2DA4502D640689767D1C9D8D013576E73A266CD4D649F1044CBB23EC1C0DE3` | baseline |
| `internal/platform/http_security.go` | `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` | baseline |
| `internal/platform/http_security_test.go` | `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` | baseline |
| `internal/platform/metrics.go` | `517B232D76C877FF335354E95668EC1D84B84385C017FCD45A4C046C6E4BC33A` | baseline |
| `internal/platform/metrics_test.go` | `FE97076376DBBABAA5B256BEFB92DAB527372AA652500F25EE0246B0D572EFB4` | baseline |
| `internal/platform/errors.go` | `99B1094A6C15A6E86E45B43928F8AC1B4F3A4EE810857BE598E9AA4336D968E3` | changed |
| `internal/platform/errors_test.go` | `5C8CEA67B738BE1AA11FB4AE206907604BEE7CC2CC0C646891CF4EF35A6D9237` | changed |
| `internal/security/argon2.go` | `BF6B6E63D315832001A010C77FCBA2341C2E1FDBEA1E0D6471B15BC3B3D406B7` | changed |
| `internal/security/argon2_test.go` | `32206B8089AF7A1645943ADC2841F458551EFA5CE7DB0D98B6B1FCA4222C937C` | changed |
| `internal/audit/store_integration_test.go` | `4A3CC84CE6B620724692D696B705D74BC39C8E1EB28F044EDD136E166FF6138C` | baseline |
| `cmd/han-server/main.go` | `944730C5C74AAB14E6BECD20FDD0B5346126DB0A38FDB7C9B0317110A581CF8E` | changed |
| `cmd/han-server/main_test.go` | `A6EDCCC58C129CEB81783E6B95150F2119CDBE158554CFD5A23B261B0C476675` | changed |
| `cmd/han-server/lifecycle.go` | `182485F059254AD2541F4A97F7C05AC6C20E857B475B9896CB43C0703260F082` | baseline |
| `cmd/han-server/lifecycle_test.go` | `9F68060F66EFE480D1AD27422F88753485C12024CE0E9953D3EF7876031FFC0F` | baseline |

未发现对象漂移、第十九文件、新依赖、schema/migration、ADR 或公共契约变化。

## 2. 评审依据

实际读取并核对：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- code-review-checklist、abstraction、coding、no-hardcode、observability、tracing、tech-stack compliance、安全红线
- `go-service` 的 coding/no-hardcode/security/observability/tracing profile
- tasks S3.2（SHA256 `2BF2018...C3B2E4`）与 Plan Context Pack I（`23B7BEC...2D88D0`）
- accepted design r26（`04FFC97C...A00217`）、test-plan r08（`C95518C5...AD23D`）、architecture r23、ADR-0016/0022
- planning review r07（`DDA81464...FA15F9`）
- F005 code review r01～r03 及原 F001～F004 关闭证据
- 十八文件全文，以及 `internal/platform/logging.go`、`internal/audit/store.go` 等直接消费者。

事实、设计合同与本次 Reviewer disposition 分开判断；未把规划 PASS、Lead 测试 PASS 或历史 PG18 PASS外推成当前代码 Review PASS。

## 3. 两轮检查

### Round 1：九个实际修改文件

完整检查 `handler.go`、`login.go`、`login_test.go`、`errors.go/_test.go`、`argon2.go/_test.go`、`main.go/_test.go`：

- 23 项配置、无默认及主要范围/交叉约束；
- production factory、prepared dependency、pool/listener 失败路径；
- process-wide non-blocking Argon2 Gate、typed capacity error、429 carrier/Header；
- recovery-bootstrap 前置状态与 adapter/Verify 顺序；
- RemoteAddr/XFF、可信代理、HMAC 来源摘要；
- Trace/router/OriginGuard/POST 顺序及 JSON 404/405；
- secret/DSN/PII/error cause、资源释放、取消和并发；
- 新增测试是否真实覆盖 test-plan，而非仅匹配实现。

除下列 findings 外，生产主路径未见新的 CRITICAL/HIGH 缺陷：真实 factory 已接入；Gate 为非阻塞且 `defer` 释放 permit；capacity 与 abuse Retry-After 分离；recovery 前三项先于 adapter；router 顺序和公共错误映射正确。

### Round 2：十八文件真实消费者

沿 `handler → service/login → security/session/audit/platform → main/lifecycle` 检查：

- 同一 gated hasher 被 production LoginService 消费，Session/CSRF entropy/TTL 显式传入；
- capacity error 由 security 产生，经 identity carrier 到 Handler；
- production route 为 `Trace → local router → OriginGuard → exact POST`；
- audit、metrics、logger、Origin observer 和双 listener 生命周期未见 I-F005-13～22 回退；
- 原 F001～F004 涉及的事务审计、fake probe、subject HMAC metadata、PostgreSQL fixture 未见回归。

消费者检查同时确认：现有测试没有完整执行 I-F005-24～29 要求的 production composition 与跨层矩阵，详见 F003/F004。

## 4. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

#### IDENTITY-S3.2-F005-R04-F001 — loader 后段失败未清零已物化 HMAC key

- **位置**：`cmd/han-server/main.go:463-544`，清零仅在 `cmd/han-server/main.go:196-211`
- **触发**：
  1. `loginBucketKey` 在 463 行物化后，任一 abuse、Session/CSRF、TTL、source 或 recovery 配置失败；
  2. `sourceKey` 在 525 行物化后，key independence、trusted proxy 或 recovery 配置失败。
- **证据**：`loadIdentityStartupConfig` 的上述错误分支直接返回，没有从 key 物化时安排覆盖成功/失败路径的 `clear`。只有 loader 成功并进入 `runComposed` 的 `prepareBusiness` 返回后才清零。529 行还通过 `string([]byte)` 比较两个 key，形成不必要的秘密表示。
- **影响**：启动失败时，启动层新增的 key byte slice 保留到 GC；违反 r26 “临时 `[]byte` 从物化时覆盖成功/失败路径 clear、adapter 复制后不再保留启动层副本”的敏感信息合同。
- **最小修复**：key 物化后立即建立覆盖所有出口的清零所有权；成功移交并由 adapter 内部复制后释放启动层引用。独立性比较使用 `bytes.Equal`，不创建不必要的 string 表示。
- **验证**：为首个 key 后、第二个 key 后的每个晚失败阶段做表驱动 fault injection；保留被传递 slice 的测试视图并断言所有错误/成功移交出口均已清零，错误、日志和输出仍不含 canary。

#### IDENTITY-S3.2-F005-R04-F002 — trusted proxy loader 接受非 canonical 空白，且 Unmap 后未重新拒绝 trust-all

- **位置**：`cmd/han-server/main.go:594-625`
- **触发**：
  1. `HAN_TRUSTED_PROXY_CIDRS` 整体或逗号项带前后空白；
  2. mapped IPv6 前缀 canonical/unmap 后等价于 IPv4 `/0`。
- **证据**：595、608 行先 `TrimSpace` 再解析，因此本应拒绝的空白输入被静默接受；610 行只在 Unmap 前检查 `Bits()==0`，617 行转换后没有重新检查转换结果的 bits。`NewTrustedSourceEvidenceProvider` 会再次拒绝后者，但前者可进入 production，且 loader 没有履行 I-F005-23 的一次性完整校验。
- **影响**：安全敏感代理信任配置没有按 accepted design fail closed；部署拼写/规范漂移被静默归一化，mapped trust-all 也能穿过唯一环境读取边界。后者虽被下游 constructor 再次关闭，仍破坏 loader 的单一校验合同和零副作用可证明性。
- **最小修复**：比较原始值与 trimmed 值并拒绝差异；每项同样拒绝前后空白；mapped prefix 完成 Unmap 后重新执行有效、masked、非 `/0`、重复等价检查。
- **验证**：补齐整体/单项空白、`none` 空白、mapped trust-all、mapped 合法边界、Unmap 后重复、0/1/64/65 项、4096 bytes、host bits 表驱动矩阵；断言失败时 prepare/connect/Ping/migrate/compose/listen 均为 0。

#### IDENTITY-S3.2-F005-R04-F003 — production factory 测试只比较函数指针，未执行真实装配与失败清理合同

- **位置**：`cmd/han-server/main_test.go:665-674`、`709-801`、`907-935`
- **触发**：prepared constructor、真实 route factory、pool-backed adapter 或 bind 阶段发生失败，或未来错误地重复构造 IDs/Clock/Gate/Handler。
- **证据**：
  - `TestProductionDependencies_UseRealLoginFactory` 只断言非 nil及函数指针相等；
  - composition 成功测试在 731 行替换 `composeBusiness`；
  - `validCompositionDependencies` 在 923 行默认使用 fake `http.NotFoundHandler`，且没有执行 production `prepareProductionLoginDependencies → newProductionBusinessHandler`。
- **影响**：I-F005-24 要求的真实 production factory、单实例 owner、constructor/factory/bind 失败与逆序清理没有自动化证据。当前 wiring 的跨边界回归可以在现有全部测试通过时进入生产入口。
- **最小修复**：在现有 `main_test.go` 内用 deterministic random/IDs/Clock 和可计数资源，真实执行 production prepare 与 handler factory；逐阶段注入构造、factory、第一/第二 bind 失败，断言单实例、无路由开放、无 goroutine、listener/pool/secret 引用按逆序回到基线。
- **验证**：新增 I-F005-24 精确测试并在 `-race` 下运行；测试不得以函数地址、build 成功或替换后的 fake route 代替真实 composition。

#### IDENTITY-S3.2-F005-R04-F004 — I-F005-25～29 的关键跨层/状态矩阵未落成

- **位置**：
  - `internal/security/argon2_test.go:12-92`
  - `internal/identity/login_test.go:661-765`
  - `internal/identity/login_test.go:1176-1203`
  - `cmd/han-server/main_test.go:617-663`
- **触发/证据**：
  - Gate 测试只用 Hash 饱和及 error/panic 释放，未证明 startup fake Hash/Verify、真实/未知用户 Verify、temporary/permanent rehash 共用同一 permit set；
  - Handler 429 测试只覆盖 abuse 动态 `Retry-After: 2`，未覆盖 capacity 的 `request.rate_limited + Retry-After: 1`、不写 abuse bucket及 denylist；
  - recovery 新测试只覆盖 `RecoveryRequired=false` 一个组合，没有四项合取及 normal/maintenance/error/unknown adapter/Verify 矩阵；
  - source 测试只有 5 个案例，缺多 header wire order、0/16/17 hops、zone/端口/歧义、完整 trusted-prefix 边界；
  - router 测试只计 login handler 调用，未计 Origin observer、Service、attempt、login metrics 的 0 调用，也未覆盖 Origin allow/reject 顺序。
- **影响**：高风险容量保护、恢复凭据门禁、来源归因和路由隔离缺少 task/test-plan 明确要求的确定性回归证据；现有 PASS 不能证明 I-F005-25～29 完成。
- **最小修复**：只在十八文件内补齐 r08 对应表驱动/同步屏障矩阵；复用同一真实 gated hasher 和可计数消费者，不使用 sleep，不用 mock/build 冒充跨层合同。
- **验证**：新增精确 targeted 命令并执行包级、四包 `-race`；capacity、abuse、recovery、source、router 每个分支同时断言响应、调用次数、permit/资源归零及敏感字段 denylist。

### LOW

无独立 LOW finding。

## 5. 历史 finding 与 F005 disposition

- 原 `IDENTITY-S3.2-REV-F001`～`F004`：**保持 CLOSED**。本轮未发现事务行锁、fake credential probe、subject HMAC 审计 metadata、PG18 audit fixture 的回归。
- `IDENTITY-S3.2-REV-F005`：**OPEN**。production 主路径已不再 fixed-fail，但 F001/F002 是当前实现缺陷，F003/F004 表明 I-F005-24～29 的任务级自动化证据尚未完成。
- 当前代码 Review Gate：**FAIL**。
- `identity-auth-S3.2` / B4：**不得关闭**。修复后需原 Reviewer 复审；独立 PASS 仍不等于部署、E2E 或 production readiness。

## 6. 实际命令与结果

Reviewer 实际执行：

```text
Get-FileHash -Algorithm SHA256 -LiteralPath <十八文件>
gofmt -l <十八文件>
go test ./internal/security -run '^TestArgon2idProcessGate_' -count=1
go test ./internal/identity -run '^(TestTrustedSourceEvidenceProvider_StrictProxyResolution|TestLoginRecoveryBootstrap_RequiresRecoveryStateBeforeAdapterOrVerify|TestLoginHandler_SuccessSetsCookieAndFailureKeepsExistingCookie|TestNewLoginService_FakeCredentialDigestMustMatchApprovedProbe)$' -count=1
go test ./internal/platform -run '^(TestDefaultRegistry_AcceptedMappings_AreStableAndUnique|TestMetricsHandler_|TestOrigin)' -count=1
go test ./cmd/han-server -run '^(TestLoadStartupConfig_|TestLoginBusinessRouter_JSONFallbackPrecedesLogin|TestProductionDependencies_UseRealLoginFactory|TestRun_ComposedInvalidConfigHasNoSideEffects|TestRun_ComposesJSONObserversRoutesAndDualBind|TestRun_SecondBindFailureDoesNotStartServers|TestCompositionMetricsAdapters)$' -count=1
go test -race ./internal/security -run '^TestArgon2idProcessGate_' -count=1
go test -race ./internal/identity -run '^(TestTrustedSourceEvidenceProvider_StrictProxyResolution|TestLoginRecoveryBootstrap_RequiresRecoveryStateBeforeAdapterOrVerify|TestLoginHandler_SuccessSetsCookieAndFailureKeepsExistingCookie|TestNewLoginService_FakeCredentialDigestMustMatchApprovedProbe)$' -count=1
go test -race ./internal/platform -run '^(TestDefaultRegistry_AcceptedMappings_AreStableAndUnique|TestMetricsHandler_|TestOrigin)' -count=1
go test -race ./cmd/han-server -run '^(TestLoadStartupConfig_|TestLoginBusinessRouter_JSONFallbackPrecedesLogin|TestProductionDependencies_UseRealLoginFactory|TestRun_ComposedInvalidConfigHasNoSideEffects|TestRun_ComposesJSONObserversRoutesAndDualBind|TestRun_SecondBindFailureDoesNotStartServers|TestCompositionMetricsAdapters)$' -count=1
```

上述 Go 测试与 race 全部 PASS；十八文件 `gofmt` clean。另通过 `Get-Content` / `Select-String` 对设计、test-plan、task、生产代码和测试取得精确行证据。

按用户本轮“禁止 Git 操作”约束，Reviewer **未执行** `git status/diff/diff --check`。Lead 提供的 2026-10-23 现场证据记录 scoped diff-check PASS、9 changed/9 unchanged、无第十九文件；本报告不把该结果冒充 Reviewer 自跑。

## 7. 未验证与外部限制

- Reviewer 本轮未独立复跑 PostgreSQL 18；Lead 已现场报告三个 PG18 命令 PASS，历史 r03 证据仍只覆盖其明确的事务/audit fixture 范围。
- 未执行 target-environment Argon2 benchmark/profile readiness。
- 未执行 production observability backend、30 天日志留存、真实 scrape/告警/Dashboard/receiver/runbook。
- 未执行部署或 E2E。
- 共享工作树相对 HEAD 的对象为 untracked 快照；结论只绑定本文列出的十八 SHA256。

## 8. 交回建议

由原作者在十八文件边界内依次处理 F001/F002，再补齐 F003/F004 的确定性测试；不得用 Lead 的全量 PASS 替代缺失的 I-F005-24～29 证据。完成后固定新 SHA，并由 Ampere 对四项 finding 做定点复审。
