# identity-auth-S10.2 独立代码评审 R01

- 日期：2026-10-24
- Reviewer：非作者 add-reviewer
- objectType：code
- 当前活动：development-delivery / implement 后 review
- 原作者：Backend-Dev Bacon `01a16ec3-1d9a-7920-b014-4d38a44b5b31`
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：**Code Review Gate: CHANGES REQUIRED**

## 1. 结论边界

本轮只评审 S10.2 八文件内的 durable schema、identity-owned Store、TemporaryCredential/Idempotency 增量及确定性测试，不评价 S10.3 三消费者 Gate、S10.1 coordinator、S10.4 worker/composition 是否已经实现。

发现 4 个 HIGH 和 1 个 MEDIUM finding。前四项直接破坏 S10.2 的 durable fail-closed、原子性、lineage 或升级前置合同，阻断当前任务关闭。定向测试通过不能替代这些未被测试覆盖的持久化约束。

本报告：

- 不关闭 `identity-auth-S10.2`；
- 不启动 `identity-auth-S10.3`；
- 不关闭或改变既有代码 finding `IDENTITY-S10.1-REV-R01-F002`；
- 不表示真实 PostgreSQL、migration、multi-connection、部署或 production readiness 已通过。

## 2. 固定对象与开始哈希

评审开始时八个固定对象均与用户给定 SHA256 一致：

| 对象 | SHA256 |
|---|---|
| `migrations/0008_password_reset_durable.sql` | `54889C465469C724626BF89AC5220C5E8210A734C78FBDE963B004735AEA49E6` |
| `internal/identity/model.go` | `68E76D2FF2D101CC0EB0510CDB852F7542ACA7B53C6947FAE41E6ED3BD0C9A0E` |
| `internal/identity/temporary_credential.go` | `66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5` |
| `internal/identity/temporary_credential_test.go` | `659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D` |
| `internal/identity/password_reset_store.go` | `33892D16016152DE770B2E2C4C81EBAEE1DBB245B31072EB2C4DEA822E358CF1` |
| `internal/identity/password_reset_store_test.go` | `69477384E86908A9599E25FCBFC4D68837B6AD140FF350F865834FC9D298B535` |
| `internal/idempotency/store.go` | `93D7595870F48B9D8C6EBD3E7D5646A5FC92A5AB5E5605AFB2E2CF596056257C` |
| `internal/idempotency/store_test.go` | `AA31398FF4CBF9C2C304CC2617095B3353C8C1B11952EDE69B1BEEBBC5C95E45` |

八文件当前均为共享工作树中的未跟踪对象；评审未据此要求全树干净，也未修改这些文件。

## 3. 两轮实际检查

### Round 1：固定对象直接检查

完整检查八个固定文件，重点包括：

1. `0008` 的 preflight、backfill、六字段 FK/UNIQUE、User/operation/credential/job/session lineage、部分唯一索引、CHECK 与 trigger；
2. `PasswordResetOperation`、RecoveryJob、delivery/settlement/terminal 状态模型；
3. `BeginOrClassifyCandidate` 的 User-first 顺序、same-key 分类、orphan/hold、resend lineage、N+1 分配与五次写入；
4. Idempotency 六字段 claim 的构造、插入与既有记录分类；
5. 测试是否证明行为、错误路径、原子性和 migration 拒绝矩阵，而非只检查字符串存在。

### Round 2：有界真实消费者与上游深审

按允许范围检查：

- S10.2 task 完整卡；
- design r39 的 §3.3.11/12、§3.4.1/2/8、§9.4；
- test-plan r16 的 I-PRD-02～05、I-PRD-16～18；
- accepted ADR-0023 与 planning review R02；
- `migrations/0002_identity.sql`、`0004_idempotency.sql`、`0007_idempotency_response_snapshot_v1.sql`；
- migration runner/static tests；
- 现役 `internal/identity/password_reset.go` 的独立 claim writer、scope 生成与事务边界；
- TemporaryCredential 和 Idempotency 的直接调用/数据定义。

