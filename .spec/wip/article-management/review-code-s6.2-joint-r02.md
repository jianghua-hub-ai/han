# S6.2 完整 PG + 十三文件 joint CodeReview R02

**S6.2JointCodeReviewGate：PASS（下列固定代码及本轮 PG 矩阵）。AM-S62-JR01-F001：CLOSED。新增 findings：0。** 不表示部署/E2E、Linux Journal fsync、旧纳秒 Journal 兼容或完整恢复 Gate 通过，不发布 Ready。

## 对象与两轮实际范围

objectType `code`；独立非作者 Reviewer。原作者：PG James，F001 修复及其余实现 Turing。工作区 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 已实核。十三文件 SHA256 全部匹配，三修订文件收口时再次核对：

| 文件 | SHA256 |
|---|---|
| internal/identity/article_disposition.go | EB053611CE9346A9A51C0485DD9AE1003EA99C57AF0A3842A4621D67ECE04593 |
| internal/identity/article_disposition_test.go | A92E8C4D65C76361094B472F35BEDEAB77F15C2FC5E0ECA54854BCDDC640EDF4 |
| internal/article/command_service.go | 585BB51D19B3FCD7BFDC9C84465D2BB5D24F41BD8285553FC20BC10B161B4D68 |
| internal/article/repository.go | E0C2430A981A5F90E84F197159158CC4A218A4FB399C74E90C203B19AC0FE078 |
| internal/article/deletion_finalizer.go | 546C69B7CC363453F6A4E866D4E3B1B68D9E6B611463159EDE69ACEC0D9B8750 |
| internal/article/deletion_test.go | 6D04CD35F0049E64FFEC6954B53EF0AB75256BF97C156E331BB5B019E87EA9C6 |
| internal/article/query_service.go | 9674130673CD05E8FA864B77528FDBF12C9AB8305AE6AD2EF7E2A85DB9136E7E |
| internal/article/handler.go | A0921B0178551A8EA51772ECF9823B50AA693DD3B450FDD347257B6F77CCC7E8 |
| internal/article/orphan_disposition_test.go | 301A40FED868BFAB85D80C84888399DAAAF3B22E8CBBFBD7C754FC3B6E5265DE |
| cmd/han-server/main.go | 4B9D83BA52E322E9E7A430F18A59DAA0FC651D4D4B21BEAFBA825E7E991A7569 |
| cmd/han-server/main_test.go | 1FB995EA3DE93A3F9AC2CC575D4AC86A9084592742EEC0C86F1ACF4E9CF28C9E |
| internal/article/deletion_coordinator.go | 6EEA01FB40FBD6B237B4E02600122708404EE1AF59626971D4CE40D72DB98481 |
| test/integration/identity/orphan_disposition_test.go | 7B38993E04D342F424C11C4D9E136DD3F06967D7E10A939C1103186F9B289D12 |

