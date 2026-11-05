# Recovery Design/Story 独立 Review R01（含 ADR 采用元数据合核）

结论：**CHANGES REQUIRED**。两轮有界独立审查完成，**2 HIGH / OPEN**；均为 identity 详细接线不能按其指定的现有服务完成恢复主路径。article design r23 与两份 Story 的本次增量未发现另一个独立根因；四对象作为一个恢复闭环不能判 PASS。两份最终 ADR/index **仅采用元数据核对 PASS**，不表示被登记草案通过设计 Review。

Reviewer：Kuhn，非作者 add-reviewer；不是 Banach。由 Main 依用户 standing 授权和项目委派规则指定。runtime ID `01a1835a-8010-7a73-a8dd-3192719b6965` 的来源为 Main 实际 spawn/resume 返回，非本报告自行启动或以角色文件推断。设计作者 Banach；两份采用元数据作者 Pascal。成果人的精确签字/接纳 **unknown**；本报告不作采用决定或风险豁免，待 Main 现场技术接收。

## 固定对象与前后摘要

固定工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；用 `.git/HEAD` 与分支 ref 文件只读核对，无 Git 写入、无切分支。下表 before 为本 Reviewer 对相应冻结对象首次获准读取时的摘要，after 为交付前复核；不是作者改写前像。

| 对象 | before SHA256 | after SHA256 |
|---|---|---|
| identity design r50 | 03D7A5BD87A1558A8CA26AAE93F3552B860AD2259921ED5BE9C57DD2B7D4A633 | 03D7A5BD87A1558A8CA26AAE93F3552B860AD2259921ED5BE9C57DD2B7D4A633 |
| article design r23 | 436E323486F02E86BF985B92CB8B2DA096B9CC3440F629B1B2B0875B8EB2731D | 436E323486F02E86BF985B92CB8B2DA096B9CC3440F629B1B2B0875B8EB2731D |
| identity stories r17 | 247887668F3F7F7DA056FC376A4C556B618884E44011564BA339AEE8061BEFC0 | 247887668F3F7F7DA056FC376A4C556B618884E44011564BA339AEE8061BEFC0 |
| article stories r11 | 33E6A46AE28B3CB5DBB3B3C5559895167B1BAB0B1F29AE73C60878A9662902A4 | 33E6A46AE28B3CB5DBB3B3C5559895167B1BAB0B1F29AE73C60878A9662902A4 |
| framework 最终采用尾 | BCB83191B5760826295BF661EAECDB2B873B31C26DF60B840E81B031DF639989 | BCB83191B5760826295BF661EAECDB2B873B31C26DF60B840E81B031DF639989 |
| decisions README 最终计数 | 157A549AD375B650479900A350464FCB842E828EFD296F3A50FA406A996FBAA1 | 157A549AD375B650479900A350464FCB842E828EFD296F3A50FA406A996FBAA1 |

前四路径均为 `.add/project/knowledge/design-workspace/<feature>/{design,stories}.md`；后二路径为 `.add/project/knowledge/decisions/{framework,README}.md`。额外固定上游 architecture r32 的 before/after 均为 `89605FAF5D0C7C663791086112E6BD1A33FC5C309A01E6DBCAA2C91795B0A27E`。未发现对象或本次已读仓内依赖漂移；详见 `static-verification.json`。

在 Arch 并行写窗期间只读 immutable `author-status/frozen/.add/project/knowledge/decisions/` 快照，framework 为 `329631B71E692E96D8B8FA918686E7942AE549B492E1B3A4B69F03C889F81F5C`，README 为 `92B7598534B934FD7214F30B46A9BFBAAD8A500617561873DBD13BDF965848AC`。收到 Main 明确 STOP/frozen 及新授权后才读/hash 两份 live；没有在同一技术判断中混换 ADR 正文。

## Round 1：完整增量和直接合同

全文读取作者四份 before→frozen 精确 diff，逐项核类型、字段、所有权、阶段、许可、取消/故障、恢复来源、升级和 Story 映射。独立将作者 hunks 应用于前像，结果与当前冻结文本逐行一致；新增/删除分别为 identity design +419/-5、article design +157/-6、identity Story +13/-5、article Story +11/-5。既有上下文采用真实行号定向补读，未冒称四份长文所有历史章节重新全审。

