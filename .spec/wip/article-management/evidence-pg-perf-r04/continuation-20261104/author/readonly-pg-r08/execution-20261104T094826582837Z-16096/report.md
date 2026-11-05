# R08 只读执行停止交接

SOURCE/LOCAL/PG STOP。仅执行本次 Lead grant 的第三组；第四组未启动。原 runner 自然 exit1 后未重跑、未改 H/runner/hash/window、未追加 PG 查询或 Go 执行；随后只有既有物证离线读取与本补充文件写入。DSN 仅从 User 同名项注入本次进程，未打印/落值。源码与旧资源未写。

实际许可09:48:08.289719Z，runner开始09:48:26.582837Z，最终收口09:48:51.025216Z；原授权起算42.736495s，不滚动重开。固定目标127.0.0.1:5433 / PG180006 / han_perf_s24 / readonly=on。

第三组 `total-fifteen-failure`：Go PID8336，自然 exit1，forced=false，真实wait于09:48:47.591491Z完成。16s受控SQL注入确实到达，外部观察PgSleep；Started09:48:32.3812414Z，Deadline09:48:47.3812414Z，Finished09:48:47.3821867Z，实际15.0009453s。accepted=false、bytes=0、context_deadline，未把失败采集当资源成功。

关闭原始资源证据：probe acquired_before=1，经60.5418ms到0；Pool.Close实际完成，整体closure163.9258ms，小于原共享5s；worker已join、root/resource_failure=true、goroutines4→3、内部residual_backends=0，无fail-stop86。这与R06即时Acquired=1导致86的旧结果严格分版。

**原 runner 拒绝原因是归集路径，不是已证实PG关闭失败**：`material.json` 将五份resources JSON同时列missing/invalid，普通绝对Path.read_bytes对278–297字符路径返回OSError errno2；同一实物加Windows绝对扩展前缀后全部可读且JSON合法。`offline-material-supplement.json` 保存每个原路径长度、原错误、实物SHA和内容；`offline-material-diagnosis.py` 可离线复核，仅读原物证。它抽取同一冻结runner的纯checks_for/instant/require函数，不运行Runner/main、不读取凭据；17项原predicate均true。这里只交补证，原 `material.json/result.json/final.json` 与其author_checks_satisfied=false、整体exit1、completed_modes=[]保持原样，待Lead/Reviewer独立接收，不作者翻写成PASS。

第四组 `earlier-freshness-failure` **NOT RUN**，因首组归集不满足即停后组，未利用剩余许可重试。R06两正例仅历史；不能称R08四组完成，完整性能仍NOT RUN，任务及旧pending不核销。

收口 `final.json`：10个owned客户端（1 Go、9 psql）均实际wait，无unresolved；第三组精确application/serverPID/backend_start对应外部backend_after=[]；最终owned集合=[]。最后外部观察者自身被查询排除，其客户端PID4964另有actual-exit.json真实wait exit0，不宣称其自身已被另一外部观察者核0。当前本机进程检查无go/test/runner16096，仅既有非本轮psql34364/父31972，归属unknown、未操作。

最终完整82schema集合与许可/before一致，active_other=0；DB1992988351、global1406932671、累计586055680、small余487686144，PGDATA可用47312252928bytes；五source hash、binary C9AFE…866B71和四旧resource摘要均原样。无DDL/DML/newSchema/Tnew/性能/清理/Git写。

索引：`start.json`、`preflight.json`、`total-fifteen-failure/{launch,pid,actual-exit,material,result}.json`及stdout/stderr保留真实执行；`queries/1…9`保存每次SQL和真实wait；`final.json`保存收口；`offline-material-supplement.json`为独立新增长路径补正。最小后续建议仅针对归集器离线处理Windows长路径，保留本次实物并复用补证；第四组需Lead另行明确执行安排。本作者不擅改旧runner或开启新组。
