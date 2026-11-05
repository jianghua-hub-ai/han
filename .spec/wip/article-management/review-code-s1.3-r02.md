# article-management-S1.3 定点代码复审 r02

- Reviewer：原非作者 Reviewer Meitner
- 实现作者：Hooke `01a16893-be5d-7632-8afa-adf77210a91c`
- 日期：2026-10-23
- objectType：code
- 任务：`article-management-S1.3`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/article-management/review-code-s1.3-r01.md`
- 原报告 SHA256：`AB499CF9AF78485D8D5920509B1BC4B110536ABA841CEA09336E3B90943D5448`
- 复审结论：**PASS**

## 1. 固定对象与起始哈希

开始复审时，下列 SHA256 均与交接值一致：

| 文件 | SHA256 |
|---|---|
| `internal/article/command_service.go` | `32A19DE3DB590E92D360E8A22A97B130738A5B03C0AF5B302D4E7452B3C105CA` |
| `internal/article/handler.go` | `58E0A6863DBF8208BA13F4F0A3B9A8FCF2B8D784B4B4C0314191F7BE0B82008E` |
| `internal/article/create_test.go` | `90F0A53153A42CDAE4A8669606259C033089FDCB54E8010F4BC3D6FDE4D0535F` |

工作树存在其他工作者的既有修改和未跟踪文件。本次未回滚、覆盖、格式化或修改任何代码及他人文件；唯一写入为本报告。

## 2. 依据与已确认方向

本次引用 r01 的两项 MEDIUM finding：

- `ARTICLE-S1.3-REV-R01-F001`
- `ARTICLE-S1.3-REV-R01-F002`

复审依据包括 Reviewer Role、`add-code-review`、代码评审清单、抽象/编码/no-hardcode 规则、Go profile 的 coding/no-hardcode/security/observability/tracing 规则，以及 Tasks S1.3、Plan B5A/B5B 与 `S1.3 → S6.1`、article boundary r13、architecture r24、design r09、test-plan r04、design Review PASS。

用户已采用以下处理方向，本次不重开设计：

1. COMMIT 返回错误后按 transaction closed、commit outcome unknown 处理；
2. 201 响应开始写出后的 write error 不再尝试第二个 HTTP 响应，只在责任边界观测一次。

## 3. 两轮实际复审范围

### Round 1：原 findings 修订与直接回归

逐行复核了三份固定代码对象中与 F001/F002 相关的实现和测试：

- commit outcome unknown 标记、错误链、公共错误映射、defer/rollback/commit 状态；
- pgx closed 语义的 transaction stub 和调用次数断言；
- 201 body 部分写失败后的控制流；
- response-write logger 的构造、字段、trace 关联与低敏约束；
- partial `ResponseWriter` 对 header/body 唯一写尝试和无第二 error envelope 的证明；
- 正常创建路径及 S1.3 既有合同的定向回归。

### Round 2：有界消费者与契约复核

沿直接链路检查了：

- pgx v5.11.0 的真实 `Commit`/`Rollback` closed 语义；
- `platform.LoggerWithTrace`；
- article `NewHandler` 的当前调用点；
- `cmd/han-server` 的统一 `*slog.Logger` 与 composition root 接缝；
- r01 所引用的上游 Tasks、Plan、boundary、architecture、design、test-plan 和 design Review。

当前 production 尚未注册 article route；article `NewHandler` 的现有消费者仅为 S1.3 测试。`cmd/han-server/main.go:245,304-313` 已把统一 logger 提供给 composition root，后续可消费导出的 `NewSlogArticleResponseWriteLogger`。因此当前接缝可组合，且不要求 S1.3 越权实现后续 production composition 或 S6.1。

## 4. 原 findings 处置

### `ARTICLE-S1.3-REV-R01-F001` — CLOSED

- 原严重度：MEDIUM
- 修订位置：
  - `internal/article/command_service.go:23-25`
  - `internal/article/command_service.go:157-162`
  - `internal/article/create_test.go:211-296`
  - `internal/article/create_test.go:570-598`
- 触发场景：COMMIT 已发出，但连接丢失、超时或协议错误导致调用方不能确认提交结果。
- 核对结果：
  1. 新增稳定内部标记 `errCreateArticleCommitOutcomeUnknown`；
  2. `Commit` 返回错误后将 transaction 视为 closed，不再调用 `Rollback`；
  3. 公共错误仍映射为 `platform.database_unavailable`；
  4. `errors.Join` 同时保留 outcome-unknown 标记和原始 commit cause；
  5. stub 模拟 pgx：`Commit` 后 closed，后续 `Rollback` 返回 `pgx.ErrTxClosed`；
  6. 测试明确断言 commit/rollback 为 `1/0`，并检查 closed、公共错误码和两段 error chain，未再宣称 commit error 必然零持久化。
- 影响判断：r01 指出的错误事务模型和误导性零持久化证明已消除；实现与已采用的 commit-outcome-unknown 合同一致。
- 验证：相关定向测试、全部 `TestCreateArticle` 和 race 定向测试均通过。
- 剩余限制：未以真实 PostgreSQL 注入 COMMIT 期间断连来观察实际行状态；这是报告限制，不再阻断本 finding 的代码语义修复。

### `ARTICLE-S1.3-REV-R01-F002` — CLOSED

- 原严重度：MEDIUM
- 修订位置：
  - `internal/article/handler.go:91-150`
  - `internal/article/handler.go:159-172`
  - `internal/article/handler.go:221-229`
  - `internal/article/create_test.go:298-368`
  - `internal/article/create_test.go:651-675`
- 触发场景：201 header 已提交后，JSON body 只写出一部分即返回错误。
- 核对结果：
  1. body write 失败后不再调用 `WriteArticleError`，不会追加第二个状态或 error envelope；
  2. partial writer 证明仅一次 `WriteHeader(201)` 和一次 body `Write`，同时模拟已写出半个 body；
  3. write error 仅进入一次 response-write logger；
  4.日志包含 `trace_id`、固定 endpoint、201 status、稳定 event/reason bucket/error code 和原始 write cause；
  5. 测试断言日志不包含 title/body；
  6. `NewHandler` 对 logger fail closed；导出的 slog adapter 可由现有 composition logger 消费，没有引入不可构造的生产契约，也未越界注册后续 article route。
- 影响判断：r01 指出的二次响应写入和错误吞没已消除；失败被单次交给可观测边界。
- 验证：partial writer 定向测试、全部 `TestCreateArticle`、构造编译检查和 race 定向测试均通过。
- 剩余限制：未在真实反向代理或客户端断连环境验证 partial write 的网络表现。

## 5. 新 findings

未发现由两项修订引入的新增 CRITICAL、HIGH、MEDIUM 或 LOW finding。

本结论仅表示在固定修订和声明的有界消费者范围内，F001/F002 已关闭；不代表对全仓或后续 article production composition 的泛化评审。

## 6. Reviewer 实际验证

```text
go test ./internal/article -run '^(TestCreateArticle_DatabaseFailuresRespectTransactionOutcome|TestCreateArticle_CommitFailurePreservesUnknownOutcome|TestCreateArticle_ResponseWriteFailureIsObservedWithoutSecondResponse)$' -count=3
go test ./internal/article -run "TestCreateArticle" -count=1
go test ./internal/article ./cmd/han-server -run '^$' -count=1
go test -race ./internal/article -run '^(TestCreateArticle_DatabaseFailuresRespectTransactionOutcome|TestCreateArticle_CommitFailurePreservesUnknownOutcome|TestCreateArticle_ResponseWriteFailureIsObservedWithoutSecondResponse)$' -count=1
gofmt -l internal/article/command_service.go internal/article/handler.go internal/article/create_test.go
```

结果：

- 上述 Go 测试全部 PASS；
- `gofmt -l` 无输出；
- 分支、HEAD、r01 报告哈希和三份固定对象起始哈希均匹配。

Lead 提供的 article/identity/server 定向检查、`go mod verify`、`go vet ./...`、`go build ./...`、contract、race、全量测试和 scoped diff-check PASS 仅记录为上游证据，不冒充 Reviewer 重跑结果。

## 7. 未验证风险与限制

- 未运行真实 PostgreSQL，尤其未做 COMMIT 连接中断/结果未知故障注入；
- 未运行浏览器、部署或真实 E2E；
- 未验证真实代理/网络对 partial response write 的具体表现；
- production article route 尚未装配；该后续 composition 不在 S1.3 本次定点复审范围；
- 未扩大为全仓泛审，也未重新执行 Lead 声明的全部检查；
- 工作树包含大量他人既有改动，本报告未对其作处置或结论。

## 8. Gate

| Finding | 状态 |
|---|---|
| `ARTICLE-S1.3-REV-R01-F001` | **CLOSED** |
| `ARTICLE-S1.3-REV-R01-F002` | **CLOSED** |

- 新增 findings：0
- **Code Review Gate：PASS**

本 PASS 不关闭任务，不授权修改 Tasks/Plan/design/manifest，不授权启动 S6.1、部署或 E2E。

## 9. 结束哈希

报告写入后再次核对，三份固定对象保持：

| 文件 | SHA256 |
|---|---|
| `internal/article/command_service.go` | `32A19DE3DB590E92D360E8A22A97B130738A5B03C0AF5B302D4E7452B3C105CA` |
| `internal/article/handler.go` | `58E0A6863DBF8208BA13F4F0A3B9A8FCF2B8D784B4B4C0314191F7BE0B82008E` |
| `internal/article/create_test.go` | `90F0A53153A42CDAE4A8669606259C033089FDCB54E8010F4BC3D6FDE4D0535F` |
