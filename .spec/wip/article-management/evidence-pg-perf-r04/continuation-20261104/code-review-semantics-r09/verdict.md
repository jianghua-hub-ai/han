# R09正常语义实际证据独立结论

**整体Gate：PARTIAL / PENDING。** Go40712真实自然exit1，8个正常子分支PASS，最后10k在seed120s父context到期时FAIL，drain未到达，九分支整组不能PASS。离线原件/版本/资源核证932项成立，不代替缺失业务证据。

**原18项本次后累计9 CLOSED / 9 PENDING**，从7C/11P基线仅新增：

- **F015 CLOSED**：sameWorker=false/true两份真实lease回执，attempt1→2、lease更新，旧结果晚到后accepted_at/attempt及清空claim/lease的稳定性断言通过，worker已join。
- **F016 CLOSED**：原始锁图三不同PID，C43312→B39868→A38348，真实Lock wait；固定消费者后续正常释放/join，资源归零。

F011/F012的竞争动作、三PID等待图及唯一赢家/版本/业务SQL断言确实运行并PASS；但**原followup-r04最低N合同所列双方返回值、最终行/审计快照没有保存到原物证**，不能报告实际哪方赢或原值。证据缺口`SEM-R09-EV001`保留，两项本次不核销；不把它定性成仍有源码HIGH，也不要求盲目重跑8成功分支。需以当前保留schema的可验证原值/绑定补齐，缺失的原参与者返回日志不得事后编造。

F009仅完整1k子条件满足（accepted delta=sent=1000、failed_terminal0），10k seed失败及W均欠；F013/F014/F017的本次N断言/跨页物证/三份生产SQL ANALYZE BUFFERS计划有效，正式W仍欠。F002/F006/F007及这些N+W项均PENDING。

CLOSED：F001/F003/F004/F005/F008/F010/**F015/F016**/F018。PENDING：F002/F006/F007/F009/F011/F012/F013/F014/F017。原E001保持CLOSED，未重开。

R09五snapshot/live/source-before/after、binary71B3…521D5、policy7C94…33E04及grant均实核同版。旧85个schema全保留，新增9个与inventory及实际差集一致（最终94）。全部9套资源共享各自原5s截止，关闭13.6744～124.5542ms、goroutine3→3、owned连接/事务归还、backend0；外部范围为空，100个owned客户端均真实wait。最终准确DB2,129,835,711B、原origin累计722,903,040B、small剩350,838,784B，未重置/清理。

完整calibration/small/medium本attempt永久NRU，旧完整性能及large/canonical NRU不变；全局hard12:52:37.932710Z不延长。本报告不批准后续运行、不改任务，S2.4 active。Reviewer未运行Go/PG或修改源。最终报告/审计SHA与actual STOP见index.json。
