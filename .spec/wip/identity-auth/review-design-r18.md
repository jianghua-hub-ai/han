# identity-auth design r18 同源独立复审

## 0. 结论摘要

- 评审角色：r17 原非作者独立 Reviewer
- 固定对象：`identity-auth-design-ir01-r18`
- 复审对象：`review-design-r17.md` 的 F001～F003 及直接回归面
- 结论：**PASS**
- Design Review Gate：**建议通过**
- 原 finding 状态：
  - F001 / MEDIUM：**CLOSED**
  - F002 / MEDIUM：**CLOSED**
  - F003 / LOW：**CLOSED**
- 新 findings：**无**
- 授权边界：本报告只证明固定 r18 draft 通过本次独立设计复审；**不等于用户确认或 checkpoint，不授权 F005 编码、测试执行、任务状态更新、B4 关闭、Git、发布或 E2E**。

## 1. 固定对象与版本验证

### 1.1 固定 design

| 项 | 验证值 |
|---|---|
| 路径 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| revision | `identity-auth-design-ir01-r18` |
| status | `draft` |
| ordinary SHA256 | `2465263BB08BBD9B5D7EE5123A730966836FFD295192B4C51CBFE6DC7E89465D` |
| object digest | `sha256:384f797b71c39df7dffcdadc2f08406e3c70a1b131b55654f7ac2b70a54baacd` |
| object digest 输入长度 | 262783 bytes |

ordinary SHA256 与指定值一致。object digest 按 `.add/tool/add.mjs` 的 `artifactSections` / `artifactObjectDigest` 规则复算：定位唯一 `## 确认记录`，同时排除其前的 `<!-- ADD:CHECKPOINTS -->` 标记及确认记录区，再对剩余 UTF-8 bytes 计算 SHA256；未用普通文件 SHA 冒充对象摘要。

### 1.2 原失败链与固定下游

| 对象 | SHA256 | 结果 |
|---|---|---|
| `.spec/wip/identity-auth/review-design-r17.md` | `08181FFAF3A3E7E95BD0FF3594F225F58688D4C422EB3653EED44FC68FECDB0F` | 匹配；保留 r17 NOT PASS 与 F001～F003 失败链 |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `7B82FA42C0FA65EAE6FBFBB1764946B251421BFC338D0468490DF492B7E893AF` | 匹配；revision r03，状态 `planned-not-executed` |
| `.spec/wip/identity-auth/review-test-plan-r03.md` | `EB3695135F3F73BB785C28B2F4B87A1C23468D654A333AAC53BA7041693CBEF2` | 匹配；原 TP-R03-F001 失败链 |
| `.spec/wip/identity-auth/review-test-plan-r03-r02.md` | `34F67FBA014401601CF8843EDBB32B9A439EF362815C6B4E6F17365FB1D98AA7` | 匹配；TP-R03-F001 CLOSED，Test-Plan Gate PASS |

历史 checkpoint `decision-0bc3a0de5aa6e86fae6d7fa874c26c0fb29437762310c5e925b62e1493fabca3` 仍明确绑定 r17 object digest `sha256:41bb2c83f63fb26c8ea8da7e4b01a8ca3cc9a53f918d25b7a4c9893610a0bf87`，只接受 r17 作为测试规划/R 对账输入且不授权 F005 实施。r18 没有伪用该 checkpoint，正文明确保持 draft、旧 checkpoint 不适用于新正文。

### 1.3 工作区

- branch：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 报告落盘前目标路径不存在。
- 共享工作树已有大量既有修改和未跟踪成果；本次未清理、回滚、覆盖或格式化它们。

## 2. 两轮实际检查范围

### Round 1：原 finding 逐项合同复审

1. 对照 r17 原报告的触发条件、影响与复审条件，检查 r18 修订记录、组件责任、接口、composition、故障管理、验证矩阵、实施边界和附录映射。
2. 全文核对 `han_platform_origin_rejections_total` 的采样顺序、标签 allowlist 和敏感/高基数禁令。
3. 核对两类 observer 及 logger/metrics adapter 的 context、deadline、有限预算、失败收敛、资源 owner 和 production fail-closed 合同。
4. 核对 Manifest 七项项目检查及 PostgreSQL 18 专用 integration 的独立阻塞表达。

