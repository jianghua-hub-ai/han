# identity-auth-S7.1 Code Review R02

- 日期：2026-10-23
- Reviewer：Meitner（原非作者独立 Reviewer）
- 作者：Noether `01a16893-bc25-7a52-8532-4bd7e203565d`
- objectType：code revision review
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/identity-auth/review-code-s7.1-r01.md`
- 原报告 SHA256：`D0F75985CAC5F52969304945B9AC49941D2848607795EAF1C9D356DFBA3746CD`
- 任务：`identity-auth-S7.1`
- 结论：**Code Review Gate = PASS（F002 已获明确范围接受并转交 S7.2，索引仍未实现）**

## 1. 固定复审对象

复审开始时核对：

| 文件 | R02 固定 SHA256 |
|---|---|
| `internal/identity/handler.go` | `CD7DD480EA0D4E6A34694B254237177583FB22DB8F0416A6FC9D9064FC16DC75` |
| `internal/identity/query.go` | `534E6E362431B669D271D689365161F95E843D9838D2DA03398F51806DD2C407` |
| `internal/identity/cursor.go` | `E0514475A7ECA31E24A8092A51D2F814DCCB112F7D2022A3D903E2AA5A1A6FFD` |
| `internal/identity/query_test.go` | `786B1A3ED94CC0DA5AF9B004250DE74FED67C92108436A0E18F1B6E7CF6886E9` |
| `internal/identity/cursor_test.go` | `4F5D647E2EA29CC4D4EB2B837178895FD40E9F24E85D4976BC71BC9C26C4864D` |

工作树存在其他工作者改动，且 `internal/` 整体未跟踪；本次未回滚、覆盖、格式化或修改任何被审代码、Tasks、Plan、设计、test-plan 或 r01。

## 2. 两轮实际复审范围

### Round 1：修订代码与原 finding 定点复审

完整读取 Reviewer Role、`add-code-review`、适用 core guidance、产品安全红线和 go-service coding/security/observability/tracing 规则；复读 r01 和五个固定文件。

逐项检查：

- F001：接口是否与真实 `pgxpool.Pool` 方法签名一致，是否有编译证明；
- F003：malformed UUID 是否在数据库前收敛为 404；
- F004：缺失、显式空、重复和有效 cursor 是否有确定语义；
- F005：代码是否修复，或是否已有有效风险接受；
- 上述修订是否破坏授权、最小披露、cursor 绑定、keyset、错误链或测试替身。

### Round 2：有界消费者、schema 与转交合同复审

核对：

- pgx v5.11.0 `pgxpool.Pool.Query/QueryRow` 和 `pgx.Rows/pgx.Row`；
- platform 公共错误注册表、`session.Principal`、User/Role/Status；
- design r28 §3.3.8、test-plan r11 I-LIST-01～03；
- users schema/index 现状；
- Tasks 中 S7.1/S7.2、Plan B5D、DAG、文件所有权、验证和关闭 Gate；
- 当前 composition 编译消费者。

未扩大为全仓泛审。

## 3. 原 findings 逐项裁定

### IDENTITY-S7.1-REV-R01-F001 — CLOSED

**原问题**：自定义数据库端口返回类型与真实 pgx 不兼容。

**修订证据**：

- `internal/identity/query.go:92-98` 直接使用 `pgx.Rows` / `pgx.Row`；
- `internal/identity/query.go:98` 加入
  `var _ UserQueryDatabase = (*pgxpool.Pool)(nil)`；
- `internal/identity/query_test.go:24` 也保留真实类型编译断言；
- 测试 rows stub 实现完整 `pgx.Rows` 契约；
- pgx v5.11.0 `pgxpool/pool.go:764,791` 的返回类型与端口一致。

**触发复核**：`*pgxpool.Pool` 作为 `UserQueryDatabase` 编译通过；identity 包、空跑 composition 编译和 vet 均通过。

**裁定**：真实生产 pool 不再被自定义返回类型阻断，原根因已关闭。真实 PostgreSQL 查询执行仍属于 F002/S7.2 和后续集成限制，不重开 F001。

### IDENTITY-S7.1-REV-R01-F002 — TRANSFERRED / ACCEPTED-WITH-FOLLOW-UP，仍 OPEN 于 S7.2

**原问题**：design r28 §3.3.8 要求的两个治理分页索引尚未实现。

**当前事实**：

- `migrations/0002_identity.sql:1-51` 仍只有 users 主键和 `users_single_initial_admin_idx`；
- `migrations/0006_user_governance_indexes.sql` 当前不存在；
- `internal/identity/repository_integration_test.go` 当前没有本 finding 的 governance index 测试；
- 因此索引**没有实现，F002 不能标为 CLOSED**。

**有效转交证据**：

- `.add/project/knowledge/tasks/tasks.md:715-734` 已真实登记 todo
  `identity-auth-S7.2`；
- S7.2 固定唯一写集为：
  - `migrations/0006_user_governance_indexes.sql`
  - `internal/identity/repository_integration_test.go`
- S7.2 requires B0、S1.2、S7.1，保持 todo、未认领、未启动；
- `.add/project/knowledge/plans/ir-01-content-community-batches.md:545-559`
  已将其安排到未来 B5D，路径为 `B5B → B5D → B6A`；
- Tasks/Plan 都明确写明 F002 仍为 MEDIUM / OPEN，不得冒充索引已修复。

**本次处置依据**：用户明确决定不扩大 S7.1 五文件边界，并接受把 F002 作为独立后续任务实施。按照 add-code-review，MEDIUM 可由责任人显式接受；本次接受只解除其对 **S7.1 当前五文件 Gate** 的阻断，不改变 F002 的全局 OPEN 状态。

**S7.2 强制约束**：

1. 仅修改登记的两个文件，不修改 `0001`～`0005` 或 S7.1 五文件；
2. 实现 `(role,status,created_at,id)` 与 `(created_at,id)` 两项索引；
3. 使用最终 S7.1 SQL，在 PostgreSQL 18 验证索引定义、role/status 与无过滤路径、同时间戳多页、不漏不重、排除自身和角色范围；
4. 提供 `EXPLAIN (ANALYZE, BUFFERS)` 或等价计划证据；
5. 验证 migration 正向顺序、重复运行只记录一次和失败事务回滚；项目无 down migration 合同，不补造反向迁移；
6. 完成适用 Manifest 检查、非作者 Review，并由原 Reviewer或获授权 Reviewer正式关闭 F002；
7. F002 未关闭前，B5D 不得关闭、不得进入 B6A，也不得宣称治理分页索引已交付。

**裁定**：对 S7.1 为 `TRANSFERRED / ACCEPTED-WITH-FOLLOW-UP`；对 S7.2 仍是 `MEDIUM / OPEN`。

### IDENTITY-S7.1-REV-R01-F003 — CLOSED

**原问题**：非 UUID 详情 ID 进入 pgx 后被误映射为 503。

**修订证据**：

- `internal/identity/query.go:241-247` 在数据库调用前用 `pgtype.UUID.Scan` 验证目标 ID；
- 无效或非 valid UUID 直接返回既有 `identity.user_not_found`；
- `internal/identity/query.go:248-254` 只把已验证的 `pgtype.UUID` 传给数据库；
- `internal/identity/query_test.go:361-404` 覆盖 users/administrators 两类合法操作者，断言 malformed ID 返回 `identity.user_not_found` 且数据库调用数为 0；
- `internal/platform/errors.go:98` 将该 code 固定映射为 HTTP 404。

**Reviewer 复跑**：`TestUserQueryDetailRejectsMalformedIDBeforeDatabase` 两个子用例通过；platform registry 稳定映射测试通过。

**裁定**：malformed UUID 在 DB 前收敛为 404，且不会伪装成数据库不可用，原 finding 关闭。

### IDENTITY-S7.1-REV-R01-F004 — CLOSED

**原问题**：显式空 `cursor` 被当成首页。

**修订证据**：

- `internal/identity/handler.go:601-613` 区分 cursor 缺失和显式空值；
- `internal/identity/handler.go:677-688` 的 helper 返回 `(value,present,error)`；
- 显式空值和重复值都在调用 service 前返回 `request.invalid_cursor`；
- `internal/identity/query_test.go:468-558` 对 users/administrators 分别覆盖 missing、empty、repeated、valid，并断言 empty/repeated 时 service 调用数为 0。

**Reviewer 复跑**：`TestUserQueryHandlerCursorPresenceContract` 八个子用例全部通过。

**裁定**：缺失 cursor 仍表示首页；显式空和重复参数稳定拒绝，原 finding 关闭。

### IDENTITY-S7.1-REV-R01-F005 — ACCEPTED RISK（LOW，非阻断）

**原问题**：查询 Handler 仍通过 `_ =` 丢弃响应写错误。

**当前事实**：`internal/identity/handler.go:593-674` 未增加查询响应写观测，原 LOW 代码现状仍存在。

**处置依据**：用户已明确接受该 LOW 风险；LOW 不影响当前正确性 Gate，且本次没有授权扩大 Handler 构造/观测接缝。该接受不代表代码已修复，也不得改写为 CLOSED。

**裁定**：`ACCEPTED RISK / NOT CODE-CLOSED`，不阻断 S7.1 Gate。后续若统一治理查询响应写观测，仍须保证单次写入、写后不二次响应、低敏 trace/endpoint/status/reason 观测。

## 4. 新 findings

**无新增 finding。**

复审未发现 F001/F003/F004 修订引入授权、最小披露、cursor 绑定、keyset、公共错误或 pgx 类型相关回归。

## 5. Reviewer 命令证据

以下命令由 Reviewer 在本报告固定哈希上运行并通过：

```text
go test ./internal/identity -run 'TestUserQuery|TestIdentityCursor' -count=3
go test -race ./internal/identity -run 'TestUserQuery|TestIdentityCursor' -count=1
go test ./internal/identity -run 'TestUserQueryDetailRejectsMalformedIDBeforeDatabase|TestUserQueryHandlerCursorPresenceContract' -count=1 -v
go test ./internal/platform -run 'Test.*Registry|TestWriteError' -count=1
go test ./internal/identity -count=1
go test ./internal/identity ./cmd/han-server -run '^$' -count=1
go vet ./internal/identity
gofmt -l internal/identity/handler.go internal/identity/query.go internal/identity/cursor.go internal/identity/query_test.go internal/identity/cursor_test.go
```

`gofmt -l` 无输出；另行执行五文件尾随空白检查，结果 PASS。真实 pgxpool 方法签名已从本机 pgx v5.11.0 源码复核。

Lead 报告的 Manifest 七项全 PASS 仅作为 Lead 现场证据记录；Reviewer 未冒充重跑全部七项。

## 6. 限制与剩余风险

- 未运行真实 PostgreSQL，未验证治理 SQL、索引定义、查询计划或跨页真实数据遍历。
- `0006_user_governance_indexes.sql` 尚未创建，S7.2 尚未启动；F002 保持 OPEN。
- 未运行 production router、浏览器、部署或 E2E。
- F005 的响应写错误观测缺口由用户明确接受，本报告不宣称已经修复。
- 当前工作树含其他工作者改动；结论只绑定本报告列出的五个 SHA256 和读取到的 S7.2/Plan 转交记录。

## 7. Gate

R01 finding 处置统计：

- CLOSED：F001、F003、F004
- TRANSFERRED / ACCEPTED-WITH-FOLLOW-UP：F002（MEDIUM，仍 OPEN 于 S7.2）
- ACCEPTED RISK：F005（LOW，非阻断，未代码关闭）
- 新 findings：0

**Code Review Gate：PASS**

该 PASS 仅表示：

1. S7.1 固定五文件没有未接受的阻断 finding；
2. F002 已由用户明确接受为独立后续实施，并有完整、可核对的 S7.2/B5D 所有权与关闭 Gate；
3. F005 的 LOW 风险已被明确接受。

该 PASS **不表示** F002 已关闭、索引已实现、S7.2/B5D 已启动或通过，也不外推为 PostgreSQL、production router、浏览器、部署或 E2E PASS。
