# S4.14 实施与局部验证交付 — STOP

Owner：Chandrasekhar / add-backend-dev / 01a187a1-c022-71f3-92f5-6a9dc032e32b。固定 Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca。依据完整 canonical s414-source-local-process-pg-start-r01.json，复用 Main 已接收的 s414-g0-r01。仅原五新增源；未提交/暂存/push/切分支、未改SSOT/设计/TP/规则/modules/共享root/B13C19。

## 实际交付

- recovery_restore.go：现有公开 RestoreDriver 实现；批准来源/工具/目标 ports 必填，缺设施拒绝；Prepare 保留同源 handle，核内容长度/hash与路径/handle身份；Replace 只消费真实 Runtime.CheckPhase/ReadPermitBinding 的当前 restore 许可且至多一次；Resume仅只读verify。固定 restore-v1/verify-v1、私有plan+target_connection stdin、显式最小ENV；严格12字段receipt、65536/65537边界、UTF8/重复/额外/null/时间/摘要/逐字段绑定；依赖错误呈现去敏而保留 errors.Is；child Wait、stdin/stdout/stderr、取消callback实际join才返回；目标独立读回不以exit0代证据。
- recovery_exclusive.go：既有 DeploymentExclusive/ExclusiveLease 的中性设施适配，实际Check、单owner幂等Close、返回owned-handle+error不丢所有权；无生产锁后端/fallback。
- 三测试文件：真实公开 NewRuntime/Enter/Resume producer、独立Begin committed回读、零许可/unknown零Replace、跨新Runtime读回、严格codec、受控当前testbinary唯一TestMain固定argv、真实child/pipe/target事务join；integration只消费授权RestorePG与0001–0010显式白名单、真实identity Store、新目标连接和独立软件来源witness。全部函数≤50行，五文件gofmt检查为空。

## 当前版本验证

| 对象 | 普通 | race | 原始结果 |
|---|---:|---:|---|
| 原Driver/DeploymentExclusive两top | 80叶 / 88通过事件 | 80叶 / 88通过事件 | green-delivery-normal/race.jsonl |
| 专属RestorePG top | 7叶 / 9通过事件 | 7叶 / 9通过事件 | pg-delivery-normal/race.jsonl |
| 限定旧root/lifecycle/config回归 | 319叶 / 348通过事件 | 319叶 / 348通过事件 | regression-normal/race.jsonl |

全部上述运行0 FAIL/0 Skip；风险组与逐叶全名见 test-inventory.json/leaf-evidence.json，未拿top PASS替代叶命中。两普通top与PGtop的-list均实际非零命中；normal/integration go vet 均退出0；source5 gofmt -l为空；Linux amd64/CGO0仅compile-only退出0，制品在独立scratch且未执行、不入包。没有声称Manifest全七项通过。

行为Red：red-behavior-v2.jsonl，骨架可编译但非法receipt/缺设施断言失败，两个top/两个组共4个FAIL事件；red-sources/保留当时四文件。首次Green的Windows held-file rename被OS拒绝，保留失败raw，修正oracle后补同字节不同handle/path真实身份负例；后续测试拆分的两次编译失败也原样保留，未冒充行为Red。初次工具transport timeout的空red-behavior输出无可用PID/测试结果，已立即只读确认无匹配运行Go，随后用记录PID的v2重新取得Red；见 raw-index.json。

## 进程、PG、去敏与所有权

process-ledger.json记录20个Go parent真实PID/close结果，全部waited，无剩余本Role Go命令；格式化与只读Git进程同步结束。leaf-evidence.json保留22行可观测child PID/Wait/pipe-join记录；全部实际工具调用统一经过同一Wait+三管道join路径，取消AfterFunc亦被join。PG目标事务阻塞时已观测Runtime.Stop撤权且deployment Close0，真实rollback/连接归还后Close1；child-hold同样覆盖真实pipe仍owned时不能返回。

RestorePG只运行指定新top，固定批准0001–0010，无全局integration/旧DSN/autoDROP。累计21个随机quoted schema全保留（含首轮），见 retained-schemas.json；实际连接/pool随每例结束关闭，Main拥有的PG server保持不动。真实DSN只在已授权PG测试的内存与私有child stdin使用，不入argv/ENV/日志/本包。raw-index.json扫描Go raw中完整DSN及synthetic canary均0命中，未持久化原child stdin/stderr。source5快照里只有合成canary定义，不是private raw。

before.json/after.json记录五路径null→当前hash与冻结对象前后；B13C19/shared root/modules/Tasks/Plan保护摘要全部相同。status-final.txt记录共享大脏树；本任务恰五个??，无Git写。untracked源不出现在普通git diff统计中，因此 source5.patch显式提供五个从/dev/null新增的完整patch，source5-after/提供同hash可复核快照，另做源尾空白检查。

Main已明确预告identity r54→r55与Article r26→r27的两处design-only合法窗口；它们不改Restore/Runtime/root/native合同，不当未知writer，不重开G0；本Role未写/签收这两份设计新修订。

## 必须保留的限制

- native OS锁/no-follow/ACL、批准真实binary、安全跨进程同handle传递、原生耐久receipt与进程树隔离仍未固定、真实启用Gate CLOSED。本组件没有production注册。当前同handle字节消费由测试专用软件handoff callback在parent完成，child验证实际固定协议/私有pipe/退出；不声称child已通过native句柄恢复真实artifact。独立PG来源witness也仅软件fixture，非真实备份还原/原生回执认证。
- deployment两两竞争为显式软件facility fixture；没有实际三production入口/多进程OS锁通过声明。normal/recovery/requeue唯一root装配归S4.16。
- 私有driver.Close可等当前call；Prepare后idle保留handle的最终Close/join仍须S4.16长期owner负责。未新增Runtime API/债/假hold，也不把长期owner义务伪称本卡已root验证。
- root回归是上述限定集合；原PasswordResetMaintenanceRoot及Main connection helper会启动非本次固定argv helper，未纳入该受控process窗口，不声称全部root测试通过。当前组件在bind端口sentinel结束，不执行身份Finalizer/实际restore/开流。
- 待非作者独立Code Review/复审与Main同版核证；本返回不关闭Task/G4，不代独立Review、全七项、native、root、真实恢复或E2E。

全部授权局部实现/自测已交回，当前STOP。未发起新的routine批准；不再写源/运行Go/PG/child，等待Main按原范围进行独立Review与接收。
