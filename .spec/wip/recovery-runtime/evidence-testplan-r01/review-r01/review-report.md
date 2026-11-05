# Recovery TestPlan 独立 Review R01

结论：**CHANGES REQUIRED**。发现 1 项 MEDIUM，`RECOVERY-TP-R01-F001` OPEN；CRITICAL/HIGH/LOW 均为 0。该 finding 是新增规划的判定 oracle 矛盾，不重开已关闭的架构或设计 finding。identity 本次恢复规划切片未发现独立缺陷；联合规划整体尚不能给 PASS。

本报告待 Main 现场技术接收，不代替人的接纳、风险接受、Planning/Code/Case/运行通过或后续执行授权。未替作者修订任何共享文件。

**身份、对象与权限**

- Reviewer：Kuhn / add-reviewer，真实 runtime ID `01a1835a-8010-7a73-a8dd-3192719b6965`，由 Main 依用户 standing 授权及项目委派规则指定；ID 来源为 Main 实际 spawn/resume 返回。非作者，非用户直接指定该称谓。
- 作者：Jason / add-test-engineer，Main 提供的真实 ID `01a18395-a83b-7512-b7d1-824b073ab907`。作者包 receipt 是作者证据，未冒充本报告独立判断。
- objectType：design，测试规划；固定 `D:\AID\VSCode\Codex\Han` / main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。独立读取 `.git/HEAD` 与 main ref 确认，无 Git 写操作。
- 收到正式 REVIEW START 后才读取两个 live 冻结 TP。未读取未来变动的 live TaskPlan；沿准备阶段已保存的精确相关任务/计划投影。共享仓库全程只读；所有新增文件仅在本报告独占 Temp。
- Role、add-code-review 取得同版全文；必要 Rule/Profile 沿准备阶段同版全文复用，8 项版本哈希再次相等。方法按非代码对象适用，未套用 Go/构建门禁。

**冻结输入与前后 SHA256**

| 对象 | 旧前像 SHA256 | Review 开始 SHA256 | Review 结束 SHA256 |
|---|---|---|---|
| identity-auth/test-plan.md r24 | `4CDC882308A2CCF6EE249880EBA5EEA469C2C983933C4EE6D42AE1168F5F03D2` | `E6906617E51CC349B783896A3A1D828DA4AA8A03716D0C104E77E9AB40680CF6` | `E6906617E51CC349B783896A3A1D828DA4AA8A03716D0C104E77E9AB40680CF6` |
| article-management/test-plan.md r21 | `DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA` | `338F35D12D002B96536DD8017949C326CE49EE7C851883E179815509511A794D` | `338F35D12D002B96536DD8017949C326CE49EE7C851883E179815509511A794D` |

两路径前缀均为 `.add/project/knowledge/design-workspace/`。作者 before 与 Main `han-recovery-testplan-lead-wrrsctup/before` 对应前像逐字节相等；作者包为 `C:\Users\J00010385\AppData\Local\Temp\han-recovery-testplan-jason-4c378a496cf2487c9379dd99ef36556e`。本轮完整读取两实际 diff：identity 221 行、article 208 行，不以关键词出现当覆盖证明。

**Round 1：完整实际增量与来源边界**

独立通读 identity 全部 7 个 hunk（+143/-15）与 article 全部 7 个 hunk（+132/-15）：header 新旧元数据、恢复规划块、场景/责任映射、Gate 与结论说明。独立静态脚本逐 hunk 检查上下文和行数，并正向重建 frozen 文件、反向逐字节还原原像；两份所有删除行均在旧 header，原正文/Case/历史无删除或替换。UTF-8 BOM、纯 LF、header key 唯一及新增相对文件链接存在、两恢复显式 anchor 均通过。ADR 标题 anchor 的本轮脚本只查文件存在，Main 已核链接的事实单独归属 Main，不扩大本脚本结果。

检查新当前 source design/story/architecture 与四个设计成果、设计 R02/R03 来源的范围：r24/r21 对应恢复增量，旧 r23/r20 header 值移至带历史前缀的字段；旧 Review/执行材料不被扩为新增恢复规划通过。design R01 两 HIGH 的失败历史、R02 design-only CLOSED、R03 metadata-only 解释继续保留。新 TP 本身为 pending independent review / NOT RUN，draft 与人 unknown 未被改为批准。

直接审查新矩阵中的前置、动作、可观测结果、最低充分层及数据/资源归还：Store 全集合与失败优先、genesis/安装、双 namespace 与 Store ABA、纯读零修复、阶段许可/屏障与 unknown join；F001/F002；restore-v1/verify-v1；publication/legacy；Case18；迁移与后继责任。发现 article 第786行正向路径与“所有外部调用为零”不一致，Round 2 再沿稳定契约和真实端口坐实。

**Round 2：沿稳定设计与直接消费者的有界核验**

