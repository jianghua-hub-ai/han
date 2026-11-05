# Identity Story S03 / F005 tasks 与 batch-plan r06 独立评审

## 0. 结论摘要

- **评审日期**：2026-10-23。
- **Reviewer**：Helmholtz（add-reviewer，非本轮 Task-Planner Kuhn）。
- **对象类型**：design；复合 planning 对象（tasks SSOT + batches Plan）。
- **最终结论**：**FAIL，规划 Gate 不通过**。
- **Findings**：CRITICAL 0 / HIGH 0 / **MEDIUM 1** / LOW 0。
- **阻断项**：`TP-S03-R06-F001`。I-F005-26～28 的显式“场景—文件映射”与 accepted design r26 的真实责任边界不一致；必须先在两个规划对象中同步修正，再由非作者复审。
- **边界说明**：本结论只评价固定 planning 对象，不代表实现、测试执行、PostgreSQL 18、benchmark/profile、部署或 E2E 的结果；也不授权编码、测试执行、任务关闭、后续批次、Git 或发布。

## 1. 固定对象与完整性

工作区：`D:\AID\VSCode\Codex\Han`  
分支：`main`  
HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`

| 固定对象 | 预期 SHA256 | 开始 SHA256 | 写报告前 SHA256 | 结束 SHA256 |
|---|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `EFC0C8E23A2E4EDC34150A74B79450EAA8F1E47F252A0AC303E6F06C0E452EC5` | `EFC0C8E23A2E4EDC34150A74B79450EAA8F1E47F252A0AC303E6F06C0E452EC5` | `EFC0C8E23A2E4EDC34150A74B79450EAA8F1E47F252A0AC303E6F06C0E452EC5` | `EFC0C8E23A2E4EDC34150A74B79450EAA8F1E47F252A0AC303E6F06C0E452EC5` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `5A455EA4938571E7CF34F7E5B4E8E10D0AE7332A088D9C8F8D366429F5A1CBB6` | `5A455EA4938571E7CF34F7E5B4E8E10D0AE7332A088D9C8F8D366429F5A1CBB6` | `5A455EA4938571E7CF34F7E5B4E8E10D0AE7332A088D9C8F8D366429F5A1CBB6` | `5A455EA4938571E7CF34F7E5B4E8E10D0AE7332A088D9C8F8D366429F5A1CBB6` |

固定对象在评审期间未漂移。两个对象的 scoped `git diff --check` 返回 0。

上游依据实际核对：

- design revision：`identity-auth-design-ir01-r26`；当前文件 SHA256 `04FFC97CA04465C23707604B56E103627663AF432C523F2F0A584B7C39A00217`。
- checkpoint：`decision-a76cb64cb07a0e37c80b3d22b15034e817e68d7c00a9be1759372e0396e6a56f`；object digest `sha256:42d8d10c37acb55a752c198caef32819d4d73744acd71af6fb1a4fba0366bc3d`。
- design review SHA256：`171E373555357493003EBC99F5704281304E86282EA8A709B14AAB4E4A82DE28`，PASS / 0 finding。
- test-plan revision：`identity-auth-test-plan-ir01-r08`；SHA256 `C95518C5AC7C5DAEE56F54D0E6691E999BFB8CF006E0D71956ED8457F97AD23D`。
- test-plan review SHA256：`4D6A1C7844116A15F8EA6E319A82F92CFADF89757E5D1D910EC1FA1D710472DA`，PASS，原 `TP-R07-F001/F002` CLOSED。
- 原 planning review r05 SHA256：`7315F6BB34F1C99620929F64986C449606780D58E3B693E30D48900C61524DE2`。

## 2. 实际读取与方法

完整读取：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/task-execution-workflow.md`（任务三态、关闭与授权条件）
- `.codex/guidance/design-doc-writing-style.md`
- `.add/project/knowledge/design-workspace/identity-auth/design.md`
- `.add/project/knowledge/design-workspace/identity-auth/test-plan.md`
- `.spec/wip/identity-auth/review-tasks-plan-s3.2-f005-r05.md`
- 两个固定 planning 对象全文

有界反查了十八个真实代码/测试消费者，重点包括 `cmd/han-server/main.go`、`internal/identity/handler.go`、`internal/platform/http_security.go` 及相关测试；未执行代码或环境测试。