### Round 2：直接回归面与下游可测性

1. 有界回归 F005 核心方案：公共 REST/错误/认证合同、事务与审计 unknown 矩阵、写后 finalizer、敏感信息、Q13、production composition 和授权边界。
2. 核对 r03 的 I-F005-08、I-F005-09、I-F005-11、资源生命周期和执行前 Gate，确认设计新增合同具有最低足够层的可执行测试计划。
3. 有界检查现役消费者接缝：登录 Handler 仍忽略 response write error；Audit ID 仍在 Service 的 `appendLoginAudit` 阶段生成；OriginGuard 尚无 observer；production main 仍使用空 `ServeMux`。这些现状与 r18 的“目标合同尚未交付”陈述一致，未被设计误写为已实现。
4. 核 tasks/Plan 当前仍绑定已 checkpoint 的 r17/r02，并保留 F005 编码、PG18、production composition、代码 Review 与 B4 Gate。r18 尚未确认/checkpoint 前不应把 draft 预写成已采用任务来源。

## 3. 原 findings 逐项结论

### F001 — MEDIUM — CLOSED

**原问题**

r17 对 Origin counter 同时出现“拒绝判定时采样”和“唯一 403 写入尝试返回后采样”，并存在互斥标签合同。

**修复证据**

- `design.md:673-676`：platform-owned observer 只在唯一 403 写入尝试返回后调用；counter 只接受 `reason,response_write`。
- `design.md:830-836`：observer 输入只含固定 reason 与低基数 `response_write=ok/failed`，写后同步调用一次。
- `design.md:2163`、`:2206-2208`、`:2369`、`:2562`、`:2607`：故障、验证、回归和追踪表均采用同一写后采样合同。
- 全文出现的 `han_platform_origin_rejections_total` 未再给出写前采样规则；标签 allowlist 统一为 `reason,response_write`。
- 原始 Origin/Header/IP、trace、账号、身份及 digest 等敏感或高基数字段均被禁止作为标签。

**触发回归**

对 request nil、Origin 缺失/不匹配、Fetch Metadata 拒绝和 writer ok/failed，规范顺序均是：完成唯一 403 写入尝试、取得 `response_write`，再调用 observer；observer 失败不得撤销写入或二次响应。

**结论**

原冲突已消除，接口、composition、风险与验证表述一致。F001 **CLOSED**。

### F002 — MEDIUM — CLOSED

**原问题**

r17 将 observer 定义为同步调用，却没有对 sink 永久阻塞时的 cancellation/deadline、有限上界、资源释放和请求收敛作出可实施合同。

**修复证据**

- `design.md:673-676`：两类 observer 与 metrics adapter 均遵从传入 context，不得另起遗留 goroutine/队列。
- `design.md:782-791`：
  - Handler 从 request context 派生 observation context；
  - composition 注入正数有限预算；
  - 请求已有更早 cancellation/deadline 时取更早者；
  - caller 在调用结束后释放 timer/cancel；
  - logger/metrics adapter 使用同一 context；
  - cancel/timeout 后一次失败返回，不重试、不切备用 sink、不启动尚未调用的 adapter；
  - 禁止改用 `context.Background()`/`TODO()` 逃避取消；
  - caller 与 adapter 分别释放所拥有的 timer、连接、响应体和临时缓冲，不遗留 goroutine 或后台写。
- `design.md:820-826`、`:1194`：缺 observer/logger/metrics、缺正数有限预算，或 adapter 无法保证遵从 context 时，受影响路由注册必须 fail closed；已装配后的运行期 sink 失败/超时保持 fail open，不改变既有登录结果。
- `design.md:832-836`：Origin Guard 使用同样的有限预算和资源释放规则。
- `design.md:2163-2165`、`:2352`、`:2367-2370`、`:2454`、`:2502`、`:2563`、`:2608`：故障管理、验证、实施边界、待决定 owner 和回归映射一致要求有界返回、单次调用、无二次响应、无事务/Session/Audit 重放及资源回收。
- 具体预算值保持 `TBD（待容量评估阶段定）`；设计没有编造配置载体、backend、SDK、阈值、dashboard、paging、runbook 或值守入口。

