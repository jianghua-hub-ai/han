# identity-auth test-plan r02 独立评审报告

## 1. 结论

- **对象类型**：design（测试规划文档）
- **固定对象**：`.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
- **修订**：`identity-auth-test-plan-ir01-r02`
- **固定 SHA256**：`0215043EECF2504756D84A5C2959BCCC2CE0183A3E96CA2A8B2605004EAA7560`
- **基线**：Git 中 `identity-auth-test-plan-ir01-r01`
- **审查差异**：76 insertions / 19 deletions；`git diff --check` 通过
- **最终结论**：**PASS**

在固定对象范围内未发现 CRITICAL、HIGH、MEDIUM 或 LOW 缺陷。r02 准确覆盖 r17 的 F005 十项合同，并给出可执行的单元、组件、契约、PostgreSQL 18、race、vet、build 分层、数据隔离/清理、任务未登记状态和 planned-not-executed 边界。

该 PASS 仅表示测试规划文档可进入用户接纳与 Task-Planner R 对账，不表示 F005 已实施、测试已运行、PostgreSQL 18 环境已验证、`identity-auth-S3.2` 可关闭或可进入 B5/E2E。

## 2. 适用依据

已实际读取并按 design 对象适用范围执行：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/profiles/go-service/README.md`
- `.codex/profiles/go-service/rules/security-coding-standards.md`
- go-service 的 coding、no-hardcode、observability、tracing 规则

评审依据还包括 r17、checkpoint、requirements、stories、ADR-0016、tasks/Plan、Manifest、测试约定及真实代码/测试消费者。未把接力摘要当作仓内事实证据。

## 3. 两轮实际检查范围

### Round 1：完整 diff 与直接缺陷

1. 核对固定 SHA256、修订号、planned-not-executed 状态、r01 基线及完整 76/19 diff。
2. 逐项检查 `I-F005-01`～`I-F005-12`：
   - ID/time/guard 在 source evidence 前生成，Handler/Service 共享 guard 且合计至多一次；
   - commit unknown 的实际已提交/未提交；
   - rollback unknown 的 Append 前/后；
   - 14 个不可变字段与 canonical metadata 完整比较、零盲写；
   - 唯一 HTTP 响应写入、写后 observer、`response_write` 与禁止二次响应；
   - observer/logger/metrics 阻塞或失败不改变 HTTP、Session、Audit；
   - OriginGuard 四类拒绝、403 先写、WARN/counter 各一次、零 login attempts；
   - 五类 attempts outcome、invalid JSON、无 `trace_id`/敏感或高基数指标标签；
   - 生产缺依赖 fail closed、deterministic fake；
   - DB/Audit 不可用时非 DB fallback，普通日志不冒充 AuditEvent。
3. 检查敏感 denylist、每例数据/清理、故障注入、屏障释放与 goroutine 回收责任。
4. 检查“未登记/待 R 对账”“本轮未运行”“不得借用既有 F001 证据”等交付边界。

### Round 2：上游契约、真实消费者与任务 SSOT

1. 对照 `identity-auth-design-ir01-r17` 的 F005 合同及 checkpoint
   `decision-0bc3a0de5aa6e86fae6d7fa874c26c0fb29437762310c5e925b62e1493fabca3`。
2. 对照：
   - `stories.md` 的 S03；
   - requirements 的 `US-IDENTITY-02.1` 登录验收；
   - ADR-0016 的 PostgreSQL 追加式审计约束；
   - tasks/Plan 中 `identity-auth-S3.2`、F005 OPEN、B4 未关闭及 R 对账 Gate。
3. 核对 `add.manifest.yaml`、`docs/testing.md`、`go.mod`：
   - Go `1.27.0`；
   - `gofmt/mod verify/vet/build/contract/race/test` 项目质量门；
   - `integration` 标签、`HAN_TEST_POSTGRES_DSN`、PostgreSQL major 18；
   - 缺必需环境变量应明确失败，不得 Skip/静默通过。
4. 核对真实包与测试入口：
   - `internal/identity`
   - `internal/audit`
   - `internal/platform`
   - `cmd/han-server`
   - 现有登录、OriginGuard、审计及 integration 测试
5. 核对 r02 建议命令与当前包布局一致；精确测试名和文件所有权仍正确留给 Task-Planner R 对账。

## 4. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

无。

### LOW

无。

## 5. 合同覆盖证据

| 合同 | test-plan 位置 | 结论 |
|---|---|---|
| source 前 ID/time/guard；Handler/Service 至多一次 | `test-plan.md:128` | 覆盖 |
| commit unknown 实际已提交/未提交 | `test-plan.md:129-130` | 覆盖 |
| rollback unknown Append 前/后 | `test-plan.md:131-132` | 覆盖 |
| 完整不可变 AuditEvent、canonical metadata、零盲写 | `test-plan.md:133` | 覆盖 |
| 唯一响应、写后 observer、response_write | `test-plan.md:134` | 覆盖 |
| observer 阻塞/失败隔离 | `test-plan.md:135` | 覆盖 |
| OriginGuard 四拒绝、403 先写、观测一次、零 attempts | `test-plan.md:136` | 覆盖 |
| 五类 attempts、invalid JSON、标签安全 | `test-plan.md:137` | 覆盖 |
| 生产装配 fail closed、deterministic fake | `test-plan.md:138` | 覆盖 |
| DB/Audit 不可用非 DB fallback | `test-plan.md:139` | 覆盖 |
| 敏感数据自动扫描 | `test-plan.md:141` | 覆盖 |
| 分层、任务未登记、命令与 Gate | `test-plan.md:250, 265-284, 313-342, 344-369` | 覆盖 |

