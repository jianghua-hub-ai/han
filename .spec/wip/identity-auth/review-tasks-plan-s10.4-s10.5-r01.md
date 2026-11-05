# identity-auth S10.4/S10.5 任务与批次规划独立评审 R01

- Reviewer：本轮独立 add-reviewer，非原 Task-Planner 作者
- 日期：2026-10-24
- objectType：design（任务 / 批次规划复合对象）
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / 基线 HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 报告范围：S10.4、S10.5、B6C-3、B6C-4、B6D 及其直接依赖、消费者、Gate 与所有权
- 最终结论：**Planning Review Gate: CHANGES REQUIRED**

## 1. 固定对象与完整性

| 被审对象 | 开始 SHA256 | 结果 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `2C18AA026B2136AF058D409F39B8202DF0529141D011C0B7EBA0A1706B9B9E9A` | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `A18B467D315DE2536036BDA4CF4CBCDEA5ED6B5B4EBB72E5FD54ADD15385AFFC` | MATCH |

开始时分支和 HEAD 与固定输入一致。工作树存在大量其他人员的已修改和未跟踪内容；本轮不回滚、不覆盖、不格式化这些内容，只按固定 SHA 读取两个对象，并且只新增本报告。

## 2. 两轮实际检查范围

### Round 1：固定对象直接检查

1. 全文检查 `tasks.md` 的 task 三态、ID 唯一性、requires、文件写集、TDD/验收命令、关闭条件、状态和历史记录。
2. 独立静态统计得到：
   - task 总数 57；
   - completed 24；
   - active 3；
   - todo 30；
   - active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`identity-auth-S10.1`；
   - task ID 无重复；
   - 238 条 requires 引用均指向已登记 task；
   - task requires 图无环。
3. 检查 S10.4：
   - `tasks.md:959` 保持 `[ ]`；
   - `tasks.md:964-972` 精确登记八文件；
   - `tasks.md:974-978` 覆盖五项必填无默认配置、pre-DSN 失败、唯一 `budget > MaxConns` 的 parse 后/connect 前失败、同一 Store/Gate、唯一 permit owner、双 worker lifecycle、无同步错误的两个 `Start(context.Context)`、`Done/Err` 运行错误通道及无 maintenance 入口；
   - `tasks.md:986-989` 的 requires、并行和关闭条件与当前状态一致。
4. 检查 S10.5：
   - `tasks.md:993` 保持 `[ ]`；
   - `tasks.md:997-1005` 明确待稳定设计、未认领、未启动、当前不可执行；
   - 设计稳定后仅顺序接管 `main.go/main_test.go`；
   - server/maintenance 模式互斥；
   - 当前无命令、runbook、Case/Run；
   - 完成前 dead-letter 持续 Gate/hold且不可运维恢复。
5. 检查批次映射：批次外层表的 57 个 task 均且仅映射一次；S10.2、S10.3、S10.1/S10.4、S10.5、S11.1 分别唯一位于 B6C-1、B6C-2、B6C-3、B6C-4、B6D。
6. 对照批次外层表、全套 DAG、详细依赖图、文件所有权表和修订记录，发现一处现役 DAG 冲突，见 F001。

### Round 2：上游依据、真实消费者与 Gate 回归

