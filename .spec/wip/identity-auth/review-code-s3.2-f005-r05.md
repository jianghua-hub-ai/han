# identity-auth S3.2 / F005 独立代码复审 r05

- Reviewer：Ampere（非作者）
- 日期：2026-10-23
- objectType：code
- 活动：development-delivery / review
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 上轮报告：`.spec/wip/identity-auth/review-code-s3.2-f005-r04.md`
- 结论：**FAIL；F001～F003 CLOSED，F004 OPEN（部分修复）；无新增阻断 finding。**

## 1. 固定对象

评审开始时按 SHA256 固定以下十八个对象，18/18 与交接值一致：

| 文件 | SHA256 |
|---|---|
| `internal/identity/handler.go` | `5B8B012A3F855C98DBBB10AA34050E15320615F3B36321A84B850B87990B2D36` |
| `internal/identity/service.go` | `8DC48D994C7B967FC586B4F7C4B70C6A99B9C3EF9DFD05BA018D278B086AA28F` |
| `internal/identity/login.go` | `762843D10B5EE95E7D876C0C0857EE523307CBEDE028AB5C101389AB422147D2` |
| `internal/identity/login_test.go` | `C7F0EAA1CF5D076CD52BD3FB9D0BC20C18EB48C90F8FFFCD1401164AAED4260C` |
| `internal/identity/login_integration_test.go` | `1D2DA4502D640689767D1C9D8D013576E73A266CD4D649F1044CBB23EC1C0DE3` |
| `internal/platform/http_security.go` | `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` |
| `internal/platform/http_security_test.go` | `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` |
| `internal/audit/store_integration_test.go` | `4A3CC84CE6B620724692D696B705D74BC39C8E1EB28F044EDD136E166FF6138C` |
| `cmd/han-server/main.go` | `E6D2570BE3F842C8C2841723067B16123D470BED0F789C089DB14C81C0901450` |
| `cmd/han-server/main_test.go` | `9A23382ED94064CADB0C9C7A3DF065448AFBAED5360C2B29F6F430175CE9AA27` |
| `cmd/han-server/lifecycle.go` | `182485F059254AD2541F4A97F7C05AC6C20E857B475B9896CB43C0703260F082` |
| `cmd/han-server/lifecycle_test.go` | `9F68060F66EFE480D1AD27422F88753485C12024CE0E9953D3EF7876031FFC0F` |
| `internal/platform/metrics.go` | `517B232D76C877FF335354E95668EC1D84B84385C017FCD45A4C046C6E4BC33A` |
| `internal/platform/metrics_test.go` | `FE97076376DBBABAA5B256BEFB92DAB527372AA652500F25EE0246B0D572EFB4` |
| `internal/security/argon2.go` | `BF6B6E63D315832001A010C77FCBA2341C2E1FDBEA1E0D6471B15BC3B3D406B7` |
| `internal/security/argon2_test.go` | `F7A382D2722D0E8F606C934DCAA0DE19C74923C5F8B39FD85932BBA338DD62B9` |
| `internal/platform/errors.go` | `99B1094A6C15A6E86E45B43928F8AC1B4F3A4EE810857BE598E9AA4336D968E3` |
| `internal/platform/errors_test.go` | `5C8CEA67B738BE1AA11FB4AE206907604BEE7CC2CC0C646891CF4EF35A6D9237` |

相对 r04 只有 `login.go`、`login_test.go`、`argon2_test.go`、`main.go`、`main_test.go` 五个对象变化。依据对象也已核对：design r26、test-plan r08、tasks、Plan 和 r04 报告哈希均与交接值一致。因本轮禁止 Git 操作，未使用 `git diff/status` 推断对象变化。

## 2. 两轮检查范围

### Round 1：修订对象直接审查

完整审查五个变化文件，并按 r04 四项 finding 定点核对：

- 启动配置中两份 HMAC key 从物化、晚失败、成功移交到启动层副本清零的所有权；
- trusted proxy 原始空白、项空白、canonical/masked、mapped IPv6 Unmap、`/0` 和等价重复；
- production prepare/factory 的真实执行、构造/factory/第一和第二 bind 失败、pool/listener/secret 清理；
- Gate、capacity 429、recovery、source、router 新增矩阵是否满足 test-plan r08 I-F005-24～29，而非仅以标签或直接调用冒充消费者合同；
- 错误去敏、响应写入、审计/abuse 顺序、并发同步和无 `time.Sleep`。

