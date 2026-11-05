# S2.4 probe budget r01 verdict

**CHANGES REQUIRED：1项MEDIUM（PROBE-BUDGET-R01-001）。**

固定design对象test-plan-probe-budget-r01.md，14802 bytes，SHA **22F490EBEA79ADE0E6C8D459CE88781C3A19750461A04CC7B2230AF71BD69112**。两轮只核合同、open-handle真实反例及旧H直接消费者，未执行测试或审在途新H。

- **Finding**：20s成功结果完成间隔不能推出20s变化识别。两次14.9s采集、15s间隔，已扫描文件在t=0.1被在途写者增长，可到t=29.9才识别；所有15s/5s/单飞条件仍成立。
- **聚焦关闭条件**：按Main倾向，将20s明确为单调时钟新鲜度/停止上限；从上次完整成功采集的开始计龄，不从finish刷新。新采集也受旧freshness截止和更早parent限制，到期即cancel/停止，不等scan结束、不允许迟到成功复活。无首次成功值不开新写入；反例在t=20停止并有原始时序/资源回归安排，不能等29.9再刷新。15s只是最大机会、tick5s保留，不保证硬实时。原TE修一次固定delta即可；计划Review不要求Go/PG。
- **待决定，非原必须项**：每批TOAST/index等物理增长上界是新增更强约束；harness observer逐phase取消/join并计原1s tail也是新的owner选择。原合同是累计准确观测、触限停止新点、安全失败取消在途，不自动要求影子quota或把一般harness关闭算phase操作尾部。结合Lead最小方向由原作者明确delta，不代改。
- 准确SQL>5s、缓存枚举active-open增长低报均有真实证据；显式15s候选须披露新负载/取消代价，不能只换timeout。原caps/base/ledger、121/60/tail1不因预算增加而放宽；full NRU分支保留。

原r04 clock计划PASS保持；本候选未获整体PASS/采用，不接收在途H实现，不核销原12项。P0、源码/实际回归/执行Review仍分开，旧履历与任务active保持。

详见 [report.md](report.md)及freshness-counterexample.json。仅写本独立目录，未Go/PG/凭据/Git/清理/源码写/代修/委派。完成后 **STOP**。
