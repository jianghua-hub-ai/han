# identity-auth design r23 — production composition 独立设计评审

**日期**：2026-10-23
**角色**：非作者独立 Reviewer
**对象类型**：design
**结论**：**FAIL**
**开放 findings**：CRITICAL 0 / HIGH 2 / MEDIUM 2 / LOW 0

## 1. 固定对象与边界

| 项目 | 固定值 |
|---|---|
| 被审对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| revision | `identity-auth-design-ir01-r23` |
| 要求 SHA256 | `0331C831D3E69D2182D2E0890FBD7354382173A79B34F97D397448F5C5076AB4` |
| 开始 SHA256 | `0331C831D3E69D2182D2E0890FBD7354382173A79B34F97D397448F5C5076AB4` |
| 写报告前 SHA256 | `0331C831D3E69D2182D2E0890FBD7354382173A79B34F97D397448F5C5076AB4` |
| 当前状态 | `draft`；r23 尚无精确对象 checkpoint，历史 r22 accepted 不自动批准 r23（design:2799-2800、2853） |

本报告只评审 r23 新增的 production 登录 business-route factory/config 合同及其对既有合同的影响；不修改 design、tasks、test-plan、ADR、代码或 checkpoint，也不把 Backend-Dev/Lead 已执行结果当成本轮命令执行。

## 2. 结论摘要

r23 正确保留了唯一 `cmd/han-server` composition root、`TraceMiddleware → OriginGuard → exact route → LoginHandler.Login` 的目标顺序、pool/listener 所有权、production fixed-fail 现状、ADR-0016/0022 事务与可观测性边界，以及 `S3.2 → S3.5 → S3.6` 三层 Gate。21 行配置矩阵数量、列结构和测试落点均存在，五个预计修改文件也处于 S3.2 原十四文件范围内。

但当前对象尚未达到“可实施、安全、无歧义”：

1. 多项认证/资源安全配置只有 `>0`、平台可表示或未定义的“获批准安全基线”，程序无法判定该基线，弱 token/HMAC 或极端内存/请求值仍会被视为合法并开放路由。
2. r23 声称 recovery-bootstrap 仍要求 `recovery_required`，但真实 Service 与 schema 没有该合取不变量；仅接入 `HAN_RECOVERY_MODE` 会保留认证绕过窗口。
3. source evidence 对 `RemoteAddr`、重复/歧义 XFF、地址规范化和 trust-all CIDR 没有形成唯一算法。
4. raw `http.ServeMux` 的 404/405 是 `text/plain`，与 ADR-0014“所有公共 REST 错误均为稳定 JSON + trace_id”冲突；现有测试映射只证明错误 method/path 不进 Handler，不能阻止公共错误漂移。

因此本轮设计 Gate 为 **FAIL**，不建议进入 r23 精确对象确认，也不应把 r23 作为已通过输入同步为 test-plan 新基线。

## 3. Round 1：设计对象直接检查

实际检查：

1. revision/change log、整改索引、factory 签名与输入对象（design:76、122、487-537）。
2. 启动、纯构造、DB/migration、factory、双 listener 与失败清理顺序（design:660-669）。
3. middleware/route、对象职责、资源所有权与五文件范围（design:622、764-785、2659-2661）。
4. 21 项配置矩阵逐行核对读取方/类型、必填/默认、范围、敏感性、失败和测试列（design:951-977）；结构解析得到 **21 行**，无重复键行。
5. Session issuer、source provider、RecoveryMode、fake probe 与秘密生命周期文字（design:537、769-771、937-949）。
6. 测试映射是否包含 production factory 不再固定失败、配置负例零副作用、route/middleware、source/recovery/session 与敏感扫描（design:2568-2572）。
7. 实现状态和确认边界：正文仍明确 `productionDependencies().composeBusiness` 固定失败、r23 factory 未实施、PG18 缺失、r23 未确认（design:2659-2661、2799-2800、2853），未虚假声明任务、部署或 E2E 完成。

直接检查确认的正面结论：

- factory 是 `cmd/han-server` 内部函数，没有第二入口、包级 service locator 或新增 module（design:526-537、784-785）。
- pool 仍由 lifecycle/startup unwind 关闭；factory 不 bind、不启动 goroutine、不关闭 pool（design:666、939-949）。
- application composition、部署材料、真实部署验证仍严格分层；S3.6 前禁止 production observability readiness/整体完成声明（design:2628-2630）。
- ADR-0016 的同事务审计、受控失败审计和非 DB observation 未被 r23 改写；ADR-0022 的预算、0.0.4、双 server 与真实部署 Gate 未被削弱。

