# R06：PROBE-R05-F001 作者交接

**SOURCE / LOCAL STOP；PG 未打开。** 固定 Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。仅改 H；最终 **550B16614224A13F31CEE931C016C3B6841F59E80F27CE2A61F24720219392E7**。全部已发起命令实际 wait 完成，最终进程查询无可见 Go/test；未读 User DSN、未执行任何 PG/SQL/DDL/DML/Tnew。

完整五哈希/快照/STOP 在 [source-freeze-r06/manifest.json](source-freeze-r06/manifest.json)，R05 E973/209145B 前像在 `before/`，精确小差分 [harness-r05-to-r06.patch](harness-r05-to-r06.patch)。另3Go/docs、R05 manifest/五snapshot、原global/window/policy/small摘要均核不变。Role/实施方法/相关规则沿有效全文；完整消费原Reviewer R05 report的F001、reader→collect→stopLocked→close消费者后实施。以下为作者回归，F001独立关闭仍待原Reviewer。

修复仅针对错误传播：Acquire/SQL两个错误分支真正调用同一窄 `pgPerfResourceReadError`。先按 `pgPerfContextOnly` 判原错；非context（含mixed）保留为安全类型、原始动态类型和经5位大写字母/数字验证的SQLSTATE，不保留/输出Message、Detail、Hint、包装原文。纯context关闭归一为context sentinel，不误报。classifier优先识别新安全类型，防mixed中的取消覆盖真实分类；collect在已有正常关闭终态时仍升级 `failedOwn`，返回终态与安全原错的join，diagnosis另存 `read_error_class`。成功但取消/late仍不发布观察。

真实行为 RED：`red-reader-r01` exit1/3.141s，旧适配器逻辑先抽取到两个真实调用点共用的helper；已cancel ctx与真实模拟 `*pgconn.PgError`/wrapped/plain/mixed 原错同时出现，8个Acquire/SQL子项及父测试失败，显示 `resource_failure=false/context_canceled`。不是编译失败，也不冒真实PG重现。

| 最终同五hash命令 | 实际结果 |
|---|---|
| `green-r06-final` | exit0 / 2.937s，16 pass事件、0 fail/skip，含子测试；8种真实helper→collector Closing组合、纯取消、nil/success取消、SQLSTATE脱敏、原Closing与请求取消。 |
| `race-r06-final` | exit0 / 17.672s，race×3共87 pass、0 fail/skip；增加原RootClose、phase/unknown/outcome必要smoke。owned非合作child的wait86及持久化PID/inventory仍按原测试核，不将故障exit算资源成功。 |
| `build-final / vet-final / format-final` | exit0；7.219s / 1.468s / 0.078s，gofmt diff空。新编译binary摘要在 [binary.json](binary.json)，未冒用R05 binary。 |

实际argv、PID、wait exit、起止/耗时、前后五hash及raw摘要见 [command-index-r06.json](command-index-r06.json) 和各命令目录。`current_five_hashes=true`才是最终版。所有旧失败和中间绿色原样保留；检查raw未输出测试敏感字符串。诊断与状态变更不改变15/5/20、owner、执行窗、配额或ledger；stopLocked/closeResources/budget parser/context-only函数正文核与R05一致。

**写集事件：**中间 `race-final` 复用原关闭测试时，旧fixture root导致在R05 `fixtures/` 新增6个MkdirTemp目录，未覆盖既有文件。这是本轮写集越界，已向Lead如实报告；未清理/移动。精确目录、其中全部文件长度/SHA见 [fixture-scope-incident.json](fixture-scope-incident.json)。已将fixture root改为R06，最终普通与race在R06/其隔离sandbox内；旧R05 freeze和原资源摘要仍匹配。不能声称旧目录完全无新增。

`TestPgPerfR05_ReadOnlySQL` 完整函数经LF归一后与R05相同，摘要已记manifest；入口选择/只读目标/usage可沿用R05 report，**但其reader依赖已变**，后续必须使用本R06 binary及当前五hash，不能拿R05旧binary/七Manifestcheck当R06结果。Main已在R05做的七检查只属历史E973。真实只读SQL/超时/连接backend回收及Growth/leak/九语义/完整性能仍NOT RUN，PG窗口保持关闭。交原Reviewer只核F001和必要delta，再由Main决定新检查及只读授权；不自报核销，不关闭S2.4。
