# B13A：基础恢复准入实施交接

2026-10-29，固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本轮已完成 B13A / identity-auth-S4.7 的中性恢复准入、公开受控 Enter/Resume、阶段许可及取消/真join机制。原R01四项在R02关闭，R02新增两HIGH经原作者修复、同Ptolemy R03全部CLOSED/新增0；Main完成同版现场核收并关闭S4.7，接收本批中性U/C机制。真实PG/恢复/root/E2E不在本结论内。

## 范围与实际归属

- Zeno / Backend-Dev，真实 ID `01a18486-3e11-7852-9723-ad380e0bb392`：唯一五文件实现者，已完成只读 G0，Main 已发 [SOURCE/LOCAL START](evidence-b13a-r01/source-local-start.json)。
- Goodall / Task-Planner，真实 ID `01a18414-b2ab-7392-9b04-fd974203bf5e`：完成真实认领及原四卡/四批r52责任同步，未改源清单、依赖或批次；已STOP并关闭；[认领证据](evidence-b13a-r01/claim-r01/index.json)。任务最终关闭由Main执行。
- Ptolemy / Reviewer，真实 ID `01a18492-af9e-7ea2-9cd5-e4dd368d7bee`：非作者，完成R01完整范围、R02/R03定点复审；R03 PASS、两HIGH CLOSED/新增0，已STOP并关闭。其独立29叶与Main复跑230叶分别归属，不混算。
- Main：固定基线、窗口、实际 diff/结果核接收；不代人的签字。文件所有权属于协作约定，不是操作系统硬隔离。

五个获准源码为 `internal/recovery/runtime.go`、`admission.go`、`runtime_test.go`、`admission_test.go`（新增），及 `journal.go`（仅中性投影/ports）。原 Journal 写方法、ctx 迁移、具体领域/pgx、SQL/迁移、其他13张新卡及 production root 不在本批写集。

## G0 与授权依据

[G0 可读回执](evidence-b13a-r01/g0-r01/receipt-utf8.md)确认五文件足够、无阻断性 design gap；原编码损坏回执留存，不以乱码作为承接依据。[精确读集](evidence-b13a-r01/g0-r01/readset.md)区分方法全文、设计切片与搜索。设计仍 draft、人的接纳 unknown；按 standing 授权推进软件，不补造环境或批准事实。