未扩成全仓泛审。S10.3、S10.4 和未来 S11.1 仅作为既定消费者边界核对，没有把“尚未接入”登记为本卡缺陷。

## 4. Findings

### [HIGH] 五对象全有或全无没有形成 commit-time 不变量

- finding ID：`IDENTITY-S10.2-REV-R01-F001`
- 对象/修订：S10.2 固定八文件
- 位置：
  - `migrations/0008_password_reset_durable.sql:16-25`
  - `migrations/0008_password_reset_durable.sql:166-182`
  - `internal/identity/password_reset_store.go:214-240`
  - `internal/identity/password_reset_store_test.go:331-387`
- 现状：
  - operation 通过六字段 deferred FK 指向 IdempotencyRecord，但没有反向约束证明 password-reset claim 必有 operation；
  - `users.password_reset_execution_generation` 只有非负/单调约束，没有 commit-time 约束证明 counter=N 必有同 User、generation=N 的 operation；
  - 因而数据库允许独立提交 password-reset claim-only 或 counter-only/generation gap；
  - `BeginOrClassifyCandidate` 先推进 counter，再依次插入 operation、credential、job、claim。Store 只接收不含 Commit/Rollback 的 caller-owned `Transaction`；后续 helper 若以成功 SQL + 非 1 行结果返回应用级冲突，接口本身不能阻止调用方误提交此前写入；
  - 当前错误注入测试只证明遇错停止继续调用，不尝试 commit，也不证明 rollback 或数据库拒绝部分状态。
- 违反依据：
  - design r39 `design.md:2093-2095,2219,2253,3399`；
  - test-plan r16 I-PRD-04；
  - S10.2 task 的“五对象全有或全无、counter-only/claim-only 不可达”关闭条件；
  - code review checklist 的“失败路径不留下半完成状态”。
- 触发：
  1. counter UPDATE 成功，后续写返回应用级 `ErrPasswordResetStateConflict`，调用方错误地提交仍可提交的 transaction；或
  2. purpose-specific `InsertPasswordResetClaim`/旧 writer 单独提交 claim，而没有 operation/credential/job。
- 影响：产生 generation gap 或孤立 claim；后续 same-key 分类只能进入 hold/不确定状态，账号认证能力被长期 fail closed，且“不连续代次”已成为持久事实。它不满足本卡要建立的 durable authority。
- 修复建议：在八文件边界内建立可核证的 commit-time 全有或全无机制，使 counter-only、claim-only、operation-only 均不可提交；并使 caller-owned transaction 的错误结果不能被误当作可提交成功。若实现必须改变已确认的 caller-owned transaction 设计或扩大文件/公共合同，停止并交 Lead 返回设计/Task-Planner 对齐，不由作者自行改架构。
- 复审条件：
  - 对每个写点注入失败后实际尝试 commit，证明只能全回滚或 commit 被约束拒绝；
  - 独立覆盖 counter-only、claim-only、operation-only；
  - 保留 User-first 和六字段 identity，不以应用约定代替数据库/事务证据；
  - 定向普通/race 测试通过，真实 PG18 约束实证仍由 S1.6 独立完成。

### [HIGH] operation 的 authority/delivery/settlement 组合未封闭，坏终态可被 exact replay 接受

- finding ID：`IDENTITY-S10.2-REV-R01-F002`
- 对象/修订：S10.2 固定八文件
- 位置：
  - `migrations/0008_password_reset_durable.sql:124-159`
  - `migrations/0008_password_reset_durable.sql:192-261`
  - `migrations/0008_password_reset_durable.sql:384-430`
  - `internal/identity/password_reset_store.go:597-643`
