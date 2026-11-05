# S6.2 切片②独立 CodeReview R01

**slice2CodeReviewGate：PASS（本片局部实现）。两轮审查 0 finding。** 不代表 production root/HTTP、真实 PostgreSQL 或完整 S6.2 通过。

## 固定对象与依据

objectType `code`；作者 Turing，本执行者非作者。固定工作区 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。以下工作树目标在审查开始及收尾均匹配：

| 文件 | SHA256 |
|---|---|
| `internal/article/command_service.go` | `585BB51D19B3FCD7BFDC9C84465D2BB5D24F41BD8285553FC20BC10B161B4D68` |
| `internal/article/repository.go` | `E0C2430A981A5F90E84F197159158CC4A218A4FB399C74E90C203B19AC0FE078` |
| `internal/article/deletion_finalizer.go` | `12F9D775F115732226908C0440D5F30D6FAF37A17CC19FC2A2CE147216F869C4` |
| `internal/article/deletion_test.go` | `6D04CD35F0049E64FFEC6954B53EF0AB75256BF97C156E331BB5B019E87EA9C6` |
| `internal/article/orphan_disposition_test.go` | `D6EC42338586933CB141550563A3050A1A38012F897420821C5E6BC0E855685E` |

design r16 SHA256 `CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC`、test-plan r14 的更正后 SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C` 均实核一致，没有忽略文档漂移。切片① R02 冻结代码双哈希仍为 `EB053611CE9346A9A51C0485DD9AE1003EA99C57AF0A3842A4621D67ECE04593` / `A92E8C4D65C76361094B472F35BEDEAB77F15C2FC5E0ECA54854BCDDC640EDF4`。

复用本会话已完整读取的 Reviewer Role、add-code-review、当前 Workflow、coding/go-service、评审清单、抽象原则、审计与安全规则。唯一写入为本报告；未修复、改 Tasks/Plan/design、Git 写入、清理或再委派。

## 两轮实际检查范围

Round 1：通过内容比较指定临时根下四份 before-image 与当前目标，检查全部变更段及必要调用上下文；新 `orphan_disposition_test.go` 全文 781 行审查。未把未跟踪文件的空 git diff 当作无变更。

before-image 根：`C:\Users\J00010~1\AppData\Local\Temp\han-s62-4d7a5257f49745bc82ad754d1798d845`，四文件基线 SHA256 依上述表顺序分别为 `7162FA880F56E95814C46347A75041B81FFDFBA00231E863F7D02F2317D8A2E2`、`22551B3B96888A2361A2A55D1C6C650A64494D2198886F85B34BBA8090C3F14C`、`E914B290DBCB59D24116ECBBD4592AD171038FCEA25081739AF4A6CFB17FD12C`、`91DE34174E2F0E8EA07257CD2EE98605E383DCCC469A5853552A961DAF0015AB`。

Round 2：沿切片① Identity helper、既有 canonical/关联计数合同、coordinator 的 Article→operation、finalizer claim/T2、worker 的 Converge/分类/退避链检查兼容性；对照真实 migration 列名、audit.Store 及 UUID/状态辅助函数。范围止于直接消费者；未审未开始的 root/HTTP 或全库锁图。

## 主要结论与证据

| 检查点 | 有界结论 |
|---|---|
| publish/transfer 锁与资格 | `command_service.go:982`～`:1050` 先授权，开启 tx 并设 READ COMMITTED，无锁定位 source 后取 Identity，再显式 Article NOWAIT；锁内核 source/author/held/active/version，不能换来源或刷新 ETag 续做。`:1053` 保留 violating 不可直接公开和版本溢出门禁。Identity 的 User→Identifier 排序/NOWAIT 复用切片①。 |
| 转移原子性 | `:1094`～`:1162` 在同一 tx 写 Notice、双 pending recipient、author/published/source/version 与处置审计，再核事实及调用 Identity；violation/category 保留。`repository.go:574` 的条件 UPDATE 只改批准字段并要求一行。回读核 Notice 四关联、双 kind/notice/status/原锁内地址；零行、错快照/关联等不以“执行过 INSERT”冒充成功。无 SMTP/投递调用。 |
| 私有 native proof | `command_service.go:924` 的字段不可由其他包填充；创建只在真实锁内工作流。`:935` 对引用指针深拷贝；`:948` 绑定 exact tx/trace，直接通过 Repository 在该 tx 回读 Article、Audit、Notice/recipient 或 operation。`:1210`～`:1309` 再核前后像、动作、审计、来源、版本及 tombstone 绑定。外部 public projection/空 proof、异 tx 或已结束 tx 不能取得完成权威。 |
| 同一快照计数 | `repository.go:604` 的单条聚合语句同时统计 active/pending，谓词完整包括 author 或 source；无业务可见性过滤、无额外 Article/operation 锁。两条工作流均在事务第一条业务查询前设 READ COMMITTED，计数发生在动作写入和来源锁之后。 |
| claim 权威 lease | `deletion_finalizer.go:776` 使用 UPDATE RETURNING 读数据库实际 operation/lease，`:794` 规范为 UTC 微秒；T2 首锁 operation 后才于 `:519` 取新鲜时钟，核 status、owner、claim lease 相等及未到期。没有把旧 Go 纳秒候选当数据库权威值。 |
| orphan/已删作者 pending | `deletion_finalizer.go:532`～`:612`：orphan 按 operation→Identity NOWAIT→Article NOWAIT；ordinary/violating 按 operation→Article NOWAIT→锁内 author 候选→Identity NOWAIT，确认 deleted 才生成清理 proof。active/suspended 候选不清理，Identity 锁仍由 tx 持有至提交。source 不取 actor/tombstone，不改 operation.kind/source。 |
| absent 与切片① F001 | orphan 有可信 operation source 时允许 absent 前像，通过原 operation/tombstone 与后像事实核验后清理；ordinary/violating absent 无来源时不构造清理 proof。现存 pending 的 version、operation ID、state/source 重验，不把 present mismatch 当“已删”。与切片① R02 接口相容。 |
| 删除完成及失败回滚 | `deletion_finalizer.go:615`～`:660` 同 tx DELETE、完成审计、operation.completed、proof 回读及 Identity 条件清理，全部成功后由 T2 commit；失败回滚 T2，不撤销 T1/Journal/已提交 claim。commit unknown 返回错误，completed 重试短路。Identity 的计数门禁、成对空值/释放审计幂等保持切片①责任。 |
| 反序退让与 worker | coordinator 仍可能 Article→operation 等待，但 T2 对 Article 和后续 Identity 只用 NOWAIT，失败整体退出释放 operation/Article。`ConvergeDeletionOperation`→`Finalize` 不经过 HTTP coordinator 的 ErrDeletionInProgress 包装；55P03 原因链保留且不携带该 sentinel，符合 worker `deletion_worker.go:209` 的 transient 分支，走既有退避；已提交 lease 留待到期重领。未引入重试框架。 |

没有按当前未接线 root/HTTP 的缺失提出本片缺陷。无需 Reviewer 代选的方向决定；无新增非阻断整改项。

## 实际验证与明确限制

Reviewer 现场执行 `go test ./internal/article -run '^(TestOrphanDisposition|TestPermanent|TestDeletion)' -count=1` **PASS**；五文件 `gofmt -l` 无输出。未重复全仓/race/vet/Manifest。测试包括既有删除回归、资格/漂移/零行、SQL 回读、proof 不可由公开投影修改、来源分支、T2 故障保留 claim、55P03 原因链和租约精度/锁后时钟。

Lead 报告：本片专项 14 顶层/51 节点 PASS、七项 Manifest 按序 PASS；起始 667 文件对账只有授权范围变化且无删除。此为 Lead 提供的现场结果，未冒充 Reviewer 独立重跑全仓。

**本轮未连接真实 PG。** 新测试通过 production Repository 方法执行的是模拟 pgx.Tx/Identity；这可验证分支、参数/调用合同和模拟回滚，不能证明 PostgreSQL 实际解析、行锁/表锁、RETURNING 精度、隔离快照或跨模块原子性。`NOWAIT_RemainsWorkerTransient` 测试直接断言 finalizer 错误链，实际 worker 消费路径本轮由静态调用链核对，未把它称作 worker+PG 联合实测。

后续门禁：切片③须将 frozen Identity 能力与真实 Article 私有 proof 通过固定 bound verifier 连接，不能只重包 public DTO；补唯一 root/HTTP/错误映射，恢复并验证受本片新增 Identity 依赖影响的既有删除入口；按 test-plan r14 完成真实 PG 多连接下 canonical/反序竞争、三步自动清理、同快照计数、审计/清理全回滚和 lease 案例。无来源 replay/active 旧备份删除与恢复 Gate 的既有限制不由本片关闭。

本报告只交回当前 Lead，局部 PASS 不替代完整任务、人的接纳或后续授权。报告完成、核对哈希后停止写入。
