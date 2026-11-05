# identity-auth design r25 production composition 独立复审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer（延续 r23/r24 finding 链）
- **对象类型**：design
- **结论**：**FAIL**
- **Finding 统计**：CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 0

## 1. 固定对象与边界

| 项目 | 固定值 / 结果 |
|---|---|
| 被审对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| revision | `identity-auth-design-ir01-r25` |
| 要求 SHA256 | `7FEE6175ADBF4DDFD38AEC9193D5122FE8918DD3DDCE825988BC8D79C6C7DAD3` |
| 开始 SHA256 | `7FEE6175ADBF4DDFD38AEC9193D5122FE8918DD3DDCE825988BC8D79C6C7DAD3` |
| 原 r24 复审 | `.spec/wip/identity-auth/review-design-r24-production-composition-r02.md` |
| 原 r24 复审 SHA256 | `FD3132103B97B958CAA0DECBC4AB18E9DE3ED985ED661FA2DEF3399918D87895` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |

本轮只判断 r25 对 `IA-R24-PC-R02-F001/F002` 的修订及其直接回归影响。design、ADR、architecture、requirements、test-plan、tasks 和代码消费者均只读；唯一写入本报告。共享工作树存在他人既有改动，本轮未回滚、清理、格式化或覆盖任何现状。

## 2. 结论摘要

r25 已修复原 HIGH finding 的核心问题：Argon2 tuple 已明确降为 benchmark 候选，profile token 与目标环境批准证据分层；单一 process permit set、`1..4` 并发范围、512MiB 乘积上限、Hash/Verify/fake/rehash 共用、non-blocking 拒绝和全路径释放均已达到可编码程度。r25 也彻底分离了 recovery-bootstrap adapter 与普通流量 RecoveryGate，原 F002 可关闭。

但原 F001 仍有一个收窄后的 MEDIUM 残留。设计的公共错误合同要求所有 `request.rate_limited` 429 必须带 `Retry-After`，而新增 Argon2 capacity rejection 只规定 typed error 映射到 429，没有规定该 Header 的正值、来源或承载方式。当前 Handler 只有在错误携带 `LoginRateLimitRetryAfter` duration 时才写 Header，而该 duration 仅来自持久 abuse bucket。实施者只能遗漏 Header或自行发明常量，不能同时满足 r25 新容量路径和既有公共 REST 合同。

因此 r25 设计 Gate 为 **FAIL**。原 F001 的 HIGH 风险已经消除，但 finding 尚不能整体关闭；原 F002 为 **CLOSED**。

## 3. Round 1：设计对象直接检查

实际检查：

1. revision、change log、评审整改台账、状态、确认记录及 r25 无 checkpoint 声明；
2. 23 项登录配置的读取方、类型、必填/默认、候选值、可执行上下限、交叉约束、敏感性、前置失败与测试落点；
3. Argon2 profile/readiness 分层、单一 permit owner、Hash/Verify/fake/rehash 覆盖、non-blocking 语义、permit 回收和资源乘积约束；
4. RecoveryMode 四项合取、普通流量 RecoveryGate 职责与完整状态矩阵；
5. factory 构造顺序、路由/middleware、事务/审计、错误、metrics、secret 生命周期及十八文件影响面；
6. fixed-fail、PG18 blocked、未实施/未部署/未 E2E、`S3.2 → S3.5 → S3.6` 和 production readiness 限制。

正面结果：

- `HAN_ARGON2_PROFILE_ID` 仅证明有界 token，不再被解释为生产批准；
- `MemoryKiB=19456..262144`、`Iterations=2..6`、`MaxConcurrency=1..4` 及 `uint64(memory)*uint64(concurrency)<=524288` 可直接编码并防溢出；
- process permit set 的唯一 owner、未来消费者复用、立即拒绝、成功/错误/panic-safe release 和同步屏障测试合同明确；
- RecoveryMode 只在前三项成立后调用一次，普通 credential mode 调用数为 0；普通流量 503 明确归独立 RecoveryGate/readiness；
- §9.2 恰列十八个既有文件，全部存在；未引入 schema、migration、依赖或第十九个文件；
- production `composeBusiness` 仍如实声明 fixed-fail，PG18 仍为 `blocked-for-execution`；未虚报 r25 实现、部署、E2E、profile evidence 或 checkpoint 完成。

