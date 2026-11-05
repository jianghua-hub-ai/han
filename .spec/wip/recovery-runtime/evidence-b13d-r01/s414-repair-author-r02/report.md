# S4.14 F001/F002 作者有界修复 — STOP，待原 Ptolemy 独立复审

固定 Han/main/HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca；原作者 Chandrasekhar/add-backend-dev/01a187a1-c022-71f3-92f5-6a9dc032e32b。已全文消费修复 START、原 R01 report/findings/receipt/frozen probe/Main reception；本人核验17份原方法同版本后有效复用，Git方法另核同版。原五 before 全部匹配。只修改 recovery_restore.go 和 recovery_restore_test.go，另三文件逐字节保持，详 before.json/after.json/repair.patch。

## 根因与实际修复

- F001：旧 Preflight 校验后关闭 image handle，handoff 后按可替换路径 Start。现在必需显式私有 image lifetime facility，批准的 checked handle 与 lease 从 acquisition 一直保留到具体 Start、child/pipe/callback 全结束及 CloseAndJoin。runner不再自行 exec.Command(configuredPath)；Start只能由该已绑定image的owner执行。缺设施/nil lease在Begin前拒绝；返回owned lease同时报错亦先真join，未以另一次任意path hash代替lifetime。测试设施在handoff前捕获批准字节，具体Start使用该owner的受控当前testbinary副本；原source image变化拒绝、child0。该设施仅在_test.go，非生产fallback或原生平台实现。
- F002：Replace把现有真实 CheckPhase/ReadPermitBinding 的精确当前permit检查交给启动路径，在handoff后及image owner具体Start前消费；ctx.Err不再代替实际lease检查。只读verify借用同一实际 ExclusiveLease.Check，不产生新grant/authority、不关闭借用lease，不增公共Runtime API。future S4.16 root须绑定该同lease检查与真实image facility。
- attempted/unknown与Resume零Begin/Replace保持；部分Start返回已启动cmd和error时仍Kill/Wait/三管道join；新image owner真实终结前不返还部署所有权。CloseAndJoin报错保留根因/取消，不能以未知当资源终结；实际owner仍held的阻塞测试观察FD可用、Stop未返/Close0，释放并关闭后才Close1。Root S4.16原idle artifact owner责任未冒充完成，未新增假hold/Runtime债。

## 当前精确版本验证

| 本作者实际执行 | 普通 | race |
|---|---:|---:|
| 原两top＋作者新增＋原Reviewer三节点 test-only overlay | 92叶，103 pass事件 | 92叶，103 pass事件 |
| 专属 RestorePG top | 7叶 | 7叶 |
| 原限定 root/lifecycle/config 回归 | 319叶 | 319叶 |

全部0 FAIL/0 Skip。机器逐名核原80/7/319全部仍存在并通过，详test-inventory.json；92=原80+作者9+原Reviewer3。overlay以当前完整test文件为精确prefix，仅追加原冻结probe，原probe SHA不变，未overlay生产文件、未删作者新增叶。最终overlay执行是作者自测，不是新的独立Review。

原Reviewer healthy/image/lease精确oracle在两模式均成立：healthy Begin1/bind1/children2/targetRead2/approvals3；image确实在approval2后改变摘要，但children0/targetRead0/bind0；lease同实际facility失权，children0/targetRead0/bind0；Stop/Close实际完成。原独立R01失败raw保持canonical原件，见original-review-failure-index.json。作者先以同触发保留了可编译行为Red：普通/race healthy PASS、image/lease FAIL（各启动1 child）；Red源像与raw在red-source/及author-red-*。不是编译失败或native缺设施反例。

普通/integration vet、原top list、五文件gofmt检查及必要Linux amd64/CGO0 compile-only均通过；Linux制品未执行、留独立scratch不入包。原profile七项全局检查未执行，不作全7声明。

## 资源、证据与并发边界

- process-ledger.json：16个本作者Go parent，实际close/Wait及输出句柄关闭均已完成。child-joins.json：197行actual PID/Wait/pipe观测，按command+PID去重132；包含Red/中间Green/最终Green，并非全Host枚举。所有工具child/callback/pipe经统一真实join路径；新owner阻塞/终结错误/已启动cmd错误均有对应新增叶。
- 本修复新增14个fresh quoted schema（7普通+7race），全部保留，见retained-schemas.json；仅固定0001–0010与唯一RestorePG51934/han_b13d_restore，无旧DSN/autoDROP/glob/globalintegration。连接、事务、pool实际结束；Main PG server未动。binary copies/artifacts全留自有OS Temp，exe/cache不入包。
- 私有restore_dsn仅在授权内存/private stdin传递，ordinary DSN为空；无原child stdin/stderr落盘。raw-index.json扫描全部Go raw的完整DSN/synthetic canary为0命中。
- B13C19、Runtime2、共享root、modules、两design、identityTP等本次实际冻结检查匹配前像。ArticleTP未读/hash。Main后来明确授权Goodall写Tasks/Plan的S410/S412/A48/B13D技术映射；自该消息后未再读/hash两活跃SSOT，不声称其整文件仍旧hash。前像与精确并发writer记录在metadata-concurrent-writer.json。
- Git只做own5范围status与no-index diff，无Git写/提交/push/切分支。未读取对方Temp候选作为自身通过证据。

实际native OS锁/ACL/安全同handle传递/批准工具与原生receipt/process isolation仍未固定，真实启用Gate保持CLOSED；真实restore/root/E2E未验证。测试owner的受控image副本只证明现有软件lifetime/Start协议，不能记native通过。

原两个HIGH仍待Ptolemy对该新freeze独立复审；作者不关闭Finding/Task/G4，不代Main接收，不声称全7/native/root/E2E。当前STOP，无继续源码/Go/PG/child动作。
