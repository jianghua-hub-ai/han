# Case18 当前 revision 执行证据 r01（S8.2 / S1.5 消费）

- 生成时间：2026-11-03（Asia/Shanghai）
- 执行主体：主会话 Lead 直接执行（未委派 Role，故本轮无 Role 产出）
- 授权来源：项目级验证授权 ADR-0028（测试项目；可执行层必须实际执行留证；真实环境不可用时标注"未真实验证"；AI 代理评审/签收须登记代理主体与范围，不冒充真人签字）+ 用户 2026-11-03「先推进 S1.5/S8.2 剩余验收」
- 工作区：`D:\AID\VSCode\Codex\Han`，branch `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（未做任何 Git 写）
- 环境：`go version go1.27.0 windows/amd64`；PostgreSQL 18.6 loopback `127.0.0.1:5433`；DSN 经 User 级 `HAN_TEST_POSTGRES_DSN` 注入，未打印、未落盘

## 1. 动机

S8.2 的 Case18 覆盖对账独立评审 R01（`evidence-case18-coverage-r01/`，CHANGES REQUIRED）列出六项关闭前置。本条只处理其中两项可在本机当前 revision 直接执行并离线核实的前提：

- 「当前 revision root/装配执行证据」（对应 `C18-COV-F001` 的 `PExecuted:false` 与旧哈希绑定）
- 部分「当前 revision 真实 SessionStore SQL/撤销」（对应 `C18-COV-F004` 的 Session 侧）

`C18-COV-F002`（全状态真实隔离）、`F003`（当前 runtime 完整恢复+开流）、`F005`（时点快照哈希）本条不处理，结论不变。

## 2. 源码绑定（执行前后逐项一致）

见 `source-hashes-before.txt` 与 `source-hashes-after.txt`，八文件前后 SHA256/字节完全相同：

| 文件 | SHA256 | 字节 |
| --- | --- | --- |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 |
| `cmd/han-server/main_test.go` | `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` | 400312 |
| `internal/article/query_service.go` | `D3151AC329C8C9EB8A1B8E00FF45A58CC467E06E1C9A00F9A63830048C0108B4` | 29621 |
| `internal/article/handler.go` | `C684B4C874083035C1BC6B1F22DF81833D8DF490544E8586719C028B203D3750` | 37839 |
| `internal/article/model.go` | `42B2BEEEAC0B4CB3B6E2138314956F2D1FCBD0E32D6E7B32BB2352C656E8851D` | 13800 |
| `internal/article/policy.go` | `6CD1543BC2E715F76EE7B6C5CDDDB3153A9A425B84C55A366648E600B78C8774` | 13831 |
| `internal/article/admin_query_test.go` | `DBAF93B7600831D61DC74804D82905CAE0719014357A9107321DFCD8C3EF3F61` | 30090 |
| `test/integration/article/admin_query_test.go` | `3ADC5BFA00410F2BF7B520C0035E0089B6E24889770BF6BD1F283A7E5B798263` | 31515 |

`main.go` 与 `main_test.go` 与 Case18 覆盖对账 R01 登记的评审基线逐字相同，因此本条证据与 `C18-COV-F001` 所指"旧 `6B45B550…` / `B53DD34D…`"不是同一对象。

## 3. 执行与结果

| # | 命令（workdir 仓库根） | 退出码 | 结果 |
| --- | --- | --- | --- |
| 1 | `go test ./cmd/han-server -run '^TestProductionAdminArticleQuery$' -count=1 -v` | 0 | 93 个 PASS 节点（1 顶层 + 92 子测试），0 FAIL / 0 SKIP，包级 3.205s |
| 2 | `go test -race ./cmd/han-server -run '^TestProductionAdminArticleQuery$' -count=1 -v` | 0 | 93 个 PASS 节点，0 FAIL / 0 SKIP，`DATA RACE` 0 行，包级 4.993s |
| 3 | `go test ./internal/article -run '^(TestAdminQuery\|TestAdminCursor\|TestAdminCursorFiveViewIsolation)$' -count=1 -v` | 0 | 135 个 PASS 节点，0 FAIL |
| 4 | `go test -race ./internal/article -run '^(TestAdminQuery\|TestAdminCursor\|TestAdminCursorFiveViewIsolation)$' -count=1 -v` | 0 | 135 个 PASS 节点，0 FAIL，`DATA RACE` 0 行 |
| 5 | `go build ./...` | 0 | 空输出 |
| 6 | `go vet ./cmd/han-server ./internal/article` | 0 | 空输出 |
| 7 | `go test -tags=integration ./test/integration/article -run '^TestAdminQueryPostgres$' -count=1 -v` | 0 | 99 个 PASS 节点（16 个顶层子测试），0 FAIL / 0 SKIP，包级 23.085s |
| 8 | `go test -tags=integration ./test/integration/identity -run '^(TestS16SessionResolveRejectsRevokedAndSuspendedUsers\|TestS16UserFirstLockOrderingAndSessionRevocation\|TestS33LoginSessionPostgres)$' -count=1 -v` | 0 | 6 个 PASS 节点，0 FAIL / 0 SKIP，包级 5.579s |

运行 7 覆盖：`RawQueryAndPresence`、`FiveViewIsolation`、`ViewStateMatrix`、`StaticCompositeKeyset`、`ScanAndPublicCursor`、`LimitsAndExhaustion`、`CurrentAuthorTransitions`、`CurrentModerationTransitions`、`StrictIdentityWholeFailure`、`IdentityScope`、`DetailPriority`、`SQLRowsCancel`、`ZeroBusinessWrites`、`TwelveFieldsAndPrivacy`、`SameArticleETag`、`ErrorAndOldConsumerCompatibility`；路径为真实 SQL → reader → `ArticleQueryService` → Admin Handler，非 production root 注册（root 层由运行 1/2 承担）。保留 schema `han_admin2_1cc188e4ed204226825e9420363a9bbb`（未 DROP、未清理，迁移 0001～0009 按序提交后执行）。

运行 8 覆盖：`TestS33LoginSessionPostgres`（登录会话轮换摘要与受保护请求、密码失败保留旧会话、审计失败回滚轮换）、`TestS16UserFirstLockOrderingAndSessionRevocation`、`TestS16SessionResolveRejectsRevokedAndSuspendedUsers`；真实 PostgreSQL 上执行 `SessionStore` SQL 解析与撤销/停用拒绝。保留 schema `han_s33_12bc6017e1f14506a10675dc82999702`（`retain_policy=always`，事务 `committed`，worker 已 join、pool 已关闭）。

## 4. 对 Case18 findings 的实际推进

- `C18-COV-F001`（HIGH，S8.2 root 证据绑定旧哈希且 `PExecuted:false`）：**在事实层消除**——当前 revision 的 root/装配用例已实际执行，普通与 `-race` 各 93 PASS / 0 fail / 0 skip，源码哈希与评审基线一致。**但 finding 本身不由 Lead 自行改判 CLOSED**，须由非作者独立复审本证据包后判定。
- `C18-COV-F004`（MEDIUM，Session/AccessGeneral 缺当前 revision 真实 `SessionStore` SQL 与会话撤销）：**Session 侧在事实层消除**（运行 8）。**AccessGeneral 侧仍未覆盖**——授权链仍只有 resolver/repository/handler 接缝证据，本条不推论其通过。finding 整体保持 open。
- `C18-COV-F002`（HIGH，全状态真实隔离）：**不变**。当前 `RecoveryGuard` 的准入状态词表为 `closed` / `internal_ready` / `public_ready` / `unknown` / `faulted` / `draining` / `stopped`，仅 `public_ready` 放行；`failed` / `expired` / `undetermined` 在本仓库属 run/lease 结局词，不是准入状态。矩阵仍是 `newS419RecoveryRuntime` 假 runtime 与零值 `pgxpool.Pool`，未做真实 runtime/PG/双系统。
- `C18-COV-F003`（HIGH，当前 runtime 完整恢复+开流）：**不变**。production `Enter`/`Install` 仍返回 `ErrAdmissionClosed`，无完整恢复→`public_ready`→生产开流证据。
- `C18-COV-F005`（MEDIUM）：**不变**，按 Lead 已裁定的"时点快照语义"保持 open。

## 5. 边界与未验证

本证据包**不**证明：真实部署 E2E、真实 SMTP/approved-provider、双系统指标/告警/job 接线、完整恢复后生产开流、`Enter`/`Install` 开流、真实 Journal/HMAC/keyring、旧备份恢复、容量/P95、production readiness、人的业务接收。

未做：Git 写、部署、文件删除/归档、凭据输出、数据库 DROP 或清理。保留的测试 schema 可能影响后续同库测试的磁盘占用，但不影响与本条结论的可核性。

本条为 AI 代理执行与登记（ADR-0028），非真人签字，不代替独立评审或人的接收。