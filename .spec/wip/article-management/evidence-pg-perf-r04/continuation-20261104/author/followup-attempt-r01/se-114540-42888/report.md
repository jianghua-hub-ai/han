# N9真实失败：SOURCE/LOCAL/PG STOP

Go PID40712：11:45:43.221962Z→11:53:40.711913Z自然exit1、forced=false；wrapper真实exit1，final11:53:46.069629Z，problem保留、closeout_error=null、failed=true。没有重跑、源改、编译、缩fixture/计数、新SQL或其他stage；本次后仅只读现raw/source和新增此交接。所有费用计原Tnew10:22:37.932710Z，原145/150不动，全量永久NRU。

前8项实际PASS：fresh-t1、moderation-race、delete-race、cross-page、late-notice-results、three-lock-chain、production-read-plans、initial-drain-1000。1k原件accepted=sent=1000/failed_terminal0。第9项initial-drain-10000实际FAIL于seed transfer notice recipients: context.deadlineExceededError；尚未进入pgPerfVerifyInitialDrain/10000 oracle，没有10k drain回执。不得将配置7500 notices/15000 recipients或前8项PASS换成整组PASS。

直接截止：H107 small root300s/seed120s；H2082每harness创建root，H2232派生seedCtx并用于首DDL/seed，H3331 copySeedRows每1000行前调用同步准确guard，继承传入seedCtx；H3116转抛当前recipient seed错误。最后失败采集Requested/ReadStarted=11:53:37.0850372Z，Deadline=11:53:40.4154537Z，Finished=11:53:40.4180558Z。截止与该seed入口首同步请求11:51:40.4154537Z相差120s；当时请求+15s=11:53:52.0850372Z，last成功START11:53:30.6027262Z+20s=11:53:50.6027262Z均更晚，故先触seed父截止，不是small root300s或drain轮数耗尽。root在资源终态后取消；此子项总128.84s，global/hard尚未到。没有改任何预算。

进度：已到transfer_notice_recipients seed调用，失败seed没有完成manifest或逐COPY行数日志，因此其实际已复制/提交行数unknown，不能写0或15000；该函数在全部COPY之后才Commit，现失败路径未到成功Commit。原件仅能证明既有阶段已返回、当前recipient阶段未完成，不能精确补造批数。最后成功准确DB2129606335、无cap-hit；collector原真实SQL约数秒/每有限批guard会计入seed时间，但不凭当前资料强归底层OS/IO。进一步修复需Main具体授权，不自行加seed预算/减guard或重跑。

9个schema全部保留，实际新增与inventory一致，旧85完整、最终94。前8资源failed=false，末项failed=true；所有closure均<5s，末项124.5542ms、goroutine3→3/residual0、workers_joined=true；末项resource_failure=true/terminal=context deadline，其他正常。外部owned_backends=[]/external_all_owned=[]、unresolved=[]，100个owned客户端均实际wait。最后观察者排自身并另有真实wait，不声称全Host或观察者自身另有外部0。

Live before2105767615（历史snapshot差24576，unknown如实记录），after2129835711，global1406932671/累计722903040/small余350838784；PGDATA可用45625118720B。五source/R09非race71B3、旧四resource和原policy7C94不变。前8有效结果与最后seed失败分别交Reviewer，N9尚不能PASS；不核销原finding。

execution-index.json列9个Go子事件、首错/父截止/最后采集、资源及核心raw SHA。material.json逐对象绑定原件SHA，e/下保留oracles/回执/锁图/计划/seed/资源及失败半成品；queries逐条保存所有范围probe及wait。final原样不覆盖。现在STOP，等待Main决定；不自动进入任何后续验证。
