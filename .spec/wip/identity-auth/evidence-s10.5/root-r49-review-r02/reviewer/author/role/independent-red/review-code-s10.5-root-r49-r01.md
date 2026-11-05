# S10.5 Root 四文件完整增量独立 CODE Review — r49 / R01

Reviewer：**Dewey / add-reviewer**；真实 Agent ID **`01a180fe-c3bb-7661-86fb-06af4f76ba1b`**。Lead 实际委派的原非作者 Reviewer；未再委派，未修作者源码。日期：2026-10-28。

**结论：REQUEST CHANGES，本固定版 Root 四文件不予 PASS。** 两轮审查完成。新增独立行为 finding **F003 / MEDIUM / OPEN**；另列 Main 原始项目检查失败 **F004 / MEDIUM / OPEN**，并不据静态提示宣称实际内存或 context 泄漏。

原 **S105-CODE-CLEANUP-R01-F001：按已正式采用的 r49 合同及本次独立代码/SDK-peer证据 CLOSED**，不是旧 r48 硬截止被实现修复。原 **F002：本版回归 CLOSED**。两项关闭均不能覆盖新增问题，也不构成 Domain、PG、七文件 CodeGate、生产 CLI/E2E 或人的验收。

## 1. 固定对象、完整增量与版本

固定工作区 `D:/AID/VSCode/Codex/Han` / main / 指定 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本次 objectType=code。

Main 现场冻结清单 `.spec/wip/identity-auth/evidence-s10.5/root-r49-review-fixed-r01/index.json` SHA256：
`F08B800438FDE0B96CB59C0825E7C9D581B6D173AC79A7D2472EC5BDCF654CC9`。

| 固定路径 | 本次审查 SHA256 |
|---|---|
| `cmd/han-server/main.go` | `84A2B2D114F707622D610882754CC10F806CBCE650FF4D4466FA47C31BE8BF8A` |
| `cmd/han-server/main_test.go` | `30A48E10A7FD841A9C342C0567DEBB03BFBD6F084FD65234AFCDE9C8A1AD1C39` |
| `cmd/han-server/maintenance_stdin_windows.go` | `620583ABAEB1551761359F93EAA7A4D0B79BE8A4E312B2FFD9C1D69A82663A43` |
| `cmd/han-server/maintenance_stdin_other.go` | `A711C46F5D0AD017CA54569BC8862F30EFCDF2989EA13B0C856A47F615D854D9` |