- 现状：
  - settlement、delivery 和 operation state 只分别校验枚举及各自字段，缺少三组之间的合法组合约束；
  - 数据库可接受例如 `state='succeeded' + settlement_state='pending'`、terminal failed/invalidated 配 accepted delivery、prepared 配非 `not_started` delivery 等相互矛盾的权威证据；
  - terminal trigger 只禁止 terminal `state` 改值，未冻结 terminal 后的 settlement、delivery result/evidence、terminal time 等安全字段；
  - same-key 的 `existing.exact` 只核六字段和关系存在，不核 operation authority、settlement、delivery 的合法组合，因此上述坏行可返回 `PasswordResetCandidateExisting`，而不是进入 hold。
- 违反依据：
  - design r39 `design.md:2109-2112,2216-2217,2407-2410`；
  - test-plan r16 I-PRD-03/I-PRD-05；
  - operation 是唯一安全权威、未知/非法状态必须 fail closed 的 ADR-0023 合同。
- 触发：worker/受控命令、手工修复、缺陷 writer 或历史坏数据写入一个各列局部合法、组合非法的 operation，随后同 key 重试。
- 影响：不一致的安全权威可被稳定 replay；后续 Gate/worker 可能对“已成功但未 settlement”或“已失败但邮件已 accepted”等状态作相反判断，导致错误能力激活、错误清 Gate 或永久卡住。
- 修复建议：
  - 以 CHECK/受控转换约束封闭 authority、delivery、settlement、terminal/recovery 字段合法组合及单调性；
  - terminal 后冻结安全证据，除明确设计允许的受控转换外不得改写；
  - exact replay 分类必须验证权威状态组合，任何漂移进入 integrity hold；
  - 测试必须执行状态矩阵/分类行为，不得仅搜索 SQL 片段。
- 复审条件：补齐 DB 拒绝矩阵、terminal 不可变矩阵和 exact replay 坏状态进入 hold 的确定性测试；不得把 generic Idempotency 状态提升为安全权威。

### [HIGH] resend successor 未证明当前 User 能力，lineage 漂移也未进入 integrity hold

- finding ID：`IDENTITY-S10.2-REV-R01-F003`
- 对象/修订：S10.2 固定八文件
- 位置：
  - `internal/identity/password_reset_store.go:170-177`
  - `internal/identity/password_reset_store.go:674-692`
  - `internal/identity/password_reset_store_test.go:390-462`
- 现状：
  - resend 查询只证明 predecessor 同 User、`state=succeeded`、settlement 属于两个值，以及指定 replaced credential 来源于 predecessor 且状态为 active/expired；
  - 未核锁定 User 的当前 `auth_generation`、`credential_mode`，也未核 predecessor settlement/target generation 与当前 User 权威能力一致；
  - 由于 F002 所述组合未封闭，查询还可能接受 `permanent_successor` 与 active/expired replaced credential 并存的矛盾 predecessor；
  - 查询返回 false 时统一返回 `ErrPasswordResetGateClosed`，没有把 terminal predecessor/current credential 漂移设置为 integrity hold；
  - 唯一 resend 测试让 stub 直接返回 `true`，没有构造合法/非法 current lineage 行为矩阵。
- 违反依据：
  - design r39 `design.md:2092,2095,2144-2152,2254,2258`；
  - test-plan r16 I-PRD-03/I-PRD-05；
  - S10.2 task 的“generation/lineage 合法 successor、坏绑定与 integrity hold”合同。
- 触发：调用方提交历史 predecessor/replaced ID；目标 User 已推进 auth generation、切换永久能力，或 predecessor/credential settlement 漂移，但表中该 credential 仍呈 active/expired。
- 影响：可从非 current 能力创建 N+1 successor，或把应进入 hold 的完整性故障降级成普通 GateClosed。前者破坏能力 lineage，后者丢失持久故障标记与受控恢复责任。
- 修复建议：在已锁 User 下核对 current credential mode/auth generation、predecessor target/settled generation、exact replaced credential 和合法 terminal settlement；区分普通资格拒绝与完整性漂移，后者在同事务设置 hold。补合法 successor、历史/stale predecessor、跨代、settlement 漂移和 hold 写入测试。
- 复审条件：非法 terminal predecessor/current credential 组合稳定进入 hold且零新 counter/claim/operation/credential/job；合法 successor 仍使用新 execution generation，same-key replay 不增代。