1. 对照 identity design r44、stories r16、test-plan r20、architecture r27、identity boundary r13 与 ADR-0023，检查 durable operation、User-first、三消费者 fail-closed Gate、dead-letter 不解锁、worker-before-pool、单一 composition root 和维护入口后置边界。
2. 对照 S10.4 Design Review R01/R02，确认原 maintenance 入口、permit 合同、配置失败时序与 Article/PasswordReset Start 语义 findings 已在 r44 中关闭；本规划未重新引入第二 Store/Gate、第二 root、隐藏 global 或 lifecycle 构造生产依赖。
3. 对照 S10.3 Code Review R02 与其真实 production consumer 交接，确认 S10.4 复用的是 S10.3 已构造和注入的同一 `*identity.PasswordResetStore`/Gate，而不是另建 authority。
4. 对照现役 test-plan r20，确认 S10.4 八文件的配置、permit、lifecycle 和 denylist 测试矩阵为 planned-not-executed；r18 Test-plan Review R02 仅是历史已评审来源，不外推为 r20 已执行或已独立测试评审。
5. 独立判断 B6D 不依赖 S10.5：
   - S11.1 的业务能力依赖 S10.1～S10.4 提供 durable operation、Gate、worker/operator 和 settlement；
   - S10.5 只补 production dead-letter 维护入口，不改变 S11.1 的 lineage、same-key/new-key 或普通请求 Gate 合同；
   - S10.5 未完成时 dead-letter 仍保持 Gate/hold，不能运维恢复，但这不会让 B6D 绕过 Gate或错误放行；
   - B6D 仍受 S10.1～S10.4 全部关闭、B6C-3 batch Gate PASS 和 `IDENTITY-S10.1-REV-R01-F002` CLOSED 约束；
   - S10.5 与 B6D 无文件交叉。

因此，**B6D 不新增对 S10.5 的硬依赖与上游契约、真实消费者、Gate/hold 和当前 E2E 分层一致**。该结论不表示 S10.5 可省略，也不表示 dead-letter 已具备生产运维恢复能力。

## 3. 实际读取文件

### 方法、角色、规则与 profile

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-route-work/SKILL.md`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/skills/add-task-decompose/SKILL.md`
- `.codex/skills/add-batch-plan/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/decisions-guide.md`
- `.codex/profiles/go-service/README.md`
- `.codex/profiles/go-service/rules/coding-standards.md`
- `.codex/profiles/go-service/rules/no-hardcode-policy.md`
- `.codex/profiles/go-service/rules/security-coding-standards.md`
- `.codex/profiles/go-service/rules/observability-standards.md`
- `.codex/profiles/go-service/rules/tracing-standards.md`

### 固定对象与上游依据

- `.add/project/knowledge/tasks/tasks.md`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`
- `.add/project/knowledge/design-workspace/identity-auth/design.md`，SHA256 `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23`
- `.add/project/knowledge/design-workspace/identity-auth/stories.md`，SHA256 `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3`
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md`，SHA256 `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB`
- `.add/project/knowledge/design/architecture.md`，SHA256 `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB`
- `.add/project/knowledge/design/identity-auth/boundary.md`，SHA256 `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3`
- `.add/project/knowledge/decisions/README.md`
- `.add/project/knowledge/decisions/identity.md`，SHA256 `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978`

### 已有评审证据

- `.spec/wip/identity-auth/review-design-r43-s10.4-r01.md`，SHA256 `DE750BE1CC2B2B493E4376512ADACF54255AC901EDCDD83829879FDA42E9D7D8`
- `.spec/wip/identity-auth/review-design-r44-s10.4-r02.md`，SHA256 `9A76D7E3C5BC29855043784057C99383844A0B3CC7F14E32F96EDBB5290F0434`
- `.spec/wip/identity-auth/review-test-plan-r18-s10.2-r02.md`，SHA256 `3A241B10EFB64EFA7A09E452F20F22CB84051AAB7D3503E511C5298451AF794C`
- `.spec/wip/identity-auth/review-code-s10.3-r02.md`，SHA256 `40D264C2E694E9CD5596DAD3FC1C574385F4CC37B08A22CA3D3127AB4769287F`
- `.spec/wip/identity-auth/review-tasks-plan-s10.1-r02.md`，SHA256 `210AAED75029CB76DF30DA30F0D4DB165B555BF97D6F52CAA4CED63E17DC0118`

## 4. Findings

### `IDENTITY-S10.4-S10.5-PLAN-R01-F001` — MEDIUM — OPEN

**位置**

- `.add/project/knowledge/plans/ir-01-content-community-batches.md:75`

**违反依据**

顶层批次 DAG 把路径写成：

