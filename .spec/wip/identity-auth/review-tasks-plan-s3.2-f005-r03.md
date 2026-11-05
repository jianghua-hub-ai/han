# identity-auth-S3.2 / F005 tasks 与 B4 Plan 独立规划 Review r03

## 0. 结论摘要

- **日期**：2026-10-22
- **角色**：非作者独立 Reviewer Hegel；未委派子 Agent。
- **对象类型**：design（tasks SSOT 与批次 Plan）。
- **结论**：**PASS（有 1 项 LOW 非阻断 finding）**。
- **B4 规划 Gate**：**建议通过**。本次没有未关闭 HIGH 或 MEDIUM finding。
- **新 finding**：
  - `PLAN-R03-F001` / LOW / OPEN：固定对象的 S3.2/B4 当前段已正确同步 Manifest 七项检查，但 tasks 其他卡片与 Plan 的 B3 历史 Context 仍残留当前时态的“Manifest 未登记/未配置项目检查命令”陈述。
- **授权边界**：本报告只证明固定 tasks/Plan 对 S3.2/F005 当前规划可承接 r18/r03 与新 checkpoint；不授权 F005 编码、测试执行、任务状态更新、B4 关闭、Git、发布或 E2E。

## 1. 固定对象与工作区

### 1.1 Review objects

| 对象 | 固定 SHA256 | 开始核对 | 写入前核对 |
|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `C4ADF7B6A783CE62D7D2A4621DC643106F7325FCE3073A4E351C5CDD76EBC60D` | 匹配 | 匹配 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `40E0665EEA545B43D20223A522A2C2E7B64E013019E1823F85C6C81BAA1FD39B` | 匹配 | 匹配 |

报告路径 `.spec/wip/identity-auth/review-tasks-plan-s3.2-f005-r03.md` 在写入前不存在。

### 1.2 工作区

- workspace：`D:\AID\VSCode\Codex\Han`
- branch：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定 tasks/Plan 在共享工作树中已有既存 modified 状态；本 Reviewer 未修改、回滚、格式化或清理它们。

## 2. 当前上游证据验证

### 2.1 design r18 与独立 Review

- 当前 design 整文件 SHA256：`364D66552DAB5F97F5E3590888DEEE29902F19D9DF44FE0899323E02AF27B298`。
- 该普通文件摘要包含新增 checkpoint 确认记录，不能冒充设计对象摘要。
- 按 `.add/tool/add.mjs` 的 `artifactSections` / `artifactObjectDigest` 规则排除 `<!-- ADD:CHECKPOINTS -->` 与 `## 确认记录` 区后复算：
  - object bytes：`262783`
  - object digest：`sha256:384f797b71c39df7dffcdadc2f08406e3c70a1b131b55654f7ac2b70a54baacd`
  - 与 tasks、Plan、checkpoint 一致。
- `.spec/wip/identity-auth/review-design-r18.md`
  - SHA256：`4DD87C7B26F7E6B9A252BB6150B3DABAECB6A19656A1A68E6F86DE2C52BD7A9B`
  - 结论：PASS；原 F001～F003 CLOSED。

### 2.2 test-plan r03 与独立 Review

- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
  - revision：`identity-auth-test-plan-ir01-r03`
  - SHA256：`7B82FA42C0FA65EAE6FBFBB1764946B251421BFC338D0468490DF492B7E893AF`
  - 状态：`planned-not-executed`
- `.spec/wip/identity-auth/review-test-plan-r03-r02.md`
  - SHA256：`34F67FBA014401601CF8843EDBB32B9A439EF362815C6B4E6F17365FB1D98AA7`
  - 结论：PASS；`TP-R03-F001` CLOSED。
- I-F005-01～12 各有且仅有一行场景定义；未缺号、未重复定义。

### 2.3 新 checkpoint

- 路径：`.add/records/checkpoints/ir-01-content-community/design/decision-e139a121ce302e24c33066fe3ac8849426ad173cb07fecac75e937e6ddb04706.json`
- record SHA256：`D92D3DBB70129CBEBF3134BD15104685CE2A91A12AECB6CCAF4EE661AFA7AD59`
- `decisionState=accepted`，`continuation=proceed`。
- `objectRef.digest` 与复算的 r18 object digest 一致。
- design 内对应 confirmation block 摘要复算为 `sha256:35e47298557756be69f15a6fadc7d10e03b11c324e4a603738a5292a75b078e8`，与 `documentRecordRef.digest` 一致，document observation 可核为 observed。
- record 明确 supersede r17 checkpoint；被 supersede 文件 SHA256 为 `34469F3556E6ABBAADBA0851365F29042858D4009299FCFB7A87AA9A32A221B6`，与 record 引用一致。
- checkpoint 的 applicability 仍因仓内既有另一条未串联的历史 decision 分支而是 `unknown / decision-chain-conflict`；这不是对象缺失、对象漂移或确认记录缺失。依据本次固定输入与用户明确决定，r18/r03/new checkpoint 仍作为本次规划基线；本报告不处理历史 decision chain。

