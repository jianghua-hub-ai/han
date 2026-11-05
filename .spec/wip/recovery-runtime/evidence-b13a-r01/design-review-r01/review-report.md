# identity design r51 有界独立 Review R01

**CHANGES REQUIRED：1项 HIGH OPEN，RECOVERY-DES-R51-R01-F001；其余严重度0。** r51 为可达 Enter/Resume、前置目的和恢复退出补足了主要契约，但 Resume 读回秘密的 retained-Begin-key 验证尚未落到明确持钥只读执行者/成功返回合同。不能仅凭“先核 fingerprint”的目标文字认定该消费接缝已闭合。

本结论仅针对 r51 增量及直接受影响合同，不是全 r50/特性/架构重审，不评当前 F234 源码。Code finding `RECOVERY-B13A-CR-R01-F001` 不由本报告关闭，F002–4未复评；后续代码动作仍由Main另行接收/授权。设计继续draft、人接纳unknown。

**身份与固定对象**

- Reviewer：Kuhn / add-reviewer，runtime ID `01a1835a-8010-7a73-a8dd-3192719b6965`，由Main按用户standing与项目委派规则指定，ID来源Main实际spawn/resume；非作者，无子委派。
- 作者：Banach / add-feature-designer `01a18389-57ab-7b70-8c86-82a61e7729f1`，来源canonical author index（SHA `9415A71955228F61D704108ADA2AAD6F98CEA2E90265A8169640FF245EBDF671`），不作人的签字。
- Han `D:\AID\VSCode\Codex\Han` / main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；objectType=design，唯一被审共享目标 `.add/project/knowledge/design-workspace/identity-auth/design.md`。
- r50精确前像：`6634BED32EDE264F6DA6F47C9A7E8152ADD0B19C2DDCBCF3ED4708AE4BA4109D`。
- r51 Review前SHA256：`F87E1EB9073F4B2A723A098776762F759332B8086384BC0ED3C79DC49F624564`。
- r51 Review后SHA256：`F87E1EB9073F4B2A723A098776762F759332B8086384BC0ED3C79DC49F624564`。
- 取得本人完整Role、add-code-review、checklist/abstraction/产品安全红线、design-doc-writing-style、tracing/observability及已启用go-service入口；按非代码对象适用，不强套编译/测试/渲染门禁。发现不等实际读取，具体范围见readset。

**Round 1：完整实际增量**

完整取得canonical r50→r51 diff（SHA `BA0BE2C288A88218B0C966E881493EA7D6CDF86EACCE94310CA6C7F304DEF076`）、作者receipt、索引及必要r50前像对照。独立逐hunk验证13 hunks、+81/-8，正向重建r51及反向逐字节恢复r50；原BOM与169个CRLF均保持，旧确认/历史尾部未改。版本改r51，status仍draft；四新增相对链接/显式Task锚有效。PlantUML起止静态配对，不代表语法或渲染PASS。

逐项审了新RevokeRecoverySessions、RecoveryConsumer/RecoveryExit及依赖字段、mode-specific依赖表、recovery_restore/recovery_bind矩阵、Enter/Resume独立当前绑定与撤旧grant、Consume后独立succeeded核证、root真实退出/identity撤Session事务及ACK-loss、§6.1验证建议与§9.1原任务映射。没有以新类型/未来实现现在不存在本身报缺陷。

**Round 2：消费者与失败路径**

定点读取r51 §3.3.1接口/投影/错误及依赖、§4.2原正常证明与阶段矩阵/恢复/信任根/出口/命令配置、§6.1与§9.1；r50必要接口及旧3737惰性秘密段；两份Main采用consult全文（只作被审解释输入，不作独立结论）。参考Ptolemy原Code R01 F001及既有职责，不读取其当前五源。article r23只核2314–2324跨阶段append矩阵；ADR27只核相应权威/流程/关闭原则。Task固定SHA `11F945B37FC9998DBC77F65B0E7831AA5D66DDBAE4DA07E4001AF10D583E1781`只投影S4.7/10/13/14/16职责/精确文件和相关关闭边界，不重审TaskPlan。

| 接缝 | 实际检查结论 |
|---|---|
| 前置目的与产证 | restore只在独立Begin committed/精确批准plan/关闭排空独占后一次Replace；bind只在真实join及独立target/receipt后绑定Store；独立running读回才identity，identity_ready读回才replay。新目的不借normal.last、不准任意purpose、自报binding或SetReady；旧grant逐次撤销，接口调用在互斥锁外，失权迟到成功不授开放。未发现第二项阻断。 |
| 模式依赖/来源 | Normal/Verify不用不适用Runs/Identity/Consumer/Exit/Restore/secret；Enter/Resume自身依赖先检查，不以no-op/fake替代。Resume既定阶段不新Begin/盲Replace，不复Consume succeeded尾部；完整历史失败/缺失优先及唯一consumed窗口不扩大。secret核验仍有下述finding。 |
| 窄Consumer与Exit | Consume仍runID签名/原结果，同Store/Journal/source Coordinator保持生产consumer；返回值不冒独立来源。Exit复用当前replay grant并额外核succeeded/精确consumed-publication，未扩running身份权限或article追加。 |
| 退出与SQL可实施性 | complete真实返回后独立root控制流程禁新维护/settlement并join，避免等自己；随后identity同executor/Store撤所有本run安装的维护Session，含旧RuntimeID/代际与过期未撤行，错安装拒绝；User→Store→Session/Audit、集合复核、审计判重、零会话仍需首次审计、ACK未知独立回读均明确。nil不是caller done，确认后再撤cap/三worker/finalproof。 |
| 真实直接定义 | 只读取5个必要现役文件：0001 audit表/action/metadata/append-only，0002 Session/FK/revoked字段，audit.Store事件类型/调用者事务Append，identity RecoveryRunStore的pool/事务/锁行，session Store的User-first/tx解析。固定审计action可用现有text+JSON字段落地，不需新审计表/枚举；0010 scope绑定仍是已定待实施设计，未冒称现存。无现役文件修改或测试。 |
| owner映射 | S4.7中性端口/机制、S4.10同identity收口、S4.16 root join/配置/装配方向清楚；S4.8 ctx/Consume、S4.9唯一Store/0010、S4.13安装、S4.14工具职责保持。r51明确原Planner后续同步S4.10/16措辞/验证，不谎称当前原卡已逐字列明；不因待同步本身发finding。 |

