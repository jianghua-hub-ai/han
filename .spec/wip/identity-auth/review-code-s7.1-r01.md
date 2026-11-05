# identity-auth-S7.1 Code Review R01

- 日期：2026-10-23
- Reviewer：add-reviewer（非作者独立 Reviewer）
- 作者：Noether `01a16893-bc25-7a52-8532-4bd7e203565d`
- objectType：code
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 任务：`identity-auth-S7.1`
- 结论：**Code Review Gate = FAIL**

## 1. 固定对象

开始审查时已核对：

| 文件 | SHA256 |
|---|---|
| `internal/identity/handler.go` | `867F052633D61DE52901913588EAA86CD4D3256844C4A051CF00866D0CCF11F8` |
| `internal/identity/query.go` | `929DFDA929986235439C7173273F62A8D92B3D172035A6CA1BF7866C34D0DE4A` |
| `internal/identity/cursor.go` | `E0514475A7ECA31E24A8092A51D2F814DCCB112F7D2022A3D903E2AA5A1A6FFD` |
| `internal/identity/query_test.go` | `4A1C2C08714B784FCF37B7D9D57D3D41A7F0EFD251D29B8FA4E6E4B838B5E4B3` |
| `internal/identity/cursor_test.go` | `4F5D647E2EA29CC4D4EB2B837178895FD40E9F24E85D4976BC71BC9C26C4864D` |

`handler.go` 给定基线 SHA256：
`1E7200AF5A46509FB1CB31E58EAE598EC27C6F5A53E43BA2F06306924449E076`。

仓库存在其他工作者的已修改/未跟踪文件，且 `internal/` 整体未跟踪；本次未回滚、覆盖、格式化或修改这些内容。

## 2. 依据与两轮范围

### Round 1：固定代码直接审查

完整读取 Reviewer Role、`add-code-review`，以及 code-review checklist、abstraction、coding、no-hardcode、产品安全红线和 go-service 中适用的 coding/security/observability/tracing 规则。

逐行检查五个固定文件，重点覆盖：

- 普通管理员、初始管理员、目标类别和排除自身的授权矩阵；
- Summary/Detail JSON allowlist 与最小披露；
- `(created_at DESC,id DESC)` keyset、`limit+1`、空页和末页；
- cursor 的 HMAC、版本、时效、endpoint/category/visibility/limit/filter 绑定；
- 参数、公共错误、数据库错误映射；
- 测试替身是否真实反映生产依赖及失败行为。

### Round 2：有界消费者与契约审查

读取并交叉核对：

- `tasks.md` S7.1；
- identity design r28 §3.3.8；
- test-plan r11 I-LIST-01～03；
- ADR-0014 与设计复审 r03 PASS；
- `User` / `Role` / `Status`、`session.Principal`；
- platform 公共错误和 JSON 边界；
- identity repository、users schema/index；
- pgx v5.11.0 的 `pgxpool.Pool` 查询签名和 UUID codec；
- 当前 `cmd/han-server` composition root；
- 相关 handler/query/cursor 测试。

未扩大为全仓泛审。

## 3. Findings

### MEDIUM

#### IDENTITY-S7.1-REV-R01-F001 — 查询数据库端口不能由真实 pgx pool/transaction 实现

- **状态**：OPEN
- **位置**：
  - `internal/identity/query.go:90-107`
  - `internal/identity/query_test.go:513-547`
  - `cmd/han-server/main.go:73-79,846-849`
  - pgx v5.11.0 `pgxpool/pool.go:764,791`
- **违反依据**：S7.1 必须形成可消费的查询实现；test-plan I-LIST-01 要求 PostgreSQL 集成；code-review checklist 要求检查真实消费者和测试替身失真。
- **触发**：生产装配尝试把 `*pgxpool.Pool` 或 `pgx.Tx` 传给 `NewUserQuery`。
- **证据**：`UserQueryDatabase.Query` 返回自定义 `UserQueryRows`，`QueryRow` 返回自定义 `UserQueryRow`；真实 pgx 方法分别返回 `pgx.Rows` 和 `pgx.Row`。Go 方法返回类型不协变，因此真实 pool/transaction 不实现该接口。测试 stub 直接返回自定义接口，掩盖了不兼容；仓内未发现 adapter。
- **影响**：当前查询 service 无法与项目真实数据库依赖直接组合；后续 router 接线会在编译期失败，现有单元测试通过不能证明生产可用。
- **最小建议**：让端口签名与 pgx 的实际返回类型一致，或在经授权范围内提供明确 adapter；不要继续用仅测试替身可实现的生产端口。
- **验证要求**：
  1. 加入并通过真实类型编译断言或等价 production composition 编译测试；
  2. 使用真实 PostgreSQL/pgx 覆盖 list/detail；
  3. 保留现有权限、错误和分页回归。

