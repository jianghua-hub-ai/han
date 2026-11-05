# article-management-S8.1 独立代码评审 r01

- CodeReviewGate：**FAIL**。
- Findings：HIGH 1；CRITICAL/MEDIUM/LOW 0。阻断项 `AM-S81-R01-F001`，交 Lead 串行安排原作者 Noether 修复。
- objectType：code；日期：2026-10-27。本会话为独立 Reviewer，非作者 Noether；未委派、未修复、未执行 Git 写入、清理或外部操作。
- 固定工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。评审对象是下面六文件的未提交修订，不能用 HEAD 或 tracked diff 代替。
- 唯一写入：本报告。发现修复需要第七个代码文件后停止扩查与新增验证，只完成固定对象复核和报告交回；未修改其他制品或任务状态。

## 固定修订与基线

开始和交回前分别实测六文件 SHA256，均与委派值一致，分支/HEAD 未漂移。

| 文件 | SHA256 |
| --- | --- |
| `internal/article/command_service.go` | `7162FA880F56E95814C46347A75041B81FFDFBA00231E863F7D02F2317D8A2E2` |
| `internal/article/handler.go` | `79A6123745C5E0DCBEE3DF239DF39EE46C5FDD6C16EE7E8B9FFA72C3F66EB565` |
| `internal/article/policy.go` | `B5A199340C18B505B8F2633715C9EC63B78910FB684332061CCE0258E12FC980` |
| `internal/article/moderation_test.go` | `1528E198627C50CF5A8D15771DA1747A1AD4A7555CF74083045DEC1F072BBC9C` |
| `cmd/han-server/main.go` | `A329EE3922FB10696B5C539B9CFC55D02A96D479C39943D1F116700B6E7DEC47` |
| `cmd/han-server/main_test.go` | `470F9BD0C40284F57489E440F0A459DA398FDA7BB3C77B158DA784114591A5E2` |

五个原有文件使用 `git diff --no-index -- <前文副本>/<相对路径> <当前相对路径>` 审查真实差异；逐文件 diff 有差异（exit 1），非执行失败。副本根目录：`C:\Users\J00010~1\AppData\Local\Temp\han-s81-a95a17322f374b42832c37a22e507132`。实测副本中没有 `moderation_test.go`，按新增文件全文审查。现场 `internal/`、`cmd/` 为 untracked，未以 tracked diff 为空声称无变更。

Lead 在会话中报告：664 文件起始 hash 对账仅六代码文件及 tasks/Plan 范围变化，无其他漂移或删除。本 Reviewer 未持有该起始全集，不将此声明写成个人重跑的全库范围证明；个人核对覆盖上述六文件及实际读取的直接依赖。

## 依据与两轮实际范围

已完整读取本端 `add-reviewer.toml`、`add-code-review/SKILL.md`、首轮定位用 `add-route-work/SKILL.md`、所属 `add-delivery-workflow/SKILL.md`；加载全文 `code-review-checklist`、`abstraction-principles`、`coding-standards`、`no-hardcode-policy`、`observability-standards` 和 `产品安全红线.md`。启用 profile 仅 `go-service`：读取 README 及 coding/security/observability/no-hardcode 规则全文。未加载未启用 profile；未改规则或采用配置。

上游依据：`tasks.md` 的 article-management-S8.1 六文件卡；article design §3.3.9、§3.4.1，及直接相关的 §2.4 写事务/审计表与 §3.3.6～7 删除入口合同；test-plan `A-MODR-03～06`。当前环节为 development-delivery 的独立代码评审，输出交回 Lead，不替代批次/E2E/人的确认。

**Round 1：六文件真实增量。** 核对两治理 POST、强 If-Match/资源版本匹配、严格 JSON/大小/UTF-8/类别、管理员及 onboarding 限制、clear/violating/pending 状态、只写违规事实及 version/updated_at、响应 AdminProjection、审计字段和错误根因。全文检查新增 moderation 测试及 main 测试增量，区分事务模拟与真实数据库行为；未把测试全绿作为契约证明。

