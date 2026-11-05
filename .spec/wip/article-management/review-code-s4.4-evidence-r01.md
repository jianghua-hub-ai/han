# S4.4 evidence增量独立CodeReview R01

结论：**有界 CodeReviewGate PASS；material findings 0；无待修复阻断项。** 仅针对新增 `replay_evidence_test.go` 的两leaf及其本轮真实PG证据。不是S4.4整卡、Issue、A-REC06/07全矩阵、S8.2/Case18或全恢复开流PASS，也不代替人的结果接收。

Reviewer：原非作者Erdos，以本项目add-reviewer Role执行；作者Hegel。沿当前delivery/Issue上下文，两轮独立审查；未Route、下级委派或代作者修改。技术交接与author状态来源为Lead，本报告的本人执行另列。日期2026-10-28，固定工作区 `D:/AID/VSCode/Codex/Han`，main，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本人只读核验）。

## 固定对象、依据与读取边界

- ObjectType=code；新文件此前不在原180源图中，全文482行即actual new-file diff。目标SHA256：`BE6C689273783CFCB4939EE987397E09DB175B30506D9C0E49881FA8CE83A925`。
- Lead平坦181 manifest：`C:\Users\J00010385\AppData\Local\Temp\han-s44-evidence-lead-v1ryhjrc\source181.flat.json`，SHA256 `D8EB37B52BC50CFC78C2E6F6014DAA790B9B10C1711C29D537756BAFEE7B745A`。原180与前轮180逐项相同。159Go只是Go子集；本人四次child及分析/落报告前后均核live和snapshot完整181路径集合与哈希相等，无增删或漂移。
- Go cwd：`C:\Users\J00010385\AppData\Local\Temp\han-s44-evidence-lead-v1ryhjrc\fixed-inputs181`；未启用overlay或改动snapshot。运行期迁移/fixture也取该snapshot。源码仅核hash不等于全文审全181项。
- design r22 SHA `8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E` 与testplan r20 SHA `DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA`保持固定；沿既有§3.3.1/3.4.7/8.7与A-REC06/07、§4.8.1合同，不重审设计、不创建新义务。
- Gibbs最终Prep完整实读，SHA `FC459960E2F64A3133F329697308DF4701CA3ABC3B6DB4AFAF0F67869A8BEB36`。完整Role/main Skill本阶段刷新；checklist/abstraction、安全红线、当前delivery/Issue、task-execution/coding/nohardcode/observability/tracing及唯一启用go-service规则采用前轮完整同版阅读并逐项重新核hash（19份）。
- 真实readset逐路径、hash与读取深度在 `C:\Users\J00010385\AppData\Local\Temp\han-erdos-s44-evidence-review-r01-b38f9a12\actual-readset-and-bindings.json`，SHA `2CDC36EE643BB86DF0E75CB26D6249EC5C490B90973406E3186654617E634031`；包括公开pgx tracer/Exec实现。原10来源/身份proof/其他历史测试只复用必要同版知识，不重新签原16PG、旧6PG或236contract执行。

## Round 1：新增全文与直接缺陷

完整检查唯一新增文件482行、两个direct leaf及全部s44Evidence helpers，核范围、真实故障、oracle、隐私、取消/资源/并发与失败收束，未发现可操作缺陷。

`DifferentBackupFirstBinding`（:194、:231、:269）：合法A/B inventory分别经过真实FileBackupInventoryReader/ProductionSource；两个独立MaxConns1 peer与store，两个LoadAttempt先见absent。仅Store.Begin并发，避免原共享ID/ReplayProbe并发访问。公开QueryTracer只观测并暂停首个真实成功INSERT，未伪造SQL/错误/提交结果；真实pgx Exec在c.exec后同步执行TraceQueryEnd，故首方尚未走Store后续advance/commit。首方RowsAffected1且TxStatus T，另一方独立PID和TxStatus T，再由独立observer看到真实pg_blocking_pids边，之后release/join；不是sleep猜胜负或memory store竞争。winner Created/gen1，loser必须BackupMismatch且不返回handle。公共Provider发布Ready20；loser公共ReadStatus及ReplayAndVerify都为严格Unavailable6/backup_binding_mismatch，零loser ReplayAll，最终单run/单evidence、无loser evidence、owner不被推进；稳定后wrong-backup调用前后完整SQL/xmin/ctid像相等。

可选XID按BE6实际合同审查，不采用G0早期静态“必须distinct”的说法。等待INSERT可能尚未分配backend_xid；null允许。两端都有XID时才断言不同。独立PID、两个真实协议T状态与真实锁边共同证明独立事务争用；没有宣称实测了两个非空XID。

`PublishRollback`（:374、:405、:429、:446、:463）：foreign run/evidence通过真实Repository在合法FK下提交，使用共享canonical重算其自身run的digest，预留evidence_ref仅作合成fixture。目标真实Begin和Source执行后，Store.Publish第一次必须返回errors.As可追到的真实PgError23505/`deletion_replay_evidence_pkey`以及空evidence。失败前后七表完整像（含xmin/ctid、owner token单独内存比较）不变；公开ReadStatus为pending/replay_in_progress且不写。独立observer NOWAIT成功取得run锁再rollback，主池仅剩预留observer连接。然后同store/handle/request/input取下一新ref成功，gen/token/backup不变，不另Begin/Replay；foreign原行保持，只有一份目标新evidence，current/input/status原子绑定并重复只读稳定。

