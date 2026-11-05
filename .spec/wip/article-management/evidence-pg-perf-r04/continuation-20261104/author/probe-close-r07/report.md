# R07 关闭门最小修复交接

**SOURCE / LOCAL STOP；本轮PG未打开。** 固定Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，唯一源H最终 **9F08F70B2E2063D5F178064F5EA0D9F84DABB1BA625CEF90402DE70D4CB7CAC2**。五文件/快照、真实STOP及前像摘要见 [source-freeze-r07/manifest.json](source-freeze-r07/manifest.json)，原R06前像在before/，精确 [r06→r07 patch](harness-r06-to-r07.patch) 已落盘。另3Go/docs与旧global/window/policy/small摘要核不变；没有覆写R06实物或旧runner。最终无可见Go/test进程，全部发起命令与owned child真实wait。

已读有效Backend Role、实施/调试/Go规范，复核固定依赖：pgx v5.11.0 `pgxpool/conn.go:18` Release遇closed/busy/nonidle调用Destroy；puddle v2.2.2 `pool.go:72` Destroy异步执行，`pool.go:687`先destructor再移除资源/归还计数；pgxpool `pool.go:314` Close destructor等待CleanupDone。**机制已证，R06那次实际动态分支仍unknown**，不把退出后backend0当close成功。

行为RED `red-async-r01` count3真实exit1/4.218s：使用现有真实puddle Pool→Acquire→Destroy，Destructor由屏障暂停；首次真实Acquired=1，旧即时检查让三个独立owned child均exit86。不是模拟计数或PG重现。相同路径先转绿，再覆盖idle资源Close析构、持续持有、Close阻塞和提前hard。

最小实现：统一沿原closeResources确定的 **closureEnd=min(开始+5s，既有更早hard)**；只在同一关闭门必要消费者间传播。probe/diskPool先有界核归还/异步销毁，再进入有界Close；business原可关闭分支也对Close完成设同一上限。tx回收、原连接settle、probe只读核证和最终goroutine检查受剩余原deadline约束，不新开阶段额度。Pool.Close在明确owned goroutine执行，成功由完成channel/owner计数确认；超界保留记录并当前owned child exit86，不返回遗弃goroutine。记录before/after、started/finished/observed/deadline、elapsed、Close完成状态。15/5/20、clock、caps、业务fixture/SQL、依赖与策略均未改。

| 最终同五hash命令 | 实际出口 / wall / 范围 |
|---|---|
| green-close-r01 | 0 / 9.609s；20 pass事件、0fail/skip，含子测试；R07关闭及R06真实错误保留。 |
| race-close-r01 | 0 / 35.094s；race×3，共60 pass事件、0fail/skip；暂态成功，持续Acquired按原5s fail-stop，提前hard约150ms与Close阻塞按更早界限fail-stop，正常取消可收尾、原真实错误不被抹除。 |
| core-r07 | 0 / 11.281s；同版binary在R07新sandbox运行原core race×3，57 pass、0fail/skip。 |
| build-r07 / vet-r07 / format-r07 | 均0；6.578s / 1.812s / 0.125s；integration/race binary、定向vet、gofmt diff空。 |

完整argv/PID/wait/起止/前后五hash及rawSHA见 [command-index-r07.json](command-index-r07.json)。负例的parent通过仅表示已核真实child wait86及其独立PID/failure/inventory，不能当资源成功。新binary哈希及构建来源见 [binary.json](binary.json)，不使用R06旧binary冒同源。fixture全部写R07或其新sandbox，旧六目录事件不操作。未重跑既有有效检查，未Git/清理/委派/TasksPlan。

当前仅离线机制与回归，不证明真实PG timeout后的Close一定能在5s内完成；如果真实driver清理仍超界必须继续FAIL，不能借库内部15s等待上限放大本契约。R06 third exit86、其后FileNotFoundError和earlier NOT RUN均原样保留。只读入口函数主体与R06相同，但关闭依赖已变，未来须用R07新binary/当前五hash；不继承R06两正例为R07全路径证明，不开启PG或新window。

交原Reviewer仅评本关闭delta及必要消费者；作者不签独立PASS、不核销原12项、不关闭S2.4。用户新增的readonly候选runner授权尚未实施；按Lead最新优先级先交本freeze供独评，runner可另续接，本报告不声称已完成候选或取得运行许可。
