# S4.10 首轮独立 CodeReview：当前 7C24 的实际 probes

**CHANGES REQUIRED。F001 HIGH 保持 OPEN；原 S001 通过有效持久负例确认，登记为 F002 MEDIUM / OPEN。STOP：本轮全部命令已实际结束并 join，执行窗交回 Main；无源码修复、无继续 Go/PG、无 CodePASS/finding CLOSED/Task/G4。**

执行者：Confucius / add-reviewer，实际 Agent ID `01a1890f-a330-73d1-86fb-236777c54751`，由 Main 返回并在本次 canonical start grant 固定。独立于 Zeno、Design/Task 作者。授权来源为 `.spec/wip/recovery-runtime/evidence-b13d-r01/s410-independent-probes-start-r01.json`；后续同 facet invented-UUID 小负例来自 Main 本会话明确补充指令。

## 对象、两轮范围与执行边界

固定 `D:/AID/VSCode/Codex/Han` / main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；原三源仍为：

| 文件 | 行数 | SHA256 |
|---|---:|---|
| recovery_execution.go | 2694 | 7C24CA226D0D8DF5FC722A35630C090DD01B26F655C56F79F6C4F9FBEE85E407 |
| recovery_execution_test.go | 1053 | E51D1B82BC7E510A3DAB944C2CA16B85BFB49B132ED28AF12EED45EF596BE9CC |
| recovery_execution_postgres_test.go | 1869 | 5907393CAC84B4D92B590A77B7B10A462D4D1C94D3510A063071D31787202980 |

本角色此前两轮实际静态审查的 5616 行全文及真实消费者范围继续有效：第一轮三源本体和所有测试全文；第二轮沿 S4.9 Store/admission reader、0001–0010 全部 DDL/索引/约束、canonical A/B→真实 Article orphan、原幂等 Store/hash/decoder、PasswordResetStore/Gate/replay/settlement 及真实 Runtime 许可和 producer 核契约。完整逐段 readset/两轮覆盖在原独立静态包 `D:/Temp/han-s410-independent-static-5d1681d69580`，manifest SHA `D0420480FE6FEFBCF87520336131CCE826F960CD55C8026ACE4E6D0E239D0CDD`。本轮再次核其 20 payload 哈希全部一致，原静态 receipt 未补改实际 ID 或运行状态。

技术依据仍是 design r56 `75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF`、TP r26 `34826759C79A58EB0B4A296D2DD5C9157014406859153DC6CA0FA6E2BD921DAB` 与原 S4.10 卡，不另加 G0、设计前审或新任务。没有重复 Article/S414 完整评审。

本轮只执行原 8 probes 普通/race；为完整隔离 S001，加入 1 个合法零会话正控，以及 Main 追加的 1 个 invented-UUID 小负控。每模式合计 10 个不同 leaf（原8＋正控1＋追加1），不是完整 U64/P79。生产 overlay=0，test overlay=2；原 U/P 测试前缀逐字节保留，附加 probes 只在独有 Temp，工作树源和 frozen24 前后全部相同。

## 实际结果

| Probe / 控制 | 普通 | race | 实际观察 |
|---|---|---|---|
| FailedRestorationPointer/lawful_positive | PASS | PASS | 合法 invalidated 失败候选、原域响应及结算审计齐，保留 pointer 后 Complete 正常接受 |
| FailedRestorationPointer/generic_failure | **FAIL** | **FAIL** | 原域 classifier 拒绝 generic_failed，但 Complete 错误接受并持久到 identity_ready |
| FailedRestorationPointer/missing_settlement_audit | **FAIL** | **FAIL** | canonical invalidated 响应存在、结算 Audit=0，Complete 仍持久到 identity_ready |
| ExitAuditCompleteness/lawful_zero_sessions_positive（补充正控） | PASS | PASS | 首次零会话仍写一次 Audit，精确同意图重试零新 Audit |
| ExitAuditCompleteness/missing_result_fields | **FAIL** | **FAIL** | 退出结果字段缺失仍返回 nil |
| ExitAuditCompleteness/null_ids | **FAIL** | **FAIL** | revoked_ids=null/count=0 仍返回 nil |
| PersistedDomainReplay/administrator409 | PASS | PASS | 原 Recover→PG持久 snapshot→原 Store.ResolveExisting decoder→真实 replay 返回原失败 |
| PersistedDomainReplay/administrator503 | PASS | PASS | 同上 |
| PersistedDomainReplay/resend | PASS | PASS | 同上，legacy new_administrator resend 原消费者 |
| ExitInventedUUID（追加小负控） | **FAIL** | **FAIL** | 0 个实际维护会话，但 Audit 宣称撤销 1 个合法随机 UUID，返回 nil |