**Round 2：有界直接调用链。** 从 production builder/router 到 OriginGuard、HTTPAuthorizer 与真实 Article handler；从 Moderate 到 Repository 的锁定扫描、限定 UPDATE、当前 pgx.Tx 上的 identity ArticleIdentityReader、audit.Store.Append、Commit/Rollback；对照既有内容更新和 member/author/admin 投影与查询；沿 DeleteViolating 到共享 coordinator/finalizer 的授权、Journal、物理 DELETE/Audit/completed 同事务。仅补读 Principal.Allows/Restricted 的直接定义。未扩成全库预读或身份模块全审。

核对要点：

- 新治理路径用 Article 行锁串行化内容修改、违规变化与删除；UPDATE 参数依次为 id/state/category/version/time，并以旧版本及 active 删除状态保护，未写正文、归属、来源或 publication。
- `main.go:3020` 将当前 pgx.Tx 传给 `identity.NewArticleIdentityReader`；reader 为普通 SELECT，无新 User 行锁、无持有事务后再次从 pool 取连接的代码路径。缺失或不支持的作者状态拒绝提交。
- `command_service.go:259` 通过共享 audit.Store 在同一事务写 actor/action/target/version/category/trace，不记录标题正文；失败保留 cause。提交失败不返回成功 payload/ETag，实际 commit ack loss 尚未验证。
- 路由保持 Trace 在外层，正确方法经过 Origin→Session/CSRF→Principal→管理员校验；405/Allow 与未知路径 404 使用既有 JSON 边界。真实生产拒绝测试未注入替代治理 handler，但 PrincipalResolver 和基础设施仍有测试替身。
- 成员查询/作者受控详情/管理员投影继续复用 VisibilityPolicy；现有测试的三视图断言为组件投影，尚非真实 HTTP+PostgreSQL 多视图集成。
- 第二轮发现下面的删除消费者拒绝合法匿名文章，阻断治理闭环。与上游合同逐项对照后停止扩大范围。

## Findings

### [HIGH] 匿名公开文章判违后没有可用的永久删除入口

- **finding ID**：`AM-S81-R01-F001`。
- **对象/修订**：上述固定六文件的 S8.1 修订及其复用的直接删除消费者。拒绝条件在既有依赖中，并非本轮新增的源码行；本卡明确承诺复用它完成违规删除，因此是本次集成验收缺陷。
- **位置**：`internal/article/deletion_coordinator.go:241`（`DeleteViolating` 的拒绝条件）；新生产接线 `cmd/han-server/main.go:3042`；本卡删除覆盖 `internal/article/moderation_test.go:476`。
- **违反依据**：design §3.4.1 明确 `published + author_id=NULL` 是合法匿名文章、管理员可治理；§3.3.9 的 `DELETE /api/v1/admin/articles/{id}` 删除当前 violating 文章，未要求作者非空。S8.1 目标包含违规永久删除，A-MODR-06 要求共享删除协议闭环。
- **触发**：已有合法匿名公开文章（`author_id=NULL`、`orphaned_from_user_id=NULL`、`publication_state=published`、`deletion_state=active`）。管理员持当前强 ETag 判定 violence/sexual，随后持新 ETag 调用违规 DELETE。该状态由设计承认；本卡 `TestModeration_Restore_RetainsOtherConstraints/anonymous` 也直接使用匿名 violating 输入。
- **实际行为/证据**：新 `moderationAllowed` 不排除该合法状态，`moderationProjection` 对无作者跳过 identity 读取，所以可正常判违。随后共享 `authorizeActiveDeletion` 的 `article.AuthorID == nil || article.ViolationState != ViolationViolating` 恒进入 `ErrArticleInvalidState`，HTTP 映射 409；在创建 DeletionOperation/写 Journal 前返回。普通 DELETE 同样要求作者非空且 clear；无归属 DELETE 在 `deletion_coordinator.go:249` 要求 source 非空且 held。因此该匿名文章没有替代删除入口。
- **影响**：管理员能够隐藏/解除匿名内容，却不能按既定合同永久删除其中违规内容；无法进入 tombstone/物理删除/审计闭环。保持隐藏不能满足永久删除验收。
- **证据性质**：直接调用链与确定性授权分支的静态证明，未伪称运行过新增匿名 DELETE 探针或真实 PostgreSQL 请求。现有删除专项只基于有作者 fixture 覆盖 success/stale/clear/audit-fails，绿灯没有覆盖这一状态。
- **最小修复方向**：由 Lead 串行交原作者，核准并调整共享违规删除授权，使合法 `published + NULL author + NULL source + violating + active` 可进入同一 coordinator/finalizer；保留 held/source 无归属入口的职责约束，勿以禁止匿名判违或另建删除实现绕过合同。
- **范围停止**：最小修复涉及第七文件 `internal/article/deletion_coordinator.go`，超出本轮六文件写集；Reviewer 未修改，按委派要求停止扩查并交回。Lead 需先安排精确写集后再由作者实施。
- **复审条件**：引用本 ID 与新修订，新增匿名 clear→mark→以新强 ETag 违规 DELETE 的失败/通过证据；验证双完成 204、共享 Journal/finalizer/`article.violation.delete` 审计；同时覆盖 stale 412、clear 409、pending 隐藏、held/source 路径隔离及审计失败不提交物理删除。重新核固定文件及新增授权文件 hash、真实 diff 和相关回归；未闭环前本 finding 保持 open。

