# article-management-S8.1 独立代码定点复审 r02

- **CodeReviewGate：PASS**（本次固定七文件代码复审范围）。
- **`AM-S81-R01-F001`：CLOSED**；新增 finding：0（CRITICAL/HIGH/MEDIUM/LOW 均 0）。
- objectType：code；日期：2026-10-27；本会话为原独立 Reviewer，非原作者 Noether，未委派或修复代码。
- 固定工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 唯一新增写入为本报告；R01 保留，任务、Plan、设计/API/schema/依赖及代码均未由 Reviewer 修改。复审结果交 Lead，不替代其同修订交付核对或任务关闭。

## 修订固定与依据

七文件合同已在 `tasks.md` 的 article-management-S8.1 条目登记。使用同会话已完整取得的 Reviewer Role、add-code-review、development-delivery 编排、checklist/security 红线及必要 core/go-service 规则全文，不重新 Route，不扩全库审查。上游继续采用 article design §3.3.9/§3.4.1、A-MODR-03～06，及 R01 的 F001 修复条件。

开始及专项执行后核对下列 SHA256，全部匹配委派固定值；分支和 HEAD 未漂移。

| 文件 | R02 SHA256 | 相对 R01 |
| --- | --- | --- |
| `internal/article/command_service.go` | `7162FA880F56E95814C46347A75041B81FFDFBA00231E863F7D02F2317D8A2E2` | 不变 |
| `internal/article/handler.go` | `79A6123745C5E0DCBEE3DF239DF39EE46C5FDD6C16EE7E8B9FFA72C3F66EB565` | 不变 |
| `internal/article/policy.go` | `B5A199340C18B505B8F2633715C9EC63B78910FB684332061CCE0258E12FC980` | 不变 |
| `internal/article/moderation_test.go` | `8094598605B8C8EB5AA5940A70BD2E3B03D3C3C3CEB3C0A81BAEB1C451933779` | 新增 F001 回归 |
| `cmd/han-server/main.go` | `A329EE3922FB10696B5C539B9CFC55D02A96D479C39943D1F116700B6E7DEC47` | 不变 |
| `cmd/han-server/main_test.go` | `470F9BD0C40284F57489E440F0A459DA398FDA7BB3C77B158DA784114591A5E2` | 不变 |
| `internal/article/deletion_coordinator.go` | `307BEEDE0F62878D5FA39D81A69529792E3155735D776C565AC3E4AFD46D1C70` | 正式追加的第七文件，修改违规删除条件 |

第七文件前文副本位于原 evidence 根 `C:\Users\J00010~1\AppData\Local\Temp\han-s81-a95a17322f374b42832c37a22e507132\internal\article\deletion_coordinator.go`，亲自核得 SHA256 `51B720A6ACFBDE81966AA959BF38898C68EB61622D7D428D3344DEE2D4CA0D1D`，与委派值一致。

执行 `git -c core.autocrlf=false diff --no-index -- <前文副本> internal/article/deletion_coordinator.go`，exit 1 表示存在差异；实际仅一个 hunk，将 `AuthorID == nil || 非 violating` 改为 `非 published || source 非空 || 非 violating`。没有用 tracked diff 为空冒充无变化。`moderation_test.go` 仍属本卡新增文件，本次全文核对，并对照本会话 R01 已读全文识别新增匿名路径测试及 helper。

R01 报告保留，交回前 SHA256 为 `AD6FED5169DBEFB6AC38481D236575807B6AA527FE185D89D803DB334C922BC2`。完整初始五文件增量审查沿用 R01，五文件固定哈希未变，本次不重复无关审查。

## 两轮实际检查范围

**Round 1：修复增量与原 finding 对照。** 核 `deletion_coordinator.go:214` 起的 active 授权，及 `moderation_test.go:523` 起的匿名 clear→mark→新 ETag 删除、两 category、stale/clear、pending/重试、held/source 隔离和故障注入；全文核对当前 moderation 测试。保留 general access、ValidateArticleState、active 删除状态和管理员检查；未放宽普通 DELETE 或 DeleteOrphan，也未新增删除实现、存储/审计权威。

**Round 2：定点消费者和失败收敛。** 沿未变的 Handler.DeleteViolating→CommandService.DeletePermanent→共享 coordinator→Journal→finalizer.completeWithResult，核对新授权状态如何进入原强 If-Match、pending/version、tombstone、物理 DELETE、Audit 和 completed 链。只补读相应 fixture/delegate/rollback wrapper 与 VisibilityPolicy 状态定义，确认测试确实经过共享 coordinator/真实 finalizer，而非返回固定 204；检查两种失败后隐身与原 ETag 重试。未扩到无关模块或全仓检查。

## 原 finding 复审结论

### `AM-S81-R01-F001` — CLOSED

