# RECOVERY-DES-R02：原 F001/F002 独立复审

**结论：PASS，仅本次固定设计修复与相关新风险的有界设计层复审。RECOVERY-DES-R01-F001、RECOVERY-DES-R01-F002 均 CLOSED（design level）；OPEN=0，LIMITED=0，新增 finding=0。** 不表示已实现、代码 Review/Go/PG/HTTP/Case18 PASS 或人的接纳。采用元数据刷新不在本轮最终接收范围。

Reviewer：Kuhn，非作者 add-reviewer；设计作者仍是 Banach。角色由 Main 依据用户 standing 授权及项目委派规则指定，不是用户直接指定 Kuhn。runtime ID `01a1835a-8010-7a73-a8dd-3192719b6965` 来源为 Main 实际 spawn/resume 返回；本轮未子委派。固定 `D:\AID\VSCode\Codex\Han`、main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，无切分支或 Git 写入。Main 技术接收本 R02：**pending**；人的精确签字/接纳：**unknown**；Reviewer 未作风险豁免或采用决定。

## 对象与原失败链

原 R01 原样保留于独占目录 `C:\Users\J00010385\AppData\Local\Temp\han-recovery-design-independent-r01-9cd2ae97be154a82b466ff08617d8b11`。本轮实际重新核其报告 SHA `69BB9C4DC548902EC2A72AA57D84386B75B57097B14EDF4927D005F43D117195`、receipt SHA `28334886983FAC0C7B92972EB42AFF23BEE1AC525FF2AFB0EA045DA8EC4C1E08`；不覆盖原 CHANGES REQUIRED、两项 HIGH 和旧证据。Main 已现场技术接收原两缺陷且不风险豁免，这是本轮输入事实，不冒充本 Reviewer 新签发决定。

作者包：`C:\Users\J00010385\AppData\Local\Temp\han-recovery-design-f001-f002-5f5ccbee4b874ad2ac7c9d11be2961ad`。四份 R01 前像与原 R01 四 SHA 匹配；只有 identity 两份 diff 非空。版本号仍 r50/r17，故以精确 SHA 区分同版返工，不把旧 r50 审查结论泛化至任意 r50。

下表 before 为本 Reviewer 本 R02 开始读取的冻结摘要，after 为交付前只读复核；作者修订前像另列于 static-verification.json。

| 对象 | before SHA256 | after SHA256 |
|---|---|---|
| identity-auth/design.md r50 | 5BB7994F3AAC309ADC913C9B032782694849BB3C084FF5852B7270B14B8E3DAD | 5BB7994F3AAC309ADC913C9B032782694849BB3C084FF5852B7270B14B8E3DAD |
| identity-auth/stories.md r17 | F386D2BCDE59845D1AF62F6B781B006E677E5487D7A970FFE3E66097DEDA73D0 | F386D2BCDE59845D1AF62F6B781B006E677E5487D7A970FFE3E66097DEDA73D0 |
| article-management/design.md r23 | 436E323486F02E86BF985B92CB8B2DA096B9CC3440F629B1B2B0875B8EB2731D | 436E323486F02E86BF985B92CB8B2DA096B9CC3440F629B1B2B0875B8EB2731D |
| article-management/stories.md r11 | 33E6A46AE28B3CB5DBB3B3C5559895167B1BAB0B1F29AE73C60878A9662902A4 | 33E6A46AE28B3CB5DBB3B3C5559895167B1BAB0B1F29AE73C60878A9662902A4 |

四路径均位于 `.add/project/knowledge/design-workspace/`。独立逐 hunk 校验作者 diff 的旧 context/removal、行数并应用到前像：identity design 7 hunks、+140/-1；identity Story 3 hunks、+7/-0；均重建为冻结文本。article 两份与前像**逐字节相同**，仅核摘要/零差异，不重审 article 全文。BOM/原 CRLF 数量保持。主干 ref/HEAD 与四对象前后无漂移，实际逐段读取的仓内依赖亦核前后无漂移；未重跑全项目保护集。

## Round 1：修复增量及相关一致性

全文读取两份非空精确 diff，核 original finding→新端口→schema/事务→状态/异常→Story 的修复链。范围限 F001/F002 和它们引入的维护 scope、initial trust proof、恢复 operation/review 元数据、Store canonical/revision、同一 root 的适配与实施责任，不重新全项目审。

F001 增量不再要求用普通 resolver 总条件接纳 recovery_required=true，而明确同一 Session 所有权内独立维护解析/签发；F002 不再把日常 TargetEligible/SetCredentialMode/settleAccepted 当恢复路径，而明确恢复资格和条件写。两者对普通拒绝条件、可信来源、错误/取消、ACK loss、无 fallback 均有固定约束；没有只增加接口名而仍要求调用被现役前置拒绝的原顶层服务。

## Round 2：真实消费者逐项复核

