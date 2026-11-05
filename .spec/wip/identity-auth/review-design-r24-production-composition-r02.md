# identity-auth design r24 production composition 独立复审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **对象类型**：design
- **结论**：**FAIL**
- **Finding 统计**：CRITICAL 0 / HIGH 1 / MEDIUM 1 / LOW 0

## 1. 固定对象与边界

| 项目 | 固定值 / 结果 |
|---|---|
| 被审对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| revision | `identity-auth-design-ir01-r24` |
| 要求 SHA256 | `F65462C4132D71E27D274F7B41B047C02871E4540724C14970BE1355114D62F1` |
| 开始 SHA256 | `F65462C4132D71E27D274F7B41B047C02871E4540724C14970BE1355114D62F1` |
| 原评审 | `.spec/wip/identity-auth/review-design-r23-production-composition-r01.md` |
| 原评审 SHA256 | `C98188DA1741949A08912D45B13392FAAA67F202ED64DCA824019CE188F623D5` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |

本轮仅评审 r24 对 `IA-R23-PC-R01-F001～F004` 的修订及其直接回归影响。design、代码、ADR、requirements、boundary、architecture、test-plan、tasks 均只读；唯一写入本报告。共享工作树存在大量既有改动，本轮未回滚、清理、格式化或覆盖任何现状。

## 2. 结论摘要

r24 已实质关闭原 F003、F004，并修复了原 F002 指出的 `RecoveryRequired=false` 绕过：四项合取、调用次数、JSON 404/405、Trace/Origin/router 顺序、XFF 与 mapped-prefix 算法均已达到可编码程度。

但原 F001 仍不能关闭。r24 一方面把未经目标环境 benchmark 的 Argon2 tuple 写成无 override 时应使用的生产建议值，直接冲突于 accepted ADR-0012 和 architecture r23 的“参数必须经目标环境基准确定、数值仍 TBD”；另一方面，设计自身仍要求“进程级信号量 + 全局配置”，21 项配置矩阵却没有 Argon2 并发上限、入队/拒绝策略或相应验证。当前允许的单次上限达到 `1 GiB × 10 iterations`，现有 hasher 也没有并发保护，不能称为完整的可执行资源安全包络。

此外，recovery-bootstrap 的核心绕过虽已修复，但正文仍同时要求“普通流量在 maintenance 统一 503”和“前三项合取不成立时不得调用 RecoveryMode”；在没有另一个明确外层 RecoveryGate 的当前 factory/文件映射下，两条规则无法同时实现。

因此设计 Gate 为 **FAIL**，r24 暂不可交精确对象确认、测试规划同步或 Task-Planner 作为已通过基线。

## 3. Round 1：设计对象直接检查

实际检查：

1. revision、状态流转、整改记录、change log、确认记录及 fixed-fail/PG18/授权声明；
2. 21 行生产登录配置矩阵的读取方、类型、必填/默认、建议值、范围、交叉关系、敏感性、失败和测试落点；
3. factory 构造顺序、secret 生命周期、Session issuer、RecoveryMode、来源证据、路由/middleware 与资源所有权；
4. §9.2 十六文件范围：共 16 行且全部现存；相对原十四文件只增加 `internal/platform/errors.go` 与 `internal/platform/errors_test.go`；
5. r24 继续明确 production `composeBusiness` 固定失败、PG18 缺失、未实施/部署/E2E、无 r24 checkpoint，未借历史 r22 checkpoint 批准 r24。

直接检查确认的正面结果：

- body、HMAC、entropy、abuse、TTL 的主要数值边界与交叉关系已补齐；
- `idle <= absolute`、HMAC 域隔离、secret 去敏及临时 `[]byte` 清理路径明确；
- `S3.2 → S3.5 → S3.6` 保持不变，S3.6 前仍禁止 production observability readiness 和需求整体完成声明；
- 未虚报任务、实现、部署、采集、告警演练、E2E 或 checkpoint 已完成。

## 4. Round 2：真实消费者与反向合同检查

实际反查：

- requirements 与 identity boundary 的登录、初始管理员和恢复维护模式边界；
- architecture r23 的 Argon2 benchmark/TBD、公共错误、单进程 composition 与 production observability readiness 合同；
- ADR-0012、ADR-0014、ADR-0016、ADR-0022；
- test-plan r06 I-F005-13～22 和独立 I-OBS-DEP-01～06 Gate；
- `identity-auth-S3.2` 当前任务边界；
- `cmd/han-server/main.go`、identity Handler/Service/login、Argon2 hasher、platform error/security/metrics/lifecycle 的真实类型与构造器。

反查结果：