#### IDENTITY-S7.1-REV-R01-F002 — 设计规定的治理分页索引未落入 schema

- **状态**：OPEN
- **位置**：
  - `internal/identity/query.go:418-453`
  - `migrations/0002_identity.sql:1-51`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:1679-1681`
- **违反依据**：design r28 §3.3.8 明确要求 `(role,status,created_at,id)` 与 `(created_at,id)` 索引；test-plan I-LIST-01 要求 PostgreSQL 集成分页证据。
- **触发**：users 数据增长后执行按 role/status 过滤并按 `created_at,id` 倒序的列表查询。
- **证据**：当前 users schema 只有主键和 `users_single_initial_admin_idx`，没有设计指定的两个分页索引。
- **影响**：治理列表可能退化为过滤/排序扫描；固定对象虽实现 keyset 语义，但没有交付设计要求的数据库访问路径。单元 stub 无法暴露该缺口。
- **最小建议**：由 Lead 授权 migration 范围，新增与最终 SQL 谓词、排序一致的索引；不要由 Reviewer 越界修改 schema。
- **验证要求**：
  1. migration 正反向/幂等检查；
  2. 真实 PostgreSQL `EXPLAIN (ANALYZE, BUFFERS)` 或等价计划证据；
  3. 同时间戳、多页、排除自身及不同角色范围的完整遍历测试。

#### IDENTITY-S7.1-REV-R01-F003 — 非 UUID 详情 ID 被误映射为数据库不可用

- **状态**：OPEN
- **位置**：
  - `internal/identity/handler.go:653-661`
  - `internal/identity/query.go:250-265,369-373,439-453`
  - `migrations/0002_identity.sql:1-2`
- **违反依据**：
  - design r28 A3：外部 ID 为不透明字符串、数据库使用 UUID；
  - design r28 公共错误表：目标缺失或不可见应为 `404 identity.user_not_found`；
  - observability 规则要求公共错误语义贴切。
- **触发**：请求 `GET /api/v1/admin/users/not-a-uuid` 或对应 administrators 路径。
- **证据**：Handler 只拒绝空值和额外 `/`；service 把原字符串作为 `users.id uuid` 参数传给 pgx。UUID 编码/转换错误随后被统一包装为 `platform.database_unavailable`，即 503。当前 stub 不执行真实 pgx 参数编码。
- **影响**：客户端提供的不可用目标被伪装成可重试数据库故障，产生错误告警和错误重试语义，并偏离最小披露的 404 合同。
- **最小建议**：在数据库调用前验证服务内部 UUID 表示；无效表示按不可用目标返回既有 `identity.user_not_found`，不要新增公开 UUID 格式合同。
- **验证要求**：
  1. users/administrators 两条详情路径覆盖 malformed ID；
  2. 断言 404、稳定 code、零数据库调用；
  3. 用真实 pgx 回归合法 UUID、缺失目标和数据库故障三类映射。

#### IDENTITY-S7.1-REV-R01-F004 — 显式空 cursor 被静默当作首页

- **状态**：OPEN
- **位置**：
  - `internal/identity/handler.go:601-605,628-632,669-677`
  - `internal/identity/query.go:176-185`
- **违反依据**：design r28 §3.3.8 分页合同要求解码失败或非法游标返回 `400 request.invalid_cursor`；test-plan I-LIST-02 要求非法游标拒绝且无数据披露。
- **触发**：请求 `/api/v1/admin/users?cursor=` 或 administrators 对应路径。
- **证据**：参数 helper 对“缺失”和“单个空值”都返回空字符串；与 `limit` 不同，cursor 没有检查参数是否实际出现。service 又只在 `Cursor != ""` 时解码，因此直接查询第一页。
- **影响**：非法 continuation 请求被解释为新查询，客户端可能无意重复读取第一页，并使公共错误合同不稳定。
- **最小建议**：区分 cursor 缺失与显式空值；显式空值必须在调用 service/数据库前返回既有 `request.invalid_cursor`。
- **验证要求**：补 users/administrators HTTP 测试，覆盖缺失、空值、重复值和有效值，并断言空值时 service/数据库调用数为零。

### LOW

#### IDENTITY-S7.1-REV-R01-F005 — 新查询 HTTP 边界静默丢弃响应写错误

- **状态**：OPEN（非阻断）
- **位置**：`internal/identity/handler.go:593-637,645-666`
- **违反依据**：
  - `coding-standards.md:21` 禁止静默吞错；
  - go-service observability 规则要求保留根因，并在唯一责任边界处理一次。
- **触发**：客户端断连或 partial `ResponseWriter` 使 `WriteError` / `WriteJSON` 返回错误。
- **证据**：新 list/detail 路径的所有写入错误均通过 `_ =` 丢弃，未进入 logger、metric 或 trace-aware observer。
- **影响**：服务端响应写故障没有可定位证据；测试无法区分成功发送和部分/失败发送。
- **最小建议**：在查询 HTTP 最终边界做一次有界、低敏的写结果观测；响应开始后不得二次写 error envelope。观测至少携带可信 trace、endpoint、尝试状态和稳定原因分类，不携带用户详情。
- **验证要求**：partial/failing writer 测试证明每个请求只有一次响应写尝试，写错误只观测一次且不泄露 Summary/Detail 数据。

## 4. 已确认无直接 finding 的检查项

- 普通管理员只能查询 `member`，且 SQL 排除 actor 自身；不能借 administrator 类别枚举管理员。
- initial admin 可以查询普通用户和其他管理员，但仍排除自身。
- Summary/Detail 使用显式 JSON allowlist；Summary 不含 email/phone，Detail 才包含，空值保持 `null`。
- 删除用户、密码摘要、临时凭据、Session、`retained_email`、审计 metadata 未进入响应模型。
- keyset 条件、倒序排序、`limit+1`、`has_more` 和 next cursor 位置计算在代码内一致。
- 空页和末页返回非 nil 空数组、`next_cursor:null`、`has_more:false`。
- cursor 包含 HMAC、版本、过期和 endpoint/category/visibility/limit/filter 绑定；已覆盖篡改和绑定不匹配。
- `limit` 的省略、非正和上限裁剪与设计一致。

这些结论不消除 F001～F004 所述真实 pgx、schema 和 HTTP 边界缺口。

## 5. Reviewer 运行证据

以下命令由 Reviewer 在固定对象上运行并通过：

```text
go test ./internal/identity -run "TestUserQuery|TestIdentityCursor" -count=3
go test -race ./internal/identity -run "TestUserQuery|TestIdentityCursor" -count=1
go test ./internal/identity -count=1
gofmt -l internal/identity/handler.go internal/identity/query.go internal/identity/cursor.go internal/identity/query_test.go internal/identity/cursor_test.go
go test ./internal/identity ./cmd/han-server -run '^$' -count=1
go vet ./internal/identity
```

`gofmt -l` 无输出。定向测试通过，但测试替身未证明 F001、F002、F003 的真实 pgx/schema 行为。

Lead 报告的全仓 vet/build/contract/race/test 和 scoped diff-check 仅作为上游证据记录，不冒充 Reviewer 重跑结果。

## 6. 待决定与非阻断建议

### 待 Lead 决定

1. F001 若采用 adapter，以及 F002 所需 migration，都超出 S7.1 当前固定五文件写集；需由 Lead 明确授权、重排任务或分派作者，Reviewer 不替作者修改。
2. production router 尚未注册 S7.1 路径。本次按已知后续 composition 范围处理，不单列 finding；在实际开放能力前仍必须完成生产接线和契约验证。

### 非阻断建议

- 将 PostgreSQL cursor 全遍历和字段 allowlist 合同测试保留在后续 `identity-auth-S1.5`，但不能以其“后续”身份延后 F001/F002 的实现可消费性修复。
- 为 `UserQueryDatabase` 加入生产类型编译证明，防止自定义 stub 再次与真实 pgx 漂移。

## 7. 限制与未验证风险

- 未运行真实 PostgreSQL，因此没有执行 SQL、pgx UUID 编码、查询计划、索引选择或跨页并发插入验证。
- 未做 production router 接线、浏览器、部署或 E2E；这些均不在本次授权范围。
- 未做未授权的 migration、adapter 或 cmd 修改。
- 当前工作树含其他工作者改动；本结论只绑定本报告列出的五个 SHA256 和上述有界消费者状态。

## 8. Gate 与复审条件

严重度统计：

- CRITICAL：0
- HIGH：0
- MEDIUM：4 OPEN
- LOW：1 OPEN（非阻断）

**Code Review Gate：FAIL**

复审至少需要：

1. 固定新的修订对象和 SHA256；
2. 逐项关闭 F001～F004；
3. 提供真实 pgx 类型可组合证据；
4. 提供授权 migration 与真实 PostgreSQL 分页/计划证据；
5. 覆盖 malformed ID、显式空 cursor 和真实错误映射；
6. 对 F005 给出修复或 Lead 明确的非阻断风险接受记录；
7. 复审开始和结束均重核对象哈希。
