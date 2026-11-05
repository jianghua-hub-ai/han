# identity password-reset durable fail-closed Design 定点复审 R02

- 日期：2026-10-24
- Reviewer：add-reviewer（原 R01 Reviewer，非作者独立复审）
- objectType：`design`（architecture + feature design + stories 复合修订）
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审目标：逐项复审 R01 F001～F005，并检查修订引入的新 design/architecture 风险
- 结论：**R01 F001～F005 均已按原触发链关闭；发现 1 个新的 Major，Design Review Gate 为 CHANGES REQUIRED**

本报告只评审固定设计对象。accepted ADR-0023 作为未变上游决定只核未被破坏，不重开方案 A。代码层 `IDENTITY-S10.1-REV-R01-F002` 继续独立 Major OPEN；本报告不关闭代码 finding、不解除 B6C/S10.1 阻塞、不启动 B6D，也不声明实现、测试、部署、E2E 或生产就绪。

## 1. 固定对象与漂移前置检查

| 对象 | 修订 / 范围 | 固定 SHA256 |
|---|---|---|
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r26` | `D2FC176D5877AA265B137C78450AE0DC0E9ABE1B45AB0C4DE672CA65D44C352F` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r37` | `E3E54B5788321883284E05D6128ED2073218F1D959B05F837BB360AE253F21EF` |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r13` | `00285EABB587C1ECA876ADFCD5DB552B167BC8B647B6062437198D537CD19A50` |
| `.add/project/knowledge/decisions/identity.md` | accepted ADR-0023 | `2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978` |
| `.spec/wip/identity-auth/review-design-r35-s10.1-r01.md` | 原 R01 报告 | `02729A1559CF8EAB7FB1F6DFB4560DEBBF4ED00C19C9CC0E0F285467F6F96E6D` |

复审开始时分支、HEAD 和五个固定对象 SHA 均匹配，无对象漂移；R02 报告路径此前不存在。

## 2. 两轮实际检查范围

### 2.1 Round 1：固定设计对象与原 findings 定点复审

完整消费 architecture r26、design r37、stories r13、ADR-0023 与原 R01，并按 design 分支检查：

- F001 的首次 claim、operation、exact credential、initial job 原子边界，真实幂等主键与六字段 identity、FK/唯一约束及 crash 验证；
- F002 的统一 User-first 锁序、无 operation 行屏障、三消费者 Gate、Session resolve caller-owned transaction 与 commit 后 Principal；
- F003 的 `auth_generation`、永久密码/credential/Session lineage、settlement、合法 successor、非法能力失效及 integrity hold；
- F004 的 Article deletion + PasswordReset 双 worker composition/lifecycle、共享 deadline、错误优先级、资源关闭与回归范围；
- F005 的 Q1 状态、已接受合同与“实现/验证仍未完成”边界；
- operation 权威、prepared/stale prepared、状态机、dead-letter、迁移/混跑/回退、测试输入和非完成性声明；
- architecture/design/stories 之间是否存在新矛盾、旁路消费者或不必要扩面。

### 2.2 Round 2：真实消费者与上游合同有界追踪

在用户允许上限内静态追踪：

- `internal/identity/password_reset.go`
- `internal/identity/login.go`
- `internal/identity/service.go` 的 PasswordChange
- `internal/identity/bootstrap.go`
- `internal/session/store.go`
- `internal/idempotency/store.go`
- `migrations/0002_identity.sql`
- `migrations/0004_idempotency.sql`
- `migrations/0007_idempotency_response_snapshot_v1.sql`
- `cmd/han-server/main.go`
- `cmd/han-server/lifecycle.go`
- 现役 requirements、identity boundary、ADR-0011/0015/0018/0023

现役代码再次证明修订影响面真实存在：当前 password-reset 仍可先独立提交幂等 claim，Session resolve 仍是无 caller-owned transaction 的查询，PasswordChange/Session schema 尚无新 generation/lineage，production lifecycle 仍只有 Article worker 单槽位。这些只作为消费者证据，不用于判定代码已修复。追踪同时发现 US-IDENTITY-03.11 普通用户 password-reset resend 仍是实际设计消费者，并触发新的 F006。

## 3. R01 F001～F005 逐项复审

### F001 — CLOSED（原 finding 触发链）

**原问题**：operation 与真实 IdempotencyRecord 未原子创建，缺少 `(scope_hash,key_locator)` 与六字段精确绑定。

**关闭证据**：

- `design.md:2070-2080` 新增唯一 `PasswordResetAtomicRepository.BeginOrClassify`，明确禁止外层先独立提交通用 Claim；首次 claim、operation、exact pending credential、initial job 在 User-first 单事务中全有或全无。
- `design.md:2084-2091,2171-2180,2300-2308,2326-2329,2342` 固定真实 `(scope_hash,key_locator)`、六字段候选 UNIQUE/FK、DEFERRABLE 约束、跨 User/坏 identity 拒绝、preflight 和 crash/constraint 矩阵。
- `architecture.md:562,617-619,1348` 已同步同一系统级合同。
- `design.md:2201-2202` 给出 claim-only 不可达、同键/异摘要/异目标和第二封邮件的确定性验证输入。

结论仅关闭 R01 所指“首次 password-reset 执行”的原子 claim 缺口；新发现的 resend 旁路另列 F006，不覆盖本项原失败链。

### F002 — CLOSED

**原问题**：Operation-first 与 User-first 自相矛盾；无 operation 行缺少线性化屏障；Session resolve 可能在 commit 前返回 Principal。

**关闭证据**：

- `design.md:2068` 将 create/claim、Login、PasswordChange、Session resolve、worker、受控命令统一为 User-first，禁止先锁 Operation 再等 User；User 行是 no-operation-row barrier。
- `design.md:2130-2136,2152-2156` 固定三个消费者的事务、锁、失败和提交点；Session 预读不授权，commit 失败绝不返回 Principal。
- `architecture.md:566,662-668,1249,1348` 同步统一锁序、no-row barrier 和 commit 后 Principal。
- `design.md:2203-2205` 固定 no-row、deadlock 与 Session commit barrier 验证。

该合同与现役 Login/PasswordChange 的 User-first 形态可对接，同时明确要求重构当前无事务 Session resolve；未发现原死锁/放行矛盾仍残留。

### F003 — CLOSED

**原问题**：缺少持久 capability lineage/settlement，无法区分合法 successor 与 F002 竞态产生的非法永久能力。

**关闭证据**：

- `design.md:2088-2091,2114,2118-2128` 定义 base/target generation、settlement 单调推进、永久密码/TemporaryCredential/Session root lineage、合法 temporary→permanent successor、后续正常改密/后续 reset/历史 replay和非法能力失效。
- `design.md:2271-2308,2328-2329` 将 generation、lineage、settlement 与 User integrity hold 映射到可迁移字段、CHECK/FK/索引。
- `architecture.md:564,608-610,618,666-667` 同步系统级能力谱系与 hold。
- `design.md:2209-2213` 覆盖 consumed 竞态、合法 successor、后续 reset、历史 replay 与 hold 清除矩阵。

`consumed` 不再被当作安全充分条件；合法 successor 与无法证明来源的能力具有不同的持久表达和处置。

### F004 — CLOSED

**原问题**：production lifecycle 只有 Article deletion worker 单槽位，未定义两个 worker 的共同装配、失败与关闭。

**关闭证据**：

- `design.md:2184-2193` 明确保留两个 worker owner，增加最小 coordinator，固定 Reconcile/Start、任一运行失败触发、Stop 各一次、shared deadline 并发 Wait、主错误优先级、cleanup 聚合、Article 资源与 DB pool 关闭顺序及 Article 回归。
- `architecture.md:576,952,986,1348` 同步双 server + 双 worker 生命周期和退出语义。
- `design.md:2215,3293` 将双 worker 成功/失败/信号、缺 worker fail closed 与现役 Article 回归纳入后续验证和实现影响面。

修订没有用 PasswordReset worker 覆盖当前 `runtime.worker`，并明确承接 `cmd/han-server` 真实单槽位改造。

### F005 — CLOSED

**原问题**：stories 仍把已关闭的 Q1 写成未决实施阻塞。

**关闭证据**：

- `stories.md:133,191-195,352,402` 均明确 Q1 已接受，固定统一改密入口、至少 8 位且同时含字母数字、强制改密 Gate 和 Session 后置。
- 同处持续声明实现与验证未完成，并排除周期轮换、定期过期与密码历史扩面。
- `architecture.md:1352-1354` 不再把 Q1 列为待确认项。

未发现 stories 仍以 Q1 未决阻塞 S02/S04/S10。

## 4. 新 findings

### BLOCKER

无。

### MAJOR

#### IDENTITY-DESIGN-R37-S10.1-R02-F006 — 普通用户 password-reset resend 仍走 ADR-0011 残余窗口，形成 operation 权威旁路

- **位置**：
  - `.add/project/knowledge/design/architecture.md:511,560,610,657,1145,1185,1303`
  - `.add/project/knowledge/design-workspace/identity-auth/design.md:2219-2261,2583,2593`
  - `.add/project/knowledge/design-workspace/identity-auth/stories.md:362-381`
  - 上游 `.add/project/knowledge/requirements/requirements.md:310-322`
  - 对照 `.add/project/knowledge/decisions/identity.md:283,293,303-310`
- **违反依据**：
  - ADR-0023 要求 password-reset 在 SMTP 前建立 durable operation/credential/job，operation 是唯一安全权威，stale prepared 不得由 worker 重发。
  - architecture r26 自身在 `architecture.md:1303` 把 US-IDENTITY-03.11 的关键设计写为“同上”，即继承 US-03.10 的 User-first 原子 claim、operation/job、generation/lineage/Gate；但其他章节又把重发整体放回旧残余窗口。
  - r37 schema 要求 password-reset TemporaryCredential 绑定 `source_operation_id`/target generation，且 operation 与 exact credential 一对一；S11 没有定义新 operation 或安全 successor operation。
- **证据**：
  1. `design.md:2233-2246` 仍按“幂等占位 → 准备下一 generation → SMTP → 成功后提交”的旧流程设计重发，没有 SMTP 前 operation/job、User-first durable barrier、worker settlement 或 hold。
  2. `design.md:2250` 明确该流程同时处理 `purpose=password_reset`，不是仅处理新管理员临时凭据。
  3. `architecture.md:560,657,1145,1185` 把“重发路径”归入 ADR-0011 residual-window 处理；但 `architecture.md:610,662,1303` 又要求 password-reset credential 绑定 operation、operation 为唯一权威、US-03.11 继承 durable 设计，内部合同不一致。
  4. stories S11 仍以 residual window、独立同键重放和 current TemporaryCredential 为主线，未声明 unresolved operation/hold 时禁止重发，也未把新 password-reset credential 纳入新的 operation generation/settlement。
- **触发场景**：
  - 目标用户已有 terminal password-reset operation/current credential，管理员调用 `POST /api/v1/admin/users/{id}/password-reset/resend`。
  - SMTP 已受理，而 S11 的数据库提交结果未知或 cleanup 失败；新邮件秘密可能对应已提交或未提交的新 credential，但没有该次 resend 的 durable operation/job 可证明和收敛。
  - 新 credential 成功提交后需要满足 r37 的 `source_operation_id`、target generation 和 one-to-one exact credential 约束；S11 没有可绑定的新 operation，复用旧 operation又违反 exact credential 一对一和历史 settlement。
  - existing operation unresolved/hold 时发起 resend；当前 S11 没有明确由同一 User-first Gate 阻断，可能产生新的旁路能力或不可收敛完整性故障。
- **影响**：
  - operation 不再是所有普通用户 password-reset 凭据的唯一安全权威；
  - F001/F003 新增的 schema、lineage、settlement 与 migration 无法覆盖 S11，Task-Planner 可能按不可同时实现的两套合同拆任务；
  - commit-ack-loss、同键 replay、stale prepared、消费者 Gate 和 worker 测试矩阵遗漏真实公共入口；
  - 即使主 reset 路径正确，resend 仍可重建 R02 F002 所针对的 durable fail-closed 缺口。
- **修复建议**：
  1. 按 purpose 明确分流：`new_administrator` resend 可继续 ADR-0011 合同；普通用户 `password_reset` resend 必须使用 ADR-0023 durable 协议。
  2. 为普通用户 resend 定义新的 operation/execution generation（或等价且满足一对一 exact credential 的 successor operation），在 SMTP 前 User-first 原子创建 claim、operation、pending credential、job；existing unresolved/hold 时拒绝新 resend。
  3. 固定旧 credential 何时保持、何时失效：SMTP 未受理/安全失败时旧值保持；已受理或提交未知时 Gate 保持并由 worker 收敛；stale prepared 只能失效本次新值，绝不重发。
  4. 把 resend 的六字段 identity、lineage/settlement、历史 replay、迁移约束、worker/受控命令及 commit-ack-loss 矩阵同步到 architecture/design/stories；最小同步现役 identity boundary 的 S6/S9/US-03.11，公共 API 与已确认失败语义无需扩展。

## 5. ADR-0023 未破坏检查与 Gate

ADR-0023 文件 SHA 未变，方案 A、A/B/C 取舍、接受来源、owner、三消费者 Gate、worker、dead-letter、升级/回退及 F002/B6C/B6D 状态边界均未被修订对象改写。r26/r37 的主 reset 合同增强了 ADR 的可实施表达。

F006 是下游 design 对 accepted ADR 的覆盖不完整和自相矛盾，不是 ADR-0023 本身的新缺陷，也不构成重开方案方向。

**ADR Review Gate: PASS（沿用固定 SHA 的原 Gate；下游 F006 必须整改）**

## 6. Design Review Gate 与状态边界

统计：

- R01 findings：F001 CLOSED、F002 CLOSED、F003 CLOSED、F004 CLOSED、F005 CLOSED
- 新 blocker：0
- 新 major：1 OPEN（F006）
- 新 minor：0

**Design Review Gate: CHANGES REQUIRED**

**综合 Design / ADR Review Gate: CHANGES REQUIRED**

architecture r26、design r37、stories r13 尚不足以作为 S10.1 test-plan 修订与 Task-Planner R 对账的稳定输入。原因不是 R01 五项未修复，而是普通用户 password-reset resend 仍保留一条未纳入 durable operation/Gate/worker 的真实入口。

状态边界：

- `IDENTITY-S10.1-REV-R01-F002`：代码层继续 Major OPEN；
- B6C/S10.1：继续 `[-] blocked-for-design-revision`；
- B6D：不得启动；
- 本报告不授权修改代码、test-plan、tasks、plan 或启动 Test-Engineer/Task-Planner/实现角色。

## 7. 待决定与非阻断建议

### 待设计责任方明确

普通用户 resend 的推荐最小方案是“每次成功候选使用新的 durable operation/execution generation”，而不是复用已经结算且 exact credential 一对一的旧 operation。若设计责任方选择其他等价模型，必须证明 operation 仍是唯一权威，并覆盖六字段幂等、lineage、stale prepared、历史 replay 和 worker 收敛。

### 非阻断建议

修订 F006 时保持范围简单：不引入外部消息系统、不改变公共 REST/错误、不把新管理员凭据重发强行迁入 ADR-0023；只把普通用户 password-reset resend 纳入已有 PostgreSQL operation/outbox 协议。

## 8. 未验证风险与限制

- 未运行 Go 测试、race、vet、build 或项目 Manifest 检查；代码只作真实消费者静态证据。
- 未连接或运行 PostgreSQL；未验证 DEFERRABLE 复合 FK、循环插入、隔离级别、commit-ack-loss、多实例租约、deadlock 或 migration/backfill。
- 未连接真实 SMTP/TLS；未验证 provider 受理、超时、延迟及保护窗口。
- 未运行部署、真实 Host 或 E2E。
- PlantUML 仅做静态 fence/文本检查，未运行 CLI 或实渲染。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 工作树已有大量合法未提交改动；本 Reviewer 不归属、不修改也不回滚这些内容。

## 9. 复审条件与 Lead 下一步

交回 Lead 前保持当前阻塞。由设计作者仅定点修订 F006 后，再提供：

1. 新 architecture/design/stories revision 与 SHA；
2. 普通用户 resend 的 durable operation/execution generation、原子 claim、User-first Gate、lineage/settlement 与 stale prepared 合同；
3. architecture/design/stories/boundary 的一致引用；
4. resend commit-ack-loss、旧值保持/失效、同键 replay、unresolved/hold 拒绝、worker 收敛及 migration 约束矩阵；
5. 不改变 ADR-0023、公共 API/错误和代码 F002 OPEN 状态的声明。

原 Reviewer 应在新固定对象上做 R03 定点复审。只有 F006 关闭且无新增阻断 finding 后，Design Gate 才可转 PASS，并交回 Lead 决定是否进入 test-plan 修订；之后才可由 Task-Planner R 对账。Design PASS 仍不得关闭代码 F002或自动启动 B6D。

## 10. 写后验证记录

- 目标报告路径：`.spec/wip/identity-auth/review-design-r37-s10.1-r02.md`
- scoped diff-check：**PASS**；使用 `git -c core.autocrlf=false diff --no-index --check -- /dev/null <report>`，只有预期的“文件存在差异”退出状态，无 whitespace error。
- 固定对象漂移复核：**PASS**；architecture r26、design r37、stories r13、ADR-0023 与原 R01 报告五个 SHA 均与固定值一致。
- 报告最终 SHA256：由最终字节写入完成后计算；为避免文件哈希自引用，精确值在交付回执中给出。
