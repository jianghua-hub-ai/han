# identity-auth test-plan r08 production composition 原 finding 复审

- **复审日期**：2026-10-23
- **Reviewer**：Helmholtz，原非作者独立 Reviewer
- **对象类型**：design（测试规划内容对象）
- **结论**：**PASS**
- **原 findings**：`TP-R07-F001` CLOSED；`TP-R07-F002` CLOSED
- **新增 findings**：CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0
- **授权边界**：本报告只复审固定 test-plan r08 及相关回归面；不修改 test-plan、design、tasks、Plan、代码或 ADR，不授权实现、测试执行、任务状态变更、Git、部署或 E2E。

## 1. 固定对象与上游验证

| 项目 | 固定值 / 核对结果 |
|---|---|
| 被审对象 | `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` |
| revision | `identity-auth-test-plan-ir01-r08` |
| 要求 SHA256 | `C95518C5AC7C5DAEE56F54D0E6691E999BFB8CF006E0D71956ED8457F97AD23D` |
| 开始 SHA256 | `C95518C5AC7C5DAEE56F54D0E6691E999BFB8CF006E0D71956ED8457F97AD23D` |
| 结束/报告落盘前 SHA256 | `C95518C5AC7C5DAEE56F54D0E6691E999BFB8CF006E0D71956ED8457F97AD23D`，无漂移 |
| source design | `identity-auth-design-ir01-r26` |
| checkpoint object digest | `sha256:42d8d10c37acb55a752c198caef32819d4d73744acd71af6fb1a4fba0366bc3d` |
| checkpoint | `decision-a76cb64cb07a0e37c80b3d22b15034e817e68d7c00a9be1759372e0396e6a56f`；`accepted / proceed` |
| design review | SHA256 `171E373555357493003EBC99F5704281304E86282EA8A709B14AAB4E4A82DE28`；PASS / 0 findings |
| 原 test-plan review | `.spec/wip/identity-auth/review-test-plan-r07-production-composition-r04.md` |
| 原 review SHA256 | `2804B8D6DF62827654165DDF8E5F7D76F52B7C231F3502FAF3369C5CE1FCC673` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |

r08 metadata、正文和修订历史一致指向 accepted r26 精确对象，并保持 `planned-not-executed`。共享工作树存在他人既有改动；本轮未回滚、覆盖、清理或格式化任何现状，唯一写入为本报告。

## 2. 两轮实际检查

### Round 1：原 findings 定点复审

直接对照原报告的触发场景、最小修复建议与复审条件检查：

1. `test-plan.md:187` 的 I-F005-26；
2. `test-plan.md:149` 的 I-AUTH-08；
3. `test-plan.md:25,58,153` 的 r06/r07/r08 历史和 I-F005-13～22 未变说明；
4. metadata、source、checkpoint、digest 与执行状态。

结果：

- I-F005-26 已把 adapter 调用和 credential Verify/接受分开，并逐类固定调用数；
- I-AUTH-08 已更新为 I-F005-01～29；
- 历史 `01～22` 和 I-F005-13～22 未变说明仍保留在明确的历史/不变语境；
- 未发现修复夹带执行状态、范围或设计合同变化。

### Round 2：accepted design、消费者与 tasks/Plan 回归

有界反查：

- accepted design r26 的 recovery 状态矩阵与测试合同；
- r26 checkpoint 与独立设计复审；
- I-F005-13～29 定义、覆盖、环境、任务建议和十八文件映射；
- tasks SSOT 与批次 Plan 的现行十四文件、S3.2/B4 状态；
- `cmd/han-server/main.go` 的 production composition 现状；
- 十八个 design r26 文件的存在性。

反查结果：

1. I-F005-26 与 `design.md:1355,1361,1365-1372,2643` 一致。
2. I-F005-13～29 各有且仅有一条场景定义。
3. tasks SHA256 仍为 `E19C35F99648044B932881C48B680A1E2DF48878E6DA9A914DA66E780D746235`；Plan SHA256 仍为 `A977A4FF101183B2A66BBC8CD36D3045569B3DBDFBEE35166FBD7CC56E7248B9`，与原 r07 评审时一致。
4. tasks/Plan 仍使用 r22/r06 十四文件现役范围：S3.2 active、F005 OPEN、B4 未关闭。r08 只把 r26 十八文件和新增四文件列为待 Task-Planner 回填输入，没有冒充已登记、已执行或已关闭。
5. `cmd/han-server/main.go:460-465` 仍固定返回 `identity business route provider is not configured`；r08 如实保持 production `composeBusiness` fixed-fail。
6. r26 十八个文件均存在；规划保持第十九文件、公共 API、schema/migration、依赖和 ADR 的停止边界。

## 3. 原 Finding disposition

### TP-R07-F001 — MEDIUM — CLOSED

**原问题**

I-F005-26 将 RecoveryMode adapter 调用与 recovery credential Verify/接受合并，未准确保留 accepted r26 的调用矩阵。

**修复证据**

`test-plan.md:187` 当前明确：