### Round 2：真实消费者与未变对象

沿 `main → prepareProductionLoginDependencies → newProductionBusinessHandler → router/OriginGuard → Handler → Service/Login → security Gate/session/audit/platform errors/metrics → lifecycle` 检查十三个未变对象及直接构造器消费者。重点确认：

- HMAC 构造器复制 key，启动层清零不会破坏长期 owner；
- production 使用单个 gated hasher 注入 Login Service；
- capacity typed error 到公共 `429 request.rate_limited`、`Retry-After: 1`、rollback、独立失败审计和 `resource_capacity` observation 的传播；
- 404/405 在 Origin/identity 前返回，exact POST 才进入 OriginGuard；
- 原 F001～F004 的事务锁、fake probe、subject HMAC metadata 和 PG18 audit fixture 未见代码回归。

## 3. r04 finding disposition

### `IDENTITY-S3.2-F005-R04-F001` — **CLOSED**

- **代码证据**：
  - `cmd/han-server/main.go:432-440` 在 identity loader 入口建立失败出口清零；
  - `cmd/han-server/main.go:487-489`、`549-551` 分别物化两份 key；
  - `cmd/han-server/main.go:553-555` 使用 `bytes.Equal`，不再建立额外 string 秘密表示；
  - `cmd/han-server/main.go:203-217` 在 prepare 返回后，无论成功或失败均清零并释放启动层引用；
  - `internal/identity/login.go:632-642` 与 `internal/identity/handler.go:55-87` 的长期 owner 均复制 key。
- **测试证据**：
  - `cmd/han-server/main_test.go:618-680` 覆盖首个 key 后和第二个 key 后的晚失败；
  - `cmd/han-server/main_test.go:682-697` 覆盖 prepare 返回出口；
  - `cmd/han-server/main_test.go:985-1038` 覆盖真实 prepare/factory 后的第一、第二 bind 失败及秘密清零。
- **结论**：根因已按最小范围修复，测试保留底层 slice 视图验证实际覆盖，不是只检查引用置空。

### `IDENTITY-S3.2-F005-R04-F002` — **CLOSED**

- **代码证据**：
  - `cmd/han-server/main.go:619-633` 拒绝整体和单项前后空白；
  - `cmd/han-server/main.go:635-647` 在解析及 Unmap 后重新检查 canonical/masked/非 `/0`；
  - `cmd/han-server/main.go:648-653` 在 canonical 化后拒绝等价重复。
- **消费者证据**：`internal/identity/handler.go:67-81` 的 source provider 构造器继续 fail closed，形成防御性二次校验，而不是替代 loader 校验。
- **测试证据**：
  - `cmd/han-server/main_test.go:699-755` 覆盖整体/单项/`none` 空白、IPv4/IPv6/mapped trust-all、mapped 合法边界、等价重复、host bits、1/64/65 项与 4096 bytes 边界；
  - `cmd/han-server/main_test.go:757-797` 断言非 canonical 配置在 prepare/connect/Ping/migrate/compose/listen 前失败。
- **结论**：loader 已履行一次性完整 fail-closed 校验。

### `IDENTITY-S3.2-F005-R04-F003` — **CLOSED**

- **代码证据**：
  - `cmd/han-server/main.go:769-833` 是真实 production prepare；
  - `cmd/han-server/main.go:836-900` 构造真实 session store、abuse protector、Login Service、attempt coordinator、Handler、OriginGuard 与 local router；
  - `cmd/han-server/main.go:925-967` 将上述 prepare/factory 接入 production dependencies；
  - `cmd/han-server/main.go:270-320` 在 factory/bind 失败时关闭已建 pool/listener，且两个 listener 均成功前不构造 server。
- **测试证据**：
  - `cmd/han-server/main_test.go:956-983` 真实执行 prepare 与 handler factory，并实际请求 production router；
  - `cmd/han-server/main_test.go:985-1040` 真实执行 factory 后注入第一/第二 bind 失败，核 prepare/factory 单次调用、pool/listener 清理和秘密清零；
  - `cmd/han-server/main_test.go:1042-1118` 覆盖 prepared constructor 与真实 handler factory 失败，断言无 listener、server、coordinator 或路由开放。
