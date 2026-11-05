# R09 leak作者交接：SOURCE/LOCAL/PG STOP

仅本次明确许可的leak阶段。Go PID35312于11:33:48.598622Z启动、11:34:00.924891Z自然exit1，forced=false；wrapper真实wait exit0，final11:34:05.367464Z无problem/closeout_error、failed=false。原始FAIL为泄漏负例，不洗成测试exit0；作者机器检查满足，仍交原Reviewer独立接收。没有N9/编译/源改/清理/Git/新Tnew或自动后点。

真实达到 `synthetic owned transaction intentionally left for the closure guard` 与 `owned transactions leaked before cleanup: 1`，不是probe/seed前置失败。实读已冻结H1204 reclaimTransactions：先从ownedTx快照得1，再逐个以min(now+3s,closureEnd)的ctx实际调用Rollback，之后才进入pool归还/Close；原stdout无reclaim错误，pre-close acquired0/return_error空，business/disk/probe均acquired_after0且Close完成。**没有单独输出tx_after注册map计数或逐tx Rollback结果JSON**，不伪造该字段；回滚路径由固定源码实际必经控制流、before=1 marker和后续池/后端结果共同支持，证据强度由Reviewer判断。

共享原5s截止11:34:05.8803883Z，实际closure16.0514ms，goroutine2→2、内部residual0、workers_joined=true；resource_failure=false/terminal=nil，说明失败来自事务guard，未被资源故障污染。外部精确run-app/backend及所有observer集合owned_backends=[]/external_all_owned=[]，unresolved=[]；12个owned客户端（1测试、1离线转档、10psql）均实际wait。最后观察者排自身并另存真实wait，不冒称其自身已取得另一外部0或全Host无连接。

live前置=历史snapshot=2104252095，drift0，身份/完整旧84/activeother/累计与PGDATA门通过才建唯一minimal skipSeed schema `han_pgperf_r04_sr2leak_336a400e1247dcd8`。实际新增与inventory完全一致，旧84全留、最终85；after DB2105743039，增1490944，global1406932671、累计698810368、small余374931456；PGDATA可用45640740864B。全部增量继续旧账，未创建medium baseline。

五source/R09 nonrace71B3、原policy7C94和旧四resource摘要未变。Tnew10:22:37.932710及145/150截止保持，全量calibration/small/medium永久NRU；缺grant那次未PG、旧snapshot前置拒绝、原growth失败及恢复均保留。此时STOP，等待独立接收和Main下一N9明确许可，不自动执行。

索引：start/snapshot-observation/preflight记录grant/hash与实际live；launch/pid/actual-exit及stdout/stderr记录真实遗留与guard；material.json绑定原schema/migration/resource对象SHA；go-events/json-conversion保留Go JSON和转档wait；queries逐条身份/准确size/精确backend与wait；final.json为原始收口。execution-index.json包含全部原件SHA、资源对象及tx记录边界。