该23505发生在evidence INSERT本身，不是在INSERT成功后的current UPDATE；注释及错误证据都保留此局部边界。两leaf没有伪造authority/SQL状态、placeholder/skip或扩大旧备份真实性。

## Round 2：直接消费者、状态与数据合同

沿真实Store Begin→INSERT/ON CONFLICT→FOR UPDATE→backup比较→generation/owner→commit，与Publish Refresh(事务外)→run锁/owner→history→evidence INSERT→current更新→commit/rollback检查。0003 run复合PK、backup immutable、evidence_ref PK、语义unique、FK与0009 owner CHECK未改。ON CONFLICT只吸收(run,namespace,version,digest)，foreign不同run的同ref冲突确实落到PK23505；不会被history复用或语义DO NOTHING吸收。失败不绕过真实事务，fresh-ID retry没有借foreign证据。

SourceB读取独立manifest并验证backup，使用同Journal受控端口及真实Finalizer/proof委托/fact reader，不复制业务资格或构造另一权威。旧B65 fixture实际先完成真实三类删除，再形成0/3结果；共享decoder/CanonicalDigest只作既有权威，另外有独立SQL字段、operation/audit digest/count oracle与固定0/3预期。本人再次从保留像以独立Python重算Ready first16、operation/audit摘要，核6份Ready/wire/SQL/Facts10及current绑定全部一致；没有只数叶名或让被测算法自证。

取消/资源：barrier可由release或ctx取消唤醒；清理先open、cancel、join两个Begin，再关闭额外peer池，最后释放observer/cancel主ctx/检查acquired0/关闭主池。Rows有Close/Err，事务错误返回有rollback，NOWAIT事务明确退出；本轮正常/race均实走这些收束断言/日志。未额外注入取消/超时故障，不把通过日志扩成每条失败分支的独立事件trace。未发现权限/隐私/资源新增缺陷；attempt_token仅内存完整比较，SQL影像屏蔽token/password，未记录DSN。

## 本人独立执行与原始证据

本人证据根：`C:\Users\J00010385\AppData\Local\Temp\han-erdos-s44-evidence-review-r01-b38f9a12`。仅本人新runner/preflight/分析和本报告有写入；无源码/测试/TaskPlan/design/Issue/root/identity写入。每child前后完整181 live/snapshot守卫。runner按最小非敏allowlist构造env，离线四env固定；PG children仅从既有User registry取得DSN并注入child env，不读Process DSN/其他PG值、不写文件/argv/日志、不改User/System环境。vet无DSN。

本人独立只读preflight先验证pgx主及全部fallback为literal loopback:5433、实际server_version_num180006、actual user与配置匹配并正常关闭连接；fixture每次CREATE前重复核实际endpoint/version/identity，CREATE独立autocommit后立即记schema，随后0001..0009迁移事务。无DROP/角色/GRANT/新实例/服务。

实际argv（cwd均为上述固定snapshot）：

```text
preflight: go run C:/Users/J00010385/AppData/Local/Temp/han-erdos-s44-evidence-review-r01-b38f9a12/preflight.go
normal: go test -tags=integration ./test/integration/recovery -run ^TestArticleReplayEvidencePostgres$ -count=1 -timeout=8m -json
race: go test -race -tags=integration ./test/integration/recovery -run ^TestArticleReplayEvidencePostgres$ -count=1 -timeout=8m -json
vet: go vet -tags=integration ./test/integration/recovery
```

普通与race各exit0：**1top、2组、3run=3pass、2leaf、0fail、0skip**，无重复节点，两mode终端集合逐名相等：

- `TestArticleReplayEvidencePostgres/DifferentBackupFirstBinding`
- `TestArticleReplayEvidencePostgres/PublishRollback`

tagged vet exit0、stdout/stderr空；preflight exit0。全部4个本人child真实waited、无outer timeout、无本人遗留child；没有运行其他integrationtop、旧16/旧6/contract或全Manifest。

| 本人run | exit | stdout SHA256 | meta SHA256 |
|---|---:|---|---|
| preflight | 0 | `968C2DA704E1C1A02B4D81ECE6F93A82F3174CFCF8ECC680B83AED0741DA1CBA` | `F0B7C7F8CCBD0674AB955400FC063C0A6601A33E6F026FED2E4F919F6D0444C4` |
| normal | 0 | `1B6BF3DD138A85FCBA8E04778C1B7AE20FD65FD8F5002EE931507D48A8B1A264` | `3AF24ADC732CA834FAFC9424AEBDEDA6D9046FF2A31B3D64A28104C7A42D3840` |
| race | 0 | `CBBE207D411F54856EDFDCC246DF7165EA1BC3586C487E2A305778EA9B1B4766` | `DDBC9EC200F696772F89DEC45165D784A0F4F47B184154032A55B9F8ED741A8C` |
| vet | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` | `06409151702A8EA1326AD4560F4231CD2C7A0F4DAF7D8B2A787524779B8769BD` |

四份stderr均空（SHA256 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`）。完整argv/PID/起止/exit/rawSHA与before/after181图在各mode/meta.json；raw/meta索引 `C:\Users\J00010385\AppData\Local\Temp\han-erdos-s44-evidence-review-r01-b38f9a12\raw-meta-index.json`，SHA `370D492DDEE04985C72A9793E28F5DF89FE1C507917E4B2347D2D7265D3D1703`。

