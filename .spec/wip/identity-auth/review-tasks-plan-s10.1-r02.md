# identity password-reset durable Tasks/Plan 定点复审 r02

- 日期：2026-10-24
- Reviewer：Gibbs，原非作者独立 Reviewer
- objectType：design/planning 复合对象定点复审
- 工作区 / 分支 / HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审范围：`IDENTITY-S10.1-PLAN-R01-F001`、`IDENTITY-S10.1-PLAN-R01-F002` 及相关回归；不重做全库规划

## 1. 固定对象

| 对象 | 固定 SHA256 | 开始核对 | 结束核对 |
|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `CFD3DE4238DD474FEF53D3A1234A53A7C71ECA14D1A17101B365F3BB31221F18` | 匹配 | 匹配 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `C7A84594DF0FFB8CC68C4800D09AC7A0427110DDBC3DF4C430B6801B6C5EF535` | 匹配 | 匹配 |
| `.spec/wip/identity-auth/review-tasks-plan-s10.1-r01.md` | `B4435639F8B420F29D93A80854BFC7D17941122965B39B7E13EE90AF10E6B35D` | 匹配 | 匹配 |

r02 报告路径在开始时不存在。共享工作树中的 Tasks/Plan 修改和 r01 均为既有现场，本复审没有修改它们。

## 2. 实际检查与命令

### Round 1：原 findings 定点复审

1. 完整读取 r01，并直接检查 Tasks 的 Story-level gap-map、二级能力矩阵、S10.1/S10.4 条目及 S11.1 启动条件。
2. 直接检查 Plan 的批次外层表、R 对账摘要和 B6C Context Pack。
3. 有界回读 r39/stories r15 的 S10/S11 合同及当前代码引用位置，确认新证据不是作者自述：
   - S10 当前仍有旧 Handler/coordinator 三文件，`password_reset.go` 仍含请求事务及 claimed rollback/cleanup 路径；
   - S11 resend 专属实现文件不存在；
   - Idempotency、账户状态与唯一 production composition 的被引位置确实存在；
   - `0008`、password-reset Store/worker/operator 均尚不存在，现役 migration 最大编号仍为 `0007`。

### Round 2：结构与回归检查

实际执行以下只读/静态检查：

- `Get-FileHash -Algorithm SHA256`：开始及结束核固定对象。
- `git branch --show-current`、`git rev-parse HEAD`：确认 `main` 和固定 HEAD。
- PowerShell checkbox 解析：任务总数、三态、active 集合及重复 ID。
- PowerShell 批次表解析：只取 Plan `## 批次外层表`，核全部 task 唯一安排。
- PowerShell requires 图解析和 DFS：核未知目标与环。
- PowerShell 文件清单解析：核 B6C 四卡与 S11.1 写集交叉。
- `Test-Path`：核规划中的新增 migration/Store/worker/operator/resend 文件仍不存在。
- `git diff --check -- <Tasks> <Plan>`：固定对象 scoped diff-check。

结果：

- 56 total / 22 completed / 3 active / 31 todo；
- active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`identity-auth-S10.1`；
- 无重复 task ID；
- 外层批次表 56 次引用、56 个唯一 task，无遗漏、额外或重复；
- requires 图 56 节点、237 条边、0 未知目标、0 环；
- B6C 四卡写集互斥；
- S11.1 只与 S10.1 共享 `internal/identity/handler.go`；
- 固定对象 scoped diff-check PASS。

## 3. 原 findings 复审

### `IDENTITY-S10.1-PLAN-R01-F001` — CLOSED

**原问题**：R gap-map 没有为 S10/S11 各给一个 Story-level 四标签结论，11 行能力矩阵计数冒充 Story 分布；“已建成共享能力”缺当前代码位置证据。

**修订证据**：

1. `.add/project/knowledge/tasks/tasks.md:94-101` 新增独立的 Story-level 表：
   - S10 且仅一行，标签为“需改”；
   - S11 且仅一行，标签为“待做”；
   - 分布明确为已建成 0 / 待做 1 / 需改 1 / 废弃 0。
2. S10 行同时引用 design r39 与当前三文件 `file:line`，现场代码仍显示已有首次 reset Handler/coordinator 和旧 cleanup 路径，支持“需改”。
3. S11 行同时引用 design r39、当前 Handler 接口位置和不存在的 resend 专属文件，支持“待做”。
4. `tasks.md:103-105` 将原 11 行明确降为“第二级能力/接缝 coverage/traceability 矩阵”，并写明 1/4/5/1 只是能力切片计数，不是 Story 分布。
5. `tasks.md:117` 的“已建成共享能力”已补：
   - `cmd/han-server/main.go:1930-2269`；
   - `internal/idempotency/store.go:241-306,478-545`；
   - `internal/identity/account_status.go:83-159,408-505`；
   - design r39 §3.1.4、§3.3.4/6/9/11、§3.4.2/8。
   现场核对这些位置分别存在 production composition、通用幂等分类及账户状态事务/repository 基线；文本同时明确 durable 新增能力不计入“已建成”。