## 4. Round 2：真实消费者与反向合同检查

实际反查：

- architecture r23、ADR-0012/0014/0016/0022、requirements/boundary、test-plan r06 I-F005-13～22、tasks 中 S3.2/S3.5/S3.6 分层；
- `cmd/han-server/main.go` 的 startup composition 与 production fixed-fail；
- `internal/identity/handler.go`、`service.go`、`login.go` 的公共错误、RecoveryMode、fake verify、rehash、事务、abuse bucket、审计和 observation 路径；
- `internal/security/argon2.go` 的现有 Hash/Verify、自描述参数、downgrade 拒绝和资源错误；
- `internal/platform/errors.go`、metrics 和 HTTP security 的公共契约。

反查结论：

- 当前 `cmd/han-server/main.go:460-465` 仍固定返回未配置错误，r25 对实施现状描述准确；
- 当前 `internal/identity/login.go:1257-1314` 可通过先检查 initial admin、`RecoveryRequired` 和 credential mode，再调用现有 adapter 完成最小修复，无需 schema；
- Argon2 Verify 在 derive 前执行 `rehashDecision`；强于 current profile 的存量参数先拒绝，因此 r25 current 参数上限可约束真正进入 derive 的存量摘要资源；
- capacity error 可沿现有 credential-error、rollback、独立失败审计和 Handler observation 路径实现 429、不写 abuse failure bucket及单次审计；十八文件范围足以承接；
- 唯一未闭环处是 429 Header：`handler.go:215-221` 仅在 `LoginRateLimitRetryAfter` 返回正 duration 时设置 `Retry-After`，`login.go:1769-1815` 的 duration 仅由 abuse decision 构造；r25 没有为 Argon2 capacity error 提供对应 duration 或稳定 response-boundary 策略。

## 5. 原 Findings disposition

| 原 Finding | 结论 | 依据 |
|---|---|---|
| `IA-R24-PC-R02-F001` | **STILL OPEN / MEDIUM（由 HIGH 收窄）** | benchmark/profile、单一 Gate、资源范围、调用路径和释放语义已修复；但新增 capacity 429 未定义既有强制 `Retry-After` 的值/来源/承载方式，公共映射仍非完整可实施合同。 |
| `IA-R24-PC-R02-F002` | **CLOSED** | `design.md:993,1354-1373,2234,2642,2891,2948` 一致固定 recovery-bootstrap 四项合取；普通流量 503 归独立 RecoveryGate/readiness，二者不是同一开关、接口或 owner，无相反残留。 |

## 6. Finding

### [MEDIUM] Argon2 capacity 429 缺少强制 `Retry-After` 的可执行语义

- **finding ID**：`IA-R24-PC-R02-F001`（r25 残留）
- **位置**：
  - `design.md:875,884`：所有 429 必须带 `Retry-After`；
  - `design.md:703,2427,2430,2639,2761`：Argon2 capacity typed error 固定映射 `request.rate_limited` 429，但未定义 Header；
  - `internal/identity/handler.go:215-221`；
  - `internal/identity/login.go:1769-1815`；
  - `internal/platform/errors.go:101,203-237`。
- **触发**：process permit set 已满，真实 credential Verify、未知用户 fake Verify 或 rehash Hash 立即返回 capacity typed error。
- **影响**：
  1. Service 可选择正确 code/status，但 Handler 没有可信 duration 可写入 Header，违反设计自身“429 必须带 `Retry-After`”的公共合同；
  2. 若实施者自行硬编码 duration，则值、owner、边界和测试预期不受设计约束，不同 429 来源可能出现不一致行为；
  3. r25 的 capacity 429、审计和 metrics 测试无法对公共响应作唯一断言，因此原 F001 仍未完整闭环。