基础 job `probes-normal`、`probes-race` 各 9 leaves、5 PASS/4 FAIL；追加 `exit-invented-normal/race` 各 1 FAIL。合计每模式 **5 PASS / 5 行为 FAIL**，两模式 leaf/result 集合相同。全部真实编译并执行；无编译/fixture/SQL准备/超时/Skip/数据竞争诊断。原始 Go 命令退出码均为 **1**，没有把负例失败包装成测试 PASS；runner 完成/解析成功只表示收据有效。

两个基础 raw 的失败 marker 分别 2 条 pointer 和 2 条 exit metadata；两个追加 raw 各命中 1 条 invented UUID 断言。独立物理 SQL 后读也确认了失败造成的错误持久状态/不完整审计，见 `leaf-schema-physical.json`，不只依据 t.Fatal 文本。

## Findings（按严重度）

### [HIGH] S410-ICR-R01-F001：保留失败 restoration pointer 缺原域终态证明

- 对象：上述当前7C24三源；[位置 recovery_execution.go:1201](D:/AID/VSCode/Codex/Han/internal/identity/recovery_execution.go:1201)。`completeFacts:1366 → reviewProof:1243/1245 → restricted:1190 → pointer`。
- 现状：查询仅核 operation.state/settlement、job.status、claim.status 标签，不读取具体失败结果/snapshot，也不验证该 invalidated 结算实际 Audit。Complete 的原 cohort 检查不能替代 finalize 后新 restoration candidate 的完整原域终态。
- 触发：合法本 run ND0 候选关联、六字段/组合FK、失效凭据和 job.succeeded 均存在；claim 为 schema 允许但原 owner 不承认的 generic failure，或原 invalidated 结算缺 Audit。明确 denied 后其余条件齐，原 cohort 合法为空。
- 实际结果：两模式各两个负控均在“incomplete failed-candidate domain proof was accepted by Complete”断言处失败；物理 SQL 见 generic_failed/outcome=failed 或 settlement_audit_count=0，同时 run.outcome=identity_ready。没有 source/provider 写权限假设，也没有通过清 pointer 或抹旧历史构造成功。
- 违反依据：design §4.2:3689–3701 的共同C/合法失败指针；TP:897–898；原卡:2207/2211。0008 reciprocal composite FK 已保证六字段关系，问题是领域响应与审计证明；不把 FK 误报为缺失。
- 影响：身份阶段提前接受不完整候选，给后继输出错误 identity_ready。本轮没有实际公共开放、生产越权或真实恢复结果的主张。
- 修复方向：原 Zeno 在当前 pointer/Complete 消费者核原 owner 对应失败路径的完整终态、响应/实际审计/job事实；复用既有分类器与不可变绑定，保留合法失败指针和历史，不改旧 owner、不授 Retry、不倒挂 S4.12/root。不同合法失败路径按各自原合同核审计，不强行发明统一新 variant。
- 证据：`raw/probes-normal`、`raw/probes-race` 的相应两个 leaf、fixture、markers及物理 SQL；合法正控两模式均 PASS，三种持久化原域 replay 正控两模式均 PASS。
- 状态/复审：**OPEN，已有独立实际行为 Red**。作者整改新 SHA 后，同一 Reviewer 对原两负控＋合法正控、原六组普通/race及实际接缝复审；不修改或覆盖原失败链。

### [MEDIUM] S410-ICR-R01-F002：退出审计结果未验证完整性和实际撤销集合

原静态疑点 `S410-ICR-R01-S001` 在本轮有效 fixture 下确认，转此 finding；不是新生产者/新范围。

- 对象：上述当前7C24；[位置 recovery_execution.go:1557](D:/AID/VSCode/Codex/Han/internal/identity/recovery_execution.go:1557)，完整分支1545–1560。
- 现状：已有同run绑定 Audit 时，只核当前未撤会话数量为0、len(m.IDs)==m.Count。未验证结果字段必填/非NULL、ID唯一性/真实性，以及该次审计声明与真实维护 Session 撤销事实对应。字段存在性本身也不足以修复语义缺口。
- 触发：同安装/run/backup/succeeded、初始强改密及真实公开 replay许可均合法，实际维护Session集合为0；既有退出 Audit 分别缺结果字段、revoked_ids=null，或明确写 revoked_ids=[合法UUID]/revoked_count=1 而库内没有对应会话。
- 实际结果：两模式的缺字段/null负控各2 FAIL；追加合法UUID负控两模式各1 FAIL，断言为“exit Audit with invented revoked UUID was accepted with zero actual maintenance sessions”。SQL 后读确认审计保留非空UUID数组/count=1，而该 schema 的维护 Session=0，receiver 已返回 nil。合法首次零会话及原receipt重试两模式PASS。
- 违反依据：design §4.2:3776–3780 要求完整、同绑定、本次实际撤销结果的退出审计和独立确认；tasks:2208–2209 的完整审计/部分事实负控。0001仅约束 metadata为object，不能替代该领域验证。
- 影响：不完整或与实际撤销事实矛盾的退出 Audit 被当作精确已完成回执，破坏退出确认与审计可信度。此反例实际会话为0，未证明任何 live Session 泄漏，因此以 MEDIUM 登记，不夸大为已发生会话越权。
- 修复方向：原作者在已有退出回读/幂等路径按原完整 Audit 语义核结果结构和实际集合/对应事实，不能只补“字段存在”或 len=count；核查真正合法非零撤销后的重试/ACK以及已有 Session 保留合同，保持首次0正向、原锁序、零新结果表/authority/生产者。
- 证据：`raw/probes-normal`、`raw/probes-race` 两负例和零会话正控；`raw/exit-invented-normal`、`raw/exit-invented-race` 的有效数据库 fixture；`raw/physical-schemas` 和 `leaf-schema-physical.json` 的真实后读。
- 状态/复审：**OPEN**，交 Main 决定整改/风险处置，不代人批准。独立复审须原3种负例及合法零/非零撤销、幂等/ACK/残留/错绑定原组，不得用拒绝所有既有退出 Audit 的做法获得“负例绿”。