1. `IsInitialAdmin`、`RecoveryRequired`、`CredentialModeRecoveryBootstrap` 任一不成立，包括非 `recovery_bootstrap`：adapter 0 次；
2. 前三项全部成立：`normal`、`maintenance`、`error/unknown` 均调用 adapter 恰好 1 次；
3. `normal` 和 `error/unknown` fail closed，recovery credential Verify 0 次；
4. 只有 `maintenance` 才进入 Verify，且仅 Verify 成功后接受凭据；
5. 普通流量维护 503 继续由独立 RecoveryGate/readiness 拥有。

该表达与 design r26 的六行状态矩阵和 §9 测试合同一致。场景动作还要求记录 RecoveryMode、Verify、audit 与 session 调用数，因此可形成逐行可执行断言。

**相关回归**

- recovery 四项合取未被改成全局维护开关；
- 非 recovery-bootstrap 路径仍按自身状态机处理；
- PG18 只作为独立回归环境，真实恢复 readiness/E2E 继续分离；
- 未改变 schema、migration、公共接口或十八文件边界。

### TP-R07-F002 — LOW — CLOSED

**原问题**

I-AUTH-08 的 production composition 追溯仍停留在 I-F005-01～22，遗漏 r07 新增的 I-F005-23～29。

**修复证据**

`test-plan.md:149` 已更新为：

> production composition 由 I-F005-01～29 细化

同时：

- Story 覆盖表继续引用 I-F005-01～29；
- §6 分别映射 I-F005-23～24、25/29、26～28；
- `test-plan.md:25,153` 仍把 I-F005-01～22 用于 r06 历史和 I-F005-13～22 合同未变说明，没有抹除历史。

## 4. 新增 Findings

无。

本次未发现因两处定点修复导致新的正确性、安全、追溯、状态或范围缺陷。

## 5. 回归范围结论

1. **planned-not-executed**：metadata、摘要、场景、命令和状态表保持计划未执行。
2. **r26 来源**：revision、checkpoint、object digest和 design review SHA均正确。
3. **十八文件上限**：仍为五个 identity、两个 security、六个平台、一个 http config、四个 composition 文件；新增四文件说法准确。
4. **fixed-fail**：production `composeBusiness` 仍未实现，未被文档 PASS 冒充为代码完成。
5. **PG18**：只保留既有事务矩阵历史证据，当前环境/复跑仍 blocked。
6. **benchmark/profile**：目标环境报告、内存/延迟预算、profile ID 与采用记录仍未提供，合法 token 不等于 readiness。
7. **deploy/E2E**：日志留存、真实 scrape、Grafana、Alertmanager、receiver、升级链、runbook 演练和用户旅程 E2E 均未执行。
8. **tasks/Plan**：r08 未修改任务三态、文件合同或批次状态；r26 R 对账仍待 Task-Planner 执行。

## 6. 待决定与非阻断建议

### 待决定

无新增决定。后续是否以及何时同步 r26/r08 到 tasks/Plan、授权 S3.2 实施和安排目标环境/部署验证，仍由对应责任方另行决定。

### 非阻断建议

无新增建议。原 r07 报告关于 trusted proxy 明列边界实例和 abuse 动态 Retry-After 多实例的建议仍可在测试实现时采用，但不影响本次 finding 关闭。

## 7. 未验证风险与限制

- 按授权未执行 Go test/race/vet/build、PostgreSQL 18、benchmark/profile、部署、scrape、告警或 E2E。
- 本次 PASS 只证明固定 r08 测试规划准确关闭原 findings，不证明 factory、Gate、Header、recovery、source 或 router 已实现。
- production `composeBusiness` 仍 fixed-fail；I-F005-13～29 没有 r26 执行结果。
- tasks/Plan 尚未消费 r26/r08；后续同步仍需独立核对十八文件、任务状态和授权边界。
- 共享工作树含大量既有改动；结论只绑定本报告列出的固定 SHA。

## 8. Gate 结论

**TP-R07-F001：CLOSED。**

**TP-R07-F002：CLOSED。**

**identity-auth test-plan r08 Test-Plan Gate：PASS。**

该 PASS 仅适用于 SHA256 `C95518C5AC7C5DAEE56F54D0E6691E999BFB8CF006E0D71956ED8457F97AD23D` 的固定测试规划对象。它不表示实现完成，不表示任何 Go/PG18/benchmark/profile/部署/E2E 已执行或通过，不关闭 S3.2/F005/B4，也不构成任务状态更新、Git 或后续实施授权。

## 9. 实际命令与落盘验证

实际只读操作包括：

- `Get-FileHash -Algorithm SHA256`
- UTF-8 全文/定点读取与模式计数
- `git branch --show-current`
- `git rev-parse HEAD`
- `git status --short --untracked-files=all`
- 十八文件存在性与必要消费者反查

报告落盘后执行对象 SHA 重核、scoped status 和 `git diff --no-index --check -- NUL <report>`。本文件不内嵌自身 SHA256，最终报告摘要记录在交付回执。