直接核了 recovery 中性端口、AdmissionRun/NULL/canonical、0010/0011 字段与约束、两 namespace/revision/ABA、genesis prepared→bound、full-set 失败优先级、normal 零 repair、article staged 写者/协调、publication/history 与 fresh 区别、RestoreDriver/receipt/ACK loss、root 精确 surface、三 worker/真实 join、Story 和既有责任边界。Round1 未因新接口尚未实现、真实工具/环境 unknown 或有意保守关闭本身登记 finding。维护登录与恢复重置的指定复用接缝进入 Round2 验证，最终确认为下列两项 HIGH。

## Round 2：沿实际消费者和约束

| 有界链 | 实际检查与结论 |
|---|---|
| 上游合同 | architecture r32 §9.1.1；冻结 ADR0018/0019/0027；两现役 boundary 的 S13/S14/S15/S16/恢复段；identity TP I-REC-01～10 与 article TP Case18 分项。没有把 identity 当前 boundary r13 整份提升 accepted，没有重审无关边界。 |
| Store/安装投影 | 实读 `identity/recovery_run_store.go` Load/Transition/映射，0005 全文及 0004/0002 相关 trigger/review 约束；新全量投影不是现有单 run Load 的别名。0010 的 NULL legacy、真正阶段转换校验、run/restore-binding revision、不可变/同 outcome 迁移可与旧 0005 并存；不能全表加立即验证的 NOT NULL 排斥合法 legacy。新 migration 尚未生成/运行，SQL 可实施性只是静态判断。 |
| Recovery journal/Consume | 实读 `recovery/journal.go` Anchor/模型/接口、Read/VerifyAgainst/loadRecords，及 `coordinator.go` identity_ready 起始至 succeeded。旧 reader 会进入可修复路径，新设计已明确原位 ctx 迁移和无副作用 parser，不能将旧实现直接冒充新 normal reader；唯一 ready_to_consume+consumed 补写窗口未被推广。 |
| Article 写者/真实调用 | 实读 `deletion_journal.go` Append/persist/ReadAllVerified、finalizer Converge/ReplayAll/convergeAndCount、worker Reconcile/Start/Stop/Wait。新合同明确将旧 sync/ensure 路径迁出 normal。startup 内部许可允许真实 pending_journal 追加；recovery_replay 只允许精确 committed 幂等命中，新任意 append 拒绝。无须先放开公共 HTTP，未发现新的准入环。 |
| Article 发布/历史 | 实读 `replay_evidence.go` Publish/owner/current/历史选择、`deletion_replay_source.go` 真 ReplayAll 与 Snapshot/operation/audit 事实、0003 evidence/run 约束及 0009 全文。0011 新 publication 可在现有同一 run 事务内扩展，FK 支撑 UNIQUE 不改变既有语义唯一性；历史 port 不能伪装 private fresh 查询。旧 Ready 可 fresh 复用与旧 succeeded 无独立历史来源仍关闭已区分；实际 PostgreSQL/FK/trigger/commit unknown 尚未验证。 |
| 身份恢复权限 | 实读 Login→SessionStore→HTTPAuthorizer→PasswordChange，形成 F001；实读 PasswordResetCoordinator actor/target、SQL 及 operation Store 的 recovery_locked 校验分支，形成 F002。canonical 的同一 Store 锁读要求 running，OpenGate nil 仅证明 maintenance+公共关闭，与新合同相容。 |
| Root/生命周期 | 实读唯一 `cmd/han-server/main.go` startup maintenance 拒启、Login/密码变更/finalizer/replay source 注入与 exact router，`lifecycle.go` 初次 Reconcile/Start、Stop/Join/资源关闭；另读 PasswordReset worker 与 Article notice 的启动、取消、BeginShutdown、Wait。新设计明确纳入三项注册、started 握手、共享截止与 server slot；不以 Start 返回或超时替代真实退出。 |
| 普通/受控 surface 与同进程切换 | exact method/path 后再申请公共 lease、Trace 最外层，维护 allowlist/CSRF/Origin/可信 surface 及 complete HTTP 退出后切换均有中央要求。静态 RecoveryMode 只参与 recovery_bootstrap 分支，不能仅因其 maintenance 常量断言 permanent 普通登录一定失败；实际不可实施处是 F001 的 Session/flag/强改密衔接。切换后须撤销恢复能力、仅允许普通策略；不能以手改 PublicReady 绕过 F001。 |
| Restore/secret/requeue | 按固定协议推演 Prepare→Begin committed→Replace once→native receipt/独立回读→Store 绑定、ACK loss/崩溃/无证据保持 restore_unknown，及 SourceSecret 只在需信任根时惰性读取并匹配 Begin retained key。未查看真实 secret/ENV/工具。共享 launcher/部署锁不改变 S10.5 原 SID/签名/epoch 范围。 |

