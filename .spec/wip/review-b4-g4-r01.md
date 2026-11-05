# B4 G4-batch 独立代码评审报告 r01

- 日期：2026-10-23
- Reviewer：Ampere（`add-reviewer`，非三项任务实现作者）
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / 基线 HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 批次：B4
- 任务：
  - `identity-auth-S3.2`
  - `identity-auth-S4.3`
  - `article-management-S1.2`
- 写入边界：本报告是本轮唯一写入；未修改代码、Tasks、Plan、设计、既有报告或依赖，未执行 stage/commit/push/PR/merge、部署、E2E、清理或 B5 工作。

## 1. 最终结论

**B4 G4-batch Gate：PASS。**

两轮独立检查未发现 CRITICAL、HIGH、MEDIUM 或 LOW finding。三张卡的 task-level PASS 不仅被汇总，而且已沿批次内真实公共合同、依赖、生产消费者和恢复/文章后续接缝交叉核对：

1. S3.2 的登录生产装配、严格来源解析、Argon2 进程级 Gate、恢复登录模式、路由/Origin 顺序、公共错误和资源生命周期没有与 S4.3 合成恢复边界冲突。
2. S4.3 明确保持 synthetic-only、production forbidden 和 unknown fail-closed；不会把 fixture、公开测试 key、未知恢复责任或未实现的 Anchor/key-retirement 能力误作生产能力。
3. article S1.2 的正交状态、三类投影、cursor、强 ETag 和公共错误合同与 identity `session.Principal`、平台错误 registry 及恢复中的 article evidence 后续责任一致，没有提前实现或覆盖后续 article 恢复任务。
4. 三卡写集互不越权；S3.2 对历史 Argon2/platform 文件的后继修改未改变 S4.3 或 article 的所有权，S4.3 fixture 未写生产状态，article S1.2 未接管 identity、recovery 或生产路由装配。

该 PASS 只解除当前 B4 的独立批次 Review Gate。它不表示 `identity-auth-S2.2`、`identity-auth-S3.5`、`identity-auth-S3.6`、部署、真实恢复、真实 E2E、production observability 或 production readiness 已完成，也不自行启动 B5A/B5B/B5C。

## 2. 固定对象与修订

评审开始与结束均逐一计算 SHA-256；29 个对象及三份固定证据全部匹配，未发生 revision drift。

### 2.1 `identity-auth-S3.2`

| SHA-256 | 文件 |
| --- | --- |
| `5B8B012A3F855C98DBBB10AA34050E15320615F3B36321A84B850B87990B2D36` | `internal/identity/handler.go` |
| `8DC48D994C7B967FC586B4F7C4B70C6A99B9C3EF9DFD05BA018D278B086AA28F` | `internal/identity/service.go` |
| `762843D10B5EE95E7D876C0C0857EE523307CBEDE028AB5C101389AB422147D2` | `internal/identity/login.go` |
| `8468CCA9405C4EF614BFA940A06AD38D7EC269180EB4051F443D3E7D6DE46256` | `internal/identity/login_test.go` |
| `1D2DA4502D640689767D1C9D8D013576E73A266CD4D649F1044CBB23EC1C0DE3` | `internal/identity/login_integration_test.go` |
| `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` | `internal/platform/http_security.go` |
| `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` | `internal/platform/http_security_test.go` |
| `4A3CC84CE6B620724692D696B705D74BC39C8E1EB28F044EDD136E166FF6138C` | `internal/audit/store_integration_test.go` |
| `E6D2570BE3F842C8C2841723067B16123D470BED0F789C089DB14C81C0901450` | `cmd/han-server/main.go` |
| `346C56C57FE09E4528C8EE01BFAAD8BF71947E57462DAED7DA750E1EDB91367C` | `cmd/han-server/main_test.go` |
| `182485F059254AD2541F4A97F7C05AC6C20E857B475B9896CB43C0703260F082` | `cmd/han-server/lifecycle.go` |
| `9F68060F66EFE480D1AD27422F88753485C12024CE0E9953D3EF7876031FFC0F` | `cmd/han-server/lifecycle_test.go` |
| `517B232D76C877FF335354E95668EC1D84B84385C017FCD45A4C046C6E4BC33A` | `internal/platform/metrics.go` |
| `FE97076376DBBABAA5B256BEFB92DAB527372AA652500F25EE0246B0D572EFB4` | `internal/platform/metrics_test.go` |
| `BF6B6E63D315832001A010C77FCBA2341C2E1FDBEA1E0D6471B15BC3B3D406B7` | `internal/security/argon2.go` |
| `BEE56CE85531BDAC700B9B5A1A8E6A88CAAA48102BADD6708D8E00CB35257A33` | `internal/security/argon2_test.go` |
| `99B1094A6C15A6E86E45B43928F8AC1B4F3A4EE810857BE598E9AA4336D968E3` | `internal/platform/errors.go` |
| `5C8CEA67B738BE1AA11FB4AE206907604BEE7CC2CC0C646891CF4EF35A6D9237` | `internal/platform/errors_test.go` |