### 2.4 Manifest

`add.manifest.yaml` SHA256 为 `E8F5E9C1CDB0548BE00F20A187DEA22347D40C8FF2E5E8D61E563699DA253F80`，现役七项检查为：

1. `gofmt -l .`
2. `go mod verify`
3. `go vet ./...`
4. `go build ./...`
5. `go test -tags=contract ./...`
6. `go test -race ./...`
7. `go test ./...`

PostgreSQL 18 integration 是额外专用 Gate，不属于上述七项。

## 3. 实际读取范围

### 3.1 角色、方法与规则

完整读取或复用当前会话中已完整有效的：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/skills/add-delivery-workflow/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/task-execution-workflow.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`

### 3.2 固定对象与依据

- `tasks.md` 全文。
- B4 Plan 全文。
- checkpoint JSON、Manifest、test-plan r03、design r18 的 F005/确认记录相关合同。
- r18 design Review、r03 test-plan 同源复审、前序 planning r01/r02。
- 有界消费者：
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

消费者只用于验证 gap-map、文件接缝和可实施性，未做代码质量 Review。

## 4. Round 1：固定对象直接检查

### 4.1 任务唯一性与三态

结构解析结果：

- 49 total
- 12 completed
- 1 active
- 36 todo
- 唯一 active：`identity-auth-S3.2`
- 重复 task ID：0
- 独立 F005 task：0
- Plan 批次外层表对 49 个 task 各安排一次，无漏排或重复安排。
- Plan 未出现 task-like checkbox 行；其状态叙述均明确引用 tasks SSOT，没有形成第二套可独立维护的 checkbox 表。

S03 R 对账保持：

- 已建成 1
- 待做 0
- 需改 3
- 废弃 0
- Story 总体“需改”
- 复用 `identity-auth-S3.2`
- 不新增 F005 task

### 4.2 当前来源与历史时间线

tasks 顶部、S3.2 卡片、R 对账段及 Plan 顶部、精确输入、B4 Context Pack、Gate/交回段一致声明：

- 未受影响 identity Story 继续使用 r09/r01。
- S3.2/F005 当前使用 design r18、stories S03、test-plan r03 与新 checkpoint。
- r17/r02 只作为历史时间线存在，不再充当当前 S3.2/F005 Gate。
- design Review PASS、test-plan Review PASS 与 checkpoint 只允许规划同步，不构成实施或测试执行授权。

### 4.3 文件合同

十个允许修改/新增文件一致：

1. `internal/identity/handler.go`
2. `internal/identity/service.go`
3. `internal/identity/login.go`
4. `internal/identity/login_test.go`
5. `internal/identity/login_integration_test.go`
6. `internal/platform/http_security.go`
7. `internal/platform/http_security_test.go`
8. `internal/audit/store_integration_test.go`
9. `cmd/han-server/main.go`
10. `cmd/han-server/main_test.go`

只读：

- `internal/platform/logging.go`
- `internal/audit/store.go`

禁止：

- `internal/identity/repository_integration_test.go`

tasks 与 Plan 均要求只读/禁止边界或依赖、公共合同、未知写者发生变化时停止；未把规划登记误写成当前代码写权。

### 4.4 串行顺序与 observer 合同

两份对象一致登记：

1. identity 合同与 deterministic fake
2. platform Origin observer
3. PostgreSQL 18 实际事务核对
4. production composition
5. 非作者代码 Review / F005 复审

observer 相关合同未遗漏：

- outcome observer、Origin observer 及 logger/metrics adapter 遵从传入 context cancellation/deadline；
- request context 已有更早 deadline 时取更早者；
- production composition 必须显式注入正数有限预算；
- 缺失、非正、非有限预算，或 adapter 无法证明可取消并有界返回时，受影响路由注册 fail closed；
- 阻塞 sink 必须因 cancel/deadline 有界返回；
- cancel/timeout 后不重试、不切备用 sink、不启动尚未调用的 adapter；
- 不二次响应，不重放主事务、Session、Audit Append 或 `RecordOnce`；
- goroutine、timer、连接和临时资源回到基线；
- 禁止用 sleep、人工永久释放或进程退出冒充收敛。

### 4.5 环境、TBD 与授权边界

- PostgreSQL 18 要求 `HAN_TEST_POSTGRES_DSN` 且先核 major 18；当前为 `blocked-for-execution`，不得 Skip 或冒充通过。
- production logger/metrics/wiring 未交付，production composition 保持 blocked。
- Q13 backend、具体预算值、阈值、dashboard、paging、runbook、值守入口保持 TBD。
- Task-Planner/实现者不得选择 SDK/backend、编造配置、引入依赖或硬编码默认预算。
- F005 编码、测试执行、B5、B9C、B9D、S4.5、S4.6、Git、发布和 E2E 均未授权。
- B4 仍未关闭；S3.2 关闭后仍需 B4 独立批次整体 Review。

## 5. Round 2：上游、下游与真实消费者反向检查

### 5.1 r18/r03 可追踪性

- design r18 已固定写后 observer、context cancellation/deadline、正数有限预算、请求更早 deadline、单次失败收敛和资源 owner。
- test-plan r03 的 I-F005-08、09、11 明确覆盖两类阻塞 observer、composition 启动 fail-closed 与资源生命周期。
- I-F005-01～12 在 tasks 的 identity/platform/PG18/composition 阶段均有承接；没有把真实 backend/E2E 作为 observer 全组合的唯一证据。
- Manifest 七项与 PG18 专用 integration 在当前 S3.2/B4 段分开表达。

### 5.2 真实消费者与 R 标签

有界核对确认规划没有把目标合同误写为已实现：

- `internal/identity/handler.go` 仍忽略登录 `WriteError/WriteJSON` 返回值，尚无写后 finalizer/outcome observer。
- audit event ID 仍在 `Service.appendLoginAudit` 中生成，晚于 Handler source evidence。
- commit error 当前直接返回，尚无实际提交状态只读核对。
- `OriginGuard` 当前只写 403，尚无 Origin rejection observer/WARN/counter。
- production `cmd/han-server/main.go` 仍使用空 `ServeMux`，尚无 identity 路由、logger/metrics/observer budget wiring。

这些事实支持“已建成 1 / 需改 3”的 gap-map，并与十写、二只读、一禁止文件合同相容；未发现必须新增 task 或改变公共设计的证据。

### 5.3 Gate 独立性

- test-plan PASS 没有被冒充为实现或测试已通过。
- design Review PASS 与 checkpoint 没有被冒充为编码授权。
- planning Review 没有替代 PostgreSQL 18、production composition、代码 Review、F005 复审或 B4 G4-batch Gate。

## 6. Findings

### LOW — PLAN-R03-F001：固定对象仍残留与现役 Manifest 冲突的当前时态陈述

**位置**

- `.add/project/knowledge/tasks/tasks.md`：共 34 处，包括 `:204,222,269,370,387,406,491,...,1099`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md:414`