**[HIGH / OPEN] RECOVERY-DES-R51-R01-F001：Resume秘密与retained Begin key的匹配缺少明确只读执行者/端口成功合同**

- 目标位置：r51 `design.md:1644`（新增Resume依赖/动作表）。相关精确定义`:1606–1607`、`:1533–1538`，继承的`:3798`；写消费者`:3659`。
- 现状：1644要求ReadInitialRecoverySecret后按retained Begin key核fingerprint。3798只写“root的惰性ENV适配”及“先用Begin所绑retained key精确核fingerprint”；没有说明Read方法持有/调用何种既有keyring与已独立验证Begin来源、是否在成功返回前完成该核验、缺key/错secret如何失败。Runtime所见Source仅`ReadInitialRecoverySecret(ctx,RunBinding)([]byte,error)`；Journal端口接secret的只有会写新Begin的Begin，VerifyAgainst只接RecoveryRun，snapshot只返回key标识/fingerprint而不提供验候选secret的合同。
- 真实依据：经本轮特别授权只读旧冻结`author-r01/journal-before.frozen`，366–400显示Begin用retained/active key调用`fingerprintSecret(key,secret)`并固定Begin key/fingerprint；463–495的VerifyAgainst只是把Store已有key/version/fingerprint与Begin比较，没有候选secret参数。它返回成功不能证明本次惰性读回秘密匹配旧Begin。普通SHA256(secret)、切到新active key或拿Store已有fingerprint对自己都不是该检查；Resume再次Begin也已明确禁止。未读取/hash当前live journal或另4个B13A源码。
- 触发：Begin用K1/secret-S提交，之后进程在PrepareRecoveryTrust提交前中断；重启时active可已变K2，Store仍证明initial信任根未准备，Resume读回候选secret。此时必须在PrepareRecoveryTrust前以K1验证，但声明的中性调用链没有可指出的持钥只读执行者/成功后置条件。靠适配器自行补一套未写合同的验证，并不能独立证明这个关口已落定。
- 后果：合规实现要停在秘密验证处，或自行添加未定key访问/验证通道；若误把ENV读取成功或VerifyAgainst成功当验证，会让错误秘密进入信任根摘要写入。此处报告的是可实施设计/权限前置缺口，不断言暂停中的F234源码已发生绕过。
- 归属与最小修订：回原设计作者Banach，把这项门槛落到明确持钥只读owner与现有端口合同。可明确root-owned RecoverySecretSource的成功返回必须先按独立已验的本run/backup/Begin keyID/version、对应retained key及原fingerprint算法核验候选秘密，并说明其复用的reader/keyring绑定、失败错误/零副作用和验证前零PrepareRecoveryTrust；若实际由Journal verifier持钥，则给出可消费的只读验证接缝及既有owner映射。无需新authority/schema、不得导出key到Runtime或以新Begin、普通SHA、新active-key、caller自报proof补洞。不能把3798的省略主语目标句当已明确的validator合同。
- 验证：同一正确secret在K1变verify-only/K2 active后仍可合法Resume；错secret、缺K1/错version/run/backup、Begin/来源漂移、取消均在零PrepareRecoveryTrust/零新Begin/零其他写入前拒绝。已准备/已改密时应零实际秘密读取；沿原buffer/Trace/ctx/独占及失败关闭合同。先静态核owner、输入/输出/错误和调用顺序闭合；后续源码/PG的真实证明另由其owner与独立Code Reviewer验证。

这条核验要求在r50已出现，r51新增可达Resume及动作依赖本次必须承接它；不是全量重开r50旧设计finding。无其他finding，无另行非阻断建议；不替作者选新架构，不以保守unknown暂停本身当bug。

**证据与限制**

独立执行仅文档读取/固定hash、13hunk正反向字节重建、BOM/CRLF/链接及PlantUML静态配对、指定Task投影与至多5个当前直接消费者定义核对。详细路径/行段/hash/命令及范围在readset、task-ownership-projection和static-verification。Main1184保护/五源F234稳定、作者32knowledge/3controls保护各按其归属，不冒本Reviewer全保护集扫描。未读/hash当前`runtime.go/admission.go/runtime_test.go/admission_test.go/journal.go`；旧冻结journal为Main补充明确授权，不是当前源码Review。

**NOT RUN**：Go/build/test/vet/race、PG/DDL、HTTP/SMTP/服务/worker/restore、实际provider/ACL/秘密或恢复工具、网络/安装、PlantUML语法/渲染、性能/部署/E2E/人确认。没有secret ENV/DSN/key读取、Git写、共享写、清理删除移档或子委派。报告不关闭Code F001，不复评Code F002–4，不授源码START；原S4.2/S10.5关闭、两S4.4/Case18/S4.5真实CAS/S4.6retirement门槛均保持。

**STOP/frozen**。只交Main现场核收，由Main安排原作者设计修订；不替作者改稿、不覆盖旧报告、不代人的采用或风险接受。