### [HIGH] migration preflight 没有阻断现役 password-reset Idempotency 遗留状态

- finding ID：`IDENTITY-S10.2-REV-R01-F004`
- 对象/修订：S10.2 固定八文件
- 位置：
  - `migrations/0008_password_reset_durable.sql:1-14`
  - `internal/identity/password_reset_store_test.go:21-59`
  - 真实消费者证据：`internal/identity/password_reset.go:231-300,1027-1035`
- 现状：
  - preflight 只检查 legacy `password_reset` 的 pending/active/expired TemporaryCredential；
  - 没有检查现役 password-reset `processing/uncertain/succeeded` IdempotencyRecord，也没有为不可解释 claim 提供迁移前阻断证据；
  - 当前代码确实以独立事务先提交 password-reset claim，且 scope 由 actor、operation、target 构成，因此升级现场存在设计明确要求处理的真实遗留生产者；
  - migration 静态测试只搜索表名/约束片段并确认 runner 能加载 version 8，没有验证 preflight 的拒绝样本。
- 违反依据：
  - design r39 `design.md:2218,2221,3405`；
  - test-plan r16 I-PRD-16/I-PRD-17；
  - S10.2 task 的 migration preflight、停止旧 writer/reader、不可安全回退合同。
- 触发：升级前数据库存在旧 coordinator 已提交的 password-reset processing、uncertain 或 succeeded claim，但对应 durable operation/job 从未存在；随后直接应用 `0008`。
- 影响：migration 成功后留下无法回填的 claim-only 历史；same-key 请求会被判完整性故障并使目标账号进入 hold，升级没有在写入新 schema 前暴露并要求受控处理该风险。
- 修复建议：增加可执行且 fail-closed 的 preflight/受控前置证据，覆盖现役 password-reset claim 状态和其他设计列明的不可解释数据；若 SQL 内无法安全识别 HMAC scope，必须由明确、可审计、在 migration/writer 开放前运行的前置工具或证明承担，不能静默略过或猜测 predecessor。
- 复审条件：静态/确定性测试覆盖 normal legacy、processing、uncertain、succeeded claim 与不可解释 current credential；坏样本阻断升级，正常样本可前向迁移。真实 PostgreSQL 18 与混跑/回退实证继续归 S1.6。

### [MEDIUM] 两个安全关键函数超过 Go profile 的强制 50 行上限

- finding ID：`IDENTITY-S10.2-REV-R01-F005`
- 对象/修订：S10.2 固定八文件
- 位置：
  - `internal/identity/password_reset_store.go:58-247`
  - `internal/identity/password_reset_store.go:293-362`
- 现状：`BeginOrClassifyCandidate` 约 190 行，`readPasswordResetCandidateBinding` 约 70 行；两者均超过 go-service profile `rules/coding-standards.md:14-18` 的强制 50 行上限。
- 触发：修改 same-key、hold、resend 或五对象写入任一分支时，需要在一个长函数中同时推理锁序、分类、状态和错误返回。
- 影响：增加遗漏 fail-closed 分支、错误复用 generation 或把完整性故障降级为普通错误的维护风险；当前 F001～F003 已显示这些分支缺少清晰的可独立验证边界。
- 修复建议：按“锁 User/派生 identity → existing/orphan 分类 → new candidate Gate/lineage → 构造 → 原子写入”拆分小函数；保持现有 User-first 顺序和单一事务，不为一次性路径引入通用框架。
- 复审条件：适用函数均满足强制复杂度规则，定向普通/race 测试不退化，拆分不改变公共合同或写入顺序。

## 5. 已确认未发现的直接回归

在本轮固定范围内：

