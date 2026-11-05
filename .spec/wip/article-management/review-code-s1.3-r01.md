# article-management-S1.3 独立代码评审 r01

- Reviewer：非作者 Reviewer（实现作者 Hooke `01a16893-be5d-7632-8afa-adf77210a91c`）
- 日期：2026-10-23
- objectType：code
- 任务：`article-management-S1.3`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审结论：**FAIL**

## 1. 固定对象与起始哈希

本次只评审下列三文件。开始时 SHA256 与交接值全部一致：

| 文件 | SHA256 |
|---|---|
| `internal/article/command_service.go` | `2E60274364686C71DFB0B56D103537EF29BB8AB3C5FB590983E1C663269516CD` |
| `internal/article/handler.go` | `60D88BC24541296A9AD744DC242E071E9F3505C2CA8C5A7064F888F53705563A` |
| `internal/article/create_test.go` | `E8A239088FCAB4B7E35213DF44360ED163400ECBE0D414827FA6268E9EB4D7DE` |

工作树存在大量非本 Reviewer 的既有修改和未跟踪文件。本次未回滚、覆盖、格式化或修改被审代码及他人文件；唯一写入为本报告。

## 2. 依据与合同

主要验收依据：

- `.add/project/knowledge/tasks/tasks.md:853-869`，尤其 `:867` 的创建纵切关闭条件；
- `.add/project/knowledge/plans/ir-01-content-community-batches.md:518-525,595-608`，尤其 `S1.3 → S6.1` 串行所有权；
- article boundary r13、architecture r24、article design r09、test-plan r04；
- `.spec/wip/article-management/review-design-r09-am-q1-r03.md` 的 design Review PASS；
- `code-review-checklist.md:13-17` 的异常/事务失败收敛要求；
- Go profile coding、no-hardcode、security、observability、tracing 规则。

已确认创建合同包括：纯文本、不 trim、Unicode 空白和 rune 边界、262144/262145 实际 wire byte 边界、非法 UTF-8 在 JSON 解码前拒绝、严格 JSON、失败零 repository/零持久化、可信 Principal 作者、published/clear/active、201/Location/ETag、成员投影，以及不得提前实现 S6.1。

## 3. 两轮实际检查范围

### Round 1：固定三文件直接审查

完整检查：

1. `internal/article/handler.go` 的实际 N+1 有界读取、UTF-8/JSON 顺序、unknown/duplicate/trailing JSON、错误映射、Location/ETag 和响应写入；
2. `internal/article/command_service.go` 的 Principal 授权、Unicode whitespace、rune 长度、原值保留、Article 默认状态、事务 begin/insert/commit/rollback 和数据库错误；
3. `internal/article/create_test.go` 对 wire 边界、非法 UTF-8、受限身份、客户端 `author_id`、业务边界、repository 调用次数、事务失败和响应投影的证明强度；
4. S1.3 文件范围内是否混入 S6.1。

确认正确的重点：

- 使用 `LimitReader(N+1)` 和实际读取长度，N=262144 可继续、N+1=262145 拒绝；
- 原始 bytes 在 JSON 解码前执行 `utf8.Valid`；
- 共享严格 decoder 拒绝 unknown、duplicate 和 trailing JSON；
- 全 Unicode 空白无效，长度按 rune，合法 title/body 不 trim 且原值持久化/返回；
- 客户端不能选择作者；作者来自可信 Principal；
- 成功对象为 published/clear/active、version=1，并返回 201、Location、强 ETag 和成员 allowlist 投影；
- design r09 明确创建不写 AuditEvent，当前实现未越界增加；
- 未实现 S6.1。

### Round 2：有界消费者、上游合同与真实依赖审查

沿直接链路检查了：

- Article 模型、投影和 repository：`internal/article/model.go`、`policy.go`、`repository.go`；
- 共享 HTTP/JSON/错误边界：`internal/platform/http.go`、`errors.go` 及 HTTP authorizer 相关实现；
- Principal/Access：`internal/session/store.go`；
- Article schema/约束：`migrations/0003_article.sql`；
- pgx v5.11.0 真实事务实现：本机 module cache 的 `github.com/jackc/pgx/v5@v5.11.0/tx.go`；
- 上述 Tasks、Plan、boundary、architecture、design、test-plan 和 design Review。