**失败收敛**

sink 阻塞到 cancel/deadline 时，observer 只返回一次失败；不得重试、切换备用 sink、写第二响应，或重复主事务、Session、Audit Append/`RecordOnce`。无有效有限预算或取消保证时，错误在 route registration 阶段确定性暴露。

**结论**

原缺失的有限阻塞、取消/deadline、资源 owner 与 fail-closed 合同已经成为 provider-neutral、可实施且可测试的强约束。F002 **CLOSED**。

### F003 — LOW — CLOSED

**原问题**

r17 错称 Manifest 尚未登记项目检查命令。

**修复证据**

- `design.md:2395` 准确列出 Manifest 七项通用检查：
  - `gofmt -l .`
  - `go mod verify`
  - `go vet ./...`
  - `go build ./...`
  - `go test -tags=contract ./...`
  - `go test -race ./...`
  - `go test ./...`
- `design.md:2452`、`:2564`、`:2609` 均声明七项已登记、本次设计复审未执行。
- PostgreSQL 18 专用 integration 命令、DSN 和 major 环境保持独立 `blocked-for-execution`，未与七项通用检查混写，也未声称通过。
- 全文未再发现“项目尚未登记检查命令”“项目检查命令仍未登记”或“无法定义完整 CI Gate”等过期表述。

**结论**

当前事实与 Manifest 一致，通用检查和 PG18 专用阻塞仍是两个独立 Gate。F003 **CLOSED**。

## 4. 相关回归结果

未发现 r18 为关闭 F001～F003 而使下列合同退化：

1. 公共 REST status/code/message、统一登录失败、Cookie/Session/CSRF、枚举防护和暴力防护规则未改变。
2. Handler-owned ID/time/guard 前置、Handler/Service 互斥审计 owner、每请求最多一次失败审计调用保持不变。
3. commit unknown 仅按原 `audit_event_id` 完整只读核对；rollback unknown 继续区分 Audit Append 前/后；禁止盲写、UPDATE/DELETE 或把未知结果伪装为 `no`。
4. 最终 observation 仍位于唯一响应写入尝试之后；observer 失败不覆盖原 cause、不撤销 Session/Audit、不二次响应。
5. Origin 403 仍不进入 identity Handler 或 `han_identity_login_attempts_total`。
6. production 缺必要依赖、正数有限预算或 adapter 取消保证时 route registration fail closed；运行期观测失败不改变业务状态。
7. Q13 的 backend、阈值、dashboard、paging/runbook/值守入口及具体预算值保持 TBD；无新依赖或配置被擅自决定。
8. PostgreSQL 18 环境未确认，integration 仍是 `blocked-for-execution`。
9. r18 仍为 draft，修订记录准确说明只关闭 r17 F001～F003，不宣称已获用户确认或实施授权。

## 5. test-plan r03 下游可测性

r03 对 F002 的下游计划充分，且其独立 Test-Plan Gate 已 PASS：

- `test-plan.md:145` / I-F005-08：
  - observer/logger/metrics sink 阻塞到 context cancel/deadline；
  - 请求更早 deadline 与 composition 有限预算取更早者；
  - Handler 有界返回、单次调用；
  - 不重试、不切备用 sink、不二次响应、不重放事务或重复审计；
  - goroutine、timer、连接和临时资源回到基线；
  - 缺有效预算或取消保证时 route registration fail closed。
- `test-plan.md:146` / I-F005-09：
  - 唯一 403 写后 observer；
  - counter 标签仅 `reason,response_write`；
  - identity Handler 与 login attempts 为 0；
  - Guard 有界返回，无重试/备用 sink/二次响应，资源回收；
  - 缺依赖、预算或取消保证时装配 fail closed。
- `test-plan.md:148` / I-F005-11：覆盖 observer/logger/metrics/预算缺失、非正/非有限预算和 adapter 无取消保证的启动失败。
- `test-plan.md:292` 规定资源生命周期基线和有界回收，不允许用无界 sleep、人工永久释放或进程退出冒充收敛。
- §11 执行前 Gate 保留 design r18 确认/checkpoint、F005 授权、PG18 环境和 production 输入限制。
- I-F005-01～12 各有且仅有一行场景定义；r02 的事务 unknown、响应写后、Origin、五类 outcome 与敏感字段覆盖未退化。