```text
B6C-2 → B6C-3 → {B6C-4 ∥ B6D}
```

按批次语义，这要求 B6C-4 等待整个 B6C-3 完成。该约束与同一计划及任务 SSOT 的现役合同冲突：

- `plans/ir-01-content-community-batches.md:61`：B6C-4 requires S10.4，不属于 B6C-3 batch Gate；
- `plans/ir-01-content-community-batches.md:699`：`S10.4 → S10.5`；
- `plans/ir-01-content-community-batches.md:730`：只等待 S10.4 task-level CLOSED并释放两文件；
- `tasks/tasks.md:1002-1003`：S10.5 只 requires S10.4，且不属于 B6C-3 batch Gate；
- 现役 task DAG 为 `S10.2 → S10.3 → {S10.1, S10.4 → S10.5}`。

评审清单要求计划的依赖、所有权、Gate 与真实消费者一致；同一现役计划不能同时给出两个不同的启动条件。

**触发场景**

S10.4 已 task-level CLOSED并释放 `cmd/han-server/main.go` / `main_test.go`，但 S10.1 的 F002 或 B6C-3 integrated Review / batch Gate 尚未关闭。

**影响**

按顶层 DAG 编排的执行者会错误阻塞 B6C-4/S10.5，直到 S10.1 和整个 B6C-3 Gate 完成；这与已拍板的 S10.5 启动合同冲突，并会无必要地延长 dead-letter 不可运维恢复的时间。反向按 task 合同执行时，顶层批次图又会显示越过批次依赖，造成计划解释分叉。

**最小修复**

只修订顶层 DAG 的 B6C 段，显式表达两个不同触发器：

1. B6C-2 后，S10.1 与 S10.4 是并行 task lane；
2. `S10.4 CLOSED → B6C-4/S10.5`；
3. `S10.1 + S10.4 CLOSED + B6C-3 batch Gate PASS + F002 CLOSED → B6D/S11.1`。

不要给 S10.5 新增对 S10.1、B6C-3 batch Gate 或 B6D 的硬依赖，也不要改变已确认的 task DAG。

**证据**

- 批次外层表和任务详细合同均把 B6C-4 的唯一 task 前置固定为 S10.4；
- 详细依赖图只包含 `S10.4 → S10.5`；
- requires 静态解析显示 S10.5 只有 `identity-auth-S10.4`；
- 上游 r44/r20 把 S10.5 定义为 S10.4 后续顺序接管 main 两文件的独立 maintenance 候选；
- B6D 的完整 Gate 独立列出 S10.1～S10.4、B6C-3 Gate 与 F002，不包含 S10.5。

**复审条件**

1. 提供只修订该 DAG 冲突的固定新 revision/hash；
2. 重新核对批次外层表、顶层 DAG、详细依赖图、文件所有权表及 `tasks.md:987,1002-1005,1013,1024-1026`；
3. 静态重跑 57-task 唯一批次映射、requires 未知目标和环检查；
4. 确认 S10.5 仍为 `[ ]`、不可执行，且 B6D 仍不依赖 S10.5。

## 5. 待决定与非阻断建议

### 待决定

无。本 finding 的最小修复不需要改变 architecture、boundary、ADR-0023、task requires 或已确认的 S10.4/S10.5/B6D 契约。

### 非阻断建议

无。F001 已包含所需的最小澄清；不建议借机重写批次结构或扩大规划范围。

## 6. 旧边界与历史残留检查

1. 固定对象的现役 S10.4 合同只使用八文件，不存在仍可执行的旧六文件写集。
2. `plans/...:908-911` 保留了旧六文件历史演进，但后续记录明确说明旧合同已由 r44/r20 和本次八文件对账取代；不会作为现役执行入口。
3. 现役 S10.4 明确不交付 production maintenance command/runbook；S10.5 明确当前无命令、runbook、Case/Run且不可执行。
4. 上游 test-plan r20 中“Tasks/Plan 仍为旧六文件”的文字是该 revision 当时的下游待对账事实；当前固定 Tasks/Plan 已完成对账。执行应以本轮固定对象的现役段落为准，不应把 r20 的修订时点描述反向解释为当前计划状态。