**证据**

- tasks 的这些未受本次 S3.2 同步影响的卡片仍写“Manifest 未登记项目检查命令”。
- Plan 的 B3 统一检查段仍写“Manifest 当前未配置项目检查命令”。
- 现役 Manifest、tasks 顶部、S3.2 卡片及 Plan 顶部/B4 段均已确认七项项目检查已登记。

**触发场景**

后续实施者或恢复流程按具体 task 卡片、B3 Context Pack 或全文搜索恢复验证来源，而不是只读取文档顶部或 S3.2/B4 当前段。

**影响**

可能继续把已登记的七项检查误当临时命令，或在未来 task/批次中复制错误的配置事实。该问题不改变本次 S3.2/B4 的命令、文件合同、状态、Gate 或授权，因此定为 LOW，不阻断本次 B4 规划 Gate。

**建议**

由原 Task-Planner 另行做有界事实同步：

1. 将当前仍有效的 task 卡片改为 Manifest 已登记七项检查；
2. 历史时点若必须保留，明确标注“当时”及其时间，不使用当前时态；
3. 保持 task 数量、三态、requires、批次、S3.2 文件合同与授权边界不变。

**复审验证**

- 搜索上述两类过期短语应无当前时态残留；
- 七项检查与现役 Manifest 一致；
- 49 / 12 / 1 / 36、唯一 active S3.2、无新增 F005 task及 B4 未关闭事实不变。

