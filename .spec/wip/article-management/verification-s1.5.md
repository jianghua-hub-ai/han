# S1.5 文章契约测试：Lead 核证与交接

本记录引用 Tasks 的原 `article-management-S1.5`，不维护另一份任务状态。固定工作区 `D:\AID\VSCode\Codex\Han`、分支 `main`、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；本轮未提交。用户授权采用建议值继续，并已确认另一会话处理身份模块。共享生产入口当前 writer 的事实仍待答复，不能以哈希相同推断窗口释放。

> **当前状态（2026-11-05）**：`article-management-S1.5` 已关闭（`[x]`，完成@2026-11-05）。整卡复审 R02（`evidence-s1.5/wholecard-review-r02/`）判 PASS FOR LEAD GATE，Lead 已做同 revision Gate，并已按 ADR-0028 登记 AI 代理接收（`acceptance-s1.5-r01.md`）。下方各节按日期保留历史时点结论，“整卡仍进行中 / 仍 `[-]`”等表述不再代表现状态。

**最新组件门禁：修复版 r03 的有界 CodeReview R02 PASS，原 F001 已由原独立 Reviewer 关闭；Lead 已核证接收本轮四文件组件交付。** R01 CHANGES REQUIRED 和假绿对照保留追溯；S1.5整卡仍进行中，完整关闭继续受原卡十二requires及真实层阻塞条件约束。本结论不是全生产路由、真实审计/应用ACL、全局恢复或人的成果确认。

## 真实审计 SQL 子集：独立评审及 Lead 核证完成

**Lead 已接收本次四场景真实 SQL 子集交付，S1.5 整卡仍进行中。** 新文件 `test/integration/article/audit_insert_rollback_test.go` 固定 SHA256 `468964647524860CEEF0167EC56A9C0033FBB40CFF340A5021FBCE66F2C89CD5`，原178项完全保留、当前完整输入179项。Dalton 的最终普通/race与 Erdos 的独立普通/race，分别各 **5 run/pass、4 direct leaf、0 fail/skip**；Mark、Restore、Transfer、Finalizer 四项均真实命中。两方 tagged vet 均 exit0。这里不是5项业务验收，也不是对真实文件 Journal、全部删除类型、应用角色权限或全局恢复的证明。

[Erdos 独立 CodeReview R01](review-code-s1.5-audit-sql-r01.md) SHA256 `DF4D48296BF532401DB1B26B2F0BBB78486FF943DB6E01BFE93F7EAF8C99587A`：两轮有界 PASS，零新增 findings；本人另跑真实 PG 普通/race及目录缺配负例。Lead 实读新文件945行和报告全文，核作者51份 raw/meta、50份最终阶段镜像、四份完整历史源码快照及独立方15份 raw/meta、50份镜像；独立连接 PID、同 tx 已写入见证、实际主键23505、失败回读/NOWAIT及 fresh-ID 提交均与固定源码和实际 JSON 相符。Finalizer 失败像等于已提交 claim 的镜像，completion DELETE 已回滚；重试按真实租约推进时钟，经公开 Finalize 完成。

Lead 对同一179项执行 Manifest 原七项，全部 exit0、gofmt输出空、完整集合与字节前后相同；[本轮 Manifest 回执](evidence-s1.5/audit-sql-implementation-r01/lead/lead-manifest-audit-r01-results.json) SHA256 `B6570C3E41B639BB36A2366182B9CBDA2D0B99FFC439264663B9C7C968067931`。默认全仓/race不含 integration，本专项 race 以两方带 integration tag 的精确命令为证；不称 full-profile 或真实 E2E PASS。作者 `%w→%v` 负控保留真实23505，四叶均在返回错误链断言处实际 FAIL，正常源码其后重新 GREEN；缺 DSN 负例为预期环境 FAIL、零 schema，不记产品 RED。Lead 初版核对器对 gofmt写入/历史meta字段及 Windows默认GBK的假设已修正，校正过程留在核对结果中，未归为产品失败。

作者16个、独立方9个 schema 的创建/阶段记录和所有成功、失败材料均保留；资源按代码断言和退出日志关闭，双方 child 已结束。未另做全库schema盘点或全机进程普查。364份作者/Lead材料见[实施证据索引](evidence-s1.5/audit-sql-implementation-r01/index.json)，SHA256 `4B93DD5351EDF960C347502148E234521A103932A911E7B8FB844860328496BF`；85份独立评审/Lead材料见[评审证据索引](evidence-s1.5/audit-sql-review-r01/index.json)，SHA256 `A4CF2B928AFBF9DF6F9466578A4D1824CA3EBEBEA7BBE1C301CE748E17A7A168`。源码副本追加 `.txt`，历史版本及 overlay 有完整映射，不引入第二份生产实现。

最终回填亦已闭环：Ptolemy 仅更新原S1.5卡（+8/-6）及唯一B10（+4/-3）并停写；Tasks最终 SHA256 `53AE60BC1FD05B8ED0DBEE3E4BF0439E4C39A1F20626E172D985238797A6804F`，Plan最终 SHA256 `C7D75A021E5C0677B93B456A6DA72083D3E1367DCB88CA4079631BCB92B8A0CA`。Lead于 `2026-10-28T05:25:59+08:00` 实读实际差异并核最终原179输入集合/字节相同，其余58卡相对本轮初始及本次写前均逐字节一致；原十二requires、完整关闭条件、BOM/LF与CRLF保持，统计仍59/44closed/4active/11todo，S1.5及S8.2仍进行中。分支/HEAD未变，暂存为空；无commit/push/清理。12份最终前后像、diff与Lead现场证明见[最终交接索引](evidence-s1.5/audit-sql-final-handoff-r01/index.json)，SHA256 `8FF7DE5635EC0191875CA9950E9E067FC0A02033055EB0A74F331F3B83700065`。Planner引用的本收据 `9EF6…` 是接收时点固定前像，已保留副本；本段仅追加metadata核证，不改变上述代码/测试/Review结果。

### 本专项运行前提与复跑方式

Lead 采纳独立评审的非阻断建议：明确列出证据输出配置，保留已评审代码。除批准的 User 级 `HAN_TEST_POSTGRES_DSN`，runner 必须创建一个保留的父目录，并向 child 设置 `HAN_AUDIT_EVIDENCE_DIR`；fixture 在其中创建随机 schema 子目录。r19裸命令省略该输出变量会在 CREATE/log后、迁移前 FAIL，不能描述成“仅注入DSN即可运行”。Erdos已实际验证这个失败路径并保留对应 schema，不是 SQL 产品回归。

以下是在仓库根、独立 PowerShell 子进程中的完整运行方式；只读既有 User 级测试 DSN，目录和数据库现场全部保留，不打印连接值。此处是复跑说明，不表示 Lead 再次执行了 PG 测试。

```powershell
$taskDsn = [Environment]::GetEnvironmentVariable('HAN_TEST_POSTGRES_DSN', 'User')
if ([string]::IsNullOrWhiteSpace($taskDsn)) { throw 'Approved test DSN is absent' }
$taskEvidenceDir = Join-Path ([IO.Path]::GetTempPath()) ('han-audit-rerun-' + [guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($taskEvidenceDir) | Out-Null
$env:HAN_TEST_POSTGRES_DSN = $taskDsn
$env:HAN_AUDIT_EVIDENCE_DIR = $taskEvidenceDir
go test -tags=integration ./test/integration/article -run '^TestArticleAuditInsertRollbackContract$' -count=1 -timeout=8m -json
if ($LASTEXITCODE -ne 0) { throw 'Audit normal test failed; retain evidence' }
go test -race -tags=integration ./test/integration/article -run '^TestArticleAuditInsertRollbackContract$' -count=1 -timeout=8m -json
if ($LASTEXITCODE -ne 0) { throw 'Audit race test failed; retain evidence' }
go vet -tags=integration ./test/integration/article
if ($LASTEXITCODE -ne 0) { throw 'Audit tagged vet failed; retain evidence' }
Write-Output ('retained_evidence_dir=' + $taskEvidenceDir)
```