未扩大为全仓泛审。

## 4. Findings

### [MEDIUM] Commit 失败被测试建模成可成功 rollback，不能证明真实 pgx 的零持久化

- finding ID：`ARTICLE-S1.3-REV-R01-F001`
- 对象/修订：code，固定三文件 SHA256 见 §1
- 位置：
  - `internal/article/command_service.go:153-159`
  - `internal/article/create_test.go:209-240`
  - `internal/article/create_test.go:421-439`
- 违反依据：
  - `code-review-checklist.md:13,17`：异常与事务失败路径必须按真实依赖语义收敛；
  - article design r09 `:1168`：数据库失败不得新增 Article；同时明确网络超时后不得盲目重试；
  - test-plan r04 `A-CREATE-01～06` 要求 PostgreSQL 数据符合性证据，当前仅 stub 计数不能替代。
- 现状：`Commit` 返回错误后，服务调用 `rollbackCreateArticle`。但 pgx v5.11.0 的真实 `dbTx.Commit` 在执行 COMMIT 后、返回任何错误前即把事务标为 closed（`tx.go:201-207`）；后续 `Rollback` 只能返回 `pgx.ErrTxClosed`（`:220-223`），不能撤销已提交或结果未知的 COMMIT。测试 stub 的 `Commit` 返回错误后仍允许 `Rollback` 返回 nil，并以 `rollbackCalls == 1` 判定 commit 失败已回滚，未模拟真实 pgx 生命周期。
- 触发：COMMIT 命令遇到连接重置、超时或协议错误；服务端可能已提交，但客户端未收到明确提交结果。
- 影响：测试名称和断言会把“尝试调用 Rollback”误当成“真实回滚并零持久化”。接口返回 `503 platform.database_unavailable` 时 Article 可能已存在；若调用方未按 design 的核对要求处理，可能重复创建。当前证据不足以关闭 DB 失败的持久化合同。
- 建议：
  1. 将 insert 前/期间失败的可回滚路径与 commit 已发出后的 closed/unknown 路径分开，不再用 rollback 调用次数宣称 commit 失败已回滚；
  2. 测试替身应模拟 pgx：`Commit` 后事务关闭，`Rollback` 返回 `pgx.ErrTxClosed`；
  3. 增加真实 PostgreSQL 或代理故障注入，核对 COMMIT 明确回滚与连接中断/结果未知两类结果，并按 generated Article ID 查询实际行；
  4. 若“任何 COMMIT 传输失败都必须保证零 Article”仍是严格合同，则需交回设计/架构选择幂等键、结果查询或协调方案，不能靠再次 `Rollback` 实现。
- 验证：
  - 新单测证明真实 closed 语义且不把 `Rollback` 调用当作持久化结论；
  - PostgreSQL 集成/故障注入记录 COMMIT 返回值、连接状态和 Article 行快照；
  - 原 Reviewer 按本 finding 复审修复修订及回归。

### [MEDIUM] 成功响应写失败后再次写错误响应，并吞掉最终写错误

- finding ID：`ARTICLE-S1.3-REV-R01-F002`
- 对象/修订：code，固定三文件 SHA256 见 §1
- 位置：
  - `internal/article/handler.go:145-146`
  - 直接消费者证据：`internal/platform/http.go:95-103`
- 违反依据：
  - `code-review-checklist.md:13,17`：失败路径必须可收敛，不能留下半完成响应状态；
  - Go profile `observability-standards.md:42-43`：同一错误只在责任边界处理一次，禁止 `_ = err` 吞掉原始错误；
  - article design r09 `:1168`：创建成功响应和网络超时后的重复风险必须有一致语义。