## 4. Round 2：真实构造器与消费者反向核查

实际核查：

1. `cmd/han-server/main.go`：当前 `startupDependencies.composeBusiness` 签名与 r23 factory 方向兼容，但 production 仍在 main:460-465 固定返回未配置错误；现有 composition 已持有 logger/metrics/origin/failure auditor 和 lifecycle 接缝。
2. `internal/identity/handler.go`：`NewLoginHandler` 已要求 service/source/body limit/coordinator/observer/budget；登录先创建 attempt，再 decode、取 source、进入 Service（handler:75-102、141-215）。
3. `internal/identity/service.go`、`login.go`：现有接口可承接 Passwords、SessionIssuer、Abuse、RecoveryMode、SessionStore、Audit 与 coordinator；Postgres repository/bucket store/failure auditor 均已存在，不要求清单外生产 adapter。
4. `internal/session/store.go`、`internal/security/argon2.go`、`internal/audit/store.go`：现有构造器可直接复用；无需修改这些只读文件，但 Argon2 现有校验只覆盖算法最低关系和平台可表示性，不知道 r23 所称“获批准安全基线”（argon2.go:181-195）。
5. `internal/platform/http.go`、`http_security.go`：Trace 外包 OriginGuard 与现有 trace/logger/403 写后观察兼容；登录不经 Session/CSRF middleware与既有合同一致。
6. 反向核 requirements、identity boundary、architecture r23、ADR-0016、ADR-0022、test-plan r06 I-F005-13～22、tasks S3.2。真实部署 readiness 仍由 S3.5/S3.6 承接；本报告 findings 不要求把部署 provider/receiver 或真实 E2E 塞回 S3.2。
7. 本机 Go 1.27 标准库只读核对：`http.Request.RemoteAddr` 对 server 请求通常由标准库设置为 `IP:port`（`C:\Program Files\Go\src\net\http\request.go:290-297`）；`ServeMux` 未匹配方法/路径走 `http.Error`/`NotFoundHandler`（server.go:2753-2764），响应为 `text/plain; charset=utf-8`（server.go:2353-2374）。

## 5. Findings

### [HIGH] 安全关键配置缺少可由程序执行的安全包络

- **finding ID**：`IA-R23-PC-R01-F001`
- **对象/修订**：design / `identity-auth-design-ir01-r23`
- **位置**：`design.md:951-975, 2704-2708, 2797`；`internal/security/argon2.go:181-195`
- **现状**：矩阵对 body limit、Argon2、abuse policy、Session/CSRF entropy、TTL 等多项只要求 `>0`、可表示/可分配；两个 HMAC key 和 entropy 又要求满足未给值、未给版本、未给可查询来源的“获批准安全基线”。正文同时声称全部配置已具备范围和可实施合同。现有 Argon2 构造器只校验非零、`memory >= 8 * parallelism` 与平台容量，无法识别该外部“批准”。
- **触发**：部署注入 1-byte Session/CSRF entropy、极短 HMAC key、近零 Argon2 成本、极大 body/entropy/Argon2 参数，或把 abuse threshold/duration 配成实际上关闭保护的极值；这些值可满足当前书面 parser 条件。
- **影响**：弱 token/HMAC 可直接削弱认证、CSRF、来源和防暴力保护；极端 body/entropy/Argon2 值可在 listener 前或请求处理中造成内存/CPU 失控。由于值在语法上“合法”，现有 fail-closed 声明和配置负例测试不会阻止业务路由开放，属于安全与资源控制缺口。
- **建议**：把“实际部署目标值可调/TBD”与“代码必须执行的安全下限及资源上限”分开。为 token/CSRF/HMAC 长度、body、可分配随机长度、Argon2 各参数和 abuse/TTL 至少固定可测试的安全包络；若 Argon2 实际强度仍依 ADR-0012 目标环境基准，则明确可机读的批准 profile/version 或前置批准证据，不得让任意正数等价于“已批准”。必要时先修订对应安全决定，再回写 21 项矩阵和边界测试。
- **验证**：逐键新增最小值前一档、最大值后一档、交叉关系和资源极值负例；全部负例断言 connect/Ping/migrate/compose/listen 为 0；安全 profile 未确认时 production factory 必须确定性失败。合法基线与合法非基线仍应可通过。

### [HIGH] RecoveryMode 合同依赖一个当前不存在的 `recovery_required` 合取门禁

