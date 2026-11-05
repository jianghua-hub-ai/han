# article-management-S6.1 定点代码复审 r02

- Reviewer：原非作者 Reviewer add-reviewer
- 实现作者：Hooke `01a16893-be5d-7632-8afa-adf77210a91c`
- 日期：2026-10-23
- objectType：code
- 任务：`article-management-S6.1`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/article-management/review-code-s6.1-r01.md`
- 原报告 SHA256：`2A8AE017EF0C24DC7F7BB588770CF35A5DF9E2F305651CC0530F840B022771E1`
- 复审对象：`ARTICLE-S6.1-R01-F001`
- 复审结论：**PASS**

## 1. 固定对象与开始哈希

开始复审时，下列哈希均与交接值一致：

| 文件 | SHA256 |
|---|---|
| `internal/article/article_port.go` | `215A224CF416D81FA217F7524EDEEA671A25AC1110BA0147CC1D1BD6315CE044` |
| `internal/article/command_service.go` | `880A5960D0B7451567EBBFDCD9CA176513C53C5DED57FD044DB78949469289D7` |
| `internal/article/orphan_canonical_test.go` | `9F7CFD3424450D74C30E1F039D4E0154C20BA03B715EE9539DBDDF82678D5D70` |
| r01 报告 | `2A8AE017EF0C24DC7F7BB588770CF35A5DF9E2F305651CC0530F840B022771E1` |

工作树仍包含其他工作者的既有修改和未跟踪文件。本次未回滚、覆盖、格式化或修改
任何代码、Tasks、Plan 及他人文件；唯一写入为本复审报告。

## 2. 复审依据与范围

本次沿用 r01 已完整读取的 Reviewer Role、`add-code-review`、code-review-checklist、
abstraction-principles、coding/no-hardcode、安全、observability/tracing 规则与
go-service profile，并重新读取：

- r01 报告及 `ARTICLE-S6.1-R01-F001` 的全部关闭条件；
- 三份固定修订；
- `Article` 模型、Article migration 状态约束、`audit.Store` 和 S1.3 Create
  构造/回归测试等必要直接定义与消费者。

### Round 1：原 finding 修订

逐行核对新增的 production `CanonicalOrphanRepository`、SQL、参数映射、行扫描、
条件更新、rows-affected 检查、接口断言和 adapter tests；同时复核未变化的
`CommandService.OrphanArticlesCanonical` 如何把 caller transaction 原样传给
repository 与 auditor。

### Round 2：有界消费者与相关回归

有界核对：

- migration 中 UUID、正交状态、held/source、violation/category 和
  deletion state 约束；
- `audit.Store.Append` 的 caller-owned transaction 接缝；
- S1.3 `NewCommandService` / Create 构造兼容；
- 后继 identity composition 边界和 S9.2 真实 PostgreSQL 专项分工。

未扩大为全仓泛审，也未启动真实 PostgreSQL、identity 编排或 E2E。

## 3. 原 finding 处置

### `ARTICLE-S6.1-R01-F001` — **CLOSED**

- 原严重度：HIGH
- 修订位置：
  - `internal/article/article_port.go:38-169`
  - `internal/article/article_port.go:176`
  - `internal/article/orphan_canonical_test.go:27-174`
  - `internal/article/orphan_canonical_test.go:262-280`
- 相关未变 service：
  - `internal/article/command_service.go:132-143`
  - `internal/article/command_service.go:191-204`

逐项核对结果：

1. **production adapter 与可组合性：CLOSED**
   - 新增导出的 `CanonicalOrphanRepository`；
   - `article_port.go:41` 提供编译期
     `OrphanArticlesCanonicalRepository` 接口断言；
   - `article_port.go:176` 同时确认 `audit.Store` 满足 auditor 接口；
   - 后继 composition 可把 production repository、`audit.Store`、ID generator
     和 clock 注入现有 `Orphaning` 依赖，不再要求 identity 直接写 Article。

2. **锁定 SQL、选择谓词与参数：CLOSED**
   - `article_port.go:43-63` 从 `articles` 读取完整扫描顺序；
   - 谓词只选择 `active|deletion_pending` 且
     `author_id=$1 OR orphaned_from_user_id=$1` 的关联行；
   - `ORDER BY id FOR UPDATE` 提供确定性锁顺序和 caller transaction 行锁；
   - `article_port.go:91` 通过传入的同一个 `pgx.Tx.Query` 执行，参数仅为
     `deletedUserID`；
   - adapter test 检查 SQL 关键谓词、`FOR UPDATE`、参数和无 commit/rollback。

3. **条件 UPDATE、状态和参数映射：CLOSED**
   - `article_port.go:65-77` 只写
     `author_id=NULL`、held、deleted-user source、`version+1` 和 `updated_at`；
   - WHERE 同时约束 article ID、`deletion_state=active`、当前 author 为 deleted
     user 和 expected version；
   - `article_port.go:129-138` 的六个参数与 SQL 占位符一致；
   - adapter test 明确核对 SQL 与参数值。

4. **违规字段保留：CLOSED**
   - UPDATE 不包含 `violation_state` 或 `violation_category`；
   - adapter test 对二者实施 deny assertion；
   - 原 service mixed matrix 继续验证 violating/category 在无归属化后不变。

5. **RowsAffected 与陈旧/异常写入：CLOSED**
   - `article_port.go:142-147` 要求且只接受 `RowsAffected()==1`；
   - adapter tests 覆盖 `UPDATE 0` 和防御性的 `UPDATE 2`，两者均返回错误且
     repository 不关闭 caller transaction。

6. **同一 transaction 与 Audit 原子接缝：CLOSED**
   - production repository 仅调用传入 `tx.Query/Exec`，没有 begin、commit 或
     rollback；
   - 未变化的 service 把同一 `tx` 依次传给 lock、conditional update 和
     auditor append；
   - 原 service tests 继续断言 repository/auditor 收到同一 transaction，且
     service 不 commit/rollback；
   - 因此代码接缝允许外层在任何 update/audit 失败时统一回滚。真实数据库回滚
     效果仍由 S9.2 验证，不影响本 finding 的代码闭合。

7. **adapter tests 与 disposed fixture：CLOSED**
   - 新增生产 lock/update/rows-affected 定向测试；
   - r01 指出的匿名公开、已转移 stub fixture 已清除旧 source，符合真实完成状态；
   - service 的零/多/违规/pending/幂等、审计、失败链和 Create 构造回归继续通过。

结论：r01 所指出的“只有测试 stub、没有 article-owned production persistence”
已消除；F001 的全部代码级复审条件均已满足。

## 4. 新 findings

未发现本次三文件修订引入新的 CRITICAL、HIGH、MEDIUM 或 LOW finding。

production adapter 尚未接入 identity composition 是后继 `identity-auth-S9.1`
职责；真实 PostgreSQL 锁、并发和跨模块回滚是已登记的 `identity-auth-S9.2`
专项，不作为本次已修复代码对象的新缺陷。

## 5. Reviewer 实际验证

```text
go test ./internal/article -run "TestOrphanArticlesCanonicalRepository|TestOrphanArticlesCanonical|TestCreateArticle" -count=1 -timeout=120s
go test -race ./internal/article -run "TestOrphanArticlesCanonicalRepository|TestOrphanArticlesCanonical|TestCreateArticle" -count=1 -timeout=180s
go test ./internal/article -count=1 -timeout=180s
go vet ./internal/article
gofmt -l internal/article/article_port.go internal/article/command_service.go internal/article/orphan_canonical_test.go
```

结果：

- 定向 adapter/service/Create 测试：PASS；
- 定向 race：PASS；
- `internal/article` 包测试：PASS；
- package vet：PASS；
- 三份固定对象 `gofmt -l` 无输出；
- 测试后固定对象和 r01 报告哈希保持匹配。

## 6. 未验证风险与限制

- 未运行真实 PostgreSQL，因此未实证 SQL 在真实数据库中的锁等待、执行计划、
  UUID 编解码、并发竞争和事务 rollback 后状态；
- 未运行 identity production composition，后继接线仍须实际注入
  `CanonicalOrphanRepository` 与 `audit.Store`；
- 未运行跨模块 canonical 用户删除、浏览器、部署或 E2E；
- 未执行全仓全部 Lead gate，也未对三份固定对象之外的工作树改动作结论；
- 当前 PASS 只覆盖固定修订、原 F001 和与其直接相关的回归。

## 7. Gate

| Finding | 原严重度 | 状态 |
|---|---|---|
| `ARTICLE-S6.1-R01-F001` | HIGH | **CLOSED** |

- 新增 findings：0
- 未关闭阻断 findings：0
- **Code Review Gate：PASS**

本 PASS 不关闭任务，不修改 Tasks/Plan，不授权 commit/push、真实 E2E 或启动后继
任务。

## 8. 结束哈希

报告写入后再次核对，三份固定代码对象保持：

| 文件 | SHA256 |
|---|---|
| `internal/article/article_port.go` | `215A224CF416D81FA217F7524EDEEA671A25AC1110BA0147CC1D1BD6315CE044` |
| `internal/article/command_service.go` | `880A5960D0B7451567EBBFDCD9CA176513C53C5DED57FD044DB78949469289D7` |
| `internal/article/orphan_canonical_test.go` | `9F7CFD3424450D74C30E1F039D4E0154C20BA03B715EE9539DBDDF82678D5D70` |