- 现状：`platform.WriteJSON` 先 `WriteHeader(201)`，再写 body；body 写失败时返回 error。`Handler.Create` 随即调用 `WriteArticleError`，尝试在已提交 201 header 的同一 `ResponseWriter` 上再写错误状态和 envelope，且再次写失败的 error 也被 `_ =` 丢弃。
- 触发：客户端断连、代理中断，或 `ResponseWriter` 在写入全部/部分成功 JSON 时返回错误。
- 影响：第二次 `WriteHeader` 已无法可靠改变状态，可能出现 201 加空/截断/混合 body、superfluous header；Article 已提交但客户端得不到可信结果。两次写错误均无可观察证据，排障和按 design 要求核对创建结果都缺少 trace 关联，盲目重试风险增加。
- 建议：成功响应开始写出后，把 write error 作为终态交给现有 HTTP observation/logger 边界一次性记录，携带 trace_id、endpoint、status 和低敏错误原因；不得再尝试写第二个 HTTP 响应。不要记录 title/body。
- 验证：增加 failing/partial `ResponseWriter` 测试，断言只发生一次响应写尝试、没有第二个错误 envelope，并且 write error 被单次观察且包含 trace 关联；正常 201/Location/ETag 回归保持通过。

## 5. 待决定与非阻断建议

### 待决定

- `ARTICLE-S1.3-REV-R01-F001` 的 COMMIT 结果未知语义需要责任人确认：若沿用 design r09 已写明的“网络超时后通过列表或业务上下文核对”，实现和测试应明确标识 unknown，而不是声称 rollback；若坚持所有此类错误严格零持久化，则属于契约/架构方向调整。

### 非阻断建议

- 无。未将风格偏好或未来 S6.1 工作列为缺陷。

## 6. Reviewer 实际验证

```text
git status --short --branch
git rev-parse HEAD
Get-FileHash -Algorithm SHA256 internal/article/command_service.go,internal/article/handler.go,internal/article/create_test.go
go test ./internal/article -run "TestCreateArticle" -count=1
```

结果：

- 分支/HEAD 与固定对象一致；
- 起始三文件哈希全部匹配；
- 定向测试 PASS：`ok han/internal/article`；
- Lead 已提供 article 定向检查、gofmt、mod verify、vet、build、contract、race、全量测试和 owner diff check PASS，本文仅记录为上游证据，不冒充 Reviewer 重跑。

## 7. 未验证风险与限制

- 未运行真实 PostgreSQL，尤其未做 COMMIT 期间连接中断/结果未知故障注入；
- 未运行浏览器、部署或真实 E2E；
- 未验证生产代理/网络对 partial response write 的实际表现；
- 未扩大检查 S1.3 三文件之外的全仓质量；Round 2 仅限声明的直接消费者和上游依据；
- 当前结论只绑定 §1 三个 SHA256。任一被审文件漂移后必须重新固定并复审。

## 8. Gate 与复审条件

- CRITICAL：0
- HIGH：0
- MEDIUM：2（均 OPEN）
- LOW：0
- **Code Review Gate：FAIL**

复审条件：

1. 原作者在授权范围处理 `ARTICLE-S1.3-REV-R01-F001`、`F002`，或由责任人对 F001 的契约方向作出明确决定；
2. 提供新的固定修订/哈希及针对两项 finding 的回归；
3. F001 提供真实 PostgreSQL/等价故障注入证据，或明确记录尚未验证的 commit-unknown 风险及被批准的处理合同；
4. 由独立 Reviewer 引用本报告 finding ID 复审，不以作者自测替代。

本报告不修改 Tasks、Plan、design、manifest，不关闭任务，也不授权 S6.1、部署或 E2E。

## 9. 结束哈希

报告写入后已再次现场核对，固定对象保持：

| 文件 | SHA256 |
|---|---|
| `internal/article/command_service.go` | `2E60274364686C71DFB0B56D103537EF29BB8AB3C5FB590983E1C663269516CD` |
| `internal/article/handler.go` | `60D88BC24541296A9AD744DC242E071E9F3505C2CA8C5A7064F888F53705563A` |
| `internal/article/create_test.go` | `E8A239088FCAB4B7E35213DF44360ED163400ECBE0D414827FA6268E9EB4D7DE` |