- **finding ID**：`IA-R23-PC-R01-F002`
- **对象/修订**：design / `identity-auth-design-ir01-r23`
- **位置**：`design.md:771, 945, 2414`；`internal/identity/login.go:1257-1318`；`migrations/0002_identity.sql:11-20`
- **现状**：r23 写明 maintenance 只为既有 recovery-bootstrap 分支提供事实，并“仍须同时满足 initial admin、recovery_bootstrap credential mode、recovery_required”。真实 Service 的条件是仅当 `RecoveryRequired == true` 且不是 initial-admin/recovery-bootstrap 时拒绝；当 `RecoveryRequired == false` 时，initial admin + `CredentialModeRecoveryBootstrap` 仍会进入 `RecoveryBootstrapAllowed` 并可继续验证。schema 也只分别限制枚举和 boolean，没有二者合取约束。现有测试的 recovery-mode fixture 正是 `RecoveryRequired=false` 仍进入 mode adapter（login_test.go:1033-1043、1831-1858）。
- **触发**：active initial admin 处于 `credential_mode=recovery_bootstrap`、`recovery_required=false`，同时 `HAN_RECOVERY_MODE=maintenance` 且恢复凭据匹配；该状态可来自不完整转换、存量漂移或人工维护。
- **影响**：恢复凭据可在缺少 design 声称的恢复要求事实时被接受，属于认证门禁绕过；接入 factory 不会自动修复该消费者行为。
- **建议**：在 design 中把 Service 条件写成明确且可直接编码的合取，并要求 `login.go` 在调用/接受 RecoveryMode 前验证 `IsInitialAdmin && RecoveryRequired && CredentialModeRecoveryBootstrap`。若作者选择由数据库保证该不变量，则必须扩大并重新授权 schema/migration 范围；当前五文件最小方案应采用 Service fail-closed 校验。
- **验证**：表驱动覆盖 `normal/maintenance × recovery_required true/false × initial admin true/false × credential mode`；只有全部必要条件满足且 mode=maintenance 时可走恢复凭据，其余统一失败且保持现有公共错误、审计和 observation 合同。

### [MEDIUM] 可信来源解析与代理信任算法仍有生产歧义

- **finding ID**：`IA-R23-PC-R01-F003`
- **对象/修订**：design / `identity-auth-design-ir01-r23`
- **位置**：`design.md:770, 944, 974, 2362, 2568`
- **现状**：合同写“从 `RemoteAddr` 严格解析 IP”，但 Go server 通常提供 `IP:port`；未规定 host/port 拆分、IPv4-mapped IPv6 归一化、zone 处理、重复 `X-Forwarded-For` header、多个逻辑链、最大 hop 数，以及 `0.0.0.0/0`/`::/0` 等 trust-all CIDR 是否必须拒绝。当前矩阵只覆盖 canonical CIDR、重复项、host bits、空项和 `none` 混用。
- **触发**：正常生产请求携带标准 `IP:port` RemoteAddr；可信代理传递重复 XFF 字段或 IPv4/IPv4-mapped IPv6 等价地址；配置误填 trust-all CIDR。
- **影响**：实现若按字面 `netip.ParseAddr(RemoteAddr)` 会让正常登录 source evidence 全部失败；不同实现者对重复 header/地址规范化的选择可产生不同 fingerprint；trust-all 或表示变体可让客户端影响来源归属、规避 source bucket 或污染审计证据。
- **建议**：固定唯一算法：生产 RemoteAddr 必须按 host:port 拆分并严格解析；定义 zone 与 IPv4-mapped IPv6 的拒绝/归一化；规定 XFF 只允许的 header 实例/链形态、hop 上限和歧义拒绝；明确禁止 trust-all 前缀，或要求更强的显式批准机制。所有失败均保持去敏 source-evidence error。
- **验证**：增加真实 `IP:port` IPv4/IPv6、mapped address、zone、重复 header、空白/多链、全可信链、超长链、trust-all CIDR 与可信/不可信 peer 表驱动；断言唯一 canonical fingerprint、原始地址不出 error/log/metric/audit。

### [MEDIUM] raw ServeMux 的 404/405 会破坏公共 REST 错误合同

