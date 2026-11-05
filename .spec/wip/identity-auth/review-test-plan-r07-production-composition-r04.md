# identity-auth test-plan r07 production composition 独立评审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **对象类型**：design（测试规划内容对象）
- **结论**：**FAIL**
- **Finding 统计**：CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 1
- **授权边界**：只评审固定 test-plan r07；不修改 test-plan、design、tasks、Plan、代码或 ADR，不执行 Go、PostgreSQL 18、benchmark、部署或 E2E，不替用户确认结果。

## 1. 固定对象、上游与漂移检查

| 项目 | 固定值 / 核对结果 |
|---|---|
| 被审对象 | `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` |
| revision | `identity-auth-test-plan-ir01-r07` |
| 要求 SHA256 | `1B0367BBE22F29BA158651DF3F3CF63615E3E75B4F2429E5D30958542A1CAAD6` |
| 开始 SHA256 | `1B0367BBE22F29BA158651DF3F3CF63615E3E75B4F2429E5D30958542A1CAAD6` |
| 结束/报告落盘前 SHA256 | `1B0367BBE22F29BA158651DF3F3CF63615E3E75B4F2429E5D30958542A1CAAD6`，无漂移 |
| source design | `identity-auth-design-ir01-r26`；checkpoint object digest `sha256:42d8d10c37acb55a752c198caef32819d4d73744acd71af6fb1a4fba0366bc3d` |
| accepted checkpoint | `decision-a76cb64cb07a0e37c80b3d22b15034e817e68d7c00a9be1759372e0396e6a56f`；`accepted / proceed` |
| design review | `.spec/wip/identity-auth/review-design-r26-production-composition-r04.md`；SHA256 `171E373555357493003EBC99F5704281304E86282EA8A709B14AAB4E4A82DE28`；PASS / 0 findings |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |

r07 的 metadata、正文与 checkpoint 对 source revision、checkpoint、object digest、独立设计复审及 `planned-not-executed` 状态表述一致。旧 r21/r06、既有代码制品和 PostgreSQL 18 历史执行证据均被限定为历史或未重跑证据，没有被冒充为 r26 application composition、benchmark/profile、部署或 E2E 已执行。

共享工作树在评审前已存在大量他人改动。本轮未回滚、覆盖、清理或格式化任何既有内容，唯一写入为本报告。

## 2. 两轮实际检查范围

### Round 1：固定 r07 直接审查

逐段检查：

1. metadata、修订历史、source/checkpoint/digest、设计复审和计划未执行状态；
2. 风险、Story/验收覆盖、测试层次、数据/清理、环境、执行边界和 Gate；
3. I-F005-13～22 的“不变”依据与旧证据适用边界；
4. I-F005-23～29 的前置、动作、预期、层次、数据清理和自动化/真实环境边界；
5. 23 项配置的缺失、空白、类型/枚举/范围/端点/交叉约束、零外部副作用失败及 profile token 与 readiness 分离；
6. 单一 process Argon2 Gate、startup/runtime Hash/Verify/rehash、non-blocking capacity、同步屏障和 permit release；
7. capacity typed error owner=`Gate`、固定 `1s`、运行期 `429 + Retry-After: 1`、无泄露、不写 abuse failure bucket及 abuse 动态 Retry-After 回归；
8. recovery、strict source、Trace/router/Origin、JSON 404/405、`Allow: POST` 和零业务调用；
9. 风险、覆盖、任务建议、十八文件映射、新增四文件及 blocked/planned/prior-evidence 状态。

结果：除 §4.1 与 I-F005-26 两处 finding 外，其余检查项与 accepted r26 一致且具备可执行边界。

### Round 2：真实消费者、任务与上游反查

有界反查：

- accepted design r26 的配置表、recovery 状态矩阵、Argon2 Gate/capacity、source、router、测试合同和十八文件映射；
- requirements r16、stories r09、architecture r23、ADR-0012/0014/0016/0022；
- checkpoint 与 r26 design review；
- tasks SSOT、批次 Plan；
- `cmd/han-server`、`internal/identity`、`internal/security`、`internal/platform` 的必要代码/测试消费者。

反查结果：

