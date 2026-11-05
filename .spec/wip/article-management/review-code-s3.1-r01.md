# Article S3.1 代码评审（作者自审口径）

- **评审对象**：`article-management-S3.1`「实现作者受控详情与强 ETag 修改」
- **评审范围（精确五文件，唯一写集）**：
  - `internal/article/command_service.go`（修改）
  - `internal/article/query_service.go`（修改）
  - `internal/article/handler.go`（修改）
  - `internal/article/update_test.go`（新增）
  - `internal/article/author_projection_test.go`（新增）
- **设计/测试依据**：article design r14 §3.3.4（US-ARTICLE-03.1）；article test-plan r12 A-ETAG-01～04、A-MOD-01～08；要求 `article-management-S1.5`、`article-management-S2.2` 承接跨模块契约与 PostgreSQL 并发专项
- **评审日期**：2026-10-27
- **评审口径（重要限制）**：本 Host 无委派能力，无独立 Reviewer Role 产出。依据用户 2026-10-27 明确授权「本 Host 可以自审、作者自审口径」，本报告由 Lead 以**作者自审**口径执行，**弱于** `add-code-review` 要求的非作者独立复审。此为已知限制，不声明等价独立复审。
- **工作区**：`D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`

## 固定对象哈希（SHA256）

| 文件 | SHA256 |
|---|---|
| `internal/article/command_service.go` | `B61ABD12ADD7AF43C7D677A89C097F44FFBB818A85D2D674CD218A81340DB6E4` |
| `internal/article/query_service.go` | `D969C1171E81889ABFC0D9CDB44DCD74466BD501828E5EDDB63073CEB1E6160B` |
| `internal/article/handler.go` | `B2E3F68995AEC9FA63C5E9F04EF0CCC458A4262045279D7A7DD794C2DD25DD00` |
| `internal/article/update_test.go` | `D6EF18BF78C12DC438C03CDC05C6DEAF4D6CF6157718BFF92C0DA4EF29366EAA` |
| `internal/article/author_projection_test.go` | `47314C84B79B53A6476F91A6F728CBC521CD78DE36AF61D68DA7298C1B0192F7` |

## 合同核对（设计 §3.3.4）

1. **作者受控详情** `GET /api/v1/users/me/articles/{id}`：`ArticleQueryService.GetAuthorArticle` 只按 `VisibilityPolicy.AuthorProjection`（Principal 精确 `author_id` 匹配 + `deletion_state=active` + `ValidateArticleState`）产出最小 `AuthorArticleView`（`title,body,moderation_status,editable`）与强 ETag；不采用成员公开可见性，故 active 作者可读取自有 violating 文章，且仅得到 `restricted`。✅
2. **最小披露**：`AuthorArticleView` 字段 allowlist 只有四字段；violating 时 `moderation_status=restricted`，**不含** `violation_category/violation_state/hidden_reasons/publication_state/orphaned_from_user_id`。✅
3. **强 ETag 修改** `PUT /api/v1/articles/{id}`：`handler.UpdateArticle` 在任何 repository 调用前完成 `requestIfMatch`（缺失→428 `request.precondition_required`；weak/多值/`*`/语法错误→400 `request.invalid_precondition`），并校验 ETag 内嵌 `id` 与路径 ID 一致（不一致→400，防跨资源 If-Match）；随后执行统一 262144-byte wire decoder 与纯文本/rune 长度合同（复用 `decodeArticleWriteJSON`/`validateArticleContent`）。✅
4. **权限矩阵**：非作者普通成员→404 `article.unavailable`；管理员非作者→403 `auth.forbidden`；管理员作为作者按作者处理（`authorContentEditAllowed`）。✅
5. **状态与字段保持**：`CommandService.ReplaceContent` 在 `SELECT ... FOR UPDATE` 锁定后校验 `deletion_state=active`、`author_id=principal`、`version=expectedVersion`（失配→412）；`UpdateArticleContent` 只写 `title,body,version,updated_at`，并以 `WHERE id=$1 AND version=$4-1` 做守卫式 CAS；publication/violation/deletion/author/source 全部原样保留。✅
6. **失败不产生副作用**：422（零值/全 Unicode 空白/超长）、400（wire 越界/非法 UTF-8）、412、404/403 均在 `UpdateArticleContent` 之前返回，测试断言 `repository.updateCalls==0`；repository 失败回滚（`rollbackArticleContent`）。（✅）
7. **提交语义**：成功 200 返回与详情同形的 `AuthorArticleView` 与新强 ETag；commit 失败以 `errReplaceArticleCommitOutcomeUnknown` 标记未知结果，沿用既有 `platform.database_unavailable` 映射。✅

## 新增生产符号与实现文件所有权