6. `tasks.md:117,121-122` 继续规定 S3.2/F005、S5.2、S8.1 不重开、不重新占用写集。
7. `.add/project/knowledge/plans/ir-01-content-community-batches.md:75` 同步区分 Story-level `0/1/1/0` 与能力切片 `1/4/5/1`。

**结论**：r01 的全部复审条件满足；未发现通过改名、历史关闭记录或作者说明绕过当前设计/代码证据。

### `IDENTITY-S10.1-PLAN-R01-F002` — CLOSED

**原问题**：S10.1/S10.4 的 task-level 关闭相互等待，并与 B6C-3 integrated/batch Gate 混在一起，形成隐藏关闭环。

**修订证据**：

1. `tasks.md:890-892` 明确 S10.1：
   - 在 S10.2/S10.3 后按自身三文件、自动化、切片非作者 Review 与 Lead §10 核证独立关闭；
   - 关闭后再单向交接 S10.4；
   - S10.4、跨卡 integrated Review、代码 F002 定点复审及 batch 接纳均不是 S10.1 关闭前置。
2. `tasks.md:972-974` 明确 S10.4：
   - 只在 S10.1 已按自身条件关闭并完成交接后启动；
   - 按自身八文件、自动化、切片非作者 Review 与 Lead §10 核证独立关闭；
   - 跨卡 Gate 在本卡关闭后另行执行。
3. 两卡 requires 没有互相引用；静态 requires 图仍为 0 环。
4. `plan.md:60` 及 `:654-655` 固定单向顺序：
   `S10.2 → S10.3 → S10.1 implementation slice → S10.4`。
5. `plan.md:655` 将跨卡 integrated Review、Hubble 对代码 F002 的定点复审/关闭和 Lead batch 接纳仅放在两卡各自 task-level 关闭后的 B6C-3 batch Gate。
6. 同一段明确：batch Gate 失败只保持 B6C-3 未接纳并阻止 B6D，不反向伪造或抹除已取得的单卡局部 PASS。
7. `tasks.md:982,995-996`、`plan.md:61,655,659` 一致要求 B6D/S11.1 仅在：
   - B6C 四卡 task-level 均关闭；
   - B6C-3 batch Gate PASS；
   - 代码 `IDENTITY-S10.1-REV-R01-F002` CLOSED；
   三项全部满足后才能启动。

**结论**：任务关闭与 batch Gate 已分层，单向交接无显式或隐藏环，且 batch 失败语义不会篡改单卡状态；r01 的全部复审条件满足。

## 4. 新增 finding 与回归

### 新增 finding

无。

### 回归确认

- 56 / 22 / 3 / 31 与 active 集合未退化。
- 外层批次仍为每个 task 恰好一次；requires 无未知目标、无环。
- B6C 四卡写集仍互斥；S11.1 只与 S10.1 共享 Handler，并保持串行。
- `IDENTITY-S10.1-REV-R01-F002` 仍为 Major OPEN；本次 planning 复审没有关闭它。
- S10.1 仍为 `[-]`、无完成日期、implementation stopped。
- S10.2、S10.3、S10.4、S11.1 均未认领、未启动。
- S1.6 真实 PostgreSQL，S5.3 local real-protocol SMTP，S3.4 approved-provider/external-receipt/真实 Host E2E，S2.2 性能容量的分界保持不变。
- lease duration、retry backoff、自动重试上限、批量领取上限和连接池预算仍是实施前显式输入；缺失时 writer/worker 不开放，规划未加入硬编码默认。

## 5. 待决定、建议与限制

### 待决定

无。两个原 finding 均为规划结构问题，固定修订已经在既有 r39/r16 合同内闭环，不需要 Reviewer 补作架构、产品或风险接受决定。

### 非阻断建议

无新增建议。

### 未验证限制

1. 按授权未运行 Go、build、test、race、vet 或 Manifest 七项；Planning PASS 不证明任何实现或测试通过。
2. 未运行 PostgreSQL、SMTP/TLS、真实 provider、Host、部署、浏览器或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
3. 尚不存在的 migration、Store、worker、operator、resend 实现及其约束、锁序、租约和 lifecycle 行为均未验证。
4. 本复审不评价代码 F002 是否已修复；Hubble 的代码定点复审仍必须在完整实现及 B6C-3 Gate 阶段独立执行。
5. 共享工作树其他既有未提交/未跟踪内容不属于本报告结论。

## 6. Planning Review Gate

**Planning Review Gate: PASS**

`IDENTITY-S10.1-PLAN-R01-F001`：CLOSED。

`IDENTITY-S10.1-PLAN-R01-F002`：CLOSED。

新增 finding：无。

本 PASS 只表示固定 Tasks/Plan 已满足本轮 R 对账与批次规划合同。它不关闭代码 `IDENTITY-S10.1-REV-R01-F002`，不自动恢复 S10.1 实施，不构成 Lead 实施授权，不启动 S10.2～S10.4，也不启动 B6D/S11.1。

## 7. 实际写入

仅新增 `.spec/wip/identity-auth/review-tasks-plan-s10.1-r02.md`。

未修改 Tasks、Plan、r01、设计、代码、migration、Manifest、任务状态或其他文件；未 stage、commit、push、切分支、移动或清理。
