# R07 close delta verdict

**CHANGES REQUIRED。唯一新增CLOSE-R07-F001（MEDIUM），无确认HIGH。**

固定Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca，objectType=code。H **223792B / 9F08F70B2E2063D5F178064F5EA0D9F84DABB1BA625CEF90402DE70D4CB7CAC2**；五snapshot/before、9个patch hunk、旧资源摘要自行核。

归还等待与有界Close已正确共用原5s/更早hard，真实puddle asyncDestroy红例及green/race有效。但H1143～1150末尾goroutine检查使用pollUntil（2449～2454先check成功后查超时），在closureEnd已过且计数此时回落时仍可能nil/failed=false。最小关闭条件：仅该消费者拒绝截止后的成功，补expired+已符合、晚回落、期限内成功三条有限回归；不加预算或重构。不声称反例已在raw发生。

8条raw记录核完，最终同hash6条均exit0：普通20pass、race count3合计60pass、core57pass，无fail/skip，另build/vet/format。真实noncontext/mixed错误及原R05 Closing语义不回退；采集/身份/入口/15/5/20未变。未来须新binary/当前hash，不跨版冒结果。

R06两正例仅历史佐证；在上述修复/实际新freeze/Lead接收与许可后，两真实只读负例若包含同版准确成功前置、更早freshness取消及正常有界归还/完整元数据，可构成最小只读P0闭环，不强增独立成功smoke。实际缺哪段才补哪段，本报告不批准执行。

growth-reuse-addendum已实读，与现FaultSeedDisk入口可复用映射一致；同一阶段准确baseline→真实增长→触限取消→外部静止SQL核账即可，不因缺独立命名入口加门。当前未有新growth结果，不核销原12finding。

仅写本目录，未Go/LOCAL/PG/凭据/源或旧物证改写/任务/采用/Git/清理/委派。**R06运行BLOCKED、6closed/12pending、完整旧性能NRU、任务active保持。** 详见[report.md](report.md)、evidence-audit.json、finding-trace.json。完成 **STOP**，摘要/actual STOP见artifact-sha256.json。