本人实际阻塞观测（两端TxStatus T为固定源码执行断言；下表PID/XID/边为实际保留JSON）：

| mode | blocker→waiter PID | winner/waiter backend_xid |
|---|---|---|
| normal | 66164 → 67856 | 28851 / null |
| race | 54260 → 50832 | 28885 / null |

本人random retained schemas，共4个，与作者4个隔离；每case9迁移成功、CREATE日志与case绑定：

| mode | leaf | schema |
|---|---|---|
| normal | DifferentBackupFirstBinding | `han_s44red_820685c21a8fba72dcc903bc` |
| normal | PublishRollback | `han_s44red_e44a19ecff823e9f5eb0de56` |
| race | DifferentBackupFirstBinding | `han_s44red_68f5ab0cc4e0d106521c5c8d` |
| race | PublishRollback | `han_s44red_3a79827ffb43a10e55062cb5` |

每mode额外两peer池关闭；主fixture observer释放、ctx取消、pool acquired0并关闭，worker join由实际断言与closing日志支撑。23505后NOWAIT unlocked、pool acquired1=observer1见每case resource JSON。完整owner等值由源码绑定且通过的reflect.DeepEqual/token比较证明，未额外落token值或伪称有独立token事件日志。

SQL/文件索引 `C:\Users\J00010385\AppData\Local\Temp\han-erdos-s44-evidence-review-r01-b38f9a12\sql-artifact-index.json` SHA `1A47DD75225C0C0377AEA496D02CAB12588F9D8801475EB4C7A1E81666E44F3F`，含58份保留payload各自SHA、schema/leaf/目录；6份Ready/wire/sql/facts、完整before/failed/retry像、blockingPIDs/optionalXID、23505constraint、foreign provenance、pending/failed envelope、NOWAIT资源证据均核。独立重算与逐case结论：`C:\Users\J00010385\AppData\Local\Temp\han-erdos-s44-evidence-review-r01-b38f9a12\own-run-sql-verification.json` SHA `BDCF49E88744C7ED66FE302CAEB9E0710F83D93F5EF77EA5C2C5497A684BD082`。失败前后完整像相同；retry前后业务五表/foreign run与evidence含xmin/ctid不变，仅目标ready/evidence切换符合预期。最终181守卫：`C:\Users\J00010385\AppData\Local\Temp\han-erdos-s44-evidence-review-r01-b38f9a12\closeout-source181.json` SHA `9295D46144F607B3525A5C557CE2A9E8218EF93402997046CD27DF643E4AD23F`。

## 作者/Lead证据归属与未验

Lead随后交回Hegel stopped的执行包 `C:\Users\J00010385\AppData\Local\Temp\han-s44-evidence-run-c88bf4eed95b4dd2ad34ae521fef8385`。本人只读核其evidence-index列24份run/meta文件SHA及4case共58份payload SHA，逐行解析作者普通/race各3/2/0fail/0skip；两mode source181 before/after均等于本轮固定图，23505前后像相同、实际阻塞边存在。此为**作者证据核证**，不是本人上表运行。记录 `C:\Users\J00010385\AppData\Local\Temp\han-erdos-s44-evidence-review-r01-b38f9a12\author-evidence-verification.json` SHA `D3547071CD116824B8A8CCDFFDA98313616E244CC2D91724F4B5E1E50718B68F`；作者schema不计入本人4个。Lead告知181 Manifest7全部0/format空，本人未机械重跑，不把Lead执行算本人命令或声称全部profile工具均已执行。

未验证/保留：真实backup镜像和旧备份恢复、真Unix FileJournal/HMAC/keyring/POSIX、真实网络/OS故障、INSERT成功后current UPDATE失败、完整A-REC07/其他旧矩阵、应用role ACL、inventory真实签发/只读挂载、双namespace/global opening、retention/RPO/RTO、S4.4六requires及完整关闭条件、S1.5十二requires、S8.2/Case18及人的accepted继续原hold。fixture真实PG+文件reader+Source/Store/Finalizer/SQL fact reader并不使controlled Journal/synthetic inventories或foreign fixture成为durable真实备份证据。没有把本片PASS改成全恢复/整卡关闭，也未替人签收。

无findings、无新增方向待决定；不以风格优化要求返工。正式报告仅此授权路径新增，原报告和原证据均保留。交Lead核实际报告/源绑定/本人raw并在其权限内收口；本人**stopped**，不关闭Task/Issue、不委派或启动下一增量。