### RECOVERY-DES-R01-F001 — CLOSED（设计层）

修复位置：identity design `:3571` 起；重点 `:3604–3628`；Story r17 S04 新验收。现役证据：`internal/identity/login.go:1310–1355,1438–1507`，`internal/session/store.go:175–341,549–690`，`internal/identity/service.go:105–115,197–207,710–750`，`internal/identity/bootstrap.go:332–364`，`internal/platform/http_security.go:284–340`，Root `main.go:2584–2610`。

| 原缺口/复审项 | 修订及真实消费者核对 |
|---|---|
| Login 要 true、普通 Session 要 false | `:3608` 固定 access_scope=ordinary/recovery_maintenance 及 run/install/runtime/gen 四绑定 CHECK；普通 resolver 保留 false 并新增 ordinary 过滤。`:3614–3622` 明确维护登录/专用 resolver 的两合法态，bootstrap+true 只到 me/password，其他维护写拒绝；普通 token/非initial/错 phase/错绑定无 fallback。原 false 条件不再造成维护路径必然自撤销，也未被全局放宽。 |
| 谁可授维护权 | `MaintenanceSessionAuthority.CheckLockedMaintenanceUser(ctx,tx,user,binding)` 由 identity 经同一 RecoveryRunStore 实现；binding 只从 Runtime 已核私有 phase/surface ctx 取得。导出 binding 结构本身不授能力；Create/Rotate/Resolve 每次在 User→Store 顺序重核。Header/URL/Cookie run、caller Principal、静态 maintenance bool 均不能构造许可。 |
| 实际接口能否替换 | HTTPAuthorizer 当前依赖最小 PrincipalResolver；维护 Store 以相同 ResolvePrincipal 签名实现，Root 可为两 surface 注入不同 authorizer，共用 pool/表/router。Login 的 Sessions Create/Rotate 和 PasswordChange 的 Rotate/Revoke 参数可由维护适配器桥接至显式 caller-tx CreateMaintenance/RotateMaintenance；签名中的 oldToken/time/reason 与现役原语一致。没有要求 Go 同名重载、第二 pool 或普通 Store fallback。 |
| 准备与强改密来源 | `:3610–3612` 在同一 User/Store 短事务固定 bootstrap G、run 初始绑定、审计/撤旧 Session；独立 secret 用现役 digest 字段，但 permanent generation=G、source_operation/source_credential=NULL，不伪装旧 reset 来源。hold/旧 unresolved 不被清掉。已准备/已改密的精确重试不覆盖密码。 |
| 原子退出 | `:3624` 明确专用 RecoveryTrustChange，不直接期待旧 SetPermanentPassword 清 flag。新摘要、两代际 G+1、mode=permanent、仅 initial 的 flag=false、run changed proof/audit、撤其他 Session及当前维护 Session轮换同事务；所有谓词/行数失败全回滚。没有“先假清 Gate 再调用普通服务”的中间窗口。 |
| ACK loss/Resume | `:3626` 按 run proof、User/Session 摘要/代际/审计精确回读，未知不重写；已提交但 cookie 丢失以新密码重新维护登录。新 RuntimeID/代际拒旧维护 Session，持久强改密事实不回退。 |
| 后续 review/普通面 | 改密后只允许已核 initial permanent 的维护 scope；`:3628` 明确禁新维护动作、真实 join、撤本 run 全部维护 Session并记审计，再撤维护能力，按既有 fresh/三 worker 门槛公共激活。普通面重新登录产生 ordinary Session；维护 cookie 不升级。静态 RecoveryMode 不再被当唯一动态准入来源。 |

结论依据：原报告的互斥前置已由一个受信、分 scope 的实际解析接缝和原子状态转换消除，且 ordinary 路径仍拒 true/维护 token。新 proof 字段和正向/负向/ACK loss/切换验收已同步 Story。无须扩大公共认证权限即可实现所声明的恢复流程。

### RECOVERY-DES-R01-F002 — CLOSED（设计层）

修复位置：identity design `:3630` 起，重点 `:3659–3687`、`:3741`；Story r17 恢复共享验收。现役证据：`password_reset.go:180–235,607–725,779–790,1293–1300`，`password_reset_store.go:83–173,210–277,500–554,1252–1284,1507–1609,1859–1868`，`password_reset_operator.go:173–220,313–354,489–590`，`password_reset_worker.go:87–168,322–399`。