- 六字段 claim 的 Go 构造与插入参数包含 `scope_hash,key_locator,key_digest,digest_key_version,request_digest,execution_generation`；
- 新候选先锁 User，再查 claim/operation，再执行写入，没有发现先 Operation 后 User 的直接倒序；
- same-key 同摘要路径不会推进 execution generation；同 key 异摘要沿现役 Idempotency conflict 返回；
- 八文件没有改动公共 REST handler、权限矩阵或公共错误注册表；
- 未发现凭据、HMAC key、临时密码或 DSN 硬编码/日志输出；
- `0001`～`0007` 未被本轮对象修改。

这些局部确认不抵消 findings，也不证明未来消费者/worker 已接入。

## 6. 待决定

无新的业务或 ADR 方向待决定。ADR-0023 方案 A 和 design r39 的 caller-owned transaction、User-first、operation authority 方向继续有效。

若 F001 的安全不变量无法在当前八文件和既定内部事务合同内实现，或修复要求改变公共 REST/错误、accepted ADR 核心、文件所有权或新增依赖，应由 Lead 停止返工并返回设计/Task-Planner；Reviewer 不替作者选择新架构。

## 7. 非阻断建议

无独立于 findings 的非阻断建议。不要以增加更多字符串包含测试替代行为、约束与 commit 结果测试。

## 8. 实际命令与结果

以下命令均在仓库根目录执行，未读取或使用 `HAN_TEST_POSTGRES_DSN`：

| 命令 | 结果 |
|---|---|
| `git branch --show-current` | `main` |
| `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 八对象 `Get-FileHash -Algorithm SHA256` | 开始哈希全部匹配 |
| S10.2 task 定向 `go test` | PASS |
| identity/idempotency 定向 `go test -race` | PASS |
| 八对象 Go 文件 `gofmt -l` | PASS，无输出 |
| `go vet ./internal/migration ./internal/identity ./internal/idempotency` | PASS |
| `go mod verify` | PASS，`all modules verified` |
| `go build ./...` | PASS |
| 八对象 scoped `git diff --check` | PASS；这些对象为 untracked，Git 不检查其内容，格式由 `gofmt`、编译与测试补充 |
| 静态函数行数检查 | `BeginOrClassifyCandidate` 和 `readPasswordResetCandidateBinding` 超过 50 行 |
| migration 静态关键不变量检查 | 未找到 Idempotency preflight、跨 authority/settlement CHECK、terminal delivery immutable、counter 反向不变量 |

未复跑 Lead 已报告的全量 test、全量 race、contract；本轮定向命令和静态审查足以复现当前 findings。

## 9. 未验证限制与剩余风险

本轮按授权未运行：

- 真实 PostgreSQL 18；
- `0008` 在 legacy/异常数据上的实际 apply/rollback；
- deferred constraint 的真实 commit 行为；
- 多连接锁序、并发 claim、deadlock/serialization；
- commit-ack-loss；
- integration tag；
- SMTP/TLS、部署、Host、E2E、容量或 production readiness。

因此不能对 SQL 在 PG18 的实际可执行性、查询计划、真实 transaction abort 状态或混跑阻断给 PASS。真实环境限制不降低本轮从静态 schema、Go 控制流和直接消费者得到的 findings。

## 10. Gate 与复审条件

**Code Review Gate: CHANGES REQUIRED**

复审前至少需要：

1. `IDENTITY-S10.2-REV-R01-F001`～`F004` 全部修复并提供相应失败先行、行为/约束证据；
2. `IDENTITY-S10.2-REV-R01-F005` 修复，或由有权责任人针对强制 profile 规则作有效处置，不能由作者自行忽略；
3. 固定八文件形成新的明确 revision/hash，原作者完成定向普通/race、适用 Manifest 检查和 scoped diff-check；
4. 原 Reviewer 对原 finding ID 做独立定点复审并检查新增风险；
5. Lead 在真实共享工作树复核对象、所有权、实际 diff、未验证边界后再决定后续动作。

在此之前，S10.2 不应 task-level 关闭，S10.3 不应启动。
