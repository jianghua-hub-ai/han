# identity-auth design r17 / F005 独立设计评审

- 评审角色：非作者独立 Reviewer
- 评审日期：2026-10-22
- 对象类型：design
- 逻辑修订：`identity-auth-design-ir01-r17`
- 结论：**NOT PASS**
- Gate 建议：**不建议通过 r17 Design Review Gate**
- 授权边界：本报告只给出设计评审结论，**不授权 F005 编码、测试执行、任务状态更新或发布**。

## 1. 固定对象与版本验证

### 1.1 设计与 checkpoint

| 对象 | 验证结果 |
|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | 普通文件 SHA256 `2E329F300617312A39597FEE95751DE0456A9DA72DFAA872B5E251F0EA6191C4` |
| 逻辑修订 | 文档 metadata 与正文均为 `identity-auth-design-ir01-r17` |
| checkpoint decision | `decision-0bc3a0de5aa6e86fae6d7fa874c26c0fb29437762310c5e925b62e1493fabca3` |
| checkpoint record | `.add/records/checkpoints/ir-01-content-community/design/decision-0bc3a0de5aa6e86fae6d7fa874c26c0fb29437762310c5e925b62e1493fabca3.json`；普通文件 SHA256 `34469F3556E6ABBA0851365F29042858D4009299FCFB7A87AA9A32A221B6` |
| `objectRef.digest` | 按 `.add/tool/add.mjs` 的 `artifactSections` / `artifactObjectDigest` 规则排除 `## 确认记录` 区后独立复算为 `sha256:41bb2c83f63fb26c8ea8da7e4b01a8ca3cc9a53f918d25b7a4c9893610a0bf87`，与 checkpoint 匹配；摘要输入 252875 bytes |
| `documentRecordRef.digest` | 按 checkpoint confirmation block 规则独立复算为 `sha256:0d3ca0061687dd881f47ba347409f41d6c6cb6a30dff6ced6904c70041d16c86`，与 checkpoint 匹配；摘要输入 1859 bytes |
| checkpoint scope | 只接受 r17 作为 test-plan 修订、独立评审和后续 R 对账基线；明确不授权 F005 实施 |

普通文件 SHA256 与 checkpoint 的对象摘要规则不同；本报告未用普通文件 SHA 冒充 `objectRef.digest`。

### 1.2 固定上游与下游

以下完整文件 SHA256 与 checkpoint/委托固定值匹配：

- requirements：`999CBA3C0CD25D2760E400A41D2A373611406854E76E24E453321913D71CBC4F`
- architecture：`1901EE5D80D9400B6E38F01C4A84F64020782D73EE9D6A83213D43F3A076A36C`
- boundary：`FB5197625EAFC07BF0A98E5F09ABD7C60E018EE6507BC1D47B88DC40B651767C`
- stories：`D8AC23A5EB7BF535B2111BC3411B6874E455065B237695470B3ADDEFDAAB4AD5`
- decisions index：`C2211EF9EAE3B722A9B115D1DEA558CAE0A729A36BD7CBAF7C8A1E504E4AC116`
- test-plan r02：`0215043EECF2504756D84A5C2959BCCC2CE0183A3E96CA2A8B2605004EAA7560`
- test-plan r02 独立评审：`5197DE52C393B61FA57D1BB13054CE2EC3D26206009AB7778A1A918CEA9BDD59`

开始评审及本报告落盘前，报告路径均不存在；分支为 `main`，HEAD 为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

## 2. 实际读取与评审依据

