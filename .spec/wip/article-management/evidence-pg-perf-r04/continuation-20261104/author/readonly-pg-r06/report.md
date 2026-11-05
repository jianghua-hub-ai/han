# R06 只读 PG 交接：两个正例完成，关闭门阻断

**SOURCE FROZEN / LOCAL / PG STOP**。未启动第四组、未重跑任何正负例、未改源码/任何freeze/旧证据、未Go重编、未DDL/DML/新schema/Tnew。只使用获准R06 binary；五source与binary前后仍固定，原base/ledger摘要核不变。原Reviewer Code PASS、Lead采用和本次执行结果分别记录，本报告不是独立执行Review或整卡PASS。

授权起点 **08:44:23Z** 不变。首条读取末尾现场时刻08:44:40.3049968Z；准备计入outer，runner启动08:49:38.988805Z。实际采纳 `readonly-window-r06-r02.json` 的09:02:23Z停新组/09:04:23Z硬停，没有新建或滚动10min窗口。最后只读外部probe在 **08:55:33Z** 完成释放；从原授权起点累计约670s。本次没有因extension继续越过已发生失败。

| 模式 | 真实进程/出口与实际结果 |
|---|---|
| natural-three | Go PID43804，exit0，child实际14.672s。原准确SQL三次 **2456.4574 / 4960.4788 / 5840.7232ms**，每次bytes=1992988351、Acquire=0ms、accepted=true。正常关闭及组后外部backend0。 |
| controlled-six | Go PID30192，exit0，child实际13.500s。注入6s SQL另记，真实size SQL5970.4483ms，collector实际区间 **11.9759854s**，满足>5且<15；accepted=true，正常关闭及组后外部backend0。不是SQL提速或自然耗时指标。 |
| total-fifteen-failure | Go PID3852，**真实exit86，非预期exit1**。注入已到达；server PID39712观察到PgSleep。请求08:50:13.9944131Z的本地等价时刻对应deadline恰+15s，实际finish为+15.0007294s，accepted=false、context_deadline；预算目标到达，但probe关闭门失败，整leaf **BLOCKED/FAIL**，不能当负例PASS。 |
| earlier-freshness-failure | **NOT RUN**；第三组异常即停止依赖活动，没有第二次注入或旧START+20s真实验证。 |

上表第三组UTC时刻应按原sample的`+08:00`转换：Started=08:50:13.9944131Z、Deadline=08:50:28.9944131Z、Finished=08:50:28.9951425Z。它在注入`pg_sleep(16)`阶段取消，size SQL未到达；Bytes0是失败/未知占位，不能当数据库大小成功观察。三组分别是独立Go进程及绝对新增evidence root。

## 第三组直接原因及归集错误关系

1. sample明确 `injection_reached=true / accepted=false / acquired_after_release=1`。`resource-close` 于08:50:29.0147192Z记 `terminal=context deadline exceeded / resource_failure=true / root_canceled=true / workers_joined=true`，没有last完整成功值，因此本组不是20s freshness反例。
2. probe-pre-close于08:50:29.0157858Z记 **acquired=1**。固定H `closeProbe` H:1203起，遇非0快照立即调用resourceFailStop；08:50:29.0181608Z不可变failure记录的PID=3852、exit86、reason=`probe still has acquired connections after worker join`，UNKNOWN/inventory保留。不是外层强杀，不是testtimeout2m或owner90s到期。
3. os.Exit86发生在正常closeProbe返回前，正常最终 `resources/readonly-no-schema.json` 没有生成。外层runner先读取这个正常路径才保存leaf结果，遂FileNotFoundError；该归集错误没有造成前面的exit86，却使第三组标准result未落盘。已按原raw/最终真实wait记录补 [recovered-result.json](total-fifteen-failure/recovered-result.json)，保留原runner和异常，不重写为exit1。第三组完整argv只能由实际已执行runner模板重建；逐leaf前置hash在代码中已核、未在异常前单独持久化，不能冒充完整durable命令链。首操作和最终五hash/binary实物仍一致。

与实测相符的机制解释（不是已取得的动态销毁trace）：已安装pgx v5.11.0 `pgxpool/pool.go:744` 的Exec通过defer调用Release；`pgxpool/conn.go:18` 的Release在closed/busy/非idle时调用Destroy；puddle v2.2.2 `pool.go:72` 的Destroy另起goroutine，`pool.go:687`在destructor完成后才移除资源并释放计数；pgxpool `pool.go:314` destructor还等待CleanupDone。这意味着**Release已调用不保证AcquiredConns立即归零**。本次是否恰经过该Destroy分支、计数本可在多少ms内归零，仍unknown；不能把一次快照断言为真实泄漏，也不能用进程死亡后的backend0倒推关闭门曾成功。

最小建议待Main另授新revision：保留15/5/20及现有关闭边界，在原可用关闭deadline内确认probe/diskPool归还与异步销毁完成，再作fail-stop裁定；不删除acquired检查、不sleep凑成功、不无界Pool.Close、不额外赠15s。补真实timeout后有界归还与持续不归还的区分证据。outer helper应在spawn时先持久化argv/hash/PID、wait后先保存exit，再按正常或fail-stop分支读取物证，避免缺正常summary掩盖真实exit。当前未实施任何源修复。

## 最终scope、证据及限制

[final-closeout.json](final-closeout.json) 补核三个**精确application**（绑定真实Go PID43804/30192/3852、server PID及backend_start），在child实际wait后外部查询全部backend=0；补充psql进程也都实际wait0，最后无可见Go/test。内部closeProbe排除自身的0没有被用作全部owned0。原08:50:32Z `final.json` 的应用集合只有前两组，不能把它反推为当时第三组backend0；以08:55:33Z补充记录为完整三组最终scope。

最终DB **1992988351 bytes**；原global1406932671，累计delta **586055680**，small余 **487686144**。82个历史perf schema完整集合前后相同，active_other=0，PGDATA卷可用 **47613984768 bytes**（实际探针时点值）；没有重置基线或给新额度。最终身份127.0.0.1:5433 / PG180006 / han_perf_s24 / default_transaction_read_only=on。

本轮全部owned客户端均已wait。后续只读主机进程观察曾另见psql PID34364/parent31972，不在本轮owned PID集合，归属unknown、未操作或终止，不声称整台主机所有PG客户端均已停。closeout第一版缺import re，未发起PG查询便失败，原脚本保留；closeout-r02补成最终完整三组scope，没有重跑测试。

原始入口：两正例各`result.json/stdout.log/stderr.log`；第三组sample/resource-close/probe-pre-close/failure/inventory与stdout/stderr；`queries/`包含每次只读query/argv/PID/real_wait_exit及app观测、每正例结束probe、最终三组probe；`start.json`和两份outer采用记录保留原时钟；第四组明确 [NOT RUN](earlier-freshness-failure-not-run.json)。没有源补丁、没有新的真实PG增长/leak/九语义或完整性能结论。下一步只交Main与原Reviewer核本物证；运行Gate阻断，原性能NRU和任务active保持。
