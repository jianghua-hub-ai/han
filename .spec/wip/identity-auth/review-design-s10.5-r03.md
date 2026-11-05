# S10.5 六对象独立 Design / ADR / TestPlan Review R03

- 日期：2026-10-28。
- 实际 Reviewer：**Dewey / add-reviewer**；真实 Agent ID **`01a180fe-c3bb-7661-86fb-06af4f76ba1b`**。Lead 实际委派的原非作者 Reviewer；没有再次委派或替作者修改被审对象。
- 固定工作区 / 分支 / HEAD：`D:/AID/VSCode/Codex/Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- **结论：PASS，仅限下表六个冻结对象的本轮 Design / ADR / TestPlan Review。** 两轮有界检查完成，可操作 finding **0**，本次设计/测试规划范围内未关闭阻断项 **0**。
- **CODE F001 仍 OPEN；本结论不关闭它。** F002 仅按原 CODE R02 对固定 Root 修订 CLOSED；没有完整 Root/七文件 CodeGate、Case/Run、真实 PG/CLI、部署、生产或人的 accepted/checkpoint 结论。
- 输出只在本次新个人 Temp；不修改六对象、Tasks/Plan 或既有报告，不自行启动下一阶段。

## 1. 完整六对象索引

来源 `.spec/wip/identity-auth/evidence-s10.5/design-r03/contracts-fixed-six.json`，SHA256 **`478E11DB183DB40B53BF3980BF897753FD8E50331992A4B468455A8A0A4E5BF5`**。Reviewer 已逐项核清单、fixed-final 原字节副本与实际工作树一致，并复制到本目录 `fixed/`。五对象与 Phase1 完全相同，仅新增最终 TE r23。

| 固定路径 | objectType / 修订与实际范围 | 最终 SHA256 |
|---|---|---|
| `.add/project/knowledge/decisions/identity.md` | **adr**；ADR-0026 `adr-0026-r04` 的合作截止、真实 join、数值端点、结果/资源及来源增量 | `C35E7DC60004BBE6CE530AB0ADDF50A93FDDDCFA802D573E6B84FAFC5D96D321` |
| `.add/project/knowledge/decisions/README.md` | **adr 导航**；r04 查询、来源及原派生引用 count=2 | `59217A3621A10003B66630DFA1F0AC43E84A91B64CBE7AAE786D290BFF1B8DC0` |
| `.add/project/knowledge/design/architecture.md` | **design**；`architecture-ir01-r31` draft，本次维护配置/部署/进程/Q23 与元数据 | `6C9BD31DF00F91198AA66AC650240707D89D55E4257AC1272A5A2B3059C81F22` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | **design**；`identity-auth-design-ir01-r49` draft，§3.3.11/§9.4.3 及相关元数据 | `E884C84F4AABCA7ED321B9E4410EBAAFA0ED39CE6458986F92EA7C07D9BE4C43` |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | **design / Story**；保留 r16 头部的 S10 maintenance r49 增量，365–374 行 | `8B2F13A7BA112C798E750E54AD9BFC7C9D4F20C88DB9BB2662CD43245D3FC6F7` |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | **design / 测试规划**；`identity-auth-test-plan-ir01-r23`，S10.5 Case/API/资源矩阵/命令/接力及来源增量 | `4CDC882308A2CCF6EE249880EBA5EEA469C2C983933C4EE6D42AE1168F5F03D2` |

作者归属：Arch Epicurus 三文件、FD Ohm 两文件、TE Russell（`01a180fb-ea7c-7331-8ebf-b9855dfdd634`）test-plan。作者均完成并由 Lead 关闭写者后进入本固定对象；作者自检与 Lead 现场核对是输入，不替代本 Reviewer 的实际检查。

采用依据为 `cleanup-contract-adoption-r01/lead-adoption.json`，SHA256 **`3ED915618A722E910835FB39D990DAF932786A4131EE6E87B131F8238ED5B84A`**；原 FD 咨询 SHA256 **`0591453C50F13F4161155E6745C7B377D064965995F99DF0B278E5172697570F`**。已实读并核采用的是排名1整包，包含数值 primary/全部 fallback 限制。咨询“未采用”是拟稿历史，后来实际 Lead 技术采用有独立记录；本报告不另选方案，也不把技术采用当人的成果接纳。

## 2. 固定与增量取证

五对象复用本 Reviewer 已实际完成的 Phase1 两轮工作，原记录 SHA256 **`2C9F707CF2DB7C9DBA24A3409DB28048DFE5E5CA5AFFCF57F25676D339BA428D`**。它仅是当时五对象阶段记录，没有额外五对象 Gate；最终六对象结论由本报告形成。Phase1 固定五份与本次五份逐字节摘要一致，原记录和读集保持不变。

已区分 Arch 技术阶段的 identity `39321252BC447C75476D728737C8AAFB1BDAC8CBB7441CDC69A7E7AF510E881B`、architecture `3815C186080EC582D09ED345438F130F9EA5CA32296E6A8CC518EAC791267F45` 与最终 C35E/6C9B：后续三处状态元数据改引正式 Review/Tasks，README 不变。最终 FD 3799 行引用的三份 Arch SHA 与本表一致；没有用旧中间状态否定最终冻结对象。

TE 本次真实写前版本为 **`86B8C4026643F4B3B7876077510FDC29103D12952E7CF5FEB850D7437D5035C4`**，即实际 r22 加既有 §9.5 登记元数据；它不是历史 Design R02 的 E431 冻结对象。Reviewer 独立正向和逆向应用 **121 项精确字节编辑**，前像与最终 r23 完全互相还原，独立 diff 核为 **+165/-134**。§9.5、既有历史 revision 行及未触及文本保持原字节；新旧均 **UTF-8 无 BOM / LF**，未虚构 BOM。历史 E431 与其 Review 仍保留。

最终 test-plan 中 R03 pending 等头部表示本次冻结前的状态。此次实际 Review 结果由本报告承载，不回写已冻结 test-plan 的摘要形成循环引用，也不把冻结时点的 pending 当作缺陷。

## 3. 两轮实际审查

### Round 1：各对象直接缺陷与新增验收

五对象沿独立 diff 核全部变化行；具体范围与证据已保存在复用 Phase1 记录。本次再次核其身份与关键引用，未重新调研不变的全部 authority/wire/identity 历史。

最终 TE r23 实读：头部/来源 1–96、§4.4.3 的 25 Case（549–583）、Go API/DTO（585–596）、输入矩阵（598–608）、资源/错误/七 ENV（610–652）、严格编码与 F003/状态/资源边界（654–685），以及相关映射/§9.5/未来命令/阶段结论的全部新增变化。核各断言的可观察性、真正证据层和正反对照，不以 Case 名称或作者静态检查等同覆盖已运行。

独立核结果：原 **I-S105-01～25** 全部保留；表中状态明确指 **r23 合同验收 NOT RUN**。八项受预算/数值端点/层次调整影响的技术行是 02/05/06/11/15/20/21/22；另 C24 只修正已登记接力引用。其余16项技术列逐字节保持。没有把旧局部 U/SDK 记录清空成“完全无实现”，也没有把它们扩成新 r49 或整 Case PASS。

### Round 2：六对象消费、实际接口与真实载体

复用同版本的 Go1.27.0、pgx v5.11.0、puddle v2.2.2 源码分析和既有独立证据；本轮没有重跑探针或 Go/PG。阶段一已实际核 Config.Host/Fallbacks/LookupFunc/DialFunc、ConnectTracer/QueryTracer/AfterConnect、构造与 Acquire 取消分离、SDK Close/CleanupDone、TLS deadline setter、monotonic time，以及现役 operator 的 Config/opaque D/typed error/private cause/事务与 permit 接缝。接口可承载新合同，不需第八源码、公共新 DTO 或虚构可取消 KDF API。

本次补核最终 TE 对这些接口的消费及 carrier 当前落点。当前独立集成 carrier 的旧精确 Peek 名称与 `testConstructorLookupLate` 占位仍属待改代码；r23 已准确指出其不能验收，要求有实质断言的替代，不将编译或占位记为通过。Root/Operator 当前旧实现与计划的新 r49 分类差异是明确保留的实现缺口，不误登记为“设计已经实施却失败”。

达成一致的关键合同及最终规划消费如下：

| 合同 | 六对象及最终 TE 的实际消费 |
|---|---|
| D 停止新业务 | 原绝对 D 不重置；到 D/取消停止新认证、Acquire/连接交接、Ping/Requeue/业务写与提交核验，不延长签名 TTL。TE C11、资源行616–620及配置矩阵覆盖。 |
| C/K 合作截止与强制 join | K=D+S、C=min(cleanup_start+S,K) 不增加；不可取消的已开始 CPU/native 可越期甚至不返回，必须实际 join，不 detach/提前 return/假造0。TE C22、610/620/627/685 明确无硬 CLI SLA，保留原 SCRAM cutoff 仍活跃观察。 |
| sticky 单调 end | Root/Operator 各自 cleanup 超时/实际 end>C 均粘性记录，最终 join 后再核；延迟回调和计数归零不清标记。TE C11/15/20/21/22，625–627 行分别覆盖。 |
| 原成功取消/越期失败6 | 实际 join 后失败、stdout 空；Root 不吞 operator 的 typed failed6。unknown5 优先、其他非零主错误保持；TE 632–637、C20/21/22 与 FD 2356/2363/2387 对齐。 |
| 明确 commit / Audit 事实 | 清理或取消不改称回滚，不追加 timeout Audit/补业务写/另开核验/再 enqueue；后续只按原授权审计重放。C15/20/21 和资源625覆盖。 |
| TLS 独立网络义务 | 三 setter 含 clear/later 裁 C，callback 覆盖同步 SDK Close 并实际 join，健康先 SDK Close→CleanupDone→Pool.Close。C22/624及正常 apply/replay 对照保留；CPU 越期例外不豁免额外5s/15s网络窗口。 |
| 全部 numeric 端点 | parse 后 connect 前核 primary 与全部 fallback；hostname/隐式 localhost 非法配置2、OS Lookup/Dial/connect0；literal Lookup/numeric Dial不启动 OS DNS。C02/05、628/649/652与真实 B/精确 U 互补；普通 server/TLS验证/SNI不改。 |
| 未变化安全合同 | pre-DSN真实SID/独立签名、canonical/int64、User-first/完整before、F003 ReadCommitted/scalar→text Get、13-string audit、one enqueue/MaxRetries2→5、hold/终态/零邮件和server兼容仍由原25Case承接；未重新选择 authority 或扩大业务动作。 |

已阅读 Dalton 的 `g0-r49-overlay-r01/impact-and-proof.md`：作者 scriptedTx overlay 显示明确 Commit 后取消仍返回 nil/applied，unknown control 保持5。它证明其 U 边界的旧实现缺口，不是真实 PG/CLI 或本 Reviewer 的运行。最终 C15/20/21/22、operator typed failed6 与 Root/Operator 一致分类已覆盖这一预期；故不以此否定新设计，也不把新设计 PASS 当作缺口修复。

## 4. 25 Case、三个入口、五命令与11前缀

三个原顶层名与五条未来命令已独立逐字核对，与实际写前版本完全一致：

```powershell
go test ./cmd/han-server -run '^TestPasswordResetMaintenanceRoot$' -count=1 -v
go test ./internal/identity -run '^TestPasswordResetMaintenanceOperator$' -count=1 -v
go test -race ./cmd/han-server ./internal/identity -run '^TestPasswordResetMaintenance(Root|Operator)$' -count=1 -v
go test -tags=integration ./test/integration/identity -run '^TestPasswordResetMaintenancePostgresCLI$' -count=1 -v
go test -race -tags=integration ./test/integration/identity -run '^TestPasswordResetMaintenancePostgresCLI$' -count=1 -v
```

这些是未来验收选择器，**本轮未执行**。测试不存在、no tests to run、skip、空叶、必失败占位或只编译均不能通过；测试 helper 进程不冒真实 `han-server` binary，父进程 race 不自动证明另建生产子 binary 的 race。

第三 carrier 的11项指定前缀及真实层次已核对：

| 前缀 | 证据边界 |
|---|---|
| `I-S105-06_half_no_eof` | B：普通半输入，parent writer 保持至 child 退出，producer 实际 join |
| `I-S105-06_complete_no_eof` | B：合法完整 signed envelope 无 EOF 仍取消，不借 parent EOF |
| `I-S105-06_blocked_producer` | B：128KiB 当前测字节上限/部分写与 writer 归还，不冒业务 deadline |
| `I-S105-06_positive_bytes_no_eof` | B：普通正字节无 EOF/取消；精确 Peek-positive→ReadFile 前取消只由 Root U 控制 |
| `I-S105-16_oversize_uuid_guard` | B 拒绝/数据事实＋PG domain adapter 的 scalar/Get 传输证据，不混同内部计数 |
| `I-S105-17_absent_guard_uuid_race` | PG 两连接 domain adapter/observer 的精确 barrier；不冒 CLI 精确排程 |
| `I-S105-22_shutdown_1s_socket_cleanup` | B＋PG 当前 User 锁取消/无 backend 事实；不独证 TLS/SDK 内部顺序，不设通用1.5s返回上限 |
| `I-S105-22_applied_idle_sdk_close` | B＋PG apply/SQL/Audit/无 backend；Root U/SDK 补内部关闭因果 |
| `I-S105-22_replay_idle_sdk_close` | B＋PG replay/零重复写/无 backend；内部顺序另证 |
| `I-S105-02_05_numeric_endpoints` | B 有效授权＋EOF 下 primary hostname/仅fallback hostname/implicit localhost 拒绝及 literal 正例；U 证明 OS Lookup/connect 精确0 |
| `I-S105-19_one_enqueue_max_retries_2_to_5` | PG＋B 的一次 requeue、真实 worker/lease及随后多次自动 attempt，不改 worker/零重发邮件 |

仅两个前缀按真实层次更名/替换：原 `I-S105-06_positive_available_cancel_before_read` 在 B 中改为 positive_bytes_no_eof，原 mandatory `I-S105-22_constructor_lookup_late` 改为 numeric_endpoints。精确 native 窗口和迟到 literal-Lookup 仍在 Root U；旧 prepared 名称/失败占位和 raw 历史保留。新 numeric 叶要求真实断言，而非删去阻断、skip 或空叶造数。不新增 CaseID、生产 debug hook 或第八源码。

保留的七路径为原 root main/main_test、两个编译选择 stdin adapter、identity operator/operator_test、独立 integration carrier；Root/Operator U、真实 SDK-peer/原语、真实 PG/B、部署 D 分层没有被“第三载体”名称混为同一证据。

## 5. 接力、历史与实际验证限制

§9.5 原字节保持，引用已登记的 S10.5/B6C-4 七路径同卡闭环及 S3.4/B12 的 C24/25 部署接力；没有把当前 r49 的规划对账/源码修订宣称已完成。本 Gate 不更新 Tasks/Plan，不代接收人签字或授予运行权限。

实际 launcher/ACL、trust/审批 key 登记保管签发、人员归属、第二真实 SID、数值IP/TLS证书匹配、撤销排空/在途unknown及恢复后 epoch、真实 ack-loss 注入与 E2E 仍分别未验。软件合成验证不以生产 key 已存在为前提，也不能据合成 SID/签名声称生产签发或两自然人在场。S3.3/S3.5 窄未验/豁免和 S3.6 原条件不变；不重开无变化历史卡或整个 identity Review。

旧 domain 部分 U（原 normal/race 126nodes/114leaves）与 F002 的真实 SDK/TLS 定向事实保留其来源、修订和限制；它们不是本轮执行，不能充作 r49 全部语义或整 Case 通过。当前 carrier 仅编译的历史保持；新 r23 合同及 PG/B/D 验收仍 NOT RUN。旧 readonly PG180006 preflight 不是 Han Case/DDL，旧六次 stdin probe 不是原漏取消竞态证明。

本轮实际执行仅只读文件/SDK/证据、SHA256、独立 scoped diff、121项字节正逆还原、Case/命令/前缀核对和输入不变性检查；**没有 Go build/test/race、PG、CLI、DNS、新探针、DSN/secret/key读取、账户/GRANT、Git mutation、清理或委派**。复用同版本完整 Role/Skill/规则和 Phase1 实际读集，不把 hash-only 记录称为语义全仓审查。

本次还独立核原185源码清单中，除四个获准修改的旧文件外，其余 **181** 原字节不变；原清单加三个获准新文件为当前 **188** 范围，另记录 go.mod/go.sum、migrations README 等控制摘要。此为变动边界核验，不是188文件 Code Review。六对象、当前七源码/控制及旧报告在结束再次校验；检查详情见本目录 verification/readset/proof JSON。

## 6. 最终分界与交回

| 范围 | 本次结论 |
|---|---|
| ADR r04 / 索引、architecture r31、FD r49 / Story | **本轮设计/ADR Review PASS**；已采用代价与接口/消费者一致，无新增可操作 finding |
| 最终 TE r23 | **TestPlan Review PASS**；25 Case/资源/层次/五命令/11前缀与五对象一致，真实运行状态保留 |
| 六对象合并 | **Design / ADR / TestPlan PASS**，仅上表完整摘要对象 |
| CODE F001 | **OPEN**；新合同采用与本 Review 均不关闭，待实际新实现及原 Reviewer 代码复查 |
| CODE F002 | 仅 CODE R02 对 main `2C2D404F344E5A8E4831C34FB4C240B234B9B099482F13897D38EA470B37FF79` / test `D46BA87371795849FD525454FB4E67942E7C3002051867B3F335D55EBE0E83B2` 固定修订 CLOSED，不外推后续源码或完整 CodeGate |
| Root-ready / 组合CLI-PG / Case Run / 生产 / 人的接纳 | 本报告不放行、不执行、不代签；继续按原责任与实际前置处理 |

原 Design R01 SHA `FAA1EAA42C2CF3AD8495611E23EF24BD731B5641D51574FE5713FE6D817EBA01`、Design R02 SHA `0EC5613B883F4853AF72A6A7324FE9E0E2661450F3305C61DDD2DCF5CF5BDCED`、CODE cleanup R01 SHA `8C1063321915846CA299E53DBC58CF037507EFE8015819A2E1EF84599AC0359A`、CODE cleanup R02 SHA `CB2BB815A36E7EE7393BB80F1BCD6D54047214C79ECD377F7783EF42E59C3580` 均保持原文与摘要。原失败、设计层旧闭环、真实 SDK/原语来源和 r48 保证被本次明确替代的关系保留，不改记旧失败为通过。

本目录保留 `objects.json`、`readset.json`、`scoped.diff`、六份固定副本、TE 原字节及正逆证明、复用 Phase1 记录；最终报告 SHA 另写 `verification.json`，避免报告自包含摘要循环。

**交回状态：stopped。** 本次六对象有界结论交 Lead；Reviewer 不自行开启规划、CODE、测试执行、部署或其他后续 phase。