完整读取并应用：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/task-execution-workflow.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`

完整读取固定 design、checkpoint、requirements、architecture、identity boundary、stories、decisions index，以及 F005 实际引用的 ADR-0005、ADR-0012、ADR-0014、ADR-0016。读取 test-plan r02、其独立评审、design r09 评审链和 tasks/Plan r01/r02 规划评审。

为验证接缝、现状假设与可实施性，有界读取了：

- `internal/identity/handler.go`
- `internal/identity/service.go`
- `internal/identity/login.go`
- `internal/identity/login_test.go`
- `internal/identity/login_integration_test.go`
- `internal/platform/http_security.go`
- `internal/platform/http_security_test.go`
- `internal/audit/store.go`
- `internal/platform/logging.go`
- `cmd/han-server/main.go`

该消费者读取不是完整代码 Review，也没有用现有实现反向改写已确认设计。

## 3. 两轮实际检查

### Round 1：设计对象直接检查

逐段检查了 F005 的目标/非目标、术语、identity/platform/audit/composition 所有权，登录成功/统一失败/锁定/会话轮换，ID/time/guard 前置，原事务与失败审计的提交、回滚和不确定结果矩阵，写后 finalizer，Origin 拒绝观测，production fail-closed，API/错误/审计字段，敏感数据和低基数边界，PostgreSQL 18 假设，以及验证与授权 Gate。

本轮发现：

- F001：Origin 安全指标的采样点和标签合同在同一设计内冲突。
- F002：同步 observer 对阻塞、取消、截止时间及资源释放没有可执行上界。
- F003：项目检查命令的当前事实已过期。

### Round 2：上游、ADR、下游与消费者回归

1. 将 F005 逐项回溯到 requirements、architecture、boundary、S03 stories 和 ADR-0016 等实际引用 ADR。
2. 核对 test-plan r02 的 I-F005-01～12、局部/组件/contract/PostgreSQL 18/race/vet/build 计划及其独立评审。
3. 核对 design r09 历史评审链，只检查 r17 新增或变更相关回归；未继承旧 PASS。
4. 核对 tasks/Plan r01/r02 只承担交接与 Gate，不把规划评审冒充设计正确性评审。
5. 对指定代码消费者检查当前接缝：Handler 仍忽略 response write error；Audit ID 仍在较晚的 Service audit append 内生成； commit/rollback unknown 尚无 r17 核对合同；OriginGuard 尚无 observer；production main 尚未装配 identity 路由和 logger/metrics observer。以上与 r17 的“尚未交付”判断相符。

Round 2 没有推翻 F001～F003，也未发现需要扩大为架构重设计的新问题。test-plan r02 按 r17 的多数正确合同形成了有效消费者约束，但不能消除设计 SSOT 内的冲突或补足生产阻塞上界。

## 4. Findings

### F001 — MEDIUM — Origin 指标采样点与标签合同自相矛盾

**位置**

- 冲突项：`design.md:2193-2199`，尤其 `:2197`
- 同文档主合同：`:671-672`、`:816`、`:821-827`、`:1087-1093`、`:1127`、`:1185`、`:1981-1982`、`:2154`、`:2359`、`:2546`、`:2588`

**违反依据**

设计 SSOT 必须给出单一、可实现和可验证的接口/采样合同；消费者不能依靠自行选择冲突段落来实现。

**触发场景**

OriginGuard 判定拒绝并写出 `403 auth.origin_rejected`。§4.7 指标表要求在“写原 403 前”递增，且标签只有 `reason`；其他 r17 主体要求在唯一 403 写入尝试返回后调用，并携带 `reason,response_write`。

**影响**

- 按 §4.7 实现时无法记录响应写入成功或失败，直接破坏 `response_write=ok/failed` 合同。
- observer 顺序测试、指标标签测试和 production composition 会得到两种互斥实现。
- test-plan I-F005-09 采用“写后 + `response_write`”合同，不能替代设计对象消除冲突。

**修复建议**

由 identity-auth Feature-Designer/当前设计 owner 将 §4.7 的 `han_platform_origin_rejections_total` 行统一为：

- 唯一 403 写入尝试返回后采样；
- 标签 allowlist 为 `reason,response_write`；
- 继续禁止原始 Origin/Header/IP、trace 和身份字段成为标签。

不要借此选择 metrics backend、SDK、阈值、dashboard 或值守入口。

**复审验证**

1. 全文搜索该指标，只有一个采样顺序和标签集合。
2. 与 OriginRejectionObserver 接口、composition、验证表及 I-F005-09 一致。
3. 不改变公共 403 envelope 和 identity Handler 不被调用的合同。

### F002 — MEDIUM — 同步 observer 缺少有界阻塞、取消和资源释放合同

**位置**

- `design.md:669-672`
- `design.md:778-784`
- `design.md:823-827`
- `design.md:1979-1982`
- `design.md:2032`
- `design.md:2154-2156`
- `design.md:2189`
- `design.md:2359`

**违反依据**

`code-review-checklist.md` 要求异常、并发、取消与设计一致，并要求资源/状态所有权和失败路径收敛。用户指定焦点也明确要求检查 finalizer/observer 的失败语义、超时、取消与资源释放。

**触发场景**

`LoginOutcomeObserver`、`OriginRejectionObserver` 或其 logger/metrics adapter 同步进入后永久阻塞，既不返回 error，也不响应 request context 的取消。

**影响**

- HTTP Handler goroutine 和连接可无限占用，形成可被放大的资源耗尽/DoS 面。
- 对小响应，完成 `ResponseWriter` 调用不保证客户端已在 Handler 返回前及时收到缓冲数据；“不改变响应”不能覆盖无限延迟。
- `han_identity_login_duration_seconds` 明确排除 observer 时间，监控无法反映这段无限延迟。
- 现有“阻塞 fake 最终由测试释放”只验证调用顺序，没有定义生产上界、取消收敛或 goroutine 回收。

**修复建议**

保持 provider-neutral，不选择 Q13 backend 或新增依赖：

1. 明确两个 observer 及其 sink 必须遵从传入 context 的 cancellation/deadline。
2. 明确 production composition 必须提供有界调用预算或等价上限；预算来源和 owner 必须可实现、可测试，但不必在本次选择具体 backend。
3. 取消/超时后一次失败即返回，不重试、不切换第二 sink、不启动遗留 goroutine、不二次写响应、不重复主事务或审计。
4. 规定 adapter 对 timer、goroutine、连接或其他临时资源的释放责任。
5. 增加“sink 持续阻塞直到 context cancel”的确定性测试，断言 Handler 有界返回、调用次数不变且测试 goroutine 全部回收。

**复审验证**

- 接口、运行语义、composition 和测试章节对 cancellation/deadline 使用同一合同。
- 测试不依赖人工无界等待；可证明超时后无重试、无二次响应、无资源遗留。
- 不编造 logger/metrics backend、阈值、dashboard、paging 或新 SDK。

### F003 — LOW — Manifest 检查命令事实过期

**位置**

- `design.md:2388`
- `design.md:2439`
- `design.md:2489`
- 对照 `add.manifest.yaml:18-25`、`:42-49`

**违反依据**

设计的当前事实、实施 Gate 和验证入口必须与现役项目配置一致，不能把已登记检查误写为待确认。

**触发场景**

实施者依据 r17 的“项目尚未登记检查命令”“检查命令待确认”安排 F005 验证。

**影响**

- 可能漏跑已登记的项目检查，或错误等待一个已经完成的配置决定。
- 容易把七项通用检查与额外的 PostgreSQL 18 专用 integration 环境阻塞混为一谈。

**修复建议**

同步现役七项检查：

1. `gofmt -l .`
2. `go mod verify`
3. `go vet ./...`
4. `go build ./...`
5. `go test -tags=contract ./...`
6. `go test -race ./...`
7. `go test ./...`

PostgreSQL 18 专用 integration 命令和环境仍应单独标为 `blocked-for-execution`，不得写成已经运行或通过。

**复审验证**

- 全文不再声称 Manifest 未配置项目检查命令。
- 七项项目检查与 PG18 专用阻塞保持两个独立 Gate。

## 5. 其余重点范围结论

除上述 findings 外，本次未发现以下合同发生阻断性缺陷：

1. F005 的目标、非目标、术语和 identity/platform/audit/composition 所有权总体清楚。
2. ID/time/一次性 guard 早于 source evidence；Handler 与 Service 互斥认领失败审计。
3. `insert_failure` 只在原事务未提交已证明时使用；commit unknown 和 Append 后 rollback unknown 只读核对完整 AuditEvent；Append 前 rollback unknown 禁止查询或补写。
4. 写后 finalizer 不覆盖原 cause、不撤销已提交 Session/Audit、不写第二响应。
5. Origin reason 封闭、敏感值禁止进入日志/指标、指标标签总体低基数；F001 是局部冲突。
6. production 缺 observer/logger/metrics 时 composition fail closed；运行期 sink error 不改变业务结果；F002 是未覆盖“永不返回”的失败类。
7. 未编造 logger/metrics backend、阈值、dashboard、值守入口或新依赖，Q13 保持 TBD。
8. 公共 API、统一登录失败、Cookie/Session/CSRF、枚举防护和暴力防护边界没有因 F005 被擅自改变。
9. PostgreSQL 18 环境未确认；设计没有把 integration 写成已通过。
10. I-F005-01～12 对事务不确定、响应写后观测、Origin、五类 outcome、敏感字段和 production composition 形成了可追踪主体；F002 修复后需补取消/有界阻塞验证。
11. r17 仍为 draft，保留独立 Review 和明确编码授权边界。
12. r09 历史 PASS 未被自动继承；只对 r17 相关回归作了必要核对。

## 6. 待决定

本次没有提出新的产品或架构方向决定，也不替 Lead 接受风险。

以下既有未决项继续保留，不能被本评审补写：

- Q13 的 logger/metrics backend、Histogram buckets、日志留存、外部管道健康、告警阈值、dashboard、paging/runbook 和值守入口。
- PostgreSQL 18 可用环境及专用 integration 实际执行。

F002 要求的是 provider-neutral 的取消和有界资源合同，不等于替项目选择上述 backend。

## 7. 非阻断建议

1. 为 Origin observer 维护一处规范采样合同，其余章节引用该合同，降低再次漂移概率。
2. 在 F005 验证矩阵中单列“context canceled/deadline exceeded”与“sink 永不主动返回”，避免只用最终人工释放的阻塞 fake。
3. 项目检查命令优先引用 Manifest 为 SSOT，并仅在设计中说明 F005 额外的 PG18 专用条件。

## 8. 限制与剩余风险

- 本次是静态设计评审；未执行 Go 单元、integration、contract、race、vet、build 或 E2E。
- PostgreSQL 18 环境未确认，所有依赖真实事务状态的验证仍为 `blocked-for-execution`。
- 未接入真实 production logger/metrics backend；Q13 运行态能力不能在本次验证。
- 消费者代码只用于接缝和可实施性核对，不构成完整代码 Review。
- 共享工作树已有大量改动；本报告依赖上述固定摘要，不对无关改动给结论。
- checkpoint 的 `accepted:proceed` 只说明可作为 test-plan/R 对账基线，不等于独立 Design Review PASS。

## 9. 结论与复审条件

**结论：NOT PASS。**

F001 与 F002 均为 MEDIUM：前者使同一指标存在互斥实现合同，后者留下同步观测永久阻塞导致请求和资源无法收敛的可用性风险。F003 为 LOW，但会误导项目检查 Gate。三项均属于 design owner 可在既有 F005 范围内修正的事实/合同问题，不要求 Backend-Dev 先行实现或承担设计决策。

交回：**identity-auth Feature-Designer / 当前 design owner**。修订后应由同源独立 Reviewer 逐项复审 F001～F003，并检查 test-plan 对 F002 新合同的对应更新。只有 findings 关闭、固定新修订重新 checkpoint 且摘要可核后，才建议 r17 后继修订通过 Design Review Gate。之后仍需独立编码授权；本报告不构成该授权。

## 10. 实际命令与落盘验证

实际使用的命令类别：

- `git branch --show-current`
- `git rev-parse HEAD`
- `Get-FileHash -Algorithm SHA256 ...`
- `Get-Content -Raw/-Encoding UTF8 ...`
- `Select-String ...`
- `git status --short -- <scoped paths>`
- PowerShell 按 `.add/tool/add.mjs` 的 `artifactSections`、confirmation block 和 SHA-256 规则独立复算 `objectRef` / `documentRecordRef`
- `git diff --no-index --check -- NUL .spec/wip/identity-auth/review-design-r17.md`

报告落盘后需再次核对 design/checkpoint 摘要、报告唯一新增语义和本报告 SHA256。本文件不内嵌自身 SHA256，以避免摘要自指；写后摘要记录在本次交付回执中。
