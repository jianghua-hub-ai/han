# article-management-S2.1 定点代码复审 R02

- 评审日期：2026-10-25
- Reviewer：Faraday，非作者独立 Reviewer
- 替代原因：R01 未记录可恢复的原 Reviewer Agent ID，由 Lead 指定 Faraday 接续；本轮四文件作者为 Hegel，Faraday 不是作者
- objectType：`code`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/article-management/review-code-s2.1-r01.md`
- 复审目标：`ARTICLE-S2.1-REV-R01-F001`、`ARTICLE-S2.1-REV-R01-F002` 及修复新增风险
- **Code Review Gate：PASS**

## 1. 固定对象与上游依据

评审开始时逐项核对通过；落报告前再次核对，结果见第 8 节。

| 对象 | git blob | SHA-256 |
|---|---|---|
| `internal/article/query_service.go` | `58900d7a9c0df745b769decdb41c23b5ad5b7103` | `539795E92A4F4276887C9ED3BCD9EEA08C51B1523271CB671D65D2B30B80C777` |
| `internal/article/query_test.go` | `4cf8af106ecb62515754919ba6d2f11fcd49d874` | `DF40F63117A0C8F7569A255EABBF9DD95F81CA86FBAE17F88EE93A8F78A0C53E` |
| `cmd/han-server/main.go` | `41f65ce7acfe38581bbaf48a131fc959c3086283` | `841CCA74E3D4CAC1C07DE7217B332B0CA6451C8C3118E106DCBFB3D79F4E680E` |
| `cmd/han-server/main_test.go` | `fe196f2982ec6a9d4977e603f2fe39ff07a92151` | `CFE64848B572701E81608280B61C3392C19479407D0DA8CB02CB6C3A53DC31E4` |

固定依据：

- R01 SHA-256：`36BB65973CDC5998567B57B8C10AED1F6B65E114DEB5560414B19179540EF129`
- Article design revision：`article-management-design-ir01-r12`
- design r12 SHA-256：`CC1710DE628B8DBD43D392E28EAA49CF74AB87C659B004D86D58C44033B16228`
- Design Review R02 SHA-256：`9C558B98AF3A9C57D0FDDE12DB974106C17464C70F670CF71BCEEEA1EAC7F0DC`，Gate PASS

## 2. 两轮实际检查范围

### Round 1：固定修复对象

完整检查四个固定文件，重点核对：

1. 三个 QueryConfig 键必填、无默认、严格 `[1-9][0-9]*`、signed 32-bit 上界与两条交叉约束；
2. QueryConfig 在 DB parse/connect/ping/migrate、composition 和 listener 之前 fail closed，错误只披露键/类别；
3. 共享 pool 构造 query repository、Identity reader、service、handler；
4. 唯一 router 中列表 GET、详情 GET 与既有 DELETE/Identity 路由共存，详情错误方法的 `Allow: GET, DELETE`；
5. repository、Identity 与本地不变量的公共错误分类，以及 HTTP envelope 的 cause 去敏；
6. keyset、隐藏密集扫描、作者状态 fail closed、统一 404、ETag、列表不含 body 等既有行为。

### Round 2：有界直接接缝

检查了：

- `internal/article/handler.go` 的列表/详情 handler、查询参数和公共错误写出；
- `internal/identity/article_reader.go` 及 `article_reader_test.go` 的批量作者状态读取、查询/scan/rows 失败与资源关闭；
- `cmd/han-server/main.go` 的唯一 production composition/root/router、共享 pool 和 Trace/Session/Principal 链；
- R01、design r12、Design Review R02 的直接合同。

未扩展为全仓预审；未修改任何被审对象。

## 3. 原 findings 状态

### CLOSED — `ARTICLE-S2.1-REV-R01-F001`（原 HIGH）

证据：

- `cmd/han-server/main.go:608-610,694-768` 从三个精确环境键构造 QueryConfig；没有代码默认或 fixture 回退。parser 拒绝空值、空白、符号、小数、指数、分隔符、非 ASCII 数字、前导零、零值和超出 `2147483647` 的值。
- `cmd/han-server/main.go:331-378` 先完成 `loadStartupConfig`，随后才调用 DB parser；因此配置失败不会进入 DB parse/connect/ping/migrate。错误仅由键和稳定类别组成。
- `cmd/han-server/main.go:2131-2166,2707-2816` 在唯一 production builder 中以共享 pool 构造 `PostgresArticleQueryRepository`、`ArticleIdentityReader`、`ArticleQueryService` 和 `QueryHandler`，并交给唯一 router。
- `cmd/han-server/main.go:3030-3043` 注册 `GET /api/v1/articles` 与 `GET /api/v1/articles/{id}`；详情与 `DELETE /api/v1/articles/{id}` 共存，错误方法返回 `Allow: GET, DELETE`。
- production wiring 测试证明组件构造、共享 pool、Trace→Session→Principal→handler 顺序、GET/DELETE/Identity 路由共存及 405 合同。
- `2/2/3` 只存在于测试 fixture；AM-Q2 生产数值、容量和 P95 仍为 TBD。

结论：原“真实 Host 无 production composition/router 入口”的触发条件已消除。

### CLOSED — `ARTICLE-S2.1-REV-R01-F002`（原 MEDIUM）

证据：

- `internal/article/query_service.go:145-153,306-333` 将详情 repository 错误和列表 repository 错误统一映射为 `503 platform.database_unavailable`；`pgx.ErrNoRows` 仍映射为统一不可见 404。
- `internal/article/query_service.go:366-381` 将 Identity 作者状态读取错误映射为 `503 platform.dependency_unavailable`。
- 候选数量/排序不变量、投影、cursor/ETag 编码等本地缺陷仍映射为 `500 platform.internal_error`。
- `internal/article/query_test.go:138-207,264-325` 分别覆盖列表和详情的 repository/Identity 503、局部不变量 500，并用 canary 断言响应 envelope 不泄露内部 cause。
- repository 的 Query、Scan、rows.Err 最终都经 repository 方法返回到上述 service boundary；Identity reader 的查询、scan、rows.Err 经同一 Identity dependency boundary fail closed。

结论：原“依赖故障统一错误标为 500”的触发条件已消除。

## 4. 新 findings

**0 个。**

没有发现修复引入第五实现文件、新依赖、migration、公共合同变化或第二 composition root 的证据。

## 5. 待决定与非阻断建议

- 新增待决定：无。
- 非阻断建议：无。
- 既有 AM-Q2 生产分页参数、文章规模、容量与 P95 继续为 TBD；本次 PASS 不批准任何生产数值。

## 6. 独立验证

执行前均仅移除当前子进程环境中的 `HAN_TEST_POSTGRES_DSN`，未读取、输出或持久化其值。

| 命令 | 结果 |
|---|---|
| 固定对象、R01、design r12、Design Review R02 的 blob/SHA-256；branch/HEAD；报告路径不存在 | PASS |
| `go test ./internal/article ./cmd/han-server -run 'Test(ArticleQueryConfig_\|Run_ArticleQueryConfigFailsBeforeDatabaseSideEffects\|ProductionArticleQueryCompositionUsesSharedPoolAndComponents\|ProductionArticleQueryRoutesUseTraceSessionPrincipalAndCoexist\|VisibleScan_\|ListArticles_DependencyFailuresAndLocalDefectsKeepStableBoundaries\|GetArticle_DependencyFailuresKeepStableBoundaries\|GetArticle_VisibleDetailReturnsETagAndInvisibleStatesShare404)$' -count=1` | PASS |
| 上述定向测试加 `-race` | PASS |
| `go test ./internal/article ./cmd/han-server -count=1` | PASS |
| `go test -race ./internal/article ./cmd/han-server -count=1` | PASS |
| `go test ./internal/identity -run '^TestGetAuthorStates_' -count=1` | PASS |
| 上述 Identity 定向测试加 `-race` | PASS |
| `go vet ./internal/article ./cmd/han-server` | PASS |
| `go build -o NUL ./cmd/han-server` | PASS；为遵守不得修改既有 `han-server.exe`，使用无产物安全等价形式 |
| `gofmt -l` 四个固定 Go 文件 | PASS，无输出 |
| `git diff --check --` 四个固定 Go 文件 | PASS |

观察到仓库根目录已有未跟踪 `han-server.exe`，SHA-256 为
`3B49A6ADA7F6E3C637FED0AAA86FA6C66974A852946E02A8C8EC74F14EFF18AC`；
本轮未删除、覆盖或修改该文件。

## 7. 未验证限制

- 未连接真实 PostgreSQL；未验证真实查询计划、索引表现、并发变化或数据库故障注入。
- 未启动真实 Host，未执行部署、浏览器或跨组件 E2E。
- 未执行依赖 `HAN_TEST_POSTGRES_DSN` 的 integration 测试。
- 未验证 AM-Q2 容量/P95、目标规模或 production readiness。
- PASS 仅绑定第 1 节固定四文件及其有界直接接缝，不能外推到共享工作树其他未提交成果。

## 8. 最终结论与漂移条件

- `ARTICLE-S2.1-REV-R01-F001`：**CLOSED**
- `ARTICLE-S2.1-REV-R01-F002`：**CLOSED**
- 新 findings：**0**
- **Code Review Gate：PASS**

该结论以固定四文件、R01、design r12、Design Review R02 在落盘前仍保持第 1 节摘要为条件；若任一摘要漂移，本报告不能用于混合 revision。