## 6. 待决定与证据缺口

### D1：r17 独立复审状态

- **test-plan 表述**：`test-plan.md:21` 将 r17 独立设计复审列为后续 Gate。
- **接力信息**：称 Kant 已对 r17 PASS（0 HIGH / 0 MEDIUM）。
- **仓内证据**：
  - `.spec/wip/identity-auth/` 仅存在 `review-design-r01.md`～`review-design-r09.md`；
  - 未找到固定 r17 对象、两轮范围和 PASS 结论的 r17 review 报告；
  - `design.md:37` 仍为“评审：待独立 Reviewer”；
  - `design.md:2548` 明确 r17 仍需独立复审；
  - tasks/Plan 仍只记录 Kant 对 r14 的 NOT PASS；
  - r17 checkpoint 仅接受其作为 test-plan 输入，并明确不等于独立复审。
- **判断**：这是**接力声明缺少仓内可复核证据**，不是 test-plan 的事实错误。当前 `test-plan.md:21` 与仓内 SSOT 一致，应保留。
- **最小处置**：若要改为“r17 已独立复审”，先补齐固定对象/哈希、Reviewer、两轮实际范围、findings 与 PASS 结论的可复核证据，再由原 owner 同步 design、tasks/Plan 和 test-plan Gate。

## 7. 非阻断建议

1. Task-Planner R 对账时基于实际落地测试函数补充 `-run` 精确表达式，但保留 test-plan 已要求的包级与项目级回归。
2. r17 独立复审证据未落仓前，不使用接力摘要提前解除设计 Review Gate。

## 8. 命令与证据

实际执行的只读检查包括：

- `Get-FileHash -Algorithm SHA256 test-plan.md`
- `git status --short --branch`
- `git diff --stat -- test-plan.md`
- `git diff --check -- test-plan.md`
- `git show HEAD:test-plan.md`
- `git diff -- test-plan.md`
- 对 design/checkpoint/stories/ADR/requirements/tasks/Plan/Manifest/testing/go.mod 的 `Get-Content`、`Select-String`
- 对 `internal/identity`、`internal/audit`、`internal/platform`、`cmd/han-server` 的包、测试、build tag、环境变量和 PostgreSQL major 检查

关键证据：

- 固定对象哈希与委托值一致；
- 报告创建前目标路径不存在；
- r01→r02 为 76 insertions / 19 deletions；
- `git diff --check` 返回 0；
- `source_design_artifact` 与 r17 checkpoint `objectRef` 均为
  `sha256:41bb2c83f63fb26c8ea8da7e4b01a8ca3cc9a53f918d25b7a4c9893610a0bf87`；
- `docs/testing.md` 与 `internal/testenv` 要求缺少 `HAN_TEST_POSTGRES_DSN` 时明确失败；
- identity/audit integration 测试已有随机 schema、DROP 清理及 PostgreSQL major 18 检查入口。

## 9. 未验证风险与限制

1. 本轮未运行任何 Go、race、vet、build、contract 或 PostgreSQL 测试，不得把计划命令称为通过。
2. 未连接 PostgreSQL 18，未验证 commit unknown fault adapter、真实提交状态或 schema 清理的运行结果。
3. F005 代码、deterministic fake、production composition 和故障适配器尚未实现；当前只能评审规划充分性和可执行边界。
4. 当前 `design.md` 原始文件 SHA256 为
   `2E329F300617312A39597FEE95751DE0456A9DA72DFAA872B5E251F0EA6191C4`，
   与 checkpoint 的 pre-record `objectRef` 及 `documentRecordRef` 均不同；test-plan 引用与 checkpoint
   `objectRef` 一致，且本轮核到的 r17 行为合同一致，但未能从仓内单独重建该 pre-record 字节对象。此项作为追溯限制记录，不据此否定 test-plan。
5. 工作树存在大量其他在途修改；本评审未修改、回滚、格式化或清理被审对象及其他文件。

## 10. 复审条件

当前无 finding，不要求 test-plan 整改复审。以下情况发生时应重新评审受影响范围：

1. 固定 test-plan 对象或其 SHA256 变化；
2. r17 的 F005 合同、ADR-0016、登录公共 HTTP 合同或生产装配边界变化；
3. Task-Planner R 对账删减 PostgreSQL 18、race/contract/项目级回归或清理责任；
4. 后续声称 r17 已独立复审，但提供的固定证据与当前 Gate/SSOT 不一致。

