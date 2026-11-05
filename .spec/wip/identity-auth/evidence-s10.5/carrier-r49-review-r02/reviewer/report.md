# S10.5 r49 Carrier 原 finding / 必需层独立复审 R02

**结论：NO FINDINGS WITH LIMITS。原 MEDIUM `S105-CODE-CARRIER-R49-F001` CLOSED；C18 真实 worker/CLI 并发、C16/F003 精确 UTF8 JSONB 字节边界两项既有必需层均已补齐。新增 finding 0。** 两轮有界补审完成；没有确认另一个需交作者补修/补验的软件必需层缺项。此结论不签发全项目 CodeGate、全部检查 PASS、Case/Task/E2E 关闭或人的接纳。

Reviewer：原非作者 Laplace；未修作者代码、未委派。工作区 `D:/AID/VSCode/Codex/Han`，main，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本次仅写当前新 retained Temp，旧 R01 和正式原件不改。日期 2026-10-28。

## 固定版本与实际范围

唯一变更对象 `test/integration/identity/password_reset_maintenance_test.go`：**`D4B0C837F27E2654784C779A3C08065F0CE4ECCE78B4ED63072B69125DEA2204`**。另六源码与 R01 相同：main `833D3E1E…`、test `4EF46AA9…`、Windows adapter `620583AB…`、other adapter `A711C46F…`、Domain operator `A1C20395…`、test `5923422E…`；完整 SHA 见 `source.initial.json` / `source.final.json`。

直接从 durable R01 `role/final-r01/carrier.final.go` 的 D452 原字节与新 live/frozen 字节独立生成 diff：85427→95658 bytes，1858→2042 LF 行，**5 hunks / +184 / -0**。本人的 diff 为12434 bytes；与 Main12440 bytes 的头部标签不同。作者 `carrier.D452.old.go` 的 CRLF 归一化后与真实 D452 完全相同；作者190820-byte diff不用于界定改动。除 cleanup 注册、两个边界测试和两条 worker 并发分支外，原1858行不重复完整评审。

作者新 seal index SHA **`864966663D9CACA10C5DFA1187A8F5505B6F58501AB0A854DEA74E88CFF2EDCD`**，root 为 `carrier-r49-f001-c18-f003-r01/` 父目录。本人独立核全部1072 payload，零不符。durable `pg-b-author-r49-r02/index.json` SHA **`5CC384886105AF867D31665FCB19334BF73789CBAC8A78859F04425C3F410A81`** 的1071 payload及3个 retained binary引用亦全符。Main onsite SHA `1178E4ED262BDD87CD7A9F8D17A7858F1B9679899D3AA2847FAE5737B939F9FA`仅作为 Main 核证，不冒本人运行或独立结论。

已核 source7/modules 与188源码库存；历史到新 freeze 只变 carrier，其他187项不变，当前全部匹配。17份同版 Role/Skill/Rule/go-service 文本摘要一致，复用既有有效规则；必要正文和安全/检查清单已补读。不重新 Route、不全库预读。

## 两轮实际检查

**Round 1：直接缺陷。** 全读真实184行增量及必要上下文，核 cleanup 注册/正常 prompt Close、Fatal 失败收敛、四个 overlay 的真实基线与故障点、测量发生在 rescue 前、两条 worker barrier 的资源和 goroutine 所有权、同步 release/join、真实 PG 字节测量和 guard/Get oracle。未替作者修复。

**Round 2：真实消费者与合同。** 沿未变 Root 的 prepare/Validate/Requeue→typed error/空输出→cleanup/final error 顺序，Domain 的 User-first 锁/ReadCommitted/global UUID scalar guard/text Get/strict/replay，实际 audit.Store 与 worker.Reconcile 的 Scan/Claim/Renew/settle/Audit/complete 链，核新增证明是否可达且有因果。只补读必要范围，并以原 R01 完整审查和当前 Root R02/Domain 精确版本桥接未变部分。上游六 manifest仍为 `478E11DB183DB40B53BF3980BF897753FD8E50331992A4B468455A8A0A4E5BF5`，依据 r49设计/r23测试计划，尤其576、670–676行；未添加产品合同或第八源码。

## 原 finding 与必需层处置

