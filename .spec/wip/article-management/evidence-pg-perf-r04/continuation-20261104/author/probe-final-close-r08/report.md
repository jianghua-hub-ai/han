# R08 CLOSE-R07-F001 作者交接

2026-11-04 09:37:54Z 已冻结并释放 SOURCE/LOCAL；PG 本阶段未授权、未连接。当前没有自有活动 Go/test 子进程，所有执行命令均实际 wait。最终只读进程清单见 `final-process-probe.json`：没有 go.exe/*.test.exe，仅既有 psql PID34364/PPID31972（归属 unknown，未操作，不能据此声称全机 PG 客户端为零）。候选 readonly runner 准备已取消，本作者未新增候选文件，既有文件不清理。

G0：复用本连续返工已读有效 Backend Role/implementation/调试及 Go 规则；实际核 main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca、R07 五文件与四旧资源摘要。完整消费 `code-review-close-r07/finding-trace.json`；其为静态反例，本轮才提供离线行为 RED，不冒称旧 PG 反例。唯一源码变更 H，证据仅本目录。旧 R07 freeze/binary/报告、其他三 Go/docs、旧 policy/baseline 均保留。

实现：H1145 真正最终 goroutine 消费者改调用 H1157 `waitFinalClose`。仅该 helper 在 predicate 前后及最终 nil 返回前检查 ctx 和共享 closureEnd；已发生的 predicate 非 context 错误直接保留。全局 pollUntil 原文不变，原 4s 本地等待/共享5s或更早hard、15/5/20、owner、采集和只读入口未变。H5281/5291 回归 expired+已符合、live ctx+过共享期限、回落晚于deadline、期限内成功、观察中取消及原42501错误保留。fixture root 仅移到本 R08 隔离目录；core 使用本目录隔离 CWD，避免旧 fixture 写入。

| 命令目录（完整 argv/PID/时点/前后五hash/rawSHA 见各 result.json） | 实际结果 |
| --- | --- |
| red-expired-r01 | exit1，3.500s；过期 ctx + true 得到 nil 的行为失败，非编译失败；H B03D…85FFB 中间版保留。 |
| green-close-r01 | exit0，8.250s；29 pass事件（含子测试），0fail/skip。 |
| race-close-r01 | race×3 exit0，32.703s；87 pass事件，0fail/skip；包含 R07 真实 puddle 归还/超期 fail-stop 子进程和 R06/R05 错误保留。 |
| build-r08 | integration/race binary 编译 exit0，4.891s。 |
| core-r08 | 新 binary / race×3 exit0，10.656s；57 pass事件，0fail/skip。 |
| vet-r08 / format-r08 | exit0，1.578s / 0.109s；gofmt diff 空。 |

六条 GREEN 都是最终 H **AC2AD1AC172C2E78D898DAAC624CEC35965719023B78E24D87101144E2B23878 / 226915B**，前后 source 一致。完整五 hash/snapshot/原资源摘要/七命令索引在 `source-freeze-r08/manifest.json`。新 binary `tools/r08.test.exe` **C9AFE05D0C48B0A0D126DB2971B958F5F72836A2C49A42A9977A3C9FD8866B71 / 32800117B**，不是旧 binary 冒用新版。R07 前像在 `before/`；精确最小差分 `r07-to-r08.patch`。freeze.py 对当前与前像的全局 pollUntil、只读入口、budget parser/collector/context分类等原文相等作断言；函数摘要取 marker 至下一 func（包含间隔文本），与旧 manifest 的提取边界可能不同，不跨提取器比较摘要值。

只供原 Reviewer 独立复审 F001 和必要 delta，不作者核销。R06 两正例仅历史证据；两真实只读负例仍待新 revision 独评与 Lead 实际窗口授权，未运行。完整性能仍 NOT RUN，任务 active、旧未验边界不变。现在 SOURCE/LOCAL STOP，PG未使用；后续不写源、不跑 Go/PG，交 Lead 接手。
