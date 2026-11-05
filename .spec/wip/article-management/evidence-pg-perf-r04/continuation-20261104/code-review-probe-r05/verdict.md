# Combined probe R05 verdict

**CHANGES REQUIRED；CLOCK-R04-F001 CLOSED；新增唯一实质finding PROBE-R05-F001（MEDIUM），无确认HIGH。**

objectType=code，固定Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca；只审clock-r04→R05 budget delta/F001及必要直接消费者。H固定 **209145B / E97318D68C8BE670DF11A196D469A3D91F5BAC979A1CF65EFDDCBAE4F09DAFC7**，五snapshot、31个patch hunk、旧资源摘要均独立核。未以live H替代快照。

PROBE-R05-F001：H1854～1858/1867～1871在ctx已取消时丢弃真实非context Acquire/SQL错误，Closing汇总可能只见纯取消而漏记资源失败。现有Closing回归直接注入collector，未覆盖此适配器。有限关闭条件：保留原错安全分类与取消状态，验证实际适配器/其直接helper的“非context错＋正常关闭”仍失败、“纯context正常关闭”不误报；不泄露凭据、不增框架或修改预算。

F001：真实loaded ledger/receipt identity已进入每harness fingerprint；同版3份raw fingerprint各含首次/复用/后续existing，已逐一复算实际ledger/receipt长度/SHA和恒定基线，policy授权不改。因此独立CLOSED，仅限证据消费者缺口。

21条result/raw/hash已核，只有最终6条同五hash且exit0：race-budget-final 33.954s/count3合计243pass0fail0skip，core-race-final 10.906s/57pass0fail0skip，以及build/format/list/vet。旧失败/旧版保留不跨hash。binary构建→sandbox CWD运行路径及当前摘要已核；并非Reviewer实际运行。

15/5/20单调START锚点、首次关闭门/独立expiry/准入/late终态、原观测cap、root/phase真实owner及121/60/tail1已按两轮静态路径核；受上述错误传播finding限制。只读PG入口仅编译/list、无newHarness/DDL，实际PG尚未执行。

**P0真实SQL/增长/取消/泄漏/资源归还与执行Review仍PENDING；原12pending不自动核销，Tnew未建立，旧完整性能NRU、large/canonical NRU、任务active不变。** 当前无整卡/性能/E2E PASS。后续仅新delta＋本finding复审，不重复全部raw。

详见[report.md](report.md)、evidence-audit.json、finding-trace.json。唯一写本目录；未Go/LOCAL/PG/凭据/Git/清理/源写/旧物证改写/委派。完成 **STOP**；actual STOP和摘要见artifact-sha256.json。
