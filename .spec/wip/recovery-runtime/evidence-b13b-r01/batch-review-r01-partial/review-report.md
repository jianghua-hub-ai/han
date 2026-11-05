# Curie B13B 跨任务 Code Review R01：中止交回（PARTIAL）

**整批状态：CHANGES_REQUIRED；当前 HIGH 尚 OPEN。自身跨卡检查未完成，不能给 batch PASS 或完整独立结论。** 按 Main 最新指令在当前安全节点停止新增分析/探针，交原作者定点修复，再由原 Ptolemy 复审；后续 R02 由本 Reviewer 补剩余与有界 delta。

固定 Han/main / HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca（本轮按 Main 固定交接，无新 Git 操作）。主对象 Journal11＋HTTP2共13，compat2为单独已接收的补充依赖；没有扩大原卡写集。原 HTTP2 Review 的0 finding及其 Main 接收继续有效，不重开、不重跑原56+6。

## 实际对象与读取范围

已在入口独立核 source-sha256.json、canonical author index、Main joint index 的 SHA，分别为 2F1E7757C82E4B4AED4E85029052B8EC4C46461AADF15E6401E88DE608824627、A5F8F81E3085E924F6EF4C3A2DD4ACCE46AD3A98CD26DD5F4AC98B874A9F9765、B31EE04DFFC09D478019180632D8C238095922EC45736881B2A42B0FB23546E3。11源实际 live SHA=manifest.after，且与作者 frozen 逐字节相等；HTTP2为原本人已核版本复用及 Main当前冻结交接，本轮未重哈希。未声称完整审计两个 index 的所有payload。

| 主对象 | SHA256 |
|---|---|
| internal/recovery/journal_snapshot.go | 5F603DD09BAEC5DF69D6C2BD2B2D44F9BB2D4D1425FD2068EEFB94D4D56E38AC |
| internal/recovery/journal_snapshot_test.go | F1A6797557BEB87E4C80B384B1A126762816BF77E8DF0EA3E2837DC9E2A2D929 |
| internal/recovery/coordinator_context_test.go | 33CC74827F8A204E630681CEE25814996C978EAFA8869B5E93B859C53DF551BC |
| internal/recovery/journal.go | 52BA8DD83911696B1D1387F3555B8EF63B5128136CFA9A59EC2D32D422AB464D |
| internal/recovery/coordinator.go | 14E56074C88B197E64500BD4E9121C5683AD5CF3ED2CE602F36689C9EBBD0BFA |
| internal/recovery/journal_test.go | EB51DB110721E4017316EA477D54ADEBAA0A799BCE6142E03C3F1A6B2B0BDF28 |
| internal/recovery/journal_windows_test.go | 03FC45B08A248F428D63219479A0D720136CEB9D027F4F9156F0DF342BB27E7F |
| internal/recovery/filelock_flock.go | D7A5BA43B1E9E47BC2F4C7B8AD72AED6F206ED5E56688AD665463C96036457E3 |
| internal/recovery/filelock_unsupported.go | 4AD82AEE6250919AB4236FCF367FAC006D26205378409591E88611E160359765 |
| internal/recovery/filelock_flock_test.go | 8C97D76682FB10F3A7CAC5A5FF74816B70325726F8F486DDEFA30A085F9A90E3 |
| internal/identity/recovery_run_store_integration_test.go | CADC4A4529ECAB2135FA2BAF3CCF632A5FC0E0B922272F2B528C50C78B7DB1F2 |
| internal/platform/recovery_guard.go | 0187E9A0B40FFC465240BFA8FF40FFD9DA7DB8A69B378F1931D94B6974C1826D |
| internal/platform/recovery_guard_test.go | EB4598546D62B932D053304BE4DEA45DE9C3CD5BD3BE6B4558C168A8764397F3 |

稳定依赖的实际必要段读取再次得到 Runtime 284886C23EA1038B685F1EA5BD1A6B1BC423077042939687C6EF062392594384，admission E234CC71C5A21D42EFD6E8B345F2499800B607F2D8AF66043A0EB1A4D38FD3F0，admission_test 02E9232EAC4A87F81582038304C1B06D6C6381DE2959CD45B95F7286B4CBFB0A；没有修改它们。Role/Skill/Rule/go-service与r52/r23/TP/prep依据沿同版上下文复用。

## 两轮实际进度

Round1 的直接差量/新增内容检查已覆盖：新 journal_snapshot.go 全文、新 journal_snapshot_test.go 全文、新 coordinator_context_test.go 全文；journal.go 全部修改hunk（diff1473–2469）、coordinator.go完整旧新差量（2470–3137，另读最终全文）、旧 journal_test全部差量（3138–3556）、两平台锁实现最终全文与其差量、Windows/锁测试及integration三处ctx/budget差量（3557–3882）。新文件通过已核live=frozen正文读取，未用摘要代替。检查了纯读/retained key、六依赖、旧调用迁移、fresh/publication与unknown流程、测试夹具迁移保留行为断言；尚未形成新 finding，不意味着这些维度获得全面PASS。

Round2 已开始但未完成：定位实际 NewFileRecoveryJournal/NewCoordinator调用者；读取 stable Runtime Resume/phase/refreshAfter/acceptRecoveryObservation/confirmSucceeded 必要段、admission复制/完整快照比较/exact append、实际keyring lookup、原纯validateRecords，以及公共Runtime与新Coordinator测试fixture的必要构造段。已核到 RecoveryJournal 是 ContextRecoveryJournal alias，原纯validator调用形状保持；Coordinator私有ctx观察只限制既有phase，实际授权仍来自Runtime。未完成从所有实际副作用边界到锁/ctx/当前authority的连续审查。原HTTP的完整评审只复用，没有新root/SessionSQL读取或验证。