设计 revision `article-management-design-ir01-r16` SHA256 `CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC`；test-plan r14 SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C`，均实核。复用本会话已读 Reviewer Role、add-code-review、delivery Workflow 及适用 coding/go-service/安全/抽象/可观测性规则。

Round 1：复用①R02、②R01、③root R01 已完成的固定局部审查；完整读取新增 PG 文件当前全部 1,920 行 / 81,510 bytes，审查三文件修复增量、局部模拟事务支持和新增回归。内存逆转 finalizer 的本轮审计改动后，SHA256 精确还原 R01 的 `12F9D775F115732226908C0440D5F30D6FAF37A17CC19FC2A2CE147216F869C4`，确认生产变更止于该分支、参数及常量，未落盘逆转。

Round 2：逐组对照 test-plan §4.5.1，沿 Identity 锁/清理、Article command/native proof/SQL、coordinator、finalizer/worker、root factory/router、handler/query 的真实接缝核联合合同；必要读取现役迁移、audit.Store、canonical/harness/commit 故障 helper。独立解析完整普通/race 与红绿/Lead 叶子原始日志，核固定 hash、实际测试节点、退出结果及保留清单。没有重审无变更旧模块全文，没有扩大到全库锁图。

## 原 finding 闭环

**AM-S62-JR01-F001（原 HIGH）：CLOSED。** 原触发为可信 tombstone 重建 ordinary operation，Article absent、actor/source 均空，却固定生成 user actor 审计，真实 PG `23514 / audit_events_check` 导致 T2 回滚、无法收敛。违反 design r16 `design.md:1963` 的 absent 幂等完成及 `article.deletion.replay` 系统审计合同。

修复证据：`deletion_finalizer.go:644` 将真实 `DeletePendingArticle` 结果传入审计构造器；`:711` 仅在结果为 `PendingArticleAlreadyAbsent`、ordinary、ActorUserID/ActorRole/OrphanedFromUserID 全 nil 时改用 `ActorSystem`、`article-deletion-replay` 和 `article.deletion.replay`。结果来自实际 DELETE 行数及存在性查询，present mismatch 先拒绝；未从 actor/tombstone 猜 source。该路径仍先核可信 tombstone、operation 锁和有效 claim；审计与 completed/audit_event_id 在同一 T2 提交。absent ordinary 不产生清理 proof，未放宽 native proof/Identity 校验。正常有 actor 的普通/违规/orphan 删除继续原审计。

局部回归 `internal/article/orphan_disposition_test.go:1038` / `:1068` 覆盖恢复成功、精确系统动作/目标/trace/completed 时间绑定、重复收敛、审计和完成写入失败全 T2 回滚及 lease 后重试；`:1023` 验证用户审计不变。真实 PG `test/integration/identity/orphan_disposition_test.go:1864` 保留原失败前提，`:1897` JOIN operation/audit 核 actor/action/source/目标/trace/时间/审计 ID、唯一审计、无内容重建、零清理调用与邮箱保留；重入比较完整持久事实快照。原叶子 RED→GREEN、普通/race 全矩阵及 Lead 独立叶子均有效执行通过，没有 skip 或补造 actor。

直接消费者：`0001_shared.sql:29` 的 actor 约束满足；`repository.go:321` 的 replay 审计读取识别该动作；本例不进入 Identity 清理。此次关闭只覆盖上述已确认缺陷，不扩展旧备份恢复能力。

## 十组矩阵充分性与联合合同

以下行号均指当前 PG 文件；真实数据库断言与冻结生产源码共同构成结论。

| §4.5.1 / PG 组 | 实际检查及判断 |
|---|---|
| 01 canonical 兼容 | `:512`、`:984`、`:1109`、`:1664`：transfer-first、canonical-first 两个 UUID 排序方向、kept_deleted、source User/Identifier 竞争；真实 PID 与 barrier/pg_blocking_pids 证明交错。55P03 回滚后新调用重核目标及归属。生产锁为 sorted 全 User→全 Identifier→Article NOWAIT，无新增等待环。 |
| 02 T2 反序退避 | `:585`、`:955`：ordinary/violating/orphan Article 冲突及 orphan Identity 冲突；真实 coordinator 在 Article→operation 等待时，T2 已持 operation，Article NOWAIT 失败并释放 operation，让 coordinator 前进。断言原生55P03、pending/审计未变及已提交 lease 保留；不是 lock_timeout 制造错误。 |
| 03 HTTP / worker 消费 | `:644`、`:674`、`:1634`：真实 command/handler/HTTPAuthorizer 消费数据库冲突，503/database_unavailable、每请求仅一次调用，后续请求成功。真实 discoverer/finalizer/worker 的55P03进入 backoff（Sleep计数），未到 lease 不重试，过期后收敛且仅一次释放；现役 worker 源码分类吻合。 |
| 04 last-of-two / pending | `:694`、`:1591`：publish+transfer、publish+delete_complete 真实多事务竞争；首个提交保留另一 active/pending，受控重试最后一篇仅一次释放。第三 pending、异常 author/source 关联仍计数。生产 `repository.go:604` 一个 COUNT FILTER 语句按 author OR source 同时取两个计数，位于动作之后、同 caller tx/Read Committed，不新锁 Article/operation。 |
| 05 可信 reference | `:726`、`:749`、`:1226`、`:1496`：三动作成功对照、引用字段逐项篡改、持久 operation 字段漂移、foreign/ended tx、trace、同 tx 重入及提交后失效。固定 verifier 调用真实 native proof→Repository 当前 tx 回读；不返回合成完成事实。生产 root 捕获 native private proof、exact tx/reference/trace 并逐字段绑定，与公开 DTO 区分。 |
| 06 快照 / 零行 / 漂移 | `:766`、`:787`、`:1755`、`:1776`、`:1800`：Notice 四关联、两种 pending recipient、原锁内地址、缺失/重复/错值；真实 trigger 造成零行写入，或真实 SQL 改坏事实，观察全库相关表快照不变。已存在 mismatch 与明确 absent 分开；retained/Identifier 成对空可幂等，单边缺失/错绑定/非 deleted 拒绝；username/phone 保留。 |
| 07 原子失败 / commit unknown | `:808`、`:1192`、`:1375`、`:1417`、`:1452`：Notice/recipient/Article/audit/verifier/清理/完成各边界失败，真实 SQL rollback、T1/Journal/claim 保留、worker 后续收敛。commit wrapper 分别真实 rollback 或真实 commit 后丢 ack，重新读取事实，避免重复 Notice/释放审计；不把未知结果报成功。 |
| 08 N001 contacts / replay | `:820`、`:1340`、`:1864`：publish/delete 对合法空/非空 retained 均成功、无 Notice 依赖；transfer 要求两有效联系快照、目标 active，Gate closed 仍提交；违规分类保留。source-free absent replay 完成系统审计而不清邮箱，F001 已闭环。 |
| 09 原设计 F001 三步交错 | `:852`、`:1063`、`:1294`、`:1695`、`:1729`：ordinary/violating T1 pending→canonical 删除作者→T2 自动清理，保留原 kind/source=NULL；其他 pending 阻止过早释放、重入无重复审计。候选来自当前锁内 author，再 Identity NOWAIT 核 deleted；active 作者不清理但锁持至 commit，使随后 canonical 见零集合。冲突释放 Article/operation；失败后无需人工 canonical 重算即可收敛。 |
| 10 N002 / 新 T1 精度 | `:860`、`:1022`、`:1255`：纳秒/非UTC输入真实 PG 写读与 tombstone 微秒等值，真实 RETURNING lease 为 UTC 微秒；T2 等待 operation 锁期间推进时钟，锁后 fresh clock 拒绝过期；owner/status/lease 漂移均无副作用。`coordinator.go:166` 在新 T1 同时进入 DB/Journal 前归一化，未覆盖旧记录。`:910` 真构造 FileJournal，在本 Windows Host 返回 unsupported 并明确未验 fsync，未 skip。 |

生产接线复核：command 和 finalizer 共享 root 的同一 lifecycle adapter；adapter 没有 pool，可信 proof 回读沿 caller tx，Identity 不直接查 Article 或跨所有权写入。公开处置先 Identity 后 Article NOWAIT；finalizer operation-first，后续 Identity/Article 均 NOWAIT；ordinary/violating 从 Article 锁内 author 取候选。55P03 cause 保留至 HTTP/worker，不误包成 completed/no-op。通知与两个地址快照、状态变更、处置审计、邮箱/Identifier 清理和释放审计在一事务；空邮箱条件只约束 transfer。

冻结 root/HTTP/query 复用③完整审查并定点核接线：唯一 router 的 orphan list/detail/publish/transfer/DELETE、admin/session/Origin/CSRF、严格 body/ETag、404/405/Allow、cursor view/pending 过滤仍有效；query 为实际生产 SQL，member 查询未替换。transfer 返回 article + notification(id/delivery_capability/两 pending kinds)，不暴露地址；violation hidden_reason 保留；Gate 在 commit 后投影，不代表投递。独立必填正 bigint `HAN_ARTICLE_TRANSFER_NOTICE_MAX_ATTEMPTS` 无默认，配置/溢出边界沿冻结 root 测试与源码结论。

PG 接缝边界：文件内 `orphanPGIdentity` 是显式 DTO/native proof 映射 adapter，不是生产 root factory 本体；成功 provider、Identity lifecycle、Repository、audit.Store、canonical、worker 及回读 SQL 为真实实现。HTTP 用例的 session resolver 和 Journal 是 fixture，未启动真实 server/session/SMTP。生产 factory/router 的可信绑定与安全由冻结源码及既有组件测试另证，PG 结果不冒充部署全链。

资源核查：本文件 harness 复用完整0001～0009、提交后 schema/fixture、pool runtime 与各 tx 的 search_path，真实独立连接/PID；pool.Close、rows.Close、失败 tx rollback、context cancel、WaitGroup join 与 barrier 退出均有对应路径。故障 trigger 仅在隔离 schema，未改 production migration。没有注册 DROP；测试留存名单，与授权一致。

## 执行证据及归属

本人独立运行：`go test ./internal/article -run '^TestOrphanDispositionFinalizer_(SourceFreeReplay|SourceBranches)' -count=1 -json`，exit0，3顶层/11个 Test pass节点，0 fail/skip。本人未另跑 PG/整仓/race/Manifest，未新增 schema。

本人独立读取并解析下列仓内原始日志（`evidence-s6.2/`），核 SHA256 及实际事件，不只消费 summary：

| 日志 | SHA256 | 执行者 / 结果 |
|---|---|---|
| pg-normal-r01.jsonl | 9A6D5F2E807D2F77E25F7610990BF56A3293FEA55F957626D6C4FA23470179F8 | Lead 首轮真实 RED；唯一 source-free replay leaf失败，保留历史 |
| pg-leaf-red-r02.jsonl | 8C80118C4CD6327BD1F33EB29FB196E1D4767C842199B1034F3A61B5150D0EB9 | 作者修复前目标叶子 RED |
| pg-leaf-green-r02.jsonl | 6FCBF95E23A084CFF6518449F27DC7E87E9AC60DD7390E8EACF8443F40E51B31 | 作者修复后目标叶子 GREEN |
| pg-normal-final.jsonl | 12D4CE7F57499184E41751DB526F7DDFC7133C53373E433502A43B6A545A888C | 作者普通 PG：package133.508s，10/10组、181 Test pass节点、0fail/skip，163schemas |
| pg-race-final.jsonl | 172846CCA19D70F6DE9DDDA62CC1A32B51DD82A9F1A2C048E978E68E2E3621CE | 作者 PG race：package145.989s，10/10组、181 Test pass节点、0fail/skip，163schemas |
| pg-lead-leaf-final.jsonl | 2E357CFC5D7EFDFA8160EEED855D76C05C4CD028E228845BED5B58511E18E417 | Lead 独立 actual PG 叶子 GREEN：3 Test pass事件（含父级）、0skip，目标leaf真实运行 |

`validation-summary.json` 与原始日志一致；`retained-schemas.json` 实核328条、328个不同 schema，与作者红绿+两完整矩阵对应。两套完整日志均实际走 Windows constructor fail-closed 分支，不宣称 fsync成功。未引用 James 断线前未知执行。

Lead 另报告：新 revision Manifest 七项按序全部 PASS，session45844 exit0，且亲自解析完整普通/race原始日志及 hash。此处 Manifest 归属 Lead，本人未重跑。作者 Article普通/race各495 pass节点、两个既有 Windows inventory能力 skip及vet/gofmt PASS来自作者摘要；这两个 skip 不属于 PG 矩阵，本人不将摘要称作独立执行结果。

## 结论与保留边界

F001 原触发、数据库约束、稳定审计合同及真实消费者均完成闭环；十三文件的冻结局部评审已与完整新增 PG 审查和联合接缝核对接合，本轮无新增 finding、无必须待决定的架构/API/schema 变更。

Linux真实 Host/fsync、旧已落盘纳秒 Journal 的受控兼容、active旧备份/无来源完整恢复及恢复 readiness、真实部署/E2E继续原独立 Gate。本报告只证明本卡固定实现与列明矩阵；不把 planned test-plan 改成已执行 SSOT，不替人签字或改变 draft/任务状态。

唯一新增写入为本报告。R01 SHA256仍为 `4F53F6603E9F4411463992AED49A707AF64388BB4650F8509F6FB4F1C28D331B`，未覆盖；未改代码、其他报告、Tasks/Plan、设计、Git，未清理/再委派。完成后停止写入，交回 **F001 CLOSED / S6.2JointCodeReviewGate PASS**。
