# identity password-reset durable fail-closed Design 定点复审 R03

- 日期：2026-10-24
- Reviewer：add-reviewer（原 R01/R02 Reviewer，非作者独立复审）
- objectType：`design`（architecture + feature design + stories + boundary 复合修订）
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审目标：定点复审 R02 F006，回归 R01 F001～F005，并检查本轮修订引入的新架构、安全、迁移、恢复、状态所有权、并发与消费者契约风险
- 结论：**R02 F006 CLOSED；R01 F001～F005 均保持 CLOSED；无新增阻断 finding；ADR Review Gate PASS；Design Review Gate PASS**

本报告只评审下列固定设计对象。accepted ADR-0023 是未变上游决定，只核其方向和合同未被本轮修订破坏，不重开方案 A。Design PASS 只表示这些固定设计对象可交回 Lead，供其决定是否进入 S10.1 test-plan 修订；不关闭代码层 `IDENTITY-S10.1-REV-R01-F002`，不自动解除 B6C/S10.1 阻塞，不启动 B6D，也不声明实现、测试、部署、E2E 或生产就绪。

## 1. 固定对象、revision 与漂移前置检查

| 对象 | revision / 范围 | 固定 SHA256 | 复审开始时 |
|---|---|---|---|
| `.add/project/knowledge/design/architecture.md` | `architecture-ir01-r27` | `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB` | 匹配 |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r39`（draft） | `71EEC596875EE9CBF44CF300C72A4203F5F2C8976522F478A1857D882AE86230` | 匹配 |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r15` | `715F5BEB95FC7AF141F7F9ED18D900979921C520D7E11A857AC017F65C481803` | 匹配 |
| `.add/project/knowledge/design/identity-auth/boundary.md` | `identity-auth-boundary-ir01-r13`（draft） | `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3` | 匹配 |
| `.spec/wip/identity-auth/review-design-r35-s10.1-r01.md` | 原 R01 | `02729A1559CF8EAB7FB1F6DFB4560DEBBF4ED00C19C9CC0E0F285467F6F96E6D` | 匹配 |
| `.spec/wip/identity-auth/review-design-r37-s10.1-r02.md` | 原 R02 | `41A988FB57E823EF3ADC7A1960245EB42677370B37FED1E4791EC2FD019D9473` | 匹配 |

上游一致性快照：

- `.add/project/knowledge/decisions/identity.md`（含 accepted ADR-0023）：`2FCE6825777B3FBF07621A3793DB080845587E651EDEBAC1272218A38FD1B978`
- `.add/project/knowledge/requirements/requirements.md`：`999CBA3C0CD25D2760E400A41D2A373611406854E76E24E453321913D71CBC4F`

复审开始时分支、HEAD 和六个固定对象均匹配，无漂移；目标 R03 报告此前不存在。工作树已有大量合法未提交改动，本 Reviewer 未归属、未修改、未格式化、未清理或回滚这些内容。

## 2. 两轮实际检查范围

### 2.1 Round 1：固定对象直接检查