- 当前 `cmd/han-server/main.go:460-465` 仍固定返回 `identity business route provider is not configured`，r24 对现状描述准确；
- 当前 `internal/identity/login.go:1257-1314` 确有原 F002 绕过；r24 的四项合取可在现有 Service/login 路径最小修复，不需要 schema/migration；
- 当前 `internal/platform/errors.go` 无可复用的通用 route 404/405 code；新增两个稳定 code 及其同文件测试是最小公共注册表变化；
- mapped IPv6 prefix 的 `bits 96..128 → IPv4 bits-96` 变换数学正确；RemoteAddr/XFF/right-strip/16-hop/trust-all/等价前缀合同没有发现新的阻断缺口；
- 当前 `internal/security/argon2.go:181-195,256-270` 只验证算法最低条件与平台可表示容量，没有 benchmark profile 或并发信号量，不能补足 r24 遗漏的资源 Gate。

## 5. 原 Findings disposition

| 原 Finding | 结论 | 依据 |
|---|---|---|
| `IA-R23-PC-R01-F001` | **OPEN / HIGH** | 大部分逐键边界已补齐，但 Argon2 建议生产 tuple 绕过 ADR-0012 benchmark 前置，且缺失设计已承诺的全局并发上限/信号量合同；见新 F001。 |
| `IA-R23-PC-R01-F002` | **CLOSED** | `design.md:981,1346-1359,2220,2625` 已固定四项合取，特别要求 `RecoveryRequired=false` 时 adapter 和 credential verify 均为 0；现有字段足够，无需 schema。maintenance 普通流量的相反残留另记新 F002。 |
| `IA-R23-PC-R01-F003` | **CLOSED** | `design.md:975-980,1010,2416,2625` 已固定 strict host:port、zone 拒绝、Unmap、wire-order、多 header、16-hop、右剥离、trust-all/等价前缀拒绝及隐私边界。 |
| `IA-R23-PC-R01-F004` | **CLOSED** | `design.md:609-625,854-874,2398,2624` 已固定 Trace 最外层、router-before-Origin、JSON 404/405、`Allow: POST` 和零下游调用；现有 registry 确无可复用通用码。 |

## 6. Findings

### [HIGH] Argon2 production baseline 违反既有 benchmark 决策，且缺失进程级资源 Gate

- **finding ID**：`IA-R24-PC-R02-F001`
- **位置**：`design.md:987-996,2413-2416,2592,2764-2771,2865,2921`；`decisions/identity.md:210,223,230`（ADR-0012）；`architecture.md:64,162,1274`；`internal/security/argon2.go:181-195,256-270`
- **触发**：
  1. 没有目标部署环境 benchmark/批准 profile 时，按 `design.md:2764,2771` 直接显式配置建议 tuple `64MiB/t=3/p=4/salt=16/key=32` 并开放 production route；
  2. 配置合法上限 `MemoryKiB=1048576`、`Iterations=10`，并发登录同时进入 Argon2 Hash/Verify；21 项矩阵没有并发容量键、信号量、排队或拒绝合同。
- **影响**：
  - accepted ADR-0012 明确要求参数经目标环境安全与延迟 benchmark 确定，且禁止把示例值变成生产承诺；r24 未修订 ADR/architecture 就把建议值升级为无 override 时的生产值；
  - 单次最高 1 GiB、10 次迭代且无进程级并发上限，可在认证入口形成 CPU/内存耗尽甚至 OOM；账号/来源 bucket 不能替代跨来源、跨账号的全局 Argon2 容量控制；
  - r24 自身 §4.4 已承诺“进程级信号量 + 全局配置”，矩阵和十六文件验证映射却未给出可实现合同，因此原 F001 的“资源上限可执行”目标仍未完成。
- **建议**：
  1. 保留已补齐的密码学最低边界，但在取得目标环境 benchmark 前，不得把 Argon2 tuple 写成可直接开放生产的批准基线；可采用版本化 benchmark/profile 证据作为 production 前置，或先修订 ADR-0012/architecture；
  2. 增加显式、有限的进程级 Argon2 并发配置及 owner，固定范围、信号量共享范围、排队或尽早拒绝、429/503 映射、context cancellation 和 shutdown 行为；
  3. 由目标环境内存/延迟预算收紧单次参数上限；若无法在十六文件边界内正确实现，先升级文件范围，不得把 limiter 留为容量 TBD 后仍开放路由。
- **验证**：
  - 目标环境 benchmark/profile 的可复核版本与采用记录；
  - 参数及并发配置的端点、越界、关系和缺失矩阵，非法值断言 connect/Ping/migrate/compose/listen 全为 0；
  - 以同步屏障而非 sleep 验证并发上限、排队/拒绝、取消、Hash/Verify 共用同一 limiter、无 permit 泄漏；
  - 合法生产 profile 下验证内存/延迟预算后再允许 production factory 通过。

