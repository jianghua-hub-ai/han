# B13E whole-batch G4 独立 Review R01 — BLOCKED

Dirac/add-reviewer，实际 ID 01a18663-d9ac-79c2-b843-ba6dcca3de7e；非本批 Lorentz/Hooke 作者。objectType=code。固定 Han/main/HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca。本批 S4.11 原12源＋S4.13原5源、冻结6文档，精确23 SHA 见 objects23.json 与 objects23-exit-release.json，逐命令前后及 release 全相同。

**整体结论：G4 BLOCKED，新增1条 HIGH。** 两个原独立组件 CodePASS及其旧 finding CLOSED 继续按原版本/范围适用；不能相加覆盖下述真实组合失败。本 Reviewer 未改源码或 SSOT、未替原 owner 修复/接受风险、未关闭任何 Task。

## [HIGH] UTC 会话中同一删除时刻因 Location 表示不同被判错绑，合法恢复无法进入信任根准备

- **稳定 ID**：RECOVERY-B13E-G4-R01-F001，OPEN。
- **主位置**：internal/identity/recovery_execution.go:817；相关消费者 recovery_auth.go:86、recovery_execution.go:894、runtime.go:1743。原共享源 SHA 6D80E8D1565298F03734FAAD3C8F41F3395C84167A737C82A79436FBB0B2288E；当前 Auth 源 SHA 346ABCE35C915F6487F937058D481DD947528BC4D31CCD5872CCDF2BDFD8001B。前者为现役已闭 S4.10 消费者，不在本次17源写权内；本 Review 没有扩大任何写窗。
- **触发**：真实同一 PG pool 使用 UTC（原 S413 PG 构造器即固定该值），合法恢复账号集合包含一名已删除账号。独立批准的 anchored_migration 安装经真实公开 Runtime.Install、真实同 Store BindInstallation 和 Runtime 后证达到 bound。实际公开 Enter 使用该安装产物与同一外部 Journal 夹具，实际 BindRestoredRun、FinalizeRestoredIdentity 成功后调用 PrepareRecoveryTrust。
- **预期**：不可变 Source 与 Review 的 deleted_at 指向同一合法微秒时刻，应视为同一事实；保留 nil/非nil、角色、状态、账号集合及其他全部校验。合法组合应进入信任根准备及后续实际维护认证。
- **实际**：原 finalization 把两个 *time.Time 用 reflect.DeepEqual 比较。审计 JSON 保存的同一零偏移时间格式为 Z，Review 的 PostgreSQL JSON 为 +00:00；解码后的 Location 分别为 UTC 与匿名零偏移 Location。观测同为 2026-10-29T07:00:00Z，time.Equal=true、DeepEqual=false，role/status 均相同；函数返回 ErrAdmissionBinding，尚未写 bootstrap proof，Auth Login/强改密均未到达。
- **真实独立证据**：checks/normal-r02 与 checks/normal-r03 均实际 exit1，E-X01 PASS、E-X02 FAIL；后次仅加入不改变返回值的只读时间观测。actual-observations.json 保存精确原输出。两次持久化实物均为 installation bound对应行1、running run1、restore_binding1、reviews2、bind/finalize Audit2，bootstrap/changed proof均NULL、Session0。4个本轮新schema已逐名真实只读SQL核验，24表各自存在。失败不是编译、缺表、批准未执行、fake phase、未等待资源或真实 restore 环境不足。
- **影响**：包含正常已删除历史的合法 UTC 恢复组合被保守拒绝，无法准备维护信任根或完成恢复；安装已 bound 并不能消费到维护 Auth。该失败保持关闭，没有观察到越权或数据损坏。它是本批真实 producer→consumer 的可实施性阻断，不是要求未来 S4.16/root/Case18 先交付，也不自动推翻旧已闭卡的所有历史结论。
- **最小建议／owner**：交 Main 安排原 identity 共享消费者 owner（原 S4.10/Zeno）核对并最小修复此时间事实比较；正确区分 nil，并按同一时刻比较，继续拒绝真正不同时刻/状态/角色。不能以把 pool 改回 +08、改 fixture 时间、忽略删除历史或放宽整个证明来绕过；不需要新设计、批准源、API、Task、DDL 或重做所有上游。
- **复审条件**：保留本次两叶原断言，当前修复冻结后由独立 Reviewer 实跑普通/race；合法 UTC/Z/+00 与非零偏移同一时刻接受，真正不同微秒或 nil错配拒绝。原安装、Auth风险组和已闭 finding 的受影响回归按精确差量选择，不机械重跑全历史。原 failed raw/schema/probe版本全部保留。

## 两轮整体判断与实际增量

Round1 已在静态阶段独立读取四个新增生产文件全文、四个既有生产完整差量、当前安装修复两文件完整diff，并对照两卡原5+5风险组及 r56/TP26、Article r27/TP、ADR0027。Round2 沿真实 Runtime/install/同Store/原legacy绑定/身份终结/maintenance Session/Gate/Audit/ACK/Resume/资源追踪。静态读集及六个 seam 见 Main 保留的 batch-g4-static-r01 原包；本 actual 根只追加新的读集，不回写静态 primary/manifest。

