# article-management-S6.1 独立代码评审 r01

- Reviewer：add-reviewer（非作者 Reviewer）
- 实现作者：Hooke `01a16893-be5d-7632-8afa-adf77210a91c`
- 日期：2026-10-23
- objectType：code
- 任务：`article-management-S6.1`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 对象状态：固定未提交代码；`internal` 未跟踪状态不作为拒审理由
- 评审结论：**FAIL**

## 1. 固定对象与起始哈希

开始评审时，下列 SHA256 均与交接值一致：

| 文件 | SHA256 |
|---|---|
| `internal/article/article_port.go` | `E7C34FE51252AFD1BF42968928ACB2213408E04604CB1D66A4739F53A9305FA0` |
| `internal/article/command_service.go` | `880A5960D0B7451567EBBFDCD9CA176513C53C5DED57FD044DB78949469289D7` |
| `internal/article/orphan_canonical_test.go` | `265129B0BC4188D98A63A828E3B05E8CDC50E4CCAC0A737036CA020802FB52AB` |

`internal/article/command_service.go` 的 S1.3 基线 SHA256 为
`32A19DE3DB590E92D360E8A22A97B130738A5B03C0AF5B302D4E7452B3C105CA`。
由于当前对象未跟踪且仓库中没有该基线内容快照，本轮以该固定基线哈希、S1.3
复审证据和当前完整对象审查 S6.1 增量，不伪称取得了可逐字节重放的 Git diff。

工作树存在其他工作者的既有修改和未跟踪文件。本次未回滚、覆盖、格式化或修改
任何代码及他人文件；唯一写入为本报告。

## 2. 依据

完整读取并应用了 Reviewer Role、`add-code-review`、code-review-checklist、
abstraction-principles、coding-standards、no-hardcode-policy、产品安全红线，
以及 go-service profile 中与本对象匹配的 coding、security、no-hardcode、
observability、tracing 规则。

业务与验收依据：

- `.add/project/knowledge/tasks/tasks.md:1094-1110`
- article design r09 `§3.4.3`，尤其
  `.add/project/knowledge/design-workspace/article-management/design.md:1691-1714`
- test-plan r04 A-ORP-01/02 及责任分界：
  `.add/project/knowledge/design-workspace/article-management/test-plan.md:159-160,259,305,308`
- `.add/project/knowledge/decisions/content.md:17,56-67,73-75`
- `.spec/wip/article-management/review-design-r09-am-q1-r03.md` 的设计复审 PASS
- `.spec/wip/article-management/review-code-s1.3-r02.md` 的 S1.3 基线复审证据

## 3. 两轮实际检查范围

### Round 1：固定对象 / 增量直接审查

逐行检查三份固定对象，重点包括：

- `pending_orphan_count` 对 active、已完成 orphan 和 `deletion_pending` 的计数；
- 普通/违规文章的 `author_id=NULL`、held、source、version+1，以及
  violation state/category 保留；
- 已无归属、匿名公开、已转移、永久删除和重复调用不回退；
- caller-owned transaction 不 commit/rollback；
- 每个新结果的 AuditEvent action、actor、target、outcome、trace、metadata 和低敏性；
- ID、时钟、repository、audit 失败的短路顺序；
- `CommandDependencies.Orphaning` 对 S1.3 构造调用的兼容性；
- 测试 stub 是否只证明自身状态转换，而未证明生产持久化合同。

当前 service 层的状态判断、结果计数、调用同一 `pgx.Tx`、不拥有事务关闭动作、
最小成功审计以及既有构造兼容未发现独立缺陷。

### Round 2：有界消费者 / 契约审查

有界读取了下列直接定义与消费者：

- `internal/article/model.go` 的 Article 正交状态；
- `internal/audit/store.go` 的 caller-owned transaction AuditEvent 接缝；
- `internal/article/repository.go` 的生产 `Repository` 能力；
- `internal/article/create_test.go` 的 S1.3 构造调用；
- `migrations/0003_article.sql` 的 Article 状态约束和索引；
- Tasks/Plan 中 S6.1、后继 `identity-auth-S9.1/S9.2` 及 repository 文件所有权；
- design/test-plan 中 article-owned service seam、PostgreSQL 专项和跨模块责任边界。