### 2.2 `identity-auth-S4.3`

| SHA-256 | 文件 |
| --- | --- |
| `73D3E03B2C5CCD0922142413E4CE742973DCB5F370DC3EF0A756148C4340F69F` | `docs/runbooks/database-recovery.md` |
| `38C3171B96BECBB09384400D695F8F3F85A5D7F042C031D7D34CEF33E47BB744` | `testdata/recovery/README.md` |
| `CB398B9E736694FEC77AA21BB08E056933591774BB3842046E63D710DD9212F0` | `testdata/recovery/keyring-fixtures.json` |
| `2F267BD497FC90D7B6D1B28F36F8E7B8B253F8E36153E5F50400BB2AC6DF856E` | `testdata/recovery/backup-manifest.json` |
| `0CD7471AD1574D05465CA3F29C8B967B631F520A2F3BA94E91BE5E346A485ACB` | `test/integration/recovery/fixtures_test.go` |

### 2.3 `article-management-S1.2`

| SHA-256 | 文件 |
| --- | --- |
| `E087DC0C4C6FB6CBA13E7112B09773F3BBDA2D123B2FC885637C4711879D5035` | `internal/article/policy.go` |
| `3DE707FA4B30996438E3E14E0ACC2C0AEA4F19427B521F8BCC31FB6594ABBFC6` | `internal/article/model.go` |
| `6DEC7643381CCE5D88D53E68A0963C48F9A45B0CFCFDC40993643071F2614F93` | `internal/article/handler.go` |
| `407FD7E67F8C43E44C262A5C20CC45DF1B41144DF86C011A4473F0681DAB937C` | `internal/article/policy_test.go` |
| `D73AB43280EC69FF9A2B2415ED48137157E092870D367F7C31FD96A0AA5514A8` | `internal/article/cursor_test.go` |
| `5135568F014CE4D1420ACC5A7DA8F6CED1F3A18318549C03FFD3D0FAAFB6652E` | `internal/article/etag_test.go` |

### 2.4 固定台账与既有评审证据

| SHA-256 | 文件 |
| --- | --- |
| `F34832660D3175321E6C54FBCBAAB9825629C91A10F47362980228E7ADB55D04` | `.add/project/knowledge/tasks/tasks.md` |
| `252D07F78E2F931C48AF026D7B584B10CA3B64186C35FF577582DF6603FEADA8` | `.add/project/knowledge/plans/ir-01-content-community-batches.md` |
| `0739133B9674B9B03EE76F0E4D0D0D3F81F6BA3D3AA1A40F7A7DEA057DB6EE67` | `.spec/wip/identity-auth/review-code-s3.2-f005-r06.md` |

## 3. 实际读取依据