本轮未改变生产/API/身份/root、原四contract或实际应用权限。A-CON-03 的 SQL 失败子集已交付，**应用身份有效 UPDATE/DELETE ACL 仍未验、载体 unknown**；测试身份postgres与trigger拒绝不能替代。原十二 requires、S8.2/Case18、真实恢复/旧备份、完整生产入口与开流等门禁保留，未关闭 S1.5、未代签人的 accepted/checkpoint。下节“结果待回交”是实际 START 时点，已由本节固定结果接续。

## r19 真实审计 SQL 子集：实际 START 历史记录

Lead 根据既有“待确认项采用建议值并继续”授权，采用第五个独立 integration 测试载体 `test/integration/article/audit_insert_rollback_test.go`，避免默认 contract 测试依赖数据库。原四文件冻结。Ohm 的 [r19 §4.7.2](../../../.add/project/knowledge/design-workspace/article-management/test-plan.md:574) 固定 SHA256 `88296F772A19600A237CCA9F47A16D4B673828EF41E5C1D0C2D88924BE2C6F66`；Pauli 的[独立规划评审 R01](review-test-plan-r19-audit-sql-r01.md) SHA256 `A89FF3A126980C7DC1CB85F9468084ACAD716FE47F33F9C80619DE0278013F1B` 两轮有界 PASS、零 findings，仅是规划结论。

Ptolemy 已将唯一新路径及 `TestArticleAuditInsertRollbackContract` 下 Mark、Restore、Transfer、Finalizer 四个必要场景登记回原 S1.5 与唯一 B10。Tasks 当前 SHA256 `76CB9A763792D862F4ADB3B21C85493B0351C8F5BEEF4A594109CD04F35B7B0D`，Plan 当前 SHA256 `C0F34ECA6FD50DF6B310296FA402C03F6790495C0D4B346874C4F3F0B86BC854`；Lead 实读两份实际 diff 并独立逐字节核其余58卡不变，59/44/4/11、原十二 requires、完整关闭条件及编码保留，原178项源码集合及字节一致。Dalton 的 G0 已确认四场景可在此一文件内实施，无新生产 API 或额外路径。

Lead 于 `2026-10-28T04:46:11+08:00` 启动核对后，向原 Dalton（`01a17dcf-f2ed-7dc2-b7f3-6c72cf771448`）实发 **CODE WRITE + LOCALPG TEST START**，submission `01a17f1d-66db-7150-954b-90eb15de6ed1`。授权仅该新文件、精确 top 的普通/race/tagged vet、现有 User 级测试 DSN 安全子进程注入，以及 Temp 完整179输入/Go overlay 的 `%w → %v` 审计错误链负控；后者仅验证错误链断言，不充作事务负控。执行时必须先核主连接和所有 fallback、实际服务器均为授权 loopback:5433 / PostgreSQL 180006；每例 schema 独立提交并立即留证后再另事务迁移，成功失败均保留。Lead 先前只读连接及配置核对成功、未执行 DDL，属于环境时点事实，不是专项测试 PASS。

真实 SQL 验收要求同事务观察业务写已发生，再由真实 `audit.Store.Append` 产生主键 `23505`；由不同 backend 连接核完整回滚镜像、NOWAIT 锁释放及 fresh-ID 重试提交。Finalizer 的 claim 独立提交可保留 owner/lease，审计失败只回滚 completion；合法重试按回读租约推进时钟，不手改数据库租约。合成 Journal 接缝不证明真实文件持久性。Erdos（`01a17da1-6bd9-7d43-8642-b8b502127296`）先只读准备，正式评审和独立 PG 复测须待作者固定代码并停写后另发窗口。

本段记录实际启动，不填新代码、SQL、普通/race或独立 CodeReview PASS。原四组件的286节点/236 leaf仍按其固定旧结果消费。应用角色 ACL 载体 unknown、真实恢复/旧备份、Case18/S8.2 和完整 S1.5 关闭条件继续保留；未接管另一会话的身份/架构/root，未授权清理、role/GRANT、新数据库或其他 secret。准备前像与预检见 [10份准备证据索引](evidence-s1.5/audit-sql-pre-start-r01/index.json)，SHA256 `79AD4F12590ADF5C907B157632EF878FECA1F1167A2937D08AC5C14EBF6E3ED1`；规划评审、metadata/G0及实际启动见 [20份交接证据索引](evidence-s1.5/audit-sql-start-sync-r01/index.json)，SHA256 `AC36C78D4647E17B0D4E81260D622AE6D594C3025B4D8B31F056D06789DC2EDA`。

## 本轮范围与准备证据

仅准备原四个 `test/contract/article/` 新增文件：`rest_contract_test.go`、`internal_ports_test.go`、`projection_contract_test.go`、`replay_status_producer_test.go`。组件子集可先行；完整十二项 requires 保留，S8.2 仍未关闭。真实 16 endpoint 注册及安全链、实际应用角色 ACL、真实审计 INSERT 失败、producer 持久化/source、全局恢复与旧备份等义务按原卡分层保留。

| 实际产出 | 执行者与固定证据 | 当前结论 |
|---|---|---|
| 测试规划 r18 | Ohm，`01a17c83-60f4-7803-b0cc-9b19c0b81fa3`；[test-plan §4.7.1](../../../.add/project/knowledge/design-workspace/article-management/test-plan.md:470)，SHA256 `BCBCFDD8CD1747540726A46C35D565FF1CC211AAB43F6059145474568841DE42` | 四 top、16 endpoint 行为 leaf、内部接缝与真实层边界；后续非作者规划评审有界 PASS，新增测试结果待回交 |
| 准备认领与 B10 对账 | Ptolemy，`01a17ca6-756c-7de1-b524-0665af10739f`；Tasks `28576192AF670683D266909AC57B218B392A703BC8AADDCBAD8395762612601C`，Plan `EC832DA255FBA89526E242E043BEDC479C54E747211793E7A76A5F29F2DE3E5D` | 仅 metadata；登记时刻 `2026-10-28T02:40:53+08:00`，非代码 START 或 B10 全批 START |
| 实施 G0 与 r18 刷新 | Dalton，`01a17dcf-f2ed-7dc2-b7f3-6c72cf771448`；个人 Temp 的 `g0-readset-baseline.json`、`r18-g0-binding.json`、`r18-coverage-prep.md` | 四文件可行，无需扩路径或真实资源；ready/stopped，待代码 START |

Lead 已读取 r18 新段和实际 diff、Planner 两份实际 diff、Dalton G0 与逐组可行性。r17 前像与 Lead 基线相同；r18 只增加当前 S1.5 规划及相关元数据，不改原 Admin2 Case/层映射。Tasks 59 张卡中仅 S1.5 改动，其余 58 卡字节相同；三态为 44 completed / 4 active / 11 todo，十二 requires 与完整关闭条件不变。Tasks 的 BOM/LF、Plan 的 CRLF 保留。174 项既有 Go/SQL/JSON/module/迁移说明输入核对无漂移；这不证明没有外部 writer。

准备原始证据分别保存在本机 Temp 的 `han-s15-lead-8swt43b0`、`han-s15-testplan-r18-e9baed6003ee445f8d91268b52f58db8`、`han-s15-dalton-prep-eb08cc7d8d084642aa10ce0c37f26ffa`、`han-s15-claim-apply-4974919ef8344dd28ea60a1c4fa0a7d8`。其中18份必要前像、实际 diff、G0、启动核对和验证脚本已逐字节复制到 [pre-start-r01 证据索引](evidence-s1.5/pre-start-r01/index.json)，索引记录原路径、角色来源和 SHA256。可执行源码副本追加 `.txt`，避免作为测试输入被工具扫描；这些准备证据不产生运行 PASS。