类型声明以 recovery 为消费端单一来源，article 使用其 PublicationBinding/VerifiedJournalSnapshot/PhasePermit；未发现 Go import 反转。双向运行回调在 root 可用私有、封闭的一次绑定适配器承接，不据此强报 import 或构造循环；是否真实同实例、未绑定不得授工作仍须未来 factory 检验，不能用 fake/no-op 填空。

TE Jason 的 16 组 `coverage-input.md` 已全文咨询，SHA `7492951773C8815ADCD99846F13402A47BD4452DC9CB0C00EA6F3C6394094388`。它只读了原 r32/TP，未审新设计，不作为本次独立 Review 或已通过测试证据。

## [HIGH] RECOVERY-DES-R01-F001 — 指定的维护登录/强改密链存在互斥的恢复标志条件

状态：**OPEN**。对象 identity design r50；目标 `design.md:3556`、`:3612`（关联旧 §4.2 `:3395`、登录合同 `:1803`、Story r17 `:188`/`:195`）。行号按 LF 计数。

现状：新 executor 要复用现有 credential 服务；维护面明列现有 POST sessions、GET me、PUT password，并把 initial admin 已强改密作为 review 和 identity_ready 前置，却没有给出与现役 Session/flag 规则相容的受控解析/转换接缝。仅设置 maintenance surface、PhasePermit 或已有 recovery_bootstrap 凭据不够。

触发与源码证据：

1. 恢复信任根写成 recovery_bootstrap 后，真实 `internal/identity/login.go:1448–1453` 要求 `IsInitialAdmin && RecoveryRequired`，否则登录拒绝；静态 mode 在 `:1482–1504` 仅完成其 maintenance 检查。即使新 executor 补上 `recovery_required=true`，成功登录也没有完成后续认证链。
2. 下一次 GET me 或 PUT password 经 `internal/platform/http_security.go:320` 调 `SessionStore.ResolvePrincipal`。`internal/session/store.go:568` 的总授权条件无条件要求 `u.recovery_required=false`；`:676–680` 的 recovery_bootstrap CASE 位于该合取之下，不能覆盖它。`:315–317` 会把 authorization_state_invalid 的 Session 撤销，`:281–287` 提交后返回 SessionRequired。故登录态无法到达强改密 handler；若改为 false 则回到上一步登录拒绝。
3. 即使局部绕开 resolver，现有 `identity/bootstrap.go:344–356` 的 SetPermanentPassword 只更新 digest/mode/auth_generation 等，并不清除 recovery_required；新完整接线未规定其可信原子退出转换。`identity/service.go:1002–1007` 支持该凭据种类，不代表 HTTP 已能到达它。

影响：在来源和环境完全健康时，主恢复流程仍无法取得“已强改密的 initial admin”去核对账号并达到 identity_ready。这不是 unknown 证据导致的有意保守可用性限制，也不是因为新类型尚未实现。

最小修订要求：在 identity 详细设计明确**同一 run/安装/代际的受控维护 Session 解析与强改密转换合同**，确定 executor 如何设置 initial-admin recovery 标志、如何在公共仍关闭时允许有限 me/password，并在同一受控事务完成新永久凭据、auth generation/Session 轮换及恢复标志退出。入口必须核真实 runtime 阶段/同一 Store，不能普通 Principal 或 maintenance bool 即豁免全局 Session 条件；普通 resolver 与非初始账号继续拒绝 recovery_required。规定完成后同进程 public surface 的普通 Session/凭据策略及恢复 capability 撤销，不能只翻 PublicReady 或全局删除 SQL 条件。owner/文件落点/Story 验收随这条接缝定向同步；不代作者选择具体类型名。

