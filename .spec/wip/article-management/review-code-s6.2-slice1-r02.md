# S6.2 切片①独立 CodeReview R02

**AM-S62-S1-R01-F001：CLOSED。slice1CodeReviewGate：PASS（仅 Identity 局部能力）。** 两轮定点复审无新增 finding；不代表真实 Article/provider/root/PG 或完整 S6.2 通过。

## 固定对象与权限

- objectType：`code`；原作者 Turing，本执行者非作者；固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- `internal/identity/article_disposition.go` SHA256：`EB053611CE9346A9A51C0485DD9AE1003EA99C57AF0A3842A4621D67ECE04593`。
- `internal/identity/article_disposition_test.go` SHA256：`A92E8C4D65C76361094B472F35BEDEAB77F15C2FC5E0ECA54854BCDDC640EDF4`。
- R01 保留：`.spec/wip/article-management/review-code-s6.2-slice1-r01.md`，SHA256 `4617DB13B703E50514EBAF11068732D6F25F8C2DD9346DA91949CF2D200F2594`。
- 复用本会话已核读的 Role/add-code-review、适用 coding/go-service/安全规则、design r16、test-plan r14 和 R01 全文审查证据。本轮未重读全库、未改代码、未委派、Git 写入或清理；唯一新增文件为本报告。

## 两轮实际检查与关闭依据

Round 1：检查生产文件三个修改点及测试增量；以内存逆向还原并匹配 R01 两文件原 SHA256，确认没有未纳入检查的文件内其他差异。生产文件只新增 absent-delete 前置分支及调整后续 orphan/non-orphan 前像门禁；测试只新增 `:419`～`:531` 的 helper/四项测试和 `:723`～`:726` 的 transfer absent 拒绝例。未落盘旧副本。

Round 2：沿既有 Clear 调用顺序，复核新分支仍经过当前 tx/source/reference、Identity deleted/Identifier 锁、固定 verifier、operation/审计/计数绑定，再决定释放或 unchanged；对照 r16 的“有来源 absent 可清理、无来源不得猜测”合同，检查重入和错误路径。SQL 锁序、事务所有权、DTO、释放 SQL 与审计写入均未修改，沿用 R01 有界结论。

| 检查项 | 证据与结论 |
|---|---|
| 原 F001 成功分支 | `article_disposition.go:403` 仅暂准 absent `delete_complete` 进入后续核验；`:499`～`:515` 仍要求 Article 后像 absent、completed operation 的 ID/article/version/audit 全等、允许的删除审计动作，以及 orphan source=Reference.SourceID。无需伪造存在前像，原接口缺口关闭。 |
| 无来源/非删除拒绝 | `:517` 对非 orphan 仍强制真实存在的 author 前像；ordinary/violating absent 即使填写 public author 字段也不放行。publish/transfer absent 仍由 `:406` 拒绝。错误来源、无来源、错 operation、当前 Article 尚存在也不能清理。 |
| 当前 tx 与来源信任 | `:373` 的 exact tx/source/reference 检查未变；Identity 先锁定并核 deleted 和标识生命周期，`:357` 必须调用构造时固定 verifier，失败保留 cause。新分支没有把 public DTO 或“Exists=false”本身当清理授权。真实来源/absence/tombstone 仍须后续 bound provider 证明。 |
| 审计和计数门禁 | `:435`～`:440` 核当前 trace、Audit ID/Article/成功结果与非负计数；operation.audit_event_id 也须匹配。`:364` 在核完 facts 后，任一 active/pending 非零或来源已清即 unchanged；不会提前省略核验。 |
| 释放与重入 | `article_disposition_test.go:427` 覆盖 orphan.delete / deletion.replay 两种审计成功、同 tx 再次核验但不重复三次写入/释放审计；`:445` 覆盖 active/pending 残留与成对已清状态，均核 verifier 调用且零写入。 |
| 拒绝及 caller rollback | `:472` 的拒绝矩阵覆盖共同 audit/counter 错误、operation/source 错误、非 orphan 无前像、跨 tx、错绑定、非 deleted、verifier 拒绝与 publish absent；新增 transfer absent 另测。`:513` 对清 User、删 Identifier、追加审计三处故障分别确认错误链与 caller rollback，未转移 commit/rollback 所有权。 |

新增风险判断：本轮放宽只作用于 absent delete 的初步门禁，最终授权仍由可信完成事实与 orphan operation.source 约束，没有扩大 ordinary/violating、publish/transfer 的资格。无新增必须由 Lead 决定的接口/架构变更，也无非阻断整改要求。

## 验证与未验边界

- Reviewer 现场：`go test ./internal/identity -run '^TestArticleDisposition' -count=1` **PASS**；两文件 `gofmt -l` 无输出；新双哈希和 R01 哈希核对一致。
- Lead 提供：修复后专项 **26 顶层 / 168 通过节点（含子例）PASS**；修改后 Manifest 复跑尚在回收，本报告不把它记成已经通过。作者提供四新增用例 RED、随后 unit/race/identity 回归/vet/gofmt PASS；未冒充 Reviewer 自行重跑。
- 本轮未新增测试或运行全仓/race/vet/Manifest，未连接 PG。模拟 tx/verifier 的绿色用例只证明局部分支与调用合同，不能证明真实锁、同快照 SQL、provider 来源、tombstone/审计实际存在、root 注入或 PG 原子性。

后续②③仍须用固定可信 factory/bound verifier 捕获 Article 私有、当前 tx 的真实引用并核验/映射，不能仅重新包装 public DTO。Lead 继续回收同修订 Manifest 和后续切片证据；本局部 PASS 不自动关闭完整任务或代替人的接纳。

若后续修改来源/tx/operation/审计门禁或引入不同完成证明，按新固定修订复核受影响项。报告完成后停止写入，交回当前 Lead。
