# Identity Story S03 / F005 tasks 与 batch-plan r07 定点复审

## 0. 结论摘要

- **复审日期**：2026-10-23。
- **Reviewer**：Helmholtz（add-reviewer，r06 原非作者 Reviewer；非 Task-Planner Kuhn）。
- **对象类型**：design；复合 planning 对象（tasks SSOT + batches Plan）。
- **原 finding**：`TP-S03-R06-F001`（MEDIUM / OPEN）。
- **复审结果**：`TP-S03-R06-F001` **CLOSED**。
- **新增 findings**：0（CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0）。
- **最终结论**：**PASS，规划 Gate PASS**。
- **边界说明**：本 PASS 只证明固定 planning revision 已正确消费 accepted r26/r08 并关闭原 finding；不等于 F005 实现、测试执行、PostgreSQL 18、benchmark/profile、部署或 E2E 通过，也不构成编码、任务关闭、后续批次、Git 或发布授权。

## 1. 固定对象与完整性

工作区：`D:\AID\VSCode\Codex\Han`  
分支：`main`  
HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`

| 对象 | 预期 SHA256 | 开始 SHA256 | 写报告前 SHA256 | 结束 SHA256 |
|---|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `2BF2018C3DFAE54329AEB1E4583B7384FF88A0697C97DA53C34EDC0747C3B2E4` | `2BF2018C3DFAE54329AEB1E4583B7384FF88A0697C97DA53C34EDC0747C3B2E4` | `2BF2018C3DFAE54329AEB1E4583B7384FF88A0697C97DA53C34EDC0747C3B2E4` | `2BF2018C3DFAE54329AEB1E4583B7384FF88A0697C97DA53C34EDC0747C3B2E4` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `23B7BEC990AF990CEEE9458076E8B266383F21DE83A74982DAE70A6B412D88D0` | `23B7BEC990AF990CEEE9458076E8B266383F21DE83A74982DAE70A6B412D88D0` | `23B7BEC990AF990CEEE9458076E8B266383F21DE83A74982DAE70A6B412D88D0` | `23B7BEC990AF990CEEE9458076E8B266383F21DE83A74982DAE70A6B412D88D0` |
| 原评审 `.spec/wip/identity-auth/review-tasks-plan-s3.2-f005-r06.md` | `09ED43AC5372864862D5217B6BCFEEF4BE657A04B820F7D621850988FD2C76C5` | `09ED43AC5372864862D5217B6BCFEEF4BE657A04B820F7D621850988FD2C76C5` | `09ED43AC5372864862D5217B6BCFEEF4BE657A04B820F7D621850988FD2C76C5` | `09ED43AC5372864862D5217B6BCFEEF4BE657A04B820F7D621850988FD2C76C5` |

固定对象与原评审在复审期间均未漂移。两个 planning 对象的 scoped `git diff --check` 返回 0。

上游对象保持：

- design revision `identity-auth-design-ir01-r26`，文件 SHA256 `04FFC97CA04465C23707604B56E103627663AF432C523F2F0A584B7C39A00217`；
- test-plan revision `identity-auth-test-plan-ir01-r08`，文件 SHA256 `C95518C5AC7C5DAEE56F54D0E6691E999BFB8CF006E0D71956ED8457F97AD23D`；
- checkpoint/object digest 与 r06 一致，无需新增设计或契约决定。

## 2. 方法与实际读取

`add-reviewer`、`add-code-review`、`code-review-checklist`、`abstraction-principles`、`task-execution-workflow` 的 SHA256 与 r06 评审时一致，复用其完整有效正文。

本轮实际完整读取：

- r06 原评审；
- 两个新固定 planning 对象；
- identity design r26；
- test-plan r08；
- finding 涉及的 identity/platform/security/composition 实现和测试消费者：
  - identity：`handler.go`、`service.go`、`login.go`、`login_test.go`、`login_integration_test.go`；
  - platform：`http_security.go/_test.go`、`errors.go/_test.go`；
  - security：`argon2.go/_test.go`；
  - composition：`main.go/main_test.go`。

## 3. Round 1：原 finding 定点复审

### Closure matrix

| 原 finding 要求 | 修订证据 | 判断 |
|---|---|---|
| I-F005-26 由 identity service/login 及两测试承担 recovery 状态机；main 只保留 wiring | `tasks.md:352,357`、`Plan:459-460` 明确由 `service.go`、`login.go`、`login_test.go`、`login_integration_test.go` 承担前三项前置、adapter 恰好一次、normal/error/unknown fail closed、maintenance 才 Verify；`main.go/main_test.go` 只保留 production wiring/装配证据 | CLOSED |
| I-F005-27 source provider 归 identity Handler；platform HTTP security 只保留 OriginGuard/router 隔离 | `tasks.md:353,357`、`Plan:459-460` 明确 `handler.go` 拥有 strict RemoteAddr/XFF source provider，既定 identity/composition tests 承接证据；`http_security.go/_test.go` 明确不承担 source provider | CLOSED |
| I-F005-28 显式包含 main、HTTP security、platform errors 及测试；identity 只是零调用下游 | `tasks.md:353,357`、`Plan:459-460` 明确 `main.go/main_test.go`、`http_security.go/_test.go`、`errors.go/_test.go` 分别承担 local router、OriginGuard 分支和公共 route error；Handler/Service 只作可计数零调用下游 | CLOSED |
| I-F005-29 保持 Gate/capacity/identity 429/Header/dynamic abuse，不替代 route errors | `tasks.md:351,357`、`Plan:459-460` 明确固定 `1s`、`Retry-After: 1`、startup 无 HTTP、动态 abuse 独立，并两次声明不得替代 I-F005-28 route errors | CLOSED |
| tasks 与 Plan 顺序、责任术语及十八文件边界一致 | 两对象均按 23-key → Gate/29 → factory/26 → source/28 → 13～22 回归排列；场景—文件映射语义一致，十八文件清单未变 | CLOSED |

### 原 finding disposition

`TP-S03-R06-F001`：**CLOSED**。

修订没有通过扩文件、换 owner 或弱化测试来规避 finding，而是在 accepted r26 的既有十八文件边界内同时修正步骤与显式映射。未发现原错误映射的当前时态残留。

## 4. Round 2：消费者与状态回归

### 4.1 真实消费者反查

1. `internal/identity/handler.go:29-31,113,166` 真实定义并消费 `RegistrationSourceProvider.Evidence(*http.Request)`；与修订后的 I-F005-27 owner 一致。
2. `internal/platform/http_security.go` 当前只有 OriginGuard/Origin rejection 职责，没有 RemoteAddr/XFF source provider；与“不承担 source provider”的修订一致。
3. design r26 `design.md:2761-2768` 仍分别把 source、recovery 与 OriginGuard 分配给 identity Handler、identity Service/Login 和 platform HTTP security；新 mapping 与其一致。
4. design r26 `design.md:2777-2778` 仍把 JSON 404/405 公共错误注册分配给 `platform/errors.go/_test.go`；新 I-F005-28 mapping 已覆盖。
5. test-plan r08 I-F005-26～29 的状态机、source、router 与 capacity 预期均被新 mapping 消费，没有把应用 conformance 与真实部署/E2E 混合。
6. `cmd/han-server/main.go:460-464` 的 production `composeBusiness` 仍 fixed-fail；本次 planning 修订没有冒充实现完成。

### 4.2 结构与状态回归

独立解析结果：

- tasks：51；
- completed 12 / active 1 / todo 38；
- 唯一 active：`identity-auth-S3.2`；
- duplicate task ID：0；
- requires 缺失引用：0；
- DAG cycle：0；
- 批次外层表分配：51 次、51 个唯一 task，missing 0 / extra 0 / nonunique 0；
- Plan task checkbox 状态副本：0。

状态和边界：

- S3.2 保持 `[-]`、owner Popper、开始日期 2026-10-22；F001～F004 CLOSED、F005 OPEN；B4 未关闭。
- gap-map 保持已建成 3 / 待做 2 / 需改 1 / 废弃 0，总体“需改”。
- 十八个目标文件均存在，清单与 Plan 完全一致；S2.1/S1.1 历史 ownership 和后继串行授权未变。
- S2.2、S3.4、S3.5、S3.6 均保持 todo；S2.2 继续独立拥有 target-profile readiness，S3.4 用户旅程独立。
- task DAG 保持 `S3.2 → S3.5 → S3.6`；B4 G4-batch、B5C、B11B 与另行授权 Gate 未弱化。
- production factory/Gate/Header/source/recovery/router、当前 PG18 复跑、benchmark/profile、部署、真实 observability 验证和 E2E 均保持未执行或 blocked。
- 未新增第十九文件、task、依赖、schema/migration、ADR 或公共契约；规划 PASS 未被写成实施或任务关闭授权。

## 5. 新 findings

**无。**

未发现新引入的 CRITICAL、HIGH、MEDIUM 或 LOW finding。

## 6. 命令与结果

- `Get-FileHash -Algorithm SHA256`：两个固定对象及 r06 开始、写报告前、结束值均与预期一致。
- 全文 `Get-Content -Raw -Encoding UTF8`：r06、两个对象、r26、r08 及相关消费者读取成功。
- Python 只读解析：51 / 12 / 1 / 38、唯一 active、重复 ID、requires、DAG、批次分配和 Plan checkbox 检查均通过。
- `rg` / `Select-String`：核对 I-F005-26～29 映射、真实 source/Origin/error/composition seam 与 blocked/not-executed 状态。
- `git diff --check -- <tasks> <Plan>`：PASS。
- 未运行 Go、PostgreSQL 18、Manifest 七项、benchmark、部署或 E2E。

## 7. 未验证限制与剩余风险

- 本轮只复审 planning 修订；没有验证 I-F005-13～29 的代码实现或运行行为。
- 未验证 PostgreSQL 18 当前环境、Argon2 目标环境性能/profile 采用记录、真实代理/network policy、部署平台、日志留存、scrape、告警、receiver 或用户旅程。
- F005、S3.2、B4 仍未关闭；后续实现必须遵守十八文件上限并取得另行编码/测试授权。
- 工作树含他人未提交内容；结论仅覆盖固定哈希对象和声明的有界消费者。

## 8. Gate

**规划 Gate：PASS。**

`TP-S03-R06-F001` 已关闭，且无新增未关闭阻断 finding。该 PASS 可作为 Lead 接纳本 revision 的独立 planning Review 证据，但不能替代实施、代码 Review、测试执行、部署或 E2E Gate。