复审证明：静态推演并沿真实消费者证明 recovery_locked→recovery_bootstrap→登录→me→带 CSRF 强改密→已改密 initial-admin review→identity_ready 可达；false flag/非 initial/错 run/公共 surface/已撤销代际仍拒绝；强改密 ACK loss 精确回读不重置新凭据，同进程转普通后不残留维护授权。后继实现须以真实 SessionStore/PasswordChange 接线验证，不能仅 fake Principal；本次不运行该测试。

## [HIGH] RECOVERY-DES-R01-F002 — restored 必须调用的日常 reset 不接受必然出现的恢复目标状态

状态：**OPEN**。对象 identity design r50；目标 `design.md:3614`（关联 `:3556`、`:3389`、`:3398`）。与 F001 独立：即使 initial admin 已能正常维护认证，本问题仍存在。

现状：r50 要求 restored 调用“既有受控密码重置”，继续用既有 durable operation/幂等/Gate，SMTP/operation 未决不完成 review；原 §4.2 又规定先重置新临时凭据，之后才能在同事务写可信 role/status、清除 recovery_required。终结后所有非初始账号为 suspended、recovery_required=true、credential_mode=recovery_locked，永久摘要已清空。

触发与源码证据：选一个有合法邮箱、可信核对结果为恢复 active member、无旧 unresolved operation 的目标。即使 F001 修复，`internal/identity/password_reset.go:180–209` 的 prepareDurableCandidate 仍会调用 `passwordResetTargetEligible`；`:779–789` 明确拒绝 RecoveryRequired，且仅允许 permanent/temporary，不接受 recovery_locked。尚未生成任何 durable candidate 就返回非法状态。仓储 `:1293–1299` 的 SetCredentialMode 同样要求 recovery_required=false，不能仅跳过上层判断完成。另有当前 role 与请求 TargetRole 的精确比较（`:201`）；恢复点 role 只能作为待核材料，不能靠伪造日常请求替换可信核对结果。

`password_reset_store.go:1597` 确实存在 recovery_locked lineage 校验分支，说明既有旧 operation 可以被安全分类；它没有绕过以上新 candidate 前置或提供 restored 的写入口，不能以关键词命中视为闭环。

影响：合法 restored 永久停在 pending/非法状态，无法完成全部 review；也会诱使后继 Task 选择提前清 flag/改 mode 来“适配”reset，破坏先新凭据结算、后解除账号 Gate 的中央顺序。不能以把所有可证明账号都 denied 替代 restored 主路径。

最小修订要求：明确 identity-owned、run=running、review=本目标 pending、可信 role/status/evidence 与阶段许可绑定的**恢复重置接缝**；复用现有 durable operation/candidate/SMTP/settlement 机制，精确说明恢复态的合法前置与仓储条件、User-first/Store/review 锁序、可信角色选择及最终原子结算。日常 reset 的 RecoveryRequired/recovery_locked 拒绝条件保持；不可先清 Gate 或伪造 permanent/temporary。恢复期间普通 worker 不启动，所以须明确由哪个受控调用驱动需要的 settlement，仍尊重原 unknown/hold/dead-letter 合同；未知保持未完成，不重新投递或清 hold 当成功。这些安全条件应由本设计固定，不能留给 Task 自选。

复审证明：在 suspended+true+recovery_locked 的合法 fixture 上完成可信 restored 路径；核新 credential/operation lineage、SMTP accepted 与 ACK loss/unknown、恢复重试、role/status/review/audit/Gate 原子结算；旧凭据始终不可用、普通 reset 同态仍拒绝、复用终态 review 零再次 reset。旧 unresolved operation 仍先按 Gate/settlement 收敛，不允许无证据新 candidate；不要求新增另一个邮件协议。

## 两份最终采用元数据：独立定点结论

**PASS，仅 metadata。** 由 Pascal 负责改写、Main 报告已初核，本 Reviewer 另做实际字节比较及路径/摘要核验：