独立复制并验全部10个冻结payload。完整首次 S10.5 前像来自 `design-r01/source-before/<rel>`，main SHA `6B45B5509A2D01C9F5DF0393CA00CAD5C8430353B5F849BF6E71BA3A47C52AC4`，main_test SHA `B53DD34D57AD6A2797E9FC080FD90034FC52CB26675686CB075D3153F193A9BC`；两个 adapter 为新增，空前像 SHA `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。本目录四份 `*.diff` 是 Reviewer 独立从原字节派生的完整功能增量，未把 r49 cleanup 小 diff 冒充整个功能审查。

另保留 `main.go.r48-to-r49.diff`，基线为 F002 已关闭的 `2C2D404F344E5A8E4831C34FB4C240B234B9B099482F13897D38EA470B37FF79`。原报告内容/摘要保持不变。

沿用独立 Design R03 报告 SHA `103407164E0DFF2ADE132DF6CA321A38CB3BFCAA742C6925DBC5719F786CD66A` 与六固定 manifest SHA `478E11DB183DB40B53BF3980BF897753FD8E50331992A4B468455A8A0A4E5BF5`：ADR r04、architecture r31、feature design r49、Story 和 test-plan r23。实际六 hash 与 fixed/live 一致，副本见 `contracts/` / `upstream-and-history.json`。本轮未重新扩大审查不变的身份认证历史。

运行时消费的 Domain 两文件仅作为编译/API依赖捕获：operator SHA `A1C2039503A748F82529A01EA57E4EB14C478B9ACD34329770D7474E546D98B5`；test SHA `5923422EFB2F81518ACA07FF178DC3FFFB63310CFE2A75F034EEDA530672C31D`。不授予其独立 Domain PASS，亦未审在途 PG carrier 为固定对象。

## 2. [MEDIUM] S105-CODE-ROOT-R49-F003 — 原 D/取消已生效仍启动 authority preparation

- 状态：**OPEN**；真正修复 owner：**Root Wegener**。
- 对象：本表固定 84A2 main.go；最小位置 **4584–4585**，实际授权调用接缝 **4700–4704**。
- 依据：冻结 design r49 **2381** 行明确到 D 停止新的授权；2387 行要求取消/越期不能返回成功。现有 `maintenanceContextError` 同时检查 `ctx.Err()` 与实际单调 deadline，DB/constructor阶段已经采用它，authority阶段遗漏。
- 现状：`runMaintenance` 建立 `operation = context.WithDeadline(ctx, deadline)` 后直接执行 `dependencies.prepare(config)`，此前没有检查 operation。生产 prepare 调用真实 `identity.NewPasswordResetMaintenanceOperator`，该入口校验 trust、获取 Windows process-owner SID、执行 executor allowlist/time 授权；它不接 operation context。因此这不是仅在本地分配一个没有动作的 DTO。
- 触发：调用方已取消；或合法的显式极短 operationBudget 在配置校验后已经耗尽。配置的 `parsePositiveDuration` 接受 `1ns`，不能假设部署始终为30s。SIGTERM 也可能在进入 prepare 前到达。
- 影响：取消/原 D 后仍开始新的身份来源读取和授权检查，违反 r49 停止阶段合同；不能据后续 stdin/DB 拒绝证明该授权动作没有发生。未观察到绕过授权访问数据库，DSN读取为0，故不升级成鉴权绕过/数据写入事故。
- 独立验证：仅个人 Temp Go overlay 追加 `TestReviewerR49AuthorityAfterD`，不修产品。`already_canceled` 与 `elapsed_1ns_budget` 两叶均实际 RED：`prepare_calls=1 dsn_reads=0 err=maintenance.failed`。实际原 Root 146叶同时通过，不把这个新增反例藏进“全绿”。命令、原始JSON日志、overlay及后缀均保留。
- 最小方向：原作者在开始 prepare 前使用原 operation 的取消和实际 deadline 门禁，原 D 不重置；保留关闭 stdin、最终检查后才 self-cancel 的顺序。补已取消/已耗尽预算不调用 prepare、及时正常准备仍可继续的回归。Reviewer 未写修复。
- 复审条件：作者/Main 新四文件冻结与精确 diff，本反例转绿且正常/typed primary/unknown及资源关闭回归无损。

## 3. [MEDIUM] S105-CODE-ROOT-R49-F004 — 本固定版未通过现役 go vet 项目检查

- 状态：**OPEN，Main 已安排原 Root 作者最小修复，尚未作为本报告新固定对象复审**。
- 位置：84A2 `main.go:4584` / `:4587`；当前取消函数承载/清理结构 `:4562–4570`。
- 实际证据来源：**Main** 的 `manifest-r49-r02`，非 Reviewer 再运行。已读取原始 `runs.json` 及 stderr；命令 `go vet ./...`，exit **1**。stderr SHA `7694D7DD012118544FE786BA4E79773B48CE7A581A9D9A883654B44E0E37FAC3`，副本 `lead-vet/3.stderr.txt`：

```text
cmd\han-server\main.go:4584:2: the cancel function is not used on all paths (possible context leak)
cmd\han-server\main.go:4587:3: this return statement may be reached without using the cancel var defined on line 4584
```

- 影响：适用项目检查失败，不能从正常/race测试成功推出该检查已通过。它与 F003 的授权门禁遗漏是不同根因。
- 事实边界：现有 outer defer 在返回时读取 cancel 变量，并在最终 `maintenanceFinalError` 检查之后调用 cancel；从该控制流本身**不能断言已经实际泄漏**。finding 针对真实现役检查未过，不把 vet 的 possible 提示写成实测泄漏。
- 修复/复审要求：由原作者保持语义地组织取消所有权，使检查真实通过；不 nolint、不禁检查。必须保留最后的原 ctx/单调时刻与 stdin.Close 检查先于自用 cancel，不能以提前 self-cancel 让正常成功变 failed/6；unknown/5及原 primary 不变。Main 指定新冻结前停止本 Reviewer 新运行，不边审换版。

## 4. 原 F001 / F002 的独立复审

| finding | 本次状态 | 关闭依据及边界 |
|---|---|---|
| `S105-CODE-CLEANUP-R01-F001` | **CLOSED，限 r49 替代合同及本版 Root 实现** | 独立核原D constructor ctx、数值 primary/所有 fallback、本地literal Lookup、numeric Dial、迟到拒绝、实际 constructor/stage/pool join、粘性超期及最终单调end。独立运行真实 pgx/内存 SCRAM peer，在 C 时仍 Constructing1/stage1/socket0，最终join晚854.1019ms，全部归零，failed6/stdout空；旧硬截止反例仍存在且被如实观察。按明确采用的新合同关闭，**未称 r48 hard cutoff 修复或可抢停PBKDF2**。F003另行阻止本版整体Root通过。 |
| `S105-CODE-CLEANUP-R01-F002` | **CLOSED，本版回归成立** | clamp、limit及三deadline setter、watch/join六函数体与已关闭2C2D版逐字一致；本次重新核唯一 SDK Close winner、健康先SDK、asyncClose不重入、同C raw中断和callback实际join、CleanupDone与Pool.Close。独立normal+定向race的真实 pgx/TLS1.2 net.Pipe peer背压均1000ms，健康均0ms整数显示；无deadline放宽、raw不先SDK、SDK1/raw1、stage/socket0，通知/entry/peer/pool实际结束。CPU合作越期不豁免TLS另开5s。 |

F001 的“可能一直 join”不是被删除的未知风险：r49 明确接受不可取消 CPU/native 若永不返回则进程可一直等待。未增加S、限制SCRAM到通常4096、parent Kill、detach或伪计数。受控local Lookup人为barrier仅证明本地阶段收束，不冒 Windows OS resolver 实验。

## 5. 两轮实际检查范围

**Round 1：完整新增生产功能与测试。** 独立核 main.go 全部首次增量：imports、main分流及新增维护段3823–4795；两个 adapter 全文101/9行；main_test全部新增/变动段1–1194、4012–4017、10301–10710。检查模式精确argv、六项pre-DSN配置加后置DSN、normal recovery互斥、原D/K/C、输入有界且只EOF后授权、preDSN opaque请求、DB依赖顺序、资源owner、结果分类/输出及新增测试实际断言。F003在这个完整阶段检查中发现，不是只看cleanup小diff。

**Round 2：真实直接生产者/消费者。** 只沿下列必要接缝，不作全identity或全旧main审查：

- main分支在 `productionDependencies()` 前决策；无参server仍走原 run/runComposed，普通配置与hostname支持未改。runComposed工厂、migration、bootstrap、HTTP/listeners、Mailer、Article Journal及worker路径不进入维护调用图。测试helper显式重设测试框架argv，避免把 `-test.*` 当产品argv；没有新增生产DEBUG开关。
- Config四字段、专用constructor、ValidateRequeue opaque指针、Requeue四依赖、typed error/Disposition/Receipt的真实Domain接缝。root不做SQL/UPDATE、不复制分类器、不伪造lease、不调worker/SMTP。当前已知commit后的取消/越期在Root可把原成功改failed6，**保留原DB提交事实**；unknown5和原非零主错误保留，错误cause只在私有链。U sentinel不证明真正提交或PG原子性。
- SID来源核实际 Go1.27.0 `os/user` Windows实现：`runAsProcessOwner`→`OpenCurrentProcessToken`→`GetTokenUser`→SID字符串；没有使用 ENV 自称 role/用户名作为认证。生产trust/allowlist/epoch及签名检查交Domain，不以合成注册当真实第二SID。
- native adapter检查 GetFileType、byte-pipe、BOOL/LastError、固定kernel32 System32加载、uintptr/uint32长度、对象存活及一次Close。核心串行Peek→正available有限ReadFile、取消重查、16KiB+1上限、EOF/取消优先，不起blocking-reader goroutine或取消提交竞态。
- 独占anonymous pipe/重复读端关闭仍为冻结受保护launcher前提；OS属性检查不认证来源或检测所有dup。未声称当前部署已经落实此前提。
- 实际 pgx v5.11.0 ConnConfig/ConnectTracer/QueryTracer、pgconn primary/fallback/CancelRequest/Close/asyncClose/CleanupDone、pgxpool constructor/BeforeConnect/AfterConnect/BeforeClose/统计，及 puddle v2.2.2 的构造统计/Close完成语义，沿用并复核原同版本分析。原D覆盖派生stage；registry锁不跨IO；拒迟到/清理中CancelRequest Dial；constructing/acquired+stage barrier后结算constructor-owned对象；一次SDK Close结果latch与callback join。
- 数值host/port到标准库net literal fast path，保证本维护路径不启动DefaultResolver/GetAddrInfoW；TLSConfig、证书验证/SNI未被放宽。最终 time.Now/After 同单调域，不用 ctx.Err 单独代替真实end>C。

相关Role/完整add-code-review、checklist/abstraction/安全红线、core coding/no-hardcode、go-service coding/security、task-execution/multi-agent为此前已取得的同版正文；本次补读Role/Skill及所需规则并核摘要复用。没有重新Route或启动另一Workflow。

## 6. Reviewer 实际执行与原始结果

运行前独立确认live Root4等于冻结4；Go依赖离线、显式隔离环境，无部署ENV/DSN/真实凭据。Go overlay固定Root4及实际读取的Domain2；产品源码未改。normal额外在个人overlay追加反例；race只用原四固定文件、没有反例后缀。详见两个 `*.run.json` 与 overlay JSON。

```text
go test -mod=readonly -overlay <Temp>/reviewer-probe-overlay.json ./cmd/han-server -run '^(TestPasswordResetMaintenanceRoot|TestReviewerR49AuthorityAfterD)$' -count=1 -json -timeout=180s
go test -race -mod=readonly -overlay <Temp>/reviewer-race-overlay.json ./cmd/han-server -run '^TestPasswordResetMaintenanceRoot$/(I-S105-15_22_sdk_close_causality|I-S105-22_cleanup_callback_join|I-S105-11_22_r49_sticky_end|I-S105-22_tls_close_backpressure_1s|I-S105-22_tls_healthy_close)$' -count=1 -json -timeout=180s
```

| Reviewer运行 | 实际结果 |
|---|---|
| normal，原Root矩阵 | **160 unique nodes / 146 leaf，全部PASS，skip0**；包含真实SDK/内存协议peer及helper subprocess，不是产品CLI或PG |
| 同次独立F003反例 | **3 nodes / 2 leaf，均FAIL**；因此整命令exit1是已定位反例RED，不是Root原146叶失败，也不是全命令PASS |
| 定向race | **21 unique nodes / 17 leaf，全部PASS，skip0，无race诊断，exit0** |

计数按完整测试路径去重；四个内嵌helper同名top不重复充数。Main旧“164nodes”是含helper出现次数的统计；本报告保留160 unique/146 leaf口径，没有改变其实际通过事实。

独立normal SCRAM测得C后175.6µs观察Constructing1/stage1/socket0，最终C后854.1019ms join；local lookup人为stage最终C后1.1328ms join且Dial0。两者sticky/failed6/stdout空。TLS背压1000ms、健康0ms，先SDK且SDK/raw各1；定向race再次验证同一关闭顺序与所有通知/peer/pool完成。全部原始值见 `own-test-counts.json` / JSONL。

作者release normal/race原日志各160 unique/146 leaf、0fail/skip/race，已独立读/重数；作者Linux build及test-compile收据已核，未亲自重跑，不说Linux运行通过。作者最终证据index SHA `1D3CAA1AC73B3A5086CD1E551AD95F7BF9BCF0C28D5ECF525BDCD4909CC2C7B0`，核68payload；不是作者替Reviewer关闭finding。

Main组合/回归证据index SHA `EF239BA6E6E908820A55CCD88AFADE7FF27D4AEAF1013AC027A836073FAE6FF2`，核11payload及各原始日志：组合normal/race各305 unique节点/276叶；相关回归normal/race各619 unique/553叶，0fail/skip/race。与其309/623计数的差别同为四次helper top重复，叶数不变。四命令为Main运行，不归入Reviewer自跑；未因此重复广测试或代Domain/PG PASS。

Main随后报告vet失败并恢复原作者写权时，Reviewer立即停止新增运行。先前两个独立运行的四文件before/after均与84A2/30A48/6205/A711严格一致；本报告始终绑定旧快照，后续live或作者修复不自动纳入。

最终只读身份检查已观察作者修订产生live漂移：main.go 为 `7051777CB03BDAEF7AF3249CB01E668C482FCA3C7236BDDF38DD0D456DB01640`，main_test.go 为 `0D3F2FB0F5AD525EE1F47F4FF7B319A45465E179FC9AB177C9B03B3FF2087ECB`；两个adapter仍为原冻结摘要。**没有在这两个新hash上运行或下结论**，也未覆盖旧四快照；此处只记录观察时点身份，正式新冻结与后续修复复审由Main交回。

## 7. 限制、交回与停止

无真实PG/SQL/DDL、生产maintenance binary/CLI、网络listener、实际部署trust/签发/ACL/launcher或第二真实SID运行。net.Pipe、内存TLS及实际SDK协议peer是明确的软件层证据；U fake/sentinel不冒PG原子性、commit-ack-loss或真实Mail/worker证据。真实PG/B由另一个获授权owner承担，本报告不预测其结果。

未执行whole-repo manifest、普通server全部网络回归或Linux运行；Main所跑mod verify/vet事实按原始来源列出，未将局部Root测试冒完整检查全绿。原Design R03 PASS、历史失败包和两份cleanup报告未改；新写仅本个人Temp。

**交回原 Root owner 修复 F003、F004；等待 Main 提供新固定四文件后仅补审真实修复与回归。F001/F002上述独立关闭结论保留在精确旧版本，不随换版外推。当前 STOP，不自行修复、发布或开启下一阶段。**
