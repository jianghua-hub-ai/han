# Admin2 Design Review R03：r21 / stories r10

日期：2026-10-28。原独立 Design Reviewer Pauli，非作者 Hubble。objectType：`design`。沿原上下文复用同版完整有效 Reviewer Role、`add-code-review` design 分支、checklist/abstraction/design-writing/decisions/安全规则；未重新 Route、未委派。

**结论：PASS，仅本次 Guard 现状前提纠正、责任/安全/证据边界设计。两轮完成，无新增 findings。** 这不代表系统恢复门禁已交付或验证：A-ADM2-18 的 readiness/recovery 子断言、Case18 整体及 S8.2 整卡接收/关闭仍阻塞。诚实保留的上游 pending 不是本次新增缺陷，也不因本报告变为运行 PASS。

## 1. 固定对象与范围

固定工作区 `D:\AID\VSCode\Codex\Han` / `main`，实际只读核验 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

| 固定对象 | 实际 SHA-256 |
|---|---|
| `.add/project/knowledge/design-workspace/article-management/design.md` r21 | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA` |
| 同目录 `stories.md` r10 | `492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9` |
| Harvey `research-admin2-guard-r01.md` | `213653A8A07EEDFFC65F07B834D5F78E6E66EB27C8FCE7C5EA2D2CB774FC2E4A` |
| 冻结 root `cmd/han-server/main.go` | `2B75C96E61F5BF758F68935F4656238CD8D891D6DC646C32B55EFD0C1A5A6A14` |
| 冻结 root `cmd/han-server/main_test.go` | `092BD3BD78169A40780169A27948D3BD1C6E423406E2C8E658B0EFC59BF3BCBA` |

基线目录 `C:\Users\J00010385\AppData\Local\Temp\han-admin2-r21-20261027T163723932Z\` 的 `baseline.json`、两个 before-image 及完整 `.r21.diff` 已实际读取。design before 为原已审 r20 SHA `6884431B6164140E5593A8253AC20E8EB1C59C1AD71B86558B6D1A8C99BF037A`，stories before 为 r09 SHA `A2BA7746F30B1C9B60DB4337995FC0A0E52DD55BFF984BAA784C20331D80C569`，均与 manifest 一致。

独立重算的 unified diff 与两份留存逐行一致：design 31 行增加/12 行删除，stories 8 行增加/4 行删除。变化仅版本/日期/来源记录、Story8 实际请求链、§8.6 的 Guard 现状/责任/七文件 ACK/Case18 门禁及确认记录；没有扩成全设计返工。全部 Go 接口代码块与 before-image 完全相同，原查询 API/构造、SQL/分页、局部授权/严格身份、Admin12/ETag/错误合同未被此次 diff 改写。

固定 test-plan r16 只读其留存副本 `C:\Users\J00010385\AppData\Local\Temp\han-admin2-testplan-r16-6fc57d5e766e4d17baef3c93eec2c95e\test-plan-r16-fixed.md.txt`，实核 SHA `6941C1D7B96F78B341CBB7B88AE74D548CF860CFEAD37D3D92E208548D896E38`。当前 Ohm r17 工作不在冻结对象，未读取或要求其同步完成。Dalton 正在写的内层七文件也未读取/计算新摘要；原 API 仅按已审 r20 before-image 核一致性。

## 2. Round 1：完整增量与上下游合同

逐段审查上述全部实际 diff，重点为 design :2002–2004、:2760–2778 及 stories 新 Story8 验收/阻塞文字，核事实与义务、组件与系统、计划与执行是否分开。

| 检查点 | 本次设计判断 |
|---|---|
| 现状前提 | 用实际 Trace→exact route→HTTPAuthorizer/SessionStore→AccessGeneral 链替代“现役通用 recovery Guard”前提；账号 Gate 不冒充全局恢复 run/Journal/evidence 检查 |
| 恢复安全义务 | 明确完整恢复/S16/MarkConsumed 前不开流，失败、失效或不可判定继续隔离；保留 `503 platform.recovery_in_progress` 与零文章下游要求，没有降成“启动成功/Session有效” |
| 零写边界 | 新查询层零业务写不扩称认证链绝对零 DB 写，允许现役身份链按原合同撤销无效 Session，未给 Admin2 新增业务写权限 |
| 七文件内层 | 四 article 生产文件与 U/C/P 三测试不拥有全局开流权威；原 API/构造/SQL/分页/身份/DTO/ETag/错误合同不变，合成数据/直接 Handler 不证明系统恢复隔离 |
| root 局部证据 | 允许未来分项证明真实 factory、同一依赖、Trace/Session 接缝及 exact routes；resolver/路由 spy 不证明真实 SessionStore SQL、Article SQL 或全局恢复隔离 |
| Case18 / 整卡 | 不删/豁免/skip 后计成功，不能用 startup、Session403、手造 router/fake503 补证；Case18 整体与 S8.2 关闭仍阻塞，上游未交付时不释放 |
| 责任/范围 | 恢复主线/架构 owner 明确唯一权威、隔离/开流生产接缝及精确装配，随后 Ohm/Planner/Lead 对齐；不因 main.go 属九文件就吸收全局装配，不指派实际人或创建 task/requires/ADR |

r19 R01 CHANGES REQUIRED、r20 R02 有界 PASS/F001 设计层 CLOSED 的原记录保留，未扩称 r21 已受审或 root 已验证。人的 accepted/checkpoint 仍 pending，未代签。没有因上游系统能力尚未交付或 r17 尚未形成而制造新增设计缺陷。

## 3. Round 2：精确上游与真实消费者核查

### 3.1 ADR / boundary 的恢复时序

按决策索引精确定位 ADR-0018/0020，只读取相应正文与 article boundary 适用行，没有预读其他 ADR/全身份设计。

- ADR-0018 `framework.md:388/394/397–405`：恢复完成前不接业务；RecoveryRunStore 是 identity 唯一权威，失败不得开放；`succeeded` 只能在 run、Journal/anchor/article evidence 最终一致后进入并开放 readiness。r21 没有把新 article 查询或组件测试提升成该权威。
- ADR-0020 `content.md:126/149–152`：恢复后先重放有效 tombstone，并与 Recovery Journal/账号门禁共同完成；证据失效/缺失/不可判定继续关闭 readiness。r21 保留该义务，不将 replay provider 被构造视为完整开流连接。
- article boundary `:180/:182–183/:279–280`：maintenance 关闭 readiness；ReplayAll→对账→ready evidence→双 namespace→MarkConsumed→开放。r21 与之相符，未改边界原文。

§8.6 将生产隔离机制留给恢复主线/架构 owner，且明确“未 Serve/已 Shutdown”不能证明现有 HTTP503 子断言，因此没有用进程生命周期证据偷换请求级验收。最终采用何种隔离组合、如何兑现原503合同仍需上游固定并独立取证，非本 Reviewer 代选。

### 3.2 固定 root / 身份 OpenGate 的必要接缝

本轮在用户允许的冻结 root 精确读取以下路径：

- `main.go:2230–2268`：唯一 production factory 组装 handler 后进入同一 router；:2911–2950 是同 pool/repo/reader/config/service 构造接缝；:2997–3031 的读取包装为 `authorizer.Protect(AccessGeneral, ...)`。
- `main.go:3360–3442`：精确路由、method/Allow、nil handler 错误与最外层 Trace；:3764–3765 的 server 直接使用传入 Handler。此处可支持 r21 所列局部 root 证据路径，不能由构造本身推出新增 GET 成功或通用恢复503。
- `http_security.go:309–340`：ResolvePrincipal 后检查 Principal access 并传给下游；`session/store.go:258–332`：同事务 User 锁、password-reset Gate、Session/User 读取、必要的 Session 撤销、commit。与 r21 的账号级恢复/认证写边界一致，不含通用恢复完成事实的证明。
- `internal/identity/deletion.go:140–146`：`CheckRecoveryOpenGate` 返回 nil **仅在 maintenance 且公共 readiness 关闭**。:757–796 的消费者是 `recovery_kept_deleted`，需 capability、同 run/backup、running authority，缺依赖/失败均拒绝；nil 是允许维护中的 canonical 重放，不是普通 GET 的“恢复后开流”。r21 没有倒置该语义。

固定 root 源码在这些接缝中与 Harvey 静态报告一致。本轮没有复跑 Harvey 的全局调查；lifecycle、Login mode、Coordinator→HTTP 未接线等更广的事实依其固定报告消费，未额外展开源码。

OpenGate 注释中出现 candidate S10.5 只是历史线索，不是任务扩权依据。r21 明确禁止据此扩大 password-reset maintenance 候选为通用恢复开流 owner，将真实装配归属交恢复/架构与 Planner 核定。本轮没有复审 S10.5 或其他身份任务卡的完整范围；Pascal 的独立归属调查/后续承接结果不作为本次通过的前置，也不被本报告伪造。

### 3.3 可实施局部与证据接收

原七文件组件合同可以按各自 Lead 窗口继续，和上游承接并行；root 两文件未来可另核真实构造/共享实例、Trace/认证接缝、精确两 GET 与旧路由共存。未承担全局 authority 的局部设计没有因前提纠正变得不可实施。

这些是未来可验证范围：本轮没有执行新 Handler、factory 测试、PG、mutation 或恢复请求；root 当前尚缺普通管理员 GET 的事实不作为本设计澄清的缺陷。真实 SessionStore SQL 与 router resolver spy、P 的真实 SQL→Handler 与 R 的 production 装配必须继续分层取证；Case18 的其他子项结果可单列，不能汇总成整体 PASS。

恢复主线交付后，需真实根拒绝/零文章下游与权威/隔离/开流接缝的同版证据，经适用独立评审、Ohm Case18 对齐、Planner 范围/依赖对账及 Lead 核证再决定窗口和关闭。r21 明确这些条件而没有伪造已实施结果，责任/停止边界合理。

## 4. Findings、Gate 与限制

**新增 findings：无。** 原 R01/F001 设计修复链与 R02 结论保留，不覆盖原报告；本次安全前提纠正形成新的有界设计结论，没有重新裁定旧有效卡或豁免。

| 范围 | 本轮结论 |
|---|---|
| r21/r10 Guard 前提、责任/安全/证据边界 | 有界 Design Review PASS |
| 原七文件内层 API 与组件合同 | 本次 diff 未变；不作为动态代码 Review/运行 ACK |
| root 局部构造/实际链/路由未来证据 | 接缝与证据边界清楚；实际实现/测试及窗口另核 |
| 通用恢复隔离与完成后生产开流 | 尚未交付/验证；不得以本 PASS 证明已具备 |
| Case18 恢复子断言、Case18 整体、S8.2 整卡关闭 | 继续阻塞，不能删断言或用替身补成功 |
| r17 / Planner / 架构归属后续 | 各 owner 独立承接，未读取动态成果或代填完成 |
| CODE/LOCAL/LOCALPG、人的 accepted/checkpoint | 只按 Lead 实际窗口及真实确认；本报告未发 START/未代签 |

未验证范围：Dalton 动态七文件、新 Admin2 运行与 PG/资源/负控、完整通用 Gate 的设计/实现、Coordinator/Journal/readiness 生产连接、真实全局恢复503/零下游、外部流量设施、任务最终装配归属、Host/部署/E2E/容量及图渲染。诚实标注这些限制不构成无条件系统 PASS，也不反过来阻断本次现状纠正的审查。

## 5. 实际 Readset 与操作

| 对象 | 实际读取 |
|---|---|
| r21/r10 当前、r20/r09 before、baseline 及两 actual diff | 全字节/hash、完整 actual diff；原接口代码块对照；重点带行号复核 design :2002–2006/:2760–2778 |
| Harvey 报告 | 本轮核 SHA，复用同会话上一轮已完整读取的有效正文 |
| 固定 r16 副本 | 核 SHA，正文 :285–289，Case18/19/20；未读取动态 r17 |
| 决策 README | ADR-0018/0020/领域文件的精确定位行 |
| `framework.md` / `content.md` | :368–410 / :87–155，ADR-0018/0020 的适用正文 |
| article boundary | :176–184/:274–282，S13/S16 与恢复开放约束 |
| 冻结 `main.go` | :2230–2271/:2910–2958/:2997–3034/:3360–3445/:3755–3782；仅必要静态接缝 |
| 冻结 `main_test.go` | 仅 SHA，未读测试正文或执行 |
| `internal/identity/deletion.go` | :138–148/:754–799，OpenGate 定义/消费精确范围 |
| `internal/session/store.go` | :258–332，证明账号级 Gate/认证事务边界 |
| `internal/platform/http_security.go` | :309–364，ResolvePrincipal/AccessGeneral/GET 安全方法接缝 |
| Role/Skill/rules | 当前会话同版完整有效正文复用，不读下一 Workflow |

必要非 root 源码摘要与 Harvey 相应证据一致；Reviewer 将本次对象/必要依赖摘要记于 Temp `review-r03-readset-hashes.json`，明确不包括动态内层七文件。末次检查固定对象、root 与所查必要依赖无漂移。

唯一仓库写入为本 R03；原 R01/R02 保留。只运行只读 Git branch/HEAD、文本/路径定位、字节/hash/diff/接口块静态检查；未运行测试/PG/DSN/SMTP/服务/Browser、Git 写、下级委派或任何清理。

**stopped：Pauli r21/r10 两轮独立设计评审完成，交回 Lead；不发 START。**