Main 留存1188文件基线，确认四个新增路径原本不存在、journal 前像存在，分支/HEAD/空暂存区符合约定。Task-Planner 仅修改认领与统计/责任引用，不改 requires、验收或九批地图。现役任务状态始终取 [Tasks 原卡](../../../.add/project/knowledge/tasks/tasks.md#task-identity-auth-s4-7)，本记录不另立状态账。

## 验证与既有失败

本卡要求先行为失败证据，再完成全量历史或 genesis、独立 revision/ABA、最多两次完整证明、封闭许可与代际撤销、自有 append 原因等实现；五个必要风险组须在普通与 race JSON 中实际 run/pass，零命中、Skip 或只父 top 通过均不够。Manifest 七项仍须逐项真实运行，不能把 build 当测试或独立 Review。

Main 在任何本批源码写入前运行根目录 `gofmt -l .`，exit 2。输出仅指向旧 `.spec/wip/identity-auth/evidence-s10.5` 的23份探针/带行号摘录等 `.go` 证据：存在未格式化和不可解析摘录。精确路径/哈希及原始 stdout/stderr 留在 Main Temp `han-b13a-lead-asiz0vv9`。保留这些旧证据，不格式化、删除、改名、移动或归档；五源格式通过不改变全仓 FAIL，也不由作者自行豁免。本卡最终检查处置须由 Main 在实际结果和独立评审后明确记录。

本地运行使用 OS/path allowlist，`GOENV=off`、`GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`，GOFLAGS与测试DSN为空；不复制或打印真实 HAN secret。已有 unit/contract 的合成 loopback fixture 和测试 binary 子进程在 LOCAL 范围内。真实 PG/DDL、integration tag、生产服务/worker/restore、网络安装、secret读取、Git写入及清理均未获本批窗口。

## R01 失败与原作者修复

[作者冻结交付](evidence-b13a-r01/author-r01/delivery.json)为4新增＋1修改、2440新增行；journal仅48行中性声明，逐字节逆换恢复修改前旧逻辑。作者普通/race各99风险leaf通过；Main全文读四新源与journal完整增量，重跑定向普通/race也各99风险leaf run/pass、零skip/fail。七项Manifest按序实际退出2/0/0/0/0/0/0，原始日志全保留。这些结果不替独立评审。

[Ptolemy R01](evidence-b13a-r01/review-r01/review-report.md)完成固定五源及直接消费者两轮审查，finding `RECOVERY-B13A-CR-R01-F001`～`F004`分别为：公开受控恢复阶段许可生产入口缺失；Stop取消未传入正在执行的proof/startup端口；Verify未结束即释放部署独占；已确认合法append并发于Verify会被误判漂移。独立overlay普通/race各5例，4行为FAIL＋1正控PASS，无编译失败/Skip/race告警。Main已全文读报告/receipt/readset并接受全部finding本轮修复；原失败链与探针留存，不以原99叶通过或格式例外覆盖。

原作者Zeno已获同五源SOURCE/LOCAL返工；F002～F004可先独立修复，F001须先说明同既有设计的中性受控生产机制，不能公开任意purpose或SetReady，也不擅自把后继真实restore/identity SQL/root移入本卡。若需契约或所有权变更，只停受影响部分并交原owner对齐。随后同Ptolemy作定点复审，Main再核同版成果；当前4项均OPEN。

Main按用户standing采用建议授权，已落[本卡有限旧格式检查例外](evidence-b13a-r01/format-exception-main-r01.json)：精确5源版本/23旧证据哈希，旧根gofmt保持FAIL，其余6项/源码格式/风险测试/独立Review不豁免，不是人的签字或全项目验证PASS。修复新版本仍需重核适用性，不能自动外推。

## F001 的原设计增量与定点复审

三个生命周期问题的作者修复停在F234版本，普通/race各114风险leaf通过（含原99及新增回归），尚未独立关闭。F001原作者给出窄Consumer/真实维护退出接缝后，Main交原设计owner Banach只读核对并采用推荐；原同份identity设计更新为r51（SHA `F87E1EB9073F4B2A723A098776762F759332B8086384BC0ED3C79DC49F624564`，+81/-8），明确restore/bind前置许可、当前run独立绑定、动作依赖、RecoveryConsumer、root RecoveryExit和identity同Store撤Session/审计责任。原源码保持F234，设计仍draft，人unknown，不预开入口实现窗口。

[原Kuhn设计R01](evidence-b13a-r01/design-review-r01/review-report.md)仅审r51增量及直接消费者，提出唯一HIGH `RECOVERY-DES-R51-R01-F001`：Resume候选秘密缺明确持retained Begin key的纯只读校验执行者/成功合同。Main已全文接收并按用户standing采用[最小修复](evidence-b13a-r01/design-repair-start-r51-r01-f001.json)：原Journal持钥读取者新增精确VerifyBeginSecret中性方法，沿旧HMAC算法和对应retained key验证候选，RootSource只惰性读取，Runtime在PrepareRecoveryTrust前独立比对既有Begin来源；不以新Begin、普通SHA或当前active key替代，不导出key。原作者已仅改单一设计SSOT至r52（SHA `5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`），同[Kuhn设计R02](evidence-b13a-r01/design-review-r02/review-report.md)已定点关闭该设计finding，新增0。Main全文核增量、报告和receipt并技术接收，向原Zeno发出[F001同五源SOURCE/LOCAL START](evidence-b13a-r01/f001-source-local-start-r52.json)。代码四项仍待独立复审；设计结论不代代码关闭，design仍draft、人的精确接纳unknown。

## 修复版、复审与最终接收

原作者已完成并STOP，[F001修复交付](evidence-b13a-r01/author-f001-r02/delivery.json)相对F234 +1233/-9行。Main已全文核1388行完整差量、G0/交付/范围回执、七项Manifest原始结果及输出，并现场核五源与冻结副本一致；Journal最初前像全部行保持，仅新增中性声明。Main守卫确认1183个受保护基线文件未改，仅显式授权的五源与identity设计、Tasks/Plan、framework发生本轮变化；分支、HEAD与空暂存区保持。

作者及Main自己的普通/race均实际156节点、139叶子run/pass，八组非零、原114用例全部保留、零skip/fail。Exit阻塞成功断言在父组主体实际执行，负向子例为1叶，具体场景对应见作者delivery，未伪增叶数。Manifest按序仍为2/0/0/0/0/0/0；Main已核写前23旧路径hash与根fmt双raw完全一致，并对当前精确五源落[有限格式处置r02](evidence-b13a-r01/format-exception-main-r02.json)，根fmt仍FAIL，其他检查/代码finding不豁免。

原Planner四卡/四批r52职责同步和原Pascal同源版本追溯已冻结；[Kuhn规划/metadata Review](evidence-b13a-r01/plan-r52-review-r01/review-report.md)新增0，[Main接收](evidence-b13a-r01/mapping-ledger-main-reception-r01.json)现场核四对象精确SHA。未增Task/源文件/requires/批次，采用来源仍6，README零改。此结论不关闭代码finding。

[同Ptolemy代码R02](evidence-b13a-r01/review-r02/review-report.md)已完成：原R01四项分别CLOSED；新增两项HIGH `RECOVERY-B13A-CR-R02-F001/F002`分别为受控恢复跨端口的无解释namespace revision漂移被接受并开流，以及consumed且缺Store当前行仍先授bind许可并调用写端口。独立普通/race各9叶、7PASS＋2行为FAIL，无编译失败/Skip/race告警；Main全文核报告/receipt、解析实际事件并核五源未漂移，已按用户standing采用修复建议向原Zeno发[同五源SOURCE/LOCAL返工](evidence-b13a-r01/repair-start-r02.json)。Reviewer已STOP并关闭；后续同Ptolemy定点R03。Task仍进行中，两项属于本卡已有写前来源/状态围栏，不能以格式例外、139例自测或后继真实实现未就绪为由跳过。新证据均retain，原失败链不覆盖；源码副本以`.frozen`留存，不新增根目录gofmt噪声。

原Zeno已完成[R03返修交付](evidence-b13a-r01/author-r03/delivery.json)并STOP：相对受审R02 +677/-22，实际改四源、journal不变，当前权威副本是`final-*.go.frozen`，早期副本与辅助失败说明保留。Main完整核860行差量、G0/交付、七项结果和raw，并确认现场五源匹配最终冻结副本；守卫仍为1183保护文件未改、只显式授权写集。两项修复保留跨端口不可变namespace观察、仅精确自有Begin/Consume可解释后继；Store恢复/事务变化单独限定；consumed且缺当前行在bind许可及写调用前拒绝。

作者及Main各自普通/race均实际251节点、230叶run/pass，八组非零，原139全保留，零skip/fail。新增用例覆盖恢复与identity/Consume/Exit等边界、双namespace revision/epoch/head/历史变化、合法Store推进及缺行/存在/状态组合；组件fixture不冒真实提供者。Manifest仍2/0/0/0/0/0/0，五源fmt0空输出；[Main精确R03格式处置](evidence-b13a-r01/format-exception-main-r03.json)只承接旧23证据，根fmt仍FAIL。

[同Ptolemy R03](evidence-b13a-r01/review-r03/review-report.md)定点PASS：R02两HIGH均CLOSED、新增0；独立普通/race各32节点、29叶全PASS、零skip/fail/race告警。包含原R01五探针、R02两负例与两公开正例、跨端口/Store/回执/历史别名风险，以及实际生产Coordinator/decoder搭配显式合成端口的五个合法状态推进；未把生产依赖替成新实现来绕过正控。Main全文读报告/receipt、解析独立原始事件并核五源前后SHA一致。R01完整范围＋R02/R03定点范围组合支持本批内部来源、状态、接口、当前绑定、用途、取消、真join和关闭优先的中性U/C机制接收，无剩余已识别组件内部阻断。

Main按用户standing采用建议授权、原任务和批次合同完成最终现场核收，G4-task及B13A机制批次有界技术接收；[原Task](../../../.add/project/knowledge/tasks/tasks.md#task-identity-auth-s4-7)已关闭，[Plan](../../../.add/project/knowledge/plans/ir-01-content-community-batches.md#b13a)仅回填成果与接收依据，不由Task标签自动推整体。关闭时Tasks快照73项、49完成、3进行中、21待办，其余72卡checkbox不变；统计权威仍为Tasks。原59、其余新卡源清单/requires/命令、九批安排及已有具名门槛保持。最终metadata为Main实际结果回填，Kuhn原技术受审对象及所有失败链保持冻结，不循环追改历史源hash或冒称最终metadata重新独立技术受审。

[Main精确接收记录](evidence-b13a-r01/main-final-reception.json)及[最终证据索引](evidence-b13a-r01/final-index.json)绑定当前五源、独立报告、检查、保护范围和Task/Plan元数据。没有提交、推送、部署、清理或人的签字；design仍draft/人的精确接纳unknown。下一安排为B13B的Journal只读实现与HTTP屏障，原S4.8/article S8.3尚未认领或开源写窗，本轮不预签其结果。

S4.7 的组件 U/C 不替代后继真实 Store/Journal/root、两 S4.4、Case18、真实恢复或 E2E。既有具名范围决定与 S4.5/S4.6 的真实验证门槛保持；无部署或需求整体验收结论。