当前仓库没有 `OrphanArticlesCanonical` 的实际消费者；后继 identity 编排尚未实现。
该事实不消除本卡应交付可组合 article-owned 端口实现的要求。

## 4. Findings

### HIGH — `ARTICLE-S6.1-R01-F001`：canonical 端口没有生产持久化实现，无法与真实 Article repository 组合

- **位置**：
  - `internal/article/article_port.go:27-34`
  - `internal/article/command_service.go:132-143,191-204`
  - 直接消费者证据：`internal/article/repository.go:12-29`
  - 测试证据：`internal/article/orphan_canonical_test.go:434-480`
- **违反依据**：
  - Tasks S6.1 的目标是“实现 canonical 无归属化端口”，关闭条件要求实际
    article-owned 状态与 Audit 合同；
  - design r09 `§3.1` 要求跨模块只消费 article service 端口，`§3.4.3`
    要求 article 锁行并写 NULL/held/source/version+1；
  - ADR-0010 要求 Article 权威字段只能由 article-management 写入；
  - test-plan 将真实跨模块 PostgreSQL 专项后移至 S9.2，但没有把本卡的
    article-owned repository 实现后移或授权 identity 改写 Article。
- **触发场景**：后继 `identity-auth-S9.1` 尝试用生产 `article.Repository{}`
  构造带 `Orphaning` 的 `CommandService` 并在用户删除事务中调用
  `OrphanArticlesCanonical`。
- **证据**：
  1. `article_port.go` 只声明了
     `LockCanonicalOrphanCandidates` / `SetCanonicalOrphan` 接口；
  2. 生产 `article.Repository` 没有实现这两个方法，仓库中也没有其他生产类型实现；
  3. 因而 `Repository{}` 不能赋给 `OrphanArticlesCanonicalRepository`；
  4. 若 production 继续省略可选 `Orphaning`，调用会在
     `command_service.go:128-131` 直接失败；
  5. 全部成功测试仅由内存 stub 自行实现锁定结果和状态写入，未执行任何生产 SQL、
     行锁、条件更新、rows-affected 或真实回滚路径。
- **影响**：
  - 当前交付只能证明 service 对测试 stub 的编排，不能在 PostgreSQL 上执行
    canonical 无归属化；
  - 后继用户删除无法按设计把 User、Article 和 Audit 放在同一真实事务中；
  - 若为绕过不可组合接缝而由 identity 直接写 Article，会破坏 article-owned
    状态所有权和模块边界；
  - A-ORP-01/02 的真实锁谓词、version+1、违规保留、并发条件和失败回滚均未由
    本实现建立。
- **最小建议**：在 article-owned 范围提供生产 repository 实现：
  1. 使用 caller `pgx.Tx` 选择并锁定与 deleted user 关联的 active/pending 行；
  2. 对 active owned 行执行带预期 version/状态条件的
     NULL/held/source/version+1/updated_at 更新；
  3. 校验 `RowsAffected()==1`，不修改 violation state/category；
  4. 保持 Audit append 使用同一 caller transaction；
  5. 增加 `var _ OrphanArticlesCanonicalRepository = Repository{}`
     一类编译期断言及 SQL/adapter 定向测试。若所需文件不在当前作者授权范围，
     由 Lead 重新分配最小文件所有权，不能由 Reviewer 越界修复。
- **验证条件**：
  - 生产 `Repository` 可通过编译期接口断言；
  - adapter 测试覆盖 active/held/pending/已处置选择谓词、`FOR UPDATE`、
    条件更新、version+1、违规字段不变和 rows-affected 失败；
  - service + production adapter + `audit.Store` 能接收同一个 caller `pgx.Tx`；
  - 本地定向/race 回归通过；真实 PostgreSQL 与跨模块回滚仍由已登记的 S9.2
    专项最终验证。