| 对象 | 独立核验及结论 |
|---|---|
| **S105-CODE-CARRIER-R49-F001 / MEDIUM** | **CLOSED**。carrier710–714：创建成功后立即 `t.Cleanup(pool.Close)`，位于任何 endpoint Query/metadata Fatal之前；正常713后仍立即Close。实际安装 pgx Close 的 once保证重复Close安全。四个作者真实PG只读探针分别为旧/新×Query无行/metadata版本不匹配；四命令均真实exit1，未称go test PASS。测量cleanup按LIFO在新Close之后、任何rescue之前执行：旧两例各TotalConns/backend1、CleanupDone0、Acquire未拒绝；新两例各0/0、CleanupDone1、Acquire拒绝。其后rescue及observer Close/Done全部实际join。overlay替换同一既有carrier，无新产品源文件，DDL guard在schema语句前。原缺陷、旧失败和旧报告保持历史事实。 |
| **I-S105-18 真实并发必需层** | **SATISFIED**。carrier941–951、1670–1794；prepared/not_started与recovery_required/accepted均用现役真实worker.Reconcile及真实CLI。首次合法approval将job重排，worker实际Scan/Claim/Renew建立attempt3有效lease；Audit适配器先调用真实Append，再在持有User/operation/job事务锁时屏障暂停。独立observer以CLI application、Lock、`pg_blocking_pids`包含worker PID及实际User FOR UPDATE query证明两个不同backend确实重叠，然后才释放。worker与child均同步join；两轮四例分别落invalidated/succeeded终态，job succeeded/attempt3，1job/1requeueAudit/1workerAudit，stale CLI exit4/空stdout且staleAudit0；原approval fresh replay0且八表图像不变。worker permit各Acquired=Released=4、Current0，选定CLI backend归零。没有把仅启动两个进程当并发，也没有新安全结果classifier。 |
| **I-S105-16/F003 cap/UTF8 必需层** | **SATISFIED**。carrier1432–1474、2033–2041；合法Event绑定通过现役audit.Store.Append提交至append-only表。含128组多字节“界”和引号/反斜线/换行/tab膨胀，PG先测render再以ASCII补齐，最终从存储行重新测 `octet_length(convert_to(metadata::text,'UTF8'))`。两轮各65536与65537；实际字符数65280/65281、value UTF8 bytes65009/65010，避免拿字符数或Go Marshal长度冒充上限。真实Domain/PG recording adapter：guard1，65536 Get1后strict拒未知成员，65537 Get0；update/append0、ReadCommitted、八表图像不变、fixtureAudit1。domainCall还核permit恰1次归还。这是r23明确允许的界内strict拒绝，不伪造合法重放成功；小合法13string/精确int64往返继承原R01已读实证。 |

失败/提前返回路径中，worker release使用sync.Once，defer释放屏障并实际等待worker/child，正常路径也先join再检查状态和permit。8s ctx、原child watchdog及poll是失败保护，未引入旧硬C进程完成承诺；无detach、以Kill冒成功或越期重开业务。网络/TLS cap没有改动，不以CPU合作期限例外放宽网络。

## 实际验证及层次

本人逐条解析新 raw RUN/PASS、cases/11prefix、全部286 observation及其对应stdout/stderr，核fixture/build/source map、新C16/C18和ACK因果proof；内容读取和纯hash库存分开记录。

| 执行归属 | 实际事实 |
|---|---|
| 作者 normal | 126 unique节点 /114叶，fail0/skip0；143 child Wait/writer join，Kill0；11必需prefix非空 |
| 作者 race | 同126/114与143 joins、Kill0；无race diagnostic，race仅父harness |
| 新fixtures | 共14个保留新schema，每轮7个，各9迁移、PG180006、identity匹配、批准数值loopback5433；不是migration.Runner升级/回滚或生产部署 |
| 实际child | 两轮均SHA `17B897979630006DE8C8B5B643CFD786E0AA65DC4AACEB9AEFC7AA03FD4DFDF1`；真实build argv无-race；前后181项build map相同并匹配现场。carrier测试不进入产品binary，故child同SHA有明确来源 |
| ACK继承/新运行回归 | 两轮真实COMMIT advisory waiter→先arm opaque discard再放锁→observer已提交job/Audit→断开；各discard18bytes、unknown5/空stdout→fresh replay0/noextraWrites，listener及两pump/child均join。两轮均negotiated_TLS=false，绝不称TLS ACK验证 |
| 本人唯一项目命令 | `go vet -mod=readonly -tags=integration ./test/integration/identity`，离线GOPROXY/GOSUMDB=off、GOWORK=off，exit0，1.953s，stdout/stderr空；运行后固定hash仍相同。未自跑go test、PG、DDL、CLI/listener或读取实际DSN |

