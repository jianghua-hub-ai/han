# identity-auth test-plan r03 独立评审报告

- 评审角色：非作者独立 Reviewer
- 评审日期：2026-10-22
- 对象类型：design（测试规划内容对象）
- 固定对象：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
- 逻辑修订：`identity-auth-test-plan-ir01-r03`
- 结论：**NOT PASS**
- Findings：0 CRITICAL / 0 HIGH / 0 MEDIUM / 1 LOW
- 授权边界：本报告只评审 test-plan，不复审或确认 design r18，不授权 F005 编码、测试执行、B4 关闭、Git、发布或 E2E。

## 1. 固定对象与基线验证

### 1.1 被评对象

| 项目 | 验证结果 |
|---|---|
| test-plan 普通文件 SHA256 | `22B9DBAE4D496BB173EB35C6BB9E8A961298DCC7E2B65C591C5750547177FD08`，与委托固定值一致 |
| metadata revision | `identity-auth-test-plan-ir01-r03` |
| 状态 | `planned-not-executed` |
| source design | `identity-auth-design-ir01-r18` / `draft` |
| design checkpoint | `pending` |
| historical checkpoint | `decision-0bc3a0de5aa6e86fae6d7fa874c26c0fb29437762310c5e925b62e1493fabca3`，仅指向历史 r17 |

### 1.2 上游与前序证据

| 对象 | 验证结果 |
|---|---|
| design r18 普通文件 SHA256 | `2465263BB08BBD9B5D7EE5123A730966836FFD295192B4C51CBFE6DC7E89465D` |
| design r18 object digest | 按 `.add/tool/add.mjs` 的确认记录排除规则独立复算为 `sha256:384f797b71c39df7dffcdadc2f08406e3c70a1b131b55654f7ac2b70a54baacd`，与 r03 metadata 匹配；摘要输入 262783 bytes |
| r17 design review | SHA256 `08181FFAF3A3E7E95BD0FF3594F225F58688D4C422EB3653EED44FC68FECDB0F` |
| r02 test-plan review | SHA256 `5197DE52C393B61FA57D1BB13054CE2EC3D26206009AB7778A1A918CEA9BDD59` |
| Manifest | SHA256 `E8F5E9C1CDB0548BE00F20A187DEA22347D40C8FF2E5E8D61E563699DA253F80` |
| tasks | SHA256 `E88ED85959EE491A2BAFE4DCE21449487593B91102507AA6E1288B7A5681A7F8` |
| B4 Plan | SHA256 `1A1BF3AE072AF143050E27FA9AF4083520EC1B1473F622A6456AD2E5BB27F3C3` |

开始评审及本报告落盘前，报告路径均不存在；分支为 `main`，HEAD 为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

r03 正确区分了：

- r18 是尚未独立 Review、用户确认或 checkpoint 的固定 draft；
- r17 checkpoint 只适用于历史 r17；
- r02 独立 PASS 只适用于旧 SHA256 `0215043EECF2504756D84A5C2959BCCC2CE0183A3E96CA2A8B2605004EAA7560`，不自动继承到 r03。

## 2. 实际读取与依据