## 恢复接线的后续设计输入

Pascal（`01a17e3d-5550-7f91-bb7e-6e6fad3d082c`）完成[恢复运行时咨询 r02](architecture-recovery-runtime-options-r02.md)，SHA256 `46D600653553C0DC3AC39007C3A713D98D8105FA134867A40CE43C349BE44412`。Lead 采用其推荐作为后续正式设计输入：同一 root 下 normal/maintenance 互斥，公共拒绝与受控恢复访问面分开，先补真实 503/零文章下游及启动、worker 屏障；完整开流继续依赖真实 CAS、同一恢复权威和 S16 最终核验。咨询不是正式设计修订、实施、独立评审或运行 PASS。

Lead 局部核对了启动 bootstrap、登录 mode 消费者、Coordinator 的 succeeded 重验、Provider.ReadStatus 及 Journal.loadRecords：现有 maintenance 配置不构成全局业务屏障；Journal 读取可能涉及 pending 收敛；正常合法业务改变当前 evidence 后的重启准入还需正式合同。未运行或证明攻击、数据泄露、真实恢复或开流。身份设计、架构及共享入口未在本轮被本会话修改。

## 实施与验证门禁

准备记录时点的四文件均不存在、Go/PG/SMTP/服务验证未运行。Pauli 后续完成两轮[独立 TestPlanReview R01](review-test-plan-r18-s1.5-r01.md)，最终 SHA256 `85A9E111677DAC31E4A99A31D07F1FB8CFBB3E263CF1574196BFF9EA30D50A9E`，对固定 r18 增量给出有界 PASS、无新增 finding；不是代码或业务运行 PASS。报告后续只把 Lead 提供的事实由误称“用户交接”更正为实际来源，未重新评审或改变结论。

Lead 于 `2026-10-28T02:47:13+08:00` 重核 main/HEAD、空 staged、174 项输入的完整集合及哈希、四路径缺失和固定 r18，随后向唯一 Dalton 发出 **CODE WRITE + LOCAL TEST START**（Host submission `01a17eb0-1fd2-7293-bc61-8b581fe05510`）。只允许原四文件组件测试、局部普通/race/tagged vet 及个人 Temp 的可编译语义负控；不授生产/身份/main/fixture/module/额外文件写权，不连接 PG/SMTP/服务。原始启动核对见 Lead Temp `lead-pre-code-start.json`。先前“待 Review/START”文字是准备时点，不能当作当前停止信号。

## 四文件组件交付 r02 与 Lead 初步核证

Dalton 于 `2026-10-28T03:23:41+08:00` 回交冻结四文件、无未结束测试进程并明确 stopped；[停止回执](evidence-s1.5/implementation-r02/author/stopped.json) SHA256 `B66D4622EB0FE74EA4B537130E25DFA5710D7BCE9E0C68747DD31E6950D91F83`。原174项不变，新增四文件后固定输入集合为178项。

| 新增文件 | 冻结 SHA256 |
|---|---|
| `test/contract/article/rest_contract_test.go` | `1386903086B0399C39CE33BCBDF3D802B72105AE51E4CE6D58374A924CD433D7` |
| `test/contract/article/internal_ports_test.go` | `DC6791D4793ABE67FE17B83736FC77E55C347859B13A4FB5071E6AC51BF64F25` |
| `test/contract/article/projection_contract_test.go` | `D7585EE66538EECA4364BF01CC2BE3A4AC69D98C9818560D6533457E8A572033` |
| `test/contract/article/replay_status_producer_test.go` | `B519DDE178CB3F4357560681490FD00639549E8CA0B369E2EFB01785488D8FDC` |

Lead 实际读完四文件全文及最后两文件补强 diff：补齐 ID/目标/非默认分页参数、parent context/response Trace、19条独立错误消息及 field-error code；UUID 大小写样例含真实 a–f，并断言 typed 参数值/顺序和结果。预检是 Lead 检查，不冒充独立 CodeReview。早期预期/fixture、脚本及引用错误保留在[失败链](evidence-s1.5/implementation-r02/author/failure-chain.md)，不是生产回归 RED。

最终全包普通/race与精确四 top 普通/race各为 **286 run/pass、236 leaf、0 fail/skip**；四 top、23风险组、16个 Handler 端点分支完整命中。分 top 的节点/叶为 REST 182/149、Ports 62/53、Projection 15/11、Producer 27/23。这些是测试树计数，不等于286个业务验收或生产16路由已验。新测试通过真实 Handler、Policy、Reader/Command/Dispatcher 和 Provider 编码路径；SQL/Store 等端口替身只证明对应组件行为。

Lead 解析实际 JSON、核最终八项局部命令的178项 before/after、核所有映射叶确实存在、178份固定源码副本与原174基线、核46次历史/最终运行共138份 raw/meta哈希；没有把历史FAIL改成GREEN。两项完整156-Go输入 overlay只改真实生产副本：作者投影增加version导致两个叶在编译后FAIL，ReadStatus错误返回Unavailable导致Ready20叶在编译后FAIL；四文件断言不变，随后完整正常源码普通/race均GREEN。实际运行脚本按环境变量键先筛除数据库测试DSN再读取其余值；Lead初版核对器仅识别pop而未识别该comprehension，保留其受限结果及源码，读真实逻辑后的r02核对通过。详情见 [Lead证据核对 r02](evidence-s1.5/implementation-r02/lead/lead-author-evidence-audit-r02.json)。

