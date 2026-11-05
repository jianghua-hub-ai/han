# R09 作者冻结交接

SOURCE/LOCAL STOP；R09未连接PG、未读DSN、未改Tnew/实际policy/旧resource或其他四source。正式freeze于2026-11-04T10:49:08.308613Z，main/HEAD6da8a保持。最终进程证据final-process-probe.json：无go/test，仅既有psql34364/父31972（unknown ownership，不操作，不称全Host无连接）。全部实际命令真实wait。后续只交证据，不再写源或运行Go/PG，供原Reviewer窄delta及Main现场核对。

## 变更与不变契约

只H：保留原每请求15s/旧freshness/owner/parent的入队deadline；flight仍一个串行槽。等待者仅在其入队后有一份新的完整成功发布、无ctx/owner/terminal错误且仍在自己原deadline内，复用该成功；不延长已有ctx、不重发SQL、不更新s.last或freshness。采集结束后才进入的新请求仍真实读取。A原START/ReadStarted/Finished保留，B单列Requested/Returned/WaitNS，B SQLNS=0、source_sql_ns指A成本、reused=true，flight_id相同；不能把回执数当独立SQL数。失败/未开始SQL不能据flight_id0推成采样成功，须连同accepted/read_started核对。

consumer真实接线：checkDiskBudgetContext→collectDiskBudgetContext→resource.collect；实际reader仍readDiskObservation、准确SQL原文不变。新增局部consumer helper仅复用已有注入reader接口以离线验证同一取证路径。诊断文件名增加局部原子回执序号并exclusive写，避免同机粗粒度time.Now导致两个快回执同名覆盖；不扩成资源框架。

cap失败改为局部typed error，保留真实observed/tier/global/cap原值及原Error文本，正常threshold/volume判断不变。FaultSeedDisk只改诊断驱动：loop最多60s，仍从已受seed120s/root/全局更早期限的h.ctx派生；每个追加批核resource准入。seed前安装defer，记录阶段/seed/loop边界、每个已完成批的rows/起止、总rows、初始值/目标值/实际cap观察、last_success及终止分类；即使seed内Fatal也保留当时状态。目标须root取消+failedOwn+真实typed cap值达到阈值，不以t.Failed/exit1或任意取消通过。真实cap仍有意exit1，deadline/freshness/其他原因明确NOT REACHED；不强制checkpoint/sync/关闭连接催cap，不给额外60s收尾。

15/5/20、root observer归属、121/60/tail1、所有caps、seed量和正式10/30×3均不变；R08关闭/全局pollUntil、budget/policy parser、reader安全错误规范化和只读入口函数原文已在freeze脚本与R08前像逐一比较相等。R08保守失败不是合同违规；R09是本轮明确采用的有限活性优化，未宣称真实PG增长已修好。

## 实际失败与GREEN

red-flight-r01真实exit1（非编译失败）：A成功/B第二次reader→旧deadline取消，reads=2；旧故障谓词把deadline/freshness误收成cap。实现过程中green-flight-r01发现快回执同名丢一份，green-flight-r02保留原子回执后发现测试把Windows相同墙钟字符串当不同请求；最终consumer用显式受控单调测试时刻证明不同Requested/原START，而运行证据靠flight/回执ID区分，未放松期限。

green-flight-r03选择式过宽误包含ReadOnlySQL，四子项均在“explicit absolute evidence directory required”处零时长FAIL，尚未打开pool/读取DSN；它不是行为RED/PG执行。该命令真实exit1保留，之后改成明确离线列表。freeze初次函数摘取跨入新type声明产生metadata断言exit1，原partial snapshot/patch保留并逐字核相同，未覆盖；修正至函数结束边界后完成正式manifest，见freeze-r01-stderr.log。

| 最终同五hash命令（各result.json含完整argv/UTC/PID/wait/前后source/rawSHA） | 结果 |
| --- | --- |
| green-offline-r04 | 普通离线回归exit0 / 13.125s |
| race-offline-r05 | race×3 exit0 / 50.360s |
| build-nonrace-r09 / build-race-r09 | 均exit0 / 2.015s、4.625s |
| core-r09 | 隔离CWD新race binary、core×3 exit0 / 10.782s |
| vet-r09 / format-r09 | exit0 / 1.469s、0.078s；格式diff空 |

覆盖：共享单采集、新请求重采、旧deadline不延长、expired/canceled waiter与late拒绝、真实SQL/noncontext+mixed原错保留、consumer两回执关联同一flight、typed cap与deadline/freshness区分及实际validator→terminal→fault谓词、R05预算/owner/Closing、R06安全错误、R07资源关闭、R08最终期限。离线测试不代替真实SQL或growth。

## 固定输入

完整五hash/snapshot：source-freeze-r09/manifest.json；H **C3DFDC66FDBFB025760ED78B0E0061CC49BD7AFB2DA0BF76685D94D0D89B8B99 / 241407B**。前像before/，最小差分r08-to-r09.patch，其他三Go/docs原hash不变。主binary.json及manifest.binary为同一个nonrace **tools/r09.test.exe / 20050944B / 71B3EF47D466A8EAB899C266311EAD224E5F5316D6B244A8E2D7EFDF705521D5**；同exe含原只读入口，不需另造PG binary。race binary **39840B6D3E9FC1B1C0D6C270AD54C5016B877A1A9C02CFB2F412C47733F44CCE / 32902075B**仅本地race回归，不能用于正式性能。

旧四resource SHA及真实small80B不变，medium本阶段未登记；实际followup execution-policy仍7C94C0…33E04，Tnew=10:22:37.932710Z/full门11:15:37.932710Z，编写/失败/等待/Review全部计入，不复活旧batch或重置余额。尚未运行R09 growth/PG，leak/9semantics/全量后点仍NOT RUN；原R08失败和metadata恢复全部保留。交原Reviewer独立评审与Lead采用后，才可另授同窗必要验证；不作者核销finding或关闭任务。