四个Fatal探针是作者原始失败路径证据，exit1保持exit1；资源oracle的旧失败/新成功不重标为整条go test PASS。Main没有新PG运行，本人也没有；本报告是对作者实际执行、实现和oracles的独立复核。

## 精确软件整体结论与保留边界

这次通过真实连接链桥接整体：Root同ctx/opaque授权和错误输出 → Domain User-first/完整before/一次job重排与同tx Audit → 实际PG schema/guard/text Get精度 → 原worker合法Claim/Renew/结算及并发 → immutable原approval重放与空失败输出。C18实证了这个链在另一真实worker持锁时的顺序与归还；C16实证了真实数据库表示到既有消费guard的端点；F001补齐了测试fixture在失败退出时的Close所有权。整体判断建立在接口、状态、事务和错误结果一致性，**不是把多个局部PASS相加**。

原carrier完整R01 report SHA **`93C7B130A5E503B8A676B83A411F800F7850F34F1B72D32EF6A4DC7C114F9EAB`**原字节继承；本R02仅更新F001、C16、C18及对应C22资源缺口，其他逐Case范围仍引用其不可变表。Domain report **`385BA0CBD60F57BD2C24B84F969C59392176446FD6679F75DE79595CD2B0A57B`**覆盖同A1C203/592342。Dewey Root R02 report **`56427E3CC2668AC15537A786EEEFE4D0BF5C373527470AAC1B3FEB07EC6B0985`**、index **`D84FF282DC9045B66B6FD2F4AB71036A88660A9F5D446D6ACC1A2BEC42231E58`**及164发布payload再核一致；以其完整R01+精确修复链覆盖当前Root4，F003/F004关闭及F001/F002必要回归结论明确归属Dewey，未代签或广重跑。

因此，当前冻结七文件的软件实现、相关消费者及已约定U/PG/B证据范围内，**无已确认剩余阻断代码finding，原F001关闭，两项明确必需缺层消除，整体软件独立审查 NO FINDINGS WITH LIMITS**。未发现另一项应交原作者补验的本轮软件必需层。C12无关行比较仍为原非阻断建议；不将每个C14/C20的U子例强做B矩阵，也不把本报告等同全部Case验收。

仍明确保留：TLS-enabled ACK loss未验；实际child无race instrumentation；未直接走reset/resend HTTP并发（本次按现役worker API+B完成C18，不作普遍接受竞态声明）；八表图像不是所有SQL尝试的全量trace，精确noextraSQL仅在静态路径和具名adapter计数范围成立；没有全Go runtime/OS线程枚举证明；没有真实部署launcher/ACL/密钥签发、第二真实SID、撤销排空/恢复/新epoch演练、发布/E2E或人的验收。

Main原manifest index `06D7F466F0FCB54B4FA0C4BF1DD5325E97AB12E91963A062F4CF0882E64416F6`所载六条非format命令exit0是原时点证据，不冒当前carrier重复广跑。当前source7 scoped gofmt0/空输出，但**全workspace `gofmt -l .`原实际exit2仍保留**（immutable证据probe/空preimage）；本次未修/删除历史证据，不宣称全部七命令PASS，不签发完整CodeGate。整体接纳和后续授权由Main/原责任方按明确范围处置。

**STOP。** 返回Main；不继续修代码、写Tasks/Plan或推进下一阶段。任何后续源码变化按实际新SHA再界定复审。

证据：`report.json`、`carrier.precise.diff`、`payload-hashes.json`、`durable-payload-hashes.json`、`independent-runtime-inspection.json`、必要raw/proof副本、`readset-evidence.json`、`readset-code.json`、SDK/Goexit摘录、own-vet收据、版本及最终hash清单。hash核验不冒全文审查；原链只引用不复制整套旧材料。