## 3. Round 1：固定对象直接检查

1. 独立解析 tasks：
   - 共 51 tasks；
   - 12 completed / 1 active / 38 todo；
   - 唯一 active 为 `identity-auth-S3.2`；
   - task ID 重复 0；
   - requires 缺失引用 0；
   - DAG 无环。
2. `identity-auth-S3.2` 保持 `[-]`、owner Popper、开始日期 2026-10-22；F001～F004 CLOSED、F005 OPEN；B4 未关闭。
3. gap-map 保持“已建成 3 / 待做 2 / 需改 1 / 废弃 0”，总体“需改”。
4. S3.2 登记精确十八个既有文件；四个新增到卡片写集的文件仍保留 S2.1/S1.1 历史归属，只授予后继、有界、串行修改范围；未发现第十九文件、新依赖、schema/migration、ADR 或公共契约扩张。
5. I-F005-13～29 均在 tasks 与 Plan 中出现；I-F005-13～22 的既有映射和 planned/not-executed 状态未发现回退。
6. S2.2 两文件合同明确覆盖目标环境、候选 tuple、concurrency 1～4、内存/延迟预算和 `profile ID → benchmark 报告 → 采用记录`，并保持 `[ ]` / `blocked-for-execution` / 未执行。
7. Plan 批次外层表把 51 个 task 各分配一次：missing 0 / extra 0 / nonunique 0；Plan 未复制 task checkbox 状态。
8. `S3.2 → S3.5 → S3.6`、B4 `G4-batch`、B5C/B11B 另行授权 Gate、S3.4 用户旅程独立边界均保留。
9. Round 1 发现 I-F005-26～28 的显式场景—文件映射与 r26 不一致，形成 `TP-S03-R06-F001`。

## 4. Round 2：真实依赖、消费者与状态深查

1. 十八个 S3.2 目标文件全部真实存在，数量为 18。
2. `cmd/han-server/main.go:460-464` 的 production `composeBusiness` 仍返回 `identity business route provider is not configured`，即 fixed-fail；规划没有把 r26 factory/Gate/Header/source/router 写成已实现。
3. `internal/identity/handler.go:29-31,113,166` 真实拥有并消费 `RegistrationSourceProvider.Evidence(*http.Request)` 接缝；现役 `internal/platform/http_security.go` 只实现 OriginGuard/Origin rejection 职责，没有 RemoteAddr/XFF source provider。这与 design r26 `design.md:2761,2768-2769` 一致，反证 tasks/Plan 把 I-F005-27 放入 platform HTTP security 是错误映射。
4. design r26 `design.md:2762-2765` 把 recovery 四项合取、adapter/Verify 调用矩阵放在 identity `service.go`、`login.go` 及其测试；test-plan r08 `test-plan.md:187,413` 同样要求 identity 状态机、Handler/集成回归。tasks/Plan 却把 I-F005-26 的显式映射收窄到 composition `main.go/main_test.go`。
5. design r26 `design.md:2777-2778` 把 JSON 404/405 稳定公共错误注册与测试放在 `platform/errors.go/_test.go`；test-plan r08 I-F005-28 要求 JSON 404/405/Allow 与禁止 raw fallback。tasks/Plan 的映射未把 errors 两文件归入 I-F005-28，反而只列在 I-F005-29。
6. production factory、PG18 当前复跑、Manifest 七项、Argon2 benchmark/profile、部署、真实 observability 验证和 E2E 均被如实标为未执行或 blocked；历史 PG18 证据未被冒充当前执行。
7. 本 revision 没有把 planning PASS、design/test-plan PASS 或合法 profile token冒充实施授权、任务关闭或 production readiness。

## 5. Findings

### `TP-S03-R06-F001` — MEDIUM / OPEN / 阻断

**标题**：I-F005-26～28 的显式场景—文件映射违反 accepted r26 的责任边界。

**位置**