该 PASS 只证明固定 r03 的测试规划可测、可执行；测试仍是 `planned-not-executed`，不能替代本设计复审、实现证据或真实环境结果。

## 6. 新 findings、待决定与非阻断建议

### 6.1 新 findings

**无。**

本次实际检查了原三项 finding、直接回归合同、现役消费者接缝和 r03 下游计划，未发现新的 HIGH/MEDIUM/LOW 缺陷。

### 6.2 待决定

没有新增需要架构/公共契约裁决的事项。以下既有输入仍按设计保留，不是本次复审阻断项：

1. 两类 observer 的具体正数有限预算：由 production composition owner 基于获批准容量评估，在对应路由注册前提供。
2. Q13 的 logger/metrics backend、阈值、dashboard、paging/runbook 和值守入口：保持 TBD。
3. PostgreSQL 18 DSN/major 环境：未确认，专用 integration 不得执行或宣称通过。

### 6.3 非阻断建议

1. r18 经用户确认并创建新 checkpoint 后，由真实 Task-Planner 将 S3.2/B4 的设计与 test-plan 来源从历史 r17/r02 同步到已接受的 r18/r03；在 checkpoint 前不要预先改写 SSOT。
2. 实施时让 deterministic 阻塞 fake 只因被测 context 取消而返回，并对 timer/连接/goroutine owner 使用可计数替身，避免 sleep 型“看似有界”测试。

## 7. 限制与剩余风险

- 本次是设计复审，没有执行 Go、PostgreSQL 18、race、vet、build、contract 或 E2E。
- 没有真实 logger/metrics backend、容量预算或生产配置，故未验证具体运行时延迟、连接池行为和告警链。
- 现役代码仍未实现 r18 的 observer、有限预算、资源释放与 production wiring；本报告不构成代码 Review 或完成证明。
- 消费者检查只覆盖 F001～F003 的直接接缝，没有重做无关全仓代码评审。
- 共享工作树含大量既有变更；结论只绑定本报告列出的固定 design/test-plan/review 摘要。
- 本报告自身 SHA256 只能在落盘后外部计算，最终交付返回提供该值；文件不能稳定自包含自身哈希。

## 8. Gate 结论与后续条件

**F001～F003 全部 CLOSED；固定 r18 的 Design Review Gate 建议 PASS。**

该 PASS 表示 r18 对原失败项的修复正确、相关 F005 合同未退化，且 r03 已提供对应可测性计划。后续仍必须依次满足：

1. 用户/真实设计接收责任人确认 r18；
2. 为 r18 创建新的正式 design checkpoint，并核其 object digest；
3. Task-Planner 在新 checkpoint 后同步 S3.2/B4 的 r18/r03 来源与 Gate；
4. 取得独立 F005 编码授权；
5. 实施、项目检查、可用 PostgreSQL 18 专用 integration、非作者代码 Review 与 F005 复审；
6. 满足任务关闭条件后，才可关闭 S3.2/B4。

旧 r17 checkpoint、r03 Test-Plan PASS 或本报告均不能单独授权编码或关闭任务。

## 9. 实际命令

本次使用的只读/验证命令类别：

```text
git branch --show-current
git rev-parse HEAD
git status --short --branch
Get-FileHash -Algorithm SHA256 <fixed objects>
Get-Content -Encoding UTF8 <bounded sections>
Select-String -Path <design/test-plan/reviews/Manifest/tasks/Plan/checkpoint> -Pattern <focused contracts>
rg -n <consumer seams> <bounded consumer files>
```

object digest 使用 PowerShell 按 `.add/tool/add.mjs:37228-37265` 的确认记录排除规则复算。报告落盘后另执行：

```text
git diff --no-index --check -- NUL .spec/wip/identity-auth/review-design-r18.md
Get-FileHash -Algorithm SHA256 .spec/wip/identity-auth/review-design-r18.md
```