| 原缺口/复审项 | 修订及真实消费者核对 |
|---|---|
| 日常资格拒 true/locked/不同当前 role | ResolveRestored 限 running/pending、非initial、未deleted、suspended+true+locked，actor为强改密维护 initial。`:3659–3663` 将可信 Role/Status/批准原件与备份 role/当前 role 明确区分，identity 定义 EvidenceReader，事务外读取并在同一 Store 锁内精确核批准。明确不调用日常 ResetPassword/TargetEligible/SetCredentialMode；旧三处拒绝仍保留。 |
| 同一 durable 机制而非第二协议 | `:3667–3671` 新恢复批准字段不可变，review 指向 exact operation。候选仍 initial_reset、原 purpose=password_reset，base G/target G+1、原 execution counter+claim+operation+pending credential+job 同事务；predecessor/replaced=NULL，旧 recovery_invalidated 历史保留，不伪装 resend 或先造旧 permanent/temporary root。 |
| Intent/幂等/source lineage | 内部 scope recovery_review_restore/v1 绑定 actor/target/run/install，request digest 包括可信提议 role/status/evidence，与 daily reset 分离。source_operation_id=本次新 operation、credential auth generation=target，current operation在同候选事务固定；不是把 backup/current 字段或 caller source token 当权限。target 当前 role 的旧日常比较不再拒绝可信改角色。 |
| 先处理旧 authority | `:3665` 在新秘密/ID/claim前分类完整旧 operation/credential/current lineage、hold 和同键事实。旧 unresolved/unknown 先走原合法无投递结算，未到 next_attempt_at 不提前；无法合法证明终结继续 pending/Gate，hold/dead-letter 不清。仅无旧 unresolved 的合法正向即可新建，不以全 denied 替代 restored。 |
| 单次投递/未知保护 | `:3673` 仅已确认新建且持原 secret 的 owner 可 ClaimMailAttempt→同一次 Send→原 accepted/rejected/unknown 记录；candidate ACK loss 先回读，不重发/造新秘密。SMTP 在事务外，受原安全/预算与真实 join 约束；accepted只保存受理证据/促结算，不提前改 User/review。Resume/Settle 无 Mailer。 |
| ordinary worker 停止时谁结算 | `:3675` 同一 identity executor 显式驱动 SettleRestored，限当前 recovery_identity/running/独占，依原 due、租约/owner/attempt/MaxRetries、连接 permit Claim/Renew；不 Start普通循环、不借 requeue 权限。durable actor 批准继续精确核验，不要求失联 HTTP Session 永远存活，但新 HTTP 写仍必须重新认证。 |
| 原 accepted operator 拒 locked | 修订已识别 `password_reset_operator.go:519–520`，不仅绕过 TargetEligible。`:3677` 明确受控条件写替代该分支，保留 converging/租约/credential/幂等分类原语；可在一个短事务先走原合法 recovery_required→converging，再完成 succeeded/temporary_active，不违背旧 trigger。 |
| 原子结算/Gate | accepted、pending/exact credential、未过期、User仍locked/true/baseG、current op/批准/review/lease都全等时，一个事务激活新 credential、User G+1/temporary/可信role-status/false、撤旧Session、operation/claim/job/review/Audit共同终结。只有提交确认才204；可信 suspended 仍停用。无提前假清 flag/mode；失败全回滚，unknown/expired/stale/hold不冒 restored。 |
| 终态/ACK loss | `:3681–3683` 扩展原 commit classifier，不能只看 operation.succeeded；精确比对 review/User/credential/claim/job/Audit，已提交零新reset/SMTP，未提交仅同 lease/证据合法才重试结算；部分/不明503。terminal review 同意图返回原204，不重复业务写，不同意图/键摘要冲突拒绝。 |

结论依据：原报告的恢复目标资格及仓储写谓词已由明确的恢复分支和真正原子结算承接；没有要求普通 reset 接受恢复冻结态，也没有靠先放开账号再执行旧 API。新实现责任包含 Store/classifier/operator 的必要共用接缝，属于后继 runtime 实施，并非给测试卡或已闭 S4.2/S10.5 写生产的授权。

## 修订相关新风险与约束检查

本轮未发现新增可触发缺陷；以下均为实际静态核对或明确后继验证边界，不另造 finding：