两条 finding 都针对明确要求拒绝的不完整持久证据，保留原约束和公共 Runtime producer、既有原域分类器与正控；不声称当前正常 owner 必然产出损坏数据，也不假设某个 Provider 可篡改任意数据库。

## 环境、命令、进程与物理收据

独有执行根 `D:/Temp/han-s410-independent-probes-503b1861cf13`；可交付 packet 为其 `packet/`，cache/work/编译产物在 packet 外。Go实际版本 `go1.27.0 windows/amd64`，local/offline、GOMAXPROCS=1、-p=1/-parallel=1；只复制获准 D 公共 module cache，未网络/安装。没有 ambient 业务 DSN/Go配置继承；DSN仅从精确 Main resource 的 identity_dsn 字段进入获准子进程环境，不进 argv/log/packet。只读物理预检和最终 SQL 两次实际确认 PG180006、127.0.0.1:51934、han_b13d_identity/han_b13d、指定 `D:/Temp/han-b13d-main-pg-r02/data`。

7 个 owned 命令：preflight、go-version、probes-normal、probes-race、exit-invented-normal、exit-invented-race、physical-schemas；精确 argv、退出码和 raw hash 在 `commands-process-receipt.json`。重用已完整检查的停止作者 `owned_windows.py` 的 Windows suspended→Job assign→resume→Wait/Job-zero/双pump join基础设施，并补记两个实际EOF；重用归属和改动见 `runner-attribution.json`，不借作者的语义结论。

全部7命令在成功或行为FAIL后都实际 parent Wait、kernel Job ActiveProcesses=0、两个pipe读到EOF并join，无超时/io_error。kernel累计 accounting为1491个进程；optional实际观察到31个不同PID，该PID集合**不是逐child完整遍历**。全后代结束依据是7个Job的kernel归零，不是PID计数。owned pending=0，所有Rows/Tx/pool/回调随测试结束归还；没有停止共享PG或操作其他Agent进程。

四个测试 job 共建20个独立新schema；最终使用另一 owned psql 命令，对全部精确名字实际查询 pg_namespace/pg_class、run/review/claim/Audit/维护Session。20项均存在且有表/合法目标绑定；原测试log名字与最终SQL集合逐项相同，不拿打印名称当存在证明。数据、SQL和raw全部保留，无DROP/清理/移动。`leaf-schema-physical.json` 给每leaf与实际schema/状态对应。

每个命令前后 frozen24 与三源SHA均精确相同；最终 `guard-final.json` 亦同。原静态包20payload全部未变。项目go.sum与Main既存cache补全收据的SHA一致；go.mod当前hash仅另记，不伪造未采集的单独前像。没有源修复、Git写、SSOT/设计/Task/Plan写入或下级委派。原user批准与实际Reader/ACL unknown边界保留。

## 保留限制与交回

S002 的原域授权generation+1/完整linked历史正向仍未执行，未制造新的业务Retry producer。完整原64U/79P、Manifest7、ownedfmt没有在本角色本轮重跑，依Main精确指令待实际整改版本再安排；原作者/Main既有结果不能转签本角色PASS。完整root/HTTP/维护认证、真实Reader/provider/批准主体/ACL、SMTP/native restore/真实恢复/E2E/容量及人的接纳仍未验证。现有fixture只证组件软件，不倒挂未来卡或重新评审其他两个已结束对象。

仅本轮证据打包曾有一次 Python 默认GBK读取JSON的后处理诊断，随后指定UTF-8完成；未启动或重跑Go/PG、未改测试/源码、未计为行为Red，也未改变任何raw。无测试编译/fixture失败。

交回 Main：先现场接收新 packet/两finding与进程/SQL/24hash，再安排原 Zeno 修复。原失败raw和静态报告保持；同一 Reviewer 后续按精确修订复审。**本角色现在 STOP，无剩余 owned 命令，不再跑绿直到实际整改与新窗口。**
