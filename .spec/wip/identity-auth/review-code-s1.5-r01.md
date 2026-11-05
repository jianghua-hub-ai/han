# Identity S1.5 代码评审（作者自审口径）

- **评审对象**：`identity-auth-S1.5`「验证身份 REST、内部端口与 ReplayStatus 消费契约」
- **评审范围（精确五文件，唯一写集）**：
  - `test/contract/identity/rest_contract_test.go`（新增）
  - `test/contract/identity/internal_ports_test.go`（新增）
  - `test/contract/identity/replay_status_consumer_test.go`（新增）
  - `testdata/contracts/article-deletion-replay/ready-v1.json`（新增）
  - `testdata/contracts/article-deletion-replay/unavailable-v1.json`（新增）
- **设计/测试依据**：identity test-plan r18 I-CON-01～03 / I-REC-06～08 / I-PRD-01/05/11～14/19；identity design r40 §3.3.11/12、§9.4；ADR-0020/0023
- **评审日期**：2026-10-27
- **评审口径（重要限制）**：本 Host 无委派能力，无独立 Reviewer Role 产出。依据用户 2026-10-27 明确授权「本 Host 可以自审、作者自审口径」，本报告由 Lead 以**作者自审**口径执行，**弱于** `add-code-review` 要求的非作者独立复审。此为已知限制，不声明等价独立复审。
- **工作区**：`D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`

## 固定对象哈希（SHA256）

| 文件 | SHA256 |
|---|---|
| `test/contract/identity/rest_contract_test.go` | `86133EF48C35F704628100AE72C865D8526142791C3F7AE10FF1725230C994A5` |
| `test/contract/identity/internal_ports_test.go` | `F749E7185D660CBFEC481CB5C83B30BA730003383354F622C6128123F12BBB1D` |
| `test/contract/identity/replay_status_consumer_test.go` | `9E9495C24D3212ADB34AEC199D90650CF99FA1E59C035CEA6ABB7DEC32C49752` |
| `testdata/contracts/article-deletion-replay/ready-v1.json` | `586A0B8F76ADE8B720E44094A0D13163C827D231AC9CBC786C6D4AF92E90F0F2` |
| `testdata/contracts/article-deletion-replay/unavailable-v1.json` | `929193CD06BA566402F3F3C1D6321E57BDA95D0270DED666B5C951D97B21AC1B` |

## 合同核对

### 1. 身份 REST 契约（I-CON-01/02）

- **严格 JSON**：`POST /api/v1/users` 边界对 unknown/duplicate field、malformed JSON、非 `application/json` Content-Type、顶层数组、trailing value、空 body 全部稳定返回 `400 request.invalid_json`，且 **service 零调用**（`registrationServiceSpy.calls==0`）。✅
- **错误 envelope 稳定**：`{code,message,field_errors?,trace_id}`；message 与 `platform.DefaultRegistry` 定义逐字一致；`field_errors` 原样透传且不含内部 cause；body `trace_id` 与响应 `TraceHeader` 完全一致。✅
- **router 错误路由**：`request.route_not_found`→JSON 404、`request.method_not_allowed`→JSON 405 且 `Allow` 精确，Content-Type 为 JSON，不回落到 plain-text。✅（router 组装由 `cmd/han-server` 承接，本卡只固定公共 envelope 合同）
- **成功响应 allowlist**：`POST /api/v1/users` 成功 body 仅 6 个字段（`id,username,role,status,must_change_password,created_at`），不泄露 `password`/`source_fingerprint`/`source_kind`。✅

### 2. 内部端口契约（I-CON-03、S1–S16 接缝）