静态阶段未建立新finding；本次按获批最小组合取得新增真实反证，因此整体不能 PASS。现有 S413 实际安装 PG 与 S411 预置 installation 的 PG 分别成功，未覆盖本次 UTC 同 Store 组合。新测试没有复制、INSERT 或 UPDATE installation 来凑绑定；只走实际安装器。外部批准/CAS/Journal/restore/exclusive 为明确 software fixture，真实 Store/Runtime/身份/Session/Gate/Audit/SQL 不替身。

E-X01 在实际 prepared/install ctx/permit 下测试真实维护能力拒绝；Auth User/run/Session/Audit整像零变化、credentials0，安装自身允许的 preparedCAS1/Media1 单独计数。该叶连续两次 PASS。

E-X02 实际安装 bound、停止安装 Runtime，再以同 Store/pool、同外部 Journal/批准/commitment 与真实 InstallationReader 构造恢复 Runtime；实际 Enter→Bind→Finalize 后在 Prepare 阻断。测试原目标的 Login/Resolve/Change/G+1/两改密Audit断言保留，未到达不能记 PASS。**race NOT RUN**：已重复确认具体产品阻断后遵守停止合同，没有为计数扩跑。

首次 Go 尝试使用了不存在的 Runtime.PublicReady，产生编译诊断且0产品叶；原件保留，测试改为当前真实 Acquire(WorkRequest) 必须拒绝。该编译诊断不算产品 Red。后两次是实际2叶各1PASS/1FAIL，不能混作夹具诊断。

## 正式上游输入已核入，不能遮盖新增反证

Main main-current23-receipt-r01.json SHA 05A9AC1A86E870F11FB93FDE107052FFDCDEF3A1D220062D3271A91DFA140428：23命令已实际结束，14定向JSON普通/race逐名为32/25/56/2 U/C、Auth42P、Install23P、原S41086P。本文沿用本人静态解析的8+15实际raw/SHA/exit/source23，并机器核正式302个精确schema实物记录，非本Reviewer重跑。Main302-name初次大SQL超时与client kill/communicatejoin历史保留；后续19批每批max16、各自只读RR/psql0，不冒一个全局RR；不把环境诊断改产品失败。

Ptolemy R03 原HIGH/MEDIUM独立CLOSED、97每模式及5个原fixture诊断保留。Main正式 reception SHA F6C48E3E692BA4FF9D75B4FAE3BC3BB1B7650352794BF663233B2398874C67B4 已核，canonical170载荷本人逐项SHA/bytes核对。Main16jobs/50SQL已接收；遗留打印literal9与canonical数组/len=16的更正不改变产品结果。真实C1 controlled Article core/同handle snapshot/Runtime postproof的 software Store/CAS 证明不与独立25PG拼成 root/E2E。

项目七项仍 **6PASS / rootfmt FAIL2**，owned17格式空PASS；root stdout D01225B146C06C238B38AEA344662D8253842CEE671EF68137BCC4B66C9379A8、stderr 53258F7B21EF40C6BE7E8E671216EEF5DD0441B27314FA806AE24F5C2CB010FD 和原23输入不变。本报告不将其改 PASS、不继承 blanket waiver。普通/race包cache命中不夸uncached全仓运行。

## 资源、性能与未验边界

本轮3次 Go＋1次只读 psql 全实际 parentWait；74个取得handle全部 Wait、4个 kernel Job-zero、双EOF/pump join、Runtime.Stop及callback返回、Rows/Tx结束、pool.Close/acquired0均有收据。4个新schema实物后像已核且 retain。source17+doc6逐job前后/release均exact。**soleLOCAL/identityPG已明确 RELEASED，pending0；后续仅本人D封包。** 未停止或改配置 Main 集群，不声称全Host/他人进程join。

安装完整 Store/双namespace重读、Auth全量投影/cohort/Audit及 Argon2 持锁成本的静态适用性沿用原 performance-applicability；本次只补功能接缝，不签容量/P95/大历史或把 CPU1/-p1当负载结果。新增合法功能阻断优先修复，不能用组件测试数量或小fixture性能抵消。

真实provider/ACL/批准主体/SMTP/native正向/Unix volume、实际DB恢复、完整root/S4.16长期handle/公共切流/Case18/E2E、人的精确checkpoint/接纳均未验证，不倒挂为本批额外门。ADR0024 decision closure非Unix证据；S3.3/S3.5窄豁免非PASS，S3.6不变。phone/缺row残余/unknown hold/deadletter安全合同未放宽。

静态读集过程更正：续接时误用旧helper新增7条，后将完整扩展字节保留在新根并恢复旧log原manifest前像；因此非append-only。Main最终确认原10844字节/SHA265C519366BD5A4711A6ADF235A820D97CDFD2EB82533D480F65EC21AE4F851E另有留存，11970 continuation独立保留。该过程日志不属产品finding，不重建旧包、不再改旧primary/manifest；当前全部新读集/报告只在本actual根。

**STOP / G4 BLOCKED。** 交 Main 与原 identity owner；不自行修复、重开source、关闭Task或接受风险。cachefree delivery保留所有raw、失败版本、SQL、来源/哈希与资源账；cache/work/modules仍留实际D根，未删除、移动、归档或子委派。