## 7. 待决定与非阻断建议

### 7.1 待决定

本次没有新增架构、公共契约、文件范围或风险接受决定。

既有待定继续由真实 owner 处理：

1. observer 具体正数有限预算值；
2. Q13 backend、阈值、dashboard、paging/runbook 与值守入口；
3. PostgreSQL 18 DSN/major 环境；
4. production logger/metrics/wiring 的实际交付与代码授权。

### 7.2 非阻断建议

1. 后续状态变化继续只更新 tasks SSOT；Plan 可引用事实，但不要增加可独立维护的 task checkbox 表。
2. 历史 Context Pack 中易漂移的配置事实宜改为带日期的历史陈述或直接引用 Manifest。
3. 既有 `decision-chain-conflict` 宜由 checkpoint/知识 owner 在独立范围处理；不得借此回退已明确接受的 r18/r03 规划基线或扩大本次授权。

## 8. 限制与剩余风险

1. 本次是 planning Review，未执行 Go、PostgreSQL 18、contract、race、vet、build 或 E2E。
2. PostgreSQL 18 DSN/major 未确认，专用 integration 仍不能宣称可执行或通过。
3. production logger/metrics/backend、预算值与运行告警链尚未交付，未验证真实 sink 的取消、延迟、连接池或资源行为。
4. 现役代码仍未实现 r18 F005 目标；消费者读取只验证 gap-map 与文件接缝，不构成代码 Review。
5. checkpoint applicability 的 `decision-chain-conflict` 是既有记录链限制；本报告只验证当前 record、object digest 和 document record 均可核，不修复该链。
6. 固定对象之外的旧 task/历史批次没有重做完整项目规划；`PLAN-R03-F001` 只记录已实际观察到的配置事实漂移。
7. 共享工作树含大量既有未提交/未跟踪成果；本次只新增本报告。
8. 本报告不能稳定内嵌自身 SHA256；落盘后的摘要由交付回执提供。

## 9. Gate 结论与后续条件

**固定 tasks/Plan 对 identity-auth-S3.2 / F005 的当前同步结果没有未关闭 HIGH 或 MEDIUM finding，B4 规划 Gate 建议 PASS。**

`PLAN-R03-F001` 为 LOW、非阻断，但应由原 Task-Planner 后续做有界事实同步。

该 PASS 不关闭 B4。后续仍须：

1. 取得 F005 编码与测试执行授权；
2. 按 identity → platform → PostgreSQL 18 → production composition 串行实施；
3. 取得可用的 PostgreSQL 18 环境并执行专用 integration；
4. 解决 production composition 的实际输入与 Q13 适用 Gate；
5. 完成项目检查、非作者代码 Review 与 Ampere/获准替代 Reviewer 的 F005 复审；
6. Lead 核范围、哈希、命令与状态后关闭 S3.2；
7. 三张 B4 task 均关闭后完成 B4 独立批次整体 Review；
8. 另行取得后续批次、Git、发布或 E2E 授权。

## 10. 实际命令

本次使用的只读/验证命令类别：

```text
git branch --show-current
git rev-parse HEAD
git status --short -- <scoped paths>
Get-FileHash -Algorithm SHA256 <fixed objects and evidence>
Get-Content -Encoding UTF8 <files / bounded sections>
Select-String -Path <files> -Pattern <focused contracts>
PowerShell 结构解析：task 三态、重复 ID、唯一 active、批次安排、Plan task-like checkbox
PowerShell 按 .add/tool/add.mjs artifactSections 规则复算 design object digest
PowerShell 复算 checkpoint confirmation block / documentRecordRef digest
```

报告落盘后执行：

```text
git -c core.autocrlf=false diff --no-index --check -- NUL .spec/wip/identity-auth/review-tasks-plan-s3.2-f005-r03.md
Get-FileHash -Algorithm SHA256 .spec/wip/identity-auth/review-tasks-plan-s3.2-f005-r03.md
Get-FileHash -Algorithm SHA256 .add/project/knowledge/tasks/tasks.md,.add/project/knowledge/plans/ir-01-content-community-batches.md
git status --short -- .add/project/knowledge/tasks/tasks.md .add/project/knowledge/plans/ir-01-content-community-batches.md .spec/wip/identity-auth/review-tasks-plan-s3.2-f005-r03.md
```
