# S2.4 follow-up r03 design delta verdict

**CHANGES REQUIRED：1项MEDIUM（FUP-R03-001）；P0及执行门PENDING。** 没有计划整体PASS、运行授权、finding新核销或性能PASS。

正式对象test-plan-followup-r03.md，21064 bytes，SHA **7B0553AEBBCD998A4BAB70B1B92CA87DA0398058191F31CCFEDEA40FEAF1C106**。原非作者Reviewer两轮仅核本delta及必要直接消费者，未全库源码重审。

- **需修正的确定接缝**：新policy/WINDOW_START=Tnew与原ledger.Window=11:29不一致；H:3038必拒绝，H:1451在CREATE前触发。候选“若不能表达则返回”的停止保护正确，但须明确Tnew/P1之前的具体时钟/资源身份绑定合同与验收。返回原TestEngineer，方案由Lead选择；Reviewer不改源、不替Lead决定，不要求非代码计划先build。
- **已核自洽**：12项L/N/W无遗漏重复；P4允许纯W取证条件继续pending，可避免互锁；145/150/53min算术正确，旧批次NRU永久保留。121/60s、累计tail1s、完整校准→small→medium、large/canonical NRU不减。
- **资源仍conditional**：原base1406932671及全部已有档位baseline不重置；历史small余额487686144 bytes≈465.09MiB，必须重新预检，不保证完整small容量，不清理或换库。
- **P0未解除**：已完成诊断有SQL本身6.955s、显式路径stat5.891s等事实，不能凭较快样本认定当前5s可靠、根因或修复方案已接收。根因最终交回/独占责任及接缝闭合前不运行。

用户已授权建议值，无需再次询问；Lead后续实际采用不由Reviewer代替。原6 CLOSED / 12 pending / E001 CLOSED、旧批次完整NRU、任务active均保持。关闭条件与两轮实读范围见 [report.md](report.md)，摘要/算术/消费者核证见offline-evidence.json。

唯一写集为本独立评审目录；未Go/PG/凭据/Git/清理/委派，未修改作者文、旧Review、Tasks/Plan或源。完成后 **STOP**。
