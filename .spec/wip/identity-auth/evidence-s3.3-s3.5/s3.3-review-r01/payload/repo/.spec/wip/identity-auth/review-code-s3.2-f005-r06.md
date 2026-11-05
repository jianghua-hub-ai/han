# identity-auth S3.2 / F005 独立代码复审 r06

- Reviewer：Ampere（非作者）
- 日期：2026-10-23
- objectType：code
- 活动：development-delivery / review
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/identity-auth/review-code-s3.2-f005-r05.md`
- 原报告 SHA256：`B125ADC991C1667FE0DA87261D1C824D3C6CF12B580FAA084978D0531572028A`
- 本轮目标：仅复审 r05 尚 OPEN 的 `IDENTITY-S3.2-F005-R04-F004`
- 结论：**PASS；F004 CLOSED；无新增 finding。**

## 1. 固定对象

评审开始时按 SHA256 固定十八个代码对象，18/18 与交接值一致：

| 文件 | SHA256 |
|---|---|
| `internal/identity/handler.go` | `5B8B012A3F855C98DBBB10AA34050E15320615F3B36321A84B850B87990B2D36` |
| `internal/identity/service.go` | `8DC48D994C7B967FC586B4F7C4B70C6A99B9C3EF9DFD05BA018D278B086AA28F` |
| `internal/identity/login.go` | `762843D10B5EE95E7D876C0C0857EE523307CBEDE028AB5C101389AB422147D2` |
| `internal/identity/login_test.go` | `8468CCA9405C4EF614BFA940A06AD38D7EC269180EB4051F443D3E7D6DE46256` |
| `internal/identity/login_integration_test.go` | `1D2DA4502D640689767D1C9D8D013576E73A266CD4D649F1044CBB23EC1C0DE3` |
| `internal/platform/http_security.go` | `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` |
| `internal/platform/http_security_test.go` | `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` |
| `internal/audit/store_integration_test.go` | `4A3CC84CE6B620724692D696B705D74BC39C8E1EB28F044EDD136E166FF6138C` |
| `cmd/han-server/main.go` | `E6D2570BE3F842C8C2841723067B16123D470BED0F789C089DB14C81C0901450` |
| `cmd/han-server/main_test.go` | `346C56C57FE09E4528C8EE01BFAAD8BF71947E57462DAED7DA750E1EDB91367C` |
| `cmd/han-server/lifecycle.go` | `182485F059254AD2541F4A97F7C05AC6C20E857B475B9896CB43C0703260F082` |
| `cmd/han-server/lifecycle_test.go` | `9F68060F66EFE480D1AD27422F88753485C12024CE0E9953D3EF7876031FFC0F` |
| `internal/platform/metrics.go` | `517B232D76C877FF335354E95668EC1D84B84385C017FCD45A4C046C6E4BC33A` |
| `internal/platform/metrics_test.go` | `FE97076376DBBABAA5B256BEFB92DAB527372AA652500F25EE0246B0D572EFB4` |
| `internal/security/argon2.go` | `BF6B6E63D315832001A010C77FCBA2341C2E1FDBEA1E0D6471B15BC3B3D406B7` |
| `internal/security/argon2_test.go` | `BEE56CE85531BDAC700B9B5A1A8E6A88CAAA48102BADD6708D8E00CB35257A33` |
| `internal/platform/errors.go` | `99B1094A6C15A6E86E45B43928F8AC1B4F3A4EE810857BE598E9AA4336D968E3` |
| `internal/platform/errors_test.go` | `5C8CEA67B738BE1AA11FB4AE206907604BEE7CC2CC0C646891CF4EF35A6D9237` |

相对 r05 仅 `internal/identity/login_test.go`、`cmd/han-server/main_test.go`、`internal/security/argon2_test.go` 三文件变化；十五个生产/消费者对象保持 r05 SHA。因本轮禁止 Git 操作，未使用 `git diff/status` 推断范围。

## 2. 两轮检查范围

### Round 1：三项测试修订直接检查

完整读取三份变化文件，逐项复核 r05 F004 的四个剩余缺口：

1. 底层 Gate 测试是否停止使用消费者名称冒充跨层证明；
2. production-created gated hasher 是否真实经 startup fake、unknown fake Verify、temporary/permanent rehash 消费者触发；
3. recovery 是否改走完整 `Service.Login` 并核 audit/session/commit/rollback；
4. trusted peer 是否补空 XFF 项和 malformed XFF 的 fail-closed 用例。

同时检查同步屏障、permit 释放、响应次数、typed `1s` capacity owner、abuse/audit/事务状态、denylist、无 `time.Sleep`/Skip。

### Round 2：未变生产消费者核对

沿以下真实链路核对新增测试没有迎合测试替身或改变合同：

- `prepareProductionLoginDependencies`：`cmd/han-server/main.go:769-833`
- production factory：`cmd/han-server/main.go:836-900`
- fake probe 构造期 Verify：`internal/identity/service.go:192-211`
- known/unknown/temporary/permanent credential 消费路径：`internal/identity/login.go:1155-1224`、`1345-1529`
- capacity 公共错误映射：`internal/identity/login.go:1935-1947`
- Handler 429/Retry-After/observation：`internal/identity/handler.go:390-400`
- source 解析：`internal/identity/handler.go:105-206`

确认 production 代码本轮未变化；新增 barrier 只在测试中选择准确的饱和时点，实际 Hash/Verify 仍委托给同一 production-created gated hasher。

## 3. Finding disposition

### `IDENTITY-S3.2-F005-R04-F004` — **CLOSED**

#### 3.1 Gate 与真实消费者

- `internal/security/argon2_test.go:94-159` 已将测试范围准确收窄为底层同一 permit set 覆盖 Hash/Verify，不再用 startup、unknown 或 rehash 名称冒充消费者证明。
- `cmd/han-server/main_test.go:991-1123`：
  - 真实调用 `prepareProductionLoginDependencies`；
  - 对 startup fake Hash 注入实际 entropy fault；
  - 以同一 prepared hasher 占用 permit，使 production factory 中 `NewLoginService` 的真实 fake Verify 命中 typed `1s` capacity；
  - 断言失败发生在 listener/server/coordinator 前、pool 恰好关闭一次，并在释放后再次 Verify 成功。
- `cmd/han-server/main_test.go:1126-1380` 使用同一个 production-created `prepared.passwords`，经真实 Handler → Service 覆盖 unknown fake Verify、temporary rehash、permanent rehash。rehash barrier 只在成功 Verify 与实际 Hash 之间占用同一 Gate，未替换被测 Hash/Verify。
- `internal/identity/login_test.go:696-951` 另以真实 Handler → Service 覆盖 known permanent Verify、unknown fake Verify及两种 rehash，并核释放后 permit 可再次获取。

结论：I-F005-25 与 I-F005-29 要求的 startup/runtime 消费者共享、非阻塞 capacity、typed `1s` owner和 permit 释放已有确定性跨层证据。

#### 3.2 Capacity HTTP、事务、审计与敏感边界

`internal/identity/login_test.go:869-948` 对四类 runtime consumer 逐例断言：

- 唯一 `429 request.rate_limited` 和 `Retry-After: 1`；
- header/body 各一次；
- abuse failure/success 均为 0；
- 主事务 0 commit / 1 rollback；
- transactional audit 为 0、独立失败审计为 1；
- Session 和 digest update 均为 0；
- observation 恰好一次且 `reason_bucket=resource_capacity`；
- typed capacity error 仍由同一 Gate 携带固定 `1s`；
- 响应不含用户名、密码、source/digest/fake credential 等 canary。

production composition 对应矩阵在 `cmd/han-server/main_test.go:1289-1377` 再次验证。没有 retry、fallback、二次响应或 abuse bucket 污染。

#### 3.3 Recovery 完整状态矩阵

- `internal/identity/login_test.go:1551-1590` 使用完整 `Service.Login` 验证前置状态不满足时 adapter/Verify 为 0，业务失败审计在主事务中提交且 Session 为 0。
- `internal/identity/login_test.go:1592-1749` 覆盖：
  - `IsInitialAdmin=false`
  - `RecoveryRequired=false`
  - normal
  - maintenance
  - adapter error
  - 非 recovery-bootstrap permanent 流量
- 每例同时核 adapter、Verify、Session、transactional/independent audit、commit/rollback；仅 maintenance 成功进入 Verify/Session，adapter error fail closed 并 rollback。

结论：r05 所列 I-F005-26 跨完整 Login 状态和副作用缺口已关闭。

#### 3.4 Trusted source 缺失分支

`internal/identity/login_test.go:1008-1047` 已补：

- trusted peer 无 forwarded hops；
- trusted peer XFF 含空项；
- trusted peer malformed XFF；
- port、zone、brackets 和全链可信。

这些用例与既有 wire order、0/16/17 hops、mapped canonicalization、非可信 peer 忽略 hostile XFF 和 trusted-prefix 边界共同覆盖 I-F005-27 本 finding 指出的缺失。

## 4. 新 findings

未发现本轮测试修订引入新的 CRITICAL、HIGH、MEDIUM 或 LOW finding。

## 5. 历史 disposition 与 Gate

- `IDENTITY-S3.2-F005-R04-F001`：**CLOSED**（r05）
- `IDENTITY-S3.2-F005-R04-F002`：**CLOSED**（r05）
- `IDENTITY-S3.2-F005-R04-F003`：**CLOSED**（r05）
- `IDENTITY-S3.2-F005-R04-F004`：**CLOSED**（本报告）
- 原任务 `IDENTITY-S3.2-REV-F001`～`F004`：未见回归，保持 **CLOSED**
- 新 finding：**无**
- 本轮 code Review Gate：**PASS**
- `IDENTITY-S3.2-REV-F005`：**可 CLOSED**

S3.2 可交回 Lead 做 task-level closure 核验；本报告不直接修改台账或关闭任务。该 PASS 不等于 S2.2、S3.5、S3.6、B4 batch、production readiness、部署或 E2E 完成。

## 6. Reviewer 实际命令

```text
Get-FileHash -Algorithm SHA256 -LiteralPath <十八文件及 r05>
go test ./internal/security -run '^TestArgon2idProcessGate_' -count=3
go test ./internal/identity -run '^(TestLoginHandler_Argon2CapacityReturnsFixed429WithoutAbuseMutation|TestTrustedSourceEvidenceProvider_StrictProxyResolution|TestTrustedSourceEvidenceProvider_TrustedPrefixBoundaries|TestLoginRecoveryBootstrap_RequiresRecoveryStateBeforeAdapterOrVerify|TestLoginRecoveryBootstrap_FourWayConjunctionAndAdapterResults)$' -count=3
go test ./cmd/han-server -run '^(TestRun_ProductionArgon2StartupConsumersFailBeforeListener|TestProductionArgon2Gate_IsSharedByRuntimeLoginConsumers)$' -count=3
go test ./internal/security ./internal/identity ./cmd/han-server -count=1
go test -race ./internal/security ./internal/identity ./cmd/han-server -count=1
gofmt -l <十八文件>
Select-String <三变化文件> 'time.Sleep|TODO|FIXME|Skip\('
```

结果：精确定向 `-count=3`、三包包级和三包 race 均 PASS；十八文件 `gofmt` clean；目标扫描 0 命中。另通过 `Get-Content` / `Select-String` 核对 r05、test-plan r08、适用 Role/Skill/Rules/Profile、三份变化测试和直接生产消费者。

Lead 提供的全量 Gate 与 PostgreSQL 18 integration PASS 仅作为上游证据记录，不冒充 Reviewer 自跑。本轮按禁止 Git 操作要求未运行任何 Git 命令。

## 7. 限制

- Reviewer 未独立复跑 PostgreSQL 18；本轮仅测试文件变化且不改 PG fixture/生产 SQL。Lead 报告 `server_version_num=180006` 和三组约定 integration PASS。
- 未执行 target-environment Argon2 benchmark/profile。
- 未执行 production observability backend、留存、真实 scrape/告警/Dashboard/receiver/runbook。
- 未执行部署或 E2E。
- 结论只绑定本文十八个 SHA256；共享工作树后续漂移需重新固定对象。

## 8. 交回建议

交回 Lead：可将 `IDENTITY-S3.2-REV-F005` 记为 CLOSED，并结合任务卡、PG18 证据和其他既有 Gate 做 S3.2 task-level closure 核验；不得由本报告外推其他任务、B4 batch 或 production readiness 已完成。
