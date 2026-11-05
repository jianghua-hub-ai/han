# article-management-S2.1 独立代码评审 R01

- 日期：2026-10-25
- objectType：`code`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- HEAD 基线：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审性质：非作者独立评审；作者会话已关闭，本评审未联系作者、未代修
- Gate：**CHANGES REQUIRED**

## 1. 固定对象

评审开始、运行验证前和报告落盘前均核对以下 SHA-256；未发现漂移：

| 文件 | SHA-256 |
|---|---|
| `internal/article/query_service.go` | `C6E99DD6806FC51E16B2221BD75B6188C5BFFA3631E44CC0B7F433C4C6A0A11D` |
| `internal/article/handler.go` | `1D1D9684D46D9CC21597D872BD342C5657213AFA2582BCAB0522703D804C8336` |
| `internal/article/query_test.go` | `1498E5C4E3594A06E0F7BD0A9966A5E1D4D0475AB14B61925FB6058C978B5FB1` |

未修改上述对象。共享工作树中 `article-management-S4.2` 的五文件及其他既有改动均未修改、回滚或清理。

## 2. 依据与成功条件

- Tasks SSOT：`.add/project/knowledge/tasks/tasks.md:1168-1185`
- Article design：§2.4.1、§3.3.3，尤其列表/详情接口、分页算法和公共错误注册
- Story 2：`.add/project/knowledge/design-workspace/article-management/stories.md:47-57`
- Test plan：A-READ-01～06
- 项目 code review checklist、abstraction principles、coding/no-hardcode、go-service coding/security 与安全红线

本轮重点检查 `(created_at,id) DESC` keyset、隐藏密集区持续扫描、作者状态批读/fail closed、`limit+1` 与公共 cursor、统一 404、列表/详情投影、ETag、AM-Q2 未决数值以及现役 router/错误合同。

## 3. 两轮实际检查范围

### Round 1：固定三文件直接审查

- 全文检查 `query_service.go`、`handler.go`、`query_test.go`。
- 逐路径核对正常、边界和故障分支。
- 确认：
  - SQL 使用 `(created_at,id) DESC` keyset；
  - 候选按批扫描至 `limit+1` 个可见项或耗尽；
  - 作者状态按批读取，缺失状态隐藏，读取错误 fail closed；
  - 内部扫描 cursor 与公共 cursor 分离，公共 cursor 基于最后返回项；
  - 探测用第 `limit+1` 个可见项未被公共 cursor 消费；
  - 列表响应不含 body，详情返回 body 与强 ETag；
  - 不可见详情统一 `404 article.unavailable`；
  - `QueryConfig` 由注入提供，生产代码未硬编码 AM-Q2 数值。

### Round 2：有界消费者与上游依据审查

实际检查：

- `internal/article` 的 model、policy、cursor、ETag、repository/migration 相关实现和直接测试；
- `internal/identity/article_reader.go` 与直接测试；
- `migrations/0003_article.sql` 的 `(created_at DESC,id DESC)` 索引；
- `cmd/han-server/main.go` 的 Article composition、安全链和唯一 router；
- `cmd/han-server/main_test.go` 的 Article router/production consumer 测试；
- Tasks、Article design、Story 2、A-READ-01～06。

未扩展为全库预读。

## 4. Findings

### HIGH — ARTICLE-S2.1-REV-R01-F001：成员列表和详情没有接入唯一生产 composition/router

**位置**

- `internal/article/handler.go:193-262`
- `cmd/han-server/main.go:2718-2742`
- `cmd/han-server/main.go:2745-2779`
- `cmd/han-server/main.go:2800-2861`
- `cmd/han-server/main_test.go:1659-1766`
- `cmd/han-server/main_test.go:1803-1821`

**违反依据**

- Story 2 要求已认证用户可通过列表和详情读取文章。
- Design 接口总表登记 `GET /api/v1/articles` 与 `GET /api/v1/articles/{id}`。
- Design 要求 Article routes 与 Identity routes 共存于唯一 router，并经过既有安全链。
- 用户指定重点要求 handler 与既有 router 合同兼容。

**证据**

1. 固定对象新增了 `NewQueryHandler`、`ListArticles` 和 `GetArticle`，但生产 composition 仍只通过 `article.NewHandler(...)` 构造 command/deletion handler。
2. 全部生产调用方中没有 `NewPostgresArticleQueryRepository`、`NewArticleQueryService` 或 `NewQueryHandler`。
3. `protectArticleHandlers` 只保护三条 DELETE handler。
4. 唯一 router 只登记三条 Article DELETE 路径；没有成员列表或详情 GET 路由。
5. 现有 router 测试明确把 `GET /api/v1/articles/{id}` 断言为 `405` 且 `Allow: DELETE`。

**触发场景**

- 请求 `GET /api/v1/articles`：落入未知路径，返回 route-not-found 404。
- 请求 `GET /api/v1/articles/{id}`：命中现有 DELETE 路径后返回 405，`Allow: DELETE`。

**影响**

S2.1 实现仅能通过测试内直接调用到达，真实 Host 无法提供 Story 2 的两个成员读取入口；详情 GET 还与现役 DELETE router 的 method 合同冲突。任务不能据此关闭。

**修复建议**