复用此前已完整读取且内容仍有效的：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/task-execution-workflow.md`

本轮完整读取：

- 固定 test-plan r03 全文；
- design r18 的 metadata、observer/context/budget/resource/composition、Origin、验证和 Gate 相关合同；
- `.spec/wip/identity-auth/review-design-r17.md`；
- `.spec/wip/identity-auth/review-test-plan-r02.md`；
- `.codex/agents/add-test-engineer.toml`；
- `.codex/skills/add-test-design/SKILL.md`。

有界核对 requirements、identity boundary、S03 stories、ADR-0016 索引与追加式审计合同、Manifest、`docs/testing.md`、tasks 的 S3.2、Plan 的 B4，以及 `internal/identity`、`internal/platform`、`internal/audit`、`cmd/han-server` 下现有测试消费者。未读取实施 Skill，未加载无关 profile。

## 3. 两轮实际检查

### Round 1：test-plan 直接缺陷与内部一致性

逐段检查了：

1. r03 metadata、source revision/object digest、draft/checkpoint 和旧评审适用边界；
2. 风险 R-I13 与最低足够测试分层；
3. I-F005-08 的请求取消、请求更早 deadline、composition 预算更早到期、错误返回、无效预算和 adapter 取消保证；
4. I-F005-09 的唯一 403 写后 observer、四类 reason、`response_write`、零 identity Handler/attempts、fan-out、失败隔离和资源回收；
5. I-F005-01～12 的定义唯一性、事务 unknown、失败审计、响应写后、五类 outcome、敏感字段和 composition；
6. 数据、环境、清理、资源生命周期、命令、任务建议和执行前 Gate；
7. 文档内部的任务登记状态是否前后一致。

结果：

- `I-F005-01`～`I-F005-12` 各有且仅有一行场景定义；其他出现均为映射或引用，不是重复定义。
- observer 新合同与 r02 既有 F005 主体覆盖均完整。
- 发现 1 项 LOW：§1.2 成功证据仍保留 R 对账前的“未登记/待 Task-Planner R 对账”状态，与同文档及 tasks/Plan 当前事实冲突。

### Round 2：上游合同、任务/环境与消费者回归

1. 将 I-F005-08/09/11 对照 design r18 的 context cancellation/deadline、正数有限预算、取更早 deadline、fan-out、一次返回、资源 owner 和 route registration fail-closed 合同。
2. 对照 r17 design review 的三项 finding：
   - Origin counter 已统一为写后采样及 `reason,response_write`；
   - observer 永久阻塞改为 cancel/deadline 驱动的有界测试；
   - Manifest 七项检查已同步。
3. 对照 r02 review，确认 commit/rollback unknown、14 字段与 canonical metadata、一次性 guard、单次响应、五类 outcome、敏感 denylist、非 DB fallback 和 production composition 未退化。
4. 对照 tasks/Plan：
   - 唯一 active task 仍为 `identity-auth-S3.2`；
   - F005 复用原 task，不新增 F005 task；
   - B4 未关闭；
   - F005 编码/测试执行、B5/B9C/B9D、S4.5/S4.6、Git、发布和 E2E 未授权。
5. 对照 Manifest 与 `docs/testing.md`：
   - 七项通用项目检查与 r03 一致；
   - PostgreSQL 18 integration 是额外专用命令；
   - `HAN_TEST_POSTGRES_DSN`/major 18 未确认时保持 `blocked-for-execution`。
6. 对照现有测试消费者确认规划的包路径存在；未以尚未实现的 fake、composition 或运行结果冒充现有证据。

Round 2 没有发现 observer 场景遗漏或分层退化，但确认 Round 1 的任务状态冲突是现行 tasks/Plan 可证实的事实错误。

## 4. Findings

### TP-R03-F001 — LOW — 成功证据仍使用 R 对账前的任务登记状态

**位置**

- 冲突项：`test-plan.md:54`
- 同文档当前状态：`test-plan.md:260`、`:310`、`:328-332`、`:368`
- tasks/Plan 事实：`tasks.md:13`、`:299-359`；Plan B4 `:435`、`:443-477`

**问题**

`test-plan.md:54` 将规划完成证据写为：

> F005 增量建议明确标为“未登记/待 Task-Planner R 对账”

但当前 tasks/Plan 已完成 S03/F005 R 对账，结论是复用已登记的 `identity-auth-S3.2`、不新增 F005 task、未授权执行。r03 自身后续章节也使用“已复用原 task / 未授权执行”。

**触发场景**

Lead、Task-Planner 或后继 Reviewer依据 §1.2 判断 r03 是否满足任务映射 Gate。

**影响**

- 同一 test-plan 对 F005 给出“待 R 对账”和“已完成 R 对账复用”两种互斥状态。
- 可能导致重复发起 R 对账，或误判当前 test-plan 尚未映射任务。
- 不影响 I-F005-01～12 的技术覆盖，也没有扩大编码授权，因此定为 LOW。

**修复建议**

将 §1.2 第 6 项同步为当前事实，例如：

- F005 增量继续引用 tasks SSOT；
- 已复用 `identity-auth-S3.2`，不新增 F005 task；
- 当前仅完成映射，编码和测试执行仍未授权；
- 任务三态只由 tasks SSOT 维护。

不得借修复更新 task 状态、关闭 B4 或把 r18 draft 写成已确认。

**复审验证**

1. §1.2、§6、§9、§11 对 F005 均使用“已复用原 task / 未授权执行”。
2. “未登记/待 R 对账”只可出现在明确标注的历史 r02 时点，不得描述当前状态。
3. tasks/Plan 不因本次文档修复而被修改。

## 5. 其余重点范围结论

除 TP-R03-F001 外，本次未发现其他分级 finding：

1. **I-F005-08**覆盖 outcome observer、logger 和 metrics sink 阻塞至 cancel/deadline；覆盖请求先取消、请求更早 deadline、composition 预算更早到期和运行错误。
2. 明确 observation context 取请求 deadline 与正数有限预算的更早者；缺预算、非正/非有限预算或 adapter 无取消保证时 route registration fail closed。
3. 明确 Handler 有界返回、observer 一次调用、未调用 adapter 不再启动、无重试/备用 sink/二次响应/事务重放/Audit Append 或 `RecordOnce` 重复。
4. **I-F005-09**覆盖四类 Origin reason、writer ok/failed、唯一 403 写后 observer、`reason,response_write` 标签、identity Handler 0、login attempts 0、Guard 有界返回和资源回收。
5. deterministic 组件/Handler 与 composition 启动测试承担主要组合证据；PostgreSQL 18 只补真实事务状态；真实 backend/E2E 没有被用作全部组合的唯一证据。
6. I-F005-01～12 各唯一一行，r02 的事务 unknown、失败审计、写后 finalizer、五 outcome、敏感扫描和非 DB fallback 均保留。
7. 资源测试明确禁止人工无界等待、永久遗留 goroutine 和用 sleep/进程退出冒充释放；不可观察资源要求先提供可测试替身。
8. Manifest 七项通用检查与 PG18 专用 integration 阻塞已分开。
9. Q13 backend、budget 具体数值、SDK、配置载体、阈值、dashboard、paging/runbook 和值守入口均未被编造。
10. r03 正确保留 r18 draft 的独立 Review、用户确认、新 checkpoint 和另行编码授权 Gate。

## 6. 待决定

本 test-plan 评审没有新增产品、架构或实现方向决定。

以下仍是既有上游/运行待决定或受阻项：

- design r18 是否通过独立 Review、是否由用户确认并形成新 checkpoint；
- observer budget 的具体数值和生产输入来源；
- Q13 logger/metrics backend、阈值、dashboard、paging/runbook、外部 sink 健康和值守入口；
- PostgreSQL 18 DSN/major 环境；
- production composition 的实际 recorder、预算输入和 wiring。

本评审不替用户确认 design r18，也不使用 test-plan 自行修订设计合同。

## 7. 非阻断建议

1. §1.2 的任务映射成功证据可只引用 tasks SSOT，避免后续 R 对账状态再次漂移。
2. 后续测试实现应以可控 context、channel/屏障和可观察资源 owner 证明收敛，不使用固定 sleep 作为通过证据。
3. r18 设计复审可把 r03 的 I-F005-08/09/11、§7 资源生命周期和 §11 Gate 作为下游可测性证据，但不能反向替代设计合同正确性判断。

## 8. 限制与剩余风险

- 本轮未执行任何 Go、PostgreSQL 18、contract、race、vet、build 或 E2E。
- F005 observer、adapter、deterministic fake、资源计数和 production composition 尚未实现；只能评审计划的完整性与可执行边界。
- 未连接真实 logger/metrics backend，无法验证生产 sink 的真实取消行为。
- design r18 尚未独立 Review、确认或 checkpoint；若后续设计合同变化，r03 受影响场景必须重新核对。
- 共享工作树含大量既有改动；本评审只绑定上述固定哈希，不评价无关成果。

## 9. Gate 结论与复审条件

**结论：NOT PASS。**

r03 对 design r18 新增的 observer 有界阻塞、取消、deadline、有限预算和资源释放合同覆盖充分，r02 的 F005 技术覆盖也未退化；但 TP-R03-F001 使当前任务映射状态在同一 test-plan 内自相矛盾，不满足“准确保持任务边界”的完成条件。

交回 test-plan 原作者/Test-Engineer，只需修正该状态同步项，不需要修改 design、tasks、Plan 或扩大测试范围。固定新对象后由同源独立 Reviewer 复审 TP-R03-F001 及相关任务/授权回归面。修复前不建议 r03 Test-Plan Gate PASS。

即使后续 r03 复审 PASS，仍不代表 design r18 通过、不授权 F005 编码或测试执行，也不关闭 B4。

## 10. 实际命令与落盘验证

实际使用的命令类别：

- `git branch --show-current`
- `git rev-parse HEAD`
- `Get-FileHash -Algorithm SHA256 ...`
- `Get-Content -Raw/-Encoding UTF8 ...`
- `Select-String ...`
- `Get-ChildItem ... *test.go`
- `git status --short -- <scoped paths>`
- PowerShell 按 `.add/tool/add.mjs` 的确认记录排除规则复算 design r18 object digest
- 锚定场景定义行的正则计数，确认 `I-F005-01`～`12` 各唯一一行
- `git diff --no-index --check -- NUL .spec/wip/identity-auth/review-test-plan-r03.md`

报告落盘后需再次核对 test-plan/design 哈希、报告仅新增语义和本报告 SHA256。本文件不内嵌自身 SHA256，以避免摘要自指；写后摘要记录在本次交付回执中。