## 7. 未验证风险与限制

1. 本轮是规划审查，没有运行 Go 测试、race、vet、build 或 Manifest 七项，不能证明 S10.4 实现正确或可关闭。
2. 未运行真实 PostgreSQL、SMTP/TLS、Host、部署、浏览器、性能、容量或 E2E；未验证真实 lease 竞争、连接预算、事务锁序、commit actual state 或 dead-letter 恢复。
3. S10.5 设计尚未稳定；`main.go/main_test.go` 两文件是否足够、授权模型、审计证据、并发状态变化和退出码仍须后续设计与测试规划确认。
4. 五项生产实际配置值仍为 TBD；本轮只确认“必填无默认”和失败时序合同，没有验证生产容量值。
5. S10.1 继续 implementation stopped，`IDENTITY-S10.1-REV-R01-F002` OPEN；本轮没有复审或关闭该代码 finding。
6. 本轮没有读取、输出或使用 `HAN_TEST_POSTGRES_DSN` 的值。
7. 工作树包含大量其他人员变更；结论仅绑定本报告列出的固定对象 SHA 和读取到的上游 revision，不覆盖未固定对象的后续变化。

## 8. 最终 Gate

- CRITICAL：0
- HIGH：0
- MEDIUM：1（OPEN）
- LOW：0
- 待决定：0

**Planning Review Gate: CHANGES REQUIRED**

S10.4/S10.5 的状态、八文件边界、配置/Store/Gate/lifecycle 合同、S10.5 不可执行边界，以及 B6D 不依赖 S10.5 的安排本身均与上游一致；但顶层批次 DAG 对 B6C-4 增加了计划其他位置不存在的 B6C-3 整体前置，形成可直接误导执行的现役冲突。在 F001 关闭前不能对复合规划对象给 PASS。

## 9. 实际验证命令

执行了以下只读或报告校验：

- `git branch --show-current`
- `git rev-parse HEAD`
- `git status --short`
- `Get-FileHash -Algorithm SHA256 -LiteralPath <固定对象与上游文件>`
- `Get-Content -Encoding utf8 -LiteralPath <固定对象/上游/评审文件>`
- `Select-String -Encoding utf8 -LiteralPath <上游文件> -Pattern <S10.4/S10.5/Gate/maintenance/MaxConns/Start/dead-letter>`
- 内联 Python 静态解析：
  - 57 个 task 三态统计；
  - active 集合；
  - task ID 唯一性；
  - 238 条 requires 的未知目标检查；
  - requires 有向图环检查；
  - 批次外层表 57-task 唯一映射检查。
- `Test-Path`：确认本报告写入前不存在。

写后执行：

- 两个固定对象结束 SHA256；
- 报告 scoped whitespace/diff check；
- 报告 SHA256；
- branch/HEAD/status 与唯一写入核对。

## 10. 写后完整性与漂移检查

| 对象 | 结束 SHA256 | 与开始值比较 |
|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `2C18AA026B2136AF058D409F39B8202DF0529141D011C0B7EBA0A1706B9B9E9A` | MATCH；无漂移 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `A18B467D315DE2536036BDA4CF4CBCDEA5ED6B5B4EBB72E5FD54ADD15385AFFC` | MATCH；无漂移 |

- 结束分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，与开始值一致。
- `git diff --no-index --check -- /dev/null <report>` 未报告 whitespace error；退出码 1 仅表示新增报告相对空文件存在预期差异，另有 Git for Windows 的 LF→CRLF 提示。
- 本轮唯一新增写入是 `.spec/wip/identity-auth/review-tasks-plan-s10.4-s10.5-r01.md`；两个被审对象及其他已有工作树内容均未由本 Reviewer 修改。
- 报告最终 SHA256 不嵌入正文，以完成返回中的最终计算值为准。