由 Lead 明确 production composition/router 文件所有权后：

1. 使用共享 DB pool 构造 query repository、Identity author reader 与 query service；
2. 从外置部署配置注入 `DefaultLimit`、`MaxLimit`、`CandidateBatchSize`，不得擅填 AM-Q2；
3. 在唯一 router 注册列表与详情 GET，并复用既有 Trace/Session/Principal 安全链；
4. 更新 router/production composition 测试，证明列表 GET、详情 GET 与 DELETE 在同一路径族正确共存。

该修复超出本轮固定三文件，属于所有权/接线方向决定；Reviewer 不代修。

### MEDIUM — ARTICLE-S2.1-REV-R01-F002：数据库和 Identity 依赖故障被统一错误标为 500 internal_error

**位置**

- `internal/article/query_service.go:145-153`
- `internal/article/query_service.go:310-319`
- `internal/article/query_service.go:366-377`
- `internal/article/query_service.go:512-519`
- `internal/article/query_test.go:130-134`

**违反依据**

- Article design 公共错误表：
  - repository boundary：`503 platform.database_unavailable`；
  - identity adapter：`503 platform.dependency_unavailable`；
  - `500 platform.internal_error` 仅属于 top-level boundary。
- 用户指定重点要求兼容既有错误合同。

**证据**

- repository 列表/详情错误和 Identity author-state reader 错误最终都进入 `articleQueryInternalError`。
- `articleQueryInternalError` 无条件产生 `platform.CodePlatformInternalError`。
- 测试把 Identity 故障应为 `platform.internal_error` 写成了回归期望，固化了错误分类。

**触发场景**

- PostgreSQL 查询、扫描或 rows 读取失败；
- Identity 作者状态批读失败。

**影响**

客户端收到不可重试语义不准确的 500，而不是设计规定的可重试 503；错误责任边界、指标/告警分类和故障处置都会失真。可见性仍 fail closed，但公共错误契约不兼容。

**修复建议**

- 在 repository 最终边界把持久化不可用映射为 `platform.database_unavailable`；
- 在 Identity adapter 最终边界把作者状态依赖故障映射为 `platform.dependency_unavailable`；
- 仅把本地不变量破坏、编码失败等真正内部缺陷映射为 `platform.internal_error`；
- 增加列表和详情 HTTP 契约测试，分别断言两个 503 code，且响应不泄露内部 cause。

## 5. 非阻断观察与待决定

- 列表 repository 当前读取完整 `Article`，包括 body；响应 serializer 已证明不输出 body。是否进一步改为最小查询投影涉及 repository 形态和容量权衡，本轮没有把它列为缺陷，也未据此宣称容量通过。
- AM-Q2 的 limit、候选批次和 P95 仍未确认；测试中的 `2/2/3` 是 fixture，不构成生产默认值或容量证据。

## 6. 独立验证

运行前已使 `HAN_TEST_POSTGRES_DSN` 在测试进程环境中不存在；未读取、输出或使用其值。

| 命令 | 结果 |
|---|---|
| `go test ./internal/article -run 'TestListArticles|TestGetArticle|TestVisibleScan' -count=1` | PASS，`0.097s` |
| `go test -race ./internal/article -run 'TestListArticles|TestGetArticle|TestVisibleScan' -count=1` | PASS，`1.125s` |
| `go test ./internal/article -count=1` | PASS，`0.755s` |
| `go test ./cmd/han-server -run 'TestArticleBusinessRouter_AWIRERTR01RTR02RTR04UsesExactDeletePaths|TestProductionBusinessHandler_AWIRERTR03RTR05ArticleSecurityChain' -count=1` | PASS，`0.179s`；该 PASS 同时证实现役消费者仍把 Article 详情 GET 当作 DELETE 路径的错误 method |

共享 S4.2 工作未造成本轮 package 编译不稳定。通过的局部测试不能覆盖或抵消上述生产接线和错误合同 findings。

## 7. 未验证风险

- 未使用真实 PostgreSQL；真实 keyset 查询、索引计划、隐藏密集数据和并发变化留给 S2.2。
- 未启动真实 `han-server` 或执行部署后 E2E。
- 未执行全仓 Manifest 七项或 package-level race；本轮按固定对象和共享工作树限制执行定向普通/race、Article 包普通及有界消费者测试。
- 未验证 AM-Q2 容量/P95；不得据当前测试宣称容量或 production readiness。

## 8. Gate 与复审条件

**Code Review Gate：CHANGES REQUIRED**

复审必须提供：

1. 关闭 `ARTICLE-S2.1-REV-R01-F001` 的 production composition/router 修订及对应固定 SHA；
2. 关闭 `ARTICLE-S2.1-REV-R01-F002` 的错误分类修订及列表/详情 HTTP 回归；
3. GET 列表、GET 详情与既有 DELETE 路径在唯一 router 共存的测试证据；
4. QueryConfig 来自受控外部配置且未硬编码 AM-Q2 的证据；
5. 受影响普通/race/package 测试结果、对象哈希、staged/越界检查；
6. 复审对象不得混入 S4.2 共享五文件的未完成修订。

风险接受、跨文件所有权调整和后续授权由 Lead/用户决定；本独立结论不代替该决定。
