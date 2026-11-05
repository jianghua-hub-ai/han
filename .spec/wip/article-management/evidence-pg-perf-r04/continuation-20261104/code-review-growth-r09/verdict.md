# R09 growth执行证据独立结论

**Gate：PASS，仅限本次growth负例目标的有界接收。** 未发现阻断该结论的新缺段；离线原件核证138项全部成立，详见evidenceaudit.json。实际Go仍为自然 **exit1**，没有改写为测试GREEN；通过的是触限取消及正常资源收口的独立证据门。

固定对象：author/followup-attempt-r01/gr-111112-25876。R09 H `C3DFDC66FDBFB025760ED78B0E0061CC49BD7AFB2DA0BF76685D94D0D89B8B99`，非race binary `71B3EF47D466A8EAB899C266311EAD224E5F5316D6B244A8E2D7EFDF705521D5`。实际snapshot、live五文件仅哈希核验、build/launch/source-before/after、policy/grant均匹配。

- 9份采集回执对应6次实际resource-size SQL读；flight4/5各有一个共享回执，不重复计SQL、不刷新源START或请求截止。另有一次terminal拒绝，未执行新SQL。
- 准确起点2,029,926,079B；4,622个成功批次/46,220行；注入阈值2,032,023,231B；真实typed cap观测2,103,916,223B，root取消、停止后点。该阈值不是物理增长上界。
- 外部静止SQL最终2,104,227,519B，真实增量74,301,440B。触限观测至最终差311,296B原值保留；触限扫描区间与写入重叠，不能将该差额全称为“触限返回后新增”。不clamp，不虚构逐批物理归属。
- 共用原5s关闭截止，实际133.985ms；worker joined、连接/事务归还、各pool Close、goroutine2→2、残余backend0。Go4876自然exit1，15个owned客户端实际wait齐全；外部精确app查询为空。
- 旧83个schema全保留，仅新增并登记sr2seed_bc59478014d93ae6。原resource origin及small80B/SHA保持；最终累计697,294,848B，small剩376,446,976B。

局部状态：本版growth/共享flight/真实cap/关闭子条件已有执行证据；不把本例局部PASS改写成原finding整项核销。原 **6 CLOSED / 12 PENDING** 保持；leak、N9及未执行的W仍PENDING。旧失败/预检拒绝均保留，未移植R08失败为PASS。

时钟边界：Tnew10:22:37.932710Z不变；本attempt完整calibration/small/medium已永久NRU，旧完整性能NRU、large/canonical NRU不变。原12:47:37.932710Z停新、12:52:37.932710Z硬停不顺延。本结论不批准leak或任何后续执行，不关闭S2.4、不构成性能或E2E PASS。

Reviewer只做离线读取、哈希、静态证据核证和本目录写入；未运行Go/PG、未采用参数或修改源/任务。最终实际STOP及报告哈希见index.json。