- **结论**：修订不再以函数指针或 fake `NotFoundHandler` 代替 production factory 合同。

### `IDENTITY-S3.2-F005-R04-F004` — **OPEN（部分修复）**

- **已补证据**：
  - `internal/identity/login_test.go:696-827` 真实覆盖一个已知用户 Verify 的 runtime capacity：固定 429、`Retry-After: 1`、abuse 零写、rollback、一次独立审计、`resource_capacity` 和响应 denylist；
  - `internal/identity/login_test.go:829-1004` 显著扩充 source/XFF 与 trusted-prefix 边界；
  - `internal/identity/login_test.go:1415-1545` 扩充 recovery 前置合取及 adapter 结果；
  - `cmd/han-server/main_test.go:799-954` 覆盖 Trace/local-router/Origin/identity 的调用顺序和 404/405 下游零调用。
- **仍未满足的合同**：
  1. `internal/security/argon2_test.go:94-179` 将闭包命名为 startup fake、真实/未知 Verify、temporary/permanent rehash，但各闭包只是直接调用同一个 `GatedArgon2idHasher.Hash/Verify`；没有经 production prepare 或 identity credential-mode/rehash 消费者同时驱动，因此不能证明 test-plan I-F005-25 要求的“所有消费者共用同一 Gate”。
  2. `internal/identity/login_test.go:696-827` 只覆盖已知永久用户 Verify 饱和；没有未知用户 fake Verify、temporary rehash、permanent rehash capacity 的 Handler/Service 路径。`cmd/han-server/main_test.go:956-1118` 也没有 startup fake Hash/Verify capacity 的 listener 前失败用例。因此 I-F005-29 的 startup/runtime 全矩阵仍缺。
  3. `internal/identity/login_test.go:1444-1545` 直接调用 `verifyLoginCredential`，未按 I-F005-26 对完整 Login 路径同时核 audit/session 调用与回滚状态。
  4. `internal/identity/login_test.go:829-961` 尚未以可信 peer 明确覆盖空 XFF 项和普通 malformed XFF 项；现有 hostile malformed header 只在非可信 peer 场景验证“忽略 XFF”，不能替代 I-F005-27 的可信代理 fail-closed 分支。
- **影响**：实现代码本身在已检查路径上未见新的安全缺陷，但 I-F005-25/26/27/29 明确要求的跨层、状态和敏感边界自动化证据仍可在消费者接线回归时假 PASS；r04 F004 的根因未完整消除。
- **最小修复**：在现有十八文件边界内，用同一真实 production-created gated hasher 与同步屏障，分别经 startup fake、未知用户、temporary/permanent rehash 的真实消费者触发饱和；补完整 Login recovery 的 audit/session/rollback 计数，以及可信 peer 的空/非法 XFF 表项。不得仅给直接 Hash/Verify 闭包改名。
- **验证**：精确 targeted + 三相关包包级与 `-race`；每例核 typed 1s owner、无 abuse failure write、一次响应/审计、permit/资源归零及 denylist。

## 4. 新 findings

未发现本次修订引入新的 CRITICAL、HIGH、MEDIUM 或 LOW finding。当前阻断仅为原 `IDENTITY-S3.2-F005-R04-F004` 未完整关闭。

## 5. 实际命令与结果

Reviewer 独立执行：

