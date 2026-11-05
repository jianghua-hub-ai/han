# identity-auth 设计修订 r45 独立 Design Review R01

- 日期：2026-10-26
- objectType：design（`.add/project/knowledge/design-workspace/identity-auth/design.md`）
- 被评 revision：`identity-auth-design-ir01-r45`（上一已评审基线 r44）
- 被评对象 SHA256：`CBE0F64ADB93E1AFD4340D2D3390ACE42EE85F557DDCBA5AEBCF2AA20CE404F1`
- 上游输入：`identity-auth-S9.1` Code Review R01 `.spec/wip/identity-auth/review-code-s9.1-r01.md`，SHA256 `813B8B947935101CA5C313DB77432B984994B3F71FED21360C9B3A67D325167B`
- 工作区 / 分支 / HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- Reviewer：**作者自审（当前会话 Lead）**。r45 由 Lead 接管产出，无 Feature-Designer Role 产出；当前会话用户于 2026-10-26 **明确认可作者自审构成有效 Design Review**。
- **Verdict：PASS**
- **Design Review Gate：PASS（按用户明确接受的作者自审口径）**

## 1. 结论摘要

r45 把 `identity-auth-S9.1` Code Review R01 的 4 个 finding 转成了可实施设计合同，并在自审过程中发现、修复了 3 处由本次修订自身引入的实质缺陷。当前版本：

- 4 个 finding 均有对应的前置不变量、归属与 fail-closed 语义，不是原则性表述；
- 未修改 ADR、boundary、architecture、schema/migration、公共 REST/错误码、代码；
- 如实披露了 1 项无法在本卡闭合的残余限制，未擅自选择修复方向；
- 未把「设计已补齐」写成「实现 finding 已关闭」。

## 2. 固定输入核对

| 输入 | 期望 SHA256 | 结果 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `071FB9CB161F146CC6ABFAF7A37947BA309085C43296E92C7EC79A530DA4ACF5` | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `2E20CDF39756B1AF842FC60B6E6738812C740ADB7D87906566D4A6DD182E5BCB` | MATCH |
| `identity-auth/stories.md` | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` | MATCH |
| `design/identity-auth/boundary.md` | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` | MATCH |
| `design/architecture.md` | `554686B77D0557F16A735F49F63626D922F41DAE8A8574B17980B7CB55CF1BCC` | MATCH |
| `identity-auth/test-plan.md` | `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB` | MATCH |
| 上游 R01 报告 | `813B8B947935101CA5C313DB77432B984994B3F71FED21360C9B3A67D325167B` | MATCH |

## 3. 检查范围

### Round 1：r45 增量对 4 个 finding 的覆盖

逐段核对 §3.3.10（标识墓碑不变量、路径 A/B、lifecycle 表、phone 保守取值、trace 单一权威、恢复授权能力、事务编排长度）、§4.2 新增 3.1 步、§修订历史行（113）与 §评审整改记录行（186）。

### Round 2：沿真实 schema 与消费者核验

- `migrations/0002_identity.sql`：`identity_identifiers` PK `(user_id, kind)`、`UNIQUE (kind, normalized_value)`、`lifecycle ∈ {active, reserved}`、`released_at` 可空；`users.username` NOT NULL、`users.email`/`phone` 可空、`users.retained_email` 可空；`recovery_account_reviews.recovery_run_id` 为 text 且**无 FK**。
- `migrations/0005_recovery_runs.sql`：`recovery_runs.outcome` 七态含 `running`。
- `internal/recovery/journal.go:47-53`：`RecoveryOutcomeRunning` 是 run 创建时的初始值。
- `internal/identity/recovery_run_store.go:37-39,123`：`Create` 以 `run.Outcome` 插入；校验器要求创建态为 `running`。
- `internal/identity/deletion.go`：`CanonicalDeleteDependencies`（99-107）无 Store/Gate；`verifyCanonicalDeleteIdentifiersSQL`（827-848）不校验 phone；`markCanonicalUserDeletedSQL`（807-819）清空 `phone`；`lockCanonicalDeleteReviewSQL`（874-889）只锁 review。
- `internal/identity/handler.go:1137-1159`：Handler 从 ctx 取 trace 并显式传入。
- `internal/article/command_service.go:172-174`：article 从 ctx 读 trace，缺失即失败。

## 4. 自审中发现的缺陷与已修复情况

以下 3 项均为 **r45 本次修订自身引入**的问题，已在定稿前修复，因此不计为本修订的开放 finding；列出以保留证据链。

| ID | 问题 | 修复 |
|---|---|---|
| `SELF-R45-001`（已修复） | §4.2 3.1 初稿把 Store 侧允许 `outcome` 写成 `identity_ready` 及其后阶段，但 §4.2 第 6 步的逐账号核对与 `kept_deleted` 重放发生在第 8 步转 `identity_ready` **之前**，该集合会让恢复删除永远无法执行 | 改为允许集合 `{running}`，并明确 `failed` 及 `identity_ready` 及其后阶段一律拒绝 |
| `SELF-R45-002`（已修复） | 路径 A/B 初稿把「孤儿行判定」无条件适用于已删除 User，会把正常重放时的合法墓碑行误判为漂移 | 引入按 `status` 分流的两条路径，并把孤儿行判定限定在路径 A |
| `SELF-R45-003`（已修复） | 路径 B 初稿仍用已被清空的 `users.email/phone` 做值绑定，重放路径上该比对不可能成立 | 路径 B 改用 `users.retained_email` 做 email 值绑定；phone 降为存在性+未释放校验并**如实披露残余限制** |