准备过的下一步思路是“真实纯读parser/reader→真实Runtime normal/受控Coordinator→HTTP”的最小合成组合，未写成probe/overlay、未执行；不把构想或作者fixture当独立执行证据。

## 已知阻断 finding（原 Ptolemy 归属，非本人的新复现）

**[HIGH] RECOVERY-B13B-S48-CR-R01-F001 — writer仅整次operation前后验锁，失锁后仍继续下一写操作。OPEN。**

- 位置：internal/recovery/journal.go:702–714，operation()后才再Check；关联CAS:410–448、stageRecords:1121–1139、persistRecords:1068–1087与pending repair:803–820。
- 依据：identity r52:1520要求失锁在任何写入前拒绝；ctx有效不等于持锁身份仍有效。
- 触发/后果：CreateTemp或首CAS同步返回后锁身份失效、ctx仍活；下一步未重核held lock。原Reviewer普通/race实际观察失锁后仍写396bytes，以及允许第二CAS，最终检查失败不能撤回中间写入，破坏独占写边界。
- 建议：原作者在原11范围携带operation-scoped held-lock连续性检查，已返回同步动作后、下一持久动作前核锁/ctx/适用authority；保持真join、一次Release、原cause与必要ErrRecoveryWriteUnknown，不静默重取锁或重设来源。
- 证据：原报告 `C:\Users\J00010385\AppData\Local\Temp\han-b13b-s48-review-r01-24ef1a5fa27a4fabbc59858fe1a48438\review-report.md`，SHA `5E82E5C7516472380582698BB1C4D62EB8B10F81ECC09E746EC3790E4A512763`；receipt SHA已实际核为 `FB4431FE408F77C62200907F11D6CD9535E5121A86FBD3832C5255EF63AE7B91`。全文已读；其中独立normal/race的两个具体FAIL与控件为 Ptolemy 执行，未由本人重跑其raw/探针，不冒自己的发现。
- 复审：原两行为失败、正向staging/committedACK和真实嵌套production边界等价失锁场景须在修复SHA通过，保持取消/unknown/无后续写/实际join/一次释放；原Ptolemy复审并由Main交接后，再核本批受影响接缝。

本人新增跨卡 findings：**尚无具体 finding；PARTIAL，不是0缺陷保证**。HIGH尚存时不建议接收本批。

## LOCAL / raw / 进程与证据状态

本轮 **Go调用0、测试调用0、测试overlay0、后台进程0**。仅复制了安全runner，尚未调用；所有工具exec是同步只读Python/rg和自有Temp文档写入，均已返回，无PTY session ID、无待wait cell或已启动未join的Go/测试子进程。一次rg Windows glob解析错误是文本导航失败，不是编译/测试Red。本轮没有独立Go raw，也没有测试PASS可汇报。

readset.json/readset.jsonl、入口manifest核验、fixed-source11.json、upstream-review-reference.json保存已完成读取/哈希与来源；工具输出为本会话已返回的只读原始输出。本轮未做根fmt/全仓七项/原HTTP56+6重跑，也未完成Main408叶/七项全部raw和77payload的独立解析。Main提供的2/0/0/0/0/0/0、source15fmt、旧23相同及新有限处置保留为Lead证据，不冒本人验证；有限fmt处置不豁免本HIGH且修复改SHA须新核。

## 剩余未核 / R02最小续接

1. 原作者修复对象、完整delta、新精确SHA/测试raw、Main冻结与fmt处置适用性；原Ptolemy对F001闭环/新增风险的真实结论。不能把尚未返回或修复自测当关闭。
2. 补完Round2 writer所有持久步骤/锁-ctx-authority连续性，pending repair、ACKunknown独立readback、真join/原因保留及reader与受控writer隔离；从原已读范围复用，只读修复delta和未覆盖相关段，不整包重做。
3. 补实际同Runtime源证明→reader/retained key→受控Coordinator/fresh/publication→HTTP工作许可接缝的最小合成定向探针；无生产overlay/私造许可/production fake，不重做原HTTP全部probe。
4. 完成Main同最终组合的408叶/原271保留、作者Red失败历史、Main7实际raw/有限格式处置适用性与必要覆盖核对；原两卡PASS不能相加成batch。
5. 尚未完成性能/覆盖全面评估：已识别全链/预算/取消与重复解析路径，但未测吞吐/P95/容量、未出完整风险覆盖矩阵。最低U/C＋race组合及源未变证据继续复用；integration/linux vet仅静态，不作P/native执行证明。

真实factory/root同实例装配、同Store/Session/publication SQL/PG、nativePOSIX锁/介质、真实provider/retained-key服务、restore/三worker、Case18/S8.2/Host/部署/E2E仍在原S4.16/S4.17及关联责任，不因本partial补造PASS或新增Task/写权。协作取消不承诺任意I/O强杀或Hijack后ownership。

**STOP / FROZEN。** 未改生产源/Task/Plan/设计/Rule/Scope，未Git写/清理/子委派/真实ENV或PG/服务/下载。本primary只记录已做事实和未完成范围；整批CHANGES_REQUIRED，待原HIGH修复闭环及后续R02有界收束。