- 0002 sessions 的现有 credential_kind 已包含 recovery_bootstrap/permanent，新 scope 不需新增认证种类或改变 Cookie/Principal wire；普通 scope 默认值只保持旧会话原语义，旧会话不获得恢复绑定。0010 后禁旧 resolver/binary 混跑，防旧代码忽略 scope。
- 0008 的 credential source/auth_generation、User current-operation/counter/外键、initial_reset 的 predecessor/replaced=NULL、target=base+1、临时凭据 current/pending 唯一与候选同事务可兼容新分支。已有 deferred FK 可容纳 claim/operation/credential 同事务建立；review先存在，operation指回review后再绑定review指针可落地。现役已有 operation_id/user_id UNIQUE，可作为复合 FK 支撑；不需删改旧历史约束。
- 0008 operation trigger 的绑定不可变、not_started→dispatching→accepted/unknown、recovery_required→converging→succeeded 和 terminal 不可变仍适用；恢复批准字段需要0010追加对应约束，不是移除旧 trigger。没有将一次 UPDATE 跳过合法中间态作为必要实现要求。
- Prepare/强改密的新 G/G+1 均符合 0008 User auth generation 单调与 permanent_digest/generation 同空约束；root来源字段NULL是新的独立恢复信任根，不复活旧 reset lineage。旧 pointer/counter不靠清零避 Gate。
- 0002 pending review 的 reviewed_by/at/reset_at 必须NULL，修订明确候选期间保持；终结事务写齐 restored 字段。新跨 review/operation 组合 FK 防串账号，terminal指针/批准不可换。0010 proof/AdmissionRun/canonical 追加顺序与 NULL 保留一致，所有实际恢复绑定/proof/review写入递增同一 revision；原0005 run绑定/阶段不变，新阶段前置不会由 migration 假回填旧历史。
- 锁序固定 initial actor User→target User→同一 Store run→review→operation/credential/session/幂等审计。Store持锁的 Complete 全体检查不反向取 User/review锁；后台结算遵守 User-first。现有顶层 operator/backend 自己开事务，后继受控实现应复用其 SQL/分类原语与 caller-owned事务接缝，不能在已持 User 锁下嵌套调用另开事务的顶层方法；设计已把必要共用接缝和适配落点纳入新责任，未要求这种错误嵌套，故不因旧方法拥有事务而强报未实现缺陷。
- 现役公共 CheckRecoveryOpenGate nil 仍仅为维护且公共关闭；canonical 同一 Store running 锁读保持，框架未获得身份业务权威。普通 resolver/普通 reset 拒绝条件未放宽，unknown保护、完整历史、fresh Consume、S13–S16/两个 namespace/Source证据边界不改。

新批准 Reader 的真实提供方/批准主体/介质/ACL依旧 unknown，生产缺来源拒绝；这是已明示的环境/信任门槛，不是本轮设计缺陷。具体 adapter/SQL/trigger、factory共享实例、真实handler权限与ACK loss仍须后续实现和真实验证。两个 finding 的 CLOSED 只针对详细设计缺口，**不把这些未运行项写成 LIMITED finding、测试通过或实现已交付**。

## 采用元数据与原门槛

本轮**不签最终 metadata 一致性**。Main 输入的现役登记仍是 framework `BCB83191B5760826295BF661EAECDB2B873B31C26DF60B840E81B031DF639989` / README `157A549AD375B650479900A350464FCB842E828EFD296F3A50FA406A996FBAA1`、引用数3；identity采用条目仍指原R01冻结 `03D7...` 是已知历史状态。以上按 Main 来源记载，本轮没有重读/hash live decisions、没有改写或重新签发其 PASS。后续由 Arch 同一条目刷新最终 SHA/精确复审来源且不加计数，待 Main 提供固定 metadata 目标后另做定点核对。不重审ADR正文或architecture，不重开原架构F001。

S1.5/S4.4/S8.2 原 active 与 Case18 整项未通过保留；identity S4.5 真实 CAS/环境门槛、S4.6 retirement 未闭环持续禁用不变；已闭 S4.2/S10.5 不重开。无新 TaskID/CaseID，不改 TP/Task/Review共享状态。Main采用建议继续与人的成果接纳、独立技术 Review及下一写窗授权彼此分开。

## 验证、readset 与停止

本轮完整读 Role/add-code-review；同版全文规则沿本会话复用，8项规则/主Skill/Profile SHA与R01实际读本一致，见 static-verification.json。全文读两非空diff及作者verification，源码仅按原findings和新字段/结算影响定向读取。`readset.jsonl` 记录实际逐段读取的文件/摘要/LF行号，`readset.md` 聚合；摘要整个文件不等于全文已审。作者源码190/其他31knowledge/controls3无漂移是作者verification和Main的来源，不冒充本Reviewer全库复扫。

实际运行：只读 Get-Content/rg/文件名定位、SHA256、ref文件核对，以及独占Temp的diff hunk应用/字节比较静态脚本。所有静态断言通过；非空diff完整读取，输出长度限制导致的部分重复取段仅用于补齐/定位，不计作测试。

**NOT RUN**：Go/build/vet/unit/integration/race；PG/0010/SQL/trigger/并发事务；HTTP/SMTP/服务/worker执行；真实restore/CAS/provider/介质/ACL/备份/E2E；PlantUML语法引擎/渲染。未读ENV/DSN/key/真实秘密，未联网/安装，未Git写，未清理/删除/移档，未子委派。共享仓库全只读，报告/readset/receipt均仅本独占Temp，原R01保持。

**STOP**：报告、receipt及摘要清单完成后停止所有写入；仅再做只读冻结摘要核验。交Main现场技术接收；后续metadata目标核对另待固定SHA/diff，不替作者修稿、不代人签字、不作采用决定。