- **最小建议**：在不改变既有 code/message 的前提下，明确 capacity rejection 的 `Retry-After` owner 和严格正值语义：由 typed capacity error 携带有界 duration，或由唯一 RateLimitResponseBoundary 按明确、可测试的稳定策略生成；同时说明该值不泄露 permit 数、profile 或参数，并保持 abuse bucket 的动态 duration 行为不变。
- **验证**：
  1. 分别令真实 Verify、fake Verify、temporary/permanent rehash Hash 遇到已满 Gate；
  2. 断言均返回 `429 request.rate_limited`、正整数 `Retry-After`、相同稳定策略且不含容量/profile/参数；
  3. 断言不写 abuse failure bucket，审计最多一次，outcome=`rate_limited`、reason=`resource_capacity`，无 permit 泄漏；
  4. 保持现有来源 bucket 429 的动态 `Retry-After` 回归通过。

## 7. 待决定与非阻断建议

### 待决定

- 仅需确定 Argon2 capacity 429 的 `Retry-After` owner 与稳定正值策略。该选择属于 r25 公共响应细化，不需要重开 Argon2 算法、profile、并发上限或 RecoveryMode 决策。

### 非阻断建议

- 测试规划同步时分别列出 startup fake Hash/Verify、运行期未知用户 fake Verify、真实 Verify、temporary/permanent rehash Hash 的 Gate 饱和用例，避免只证明 raw hasher 的共享 permit。
- 在实现映射中把 capacity error 的识别/公共包装具体落到现有 credential-error helper 所在文件，避免仅以“service.go”文件名掩盖 `login.go` 的实际错误路径；这不要求扩展十八文件范围。

## 8. Gate 与接力结论

### 设计 Gate：FAIL

- 原 F001：**STILL OPEN / MEDIUM**；
- 原 F002：**CLOSED**；
- 无 CRITICAL/HIGH，新发现或残留共 1 个 MEDIUM；
- 暂不建议把 r25 作为通过基线发起精确对象确认、同步 test-plan 或交实施；
- 不授权编码、测试执行、部署、E2E、checkpoint 或 Git 操作。

production observability readiness Gate 未被削弱：S3.2 application composition、S3.5 provider-neutral 部署材料、S3.6 真实部署/采集/告警演练仍保持分层；目标环境 Argon2 benchmark/profile 报告、内存/延迟预算和采用记录仍是 production readiness 前置；S3.6 前不得宣称 production observability readiness 或需求整体完成。

## 9. 限制与剩余风险

- 按授权未运行 Go test/race/vet/build、部署、真实 PostgreSQL 18、benchmark、scrape、告警或 E2E；
- Backend-Dev/Lead 的既有 PASS 只作背景，未被本轮重跑或外推为 r25 factory/Gate 已实现；
- 未验证真实 Argon2 目标环境 profile、容量、延迟、账号、receiver、平台或演练结果；
- 本结论只证明设计与当前消费者合同，不证明尚未实现的 parser、Gate、factory、adapter 或 router 正确落地；
- tasks/test-plan 仍引用 r22/r06，只有 r25 设计通过并获得精确对象确认后才可按授权同步，历史 r22 checkpoint 不批准 r25。

## 10. 复审条件

1. 补齐 Argon2 capacity 429 的 `Retry-After` owner、正值策略和测试断言；
2. 保持本轮已关闭的 benchmark/profile 分层、单一 process Gate、资源上下限、RecoveryMode/RecoveryGate 分离及十八文件边界；
3. 保持 fixed-fail、PG18 blocked、未实施/未部署/未 E2E 和无 r25 checkpoint 的真实状态；
4. 形成新的固定 revision/SHA256 后，由非作者 Reviewer 复审残留 finding 与真实消费者。