- **原问题/依据**：design §3.4.1 承认合法 `published + author_id=NULL + source=NULL` 匿名文章；§3.3.9 要求管理员可永久删除当前 violating 文章。旧条件无条件拒绝作者为空，导致匿名文章判违后不能永久删除。
- **修复位置**：`internal/article/deletion_coordinator.go:241`。新条件仅允许 published/source=NULL/violating 状态进入违规删除；不再以作者为空拒绝合法匿名文章。
- **权限和非法状态**：通用 `principal.Allows`、`ValidateArticleState`、active 校验及 admin role 校验仍在条件之前。合法有作者 published/violating 路径继续允许；held/source 不能通过违规入口；非法 publication/source/author 组合继续由既有状态校验拒绝。
- **匿名成功证据**：`TestModeration_AnonymousMarkThenDelete_SharedFinalizerCompletes` 对 violence/sexual 从匿名 clear 状态调用真实 MarkViolation handler，确认匿名归属/publication 保持、取得新 ETag，再由同一 handler 的删除能力进入共享 coordinator/finalizer，得到空 body 204；断言 Article 移除、operation completed/AuditEventID、唯一 tombstone 及其文章版本/operation 绑定、`article.violation.delete` action/actor/metadata。
- **冲突及恢复证据**：`TestModeration_AnonymousDelete_StaleAndRestoredClearAreRejected` 证明判违前旧 ETag 删除返回 412，restore 后以新 ETag 违规删除返回 409，均不启动不可逆删除。
- **失败及 pending 证据**：`TestModeration_AnonymousDelete_PendingHiddenAndRetryConverges` 覆盖 Journal 与 Audit 失败：503、Article 仍 pending、无 completed/audit 绑定、三投影不可读；pending 新 ETag 被拒 412，消除故障后用原删除 ETag 重试收敛到空 body 204，并复核共享完成事实。
- **入口隔离证据**：`TestModeration_AnonymousDelete_HeldSourceRequiresOrphanPath` 对 clear/violating 的 held/source 文章均在违规入口得到 409、未启动删除；改用既有 DeleteOrphan 成功 204，审计为 `article.orphan.delete`。
- **原失败链保留**：R01 仍保留旧条件和失败后果。作者在交接中报告 RED 为匿名 clear→mark→新 ETag 违规 DELETE 得到 409 `article.invalid_state`，随后 GREEN；Reviewer 未回写旧代码重跑 RED，不将作者 RED 说成个人实测。
- **关闭判断**：生产授权条件已直接修复原根因，定点回归覆盖原失败路径及邻近入口隔离，无新增 finding。本 ID 在本固定修订上 CLOSED。

## Reviewer 独立运行证据

工作目录为仓库根目录，均 exit 0，未运行全仓七项：

```text
go test ./internal/article -run '^TestModeration_(Anonymous|DeleteViolating_)' -count=1 -v
PASS
ok han/internal/article 0.193s

go test ./internal/article -run '^TestPermanentDeletion_(AuthorizationAndIfMatch_PrecedeT1|AWIRERTR05ThreeRouteRoleStatusAuthorizationMatrix)$' -count=1
ok han/internal/article 0.112s
```

第一项非零命中 5 个顶层测试及 10 个子例（共 15 个通过节点）：原有有作者删除 success/stale/clear/audit-fails，加本轮匿名两类别成功、stale/restored-clear、Journal/Audit 故障与重试、held/source 两状态隔离。第二项验证原共享授权/If-Match 和三删除入口角色/状态矩阵。未因 hash 未变重复执行 server 或全仓测试。

作者报告专项 article/server 与包级普通/race PASS；Lead 在本轮报告 postfix `-count=1` JSON article 70/server 33 个测试节点 PASS，Manifest 新修订检查已至 race PASS，最后 unit 结果待其回收。这些均按来源记录，不冒充 Reviewer 重跑，不沿用修复前七项全绿证明新修订全部交付检查完成。

## 证据边界与交回

- 匿名 mark 使用真实 Repository/audit.Store SQL 配合模拟 pgx.Tx；测试将 mark 后的 Article 快照交给删除 fixture。删除 fixture 的 repository/Journal/transaction/auditor 为替身，finalizer delegate 调用真实 DeletionFinalizer。它是组件级串联，未贯穿同一真实数据库，也未证明真实生产成功请求 DB 链。
- Audit 失败的组件回滚由测试事务 wrapper 保存/恢复快照；代码核对确认真实 finalizer 在 Audit 前物理删除、Audit 失败返回并触发 defer Rollback，成功后才 complete/Commit。此证据不能代替真实 PostgreSQL 原子性、行锁竞争、参数编码、取消或 commit ack loss 实测。
- 真实 PG、多视图数据库集成、Host/浏览器 E2E、部署、容量与 production readiness 继续未验证；本次 PASS 不外推这些范围。
- 新增缺陷：0；额外待决定/非阻断建议：无。未要求第八文件或改变既有设计/API/schema/依赖。
- **交回 Lead：F001 CLOSED，0 新增 finding，CodeReviewGate PASS。** Lead 继续回收本修订剩余检查并核所有权/范围与 task 关闭条件；Reviewer 不自行更新 task 或执行后续 Git/发布动作。