1. design r26 明确要求前三项 `IsInitialAdmin && RecoveryRequired && CredentialModeRecoveryBootstrap` 成立时才调用 RecoveryMode adapter；此时 `normal`、`maintenance`、`error/unknown` 都必须恰好调用一次，只有 `maintenance` 才进入 recovery credential Verify。
2. production `composeBusiness` 当前仍固定返回 `identity business route provider is not configured`；规划未把 factory/Gate/Header 写成已实现。
3. tasks/Plan 仍以十四文件、design r22/test-plan r06 为现役 S3.2 对账范围；S3.2 active、F005 OPEN、B4 未关闭。r07 如实将 r26 十八文件与新增四文件写为待 Task-Planner 回填的建议，没有冒充已登记或已执行。
4. design r26 的十八个文件全部存在；相对当前十四文件范围，新增四文件说法准确：`internal/security/argon2.go`、`internal/security/argon2_test.go`、`internal/platform/errors.go`、`internal/platform/errors_test.go`。
5. PostgreSQL 18 三组矩阵只保留为历史证据；当前未复跑。I-F005-13～29、目标环境 benchmark/profile、production observability 部署 Gate 和 E2E 均保持 planned/blocked，未混淆为通过。

## 3. Findings

### [MEDIUM] TP-R07-F001 — I-F005-26 未保留 RecoveryMode adapter 的精确调用矩阵

- **对象/修订**：test-plan `identity-auth-test-plan-ir01-r07`
- **位置**：`test-plan.md:186`
- **违反依据**：
  - accepted design `design.md:1361,1365-1372`；
  - design 测试合同 `design.md:2643`；
  - 测试规划必须准确消费已确认设计，并为安全 fail-closed 状态机给出可执行调用次数断言。
- **现状**：I-F005-26 写为“只有四项合取含 maintenance 才调用并接受 recovery credential”。该句把 **RecoveryMode adapter 调用** 与 **recovery credential Verify/接受** 合并，且未明确前三项成立时 `normal` 和 `error/unknown` 也须调用 adapter 恰好一次。
- **触发场景**：实现 I-F005-26 的表驱动测试，输入 `true / true / recovery_bootstrap / normal` 或 adapter error/unknown。
- **影响**：
  - 测试实现者可能据此断言 adapter 调用数为 0，或完全漏断言这三行的调用次数；
  - accepted r26 的 owner/call-order 合同不能被测试证明，错误短路、绕过 adapter 或调用次数漂移可漏过；
  - 该路径控制 recovery credential 是否开放，属于现实安全状态机测试盲区，因此为 MEDIUM。
- **最小修复建议**：仅澄清 I-F005-26 预期，不改变设计或文件边界：
  1. 前三项任一不成立，或 credential mode 非 `recovery_bootstrap`：adapter 0 次；
  2. 前三项全部成立：`normal`、`maintenance`、`error/unknown` 均调用 adapter 恰好 1 次；
  3. `normal` 与 error/unknown fail closed且 credential Verify 0 次；
  4. 只有 `maintenance` 才允许进入 recovery credential Verify，Verify 成功后才可接受凭据。
- **复审验证**：逐行对照 design r26 六行状态矩阵，分别断言 adapter、Verify、audit、session 调用数及 fail-closed 结果；普通流量 503 仍由独立 RecoveryGate/readiness 证明。

### [LOW] TP-R07-F002 — I-AUTH-08 的 production composition 追溯仍截断在 I-F005-22

- **对象/修订**：test-plan `identity-auth-test-plan-ir01-r07`
- **位置**：`test-plan.md:148`
- **违反依据**：
  - `test-plan.md:100` 已将登录覆盖更新为 I-F005-01～29；
  - `test-plan.md:165-174,318-320` 已正式定义和映射 I-F005-23～29；
  - 同一规划的验收追溯应使用当前场景范围。
- **现状**：I-AUTH-08 仍称 production composition 由 `I-F005-01～22` 细化。
- **触发场景**：Lead、Test-Engineer 或后继 Reviewer 从 I-AUTH-08 反查 production composition 的细化用例。
- **影响**：追溯结果会遗漏 23 项配置/factory、Argon2 Gate/capacity、recovery、source 和 router 的 I-F005-23～29。详细场景与覆盖表本身仍存在，未改变技术预期，故为 LOW。
- **最小修复建议**：将该处更新为 `I-F005-01～29`；保留 r06 历史和“I-F005-13～22 合同未变”的独立说明。
- **复审验证**：I-AUTH-08、Story 覆盖表、场景表和 §6 映射均引用当前 `01～29`，旧 `01～22` 只用于明确的历史或“不变范围”说明。

## 4. 已通过的重点核对

除上述 findings 外，未发现以下范围的新增缺陷：