## 运行证据

**Reviewer 亲自执行**（仓库根目录，exit 0）：

```text
go test ./internal/article -run '^TestModeration_(Failures_RollBackArticleAndAudit|DeleteViolating_UsesSharedFinalizer|HTTP_CompareFailureIsPreconditionFailed)$' -count=1 -v
PASS
ok han/internal/article 0.297s
```

命中三个顶层测试及 16 个子例：begin/lock/reader/update/compare/audit/commit/rollback/missing-author/audit-ID/clock/trace，CAS 412，删除 success/stale/clear/audit-fails。全部通过；使用真实 Repository/audit.Store SQL 配合模拟 pgx.Tx，删除场景沿真实 coordinator/finalizer 配合 repository/transaction fixture，不证明 PostgreSQL 的实际回滚或锁竞争。

**Lead 本轮现场报告，非 Reviewer 重跑**：`go test -json` 的 `^TestModeration`、`-count=1` 专项 PASS，article 60、server 33 个通过节点（包括顶层与子例，非 93 个独立顶层测试）；Manifest 固定七项 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...` 全部 PASS，目标非零命中。作者普通/race/vet/build PASS 和 RED 先缺接口/404 的历史由交接输入提供；Reviewer 未重建 RED 环境。按用户要求未重复全仓检查。

## 限制、待决定与交回

- 未实测真实 PostgreSQL SQL 类型/参数编码、真实行锁竞争/取消、数据库提交确认丢失及实际回滚；模拟 commit 失败不能证明服务器未提交。
- production 测试证明真实治理 handler 的拒绝链和当前事务 reader；成功请求穿过真实生产 builder/SessionStore/Repository/数据库的全链尚未实测。三视图为组件验证，未完成真实 PostgreSQL 多视图或浏览器/Host E2E、部署、容量与 production readiness。
- 停止条件触发后未追加其他专项或扩大依赖范围；本结论不宣称全库、全部消费者或全部安全红线均验证。
- 待决定：Lead 安排涉及共享删除授权的第七文件写集及串行修复；不由 Reviewer 改架构/契约、接受风险或关闭 task。
- 非阻断建议：无另列；不以风格意见增加作者整改范围。
- 交回：`CodeReviewGate FAIL`，`AM-S81-R01-F001` open。报告交 Lead；原作者修复后由独立 Reviewer 针对原 finding 和新修订复审。