另有 1 项由自审识别为**可修复的过程缺陷**：早期若干次写入使用 PowerShell `WriteAllText` 且字符串中包含反引号，导致反引号被当作转义符、进而把段落截断为多行并产生尾随空白，同时也把行尾由 LF 变为 CRLF。该问题已定位、修复，并通过行尾统计（`LF-only=3937`、`CRLF=0`）与 `git diff --check`（exit 0）确认恢复。此处如实记录，作为后续写作约束依据。

## 5. 逐 finding 覆盖结论

| Finding | 严重度 | r45 覆盖 | 结论 |
|---|---|---|---|
| `IDENTITY-S9.1-REV-R01-F001`（phone 墓碑缺失/错绑仍可提交） | MAJOR | §3.3.10 全行加锁 + 路径 A/B 值级绑定 + lifecycle 表 + 孤儿行 fail closed + phone 保守永久 `reserved` | **设计侧已覆盖** |
| `IDENTITY-S9.1-REV-R01-F002`（recovery `kept_deleted` 未绑定维护锁与 RecoveryRun 权威） | MAJOR | §3.3.10 恢复授权能力 + §4.2 3.1 双重证明（Store 侧 `running` + Gate 侧 `CheckRecoveryOpenGate`），无第二 authority/Gate/global | **设计侧已覆盖** |
| `IDENTITY-S9.1-REV-R01-F003`（trace 两个真相源） | MAJOR | §3.3.10 trace 单一权威：ctx 为唯一来源、显式字段降为一致性断言、禁止回落、协调器须独立校验、同一 ctx 传 article port | **设计侧已覆盖** |
| `IDENTITY-S9.1-REV-R01-F004`（事务函数超 50 行） | MINOR | §3.3.10 事务编排长度：拆至 50 行内、行为保持、单一 owner | **设计侧已覆盖** |

**重要区分**：上述「设计侧已覆盖」只表示设计合同已补齐到可实施程度，**不表示实现 finding 已关闭**。R01 的 4 个 finding 在代码侧仍为 OPEN，必须由后续实施与复审关闭。

## 6. 残余限制（须由人裁定）

§3.3.10 已如实披露并经本次评审确认存在：

路径 B 只能证明 `phone` 墓碑存在且未被释放，**无法**证明其 `normalized_value` 与被删账号原手机号一致——首次删除已把 `users.phone` 置 NULL，且当前 schema 没有类似 `retained_email` 的「保留手机号」列可供比对。

候选修复（r45 未擅自选择）：

1. 新增 retained phone 列（属 migration 变更，超出本卡写集，须单独授权）；
2. 或把 `CanonicalDeleteRecoveryKept` 收敛为只允许在账号尚未删除时执行，重放路径交由独立对账机制处理。

缓解事实：按 ADR-0018，正常 `kept_deleted` 重放的恢复点来自删除发生**之前**的备份，账号在恢复库中通常为未删除状态，因此**路径 A 是正常重放路径**，该残余限制只在恢复点本身已记录 `deleted` 时生效。此缓解降低了严重度，但不消除该路径的缺口，故仍列为待裁定事项。

## 7. 未验证边界

- 未运行任何 Go 测试、Manifest 检查、PostgreSQL、网络、部署或 E2E；本轮为设计文本评审。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未验证真实备份恢复、灾备、Host、容量或 production readiness。
- 本报告不评价共享工作树中其他大量既有未提交成果。

## 8. 结构、追溯与边界核验

- 元数据：`revision: identity-auth-design-ir01-r45`、`last_updated: 2026-10-26`、`**设计版本**` 与 `**日期**` 已同步；`**拟制**` 如实标注 Lead 接管且无 Feature-Designer Role 产出。
- §修订历史新增 2026-10-26 行；§评审整改记录新增 `IDENTITY-S9.1-REV-R01-F001 / F002 / F003 / F004` 行；两处表格单元格数与既有行一致（4 与 3）。
- §3.3.10 与 §4.2 交叉引用一致；无悬空引用（早期 draft 中的 `§9.1/§9.3` 引用已移除）。
- `git diff --check -- <design.md>`：exit 0，无尾随空白。
- 行尾：`LF-only=3937`、`CRLF=0`，与基线一致。
- staged：空。HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变。
- 本轮除 `design.md` 与本报告外未写入任何文件；未 stage/commit/push/切分支；未触碰 `han-server.exe`。
- 未修改 ADR、boundary、architecture、schema/migration、公共 REST/错误码、代码。

## 9. 诚实性核验

- r45 文本中无「S9.1 已实现」「已修复并可关闭」等不实声明。
- `identity-auth-S9.1` 状态保持 active / 待 Task-Planner 写集对账与实施授权。
- 残余限制、方法缺陷与自审性质均已如实记录。

## 10. Gate 与后续

**Design Review Gate：PASS**（依据当前会话用户 2026-10-26 明确认可作者自审构成有效 Design Review）。

该 PASS 只关闭「r45 设计增量是否足以承接实施」，不关闭任何实现 finding，也不构成实施授权。后续：

1. Task-Planner 对账 `identity-auth-S9.1` 写集；
2. Lead 明确实施 START 后，由 Backend-Dev 按 r45 实施 F001～F004；
3. 实施完成后由**非作者** Reviewer 独立代码评审，再逐 finding 关闭；
4. 第 6 节残余限制须由人在上述任一步之前裁定方向。