Lead 对精确同一178项输入运行 Manifest 原七项：`gofmt -l .`（空）、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`，全部exit0，前后哈希和文件集合不变。[Manifest回执](evidence-s1.5/implementation-r02/lead/lead-manifest-r01-results.json) SHA256 `0DC59DEBEDC538D4853F95F513C268A4A68BC10E12A55BFA691531EE1B8BEAAD`。这些检查在作者最后文档回交期间对已固定的同一输入运行，随后核其停止回执四哈希完全一致；不把该读取检查当作共享root写窗口已释放。Manifest的普通全仓race不带contract标签，新增四文件的race证据取上述带标签局部运行，二者不互相替代。目标四top零skip不外推全仓平台用例零skip；goimports/golangci-lint未运行，不给full-profile PASS。

上述371份必要副本已固化到 [implementation-r02索引](evidence-s1.5/implementation-r02/index.json)，SHA256 `67BF9A2AA31D17F0C1B7E585DE95630C82887471F54907D7E398F2E5593ABFA8`，约10.9MB。原始绝对路径、角色来源和各副本哈希保留；可执行源码均为 `.txt`。相对 overlay映射用于在相同源码上重建路径，固化操作没有重跑Go、改断言或清理证据。

Erdos（`01a17da1-6bd9-7d43-8642-b8b502127296`）完成两轮[独立代码评审 R01](review-code-s1.5-components-r01.md)，SHA256 `D88CCCC9B6448E154266B00E480E2D998F763E64D87E8660463D2A2795A01938`，结论 **CHANGES REQUIRED**，仅 `AM-S15-COMP-R01-F001 / MEDIUM OPEN`。其个人普通/race同为286/236、0fail/skip、tagged vet通过；实际固定生产上限仍为262144，缺陷是新增wire测试的N/N+1取自待测生产常量，测试预期可随实现一起漂移。

Reviewer在Temp只把生产上限改为262145，原286节点仍GREEN；另以独立固定262145-byte见证证明原生产Create/Replace均400/零下游，mutant却分别201/200且各调用下游一次。该对照是真实编译后行为，定位测试检错盲区，不声称当前生产已经超限放行。R01原始记录与见证/overlay保留在 [review-r01索引](evidence-s1.5/review-r01/index.json)。

Lead决定本轮修复并已恢复原Dalton，仅释放 `rest_contract_test.go` 的修复/局部测试窗口（submission `01a17ee6-3878-7291-b2e9-7db1a3be2fc9`）：测试独立冻结262144合同值，保留Create/Replace实际N/N+1与零调用断言，同一上限漂移须令真实leaf FAIL，原源码完整普通/race及tagged vet GREEN。其他177项受保护，不改生产上限/上游合同、不落Reviewer第五虚拟文件。新修订后由原Erdos复审，当前不关闭finding、不接收组件或整卡；原r02源码/运行/失败链完整保留，Planner候选暂不应用。

## F001 修复 r03 与独立闭环

原Dalton于 `2026-10-28T03:52:59+08:00` 回交[修复停止/G0回执](evidence-s1.5/fix-r03/author/stopped.json)，SHA256 `B84A122C9B3C74A2D0B90FE70F4ED9F68A851E00321984ED740B13281BEC62F3`。REST文件新SHA256为 `CEBA105C2CBF70B91CB5DFF93D9D4FC37077153A7EDB4F9C2806C29E650E9533`；仅增加测试内独立262144常量并替换N/N+1两处来源，实际3增/2删，其余177项不变、总集合178。Lead已读[实际修复diff](evidence-s1.5/fix-r03/author/rest-only.patch)并核全部84份停止回执声明的artifact哈希，未出现第五测试文件或生产修改。

Lead重新解析r03全包普通/race：仍各4top、23风险组、16 Handler分支、286 run/pass、236 leaf、0fail/skip；创建与替换N/N+1四leaf实际通过。作者在新178项完整快照上重绑三项语义负控，四份断言不变：原作者投影泄漏与producer错误variant继续真实FAIL；新增wire漂移只把Temp生产上限改为262145，Wire组24节点/20leaf中仅Create/Replace的Nplus1两leaf失败，分别观测HTTP201/200，而两个N仍PASS。原源码随后全四top普通/race GREEN；不以父层常量比较提前失败冒充真实入口检错。

Lead对同一r03新输入重新完成Manifest七项，全部exit0/gofmt空，178项文件集合与哈希前后不变。[r03 Manifest回执](evidence-s1.5/fix-r03/lead/lead-manifest-r03-results.json) SHA256 `8904DA4630D4441B7EBE06C15A86FD24C49190CBE46AFCC3CFE7B06F37E61047`；r02的0DC回执只对应旧修订。Lead核19次r03运行的57份raw/meta、最终八个局部命令的178项before/after、完整178源码副本及三条156-Go overlay实际变更/失败叶/断言未变，未验项仍按原边界保留，见[修复证据核对](evidence-s1.5/fix-r03/lead/lead-fix-r03-evidence-audit.json)。

302份必要修复/运行/源码/负控/Lead检查副本已固化至 [fix-r03索引](evidence-s1.5/fix-r03/index.json)，SHA256 `6F0A6D5D86CF10B7A384DC1621086D1F4424050F54E1ACAC6AD0834D649F856A`，约6.9MB。原r02、独立R01假绿对照、操作失败及所有Temp材料保留；没有清理、部署、PG/SMTP或凭据操作。

原Erdos完成[独立复审R02](review-code-s1.5-components-r02.md)，SHA256 `52298FEDD5F13E9A9017334C1FEA72A1BD72C9A9658A2384660D9075D6F59933`：**F001 CLOSED、有界CodeReviewGate PASS、无新增finding**。Reviewer独立普通/race同为286/236、23风险组/16 Handler分支、0fail/skip，tagged vet通过；其独立上限漂移负控也由真实两个Nplus1叶检出，两个N仍pass。实际命中与178项before/after由Lead再次解析/核验，全部12份个人raw/meta哈希一致，不只转述Reviewer摘要。

27份复审与Lead核证副本已固化至 [review-r02索引](evidence-s1.5/review-r02/index.json)，SHA256 `486764B054A92AB3A6B39F23FA39C89EB66B2922539315314EEDB7341D619AD5`。其中[Lead现场接收核对](evidence-s1.5/review-r02/lead/lead-final-code-acceptance.json)记录 `2026-10-28T04:06:44+08:00` 的main/固定HEAD、空staged、178项实际集合/哈希一致及目录恰四Go文件；[个人原始结果核对](evidence-s1.5/review-r02/lead/lead-review-r02-evidence-check.json)保留两个真实Nplus1失败/N通过的独立复算。

Lead据最终实际diff、作者与独立Reviewer结果、三项语义负控、新修订Manifest七项及现场范围核对，接收本轮四文件组件子集。作者/Reviewer均明确stopped并关闭Agent，写窗口回Lead，后续修改须重新协调；不将Agent停止当成全局共享root writer已经澄清。原Task和B10仅回填局部证据/未验层，状态保持[-]，不新增任务、删除requires、标B10全批PASS或代签人的accepted/checkpoint；最终metadata差异另保留交接回执。

## 当前未验与接收边界

真实 Audit INSERT 本身失败、实际应用身份 UPDATE/DELETE ACL 两个候选 top未实施/未运行、无skip占位；角色载体unknown。受控ReadCurrent不证明真实Publish/ReadCurrent/source，真Journal/旧备份及全局恢复隔离/成功后开流仍待原责任。S14/S15没有新增组件测试，Case/leaf映射如实注明；Publish/Transfer旧root raw未在本片逐项消费，不据16 Handler分支宣布全生产root通过。已有S4.2自审和S2.2/S8.2等旧证据只按原版本及实际层消费，不重新签发旧PASS。完整十二requires（尤其未闭S8.2）、Case18两项上游阻塞、完整S1.5及人的成果确认均不由本轮局部GREEN取得满足。状态只以原Tasks为准。

## 最终 Tasks/B10 回填核证

Ptolemy已完成原S1.5卡与Plan中B10对应部分的最终局部交接回填并stopped。Tasks最终SHA256 `3B1EE1F9BD6A89A3F9B0A82C666A6E6251EB423E14D352A7A5C137292F75B373`；Plan最终 `DE377B618B1FF3E31C9CBF1F778ACED31B275CB7605DB9BB338EAA04A6C787A1`。Lead实读两份actual diff，独立按本轮初始前像逐卡核对：其他58卡（含35身份卡）字节相同，原十二requires与完整关闭标准原文相同，S1.5仍[-]，统计59/44 completed/4 active/11 todo，Tasks BOM/LF、Plan CRLF保持。

最终前后像、差异、写入/读回回执及Lead现场核对16份副本见 [final-handoff-r01索引](evidence-s1.5/final-handoff-r01/index.json)，SHA256 `A275BCDD90072F55883845BC314172BD0EACE36342D1A9C93B733D115CD00CA9`。Lead于 `2026-10-28T04:18:03+08:00` 再核178项源码集合/哈希等于r03、main/固定HEAD、空staged；metadata回填后没有源码漂移。所有本会话受派Agent均已stopped/关闭，保留全部证据，未提交、发布或清理材料。本轮组件交付已交回，完整任务及恢复主线仍按上述未验/阻塞范围推进。

## S1.5 子门禁现状矩阵（2026-10-31 汇总，只读描述）

本节由 Lead 依据各证据目录实算哈希汇总，只描述现状，不改变原卡 `[-]`、十二 requires 与完整关闭条件，也不代替人的接收或 ADR-0028 下的代理接收登记。

| # | 子门禁 | 现状 | 证据 |
| --- | --- | --- | --- |
| 1 | 四 contract top（REST / 内部端口 / 投影 / producer） | 普通与 `-race` 各 286 run / 287 pass / 0 fail / 0 skip，`go vet -tags=contract` exit 0 | [index.json](evidence-s1.5/contract-suite-r01/index.json) `5477B1F56DDCFAB5F8B3787902B772C2B3F62A8A84B25F58C49705DA03F8AC93` / 5418 B；四文件现役哈希 rest `CEBA105C…`、internal `DC6791D4…`、projection `D7585EE6…`、producer `6EB4A677…` |
| 2 | 16 端点真实业务 API root（method/path、Allow、404/405、平台层安全链边界） | 独立非作者评审 PASS WITH MINOR（Lorentz / add-reviewer，actualID `01a19267-1404-7050-b09d-56e91fd57ddd`）：HIGH 0 / MEDIUM 0 / LOW 8，8 条已处置（`S15-RE-F008` 于 2026-10-31 修复，见第 8 行） | [index.json](evidence-s1.5/root-16-endpoint-r01/index.json) `A4155C64E277B56C40D3A5E0E85AD11B31711D59DDCA666063DA6038E2B8330F` / 24714 B（7 runs / 10 artifacts；2026-11-05 按整卡复审 F003 为 `16-endpoint-map.md`、`lead-verification.md` 追加 F008 前向指针后字节与哈希更新，原 `E039B1F1…` 保留为历史指针；索引内 `known_gaps` 仍记 `S15-RE-F008 state=open`，属 2026-10-31 快照时点，现役状态以 `f008-review-r02/verdict.md` 的 CLOSED 为准，未改该快照字段）、[报告](review-code-s1.5-root-endpoints-r01.md) `DA2BD8D786C13A951343C095EBF48BEA2A14973322B9F0FB53CB5ABE375D34B4` |
| 3 | 应用角色 ACL 子集（真实 PostgreSQL 18.6） | R03 独立复审 PASS WITH MINOR + Lead Gate PASS；普通与 `-race` 各 12/12 子测试 PASS | [acl-r03/index.json](evidence-s1.5/acl-r03/index.json) `8ED9FF3E9F8E0CB22691A32F1B394C40AD4A6FDFFCB9E0179525777379DDFCA5`、[lead-gate.log](evidence-s1.5/acl-r03/lead-gate.log) `D21DB1F19903FA2D25685A4ADE9C58BE1FF89C4D47E9023FB7B0C269AE608155`；该索引在 `R03-F002` 补齐后已不同于旧记录 `0ECAACE5…`，以本行为准 |
| 4 | 真实 Audit SQL 子集（r19 第五 integration 载体） | 独立评审 PASS（Erdos）、Lead 实读 945 行并接收 SQL 子集；179 项输入、Finalizer 保留 claim、Journal 仅受控匿名分支 | [implementation index](evidence-s1.5/audit-sql-implementation-r01/index.json) `4B93DD5351EDF960C347502148E234521A103932A911E7B8FB844860328496BF`、[review index](evidence-s1.5/audit-sql-review-r01/index.json) `A4CF2B928AFBF9DF6F9466578A4D1824CA3EBEBEA7BBE1C301CE748E17A7A168` |
| 5 | 模拟环境收口 r01/r02（A-CON-03 + 恢复隔离矩阵 25 路由） | PASS WITH MINOR，MEDIUM/LOW 已修；`-count=25` 复跑通过 | [index.json](evidence-s1.5/sim-closeout-r01/index.json) `3E737B8CD92A4585B48B011975DB24D4661DEDD9527625016F89CA9C38C91DDA` / 12332 B；Case18 真实双系统仍 NOT RUN |
| 6 | 四文件组件 R02 | PASS，但绑定 producer 旧哈希 `B519DDE1…`；现役 producer `6EB4A677…` 由 S4.4 评审覆盖 | [review-code-s1.5-components-r02.md](review-code-s1.5-components-r02.md) `52298FEDD5F13E9A9017334C1FEA72A1BD72C9A9658A2384660D9075D6F59933`；不得表述为 R02 覆盖现役版本 |
| 7 | producer 真实 `Publish`/`ReadCurrent`（真实 PostgreSQL 18.6，消费 recovery-runtime B13d 外部卡） | 双轮 `required-pg-normal` / `required-pg-race` 各 58 pass / 0 fail / 0 skip（exit 0，无 `-overlay`）；运行绑定 10 源与实际工作树 SHA256/字节逐项一致；门禁空 DSN 即 `Fatal`、无 skip | [index.json](evidence-s1.5/producer-real-r01/index.json) `F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B` / 19567 B、[report.md](evidence-s1.5/producer-real-r01/report.md) `1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749` / 8267 B（评审后仅改命令展示口径与 sentinel 描述，其余数字/命令引用未变；原 `A9A4ECBB…` / 8023 B 保留为历史指针）；**独立评审 R01 PASS WITH MINOR**（Halley / add-code-review，actualID `01a1972f-65d8-76a1-8912-7bf5b437d291`）：HIGH 0 / MEDIUM 0 / LOW 2，Lead 已于 2026-11-01 接收 |
| 8 | `S15-RE-F008` 修复（orphan 405 `Allow` 精确值 + 仅 trace 事件断言） | **R02 非作者独立复审判 CLOSED（2026-11-05）**：R01 曾判 NOT CLOSED（仅证据记账：叶子数、耗时、缺 `gofmt` 日志），三项均由 r02 更正包消除，无阻断 finding；本机 `-race` 未能复跑（`-race requires cgo`），改审阅保存日志；该行 2026-10-31 的「R02 CLOSED」原文无独立产物支撑，经整卡复审 Aristotle 指出、Lead 复核后撤回，现结论以 evidence-s1.5/f008-review-r02/ 为准（该行原文误记为「R02 CLOSED」，2026-11-05 由整卡复审 Aristotle 指出、Lead 复核确认后更正）：单轮亲跑 34 叶子 PASS、race 带 `-v` 后 34 叶子 + 2 顶层 PASS、整包 exit 0、`go vet` 与 `gofmt -l` 输出空，耗时逐字取自日志（1.326s / 3.027s / 39.076s）；`mainOrphanAllow` 未知 action 改 `t.Fatalf`；生产 `main.go` 零改动 | [index.json](evidence-s1.5/f008-allow-assertions-r02/index.json) `410A65ADD91FCDDD3D30E54090FC0EB64890CC73990E0F55C88B13DEAC0FCC6E` / 5564 B、[report.md](evidence-s1.5/f008-allow-assertions-r02/report.md) `400E593A7A6B8BDEEA034731375B21AEFC283FA2E74818001C5B0393D5C63681` / 5808 B（R02 后仅按 L-4 改写一句措辞，数字/命令未变）；`main_test.go` `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` / 400312 B；[复审记录](review-code-s1.5-f008-producer-r01.md) `8437743308F17FA2187F8D1F2A232030591ADC6F8D13A18BA42B92E0AAA99DDD` / 5067 B；[R02 评审 report](evidence-s1.5/f008-review-r02/report.md) `B715476E4BA12715CFD408966AE8D1BF633EBD3327E7E9EBAB90EF15500E4C8D` / 11030 B、[verdict](evidence-s1.5/f008-review-r02/verdict.md) `1749A628034497109460395082953C86867A5D37B08DFB75084065CCDC6645D7` / 4533 B（Lead 现场复算一致）；[R01 复审记录](review-code-s1.5-f008-producer-r01.md) `8437743308F17FA2187F8D1F2A232030591ADC6F8D13A18BA42B92E0AAA99DDD` / 5067 B（NOT CLOSED，原样保留追溯） |

仍未满足或 unknown（不因上表任何一行而释放）：真实 Journal / 旧备份恢复；Case18 全局恢复隔离；真实部署 E2E；人的业务接收或 ADR-0028 下记录的 AI 代理接收。（2026-11-05 更新：`article-management-S8.2` 整卡已于当日关闭 `[x]`，见 tasks.md 的 S8.2 卡与 gap-r01:20；此前该处列为未满足项，按整卡复审 `S15-WC-R01-F002` 更正。）第 7 行 producer 为外部卡双轮证据，已由独立评审 R01 判定 PASS WITH MINOR 并经 Lead 接收（见下方接收记录），但只证明该子门禁可核性，不据此关闭整卡。Case18 全局恢复隔离已由 2026-11-01 覆盖对账独立评审 R01（Avicenna / add-reviewer，actualID 01a19741-0af8-7a23-8c45-7ba24420c702）判为五分项 4×PARTIAL + 1×NOT COVERED、CHANGES REQUIRED（7 findings），S8.2 不具备进入关闭评估的条件，登记见 .add/project/knowledge/tasks/tasks.md 的 S8.2 记录（:1873）；本项 unknown 结论不变。2026-11-01 补充：已按当前 revision 补取隔离矩阵执行证据（Lead 直跑、未委派 Role，目录 [evidence-case18-matrix-current-r01](evidence-case18-matrix-current-r01/report.md) `C1521F27441352C04903F707E0613314AB5F54005854DAC70DFD19E59E290D93` / 6999 B；`closed`/`unknown` 各 25 路由 + `guard_open` 对照，普通与 `-race` 双跑均 66 pass / 0 fail / 0 skip、`DATA RACE` 0），仅证明「当前 revision 有执行证据且路由覆盖 25 条」，仍是假 runtime 与零值 `pgxpool.Pool`，未覆盖 `failed`/`expired`/`undetermined`、真实 PG/双系统、`Enter`/`Install` 开流与部署 E2E，完整恢复后生产开流仍 NOT COVERED；不改变本项 unknown 结论。

### 2026-11-01 第 7 子门禁独立评审与接收

- 评审：非作者独立评审 R01（Halley / add-code-review，actualID `01a1972f-65d8-76a1-8912-7bf5b437d291`，只读，唯一写入为评审目录），结论 **PASS WITH MINOR**，6 项断言全部 PASS，HIGH 0 / MEDIUM 0 / LOW 2。产出 [report.md](evidence-s1.5/producer-real-review-r01/report.md) `B51E2F55CF03A408DFAE891C7F4C52595FEF35A5BA357236BE21CC297D0ED134` / 14173 B、[verdict.md](evidence-s1.5/producer-real-review-r01/verdict.md) `812A56FE8B80E9789BBBF3C41DC5C168CB0ED335E6CD482DD4D6F958CB79CB6E` / 796 B、commands.log `6FB81EE8A8770D588B4BD75D32754C6D4580DB8A38F04E5931326EA85FBFD5B4` / 12824 B；Lead 已实读并复算三份哈希一致。
- 断言要点：上游 index 106 登记 / 107 实物 / 0 缺项 / 0 哈希不符；10 个运行绑定源与 `replay_evidence.go`（`7C14E74D…` / 34397 B）现值逐项一致，但 177 项保护清单只证明 B13d entry→exit 期间未变，当前工作树相对 exit 有 28 项漂移，不称全量稳定；两轮 58/0/0 由 stdout 独立重算并与 leaves/result 逐项一致、均无 `-overlay`、无 DATA RACE、进程 job 9/9 与 45/45 收口；`run.py` 无环境 DSN 回落、空 DSN 即 `Fatal`；PG 18.6 / `127.0.0.1:51934` / `han_b13d_article` 由 preflight 原始日志支撑；消费登记未放大，`not_proven` 与 `known_gaps` 完整。
- Finding 处置：`S15-PR-F002`（LOW）已在消费登记 `report.md` 改写命令展示口径（标注 shell 引用存在、argv 以 `commands.json` 为准）；`S15-PR-F001`（LOW）不改他人来源证据包，按“来源包不自包含复跑、依赖包外 PG resource 快照与未随包的 `source10-before.json`”如实登记为已知局限，消费侧 sentinel 描述已改为“包内不存在的 sentinel 路径”。来源证据目录零改动。
- 复跑：**NOT RUN**。环境存在 `HAN_TEST_POSTGRES_DSN` 且 `psql` 为 18.6，但测试 helper 会创建并保留 schema；在不创建/删除数据库、不 DROP 既有对象的前提下无法确证目标库即现役测试库，故以已保存原始产物离线复算为准。
- 边界：仅把该子门禁推进到“已评审且已接收”，不关闭 S1.5 整卡，不改 `[-]` 与十二 requires；不宣称真实 Journal/HMAC/keyring、旧备份恢复、Case18、应用角色 ACL 载体、部署 E2E、S8.2 或 production readiness；AI 代理评审，非真人签字。

现场说明（2026-10-31 更正）：本节原文把 PID 35652 记为“另一 Codex 会话仍存活”，实查有误——35652 是本会话 Codex 宿主（`codex.exe app-server`，父进程 `Code.exe` 44044，09:46:25 启动，当前 shell 为其子进程），不构成外部写者；`cmd/han-server` 下 21:45 的改动为 `recovery_main_dispatcher_integration_test.go`（内容哈希 `BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A` / 23693 B），与 Lead 已核版本一致。真实外部并发风险为 `xDesk.exe`（当前 PID 35384，19:13:13 启动，5 个子进程仍存活；是否仍在写入未逐进程确证）。因此“共享 root 写窗未确认释放”的依据不成立；`S15-RE-F008` 当时确未实施、`cmd/` 与 `internal/` 零改动的事实不变。

## 2026-11-03 Case18 当前 revision 执行补证（Lead 直跑，非关闭）

Lead 直接执行（未委派 Role，本轮无 Role 产出），证据包 [evidence-s8.2/case18-current-r01](evidence-s8.2/case18-current-r01/report.md)（`report.md` `C54E4EAB1AF5761F2F250CE51A1E0B4EAD63A22AC51B02AC60E99D20F46EEC3C` / 7384 B、`index.json` `609CFBF075078808E1483B9F20E5EDBF5395AB426652EB756054CEF4E36DFCD2` / 6640 B）。

- 源码与 Case18 覆盖对账 R01 基线一致：`main.go` `6D5373CA…` / 173317 B、`main_test.go` `3720652E…` / 400312 B；八文件执行前后一致。
- 已实际执行：S8.2 root/装配普通与 `-race` 各 93 PASS / 0 fail / 0 skip；`internal/article` U/C 普通与 `-race` 各 135 PASS / 0 fail；`TestAdminQueryPostgres` 真实 PG 99 PASS / 0 fail / 0 skip（保留 schema `han_admin2_1cc188e4ed204226825e9420363a9bbb`）；真实 `SessionStore` SQL/会话撤销 6 PASS / 0 fail / 0 skip（保留 schema `han_s33_12bc6017e1f14506a10675dc82999702`）；`go build ./...` 与定向 `go vet` exit 0。
- 后果：`C18-COV-F001` 的“当前 revision 未执行/旧哈希”事实层消除；`C18-COV-F004` 的 Session 侧消除，AccessGeneral 侧仍未覆盖。
- 不变量：`C18-COV-F002`（假 runtime、无真实 PG/双系统）、`C18-COV-F003`（`Enter`/`Install` 仍 `ErrAdmissionClosed`）、`C18-COV-F005`（时点快照）均不变；本卡 unknown 清单中的真实 Journal/HMAC/keyring、旧备份恢复、Case18 全局恢复隔离、S8.2 整卡、真实部署 E2E、人的接收六项均不变。
- 边界：不改任何 finding 状态（须非作者独立复审后判定），不证明部署 E2E/真实 SMTP/双系统/完整恢复后开流/容量/人的接收；未做 Git 写、部署、删除或 DB 清理；AI 代理执行与登记，非真人签字。

## 2026-11-03 Case18 当前 revision 执行补证的独立评审与登记补正

- 评审：非作者独立评审（Anscombe / add-reviewer，actualID `01a1a0b4-231d-7730-b702-a0aaad930a07`，只读，唯一写入为评审目录），结论 **CHANGES REQUIRED**。产出 [report.md](evidence-s8.2/case18-current-r01-review-r01/report.md) `AAFFA18532814397036C5EF1066345B8B7DE1E6487D8D70776D369C1FEBC192C` / 19588 B、[verdict.md](evidence-s8.2/case18-current-r01-review-r01/verdict.md) `AE2BC8D3EE4C6BF509A837C408AE0F62D095168F2EAD6731781DDF5CEE0471DD` / 3082 B、commands.log `71C1619CF44DF6CE12778351E78CBC1758C8FF015247171C61DF549A5DFE0251` / 9985 B；Lead 已实读并复算三份哈希一致。
- 逐项判定：`C18-COV-F001` **CLOSED（仅 finding 级）**；`C18-COV-F004` Session 侧已覆盖、**整体仍 open**（AccessGeneral 侧未覆盖）；`C18-COV-F002` / `C18-COV-F003` / `C18-COV-F005` 保持 open；Case18 六项关闭前置中第 3～6 项仍未满足，**不具备进入关闭评估条件**。
- 独立重跑与上节计数一致：root 普通/`-race` 各 93 PASS、UC 普通/`-race` 各 135 PASS、article integration 99 PASS、identity integration 6 PASS，全部 0 fail / 0 skip；`go build ./...` 与定向 `go vet` exit 0。
- 新增 finding 与补正：`C18-R01-REV-F001`（MEDIUM）保留 schema 少计 29（实为 28 个 `han_admin2_*` + 3 个 `han_s33_*`，Lead 已按 run 日志全量去重复核）；`C18-R01-REV-F002`（LOW）准入状态词表漏 `checking` 并混入 observation/替身词。补正 [registration-correction-r01.md](evidence-s8.2/case18-current-r01/registration-correction-r01.md) `CFDCE04CA654DA813129C9BA0929934EE23C35A7ED5989309E40258BD61EC84C` / 4604 B，只追加、不改原 `report.md` / `index.json` / 8 份 run 日志字节。
- 不变量：本节上文“仍未满足或 unknown”所列真实 Journal / 旧备份恢复、Case18 全局恢复隔离、`article-management-S8.2` 整卡、真实部署 E2E、人的业务接收（或 ADR-0028 下登记的 AI 代理接收）均不变；`C18-COV-F005` 维持 2026-11-01 的“时点快照”裁定（消费侧 `producer-real-r01/report.md` 现为 `1B2FB3CF…` / 8267 B，`index.json` 仍 `F840BC2C…` / 19567 B，未重写）。
- 边界：AI 代理评审，非真人签字；本轮未改代码、未重跑测试、未做 Git 写/部署/删除/数据库清理，未 DROP 任何保留 schema。
### 2026-11-03 复审 R02 闭环（PASS WITH MINOR）

- 复审：同一非作者评审（Anscombe / add-reviewer，actualID `01a1a0b4-231d-7730-b702-a0aaad930a07`，只读，唯一写入为复审目录），结论 **PASS WITH MINOR**。产出 [report.md](evidence-s8.2/case18-current-r01-review-r02/report.md) `C732752A552672FB432A3BA1908CF56B103A9DB1675FC46C767021D3B357B2A9` / 10370 B、[verdict.md](evidence-s8.2/case18-current-r01-review-r02/verdict.md) `BB8E3B41AFA13B0D8D6DA2666DE35920FD3D90F9F645E653B070E31FDAE8A2B7` / 2651 B、commands.log `E8009A4086B4A0E3564164B87E34C018FD02D777CBFE55456EB4A40B4A2E946A` / 6694 B；Lead 已实读并复算三份哈希一致。
- 判定：`C18-R01-REV-F001` **CLOSED**（28+3 与两份日志去重集合逐项一致，漏 0 / 多 0；“显式作废旧登记值 + 同目录只追加更正”足以关闭）；`C18-R01-REV-F002` **CLOSED**（六态常量、`recovery_guard.go:59` 仅 `public_ready` 放行、`internal_ready` 为 observation 字面量、`unknown` 来自测试替身均复核成立）；`C18-COV-F001` 据此**正式登记 CLOSED（仅 finding 级）**。
- 保持 open：`C18-COV-F004` 整体（AccessGeneral 侧未覆盖）、`C18-COV-F002`、`C18-COV-F003`、`C18-COV-F005`。
- 非阻断观察（本轮未处置，登记留痕）：补正 `:54` 的 session 字段措辞与日志字面（`created_here=true retain_policy=always`）不同但语义一致；补正 `:61` 把 Readiness 503 probe 与受保护请求的 `writeRecoveryHTTPClosed` 混写，核心结论成立；补正文件未登记进 `index.json` 的 artifacts/runs，因原索引“零字节改动”声明须保持，后续如需消费者指针应另立索引而非改写旧索引。
- 不变量：S1.5 与 S8.2 仍 `[-]`，`blocked-upstream` 不解除，Case18 不具备关闭评估条件；上文六项未满足项不变。
- 边界：复审按授权未执行任何 `go test`、未连接或写入 PostgreSQL、未 DROP 或清理 schema、未做 Git 写；`git status --porcelain` 310 行与开工一致，仅新增复审目录 3 文件。AI 代理评审，非真人签字。

## 2026-11-04 C18-COV-F005 当前消费绑定关闭登记（Lead 已接收）

- 当前消费入口为 [current r02 index](evidence-s1.5/producer-real-current-r02/index.json)（SHA256 `E7DF162F0716558369DFA996DB7FE38FAD20E0AF340B4873685538C900A54E56` / 21133 B）及 [hash-manifest](evidence-s1.5/producer-real-current-r02/hash-manifest.json)（`996EE23E23F48F4988DC583EF0C9A4F4CC85C11815BC8C608E3359EB56B901B9` / 10312 B）。旧 producer index/report 原样保留；上文 F005 OPEN 及索引中的 pending/NOT REVIEWED 是各自形成时事实，不改写历史。
- 非作者 Carver / add-reviewer，实际 Agent ID `01a1a491-533c-7123-a0cb-b0d2e955a6dc`（Lead 本轮提供），[独立 report](evidence-s1.5/producer-real-current-r02-review-r01/report.md)（`F5F8CF5B0EDC53AD89C7AC8714FB95DA634A01F55EA751DA225DC2302512C7AF` / 12090 B）与 [verdict](evidence-s1.5/producer-real-current-r02-review-r01/verdict.md)（`053C3CBD3C5F837D9C52C5F01EC7BFA7DF4126E3FC866F18E5446DC873917161` / 1782 B）判 **C18-COV-F005 CLOSED，Review gate PASS，仅当前消费绑定，0 新 finding**。
- [Lead reception](evidence-s1.5/producer-real-current-r02-lead-r01.md)（`31792BDC56E309CBD0DF614934856C7B492E99811D48CE74F4D5DF597B99CC99` / 2788 B）正式接收该 finding 级关闭，记载 Lead 已实读独立结论并现场实算 43/43 manifest、8 份旧保护证据无漂移。本条由 Planner 登记已决定结果；不把 Lead 的全范围核证或 Reviewer 的离线复算当本角色重新执行。
- 当前 report 实物 `1B2FB3CF…` / 8267 B 与旧声明 `A9A4ECBB…` / 8023 B 的界限继续保留：旧原像 unknown，不重建；旧 index `F840BC2C…` / 19567 B 不覆盖，不宣称单独消费旧 index 已自洽。两个历史普通/race 日志各 58 pass / 0 fail / 0 skip 仅为离线复算；当前 10 源 9/10 匹配，`deletion_replay_source_test.go` 漂移原因/影响/当前运行 unknown，不据此声称当前源码或 runtime PASS。
- **保持开放与阻塞**：C18-COV-F002、F003、F004 整体仍 OPEN；S1.5/S8.2 仍 `[-]`、`blocked-upstream` 不解除，Case18 仍阻塞且不具备关闭评估条件。不改两卡 checkbox/写集/requires/统计，不把 F005 关闭外推其他 finding、真实 Journal/HMAC/keyring、旧备份、完整恢复开流、部署/E2E 或 production readiness。
- 本次仅追加元数据和消费指针；AI 独立评审加 Lead 接收，非真人签字。未运行 Go/DB、未改源码/producer 旧文件/旧证据、未做 Git 写/清理/下级委派。
- 接收时点与当前版本：43/43 匹配是 Lead 接收且本条追加之前的核证事实；原 r02 manifest 的 `verification_context` 绑定本文件追加前 42,439 B / `1C7C0E45079AA2FEB284798DD6A712E3F1C5AC15104C4360761CE71F54BE95F6`。本次按明确授权只追加接收元数据，原 42,439 B 逐字节仍为当前文件前缀；不回写旧 index/manifest，不把追加后的整文件继续称为原 hash 匹配。追加前后摘要及前缀相等核证另存 [planner/f005-reception](evidence-pg-perf-r04/continuation-20261104/planner/f005-reception/)，本段不重新执行或扩大独立 Review 的覆盖范围。

- **2026-11-04 r03 current 消费接续登记（仅 C18-COV-F005；Lead 已接收）**：旧r02在本文件由42439 B追加至45600 B后当前为42/43，失配仅活verification整文件；原43/43及r02独立评审/接收仅属其旧时点，原r02记录与文件全部保留。当前入口改用 [r03 index](evidence-s1.5/producer-real-current-r03/index.json)（19180 B / `4B3F9979D749279D0EA8DDF0A0BC5A27F9DD32EF10B6E3033586445BFF3738CB`）、[manifest](evidence-s1.5/producer-real-current-r03/hash-manifest.json)（11293 B / `AB98DBCE882CC6A2C1B445A69FBD7808111878E4390042621A2297C73255D0DD`）及其固定 [snapshot](evidence-s1.5/producer-real-current-r03/verification-context.snapshot.md)（45600 B / `7997E1ADF37D82FB663B40B41BD43BE4AF191120E66A60BC7E6C1C25CA1F4AF9`）；活verification仅导航，不在新manifest硬绑定中，合法追加不改变固定snapshot。原非作者Carver / add-reviewer `01a1a491-533c-7123-a0cb-b0d2e955a6dc` 的 [report](evidence-s1.5/producer-real-current-r03-review-r01/report.md)（`C6F78C972EFC2A84BA15F058F68A2E2300563BA3B2DFB409BC2DEE84D4B2A5FD`）和 [verdict](evidence-s1.5/producer-real-current-r03-review-r01/verdict.md)（`E0C546937D593FDE13AED744F3F634BCE8E10A2A851E61E96F99A244B635572A`）判r03 delta PASS、0 finding、F005当前绑定CLOSED；[Lead receipt](evidence-s1.5/producer-real-current-r03-lead-r01.md)（2107 B / `DB670CF7E26D740362D3686DE3E764E270DE82D8F677EB57F30D26E53D7EF9DB`）记载已实读并独立核50/50、无liveverification硬绑定且正式接收。Planner仅追加该已决定结果并核追加后50/50，不代签Reviewer/Lead。F002/F003/F004整体仍OPEN，S1.5/S8.2仍active、Case18仍blocked；历史58/0/0、十源9/10及原像/漂移unknown边界保留，不改checkbox/统计/requires/写集或Plan，不改变S2.4原18 finding性能代码CHANGES REQUIRED，不外推S2.4/源码/runtime/性能/E2E PASS。AI独立评审加Lead接收，非真人签字；本次无Go/DB、源码或旧证据修改、Git写、清理及下级委派。
### 2026-11-05 整卡复审 R02 与 Lead 同 revision Gate

- 复审：非作者 AI 代理 Reviewer（agent `01a1ac16-8717-7b13-b1c2-9164f0759686`，只读，唯一写入 `evidence-s1.5/wholecard-review-r02/`），结论 **PASS FOR LEAD GATE**：上游 F001/F002/F003 均判 CLOSED；10 项关闭条件第 1–9 项 PASS、第 10 项 PASS FOR LEAD GATE；12/12 requires `[x]`。产物 [report.md](evidence-s1.5/wholecard-review-r02/report.md) `E2ED8CA0A5BDA9A2CACB6FF01AD26CBC6534B9A68C7AD1150CB894FBD4B3384C` / 20,822 B、[verdict.md](evidence-s1.5/wholecard-review-r02/verdict.md) `A5E1032935085D2053D4E86DA966288F131972402329F291EBAF765D290F3FB5` / 9,468 B；Lead 实读并复算哈希一致。
- 复审独立复跑：`go test -tags=contract ./... -count=1` exit 0（28.514s）、`go vet -tags=contract ./...` exit 0（2.218s）、`gofmt -l cmd internal test` exit 0（0.364s）；contract 普通/race 各 286 run / 287 pass / 0 fail / 0 skip（离线解析既有日志）。**NOT RUN**：`-race` 实跑、真实 PostgreSQL/ACL/Audit/producer 复跑、部署 E2E、Journal/HMAC/keyring、旧备份恢复。
- **Lead 同 revision Gate**：HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`、分支 `main` 不变；`main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173,317 B、`main_test.go` `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` / 400,312 B 与 S1.5 绑定值逐字一致，期间无源码/测试写入；`git status --porcelain` 321 项与复审前相同；12/12 requires 仍 `[x]`，统计 76 completed / 1 active / 9 todo。
- 范围：本轮写入限定在 `.spec/wip/article-management/`（F008 R02 复审、整卡 R02 复审、`verification-s1.5*.md`）与本任务记录；未 stage/commit/push，未删除文件，未触碰 `.spec/wip/identity-auth/_scratch/`。
- 保留 LOW（非阻断）：① `root-16-endpoint-r01/index.json` 的 `known_gaps` 仍记 `S15-RE-F008 state=open`，属 2026-10-31 快照字段，未改写，现役状态以 `f008-review-r02/verdict.md` 为准；② F008 R02 评审产物未内嵌 add-reviewer actualID，`tasks.md` 记录的是执行会话 agent ID `01a1ac0f-…`。
- **接收登记（2026-11-05）**：按 ADR-0028 登记 AI 代理接收。授权来源：用户 2026-10-31 项目级授权（登记于 ADR-0028）+ 本会话对「按 ADR-0028 登记 AI 代理接收并关闭 S1.5」建议的「继续」回复；代理主体：本 Codex 会话 Lead（AI 代理，非真人）；范围与不覆盖项见 [acceptance-s1.5-r01.md](acceptance-s1.5-r01.md) `67A5CF3A2FF0055BF6EF4FA4551C720EB0607EAA2F2E7A9E731E1B8550BD87DA` / 3,090 B。- **状态**：Gate 通过且接收已登记，`article-management-S1.5` 关闭（`[x]`，完成@2026-11-05）；统计 77 completed / 0 active / 9 todo。
- 声明：**AI 代理评审与 AI 代理 Gate 复核，非真人签字**；不替代人的业务接收、发布授权或 production readiness 结论。