- `command_service.go`：新增 `ArticleContentRepository` 接口、`CommandService.ReplaceContent`、`authorContentEditAllowed`、`rollbackArticleContent`、`Repository.LockArticleForUpdate`、`Repository.UpdateArticleContent`、两条 SQL 常量、`errReplaceArticleCommitOutcomeUnknown`，并加 `var _ ArticleContentRepository = Repository{}` 编译期断言。
- `handler.go`：新增 `ArticleContentReplacer`、`AuthorArticleReader` 接口、`Handler.replacer/authorReader` 字段、`NewAuthorDetailHandler`、`Handler.GetAuthorArticle`、`Handler.UpdateArticle`；`NewHandler` 以类型断言从 creator 复用 replacer。
- `query_service.go`：新增 `ArticleQueryService.GetAuthorArticle`（引入 `han/internal/session` 导入）。
- 说明：设计 §3.3.4 类图把 `+LockByID/+UpdateContent` 列在 `ArticleRepository`，本卡遵循 S4.1 既有先例（Repository 方法置于功能文件、未扩第 6 文件），把两个 Repository 方法放在 `command_service.go`。设计未要求独立 `AuthorEditView` 类型，复用既有 `AuthorArticleView`（model.go:322，等价设计 `AuthorEditView`），无新增 model。

## 验证证据

**本卡专用命令**（`go test ./internal/article -run "TestAuthorArticle|TestUpdateArticle"`）：

```
--- PASS: TestAuthorArticle_ReturnsMinimalProjectionAndETag
--- PASS: TestAuthorArticle_ViolatingArticleIsRestrictedWithoutLeakingCategory
--- PASS: TestAuthorArticle_NonAuthorMemberIsUnavailable
--- PASS: TestAuthorArticle_NotFoundForUnknownArticle
--- PASS: TestAuthorArticle_DeletedArticleIsUnavailable
--- PASS: TestUpdateArticle_AuthorReplacesContentWithMatchingETag
--- PASS: TestUpdateArticle_ViolatingArticleStaysViolatingAndRestricted
--- PASS: TestUpdateArticle_NonAuthorMemberIsNotFound
--- PASS: TestUpdateArticle_AdminNonAuthorIsForbidden
--- PASS: TestUpdateArticle_AdminAsAuthorIsTreatedAsAuthor
--- PASS: TestUpdateArticle_WireByteLimitBoundaries
--- PASS: TestUpdateArticle_StaleETagIsPreconditionFailed
--- PASS: TestUpdateArticle_MissingOrMalformedIfMatchFailsBeforeRepository
--- PASS: TestUpdateArticle_InvalidContentFailsBeforeRepository
--- PASS: TestUpdateArticle_DeletedOrPendingArticleIsUnavailable
--- PASS: TestUpdateArticle_RepositoryFailureRollsBack
ok  	han/internal/article	0.077s
```

`-race` 同 selector：`ok  han/internal/article  1.118s`。

**Manifest 七项（固定顺序）**：`gofmt -l .`（空）、`go mod verify`（all modules verified）、`go vet ./...`（空）、`go build ./...`（空）、`go test -tags=contract ./...`（无 FAIL）、`go test -race ./...`（无 FAIL）、`go test ./...`（无 FAIL）。全部 PASS。

**真实 PostgreSQL 验证（本机 5433，PostgreSQL 18.6）**：以临时 probe（`//go:build integration`，**已删除**，不在写集内）在真实 schema 上验证 `LockArticleForUpdate`/`UpdateArticleContent`：锁定读回版本、CAS 更新后 `version=2`、author/publication/violation/deletion 不变、stale CAS 返回 `ErrPreconditionFailed`、缺失行返回 `pgx.ErrNoRows`。
`go test -tags=integration ./internal/article -run '^TestProbeReplaceContentSQL_RealPostgres$' -count=1 -v` → `PASS`，`server_version_num=180006`。探针文件已移除，`git status` 无残留。

**五文件检查**：`gofmt -l` 空；`git diff --check`（三个修改文件）无输出。

## Findings

| ID | 严重度 | 描述 | 状态 |
|---|---|---|---|
| ARTICLE-S3.1-R01-F001 | INFO | 本卡为作者自审口径，非非作者独立复审 | 已知限制 |
| ARTICLE-S3.1-R01-F002 | LOW | `PUT /api/v1/articles/{id}` 与 `GET /api/v1/users/me/articles/{id}` 尚无 production `cmd/han-server` 路由接线 | 转交独立任务 |
| ARTICLE-S3.1-R01-F003 | LOW | A-ETAG-02 双 writer PostgreSQL 行锁并发、A-MOD-04 三视图跨模块契约、A-MOD-06/07/08 精确 N/N+1 wire 与 PostgreSQL 前后快照 | 转交 `article-management-S2.2`、`article-management-S1.5` |

## 结论

**Code Review Gate: PASS（作者自审口径）** — 设计 §3.3.4 的作者受控详情、强 If-Match、最小披露、权限矩阵、字段保持与失败零副作用均落到精确五文件；本卡专用命令、race、Manifest 七项与真实 PostgreSQL SQL 探针全部 PASS；无 HIGH/MEDIUM finding。F002/F003 属本卡关闭条件中「真实 PostgreSQL 并发、跨模块契约」的**独立后续任务**（`S1.5`/`S2.2`），不构成本卡代码级阻塞。

**未验证边界**：production router 接线、部署、浏览器/E2E、容量/P95、production readiness 均未验证。本 PASS 弱于非作者独立复审。
