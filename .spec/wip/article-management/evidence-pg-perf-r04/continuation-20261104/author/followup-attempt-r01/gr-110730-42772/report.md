# R09 growth 前置拒绝：SOURCE/LOCAL/PG STOP

同一attempt/Tnew=10:22:37.932710Z及full门11:15:37.932710Z保持。按唯一Lead新grant启动现成CLI，未编译/改源/改grant/policy/旧账，也未重跑或进入下一stage。DSN仅授权进程私密注入，未打印或落值。

runner于11:07:30开始，11:07:40.973061Z收口，真实wait exit1。**growth测试binary未spawn，无Go PID/测试stdout/typedcap/共享flight/collector SQL样本。** 身份和集合通过；query3准确SQL实读DB2029926079，grant期望2029893311，差32768B，严格一致性预检拒绝，尚未进入新schema/DDL。之后静止核账仍2029926079。差额来源unknown，不猜外部写者/后台组件，不扣噪声、不clamp、不自行修改expected值。

完整83schema集合与grant一致，active_other=0，实际新schema=[]；final.problem保留、closeout_error=null/failed=true。8个owned psql均实际wait exit0、unresolved=[]；owned_backends=[]及最终external_all_owned=[]。最终外部观察者自身排除，queries/8-all-owned-after保存其application/serverPID/backend_start及客户端实际wait；不称观察者自身已有另一外部probe0，也不称全Host无连接。

最终global1406932671、累计622993408、small余450748416、PGDATA可用45653839872B；五source、nonrace71B3…5521D5、旧四resource摘要和实际policy7C94…33E04均未变。没有任何新schema或数据，本次不改变原growth失败历史；R09 SourceGate PASS仍仅源码，R09 PG增长/共享/typedcap/关闭尚NOT RUN，不能把此wrapper exit1当期望负例成功。后续须Lead实际接收该漂移事实并另作明确决定，不自动再开。

索引：start.json含完整grant/SHA与版本；queries/1-identity、2-catalog-before、3-size-before为实际前置；queries/4…8为资源/集合/准确size/身份/最终owned收口；每query有launch/pid/actual-exit/result。final.json原样，execution-index.json存完整raw摘要及前置差额，无复写历史。