上游沿准备阶段已读并保存的 Main `verification-design-r50-r23-r01.md`、四个稳定设计/Story、architecture r32、现役 boundary 恢复切片、原 Review R01/R02/R03 及 Jason 旧16组咨询。旧咨询只作覆盖输入，不是新 TP 独立 Review 或执行证据。正式阶段针对修复相关关键段再读设计并定点读取下列真实 seams，未扩成全项目审查。

| 审查链 | 本轮实际核对与结果 |
|---|---|
| F001 维护 Session 到正常身份 | 对照 identity design 3618–3632 与 SessionStore 259–322、563–570、Root Login/PasswordChange 装配。规划要求真实维护 Login/Session、run/install/generation/initial-admin/control-surface 绑定、受限 me/password、事务内 G+1/凭据/Session轮换/flag退出整组回读与 ACKloss，普通 resolver 的 true 拒绝仍保留；结束维护能力后同进程普通策略和重新登录有正控。未以 PublicReady 单独变化掩盖旧静态维护策略；未将既有拒绝行为倒写成恢复预期。 |
| F002 恢复重设与唯一结算 | 对照 identity design 3659–3691（3679–3683 独立重读）、PasswordReset/Operator/Worker 的实际 eligibility、lease/attempt、unknown 与激活结算 seams。规划分可信批准角色与 backup role、User→Store/review 锁序、initial_reset durable intent/lineage/单次发送、ordinary worker 停止时无 Mailer 的受控 driver、unknown hold/deadletter/旧 unresolved、review+role/status+audit+Gate 同事务及 terminal zero-reset。没有扩大普通 reset 的资格或先假清 flag。 |
| Store、Anchor 与正常构造 | Coordinator Consume 的 identity_ready 前置及精确 ready_to_consume+consumed 收敛窗口，与两 TP 的全历史集合失败优先、同代重试、2次受限重核、两 namespace/Store token 隔离、不复用 revision、prepared→bound/genesis/纯读零修复相接。Journal Anchor.Load 及 ReadAllVerified 旧实现用于核对必要接点，缺新实现本身不记缺陷。 |
| article 写入与历史 closure | 对照 article design 2448–2470、实际 Finalizer Append、ReplayEvidence 发布事务、ReplaySource Snapshot/ReadAllVerified。新矩阵分 stage→CAS pending→publish sync→CAS committed→耐久读确认，控制协调与 ordinary worker/startup/recovery 幂等命中、own-cause 屏障及无锁内 I/O/不自等/真实 join；history 不得调用 private fresh helper 或让 identity 查 article 私表。publication 17字段独立 codec/golden、Ready20/first16/首次计数不变、当前原子发布与独立可信 legacy import 分开。仅第786行 oracle 的过强零调用有 finding，见下。 |
| Root、工具与运行生命周期 | actual Root exact registry/405-Allow/Trace 外层、platform resolver/CSRF、lifecycle 三 worker Reconcile→Start→Serve 与真实 join。规划覆盖稳定同一 artifact handle、原生 receipt/独立 origin-target、exact argv/private stdin/stdout上限/ACKloss不再替换、恢复 Begin 先于真实替换、惰性 SourceSecret、7-route允许集、维护/正常surface切换、requeue共享锁。未要求当前尚未实现的新接口已存在。 |
| 迁移与旧约束 | 定点读 0002 recovery_account_reviews 215–247、0005 binding/transition 49–91、0008 request_kind/settlement/delivery/counter complete/不可逆转换相关段、0009 generation/token 全文。新规划明确真实 PG catalog/FK/触发器/旧行 phaseNull、来源token与0010/0011的前后兼容，未把 fixture/in-memory 或 superuser trigger 拒绝当 ACL/CAS 证据；不要求修改既有 S10.5 scope。 |
| Case18、层次与任务承接 | 两个真实 GET503、原 envelope/Trace、Session/Article 前拦截及 Article service/repository/author-reader 各0，恢复后同 root 正常 Auth 与 exact-router 404/405/Allow 正控均有明确 oracle。U/C/P/R 分工有真实 PG、多连接故障和 production wiring 的最低充分层，未用手造503/router或拒启动代替 Case18。原任务引用仅从保存投影核对，新增实施/命令为未登记、未认领建议，不预分配 TaskID、不赋测试卡生产写权。 |

上述为规划可检验性和接点核对，不是证明新 runtime 已实现或全链已运行。全部精确路径、实际行段与 SHA 见 `readset.jsonl`/`readset.md`；未在本轮重新读取全部旧设计或全部消费者。

**[MEDIUM / OPEN] RECOVERY-TP-R01-F001：历史 closure 正向场景的零外部调用判据禁止了必需纯读核验**

