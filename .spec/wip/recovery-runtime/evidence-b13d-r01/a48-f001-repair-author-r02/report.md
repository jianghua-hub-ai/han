# Article S4.8 F001 局部返工报告 — STOP

原作者 Noether / add-backend-dev，Main Host 实际派发 `01a1861b-9405-7942-8dd2-2db8b8d39eb2`。固定 Han/main/HEAD6da8abb；分支/HEAD沿Main固定值，本轮不运行Git。仅原两源改变，其余原八源及枚举的生产/测试/模块保护字节均不变；未改任务、设计、规则、旧Issue证据或其他角色文件。

本轮处理独立 Dirac 的 MEDIUM **ARTICLE-B13D-S48-CR-R01-F001**。选中 evidence 在提交前被赋入命名返回值，后续SQL回滚却保留了非零结果，回归原消费者 zero-on-error 合同。修复在函数入口注册私有结果归零 defer；后注册的原事务最终收口先执行，归零 defer观察最终 error（含原rollback join），仅失败时清空 evidence。不吞错、不recover panic、不改 healthy/winner、unknown分类、错误cause、SQL或独立exact三者回读算法。原PG helper恢复整个零结构体断言，原23514、九表物理回滚、owner、元数据、首份分项与锁释放断言均保留。

先在**未改生产字节**上使用原Dirac fragment取得真实Red，普通/race每模式 Healthy1PASS、CurrentUpdate和PublicationInsert2FAIL。两条失败先证明 evidence INSERT已发生、真实SQL23514、九表rows/xmin/ctid精确回滚、NOWAIT解锁；失败仅为返回值非零。再应用已确认最小两源patch，同一fragment普通/race各3PASS、0fail/skip；回滚返回ref/digest均空、原error仍可errors.As识别23514，健康发布仍返回持久化evidence。

原required普通/race分别 **54U/C PASS、58PG PASS，零fail/skip**。五组U/C命中11/8/1/1/2，P命中13/1/8/2/28；原31U/C和6PG叶逐名保留。真实COMMIT成功后ACKerror与独立exact三者回读、late winner/历史复用/取消join等原断言仍通过。两源fmt为空；普通及integration vet通过。按Main本轮收窄，不重复Linux跨平台、额外consumercompile、旧全局或Manifest7；历史Linux结果只保留旧时点，不覆盖本次新SHA。

原probe全文SHA始终为 `E8DF4FB70E7C00D2F6D7946B8791208B23392845AEACA6FFB9AEB29898A28084`，只test-only overlay附加，不做生产overlay。原旧122-schema receipt top在freshD **NOT RUN**。本轮新128schema均按具体名单实际SQL读取table/rowcount，128/128存在，Rows关闭、acquired=0；schema/数据全保留，不DROP/清理/停Main cluster。固定迁移0001/2/3/5/9/11不变。

所有执行使用D独有TEMP/TMP/GOTMPDIR/GOCACHE及fixture目录，offline/local/modreadonly/GOMAXPROCS2/-p1串行；普通DSN空，PG只注入Main当前article_dsn。r02实际physical preflight证明18.6、127.0.0.1:51934、han_b13d_article、han_b13d、data_volume=D:。初始一次Go -overlay位置错误exit2、零产品叶，作为runner setup错误原样保留，绝不计Red；原C盘磁盘满材料不删不动。

所有15条命令父Popen.wait均真实返回。每条命令的Windows process job记录全树创建/退出通知，所有取得的进程handle实际WaitForSingleObject，创建/退出FILETIME及逐PID记录均保留；job进程总数与记录数精确相同、全部ACTIVE_PROCESS_ZERO。stdout/stderr读线程均到EOF后join，fixture原action/worker/Journal/Rows/Tx/pool实际收口断言通过。详actual-join.json和各processes.json；不以exit0/currentPIDgone替代，也不声称其他角色或Main PG进程已join。

当前无第三源/公开合同/资源阻断。原两源冻结交Main及非作者复审；**finding仍待独立复审，作者不关闭finding/Task/G4**。生产Root/native正控/真实旧批准来源或ACL/完整S4.13/E2E、本轮Linux及全局7均未据此签PASS。最终包不含cache/tmp/binary；原执行目录保留，不清理。