- **S4 canonical orphaning**：`identity.ArticleAccountPort.OrphanArticlesCanonical(ctx,pgx.Tx,string)` → `(CanonicalDeleteArticleResult,error)`；`ArticleAccountPortFunc` 可满足该接口（article-owned 实现可注入，identity 不导入 article）。✅
- **S2 作者状态**：`ArticleIdentityReader.GetAuthorStates(ctx,[]string)` → `(map[string]UserStatus,error)`，批量只读。✅
- **S13/S15 恢复 run authority**：`identity.RecoveryRunLocker` 单方法 `LockRecoveryRunForDeletion(ctx,pgx.Tx,string)`→`(recovery.RecoveryRun,error)`，**不暴露** Transition；identity-owned `RecoveryRunStore` 持有 `Transition` 权威。✅
- **maintenance/readiness Gate**：`RecoveryOpenGate.CheckRecoveryOpenGate(ctx,pgx.Tx) error` 只读。✅
- **S14 恢复删除能力**：`NewRecoveryDeleteCapability` 只接受非空、无前后空白绑定；capability 结构体字段全部非导出（不可伪造/不可变绑定）。✅
- **S16 消费接缝**：`recovery.ArticleDeletionGate` 同时提供 `ReplayAndVerify`/`ReadStatus`，签名 `(ctx,runID,backupRef)→([]byte,error)` 一致；`recovery.Coordinator.Consume` 为唯一协调入口。✅
- **公共错误注册**：17 个 identity/shared code 的 HTTP status 与 retry policy 与 `platform.DefaultRegistry` 逐项一致。✅
- **password-reset durable 内部端口（r40/r18）**：`session.PasswordResetGate.CheckLockedUser(ctx,pgx.Tx,string) error` 为消费者中性端口（无 operation DTO/lineage 枚举/SQL/settlement 字段）；`identity.PasswordResetStore` 同时实现该 Gate 与 `BeginOrClassifyCandidate(ctx,idempotency.Transaction,PasswordResetCandidate)`→`(PasswordResetCandidateResult,error)`；分类枚举闭合为 `created/existing/integrity_hold`；closed cause 为稳定 sentinel `ErrPasswordResetGateClosed`。三消费者（`LoginDependencies`、`PasswordChangeDependencies`、`session.StoreDependencies`）均声明同一 `session.PasswordResetGate` 类型，生产可注入唯一共享实例。✅

### 3. ReplayStatus 严格消费者（I-REC-06～08）

- **Ready20 与唯一 authority 对账**：共享 fixture 恰 20 字段、字段集与 `internal/recovery` 权威一致；`EvidenceVersion==ReplayEvidenceVersionV1`、`Namespace==ArticleDeletionNamespaceV1`；声明 digest、常量 `ReadyFixtureDigestV1`、`CanonicalDigest()` 重算三者相等（`sha256:169e0f…926a7`）。✅
- **canonical 前 16 字段**：4 个 metadata 字段（status/evidence_digest/evidence_ref/completed_at）变化不改变 digest；canonical 字段变化改变 digest。✅
- **Unavailable6 立即停止**：仅 6 字段 `pending/unknown/failed` 三状态接受，且 **零 digest 调用**。✅
- **闭合 schema fail closed**：extra/missing/null/错类型/未知版本/count 不变量破坏对 Ready 与 Unavailable 都返回唯一 `ErrInvalidReplayStatus`。✅
- **跨 run/namespace 绑定**：错误 namespace、错误 evidence_version 稳定拒绝。✅

## 验证证据

**本卡专用命令**（`go test -tags=contract ./test/contract/identity`）：

```
ok  	han/test/contract/identity	0.081s   （19 个顶层测试，全部子例 PASS）
```

`-race -tags=contract ./test/contract/...` → `ok 1.161s`。

**Manifest 七项（固定顺序）**：`gofmt -l .`（空）、`go mod verify`（all modules verified）、`go vet ./...`（空）、`go build ./...`（空）、`go test -tags=contract ./...`（无 FAIL）、`go test -race ./...`（无 FAIL）、`go test ./...`（无 FAIL）。全部 PASS。

**五文件检查**：`gofmt -l test/contract/identity/` 空。所有文件均带 `//go:build contract`，标签约束下无外部依赖（不连 DB/服务/浏览器），符合 `docs/testing.md` 约定。

## Findings

| ID | 严重度 | 描述 | 状态 |
|---|---|---|---|
| IDENTITY-S1.5-R01-F001 | INFO | 本卡为作者自审口径，非非作者独立复审 | 已知限制 |
| IDENTITY-S1.5-R01-F002 | LOW | router 404/405 的**实际组装**断言在 `cmd/han-server`（S4.1/S5.2 已关闭并接线），本卡只固定其依赖的公共 envelope 合同 | 由既有 router 关闭证据承接 |
| IDENTITY-S1.5-R01-F003 | LOW | 真实 PostgreSQL 多连接/commit-ack-loss/migration 混跑（I-PRD-04/05/08～17 的真实 PG 实例）由 `identity-auth-S1.6` 承接 | 转交 S1.6 |

## 结论

**Code Review Gate: PASS（作者自审口径）** — I-CON-01～03、I-REC-06～08 与 r40/r18 要求的 password-reset durable 内部端口/三消费者/Gate 分类均落到精确五文件；专用命令、race 与 Manifest 七项全部 PASS；无 HIGH/MEDIUM finding。F002/F003 属本卡关闭条件中「真实 PostgreSQL 并发/真实 router 组装」的**独立后续任务**（`S1.6`、既有 router 关闭证据），不构成本卡代码级阻塞。

**未验证边界**：真实 PostgreSQL 实例、真实部署、浏览器/E2E 与 production readiness 未验证。本 PASS 弱于非作者独立复审。