- 位置：`.add/project/knowledge/design-workspace/article-management/test-plan.md:786`，§4.8.2“历史port暗读currentFacts或候选冒published”行的证据列。
- 现状：同一行要求短只读 PG lookup、事务外同 Journal 快照核可信前缀、namespace 前后 revision 核对及外层 Store/两 namespace 复核，却把证据写为“所有外部调用为零”。这不是只约束 Refresh/Replay/Fail/Publish 等动作；当前字面没有给必需读取留范围。
- 触发：已有合法历史 publication，普通删除后当前 Facts/head 前进。调用真实 ReadPublishedClosure 验证该历史 closure，执行规定的 Journal snapshot 与 Anchor/source 读取。
- 依据：稳定 article `design.md:2468` 明确要求关闭 DB 事务后读同一锚定 Journal 快照并核前后 namespace revision，`:2470` 返回经核验的 revisions/heads；现役 `internal/recovery/journal.go:75` 的 JournalHeadAnchor.Load 是 provider-neutral 外部端口，`:614` 的 reader 实际调用该 Load。新 port 实现尚未存在不构成本 finding；合同本身已决定读取不能全零。
- 后果：按“全部外部调用=0”实现 spy/验收会拒绝合法核验路径，或诱导用缓存/测试替身跳过当前独立 head/revision 读取以得到绿灯，形成 fail-closed 核验盲区。仅凭存在“纯读”文字无法消除这一冲突。
- 最小修订：由原作者 Jason 在该行界定计数对象，禁止的 Refresh、ReplayAll、Consume/Fail、Publish、Anchor CAS/写入及其他修复等调用分别为0；同时正向记录并核验本合同必需的 DB精确lookup、Journal snapshot、Anchor/source读取与前后revision。可引用既有统一调用账定义，不需另建权威/API/Case或 TaskID。
- 复审验证：定点读取修订 diff 与计数定义，确认合法历史 head 前进路径保留真实 snapshot/Anchor 读和稳定 token，禁止动作=0；读间 revision 变化时仍关闭。该证明可先以规划 oracle 的清晰一致性闭合，真正 U/C/P/R 执行留后续授权，不能伪造已执行结果。

**待决定、建议与接收边界**

无新增架构/业务决定请求，未替任何人作采用或风险接受决定；无另行非阻断优化清单。既有 provider、独立故障域、审批/主体/ACL、备份/目标/native工具receipt、retention/RPO/RTO、运行责任与人的成果接纳等 unknown 仍原样 unknown/blocked。

RECOVERY-DES-R01-F001/F002 继续为 R02 **design CLOSED**；ARCH 既有 F001 不重开。S1.5/S4.4/S8.2 及 Case18 整项未闭；真实 S4.5 CAS / S4.6 retirement 门槛不降；已闭 S4.2/S10.5 不重开。ADR0024/S3.3/S3.5 狭窄豁免不扩展、不记 PASS，S3.6 不变。原任务59/48completed/3active/8todo仅引用准备投影，不在本轮扫描/改写 TaskPlan。

**实际检查与 NOT RUN**

- 独立实际执行：全文读两 diff/作者receipt；按 readset 定点读稳定设计/真实 seams/旧 SQL；Python只读文本、哈希、diff 正反向重建、前像相等、BOM/LF/header/新增链接与明确anchor、7个稳定上游及8项 Rule/Profile hash、全部已记录读取文件结束无漂移、固定 ref/HEAD 核对。`static-verification.json` 给出实际断言结果。
- 作者 verification.json 已解析，其检查与结论只作作者归属；Main 的375保护文件、staged0等现场检查为 Main 证据。本 Reviewer 未重复全保护集扫描，也未把作者/Main 检查冒称自己的执行。
- 实际工具/命令类别：PowerShell UTF-8输出设置、Get-Content读取限定材料；rg定位声明与相关文件；`python read.py <path> <ranges>`；`python audit.py`；独占 Temp 内报告生成。没有调用 Go、数据库、HTTP/SMTP、服务、worker、恢复工具或网络。精确读命令可由 readset 路径/行段重放，范围日志与脚本留存，不声称另有完整 shell transcript。
- **NOT RUN**：Go/编译/单测、PG/DDL/真实CAS、HTTP/SMTP、服务/三worker运行、真实恢复/迁移/安装、真实provider/Unix/ACL/备份/target/retirement、性能/E2E/人确认；没有网络、secret读取或 Git 写、清理/移档、子委派。PlantUML 未渲染，无渲染 PASS。
- 非代码设计期 Review 不因未运行 Go 本身失败；本次 CHANGES REQUIRED 唯一依据是上面的可复核 MEDIUM。修订交回后只复审该 finding 及相关新风险。

报告、receipt、readset 和静态验证只在独占 Temp 留证，R01/R02/R03 既有证据均未覆盖。完成后 **STOP/frozen**，交 Main 现场接收并安排原作者后续动作。
