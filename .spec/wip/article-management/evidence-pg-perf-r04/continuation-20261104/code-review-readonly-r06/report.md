# R06 readonly实际物证独立接收

**整体执行Gate：BLOCKED。两条正例的有限证据可接收；第三组已到15s注入取消目标，但资源关闭FAIL/真实exit86，不能记negative PASS；第四组NOT RUN。** 原R06静态code PASS及PROBE-R05-F001 CLOSED保留其既有边界；本次不核销原12pending，不接受完整性能/整卡PASS。

## 固定范围与核验方法

固定Han/main，指定HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。原非作者Reviewer沿有效Role/add-code-review继续，只消费 `author/readonly-pg-r06/report.md/index.json/final-closeout.json`、三leaf raw、其直接查询/时钟/恢复记录及固定R06必要源码。唯一写集为本目录；未运行Go/LOCAL/PG/SQL、作者脚本或binary，未读凭据、未源改/任务改/采用决定/Git/清理/委派。

源码只核 `author/probe-error-r06/source-freeze-r06/`：H **214370B / 550B16614224A13F31CEE931C016C3B6841F59E80F27CE2A61F24720219392E7**，五snapshot匹配；R06 binary **99F5004989B701191EEB2AF0D363410026A993A93A544442D796BFE792958A5D**匹配交接实物。原code-review-error-r06封存文件摘要不变。未读在途live H，不评未来R07，不重开R04/R05全量或18项源码。Lead七check全0属于Lead的检查结果，本轮不复跑，也不拿它替代PG运行证据。

第一轮逐leaf核命令/版本/raw sample/资源关闭/出口；第二轮只沿reader入口→collector→关闭fail-stop→外层归集/外部backend scope核证据归属与缺口。独立复算index所列**54份物证**bytes/SHA，18条查询的stdout JSON与value、wait0一致；核source/binary前后记录、实际客户端PID、时间与保留的原始错误。使用整数纳秒解析Go时间字符串，未用Python微秒截断后的四舍五入值冒精确时长。详见verify_offline.py/evidence-audit.json。

## 两条正例：真实完成，接收范围有限

| leaf | 独立核到的实际证据 | 接收判断 |
|---|---|---|
| natural-three | 真实Go PID43804、wait exit0，记录elapsed14.672s，stdout父/子PASS；三次准确size SQL **2.4564574 / 4.9604788 / 5.8407232s**，完整collector区间 **2.4569748 / 4.9605021 / 5.8407507s**。无注入，均accepted=true，bytes1992988351、AcquireNS0、acquired_after_release0、read_only=on；均在本次15s/适用deadline内。 | 接收三次自然准确采集、成功发布及本leaf正常关闭。**仅第三次自然SQL>5s**，不能称三次都>5s，也不是性能窗口或提速证明。 |
| controlled-six | 真实Go PID30192、wait exit0，记录elapsed13.500s，stdout父/子PASS；明确SQL注入6s，实际size SQL **5.9704483s**，同一collector总区间 **11.9759854s**，严格>5且<15。injection_reached=true/accepted=true，与自然样本分账。 | 接收共享collector能在同一总机会内计入受控等待+准确size并成功。不是自然SQL耗时，也不是sleep补业务采样窗。 |

两leaf各有完整argv、绝对新evidence root、source_before/after五hash及binary_before/after匹配；不是用R05旧binary运行。固定入口先设默认事务只读并校验loopback:5433/PG180006/han_perf_s24，再复用resource.collect/readDiskObservation；没有newHarness/DDL/DML。样本原resource origin保持 `2026-11-04T03:29:04.586115+00:00`。

正常收口两组均有resource_failure=false、last_complete_success对应最后实际样本、workers_joined=true、probe-pre-close acquired0、resources.failed=false、goroutine4→3。内部residual0排除了probe自身，所以另核每组真实app的组后外部空集合及最终三app精确probe，未把内部0冒全部owned0。

这两组充分支持上述有限正例，不支持5s后台tick/所有root observer并发场景、实际旧START+20s expiry、真实写增长/cap、leak/九语义或完整校准small/medium。不是要求额外盲跑，而是把已取得的事实与仍未执行条件分开。两正例成功不抵消后续关闭失败，也不跨未来新hash自动转绿。

## 第三组：到达和未到达的门

`total-fifteen-failure` Go PID **3852**，精确application绑定server PID **39712**及backend_start；外部08:50:14查询真实看到 `wait_type=Timeout / wait=PgSleep`。sample与stdout确认 `pg_sleep(16)` 已进入，注入不是在前置别处失败。

| 门 | 本次真实状态 |
|---|---|
| 15s总预算目标 | Requested/Started **08:50:13.9944131Z**，Deadline **08:50:28.9944131Z**；Finished **08:50:28.9951425Z**。实测区间 **15.0007294s**，比逻辑deadline晚 **0.7294ms**返回。目标期限确为15s，实际超界原样记录，不宣称精确硬实时退出。 |
| 取消/失败发布 | injection_reached=true，accepted=false，context_deadline；resource-close在08:50:29.0147192Z记录resource_failure=true、root_canceled=true、workers_joined=true。last_complete_success为空，未把失败当成功。 |
| 准确size SQL | **未到达**：失败发生在注入SQL阶段，未调用之后的readDiskObservation；Bytes/SQLNS/Free=0是未知占位，不是准确size=0或size SQL本身超15s的证明。 |
| 20s旧freshness | **未验证**：本leaf没有上次完整成功START；本次是首次请求15s截止，不是旧freshness截断。第四组尚未执行。 |
| 新schema/业务后点 | 入口本来没有DDL/业务写；inventory checkpoint实际schemas0。异常后第四组未启动。不能据只读零schema证明真实写harness全部后点准入已验证。 |
| 正常资源归还/关闭 | **FAIL/BLOCKED**：sample acquired_after_release=1；08:50:29.0157858Z probe-pre-close仍acquired=1。固定H1207～1210对此立即fail-stop。08:50:29.0181608Z不可覆盖记录PID3852/exit86/UNKNOWN/root canceled，stderr明确原因。 |
| 实际进程出口 | final.json的owned wait表与recovered-result均为 **86**，预期1未取得。不是外层FileNotFoundError造成86，也不是2m testtimeout或90s owner期限触发。 |

