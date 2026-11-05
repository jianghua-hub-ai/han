# Article S4.8 F001 — 独立 CODE Review R02

**结论：有界 CodeGate PASS；ARTICLE-B13D-S48-CR-R01-F001 CLOSED；new0 / open component finding0。** 原 Dirac / add-reviewer，非作者；此结论基于本角色两轮静态复审和本轮实际独立 Go/ArticlePG，不以作者/Main自测代替。

## 原 finding 与修复判断

F001 原为 MEDIUM：Publish 在实际 SQL 回滚后返回非零、未提交 evidence，新 PG helper 丢弃该返回值而漏原消费者断言。R01真实失败 raw/报告保留，其原 ref/digest 结果与本轮新结果可逐一对照，未覆盖或重写。

Round1 完整审阅41行 `source2.diff` 及受影响正文，独立核 before=R01、after=live、两源 inverse 精确还原其余全部旧字节、八源 SHA 原样。`replay_publication.go:873` 先登记清零 defer，`:878` 后登记原事务最终收口；LIFO使 Rollback/错误合并先完成，`:921–924` 再观察最终 result 清空失败 evidence。错误cause和unknown标记原样；nil error的健康/胜出 evidence保留；不recover panic，不改SQL/公开接口/原算法。`replay_publication_integration_test.go:1379–1384` 恢复整个零结构体断言，所有原回滚/owner/首份/事件顺序/解锁断言未改。

Round2 沿真实 Store.Publish→publishPublicationRows、finishReplayPublicationTx，以及 provider.publish/tryReuseAfterUnknown 追错误消费和真实事务时序。err分支不把返回 evidence当证据，先独立精确读取current/evidence/publication，再决定Fail/重试；归零不触发提前Fail、Replay或盲写。对应真实Commit成功后ACKerror、历史/胜出复用、双revision/无权动作拒绝/缺Legacy只禁Import等原5组，已在本轮普通/race中实际命中。原R01完整10源及157k差量的未变范围按精确SHA复用，未机械重复整库审查。

当前两源固定SHA：

- `internal/article/replay_publication.go`：`34B698A231B32101ACBF1B91AB7843FF1710D529C748072CFDF27BA1CF180B8E`
- `internal/article/replay_publication_integration_test.go`：`8F5DD24DD40BE6A1A6CD4FBB9EF5990EE7EDBCEB964FF9ACF6BD41E5967540FD`

完整10源及24冻结对象精确SHA、main/HEAD6da8abb…入口/退出独立核对均一致，见 `source10-entry.json`、`source10-exit.json`、`frozen24-{entry,exit}.json` 与 `head-{entry,exit}.json`。Identity/Restore八源只做本轮要求的冻结hash保护，不签其语义Review。

## 本角色实际执行及原叶保护

| 范围 | 普通 | race |
|---|---:|---:|
| 原required U/C：新23＋owned旧31 | 54 PASS | 54 PASS |
| 原required P：新52＋owned旧6 | 58 PASS | 58 PASS |
| 原Dirac三叶probe：Healthy/CurrentUpdate/PublicationInsert | 3 PASS | 3 PASS |

上述原旧/新名字逐名匹配R01，普通/race终态map一致；**0 fail/skip/unfinished/race诊断**。每模式5组 U/C 为11/8/1/1/2，P为13/1/8/2/28：PublicationCurrentEvidenceAtomic、CompositeFKImmutableCanonical、ExactHistoricalClosureAllowsRequiredReads、HeadAdvanceAndReadDrift、LegacyReassertionVersusTrustedImport。不是把计数相加当集成通过：直接用当前真实Store/provider/Runtime安装与同handle控制核心/SQL消费者的原断言及下列独立实物oracle核合同。

原fragment全字节SHA `E8DF4FB70E7C00D2F6D7946B8791208B23392845AEACA6FFB9AEB29898A28084` 未改，test-only overlay始终是最新整个8F5DD前缀＋原完整fragment，没有production overlay或替换旧断言。普通/race各两条实际SQLSTATE23514均在evidence INSERT后触发，九表rows/xmin/ctid精确回滚，NOWAIT证实run锁释放，返回 `ref="" digest=""`；健康三表原子发布实际成功。原postCommit测试记录两模式均为 **ActualCommitNil → SyntheticACKError → ReadCurrent → ReadEvidence → ReadPublication**，实际独立observer核精确三者，重放/候选/元数据计数未增加。synthetic是ACK响应/外部facilities，真实Commit/Runtime/core/parser/SQL按相应原测试边界执行，未冒充真实批准ACL或native正控。

freshD物理预检独立PASS。仅本角色新增 **122 schema，122/122逐名namespace SELECT＋九表实物/行数SELECT（1098表读）**，Rows关闭、acquired0；全部保留。原C122和作者D128名单不混入，不运行旧 `TestDiracA48RetainedSchemas` top，不glob/DROP/清理/停服。`physical-schema-data-actual.json`、`schema-origins-actual.json`、receipt-only SQL fragment和schema-receipt raw可核。

共8条本角色Go命令实际exit0/父Popen.wait返回；**272个own job进程记录，272个已取得handle均实际Wait，创建/退出FILETIME与OS通知保留；8个job全部ACTIVE_PROCESS_ZERO，stdout/stderr EOF后reader join**。不以PID消失/父exit0代真join，不声明Main PG/其他Host/另角色已join。OS记录另保留12条race子进程exit1（每次race四条），不冒称全子进程exit0；未另分类其子工具原因，Go父全部0且产品JSON无失败/未完成。详 `known-owned-pids-actual.json`、`checks/*/processes.json`、`child-exit-diagnostics.json`，不是新增产品finding或额外Gate。

全部串行onejob，GOMAXPROCS1/-p1，offline/local/modreadonly；TEMP/TMP/GOTMPDIR/GOCACHE/GOPATH/GOMODCACHE/work/fixtures均D，固定32个D module输入逐hash复制。普通DSN空；PG只读取Main当前resource的article_dsn，目标freshD127.0.0.1:51934/han_b13d_article，不读取或操作Restore lane。未改repo/source/SSOT、Git写、服务配置，未下载/清理/子委派。

## 接收范围与停止

本结论只接收当前Article S4.8 actual10/F001修复的组件CodeGate。独立vet/fmt/Linuxcompile、全局7、旧Issue两个未授权PGtops、native具名正控、真实provider/批准ACL、完整S4.13/root/S4.16/E2E及人的接纳仍NOT RUN/unknown；作者历史证据仅留来源，不扩成本轮PASS。功能小fixture与原static成本判断不宣称capacity/P95/大历史性能测量。下游未验不倒挂成组件新前置。

**STOP。交Main接收；本角色不Taskclose、不签G4/root/native/E2E或人批准，也不启动Identity新目标。**

本无cache交付包：`D:/Temp/han-b13d-a48-r02-independent-review-f6d45add64`。原D scratch/cache/module/work/fixtures仍保留：`D:/Temp/han-b13d-a48-r02-review-prep-a7cde62f78`，未移动或删除。