```text
Get-FileHash -Algorithm SHA256 -LiteralPath <十八文件>
go test ./internal/security -run '^TestArgon2idProcessGate_' -count=1
go test ./internal/identity -run '^(TestLoginHandler_Argon2CapacityReturnsFixed429WithoutAbuseMutation|TestTrustedSourceEvidenceProvider_StrictProxyResolution|TestTrustedSourceEvidenceProvider_TrustedPrefixBoundaries|TestLoginRecoveryBootstrap_RequiresRecoveryStateBeforeAdapterOrVerify|TestLoginRecoveryBootstrap_FourWayConjunctionAndAdapterResults)$' -count=1
go test ./cmd/han-server -run '^(TestLoadIdentityStartupConfig_ClearsMaterializedSecretsOnEveryLateFailure|TestRun_ComposedClearsStartupSecretsAfterSuccessfulPreparation|TestParseTrustedProxyConfiguration_StrictCanonicalBoundaries|TestRun_NonCanonicalTrustedProxyHasZeroSideEffects|TestLoginBusinessRouter_JSONFallbackPrecedesLogin|TestLoginBusinessRouter_CrossLayerOrderingAndZeroCalls|TestProductionDependencies_ExecuteRealLoginPreparationAndFactory|TestRun_RealProductionCompositionUnwindsBeforeRouteOpenOnBindFailure|TestRun_RealProductionCompositionConstructorAndFactoryFailuresHaveNoRouteOrListener)$' -count=1
go test -race ./internal/security -run '^TestArgon2idProcessGate_' -count=1
go test -race ./internal/identity -run '^(TestLoginHandler_Argon2CapacityReturnsFixed429WithoutAbuseMutation|TestTrustedSourceEvidenceProvider_StrictProxyResolution|TestTrustedSourceEvidenceProvider_TrustedPrefixBoundaries|TestLoginRecoveryBootstrap_RequiresRecoveryStateBeforeAdapterOrVerify|TestLoginRecoveryBootstrap_FourWayConjunctionAndAdapterResults)$' -count=1
go test -race ./cmd/han-server -run '^(TestLoadIdentityStartupConfig_ClearsMaterializedSecretsOnEveryLateFailure|TestRun_ComposedClearsStartupSecretsAfterSuccessfulPreparation|TestParseTrustedProxyConfiguration_StrictCanonicalBoundaries|TestRun_NonCanonicalTrustedProxyHasZeroSideEffects|TestLoginBusinessRouter_JSONFallbackPrecedesLogin|TestLoginBusinessRouter_CrossLayerOrderingAndZeroCalls|TestProductionDependencies_ExecuteRealLoginPreparationAndFactory|TestRun_RealProductionCompositionUnwindsBeforeRouteOpenOnBindFailure|TestRun_RealProductionCompositionConstructorAndFactoryFailuresHaveNoRouteOrListener)$' -count=1
go test ./cmd/han-server ./internal/identity ./internal/security ./internal/platform -count=1
gofmt -l <十八文件>
Select-String <五个变化文件> 'time.Sleep|TODO|FIXME|fmt/log Print|HAN_TEST_POSTGRES_DSN'
```

结果：上述测试与 targeted race 均 PASS；四包包级测试 PASS；十八文件 `gofmt` clean；扫描 0 命中。通过 `Get-Content` / `Select-String` 读取并核对 Role、Skill、Rules/Profile、design r26、test-plan r08、tasks/Plan、r04、十八文件与直接消费者。

本轮按禁止 Git 操作要求未执行 `git status/diff/diff --check`。Lead 提供的全量 Gate、PG18 与 18/18 `diff --no-index --check` PASS 仅作为上游证据记录，不冒充 Reviewer 自跑。

## 6. 限制

- Reviewer 未独立复跑 PostgreSQL 18；Lead 报告 `server_version_num=180006` 且三组约定 integration PASS。本轮修订不涉及 PG fixture，但该结果仍由 Lead 持有。
- 未执行 target-environment Argon2 benchmark/profile、production observability backend/留存/scrape/告警/Dashboard/receiver/runbook。
- 未执行部署或 E2E。
- 未评审或宣称 S2.2、S3.5、S3.6 完成。
- 结论仅绑定本文十八个 SHA256；共享工作树后续变化必须重新固定对象。

## 7. Gate 与交回

- F001：**CLOSED**
- F002：**CLOSED**
- F003：**CLOSED**
- F004：**OPEN（部分修复）**
- 新阻断 finding：**无**
- 本轮 code Review Gate：**FAIL**
- `IDENTITY-S3.2-REV-F005`：**保持 OPEN**
- `identity-auth-S3.2` / B4：**不得因本报告关闭**

交回 Lead：请由原作者仅补齐 F004 所列真实消费者/状态矩阵，再固定新 revision 交原 Reviewer 定点复审。即使后续代码 Review PASS，也不等于 S2.2、S3.5、S3.6、B4 batch、production readiness、部署或 E2E 完成。