## 5. 待决定与非阻断建议

### 待决定

无新的产品或架构方向待决定。F001 是已确认设计下的实现完整性缺口，不需要重开
canonical 合同。

### 非阻断建议

F001 修复时，应把“匿名公开/已转移”作为真实已退出选择集合的 repository
场景验证。当前 `orphan_canonical_test.go:118-127` 的两条 stub 行仍携带旧
`orphaned_from_user_id`，不符合现有数据库约束和完成处置后清 source 的合同；
它们只能证明 service 不改写这两种构造出来的内存行，不能替代生产选择谓词测试。
此项不另立 finding，因为根因和关闭证据均包含在 F001。

## 6. Reviewer 实际验证

```text
go test ./internal/article -run "TestOrphanArticlesCanonical" -count=1 -timeout=60s
go test -race ./internal/article -run "TestOrphanArticlesCanonical" -count=1 -timeout=90s
go test ./internal/article -run "TestCreateArticle|TestOrphanArticlesCanonical" -count=1 -timeout=120s
go test ./internal/article -count=1 -timeout=120s
go vet ./internal/article
gofmt -l internal/article/article_port.go internal/article/command_service.go internal/article/orphan_canonical_test.go
```

结果：

- 上述 Go 测试、race 与 vet 全部 PASS；
- `gofmt -l` 无输出；
- 定向搜索确认仓库中没有生产
  `LockCanonicalOrphanCandidates` / `SetCanonicalOrphan` 实现；
- 分支、HEAD 和三份固定对象起始哈希匹配。

Lead 提供的定向、article 包、race、`go vet ./...`、`go test ./...`、三文件
gofmt 和 no-index diff-check PASS 仅记录为上游证据，不冒充 Reviewer 重跑结果。

## 7. 未验证风险与限制

- 未运行真实 PostgreSQL；未验证真实行锁、条件 UPDATE、并发或 rollback；
- 未实现的 production adapter 使本轮无法验证 Article 状态变化与 Audit append
  在真实数据库事务中的原子性；
- 未实现 identity composition，未运行跨模块 canonical 删除；
- 未运行浏览器、部署或 E2E；
- 没有执行 Lead 声明的全仓全部命令，也未扩大为全仓泛审；
- 当前对象未提交且基线内容不是可检出的 Git object，Round 1 的增量判断采用固定
  基线哈希、既有 S1.3 复审证据和当前对象全文；
- 工作树包含大量他人既有改动，本报告未对其作处置或结论。

## 8. Gate 与复审条件

| Finding | 严重度 | 状态 |
|---|---|---|
| `ARTICLE-S6.1-R01-F001` | HIGH | **OPEN** |

- 未关闭阻断 finding：1
- **Code Review Gate：FAIL**

复审须固定包含 F001 修订的代码对象及哈希，重点核对 production repository
接口实现、SQL 锁定/更新语义、同一 caller transaction、adapter 测试及现有
service/race 回归。真实 PostgreSQL 跨模块专项仍按 S9.2 边界执行，不要求
Reviewer 在本卡擅自启动。

本 FAIL 不授权 Reviewer 修改代码、Tasks/Plan/design/ADR/manifest，不关闭任务，
也不授权启动 B5C、部署或 E2E。

## 9. 结束哈希

报告写入后再次核对三份固定对象；最终值记录如下。

| 文件 | SHA256 |
|---|---|
| `internal/article/article_port.go` | `E7C34FE51252AFD1BF42968928ACB2213408E04604CB1D66A4739F53A9305FA0` |
| `internal/article/command_service.go` | `880A5960D0B7451567EBBFDCD9CA176513C53C5DED57FD044DB78949469289D7` |
| `internal/article/orphan_canonical_test.go` | `265129B0BC4188D98A63A828E3B05E8CDC50E4CCAC0A737036CA020802FB52AB` |