因此，接收“目标注入到达、15s取消/不接纳、真实资源关闭失败及fail-stop留证”的事实，**不接受整条负例PASS**。最终backend0是在进程真实退出后的外部状态，不能倒推退出前的正常归还曾通过；acquired1也只是一时点快照，不能单独定性永久连接泄漏。

作者提出Release/异步销毁机制作为解释，但本次没有真实Destroy分支及后续计数归零trace。独立结论限上述实际数据，不把机制假设当已定位到确定分支，也不在本次审查未来R07实现。

## 外层归集缺口保留

固定runner `run.py:84～94`先wait拿到真实code，再读取正常 `resources/readonly-no-schema.json`，之后才保存leaf result。os.Exit86先于该正常summary生成，因此读取触发FileNotFoundError，第三组标准result未落盘；这条因果链与原raw、fail-stop及final.json一致。

recovered-result为事后恢复记录：真实exit86可由原wait表/故障物证交叉核，样本与资源记录也可独立读；**argv是从实际runner模板重建，第三组前置source/hash/PID未在异常前逐leaf持久化**。总体start/final/recovery五hash和binary仍一致，但不能补称完整逐命令durable版本链。该缺口如实保留，不覆盖原异常，不把恢复JSON冒成运行前原始记录，不改成exit1。

`READONLY_RESOURCE_FAILURE`已在stdout出现，随后关闭故障发生在summary落盘前；外层归集错误不能解释或抹掉此前的资源关闭故障。后续执行记录应先持久化实际argv/hash/PID和wait，再按正常/故障分支收证；本报告不代作者修改runner或源。

## 时钟、最终scope与累计账

原outer起点 **08:44:23Z**，首读取08:44:40.3049968Z、runner启动08:49:38.988805Z，准备和等待全部计入。原stop-new08:52:23Z、hard08:54:23Z保留；实际三组在08:50:29左右已失败停止，没有在随后3秒接缝新增测试组。

已消费的外层extension文件SHA **14AC966E88EA4AF79931921973C6E12110830E4A1E54555328D83EFDC48E7563**已自行核与Lead实际文件一致：**08:52:26Z**前瞻采用stop-new **09:02:23Z**、hard **09:04:23Z**，起点仍08:44:23Z。它没有追溯授权08:52:23～26间操作，没有洗旧elapsed，也未用于失败后重跑第三/第四组。最终补查在当前外层范围内完成，不是新性能Tnew。

08:50:32旧final的apps只包含前两组，故其owned_backends=[]不能反推当时第三组为0；旧catalog的active_other=1亦保留。真正完整scope以 **08:55:28.784～08:55:29.033561Z** 的外部查询为准：SQL逐字列出三条完整application，绑定各真实Go PID、已观察server PID/backend_start，查询wait0且返回空集合。最终closeout记录 **08:55:32.973693Z**（约08:55:33Z）STOP，从原起点累计 **669.973693s**。

18条psql查询均有真实wait0，连同Go PID43804/30192的wait0及PID3852的wait86，共21个owned客户端已wait，表间PID集合相符。三app最终backend0、无可见Go/test是该时点/该scope证据，不是Reviewer现场跑probe，不声称整台主机所有PG客户端都已停。作者另见不属owned集合的psql归属unknown，不据此扩授权或操作它。

最终身份127.0.0.1:5433 / PG180006 / han_perf_s24 / default_transaction_read_only=on；起止准确DB大小同为 **1992988351B**，历史perf schema **82个完整集合相同**。原base **1406932671B**及small/旧账摘要不变，累计 **586055680B**，small剩余 **487686144B（约465.09MiB）**；余量不是可保证未来fixture装得下的承诺。最终PGDATA available **47613984768B**仅为当时观测。

## 接收结论与停止条件

- natural-three / controlled-six：接收R06这两条实际正例，限本报告范围。
- total-fifteen-failure：目标取消事实已取证，**关闭门FAIL、整体BLOCKED**，argv/前置元数据durable链不完整，不能作negative PASS。
- earlier-freshness-failure：**NOT RUN**；没有因外层extension或最终backend0改绿。
- 原6 CLOSED/12pending、旧完整性能NRU、large/canonical NRU、任务active保持；不作任务或采用决定。

已授权的R07后续以真实新freeze/delta另审，只关注原关闭边界内异步归还/有界Close及必要记录链；不加15s，不改15/5/20/clock/resource预算。当前不要求也不运行新Go/PG，不等待live源，不替作者修复。真实运行前置仍未全满足，不能据本物证启动完整性能验收。

独立证据接收完成，**STOP**。原制品不覆盖；本报告、verdict和audit固定摘要及Reviewer实际STOP见artifact-sha256.json。