- `.add/project/knowledge/tasks/tasks.md:353,357`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md:459-460`
- 对照依据：
  - `design.md:2761-2769,2771-2778`
  - `test-plan.md:187-190,413-417`

**触发场景**

后续 Backend-Dev 按 tasks 卡或 Plan Context Pack 的显式映射实施，而不重新推翻 planning 指令：

1. I-F005-26 只在 `main.go/main_test.go` 做 factory/composition 测试；
2. I-F005-27 在 `internal/platform/http_security.go/_test.go` 实现 strict RemoteAddr/XFF；
3. I-F005-28 未在 `internal/platform/errors.go/_test.go` 登记 404/405 公共错误映射。

**影响**

- recovery 的前三项前置、adapter 恰好一次、normal/error fail-closed、maintenance 才 Verify 等状态机证据可能遗漏真实 identity owner；
- source provider 可能被错误放进 platform OriginGuard，混淆 Origin 隔离与可信来源证据边界，并使 identity Handler 的去敏 source 接缝没有正确实现/测试；
- JSON 404/405 的稳定 code/envelope/`Allow: POST` 可能缺少公共错误注册测试，增加 raw fallback 或路由契约不完整风险；
- tasks 与 Plan 相互复制同一错误，Context Pack 不能作为 r26 的可靠执行输入。

**最小修复建议**

只修订两个 planning 对象的步骤和场景—文件映射，不改变十八文件上限、不新增 task/依赖/公共契约：

1. I-F005-26 映射到 identity `service.go`、`login.go`、`login_test.go`、`login_integration_test.go`，并保留 `main.go/main_test.go` 的 production wiring 证据；
2. I-F005-27 映射到 `internal/identity/handler.go` 及十八文件内既定的 identity/composition 测试消费者；`platform/http_security.go/_test.go` 只保留 OriginGuard 与 router 隔离回归；
3. I-F005-28 明确包含 `main.go/main_test.go`、`platform/http_security.go/_test.go` 和 `platform/errors.go/_test.go`；identity Handler/Service 只作为可计数下游/零调用消费者，不承担 local router 或公共错误注册；
4. I-F005-29 继续承接 Argon2 Gate、typed capacity error、identity 429 carrier/Header 与动态 abuse 回归，不用它替代 I-F005-28 的 route errors。

**复审条件**

- 固定新的 tasks/Plan SHA；
- 两对象的 I-F005-26～29 映射与 r26 §9.2、test-plan r08 一致；
- 十八文件上限、历史 ownership、任务三态、DAG、B4/B5C/B11B Gate 和 blocked/not-executed 状态不变；
- scoped `git diff --check` PASS；
- 非作者 Reviewer 重新执行两轮定点复审。

## 6. 历史 finding 回归、待决定与建议

### 历史 finding 回归

r05 已关闭的 `TP-S03-R04-F001～F004` 未发现回退：S3.2 可按自身 Gate 独立关闭而不等待 S3.5/S3.6；S3.5 材料、S3.6 真实验证、B5C/B11B 外层 Gate及 Manifest 当前事实仍保持分离。

### 待决定

无需要新增架构/ADR、公共契约、第十九文件或新依赖的方向决定。`TP-S03-R06-F001` 可在既有 accepted r26 与十八文件边界内修正。

### 非阻断建议

修订时保持 tasks 与 Plan 的场景—文件映射采用同一顺序和同一责任术语，避免 Context Pack 再次从 SSOT 复制出不同 owner。

## 7. 未验证限制与剩余风险

- 按授权未运行 Go、PostgreSQL 18、Manifest 七项、benchmark/profile、部署、E2E、真实代理、真实 observability 平台或告警演练。
- 未验证 I-F005-13～29 的实现行为；当前只核 planning 合同、静态消费者与状态真实性。
- 未验证目标环境 CPU/内存/延迟预算、profile 采用记录、provider/receiver、账号、费用、自然人排班、网络策略或演练窗口。
- 工作树包含他人未提交内容；本报告只对两个固定哈希对象及有界依赖给结论，不声称全仓无其他漂移。
- planning FAIL 不等于既有 I-F005-01～12 或历史 PG18 证据失败；也不授权 Reviewer 修复规划对象。

## 8. Gate

**规划 Gate：FAIL。**

在 `TP-S03-R06-F001` CLOSED 前，不应把本 revision 作为 S3.2/F005 的可执行 planning Gate。修订不得改变现有任务三态、十八文件上限、历史 ownership、blocked/not-executed 状态或后续另行授权边界。