1. 23 项配置覆盖缺失/空白/类型/枚举/范围/端点和关键交叉约束；非法配置在 connect/Ping/migrate/compose/listen 前零副作用失败，错误不含值。
2. 合法 profile ID 仅允许构造，不被写成 benchmark/profile 已批准或 production readiness。
3. startup fake Hash/Verify、真实/未知用户 Verify、temporary/permanent rehash 共用单一 process Gate；饱和 non-blocking，成功/错误/panic 均要求释放 permit，禁止 sleep。
4. capacity typed error 由 Gate 唯一拥有固定 `1s`；运行期稳定 `429 request.rate_limited + Retry-After: 1`，startup 无 HTTP；不写 abuse failure bucket且 abuse 动态 Retry-After 保持独立。
5. recovery 四项合取与普通流量 RecoveryGate/readiness 职责分离的方向正确；缺陷仅在 I-F005-26 的精确调用次数表达。
6. strict RemoteAddr/XFF、mapped IPv6 Unmap、16-hop、右向剥离、trust-all/等价重复拒绝、HMAC 输出和原始值不外泄具备场景落点。
7. `Trace → local router → {JSON 404/405 | OriginGuard → exact POST}`、405 `Allow: POST`、trace 一致、零 Origin/业务/attempt/login metric 调用和禁止 raw fallback均已规划。
8. I-F005-13～22 的技术合同有 r26/architecture/ADR 依据，且保持计划未执行；I-F005-23～29 除 F001 外具备明确前置、动作、预期、层次、数据清理和自动化/真实环境边界。
9. 十八文件映射与 design r26 一致，新增四文件准确；task/Plan 当前十四文件事实与“待回填”状态未被覆盖。

## 5. 待决定与非阻断建议

### 待决定

无新增架构、公共契约或十八文件边界决定。两个 finding 均可在 test-plan 内最小修正；若修复需要改变 accepted design 或第十九文件边界，必须停止并升级方向决定。

### 非阻断建议

1. I-F005-23 可在后续修订中直接列出 trusted proxy 的 `0/1/64/65` 项、4096-byte 上限、host bits、mapped prefix 与等价重复实例，降低测试实现者从 design 二次转译的成本；现有 §7 fixture 已提供这些边界，因此不列 finding。
2. I-F005-29 的 abuse 动态 Retry-After 回归宜至少包含两个非 `1s` duration及一个向上取整实例，防止 capacity 固定值污染动态路径；现有“动态 duration fixture”已构成最低覆盖，因此不列 finding。

## 6. 限制与未验证风险

- 按授权未执行 Go test/race/vet/build、PostgreSQL 18、benchmark/profile、部署、scrape、告警或 E2E。
- 本报告只判断规划是否准确、完整、可执行，不证明 r26 factory、Gate、Header、source、recovery 或 router 已实现。
- 当前 production `composeBusiness` 仍 fixed-fail；I-F005-13～29 均无本轮执行结果。
- PostgreSQL 18 历史 PASS 只适用于既有事务矩阵，当前环境/DSN/major 未复核。
- 目标环境 Argon2 profile、内存/延迟预算、采用记录、真实 provider/receiver、30 天日志查询、告警演练和责任人仍未验证。
- tasks/Plan 尚未消费 r26/r07；后续同步若改变场景、文件边界或状态，须重新核对本报告结论。
- 共享工作树含他人既有改动；本结论只绑定本报告列出的固定对象和哈希。

## 7. Gate 结论与复审条件

**identity-auth test-plan r07 Test-Plan Gate：FAIL。**

阻断原因是 `TP-R07-F001`：安全敏感 recovery-bootstrap 矩阵的 adapter 调用次数未被准确表达。`TP-R07-F002` 为非阻断 LOW，但应在同次修订中一并同步。

复审需要：

1. Test-Engineer 产生新的固定 test-plan revision 与 SHA256；
2. 保持 accepted design r26、十八文件边界、tasks/Plan 和代码只读；
3. 定点关闭 TP-R07-F001/F002，并回归 I-F005-23～29、Story/验收追溯及 planned-not-executed 状态；
4. 由非作者 Reviewer 复审；本 FAIL 不授权修复、实施、测试执行、任务状态变更、Git、部署或 E2E。

## 8. 实际命令与 diff-check

实际只读操作包括：

- `Get-FileHash -Algorithm SHA256`
- UTF-8 全文/定点读取与 `Select-String`
- `git branch --show-current`
- `git rev-parse HEAD`
- `git status --short --untracked-files=all`
- 十八文件存在性和必要消费者反查

报告落盘后执行 scoped status、对象 SHA 重核和 `git diff --no-index --check -- NUL <report>`。本文件不内嵌自身 SHA256，避免摘要自指；最终报告 SHA256记录在交付回执。