完整读取并适用：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`
- `.codex/profiles/go-service/README.md`
- go-service 的 coding、security、observability、tracing、no-hardcode 适用规则
- Tasks 三张任务卡、Plan 的 B4/Context Pack 与 G4-batch Gate
- 上述29个固定对象全文
- S3.2 r06 及 identity design r26/test-plan r08/architecture/ADR 中与当前生产装配直接相关的合同
- article design/test-plan 中状态、投影、cursor、ETag、错误合同
- 恢复 fixture 的 runbook/README/JSON、`internal/recovery` 的 Keyring/Journal/RecoveryRun 直接合同
- 直接消费者：`session.Principal`、platform error registry/JSON writer、identity composition/router、恢复状态/evidence 接缝

未进行无界全仓泛审。

## 4. Round 1：29 个交付对象直接检查

### 4.1 S3.2

- 核对 startup 配置、生产依赖准备和业务组合路径：`cmd/han-server/main.go:197-903`。
- 核对单实例 gated Argon2 组合：`internal/security/argon2.go:91-121`，及 login fake/known/unknown/temporary/permanent/recovery consumers。
- 核对严格 RemoteAddr/XFF 来源、登录 Handler/Service、统一错误、审计/session/transaction 路径。
- 核对 `Trace → local router → 404/405 或 OriginGuard → exact POST`，以及 429 capacity 与 abuse Retry-After 相互独立。
- 核对资源释放、错误 cause 外部去敏、日志/metrics 标签、无 secret/DSN/原用户名泄漏。
- 复核 r06 中 F001～F005 均 CLOSED 的固定 revision；未发现使历史 finding 重新打开的新证据。

### 4.2 S4.3

- `docs/runbooks/database-recovery.md:3-75` 明确该成果不是生产执行手册或生产就绪证明，unknown 阻断、maintenance 优先、当前独立 Anchor 只读预检、禁止 restore/reset/bootstrap/rollback、同一 run/backup/key/evidence 绑定和 owner-confirmed cleanup 顺序一致。
- `testdata/recovery/README.md:1-87` 明确公开 synthetic key/input 不是秘密且 production forbidden，清理责任继续 unknown。
- 两份 JSON 固定一个 active v3、两个 verify-only 历史 key、三组同输入 HMAC、合成 backup/run、UTC 微秒、未包含真实备份和完整 unknown Gate。
- `test/integration/recovery/fixtures_test.go:23-608` 使用严格 JSON 解码、重复字段/unknown/trailing 拒绝、HMAC 复算、跨版本 input drift 拒绝、run/key/fingerprint 绑定、入口 Gate、恢复步骤和 Anchor 只读约束；测试没有 `Skip` 或假 PASS。

### 4.3 article S1.2

- `internal/article/policy.go:59-97` 对 publication/violation/deletion 三组正交状态及关联字段 fail closed。
- `internal/article/policy.go:101-199` 分离成员、作者、管理员投影；不可见资源统一不可用，普通投影不泄漏治理字段。
- `internal/article/policy.go:201-287` 的 cursor 严格绑定版本、view、UTC time 和 ID；强 ETag 拒绝 weak、`*`、multi-value、非规范版本。
- `internal/article/handler.go:73-225` 只允许 article/shared allowlist 公共错误穿越边界，未知错误转内部错误；响应使用统一 JSON/trace/nosniff。
- 三组测试覆盖状态/投影、cursor 和 ETag/公共错误主要失败矩阵，未发现以 mock 或构建成功冒充合同。

## 5. Round 2：真实消费者与批次整体一致性

### 5.1 identity → production composition

- `runComposed`、`prepareProductionLoginDependencies`、`newProductionBusinessHandler` 和 `newLoginBusinessRouter` 构成真实生产路径，不是 fixed-fail 或 test-only port。
- recovery-bootstrap 只影响显式维护模式登录；普通/error/unknown 状态 fail closed，不把 S4.3 fixture 当运行配置或 credential。
- startup config、Argon2 Gate、route/Origin/source、metrics/observer 和 lifecycle 的所有权仍属于 S3.2；S4.3 与 article 没有重复注册或共享可变全局状态。

### 5.2 S4.3 → recovery 直接合同

- fixture 的 key state、HMAC 指纹、UTC 微秒和 run 绑定可由现役 recovery Keyring/Journal/RecoveryRun 表示。
- manifest 中 article evidence 明确 `not_included` 并指向后续任务；不会让当前 B4 越过 article replay producer、CAS Anchor 或 key-retirement Gate。
- unknown 值和 synthetic artifact 缺失会阻止真实演练；没有生产路径、凭据、真实用户数据或 provider 配置。

### 5.3 identity/platform → article 公共合同

- article 消费 `session.Principal` 的现役访问语义，不自行解析 cookie/session，也不弱化 identity 状态。
- article 自有错误 registry 与 `platform.DefaultRegistry` 的共享代码分工清晰；未知跨域错误 fail closed。
- S3.2 当前只装配 login 路由不构成 article S1.2 缺陷：S1.2 的任务目标是公共策略/表示/cursor/ETag 基础，article 纵切和生产路由属于后续任务。

### 5.4 文件与责任隔离

- 三卡 29 文件零交叉；没有第30个批次对象、新依赖、schema/migration 或公共 ADR 变化。
- S4.3 只提供合成恢复输入，不实现 S4.5/S4.6。
- article S1.2 没有提前接管 create/query/update/deletion/replay。
- Tasks 与 Plan 对三卡 CLOSED、B4 待 G4-batch、后续 B5/S2.2/S3.5/S3.6 限制的描述一致。

## 6. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

无。

### LOW

无。

没有未关闭阻断 finding。

## 7. 待决策与非阻断建议

### 待决策

当前 B4 代码 Gate 内无新增待决策项。恢复 runbook 中的 RPO/RTO、容量、责任、retention、生产 Journal/Anchor/key-retirement、目标 POSIX 证明等仍按合同保持 `unknown`，由后续 owner/任务决定；本评审未替项目填写默认值。

### 非阻断建议

后续若修改 S4.3 fixture，可考虑给 `cleanup`、`article_evidence` 和 `backup.data_classes` 增加逐字段精确断言，进一步缩小文档/fixture 漂移空间。当前固定 JSON 已人工核对，现有任务关闭条件也未要求这些字段全部机械等值断言，因此不构成本轮 finding。

## 8. 实际命令与结果

以下为 Reviewer 本轮独立执行，不将 Lead 结果冒充自跑：

| 命令 | 结果 |
| --- | --- |
| 32 项 `Get-FileHash -Algorithm SHA256` + branch/HEAD 核对 | PASS；`HASH_OK count=32`，`main`，HEAD 匹配 |
| `go test ./internal/identity -run 'TestLogin\|TestCredentialMode\|TestLoginAbuse' -count=1` | PASS |
| `go test ./cmd/han-server ./internal/identity ./internal/platform ./internal/security -count=1` | PASS |
| `go test -race ./cmd/han-server ./internal/identity ./internal/platform ./internal/security -count=1` | PASS |
| `go test ./internal/article -run 'TestVisibility\|TestCursor\|TestETag\|TestArticleErrors' -count=1` | PASS |
| `go test ./internal/article -count=1` | PASS |
| `go test -race ./internal/article -count=1` | PASS |
| `go test -tags=integration ./test/integration/recovery -run '^TestRecoveryFixtures$' -count=1` | PASS |
| `go test -race -tags=integration ./test/integration/recovery -run '^TestRecoveryFixtures$' -count=1` | PASS |
| `gofmt -l`（29 对象中的全部 Go 文件） | PASS；无输出 |
| `rg -n 'TODO\|FIXME\|time\.Sleep\|t\.Skip\|fmt\.Print\|log\.Print'`（29 对象） | PASS；零命中 |

Lead 记录的 Manifest 全套 Gate 和 PostgreSQL 18.6 三组 integration PASS 作为上游证据核对，但 Reviewer 本轮没有再次加载或输出 DSN，也没有独立复跑 PG18。

## 9. 未验证范围与限制

1. 未独立复跑 PostgreSQL 18.6 登录 prepared-audit、row-lock、identity+audit 集成矩阵；采用 Lead 固定 revision 的现场 PASS 证据。
2. 未执行真实数据库恢复、真实备份、真实 Journal 持久卷、目标 POSIX owner-only/atomic replace/fsync/flock、独立 CAS Anchor 或 key-retirement。
3. 未执行目标部署环境 Argon2 benchmark；S2.2 仍独立未完成。
4. 未执行 production observability backend、部署、告警演练或真实 E2E；S3.5/S3.6/S3.4 仍独立未完成。
5. 共享工作树包含其他未提交内容；本结论只适用于上列固定 SHA-256 对象，不可外推到后续漂移 revision。

## 10. Gate 与交回

- 三项 task-level 状态：保持 CLOSED。
- `IDENTITY-S3.2-REV-F005`：保持 CLOSED；本轮没有重新打开 F001～F005 的新证据。
- B4 独立批次代码 Review：**PASS**。
- G4-batch：**PASS**。
- 交回 Lead：可据此处理 B4 批次 Gate；是否启动任何后续批次仍需另行授权。