完整读取并按 `add-code-review` 的 design 分支检查：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/decisions-guide.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`
- go-service profile 及其适用 coding、security、no-hardcode、observability、tracing 规则
- architecture r27、identity design r39、stories r15、boundary r13、R01、R02

直接检查聚焦：

1. F006 的 purpose 分流、successor operation/execution generation、同键 replay、发送权与 stale candidate；
2. old active/new pending、三消费者 Gate、settlement、lineage、integrity hold；
3. worker 租约/接管/dead-letter、双 worker lifecycle；
4. migration、约束、旧 writer 停止、混跑阻断、不可安全回退；
5. architecture/design/stories/boundary 的合同一致性和 test-plan / Task-Planner R 可验证输入；
6. R01 F001～F005 是否因 F006 扩展而复发；
7. 是否误关代码 F002、误解锁 B6C/B6D，或虚构已实现、已验证。

### 2.2 Round 2：上游依据与真实消费者有界深审

在用户允许的边界内读取或复核：

- requirements 中 US-IDENTITY-03.10/03.11 及关联合同；
- identity boundary、ADR-0011、ADR-0015、ADR-0018、ADR-0023；
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

现役代码和 migration 仍只作为真实消费者与改造面证据：当前实现尚未具备本设计要求的 durable operation、完整 generation/lineage、caller-owned Session resolve transaction 和 PasswordReset worker composition。因此本轮设计结论不能解释为代码已修复。

## 3. R02 F006 定点复审

### IDENTITY-DESIGN-R37-S10.1-R02-F006 — CLOSED

**原问题**：普通用户 `password_reset` resend 仍走 ADR-0011 SMTP/数据库残余窗口，绕过 ADR-0023 的 durable operation/Gate/worker；旧设计没有为每个新候选建立新的安全权威，无法闭环 commit-ack-loss、stale prepared、lineage、settlement 和迁移约束。

**逐项关闭证据**：

1. **purpose 分流与唯一安全权威已明确。**
   - `design.md:2049` 明确 `PasswordResetOperation` 同时覆盖首次 reset 与普通用户 resend；`new_administrator` resend 继续 ADR-0011。
   - `architecture.md:563-567`、`boundary.md:195,198,225-226,294-297` 使用同一分流，且客户端不能提交或串改 purpose。
   - 公共 REST、公共错误、权限和 requirements 范围未扩张。

2. **每个新候选使用新的 operation/execution generation；same-key replay 无新副作用。**
   - `design.md:2088-2095` 要求 User-first 单事务分类真实 `(scope_hash,key_locator)`，原子推进 execution counter 并创建 claim、successor operation、exact pending credential、initial job；resend 绑定 predecessor operation 与 replaced credential。
   - 同键同摘要只重放原 operation/generation，不生成新秘密、不发第二封邮件；同键漂移、孤立 claim/operation 或跨 User 绑定进入冲突或 integrity hold。
   - `architecture.md:565-567,1284-1296`、`stories.md:377-385` 与上述原子性和 replay 合同一致。

3. **发送权、保护窗口与 stale candidate 已封闭。**
   - `design.md:2097-2099` 固定只有明确提交 `prepared→dispatching` 的原请求取得一次发送权；dispatch commit acknowledgement 丢失时不发送，worker/replay 不补发。
   - `design.md:2111-2140,2177` 区分 delivery evidence 与 operation 状态；worker 只做数据库 settlement，stale `prepared/dispatching/commit_unknown` 在无 durable accepted 证据时只失效本次 new candidate。
   - `architecture.md:567-569,1201-1202,1291-1305` 和 `stories.md:385-386` 没有恢复“worker 重发邮件”的旁路。

4. **old/new 能力、终态、successor 与消费者 Gate 已可判定。**
   - `design.md:2144-2152` 固定 auth generation、永久密码/TemporaryCredential/Session root lineage、合法 temporary→permanent successor 与 integrity hold。
   - `design.md:2154-2165` 明确 old active/new pending 可暂时共存，但 unresolved 时不可认证；只有 succeeded settlement 原子激活 new、supersede old、推进 auth generation 并撤销旧 Session。SMTP 明确未受理或 stale 且无 accepted 证据时 old、generation、Session 保持。
   - `design.md:2167-2173` 固定 Login、PasswordChange、Session resolve 的 User-first Gate；Session resolve 仅在 caller-owned transaction commit 成功后返回 Principal。
   - `architecture.md:569-573,1197-1202,1303-1316` 与 `stories.md:383-389` 保持相同安全时点和竞态结果。

5. **worker、dead-letter、恢复和生命周期没有新旁路。**
   - `design.md:2140,2175-2194` 固定租约、过期接管、retry_wait、dead_letter、受控命令、审计与失败后持续 Gate；受控命令也没有 Mailer 权限。
   - 原双 worker coordinator、shared shutdown deadline、稳定主错误优先级、cleanup 聚合与 Article 资源/DB pool 关闭顺序继续保留；`architecture.md:1191-1192,1415-1417` 明确 F006 增量仍回归双 worker lifecycle 与 Session commit barrier。

6. **migration、混跑和不可回退合同覆盖普通用户 resend。**
   - `design.md:2209-2234` 覆盖 execution counter、request kind、predecessor/replaced/delivery 字段、六字段 FK、UNIQUE/CHECK/索引、old-active/new-pending 约束、legacy preflight、停止旧 reset/resend writer/reader、完整装配后才开放及写新行后不可回退旧二进制。
   - `architecture.md:1198-1202,1417` 与 `stories.md:388-389` 给出一致的迁移和验证输入。

7. **边界、设计和 Story 已同步。**
   - `boundary.md:195,198,225-226,255-256,294-297,464` 固定两类 resend 分流、普通用户新 generation、unresolved/hold 零写入、stale candidate 和 worker 接缝。
   - `design.md:2266-2363` 将 S11 purpose router、successor operation、predecessor/replaced lineage、commit-ack-loss、stale candidate 和 consumed race 纳入详细设计。
   - `stories.md:327-389` 提供可由 test-plan 和 Task-Planner R 直接消费的场景、验收与验证矩阵。

**关闭结论**：R02 所指“普通用户 resend 形成 operation 权威旁路”的触发链已被固定对象完整消除。operation 仍是普通用户首次 reset 与 resend 的唯一安全权威；新候选、same-key replay、发送权、settlement、Gate、worker 和 migration 已形成同一可验证协议。F006 CLOSED。

## 4. R01 F001～F005 回归复审

### F001 — CLOSED，未复发

- `design.md:2088-2095,2103-2112,2209-2219` 继续要求真实六字段 identity、`(scope_hash,key_locator)` 定位和 counter/claim/operation/exact credential/job 原子创建。
- F006 对 resend 复用同一专用事务端口，而非恢复“外层先独立提交 Claim”。
- architecture、stories 的 crash/constraint 验证仍覆盖 claim-only、operation-only、counter-only 和错误复合绑定。

### F002 — CLOSED，未复发

- `design.md:2084,2167-2173,2189-2194` 继续统一 `User → Operation/job → Credential → Sessions → Idempotency/Audit`；无 operation 行仍由 User 锁作为插入屏障。
- Session resolve 保持 caller-owned transaction、事务内 Gate/lineage 复核和 commit 后 Principal。
- resend 增量没有引入 Operation-first、扫描事务持锁授权或 commit 前返回 Principal。

### F003 — CLOSED，未复发

- `design.md:2138,2144-2163` 继续以 auth generation、root lineage、settlement 和 integrity hold 区分合法 successor 与非法竞态能力。
- consumed 不被当作安全充分条件；PasswordChange 与 resend 竞态、历史 replay、后续正常改密/后续 reset 均有确定结果。
- resend predecessor/replaced lineage 扩展了原模型，没有复用已结算 operation 承载新的 exact credential。

### F004 — CLOSED，未复发

- 双 worker composition/lifecycle 合同仍包含 Reconcile/Start、任一运行异常触发共同关闭、Stop 各一次、同一 deadline 并发 Wait、主错误优先级和 cleanup 聚合。
- PasswordReset worker 仍先停止，Article 专属资源随后关闭，DB pool 最后关闭；F006 没有新增第三个 worker 或覆盖现役 Article lifecycle。

### F005 — CLOSED，未复发

- stories r15 继续把 Q1 记为已接受的统一改密规则与 Gate，不再作为未决实施阻塞。
- 文档仍明确实现和验证未完成，且没有扩成周期轮换、定期过期或密码历史。

## 5. 本轮新增 findings

### BLOCKER

无。

### MAJOR

无。

### MINOR

无。

本轮未发现需要阻断 ADR Gate 或 Design Gate 的新增架构、安全、迁移、恢复、状态所有权、并发/竞态或消费者契约缺陷。

## 6. 待决定与非阻断建议

### 待决定

无。F006 的方向已经由用户于 2026-10-24 明确采用，修订对象没有留下需要 Reviewer 代替责任方选择的 A/B 方案。

### 非阻断建议

1. `architecture.md:17` 的 metadata source 写为不存在的 `.add/project/knowledge/design-workspace/identity-auth/boundary.md`；实际对象位于 `.add/project/knowledge/design/identity-auth/boundary.md`。建议下次架构修订时更正追溯路径。正文合同、revision、固定 SHA 和本轮 Gate 不依赖该错误路径，因此不登记为 finding。
2. boundary r13 的尾部仍保留其产生时“下游尚未返工、Design Gate CHANGES REQUIRED”的过程状态。R03 报告是本轮关闭 F006 的新证据；建议 Lead 接收/后续知识同步时避免把该历史过程文字误读为 R03 后仍存在的合同缺口。
3. PlantUML 本轮仅做静态 fence/文本检查；后续文档验证可运行真实 CLI 渲染，不能把当前静态检查写成图已渲染通过。

## 7. ADR Review Gate

ADR-0023 文件 SHA 未变。方案 A 的 durable operation/job、三消费者 Gate、数据库租约 worker、dead-letter/受控收敛、升级/回退边界，以及 F002/B6C/B6D 状态边界均未被本轮修订破坏。

普通用户 resend 被纳入既有 ADR-0023 协议，`new_administrator` resend 保持 ADR-0011；这属于已授权的下游设计闭环，不改变 ADR-0023 的核心取舍，也不需要 Reviewer 新增 ADR。

**ADR Review Gate: PASS**

该 PASS 只表示未变的 accepted ADR-0023 仍可作为上游决定；不表示本报告重新接受 ADR、批准实现或验证代码。

## 8. Design Review Gate 与状态边界

统计：

- R02 F006：CLOSED
- R01 F001：CLOSED，未复发
- R01 F002：CLOSED，未复发
- R01 F003：CLOSED，未复发
- R01 F004：CLOSED，未复发
- R01 F005：CLOSED，未复发
- 新 blocker：0
- 新 major：0
- 新 minor：0

**Design Review Gate: PASS**

architecture r27、identity design r39、stories r15 与 boundary r13 已足以作为 Lead 后续决定 S10.1 test-plan 修订、再由 Task-Planner 执行 R 对账的稳定设计输入。该判断基于设计合同闭环，不基于当前代码已满足合同。

状态边界：

- `IDENTITY-S10.1-REV-R01-F002`：代码层继续 **Major OPEN**；
- B6C / identity-auth-S10.1：继续 `[-] blocked-for-design-revision`，是否按流程转入下一阶段由 Lead 另行决定，本报告不直接改状态；
- B6D / S11.1：未启动且不得由本报告启动；
- 本报告不授权修改代码、test-plan、tasks、plans、checkpoint 或 manifest，不启动 Test-Engineer、Task-Planner 或实现角色。

## 9. 未验证风险与限制

- 未运行 Go test、race、vet、build、gofmt 或 Manifest check；代码仅作静态消费者证据。
- 未连接或运行 PostgreSQL；未验证 DEFERRABLE 复合 FK、循环插入、隔离级别、deadlock、commit-ack-loss、多实例租约、migration/backfill 或部分唯一约束。
- 未连接真实 SMTP/TLS；未验证 provider 受理、超时、延迟、投递或保护窗口。
- 未运行部署、真实 Host、故障注入或 E2E。
- 未运行 PlantUML CLI 或真实渲染。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- boundary r13 与 design r39 仍是 draft，最终接收属于 Lead/确认责任，不由 Reviewer 代签。
- 工作树含大量合法未提交内容；本报告只对固定 SHA 对象下结论，不对其他并行改动背书。

## 10. 复审条件与 Lead 下一步

本轮 F006 已关闭，无需再次设计返工复审。若 Lead 接收本报告，可按既定顺序：

1. 保留代码 `IDENTITY-S10.1-REV-R01-F002` OPEN；
2. 由 Lead 决定进入 S10.1 test-plan 修订；
3. test-plan 经相应独立评审后，再由 Task-Planner 做 R 对账；
4. 不直接进入实现，不启动 B6D。

以下任一情况需要重新复审受影响 Gate：

- 四个设计对象任一 revision/SHA 漂移；
- purpose 分流、operation 唯一权威、same-key replay、发送权或 stale candidate 合同被改写；
- old/new settlement、auth generation/lineage、三消费者 Gate 或 Session commit barrier 被弱化；
- worker 获得重发权限，或 migration/混跑/不可回退边界被删除；
- 公共 REST、错误、权限、requirements 或 accepted ADR 方向发生变化；
- test-plan / Task-Planner 发现固定设计内部无法同时实现的矛盾。

## 11. 写后验证记录

- 本轮唯一写入归属：`.spec/wip/identity-auth/review-design-r39-s10.1-r03.md`
- 定向 `diff --check`：PASS
- 六个固定对象写后 SHA 漂移检查：PASS，均与第 1 节一致
- 分支 / HEAD 写后检查：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 报告最终 SHA256：由写后交付回执提供；SHA 值不写入文件正文，以避免自指写入再次改变文件摘要
