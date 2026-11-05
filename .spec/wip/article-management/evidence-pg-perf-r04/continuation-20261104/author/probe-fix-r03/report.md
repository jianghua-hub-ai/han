# 资源guard候选可行性：open-handle增长反例

**候选停止采用，H及另外四份冻结源均未修改。SOURCE/LOCAL/PG于14:52:33真实STOP并释放。** 本阶段未创建任何PG连接，无DDL/DML、无新性能窗口、无时钟或baseline协议修改。

G0必要Role/Skill/rules/profile已读，见g0.md。按Lead要求先验证增长准确性，再考虑idle SQL对照和H实施。native helper直接调用os.Open(directory).ReadDir(128)，每一轮重新打开目录、每项Info取枚举缓存；没有复用旧DirEntry。权威比较值为写句柄File.Stat实时EOF，跨进程时另核写进程上报EOF与父句柄EOF相同。

| 阶段 | 实时句柄EOF bytes | 全新枚举合计bytes | 一致 |
|---|---:|---:|---|
| 同进程持有写句柄，第1次写 | 8192 | 0 | 否 |
| 同进程持有写句柄，第2次写 | 16384 | 0 | 否 |
| 同进程持有写句柄，第3次写 | 24576 | 0 | 否 |
| Sync后，写句柄仍开 | 24576 | 24576 | 是 |
| 第二进程持有追加句柄，第1次增长 | 28672 | 24576 | 否 |
| 第二进程持有追加句柄，第2次增长 | 32768 | 24576 | 否 |
| 第二进程持有追加句柄，第3次增长 | 36864 | 24576 | 否 |
| 写句柄均关闭后的读对照 | 36864 | 36864 | 是 |

Go build实际exit0/1.453s；隔离实验实际exit1/0.140s，是候选准确性反例，不是编译失败或原H回归失败。子写进程通过stdin按序受控增长、每次Write返回并核两句柄EOF后才采新枚举；随后stdin关闭、实际Wait正常结束，没有依靠sleep推断时序。所有fixture在本目录open-growth-1719438315/，保留不清理。

**判断：本机原生目录枚举缓存可以漏计持续打开文件已经完成的增长。** 之前idle PG字节等值、约47ms只能证明那一时点，不足以授权把该缓存作为准确计费。在当前独占/只读PG范围内没有继续试验真实PG增长；本反例发生在获准D盘本地fixture，不声称已证明PGDATA C盘每种写入模式同样滞后，但足以否定未证明的新采集器准确性前提。不能把强制Sync/关闭PG文件当修复来刷新metadata，这会扩大范围并改变负载行为。

因此未做：native idle SQL三次等值、新guard三次<5s、H补丁/新freeze、真实PG增长、其他guard故障回归。它们均NOT RUN而非PASS。原SQL/5s失败闭合、全DB/tier基线和资源门保持原样，原超时问题尚未修复。

最小安全返回建议：暂保留现有准确SQL与fail-closed。当前没有已证明既准确又<5s的纯目录缓存替代。若继续优化，需另行评估能取得实时EOF的native文件查询方式并以此增长反例为先决门；逐文件实时查询可能仍有原成本，未测不承诺提速。若改资源探针预算/新鲜度及写入准入协议，属于另一决策，不能本次私自延长5s、silent fallback、近似计费或掩盖滞后。交Main采用下一范围，不在本阶段继续试更多方案。

原始实物：open-growth.go、build-open-growth/{result.json,stdout.log,stderr.log}、open-handle-growth/{result.json,stdout.log,stderr.log}。实验stdout SHA256 CF87FADC0A0F2966AACF6D501A3CAD6613C47204E68D0D31599A6FFB497ED57B；两条命令均保存五源before/after完整hash。源码仍为B8D11…/38ADA…/F25D…/3E531…/D476…。

14:52:33.3630664+08只读Win32核：open-growth.exe无存活，原build Go PID40348、实验父PID33308均已结束；helper成功走完child.Wait，runner actual_wait_exit=1。没有PG会话需要收口，不冒称本阶段另做了PG backend probe。旧源码/snapshot/日志/地图均不改，无Git写/清理/委派。S2.4保持active，独立Review及性能验收不代签。