- framework 只有第807/808两行插入；删除这两行即可**逐字节**恢复 SHA `329631...` 的冻结前像，ADR0027 r02 技术正文、旧 Review 状态、其他 ADR 与原 architecture 引用均不变。两条相对链接实际解析到本次两份 design.md，嵌入 SHA 与四对象中的两设计精确一致；draft/待独立Review/人的接纳unknown/不等于实现测试通过均保留。
- README 第35行仅末列 `1→3`；还原该数字即可**逐字节**恢复 SHA `92B759...` 的冻结前像，状态与 R02 相对链接等其余字节不变。
- 三项计数为原 architecture 实际采用1＋两设计实际采用2；两 Story 只是派生同步，不另计，不预记 TE/Planner/Code，不含人签字。设计存在 OPEN findings 不否定“草案实际采用 ADR”这一登记事实，但也不能据计数授予设计 PASS。

本轮不重审 core policy、不重开 `ARCH-R32-R01-F001`；该项维持先前架构文档级 CLOSED。两份元数据 PASS 不覆盖本次两个新的 DES finding。

## 待决定、非阻断建议与证据边界

实际 CAS/provider、独立故障域/介质/ACL、安装/备份批准者、restore 原生工具及其可信执行主体、真实恢复回执和人的签字均仍 **unknown**，由 Main/对应部署与业务责任人按原门槛落实；不是 Reviewer 代选。无新的业务方向待本 Reviewer 决定，没有风险接受决定。

Main 提供的 fd 疑点不列 finding：`:3576` 已硬性要求校验/实际 restore 同一稳定 handle 且禁止按可替换路径重开，`:3568` 同时给 root 对 fd/进程树预检责任；固定 argv/stdin 字段不排斥受信 cmd adapter 以其私有 fd 继承/复制约定承接。当前未交付具体工具，因此其跨进程句柄传递与“未重开路径”只能列**后继实际验证缺口**，不能宣称已验证，也不能放宽为 pathname fallback。

没有新增非阻断格式修订要求。类型细化/私有装配适配器可以由实施具体化；F001/F002 的授权/状态/原子退出合同必须先回详细设计固定。

## 实际检查、NOT RUN 与 STOP

实际运行仅本地只读文本/行号读取、定向 rg、SHA256、ref 文件读取，以及独占 Temp 内静态审计脚本。`static-verification.json` 记录六冻结对象＋architecture、四 diff hunks 应用比对、两 metadata 字节保真检查和本次已逐段读取仓内依赖的前后摘要。`readset.jsonl` 是实际读取范围与 SHA；`readset.md` 聚合可读清单。哈希整个文件不意味着全文审查该依赖；rg 定位和全文读取已区分。

静态脚本首次尝试重新生成同格式 diff，因重复 Markdown 空行的 context 选点不同而断言失败；改用逐 hunk 应用作者 diff 后，全部旧文本/计数及冻结目标逐行相符。该脚本检查方式修正只发生于独占 Temp，不是对象漂移或一条产品 finding；未把首次脚本失败隐藏为 Go/PG 测试结果。

作者 verification 中 source190/其他knowledge29/controls3 的无漂移及 Main 全文现场复核，均按其来源消费；本 Reviewer **没有重跑全库保护集**，也不以他们的静态检查代替自己的两轮技术判断。

**NOT RUN**：Go/PG/编译/vet/单元/集成/race、真实迁移 SQL/触发器、服务/HTTP/SMTP、恢复工具/子进程、真实 CAS/ACL/持久卷/备份/E2E；未查 ENV/DSN/key/真实秘密、未安装/联网。PlantUML CLI/jar 缺失沿已知证据；本次无语法引擎或渲染证据，静态阅读不得称 syntax/render PASS。没有新建 TaskID，没有共享源/知识/Task/Review 写入，没有 Git 写/清理/删除/移档或下级委派。

S1.5/S4.4/S8.2 原 active 与 Case18 整项未通过保留；Case18 两真实 GET503/Trace/零 Article 下游及正常 Auth/exact router 兼容义务未缩减。identity S4.5 真实 CAS/环境门槛、S4.6 retirement 缺闭环持续禁用，已闭 S4.2/S10.5 不重开。架构/设计/ADR 结论均不等于实现 Review、测试 PASS、部署授权或人的接纳。

报告/receipt 完成后 **STOP 所有写入**；交 Main 按实际冻结版现场接收、安排原作者设计返工及原稳定 ID 独立复审。本 Reviewer 未改被审稿，也未替 Main 选择采用或签字。R01/R02/R03 架构旧报告原样保留。