- **finding ID**：`IA-R23-PC-R01-F004`
- **对象/修订**：design / `identity-auth-design-ir01-r23`
- **位置**：`design.md:666, 811-845, 949, 2164, 2570`；`framework.md:232-263`（ADR-0014）
- **现状**：factory 明确返回 `TraceMiddleware(originGuard.Protect(mux))`，局部 raw `http.ServeMux` 只注册 `POST /api/v1/sessions`。Go 1.27 对 method mismatch/unknown path 默认返回 `text/plain` 405/404，没有稳定 JSON `code/message/trace_id`；r23 测试只要求错误 method/path 不进入 Handler，没有断言公共 envelope。OriginGuard 包在 mux 外还会使无可信 Origin 的未知路径先变成 403，正文未说明这是有意的公共行为。
- **触发**：带可信 Origin 的 `GET /api/v1/sessions`、`POST` 到未知业务路径，或无 Origin 的未知路径。
- **影响**：公共客户端收到裸字符串且无 trace_id，违反 ADR-0014“所有公共 REST 错误统一稳定 JSON”与 r23“公共 REST/错误不变”的声明；不同 Origin 条件还会改变 unmatched route 的状态语义。
- **建议**：在不改变登录主链顺序的前提下，为 method/path fallback 定义稳定公共 code、status、message、trace 和 OriginGuard 相对路由匹配的明确语义；若复用共享 top-level router，则必须在 design 中写出真实 owner，而不能让本地 raw mux承担默认错误。新增公共 code 或责任边界时同步设计注册表，必要时补相应决策引用。
- **验证**：HTTP 契约测试覆盖正确 POST、错误 method、未知 path、可信/缺失/错误 Origin；逐例断言 status、JSON envelope、trace header/body、Allow header、Handler/Service/attempt/metrics 调用次数及 Origin rejection 计数，且无 stdlib plain-text fallback。

## 6. 待决定

1. **安全配置批准机制**：实际 Argon2 强度、token/CSRF/HMAC 最低强度及 body/abuse/TTL 安全包络由安全责任人/技术代表确认；Reviewer 不替代该决定。容量目标可以继续 TBD，但不能让未决定状态被任意正数冒充。
2. **公共 route fallback 合同**：由设计/架构责任人确认 method/path 的稳定 code 与 OriginGuard 相对匹配顺序。若需要新增公共错误，不得由 Backend-Dev 在实现时临时发明。

## 7. 非阻断建议

1. `preparedLoginDependencies.IDs/Clock`、factory 步骤中再次构造 `RandomUUIDGenerator/SystemClock`，以及 `businessComposition.failureAuditFor` 与已 bounded auditor 的责任有重复；建议在修订时收敛为单一 owner，减少实现分叉。
2. 明确 Go 的环境变量和 fake probe 最终会经过不可原地清零的 `string`：可承诺的是不日志/不错误回显、减少复制、使用后释放全部引用；HMAC 临时 `[]byte` 应从物化时安装 `defer clear`，覆盖所有成功/失败返回。不要把“丢弃引用”表述成已证明物理清零。

## 8. Gate 与接力结论

### 设计 Gate：FAIL

- 有 2 个 HIGH、2 个 MEDIUM finding，均需回到 design 修订并由非作者 Reviewer 复审。
- **不建议进行 r23 精确对象确认**；历史 r22 checkpoint 不得借用。
- **暂不可作为已通过对象交 Test-Engineer 同步 test-plan 基线**。可以把本报告作为返工输入，但 I-F005/r23 新场景应在修订对象 PASS 后再正式同步。
- 不关闭 F005、S3.2 或 B4，不构成编码、测试执行、部署、E2E、Git、provider/账号或发布授权。
- production observability readiness Gate 未被本报告放宽：仍须 S3.2 application composition、S3.5 provider-neutral 部署材料、S3.6 真实采集/scrape/告警/Dashboard/receiver/runbook 演练依次通过。

## 9. 限制与剩余风险

- 按授权未运行 Go、race、build、contract、PostgreSQL、部署或 E2E；Lead 提供的既有 PASS 只作为背景，不是本轮复跑。
- PostgreSQL 18 环境缺失，相关专用矩阵仍为 `blocked-for-execution`；本报告没有把该限制外推为 factory 设计通过或失败的唯一依据。
- 未验证未来真实配置值、目标环境 Argon2 基准、真实反向代理行为、日志/metrics backend 或生产 receiver。
- 本轮只读检查了当前代码与直接消费者；若修订选择 schema 约束、新公共错误、共享 router 或新的安全 profile，必须重新核文件范围、ADR/architecture 影响和 tasks/test-plan。

## 10. 复审条件

复审至少需要：

1. 新 revision 与固定 SHA256。
2. F001 的可执行安全包络/批准机制及完整边界测试映射。
3. F002 的 recovery-required 合取门禁与状态矩阵。
4. F003 的 RemoteAddr/XFF/CIDR 唯一算法与歧义负例。
5. F004 的 method/path JSON error 合同、Origin 顺序和测试。
6. 继续保持当前五文件或明确取得新的文件/ADR/architecture 范围授权；继续声明当前实现与部署/E2E 未完成。

