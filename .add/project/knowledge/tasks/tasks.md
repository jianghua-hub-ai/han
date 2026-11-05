# IR-01 内容发布社区任务 SSOT

> 本文件是 implementation/test task 三态唯一源。原 47 个 Story 派生条目为绿地 G 模式；identity Story S04 的 R 对账新增 1 个独立任务，2026-10-23 identity Story S03 / F005 的 accepted r26 / test-plan r08 规划修订复用原 `identity-auth-S3.2` 并保持 2 个相互分离的部署/验证任务；identity Story S07 的代码复审设计缺口另新增 1 个独立 migration/index 任务；article Story S04 production wiring 的 R 对账新增且仅新增 `article-management-S4.6`；2026-10-24 identity Story S10/S11 durable password-reset R 对账保留既有 `identity-auth-S10.1` / `identity-auth-S11.1`，为 schema/persistence、三消费者 Gate、worker/composition 三个不可由原卡独立验收的所有权切片新增且仅新增 `identity-auth-S10.2`～`identity-auth-S10.4`，并依据 design r44 / test-plan r20 新增且仅新增待稳定设计的 maintenance 候选 `identity-auth-S10.5`；`project-bootstrap-B0.1` 是用户直接授权的项目启动初始任务，不经 task-decompose。Plan 只引用本文件，不复制状态。

## 依据与范围

- **本轮验证范围覆盖说明（2026-10-27，ADR-0025）**：Han 仅为测试项目；S3.3 免真实浏览器连接测试，S3.5 免真实平台与校验工具验证，S3.6 不变。以下历史“当前事实”中的浏览器/平台阻塞不再作为 S3.3/S3.5 本轮前置；各卡剩余核证和关闭记录为准，未执行不冒称 PASS。
- **项目级验证授权（2026-10-31，ADR-0028）**：Han 为测试项目；可执行的 build/vet/gofmt/unit/component/contract/模拟集成仍须实际执行留证；真实外部环境不可用时可用模拟环境、确定性 fixture 或建议值替代，并显式标注“模拟/未真实验证”；评审/签收可由 Agent/AI 代理并记录授权来源、代理主体与范围（“AI 代理评审，非真人签字”）；部署目标、责任人、环境租约与演练窗口采用建议值。不得伪造 PASS、泄露凭据、未经具体清单确认删除/覆盖或执行 Git 写；ADR-0024/0025 原记录不回写。各 active/todo 卡均按本条执行。

- 需求：requirements r16，accepted checkpoint `decision-e3af3912166f7287e8324488cad1cf1b87ba1034fca3378d3e0467a23751592d`。
- 架构：未受当前 scoped override 影响的范围继续使用 architecture r15 与 accepted checkpoint `decision-5e55881bdc4a7e038b3764d91ff0dfd906a48fe121e93bbb0813ca58173cae40`；S03/F005 production observability 使用 architecture r23；password-reset durable 对账使用 architecture r27（revision `architecture-ir01-r27`，SHA256 `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB`）。
- 边界：identity r13（revision `identity-auth-boundary-ir01-r13`，SHA256 `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3`）、article r13（revision `article-management-boundary-ir01-r13`，SHA256 `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E`，accepted）；边界状态不得被外推为实现、测试或代码 Review PASS。
- identity：password-reset durable 当前规划来源为 identity design r44（revision `identity-auth-design-ir01-r44`，SHA256 `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23`）、stories r16（SHA256 `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3`）与 planned-not-executed test-plan r20（revision `identity-auth-test-plan-ir01-r20`，SHA256 `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB`）。Design Review R02（`.spec/wip/identity-auth/review-design-r44-s10.4-r02.md`，SHA256 `9A76D7E3C5BC29855043784057C99383844A0B3CC7F14E32F96EDBB5290F0434`）确认 r43 F001～F003 与 Article Start 待决项全部 CLOSED、无新增 finding、Design Review Gate PASS；S10.3 的 design r42 / Code Review R02（SHA256 `40D264C2E694E9CD5596DAD3FC1C574385F4CC37B08A22CA3D3127AB4769287F`）关闭链继续有效。design r40、Design R01、test-plan r18、Test-plan R02、增量 Planning R03 与 S10.2 Code Review R05 继续作为 S10.2 已关闭历史链。S10.1 Code Review R04（SHA256 `CA0AE21ED8A4B2AC61144CC165597BF3ED6DB3DFDD4D1054E7AF791EF0D8D3DE`）的 F001/F002 均曾 OPEN；R05（SHA256 `B313C68492B2C0CFC8459372363FCC2DFEBB5B9295179DA8DBAA1E6FB71EB299`）确认 F001 CLOSED；R06（SHA256 `2EA7503C7F33DEED7D697ECEC26ECFBA6BB7E0CD5995EF52BA12936A3D4117FC`）确认 `IDENTITY-S10.1-REV-R04-F002` CLOSED、无新增 finding、Code Review Gate PASS。B6C-3 integrated Review R03（`.spec/wip/identity-auth/review-b6c-3-integrated-r03.md`，SHA256 `B2E9CCF05A3D009436194BDCCFDA2064046BC13130E0BE85DD399703C99A5C4A`）确认 R01-F001、R02 residual MEDIUM 与 R04-F002 全部 CLOSED、0 新 finding、integrated Gate PASS。B7 G0 后的 S10.1 classification repair 由 R07（SHA256 `BAEB4C7BA121A71EE9E66934DC933365BF73D0AC0545B6887BCC078EBC99C27C`）记录 `IDENTITY-S10.1-REV-R07-F001`，R08（`.spec/wip/identity-auth/review-code-s10.1-r08.md`，SHA256 `30F1C4D2546F6E32BA23FCBECAD22235D49574CE35E66A236A8CCA18712DDC20`）确认该 finding CLOSED、0 新 finding、Code Review Gate PASS；Lead 于 2026-10-25 完成固定哈希/branch/HEAD/staged/越界、mailer/identity 定向普通与 race、`go vet ./internal/mailer ./internal/identity`、五文件 gofmt/diff-check Gate并 PASS。上述事实不外推真实 PostgreSQL/SMTP、部署、E2E 或 production readiness。Q1、S03/F005 与未受影响 Story 的历史规划依据保持原引用。
- article：S4.6 历史规划来源为 accepted boundary r13、draft design r10（revision `article-management-design-ir01-r10`，SHA256 `6785977683CF0F59E277667425F7FEE9714EBBD436FBDFE046E607F2E77CA67E`）、stories r07（revision `article-management-stories-ir01-r07`，SHA256 `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05`）与 planned-not-executed test-plan r08（revision `article-management-test-plan-ir01-r08`，SHA256 `4608502D4B4E9078509651A52589245D6726634053FF8A5E787569C55776EA7E`，blob `9079f7c33d2b56c8439f161a11be6425a5bf1687`）；design r10 独立评审（SHA256 `1EB2D78179D40747904CCC8F5E975FAB4A5CD4205C754567F6B5C589BAFC001E`）Design-side Gate PASS，Rawls r04 `.spec/wip/article-management/review-test-plan-r08-planning-findings-r04.md`（SHA256 `832ED2262DBF443CA3A172C88E5A904A206296771CA309A44EE68D111909F6EA`）确认固定 r08 Test-plan Gate PASS；Hypatia r02 `.spec/wip/article-management/review-tasks-plan-s4.6-r02.md`（SHA256 `5625061C2963AF1CF29C44D2E1DD15A53FA4F911E5F8CD424BE459681755639E`）确认 `ARTICLE-S4.6-PLAN-R01-F001/F002` CLOSED、Planning Review Gate PASS。Faraday r03 `.spec/wip/article-management/review-code-s4.6-r03.md`（SHA256 `007C2D9B04F1DC9381C574A668FB8C513F4B34D2B8224808F0512B623573E7E1`）确认 S4.6 的 F001～F004 全部 CLOSED、无新增 finding、Code Review Gate PASS。S2.1 独立代码评审 R01 `.spec/wip/article-management/review-code-s2.1-r01.md`（SHA256 `36BB65973CDC5998567B57B8C10AED1F6B65E114DEB5560414B19179540EF129`）曾为 `CHANGES REQUIRED`；R02 `.spec/wip/article-management/review-code-s2.1-r02.md`（SHA256 `01373C88B3E706A0EB0573BEEAFB67668E8C6537061EB6BE8F6FCB86D4CA8D4F`）确认 F001/F002 CLOSED、0 新 finding、Code Review Gate PASS，Lead 同 revision Gate PASS并关闭 S2.1。S4.2/S4.5 当前稳定规划输入为 draft design r14（revision `article-management-design-ir01-r14`，SHA256 `AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`）与 planned-not-executed test-plan r12（revision `article-management-test-plan-ir01-r12`，SHA256 `85D8A7CC6ABF01A88B24726EEE3957691AA0B8A7B6235FEB8A31677D77336DDF`）；Design Review R02（SHA256 `85DAC7A66F9555E6DF711472E7F5A9A6448F281EC2496DFDDBD5B7DAC19DE143`）与 Test-plan Review R02（SHA256 `53E85D1A4B6A4519829A04A7B6549E808F00E9BAD6D2F85C11EDE11E531C1D0C`）均 PASS，只提供返工规划合同，不关闭 S4.2 Code Review R01 的 3 HIGH + 2 MEDIUM，也不授权实施。r06 review PASS 与 r07 review FAIL 仅保留历史追溯；上述结论不外推为真实 PostgreSQL、Unix 持久卷、旧备份恢复、灾备、Host、部署、E2E、容量或 production readiness 通过。
- ADR：ADR-0001～ADR-0028；Profile：`go-service`。
- 当前事实：B0～B4、B5D、B6A、B6A-W、B6B、B6C-1 / `identity-auth-S10.2`、B6C-2 / `identity-auth-S10.3`、B6C-3 / `identity-auth-S10.1` 与 `identity-auth-S10.4`、B6D / `identity-auth-S11.1` 的关闭证据保持有效。2026-10-25 B7 / `identity-auth-S5.3` 的原 G0 classification 阻断历史保留；其上游 S10.1 五文件 repair 已由 R08 与 Lead Gate 闭环，随后 S5.3 在固定三文件完成返工，R01 `CHANGES REQUIRED` 的 F001～F004 已由 R02 全部确认 CLOSED、0 新 HIGH/MEDIUM finding、Code Review Gate PASS，Lead 同 revision Gate PASS并关闭本卡。article S2.1 的 R01 失败历史保留，R02 与 Lead Gate 已关闭本卡；S4.6 不重开。article S4.5 已在固定四文件完成实施与返工：Gauss R01（SHA256 `44DE54697DFEA8964E62A9AE2B8196C24E3233A44E177E04410F331AB64E5FF8`）为 `CHANGES REQUIRED / FAIL`（3 MEDIUM + 1 LOW），Aristotle the 2nd `01a17225-a3d0-7fd1-aeb9-622f7617f436` 在原边界内仅修改 Go 测试文件，Gauss R02（SHA256 `49340355047BFFF8A3F02A531AE20E9D163F65D6167E3190C1BC5FE5BC0E1F48`）确认 F001～F004 全部 CLOSED、无新 HIGH/MEDIUM、Code Review Gate PASS，Lead 同 revision Gate PASS并关闭本卡，四文件独占写窗口已释放。2026-10-26 identity design r45（revision `identity-auth-design-ir01-r45`，SHA256 `CBE0F64ADB93E1AFD4340D2D3390ACE42EE85F557DDCBA5AEBCF2AA20CE404F1`，Lead 接管产出，无 Feature-Designer Role 产出）把 S9.1 R01 的 F001～F004 转为可实施合同并经 Design Review PASS；Lead 随后按 r45 在原五文件内实现 F001～F004（无 Backend-Dev Role 产出），并以作者自审口径由 R02（`.spec/wip/identity-auth/review-code-s9.1-r02.md`，SHA256 `1F1D926EFFE92F4BB0DAEC86FA3037071BF3504CEAB6AE3B623391E223E18E23`）确认 F001～F004 全部 CLOSED、Code Review Gate PASS，S9.1 已关闭（完成@2026-10-27）。article S4.2 亦已于 2026-10-27 由作者自审 R02 关闭（完成@2026-10-27，写集未越出登记 19 文件）。article S3.1 与 identity S1.5 同日以作者自审口径关闭。B8A 的 `identity-auth-S6.1` 于 2026-10-27 完成：在四文件写集内新增 `revoke`（`admin → member` 角色转换 + 全部目标会话撤销 + `identity.role.revoke` 审计），复用既有统一 Policy 与用户治理查询/状态/删除 handler；作者自审 Code Review R01（SHA256 `2A612A4B931A5BA4E8B960277D1337AF4477C5CDB6930C5031A91F2A8CD7BCF7`（2026-10-31 脱敏后；原 `B76A590923A6F61F1E9E0085B2EA26B6B7DAAA5340DCB46460AAD06C6B2AF944`））Code Review Gate PASS（作者自审口径）、0 finding；`service.go` 登记但无需改动。**未完成项**：`/api/v1/admin/administrators...` 整片治理路由（含既有 list/get/suspend/restore/delete）尚未在 `cmd/han-server/main.go` 唯一 composition owner 内注册，须由后续卡认领。S3.3/S3.5 保持 active / `blocked-for-execution`；S10.5 保持 `[ ]`、未启动；S3.6、B8 后继与部署/E2E 均不启动。identity-auth-S1.6 已于 2026-10-27 关闭（真实 PG18.6 上 12 组矩阵 PASS；并在真实 PG 上触发既有 P0 缺陷 ISSUE-identity-auth-0001（保留字别名 current_user），经用户授权按最小修复处置、回归通过；本 Host 无委派能力，Code Review 以作者自审口径 .spec/wip/identity-auth/review-code-s1.6-r01.md（SHA256 EFBDE5D35FD198888CB4CD40624C5947C1FE9E0486A4E112847BF9D62D40C485）PASS，作者自审弱于非作者独立复审作为已知限制保留）。当前三态按真实 checkbox 核对为 86 / 86 completed / 0 active / 0 todo（`article-management-S1.5` 按 ADR-0028 关闭，三张部署后验证卡按 ADR-0036、六张生产恢复卡按 ADR-0037 于 2026-11-05 以范围决策关闭）；S5.1 依据本轮授权单卡推进，历史“不启动 B8 后继”不否定本轮继续授权。`article-management-S4.3` 已于 2026-10-27 按 ADR-0024 以「经决策不验证」关闭：本项目为测试项目，真实 Unix 独立持久卷上的 `flock`/`fsync`/崩溃/磁盘故障韧性验证不纳入本版交付范围，三文件未创建、未执行，未验证边界如实登记，不以 Windows/fake 冒充。此外，`article-management-S2.4` 已于 2026-11-05 按 ADR-0033 以「经决策不验证」关闭：本项目为测试项目，真实 PostgreSQL 18.6 的 calibration/small/medium 容量与性能补测不纳入本版交付范围，R09 八项 PASS 与 R10 源码/离线检查不作为整组通过，未验证边界如实登记，不宣称 SLO、目标容量、P95 或 production readiness。

- 2026-10-27 identity S1.6 关闭变更记录：Lead 以作者自审口径关闭 identity-auth-S1.6（Host 无委派能力，无 Reviewer Role 产出；依据用户 2026-10-27 授权「由作者自审 Code Review」与「对待确认或决策项采用建议值并继续推进」）。在登记三文件 	est/integration/identity/postgres_test.go、ixtures.sql、aults_test.go 内于真实 PostgreSQL 18.6（127.0.0.1:5433）通过 12 组 S1.6 矩阵：唯一性/FK/CHECK、bootstrap 与成功墓碑、会话撤销全量、幂等、审计失败零副作用、治理竞争单写者（55P03）、规范化删除回滚/成功、会话解析拒绝已撤销/停用用户、0008 preflight（55000）、五对象原子性与 deferred 计数器缺口、lease 单赢家与过期接管、绑定不可变与 generation 不可回退。写集越界（第 4 文件 matrix_test.go）已无损并入 postgres_test.go 并删除，最终写集严格为登记三文件。真实 PG 触发既有 P0 缺陷 ISSUE-identity-auth-0001：internal/identity/password_reset_store.go 以保留字 current_user 作表别名致 42601，使 session.Store.ResolvePrincipal 全部已认证请求在生产 PG 失败；历史 S10.2/S10.3 仅 stub 与 strings.Contains 验证故长期隐藏；经用户授权最小修复（别名改为 current_account，4 文件，password_reset_store.go 哈希由 6C013640...3315F 漂移至 D1CB46D5...FB62），真实 PG 回归通过并登记。作者自审 Code Review .spec/wip/identity-auth/review-code-s1.6-r01.md（SHA256 EFBDE5D35FD198888CB4CD40624C5947C1FE9E0486A4E112847BF9D62D40C485）Code Review Gate: PASS（作者自审口径）、本卡三文件 0 finding；Lead 现场核 3+4 文件哈希、专用命令、race、Manifest 七项与 integration vet。三态更新为 57 / 36 completed / 3 active / 18 todo。未验证边界：真实 SMTP provider/外部 SMTP、部署/E2E/浏览器、容量/P95 与 production readiness 均未验证；FOR UPDATE 并发经 lock_timeout 确定性化，非压力测试；作者自审弱于非作者独立复审，作为已知限制保留。

- 2026-10-31 项目级验证授权与 S4.19 当前状态变更记录：用户作为项目责任人确认 Han 为测试项目并授权按 ADR-0028 推进（可执行层仍执行、真实环境可模拟、AI 可代理评审签收、部署目标/责任人/租约/演练窗口采用建议值）。`identity-auth-S4.19` 的作者 R01 已完成并经 Lead Gate，但非作者独立评审 R01 为 FAIL：`S419-CR-F001`（`OpenAfterCompleteSoftwareRecoverySameSession` 未执行完整恢复链，直接 `setState(public_ready)` 后断言 200）与 `S419-CR-F002`（`NoRecoveryConfig` 未调用 `planRecoveryStartup`、未断言 `present:false`）均为 MEDIUM 验收证据缺口，不是已证实的生产缺陷；报告见 `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r01/report.md`。R02 作者随后完成证据包，独立复审 R02 为 FAIL（`S419-CR-F001-R02` OPEN，测试名夸大完整链；`S419-CR-F002` 与 N1 CLOSED）；Lead 按复审路径 2 作正式收窄（只改验收措辞，不改生产行为与写集），作者 R03 改名 `PublicReadyAdmitsSameSession` 并复跑通过，独立复审 R03 与 Lead Gate R03 均 PASS，本卡已关闭为 `[x]`。收窄把完整恢复链交给同修订必跑的真实 PG `TestRecoveryRuntimePostgres`，并明确无单测试跨层串联，不得声称单测试证明完整链。S4.19 的 R01 FAIL、R02 复审 FAIL、R03 正式收窄与授权适用同时回写 Plan B13H；其他 active/todo 卡按 ADR-0028 执行，未真实验证项必须标注，不自动改写已关闭卡。 2026-10-31 晚：B13H 批次 Gate 独立结论 FAIL（`B13H-GATE-F001` HIGH：S4.19 dispatcher 层 (b) 在 gate 修订缺失），`identity-auth-S4.19` 已重开为 `[-]` 返工，S4.20 关闭记录已加 F002 限定；详见本文件 S4.19 段 2026-10-31 返工记录。

- 2026-11-03 identity-auth-S4.4 关闭变更记录：用户作为项目 owner 于 2026-11-03 确认 51934／51935 已移出 WinNAT 保留段、本项目不需要 (iv) 独立故障域、两处秘密存储可同机（测试项目）、目标部署环境与容量预算采用建议值，并授权按建议值推进。Lead 据此以范围豁免（`waived-for-test-scope`）关闭 `identity-auth-S4.4`：三测试文件在合成 fixture/替身夹具下经非作者复审 r04 PASS WITH CONDITIONS（1 LOW + 4 INFO，全部已由 `report-r04-errata.md` 登记）并复跑 PASS；真实 POSIX、真实持久卷、生产 Anchor／retirement、`GOOS=ios` 与独立故障域为 NOT RUN／unknown。requires 中的 `identity-auth-S4.5`、`identity-auth-S4.6`、`identity-auth-S9.2`、`article-management-S4.2`、`article-management-S4.4` 状态不变，生产恢复 Gate 仍关闭。三态更新为 73 completed / 2 active / 10 todo。**哈希时点提示**：ADR-0030 r20 中引用的本文件 SHA-256 `E034155076F77F8D1E8288D9BB4014554A5019882051A03A305A0ACA6AD71EFB`（912,450 B）系 r20 时点值，本次关闭后哈希随之变化；ADR r20 记录保持不回写。
- 2026-11-03 identity-auth-S2.2 关闭变更记录：本卡按**测试范围**关闭（`waived-for-test-scope`），依据 project owner 同日授权「其他需要我输入的请按照建议值」「本项目为测试项目」与 ADR-0028；AI 代书，非真人签字。交付：`test/performance/identity/benchmark_test.go`（新增）、`docs/benchmarks/identity.md`（新增）、证据包 `.spec/wip/identity-auth/evidence-s2.2/`（`report.md`、`profile-adoption-r01.md`、`scope-waiver-r01.md`、`lead-disposition-r01.md`）。评审链：非作者 Reviewer（Godel）r01 `CHANGES REQUIRED`（F001 HIGH、F002 MEDIUM、F003 LOW、F004 INFO）→ 返工 → r02 `PASS WITH CONDITIONS`；r01 四项全 CLOSED，新增 1 MEDIUM（`IA-S2.2-R02-F002`：`unresolved age` 未测）、1 LOW（`IA-S2.2-R02-F001`：采用记录引用返工前 r00 数值）、3 INFO。处置：R02-F001 已由 Lead 修正采用记录 §2.3 为 r02 复核值；R02-F002 由范围豁免显式处置（**豁免而非修复**，真实 PG lease 竞争/retry/dead-letter 与 `unresolved age` 保持 NOT RUN）；F003～F005 为 INFO 留档、不返工。三类结论：① 合法 token/tuple `PASS（仅机械校验）`；② 目标环境可运行 `NOT ESTABLISHED`；③ 满足批准预算 `ADOPTED / NOT VERIFIED`（profile ID `identity-argon2id-local-20261103-r01`，4 vCPU / 512 MiB / `GOMAXPROCS=4` / `MaxConcurrency=4`；RPO≤24h、RTO≤4h、retention 30 天为测试暂定值）。**哈希时点提示**：上述对象哈希为关闭时点值，后续修改会使其漂移，历史记录不回写。三态更新为 74 completed / 2 active / 9 todo。Git：未 add/commit/push，`main`/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变。

- 2026-11-05 article-management-S2.4 范围决策关闭变更记录：项目负责人明确确认并授权「经决策不验证：像 ADR-0024 那样落一条 ADR，按变更后合同关闭 S2.4」。新增永久决定 **ADR-0033**（`.add/project/knowledge/decisions/content.md`，`内容` 领域）并同步 `decisions/README.md` 全局索引、领域文件表、场景查询表与产出归属说明。S2.4 由 `[-]` 改为 `[x]`（完成@2026-11-05），`阻塞注记` 改为 `范围变更与关闭注记`，`本任务关闭条件` 改为变更后五条可核条件；三态由 74/3/9 更新为 75/2/9，active 精确为 `article-management-S1.5`、`article-management-S8.2`。ADR-0033 只改变 S2.4 验证范围，不改变 S2.2/S2.3 已关闭结论、AM-Q2 参数、产品合同与生产代码；R09/R10 证据与 NRU 不改写；不宣称 SLO、目标容量、P95 或 production readiness。Lead 直接执行并现场核证（本 Host 无委派能力，无 Backend-Dev / Reviewer / Task-Planner Role 产出）。
- 2026-11-05 article-management-S8.2 / C18-COV-F002 范围决策关闭变更记录：项目负责人确认「不需要这项门禁」（测试项目、经决策不验证）。新增永久决定 **ADR-0034**（`decisions/content.md`，`内容` 领域）并同步 `decisions/README.md` 四处索引；`C18-COV-F002` 由「待补 failed/expired/undetermined 三状态 + 真实 runtime/PG」转为未验证边界，依据是实测 admission `Verify` 不读取 `recovery_runs.outcome`（temporary=`internal_ready`、live=`public_ready`、改 outcome 无影响），三状态门禁若需强制应落在 coordinator `Resume`/start。S8.2 保持 `[-]`（F003 待复审、F004 待复核、真实部署 E2E/双系统未验证），S1.5 保持 `[-]`；三态 75/2/9、requires/DAG/写集不变；不改任何生产代码；Lead 直接执行，无 Backend-Dev / Reviewer / Task-Planner Role 产出。
- 2026-11-05 article-management-S8.2 / Case18 真实部署 E2E 与双系统范围决策关闭变更记录：项目负责人确认「经决策不验证」。新增永久决定 **ADR-0035**（`decisions/content.md`，`内容` 领域）并同步 `decisions/README.md` 四处索引；Case18 第 5 项（真实部署 E2E/双系统，`C18-COV-F007` LOW）转为未验证边界，不以本机模拟或单机证据冒充部署结论。不改变 `C18-COV-F003`/`C18-COV-F004` 复审链、S8.2 与 S1.5 的 `[-]` 状态、三态 75/2/9、requires/DAG/写集；不改生产代码或部署材料；Lead 直接执行，无 Backend-Dev / Reviewer / Task-Planner Role 产出。
- 2026-11-05 article-management-S8.2 关闭变更记录：Lead 按 `task-execution-workflow.md` §10.3 关闭本卡（`[-]`→`[x]`，完成@2026-11-05）。Case18 F003/F004 由非作者 Reviewer（Copernicus `01a1abdf-853e-7593-a6a5-330a1972dd43`，default 角色承载 add-code-review 合同）复审 CLOSED；F002/F007 按 ADR-0034/0035 范围决策关闭；Lead 现场核评审文件哈希/字节、复跑 `EXIT=0`、源码未变。保留边界：production Enter/Install closed、simulated journal/keyring、F004 仅 store 层、真实部署 E2E/双系统未验证。三态由 75/2/9 更新为 76/1/9，active 仅 `article-management-S1.5`；requires/DAG/写集不变；不改生产代码，无 Git 写。Lead 直接执行并现场核证。
## 全局约束与项目启动前置

### 正式项目启动前置：`project-bootstrap-B0.1`

`project-bootstrap-B0.1` 是所有代码与测试任务的硬前置。它是用户明确授权的项目启动期初始任务，不从 Story 派生；完成前 B1～B12 全部暂停。B0 只建立 Go、依赖、迁移与测试执行基线，不实现身份、文章或其他业务行为。

### 共享所有权

- `project-bootstrap-B0.1` 唯一拥有 Go module、最小可执行入口、迁移执行器、测试标签/环境基线、依赖审查记录及 Manifest 检查命令登记；后续 Story task 只消费。
- 共享平台、隐私、审计事实由 `identity-auth-S1.1` 唯一拥有；article 只依赖。
- ReplayStatus v1 消费/fixture 与恢复协调由 `identity-auth-S4.2` / `identity-auth-S1.5` 拥有，article producer 依赖同一 fixture，不复制 schema 状态。S4.2 仅拥有 provider-neutral Anchor 与 opaque key-retirement capability 协议/确定性 fake；生产 Anchor 由 `identity-auth-S4.5` 独立拥有，生产 key-retirement capability 闭环由 `identity-auth-S4.6` 独立拥有。
- F005 application composition 与项目内 metrics registry/exposition 由 `identity-auth-S3.2` 承接。`internal/security/argon2.go` / `argon2_test.go` 是已关闭 `identity-auth-S2.1` 的历史成果，`internal/platform/errors.go` / `errors_test.go` 是已关闭 `identity-auth-S1.1` 的历史成果；本轮只把这四个既有文件加入 S3.2 的后继、有界、串行修改/回归范围，不抹除历史完成事实，也不形成与已关闭任务或其他写者的并行所有权。`deploy/observability/identity-alerts.yaml`、`deploy/observability/identity-alerts.test.yaml`、`deploy/observability/identity-grafana-dashboard.json`、`deploy/observability/identity-alertmanager-routing.yaml` 与 `docs/runbooks/identity-observability.md` 由 `identity-auth-S3.5` 唯一拥有；I-OBS-DEP-01～06 的 Case/Run 台账和真实平台演练证据由 `identity-auth-S3.6` 唯一拥有。三层必须按 `S3.2 → S3.5 → S3.6` 串行，不得互相吞并；S3.6 通过前不得宣称 production observability readiness、真实部署/采集/告警已通过或需求整体完成。
- `cmd/han-server/main.go` 与 `cmd/han-server/main_test.go` 保留已关闭 `identity-auth-S3.2` 的历史完成证据，但 S3.2 不再占有或写入；两文件从 S3.2 关闭结果顺序移交给现已关闭的 `identity-auth-S4.1`，仅用于 Q1 production router/composition 接线与回归。该移交不重开 S3.2，也不得借 S3.2 继续修改。
- `internal/identity/handler.go` 已从关闭的 `identity-auth-S4.1` 顺序移交并由现已关闭的 `identity-auth-S7.1` 消费；`internal/article/command_service.go` 已从关闭的 `article-management-S1.3` 顺序移交并由现已关闭的 `article-management-S6.1` 消费。后续任务须按自身依赖和授权重新认领，不得回写已关闭卡状态或把顺序移交误作并行所有权。
- `migrations/0006_user_governance_indexes.sql` 与 `internal/identity/repository_integration_test.go` 的本次分页索引增量由现已关闭的 `identity-auth-S7.2` 独立交付；S7.1 五文件边界和已关闭 S1.2 的 `migrations/0002_identity.sql` 均未扩大或修改。F002 已由 S7.2 的独立复审正式关闭；任何后续写入仍须重新核任务、授权与所有权。
- password-reset durable 范围按 task DAG `identity-auth-S10.2 → identity-auth-S10.3 → {identity-auth-S10.1, identity-auth-S10.4}`；S10.1～S10.4 与 B6D/S11.1 的关闭证据保持有效。S10.1 classification repair 已在 `internal/mailer/mailer.go`、`internal/mailer/smtp.go`、`internal/mailer/smtp_test.go`、`internal/identity/password_reset.go`、`internal/identity/password_reset_test.go` 五文件内完成并关闭；`Mailer.Send(context.Context, Message) error` 签名与公共 REST/schema/错误码未变，既有唯一 coordinator 已消费稳定“明确未受理”分类并调用 `RecordRejectedDelivery`。S11.1 未重开，仅作为共享 coordinator 的真实 resend 消费者纳入回归；第二 coordinator/Store/Gate/operator/composition root/隐藏全局、新 migration、第六文件或字符串匹配错误分类均未引入。S10.5 仍须等待稳定设计、测试规划、文件充分性复核与 Lead 授权。
- canonical 用户删除编排由 `identity-auth-S9.1` 拥有；article 端 `OrphanArticlesCanonical` 由 `article-management-S6.1` 拥有。
- Article Deletion Journal、finalizer 与 replay evidence 由 article Story 4 唯一拥有；identity 只消费 DTO。
- Article production wiring 由 `article-management-S4.6` 唯一拥有：只在冻结的 13 文件内接唯一 production composition/router、Journal provider、Article 私有平台锁适配、三条 DELETE、后台 worker 与 lifecycle；不得新增 migration、依赖、第二 composition/router、修改/导出/复制 `internal/recovery` 私有锁原语或公共合同，或使用第 14 个文件。
- B7 剩余三条 active lane 的登记写集仍互斥：S9.1 五个 `internal/identity` canonical deletion 文件已按 r45 实施 R01 F001～F004（含 §4.2 新增 3.1 步同一 identity-owned RecoveryRunStore + maintenance/readiness Gate 双重证明，无第二 authority），当前 `implementation done pending non-author Code Review`，写集仍为原五文件、未扩为第六文件；article S4.2 已登记精确 19 文件，其中 `cmd/han-server/main.go` / `main_test.go` 必须与已关闭 S4.6 的唯一 composition owner 串行，`internal/recovery/replay_status_v1.go` 与 identity consumer 只读，Planning Review R02 已 PASS但仍无 Lead START，故继续暂停；article S4.3 三个 `test/integration/article` Journal/finalizer 文件按 ADR-0024（2026-10-27）以范围决策关闭：真卷韧性验证不纳入本版交付范围，三文件未创建、未执行，未验证边界如实登记。article S4.5 的一个 runbook、两个合成 recovery fixture 与一个专用 integration 测试文件已完成 R01 FAIL→原 owner 返工→R02/Lead Gate PASS并关闭，固定四文件独占窗口已释放；S5.3 固定三测试文件与 article S2.1 四文件也已由各自 R02/Lead Gate 关闭并释放独占窗口；S4.6 保持 CLOSED且不重开。`internal/identity/handler.go` 与 `internal/article/handler.go` 是不同路径。S3.3/S3.5 继续独占既有 browser 与 deploy/docs 写集；S4.2 若需第 20 文件、新 provider/协议/依赖、第二 source/reader/root/global/locator、公共合同或 architecture/ADR 变化即停止；S4.5 已关闭，不得无新授权重开或扩大其四文件合同；任何未知写者、跨卡共享文件或其他任务条目外文件也停止受影响 lane 并回 Task-Planner/Lead。
- 开发 task 自带局部测试和独立 Review；专项集成/安全/韧性/性能及部署后 E2E 另列，不阻塞开发 task 按自身条件关闭。
- 未决 Q2、容量、隐私部署值、RPO/RTO、retention 等只阻塞卡片明确标注的范围，不替用户填值；Q1 与 AM-Q1 已采用用户确认值，不再作为当前决策阻塞。

## 任务统计

- 总数：86
- 三态：86 completed / 0 active / 0 todo
- project-bootstrap：1
- identity-auth：58
- article-management：27
- 类型：项目启动 1；决策文档 4；开发 59；部署材料 1；真实部署验证 1；运行输入准备 3；专项契约测试 2；专项集成测试 4；专项性能测试 3；专项安全测试 1；E2E 2；专项韧性测试 2；专项安全/集成测试 1；专项契约/韧性测试 1；专项集成/安全测试 1

## project-bootstrap（B0，非 Story）

- [x] project-bootstrap-B0.1 建立 Go 服务、依赖、迁移与测试执行基线 完成@2026-10-21

- **目标**：建立不含业务行为的 Go 1.27 项目启动基线，使后续 47 个 Story task 可在同一 module、数据库迁移合同与测试环境约定上实施。
- **类型**：项目启动
- **实施责任**：Backend-Dev；Agent ID `01a15c1c-35d3-7d31-8ce4-86153871411e`；开始@2026-10-21；Lead 关闭@2026-10-21
- **文件**：
  - `go.mod（新增）`
  - `go.sum（新增）`
  - `cmd/han-server/main.go（新增）`
  - `internal/migration/runner.go（新增）`
  - `internal/migration/runner_test.go（新增）`
  - `internal/migration/runner_integration_test.go（新增）`
  - `internal/migration/testdata/0001_create_probe.sql（新增）`
  - `internal/migration/testdata/0002_fail_probe.sql（新增）`
  - `migrations/README.md（新增）`
  - `internal/testenv/env.go（新增）`
  - `internal/testenv/env_test.go（新增）`
  - `docs/testing.md（新增）`
  - `docs/dependencies.md（新增）`
  - `add.manifest.yaml（修改；实施时登记真实检查命令）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go version`
  - `go list -m`
  - `go list -m -f "{{.Path}} {{.Version}}" github.com/jackc/pgx/v5 golang.org/x/crypto`
  - `go mod verify`
  - `gofmt -l .`
  - `go vet ./...`
  - `go build ./cmd/han-server`
  - `go test ./internal/migration ./internal/testenv`
  - `go test -tags=contract ./...`
  - `go test -tags=integration ./internal/migration -run "TestRunner"`
  - `go test -race ./...`
  - `go test ./...`
  - 当前状态：2026-10-21 Lead 在固定工作树执行全部上述验收命令并通过；Go 1.27.0、pgx/v5 v5.11.0、x/crypto v0.58.0 已锁定并验证，PostgreSQL 18.6 integration I-01～I-06 通过，随机 schema 与 advisory lock 清理为 0，独立 Review PASS。
- **依赖**：无（B0 根任务，不依赖自身）
- **并行**：串行；B0 完成并经独立 Review 后才可启动 B1～B12。
- **阻塞注记**：已解除；依赖版本、审查记录和 PostgreSQL 18 真实集成证据见实际文件及 `project-bootstrap-test-plan-b0.1-r05`。
- **本任务关闭条件**：`go.mod` 声明 Go 1.27、module `han`，`go.sum` 锁定并验证已批准的 pgx/v5 与 x/crypto，`docs/dependencies.md` 记录实际版本及审查证据；最小 `han-server` 可构建且不含业务路由/规则；迁移执行器只承担 `migrations/NNNN_name.sql` 唯一顺序、`schema_migrations`、单迁移事务、PostgreSQL advisory lock 和失败停止，单元测试及 PostgreSQL 18 集成测试覆盖顺序、重复版本、提交记录、回滚与并发锁；`contract`、`integration`、`browser`、`e2e` 标签在 `docs/testing.md`/`internal/testenv` 形成单一约定：`contract` 不依赖外部服务，`integration` 统一读取 `HAN_TEST_POSTGRES_DSN`，`browser`/`e2e` 统一读取 `HAN_TEST_BASE_URL`，浏览器专项读取 `HAN_TEST_BROWSER_BIN`，带标签命令缺必需变量时明确失败而不静默通过；Manifest 的 `check_commands` 登记现场可真实运行的 Go 检查命令，不把不可用工具或未运行验证登记为通过；独立 Review 通过，Lead 核固定工作树 status/diff/所有权后关闭。
- **剩余独立验证**：测试计划 I-07（真实服务启动失败时 listener 不建立）尚未实施、未验证；它不冒充通过，也不阻塞本开发任务按自身关闭条件关闭。
- **后续验证关联**：B1～B12；原 46 个 identity-auth/article-management task。
- **设计/测试来源**：architecture r13 §4.1/§4.9/§7.2/§8.1～8.2/§11；ADR-0009/0012/0021；go-service Profile。

## identity-auth（11 Story）

### Story S10/S11 durable password-reset R 对账（architecture r27 / design r44 / stories r16 / test-plan r20）

- **S10.5 当前依据（2026-10-28 技术关闭）**：原S10.5 / B6C-4已按r49/r23完成本开发验收，maintenance能力已建成；原Root/Domain及carrier R01→R02的跨七文件软件独立审查、作者U/PG/B和Main同版现场证据齐备，Lead已技术关闭。限定格式例外及未验边界见原卡；不是全项目/需求或人的accepted/checkpoint。旧F001 OPEN、PG NOT RUN、CODE HOLD等仅保留其历史时点；原S3.4/B12 C24/25部署仍NOT RUN。

#### Story-level 唯一四标签结论

| Story | 唯一四标签 | 当前设计、当前代码与原 task 证据 |
| --- | --- | --- |
| S10 重置用户密码 | 已建成 | 原S10.1～S10.4关闭证据保持；S10.5 maintenance七路径按r49/r23及最终Root/Domain/carrier独立审查完成软件闭环，Lead于2026-10-28技术关闭，当前证据见原卡。部署C24/25、生产签发/第二真实SID与人的接纳仍归原S3.4/B12，未据软件关闭宣称通过。 |
| S11 重发临时密码邮件 | 需改 | S11.1 的 purpose router、successor generation、predecessor/replaced、same-key replay 与 stale candidate 六文件成果保持关闭；普通 `password_reset` resend 复用 S10.1 唯一 coordinator 的 delivery classification 已纳入本次五文件回归并通过，S11.1 未重开、未复制分类逻辑。 |

- **Story-level 四标签分布**：已建成 1 / 待做 0 / 需改 1 / 废弃 0。S10、S11各且仅有一个整体标签；S10本开发闭环已核证，S11原标签/历史不在本轮重判，Story不是task单位。

#### 第二级能力/接缝 coverage/traceability 矩阵

- **矩阵性质**：以下 12 行只用于把两个 Story 下钻到设计、代码、task 与专项验证接缝；其已建成 9 / 待做 2 / 需改 0 / 废弃或不适用 1 是能力切片计数，不是 Story 四标签分布。

| 能力 / 接缝切片 | 四标签 | 当前代码、设计与原 task 证据 | 对账承接 |
| --- | --- | --- | --- |
| S10.1 首次 reset / 普通 resend 共享 coordinator 与 Mailer delivery classification | 已建成 | B7 G0 发现的统一 unknown / `RecordRejectedDelivery` 无调用路径缺口已在五文件内修复；R07 的 `IDENTITY-S10.1-REV-R07-F001` 由 R08（SHA256 `30F1C4D2546F6E32BA23FCBECAD22235D49574CE35E66A236A8CCA18712DDC20`）确认 CLOSED、0 新 finding、Code Review Gate PASS，Lead 同 revision Gate PASS。 | `identity-auth-S10.1` 已再次关闭；S11.1 只消费且未重开，S5.3 上游 classification repair Gate 已满足。 |
| durable schema、execution counter、operation/job、六字段 Idempotency 与 credential lineage/settlement/hold | 已建成 | r44/r20 继续复用 r40 的 history/current classifier、A/B/C、same-key/new-key、current projection drift/hold 与瞬时故障边界；固定八文件已完成，Gibbs R05 确认 F003/F004 CLOSED，Lead §10 核证 PASS。真实 PG18/multi-connection/migration 混跑仍属 S1.6。 | `identity-auth-S10.2` 已 task-level CLOSED，不新增 task 或文件。 |
| Login unresolved-operation Gate | 已建成 | S10.3 已按 design r42/test-plan r19 `I-PRD-11/15.CW-*` 完成唯一 non-nil Gate constructor、同一 production Store/Gate 注入、clear/closed/query/commit 错误映射与零旧授权入口；Code Review R02、Lead 定向/race/vet 与 Manifest 七项均 PASS。已关闭 S3.2/F005 继续只提供回归基线。 | `identity-auth-S10.3` 已 task-level CLOSED。 |
| PasswordChange Gate 与 succeeded lineage | 已建成 | S10.3 已按 design r42/test-plan r19 `I-PRD-12/15.CW-*` 完成 PasswordChange constructor 强制非 nil Gate、共享 production 实例、失败零 credential 写入并保持 succeeded lineage；Code Review R02 与 Lead Gate PASS。已关闭 S4.1 不重开。 | `identity-auth-S10.3` 已 task-level CLOSED。 |
| Session resolve caller-owned transaction / commit barrier | 已建成 | S10.3 已按 design r42/test-plan r19 `I-PRD-13/15.CW-*` 完成 token 预读只定位候选 User，并在 caller-owned transaction 内按 User→Gate→Session/credential 重读→必要 revoke→commit→Principal；Code Review R02 与 Lead Gate PASS。 | `identity-auth-S10.3` 已 task-level CLOSED。 |
| password-reset 自动 worker、internal operator、connection permit 与双 worker production lifecycle/config | 已建成 | design r44/test-plan r20 固定 S10.4 顺序接管精确八文件，复用 S10.3 同一 Store/Gate，增加 internal `PasswordResetOperator`、自动 worker、五项无默认配置、唯一 permit owner及双 worker wiring/lifecycle；Article/PasswordReset `Start` 均无同步 error，运行错误只经 `Done/Err`。Tesla 已完成实施与返工，Pascal R03 确认全部既有 finding CLOSED、无新增 finding且 Code Review Gate PASS，Lead 同 revision Gate PASS。 | `identity-auth-S10.4` 已 task-level CLOSED，开始@2026-10-24、完成@2026-10-25。 |
| S10.5 互斥 maintenance mode 与真实维护入口 | 已建成 | r49/r23最终七源码与适用U/SDK-peer/真实PG/B已核证；Root F001～F004、carrier F001均CLOSED，C18真实worker/CLI重叠及C16 UTF8 JSONB 65536/65537边界SATISFIED。原非作者整体软件Review NO FINDINGS WITH LIMITS，Lead接受仅immutable证据格式限定例外后技术关闭，不冒所有Case或部署PASS。 | 原 `identity-auth-S10.5` / B6C-4本开发闭环；开始/完成@2026-10-28，状态只取原卡。七路径、两owner、254edges不变；C24/25部署仍原S3.4/B12 NOT RUN。 |
| 唯一 Gate production construction/wiring 与三消费者共享实例 | 已建成 | S10.3 已按 design r42/test-plan r19 `I-PRD-15.CW-01～05` 完成 production root 唯一 `*identity.PasswordResetStore` 构造与 Login/PasswordChange/Session 同一非 nil Gate 注入；r44/r20 要求 S10.4 operator/worker继续复用同一指针且禁止第二 authority。 | `identity-auth-S10.3` 已 task-level CLOSED；S10.4 不得重复拥有 Gate construction。 |
| S11.1 purpose router、password_reset successor generation、predecessor/replaced、same-key replay 与 stale candidate | 已建成 | design r44、stories r16 与 test-plan r20 的 I-PRD-01～03/06～10/12/19 已由六文件实现与回归覆盖；两条既定 public route 已在唯一 production router/composition 可达，R03 与 Lead 最终 Gate PASS。 | `identity-auth-S11.1` 已 task-level CLOSED，完成@2026-10-25；B6D CLOSED，六文件独占窗口已释放。 |
| S3.2/F005、S5.2、S8.1 已关闭共享能力 | 已建成 | design r44 继续复用唯一 production root、既有 Session/幂等持久化与账户状态能力；password-reset durable Gate 的唯一 Store construction/wiring 与三消费者共享注入已由 S10.3 的 13 文件关闭。S10.4 八文件只顺序接管普通 server config/operator/worker/runtime wiring；历史 task/Review 只作为各自关闭链证据。 | 不重开已关闭卡；S10.4 已由 Tesla 在精确八文件内完成并关闭。 |
| 公共契约、真实 PG18、local real-protocol SMTP、approved-provider/E2E、性能 | 待做（既有专项） | test-plan r20 继续映射 `identity-auth-S1.5`、`S1.6`、`S5.3`、`S3.4`、`S2.2`；其中 S5.3 已在固定三文件完成 loopback 本地合成 SMTP/TLS/AUTH、accepted/rejected/unknown 与凭据残余窗口闭环，其余仍为既有 todo/blocked task。 | 复用原卡，不复制、不并入 B6C 开发关闭条件；S5.3 的局部关闭不外推真实 provider、外部 SMTP、真实 PostgreSQL、部署/E2E、容量或 production readiness。 |
| 旧 claim-first + 请求线程有限 cleanup、optional/no-op/allow-all/deny-all Gate、第二 authority 或伪 maintenance 入口 | 废弃/不适用 | design r44 与 ADR-0023 明确禁止缺 Gate 可授权入口、旧 constructor、可选/旁路 Gate、第二 Store/Gate/root/global，以及在 S10.5 稳定设计前宣称 production maintenance 命令或 runbook；有限 cleanup 也不能证明 commit-unknown 后 durable fail-closed。 | 禁止延续或以旧测试 PASS 关闭 R04-F002/S10.1。 |

- **R 对账决定**：方案 B 的既有重开范围保持不变且现已闭环：只由 `identity-auth-S10.1` 承接五文件 classification repair，不扩展 S5.3 production 写集、不重开 S11.1、不新增 task或改变 requires；历史 Gate `S10.1 classification repair → S5.3 G0 re-entry` 已满足，S5.3 后续三文件返工与关闭也已完成。当前任务总数/三态为 57 / 30 completed / 5 active / 22 todo。
- **Gate**：R08（SHA256 `30F1C4D2546F6E32BA23FCBECAD22235D49574CE35E66A236A8CCA18712DDC20`）与 Lead Gate 已关闭 S10.1 classification repair。S5.3 R01（SHA256 `78835EC544357CFB6BD970DF5F59EC5E20C9FD3CFCFA00AC3495B82EE0E7049B`）曾为 `CHANGES REQUIRED`；R02（SHA256 `57FC85996662492B35AC78F6BCB25F262EFBA35C0DD2B053822765DEB5453ABA`）确认 F001～F004 CLOSED、0 新 HIGH/MEDIUM finding、Code Review Gate PASS，Lead 同 revision Gate PASS并关闭 S5.3。active 精确为 S3.3/S3.5/S9.1/article S4.2/S4.3，S10.5 不启动。

- **2026-10-25 integrated Review R01 变更记录**：保留同日 S10.1 task-level 关闭历史证据，不改写旧快照；本次仅把现役 S10.1 重开、收窄两文件返工写集并同步 B6C-3/B6D/S10.5 Gate。未修改代码，未运行 Go/PG/SMTP/E2E，未读取 `HAN_TEST_POSTGRES_DSN`，未 stage/commit/push。
- **2026-10-25 integrated Review R03 / B6D 启动变更记录**：固定两文件修复与测试哈希、Copernicus R03 和 Lead 最终 Gate 全部 PASS后，S10.1 再次关闭，B6C-3 由 Lead 接纳；随后仅启动既有 S11.1，S10.5 与其他任务保持未启动。本次 Task-Planner 只同步 Tasks/Plan，未运行 Go/PG/SMTP/部署/E2E，未读取 `HAN_TEST_POSTGRES_DSN`，未 stage/commit/push。
- **2026-10-25 B6D/S11.1 G0 R 对账变更记录**：Curie 尚未修改文件、未进入 RED。现有 production router 只匹配 `.../password-reset`，无法在原三文件内注册 design r44 已固定的两条 resend route或复用现有唯一 PasswordReset coordinator 与 `AdministratorCreateCoordinator`。S11.1 保持 `[-]` 与开始日期，implementation paused / G0 blocked pending non-author Planning Review；唯一写集最小扩为五文件，任务总数、requires、DAG、三态与 active 集合不变。S10.5 不启动；未修改代码、设计、test-plan 或 Review 报告，未运行 Go/PG/SMTP/部署/E2E，未 stage/commit/push。
- **2026-10-25 B7 启动 R 对账变更记录**：逐卡核对六张候选的 direct requires 均为 CLOSED，设计/test-plan/既有 task 合同足以启动，精确写集彼此及与 active S3.3/S3.5 零交叉；六卡从 `[ ]` 更新为 `[-]`、登记开始@2026-10-25，仅完成认领准备，实施 Agent 与非作者 Reviewer待 Lead 分配。AM-Q2 只阻塞容量/P95结论，不阻塞 S2.1 固定正确性合同；S4.3 缺真实独立持久卷时不得关闭；S4.5 不填未确认 retention/RPO/RTO且不得宣称灾备就绪。三态更新为 57 / 27 completed / 8 active / 22 todo，S10.5 与其他任务未启动；本轮未运行 Go/PG/SMTP/Browser/部署/E2E，未读取或使用 `HAN_TEST_POSTGRES_DSN`，未 stage/commit/push。
- **2026-10-25 B7/S5.3 G0 最小 R 对账变更记录**：Kant `01a1716c-b694-71c3-8d72-2a4fc1ab4eb3` 在 S5.3 G0 停止且零写入。Lead 现场确认所有 Mailer error 统一进入 unknown、`RecordRejectedDelivery` 无调用路径，故 S5.3 无法在三测试文件内诚实 GREEN。采用 B，仅重开 S10.1 并把当前返工写集固定为五文件；不新增 task、不重开 S11.1、不改 requires/公共 REST/schema/migration/ADR。S5.3 保持 active 但 implementation paused，其他 B7 lane 不受影响；三态为 57 / 26 completed / 9 active / 22 todo。本轮未改代码/设计/test-plan/Review/Manifest，未运行 Go/SMTP/E2E，未 stage/commit/push/切分支/清理，`han-server.exe` 必须保留。
- **2026-10-25 B7/S5.3 关闭变更记录**：原 G0 classification 阻断与 S10.1 repair 历史保留；Kant 在固定三文件完成返工。R01（SHA256 `78835EC544357CFB6BD970DF5F59EC5E20C9FD3CFCFA00AC3495B82EE0E7049B`）的 F001 HIGH/F002～F004 MEDIUM 曾 OPEN，R02（SHA256 `57FC85996662492B35AC78F6BCB25F262EFBA35C0DD2B053822765DEB5453ABA`）确认四项全部 CLOSED、0 新 HIGH/MEDIUM、Code Review Gate PASS；Lead 同 revision Gate PASS并关闭任务。三态更新为 57 / 29 completed / 6 active / 22 todo，active 精确移除 S5.3；未连接真实 PostgreSQL或外部 SMTP，未部署/E2E，不声明真实 provider、容量或 production readiness。
- **2026-10-26 S9.1 设计增量最小 R 对账变更记录**（Lead 接管执行；Host 无委派能力，无 Task-Planner Role 产出；依据用户 2026-10-26「对待确认或决策项采用建议值并继续」授权）：消费 identity design r45（revision `identity-auth-design-ir01-r45`，SHA256 `CBE0F64ADB93E1AFD4340D2D3390ACE42EE85F557DDCBA5AEBCF2AA20CE404F1`）与其 Design Review `.spec/wip/identity-auth/review-design-r45-s9.1-r01.md`（SHA256 `5905C0D1CF1EF66FAECDF559AC01DD5795A1A2BAA78AAF7874232A2FB13D8D74`，Design Review Gate PASS，依据用户明确接受作者自审口径）与上游 `identity-auth-S9.1` Code Review R01（SHA256 `813B8B947935101CA5C313DB77432B984994B3F71FED21360C9B3A67D325167B`，`CHANGES REQUIRED`，F001～F004 均 OPEN）。逐条核对：r45 §3.3.10 覆盖 F001（标识墓碑全行加锁 + 路径 A/B 值级绑定 + lifecycle + phone 保守永久 `reserved`）与 F003（trace 单一 ctx 权威、显式字段降为一致性断言），§4.2 新增 3.1 步覆盖 F002（同一 identity-owned RecoveryRunStore + maintenance/readiness Gate 双重证明，无第二 authority/Gate/global/可伪造 capability），F004 为行为保持拆分且限 50 行；四项均已达到可实施程度。写集仍为原五文件（`internal/identity/deletion.go`、`article_port.go`、`deletion_test.go`、`service.go`、`handler.go`），未扩为第六文件；与 active lane S3.3（browser 写集）、S3.5（deploy/docs 写集）、article S4.2（19 文件，含 `cmd/han-server` 两文件）、article S4.3（`test/integration/article` 三文件）写集零交叉。S9.1 状态由 `blocked-for-design-revision` 更新为 `implementation paused pending Lead START`，保持 `[-]`、开始@2026-10-25，不新增/关闭/重开任何 task，不改 requires/DAG/批次范围/公共合同。用户已授权采用 r45 §6 残余限制建议值：保持现状（路径 A 为正常重放权威路径，路径 B 对 phone 仅做存在性/未释放校验），不新增 migration、不扩张写集，「新增 retained_phone 列」登记为需单独授权的后续候选。R01 F001～F004 在代码侧仍 OPEN，设计 PASS 不关闭实现 finding、不构成实施 START。任务总数/三态保持 57 / 30 completed / 5 active / 22 todo，active 精确为 S3.3/S3.5/S9.1/article S4.2/S4.3。本轮未修改代码、schema/migration、design/ADR/boundary/architecture/test-plan/Review、Manifest 或 checkpoint；未运行 Go/PG/SMTP/Browser/部署/E2E，未读取或使用 `HAN_TEST_POSTGRES_DSN`，未 stage/commit/push/切分支/清理，`han-server.exe` 保留。
- **2026-10-26 S9.1 实施变更记录**（Lead 接管实施；Host 无委派能力，无 Backend-Dev Role 产出；依据用户 2026-10-26「确认授权；对待确认或决策项采用建议值；继续推进」）：在原五文件边界内按 r45 实现 F001～F004——`deletion.go`：新增 `RecoveryDeleteCapability`（不可由 HTTP/普通内部调用构造）+ `RecoveryRunLocker`/`RecoveryOpenGate` 依赖与 `authorizeCanonicalDeleteRecovery` 双重证明（同一 identity-owned RecoveryRunStore 只读锁定 + maintenance/readiness Gate；`outcome` 必须为 `running`、`backup_ref` 逐字节一致；缺失/不一致/Gate 非 open 一律写前 fail closed）；新增 `LockIdentifiers` + `validateCanonicalDeleteIdentifierTombstones`（路径 A/B 逐 kind 存在性/值/lifecycle 校验，孤儿行与值漂移 fail closed）；`canonicalDeleteTrace` 使 ctx 成为唯一 trace 权威、显式字段降为一致性断言；`canonicalDeleteInTransaction` 拆为 `prepareCanonicalDelete`/`applyCanonicalDeleteIdentity`/`appendCanonicalDeleteAudits`/`completeCanonicalDeleteRecoveryReview`，主函数 43 行满足 profile ≤50。`RecoveryRunStore.LockRecoveryRunForDeletion` 定义在 `deletion.go`（保持单一 authority、不扩写集）。`deletion_test.go` 补齐 fake 与 F001/F002/F003 失败注入回归。验证：`go test ./internal/identity`、`go test -race ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle"`、`go build ./...`、`go vet ./internal/identity ./cmd/han-server`、`gofmt` 全部 PASS；`internal/article` 与 `cmd/han-server` 测试 PASS。最终 SHA256：`deletion.go`=`119D378C3508A790FFC6E7371C5AFC0D8CE100A33C2F47AFACD305F7683A2721`、`deletion_test.go`=`2F5975867E74EC6B6E8A0AD1DFD128382EC5123FB308A7924A5B6D4294ED6687`。S9.1 保持 `[-]`、开始@2026-10-25，状态更新为 `implementation done pending non-author Code Review`；F001～F004 代码侧仍 OPEN，须由非作者独立 Code Review 与 Lead Gate PASS 才可关闭（本 Host 无法产出非作者 Review）。任务总数/三态保持 57 / 30 completed / 5 active / 22 todo，active 精确为 S3.3/S3.5/S9.1/article S4.2/S4.3。本轮未修改 design/ADR/boundary/architecture/schema/migration/公共合同/Manifest/checkpoint，未运行真实 PostgreSQL/SMTP/Browser/部署/E2E，未读取或使用 `HAN_TEST_POSTGRES_DSN`，未 stage/commit/push/切分支/清理，`han-server.exe` 保留。
- **2026-10-27 S9.1 / article S4.2 作者自审 Code Review 关闭变更记录**（Lead 接管执行；Host 无委派能力，无 Reviewer / Backend-Dev Role 产出；依据用户 2026-10-26 明确授权「由作者自审 Code Review（本 Host 可以自审）」与「对待确认或决策项采用建议值并继续推进」）：（1）`identity-auth-S9.1`：在原五文件边界内实现的基础上，由 Lead 以作者自审口径输出 `.spec/wip/identity-auth/review-code-s9.1-r02.md`（SHA256 `1F1D926EFFE92F4BB0DAEC86FA3037071BF3504CEAB6AE3B623391E223E18E23`），逐项复核 R01 F001（phone identifier 逐 kind 存在性/值/lifecycle 前置校验 + 零行/孤儿行 fail closed）、F002（同一 identity-owned RecoveryRunStore 只读锁定 + maintenance/readiness Gate 双重证明，capability 不可伪造、Outcome 必须 running、backup_ref 逐字节一致）、F003（ctx 单一 trace 权威、显式字段降为一致性断言）、F004（事务拆分至 ≤50 行）均 CLOSED，0 新 HIGH/MEDIUM，Code Review Gate PASS；固定五文件哈希 `deletion.go`=`119D378C…A2721`、`deletion_test.go`=`2F597586…D6687`，`article_port.go`/`service.go`/`handler.go` 与 R01 值一致未改；`go test ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle"` 普通与 race PASS，`go build/vet/gofmt` PASS；Lead 现场核 branch/HEAD/staged/哈希/写集后关闭本卡（完成@2026-10-27）。（2）`article-management-S4.2`：由 Lead 以作者自审口径输出 `.spec/wip/article-management/review-code-s4.2-r02.md`（SHA256 `9D06B84370E196C4881479701EA0B96D6B991A8DD52FB300F964B6061320F3AA`），确认 R01 F001（唯一 `ProductionDeletionReplaySource` + `FileBackupInventoryReader` + 唯一 composition wiring）～F004（复用 `internal/recovery` authority，无第二字面量）CLOSED；复审初判 F005 仍缺 A-REC-05 真实 PostgreSQL 证据，遂在已登记的第 11 项 `deletion_replay_integration_test.go`（唯一写集内，未扩第 20 文件）追加 A-REC-05 用例——实例化真实 `FileBackupInventoryReader` + 唯一 source，`os.Rename` 原子替换 inventory，经真实 `PostgresReplayEvidenceStore.ReadCurrent` 断言 drift 后 `unknown/evidence_stale`、Replay 后新增唯一 evidence、current 原子切换、再次读取新 Ready；`go test -tags=integration ./internal/article -run '^TestDeletionReplayPostgres$' -count=1` → `ok 4.87s`（A-REC-04/05/06/07 全 PASS），`-race` → `ok 6.55s`，Manifest 七项全部 PASS；F005 转 CLOSED，0 新 HIGH/MEDIUM，Code Review Gate PASS；最终 `deletion_replay_integration_test.go`=`F132361840952A2B4111FDC5289807DB993737F72D536C86D01423C21C8EFA0B`，`repository_integration_test.go` 对账更正为 `3A15B7635D97D66AB4DE5CE69155C37E414F75BC3DBB0AD4F14D869019A0DCF4`（先前 `EE9A68CC…269355` 为滞后值），其余 17 文件与返工后固定值一致；Lead 现场核 branch/HEAD/staged/哈希/写集后关闭本卡（完成@2026-10-27）。三态由 57 / 30 completed / 5 active / 22 todo 更新为 57 / 32 completed / 3 active / 22 todo，active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`article-management-S4.3`。两项 PASS 均以作者自审口径成立，弱于 add-code-review 的非作者独立复审，作为已知限制保留；不关闭/不重开其他卡，不启动 S10.5、S3.6、B8 与部署/E2E，不证明真实旧备份恢复、manifest 真实签发/挂载授权、retention/cleanup、灾备、Host、部署、E2E 或 production readiness。

### Story S04 R 对账（architecture r19 / identity design r13）

| Story 切片 | 标签 | 当前代码/设计证据与任务承接 |
| --- | --- | --- |
| S04 Recovery Journal 与 key retirement | 已建成（S4.2）；待做（S4.5/S4.6） | `identity-auth-S4.2` 十三文件已实现并完成 task-level 验证；Halley 按替代授权仅修改既有五文件完成返工，Pasteur 将 `IA-S4.2-CR-002-F001`～`F004` 全部 CLOSED，代码 Gate PASS、无新 finding；Lead 确认 PostgreSQL 18 integration/integration-race 与约定 Go 检查均 PASS。Pasteur 已完成 B3 独立批次整体 Review，开始/结束固定哈希一致、无阻断 finding，正式结论 `B3 G4-batch Gate: PASS`，B3 批次关闭。真实目标 POSIX 的 owner-only、atomic replace、file/directory fsync 与 flock 仍未实测。用户于 2026-10-22 授权启动且仅启动 B4 的三张既有任务；该授权不包含 `identity-auth-S4.5` 与 `identity-auth-S4.6`，二者分别承接生产 Anchor 与生产 key-retirement 闭环，均未认领、未启动。 |

- [x] identity-auth-S1.1 建立共享 REST、配置、隐私门禁与追加式审计底座 开始@2026-10-21 完成@2026-10-21

- **目标**：建立共享 REST、配置、隐私门禁与追加式审计底座。
- **类型**：开发（共享基础，identity-auth 所有）
- **实施责任**：Backend-Dev；Agent ID `01a15d88-1e39-7503-8237-8292667d7d1d`（Bernoulli）；开始@2026-10-21
- **独立复审**：Reviewer；Agent ID `01a15d96-93cd-7fc1-aa85-c743745c3236`（Godel）；`B1-REV-F001`～`F006` 已关闭，最终结论 PASS。
- **文件**：
  - `migrations/0001_shared.sql（新增）`
  - `internal/platform/config.go（新增）`
  - `internal/platform/errors.go（新增）`
  - `internal/platform/logging.go（新增）`
  - `internal/platform/http.go（新增）`
  - `internal/privacy/store.go（新增）`
  - `internal/audit/store.go（新增）`
  - `internal/platform/config_test.go（新增）`
  - `internal/platform/errors_test.go（新增）`
  - `internal/privacy/store_test.go（新增）`
  - `internal/audit/store_integration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/platform ./internal/privacy ./internal/audit`
  - `go test ./...`
  - 当前状态：上述局部命令及 Manifest 登记的 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...` 已由 Lead 在固定工作树复跑并通过。PostgreSQL 18.6 审计集成已有通过证据；最终 HTTP 返工未触及 migration/audit 文件，本次关闭时因进程未设置 `HAN_TEST_POSTGRES_DSN` 未重复执行该集成。
- **依赖**：requires `project-bootstrap-B0.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：ProcessingGate 缺值 fail closed、公共错误 envelope/trace 映射、AuditEvent 只追加与事务失败回滚均有失败证据和局部测试；独立 Review 通过；Lead 核范围后关闭。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S1.6、article-management-S1.2
- **设计/测试来源**：identity design §3.3.1/§3.4.2；identity test I-AUD/I-CON；ADR-0014/0016/0017

- [x] identity-auth-S1.2 建立身份账号、标识与隐私证据数据模型 开始@2026-10-21 完成@2026-10-21

- **目标**：建立身份账号、标识与隐私证据数据模型。
- **类型**：开发
- **B2 认领/责任**：Backend-Dev Agent `01a15dd1-4baa-7e60-ac6f-ad3d93002dec`（Averroes）实施；独立 Reviewer Agent `01a15deb-15d2-7662-b233-a56147d298c1`（Copernicus）完成复审，四个 finding 均已关闭，结论 PASS。
- **文件所有权**：Averroes 在 B2 内唯一拥有下列四个文件；另两张 B2 卡只读。发现未知写者或必须修改清单外文件时停止受影响写入并交 Lead。
- **文件**：
  - `migrations/0002_identity.sql（新增）`
  - `internal/identity/model.go（新增）`
  - `internal/identity/repository.go（新增）`
  - `internal/identity/repository_integration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity`
  - `go test -tags=integration ./internal/identity`
  - 当前状态：四个目标文件均已存在；Copernicus 对四个 finding 的复审结论为 closed/PASS，Lead 已取得真实 PostgreSQL integration PASS。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`
- **并行**：[P]；前置已满足，且与 `identity-auth-S1.4`、`identity-auth-S2.1` 文件零交叉，可在各自唯一 owner 下并行。
- **阻塞注记**：无；Lead 于 2026-10-21 核对固定工作树、文件所有权、真实 PostgreSQL 18.6 integration、独立复审与未提交交付约定后关闭本卡。
- **本任务关闭条件**：users、identity_identifiers、sessions、temporary_credentials、隐私/登录桶/恢复核对表的约束与索引落地；真实 PostgreSQL 约束测试通过；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S9.2
- **设计/测试来源**：identity design r09 §3.4.1/§3.4.2/§5.5；identity test I-REG-01/03～04/06、I-AUTH-04～07、I-ADM-01/05；architecture r15 §4.9/§6.4/§7.1～7.2/§11；ADR-0003/0013/0016/0017/0018/0021

- [x] identity-auth-S1.3 实现注册与当前隐私确认的原子创建 开始@2026-10-22 完成@2026-10-22

- **目标**：实现注册与当前隐私确认的原子创建。
- **类型**：开发
- **B3 认领/责任**：Backend-Dev Agent `01a16123-8194-70b0-85e8-0136d892a880`（Cicero）完成实施与返工；非作者 Reviewer Agent `01a1612d-6254-7070-89d2-cc9f14b8349f`（Erdos）关闭 `IA-S1.3-REV-001` 并复审 PASS。Lead 核固定四文件哈希无漂移，定向测试、全包、race、vet、build、gofmt 与 diff-check 均 PASS，并于 2026-10-22 关闭本任务。
- **文件所有权**：Cicero 对下列四个文件的 B3 identity lane 交接已完成；`identity-auth-S1.3 → identity-auth-S5.1` 串行 Gate 已解除，S5.1 后续只按其独立八文件写集恢复。发现未知写者或必须修改清单外文件时停止受影响写入并交 Lead。
- **文件**：
  - `internal/identity/handler.go（新增/修改）`
  - `internal/identity/service.go（新增/修改）`
  - `internal/identity/registration.go（新增）`
  - `internal/identity/registration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestRegister"`
  - `go test ./internal/identity`
  - 当前状态：Lead 与原 Reviewer 已完成上述实施、验证和复审闭环；固定对象无漂移，本任务已关闭。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`、`identity-auth-S1.4`、`identity-auth-S2.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先以失败用例证明旧 notice、缺门禁、并发标识冲突和审计失败不会留下部分账号；注册成功返回 201 且不建 Session；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S3.4
- **设计/测试来源**：identity Story S01；identity design §3.3.2；test I-REG-01～05

- [x] identity-auth-S1.4 准备隐私、同源、代理与 SMTP 处理门禁运行输入 [P] 开始@2026-10-21 完成@2026-10-21

- **目标**：准备隐私、同源、代理与 SMTP 处理门禁运行输入。
- **类型**：运行输入准备
- **B2 认领/责任**：Backend-Dev Agent `01a15dd1-4dc6-7250-bfcc-a0521445967c`（Newton）完成 reader/启动链四个 Go 文件及用户追加授权的示例、runbook 修订；非作者 Reviewer Agent `01a15deb-1823-7950-b793-9a67ffc3dd08`（Aquinas）分别 CLOSED `S1.4-REV2-F001`、`S1.4-REV2-F002`，无新增 finding。
- **文件所有权**：Newton 的实际 S1.4 交付范围为 `internal/platform/config.go`、`internal/platform/config_test.go`、`cmd/han-server/main.go`、`cmd/han-server/main_test.go`、`deploy/processing-gate.env.example`、`docs/runbooks/processing-gate.md`；后两文件由用户在实施中追加授权。当前不得据此继续修改六文件或扩大到其他实现，后续变更须重新核授权与所有权。
- **文件**：
  - `deploy/processing-gate.env.example（修改；用户追加授权）`
  - `docs/runbooks/processing-gate.md（修改；用户追加授权）`
  - `internal/platform/config.go（修改）`
  - `internal/platform/config_test.go（修改）`
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/platform -count=1`
  - `go test ./cmd/han-server -run "^TestRun_StartupGate$" -count=1`
  - `go test ./internal/platform ./cmd/han-server -count=1`
  - `go test -race ./internal/platform ./cmd/han-server`
  - `go vet ./internal/platform ./cmd/han-server`
  - `go build ./cmd/han-server`
  - 当前状态：六文件实现与修订已完成，Aquinas 已关闭 `S1.4-REV2-F001/F002` 且无新增 finding。Lead 定向 test、race、vet、临时 build、`go mod verify`、格式与定向 diff-check 均 PASS；在真实 User-scope 环境下，PostgreSQL 18.6（端口 5433）连接、Ping、迁移与 HTTP listener 均 PASS，验证进程已停止、临时 binary 已删除，PostgreSQL 16 已停止，仓库根目录不存在 `han-server.exe`。用户作为本次部署、隐私与安全责任人明确批准当前九个参考占位值作为 local-dev 正式登记值；可选 CA 路径留空并使用系统信任根。随后 Manifest 七项检查全部 PASS，未输出或持久化 DSN/密码。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无。用户于 2026-10-22 明确以本次部署、隐私与安全责任人身份批准当前参考占位值，批准范围严格限定为 local-dev 本机部署；该决定不代表生产环境、外部 SMTP 处理方或法律结论，后续环境必须重新登记和批准。
- **本任务关闭条件**：为示例中的 10 个非秘密键建立真实 reader；必填值缺失、空白或非法时在数据库连接与监听前 fail closed。`HAN_PUBLIC_ORIGIN` 仅接受 canonical HTTP(S) origin，无 userinfo/query/fragment，路径仅空或 `/`，返回无尾斜杠规范值；`HAN_TRUSTED_PROXY_CIDRS` 必填且至少含一个合法 CIDR，只保存规范 CIDR、不信任 Header；`HAN_SMTP_TLS_MODE` 仅接受 `implicit_tls`/`starttls_required`，host/server name 必填，`HAN_SMTP_CA_BUNDLE` 为空时使用系统信任根且仅作为可选受控 CA 路径。不得新增 SMTP 端口/用户名/密码键或实现 mailer、Origin middleware、代理 Header 消费、SMTP 连接/发送；启动链只加载并验证配置。上述实现及测试、Manifest 适用检查和非作者 Review 通过；实际地区、控制者、处理依据、canonical origin、可信代理、SMTP 处理方与部署值由责任人登记且未写入凭据。
- **后续验证关联**：identity-auth-S3.3、identity-auth-S5.3、article-management-S7.2
- **设计/测试来源**：identity design r09 §3.3.1/§3.4.2 安全与部署配置/§5.5/§10；identity test I-REG-05、I-AUTH-03/07、I-CRED-05、§7/§8/§11；architecture r15 §4.4/§4.6/§4.11/§8.2～8.3/§11/§12；ADR-0006/0008/0017；用户 2026-10-21 明确确认的四文件有界扩展

- [x] identity-auth-S1.5 验证身份 REST、内部端口与 ReplayStatus 消费契约 [P] 开始@2026-10-27 完成@2026-10-27

- **目标**：验证身份 REST、内部端口与 ReplayStatus 消费契约。
- **类型**：专项契约测试
- **文件**：
  - `test/contract/identity/rest_contract_test.go（新增）`
  - `test/contract/identity/internal_ports_test.go（新增）`
  - `test/contract/identity/replay_status_consumer_test.go（新增）`
  - `testdata/contracts/article-deletion-replay/ready-v1.json（新增）`
  - `testdata/contracts/article-deletion-replay/unavailable-v1.json（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=contract ./test/contract/identity`
  - 当前状态：本卡专用命令 `go test -tags=contract ./test/contract/identity -count=1`（19 个顶层测试）与 `-race -tags=contract ./test/contract/...` 均 PASS；Manifest 七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）全部 PASS。全部文件带 `//go:build contract`，无外部依赖。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.1`、`identity-auth-S4.2`、`identity-auth-S5.1`、`identity-auth-S9.1`、`article-management-S4.2`、`article-management-S6.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：11 US 公共 REST、S1–S16 端口、Ready20/Unavailable6 严格消费者与固定 digest 通过；额外/缺失/null/错类型先红后绿；另按 r40/r18 验证 password-reset atomic/dispatch/Gate/worker/command 内部端口、三消费者、history/current classifier 与公共 REST/error/权限兼容；独立 Review 通过。
- **后续验证关联**：identity-auth-S3.4、article-management-S1.5
- **设计/测试来源**：identity test I-CON-01～03/I-REC-06～08；test-plan r18 I-PRD-01/05/11～14/19；identity design r40 §3.3.11/12、§9.4；ADR-0020/0023
- **独立复审与 Lead Gate**：本 Host 无委派能力，无独立 Reviewer Role 产出；经用户 2026-10-27 授权以**作者自审**口径执行 Code Review，产出 `.spec/wip/identity-auth/review-code-s1.5-r01.md`（SHA256 `6CF841EC84CDF57AF6B19637B7B56E4E2AC5716B85854C4715AB76F7DA9F421A`），`Code Review Gate: PASS（作者自审口径）`，无 HIGH/MEDIUM finding。Lead 在同 revision 核五文件哈希、专用命令、race、Manifest 七项与写集范围。
- **关闭证据与限制**：最终 SHA256 为 `rest_contract_test.go`=`86133EF48C35F704628100AE72C865D8526142791C3F7AE10FF1725230C994A5`、`internal_ports_test.go`=`F749E7185D660CBFEC481CB5C83B30BA730003383354F622C6128123F12BBB1D`、`replay_status_consumer_test.go`=`9E9495C24D3212ADB34AEC199D90650CF99FA1E59C035CEA6ABB7DEC32C49752`、`ready-v1.json`=`586A0B8F76ADE8B720E44094A0D13163C827D231AC9CBC786C6D4AF92E90F0F2`、`unavailable-v1.json`=`929193CD06BA566402F3F3C1D6321E57BDA95D0270DED666B5C951D97B21AC1B`。关闭只覆盖契约测试五文件与本地/`contract` 标签验证；真实 PostgreSQL 多连接/commit-ack-loss/迁移混跑（I-PRD-04/05/08～17 的真实 PG 实例）由 `identity-auth-S1.6` 承接，真实 router 组装由既有已关闭 router 证据承接；部署、浏览器/E2E、容量/P95 与 production readiness 均未验证。作者自审弱于非作者独立复审，作为已知限制保留。

- [x] identity-auth-S1.6 验证身份 PostgreSQL 事务、锁与并发矩阵 [P] 开始@2026-10-27 完成@2026-10-27

- **目标**：验证身份 PostgreSQL 事务、锁与并发矩阵。
- **类型**：专项集成测试
- **文件**：
  - `test/integration/identity/postgres_test.go（新增）`
  - `test/integration/identity/fixtures.sql（新增）`
  - `test/integration/identity/faults_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/identity`
  - 当前状态：专用命令 `go test -tags=integration ./test/integration/identity -count=1`（真实 PG18.6）与 `-race` 均 PASS；Manifest 固定七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）已按序全部 PASS。二者互不替代。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.3`、`identity-auth-S6.1`、`identity-auth-S9.1`、`identity-auth-S10.1`、`identity-auth-S11.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：隔离 PostgreSQL 18 上唯一性、bootstrap、会话撤销、幂等、审计失败、治理竞争与删除回滚矩阵通过；另覆盖 `0008` preflight/升级、六字段 FK/UNIQUE/CHECK/index、五对象原子性、User-first 锁序、lease/多 worker、commit-ack-loss、混跑与旧 binary 回退阻断；无真实 PostgreSQL 时保持未关闭；独立 Review 通过。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.4
- **设计/测试来源**：identity test 身份 PostgreSQL 事务与并发专项；I-REG/I-ADM/I-CRED/I-GOV/I-DEL；test-plan r18 I-PRD-03～05/08～17；identity design r40 migration/锁序/混跑合同

- **关闭证据与限制**：本卡已于 2026-10-27 关闭（完成@2026-10-27）。登记三文件全部新增并在真实 PostgreSQL 18.6（`127.0.0.1:5433`）通过：`test/integration/identity/postgres_test.go` `C3A7F455C6AD40DC43557F578286CD53B1EB48A018DFAC699A43F152C0B9D1CB`（含 harness 与 12 组 S1.6 矩阵）、`fixtures.sql` `F6760020DE75F4E43FD758298FA7FFA2FA1E97B0ECA39543B28BB1469B9883BF`、`faults_test.go` `900F5CBE83F80DAE2BAEDEDF108DE94283F364FD0AFAD2372D0A4DC454DB1F63`。覆盖唯一性/FK/CHECK、bootstrap 与成功墓碑、会话撤销全量、幂等、审计失败零副作用、治理竞争单写者（`55P03`）、规范化删除回滚/成功、会话解析拒绝已撤销/停用用户、`0008` preflight（`55000`）、五对象原子性与 deferred 计数器缺口、lease 单赢家与过期接管、绑定不可变与 generation 不可回退（旧 binary 回退阻断）。**写集越界处置**：实现期一度产出第 4 文件 `matrix_test.go`，已无损并入 `postgres_test.go` 并删除，最终写集严格为登记三文件。**本卡触发既有 P0 缺陷** `ISSUE-identity-auth-0001`：`internal/identity/password_reset_store.go` 的 `passwordResetGateSQL`/`passwordResetResendLineageStatusSQL` 以 PostgreSQL 保留字 `current_user` 作表别名，真实 PG 恒报 `42601`，导致 `session.Store.ResolvePrincipal` 全部已认证请求失败；历史 S10.2/S10.3 仅用 stub+`strings.Contains` 验证故长期隐藏。经用户 2026-10-27 授权按最小修复处置（别名 → `current_account`，4 文件：`password_reset_store.go` `D1CB46D5BF038FCB8A7908FF06BA5FDAD6CB31903DCC66D9DBBF286D54D1FB62`、`password_reset_store_test.go` `CEE5617805058F9204FF5F45DC0F03B90A006C1CC987CF2C52A3BDDFD69BC777`、`temporary_credential_resend_test.go` `592D00E1CC28FE81B3E2362A4461654D60C1142A5B99A9DBC13D4DD252CD5465`、`smtp_credentials_test.go` `C9AED4062DB032DAE67A9F0AA683FC46184E91FBEA7386E6D68D473CE0F67EC1`）；`password_reset_store.go` 哈希由 `6C013640...3315F` 漂移，属已关闭 S10.3 固定哈希的显式漂移，已登记。**独立复审**：本 Host 无委派能力，无独立 Reviewer Role 产出；经用户 2026-10-27 授权以**作者自审**口径执行，产出 `.spec/wip/identity-auth/review-code-s1.6-r01.md`（SHA256 `EFBDE5D35FD198888CB4CD40624C5947C1FE9E0486A4E112847BF9D62D40C485`），`Code Review Gate: PASS（作者自审口径）`，本卡三文件 0 CRITICAL/HIGH/MEDIUM/LOW finding；作者自审弱于非作者独立复审，作为已知限制保留。**未验证边界**：真实 SMTP provider/外部 SMTP、部署/E2E/浏览器、容量/P95 与 production readiness 均未验证；`FOR UPDATE` 并发经 `lock_timeout` 确定性化，非压力测试。Git：暂存区为空，未 add/commit/push，`main`/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变，`han-server.exe` 保留。
- **后续验证关联**（关闭后）：identity-auth-S2.2、identity-auth-S3.4；本卡发现并修复的 P0 缺陷为 `ISSUE-identity-auth-0001`。
- [x] identity-auth-S2.1 实现密码策略与 Argon2id 凭据哈希 [P] 开始@2026-10-21 完成@2026-10-21

- **目标**：实现密码策略与 Argon2id 凭据哈希。
- **类型**：开发
- **B2 认领/责任**：Backend-Dev Agent `01a15dd1-50c2-75c2-b39b-7ca186df0a23`（Planck）实施；非作者 Reviewer Agent `01a15e01-f325-70b0-9ced-56fb6ca98a08`（Faraday）完成两轮独立 Review，结论均为 PASS，无 CRITICAL/HIGH/MEDIUM/LOW finding，七文件开始/结束哈希无漂移；Lead 于 2026-10-21 完成关闭复核。
- **文件所有权**：Planck 在 B2 内唯一拥有四个 security 文件，并仅获额外授权对 `go.sum`、`docs/dependencies.md` 写入 `golang.org/x/sys v0.49.0` 所需的最小校验和/依赖记录，以及仅在 `go.mod` 现有 indirect require 块增加 `golang.org/x/sys v0.49.0 // indirect` 一行；不得借此改动文件内其他内容。另两张 B2 卡只读。
- **文件**：
  - `internal/security/password.go（新增）`
  - `internal/security/password_test.go（新增）`
  - `internal/security/argon2.go（新增）`
  - `internal/security/argon2_test.go（新增）`
  - `go.sum（修改；仅补 golang.org/x/sys v0.49.0 校验和）`
  - `docs/dependencies.md（修改；仅补 golang.org/x/sys v0.49.0 依赖记录）`
  - `go.mod（修改；仅在现有 indirect require 块增加 golang.org/x/sys v0.49.0 // indirect）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `GOFLAGS=-mod=readonly go test -count=1 ./internal/security`
  - `GOFLAGS=-mod=readonly go vet ./internal/security`
  - `GOFLAGS=-mod=readonly go build ./internal/security`
  - `GOFLAGS=-mod=readonly go test -count=1 -race ./internal/security`
  - `GOFLAGS=-mod=readonly go mod verify`
  - `go test ./internal/security`
  - `go test ./...`
  - 当前状态：历史失败证据为 Go 1.27 下 `GOFLAGS=-mod=readonly go test ./internal/security` 输出 `updates to go.mod needed`；用户随后仅授权上述唯一 `go.mod` 行，未授权 `go mod tidy`、其他 `go.mod` 变化或依赖升级。Faraday 独立执行 readonly test/vet/build/race、`go mod verify`、coverage（88.5%）、gofmt 与定向 diff-check 均 PASS；Lead 再执行上述五条 readonly 命令、gofmt 与定向 diff-check 均 PASS，七文件哈希仍等于固定快照。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：本卡关闭证据已满足，无本卡阻塞。生产 Argon2id 内存、迭代、并行、容量与性能仍由 `identity-auth-S2.2` 承接；当前无生产消费者，后续接线须独立评审，均不阻塞 S2.1 自身关闭。Faraday 关于 `docs/dependencies.md:10` 历史阶段措辞的非阻断建议留给对应 owner，未在本轮修改该文件。任何超出获准一行的 `go.mod` 改动、`go mod tidy` 或依赖升级仍禁止。
- **本任务关闭条件**：先以失败用例覆盖长度/字母/数字、参数缺失、资源错误和旧参数；仅保存自描述 Argon2id 编码且无明文；独立 Review 通过。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.2
- **设计/测试来源**：identity Story S02；identity design r09 §3.3.1/§3.3.3/§3.4.1/§5.5；identity test I-REG-06/I-AUTH-06；architecture r15 §2.3/§4.1/§4.4/§6.3/§7.2/§11/§12；ADR-0009/0012

- [x] identity-auth-S2.2 建立 identity 容量与性能基线 [P] 开始@2026-11-03 完成@2026-11-03

- **目标**：在实际目标部署环境（明确 CPU 型号/配额、内存限制、Go 版本与运行参数）建立 identity 容量与性能基线，并独立完成 Argon2 target-profile readiness；候选 tuple 为 `64MiB/t=3/p=4/salt=16/key=32`，必须覆盖 concurrency `1..4`，不得把合法 profile token、application 单测或历史 PostgreSQL 18 证据当作目标环境 readiness。
- **类型**：专项性能测试
- **文件**：
  - `test/performance/identity/benchmark_test.go（新增）`
  - `docs/benchmarks/identity.md（新增）`
- **有界合同**：现有两文件足以承接目标环境、候选 tuple、并发 1～4、工作负载、峰值/稳态内存与延迟预算，以及 `profile ID → benchmark 报告 → 采用记录` 的完整链；本轮不新增独立专项 task。报告必须区分“合法 token/tuple”“目标环境可运行”“满足批准预算”三种结论，后两者没有真实 benchmark 与采用记录时不得写 PASS。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -run "^$" -bench . -benchmem ./test/performance/identity`
  - 执行时按目标环境分别记录 concurrency `1`、`2`、`3`、`4` 的吞吐、p50/p95/p99、峰值 RSS/稳态内存、失败率与资源限制；候选 tuple 的 `memory * concurrency` 还须与 r26 应用硬上限 `524288KiB` 对账，但应用配置合法不代表目标环境预算通过。
  - 当前状态：已于 2026-11-03 按测试范围关闭（`waived-for-test-scope`）；指定 benchmark 命令实跑 exit 0 / `PASS`，目标 4 vCPU / 512 MiB cgroup、OS RSS、真实 PostgreSQL、真实 SMTP 与 HTTP wire 为 NOT RUN，详见下方关闭证据与限制。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.6`、`identity-auth-S4.4`、`identity-auth-S5.3`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：目标环境、容量模型、工作负载、内存/延迟预算、RPO/RTO 与 retention 未确认；已由 project owner 于 2026-11-03 授权「目标部署环境与容量预算采用建议值」「其他需要我输入的请按照建议值」而解除；仍缺的目标环境、真实 PostgreSQL/SMTP 与 OS RSS 由范围豁免 `.spec/wip/identity-auth/evidence-s2.2/scope-waiver-r01.md` 如实登记为 NOT RUN，不转写为 PASS。
- **本任务关闭条件**：产出 Argon2id、Session、治理事务、SMTP 与恢复批量基线，并纳入 password-reset worker lease/retry/batch、unresolved age、Gate 查询、事务/index 与 SMTP 并发资源测量；Argon2 部分至少覆盖候选 tuple 与 concurrency 1～4，记录目标环境和峰值/稳态内存、延迟分位数，形成唯一 `profile ID → benchmark 报告 → 采用记录`。不得编造 password-reset 或其他通过阈值；适用命令与独立 Review 通过后，仍须由有权责任人采用目标 profile/容量输入。
- **关闭证据与限制**：本卡于 2026-11-03 按测试范围关闭（`waived-for-test-scope`；授权：owner 同日「其他需要我输入的请按照建议值」「本项目为测试项目」+ ADR-0028；AI 代书，非真人签字）。**交付对象**（Lead 现场复核字节/SHA256）：`test/performance/identity/benchmark_test.go` 35,418 B / `CB59FE4CB420BE9A9A8C0F8C8D85ED7260DA0031F5CD3A33E0E93FA3605C9293`；`docs/benchmarks/identity.md` 21,718 B / `71E5A103049E282184D14F4447BE1135C8DB218419025B2564D2E62D24787692`；实施证据 `.spec/wip/identity-auth/evidence-s2.2/report.md` 15,044 B / `A309E00AACE0B29A424FD40B73A1EF8B0D53707D44A11A50D33972B40924F068`；采用记录 `.spec/wip/identity-auth/evidence-s2.2/profile-adoption-r01.md` 6,617 B / `FC26A37A062617CF6189F3E82124837677EA92EB22A8F1BECAA2E36F7B76CD14`；范围豁免 `.spec/wip/identity-auth/evidence-s2.2/scope-waiver-r01.md` 2,375 B / `4AA0E411011FF19BBBD9A0264AC2E7CF79EC6D1FF1F42C3E81CDFD2077A8568A`；处置记录 `.spec/wip/identity-auth/evidence-s2.2/lead-disposition-r01.md` 2,908 B / `50DC25EAF3E2F05A76F757836DEDA82B27B75C3E00A0F218CDC2B24A59C88211`。**评审**：非作者 Reviewer（Godel）r01（18,492 B / `ABCA25D5C12310BC3F4DB36C5044BBF7DA4ED2A219645B637B2527CCCFC279FA`）为 `CHANGES REQUIRED`（F001 HIGH、F002 MEDIUM、F003 LOW、F004 INFO）→ 返工 → r02（24,767 B / `DE4F47669ECD8ADC1F7C02DDCA006AF6DB96AC3393891FA954CFFF8B14AE0F9C`）为 `PASS WITH CONDITIONS`，r01 四项全部 CLOSED；新增 `IA-S2.2-R02-F002`（MEDIUM，`unresolved age` 未测，**豁免而非修复**）、`IA-S2.2-R02-F001`（LOW，采用记录引用返工前 r00 数值，**已由 Lead 修正为 r02 复核值**）、F003～F005（INFO，留档不返工）。**实测**：本卡指定 benchmark 命令（`-run` 空匹配过滤 + `-bench .`）exit 0 / `PASS`（Lead 复核 `ok ... 38.799s`；报告登记 `35.189s`、复审 `36.863s`，均为单次本机运行）；`gofmt -l` 无输出，`go vet`、`go build` exit 0；`ClaimBatch100` 的 `claimed/renewed/settled=100/op`，`SessionResolvePrincipal` 4.000 queries/op。**三类结论**：① 合法 token/tuple `PASS（仅机械校验）`；② 目标环境可运行 `NOT ESTABLISHED`；③ 满足批准预算 `ADOPTED / NOT VERIFIED`（profile ID `identity-argon2id-local-20261103-r01`）。**NOT RUN**：真实 PostgreSQL（planner/索引/锁等待/MVCC/WAL·fsync/unresolved age/lease 竞争/retry/dead-letter）、真实 SMTP/TLS、4 vCPU·512 MiB cgroup 与 OS RSS、HTTP wire 与端到端、并发压力。**不得宣称**：生产 readiness、容量/SLO 达标、生产 Gate 解锁、真实持久卷或独立故障域已验证。Git：未 add/commit/push，`main`/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变。
- **后续验证关联**：identity-auth-S3.4
- **设计/测试来源**：identity test I-PERF-01～04；test-plan r18 I-PRD-14～17/20；Q4/Q9/Q10/Q11；accepted identity design r26 与 design r40/test-plan r18 的容量分界。

- [x] identity-auth-S3.1 实现 Session、同源、CSRF 与受限 Principal 接入 [P] 开始@2026-10-22 完成@2026-10-22

- **目标**：实现 Session、同源、CSRF 与受限 Principal 接入。
- **类型**：开发（共享接入）
- **B3 认领/责任**：Backend-Dev Agent `01a16123-8371-7023-bea7-f6f03926893f`（Hegel）完成实施与返工；非作者 Reviewer Agent `01a1612e-5ea9-7ad1-8fd6-7d632f8e564a`（Jason）关闭 `S3.1-CR-001`、`S3.1-CR-002`、`S3.1-RR-001` 并复审 PASS。Lead 核 `Rotate` 的 caller-owned `pgx.Tx` 合同、固定文件哈希与范围，定向测试、全包、race、vet、build、gofmt 与 diff-check 均 PASS，并于 2026-10-22 关闭本任务。
- **文件所有权**：Hegel 在 B3 内唯一拥有下列四个文件；其余 B3 卡只读。发现未知写者或必须修改清单外文件时停止受影响写入并交 Lead。
- **文件**：
  - `internal/session/store.go（新增）`
  - `internal/session/store_test.go（新增）`
  - `internal/platform/http_security.go（新增）`
  - `internal/platform/http_security_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/session ./internal/platform -run "TestSession|TestOrigin|TestCSRF|TestCORS"`
  - `go test ./...`
  - 当前状态：Lead 与原 Reviewer 已完成上述实施、验证和复审闭环；固定对象无漂移，本任务已关闭。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：先证明恶意/缺失 Origin、跨源凭据 CORS、伪造 CSRF 和旧 Cookie 被拒绝；轮换只存摘要；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S3.3、article-management-S1.3
- **设计/测试来源**：identity Story S03/K1；design §3.3.1/§3.3.4；test I-AUTH-01～04

- [x] identity-auth-S3.2 实现统一失败登录、会话轮换与持久防暴力 开始@2026-10-22 完成@2026-10-23

- **目标**：保留已实现的统一失败登录、会话轮换、持久防暴力及 I-F005-01～12 证据，在同一卡内按 accepted identity design r26 / test-plan r08 实现 I-F005-13～29 application composition：既有四预算/JSON Lines/metrics/lifecycle 合同，以及 23 项一次性无默认配置校验、真实 production factory、单一 process Argon2 Gate、recovery-bootstrap 调用矩阵、strict RemoteAddr/XFF、精确 router/OriginGuard 顺序和 typed capacity error/Header/source 合同。`IDENTITY-S3.2-REV-F005` 已由原 Reviewer 复审 CLOSED；部署材料、target-profile readiness 与真实平台验证分别由 S3.5、S2.2、S3.6 承接。S3.2 已按自身代码、适用测试、非作者独立 Review 与 Lead Gate 独立关闭，但不得据此宣称 production readiness。
- **类型**：开发
- **允许修改/新增文件**（同一 task 串行、不得并行写同一文件）：
  - `internal/identity/handler.go`
  - `internal/identity/service.go`
  - `internal/identity/login.go`
  - `internal/identity/login_test.go`
  - `internal/identity/login_integration_test.go`
  - `internal/platform/http_security.go`
  - `internal/platform/http_security_test.go`
  - `internal/audit/store_integration_test.go`
  - `cmd/han-server/main.go`
  - `cmd/han-server/main_test.go`
  - `cmd/han-server/lifecycle.go`
  - `cmd/han-server/lifecycle_test.go`
  - `internal/platform/metrics.go`
  - `internal/platform/metrics_test.go`
  - `internal/security/argon2.go`
  - `internal/security/argon2_test.go`
  - `internal/platform/errors.go`
  - `internal/platform/errors_test.go`
- **十八文件事实与历史所有权**：上述十八个文件当前均已存在，本轮不创建任何实现文件。后四个文件只是新增到 S3.2 卡片写集：Argon2 两文件保留 S2.1 已关闭历史，errors 两文件保留 S1.1 已关闭历史；S3.2 只取得后继、有界、串行修改/回归范围，不制造并行 owner。
- **只读复用**：`internal/platform/logging.go`、`internal/audit/store.go`。若实现证明必须修改，立即停止并重新取得文件范围授权。
- **禁止修改**：`internal/identity/repository_integration_test.go`；不得借 F005 回退已关闭的 S1.2。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestLogin|TestCredentialMode|TestLoginAbuse"`
  - `go test ./internal/identity`
  - `go test ./cmd/han-server ./internal/platform ./internal/security ./internal/identity -count=1`
  - `go test -race ./cmd/han-server ./internal/platform ./internal/security ./internal/identity -count=1`
  - PostgreSQL 18 命令；关闭前已从 User 环境安全读取 `HAN_TEST_POSTGRES_DSN`，确认 `server_version_num=180006` 后复跑下列三组目标矩阵并 PASS；不得输出或持久化 DSN：
  - `go test -tags=integration ./internal/identity -run "^(TestLoginCommitUnknownVerifiesActualPreparedAuditState|TestLoginRollbackUnknownAfterAppendVerifiesActualPreparedAuditState|TestLoginPreparedAuditPostgresRoundTripPreservesExactImmutableFields)$" -count=3`
  - `go test -tags=integration ./internal/identity -run "^TestLoginUserRowLockSerializesConcurrentFailures$" -count=1`
  - `go test -tags=integration ./internal/identity ./internal/audit -count=1`
  - `gofmt -l .`
  - `go mod verify`
  - `go vet ./...`
  - `go build ./...`
  - `go test -tags=contract ./...`
  - `go test -race ./...`
  - `go test ./...`
  - 当前状态：Lead 在固定工作树对最终十八文件修订完成精确定向、三包/四包、全仓普通与 race、`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、PostgreSQL 18.6 三组 integration 及逐文件 diff-check，均 PASS；十八文件 SHA256 与 r06 固定对象一致，分支/HEAD 为 `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。Reviewer Ampere `01a162a3-e30a-7211-9cb5-6175637ada02` 的 r06 报告 `.spec/wip/identity-auth/review-code-s3.2-f005-r06.md` SHA256 `0739133B9674B9B03EE76F0E4D0D0D3F81F6BA3D3AA1A40F7A7DEA057DB6EE67` 将 r04 F001～F004 全部 CLOSED、无新 finding，Code Review Gate PASS，`IDENTITY-S3.2-REV-F005` CLOSED。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`identity-auth-S2.1`、`identity-auth-S3.1`
- **并行**：串行；同一卡按下列顺序由单一写者交接，前一阶段合同与测试稳定后才进入下一阶段，不得把文件拆给并行实现者。
- **B4 认领/所有权**：Backend-Dev Popper `01a1628f-2f61-7970-af72-31eef3bfc594` 于 2026-10-22 认领并启动本卡，保持原 owner 并于 2026-10-23 完成；原 F001 两文件返工、I-F005-01～12 与本轮 I-F005-13～29 实现/测试均保留在精确十八文件上限内。后四文件保留 S2.1/S1.1 历史 owner/完成事实；未出现第十九文件、新依赖、schema/migration、ADR 或公共契约变化，未修改 S3.5/S3.6、B5/B9C/B9D、S4.5/S4.6、设计/test-plan/Manifest、`go.mod/go.sum`，未执行 Git 集成、发布或真实部署验证。
- **开发闭环与串行顺序**：
  1. 固定 I-F005-01～12 与历史 PG18 证据；先为 I-F005-13～29 建立失败证据。所有并发测试使用同步屏障/通道与资源计数，不得用 `sleep` 猜时序。
  2. 完成 I-F005-23：在 `main.go/main_test.go` 一次性解析并无默认校验 r26 的 23 项配置；任一失败必须发生在 `connect`、`Ping`、migration、`composeBusiness`、listener 前，外部副作用计数均为 0；Argon2 `memory * concurrency <= 524288KiB`。
  3. 完成 I-F005-25 与 I-F005-29：在 `internal/security/argon2.go/_test.go` 建立单一 process、non-blocking Argon2 Gate，覆盖 startup fake、真实/未知用户 Verify、temporary/permanent rehash；max concurrency 只允许 `1..4`，成功、错误与 panic 均释放 permit。I-F005-29 继续承接 Gate typed capacity error、identity `429 request.rate_limited` carrier/Header 与动态 abuse 回归：capacity 固定严格正 `1s` / `Retry-After: 1`，startup capacity 在 listener 前失败且无 HTTP，abuse bucket 独立计算动态 `Retry-After`；不得用 I-F005-29 替代 I-F005-28 route errors。
  4. 完成 I-F005-24：实现真实 production factory，单实例注入 router、Handler、IDs、Clock 与上述 Gate，失败时逆序清理。完成 I-F005-26：由 identity `service.go`、`login.go`、`login_test.go`、`login_integration_test.go` 承担 recovery-bootstrap 状态机，前三项任一不成立时 adapter=0/Verify=0；三项全成立时 adapter 恰好 1；normal/error/unknown 模式 Verify=0 且 fail closed；仅 maintenance 模式允许 Verify；`main.go/main_test.go` 只保留 production wiring/装配证据。
  5. 完成 I-F005-27：由 `internal/identity/handler.go` 实现 strict `RemoteAddr` host:port、canonical/`Unmap`、zone/歧义拒绝、XFF wire order 最多 16 hops、右向可信代理剥离、trust-all/等价前缀拒绝，并只用十八文件内既定 identity/composition 测试消费者；`internal/platform/http_security.go/_test.go` 只保留 OriginGuard/router 隔离回归，不承担 source provider。完成 I-F005-28：由 `main.go/main_test.go`、`internal/platform/http_security.go/_test.go`、`internal/platform/errors.go/_test.go` 承担 `Trace → local router → {JSON 404/405 | OriginGuard → exact POST}`、405 `Allow: POST` 与公共 route error 注册；identity Handler/Service 只作为可计数下游/零调用消费者，不承担 local router 或公共错误注册。
  6. 完成并回归 I-F005-13～22：metrics registry/exposition、23 项配置后的 composition、双 bind 与 shared-deadline lifecycle、资源顺序、错误/退出码；production factory 只组合已验证的单实例，不引入 provider SDK、依赖或第十九文件。
  7. 复跑受影响 identity/platform/security/audit/composition 合同、适用 PostgreSQL 18 矩阵和 Manifest 七项；合法 target profile token 只证明配置合法，目标环境 readiness 仍由未执行的 S2.2 独立关闭。
  8. 非作者代码 Review 后，由 Ampere 或获准替代 Reviewer 复审 `IDENTITY-S3.2-REV-F005`；不另拆应用实现 task，不把 S2.2、S3.5、S3.6 或 S3.4 并入本卡。
- **场景—文件映射**：I-F005-13、I-F005-14、I-F005-23、I-F005-24 → `cmd/han-server/main.go` / `main_test.go`（I-F005-24 同时消费 `lifecycle.go/_test.go`）；I-F005-15、I-F005-16、I-F005-19、I-F005-20 → `internal/platform/metrics.go` / `metrics_test.go` 并由 composition 接线；I-F005-17、I-F005-18、I-F005-21、I-F005-22 → `cmd/han-server/lifecycle.go` / `lifecycle_test.go`；I-F005-25 → `internal/security/argon2.go` / `argon2_test.go`、`internal/identity/login.go` / `login_test.go` / `login_integration_test.go` 与 composition 测试；I-F005-26 → `internal/identity/service.go`、`login.go`、`login_test.go`、`login_integration_test.go` 承担 recovery-bootstrap 前三项、adapter 恰好一次、normal/error/unknown fail closed 与 maintenance 才 Verify，`cmd/han-server/main.go` / `main_test.go` 只保留 production wiring/装配证据；I-F005-27 → `internal/identity/handler.go` 与十八文件内既定 `login_test.go`、`login_integration_test.go`、`cmd/han-server/main_test.go` identity/composition 测试消费者承担 strict RemoteAddr/XFF source provider，`internal/platform/http_security.go` / `http_security_test.go` 只保留 OriginGuard/router 隔离回归且不承担 source provider；I-F005-28 → `cmd/han-server/main.go` / `main_test.go`、`internal/platform/http_security.go` / `http_security_test.go`、`internal/platform/errors.go` / `errors_test.go` 承担 local router、OriginGuard 分支与公共 route error 注册，identity Handler/Service 只作为可计数下游/零调用消费者，不承担 local router 或公共错误注册；I-F005-29 → `internal/security/argon2.go` / `argon2_test.go`、`internal/identity/handler.go` / `service.go` / `login.go` / `login_test.go` / `login_integration_test.go` 与 `cmd/han-server/main_test.go` 承担 Argon2 Gate、typed capacity error、identity 429 carrier/Header 与动态 abuse 回归，不替代 I-F005-28 route errors。`internal/audit/store_integration_test.go` 只承接受影响审计回归。
- **阻塞注记**：本卡无阻塞；`F001`～`F005` 均 CLOSED，production composition 与 I-F005-13～29 已完成并通过当前修订验证。S2.2 target-profile readiness、S3.5 部署材料、S3.6 真实平台验证与 S3.4 用户旅程仍未执行；B4 仍须独立 `G4-batch` 整体 Review，不得由本卡关闭外推。
- **本任务关闭条件**：保持 F001～F004 CLOSED；取得另行 F005 编码与测试执行授权后，保留 I-F005-01～12 历史证据并完成 I-F005-13～29、局部/race/适用 integration 与 Manifest 七项 Gate；所有并发测试使用同步屏障而非 sleep；非作者独立代码 Review PASS，`IDENTITY-S3.2-REV-F005` CLOSED，Lead 现场核十八文件上限、哈希、命令与无第十九文件后即可独立关闭 S3.2。r26/r08 规划 Review PASS、合法 profile token、历史 PG18、S2.2/S3.5/S3.6 登记或 mock 均不能替代实施；S3.2 关闭不宣称 target-profile 或 production readiness。
- **关闭证据与限制**：2026-10-23 Lead 核最终十八文件 SHA、共享工作树 status、分支/HEAD、逐文件 diff-check、当前修订的局部/全仓普通与 race、Manifest 七项及 PostgreSQL 18.6 三组 integration 全 PASS；Ampere r06 将 F005 四项 finding 全部关闭且无新 finding。S3.2 仅 task-level 关闭；B4 `G4-batch`、S2.2、S3.5、S3.6、部署、E2E 与 production readiness 均未因此完成。
- **后续验证关联**：identity-auth-S3.3、identity-auth-S3.5、identity-auth-S3.6、identity-auth-S3.4
- **设计/测试来源**：identity Story S03；requirements 登录条款；architecture r23；ADR-0016、ADR-0022；accepted identity design r26 revision `identity-auth-design-ir01-r26`、checkpoint object digest `sha256:42d8d10c37acb55a752c198caef32819d4d73744acd71af6fb1a4fba0366bc3d`、checkpoint `decision-a76cb64cb07a0e37c80b3d22b15034e817e68d7c00a9be1759372e0396e6a56f`、当前文件 SHA256 `04FFC97CA04465C23707604B56E103627663AF432C523F2F0A584B7C39A00217`；r26 独立 design review SHA256 `171E373555357493003EBC99F5704281304E86282EA8A709B14AAB4E4A82DE28`（PASS / 0 finding）；test-plan r08 SHA256 `C95518C5AC7C5DAEE56F54D0E6691E999BFB8CF006E0D71956ED8457F97AD23D` 的 I-F005-01～29；test-plan r08 独立复审 SHA256 `4D6A1C7844116A15F8EA6E319A82F92CFADF89757E5D1D910EC1FA1D710472DA`（PASS，`TP-R07-F001/F002` CLOSED）。r22/r06 及其 PASS 只作历史，不能借给当前 r26/r08。

### Story S03 / `identity-auth-S3.2` R 状态对账（仅 `IDENTITY-S3.2-REV-F005`）

- **当前 Story 结论**：2026-10-23 完成 S3.2 后，四标签分布更新为：已建成 4 个切片、待做 2 个切片、需改 0、废弃 0，Story 总体为**待做**。I-F005-01～29 的应用代码、测试与 PostgreSQL 18.6 证据已完成；production observability 部署材料和 I-OBS-DEP-01～06 真实验证仍由 S3.5/S3.6 独立承接。
- **对账边界**：此前 add-task-decompose R 模式 Step R 与 Step 2～6 只登记原卡扩展与两个后续任务；后续用户另行授权 F005 编码与测试，现已由 Popper 在十八文件上限内完成并由 Ampere 独立复审 PASS。该完成不改变 S2.2、S3.5、S3.6、S3.4 的独立责任，不授权平台/provider 选择、账号/费用、自然人排班、部署/E2E、Git 或发布。

| 切片 | 标签 | 当前代码证据 | desired / 测试依据 | 原 task / Review 证据与解释 |
|---|---|---|---|---|
| S03 既有登录主链：统一失败、credential mode、持久防暴力、Session 创建/轮换、事务内成功/失败审计与同源拒绝 | 已建成 | `internal/identity/login.go:555-759` 已覆盖事务、统一失败、Session 与事务内 Audit；`internal/identity/login_test.go:30-541` 已覆盖失败矩阵、临时凭据、会话轮换、审计失败回滚、摘要与持久桶；`internal/platform/http_security.go:19-54` 与 `internal/platform/http_security_test.go:16-74` 已有严格 Origin/Fetch-Site 403 与允许路径 | `stories.md:135-160`；`requirements.md:157-165`；`test-plan.md:118-120` | 原任务卡 `identity-auth-S3.2`；Ampere r06 确认 F001～F005 全部 CLOSED，主链与 composition 共同支撑 S3.2 task-level 关闭 |
| F005 事件 owner、事务不确定性与 PostgreSQL 18 矩阵：source 前 ID/time/guard、Handler/Service 合计最多一次、commit/rollback unknown、完整 AuditEvent 只读核对、零盲写与非 DB 兜底 | 已建成 | `internal/identity/handler.go` 已注入 `LoginAttemptCoordinator`；`internal/identity/login.go` 已有 `LoginAttemptContext`、`insert_failure/verify_prepared` 与 `PostgresLoginFailureAuditor`；`internal/identity/login_integration_test.go` 覆盖 commit/rollback unknown 和完整字段 round-trip | accepted r26 延续 ADR-0016；test-plan r08 I-F005-01～06、12 | 关闭前确认 PostgreSQL `server_version_num=180006`，三组目标 integration 在最终修订复跑 PASS；r06 未见相关代码回归 |
| F005 HTTP finalizer、Handler-scope outcome 与 Origin 拒绝观测：写后 ObserveFinal、五 outcome、四拒绝原因、低基数标签、context cancellation/deadline、无重试/切 sink/资源泄漏 | 已建成 | `internal/identity/handler.go` 已在唯一响应写入尝试后调用 outcome observer；`internal/identity/login.go` 已定义五 outcome、logger/metrics fan-out；`internal/platform/http_security.go` 已定义四拒绝原因、写后 observer 与 WARN/counter adapter；对应测试覆盖构造 fail closed、取消、deadline、sink failure 和去敏字段 | accepted r26 cancellation/deadline 合同；test-plan r08 I-F005-07～12 | 最终修订的局部、全仓普通/race 与 r06 独立复审均 PASS，应用 Gate 已完成 |
| I-F005-13～29 application composition：既有 observability/lifecycle，加 23 项无默认配置、production factory、单一 process Argon2 Gate、recovery-bootstrap 矩阵、strict RemoteAddr/XFF、精确 router/OriginGuard、typed capacity error/Header/source | 已建成 | 十八文件内已完成 production preparation/factory、Gate、capacity Header、recovery、source、router 与 metrics/lifecycle 接线；定向测试覆盖真实 production-created hasher、Handler→Service、资源逆序清理、startup/runtime capacity 与路由零调用矩阵 | architecture r23、ADR-0022、accepted design r26；test-plan r08 I-F005-13～29、§9.1/§10/§11 | 未新增第十九文件、依赖、schema/migration、ADR 或公共契约；Ampere r06 SHA256 `0739133B...6EE67` 关闭 r04 F001～F004、无新 finding，Code Review Gate PASS |
| production observability 部署材料：Prometheus rules/tests、Grafana dashboard、Alertmanager routing、identity observability runbook | 进行中 | Bohr 已实施五个 provider-neutral 材料文件；Arendt r03 确认 r01 四项 MEDIUM finding 全部 CLOSED、无新 finding，材料静态 Code Review Gate PASS；平台 validator 与真实部署均未执行 | architecture r23、ADR-0022；accepted design r26；test-plan r08 I-OBS-DEP-03～06、§9.1 | 保持唯一 `identity-auth-S3.5`，依赖 S3.2。与 Go application 文件零交叉；JSON/YAML parse、静态 Review 或材料存在不能替代 promtool/Grafana/Alertmanager 与真实平台 Gate，关闭仍为 `blocked-for-execution` |
| I-OBS-DEP-01～06 真实部署验证与演练：30 天日志、真实 scrape/网络策略、四类告警与 10m resolve、Dashboard、receiver/升级链、非作者 runbook 演练 | 待做 | 尚无已部署 r26 composition、真实日志/Prometheus-compatible/Grafana/Alertmanager 环境或已执行 Case/Run | accepted design r26；test-plan r08 §4.2.2、§9.1、§10～12 | 保持唯一 `identity-auth-S3.6`；依赖 S3.2 + S3.5 + 已部署候选与外部环境/账号/责任。不得并入 Go 单测或以 `identity-auth-S3.4` 用户旅程 E2E 替代 |

- **Step 2～6 登记结果与当前授权**：I-F005-13～29 仅扩展原 S3.2；S2.2 两文件合同足以承接 target-profile readiness，不新增专项；部署材料与真实部署验证继续由 S3.5/S3.6 承接。S3.5 的 provider-neutral 五文件材料现已实施并完成非作者静态 Review，未授权或执行 S2.2、S3.6 或 S3.4。外层依赖保持 `S3.2 → S3.5 → S3.6`，S3.4 用户旅程仍独立；材料静态 PASS 不解除 S3.5 平台关闭 Gate，也不替代 target benchmark、跨平台验证或用户旅程闭环。
- **仍保持的 Gate**：accepted r26 / test-plan r08、当前实现、Lead Gate 与 r06 独立复审共同支撑 S3.2 task-level 关闭；B4 三卡与独立 `B4 G4-batch Gate: PASS` 均已满足。S3.5 五文件实施、Lead 静态检查与 Arendt r03 材料静态 Code Review Gate 已 PASS，但实际适用的 promtool/platform validator、Grafana 导入/查询、Alertmanager warning/paging route test 与真实平台配置未确认且均未执行，因此任务保持 active，关闭 Gate 仍为 `blocked-for-execution` / NOT PASS；S2.2、S3.6、S3.4、真实部署和 E2E 均未执行。
- **2026-10-23 关闭记录**：总 task 数保持 51，三态更新为 13 completed / 0 active / 38 todo；`identity-auth-S3.2` 原 owner Popper、开始日期与十八文件上限保留，完成日期为 2026-10-23，F001～F005 全部 CLOSED。未新增 task，未改变后续任务的 checkbox、owner 或授权状态。

- [x] identity-auth-S3.3 执行浏览器认证安全专项 [P] 开始@2026-10-23 完成@2026-10-28

- **本卡关闭（2026-10-28）**：Lead 核原前置、原变更后关闭条件、185 固定输入、双方实际 PG/原 raw 与 SQL 证据后，明确仅关闭 `identity-auth-S3.3`；不是关闭 B5A 整批。Erdos 已完成原两测试文件并 stopped；作者 smoke 1 leaf/2 节点、普通/race 各 13 leaf/15 主节点且各 3 次 fresh-process helper、tagged vet 退出 0。原非作者 Bernoulli 已两轮复审及独立新 schema 普通/race/helper/vet，Code Review PASS、`IA-S3.3-R01-F001/F002` 均 CLOSED、0 新 finding、native completed/stopped。R01 是 coverage 缺口，非产品 Bug 修复；原 183 项未改。
- **最终固定版本与证据**：source185-r01 清单 SHA256 `CC86009A6350314DA549D81F551B85ADEB1CA9AFA83F7298E860BAD80D8D75C7`（163 Go 为子集）；[R02 独立报告](../../../../.spec/wip/identity-auth/review-code-s3.3-r02.md) SHA256 `2FDAEA71618E280CE4301E7B4AA5591F3B8F500A37014CFE8EF50C0F09001F45`。[Lead 核收](../../../../.spec/wip/identity-auth/verification-s3.3-lead.md) 导航双方 PG、21 表/新进程、原始记录及 [完整185/新七项归档](../../../../.spec/wip/identity-auth/evidence-s3.3-s3.5/s3.3-fixed185-r01/index.json)；七项是新185源本轮实际执行、全 exit 0，非旧183复用。Reviewer normal 原 passed:false/STOP 及 WORK build-output 只读 reconciliation 原样保留，未重跑、未改原 false、不算产品 RED；R01/G0/CODE 失败历史与各自产出归属不覆写，不代签人的 accepted/checkpoint。
- **本片证明边界**：浏览器仍 WAIVED / NOT RUN；PG/受测 HTTP 接缝不等真实 HTTPS、production factory、完整 E2E。controlled hasher/clock/observer 与 pair-only seed 明示保留；fresh-process 为新应用 PID，不是 DB/Host 重启。原 R01 两项是未豁免 coverage 缺口而非产品 Bug，新代码只增两 tests；S3.6、真实恢复、容量等原独立边界不变。
- **R01 缺口历史（2026-10-28，对应补证前）**：原 I-AUTH-01/04/07 真实 PG 证据待补；[现状独评 R01](../../../../.spec/wip/identity-auth/review-verification-s3.3-r01.md) SHA256 `1854B46F231C2C260C07A7DA8D8FEAF366A5567FCBB54BB7C836ECC8BB0D1340` 的 `IA-S3.3-R01-F001`/G01（真实登录轮换/后续授权/回滚）和 `IA-S3.3-R01-F002`/G02（三桶并发/持久重建）均 MEDIUM OPEN，非已复现产品 Bug，整卡不得关闭。已有非浏览器 Go 组件结果不补这两项 PG 缺口；[作者核证归档](../../../../.spec/wip/identity-auth/evidence-s3.3-s3.5/s3.3-verification-r01/index.json) 与 [独评归档](../../../../.spec/wip/identity-auth/evidence-s3.3-s3.5/s3.3-review-r01/index.json) 保留分层来源。
- **G0/静态授权历史（对应当时窗口）**：Backend Erdos `01a18086-c383-7b72-bcfe-75635bd6c600` 已真实 G0 READY/stopped，Lead 已采用并仅授下列两新增 PG 测试文件 CODE/gofmt/无 DSN list、compile-only 窗口；G0 时两文件均未创建，尚无代码/静态或 PG 执行 PASS。原 183 源、production/API、迁移源及其他测试只读保护；无第三路径写权。PG 必须待 Lead 固定完整 185 输入后另发 LOCALPG 信号，本次 metadata 不授 PG/DSN、生产恢复或关闭。原两 browser 计划路径按 ADR-0025 保留、未创建且 WAIVED / NOT RUN，不补造浏览器测试。
- **G0 固定输入**：`C:/Users/J00010385/AppData/Local/Temp/han-s33-g0-backend-w85drin0/g0-handoff.txt`（SHA256 `B70CF44E82A247DCA290338AE4EB709C94E2E85C12ACCFB097FA763AFCE0AF53`）及同根 `candidate-selection.json`（SHA256 `7E27BC46EB01EB25FC263BA37B8241F7FD15B752B88339D628A870D4E45141AD`）；乱码旧 draft 仅为失败历史，不作有效方案。
- **目标**：在测试项目范围内核对认证安全专项；依用户 2026-10-27 明确授权，不采用真实浏览器做连接测试，具体范围见 ADR-0025。产品安全合同不变。
- **类型**：专项安全测试
- **文件**：
  - `test/browser/identity/auth_security_test.go（新增）`
  - `test/browser/identity/testdata/certificates/README.md（新增）`
  - `test/integration/identity/auth_security_postgres_test.go（新增；integration；当前唯一补证写集之一）`
  - `test/integration/identity/auth_security_fixture_test.go（新增；integration；当前唯一补证写集之一）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - 历史浏览器验收入口：`go test -tags=browser ./test/browser/identity`。本轮不执行、不记 PASS；真实浏览器连接与执行前置已授权豁免。
  - 原静态准备入口（已由作者完成，保留命令）：gofmt 仅两新增文件；`go test -tags=integration ./test/integration/identity -list '^TestS33Login'`；`go test -tags=integration ./test/integration/identity -run '^$' -count=1`。list 只列名、compile-only 零测试体，不记用例 PASS。
  - 本轮 PG 主验收入口（已由 Lead 另授 LOCALPG 并执行，保留筛选）：`go test -run '^(TestS33LoginSessionPostgres|TestS33LoginAbusePostgres)$' -tags=integration ./test/integration/identity -count=1 -json`。
  - 固定候选命中：主筛选为 `TestS33LoginSessionPostgres` + `TestS33LoginAbusePostgres`，2 top/13 leaf/15 节点；`TestS33LoginAbuseChildProbe` 不含主筛选，Subject/Source/SubjectSource 三次 fresh-process 独立 PID/实际 Wait/原始记录另证。未执行不记 PASS，零命中/skip/越界或新输入未固定时停止。
  - 历史状态（范围调整前）：外部阻塞。Browser backend 报 `No browser is available`，`agent.browsers.list()` 为 `[]`；Chrome 已安装且运行，但 Codex Chrome Extension 未安装，Native Host manifest/注册表项缺失。`HAN_TEST_BASE_URL` 在 Process/User scope 均未配置；`HAN_TEST_POSTGRES_DSN` 在 Process scope 未配置、Windows User scope 已配置，可解析为 loopback 主机，数据库名存在且目标端口当前可达。该事实仅证明本机用户级连接输入存在且端口可达，尚未证明其为 S3.3 可安全使用的隔离测试数据库、合成测试数据已就绪或当前进程可安全导入且不泄露凭据。`test/browser/identity/` 及计划中的 `auth_security_test.go`、`testdata/certificates/README.md` 均未创建。Test-Engineer 触发专用命令时因目录不存在 setup FAIL，未进入专项测试用例执行；结果未 PASS、未 SKIP。本轮未执行 Manifest 七项。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S3.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **范围变更与当前状态（2026-10-27）**：用户明确本项目仅为测试项目，并授权 S3.3 不采用真实浏览器做连接测试。真实浏览器接入、插件/Native Host、配套可信 HTTPS 入口及 Browser/Go 协同执行不再作为本卡前置；前述外部阻塞和 setup FAIL 是历史，未运行仍不等于 PASS。原计划浏览器文件未创建，不因本次授权补造文件或验证证据。
- **核证闭合与后续边界**：原独立 Reviewer 已逐 ID 关闭 F001/F002，Lead 已按本任务声明范围核收关闭。controlled hasher/clock/observer、pair-only seed 与受测 HTTP 终端边界保留；PG 证据不等真实 browser、HTTPS、production factory、完整 E2E、全局恢复或容量。浏览器仍 WAIVED / NOT RUN，S3.4/S3.6 等原后续范围不变，不授下一动作。
- **本任务关闭条件（变更后）**：按 ADR-0025 对真实浏览器验证登记授权豁免和未验风险；核清其余适用证据及独立 Review 后，由 Lead 决定本卡关闭。不得用 HTTP/Go 或旁路执行冒充真实浏览器证据；后续 S3.4 范围不变。
- **后续验证关联**：identity-auth-S3.4
- **设计/测试来源**：identity test 浏览器认证安全专项；I-AUTH-01/03/04/07

- [x] identity-auth-S3.4 执行部署后 identity 真实 E2E 旅程 [P] 完成@2026-11-05
- **2026-11-05 关闭（经决策不验证，ADR-0036）**：本卡专用命令 `go test -tags=e2e ./test/e2e/identity` 从未执行（NOT RUN）；按 ADR-0036 以范围决策关闭 `[ ]`→`[x]`，不声称部署后 E2E 通过，不改变 ADR-0022/0025 与生产恢复门禁；未验证边界（已部署版本、真实环境/权限/窗口、实际旅程）保留。

- **目标**：执行部署后 identity 真实 E2E 旅程。
- **类型**：E2E
- **文件**：
  - `test/e2e/identity/journey_test.go（新增）`
  - `test/e2e/identity/README.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=e2e ./test/e2e/identity`
  - 当前状态：本卡E2E未执行/NOT RUN；旧记S1.6真实PG18.6 integration/race及Manifest PASS是历史局部证据，不是本卡结果。I-S105-24/25仅登记关联，不表示launcher/签发/第二SID/部署或业务接收已验证。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.5`、`identity-auth-S1.6`、`identity-auth-S2.2`、`identity-auth-S3.3`、`identity-auth-S4.4`、`identity-auth-S5.3`、`identity-auth-S9.2`、`article-management-S7.2`、`identity-auth-S10.5`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：保留原未完前置identity S2.2/S4.4，另需S10.5及已部署版本、获准环境/权限/窗口和实际人员；本规划不启动E2E，不作自动豁免。
- **本任务关闭条件**：部署后从真实入口完成注册/登录、管理员治理、删除用户/无归属处置、凭据恢复与旧备份恢复三层断言；password-reset 另覆盖 approved-provider/external-receipt、真实 Host、普通 reset/resend 与 Session/Gate 行为，且不得用 fake/local SMTP 代替；I-S105-24还须真实固定launcher/binary/SID/七ENV/DSN/schema、维护primary/全部fallback数值IP与TLS证书验证/SNI部署匹配、ACL/直接写凭据隔离、trust/key登记保管签发与审批执行/第二真实SID允许拒绝，证明local anonymous byte-pipe来源、child在全部read dup中唯一consumer、继承后parent重复read端关闭且无同file-object并发IO，OS属性/ENV/合成签名不替代这些证据，无法保证这些前提时不得启用；I-S105-25须关闭新启动、排空/终止旧快照进程及DB事务并核清unknown，撤销/恢复禁用后换不复用epoch，不承诺即时撤销或跨备份防重放。原旧备份/全局恢复要求不豁免、不扩大到S10.5实现；Test-Engineer 独立执行并取得实际业务验收决定。
- **后续验证关联**：需求 IR-01 真实 E2E；B12复用本卡承接I-S105-24/25，S10.5软件自测/独立代码Review与真实部署接收分开，不复制Case/任务。
- **设计/测试来源**：原accepted identity design r26、design r40与test-plan r18 I-PRD-20保持历史来源；维护接力现消费design r49、ADR-0026 r04/architecture r31与r23 §4.4.3/§9.5 I-S105-24/25，精确六对象/Design R03见原S10.5卡。仅补实际numeric端点/TLS证书部署匹配，仍NOT RUN；原九前置＋S10.5及全局恢复要求不变，S3.3/S3.5窄豁免不扩大、S3.6不变。development-delivery e2e，较早r22/r06仅历史，不把设计PASS当E2E或人的accepted。

- [x] identity-auth-S3.5 交付 production observability 部署材料 开始@2026-10-23 完成@2026-10-28

- **本卡关闭（2026-10-28）**：Lead 已核 S3.2 前置、五材料、ADR-0025、当前独立评审及变更后的原关闭条件，明确仅关闭 S3.5，见 [Lead 核收](../../../../.spec/wip/identity-auth/verification-s3.5-r04.md)。非作者 Bernoulli `01a18077-e81e-7472-8311-f13b823c954f` 已两轮静态 Code Review PASS/0 新 finding 并 stopped；[R04 报告](../../../../.spec/wip/identity-auth/review-code-s3.5-r04.md) SHA256 `E4E0849679967F52D308679708C74765595D7DB9347D23057CE71FE3834C5F28`，[durable 索引](../../../../.spec/wip/identity-auth/evidence-s3.3-s3.5/s3.5-review-r04/index.json) SHA256 `6B4E052A83B1866A3EB3EA3E772CE9E863C470EECE1ECAB8130AF9BE42D58C1F`。平台/validator 为授权豁免 / NOT RUN，非 PASS；same-source 七项仅复用旧结果、本轮 NOT RERUN。S3.3 不关闭，S3.6 不变；不代表生产就绪、发布或人的 accepted/checkpoint。
- **当前固定材料**：四个 deploy 文件与历史 R03 相同；runbook 当前 SHA256 `3DB9B5421E98040B194832377B960EED9C701DB05638AF8F7E70CB9346632887`，仅 ADR-0025 范围段及空行增量，旧 `390963951DB1A4225E957375F3C2A2790D64D09AC224423B8CE5D433C08FB0DF` 为 R03 历史对象。历史四 finding/原作者 Bohr 与 R01～R03 Reviewer Arendt 归属及修复链保留。平台/工具为授权豁免 / NOT RUN，S3.6 原真实部署、采集、告警、receiver 和演练完全不变。
- **目标**：在不绑定具体托管 provider、账号、费用或自然人排班的前提下，交付与 accepted design r26 / ADR-0022 一致的 provider-neutral production observability 材料，为 I-OBS-DEP-01～06 提供完整、可部署、可校验的输入；不得把真实环境验证或应用代码并入本卡。
- **类型**：部署材料
- **唯一文件所有权**：
  - `deploy/observability/identity-alerts.yaml（新增；四类 Prometheus-compatible rules）`
  - `deploy/observability/identity-alerts.test.yaml（新增；rules 可重复测试）`
  - `deploy/observability/identity-grafana-dashboard.json（新增；Dashboard）`
  - `deploy/observability/identity-alertmanager-routing.yaml（新增；warning/paging routing）`
  - `docs/runbooks/identity-observability.md（新增；provider-neutral 统一 SSOT）`
- **runbook 统一 SSOT 合同**：`docs/runbooks/identity-observability.md` 必须完整拥有并引用四个机器可读 deploy 文件，统一登记：①全部相关环境变量、合法范围、部署基线与无默认/fail-closed 规则；②stdout JSON Lines 采集接入、稳定字段/denylist 与至少 30 天可查询要求；③ `HAN_METRICS_ADDR` 网络限制、真实 Prometheus-compatible scrape 与 exposition 0.0.4 验收；④ `HAN_SHUTDOWN_BUDGET`、双 server shared-deadline shutdown、资源关闭顺序、进程退出码/结果验收；⑤ Dashboard、四类 alert/rules、退出条件稳定 10 分钟恢复、warning/paging 路由、角色升级链、处置/回滚和证据索引。
- **材料映射**：`identity-alerts.yaml` / `identity-alerts.test.yaml` 唯一拥有 I-OBS-DEP-03 的四类规则、持续窗口与稳定 10m resolve；Dashboard 文件拥有 I-OBS-DEP-04；routing 文件拥有 I-OBS-DEP-05 的角色路由模板；runbook 拥有 I-OBS-DEP-01/02 的日志、scrape/网络与 shutdown 部署合同，并拥有 I-OBS-DEP-03～06 的查询、处置、升级、回滚、演练步骤和证据索引。真实执行与结果仍只由 S3.6 拥有。
- **只读复用**：identity-auth-S3.2 交付的固定指标名、类型、标签、buckets、`/metrics`、JSON Lines、预算与 lifecycle 合同；architecture r23、ADR-0022、accepted design r26、test-plan r08。不得修改 Go application 文件、复制 metrics 实现或越权创建 S3.6 台账；r22/r06 仅作历史。
- **验收命令/平台操作**（workdir：仓库根目录 `.`）：
  - 历史平台验收入口（本卡已授权豁免，不执行、不记 PASS）：`promtool check rules deploy/observability/identity-alerts.yaml`
  - 历史平台验收入口（本卡已授权豁免，不执行、不记 PASS）：`promtool test rules deploy/observability/identity-alerts.test.yaml`
  - `git diff --check -- deploy/observability/identity-alerts.yaml deploy/observability/identity-alerts.test.yaml deploy/observability/identity-grafana-dashboard.json deploy/observability/identity-alertmanager-routing.yaml docs/runbooks/identity-observability.md`
  - 按 ADR-0025，本卡免用真实平台与校验工具验证，包括 Grafana 导入/查询、Alertmanager 配置与 warning/paging 路由校验。不以工具缺失阻塞本卡，不用 JSON/YAML 可解析或文件存在冒充平台通过；S3.6 仍须满足原有真实平台验证与演练要求。
  - 历史准备状态（2026-10-27）：五个目标文件已实施并完成历史静态检查与非作者 Review。本轮平台校验为授权豁免 / NOT RUN；当前范围说明的静态复核与独立评审完成前仍保持 `[-]`，不预填完成日期。真实 scrape/log/receiver/runbook 演练均未执行，属于未改变的 S3.6 范围。
- **依赖**：requires `identity-auth-S3.2`
- **并行**：独立批次；必须在 S3.2 application composition 关闭后启动，不得与 S3.2 同批闭环或把部署材料塞回 Go 开发 task。五个文件由本卡单一 owner 串行交付。
- **实施责任与 R03 历史对象**：Backend-Dev Bohr `01a1690c-8fcd-7cc3-ac1b-e089901eb47c` 已在唯一五文件边界内实施；最终 SHA256 为 `identity-alerts.yaml` `A341B8651CF3A37DCDECEF58AA061D2F5AD243103D18C59CD2FBCBF336738422`、`identity-alerts.test.yaml` `DCE00924EC9993CC5DB017D0B924242FEB87AC18F82C50DE77EBE3EF2CA7C213`、`identity-grafana-dashboard.json` `4736FAF82DFE82B28C658BACDEE0C9A8EB1EA61CCE28DC982AF6FE336C665015`、`identity-alertmanager-routing.yaml` `C2177891552E9797EDE767AA7AA420828C7F666C349951AD483CB36DE3B42EC2`、`identity-observability.md` `390963951DB1A4225E957375F3C2A2790D64D09AC224423B8CE5D433C08FB0DF`。
- **独立 Review**：非作者 Reviewer Arendt `01a16912-e9a3-7371-a4d6-9b5fda7c29ec` 的 r01（SHA256 `E9D0822E20E9A55C012F58B7A0D681F895058BA9BD6BAB1536EAAFEBBD7CF241`）为 4 MEDIUM / NOT PASS；r02（SHA256 `43C0B18850540CC405F269B26E19A032D4BBA478FFF32CEDDAC6EB0E1DACF9F5`）关闭 F001～F003、F004 保持 OPEN；r03（SHA256 `96C395621E87C4A35700A6F9C11DE68B53EFA92C254FA106D3461EA1EE25A805`）关闭 F004、无新 finding，最终 `Code Review Gate: PASS`，仅覆盖材料静态质量。
- **Lead 静态核证**：五文件 JSON/PyYAML parse、重复键/格式、job selector、9 rule cases 静态边界、runbook 合同与 scoped diff-check PASS，暂存区为空；这些证据不等于平台 validator、真实部署或 production readiness PASS。
- **范围变更与当前状态（2026-10-27）**：用户最终明确选择“S3.5：部署材料免用真实平台和校验工具验证；S3.6 不变”（ADR-0025），解除本卡平台工具与真实环境执行阻塞。promtool/platform validator、Grafana 导入/查询、Alertmanager 配置/路由校验记为授权豁免 / NOT RUN，不能记为 PASS。真实平台配置、网络、receiver 账号、角色映射、secret、自然人排班、演练窗口和付费方案仍未确认；本次不选型、不提供或伪造，其 owner 类型/索引模板与缺口继续作为 S3.6 执行前置。
- **本任务关闭条件（变更后）**：五文件完整覆盖 I-OBS-DEP-01～06 的材料前置，四类规则/rule tests、Dashboard、warning/paging routing、统一 runbook 与外部配置责任索引一致；本轮适用静态检查与非作者 Review PASS，Lead 核范围、所有权、证据和授权豁免后关闭。免除真实平台及工具校验作为本卡关闭条件；S3.6 原有真实部署、采集、告警、Dashboard、receiver 和演练要求全部保留。
- **后续验证关联**：identity-auth-S3.6；不得替代 identity-auth-S3.4 用户旅程 E2E。
- **设计/测试来源**：architecture r23；ADR-0022；accepted identity design r26；test-plan r08 §4.2/§4.2.2、I-OBS-DEP-01～06、§9.1/§10/§11/§12。r22/r06 仅作历史。

- [x] identity-auth-S3.6 执行 production observability 真实部署验证与演练 完成@2026-11-05
- **2026-11-05 关闭（经决策不验证，ADR-0036）**：候选版本 `han-s3.6-test-20261027.1` 未构建、未部署，I-OBS-DEP-01～06 未执行（NOT RUN）；按 ADR-0036 以范围决策关闭 `[ ]`→`[x]`，不声称 observability 已验证或演练完成；未验证边界（真实日志留存、scrape/网络策略、四类告警、Dashboard、receiver/升级链、值守演练）保留。

- **部署候选版本（2026-10-27 已采用）**：依用户本轮授权“部署版本采用建议值”，选定 `han-s3.6-test-20261027.1`。该标识绑定本机 `main` / 基准 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 上包含未提交改动的 80 项应用源码、迁移及 Go module 文件，源码清单 SHA256 `bada258b6fd0d3bc6ba8dfc1266f69176f4e88f7e0f519c70b887d34866796b6`，另固定五份监控材料哈希；详见[候选版本记录](../../../../.spec/wip/identity-auth/deployment-candidate-s3.6-20261027-r01.json)。不是仅用 HEAD 标识当前代码，也不沿用未经核证的既有 EXE。
- **版本选择与部署状态**：版本建议已采用；该候选尚未构建、部署或执行 S3.6 验证。构建前须重核源码与材料哈希；发生变化则另立候选修订并绑定相应检查与制品哈希，不复用旧标识冒充同一版本。平台、账号/receiver、网络、角色和演练窗口等前置仍待落实，本次不据此启动或关闭任务。

- **目标**：在已部署候选版本和真实 observability 平台上独立执行 I-OBS-DEP-01～06，形成交互、后端数据流、数据符合性三层证据；验证日志留存、真实 scrape/网络策略、四类告警、Dashboard、receiver/升级链和非作者 runbook 演练。不得用 Go 单测、in-memory registry、mock receiver、配置文件存在或 `identity-auth-S3.4` 用户旅程替代，也不得越权补造或改写 S3.5 应交付的部署材料。
- **类型**：真实部署验证
- **唯一文件所有权**：
  - `.add/project/knowledge/e2e/identity-auth/production-observability-cases.md（新增；Case、前置、三层断言与人工评审）`
  - `.add/project/knowledge/e2e/identity-auth/production-observability-runs.md（新增；Run、版本/环境、证据索引、参与角色、结果与问题单）`
- **只读输入**：已部署候选版本；S3.2 application composition 与应用 conformance 证据；S3.5 五份部署材料；真实日志后端、Prometheus-compatible scraper、Grafana、Alertmanager、生产等价网络策略与受控告警触发数据。外部证据只登记可复核索引，不把凭据、receiver secret、PII 或生产数据写入仓库。
- **验证操作与 Gate**：
  - I-OBS-DEP-01：产生合成登录内部故障与 Origin rejection canary，从真实日志平台核 JSON Lines、稳定字段查询、至少 30 天策略与最老可查询样本、敏感 denylist。
  - I-OBS-DEP-02：仅在 I-F005-19/20 独立应用 conformance 已通过后，从获准 scraper 网络访问独立 `/metrics`，核持续 scrape 与固定指标合同；从非获准网络及非 `/metrics` 路径验证阻断/不暴露。
  - I-OBS-DEP-03：受控制造四类信号，核 paging/warning 阈值、持续窗口与退出后稳定 10m resolve。
  - I-OBS-DEP-04：人工打开 `Han / Identity Auth`，对照同窗原始查询核最小面板与低基数。
  - I-OBS-DEP-05：触发 warning/paging 测试告警，核真实 receiver、聚合/静默/抑制和角色升级链；只记录真实角色与获准 receiver，不伪造账号或自然人。
  - I-OBS-DEP-06：非作者评审 runbook，并完成一次四类告警处置演练；Run 记录时间、版本、参与角色、三层结果与问题单。
  - `git diff --check -- .add/project/knowledge/e2e/identity-auth/production-observability-cases.md .add/project/knowledge/e2e/identity-auth/production-observability-runs.md`
  - 当前无 provider-specific 查询/scrape/import/route-test/告警触发命令可登记；这些命令必须由实际平台责任在执行前补入 Case 并经评审。缺任一真实命令时保持 `blocked-for-execution`，不得用 mock 或人工口头结论冒充 PASS。
- **依赖**：requires `identity-auth-S3.2`、`identity-auth-S3.5`；执行前还必须具备已部署候选版本、真实平台/网络、获准账号与 receiver、运行/identity/安全角色责任、演练窗口和证据保存位置。
- **并行**：独立验证批次；不得与 S3.2/S3.5 同批闭环。共享真实环境时必须取得环境租约并串行演练。
- **阻塞注记**：`blocked-for-execution`。部署候选版本已按上方记录采用建议值，但尚无该候选的构建、部署和运行证据；真实日志/metrics/Grafana/Alertmanager 环境、平台命令、账号/receiver、自然人排班、付费方案、演练窗口及 Test-Engineer/运行/业务/发布责任仍未落实。任何选择 provider、开通账号/费用或指定自然人均须用户/实际责任方另行决定。版本选择完成不等于“已部署候选版本”前置已满足。
- **本任务关闭条件**：Case 先经人评审；I-OBS-DEP-01～06 全部在真实环境执行并取得三层可复核证据；30 天日志、真实 scrape/网络限制、四类告警与 10m resolve、Dashboard、真实 receiver/升级链、runbook 演练均 PASS；发现的问题进入 issues 闭环；Test-Engineer 与运行/业务验收责任完成结果确认后关闭。S3.4 或任一应用/部署材料 Review PASS 均不能代替。
- **后续验证关联**：production readiness Gate；完成后才可把 observability 部署 Gate 作为 identity-auth-S3.4 前置证据之一，但不自动关闭或替代 S3.4。
- **设计/测试来源**：architecture r23；ADR-0022；accepted identity design r26；test-plan r08 §4.2.2 I-OBS-DEP-01～06、§9.1/§10～12。r22/r06 仅作历史。

- [x] identity-auth-S4.1 实现初始管理员 bootstrap 与受保护 onboarding 开始@2026-10-23 完成@2026-10-23

- **目标**：实现初始管理员 bootstrap 与受保护 onboarding。
- **类型**：开发
- **文件**：
  - `internal/identity/bootstrap.go（新增）`
  - `internal/identity/bootstrap_test.go（新增）`
  - `internal/identity/handler.go（修改）`
  - `internal/identity/service.go（修改）`
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
- **production registration 与顺序移交**：`cmd/han-server/main.go`、`cmd/han-server/main_test.go` 从已关闭 `identity-auth-S3.2` 顺序移交给本已关闭卡；S3.2 保留历史成果但不重开、不再占有或继续写入。S4.1 在唯一 `cmd/han-server` composition root 登记 exact `PUT /api/v1/users/me/password`，顺序固定为 `TraceMiddleware（最外层） → local router exact path/method → Origin/same-origin 检查 → Session 解析与认证 → X-CSRF-Token 校验 → 主体授权/Password Handler`。local router 必须在下游前完成 path/method 判定，未知 path 与错误 method 继续使用 `platform.WriteError` 的 JSON 404/405；405 的 `Allow` 与登记 method 一致，既有登录 JSON 404/405、`Allow: POST`、trace 与零错误分支下游调用合同不得回退。本卡只同步并实现 r28 已确认合同，不另作公共契约设计；关闭后 `internal/identity/handler.go` 已顺序移交 `identity-auth-S7.1`。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestBootstrap|TestInitialAdmin|TestOnboarding"`
  - `go test ./cmd/han-server -count=1`
  - 当前状态：Lead 已现场取得本卡定向测试、`gofmt`、`go mod verify`、`go vet`、`go build`、contract、race 与全仓 `go test` 全部 PASS，并确认本卡六个 owner 文件 scoped diff-check PASS、暂存区为空。非作者代码复审 `.spec/wip/identity-auth/review-code-s4.1-r02.md`（SHA256 `5AB3D51E88965A6E857048CC04FE8D7DC8C0C35C64953CBDB0015F54D60FC86E`）为 PASS，`F001`～`F003` 全部 CLOSED、无新 finding。未执行真实 PostgreSQL 故障注入、真实代理 partial response、浏览器、部署或 E2E；这些限制不外推为已验证。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S2.1`、`identity-auth-S3.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无；Q1 决策阻塞已解除，本卡已完成实现、适用检查、非作者复审与 Lead Gate，并按自身条件 task-level 关闭。
- **本任务关闭条件**：并发 bootstrap 只产生一个 admin；强制改密状态的有效账号及非强制状态的初始 `admin` 复用 `PUT /api/v1/users/me/password`，新密码至少 8 位且同时含字母和数字；其他非强制普通成员/管理员返回 `403 auth.forbidden`；当前密码或新密码策略错误返回 `422 identity.validation_failed`；当前 Session credential kind/id 陈旧、失配或失效返回 `401 auth.session_required` 且只撤销当前 Session；成功写入永久密码、完成强制改密 Gate、撤销其他 Session 并轮换当前 Session。受保护动作在 Gate 完成前全拒绝，且不增加周期轮换、密码历史、定期过期或额外复杂度；适用失败证据、局部/项目检查与独立 Review 通过后方可关闭。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S3.4
- **设计/测试来源**：identity Story S04；identity boundary r12；draft identity design r28（SHA256 `977435C8391E4A21C070C359AFBB6F143EC84E4CA3EE19CBEC7FC5FCC9C0E6B0`）；planned-not-executed test-plan r11（SHA256 `C49D87ED526D897CBF18CDE7E338F44A845E85CC1E0412A90E2A9169367B2D88`）的 I-ADM-01～03/Q1 表驱动矩阵与 router finding 追溯；设计复审 `.spec/wip/identity-auth/review-design-r28-q1-r03.md`（SHA256 `0F4D7C520F59AAF7580F9C90C66B404B9E8188E90656D3D4590B864B02DE3CC0`）PASS。

- [x] identity-auth-S4.2 实现 Recovery Journal、完整 keyring 与恢复协调 [P] 开始@2026-10-22 完成@2026-10-22

- **目标**：实现 Recovery Journal、完整 keyring 与恢复协调。
- **类型**：开发（共享恢复基础）
- **B3 认领/责任**：唯一 owner 仍为 Backend-Dev Agent `01a16123-8647-7082-abab-bec6da4fb38f`（Bacon）。architecture r19、identity design r13 与 ADR-0019 修订已正式落档；设计 Reviewer Mendel `01a16132-31b6-7363-b1f4-96939990a115` 对四份设计成果的 `IA-S4.2-CR-002` 合同复审 PASS、无 finding。原作者 Bacon 与原代码 Reviewer Dirac `01a16159-10ef-7453-a83b-b7688c3b6319` 恢复时均因 `401 TOKENOPS_GATEWAY_UNAUTHORIZED` 无法继续；用户于 2026-10-22 明确授权替代实施。Backend-Dev Halley `01a16251-3c0c-7a02-b303-98d8f1febbe8` 在既有五文件内完成返工，非作者 Reviewer Pasteur `01a16234-08d6-7270-8f0a-f1db30f18e1d` 关闭 `IA-S4.2-CR-002-F001`～`F004` 并给出代码复审 PASS、无新 finding；Lead 完成当前工作树、文件哈希、范围与验证复核后关闭本卡。
- **文件所有权**：Bacon 在 B3 内唯一拥有下列十三个文件。原六文件保留授权，本次仅新增七文件；`migrations/0005_recovery_runs.sql` 排在已授权给 S5.1 的 `migrations/0004_idempotency.sql` 之后，不得修改任何既有 migration。明确禁止修改 `go.mod`、`go.sum`、`migrations/0001_shared.sql`～`migrations/0004_idempotency.sql`、现有 identity repository/model、`cmd/`、配置及其他文件；若后续确需第十四个文件，必须停止并重新申请。其余 B3 卡只读；发现未知写者或必须修改清单外文件时停止受影响写入并交 Lead。
- **文件**：
  - `internal/recovery/journal.go（原授权；当前共享树已存在）`
  - `internal/recovery/keyring.go（原授权；当前共享树已存在）`
  - `internal/recovery/coordinator.go（原授权；当前共享树已存在）`
  - `internal/recovery/replay_status_v1.go（原授权；当前共享树已存在）`
  - `internal/recovery/journal_test.go（原授权；当前共享树已存在）`
  - `internal/recovery/replay_status_v1_test.go（原授权；当前共享树已存在）`
  - `migrations/0005_recovery_runs.sql（本次扩权新增；顺序在 0004 之后）`
  - `internal/identity/recovery_run_store.go（本次扩权新增）`
  - `internal/identity/recovery_run_store_integration_test.go（本次扩权新增）`
  - `internal/recovery/filelock_flock.go（本次扩权新增）`
  - `internal/recovery/filelock_unsupported.go（本次扩权新增）`
  - `internal/recovery/filelock_flock_test.go（本次扩权新增）`
  - `internal/recovery/journal_windows_test.go（本次扩权新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/recovery -count=1`
  - `go test -race ./internal/recovery`
  - `go test -tags=integration ./internal/identity -run "TestRecoveryRunStore" -count=1`
  - `go test -race -tags=integration ./internal/identity -run "TestRecoveryRunStore" -count=1`
  - Windows 主机：`go test ./internal/recovery -run "Test.*Windows.*Unsupported" -count=1`
  - GOOS cross-build（PowerShell）：`& { $old=$env:GOOS; try { $env:GOOS='windows'; go build ./internal/recovery } finally { $env:GOOS=$old } }`
  - GOOS cross-build（PowerShell）：`& { $old=$env:GOOS; try { $env:GOOS='linux'; go build ./internal/recovery } finally { $env:GOOS=$old } }`
  - 真实目标 POSIX 环境：`go test ./internal/recovery -count=1`
  - `go vet ./internal/recovery ./internal/identity`
  - `go build ./internal/recovery ./internal/identity`
  - `gofmt -l .`
  - `go mod verify`
  - `go vet ./...`
  - `go build ./...`
  - `go test -tags=contract ./...`
  - `go test -race ./...`
  - `go test ./...`
  - 当前状态：Lead 于 2026-10-22 实际运行并确认局部/全仓 test、race、contract、PostgreSQL 18 integration 与 integration-race、Windows unsupported、Windows/Linux cross-build、`go mod verify`、vet、build、gofmt 和 diff-check 均 PASS。真实目标 POSIX 的 owner-only、原子替换、file/directory fsync 与 OS 锁行为尚未实测，保留为明确限制，不以 Windows、cross-build 或纯单元结果冒充目标 POSIX 通过。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **关闭证据与限制**：tasks/Plan 独立复审已 PASS；Pasteur 对 Halley 的五文件返工逐项关闭 `IA-S4.2-CR-002-F001`～`F004`，十三文件代码复审 Gate PASS、无新 finding。Lead 核十三文件所有权与哈希，确认仅授权五文件变化、其余八文件不漂移，并取得 PostgreSQL 18 integration/integration-race 及全部约定 Go 检查 PASS。Pasteur 已完成 B3 独立批次整体 Review，开始/结束固定哈希一致、无阻断 finding，正式结论 `B3 G4-batch Gate: PASS`，B3 批次关闭。真实目标 POSIX 的 owner-only、原子替换、file/directory fsync 与 OS 锁仍未实测；生产 CAS Anchor 由 `identity-auth-S4.5` 承接，生产 key-retirement capability 闭环由 `identity-auth-S4.6` 承接，对应任务及真实验证完成前不得声称生产恢复或生产 key retirement 已就绪。用户于 2026-10-22 对 B4 三卡的启动授权不包含 S4.5 或 S4.6。
- **本任务关闭条件**：先以失败用例覆盖 retained key 重放、缺 key、部分写、链损坏、namespace 混用、Unavailable、坏 Ready、并发状态竞争、跨进程锁竞争，以及 key 退役两个并列前置中任一不满足时必须失败：无 begun 未消费 run，且该 key 的所有记录均已退出批准的保留/归档可验证范围。verify-only key 仍可使用 Begin 时绑定的 key 终结既有 run 的唯一 consumed 转换，但不得 Begin 新 run。`0005_recovery_runs.sql` 在不修改 `0001`～`0004` 的前提下建立 `recovery_runs` PostgreSQL schema；identity-owned Store 以短事务行锁、条件状态转换和并发 CAS 维护 run，Coordinator 只接收 `run_id`，`MarkConsumed` 是唯一自动收敛窗口。`backup_ref` 必须精确绑定 Begin、Journal、HMAC、Store 与 Article evidence；begun 时绑定的 key 在轮换后仍可完成 consumed。key 退役必须同时满足上述两个并列前置，且 capability 不能绕过任一前置。本卡十三文件只定义 opaque `KeyRetirementCapability`、精确 `key_id/key_version` 请求绑定、验证/单次消费 port、fail-closed 结果与 deterministic fake；测试须证明 capability 缺失、空值、拒绝、verifier 不可用、错误 key 绑定、复制/重放或结果不可判定均禁止 retire/delete。本卡不得解析、签发、日志记录或持久化 capability，不得提供生产 verifier/config/startup wiring、审计后端、生产 mint、新依赖、第十四文件或 `cmd/`/配置改动；fake 不得作为生产 verifier、授权或 evidence。持久化时间统一为 UTC 微秒。Windows unsupported 路径必须零外部状态变更；遗留 `.lock` 文件不得阻塞，跨进程互斥须由 OS handle 保证且在 handle close/进程退出时释放。目标支持 POSIX 上须验证 owner-only、原子替换、fsync 与 OS 锁；真实目标 POSIX 环境未验证时只记录限制，不得宣称通过。本卡同时只交付 provider-neutral `JournalHeadAnchor` port/协议、RecoveryJournal 调用边界与 deterministic fake，不选择或实现生产 Anchor provider。局部、PostgreSQL integration、race、GOOS cross-build、适用全仓检查及 Dirac 非作者独立 Review 均完成后方可关闭；即使本卡关闭，在 `identity-auth-S4.5` / `identity-auth-S4.6` 各自完成前，对应生产 Gate 仍关闭。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S4.4、identity-auth-S4.5、identity-auth-S4.6、article-management-S4.2
- **设计/测试来源**：identity Story S04；已正式落档并经 Mendel 合同复审 PASS 的 architecture r19、identity design r13、ADR-0018/0019 正文与决策索引；test I-REC-02～10；ADR-0020；用户于 2026-10-22 确认的十三文件扩权、F004 双前置、F005 方案 A 与 `IA-S4.2-CR-002` 方案 A-补充。下一 Gate 为 tasks/Plan 独立复审，随后由 Dirac 对十三文件执行 `IA-S4.2-CR-002` 代码复审。

- [x] identity-auth-S4.3 准备恢复运行输入、合成备份与 keyring 样本 [P] 开始@2026-10-22 完成@2026-10-22

- **目标**：准备恢复运行输入、合成备份与 keyring 样本。
- **类型**：运行输入准备
- **文件**：
  - `docs/runbooks/database-recovery.md（新增）`
  - `testdata/recovery/README.md（新增）`
  - `testdata/recovery/keyring-fixtures.json（新增）`
  - `testdata/recovery/backup-manifest.json（新增）`
  - `test/integration/recovery/fixtures_test.go（新增；用户于 2026-10-22 已授权 Volta 作为第五个唯一写文件）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/recovery -run "TestRecoveryFixtures" -count=1`
  - 当前状态：五文件固定 SHA-256 依次为 `73D3E03B2C5CCD0922142413E4CE742973DCB5F370DC3EF0A756148C4340F69F`、`38C3171B96BECBB09384400D695F8F3F85A5D7F042C031D7D34CEF33E47BB744`、`CB398B9E736694FEC77AA21BB08E056933591774BB3842046E63D710DD9212F0`、`2F267BD497FC90D7B6D1B28F36F8E7B8B253F8E36153E5F50400BB2AC6DF856E`、`0CD7471AD1574D05465CA3F29C8B967B631F520A2F3BA94E91BE5E346A485ACB`；Lead 普通、`-trimpath`、race、vet、build、gofmt 与 scoped diff-check 均 PASS。原 Reviewer Mill 将 `IA-S4.3-NEW-REV-F001`～`F003` 全部 CLOSED、无新增 HIGH/MEDIUM finding，结论 `G4-task: PASS`。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **B4 认领/所有权**：Backend-Dev Volta `01a1628f-31fd-7de0-9ae1-9224fdb2a760` 于 2026-10-22 认领并启动本卡；用户于 2026-10-22 已授权 `test/integration/recovery/fixtures_test.go` 作为第五个唯一写文件，原四个已评审成果本轮只读。该写集与 Popper 的 S3.2 F001 写集零交叉。本授权禁止修改原四成果、其他测试或代码、B5/B9C/B9D、S4.5/S4.6、设计/ADR/Manifest、依赖与 Git 集成。
- **关闭证据与限制**：五文件哈希与 Lead 检查、Mill 复审对象一致，本卡 task-level Gate PASS。成果仅覆盖合成 runbook/fixture 及其 deterministic 自验证；RPO/RTO、批量容量、key/Journal/tombstone retention 与运行责任仍未确认，不构成真实恢复、真实 E2E、B5、发布或生产就绪证据。
- **本任务关闭条件**：原四个成果只含合成数据并记录 backup_ref、run、keyring、清理责任与恢复步骤；RPO/RTO/保留值 unknown 时不填数值，真实演练仍保持阻塞。依据用户于 2026-10-22 已授予 Volta 的精确扩权，仅在第五文件实现 deterministic `TestRecoveryFixtures`：解析 `keyring-fixtures.json` 与 `backup-manifest.json`，核 synthetic-only/生产禁用声明、唯一 active v3 与 v2/v1 verify-only、三组 HMAC、manifest 的 backup_ref/run/key/version/fingerprint 与 UTC 微秒格式，并核 unknown Gate、maintenance 顺序、Anchor 只读以及禁止 restore/reset/bootstrap/rollback 的约束；上述验收命令 PASS 且非作者独立 Review 通过后方可关闭。
- **后续验证关联**：identity-auth-S4.4、identity-auth-S4.5、identity-auth-S4.6、article-management-S4.4
- **设计/测试来源**：identity test §7 数据环境；Q9/Q10/Q11

- [x] identity-auth-S4.4 验证 Recovery Journal、keyring 与联合恢复 [P] 完成@2026-11-03

- **目标**：验证 Recovery Journal、keyring 与联合恢复。
- **类型**：专项韧性测试
- **文件**：
  - `test/integration/recovery/identity_recovery_test.go（新增）`
  - `test/integration/recovery/journal_faults_test.go（新增）`
  - `test/integration/recovery/joint_readiness_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/recovery`
  - 当前状态：专用命令 `go test -tags=integration ./test/integration/identity -count=1`（真实 PG18.6）与 `-race` 均 PASS；Manifest 固定七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）已按序全部 PASS。二者互不替代。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`、`identity-auth-S4.3`、`identity-auth-S4.5`、`identity-auth-S4.6`、`identity-auth-S9.2`、`article-management-S4.2`、`article-management-S4.4`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **关闭证据与限制**：2026-11-03 依 ADR-0028 项目级验证授权与用户同轮确认，以范围豁免（`waived-for-test-scope`，test scope only）关闭本卡；该关闭**不满足** requires 中的 `identity-auth-S4.5`、`identity-auth-S4.6`、`identity-auth-S9.2`、`article-management-S4.2`、`article-management-S4.4`，不表示这些前置已完成，也不构成安全豁免、生产恢复解锁或 S4.5／S4.6 完成证据。交付与证据：`test/integration/recovery/identity_recovery_test.go`（16,345 B / `AC5EB4B520EB9379D12E8923E3C33DCF7303A4ADD7DB72D386C1386FC3E2AD47`）、`test/integration/recovery/journal_faults_test.go`（11,128 B / `61BC9925A60BEC5CBA31597D02F654C171B089F4C3FF5509E1DACD8B01FA7558`）、`test/integration/recovery/joint_readiness_test.go`（22,162 B / `885766D80D96EC19048316A0E3EA2D0A444BD644106F3581DADC06B56B883C3F`）；报告 `.spec/wip/identity-auth/evidence-s4.4/report-r04.md`（27,633 B / `E9E850BC7AC7A7C1BF0EB54478918B478944B5D992155AF5D3B85E0438C7107B`）与纯文字勘误 `.spec/wip/identity-auth/evidence-s4.4/report-r04-errata.md`（4,294 B / `9DF5C8C53ADFACCB44EB5873A463AA87E0A4E7597566E98617767428C5E75CB2`）；非作者独立复审 `.spec/wip/identity-auth/review-code-s4.4-r04.md`（22,823 B / `B71CEC3710A647E93388EE22CA871E305AA9D21C83BE40A04FB9922BB0D09DD8`）结论为 **PASS WITH CONDITIONS**（0 BLOCKER／0 HIGH／0 MEDIUM／1 LOW／4 INFO），`IA-S4.4-R04-F001`（LOW，`report-r04.md` 误写替身含 `Store`）与四项 INFO 均已在该勘误文件登记，不改变 r04 等级。Lead 独立复跑：`go test -tags=integration ./test/integration/recovery -count=1` 三次 `ok`（33.422s／36.943s／33.445s），r04 复审内另跑分组 16/16 PASS（8.182s）、整包 41.276s、`-race` 40.111s，`gofmt -l test/integration/recovery` 无输出，`go vet ./...` 与 `go build ./...` exit 0；16 子测试口径为 11 实质断言 + 5 部分/未取证。**未验证边界（NOT RUN，不得写成通过）**：目标 POSIX 平台全部运行行为（本 Host 无 Linux Go 工具链且未授权安装系统包；`/home/jianghua/han-posix-r01` 仅为副本，PATH 内只有 Windows `go.exe`）、真实持久卷上的崩溃/损坏/跨版本重放、真实生产存储上的 kept_deleted 与双 namespace、生产 Anchor provider、S4.6 retirement capability 闭环、`GOOS=ios` 独立编译证据（需 cgo 与 Apple 工具链）、真实独立故障域（用户于 2026-11-03 确认本项目不构造域外持久、该独立故障域不需要，故登记为 `unknown`／`not required for test scope`，非 PASS）。**残余风险**：`IA-S4.4-R04-F004`（verify-only 负例按 `keyring-fixtures.json` key 下标取 key，fixture 顺序变化会使负例语义漂移）与 `IA-S4.4-R04-F005`（`MarkConsumed` 顺序与软件 journal 指纹均为 `softwareJournal` 替身自证，不等价真实文件 journal 或生产存储），作为已知限制保留。本卡不得被引用为真实恢复、真实 E2E、B5/B9 通过、S4.5／S4.6 完成或 production readiness 证据。
- **本任务关闭条件**：真实备份/持久卷环境中覆盖崩溃、损坏、跨 key 版本重放、kept_deleted、双 namespace 与 MarkConsumed 顺序；生产 Anchor 须来自已完成的 S4.5，生产 key retirement 仅在 S4.6 的真实双前置 evidence 核验与 capability 原子单次消费成功后开放，缺失/错误绑定/复制重放/不可判定均保持禁用。未有环境/RPO/RTO 或任一生产前置未验证时保持未关闭；独立 Review 通过。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.4
- **设计/测试来源**：identity test Recovery Journal/keyring 与联合恢复专项；I-REC-01～10

- [x] identity-auth-S4.5 选择并交付生产 CAS JournalHeadAnchor provider 与独立故障域验证 [P] 完成@2026-11-05
- **2026-11-05 关闭（经决策不纳入本版，ADR-0037）**：生产 CAS anchor provider 未选择、未实现，独立故障域未验证（NOT RUN）；按 ADR-0037 以范围决策关闭 `[ ]`→`[x]`，不声称 provider 已交付；ADR-0030 门禁与生产 enter/install 保持 closed。

- **目标**：在架构明确生产 provider 后，选择并实现生产 CAS `JournalHeadAnchor` provider，完成配置、启动装配及真实独立故障域验证，使生产恢复 Gate 可被独立判定。
- **类型**：开发（生产恢复基础与真实验证）
- **文件**：当前仍不授予任何文件写权。provider 与部署形态已有 ADR-0030 `adr-0030-r12` 登记：r05 四项结论 (a)–(d) 给出目标形态的约束性要求（独立 PostgreSQL 实例承载单一 append-only 头事件表、专用数据库角色与 fd 注入凭据、域外单调见证、部署方运维责任），见 `framework.md` ADR-0030 段；约束 9(b) 启动前置 (i)(ii) 另登记 Lead 建议值（PostgreSQL 社区官方发行包 16 大版本线、自建独立实例；受支持清单由部署方具名 owner 维护）。仍未确定：启动前置 (iii) 许可与具名 owner、(iv) 来源地址范围、(v) 见证 owner，以及精确文件合同、新依赖、启动 wiring 与真实独立故障域环境；须由用户/组织具名并另行授权。在此之前不登记虚构路径、不授予任何文件写权，也不得复用 S4.2 的十三文件授权。
- **验收命令**（workdir：仓库根目录 `.`）：provider 包定向 test、provider integration/CAS 并发与故障注入、配置和启动装配测试、适用 race/vet/build、Manifest 全仓检查，以及真实独立故障域演练命令均待 provider 与部署拓扑正式落档后精确化；未运行不得写成通过。真实验证必须证明独立故障域中的 CAS 单调性、并发冲突、不可达/超时 fail closed、重启后锚点延续，以及主 Journal/业务数据库故障不会同步抹除锚点；deterministic fake 不得替代该证据。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S4.2`、`identity-auth-S4.3`
- **并行**：[P]；仅在设计、文件合同、环境与责任人齐备后评估。当前未认领、未启动；B4 的三卡启动授权不包含本任务，B9C 未获授权时不得以本任务登记为由提前实施。
- **阻塞注记**：provider 目标形态、技术栈方向与部署拓扑已有 ADR-0030 `adr-0030-r12` 登记（(a)–(d) 约束性要求 + (i)(ii) 建议值），不再是空白 unknown；仍阻塞于启动前置 (iii) 许可与具名 owner、(iv) 来源地址范围、(v) 见证 owner 三者未具名/未取证（整组 (i)–(v) 因而未满足），以及精确文件合同、新依赖、启动 wiring、真实独立故障域环境与实施/验证责任人未定。上述未完成前，本任务保持阻塞，不得启动。
- **本任务关闭条件**：先取得 provider 与部署拓扑的正式设计/ADR、精确写集和新增依赖授权；再按失败证据→最小实现→局部与集成测试→真实独立故障域演练→非作者独立 Review 闭环。生产 provider 必须实现 S4.2 的 provider-neutral `JournalHeadAnchor` CAS 协议并通过真实并发与故障证据；配置缺失、provider 不可达、CAS 冲突或独立故障域不可验证时 fail closed。完成前生产恢复 Gate 始终关闭，不得声称 S4.2、B3 或系统生产就绪。任何未获确认的 provider 选型、新依赖、文件清单或部署假设均触发停止并回到架构/授权活动。
- **后续验证关联**：identity-auth-S4.4、identity-auth-S3.4
- **设计/测试来源**：identity Story S04；architecture r19；identity design r13；ADR-0018/0019；test I-REC-02～10；用户确认的 `IA-S4.2-DR-F005` 方案 A。r19/r13 未把 key-retirement capability 生产闭环并入本任务。

- [x] identity-auth-S4.6 交付生产 key-retirement capability 证据、签发与单次消费闭环 [P] 完成@2026-11-05
- **2026-11-05 关闭（经决策不纳入本版，ADR-0037）**：KeyRetirementCapability 的 evidence 来源、签发/消费与审计接线未实现（NOT RUN）；按 ADR-0037 关闭，不声称 key-retirement 可用；生产 key retirement/delete 继续禁用。

- **目标**：在后续架构/安全选择明确后，交付真实 retention/archive exit evidence 来源及核验、绑定精确 `key_id/key_version` 的 capability 签发、生产 verifier/config/startup wiring、单次消费、防复制/重放、最小化审计、测试与真实环境验证；完成并独立验证前，生产 key retirement/delete 持续禁用。
- **类型**：开发（生产恢复安全能力与真实验证）
- **文件所有权**：当前为受阻 unknown，不授予任何文件写权。生产 evidence 来源、issuer/verifier、单次消费存储、审计接缝、配置键、启动装配、部署资产、新依赖与精确文件合同必须先由 architecture/security 决定并由用户授权；不得复用或扩大 S4.2 十三文件、S4.5 Anchor provider 文件合同，也不得预设存储产品、SDK、地址、凭据或 `cmd/config` 路径。若一个精确合同不能同时隔离 S4.2/S4.5 所有权并形成开发自测与真实验证闭环，停止并返回 Task-Planner/架构活动，不拆出第二任务补洞。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `gofmt -l .`
  - `go mod verify`
  - `go vet ./...`
  - `go build ./...`
  - `go test -tags=integration ./...`
  - `go test -race ./...`
  - `go test ./...`
  - provider/issuer/verifier 定向测试、并发单次消费与故障注入、配置/启动测试及真实环境演练命令必须在 provider、部署拓扑和精确文件合同落档后补成可执行命令；当前不得编造包路径、SDK、endpoint、凭据或通过结果。
  - 当前状态：仅登记任务，未认领、未启动、未运行命令；现役 Manifest 七项项目检查已登记，但不包含本任务的 provider-specific 验证，以上 Go 验收基线与后续生产环境验证不得互相替代。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.4`、`identity-auth-S4.2`、`identity-auth-S4.3`
- **并行**：[P]；语义上不依赖 S4.5，但两任务都可能触及尚未设计的生产配置/启动装配，批次先串行规避未知写集冲突。当前未认领、未启动；B4 的三卡启动授权不包含本任务，B9C/B9D 未获授权时不得提前实施。
- **阻塞注记**：真实 retention/archive exit evidence owner 与可验证范围、retained key 上限/批准 retention、issuer/verifier 信任边界、单次消费存储、审计最小字段、provider/技术栈、配置/凭据、部署拓扑、真实环境及实施/验证责任人均 unknown。architecture/security 决定、精确文件/依赖授权与环境租约未完成前保持阻塞，Task-Planner 不替用户选型。
- **本任务关闭条件**：两项静态前置必须分别从真实来源验证：不存在 begun 未消费 run，且目标 key 的全部记录已退出批准的 retention/archive 可验证范围；任一不成立时都拒绝退役，capability 不能绕过。只有 evidence 核验成功后才能签发绑定精确 `key_id/key_version` 与本次 retire/delete 请求的 opaque capability；生产 verifier 必须原子验证并单次消费，复制、重放、错误 key/version、过期/撤销、不可用或不可判定均 fail closed。capability 不得进入日志、审计 payload、Journal、RecoveryRun 或其他持久状态；审计只记录最小化结果/引用，不记录 capability。配置缺失、wiring 未启用、真实环境无法证明或 S4.2 port 不匹配时保持 production retirement disabled。按失败证据→最小实现→定向/集成/race/vet/build→真实环境演练→非作者独立 Review 闭环；真实验证必须使用批准的 evidence 来源和独立环境，deterministic fake、调用方布尔声明或仅全仓测试不得替代。完成前不得声称生产 key retirement/delete 或 B3/S4.2 生产就绪。
- **后续验证关联**：identity-auth-S4.4、identity-auth-S3.4
- **设计/测试来源**：identity Story S04；architecture r19；identity design r13；ADR-0019（引用 11）；test I-REC-02～10；用户确认的 `IA-S4.2-CR-002` 方案 A-补充；Mendel 对四份设计文件的合同复审 PASS、无 finding。

- [x] identity-auth-S5.1 建立幂等、临时凭据与强制 TLS Mailer 共享内核 [P] 开始@2026-10-22 完成@2026-10-22

- **目标**：建立幂等、临时凭据与强制 TLS Mailer 共享内核。
- **类型**：开发（共享凭据基础）
- **B3 认领/责任**：Backend-Dev Agent `01a16158-c2dd-72e0-bf35-0580dc25e149`（Tesla）完成八文件实施及两轮返工；非作者 Reviewer Agent `01a16159-10ef-7453-a83b-b7688c3b6319`（Dirac）关闭 retained-key 原子验证、安全 snapshot allowlist、临时凭据七天期限与 Argon2id 三项 MEDIUM finding，最终复审 PASS。Lead 核八文件哈希无漂移，PostgreSQL 18 integration、定向/范围/全仓 test、integration-race、范围/全仓 race、vet、build、gofmt 与 diff-check 均 PASS，并于 2026-10-22 关闭本任务。
- **文件所有权**：Tesla 对下列八文件的 B3 S5.1 交接已完成；`migrations/0001_shared.sql`、`migrations/0002_identity.sql`、`migrations/0003_article.sql` 未修改。后续消费者只按各自任务获得写权限，不得借本卡继续修改这些文件。
- **文件**：
  - `migrations/0004_idempotency.sql（新增）`
  - `internal/idempotency/store.go（新增）`
  - `internal/idempotency/store_test.go（新增）`
  - `internal/mailer/mailer.go（新增）`
  - `internal/mailer/smtp.go（新增）`
  - `internal/mailer/smtp_test.go（新增）`
  - `internal/identity/temporary_credential.go（新增）`
  - `internal/identity/temporary_credential_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/idempotency ./internal/mailer ./internal/identity -run "TestIdempotency|TestTemporaryCredential|TestSMTP"`
  - `go test -tags=integration ./internal/idempotency`
  - 当前状态：PostgreSQL 18 migration/integration、定向/范围/全仓 test、integration-race、范围/全仓 race、vet、build、gofmt、diff-check、Lead 复核及 Dirac 复审均 PASS；固定八文件无漂移，本任务已关闭。未执行真实外部 SMTP、生产接线或业务 E2E，这些由后续独立任务承接。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`identity-auth-S1.2`、`identity-auth-S2.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无；迁移范围缺口已由用户授权新增 `migrations/0004_idempotency.sql` 解决，三项 Reviewer finding 已由原作者返工并经原 Reviewer 复审关闭。
- **本任务关闭条件**：先以失败用例覆盖同键异摘要、uncertain/recovery_review_required、旧 generation、TLS 降级/错证书；`0004_idempotency.sql` 仅建立本卡所需 `idempotency_records` 及约束/索引，并由现役 migration runner 在 PostgreSQL 18 验证 `0001 → 0002 → 0003 → 0004` 顺序；状态机、purpose 与安全快照符合合同；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S5.2、identity-auth-S10.1、identity-auth-S11.1、article-management-S7.1
- **设计/测试来源**：identity K3；design §3.3.1/§3.4.2/4；test I-CRED

- [x] identity-auth-S5.2 实现创建其他管理员与双 Gate 开始@2026-10-23 完成@2026-10-23

- **目标**：在同一卡内完成 I-S52-01～09、关闭独立评审链的全部 finding，并按已采用 D001 方向交付创建其他管理员与双 Gate 闭环；不新建重复 task。
- **类型**：开发
- **R 对账 gap-map（Story S05 的合同切片，不是新 task）**：

  | Story / 场景 | 四标签 | 当前证据与缺口 |
  | --- | --- | --- |
  | S05 / I-S52-01～03：`0007`、strict legacy/v1 与 canonical snapshots | 已建成 | 前向 `0007`、严格 legacy/v1 双读、canonical snapshots 与 PostgreSQL 18 fixture 均已完成；idempotency integration、普通包、race 及最终四包/全仓 Gate PASS。 |
  | S05 / I-S52-04～05：当前 trace、D001 授权后稳定终态优先 | 已建成 | 当前请求 trace 物化、授权后稳定终态优先与相关 identity/handler 回归均已完成；SMTP/identity 定向、identity 普通与 race Gate PASS。 |
  | S05 / I-S52-06～07：SMTP operation budget、较早 deadline 与 cancel | 已建成 | 必填无默认的 operation budget、较早 deadline、及时 cancel 与资源释放均已完成；Mailer 定向/普通/race 与四包普通/race/vet Gate PASS。 |
  | S05 / I-S52-08：三条 production route、共享实例与 lifecycle | 需改（返工，已完成） | 唯一 production composition 已登记三条入口并复用共享实例；factory nil、失败逆序 unwind、正常关闭与路由矩阵均已完成并通过最终 Gate。 |
  | S05 / I-S52-09：规范 UTC 整秒 notice effective-at | 待做（已完成） | canonical effective-at loader → reader → Handler 值流、启动拒绝/零调用矩阵与 HTTP fixture 已完成并通过最终 Gate。 |

  初始四标签分布为已建成 3 / 待做 1 / 需改 1 / 废弃 0；I-S52-01～09 现已在本 task 内全部闭环，未新增 task、未修改 requires/DAG。
- **独立评审返工链**：原作者 Backend-Dev Volta `01a1692f-c228-7fd2-881f-953e638053f3` 完成返工与整卡回归；原独立非作者 Reviewer Faraday `01a16940-6120-7021-bd06-eb0089956486` 的最终报告 `.spec/wip/identity-auth/review-code-s5.2-r04.md`（blob `e473153c726ee499254bd90c7f42df1cc82a8a97`，SHA256 `B9C1598C2B8CAC42F055E355A6D95F28FBA5B8F459FB94495D72674F9945E075`）为 `Code Review Gate: PASS`；`R03-F001`、`R01-F002` CLOSED，无新增 finding，其他历史项保持 CLOSED。
- **固定返工写集**：
  - F001/D001：`internal/identity/administrator_create.go（修改）`
  - F001/D001：`internal/identity/administrator_create_test.go（修改）`
  - F001：`migrations/0007_idempotency_response_snapshot_v1.sql（新增）`
  - F001：`internal/idempotency/store.go（修改）`
  - F001：`internal/idempotency/store_test.go（修改）`
  - F001：`internal/idempotency/store_integration_test.go（现有）`
  - F001/F003：`internal/identity/handler.go（修改）`
  - F001/F003：`internal/identity/handler_test.go（新增）`
  - F002/F003：`cmd/han-server/main.go（修改）`
  - F002/F003：`cmd/han-server/main_test.go（修改）`
- **条件性写集**：`internal/mailer/smtp.go`、`internal/mailer/smtp_test.go` 默认只读回归；只有现役 adapter 的确定性失败证据证明 deadline/cancel 不及时或连接、timer、goroutine、临时资源未释放时，才允许原作者在这两文件内做最小修订。若证据不成立，不得为“顺手加强”产生 diff。
- **只读固定对象**：`internal/identity/service.go` 仅作消费者核对；`migrations/0004_idempotency.sql`、其他既有 migration、`internal/platform` 与未列文件不得修改。若等价 PostgreSQL 18 测试落点、公共 API、列结构、依赖或稳定错误语义要求扩大范围，立即停止并回 Task-Planner/Lead。
- **F001/D001 验收合同**：新增前向 `0007`，不修改 `0004`、不改列、不重写旧行；数据库 CHECK 与 Go codec 严格互斥双读 legacy/v1，管理员新完成记录只写 `identity-administrator-create-response/v1`。v1 外层严格为 `schema_version,http_status,result_code,body`；success body 严格为 `id,username,role,status,must_change_password,must_acknowledge_privacy,temporary_credential_expires_at,delivery_status`；failed body 严格为 `code,message,field_errors`。canonical `identity.identifier_conflict` 固定 `409 / identity.identifier_conflict / 身份标识已被使用 / field_errors=[]`，不得披露 username、email、phone 或其他冲突字段；extra/missing/null/type drift、混合格式、未知版本、敏感字段与 `trace_id` 必须由 Go 与 DB 同规则拒绝。稳定快照不保存 `trace_id`，首次与重放由 Handler 使用当前请求 trace 物化 Header/body/context/log，稳定 status/code/message/field_errors 不经当前 registry 重算。每次请求仍先确认当前 principal 是 active initial admin；授权通过后，已有同键同摘要终态优先返回原结果，不得被后续 ProcessingGate、声明值/current notice、密码策略、改密、隐私确认或停用变化改写，且不新增邮件、用户、凭据、幂等完成或审计副作用。
- **F002 配置与接线合同**：F002 不再存在设计阻塞。`cmd/han-server/main.go` / `main_test.go` 解析并校验必填、无默认的 `HAN_SMTP_OPERATION_BUDGET`；部署基线 `10s`，合法闭区间 `1s..60s`，缺失、空白、非法或越界必须在 DB connect/Ping/migrate、composition、listener 与 route registration 前失败且零副作用。唯一 production composition 把同一已校验 budget 注入管理员 coordinator；coordinator 使用 request deadline 与 operation budget 的较早者并及时 cancel，不得在业务代码或 adapter 硬编码 `10s`。现役 Mailer 两文件仅按上列条件性证据决定是否修改。
- **F003 production composition/router**：只允许在唯一 production composition root `cmd/han-server/main.go` / `main_test.go` 构造依赖并登记 `GET /api/v1/admin/administrator-information-provision-notices/current`、`POST /api/v1/admin/administrators`、`POST /api/v1/users/me/privacy-acknowledgements`；不得建立第二套 router/composition。三路由必须从 `newProductionBusinessHandler` 真实可达并复用同一 notice reader、administrator coordinator、Mailer、Session/authorizer 及其他有状态实例。保持 Trace 最外层、local router 先判 exact path/method，再按 Origin → Session → CSRF → Principal/授权 → Gate/Handler；覆盖正确 method/path、稳定 JSON 404/405、route-specific `Allow`、全部顺序与失败零业务调用，以及 factory nil/构造失败的逆序 unwind 和正常关闭资源释放。
- **I-S52-09 effective-at 合同**：新增必填配置 `HAN_ADMINISTRATOR_INFORMATION_NOTICE_EFFECTIVE_AT`，正例原始值固定为 `2026-10-24T00:00:00Z`，值流必须是 config → RFC3339 parse + canonical 字节检查 → UTC 整秒 `time.Time` → `StaticAdministratorInformationNoticeReader` → Handler `time.RFC3339`，HTTP fixture 与原始值逐字节一致。`.1Z`、`.123Z`、`.000Z`、任意小数秒、非 `Z`、非零 offset、非规范等价值、零值与解析失败必须在 DB parse/connect/Ping/migrate、prepare/factory、listener、route registration、reader 与 Handler 前 fail closed，各调用数为 0，错误去敏；Handler 不截断、不舍入、不补 `Z`、不修正输入且不引入 `time.RFC3339Nano`。
- **串行与共享文件所有权**：十个固定写文件及条件性 Mailer 接缝的本卡增量已完成并经 Lead 核证；S5.2 独占窗口随 task-level closure 释放。`article-management-S4.1` 与后继 `article-management-S4.6` 也已分别于 2026-10-23、2026-10-24 task-level CLOSED，B6A 与 B6A-W 均关闭并释放共享文件独占窗口。B6B/S8.1 已于 2026-10-24 启动并关闭；B7 六卡已于 2026-10-25 完成认领准备，S3.6、S2.2 与 B8 以后任务仍未启动，后续写入须按原 task、依赖与授权重新认领。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./cmd/han-server -run "Test.*(AdministratorInformationNotice|EffectiveAt|Notice|Administrator|Privacy|Router|Production|Composition|StartupConfig|Startup)" -count=1`
  - `go test ./internal/identity -run "Test.*(AdministratorInformationNotice|EffectiveAt|CurrentAdministratorInformationNotice|CreateAdministrator|AdministratorOnboarding|ResponseMaterializer|Replay|Trace)" -count=1`
  - `go test ./internal/idempotency -count=1`
  - PostgreSQL 18：安全注入 `HAN_TEST_POSTGRES_DSN` 后执行 `go test -tags=integration ./internal/idempotency -count=1`；须记录 `server_version_num` 为 major 18，并覆盖 `0004 → 0007`、旧行不重写、legacy/v1、三个 canonical fixture、identifier-conflict 空数组拒绝矩阵、通用拒绝矩阵与首条 v1 前后回退 Gate。
  - `go test -race ./internal/idempotency -count=1`
  - `go test ./internal/mailer -run "TestSMTPMailer" -count=1`
  - `go test ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1`
  - `go test -race ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server -count=1`
  - `go vet ./internal/identity ./internal/idempotency ./internal/mailer ./cmd/han-server`
  - 全仓按 Manifest 固定顺序执行：`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`
  - owner 核对：`git diff --check -- internal/identity/administrator_create.go internal/identity/administrator_create_test.go migrations/0007_idempotency_response_snapshot_v1.sql internal/idempotency/store.go internal/idempotency/store_test.go internal/idempotency/store_integration_test.go internal/identity/handler.go internal/identity/handler_test.go cmd/han-server/main.go cmd/han-server/main_test.go internal/mailer/smtp.go internal/mailer/smtp_test.go`；并由 Lead 复核 `git status --short --` 同一清单、scoped diff、所有新增/修改文件最终 SHA256、条件性 Mailer 是否确有失败证据及无范围外写入。未跟踪新增文件不以 `git diff --check` 单独冒充内容检查。
  - 当前状态：Lead 最终修订现场的 SMTP/identity 定向、mailer/identity 普通与 race、四包普通/race/vet、安全 PostgreSQL 18 integration、Manifest 七项、owner hash/status/whitespace 与暂存区核对全部 PASS；I-S52-01～09 已完成。真实 SMTP/TLS、Host、部署、E2E、容量与 production readiness 未执行，不得由本 task closure 外推。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.3`、`identity-auth-S1.4`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S5.1`
- **并行**：独占串行；返工闭环期间不得与任何其他 task 并行实施，尤其不得并行写 `internal/idempotency`、`internal/platform`、`cmd/han-server` 或 identity 包级编译面。
- **阻塞注记**：本 task 无剩余设计、实施、验证或评审阻塞，已关闭。真实 SMTP/TLS、Host、部署、E2E、容量与 production readiness 属后续独立验证边界，未执行且不回退本 task closure。
- **本任务关闭条件**：已满足。`0007`、strict legacy/v1、管理员只写 v1、canonical snapshots、当前请求 trace、D001、SMTP operation budget/deadline/cancel、三条 production route、共享 lifecycle 与 canonical effective-at 值流/启动拒绝矩阵均完成；PostgreSQL 18、定向/包级/race/vet、Manifest 七项、owner 核对、Faraday r04 与 Lead §10 Gate 全部 PASS。
- **关闭证据**：最终主要对象 SHA256：`administrator_create.go` `DD1D1DA23CB9D9E13E3200E09867B2A2C2F6BCBF06894C46074F60AC94D75813`；`administrator_create_test.go` `26E059CD710160FFE7E6F6DCE793859DA478689D0038641EFDC2E5D335463B34`；`smtp.go` `EA42BC63AFE1E8577E4F5C7C01FDCBF30C5A00496D8ED53051A199234CF633D8`；`smtp_test.go` `A4B7784F224800F6476C585526E7ACAB1D87E5882427799155BA2EE35B6D5364`；`main.go` `FDED5C74F9BB06D3BACD6E3B349D05ABF59B75C93C74C4EDA59D28EE0E9ABA30`；`main_test.go` `2B5336CF9994F795CB01928BB07CDAE3D47C0FED4E0D0AF07E66BAD9E3442884`；`store.go` `E9AAE069942F310EB155FBCAE3E981B55C87ABCAFCF3ACA74730292483C57FA5`；`store_test.go` `DB291F9AE7F501F60ABF71F715FC2DD2961067E8598CED64348A7A6AC77DAC5C`；`store_integration_test.go` `BAE146223ABADF5E3884962E2BDD119AE46964D694B3BE4A38B331DAA956CD5A`；`handler.go` `F21E1E9B824DADCD784AA89E4A9865CDD85BF17A372B4D9C2964A91250C456D9`；`handler_test.go` `7A32FEF8C9C94489D8D8E055C6B5ACEB7E2EA0304BCF028BDE0CF1D0BCA6373F`；`0007_idempotency_response_snapshot_v1.sql` `9D3E642D4F5C5DE81113E23AEBD70E2E07BF583E140697D335DB4DDA35F01F9F`。
- **后续验证关联**：identity-auth-S5.3、identity-auth-S3.4
- **设计/测试来源**：identity Story S05；identity design revision `identity-auth-design-ir01-r33`，blob `1c6ea6827a18158a9936a4612aac523b258e4e5d`，SHA256 `47FC3598CE2BCA7A1E80032FEC040BCF63344C869B4EF12F9BE5998589563513`；test-plan revision `identity-auth-test-plan-ir01-r13`，blob `c1b7ec4e9c201c2c24a5afe7300532eb7e06e4e3`，SHA256 `F5A3E2CEF38C36E595D8217BDD85BBA7CF55FC0F1DBB311D773E7DDBD1076749`；design review `.spec/wip/identity-auth/review-design-r33-s5.2-r05.md` SHA256 `8E3E478BD94D4CF827B2A7E1E5FD31DE052EFC2174172AB2C3191C2AE38F245D`（原 `IDENTITY-DESIGN-R32-S5.2-R04-F001` CLOSED、无新增 finding、Design Review Gate PASS）；final code review `.spec/wip/identity-auth/review-code-s5.2-r04.md` blob `e473153c726ee499254bd90c7f42df1cc82a8a97`，SHA256 `B9C1598C2B8CAC42F055E355A6D95F28FBA5B8F459FB94495D72674F9945E075`，Gate PASS；identity/framework decisions；test I-CRED-01/02/04、I-S52-01～09。r31/r12/r03 仅保留为历史追溯，不再是当前执行来源。

- [x] identity-auth-S5.3 验证 SMTP/TLS 与凭据残余窗口 [P] 开始@2026-10-25 完成@2026-10-25

- **目标**：验证 SMTP/TLS 与凭据残余窗口。
- **类型**：专项安全/集成测试
- **责任与当前状态**：Backend-Dev Kant `01a1716c-b694-71c3-8d72-2a4fc1ab4eb3` 的原 G0 停止/三文件零写入与 S10.1 上游 repair 历史保留；随后在固定三文件完成最小返工。R01 为 `CHANGES REQUIRED`，R02 确认四项 finding 全部 CLOSED、0 新 HIGH/MEDIUM finding、Code Review Gate PASS；Lead 同 revision Gate PASS并于 2026-10-25 关闭本卡。
- **文件**：
  - `test/integration/identity/smtp_credentials_test.go（新增）`
  - `test/integration/identity/smtp_test_server.go（新增）`
  - `test/integration/identity/testdata/certificates/README.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/identity -run "TestCredentialSMTP|TestSMTPResidualWindow"`
  - 当前状态：固定 revision 上的定向 `-count=10`、定向 race、scoped vet、三文件 gofmt/diff-check、错误文本扫描与 Manifest 七项均由 Reviewer/Lead 核证 PASS；专用命令与项目检查未互相替代。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S5.2`、`identity-auth-S10.1`、`identity-auth-S11.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **认领状态**：Kant 的三测试文件实施窗口已随 task-level closure 释放；后续写入须由新的已授权任务重新核依赖、基线、未知写者与所有权，不得借已关闭卡继续修改。
- **阻塞历史**：原 G0 因共享 coordinator classification 缺口暂停；S10.1 五文件 repair、R08 与 Lead Gate 后解除。该历史不改变本卡始终只允许 local real-protocol SMTP/TLS、不发外部邮件，且不得用 fake、错误字符串匹配或把 unknown 当 rejected 伪造 PASS的合同。
- **关闭证据**：R01 `.spec/wip/identity-auth/review-code-s5.3-r01.md` SHA256 `78835EC544357CFB6BD970DF5F59EC5E20C9FD3CFCFA00AC3495B82EE0E7049B`；R02 `.spec/wip/identity-auth/review-code-s5.3-r02.md` SHA256 `57FC85996662492B35AC78F6BCB25F262EFBA35C0DD2B053822765DEB5453ABA`。最终三文件 SHA256：`smtp_credentials_test.go`=`9CB60DE64DB1B00B3AA92EB32788CA042A3390FE583D60709FA1F295CA4C8826`、`smtp_test_server.go`=`FEFF245A580D7D848A780EB63C5AD1D0E2B13582ADDCC3BE6C9506C9F82DFB26`、`testdata/certificates/README.md`=`B9CE292F5A8701756646170C128825E6D955BF302D422E928B6664FE4847F73F`；Lead 固定 branch/HEAD/staged/哈希/唯一写集/`han-server.exe`，定向 `-count=10`、race、scoped vet、gofmt、diff-check、错误文本扫描与 Manifest 七项全部 PASS。
- **本任务关闭条件（已满足于上述固定 revision）**：local real-protocol SMTP 下覆盖 implicit TLS/required STARTTLS、降级、证书故障、未受理、accepted/unknown、保护窗口与重试；分别验证 new_administrator residual-window 和普通 password_reset 首次/resend 的唯一 dispatch owner/settlement，worker/replay 不发信；不发外部邮件；独立 Review 通过。
- **未验证边界**：仅证明 loopback 本地合成 SMTP/TLS/AUTH、accepted/rejected/unknown 与凭据残余窗口；未连接真实 PostgreSQL或外部 SMTP，未部署/E2E，不外推真实 provider、容量或 production readiness。
- **后续验证关联**：identity-auth-S2.2、identity-auth-S3.4
- **设计/测试来源**：identity test SMTP/TLS 与凭据残余窗口专项；I-CRED-01～06；test-plan r18 I-PRD-06～10/18/20；identity design r40 §3.3.11/12、§4.5、§9.4

- [x] identity-auth-S6.1 实现其他管理员查询、停用、恢复、删除与撤权 开始@2026-10-27 完成@2026-10-27

- **目标**：实现其他管理员查询、停用、恢复、删除与撤权。
- **类型**：开发
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/service.go（修改）`
  - `internal/identity/administrator_governance.go（新增）`
  - `internal/identity/administrator_governance_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestAdministratorGovernance"`
  - 当前状态：专用命令与 `-race` 均 PASS（`ok 0.088s` / `ok 1.105s`）；`go test ./internal/identity -count=1` PASS；现役 Manifest 七项按固定顺序全部 PASS（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）；另 `go vet -tags=integration ./...` PASS。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S7.1`、`identity-auth-S8.1`、`identity-auth-S9.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：初始 admin 是唯一治理者；旧会话即时失效；重复目标状态稳定；initial admin 永久受保护；失败证据和独立 Review 完成。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S3.4
- **设计/测试来源**：identity Story S06；test I-ADM-04～05
- **独立复审与 Lead Gate**：本 Host 无委派能力，无独立 Reviewer Role 产出；经用户 2026-10-27 授权以**作者自审**口径执行 Code Review，产出 `.spec/wip/identity-auth/review-code-s6.1-r01.md`（SHA256 `2A612A4B931A5BA4E8B960277D1337AF4477C5CDB6930C5031A91F2A8CD7BCF7`（2026-10-31 脱敏后；原 `B76A590923A6F61F1E9E0085B2EA26B6B7DAAA5340DCB46460AAD06C6B2AF944`）），`Code Review Gate: PASS（作者自审口径）`，0 CRITICAL/HIGH/MEDIUM/LOW finding。Lead 在同 revision 核四文件哈希、专用命令、race、Manifest 七项、integration vet、`git diff --check` 与写集范围。
- **关闭证据与限制**：最终 SHA256 为 `handler.go`=`EC4533A6CAA6819255BF9FC7D19EEE3AEEAA90B9E3A9128A5D24B5017D1F1EA3`、`administrator_governance.go`=`DC43A60525D1A4B7E40587A40EB5EE85183FE4E46F79FE13020DA48AB0C5DBA3`、`administrator_governance_test.go`=`E64924FF592F1CA25F5158BB248A47530DF1C59E2AB5960B79F6C09C4902A122`；`service.go` 登记但本卡无需改动、未修改（`7E5064E40447A35EFC4F6537ED845E6EBD4A7199169208302B63C5905643C70D`）。真实 PostgreSQL 18.6（本机 5433）临时 integration 探针（不在写集内、验证后删除）证明新 `UpdateRole` SQL 与复用会话撤销 SQL 落库正确、重复撤权 0 行报错。本卡只闭合 `revoke`（`admin → member` 角色转换 + 撤销全部目标会话 + `identity.role.revoke` 审计）；`List/Get/Suspend/Restore/DeleteAdministrator` 五个 handler 在卡前已存在并归 S7.1/S8.1/S9.1 关闭链。**production 路由接线未完成**：`/api/v1/admin/administrators...` 整片治理路由（含既有 suspend/restore/delete/list/get）尚未在 `cmd/han-server/main.go` 注册，属 `cmd/han-server` 唯一 composition owner 的独立工作范围，不在本卡四文件写集内，须由后续卡按所有权认领。无真实 HTTP/浏览器/E2E、部署、容量与 production readiness 证据。作者自审弱于非作者独立复审，作为已知限制保留。

- [x] identity-auth-S7.1 实现用户与管理员最小披露查询和游标分页 开始@2026-10-23 完成@2026-10-23

- **目标**：实现用户与管理员最小披露查询和游标分页。
- **类型**：开发
- **实施责任**：Backend-Dev Noether `01a16893-bc25-7a52-8532-4bd7e203565d`；非作者 Reviewer Meitner；Lead 关闭@2026-10-23。
- **文件**：
  - `internal/identity/handler.go（修改）`
  - `internal/identity/query.go（新增）`
  - `internal/identity/cursor.go（新增）`
  - `internal/identity/query_test.go（新增）`
  - `internal/identity/cursor_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestUserQuery|TestIdentityCursor"`
  - 当前状态：Lead 已在五个最终文件固定 SHA256 上取得本卡定向测试、`go test ./internal/identity`、定向 race、`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...` 与 `go test ./...` 全部 PASS，五文件 scoped diff-check PASS，暂存区为空。非作者复审 `.spec/wip/identity-auth/review-code-s7.1-r02.md`（SHA256 `53836018C0A4C9C7D374CC3D8ED6CDE651E09D567E8F44A3A0A818BE81AB8E15`）给出 Code Review Gate PASS、无新 finding；该报告在当时记录 F001/F003/F004 CLOSED、F002 为 `TRANSFERRED / ACCEPTED-WITH-FOLLOW-UP` 并转交 S7.2、F005 为用户已接受的 LOW 风险且未代码关闭。F002 后续已由 S7.2 独立复审正式 CLOSED，不改写 r02 的历史时点结论。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`identity-auth-S3.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：普通管理员不可枚举管理员；字段 allowlist、稳定 created_at/id 游标、非法/过滤不匹配游标先红后绿；独立 Review 通过。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S8.1、identity-auth-S9.1
- **设计/测试来源**：identity Story S07；test I-LIST-01～03
- **关闭证据与限制**：最终五文件 SHA256 分别为 `handler.go`=`CD7DD480EA0D4E6A34694B254237177583FB22DB8F0416A6FC9D9064FC16DC75`、`query.go`=`534E6E362431B669D271D689365161F95E843D9838D2DA03398F51806DD2C407`、`cursor.go`=`E0514475A7ECA31E24A8092A51D2F814DCCB112F7D2022A3D903E2AA5A1A6FFD`、`query_test.go`=`786B1A3ED94CC0DA5AF9B004250DE74FED67C92108436A0E18F1B6E7CF6886E9`、`cursor_test.go`=`4F5D647E2EA29CC4D4EB2B837178895FD40E9F24E85D4976BC71BC9C26C4864D`。本卡关闭本身只覆盖五文件查询/分页闭环，不曾证明 S7.2 索引或真实 PostgreSQL 查询计划；这些后续由独立 S7.2/B5D 闭环完成。production router、浏览器、部署或 E2E 仍不在本卡范围。

- [x] identity-auth-S7.2 为用户治理列表补充分页索引 migration 开始@2026-10-23 完成@2026-10-23

- **目标**：在不扩大 S7.1 五文件边界、不修改既有 migration 的前提下，为用户治理列表交付设计规定的 PostgreSQL 分页索引及可复核访问路径证据。
- **类型**：开发（数据库 migration）
- **实施、Reviewer 与 Lead 责任**：Backend-Dev Newton `01a168f4-af27-7a11-bd44-5641a1a26efa` 仅实施本卡两文件；`migrations/0006_user_governance_indexes.sql` SHA256 `6F93D99F6DFA0DE46DF8D891F8979039F679493932C28326EEB2AF28E2F24660`，`internal/identity/repository_integration_test.go` SHA256 `89E6284C41937C13BC2EECBFD509061A0FB5D7DB04EE0DC1E303C65F36A6B8EA`。接手前 repository test 的 Lead 真实 SHA256 为 `FE6F63FF1C89E68CD61EA24B94D51D7FDED8947BEDA6FDB3CCEB7837F3A07448`；反向剔除本轮 import 与连续测试块后恢复 1069 行、32092 bytes 及该 SHA，证明既有内容字节级未改。非作者 Reviewer Aristotle `01a168fb-9111-7f32-b542-64270559e9c1` 的唯一报告 `.spec/wip/identity-auth/review-code-s7.2-r01.md` SHA256 `F0CBEC197498E49B8536F2EEF064B3C7D82B94BD55CAA2AC5D5E793940E12997`，0 finding，`IDENTITY-S7.1-REV-R01-F002 — CLOSED`，`Code Review Gate: PASS`。Lead 负责并已完成最终哈希、范围、测试、Review、暂存区与 §10 关闭核证。
- **文件**：
  - `migrations/0006_user_governance_indexes.sql（新增）`
  - `internal/identity/repository_integration_test.go（修改）`
- **文件所有权**：本卡唯一拥有上述两文件的本次增量；`migrations/0001_shared.sql`～`0005_recovery_runs.sql`、S7.1 的五个实现/单测文件、设计、test-plan 与 Review 报告均只读。发现需要第三个写文件、修改既有 migration 或改变公共查询契约时停止并回 Task-Planner/设计活动。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./internal/identity -run "TestIdentityRepository_UserGovernancePaginationIndexes" -count=1`
  - `go test -tags=integration ./internal/identity -count=1`
  - `go test -tags=integration ./internal/migration -run "TestRunner_(AppliesInOrder_RecordsOnlyOnce|FailingMigration_RollsBackAndStops)" -count=1`
  - 当前状态：PostgreSQL 18.6 上本卡目标测试 PASS；`(role,status,created_at DESC,id DESC)` 用于 role/status 过滤，`(created_at DESC,id DESC)` 用于无过滤与最终 S7.1 SQL；同时间戳多页无漏无重，deleted/self/角色范围均通过。identity integration 包 PASS；migration 顺序、重复只记录一次与失败事务回滚两用例 PASS。Lead 按 Manifest 固定顺序运行 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`，七项全部 PASS；两 owner 文件及 Review 报告 diff-check PASS，暂存区为空，S7.1 五文件哈希与 r02 固定值一致。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`identity-auth-S7.1`
- **并行**：B5D 内串行且唯一 task 为本卡；B5B/S7.1 已完成本卡所消费的最终 SQL/Review 转交，技术前置已满足。用户已授权本轮启动，其两文件与 blocked Browser lane 的写集无交叉；该授权不启动 B5C、B6A 或其他 task。
- **阻塞注记**：无。`IDENTITY-S7.1-REV-R01-F002` 已由 Aristotle 在 S7.2 r01 正式复审关闭；S7.1 r02 的 `TRANSFERRED / ACCEPTED-WITH-FOLLOW-UP` 保留为历史时点记录。
- **本任务关闭条件**：新增 migration 仅向 `users` 增加与最终 S7.1 SQL 过滤及 `created_at DESC, id DESC` keyset 排序一致的 `(role,status,created_at DESC,id DESC)` 与 `(created_at DESC,id DESC)` 索引，不修改 `0001`～`0005`；PostgreSQL 18 集成测试验证两项索引定义，以及 role/status 与无过滤访问路径、同时间戳多页、不漏不重、排除自身和角色范围；取得 `EXPLAIN (ANALYZE, BUFFERS)` 或等价计划证据。现役 migration runner 须证明正向顺序应用、重复运行只记录一次及失败事务回滚；项目没有 down migration 合同，不虚构反向迁移。局部测试、适用 Manifest 七项、非作者代码 Review、F002 复审关闭与 Lead §10 核证现已全部齐备。
- **E2E 分界**：本卡关闭只覆盖 migration、PostgreSQL 18 查询计划与数据库访问路径、局部集成测试及 Review；不外推生产负载性能、Browser、部署、真实用户旅程、跨系统 E2E 或 production readiness。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S3.4
- **设计/测试来源**：identity design r28 §3.3.8；test-plan r11 I-LIST-01；Review `IDENTITY-S7.1-REV-R01-F002`

- [x] identity-auth-S8.1 实现停用/恢复与作者状态只读接缝 开始@2026-10-24 完成@2026-10-24

- **目标**：实现停用/恢复与作者状态只读接缝。
- **类型**：开发
- **实施、Reviewer 与 Lead 责任**：Backend-Dev James `01a16dc2-80e8-7603-b685-ea20cfe4887d` 为唯一实现 owner；非作者 Reviewer Erdos `01a16dcb-0ccc-79d1-be80-ed4624b5724e` 完成独立复审；Lead 已核固定对象、真实命令输出、diff、文件所有权、Review 与未验证边界并按 §10 关闭。
- **文件**：
  - `internal/identity/handler.go（修改）`，SHA256 `40E4D8D6D4BE5497B3DDD7C513DA144AB105EDC0A5DADF068BDFDF06020450C6`
  - `internal/identity/account_status.go（新增）`，SHA256 `C03B8D155580CECE26707863D24062FC54B6C3707F2C006F38C3DAB408C00FA0`
  - `internal/identity/article_reader.go（新增）`，SHA256 `D9315AFA411CB8C4612459C8E97979BD2274780A7B7B32D78F3AB735C3F82B6B`
  - `internal/identity/account_status_test.go（新增）`，SHA256 `5E349230E58FE6F8EF1C899559B10DB9AD47050D9BD54660EA31B65599F26CCA`
  - `internal/identity/article_reader_test.go（新增）`，SHA256 `C6D2B246236D24FC27C9E2C66413DE03999EA98D61CD61B2054D0152206C28AC`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestAccountStatus|TestGetAuthorStates" -count=1`
  - 定向 race：`go test -race ./internal/identity -run "TestAccountStatus|TestGetAuthorStates" -count=1`
  - 当前状态：Lead 最终现场的定向测试与定向 race 均 PASS；返工后按 Manifest 固定顺序执行 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`，七项全部 PASS；五个 owner 文件与 r01/r02 scoped `git diff --check` PASS，暂存区为空。未读取或使用 `HAN_TEST_POSTGRES_DSN`。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S7.1`
- **并行**：本卡已关闭并释放五文件独占窗口；S10.1～S10.4 与 B6D/S11.1 保持 task-level CLOSED，B7 classification repair 的五文件窗口也已释放。S10.5 尚未启动。
- **阻塞注记**：无。
- **本任务关闭条件**：目标状态幂等、每次重新授权、停用撤销全部 Session；Article 零写入并只读返回作者状态；独立 Review 通过。
- **Review 与关闭证据**：r01 `.spec/wip/identity-auth/review-code-s8.1-r01.md`（SHA256 `3E597ED60AA7523EF62B20B2FFE32586726E607C578108EE7762E1C5F4303109`）记录 3 Major + 1 Minor、`CHANGES REQUIRED`；r02 `.spec/wip/identity-auth/review-code-s8.1-r02.md`（SHA256 `7EA534A0CC7A07EFC0C9883B2E42DDEB012B4F0CB3E8C357EB47D31750BB1EF7`）确认 F001～F004 全部 CLOSED、无新增 finding、`Code Review Gate: PASS`。Lead 最终现场证据满足上述命令、固定对象与所有权核对，任务完成@2026-10-24。
- **关闭边界**：真实 PostgreSQL `FOR UPDATE` 竞争、物理回滚与旧 Session 下一请求失效继续由已登记 `identity-auth-S1.6` 闭环；production composition/route 与 Article 消费接线属于后续任务。Browser、部署、E2E 与 production readiness 均未因本卡关闭而完成。
- **后续验证关联**：identity-auth-S1.6、identity-auth-S9.1、article-management-S2.1
- **设计/测试来源**：identity Story S08；test I-GOV-01～05；S2/S3

- [x] identity-auth-S9.1 实现 canonical 用户删除与标识/retained_email 生命周期 开始@2026-10-25 完成@2026-10-27

- **目标**：实现 canonical 用户删除与标识/retained_email 生命周期。
- **类型**：开发（跨模块协调）
- **文件**：
  - `internal/identity/deletion.go（新增）`
  - `internal/identity/article_port.go（新增）`
  - `internal/identity/deletion_test.go（新增）`
  - `internal/identity/service.go（修改）`
  - `internal/identity/handler.go（修改）`
- **当前对象与 Review 证据**：R01 `.spec/wip/identity-auth/review-code-s9.1-r01.md`（SHA256 `813B8B947935101CA5C313DB77432B984994B3F71FED21360C9B3A67D325167B`）结论为 `CHANGES REQUIRED`；**历史保留**。**R01 固定对象 SHA256（历史）**：`deletion.go`=`87F697959265240EE303B3BE4E2A9A5B49644A84B6ED58D83E3433685B5E9821`、`article_port.go`=`F12DB7648F36CD530879434A9B0D0E4231E955253614ADFA5F3999AF40A1ED10`、`deletion_test.go`=`A3E2F1871A88455149805B1F0B9B798D1E67F39599C2508BD5E41617570E1663`、`service.go`=`7E5064E40447A35EFC4F6537ED845E6EBD4A7199169208302B63C5905643C70D`、`handler.go`=`FC4B1711F8B25DB8508DAA8DE1ACD87AB38C7B0F1655679768FD64C80304C8FC`。上游设计阻塞已由 identity design r45（revision `identity-auth-design-ir01-r45`，SHA256 `CBE0F64ADB93E1AFD4340D2D3390ACE42EE85F557DDCBA5AEBCF2AA20CE404F1`）承接并经 Design Review `.spec/wip/identity-auth/review-design-r45-s9.1-r01.md`（SHA256 `5905C0D1CF1EF66FAECDF559AC01DD5795A1A2BAA78AAF7874232A2FB13D8D74`）Design Review Gate PASS。**2026-10-26 Lead 接管实施后对象 SHA256（最终固定）**：`deletion.go`=`119D378C3508A790FFC6E7371C5AFC0D8CE100A33C2F47AFACD305F7683A2721`、`deletion_test.go`=`2F5975867E74EC6B6E8A0AD1DFD128382EC5123FB308A7924A5B6D4294ED6687`（`article_port.go`/`service.go`/`handler.go` 与 R01 固定值一致、未改）。**复审**：`.spec/wip/identity-auth/review-code-s9.1-r02.md`，Code Review Gate PASS（F001～F004 全部 CLOSED、0 新 HIGH/MEDIUM；依据用户 2026-10-26 作者自审授权）。
- **Story-level R gap-map**：`identity-auth-S9.1`（`S09 删除用户与建立墓碑`）→ **已建成（R01 F001～F004 已由 r45 设计修订 + r02 复审闭合）**（历史：需改/返工）。Desired 定位：`.add/project/knowledge/design-workspace/identity-auth/stories.md:298,417` 与 `.add/project/knowledge/design-workspace/identity-auth/design.md:705`、design r45 §3.3.10 与 §4.2 新增 3.1 步固定 canonical 删除及 identity-owned RecoveryRunStore 唯一 authority、RecoveryGate/Coordinator 对账边界与标识墓碑不变量；R01 F001～F004 已由 Lead 按 r45 在原五文件内实现，**代码侧仍 OPEN**，待非作者独立 Code Review 逐项复核。写集仍为原五文件，保持 `[-] implementation done pending non-author Code Review`；规划或自实现证据不构成符合现役设计的最终确认。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity -run "TestCanonicalDelete|TestIdentifierLifecycle"`
  - 当前状态：R01 已给出 `CHANGES REQUIRED`；上游设计已由 r45 补齐到可实施程度，写集仍为原五文件，未扩为第六文件。不得登记为已测试、Review PASS或可关闭。本轮只做设计增量对账，未执行本卡专用命令或 Manifest 七项。Lead START 前不得运行返工验收。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S7.1`、`identity-auth-S8.1`、`article-management-S6.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **认领状态**：B7 内五文件已有实现候选；R01 自称作者为 Poincare `01a1716c-b94a-7602-b84b-02978e3d1db8`，但仅作为作者线索，权威 owner 与 Reviewer 均为 unknown；当前不构成返工认领。设计修订由 Lead 接管产出（r45，无 Feature-Designer Role 产出）；实施 owner 与非作者 Reviewer 待 Lead 在写入前分配。
- **阻塞注记**：CLOSED（2026-10-27）。R01 要求关闭 F001 phone identifier fail-closed、F002 kept_deleted 的 RecoveryRun/maintenance authority gate、F003 TraceID 双来源、F004 函数长度，r45 已在原五文件边界内把这四项转为可实施合同（F002 通过 §4.2 新增 3.1 步同一 identity-owned RecoveryRunStore + maintenance/readiness Gate 双重证明实现，不新增第二 authority），Lead 已按此在原五文件内实现。**本 Host 无委派能力，无 Backend-Dev Role 产出；亦无法产出非作者独立 Code Review**；依据用户 2026-10-26 明确授权「由作者自审 Code Review」，由 Lead 以作者自审口径输出 `.spec/wip/identity-auth/review-code-s9.1-r02.md`，F001～F004 全部 CLOSED、0 新 finding、Code Review Gate PASS；该口径弱于非作者独立复审，作为已知限制保留。未修改 article 文件、schema/migration 或另建 canonical authority。r45 §6 残余限制（路径 B 的 phone 值级绑定）已由用户 2026-10-26 授权采用建议值=保持 r45 现状：路径 A 为正常重放权威路径，路径 B 对 phone 仅做「行存在 + `reserved` + `released_at IS NULL`」存在性/未释放校验，不新增 migration、不扩张本卡写集；「新增 retained_phone 列」登记为需单独授权的后续候选。
- **本任务关闭条件**：实施已由 Lead 按 r45 §3.3.10/§4.2 完成；R01 F001～F004 已由复审报告 `.spec/wip/identity-auth/review-code-s9.1-r02.md` 逐项复核为 CLOSED（phone/值绑定异常全事务失败、maintenance/RecoveryRun 权威 Gate 不可由 request 或孤立 review 伪造、ctx 单一 trace 权威、函数满足 profile ≤50 行），定向故障用例证明 User/Identifier/Session/凭据/Article/Audit 任一步失败全回滚；Lead 现场 Gate PASS。**依据用户 2026-10-26 明确授权「由作者自审 Code Review（本 Host 可以自审）」，本卡以作者自审口径闭环**；该口径弱于 add-code-review 的非作者独立复审，作为已知限制记录。已于 2026-10-27 关闭。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S9.2
- **设计/测试来源**：identity Story S09；test I-DEL-01～07；S4/S5/S14

- [x] identity-auth-S9.2 验证 canonical 用户删除跨模块事务 [P] 开始@2026-10-28 完成@2026-10-28

- **本卡关闭（2026-10-28）**：Lead 已核原四 requires、两文件成果、真实执行与原关闭条件，正式决定且仅关闭 `identity-auth-S9.2`。作者 Rawls 与非作者 Lovelace 分别完成普通/race 各 64 leaf/76 节点、0 fail/skip 及专项 vet；Lead 项目七项全 exit 0。Lovelace 已两轮独立评审、Code Review Gate PASS/0 finding 并 stopped；不是把作者结果算作独立执行，也不升级原 S9.1 作者自审来源。
- **最终版本与证据**：固定 source183-r02 清单 SHA256 `03594FB126EB3E2493A14876E33AE5C1475E2F66C908BDB3D484B6E61CCBB73E`，原 181 源保持基线、仅本卡两新文件；[独立报告](../../../../.spec/wip/identity-auth/review-code-s9.2-r01.md) SHA256 `4AD92EE874C7FC2A688A5AABDDBBD6581D8873C650CD29C8BEB0126D320BB051`。详细作者/Reviewer 原始日志、SQL/留存 schema 核证、Lead Manifest 与四个 durable 索引见 [Lead 核证记录](../../../../.spec/wip/identity-auth/verification-s9.2.md)；不复制第二份运行台账，不代签人的 accepted/checkpoint 或授予下一动作。
- **准备认领历史（2026-10-28，对应当时窗口）**：Lead 已实际启动 TestEngineer Zeno `01a18020-4347-7b72-9f08-5661c319d33a`，只读+个人 Temp Prep 已 completed/stopped，Lead 已全文采用公开接口、真实事务、原生 proof 与 retained fixture 的只读可行结论，见 [Zeno Prep](C:/Users/J00010385/AppData/Local/Temp/han-s92-lead-guoi9beq/testdesign-prep.md)；原两文件采用两个 top/六 nested 风险组（TestCanonicalUserDeletePostgres、TestCanonicalUserDeleteFaultsPostgres），均命中原验收筛选；不代表测试 PASS 或 full I-DEL/注册/HTTP/production Gate 已验证。仅据真实准备认领登记进行中；Lead 后续已实际启动 Backend-Dev Rawls `01a18028-4625-7bf3-9824-9409764e641b`，仅原两新文件 G0/CODE/gofmt 与无 DSN 的 list/compile-only 窗口，原 181 源保护；G0 与静态结果尚待作者回交，独立 Reviewer 未实际分配，无 PG/DSN/生产写授权。本次 metadata 不另发实施或 PG START。
- **固定输入与范围**：[identity design r45](../design-workspace/identity-auth/design.md) §3.3.10/§4.2（SHA256 `D57BB2152A93F6F651165EE7DBE1913B28E9E8719B6991B1FDD36E8B7DBAB7FE`）；[identity test-plan](../design-workspace/identity-auth/test-plan.md) I-DEL-01～07（SHA256 `0052894D3C4B58F644749F8337BB264FD2768DD26A31C7EA0A361F0F61D939BD`）。准备基线核对时原两文件均未创建、待 Rawls 回交，现已交付，最新执行/评审状态见上述 Lead 核证记录；路径 A 保持标识值绑定，路径 B 的 phone 仅证明墓碑存在、reserved 且未 released，沿用已接受的残余窗口，不加强合同或扩迁移。
- **目标**：验证 canonical 用户删除跨模块事务。
- **类型**：专项集成测试（identity 主责）
- **文件**：
  - `test/integration/crossmodule/canonical_user_delete_test.go（新增）`
  - `test/integration/crossmodule/canonical_delete_faults_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/crossmodule -run "TestCanonicalUserDelete"`
  - 当前状态：专用命令 `go test -tags=integration ./test/integration/identity -count=1`（真实 PG18.6）与 `-race` 均 PASS；Manifest 固定七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）已按序全部 PASS。二者互不替代。
  - 历史证据边界：上一行“当前状态”为其他 identity 专项/Manifest 的历史原句，原文保留，不能替代本卡 crossmodule 两文件或新 top 的执行；本卡当时实施/静态结果仍待回；本轮执行、独立评审与 Lead 仅关闭本卡的最终决定见上述核证记录，不以旧 identity 专项证明本卡完成。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S9.1`、`article-management-S4.1`、`article-management-S6.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：原四 requires 均已关闭，本任务声明范围的真实 PG、独立 Review 与 Lead 核收已满足。controlled maintenance Gate 仅为明示 local test seam，非 production 全局恢复/开流；SQL UNIQUE 非完整注册，组件矩阵非 HTTP/浏览器验证。真实 Journal/旧备份/global recovery/MarkConsumed 与后续验证仍归原关联，不倒挂 article S4.4，也不释放 root/identity/架构写权。
- **本任务关闭条件**：真实 PostgreSQL 上覆盖零/多/违规/deletion_pending/kept_deleted、审计失败和 retained_email 清理；任一模块未实现时不关闭；独立 Review 通过。
- **后续验证关联**：identity-auth-S4.4、identity-auth-S3.4
- **设计/测试来源**：identity/article test canonical 删除跨模块专项

- [x] identity-auth-S10.1 实现管理员重置用户密码 开始@2026-10-24 完成@2026-10-25

- **目标**：在保留既有 durable protection-window、两条 public REST route、权限/错误/秘密安全合同与唯一 production composition 的前提下，使唯一 Mailer/PasswordReset coordinator 对 accepted、明确未受理与 unknown 做稳定三分流：明确未受理调用既有 `RecordRejectedDelivery` 且只失效本次 new candidate；unknown 继续保持 unresolved Gate；accepted 继续进入既有 accepted settlement。
- **类型**：开发返工（R 对账复用既有 task）
- **实施、Reviewer 与当前状态**：历史关闭/重开链保留：Curie/Copernicus/Lead 的旧 revision closure、B6C-3 integrated R03、B7 G0 再次重开与 Kant 五文件 repair 均可追溯。R07（SHA256 `BAEB4C7BA121A71EE9E66934DC933365BF73D0AC0545B6887BCC078EBC99C27C`）的 `IDENTITY-S10.1-REV-R07-F001` 已由非作者 R08（SHA256 `30F1C4D2546F6E32BA23FCBECAD22235D49574CE35E66A236A8CCA18712DDC20`）确认 CLOSED，0 新 finding，Code Review Gate PASS；Lead 于 2026-10-25 接纳并关闭本卡。
- **文件**（本轮精确且唯一五个）：
  - `internal/mailer/mailer.go（修改）`
  - `internal/mailer/smtp.go（修改）`
  - `internal/mailer/smtp_test.go（修改）`
  - `internal/identity/password_reset.go（修改）`
  - `internal/identity/password_reset_test.go（修改）`
- **文件所有权与消费者**：Kant 独占上述五文件；先在 mailer package 保持 `Send(context.Context, Message) error` 签名不变地提供稳定、可用 `errors.Is` 判定的 proved non-acceptance taxonomy，再由唯一 PasswordReset coordinator 消费。真实消费者包括首次 reset 与 S11.1 普通 `password_reset` resend；S11.1 不拥有或复制分类逻辑。禁止解析错误字符串、修改 administrator/article 消费者、worker/operator/store/schema/migration/composition/public REST、引入第二 authority/root/global或第六文件。
- **TDD ②档失败证据**：
  - SMTP adapter 对协议明确拒绝/明确未接收返回稳定 rejected classification；dial/TLS/AUTH/timeout/cancel/EOF/写入中断、DATA acceptance reply 不可判定等均不得误标 rejected，继续归 unknown；DATA 已明确 accepted 后保持 nil/success。不得按错误文本分类。
  - coordinator 分别覆盖首次 reset 与普通 resend：accepted 调用 accepted path；rejected 恰好一次调用 `RecordRejectedDelivery`，operation/job terminal、只 invalidated exact new credential，old credential/auth generation/Session 保持且 Gate clear；unknown 恰好一次走 unknown path并保持 unresolved Gate。
  - rejected/unknown 的结果记录 begin/lock/write/commit 失败或 commit acknowledgement unknown 必须 fail closed，不得把不确定结果降格为 rejected、清 Gate或失效 old；保留 same-key replay、secret 清空、Mailer=1、worker/replay Mailer=0 与原 protection-window before/boundary/after 回归。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/mailer -run '^TestSMTPMailer.*(Accepted|Rejected|Unknown|Cancellation|TLS)' -count=1`
  - `go test ./internal/identity -run '^TestPasswordReset.*(Accepted|Rejected|Unknown|Delivery|ProtectionWindow|Replay)' -count=1`
  - `go test -race ./internal/mailer ./internal/identity -run '^(TestSMTPMailer|TestPasswordReset)' -count=1`
  - `go test ./internal/mailer ./internal/identity -count=1`
  - `go test -race ./internal/mailer ./internal/identity -count=1`
  - `gofmt -l internal/mailer/mailer.go internal/mailer/smtp.go internal/mailer/smtp_test.go internal/identity/password_reset.go internal/identity/password_reset_test.go`
  - `git diff --check -- internal/mailer/mailer.go internal/mailer/smtp.go internal/mailer/smtp_test.go internal/identity/password_reset.go internal/identity/password_reset_test.go`
  - 同一 revision 执行 Manifest 七项：`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。
  - 当前状态：Lead 于 2026-10-25 在固定五文件 revision 上核哈希、branch `main`、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`、staged 与越界，执行 mailer/identity 定向普通与 race、`go vet ./internal/mailer ./internal/identity`、五文件 gofmt/diff-check，全部 PASS。未读取/使用 `HAN_TEST_POSTGRES_DSN`；未跑真实 PostgreSQL/SMTP/部署/E2E，`han-server.exe` 保留。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S2.1`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S5.1`、`identity-auth-S7.1`、`identity-auth-S10.2`、`identity-auth-S10.3`
- **并行**：本卡五文件 repair 已关闭并释放独占窗口；article S2.1 与 article S4.5 也已由各自 R02/Lead Gate关闭并释放四文件窗口，S9.1、article S4.2/S4.3 仍按各自 Gate 与顺序所有权继续。S10.4、S11.1 保持 CLOSED且未重开；S10.5 不启动。
- **阻塞注记**：无；五文件 classification repair、非作者 R08 与 Lead Gate 已完成并关闭。任何未来新增缺口仍须按新证据重新对账，不得借本卡继续写入。
- **本任务关闭条件**：已满足。五文件 accepted/rejected/unknown 与首次 reset/resend 消费者矩阵完成；R08 Code Review Gate PASS，Lead 固定哈希/branch/HEAD/staged/越界、定向普通/race、两包 vet、五文件 gofmt/diff-check 均 PASS。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S1.6、identity-auth-S5.3、identity-auth-S3.4、identity-auth-S2.2
- **设计/测试来源**：identity design r40 §3.3.11/12、stories r16 S10/S11、test-plan r18 I-PRD-07 及 r44/r20 保持的 durable protection-window/consumer 合同；旧关闭/重开链保留，现行关闭证据为 R08 SHA256 `30F1C4D2546F6E32BA23FCBECAD22235D49574CE35E66A236A8CCA18712DDC20` 与 2026-10-25 Lead Gate。

- [x] identity-auth-S10.2 建立 password-reset durable schema、权威 Store 与六字段原子事务 开始@2026-10-24 完成@2026-10-24

- **目标**：以前向 schema 和 identity-owned Store 建立 execution counter、PasswordResetOperation、recovery/outbox job、exact pending credential、auth generation/lineage/settlement/integrity hold 与六字段 Idempotency claim 的 User-first 单事务权威状态。
- **类型**：开发（durable schema/persistence）
- **责任与状态**：Backend-Dev Bacon `01a16ec3-1d9a-7920-b014-4d38a44b5b31` 完成固定八文件实施与返工；原非作者 Reviewer Gibbs 的 R05 确认代码 F003/F004 CLOSED、无新增 finding、Code Review Gate PASS；Lead 于 2026-10-24 完成 §10 核证并关闭本卡。S10.1/F002 状态不变；S10.3 已于 2026-10-24 task-level CLOSED，S10.4 已于 2026-10-25 task-level CLOSED。
- **文件**：
  - `migrations/0008_password_reset_durable.sql（新增）`
  - `internal/identity/model.go（修改）`
  - `internal/identity/temporary_credential.go（修改）`
  - `internal/identity/temporary_credential_test.go（修改）`
  - `internal/identity/password_reset_store.go（新增）`
  - `internal/identity/password_reset_store_test.go（新增）`
  - `internal/idempotency/store.go（修改）`
  - `internal/idempotency/store_test.go（修改）`
- **TDD ②档失败证据**：在既有 I-PRD-02～05 五对象全有或全无、真实 `(scope_hash,key_locator)` 六字段绑定、复合 FK/UNIQUE/CHECK、User-first/no-row barrier、generation/lineage、坏绑定与 integrity hold 基础上，新增且必须直接覆盖：A 后正常改密、A→B current permanent、A→B→current temporary C；A same-key replay 返回 A 原终态；new-key 再引用旧 A 时为 `legal_history_stale` / 普通 stale-ineligible，并逐例断言 claim、operation、credential、job、generation、能力候选与 hold 零写入。`I-PRD-05.CP-01`、`CP-T01`～`CP-T05`、`CP-P01`～`CP-P04` 必须按 current projection 逐字段/单行 drift 进入 `integrity_drift`，同事务留下 durable hold 与去敏 Audit，除 hold/Audit 外仍为零写入；`CP-N01` 合法 root replacement 必须零 hold/零 drift Audit；`CP-F01` 分别注入 query、User lock、commit 瞬时故障，只 fail closed，禁止新增 hold、drift Audit、claim、operation、credential、job、generation 或能力写入。migration 仍先以 runner/static fixture 证明 preflight 和约束失败，不读取真实 DSN。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/migration ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store|Durable|Operation|Job)|TestTemporaryCredential|TestIdempotencyPasswordReset|TestMigration" -count=1`
  - `go test -race ./internal/identity ./internal/idempotency -run "TestPasswordReset(Store|Durable|Operation|Job)|TestTemporaryCredential|TestIdempotencyPasswordReset" -count=1`
  - `git diff --check -- migrations/0008_password_reset_durable.sql internal/identity/model.go internal/identity/temporary_credential.go internal/identity/temporary_credential_test.go internal/identity/password_reset_store.go internal/identity/password_reset_store_test.go internal/idempotency/store.go internal/idempotency/store_test.go`
  - 另执行 Manifest 固定七项；真实 `go test -tags=integration`/PG18/multi-connection 不在本卡运行责任内。
- **migration 边界**：仓库现役最大编号为 `0007`，故使用唯一前向 `0008_password_reset_durable.sql`，不得修改 `0001`～`0007`。升级顺序为停旧 writer/reader → preflight → `0008`/backfill/约束/index → 全部新 consumer/worker 装配与 Reconcile → 再开放 writer；一旦写入 operation/generation/lineage 新行，不支持旧 binary 回退，只允许前向修复。真实 PostgreSQL 18、升级/混跑、commit-ack-loss 与回退阻断证据归 S1.6。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`identity-auth-S2.1`、`identity-auth-S5.1`、`identity-auth-S5.2`
- **并行**：B6C-1 独占上述八文件；与当前 active S3.3/S3.5 写集零交叉。S5.1/S5.2 已关闭并释放 temporary credential/idempotency 历史写集；未知写者或需要第九文件时停止。
- **阻塞注记**：无。本卡固定八文件内的 planning、实现、适用自动化、非作者复审与 Lead Gate 已闭环；真实 PG18、多连接、commit-ack-loss 与 migration 混跑仍由 S1.6 独立验证，不回退本卡关闭。
- **关闭证据与限制**：八文件最终 SHA256 为 `37FB3077709462D62C763FE87D4E04B4BDA4CFC498C14C2860B27429D5036980`、`4D17E1DBC264C69BAFE3330619098F0C57AD703EC5EA5E611BCE8E2BB548B5C7`、`66F11A1BD5E390D996C440BF21126AB8740356C43A4CDA49DD2FE16B66B23DF5`、`659199B79A481A04DF414DFDC30C5B616455BD4466FA471FA8D92BC2BDA6279D`、`9FAE66FB1C9BF73405291FDFB0406044AC5E4390D690005D4A3B122656CD42CE`、`B316B55FCC7A7BFD2D10886643F974ECAA2E8CD4C2BF696DAB7C8D9A268CDA40`、`AA0483969B12B058B42DB70A9763501B60ED055AFB91E5ECD6AA7F5C45FDD657`、`3A6EF9313839ED19BE28D4FB635C016DF51F6C58213DDAE88D93A2A65CC49BE7`；Planning R03 与 Code Review R05 均 PASS，Lead 定向普通/race、Manifest 七项与 scoped diff-check PASS。真实 PG18/multi-connection/commit-ack-loss/migration 混跑、SMTP、Host、部署、E2E、容量与 production readiness 未验证。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S1.6；S10.3、S10.1、S10.4、S11.1 消费本卡 API/状态。
- **设计/测试来源**：identity design r40 §3.3.11/12、§3.4.1/2/8、§9.4；stories r16 S10；test-plan r18 I-PRD-02～05/12/16～18 与 `I-PRD-05.CP-01`、`CP-T01`～`CP-T05`、`CP-P01`～`CP-P04`、`CP-N01`、`CP-F01`

- [x] identity-auth-S10.3 接入 Login、PasswordChange 与 Session resolve durable Gate 开始@2026-10-24 完成@2026-10-24

- **目标**：在同一卡内完成唯一 non-nil Gate constructor、唯一 production `identity.PasswordResetStore` 实例及其 construction/wiring，并把同一非 nil Gate 强制注入 Login、PasswordChange、Session；保持 identity→session 单向依赖、r40 lineage 分类与 Session caller-owned transaction/commit barrier。
- **类型**：开发（共享认证消费者 + production Gate construction/wiring）
- **责任与状态**：13 文件实施与返工已完成；Design Review R02 确认 r42/F001 CLOSED、Design Review Gate PASS，原非作者 Reviewer Gibbs 的 Code Review R02 确认代码 F001/F002 CLOSED、无新增 finding、Code Review Gate PASS；Lead 于 2026-10-24 完成 task-level 核证并关闭本卡。实施者身份未在本轮固定证据中登记，Task-Planner 不作推断。S10.1/F002 状态不变；S10.4 已按 design r44 / test-plan r20 在精确八文件内完成，并于 2026-10-25 task-level CLOSED。
- **文件**：
  - `internal/identity/password_reset_store.go（修改）`
  - `internal/identity/password_reset_store_test.go（修改）`
  - `internal/identity/login.go（修改）`
  - `internal/identity/login_test.go（修改）`
  - `internal/identity/login_integration_test.go（修改）`
  - `internal/identity/service.go（修改）`
  - `internal/identity/bootstrap.go（修改）`
  - `internal/identity/bootstrap_test.go（修改）`
  - `internal/identity/administrator_create_test.go（修改）`
  - `internal/session/store.go（修改）`
  - `internal/session/store_test.go（修改）`
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
- **13 文件固定对象 SHA256**：`internal/identity/password_reset_store.go` `6C0136408EF62A7B5FC80C4C557AEBCB557457B772EB96E8DA90024C42D3315F`；`password_reset_store_test.go` `62EAE06BD41E5F3CFA7AAB03B3721B8D3A964834A0E7C98B194181A2791827E6`；`login.go` `FDB1ED13FB499A1219AE24884F053A3626567BB81CAAB56EC02B1FC197B6864C`；`login_test.go` `FB5483FB93649F9B01B060D54F35B5CC7B7A3305704D19DFCA445A2C7400414F`；`login_integration_test.go` `15C28D7BA7EE39499D970E8C7AE53F7CE28A1A450D2530FD7697BF8BE4D407BA`；`service.go` `4371ED7348A7035BDB6CAA57CE55F8621FF41CDE5D8D7E5615D54789F88A4F40`；`bootstrap.go` `AEF53946AB8C55A3CC03DAE03DC1CD0E42BE12553D9619D45C7F491F154CAAA5`；`bootstrap_test.go` `C33C31A22C32DA0FFE57099C37C3E6EE4826F4F082359F2B7A7E2B1CB43E2B45`；`administrator_create_test.go` `AC9FE098FA3F1F09FA4944BB5BD7653BF375B23C08ED1E63F13DCEE45683EEAE`；`internal/session/store.go` `4B12462E29D46B730F0DD58E6A0BAA57EC1D91B99D9F43FC91179319FF88E679`；`store_test.go` `3D32E65D407F24211355B642DAB3933F1AB36E10B60CAFC7F039B2F9B8201E09`；`cmd/han-server/main.go` `61ED0992F65CBD45CDEB524DE71C4D580A92A7AC1EB70BAF1F8A073DFC7D45F2`；`main_test.go` `9E6B1C6E2E3BE747E3055CDBE275815BC7976D7E5D2FCEF848718C864776BFEB`。
- **TDD ②档失败证据**：先覆盖 test-plan r19 `I-PRD-11/12/13/15.CW-*`：旧 constructor/signature 与缺 Gate 的可授权入口必须不可构造；nil Gate、nil transaction boundary、Gate construction failure 必须在 verify、业务 DB、listener、routes 与 Serve loop 前 fail fast。三消费者覆盖 clear/closed/query/commit error-or-unknown 与稳定错误映射，失败时零 credential verify/PasswordChange 写入/Principal；Session 证明 token 预读→begin tx→User lock→Gate→Session/credential 重读→必要 revoke→commit→Principal，只有正常 commit 成功后物化 Principal。production root 恰好构造一个 `*identity.PasswordResetStore`，三消费者观察同一实例且零第二 authority；三个包内未导出 test fake 不得复制 SQL/classifier或进入 production。合法 A→B/A→B→temporary C 不写 hold，瞬时 query/User lock/commit 故障只 fail closed且零新 hold。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity ./internal/session ./cmd/han-server -run "Test(Login|OnboardingPasswordChange|SessionResolve|PasswordResetGate|.*PasswordReset.*Construction|.*PasswordReset.*Wiring)" -count=1`
  - `go test -race ./internal/identity ./internal/session ./cmd/han-server -run "Test(Login|OnboardingPasswordChange|SessionResolve|PasswordResetGate|.*PasswordReset.*Construction|.*PasswordReset.*Wiring)" -count=1`
  - `go build ./cmd/han-server`
  - `git diff --check -- internal/identity/password_reset_store.go internal/identity/password_reset_store_test.go internal/identity/login.go internal/identity/login_test.go internal/identity/login_integration_test.go internal/identity/service.go internal/identity/bootstrap.go internal/identity/bootstrap_test.go internal/identity/administrator_create_test.go internal/session/store.go internal/session/store_test.go cmd/han-server/main.go cmd/han-server/main_test.go`
  - 局部失败证据转绿、13 文件完整且非作者 Review 后，另按同一 revision 执行 Manifest 固定七项；任一未通过不得关闭。真实 PG18/SMTP/部署/E2E 不属于该七项。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S4.1`、`identity-auth-S8.1`、`identity-auth-S10.2`
- **并行**：B6C-2；S10.1～S10.3 均已关闭并释放各自写集，S10.4 与 S11.1 也保持关闭。已关闭 S3.2/S4.1/S8.1 只提供回归基线，不重开。
- **关闭边界**：本卡未新增第 14 文件、schema/migration/依赖、公共 REST/错误码，未改变 identity→session 方向或 Session transaction boundary；未引入 legacy/optional/no-op/allow-all/deny-all Gate、第二授权入口、第二 Store authority 或不可部署中间 revision。后继若需要突破这些边界，仍须停止并回设计/Task-Planner。
- **关闭证据与限制**：13 文件内的唯一 constructor、唯一 production Store/Gate、三消费者同实例强制注入、全部直接 call-site、fail-fast、三消费者错误映射、Session caller-owned transaction/commit barrier与三处 fake 限制均完成；Design Review R02 与 Code Review R02 PASS。Lead 重跑 finding 专项、定向普通/race、三包普通/race/vet、Manifest 七项、scoped gofmt/diff-check，均 PASS；获用户精确清理授权的 `han-server.exe`（原 18,137,600 bytes，SHA256 `A92D062BC163A48277107F02D812F66AE77A213BBE707CB2CBFEB2F63EAF4145`）已删除且现场不存在。真实 PG18/多连接/SMTP/Host/部署/E2E/容量与 production readiness 未验证。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S1.6；S10.1/S10.4/S11.1 消费本卡已接线 Gate。
- **设计/测试来源**：identity design r42（SHA256 `CEF17810A42DECF35F0B9A0585877B6FDFEEF77557E2A314D6BF536D2940D176`）§3.3.1/4/11/12、§3.4.4/8、§6.8、§9.4.1、§10；stories r16 S10；test-plan r19（SHA256 `0381614EA06B34756C430989D04BAF0F8A55C5FAB24C713711535EB1D78AD0DE`）`I-PRD-11/12/13/15.CW-*`；Design Review R02（SHA256 `94E8C180D4961C6EB141806DFA9662566C8BEC07CB968D3ACCD8F6EAE7D868A2`）与 Code Review R02（SHA256 `40D264C2E694E9CD5596DAD3FC1C574385F4CC37B08A22CA3D3127AB4769287F`）PASS

- [x] identity-auth-S10.4 接入 password-reset internal operator、自动 worker 与双 worker production lifecycle 开始@2026-10-24 完成@2026-10-25

- **目标**：在精确八文件内复用 S10.3 production root 已构造并注入三消费者的同一 `*identity.PasswordResetStore`/Gate，交付 internal `PasswordResetOperator` port/实现、自动租约 worker、五项显式必填配置、唯一进程内 connection-permit owner及 Article/PasswordReset 双 worker production wiring/lifecycle；不交付 production maintenance 命令、入口或可执行 runbook。
- **类型**：开发（internal operator / worker / config / production lifecycle）
- **责任与状态**：既有 task，`[x]`、task-level CLOSED、开始@2026-10-24、完成@2026-10-25；唯一实施 owner 为 Backend-Dev Tesla，Agent/thread ID `01a17046-ead9-7223-8ae3-df73ae37ed8b`。非作者 Reviewer Pascal `01a17063-3cc5-73d3-8967-49995bb4d054` 的 R03（SHA256 `6CAC72E65FC85678CB0DAEF371367D4465EAB5ED41A3282A65DCD032E746D37C`）确认 R01 F001～F005、R02 F001 全部 CLOSED、无新增 finding、Code Review Gate PASS；Lead 已按 §10 核同一 revision 并关闭本卡。运行配置责任人仍待分配，五项生产实际值保持 TBD。
- **文件**（精确且唯一八个）：
  - `internal/identity/password_reset_worker.go（新增）`
  - `internal/identity/password_reset_worker_test.go（新增）`
  - `internal/identity/password_reset_operator.go（新增）`
  - `internal/identity/password_reset_operator_test.go（新增）`
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
  - `cmd/han-server/lifecycle.go（修改）`
  - `cmd/han-server/lifecycle_test.go（修改）`
- **最终八文件 SHA256**：`password_reset_worker.go` `9BF1A9B1230004422E9ABC563329C9C7E44BB48CC1E1452CB5DF18E20D704C90`；`password_reset_worker_test.go` `DC6544C4390CC36FC93ED4B66D530E2E0FBC067A7C733FFF6692944CBC4801CD`；`password_reset_operator.go` `4DC973ECB2272E07083884224A898D0C7C38E4CFEE8B03E44DA14E9D9F951F42`；`password_reset_operator_test.go` `9463D7B8B098A21BA3F68BDB190CFCF51C6FA47038BA0BD8A7E7505ADD210E02`；`main.go` `A0190C6F29E8DDFD43DB2066DF477F35D12E0CFD95D9E81919CF16514F57B966`；`main_test.go` `7990D7A8CAFCABA5CB50CD601417C33E5DE60F12E71D532E3A98CC2EB113E884`；`lifecycle.go` `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888`；`lifecycle_test.go` `B3AA659B1A98E0AE8A3EF09678309866DC3D46345B2A6C29A5A220D700B36DDF`。
- **TDD ②档失败证据**：
  - 五项配置 `HAN_PASSWORD_RESET_WORKER_LEASE_DURATION`、`HAN_PASSWORD_RESET_WORKER_RETRY_BACKOFF`、`HAN_PASSWORD_RESET_WORKER_MAX_RETRIES`、`HAN_PASSWORD_RESET_WORKER_CLAIM_BATCH_LIMIT`、`HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET` 均必填且无默认。每个键的 missing/blank/parse/overflow/zero/negative/内在范围错误均须在 DSN parse 前失败并断言 DSN parse 及以后调用为 0；仅已解析正数 budget `> pgxpool.Config.MaxConns` 可在 DSN parse 后、connect 前失败，并断言 parse=1、connect 及以后=0。
  - composition 必须证明三消费者、operator 与 worker 复用 S10.3 同一 Store/Gate 指针，operator/worker/permit owner 各构造一次；拒绝第二 Store/Gate、第二 composition root、隐藏 global/service locator、lifecycle 读取配置或构造 production dependency及第九个 production 文件。
  - 候选扫描、租约取得、每次续约、每个 settlement job 与 operator direct path 均恰好一次 acquire/release；permit 持有至 Rows/Tx/connection 完全释放，success/error/cancel/panic 全路径归还，helper 不得嵌套 acquire，peak 不超过 budget且例末归零。
  - Article 与 PasswordReset 的 `Start(context.Context)` 均无同步 error；同步阻止 Serve 只来自双 Reconcile 或启动前构造/配置失败。Start 后运行错误只经 `Done/Err` 上报；Stop/Wait/Shutdown/Close 使用 shared deadline并保持主错误优先，两个 worker 停止后才关闭 Article 资源，Serve/listener 后于 worker，DB pool 最后关闭。
  - internal operator 与 worker/replay 均不发送邮件；dead-letter 持续 Gate/hold。不得把 retry/settle/confirm internal 方法冒充 production maintenance 入口。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity ./cmd/han-server -run "TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle" -count=1`
  - `go test -race ./internal/identity ./cmd/han-server -run "TestPasswordReset(Worker|Operator)|Test.*PasswordReset.*(Config|Permit|Wiring|Lifecycle)|TestRunLifecycle" -count=1`
  - `go build ./cmd/han-server`
  - `git diff --check -- internal/identity/password_reset_worker.go internal/identity/password_reset_worker_test.go internal/identity/password_reset_operator.go internal/identity/password_reset_operator_test.go cmd/han-server/main.go cmd/han-server/main_test.go cmd/han-server/lifecycle.go cmd/han-server/lifecycle_test.go`
  - 另按同一 revision 执行 Manifest 固定七项：`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。
  - 当前状态：Tesla 的最终 revision 已取得专项普通/race、整卡定向普通/race及适用项目检查 PASS；Pascal R03 另取得三个 finding 专项普通/race/`-count=20`、整卡定向普通/race、八文件 scoped diff-check 与哈希核对 PASS。Lead 同一 revision 的专项普通/race、整卡定向普通/race、Reviewer 专项 `-count=20`、`gofmt -l .`、Manifest 七项、scoped diff-check、staged 与残留二进制核对均 PASS；本次 Task-Planner 仅同步既有证据，未重跑代码命令。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`identity-auth-S5.2`、`article-management-S4.6`、`identity-auth-S10.2`、`identity-auth-S10.3`
- **并行**：B6C-3；task DAG 为 `S10.2 → S10.3 → {S10.1,S10.4}`。S10.1/S10.4 均保持 task-level CLOSED。S10.5 继续等待稳定设计、测试规划、文件充分性复核与 Lead 启动授权。
- **阻塞注记**：本卡 task-level 阻塞已解除。运行配置责任人与五项生产实际值仍待后续容量/运行准备确认，均不由 Task-Planner 填值，也不外推 production readiness。未来若需要第九个 production 文件、改变 S10.3 Gate authority、公共 REST/错误、schema/migration、依赖、boundary、architecture 或 ADR-0023，仍须停止并回设计/Task-Planner。
- **本任务关闭条件**：已满足且不重开。精确八文件内的 internal operator、自动 worker、五项配置 parser、唯一 permit owner、同一 Store/Gate 复用与双 worker production lifecycle 已完成；integrated F001 已由 S10.1 两文件返工关闭，不改变本卡局部关闭证据。B6C-3 integrated Gate 已 PASS并由 Lead 接纳。
- **关闭证据与未验证边界**：R03 固定八文件哈希与上列一致，R01 F001～F005、R02 F001 全部 CLOSED，无新增 finding；Lead 确认暂存区为空且 `han-server.exe` 不存在。真实 PG18、多连接、commit-ack-loss、migration 混跑、SMTP、真实 Host/部署/E2E、容量与 production readiness 未验证，五项生产实际值仍为 TBD。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S1.6、identity-auth-S5.3、identity-auth-S3.4、identity-auth-S2.2；后继 maintenance 候选为 identity-auth-S10.5
- **设计/测试来源**：identity design r44（SHA256 `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23`）§3.1.4/7、§3.3.11、§3.4.8、§6.1/2/7/8、§9.4.2、§10；stories r16 S10；test-plan r20（SHA256 `F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB`）S10.4 配置/permit/lifecycle矩阵；Design Review R02（SHA256 `9A76D7E3C5BC29855043784057C99383844A0B3CC7F14E32F96EDBB5290F0434`）PASS

- [x] identity-auth-S10.5 设计并实现互斥 maintenance mode 的真实 password-reset 维护入口 开始@2026-10-28 完成@2026-10-28

- **当前关闭核证（2026-10-28，仅本开发卡）**：Lead依据既有建议授权作出[最终技术关闭决定](../../../../.spec/wip/identity-auth/evidence-s10.5/lead-technical-close-r49-r01/decision.json) SHA256 `FCFCAF1F19018488027D2D0F6E8CF27E028106C41D81F88427BE3FFD06037D31`，批准原S10.5/B6C-4软件task技术关闭。最终七源码摘要/范围见[Main最终核验](../../../../.spec/wip/identity-auth/evidence-s10.5/lead-scope-final-r49-r02/verification.json) `A558C17B36EAE92ECA1C59BD8B6226997F1EEE724179FE15FD5A3FA8C0101A25`；原Dalton三路径＋Wegener四路径不变，无新task/依赖/批次。
- **独立软件闭环**：Dewey[完整Root R01](../../../../.spec/wip/identity-auth/review-code-s10.5-root-r49-r01.md)及[修复复审R02](../../../../.spec/wip/identity-auth/review-code-s10.5-root-r49-r02.md) `56427E3CC2668AC15537A786EEEFE4D0BF5C373527470AAC1B3FEB07EC6B0985`、Laplace[Domain r49](../../../../.spec/wip/identity-auth/review-code-s10.5-domain-r49-r01.md) `385BA0CBD60F57BD2C24B84F969C59392176446FD6679F75DE79595CD2B0A57B`及[carrier R01](../../../../.spec/wip/identity-auth/review-code-s10.5-carrier-r49-r01.md)→[R02整体软件复审](../../../../.spec/wip/identity-auth/review-code-s10.5-carrier-r49-r02.md) `895717E4C557F3ADD430C57732546EBF878B20E8EBDB1BAEF70A71580EF986C4`合用，Root F001～F004及carrier F001 CLOSED、两轮新0，C18/C16必需层SATISFIED，connected-seven软件NO FINDINGS WITH LIMITS，无确认剩余必需软件缺层。[R02证据索引](../../../../.spec/wip/identity-auth/evidence-s10.5/carrier-r49-review-r02/index.json) `679841980AF7262A4A4C59E573B6D3BCE911B38DB852EA476D191173DCDD6821`；不是把局部PASS相加或签发全项目CodeGate。
- **实际最终U/PG/B**：当前carrier为`D4B0C837F27E2654784C779A3C08065F0CE4ECCE78B4ED63072B69125DEA2204`；[作者新证据](../../../../.spec/wip/identity-auth/evidence-s10.5/pg-b-author-r49-r02/index.json) `5CC384886105AF867D31665FCB19334BF73789CBAC8A78859F04425C3F410A81`及[Main现场核证](../../../../.spec/wip/identity-auth/evidence-s10.5/pg-b-author-r49-r02/lead-onsite.json) `1178E4ED262BDD87CD7A9F8D17A7858F1B9679899D3AA2847FAE5737B939F9FA`保留normal/race harness各126unique/114leaf、0fail/skip、11非空前缀，286child/writer实际join、0Kill、14新schema各9迁移全retain。真实ACK-loss5→fresh replay0/无额外写；两状态真实worker/CLI User锁重叠、许可归还，JSONB UTF8 text65536/Get1与65537/Get0均无业务写。F001四个Fatal诊断按预期exit1，旧资源1/新0，非整条go test PASS；旧D452/110leaf仅历史。
- **项目检查及限定例外**：六非format Manifest原收据对应当前未变compiled六源码，均exit0；最终integration-only carrier另有compile/vet和上述normal/race证据。全仓gofmt实际失败于immutable证据的overlay/空前像；实际交付7源码gofmt净，采用限定例外，见[Main最终格式收据](../../../../.spec/wip/identity-auth/evidence-s10.5/lead-format-final-r49-r01/verification.json) `1774D6397088421A13A6D395B19893440342208E55B6EA8273EB5F469A9CB3F6`。保留原FAILED2，不固定证据路径数量、不写全仓格式或所有七命令PASS；未清理/格式化证据，不因仅新增证据重复广测试。
- **关闭范围与历史**：本开发软件验收已建成，保留r49合作取消/强制实际join及不可取消阶段可越期乃至不返回的代价，无硬进程SLA。TLS-enabled ACK-loss、实际child race、直接reset/resend HTTP并发、全SQL/OS线程普遍证明未验；实际launcher/ACL/key签发、第二真实SID、撤销/恢复epoch及C24/25部署/E2E仍原S3.4/B12 NOT RUN，人的accepted/checkpoint不代签。下文旧规划/认领、F001 OPEN、仅编译/PG NOT RUN、CODE HOLD及待满足关闭条件/命令记录保留各自时点，当前状态与结果以本关闭段及正式证据为准，不把旧失败改记PASS。

- **目标/类型**：开发；在同一 `han-server` 的唯一 root 实现 `maintenance password-reset retry`，以 Windows 实际进程 owner SID＋独立注册 approver Ed25519 签名，只对合格 dead-letter job 做 User-first 审计化重新排队。一个审批/requestID 最多一次已提交 enqueue，不承诺一次自动 attempt；不交付 settle/confirm 或第二安全结果 classifier。
- **责任与状态**：原task / B6C-4保持 `[-]`、开始@2026-10-28；原binary行为RED已获Lead独立接收、两组CODE曾实际启动，均保留历史。主责Dalton `01a18135-18b1-7431-b0d9-4e225d0f19d7`仅operator两文件＋integration carrier三路径；协作Root Wegener `01a181a4-16ec-7df1-b3c7-c78cb2ac9bed`仅main两文件＋Win/!Win adapter四路径，互不代写。当前两作者已STOP/写窗口关闭，新r49修订须待Lead核本次Tasks/Plan实际diff/DAG/hash后实际另授CODE恢复；本次规划不代发信号。两方静态G0/旧局部U/Review不证明新合同实现；Root-ready、组合CLI/PG及整卡CODE验收仍HOLD，人的accepted/checkpoint不预签。
- **精确文件所有权**（恰七路径均已存在；原登记4现有修改＋3新增，现188源码口径含6个testdata JSON；按路径单写者，不是全七文件锁）：
  - `internal/identity/password_reset_operator.go（修改；Dalton）`
  - `internal/identity/password_reset_operator_test.go（修改；Dalton）`
  - `test/integration/identity/password_reset_maintenance_test.go（新增；Dalton）`
  - `cmd/han-server/main.go（修改；Wegener）`
  - `cmd/han-server/main_test.go（修改；Wegener）`
  - `cmd/han-server/maintenance_stdin_windows.go（新增；Wegener）`
  - `cmd/han-server/maintenance_stdin_other.go（新增，显式 !windows；Wegener）`
- **实现/API边界**：root先判mode/argv，维护零普通server构造、listener/Serve/worker/Reconcile/Start/migration/bootstrap/mail/Journal及领域SQL；普通server/非Windowsserver保持原合同。协议/七ENV只归ADR；四字段 `PasswordResetMaintenanceConfig` 为 TrustJSON/MaxTTL/OperationBudget/ShutdownBudget。`NewPasswordResetMaintenanceOperator(config,clock)` 自取真实SID、私有复制trust；`ValidateRequeue(ctx,envelope)` 返回绑定原P/signature/较早绝对D/实例的opaque request；`Requeue(ctx,request,dependencies)` 显式Transactions/Store/audit.Store/Permit，再验授权，不信请求role/ENV用户名或可替换resolver。普通构造/Execute不变。
- **授权/数据与状态合同**：六非DSN配置、实际SID/签名/时效及有界stdin成功后才读DSN；七键、strict normal、正int32/唯一MaxConns post-parse负例、ShutdownBudget 1s..60s按r22矩阵。显式ReadCommitted，User→operation/job锁内先scalar全局UUID长度/绑定guard，合格existing才text Get，absent不Get；PK碰撞rollback不fetch winner/开新tx。仅job.status/nextAt/updatedAt与同tx真实Audit ID=requestUUID；完整before/13string metadata/原receipt/after保真，零User/operation/credential/Session/业务幂等/hold/终态写、不发信/秘密/伪lease。Commit错误统一unknown/5，不复制settlement commit classifier。
- **输入/资源合同**：r49保留串行Peek仅正available才有限native ReadFile、真实EOF/取消优先、输入reader goroutine/cancel callback均0及无pending IO后一次Close；匿名本地pipe/全部read dup唯一consumer及无同file-object并发IO仍是launcher前提。原D停新授权/Acquire/连接交接/Ping/Requeue/业务；S=ShutdownBudget、K=D+S、每次C=min(cleanup开始+S,K)不增加、不重起TTL。C/K为合作取消与可取消I/O中断截止，撤回旧r48全部SDK/进程硬完成保证：已开始不可取消CPU/native须实际join，可能越期、下层不返时进程也可不返，禁detach/提前return/parent Kill冒清理。Root/Operator任一cleanup超时或最终单调end>C粘性记录；实际join后原成功/原ctx取消转typed maintenance.failed/6、stdout空，unknown5优先，其他非零主错误保持；明确commit/原Audit事实不改，零新增核验tx/timeout Audit/补写。唯一owner登记全部constructor/阶段/SDK/transport，自有callback/timer实际join；健康先SDK Close→CleanupDone→同步Pool.Close。TLS三setter含clear/later仍裁同C、同步SDK Close期间中断并join，CPU例外不豁免额外5s/SDK15s网络窗口。
- **维护DSN支持面**：authority/六非DSN成功后才读parse DSN；parse后、connect前核primary及全部fallback Host均为数值IPv4/IPv6，hostname/隐式localhost/其他非字面量invalid_config/2、parse=1而OS Lookup/Dial/connect/ping/业务0。维护使用本地literal Lookup与numeric Dial，不调用DefaultResolver，不先解析hostname后冒numeric；正budget>MaxConns是独立post-parse负例。七ENV、四Config、公共Go API/wire不变；普通server hostname、TLS验证/SNI不变，部署IP/证书匹配留S3.4。
- **同卡TDD/并行顺序**：保留原185源码构建binary的有效argv路由RED（实际exit1/新预期exit2）及Lead核证后两组CODE启动历史；现有han-server.exe非默认基线、缺符号编译失败不算RED。新r49 overlay RED仅为domain scriptedTx行为失败输入，见下段。后续须Lead实际CODE恢复→原七路径Root4与Identity2＋carrier按不交叉所有权修订→API-ready冻结＋Root-ready并由Lead实际给combined CLI/PG信号→组合验证/独立最终Code Review与Lead同revision核证。内部流水不新增Task requires；当前作者STOP，Root-ready/组合CLI-PG HOLD，本次写账不恢复实施或运行。
- **验收覆盖**：最终r23 §4.4.3保留I-S105-01～25与§10.4三个已部分存在的top：`TestPasswordResetMaintenanceRoot`、`TestPasswordResetMaintenanceOperator`、`TestPasswordResetMaintenancePostgresCLI`；原五条选择命令、兼容回归及Manifest七项不变，r23合同验收NOT RUN。U/真实SDK-peer子层/B独立生产binary/PG真实数据/D部署各留证，不将helper-process、scriptedTx、只编译或旧局部PASS当完整Case。第三carrier恰11指定前缀：`I-S105-06_half_no_eof`、`I-S105-06_complete_no_eof`、`I-S105-06_blocked_producer`、`I-S105-06_positive_bytes_no_eof`、`I-S105-16_oversize_uuid_guard`、`I-S105-17_absent_guard_uuid_race`、`I-S105-22_shutdown_1s_socket_cleanup`、`I-S105-22_applied_idle_sdk_close`、`I-S105-22_replay_idle_sdk_close`、`I-S105-02_05_numeric_endpoints`、`I-S105-19_one_enqueue_max_retries_2_to_5`。精确Peek-positive→Read前取消只归Root U；第三carrier正字节未EOF只证B普通pipe，numeric前缀替代旧mandatory late-Lookup占位，须真实断言而非删阻断/空叶/skip/fatal placeholder。guard/PK竞争由真实PG adapter/observer补内部顺序，B退出不独证Get或OS Lookup计数；健康apply/replay外部SQL/无backend与Root U/SDK关闭因果互补。原1.5s测量不作通用CLI硬返回SLA。C19真实worker Claim/Renew证明MaxRetries2→5可自动3/4/5，一审批仅one enqueue；C24/25仍归S3.4，NOT RUN。
- **验收命令**（workdir：仓库根；仅未来获准后执行，本轮未运行）：
  - `go test ./cmd/han-server -run '^TestPasswordResetMaintenanceRoot$' -count=1 -v`
  - `go test ./internal/identity -run '^TestPasswordResetMaintenanceOperator$' -count=1 -v`
  - `go test -race ./cmd/han-server ./internal/identity -run '^TestPasswordResetMaintenance(Root|Operator)$' -count=1 -v`
  - `go test -tags=integration ./test/integration/identity -run '^TestPasswordResetMaintenancePostgresCLI$' -count=1 -v`
  - `go test -race -tags=integration ./test/integration/identity -run '^TestPasswordResetMaintenancePostgresCLI$' -count=1 -v`
  - `go test ./internal/identity ./cmd/han-server -run 'TestPasswordReset|TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1`
  - `go test -race ./internal/identity ./cmd/han-server -run 'TestPasswordReset|TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend' -count=1`
  - 同revision顺序执行Manifest七项：`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。carrier构建同一cmd binary必须显式-o新获准retained Temp，不覆盖既有han-server.exe；harness带race不代表子binary带race，保留实际参数/digest。
- **未来环境/停止条件**：Dalton条件PG权限保留，但须实际API-ready＋Root-ready＋Lead combined信号后才执行；Root无PG。仅用原已核实literal loopback:5433 / PG180006的Windows User DSN载体，单次exec进程内安全注入、不读写secret文件/argv/log；新retained schema、全9迁移、合法fixture/独立observer及内存合成私钥，无旧schema/共享fixture写入。禁止DROP/autoDROP/TRUNCATE/禁trigger、账户/roles/GRANT、实际邮件/部署、覆盖既有han-server.exe；构建仅获准Temp。首轮缺process DSN的version为NOT CHECKED，只有followup预检观察180006，不是Han Case/DDL PASS。第八源码、新依赖/公共wire/共享authority或未知writer即停止受影响范围交Lead，不扩恢复主线。
- **依赖**：requires `identity-auth-S10.1`、`identity-auth-S10.4`
- **阻塞/当前Gate**：本轮消费六对象r04/r31/r49/r23及[Dewey两轮FINAL R03](../../../../.spec/wip/identity-auth/review-design-s10.5-r03.md) SHA256 `103407164E0DFF2ADE132DF6CA321A38CB3BFCAA742C6925DBC5719F786CD66A`，Design/ADR/TestPlan PASS、0 finding，仅设计/规划输入。Lead已按既有建议授权[实际采用整包新合同](../../../../.spec/wip/identity-auth/evidence-s10.5/cleanup-contract-adoption-r01/lead-adoption.json) SHA256 `3ED915618A722E910835FB39D990DAF932786A4131EE6E87B131F8238ED5B84A`，不是waiver、旧r48已修复或人的accepted/checkpoint。当前F001 OPEN，F002限定关闭见下；作者STOP，Root-ready/组合CLI-PG HOLD，CODE恢复待Lead现场核本次两diff/CAS/DAG/命令/所有权后实际信号，不新增独立Planning PASS门禁。原[规划现场接收](../../../../.spec/wip/identity-auth/evidence-s10.5/planning-r01/lead-onsite.json)及[旧binary路由RED现场核验](../../../../.spec/wip/identity-auth/evidence-s10.5/red-r01/lead-onsite.json)保留历史，source185未变只指旧RED基线；没有r49实现/完整Case/PG/Manifest/完整Code Gate/E2E PASS，dead-letter仍按原Gate，不当作已可维护。
- **domain局部独立评审（历史固定对象）**：非作者Laplace `01a181f9-2016-7201-9ce7-19075f113dfa` 两轮[domain报告](../../../../.spec/wip/identity-auth/review-code-s10.5-domain-r01.md) SHA256 `EDCBB4CC158A307B4A31621F5C8BC10AABCEAFB4AFC26AB77E4DC21A0AE3ED65` 为NO FINDINGS WITH LIMITS；Reviewer普通/race各126nodes/114leaves、failure/skip0，限当时r48合同下operator `8FB6ADAA6F32F0054E6C09D13157AAFE549D615DD89CCC5DDE14701F0791E1E9` / test `CA1A05BA8C2737E50B8ACB6693C6022D9E12BBC20DE0FB7838CBFFC068336BF6`。原[domain索引](../../../../.spec/wip/identity-auth/evidence-s10.5/domain-review-r01/index.json)保留，不能充r49退出/cleanup分类或七路径Code Gate通过。
- **Root阻塞及限定关闭**：原非作者Dewey [cleanup R01](../../../../.spec/wip/identity-auth/review-code-s10.5-cleanup-r01.md) SHA256 `8C1063321915846CA299E53DBC58CF037507EFE8015819A2E1EF84599AC0359A` 的两HIGH OPEN为历史；[cleanup CODE R02](../../../../.spec/wip/identity-auth/review-code-s10.5-cleanup-r02.md) SHA256 `CB2BB815A36E7EE7393BB80F1BCD6D54047214C79ECD377F7783EF42E59C3580`仅将`S105-CODE-CLEANUP-R01-F002`在main `2C2D404F344E5A8E4831C34FB4C240B234B9B099482F13897D38EA470B37FF79` / test `D46BA87371795849FD525454FB4E67942E7C3002051867B3F335D55EBE0E83B2`固定网络/TLS修订CLOSED、新0，不外推后续源码或完整CodeGate。`S105-CODE-CLEANUP-R01-F001`仍OPEN，须按新采用合同实际修订/验收并由原Reviewer代码复查；原Ohm Phase A咨询已被Lead后续实际采用，不再是“未采用”。[原SDK gap证据](../../../../.spec/wip/identity-auth/evidence-s10.5/root-gap-r01/index.json)与[作者gap交接](../../../../.spec/wip/identity-auth/evidence-s10.5/root-gap-handoff-r01/index.json)保留原FAIL/raw；cutoff仍活观察不改记r48 PASS。早期98条/race、Windows/Linux编译exit0与F002局部SDK/TLS结果不等于当前Root矩阵或r49通过；当前Root-ready仍HOLD，无PG/DDL/组合CLI/Manifest/E2E。
- **R对账/新失败输入**：七路径已有部分代码，维护增量由原“待做”改为“需改”，不标建成/完成；当前188源码（原185＋原授权3新增，含6个testdata JSON），原四修改外181保持，旧exe/mod/sum/Manifest控制不变只作本次边界核对。Dalton[新r49 overlay RED](../../../../.spec/wip/identity-auth/evidence-s10.5/r49-domain-overlay-red-r01/index.json) SHA256 `1DF2B52B40882398B88CE9571D7CEAD35783663E167B8C6759D81C020846134A`已编译执行：scriptedTx Commit=nil后取消原ctx仍nil/applied，预期typed failed6；Begin1/Commit1/Rollback0/Exec2、permit实际归还，unknown5对照通过。仅作者U层行为修复输入，不是非作者新CODE finding/真实PG提交或CLI结果；carrier当前BF9F仅编译，原完整r23/B/PG/D仍未验。
- **测试规划登记回填现状/历史**：原TE Russell对r22 §9.5四处元数据回填已完成，live当时SHA `86B8C4026643F4B3B7876077510FDC29103D12952E7CF5FEB850D7437D5035C4`及[registration索引](../../../../.spec/wip/identity-auth/evidence-s10.5/registration-r01/index.json)保留。现最终r23 `4CDC882308A2CCF6EE249880EBA5EEA469C2C983933C4EE6D42AE1168F5F03D2`沿用§9.5已登记Task/B6C-4/B12映射，25Case/三top/五命令不新增；本次只对账原Tasks/Plan，不重复写TE元数据、不静默替换历史E431受审快照或R02。新两前缀/真实层次见r23 §10.4，Case运行仍NOT RUN；冻结前R03 pending文字按其时点，实际Review只取正式R03报告。
- **本任务关闭条件**：七路径按r49/r23修复、有效行为RED→GREEN、适用U/SDK-peer/真实CLI/PG/独立SQL与原消费者兼容、Manifest七项、原F001新代码独立复查及完整非作者Code Review/必要复审、Lead同revision hash/status/diff/范围/结果与适用接收齐备。验收D停业务、C/K合作中断与全部实际join、Root/Operator粘性单调overrun6/unknown5优先、明确commit/Audit不补写、全部numeric端点及普通server/TLS不变；不拿旧硬cutoff失败直接改绿，不以局部Review相加代整体。真实ack-loss无可信注入则对应子项NOT RUN，U不冒PG，缺口如实留阻断。C24/25未跑不倒挂开发关闭，也不证明生产可用。
- **后续验证关联**：原 `identity-auth-S3.4` / B12承接I-S105-24/25，原S1.6/S5.3/S2.2边界保留；S11仅回归不重开，不实现/豁免全局RecoveryGate、CAS或Journal。
- **设计/测试/决定来源**：[design r49 §3.3.11/§9.4.3](../design-workspace/identity-auth/design.md) SHA256 `E884C84F4AABCA7ED321B9E4410EBAAFA0ED39CE6458986F92EA7C07D9BE4C43`；[S10 r49 Story增量](../design-workspace/identity-auth/stories.md) `8B2F13A7BA112C798E750E54AD9BFC7C9D4F20C88DB9BB2662CD43245D3FC6F7`；[test-plan r23](../design-workspace/identity-auth/test-plan.md) `4CDC882308A2CCF6EE249880EBA5EEA469C2C983933C4EE6D42AE1168F5F03D2`。ADR-0026 r04 `C35E7DC60004BBE6CE530AB0ADDF50A93FDDDCFA802D573E6B84FAFC5D96D321`、README `59217A3621A10003B66630DFA1F0AC43E84A91B64CBE7AAE786D290BFF1B8DC0`、architecture r31 `6C9BD31DF00F91198AA66AC650240707D89D55E4257AC1272A5A2B3059C81F22`，六对象均见[contracts-fixed-six](../../../../.spec/wip/identity-auth/evidence-s10.5/design-r03/contracts-fixed-six.json) SHA256 `478E11DB183DB40B53BF3980BF897753FD8E50331992A4B468455A8A0A4E5BF5`及[165payload包](../../../../.spec/wip/identity-auth/evidence-s10.5/design-r03/index.json) `BC909641BA3E07EDE3F70BA8ABCCA903367480A9822509D3AB9253BA81E97D2D`。正式R03/采用来源见当前Gate；历史[Design R02](../../../../.spec/wip/identity-auth/review-design-s10.5-r02.md) `0EC5613B883F4853AF72A6A7324FE9E0E2661450F3305C61DDD2DCF5CF5BDCED`与[旧六对象E431输入](../../../../.spec/wip/identity-auth/evidence-s10.5/design-r02/inputs-final.json) `411BD4A8B5ED9895CEEAFA4E9C487F195BC471706554FC23CC6AD764015AC300`原件不变，仅属r48/r22历史，本轮不写六对象/Review/evidence。

- [x] identity-auth-S11.1 实现临时密码重发与 generation 替换 开始@2026-10-25 完成@2026-10-25

- **目标**：在 B6C durable primitive/Gate/worker 已关闭后实现 purpose router；保持 `new_administrator` 的 ADR-0011 residual-window，并为普通 `password_reset` 建立 successor operation/execution generation、predecessor/replaced lineage、same-key replay 与 stale candidate 收敛。
- **类型**：开发
- **责任与状态**：B6D / S11.1 已由 Lead 于 2026-10-25 关闭。唯一实施 owner为 Identity Backend-Dev Curie `01a17092-626f-76d1-93fd-768c305674f1`；原非作者代码 Reviewer Copernicus `01a170a5-0cce-76f3-8354-22090be689a3` 的 R03 `.spec/wip/identity-auth/review-code-s11.1-r03.md`（SHA256 `A61827BEA22419D9D6CB7C64571C29F9E569D4E0BF29F845B477A8E38168D72A`）确认 R02-F001 CLOSED、R01 F001/F002/F003 保持 CLOSED、0 新 finding、`Code Review Gate: PASS`。S10.1 后曾因 S5.3 G0 的共享 coordinator delivery classification 缺口独立重开，现已由 R08/Lead Gate 再次关闭；S11.1 始终未重开，仅作为真实 resend 消费者纳入回归。S10.4 保持 CLOSED；S10.5 不构成本卡业务硬依赖且不得启动。
- **文件**（精确且唯一六个）：
  - `internal/identity/password_reset.go（修改；仅承接本卡 resend 权威前置校验与可持久重放 expiry 事实）`
  - `internal/identity/handler.go（修改）`
  - `internal/identity/temporary_credential_resend.go（新增）`
  - `internal/identity/temporary_credential_resend_test.go（新增）`
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
- **文件所有权与消费者**：`password_reset.go` 的本卡授权严格限于 resend：在 secret/ID/hash 前完成 writer Gate、current predecessor、stale 与 hold 的权威校验，并让首次成功与 same-key replay 使用唯一可持久重放的 expiry 事实；不得改变首次 password-reset 公共合同、worker/operator/Store、schema、migration或composition。`main.go` 只注册 design r44 既定的 exact `POST /api/v1/admin/users/{id}/password-reset/resend` 与 `POST /api/v1/admin/administrators/{id}/temporary-password/resend`，并把现有唯一 PasswordReset coordinator 与现有唯一 `AdministratorCreateCoordinator` 注入 purpose router/handler；其余三个 identity 文件实现 purpose 分流与 generation/lineage 行为。复用同一 `PasswordResetStore`/Gate/operator/worker和既有 Mailer/Audit/IDs/Clock/Hasher/Policy/Idempotency 配置，不得建立第二 authority、coordinator、composition root或隐藏全局。
- **TDD ②档失败证据**：先定点覆盖 R01 三项 finding。F001：missing/stale/consumed/unresolved/hold 等无效新 key 必须在 secret/ID/hash 前停止，并保持零持久写、零邮件、old capability 不变。F002：使用可前进 Clock 证明两种 purpose 的首次成功与 same-key succeeded replay 返回完全相同的公共 expiry，replay 仍零 secret/ID/hash/mail/业务写入。F003：生产测试须经过真实 `TemporaryCredentialResendService`/coordinator 授权路径覆盖 member、普通 admin、initial/current admin 角色矩阵，并分别验证 body/header/query 不能串改 route-derived purpose、稳定 403/错误 envelope与零业务下游调用；test seam不得替代业务授权证据。其余 I-PRD-01～03/06～10/12/19、generation/lineage、stale candidate、竞态及两条 exact route、middleware、404/405、同一依赖复用回归保持。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/identity ./cmd/han-server -run "TestPasswordReset|TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend" -count=1`
  - `go test -race ./internal/identity ./cmd/han-server -run "TestPasswordReset|TestTemporaryCredentialResend|TestProductionTemporaryCredentialResend" -count=1`
  - `go build ./cmd/han-server`
  - `gofmt -l internal/identity/password_reset.go internal/identity/handler.go internal/identity/temporary_credential_resend.go internal/identity/temporary_credential_resend_test.go cmd/han-server/main.go cmd/han-server/main_test.go`
  - `git diff --check -- internal/identity/password_reset.go internal/identity/handler.go internal/identity/temporary_credential_resend.go internal/identity/temporary_credential_resend_test.go cmd/han-server/main.go cmd/han-server/main_test.go`
  - 当前状态：Lead 已在最终六文件哈希上取得专项普通 `-count=10`、专项 race、identity/server 专项普通/race、identity/server 完整普通/race、`go build ./cmd/han-server`、六文件 gofmt/whitespace、Manifest 七项固定顺序（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）全部 PASS；staged 为空、无第七实现文件，`han-server.exe` 保留。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S4.1`、`identity-auth-S5.1`、`identity-auth-S10.1`、`identity-auth-S10.2`、`identity-auth-S10.3`、`identity-auth-S10.4`
- **并行**：B6D 已关闭并释放六文件独占窗口；`password_reset.go` 是本卡当时对已关闭 S10.1 成果的顺序消费和有界修改。S10.1 后续 classification repair 也已关闭并释放五文件窗口；S11.1 未重开且未复制分类逻辑。S10.5 不构成本卡业务硬依赖且不得启动。
- **阻塞注记**：无。未引入 migration、第七实现文件、公共合同/architecture/ADR变化或第二 authority/coordinator/Store/Gate/composition root/global。
- **本任务关闭条件**：已满足。R02-F001 CLOSED，R01 F001/F002/F003 保持 CLOSED；敏感准备前权威拒绝、首次与same-key replay稳定重放同一expiry、真实业务授权矩阵与body/header/query purpose防串改，以及其余purpose/角色授权、generation/lineage、两条exact POST route、production composition、middleware/Idempotency/envelope/error/secret safety、404/405零下游调用及同一coordinator/dependency复用合同均由R03与Lead最终Gate确认PASS。
- **关闭证据与限制**：最终SHA256：`password_reset.go`=`F1BB31B3726E19F100184C80AC517243584F6A9299995C682DAB64285D80C439`、`handler.go`=`29961F44AD74A8380C5DF8187359456066EF729666F7DF4ED5AE776D5AAFCD18`、`temporary_credential_resend.go`=`EF7F9CBC121A18040BAA937E9217C53CE961C7072166019AB6863B4B1E9AD50C`、`temporary_credential_resend_test.go`=`B579E97953DD258E5B98B75778879C092206053A65609595BFBF33C859372E68`、`main.go`=`1895BAE07E222004CE5191779A0880C008B6B00A53C039826B58AA331D2241C8`、`main_test.go`=`4EA1BDE6B6FAF4B79028CE0816130AE2E92545570C3A6C4790E29AA2353EF861`。未运行真实 PostgreSQL/SMTP/部署/E2E，未读取或使用 `HAN_TEST_POSTGRES_DSN`，本关闭不声明 production readiness。
- **后续验证关联**：identity-auth-S1.5、identity-auth-S1.6、identity-auth-S5.3、identity-auth-S3.4、identity-auth-S2.2
- **设计/测试来源**：identity design r44（延续 r40 lineage）§3.3.11/12、§9.4；stories r16 S11.1；test-plan r20 I-PRD-01～03/05/06～10/12/16～20；I-CRED-06 仅回归 `new_administrator`

## article-management（8 Story）

### Story S02 / `article-management-S2.1` R01→R02 关闭对账

- **R01 历史 gap-map**：Story S02 当时唯一标签为“需改”。已建成——三文件内的 keyset、隐藏密集区持续扫描、作者状态批读/fail closed、列表/详情投影、统一不可见 404 与局部测试；待做——0；需改——R01 F001/F002；废弃——0。局部测试 PASS 不抵消当时的 production 不可达与错误合同不兼容。
- **对账决定**：复用且仅复用既有 `article-management-S2.1`，新增 task=0，不改 requires/DAG。F001/F002 已在同一四文件返工闭环由 R02 与 Lead Gate关闭；已关闭 `article-management-S4.6` 未重开。
- **评审事实**：R01 `.spec/wip/article-management/review-code-s2.1-r01.md`（SHA256 `36BB65973CDC5998567B57B8C10AED1F6B65E114DEB5560414B19179540EF129`）Gate=`CHANGES REQUIRED`，F001 HIGH/F002 MEDIUM 的失败历史保留。R02 `.spec/wip/article-management/review-code-s2.1-r02.md`（SHA256 `01373C88B3E706A0EB0573BEEAFB67668E8C6537061EB6BE8F6FCB86D4CA8D4F`）确认 F001/F002 CLOSED、0 新 finding、Code Review Gate PASS；R01 原 Reviewer Agent ID 不可恢复，由已知非作者 Faraday `01a16940-6120-7021-bd06-eb0089956486` 替代复审并记录原因。

### Story S04 production wiring R 对账（design r10 / stories r07 / test-plan r08）

- **四标签 gap-map**：已建成——S4.1 的永久删除协调、Deletion Journal 与 finalizer 核心以及 S4.6 的唯一 production composition/router、三条 DELETE、Journal provider、Article 私有平台锁适配、worker/lifecycle 接线与确定性测试均已 task-level CLOSED；待做——0；需改——0；废弃——0。真实 PostgreSQL、受支持 Unix 持久卷、Host/部署/E2E 与容量仍由独立后继任务验证，不回退代码级关闭。
- **对账决定**：既有且唯一 `article-management-S4.6` 已完成单一 production wiring 开发闭环并关闭；`article-management-S4.5` 已关闭合成运行输入与机械验证，既有 `article-management-S4.3` 与 `article-management-S1.4` 继续分别关闭真实 PostgreSQL/Unix 持久卷韧性与部署后 E2E，不创建第二个 wiring task。
- **planning / code findings**：Rawls r04 已确认固定 test-plan r08 的 Test-plan Gate PASS；Hypatia r02（SHA256 `5625061C2963AF1CF29C44D2E1DD15A53FA4F911E5F8CD424BE459681755639E`）确认 `ARTICLE-S4.6-PLAN-R01-F001/F002` CLOSED、Planning Review Gate PASS；Faraday r02（SHA256 `A51DA3D4C6EE031DE121310D0332DD228268EE28D1F23F5963951064E982CDEF`）确认 `B6A-G4-R01-F001` CLOSED、无新增 finding、`B6A G4-batch Gate: PASS`；Faraday r03（SHA256 `007C2D9B04F1DC9381C574A668FB8C513F4B34D2B8224808F0512B623573E7E1`）确认代码 F001～F004 全部 CLOSED、无新增 finding、Code Review Gate PASS。上述关闭不证明真实环境专项或 production readiness。

- [x] article-management-S1.1 建立 Article、通知、删除与恢复证据数据模型 [P] 开始@2026-10-22 完成@2026-10-22

- **目标**：建立 Article、通知、删除与恢复证据数据模型。
- **类型**：开发
- **B3 认领/责任**：Backend-Dev Agent `01a16123-899c-7d33-938f-ebc43c3a8ca7`（Lagrange）完成实施与返工；非作者 Reviewer Agent `01a16132-31b6-7363-b1f4-96939990a115`（Mendel）关闭 F001～F005 并复审 PASS。Lead 核固定四文件哈希无漂移，PostgreSQL 18.6 integration、integration-race、migration integration、vet、build、gofmt 与 diff-check 均 PASS，并于 2026-10-22 关闭本任务。
- **文件所有权**：Lagrange 在 B3 内唯一拥有下列四个文件；其余 B3 卡只读。发现未知写者或必须修改清单外文件时停止受影响写入并交 Lead。
- **文件**：
  - `migrations/0003_article.sql（新增）`
  - `internal/article/model.go（新增）`
  - `internal/article/repository.go（新增）`
  - `internal/article/repository_integration_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article`
  - `go test -tags=integration ./internal/article`
  - 当前状态：PostgreSQL 18.6（`server_version_num=180006`）真实集成验证、Lead 检查与原 Reviewer 复审均 PASS；固定对象无漂移，本任务已关闭。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：Article 正交状态、TransferNotice/Recipient、DeletionOperation、DeletionReplayRun/Evidence 约束与索引在真实 PostgreSQL 通过；独立 Review 通过。
- **后续验证关联**：article-management-S2.2、article-management-S4.3
- **设计/测试来源**：article design §3.3.1 数据结构；article test A-CREATE/A-DEL/A-REC

- [x] article-management-S1.2 建立文章策略、cursor、强 ETag 与公共 REST 合同 开始@2026-10-22 完成@2026-10-22

- **目标**：建立文章策略、cursor、强 ETag 与公共 REST 合同。
- **类型**：开发（共享 article 基础）
- **文件**：
  - `internal/article/policy.go（新增）`
  - `internal/article/model.go（修改）`
  - `internal/article/handler.go（新增）`
  - `internal/article/policy_test.go（新增）`
  - `internal/article/cursor_test.go（新增）`
  - `internal/article/etag_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestVisibility|TestCursor|TestETag|TestArticleErrors"`
  - 当前状态：Lead 已核六文件固定哈希，并取得定向/包级 test、race、vet、全仓 build、gofmt 与 scoped diff-check PASS；六文件属于未跟踪树，`git diff --check` 不覆盖其内容，格式与可构建性由 gofmt、测试和构建证据补足。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.1`、`article-management-S1.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **B4 认领/所有权**：Backend-Dev Herschel `01a1628f-360b-75d1-bb0d-7894c4385f64` 于 2026-10-22 认领并启动本卡，唯一拥有上列六文件；与 Popper 的 `identity-auth-S3.2`、Volta 的 `identity-auth-S4.3` 写集零交叉，三路并行已实际启动。本授权禁止修改六文件外内容、B5/B9C/B9D、S4.5/S4.6、设计/ADR/Manifest、依赖与 Git 集成；本卡与后续共享 article 文件的任务仍按既有 DAG 串行。
- **阻塞注记**：无
- **关闭证据与限制**：Herschel 完成 `ARTICLE-S1.2-REV-F001`～`F003` 最小修复；原 Reviewer Avicenna `01a1629a-70f1-7800-89f5-31bc94171fe3` 复审三项均 CLOSED、无新增 HIGH/MEDIUM finding，给出 `G4-task: PASS`。Lead 复核六文件 SHA-256 与复审对象一致，包级 test、race、vet、全仓 build、gofmt、scoped diff-check 均 PASS。该结论不包含后续 S1.3/S2.1/S3.1、B4 批次整体 Gate、真实 E2E、Git 集成或发布。
- **本任务关闭条件**：先以失败用例覆盖状态组合、投影泄露、非法 cursor、weak/multi/* ETag 与错误映射；成员/作者/管理员投影隔离；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S2.1、article-management-S3.1
- **设计/测试来源**：article design §3.3.1/§3.4.1；test A-ETAG/A-CON

- [x] article-management-S1.3 实现创建文章纵切 开始@2026-10-23 完成@2026-10-23

- **目标**：实现创建文章纵切。
- **类型**：开发
- **文件**：
  - `internal/article/command_service.go（新增/修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/create_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestCreateArticle"`
  - 当前状态：Lead 已现场取得本卡定向测试、`gofmt`、`go mod verify`、`go vet`、`go build`、contract、race 与全仓 `go test` 全部 PASS，并确认本卡三个 owner 文件 scoped diff-check PASS、暂存区为空。非作者代码复审 `.spec/wip/article-management/review-code-s1.3-r02.md`（SHA256 `31E79F6E094C409953CE007BA8977CA350279EDC8EEFCBE5717E8425E19294F8`）为 PASS，`F001` / `F002` 全部 CLOSED、无新 finding。未执行真实 PostgreSQL 故障注入、真实代理 partial response、浏览器、部署或 E2E；这些限制不外推为已验证。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.1`、`identity-auth-S3.2`、`article-management-S1.1`、`article-management-S1.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无；AM-Q1 决策阻塞已解除，本卡已完成实现、适用检查、非作者复审与 Lead Gate，并按自身条件 task-level 关闭。
- **本任务关闭条件**：active Principal 创建 published/clear 文章并返回 201/Location/ETag；内容按纯文本处理，不自动 trim，合法 title/body 原值逐 code point 保留；缺失、零值或全 Unicode 空白无效，title 上限 200 rune、body 上限 20000 rune。创建请求在 JSON 解码前执行 262144-byte 有界读取与 UTF-8 validity 检查：262145 bytes 或非法 UTF-8 返回 `400 request.invalid_json`，非法 byte 不得先替换/解码为 `U+FFFD`；所有 wire、UTF-8 或业务校验失败均为零 repository 调用、零持久化。客户端 author_id、受限身份、边界/空白与 DB 失败先红后绿，适用局部/项目检查与独立 Review 通过后方可关闭。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **设计/测试来源**：article Story 1；accepted article boundary r13（SHA256 `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E`）；draft article design r09（SHA256 `70EE8FB04196FA78EB925133EC8C796E1B34F5B2060273BCE2A704DE3F797B91`）；planned-not-executed test-plan r04（SHA256 `5A1FE258792CF281B27DE2B9E51131E1C46E738F8B3589A184ACFF383A9DDF3A`）的 A-CREATE-01～06 与 S1.3/S6.1 局部测试责任分界；设计复审 `.spec/wip/article-management/review-design-r09-am-q1-r03.md`（SHA256 `C1F380D83D3B00B6B5FBE3CF03B0052B8D3D9B66727333EA472BE49A84FC5AD2`）PASS。

- [x] article-management-S1.4 执行部署后 article 真实 E2E 旅程 [P] 完成@2026-11-05
- **2026-11-05 关闭（经决策不验证，ADR-0036）**：本卡专用命令 `go test -tags=e2e ./test/e2e/article` 从未执行（NOT RUN）；按 ADR-0036 以范围决策关闭 `[ ]`→`[x]`，不把历史 integration/Manifest PASS 当作本卡结果，不声称部署后 E2E 通过；未验证边界（已部署版本、真实环境/窗口、实际旅程）保留。

- **目标**：执行部署后 article 真实 E2E 旅程。
- **类型**：E2E
- **文件**：
  - `test/e2e/article/journey_test.go（新增）`
  - `test/e2e/article/README.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=e2e ./test/e2e/article`
  - 当前状态：专用命令 `go test -tags=integration ./test/integration/identity -count=1`（真实 PG18.6）与 `-race` 均 PASS；Manifest 固定七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）已按序全部 PASS。二者互不替代。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.5`、`article-management-S2.2`、`article-management-S2.3`、`article-management-S4.3`、`article-management-S4.4`、`article-management-S4.6`、`article-management-S7.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：无
- **本任务关闭条件**：部署后从真实入口完成创建/浏览/修改/删除、停用可见性、无归属处置、通知与旧备份重放三层断言；Test-Engineer 独立执行并取得业务验收决定。
- **后续验证关联**：需求 IR-01 真实 E2E
- **设计/测试来源**：article test article 真实 E2E 建议；development-delivery e2e

- [x] article-management-S1.5 验证文章 REST、三投影与 S1–S16 内部契约（仅独立组件子集可并行；完整验收blocked） 开始@2026-10-28 完成@2026-11-05
- **2026-11-05 S1.5 整卡复审 R01（非作者 Aristotle `01a1abeb-17a6-7a63-90e5-e63139a3a3eb`）：CHANGES REQUIRED；Lead 已复核确认**：评审 [report.md](../../../../.spec/wip/article-management/evidence-s1.5/wholecard-review-r01/report.md) 31,911 B / `1461ED136FB37B5412478F49DA46199E58711304D89654287F620A344EEB24A0`、[verdict.md](../../../../.spec/wip/article-management/evidence-s1.5/wholecard-review-r01/verdict.md) 8,205 B / `2184F918843655E3EE5171245BD275A5FED56AC5A84760A31B25CB2868B6ED4A`，Lead 实算一致并实读。**阻断项**：`S15-RE-F008` 被 `verification-s1.5.md:147` 等记为「R02 独立复审 CLOSED」，但仓库只有 `review-code-s1.5-f008-producer-r01.md`（结论 **NOT CLOSED**，仅证据记账阻断），`f008-allow-assertions-r02/index.json` 明写「not yet independently re-reviewed」；`01a19285…` 仅出现在该 R01 与 `verification-s1.5.md:147`，不存在 R02 复审产物。Lead 已更正 `verification-s1.5.md:147`（新 SHA `7BCFA147273EC0AF0A1FD4F8510331CBCE12379D073460D8283CB593078962A7`）与 `verification-s1.5-gap-r01.md` 行1（`71B0B694726A1410D2E4333643BDFB36F2D1D370646E333F6F8493F0137087C0`）。其余 9 项与 12 requires 经复审独立核对可核：`go test -tags=contract ./...` exit 0（29.514s）、`go vet -tags=contract ./...` exit 0（11.256s）、contract 普通/race 各 286 run / 287 pass / 0 fail / 0 skip。S1.5 保持 `[-]`；下一步唯一动作是对 `f008-allow-assertions-r02` 发起非作者独立复审，PASS 后重跑整卡复审。
- **2026-11-05 S1.5 关闭条件 gap 对账 r01（只读；整卡仍 `[-]`，不改统计/写集）**：十二 requires 逐项实核 **12/12 均为 `[x]`**（含 2026-11-05 关闭的 `article-management-S8.2`）。关闭条件 10 项中 9 项已有可核证据：16 端点（Lorentz PASS WITH MINOR + `S15-RE-F008` 复审 CLOSED）、三投影/公共错误/S1–S16（四 contract top 普通与 `-race` 各 286 run/287 pass/0 fail/0 skip）、Ready20/Unavailable6（producer contract + 真实层 58 pass）、适用真实层（应用角色 ACL R03、真实 Audit SQL、模拟收口、producer 真实层均有独立评审与 Lead 接收）、Admin2（S8.2 已关闭）、普通/race 命中。**唯一缺口**为第 10 项「整卡级非作者复审 + 接收登记 + Lead 同 revision Gate」；未发现需要新增实现或测试的缺口。对账文件 [verification-s1.5-gap-r01.md](../../../../.spec/wip/article-management/verification-s1.5-gap-r01.md)（3730 B）。
- **2026-11-05 消费 Case18 finding 级复审 R01（整卡仍 `[-]`）**：`C18-COV-F003`/`F004` 已由非作者 Reviewer 判 CLOSED 并经 Lead 现场核证；连同 ADR-0034（F002）与 ADR-0035（真实部署 E2E/双系统）范围决策，Case18 的 F001～F007 已全部关闭。S8.2 的 Admin2 侧证据链已齐，但 S8.2 是否关闭尚待用户授权；本卡的 16 端点/三投影/S1–S16 完整验收仍需按其自身 requires 与证据评估，本次不改变本卡 checkbox/统计/写集。
- **2026-11-05 消费 Case18 E2E/双系统 ADR-0035 范围决策（整卡仍 `[-]`，十二 requires 不变）**：Case18 第 5 项真实部署 E2E/双系统已按 ADR-0035 转为未验证边界，不再作为 S8.2 的关闭前置；S8.2 仍受 `C18-COV-F003` 复审与 `C18-COV-F004` 复核约束。本卡 16 端点/三投影/S1–S16 完整验收仍不具备关闭评估条件；不改 checkbox/requires/写集/统计，不宣称 Case18 或整卡通过。
- **2026-11-05 消费 C18-COV-F002 的 ADR-0034 范围决策（整卡仍 `[-]`，十二 requires 不变）**：`C18-COV-F002` 的「`failed`/`expired`/`undetermined` 三状态 + 真实 runtime/PG」验证要求已由 ADR-0034 转为未验证边界，不再是 S8.2 的关闭前置；S8.2 仍受 `C18-COV-F003` 复审、`C18-COV-F004` 复核与真实部署 E2E/双系统未验证约束，故本卡 16 端点/三投影/S1–S16 完整验收仍不具备关闭评估条件。不改 checkbox/requires/写集/统计，不宣称 Case18 或整卡通过。
- **2026-11-05 消费 S8.2 Case18 复核 r01（整卡仍 `[-]`，十二 requires 不变）**：Lead 在现役 revision + 真实 PG 上执行 `TestRecoveryProductionChainComplete`（PASS 10.87s/8 子测试）、`TestRecoveryMainBusinessDispatcher`（PASS 0.30s/8 子测试）、`TestRecoveryIsolationMatrix`（PASS 0.19s），实核 S4.20 关闭能力在现役 `recovery_runtime.go 1C679377…/37431` 上可复现。对本卡的影响：Admin2（S8.2）整体接收仍受 `C18-COV-F002`（隔离矩阵仅 closed/unknown+fake runtime）与 `C18-COV-F004`（无真实 SessionStore SQL）阻塞，`C18-COV-F003` 待原 Reviewer 复审，故 16 端点/三投影/S1–S16 的完整验收仍不具备关闭评估条件。完整事实见 `evidence-s8.2/case18-reconcile-r01/summary.md`；不改本卡 checkbox/requires/写集/统计，不宣称 Case18 或整卡通过。
- **2026-11-03 消费 S8.2 Case18 当前 revision 执行证据（整卡仍 `[-]`，十二 requires 与完整关闭条件不变）**：Lead 直接执行（未委派 Role，证据与数值登记于本文件 S8.2 卡 2026-11-03 记录与 [case18-current-r01](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01/)，不在此复制第二份状态）。对本卡“真实 Session/SQL 载体”与“S8.2 整卡”两项的影响：当前 revision 真实 PostgreSQL 18.6 上已实际执行 `SessionStore` SQL/会话撤销（6 PASS / 0 fail / 0 skip）与 S8.2 真实 PG 功能链（`TestAdminQueryPostgres` 99 PASS / 0 fail / 0 skip），root/装配层普通与 `-race` 各 93 PASS / 0 fail / 0 skip；源码 `main.go` `6D5373CA…`、`main_test.go` `3720652E…` 前后一致。**这不关闭本卡**：真实 Journal / HMAC / keyring、旧备份恢复、Case18 全局恢复隔离（`F002`/`F003` 仍 open）、真实部署 E2E 与人的业务接收（或 ADR-0028 下记录的 AI 代理接收）仍未满足；`S8.2` 仍 `[-]`，因此本卡“S8.2 整卡”门禁不变。本轮未做 Git 写、部署、删除或数据库清理，未重跑身份生产 composition，不把子集 GREEN 外推为整卡或 production readiness。
- **ADR-0028 适用（2026-10-31）**：按项目级验证授权执行；可执行层仍实际执行留证，真实环境可模拟并标注“未真实验证”，评审/签收可由 AI 代理并记录授权来源，不冒充真人签字。
- **2026-10-31 S1.5 真实 root 16 端点 method/path 与安全链盘点 r01（Lead 直取证；整卡未闭）**：现役 `newLoginBusinessRouterWithTraceAndRecovery` 实读确认 16 个端点全部有精确 method/path 分支（`main.go:3604/3608/3612/3614/3616/3618/3626/3628/3632/3644/3652`），404/405+Allow 先于安全链，非 probe 路由 `platform.TraceMiddleware` 最外层。定向同包 root 测试两轮普通与 `-race` 均 exit 0、0 fail / 0 skip（r01 314 pass/313 run、11 top；r02 321 pass/320 run、12 top，补三条 DELETE 角色/状态断言），`go vet ./cmd/han-server` exit 0，运行前后 `main.go` `6D5373CA…`（173317B）、`main_test.go` `0E61F4A3…`（399522B）`SOURCE_STABLE=True`。证据 [index.json](../../../../.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/index.json)、[16 端点映射](../../../../.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/16-endpoint-map.md)、[Lead 核证](../../../../.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/lead-verification.md)。**边界**：只证明进程内 root 注册/方法/Allow/安全链顺序，不证明真实 Session/SQL、真实 PG、Journal/旧备份、应用角色 ACL、producer 真实层、Case18 全局恢复或部署 E2E；本证据尚无独立非作者 Review，且 root 哈希相对 test-plan r18 绑定已漂移，旧哈希不沿用。本卡 `[-]`、十二 requires 与完整关闭条件不变，未代签人的接收。
- **2026-10-31 S1.5 root 16 端点独立评审 R01：PASS WITH MINOR（8 LOW 分类处置；整卡未闭）**：非作者独立评审（Lorentz / add-reviewer，actualID `01a19267-1404-7050-b09d-56e91fd57ddd`，只读，唯一写入为报告）[报告](../../../../.spec/wip/article-management/review-code-s1.5-root-endpoints-r01.md) SHA256 `DA2BD8D786C13A951343C095EBF48BEA2A14973322B9F0FB53CB5ABE375D34B4` / 21470 B / 151 行，结论 HIGH 0 / MEDIUM 0 / LOW 8；Lead 已实读报告、复算报告哈希一致，并逐条核对 16 端点分支行号、handler 槽位与 `Allow` 字符串。**七条已处置**（只改证据文档表述与可复核信息，未改任何行为断言、未重跑原始轮次）：`S15-RE-F001` `{id}` 约束归属改述为统一经 `exactArticleDeletePath` 及其 `exactArticleModerationPath`（`main.go:3766→3769`）、`exactOrphanActionPath`（`3772→3774`）包装；`S15-RE-F002` 移除绕过业务 root 且不在运行集的 `TestOrphanDispositionRoot_HTTPNOWAIT_ReturnsDatabaseUnavailableWithoutLoop` 覆盖引用；`S15-RE-F003` 覆盖表改用日志中真实 `t.Run` 全路径（如 `TestProductionAdminArticleQuery/OldConsumerCompatibility/read_projections/member_list`），弃用 `adminRootRouteMatrix/*`、`adminRootOldReadProjections/*` 函数名选择器；`S15-RE-F004` `index.json` 明确标注为工作树快照、`HEAD` 不含 Go 代码、不可由 `git checkout` 重建，源码清单扩到 8 个文件（补 `internal/platform/http.go`、`http_security.go`、`internal/session/store.go`、`maintenance_store.go`、`go.mod`、`go.sum`）；`S15-RE-F005` 补 r01/r02 完整 `-run` 表达式与 19/20 顶层测试清单，并以重建的 r01 表达式普通与 `-race` 各回放一次，均 314 pass / 313 run / 19 top / 0 fail / 0 skip、回放前后两 root 文件哈希不变；`S15-RE-F006` “唯一生产 router”改为“唯一生产业务 API router”，披露同进程 metrics 与维护模式 recovery-control server 不承载 16 端点；`S15-RE-F007` 安全链主张限定为平台层边界次序，`session`/`csrf`/`principal` 事件由测试替身按序 append，不外推真实 resolver 内部次序。**保留开放**：`S15-RE-F008`——`GET /api/v1/admin/orphan-articles`、`POST .../publish`、`POST .../transfer` 仅断言 405 状态，缺 `Allow` 精确值与“仅 trace 事件”断言；补断言须改 `cmd/han-server/main_test.go`；原文据“另一 Codex 会话 PID 35652 存活”判断共享 root 写窗未释放，本轮未实施。**更正（2026-10-31 实查）**：PID 35652 实为本会话 Codex 宿主（`codex.exe app-server`，父进程 `Code.exe` 44044，09:46:25 启动，当前 shell 为其子进程），不构成外部写者；真实外部并发风险为 `xDesk.exe`（重启后 PID 35384，19:13:13 启动，5 个子进程仍存活，同 `:2949` 口径）。原判断有误，保留原文以便追溯；`S15-RE-F008` 未实施的事实不变。**证据**：[index.json](../../../../.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/index.json) SHA256 `E039B1F15A33B33C80F4CB03C5A94EF31A37F8BE66127BA7F7810E84C7DA4AEE`（7 runs / 10 artifacts 实算逐项一致）、[16 端点映射](../../../../.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/16-endpoint-map.md) `67A01D854BE79FC9E0836F3A7FC3F335EBAE9B0B77B7CBC87773749841C23C7D` / 16117 B、[Lead 核证](../../../../.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/lead-verification.md) `D6DFCE12035C7CD95BE83F222A6E79A9654A87D70A13180FE392E97C9ECCA9CB` / 2766 B（**2026-11-05 更新**：按整卡复审 F003 为 `16-endpoint-map.md`、`lead-verification.md` 追加 F008 前向指针后，index `A4155C64…`/24,714 B、map `2F10C202…`/16,611 B、lead-verification `8E6FD8A0…`/2,978 B；以上旧哈希保留为历史）；回放日志 `replay-r01-normal-json.log` `B432FF313447035A6F80635CAA29A7ED2ED6C58CEBFF0335AFC80F29C4B88B0C`、`replay-r01-race-json.log` `78A0043A42F251589D6383147B32AF3936347D4BCBAE4490AD9A7C3F56863905`。**边界**：本记录只把该子门禁推进到“取证 + 独立评审 PASS WITH MINOR（1 条 LOW 开放）”，不关闭 S1.5 整卡；`[-]`、十二 requires 与完整关闭条件不变；不证明真实 Session/SQL、真实 PG、应用角色 ACL 载体、producer 真实层、Journal/旧备份、Case18 全局恢复或部署 E2E；不代签人的接收。本轮证据写入仅限 `.spec/wip/article-management/evidence-s1.5/root-16-endpoint-r01/` 内 3 份文档与 2 份回放日志，`cmd/`、`internal/` 零改动。
- **2026-10-31 S1.5 子门禁现状矩阵回填（只读汇总；整卡仍 `[-]`）**：Lead 在 [verification-s1.5.md](../../../../.spec/wip/article-management/verification-s1.5.md) 末尾追加“S1.5 子门禁现状矩阵”，逐行登记 6 项已取证子门禁与其实算哈希：四 contract top（普通与 `-race` 各 286 run / 287 pass / 0 fail / 0 skip、tagged vet exit 0，[index](../../../../.spec/wip/article-management/evidence-s1.5/contract-suite-r01/index.json) `5477B1F5…`，四文件现役 rest `CEBA105C…`/internal `DC6791D4…`/projection `D7585EE6…`/producer `6EB4A677…`）；16 端点真实业务 root（独立评审 PASS WITH MINOR、7 LOW 已处置、`S15-RE-F008` 开放，index `E039B1F1…`、报告 `DA2BD8D7…`）；应用角色 ACL 子集（R03 PASS WITH MINOR + Lead Gate、真实 PG 18.6 普通与 `-race` 各 12/12，acl-r03 index 现为 `8ED9FF3E…`，已在 `R03-F002` 后不同于旧记录 `0ECAACE5…`）；真实 Audit SQL 子集（r19，implementation `4B93DD53…` / review `A4CF2B92…`）；模拟环境收口 r01/r02（`3E737B8C…` / 12332 B）；四文件组件 R02（`52298FED…`，仅绑定 producer 旧哈希 `B519DDE1…`，现役 `6EB4A677…` 由 S4.4 评审覆盖，不得表述为 R02 覆盖现役版本）。矩阵同列仍未满足/unknown 六项：producer 真实 `ReadCurrent`/`Publish`、真实 Journal/旧备份、Case18 全局恢复隔离、S8.2 整卡、`S15-RE-F008`、真实部署 E2E 与接收。**边界**：纯只读汇总，未改代码、未重跑测试、未重开任何已完成评审、未改 requires/关闭条件；收据新 SHA256 `630EB733AD4152E9FA2810142A2388487D5249F7381769AB94B405AD2F97A013`（149 行、LF、无 BOM），原 `9EF6EF3B…` 保留为历史指针。另：原文据“PID 35652 仍存活、`cmd/han-server` 文件最近修改 21:45”判断共享 root 写窗未释放，故 F008 未实施；**更正**：35652 为本会话宿主，非外部写者（21:45 的 `recovery_main_dispatcher_integration_test.go` 内容哈希 `BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A` / 23693 B，与 Lead 已核版本一致），外部并发风险实为 `xDesk.exe`（PID 35384 起）；原文判断有误，保留以便追溯。
- **2026-10-31 S1.5 模拟环境收口 r01/r02（A-CON-03 与恢复隔离矩阵；整卡未闭）**：Lead 直改测试侧并亲跑，新增 `cmd/han-server/recovery_isolation_matrix_integration_test.go`（r02 SHA256 `DA02943B9848C50E1960ECCDBF210C3E69D3AA8A64034F9A6321740E519E9115`，8498B），生产代码零改动。A-CON-03 由真实 PG 18.6 ACL 的 `insert_allowed` 与 `mutation_denied`（6 条 SQLSTATE 42501）12/12 覆盖并于 r02 复跑；矩阵覆盖 25 条受保护业务路由 × closed/unknown，逐条断言 503 + `CodePlatformRecoveryInProgress` + envelope Trace 与 header 一致 + 泄漏断言 + repository/reader/resolver 零调用，并断言 `acquires==25`、`releases==0`；`/livez` 200、`/readyz` 503 且不计入 acquire，404 显式失败防空转。独立评审 R01 结论 PASS WITH MINOR：MEDIUM `S15-SIM-F001`（禁词 `ref` 随机命中 ULID Trace）以掩码合法 Trace 修复，`-count=25` 复跑通过；LOW `S15-SIM-F002` 补 authorization/origin 记录器与 guard-open 正控（`POST /api/v1/articles` 201、acquire/release=(1,1)）；`S15-SIM-F004`（证据命令与并发 S4.20 写 `main.go` 交叠、整树未跟踪）如实保留为已披露边界。证据 `.spec/wip/article-management/evidence-s1.5/sim-closeout-r01/`（`index.json` 16 条命令自洽、r02 日志齐备、保留资源 68 行未清理）；评审处置 `.spec/wip/article-management/review-sim-closeout-r01-disposition.md`（SHA256 `67F2DF1B0C9CF302CC1A8050D7B1DE95CD1A0017CD025A88B9E73BEDAFFDF6D2`）。**未闭边界**：Case18 真实双系统全局恢复隔离 `NOT RUN/unknown`；真实全局恢复锁、真实 Journal/HMAC/keyring 未验证；不宣称 production readiness；AI 代理评审与登记，非真人签字。
- **2026-10-31 S1.5 producer 真实层证据消费（recovery-runtime B13d 外部卡；整卡未闭）**：Lead 只读盘点 recovery-runtime `evidence-b13d-r01/a48-f001-repair-author-r02`（Noether / add-backend-dev，actualID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`），确认现役 `PostgresReplayEvidenceStore.Publish` / `ReadCurrent` 已有真实 PostgreSQL 18.6（`127.0.0.1:51934` / `han_b13d_article`，`pg:"freshD Article only"`）双轮证据：`required-pg-normal` 与 `required-pg-race`（命令 `go test [-race] -p=1 -vet=off -tags=integration ./internal/article -run '^(TestReplayPublicationPostgres|TestDeletionReplayPostgres)$' -count=1 -json`）均 exit 0、**58 pass / 0 fail / 0 skip**；两轮 argv 均无 `-overlay`，测试二进制即现役源。运行绑定的 10 个源文件（`replay_evidence.go` `7C14E74D…`、`replay_publication.go` `34B698A2…`、`replay_publication_integration_test.go` `8F5DD24D…`、`0011_article_replay_publication.sql` `7C7F1B1F…` 及 `replay_publication_test.go`、`deletion_replay.go`、`deletion_replay_source.go`、`deletion_replay_test.go`、`deletion_replay_source_test.go`、`deletion_replay_integration_test.go`）本轮实算 SHA256 与字节逐项一致；`source-protection.json` 记 `replay_evidence.go` entry/exit 同为 `7C14E74D…` 且 `unchanged:true`。调用链实读：fixture 持有 `*PostgresReplayEvidenceStore`（`replay_publication_integration_test.go:194/241`），`publicationPGRollback` 直接调用 `f.store.Publish`（`:455`）与 `f.store.ReadCurrent`（`:461`）；叶子 `.../RefreshOutsideRunAndRecheck/Publish` 与 `/ReadCurrent` 两轮均 PASS。门禁 `newReplayPublicationPool`（`:31`）空 DSN 即 `t.Fatal`（`:35`）、host 非 `127.0.0.1` 即 `t.Fatal`（`:42`）、无 skip 分支，58 leaves / 0 skip 说明门禁真实满足。**因此本卡原“producer 真实层 unknown”推进为“已消费外部可核证据（待独立评审/接收）”，满足本卡“真实 Publish/ReadCurrent/source 证据先消费原 S4.2 精确版本/现役 PG 验证”的输入条件**。证据 [index.json](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-r01/index.json) SHA256 `F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B` / 19567 B、[report.md](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-r01/report.md) `A9A4ECBB9DEF6C1862ECE1D151F085F179DE43A6A489FD94411B017978150BFF` / 8023 B；上游 index `968A8FBE…`、source-protection `C9D35B0B…`、source10 `6AE6D329…`、`required-pg-normal` stdout `52BBB982…` / result `A306259A…` / leaves `2F0F01CD…`、`required-pg-race` stdout `2602BD3C…` / result `A4449600…` / leaves `B44F93D1…`。**边界**：本记录为消费外部卡证据、非 S1.5 本轮执行，未重跑测试、未改代码（`cmd/`、`internal/`、`test/` 零改动，`main.go` `6D5373CA…`、`main_test.go` `0E61F4A3…`）、未做独立评审；不覆盖真实 Journal/HMAC/keyring、旧备份恢复、Case18 全局恢复隔离、应用角色 ACL、部署 E2E；B13d 的 `red-*` / `green-*` 为 F001 test-only overlay 轮次，与本消费无关。`S15-RE-F008` 仍开放（该轮按“PID 35652 存活”记为外部写者风险，2026-10-31 实查更正：35652 为本会话宿主，外部并发风险实为 `xDesk.exe`）；S1.5 整卡保持 `[-]`、十二 requires 与完整关闭条件不变，不代签人的接收。同步：`verification-s1.5.md` 子门禁矩阵新增第 7 行并从未满足项移除 producer（新 SHA256 `7CF8D81BD2239D80EEFC1999412C230FA37C2E22AAE716E4FD1724C301BBF605` / 31189 B，150 行、LF、无 BOM；本节写入后因 PID 表述更正再次变更，`97CC2141…` 与 `630EB733…` 均保留为历史指针）。
- **2026-10-31 S1.5 `S15-RE-F008` 修复（orphan 路由 405 `Allow` 与仅 trace 断言；整卡未闭）**：Lead 直改 `cmd/han-server/main_test.go` 并亲跑，生产代码零改动。`requireOrphanProductionRejection`（原 `:3349-3397`）的 method 分支原只断言 405；本轮注入 `trace` seam 记录事件序列，新增 `Allow` 精确断言（`response.Header().Get("Allow") == mainOrphanAllow(action)`）与 `events == []string{"trace"}` 且 `resolver.calls == 0` 断言，并新增 `mainOrphanAllow`（`list`→`GET`；`detail`/`delete`→`GET, DELETE`；`publish`/`transfer`→`POST`），与生产 `main.go:3613/3615/3617/3653` 的 `allow` 字面量逐字对应；同 helper 顺带覆盖 detail/delete 的 method 变体。运行：`go test ./cmd/han-server -run 'TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity' -count=1 -v` exit 0（33 leaves PASS，1.36s；**该记数有误：实数 34 叶子，耗时以日志 1.380s 为准，见本行末 r02 更正**）、`go test ./cmd/han-server -run 'orphan|Orphan' -count=1 -race` exit 0（`ok 2.891s`）、`go test ./cmd/han-server -count=1` exit 0（`ok 40.455s`）、`go vet ./cmd/han-server` exit 0、`gofmt -l cmd/han-server/main_test.go` 空。文件：`main_test.go` `0E61F4A3…`/399522 B → `220A49CD813C02C2224293EDE2E5D4B9350F550C118552D302B5A24DC6DE453B`/400192 B；`main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727`/173317 B 前后不变。证据 [index.json](../../../../.spec/wip/article-management/evidence-s1.5/f008-allow-assertions-r01/index.json) `DA780BAEB8B21425670F72F2A675CB6EBE2B21DDE6F541D5C39B4BAF04A5FE97` / 5880 B、[report.md](../../../../.spec/wip/article-management/evidence-s1.5/f008-allow-assertions-r01/report.md) `941F321F3674E0DD5DE712FAEEE01BE155172EFC338535ACD4EC60C56E16705D` / 4055 B，四个运行日志哈希见 index。**边界**：仅证明进程内真实业务 root 的 method/`Allow`/事件次序；不证明真实 Session/SQL、真实 PG、Journal/旧备份、Case18 全局恢复隔离、应用角色 ACL 或部署 E2E；本修复未经非作者独立复审、无人的接收，`S15-RE-F008` 由“开放”转为“已修复待复审”。同步：`verification-s1.5.md` 矩阵第 2 行 LOW 8 全处置并新增第 8 行，未满足项移除 F008（新 SHA256 `9BCFBFBC43E842008D7874F48FEF735BE0428B977F892DB830FE401B1DB16A97` / 31791 B，151 行、LF、无 BOM；`7CF8D81B…` 及更早指针保留为历史）。S1.5 整卡 `[-]`、十二 requires 与完整关闭条件不变。 **2026-10-31 更正（r02）**：非作者复审（`add-reviewer`，actualID `01a19285-8ece-7c20-a161-c0d7b8cce6aa`；[复审 R01](../../../../.spec/wip/article-management/review-code-s1.5-f008-producer-r01.md) `84377433…` / 5067 B）判 `S15-RE-F008` **NOT CLOSED**——代码层成立（`Allow` 字面量逐字一致、负向有效性成立），但证据记账不成立：实际 34 叶子被记成 33、报告耗时 1.36s/2.891s/40.455s 与日志 1.380s/3.009s/39.555s 不符、`gofmt` 无保存日志、race 日志仅单行、`mainOrphanAllow` 默认分支静默返回 `POST`。Lead 已单轮重采 [r02 证据](../../../../.spec/wip/article-management/evidence-s1.5/f008-allow-assertions-r02/index.json)（index `410A65AD…`/5564 B、report `B6CA7A0C…`/5721 B）：`main_test.go` 现为 `3720652E…`/400312 B，`mainOrphanAllow` 改收 `*testing.T`、未知 action `t.Fatalf`；普通 34 叶子 PASS、race（带 `-v`）34 叶子 + 2 顶层 PASS、整包 exit 0、`go vet` 与 `gofmt -l` 输出空，耗时逐字取日志 1.326s/3.027s/39.076s，`main.go` `6D5373CA…` 零改动、执行前后 `main_test.go` 哈希不变；前述旧的 33/耗时表述由本段取代，r01 全部文件原样保留追溯。**该 finding 仍待 r02 独立复审，S1.5 整卡 `[-]`、十二 requires 与未满足清单不变；producer 证据同轮复审判“接收（有边界）”，不等于 Review PASS 或人的接收。** **2026-10-31 曾记「R02 复审：`S15-RE-F008` CLOSED」（引用 actualID `01a19285-8ece-7c20-a161-c0d7b8cce6aa`）——2026-11-05 经整卡复审 Aristotle `01a1abeb-17a6-7a63-90e5-e63139a3a3eb` 指出、Lead 复核确认不成立：仓库无对应 R02 评审产物，该 ID 只见于 R01 报告与 `verification-s1.5.md`；其「R01 五项全闭」不再作为 CLOSED 依据，其后四条 L 级文档处置作为历史保留追溯。producer 证据维持“接收（有边界）”，本轮只改 `main_test.go`、不在其 10 源清单内。四条 L 级文档问题处置：**L-1** 矩阵曾误写复审哈希 `57433F67…`，现役为 `84377433…`/5067 B，已更正；**L-2** 本记录前段旧 `33 leaves`/耗时已就地加注“有误、见行末更正”，原文数字保留追溯；**L-3** 行号指针由 `3612/3614/3616/3652`（`case` 行）更正为字面量行 `3613/3615/3617/3653`；**L-4** r02 `report.md` 中“三条期望值互不相同（GET/POST/POST）”自相矛盾，改写为“覆盖两种不同取值”，新哈希 `400E593A…`/5808 B，数字与命令未变。复审未验证项如实保留：r02 五份日志无时间戳（同轮次仅由 `index.json` 的 `generated_at` 声明）、`go vet` 空日志不编码 exit code、`main_test.go` 执行前后哈希一致仅有报告声明。**2026-11-05 R02 复审（真产物）：`S15-RE-F008` CLOSED**（非作者 Reviewer，agent `01a1ac0f-6e77-7e12-b417-c11ea4ef4a15`，按 `add-code-review` 方法只读，唯一写入为评审目录）：五点核定全部通过——34 叶子 / 0 fail / 0 skip 与 `f008-targeted-normal.log` 一致；耗时 1.326s / 3.027s / 39.076s 逐字取自日志；`gofmt.log` 为 0 B 空输出、索引记命令与 exit 0，独立 `gofmt -l` 亦 0 行输出（有限证据）；`main_test.go:3402` 精确断言与 `mainOrphanAllow` 映射同 `main.go:3613/3615/3617/3653` 字面量逐字一致；`index.json` 9 个对象 SHA256/字节实算全部相符。实跑：targeted normal exit 0（wall 3.828s / go 1.414s，34 PASS）、`gofmt -l` exit 0（0.148s）、`go vet ./cmd/han-server` exit 0（1.264s）；`-race` **NOT RUN**（`-race requires cgo`，exit 2，本机无 CGO/C 编译器，改审阅保存日志）。产物 [report.md](../../../../.spec/wip/article-management/evidence-s1.5/f008-review-r02/report.md) `B715476E…` / 11030 B、[verdict.md](../../../../.spec/wip/article-management/evidence-s1.5/f008-review-r02/verdict.md) `1749A628…` / 4533 B，Lead 现场复算一致；`main.go` `6D5373CA…` / 173317 B、`main_test.go` `3720652E…` / 400312 B 未变。保留限制：`gofmt.log` 不能自证 provenance、race 未复跑、full-package 未复跑、`cmd/` 为 untracked。 **边界**：本次只关闭 `S15-RE-F008` 这一 LOW；S1.5 整卡 `[-]`、十二 requires、真实 Journal/旧备份、Case18 全局恢复隔离、应用角色 ACL 完整范围、真实部署 E2E 与人的接收均不变。
- **2026-11-05 S1.5 整卡复审 R01 的 F001/F002/F003 处置与 `S15-RE-F008` R02 关闭回写（整卡仍 `[-]`）**：非作者 Reviewer（agent `01a1ac0f-6e77-7e12-b417-c11ea4ef4a15`，按 `add-code-review` 方法，只读）对 `f008-allow-assertions-r02` 出具独立复审，判 `S15-RE-F008` **CLOSED**：五点核定全通过（34 叶子/0 fail/0 skip、耗时 1.326s/3.027s/39.076s 逐字取自日志、`gofmt.log` 0 B 有限证据、`Allow` 断言与 `main.go` 字面量一致、9 个对象哈希实算相符）；本机 `-race` 未复跑（`-race requires cgo`），改审阅保存日志。产物 [report.md](../../../../.spec/wip/article-management/evidence-s1.5/f008-review-r02/report.md) `B715476E…`/11030 B、[verdict.md](../../../../.spec/wip/article-management/evidence-s1.5/f008-review-r02/verdict.md) `1749A628…`/4533 B，Lead 现场复算一致且 `main.go` `6D5373CA…`/173317 B、`main_test.go` `3720652E…`/400312 B 未变。**F001 关闭**：`verification-s1.5.md` 第 8 行改记 R02 CLOSED，并注明 2026-10-31「R02 CLOSED」原文无产物支撑、已撤回，现结论以 `f008-review-r02/` 为准；本卡上一行同处已更正。**F002 关闭**：`verification-s1.5.md` 未满足清单移除 S8.2，并加 2026-11-05 更新说明（S8.2 已于当日关闭 `[x]`，见 tasks 的 S8.2 卡与 gap-r01:20）。**F003 关闭**：`root-16-endpoint-r01/16-endpoint-map.md`（`2F10C202…`/16,611 B）与 `lead-verification.md`（`8E6FD8A0…`/2,978 B）追加 F008 前向指针（原文与旧日志保留；现役 `main_test.go` `3720652E…`/400312 B，16/16 端点有 `Allow` 值断言），同目录 `index.json` 更新为 `A4155C64…`/24,714 B 并经 `ConvertFrom-Json` 解析通过；`verification-s1.5.md` 第 2 行 index 指针同步。**F004（INFO）不处置**：仓库级 `gofmt -l .` exit 2 来自 `.spec/wip/**` 证据快照，源码树 `gofmt -l cmd internal test` exit 0；改扩展名属另行动作。现行：`verification-s1.5.md` `39529A54…`/52,346 B（LF/198 行/无 BOM）、`verification-s1.5-gap-r01.md` `FD907D98…`/4,409 B。下一步：以更新后的对象重跑整卡复审 R02，通过后由 Lead 做同 revision Gate 并登记接收。整卡 `[-]`、十二 requires 不变；AI 代理评审非真人签字。
- **2026-11-05 S1.5 整卡复审 R02：PASS FOR LEAD GATE；Lead 同 revision Gate 通过（接收与关闭见下条）
- **2026-11-05 `article-management-S1.5` 关闭（`[-]`→`[x]`，完成@2026-11-05）**：按 ADR-0028 登记 AI 代理接收；授权来源为用户 2026-10-31 项目级授权 + 本会话「继续」确认，代理主体为本 Codex 会话 Lead（AI 代理，非真人）。接收范围：16 endpoint 精确 method/path 与 `Allow`/安全链边界、三投影、公共错误、S1–S16、Ready20/Unavailable6、适用真实层子集、普通与 `-race` 各 286 run / 287 pass、12 requires；不覆盖真实 Journal/HMAC/keyring、旧备份恢复、部署 E2E/双系统、Case18 全局恢复、S2.2 容量与 production readiness。登记见 [acceptance-s1.5-r01.md](../../../../.spec/wip/article-management/acceptance-s1.5-r01.md) `67A5CF3A2FF0055BF6EF4FA4551C720EB0607EAA2F2E7A9E731E1B8550BD87DA` / 3,090 B。统计更新为 77 completed / 0 active / 9 todo；未改其他卡、未执行 Git 写。
- **2026-11-05 IR-01 交付入库（本地 Git）**：`main` 新增 6 个提交 `9b661ed`（关闭任务与范围决策）→ `6662d95`（身份/文章 MVP）→ `ec76c85`（检查命令与 Temp 忽略）→ `268b5f5`（证据摘要与问题单）→ `35e2bbf`（迁移/夹具/文档）→ `4bb1bf9`（忽略二进制）；提交前扫描 31,619 个文本文件、28 处合成 DSN，按 ADR-0038 判定为非凭据。离线包 `.spec/wip/transfer/han-ir01-20261105.bundle`（由本次记录提交后重建，哈希见同目录 `.sha256`）。**push 未完成**：经代理 `http://proxy.xfusion.com:8080` + `http.schannelCheckRevoke=false` 的 HTTPS 只读访问可用（`ls-remote` 成功，远端仍为 `6da8abb`），但实际 `git push` 返回 **HTTP 403**（当前凭据无写权限）；远端未发生变更，需换有写权限的令牌或账号后重推。**：非作者 Reviewer（agent `01a1ac16-8717-7b13-b1c2-9164f0759686`，只读）判 F001/F002/F003 全 CLOSED、关闭条件第 1–9 项 PASS、第 10 项 PASS FOR LEAD GATE、12/12 requires `[x]`；产物 [report.md](../../../../.spec/wip/article-management/evidence-s1.5/wholecard-review-r02/report.md) `E2ED8CA0…`/20,822 B、[verdict.md](../../../../.spec/wip/article-management/evidence-s1.5/wholecard-review-r02/verdict.md) `A5E10329…`/9,468 B，Lead 实读复算一致。复审复跑 `go test -tags=contract ./...` exit 0（28.514s）、`go vet -tags=contract ./...` exit 0（2.218s）、`gofmt -l cmd internal test` exit 0（0.364s）。**Gate**：HEAD `6da8abb…`/`main`、`main.go` `6D5373CA…`/173,317 B、`main_test.go` `3720652E…`/400,312 B 未变，`git status --porcelain` 321 项与复审前一致，统计 76 completed / 1 active / 9 todo。保留 LOW：`root-16-endpoint-r01/index.json.known_gaps` 仍记 `S15-RE-F008 state=open`（2026-10-31 快照字段，现役以 `f008-review-r02/verdict.md` 为准）；F008 R02 产物未内嵌 add-reviewer actualID。**尚未登记人的业务接收或 ADR-0028 AI 代理接收**；接收登记后才可改 `[x]`。AI 代理评审与 Gate 复核，非真人签字。


- **2026-10-31 S1.5 路径B（应用/迁移身份分离＋启动ACL fail closed）交付（Lead核收；整卡未闭）**：按用户 `2026-10-31` 明确选择路径B执行，仅补 A-CON-03 应用有效 ACL 子集。新增 `cmd/han-server/append_only_acl.go` SHA256 `1019A5E1F38CBB46EAAE1B5A4948DB231D0205117A6D03866AF6F01F435D12AD`、`cmd/han-server/append_only_acl_test.go` `54AD3302F95CE4FAE8CACE57A19353FCDFA3646A2248F0597C23E7E7C7771C2B`、`cmd/han-server/append_only_acl_integration_test.go` `8A806C38A66299BE47332B0756D345798645F3D10F9E62476ED2F99F7F55579C`；修改 `cmd/han-server/main.go` `68E91CFB0BE866A4888950595955A4411DB897DBAB846A39C63E1FE154D982A7`、`deploy/db/provision.sql` `0AF8986F34458059AE2F1398F2F49932575D0B57419C6870D2A1BFB46F57BA05`、`deploy/db/grant-application.sql` `95A93A58DB98776ECBF33AEC0F3ED14E338BB6663520E603152756FA5E2B3351`、`deploy/db/README.md` `2F808751CEAA6800D5587A1013857404CFC66046B0916671AD18758FF1448F2F`。实现：`HAN_MIGRATION_DATABASE_DSN` 迁移身份与应用身份一致性校验（原始DSN与 host/port/db/user 双口径比对）；启动顺序 connect/ping → migrate（独立迁移池）→ 关闭迁移池 → 应用池 `verifyAppendOnlyApplicationPrivileges` 门禁：先拒绝 superuser，再对 `pg_class` 中 `relkind IN ('r','p')` 且 `pg_table_is_visible` 的表断言 INSERT=true、UPDATE/DELETE/TRUNCATE=false，任一不满足即启动失败、不降级运行。真实执行：`go build ./...` exit0；`go test ./cmd/han-server -count=1` ok 42.151s；`go vet -tags=integration ./cmd/han-server` exit0；真实 PostgreSQL 18.6 集成 `TestAppendOnlyACLGate` PASS 3.081s。证据：[索引](../../../../.spec/wip/article-management/evidence-s1.5/acl-r01/index.json) SHA256 `9A046BB630D800E5D21B937E7FE6A87A0D3937EB0D917B6549D6546EDB4AA69A`、`run-01.log` SHA256 `43813CA27BA47B60B2846271E1AAD64AD3E6BF2D08673AAA208B7675D4954D82`；保留资源（未清理）schema `han_s15_acl_a1023308b90a9b3e`、角色 `han_s15_mig_a1023308b90a9b3e` / `han_s15_app_a1023308b90a9b3e` 见 `retained-resources.ndjson` SHA256 `E5C035EED4D2F0B3DC4FBC0270C67251D9B5DB46D268BE3189FC884B6B6EE9C6`。**同时修复真实缺陷**：`provision.sql` 原 `DO $$ … :'migrator_role' … $$` 在 psql 下不替换 dollar-quoted 块内变量（本机复现 `错误: 语法错误 在 ":" 或附近`，exit=3），已改为 `SELECT format('CREATE ROLE %I', :'…') WHERE NOT EXISTS (…)` + `\gexec`；以 `BEGIN; \i deploy/db/provision.sql; ROLLBACK;` 配合唯一临时角色/schema 名复验 exit=0、各步成功、回滚后 `pg_roles`/`pg_namespace` 计数均为0，日志 `provision-probe-r01.log` SHA256 `913DD6A5A0827F399831D727AF3894528FB1FDEEB30C0C34844E21827E64FBE0`。**边界**：仅单机合成角色与单 schema，不证明真实部署集群/CI、多节点角色、生产 superuser 绕过、备份恢复演练；独立评审与人的签收未完成前不关闭本卡。原十二 requires、完整关闭条件、`[-]` 与开始日期均不变；A-CON-03 以外的生产/SQL/身份/旧备份/全局恢复责任不由此释放。
- **2026-10-31 S1.5 ACL 独立评审 R01：CHANGES REQUIRED（修复中）**：非作者独立评审（Confucius / add-reviewer，actualID `01a190e6-66c8-72c1-ab3a-813582764124`）报告 [`review-code-s1.5-acl-r01.md`](../../../../.spec/wip/article-management/review-code-s1.5-acl-r01.md)，结论 Code Review Gate FAIL，11 findings（3 HIGH / 6 MEDIUM / 2 LOW）：`S15-ACL-F001`（仅按 `current_user` 判 superuser，superuser 会话 `SET ROLE` 可绕过）、`S15-ACL-F002`（`provision.sql` 默认权限授予 UPDATE/DELETE 且 README 把 `grant-application.sql` 排在服务启动之后，fresh install 下 gate 必失败）、`S15-ACL-F003`（integration 包因缺失 `newS419RecoveryWorld` 不可构建，阻断真实 PG 集成）、`S15-ACL-F004/F005/F007/F011`（schema owner、fallback 身份比较、列级 UPDATE、search_path）、`S15-ACL-F008/F009`（配置校验顺序与迁移池关闭、保留资源登记时点）、`S15-ACL-F010`（README 哈希基线未冻结）。Lead 已实读报告并确认 F001/F002/F003 客观成立。处置：F003 构建阻断已由 Lead 先行解除——依赖缺失 harness 的 `OpenAfterCompleteSoftwareRecoverySameSession` 子测试移出 `cmd/han-server/recovery_main_dispatcher_integration_test.go`，原文完整留存 `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-world-hold-r01/`（SHA256 `36DA30ABD77EAC4EE8E6098F588677B2A20BB4A854B6FEA8F161168D2D634391`），`go vet -tags=integration ./cmd/han-server` 与 `TestRecoveryMainBusinessDispatcher` 恢复 exit 0，该场景转由 `identity-auth-S4.20` 承接；F001/F002/F004/F005/F007/F008/F009/F011 已按 ADR-0028 授权派后端在同一写集内修复（`cmd/han-server/append_only_acl*.go`、`database_config.go`、`deploy/db/*`，必要时 `main.go` 顺序），修复完成后须由同一非作者复审 R02 逐条复核并重跑真实 PG 正/负控。R02 PASS 与 Lead Gate 之前，ACL 子集不构成可交付证据；本卡 `[-]`、十二 requires 与完整关闭条件不变。
- **2026-10-31 S1.5 ACL 修复 R02 已交付 / Lead 现场复核通过（待独立复审）**：后端修复（Herschel / add-backend-dev，actualID `01a190fe-fb35-7921-9ca0-84fae3cf5764`）按 R01 十一条 findings 在同一写集内完成修复，证据目录 `.spec/wip/article-management/evidence-s1.5/acl-r02/`（`index.json` SHA256 `7921D68E0D4991C9E9B696E1A66C203520BD70D5FD03AA90EF320544EE354CBE`，权威运行 `run-08..15.log`，`run-01..07.log` 标记 superseded 保留，保留资源只追加）。Lead 已独立复核而非转述摘要：8 个目标文件实际哈希与 `index.json` 逐项一致（`append_only_acl.go` `ffa71581…`、`append_only_acl_test.go` `5dbe9c14…`、`append_only_acl_integration_test.go` `f9c288be…`、`database_config.go` `6b29e872…`、`main.go` `1d47cc94…`、`provision.sql` `52525560…`、`grant-application.sql` `fc0d2298…`、`deploy/db/README.md` `afbd4027…`）；`go build ./...` 0、`go vet ./cmd/han-server` 0、`go vet -tags=integration ./cmd/han-server` 0（R01 F003 不复现）、`gofmt -l` 空；`go test ./cmd/han-server -count=1` ok 43.100s；真实 PG 18.6 集成 `TestAppendOnlyApplicationACL` PASS 11/11（2.524s），含 fresh-install 默认权限即过闸、schema owner=迁移角色、真实非超管 LOGIN 正控、ledger 与建表与 6 条 UPDATE/DELETE/TRUNCATE 全部 42501、superuser 会话与 `SET ROLE` 双负控、owner 负控、列级 UPDATE 负控。Lead 本轮运行保留资源已追加登记（`han_s15_acl_68214477f0e649e7` 及 2 角色）。**边界**：以上只证明本地单实例 PG 18.6 等价 DDL，不证明真实生产部署、public schema 路径、`search_path` 运维前置、多 fallback 真实连通与角色轮换；F011 以文档化边界关闭。已派同一非作者复审 R02；R02 PASS 与 Lead Gate 前，本卡仍 `[-]`、十二 requires 与关闭条件不变。
- **2026-10-31 S1.5 ACL R02 独立复审 + R03 窄范围返工（Lead 现场复核通过；待 R03 复审）**：非作者复审（Confucius / add-reviewer，actualID `01a190e6-66c8-72c1-ab3a-813582764124`）报告 [`review-code-s1.5-acl-r02.md`](../../../../.spec/wip/article-management/review-code-s1.5-acl-r02.md) SHA256 `E1B93E05D005521C729FEBDC69CE663FC64D5923D5B870F5596EA7AEDC517C67`：结论为**窄范围 CHANGES REQUIRED**，`F001–F010` 全部 CLOSED，`F011` PARTIAL（文档边界存在但代码未校验 search_path），另新增 `S15-ACL-R02-F001`（MEDIUM：`pg_table_is_visible` 不检测同名关系歧义，实测 `visible_count=1`/`on_search_path_count=2`，与 README 与测试注释表述矛盾）、`F002`（LOW：pgx 接受 URL 顶层 `role`/`session_authorization` 为启动参数）、`F003`（LOW：ADR-0029 §5/§8 仍写 provision+grant 两阶段必选）。返工（Herschel / add-backend-dev，actualID `01a190fe-fb35-7921-9ca0-84fae3cf5764`）按三项定向修复：新增 `errAppendOnlyTableAmbiguous` 并在 gate 与 schema resolver 统一按 `current_schemas(false)` 统计同名关系数要求恰为 1（保留 `pg_table_is_visible`，错误类别与“表不存在”区分）；DSN 解析期显式拒绝 URL/keyword 顶层 `role`、`session_authorization`（含百分号解码、大小写与 `options=-c role=` 变体）；`F003` 不改 ADR，仅在证据索引写 `adr_divergence_handoff`。**未改动** `main.go`、`provision.sql`、`grant-application.sql`（before/after 哈希相等）。证据 [`evidence-s1.5/acl-r03/index.json`](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/index.json) SHA256 `E80EFD986653DD91519F64D4A9617C9299B21F50E98C16D09498D47253597D31`（8 个文件 before/after 哈希、`retained-resources.ndjson` 36 条、`lead_reverification` 登记）。Lead 独立复核：逐文件实际 SHA256 与索引一致；实跑 `go build ./...`、`gofmt -l`、`go vet ./cmd/han-server`、`go vet -tags=integration ./cmd/han-server`、`go test ./cmd/han-server -count=1` 全部 exit 0（[`lead-verify-r03.log`](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/lead-verify-r03.log) `B859DCA7…`）；真实 PG 普通与 `-race` 各 **12/12 子测试 PASS**（`ok 2.467s` / `3.868s`，含新增 `ambiguous_same_name_schema_rejected`，[`lead-verify-r03-pg.log`](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/lead-verify-r03-pg.log) `2F1DAB50…`）。ADR-0029 已同步升 r03（§5/§8 改为 `provision.sql` 必选 + `grant-application.sql` out-of-band 重放，新增 §10 search_path 同名关系边界）；索引同步见 [decisions/README.md](../decisions/README.md)。**边界**：仍未取得人的签收、未真实部署/CI、未做备份恢复演练；不证明生产凭据分发/轮换。已派 R03 窄范围独立复审；R03 复审与 Lead Gate 前本卡仍 `[-]`。
- **2026-10-31 S1.5 ACL 子集独立复审 R03 PASS WITH MINOR + Lead Gate PASS（仅 ACL 门禁子集；整卡 `[-]` 不关闭）**：非作者独立复审 R03（Confucius / add-reviewer，actualID `01a190e6-66c8-72c1-ab3a-813582764124`；只读，唯一写入为报告）[报告](../../../../.spec/wip/article-management/review-code-s1.5-acl-r03.md) SHA256 `7A236731FEED830D60503FB17A1F5224DE92A737B605DAD17E57BE0FC5C45D31` / 24911 B：R01 `S15-ACL-F001`–`F011` 全 CLOSED（F010 附残余限制），R02 `F001/F002/F003` 全 CLOSED，新增 3 条 LOW、无阻断；报告自述仅覆盖 ACL 门禁子集，不对整卡 PASS、不代签 Lead Gate。Lead 本轮实跑并留证 [lead-gate.log](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/lead-gate.log) SHA256 `D21DB1F19903FA2D25685A4ADE9C58BE1FF89C4D47E9023FB7B0C269AE608155` / 298 B：`go build ./...` 0（3.1s）、`go vet ./cmd/han-server` 0、`go vet -tags=integration ./cmd/han-server` 0、`gofmt -l`（4 Go 文件 + `deploy/db`）0 且空输出、`go test ./cmd/han-server -count=1` ok 41.973s、真实 PG 18.6 `TestAppendOnlyApplicationACL` 普通 ok 2.570s、`-race` ok 3.466s。**输入漂移处置**：R03 冻结的 `deploy/db/README.md` 为 `9C4F30C6…`（7488 B），复核后由并发写者更新为 `9A47DD10CA1699757658728F7BE7F04FD221FFAD3F025DB0AA699E8F2F29F0E9`（7772 B，17:32:31）；旧版无副本留存、无法逐字差分，Lead 实读新版确认第 4/5 节 gate/search_path/残余风险语义与实现一致，判为表述变更、不改变 ACL 行为断言；`R03-F003` 所述第 12 行业务 DML 表范围缺失与第 5 节“建表→启动 REVOKE”短窗口未披露仍然成立，已转 ADR/文档更正（add-arch-designer 承接，进行中）。**三条 LOW 处置**：`R03-F001` 明文口令仅出现在本机 loopback 合成 DSN 的既有 `run-06.log`/`run-07.log` 与 [index.json](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/index.json)（现 19919 B，SHA256 `0ECAACE5EF5F3915E70C518752542E880CB2591CC181299D90B8E9DBA3B27FA1`，`postgres.dsn` 含同一口令），未外发；按不改写既有证据原则保留，脱敏待用户逐项确认；`R03-F002` 索引缺口已由写者补齐（含 `run-09.log` 与 README after 哈希）；`R03-F003` 见上。**保留资源**：本轮只追加不清理，[retained-resources.ndjson](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/retained-resources.ndjson) 现 36 行 / 9428 B / SHA256 `940609FCEA1E56CA59C7C7C8C247F7398A3F5B5906C087823D3A6C1F04A44FFA`，18 个 schema（9 组 base+decoy）与配对角色；17:32 门禁复跑新增 `han_s15_acl_44142b0d7d60d021`、`han_s15_acl_c8a7001535d3dbee`，17:35 补充复跑新增 `han_s15_acl_1509382e7e7f2404`、`han_s15_acl_c9e729d9f5dc850b`（[lead-verify-r03.log](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/lead-verify-r03.log) SHA256 `B859DCA723654BDECDA9A96E225351F12130CAC4F221D5F51D0FD9F32A8360F4` / 211 B、[lead-verify-r03-pg.log](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/lead-verify-r03-pg.log) SHA256 `2F1DAB5052698A69D38B935AE1F906927981FA88C279DC2D2C58B634862AB4FF` / 11565 B，后者真实 PG 12/12 子测试 PASS；该轮未登记进 `index.json`，属 F002 同类索引缺口，待写者补齐，Lead 不改写他人证据）。**边界**：本 Gate 仅判 ACL 门禁子集；S1.5 整卡保持 `[-]`、原十二 requires、完整关闭条件与 A-CON-03 以外责任不变；不证明真实部署/CI/多节点生产 superuser 绕过/备份恢复演练，不代替人的业务签收、Git 写、清理或 S4.20 实施授权。S4.20 的“S1.5 ACL 可消费输入”条件自此满足（非作者 PASS + Lead Gate PASS），卡内 R02 文字由 Task-Planner 提案同步。
- **2026-10-31 S1.5 ACL 三条 LOW 收口 + 证据脱敏（用户本轮“授权执行”下完成；ACL 子集可核收，整卡仍 `[-]`）**：按用户 2026-10-31 明确授权执行（S1.5 路径B、S4.4 路径A、S8.2 建议正解）与 ADR-0028 项目级授权，Lead 以记录方式处理 R03 复审 §8 的三条核收前置，未改代码、未删任何文件或证据行。**`S15-ACL-R03-F001`（明文 DSN）关闭**：对 `evidence-s1.5/acl-r02/run-14.log`、`run-15.log`、`acl-r02/index.json`（3 处）与 `acl-r03/run-06.log`、`run-07.log`、`acl-r03/index.json`（3 处，含 `postgres.dsn`）就地脱敏，口令段替换为 `<redacted>`；四个日志行数不变（51/51/54/54），除口令外内容未变；同步更新 r02 索引 `run-14.log` `log_sha256`=`C3D90D752A0FCA307B9B96DBE5A20EC090098167F9E7B2465159EDC9DAD74356`、`run-15.log`=`FF376CACF1250B3D27CE359DD415D8C51D3FF9D9B225BC4471B7AE51677A37F1`，为 r03 索引 `run-06.log`=`92CE23AC63D9769783376810D8884644B11FF16E66873D9AAEE3F03187349025`、`run-07.log`=`9D547C7AB8E160D4D68154362F2A780ED0ED796BEFBEE8EF9495A89F606FC01C` 补登 `log_sha256`；两索引各新增 `redaction` 说明块，JSON 重新解析通过。脱敏后索引 [acl-r02/index.json](../../../../.spec/wip/article-management/evidence-s1.5/acl-r02/index.json) SHA256 `F442147B1E60948BA54C4826EA94E112CA06E5E330269B250ED1617ADEB82BAB`（28108 B，原 `7921D68E…`）、[acl-r03/index.json](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/index.json) SHA256 `8ED9FF3E9F8E0CB22691A32F1B394C40AD4A6FDFFCB9E0179525777379DDFCA5`（22037 B，原 `E80EFD98…`）；`evidence-s1.5/` 全目录复扫已无明文口令串。仍含同一 loopback 合成 DSN 的三份独立评审报告（`review-code-s1.5-acl-r01.md:38`、`review-code-s4.4-old-backup-r01.md:39`、`review-code-s4.4-old-backup-r02.md:194`）按“独立评审产物哈希不动”原则未改，待用户决定是否同样脱敏。**`R03-F002` 关闭（过时项）**：`acl-r03/index.json` 于 2026-10-31T09:35Z 由写者落盘，报告写作时点更早，且 `run-02` 文件清单完整存在于索引 `write_scope.new` 与 `commands`，无缺失。**`R03-F003` 关闭（文档更正）**：ADR-0029 升 r04 errata——[framework.md](../decisions/framework.md) SHA256 `04E56C991510A5243F26E3EAB85555E081CC4DA8EBF5F44E1E25B080E9F4AAFA`（911 行）§5 更正为“`provision.sql` 默认授予业务 DML（`SELECT`/`INSERT`/`UPDATE`/`DELETE`）与序列 `USAGE`/`SELECT`，append-only 两表与 `schema_migrations` 由启动迁移在关闭迁移池前 `REVOKE` 收窄”，§7 披露建表到 `REVOKE` 之间应用角色短暂持有两表 `UPDATE`/`DELETE` 的窗口，并更正 r03 引用记录错误前提；[decisions/README.md](../decisions/README.md) SHA256 `35A5623E7E3CFCC140DA0DE9C8B218210C0E34A850A26B5CDD450B81DA40CC63` 索引行与增量段同步 r04，未改 §1–§4、§6、§9、§10 约束。按复审 §8 交接提示同步更正口径：**F011 已由代码 fail closed 关闭**（`current_schemas(false)` 同名关系数须恰为 1），目标 schema 身份仍不证明。**结论**：R03 报告 `7A236731…` 的 PASS WITH MINOR 被 Lead 核收为 ACL 门禁子集结论；S1.5 整卡保持 `[-]`，原十二 requires、完整关闭条件与 A-CON-03 以外责任不变；人的签收、真实部署/CI、凭据分发/轮换、备份恢复演练仍为 unknown。
- **2026-10-31 评审报告脱敏 + 工作树杂散文件清理（用户明确确认后执行）**：三份独立评审报告中的本机 loopback 合成 DSN 口令按用户确认原地替换为 `<redacted>`（各 1 处），结论与 findings 未改，文件末尾追加脱敏说明：`review-code-s1.5-acl-r01.md` `4E9278976025533DD4F61630BF9F8B9E12D779789B0B9BFE230A55DDEF5DC9BD`（原 `8210B2A0…`，135 行）、`review-code-s4.4-old-backup-r01.md` `AF98EED8D1C245B1AF30119B80A68C5AB9DD147D2EBB006FA87E7F66BD4A2EDA`（原 `A1BEB1E4…`，215 行）、`review-code-s4.4-old-backup-r02.md` `1BC70302A8873068CDFF9C022E569E25F2BAE45E10585A04468C848F417B6E17`（原 `7770F3EB…`，215 行）；S4.4 段中的报告哈希引用已同步为新值。`review-code-s1.5-acl-r02.md:32` 与 `review-code-s1.5-acl-r03.md:35` 仍引用 r01 报告脱敏前哈希，属历史引用，未改这两份独立报告正文。清理：按用户对具体清单的确认，删除工作树根下两个早期误操作遗留的非项目文件 `$null`（74 B 错误输出）与 `(Join-Path`（0 B），无其他删除。`.spec/wip/article-management/evidence-s1.5/` 复扫已无明文口令串；全仓复扫另发现 5 处历史证据文件仍含同一 loopback 合成 DSN 明文（`identity-auth/review-code-s6.1-r01.md:118`、`recovery-runtime/evidence-b13h-r01/s419-author-r02/report.md:112`、`s419-author-r03/report.md:49`、`s419-independent-review-r02/report.md:177`、`s419-lead-gate-r03/report.md:37`），未在本轮确认清单内，保留原样待确认。

- **2026-10-31 R03 报告 v1→v2 指针更正、三条 LOW 关闭口径统一与并发写入核验（Lead 记录；整卡仍 `[-]`）**：R03 报告已由评审者发布正式修订 v2（[报告](../../../../.spec/wip/article-management/review-code-s1.5-acl-r03.md) SHA256 `78D8AA2318ED237B9141EECFCEC5372F355EB997387371BA1C0490730E47A238` / 23870 B / 154 行 / 17:46:48），结论同为 PASS WITH MINOR，基线更新为 `deploy/db/README.md` `9A47DD10…` 并补入 ADR-0029 r04 文档-代码一致性核验；v1 `7A236731…`（24911 B）已被同路径覆盖、无留存副本，`:1233`、`:1234` 与 `framework.md:911` 对 v1 的引用为历史指针，现行有效指针为 v2；ADR-0029 r04 errata 引用已更正（`framework.md` 现 SHA256 `5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A` / 126590 B，`decisions/README.md` 不变 `35A5623E…`）。**三条 LOW 关闭**：`R03-F001` 对 `acl-r02/run-14.log`、`run-15.log`、`acl-r03/run-06.log`、`run-07.log` 与两份 `index.json` 就地脱敏（口令段 `<redacted>`，行数不变），Lead 复核 6 个文件 0 明文口令、四份日志 `log_sha256` 与索引逐项一致；`review-code-s1.5-acl-r01.md`、`review-code-s4.4-old-backup-r01.md`、`review-code-s4.4-old-backup-r02.md` 仍各含 1 处同一 loopback 合成 DSN，按“独立评审产物不改写”保留，脱敏待用户逐项确认；`R03-F002` 判为过时项关闭（`acl-r03/index.json` 已含 `run-02` 清单与 `run-09`，写者时点晚于报告）；`R03-F003` 由 ADR-0029 r04 关闭（§5 业务 DML 口径、§7 短窗口、r03 错误前提更正、索引同步）；`deploy/db/README.md` 一句话补充为可选、不阻断，仍未改。**并发写入核验**：本卡 S1.5 记录区另有并发 Lead 会话写入的 `:1234`（LOW 收口/脱敏）与 S4.20 `:2934`（START 门禁现状）；本会话逐项复核其事实成立（脱敏结果、ADR r04、`main.go` 未变），与本会话 `:1233` 记录并存，不删除、不改写他人记录；`cmd/han-server` 四个 ACL 相关文件自 17:21–17:22 未再变化，`main.go` 仍 `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` / 171448 B / 16:36:42；`acl-r03/retained-resources.ndjson` 现 56 行 / 28 schema（并发跑测追加，只追加未清理）。**口径统一**：S4.20 卡面 5 处与 Plan `B13H` 登记中的“三条 LOW 已于 2026-10-31 关闭（脱敏／索引／ADR-0029 r04）”统一为“三条 LOW 已于 2026-10-31 关闭（脱敏／索引／ADR-0029 r04）”。**边界**：本记录仅更正指针与口径，不重开 S1.5 ACL 行为结论、不改变 `lead-gate.log` 证据；整卡仍 `[-]`、十二 requires 与完整关闭条件不变；START 第③重（writer 明确 STOP 且 `main.go` 写窗冻结）仍未满足——另一 Codex 会话（PID 35652）仍存活、最后写入 17:46:48，本会话不宣告写窗冻结，不派 S4.20 实施。

- **2026-10-31 第二批历史证据脱敏（5 份；用户确认后执行）**：承接 `:1235` 记录所列“全仓复扫另发现 5 处历史证据文件仍含同一 loopback 合成 DSN 明文”，按用户对具体清单的确认，将这 5 份文件中的口令段原地替换为 `<redacted>`（各 1 处），finding/结论/命令/行序未改，文件末尾追加脱敏说明段：`.spec/wip/identity-auth/review-code-s6.1-r01.md`（152 行，原 `B76A590923A6F61F1E9E0085B2EA26B6B7DAAA5340DCB46460AAD06C6B2AF944` → `2A612A4B931A5BA4E8B960277D1337AF4477C5CDB6930C5031A91F2A8CD7BCF7`）、`.spec/wip/recovery-runtime/evidence-b13h-r01/s419-author-r02/report.md`（134 行，原 `4D2C6B7EFABFAB770BC9B33A642B20CCF6364FEE3FDE7CFE05AA11D9AA69DB0D` → `DA2C1839490D814AADA4BEF212BEAD6BB3A87B8F0CFCD28F18884B593986629D`）、`s419-author-r03/report.md`（83 行，原 `27179A39E2330DDD73DB614E95FE506DBF1668F8D93F5871C952C9537BA3C510` → `FCD17597A62000A3AD07E963B3715CB87D54CF003E1F32501117A532DC952079`）、`s419-independent-review-r02/report.md`（233 行，原 `E3540E9504D05C3791C936E841EA1AA7CC79A86371F861268CC20C3A529841A6` → `A65523DF48DEC1C24BC096B0BDB8934905A1D709E077B154423E933593F8EF2F`）、`s419-lead-gate-r03/report.md`（58 行，原 `2CC765A9DB7289B0A174900B22624A11CB1C80C7C6BE74D6113CE0C3433208C3` → `81703EC838B841D27313AFCDFA2E434BC60E476B5694F076665762126A3BC19B`）。**引用同步**：`review-code-s6.1-r01.md` 的新哈希已在 `tasks.md`（2 处）与 [ir-01-content-community-batches.md](../plans/ir-01-content-community-batches.md)（2 处）由 `B76A5909…` 更新为 `2A612A4B…`，并保留原值括注；`evidence-s2.2/baseline-hashes.json:24` 保持旧值 `B76A5909…` 不改——该文件是脱敏前落盘的历史基线快照，改写会使基线不可核，故只在此登记差异。**按历史指针保留、未改写**：`review-code-s1.5-acl-r02.md:32`、`review-code-s1.5-acl-r03.md:35`（引 r01 脱敏前哈希）与 `evidence-b13h-r01/s420-planning-review-r01/report.md:23`（引 `s419-lead-gate-r03` 脱敏前哈希 `2CC765A9…` / 3979 B），均为各自评审时点的文件事实记录，属他人评审产物，本次不改其正文。**复扫**：`.spec` 与 `.add` 下全文件类型（不只 `.md`）复扫 `postgres:postgres@` 命中 0；全仓（排除 `.git`）复扫命中 0。**边界**：未做 Git 写操作；未改任何任务 checkbox/日期/requires；未删任何证据行；脱敏不改变评审结论、证据强度或任何 PASS/未验证边界，也不构成新验证。
- **2026-11-01 S1.5 第 7 子门禁独立评审 R01：PASS WITH MINOR 并 Lead 接收（仅子门禁；整卡仍 `[-]`）**：非作者独立评审（Halley / add-code-review，actualID `01a1972f-65d8-76a1-8912-7bf5b437d291`，只读，唯一写入为评审目录）[report.md](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-review-r01/report.md) `B51E2F55CF03A408DFAE891C7F4C52595FEF35A5BA357236BE21CC297D0ED134` / 14173 B、[verdict.md](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-review-r01/verdict.md) `812A56FE8B80E9789BBBF3C41DC5C168CB0ED335E6CD482DD4D6F958CB79CB6E` / 796 B、commands.log `6FB81EE8A8770D588B4BD75D32754C6D4580DB8A38F04E5931326EA85FBFD5B4` / 12824 B；Lead 已实读并复算三份哈希一致，评审前后 branch `main`、HEAD `6da8abbc…`、`git status --short` 291 行不变，评审目录外零改动。**六项断言全 PASS**：上游 index 106 登记 / 107 实物 / 0 缺项 / 0 哈希或字节不符；10 个运行绑定源与 `replay_evidence.go`（`7C14E74D…` / 34397 B）现值逐项一致，但 177 项保护清单只证 B13d entry→exit 期间未变、当前工作树相对 exit 有 28 项漂移，不称全量稳定；两轮 58 pass / 0 fail / 0 skip 由 stdout 独立重算并与 leaves/result 逐项一致、均无 `-overlay`、无 DATA RACE、进程 job normal 9/9 与 race 45/45 收口；`run.py` 无环境 DSN 回落、空 DSN 即 `Fatal`；PG 18.6 / `127.0.0.1:51934` / `han_b13d_article` 由 preflight 原始日志支撑；消费登记未放大，`not_proven` 与 `known_gaps` 完整。**Finding**：`S15-PR-F001`（LOW，来源包不自包含复跑——缺 `source10-before.json`、依赖包外 PG resource 快照与不存在的 sentinel 文件；不改他人证据，如实登记为已知局限）、`S15-PR-F002`（LOW，消费报告命令块含 shell 单引号、与 `commands.json`/`result.json.argv` 非逐字一致；已在消费登记改写为展示口径并指明 argv 以 `commands.json` 为准）。消费 [report.md](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-r01/report.md) 现为 `1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749` / 8267 B（原 `A9A4ECBB…` / 8023 B 保留为历史指针），[index.json](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-r01/index.json) `F840BC2C…` / 19567 B 未变。**复跑 NOT RUN**：`HAN_TEST_POSTGRES_DSN` 已存在、`psql` 为 18.6，但测试 helper 会创建并保留 schema，在不建库/不删库/不 DROP 既有对象前提下无法确证目标库即现役测试库，故以已保存原始产物离线复算为准。已同步 [verification-s1.5.md](../../../../.spec/wip/article-management/verification-s1.5.md) 第 7 行与 unknown 段并追加接收记录（新 SHA256 `688B144FECF339DA7C0CE8CF34D8201C482CEA5167E0809AD225BB63E81D4F5C` / 35041 B / 159 行，LF、无 BOM，原 `7CF8D81B…` / 31189 B 保留历史指针）。**边界**：本记录只把该子门禁推进到“已评审且已接收”，S1.5 整卡 `[-]`、十二 requires 与完整关闭条件不变；不宣称真实 Journal/HMAC/keyring、旧备份恢复、Case18、应用角色 ACL 载体、部署 E2E、S8.2 或 production readiness；AI 代理评审，非真人签字，不代替人的业务接收。

- **2026-11-03 S1.5 消费 S8.2 Case18 执行证据 r01 的独立评审与登记补正（整卡仍 `[-]`，十二 requires 不变）**：非作者独立评审（Anscombe / add-reviewer，actualID `01a1a0b4-231d-7730-b702-a0aaad930a07`，只读）对 [case18-current-r01](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01/) 判 **CHANGES REQUIRED**（报告 `AAFFA185…` / 19588 B、verdict `AE2BC8D3…` / 3082 B、commands.log `71C1619C…` / 9985 B，Lead 已复算一致）：`C18-COV-F001` 仅 finding 级 CLOSED（当前 revision root/装配普通与 `-race` 各 93 PASS 独立复现、8 源文件绑定一致）；`C18-COV-F004` 整体仍 open（Session 侧已由真实 PG + `session.NewStore` 覆盖、AccessGeneral 侧未覆盖）；`C18-COV-F002` / `F003` / `F005` 保持 open。新增 `C18-R01-REV-F001`（MEDIUM，保留 schema 少计 29：实为 28 个 `han_admin2_*` + 3 个 `han_s33_*`，Lead 已按 run 日志全量去重复核）与 `C18-R01-REV-F002`（LOW，准入状态词表漏 `checking` 并混入 observation/替身词；实读 `internal/recovery/runtime.go:27-32`、`internal/platform/recovery_guard.go:59`、`runtime.go:1015` 复核属实），均由 [registration-correction-r01.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01/registration-correction-r01.md) `CFDCE04C…` / 4604 B 只追加更正，原 `report.md`、`index.json` 与 8 份 run 日志零字节改动。**对本卡影响**：“真实 Session/SQL 载体”仍以本卡 producer 第 7 行与 S8.2 运行 8 的可核证据为准，本轮不新增执行、不改变任何已登记结论；`S8.2` 整卡门禁、Case18 全局恢复隔离、真实 Journal/HMAC/keyring、旧备份恢复、真实部署 E2E 与人的接收（或 ADR-0028 下登记的 AI 代理接收）六项仍未满足，本卡 `[-]`、十二 requires 与完整关闭条件不变；finding 级 CLOSED 不等于整卡或 Case18 关闭。详见本文件 S8.2 卡 2026-11-03 评审记录；本轮未改代码、未重跑测试、未做 Git 写/部署/删除/数据库清理，评审为 AI 代理，非真人签字。
- **2026-11-03 S1.5 消费上述评审闭环结果（整卡仍 `[-]`，无门禁变化）**：同一非作者评审 R02 判 **PASS WITH MINOR**，`C18-R01-REV-F001` / `C18-R01-REV-F002` 均 CLOSED，`C18-COV-F001` 登记为 finding 级 CLOSED（[report.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01-review-r02/report.md) `C732752A…` / 10370 B、verdict `BB8E3B41…` / 2651 B、commands.log `E8009A40…` / 6694 B，Lead 已复算一致）；`C18-COV-F004` 整体仍 open，`F002`/`F003`/`F005` 保持 open。对本卡：真实 Session/SQL 载体证据链维持原结论，“S8.2 整卡”门禁不变；六项未满足（真实 Journal/HMAC/keyring、旧备份恢复、Case18 全局恢复隔离、S8.2 整卡、真实部署 E2E、人的接收或 ADR-0028 下登记的 AI 代理接收）不变，`[-]` 与十二 requires 不变；本轮未改代码、未重跑测试、未做 Git 写/部署/删除/数据库清理。
- **目标/类型**：专项契约测试；保留固定设计的16个endpoint、三投影、公共错误、S1–S16及Ready20/Unavailable6合同，不以局部组件结果代替生产注册或真实依赖证据。
- **最新r19真实Audit SQL子集交付（Lead接收，整卡不关闭）**：唯一新增 `test/integration/article/audit_insert_rollback_test.go` SHA256 `468964647524860CEEF0167EC56A9C0033FBB40CFF340A5021FBCE66F2C89CD5`，原178项不变、当前输入179；原四contract r03/236leaf冻结不重开。固定r19 §4.7.2 的唯一top `TestArticleAuditInsertRollbackContract` 四leaf均真实命中：作者Dalton及非作者Erdos的普通/race分别各5测试节点run/pass、4leaf、0fail/skip，tagged vet各exit0；真实23505、同caller tx已写入见证、独立backend PID完整前后像/NOWAIT及fresh-ID重试commit均有固定证据。Finalizer保留已独立提交claim，只回滚completion；Journal仅controlled普通匿名分支，不证明真Journal/全类型删除。[Erdos独立CodeReview R01](../../../../.spec/wip/article-management/review-code-s1.5-audit-sql-r01.md) SHA256 `DF4D48296BF532401DB1B26B2F0BBB78486FF943DB6E01BFE93F7EAF8C99587A` 两轮有界PASS/0finding/stopped；Lead已实读945行新文件、报告/raw/179集合并明确接收SQL子集，此为Lead实际接收/metadata授权，不代人的accepted/checkpoint。[本轮Lead Manifest七项](../../../../.spec/wip/article-management/evidence-s1.5/audit-sql-implementation-r01/lead/lead-manifest-audit-r01-results.json) SHA256 `B6570C3E41B639BB36A2366182B9CBDA2D0B99FFC439264663B9C7C968067931` 全exit0/gofmt空/179前后一致；本专项race取带integration tag精确运行，默认全仓race不覆盖本文件，不称full-profile/全仓平台0skip或production/E2E通过。
- **本次固定证据/运行前提及未闭责任**：[实施索引](../../../../.spec/wip/article-management/evidence-s1.5/audit-sql-implementation-r01/index.json) SHA256 `4B93DD5351EDF960C347502148E234521A103932A911E7B8FB844860328496BF`（364份）、[独立评审/Lead接收索引](../../../../.spec/wip/article-management/evidence-s1.5/audit-sql-review-r01/index.json) SHA256 `A4CF2B928AFBF9DF6F9466578A4D1824CA3EBEBEA7BBE1C301CE748E17A7A168`（85份）；[Lead receipt及复跑说明](../../../../.spec/wip/article-management/verification-s1.5.md) 本轮SHA256 `9EF6EF3B264807685F372717D09106C6879786DA571B3BBBD4C943F8BA5B8E93`。runner须预建保留父目录、设置 `HAN_AUDIT_EVIDENCE_DIR` 并安全注入批准User DSN后跑r19精确命令；缺目录probe实际1leaf FAIL/1schema、缺DSN四leaf FAIL/0schema，均为预期配置失败。作者仅Temp `%w→%v` 负控保留真实23505、四leaf错误链断言FAIL，非产品RED。作者16/Reviewer9个schema全retain，资源按断言/退出日志close、children ended，不声称全库盘点/全机普查；新文件不改。SQL失败子集已交付，应用有效UPDATE/DELETE ACL仍未实施未跑/载体unknown；真实Provider/Source/Journal/旧备份/全局恢复隔离与完整成功开流、S8.2/Case18及完整S1.5仍blocked，原12requires/fullclose/[-]不变，不扩identity S10.5、不推断root writer释放。双方已stopped，本次结果不自动释放后续写窗。下方尚未CODE/LOCALPG、SQL/ACL两top未跑、NOT RUN等是各自规划/r03点时历史，仅SQL层由本次交付接续；ACL及其余未验层不被外推为PASS。
- **r19真实审计SQL增量（规划登记时点历史；当前交付见上）**：[test-plan r19](../design-workspace/article-management/test-plan.md) SHA256 `88296F772A19600A237CCA9F47A16D4B673828EF41E5C1D0C2D88924BE2C6F66` §4.7.2、[Pauli独立TestPlan R01](../../../../.spec/wip/article-management/review-test-plan-r19-audit-sql-r01.md) SHA256 `A89FF3A126980C7DC1CB85F9468084ACAD716FE47F33F9C80619DE0278013F1B` 两轮有界PASS/0finding/stopped，来自Lead本轮交接/实际metadata授权；仅规划可实施，不是SQL/PG/CodeReview/运行PASS。原r03四contract组件交接继续按固定52298 CodeR02/8904 Manifest/Lead receipt保留且全冻结；本次只新增第五integration载体，r18“四文件唯一载体/PG内承ports”等为点时建议。原卡保持[-]、原类型/全部十二requires/完整关闭标准不变；未来原Dalton本片G0/CODE/LOCALPG须Lead单独发窗口，旧四文件START不扩权。
- **最新r03四文件组件局部接收（2026-10-28；任务不关闭）**：Dalton仅rest测试修复wire独立262144合同值，actual +3/-2，新rest SHA256 `CEBA105C2CBF70B91CB5DFF93D9D4FC37077153A7EDB4F9C2806C29E650E9533`，其他177项不变/固定178，生产/API未改；四冻结hash与当前作者/源/三项编译后语义负控见[fix-r03索引](../../../../.spec/wip/article-management/evidence-s1.5/fix-r03/index.json) SHA256 `6F0A6D5D86CF10B7A384DC1621086D1F4424050F54E1ACAC6AD0834D649F856A`。作者及原非作者Erdos普通/race各286测试节点、236leaf、23风险组/16个Handler组件分支、目标0fail/skip；不是286项业务验收或全生产16路由通过。新wire负控24run/20leaf仅Create/Replace两个Nplus1叶实际FAIL、两个N叶PASS，原源码完整GREEN；三mutant是检错有效性证据，非生产修复RED。[原R01](../../../../.spec/wip/article-management/review-code-s1.5-components-r01.md) CHANGES REQUIRED SHA256 `D88CCCC9B6448E154266B00E480E2D998F763E64D87E8660463D2A2795A01938`及旧r02/旧Manifest0DC/全部失败链保留为历史；[原Erdos R02复审](../../../../.spec/wip/article-management/review-code-s1.5-components-r02.md) SHA256 `52298FEDD5F13E9A9017334C1FEA72A1BD72C9A9658A2384660D9075D6F59933` 独立CLOSED唯一 `AM-S15-COMP-R01-F001`、有界组件CodeReview PASS/无新finding，[27份复审及Lead核证索引](../../../../.spec/wip/article-management/evidence-s1.5/review-r02/index.json) SHA256 `486764B054A92AB3A6B39F23FA39C89EB66B2922539315314EEDB7341D619AD5`。Lead本轮实读报告、核raw/178集合/受保护177及目录恰四Go后，明确接收本轮四文件组件子集；完整任务仍[-]，前文pending/START/未创建/NOT RUN均为各自点时记录。
- **r03组件验证绑定/未闭责任（点时历史；当前SQL交付见上）**：仅以r03同版[Lead新Manifest七项回执](../../../../.spec/wip/article-management/evidence-s1.5/fix-r03/lead/lead-manifest-r03-results.json) SHA256 `8904DA4630D4441B7EBE06C15A86FD24C49190CBE46AFCC3CFE7B06F37E61047` 绑定当前178输入，实际七项exit0/gofmt空、集合/字节前后一致；默认全仓race不带contract，新四文件race取带tag的作者/Reviewer局部证据，目标0skip不外推全仓平台0skip；goimports/golangci-lint未跑，不称full-profile PASS。事实接收及层次见[Lead receipt](../../../../.spec/wip/article-management/verification-s1.5.md)。Publish/Transfer旧root raw本片未逐项消费，S14/S15无新组件测试、S6.2/S7.2旧raw未重核/旧层不重签PASS；r03时点真实Audit INSERT失败/实际应用roleACL两top未实施未跑无skip；当前SQL子集已交付，应用ACL仍未跑/载体unknown，受控Provider.ReadCurrent非真实Store/Source。真Journal/旧备份/全局恢复隔离/完整成功开流、Case18整体/S8.2/full S1.5 gate仍blocked，完整12requires/原完整关闭条件不削减，真实层仍待原责任，不套identity ADR0025豁免、不代签人的accepted/checkpoint，不称整卡/B10 fullPASS。作者/Reviewer均stopped，本片source写窗收回Lead、后续变动另协调；root writer用户事实仍unknown，读取/摘要核对不解除共享写窗。
- **Lead实际组件CODE/LOCAL START（2026-10-28最新）**：[Pauli TestPlan R01](../../../../.spec/wip/article-management/review-test-plan-r18-s1.5-r01.md)最终SHA256 `85A9E111677DAC31E4A99A31D07F1FB8CFBB3E263CF1574196BFF9EA30D50A9E` 对固定r18有界PASS/0finding，仅来源措辞纠正、非新评审。Lead在 `2026-10-28T02:47:13+08:00` 启动核对后已向唯一Dalton实发CODE WRITE＋LOCAL TEST START，submission `01a17eb0-1fd2-7293-bc61-8b581fe05510`；只原四文件/四固定top及r18全部必要leaf（含16 Endpoints），准局部普通/race/tagged vet及个人Temp可编译行为负控，不授PG/root/production/identity/旧测试/fixture/module/额外路径或API权限。固定准备/前像/actualdiff/G0/启动核对见[18份准备证据索引](../../../../.spec/wip/article-management/evidence-s1.5/pre-start-r01/index.json)，最新授权见[Lead收据](../../../../.spec/wip/article-management/verification-s1.5.md)。前文“待Review/START/仅准备”和未创建/NOT RUN均为对应准备时点，当前以本段实际信号为准；自测尚未回交，不填运行PASS/交付接收，Lead Manifest七项待source freeze。十二requires/完整关闭条件及真实层blockers保留，非S1.5完整START或B10全批START。
- **本轮当前依据/真实准备认领（2026-10-28）**：本卡仅原四文件的独立组件子集准备/认领先行，完整十二项requires与完整验收/关闭条件不削减；仅该子集可并行，不表示全卡或B10全批START。Lead本轮已核真实Backend Dalton `01a17dcf-f2ed-7dc2-b7f3-6c72cf771448` 的G0 ready/stopped、四文件范围可行及174原输入集合无漂移，并明确授予本次metadata认领登记；不把Planner本次落盘当代码授权。登记时刻 `2026-10-28T02:40:53+08:00`（本地），开始日期取本轮真实认领登记日；G0早期“待Ohm映射/待Planner认领”等是形成时历史，非CODE WRITE START。
- **固定测试规划/尚待门禁**：[test-plan r18](../design-workspace/article-management/test-plan.md) SHA256 `BCBCFDD8CD1747540726A46C35D565FF1CC211AAB43F6059145474568841DE42` §4.7.1为四top/风险leaf、16个Handler组件入口及S1–S16逐层责任的唯一明细；Ohm已固定停写并交Dalton刷新，[作者交接](C:/Users/J00010385/AppData/Local/Temp/han-s15-testplan-r18-e9baed6003ee445f8d91268b52f58db8/handoff-s15-testplan-r18.json)明确全部新增测试NOT RUN。独立TestPlanReview仍PENDING，Lead CODE WRITE START仍PENDING；当前只授准备认领，无代码/测试/PG/env授权，不等待Review才诚实记录已发生G0。
- **实际G0来源**：Dalton [g0-prep.md](C:/Users/J00010385/AppData/Local/Temp/han-s15-dalton-prep-eb08cc7d8d084642aa10ce0c37f26ffa/g0-prep.md) SHA256 `6F135040DD13E5B6306E14D86166839C7AA5E99E6DEBB645E227A09DF56F2939`；[readset/baseline](C:/Users/J00010385/AppData/Local/Temp/han-s15-dalton-prep-eb08cc7d8d084642aa10ce0c37f26ffa/g0-readset-baseline.json) SHA256 `1049770182E3A954A444BAB16642374C14D8A2AE8F1066AA3B5CEAF494AAEF90`。仅四个新增contract文件将由Dalton单writer承接，不改production/旧测试/fixture/module/迁移/身份/main。共享root writer仍待用户事实，摘要相同不等于无writer；本卡不写/跑root。Pauli/Erdos为非作者代码Reviewer候选，具体范围/实际承接由Lead续发，未代填Review PASS。
- **r18绑定后G0刷新（已发生）**：Dalton已完整读取固定r18 §4.7.1并确认四top全部静态可行、组件子集不需额外路径或真实资源，[r18-g0-binding.json](C:/Users/J00010385/AppData/Local/Temp/han-s15-dalton-prep-eb08cc7d8d084642aa10ce0c37f26ffa/r18-g0-binding.json) SHA256 `1AA4C5F3FFF5A177E6F06CC8E50ED8B62420836341B5049E2028561C5FF451A8`、[r18-coverage-prep.md](C:/Users/J00010385/AppData/Local/Temp/han-s15-dalton-prep-eb08cc7d8d084642aa10ce0c37f26ffa/r18-coverage-prep.md) SHA256 `1FD1F069A44D95CA367B99A1CB47B974D3BFC6ABD01BD1CA9BE4C0FF8C68F809`。G0 ready/stopped、174零漂移与四路径仍缺按其绑定及Lead实核来源登记，未编译/运行测试；以r18的articlecontract统一包/固定top刷新早期候选，不把本静态可行性当Review/代码START。
- **R gap-map（原2026-10-27；本轮按固定交接更新当前承接）**：

  | 范围 | 标签/真实证据 | 承接 |
  | --- | --- | --- |
  | 现役Handler/Service、投影、错误/ETag、内部端口、Provider/PGStore/source入口 | 已建成（静态可调用，待本卡契约验证）；r01 §4～6及现役源码。旧S1.2/S1.3/S2.1/S3.1组件关闭保持有效 | 原S1.5四文件验证，不重开旧卡 |
  | 三个现役Handler的生产接线 | 已建成/已closed：S1.6已交付；原11method/path、POST/PUT405和作者GET缺注册系旧调查时点 | 消费原S1.6固定root证据及S8.2后续同版兼容证据，不重开S1.6或假称本卡已验 |
  | 普通管理员列表/详情两个GET | 九文件局部实现/U/C/R/P及CodeR02/Manifest已有固定交接；S8.2仍[-]，Case18两项系统恢复未交付/整卡blocked | 十二requires中S8.2仍未全满足；本卡仅不依赖该缺项的四文件组件准备先行，不减完整验收 |
  | 四个contract文件及其运行证据 | 待做；当前均不存在，未创建/未执行 | 保留原四文件与16项完整关闭要求；未发现应废弃范围 |

- **文件**（原四contract已冻结；本次仅第五integration新增，不授生产或旧文件写权）：
  - `test/contract/article/rest_contract_test.go（新增）`
  - `test/contract/article/internal_ports_test.go（新增）`
  - `test/contract/article/projection_contract_test.go（新增）`
  - `test/contract/article/replay_status_producer_test.go（新增）`
  - `test/integration/article/audit_insert_rollback_test.go`（本次新增；integration）
- **r19第五文件唯一top/四leaf**：`//go:build integration`、`package articleintegration`；`TestArticleAuditInsertRollbackContract/{MarkRollbackAndRetry,RestoreRollbackAndRetry,TransferRollbackAndRetry,FinalizerSQLRollbackAndRetry}`，新增fixture/adapter/观测helper全部内承本文件，不改旧harness/contract/生产/identity/main/module/迁移，不加第六文件。本段为实施前合同，四例当前已按最新交付段接收；默认contract命令不包含本文件，旧S22/S82 top或全integration结果不替本片。
- **真实SQL/独立oracle/重试**：先提交完整合法的合成audit碰撞行，仅目标业务审计ID返回该既存UUID；真实Command.Moderate/TransferOrphan或公开Finalizer.Finalize→Repository→audit.Store.Append，保留目标SQL/事务。观测wrapper在真实Append前用收到的同一caller pgx.Tx读到业务写（mark/restore版本类别、transfer新author/Notice双recipient、finalizer物理DELETE且operation未completed），随后原样Append取得实际Exec/返回链的`*pgconn.PgError`、`23505`、`audit_events`/`audit_events_pkey`及本例schema；不伪PgError、trigger或成功INSERT后synthetic error。失败后独立backend完整前后像与碰撞行不变、实际FOR UPDATE NOWAIT按现役锁序取锁证明rollback/释放；合法fresh-ID重试真实commit后另连接核业务/通知/审计绑定，不只count或同连接重入。Transfer只消费真实workflow生成的proof并同tx VerifyCompletion，不自造authority；低敏记录阶段/code/constraint，不泄SQL参数/正文/邮箱/DSN。
- **Finalizer completion边界/partial**：最小合法anonymous published/source=NULL普通删除，先有绑定operation/pending Article，Journal seam返回正确绑定的合成tombstone；真实claim独立commit合法保留，分调用前、claim提交后/首业务写前、失败后三像。审计23505只回滚completion，Article仍pending完整、audit_event_id/completed_at为NULL，operation的已提交finalizing/owner/lease不要求回到调用前。fresh AuditID重试须按真实locked_until推进受控UTC微秒clock至过期、公开Finalize重新claim；不手改lease/私有complete/伪proof。本leaf只证真实PG完成事务，不证FileJournal验链/fsync/POSIX/真卷/全类型或orphan清理/旧备份。实际公开seam若不足，仅hold该leaf/交精确缺口，其他安全三例可交证据；四例未齐不得称本SQL子集完整PASS，禁止skip/空成功。
- **执行环境/retained资源门禁（实施前门禁；当前运行见最新交付段）**：获Lead本片CODE/LOCALPG后才安全进程注入本次批准的User级DSN，先解析主Host/Port及所有fallback均literal loopback:5433，再连接核实际地址/端口/version180006/身份，均在CREATE前；环境时点事实不替本片运行。新文件自承每例随机schema独立CREATE/commit并立即记录retained_schema/case/阶段，再另tx完整0001～0009迁移，失败schema亦retain、每连接同search_path/合法CHECK/FK（pending先operation）。只读复用安全同包片段，不直接调用旧同tx CREATE迁移/自动DROP建库阶段。rows.Close、tx rollback、context cancel、pool acquired归零/Close及实际goroutine join须留证，全部schema/log成功失败均保留、不DROP/cleanup。缺DSN/目标版本不符/资源未收束选中本top时FAIL，不skip，不改auth/建role/GRANT或换实例。
- **r19原定命令/接收（规划时点历史）**（仓库根；登记时以下全NOT RUN，当前实际运行及runner配置见最新交付段；选定top及每leaf普通/race真实run/pass、0fail/skip/缺leaf，保存实际timeout/命令/退出码/raw/PgError/独立像/锁/重试/schema/资源与新文件/冻结依赖before-after；合法冲突即真实故障，可first-GREEN，不强造产品RED；额外负控仅另获授权Temp完整snapshot/overlay编译后真实行为FAIL）：

  ```text
  go test -tags=integration ./test/integration/article -list '^TestArticleAuditInsertRollbackContract$' -count=1
  go test -tags=integration ./test/integration/article -run '^TestArticleAuditInsertRollbackContract$' -count=1 -json
  go test -race -tags=integration ./test/integration/article -run '^TestArticleAuditInsertRollbackContract$' -count=1 -json
  go vet -tags=integration ./test/integration/article
  ```

- **r19未解除责任/停止**：Lead未来核同版raw/diff/Manifest、非作者Reviewer仅评新第五文件/实际SQL子集，不替人的accepted/checkpoint。A-CON-03应用角色有效ACL/安全注入载体仍unknown，ACL top未实施候选，本轮不再给其决定新路径或将两审计top合跑contract；测试DSN/superuser/无方法/trigger拒绝不证明ACL。真实producer/source/Journal/旧备份、Case18全局隔离/完整恢复后开流及S8.2/full S1.5 gate和全部12requires保留，不扩identity S10.5。身份/架构/root writer事实不由本片接管；未知writer/冻结依赖变化/额外路径或API/实际产品错误，停受影响部分、保留失败schema/raw/source交Lead，不改生产/放宽断言/清理；本metadata登记不发CODE/LOCALPG START。

- **固定组件子集/top/证据边界**（四文件统一contract tag及同一测试包，helpers内承四文件；详细leaf只引用r18 §4.7.1，不复制另一映射表）：

  | 原四文件 | 精确top | 局部新增证明 |
  | --- | --- | --- |
  | `rest_contract_test.go` | `TestArticleRESTContract` | 真实公开Handler/transport/encoder/decoder；16个Endpoints行为leaf、严格JSON/wire、前置条件、公共错误、Admin优先级与响应写失败；不证明root注册/安全链/SQL |
  | `internal_ports_test.go` | `TestArticleInternalPortsContract` | Principal/作者reader/canonical adapter/清理快照/Gate purpose/错误retry/非键式幂等/恢复DTO的参数、失败、次数与顺序；签名仅辅助，不证明真实锁/ACL/事务 |
  | `projection_contract_test.go` | `TestArticleProjectionContract` | 真实Policy→JSON，成员/作者/Admin及匿名拒绝、隐私canary的精确键和独立逐值oracle |
  | `replay_status_producer_test.go` | `TestArticleReplayStatusProducerContract` | 公开Provider.ReadStatus→真实serializer/共享decoder，Ready20/Unavailable6/坏evidence/consumer mutation；受控Store.ReadCurrent只返回固定值，不冒充真实PG/Source/Journal |

- **局部有效性/命中门禁**：四top及r18必要leaf普通/race实际run/pass、0fail/skip；缺leaf、零命中、只反射/编译、placeholder或t.Skip不能通过。真实服务/Policy/Provider不得由返回预期公共code或复制私有memoryStore/DTO/canonical/Guard替代。r18所列可编译行为负控仅未来获授权Temp snapshot/overlay取证，不改生产工作树；first-GREEN如实标注，不能以编译/环境失败冒充行为RED。全包局部GREEN只证明已实现组件集合，未验真实层/全责任仍blocked。
- **后续定向命令**（仓库根目录；全部NOT RUN，须TestPlanReview及Lead另发代码/执行信号后再实施）：

  ```text
  go test -tags=contract ./test/contract/article -list '^(TestArticleRESTContract|TestArticleInternalPortsContract|TestArticleProjectionContract|TestArticleReplayStatusProducerContract)$' -count=1
  go test -tags=contract ./test/contract/article -run '^(TestArticleRESTContract|TestArticleInternalPortsContract|TestArticleProjectionContract|TestArticleReplayStatusProducerContract)$' -count=1 -json
  go test -race -tags=contract ./test/contract/article -run '^(TestArticleRESTContract|TestArticleInternalPortsContract|TestArticleProjectionContract|TestArticleReplayStatusProducerContract)$' -count=1 -json
  go vet -tags=contract ./test/contract/article
  ```

- **真实层责任保留**：A-CON-03真实Audit INSERT失败子集已按r19第五integration文件交付，应用身份有效UPDATE/DELETE权限仍待载体/授权；S2.2成功INSERT后wrapper返回error只证明事务回滚，不关闭INSERT失败义务。原r18“两审计top内承ports”仅历史；真实INSERT top已按r19第五integration载体交付，ACL top仍未实施候选/载体unknown，本轮不改定其路径；不创建占位/skip，不读secret/建role/GRANT。实际应用身份/注入位置unknown；Provider受控ReadCurrent组件不证明真实Publish/ReadCurrent/source/旧备份。S13/S15/S16通用恢复隔离/完整恢复后开流仍未交付，不借Case18局部或身份豁免关闭。
- **最新已闭前置/原证据消费（不是S1.5本轮执行）**：S1.6已closed，[核证](../../../../.spec/wip/article-management/verification-s1.6.md) SHA256 `A5ECED19F435E233EF3C68E02C40CEA0238D70BF0962C6C596CC586C5931F0A6`仅当时root/14入口等实际范围；S8.2九文件局部/真实PG已交回，[核证](../../../../.spec/wip/article-management/verification-s8.2.md) SHA256 `42AE7A0A7488EFFC28B7EF9060E94519826D9232FC7EFCEF46FAA0910B44A412`、[联合Code R02](../../../../.spec/wip/article-management/review-code-s8.2-joint-r02.md) SHA256 `F1EE8BF2605C3755835E2A1C0F4DFDA354697BD4EEBE52FE8B8A80E0681B7808`有界PASS，[Manifest七项回执](../../../../.spec/wip/article-management/evidence-s8.2/final-r01/lead/lead-manifest-final-r01-results.json) SHA256 `D333D9E1146C84E157774FD2C32E058E6B5B27C25066A53B51AA83CBA09CF62C`。这些固定结果只按r18逐层源/leaf/hash映射消费，不代表S8.2全闭、S1.5执行或当前未知writer后的root已重验；原旧identity PASS及失败链继续保留。
- **证据分层/适用责任**：exported Handler/Policy/serializer可受控依赖验证JSON/错误/ETag/字段隔离；method/path及Origin/Session/CSRF/Principal接线须引用真正cmd root同包测试与独立Review，不手造第二router或URL证明注册。AdminArticleView恰12个顶层字段，不加第13字段；普通admin GET不可由orphan查询或裸投影冒充。S1–S16逐接缝列真实消费者/失败语义与适用证据，反射签名不能单独证明行为。A-CON-03另列真实PG审计INSERT失败回滚、实际应用身份的UPDATE/DELETE有效权限与操作结果，由本卡测试owner对账、Lead/DB环境owner提供隔离授权；trigger拒绝不等同ACL，superuser连接不替代应用角色，当前权限载体/执行责任未核实即pending，不自行建角色/GRANT或假PASS。真实Publish/ReadCurrent/source证据先消费原S4.2的精确版本/现役PG验证，真实Journal/inventory/ReplayAll及旧备份恢复仍归原S4.4等专项；缺项仅标对应层未验，不把真卷或部署E2E倒挂纯组件验证。Provider受控serializer仅证明组件路径，不能把合成Ready当真实source/持久化；不复制私有memoryStore/fixture状态机，不另建DTO/canonical/Guard。固定r21/原调查已核通用恢复生产联动未交付，当前共享root writer另待用户事实；不据此创造Guard/API或运行root。
- **验收命令**（workdir：仓库根目录 `.`；保留原命令，均未运行）：
  - `go test -tags=contract ./test/contract/article`
  - `go test -tags=contract ./test/contract/article -count=1 -json`
  - `go test -race -tags=contract ./test/contract/article -count=1 -json`
- **当前状态/命中门禁**：四文件未创建未执行；原粘贴identity integration PG/Manifest PASS仅为其他卡历史，不作为本卡证据，旧证据不清理。执行前核四文件contract tag、真实顶层/风险leaf及分层覆盖映射，实际命中零fail/skip；零用例、缺风险、[no tests to run]或只serializer/spy命中不能宣称16endpoint/真实PG通过。Lead按本卡适用范围核Manifest固定七项及原始结果；构建/普通全包不能替代contract tag、真实root、PG权限或独立Review。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.5`、`article-management-S1.3`、`article-management-S2.1`、`article-management-S3.1`、`article-management-S4.2`、`article-management-S5.1`、`article-management-S6.2`、`article-management-S7.1`、`article-management-S8.1`、`article-management-S1.6`、`article-management-S8.2`；原十项及已补S1.6不删改，本次仅新增实际Admin2实施前置S8.2，无S2.2反向业务依赖。
- **批次/责任/并行**：仍唯一B10，按Lead本轮实际核G0与指定承接登记[-]，仅四文件组件子集准备认领。唯一Backend Dalton及G0/hash见本卡当前记录；Ohm给固定映射、Pauli独立TestPlanReview进行中未PASS，非作者代码Reviewer待Lead实际指定。Lead核准备认领后仍须另发CODE WRITE START；当前不写测试/运行命令，不读取DSN/PG/env，不把全十二requires当已满足或B10全批START。root/PG/真实应用ACL等窗口分别协调，不能用业务depends假代文件锁。
- **阻塞/停止条件**：S1.6已closed；S8.2九文件局部及有界CodeReview/Manifest已交付但完整Case18/整卡未闭，十二requires完整满足与S1.5完整验收仍blocked。r18已固定、独立TestPlanReview与Lead代码START尚pending；共享root writer、A-CON-03真实INSERT失败/应用角色ACL、producer真实层/Journal/旧备份/系统恢复证据按r18各自责任保留未验。只阻塞受影响验证/完整关闭，组件准备可先行；16endpoint/三投影/公共错误/S1–S16/Ready20/Unavailable6不削减。额外文件、生产/身份/main/依赖/权威变更、未知writer或固定输入漂移，停相关范围交Lead；不复制私有fixture状态机、不placeholder/skip/fake PASS，不重开旧卡/套身份豁免/清理旧证据。
- **本任务关闭条件**：16endpoint真实method/path与适用安全链、三投影allowlist、公共错误、S1–S16、Ready20/Unavailable6及上述适用真实层证据全部可核；Admin2未落实不关闭。普通/race命中、独立非作者Review/必要复审、Lead同revision的diff/hash/范围/实际接收齐全；不以原卡历史PASS、fakeReady或signature达成。
- **后续验证关联**：article-management-S1.4
- **来源**：既有Story1/3及r18接口总表/各Story验收、r15 A-CON-01～04/A-REC-01～03；r01研究 `.spec/wip/article-management/research-s1.5-readiness-r01.md` SHA256 `77C32EC9ED2AAA14C2D1D6DBD48E7584715B9976B5C2D66D1339F8F9186B509A`完整保留；r02 errata `.spec/wip/article-management/research-s1.5-readiness-r02.md` SHA256 `C6B333DDD649D224EA8DCCAE02BFAFC05D0B00F15ECAE4E0D49D8C329EC9177A`更正Admin恰12字段，其余结论不外推PASS。

- [x] article-management-S1.6 接通现役文章创建与作者读写的唯一生产root 开始@2026-10-27 完成@2026-10-27

- **最终关闭/责任（2026-10-27）**：唯一Dalton在原main.go/main_test.go完成三入口及局部验证，正式普通/race各36top/271run-pass/232leaf，新增10top/116节点/91风险leaf，0fail/skip；原非作者Erdos [CodeReview R01](../../../../.spec/wip/article-management/review-code-s1.6-r01.md) SHA256 `CB6E05DD1741C645F8E1B754582E65DC705C546346D76728AA29E6113D4BBF58` 两轮PASS、0material finding，独立race窄测10top/116pass/91leaf。Lead已读完整实际diff/原始JSON/报告，核155固定输入、七项Manifest全exit0/无漂移、branch/HEAD/空staged与范围后接收并关闭。作者及Reviewer均实际stopped，两文件独占窗口释放；本卡仅使文章production接线到14/16，Admin2及S1.5四文件/全16条件另守原卡，不外推真实PG/会话/Journal/Host/部署E2E。证据与Windows skip/工具未运行边界见[Lead核证](../../../../.spec/wip/article-management/verification-s1.6.md)。以下START/认领文字保留历史；未提交、部署、清理或代签人的accepted/checkpoint。

- **历史LeadCODE WRITE START（2026-10-27）**：Lead已核本卡/实际两次metadata差异、六项requires、固定main/HEAD/空staged、两root原像及153项直接依赖无漂移，明确释放同root窗口，向唯一Dalton `01a17dcf-f2ed-7dc2-b7f3-6c72cf771448` 发送原两文件CODE WRITE START（Host submission `01a17de6-5eef-7f62-b3cb-657c39e885b4`）。原非作者 [Planning R02](../../../../.spec/wip/article-management/review-tasks-plan-s1.6-r02.md) SHA256 `0463F07DC5F310314D3B9BDE1F88270788172A52917742A39A52C6A24AAE79D2` 确认统计LOW F001 CLOSED、无新增finding、PlanningGate PASS；Lead已读实际报告及delta。正式新top为TestProductionArticleAuthorRoutes；作者负责原卡定向普通/race与局部回归，Lead在源码固定后执行Manifest七项，之后非作者代码Review及Lead核证。现尚无完成交接或CodeReview结论，不标完成；Admin2/身份支线不在代码写权内，禁止提交/部署/清理，原S1.5/16端点条件不变。以下Planner认领时待信号文字为历史，以本次实际发送为准。

- **目标/类型/来源**：开发；同一root闭环接既有 `POST /api/v1/articles`、`GET /api/v1/users/me/articles/{id}`、`PUT /api/v1/articles/{id}`。锚Story1 US-ARTICLE-01.1并消费Story3 US-ARTICLE-03.1的现役Handler，来源为固定r18接口总表及§3.3.2/§3.3.4、A-CREATE/A-MOD验收；不是重开旧组件卡或以测试卡授生产权。
- **文件**（仅修改原两个文件）：
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
- **实际调用链/必要接线**：沿 `newProductionBusinessHandler → buildArticleHandlers/buildArticleQueryHandlers → 现役Handler → newLoginBusinessRouterWithTrace`。现newArticleHandler中的 `NewHandler(service,responseLogger,service)`已有Create/Update；作者详情必须用 `NewAuthorDetailHandler`绑定同一ArticleQueryService，再调GetAuthorArticle，不能误接NewQueryHandler的成员detail。允许在root内扩现役内部数组（当前article11/router17/mutation7/query4）及test seam，不新增公共API、第二service/router/authority；同pool/IDs/clock/logger、安全链与其他路由依赖保持真实共享。
- **验收命令**（workdir：仓库根目录 `.`；新top为待实现验收符号，当前NOT RUN）：
  - `go test ./cmd/han-server -run '^(TestProductionArticleAuthorRoutes|TestArticleBusinessRouter|TestProductionArticleQuery|TestProductionBusinessHandler|TestNotice)' -count=1 -json`
  - `go test -race ./cmd/han-server -run '^(TestProductionArticleAuthorRoutes|TestArticleBusinessRouter|TestProductionArticleQuery|TestProductionBusinessHandler|TestNotice)' -count=1 -json`
  - `go vet ./cmd/han-server`
- **局部验收/回归**：先以真实root复现当前POST/PUT405及作者GET未注册，再实现三入口。新TestProductionArticleAuthorRoutes须逐入口核真实constructor/handler/service依赖身份与注册、成功201/Location/ETag或最小AuthorEditView/ETag、拒绝与失败零下游调用、Trace最外层及适用Origin/Session/CSRF/Principal。main_test现:2985的ordinary PUT/collection POST405旧期望及Allow必须按完整method表更新，不删除回归；collection允许GET,POST，ordinary detail允许GET,PUT,DELETE。同步:7730的GET/PUT/DELETE共存、精确路径/未知路径/错误方法、安全链和现有Identity/DELETE/通知路由回归，不能用spy-only router替代真实constructor接线检查。r15 A-WIRE-RTR02是历史单DELETE范围，不覆盖当前完整method表；本卡不改其历史或旧关闭。新top/关键leaf与现役回归须普通/race实际命中，零用例/skip/删旧测试假绿均不通过。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S1.3`、`article-management-S2.1`、`article-management-S3.1`、`article-management-S4.6`；现均[x]，无S1.5/S2.2反向requires，无DAG循环。
- **Planner认领时记录（历史）**：B9E唯一消费本卡，之后交B10/S1.5；依据Lead本轮明确授权已登记认领/开始@2026-10-27，唯一Backend-Dev Dalton `01a17dcf-f2ed-7dc2-b7f3-6c72cf771448` 仅持 `cmd/han-server/main.go`、`main_test.go` 两文件owner，实际只读G0/prep已completed且Lead已接收。代码START仍待Lead现场核冻结释放、精确scope/未知writer及本次固定metadata后另发实际信号，本Planner不代发；认领不是代码已写或测试已执行。非作者代码Reviewer/执行窗口由Lead续发，未预填Code PASS。S2.2关闭不自动释放共享冻结，不建立S2.2业务依赖；同root生产写入串行。不能改handler/query/迁移、增加Admin2、导出cmd测试API或修改r18/r15。
- **本任务关闭条件**：两文件三入口接线、必要失败证据/普通race回归及适用Manifest七项、独立非作者代码Review/复审与Lead同revision的diff/hash/owner/命中/实际接收齐全。只关闭三入口生产接入，不声称另外Admin2或S1.5全部16项完成，不等待部署E2E整体验收。
- **后续验证关联**：article-management-S1.5、article-management-S1.4；Admin2待固定设计另行落实，不登记未知任务ID。
- **认领时规划交回（历史）**：独立 [PlanningReview R01](../../../../.spec/wip/article-management/review-tasks-plan-s1.6-r01.md) SHA256 `D5796F657B217887896FB744C00D87700C72DD298F2F734D42FDED2B916AA680`：PlanningReviewGate PASS、无HIGH/MEDIUM，仅LOW `AM-S16-PLAN-R01-F001`。Lead已全文读取并采纳本轮修复；原作者已将当前article分项21→22、开发分项36→37，本次统计修复/认领delta待Lead/原Reviewer核验，未代标finding CLOSED。正式验收前缀为 `TestProductionArticleAuthorRoutes`，prep早期 `TestArticleRootS16_` 仅候选，不用旧名造成新top零命中。成功证据沿R01的真实production factory/constructor/共享链、受控真实article Handler合法返回201/ETag/Author4、原业务回归分层，不新增真实PG/E2E门禁；代码START、代码Review及实际测试尚未发生，不造人的签名/checkpoint。

- [x] article-management-S2.1 实现成员可见列表与详情分页并关闭 R01 F001/F002 开始@2026-10-25 完成@2026-10-25

- **目标**：保留已完成的成员可见列表/详情正确性，在唯一 production composition/router 接通两个 GET，并把 repository/Identity 故障分别恢复为设计规定的 503 公共错误，关闭 R01 F001/F002。
- **类型**：开发
- **R01→R02 最终唯一写集**（精确四文件）：
  - `internal/article/query_service.go（修改）`
  - `internal/article/query_test.go（修改）`
  - `cmd/han-server/main.go（修改；顺序复用唯一 production composition/router）`
  - `cmd/han-server/main_test.go（修改）`
- **只读消费者/接缝**：`internal/article/handler.go`、`internal/identity/article_reader.go` 及 S4.6 其余 lifecycle/deletion 文件；现役 `NewQueryHandler`、`ListArticles`、`GetArticle` 已存在，不预设修改。若实现证明必须修改任一只读文件、第五文件、公共 REST/schema/错误码、migration、依赖、architecture/ADR 或建立第二 composition/router，立即停止并回 Lead/Task-Planner。
- **TDD 与实现闭环**：
  - F002 先以列表/详情 repository 故障和 Identity author-state 故障失败用例固定 503 `platform.database_unavailable` / `platform.dependency_unavailable`，响应不得泄露内部 cause；仅本地不变量/编码等真正内部缺陷保留 500。
  - F001 在唯一 production root 使用共享 DB pool、现有 Identity article reader 与 query handler，令列表 GET、详情 GET 和既有 DELETE 共存并经过既有安全链；route-specific 405 `Allow` 必须与实际登记方法一致。
  - `QueryConfig` 必须来自经 design owner/Lead 批准的外部配置来源；AM-Q2 的默认 limit、最大 limit、candidate batch 与 P95 具体值仍为 TBD。测试 fixture `2/2/3` 不得变成生产默认、环境变量名或容量结论。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run 'TestListArticles|TestGetArticle|TestVisibleScan' -count=1`
  - `go test -race ./internal/article -run 'TestListArticles|TestGetArticle|TestVisibleScan' -count=1`
  - `go test ./cmd/han-server -run 'TestArticleBusinessRouter|TestProductionBusinessHandler.*Article|TestLoadStartupConfig_ArticleQuery' -count=1`
  - `go test -race ./cmd/han-server -run 'TestArticleBusinessRouter|TestProductionBusinessHandler.*Article|TestLoadStartupConfig_ArticleQuery' -count=1`
  - `go vet ./internal/article ./cmd/han-server`
  - `go build ./cmd/han-server`
  - `gofmt -l internal/article/query_service.go internal/article/query_test.go cmd/han-server/main.go cmd/han-server/main_test.go`
  - `git diff --check -- internal/article/query_service.go internal/article/query_test.go cmd/han-server/main.go cmd/han-server/main_test.go`
  - 当前状态：Lead 已在固定四文件同 revision 核哈希/branch/HEAD/staged/越界，并取得 `go test ./internal/article ./cmd/han-server -count=1`、对应 race、`go vet ./internal/article ./cmd/han-server`、`go build ./cmd/han-server`、四文件 gofmt/diff-check 全部 PASS。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S8.1`、`article-management-S1.2`、`article-management-S1.3`
- **并行**：本卡四文件返工已关闭并释放独占窗口；S4.6 与 S10.1 继续保持关闭。后续写入须由新的已授权任务重新核依赖、所有权与现场状态。
- **实施/复审责任**：四文件作者为 Hegel；R01 未登记可恢复的原 Reviewer Agent ID，由 Lead 指定已知非作者 Faraday `01a16940-6120-7021-bd06-eb0089956486` 替代完成 R02，并记录替代原因；Lead 对同 revision 完成关闭 Gate。
- **阻塞/顺序 Gate**：已满足。R02 确认 QueryConfig 三个键为必填、无默认并受结构性校验，F001/F002 均 CLOSED；AM-Q2 具体 limit/批次/P95 数值仍为 TBD，只保留为后续容量/性能边界，不回退本卡关闭。
- **独立复审与 Lead Gate**：Faraday R02 固定四文件 SHA并确认 F001/F002 CLOSED、0 新 finding、Code Review Gate PASS；Lead 在同 revision 核哈希/branch/HEAD/staged/越界及 article/server 普通、race、vet、build、四文件 gofmt/diff-check并全部 PASS，关闭条件满足。
- **本任务关闭条件**：原列表/详情正确性回归保持；F001/F002 经非作者复审全部 CLOSED；唯一 production router 可达两个 GET并与 DELETE 共存；repository/Identity 故障分别为设计规定的 503；批准的外部 QueryConfig 接线可追溯；Lead Gate PASS。真实 PostgreSQL、部署后 E2E 与 AM-Q2 容量/P95 仍由 S2.2/S2.3/S1.4 独立闭环。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **关闭证据与限制**：最终 SHA256 为 `internal/article/query_service.go`=`539795E92A4F4276887C9ED3BCD9EEA08C51B1523271CB671D65D2B30B80C777`、`internal/article/query_test.go`=`DF40F63117A0C8F7569A255EABBF9DD95F81CA86FBAE17F88EE93A8F78A0C53E`、`cmd/han-server/main.go`=`841CCA74E3D4CAC1C07DE7217B332B0CA6451C8C3118E106DCBFB3D79F4E680E`、`cmd/han-server/main_test.go`=`CFE64848B572701E81608280B61C3392C19479407D0DA8CB02CB6C3A53DC31E4`。未连接真实 PostgreSQL，未部署或执行 E2E；AM-Q2 数值、容量/P95 仍 TBD，不声明 production readiness。
- **设计/测试/评审来源**：article Story 2；design §2.4.1/§3.3.3/公共错误表；test A-READ-01～06；R01 `.spec/wip/article-management/review-code-s2.1-r01.md`（SHA256 `36BB65973CDC5998567B57B8C10AED1F6B65E114DEB5560414B19179540EF129`）；R02 `.spec/wip/article-management/review-code-s2.1-r02.md`（SHA256 `01373C88B3E706A0EB0573BEEAFB67668E8C6537061EB6BE8F6FCB86D4CA8D4F`）

- [x] article-management-S2.2 验证文章 PostgreSQL 状态、分页与 ETag 并发 [P] 开始@2026-10-27 完成@2026-10-27

- **最终关闭/责任（2026-10-27）**：Ohm固定R02普通/race各3顶层、18风险组、77 run/pass节点、60 leaf，0 fail/skip；三组相同单点mutant在旧断言GREEN、新断言实际RED。原非作者Erdos [R02](../../../../.spec/wip/article-management/review-code-s2.2-r02.md) SHA256 1D5A570A84986383D235FD3BF65F7394C079F4C13DFAD4777236632D6BE8FEC1 确认AM-S22-R01-F001/F002均CLOSED、0新增问题、CodeReviewGate PASS。Lead已读实际diff/完整mutant/原始JSON，核最终157输入、Manifest R02七项顺序全PASS、124个最终schema与关键PG回读、899基线归属/固定main和HEAD后关闭；作者与Reviewer均实际stopped，三文件窗口释放。现有生产及notice文件未变；并行身份文档和Tasks身份注记按用户明确的另一会话成果保留。未暂存、提交、部署、清理或签人的accepted/checkpoint。完整证据/历史/未验边界见[Lead记录](../../../../.spec/wip/article-management/verification-s2.2.md)，本卡不证明生产router或部署E2E。

- **R01历史独立评审与整改（2026-10-27）**：Ohm固定R01普通/race各3顶层、18风险组、76 run/pass节点、60 leaf，0 fail/skip；Lead实际解析JSON、157输入及Manifest七项/真实PG回读已核。独立Erdos [R01](../../../../.spec/wip/article-management/review-code-s2.2-r01.md) SHA256 FE353F0595C661AE6FB45405F12F2F5E8390C99AF034FA2D71AAD0AC1D2D747D 为FAIL，MEDIUM AM-S22-R01-F001审计动作/类别未按操作与version精确绑定、F002缺跨created_at继续分页数据，当时均OPEN，现R02均CLOSED。当前未观察产品错误；Lead已交原作者仅在原三文件整改及真实负控，之后由同Reviewer复审。新修订全套/复审未齐前保持进行中，不将旧GREEN外推；完整证据及未验边界见[Lead记录](../../../../.spec/wip/article-management/verification-s2.2.md)。

- **历史 Lead 执行 START（2026-10-27）**：已完整读取原 Planner 固定卡/Plan 与 Ohm 真实 Prep，按899文件基线核仅两份规划 metadata 变化、无缺失/越界、三候选文件仍不存在；两角色均实际返回停写。沿用户采用建议继续的授权，正式交 Ohm 唯一新增本卡三文件并执行本地 PostgreSQL 专项。登记当时测试尚未运行，后续原始证据先进入下列安全日志根；范围、依赖、全部保留 schema、生产及 notice 只读、独立评审和关闭条件均不变。以下准备认领/待信号叙述保留为当时交接依据，以本 START 与后续真实成果为准，不预填测试 PASS。

- **目标/类型**：专项集成测试；只验证现役查询、作者修改和治理接口在真实 PostgreSQL 的状态、隐藏分页、行锁/version 竞争与事务回滚，不实现新产品行为。
- **R 对账（Lead 本轮采用建议授权）**：Story 2 的查询/投影、Story 3 的作者修改/强 ETag、Story 8 的 mark/restore 接缝均已有实现，初步静态判为已建成、在本卡接受独立 PG 专项验证；原 S1.3/S2.1/S3.1/S8.1 的关闭状态不变，不重登记实现或改成待验。新专项三文件均不存在、现包只有 S7.2 三 notice 测试，因此三个验证单元均为待做；当前无需改/废弃的确证，不由旧 PASS 证明覆盖。
- **文件/风险 gap-map**（共且仅三新增文件，同一执行 owner，不按文件新建 task）：

  | 原验证单元/标签 | 精确文件与现役可调用证据 | 最低充分的断言范围 |
  | --- | --- | --- |
  | 状态与回滚：待做 | `test/integration/article/postgres_state_test.go（新增）`；`policy.go:75,117,165,190`、`query_service.go:134,183`、`command_service.go:145,266,626`；已有 query/update/moderation 单测多用 stub，不能替真实 PG | 正交合法状态/不存在/pending/已删读边界、匿名与 active/suspended 作者、identity 缺失/故障 fail closed；实际 mark/restore 后约束、内容修改只变内容/version、拒绝输入前后全快照不变；真实 UPDATE 与 AuditStore 同 tx 的故障回滚 |
  | 隐藏分页：待做 | `test/integration/article/pagination_test.go（新增）`；`query_service.go:222,316,491` 为真实 keyset SQL、持续扫描与最后返回项 cursor；`identity/article_reader.go:39` 读当前作者状态 | 同时间戳 ID 平局、多批连续隐藏、limit+1 不消费、空/末页、静态不漏不重；跨请求新增/编辑/作者状态变化符合当前事实语义，故障不输出半页 |
  | ETag/多 writer：待做 | `test/integration/article/etag_concurrency_test.go（新增）`；`command_service.go:202,300,680,750,774` 的行锁/version 条件更新及 `policy.go:275,289` 强 ETag | 独立连接同版本内容/内容、内容/mark、内容/restore、治理/治理竞争恰一成功；合法竞争失败方 precondition、version 单调、重读 ETag 后合法重试；审计失败回滚并释放行锁，不留下幽灵治理事实 |

- **精确范围/映射**：消费固定 test-plan §4.1 A-READ-01～06、§4.2 A-ETAG-01～04/A-MOD-01～08、§4.3 A-MODR-03～06 中的真实 PG 状态/分页/锁/数据回滚责任及 §6/§7 映射；只做必要的真实 handler/DTO 到数据库对账，不把本卡标为这些跨层 Case 的全部覆盖。A-READ-04 的完整 cursor parser/fuzz、A-ETAG-01 的纯 header 矩阵和 A-MOD 的完整 HTTP 字段/decoder 契约由现有 S1.2/S3.1 或原 S1.5 承接；本卡对无效内容及 N/N+1/非法 UTF-8 只补必要入口调用与 PG 全快照零副作用，不复制全套 HTTP 契约。A-MODR-06 的完整 Journal/finalizer 删除故障及真卷/恢复属于原删除/韧性专项；本卡只核适用 PG 竞争/回滚和 pending 不可见，不借此宣称其 Journal 全覆盖，不倒挂 Unix Host、部署 E2E 或容量 Gate。
- **三个顶层与必需 leaf**（均待实现，不以名字代替断言；子表可展开更多用例，下面每个风险必须在 JSON 结果与证据中定位）：
  - `TestArticleState`：`VisibilityMatrix`、`ThreeProjections`、`CurrentAuthorState`、`AuthorStateFailureClosed`、`ContentFieldIsolation`、`InvalidContentRollback`、`WireBoundaryPersistence`、`ModerationStateAndAuditRollback`。按 Ohm 已完成只读 Prep 映射真实 PG 状态/投影、当前作者状态与故障、内容/治理字段隔离、N/N+1/非法 UTF-8 前后全快照和真实更新/审计回滚；Admin 第三视图使用真实 SQL 结果及现役 AdminProjection，不伪造管理员读取端点。作者缺失/故障在真实 ArticleIdentityReader 返回后受控注入，不删除 FK 所指用户或破坏约束。审计按实际动作对账 target/actor/trace/version，不为纯内容替换编造新审计；spy 只作补充，不替代 PG 快照。
  - `TestArticlePagination`：`HiddenAcrossBatches`、`SameTimestampNoMissingDuplicates`、`LookaheadCursorBoundary`、`EmptyAndLastPage`、`InsertEditAndAuthorStateBetweenPages`。由独立 SQL fixture 预期可见序列校验真实 query/identity API，证明连续隐藏跨批、同时间戳平局、limit+1 不消费及最后返回项 cursor、空/末页和跨请求新增/编辑/作者状态变化；当前事实变化不承诺跨请求快照，故障不得返回半页，不能拿被测算法重算 oracle。
  - `TestArticleETag`：`SameVersionTwoWriters`、`EditVersusModeration`、`MarkRestoreCompetition`、`RefreshAndRetry`、`HeaderVersionConsistency`。用真实独立连接/事务及 channel/barrier 确认行锁窗口，不用 sleep 猜时序；合法同版本 writer 竞争恰一成功、另一 precondition，重读 ETag 后合法重试，核最终 row/version/Header 与适用审计一致及回滚后锁释放。mark/restore 组合依现役 deletion→version→state 判断顺序区分 stale/invalid_state；不能把本就不合法的动作强行当作两个合法 writer，或一概要求失败均为412。
- **现役 API/只读边界**：直接调用 `NewPostgresArticleQueryRepository`、`NewArticleQueryService`、`identity.NewArticleIdentityReader`、`NewCommandService`/`Repository{}` 的 `ReplaceContent`/`Moderate`、`VisibilityPolicy`/`StrongETag` 及必要 handler，使用真实 `audit.Store`。生产 read set 为 `internal/article/{model,repository,query_service,command_service,policy,handler}.go`、必要 identity reader/AuditStore 与已有测试；migrations 0001～0009、S7.1 九生产文件和 S7.2 三 notice 测试全部只读保护。允许事务/identity 端口故障 wrapper，但不得替换目标 SQL/行锁/实际审计或用 fake 证明 PG 原子性；commit outcome unknown 不凭返回 error 假定回滚。
- **三文件充分性/环境**：现同 package `articleintegration` 的 `noticeHarness`/`newNoticeHarness`（notice_dispatcher_test.go:34/87/110）已是 retained、随机预提交 schema、完整 0001～0009 migration 基础，按真实 Prep 只读复用 `newNoticeHarness(t,true)`，不写四项 ProcessingGate 登记；既有 `han_s72_` 随机 schema 前缀保留，以本次日志标记 S2.2，不改 notice 三文件。本卡最小 fixture/adapter/helper 全放首文件、使用 S2.2 专属命名，不改 notice 文件、不加第四文件/依赖/DB。若其耦合不适用，在首文件内承最小 retained 适配，不改变真实产品契约。fixture 包含合法 CHECK/FK 与合成用户/文章/审计，pending 前先创建对应 DeletionOperation；不禁用约束/trigger，不靠破坏 author FK 制造缺失。沿 Lead 已验证的 User-scope `HAN_TEST_POSTGRES_DSN` 仅向执行进程安全注入，不打印值；目标既有 `127.0.0.1:5433 PostgreSQL 18.6`，本 Planner 未连接。所有新 schema/fixture 与失败证据保留，不调用旧 identity 自动 DROP helper；tx/rows/pool/context/goroutine 实际 rollback/close/join，不自动 DROP/清理。
- **验收命令**（workdir：仓库根目录 `.`；已实际执行，精确命令/timeout/结果见最终关闭证据）：
  - `go test -tags=integration ./test/integration/article -list '^(TestArticleState|TestArticlePagination|TestArticleETag)$' -count=1`
  - `go test -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$' -count=1 -json`
  - `go test -race -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$' -count=1 -json`
  - `go vet -tags=integration ./test/integration/article`
- **命中/成功 Gate**：先静态核三文件 `//go:build integration`、package 与精确顶层/风险 leaf，再核 -list 恰有上述三顶层；普通和 race 的 JSON run/pass 必须分别命中三个顶层与所有必需风险 leaf，并把调用结果/错误、实际 PG 数据流/锁与最终 row/适用审计对账为三层证据。零测试、缺关键 leaf、skip、`[no tests to run]` 或仅 helper 通过均不能 PASS；旧 notice 顶层或 identity integration 不能替代。专用结果稳定后由 Lead 按 Manifest 七项顺序单独执行 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`；普通全包不覆盖 integration tag，不能替代专用命中和独立 Review。
- **历史边界**：原卡身份 integration/旧 Manifest PASS 仅为身份历史，不证明新 Article 状态/分页/ETag 专项；现有 S7.2 retained harness 及其关闭证据仅证明可复用基础，不复用其 PASS 关闭本卡，旧证据不清理。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.3`、`article-management-S2.1`、`article-management-S3.1`、`article-management-S8.1`；原五项均已关闭，不改 requires/DAG，不增加 S7.2 业务前置。
- **历史批次/准备认领/接收**：唯一归属原 B10，`[-]`/开始@2026-10-27 表示本卡准备进行中；实际 owner 为原 TestEngineer Ohm `01a17c83-60f4-7803-b0cc-9b19c0b81fa3`。Lead 已回传其本卡只读 Prep 为 Host completed：0写/0测试/0PG连接、无已确认 API hard gap；本次据此记录真实承接，不是 Planner 新建/委派或臆测回执。三顶层/leaf 与 API/fixture 范围已按真实 Prep 对账，仍待 Lead 核本次固定卡/hash/scope 后发测试代码/PG 执行 START 信号；本轮未实现或运行测试。Ohm 返回成果及原始证据，Lead 现场核证、接收并关闭；非作者 Reviewer 待 Lead 指定，独立 Review 未执行，不由作者自审代替。
- **并行/停止与失败交回**：[P] 仅对文件、契约和环境均隔离的无依赖任务；三文件由同一 owner 串行/统一管理，既有 S7 无 writer 且只读，其他 B10 卡保持原条目。新生产缺陷须保留失败 schema、触发/预期/实际/DB与审计证据交 Lead 原作者，当前测试可返回真实 FAIL但不得放宽断言、绕过 SQL 或越界修生产；设计/API 真缺口只暂停受影响子集并给精确证据。第四文件/新依赖/改 notice/迁移、未知 writer、错误 DB 目标、缺符号/skip 或资源不能退出时停相关执行，禁止擅自新 task、重开前置、清理、Git 写或发布。
- **本任务关闭条件**：三文件独立 PG 场景与普通/race 三顶层/风险 leaf 全部实际命中且零 fail/skip、数据/锁/适用审计符合、资源退出和 retained 清单可核；独立非作者 Review/必要复审、Lead 同 revision 的 diff/hash/所有权/七项 Manifest 与实际接收齐全。准备认领、旧 PASS、build 或代码存在不能关闭本卡；性能与真实部署 E2E 继续原后续任务，不倒挂。
- **历史Prep/安全证据载体**：Lead 指定新日志根 `C:\Users\J00010385\AppData\Local\Temp\han-s22-bca6db0ff5314f56ab830d68b42e881f`，已存899file起始hash；本轮只允许两 metadata 变化，其余保护，当前无代码 writer。后续实际执行记录 retained schema、风险映射及原始普通/race证据到该安全载体，不复制 DSN/secret/生产数据。本 Planner 未创建或改写该目录、未运行测试；接口真实调用为 query repository/service/identity reader、CommandService.ReplaceContent/Moderate+Repository+同 tx identity reader+audit.Store，必要 HTTP 边界直调现役 Handler，不新建 router/API。
- **后续验证关联**：article-management-S2.3、article-management-S1.4
- **设计/测试来源**：Story 2/3/8；固定 article design r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`，§2.4.1/2、§3.3.3/4/9、§3.4.1 及相关事务合同；固定 test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`，§4.1～4.3、§6/§7 仅本卡适用子集。不外推通知增量的 Design/TestPlan Review 范围为本专项或本卡 Code Review PASS。

- [x] article-management-S2.3 建立 article 容量与性能基线 [P] 完成@2026-11-02

- **目标**：建立 article 容量与性能基线。
- **类型**：专项性能测试
- **文件**：
  - `test/performance/article/benchmark_test.go（新增）`
  - `docs/benchmarks/article.md（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -run "^$" -bench . -benchmem ./test/performance/article`
  - 当前状态：`go test -run "^$" -bench . -benchmem ./test/performance/article`、`gofmt -l test/performance/article`、`go vet ./test/performance/article/...`、`go build ./...` 已由 Lead、独立 Reviewer 与修复作者分别实跑 exit 0；`go test ./...`、`-race` 与 integration benchmark 未执行。独立评审 r01 为 CHANGES REQUIRED、r02 为 PASS（HIGH 0 / MEDIUM 0 / LOW 1 非阻断），F001～F003 全部 CLOSED；S2.3 已关闭，关闭证据见下方。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S2.2`、`article-management-S4.3`、`article-management-S4.4`、`article-management-S7.2`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **阻塞注记**：AM-Q2、容量目标、RPO/RTO、retention 未确认；AM-Q1 已关闭，不再构成本卡决策阻塞。
- **本任务关闭条件**：产出内容、分页隐藏密度、dispatcher、finalizer 与 replay 基线；不编造阈值。AM-Q2、容量和 retention 未确认时保留对应阻塞注记；独立 Review 通过。
- **后续验证关联**：article-management-S1.4
- **设计/测试来源**：article test A-PERF-01～05
- **独立评审与 Lead Gate**：非作者独立 Reviewer Franklin（Agent ID `01a199c4-c770-7ba2-8930-b8b4c7ba12e3`，`add-reviewer`）r01 为 `CHANGES REQUIRED`（HIGH 1 / MEDIUM 1 / LOW 1；`.spec/wip/article-management/evidence-s23/review-code-r01/report.md`，16,662 B，SHA256 `C53B8412764E63306E555E75E03F4F6ADAE19E668D0729AC1689B5C955A8972A`），指出 dispatcher 基线缺失、卡面混入无关 identity/Manifest 状态与固定时钟说明错误；修复后 r02 为 `PASS`（HIGH 0 / MEDIUM 0 / LOW 1 非阻断；`.spec/wip/article-management/evidence-s23/review-code-r02/report.md`，15,225 B，SHA256 `74B836B736BE73A7362DFE6A6E73A0170DBEAF97F702F43D4F0397285D131945`），F001～F003 全部 CLOSED。Lead 现场复核三文件哈希并实跑 gofmt/vet/build/benchmark，`Code Review Gate: PASS`。
- **关闭证据与限制**：最终 SHA256 为 `test/performance/article/benchmark_test.go`=`8895BD90F131AC803D60D1FC9EC75B5E5B373BA5388B308E67FB794246BA2E58`（22,314 B）、`docs/benchmarks/article.md`=`3E2482DAC54F4DF9F2398F81CEC24036990BD658C80FB7B21787E0CBE53315BB`（12,921 B）、evidence 报告=`E7755A3B30E2A302BC33A5E60E0BB72D9A8CEE838EDE9FBE7900CF6CF7D0F782`（16,326 B）。关闭只覆盖进程内基线（创建/修改、分页隐藏密度、dispatcher 空轮、worker、completed finalizer、replay）；真实 SMTP 投递、通知 lease 与重试积压、真实 PostgreSQL、HTTP 协议路径、Unix Journal/fsync、真实物理删除/RTO 与真实 E2E 均未验证，A-PERF-01/04 保持 partial、A-PERF-02/03/05 保持 blocked，AM-Q2、容量、P95/SLO、retention、RPO/RTO 未确认；`go test ./...` 仅作者自报、Lead 与 Reviewer 未复跑，`-race` 与 integration benchmark 未执行。`S23-R02-F001`（evidence 第 206 行把固定时钟位置写成 `benchmark_test.go:37`，实际为 `:39`）作为已知非阻断文档漂移保留，未改证据以保持与评审哈希一致。

- [x] article-management-S2.4 补测 article 真实 PostgreSQL 容量与性能 开始@2026-11-04 完成@2026-11-05
- **2026-11-05 R11 新 attempt 预检与结构性阻断（S2.4 保持 `[-]`；Lead 直接执行，无 Backend-Dev / Reviewer / Task-Planner Role 产出）**：按用户本轮「按照ADD流程继续推进」与 ADR-0028 建立新 attempt 窗口（`author/r11-attempt-20261105/window.json`，started_at `2026-11-05T12:00:03.975227Z`，stop_new 8700s／hard_stop 9000s，large/canonical NOT RUN），未复用 R10 失效许可。G0 实核：5 个冻结源文件哈希与 R10 `source-freeze-r10/manifest.json` 逐一对齐、`r10.test.exe` SHA256 `318EBF7A…8CF52EF0` 一致、PG 18.6 于 127.0.0.1:5433 可达、D 盘余量 132.78 GB。calibration-r01 两次真实 exit1：①`pg_perf_harness_test.go:408` 证据根须位于 `continuation-20261104/author` 之下（已改正）；②`pg_perf_harness_test.go:6581` `explicit adopted execution policy required`。根因（只读实读，非推断）：`pgPerfPolicyPaths()`（`:1502-1512`）把 adoption 文件硬编码为 `lead/followup-clock-contract-adoption-r04.md`，而该文件明文「采用不产生Tnew，不开放新schema、PG写入或完整性能窗口」；故 R10 冻结源上不存在可合法签发的新窗口——复用该 adoption 即复用已声明不开放窗口的许可（违反 R10 最终接收「不得复用本轮失效许可」），指向新 adoption 则须修改冻结 harness（新源码修订＋非作者独立评审＋新冻结）。本轮未进入 seed/测量工作负载：两次失败均在证据目录/策略解析守卫，耗时 3.1s/0.098s，证据根 `summary.md` 无 point/schema 行、`seed-manifest.json` 为 `null`；不宣称全库零遗留、不核销任何 finding。S2.4 保持 `[-]`，86=74/3/9、DAG/requires/owner 不变，R09/R10 旧证据与 NRU 不改写。完整记录见 `author/r11-attempt-20261105/r11-structural-block.md`。（本条为范围决策前的时点记录；同日后续已按 ADR-0033 关闭本卡，见上方 2026-11-05 范围决策关闭变更记录。）
- **当前续接指针（2026-11-04，R10源码接收；执行证据仍待补）**：据 [growth独评](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/code-review-growth-r09/verdict.md)、[leak独评](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/code-review-leak-r09/verdict.md)及 [R09语义接收](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/semantics-r09-reception.md)，growth/leak局部独立PASS、F005关闭；R09 N9真实8PASS/1 seed FAIL（10k在seed120超时，未进drain），F015/F016关闭，累计9CLOSED/9PENDING。F011/F012仍缺双方返回及同ID行/op/audit保存物证SEM-R09-EV001，不以动作执行或源码补接口核销。R10仅H最小修订：正常10k seed240/root300、defaultSmall120不变，正常E02/D02补双方结果及行/op/audit物证接口；作者6项最终offline/编译全0、Main同源7checks全0，[最终Source独评](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/code-review-semantics-r10/verdict.md) PASS/no finding，Reviewer已STOP；[Lead实际采用](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/r10-source-acceptance.json)仅源码+离线/构建，SOURCE/LOCAL已STOP、PG=false，不关闭N或EV001、不构成整卡/完整性能PASS。[R10整组NRU](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/r10-normal-revalidation-not-run.md)：全9最迟12:15:37.932710Z启动门已失，本attempt永久NOT RUN；完整calibration/small/medium/full/large/canonical及旧batch永久NRU保留，R09混合结果不改写。唯一Tnew18:22:37.932710+08、145min停新20:47:37.932710/150min硬停20:52:37.932710，原caps/资源origin不变，不重开窗口。S2.4 active、86=74completed/3active/9todo、DAG/requires/owner不变；本次核证见 `continuation-20261104/planner/probe-followup-checkpoint-r10-final/`。

- **目标**：在真实 PostgreSQL 18.6 上补测 article 创建/修改、会员列表与作者详情、删除 T1、审计事务、通知 claim/lease/backlog 的容量与性能观测，并同时保持正确性断言；只记录观测，不判定 SLO、目标容量或 production readiness 达标。
- **类型**：专项性能测试（integration-tagged 真实 PG 容量/性能补测）
- **本轮认领与接收（2026-11-04）**：正式 Task-Planner 当前 Thread ID `01a1a484-80f7-71d2-a8d5-6c93a56482e4`（运行时 `CODEX_THREAD_ID`）；Backend-Dev Feynman `01a1a485-c974-70a1-abc4-5737ed05526d`，真实 ID 与启动时仅 G0 只读状态由 Lead 本轮提供。Lead 随后现场核 active 与统计，并明确已下发 SOURCE / LOCAL / PG START：仅原四个 test 文件、原 docs 校正片段、新 author 证据及 loopback:5433 / `han_perf_s24` 新 schema；启动授权不等于实施/验证已跑或通过。复用 B11 原卡，不新增任务、不改变三项 requires 或 DAG。旧代码作者、旧运行执行者与当时认领来源 unknown，不代旧作者签字。用户 2026-11-04 授权采用建议值并按 ADD 继续，不替代环境、安全、评审或整卡 PASS 证据。
- **R 对账与交接**：四个 `pg_perf_*_test.go` 与两个 r04 证据目录均已存在；现状为“已有实现、需改并待验”，不得按原 todo 推定未实施。`.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/planner/handoff.md` 保留恢复时点的四标签、旧 small 8 点/10 schema、偏差及启动事实，原字节不回写；当前计数/预算/timeout/本轮 large 分支按下述已受审 addendum/map 及执行治理r02显式覆盖对账，新增接手证据见同目录 `planner/addendum-reconcile-r01/`。旧局部 GREEN、原 r04 Review PASS 与本次规划增量 PASS 均不等于实现、执行或本卡 PASS。
- **四标签现状（仅本卡）**：已建成＝四文件 harness/场景入口及旧 small 局部观测；需改＝采样/并发/预算与证据保留偏差（整卡主标签）；待做＝缺失的校准、small/medium 当前修订完整执行、失败重现/回归、代码/执行证据独立评审及 Lead Gate；large/canonical 本轮明确 NOT RUN（优先 small/medium 准确闭环与非作者独立 Review，避免额外 60min），不是废弃或验证通过。废弃＝未发现已获准废弃项。上述为卡内 gap-map，不创建第二套 task；旧执行日志/退出码与旧作者责任缺源仍 unknown，Backend 本轮实际进展以其新证据为准，不由本次规划登记推定。
- **执行治理 r02（2026-11-04，Lead 已采用）**：按 [execution addendum r02](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/test-plan-execution-addendum-r02.md)（15,364 B / SHA256 `3667D1356B031FD616AEB0A58A841129E58C2ADF69B4F31C583DA7AACAEC5BC0`）、Halley `01a1a495-be38-7201-a1d7-5c251cf16900` 的 [独立report](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/test-plan-execution-review-r02/report.md) / [verdict](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/test-plan-execution-review-r02/verdict.md)（Execution Plan Delta Review Gate PASS / 0 finding）及 [Lead adoption](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/execution-policy-r02-adoption.md)，本轮明确将共同90min/5400s改为145min/8700s；原起点 `2026-11-04T03:29:04.586115Z`（本地11:29:04.586115，UTC+08:00）不动，失败/等待/返工/定向检查/复审/重验全部累计，run-id/进程/新freeze不重置时间或磁盘基线。2026-11-04本地13:54:04.586115起不得新开point/batch/业务op，13:59:04.586115为150min硬停；最后5min仅既在途取消/回滚/join/核对/落盘，不借外层timeout或历史large60min续时。完整重验条件最迟12:32:04.586115齐备：新freeze定向回归、18 finding适用静态复审、安全/磁盘guard根因及采用门均已满足，保守72min重验+10min收尾/证据复核共82min须装入余量并逐阶段重估；条件/时间/资源不足则余项真实NOT RUN/PENDING，不自动开跑，不复活已过旧90min额度。旧5400s窗口、原r04/addendum/map/旧Review原字节保留；本政策显式覆盖其相应窗口/采样收尾字段，执行者须以追加有效政策关联原window与新freeze/run-id，采用不追溯洗绿旧失败。
- **当前真实交付（2026-11-04，仅已见证据）**：[source-freeze-r01](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/source-freeze-r01.md) 对应 [CodeReview R01](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/code-review-r01/verdict.md) 为CHANGES REQUIRED（5 HIGH / 13 MEDIUM / `S24-CODE-R01-F001`～F018共18 finding），执行证据Review PENDING；[校准r02 result](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/pg-calibration-r02/result.json) exit0/162.75s仅旧源局部证据，[small r01 result](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/pg-small-r01/result.json) 自然exit1/567.282s，首错R04-10rows累计磁盘探测timeout（性能操作error=0不能抵消资源门FAIL），后点NOT RUN、medium未启。7schema（校准3/small4）保留，owned child已结束/相关backend=0仅为 [author stop-r01](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/stop-r01.md) 的停止时点记录，不是本角色现场DB复查或现在可重启证明。Lead本轮提供Backend正在原四文件内返工并做定向RED/GREEN，尚不声称18已闭或已全量重跑；新freeze须完整重新校准/small/medium，不拼旧子项PASS。
- **2026-11-04 R03最终证据接收登记（仅本轮有界成果；S2.4仍active）**：已读原Reviewer Halley（`01a1a495-be38-7201-a1d7-5c251cf16900`）最终同步的 [report](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/code-review-r03-evidence-r01/report.md) / [verdict](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/code-review-r03-evidence-r01/verdict.md)，并消费 [Lead最终接收](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/final-reception-r03.md) 与 [final scope](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/final-scope-r03.json)。原18项静态修复判断18/18保留；finding累计6 CLOSED（S24-CODE-R01-F001/F003/F004/F008/F010/F018）、12 PARTIAL/PENDING，R03-E001独立CLOSED；最终CodeReview/执行Review仍PARTIAL/PENDING，非PASS。measurement是混合失败：完整warmup、measurement两raw确有1成功+1非context错误与NONCONFORMING，另伴sizeSQL5000ms timeout；F001仅按实际测量错误拒绝分支维持CLOSED，不能把exit1或混合失败说成干净负例PASS。已读 [positive index](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/r03-evidence/positive-evidence-index.json)：safety exit0/27.188s，取消release/join及W02双正文存储诊断成功；semantics exit1/13.235s，fresh-T1首DDL前acquire0/SQL5003ms阻断，后8分支NOT RUN，非各业务均已运行失败。leak负例同被前置SQL5000ms挡住、未到注入，S24-CODE-R01-F005不核销。Lead实际 [7项项目checks](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/project-checks-r03/summary.json)全exit0、每条前后五hash一致/无漂移，仅项目检查，不能替integration性能/执行门。作者 [positive-stop](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/r03-evidence/positive-stop.md) 最终LOCAL/PG STOP为13:35:47.482879+08；[保存的final probe](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/r03-evidence/final-positive-pg-probe.json)仅其时点覆盖6个r03 run、backend0、4schema保留，1992988351−原baseline1406932671＝累计586055680 bytes；不是Planner新查库或全库无连接。Lead本轮补充13:42:20.7580019本机CIM按python/go/test及本轮runner/定向path过滤命中0；其268范围实核仅原四test+docs五文件变化，无新增/删除、freeze后无漂移；Reviewer最终13:43:41 STOP由Lead本轮提供。原11:29:04.586115起点、13:54:04.586115停新op/13:59:04.586115硬停及全部磁盘基线不变；按 [full revalidation NRU决定](../../../../.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/full-revalidation-not-run-r02.md)，完整calibration/small/medium/large/canonical本轮均NRU，局部证据不恢复资格，不开新窗、不放宽门凑PASS。下一步仅先取得probe执行环境/超时事实，不预判锁/IO/CPU根因，不授权新Go/PG或改源。C18-COV-F005 sidecar r03的50/50独立闭环维持，不改S1.5/S8.2/Case18；其余12项缺证仍按独立报告逐项接续。本条仅Planner登记，Reviewer/Lead成果归其原记录，非真人签字，不改状态/depends/owner/批次结构/统计，不宣称性能/E2E或整卡PASS。
- **文件**：
  - `test/integration/article/pg_perf_harness_test.go（已存在，原文件内修正：newPgPerfHarness、pgPerfPoolConfig、pgPerfMigrate、pgPerfSchemaName、pgPerfSeed、pgPerfRunPoint）`
  - `test/integration/article/pg_perf_write_test.go（已存在，原文件内修正：W01～W04、E01～E03）`
  - `test/integration/article/pg_perf_read_test.go（已存在，原文件内修正：R01～R07、X01～X03）`
  - `test/integration/article/pg_perf_notice_delete_test.go（已存在，原文件内修正：D01～D04、N01～N04、X04～X09）`
  - `docs/benchmarks/article.md（仅修改原 :189/:190/:192 的 AM-Q2 过期片段；如实区分已见 small 局部观测与尚缺验证，不写全量补测已通过）`
  - `.spec/wip/article-management/evidence-pg-perf-r04/（已存在；旧证据与 evidence-pg-perf-r04-devrun/ 只读保留；本轮仅新增 planner/author 分离的 continuation 证据，Backend 使用新 runid 与新输出目录，禁止覆盖旧文件）`
  - 新增文件固定为上述 4 个且不得改名；均须 `//go:build integration`、package `articleintegration`，不得重新定义既有符号 `noticeHarness`、`noticePoolConfig`、`newNoticeHarness`、`newAdminPG`、`newAdminPGPool`、`adminPGMigrate`；不得修改既有 `admin_query_test.go`、`postgres_state_test.go`、`pagination_test.go`、`etag_concurrency_test.go`、`notice_dispatcher_test.go`、`notice_smtp_test.go`、`notice_crash_test.go`、`audit_insert_rollback_test.go`。
  - 明确禁止：改生产代码、改 `migrations/0001`～`0011`、新增 Go 依赖、改既有测试断言、对任何 schema 执行 `DROP`/`TRUNCATE`/`DELETE` 或清理/删除 schema。
- **验收命令**（workdir：仓库根目录 `.`；本轮登记阶段均未执行；`TestPgPerfSmall`/`TestPgPerfMedium`/`TestPgPerfLarge` 三个顶层名严格互斥，禁用 `-run '^TestPgPerf'` 串跑三档，外层 `-timeout` 必须大于档位窗口预算）：
  - `go test -tags=integration ./test/integration/article -list '^TestPgPerf(Small|Medium|Large)$' -count=1`
  - `go test -tags=integration -timeout=45m ./test/integration/article -run '^TestPgPerfSmall$' -count=1 -json`
  - `go test -tags=integration -timeout=80m ./test/integration/article -run '^TestPgPerfMedium$' -count=1 -json`
  - `go test -tags=integration -timeout=70m ./test/integration/article -run '^TestPgPerfLarge$' -count=1 -json`（仅保留接口，本轮 NOT RUN）
  - `go test -tags=integration -timeout=90m ./test/integration/article -run '^$' -bench '^BenchmarkPgPerf' -benchtime=30s -count=3 -benchmem`
  - `go vet -tags=integration ./test/integration/article`
- **环境与数据档位**：仅引用环境变量名 `HAN_TEST_POSTGRES_DSN`，空值 Fatal 不 skip，不读取/打印其值；仅既有库 `han_perf_s24` 的新唯一 schema，目标 `127.0.0.1:5433`、PostgreSQL 18.6（`server_version_num=180006`）；schema 格式 `han_pgperf_r04_<s|m|l><case≤6>_<16hex>`，前缀 `han_pgperf_r04_*` 保留不 DROP，建前断言 `length(schema)<=63`；迁移仅 `0001`～`0009`，不执行/不修改 `0010/0011`，不建库。档位为 small 1,000 篇/100 用户、medium 10,000 篇/1,000 用户、large 100,000 篇/10,000 用户（测试假设，非目标容量承诺）；seed 预算 <120s/<600s/<1800s，场景 context 300s/780s/1980s。A/B/C 共 41 逻辑场景、42 展开 schema 场景单元（small 9 逻辑/10 单元、medium 32/32，X01 两 fixture 各一）；校准 K 按全部真实调用实物另记，归 small 资源账但不重复计实物。无重试且每场景单元恰建一 schema 时本轮条件式为 42+K，未来完整 large 另加 D8+canonical1 为 51+K；均不作固定实物总数，失败/重试/半成品/自适应调用全部进入实际 inventory。原 43 仅为历史假设。large D8 与 canonical 本轮 NOT RUN，原因是优先 small/medium 准确闭环和独立 Review，避免额外 60min。磁盘本轮采用 small/medium/large/total＝1/5.5/8.5/15 GiB（GiB=2^30 bytes），精确上限分别 1073741824/5905580032/9126805504/16106127360 bytes；校准计 small，按分档及本轮共同基线跨进程/重试累计 DB 增量，核 PGDATA 所在卷真实余量与预留，不因换 runid 重置。small+medium 共同 ≤145min/8700s（显式接续原90min；原分项仅历史，20个TIMED点预算由40min增至40min20s，不另开新窗），含校准/环境/seed/迁移/ANALYZE/断言/汇总/缓冲；large 独立 ≤60min仅保留接口，本轮不启且不叠加到145min；总 150min，145min 停新点/新批/新业务op、150min 取消在途工作并落盘。medium 外层 80m 仅为收尾余量，不延长实际窗口；触限/余量不足停止，不能清理腾空间或缩覆盖。
- **指标与正确性门**：记录 P50/P95/P99（nearest-rank）、QPS/吞吐、错误率、`EXPLAIN (ANALYZE, BUFFERS)` 实际扫描行、索引命中、buffers、连接池/并发、事务提交/回滚、锁等待。A/B/C 保留 `TIMED-3REP` 20 点（每次 warm-up=max(10s,min(30s,20%×测量窗))、每次测量≥30s、真实3重复、名义活跃预算3×(10+30)=120s不减，另计整点所有phase累计≤1s非活跃调度/取消/回滚/join尾部，actualwall≤121s；不是逐phase各1s或额外活跃准入）、`SINGLE-RUN` 15 点（真实 warm-up≥10s、真实活跃测量≥30s、单次actualwall≤60s，整点累计≤1s尾部计入60s内、标 exploratory）、`CORRECTNESS-ONLY` 6 逻辑场景（不产性能样本）；共41逻辑/42单元。X01/empty 与 X01/ten 两 fixture 全留，small 必含 D04-100；D04/N03/N04 有限工作须按 addendum §4 用新唯一合成 ID 后续批维持实际操作，原完整基准批断言不减，新增批/准备/资源均计费；不能用 completion-only、manual summary、sleep/空轮补时冒充有效窗口，不能达成则如实 FAIL/NOT RUN。W02 仅 c1/c16 测量，c4 是 c16 点内观测、不单独 P95；N02-c4 是 N02-c2 同 schema 正确性子断言、不增点/schema；X09 只记 BLOCKED。500样本为软目标，N<500 如实标 exploratory，不缩采样或延长预算；测量 `statement_timeout=30s`、seed/克隆=0、负例 `lock_timeout=5s`、默认=0；共享开发机噪声不可比。可见序列独立 SQL oracle（不用被测分页算法）、404/空页、R06/R07 分开、version/ETag、T1 pending、审计回滚、notice claim 无重复等断言全部保留；phase/point及共同截止后无新op（含复合操作中新业务DB步骤）；boundary-canceled/late/unknown分账且不计窗内成功，结果unknown核不清、业务错、截断/少重复、超限或泄漏真FAIL并停后点。删失时仅完成样本条件分位数须标exploratory，不隐藏长尾；失败即停止性能结论并留证。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S2.2`、`article-management-S2.3`；三项均已关闭，不改其他卡 requires。
- **并行**：[P] 仅表示新增测试文件、文档校正与证据目录在文件/契约/环境隔离的切片可并行准备；同一 `articleintegration` package 与同一 PostgreSQL 实例上的编译、迁移与执行必须串行，不得并发共享 schema/连接池。
- **范围变更与关闭注记（2026-11-05，ADR-0033）**：AM-Q2 已于 2026-11-02 关闭（`DefaultLimit=20`、`MaxLimit=100`、`CandidateBatchSize=200`；首期 100,000 篇/单用户 10,000 篇；P95 参考基线列表 ≤300ms、详情 ≤150ms，仅开发机参考）。原「目标部署环境真实 P95/并发/QPS/数据分布/连接池/告警阈值/retention/RPO/RTO」验证不纳入本版：R09 仅取得八项 PASS、一项 seed FAIL；R10 只完成源码接收与离线/构建检查，整组永久 NOT RUN；R10 冻结 harness 的 adoption 路径硬编码到一份明文声明「不开放新 schema、PG 写入或完整性能窗口」的文件，2026-11-05 复核确认在冻结源上无法合法签发新窗口（见 `author/r11-attempt-20261105/r11-structural-block.md`）。本卡据此按**范围决策**关闭，不是验证通过。
- **边界**：不重开 S2.2/S2.3，不复用 S2.3 进程内数字作结论，不冒充 S1.5/S8.2 结论，不顶替 E2E/部署/备份恢复/RPO-RTO；真实 SMTP/Journal fsync/HTTP wire/retention 等仍不在本卡范围。
- **本任务关闭条件（变更后，2026-11-05 / ADR-0033）**：(1) 范围决策已有项目负责人明确授权，并落为 ADR-0033、同步 `decisions/README.md` 全局索引/领域文件表/场景查询表；(2) 关闭理由写明「经决策不验证」并区分范围决策与验证通过；(3) 未验证边界如实保留：真实 PG 完整 calibration、small/medium/large 三档容量与分位数、并发/QPS、数据分布、目标部署 P95/容量、large 与 canonical、旧 backup/restore 关联；(4) R09 八项 PASS、R10 源码/离线检查与组件测试不被拼接为整组通过；既有 `pg_perf_*` 四文件、`docs/benchmarks/article.md` 与全部 evidence 目录保留原字节，不清理、不删除任何 schema；(5) 不宣称 SLO、目标容量、P95 或 production readiness，不解除 S1.5/S8.2 等其它卡阻塞。原四文件编译、`-list` 命中、12:32 启动门与 18 finding 复审属执行层条件，随范围变更归档，不再作为关闭前置。
- **后续验证关联**：article-management-S1.4（部署后真实 E2E，不在本卡）；S1.5/S8.2 的契约/管理员结论不由本卡替代。
- **设计/测试来源**：原 `.spec/wip/article-management/test-plan-pg-perf-r04.md` 仍为76,409 B / SHA256 `EEB026E5FEE5AB0FE207BD9ABC66C3E2F9227EF5112F226286BF5E6EAEB6FC17`，原独立评审 `.spec/wip/article-management/review-test-plan-pg-perf-r04/report.md`（`9D4AC7DB65EFF3DDF6211184B0349B9153C2AD8C0D3371AD615B2FDE642DAC7F`）及旧证据原字节保留；`PGPERF-A`～G仍为本卡切片。当前有界增量采用 `.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/test-plan-addendum.md` §2～6（现场15,122 B / `2C3F874B43653D4E31C4FD65AA21AAB91CA795386413235E829AD4057B96C774`；初交接15,094 B为过时计数）及同目录 `scenario-schema-map.json`（25,810 B / `383215FA893DC9AC0A8CD8777B4B71C22C1BA411DD79DEF864FBC03B4CE95AD7`）。非作者 Halley / add-reviewer，实际ID `01a1a495-be38-7201-a1d7-5c251cf16900`（Lead提供），同目录 `test-plan-review-r01/report.md`（12,664 B / `ECBDC101942E62CD7DE14167889EBA1A0413D737CCF1CF6DF4802BED9C5DFDAA`）与 `verdict.md`（1,903 B / `6A406752E1BDCBBDC3D3E5C8C03CC0BC379D74ACC6DC3DD195E155C3E630F1B1`）为 **Test Plan Delta Review Gate PASS / 0 finding**，仅预算/timeout/计数delta；Lead本轮明确已实读接受并授权本对账。该规划PASS不核销G0实现问题，不证明执行/性能/资源关闭；原采样、安全、代码/执行Review和Lead同revision关闭义务继续有效。article design `:190/961-984` AM-Q2、S2.2/S2.3及docs仍为原依据，不新增task/文件写集/requires或DAG，不改变active/统计或A-PERF状态。

- [x] article-management-S3.1 实现作者受控详情与强 ETag 修改 完成@2026-10-27

- **目标**：实现作者受控详情与强 ETag 修改。
- **类型**：开发
- **文件**：
  - `internal/article/command_service.go（修改）`
  - `internal/article/query_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/update_test.go（新增）`
  - `internal/article/author_projection_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestAuthorArticle|TestUpdateArticle"`
  - 当前状态：本卡专用命令 `go test ./internal/article -run 'TestAuthorArticle|TestUpdateArticle' -count=1` 与 `-race` 均 PASS（5 个作者详情 + 11 个修改用例）；Manifest 七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）全部 PASS；另以临时 integration probe（已删除、不在写集内）在真实 PostgreSQL 18.6（本机 5433）验证 `LockArticleForUpdate`/`UpdateArticleContent` 的锁定/CAS/状态保持/失配语义 PASS。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.2`、`article-management-S2.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：先证明缺/坏 If-Match、双 writer、非作者与字段越界失败；只改内容/version，violating 约束保持且作者投影最小；独立 Review 通过。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **设计/测试来源**：article Story 3；test A-ETAG/A-MOD
- **独立复审与 Lead Gate**：本 Host 无委派能力，无独立 Reviewer Role 产出；经用户 2026-10-27 授权以**作者自审**口径执行 Code Review，产出 `.spec/wip/article-management/review-code-s3.1-r01.md`（SHA256 `317BE33810150CE99B03DDA04A2A9BE4FB77321B5373128569387879891FA732`），`Code Review Gate: PASS（作者自审口径）`，无 HIGH/MEDIUM finding。Lead 在同 revision 核五文件哈希、专用命令、race、Manifest 七项与真实 PostgreSQL SQL 探针。
- **关闭证据与限制**：最终 SHA256 为 `command_service.go`=`B61ABD12ADD7AF43C7D677A89C097F44FFBB818A85D2D674CD218A81340DB6E4`、`query_service.go`=`D969C1171E81889ABFC0D9CDB44DCD74466BD501828E5EDDB63073CEB1E6160B`、`handler.go`=`B2E3F68995AEC9FA63C5E9F04EF0CCC458A4262045279D7A7DD794C2DD25DD00`、`update_test.go`=`D6EF18BF78C12DC438C03CDC05C6DEAF4D6CF6157718BFF92C0DA4EF29366EAA`、`author_projection_test.go`=`47314C84B79B53A6476F91A6F728CBC521CD78DE36AF61D68DA7298C1B0192F7`。关闭只覆盖五文件代码与本地/真实 PostgreSQL SQL 级验证；production `cmd/han-server` 路由接线（GET `/api/v1/users/me/articles/{id}`、PUT `/api/v1/articles/{id}`）、A-ETAG-02 PostgreSQL 双 writer 并发、A-MOD-04 三视图跨模块契约、A-MOD-06/07/08 精确 N/N+1 前后快照、部署、浏览器/E2E、容量/P95 与 production readiness 均未验证，由 `article-management-S2.2`、`article-management-S1.5`、`article-management-S1.4` 独立闭环。作者自审弱于非作者独立复审，作为已知限制保留。

- [x] article-management-S4.1 实现永久删除协调、Deletion Journal 与 finalizer 开始@2026-10-23 完成@2026-10-23

- **目标**：实现永久删除协调、Deletion Journal 与 finalizer。
- **类型**：开发
- **任务责任与关闭**：本卡六文件实现、r01～r03 非作者代码评审链与 Lead §10 Gate 已完成；Reviewer Turing 的最终 r03 为 `Code Review Gate: PASS`。本次仅同步既有关闭事实，不新增或推断实施者 Agent ID，不修改 requires/DAG。
- **文件**：
  - `internal/article/deletion_coordinator.go（新增）`
  - `internal/article/deletion_finalizer.go（新增）`
  - `internal/article/deletion_journal.go（新增）`
  - `internal/article/command_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/deletion_test.go（新增）`
- **最终对象 SHA256**：`deletion_coordinator.go` `51B720A6ACFBDE81966AA959BF38898C68EB61622D7D428D3344DEE2D4CA0D1D`；`deletion_finalizer.go` `9ABEEE4D1314F52E44B25553C19B3E9989FFC396C738ED43375C3AA5CC319B17`；`deletion_journal.go` `E2539017B9C88EC6C60E3BBC0DA6DFF581EE885AE337E7B17E368129F7193343`；`command_service.go` `58A1FFDFD19C7C7214C07C28F90CF99F50A172B9AD054787A16C90BF08CC0603`；`handler.go` `874A2BAE6AE69C1C96A888E80AC1E35FE94452398B2CABE5F38C81696CE60F0A`；`deletion_test.go` `7D8841D39647EB0AE22E051B89FDE8C30917CF1732EEC6B7F337E5D4517F297B`。
- **独立评审链**：最终报告 `.spec/wip/article-management/review-code-s4.1-r03.md`（blob `41e2c84b67804bf99acd622b876aa7464339575e`，SHA256 `9FE6E887F2D90192CA20DC2FA602DFF1DCF5980EE9107DDE71430315D5401BBC`）确认 `ARTICLE-S4.1-R01-F001`～`F005` 全部 CLOSED、无新增 finding，`Code Review Gate: PASS`；r01/r02 保留为历史追溯。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestPermanentDeletion|TestDeletionJournal|TestDeletionFinalizer"`
  - 当前状态：Lead 现场专用 `go test ./internal/article -run 'TestPermanentDeletion|TestDeletionJournal|TestDeletionFinalizer' -count=1`、directory fsync 回归 ×100、并发 reconciliation ×1000、article package、scoped race、article vet、六文件 gofmt 全部 PASS；Manifest 七项 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...` 全部 PASS。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S1.3`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无；本卡 task-level Gate 已 PASS 并关闭。真实 PostgreSQL、Unix 持久卷、生产接线/worker、Host/部署/E2E 等外部验证仍未做，但不回退本卡代码级关闭。
- **本任务关闭条件**：已满足。Journal/fsync/DB/finalizer 失败不返回 204且无重新可见窗口、重复 DELETE/双 finalizer 收敛、tombstone 无 PII、r03 独立 Review 与 Lead §10 Gate 均已 PASS。
- **关闭证据与未验证边界**：关闭只覆盖六文件代码、确定性/并发/Windows fail-closed 回归、适用包级/race/vet/gofmt、Manifest 七项、非作者 r03 与 Lead 核证。未运行真实 PostgreSQL，真实事务回滚/行锁/lease/commit outcome unknown；未在受支持 Unix 目标卷验证 directory fsync、进程 kill、崩溃、断电、磁盘满/故障；production composition/router 与后台 worker 未接线/未运行，真实 Host、部署、浏览器/E2E、容量与 production readiness 均未验证。Windows 结果仅证明 fail closed。
- **后续验证关联**：article-management-S4.3、article-management-S5.1、article-management-S6.2、article-management-S8.1
- **设计/测试来源**：article Story 4；test A-DEL-01～10；ADR-0020

- [x] article-management-S4.2 实现 ReplayStatus producer、证据存储与旧备份重放 [P] 开始@2026-10-25 完成@2026-10-27

- **目标**：实现 ReplayStatus producer、证据存储与旧备份重放。
- **类型**：开发
- **文件**（精确且唯一 19 个）：
  1. `internal/article/deletion_replay.go（修改）`
  2. `internal/article/replay_evidence.go（修改）`
  3. `internal/article/replay_status_v1.go（修改）`
  4. `internal/article/deletion_replay_test.go（修改）`
  5. `internal/article/replay_status_v1_test.go（修改）`
  6. `internal/article/model.go（修改）`
  7. `internal/article/deletion_replay_source.go（新增）`
  8. `internal/article/deletion_replay_source_test.go（新增）`
  9. `internal/article/backup_inventory.go（新增）`
  10. `internal/article/backup_inventory_test.go（新增）`
  11. `internal/article/deletion_replay_integration_test.go（新增）`
  12. `migrations/0009_article_replay_attempt.sql（新增）`
  13. `internal/article/deletion_journal.go（修改）`
  14. `internal/article/deletion_finalizer.go（修改）`
  15. `internal/article/deletion_test.go（修改）`
  16. `internal/article/repository.go（修改）`
  17. `internal/article/repository_integration_test.go（修改）`
  18. `cmd/han-server/main.go（修改）`
  19. `cmd/han-server/main_test.go（修改）`
- **只读 authority/consumer**：`internal/recovery/replay_status_v1.go` 与 identity ReplayStatus consumer。第 18～19 项必须与已关闭 S4.6 的唯一 composition owner 串行，禁止第二 root。`0009` 只允许前向增加 generation/token 与一致性约束，禁止修改 `0001`～`0008`。
- **当前对象与 Review 证据**：**R01 固定对象 SHA256（历史）**：`deletion_replay.go`=`7B19CB5A08D9A7770CA1294E6FD18821D62D74865D99076AD7A37FA80FC33549`、`replay_evidence.go`=`698E6B18D482E8BFE59D6791774542475A6EB1C2CBDF42541102FA774405C6E8`、`replay_status_v1.go`=`3E14EB098E29664463A895A05280DB4E941DDBC16B76C44A73D97EC709E010F7`、`deletion_replay_test.go`=`60CCA8C40F910231D0F220A440394DBC5DCF84255401005E6CC0B3823FA0A206`、`replay_status_v1_test.go`=`DFAFB02787B895BE0F5E50B018ECD57075D038649BADCD3954B4AE98B241BCBD`。固定 design r14（SHA256 `AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`）与 Design Review R02（SHA256 `85DAC7A66F9555E6DF711472E7F5A9A6448F281EC2496DFDDBD5B7DAC19DE143`）PASS；固定 test-plan r12（SHA256 `85D8A7CC6ABF01A88B24726EEE3957691AA0B8A7B6235FEB8A31677D77336DDF`）与 Test-plan Review R02（SHA256 `53E85D1A4B6A4519829A04A7B6549E808F00E9BAD6D2F85C11EDE11E531C1D0C`）PASS。Code Review R01 `.spec/wip/article-management/review-code-s4.2-r01.md`（SHA256 `777CF2B5029049F86E8B2484DC69C28C11A068A9252F65AB17E8EC97647165C1`）FAIL（3 HIGH + 2 MEDIUM）。**2026-10-26 Lead 接管完整返工后对象 SHA256**（F001～F005 均已实现，R01 关闭待非作者复审）：`deletion_replay.go`=`1A77C574BD134B59C8203702B022FD1F510BBC23689CE4B0ED8FA4448D34C446`、`replay_evidence.go`=`54FAB8529DB4275E10FEB65A5ADBBF64B8CB79E82B39B21AFDD7538079774351`、`replay_status_v1.go`=`244D606F45DC4BC885083F52A3ACCFD065A2F3148ED04B0AECBF7C0F8C62FA16`、`deletion_replay_test.go`=`2E0BDD1E7B5081364C6412F53269658BF159AE17F39772F3CE992FBE4D7D3194`、`replay_status_v1_test.go`=`9788B89EA4590F87AA5D99798A670F13A5D416DB1D8CC0DDB8DE65FED7B3E8CD`、`model.go`=`3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4`、`deletion_replay_source.go`=`0D53B7FF460FF6923EF6E8B6134B3FB40F9D921EE0498C45AD63F1B77B04492C`、`deletion_replay_source_test.go`=`DB669095E4C77ACB30FFFBD610BE99BCC8B169C88386D1059349CB5395B66B9A`、`backup_inventory.go`=`B247DB0CC10723EB1D4464F59A9D543C6421AE296E7AADFF21A440F61C90B7EF`、`backup_inventory_test.go`=`7260756A356795C913AAB391E8CF19F9C53EB779585D81DEB6C8A20BF6049B12`、`deletion_replay_integration_test.go`=`38FE33F115A136A8D6586CEDE2C898ED927AEC11DF2EAA5CC1318B717EA34B14`、`0009_article_replay_attempt.sql`=`4E10F50108A57D709C866044DE6678A00E11A4D6ACB589E708C7DC3779A9AA1D`、`deletion_journal.go`=`0A3F2AE9AEADDDB85C62A1F7CA1BD284AACF149DA0D5ED02184F1E30F0EB4333`、`deletion_finalizer.go`=`989C51D418F10A46880FE38FBB3069CAAA80D66319708E44A6FED8B4D3AB255F`、`deletion_test.go`=`91DE34174E2F0E8EA07257CD2EE98605E383DCCC469A5853552A961DAF0015AB`、`repository.go`=`22551B3B96888A2361A2A55D1C6C650A64494D2198886F85B34BBA8090C3F14C`、`repository_integration_test.go`=`EE9A68CCA141AC6F84C412C6BCE54457C543003ABB2812D4B8C3690D9B269355`、`cmd/han-server/main.go`=`1C9D22526BB84D6E00CE8D3BD1C453CCEB75642C2BD7F2C87CCFF4B3987EAB48`、`cmd/han-server/main_test.go`=`CD86C9012928AA80167FD6915D0D36F75C778C8B9DCC313C0F8B094B8904014A`。
- **复审与最终固定（2026-10-27）**：复审 `.spec/wip/article-management/review-code-s4.2-r02.md`（SHA256 `9D06B84370E196C4881479701EA0B96D6B991A8DD52FB300F964B6061320F3AA`），Code Review Gate PASS（R01 F001～F005 全部 CLOSED、0 新 HIGH/MEDIUM；依据用户 2026-10-26 作者自审授权）。最终 `deletion_replay_integration_test.go` 因新增 A-REC-05 真实 PostgreSQL 用例更新为 `F132361840952A2B4111FDC5289807DB993737F72D536C86D01423C21C8EFA0B`（替换先前 `38FE33F1…34B14`）；`repository_integration_test.go` 实测为 `3A15B7635D97D66AB4DE5CE69155C37E414F75BC3DBB0AD4F14D869019A0DCF4`（先前记录的 `EE9A68CC…269355` 为滞后值，已对账更正）；其余 17 文件与返工后固定值一致。**A-REC-04～07 真实 PostgreSQL 证据（本轮实测，非缓存）**：`go test -tags=integration ./internal/article -run '^TestDeletionReplayPostgres$' -count=1` → `ok 4.87s`（A-REC-04/05/06/07 全 PASS）；`-race` → `ok 6.55s`；`go test -tags=integration ./internal/article -run Repository -count=1` → PASS；Manifest 七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）全部 PASS。Lead 现场核 branch=`main`、HEAD=`6da8abb…`、staged 为空、写集未越出登记 19 文件。该 PASS 仅覆盖 19 文件与 A-REC-04～07 数据库边界，不证明真实旧备份恢复、manifest 真实签发/挂载授权、retention/cleanup、灾备、Host、部署、E2E 或 production readiness。
- **Story-level R gap-map**：`article-management-S4.2`（Story 4 `作者删除文章` 的 replay producer/store/source/composition 切片）→ **已建成（R01 F001～F005 已由返工 + r02 复审闭合）**。历史 Desired 定位：`.add/project/knowledge/design-workspace/article-management/stories.md:94-121`、`.add/project/knowledge/design-workspace/article-management/design.md:87-91,494,554,560-567,1974,2212-2217` 与 `.add/project/knowledge/design-workspace/article-management/test-plan.md:52,60,62,87,292-295` 固定 strict production inventory/file safety/TOCTOU/freshness、唯一 source/wiring、`0009`、generation+token CAS、内部 authoritative handle、UTC 微秒及真实 PostgreSQL A-REC-04～07；当前代码 `internal/article/replay_evidence.go:38-41,135-176,424-499,627-631`、`internal/article/deletion_replay.go:54-83`、`internal/article/replay_status_v1.go:66-73,110-149`、`internal/article/deletion_replay_test.go:15-209,238-477` 仍不符合该合同，且 production source/inventory/wiring、PostgreSQL 集成测试与 `migrations/0009_article_replay_attempt.sql` 尚不存在。Code Review R01 仍 FAIL；design r14/test-plan r12 PASS 只作为返工依据。
- **验收命令**（workdir：仓库根目录 `.`）：
  - 符号前置 Gate（文件、符号或 tag 任一缺失即 FAIL）：
    ```powershell
    $checks = @(
        @{ Path = 'internal/article/backup_inventory_test.go'; Pattern = '^func TestFileBackupInventoryReader\(t \*testing\.T\) \{$' },
        @{ Path = 'internal/article/deletion_replay_source_test.go'; Pattern = '^func TestProductionDeletionReplaySource\(t \*testing\.T\) \{$' },
        @{ Path = 'cmd/han-server/main_test.go'; Pattern = '^func TestArticleReplayProductionWiring\(t \*testing\.T\) \{$' },
        @{ Path = 'internal/article/deletion_replay_integration_test.go'; Pattern = '^func TestDeletionReplayPostgres\(t \*testing\.T\) \{$' }
    )
    foreach ($check in $checks) {
        if (-not (Test-Path -LiteralPath $check.Path -PathType Leaf)) { throw "missing $($check.Path)" }
        if (-not (Select-String -LiteralPath $check.Path -CaseSensitive -Quiet -Pattern $check.Pattern)) { throw "missing exact test symbol in $($check.Path)" }
    }
    if ((Get-Content -LiteralPath 'internal/article/deletion_replay_integration_test.go' -TotalCount 1) -ne '//go:build integration') { throw 'missing exact integration build tag' }
    ```
  - `gofmt -l internal/article/deletion_replay.go internal/article/replay_evidence.go internal/article/replay_status_v1.go internal/article/deletion_replay_test.go internal/article/replay_status_v1_test.go internal/article/model.go internal/article/deletion_replay_source.go internal/article/deletion_replay_source_test.go internal/article/backup_inventory.go internal/article/backup_inventory_test.go internal/article/deletion_replay_integration_test.go internal/article/deletion_journal.go internal/article/deletion_finalizer.go internal/article/deletion_test.go internal/article/repository.go internal/article/repository_integration_test.go cmd/han-server/main.go cmd/han-server/main_test.go`
  - `go test ./internal/article ./cmd/han-server -run '^(TestReplayStatus|TestReplayEvidence|TestDeletionReplay|TestFileBackupInventoryReader|TestProductionDeletionReplaySource|TestArticleReplayProductionWiring)' -count=1`
  - `go test -race ./internal/article ./cmd/han-server -run '^(TestReplayStatus|TestReplayEvidence|TestDeletionReplay|TestFileBackupInventoryReader|TestProductionDeletionReplaySource|TestArticleReplayProductionWiring)' -count=1`
  - 在另行批准的隔离 PostgreSQL 环境执行：`go test -tags=integration ./internal/article -run '^TestDeletionReplayPostgres$' -count=1`
  - 同一批准环境执行：`go test -race -tags=integration ./internal/article -run '^TestDeletionReplayPostgres$' -count=1`
  - `go vet ./internal/article ./cmd/han-server`
  - `go build ./cmd/han-server`
  - `go mod verify`
  - 按 Manifest 固定顺序执行七项：`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`
  - `git diff --check -- internal/article/deletion_replay.go internal/article/replay_evidence.go internal/article/replay_status_v1.go internal/article/deletion_replay_test.go internal/article/replay_status_v1_test.go internal/article/model.go internal/article/deletion_replay_source.go internal/article/deletion_replay_source_test.go internal/article/backup_inventory.go internal/article/backup_inventory_test.go internal/article/deletion_replay_integration_test.go migrations/0009_article_replay_attempt.sql internal/article/deletion_journal.go internal/article/deletion_finalizer.go internal/article/deletion_test.go internal/article/repository.go internal/article/repository_integration_test.go cmd/han-server/main.go cmd/han-server/main_test.go`
  - 第 2～5 项任一输出 `[no tests to run]`，即使退出码为 0 也必须登记 FAIL。真实 PostgreSQL用例须以多连接/多事务覆盖 A-REC-04～07：三类 Begin commit unknown、generation/token CAS、UTC 微秒、旧 Fail/Ready 晚到、TOCTOU/freshness 与 current/evidence 原子性；不得用 memory、fixture、compile-only 或 `t.Skip` 替代。本轮只登记，未执行任何命令，也未读取或使用 `HAN_TEST_POSTGRES_DSN`。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`、`article-management-S4.1`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
**认领状态**：CLOSED（2026-10-27）。B7 内原五文件已有实现候选；R01 自称作者为 Pasteur `01a1716c-c0a1-7263-b02c-0c466e4ecaaa`，但仅作为作者线索，权威 owner 与 Reviewer 均为 unknown。2026-10-26 用户以确认授权与采用建议值给出 Lead START；本 Host 委派不可用（`spawn_agent` unsupported），Lead 接管在精确 19 文件内完成 F001～F005，产出如实标注无 Backend-Dev Role 产出。依据用户 2026-10-26 明确授权「由作者自审 Code Review」，由 Lead 以作者自审口径输出 `.spec/wip/article-management/review-code-s4.2-r02.md`，F001～F005 全部 CLOSED、0 新 HIGH/MEDIUM、Code Review Gate PASS（该口径弱于非作者独立复审，作为已知限制保留）；Lead 现场核 branch/HEAD/staged/哈希/唯一写集后关闭本卡。
- **阻塞注记**：已解除（2026-10-27 CLOSED）。F001～F005 已实现并在真实 PostgreSQL（含 A-REC-05）验证 PASS；原「非作者独立 Code Review」阻塞由用户 2026-10-26 作者自审授权解除。第 20 文件、新 provider/协议/依赖、第二 source/Journal parser/inventory reader/provider/root、global/service locator、修改 `0001`～`0008`、公共合同或 architecture/ADR 变化仍为停止条件。本次未改 `0001`～`0008`，仅新增 `0009`；未修改 identity consumer 或复制第二 ReplayStatus authority；本轮仅在已登记的第 11 项 `deletion_replay_integration_test.go` 内追加 A-REC-05 用例，写集未越界。
- **本任务关闭条件**：Planning Review PASS 且 Lead 明确 START后，精确 19 文件内完成 strict variant、production inventory/file safety/TOCTOU/freshness、唯一 source/wiring、generation/token CAS、UTC 微秒与 authority；普通/race、A-REC-04～07 真实 PostgreSQL普通/race、vet/build/Manifest/diff Gate均真实通过且无 `[no tests to run]`；R01 F001～F005 经（授权作者自审口径）复审关闭，Lead 同 revision Gate PASS。已满足并关闭@2026-10-27。关闭不证明真实旧备份恢复、manifest 真实签发/挂载授权、灾备、Host、部署、E2E 或 production readiness。
- **后续验证关联**：identity-auth-S1.5、article-management-S1.5、article-management-S4.4
- **设计/测试来源**：article design r14、test-plan r12 §9.3、Article Story 4/S16；test A-REC-01～07

- [x] article-management-S4.3 验证 Article Deletion Journal 与 finalizer 韧性 [P] 开始@2026-10-25 完成@2026-10-27

- **目标**：验证 Article Deletion Journal 与 finalizer 韧性。
- **类型**：专项韧性测试
- **文件**：
  - `test/integration/article/deletion_journal_test.go（新增）`
  - `test/integration/article/deletion_finalizer_test.go（新增）`
  - `test/integration/article/journal_faults_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/article -run "TestDeletionJournal|TestDeletionFinalizer"`
  - 当前状态：本卡三文件未创建、未执行；专用命令 `go test -tags=integration ./test/integration/article -run "TestDeletionJournal|TestDeletionFinalizer"` 与 Manifest 七项均 NOT RUN。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S4.1`、`article-management-S4.6`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **认领状态**：B7 内独占本卡三个 integration 测试文件；与同批其他五卡及 active S3.3/S3.5 无写集交叉。实施 Agent 与非作者 Reviewer待 Lead 分配，尚未实施、测试或 Review。
- **范围变更与关闭注记**：按 ADR-0024（2026-10-27）**经决策不验证**关闭，不是验证通过。本项目为测试项目、非真实交付项目，真实 Unix 独立持久卷上的 `flock`/file+directory `fsync`/崩溃/磁盘故障韧性验证不纳入本版交付范围；三文件写集保持未创建、未执行。
- **本任务关闭条件（变更后）**：经验证的真实卷部分不存在。本卡按 ADR-0024 以范围决策关闭，条件为：①用户明确决定不做真卷韧性验证（已满足，2026-10-27）；②ADR-0024 已落库并同步索引；③未验证边界（真实跨进程 `flock`、file/directory `fsync` 耐久性、崩溃/断电、磁盘空间与介质故障、真实 journal provider/root/keyring）如实登记；④不以 Windows/fake 冒充真实卷结论。
- **后续验证关联**：article-management-S2.3、article-management-S4.4、article-management-S1.4
- **设计/测试来源**：article test Article Deletion Journal/finalizer 专项

- [x] article-management-S4.4 验证 ReplayStatus producer 与旧备份恢复（三路径模拟环境收口；真实备份演练边界保留） 开始@2026-10-28 完成@2026-10-31
- **ADR-0028 适用（2026-10-31）**：按项目级验证授权执行；可执行层仍实际执行留证，真实环境可模拟并标注“未真实验证”，评审/签收可由 AI 代理并记录授权来源，不冒充真人签字。
- **2026-10-31 S4.4 模拟环境收口 r01（minor-01/minor-02 CLOSED；整卡未闭）**：Lead 直改测试侧并亲跑，修改 `test/integration/recovery/article_replay_test.go`（SHA256 `1BD2B0ED9385973533C611EEEF9CD17A9B70149AFBD5339829B9EDEF9906B0CD`）与 `test/integration/recovery/old_backup_restore_test.go`（`36B13366F968FF9BA1BD1AA00AF90458A7EE89DD3A7FCBDFE69F6FF07317E536`），新增 `internal/article/deletion_journal_hash_test.go`（`10F940D8C18984BD5D806BB9D5D6379DC644B22ACE626EB2BF6E4A59AC33FB48`），生产代码零改动。`(*s44PublicationRuntime).CheckPhase` 覆盖 `recovery.PhaseRecoveryReplay` 与未导出 `replayPublicationInstall("install")` 两个 purpose，非法 purpose/ctx 返回 `ErrAdmissionBinding`；两份测试侧 hash 复刻合并为单一 `s44PublicationRecordHash`；新增删除日志 hash 钉住测试，期望摘要 `sha256:4ce06ce79ad8639d2468f0774d6d2d2097909bab6203a2dd5f63701d2f4bfacf` 由 PowerShell/.NET 独立计算（475B、UTC RFC3339Nano、无 HTML 转义）。三路径集成重跑 PASS（`ok 19.872s`）；独立评审 Confucius/add-reviewer R01 PASS WITH MINOR（报告 `.spec/wip/article-management/review-sim-closeout-r01.md` SHA256 `3BD9139322745DC907C59F81FCBB114879FDA348FC18FA0613174CADAA31DAAE`），`S15-SIM-F003`（`CheckPhase` 当前三路径不可达）保留为已披露边界。证据 `.spec/wip/article-management/evidence-s4.4/sim-closeout-r01/`（`index.json`、5 条命令日志）。**边界不变**：真实 POSIX Journal fsync/flock、真实 keyring/HMAC、真实旧备份恢复/RPO/RTO、identity 双 namespace/MarkConsumed 未真实验证；Case18 `NOT RUN/unknown`；AI 代理评审与登记，非真人签字。


- **2026-10-31 第三路径（旧备份恢复）技术子集 + 共享 harness attributes 补齐（Lead 现场复核通过；整卡未闭）**：Planck / add-backend-dev（actualID `01a190e6-877a-7272-b7e8-58332164b10a`）在 R01 复审后补齐共享 harness 的 publication ports 装配，并把 `migrations/0011_article_replay_publication.sql` 收敛为唯一幂等入口 `s44ApplyPublicationMigration`（`article_replay_test.go:254` 调用、`:261` 定义、`:267` 已存在即跳过），`old_backup_restore_test.go:290` 改为纯委托。证据 `.spec/wip/article-management/evidence-s4.4/harness-publication-r01/`（`index.json` `4E3D54C1…`、`lead-verify-r01.log` `BCE990A6…`、`retained-resources.ndjson` 102 行 `ACE4E82D…`）。Lead 独立复核文件实际字节与哈希（`article_replay_test.go` `04172354…` 86039B、`replay_evidence_test.go` `c23c5c77…` 29878B、`old_backup_restore_test.go` `e0c202f2…` 15848B/385 行），`go build ./...`/`go vet -tags=integration`/`gofmt -l` 均 0，并**亲跑** `go test -tags=integration ./test/integration/recovery -run '^(TestArticleReplayPostgres|TestArticleReplayEvidencePostgres|TestOldBackupRestore)$' -count=1 -v` → exit 0、`ok 24.433s`：`TestArticleReplayPostgres` 17.42s/16 leaf、`TestOldBackupRestore` 2.96s/4 子测试、`TestArticleReplayEvidencePostgres` 3.59s/3 leaf 全 PASS。这同时解除了此前记录的“既有第一路径在当前 HEAD 必失败”阻塞。Lead 裁定 `PublishCommitAckLoss` 断言改为精确事件序列 `[ReadCurrent, Begin, Publish, CommitObservedThenAckLost]` 并保留唯一 Begin/Publish/ID/Clock 与禁 Fail、evidence=1 检查，属**更严**，未放宽。`old-backup-r01/index.json` 已升 r03（`6CA7332B…`）：修正 `final_log` 指向 `run-02.log` 并登记字节相同的重复拷贝、`retained_schemas` 由 6 补足 12 条并区分有无日志、新增 `superseded_for_current_tree` 指向本轮权威证据。**边界不变**：合成 fixture/Journal/Runtime/PublicationIDs，不证明真实备份恢复、POSIX Journal/fsync/flock、真实 keyring/HMAC、identity 双 namespace 与 MarkConsumed、RPO/RTO、生产 `main.go` 未调用 `NewReplayPublicationPorts`（归 `identity-auth-S4.20`）。已派非作者复审 R02；R02 与 Lead Gate 前本卡仍 `[-]`，关闭条件不变。
- **2026-10-31 S4.4 第三路径 R02 独立复审（有界 PASS；整卡仍未闭）**：非作者复审（Boyle / add-reviewer，actualID `01a190fb-de9f-76a2-93f3-a9e5d3ae3eae`）报告 [`review-code-s4.4-old-backup-r02.md`](../../../../.spec/wip/article-management/review-code-s4.4-old-backup-r02.md) SHA256 `1BC70302A8873068CDFF9C022E569E25F2BAE45E10585A04468C848F417B6E17`（2026-10-31 按用户确认脱敏后；原 `7770F3EB…`）：**第三路径 test-only 技术子集 PASS（有界）**，真实 PG 18.6 + 真实 Store/Provider/Source/Finalizer 写路径 4/4 PASS，未发现 blocker、未发现伪造 PASS；**S4.4 整卡仍不可关闭**（R01“整卡不关闭”结论复核成立，生产装配缺口 `main.go:3290` 未调用 `NewReplayPublicationPorts`，归 `identity-auth-S4.20`）。finding 状态：`major-01` CLOSED（单入口 `s44ApplyPublicationMigration`，`article_replay_test.go:254/:261/:267`，run-05 实证 `applied_in_tx`→`already_applied`）、`major-02` CLOSED（仅边界登记，生产装配 OPEN）、`minor-01` OPEN（LOW，非阻断：两份 hash 复刻仍在，生产 `validateReplayPublicationSnapshot` 不重算 hash，未来投影变化会静默漂移）、`minor-02` OPEN（LOW，非阻断：合成 Runtime 仅实现 `ReadPermitBinding`，`CheckPhase` 未覆盖，当前路径不触发）、`minor-03`/`minor-04` CLOSED。新增 `R02-N01`（`harness-publication-r01/index.json` 的 `retained_resources.sha256`/`record_count` 与实际 ndjson 不符）已由 Lead 修正为 `ACE4E82D…`/102 条/`attributed_by_transcript` 61 并追加 `metadata_correction` 说明（索引 SHA256 变为 `6878BFE699FB48A2A5A693CD972D5CACC642091EEA4499E2DFAAE5D6420A4A55`）；`R02-N02`（两套 assembler 共存、`old_backup_restore_test.go:248` 注释过时）已由 Lead 仅改注释消除（文件 SHA256 由 `e0c202f2…` 变为 `A3320C51A064C132E082DE34C4E460AF1ADC6032435B3830512EA3F670676AFA`，行为未变，`gofmt -l` 空）。Lead 复核 Boyle 专项结论：ACK-loss 断言“更严”成立（精确序列 + 唯一 Begin/Publish/ID/Clock + 禁 Fail + evidence=1，commit 可见性由独立 SQL oracle 覆盖），`snapshotHook` 无死锁路径、`s44StoreProbe` 未导致漏测、双 Store 封存符合生产 sealing 语义。**边界不变**：合成 fixture/Journal/Runtime/PublicationIDs；不证明真实备份恢复、POSIX Journal/fsync/flock、真实 keyring/HMAC、identity 双 namespace/MarkConsumed、RPO/RTO。`minor-01/02` 作为残余登记，不阻断本卡有界结论。
- **2026-10-28 current UPDATE 回滚子集（Lead核收；整卡未闭）**：Euler仅修改原第二路径 replay_evidence_test.go，新增 CurrentUpdateRollback，最终SHA256 CF1B07C58F5A7DDCDEA256484CFC9E5F1C0E595B0A1A13B3B1B49DDB410E9048；原两leaf/helper及另外184输入保持。作者与非作者Curie亲自普通/race各4个run/pass节点、3leaf、零fail/skip/extra，tagged vet各exit0，各6个独立新schema/92份payload全保留。真实同事务成功INSERT后，schema-local BEFORE current UPDATE trigger抛23514；七表字段/xmin/ctid完整回滚、owner/Input保持、pending零写与NOWAIT资源归还、同handle新ref重试及稳定Ready均获验证。Curie[两轮独立Review](../../../../.spec/wip/article-management/review-code-s4.4-current-update-r01.md)有界PASS/0finding，报告SHA256 938C2EBE4907E9B1D85F529E3BF12CBF512277C8487A5FBC98919E3AA2027470。Lead核双方原始JSON、逐child185前后图/184payload哈希/SQL像/calls，独立重算新leaf四次运行的8组Ready canonical及operation/audit摘要；当前185 Manifest七项新执行全0/gofmt空。见[本轮完整证据](../../../../.spec/wip/article-management/evidence-s4.4/current-update-r01/index.json)与[唯一Lead交接](../../../../.spec/wip/article-management/verification-s4.4.md)。仅补齐A-REC-07的post-INSERT current UPDATE失败子集；完整A-REC-07、真实旧备份/真Journal/双namespace/MarkConsumed/global opening/inventory/ACL/RPO-RTO-retention/S8.2/Case18继续原hold。原三路径/六requires/fullclose/开始日期/[-]及人的精确接收、Issue状态保持；以下181/180数字与两leaf边界为对应历史阶段。

- **目标**：验证 ReplayStatus producer 与旧备份恢复。
- **类型**：专项契约/韧性测试
- **181输入阶段第二原路径两leaf交付（历史证据；整卡未闭）**：Hegel仅新增 `test/integration/recovery/replay_evidence_test.go`，482行、SHA256 `BE6C689273783CFCB4939EE987397E09DB175B30506D9C0E49881FA8CE83A925`；[固定181 flat](../../../../.spec/wip/article-management/evidence-s4.4/evidence-followup-prep-r01/lead/source181.flat.json) SHA256 `D8EB37B52BC50CFC78C2E6F6014DAA790B9B10C1711C29D537756BAFEE7B745A`，其他180全部原字节，159Go仅为编译子集。唯一 `TestArticleReplayEvidencePostgres`，仅 `DifferentBackupFirstBinding`、`PublishRollback` 两leaf；作者与Erdos亲自普通/race分别各3run/pass节点、2leaf、0fail/skip，tagged vet各exit0，Reviewer独立只读preflight0/四child全部wait。作者4schema、Reviewer另4schema不交叉，8个全retain，不冒称全库盘点。不同backup用两份真实生产reader消费的合成backup-ref manifests、两个实际backend Begin和真实blocking；Publish以foreign-run预留ref触发真实PK23505，失败前后完整像/xmin/ctid不变，原owner/input只换新ID重试，不另Begin/Replay，foreign行保留。owner/T状态/noReplay取固定执行断言；可选XID的waiter实际null，不虚构两个非空XID。此为evidence INSERT自身失败/无部分提交与重试子集，不证明INSERT成功后current UPDATE失败或完整A-REC-07。
- **本轮独立PASS/Lead核收与证据导航**：[Erdos正式独立CodeReview R01](../../../../.spec/wip/article-management/review-code-s4.4-evidence-r01.md) 两轮有界PASS/0finding；最终报告已完成六requires/S1.5十二requires措辞更正，技术对象BE6及两轮PASS不变；本卡仅按正式路径引用，最终索引统一导航verification，不重复冻结报告hash。Lead已实读报告/新文件及实际diff，核双方全部raw-meta、181live/snapshot全部前后图、各58payload/6组Ready-SQL-Facts，独立重算operation/audit摘要，真实blocking/23505/回滚像/重试/foreign保留已核收。[Prep/CODE/静态/181绑定及Manifest索引](../../../../.spec/wip/article-management/evidence-s4.4/evidence-followup-prep-r01/index.json) SHA256 `2AB5AF916504D0F0848E5AA803E2453817808588CE379E1DA7027974E367B683`（51payload），[作者PG及Lead核证索引](../../../../.spec/wip/article-management/evidence-s4.4/evidence-followup-pg-r01/index.json) SHA256 `AAB90277426341B5ADA4BB8E2E32C8A8983548BB6742026E0AFF00C432F262F7`（85payload/4case）；[Lead181 Manifest七项](../../../../.spec/wip/article-management/evidence-s4.4/evidence-followup-prep-r01/lead/lead-evidence181-r01-results.json) SHA256 `19526144B6775B9FF974D92229326AE92220F096CE3A28C59546D89BA489F1FC` 原序全exit0/gofmt空/181前后不漂；integration普通/race取双方显式tag结果，不用默认Manifest race代替。最新接收及后续独立Review durable统一导航[verification原件](../../../../.spec/wip/article-management/verification-s4.4.md)，不写未来pack链接或未知hash。当前仅原第三路径 `test/integration/recovery/old_backup_restore_test.go` 仍未创建，原三路径/六requires/fullclose/[-]/开始日期及59-44-5-10不变；人的精确两方接收仍pending，不关闭整卡或原P1 Issue，不外推真UnixJournal/HMAC/真实backup/双namespace/MarkConsumed/global恢复、inventory/RPO-RTO-retention/ACL/S8.2/Case18等hold。用户已明确另一会话结束，此事实不授root/identity/架构写权或释放生产恢复Gate。以下180/新16旧6/236contract等保留为对应修复阶段真实历史证据，早期unknown及准备/授权文字仍按各自时点消费。
- **180项计数/Trace修复阶段交付（当时Lead核收；历史真实证据）**：原Issue四生产＋五既有测试由Dalton完成，新增 `test/integration/recovery/article_replay_test.go` 由Hegel接手补齐并执行；统一固定180输入[flat清单](../../../../.spec/wip/article-management/evidence-s4.4/lead-checks-r01/source180-r01.flat.json) SHA256 `18270024690941D7A20294FE78C56696FCFD8B19414B07823ED3E5E8368D2071`，新文件SHA256 `B65FAE92A2AE95E291C8CB2CC136769E22209C4E888546CEDF43CC68727F659A`，其他170项保护。旧API计数/两种Trace三leaf真实RED后，同场景修复回归GREEN；Dalton unit普通/race各67节点/59leaf、四contract各286节点/236leaf全部原名保留、0fail/skip，producer仅六处typed内部mock/回调联动，其余三contract及AuditSQL字节保护。Hegel新PG普通/race各24节点/16终端leaf、0fail/skip、32schema保留；四组13命名风险场景展开数按实际JSON消费，不以13代实际leaf数。仅Journal公开controlled seam，真实Finalizer/PG facts→Store→Provider/共享decoder的本片子集已交付，不是整卡、全生产/E2E或完整恢复PASS。
- **180项修复阶段独立CodeReview/Lead核收（历史真实证据）**：[Erdos两轮独立CodeReview R01](../../../../.spec/wip/article-management/review-code-s4.4-issue0001-r01.md) SHA256 `68C412EC34D66272AA4F4783ABC8F6EB4105AB0990E838B2A9AE9736A7F49BAC` 有界PASS/0finding/stopped；Erdos亲自新PG普通/race各24节点16leaf、授权retention-only overlay旧六PG各7节点6leaf，均0fail/skip/tagged vet0，全部child已wait。其46schema=新32＋旧14（含原unused pool两份），与作者32schema不重叠、全部retain，非全库盘点或全机进程普查；旧autoDROP harness仍不得无overlay直跑，原六断言及完整harness-only patch保留。Lead已实读报告及7条实际raw/meta、live/snapshot180全部前后像、独立58组Ready/SQL/Facts及366payload、overlay仅old integration一个编译输入替换，明确技术核收有界独立Gate；[Lead核证durable原件](../../../../.spec/wip/article-management/evidence-s4.4/review-r01/lead/lead-reviewer-verification.json)保留来源，不当Planner或Lead重跑PG。[Lead Manifest七项回执](../../../../.spec/wip/article-management/evidence-s4.4/lead-checks-r01/lead-manifest-recovery-r01-results.json) SHA256 `F0184DC556CBFA666B7D301A048FCBF8A5E742822EABC228406FE7CC69459D5F` 全exit0/gofmt空/180前后不漂；默认全仓race不带integration，本片race取显式tag结果，不称full-profile或全仓平台0skip。
- **180项修复阶段证据与当前剩余责任**：[PG durable索引](../../../../.spec/wip/article-management/evidence-s4.4/pg-execution-r01/index.json) SHA256 `91C326F7C0E50EA66E18A7854C671F747D481E909AB6021C40821E634EAFACA1`、[九文件修复索引](../../../../.spec/wip/article-management/evidence-s4.4/repair-nine-r01/index.json) SHA256 `6907F001C14F48F8F6D56A8D91AB4AA6F7BF33CB370DEA428D1BC68303369CB2`、[Lead检查索引](../../../../.spec/wip/article-management/evidence-s4.4/lead-checks-r01/index.json) SHA256 `314393F2A6FBB8EE92F54A4D48C131C4F374F0D0F5B22A0DA433A158345C4609`；最新接收导航见[verification](../../../../.spec/wip/article-management/verification-s4.4.md)。[独立Review durable索引](../../../../.spec/wip/article-management/evidence-s4.4/review-r01/index.json) SHA256 `B6691BAAAA23113EC02A996394B5619B163ECE4371C6FE0A9CB244B6B4D859A4` 已固定（414份副本、32个新PG case资料、46schema记录，含报告68C412及Lead当前核收proof）；旧14schema仅原断言/退出记录与最终inventory两文件，无初始文件独立归档，不扩大证据声明；[唯一RCA](../../../../.spec/wip/article-management/rca-issue0001-r01.md)收束中仅链接、不freeze旧hash。Issue仅技术修复已核收，人的精确两方结果接收仍pending，不代签closed/accepted/checkpoint。S4.4保持[-]/原开始日期/59-44-5-10，原三路径/六requires/fullclose不变，当前仅原第三路径 `test/integration/recovery/old_backup_restore_test.go` 仍未创建；真UnixJournal/HMAC/keyring/旧备份/双namespace/MarkConsumed/global opening、inventory授权/RPO-RTO-retention/ACL、S8.2/Case18仍hold，S4.3 ADR0024不构成S4.4新豁免。用户最新已明确另一会话结束；本片仍无root/identity/架构写权、未释放生产恢复Gate，后续变动另协调、不扩S10.5、不重开S4.2/S1.5。下方首RED/未授九existing修复/Hegel仅Temp/Reviewer准备等为各自点时历史，当前技术交付与非作者PASS以上述固定证据为准，不继续表述为当前pending。
- **固定设计/规划依据（2026-10-28）**：[design r22](../design-workspace/article-management/design.md) SHA256 `8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E`、[Pauli DesignReview R01](../../../../.spec/wip/article-management/review-design-r22-replay-counters-r01.md) SHA256 `74BB3F80998A08EA917347DD68D042CB995BAF576CD3EA9B9203300A58FD99A1` 有界设计PASS；[test-plan r20 §4.8.1](../design-workspace/article-management/test-plan.md) SHA256 `DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA`、[Pauli独立TestPlanReview R01](../../../../.spec/wip/article-management/review-test-plan-r20-s4.4-r01.md) SHA256 `1F5FBF2FC190AA7429C79441FD88CEADC35F52C7AAC1E0166041B3BB7B6E4402` 两轮有界PASS/0finding/stopped，Lead已读并采用；不等于修复、运行或独立CodeReview PASS。
- **首阶段授权/唯一writer（首RED时点历史）**：Lead现仅授原Dalton `01a17dcf-f2ed-7dc2-b7f3-6c72cf771448` 新增 `test/integration/recovery/article_replay_test.go` 的old-API CODE＋LOCALPG首RED阶段，原179输入保护；先隔离计数行为，再留真实Source→Finalizer合法parent/缺省Trace链证据，不用未来API编译失败或adapter改ctx/参数冒充RED。实际结果待作者回交，不预填RED/GREEN；九existing修复须真实RED交Lead核后另发信号。Hegel `01a17f82-6dce-7531-967b-0ccadd3ec777` 仅Temp准备后续10场景、文件所有权未转，后续test-only范围/窗口另协调；本metadata只消费Lead实际授权，不代发新START。
- **原Issue后续4＋5（修复信号前的规划记录）**：归 [ISSUE-article-management-0001](../../../../.spec/issues/ISSUE-article-management-0001.md)，生产精确为 `internal/article/deletion_replay_source.go`、`internal/article/replay_evidence.go`、`internal/article/deletion_replay.go`、`internal/article/replay_status_v1.go`；既有测试精确为 `internal/article/deletion_replay_source_test.go`、`internal/article/deletion_replay_test.go`、`internal/article/deletion_replay_integration_test.go`、`internal/article/replay_status_v1_test.go`、`test/contract/article/replay_status_producer_test.go`。producer仅内部typed port/mocks必要联动，四contract top/原23组/16Handler/236leaf断言完整普通-race回归，新增语义实际计数另解析；其余三contract与AuditSQL保护，不重开S4.2/S1.5、不改三路径/六requires/fullclose、不新增Task/requires。
- **原实施验收/保护安排（首RED时点；当前交付见上）**：唯一 `TestArticleReplayPostgres` 四组13个必需命名风险场景及AllPending/OperationAuditDrift/Refresh必需嵌套子例按r20实际命中，不把13当实际底层leaf数或首RED当全top通过。Journal仅controlled公开seam；后续修复、普通/race/tagged vet、独立CodeReview和Lead接收分别取证，新证据/schema全retain。既有普通unit的t.TempDir生命周期不等于Host删项目材料，不据此机械阻断回归；旧六PG自动DROP harness禁止直跑，零匹配tag检查仅编译、retention-only Temp overlay另授。整卡[-]/59-44-5-10不变，真Journal/旧备份/全局恢复/S8.2/Case18/应用ACL/identity与root writer unknown不解除。下方r19未固定、Hubble只读/合同待澄清、未授CODE/PG均为原准备时点历史，现由本段接续，非完整修复或S4.4/B8A PASS。
- **准备认领/授权边界（早期准备时点历史，2026-10-28）**：Lead已实际采用Ohm最小测试候选作为准备方向，并授权登记“恢复专项准备进行中/执行待合同与窗口”；仅据本轮Lead认领来源转[-]，不是完整实施START，不授CODE/PG或尚未固定的修复写集/合同。准备协调由Lead承接，Ohm只读gap准备及Dalton诊断均已stopped，Hubble仅只读咨询计数语义/最小修订、无design/stories写权；不将候选作者或Role推断为新实施承接。六项requires均[x]，原三路径及完整关闭条件原文保留，正式执行仍受以下门禁限制。
- **R准备对账/未固定候选（r19准备时点历史）**：现役producer/source/Finalizer/PGStore已建成，但计数canonical接缝已确认需改，修复合同/文件尚待原设计owner澄清；本卡三测试文件均不存在/未执行，属于待做，无已确认废弃范围。[准备索引](../../../../.spec/wip/article-management/evidence-s4.4/prep-r01/index.json) SHA256 `DD878216BD69694446ED159629D10BD2C467FCB49FB9B6DFE13B3EAE52D6A66A`；[Ohm最小gap建议](../../../../.spec/wip/article-management/evidence-s4.4/prep-r01/test-engineer/prep-s44-gap.md) SHA256 `1F34D147A013E0C132182A25E34A23378B6387291308106A9F879A8716356CF8`。仅候选原 `article_replay_test.go` / `TestArticleReplayPostgres` 四组：真实Finalizer/PG facts→Store→Ready20；absent/mixed/repeat稳定性；operation/audit drift后重新Ready；真实Publish commit后丢ACK。只Journal用公开controlled seam，其余两原路径/三类tombstone/fullclose不削减。候选尚未写入test-plan、未固定/评审/实施，当前test-plan仍r19；本段不是新增测试验收合同或CODE/PG信号。
- **初始P1/组件诊断边界（修复前历史）**：[ISSUE-article-management-0001](../../../../.spec/issues/ISSUE-article-management-0001.md) 已定位source counter canonical mismatch，必要计数合同待澄清、无修复/接收关闭；[59份诊断索引](../../../../.spec/wip/article-management/evidence-s4.4/diagnostic-r01/index.json) SHA256 `FE693C3DD308408DD78D43FCBA56654903FE46C19F3B9BE23DE9D72468E52999`。Dalton仅在179输入Temp overlay三次实际编译复现：control PASS、completed/absent FAIL；同一固定事实仅两计数不同，ReplayOldBackup为0/1而Snapshot为1/0，共享canonical摘要不一致。实际Provider加原memoryStore helper返回unknown/evidence_stale/evidence=0，后ReadStatus为pending/replay_in_progress；这不是PG故障实测，PG对比分支仅静态核。原S4.2 closed及授权作者自审CodeR02/原Lead结果保持历史有效，不改称独立评审、不重开旧卡、不篡改旧证据。
- **文件**：
  - `test/integration/recovery/article_replay_test.go（新增）`
  - `test/integration/recovery/replay_evidence_test.go（新增）`
  - `test/integration/recovery/old_backup_restore_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test -tags=integration ./test/integration/recovery -run "TestArticleReplay|TestOldBackupRestore"`
  - 历史非本卡证据：以下误贴identity专项/Manifest结果仅作历史追溯，不能替代本卡三文件/恢复专项执行；本卡在原登记时未创建未执行；当前交付以最新结果段为准。原记录：专用命令 `go test -tags=integration ./test/integration/identity -count=1`（真实 PG18.6）与 `-race` 均 PASS；Manifest 固定七项（`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`）已按序全部 PASS。二者互不替代。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`、`identity-auth-S4.3`、`article-management-S4.2`、`article-management-S4.3`、`article-management-S4.5`
- **并行/后续写窗**：原180项修复阶段按上述历史交接保留；本轮Hegel仅新增第二原路径交付，Erdos两轮有界PASS，最终报告已完成轻微措辞更正；本记录仅导航原件、不另发CODE/PG/源文件写权。用户已明确另一会话结束，仍不授本片root/identity/architecture写入，不等于生产恢复Gate释放。
- **阻塞注记**：r22/r20、原计数/Trace修复与本轮第二路径两leaf真实PG子集、独立CodeReview/Lead181项目检查已按对应结果段核收；人的精确两方结果接收仍pending，原完整恢复与整卡关闭门禁继续有效，不代签Issue关闭。Lead既有只读环境核对为主配置/全部fallback均loopback:5433、PG180006可连接并close/无DDL，该预检仅为执行前时点事实；当前实际PG结果与证据见最新交付段。Windows公开FileJournal明确unsupported；既有WSL status50/User Lxss0、无Docker/Podman命令的检查未发现本机可用Unix载体，不外推未披露远端、不安装/绕过私有构造。真Journal/old backup/inventory真实授权、RPO/RTO/retention及联合identity恢复隔离/完整成功开流仍unknown/blocked，局部Source计划不能解除；S4.3按ADR0024范围决定关闭不是真卷PASS，也不豁免或放弃S4.4。用户已明确另一会话结束；本片仍无root/identity/架构写权、未释放生产恢复Gate，不扩identity S10.5。本次仅完成声明的PG/技术修复子集，当前仅原old_backup_restore_test.go仍未创建，原三路径完整专项尚未交付；其他卡历史、组件诊断、规划或本片有界PASS均不等于S4.4/full恢复通过。
- **2026-10-31 关闭时事实更正**：上述“当前仅原 old_backup_restore_test.go 仍未创建”与“main.go 未调用 NewReplayPublicationPorts”等为历史时点；当前三路径均已存在，production seal 已接线，旧备份恢复仅有模拟 fixture 子集，真实备份演练仍未关闭。现行证据、索引更正与未验证边界见文件末尾 S4.4 关闭记录。
- **本任务关闭条件**：三类 tombstone、operation/audit 对账、双 backup 并发、输入漂移、双次 ReadStatus 与旧备份恢复通过；缺 inventory/RPO/RTO 时不关闭真实演练部分；独立 Review 通过。
- **后续验证关联**：identity-auth-S4.4、article-management-S2.3、article-management-S1.4
- **设计/测试来源**：article test ReplayStatus producer 与旧备份恢复专项；A-REC-01～08

- [x] article-management-S4.5 准备备份 inventory、tombstone/key/evidence 留存输入 [P] 开始@2026-10-25 完成@2026-10-25

- **目标**：准备备份 inventory、tombstone/key/evidence 留存输入。
- **类型**：运行输入准备
- **owner / Reviewer 历史**：Backend-Dev Aristotle the 2nd `01a17225-a3d0-7fd1-aeb9-622f7617f436` 为唯一实施 owner；非作者 Reviewer Gauss 完成 R01/R02。任务关闭后固定四文件独占写窗口已释放。
- **文件**：
  - `docs/runbooks/article-deletion-recovery.md（新增）`
  - `testdata/recovery/article-backup-inventory.json（新增）`
  - `testdata/recovery/article-deletion-fixtures.json（新增）`
  - `test/integration/recovery/article_fixtures_test.go（新增）`
- **Story-level R gap-map**：`article-management-S4.5`（Story 4 `作者删除文章` 的合成恢复输入切片）→ **已建成**。Desired 定位：`.add/project/knowledge/design-workspace/article-management/stories.md:94-121` 与 `.add/project/knowledge/design-workspace/article-management/test-plan.md:119,598,717-722` 固定四文件制品、strict parse/cross-reference/runbook 自动检查及精确 `TestArticleRecoveryFixtures`；当前可定位实现证据为 `docs/runbooks/article-deletion-recovery.md`（SHA256 `41ECB0CF593511119A1BBC910C3B707D094282D1D06875134F531D58F6DD4A09`）、`testdata/recovery/article-backup-inventory.json`（SHA256 `1355094A5A6751E6E479F4769C9D6E41C993F2A054D23ACF948D3F7E5C4E9B8C`）、`testdata/recovery/article-deletion-fixtures.json`（SHA256 `E86F32A1731E04F3B1590A87282F71F4332B488F2B6040E74AAF7F5876CF7D81`）与 `test/integration/recovery/article_fixtures_test.go`（SHA256 `88FD408A466CAF208E780112F82B6EB82BCCDB58D5CB1E848C07B52DFDC3C4F6`）；R02 与 Lead 同 revision Gate 支撑本卡关闭。
- **验收命令**（workdir：仓库根目录 `.`）：
  - 精确符号前置 Gate（任一缺失即停止并登记 FAIL）：
    ```powershell
    $path = 'test/integration/recovery/article_fixtures_test.go'
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "missing $path"
    }
    if ((Get-Content -LiteralPath $path -TotalCount 1) -ne '//go:build integration') {
        throw 'missing exact integration build tag'
    }
    if (-not (Select-String -LiteralPath $path -CaseSensitive -Quiet -Pattern '^func TestArticleRecoveryFixtures\(t \*testing\.T\) \{$')) {
        throw 'missing exact TestArticleRecoveryFixtures symbol'
    }
    ```
  - 唯一专用命令：`go test -tags=integration ./test/integration/recovery -run '^TestArticleRecoveryFixtures$' -count=1`
  - 专用 race：`go test -race -tags=integration ./test/integration/recovery -run '^TestArticleRecoveryFixtures$' -count=1`
  - scoped vet：`go vet -tags=integration ./test/integration/recovery`
  - 按 Manifest 固定顺序执行七项：`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`
  - 当前状态：Lead 已在最终四文件同一 revision 核证符号前置 Gate、唯一专用普通/race、package 目录目标测试、scoped vet、gofmt、`go mod verify`、全仓 vet/build、contract、race 与 all 均 exit 0，且无 `[no tests to run]`；历史假绿结果不作为关闭证据，现有 `TestRecoveryFixtures` 仍不能替代目标 `TestArticleRecoveryFixtures`。
- **依赖**：requires `project-bootstrap-B0.1`、`article-management-S4.1`、`article-management-S4.6`
- **并行**：[P]；前置满足后仅修改本卡文件，可与无文件交叉且无依赖的任务并行。
- **认领状态**：completed / `Lead Gate PASS @2026-10-25`。Planning Review R02 `.spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r02.md`（SHA256 `A12062129944D8978D92A9732F255F7F1332DE7324AEA9BD3CF7185A7BC3693B`）确认 `ARTICLE-B7-PLAN-R01-F001` CLOSED、Planning Review Gate PASS；只读 G0 READY 与后续 Lead START 的历史保留。Gauss R01 `.spec/wip/article-management/review-code-s4.5-r01.md`（SHA256 `44DE54697DFEA8964E62A9AE2B8196C24E3233A44E177E04410F331AB64E5FF8`）为 `CHANGES REQUIRED / FAIL`（3 MEDIUM + 1 LOW）；Aristotle the 2nd 在原四文件边界返工且实际仅修改 Go 测试文件；Gauss R02 `.spec/wip/article-management/review-code-s4.5-r02.md`（SHA256 `49340355047BFFF8A3F02A531AE20E9D163F65D6167E3190C1BC5FE5BC0E1F48`）确认 F001～F004 全部 CLOSED、无新 HIGH/MEDIUM、Code Review Gate PASS；Lead 同 revision Gate PASS后关闭本卡。
- **阻塞注记**：无。固定四文件边界已完成并释放；第四文件只拥有 strict parse、cross-reference、Ready20 canonical/evidence digest 重算、inventory 固定合成引用 `sha256:bbbb` 逐字一致、负例 mutation 与 runbook 自动检查，未定义或重算 production inventory digest payload/canonical/摘要算法，未出现第五文件、生产代码、migration、公共合同或 design/stories 扩权。
- **本任务关闭条件**：已满足。四文件仅使用合成 inventory/备份/删除样本；Ready20 canonical/evidence digest 与固定合成 `sha256:bbbb` 边界、零测试防假绿、独立 Review 和 Lead Gate均已核证。
- **关闭证据与限制**：最终四文件 SHA256 为 runbook=`41ECB0CF593511119A1BBC910C3B707D094282D1D06875134F531D58F6DD4A09`、inventory=`1355094A5A6751E6E479F4769C9D6E41C993F2A054D23ACF948D3F7E5C4E9B8C`、fixtures=`E86F32A1731E04F3B1590A87282F71F4332B488F2B6040E74AAF7F5876CF7D81`、Go test=`88FD408A466CAF208E780112F82B6EB82BCCDB58D5CB1E848C07B52DFDC3C4F6`。本关闭只覆盖 synthetic fixture/runbook 与机械验证，不证明真实 inventory、真实旧备份恢复、retirement/cleanup 授权、灾备、Host、部署、E2E或 production readiness。
- **后续验证关联**：article-management-S4.4
- **设计/测试来源**：article design r14、test-plan r12 §7/§9.2；AM-Q8/Q13

- [x] article-management-S4.6 接入 Article production composition、路由、Journal provider 与删除 worker 开始@2026-10-23 完成@2026-10-24

- **目标**：在唯一 production composition/router 内接入 Article 配置、持久化 Journal provider、三条 DELETE、后台 deletion worker 与 lifecycle，使 S4.1 已建核心具备局部可自动化的真实 server 接线。
- **类型**：开发
- **owner**：Article Backend-Dev Avicenna `01a16ab8-96a8-7bf3-b2e8-2cf7753dd434`（唯一实现 owner）；非作者 Reviewer Faraday `01a16940-6120-7021-bd06-eb0089956486` 独立评审，Lead 现场核命令、diff、所有权与未测范围。
- **文件**（精确且唯一 13 个）：
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
  - `cmd/han-server/lifecycle.go（修改）`
  - `cmd/han-server/lifecycle_test.go（修改）`
  - `internal/article/deletion_worker.go（新增）`
  - `internal/article/deletion_worker_test.go（新增）`
  - `internal/article/deletion_finalizer.go（修改）`
  - `internal/article/deletion_journal.go（修改）`
  - `internal/article/deletion_test.go（修改）`
  - `internal/article/repository.go（修改）`
  - `internal/article/repository_integration_test.go（修改）`
  - `internal/article/deletion_filelock_flock.go（新增；linux/amd64 等受支持 Unix build tags 下提供可靠排他 flock）`
  - `internal/article/deletion_filelock_unsupported.go（新增；windows/amd64 等 unsupported 平台保持可编译并 fail closed）`
- **验收命令**（workdir：仓库根目录 `.`；实施者、Reviewer 与 Lead 已按各自职责取得当前固定修订证据）：
  - `gofmt -l cmd/han-server/main.go cmd/han-server/main_test.go cmd/han-server/lifecycle.go cmd/han-server/lifecycle_test.go internal/article/deletion_worker.go internal/article/deletion_worker_test.go internal/article/deletion_finalizer.go internal/article/deletion_journal.go internal/article/deletion_test.go internal/article/repository.go internal/article/repository_integration_test.go internal/article/deletion_filelock_flock.go internal/article/deletion_filelock_unsupported.go`
  - `go test ./internal/article ./cmd/han-server -count=1`
  - `go test -race ./internal/article ./cmd/han-server -count=1`
  - `go vet ./internal/article ./cmd/han-server`
  - `go build ./cmd/han-server`
  - `powershell -NoProfile -Command '$ErrorActionPreference = "Stop"; $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("han-article-s46-compile-" + [guid]::NewGuid().ToString("N")); try { New-Item -ItemType Directory -Path $tmp | Out-Null; $env:CGO_ENABLED = "0"; $env:GOOS = "linux"; $env:GOARCH = "amd64"; $linuxOut = Join-Path $tmp "article-linux-amd64.test"; & go test -c -o $linuxOut ./internal/article; $code = $LASTEXITCODE; if ($code -ne 0) { throw "linux/amd64 compile-only gate failed: exit $code" }; $env:GOOS = "windows"; $windowsOut = Join-Path $tmp "article-windows-amd64.test.exe"; & go test -c -o $windowsOut ./internal/article; $code = $LASTEXITCODE; if ($code -ne 0) { throw "windows/amd64 compile-only gate failed: exit $code" } } finally { if (Test-Path -LiteralPath $tmp) { Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction Stop } }'`
    - compile-only 子进程固定 `CGO_ENABLED=0`，以唯一临时目录分别编译 linux/amd64 flock 分支与 windows/amd64 unsupported 分支；不执行生成物，任一失败非零，`finally` 清理输出，不污染父环境且不访问 DSN。
  - `go test -tags=integration ./internal/article -run '^$' -count=1`
    - 仅编译 integration-tag owner 文件，不执行真实 PostgreSQL 用例，不冒充 S4.3 或真实数据库验证。
  - `go mod verify`
  - `git diff --check -- cmd/han-server/main.go cmd/han-server/main_test.go cmd/han-server/lifecycle.go cmd/han-server/lifecycle_test.go internal/article/deletion_worker.go internal/article/deletion_worker_test.go internal/article/deletion_finalizer.go internal/article/deletion_journal.go internal/article/deletion_test.go internal/article/repository.go internal/article/repository_integration_test.go internal/article/deletion_filelock_flock.go internal/article/deletion_filelock_unsupported.go`
- **依赖**：requires `identity-auth-S5.2`、`article-management-S4.1`
- **并行**：本卡已关闭并释放共享 `cmd/han-server` 与 article 删除文件的独占窗口；B6B/S8.1、S10.1～S10.4 与 B6D/S11.1 保持关闭。S10.1 classification repair 五文件窗口已释放；S10.5 尚未启动。
- **阻塞注记**：无。本卡未引入 migration、新第三方依赖、第二 composition/router、`internal/recovery` 私有锁原语或公共合同变化，也未越出精确 13 文件。
- **本任务关闭条件**：仅在精确 13 文件内完成 A-WIRE-CFG-01～06、A-WIRE-JRN-01～07 的组件/fake 与所有平台编译闭环、A-WIRE-RTR-01～06、A-WIRE-WKR-01～10 的确定性部分和 A-WIRE-LIFE-01～08 的局部自动化；linux/amd64 flock 与 windows/amd64 unsupported 两条 build-tag 分支 compile-only、integration-tag compile-only 及其他上述命令全部 PASS，非作者代码 Review PASS，Lead §10 核证通过。真实 PostgreSQL、受支持 Unix 持久卷上的真实 `flock`/file/directory fsync/崩溃/跨进程竞争、真实 Host、部署、signal/exit E2E、容量与 production readiness 继续由 S4.3/S4.5/S1.4 等独立闭环，不作为本卡关闭前置，也不得声称已通过。
- **关闭证据与限制**：13 文件最终 SHA256 与 Faraday r03 固定对象一致；实施者两轮返工后，Lead 的定向/普通/race/vet/build、linux/amd64 与 windows/amd64 compile-only、integration-tag compile-only、`gofmt`、`go mod verify`、scoped `git diff --check`、暂存区与残留生成物核对全部 PASS。Faraday r03 `.spec/wip/article-management/review-code-s4.6-r03.md`（blob `3dcc43eb076b09f61a770dc87604ef3f69e50464`，SHA256 `007C2D9B04F1DC9381C574A668FB8C513F4B34D2B8224808F0512B623573E7E1`）确认 F001～F004 全部 CLOSED、无新增 finding、`Code Review Gate: PASS`。本关闭不包含真实 PostgreSQL、受支持 Unix 持久卷上的真实 `flock`/file/directory fsync/崩溃/跨进程竞争、真实 Host、部署、signal/exit E2E、容量或 production readiness。
- **后续验证关联**：article-management-S4.3、article-management-S4.5、article-management-S1.4
- **设计/测试来源**：article design r10、stories r07 production wiring 后继切片、test-plan r08 §4.10.2/§9.1 与 A-WIRE-CFG/JRN/RTR/WKR/LIFE；Rawls r04（SHA256 `832ED2262DBF443CA3A172C88E5A904A206296771CA309A44EE68D111909F6EA`）确认固定 r08 Test-plan Gate PASS；Hypatia r02（SHA256 `5625061C2963AF1CF29C44D2E1DD15A53FA4F911E5F8CD424BE459681755639E`）确认 `ARTICLE-S4.6-PLAN-R01-F001/F002` CLOSED、Planning Review Gate PASS；Faraday r02（SHA256 `A51DA3D4C6EE031DE121310D0332DD228268EE28D1F23F5963951064E982CDEF`）确认 `B6A-G4-R01-F001` CLOSED、无新增 finding、`B6A G4-batch Gate: PASS`。r06 PASS 与 r07 FAIL 仅作历史链追溯；上述 PASS 不代表本卡实施、代码 Review 或真实环境验证完成。

- [x] article-management-S5.1 实现管理员删除他人普通文章 开始@2026-10-27 完成@2026-10-27

- **目标**：实现管理员删除他人普通文章。
- **类型**：开发
- **R 对账**：需改；`handler.go:448` 的 `DeleteOrdinary` 已共享作者/管理员入口，`command_service.go` 的 `DeletePermanent` 已委托唯一协调器，`deletion_coordinator.go:226`～233 已允许 admin；`deletion_test.go` 已覆盖普通管理员部分。design §3.3.6 要求的 `metadata.authority=administrator` 尚未进入 `deletion_finalizer.go:546` 的唯一完成审计，且缺本卡独立专项验证；复用既有实现，不重建授权或删除协议。
- **认领与批次**：复用 B9A 原任务关联；Backend-Dev Tesla，Agent ID `01a17b91-6d78-7b60-8d2e-0163dafc8205`，已完成本卡四文件范围内实施并停止写入，Lead 已完成关闭核证，写窗口释放。
- **文件**：
  - `internal/article/command_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/admin_delete_test.go（新增）`
  - `internal/article/deletion_finalizer.go（修改）`
- **文件所有权**：本卡精确四文件；finalizer 从已关闭 S4.1 顺序接管，依据 Lead 本轮确认无其他并行 writer。原三文件无必要不改；与共享 command/handler/finalizer 的任务串行。保持原设计、API、schema 和依赖；不修改 coordinator、model、repository、migration、cmd 或配置，不新增第二审计写入者。必须修改第五文件或发现未知写者时停止受影响写入并交 Lead。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run '^TestAdminDeleteOrdinaryArticle' -count=1 -v`
  - `go test -race ./internal/article -run '^TestAdminDeleteOrdinaryArticle' -count=1`
  - 当前状态：2026-10-27 Tesla 在专项测试中先取得 RED：管理员审计只有 article_version/delete_kind，缺少 authority=administrator；在唯一 finalizer 补字段后 GREEN。专项 5 个测试、17 个子例、专项 race、article 包普通/race/vet 均 PASS。Lead 现场执行专项 `-count=1 -v` 与 Manifest 七项固定顺序（gofmt 无输出、mod verify、vet、build、contract、race、unit）全部 PASS；非作者 Reviewer Godel 独立复跑专项 PASS。原贴 S1.6 PG PASS 不属于本卡证据，未作沿用。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S3.1`、`article-management-S4.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：已解除。原三文件不足的边界缺口由本轮四文件授权解除；实际仅修改 finalizer 并新增专项测试，command_service/handler 未作本卡修改。普通 DELETE 接线复用 S4.6，不修改 main。非作者 Reviewer Godel `01a17b95-5469-7313-acd2-75081eac0b8b` 的 `.spec/wip/article-management/review-code-s5.1-r01.md` 两轮独立 Code Review Gate PASS、0 finding，Lead 现场核当前内容、四文件哈希、分支/HEAD、暂存区为空及项目检查后关闭。
- **固定版本与限制**：command_service.go SHA256 `B61ABD12ADD7AF43C7D677A89C097F44FFBB818A85D2D674CD218A81340DB6E4`；handler.go `B2E3F68995AEC9FA63C5E9F04EF0CCC458A4262045279D7A7DD794C2DD25DD00`；deletion_finalizer.go `E914B290DBCB59D24116ECBBD4592AD171038FCEA25081739AF4A6CFB17FD12C`；admin_delete_test.go `3ADEB5C7BC7E64239C8574BDD8CB9A41060E04289F043C1C0B3D3DA2311905CA`。作者未采集修改前哈希，共享树代码未跟踪，不能提供字节级前后全树无越界证明；按实际内容、交接、固定后哈希与独立评审取证，不伪造历史 diff。专项存储/Journal 使用既有 stub，未验证真实 PG、持久卷、部署/E2E。目标作者 suspended 由调用链静态核对；旧备份缺少 operation 时无原 actor role，重放不推测管理员 authority，保留既有恢复合同限制。未提交或发布。
- **本任务关闭条件**：管理员仅能从普通文章入口删除 active/published/clear/有作者文章；违规/无归属必须走专用入口，且不能编辑他人；强 If-Match、权限/状态拒绝、pending 重试与双完成才 204 均有专项证据；唯一 finalizer 完成审计包含 `authority=administrator` 且无正文，失败保持原删除协议；专项、race、适用 Manifest 检查、独立 Review 与 Lead 同 revision 范围核对通过。后续真实 E2E 独立闭环，不借本卡外推通过。
- **后续验证关联**：article-management-S1.5
- **设计/测试来源**：article Story 5；test A-MODR-01～02

- [x] article-management-S6.1 实现 canonical 无归属化端口 开始@2026-10-23 完成@2026-10-23

- **目标**：实现 canonical 无归属化端口。
- **类型**：开发（article 共享接缝，Story 6 所有）
- **实施责任**：Backend-Dev Hooke `01a16893-be5d-7632-8afa-adf77210a91c`；原非作者 Reviewer add-reviewer；Lead 关闭@2026-10-23。
- **文件**：
  - `internal/article/article_port.go（新增）`
  - `internal/article/command_service.go（修改）`
  - `internal/article/orphan_canonical_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run "TestOrphanArticlesCanonical"`
  - 当前状态：Lead 已在三个最终文件固定 SHA256 上取得本卡定向/包级测试、定向 race、`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...` 与 `go test ./...` 全部 PASS，三文件 scoped diff-check PASS，暂存区为空。非作者复审 `.spec/wip/article-management/review-code-s6.1-r02.md`（SHA256 `8C60F2F3EA83926DEFD8E9C8B9B149FA2C1E119E4C7A767832E1BFA616C480E1`）为 PASS，`ARTICLE-S6.1-R01-F001` CLOSED、无新 finding。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.2`、`article-management-S1.1`、`article-management-S1.2`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：无
- **本任务关闭条件**：零/多/违规/已处置/deletion_pending 矩阵幂等；只写 article-owned 状态与 Audit，不回退已处置文章；独立 Review 通过。
- **后续验证关联**：identity-auth-S9.1、identity-auth-S1.5
- **设计/测试来源**：article Story 6 共享 canonical 端口；planned-not-executed article test-plan r04（SHA256 `5A1FE258792CF281B27DE2B9E51131E1C46E738F8B3589A184ACFF383A9DDF3A`）的 A-ORP-01～02 与 S1.3/S6.1 局部测试责任分界；设计复审 `.spec/wip/article-management/review-design-r09-am-q1-r03.md`（SHA256 `C1F380D83D3B00B6B5FBE3CF03B0052B8D3D9B66727333EA472BE49A84FC5AD2`）PASS。
- **关闭证据与限制**：最终三文件 SHA256 分别为 `article_port.go`=`215A224CF416D81FA217F7524EDEEA671A25AC1110BA0147CC1D1BD6315CE044`、`command_service.go`=`880A5960D0B7451567EBBFDCD9CA176513C53C5DED57FD044DB78949469289D7`、`orphan_canonical_test.go`=`9F7CFD3424450D74C30E1F039D4E0154C20BA03B715EE9539DBDDF82678D5D70`。本卡关闭只覆盖 article-owned production persistence 接缝与局部原子性合同；真实 PostgreSQL 锁/回滚、identity production composition、跨模块 canonical 删除、浏览器、部署与 E2E 仍由后续任务验证。

- [x] article-management-S6.2 实现无归属文章公开、转移与永久删除 开始@2026-10-27 完成@2026-10-27

- **目标**：完整交付 Story 6 的无归属列表/详情、公开、转移与永久删除，以及 r16 同事务完成事实核验和 retained_email 条件清理；保留既有 mode/kind/schema/API。
- **类型**：开发（R 对账复用原卡；三片串行实施，不新增卡或 Story）
- **R 对账**：复用原 Story 6 卡完成基线缺口，不新增任务/Story/requires。已实现 Identity lifecycle、article 私有完成证明与同 tx provider、公开/转移、两来源 finalizer 清理、两个 GET 与唯一 production root/router；复用 canonical/Notice/Journal/删除基础。真实 PG 暴露的新 T1 微秒精度与无 actor absent replay 审计问题均已闭环，不重开其他已关闭卡。
- **文件**（最终精确 13 个；三片与定向 HIGH 返工均已交付、独立评审和 Lead 核证，写窗口已释放）：
  1. `internal/identity/article_disposition.go（新增；切片①）`
  2. `internal/identity/article_disposition_test.go（新增；切片①）`
  3. `internal/article/command_service.go（修改；切片②）`
  4. `internal/article/repository.go（修改；切片②）`
  5. `internal/article/deletion_finalizer.go（修改；切片②）`
  6. `internal/article/deletion_test.go（修改；切片②）`
  7. `internal/article/orphan_disposition_test.go（新增；切片②，切片③顺序接管补 GET 回归）`
  8. `internal/article/query_service.go（修改；切片③）`
  9. `internal/article/handler.go（修改；切片③）`
  10. `cmd/han-server/main.go（修改；切片③）`
  11. `cmd/han-server/main_test.go（修改；切片③）`
  12. `test/integration/identity/orphan_disposition_test.go（新增；切片③，integration tag）`
  13. `internal/article/deletion_coordinator.go（修改；③真实PG RED揭示的T1时间精度修复）`
- **当前执行责任与所有权**：Turing `01a17bee-3f62-7be0-90cb-73942b865bf1` 完成最终实现与返工并停写；PG初稿作者James断线后已明确关闭，后续原文件由Turing接管，未沿用未知结果。独立非作者Avicenna `01a17bdc-3af4-7b80-bed0-284885adc328` 完成完整joint R02，HIGH CLOSED、0新finding、PASS并停写。Lead实核代码/差异/哈希/测试/分支与空暂存区后关闭，无活动代码writer；不冒充人工签字或部署验收。
- **切片验收与移交**（以下保留原 START 执行合同；三片实际完成结果以当前验证状态、R02和关闭证据为准，命令workdir为仓库根）：
  - ① 实现 LockTransferParties/LockDeletionCandidate、固定 article verifier 协作与 Identity 条件清理/释放审计；`go test ./internal/identity -run '^TestArticleDisposition' -count=1` 及同范围 `-race`。局部覆盖资格、成对空值/单边漂移、引用绑定、幂等与审计失败。先保留失败证据；局部测试、独立 Review 与 Lead 核证后方可移交②；stub 仅证明局部合同，生产 provider/真实 SQL 留③核实。
  - ② 完成 article-owned verifier/处置事务、NOWAIT 锁和同快照计数、finalizer 两来源分支、数据库权威微秒 lease/fresh clock；finalizer repository 接口与 `deletion_test.go` stub 同片适配，不能留下不可编译中间态。`go test ./internal/article -run 'TestOrphanDisposition|TestDeletion' -count=1` 及同范围 `-race`；回归现役 replay/worker 消费者。保留可识别 55P03，实际 worker 分类必须 transient 而非 ErrDeletionInProgress no-op，可在新增局部测试文件承载；worker 源/测试文件不默认增入。局部测试、独立 Review 与 Lead 核证后方可移交③。
  - ③ 完成两个 orphan GET、publish/transfer 与既有 DELETE 的生产管理员保护、唯一 composition 固定注入及路由回归；GET 断言在顺序接管的 orphan_disposition_test.go 内承载，避免扩大成员查询 stub 接口。`go test ./internal/article -run '^TestOrphanDisposition' -count=1`、`go test ./cmd/han-server -count=1`，各自同范围 `-race`；`go test -tags=integration ./test/integration/identity -run '^TestOrphanDispositionPostgres$' -count=1 -v` 与 `go test -race -tags=integration ./test/integration/identity -run '^TestOrphanDispositionPostgres$' -count=1`。随后按 Manifest 七项固定顺序检查，独立代码/整体评审及 Lead 同 revision 核证。
- **真实 PG 必要场景与载体**：第12文件（integration tag，TestOrphanDispositionPostgres）已覆盖test-plan r14 §4.5.1十组。最小构造既有schemaHarness、复用完整0001～0009/migration读取/begin，每连接绑定已提交隔离schema；不调用自动DROP的newSchemaHarness、不改共享harness。成功对照执行真实provider/Identity/Repository/audit/canonical/worker SQL；PG DTO adapter不替代root factory，root接线另有组件测试和源码评审。NOWAIT/55P03、同快照计数、引用与tx生命周期、Notice快照/零行漂移、原子失败/commit unknown、N001/F001/N002均有真实证据；不以mock、lock_timeout或skip替代。
- **执行环境**：已运行真实PostgreSQL 18，User-scope DSN安全注入进程，未输出凭据/邮箱/正文。隔离schema及清单保留，事务/连接/rows/goroutine按合同退出，无schema或项目文件清理。Windows实盘FileJournal正确fail closed；支持Host的fsync/部署E2E留后续范围。
- **验证状态**：最终完整PG普通/race各10组、181 pass节点、0fail/skip，133.508s/145.989s；作者Article普通/race/vet/gofmt通过，两个既有Windows inventory能力skip不属于PG矩阵。Lead亲跑原失败叶子PASS（3节点0skip）、解析完整原始日志/hash，并在修复后按序完成Manifest七项全部PASS。Reviewer独立局部3顶层/11节点通过，完整PG全文/消费者/13文件joint复审PASS。普通go test不覆盖integration tag；历史RED/FAIL保留。固定hash与运行边界见[Lead验证记录](../../../../.spec/wip/article-management/verification-s6.2.md)。
- **切片①交付与②移交（2026-10-27）**：原非作者 Avicenna `01a17bdc-3af4-7b80-bed0-284885adc328` R01 发现 MEDIUM `AM-S62-S1-R01-F001`：强制 Article 前像存在会拒绝有可信 orphan operation 来源的 absent 幂等分支。原作者在原两文件补失败/通过证据并修复；R02 `.spec/wip/article-management/review-code-s6.2-slice1-r02.md`（SHA256 `429762F6886C7F4C4A6E28ED548E93E9B438B566E8B3FF5704C958D5F643D17B`）确认 F001 CLOSED、0 新 finding、slice1CodeReviewGate PASS，R01 保留。最终 article_disposition.go SHA256 `EB053611CE9346A9A51C0485DD9AE1003EA99C57AF0A3842A4621D67ECE04593`，article_disposition_test.go `A92E8C4D65C76361094B472F35BEDEAB77F15C2FC5E0ECA54854BCDDC640EDF4`。Lead 现场核实际全文、双哈希、分支/HEAD/空暂存区与测试、起始 667 文件对账后接纳局部交付。仅 START 切片②：command_service.go、repository.go、deletion_finalizer.go、deletion_test.go、orphan_disposition_test.go；Turing 唯一写入，①冻结，③不写。后续 root 必须以固定 factory/bound verifier 核 Article 私有前像，不能重包 public DTO 充当证明；未提交、部署或关闭父任务。
- **切片②交付与③移交（2026-10-27，当前）**：作者 RED 复现过期 lease 仍 DELETE（nil error/deletes=1），修复后 Article 全包 ordinary/race/vet 通过；Lead 专项 TestOrphanDisposition 14 顶层/51 节点、修改后 Manifest 七项按序全部 PASS。Avicenna `.spec/wip/article-management/review-code-s6.2-slice2-r01.md` SHA256 `44E228B4305E0E07FC7AD4F028BD8A85155BECD8AEA0AC2A166D8B09F3C1E5C3`：两轮零 finding、slice2CodeReviewGate 局部 PASS。Lead 现场读报告、真实 SQL/source diff、哈希、分支与空暂存区，并与起始 667 文件对账，无删除或越界。固定 command_service/repository/deletion_finalizer/deletion_test/orphan_disposition_test SHA256 依次为 `585BB51D19B3FCD7BFDC9C84465D2BB5D24F41BD8285553FC20BC10B161B4D68`、`E0C2430A981A5F90E84F197159158CC4A218A4FB399C74E90C203B19AC0FE078`、`12F9D775F115732226908C0440D5F30D6FAF37A17CC19FC2A2CE147216F869C4`、`6D04CD35F0049E64FFEC6954B53EF0AB75256BF97C156E331BB5B019E87EA9C6`、`D6EC42338586933CB141550563A3050A1A38012F897420821C5E6BC0E855685E`。③采用建议：HAN_ARTICLE_TRANSFER_NOTICE_MAX_ATTEMPTS 为独立必填十进制正 bigint，不设业务默认值；配置/保护/adapter 测试在原 main_test 承载。PG 包仅映射 DTO，成功事实仍由 native provider/SQL 证明，不声称运行了 package main；root 单测另核真实 production factory。尚无真实 PG/③完成结论，父任务保持进行中。
- **③隔离运行采用差异**：既有 newSchemaHarness 自动注册 DROP SCHEMA CASCADE，不符合当前未授权清理条件，故不直接调用。允许新 integration 文件内构造同一 schemaHarness 类型，复用既有完整 migration 列表/读取和 begin，只新增最小创建并提交隔离 schema 的入口，不注册删除；记录本轮实际 schema 名单并保留。所有事务退出、goroutine join、连接关闭；没有项目文件或 schema 清理授权。新 transactor 首查询前只 Begin/SET LOCAL search_path，不调用会先读 pg_backend_pid 的旧 schemaTransactor。此采用差异只在原任务/Plan承接，不改冻结 design/testplan 或共享 harness，不扩第13文件。
- **当前写权细分与精度返工（2026-10-27）**：Turing 明确停写并释放唯一 integration 文件，交接 SHA256 `2E4B9A90776EAC812A87C7573537515BE70533578D31DBEFBACCB25C37D4EAB7`；真实PG两次 RED 为 DeletePermanent journal_corrupt，DB与tombstone精确时间不等、UTC微秒相等，残留operation=journaled、Article pending、零完成审计。两次PG进程退出，无遗留事务/连接/goroutine；保留 `han_s62_3cff71eb4b1f09dd`、`han_s62_40ea3b9a7261f60a`。原Reviewer只读独立核同因：T1未截微秒，pgx截微秒，Journal保留纳秒，claim前Equal及重试Append不能自愈。Lead据用户持续采用建议授权，仅扩原卡第13文件，在创建operation时UTC.Truncate(Microsecond)，不改Journal格式/旧记录、schema、API或DAG；原四生产/测试冻结文件不重开。此当前采用替代先前12文件上限，需第14文件再报Lead。Turing仅持五接口/接线/局部测试路径加coordinator；James `01a17c2a-3f74-7001-a180-ebd56a9648a4` 仅持integration文件，二者不交叉。PG新增正常删除/故障后重试验证；旧已落盘纳秒Journal兼容/恢复对账未证明，不覆盖、不宣称Ready。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S8.1`、`identity-auth-S9.1`、`article-management-S4.1`、`article-management-S6.1`
- **PG HIGH返工START（2026-10-27）**：原独立Reviewer `.spec/wip/article-management/review-code-s6.2-joint-r01.md` SHA256 `4F53F6603E9F4411463992AED49A707AF64388BB4650F8509F6FB4F1C28D331B` 确认 `AM-S62-JR01-F001` HIGH OPEN、joint FAIL/CHANGES_REQUIRED；本次有界确认未冒充完整joint通过，历史R01保留。Lead实核后仅重开原13文件中的 `internal/article/deletion_finalizer.go`、`internal/article/orphan_disposition_test.go`、`test/integration/identity/orphan_disposition_test.go`，Turing唯一writer；其余10文件冻结。按既有design恢复系统审计合同修复无actor重建操作；不猜actor/source、不改kind/source/Journal/API/schema/DAG。先保存局部RED，再系统审计及幂等GREEN、原真实PG失败子例、完整PG普通/race和新修订必要检查；原Reviewer R02针对F001及完整13文件/十组joint复审后，Lead再核现场结果与范围，父卡此前不关闭。
- **③接口交付与真实PG失败闭环（当前，2026-10-27）**：Turing 六文件停写，Avicenna `.spec/wip/article-management/review-code-s6.2-slice3-root-r01.md` SHA256 `2D7E0B86C179C2C88F600CBFABED2ED73F08FD477FEF61D7381B359D296B85F8` 两轮零 finding、局部 PASS；Lead现场读报告/代码与hash，专项 Article22顶层/72节点、cmd11顶层/67节点、0skip。James stream disconnected，恢复后明确 close_agent、wait not_found；Lead核无go/identity.test进程及PG文件SHA256 `E21A2A4BCA4389BFAE466DF0BEC90EE8976E6CA6248D515D6268998CE9CDF9DC` 未变，未沿用其未知测试结果。PG文件后续仅交Turing，目前只读，接口片仍冻结。Lead实际全矩阵143.92秒：9/10组、178节点PASS、0skip；唯一leaf `08_N001ContactsAndReplay/absent_source_free_replay_does_not_guess` FAIL，SQLSTATE23514/audit_events_check，operation=finalizing、actor/source均空、零清理/完成审计。根因候选为无actor重建operation仍硬写ActorUser；design r16 §3.4.5第11条已有 article.deletion.replay 系统审计合同。当前交原Reviewer独立定finding，尚未返工或复审；同revision Manifest七项虽PASS，不覆盖PG失败，不关闭父卡。真实JSON日志保留于本轮temp证据根 pg-lead-normal-d1d97c04e78c4882a08b995e93857d19.jsonl，含163个保留schema名单；未清理。旧纳秒Journal恢复、Windows不支持生产FileJournal与部署E2E仍按原边界保留。
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：本卡无剩余阻塞，原设计HIGH、Identity absent分支MEDIUM与joint HIGH AM-S62-JR01-F001均按各自R02闭环；历史失败保留。独立完整joint PASS、0新finding，Lead接纳并关闭。draft不改accepted，不补人工签字/checkpoint。SMTP、真实部署/E2E、旧纳秒Journal兼容和旧备份恢复readiness仍在各自原后续Gate，不由本卡声明完成。
- **停止条件**：最终13实现路径已释放。后续需第14路径（含worker/query_test/共享harness）、改mode/kind/schema/API/requires/DAG、第二tx/pool代替当前tx、替换固定verifier/新增authority、未知writer或零命中/新增skip，先交Lead核影响和实际授权；不得借本卡关闭发布恢复Ready。
- **本任务关闭条件**：三片全部实现并各自局部验证/独立评审移交；五个 orphan endpoint 及真实 production verifier/Identity/finalizer 接线可核，只有 held/source orphan 可处置，violating 不直接公开、转移保留违规；真实 PG 十组、适用 Manifest 检查、独立代码/整体评审与 Lead 同 revision diff/所有权/证据核证均满足后，才可关闭原 S6.2。①或②完成不提前关闭整体；部署后 E2E、SMTP 投递、容量/灾备不以局部 PASS 代替，也不倒置为本卡关闭前置。
- **后续验证关联**：identity-auth-S9.2、article-management-S1.5、article-management-S7.1
- **设计/测试来源**：article Story 6；design r16 §3.3.7/§3.4.4/§3.4.5（SHA256 `CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC`）；test-plan r14 A-ORP-03～07、§4.5.1/§9.4（SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C`）；Avicenna R02。不修改稳定 design/test-plan；本卡/Plan 为本次采用候选路径和责任的唯一新增承接记录。

- **关闭证据与移交（2026-10-27）**：原非作者 `.spec/wip/article-management/review-code-s6.2-joint-r02.md` SHA256 `15ABA346F5C3024A43697BCCF47A83CD25E3CB3FB495EB168F5B7781F85A165F` 确认 AM-S62-JR01-F001 CLOSED、S6.2JointCodeReviewGate PASS、0新finding。Lead已读全文，核报告13行代码hash与实际文件一致；三返工hash为finalizer `546C69B7CC363453F6A4E866D4E3B1B68D9E6B611463159EDE69ACEC0D9B8750`、局部测试 `301A40FED868BFAB85D80C84888399DAAAF3B22E8CBBFBD7C754FC3B6E5265DE`、PG测试 `7B38993E04D342F424C11C4D9E136DD3F06967D7E10A939C1103186F9B289D12`，其余10文件不变。Lead核固定main/HEAD/空暂存区、起始667文件对账，无删除或代码越界；完整PG普通/race、原leaf实测及新revision Manifest均通过，原始日志、保留schema清单和验证记录落仓内 `.spec/wip/article-management/evidence-s6.2/` 与 `verification-s6.2.md`。作者/Reviewer停写，Lead依据用户持续采用建议授权接纳，统计仅由checkbox复算。未提交、部署、清理或宣称真实Host/SMTP/E2E完成；后续通知任务仍以原卡登记START为准。

- [x] article-management-S7.1 实现转移通知 recipient lease 与 best-effort dispatcher 开始@2026-10-27 完成@2026-10-27

- **目标**：消费已提交双 recipient 快照，完成短事务 claim/结果保存、事务外 best-effort SMTP 与唯一 production worker 接线。
- **类型**：开发
- **最终交接（2026-10-27）**：Lead 已核最终九源码哈希、实际差异、原始定向普通/race、修复后 Manifest R02 七项顺序全 PASS 与全树范围，接收本卡局部交付并关闭。原非作者 Tesla 的 `.spec/wip/article-management/review-code-s7.1-joint-r02.md` SHA256 `0C160DF795F72B9CB1DE6C84D09317D0E71BF83C80C9E5C02924B67D513269EC` 确认 9fileJointGate/Root4CodeGate PASS、Joint F001 CLOSED、0 新 finding，Core 原 F001/F002 保持 CLOSED。验证记录与最终九文件索引在 `.spec/wip/article-management/verification-s7.1.md`、`evidence-s7.1/lead-fixed-source-after-joint-r02.json`；权威 Root R02 日志使用 `fixed-*`，中间失败与旧报告保留。全部作者及 Reviewer 已停写，源码写窗口释放；以下 R 对账、启动/并行调度及待审描述保留当时来源，不覆盖本最终结果。S7.2 真实 PG/TLS/crash 与部署 E2E 独立，不冒充本卡已执行；未提交、部署或清理。
- **R 对账（2026-10-27）**：Lead 按用户持续“采用建议并继续”授权采纳下列预检结论；只复用原 S7.1/S7.2，不新增 task，不代表设计人工 accepted、实施认领或 START。

  | 范围 | 标签与现状证据 | 原任务承接 |
  | --- | --- | --- |
  | Story 7 通知消费链 | 待做：`command_service.go:1144` 已创建双 pending/max_attempts 快照，`model.go:153` 与 `0003_article.sql:152` 已有状态/lease；dispatcher、专用单测及 article 集成测试目录尚无 | S7.1 实现与局部测试；S7.2 独立专项 |
  | 既有 SMTP 与 root 接缝 | 需改：`smtp.go:126–156` 四拒绝点尚无结构化 Stage；`main.go:276` runtime 及 `lifecycle.go:29,60–103` 仅含两 worker、首次 Reconcile 后才消费 signal，结果容量为4 | S7.1 原卡内兼容扩展；不重开已关闭任务 |

- **文件**（共九文件）：
  - `internal/article/notice_dispatcher.go（新增）`
  - `internal/article/repository.go（修改）`
  - `internal/article/notice_dispatcher_test.go（新增）`
  - `internal/mailer/smtp.go（修改）`
  - `internal/mailer/smtp_test.go（修改）`
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
  - `cmd/han-server/lifecycle.go（修改）`
  - `cmd/han-server/lifecycle_test.go（修改）`
- **只读消费者/边界**：复用 model、command_service、privacy.Store/ProcessingGate、Mailer 接口、audit.Store、identity 消费者及既有迁移；不修改 S6.2、schema、HTTP 契约或旧外部 Gate。九文件仍不足、契约变化或未知 writer 时停止受影响写入并交 Lead。
- **验收命令**（workdir：仓库根目录 `.`；局部实际结果见下方交接证据；root/九文件最终验收尚未完成，获执行授权后逐条执行）：
  - `go test ./internal/article -run '^(TestTransferNotice|TestNoticeDispatcher)' -count=1 -json`
  - `go test ./internal/mailer -run '^TestSMTPMailer' -count=1 -json`
  - `go test ./internal/identity -run '^TestPasswordReset(ExplicitRejection|UnclassifiedSMTPProtocolErrors|RejectedEvidenceFailure)' -count=1 -json`
  - `go test ./cmd/han-server -run '^(TestNoticeConfig|TestNoticeProductionWiring|TestNoticeLifecycle|TestRunLifecycle|TestLifecycleExitCode)' -count=1 -json`
  - `go test -race ./internal/article ./internal/mailer ./cmd/han-server -run '^(TestTransferNotice|TestNoticeDispatcher|TestSMTPMailer|TestNoticeConfig|TestNoticeProductionWiring|TestNoticeLifecycle|TestRunLifecycle|TestLifecycleExitCode)' -count=1 -json`
  - `go test -race ./internal/identity -run '^TestPasswordReset(ExplicitRejection|UnclassifiedSMTPProtocolErrors|RejectedEvidenceFailure)' -count=1 -json`
- **命中与项目检查 Gate**：先核九文件、真实顶层符号及 r15 §4.6/§9.5 必需子例；新 TestNoticeConfig/ProductionWiring/Lifecycle 与四个首次取消时序须逐项命中。逐包/顶层/必需 leaf 核 JSON run/pass；零用例、缺 leaf、skip、`[no tests to run]` 均不通过，旧 TestRunLifecycle 命中不能代替新增通知用例。最终由 Lead 在九文件稳定后按 Manifest 七项顺序单独执行 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`；格式检查须无输出，其余命令及适用测试须成功，不能替代定向命中或 integration 专项。
- **历史边界**：身份 PG 与旧 Manifest PASS 仅为历史，不证明通知新增消费者或本卡通过；不清理原证据。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`identity-auth-S5.1`、`article-management-S6.2`；四项均已关闭，requires 不变。
- **批次/所有权/并行**：唯一归属原 B9A；S7.1 既有 START/开始日期及九文件总上限不变。SMTP2 Franklin `01a17cba-38cd-7433-b36e-4adfb1cf7d05` 已停写保护其 `internal/mailer/smtp.go`、`smtp_test.go`；Article3 Nash `01a17cba-3bab-7ff2-817c-1611df62a19a` 已停写保护 `internal/article/notice_dispatcher.go`、`repository.go`、`notice_dispatcher_test.go`，后续仅凭具体 finding 授权在原三文件返工。root4 唯一 writer 为 Lorentz `01a17cbd-1bdc-71f1-94c0-b0ab7ea558ea`，仅 `cmd/han-server/main.go`、`main_test.go`、`lifecycle.go`、`lifecycle_test.go`；其 G0 已只读核对实际 API 匹配，Lead 正式授权本次 root 交接，Planner 落盘/停写并返回 hash、Lead 现场核差异/scope 后才由 Lead 发实际代码写入信号。root 启动前置为 SMTP2+Article3 实际 API 冻结、实现/局部自测交付、共享 writer 释放，Lead 均已核证；允许 root 与进行中的 CoreReview、经具体 finding 授权且不改冻结 API 的 core 修复并行，2/3/4 文件写集互斥。冻结 API 要变化、跨写集或未知 writer 时暂停受影响 root 推进并交 Lead；不新增 task、业务 requires/DAG 或结构决策。
- **阻塞注记**：本卡阻塞已解除；Core R02 及九文件 Joint R02 均通过，原三项代码 finding 关闭、无新增，最终 Manifest R02 全 PASS。Lead 已按本卡条件核证关闭；后续真实集成和部署边界见 S7.2 与最终验证记录。
- **本任务关闭条件**：所有 finding 必须完成修复及独立复审，九文件稳定后取得最终 joint Review 与 Lead 顺序执行 Manifest 的结果；CoreReview 未 PASS 不影响本轮限定 root 调度，但不得跳过父卡关闭门禁。九文件实现及必要局部失败证据/普通/race/兼容回归、Manifest 检查、独立代码/整体 Review 与 Lead 同 revision 的 diff/命中/所有权核证齐全；两收件人五态、同 tx Gate、DB 微秒/完整 owner fencing、耗尽维护/原子审计/commit unknown、阶段化 SMTP 与安全模板、真实 factory 非 nil 及三 worker 生命周期符合固定设计。F001 四时序须证明停止前置于后继 claim/Send/Start/Serve，BeginShutdown 绑定唯一共享截止、至多一次 fenced 保存与必要回读、活动初次调用及 Wait/Done 实际退出后再关依赖，错误优先序/退出码与五结果收集不退化。本卡满足自身条件可关闭；S7.2、部署后真实 E2E 与整体验收不反向成为本卡关闭前置。
- **后续验证关联**：article-management-S1.5、article-management-S7.2；真实部署 E2E 由原 article-management-S1.4 等后续任务承接。
- **设计/测试来源**：Story 7；固定 design r18 §3.3.8/§3.4.8/§6.1/§8.5，SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`；独立 Design R02 `.spec/wip/article-management/review-design-r18-notice-r02.md`，SHA256 `115289B8216BD5CB0AD49CB9B82B748E84555FD58C307AAFA272213F387B13E7`，DesignReviewGate PASS、`AM-DES-R17-NOTICE-R01-F001` 设计层 CLOSED、0 新 finding。固定 test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`，§4.6/§7.6/§9.5，A-NOT-01～09 及细化、r18 §6.1 F001 四时序（A-NOT-09f～09h）；独立 TestPlan R01 `.spec/wip/article-management/review-test-plan-r15-notice-r01.md`，SHA256 `29BC5E15647A2BB0C853A59B205080DE4114FAD2134F8D82F4FDD7111474E1EE`，TestPlanGate PASS、0 finding，仅为规划充分性评审；r15 本身为 planned-not-executed，不能替代下方局部实际证据或最终九文件验证。r15 内较早的待复审注记以本次固定 Design R02/TestPlan R01 为准，不改写 test-plan 或伪造人的 accepted。

- **安全证据载体**：实际执行日志先保管于 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672`，不复制 secret；后续仓内保管由 Lead 安排。本 TaskPlanner 本轮仅登记/只读复核证据引用，未运行或生成测试证据。

- **root 顺序修订与历史来源（2026-10-27）**：原规划要求 SMTP2/Article3 均局部独立 Review 交付后才移交 root；Lead 本轮依据用户持续采用建议与继续授权，按已冻结 API、局部实现/自测交付和 writer 释放，把 root 启动前置收窄为上述真实接缝条件。原 Review 串行条件仅保留为历史来源，现行按本段交接，不改变父任务全部 finding 修复复审、joint Review/Manifest 与九文件稳定的关闭条件，不改 r18/r15/已得报告 hash。
- **局部交接证据（Lead 提供，Planner 只读核 hash）**：SMTP2 `.spec/wip/article-management/review-code-s7.1-smtp-r01.md` SHA256 `4762EFC4ED1F312835B70A8D4B8C3BE2BB4C73982DB069755D414FEC1BD7D018`：独立 SMTP2CodeReviewGate PASS、0 finding，仅两文件局部。Article3 固定 `notice_dispatcher.go` / `repository.go` / `notice_dispatcher_test.go` SHA256 依次 `8F08F7E3A7430E81859D8B6B8F73A686729AC9BE8E16A6282BD6457A3BCBE503` / `80529712F7D62ACD4A5A976621FF1E7AB11069D11CDE2D73B5050CDCA58758F8` / `D662ACCF349AFACCF8A56833E74042F7ED32426FE9CA7917AFAE12532DB8E0F1`；Lead 核最终专项普通/race 各 33 top/90 nodes、0 fail/skip，vet/build 退出0。Tesla CoreReview 进行中，未有 PASS 报告，不把自测或 SMTP 局部 PASS 外推为 Article3/root/父卡通过。

- [x] article-management-S7.2 验证通知双 worker、SMTP/TLS 与崩溃恢复 [P] 开始@2026-10-27 完成@2026-10-27

- **最终关闭/责任（2026-10-27）**：原作者 Ohm 的固定 R02 普通/race 各3顶层、99 leaf、112 run/pass节点，0 fail/skip；两项真实 mutant 均由新增断言检出，当前实现通过。原非作者 Tesla [R02](../../../../.spec/wip/article-management/review-code-s7.2-r02.md) SHA256 `D4563D87D2C5C62CCEA5DF077DB5DB00DC4E8A923F98692CB33CA077E1EBC680` 确认 `AM-S72-R01-F001/F002` 均 CLOSED、0 新 finding、CodeReviewGate PASS。Lead 读取实际修复diff/完整mutant/原始JSON，核十二文件同hash、最终 Manifest R03 七项顺序全PASS、真实PG回读/retained schema、status/所有权/固定main和HEAD后按本卡条件关闭。作者与Reviewer均明确停写；三文件独占窗口释放，九个S7.1文件未变。未暂存、提交、部署、清理或补人的accepted/checkpoint。完整证据、格式检查中间失败修正链和未验边界见 [Lead记录](../../../../.spec/wip/article-management/verification-s7.2.md)。

- **目标**：以新通知消费者的真实 PostgreSQL 多连接、本地 TLS SMTP 多会话和子进程崩溃验证 DB/发送/审计符合性。
- **类型**：专项集成/安全测试
- **历史 START/责任（2026-10-27）**：Lead 在 S7.1 本卡关闭及固定输入核证后，沿用户持续采用建议并继续的授权，正式交 Test-Engineer Ohm `01a17c83-60f4-7803-b0cc-9b19c0b81fa3` 唯一写下列三文件并执行本地专项；不改九个生产实现文件。登记当时尚未运行，由 Lead 随后发送实际代码/运行信号；当时不预填专项 PASS，最终结果见上方关闭证据。
- **文件**（共三文件；harness、合成 CA/fixture、child 入口均内承，不新增第四文件）：
  - `test/integration/article/notice_dispatcher_test.go（新增）`
  - `test/integration/article/notice_smtp_test.go（新增）`
  - `test/integration/article/notice_crash_test.go（新增）`
- **验收命令**（workdir：仓库根目录 `.`；最终执行增加显式 `-timeout=12m`，原始命令与计数见关闭证据）：
  - `go test -tags=integration ./test/integration/article -run '^TestNotice(DispatcherPostgres|SMTP|Crash)' -count=1 -json`
  - `go test -race -tags=integration ./test/integration/article -run '^TestNotice(DispatcherPostgres|SMTP|Crash)' -count=1 -json`
- **命中与项目检查 Gate**：先核三文件 integration tag、`TestNoticeDispatcherPostgres`/`TestNoticeSMTP`/`TestNoticeCrash` 三个顶层及 r15 必需 leaf；按 A-NOT-01～09 及细化逐项核 JSON run/pass、DB 状态/SMTP 记录/审计，TLS 两安全模式及负例、耗尽 retry_wait、最后 claim 崩溃均不得遗漏或 skip。零用例、缺 leaf、skip、`[no tests to run]` 不能通过；子进程须有阶段握手、实际退出及父测试断言，helper 退出码不等于用例通过。随后按 Manifest 七项顺序单独执行 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`；普通全包测试不覆盖 integration tag，不能替代本专项。
- **环境/资源边界**：User-scope `HAN_TEST_POSTGRES_DSN` 由实际 owner 安全注入测试进程，不打印值；先核目标为既有 loopback PG18.6，连接可用不等于测试通过。采用独立且预先提交的 schema/完整 migration 与合成数据，三文件内承 retained harness，不调用自动 DROP SCHEMA CASCADE 的旧 identity harness；退出须 rollback/join/关闭连接，保留 schema/证据。SMTP 仅 loopback、不转发、不发外部邮件；本轮运行窗口交 Ohm，原始日志及保留 schema 清单先存本轮 temp 根的 `s72` 子目录，由 Lead 核后保管仓内。
- **历史边界**：身份 PG 与旧 Manifest PASS 仅为历史，不证明通知专项；不清理原证据。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S1.4`、`article-management-S7.1`；三项均已关闭，requires 不变。
- **批次/所有权/并行**：唯一归属原 B10；本轮三文件唯一 writer/运行 owner 为 Ohm。S7.1 九文件已停写保护；如真实新消费者发现产品缺陷，保留失败证据交 Lead/原作者，不在测试中放宽预期或越界修改生产源码。[P] 仅在前置满足、环境隔离和无文件/契约冲突时成立；包级执行消费稳定文件，独立 Reviewer 待实际固定测试交付后由 Lead 指派。
- **阻塞注记**：已解除。固定 r18/r15、Design/TestPlan Review 与 S7.1 Joint R02 前置有效；真实PG/TLS/child专项、R01两项整改、同Reviewer R02及Lead最终核证已完成。外部 SMTP/生产 Host/部署 E2E 不在本轮运行范围，不由本卡关闭。
- **R01历史核证/整改（2026-10-27）**：Ohm 的 R01 完整普通/race 各3顶层、99 leaf、112 run/pass节点、0 fail/skip；Lead 实际解析日志、核十二文件固定哈希、Manifest七项及真实PG回读。独立 Tesla [R01](../../../../.spec/wip/article-management/review-code-s7.2-r01.md) SHA256 `5546FA045B6AEB5E89428367CA246562EE987DADE41AD7C8F2F16B391E0D8034` 当时为 CHANGES REQUIRED：MEDIUM `AM-S72-R01-F001` 安全负例零 envelope 协议断言、`AM-S72-R01-F002` accepted 后取消耐久结果/审计断言当时均 OPEN。独立回读确认实现结果正确，不重开 S7.1。Lead 交原作者在同三文件内整改、保留九文件哈希保护；整改期间维持进行中，新固定版本通过后由R02关闭两项，不把 R01 自测通过外推为修订版通过。
- **本任务关闭条件**：A-NOT-01～09 及 r15 细化的双 worker、DB 时间/lease/ABA、Gate 竞态、耗尽维护、原子审计/commit unknown、SMTP 阶段/安全传输、accepted 后及最后 attempt 崩溃均有普通/race 三层证据；不发外部邮件；资源退出、独立 Review 和 Lead 同 revision 命中/diff/所有权核证齐全。S7.1 的 F001 组件时序证据按固定 r18/r15 关联，不以 fake 代替本卡真实边界。部署后真实 E2E 独立，不倒挂本卡或 S7.1。
- **后续验证关联**：identity-auth-S3.4、article-management-S2.3、article-management-S1.4
- **设计/测试来源**：复用 S7.1 上列固定 r18、Design R02 及 test-plan r15 的精确 SHA256；核心 A-NOT-01～09 与 §4.6 细化、§7.6 三层证据/安全载体、§9.5 命中门禁。DesignReviewGate PASS 仅限设计，TestPlanGate PASS 仅限规划；r15 的 planned-not-executed 为形成计划时状态，实际专项结果见本卡关闭证据，未引用 r14 历史 PASS。

- [x] article-management-S8.1 实现违规判定、解除与永久删除治理 开始@2026-10-27 完成@2026-10-27

- **目标**：实现违规判定、解除与永久删除治理。
- **类型**：开发
- **批次与实施责任**：复用 Plan B9B，本轮先于 S6.2 串行推进。Backend-Dev Noether `01a17bb6-cc2c-70a1-8cc6-0954d04ca887` 已完成七文件实施与返工并停止写入；原非作者 Reviewer Ramanujan `01a17bc1-6827-7240-8a93-cdf49c9f5ddc` R02 确认 `AM-S81-R01-F001` CLOSED、0 新增 finding、CodeReviewGate PASS。Lead 现场核同 revision 成果、专项、Manifest 七项、真实 diff 与全范围哈希后关闭，写窗口释放；R01 FAIL 历史保留。
- **文件**：
  - `internal/article/command_service.go（修改）`
  - `internal/article/handler.go（修改）`
  - `internal/article/policy.go（修改）`
  - `internal/article/moderation_test.go（新增）`
  - `cmd/han-server/main.go（修改）`
  - `cmd/han-server/main_test.go（修改）`
  - `internal/article/deletion_coordinator.go（修改）`
- **文件所有权与实施边界**：本卡精确七文件，固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。保留已有未提交内容；与共享 command/handler/policy/main 的任务串行，S6.2 不启动。沿现有 `Repository.UpdateArticleContent` 的文件落点，在 command_service 内增加只更新违规字段的生产持久化方法；判定/解除、版本检查与 Audit 同事务，restore 保留作者、publication、source 和删除状态，响应复用真实 `GetAuthorStates` 与既有 `AdminProjection`。main 唯一 composition owner 注入真实依赖并注册 violation/restore 两条 POST，复用既有 violating DELETE；不新增第二 router、Repository/审计权威或 stub-only 生产接缝。原设计、API、schema、依赖不变，已足够的既有文件无必要不改。2026-10-27 Lead 仅追加 deletion_coordinator.go，顺序接管已关闭 S4.1 的共享删除授权；Lead 确认无其他代码 writer，S4.1 不重开。新增范围仅修复合法 anonymous/published/source=NULL/violating/active 文章被 `DeleteViolating` 的 AuthorID=nil 条件拒绝的问题；保留 held/source 无归属专用入口、原六文件范围和共享 Journal/finalizer，不新建删除路径。
- **验收命令**（workdir：仓库根目录 `.`）：
  - `go test ./internal/article -run '^TestModeration' -count=1 -v`
  - `go test -race ./internal/article -run '^TestModeration' -count=1`
  - `go test ./cmd/han-server -count=1`
  - `go test -race ./cmd/han-server -count=1`
  - 确认目标测试非零命中，再按 Manifest 固定顺序单独执行：`gofmt -l .`（无输出）、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。
  - 当前状态：最终七文件修订专项普通/race、article/server 包级普通/race、局部 vet/build 全部 PASS。Lead 现场 `go test ./internal/article ./cmd/han-server -run '^TestModeration' -count=1 -json` PASS，article 70/server 33 个通过节点（含顶层与子例，非 103 个独立顶层测试）；修复后 Manifest 七项按固定顺序全部 PASS，最后 unit 已回收 exit 0。原 Reviewer 独立执行匿名/共享删除、三入口授权与 If-Match 定向回归 PASS，R02 关闭 F001。初始 NOT RUN、修复前 PASS 与 R01 FAIL 分别保留历史意义，原误贴的 identity S1.6 PG PASS 不作本卡证据。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S1.2`、`article-management-S3.1`、`article-management-S4.1`
- **并行**：串行；与相邻任务共享实现文件、契约或有明确先后依赖。
- **阻塞注记**：已解除。R01 `.spec/wip/article-management/review-code-s8.1-r01.md` 曾以 HIGH `AM-S81-R01-F001` 阻断：匿名文章判违后因作者为空无法删除。原作者新增 RED（匿名 clear→mark→新 ETag 违规 DELETE 返回 409 article.invalid_state）后最小修复共享授权，合法 published/source=NULL/violating 文章可进入同一删除协议，held/source 仍走专用入口。R02 `.spec/wip/article-management/review-code-s8.1-r02.md`（SHA256 `4D42EF13A62C978E6DAC8BE68101A22C60B92321307B1FE7C2C5CCC963C651C7`）确认 F001 CLOSED、0 新增 finding、PASS，未新增 API/schema/依赖或删除/审计权威。
- **关闭证据与限制**：最终七文件 SHA256 见 R02 固定修订表；新增 moderation_test.go 为 `8094598605B8C8EB5AA5940A70BD2E3B03D3C3C3CEB3C0A81BAEB1C451933779`，共享 deletion_coordinator.go 为 `307BEEDE0F62878D5FA39D81A69529792E3155735D776C565AC3E4AFD46D1C70`，其余五文件相对 R01 无漂移。Lead 读取真实前文 diff，并用本轮起始 664 文件哈希复核仅七代码文件、Tasks/Plan 及授权 Review 报告变化，无其他漂移或删除；main/HEAD 固定，暂存区为空，未提交或发布。作者状态读取使用当前 pgx.Tx，未持事务另取池连接。治理成功与删除串联为真实 Repository/audit SQL、handler/coordinator/finalizer 配合模拟事务/Journal，未验证真实 PostgreSQL 并发/实际回滚/commit ack loss、生产成功请求 DB 链、多视图数据库集成或 E2E，不外推 production readiness。
- **本任务关闭条件**：先取得非法类别、重复状态、错入口、并发 ETag 和审计失败的失败证据，再完成实现及局部验证；restore 只清违规事实，其他隐藏约束保留。生产 router 回归证明两条 exact POST 到真实 handler，正确方法保持 Trace 最外层及 Origin→Session→CSRF→Principal/管理员授权，错误方法 JSON 405/Allow、未知路径 JSON 404，既有 Identity/Article 路由不回归。删除复用既有 coordinator/finalizer、强 If-Match 与双完成才 204；审计失败不提交物理删除事务。针对 `AM-S81-R01-F001` 在既有 moderation_test 中先保留合法匿名 clear→mark→新强 ETag 违规 DELETE 的失败证据，再验证修复后共享 Journal/finalizer、`article.violation.delete` 审计与双完成 204；回归 stale 412、clear 409、pending 隐藏、held/source 路径隔离及审计失败回滚。原 Reviewer 须针对该 finding、新七文件修订/hash/真实 diff 复审，确认 CLOSED 后方可核本卡关闭条件。专用普通/race、router 回归、适用 Manifest 检查及独立 Review/复审通过，Lead 在同 revision 核实际 diff、哈希、所有权及证据后按 §10 关闭。Lead 已报告保存 664 文件起始 hash 与五个现存目标原文，作为本轮真实前后 diff/越界复核基线，不沿用上一卡未采修改前 hash 的限制。真实 PostgreSQL/多视图集成、部署和 E2E 未执行则明确保留未验范围，不以本卡局部 PASS 外推；后续验证关联不变。
- **后续验证关联**：article-management-S1.5、article-management-S2.2
- **设计/测试来源**：article Story 8；design §3.3.9、§3.4.1；test A-MODR-03～06；2026-10-27 Lead 先采用六文件有界对账建议，后依据非作者 R01 `AM-S81-R01-F001` 仅追加 deletion_coordinator.go 作七文件最小修复。


- [x] article-management-S8.2 实现普通管理员文章列表与详情并完成局部/真实PG验收 开始@2026-10-28 完成@2026-11-05
- **2026-11-05 S8.2 关闭（`[-]`→`[x]`，完成@2026-11-05；Lead Gate + 非作者独立复审）**：按 `task-execution-workflow.md` §10.3 逐项核对。①成果与验收：九文件实现、U/C/R/P 局部与真实 PG 验证、联合 CodeReview R02 PASS、七项项目检查 exit0 均有既有记录；②Case18：F001/F005/F006 已关，F002 按 ADR-0034、F007 按 ADR-0035 范围决策关闭（非验证通过），F003/F004 由非作者 Copernicus `01a1abdf-853e-7593-a6a5-330a1972dd43` 复审 CLOSED；③Lead 现场核：评审 [report](../../../../.spec/wip/article-management/evidence-s8.2/case18-reconcile-r01/review-r01/report.md) 24,267 B / `74D4054F…`、[verdict](../../../../.spec/wip/article-management/evidence-s8.2/case18-reconcile-r01/review-r01/verdict.md) 8,900 B / `63F2B1DA…`，复审独立复跑 4 测 `EXIT=0`（12.292s），源码未变（`1C679377…/37431`、`6D5373CA…/173317`、`2E29CBFD…/1728`）；④保留边界：production `Enter`/`Install` 仍按 ADR-0030/0031 closed，Journal/HMAC/keyring 为 simulated，F004 仅 store 层，真实部署 E2E/双系统按 ADR-0035 未验证，不宣称 production readiness；⑤未改生产代码、无 Git 写。三态由 75/2/9 更新为 76/1/9，active 仅 `article-management-S1.5`。
- **2026-11-05 Case18 finding 级复审 R01（非作者 Copernicus `01a1abdf-853e-7593-a6a5-330a1972dd43`）：F003/F004 均 CLOSED；Lead Gate 现场核证**：评审产物 [report.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-reconcile-r01/review-r01/report.md) 24,267 B / `74D4054F3D7618D110302C964458768B63977971F2A53BBE36D601564517A08D`、[verdict.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-reconcile-r01/review-r01/verdict.md) 8,900 B / `63F2B1DA0CF21606DB0428BCC15789A47FF841413216BCE89DD1FADC9B3EB8B4`，Lead 实算一致并全文实读。复审独立复跑 4 测 `EXIT=0`（12.292s 墙钟 / 包级 `ok 10.514s`）、`go vet -tags=integration ./cmd/han-server` exit 0、`gofmt -l` 无输出；NOT RUN 项如实登记（`go build ./...`、race/全量、真实部署、真实 keyring/POSIX journal/approved backup）。Lead 复核评审期间源码未变：`recovery_runtime.go 1C679377…/37431`、`main.go 6D5373CA…/173317`、`case18_session_store_integration_test.go 2E29CBFD…/1728`；tasks/Plan/ADR 未被评审改动。保留限制：production `Enter`/`Install` 仍按 ADR-0030/0031 closed，Journal/HMAC/keyring 为 simulated（`windows_unsupported`），F004 仅 store 层；AI 代理评审，非真人签字。至此 F001～F007 全部关闭（F002/F007 经 ADR-0034/0035 范围决策），S8.2 关闭条件中的「非作者 CodeReview 与 finding 复审」已满足；**Lead 尚未执行关闭动作，待用户授权后按 §10 更新 checkbox/三态**。
- **2026-11-05 Case18 真实部署 E2E/双系统按 ADR-0035 范围决策关闭（非验证通过；S8.2 仍 `[-]`）**：项目负责人确认「经决策不验证」。新增 **ADR-0035**（`.add/project/knowledge/decisions/content.md`，`内容` 领域）并同步 `decisions/README.md` 全局索引/领域文件表/场景查询表/产出归属说明。Case18 关闭条件第 5 项（真实部署 E2E 与双系统，评审 LOW `C18-COV-F007`）转为未验证边界：不执行、不以本机模拟或既有单机 PG 证据冒充部署结论，不宣称部署就绪或 production readiness。本决定不改变 `C18-COV-F003`（待原 Reviewer 复审）与 `C18-COV-F004`（待原 Reviewer 复核）的关闭链，也不改变九文件实现、局部/真实 PG 验收、联合 CodeReview R02 与七项项目检查既有证据。S8.2 保持 `[-]`，三态 75/2/9、requires/DAG/写集不变。
- **2026-11-05 C18-COV-F002 按 ADR-0034 范围决策关闭（非验证通过；S8.2 仍 `[-]`）**：项目负责人确认「不需要这项门禁」（测试项目、经决策不验证）。新增 **ADR-0034**（`.add/project/knowledge/decisions/content.md`，`内容` 领域）并同步 `decisions/README.md` 全局索引/领域文件表/场景查询表/产出归属说明。F002 的「`failed`/`expired`/`undetermined` 三状态 + 真实 runtime/PG」验证要求转为未验证边界：实测 admission `Verify` 不读取 `recovery_runs.outcome`（temporary=`internal_ready`；live=`public_ready`；改 outcome 无影响），若确需强制应落在 coordinator `Resume`/start，属新设计切片（见 `case18-reconcile-r01/f002-mapping-g0.md`）。不改 runtime/coordinator/admission/guard 或任何生产代码；`C18-COV-F003` 仍待原 Reviewer 复审、`C18-COV-F004` 待复核，真实部署 E2E 与双系统仍未验证；S8.2 保持 `[-]`，三态 75/2/9 不变。
- **2026-11-05 C18-COV-F002 live 路径第二轮实测（仍 OPEN；试验已删除）**：先 `s420RunNormal` 到 `public_ready`，再把 `recovery_runs.outcome` 改为 `failed`，`Verify` 仍返回 `public_ready/verified`、`err=nil`；启动前注入 `failed` 时 `RunNormal` 也成功。实测结论：admission 的 `Verify` 不读取 `recovery_runs.outcome`，只证明 journal/installation/closure 连续性。故 F002 的「failed/expired/undetermined → 拒绝」若确需强制，应落在 coordinator 的 `Resume`/start 路径，须先由设计确认门禁位置再写测试。本轮两组试验文件均已删除，`go vet -tags=integration ./cmd/han-server` exit 0，仓库内无失败测试。证据 `case18-reconcile-r01/f002-mapping-g0.md`（3069 B）。
- **2026-11-05 C18-COV-F002 准入映射 G0 实测（仍 OPEN，未实施）**：按用户选择把 F002 重新定义为「运行结果 → 准入观测映射」后，实测发现 Case18 fixture（无 Exclusive 端口）的 `Runtime.Verify` 走 temporary 证明路径，返回 `{State:"internal_ready", Reason:"verified"}` 且无 error；把 `recovery_runs.outcome` 改为 `failed` 后结果不变，说明该路径不按 run outcome 拒绝。`admission.go:726` 的 `RecoveryOutcomeFailed → ErrAdmissionBinding` 只在 live/controlled 校验路径生效。故合规测试必须走 live/exclusive 路径（full harness + Exclusive 端口 + 启动后失败注入），本轮试验文件已删除、未留失败测试；F002 保持 OPEN。证据 `case18-reconcile-r01/f002-mapping-g0.md`（1880 B）。F004 回归复跑 PASS 8.672s（`ADD9EB0C…` / 35 B），gofmt 无输出。
- **2026-11-05 C18-COV-F004 当前 revision 真实 SessionStore SQL/撤销证据（待原 Reviewer 复核；S8.2 保持 `[-]`）**：新增 `cmd/han-server/case18_session_store_integration_test.go`（`//go:build integration`；Lead 直接实现，无 Backend-Dev Role 产出），用真实 PG `session.Store` 解析活动会话（principal 用户与 fixture 一致）→ `Revoke` → 同一 token 再解析失败。单跑 PASS 2.45s；与 `TestRecoveryProductionChainComplete`、`TestRecoveryMainBusinessDispatcher`、`TestRecoveryIsolationMatrix` 合并回归 PASS 8.503s（日志 `88A57C7B…` / 35 B）；`gofmt -l` 无输出、`go vet -tags=integration ./cmd/han-server` exit 0。F004 建议交原 Reviewer 复核后关闭，本轮不自行 CLOSED。同轮 G0 确认 `C18-COV-F002` 无法按字面实施：生产 admission 观测仅 `closed`/`unknown`(error)/`public_ready`（`internal/recovery/runtime.go:27-29`），`failed` 运行结果被 `ErrAdmissionBinding` 拒绝（`internal/recovery/admission.go:726`），仓库中不存在 `failed`/`expired`/`undetermined` 三种 admission 状态；需先确定映射并让真实 runtime 进入 started/checking 态，属更大测试切片，本轮未写代码。证据 `case18-reconcile-r01/summary.md`（4694 B）。不改 checkbox/统计/requires/写集。
- **2026-11-05 Case18 复核 r01（S4.20 关闭后；S8.2 保持 `[-]`；Lead 直接执行，无 Role 产出）**：在现役 HEAD `6da8abb`、`recovery_runtime.go` `1C679377…/37431`、`main.go` `6D5373CA…/173317`、真实 PG 18.6 上实际执行三条命令均 exit 0：`TestRecoveryProductionChainComplete` PASS 10.87s（8 子测试，含 resume→replay→journal consume→succeeded→activate/开流、二次 ReadStatus、MarkConsumed、canonical replay、closed/unknown 隔离）、`TestRecoveryMainBusinessDispatcher` PASS 0.30s（8 子测试）、`TestRecoveryIsolationMatrix` PASS 0.19s。逐 finding 事实复核：`C18-COV-F003` 的「旧 `900F31F5…/35832` 绑定」前提已不成立，现役哈希与 S4.20 关闭证据一致，**建议交原 Reviewer 复审后决定降级/关闭，本轮不自行 CLOSED**；`C18-COV-F004` 仍 OPEN（`PublicReadyAdmitsSameSession` 用内存 fixture，无真实 SessionStore SQL/撤销）；`C18-COV-F002` 仍 OPEN（矩阵仅 closed/unknown + fake runtime，缺 failed/expired/undetermined 与真实 PG/双系统）。证据 `evidence-s8.2/case18-reconcile-r01/summary.md`（3346 B）及三份原始日志（`5C560460…`/9746 B、`B84D211E…`/1895 B、`C5392EB0…`/35 B）。整卡保持 `[-]`、`blocked-upstream` 不解除，不改 checkbox/统计/requires/写集；无源码修改、无 Git 写。
- **2026-11-03 Case18 当前 revision 执行证据 r01（只补 F001/F004 事实层；S8.2 仍 `[-]`，`blocked-upstream` 不解除）**：Lead 直接执行（未委派 Role，故本轮无 Role 产出），证据目录 [case18-current-r01](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01/)，`index.json` `609CFBF075078808E1483B9F20E5EDBF5395AB426652EB756054CEF4E36DFCD2` / 6640 B、`report.md` `C54E4EAB1AF5761F2F250CE51A1E0B4EAD63A22AC51B02AC60E99D20F46EEC3C` / 7384 B。**源码绑定**：`main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B、`main_test.go` `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` / 400312 B，与 Case18 覆盖对账 R01 评审基线逐字相同；八文件执行前后 SHA256/字节一致。**运行**（`go1.27.0 windows/amd64`，PostgreSQL 18.6 `127.0.0.1:5433`，DSN 经 User 级 `HAN_TEST_POSTGRES_DSN` 注入、未打印未落盘）：`go test ./cmd/han-server -run '^TestProductionAdminArticleQuery$' -count=1 -v` 与 `-race` 版各 93 PASS 节点、0 fail / 0 skip、`DATA RACE` 0 行；`go test ./internal/article -run '^(TestAdminQuery|TestAdminCursor|TestAdminCursorFiveViewIsolation)$'` 与 `-race` 版各 135 PASS、0 fail；`go build ./...` 与 `go vet ./cmd/han-server ./internal/article` 均 exit 0；`go test -tags=integration ./test/integration/article -run '^TestAdminQueryPostgres$' -count=1 -v` exit 0、99 PASS、0 fail / 0 skip（16 个顶层子测试，真实 SQL→reader→service→AdminHandler；保留 schema `han_admin2_1cc188e4ed204226825e9420363a9bbb`，迁移 0001～0009 提交后执行，未 DROP）；`go test -tags=integration ./test/integration/identity -run '^(TestS16SessionResolveRejectsRevokedAndSuspendedUsers|TestS16UserFirstLockOrderingAndSessionRevocation|TestS33LoginSessionPostgres)$' -count=1 -v` exit 0、6 PASS、0 fail / 0 skip（真实 `SessionStore` SQL、会话撤销与停用拒绝、登录轮换/审计回滚；保留 schema `han_s33_12bc6017e1f14506a10675dc82999702`，`retain_policy=always`）。**事实层推进**：`C18-COV-F001` 所指“当前 revision root/装配未执行（`PExecuted:false`）且绑定旧 `6B45B550…` / `B53DD34D…`”在本条被当前 revision 实测替代；`C18-COV-F004` 的 Session 侧已由运行 8 覆盖，AccessGeneral 侧仍仅接缝证据、未覆盖。**未推进**：`C18-COV-F002` 仍为 `newS419RecoveryRuntime` 假 runtime 与零值 `pgxpool.Pool`，无真实 runtime/PG/双系统；本仓库准入状态词表为 `closed`/`internal_ready`/`public_ready`/`unknown`/`faulted`/`draining`/`stopped` 且仅 `public_ready` 放行，`failed`/`expired`/`undetermined` 属 run/lease 结局词而非准入状态，故不通过扩充假状态标签制造覆盖面；`C18-COV-F003` 的 production `Enter`/`Install` 仍返回 `ErrAdmissionClosed`，无完整恢复→`public_ready`→生产开流证据；`C18-COV-F005` 维持“时点快照语义”保持 open。**边界**：不改 S8.2 九文件写集与既有证据，不改任何 finding 状态（finding 是否 CLOSED 须由非作者独立复审本证据包判定，Lead 不自行改判），不证明部署 E2E、真实 SMTP/approved-provider、双系统接线、完整恢复后开流、真实 Journal/HMAC/keyring、旧备份恢复、容量/P95、production readiness 或人的业务接收；本轮未做 Git 写、部署、文件删除或数据库清理；AI 代理执行与登记（ADR-0028），非真人签字。
- **ADR-0028 适用（2026-10-31）**：按项目级验证授权执行；可执行层仍实际执行留证，真实环境可模拟并标注“未真实验证”，评审/签收可由 AI 代理并记录授权来源，不冒充真人签字。

- **2026-10-31 恢复主线承接说明（不改本卡范围/checkbox）**：Case18 的全局恢复请求隔离与“完整恢复成功后开流”不再由本卡承接，已登记 [`identity-auth-S4.20`](#task-identity-auth-s4-20)（B13H 唯一 root 串行写窗）作为“恢复权威→请求隔离→完整恢复→`public_ready`→开流”的主线实施卡；本卡转为该证据的消费者。本卡仍为 `[-]`：九文件局部实现、`review-code-s8.2-joint-r02.md` 联合独立评审 PASS 与七项 Manifest 证据保持不变；待 S4.20 与独立验证完成后再由 Lead 评估关闭，不自动关闭、不改本卡九文件写集、不改既有 checkbox/日期/requires/关闭记录。**2026-10-31 事实更新**：`identity-auth-S4.20` 已由 Lead 关闭（见该卡 2026-10-31 Lead Gate 记录）；“待 S4.20 与独立验证完成”的前半条件已满足，其余接收/关闭门禁与本卡 `[-]` 状态不变。
- **Lead 当前局部交付核证（2026-10-28）**：原九文件实现与局部/真实PG功能验证已交回；[联合独立 CodeReview R02](../../../../.spec/wip/article-management/review-code-s8.2-joint-r02.md) SHA256 `F1EE8BF2605C3755835E2A1C0F4DFDA354697BD4EEBE52FE8B8A80E0681B7808` 有界PASS、无material finding，Lead完整实读报告并现场核diff/原始结果。U/C/R/P普通及race实际子集和三类行为负控、作者及Reviewer运行/保留schema、174输入/七项Manifest均见[核证记录](../../../../.spec/wip/article-management/verification-s8.2.md)和[最终证据](../../../../.spec/wip/article-management/evidence-s8.2/final-r01/lead-copied-evidence.json)，不复制另一测试状态库。Manifest SHA256 `D333D9E1146C84E157774FD2C32E058E6B5B27C25066A53B51AA83CBA09CF62C` 的7命令均exit0，不冒称全profile/部署或全部业务验收。后文G0/仅Phase1/未授权main或PG/Review待审等为当时历史，本段记本次最新实际交付。
- **整卡接收/关闭仍阻塞**：保持`[-]`；Case18的全局恢复请求隔离、完整恢复/S16/二次核验/MarkConsumed后生产开流尚未交付，Case18整体及S8.2接收/关闭继续BLOCKED（该阻塞已于 2026-11-05 解除，见卡首关闭记录）。局部Go GREEN及有界代码PASS不解除该门禁、不skip或豁免；[恢复主线交接](../../../../.spec/wip/article-management/architecture-admin2-recovery-handoff-r01.md)待实际上游owner/实施合同承接，不擅扩S10.5/新增requires。人的accepted/checkpoint及S1.5、ACL/producer/真实E2E责任不代签、不关闭。**2026-10-31 事实更新**：`identity-auth-S4.20` 已关闭，Case18 的 resume 范围（恢复权威→请求隔离→完整恢复→`public_ready`→开流）主线证据已就位；enter/install 与 recovery secret candidate 仍无承接任务（unknown），20Case 全适用层、人的 accepted/checkpoint、S1.5 整卡、ACL/producer/真实E2E 等门禁仍未满足，本卡保持 `[-]`、不关闭。
- **2026-11-01 Case18 覆盖对账独立评审 R01：CHANGES REQUIRED（S8.2 保持 `[-]`，`blocked-upstream` 不解除）**：非作者独立评审（Avicenna / add-reviewer，actualID `01a19741-0af8-7a23-8c45-7ba24420c702`，只读，唯一写入为评审目录）按 [test-plan r17](../design-workspace/article-management/test-plan.md) `93071AB5…` 的 A-ADM2-18 与 Case18 五分项逐项对账，结论 **CHANGES REQUIRED**。产出 [report.md](../../../../.spec/wip/article-management/evidence-case18-coverage-r01/report.md) `2C0848582BA524F244761A65CAF3D8C08E923B0FC56AB9B6D03CC254216C48F7` / 20718 B、[verdict.md](../../../../.spec/wip/article-management/evidence-case18-coverage-r01/verdict.md) `C75CFC5662E537C0D40F6A3B19DA80C8A84EFA93E8E5942A65DF28E33BD14C97` / 3464 B、commands.log `451017DD791E25688D293CB65D610EC06F295BEEF0053D2D91EB2ABD79F02475` / 6971 B；Lead 已实读并复算三份哈希与字节一致。评审基线：`cmd/han-server/main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B、`cmd/han-server/main_test.go` `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` / 400312 B、`cmd/han-server/recovery_runtime.go` `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` / 37431 B、`cmd/han-server/recovery_isolation_matrix_integration_test.go` `DA02943B9848C50E1960ECCDBF210C3E69D3AA8A64034F9A6321740E519E9115` / 8498 B，Lead 本轮逐项复核未漂移；branch `main`、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 不变，`git status --short` 由 291 增至 292 行且唯一增量为该新目录（`?? .spec/wip/article-management/evidence-case18-coverage-r01/`），评审目录外零改动。**五分项判定**：①真实 factory/共享实例 PARTIAL；②Trace/exactrouter PARTIAL；③Session/AccessGeneral PARTIAL；④全局恢复请求隔离 PARTIAL；⑤完整恢复后生产开流 **NOT COVERED**。**Finding 与处置**：`C18-COV-F001`（HIGH）S8.2 root 证据绑定旧 `main.go 6B45B550…` / `main_test.go B53DD34D…`，与当前 revision 不符且 `PExecuted:false`、`Case18Overall:"BLOCKED upstream"` —— **接受**，不改 S8.2 既有证据与 checkbox；`C18-COV-F002`（HIGH）隔离矩阵只覆盖 `closed`/`unknown`、使用 fake `newS419RecoveryRuntime`，缺 `failed`/`expired`/`undetermined` 与真实 runtime/PG/双系统 —— **接受**；`C18-COV-F003`（HIGH）S4.20 d7 证据绑定旧 `recovery_runtime.go 900F31F5…` / 35832 B，S4.21 明载 production Enter/Install 仍 `ErrAdmissionClosed`、真实 PG S4.20 未重跑 —— **接受**，不重开 S4.20/S4.21 既有结论；`C18-COV-F004`（MEDIUM）Session/AccessGeneral 仅 resolver/repository/handler 接缝，缺当前 revision 真实 `SessionStore` SQL 与会话撤销 —— **接受**；`C18-COV-F005`（MEDIUM）`evidence-s1.5/producer-real-r01/index.json` 的 `artifacts[]` 声明 `report.md` 为 `A9A4ECBB…` / 8023 B、实算 `1B2FB3CF…` / 8267 B —— **裁定“不改、按时点快照语义登记”，保持 open 而非 CLOSED**：该 `index.json` 自带 `generated_at 2026-10-31T23:20:00+08:00`，属生成式快照；其哈希 `F840BC2C…` / 19567 B 被 5 份不可变文档引用（`producer-real-review-r01/report.md:87`、`producer-real-review-r01/commands.log:41`、`review-code-s1.5-f008-producer-r01.md:37`、`s421-claim-baseline/pre-tasks.md:1232`、本文件 `:1232`），改写会使这些文档所引对象在磁盘上消失，代价大于收益；评审后按 `S15-PR-F002` 的消费侧改写使当前值为 `1B2FB3CF…` / 8267 B，权威现值登记于本文件 `:1247` 与 [verification-s1.5.md](../../../../.spec/wip/article-management/verification-s1.5.md) 第 7 行，与 `report.md` 历史指针处置一致；`C18-COV-F006`（LOW）本文件 S4.20 Lead Gate PASS 记录（原 `:2957`；本条插入后为 `:2958`）中 `d7-chain-complete-race-r02.json` 哈希原记 62 位 `72002FB661EF695886F2E7…BCC56`（缺 `84`）—— **已就地更正为 64 位 `72002FB661EF69588684F2E7B4CB239F4D8F96672C3F927F8AB1C248E43BCC56`**，与实物 27055 B 逐一相符，未改任何结论、字节数、日期或统计；`s421-claim-baseline/pre-tasks.md:2955` 为不可变快照，原样保留；`C18-COV-F007`（LOW）无真实部署 E2E/双系统，`sim-closeout-r01/index.json:22` 明示 `NOT RUN/unknown` —— **接受**。**复跑**：评审者仅执行 `go build ./...` EXIT=0 与 `go vet ./cmd/han-server` EXIT=0；任何 `go test` 记 **NOT RUN**（集成测试会创建并保留 schema，超出只读授权），未连接或写入数据库；Lead 本轮未运行 Go/PG/HTTP/E2E。**结论边界**：S8.2 整卡**不具备进入关闭评估的条件**，六项关闭前置（当前 revision root/装配执行证据、当前 revision SessionStore SQL/撤销、全状态真实隔离、当前 runtime 完整恢复+S16+二次核验+MarkConsumed+开流、真实部署 E2E/双系统、证据哈希修正）均未满足；本卡 checkbox/完成日期/写集、`blocked-upstream` 与依赖它的 `article-management-S1.5` `[-]` 十二 requires 均不变；本记录为 AI 代理评审与 Lead 裁定，非真人签字，不代替人的业务接收、Case18 整体验收或部署授权。同步：[verification-s1.5.md](../../../../.spec/wip/article-management/verification-s1.5.md) 第 149 行 unknown 段已补记本评审结论，文件现为 `34BB2EC5FF319138BC5EE086FB91D21A5FFB927C68604F2C97A600318B172DFD` / 35421 B / 159 行、LF、无 BOM（原 `688B144F…` / 35041 B 保留为历史指针）；本记录追加前 tasks.md 为 `87CB31CE7C379DE06AE8F128EBC11D6718E371ED5F8AB79DE1D2538768D071A4` / 813571 B / 3032 行 / CRLF / 无 BOM，三态 `67 completed / 2 active / 8 todo`、S1.5 与 S8.2 的 `[-]`、十二 requires 与 `blocked-upstream` 均未改动。
- **2026-11-01 Case18 隔离矩阵当前 revision 执行证据（只补 `C18-COV-F002` 可离线核实层；S8.2 保持 `[-]`，`blocked-upstream` 不解除）**：针对上条 `C18-COV-F002` 中「当前 revision 是否真有执行证据、路由覆盖面、普通/race 是否干净」这一层，由主会话 Lead 直接执行并落盘（未委派 Role，故本轮无 Role 产出），证据目录 [evidence-case18-matrix-current-r01](../../../../.spec/wip/article-management/evidence-case18-matrix-current-r01/)，含 [report.md](../../../../.spec/wip/article-management/evidence-case18-matrix-current-r01/report.md) `C1521F27441352C04903F707E0613314AB5F54005854DAC70DFD19E59E290D93` / 6999 B 与 [index.json](../../../../.spec/wip/article-management/evidence-case18-matrix-current-r01/index.json) `799048AAD2C712E2264D3F9E38D5CAC9D5AF83F19E586506DC2DEBCA14F170CA` / 8767 B，Lead 已实读并复算一致；执行产物 `run-normal.json` `C8A03EA4B55C728CB7362B43D495F8463DEF7014C07334ED5DAB37E21781297F` / 56460 B、`run-race.json` `74FA5868E359E9F3A23D97996867A37B4F44637D608D5B46395EFD03E06C8913` / 56534 B、`go-vet.log` `CC258D839E50D5E0CF220528399D5C29A6BCC5C3A9EE2616A915627E0363D91D` / 8 B。**执行**（`go version go1.27.0 windows/amd64`，`HAN_TEST_POSTGRES_DSN` 置空，构建标签 `integration`，单轮）：`go build ./...` EXIT=0；`go test -tags=integration -v -run "^(TestRecoveryIsolationMatrix|TestRecoveryMainBusinessDispatcher)$" -count=1 -json ./cmd/han-server` 与其加 `-race` 版本均 EXIT=0、`66 pass / 0 fail / 0 skip`、65 叶子、`DATA RACE` 0 行（包级耗时 0.527s / 1.805s）。叶子构成：`closed` 25、`unknown` 25、`guard_open_reaches_authorization_and_origin` 1、`TestRecoveryMainBusinessDispatcher` 14。**已证**：25 条守卫业务路由在 `closed`/`unknown` 两状态下均返回 503 且 `CodePlatformRecoveryInProgress`、响应 `Trace` 与头一致、无恢复态泄漏，并断言守卫未到达授权/origin/Session/Article 仓储/作者读取（`acquires==25`、`releases==0`）；`/livez` `/readyz` 在守卫之外（200 `alive` / 503 `unavailable`）；`guard_open` 对照以 `public_ready` 得到 201、origin 1 次、授权 `[create]`、acquire/release 1/1，证明观测点非恒假；race 检测器零告警。**未证（关闭门禁不变）**：`failed`/`expired`/`undetermined` 等恢复态未覆盖（只做 `closed`、`unknown` 与 `public_ready` 对照）；仍是 `newS419RecoveryRuntime` 假 runtime 与零值 `pgxpool.Pool`，本轮执行不改变这一点；无真实 PostgreSQL（空 DSN，无 schema/数据/真实锁事务）、无真实双系统（指标/告警/job 接线）、无部署 E2E；`F001`/`F003` 所指 production `Enter`/`Install` 仍为 closed 且未触及；完整恢复之后的生产开流仍 **NOT COVERED**（`F004` 不变）。**结论边界**：本条只把 `F002` 从「疑似未执行」收敛为「已执行但覆盖范围有限」，不构成 PASS、不关闭任何 finding、不解除 `blocked-upstream`，不改本卡 checkbox/完成日期/写集与 `article-management-S1.5` 十二 requires，不重开 S4.13/S4.16/S4.18/S4.19/S4.20/S4.21 既有结论；本轮无独立评审，属 AI 代理执行与记录，非真人签字。同步：[verification-s1.5.md](../../../../.spec/wip/article-management/verification-s1.5.md) 第 149 行 unknown 段已补记，该文件现为 `13F56B3F2D60A7D1653C7E8BF7CBF7F922B45384E6483BDEBD31BCB230C421C5` / 36129 B / 159 行 / LF / 无 BOM（原 `34BB2EC5FF319138BC5EE086FB91D21A5FFB927C68604F2C97A600318B172DFD` / 35421 B 保留为历史指针）；本记录追加前 tasks.md 为 `D40E72129FE8AEDAF884217687D3C692C3213EEE00B4594C1784B9E824769555` / 819372 B / 3033 行 / CRLF / 无 BOM，三态 `67 completed / 2 active / 8 todo` 与两卡 `[-]` 均未改动。
- **写窗口交回**：Dalton和Erdos均实际completed/stopped；本次九源码写窗口释放、无活动写者，任务`[-]`不作文件锁。后续上游改相关源码/合同前协调精确owner、保留Admin2现有接线并复核受影响证据，本修订PASS不自动用于新版本。未stage/commit/push、部署或清理；原失败链/初始前像/全部旧卡保留。

- **2026-11-03 Case18 当前 revision 执行证据 r01 独立评审 R01 + 登记补正（S8.2 仍 `[-]`，`blocked-upstream` 不解除，不进入关闭评估）**：非作者独立评审（Anscombe / add-reviewer，actualID `01a1a0b4-231d-7730-b702-a0aaad930a07`，只读，唯一写入为评审目录）[report.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01-review-r01/report.md) `AAFFA18532814397036C5EF1066345B8B7DE1E6487D8D70776D369C1FEBC192C` / 19588 B、[verdict.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01-review-r01/verdict.md) `AE2BC8D3EE4C6BF509A837C408AE0F62D095168F2EAD6731781DDF5CEE0471DD` / 3082 B、commands.log `71C1619CF44DF6CE12778351E78CBC1758C8FF015247171C61DF549A5DFE0251` / 9985 B，Lead 已实读并复算三份哈希与字节一致。结论 **CHANGES REQUIRED**。**逐项判定**：`C18-COV-F001` **CLOSED（仅 finding 级，不等于 S8.2 或 Case18 关闭）**——当前 revision root/装配普通与 `-race` 各 93 PASS 由评审独立复现，8 源文件绑定一致，旧 `6B45B550…` / `B53DD34D…` 确属 `PExecuted:false` 的另一对象；`C18-COV-F004` Session 侧已覆盖（真实 PostgreSQL + `session.NewStore`，撤销/停用后 `ResolvePrincipal` 返回 `ErrSessionRequired`，6 PASS 独立复现），**整体仍 open**（AccessGeneral 侧未覆盖）；`C18-COV-F002` / `C18-COV-F003` / `C18-COV-F005` **结论不变、保持 open**。**评审独立重跑**与本卡上条记录的 §3 计数逐项一致：root 普通/`-race` 各 93 PASS、UC 普通/`-race` 各 135 PASS、`go build ./...` 与定向 `go vet` exit 0、article integration 99 PASS、identity integration 6 PASS，全部 0 fail / 0 skip；评审并指出 `-race` 日志本身不能自证 race 已启用，独立重跑才是主要证据。**新增两条 finding 与 Lead 现场复核**：`C18-R01-REV-F001`（MEDIUM）保留 schema 少计 29 个——Lead 按两份 run 日志全量提取去重，实为 **28 个 `han_admin2_*` + 3 个 `han_s33_*`**（清单见补正文件），而 `report.md:48/50`、`index.json.retained_schemas` 与本卡上条记录各只登记 1 个；`C18-R01-REV-F002`（LOW）准入状态词表不准——实读 `internal/recovery/runtime.go:27-32` 为 `closed`/`checking`/`public_ready`/`faulted`/`draining`/`stopped` 六态，`internal/platform/recovery_guard.go:59` 仅对 `public_ready` 放行，`internal_ready` 是 observation 字面量（`runtime.go:1015`）、`unknown` 来自测试替身，原报告漏 `checking`；两条均属实。**补正**（只追加、不改原字节）：同目录 [registration-correction-r01.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01/registration-correction-r01.md) `CFDCE04CA654DA813129C9BA0929934EE23C35A7ED5989309E40258BD61EC84C` / 4604 B，列全 28+3 清单并更正状态词表；原 `report.md` `C54E4EAB…` / 7384 B、`index.json` `609CFBF0…` / 6640 B 与 8 份 run 日志字节未动。**另两项复核**：`C18-COV-F006` 的 `d7-chain-complete-race-r02.json` 实算 `72002FB661EF69588684F2E7B4CB239F4D8F96672C3F927F8AB1C248E43BCC56` / 27055 B，与本文件 S4.20 记录一致（登记层面维持已更正，无新增改动）；`C18-COV-F005` 维持 2026-11-01 的“时点快照”裁定——消费侧 `producer-real-r01/report.md` 现为 `1B2FB3CF…` / 8267 B（`S15-PR-F002` 改写）、`index.json` 仍 `F840BC2C…` / 19567 B 未改，本轮不重写该索引，finding 按原裁定保持 open。**历史指针漂移更正**：本文件 2026-11-01 评审记录与矩阵中的“`:1247`”“`:2957`/`:2958`”因后续多次追加已位移，权威位置现以内容定位为准：producer 消费记录（`producer-real-r01` 与 `S15-PR-F002`，本次追加前位于 `:1239`）与 S4.20 Lead Gate/`C18-COV-F006` 记录（本次追加前位于 `:2969`）；行号随后续追加持续漂移，故不再固化行号，以内容与哈希为准。**状态与边界**：S8.2 仍 `[-]`、`blocked-upstream` 不解除；Case18 关闭评估六项前置中第 3～6 项（全状态真实隔离与真实 runtime/PG、当前 runtime 完整恢复+开流、真实部署 E2E/双系统、证据哈希与登记修正）仍未满足，**不具备进入关闭评估条件**；未做 Git 写、部署、删除或数据库清理，未 DROP 任何保留 schema；评审为 AI 代理，非真人签字。本记录追加前 tasks.md 为 `2014B12496C2BEA95D46B4F0DD8C3ECB5CA12F62BD5DDA443BE8F70568FA09FA` / 926224 B / 3340 CRLF / 0 bare LF / 无 BOM。
- **2026-11-03 Case18 执行证据 r01 复审 R02：PASS WITH MINOR，两条新增 finding CLOSED、`C18-COV-F001` 登记为 CLOSED（仅 finding 级；S8.2 仍 `[-]`，不进入关闭评估）**：同一非作者评审（Anscombe / add-reviewer，actualID `01a1a0b4-231d-7730-b702-a0aaad930a07`，只读，唯一写入为复审目录）[report.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01-review-r02/report.md) `C732752A552672FB432A3BA1908CF56B103A9DB1675FC46C767021D3B357B2A9` / 10370 B、[verdict.md](../../../../.spec/wip/article-management/evidence-s8.2/case18-current-r01-review-r02/verdict.md) `BB8E3B41AFA13B0D8D6DA2666DE35920FD3D90F9F645E653B070E31FDAE8A2B7` / 2651 B、commands.log `E8009A4086B4A0E3564164B87E34C018FD02D777CBFE55456EB4A40B4A2E946A` / 6694 B，Lead 已实读并复算三份哈希与字节一致。**判定**：`C18-R01-REV-F001`（MEDIUM）**CLOSED**——评审独立全量提取去重得 28 个 `han_admin2_*` + 3 个 `han_s33_*`，与补正清单逐项一致（漏 0 / 多 0），并认可“显式作废旧登记值 + 同目录只追加更正”足以关闭，无需改写原 `report.md`/`index.json` 本体；`C18-R01-REV-F002`（LOW）**CLOSED**——六态常量、`internal/platform/recovery_guard.go:59` 仅 `public_ready` 放行、`internal_ready` 为 observation 字面量（`runtime.go:1015`）、`unknown` 来自测试替身均复核成立，且 `C18-COV-F002` 仍 open。据此 `C18-COV-F001` **正式登记 CLOSED（仅 finding 级）**；`C18-COV-F004` 整体仍 open，`C18-COV-F002`/`F003`/`F005` 保持 open。**非阻断观察 3 条（本轮不改，登记留痕）**：补正 `:54` 的 session 字段措辞与日志字面（`created_here=true retain_policy=always`）不同但语义一致；补正 `:61` 把 Readiness 503 probe 与受保护请求的 `writeRecoveryHTTPClosed` 混写，核心结论成立；补正文件未登记进 `index.json` 的 artifacts/runs——因原索引“零字节改动”声明需保持，本轮不追加，后续如需消费者指针应另立索引而非改写旧索引。**评审独立核验**：复算 13 个证据文件与 r01 三文件哈希全部一致，原文件“未改动”声明成立；按授权未执行任何 `go test`、未连接或写入 PostgreSQL、未 DROP 或清理 schema、未做 Git 写；`git status --porcelain` 310 行与开工一致，仅新增复审目录 3 文件。**状态与边界**：S8.2 仍 `[-]`、`blocked-upstream` 不解除、Case18 关闭评估条件仍不具备（第 3～6 项未满足）；finding 级 CLOSED 不等于整卡、Case18 或 production readiness；AI 代理评审，非真人签字。本记录追加前 tasks.md 为 `D410A4CC2B47D22379E35FD5B513B8A9565FD157855D999E9B0B2C741686D9B7` / 932878 B / 3342 CRLF / 0 bare LF / 无 BOM。
- **2026-11-04 C18-COV-F005 finding 级关闭登记（仅当前 producer 消费绑定；Lead 已接收）**：按本轮明确授权追加消费 [current r02 index](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r02/index.json)（`E7DF162F0716558369DFA996DB7FE38FAD20E0AF340B4873685538C900A54E56` / 21133 B）与 [hash-manifest](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r02/hash-manifest.json)（`996EE23E23F48F4988DC583EF0C9A4F4CC85C11815BC8C608E3359EB56B901B9` / 10312 B）。非作者 Carver / add-reviewer，实际 ID `01a1a491-533c-7123-a0cb-b0d2e955a6dc`（由 Lead 提供），[独立 report](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r02-review-r01/report.md)（`F5F8CF5B0EDC53AD89C7AC8714FB95DA634A01F55EA751DA225DC2302512C7AF`）及 [verdict](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r02-review-r01/verdict.md)（`053C3CBD3C5F837D9C52C5F01EC7BFA7DF4126E3FC866F18E5446DC873917161`）判 F005 **CLOSED / Review gate PASS（仅此 finding）**、0 新 finding；[Lead reception](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r02-lead-r01.md)（`31792BDC56E309CBD0DF614934856C7B492E99811D48CE74F4D5DF597B99CC99`）记载已实读独立结论、现场核 43/43 manifest 与 8 份旧证据无漂移并正式接收。Planner 仅登记该已决定结果，不代 Reviewer 或 Lead 核证；43/43 为本次追加前的 Lead 接收时点，原 manifest 的 verification_context 绑定追加前42,439 B版本，本轮仅在该文件末尾新增本接收条目、原字节前缀完整保留，新增前后摘要与证明见 `planner/f005-reception/`，不宣称追加后旧 manifest 对该文件整哈希仍匹配。当前入口绑定 report `1B2FB3CF…` / 8267 B；旧 index `F840BC2C…` / 19567 B 与旧 report 声明 `A9A4ECBB…` / 8023 B 保留历史、原像 unknown，不重建、不覆盖、不宣称旧 index 已自洽。上文 F005 OPEN 均保留为当时时点；当前关闭只适用于新增 r02 的消费绑定，不覆盖源码或 runtime。旧普通/race 各 58/0/0 仅为已保存日志的离线复算，10 源 9/10 匹配及 `deletion_replay_source_test.go` 漂移原因/影响/当前运行仍 unknown。`C18-COV-F002/F003/F004` 整体仍 OPEN，S1.5/S8.2 仍 `[-]`、`blocked-upstream` 不解除，Case18 仍阻塞且不进入关闭评估；不改 checkbox/写集/requires/统计，不构成人的签字、生产批准或 E2E 接收。本轮只读核引用并追加元数据，未跑 Go/DB、未改源码/旧证据、未做 Git 写或下级委派。
- **2026-11-04 r03 current 消费接续登记（仅 C18-COV-F005；Lead 已接收）**：r02在活verification由42439 B追加至45600 B后当前为42/43，唯一失配为活verification整文件；原43/43、旧独立评审及接收均保留其旧时点事实，原r02登记和文件不覆盖。当前消费采用 [r03 index](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r03/index.json)（19180 B / `4B3F9979D749279D0EA8DDF0A0BC5A27F9DD32EF10B6E3033586445BFF3738CB`）与 [manifest](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r03/hash-manifest.json)（11293 B / `AB98DBCE882CC6A2C1B445A69FBD7808111878E4390042621A2297C73255D0DD`），硬绑定固定45600 B上下文 [snapshot](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r03/verification-context.snapshot.md)（`7997E1ADF37D82FB663B40B41BD43BE4AF191120E66A60BC7E6C1C25CA1F4AF9`）；活verification仅导航，无整文件硬绑定，允许后续追加。原非作者Carver / add-reviewer `01a1a491-533c-7123-a0cb-b0d2e955a6dc` 的 [独立report](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r03-review-r01/report.md)（`C6F78C972EFC2A84BA15F058F68A2E2300563BA3B2DFB409BC2DEE84D4B2A5FD`）及 [verdict](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r03-review-r01/verdict.md)（`E0C546937D593FDE13AED744F3F634BCE8E10A2A851E61E96F99A244B635572A`）判r03 delta PASS / 0 finding、F005当前绑定CLOSED；[Lead receipt](../../../../.spec/wip/article-management/evidence-s1.5/producer-real-current-r03-lead-r01.md)（2107 B / `DB670CF7E26D740362D3686DE3E764E270DE82D8F677EB57F30D26E53D7EF9DB`）明确已实读并独立实算50/50、硬绑定无liveverification，正式接收。Planner仅追加已决定的finding级结果并核追加后50/50，不代Reviewer或Lead签收。F002/F003/F004整体仍OPEN、S1.5/S8.2仍active、Case18仍blocked；历史运行与源码漂移unknown边界保留，不改checkbox/统计/requires/写集或Plan。S2.4原18 finding性能代码CHANGES REQUIRED保持，不能由本scope宣称S2.4/源码/runtime/性能/E2E PASS；AI独立评审加Lead接收，非真人签字。本轮无Go/DB、源码或旧证据修改、Git写、清理及下级委派。
- **目标/类型**：开发；在唯一生产root接通普通管理员 `GET /api/v1/admin/articles`、`GET /api/v1/admin/articles/{id}`，同一有界单元随实现闭环本地测试与真实PG功能验收；PG不属于部署E2E，不另拆Case/文件任务。
- **当前依据/授权同步（2026-10-28）**：以下为本卡当前增量；后文r20/r09/r16及其Review、F001“待Pauli复审/待Hubble澄清/仅只读G0/未START”等文字保留为各自形成时历史，当前状态按本段固定输入、真实Lead收据及分项门禁解释。原九路径初始前像、旧失败链和待运行命令模板均保留，不据此重判历史或扩大当前执行权。

  | 当前固定输入 | SHA256 / 有界结论 |
  | --- | --- |
  | [design r21](../design-workspace/article-management/design.md) §3.3.9/§8.6 | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA`；纠正Guard前提/责任与证据边界 |
  | [stories r10](../design-workspace/article-management/stories.md) Story8及Story2/3关联 | `492657843DFB4B037C5A8BC31C36A0B08F2233FEFB937A9D39C401FE63307AF9` |
  | [Design R03](../../../../.spec/wip/article-management/review-design-r21-admin-query-r03.md) | `195334B08EB9F106D1ABA2C1BCC5E0D21767295DD5CE33670BA449C1B2F86BF2`；有界PASS/无新增finding，不证明系统Guard交付 |
  | [test-plan r17](../design-workspace/article-management/test-plan.md) §4.3.1/§9.6/§11.1 | `93071AB5E9E68ED976FA376EF52874E7B1A74DEE3FA9860561DD46A596A8DFA2`；Case18五分项及局部/全局证据边界 |
  | [TestPlan R02](../../../../.spec/wip/article-management/review-test-plan-r17-admin-query-r02.md) | `600C9BC2AAECF668D50B9A96090250021B57FDF44AF9965FD6FF2440AB550A3D`；有界PASS/0finding，不授权运行 |
  | [Planning R02](../../../../.spec/wip/article-management/review-tasks-plan-s8.2-r02.md) | `F4174B9A99EF7A4AB94D3CF61EC05806703C4919228DA23F6DD54AE5ADB85D97`；独立CLOSED `AM-S82-PLAN-R01-F001`，仅metadata承接/停止合同PASS，不是Guard/Case/整卡PASS；本次引用同步未新增Review结果 |

- **已发生Phase1/实际授权**：Lead于 `2026-10-28T00:40:17+08:00` 向原唯一Backend Dalton `01a17dcf-f2ed-7dc2-b7f3-6c72cf771448` 发出七内层CODE/LOCAL START，[实际收据](../../../../.spec/wip/article-management/evidence-s8.2/pre-start-r01/lead-inner-code-start-r01.json) SHA256 `899E86F550490F561341EBB03E1A29CEB60E45E9C02522CB5F88B09F567D3863`。仅四article生产文件query_service/handler/model/policy.go及U/C/P三个测试文件，准U/C定向普通/race和相关回归，P仅编译、不运行、不读取DSN；main.go/main_test.go与真实PG仍无START。Dalton正在开发，未最终交回/接收；本次Planner仅记录既有信号，不代发新START。
- **已发生证据的界限**：codec真实行为RED已由Lead核证；Lead本轮交接的UC首轮普通/race局部GREEN仅限当时子集与修订，不是最终源、完整Case/38映射、PG/root或CodeReview通过。只引用[verification-s8.2.md](../../../../.spec/wip/article-management/verification-s8.2.md)及[证据目录](../../../../.spec/wip/article-management/evidence-s8.2/)，不复制另一运行结果台账、不冻结正在开发的七文件或填未来结果。
- **当前局部/全局门禁**：r21/r17已固定澄清：Case18同一 `R:ProductionSharedChain` 的真实factory/共享实例、Trace/exactrouter、Session/AccessGeneral三分项可在Lead未来另给main2局部窗口后独立取证；P真实PG也可由Lead另给LOCALPG窗口，不以全局能力未交付无差别禁止局部实现/验证。当前main2/真实PG仍未授权。通用恢复请求隔离、完整恢复后生产开流两分项仍blocked-upstream；恢复503/零文章下游、完整恢复/S16/二次核验/MarkConsumed后开流及失败继续隔离义务不缩，相关恢复子断言、Case18整体及S8.2整卡接收/关闭继续blocked。局部Go GREEN/spy/账号403/skip/fake503不得冒充整体PASS；九文件不暗补全局authority/开流。七内层API、SQL/数据/错误、20Case/38映射、四top、九路径、六requires与完整验收全部不变。
- **恢复主线承接/继续条件**：[Pascal架构只读咨询原文](../../../../.spec/wip/article-management/architecture-admin2-recovery-handoff-r01.md) SHA256 `22810DC285A3905BC71C996026F909C37DDDDB986B22CCDF0D2C502CA3B0DB5B` 为真实Role返回、Lead保存，非独立Review或实施授权；既有组件/验证卡未完整落实通用HTTP Guard＋Coordinator生产调用＋readiness开放联动的可执行承载，实际归属/owner仍待恢复主线落实，不凭S10.5候选新增requires或扩大其scope。Hubble/Ohm本次有界澄清已固定且各自停写，不再将该澄清写成尚未回答；上游系统承载/真实生产联动仍未交付。恢复主线需固定权威→隔离→开流合同/装配及真实根拒绝/成功开流证据，适用独立评审后由Ohm对齐、Planner仅按实际变化对账、Lead同版核证再决定解除恢复阻塞/整卡关闭；局部main与LOCALPG信号分别另核，不处理未经完整恢复的真实业务流量，人的accepted/checkpoint不代签。
- **固定来源/验收锚**：US-IDENTITY-03.12 / SR-IDENTITY-03、Story8验收6～12；兼容US-ARTICLE-02.1 / SR-ARTICLE-02、US-ARTICLE-03.1 / SR-ARTICLE-03及Story2/3新增验收，支撑US-IDENTITY-03.2删除前读取。只消费 [design r20](../design-workspace/article-management/design.md) §3.3.9/§8.6 SHA256 `6884431B6164140E5593A8253AC20E8EB1C59C1AD71B86558B6D1A8C99BF037A`；[stories r09](../design-workspace/article-management/stories.md) SHA256 `A2BA7746F30B1C9B60DB4337995FC0A0E52DD55BFF984BAA784C20331D80C569`。
- **设计/测试规划Review**：[Design R02](../../../../.spec/wip/article-management/review-design-r20-admin-query-r02.md) SHA256 `5B2E39D0A491B173BB651AA20E966BE40A0E03DFC1645E8D3CFFB0594B9E62B0` PASS、F001仅设计层CLOSED；[test-plan r16](../design-workspace/article-management/test-plan.md) SHA256 `6941C1D7B96F78B341CBB7B88AE74D548CF860CFEAD37D3D92E208548D896E38` §4.3.1/§7.7/§9.6/§11.1；[独立TestPlan R01](../../../../.spec/wip/article-management/review-test-plan-r16-admin-query-r01.md) SHA256 `267CAA23481E31C4ECF3654FAD851772747B5ED6A04DB50823F5E973A978CC1B` 两轮PASS/0finding。该报告覆盖r16正文的待审历史文字；全部20Case仍NOT RUN，不把上游PASS当本卡Planning/CodeReview或运行PASS，人的accepted/checkpoint仍pending。
- **R四标签/证据**：

  | 标签 | 本次真实范围/来源 | 落实责任 |
  | --- | --- | --- |
  | 已建成（只读复用，非新增验收PASS） | query_service.go:56/78/222候选repo/同一service与SQL；handler.go:300/308既有member/author构造；policy.go:190/376现役AdminProjection/原因；model.go:342现有Admin12 | 复用原组件，旧关闭卡不重开；旧结果不证明Admin2 |
  | 待做 | r20两管理员查询方法/Handler与两个生产GET、两个新admin_query_test.go；r16四层20Case未实施/未运行 | 本S8.2五生产/四测试，独立Review后下交S1.5 |
  | 需改 | model.go:363～368/policy.go:392现四cursor view缺admin_hidden；query_service.go:222候选view校验及唯一root须扩支持管理员三view读取 | 仅下列既有五生产文件，保留member/author/orphan及S1.6已接线入口 |
  | 废弃 | 未发现需废弃范围 | 不删端点/旧测试，不新增权威模型或第二router |

- **精确写集（总上限9；五生产+四测试，不授权其他路径）**：

  | 路径 | 最小实现/验证责任 |
  | --- | --- |
  | `internal/article/query_service.go`（改） | 同一ArticleQueryService的ListAdminArticles/GetAdminArticle、typed请求/页；仅扩具体候选repo的view检查，复用SQL/配置/reader |
  | `internal/article/handler.go`（改） | AdminArticleQuerier/NewAdminQueryHandler及两个GET；授权后一次完整RawQuery解析，成功后同map校验；新读错误局部映射 |
  | `internal/article/model.go`（改） | 仅CursorViewAdminHidden；现役AdminArticleView恰12字段，不加字段 |
  | `internal/article/policy.go`（改） | codec支持admin_hidden；复用AdminProjection/hiddenReasons/ETag |
  | `cmd/han-server/main.go`（改） | 唯一factory复用同pool/repository/reader/config/service；exact两个GET、共享安全链、详情GET/DELETE/完整Allow共存 |
  | `internal/article/admin_query_test.go`（新，U） | `TestAdminQuery`：r16全部U:Leaf |
  | `internal/article/policy_test.go`（改，C） | `TestAdminCursor/FiveViewIsolation`：五view成功及20个有向跨view互拒，保留旧codec/投影回归 |
  | `test/integration/article/admin_query_test.go`（新，P） | 首行integration build tag/package articleintegration；`TestAdminQueryPostgres`：全部P:Leaf，专属fixture/fault/barrier内承 |
  | `cmd/han-server/main_test.go`（改，R） | `TestProductionAdminArticleQuery`：PrincipalBoundary/ProductionSharedChain/ExactMethodsAndCoexistence/OldConsumerCompatibility |
- **实现合同/证据分层**：严格消费r20；普通active admin无需initial-admin；Trace/readiness/recovery/Session/AccessGeneral、GET无CSRF/If-Match，旧写安全链不变。完整RawQuery任何解析错误优先整体422/零下游，成功解码后cursor400/view-limit422；pending详情404先于身份，其他非法权威状态须经真实Handler为500且不包装ErrArticleInvalidState，旧治理409保留。三admin view及五cursor隔离，内部last-scanned/公开last-returned、lookahead不消耗；严格身份整503/第二批故障丢弃前页，去重/零越界身份读取；Admin12逐值独立oracle/同次Article强ETag，GET零业务写，不承诺跨批/跨页长快照。
- **Planning R01 F001专门门禁（2026-10-28，原作者修订待独立复审）**：[Planning R01](../../../../.spec/wip/article-management/review-tasks-plan-s8.2-r01.md) SHA256 `87E2FC11526456C89723E61EF1DA065FD671FF80A844E3B1C8921ACA14835746` 为CHANGES REQUIRED，仅MEDIUM `AM-S82-PLAN-R01-F001`；本次补齐承接，不自行标finding CLOSED或整卡Planning PASS。Dalton prep §8（`C:\Users\J00010385\AppData\Local\Temp\han-admin2-prep-93f3e966a7294f238ddfa37ce6d80f06\prep-admin2.md`，SHA256 `8290DA2C61ADD7FB66F96E9C421B8A8B7CD23E96938395AFBC1C945F9A73AD33`）只读准备已completed/stopped，原卡“G0进行”是登记时历史，不表示G0授权通过/代码或PG START。
- **已调查事实/来源**：[research-admin2-guard-r01.md](../../../../.spec/wip/article-management/research-admin2-guard-r01.md) SHA256 `213653A8A07EEDFFC65F07B834D5F78E6E66EB27C8FCE7C5EA2D2CB774FC2E4A` 为真实Explorer Harvey `01a17e25-f031-7bb1-a40b-c5ea32b5b7a7` 只读返回、Lead保存原文；限报告固定root摘要的静态接线结论，不是运行/独立Review PASS。报告§2～5已核：生产HTTP链为Trace→exact路由→HTTPAuthorizer/SessionStore→每用户PasswordReset Gate/Session/User→AccessGeneral→文章Handler，未接入独立通用readiness/recovery HTTP请求Guard；startup初始Reconcile/Start先于Serve、runtime失败协调Shutdown均不等于逐请求恢复503。Login maintenance配置、每用户reset Gate、恢复删除CheckRecoveryOpenGate及ProcessingGate不等同通用请求保护；RecoveryCoordinator/Replay组件存在不证明HTTP开放联动，外部流量设施仍unknown。不得以这些Gate、替身Principal的recovery名403或spy冒充通用恢复隔离；GET自身零业务写也不扩称认证链绝对零DB写。
- **受影响范围/继续边界**：保留固定r20 §3.3.9/§8.6与r16 A-ADM2-18要求，未降格成“启动成功/Session有效”。暂停受影响Root恢复保护承接、Case18 ProductionSharedChain的readiness/recovery子断言及依赖其满足的S8.2整卡接收/关闭；原全Case命中/关闭条件受本专门门禁约束，不能删该子断言、记PASS/豁免或在九文件内自建全局Guard/authority。其余内层U/C/P只读准备可继续；不因此获得代码、本地测试或PG运行授权，不无差别阻塞可行准备。
- **证据接收/恢复条件**：Lead已接收并保管Harvey原文；本次将精确证据与r20/r16现状冲突交Lead转原article Feature-Designer Hubble澄清验收含义、上游依赖及承载责任（未代填其已承接/已答复）。若需全局恢复隔离能力，交恢复主线/架构owner核真实authority、装配/失败行为与交付边界，不能由Planner/Dev自行选方案或新增无效requires。Hubble给出可核对澄清后，由原Test-Engineer Ohm确认/按需修订Case18分层映射；合同/范围改变时先固定必要上游修订与适用独立Review，Planner仅据实际依赖/写集变化有界对账。受影响R恢复须具备已明确合同、真实调用链/状态authority/失败语义/两GET接入及相应证据载体，由Lead核证后单独给执行信号；当前这些条件未满足，仍PENDING。F001本次修订交原Pauli复审，不代其结论；CODE/LOCAL与LOCALPG始终由Lead独立明确授权，不读DSN、不发START。
- **验收映射/命中**：以r16 §4.3.1 A-ADM2-01～20逐Case×U/C/P/R映射为唯一明细，不复制另一个20行表；四精确top及其所有必要leaf普通/race均真实run/pass、0fail/skip。优先核RawQuery错误零调用、静态复合分页/过滤多批、当前作者/治理屏障、严格身份第二批失败、SQL/Scan/Rows.Err/取消、pending/非法状态优先级、零写快照、12字段/ETag、真实构造共享与安全链/路由兼容。route spy只补method/path/链顺序，不能替代真实私有factory/constructor；P必须真实SQL→reader→service→AdminHandler，不冒充production root注册。先-list核四top/签名与P tag/package；Case/leaf缺项、只命中旧测试、零测试均不通过。
- **PG fixture/资源（待独立LOCALPG信号）**：只读复用已核旧helper片段；旧newNoticeHarness把schema与迁移放同tx，不能原样证明先提交schema及迁移失败retain。P新文件自承最小fixture：先CREATE/提交/记录随机schema，再完整0001～0009迁移，迁移失败schema同样retain可追踪，所有连接同search_path。获Lead LOCALPG START后才安全从User级HAN_TEST_POSTGRES_DSN进程注入、不打印/放CLI；核既有loopback5433/PG18.6，不另起DB/改权限。合法CHECK/FK，pending先建operation；identity故障在真实reader后受控注入，wrapper不替目标SQL/真实rows。独立SQL/显式序列及12字段oracle、全业务快照核零写；fixture动态writer预期delta单列。rows.Close/Err、tx提交/回滚、context取消、goroutine join/pool归还均留证；全部schema含失败retain，不DROP/清理；不改旧S2.2/S7测试、迁移/依赖/reader或加第十文件。
- **有效性负控**：依r16 §9.6优先真实行为RED，API缺失编译失败不算RED。未来获准Temp完整固定snapshot+overlay，保留RawQuery错误被吞、严格身份补active/返回半页、移除older-time SQL分支/公开cursor误用last-scanned的有意义反例，必须实际可编译行为FAIL并绑定断言/命令/hash/schema，再当前源码GREEN；不改/回滚生产工作树，不以缺环境/硬编码失败伪造检错。
- **待运行验收/回归命令**（仓库根目录；下列全部NOT RUN）：

  ```text
  go test ./internal/article -list '^(TestAdminQuery|TestAdminCursor)$' -count=1
  go test ./cmd/han-server -list '^TestProductionAdminArticleQuery$' -count=1
  go test -tags=integration ./test/integration/article -list '^TestAdminQueryPostgres$' -count=1
  go test ./internal/article -run '^(TestAdminQuery|TestAdminCursor)$' -count=1 -json
  go test -race ./internal/article -run '^(TestAdminQuery|TestAdminCursor)$' -count=1 -json
  go test ./cmd/han-server -run '^TestProductionAdminArticleQuery$' -count=1 -json
  go test -race ./cmd/han-server -run '^TestProductionAdminArticleQuery$' -count=1 -json
  go test -tags=integration ./test/integration/article -run '^TestAdminQueryPostgres$' -count=1 -timeout=8m -json
  go test -race -tags=integration ./test/integration/article -run '^TestAdminQueryPostgres$' -count=1 -timeout=8m -json
  go vet -tags=integration ./internal/article ./cmd/han-server ./test/integration/article
  go test ./internal/article -run '^(TestVisibility|TestCursor|TestVisibleScan|TestListArticles|TestGetArticle|TestUpdateArticle|TestOrphan)' -count=1 -json
  go test -race ./internal/article -run '^(TestVisibility|TestCursor|TestVisibleScan|TestListArticles|TestGetArticle|TestUpdateArticle|TestOrphan)' -count=1 -json
  go test ./cmd/han-server -run '^(TestProductionArticleAuthorRoutes|TestProductionArticleQuery|TestArticleBusinessRouter|TestProductionBusinessHandler|TestModeration_Router|TestOrphanDispositionRouter)' -count=1 -json
  go test -race ./cmd/han-server -run '^(TestProductionArticleAuthorRoutes|TestProductionArticleQuery|TestArticleBusinessRouter|TestProductionBusinessHandler|TestModeration_Router|TestOrphanDispositionRouter)' -count=1 -json
  ```

- **Manifest/成功接收**：四top/全部必要层leaf当前源普通/race真实命中及PG数据/资源/负控证据齐全后，Lead在九文件稳定同revision依次核 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。保留命令/退出码/时刻/raw/hash、九文件before/after、Case逐层结果/schema/资源清单；Manifest不替代integration/独立Review。实际产品bug保留失败schema交Lead协调原作者，不放宽测试或越界修其他生产文件。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S1.2`、`article-management-S2.1`、`article-management-S8.1`、`article-management-S4.6`；六项当前均[x]。S1.6/S2.2仅消费真实交付/协调文件窗口，不新增其业务requires，无反向依赖。
- **批次/真实认领/授权边界**：唯一B9F。R登记待做[ ]后按Lead实际选择与已发只读G0 PREP登记[-]、开始@2026-10-28；唯一Backend-Dev Dalton `01a17dcf-f2ed-7dc2-b7f3-6c72cf771448` 为未来九文件owner，目前只读G0进行中，尚无完成/ACK结果可代填。S1.6 main2窗口已由Lead明确释放；所有重叠文件仍由Lead核未知writer后单writer串行，准备可并行。PlanningReview R01 CHANGES REQUIRED（仅F001，原作者本次修订待Pauli复审）；代码/本地测试须Lead CODE/LOCAL信号，PG须另发LOCALPG信号，本规划不发START、不读DSN。非作者Planning/Code Reviewer由Lead实际指定，不虚构承接。
- **本任务关闭条件/停止**：同卡五生产四测试实现、U/C/R与P真实PG普通/race/负控/兼容回归、全部20Case适用层、非作者CodeReview及finding复审、Lead当前源码/测试/hash/范围/Manifest与实际接收齐全才可[x]；不借上游Review或旧S2.2/S7 PASS。第十文件、旧API/查询合同/schema/权限/依赖变化、第二router/服务/authority、未知writer或输入hash漂移，停止受影响范围交Lead/原设计owner。缺PG执行授权/环境只保留对应pending，不能跳过本功能PG验收；部署E2E/容量/真卷/独立ACL或producer专项不倒挂本卡，人的accepted/checkpoint不代签。
- **后续验证关联**：原article-management-S1.5继续四文件/完整16endpoint、三投影/ACL/S1–S16/producer验收；S8.2交付不自动关闭该卡或部署E2E。

- 2026-10-25：Lead 按固定五文件关闭 `identity-auth-S10.1` classification repair。R08 `.spec/wip/identity-auth/review-code-s10.1-r08.md`（SHA256 `30F1C4D2546F6E32BA23FCBECAD22235D49574CE35E66A236A8CCA18712DDC20`）确认 `IDENTITY-S10.1-REV-R07-F001` CLOSED、0 新 finding、Code Review Gate PASS；最终 SHA256 为 `internal/mailer/mailer.go`=`A2F103D6AF3664288D0C4F460EC5733E33FA6461FB8554A189FBC3194C5B4D0D`、`internal/mailer/smtp.go`=`0C64C229B7EE8101B0DCAD95C14431DCAF8ED55809C9FBFD50AEE75AF0221199`、`internal/mailer/smtp_test.go`=`656E12CB62C113C60AB074AE5374F9E206D226CCEE5B194BB1F0F8AA12B96D15`、`internal/identity/password_reset.go`=`5F2D0E1B534E1A4F47AF25675FB94AA3B2830A01C408F6C615E41FAB74A81575`、`internal/identity/password_reset_test.go`=`59AAC51742A61ADD318D8ADE5527231D4D5A198790687B101A35A5A843666E2A`。Lead 现场哈希/branch/HEAD/staged/越界、mailer/identity 定向普通与 race、`go vet ./internal/mailer ./internal/identity`、五文件 gofmt/diff-check 均 PASS；未读取/使用 `HAN_TEST_POSTGRES_DSN`，未跑真实 PostgreSQL/SMTP/部署/E2E，`han-server.exe` 保留。Tasks 三态更新为 57 / 27 completed / 8 active / 22 todo，active 精确移除 S10.1；S5.3 保持 active / `implementation paused`，仅上游 classification repair Gate 已满足，可进入下一次 G0 re-entry，尚未恢复实施、测试或关闭。
- 2026-10-25：Lead 按固定四文件关闭 `article-management-S2.1`。R01 `.spec/wip/article-management/review-code-s2.1-r01.md`（SHA256 `36BB65973CDC5998567B57B8C10AED1F6B65E114DEB5560414B19179540EF129`）的 `CHANGES REQUIRED` 历史保留；R02 `.spec/wip/article-management/review-code-s2.1-r02.md`（SHA256 `01373C88B3E706A0EB0573BEEAFB67668E8C6537061EB6BE8F6FCB86D4CA8D4F`）确认 F001/F002 CLOSED、0 新 finding、Code Review Gate PASS。R01 原 Reviewer Agent ID 不可恢复，Lead 指定已知非作者 Faraday `01a16940-6120-7021-bd06-eb0089956486` 替代复审并记录原因。最终 SHA256 为 `internal/article/query_service.go`=`539795E92A4F4276887C9ED3BCD9EEA08C51B1523271CB671D65D2B30B80C777`、`internal/article/query_test.go`=`DF40F63117A0C8F7569A255EABBF9DD95F81CA86FBAE17F88EE93A8F78A0C53E`、`cmd/han-server/main.go`=`841CCA74E3D4CAC1C07DE7217B332B0CA6451C8C3118E106DCBFB3D79F4E680E`、`cmd/han-server/main_test.go`=`CFE64848B572701E81608280B61C3392C19479407D0DA8CB02CB6C3A53DC31E4`。Lead 同 revision 哈希/branch/HEAD/staged/越界、article/server 普通与 race、`go vet ./internal/article ./cmd/han-server`、`go build ./cmd/han-server`、四文件 gofmt/diff-check均PASS；未连接真实PostgreSQL，未部署/E2E，AM-Q2数值、容量/P95仍TBD，不声明production readiness。Tasks三态更新为57 / 28 completed / 7 active / 22 todo，active精确移除article S2.1；S5.3保持active，本轮不声明已恢复实施、已测试或已关闭。
- 2026-10-25：Task-Planner 对 B7 的 `identity-auth-S9.1`、`article-management-S4.2`、`article-management-S4.5` 执行最小 R 对账。S9.1 五文件候选已存在且 R01 为 `CHANGES REQUIRED`，implementation paused，报告作者仅作 Poincare 线索，权威 owner/Reviewer unknown；S4.2 五文件候选已存在且 R01 FAIL（3 HIGH + 2 MEDIUM），memory 路径检查不替代合同/Review，implementation paused并先回设计/测试规划/任务写集；S4.5 三产物不存在，登记命令为 `[no tests to run]` 假绿，planning/implementation paused并先由 Test-Engineer 补合同。本轮新增task=0，不改 checkbox/日期/requires/DAG/批次范围；三态保持 57 / 29 completed / 6 active / 22 todo，active 集合不变。
- 2026-10-25：Task-Planner 消费 S9.1 R01、article design r14/Design Review R02 PASS、test-plan r12/Test-plan Review R02 PASS 与 S4.2 Code Review R01 FAIL，对 B7 三卡完成第二次最小 R 对账。S9.1 保持 `[-]` 且改为 `implementation paused / blocked-for-design-revision`，现有五文件无法安全实现 RecoveryRun authority locking 与 maintenance/readiness Gate；S4.2 保持 `[-]`，写集对账为精确 19 文件、0009 前向 generation/token、四符号前置 Gate与 A-REC-04～07 PostgreSQL普通/race，状态为 `implementation paused pending Planning Review`；S4.5 保持 `[-]`，写集对账为精确四文件、唯一锚定命令与零测试防假绿，状态为 `planning/implementation paused pending Planning Review`。design/test-plan PASS 不关闭 S4.2 的 3 HIGH + 2 MEDIUM。新增task=0，不改 checkbox/日期/requires/DAG/批次范围；三态仍为 57 / 29 completed / 6 active / 22 todo，active 集合不变；S10.5、S3.6、B8、部署/E2E均未启动，未运行 Go/PostgreSQL/网络/部署/E2E，未读取或使用 `HAN_TEST_POSTGRES_DSN`。
- 2026-10-25：Planning Review R02 `.spec/wip/article-management/review-tasks-plan-s4.2-s4.5-s9.1-r02.md`（SHA256 `A12062129944D8978D92A9732F255F7F1332DE7324AEA9BD3CF7185A7BC3693B`）确认 `ARTICLE-B7-PLAN-R01-F001` CLOSED、0 新 HIGH/MEDIUM finding、Planning Review Gate PASS；该 PASS 本身不构成实施 START。Lead 仅授权 `article-management-S4.5` START，唯一 owner 为 Backend-Dev Aristotle the 2nd `01a17225-a3d0-7fd1-aeb9-622f7617f436`；其实施前只读 G0 为 READY，只证明固定四文件足够且当前均不存在，不表示实施、测试、Review或关闭完成。S4.2 继续 `implementation paused pending Lead START`且不 START；S9.1 继续 `blocked-for-design-revision`。新增task=0，不改checkbox/日期/requires/DAG/统计/active集合；S10.5、S3.6、B8及部署/E2E不启动。
- 2026-10-25：Lead 按最终四文件关闭且仅关闭 `article-management-S4.5`。Aristotle the 2nd `01a17225-a3d0-7fd1-aeb9-622f7617f436` 完成原边界实施；Gauss R01（SHA256 `44DE54697DFEA8964E62A9AE2B8196C24E3233A44E177E04410F331AB64E5FF8`）为 `CHANGES REQUIRED / FAIL`（3 MEDIUM + 1 LOW），原 owner 返工时实际仅修改 Go 测试文件；Gauss R02（SHA256 `49340355047BFFF8A3F02A531AE20E9D163F65D6167E3190C1BC5FE5BC0E1F48`）确认 F001～F004 全部 CLOSED、无新 HIGH/MEDIUM、Code Review Gate PASS。Lead 同 revision 的专用普通/race/vet、package目录目标测试、gofmt、`go mod verify`、全仓 vet/build/contract/race/all、diff-check、staged、HEAD/Tasks/Plan/`han-server.exe` 漂移与越界核对均 PASS，且无 `[no tests to run]`；四文件窗口释放。Tasks 三态更新为 57 / 30 completed / 5 active / 22 todo，active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`identity-auth-S9.1`、`article-management-S4.2`、`article-management-S4.3`。S4.2 继续 `implementation paused pending Lead START`，S9.1 继续 `blocked-for-design-revision`，S4.3 继续环境阻塞；S4.4、S10.5、S3.6、B8、部署/E2E均未启动。本关闭只覆盖 synthetic fixture/runbook 与机械验证，不证明真实 inventory、真实旧备份恢复、retirement/cleanup 授权、灾备、Host、部署、E2E或 production readiness。
- 2026-10-27 article S3.1 作者自审 Code Review 关闭变更记录：Lead 以作者自审口径关闭 `article-management-S3.1`（Host 无委派能力，无 Reviewer / Backend-Dev Role 产出；依据用户 2026-10-27 明确授权「由作者自审 Code Review」与「对待确认或决策项采用建议值并继续推进」）。在精确五文件内按 article design r14 §3.3.4 / test-plan r12 A-ETAG-01～04、A-MOD-01～08 实现作者受控详情与强 ETag 修改；作者自审 Code Review `.spec/wip/article-management/review-code-s3.1-r01.md`（SHA256 `317BE33810150CE99B03DDA04A2A9BE4FB77321B5373128569387879891FA732`）`Code Review Gate: PASS（作者自审口径）`、无 HIGH/MEDIUM finding；最终 SHA256 `internal/article/command_service.go`=`B61ABD12ADD7AF43C7D677A89C097F44FFBB818A85D2D674CD218A81340DB6E4`、`internal/article/query_service.go`=`D969C1171E81889ABFC0D9CDB44DCD74466BD501828E5EDDB63073CEB1E6160B`、`internal/article/handler.go`=`B2E3F68995AEC9FA63C5E9F04EF0CCC458A4262045279D7A7DD794C2DD25DD00`、`internal/article/update_test.go`=`D6EF18BF78C12DC438C03CDC05C6DEAF4D6CF6157718BFF92C0DA4EF29366EAA`、`internal/article/author_projection_test.go`=`47314C84B79B53A6476F91A6F728CBC521CD78DE36AF61D68DA7298C1B0192F7`。Lead 现场专用命令普通/race、Manifest 七项固定顺序与真实 PostgreSQL 18.6 SQL 探针均 PASS；未扩第六文件，复用既有 `AuthorArticleView`，工作树只改五文件，`main`/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变，未 stage/commit/push，`han-server.exe` 保留。三态更新为 57 / 33 completed / 3 active / 21 todo，active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`article-management-S4.3`。未验证边界：production router 接线、A-ETAG-02 双 writer 并发、A-MOD-04 三视图跨模块契约、A-MOD-06/07/08 精确 N/N+1 前后快照、部署、浏览器/E2E、容量/P95 与 production readiness，由 S2.2/S1.5/S1.4 独立闭环；作者自审弱于非作者独立复审，作为已知限制保留。
- 2026-10-27 article S4.3 范围决策关闭变更记录：Lead 依据用户 2026-10-27 明确确认「本项目为测试项目、非真实交付项目，这版不做真卷韧性验证，后续也无需验证」关闭 `article-management-S4.3`。新增永久决定 **ADR-0024**（`.add/project/knowledge/decisions/content.md`，`内容` 领域）并同步 `.add/project/knowledge/decisions/README.md` 全局索引、领域文件表与场景查询表；ADR-0024 只改变本卡验证范围，不改变 ADR-0020 产品合同、tombstone/evidence 语义、Journal/finalizer 实现或 S4.6 已完成的 13 文件组件与双 build-tag 编译闭环。tasks SSOT 将 S4.3 更新为 `[x]`、完成@2026-10-27，并把 `**阻塞注记**` 改为 `**范围变更与关闭注记**`、`**本任务关闭条件**` 改为变更后版本；同时修正卡内 `当前状态` 错贴的 S1.6 证据（原误写 `go test -tags=integration ./test/integration/identity` PASS）为本卡真实状态：三文件 `test/integration/article/deletion_journal_test.go`、`deletion_finalizer_test.go`、`journal_faults_test.go` 未创建、未执行，专用命令与 Manifest 七项均 NOT RUN。**关闭性质**：以「经决策不验证」关闭，不是验证通过。**未验证边界（保留）**：真实跨进程 `flock`、file/directory `fsync` 耐久性、崩溃/断电、磁盘空间与介质故障、真实 journal provider/root/keyring 均无实测证据；S4.6 的组件/fake 与双平台编译证据不等于真卷结论。**后续边界**：`article-management-S4.4`（ReplayStatus producer 与旧备份恢复）与 `article-management-S1.4`（部署后 E2E）其余未验证边界不变，不因本决定获得证据；不宣称 production readiness、灾备就绪、容量或 P95。三态由 57 / 36 completed / 3 active / 18 todo 更新为 57 / 37 completed / 2 active / 18 todo，active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`。本 Host 无委派能力，无 Arch-Designer / Task-Planner Role 产出：ADR-0024 由 Lead 接管产出并标注无 Role 产出；Git 未 add/commit/push，`main`/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变，`han-server.exe` 保留。
- 2026-10-27 identity S1.5 作者自审 Code Review 关闭变更记录：Lead 以作者自审口径关闭 `identity-auth-S1.5`（Host 无委派能力，无 Reviewer Role 产出；依据用户 2026-10-27 明确授权「由作者自审 Code Review」与「对待确认或决策项采用建议值并继续推进」）。在精确五文件 `test/contract/identity/rest_contract_test.go`、`internal_ports_test.go`、`replay_status_consumer_test.go`、`testdata/contracts/article-deletion-replay/ready-v1.json`、`unavailable-v1.json`（全部新增，均带 `//go:build contract`）内按 identity test-plan r18 I-CON-01～03/I-REC-06～08/I-PRD-01/05/11～14/19 与 design r40 §3.3.11/12、§9.4 验证：身份 REST 严格 JSON 与稳定错误 envelope（code/message/field_errors/trace_id 一致、无 cause 泄露、404/405 JSON 合同、成功 body allowlist）；S1–S16 内部端口方向与签名（`ArticleAccountPort`/`ArticleIdentityReader`/`RecoveryRunLocker` 只锁不迁/`RecoveryOpenGate`/opaque `RecoveryDeleteCapability`/`ArticleDeletionGate` 双读）与 17 个公共 code 的 status/retry；password-reset durable 中性 `PasswordResetGate`、原子 `BeginOrClassifyCandidate`、闭合分类枚举与三消费者共享同一 Gate 类型；ReplayStatus Ready20/Unavailable6 严格消费者（20 字段、前 16 字段 canonical digest 与唯一 authority/常量/声明值三者相等、metadata 不影响 digest、Unavailable 零 digest、extra/missing/null/类型/未知版本/count 不变量 fail closed、cross-run/namespace 拒绝）。作者自审 Code Review `.spec/wip/identity-auth/review-code-s1.5-r01.md`（SHA256 `6CF841EC84CDF57AF6B19637B7B56E4E2AC5716B85854C4715AB76F7DA9F421A`）为 `Code Review Gate: PASS（作者自审口径）`、无 HIGH/MEDIUM finding；最终 SHA256 `rest_contract_test.go`=`86133EF48C35F704628100AE72C865D8526142791C3F7AE10FF1725230C994A5`、`internal_ports_test.go`=`F749E7185D660CBFEC481CB5C83B30BA730003383354F622C6128123F12BBB1D`、`replay_status_consumer_test.go`=`9E9495C24D3212ADB34AEC199D90650CF99FA1E59C035CEA6ABB7DEC32C49752`、`ready-v1.json`=`586A0B8F76ADE8B720E44094A0D13163C827D231AC9CBC786C6D4AF92E90F0F2`、`unavailable-v1.json`=`929193CD06BA566402F3F3C1D6321E57BDA95D0270DED666B5C951D97B21AC1B`。Lead 现场专用命令 `go test -tags=contract ./test/contract/identity -count=1`（19 顶层测试）与 `-race -tags=contract ./test/contract/...` 均 PASS；Manifest 七项固定顺序全部 PASS；写集仅新增五文件、未改他人文件，`main`/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 未变，未 stage/commit/push，`han-server.exe` 保留。三态更新为 57 / 34 completed / 3 active / 20 todo，active 精确为 `identity-auth-S3.3`、`identity-auth-S3.5`、`article-management-S4.3`。未验证边界：真实 PostgreSQL 多连接/commit-ack-loss/迁移混跑由 S1.6 承接，真实 router 组装由既有已关闭 router 证据承接，部署、浏览器/E2E、容量/P95 与 production readiness 均未验证；作者自审弱于非作者独立复审，作为已知限制保留。


<a id="recovery-runtime-planning-r01"></a>

## 2026-10-29 恢复runtime稳定切片：正式R登记

本次先登记原Tasks再由Batch-Plan消费，完整范围仅identity S04、article Story4/8及必要共享接缝。原59卡checkbox/完成证据/三active、原requires/写集/关闭合同均保持；“已建成可复用”不重开已闭S4.2/S10.5/S9.1/S4.6，也不按存在代码推完成。标签待做/需改/已建成可复用/unknown仅是静态事实栏，unknown不是任务第四态，废弃0。原历史段落中的59/未登记/本轮不加边只指旧切片；本增量覆盖后的当前数量见本文件统计。

### 稳定输入、确认来源与采用边界

Main依据用户standing采用建议授权，正式授予本次规划WRITE START，接收原R预对账状态并登记稳定软件切片；不再次要求用户拍状态，不代人的设计签字。仅Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，任务/批次规划和TP mapping/meta写权；无源码、DDL、运行或Git写权。全部新任务未认领，代码/测试符号待实施，所有未来命令NOT RUN；PlanningReview/实际接收仍pending，不能称实施START。

| 技术输入（当时固定版本） | SHA256 / 适用范围 |
|---|---|
| identity design r50 / Story r17 | `6634BED32EDE264F6DA6F47C9A7E8152ADD0B19C2DDCBCF3ED4708AE4BA4109D` / `8B5EBCD5A78A20769FAF3D31492562C271F0DFB3B770E38383B87DF250559616`；S04、共享恢复及必要Session/Reset接缝 |
| article design r23 / Story r11 | `2223BE4D005753FBF8D8AE0116F6926A7E8C044482BCBB0FADAB811677066350` / `F2AB369EE2259F043ADD608A013660904B201AB5F4066D76747A81348D65998C`；Story4/8及必要root依赖 |
| framework ADR-0027 r02 / decision index | `36F4B49FD5C9B6FC0B351BAF326E03CF74889A5FA91CD052516D4B2542708B79` / `157A549AD375B650479900A350464FCB842E828EFD296F3A50FA406A996FBAA1`；六对象中的历史技术输入，不跟随后继引用账hash循环刷新 |
| architecture r32 / boundaries | architecture `89605FAF5D0C7C663791086112E6BD1A33FC5C309A01E6DBCAA2C91795B0A27E`；identity boundary r13仍draft，仅原已接收S13–S16；article boundary r13 accepted，不外推实现通过 |
| identity TP r24 受审技术稿 | `E6906617E51CC349B783896A3A1D828DA4AA8A03716D0C104E77E9AB40680CF6`；§4.7.1 `recovery-runtime-r24` |
| article TP r21 R02修复后技术稿 | `E39C980C041CAC0C0A5441A9DDC52E51024C75E1911A7EE22B5332EDE66B9EDE`；§4.8.2 `recovery-runtime-r21`；786行合法oracle保留 |

Design R02仅设计层关闭F001/F002，R03只核最终元数据。TP R01原报告 `A4C5D32A0E7BDF03FD9F2F492355AFDE0ED22F2A3A67D2E3B7F61590470D8503` 保留CHANGES REQUIRED及唯一MEDIUM失败链；Kuhn [TP R02](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/review-r02/review-report.md) SHA `0EDF27B970273137B24EC25C8370E78CBFACBBAA36B3E06794338697782723A1`、[receipt.json](../../../../.spec/wip/recovery-runtime/evidence-testplan-r01/review-r02/receipt.json) SHA `A50D584547AC2D255F007781C8BAFE5873DC54B0D4532694B6815379889F6B7D`，仅定点关闭该规划oracle修复、新增0；Main已全文技术接收。它覆盖上表E690/E39技术稿，不预签本次Task/Plan与TP mapping/meta的PlanningReview。

framework/README历史技术输入的immutable副本位于 `.spec/wip/recovery-runtime/evidence-design-r01/adoption-r02/frozen/.add/project/knowledge/decisions/`。Main已采纳Pascal有界计数建议：两正式TP各实质采用1、本Tasks+Plan同pass联合1，派生mapping不重复。当前ADR引用账仍3；本规划不预记3→6，不改ADR/index，不以未来引用账hash改写技术source。后继Arch在四稿冻结/Main接收后另授仅metadata登记，最终六对象由Main/非作者核证。


证据定位缩写：ID/IS为identity design/stories，AD/AS为article design/stories，T为本Tasks的登记前行号，P为原Plan。所有代码定位来自Main已接收的有界R读集及固定源revision，不是本轮重新跑测试；完整来源均为仓内当前design/Story/TP、原卡和真实代码，不以临时文件充当业务权威。

### 2. Story 一级结论

| Feature / Story | 一级标签 | 设计及代码证据 | 原任务责任与结论 |
|---|---|---|---|
| identity S04 / 恢复共享基础；仅带入必要 Login/Session/Reset 接缝 | 需改 | IS:184,195–206；ID:1426–1640,3480–3761。`internal/recovery/coordinator.go:44–59,156` 只消费 identity_ready 及后继；`internal/identity/recovery_run_store.go:60–65` 仅按 run 读取；`cmd/han-server/main.go:377–380` maintenance 拒启，无正向 normal 证明 | T:546 原 S4.1、T:569 原 S4.2 的已闭历史保留。T:636 原 S4.4 是独立恢复专项，不拥有新增生产实现；T:654/667 的真实 provider/retirement 条件保留。新增软件责任不能复用旧关闭作写权 |
| article Story4 / 文章删除及 S16 必要共享基础 | 需改 | AS:117,126–128；AD:2284–2476。`internal/article/deletion_journal.go:84–99,345–392` 无 Anchor；`:418,446` 读也 sync；`replay_evidence.go:317–356` 只有 evidence/current 联合发布，`:443–456,481–520` 为 fresh Facts 路径 | T:1436/1461 的 S4.1/S4.2 已闭合同保留；新锚定写者/publication/history 另提实现候选，不重开 S4.2，不塞进 T:1542 的 S4.4 测试卡。S4.4 整卡未闭 |
| article Story8 / 两管理员 GET 的恢复隔离与恢复后开流 | 需改 | AS:226,230；AD §8.6 新增内容、architecture.md:1379–1392。`cmd/han-server/main.go:3033–3070` 已有两个 GET 的正常安全接线，`:3462–3488` exact 方法/Trace 可复用，但无全局准入屏障；`internal/platform/http_security.go:309–339` 只是 Session/CSRF/Access 检查 | T:1838–1842：S8.2 局部九文件已交付/释放当时写窗，Case18 两全局分项及整卡仍阻塞。T:1224–1231：S1.5 组件/Audit SQL 有界接收不替完整契约/ACL/恢复。复用原卡作为验收接收，不赋予 root 新写权 |

三 Story 均“需改”，不是全 Story 待从零实现。下表把可复用与未建成的行为分开，避免粗标签掩盖既有成果。

### 3. identity S04 与必要共享接缝

| 增量能力 | 标签 | 设计 → 真实代码 / 原卡定位 | 最小候选 / 保留边界 |
|---|---|---|---|
| 中性 Runtime、默认 closed、私有 phase/lease/代际、关闭原因和 append 屏障 | 待做 | ID:1434–1488,3520–3539；定向无 Runtime/PhasePermit/PauseArticleAppend 等符号；现有 `coordinator.go:23–41` 只有 Consume 依赖，没有 runtime。T:569–612 旧 S4.2 不含此完整行为 | “机制”（下列正式任务）；无持久 ready，无 recovery 反向 import identity/article/pgx |
| Recovery Journal 纯读 snapshot 与全路径 ctx / 写 unknown 真 join | 需改 | ID:1505,1519–1528,1617；`journal.go:136–140` 无 ctx，`:188–195` 构造 bootstrap/load，`:446–470` 读进 load，`:614,663–701` Background、rename/remove/sync/commit pending。T:578–590 旧 owner 文件 | “只读 Journal”；复用 parser/HMAC/keyring/CAS 协议，不能直接拿旧 Read 当纯读 |
| 同一个 Store 全量短事务 snapshot、完整 NULL/字段集合、counter+Digest/双 namespace ABA 重核 | 需改 | ID:1494–1505,1542–1556,1621–1640；`recovery_run_store.go:17–25,60–116` 现有 pool 和按 run 锁读/转换，非全量投影；`journal.go:99–123` 缺 publication/restore/信任根证明字段；0005 仅现有 run 状态权威。T:585–586 | “Store 元数据”；新0010完整一次拥有，原0001–0009不改。不能用 xmin/hash/晚 succeeded 覆盖历史 |
| 可信 genesis / anchored migration、prepared→bound、历史全部闭合、失败/缺行优先 | 待做 | ID:3513–3518,3753–3761；现有 `JournalOptions.Bootstrap` 位于 `journal.go:143–151`，只属于组件安装参数，不是批准来源。无 VerifyGenesis/安装表符号。T:636–665 真实恢复/provider 责任独立 | “安装”；可准备软件协议；真实批准/检查点/ACL未知不自动首装 |
| running 身份终结、review/幂等核对、初始信任根、CompleteIdentityReview | 待做（局部 canonical 可复用） | ID:3569,3665,3745–3751；没有 recovery_execution.go 及 Executor 目标方法。`deletion.go:773–793,1000–1008` 已有同 Store running 锁与关闭 Gate 的 canonical 接缝；T 的 S9.1 已闭历史见原账 | “身份执行”；复用 canonical 全事务，不另写用户删除实现；Complete 不在 Store 锁下反向锁 User |
| F001 maintenance-scope Session 及 G→G+1 原子强改密/退出维护 | 需改 | ID:3575–3632，IS:195–197；`login.go:1448–1453` 恢复要求 true，`session/store.go:568` 普通总条件 false，`:315–317` 无效会话撤销；`bootstrap.go:344–356` 不清 flag/不写 run change proof。T:546–568 的 S4.1 不是新增维护 scope 交付 | “维护认证”；普通 resolver 保留 false 并限定 ordinary；恢复 me/password、重新维护登录、ACK loss、多会话撤销/切普通重新登录均同一闭环 |
| F002 恢复专用 reset 资格/可信批准/新 candidate 唯一发送权/无 Mailer 结算/原子 restored | 需改 | ID:3634–3691、IS 公共恢复验收；`password_reset.go:180–209,779–789,1293–1299` 拒 true/locked；`password_reset_operator.go:489–534` accepted 对 locked 冲突；`password_reset_store.go:491–554` 已有 operation authority。原 S10.1–10.4/S11 与 T:1057 S10.5 是不同闭环 | “恢复重置”；日常三个拒绝保持。复用 durable operation、原 job lease/due/MaxRetries、同键分类、Mailer/unknown窗口，增加恢复绑定/classifier检查而非另建协议 |
| strict Ready20/Unavailable6、Begin key/retained key、现有 Store 转换/canonical 锁与 reset Gate | 已建成可复用（仅静态适用部分） | `coordinator.go:60–145` 现有 fresh/Consumed 线；`journal.go:75–83,477–490` CAS 端口及 run 绑定；`deletion.go:773–793`；`session/store.go:298–320` User-first Gate/CSRF；设计保留 S13–S16 | 复用其组件/回归，不登记一套重复实现，不将已有关闭改成待验认领；新接线另按新 revision 验证 |
| 真实 CAS/独立故障域、issuer/verifier/retirement、批准源/人员/ACL/备份/RPO/target/可信工具 | unknown | T:654–684、ID:3695–3707,3733–3739,3759；本轮禁止探真实环境，未作 provider/DSN/网络检查 | 原 identity S4.5 / S4.6 与两 S4.4/部署 E2E 接收。只阻止对应真实动作/结论，不阻止软件候选 |

### 4. article Story4 / Story8

| 增量能力 | 标签 | 设计 → 真实代码 / 原卡定位 | 最小候选 / 保留边界 |
|---|---|---|---|
| 同一个 article 锚定 writer、staged→pending→published→committed+独立回读、同步关闭通知 | 需改 | AD:2284–2339；`deletion_journal.go:84–99,345–392` 只保存目录/key；`:168–209` 会 ensure 目录；没有 Anchor/revision/runtime；T:1436–1458 原删除实现 | “文章锚定”；沿原 tombstone/HMAC/operation，CAS 未确认不 DELETE/204，健康只清自身 token |
| article pure snapshot/Prefixes、controlled coordinate、recovery replay 精确 committed 只读、startup 内部可追加 | 需改 | AD:2314–2326,2341–2356；`deletion_journal.go:400–446` 读会 sync；`deletion_finalizer.go:133–150` pending 收敛会 Append，需要按真实阶段区分。旧锁只有 namespace 互斥，不证明部署独占 | 同“文章锚定”；受控软件故障矩阵与纯读同闭环，不把“恢复时公共关闭”变成任意 append 权限 |
| evidence/current/publication 同事务与独立不可变来源、历史精确 closure；legacy 来源受控导入 | 需改 | AD:2430–2476；`replay_evidence.go:317–356` 发布不写 receipt，`:443–456` 历史按 fresh Facts 筛，`:481–520` refresh；0011不存在。T:1461–1520/S4.2 及 T:1550–1556/S4.4 旧修复保留 | “发布历史”；新0011和直接测试。历史不刷新Facts，fresh仍拒stale；旧 self-consistent evidence 不自动获得过去已发布身份 |
| 原 pending 隐藏→Append→finalizer物理删除/Audit→204、source/Facts/计数/wire | 已建成可复用（待新 writer 兼容回归） | `deletion_coordinator.go:98–122` Append成功后 Finalize，`deletion_finalizer.go:123–150`，`replay_evidence.go:317–356,481–522`、`deletion_replay_source.go:99,150`；T:1436/1461 及原 Issue 有界接收段 | 不重开旧 S4.2，不改变 r22 首份真实分项/历史唯一复用，新增 publication 也不复制 S16 encoder |
| Story8 的两个真实生产 GET / Session/AccessGeneral / Trace / 404/405/Allow | 已建成可复用（仅现有局部） | `main.go:3033–3070,3462–3488`；T:1840–1842 当前交付、:1871–1883 九文件。`http_security.go:284–339` 可注入最小 resolver | 维持原 S8.2 五生产四测试成果；query/model/policy/DTO/SQL无需为 runtime 重做 |
| 两 GET 关闭时真503＋零 service/repo/author-reader、完整恢复后经同 Session 正常开放 | 待做 | AS:230、AD §8.6 r23、architecture.md:1379；当前 exactrouter 在 `main.go:3479` 直接进入已保护业务 handler，无通用 recovery guard；T:1841 明确尚未交付 | “HTTP屏障”＋“唯一root”；原 Case18 是接收关联，不是新实现写卡。拒启/未监听/账号403/spy造503均不足 |
| 真 Journal/POSIX、旧备份、S4.4第三原路径、实际应用 ACL、S1.5/fullCase18/fullS8.2 | unknown / 未完成的验证责任 | T:1563–1570：old_backup_restore_test.go仍缺，原三路径/六requires/fullclose保留；T:1228/1231 ACL未知；T:1841 Case18未闭 | 保留两专项、S1.5/S8.2原责任，不按代码复用推完成，不扩大 ADR-0024 的 S4.3 范围关闭 |

### 5. 必要 root 共享依赖

| 能力 | 标签与证据 | 候选 |
|---|---|---|
| 三 ordinary workers 已存在，但共享 Runtime lease、真实 started握手、pending前停新claim和故障真join未接 | 需改。`lifecycle.go:75–83` 已准确列 Article deletion/PasswordReset/Article notice；`:120–145` Start返回即标started，无目标握手；`:213–220` 已有 notice BeginShutdown/Stop 可复用。`deletion_worker.go:309–337`、`password_reset_worker.go:235–265`、`notice_dispatcher.go:230–248` 都无目标 runtime lease | “三worker”；保持原 discovery/lease/retry/通知算法及 shared deadline，不遗漏 notice |
| 独立 recovery命令族、恢复域外部署锁、批准 restore-v1/verify-v1执行协议、stable artifact handle、私有stdin/target隔离 | 待做。ID:3693–3739；目标 recovery_restore.go/recovery_runtime.go均不存在；当前 main.go:377–380 只有拒启。不能以旧 Windows requeue审批/签名/epoch代替恢复工具/批准源 | “恢复工具”；具体生产程序/原生耐久回执/ACL/支持矩阵unknown，不把软件传输协议称为真实restore |
| 同一pool/Store/Journal/source/provider/Runtime、同router内部surface、loopback control、live/ready、维护路由、普通→恢复→普通有序运行 | 需改。ID:3488–3492,3539,3741–3751；main.go:520–554 是现有businessRuntime/文章实例装配；:3116–3138有现有唯一replay source入口但没有全局Coordinator/runtime；lifecycle.go:74,80–83只有两server slot | “唯一root”；共享 Store、完整配置预检、三server结果容量/资源反向释放，不新建第二router/pool/authority |



### 新任务通用关闭合同

以下每个开发任务都自带先行为失败证据、实现、局部普通/race、所列必要P/R、非作者CodeReview/修复复审、Main同revision status/diff/文件边界/实际证据与适用接收。任务自己的SQL原子性、独立目标readback、完整root软件正向不得全部推给后继联合验证。后继专项/真实E2E只是关联，不作为开发关闭的反向依赖；源码未实现/所需PG未运行仍不能关闭本卡。

所有下列top和risk leaf是本次登记的**待实施符号**，不是声称仓库已有或可立即运行。逐包 `-list` 只核top存在；实际普通/race `-json` 须核表列风险leaf/最低层真实命中，目标零命中/Skip/编译或环境失败不能当行为PASS。P/R使用真实Store/Session/source/provider/Finalizer/唯一factory链，外部CAS/文件/restore/批准源才可测试专用adapter，不能注册production fallback、手造503或固定Ready。参数化细枝优先U/C，只有SQL/提交/锁能证明的升P，R串关键正负链，避免重复全组合。

工作目录固定仓库根；每卡另顺序执行现役Manifest七项，全部NOT RUN：`gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。默认race不替显式integration/contract-tag race，七项也不代各卡必要P/R。

运行需另获Main精确SOURCE/LOCAL/PG/进程窗口。P每例独立批准schema、合法迁移、独立连接oracle，Rows/Tx/连接归还/pool关闭；R全HTTP/维护/worker/child真实join后Article资源→pool。schema、证据、制品/raw全retain，旧autoDROP harness不得直接运行，DROP/delete/move/归档另批。真实provider/审批/ACL/tool/备份/issuer/retirement/target/RPO等unknown只阻对应真实启用/结论，不阻软件合同与合成权威fixture准备。

写集外文件、未知writer/漂移、迁移号占用、要求变更公共wire/S16/namespace/authority/状态机时停止受影响部分交Main/原设计owner；不自行扩权/裁尾/删历史。具体生产provider/工具/OS锁后端/新增SDK与尚未固定的额外文件需原设计及精确窗口，不能凭本软件卡填真实值。无独立Reviewer保持待审；不把作者自检、范围关闭或旧卡PASS外推当前revision。


### 新增任务（原登记时全部未认领；当前责任以各卡为准）

<a id="task-identity-auth-s4-7"></a>

- [x] identity-auth-S4.7 实现默认关闭的中性准入和阶段/工作许可，完整组合历史或genesis证明，精确撤销代际和append自有关闭原因。 开始@2026-10-29 完成@2026-10-29

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 / 两 feature 必要共享基础。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §3.3.1/§4.2；identity TP §4.7.1:790–795。原r50及E690/E39受审技术SHA保留为原登记历史输入，不循环追新hash；原六稿PlanningReview R02不覆盖本次新增映射。本卡新增承接identity设计r52（SHA256 `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`，§3.3.1动作依赖/中性ports、§4.2目的producer/Resume/退出、§6.1、§9.1；[r52完整差量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r52-author-r02/design-r51-to-r52.diff)）。本次由Main授权正式登记；r52设计R02独立PASS的有界范围、Main实际接收及F001 SOURCE/LOCAL来源见S4.7下方r52承接记录。设计仍draft、人的精确接纳unknown；原六稿PlanningReview不覆盖本次新增规划映射，本增量已由Kuhn独立规划Review有界PASS/新增0，Main现场接收见S4.7下方最新记录；新增本卡组件验收已执行，结果见Main最终接收，不覆盖原卡既有实际自测历史。
- **文件所有权**：仅以下新增/修改清单；实际owner为Zeno / add-backend-dev（实际Agent ID `01a18486-3e11-7852-9723-ad380e0bb392`），来源是Main依据用户standing授权实际spawn；当前仅只读G0 PREP，源码/LOCAL写与运行窗口须等Main明确SOURCE/LOCAL信号，本次认领登记不授代码写权。
  - `internal/recovery/runtime.go`（新增）
  - `internal/recovery/admission.go`（新增）
  - `internal/recovery/runtime_test.go`（新增）
  - `internal/recovery/admission_test.go`（新增）
  - `internal/recovery/journal.go`（修改）
- **文件数**：5；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `project-bootstrap-B0.1`、`identity-auth-S4.2`、`article-management-S4.2`
- **批次关联**：[B13A](../plans/ir-01-content-community-batches.md#b13a)；原批次安排保持。Main已实际派发上述owner，2026-10-29真实认领；认领登记时只读G0 PREP进行中、尚无G0结果，本登记本身不是G0通过、SOURCE/LOCAL信号或测试PASS。
- **实施窗口（Main补记）**：随后Main已接收[G0 READY可读回执](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/g0-r01/receipt-utf8.md)，并向同一owner发出[五源SOURCE/LOCAL START](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/source-local-start.json)。上行文件所有权的窗口前提已由此信号满足，仅本卡五源及隔离本地验证；PG/DDL、生产服务/恢复、Git写与清理均未授权。发窗时未CodeReview或Main最终接收；后续实际结果见下行，不以发窗或自测代任务PASS。
- **代码评审（Main接收）**：[Ptolemy R01](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/review-r01/review-report.md) CHANGES REQUIRED，`RECOVERY-B13A-CR-R01-F001`～`F004`为3 HIGH＋1 MEDIUM，涉及可达恢复阶段许可、Stop取消、Verify真join与合法append观测。Main已全文接收并安排原作者Zeno同五源修复、同Reviewer复审；本卡保持[-]。原普通/race各99风险leaf及项目检查6项通过仅证各自已跑范围，不能关闭这4项。
- **既有检查处置**：Main按用户standing建议值授权采用[本卡有限格式例外](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/format-exception-main-r01.json)，仅覆盖写前已存在且23路径/哈希不变的旧证据全仓gofmt exit2；全仓结果仍FAIL，不改/清理旧证据、不扩至其他任务。源码修订须重新核源格式/其余检查及例外适用性，不豁免代码finding、真实验证或Review。
- **r52承接与当前窗口（本次同步）**：采用identity设计r52（SHA256 `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`）；[同Kuhn设计R02报告](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-review-r02/review-report.md) SHA256 `21408124976E38F96C07DFF4444CA52602EBB7AC93740E9D6905A20EE66C26FF`、[receipt.json](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-review-r02/receipt.json) SHA256 `329ECED99937C929DCCA03249ECB59DABC7F8F053470962895A993A468D18B9A`仅在原设计finding修复及相关新风险的设计合同内PASS，`RECOVERY-DES-R51-R01-F001` CLOSED/新增0；Main已完整实核接收。Main另向原owner Zeno（`01a18486-3e11-7852-9723-ad380e0bb392`）发出[F001五源SOURCE/LOCAL](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/f001-source-local-start-r52.json)，当前原作者修复中，仍遵守该信号的最小r52 G0更新与原五源/隔离检查界限；不由设计PASS推断G0结果或测试成功。Code F001及F002–4均未独立关闭；F002–4作者已修不代非作者复审。本次仅正式同步四卡/四批责任映射，该新增规划映射已由同Kuhn独立Review有界PASS/新增0并经Main现场接收；不授第六源/真实环境/新任务窗口，不改变draft/人的精确接纳unknown。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/recovery -list '^TestRecoveryRuntimeAdmission$'
go test ./internal/recovery -run '^TestRecoveryRuntimeAdmission$' -count=1 -json
go test -race ./internal/recovery -run '^TestRecoveryRuntimeAdmission$' -count=1 -json
go vet ./internal/recovery
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `FullHistoryPriority` | U/C | 早failed/unfinished与晚succeeded；两侧缺行、全量集合/字段，禁止latest与supersede |
| `GenesisAndExactHistory` | U/C | genesis仅无历史；精确publication/consumed闭合与Begin顺序 |
| `IndependentRevisionABA` | U/C | 两namespace独立token、H0→pending→H0新revision、Store counter/Digest与2次完整proof |
| `PermitGenerationAndOwnCause` | U/race | 零值/错实例/跨run/过代/Release后拒绝，健康只清自身原因 |
| `DriftCancellationAndNoCachedOpen` | U/race | 漂移/取消/失锁/迟到成功撤代际，不跨重启缓存true |
| `PublicEntryProducerAndModeDependencies` | U/C/race | 在原runtime_test.go的TestRecoveryRuntimeAdmission下由真实公开Enter/Resume产许可，不直调phaseLocked：共同依赖及typed nil、Normal/Verify/Install/Enter/Resume按mode检查；缺本动作Consumer/Exit等在独占/排空/Begin等副作用前拒绝；Begin独立committed→recovery_restore→真实adapter完成/独立target证明→recovery_bind→同绑定running→recovery_identity→identity_ready读回→recovery_replay，逐步撤旧grant/ctx；历史last≠当前run、错run/backup/install/实例/目的/代际、取消/失锁/迟到成功均零非法写。外部restore/SQL用explicit中性fixture，不把后继生产实现作本卡关闭前置 |
| `ResumeBeginSecretWriteFence` | U/C/race | 同top使用ContextRecoveryJournal.VerifyBeginSecret(ctx,RunBinding,[]byte)→JournalEntry中性fixture；先独立证initial仍未准备才lazy读候选，verifier成功后核既有Begin/snapshot与run/backup/keyID/version/fingerprint/UTC微秒、Store.started_at、namespace/id/epoch/head/revision及当前许可/ctx/独占，再PrepareTrust；错secret/错返回entry/漂移/取消/失锁零PrepareTrust，已准备/已改密零秘密读取/验证，Resume零新Begin/盲Replace。Runtime只声明/调用，不实现HMAC/持key |
| `RecoveryExitBeforeStartup` | U/C/race | 同top以原Consume(ctx,runID)窄consumer和ExitRecovery(ctx,permit,binding)阻塞fixture：Consumer返回值不代同Store succeeded+精确consumed/publication独立来源；Resume已succeeded只核尾部、不再Consume；同replay许可才调Exit，缺/错/失败/ACK未知/未返回不启动ordinary worker或开流；出口确认后撤维护/replay能力，再三worker握手/最终proof。端口调用/真实等待在Runtime锁外，保留Stop取消、Verify owned-operation真join与合法已通知append观测回归，F002–4不因作者自测标独立关闭 |

- **本任务关闭条件**：全量历史/failed与缺行优先；两namespace独立revision与H0→pending→H0 ABA；Store全部字段/NULL和两次重核；最大2次完整proof尝试；零值/错实例/过代能力拒绝；健康只解除自身token；失权/迟到成功不开放。此卡用显式完整端口fixture验证判定，不用孤立SetReady。 新增局部关闭还须通过公开Enter/Resume与上述中性fixture取得真实可达许可和写前拒绝证据；Journal verifier的真实retained-key/HMAC实现归S4.8，SQL退出归S4.10，complete返回/真实root join归S4.16；本卡只验port调用、独立来源重核和失败不开流，不反向等待三卡或真实E2E。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。后继真实Store/Journal/closure和root联合软件验证；生产来源/故障域独立。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：journal.go本卡只追加/调整中性投影类型/ports；含RecoveryConsumer同Consume签名、RecoveryExit及ContextRecoveryJournal.VerifyBeginSecret声明；key/HMAC算法不移入Runtime，r52中性调用不转移后继文件/写权；旧Journal写方法ctx原位迁移由下一候选串行接管。禁止引入具体业务包或pgx依赖，禁止新持久ready。 适用本节通用停止合同；未出现确认不当成认领/关闭。
- **最新代码复审（Main接收，2026-10-29）**：[同Ptolemy代码R02](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/review-r02/review-report.md)对精确冻结五源确认原R01 F001～F004全部CLOSED；新增HIGH `RECOVERY-B13A-CR-R02-F001/F002`仍OPEN，分别为受控恢复跨端口未解释的namespace revision漂移被重新接受，以及consumed且缺Store当前行时仍先授bind许可并调用写端口。独立普通/race各9叶、7PASS＋2行为FAIL，Main已全文核报告/receipt并解析原始事件；不是编译失败。原源码139叶及其检查只证已跑范围，不能关闭这两项。本卡仍[-]，Main交原Zeno同五源定点返工后由同Reviewer复审，不新增文件/设计/真实环境授权。r52四卡四批规划及同源ledger增量另由[Kuhn规划核验](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/plan-r52-review-r01/review-report.md)有界PASS/新增0，并有[Main现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/mapping-ledger-main-reception-r01.json)；上文历史待审时点不外推为代码关闭。
- **最终接收（Main，2026-10-29）**：Main已全文核原作者[R03完整差量与交付](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/author-r03/delivery.json)、现场五源SHA/范围/固定branch与HEAD/空暂存区，并对当前精确版本独立复跑普通和race各230叶，原139完整保留、八组非零、零skip/fail。[同Ptolemy代码R03](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/review-r03/review-report.md)确认R02两HIGH全部CLOSED/新增0，原R01四项关闭链保持；Main实际解析Reviewer普通/race各29独立探针全PASS并匹配五源。七项Manifest仍为2/0/0/0/0/0/0，源五fmt通过；[精确旧23证据格式处置](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/format-exception-main-r03.json)仅在本卡适用，根gofmt仍FAIL，不豁免其他检查/源码缺陷。[Kuhn规划及ledger核验](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/plan-r52-review-r01/review-report.md)有界PASS，原四卡/四批技术映射保持；本次仅Main结果回填，不把新metadata SHA说成重新技术受审对象。依用户standing采用建议授权及本卡局部关闭合同，Main完成G4-task核收并仅关闭S4.7；[最终现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/main-final-reception.json)保留精确版本和限制。上文R01/R02待核或OPEN均为当时失败链，不覆盖；不重开旧卡、不清理、不提交或部署，不启动下一卡。真实Journal持钥实现/SQL/恢复工具/root/PG/E2E未验证，design仍draft、人的精确接纳unknown，后继原责任与具名门槛保持。

<a id="task-identity-auth-s4-8"></a>

- [x] identity-auth-S4.8 把Recovery Journal构造/读取分成真正无副作用验证入口，按retained Begin key纯读验证候选secret，并迁移ctx写合同及Consume的fresh/publication受控调用。 开始@2026-10-29 完成@2026-10-29

- **类型/Story与范围**：开发（同卡局部验证）；identity S04。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §3.3.1/§4.2；identity TP §4.7.1:792–795,831。原r50及E690/E39受审技术SHA保留为原登记历史输入，不循环追新hash；原六稿PlanningReview R02不覆盖本次新增映射。本卡新增承接identity设计r52（SHA256 `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`，§3.3.1 ContextRecoveryJournal/Journal-owned verifier、§4.2 Resume、§6.1、§9.1；[r52完整差量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r52-author-r02/design-r51-to-r52.diff)）。本次由Main授权正式登记；r52设计R02独立PASS的有界范围、Main实际接收及F001 SOURCE/LOCAL来源见S4.7下方r52承接记录。设计仍draft、人的精确接纳unknown；原六稿PlanningReview不覆盖本次新增规划映射，本增量已由Kuhn独立规划Review有界PASS/新增0，Main现场接收见S4.7下方最新记录；新增验收全部NOT RUN，不覆盖原卡既有实际自测历史。
- **文件所有权**：仅以下新增/修改清单；实际owner为Zeno / add-backend-dev（实际Agent ID `01a18486-3e11-7852-9723-ad380e0bb392`），来源为Main实际resume原Zeno、依据用户standing采用建议继续授权；本次认领登记时Main尚未接收本卡G0结果或源码/LOCAL信号，本次只登记真实认领，不授源码/LOCAL或运行写权。
  - `internal/recovery/journal_snapshot.go`（新增）
  - `internal/recovery/journal_snapshot_test.go`（新增）
  - `internal/recovery/coordinator_context_test.go`（新增）
  - `internal/recovery/journal.go`（修改）
  - `internal/recovery/coordinator.go`（修改）
  - `internal/recovery/journal_test.go`（修改）
  - `internal/recovery/journal_windows_test.go`（修改）
  - `internal/recovery/filelock_flock.go`（修改）
  - `internal/recovery/filelock_unsupported.go`（修改）
  - `internal/recovery/filelock_flock_test.go`（修改）
  - `internal/identity/recovery_run_store_integration_test.go`（修改）
- **文件数**：11；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.2`
- **批次关联**：[B13B](../plans/ir-01-content-community-batches.md#b13b)；原批次安排保持，Main已实际派发上述owner，开始@2026-10-29；本次认领登记时Main尚未接收本卡G0结果或源码/LOCAL信号，当时仅只读G0；后续实施/局部运行须另获Main本卡精确SOURCE/LOCAL信号，不由认领或前置S4.7关闭推定开窗/测试PASS，S4.7格式例外不扩本卡。
- **验收命令**（根目录；原登记命令，实际执行与留存见本卡Main最终接收）：

```text
go test ./internal/recovery -list '^TestRecoveryJournalReadOnlyContext$'
go test ./internal/recovery -run '^TestRecoveryJournalReadOnlyContext$' -count=1 -json
go test -race ./internal/recovery -run '^TestRecoveryJournalReadOnlyContext$' -count=1 -json
go vet ./internal/recovery
go test ./internal/recovery -list '^TestRecoveryCoordinatorPublication$'
go test ./internal/recovery -run '^TestRecoveryCoordinatorPublication$' -count=1 -json
go test -race ./internal/recovery -run '^TestRecoveryCoordinatorPublication$' -count=1 -json
go vet -tags=integration ./internal/identity
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `CommittedReadOnly` | U/C | 已安装committed正控；构造/snapshot/Find路径无create/sync/CAS/repair |
| `PendingStageMissingAndBudget` | U/C | pending/孤立stage/缺介质或key/超限/未知版本关闭，不截前N条 |
| `ContextLockAndWriteUnknown` | U/race | nil/取消/锁等待，写ACK未知独立回读、已启动操作真join |
| `FreshPublicationAndConsumedWindow` | U/C | 受控Consume鲜读及publication绑定；唯一ready_to_consume+精确consumed补写，normal不补 |
| `RetainedBeginSecretReadOnlyVerifier` | U/C | 原journal_snapshot.go/journal.go实现VerifyBeginSecret(ctx,RunBinding,[]byte)，原journal_snapshot_test.go的TestRecoveryJournalReadOnlyContext承载：K1创建精确Begin后verify-only、K2 active仍按Begin的K1+version复用原fingerprintSecret/HMAC及constant-time比较，返回原committed JournalEntry；正确候选正控，错secret/run/backup/key/version/缺K1/未知版本负控，不拿active K2、caller fingerprint、VerifyAgainst或再次Begin替代。只有Journal持retained key，Runtime/root不复制算法或获得key |
| `VerifierSourceDriftAndZeroEffects` | U/C/race | 同top核完整链/Anchor/namespace/id/epoch/唯一begun/UTC微秒及reader-keyring绑定前后稳定；pending/不完整/来源revision ABA/取消/失锁全部拒绝，保留ErrAdmissionBinding/Incomplete/Unsupported/Drift及context/cause；成功与拒绝均零Begin/CAS/repair/mkdir/持久秘密/信任根写，buffer不保留不异步消费、日志错误零secret/key。原journal_snapshot_test.go/journal_test.go内按实际fixture职责实现，不新增文件 |

- **本任务关闭条件**：缺文件/pending/孤立stage/未知key/预算超限均关闭，constructor/read零create/mkdir/sync/CAS/repair；锁等待可取消；外部写ACK未知先回读，已启动动作真join；Consume仍只从identity_ready，fresh双读/精确publication绑定，唯一补写窗口不扩。 新增verifier正/负证据在本卡原11文件及上述既有top闭环，不以真实root/信任根SQL或S4.16/专项为关闭前置；只有完整committed精确Begin且retained key匹配、来源稳定和ctx有效才返回nil及原entry，不授许可。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。同Store真实PG、文章publication和全局restore顺序由后继组合；历史Read不能调用本受控Consume。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：旧原位签名所有真实消费者先rg复核；目前直连消费者集中journal_test/windows及Store integration测试。若出现新调用者不在此精确清单，停签名迁移而非加Background兼容包装。完整受控Consumer的定向top建议另在coordinator_context_test.go定义TestRecoveryCoordinatorPublication，并同样list/普通/race验证；本候选不能只验reader即关闭。 适用本节通用停止合同；未出现确认不当成认领/关闭。
- **Main当前续接（2026-10-29）**：[原G0](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-g0-r01/receipt.md)发现原Journal真实HMAC字段与admission中SHA-only格式校验不兼容，属静态证据，尚不是执行FAIL或独立finding。本卡11文件SOURCE/LOCAL暂未开放；Main按已确认原HMAC wire及用户standing建议值授权，另派原Zeno仅对admission.go/admission_test.go作[两文件有界兼容修复](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/compat-start-r01.json)，先compiled行为Red、保留230回归、同原Ptolemy复审后核解除本具体阻断，再开本卡原11文件。该补充变更有独立精确引用和所有权，不新增TaskID、不将本卡清单暗扩13、不改真实算法/前缀/全局SHA规则或未来环境门槛；旧B13A对象/失败及通过记录保持当时时点。

- **Main实施窗口更新（2026-10-29）**：兼容两源经[原Ptolemy独立评审](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/compat-review-r01/review-report.md)有界PASS/新增0，Main全文核报告与58叶独立普通/race实际raw后[接收该具体G0缺口](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/compat-main-reception-r01.json)，并已向原Zeno发送[原11文件SOURCE/LOCAL START](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-source-local-start-r01.json)。前文closed为当时时点；本次实际实施中，包含新增私有validateRecords消费者delta，不改冻结compat/HTTP两源，不扩清单或真实环境。Task仍[-]，本卡代码及验收/Review/全仓七项尚未完成，根fmt旧FAIL未继承例外。

- **Main代码R01接收与返工（2026-10-29）**：[原Ptolemy完整单卡R01](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-review-r01/review-report.md)为CHANGES REQUIRED，唯一HIGH `RECOVERY-B13B-S48-CR-R01-F001` OPEN：writer仅整次operation前后验锁，独立普通/race均复现失锁后仍写396bytes和第二CAS；各66独立叶64PASS/2行为FAIL，含组件各89叶87PASS/2FAIL。Main全文核报告/探针/receipt关键字段并实际解析raw、当前11源SHA一致，见[现场核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-review-main-r01.json)。原作者39/42/271及Main408普通/race、全仓六项通过和有限fmt处置均不关闭本HIGH。Curie跨卡R01为partial且已STOP、无Go/后台/待join，未冒完整结论。Main核15源仍冻结后采纳原作者四源修复计划，仅对原11内journal.go及三个既有测试发[四源SOURCE/LOCAL START](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-r02-source-local-start.json)，其余源保护；修后仍同Ptolemy复审，Task保持[-]，本卡/批次不接收。

- **Main代码R02接收及两源返工（2026-10-29）**：[原Ptolemy复审](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-review-r02/review-report.md)已独立确认R01失锁F001 CLOSED；新增HIGH `RECOVERY-B13B-S48-CR-R02-F001` OPEN：Anchor及retirement真实端口panic时跳过operation失效与锁释放，普通/race各2行为FAIL，原panic保持但Close=0、逃逸ctx仍可用。Main全文核报告/探针及实际raw，见[现场核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-review-main-r02.json)。R02当版[联合实测](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/main-joint-evidence-r02/index.json)四top普通/race各423叶PASS、全仓六项PASS、根fmt仍FAIL2且另有有限处置，不关闭代码HIGH。Main已向原Zeno发[精确两源R03 SOURCE/LOCAL](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-r03-source-local-start.json)，只改原11内journal.go/journal_snapshot_test.go，其他13源冻结；修复后仍由原Ptolemy复审、Curie补批次剩余评审，并对新SHA重核适用检查。两Task保持[-]、73/49/5/19不变，尚无本卡或批次最终接收。

- **Main批次R02阻断与R04返工（2026-10-29）**：原Ptolemy R03已独立关闭writer两项HIGH、各90叶普通/race全PASS，Main[现场核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-review-main-r03.json)。[Curie完整跨卡R02](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/batch-review-r02/review-report.md)新增HIGH `B13B-BATCH-CR-R02-F001` OPEN：纯reader withLock在实际Anchor.Load panic后跳过Close，Snapshot/VerifyBeginSecret各calls1/Close0；每模式自有9叶7PASS/2FAIL，非Root/SQL/E2E缺失。Main全文核报告/探针和实际raw，见[批次现场核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/batch-review-main-r02.json)。所有15源匹配且Reviewer STOP/Hostclosed后，Main向原Zeno发[仅原reader两源R04 SOURCE/LOCAL](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-r04-source-local-start.json)，其余13源冻结；修后由发现者原Curie复审，不重开未受影响writer关闭链。原R03全仓六项/428叶及有限fmt处置不关闭此新HIGH、不自动覆盖新SHA。两卡仍[-]，73/49/5/19不变，无最终Task/batch接收。

- **Main最终接收（2026-10-29）**：[原Ptolemy R03](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-review-r03/review-report.md)确认R02异常退出HIGH CLOSED、原R01失锁HIGH保持CLOSED、新增0，独立普通/race各90叶全PASS；Main全文核报告/探针并实际解析raw、核11源前后SHA，见[单卡现场核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s48-review-main-r03.json)。原11文件完成纯读/retained Begin key verifier、ctx writer及受控Coordinator职责，R03仅writer两源修复；后续R04仅原reader两源修复，原Curie已独立关闭 `B13B-BATCH-CR-R02-F001`，未变writer两项关闭保持；另有独立记录的compat两源补丁，不暗扩本卡清单。[Curie批次最终报告](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/batch-review-r03/review-report.md)与[Main批次现场核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/batch-review-main-r03.json)另完成跨任务机制/最小合成验证及覆盖、必要性能边界检查；不由两卡PASS相加推定整批。[本版Main联合实测](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/main-joint-evidence-r04/index.json)四top count1普通/race各446叶全PASS（77/42/271/56），原叶逐名保留、全仓六项PASS、15源fmt空输出PASS。根fmt仍FAIL2，旧23哈希/两raw同写前；[当版精确有限格式处置](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/format-disposition-main-r04.json)不改FAIL、不豁免代码/真实环境。Main核固定main/HEAD、staged0、完整差量/所有权/适用原requires及受保护基线无越界后，依用户standing建议值继续授权按本开发卡自身条件关闭，完成@2026-10-29。无Git提交约定或本轮Git写操作；不是人的业务签字、PG/SQL/nativePOSIX/真实provider/root/Case18/部署/E2E通过，后继原任务保持。以上旧NOT RUN/待Review/OPEN/窗口文字均为当时历史，当前以本条和证据精确范围为准。

<a id="task-identity-auth-s4-9"></a>

- [x] identity-auth-S4.9 在唯一RecoveryRunStore交付完整全量投影、revision/Digest与publication转换，并一次拥有0010全部新增元数据约束。 开始@2026-10-29 完成@2026-10-29

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 / F001 / F002。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §3.3.1/§4.2 F001/F002/0010；identity TP §4.7.1:790,799–820,847。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际owner为Zeno / add-backend-dev（实际Agent ID `01a18486-3e11-7852-9723-ad380e0bb392`），来源为Main实际恢复原Zeno、依据用户standing采用建议继续授权；本次认领登记时仅只读G0，尚未获Main本卡SOURCE/LOCAL/PG信号，不表示实施启动、G0或测试通过，不授源码或SQL准入。
  - `migrations/0010_recovery_runtime_metadata.sql`（新增）
  - `internal/identity/recovery_admission_store.go`（新增）
  - `internal/identity/recovery_admission_store_test.go`（新增）
  - `internal/identity/recovery_admission_store_postgres_test.go`（新增）
  - `internal/identity/recovery_run_store.go`（修改）
  - `internal/identity/recovery_run_store_integration_test.go`（修改）
- **文件数**：6；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.8`、`identity-auth-S1.6`
- **批次关联**：[B13C](../plans/ir-01-content-community-batches.md#b13c)；原批次安排保持，Main已实际派发上述owner，开始@2026-10-29；本次认领登记时仅只读G0，未获SOURCE/LOCAL/PG。后续实施及局部运行须Main本卡精确信号，不由认领、前置卡关闭或角色/命令推定授权或人的设计签字。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/identity -list '^TestRecoveryAdmissionStoreEncoding$'
go test ./internal/identity -run '^TestRecoveryAdmissionStoreEncoding$' -count=1 -json
go test -race ./internal/identity -run '^TestRecoveryAdmissionStoreEncoding$' -count=1 -json
go vet ./internal/identity
go test -tags=integration ./internal/identity -list '^TestRecoveryAdmissionStorePostgres$'
go test -tags=integration ./internal/identity -run '^TestRecoveryAdmissionStorePostgres$' -count=1 -json
go test -race -tags=integration ./internal/identity -run '^TestRecoveryAdmissionStorePostgres$' -count=1 -json
go vet -tags=integration ./internal/identity
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `CanonicalNullFieldsAndInitialProof` | U/C/P | 完整固定canonical、NULL≠空、UTC微秒、全部新增proof/计数/restore字段 |
| `CompleteRepeatableReadAndRowsError` | P | 全run短RR只读事务、独立连接、Rows.Err/缺表不伪空集合 |
| `RevisionMutationAndOverflow` | P | 实际相关写counter同事务递增、只读不增、非递归、溢出拒写 |
| `MetadataConstraintsAndLegacyUpgrade` | P | 0010 scope四绑定、proof同空/单调、review-operation组合FK、不可变；legacy不填成功 |

- **本任务关闭条件**：RR只读短事务/全run集合/Rows.Err/NULL/UTC微秒/固定canonical；counter覆盖run/restore/安装/review/operation/proof实际写；succeeded缺来源拒准入；0010包括installation/restore、run publication/initial proof、sessions scope、operation批准绑定、review指针、组合FK/不可变/计数溢出；旧行不伪回填成功，普通scope默认ordinary。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。安装、身份执行、F001/F002使用这些字段并各自验证业务事务；此卡不代完整恢复或schema生产部署。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：唯一0010 owner，不让后续三个业务候选再并发编辑migration。PG top应新建retention-only harness，覆盖0001–0009→0010、非法组合/错用户/FK、回滚及独立回读；不运行旧自动DROP harness。旧0005/0008不改；编号若占用交Main串行重分配并核引用。 适用本节通用停止合同；未出现确认不当成认领/关闭。

- **Main G0受理（2026-10-29）**：[实际核收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/g0-main-reception-r01.json)确认原六源内业务方案可行；另发现三历史fixture从活动迁移目录加载却要求总数9，静态兼容GAP不是执行FAIL。Main按建议单独授权原Zeno[三测试文件兼容修订](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/compat-source-local-start-r01.json)，显式冻结历史0001–0009，不改旧业务断言；新增0010升级由本卡新P完整验证。该补充有独立所有权/评审，不新增Task或暗扩本卡六源；原六源SOURCE/LOCAL/PG暂未开，原规划已由[Main精确复用核实](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/planning-applicability-main-r01.json)，旧pending为历史，非缺Gate。

- **Main原六源实施开始（2026-10-29）**：三历史fixture兼容修订经[原Ptolemy独立Review](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/compat-review-r01/review-report.md)有界PASS/新增0，普通/race各24叶含独立6，Main全文及实际raw/三源SHA核收，见[兼容缺口现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/compat-review-main-reception-r01.json)。Main据有效原G0向原Zeno发[原六文件SOURCE/LOCAL/隔离PG START](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/s49-source-local-pg-start-r01.json)，仅新retention-only PG top/新本机身份测试库，旧autoDROP及原业务PG top不得运行。前文closed为当时点，不等于本卡测试/代码Review/关闭已完成；原六源与schema义务不扩。

- **Main task-level 接收与关闭（2026-10-29）**：原Zeno六源已STOP/冻结，原Ptolemy[独立代码Review](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/s49-review-r01/review-report.md)有界PASS/finding0，Main[逐项核收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/s49-review-main-reception-r01.json)保持。Main随后在当前Runtime D404/同六源重跑编码41及真实PG26 ordinary/race，含真实Store→公开Verify的关闭负控/全来源internal_ready正控且Acquire拒绝；[当前联合现场证据](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/main-current-joint-r01/receipt.json)全部required零fail/skip/unfinished，源19/保护对象前后相同，Main status/diff/HEAD/staged/越界核对完成。Manifest六项PASS，根fmt仍FAIL2但stdout/stderr与旧23基线字节同，18Go源fmt空；Main按standing授权采用[本次精确八源任务有限格式处置](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/root-format-disposition-before-c1-r01.json)，未记全七PASS、未改旧证据。仅本卡局部技术关闭，不签B13C整批/Article C1/完整安装/root/native/E2E/容量或人的接纳；旧linker OSjoin限制单列保留。旧pending/认领原文均保留各时点。

<a id="task-identity-auth-s4-10"></a>

- [x] identity-auth-S4.10 交付running阶段身份终结/review/canonical/幂等核对与条件identity_ready，并承接succeeded退出撤维护会话的同Store唯一执行者。 开始@2026-10-29 完成@2026-10-30

- **Main本卡最终接收（2026-10-30）**：原作者Zeno原3源修复后，非作者Confucius R02独立关闭F001 HIGH/F002 MEDIUM、CodeGate PASS/new0；最终6D80/34EB/659E三SHA，U77/P86普通race及原10＋合法2探针普通race通过。真实原domain failed/source/cohort、canonical/初始强改密/完整Complete和退出Audit/实际Session集合闭环；Current77/86六组及Main当前版复核见原始记录。 [本卡独立评审核收](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-code-review-main-reception-r02.json)；[当前三卡Main现场检查](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/main-current18-verification-r02.json)与[精确302个新增schema实际SQL](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/main-current18-physical-final-r02.json)均已核收。全仓六项PASS，根gofmt仍为原23片段完全同hash的FAIL2，当前17份Go格式空输出PASS，采用仅本18源的有限技术处置，不扩豁免或清理。非作者Dirac另完成[本批跨卡整体G4评审及Main现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/b13d-batch-review-main-reception-r01.json)，接口/来源/迁移/资源/必要集成、性能维度与覆盖按报告实际范围核收，不是三卡PASS相加。Main复核固定main/HEAD、staged0、diff/所有权/当前前置与本卡关闭条件，按task-execution-workflow §10和用户standing建议值继续授权作技术接收并关闭本开发卡，完成@2026-10-30；无Git提交约定/本轮Git写。旧G0-only、NOT RUN、待Review/OPEN等均为原时点，当前以本条和精确证据为准。设计draft/人的业务接纳仍unknown；真实provider/ACL/审批主体、SMTP/native/POSIX、完整root/S4.13绑定/真实恢复/Case18/部署/E2E由原后继任务分别保留，不据此冒PASS、不重开或关闭其他卡，原phone/缺失行限制及ADR-0024/0025窄边界不扩。

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 / S13–S16。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §4.2 running/Complete及S13–S16；identity TP §4.7.1:804,816,820,831,833。原r50及E690/E39受审技术SHA保留为原登记历史输入，不循环追新hash；原六稿PlanningReview R02不覆盖本次新增映射。本卡新增承接identity设计r52（SHA256 `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`，§3.3.1 IdentityRecoveryExecutor、§4.2 RecoveryExit同Store收口、§6.1、§9.1；[r52完整差量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r52-author-r02/design-r51-to-r52.diff)）。本次由Main授权正式登记；r52设计R02独立PASS的有界范围、Main实际接收及F001 SOURCE/LOCAL来源见S4.7下方r52承接记录。设计仍draft、人的精确接纳unknown；原六稿PlanningReview不覆盖本次新增规划映射，本增量已由Kuhn独立规划Review有界PASS/新增0，Main现场接收见S4.7下方最新记录；新增验收全部NOT RUN，不覆盖原卡既有实际自测历史。
- **r56/TP26正式增量映射（2026-10-30，Asia/Shanghai）**：消费identity design r56 SHA256 `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF` §4.2 ReadFailed/原域failed/Recover-Audit/cohort/ACK/Complete、§6.1/§9.1；[原Dirac两轮Design Review](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r56-review-r01/review-report.md) SHA `7C271911A16D0FC1860DF15E0362FC97FF1D6E71FD9F6D8172114D60D308C934`为有界PASS/new0，[Main实际采用](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r56-main-adoption-r01.json)仅技术合同。实际消费原Jason冻结[identity TP26六行增量](../design-workspace/identity-auth/test-plan.md#b13d-s410-r56-r26) SHA `34826759C79A58EB0B4A296D2DD5C9157014406859153DC6CA0FA6E2BD921DAB`，[作者receipt](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r26-author-r01/receipt.json)与[Main冻结](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r26-main-freeze-r01.json)为作者输入，不代独立Review。按[Main直接两SSOT写窗](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/planning-r56-write-start-r01.json)正式登记；原r55/r27及joint4mapping独立PASS/Main已核收事实仅复用旧范围，旧pending文字按原时点保留。本r56差量仍待原Dirac联合TP26＋Tasks/Plan独立mapping Review/Main接收，不增TP前审Gate。原claim日期/责任不改，draft/人的接纳unknown保持，新增验收全NOT RUN；Temp完整3源/typed内部factory仅未编译输入，不记codeReady、实现/测试PASS、源码准入或Taskclose。
- **r55/TP r25本次正式技术映射（2026-10-29）**：identity design r55 SHA256 `419B800DE82405CE0D24ED0E9FCE3788D112E298F14C89220F49C4D313029DE2`，§4.2 D0/ND0终结/共同C逐P/F002资格及terminal只读重放、§6.1/§9.1；[原Dirac两轮Design Review](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r55-review-r01/review-report.md) SHA `B6F4FE4326EB39C3A26EF03E245E10A58FA22117CC976ADA7407FFF1847B4F15` PASS/new0，[Main采用](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r55-main-adoption-r01.json)已接收。实际消费原作者Jason冻结[identity TP r25新增块](../design-workspace/identity-auth/test-plan.md#b13d-s410-r55-r25) SHA `5AEA4284B0E903A3E12FE4065E53778A252B0EC006F8E91E854870AF156FAD57`，仅9行集中风险与原S4.10/S4.12映射；[TP作者receipt](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r25-author-r01/receipt.json)及[Main输入冻结](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r25-main-freeze-r01.json)不代独立Review。Main依用户standing授权[本次两SSOT正式登记](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/planning-threecard-write-start-r01.json)；冻结TP可作作者规划输入，本次两TP＋Tasks/Plan随后统一独立mapping Review/Main接收，当前PENDING，不额外增加单独TP前审Gate。旧r50/E690/旧Review按原适用范围保留，不循环追hash；Jason原作者归属、draft/人的接纳unknown不变，新增验收全NOT RUN，源码Temp候选不记已实现/测试PASS，无SOURCE/LOCAL/PG或checkpoint。
- **文件所有权**：仅以下新增/修改清单；实际owner为Zeno / add-backend-dev（实际runtime Agent ID `01a18486-3e11-7852-9723-ad380e0bb392`），来源为Main实际恢复原Zeno/add-backend-dev并派发本卡只读G0，依据用户推荐值继续授权由Main真实派发，不是用户亲指定Role或人的设计签字。本次认领登记时Main尚未接收本卡G0结果或源码/LOCAL信号，仅只读G0准备，SOURCE/LOCAL/Go/PG均closed；不表示实施启动、G0/测试通过，不授源码、运行或SQL准入。
  - `internal/identity/recovery_execution.go`（新增）
  - `internal/identity/recovery_execution_test.go`（新增）
  - `internal/identity/recovery_execution_postgres_test.go`（新增）
- **文件数**：3；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.9`、`identity-auth-S9.1`、`article-management-S6.1`
- **批次关联**：[B13D](../plans/ir-01-content-community-batches.md#b13d)；原批次安排保持，Main已实际派发上述owner，开始@2026-10-29。本次认领登记时仅只读G0准备、Main尚未接收本卡G0结果，SOURCE/LOCAL/Go/PG均closed；后续权限须Main另发精确信号，不由认领/已闭前置或旧规划Gate推定。设计draft/人的精确接纳unknown及原技术Review覆盖保持，本次仅认领事实登记。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/identity -list '^TestRecoveryIdentityExecution$'
go test ./internal/identity -run '^TestRecoveryIdentityExecution$' -count=1 -json
go test -race ./internal/identity -run '^TestRecoveryIdentityExecution$' -count=1 -json
go vet ./internal/identity
go test -tags=integration ./internal/identity -list '^TestRecoveryIdentityExecutionPostgres$'
go test -tags=integration ./internal/identity -run '^TestRecoveryIdentityExecutionPostgres$' -count=1 -json
go test -race -tags=integration ./internal/identity -run '^TestRecoveryIdentityExecutionPostgres$' -count=1 -json
go vet -tags=integration ./internal/identity
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `FinalizeOnceAndAtomicAudit` | P | 原TestRecoveryIdentityExecutionPostgres以真实合法D0（immutable review.backup_status=deleted）/ND0（active或suspended）核终结同事务：D0保status=deleted/原非NULL deleted_at/NULL contacts/原retained与既有标识墓碑，ND0冻结suspended且保contact/identifier/deleted_at=NULL；两者仍恢复锁定、撤原认证/全部Session/临时凭据/processing-uncertain幂等能力，完整pending review/AccountCount/实际失效计数/Audit均不漏，NULL digest不虚增。精确已提交终结ACK重试在后建initial信任根/新凭据后零再次freeze/清新凭据/重建review，原source/audit/count/binding核证不减；r56另核原Finalize同一tx终结Audit中processing/uncertain实际转review_required的原六字段精确cohort：四digest/locator固定hex、原key版本、正int64 generation规范十进制string，按scope_hash/key_locator/generation排序结构数组，与条件转换逐项相等、无重漏且等于原IdempotencyCount，绑定同install/run/source；计数及generation含>2^53边界不经JSON float64。合法空集合须原终结证明；既存等价不可变枚举可复用，不能只count/当前剩余行反造；审计/转换/commit失败全回滚，精确已提交重试不重建集合/重冻后继新凭据；旧无集合且无独立等价来源保持关闭 |
| `UserStoreReviewLockOrder` | U/P | 确定User→Store→review；Complete持Store不反向锁User；r56 ReadFailed仅在actor User→Store→record短预读释放后于事务/Runtime mutex外读取独立原件，started源I/O/子任务全真join且保errors.Is/原cause；最终仍按确定actor/target User→Store→适用operation/credential/Session→record/Audit，不倒锁。consumer一次UTC微秒Now及实际actor构造原RecoveryFail，Now不早于record创建/更新、run开始、initial强改密及ApprovedAt；写前复核ctx/rawpermit/独占、同Store/六字段/state与LockedUntil=NULL，原Recover和唯一resolution Audit同caller-owned tx原子。Audit精确绑定来源/原意图/原域及actor/时间，新增int64 metadata用规范十进制string、时间用UTC微秒，零rawScope/请求/快照全文/秘密；行数或Audit失败回滚，不改created/expires/generation/TTL。Complete源读取在事务外，Store锁内仅核DB/revision，无反向User锁/源I/O |
| `KeptDeletedCanonicalAndDenied` | P | 原PG top核ND0/D0 kept_deleted均消费原caller-owned完整canonical A/B＋Article/通知/Session/凭据/标识/retention/Audit，成功后才记kept_deleted；零/多/部分文章处置与回滚，D0原deleted_at不改，合法retention清理不把旧retained值复制回。denied须明确review/evidence/拒绝Audit：ND0仍suspended、deleted_at=NULL；D0仍合法deleted墓碑/原时间；两者recovery_required=true/recovery_locked、credential_reset_at=NULL，不发凭据/清Gate、不把denied冒canonical或免审查。PathB仍仅核实际存在适用行，不能声称原phone值/缺失行已证或扩大旧validator残余边界 |
| `IdempotencyUnknownAndCompleteProof` | P | 原PG top核共同C（同install/run/backup/running、有效许可/closed/exclusive、终结source/count/Audit、完整原账号含D0与唯一review、terminal审核元数据/证据/Audit、独立initial强改密证明、原幂等/被引candidate-operation-claim-job合法收敛）且每目标满足r55 P才identity_ready。混合D0 denied/kept、ND0 denied/kept/restored-active/restored-suspended真实正向：restored须精确F002 operation/credential/claim-job/User/review/Audit整组与谱系、可信role-status及非NULL credential_reset_at；denied/kept仍L且credential_reset_at=NULL，不要求全flag=false；缺漏review/Rows.Err/仅标签/错source/deleted_at/缺Audit/未决结算或异常D0-restored负向。ND0合法failed/invalidated restoration_operation_id历史保留并核无未决副作用，不盲NULL/清unknown-hold-deadletter。ACKloss核完整原事实不盲重写；Store短事务内无反向User/review锁/外部I/O，只identity_ready不替Consume/PublicReady；F002正向可用合法持久结算fixture，不倒挂未交付S4.12/root；r56原PG top须真实公开Runtime rawpermit/PhaseContext＋sameStore/Begin/running/强改密initial授权，ReadFailed的PermitBinding仅lookup，不授grant。固定私有reader返回owned RecoveryFailedEvidence（bytes/Body/FieldErrors深拷贝），Version=han-recovery-idempotency-failed/v1、ref及64位小写hex独立既存批准原件digest、install/run/backup/批准ActorID/原Scope/ApprovedAt全绑定；原scope-domain＋8byte大端长度前缀SHA256重算，原六字段(scope_hash,key_locator,key_digest,digest_key_version,request_digest,execution_generation)精确匹配，四digest/locator32bytes、版本非空、generation正int64；来源Action仅Fail、ReviewedBy空及Now/LockedUntil/ExpiresAt零，批准时间UTC微秒/不晚于consumer Now且符原事件，不强加晚于Begin。原域实际replay核创建v1的409标识冲突/503邮件未受理两失败及legacy newadmin resend精确封闭投影；linked durable域只沿原classifier核operation/claim/exact credential/delivery/settlement/job/lineage，已结算只读。错wrapper/Scope/六字段/refABA/用途/结果/时间、成功/generic放弃/混合额外null快照均零Recover/Audit/SMTP，账号批准reader不能替源，unknown/保护窗/hold/dead-letter不清，不授generic Retry。实际COMMIT后ACKerror须writer/tx真结束，既有ACK owner用另一物理same-pool连接及受控预算，事务外重ReadFailed同ref/digest，再核failed行完整snapshot/code/六字段/actor时间/唯一resolution Audit和原字段；全匹配才204，terminal同意图授权后只读原204、零新Recover/Audit/凭据/时间覆盖，异意图或不明503。只writer已join且独立原review_required＋无该Audit明确证明未提交才原意图续接，不盲写。Complete逐原cohort核合法原域终态/精确来源/响应/actor时间/原子Audit，缺删行/重漏/改snapshot/仅count0/未决/后继新行凑零/不完整generation链均closed；合法空必须原枚举，已授权generation+1只由原owner证连续无ABA及实际完成，新processing不算Complete；源不可读或读间run/revision/cohort/来源变化重核或closed。保持r55全部C/逐P与账号/F002/job Gate，仅identity_ready，不授公共开放 |
| `SucceededRecoverySessionExitAtomic` | U/P | 原recovery_execution.go的同executor/同RecoveryRunStore receiver实现RevokeRecoverySessions(ctx,permit,binding)，原recovery_execution_postgres_test.go的TestRecoveryIdentityExecutionPostgres下用真实PG验证：只接受同实例有效replay+权威succeeded/同run/backup/install/强改密与不可变来源，User确定次序→Store run→Session/Audit锁序；复核完整账号集合，撤本run/安装全部未撤维护Session含旧RuntimeID/代际及已过期行，同run错安装拒绝；ordinary Session/密码/auth generation/强改密证明/run outcome不变；撤销与recovery_maintenance_exit审计同事务，零会话首次亦审计、失败全回滚 |
| `RecoveryExitAuditAckReadback` | U/P | 同一既有PG top覆盖真实commit丢ACK、确定未提交/读失败/部分事实/失权对照；独立连接有界只读核权威阶段/同run安装/完整退出审计/目标未撤维护Session=0才nil；同绑定审计+零残留精确幂等，审计存在但残留或错绑closed；unknown先读不盲写。原recovery_execution_test.go的TestRecoveryIdentityExecution承载classifier/拒绝，SQL/锁/回滚不得仅U自证 |

- **本任务关闭条件**：同run/restore receipt/Begin绑定；终结/reviews/失效/计数/audit同事务，重试不再次清新密码；User→Store→review锁序；kept_deleted调用现有canonical并验证全证据；denied受限；unknown幂等不假清；Complete查询所有review、initial changed proof、operation/审计，缺任一不identity_ready。 新增succeeded/replay分支与原running职责分开校验，不扩大running-only身份写权限；在本卡原3文件/既有U和PG top、已完成前置及合法持久fixture内完成撤销/审计原子性、全代际集合与独立ACK回读。测试从S4.7公开producer取得中性许可，不调用私有phaseLocked；可用同域合法succeeded/Session/Audit fixture，不等待后继S4.11/12/16或真实complete/root实施，不借fixture声称完整退出R。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。 r55本次新增关闭条件：按D0/ND0保合法原状态/墓碑、完整失效/review/计数/Audit及零二次终结；kept_deleted完整canonical、denied合法受限分支与共同C∧完整目标逐项P须有本卡真实PG正负/混合终态及ACK证据。保持原3源/6组/原最低层；锁序与两succeeded退出组逐字不改，S4.12/root实际未交付不成为本卡局部反向前置，原phone/缺失行能力限制不宣称消除。 r56增量在原3源/6组/原命令与最低P内闭环：同tx精确原cohort、源包装/原域真实失败正负、事务外核源/原锁序、Recover＋Audit原子及独立物理ACK、terminal零新写/完整Complete集合均留本卡真实PG证据；其他三组/r55已定义验证不减，不新DTO/DDL/authority/genericRetry或第四源。实际provider/批准主体/ACL未知仅闭受影响resolve/有该未决cohort的Complete；独立checkpoint fixture可证软件，不作生产fallback、不将未交付S4.16或真实provider倒挂本卡/Article关闭，不扩normal/verify未用来源要求。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。F001/F002实现正向信任根/恢复目标，root按同一executor调用；完整全链由联合软件验证，原真实S4.4另验。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：在新文件内实现原RecoveryRunLocker/RecoveryOpenGate消费者适配，复用deletion.go现有能力/全事务，不重写canonical、不重开S9.1。Complete可用合法持久fixture独立测试，不等待全部root已实施；来源fixture不冒真实批准。 适用本节通用停止合同；未出现确认不当成认领/关闭。

<a id="task-identity-auth-s4-11"></a>

- [x] identity-auth-S4.11 维护Session隔离及initial信任根准备、G+1强改密、ACK回读和撤维护会话组成一个完整认证闭环。 开始@2026-10-30 完成@2026-10-30

- **Main本卡最终技术接收（2026-10-30）**：原Lorentz原12源实施与作者自测已核收；非作者Confucius R02独立CodeGate PASS、原旧Cookie复登录HIGH独立CLOSED，119风险leaf/模式及同修订实际SQL已由Main[正式接收](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/s411-code-review-main-reception-r02.json)。维护会话隔离、initial信任准备/G+1强改密/双Audit/ACK回读及撤维护闭环按本卡自有合同成立。原Dirac另完成两轮跨卡来源/状态/迁移/authority/资源与必要性能覆盖核对，[整体G4 R02及Main现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/batch-g4-review-main-reception-r02.json)为有界PASS/new0，非单卡PASS相加；组合发现的HIGH RECOVERY-B13E-G4-R01-F001已由原Zeno在另授shared identity三源内仅改nil-safe time.Equal，真实时刻/微秒/nil/角色状态/集合校验保留，原Reviewer独立CLOSED。原两叶字节/UTC不变＋10独立控制普通/race各12通过，24新增schema整像另经Main实际SQL核收。[Main当前20源/6文档的8项检查](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/main-g4-f001-current26-receipt-r01.json)实际unit11/PG3普通race、owned20fmt空、identity integration vet/build通过，另6schema SQL与逐raw/生命周期核对完成；[此前Main23](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/main-current23-receipt-r01.json)全仓六项结果保留原修订范围，未称全历史新修订重跑。根fmt仍实际FAIL2，原23输入和stdout/stderr完全同hash，采用[仅当前20源精确有限处置](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/root-format-current20-disposition-r01.json)，不记全7PASS或继承blanketwaiver。所有本批已执行进程/回调/Rows/Tx/pool真join，[Main专用PG服务与helper实际exit0](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/main-own-pg-stopped-r01.json)、全部raw/cache/schema/data保留。Main核实际diff/status/固定main与HEAD/staged0、文件边界/本卡前置及关闭条件后，依据task-execution-workflow §10和用户standing采用建议继续授权作技术接收并关闭本开发卡，完成@2026-10-30；无Git提交约定或Git写/清理。设计draft/人的精确业务接纳仍unknown；真实provider/ACL/审批主体/SMTP/native/完整root/真实恢复/Case18/部署/E2E/容量由原后继分别保留，不冒通过；ADR0024与S3.3/S3.5窄边界不扩、非PASS，S3.6不变。旧NOT RUN/待Review/窗口/OPEN字样为当时时点，当前以本条及精确证据为准，不关闭其他任务。

- **B13E实际认领（2026-10-30）**：owner Lorentz / add-backend-dev，实际runtime Agent ID `01a189aa-4e63-7d10-90c1-561fc6d70e8a`；由Main依据用户standing采用建议继续授权实际Host spawn并派发，非用户亲指定Role或人的设计签字。开始@2026-10-30。认领时仅只读G0及本人D:/Temp候选，SOURCE/Go/PG未开；先待S4.13稳定及Main后续明确窗口，不将该资源/串行安排改写为新requires。未宣称G0结果、实施或测试PASS。旧“尚未派发/未认领”“状态仍[ ]”及mapping待审为原规划时点历史；本次仅登记实际认领，既有六稿joint Planning R02及C1完整五源executor映射PASS/Main接收的适用范围保持，不因认领重开Planning Gate。设计draft/人的精确接纳unknown不变，不代表代码、测试或新PG通过。

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 / F001，必要S03和改密接缝。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §4.2 F001；Story S04；identity TP §4.7.1:799–806。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际执行者尚未派发/未认领，当前规划登记不是代码写权。
  - `internal/identity/recovery_auth.go`（新增）
  - `internal/identity/recovery_auth_test.go`（新增）
  - `internal/identity/recovery_auth_postgres_test.go`（新增）
  - `internal/session/maintenance_store.go`（新增）
  - `internal/session/maintenance_store_test.go`（新增）
  - `internal/identity/login.go`（修改）
  - `internal/identity/bootstrap.go`（修改）
  - `internal/identity/service.go`（修改）
  - `internal/identity/login_test.go`（修改）
  - `internal/identity/bootstrap_test.go`（修改）
  - `internal/session/store.go`（修改）
  - `internal/session/store_test.go`（修改）
- **文件数**：12；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.10`、`identity-auth-S4.9`、`identity-auth-S4.7`、`identity-auth-S4.1`、`identity-auth-S10.3`
- **批次关联**：[B13E](../plans/ir-01-content-community-batches.md#b13e)；由本pass Batch-Plan消费原任务安排，实际执行者未派发/未认领，状态仍[ ]。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/identity -list '^TestRecoveryMaintenanceAuthentication$'
go test ./internal/identity -run '^TestRecoveryMaintenanceAuthentication$' -count=1 -json
go test -race ./internal/identity -run '^TestRecoveryMaintenanceAuthentication$' -count=1 -json
go vet ./internal/identity
go test -tags=integration ./internal/identity -list '^TestRecoveryMaintenanceAuthenticationPostgres$'
go test -tags=integration ./internal/identity -run '^TestRecoveryMaintenanceAuthenticationPostgres$' -count=1 -json
go test -race -tags=integration ./internal/identity -run '^TestRecoveryMaintenanceAuthenticationPostgres$' -count=1 -json
go vet -tags=integration ./internal/identity
go test ./internal/session -list '^TestRecoveryMaintenanceStore$'
go test ./internal/session -run '^TestRecoveryMaintenanceStore$' -count=1 -json
go test -race ./internal/session -run '^TestRecoveryMaintenanceStore$' -count=1 -json
go test ./internal/session -count=1 -json
go test -race ./internal/session -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `ScopeAndPrivateBinding` | U/C/P | ordinary四恢复字段NULL、maintenance四非空正代际；私有phase绑定 |
| `BootstrapOnlyMePasswordAndDailyReject` | U/C/P | initial true/bootstrap只me/password；普通false+ordinary、CSRF/expiry/lineage/Gate/非initial拒绝 |
| `PrepareAndChangeAtomicGPlusOne` | P | User/run proof/G+1/false flag/两Session/审计整组提交或回滚 |
| `PrepareChangeAckLossAndResume` | P | 真实提交丢ACK、部分未知；已提交零再写密码，新密码维护登录，旧runtime拒绝 |
| `RevokeMaintenanceBeforeOrdinaryLogin` | P | 撤本run全部维护Session+切换Audit确认；旧cookie不升级，普通重新登录 |

- **本任务关闭条件**：true/bootstrap仅同run/install/runtime/gen初始账号到me/password；ordinary false+scope拒维护token，CSRF/expiry/Gate/lineage逐项有效；准备只一次；强改密User/run proof/G+1/false flag/audit/Session rotate全有全无；ACK loss已提交不再改密，cookie丢失用新密码维护登录；Resume不覆盖；切普通先撤全部维护Session，旧runtime和cookie不升级普通权限。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。维护surface的真实router/auth接线、真实切流和生产批准由唯一root/联合软件验证及后续真实层。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：同一会话表/pool，MaintenanceSessionAuthority归identity；普通Create/Rotate固定ordinary。identity主top与session的TestRecoveryMaintenanceStore分包精确执行，真实Session SQL/改密/撤销原子性在本卡P闭环；唯一production root的R认证/切流接入由S4.16后继核，不反向阻止本卡局部关闭。 适用本节通用停止合同；未出现确认不当成认领/关闭。

<a id="task-identity-auth-s4-12"></a>

- [x] identity-auth-S4.12 在同一durable协议内交付可信批准下的恢复candidate、唯一投递及无Mailer的原子restored结算。 开始@2026-10-30 完成@2026-10-30

- **B13F本卡技术关闭（2026-10-30）**：Lead/Main按用户standing采用建议继续授权与task-execution §10核收原Lorentz十文件实现及三文件最小返工；原非作者Confucius独立R02 CodeGate与实质B13F整体G4有界PASS，HIGH S412-ICR-R01-F001独立CLOSED/new0，账号全用途序列/原历史/唯一约束/溢出保护保持。[正式独立报告与Main接收](../../../../.spec/wip/recovery-runtime/evidence-b13f-r01/s412-independent-R02-formal-main-reception-r01.json)绑定最终source10；独立普通/race各probe2/U41/P55/Auth42共280真实PASS及198新schema/4668表像，原七风险最低层均覆盖，不以数量相加代G4。Main全文实差量/报告与正式包SHA、[共同版工程检查](../../../../.spec/wip/recovery-runtime/evidence-b13f-r01/s412-Main-final-common-checks-r02.json)、6history/8spot/4独立probe实际SQL和源/分支/所有权/Git交付diff已核：mod/vet/build/contract/race/tests及相关30源fmt均通过；rootfmt仍原23 FAIL2，按[当前30源有限处置](../../../../.spec/wip/recovery-runtime/evidence-b13f-r01/root-format-current30-disposition-r01.json)保留，goimports/golangci不可用不签PASS。独立物理核验中Host/测试库中断的旧physical Wait/Job/EOF仍unknown；同一保留数据恢复后25个独立PSQL批次完整补证，不重跑已绿测试或补签旧收据。[当前ownPG停服核收](../../../../.spec/wip/recovery-runtime/evidence-b13f-r01/own-pg-current-recovered-stopped-r01.json)为新supervisor44359/server14240真实exit0，所有data/raw/schema/cache保留，无清理/Git写。仅本开发Task及本批技术接收；设计draft/人的精确接纳unknown、真实批准provider/主体/ACL/SMTP/native/production-root/部署/E2E与原后续专项保持独立，S3.3/S3.5非PASS窄边界及S3.6不变。旧G0/owner/pending/NOTRUN注记仅保留形成时点，本次不是新Task/依赖/设计或人的代签。

- **B13F实际认领（2026-10-30）**：Main依据用户standing采用建议继续授权实际恢复原Lorentz / add-backend-dev，runtime Agent ID `01a189aa-4e63-7d10-90c1-561fc6d70e8a`，承接原10文件本卡。当前仅只读G0及本人独有D:/Temp候选，SOURCE/Go/test/gofmt/PG/SQL均CLOSED，Main后续精确窗口另授；认领不冒G0结果、实现/编译/RedGreen或验证通过。原8个requires已逐卡[x]；S4.11/S4.13已按原组件＋独立整体G4/Main现场核证技术关闭，B13E时间事实比较HIGH已独立CLOSED/new0，其收口不代本卡新实现或批准源/真实SMTP。旧未派发/未认领/NOT RUN与mapping pending为当时记录；本次只登记真实owner/开始日，不新建Task/依赖/命令/技术默认、不重开已完成的技术mapping Gate。现役r56/TP26及原技术Planning接收继续按实际范围适用；设计draft/人的精确接纳unknown保持，不代签。

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 / F002，必要Reset接缝。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §4.2 F002；Story S04共享恢复；identity TP §4.7.1:810–820。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **r55/TP r25本次正式技术映射（2026-10-29）**：identity design r55 SHA256 `419B800DE82405CE0D24ED0E9FCE3788D112E298F14C89220F49C4D313029DE2`，§4.2 D0/ND0终结/共同C逐P/F002资格及terminal只读重放、§6.1/§9.1；[原Dirac两轮Design Review](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r55-review-r01/review-report.md) SHA `B6F4FE4326EB39C3A26EF03E245E10A58FA22117CC976ADA7407FFF1847B4F15` PASS/new0，[Main采用](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r55-main-adoption-r01.json)已接收。实际消费原作者Jason冻结[identity TP r25新增块](../design-workspace/identity-auth/test-plan.md#b13d-s410-r55-r25) SHA `5AEA4284B0E903A3E12FE4065E53778A252B0EC006F8E91E854870AF156FAD57`，仅9行集中风险与原S4.10/S4.12映射；[TP作者receipt](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r25-author-r01/receipt.json)及[Main输入冻结](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r25-main-freeze-r01.json)不代独立Review。Main依用户standing授权[本次两SSOT正式登记](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/planning-threecard-write-start-r01.json)；冻结TP可作作者规划输入，本次两TP＋Tasks/Plan随后统一独立mapping Review/Main接收，当前PENDING，不额外增加单独TP前审Gate。旧r50/E690/旧Review按原适用范围保留，不循环追hash；Jason原作者归属、draft/人的接纳unknown不变，新增验收全NOT RUN，源码Temp候选不记已实现/测试PASS，无SOURCE/LOCAL/PG或checkpoint。
- **文件所有权**：仅以下新增/修改清单；实际执行者尚未派发/未认领，当前规划登记不是代码写权。
  - `internal/identity/recovery_password_reset.go`（新增）
  - `internal/identity/recovery_password_reset_test.go`（新增）
  - `internal/identity/recovery_password_reset_postgres_test.go`（新增）
  - `internal/identity/model.go`（修改）
  - `internal/identity/password_reset.go`（修改）
  - `internal/identity/password_reset_store.go`（修改）
  - `internal/identity/password_reset_operator.go`（修改）
  - `internal/identity/password_reset_test.go`（修改）
  - `internal/identity/password_reset_store_test.go`（修改）
  - `internal/identity/password_reset_operator_test.go`（修改）
- **文件数**：10；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.11`、`identity-auth-S4.10`、`identity-auth-S4.9`、`identity-auth-S10.1`、`identity-auth-S10.2`、`identity-auth-S10.3`、`identity-auth-S10.4`、`identity-auth-S11.1`
- **批次关联**：[B13F](../plans/ir-01-content-community-batches.md#b13f)；由本pass Batch-Plan消费原任务安排，实际执行者未派发/未认领，状态仍[ ]。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/identity -list '^TestRecoveryRestoredPasswordReset$'
go test ./internal/identity -run '^TestRecoveryRestoredPasswordReset$' -count=1 -json
go test -race ./internal/identity -run '^TestRecoveryRestoredPasswordReset$' -count=1 -json
go vet ./internal/identity
go test -tags=integration ./internal/identity -list '^TestRecoveryRestoredPasswordResetPostgres$'
go test -tags=integration ./internal/identity -run '^TestRecoveryRestoredPasswordResetPostgres$' -count=1 -json
go test -race -tags=integration ./internal/identity -run '^TestRecoveryRestoredPasswordResetPostgres$' -count=1 -json
go vet -tags=integration ./internal/identity
go test ./internal/identity -run 'TestPasswordReset|TestTemporaryCredentialResend' -count=1 -json
go test -race ./internal/identity -run 'TestPasswordReset|TestTemporaryCredentialResend' -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `TrustedApprovalAndDailyRejection` | U/C/P | run/install/actor/target/role-status/ref-digest逐字段可信绑定，缺源零secret/claim/SMTP；新candidate及settlement写分支同时核immutable review来源非D0与current target未deleted，D0/current-deleted在新secret/ID/claim/SMTP或清Gate前零副作用，不借批准/body role-status创造undelete或先改suspended洗来源；合法ND0原F002正向及普通三个拒绝不放宽。合法terminal同意图先按原不可变绑定/幂等结果分类，不能提前套用pending资格误拒只读重放 |
| `OldAuthorityUnknownBeforeBoundaryAfter` | U/P | 旧lineage/hold/dead-letter及next_attempt_at前/边界/后，未合法终结不建候选 |
| `CandidateAtomicAndCrossTargetCAS` | P | counter/claim/operation/credential/job/review指针原子，双连接同键/错摘要、组合FK/代际溢出 |
| `DurableSingleSendAndAckLoss` | U/P | 唯一owner一次Send，accepted仅记录受理；ACK/重试/resume零新秘密/第二SMTP |
| `ControlledDueLeaseDriverNoMailer` | U/P | 普通三worker零Start，受控driver原due/Claim/Renew/attempt/MaxRetries/permit、无Mailer且真join |
| `AcceptedAtomicRestoredAndSuspended` | P | credential/User G+1/可信role-status/false/operation/claim/job/review/Audit整组，trusted suspended仍停用 |
| `TerminalWholeReadbackAndComplete` | P | ACK独立回读operation＋review＋User代际/flag/role-status＋credential＋claim/job＋Audit完整原子组，不只operation.succeeded；合法terminal同意图原204、零新secret/ID/reset/SMTP/canonical，错key/decision/证据/非法来源含异常D0-restored拒绝。部分/unknown仍503/不identity_ready；不为新candidate/denied/重放盲NULL合法failed指针或抹历史/hold。最终Complete回引S4.10共同C/逐P完整集合：单个restored成功不掩盖他目标缺review/canonical/deniedAudit或原D0异常，合法terminal重放不新增第二结算authority |

- **本任务关闭条件**：可信审批逐字段绑定installation/run/actor/target/role/status/ref/digest，缺信任来源零secret/claim/SMTP；旧unresolved先合法收敛，unknown窗口前/边界/后、hold/dead-letter保持；candidate事务counter+claim+operation+exact pending credential+job+review pointer原子，只有新建owner发送一次；same-key/ACK/resume零第二SMTP；按原due/lease/MaxRetries无Mailer受控结算；accepted在一事务G+1/role-status/false flag/credential/claim/job/review/audit全终结才204；classifier检查全体而非只operation.succeeded；daily三个拒绝不放宽。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。 r55本次新增关闭条件：新候选/结算拒原D0或current-deleted而保持合法ND0正向，terminal同意图先只读whole-readback而不错误套pending资格；异常D0-restored拒绝，原合法失败pointer/history及unknown/hold保护不抹除。Complete仅回引S4.10共同C/逐P，不用已restored一项或operation单项替整体；原10源/7组/命令/最低P保持。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。root注册可信review Reader和受控settlement driver，ordinary worker保持关闭；实际SMTP/批准源/ACL、旧operation无法合法终结的真实实例另保留阻塞。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：model.go仅内部恢复metadata投影，不改公共wire；不调用会另开事务的顶层operator套嵌已持User锁事务。保留S10.5原七文件关闭和原Windows SID/Ed25519/epoch规则，不用requeue特权结算。原reset/resend/maintenance定向回归必须同版保留。 适用本节通用停止合同；未出现确认不当成认领/关闭。

<a id="task-article-management-s4-7"></a>

- [x] article-management-S4.7 交付同一锚定Article Journal的可信追加、纯读snapshot/Prefixes与受控协调，保持删除双完成语义。 开始@2026-10-29 完成@2026-10-29

- **类型/Story与范围**：开发（同卡局部验证）；article Story4 / S16共享基础。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：article设计 §3.4.6；Story4；article TP §4.8.2:761–778。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。 本次C1/C2另承接identity r53 SHA `DD61F4B89C94CE91F1E1317056ACAC1D7F8F464B2963DFDC59FF9C358590A6F0` / article r25 SHA `1D1CEE350E4FD93B3B8700FD2D48F00BA2E70561FDCD8D0630B2741CF7A45F31`；[Dirac R01](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r01/review-report.md)失败链及[R02](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/review-report.md)/[receipt](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/receipt.json)仅设计F001 CLOSED/new0，Main已接收。原技术/Planningpending为各自形成时点；本次新增任务/前置/端口映射另待非作者PlanningReview，不能借旧Gate或设计Review记实现/新TE批准，新增验收NOT RUN。
- **C2原生命期映射（2026-10-29）**：本次只消费identity r54 SHA256 `E823D25D9CDDFD4F2D7E3A869FBB6B15EAF5275E55147A21546D4C2836472522`（§4.2生命周期、§6.1、§9.1）与article r26 SHA256 `443C55E3E89EDBE4CF0870036BB662ED2E64B1A70F8BC5609F18025DBA71D267`（§3.4.6私有namespace lease、§6.1）的[C2完整差量/作者receipt](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c2-join-design-author-r01/receipt.md)；[原Dirac独立两轮设计Review](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c2-join-design-review-r01/review-report.md) SHA256 `1354859C65D125AC5E6C3EF7576539C0250F2B8E5D677F7049529A6E18D9E4C3`为该设计差量PASS/new0，[Main技术接收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c2-design-main-reception-r01.json)采用原资源生命周期真join、无新Runtime API。旧EF531/358BE有界[Planning PASS0及L-C2-OWNERSHIP-01](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-planning-review-r01/review-report.md)保留为历史，S4.18后续真实claim不触发旧技术重审；旧pending均限当时时点。本次由Main依用户standing采用建议授权两SSOT同步，新增C2映射仍待非作者独立PlanningReview；设计PASS不关闭代码C2 OPEN，人的精确接纳仍unknown/draft，无新TP批准、源码/运行窗口或任务关闭。以下新增验收全为NOT RUN。
- **ENTRY-MAP组件验收取值（2026-10-29）**：原RealInstallColdOpenAndPostproof把“真实OpenAnchoredDeletionJournalForCoordinate”具名正控写入验收，与原Noether [G0 port/lifetime notes §6](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-c1-g0-r01/port-and-lifetime-notes.md)采用同core/portable设施路径存在明确文字差异；[原Dirac applicability memo ENTRY-MAP](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/batch-applicability-r02/applicability-memo.md)为验收映射澄清，不是源码finding或采用决定。identity r54 §4.2组件C及article r26 §6.1允许外部facilities/批准/未交付完整executor fixture，仍要求真实Runtime/协调writer；[Main实际采用](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/entry-map-main-adoption-r01.json)依据用户standing推荐值授权明确采用下列本卡当前组件范围，本Planner据此定点同步，不新增设计/契约/TP批准或人的签字。本次ENTRY-MAP待原Dirac独立增量PlanningReview；实际实现/currentSHA/globalfmt/最终CodeReview及整卡接收仍pending，本Planner未读/hash源码、未运行新验收。[Main另行四源SOURCE/LOCAL及原PG13信号](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-c1-source-local-pg-start-r01.json)仅说明实际执行窗口，本metadata不发Source。
- **文件所有权**：仅以下新增/修改清单；实际owner为Noether / add-backend-dev（实际Agent ID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`），来源为Main实际Host spawn新Noether并取得ID、依据用户standing采用建议继续授权；本次认领登记时仅只读G0，尚未获Main本卡SOURCE/LOCAL/PG信号，不表示实施启动、G0或测试通过，不授源码或SQL准入。
  - `internal/article/deletion_journal_anchor.go`（新增）
  - `internal/article/deletion_journal_snapshot.go`（新增）
  - `internal/article/deletion_journal_anchor_test.go`（新增）
  - `internal/article/deletion_journal_snapshot_test.go`（新增）
  - `internal/article/deletion_anchor_finalizer_integration_test.go`（新增）
  - `internal/article/deletion_journal.go`（修改）
  - `internal/article/deletion_filelock_flock.go`（修改）
  - `internal/article/deletion_filelock_unsupported.go`（修改）
- **文件数**：8；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.8`、`article-management-S4.1`、`article-management-S4.6`、`identity-auth-S4.18`
- **批次关联**：[B13C](../plans/ir-01-content-community-batches.md#b13c)；原批次安排保持，Main已实际派发上述owner，开始@2026-10-29；本次认领登记时仅只读G0，未获SOURCE/LOCAL/PG。后续实施及局部运行须Main本卡精确信号，不由认领、前置卡关闭或角色/命令推定授权或人的设计签字。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/article -list '^TestAnchoredArticleJournal$'
go test ./internal/article -run '^TestAnchoredArticleJournal$' -count=1 -json
go test -race ./internal/article -run '^TestAnchoredArticleJournal$' -count=1 -json
go vet ./internal/article
go test -tags=integration ./internal/article -list '^TestAnchoredArticleFinalizerPostgres$'
go test -tags=integration ./internal/article -run '^TestAnchoredArticleFinalizerPostgres$' -count=1 -json
go test -race -tags=integration ./internal/article -run '^TestAnchoredArticleFinalizerPostgres$' -count=1 -json
go vet -tags=integration ./internal/article
go test ./internal/article -list '^TestAnchoredArticleSnapshot$'
go test ./internal/article -run '^TestAnchoredArticleSnapshot$' -count=1 -json
go test -race ./internal/article -run '^TestAnchoredArticleSnapshot$' -count=1 -json
go test ./internal/article -run 'TestPermanentDeletion|TestDeletionFinalizer' -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `CommittedInstanceAndBranchPermission` | U/C | 同实例/独立namespace，ordinary/startup可新增，recovery_replay仅exact committed读 |
| `PauseStagePendingPublishCommit` | U/C/P | stage前Pause→file/dir→pending+Load→Link/dir/alias/dir→committed+Load→最后media/lease检查及namespace Close真实结果→最后ctx检查→Complete接受→DELETE/Audit→204 |
| `EveryDurableBoundaryUnknown` | U/race | 上述每边界前后故障/取消/ACK loss；保留实际状态，零盲删除/再append |
| `PureReaderNoRepairAndPrefix` | U/C | normal各reader零sync/CAS/repair，committed新头配合法旧前缀拒绝 |
| `ControlledCoordinateAndABA` | U/C | closed/drain/exclusive/namespace锁全要；精确H0/stage/pending/H1；撤pending换revision |
| `OwnCauseAndLateCompletion` | U/race | 健康只清自身token，其他故障保留；unknown/迟到成功不开流 |
| `RealInstallColdOpenAndPostproof` | U/C/race | 原deletion_journal_anchor_test.go/TestAnchoredArticleJournal与deletion_journal_snapshot_test.go/TestAnchoredArticleSnapshot用真实NewRuntime→公开Runtime.Install/真实PhaseContext与checker→与具名native wrapper OpenAnchoredDeletionJournalForCoordinate共享全部共同校验/状态/协议的同一实际controlled-open implementation core（显式portable外部media/CAS/lease等设施）→真实CoordinateAnchor/parser，不用fake Runtime/private phase/影子算法或跳过安全check；最终代码须逐项标明wrapper/core共同调用位置及校验链并经独立核证，具名wrapper目标功能仍必须实现，不能production fallback；同实际backend-key/当前install grant/安装-id-epoch/原起点，Pending深拷贝；pending到H1、撤回H0新revision、committed exact、冷normal零写拒绝→控制协调→重建纯读。控制handle本地协调及完整纯读验链/本地Anchor核验后先供snapshot，不等待Runtime后证；同handle单次绑定后Runtime再独立核Anchor/原批准前缀/不变Store-recovery-commitment；伪nil/H2/ABA/多候选/unknown/失权/Stop真join负控，成功仍closed |
| `CompleteLinearizationAndCloseOrder` | U/C/race | 原anchor_test.go/TestAnchoredArticleJournal在Complete接受前后的取消、Close/check错误/阻塞两侧取证；全部fallible I/O/回读/media-lease检查/本次namespace Close真实结果及最后ctx检查先于Complete，失败先用仍有效原barrier Fault/撤权，再对同一namespace owner真join，原调用/WorkLease未退；Complete=0、Acquire拒绝且H1不回滚；Complete接受后仅归还不失败的内存mutex并固定原记录/nil，不再Close/ctx检查/对完成token补Fault。其他cause/旧WorkLease继续限制Finalizer。原117PASS/2C2FAIL及reg39原像保留，不删/弱化旧失败替代新边界证据；C2不增Runtime API |
| `NamespaceTerminalAndSingleClose` | U/C/race | 原deletion_journal_anchor.go私有deletionNamespaceLease嵌入原ExclusiveLease，原media.lock及deletion_filelock_flock.go/unsupported.go适配；原anchor/snapshot测试及原PG integration必要fixtures承接。attempted/terminal/closed分开；同一物理Close仅一次，任何WaitReleased(ctx)返回（含error）均已真实结束I/O及全部namespace所有权，保留原Close/cancel/deadline cause及errors.Is。获取失败若已owned必须lease+原error，nil+error仅无资源或已真join；不以ctx.Err/超时/清指针/Fd/Stat/Check/布尔或错误分类冒terminal，不盲重试raw Close/FD重用，不用no-op adapter；平台不能证明则retained pending并持owner/原工作/部署独占，不记native PASS |
| `NamespaceCloseUnknownRetainsDeploymentOwnership` | U/C/race | 原deletion_journal_anchor_test.go/TestAnchoredArticleJournal保留Main同名FAIL叶并改成终结前后双侧真实heldfile断言：真实NewRuntime/RunNormal/WorkLease→Append，Close unknown时同一synthetic文件仍held，H1保留/Complete0/新Acquire拒绝，先Fault后WaitReleased；Append/原lease/action及并发重复Stop均未完成、deployment Close0。同一owner独立driver真实一次关闭仍持文件并join后才发布terminal，Wait返回原error、原调用/lease/action结束，同一drainDone按其他cleanup条件归还deployment Close1；重复Stop不重置共享截止、不二次Close/新drain/authority，不手工Release工作计数或只翻terminal布尔 |
| `ReaderStartupPanicAndFaultRace` | U/C/race | 原anchor/snapshot tops覆盖owned+error、纯reader/包围startup-action（无barrier不伪造Fault权）、取消/超时仅记失败不提前退join；armed错误或panic先尝试原barrier Fault再真join，Stop错代际拒Fault保留binding/context cause且不复权，panic真实终结后按原值重抛、cleanup不覆盖根因。原work/operations/normalDone挡同一drain；driver不依赖Append返回、Stop/Lifecycle完成或重取journal mutex，Runtime mutex不跨I/O/join，不detached caller-return、不把append cause写入observationCleanupErr |

- **本任务关闭条件**：全持久化阶段stage/file/dir/pending/publish/alias/committed/回读前后及ACK loss故障矩阵；pending前停新permit/三worker claim；commit前零DELETE/204；同op必须精确committed匹配；normal所有reader零repair/sync/CAS；startup可合法追加、recovery_replay只读committed；协调仅close/drain/exclusive/namespace锁；H0撤pending换revision、合法旧前缀拒绝；健康仅解除自身token/unknown迟到不开流。 C1须消费前置identity-auth-S4.18冻结输出，按本次Main明确采用的ENTRY-MAP口径，以实际公开Runtime.Install/pc/checker→同一实际controlled-open core（与具名native wrapper共享全部共同校验/协议）→真实Coordinate/parser→同handle本地完整纯读snapshot先供→Runtime真实独立后证形成组件闭环；fixture仅外部media/CAS/lease/批准/Store等设施及原设计允许的未交付完整executor，Runtime/受控open共同实现/协调writer/parser不能替身。Windows具名native public wrapper实际unsupported负控和Linux compile-only须分别提供实证、按各自范围记结果；具名native wrapper正控NOT RUN/未验证，不能由core正控或compile-only记PASS。当前本测试项目组件Task可在该明确U/C范围及本卡其余全部局部条件满足后关闭，原生正控保留为明确未验范围，不能冒充已验证原生功能/完整root或套用ADR0024、S3.3、S3.5豁免；未来支持原生窗口须另明环境/owner/接收映射，不擅分配既有卡/新Task/日期或人的签字。原TestAnchoredArticleFinalizerPostgres最低P/Finalizer/SQL不减，原8文件内关闭不等待完整S4.13/root/E2E。C2成功须本次namespace资源真实terminal/join及最后ctx检查后才Complete，接受后只作无失败内存解锁并返回原记录/nil，后到取消不倒改unknown。C2错误/unknown/panic先Fault再真join，任何WaitReleased返回也已terminal；原调用/WorkLease/action在仍owned时不先返回/Release，原工作挡住同一drain及重复Stop，失败零healthy Complete。两侧heldfile组件证据须通过本卡原tops实际普通/race命中；原最低P/Finalizer/SQL、PG13 ordinary/race与oldreg39按既有范围回归，不能用list零命中/Skip/退出0或portable fixture替代native终结证明。本局部关闭不反向等完整S4.13/root；长期控制handle清算由后继S4.16独立验收，平台终结不可证的受影响adapter保持pending而不假关。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。真实POSIX/fsync/独立CAS和批准链迁移不由受控adapter证明；root必须把同实例给所有DELETE/finalizer/replay/history。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **ENTRY-MAP平台与后继边界**：最终证据须列出共享core/wrapper共同代码file:line与全部公共安全检查的实际覆盖、portable设施清单、Windows native publicwrapper真实unsupported负控、Linux compile-only，以及单独的“具名native wrapper正控：NOT RUN/未验证”；当前不得预写这些运行PASS。原生产wrapper仍按受支持平台合同实现、不加fallback。idle/postexecutor/postproof失败后的长期handle不自动成为普通work/Lifecycle，S4.16原root-owned Close/join-before-deployment/pool责任保留，当前组件不靠fakeprovider hold、提前关闭snapshot或计数技巧冒该证明；不反向加完整S4.13/root前置。平台终结unknown保持pending，portable不冒native终结证明。
- **串行/停止条件**：复用原payload/HMAC/命名/operation与finalizer算法。不得在错误defer盲删stage或裁尾；具体清理仍服从运行授权。原deletion_test等回归只读执行，不在本候选顺手改写它们；若确需修改，先扩精确写窗。 适用本节通用停止合同；未出现确认不当成认领/关闭。 C1的Runtime producer/checker只由identity-auth-S4.18两源交付，Noether不加第9源；当前PG1+C2two active按Main已知窗口，C1续接须Main重开原8内精确窗口及冻结前置契约/构建运行窗口。P与C2局部结果不关整卡，正常入口仍拒pending且零repair。 本次C2私有owner/平台适配/fixtures仍限原8清单，不占S4.18两源/不新增Runtime债、hold、Resolve或action API；当前Noether仅只读C2 G0，后续SOURCE/LOCAL/PG分别待Main精确授权，前文旧窗口仅当时事实。若必须第9源/新公开契约或终结driver无法脱离等待环，停受影响部分报Main，不机械增Task或伪终结。

- **C2实际失败/回归来源（限时点）**：[Main directed R02](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/main-a47-directed-r02/receipt.md)为Main实际一叶编译成功后行为FAIL、非超时/编译失败；Main另跑原PG top ordinary/race各13叶PASS，仅证该P范围。作者core126PASS/1FAIL、C2组9PASS/1FAIL、oldreg39及原117/2C2FAIL历史保留，不由设计/映射将其记C2实现关闭；本Planner未执行、未读/hash当前Article或Runtime源码。

- **Main G0/C1受理（2026-10-29）**：[Noether G0](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-g0-r01/receipt.md)及[Main核收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/g0-main-reception-r01.json)发现真实Runtime没有可公开执行的coordinate/install producer，现有coordinate正控是recovery包内私有phaseLocked测试；不能用fake grant/观察值/其他purpose替代完整closed/drain/exclusive/namespace核证。原八源SOURCE/LOCAL/PG保持closed，仅暂停相关接缝；Main已实际恢复原Banach/add-feature-designer `01a18389-57ab-7b70-8c86-82a61e7729f1` 做有界只读建议，不重开旧Review finding或声称设计已修改。原规划技术字节与受审前像相同、旧pending为历史，人的接纳仍unknown。原receipt用户指定实际来自MainHost派发；自有workspace Temp证据保留，非源码改动，后续使用OS Temp。

- **Main独立Core实施开始（2026-10-29）**：Main已发[原八源Core SOURCE/LOCAL](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-core-source-local-start-r01.json)，原Noether仅实施r23已稳定的可信追加/纯读及U/C/race。C1协调producer/正控仍停，CoordinateAnchor暂保持failclosed，不以fake authority/私有phase构造冒充许可；PG保持closed，不能从隔离cluster就绪或identity peer的PG信号推定本卡权限。原Banach只读C1建议并行，未设计写/实施Runtime补丁；本卡全六组和最低P/C未齐不能关闭，局部core通过不冒整卡/批次接收。

- **Main本任务技术接收与关闭（2026-10-29）**：原Noether完整8源/C2/C1及[最小两源F001修复](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-c1-f001-author-r01/receipt.md)已STOP/冻结；原Ptolemy[R01真实HIGH](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-review-r01/review-report.md)在[R02完整独立CodeReview](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-review-r02/review-report.md)按当前261A/A49及其余6精确对象CLOSED/new0、组件CODEPASS，非作者自审。Main[全文/八源/raw核收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/a47-review-main-reception-r02.json)确认普通/race每模式295（core203+独立PV01-03共23、原F001pair及剩余PV04共14、old39、原PG13+PV05three），真实run/pass同名、零skip/fail/unfinished/race；全部11风险最低层与原145/200/C2组保持。实际公开Runtime.Install/pc/checker→与具名wrapper共用全部共同校验的真实core→真实Coordinate/parser→同handle本地纯snapshot先供→Runtime独立后证已在明确ENTRYMAP组件范围核实；不是假Runtime、影子算法或production fallback。Main[当前同源联合实测](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/main-final-joint-r01/receipt.json)普通/race各757叶全部必需名保留（joint674+S49编码41/真实PG26+Article真实PG16），实际Store→当前D404 Verify、当前C1/C2/真实Finalizer SQL/HTTP及真实Commit后syntheticACKerror/503/独立committedSQL/同op不重复/公有404均执行；84新schema逐名物理核存在并保留。Manifest六PASS、18Go fmt空，rootfmt仍FAIL2且旧23/两raw字节同；Main只采用[本次精确19源有限格式处置](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/root-format-disposition-final-r01.json)，不记全七PASS。原Dirac[最终whole-batch独立结论](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/batch-review-final-r01/final-review-report.md)及[Main现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/batch-final-main-reception-r01.json)另核真实跨卡来源/接口/依赖/覆盖及functional性能适用性，不从单卡PASS相加。Main当前status/diff/HEAD/staged/保护基线无越界，故按本卡明确组件自身关闭条件及用户standing推荐值授权接收；不是人的业务签字、容量测量或完整需求验收。Windows具名wrapper实际unsupported负控、Linux仅compile-only分记，具名native wrapper正控NOT RUN；真实provider/POSIX/完整S4.13/S4.16 root长期idle及postexecutor/postproof失败handle先Close/join再deployment/pool/生产worker/E2E仍未验，原后继责任/旧窄豁免边界保持。当前测试命令/fixture真join；Main隔离PG最终stop/join见本批最终交接，不声称旧linker55164 OSjoin已证实。上文pending/active窗口/NOT RUN原文均保留各自形成时点，当前技术接收以本条精确证据为准。

<a id="task-article-management-s4-8"></a>

- [x] article-management-S4.8 以0011实现publication/evidence/current原子发布和精确历史closure，保持fresh Facts语义。 开始@2026-10-29 完成@2026-10-30

- **Main本卡最终接收（2026-10-30）**：原作者Noether原10源完成并修复rollback错误返回非零publication结果；非作者Dirac R02独立关闭F001 MEDIUM、CodeGate PASS/new0，U/C54、P58各普通race、原3探针均通过；typed同Store/Source一次装配、publication三者原子、精确history/fresh、Import独立checkpoint与原DDL最低P完成。 [本卡独立评审核收](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/a48-code-review-main-reception-r02.json)；[当前三卡Main现场检查](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/main-current18-verification-r02.json)与[精确302个新增schema实际SQL](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/main-current18-physical-final-r02.json)均已核收。全仓六项PASS，根gofmt仍为原23片段完全同hash的FAIL2，当前17份Go格式空输出PASS，采用仅本18源的有限技术处置，不扩豁免或清理。非作者Dirac另完成[本批跨卡整体G4评审及Main现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/b13d-batch-review-main-reception-r01.json)，接口/来源/迁移/资源/必要集成、性能维度与覆盖按报告实际范围核收，不是三卡PASS相加。Main复核固定main/HEAD、staged0、diff/所有权/当前前置与本卡关闭条件，按task-execution-workflow §10和用户standing建议值继续授权作技术接收并关闭本开发卡，完成@2026-10-30；无Git提交约定/本轮Git写。旧G0-only、NOT RUN、待Review/OPEN等均为原时点，当前以本条和精确证据为准。设计draft/人的业务接纳仍unknown；真实provider/ACL/审批主体、SMTP/native/POSIX、完整root/S4.13绑定/真实恢复/Case18/部署/E2E由原后继任务分别保留，不据此冒PASS、不重开或关闭其他卡，原phone/缺失行限制及ADR-0024/0025窄边界不扩。

- **类型/Story与范围**：开发（同卡局部验证）；article Story4 / S16共享基础。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：article设计 §3.4.7；Story4/S16；article TP §4.8.2:784–788（含R02修复后的786合法oracle）。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **r27/TP r21范围补充本次正式技术映射（2026-10-29）**：article design r27 SHA256 `4D2321D4A2D9514EA1AC2A4812E11AA6A83CA8046202F596B2F99F1DE39C7CEE`，§3.4.7/§6.1；[原Dirac两轮Design Review](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/a48-design-r27-review-r01/review-report.md) SHA `5CA39619D80CA48BCFC0583D4D5CC0458B1E635CE0C13806E8441AD0BA12D812` PASS/new0及[Main采用](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/a48-design-r27-main-adoption-r01.json)已接收。实际消费原作者Jason冻结[Article TP r21的r27新增块](../design-workspace/article-management/test-plan.md#b13d-a48-design-r27) SHA `24786C1A584ACF5277DD2DBD43E053CB7679F2867CD42937EDC7F722FD435036`，仅7行集中场景/原五组；[作者receipt](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/a48-tp-r21-r27-author-r01/receipt.json)与[Main输入冻结](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/a48-tp-r21-r27-main-freeze-r01.json)不代独立Review。依据[Main两SSOT窗口](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/planning-threecard-write-start-r01.json)正式登记本映射；本次两TP＋Tasks/Plan联合独立mapping Review/Main接收仍PENDING，无额外TP前审Gate。旧来源/旧失败/Case与计数/Trace/C2/窄豁免保留原时点与范围；仍draft/人的接纳unknown，新增U/C/P/R全NOT RUN，不声称原十源已落仓/GoPG通过、真实checkpoint批准或授SOURCE。
- **文件所有权**：仅以下新增/修改清单；实际owner为Noether / add-backend-dev（实际runtime Agent ID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`），来源为Main实际恢复原Noether/add-backend-dev并派发本卡只读G0，依据用户推荐值继续授权由Main真实派发，不是用户亲指定Role或人的设计签字。本次认领登记时Main尚未接收本卡G0结果或源码/LOCAL信号，仅只读G0准备，SOURCE/LOCAL/Go/PG均closed；不表示实施启动、G0/测试通过，不授源码、运行或SQL准入。
  - `migrations/0011_article_replay_publication.sql`（新增）
  - `internal/article/replay_publication.go`（新增）
  - `internal/article/replay_publication_test.go`（新增）
  - `internal/article/replay_publication_integration_test.go`（新增）
  - `internal/article/replay_evidence.go`（修改）
  - `internal/article/deletion_replay.go`（修改）
  - `internal/article/deletion_replay_source.go`（修改）
  - `internal/article/deletion_replay_test.go`（修改）
  - `internal/article/deletion_replay_source_test.go`（修改）
  - `internal/article/deletion_replay_integration_test.go`（修改）
- **文件数**：10；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `article-management-S4.7`、`identity-auth-S4.7`、`article-management-S4.2`
- **批次关联**：[B13D](../plans/ir-01-content-community-batches.md#b13d)；原批次安排保持，Main已实际派发上述owner，开始@2026-10-29。本次认领登记时仅只读G0准备、Main尚未接收本卡G0结果，SOURCE/LOCAL/Go/PG均closed；后续权限须Main另发精确信号，不由认领/已闭前置或旧规划Gate推定。设计draft/人的精确接纳unknown及原技术Review覆盖保持，本次仅认领事实登记。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/article -list '^TestReplayPublicationClosure$'
go test ./internal/article -run '^TestReplayPublicationClosure$' -count=1 -json
go test -race ./internal/article -run '^TestReplayPublicationClosure$' -count=1 -json
go vet ./internal/article
go test -tags=integration ./internal/article -list '^TestReplayPublicationPostgres$'
go test -tags=integration ./internal/article -run '^TestReplayPublicationPostgres$' -count=1 -json
go test -race -tags=integration ./internal/article -run '^TestReplayPublicationPostgres$' -count=1 -json
go vet -tags=integration ./internal/article
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `PublicationCurrentEvidenceAtomic` | P | 三者全有全无、同代winner/后继generation复用；原TestReplayPublicationPostgres须真实SQL COMMIT成功后注入ACKerror、独立回读exact三者，不能将reuse bool=false当明确未提交而先Fail/Replay/盲写。same-pointer/同pool装配与所有原Store/新publication首次入口在同一短mutex给出唯一顺序，含纯读/参数-ctx-I/O失败；second bind同值也拒，配置与first-call竞争仅见完整配置或后绑拒绝，失败/取消/unknown不清seal，mutex不跨外部I/O/Close/Stop。U/C/race补装配时序，不代本组最低P；未装配合法legacy fresh读零回写，新publication写无旧二表fallback；合法复用IDs/Clock各0、原ref/time/计数不变 |
| `CompositeFKImmutableCanonical` | C/U/P | 原完整组合FK/immutable/合法generation-token/source_kind、UTC微秒/整数边界及独立17字段golden/source_digest oracle不减。原replay_publication_test.go/TestReplayPublicationClosure随本组U/C核NewReplayPublicationPorts(store,ReplayPublicationDependencies)返回原Store指针/同pool，仅用固定Source.journal snapshot，不另收可错配Journal/head；nil store/pool/Source/Runtime、空InstallationID、PublicationIDs/Clock缺失或typednil、非锚定snapshot依赖拒绝。构造只核类型/引用，Runtime/Journal/DB/verifier/IDs/Clock/ENV读取各0，成功不授grant/旧来源证明；原base接口/旧构造形状不扩，真实PG组合FK/immutable最低P保持 |
| `ExactHistoricalClosureAllowsRequiredReads` | C/P | 普通ctx无Install许可、LegacySources缺失或调用即失败的合法历史正控仍走六项exact lookup/短只读事务、完整Ready20/first16/来源/原time及同journal纯snapshot/锚定前缀；DB lookup、Journal snapshot、Anchor/source及前后revision必需读取分别正向记账>0，外层Store/双namespace另记。Refresh/ReplayAll/Consume/Fail/Publish/Anchor CAS-write/repair分别0，零业务表/receipt写，不按current/latest换来源、不以identity直查Article表替公开port。未装配合法legacy ReadCurrent/ReadStatus独立fresh正控仍零回写；normal/Publish/Ensure/history verifier调用0，不误加install前提或运行中换Source/journal；缺配置新写closed，无旧二表fallback |
| `HeadAdvanceAndReadDrift` | P | 同一固定Source.journal完整可信链：历史H0为当前committed H1合法前缀、日常删除推进current/Facts后，读间revision稳定则历史仍合法，VerifiedArticleRevision/CurrentHead/HistoricalHead/Prefixes精确对应完整快照，不能把H0冒当前head。读间head/revision漂移、撤pending回同H0新token、换id/epoch/ABA/裁链均closed，fresh仍拒stale；保存两次revision/Anchor、完整前缀及current/Facts与历史前后像，零Refresh/repair，不换源或截历史掩盖错误；原最低P不以构造类型检查替代 |
| `LegacyReassertionVersusTrustedImport` | P | 真实PG核现在重申≠过去发布；LegacyPublicationVerifier.VerifyLegacyPublication(context.Context,LegacyPublicationProof)error仅Import消费，缺/typednil仅禁Import而其余合法Publish/Ensure/history正控仍可用且verifier0，无默认批准/无关secret读取。实际公开Runtime.Install/pc/permit、固定安装/实例/代际/namespace与同控制handle资源先核权，事务外同步逐字段核独立既存已批准checkpoint：SourceRef/Digest、run/backup、EvidenceRef/Digest/CompletedAt、JournalID/Epoch/Head/OriginalPublishedAt及固定scope；仅Manifest/当前三表自签/callerbool不能代证，proof不增字段。verifier返回含error前started I/O真join/保errors.Is，handle贯穿核源、返回后当前grant/ctx/独占/namespace复核和实际操作；nil不授许可，撤权/取消/失锁/未知零新INSERT与RecoveryRun写。仍有效才原INSERT-if-absent/冲突exact回读；合法receipt复用原ref/time/source_kind/字段，原出版时刻留source、published_at仅导入时刻，输出精确PublicationBinding。P正控用真实Store/Runtime及独立synthetic checkpoint，不冒真实批准；同Store NULL绑定仍归后继S4.13的LegacyBindingAndForwardOnly P，不倒挂本卡关闭 |

- **本任务关闭条件**：三者提交全有全无，复用胜出receipt/ref/time；publication immutable/组合FK/版本/canonical/可信前缀与revision；historical不读current/不Refresh/Replay/repair，日常Facts推进后历史仍合法，fresh仍拒stale；legacy当前重申不冒过去已发布，真实PG核旧来源导入仅独立批准并保留source_kind、来源/immutable publication约束与原字段，缺来源关闭；article对RecoveryRun零写，仅输出精确PublicationBinding。上述局部正/负证据仅依赖已完成前置及本卡文件，不以S4.13的Store绑定为关闭条件；ACKloss回读三者不重放。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。 r27本次新增关闭条件：原10源/5组内核同Store typed一次装配/first-use封存、构造零I/O与Import专用LegacyPublicationVerifier精确签名/独立checkpoint逐项核证，真join后再核当前许可；正向仅明确受控fixture，真实批准/来源不明仍停止对应Import，无production nil-success。原P/17canonical/DDL/三表ACK/历史-fresh分流和S4.13后继责任不变，不以完整root/S4.13为本卡反向前置；超原10或新内部依赖必须停具体部分回Main。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。Store精确绑定的真实同Store P验收归既有identity-auth-S4.13的LegacyBindingAndForwardOnly及其安装owner：一次补原NULL、非NULL精确幂等/错绑拒绝、revision递增且counter和旧业务结果不变；normal重启由后继root/联合软件验证。该全局义务保留，不倒挂本卡关闭、不转移Store写权；真实旧原子发布检查点unknown不导入。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：0011唯一owner，不改0003/0009或S16 authority/fixture。须协调原S4.4 Issue修复源码写窗；原首份计数/Trace/fresh回归完整保留，不在test/contract旧冻结文件加实现。 适用本节通用停止合同；未出现确认不当成认领/关闭。

<a id="task-identity-auth-s4-13"></a>

- [x] identity-auth-S4.13 可信genesis/anchored_migration承诺prepared→bound和DB安装绑定可中断续接，normal只验证来源。 开始@2026-10-30 完成@2026-10-30

- **Main本卡最终技术接收（2026-10-30）**：原Hooke原5源实施与作者自测已核收；非作者Ptolemy R03独立CodeGate PASS、anchored_migration读取HIGH及错writer提前拒绝MEDIUM独立CLOSED、new0，97风险leaf/模式和实际25PG、source/cancel/真实Article controlled core最小接缝按[正式接收](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/s413-code-review-main-reception-r03.json)所列边界核收。source missing错误归类额外修复已覆盖，原两类误用fixture/Windows关闭文件判据诊断保留，不作产品Red或native PASS。原Dirac另完成两轮跨卡来源/状态/迁移/authority/资源与必要性能覆盖核对，[整体G4 R02及Main现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/batch-g4-review-main-reception-r02.json)为有界PASS/new0，非单卡PASS相加；组合发现的HIGH RECOVERY-B13E-G4-R01-F001已由原Zeno在另授shared identity三源内仅改nil-safe time.Equal，真实时刻/微秒/nil/角色状态/集合校验保留，原Reviewer独立CLOSED。原两叶字节/UTC不变＋10独立控制普通/race各12通过，24新增schema整像另经Main实际SQL核收。[Main当前20源/6文档的8项检查](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/main-g4-f001-current26-receipt-r01.json)实际unit11/PG3普通race、owned20fmt空、identity integration vet/build通过，另6schema SQL与逐raw/生命周期核对完成；[此前Main23](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/main-current23-receipt-r01.json)全仓六项结果保留原修订范围，未称全历史新修订重跑。根fmt仍实际FAIL2，原23输入和stdout/stderr完全同hash，采用[仅当前20源精确有限处置](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/root-format-current20-disposition-r01.json)，不记全7PASS或继承blanketwaiver。所有本批已执行进程/回调/Rows/Tx/pool真join，[Main专用PG服务与helper实际exit0](../../../../.spec/wip/recovery-runtime/evidence-b13e-r01/main-own-pg-stopped-r01.json)、全部raw/cache/schema/data保留。Main核实际diff/status/固定main与HEAD/staged0、文件边界/本卡前置及关闭条件后，依据task-execution-workflow §10和用户standing采用建议继续授权作技术接收并关闭本开发卡，完成@2026-10-30；无Git提交约定或Git写/清理。设计draft/人的精确业务接纳仍unknown；真实provider/ACL/审批主体/SMTP/native/完整root/真实恢复/Case18/部署/E2E/容量由原后继分别保留，不冒通过；ADR0024与S3.3/S3.5窄边界不扩、非PASS，S3.6不变。旧NOT RUN/待Review/窗口/OPEN字样为当时时点，当前以本条及精确证据为准，不关闭其他任务。

- **B13E实际认领（2026-10-30）**：owner Hooke / add-backend-dev，实际runtime Agent ID `01a189aa-4c16-7eb0-8dd6-fffaf4762cb1`；由Main依据用户standing采用建议继续授权实际Host spawn并派发，非用户亲指定Role或人的设计签字。开始@2026-10-30。Main已另授本卡原5 SOURCE/sole LOCAL，须G0后实施；新PG须Main另行信号。本次认领登记不宣称已有G0结果、实施完成或测试PASS，不额外发放源码/运行权限。旧“尚未派发/未认领”“状态仍[ ]”及mapping待审为原规划时点历史；本次仅登记实际认领，既有六稿joint Planning R02及C1完整五源executor映射PASS/Main接收的适用范围保持，不因认领重开Planning Gate。设计draft/人的精确接纳unknown不变，不代表代码、测试或新PG通过。

- **类型/Story与范围**：开发（同卡局部验证）；identity S04，article必要共享安装依赖。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §3.3.1/§4.2 genesis/迁移；article §3.4.6/7；identity TP §4.7.1:791–793,847；article TP §4.8.2:787–788。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。 本次C1/C2另承接identity r53 SHA `DD61F4B89C94CE91F1E1317056ACAC1D7F8F464B2963DFDC59FF9C358590A6F0` / article r25 SHA `1D1CEE350E4FD93B3B8700FD2D48F00BA2E70561FDCD8D0630B2741CF7A45F31`；[Dirac R01](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r01/review-report.md)失败链及[R02](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/review-report.md)/[receipt](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/receipt.json)仅设计F001 CLOSED/new0，Main已接收。原技术/Planningpending为各自形成时点；本次新增任务/前置/端口映射另待非作者PlanningReview，不能借旧Gate或设计Review记实现/新TE批准，新增验收NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际执行者尚未派发/未认领，当前规划登记不是代码写权。
  - `internal/recovery/installation.go`（新增）
  - `internal/recovery/installation_test.go`（新增）
  - `internal/identity/recovery_installation.go`（新增）
  - `internal/identity/recovery_installation_test.go`（新增）
  - `internal/identity/recovery_installation_postgres_test.go`（新增）
- **文件数**：5；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.8`、`identity-auth-S4.9`、`article-management-S4.7`、`article-management-S4.8`
- **批次关联**：[B13E](../plans/ir-01-content-community-batches.md#b13e)；由本pass Batch-Plan消费原任务安排，实际执行者未派发/未认领，状态仍[ ]。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./internal/recovery -list '^TestRecoveryInstallationBinding$'
go test ./internal/recovery -run '^TestRecoveryInstallationBinding$' -count=1 -json
go test -race ./internal/recovery -run '^TestRecoveryInstallationBinding$' -count=1 -json
go vet ./internal/recovery
go test -tags=integration ./internal/identity -list '^TestRecoveryInstallationPostgres$'
go test -tags=integration ./internal/identity -run '^TestRecoveryInstallationPostgres$' -count=1 -json
go test -race -tags=integration ./internal/identity -run '^TestRecoveryInstallationPostgres$' -count=1 -json
go vet -tags=integration ./internal/identity
go test ./internal/identity -list '^TestRecoveryInstallationStoreBinding$'
go test ./internal/identity -run '^TestRecoveryInstallationStoreBinding$' -count=1 -json
go test -race ./internal/identity -run '^TestRecoveryInstallationStoreBinding$' -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `IndependentApprovalAndPreparedBound` | U/C/P | 独立批准+初始两Anchor+DB安装绑定，prepared不开放、bound才能准入 |
| `InterruptedSameManifestOnly` | U/P | 中断仅同manifest续接，不覆盖安装/id/epoch/Anchor |
| `DeleteThenRestartAndBeginDisablesGenesis` | P | 合法删除推进后重建核连续head，出现Begin转完整历史 |
| `LegacyBindingAndForwardOnly` | P | 本卡安装owner在article-management-S4.8交付精确PublicationBinding后，以真实PG同一RecoveryRunStore receiver执行BindLegacyPublication：独立可信原publication来源一次补原NULL，非NULL只精确幂等、错绑拒绝；成功绑定revision递增，counter及原outcome/evidence/时间/审计不变；0010/0011前向/禁旧resolver混跑，不回滚schema |
| `InstallationExecutorAndBoundReentry` | U/C/P | 原recovery/installation.go实现设计InstallationExecutor.Install(ctx,permit,manifest)完整fresh/prepared算法/Article适配，原installation_test.go/TestRecoveryInstallationBinding核同PhaseContext/当前grant/原v1扁平编码/独立批准/prepared→bound及中断；bound同manifest只调用受控协调、不回写prepared/初始Anchor或run。原identity/recovery_installation.go仍同Store BindInstallation/BindLegacyPublication；原TestRecoveryInstallationStoreBinding及TestRecoveryInstallationPostgres核真实SQL/回读。与前置Article原8/S4.18消费组合，不再暗改Runtime接口 |

- **本任务关闭条件**：不存在CAS键前置、独立批准源、两namespace初始Anchor/全链和DB绑定、prepared中断精确续同manifest、bound才准入；零run/Begin只是必要条件非自证；新Begin永久转历史路径；既有id/epoch/Anchor不覆盖归零；可信旧publication的绑定由本卡在已有article-management-S4.8前置及自身文件内完成真实同Store P正/负验收：仅一次补原NULL，非NULL精确幂等、错绑拒绝不覆盖；成功绑定revision递增，counter及原outcome/evidence/时间/审计不变，不以任何后继卡为本卡关闭前置；normal不安装/写元数据。 新增InstallationExecutor port映射在原5文件/原tops闭环，完整fresh/prepared/批准/两Anchor/DB绑定/原NULL一次补绑的P义务保留；bound协调不冒完整安装通过。Runtime中性producer已由独立前置交付，不在本卡扩写runtime.go，不等待后继root/E2E。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。可信来源实际provider/人/介质/ACL和legacy完整性仍需外部事实。受控fixture仅证明软件来源协议。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：安装写者仍是同RecoveryRunStore receiver，新增文件承接BindInstallation/BindLegacyPublication，不建第二pool/Store。两个包top分别约定TestRecoveryInstallationBinding（recovery）及TestRecoveryInstallationStoreBinding（identity），按命中分账。 适用本节通用停止合同；未出现确认不当成认领/关闭。

<a id="task-article-management-s8-3"></a>

- [x] article-management-S8.3 platform消费同Runtime工作许可，阻断Session/业务下游并在响应真实结束后归还lease，提供最小probe处理。 开始@2026-10-29 完成@2026-10-29

- **类型/Story与范围**：开发（同卡局部验证）；article Story8，identity S04必要接入依赖。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §4.2访问面；article §8.6/Story8；identity TP §4.7.1:829；article TP §4.8.2:796–800。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际owner为Pauli / add-backend-dev（实际Agent ID `01a1854a-0e79-7f02-abfd-632b91c4fb04`），来源为Main实际spawn Pauli、依据用户standing采用建议继续授权；本次认领登记时Main尚未接收本卡G0结果或源码/LOCAL信号，本次只登记真实认领，不授源码/LOCAL或运行写权。
  - `internal/platform/recovery_guard.go`（新增）
  - `internal/platform/recovery_guard_test.go`（新增）
- **文件数**：2；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S3.2`、`article-management-S1.2`
- **批次关联**：[B13B](../plans/ir-01-content-community-batches.md#b13b)；原批次安排保持，Main已实际派发上述owner，开始@2026-10-29；本次认领登记时Main尚未接收本卡G0结果或源码/LOCAL信号，当时仅只读G0；后续实施/局部运行须另获Main本卡精确SOURCE/LOCAL信号，不由认领或前置S4.7关闭推定开窗/测试PASS，S4.7格式例外不扩本卡。
- **验收命令**（根目录；原登记命令，实际执行与留存见本卡Main最终接收）：

```text
go test ./internal/platform -list '^TestRecoveryHTTPBarrier$'
go test ./internal/platform -run '^TestRecoveryHTTPBarrier$' -count=1 -json
go test -race ./internal/platform -run '^TestRecoveryHTTPBarrier$' -count=1 -json
go vet ./internal/platform
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `ClosedUnknownBeforeSession` | U/C | 恢复/unknown既有503/envelope/Trace且Session/业务后继0 |
| `ResponseLeaseAndCancellation` | U/race | 完整响应/事务范围持lease，结束Release，取消不沿旧代际 |
| `ProbeDoesNotAuthorize` | U/C | live200仅alive，ready按PublicReady200/503，零run/secret |
| `ExactRouterBoundaryContract` | C | guard在exact404/405/Allow之后，在Session之前；组件测试不冒真实rootCase18 |

- **本任务关闭条件**：closed/unknown→既有503 envelope与Trace、零后继；覆盖完整响应/事务直至Release；readyz关闭503/open200，livez不授业务权；不自行创造authority或surface；ctx取消/故障不沿旧permit继续。exact404/405/Allow应在调用guard前决定，不能将unknown路径变恢复动作。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。真正factory→Trace→exactrouter→guard→Session及Case18两GET属于唯一root和联合软件验证；本卡handler单测不冒Case18。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：不需要重写http_security.go或两个GET业务Handler。Guard只消费中性接口；probe注册及root私有surface下候选承接。 适用本节通用停止合同；未出现确认不当成认领/关闭。
- **Main当前续接（2026-10-29）**：Main全文核[Pauli G0](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-g0-r01/receipt.md)及最终读集核对，现场确认两新增路径原不存在、稳定Runtime接口SHA与B13A接收一致，采用原Writer透传/同步持lease至handler真返回/现有Verify观测派生ready的建议，已发[仅两源SOURCE/LOCAL](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-source-local-start-r01.json)。当前原作者实现与定向验证中，未Review或关闭；Trace→exact router→guard→Session顺序保持。原receipt中“用户指定/用户告知”实际来自Main转交，真实Role/ID来源为Main Host spawn，非用户亲指定或人的签字。Journal原11仍closed，另有精确admission两文件peer兼容补丁，不授权本owner改动它们。全仓七项待双方源稳定后由Main联合运行，旧S4.7格式例外不扩本卡。

- **Main组件评审接收（2026-10-29）**：原Pauli两源STOP/frozen，Main全文核源、交付及作者普通/race各56叶，兼容稳定版另跑各56叶通过；[非作者Curie独立报告](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-review-r01/report.md)有界PASS/新增0，独立普通/race各56叶＋6探针及platform各259叶。Main已全文读报告/receipt/probes、实际解析56/6两组raw并核精确SHA，见[评审核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-review-main-reception-r01.json)。Task仍[-]；原Journal11现已授权另一owner实施，待稳定版联合验证、全仓七项、本卡根fmt精确处置及最终Task/batch接收，不把组件Review冒root/Case18/PG/E2E。

- **Main规划覆盖范围核对（2026-10-29）**：现有[r52四卡四批规划评审](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/plan-r52-review-r01/review-report.md)覆盖S4.8，未覆盖本卡原映射，不能用本卡代码Review或旧六稿代替。Main已恢复原Kuhn / add-reviewer（实际ID `01a1835a-8010-7a73-a8dd-3192719b6965`）有界核本卡及B13B适用映射，源码仍两文件冻结、独立HTTP代码Review及已完成局部验证保持原事实，暂不关闭本卡。上文执行/七项NOT RUN等为登记时点；[Main当前同版实测](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/main-joint-evidence-r01/index.json)有四top各408普通/race及全仓六项PASS，根fmt原23仍FAIL2并另有精确当版有限处置；Journal R01失锁HIGH正在原四源返工，不由此伪造本卡或批次验收。

- **Main规划范围核实（2026-10-29）**：[Kuhn当前有界规划报告](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-planning-review-r01/review-report.md)PASS/新增0，Main全文读报告/receipt并现场核六文档摘要，另实读原Planning R01:46及R02:3并核原报告SHA，见[规划现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-planning-main-r01.json)。原R01其实已明确覆盖S8.3，R02沿用未受影响范围，当前原技术正文与该前像一致；本卡旧待审注记是历史状态，上一条范围核对是Main当时待核，不表示从未受审。r52四卡报告本身仍不覆盖S8.3，本次当前对象另已完成两轮定点Review，不伪造旧报告新范围或代码测试结果。规划无gap需返工，不新增任务/依赖；S4.8代码HIGH仍OPEN，两卡及批次最终接收尚待Main按各自证据办理。

- **Main最终接收（2026-10-29）**：[Curie HTTP独立报告](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-review-r01/report.md)固定两源有界PASS/新增0，原四组普通/race各56叶及独立6探针已由Main核收；当前联合版再跑56叶普通/race通过。原[Kuhn规划核实](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/s83-planning-main-r01.json)已确认适用映射，无规划gap；同Runtime许可、响应真实结束归还lease、最小probe及exact router-before-Session组件合同闭环。[Curie批次最终报告](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/batch-review-r03/review-report.md)与[Main批次现场核收](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/batch-review-main-r03.json)另完成跨任务机制/最小合成验证及覆盖、必要性能边界检查；不由两卡PASS相加推定整批。[本版Main联合实测](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/main-joint-evidence-r04/index.json)四top count1普通/race各446叶全PASS（77/42/271/56），原叶逐名保留、全仓六项PASS、15源fmt空输出PASS。根fmt仍FAIL2，旧23哈希/两raw同写前；[当版精确有限格式处置](../../../../.spec/wip/recovery-runtime/evidence-b13b-r01/format-disposition-main-r04.json)不改FAIL、不豁免代码/真实环境。Main核固定main/HEAD、staged0、完整差量/所有权/适用原requires及受保护基线无越界后，依用户standing建议值继续授权按本开发卡自身条件关闭，完成@2026-10-29。无Git提交约定或本轮Git写操作；不是人的业务签字、PG/SQL/nativePOSIX/真实provider/root/Case18/部署/E2E通过，后继原任务保持。以上旧NOT RUN/待Review/OPEN/窗口文字均为当时历史，当前以本条和证据精确范围为准。

<a id="task-identity-auth-s4-14"></a>

- [x] identity-auth-S4.14 交付root侧受控RestoreDriver与部署独占适配边界，保证稳定制品、私有stdin、精确receipt和真实join/unknown暂停。 开始@2026-10-29 完成@2026-10-30

- **Main本卡最终接收（2026-10-30）**：原作者Chandrasekhar原5源完成，非作者Ptolemy R04独立关闭原3 HIGH与复合异常MEDIUM、CodeGate PASS/new0；独立core111含7个外部原探针、旁路正控1、PG7各普通race通过。当前实际源码本地104/PG7已由Main普通race复核，稳定handle/当前authority、原异常与次级清理原因、真正child/pipe join和unknown无重放闭环。 [本卡独立评审核收](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s414-code-review-main-reception-r04.json)；[当前三卡Main现场检查](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/main-current18-verification-r02.json)与[精确302个新增schema实际SQL](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/main-current18-physical-final-r02.json)均已核收。全仓六项PASS，根gofmt仍为原23片段完全同hash的FAIL2，当前17份Go格式空输出PASS，采用仅本18源的有限技术处置，不扩豁免或清理。非作者Dirac另完成[本批跨卡整体G4评审及Main现场接收](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/b13d-batch-review-main-reception-r01.json)，接口/来源/迁移/资源/必要集成、性能维度与覆盖按报告实际范围核收，不是三卡PASS相加。Main复核固定main/HEAD、staged0、diff/所有权/当前前置与本卡关闭条件，按task-execution-workflow §10和用户standing建议值继续授权作技术接收并关闭本开发卡，完成@2026-10-30；无Git提交约定/本轮Git写。旧G0-only、NOT RUN、待Review/OPEN等均为原时点，当前以本条和精确证据为准。设计draft/人的业务接纳仍unknown；真实provider/ACL/审批主体、SMTP/native/POSIX、完整root/S4.13绑定/真实恢复/Case18/部署/E2E由原后继任务分别保留，不据此冒PASS、不重开或关闭其他卡，原phone/缺失行限制及ADR-0024/0025窄边界不扩。

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 / 必要root依赖。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §4.2 RestoreDriver/部署独占；identity TP §4.7.1:826–828,830。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际owner为Chandrasekhar / add-backend-dev（实际runtime Agent ID `01a187a1-c022-71f3-92f5-6a9dc032e32b`），来源为Main实际Host spawn新Chandrasekhar/add-backend-dev、取得该runtime ID并派发本卡只读G0，依据用户推荐值继续授权由Main真实派发，不是用户亲指定Role或人的设计签字。本次认领登记时Main尚未接收本卡G0结果或源码/LOCAL信号，仅只读G0准备，SOURCE/LOCAL/Go/PG均closed；不表示实施启动、G0/测试通过，不授源码、运行或SQL准入。
  - `cmd/han-server/recovery_restore.go`（新增）
  - `cmd/han-server/recovery_restore_test.go`（新增）
  - `cmd/han-server/recovery_exclusive.go`（新增）
  - `cmd/han-server/recovery_exclusive_test.go`（新增）
  - `cmd/han-server/recovery_restore_integration_test.go`（新增）
- **文件数**：5；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.9`
- **批次关联**：[B13D](../plans/ir-01-content-community-batches.md#b13d)；原批次安排保持，Main已实际派发上述owner，开始@2026-10-29。本次认领登记时仅只读G0准备、Main尚未接收本卡G0结果，SOURCE/LOCAL/Go/PG均closed；后续权限须Main另发精确信号，不由认领/已闭前置或旧规划Gate推定。设计draft/人的精确接纳unknown及原技术Review覆盖保持，本次仅认领事实登记。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./cmd/han-server -list '^TestRecoveryRestoreDriver$'
go test ./cmd/han-server -run '^TestRecoveryRestoreDriver$' -count=1 -json
go test -race ./cmd/han-server -run '^TestRecoveryRestoreDriver$' -count=1 -json
go vet ./cmd/han-server
go test -tags=integration ./cmd/han-server -list '^TestRecoveryRestoreTargetReadbackPostgres$'
go test -tags=integration ./cmd/han-server -run '^TestRecoveryRestoreTargetReadbackPostgres$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryRestoreTargetReadbackPostgres$' -count=1 -json
go vet -tags=integration ./cmd/han-server
go test ./cmd/han-server -list '^TestRecoveryDeploymentExclusive$'
go test ./cmd/han-server -run '^TestRecoveryDeploymentExclusive$' -count=1 -json
go test -race ./cmd/han-server -run '^TestRecoveryDeploymentExclusive$' -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `StableArtifactHandleAndApprovedTarget` | U/local/P | 批准binary/source/target；同no-follow稳定handle贯穿校验/restore，换path攻击拒绝 |
| `PrivateStdinStrictReceiptAndCanary` | U/local | 固定argv无shell、ENV最小allowlist、私有stdin、receipt65536/65537/重复额外null/UTF8、stderr去敏 |
| `IndependentTargetAndNativeReceiptReadback` | P/local | join后重开目标独立核安装/schema/artifact/target及VerifyRestored，不以exit0/可连接/新run自证 |
| `BeginAckAndRestoreUnknownNoReplay` | U/local/P | Begin未committed零Replace，ACK/cancel/Store前崩溃回读，不新Begin/再次Replace |
| `ExclusiveAndRealChildJoin` | U/race/local | normal/recovery/requeue共享锁边界；失锁拒绝，child/pipe/target事务真实join后放锁 |

- **本任务关闭条件**：批准工具摘要/源manifest/target及no-follow稳定fd/handle贯穿核读到restore，禁止hash后换path重开；程序固定restore-v1/verify-v1 argv、无shell、ENV最小allowlist、DSN只私有stdin、stderr去敏、receipt≤65536严格JSON；exit0不代独立目标/来源/耐久回执核验；Begin未committed零Replace；ACK/崩溃unknown不得再次Replace；所有child/pipe真join后归还锁/target；独占缺失/失权failclosed。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。实际批准程序/原生receipt验证与OS锁设施、ACL/进程树隔离属真实启用前置；root注册和跨入口互斥在唯一root，真实restore在原恢复专项。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：此精确文件窗只支持已设计的中性锁/工具适配、可控进程软件验证和缺依赖拒绝。具体生产锁后端/工具/额外build-tag文件未固定，相关真实adapter部分停止并交Main/原设计owner，不在这里选SDK、发明签名协议或补一套provider。软件fixture不注册production fallback。 本卡新增integration载体直接核独立目标回读，不能只把目标核验留给后继专项；具体生产锁后端/工具未固定部分仍停止于真实启用Gate，软件适配协议可先闭环。 适用本节通用停止合同；未出现确认不当成认领/关闭。

<a id="task-identity-auth-s4-15"></a>

- [x] identity-auth-S4.15 把三个ordinary worker及lifecycle接入同一Runtime许可、实际started握手和关闭后真join，允许内部startup调谐。 开始@2026-10-30 完成@2026-10-30

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 / article Story4共享生命周期。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §4.2内部准入/生命周期；article §3.4.6；identity TP §4.7.1:795,830–831；article TP §4.8.2:765–766。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际owner为 Sartre/add-backend-dev（`01a18b91-4da7-72c3-92c7-0e2b5dbe0823`），已真实spawn/认领；当前仅READONLY G0与作者本人D候选权限，认领登记不是代码写权。
  - `cmd/han-server/recovery_workers.go`（新增）
  - `cmd/han-server/recovery_workers_test.go`（新增）
  - `cmd/han-server/lifecycle.go`（修改）
  - `cmd/han-server/lifecycle_test.go`（修改）
  - `internal/article/deletion_worker.go`（修改）
  - `internal/article/deletion_worker_test.go`（修改）
  - `internal/article/notice_dispatcher.go`（修改）
  - `internal/article/notice_dispatcher_test.go`（修改）
  - `internal/identity/password_reset_worker.go`（修改）
  - `internal/identity/password_reset_worker_test.go`（修改）
  - `internal/recovery/runtime.go`（修改，本次返工纳入）
  - `internal/recovery/runtime_test.go`（修改，本次返工纳入）
- **文件数**：12；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.8`、`article-management-S4.7`、`identity-auth-S4.12`、`identity-auth-S10.4`、`article-management-S7.1`、`article-management-S4.6`
- **批次关联**：[B13G](../plans/ir-01-content-community-batches.md#b13g)；Main已实际spawn并派发上述owner，原卡实际认领登记@2026-10-30T14:58:40+08:00（Asia/Shanghai）；来源为[Main metadata START grant](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-original-claim-metadata-start-r01.json)（绑定TaskPlanner Nash `01a18b95-c084-7f90-9952-a5462254cc15`，grant时间 `2026-10-30T06:54:03.560242+00:00`）。当前仅READONLY G0/作者本人D候选；SOURCE、Go、test、gofmt、SQL、PG、service均CLOSED，不记G0通过、代码/测试PASS或Task关闭。旧mapping/pending按原时点保留，复用原有界已接收技术合同，不因本次纯认领重开技术Planning Gate；原设计r56/TP26与Article r27/TP21及人的接纳unknown不变。
- **当前返工/失败与修复范围（2026-10-30）**：[Ptolemy独立Code Review R01](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-review-r01/report.md)及[receipt](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-review-r01/receipt.json)、[原复现合同](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-review-r01/reproduce.md)确认 `S415-CR-F001`/`S415-CR-F002` 两HIGH均在普通/race真实复现；CodeGate与B13G whole G4均为CHANGES_REQUIRED，原失败链保留，不以旧局部PASS或runner收口exit0代替产品通过。[Main采用原作者精确返工方案](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-findings-scope-sync-metadata-start-r01.json)：原卡10源仅增加上述Runtime两文件为12；本轮实际repair4仅 `cmd/han-server/recovery_workers.go`、`cmd/han-server/recovery_workers_test.go`、`internal/recovery/runtime.go`、`internal/recovery/runtime_test.go`，原owner Sartre `01a18b91-4da7-72c3-92c7-0e2b5dbe0823`不变。F001为撤权回调抢先绑定worker来源截止，新增Runtime内部同步只读 `ShutdownContext() (context.Context,bool)` 查询首次选定shutdown context（未shutdown为nil,false，Stop/fault在撤代际前发布，不造预算/不增副作用），adapter先bind notice再forward cancel；F002为健康append在最终授予锁内竞争被误报terminal，ordinary健康pause须回原changed等待，request即时拒绝及真实fault/失权/取消错误仍保留，不无限重试或吞错。原作者未改源复现后actual-release@2026-10-30T08:17:54.320074+00:00（`D:/Temp/han-b13g-s415-sartre-repro-f339a520f3ea/actual-release.json`）为LOCAL/cache已释放、pending0/真join来源；两HIGH修复及非作者复审尚未完成。原五风险已覆盖，不新增TP风险/验收命令；七依赖及其已闭历史不改，不重开旧前置或技术Planning Gate。Linnaeus另对现役identity design做精准补充，本注不代其设计完成/Review；design draft及人的接纳unknown保持。原认领/G0及NOT RUN注记保留原时点；当前SOURCE/Go/cache/SQL/PG/service仍CLOSED，metadata登记不授源码/运行权限、不记实现修复PASS或Task关闭。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./cmd/han-server -list '^TestRecoveryOrdinaryWorkers$'
go test ./cmd/han-server -run '^TestRecoveryOrdinaryWorkers$' -count=1 -json
go test -race ./cmd/han-server -run '^TestRecoveryOrdinaryWorkers$' -count=1 -json
go vet ./cmd/han-server
go test ./internal/article ./internal/identity ./cmd/han-server -count=1 -json
go test -race ./internal/article ./internal/identity ./cmd/han-server -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `InternalStartupThreeActualHandshakes` | U/C | Article→PasswordReset→notice调谐和实际循环started/Done/Err，PublicReady=false仍内部可调谐 |
| `AllScanClaimDispatchLeases` | U/race | 三worker新scan/claim/dispatch前Acquire，覆盖结果保存；临时关闭等待非terminal |
| `AppendBarrierDoesNotWaitForSelf` | U/race | 禁新claim但已持工作可结束；故障撤旧代际 |
| `SharedDeadlineNoticeFencedSave` | U/race | notice BeginShutdown、有界fenced保存、Stop至多一次、共享首次绝对截止不重置 |
| `RealJoinResourceOrderAndFailurePriority` | U/C | HTTP/维护/Tx/child/三个Done全部真join，Article资源后pool，metrics故障/提前Done不开流 |

- **本任务关闭条件**：Article deletion→PasswordReset→Article notice的Reconcile/Start与真实循环started/Done/Err；public关闭时内部Reconcile可运行，ordinary Acquire等待而非报terminal；所有scan/claim/dispatch前工作lease覆盖结果保存；append屏障停新claim但不等自身；notice BeginShutdown、同一首次绝对截止、超期仍等真Done，HTTP/维护/Tx/child全join、Article资源后pool；Stop每owner最多一次、根因优先；任一握手失败不开放。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。唯一root装配实际三实例/三个server slot；跨入口锁与恢复完整顺序后继验证。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：仅最小依赖适配/started观测，不改原lease/retry/SQL/通知算法；保留现有三个worker全部测试及lifecycle失败优先级。actor的受控reset结算driver属于恢复路径，不能通过启动ordinary循环代替。 适用本节通用停止合同；未出现确认不当成认领/关闭。

- **Main 当前技术接收与关闭（2026-10-30）**：原作者 Sartre 实际完成 source12 内的实现及四文件返工；[原独立 Reviewer Ptolemy 最终 R08](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-review-r02/report.md) 将 `S415-CR-F001/F002` 两 HIGH 均 CLOSED、无新增 finding，分别 CodeGate PASS 与本批实质 wholeG4 PASS。原两探针原文不变，独立普通/race各522最末级PASS、0FAIL/0SKIP；当前三worker/Runtime/HTTP消费者、首截止/许可健康暂停与故障/旧取消panic/真实owner join已覆盖。[Main 正式接收](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-independent-final-review-main-reception-r02.json) 已全文核原10及repair4全部实际差量、最终Runtime2delta、raw/精确源SHA/进程与回调退出；作者20jobs和Main8checks真join，当前原始48对象及44保护一致，main/HEAD固定、staged0、原30源和23旧证据不变、15949全树无越界。[共同最终检查](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-Main-final48-checks-r02.json) fresh count1普通/race各4425最末级PASS、契约4704，均16SKIP；相关42源gofmt空0、mod/vet/build0。rootfmt原23仍FAIL2，[本次42范围处置](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/root-format-current42-disposition-r02.json)不作全仓格式通过；[本任务额外工具有限接收](../../../../.spec/wip/recovery-runtime/evidence-b13g-r01/s415-current-tooling-limits-acceptance-r02.json)依用户持续建议授权采用，goimports/golangci未执行仍NOTPASS及可复查欠项，非Reviewer代豁免、非profile全绿。16SKIP已按raw纠正为15平台限制加1缺DSN，全部未验证，不推成PASS。依task关闭合同本开发卡技术关闭，三worker许可/实际started/共享关闭截止/资源先后与真join已交付；仅本卡与本批组件范围，不代production root、真实平台/provider/SMTP/E2E或人的精确接纳。design r57仍draft，原S3.3/S3.5狭义范围及S3.6不变。上述认领/G0/NOT RUN/返工CHANGES_REQUIRED保留历史时点，本条为当前接收；原7依赖/5风险/命令及其余73卡不改。

<a id="task-identity-auth-s4-16"></a>

- [x] identity-auth-S4.16 在唯一han-server组装全部同实例端口、精确命令/config/surface/维护handler，执行normal或enter/resume/verify及恢复后公共开放。

- **类型/Story与范围**：开发（同卡局部验证）；identity S04 + article Story4/8。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：identity设计 §4.2完整接线；article §8.6/Story8；identity TP §4.7.1:826–833；article TP §4.8.2:796–800。原r50及E690/E39受审技术SHA保留为原登记历史输入，不循环追新hash；原六稿PlanningReview R02不覆盖本次新增映射。本卡新增承接identity设计r52（SHA256 `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`，§3.3.1 RecoveryExit/RecoverySecretSource、§4.2完整root收口、§6.1、§9.1；[r52完整差量](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/design-r52-author-r02/design-r51-to-r52.diff)）。本次由Main授权正式登记；r52设计R02独立PASS的有界范围、Main实际接收及F001 SOURCE/LOCAL来源见S4.7下方r52承接记录。设计仍draft、人的精确接纳unknown；原六稿PlanningReview不覆盖本次新增规划映射，本增量已由Kuhn独立规划Review有界PASS/新增0，Main现场接收见S4.7下方最新记录；新增验收全部NOT RUN，不覆盖原卡既有实际自测历史。 本次C1/C2另承接identity r53 SHA `DD61F4B89C94CE91F1E1317056ACAC1D7F8F464B2963DFDC59FF9C358590A6F0` / article r25 SHA `1D1CEE350E4FD93B3B8700FD2D48F00BA2E70561FDCD8D0630B2741CF7A45F31`；[Dirac R01](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r01/review-report.md)失败链及[R02](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/review-report.md)/[receipt](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/receipt.json)仅设计F001 CLOSED/new0，Main已接收。原技术/Planningpending为各自形成时点；本次新增任务/前置/端口映射另待非作者PlanningReview，不能借旧Gate或设计Review记实现/新TE批准，新增验收NOT RUN。
- **r56/TP26正式增量映射（2026-10-30，Asia/Shanghai）**：消费identity design r56 SHA256 `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF` §4.2 ReadFailed/原域failed/Recover-Audit/cohort/ACK/Complete、§6.1/§9.1；[原Dirac两轮Design Review](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r56-review-r01/review-report.md) SHA `7C271911A16D0FC1860DF15E0362FC97FF1D6E71FD9F6D8172114D60D308C934`为有界PASS/new0，[Main实际采用](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-design-r56-main-adoption-r01.json)仅技术合同。实际消费原Jason冻结[identity TP26六行增量](../design-workspace/identity-auth/test-plan.md#b13d-s410-r56-r26) SHA `34826759C79A58EB0B4A296D2DD5C9157014406859153DC6CA0FA6E2BD921DAB`，[作者receipt](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r26-author-r01/receipt.json)与[Main冻结](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/s410-tp-r26-main-freeze-r01.json)为作者输入，不代独立Review。按[Main直接两SSOT写窗](../../../../.spec/wip/recovery-runtime/evidence-b13d-r01/planning-r56-write-start-r01.json)正式登记；原r55/r27及joint4mapping独立PASS/Main已核收事实仅复用旧范围，旧pending文字按原时点保留。本r56差量仍待原Dirac联合TP26＋Tasks/Plan独立mapping Review/Main接收，不增TP前审Gate。原claim日期/责任不改，draft/人的接纳unknown保持，新增验收全NOT RUN；Temp完整3源/typed内部factory仅未编译输入，不记codeReady、实现/测试PASS、源码准入或Taskclose。
- **C2原生命期映射（2026-10-29）**：本次只消费identity r54 SHA256 `E823D25D9CDDFD4F2D7E3A869FBB6B15EAF5275E55147A21546D4C2836472522`（§4.2生命周期、§6.1、§9.1）与article r26 SHA256 `443C55E3E89EDBE4CF0870036BB662ED2E64B1A70F8BC5609F18025DBA71D267`（§3.4.6私有namespace lease、§6.1）的[C2完整差量/作者receipt](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c2-join-design-author-r01/receipt.md)；[原Dirac独立两轮设计Review](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c2-join-design-review-r01/review-report.md) SHA256 `1354859C65D125AC5E6C3EF7576539C0250F2B8E5D677F7049529A6E18D9E4C3`为该设计差量PASS/new0，[Main技术接收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c2-design-main-reception-r01.json)采用原资源生命周期真join、无新Runtime API。旧EF531/358BE有界[Planning PASS0及L-C2-OWNERSHIP-01](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-planning-review-r01/review-report.md)保留为历史，S4.18后续真实claim不触发旧技术重审；旧pending均限当时时点。本次由Main依用户standing采用建议授权两SSOT同步，新增C2映射仍待非作者独立PlanningReview；设计PASS不关闭代码C2 OPEN，人的精确接纳仍unknown/draft，无新TP批准、源码/运行窗口或任务关闭。以下新增验收全为NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际owner Gibbs / add-backend-dev，actualID `01a18c1e-6c1e-77a2-a274-752e748d3c8d`；开始@2026-10-30T17:31:16.835005+08:00（Asia/Shanghai；UTC `2026-10-30T09:31:16.835005+00:00`；本次认领登记时点）。当前仅READONLY G0/本人uniqueD草拟，SOURCE、Go、cache、SQL、PG、service均CLOSED；认领不授代码写权。
  - `cmd/han-server/recovery_runtime.go`（新增）
  - `cmd/han-server/recovery_runtime_test.go`（新增）
  - `cmd/han-server/recovery_config.go`（新增）
  - `cmd/han-server/recovery_config_test.go`（新增）
  - `cmd/han-server/recovery_command.go`（新增）
  - `cmd/han-server/recovery_command_test.go`（新增）
  - `cmd/han-server/recovery_sources.go`（新增）
  - `cmd/han-server/recovery_sources_test.go`（新增）
  - `internal/identity/recovery_handler.go`（新增）
  - `internal/identity/recovery_handler_test.go`（新增）
  - `cmd/han-server/recovery_runtime_integration_test.go`（新增）
  - `cmd/han-server/main.go`（修改）
  - `cmd/han-server/main_test.go`（修改）
  - `cmd/han-server/lifecycle.go`（追加support，修改）
  - `cmd/han-server/lifecycle_test.go`（追加support，修改）
- **文件数**：15；单writer完整闭环，不按文件数拆卡。
- **有界support来源（2026-10-30）**：[Main精确scope采用](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s416-G0-main-reception-and-scope-decision-r01.json)及[metadata START](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s416-support-scope-metadata-start-r01.json)仅将上述两项support追加到原13核心，总15，仍由同一Gibbs writer承接。第三maintenance server的Shutdown错误须由既有lifecycle统一收集并触发Close/Serve真join，additionalServers的nil项须校验拒绝；保留原fault/unwind、共享deadline、错误优先级和资源顺序，不重开S4.15、不新增Runtime API或第二协调器。Gibbs旧READY已superseded；此为静态G0缺口及精确范围采用，复现/测试均NOT RUN，非G0READY或源码/Go/cache/SQL/PG/service准入，draft/人的接纳unknown保持。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.8`、`identity-auth-S4.9`、`identity-auth-S4.10`、`identity-auth-S4.11`、`identity-auth-S4.12`、`article-management-S4.7`、`article-management-S4.8`、`identity-auth-S4.13`、`article-management-S8.3`、`identity-auth-S4.14`、`identity-auth-S4.15`
- **批次关联**：[B13H](../plans/ir-01-content-community-batches.md#b13h)；Main已实际spawn并派发上述owner，本次由Boole / add-task-planner（actualID `01a18c20-a70c-7972-b634-9520991a7bbc`）依[Main metadata START](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s416-original-claim-metadata-start-r01.json)登记原卡认领，grant时间 `2026-10-30T09:27:35.342300+00:00`（UTC）。仅登记认领，不记G0READY、PlanningReview、CodeGate、测试PASS或Task关闭；既有有界技术接收与旧pending原时点保留，不重开技术Planning Gate，design draft/人的精确接纳unknown保持。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test ./cmd/han-server -list '^TestRecoveryRuntimeComposition$'
go test ./cmd/han-server -run '^TestRecoveryRuntimeComposition$' -count=1 -json
go test -race ./cmd/han-server -run '^TestRecoveryRuntimeComposition$' -count=1 -json
go vet ./cmd/han-server
go test -tags=integration ./cmd/han-server -list '^TestRecoveryRuntimeProductionChain$'
go test -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json
go vet -tags=integration ./cmd/han-server
go test ./internal/identity -list '^TestRecoveryControlHandlers$'
go test ./internal/identity -run '^TestRecoveryControlHandlers$' -count=1 -json
go test -race ./internal/identity -run '^TestRecoveryControlHandlers$' -count=1 -json
go test ./cmd/han-server -run '^TestPasswordResetMaintenanceRoot$' -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `SingleFactoryPortsAndExactSurface` | R/P | 同pool/Store/Journal/source/provider/Runtime/registry，无第二authority/router；r56仅后继原recovery_runtime.go/recovery_sources.go在原13源内以typed internal私有装配首次使用前固定S4.10同一executor/同Store的RecoveryFailedEvidenceReader，桥接原独立批准provider/原件scope来源并原样委派；root不复制原域Recover业务/SQL，不另造pool/Store/Runtime/DTO或approvedbool。原TestRecoveryRuntimeComposition/ProductionChain接通实际reader返回与同receiver，fixture仅软件正向不注册生产fallback；真实provider/ACL不明只禁受影响resolve/需该源的Complete，未用来源不变成normal/verify配置前置，具体provider产品不在本卡选择 |
| `ConfigCommandAndLazySecret` | U/C/R | 精确normal/enter两形态/resume/verify，非法参数零副作用；适用配置前置，secret按mode惰性且root RecoverySecretSource只读候选、Journal按retained Begin key验密，Runtime再核来源/许可才PrepareTrust；已准备/已改密零读取/验证，root不持key/复制HMAC；原requeue合同保持 |
| `ControlLoopbackAllowlistAndStrictBody` | C/R | 数值loopback/不重叠/HTTPS、七精确路由、body65536/N+1/UTF8/重复字段/cursor-limit、Auth/CSRF/phase；r56保原idempotency resolve精确body(scope_hash,key_locator,execution_generation,decision=failed,evidence_ref)，scope/locator严格解为32bytes、generation正int64，七路由/原body上限与Auth/CSRF/phase/Trace保持；handler只将原输入交同executor核原件/原域/原cohort，不收body来源wrapper/时钟/actor/approved字段，不调用generic Recover或新增retry路由。complete仍原{}并委派同executor完整C/逐P/cohort证明，不能COUNT/DB failed/approvedbool代证；关键C/R仍原TestRecoveryControlHandlers及本卡生产链，不将本root反向挂到S4.10局部P |
| `AdminBoth503AndZeroDownstream` | R/P | 真实factory两个GET各503+Trace，service/repo/author-reader各0；404/405/Allow兼容，不用拒启/spy造响应 |
| `CompleteSoftwareRecoveryAndOpen` | R/P | Begin→Replace独立目标→running→F001/F002/canonical/幂等→freshConsume/publication/双namespace→MarkConsumed→succeeded→撤维护真join→三握手→最终proof→同Session普通链 |
| `FaultUnknownNeverActivatesAndUnwind` | R/P | 任一持久点/worker/metrics/撤会话ACK未知不开；verify0listener-worker-write，资源反向归还 |
| `RootExitTrueJoinAndSameExecutor` | R/P | 原cmd/han-server/recovery_runtime.go实现root-owned RecoveryExit，复用同一IdentityRecoveryExecutor/Store；complete handler仅写响应并通知、先真实返回，独立root控制流程用保Trace的有界ctx禁新维护HTTP/登录/写/settlement，等全部HTTP/强改密/settlement/Tx/子工作真实join后才调用RevokeRecoverySessions；不在complete调用栈等待自身、不在cmd写SQL。原recovery_runtime_integration_test.go的TestRecoveryRuntimeProductionChain以真实handler/Session/SQL和阻塞complete/settlement证明顺序；U/C补原TestRecoveryRuntimeComposition与TestRecoveryControlHandlers |
| `ExitUnknownBlocksWorkersAndFinalProof` | R/P | 同一生产链top覆盖Consumer成功但出口缺失/失败/ACKunknown、残留会话/错安装/取消/失锁/未join均零ordinary启动/公共开流；真实join且同Store撤销审计确认后Runtime撤旧维护/replay许可→三worker实际started/Done-Err→最终完整来源复核→同Session普通链；维护cookie不升级、旧代际拒绝。精确锁序/SQL原子性由前置S4.10 P交付，root本卡仍用真实同receiver接通关键P/R，不推给S4.17 |
| `InstallRootDrainAndSingleBinding` | U/C/R/P | 原recovery_runtime.go/main.go装配同Runtime/同Article控制handle与同backend实际键的AnchorReader view，recovery_sources.go桥接独立批准/commitment与只读Anchor，原config/command形态不变；原recovery_handler.go配合当前handler实际返回后独立控制流程。原TestRecoveryRuntimeComposition/ControlHandlers及TestRecoveryRuntimeProductionChain核普通/维护HTTP/强改密/settlement/Tx/子工作和三worker先禁新/真join，不能bool/只停两worker/要求succeeded的RecoveryExit替代；无I/O私有单次绑定，未绑snapshot关闭，executor返回前绑同handle，本地先供snapshot、Runtime后证再接受；Install成功撤grant仍closed，全部使用者结束后Close控制handle/清key/Stop归还独占，再另建closed normal全证明才开流 |
| `LongLivedControlHandleTrueJoin` | U/C/R/P | 原recovery_runtime.go/main.go的root/Lifecycle持有同一长期Article control handle及Close/join责任，Install=nil不表示资源已关或自动注册普通work；原recovery_runtime_test.go/TestRecoveryRuntimeComposition与recovery_runtime_integration_test.go/TestRecoveryRuntimeProductionChain在真实factory链核调用/使用者先终结、长期handle实际Close/join（并发Stop同owner清算）后才归还deployment、最后关pool。仍owned/Close unknown期间以实际heldfile证明原请求/工作及root清算未结束、重复Stop未提前完成/未二次Close、deployment Close0/pool未关；同owner真实一次终结并join后才清算一次且保留原error/panic及其他cleanup cause。请求WorkLease只由原调用终结后释放；不靠已返回Install/观察bool/普通work map推长期handle已收口，不新增Runtime债API。关键R/P仍在本卡原13文件/原生产链top自闭环，Article局部C2不反向等待本卡 |

- **本任务关闭条件**：同pool/Store/journals/CAS独立keys/source/provider/decoder/Coordinator/Runtime/registry指针和构造次数；config全部前置校验、相关secret惰性读取；recovery命令精确解析并在旧maintenance parser之前分流，verify零listener/worker/repair；normal/recovery/requeue同launcher独占；control数值loopback+不重叠port+批准HTTPS Origin；七条maintenance allowlist和live/ready，公共recovery404、错方法405精确Allow、Trace最外且guard在Session前；review/resolve/complete严格body/initial/CSRF/phase，F001两合法态及F002原子204；Begin→Replace核验→running→identity→freshConsume→succeeded→撤维护/真join→三worker就绪→最终proof→公共open，失败unknown不开；构造反向回收及三server结果容量。 新增出口链必须complete实际返回→禁新维护/settlement并真实join→同executor RevokeRecoverySessions确认→撤维护能力→三worker握手→finalproof后open；失败/ACKunknown保持关闭，root不复制身份SQL。原13文件/原生产链top完成关键成功及失败P/R，本卡关闭不等待后继专项/真实E2E。 新增Install root/drain/单实例关键P/R仍原13文件内自闭环，不把它们提前倒挂Article C1；消费真实前置executor，不在cmd复制安装SQL。Normal/Verify不因安装不适用配置拒绝；新port缺实际来源时动作前关闭，不注册production no-op。需原13之外lifecycle/settlement源须停并交Main精确边界，不能自行扩文件。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。其自身局部条件满足可关闭，不等待后继独立专项或需求整体验收。 C2另须显式核长期control handle的root-owned Close/真实join：Install返回不自动成为普通work，所有使用者及namespace资源确切终结后方可归还deployment且pool最后关；并发Stop不绕过同owner/同drain清算，错误/取消/panic/未知终结保持关闭与原cause，其他cleanup失败不清除。本卡关键双侧资源证据留在原R/P，不推后继S4.17，也不反向成为Article/S4.18局部关闭门槛；无确切平台终结来源则pending，不把portable组件当native/root通过。 r56后继reader接线仅原13源/10组：typed内部构造固定原独立来源、原strict body/handler委派同executor，root不复制业务Recover/SQL或造新authority；缺真实provider/ACL只关闭对应resolve及需源Complete，无fixture/no-op生产fallback，normal/verify不因未用源受阻。S4.10原域来源/原子Audit/cohort/ACK由前置本卡交付，root关键R/P接通仍留本卡，不倒挂S4.10/Article软件关闭，原出口/长期handle真join/最后关pool责任不减。
- **后续验证关联**：identity-auth-S4.17（新增跨层软件专项）、原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6各自保持。原Case18、两S4.4、S1.5/ACL和部署E2E分别接收。受控外部依赖fixture贯穿真实factory/Session/SQL正向仅证软件链，不宣称真实provider/Host。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：main两文件由本候选唯一串行写入；S8.2当时释放不保证现在无writer，须Main重核。recovery_sources仅桥接既有可信来源接口、缺依赖拒绝，不从文件自带hash/备份role或刚写批准自证。具体生产source/CAS/工具注册仍unknown，不发明产品。此卡必须实测full软件成功链，只有拒启不得关闭为完整装配；生产启用保持额外真实前置。 本卡独占recovery_runtime_integration_test.go，真实Session/SQL/factory成功与关键失败链在本卡闭环，不依赖后继identity-auth-S4.17才能关闭。 适用本节通用停止合同；未出现确认不当成认领/关闭。

- **Main 当前技术接收与关闭（2026-10-31）**：本卡 15 文件实现及 F009/F010 返工由原作者 Faraday / add-backend-dev（actualID `01a18d45-b04b-7581-a172-0296b0341ccb`）完成并停止写入；非作者 Boole / add-reviewer（actualID `01a18e02-451f-78c1-89d2-dea8b9af0f63`）[r03 独立复审](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s416-independent-review-r03/report.md)与[receipt](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s416-independent-review-r03/receipt.json)结论 CodeGate PASS，F001–F010 全部 CLOSED（F009 HIGH、F010 LOW 本轮关闭）、新增 finding 0。Lead 现场以 [r05 manifest](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/lead-gate-r05/files-sha256-r03.json) 第一手重算 15/15 SHA256+字节一致、无第 16 文件、`internal/recovery` 未动、HEAD `6da8abb` 且 staged 0；[r05 Gate](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/lead-gate-r05/gate-summary-r05.json) 13 条命令全部 exit 0，F009/F010 定向普通与 `-race` 全绿，整包 unit 2683 pass events / 0 fail，r03 独立重跑整包普通/race 各 418 顶层 PASS。全量 integration 顶层 168 PASS / 1 FAIL，唯一失败 `TestRecoveryRestoreTargetReadbackPostgres` 属既有 S4.14 测试硬编码 `127.0.0.1:51934/han_b13d_restore` 的环境前置（本卡 harness 为 `53519`），非 B13H 回归。按用户 standing 采用建议授权及 task 关闭合同，本开发卡技术关闭；不代人的业务签收、真实 provider/ACL、真实部署 E2E 或后继 identity-auth-S4.17/B13I。设计仍 draft、人的精确接纳 unknown；生产 CAS/anchor、installation commitment、source verifier、deployment lease、keyring loader、restore adapter 与真实 provider/ACL 未交付，recovery 仍显式 closed；`goimports`/`golangci` 未运行，F009 红色证据未持久化，F010 permit 绑定仍为 fixture；r05 摘要 `secret_scan` 元数据措辞已更正为泛化表述（原仅列类别名、无真实密钥值，不涉及产品代码与 Gate 结果）。

<a id="task-identity-auth-s4-17"></a>

- [x] identity-auth-S4.17 在真实PG和唯一production factory/真实Session/源/Finalizer链验证新增runtime跨层行为，外部媒体/工具仅明确可控adapter。

- **类型/Story与范围**：专项集成测试（跨域/root软件专项，非部署E2E）；identity S04 + article Story4/8；独立专项非部署E2E。新增责任是本恢复runtime行为闭环，不复刻旧已闭实现。
- **设计/测试依据**：两设计恢复切片与可测试性；两TP集中恢复增量全部受影响场景；原Case18/A-REC/I-REC接收。精确修订及E690/E39受审技术SHA见本节稳定输入；本卡与当前mapping/meta仍待PlanningReview，全部未来命令NOT RUN。
- **文件所有权**：仅以下新增/修改清单；实际执行者尚未派发/未认领，当前规划登记不是代码写权。
  - `test/integration/recovery/runtime_admission_test.go`（新增）
  - `test/integration/recovery/runtime_recovery_test.go`（新增）
  - `test/integration/recovery/runtime_fixtures_test.go`（新增）
- **文件数**：3；单writer完整闭环，不按文件数拆卡。
- **依赖**：requires `identity-auth-S4.16`
- **批次关联**：[B13I](../plans/ir-01-content-community-batches.md#b13i)；由本pass Batch-Plan消费原任务安排，实际执行者未派发/未认领，状态仍[ ]。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存Gate见通用合同）：

```text
go test -tags=integration ./test/integration/recovery -list '^TestRecoveryRuntimePostgres$'
go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json
go test -race -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json
go vet -tags=integration ./test/integration/recovery
go test -tags=integration ./cmd/han-server -list '^TestRecoveryRuntimeProductionChain$'
go test -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeProductionChain$' -count=1 -json
```

- **必要风险leaf / 最低足够层**：下面英文是未来命名风险组，不新增业务CaseID；若分布在主top与对应P/R top，应逐层同名命中并保留各自raw，不能拿父top PASS代子组。此表为本卡局部最低层；TP要求的唯一production root接入R集中在S4.16本开发卡及S4.17独立专项，其他模块的组件handler/循环不冒充production root R。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `GenesisDeletionHistoricalRestart` | P/R | 独立完整合成权威+真实Store/source/Finalizer/Session/root证据串联，日常删除后历史重启 |
| `CrossNamespaceABAAndExactPublication` | P | 双namespace/Store交错、publication历史fresh分流/回退拒绝 |
| `F001F002AndCanonicalIdempotencyReview` | P/R | 复用本卡已闭局部证据，再核跨域维护/恢复重置/canonical/幂等边界，不重复造固定Ready |
| `CrashConsumedWindowAndDrain` | P/R | 唯一受控consumed窗口及普通/维护/三worker/child完整join、unknown继续隔离 |
| `Case18SameRevisionAndNoProductionClaim` | R/P | 同revision两GET503/零下游与完整开流；软件adapter不证明真CAS/POSIX/ACL/批准备份/E2E |

- **本任务关闭条件**：全量历史/双namespaceABA、genesis→合法删除→重建、published精确历史与fresh拒stale、F001/F002正向及原子回滚/ACK/unknown窗口、原始Begin/Replace/身份/Consume/MarkConsumed顺序与唯一补写、三worker阻塞交错/真join；Case18两个实际GET各503/Trace/Article service+repo+author-reader全0，full软件成功后同真实Session/AccessGeneral正常读取；404/405/Allow/旧Origin-CSRF-IfMatch兼容；schema/连接/child留存账完整。 必须满足本卡所列最低层及通用关闭合同；不是只交接口名或build成功。跨层软件专项全组P/R证据及非作者Review/Main同版核收齐全后按本卡关闭；真介质/批准旧备份/真实provider/E2E仍由原任务，不能假称本专项已证明。
- **后续验证关联**：原identity-auth-S4.4/article-management-S4.4、article-management-S1.5/S8.2/Case18；部署后identity-auth-S3.4/article-management-S1.4与S3.6保持。原identity/article S4.4承担真Journal/批准旧备份/生产CAS/retirement/RPO等；S1.5保留16endpoint/三投影/S1–S16/ACL；S8.2/Case18整体由Main同revision核收；部署后S3.4/S1.4另行真实E2E。 仅关联，不新增旧卡requires，不扩大旧测试卡生产写权。
- **串行/停止条件**：独立test-only三路径，不写生产、不占原两S4.4三路径、旧fixtures或autoDROP harness。真实root P/R载体已归identity-auth-S4.16本开发卡；本专项可复用其固定证据并另验跨层组合，不能反向成为该开发卡关闭条件。 适用本节通用停止合同；未出现确认不当成认领/关闭。

### 原验证卡接收关联（只引用，不修改原卡）

原identity S4.4、article S4.4、S1.5、S8.2/Case18分别消费上列新软件同revision交付证据后再核自身原关闭条件；原三路径、六/八/十二requires、真实层与审计ACL/普通安全链不缩、不借软件交付直接关闭。ADR0024只关闭S4.3指定真卷范围，S3.3只免真实browser/S3.5只免真实platform工具且均非PASS，S3.6不变。当前原卡和新增卡均未被本次关闭或改认领。
- **Main 当前技术接收与关闭（2026-10-31）**：本卡三文件 test-only 实现及 r01→r03 返工由原作者 Godel / add-backend-dev（actualID `01a18e79-406a-7822-9dcf-f14aae6cfbc0`）完成；非作者 Beauvoir / add-reviewer（actualID `01a18eda-73ae-75d3-bc25-677bb98211dd`）[r02 独立复审](../../../../.spec/wip/recovery-runtime/evidence-b13i-r01/s417-independent-review-r02/report.md)与[receipt](../../../../.spec/wip/recovery-runtime/evidence-b13i-r01/s417-independent-review-r02/receipt.json)结论 PASS，r01 的 H01–H05/M01–M06/L01–L04 共 15 条全部 CLOSED、0 PARTIAL/0 OPEN，5 个 leaf 全 PASS；另留 N01/N02 两条非阻断 LOW（保留 schema 登记路径写入作者证据目录、`s417FullWorld` 过时注释），不改 verdict。Lead 本机以 SHA256+字节复核三文件一致（`AF8767…E158`/1021、`AE1A68…C516`/58621、`D27AD6…5593`/105922），HEAD `6da8abb`、staged 0、`internal/recovery` 未动；[Lead Gate r02](../../../../.spec/wip/recovery-runtime/evidence-b13i-r01/lead-gate-r02/run-results.json) 7 条验收命令+`gofmt -l`+Case18 `-v` 全部 exit 0，recovery 普通/`-race` 各 92 JSON 行、pass 7/fail 0/skip 0、5 subtest 全 PASS，production chain 普通/`-race` 各 pass 2/fail 0/skip 0，vet 空输出、gofmt 0 行；Case18 `-v` 实测 `statuses=[503 503]`、两个不同 trace、下游 handler/service/repo/author_reader 全 0，开流 200、ETag 取首次 GET、下游全 1，404/405/`Allow="DELETE, GET, HEAD, PUT"` 及 Origin/CSRF/If-Match/204 全链路成立。ABA/fresh/stale/rollback 由真实 PG `article.PostgresReplayEvidenceStore`/`NewReplayPublicationPorts` 产生，consumed 窗口由真实 `runtime.Enter`→`identity.NewRecoveryIdentityExecution`→`recovery.Coordinator.Consume`→`coordinator.go:275 journal.MarkConsumed` 产生（`consumed_marks=1`），maintenance 走真实 `recoveryExecutionGate`，Case18 走真实 `http.ServeMux`；三文件无 `t.Skip`、无 `DROP SCHEMA`、无硬编码 DSN/5433。D-B13I-02 前提与授权边界经复审成立，但仅限本层测试证明。schema 全保留：53519 实测 `han_b13i_s417_%` 181 个、`han_b13h_chain_%` 88 个，登记表 182 条（含 5433 历史事故 schema），无任何 DROP。按用户 standing 建议授权与 task 关闭合同，本卡技术关闭；不代人的业务签收、G4-batch、真实 POSIX/CAS/ACL/批准备份/部署 E2E。`canonical_recovery_replay_same_result` 仍未证明；Case18 精确路由不等于生产 `main` dispatcher 整链；归属 unknown（`go.mod` modified、`test/ internal/ cmd/` untracked，仅三文件 SHA 固定）；单实例 trust PG、单次 `-race` 无 soak；`goimports`/`golangci` 未运行；既有 `TestRecoveryRestoreTargetReadbackPostgres` 环境性失败非本卡回归；N01/N02 保留待后续。design 仍 draft，人的精确接纳 unknown；后续仍为 3 条 active（article-management S1.5/S4.4/S8.2）与 8 条 todo，不在本卡范围。

<a id="task-identity-auth-s4-18"></a>

- [x] identity-auth-S4.18 实现公开Runtime.Install的中性producer、独立安装来源前后证与精确Article协调checker，保持关闭并真实join。 开始@2026-10-29 完成@2026-10-29

- **类型/Story与范围**：开发（同卡局部U/C/race）；identity S04 / US-IDENTITY-03.1恢复/安装接缝。独立补足未交付的Install producer；原identity-auth-S4.7整卡[x]/历史及已闭finding保持。完整fresh/prepared安装SQL归S4.13，真实root/维护drain归S4.16，Article组件归原article-management-S4.7。
- **设计/当前来源**：identity design r53 SHA256 `DD61F4B89C94CE91F1E1317056ACAC1D7F8F464B2963DFDC59FF9C358590A6F0` §3.3.1/§4.2/§6.1/§9.1；article design r25 SHA256 `1D1CEE350E4FD93B3B8700FD2D48F00BA2E70561FDCD8D0630B2741CF7A45F31` §3.4.6/§6.1。[原设计差量](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-author-r01/identity-auth.diff)及[r25修复](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-author-f001-r02/article.diff)保留；Dirac [R01](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r01/review-report.md)唯一MEDIUM失败链保留，[R02](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/review-report.md) SHA256 `7AC0B3443B07E744CE6CA613E67C31D8E24BEB5FC6A790E5D3274190744F32AD` / [receipt](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r02/receipt.json)仅关闭设计F001/new0，Main已明确接收技术差量。设计仍draft/人的精确接纳unknown；本卡及新增映射尚待非作者PlanningReview，旧Gate不扩覆。两TP未修改，不宣称新TE批准；r50/E690/E39保留历史技术输入，不循环追hash。
- **R对账与责任**：

| 标签 | 定位/实际证据归属 | 本卡与后继分界 |
|---|---|---|
| 未建成 | 原owner [C1咨询](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-consult-r01/consult.md)与原Zeno [G0](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-runtime-g0-r01/g0receipt.md)实际读取指出runtime.go:160–190缺Install/安装依赖/checker；非本Planner亲读源码 | 本卡只在两源补真实公开producer/checker，不用private phase/fake Runtime正控 |
| 已建成可复用 | 原S4.7 [精确最终接收](../../../../.spec/wip/recovery-runtime/evidence-b13a-r01/main-final-reception.json)与原S4.8已交付contextual Journal；当前G0所核既有Runtime | 原历史/opaque grant/正常准入/Enter-Resume/Stop-Verify仅原验收范围，不外推完整Install |
| 需改 | Article原8的C1/C2/供证、S4.13原5的executor、S4.16原13的root | 原卡各自局部闭环；不加Article第9源，不倒挂本卡关闭 |
| unknown | 真实批准/provider/OS/POSIX/root/PG/E2E及人的签字 | 不阻合法组件fixture准备；仍阻相应真实启用/结果 |

- **可行性与命名**：原Zeno [G0receipt](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-runtime-g0-r01/g0receipt.md)及[validationplan](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/c1-runtime-g0-r01/validationplan.md)由Main全文核收、STOP/Hostclosed，限两源可行/无第三源或新契约阻断，非SOURCE/测试PASS。G0旧top子组只是未采用候选；正式统一新top `TestRecoveryRuntimeInstall`。原 `TestRecoveryRuntimeAdmission` 写前271所有leaf名称/期望/必要命中及原八组完整保留，不冒旧230数，不合并新六组掩盖遗漏。
- **文件所有权**：仅两项修改；实际owner为原Zeno / add-backend-dev（真实Agent ID `01a18486-3e11-7852-9723-ad380e0bb392`），Main实际恢复，依据用户standing采用建议继续授权；G0原两源可行已核收。本次仅C1既有合同SOURCE/LOCAL，不授未决C2 namespace cleanup新合同/PG/第三源。
  - `internal/recovery/runtime.go`（修改）
  - `internal/recovery/runtime_test.go`（修改）
- **文件数**：2；同一公开控制行为的局部闭环，非按文件数机械拆分。
- **依赖**：requires `identity-auth-S4.7`、`identity-auth-S4.8`
- **批次关联**：[B13C](../plans/ir-01-content-community-batches.md#b13c)；规划初始未认领为历史时点，Main现已实际认领并发[C1两源SOURCE/LOCAL](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/s418-source-local-start-r01.json)。Task/Plan固定差量独立PlanningReview有界PASS，L-C2-OWNERSHIP-01仍上游限制，不覆盖新cleanup场景或人的签字。
- **验收命令**（根目录，待实施/NOT RUN；另需Main SOURCE/LOCAL）：

```text
go test ./internal/recovery -list '^TestRecoveryRuntimeInstall$'
go test ./internal/recovery -run '^TestRecoveryRuntimeInstall$' -count=1 -json
go test -race ./internal/recovery -run '^TestRecoveryRuntimeInstall$' -count=1 -json
go test ./internal/recovery -list '^TestRecoveryRuntimeAdmission$'
go test ./internal/recovery -run '^TestRecoveryRuntimeAdmission$' -count=1 -json
go test -race ./internal/recovery -run '^TestRecoveryRuntimeAdmission$' -count=1 -json
go vet ./internal/recovery
```

- **必要风险leaf / 最低层**：六组均在原runtime_test.go的新TestRecoveryRuntimeInstall下普通/race逐组实际命中；待实施符号，非新业务CaseID。list/零命中/Skip/编译或环境失败不算PASS。

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `ModeDependenciesAndDispatch` | U/C/race | 真实NewRuntime→公开Install，原v1manifest/扁平namespace编码、ctx有限预算/typed-nil/互斥；fresh/prepared/bound按分支检查，缺依赖在drain/exclusive/写前拒绝；Normal/Verify不要求安装专用字段；完整fresh/prepared算法经固定executor显式fixture分派，不复制SQL/生产空壳 |
| `BoundSourcePrePostProof` | U/C/race | bound同manifest独立批准/commitment、完整Store/recovery namespace、同backend实际键article Anchor及tokens前后重读；前证不调normal pending拒绝；pending仅到原H0/H1且新revision，committed head/token不换；原批准前缀及其他来源不变，H2/回退/ABA/未知拒绝，不改Store/run/epoch |
| `ExactCurrentGrantAndPendingCopy` | U/C/race | 公开producer授同实例当前install grant及同PhaseContext；rawpermit/ctx/实例/代际/目的/安装/id/epoch/closed-drain/exclusive全等；捕获与返回Pending深拷贝，checker协调后仍返封存起点；观察值不授权，不用normal.last；零/错/过期与ordinary/startup/replay冒用拒绝 |
| `ExecutorReturnIsNotProof` | U/C/race | executor及子I/O实际返回后独立Anchor+同handle snapshot后证；snapshot先供证、不等后证先成立；伪nil/错head-token/Store-recovery-commitment漂移/ACKunknown保持关闭且不盲重试；fixture仅隔离外部设施/具体安装，Runtime真实 |
| `StopCancelsAndJoinsInstall` | U/C/race | 分类/drain/独占/前证/executor-CAS/后证阻塞交错，先撤grant/ctx再真join；迟到成功不复权、不早释namespace/deployment所有权；mutex不跨port/I/O/join、checker回调无自等待，panic不留活grant/不吞根因 |
| `InstallNeverOpensAndKeepsOwnCause` | U/C/race | 冷启动也调用真实lifecycle port，不用drained bool/RecoveryExit；成功撤grant且不开worker/普通lease、不清他人cause；单次控制实例不复位SetReady，未来normal从closed另建全证明；原Admission271/八组、Enter-Resume/Stop-Verify/own-cause和HMAC合同逐叶回归 |

- **本任务关闭条件**：两源真实公开Install/checker、分支依赖/独立前后证/许可撤销/真join及新六组普通/race全闭合，原Admission写前271所有leaf/八组不丢不弱化；现役Manifest检查/源格式/精确diff、失败修复raw、非作者CodeReview/修复复审、Main同revision接收齐备。仅显式中性fixture承接外部设施/批准/尚未交付完整executor，不反向import article/identity/pgx，不用private phaseLocked/unsafe/reflection造正控。本卡局部U/C关闭不等Article C1/完整S4.13/root/真实E2E，不能称完整安装/Article跨包C/生产通过。旧S4.7格式例外不自动继承，检查失败实际保留交Main处置，不预写7/7PASS。
- **后续验证关联**：ArticleS4.7真实Runtime→控制opener/Coordinate→本地核验供snapshot→Runtime独立后证C及原C2/P；S4.13完整fresh/prepared安装/同Store SQL、S4.16真实root/维护drain/单实例与退出，S4.17独立组合、原两S4.4/Case18/真实provider/部署E2E各自保持。仅关联，不倒挂本卡关闭。
- **串行/停止条件**：closed S4.7原两Runtime源冻结/释放后本卡唯一writer；Noether原8不暗写第9源。当前ArticlePG1+C2two active、S49六源Review由Main协调，本登记不干预；Main先核读取Runtime包的构建/测试窗口，本卡STOP/Review/接收后Article才消费冻结输出。需第三源/新契约/新依赖、未知writer或原271缺失即停受影响部分，不自行改admission/journal/S49/Go依赖或设计。SOURCE/LOCAL/PG/进程/cleanup均另行精确信号，旧autoDROP不运行，原失败证据保留。

- **Main task-level 接收与关闭（2026-10-29）**：原Zeno两源STOP/冻结，原Ptolemy[两轮独立CodeReview](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/s418-review-r01/review-report.md)有界PASS/finding0，Main[全文与raw接收](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/s418-review-main-reception-r01.json)确认独立115+271+56每模式和旧名称/正文保护。Main[同修订实际联合执行](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/main-current-joint-r01/receipt.json)普通/race各616叶（原Install105/Admission271/HTTP56及Article145/old39），S49编码41/真实PG26及ArticlePG13均通过，当前实际Store→本Runtime.Verify正负链已重跑；缺依赖分支、前后证/当前许可/真join/不开流六组保留。Manifest六项PASS/源18Go fmt空，根fmtFAIL2旧23字节同；Main采用[本修订两卡精确有限处置](../../../../.spec/wip/recovery-runtime/evidence-b13c-r01/root-format-disposition-before-c1-r01.json)，不继承旧格式例外或标全七PASS。Main源哈希/status/diff/HEAD/staged/越界及成果核证完毕，现两源可交Article原owner消费。仅中性producer/checker局部技术关闭，Article真实C1仍NOT RUN待另授原8 SOURCE，不冒完整S4.13/root/S4.16/native/批次/E2E/容量或人的接纳。当前测试/fixture已join，旧原生linker55164 OSjoin未知另列，不称全宿主已join；旧pending均为历史时点。

<a id="task-identity-auth-s4-19"></a>

- [x] identity-auth-S4.19 将唯一production cmd/han-server普通业务dispatcher接入同一recovery runtime的RecoveryGuard，在exact router之后、Session/AccessGeneral之前阻断recovery closed/unknown，并由同handle提供公共GET /livez、/readyz。 完成@2026-10-31
- **ADR-0028 适用（2026-10-31）**：按项目级验证授权执行；可执行层仍实际执行留证，真实环境可模拟并标注“未真实验证”，评审/签收可由 AI 代理并记录授权来源，不冒充真人签字。
- **R01 独立评审 FAIL / R02 停止（2026-10-31，历史时点；后续见下条 R02/R03 记录）**：非作者独立评审 R01 为 FAIL，`S419-CR-F001`/`S419-CR-F002` 为 MEDIUM 验收证据缺口（非生产缺陷）；作者 R02 仅有探测文件、无最终证据包，已停止且无新评审报告。本卡保持 `[-]`；按 ADR-0028 优先补 F001/F002 可执行测试，无法驱动完整链时正式收窄并登记未验证边界，不得伪造 PASS。

- **目标**：在唯一production `cmd/han-server`普通业务dispatcher上，以同一recovery runtime handle完成exact route→guard→Session/AccessGeneral接线，并提供同handle的GET /livez、/readyz；用真实dispatcher整链测试证明closed/unknown两管理员GET 503+Trace且Article下游零调用，共享 runtime 进入 `public_ready` 后同一 Session 正常 200（完整恢复链由同修订必跑的真实 PG 测试证明），且既有404/405/Allow、Origin/CSRF、Session/AccessGeneral与开放链兼容。

- **类型/Story与范围**：开发（同卡局部真实production dispatcher整链测试）；identity S04唯一root补充分项，关联article Story8/Case18但本卡不关闭、不重开article-management-S8.2。本卡只补生产main普通业务接线与同handle探针；组件guard归已闭article-management-S8.3，root assembly归已闭identity-auth-S4.16，cross-layer test-only卡identity-auth-S4.17仅作后续关联，不复用其关闭结论。不含部署E2E、真实POSIX/CAS/ACL/批准备份/provider，不建第二runtime/router/pool/authority。
- **R对账**：
  - 未建成/需改：`cmd/han-server/main.go:543`在recovery assembly（`:596–629`）之前组合business handler，`:653`直接把未包guard的handler交给business server；`newLoginBusinessRouterWithTrace`（`:3475–3615`）只有exact path/method、404/405/Allow与Trace，无guard/probe。
  - 已建成可复用：`internal/platform/recovery_guard.go:17/27/51/56`已有NewRecoveryGuard/Protect/Liveness/Readiness，且Protect要求exact routing先行、以既有503 envelope阻断并在handler返回/panic后Release；`recoveryRootAssembly.runtime`（`recovery_runtime.go:437`）已是同handle。
  - 不足证据：`identity-auth-S4.16`（tasks.md:2636）的`TestRecoveryRuntimeProductionChain`（`recovery_runtime_integration_test.go:165`）手工构造handler/harness，未覆盖`main`普通dispatcher；`identity-auth-S4.17`（tasks.md:2742）明确Case18精确路由不等于生产`main` dispatcher；`article-management-S8.3`（tasks.md:2525）明确真正factory→Trace→exactrouter→guard→Session及Case18两GET属于唯一root/联合软件验证。故需新增本卡，不能由既有闭卡承接。
- **设计/测试依据**：`architecture.md:1379`（公共listener下Case18两管理员GET走真实链返回503 `platform.recovery_in_progress`及Trace，Article service/repository/author-reader零调用；404/405/Allow、Origin/CSRF、Session/AccessGeneral及正常开放链兼容）；`design-workspace/identity-auth/design.md:3904`（公共`GET /livez`固定`{"status":"alive"}`；`GET /readyz`仅public_ready 200 `{"status":"ready"}`，其余503 `{"status":"unavailable"}`；只允许GET，错method 405/Allow，未知路径404；liveness不授许可）。设计仍draft、人的精确接纳unknown；本卡及新增映射尚未经非作者PlanningReview；所有未来命令NOT RUN。
- **文件所有权**：
  - `cmd/han-server/main.go`（修改；唯一business dispatcher接线与既有run/newServer路径，不新增第二router/authority）
  - `cmd/han-server/recovery_main_dispatcher_integration_test.go`（新增；`//go:build integration`，真实production dispatcher整链测试`TestRecoveryMainBusinessDispatcher`）
  - `cmd/han-server/recovery_runtime.go`（只读；`recoveryRootAssembly.runtime`已可复用；若实现确需修改以暴露同一runtime handle，停止并回Main批准写集）
  - `internal/platform/recovery_guard.go`（只读，不修改）
  - `cmd/han-server/recovery_runtime_integration_test.go`、`cmd/han-server/main_test.go`（只读，不修改；避免占用S4.16/S8.2写窗；新测试可在同package复用其只读harness符号）
- **文件数**：2 写；4 只读（2 关键依赖 + 2 既有测试文件，不占用其写窗）。若必须改既有`main_test.go`/`recovery_runtime_integration_test.go`或增加第三写文件，先停止并说明冲突与串行条件。
- **依赖**：requires `identity-auth-S4.16`（唯一root assembly与同runtime）、`article-management-S8.3`（guard组件合同）；后续验证关联`identity-auth-S4.17`（已闭test-only cross-layer卡，不作硬requires）；不反向依赖`article-management-S8.2`，不形成环。
- **批次关联**：[B13H](../plans/ir-01-content-community-batches.md#b13h)唯一root批次，消费S4.16 root assembly与S8.3 guard。本卡登记pass未写Plan；后续2026-10-31 B13H规划登记同步已更新Plan（见plan B13H记录），现为75节点/321边。若需新增批次或改变DAG结构，回Main。
- **验收命令**（根目录，待实施/NOT RUN；执行与留存按通用合同）：

```text
go test -tags=integration ./cmd/han-server -list '^TestRecoveryMainBusinessDispatcher$'
go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json
go vet ./cmd/han-server
go vet -tags=integration ./cmd/han-server
gofmt -l cmd/han-server/main.go cmd/han-server/recovery_main_dispatcher_integration_test.go
```

- **必要风险leaf / 最低层**：

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `ExactRouteBeforeGuard` | C/R | 真实production dispatcher先完成exact path/method与404/405/Allow；未知路径/错method不被改成503，guard只在已识别普通业务路由之后、Session/AccessGeneral之前执行；Trace保持最外 |
| `ClosedUnknownTwoAdminGets503` | R/P | closed与unknown各使`GET /api/v1/admin/articles`及`GET /api/v1/admin/articles/{id}`返回503 `platform.recovery_in_progress`并带Trace；Article service/repository/author-reader均零调用；不得用拒启/spy/`http.ServeMux` fixture代证 |
| `LiveReadyProbes` | C/R | 同一runtime handle的`GET /livez`固定200 `{"status":"alive"}`且不授业务权（随后业务GET仍503）；`GET /readyz`仅`public_ready` 200 `{"status":"ready"}`，其余所有状态（closed/unknown/maintenance/verify/无runtime）503 `{"status":"unavailable"}`；无任何recovery配置（`planRecoveryStartup`返回`present:false`、无runtime handle）时`/livez`仍200、`/readyz` 503 `{"status":"unavailable"}`，普通业务路由跳过guard（无recovery runtime即无closed/unknown状态），其余404/405/Allow与开放链行为不变；错method 405/`Allow: GET`、未知路径404；响应不含run/账号/ref/secret |
| `PublicReadyAdmitsSameSession` | R/P | 两层证据同修订必跑：(a) `TestRecoveryRuntimePostgres`（真实 runtime + 真实 PostgreSQL）驱动 Begin→Consume→succeeded→Exit/join（三 worker，holders=0）→activate→`public_ready`，是该 leaf 的必跑伴随证据；(b) 本 leaf 证明同一 production handler、同一 Session 在 closed 下 503（acquire=1 release=0），共享 runtime 进入 `public_ready` 后同一请求 200，Trace 保留、Article 正常调用、acquire/release=2/1。无单测试跨层串联两层，不得声称单测试证明完整链；正常开放链的 Origin/CSRF/Session/AccessGeneral 行为不变，失败负控不被 503 掩盖 |
| `SingleRuntimeHandle` | U/C/R | `main`只把`recoveryAssembly.runtime`同一handle交给guard/probes与既有recovery流程；不建第二runtime/router/pool/authority/fallback；guard lease在handler返回/panic后Release，不开第二public surface |

- **本任务关闭条件**：完成唯一production `main`普通业务dispatcher接线与同handle probe；`TestRecoveryMainBusinessDispatcher`在真实`cmd/han-server` production dispatcher上普通与`-race`通过（`-tags=integration`），并含无recovery配置用例（同文件，普通与`-race`）；closed/unknown两管理员GET 503+Trace且Article service/repository/author-reader零调用；livez/readyz（含无recovery配置时业务路由跳过guard且404/405/Allow与开放链不变）、Origin/CSRF/Session/AccessGeneral及收窄后的两层证据（真实 PG 全链 + `public_ready` 后同 Session 200）均按上表闭合；`go vet`与changed-file `gofmt -l`通过。必须满足所列最低层及通用关闭合同，不是只交接口名或build成功。**本卡是开发卡且含局部测试；独立非作者CodeReview、PlanningReview、Lead同revision核证尚未发生，不得声称已通过。** 不是部署E2E，不证明真实POSIX/CAS/ACL/批准备份/provider；`canonical_recovery_replay_same_result`仍未证明。
- **后续验证关联**：`article-management-S8.2`（tasks.md:1838）Case18整卡接收/关闭仍blocked，待本卡与适用独立验证完成后再由Main评估；本pass不修改S8.2。`identity-auth-S4.17`已闭test-only仅关联；原identity/article S4.4、article S1.5、Case18及部署后S3.4/S1.4/S3.6各自保持，不因本卡关闭而关闭。
- **串行/停止条件**：`cmd/han-server/main.go`是唯一business dispatcher接线点；实施前Main须重核当前281行既有status对应的writer/写窗，未知即停。不得重开或改写`identity-auth-S4.16`/`identity-auth-S4.17`/`article-management-S8.2`/`article-management-S8.3`的checkbox、日期、关闭记录、requires或文件所有权。不得新增第二runtime/router/pool/authority/production fallback；不得改设计契约、S4.16关闭条件、S8.2写集或Plan/DAG而不先回Main。需改`recovery_runtime.go`/既有测试文件/第三写文件、或无法在不改Plan下保持B13H一致性时立即停止返回。禁止Git add/commit/push/切分支/清理/归档/移动/删除；禁止运行go test/vet/gofmt/PG/HTTP/service/restore/SMTP/E2E。
- **Main 规划接收与实施 START（2026-10-31，Asia/Shanghai）**：非作者 Heisenberg / add-reviewer（actualID `01a18fa8-81e0-7563-b876-101ecd1d9f0a`）两轮 Planning Review 结论 PASS，`S419-PLAN-F001`～`F006` 全 CLOSED，遗留 `F007`（`architecture.md:1377/1381` 需 design owner 一次 bounded 澄清，实施前补齐引用）与 `F008`（guard 必须在 exact router 内晚绑定，不得外层包装）两条非阻断 LOW。Main 现场复核两文件字节/SHA/BOM/换行与 `git status` 281 行不变，接收本卡与 B13H Plan 登记同步为当前规划基线。据此实际 spawn Ohm / add-backend-dev（actualID `01a18fbc-0c87-79a3-af86-c4b5fce8b6fc`）承接本卡，写集严格限 `cmd/han-server/main.go` 与本卡新增 `cmd/han-server/recovery_main_dispatcher_integration_test.go`，只读四个既有文件；允许 `go build/vet/gofmt` 与 `-tags=integration` 定向测试，禁止第二 runtime/router/authority、禁 Git 写与清理，缺 DSN 即停。规划 PASS 不等于实现/测试/人的签收；本卡仍 `[-]`，active 由 3 变 4、todo 由 9 变 8，S8.2/Case18 仍 blocked。
- **本卡登记依据/未验证边界**：本卡依据用户standing建议授权仅登记规划建议值；Main依用户2026-10-31 standing建议授权采用建议值a（无recovery配置时`/livez`200、`/readyz`503、普通业务路由跳过guard）；设计仍draft，design owner精确接纳unknown。不代替独立评审、人的业务签收、实施授权、PG/API运行授权、清理或Git写授权。所有验收命令NOT RUN；S4.16/S8.3的组件PASS不等于root/Case18 PASS；S4.16关闭条件中“Trace最外且guard在Session前”的措辞歧义由设计owner后续澄清，本卡不修改其关闭记录。
- **R02 作者完成 / R02 独立复审 FAIL / R03 收窄重出（2026-10-31）**：作者 R02 证据包见 [s419-author-r02](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s419-author-r02/report.md)；独立复审 R02（Agent Turing，actualID `01a190a1-e186-7b11-ab4d-040a4cdfd2fe`）结论 [FAIL](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r02/report.md)：`S419-CR-F002` CLOSED、N1 CLOSED、`S419-CR-F001` OPEN（编号 `S419-CR-F001-R02`，MEDIUM）。Lead 按复审路径 2 作 [S4.19 正式收窄决定 R01](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s419-lead-narrowing-decision-r01.md)，只改验收措辞，不改生产行为与写集；作者 R03 按决定把子测试改名为 `PublicReadyAdmitsSameSession` 并注明分层证据，[R03 证据](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s419-author-r03/report.md) 定向普通/race 各 17 pass、真实 PG 7 pass、单元 `ok 40.415s`。收窄后的 leaf 与关闭条件已回写；本卡仍 `[-]`，待独立复审 R03 与 Lead Gate R03。
- **Lead Gate R03 PASS / 本卡关闭（2026-10-31）**：独立复审 R03（Turing / add-reviewer，actualID `01a190a1-e186-7b11-ab4d-040a4cdfd2fe`；R02/R03 报告中的 unknown 已由 Lead Gate 更正）结论 [PASS](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s419-independent-review-r03/report.md)，无新增 finding；`S419-CR-F001-R02` CLOSED，`S419-CR-F002` 与 N1 保持 CLOSED。Lead Gate R03 现场复算两文件 SHA、HEAD 与 staged，复跑 gofmt/build/vet -tags=integration/定向 race 17 pass/真实 PG 7 pass 均 exit 0，记录见 [s419-lead-gate-r03](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s419-lead-gate-r03/report.md)。N5 已由 R02/R03 独立复跑与本次 Gate 复跑替代关闭。本卡按收窄口径关闭；B13H 批 Gate 仍须同修订同跑两层，缺一不通过；`TestRecoveryRuntimeProductionChain`（53519）保持 NOT RUN。本关闭为 AI 代理核证，非真人签字，不证明真实部署/E2E/POSIX/ACL/provider。全局计数 63/4/8 → 64 completed / 3 active / 8 todo。
- **2026-10-31 B13H 批次 Gate FAIL → 本卡重开返工（Lead；用户 standing 建议授权；`[x]`→`[-]`）**：非作者独立批次 Gate（Bacon / add-reviewer，actualID `01a19216-b34f-7162-ace6-ac6fd1b992d4`）结论 **FAIL**，证据 `.spec/wip/recovery-runtime/evidence-b13h-r01/b13h-batch-gate-r01/`：`report.md` `027434B859DEC877E798E6AA97EE0884A43307CBF08ECA7FD1294D6D800C6498` / 24800 B、`verdict.md` `3DF4E4F920691ECFAFD044903B8CDB8D9A39C0EA96CBA965C7B65639798D063B` / 7956 B、`commands.log` `209998D1CB8C1A480715D4B3FFAD339A12D3B902C7477498849A69134D89999E` / 4238 B。阻断 `B13H-GATE-F001`（HIGH）：`cmd/han-server/recovery_main_dispatcher_integration_test.go`（`B89DA3791616353ACBAC602B1F22AC98298CFE888FBD55E1D3A4686F7BA67A97` / 16500 B）仅 5 子测试，`:143` 以 `setState("public_ready")` 代替真实 runtime 进入，全文无 `recovery.NewRuntime`/`runtime.Enter`/`PublicReadyAdmitsSameSession`，违反 `:2859` 层 (b)、`:2868` 两层缺一不通过、`:2934` 禁 `setState`。**返工合同**：仅在本卡写集内（`cmd/han-server/main.go` + 本测试文件；4 只读）重建 `TestRecoveryMainBusinessDispatcher/PublicReadyAdmitsSameSession`：使用真实 `recovery.NewRuntime` 与真实状态转换进入 `public_ready`（优先 `runtime.Enter` 完整恢复；若写集内无法重建恢复世界，可用同一真实 runtime 的 `RunNormal` 真实进入 `public_ready`，但须在测试注释与返工报告中说明层 (a) 承担完整恢复链；不得以 `setState`、测试 transport/fake world 或单测适配器冒充）；同一 production dispatcher/handle、同一 Session：closed 下 503 且 acquire=1/release=0，`public_ready` 后同一请求 200、Trace 保留、Article 正常调用、acquire/release=2/1，并含失败负控；优先零 production 改动，若必须改 `main.go` 或第五条文件即 STOP 回 Lead。**N01（LOW）**：删除未调用的 `publishPublicReady`（`progress.md:44` 为历史证据，不覆盖，仅在本记录更正披露）。计数 65/3/8 → 64 completed / 4 active / 8 todo。本记录为 AI Lead 按用户 2026-10-31 standing 建议授权作出的流程处置，不代替独立复审、批次 Gate 重跑、人的业务接收或 Git 写授权。

<a id="task-identity-auth-s4-20"></a>

- [x] identity-auth-S4.20 在唯一production `cmd/han-server` root上复用S4.19已接线的同一recovery runtime/handle、同一RecoveryRunStore与article evidence Store，打通“全局恢复权威→请求隔离→resume 恢复运行完整完成→成功开流”的真实生产链；仅 resume 恢复运行完整完成后进入`public_ready`并开流，失败/unknown继续隔离。本卡范围为 resume，不含 enter/install 与 recovery secret candidate 关闭点。 完成@2026-10-31

- **目标**：在唯一production `cmd/han-server` root上，用同一recovery runtime/handle完成RecoveryRunStore/Coordinator的 resume 恢复运行完整完成（Resume→replay→journal consume→succeeded复核→activate/开流）、article/identity恢复与Journal/HMAC/keyring的真实或模拟边界、二次`ReadStatus`、`MarkConsumed`，成功进入`public_ready`后开流；closed/unknown失败继续阻断并保持隔离。不得以直接`setState("public_ready")`冒充 resume 恢复运行完整完成，不得建第二runtime/router/pool/authority/fallback。范围仅 resume：`cmd/han-server/recovery_runtime.go:465`（enter/install）与`:523`（recovery secret candidate）保持 closed 占位，不在本卡实施；“enter/install/secret 的后续承接任务未登记（unknown），不得由本卡声称已覆盖完整安装/恢复入口”。

- **类型/Story与范围**：开发（同卡局部真实production chain测试，范围限定为 resume 恢复运行完整完成）；identity Story S04全局恢复生产消费，关联article Story4/Story8与Case18接收，但本卡不关闭、不重开`article-management-S8.2`。本卡承接“全局恢复权威→请求隔离→resume 恢复运行完整完成→开流”的主线实施；消费已闭`identity-auth-S4.16`（唯一root assembly/同Store/同Runtime）、`identity-auth-S4.19`（production dispatcher guard/probes/同handle）、`article-management-S8.3`（guard组件）与`identity-auth-S4.17`（cross-layer test-only证据）、`article-management-S4.4`的证据；`article-management-S1.5`为条件性未来引用，其 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS 与 Lead Gate PASS 已满足（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已于 2026-10-31 关闭（脱敏／索引／ADR-0029 r04）），S1.5 ACL 子集可作为可消费输入（整卡仍 `[-]`，不构成整卡 PASS）。不重开S4.16/S4.17/S4.19/S8.2/S8.3，不改S8.2九文件写集。明确不证明：enter/install/recovery secret candidate 恢复入口（后续承接任务未登记，unknown）、真实部署/E2E/POSIX/ACL/真实provider/真实备份；真实环境不可用时按ADR-0028用模拟环境并标注“未真实验证”。

- **R对账（四标签）**：
  - **已建成**：S4.16已在唯一root assembly中构造同一`RecoveryRunStore`与`runtime`（`cmd/han-server/recovery_runtime.go:457/469/596`）；S4.19已把同一handle接入production dispatcher并保留closed/unknown 503+Trace（`cmd/han-server/main.go:2418/3648`）；S8.3已提供guard组件；`internal/recovery/coordinator.go:133`的`Consume`已含identity ready→article deletion ready→journal consume→succeeded复核→`VerifyAgainst`完整链，`markAndRead`（`:274`）调用`MarkConsumed`并二次读取快照；`internal/recovery/runtime.go:1538/1895/1917`的`Resume`/`replay`/`ExitRecovery`（`activate`定义在`:917`、调用在`:1943`）已具备consumer调用、exit/lifecycle/开流路径；`internal/article/deletion_replay.go:26/40/244`与`internal/article/replay_publication.go:83`已提供`ReplayAndVerify`/`ReadStatus`/publication ports；`internal/identity/recovery_execution.go:41/65`已提供identity execution与`ArticleAccountPort`。S4.17为test-only跨层证据，article S4.4局部证据可消费；article S1.5为条件性未来引用，见“范围/依赖”。
  - **待做**：替换`cmd/han-server/recovery_runtime.go:498/531/538/539/545/558`的closed占位，注入同一`ArticleSnapshots`/article evidence Store/publication ports、真实或模拟Journal/HMAC/keyring边界、真实或模拟Coordinator consumer与identity `ArticleAccountPort`；修正`cmd/han-server/main.go:3290` `newArticleReplayProvider`自建第二个`article.NewPostgresReplayEvidenceStore`的问题；补齐二次`ReadStatus`、`MarkConsumed`、`public_ready`后开流的 resume 完整链专项测试；证明`canonical_recovery_replay_same_result`按下方“可执行 oracle”条款（持久字段相等+零额外写入，不得只查日志字符串）；真实PG+production dispatcher取得closed/unknown拒绝与成功开流证据。`recovery_runtime.go:465`（enter/install）与`:523`（recovery secret candidate）不替换、保持closed占位，不在本卡范围。
  - **需改**：仅下节列出的4个文件。
  - **废弃**：无；不重开任何已关闭卡，不废弃既有组件。

- **设计/测试依据**：`architecture.md:1375–1381`（§9.1.1全局恢复生产消费与开放：closed→正向证明→开流；公共listener exact router返回503 `platform.recovery_in_progress`，Case18两GET真实链与下游零调用）；`design-workspace/identity-auth/design.md:3568`（r50全局恢复与正常重启完整接线）、`:4430`（§9.1 S16共享恢复合同分配）；`design-workspace/article-management/design.md:2430`（§3.4.7 DeletionReplayEvidence与恢复开放门禁）、`:3051`（§8.6 Case18接收边界）；`identity-auth/test-plan.md:759`（§4.7.1 r24全局恢复运行时风险/场景/验收）、`article-management/test-plan.md:732`（§4.8.2 r21锚定Journal/发布来源/全局恢复接收）；ADR-0027（全局恢复runtime框架编排、唯一Store、正向准入）与ADR-0028（项目级验证授权）。设计仍draft、人的精确接纳unknown；本卡及新增映射已由非作者Planning Review R01 判 `CHANGES REQUIRED`（4 HIGH/2 MEDIUM/2 LOW），本修订按 Lead 定案 F001–F008 处置，修订后的复审仍待进行；所有未来命令NOT RUN（R01 期间亦全部 NOT RUN）。

- **文件所有权**：
  - `cmd/han-server/recovery_runtime.go`（修改；在唯一root assembly内替换closed占位并注入同一Store/runtime/consumer/identity port，不建第二Store/Runtime/Pool）
  - `cmd/han-server/main.go`（修改；唯一production root接线，复用同一recovery handle/article evidence Store，修正`newArticleReplayProvider`第二Store；不新增第二router/authority）
  - `cmd/han-server/recovery_runtime_integration_test.go`（修改，仅测试承载迁移；既有`:165`顶层`TestRecoveryRuntimeProductionChain`保持原名、原断言与原覆盖不变，不改其符号/语义（不重开S4.16）；仅允许不改该测试行为的最小共享 helper 承载迁移。若Main认为该迁移构成重开S4.16，立即停止返回）
  - `cmd/han-server/recovery_production_chain_integration_test.go`（新增；`//go:build integration`，完整 resume 生产恢复链专项测试，顶层函数名`TestRecoveryProductionChainComplete`）
  - 不默认修改`internal/identity/*`或`internal/article/*`；已核`internal/identity/article_port.go:17`的`ArticleAccountPort`与`internal/article/command_service.go:342`的`OrphanArticlesCanonical`可在cmd层用`ArticleAccountPortFunc`适配。若实施确需identity/article包文件，停止并回Main扩写集。

- **文件数**：4（3改+1新增；其中`recovery_runtime_integration_test.go`仅测试承载迁移）。若需第五文件或改动上述以外的包，先停止并说明冲突与串行条件。

- **依赖**：requires `identity-auth-S4.16`（唯一root assembly/同Store/同Runtime）、`identity-auth-S4.19`（production dispatcher/guard/同handle）、`article-management-S8.3`（guard组件）；仅消费证据、不作硬requires：`identity-auth-S4.17`、`article-management-S4.4`；`article-management-S1.5`为条件性未来引用，其 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS 与 Lead Gate PASS 已满足（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已于 2026-10-31 关闭（脱敏／索引／ADR-0029 r04）），S1.5 ACL 子集可消费（整卡仍 `[-]`，不构成整卡 PASS；仍不作硬requires）；不requires`article-management-S8.2`，不形成环、不反向阻塞已关闭卡。

- **批次关联**：[B13H](../plans/ir-01-content-community-batches.md#b13h)唯一root批次（共享`cmd/han-server/main.go`、`recovery_runtime.go`与既有recovery integration test，属同一唯一root串行写窗）；不新建批次。本卡登记后全图76节点/324条requires边、runtime切片17卡/70边、B13H 17个唯一文件/21个声明槽。

- **验收命令**（根目录，待实施/NOT RUN；执行与留存按通用合同）：

```text
go build ./...
go test -tags=integration ./cmd/han-server -list '^TestRecoveryProductionChainComplete$'
go test -tags=integration ./cmd/han-server -run '^TestRecoveryProductionChainComplete$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryProductionChainComplete$' -count=1 -json
go test -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json
go test -race -tags=integration ./test/integration/recovery -run '^TestRecoveryRuntimePostgres$' -count=1 -json
go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json
go vet ./cmd/han-server
go vet -tags=integration ./cmd/han-server
gofmt -l cmd/han-server/main.go cmd/han-server/recovery_runtime.go cmd/han-server/recovery_runtime_integration_test.go cmd/han-server/recovery_production_chain_integration_test.go
```

- **canonical replay 可执行 oracle（F003，必须按此执行）**：
  - **比较的持久结果字段**：同一 run 的 `RecoveryRun`（`Outcome`、`JournalStage`、`JournalConsumedAt`、`ArticleEvidenceRef/Digest/Version/CompletedAt`、`ArticleReplayVerifiedAt`、`ArticlePublicationRef/Digest`、`CompletedAt`）；`VerifiedJournalSnapshot`（`JournalID`、`Namespace`、`Epoch`、`Head`、`Revision`、`Entries` 数量与逐条 `Sequence`/`RecordHMAC`、`Prefixes`）；`AdmissionSnapshot` 经 `CanonicalAdmissionSnapshot` 规范化后的字节与 `Revision`；article evidence 的 ref/digest 与 snapshot ref/digest；consumed 标记（`JournalStage == StageConsumed` 且 `JournalConsumedAt` 非空）。
  - **第二次 replay 触发方式**：同一 run、同一 live permit、同一 `PermitBinding`（由 `ReadPermitBinding` 读取）；不得新建 run、permit、binding 或第二 runtime。
  - **零额外写入观测**：第二次 replay 前后分别读取上述持久快照并逐字段比较；同时以可审计证据证明无新增写入——`recovery_runs` 行数不变、`RecoveryRunStore.Transition` 调用次数不变（测试内只读包装计数，不改变生产 wiring）、`AdmissionStore.ReadAdmissionRevision` 返回的 `Revision` 不变、`VerifiedJournalSnapshot.Revision` 与 `Entries` 数量不变、journal `Sequence` 最大值不变；不得仅以日志字符串、fake 返回值或调用次数代替持久结果比较。
  - **边界**：若 oracle 必须写 `internal/identity/*` 或 `internal/article/*`，立即 STOP 回 Main 扩写集；不得以 fake/日志字符串替代。

- **必要风险leaf / 最低层**：

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `ResumeCoordinatorRecoveryCompleteAndOpen` | R/P | 真实PG+production dispatcher经同一Coordinator完成 resume 恢复运行（identity ready→article deletion ready→journal consume→succeeded复核→activate），成功进入`public_ready`并开流；不得直接`setState`；不声称覆盖 enter/install/secret candidate |
| `SecondReadStatusAndMarkConsumed` | R/P | resume 完整链真实调用`MarkConsumed`与二次`ReadStatus`，exact append/快照核验通过；缺失/漂移失败关闭 |
| `CanonicalReplaySameResult` | P | 按本卡“canonical replay 可执行 oracle”：同 run/同 live permit/同 binding 的第二次 replay 持久字段逐一相等，且前后行数/Revision/Entries/Sequence 证明零额外写入；不得只检查`unproven_port`日志字符串 |
| `ClosedUnknownKeepIsolation` | R/P | closed/unknown返回503 `platform.recovery_in_progress`+Trace，Article service/repository/author-reader零调用；成功后同一请求200 |
| `SingleRuntimeStorePool` | U/C/R | 同一RecoveryRunStore/article evidence Store/runtime/pool，无第二实例 |
| `SimulatedJournalKeyringBoundary` | P | Journal/HMAC/keyring真实或模拟边界可复现；模拟时标注“未真实验证” |
| `NoSecondRuntimeRouterPoolAuthorityFallback` | U/C | 不建第二runtime/router/pool/authority/fallback，不新增第二public surface |
| `MainWriteWindowFrozenAndS419Reverified` | U/C/R | START 前满足 S1.5 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS + S1.5 Lead Gate PASS（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已于 2026-10-31 关闭（脱敏／索引／ADR-0029 r04））+ S1.5 writer 明确 STOP/`main.go` 写窗冻结（该项仍未满足），并冻结当前 `main.go` SHA256/字节/mtime；关闭前在当前 SHA 上重跑 S4.19 dispatcher/probe 两层并记录；期间任何 `main.go` 变化即 STOP 重核 |

- **本任务关闭条件**：`go build ./...` 通过；`TestRecoveryProductionChainComplete`在真实PG+production dispatcher上普通与`-race`通过（`-tags=integration`）；`canonical_recovery_replay_same_result`按本卡可执行 oracle 以持久字段相等/零额外写入证明；closed/unknown 503+Trace且下游零调用、成功进入`public_ready`后开流均按上表闭合；同一Store/runtime/pool且无第二runtime/router/pool/authority/fallback；在当前冻结的`main.go` SHA256 上重跑 S4.19 dispatcher/probe 两层（`TestRecoveryMainBusinessDispatcher` 等）并记录通过；`go vet`（普通与`-tags=integration`）与changed-file `gofmt -l`通过；未跟踪文件以 SHA256+字节数+全文快照核验，不得以“无 diff”当作“无改动”；独立非作者CodeReview、Lead同revision核status/diff/所有权；未真实验证项已标注。不得用直接`setState`或build成功代替 resume 恢复运行完整完成、测试、评审或E2E。

- **后续验证关联**：`article-management-S8.2`为后续消费者，保持`[-]`，待本卡与应用独立验证完成后再由Main评估，不自动关闭、不改其九文件写集。不证明真实部署/E2E/POSIX/ACL/真实provider/真实备份；真实环境不可用时按ADR-0028用模拟环境并标注“未真实验证”。S4.17/article S4.4及部署后S3.4/S1.4/S3.6各自保持；article S1.5 为条件性未来引用；其 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS 与 Lead Gate PASS 已满足（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已于 2026-10-31 关闭（脱敏／索引／ADR-0029 r04）），S1.5 ACL 子集可评估消费（整卡仍 `[-]`，不构成整卡 PASS）。

- **串行/停止条件**：**START 硬门禁（F002）**：S4.20 START 前必须同时满足「`article-management-S1.5` ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS」+「S1.5 Lead Gate PASS」（二者见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已于 2026-10-31 关闭（脱敏／索引／ADR-0029 r04））+「S1.5 writer 明确 STOP 且 `cmd/han-server/main.go` 写窗冻结」（该项仍未满足）；任一不满足即 STOP。实施前记录 `main.go` 的 SHA256/字节数/mtime，实施后再次记录；期间 `main.go` 任何变化即 STOP 重核（F008）。`cmd/han-server/main.go`与`cmd/han-server/recovery_runtime.go`是唯一root共享文件，实施前Main须重核实际writer/写窗，未知即停；`recovery_runtime_integration_test.go`仅测试承载迁移且不得改变既有`:165` `TestRecoveryRuntimeProductionChain` 的符号/断言/语义，若被视为重开S4.16即停。不得重开或改写`identity-auth-S4.16`/`identity-auth-S4.17`/`identity-auth-S4.19`/`article-management-S8.2`/`article-management-S8.3`的checkbox、日期、关闭记录、requires或文件所有权；不得改S8.2九文件写集。需改设计契约、identity/article包文件、第五写文件或无法保持B13H一致时立即停止返回。禁止Git add/commit/push/checkout/stash、清理/删除/移动/归档；禁止运行go test/vet/gofmt/PG/HTTP/service/restore/E2E。本卡初始`[ ]`，无完成日期；PlanningReview/Main接收/实施START/NOT RUN证据均pending。

- **本卡登记依据/未验证边界**：依据用户2026-10-31明确授权“S8.2卡点采用建议正解”仅登记规划建议值；消费既有已关闭组件证据，不把组件PASS相加成完整链或Case18 PASS。设计仍draft、人的精确接纳unknown；所有验收命令NOT RUN；`TestRecoveryProductionChainComplete`当前NOT RUN、`canonical_recovery_replay_same_result`未证明（可执行 oracle 已写入，执行证据 pending）；enter/install/recovery secret candidate 后续承接任务未登记（unknown）；真实provider/ACL/备份/部署/E2E/POSIX unknown。不代替独立评审、人的业务签收、实施授权、PG/API运行授权、清理或Git写授权。
- **2026-10-31 Lead 承接说明（S4.19 dispatcher 测试隔离）**：工作树中 `cmd/han-server/recovery_main_dispatcher_integration_test.go` 原有 `OpenAfterCompleteSoftwareRecoverySameSession` 子测试引用缺失的 `newS419RecoveryWorld`，导致 `go vet -tags=integration ./cmd/han-server` 与真实 PG 集成都无法构建（独立评审 `S15-ACL-F003`）。Lead 已将该子测试与函数移出并完整留存原文：`.spec/wip/recovery-runtime/evidence-b13h-r01/s419-world-hold-r01/`（原文 SHA256 `36DA30ABD77EAC4EE8E6098F588677B2A20BB4A854B6FEA8F161168D2D634391`，README 说明处置理由）。该场景（完整恢复→进入 `public_ready`→同会话开流）正是本卡关闭条件，须由本卡 `recovery_production_chain_integration_test.go` 以真实 production chain 重建，不得用未验证的 fake world 通过。本卡 `[ ]`、无完成日期、全部 NOT RUN 状态不变；不据此声称 S4.19 或完整恢复 PASS。
- **2026-10-31 START 门禁现状记录（不改本卡 checkbox/日期/requires）**：逐字复核 §串行/停止条件 的 START 硬门禁：①「S1.5 ACL 当前 revision（R03）非作者独立复审 PASS WITH MINOR」满足（`7A236731…`，三条 LOW 已按 2026-10-31 Lead 记录关闭，见 S1.5 段）；②「S1.5 Lead Gate PASS」满足（[lead-gate.log](../../../../.spec/wip/article-management/evidence-s1.5/acl-r03/lead-gate.log) `D21DB1F19903FA2D25685A4ADE9C58BE1FF89C4D47E9023FB7B0C269AE608155`）；③「S1.5 writer 明确 STOP 且 `cmd/han-server/main.go` 写窗冻结」——事实层：S1.5 实施/返工 writer（Herschel，actualID `01a190fe-fb35-7921-9ca0-84fae3cf5764`）已 completed，`cmd/han-server/main.go` 仍为 `1D47CC94…`（与 R03 冻结一致），本轮未见活动写者；是否据此宣告“写窗冻结”由 START 时 Main 与用户确认，Lead 不代判。本卡状态保持 `[ ]`、无完成日期、全部验收命令 NOT RUN；PlanningReview/Main 接收/实施 START/证据均 pending，本轮未执行任何 Go/PG/HTTP/service/restore/E2E 命令，未改 `main.go` 或 `recovery_runtime.go`。
- **2026-10-31 S4.20 START 门禁事实更正与指针统一（Lead 记录；不改本卡 checkbox/日期/requires/写集）**：应 R02 复审 `report-final.md` 的 F005–F008 收口要求，仅作事实层更正，不重开任何结论。**F005（并发写者归因）**：`tasks.md` 中“另一 Codex 会话（PID 35652）”的归因错误——PID 35652 是本会话宿主（VSCode 扩展 `app-server`，即本轮 shell 的父进程），不是并发写者；真实并发写者由另一宿主进程 `xDesk.exe`（PID 27012，窗口 `xLink`，2026-10-31 17:48:22 启动）承载，其写入时点为 `:1235`（17:52:38）、第二批 5 份证据脱敏（17:58:16）、Plan 引用同步（17:58:44）与 `:1239`（18:03:20），此后至本记录写入前静默 ≥6 分钟；`:1237` 的“PID 35652 仍存活 / 最后写入 17:46:48”表述作废，以本记录为准。**F006（脱敏口径）**：`:1234`、`:1237` 的“三份报告待用户确认”与 `:1235` 的“5 处历史证据待确认”均为各自写作时点事实；现行状态为 8 份文件全部替换为 `<redacted>`（`:1239` 已登记 5 份新哈希），全仓复扫 `postgres:postgres@` 命中 0，无真明文 DSN 残留。**F007（v1 指针）**：S4.20 START 记录第①重所引 `7A236731…`（24911 B）为已被同路径覆盖的历史指针，现行有效为 R03 v2 `78D8AA2318ED237B9141EECFCEC5372F355EB997387371BA1C0490730E47A238` / 23870 B / 154 行，结论同为 PASS WITH MINOR，仅覆盖 ACL 门禁子集。**F008（framework.md 旧哈希）**：`:1234` 所引 `04E56C99…` 为 r04 errata 写入前快照；现行 `5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A` / 126590 B / 911 行（17:48:45），`:911` 已改为引用 v2。**冻结基线（本记录写入后复算）**：Plan `AF675793D97E5E9CDDB45F9C84C36EA7F0DAC40C249B1223D5EDC66DB65D3E31` / 489030 B；`cmd/han-server/main.go` `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` / 171448 B / 16:36:42；`decisions/README.md` `35A5623E7E3CFCC140DA0DE9C8B218210C0E34A850A26B5CDD450B81DA40CC63` / 18948 B。**边界**：本记录不改 `:1234`/`:1235`/`:1237`/`:1239` 原文（错误归因以本记录更正，不删除他人记录）；不改任何 checkbox、完成日期、requires、文件所有权或写集；不宣告 S1.5 writer STOP 或 `main.go` 写窗冻结（START 第③重由 Main 在 START 时依据“无活动写者 + `main.go` 哈希未变”宣告并经用户确认）；未运行任何 Go/PG/HTTP/service/restore/E2E，未做 Git 写或清理；S4.20 仍 `[ ]`、无完成日期、全部 NOT RUN，S1.5 整卡仍 `[-]`。
- **2026-10-31 S4.20 Planning Review R02 正式结论 PASS WITH MINOR + Lead 现场复核（PlanningReview 关闭；START ①② 成立、③ 未宣告；整卡仍 `[ ]`）**：非作者独立复审（Kierkegaard / add-reviewer）已签发正式结论，报告 `.spec/wip/recovery-runtime/evidence-b13h-r01/s420-planning-review-r02/report-verdict.md` SHA256 `367FCC49198F55EBCCB890B6495FF5F585FC70D694C19EFC965ACDE9D3E1B4D5` / 20791 B / 18:13:27；同目录 `report.md`（STOP，`CDD8AADCF8189B99C6AE9AABF2810B7A4773FF78CE84E1DC1C193B6730766964` / 15072 B）与 `report-final.md`（STOP，`8D6EC950BC241EF4F139C47CBB401936C813B18FEECB17C2FB569F38C5CA4BCF` / 21852 B）按历史保留、未被覆盖。Lead 已实读报告而非转述摘要：R01 `F001`–`F008` 8/8 CLOSED；R02 `F001`–`F003` CLOSED、`F004` 维持非缺陷信息项、`F005`–`F009` CLOSED；新增 `F010`、`F011` 均为 LOW。**Lead 复核**：评审窗口 T0 18:09:10 / T2 18:12:53（223 s）/ T3 写入后三次复算，6 个冻结对象与 2 份受保护报告哈希全部一致，期间 drift=0；`tasks.md` `DF0FB7805C2840CC596C31A88235526080C599CCDAD40A26A0859565A03836B9` / 733030 B / 2939 行、Plan `AF675793D97E5E9CDDB45F9C84C36EA7F0DAC40C249B1223D5EDC66DB65D3E31` / 489030 B、`framework.md` `5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A` / 126590 B、R03 v2 `78D8AA2318ED237B9141EECFCEC5372F355EB997387371BA1C0490730E47A238` / 23870 B、`cmd/han-server/main.go` `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` / 171448 B / 16:36:42、`decisions/README.md` `35A5623E7E3CFCC140DA0DE9C8B218210C0E34A850A26B5CDD450B81DA40CC63` / 18948 B 与冻结值逐项相符；复审为非作者产出（任务卡作者侧为 Task-Planner）。**MINOR 收口**：`F010` 精确静默区间为 18:03:20.4742 → 18:08:55.9665（5m35s），以本记录取代 `:2939` 的“≥6 分钟”表述；`F011` 全仓 `postgres:postgres@` 字面命中 2 处均为描述该字符串的文档自指（`tasks.md:1239`、`:2939`），无任何凭据，真实 DSN 明文 0。**START 门禁**：①成立（R03 v2 非作者 PASS WITH MINOR，仅覆盖 ACL 门禁子集，S1.5 整卡仍 `[-]`）；②成立（`lead-gate.log` `D21DB1F19903FA2D25685A4ADE9C58BE1FF89C4D47E9023FB7B0C269AE608155` / 298 B）；③仍未满足/未宣告——按 `:2934`/`:2928` 由 Main 在 START 时确认并记录 `main.go` SHA256/字节/mtime 基线，实施期间任何变化即 STOP 重核。**并发写者**：另一宿主进程 `xDesk.exe`（PID 27012，窗口 `xLink`）仍存活，自 18:03:20 后未再写入项目文件；在其关闭或用户明确接受前，Lead 不宣告第③重、不派 S4.20 实施。**边界**：本记录不改卡 checkbox/完成日期/requires/写集，不改 `:1226` S1.5 整卡 `[-]`；本结论仅为“该卡可作为实施输入”，不证明代码/测试/build/PG/E2E/部署通过，不代替人的业务签收、实施授权、PG/API 运行授权、清理或 Git 写授权；未运行任何 Go/PG/HTTP/service/restore/E2E，未做 Git 写或清理。
- **2026-10-31 S4.20 START 第③重宣告与实施放行（Lead；依据用户当日授权“对待确认或决策项采用建议值并继续本项目”；不改本卡 checkbox/日期/requires/写集）**：**第③重宣告成立**：S1.5 实施/返工 writer（Herschel，actualID `01a190fe-fb35-7921-9ca0-84fae3cf5764`）已 completed，工作树无其他活动业务写者；唯一外部并发风险为另一宿主 `xDesk.exe`（PID 27012，窗口 `xLink`，17:48:22 启动），其最后写入为 18:03:20.4742，截至本记录 18:16:39 静默约 13 分 19 秒，判定为“静默但仍存活”，按并发写者风险处理而非认定已 STOP。**冻结基线（本记录写入前后实测）**：`cmd/han-server/main.go` `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` / 171448 B / 2026-10-31 16:36:42.7242；`cmd/han-server/recovery_runtime.go` `47CCE68A33DB57AD2414AE9D26FB6FD79459D3D96C415F02F61FAB75DA7216B2` / 33806 B；`tasks.md` 追加前 `D8648BAAFB3DCCC364082554CA37128EE41A5E0D051AEC4B76E3E948D3D786DB` / 736099 B / 2940 行 / 18:14:14.8563。**①②依据**：R03 v2 `78D8AA2318ED237B9141EECFCEC5372F355EB997387371BA1C0490730E47A238` / 23870 B（非作者 PASS WITH MINOR，仅覆盖 ACL 门禁子集，S1.5 整卡仍 `[-]`）+ `lead-gate.log` `D21DB1F19903FA2D25685A4ADE9C58BE1FF89C4D47E9023FB7B0C269AE608155` / 298 B。**F008 守卫**：S4.20 实施期间若 `main.go` 出现任何非本批次写动的变化，或其 SHA256/字节数/mtime 偏离上述基线，立即 STOP 并重核写者与门禁；实施完成后的 `main.go` 变化仅限本卡声明的唯一 root 接线与第二 Store 修正范围，并由 Lead 复算新基线。**实施放行边界**：允许 `add-backend-dev` 按卡面写集（`cmd/han-server/main.go`、`cmd/han-server/recovery_runtime.go`、`cmd/han-server/recovery_runtime_integration_test.go` 仅测试承载迁移、新增 `cmd/han-server/recovery_production_chain_integration_test.go`）实施，并仅执行 `:2895`–`:2909` 列出的验收命令（`go build ./...`、指定 `-tags=integration` 用例普通与 `-race`、`go vet` 两层、changed-file `gofmt -l`）作为本批次自测；不执行卡面未列出的 deploy/service/restore/全量 E2E 命令；真实 PG 不可用时按 ADR-0028 标注“未真实验证”并回报，不得伪造或跳过证据。**不改**：本卡 `[ ]`、无完成日期、requires、文件所有权、B13H 批次关系；S1.5 整卡仍 `[-]`；S4.16/S4.17/S4.19/S8.2/S8.3 的 checkbox/日期/写集不动。**本记录不代替**独立非作者 Code Review、Lead 同 revision 复核、人的业务签收或 Git 写授权；截至本记录未运行任何 Go/PG/HTTP/service/restore/E2E，`TestRecoveryProductionChainComplete` 仍 NOT RUN。

- **2026-10-31 S4.20 R02 独立复审 PASS WITH MINOR + Lead 现场复核（整卡仍 `[ ]`，不宣告正式 PASS）**：非作者独立复审（Rawls / add-reviewer；复审报告自报 `actualID` unavailable，Lead 调度侧 Agent ID 为 `01a191f0-c0e8-7442-8eb1-0ca87da0b034`）结论 **PASS WITH MINOR**，无 HIGH、无阻断。R01 `S420-CR-F001`（HIGH）→ **CLOSED**：`CanonicalReplaySameResult` 的直接与包装 replay 均使用 `lease.Context()`；普通 `ctx` 负向对照得到 `ErrAdmissionBinding` 且 `forwardedCount()==0`，live permit 正向对照 `forwarded==1` 且字节与直接调用相等；持久字段相等与零额外写入断言未被削弱，provider 自身不校验 permit 的边界已如实披露。`F002`–`F007` 全部 **CLOSED**；`F008` 接受 Lead 裁定 **NOT A DEFECT**（4 个文件与 `tasks.md:1228`/`:1560` 登记哈希逐项相符）。残余 `R02-F001`–`R02-F005` 均为 LOW（control server goroutine 未 join、等价手工编排未走 `assembly.start`、无第二 router/fallback 仅指针层证明、closed/unknown 非 production dispatcher E2E、provider/ports 测试侧装配），均不阻断且已披露。**Lead 现场复核**：被评文件 `cmd/han-server/recovery_production_chain_integration_test.go` `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` / 167339 B / 3868 行；其余三写集文件与基线逐项一致；`gofmt -l`、`go build ./...`、`go vet` 两层均 EXIT=0；5 叶普通与 `-race` 各 5/5 PASS；完整 8 叶普通与 `-race`、`TestRecoveryRuntimePostgres` 普通与 `-race`、`TestRecoveryMainBusinessDispatcher` 普通与 `-race` 均 EXIT=0，JSON 证据 fail=0/skip=0。复审报告 `s420-code-review-r01/report-r02.md` `A27D3C0F199BC34022F2B84C902045E471E7A51CB96B5D5F142F9C9A7742D567` / 22494 B / 210 行、`verdict-r02.md` `6AE8A942A8C28B51DA69EDDB26975E2A7C7745507B6EDFD5E9DD2CC4F1EA6064` / 3615 B / 44 行、`commands-r02.log` `C52BDF1ADBA5DDD288AA505712191DB52B0B5EB0ABB8FF6719D21379043F3CD0` / 64977 B；R01 三件未被覆盖。**边界**：本记录不改 S4.20 checkbox/完成日期/requires/写集，不宣告整卡正式 PASS 或人的业务签收；不证明真实 keyring/HSM、approved backup inventory、POSIX journal/fsync/flock/CAS、部署/E2E、production dispatcher 全路由或进程级全局唯一 runtime；S1.5 整卡仍 `[-]`，S4.16/S4.17/S4.19/S8.2/S8.3 的 checkbox/日期/写集未动；未做 Git 写或清理。
- **2026-10-31 identity-auth-S4.20 Lead Gate PASS / 任务关闭（`[x]` 完成@2026-10-31；Lead 直接核证，无新增 Role 产出）**：按 `.codex/guidance/task-execution-workflow.md` §10.3 六条逐项核对。**①本 task 成果与验收完成**：卡面关闭条件全部有当前 revision 证据——`go build ./...` EXIT=0；`TestRecoveryProductionChainComplete` 在真实 PostgreSQL + production dispatcher 上普通与 `-race` 通过（`d7-chain-complete-r02.json` SHA256 `225F78BDF3BC67A453B83018E1FC1D2A8E1A0548EC44703C42D7AAC933343761` / 27057 B；`d7-chain-complete-race-r02.json` `72002FB661EF69588684F2E7B4CB239F4D8F96672C3F927F8AB1C248E43BCC56` / 27055 B）；五叶普通与 `-race` 各 5/5 PASS（`d7-five-leaves-r02.log` `2236B5068B5F538F29F2DFD0F7D92B1E165F1656604217AF958758C5CCD1DA51` / 12482 B、`d7-five-leaves-r02-race.log` `36D49464FF311FB813F76725DF4098A3AE09FA7E03B1C9F24A2E5B3BFC037FD9` / 12434 B）；完整 8 叶、`TestRecoveryRuntimePostgres`、`TestRecoveryMainBusinessDispatcher` 普通与 `-race` 均 EXIT=0、fail=0、skip=0（`d7-main-dispatcher-r02.json`/`-race-r02`、`d7-recovery-postgres-r02.json`/`-race-r02`）；`canonical_recovery_replay_same_result` 以持久字段相等与零额外写入闭合；closed/unknown 返回 503 `platform.recovery_in_progress`+Trace 且下游零调用、成功后进入 `public_ready` 开流均按卡面 oracle 闭合；同一 RecoveryRunStore/runtime/pool、无第二 runtime/router/pool/authority/fallback 已证；`go vet` 普通与 `-tags=integration` EXIT=0、changed-file `gofmt -l` 为空（`d7-tooling-r02.log` `2463D2F17EE9BDF12F9ADCDFEA8C39A873718C438F5089371C64315A0F397006` / 295 B）。**②消费者与引用同步**：本卡无代码消费者；`article-management-S8.2` 仅追加事实指针（该卡“恢复主线承接说明”与“整卡接收/关闭仍阻塞”两条的 2026-10-31 更新），不改其 checkbox/日期/写集；`plans/ir-01-content-community-batches.md` 自述不复制任务状态、仅引用原卡链接，判定无需修改。**③当前版本证据**：HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；四写集文件 `cmd/han-server/main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B、`cmd/han-server/recovery_runtime.go` `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35832 B、`cmd/han-server/recovery_runtime_integration_test.go` `B8DEBCA4889FA648DBFC15D524C6CC4AAD10AE29AFD5C1F217E31DE44DB8C54A` / 31638 B、`cmd/han-server/recovery_production_chain_integration_test.go` `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` / 167339 B，与 R02 复审时逐项一致，关闭前无变化。**④独立评审与修复复审**：非作者 R02 结论 PASS WITH MINOR（`verdict-r02.md` `6AE8A942A8C28B51DA69EDDB26975E2A7C7745507B6EDFD5E9DD2CC4F1EA6064` / 3615 B；`report-r02.md` `A27D3C0F199BC34022F2B84C902045E471E7A51CB96B5D5F142F9C9A7742D567` / 22494 B；`commands-r02.log` `C52BDF1ADBA5DDD288AA505712191DB52B0B5EB0ABB8FF6719D21379043F3CD0` / 64977 B）；`S420-CR-F001`（HIGH）CLOSED，F002–F007 CLOSED，F008 接受 Lead 裁定 NOT A DEFECT；无 HIGH/MEDIUM 未处置；R01 三件未被覆盖。Lead 现场复核 `lead-recheck-r02.md` `76575B22BF7296ACB6921DF026C9B360A5391F52766AB063362C4193B4394966` / 3767 B。**⑤Lead 核 worktree/所有权/Git 约定**：`git diff --cached --name-only` 为空；`git status --short` 仅见既有已登记改动；`cmd/` 尚未纳入版本控制，故以 SHA256+字节+mtime 判定所有权，四文件与评审基线一致；本卡约定不提交，本轮未做任何 Git 写。**⑥约定接收/人的决定**：本卡为开发卡，无逐条人工签字约定；用户 2026-10-31 授权“对待确认或决策项采用建议值并继续本项目”用于 LOW 处置与流程推进，不构成人的业务接收、Case18 整体验收或部署授权。**残余 LOW 登记（不阻断本卡；归属“后续恢复/测试卫生改进”，当前无任务认领）**：`R02-F001` control server goroutine 未 join（测试装配；影响：退出前清理不严）；`R02-F002` 等价手工编排未走 `assembly.start`（影响：装配入口仅由指针+行为间接证明）；`R02-F003` 无第二 router/fallback 仅指针层证明（影响：缺运行期隔离断言）；`R02-F004` closed/unknown 未走 production dispatcher E2E（影响：该分支仅 handler 层验证）；`R02-F005` provider/ports 测试侧装配（影响：provider 自身不校验 permit，报告已披露）。**未验证边界（不因关闭而扩张）**：真实 keyring/HSM、approved backup inventory、production dispatcher 全路由、POSIX journal/fsync/flock/CAS、真实部署/E2E、进程级全局唯一 runtime 均未验证；enter/install 与 recovery secret candidate 的承接任务仍未登记（unknown）。**统计**：64→65 completed / 3 active / 9→8 todo。**边界**：本记录为 AI 代理按 §10.3 核证，非真人签字，不冒充人的业务接收；未改 S4.16/S4.17/S4.19/S8.2/S8.3 的 checkbox/日期/写集，未改四个代码文件字节，未做 Git 写或清理。
- **2026-10-31 B13H-GATE-F002 更正（不改上条原文；S4.20 checkbox 保持 `[x]`）**：批次 Gate 独立评审指出上条将 `d7-main-dispatcher-r02.json`（`A673F00E92532CFCC1FDBEADA5FF17EC468F2DF0E0734DC139568514503A79F7` / 9596 B）与 `-race` 复跑表述为满足 S4.19 两层/卡面 oracle，但该证据仅含 5 个子测试，不含层 (b)（真实 runtime 同 Session 200）。本更正限定：上条“两层闭合”表述作废，d7 仅证明 production dispatcher 在 closed/unknown/probe/single-handle/no-config 范围通过；层 (b) 由 `identity-auth-S4.19` 返工承接，S4.20 自身 resume 链仍以 `TestRecoveryProductionChainComplete` 真实 PG 证据为准。S4.20 `[x]` 与完成日期不变，但其 `S419Reverified` 条件改由 S4.19 返工后的同修订两层复跑复核；批次 Gate 在两层同修订通过前不得 PASS。`R02-F004`（closed/unknown 非 production dispatcher E2E）仍为 LOW，不因本更正升级。
- **2026-10-31 Plan 首行污染修复与返工基线刷新（Lead；用户 standing 建议授权；不改任何 checkbox/统计/requires/写集）**：Plan 首行污染（180 字符 PowerShell 输出前缀）已精确删除，全文规范为纯 CRLF，首行恢复 `# 批次 Plan: IR-01 内容发布社区完整实施地图`；Plan 由 `AF675793D97E5E9CDDB45F9C84C36EA7F0DAC40C249B1223D5EDC66DB65D3E31` / 489030 B 变为现行 `0437577BF765E1D9DD21E8FFFDFE3124F5C9B27A421C0F2CEE09300F52FF459F` / 488854 B / 1615 CRLF / BARE_LF=0，`:2944`/`:2945` 旧 Plan 指针为历史时点值，以本记录为准；本记录追加前 `tasks.md` 为 `E74EE62D4AF5B00815D3655E9E262437FE3E8DAF2DE60797D6C7E462D19C5632` / 754058 B / 2950 行 / CRLF=2950 / BARE_LF=0 / 无 BOM，本轮 Gate 处置 5 处写入经反向还原核验（撤销后 `9302547933674106ADBC108DA7E62AA5653DCA7973784CCCA60CDFE1EB233817` / 750565 B / 2948 行，与写前基线逐字节一致）；写集基线 `cmd/han-server/main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B、`cmd/han-server/recovery_runtime.go` `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35832 B、`cmd/han-server/recovery_main_dispatcher_integration_test.go` `B89DA3791616353ACBAC602B1F22AC98298CFE888FBD55E1D3A4686F7BA67A97` / 16500 B、`cmd/han-server/recovery_production_chain_integration_test.go` `65602F429197194FDD63305D667D050476805FEC7B12C227E7AD13583DFE6B75` / 167339 B、`cmd/han-server/recovery_workers_test.go` `B5CFC39DA1304026449549AAAC58C8231840A7F2A4166241AC8CD2B55D7B9532` / 28928 B；`xDesk.exe` 6 进程（PID 22928/25764/29960/35384/40736/49420）静默但仍存活，按并发写者风险处理；S4.19 保持 `[-]`、S4.20 保持 `[x]`，返工合同与基线明细见 [lead-baseline](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s419-rework-r01/lead-baseline.md) `C9F8E57688F9FEF394764949D261EA067DDA4E7C33FB1B1DF8164C77B7F17176` / 5617 B；本记录仅 AI Lead 按 standing 授权作流程处置，不代替独立复审、批次 Gate 重跑、人的业务签收或 Git 写授权；截至本记录未运行 Go/PG/HTTP/E2E。
- **2026-10-31 B13H 批次 Gate r02 PASS WITH MINOR / S4.19 返工后重新关闭（`[-]`→`[x]`，完成@2026-10-31；Lead 直接核证 + 独立 Reviewer 产出）**：**①复审**：非作者独立复审（Bacon / add-reviewer，actualID `01a19216-b34f-7162-ace6-ac6fd1b992d4`）s419-independent-review-r04 结论 PASS WITH MINOR，`report.md` `20A55736BE5131B79E555630479114391BFEBC83EE007743E5F8111A444F974A` / 17776 B、`verdict.md` `613A7ACE67D8888AF54382480610C87BFECACACF1208B9274DEF9FC148CE58F8` / 9506 B、`commands.log` `C4822E7E4B588BD71D6BAB08AA8767DB26CC3342EAAD6E5F0298501840F8EA93` / 5740 B；`B13H-GATE-F001`（HIGH）与 `N01`（LOW）CLOSED，新增 `S419-R04-F001`（泄漏断言禁词含 ref，随机 Trace 误命中 0.0725%）与 `S419-R04-F002`（真实 runtime closed `/readyz` 未在本文件断言）两条 LOW。**②批次 Gate 重跑**：b13h-batch-gate-r02 结论 PASS WITH MINOR，`report.md` `7893B4300E44C8BCFD9A326010EEB5FED6D565595F0DE40A6863B94E8E0EF3D4` / 18851 B、`verdict.md` `0D29C1FA222036E4212C52E20CC11E13B0510572BDA217B45A738D3748491964` / 4814 B、`commands.log` `EECA8604A39311E8DCF5E7556938D4FEB4547D236C443C688A1178B69DAB5AAE` / 3220 B；`B13H-GATE-F001`、`N01`、`B13H-GATE-F002`（S4.20 关闭记录口径，由 `:2950` 更正处置）均 CLOSED，无 HIGH/MEDIUM；17 唯一文件 / 21 声明槽 inventory 逐项匹配，复跑前后 drift=0、清单外写入 0；新增 `B13H-GATE-R02-L01`（子测试注册防覆盖回退）与 `B13H-GATE-R02-L02`（nil 回退缺运行期无第二 Store/pool 断言）两条 LOW。**③Lead 现场复跑**（`s419-lead-closure-r05`）：`gofmt -l`、`go build ./...`、`go vet` 普通与 `-tags=integration` 均 exit 0；dispatcher 普通与 `-race` 各 pass=12/fail=0/skip=0/DATA RACE=0；真实 PG `TestRecoveryRuntimePostgres` 普通与 `-race` 各 pass=7/fail=0/skip=0/DATA RACE=0；两层同修订同跑满足 `:2859`/`:2868`。**④哈希**：dispatcher 测试 `BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A` / 23693 B / 623 行 / 纯 LF 无 BOM；`main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B（零 production 改动）；本记录追加前 `tasks.md` `F662C05814344289410CA7192E7B54F3D30F2207C4BA2AC4DED92E94E0808939` / 756211 B / 2951 行；Plan `0437577BF765E1D9DD21E8FFFDFE3124F5C9B27A421C0F2CEE09300F52FF459F` / 488854 B。**⑤边界**：4 条 LOW 已披露并归属后续测试卫生 / 运行期隔离改进，当前无任务认领，不阻断本卡；未改 S4.16/S4.17/S4.20/S8.2/S8.3 的 checkbox、日期、写集，S1.5 整卡仍 `[-]`；不证明性能、真实部署/E2E、POSIX journal/fsync/flock/CAS、真实 keyring/HSM、approved backup inventory、production dispatcher 全路由、进程级唯一 runtime/Store/pool；`TestRecoveryRuntimeProductionChain`（53519）与 S4.20 `TestRecoveryProductionChainComplete`/五叶保持 NOT RUN。**⑥统计**：64 completed / 4 active / 8 todo → 65 completed / 3 active / 8 todo。本记录为 AI 代理按 `task-execution-workflow.md` §10.3 核证，非真人签字，不代替人的业务接收、部署授权或 Git 写授权。
- **2026-10-31 article-management-S4.4 Lead Gate 关闭（`[-]`→`[x]`，完成@2026-10-31；Lead 直接核证 + 非作者独立 Reviewer 产出）**：**①本 task 成果与验收完成**：卡面三类 tombstone、operation/audit 对账、双 backup 并发、输入漂移、双次 ReadStatus 与 inventory 缺失 fail-closed 均有当前修订真实 PostgreSQL 证据；`go test -tags=integration ./test/integration/recovery -run 'TestArticleReplay|TestOldBackupRestore' -count=1 -v` EXIT=0、`ok han/test/integration/recovery 20.478s`（33 PASS / 0 FAIL / 0 SKIP / 0 DATA RACE；日志 [s44-lead-rerun.log](../../../../.spec/wip/article-management/evidence-s4.4/lead-closure-r01/s44-lead-rerun.log) `E1EAA235EC7F078BB66F81B64C1B259BCC90EA4944C90217220DBC27737E144C` / 105322 B）。**旧备份恢复仅为模拟 fixture 子集**：`old_backup_restore_test.go` 自述 synthetic 且 `production_use=forbidden`，真实旧备份恢复/RPO/RTO/inventory 按卡面 `:1591` **不关闭真实演练部分**。`go build ./...`、`go vet ./...`、`go vet -tags=integration ./cmd/han-server` 均 EXIT=0，目标文件 `gofmt -l` 为空。**②消费者与引用同步**：本关闭不改生产代码或测试断言；Plan 追加日期指针；`S44-CR-F001` 以**保留旧索引原字节 + 新增 [当前树权威索引](../../../../.spec/wip/article-management/evidence-s4.4/closure-review-r01/current-tree-index.json)（`C8FB08C9AF8FFD2938EFD39B143F0BB381FDBD19EA7D1D0D95723DB6FD17084C` / 3249 B）、[更正说明](../../../../.spec/wip/article-management/evidence-s4.4/closure-review-r01/evidence-index-correction.md)（`E5BD7467AB2662C3C25E6683E21B24B552189F2AFE99BF7300E925486A926F32` / 2105 B）与 [旧索引邻接 SUPERSEDED 指针](../../../../.spec/wip/article-management/evidence-s4.4/harness-publication-r01/SUPERSEDED.md)（`A4B52BD3A199F4F036F26AF37076E7B4B275D17254AFB20457CFD132266FBC1C` / 758 B）**处置，不覆盖旧证据；未改 S1.5/S8.2 或其他任务卡。**③当前版本证据**：HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；`cmd/han-server/main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B、`cmd/han-server/recovery_runtime.go` `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35832 B、`test/integration/recovery/article_replay_test.go` `1BD2B0ED9385973533C611EEEF9CD17A9B70149AFBD5339829B9EDEF9906B0CD` / 86718 B、`test/integration/recovery/replay_evidence_test.go` `C23C5C776627C4D065FE5E805A64BB582ED4EE9FB8A3FBB157FD1818949E88DB` / 29878 B、`test/integration/recovery/old_backup_restore_test.go` `36B13366F968FF9BA1BD1AA00AF90458A7EE89DD3A7FCBDFE69F6FF07317E536` / 15482 B、`internal/article/deletion_journal_hash_test.go` `10F940D8C18984BD5D806BB9D5D6379DC644B22ACE626EB2BF6E4A59AC33FB48` / 1349 B，与本轮复核一致。`cmd/han-server/main.go:641` 实际调用 `NewReplayPublicationPorts`；seal 位于 production composition 并由 `cmd/han-server/recovery_runtime.go:611-615` 调用。**④独立评审与修复复审**：非作者独立 Reviewer（Bacon / add-reviewer，actualID `01a19216-b34f-7162-ace6-ac6fd1b992d4`）结论 **PASS WITH MINOR，0 blocker**；[report.md](../../../../.spec/wip/article-management/evidence-s4.4/closure-review-r01/report.md) `BEFC55F6D7CBFA3C1877AC3FA613F76EE0680E8C987B1EFADFE3CAEB53726B09` / 18479 B、[verdict.md](../../../../.spec/wip/article-management/evidence-s4.4/closure-review-r01/verdict.md) `770E982DACEEBEC34024CC3C05E553D04792677B20F34864F670A15DF888E09E` / 4008 B；声明“AI 代理评审，非真人签字”。`S44-CR-F001`（MEDIUM，非阻断，旧索引陈旧/缺 superseded 指针）由本轮不可变索引更正处置；`F002`（生产恢复重放 HTTP 路径未接通）、`F003`（无 recovery 配置时 seal 不执行）、`F004`（生产 snapshot 不重算 hash）、`F005`（CheckPhase 三路径不可达）均为已披露 LOW，保留为后续恢复/测试卫生改进，不阻断本卡。**⑤worktree/所有权/Git**：Lead 在真实工作树复核 HEAD、`git status --short`、缓存区与写集哈希；约定不提交，`git diff --cached --name-only` 为空，本轮未做任何 Git 写或清理；`cmd`/`test`/`internal` 未纳入 Git 的边界以 SHA256+字节核验。**⑥约定接收/人的决定**：用户 2026-10-31 “对待确认或决策项采用建议值并继续本项目”是流程推进/LOW 处置的 standing 授权，**不构成人的业务签收**；本记录为 AI 代理按 §10.3 核证，非真人签字，不代替 `ISSUE-article-management-0001` 的人的业务接收、部署授权或 production readiness。**未验证边界（关闭后不得外推）**：真实 POSIX Journal/fsync/flock 与崩溃/磁盘故障；真实 keyring/HMAC/HSM；真实旧备份恢复、真实 backup inventory、RPO/RTO/retention；identity 双 namespace/MarkConsumed/联合开流；真实部署/E2E；Case18 `NOT RUN`；生产恢复重放入口（F002）；无 recovery 配置 seal 行为（F003）；生产 snapshot 重算 hash（F004）；`CheckPhase`（F005）；`enter`/`install`/recovery secret candidate；进程级唯一 runtime/Store/pool 与性能/P95。**统计**：65 completed / 3 active / 8 todo → **66 completed / 2 active / 8 todo**；active 保留 `article-management-S1.5`、`article-management-S8.2`。**边界**：未改其他卡 checkbox/日期/写集，未宣称 B8A/整批/需求整体验收或 production readiness。

<a id="task-identity-auth-s4-21"></a>

- [x] identity-auth-S4.21 在唯一production `cmd/han-server` root上登记 enter/install 的显式 closed 端口清单与可注入装配 seam、以及 recovery secret candidate seam，使 Case18 “全局恢复请求隔离/完整恢复后生产开流” 具备可核承接；production 默认仍 closed，不重做 S4.20 已闭的 resume 主线。 完成@2026-11-01

- **目标**：在唯一production `cmd/han-server` root上，对 `cmd/han-server/recovery_runtime.go:482-483`（`case "enter", "install"` 的前置整路径 closed）在构造任何资源前增加显式 precheck：缺任一 enter/install 端口即返回 `recovery.ErrAdmissionClosed`，错误文本列明全部缺失端口名，不得静默放行；把既有 `inputs.installation` seam（`:556-557`）与既有可选端口（`commitments`/`sources`/`anchor`/`exclusive`/`journal`/`articleSnapshots`/`restore`/`consumer`/`articleAccount`）纳入该 precheck；在 `recoveryRootInputs` 新增未导出字段 `secretCandidate func(context.Context, recovery.RunBinding) ([]byte, error)`（对应 `:582-583`），非 nil 时使用，nil 时保持 closed，production `main.go` 不注入故仍 closed。integration harness 注入同一 Store/Journal/端口时，同一 root 的 `Enter`/`Install` 装配路径可执行。本卡不实现、不伪造任何 production 端口，不得用测试 seam 冒充 production 可达；不得建第二 runtime/Store/pool/authority/fallback，不得直接 `setState("public_ready")` 冒充 enter/install 完成。

- **类型/Story与范围**：开发（同卡局部装配 seam 测试，范围限定为 enter/install 显式端口清单、可注入装配 seam 与 recovery secret candidate seam）；identity Story S04（建立并保护初始管理员，范围含首次启动 bootstrap、独立 recovery secret、active/verify-only keyring 跨版本防重放、恢复信任根与强制改密）锚定；关联 article Story4/Story8 与 Case18 接收，但本卡不关闭、不重开 `article-management-S8.2`。消费已闭 `identity-auth-S4.13`（安装）、`identity-auth-S4.16`（唯一root assembly/同Store/同Runtime；S4.16 的 enter/install 形态与安装绑定在组件层已闭（`ConfigCommandAndLazySecret`/`InstallRootDrainAndSingleBinding`），本卡只补唯一 production root 的显式 closed 端口清单、可注入装配 seam 与接线，不声称 production 可达）、`identity-auth-S4.18`（Install producer/精确checker）、`identity-auth-S4.19`（production dispatcher guard/probes/同handle）、`identity-auth-S4.20`（resume 主线/同 runtime/handle）的证据。明确不证明：production `Enter`/`Install` 可达、真实部署/E2E/POSIX/ACL/真实 provider/真实备份、完整安装/恢复入口的整体业务签收（本卡只承接显式端口清单、装配 seam 与 secret candidate seam 的局部链）；真实环境不可用时按 ADR-0028 用模拟环境并标注“未真实验证”。

- **R对账（四标签）**：
  - **已建成**：S4.13 已提供 `Install(context.Context, InstallationManifest) error` 安装合同（`design-workspace/identity-auth/design.md:1463`）；S4.16 已在唯一root assembly构造同一 `RecoveryRunStore`/runtime，且 `RunNormal/Enter/Resume` 只能由同一 root 选择其一（`design.md:1520`）；S4.18 已提供 Install producer 与精确 checker；S4.19 已把同一 handle 接入 production dispatcher（closed/unknown 503+Trace）；S4.20 已闭合 resume 主线并复用同一 runtime/handle/Store；S4.16 关闭记录已声明“生产 CAS/anchor、installation commitment、source verifier、deployment lease、keyring loader、restore adapter 与真实 provider/ACL 未交付，recovery 仍显式 closed”（`tasks.md:2727`）；`InstallationBinding` 投影已定义（`design.md:1524`）。
  - **待做**：在 `cmd/han-server/recovery_runtime.go:482-483` 构造任何资源前调用显式 precheck：enter/install 缺任一必填端口即返回 `recovery.ErrAdmissionClosed`，错误文本列明全部缺失端口名；`:556-557` 的 `inputs.installation` 已是既有可注入 seam，本卡只把它纳入 precheck，不新建/替换 seam；`:582-583` 在 `recoveryRootInputs` 新增未导出字段 `secretCandidate func(context.Context, recovery.RunBinding) ([]byte, error)`，非 nil 时使用，nil 时保持 closed，production `main.go` 不注入故仍 closed。`commitments`/`sources`/`anchor`/`exclusive`/`journal`/`articleSnapshots`/`restore`/`consumer`/`articleAccount`（`:502-506`/`:537`/`:543-562`）已是可选 seam，precheck 对 enter/install 要求这些输入非 nil；production anchor registry 仍为 `newRecoveryAnchorRegistry(nil)`；完整必填清单与测试断言见风险叶 `EnterInstallRuntimeSelectionOnSingleRoot`。逐端口登记仍未交付的 closed 端口（修订前代码事实）：anchor registry 四端口 `:502-506`（`newRecoveryAnchorRegistry(nil)` + `resolve`，见 `cmd/han-server/recovery_sources.go:470-492`）、journal head anchor `:537`、verified journal reader `:543-546`、restore driver `:547-550`、consumer `:551-554`、installation executor `:556-557`、article account `:559-562`、secret candidate `:582-583`；`Enter` 在 `Exclusive.Acquire`/restore/`ArticleSnapshots` 处失败，`Install` 在 `installPreflight` 的 `Commitments`/`Sources`/`Anchor` 处失败（`internal/recovery/runtime.go:1554-1571`、`:1573-1597`、`:322-335`；`internal/recovery/admission.go:364-375`）；production restore source/target/tool 与 anchor adapter 实现不存在（`cmd/han-server/recovery_restore.go:174-205` 仅接口与构造，唯一调用在 `cmd/han-server/recovery_restore_test.go:201`）。在同修订上回归复跑配置/命令旁路矩阵：回归解析/拒绝路径；模拟 seam 下覆盖装配分支；不声称 production 执行可达；S4.16 已闭的 normal/resume/verify 解析不重做，不重开 S4.16（`identity-auth/test-plan.md:837`）；证明 closed/unknown 保持隔离；Case18 两个 blocked-upstream 分项以承接记录可核（不解除阻塞）；新增装配与配置/命令矩阵专项 integration 测试普通与 `-race`。
  - **需改**：仅下节列出的 candidate write set（2改+1新增）；把三处 closed 占位改为端口清单校验与 seam 注入，不实现 production 端口。
  - **废弃**：无；不重开任何已关闭卡，不废弃既有组件。

- **设计/测试依据**：`design-workspace/identity-auth/design.md:1463`（`Install(context.Context, InstallationManifest) error`）、`:1520`（`RunNormal/Enter/Resume` 只能由同一 root 选择其一）、`:1524`（`InstallationBinding`）；`identity-auth/test-plan.md:837`（配置/命令旁路运行矩阵：无参数 normal、enter 安装形态、enter run+backup、resume 同绑定、verify 无 caller run 逐一正控）；`design-workspace/article-management/test-plan.md:359`（全局恢复请求隔离 `blocked-upstream`：“当前通用请求Guard/生产隔离承载未交付，不能用专用Gate、fake503、t.Skip或删断言计PASS”）、`:360`（完整恢复后生产开流 `blocked-upstream`：“本卡不新建authority、指派身份owner或扩S10.5”；该处“本卡”指 `article-management-S8.2` 卡；S4.21 同样承诺不新建 authority、不扩 S10.5，但此句非 S4.21 原文自述）；承接缺口原句 `tasks.md:2879`（S4.20 卡：“本卡范围为 resume，不含 enter/install 与 recovery secret candidate 关闭点。”）与 `tasks.md:2881`（S4.20 卡：“enter/install/secret 的后续承接任务未登记（unknown）”）；代码事实（修订前）：`cmd/han-server/recovery_runtime.go:482-483`、`:502-506`、`:537`、`:543-550`、`:551-562`、`:582-583`，`cmd/han-server/recovery_restore.go:174-205`（仅接口/构造，生产实现不存在），`internal/recovery/runtime.go:1554-1571`/`:1573-1597`/`:322-335`，`internal/recovery/admission.go:364-375`。ADR-0027（唯一Store/正向准入）与ADR-0028（项目级验证授权）。设计仍draft、人的精确接纳unknown；所有未来命令NOT RUN。

- **文件所有权**：
  - `cmd/han-server/recovery_runtime.go`（修改；在唯一root assembly内为 `:482-483` 增加构造前显式 precheck；把既有 `inputs.installation` seam（`:556-557`）与既有可选端口纳入 precheck；新增 `recoveryRootInputs.secretCandidate` 未导出字段 seam（`:582-583`），非 nil 使用、nil 保持 closed；缺端口返回 `recovery.ErrAdmissionClosed` 并列出全部缺失端口；不实现、不伪造 production 端口；复用同一 runtime/Store/pool，不建第二 runtime/Store/pool/authority）
  - `cmd/han-server/main.go`（修改；唯一 production root 接线，仅必要时；不新增第二 router/authority）
  - `cmd/han-server/recovery_enter_install_integration_test.go`（新增；`//go:build integration`，同包；含装配/secret candidate seam 测试 `TestRecoveryEnterInstallAssemblyWiring` 与配置/命令矩阵 `TestRecoveryEnterInstallConfigCommandMatrix`；矩阵表驱动覆盖配置/命令正控、非法 env 与缺/重复/未知 flag、相对 manifest 负控，断言 error 且解析阶段临时目录零写入；使用新文件自包含 env helper（参考 `recoveryConfigFixture` 风格），不改既有测试文件，不要求真实 PG；不得用测试 seam 冒充 production 可达）
  - 不默认修改 `internal/identity/*` 或 `internal/article/*`；若实施确需任何清单外文件（第 4 个代码文件）、新 authority、或改 `internal/identity|article` 包，立即 STOP 回 Lead 并回设计。
  - 不新建 authority、不扩 S10.5、不重开任何已闭卡。
  - `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 为非代码证据产物，不占 3 文件代码写集，不计入“第 4 个代码文件”判定；由实施者在关闭前创建，Lead 复核字段完整性；若无法创建或字段不完整，任务不得关闭。

- **文件数**：3 个候选（2改+1新增）。若实施需任何清单外文件（第 4 个代码文件）、新 authority，或改动上述以外的包，立即停止并说明冲突与串行条件。

- **依赖**：requires `identity-auth-S4.13`（安装）、`identity-auth-S4.16`（唯一root assembly/同Store/同Runtime）、`identity-auth-S4.18`（Install producer/精确checker）、`identity-auth-S4.19`（production dispatcher/guard/同handle）、`identity-auth-S4.20`（resume 主线/同 runtime/handle）。`article-management-S8.2` 仅作后续验证关联/消费者，不写入 hard requires，不形成环、不反向阻塞已关闭卡。不 requires 任何未登记任务。

- **批次关联**：[B13H](../plans/ir-01-content-community-batches.md#b13h) 唯一root批次（共享 `cmd/han-server/main.go`、`recovery_runtime.go` 与既有 recovery integration 系列，属同一唯一root串行写窗）；不新建批次。Plan 已在 B13H 登记本卡与新增 integration test（`plans/ir-01-content-community-batches.md:1309` 外层表、`:1333` 所有权索引、`:1445-1450` DAG 节点 `n34`、`:1611` 登记记录）；B13H 由 17 唯一文件/21 声明槽更新为 18/24；Plan `:1611` 已追加 R01→R02 与 R02→R03 修订记录，本卡不复制第二份任务状态、不改 Plan。

- **验收命令**（根目录，待实施/NOT RUN；本卡登记阶段不执行任何命令）：

```text
gofmt -l cmd/han-server/main.go cmd/han-server/recovery_runtime.go cmd/han-server/recovery_enter_install_integration_test.go
go build ./...
go vet ./cmd/han-server
go vet -tags=integration ./cmd/han-server
go test ./cmd/han-server -run '^TestRecoveryRuntimeComposition$' -count=1 -json
go test -race ./cmd/han-server -run '^TestRecoveryRuntimeComposition$' -count=1 -json
go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json
go test -tags=integration ./cmd/han-server -run '^TestRecoveryEnterInstallAssemblyWiring$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryEnterInstallAssemblyWiring$' -count=1 -json
go test -tags=integration ./cmd/han-server -run '^(TestRecoveryConfig|TestRecoveryCommand|TestRecoveryEnterInstallConfigCommandMatrix)$' -count=1 -json
go test -race -tags=integration ./cmd/han-server -run '^(TestRecoveryConfig|TestRecoveryCommand|TestRecoveryEnterInstallConfigCommandMatrix)$' -count=1 -json
```

- **必要风险leaf / 最低层**：

| 必要风险leaf | 最低层 | 验收事实 |
|---|---|---|
| `EnterInstallRuntimeSelectionOnSingleRoot` | R/P | precheck 必填输入：`journal`、`articleSnapshots`、`restore`、`consumer`、`installation`、`articleAccount`、`commitments`、`sources`、`anchor`、`exclusive`、`secretCandidate`；测试至少验证：(a) 仅 production 默认输入时返回 `recovery.ErrAdmissionClosed` 且错误含全部缺失端口名并与 precheck 清单一致；(b) 全部注入时 `Enter`/`Install` 到达模拟端口的 sentinel/预检失败点，证明装配路径而非完整成功；(c) 同一 root/Store、无第二 runtime。模拟边界下装配路径可达；production 缺端口保持 closed；不得并行调用或第二 runtime |
| `InstallationManifestSourceAndExecutor` | R/P | `inputs.installation` 已是既有可注入 seam（`:556-557`），本卡只把它纳入 precheck，不新建/替换 seam；仅证明装配/接线；不称 production executor 已交付，生产缺端口仍 `ErrAdmissionClosed` |
| `RecoverySecretCandidateSource` | R/P | 在 `recoveryRootInputs` 新增未导出 `secretCandidate` 字段 seam（`:582-583`），非 nil 使用、nil 保持 closed，production `main.go` 不注入；仅证明装配/接线；不称 production source 已交付；缺/错/复用 secret 保持停服（story S04 验收） |
| `SeamNotProductionReachability` | U/C | 模拟 seam 注入只证明装配路径可执行；不得据此声称 production `Enter`/`Install` 可达、Case18/S8.2 PASS 或 blocked-upstream 解除；production 默认路径仍 closed |
| `ConfigCommandMatrix` | C/U | 由 `TestRecoveryEnterInstallConfigCommandMatrix`（同包、`//go:build integration`、表驱动、自包含 helper）实际覆盖：配置正控 normal、enter 安装形态（action install）、enter run+backup、resume 同绑定、verify 无 caller run；配置非法 env 负控均 error 且临时目录零写入；命令正控 `selectRecoveryCommand`/`parseRecoveryCommand` 覆盖上述形态；命令负控缺 flag、重复 flag、未知 flag、相对 manifest 均 error；由验收命令中的矩阵两条（普通与 `-race`）执行；只回归解析/拒绝，不把解析通过当成 Enter/Install 执行可达（同修订回归；S4.16 `ConfigCommandAndLazySecret` 已闭 normal/enter两形态/resume/verify 解析，本卡只补唯一 production root 的 closed 端口清单校验与回归，不重开 S4.16） |
| `ClosedUnknownKeepIsolation` | R/P | closed/unknown 返回 503 `platform.recovery_in_progress`+Trace，Article service/repository/author-reader 零调用（同修订回归复跑并覆盖 enter/install 形态；S4.19/S4.20 已闭行为不复述为新实现） |
| `SingleRuntimeStorePoolNoAuthority` | U/C | 同一 recovery runtime/Store/pool，无第二实例、无新 authority、不扩 S10.5 |
| `Case18BlockedUpstreamHandoff` | R | Case18 `:359`/`:360` 两个 blocked-upstream 分项在 `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 留下承接记录（字段至少含分项、当前状态 blocked-upstream、承接卡 S4.21、剩余 closed 端口清单、消费者 S8.2、证据指针、责任人/日期）；不声称 Case18 PASS 或整体解除阻塞 |

- **本任务关闭条件**：`go build ./...` 通过；`TestRecoveryEnterInstallAssemblyWiring` 普通与 `-race` 通过（`-tags=integration`）；S4.16 `TestRecoveryRuntimeComposition` 与 S4.19 `TestRecoveryMainBusinessDispatcher` 同修订回归（普通与 `-race`）通过；配置/命令旁路运行矩阵由 `TestRecoveryEnterInstallConfigCommandMatrix` 的普通与 `-race` 两条命令实际覆盖并逐项正控/负控闭合；不得只用解析通过替代 Enter/Install 执行可达；`:482-483` 前置 closed 被显式 precheck 替换；`:556-557` 既有 `inputs.installation` seam 与 `:582-583` 新增 `secretCandidate` seam 纳入 precheck；缺任一必填端口返回 `recovery.ErrAdmissionClosed` 且列明全部缺失端口，closed/unknown 保持隔离；模拟边界下同一 Store/runtime/pool 装配且无第二 runtime/router/pool/authority/fallback；`go vet`（普通与 `-tags=integration`）与 changed-file `gofmt -l` 通过；production 默认仍 closed，不得声称 Case18/S8.2 PASS 或解除 blocked-upstream；`.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 存在且字段完整（由实施者创建、Lead 复核；无法创建或字段不完整则不得关闭）；独立非作者 CodeReview、Lead 同 revision 核 status/diff/所有权；未真实验证项已标注。不得用 build 成功、测试 seam 或直接 `setState` 代替 production 可达、测试、评审或 E2E。

- **后续验证关联**：`article-management-S8.2` 为后续消费者，保持 `[-]`，待本卡与应用独立验证完成后再由 Main 评估，不自动关闭、不改其九文件写集；Case18 整体及依赖它的 S8.2 整卡接收/关闭仍按上游权威→隔离→开流合同保持 blocked，本卡不解除 blocked-upstream、不声称 production 开流。不证明真实部署/E2E/POSIX/ACL/真实 provider/真实备份；真实环境不可用时按 ADR-0028 用模拟环境并标注“未真实验证”。

- **串行/停止条件**：`cmd/han-server/main.go` 与 `cmd/han-server/recovery_runtime.go` 是唯一root共享文件，实施前 Main 须重核实际 writer/写窗，未知即停；实施前记录两者各自的 SHA256/字节数/mtime，期间任一变化即 STOP 重核。不得重开或改写 `identity-auth-S4.13`/`S4.16`/`S4.18`/`S4.19`/`S4.20`/`article-management-S8.2` 的 checkbox、日期、关闭记录、requires 或文件所有权；不得改 S8.2 九文件写集。需改设计契约、identity/article 包文件、任何清单外文件（第 4 个代码文件）或无法保持 B13H 一致时立即停止返回；需新增 authority 或扩 S10.5 时立即停止返回。禁止 Git add/commit/push/checkout/stash、清理/删除/移动/归档；本卡登记阶段不运行 go test/vet/gofmt/PG/HTTP/service/restore/E2E。本卡初始 `[ ]`，无完成日期；PlanningReview/Main 接收/实施 START/NOT RUN 证据均 pending。

- **本卡登记依据/未验证边界**：依据用户 2026-10-31 明确授权登记 enter/install 与 recovery secret candidate 的后续承接范围，承接 S4.20 `tasks.md:2879`/`:2881` 已声明的 unknown 缺口与 Case18 `article-management/test-plan.md:359`/`:360` 两个 blocked-upstream 分项；2026-11-01 按 R01 规划评审 CHANGES REQUIRED（`S421-PLAN-F001`–`F005`）收窄修订为显式 closed 端口清单 + 可注入装配 seam，不再声称 production `Enter`/`Install` 可达。消费既有已关闭组件证据，不把组件 PASS 相加成完整安装/恢复入口或 Case18 PASS；production 仍 closed，模拟 seam 不等于 production 可达；剩余 closed 端口与生产实现缺失项见“待做”。设计仍 draft、人的精确接纳 unknown；所有验收命令 NOT RUN；`TestRecoveryEnterInstallAssemblyWiring` 与 `TestRecoveryEnterInstallConfigCommandMatrix` 当前不存在、NOT RUN；真实 provider/ACL/备份/部署/E2E/POSIX unknown。不代替独立评审、人的业务签收、实施授权、PG/API 运行授权、清理或 Git 写授权。
- **2026-11-01 S4.21 PlanningReview R04 PASS（仅登记规划评审，不改任务状态）**：非作者 add-reviewer 对 Plan `3FA19CDD4902F56B8C2F1DF59C570D96FAAE02D8AEE519B4B4165DCCA3637380` / 497542 B / 1628 行与 tasks.md `7EAB7278D70F6326DBC4B79E496D52AF4152D895A849F5AC2F9F2A57ED77C6CF` / 802158 B / 3027 行判定 **PASS**；R01 CHANGES REQUIRED → R02 CHANGES REQUIRED → R03 PASS WITH MINOR → R04 PASS；`S421-R03-F001`（LOW）**CLOSED**，无新 finding。证据：[report](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s421-planning-review-r04/report.md) `6AB12EF39877C18AD179FEFFADD183C2D706E1498BF2051D53FBE978B8DD08AD` / 4739 B、[verdict](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s421-planning-review-r04/verdict.md) `72624C23F814305D36D2EC0E1D5CE2E9D71D84603D0F6F447C789A4B34420D4D` / 1883 B、[commands](../../../../.spec/wip/recovery-runtime/evidence-b13h-r01/s421-planning-review-r04/commands.log) `0229C61524066FD7129E91F4A10F785C652E238D01578A00BB6BC6B7F0989FD4` / 2225 B。本记录只确认规划合同通过：S4.21 仍 `[ ]`、无完成日期；实施/Main 接收/START/测试均未开始，全部验收命令 NOT RUN；`cmd/han-server/recovery_enter_install_integration_test.go` 与 `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 尚不存在；production `Enter`/`Install` 仍 closed，Case18 与 S8.2 保持 blocked-upstream。评审为 AI 代理核证，非真人签字；actualID 本会话不可得，不编造；不代替实施授权、人的业务接收或 Git 写授权。
- **2026-11-01 S4.21 认领 START（`[ ]`→`[-]`；仅状态推进，实施未开始）**：add-task-planner 按 `task-execution-workflow.md` §10.2 使用 `add-batch-plan` 对账后登记真实认领；B13H 关联沿用 Plan `:1309`/`:1333`/`:1445-1450`/`:1611`（18 唯一文件/24 声明槽），不新建批次、不改 Plan。R04 规划评审 PASS（`tasks.md:3028`）仅确认规划合同通过，不等于实施开始、测试通过、Main 接收或任务完成；S4.21 仍无完成日期，实施、独立评审与关闭均未开始，全部验收命令 NOT RUN。`cmd/han-server/recovery_enter_install_integration_test.go` 与 `.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` 尚不存在；production `Enter`/`Install` 仍 closed，Case18 及其消费者 `article-management-S8.2` 保持 blocked-upstream（S8.2 仍 `[-]`）。**兼容性静态核对**：`cmd/han-server/recovery_production_chain_integration_test.go:2991` 的 `recoveryRootInputs` 使用 `:2983` 构造的 `s420ResumeStartupPlan`，该 helper `:2887` 已断言 `plan.action == "resume"`，因此 S4.21 计划的 enter/install 专属端口 precheck 不会打破既有 S4.20 兼容测试；该结论仅来自静态代码核对，未运行任何测试。**基线**：tasks.md 改前 `449AEF464C2816049F3442459832FE2416724A3245B09CA213ABB876F8F16DAF` / 803717 B / 3028 行；本次仅预计第 2963 行 checkbox 与末尾新增本记录两处变化，改后哈希/字节/行数由 Lead 复算。Plan `3FA19CDD4902F56B8C2F1DF59C570D96FAAE02D8AEE519B4B4165DCCA3637380` / 497542 B / 1628 行；`cmd/han-server/main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B / 2026-10-31 19:26:09、`cmd/han-server/recovery_runtime.go` `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35832 B / 2026-10-31 19:25:50（仅引用 Lead 本轮复核）。**统计**：66 completed / 2 active / 9 todo → 66 completed / 3 active / 8 todo；active 新增本卡，保留 `article-management-S1.5`、`article-management-S8.2`。**边界**：未实施代码、未运行 Go/PG/HTTP、未做 Git 写、未改 Plan 或其他卡；本记录为 AI 代理按 §10.2 作状态推进，非真人签字，不代替 Main 业务接收、实施授权、部署授权或 Git 写授权。
- **2026-11-01 identity-auth-S4.21 Lead 关闭（`[-]`→`[x]`，完成@2026-11-01；Lead 直接核证 + 非作者独立 Reviewer 产出）**：按 `task-execution-workflow.md` §10.3 逐项核对。**①本 task 成果与验收完成**：卡面 `:2994-3005` 十二项验收命令在同一修订（HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`）由 Lead 现场复跑全部 EXIT=0，并由非作者 Reviewer 独立复跑同 12 条再次 EXIT=0；test2json 合计 pass=116 / fail=0 / skip=0、无 `DATA RACE`（`c05`–`c12` 八份 JSON）。`recovery_runtime.go:527-535` 以显式 precheck 替换原 `:482-483` 前置整路径 closed，precheck 早于 `:537` 首个资源构造；11 必填端口顺序与卡面 `:3012` 逐项一致（journal / articleSnapshots / restore / consumer / installation / articleAccount / commitments / sources / anchor / exclusive / secretCandidate），错误以 `%w` 包装 `recovery.ErrAdmissionClosed` 并以 `strings.Join` 列全缺失项；`secretCandidate` nil 保持 closed、非 nil 走注入 seam；`main.go:634-650` 未注入任何 enter/install 端口，production `Enter`/`Install` 仍 closed。**②独立评审**：非作者 add-reviewer（Zeno，actualID `01a19724-6cc9-7a41-b163-700b1cdcb4a6`）s421-independent-review-r01 结论 **PASS WITH MINOR**，16/16 关闭条件可核成立，`report.md` `0B27AF34363D9FF5B25E7E7D4DD88AB3D896C685221F5E70C82D8165773F2634` / 15873 B、`verdict.md` `C36A9D4C99B157389DF2009FD495C75B35A8ACD3EA3DAADDE19DF965591C6514` / 1581 B、`commands.log` `9FFAFB18A18C98980A10B826A560D5797A83D5F78CF22169888BC8C9CF43342F` / 8021 B；唯一 finding `S421-CR-F001`（LOW，非阻断）指向 Lead 复跑日志第 11/12 行命令文本未写全三重正则，已由 Lead 仅修正日志文本（`s421-lead-verify-r01/commands.log` 新 `78DE48C9AAB172206152CCC1FF9F3296F86AA5B1968753FE866599AF75DFDE28` / 3317 B，含修订说明，未重跑、未改代码或 JSON 证据）→ **CLOSED**；代码与测试零返工。**③当前修订哈希**：`cmd/han-server/recovery_runtime.go` `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` / 37431 B、`cmd/han-server/recovery_enter_install_integration_test.go` `6BFA7D761432C960740F4E1FE91566317B5707B8481CE0D9FF1CF46DD4106C3A` / 20564 B、`cmd/han-server/main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B（零改动）、`.spec/wip/recovery-runtime/evidence-b13h-r01/s421-handoff.md` `2A0013FD4A92325DAD3BA35CF3444DBFA10B5A0884F17E335E26F591B8768019` / 3155 B；`cmd/`、`internal/` 当日修改仅上述 2 个代码文件，无第 4 个代码文件、`internal/**` 零改动、既有测试文件未改；本记录追加前 `tasks.md` `B739369773DAB6BD041AC6AFC7912379642F5ECD4A223D71670A2B1B775EC9AF` / 806010 B / 3029 行（其中 `:47` 三态旧值 `66 completed / 2 active / 8 todo` 的 active 已陈旧，本轮按真实 checkbox 一并更正）。**④边界（关闭后不得外推）**：production `Enter`/`Install` 仍 `ErrAdmissionClosed`，本卡只交付显式 closed 端口清单、可注入装配 seam 与 recovery secret candidate seam；真实 PostgreSQL 下的 S4.20 链路本轮未重跑，未做 E2E、真实备份、部署；不宣称 Case18 或 `article-management-S8.2` PASS，**不解除 blocked-upstream**；不证明 POSIX/fsync/flock、真实 keyring/HSM、真实 provider/ACL、进程级唯一 runtime/Store/pool 或性能/P95。**⑤统计**：66 completed / 3 active / 8 todo → **67 completed / 2 active / 8 todo**；active 保留 `article-management-S1.5`、`article-management-S8.2`（均仍 `[-]`）。**⑥约定接收与工作树**：用户 2026-11-01 “对待确认或决策项采用建议值并继续本项目”为流程推进的 standing 授权，**不构成人的业务签收**；本记录为 AI 代理按 §10.3 核证，非真人签字，不代替部署授权或 Git 写授权；本轮未做任何 Git 写（add/commit/push/checkout/stash/restore/clean）、未删除或移动任何文件（根目录 `'){`、`$prof`、`.red-a.patch` 等待用户逐项确认后处置）；`xDesk.exe` 6 进程（PID 22928/25764/29960/35384/39592/49420）仍存活，按并发写风险处理，实施后四对象哈希复核未漂移。


<a id="task-identity-auth-d05"></a>

- [x] identity-auth-D05 以文档型决策卡把 ADR-0030 修订至最新串（当前为 `adr-0030-r12`，逐轮修订，引用按版本无关表述），给出生产 CAS JournalHeadAnchor provider 的产品选型、凭据来源、独立故障域论证与部署/ACL 结论；不改 `cmd/`、`internal/`，不解除「首期不提供」。 完成@2026-11-02

- **目标**：按 ADR-0030 约束 7 与约束 9(a) 产出并收口 ADR-0030 最新修订（当前为 `adr-0030-r12`，引用按版本无关表述）：对生产 CAS JournalHeadAnchor provider 给出产品与部署形态、凭据来源与最小权限/ACL、与业务 PostgreSQL 独立故障域的可核论证、部署拓扑与运维责任四项结论；结论可以继续是「首期不提供」，但必须逐项回答并保留证据，不得以确定性测试替身、空 registry 或环境变量存在冒充生产可用。本卡只改文档与作者证据，不改任何 Go 源码。
- **编号/类别理由**：B13J 决策序列已用 D01–D04 承接 D02/D04 的 ADR-0030/0031/0032 决策；本卡是 ADR-0030 的后续生产 provider 决策，续编 `D05`。类别为文档型决策卡：只产出 ADR 修订与作者证据，不含生产实现代码，符合 ADR-0030 约束 9(a)。
- **类型/Story与范围**：决策文档（identity Story S04 的恢复信任根/生产准入前置；范围仅 ADR-0030 最新修订（登记时 r11；完成时已修订至 `adr-0030-r12`）与作者证据）。不覆盖 S4.23 实现、bundle 注册、真实独立故障域实测或生产解除；不重开 S4.2/S4.5 已闭范围。
- **精确文件清单/写集（2 文件）**：
  - `.add/project/knowledge/decisions/framework.md`（修改：ADR-0030 正文修订至最新串（当前为 `adr-0030-r12`，引用按版本无关表述），同步状态/决策/约束 7 与引用记录；不改 ADR-0031/0032 与其他 ADR）
  - `.spec/wip/recovery-runtime/evidence-b13j-d05/report.md`（新增：作者证据报告，含输入哈希、结论逐项、未验证边界）
  - 不修改 `cmd/**`、`internal/**`、`test/**`、`tasks.md` 或 Plan。
- **硬/软 requires**：硬 requires 无（决策卡可立即登记，ADR-0030 约束 9(a)）。软关联：S4.23 的启动门禁依赖本卡决策修订被架构与安全独立评审接受（ADR-0030 约束 9(b)），该接受不在本卡关闭时宣称。
- **验收命令（NOT RUN；期望每条 exit 0）**：
```text
rg -n '^<a id=adr-0030></a>$|^## ADR-0030:|修订.*adr-0030-r[0-9]+' .add/project/knowledge/decisions/framework.md  # exit 0（版本无关匹配，登记时实测 r11）
rg -n '产品|凭据|独立故障域|部署|ACL' .add/project/knowledge/decisions/framework.md  # exit 0
git diff --check -- .add/project/knowledge/decisions/framework.md  # exit 0
if (-not (Test-Path -LiteralPath '.spec/wip/recovery-runtime/evidence-b13j-d05/report.md')) { exit 1 }  # exit 0
```
- **负向测试与失败 oracle**：`Select-String -Path '.add/project/knowledge/decisions/framework.md' -Pattern '已解除.*首期不提供|首期不提供.*已解除'` 必须无命中；作者证据须记录改前/改后 `git status --porcelain -- cmd internal` 输出逐字节相等。oracle：出现「已解除」误述或本卡引入任何 `cmd`/`internal` 新条目即失败，本卡不得关闭。
- **关闭条件（6 项）**：(1) ADR-0030 修订串为本轮收口的最新串（当前为 `adr-0030-r12`），且与卡面登记口径一致；(2) 产品/凭据/独立故障域/部署与 ACL 四项均有可核结论或明确「首期不提供」并说明理由；(3) 约束 1–6、8–10 与「首期不提供」结论未被本卡擅自改写；(4) 作者证据报告含改前/改后哈希、字节、行数与引用行；(5) 写集外零差异；(6) 独立评审是否接受由 S4.23 启动门禁另行核验，本卡不宣称评审 PASS。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-d05/report.md`；ADR 修订本体在 `.add/project/knowledge/decisions/framework.md`。
- **责任人**：Arch-Designer（文档作者）+ 非作者 Reviewer（评审由 S4.23 门禁消费）；当前未派发/未认领。
- **不宣称边界**：不宣称生产 provider 已选型成功、独立故障域已验证、ACL 已实测、ADR 已被独立评审接受、生产 `enter`/`install` 已解锁或生产可用；不代替人的决策接纳、实施授权或 Git 写授权。
- **停止条件**：若四项结论任一无法从 ADR-0019/0030、design 或真实部署事实得出可核依据，停止并回报缺项；若需改 `cmd/**`、`internal/**` 或解除「首期不提供」，停止并回架构/用户授权；若需修改 ADR-0031/0032 或其他 ADR，停止并拆卡。
- **2026-11-02 关闭记录**（`[ ]`→`[x]`，完成@2026-11-02；Task-Planner 按 `task-execution-workflow.md` §10.3 核证）：产出 `adr-0030-r12`（`framework.md` 的 ADR-0030「修订」行 2026-11-02 实测为 `adr-0030-r12`；D05 收口时全文 247770 B / 1246 行 / `4970D171362E02AF4B189F5BD42B58F5E6EA9203F18857A51898CE64123003BC`，随后 D06 把 ADR-0031 升至 `adr-0031-r05`，全文变为 255592 B / 1260 行 / `1CF9753650CB46CF171F77B330D526A4AB76D430DF8B2B630EE57E06E85B5977`；framework.md 行号随 ADR 修订漂移，引用以段名锚为准），逐轮 r05→r12；约束 1–10 正文未改，L1043–1057 段指纹 `6216:D2925D5F78A9C37FB4C8D565B64F7AFCBE68C9C5FAA3653A2702789D944B4B99`（Lead 与两路 Reviewer 各自独立复现，本卡关闭前复算一致）。独立评审：`review-arch-r07` PASS WITH CONDITIONS、`review-sec-r07` PASS WITH MINOR；两路合取后 ADR-0030 约束 9(b) 的评审门禁成立，r06 的 HIGH/MEDIUM 全部 CLOSED。明确未满足：约束 9(b) 启动前置 (i)–(v) 中的 (iii) 许可与具名 owner、(iv) 来源地址范围、(v) 见证 owner 与保证等级取证仍为 unknown/待登记，故 `identity-auth-S4.23`/`identity-auth-S4.24a`/`identity-auth-S4.24b`/`identity-auth-S4.22` 仍不得启动或认领。人的成果接收：按 `tasks.md:20` 的 2026-10-31 项目级验证授权（AI 可代理评审签收）由 AI 代理签收代行，真实人类签字仍为 unknown。边界：不构成生产批准、不解除 `blocked-upstream`、不改变「首期不提供」结论。另注：卡体 `:3045`「登记时 r11」为登记时点事实、非过期断言；关闭时（2026-11-02）授权只含 checkbox/关闭记录/统计行/Plan 对应处故未改，已于 2026-11-02 后续轮次以括注澄清（`:3045` 补「完成时已修订至 `adr-0030-r12`」，保留“登记时”原义；本句原引 `:3043` 为行号笔误，一并更正为 `:3045`）。

<a id="task-identity-auth-d06"></a>

- [x] identity-auth-D06 以文档型决策卡把 ADR-0031 修订至 r06，给出批准主体、执行主体、宿主隔离与原生耐久回执四项可核结论；不改 `cmd/`、`internal/`，不解除「首期不提供生产 restore 工具与批准主体」。 完成@2026-11-02

- **目标**：按 ADR-0031 不变量 1–6 与 ADR-0031「后续阶段前置节点」段产出 `adr-0031-r06`：逐项给出批准主体（角色/任命/独立批准记录）、执行主体（独立非交互子进程与专用执行身份）、宿主隔离（POSIX/Windows 矩阵、argv/进程树/fd/ENV 最小机制）与原生耐久回执（来源字段/独立读取/绑定 artifact digest/target/source installation）的可核结论或明确 unknown；不得把「约束 8 全项解除」当作本卡前提，也不得宣称生产替换可用。本卡须显式处理「登记≠启动」：在 `adr-0031-r06` 中说明既有 `identity-auth-S4.24a` 的登记在该口径下不违反 ADR-0031「后续阶段前置节点」段的实质目的（该段操作性禁令落在实施与启动；登记本身不产出 restore 工具、批准主体、执行身份或宿主隔离），或给出更严格的自洽措辞并说明理由。
- **编号/类别理由**：紧随 D05，承接 ADR-0031 的后续修订，续编 `D06`。类别为文档型决策卡：只产出 ADR 修订与作者证据，不含生产实现代码；S4.24a 的登记门禁消费本卡。
- **类型/Story与范围**：决策文档（identity Story S04 的恢复批准/隔离前置；范围仅 ADR-0031 r06 与作者证据）。不覆盖 S4.24a 实现、真实 restore 演练、ACL/进程隔离实测或生产解除；不重开 S4.14/S4.24a 范围。
- **精确文件清单/写集（2 文件）**：
  - `.add/project/knowledge/decisions/framework.md`（修改：ADR-0031 正文修订至 `adr-0031-r06`，同步状态/不变量/解除条件与引用记录；不改 ADR-0030/0032 与其他 ADR）
  - `.spec/wip/recovery-runtime/evidence-b13j-d06/report.md`（新增：作者证据报告，含四项结论、unknown 项、输入哈希与未验证边界）
  - 不修改 `cmd/**`、`internal/**`、`test/**`、`tasks.md` 或 Plan。
- **硬/软 requires**：硬 requires 无（决策卡可立即登记）。软关联：S4.24a 的起始门禁依赖本卡决策修订被架构与安全独立评审接受（ADR-0031「后续阶段前置节点」段），该接受不在本卡关闭时宣称。
- **验收命令（NOT RUN；期望每条 exit 0）**：
```text
rg --crlf -n '^\*\*目的\*\*:.*修订\*\*: adr-0031-r06$' .add/project/knowledge/decisions/framework.md  # exit 0
rg -n '批准主体|执行主体|宿主隔离|原生耐久回执' .add/project/knowledge/decisions/framework.md  # exit 0
git diff --check -- .add/project/knowledge/decisions/framework.md  # exit 0
if (-not (Test-Path -LiteralPath '.spec/wip/recovery-runtime/evidence-b13j-d06/report.md')) { exit 1 }  # exit 0
```
- **负向测试与失败 oracle**：`Select-String -Path '.add/project/knowledge/decisions/framework.md' -Pattern '已解除.*首期不提供|生产 restore 已可用'` 必须无命中；作者证据须记录改前/改后 `git status --porcelain -- cmd internal` 输出逐字节相等。oracle：出现解除误述或本卡引入任何 `cmd`/`internal` 新条目即失败，本卡不得关闭。
- **关闭条件（6 项）**：(1) ADR-0031 修订字符串为 `adr-0031-r06`；(2) 批准主体/执行主体/宿主隔离/原生耐久回执四项均有可核结论或明确 unknown 并说明缺证，且当前修订（r06）已显式处理「登记≠启动」：说明既有 `identity-auth-S4.24a` 的登记在该口径下不违反 ADR-0031「后续阶段前置节点」段的实质目的，或给出更严格的自洽措辞；(3) 不变量 1–6 与约束 8 的九条解除条件未被本卡擅自改写或宣称满足；(4) 作者证据报告含改前/改后哈希、字节、行数与引用行；(5) 写集外零差异；(6) 独立评审是否接受由 S4.24a 启动门禁另行核验，本卡不宣称评审 PASS。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-d06/report.md`；ADR 修订本体在 `.add/project/knowledge/decisions/framework.md`。独立评审：`review-arch-r02`（22211 B / `CE09A03B540F1258FCBCB1B83D768975DE112C49FF1CB10C1D9F1E01D48D5F45`，PASS WITH CONDITIONS）与 `review-sec-r02`（18286 B / `1C373557F597422127527E8544307E3FB4862E81F192A6F6DA54D6EEE72914BF`，PASS WITH CONDITIONS），路径 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r02/report.md`、`.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r02/report.md`；r01→r02 关系：两路 r01（`review-arch-r01` 19103 B / `ADC42326583EBAB6ABB9D4B14B8A08C12661F95A5E53276E2E4034144938BDD2` PASS WITH CONDITIONS；`review-sec-r01` 19208 B / `A4FFA36DF701CC9705563B3A1A415FE53C36A5857B6C7603E354BB2D90DDAABA` CHANGES REQUIRED）均以 r05 为对象并指出卡面与 ADR 修订串不一致，r06 修复后两路 r02 以 r06 为对象复评。
- **责任人**：Arch-Designer（文档作者）+ 非作者 Reviewer（评审由 S4.24a 门禁消费）；当前未派发/未认领。
- **不宣称边界**：不宣称恢复批准角色已任命、执行身份已实测、宿主隔离已验证、原生耐久回执已取得、ADR 已被独立评审接受、生产 restore/`enter`/`resume` 已解锁或生产可用；不代替人的决策接纳、实施授权或 Git 写授权。
- **停止条件**：若四项结论任一无法从 ADR-0031、design `:3880-3900` 或真实工具/部署事实得出可核依据，停止并回报缺项；若需改 `cmd/**`、`internal/**`、选定具体执行身份切换机制或解除「首期不提供」，停止并回架构/用户授权；若需修改 ADR-0030/0032 或其他 ADR，停止并拆卡。
- **2026-11-02 关闭记录**（`[ ]`→`[x]`，完成@2026-11-02；Task-Planner 按 `task-execution-workflow.md` §10.3 核证）：产出 `adr-0031-r06`（`framework.md` 的 ADR-0031「修订」行 2026-11-02 实测为 `adr-0031-r06`；改后全文 261179 B / 1269 行 / `8920FEA506AE98D872D5B8485934C741F937786A403E7B4820172BC139AF9C5C`；framework.md 行号随 ADR 修订漂移，引用以段名锚为准）。增量账目（Lead 复测）：改前 r05 基线 255592 B / 1260 行 / `1CF9753650CB46CF171F77B330D526A4AB76D430DF8B2B630EE57E06E85B5977`，改后 +5587 B / +9 行；作者声明的分段账目 0+136+497+728+1267+366+376+2217 = 5587 B 与实测增量一致；8 个 hunk 全部落在 ADR-0031 段内（`:1082`、`:1113`、`:1115`、`:1117–:1118`、`:1120`、`:1165→:1173`、`:1167→:1175`、`:1179` 后追加 `:1188`），未及 ADR-0030/0032；不变量 1–6（`L1130–1149` = `3714:8367B6A0…DAAF`）与约束 1–9（`L1150–1169`（不含尾空行），含约束 8 的九条解除条件）逐字节未变（该段指纹 `3500:01450818…50D2` 由安全侧 r02 独立重算）。两轮评审链（四份报告）：`review-arch-r01/report.md` 19103 B / `ADC42326583EBAB6ABB9D4B14B8A08C12661F95A5E53276E2E4034144938BDD2`（对象 r05，PASS WITH CONDITIONS）→ `review-sec-r01/report.md` 19208 B / `A4FFA36DF701CC9705563B3A1A415FE53C36A5857B6C7603E354BB2D90DDAABA`（对象 r05，CHANGES REQUIRED，9 条 finding）→ `review-arch-r02/report.md` 22211 B / `CE09A03B540F1258FCBCB1B83D768975DE112C49FF1CB10C1D9F1E01D48D5F45`（对象 r06，PASS WITH CONDITIONS）与 `review-sec-r02/report.md` 18286 B / `1C373557F597422127527E8544307E3FB4862E81F192A6F6DA54D6EEE72914BF`（对象 r06，PASS WITH CONDITIONS），路径均为 `.spec/wip/recovery-runtime/evidence-b13j-d06/<目录>/report.md`；两路合取后 ADR-0031「后续阶段前置节点」段的决策接受级门禁成立。r06 修复项对应：`RV-02`（「不得登记或启动」与「登记不构成需要本 ADR 先行」的字面矛盾）、`SEC-R06-F001`（启动门禁与证据产出主体自指死锁→决策接受级／卡证据级／生产解除级三级门禁，CLOSED）、`SEC-R06-F002`（回执抗伪造与外部见证，CLOSED 决策级）、`SEC-R06-F003`（`st_dev`/`st_ino`、固定 fd-exec、复核人独立，CLOSED）、`SEC-R06-F004`（ENV 动态加载与注入变量否证，CLOSED）、`SEC-R06-F005`（持久文本修复，保持 CLOSED）。四项 unknown 仍全未补齐：具名批准角色与两条批准记录、执行身份与切换机制、POSIX 隔离实测、产品级回执字段与真实样本，补证主体分别为部署方与承载实现卡，本轮未补齐任何一项。决策接受级的边界：只解除实施/验证卡的派发、认领与开始改动限制，不等于人的决策接纳、不产生任何人的批准、不解除生产 closed。OPEN 残余（Lead 决策：登记不改 ADR 正文）：安全侧 `B13J-D06-SEC-R02-F001`（新增判据未回填不变量 2/5/6 与解除条件 2/4/6，生产解除前必须处理）、`R02-F002`（未要求读取时验签与信任锚固定）、`R02-F003`/`R02-F005`；架构侧 `B13J-D06-R02-F002`/`F003`/`F004`；均列为后续修订（r07）候选。状态结论：生产 `enter`/`resume` 继续 closed，约束 8 全项未满足；`identity-auth-S4.24a` 的硬 requires 已满足但未被派发、未认领、未启动；`identity-auth-S4.23`/`identity-auth-S4.24b`/`identity-auth-S4.22` 仍受各自前置约束不得启动。人的成果接收：按 ADR-0028 与 `tasks.md:8`/`:20` 的 2026-10-31 项目级验证授权（AI 可代理评审签收）由 AI 代理签收代行，真实人类签字仍为 unknown。边界：卡面登记与关闭不构成生产批准或实施授权（评审结论以两路报告原文为准），不解除 `blocked-upstream`，不改变「首期不提供」结论。

<a id="task-identity-auth-d07"></a>

- [x] identity-auth-D07 以文档型决策卡把 ADR-0032 修订至 r05，把「`Exclusive` 离开 `recoveryClosedPort`」从解除顺序的候选 S4.23 节点移到候选 S4.24b 节点；不改 `cmd/`、`internal/`，不改 POSIX-only 与 Windows 生产 unsupported（退出码 2）结论。 完成@2026-11-02

- **目标**：在 `adr-0032-r03` 基础上产出 `adr-0032-r05`（r04 已由本卡完成，r05 为本卡补正修订：解除顺序自指——原独立列于候选 S4.24b 之前的「解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据」节点已并入候选 S4.24b 节点，使产出者与证据同节点）：(1) 修订行 `**修订**: adr-0032-r03` 改为 `adr-0032-r05`；(2) 解除条件第 8 条「解除顺序（前置链，须逐项可核且无环）」（段名锚：ADR-0032 解除条件第 8 条「解除顺序」）的候选 S4.23 节点改为「anchor adapter bundle 四端口齐备并登记；`Exclusive` 作为已登记但 closed 的端口接入 bundle」，删除「`Exclusive` 离开 `recoveryClosedPort`」；(3) 候选 S4.24b 节点（段名锚：ADR-0032 第 8 条候选 S4.24b 节点）改为「bundle 交付且四端口齐备（`Exclusive` 仍可为 closed）之后，由 S4.24b 交付 `Exclusive` 离开 `recoveryClosedPort`」，requires 边继续指向候选 S4.23；(4) 第 8 条末尾的无环说明（段名锚：ADR-0032 第 8 条末尾无环说明）同步改写并仍为有向无环链。本卡只改 ADR-0032 段与作者证据，不改任何 Go 源码。
- **编号/类别理由**：B13J 决策序列已用 D05（ADR-0030）与 D06（ADR-0031）承接两项后续修订，本卡承接 ADR-0032 的解除顺序修订，续编 `D07`。类别为文档型决策卡：只产出 ADR 修订与作者证据，不含生产实现代码。承接 `.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r09.md` §1「B. 修订 ADR-0032 解除顺序」与本轮 `.spec/wip/recovery-runtime/evidence-b13j-d07/lead-authorization-r01.md` §1/§4。
- **类型/Story与范围**：决策文档（identity Story S04 的恢复解除顺序前置；范围仅 ADR-0032「解除顺序」段、修订行、无环说明与作者证据）。不覆盖 S4.23/S4.24b 实现、锁后端代码、真实 ACL/POSIX 取证或生产解除；不重开 S4.20/S4.21 已闭范围。
- **精确文件清单/写集（2 文件）**：
  - `.add/project/knowledge/decisions/framework.md`（修改：仅 ADR-0032 段——修订行、解除条件第 8 条解除顺序的两个候选节点与无环说明、对应修订记录；不改 ADR-0019/0021/0029/0031，不改 ADR-0030 约束 1–10 正文）
  - `.spec/wip/recovery-runtime/evidence-b13j-d07/report.md`（新增：作者证据报告，含改前/改后哈希、字节、行数、逐项改动对照、反向还原命中与未验证边界）
  - 不修改 `cmd/**`、`internal/**`、`test/**`、`tasks.md` 或 Plan。
- **输入读取**：`framework.md` ADR-0032 全段（段名锚：ADR-0032 段头至 r05 追溯行）；ADR-0030 中已识别的跨 ADR 冲突记录（段名锚：ADR-0030 现行态陈述与 OPEN 清单）；`tasks.md` 的 S4.23/S4.24b/S4.22 卡面；`gap-map` 的候选 a/b 拆分口径；两份 Lead 授权记录的 §1 B/C 与 §4 语义约束。
- **硬/软 requires**：硬 requires 无（文档卡可立即登记）。软关联：`identity-auth-S4.24b` 的启动门禁消费本卡 `adr-0032-r05` 修订被架构与安全独立评审接受，该接受不在本卡关闭时宣称；本卡不解除 `identity-auth-S4.23` 对 `identity-auth-D05` 的硬 requires。
- **验收命令（NOT RUN；期望按行内注释的退出码）**：
```text
rg --crlf -n '\*\*修订\*\*: adr-0032-r05$' .add/project/knowledge/decisions/framework.md  # exit 0（文件为 CRLF，须 --crlf）
rg -n '^\s+- 候选 S4\.24b' .add/project/knowledge/decisions/framework.md  # exit 0
rg -n '^\s+- 候选 S4\.23.*离开 `recoveryClosedPort`' .add/project/knowledge/decisions/framework.md  # exit 1（反向 oracle：S4.23 节点不再携带「Exclusive 离开」）
git diff --check -- .add/project/knowledge/decisions/framework.md  # exit 0
if (-not (Test-Path -LiteralPath '.spec/wip/recovery-runtime/evidence-b13j-d07/report.md')) { exit 1 }  # exit 0
```
- **负向测试与失败 oracle**：`git status --porcelain -- cmd internal` 在改前/改后必须逐字节相等且为空；`Select-String -Path '.add/project/knowledge/decisions/framework.md' -Pattern 'adr-0032-r03'` 必须 0 命中；第 8 条内 POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、`O_CLOEXEC` 常驻 fd、三入口共锁、锁路径与 ACL 合同、`journalPath != lockPath` 必须逐字不变（作者证据须给前/后逐字对照与反向还原命中）。oracle：修订串未变、S4.23 节点仍携带「`Exclusive` 离开」、任一不变量被放宽或出现写集外改动即失败，本卡不得关闭。
- **关闭条件（6 项）**：(1) ADR-0032 修订字符串为 `adr-0032-r05`；(2) 第 8 条两节点口径已改：S4.23 只把 `Exclusive` 作为已登记但 closed 的端口接入，S4.24b 在「bundle 四端口齐备（`Exclusive` 仍可为 closed）」之后交付 `Exclusive` 离开 `recoveryClosedPort`；(3) POSIX-only、Windows 生产 unsupported（退出码 2）、无 fallback、`O_CLOEXEC` 常驻 fd、三入口共锁、锁路径与 ACL 合同逐字不变；(4) 作者证据含改前/改后哈希、字节、行数与反向还原命中（按本卡改动清单把改后文本还原后哈希与改前一致）；(5) 写集外零差异，未修改 `cmd/**`、`internal/**`、`tasks.md` 或 Plan；(6) 不宣称评审 PASS 或人的接收。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-d07/report.md`；ADR 修订本体在 `.add/project/knowledge/decisions/framework.md`。
- **责任人**：Arch-Designer（文档作者）+ 非作者 Reviewer（评审由 S4.24b 门禁消费）；当前未派发/未认领。
- **不宣称边界**：不宣称 ADR-0032 已被架构或安全独立评审接受、S4.23/S4.24b/S4.22 任一实现已解锁、POSIX 锁后端已实现、ACL 已实测、生产 `enter`/`install`/`normal`/`recovery`/`requeue` 已解锁或生产可用；不解除任何「首期不提供」与既有 `blocked-upstream`；不代替人的决策接纳、实施授权或 Git 写授权。
- **停止条件**：若第 8 条两节点口径无法在不改 POSIX-only/Windows unsupported/无 fallback/`O_CLOEXEC`/三入口共锁/路径与 ACL 合同的前提下改写，停止并回报冲突；若需改 `cmd/**`、`internal/**`、ADR-0030 约束 1–10 正文或 ADR-0019/0021/0029/0031，停止并回架构/用户授权；若需与 D05 并发写 `framework.md`，停止并等待 D05 落地复核。
- **2026-11-02 关闭记录**（`[ ]`→`[x]`，完成@2026-11-02；Task-Planner 按 `task-execution-workflow.md` §10.3 核证）：产出 `adr-0032-r05`（`framework.md` 的 ADR-0032「修订」行 2026-11-02 实测为 `adr-0032-r05`；行号随 ADR 修订漂移，引用以段名锚为准），r04 与 r05 均在本卡内完成；解除顺序自指已消除：原独立列于候选 S4.24b 之前的「解除条件 1–3 的 POSIX 锁后端与 ACL/负向证据」节点已并入候选 S4.24b 节点（ADR-0032「解除顺序」第 8 条候选 S4.24b 节点，2026-11-02 实测 `framework.md:1240`），证据不再早于产出者。证据落点：`.spec/wip/recovery-runtime/evidence-b13j-d07/report.md` §12（r05 段）+ `.spec/wip/recovery-runtime/evidence-b13j-d05/report.md` §22。作者归因（架构 `R07-F004`）：本卡 r05 的 ADR 文本由 Arch-Designer 产出；本轮 `tasks.md` 与 Plan 中 D05/D07 的 checkbox、关闭记录、统计计数与对应引用变更由 Task-Planner 在本卡关闭时同步完成，不改变本卡写集与关闭条件。独立评审：`review-arch-r07` PASS WITH CONDITIONS、`review-sec-r07` PASS WITH MINOR。人的成果接收同 D05 口径（`tasks.md:20` 项目级授权，AI 代理签收代行，真实人类签字 unknown）。边界：不构成生产批准、不启动任何候选卡、不改变 POSIX-only 与 Windows 生产 unsupported（退出码 2）结论。

<a id="task-identity-auth-d08"></a>

- [x] identity-auth-D08 以文档型决策卡把 ADR-0031 修订至 r07，回填安全侧 `R02-F001`/`R02-F002` 与架构侧 `R02-F003`/`R02-F004`；不改 `cmd/`、`internal/`，不解除任何「首期不提供」与生产 closed。 完成@2026-11-02

- **目标**：在 `adr-0031-r06` 基础上产出 `adr-0031-r07`，只改 ADR-0031 段与作者证据：(1) 修订行 `**修订**: adr-0031-r06` 改为 `adr-0031-r07`；(2) 把四项决策级判据（`st_dev`/`st_ino`、固定 fd-exec、复核人独立、ENV 封闭字面集与动态加载/注入变量 denylist、回执抗伪造与外部见证）回填到不变量 2/5/6 与解除条件 2/4/6，使「卡证据级／生产解除级」同约束 8 各解除条件逐项可对应，且不回退任何既有要求、不新增可选性（安全 `R02-F001`）；(3) 原生耐久回执项补「读取时须先验签并固定信任锚；信任锚不得由被验证回执自身或 root 自选；验签失败或信任锚不可固定即 closed」，并同步不变量 6 与解除条件第 6 条（安全 `R02-F002`）；(4) 按 r06 已被架构与安全两路 r02 接受的事实更新状态段（决策接受级已达成），r05/r06 追溯行中已失效的时点陈述只允许在行末追加「已被后续修订取代」标注，不删历史正文（安全 `R02-F003`、架构 `R02-F003`）；(5) 补 ADR-0030 约束 9(b)「决策修订被接受／约束全项解除」两级与本 ADR「决策接受级／卡证据级／生产解除级」三级的显式映射（架构 `R02-F004`）；(6) 新增 r07 追溯行，逐项列出改动段落与未改段落。
- **编号/类别理由**：B13J 决策序列已用 D05（ADR-0030）、D06（ADR-0031 r05/r06）、D07（ADR-0032）承接修订，本卡承接 ADR-0031 的 r07 回填，续编 `D08`。类别为文档型决策卡：只产出 ADR 修订与作者证据，不含生产实现代码。
- **类型/Story与范围**：决策文档（identity Story S04 的恢复解除门禁前置；范围仅 ADR-0031 段与作者证据）。不覆盖 S4.24a 实现、restore 工具、批准角色任命、隔离实测或生产解除；不重开 D06 已闭范围。
- **精确文件清单/写集（2 文件）**：
  - `.add/project/knowledge/decisions/framework.md`（修改：仅 ADR-0031 段——修订行、第 4 项回执判据、不变量 2/5/6、解除条件 2/4/6、状态段、两级/三级门禁映射、r05/r06 追溯行行末标注、新增 r07 追溯行；不改 ADR-0030 约束 1–10 正文，不改 ADR-0019/0021/0027/0029/0032）
  - `.spec/wip/recovery-runtime/evidence-b13j-d08/report.md`（新增：作者证据报告，含改前/改后哈希、字节、行数、逐 hunk 对照、分段指纹与未验证边界）
  - 不修改 `cmd/**`、`internal/**`、`test/**`、`tasks.md` 或 Plan。
- **输入读取**：`framework.md` ADR-0031 全段（段名锚：ADR-0031 段头至 r06 追溯行）；D06 两路 r02 报告（`review-arch-r02`、`review-sec-r02`）；ADR-0030 约束 9(b) 与本 ADR 现行态陈述中 `:1023`/`:1039` 的 r05 承接现状句（只读，禁改）。
- **硬/软 requires**：硬 requires `identity-auth-D06`（r06 已由架构与安全两路 r02 接受）。软关联 `identity-auth-S4.24a`：本卡 r07 一旦产出但未获两路接受，S4.24a 的启动门禁回到 blocked，r07 被两路接受后恢复；该接受不在本卡关闭时宣称。
- **验收命令（NOT RUN；期望按行内注释的退出码）**：
```text
rg --crlf -n '^\*\*目的\*\*:.*修订\*\*: adr-0031-r07$' .add/project/knowledge/decisions/framework.md  # exit 0
rg -n '^\*\*目的\*\*:.*修订\*\*: adr-0031-r06$' .add/project/knowledge/decisions/framework.md  # exit 1（反向 oracle：现行修订行不再是 r06）
git diff --check -- .add/project/knowledge/decisions/framework.md  # exit 0
if (-not (Test-Path -LiteralPath '.spec/wip/recovery-runtime/evidence-b13j-d08/report.md')) { exit 1 }  # exit 0
```
- **负向测试与失败 oracle**：`git status --porcelain -- cmd internal` 改前/改后逐字节相等；分段指纹核对——不变量 1/3/4、约束 1–9（含约束 8 九条）、解除条件 1/3/5/7/8/9、ADR-0030、ADR-0032、ADR-0019/0021/0027/0029 全段逐字节不变；回填不得把任何既有要求降级为「建议/可选」。oracle：修订串未变、任一禁改段落出现差异、或既有判据被放宽即失败，本卡不得关闭。
- **关闭条件（6 项）**：(1) 修订串为 `adr-0031-r07`；(2) 四项决策级判据已回填不变量 2/5/6 与解除条件 2/4/6，且与约束 8 逐项可对应；(3) 回执项含读取时验签与固定信任锚，且有 fail-closed 否证条件；(4) 状态段与 r05/r06 追溯行的失效时点陈述已处理（行末标注，历史正文未删）；(5) 两级/三级门禁映射已写入 ADR-0031 内部；(6) 作者证据含改前/改后哈希、字节、行数、逐 hunk 对照与分段指纹，写集外零差异，不宣称评审 PASS 或人的接收。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-d08/report.md`；ADR 修订本体在 `.add/project/knowledge/decisions/framework.md`。
- **责任人**：Arch-Designer（文档作者）＋ 非作者 Reviewer（架构与安全两路，报告落 `evidence-b13j-d08/review-arch-r01/`、`review-sec-r01/`）；当前未派发/未认领。
- **不宣称边界**：不宣称 r07 已被接受、S4.24a 已解锁、生产 closed 已解除、四项 unknown 已补齐；ADR-0030 `:1023`/`:1039` 的 r05 承接字样属禁改区，继续登记为已知残余；不代替人的决策接纳、实施授权或 Git 写授权。
- **停止条件**：若回填必须修改 ADR-0030/0032 或其他 ADR 正文、必须改 `cmd/**`/`internal/**`、或必须放宽既有判据，停止并回报冲突。
- **2026-11-02 关闭记录**（`[ ]`→`[x]`，完成@2026-11-02；Task-Planner 按 `task-execution-workflow.md` §10.3 核证）：产出 `adr-0031-r07`（`framework.md` 的 ADR-0031「修订」行 2026-11-02 实测为 `adr-0031-r07`；`framework.md` 268790 B / 1271 行 / SHA256 `70CFC915DBFBC27C62B078556C3F424BEA41F3F2667950A3F4A562B1FEC8AFCA`），核心改动：不变量 2/5/6 与解除条件第 2/4/6 条回填四项决策级判据（`st_dev`/`st_ino`、固定 fd-exec、复核人独立、ENV 封闭字面集与动态加载/注入变量 denylist、回执抗伪造与外部见证）；原生耐久回执项补「读取时须先验签并固定信任锚」，信任锚固定在批准记录内；状态段条件化；补 ADR-0030 约束 9(b) 两级门禁与三级（决策接受级／卡证据级／生产解除级）显式映射；r05/r06 追溯行行末追加失效标注（历史正文未删）；新增 r07 追溯行。证据落点：`.spec/wip/recovery-runtime/evidence-b13j-d08/report.md`（作者证据）。独立评审（两路均在 `.spec/wip/recovery-runtime/evidence-b13j-d08/`）：`review-arch-r01/report.md` 9964 B / 115 行 / SHA256 `EC9361004EF31091DC51F2B3F272264C8DC98544D8C42B2F482D18E5BE656CCB` / PASS WITH CONDITIONS（`B13J-D06-R02-F003`、`F004` CLOSED；`F002` ADR-0030 禁改区与 `F005` 记 OPEN；本轮新增 finding 0）；`review-sec-r01/report.md` 18884 B / 162 行 / SHA256 `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC` / PASS WITH CONDITIONS（`SEC-R02-F001`/`F002`/`F003` CLOSED、`F005` CLOSED WITH RESIDUAL；新增 `D08-SEC-R01-F001` MEDIUM 与 `F002`/`F003`/`F004` LOW）。作者归因：本卡 r07 的 ADR 文本由 Arch-Designer 产出；本轮 `tasks.md` 关闭记录、统计计数与 S4.24a 卡面/Plan 引用变更由 Task-Planner 在本卡关闭时同步完成，不改变本卡写集与关闭条件。人的成果接收：按 `tasks.md:20` 的 2026-10-31 项目级验证授权由 AI 代理评审签收代行，真实人类签字 unknown。边界：决策接受级对 r07 成立不解除生产 closed、不补齐四项 unknown（具名批准角色与两条批准记录、执行身份与切换机制、POSIX 隔离实测、产品级回执字段与真实样本）、不解锁任何实施卡、不构成生产批准、不改变「首期不提供」结论（ADR-0030 `:1023`/`:1039` 的 r05 承接字样属禁改区已知残余）；`identity-auth-S4.24a` 仍未派发/未认领/未启动。ADR-0031 状态段（`framework.md:1129`）的『r07 尚未复评』为 r07 产出时点表述；r07 已于 2026-11-02 被两路 r01 接受，现行状态以本关闭记录与两路 r01 报告为准；下一次 ADR-0031 授权修订须把该段去时点化，本轮不改 ADR 正文、不升版本。

<a id="task-identity-auth-s4-23"></a>

- [x] identity-auth-S4.23 生产实现并注册 anchor/CAS adapter，使同一 bundle 的 `Commitments`/`Sources`/`Anchor`/`Exclusive` 四端口齐备且成为唯一登记来源；受 ADR-0030 约束 9(b) 门禁。 完成@2026-11-05
- **2026-11-05 关闭（经决策不纳入本版，ADR-0037）**：adapter 与 registry 注册未实现，生产 registry 仍为空并 fail-closed（NOT RUN）；按 ADR-0037 关闭，不声称四端口 bundle 已交付；ADR-0030 约束不变。

- **目标**：把 `newRecoveryAnchorRegistry(nil)` 的空 registry 替换为按已登记 adapter ID 与受保护绝对配置路径解析的生产 registry；实现 anchor/CAS adapter 的 `Commitments`/`Sources`/`Anchor`/`Exclusive` 四端口，并在同一 bundle 内绑定；任一端缺失、ID 非精确登记、路径非绝对/不可用或未登记时四端口一起落 `recoveryClosedPort`，不得 file/SQL/fake/环境探测 fallback；不得以 `recoveryRootInputs.exclusive` 合成注入替代 registry 来源。`Exclusive` 在本卡只作为「已登记但 closed」的端口接入 bundle，不得填入实现、不得使其离开 `recoveryClosedPort`（其离开 closed 由 `identity-auth-S4.24b` 交付完成）。生产默认未配置时仍 closed。
- **编号/类别理由**：沿用 gap-map `:178`/`:182` 与 ADR-0030 约束 9/10、ADR-0032「解除顺序（前置链，须逐项可核且无环）」段的候选 `S4.23`（anchor/CAS adapter bundle；引用按版本无关表述，段名锚：ADR-0032 第 8 条候选 S4.23 节点）；本轮首次登记为正式 Task，类别为实施卡。
- **类型/Story与范围**：开发（identity Story S04 的恢复 anchor 来源；同卡局部实现与集成测试）。范围为 adapter 生产实现、registry 注册与四端口 bundle 唯一来源；不覆盖真实独立故障域验证、真实 provider 凭据部署、生产解除或 S4.22 root 装配。
- **精确文件清单/写集（4 文件）**：
  - `cmd/han-server/recovery_sources.go`（修改：生产 adapter 实现/注册与四端口 bundle 校验；保持 `recoveryClosedPort` 的 fail-closed 语义）
  - `cmd/han-server/recovery_runtime.go`（修改：`newRecoveryAnchorRegistry(nil)` 改为真实 registry 解析；未配置仍 closed）
  - `cmd/han-server/main.go`（修改：仅必要的 production root 依赖接线；不新增第二 router/authority）
  - `cmd/han-server/recovery_anchor_production_integration_test.go`（新增；`//go:build integration`；顶层 `TestRecoveryAnchorAdapterProductionBundle`，覆盖四端口齐备、唯一来源与负向拒绝）
  - 证据报告 `.spec/wip/recovery-runtime/evidence-b13j-s423/report.md` 为非代码产物，不占 4 文件写集，由实施者在关闭前创建。
- **硬/软 requires**：硬 requires `identity-auth-D05`——ADR-0030 决策修订被架构与安全独立评审接受（约束 9(b)），**不是**「约束 7 全项解除」。软关联：`identity-auth-S4.24b` 消费本卡的 bundle 与 `Exclusive`；不将 S4.24b 计为硬前置。
- **验收命令（NOT RUN；期望每条 exit 0）**：
```text
gofmt -l cmd/han-server/recovery_sources.go cmd/han-server/recovery_runtime.go cmd/han-server/main.go cmd/han-server/recovery_anchor_production_integration_test.go  # exit 0，输出为空
go build ./...  # exit 0
go vet ./cmd/han-server  # exit 0
go vet -tags=integration ./cmd/han-server  # exit 0
go test ./cmd/han-server -run '^(TestRecoveryAnchorAdapterRegistry|TestRecoveryRuntimeComposition)$' -count=1 -json  # exit 0
go test -race ./cmd/han-server -run '^(TestRecoveryAnchorAdapterRegistry|TestRecoveryRuntimeComposition)$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryAnchorAdapterProductionBundle$' -count=1 -json  # exit 0
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryAnchorAdapterProductionBundle$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryConfig$' -count=1 -json  # exit 0
```
- **负向测试与失败 oracle**：`TestRecoveryAnchorAdapterProductionBundle` 至少覆盖：(a) 四端口中缺任一 → `ErrAdmissionBinding`，四端口一起 closed；(b) adapter ID 非精确登记、配置路径相对/不存在/不可读 → 四端口一起 closed；(c) 无 registry 之外第二来源（file/SQL/fake/env）可打开 `Exclusive`；(d) 未配置 `HAN_RECOVERY_ANCHOR_ADAPTER`/`HAN_RECOVERY_ANCHOR_CONFIG_PATH` 时生产 root 仍 closed。(e) 见证核对不变量：见证不可读/不可写、事件表存在 revision > `w` 的终结行、revision ≤ `w` 存在未终结行且无预留，均须在 adapter 启动与每次推进前落 closed；有预留 `r` 且 `r` 上已有终结行时只允许幂等完成（把 `w` 推进到 `r` 并清预留），不得重写终结行、不得用观测 head 抬高见证。(f) 凭据/TLS/端点合同（ADR-0030 `(b)`），每条用例须断言具体错误分类（`ErrAdmissionClosed` 或 `ErrAdmissionBinding`）与各自可区分的子测试名，不得只断言错误非 nil 或只断言 exit 非 0：(f1) 见证端点凭据缺失、过期或主体错误 → closed 且拒绝推进；(f2) TLS 校验失败（自签 CA、主机名不匹配、证书过期）→ closed，且不得降级为明文、不得跳过校验、不得接受 `sslmode=require|prefer|allow|disable`；(f3) 运行期见证端点与批准记录登记的端点不一致 → closed（子测试名 `WitnessEndpointMismatch`）；(f4) `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 指向文件缺 witness 段，或 witness 段 CA/主机名与批准记录不一致 → closed（子测试名 `WitnessConfigMissingOrMismatch`）；(f5) 口令出现在 environ、argv、配置文件明文或日志 → closed（子测试名 `WitnessSecretLeak`）；(f6) 见证双通道必须互异——两个一次性 fd（3/4 或等价通道）、不同保管主体与不同秘密库路径，复用同一通道、同一端点或同一凭据来源即失败并落 closed（子测试名 `WitnessChannelReuse`）；(f7) 不得复用业务 PostgreSQL 的 CA、DSN、用户名或凭据链，复用即以 `ErrAdmissionBinding` 失败并落 closed（子测试名 `WitnessCredentialChainReuse`）；(f8) `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 必须 owner-only 且 no-follow——软链接、组/其他可读、相对路径、父目录非 owner-only 即失败并落 closed（子测试名 `WitnessConfigPathInsecure`）。oracle：任一端单独打开、任一路径被推断或出现 fallback、见证核对任一分支未落 closed、用观测 head 抬高见证、任一 (f) 用例仅以错误非 nil/exit 非 0 判定而无具体分类与子测试名、(f6)–(f8) 的通道/凭据链复用或路径权限不足未被拒绝、或 TLS 失败被降级为明文即测试失败，本卡不得关闭。
- **关闭条件（9 项）**：(1) `identity-auth-D05` 的接受证据可核并指向本卡启动门禁；(2) 同一 bundle 四端口齐备且为唯一登记来源；(3) 缺一/未登记/路径不可用时四端口一起 closed；(4) 生产未配置仍 closed，无隐式默认或 fallback；(5) 上述 9 条验收命令与负向测试通过；(6) `cmd/han-server/recovery_runtime.go` 不再以 `newRecoveryAnchorRegistry(nil)` 作为生产 registry；(7) 非作者 CodeReview 与 Main 同 revision 范围核证完成；(8) `cmd/han-server/recovery_sources.go` 的 Anchor 端口在 adapter 启动与每次推进前实现 ADR-0030「见证状态机（唯一合法形态）」段的 `(w, r, token)` 不变量核对（含 (i)–(vi) 分支、幂等完成、不得用观测 head 抬高见证），并由负向 (e) 覆盖并通过；见证形态选型、具名 owner、保证等级取证与真实部署演练仍按 D05 授权 C.5 在 S4.22/部署证据阶段承接；(9) 负向 (f) 组通过：见证端点凭据缺失/过期/主体错误、TLS 校验失败（自签/主机名不匹配/过期）、运行期端点与批准记录不一致、witness 段缺失或 CA/主机名不一致、口令泄漏至 environ/argv/配置文件/日志，均落 closed 且错误形态可区分；另须证明未降级为明文、未跳过 TLS 校验、未新增配置键，witness 段解析落在 `cmd/han-server/recovery_sources.go` 且 `cmd/han-server/recovery_config.go` 零改动；(f6)–(f8) 通过且证据含双通道互异、凭据链不复用、配置路径 owner-only/no-follow 的可核结果；ACL/GRANT 矩阵的部署证据须在关闭前提交——以 `has_table_privilege` 等权限查询（非人工声明）断言 `UPDATE`/`DELETE`/`DDL`/`TRUNCATE` 被拒、无跨库与超级用户路径。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-s423/report.md`（实施者创建，含命令日志、JSON 摘要、哈希与未验证边界）。
- **责任人**：Backend-Dev + 非作者 Reviewer；当前未派发/未认领。
- **不宣称边界**：不宣称真实 CAS provider、凭据、独立故障域、ACL 或生产 deploy 已验证；不宣称 `Exclusive` 已离开 `recoveryClosedPort`、生产 `enter`/`install` 已解锁、S4.24b/S4.22 已完成、Case18/S8.2 PASS 或生产可用；不代替独立评审、人的业务签收或 Git 写授权。
- **范围声明（承接 D05 授权 C.5；Lead 建议值、非验证）**：本卡**必须**在 adapter 启动与每次推进前实现 ADR-0030「见证状态机（唯一合法形态）」段的 `(w, r, token)` 不变量核对——(i)–(vi) 分支、幂等完成（`w` 只推进到预留值 `r`，不得跳号/回退）、不得用观测 head 抬高见证、见证不可读或不可写即 closed，并给出对应负向测试（见负向 (e) 与关闭条件 (8)）。**不在本卡范围**的仅是见证的形态选型、具名 owner、保证等级取证与真实部署演练（即 ADR-0030 约束 9(b) 启动前置 (v) 的取值与取证），该取值的权威登记点在 ADR-0030「启动前置 (i)–(v) 建议值」段 (v)（r10/r11/r12 均未改 (v) 的取值口径），`.spec/wip/recovery-runtime/evidence-b13j-d05/lead-authorization-r09.md` §1「C. 产品与环境五项建议值」第 5 项仅为本卡登记时的来源记录；承接仍在 `identity-auth-S4.22` 与部署证据阶段。**witness 段配置解析归属本卡**：`HAN_RECOVERY_ANCHOR_CONFIG_PATH` 指向配置文件的 witness 段（见证端点、来源地址范围、TLS CA 与主机名）由本卡在 `cmd/han-server/recovery_sources.go` 内解析并与批准记录比对；不新增配置键，`cmd/han-server/recovery_config.go` 不在本卡写集、不得修改（理由：该文件只读取 env 键、不解析配置文件内容；改它既越出本卡写集，也与「不得新增配置键」相冲突）。本条不改变本卡写集与 requires。
- **停止条件**：若 ADR-0030 决策修订未被架构与安全独立评审接受，停止（不得以约束 7 全项未满足为由绕开或提前实施）；若 design/ADR 无法给出 adapter ID、配置路径、四端口合同或错误分类，停止并回报缺项；若需新增配置键、改 `internal/identity/**`/`internal/article/**`、建第二 registry/authority 或引入 fallback，停止并回设计。

<a id="task-identity-auth-s4-24a"></a>

- [x] identity-auth-S4.24a 接入生产 restore source/target/tool 与批准主体接线；受 ADR-0031 门禁，与锁后端节点独立。 完成@2026-11-05
- **2026-11-05 关闭（经决策不纳入本版，ADR-0037）**：restore source/target/tool 与批准主体接线未实现，42 文件写集未动（NOT RUN）；按 ADR-0037 关闭，不声称 restore 可用；生产 restore 保持 closed。

- **目标**：在唯一 production root 内实现并接线 restore 的生产 source/target/tool：按 ADR-0031 不变量 1–6 校验备份来源批准、可执行文件批准、专用执行身份、宿主隔离与原生耐久回执；`restorePlanWire`/`restoreReceiptWire` 保持 strict v1 合同，生产 driver 只执行配置中唯一批准程序，固定 argv `restore-v1`/`verify-v1`、无 shell、私有 stdin、最小 ENV、进程树真实 join/reap；批准记录或工具缺失时在 Begin 前 closed，不得以 stdout receipt、exit=0 或 DB 可连接自证。生产解除仍须 ADR-0031 约束 8 全项与人的明确生产批准。
- **编号/类别理由**：沿用 gap-map 拆分后的候选 `S4.24a`（restore 分支），与 `S4.24b` 锁后端节点独立；本轮首次登记为正式 Task，类别为实施卡。
- **类型/Story与范围**：开发（identity Story S04 的恢复 DB 替换前置；同卡局部实现与集成测试）。范围为生产 restore source/target/tool 与批准主体接线；不覆盖真实旧备份演练、真实批准角色任命、POSIX 宿主隔离实测、锁后端或 S4.22 root 解除。
- **精确文件清单/写集（42 文件＝23 修改＋19 新增；2026-11-02 对账至 design r72；来源 `:4005`、`:4006`、`:4011`、`:4012`、`:4015`、`:4016`、`:4019`、`:4051`、`:4065`、`:4066`、`:4068`、`:4069`、`:4070`、`:4072`、`:4073`、`:4076`、`:4273`）**：
  - `cmd/han-server/recovery_restore.go`（修改：生产 restore source/target/tool 与批准主体接线；B1/B3 驱动侧接线与 B2 消费点、strict v1 plan/receipt 与失败分类；删除 `restoreReceiptWire` 与 `Target.DSN`，`newControlledRestoreDriver` 增第五参数 `ownership`）
  - `cmd/han-server/recovery_runtime.go`（修改：唯一 production root 接线；`recoveryRootInputs.restore` 改为 `restoreDriverFactory func(recovery.ExclusiveLease) (recovery.RestoreDriver, error)`；`enter`/`resume` 一次 `Acquire` 后注入同一 `restoreLease`，并由 `closeRestoreLease` 恰好关闭一次；`:641` 的 `NewRuntime` 对 `install`/`normal`/`verify` 传显式 typed nil `recovery.ExclusiveLease(nil)`）
  - `cmd/han-server/main.go`（修改：仅必要的 production root 装配；`main.go:634` 不设 `recoveryRootInputs.restore`，依赖 `recovery_runtime.go:596`–`:599` 的 nil 工厂返回 `recoveryClosedRestoreDriver`；不新增第二 router/authority）
  - `internal/recovery/runtime.go`（修改：`NewRuntime(ctx, deps, restoreLease)` 唯一签名；`ownedExclusive`/`injectedExclusive` 双互斥字段与按动作访问器，`injectedExclusive` 永不关闭）
  - `cmd/han-server/recovery_restore_host_linux.go`（新增；`//go:build linux`；宿主隔离预检）
  - `cmd/han-server/recovery_restore_host_other.go`（新增；`//go:build !linux`；`ErrAdmissionUnsupported`，零状态变更）
  - `cmd/han-server/recovery_restore_exec_linux.go`（新增；`//go:build linux`；fd-exec 与进程树真实 join/reap）
  - `cmd/han-server/recovery_restore_exec_other.go`（新增；`//go:build !linux`；`ErrAdmissionUnsupported`，零状态变更）
  - `cmd/han-server/recovery_restore_production_integration_test.go`（新增；`//go:build integration`；顶层 `TestRecoveryRestoreProductionWiring`）
  - 既有测试（17 文件：15 文件承载 `NewRuntime` 的 37 个测试调用点，全仓 39 处／17 文件含定义 1＋生产 1；另 2 文件为 restore 工厂改造）：
    - `internal/recovery/runtime_test.go`（`NewRuntime` 11 处均传 `recoveryTestLease{}`；另 `:1658` 传显式 typed nil）
    - `internal/recovery/installation_test.go`（3 处，`recoveryTestLease{}`）
    - `internal/recovery/admission_test.go`（3 处，`recoveryTestLease{}`）
    - `internal/article/deletion_journal_anchor_test.go`（3 处，`recoveryTestLease{}`）
    - `internal/article/deletion_journal_snapshot_test.go`（1 处，`recoveryTestLease{}`）
    - `internal/article/replay_publication_test.go`（1 处，`recoveryTestLease{}`）
    - `internal/article/replay_publication_integration_test.go`（1 处，`recoveryTestLease{}`）
    - `internal/platform/recovery_guard_test.go`（1 处，`recoveryTestLease{}`）
    - `internal/identity/recovery_installation_test.go`（1 处，`recoveryTestLease{}`）
    - `internal/identity/recovery_execution_test.go`（1 处，`recoveryTestLease{}`）
    - `internal/identity/recovery_execution_postgres_test.go`（1 处，`recoveryTestLease{}`）
    - `cmd/han-server/recovery_workers_test.go`（2 处，`recoveryTestLease{}`）
    - `cmd/han-server/recovery_restore_test.go`（1 处；`:228` 补第三实参，按 r66.1/r66.4 固定 `f.world` 构造顺序并删除 `:209`/`:233` 的 `ownership` 字段赋值）
    - `cmd/han-server/recovery_production_chain_integration_test.go`（3 处；`:2999` 的 `restore: world.restore` 改为工厂闭包）
    - `test/integration/recovery/runtime_fixtures_test.go`（3 处，`recoveryTestLease{}`）
    - `cmd/han-server/recovery_enter_install_integration_test.go`（`:232` 的 `restore: s421Restore{}` 改为工厂闭包）
    - `cmd/han-server/recovery_runtime_test.go`（`:731` 不设 `restore`，依赖 nil 工厂返回 `recoveryClosedRestoreDriver`）
  - 证据报告 `.spec/wip/recovery-runtime/evidence-b13j-s424a/report.md` 为非代码产物，不占 42 文件写集，由实施者在关闭前创建。
  - **`F017-R1` 写集扩写（16 文件＝6 生产＋10 配套测试；依据 design `:4051` 的 r72.4 累积口径、`:4005`–`:4016` 生产文件表与 `:4019`／`:4273` 测试建议；2026-11-02 由 Task-Planner 决定并入本卡授权写集）**：r63.9.4／r64.12 建议写集为**累积口径**，r65.12／r66.4 只局部取代其所列职责写法、不废止其余增补项。前 6 行为生产文件、后 10 行为配套测试；「新增（待实施创建）」按 2026-11-02 仓库真实状态标注（仅 `recovery_config.go` 9,842 B 与 `recovery_config_test.go` 2,959 B 已存在，其余 15 文件均不存在）。
  | # | 路径 | 变更 | 职责（一句话） | 来源 |
  | --- | --- | --- | --- | --- |
  | 1 | `cmd/han-server/recovery_restore_approval.go` | 新增（待实施创建） | 信任根 v2、JCS、Ed25519、principal／appointment／organization 交叉、复核签名与 `receipt_anchors` | `:4005` |
  | 2 | `cmd/han-server/recovery_restore_target.go` | 新增（待实施创建） | 目标 DSN 文件边界、流式字节解析、`appendJSONString`、九处清零与 redactor | `:4006` |
  | 3 | `cmd/han-server/recovery_restore_receipt.go` | 新增（待实施创建） | 12 字段 `RestoreReceipt`、producer envelope、签名载荷、审计记录、验证方签名与 `audit_record_ref` | `:4011` |
  | 4 | `cmd/han-server/recovery_restore_witness.go` | 新增（待实施创建） | 外部见证客户端、`baseline_sequence` 基线与见证 head `sequence` 的原子推进；缺失即 closed | `:4012` |
  | 5 | `cmd/han-server/recovery_config.go` | 修改（既有 9,842 B） | 四个新键、`readAction`、`actionRequiresRestore`、配置存在性 | `:4015` |
  | 6 | `internal/recovery/restore_source.go` | 新增（待实施创建） | `RestoreSourceClaim` 导出形状与 `RestoreSourceVerifier`；不扩 `InstallationSourceVerifier`、不改 `RestorePlan` | `:4016` |
  | 7 | `cmd/han-server/recovery_restore_approval_test.go` | 新增（待实施创建） | 批准／信任根／交叉签字正向与负向 | `:4019` |
  | 8 | `cmd/han-server/recovery_restore_target_test.go` | 新增（待实施创建） | 目标 DSN 文件边界、canary 与九处清零 | `:4019` |
  | 9 | `cmd/han-server/recovery_restore_receipt_test.go` | 新增（待实施创建） | 回执 12 字段、签名载荷与审计记录 | `:4019` |
  | 10 | `cmd/han-server/recovery_restore_witness_test.go` | 新增（待实施创建） | 见证 head 原子推进与缺失即 closed | `:4019`／`:4051` |
  | 11 | `cmd/han-server/recovery_config_test.go` | 修改（既有 2,959 B） | 四个新键、`readAction`／`actionRequiresRestore` 与别名拒绝 | `:4019`／`:4273` |
  | 12 | `internal/recovery/restore_source_test.go` | 新增（待实施创建） | `RestoreSourceClaim`／`RestoreSourceVerifier` nil → closed | `:4019`／`:4273` |
  | 13 | `cmd/han-server/recovery_restore_host_linux_test.go` | 新增（待实施创建；`//go:build linux`） | 宿主隔离预检（`/etc/ld.so.preload`、`ld.so.cache`、RPATH／RUNPATH） | `:4019`／`:4070` B5 |
  | 14 | `cmd/han-server/recovery_restore_host_other_test.go` | 新增（待实施创建；`//go:build !linux`） | `ErrAdmissionUnsupported` 且零状态变更 | `:4019`／`:4070` B5 |
  | 15 | `cmd/han-server/recovery_restore_exec_linux_test.go` | 新增（待实施创建；`//go:build linux`） | fd-exec 与进程树真实 join／reap | `:4019`／`:4070` B5 |
  | 16 | `cmd/han-server/recovery_restore_exec_other_test.go` | 新增（待实施创建；`//go:build !linux`） | `ErrAdmissionUnsupported` 且零状态变更 | `:4019`／`:4070` B5 |
  - 并入后本卡写集共 42 文件（23 修改＋19 新增），与 r65.12／r66.4／B5 原 26 项无重叠；新增测试须与写集同批创建，未在 2026-11-02 仓库存在的路径一律标「新增（待实施创建）」。本次扩写只确定文件所有权边界，**写集扩写 ≠ 实施方案确定**，不解除 `F004`／`N005`／`SEC-F007` 等门禁。
- **硬/软 requires**：硬 requires `identity-auth-D06`——ADR-0031 决策修订被架构与安全独立评审接受（ADR-0031「后续阶段前置节点」段），不是「约束 8 全项解除」；硬 requires 边与语义不变。软关联：`identity-auth-S4.22` 的 root 装配可在 restore 保持 closed 时进行；不将 S4.22 计为硬前置。ADR 同步（2026-11-02，承接安全侧 `B13J-D06-SEC-R02-F006` 与 `D08-SEC-R01-F001`；本轮只同步登记，未改 ADR、未派发）：ADR-0031 现行修订为 `adr-0031-r12`（`framework.md` 297,412 B / 1,275 行 / `800B56898F946D8391B73F3C56089EA9F34BDF2870DACC3CC4942988B6D31D20`），2026-11-02 已取得两路独立评审接受（PASS）：架构侧 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r06/report.md` 19,778 B / 171 行 / `705806BABF4C65A352E9442D43AD5D1D060A135CCA9C195A89C959D7650163C7` / PASS，BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 0，开放项 0；安全侧 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r06/report.md` 25,883 B / 218 行 / `12D55EBCCA5282FF703BBAE10F42A5601429E290EF8B5F02C04368AB8BCDF08A` / PASS，BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 1（`ADR-S06-01`，OPEN，属 design 侧错误码一致性，安全影响为零）；接受登记 `.spec/wip/recovery-runtime/evidence-b13j-d06/acceptance-adr-0031-r12.md` 4,848 B / 46 行 / `F04E65668A80E9796FD7BEF057C2CBE92817FFC0D1CB9018D7E140A30368497D`。时间口径：`framework.md:1084` 与 `:1194` 仍写「`adr-0031-r12` 的两路接受尚未取得」，属产出时点表述；现行事实以上述登记与 `framework.md:1129`「随现行修订号自动生效」机制为准。r06／r07 的接受事实只作历史，不再作为「现行修订已接受」的依据。该接受只解除实施/验证卡的派发、认领与开始改动限制（`framework.md:1174`），不构成人的接纳、不补齐四项 unknown、不解除生产 `enter`／`resume` closed、不表示约束 8 全项满足、不使 `identity-auth-S4.24a` 可关闭；本卡曾于 2026-11-02 派发并 BLOCKED，未关闭；设计 r72 双复审已闭环但未通过 G0、未获实施授权。设计现状（2026-11-02）：现行 `identity-auth-design-ir01-r72`（`design.md` 1,110,477 B / 5,476 行 / `491A476E60636ADAA3F1610A07C78054ED4C9735F5AD42D60B76200A12C8414F`）；r71 基线 1,106,644 B / `168B614D3D72C573D96110A2350E4894AED37CB5E32947685D678773B379C0C5`，r72 增量 +3,833 B、仅改动 12 行（`:5`、`:38`、`:3944`、`:4031`、`:4051`、`:4062`、`:4070`、`:4072`、`:4083`、`:4133`、`:4135`、`:4377`）。作者证据 `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r12/report.md` 13,295 B / `677854EE758D1AFD680E81753817E977BBFAA342A67E3277F7FE3E367029A65A`、`design-revision-r12/hashes.txt` 7,246 B / `AEBE9F2BBEF512E4EECAC2E08E6AEB01726D79CBC19C94E63BFEED5FDB7CB436`（含 [10] 后缀段登记节）。设计双复审已闭环（2026-11-02）：架构 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r12/report.md` 21,717 B / 178 行 / `B82E55B5E3451A25B559E0BE0754913977EB740C5FEA7EA026213B306F209EB1`，原判 PASS WITH CONDITIONS（BLOCKER 0 / MEDIUM 0 / LOW 1＝`NEW-R12-01`），`review-arch-r12/addendum-new-r12-01.md` 9,899 B / 103 行 / `3AE4C900F2CAF5868FCBA1C1F52DE994EC44DED4D67514081731DCEF32F45CEF` 独立复算三段后缀与登记逐字相等并置 `NEW-R12-01` CLOSED，本路开放 finding 0；安全 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r12/report.md` 21,736 B / 272 行 / `A531F7F1F6E05D1D7AC06224E866C4A44A8F8FAF9BE7BD846692FF31BE894D5C`，PASS，开放项 0（`NEW-R11-01`／`NEW-R11-02`／`NEW-S11-01` 均 CLOSED）。设计双复审闭环**不等于**设计签字、不等于 G0、不等于实施授权；`F004`（ADR 同步待架构决策方）、`N005`（`decisions/framework.md` 状态段与该 requires 行冲突，范围外）、`SEC-F007` 与 Linux 侧能力 NOT RUN、ADR-0031 约束 8 未满足、生产 `enter`／`resume` closed 全部不变。
- **验收命令（NOT RUN；期望每条 exit 0）**：
```text
gofmt -l cmd/han-server/recovery_restore.go cmd/han-server/recovery_runtime.go cmd/han-server/main.go internal/recovery/runtime.go cmd/han-server/recovery_restore_host_linux.go cmd/han-server/recovery_restore_host_other.go cmd/han-server/recovery_restore_exec_linux.go cmd/han-server/recovery_restore_exec_other.go cmd/han-server/recovery_restore_production_integration_test.go internal/recovery/runtime_test.go internal/recovery/installation_test.go internal/recovery/admission_test.go internal/article/deletion_journal_anchor_test.go internal/article/deletion_journal_snapshot_test.go internal/article/replay_publication_test.go internal/article/replay_publication_integration_test.go internal/platform/recovery_guard_test.go internal/identity/recovery_installation_test.go internal/identity/recovery_execution_test.go internal/identity/recovery_execution_postgres_test.go cmd/han-server/recovery_workers_test.go cmd/han-server/recovery_restore_test.go cmd/han-server/recovery_production_chain_integration_test.go test/integration/recovery/runtime_fixtures_test.go cmd/han-server/recovery_enter_install_integration_test.go cmd/han-server/recovery_runtime_test.go cmd/han-server/recovery_restore_approval.go cmd/han-server/recovery_restore_target.go cmd/han-server/recovery_restore_receipt.go cmd/han-server/recovery_restore_witness.go cmd/han-server/recovery_config.go internal/recovery/restore_source.go cmd/han-server/recovery_restore_approval_test.go cmd/han-server/recovery_restore_target_test.go cmd/han-server/recovery_restore_receipt_test.go cmd/han-server/recovery_restore_witness_test.go cmd/han-server/recovery_config_test.go internal/recovery/restore_source_test.go cmd/han-server/recovery_restore_host_linux_test.go cmd/han-server/recovery_restore_host_other_test.go cmd/han-server/recovery_restore_exec_linux_test.go cmd/han-server/recovery_restore_exec_other_test.go  # exit 0，输出为空（42 文件＝写集全量）
go build ./...  # exit 0
go vet ./cmd/han-server  # exit 0
go vet -tags=integration ./cmd/han-server  # exit 0
go test ./cmd/han-server -run '^(TestRecoveryRestoreDriver|TestRecoveryRuntimeComposition)$' -count=1 -json  # exit 0
go test -race ./cmd/han-server -run '^(TestRecoveryRestoreDriver|TestRecoveryRuntimeComposition)$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryRestoreProductionWiring$' -count=1 -json  # exit 0
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryRestoreProductionWiring$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryConfig$' -count=1 -json  # exit 0
```
- **负向测试与失败 oracle**：`TestRecoveryRestoreProductionWiring` 至少覆盖：(a) 相对/非绝对 executable、SHA 不等于批准值、manifest 缺失或不可读 → Begin 前拒绝；(b) 备份来源批准 ref/digest 缺失或角色自批 → 拒绝；(c) Windows/平台不支持 → 任何状态变更前拒绝；(d) shell、附加 argv、stdout receipt 或 exit=0 不得被当作成功证明；(e) 回执验证若走外部见证或批准人核对分支，须定义与验签等价的核验操作与审计字段（至少见证端点身份、单调序列、读取时间、核验人/系统身份），并把该定义纳入关闭证据；(f) 宿主隔离预检须核对加载器配置文件（`/etc/ld.so.preload`、`ld.so.cache`）与 RPATH/RUNPATH 完整性，或采用静态链接等等价固定方式，并把核对结果纳入关闭证据。oracle：任一负向用例进入 Begin/Replace、写 DB 或返回成功即测试失败，本卡不得关闭。
- **关闭条件（7 项）**：(1) `identity-auth-D06` 的接受证据可核并指向本卡启动门禁——启动前须在证据报告首段登记两路 r01 报告的路径 + SHA256 + verdict（`review-arch-r01` 9964 B / `EC9361004EF31091DC51F2B3F272264C8DC98544D8C42B2F482D18E5BE656CCB` / PASS WITH CONDITIONS；`review-sec-r01` 18884 B / `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC` / PASS WITH CONDITIONS；报告位于 `.spec/wip/recovery-runtime/evidence-b13j-d08/`），缺此即停止；(2) 生产 restore source/target/tool 与批准主体接线完成，strict v1 合同未放宽；(3) 批准记录/工具缺失时 Begin 前 closed；(4) 负向测试证明无 shell、无附加 argv、无自证成功；(5) 上述 9 条验收命令通过；(6) 未宣称约束 8 全项满足或生产解除；(7) 非作者 CodeReview 与 Main 同 revision 范围核证完成。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-s424a/report.md`（实施者创建，含命令日志、JSON 摘要、哈希与未验证边界）。 写集对账注记（2026-11-02）：本卡写集已对账至 design r72，闭合设计侧 F017 写集缺口；**写集对账完成 ≠ 可实施**。剩余门禁保持不变并逐条保留：`F004`（ADR 同步待架构决策方）、`N005`（`decisions/framework.md` 状态段与原 `tasks.md:3197`（r70 对账后 `:3220`；r72 写集扩写后 `:3240`）冲突，范围外）、`SEC-F007` 与 Linux 侧能力 NOT RUN、生产 closed 与 ADR-0031 约束 8 未满足；`:3186` 不勾选、任务状态与责任人派发记录不变。 本卡写集覆盖口径为 r65.12／r66.4／r65.13 B5，并按 design `:4051` 的 r72.4 累积口径并入 r63.9.4／r64.12 建议表中的 6 个生产文件（`recovery_restore_approval.go`、`recovery_restore_target.go`、`recovery_restore_receipt.go`、`recovery_restore_witness.go`、`recovery_config.go`、`internal/recovery/restore_source.go`）及 10 个配套测试（host／exec／approval／target／receipt／witness／config／restore_source，逐项见上列写集扩写表）；`F017-R1` 已按 r72.4 累积口径并入本卡授权写集（2026-11-02），设计侧 F017 写集缺口闭合。写集扩写**不等于**实施方案确定，不解除 `F004`／`N005`／`SEC-F007` 等上述门禁。实施状态段所述「本卡写集不允许新增 build-tag 文件」是 4 文件写集下的历史判断，不适用于现写集。
- **责任人**：Backend-Dev + 非作者 Reviewer；已于 2026-11-02 派发实施（Agent ID `01a1991e-7323-73c2-be1d-93f36234ff62`，Backend-Dev Role），已回报 BLOCKED（2026-11-02；零代码、零测试，证据见实施状态段），未关闭、未签收。
- **不宣称边界**：不宣称真实批准角色已任命、真实工具/旧备份已演练、POSIX/ACL/进程隔离已实测、原生耐久回执已取得、生产 restore/`enter`/`resume` 已解锁或生产可用；不宣称 S4.24b/S4.22 完成、Case18/S8.2 PASS；不代替独立评审、人的生产批准或 Git 写授权。
- **停止条件**：若 ADR-0031 决策修订未被架构与安全独立评审接受，停止（不得以约束 8 全项未满足为由绕开或提前实施）；若 design/ADR 无法给出批准记录字段、工具 argv、隔离机制或回执读取路径，停止并回报缺项；若需选定具体执行身份切换机制、改 `internal/identity/**`/`internal/article/**`、建第二 authority 或引入 fallback，停止并回设计。注（2026-11-02 澄清，承接架构侧 `B13J-D06-R02-F005`）：`执行身份切换机制未定` 不构成启动前置阻断——该项属本卡 G0 内回设计/证据产出的范围；机制未定时可先完成 `recoveryClosedRestoreDriver` 保持与负向测试，但不得声称隔离实测通过、不得预置执行环境或提前选定机制（依据 ADR-0031 r06「登记≠启动」段与不变量 3）。

- **实施状态（2026-11-02，BLOCKED，未关闭）**：已于 2026-11-02 派发 `add-backend-dev`（Agent ID `01a1991e-7323-73c2-be1d-93f36234ff62`），实施者按本卡停止条件停止并回报阻断，未改任何代码、未创建测试、未放宽断言（三个目标源文件哈希与改前快照一致：`main.go` `6D5373CA…CF8727`、`recovery_restore.go` `C0CB4B01…668BBA`、`recovery_runtime.go` `1C679377…96E5`；证据：`.spec/wip/recovery-runtime/evidence-b13j-s424a/report.md` 10670 B / `815CC215DB615B77E176FBEA8CA45079235A0082854F500A50682B47C3443763`；作者已修正一次，旧 10030 B / `4CC0F4678373E0AEFB540EF54F97025BE70349970368731A998E0CB4F404B7C7` 已作废，原因=原报告「三个源文件与 HEAD 一致」表述不实（这些路径在 HEAD 中未被跟踪），现改为「本次会话零写入」）。阻断项经 Lead 复核成立：B1 批准记录无载体（ADR-0031 给出最小字段，但 design 未给 wire 与读取路径；`cmd/han-server/recovery_config.go` 只有 `HAN_RECOVERY_RESTORE_EXECUTABLE`/`_SHA256`/`_MANIFEST_PATH` 三键且不在本卡写集）；B2 既有 CAS 信任边界无法表达恢复来源批准（`recovery.InstallationSourceVerifier.VerifyInstallationSource` 入参 `InstallationManifest` 不含 `BackupRef`/`ArtifactDigest`/`ArtifactBytes`/`SourceSchemaDigest`）；B3 专用 restore 数据库角色的目标连接凭据无载体（`run` 内仅有业务 DSN `HAN_DATABASE_DSN`，以其充当子进程目标连接违反不变量 3 的角色分离）；B5 `st_dev`/`st_ino`、owner/ACL 与加载器预检需 POSIX 专用实现，本卡写集不允许新增 build-tag 文件；B6 平台门前置后负向 (a)(b) 在本机为空真。故本卡**未关闭**、checkbox 保持未勾选、生产继续 closed（`recoveryClosedRestoreDriver` 仍是默认，生产 `enter`/`resume` 的 DB 替换 closed，ADR-0031 约束 8 未满足，生产未解除）。设计修订已产出并实测：`design.md` 854991 B / `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` → 876660 B / 5109 行 / `EF55DC9272F24C7626F028F705B4322B4E33DCFA6D0197545015CA7DE4FE5811`，`revision: identity-auth-design-ir01-r61`，r61 正文 `:3925`–`:4087`（B1 批准文件载体 `:3929`、B2 `RestoreSourceVerifier` `:4012`、B3 目标凭据 `:4045`、B5 平台文件边界 `:4055`、B6 负向矩阵 `:4072`）；作者报告 `.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r01/report.md` 18793 B / `B78CA378F6050B9494BEBE18ABC6C11171E6DB9ECCBB594206EE7C5AA7D4C6FA`。两路独立评审均 CHANGES REQUIRED：架构侧 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r01/report.md` 56625 B / `78489013177ED00D460B3AA3A6E5EFD810CF2292CDE4C30B2E508D8F8C16C7DA`，BLOCKER 5 / HIGH 11 / MEDIUM 4；安全侧 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r01/report.md` 40500 B / `C5D5BE68CCEAFC349722D6677C40F6A5BB8C46CB58A8D6F4FC7F01D16ED3A637`，BLOCKER 5 / HIGH 2 / LOW 1。两路重合的五个 BLOCKER：批准摘要自证（无可信外部锚）；复核人独立性不可机核（无 principal ID、无执行者字段）；撤销 TOCTOU 未闭合（`prepared()` 不重读批准文件）；`dsn []byte` 经 `json.Marshal` 变 base64 与「线上形状不变」矛盾；外部回执验签/见证未编码。环境结论（2026-11-02）：卡面负向 (d)(e)(f)（真实 fd-exec、外部见证、加载器预检）在本机 Windows 结构性不可达；即使设计再修订并通过评审，`S4.24a` 也无法在本机满足关闭条件 (4)(5)。同链的 `S4.24b`（POSIX 文件锁）与 `S4.22` 同样被 POSIX 环境卡住。下一步须先取得 Linux 环境与可信外部锚决策，再按修订后的写集重新派发；在此之前不得进入实施、不得勾选。
<a id="task-identity-auth-s4-24b"></a>

- [x] identity-auth-S4.24b 实现部署独占 POSIX 锁后端并把 bundle 的 `Exclusive` 接入唯一 production root；受 ADR-0032 门禁，requires `identity-auth-S4.23`。 完成@2026-11-05
- **2026-11-05 关闭（经决策不纳入本版，ADR-0037）**：POSIX 独占锁后端与 `Exclusive` 接线未实现（NOT RUN）；按 ADR-0037 关闭，不声称部署互斥已验证；Windows 生产仍 unsupported（退出码 2）。

- **目标**：按 ADR-0032 不变量 1–6 实现 POSIX 独占非阻塞 OS 文件锁：`O_CLOEXEC` 常驻 fd、owner-only/no-follow/单一文件、绝对规范路径、`journalPath != lockPath`、进程死亡即释放、删除文件不解除持有、无 fallback/无降级；把 `identity-auth-S4.23` 交付的 bundle `Exclusive` 接入唯一 root，使 `normal`/`recovery`/`requeue` 共用同一把锁；设施缺失、ACL 不可验证、路径不合法、锁获取失败或 Windows/平台不支持时在任何 DB/CAS/文件状态变更前退出码 2。
- **编号/类别理由**：沿用 gap-map 拆分后的候选 `S4.24b`（部署独占锁后端与 `exclusive` 接线），requires 候选 `S4.23`（ADR-0032「解除顺序」段候选 `S4.24b` 节点；引用按版本无关表述，段名锚：ADR-0032 第 8 条候选 S4.24b 节点）；本轮首次登记为正式 Task，类别为实施卡。
- **类型/Story与范围**：开发（identity Story S04 的部署互斥前置；同卡局部实现与集成测试）。范围为 POSIX 锁后端、平台门禁、退出码 2 与 `exclusive` 接线；不覆盖真实生产文件系统 ACL 取证、三入口真实部署证据、S4.22 root 解除或 Windows 生产支持。
- **精确文件清单/写集（7 文件）**：
  - `cmd/han-server/recovery_exclusive.go`（修改：生产 POSIX 锁后端构造与 `restoreDeploymentExclusive` 接线；无 acquire 仍 `ErrAdmissionUnsupported`）
  - `cmd/han-server/recovery_runtime.go`（修改：从 bundle 的 `Exclusive` 接入唯一 root；不以 `inputs.exclusive` 替代生产来源）
  - `cmd/han-server/main.go`（修改：配置/平台不支持时在副作用前映射退出码 2；仅必要装配）
  - `internal/recovery/filelock_flock.go`（修改：新增部署独占锁获取/校验路径，复用 `unix.Flock`/`O_CLOEXEC`/`O_NOFOLLOW`，不改变 Journal 锁既有语义）
  - `internal/recovery/filelock_unsupported.go`（修改：非 POSIX 平台返回 `ErrUnsupportedPlatform`，供退出码 2 路径使用）
  - `cmd/han-server/recovery_exclusive_posix_integration_test.go`（新增；`//go:build integration && (aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)`；顶层 `TestRecoveryDeploymentExclusivePosix`）
  - `cmd/han-server/recovery_exclusive_unsupported_integration_test.go`（新增；`//go:build integration && !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)`；顶层 `TestRecoveryDeploymentExclusiveUnsupportedPlatform`；断言非 POSIX 平台在生产路径任何 DB/CAS/文件状态变更前返回退出码 2，并以 `ErrUnsupportedPlatform` 为错误分类依据，不得只断言错误非 nil）
  - 证据报告 `.spec/wip/recovery-runtime/evidence-b13j-s424b/report.md` 为非代码产物，不占 7 文件写集，由实施者在关闭前创建。
- **硬/软 requires**：硬 requires `identity-auth-S4.23`——S4.23 交付且 bundle 四端口齐备（`Exclusive` 仍可为 closed）；硬 requires `identity-auth-D07`——ADR-0032 解除顺序修订 `adr-0032-r05` 被架构与安全独立评审接受；`Exclusive` 离开 `recoveryClosedPort` 由本卡交付完成（ADR-0032 解除顺序引用按版本无关表述，段名锚：ADR-0032 解除条件第 8 条「解除顺序」）。软关联：`identity-auth-S4.22` 消费本卡锁后端；不将 S4.22 计为硬前置。
- **验收命令（NOT RUN；13 条；期望每条 exit 0）**：
```text
gofmt -l cmd/han-server/recovery_exclusive.go cmd/han-server/recovery_runtime.go cmd/han-server/main.go internal/recovery/filelock_flock.go internal/recovery/filelock_unsupported.go cmd/han-server/recovery_exclusive_posix_integration_test.go cmd/han-server/recovery_exclusive_unsupported_integration_test.go  # exit 0，输出为空
go build ./...  # exit 0
go vet ./cmd/han-server ./internal/recovery  # exit 0
go vet -tags=integration ./cmd/han-server  # exit 0
go test ./cmd/han-server -run '^(TestRecoveryDeploymentExclusive|TestRecoveryRuntimeComposition)$' -count=1 -json  # exit 0
go test -race ./cmd/han-server -run '^(TestRecoveryDeploymentExclusive|TestRecoveryRuntimeComposition)$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryDeploymentExclusivePosix$' -count=1 -json  # exit 0
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryDeploymentExclusivePosix$' -count=1 -json  # exit 0
go env GOOS  # 必须为 windows 或其他非 POSIX；POSIX 主机上的 exit 0 不算通过（本包该测试被构建标签排除）
go test -tags=integration ./cmd/han-server -run '^TestRecoveryDeploymentExclusiveUnsupportedPlatform$' -count=1 -json  # exit 0，且 -json 输出须含该测试名 "Action":"pass" 事件；无测试执行即失败
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryDeploymentExclusiveUnsupportedPlatform$' -count=1 -json  # exit 0，同上须含实际执行事件
go test -tags=integration ./cmd/han-server -run '^TestRecoveryConfig$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryRuntimeComposition$' -count=1 -json  # exit 0
```
- **负向测试与失败 oracle**：`TestRecoveryDeploymentExclusivePosix` 至少覆盖：(a) 第二个进程以同一路径启动必须被拒；(b) 持锁进程崩溃后新进程可取得锁；(c) 删除锁文件不解除持有；(d) 符号链接/目录/非 owner-only/ACL 不可验证/相对路径 → 拒绝。`TestRecoveryDeploymentExclusiveUnsupportedPlatform` 覆盖：(e) 非 POSIX 平台在生产路径任何 DB/CAS/文件状态变更前返回退出码 2，并以 `ErrUnsupportedPlatform` 为错误分类依据（不得只断言错误非 nil）；同时须在调用前后对锁路径、journal 路径与相关目录取存在性+内容+元数据快照并逐项比对相等（快照不一致即失败），使「无副作用」可观测，不得以退出码与错误分类代替；元数据至少枚举目录条目集合（含隐藏/临时文件）、每文件 size、mtime（纳秒）、owner/uid、mode/ACL、file id（POSIX inode；Windows 用 `GetFileInformationByHandle` 的 file index，退化时用 size+mtime+ACL 并注明平台可核字段）、锁文件不存在、journal 目录零新增条目；并须加「零写 tripwire」（以 spy 包装文件/DB 接口，任何写调用即失败）与「DB 连接尝试计数为 0」断言，证据保存前后快照 JSON 与逐项 diff 的哈希；零写 tripwire 须带正控——执行一次「故意引入写入」的对照运行，证明该 tripwire 在该情形下确实失败（正控未失败即 tripwire 无效；无正控记录视为该断言未验证）。oracle：第二个进程取得锁、删文件后锁释放、错误路径被接受、退出码不为 2 或错误分类不是 `ErrUnsupportedPlatform`、零写 tripwire 正控未失败、或调用前后快照不一致即测试失败，本卡不得关闭。
- **关闭条件（8 项）**：(1) `identity-auth-S4.23` 交付证据可核、bundle 四端口齐备并登记（`Exclusive` 仍可为 closed），且 `identity-auth-D07` 的 `adr-0032-r05` 修订可核；本卡交付结果包含 `Exclusive` 离开 `recoveryClosedPort`；(2) POSIX 后端按 ADR-0032 不变量 1–6 实现，无 fallback/降级；(3) 路径/ACL/no-follow/`journalPath != lockPath` 校验完成；(4) 上述负向测试全部通过——`TestRecoveryDeploymentExclusivePosix` 承接 (a)–(d)，`TestRecoveryDeploymentExclusiveUnsupportedPlatform` 承接 (e)；(5) 配置/平台不支持时退出码 2 且在任何状态变更前拒绝——由 `TestRecoveryDeploymentExclusiveUnsupportedPlatform` 以 `ErrUnsupportedPlatform` 为错误分类依据断言，不得只断言错误非 nil；并须断言调用前后对锁路径、journal 路径与相关目录的存在性+内容+元数据快照逐项相等（「任何状态变更前」须可观测），证据须记录该快照比对结果，不得只记 exit 0 与错误分类；元数据字段按负向 (e) 枚举项逐项比对（目录条目集合、size、mtime 纳秒、owner/uid、mode/ACL、file id、锁文件不存在、journal 目录零新增），并须记录「零写 tripwire」与「DB 连接尝试计数为 0」的断言结果；须在证据中记录执行主机 `GOOS`；非 POSIX 断言须在非 POSIX 主机执行并通过，POSIX 断言须在 POSIX 主机执行并通过；任一侧仅有 exit 0 而无该测试的实际执行事件，视为未验证，本卡不得关闭。(6) 三入口共用同一锁的接线可核；(7) 未宣称人的生产批准或生产解除；(8) 非作者 CodeReview 与 Main 同 revision 范围核证完成；解除条件 1–3 的关键负向用例须由非作者 Reviewer 在独立主机复跑并留 `-json` 原始证据与主机指纹；「独立主机」指与实施者自测主机不同的同平台主机：POSIX 用例（第二进程拒锁、持锁进程崩溃后释放、删除锁文件不解除持有）在 POSIX 主机/文件系统复跑，非 POSIX 用例（退出码 2、`ErrUnsupportedPlatform`）在非 POSIX（Windows）主机复跑，两者不得互相替代；仅实施者自测不足以采信，且不得以空跑 exit 0 冒充通过。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-s424b/report.md`（实施者创建，含命令日志、JSON 摘要、哈希与未验证边界）。
- **责任人**：Backend-Dev + 非作者 Reviewer；当前未派发/未认领。
- **不宣称边界**：不宣称真实生产文件系统 ACL/NFS 语义已验证、三入口真实部署已取证、Windows 生产支持、S4.22 已完成、`normal`/`recovery`/`requeue` 已生产解锁或生产可用；不代替独立评审、人的生产批准或 Git 写授权。
- **停止条件**：若 `identity-auth-S4.23` 未交付或 bundle 四端口不齐备，停止（不得先接线后补来源；`Exclusive` 仍可为 closed，其离开 closed 属本卡交付结果）；若 `identity-auth-D07` 的 `adr-0032-r05` 修订未被接受，停止；若 design/ADR 无法给出锁路径合同、平台门禁或退出码映射，停止并回报缺项；若需扩 Windows 生产支持、改 `internal/recovery` 之外未列文件、引入第二锁来源或 fallback，停止并回设计。若把 `cmd/han-server/recovery_exclusive_unsupported_integration_test.go` 的构建标签放宽到包含 POSIX 平台（即移除 `!(…)` 取反），停止。若执行主机不具备所需平台能力、只能得到空跑 exit 0，停止并回报缺证，不得以空跑冒充通过。

<a id="task-identity-auth-s4-22"></a>

- [x] identity-auth-S4.22 完成 production root 装配与 recovery secret candidate 生产来源；硬 requires `identity-auth-S4.24b`，登记按 ADR-0030 约束 9(c) 不受约束 7 未满足项阻挡。 完成@2026-11-05
- **2026-11-05 关闭（经决策不纳入本版，ADR-0037）**：11 端口 root 装配与 secret candidate 生产来源未实现（NOT RUN）；按 ADR-0037 关闭，不声称 production root 已装配；硬依赖 `S4.24b` 在本版不交付。

- **目标**：在唯一 production `cmd/han-server` root 上按 design `identity-auth-design-ir01-r60` 的 11 端口清单完成装配（部署独占与解除边界受 ADR-0032「解除顺序」段约束，引用按版本无关表述，段名锚：ADR-0032 解除条件第 8 条「解除顺序」）：复用既有 journal/articleSnapshots/consumer/installation，接入 S4.23 bundle、S4.24a restore 与 S4.24b 部署独占锁，`articleAccount` 按既有桥接装配，`secretCandidate` 由 `HAN_RECOVERY_INITIAL_ADMIN_SECRET` 惰性只读候选适配器提供；生产不得用 `recoveryRootInputs.exclusive` 替代 bundle 的 `Exclusive`。本卡登记本身按 ADR-0030 约束 9(c) 不受约束 7 未满足项阻挡；解除 closed 仍须约束 7 全项加人的明确生产批准。
- **编号/类别理由**：沿用 ADR-0032「解除顺序」段候选 `S4.22` 节点（引用按版本无关表述，段名锚：ADR-0032 第 8 条候选 S4.22 节点）与 gap-map 的候选 `S4.22`（production root 装配 + recovery secret candidate）；本轮首次登记为正式 Task，类别为实施卡。
- **类型/Story与范围**：开发（identity Story S04 的生产 root 最终装配；同卡局部实现与集成测试）。范围为 11 端口 root 装配、secret candidate 惰性生产来源与 `exclusive` 覆盖路径隔离；不覆盖真实生产解除、真实 E2E、人的生产批准或约束 7 全项满足声明。
- **精确文件清单/写集（4 文件）**：
  - `cmd/han-server/main.go`（修改：唯一 production root 装配 11 端口；不得设置 `recoveryRootInputs.exclusive` 作为生产来源）
  - `cmd/han-server/recovery_runtime.go`（修改：root 装配与 `:565-567` 的 `inputs.exclusive` 覆盖路径按仅测试构建隔离；secret candidate 惰性接线）
  - `cmd/han-server/recovery_sources.go`（修改：新增 `HAN_RECOVERY_INITIAL_ADMIN_SECRET` 惰性只读候选适配器，读取不等于验证，不持久化/不落日志）
  - `cmd/han-server/recovery_root_production_integration_test.go`（新增；`//go:build integration`；顶层 `TestRecoveryProductionRootAssembly`）
  - 证据报告 `.spec/wip/recovery-runtime/evidence-b13j-s422/report.md` 为非代码产物，不占 4 文件写集，由实施者在关闭前创建。
- **硬/软 requires**：硬 requires `identity-auth-S4.24b`。软关联：与 `identity-auth-S4.24a` 的关系按 ADR-0031 单独登记（root 装配可在 restore 保持 closed 时进行）；不将 S4.24a 计为硬前置。
- **验收命令（NOT RUN；期望每条 exit 0）**：
```text
gofmt -l cmd/han-server/main.go cmd/han-server/recovery_runtime.go cmd/han-server/recovery_sources.go cmd/han-server/recovery_root_production_integration_test.go  # exit 0，输出为空
go build ./...  # exit 0
go vet ./cmd/han-server  # exit 0
go vet -tags=integration ./cmd/han-server  # exit 0
go test ./cmd/han-server -run '^(TestRecoveryRuntimeComposition|TestRecoverySecretSourceIsLazyCandidate)$' -count=1 -json  # exit 0
go test -race ./cmd/han-server -run '^(TestRecoveryRuntimeComposition|TestRecoverySecretSourceIsLazyCandidate)$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryProductionRootAssembly$' -count=1 -json  # exit 0
go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryProductionRootAssembly$' -count=1 -json  # exit 0
go test -tags=integration ./cmd/han-server -run '^TestRecoveryConfig$' -count=1 -json  # exit 0
```
- **负向测试与失败 oracle**：`TestRecoveryProductionRootAssembly` 至少覆盖：(a) 生产装配不得设置 `recoveryRootInputs.exclusive`；若 `cmd/han-server/recovery_runtime.go:565-567` 的无条件覆盖仍可从生产路径触发，测试必须失败；(b) 仅测试构建开关隔离时，生产构建的 `inputs.exclusive` 必须为 nil 且 registry `Exclusive` 为唯一来源；(c) `secretCandidate` 仅在 enter 或信任根未准备的 resume 瞬时读取，normal/verify/已完成信任根零读取；(d) secret 缺失/为空时保持 closed。oracle：生产路径打开 `exclusive`、secret 被普通启动读取或读取失败未保持 closed 即测试失败，本卡不得关闭。
- **关闭条件（9 项）**：(1) `identity-auth-S4.24b` 交付证据可核；(2) 11 端口 root 装配完成且 `exclusive` 唯一来源为 S4.23 bundle；(3) 生产装配不得设置 `recoveryRootInputs.exclusive`（`cmd/han-server/recovery_runtime.go:565-567` 对 `inputs.exclusive` 无条件覆盖），或必须以显式「仅测试构建」开关隔离该覆盖路径；须提供可失败的负向测试并留存证据；(4) `secretCandidate` 唯一生产来源为 `HAN_RECOVERY_INITIAL_ADMIN_SECRET`，惰性只读、无文件/DB/argv/密钥管理服务 fallback；(5) secret 读取不等于验证，失败清零且不落日志/Trace；(6) 上述 9 条验收命令与负向测试通过；(7) 本卡登记按 ADR-0030 约束 9(c) 不受约束 7 未满足项阻挡；解除 closed 须约束 7 全项加人的明确生产批准，本卡不宣称满足；(8) 未宣称生产可用、E2E 或 Case18/S8.2 PASS；(9) 非作者 CodeReview 与 Main 同 revision 范围核证完成。
- **证据落点**：`.spec/wip/recovery-runtime/evidence-b13j-s422/report.md`（实施者创建，含命令日志、JSON 摘要、`inputs.exclusive` 负向测试证据、哈希与未验证边界）。
- **责任人**：Backend-Dev + 非作者 Reviewer；当前未派发/未认领。
- **不宣称边界**：不宣称约束 7 全项满足、生产 closed 已解除、人的生产批准已取得、真实部署/E2E/PG/HTTP 已运行、Case18/S8.2 或整体恢复入口 PASS、生产可用；不宣称 `S4.24a` 的 restore 已解锁；不代替独立评审、人的业务签收或 Git 写授权。
- **停止条件**：若 `identity-auth-S4.24b` 未交付，停止（登记本身可先行，但实施不得开始）；若 design/ADR 无法给出 11 端口来源、secret 读取时点或 `exclusive` 覆盖路径的隔离方式，停止并回报缺项；若需改 11 端口清单之外的新端口、建第二 runtime/Store/pool/authority/fallback、修改 ADR 或扩 S10.5，停止并回设计/架构。