### [MEDIUM] maintenance 普通流量 503 与 RecoveryMode 零调用矩阵仍相互冲突

- **finding ID**：`IA-R24-PC-R02-F002`
- **位置**：`design.md:981,1340,1346-1359,2220,2625`；`internal/identity/service.go:126-129`；`internal/identity/login.go:1250-1314`
- **触发**：用户处于 permanent/bootstrap/temporary 等非 recovery-bootstrap 路径，同时 `HAN_RECOVERY_MODE=maintenance`。
- **影响**：
  - `design.md:1340` 要求普通流量在 maintenance 统一 503，实施必须读取 maintenance 状态；
  - 同一设计又要求只有先证明 initial admin + recovery_required + recovery_bootstrap 后才可调用 RecoveryMode，其他状态调用数必须为 0；
  - 当前唯一接口是 recovery-bootstrap-specific `RecoveryBootstrapAllowed`，r24 factory/十六文件映射未定义另一个外层 RecoveryGate。实现者只能选择违反 503 行为或违反零调用矩阵，恢复期间的公共流量隔离语义不唯一。
- **建议**：明确二选一且与 boundary/architecture 同步：
  1. 普通流量由独立、外层 RecoveryGate/readiness 路由门禁统一 503，并明确 owner、调用顺序、文件与测试；`LoginRecoveryMode` 继续只服务四项 recovery-bootstrap 合取；或
  2. 若 S3.2 本轮只承诺 recovery-bootstrap 校验，则删除/收窄 §3.3.4 的“普通流量 maintenance 统一 503”，不得暗示当前 adapter 承担全局维护门禁。
- **验证**：表驱动同时覆盖普通 credential modes 与 recovery-bootstrap 四项矩阵；逐例断言公共状态/code、RecoveryMode/RecoveryGate 调用次数、credential verifier 次数、事务/Session/Audit/observer 次数，证明 maintenance 隔离与 `RecoveryRequired=false` 回归均成立。

## 7. 待决定与非阻断建议

### 待决定

1. Argon2 production 参数是保持 ADR-0012 的“benchmark/profile 前置”，还是通过正式 ADR/architecture 修订接受一个无需目标环境 benchmark 的固定基线。Reviewer 建议保持现有 ADR，不用 r24 建议值隐式改写 accepted 决策。
2. maintenance 普通流量由独立 RecoveryGate 承担，还是从本轮 S3.2 登录 factory 合同中删除该全局语义。Reviewer 建议使用独立外层 Gate，保持 recovery-bootstrap adapter 单一职责。

### 非阻断建议

- 为两个 key-version token 增加有限长度，并为 `HAN_TRUSTED_PROXY_CIDRS` 增加总长度/项数上限；当前语法和 OS 环境大小提供了现实约束，但补齐显式容量边界可使“逐键 executable envelope”陈述更准确。
- 修正 `design.md:1348` 的“除最后两行外”措辞：表中实际最后三行（normal、maintenance、adapter error/unknown）的 adapter 调用数均为 1。

## 8. Gate 与接力结论

### 设计 Gate：FAIL

- 存在 1 个 HIGH、1 个 MEDIUM finding；
- 不建议发起 r24 精确对象确认；
- 不建议把 r24 作为已通过基线同步 test-plan 或交 Task-Planner；
- 不授权实施、测试执行、部署、E2E、checkpoint 或 Git 操作。

production observability readiness Gate **未被削弱**：S3.2 application composition、S3.5 provider-neutral 部署材料、S3.6 真实采集/scrape/告警/Dashboard/receiver/runbook 演练仍须分层完成；S3.6 前不得声明 production observability readiness 或需求整体完成。

## 9. 限制与剩余风险

- 按授权未运行 Go test/race/vet/build、部署、真实 PostgreSQL 18、scrape、告警或 E2E；
- Backend-Dev/Lead 的既有 PASS 只作为背景，没有被本报告重跑或外推；
- 未评审 r24 之外的完整 identity-auth 设计历史，也未验证真实 provider/receiver、账号、排班、费用和容量；
- 本轮结论只证明文档与当前消费者合同，不证明尚未实现的 factory、配置 parser、adapter 或 router 已正确落地。

## 10. 复审条件

1. 修复 `IA-R24-PC-R02-F001`：与 ADR-0012/architecture 对齐 benchmark/profile 前置，并补齐 Argon2 全局并发资源 Gate；
2. 修复 `IA-R24-PC-R02-F002`：统一 maintenance 普通流量与 RecoveryMode 调用次数语义；
3. 保持原 F002～F004 已关闭合同、十六文件最小边界或对新增范围重新授权；
4. 形成新的固定 revision/SHA256 后，由非作者 Reviewer 复审对象直接合同与真实消费者。
