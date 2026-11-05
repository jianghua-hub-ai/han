# S2.2 主会话验证记录

文章 PostgreSQL 专项已完成自身闭环：**最终 R02 普通/race、三组新负控、Lead Manifest七项和独立 R02均通过；F001/F002均关闭，无新增问题**。Lead已核实际diff/源码、原始运行、数据回读、资源与范围，原任务关闭以Tasks为准。下面保留R01历史，不用旧结果代替R02。

固定工作区 `D:\AID\VSCode\Codex\Han`，main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本轮未暂存、提交、推送、部署、删除、移动或清理文件/schema。

## 产出、责任与固定输入

- 原作者 Test-Engineer Ohm `01a17c83-60f4-7803-b0cc-9b19c0b81fa3` 在原三文件内完成R01及两项返工；最终R02已实际返回completed/stopped。
- 唯一测试写集为 `test/integration/article/postgres_state_test.go`、`pagination_test.go`、`etag_concurrency_test.go`；integration tag、同 package，最小 fixture 内承首文件，只读复用已关闭 S7.2 的 retained harness。
- [最终R02固定157输入](evidence-s2.2/lead-fixed-source-r02.json)绑定真实依赖、article设计/测试计划和三测试；仅State/Pagination两测试相对R01变化，ETag与生产依赖未变。作者普通/race、独立Reviewer及Lead Manifest前后逐项值一致；[R01索引](evidence-s2.2/lead-fixed-source-r01.json)保留，不以manifest自身序列化字节不同推断源码变化。
- 固定 article design r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`；test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`。仅消费原 S2.2 的 PG 状态/分页/锁/回滚责任，未改变业务、API 或设计。
- 独立Reviewer Erdos `01a17da1-6bd9-7d43-8642-b8b502127296` 的 [R02](review-code-s2.2-r02.md) SHA256 `1D5A570A84986383D235FD3BF65F7394C079F4C13DFAD4777236632D6BE8FEC1` 为PASS，F001/F002均CLOSED、0新增material finding，已completed/stopped。[R01](review-code-s2.2-r01.md) FAIL原件保留：两项MEDIUM为测试保护不足，不是已观察到的产品错误。

## 最终R02核证

| 执行者/检查 | 最终结果与证据 |
|---|---|
| Ohm：完整普通/race | 各3顶层、18风险组、77 run/pass节点、60 leaf，0 fail/skip；[普通](evidence-s2.2/ordinary-final-r02.result.json)、[race](evidence-s2.2/race-final-r02.result.json)、[Lead逐项JSON解析](evidence-s2.2/lead-final-json-verification-r02.json) |
| Ohm：新负控 | clear-action、sexual-category、older-timestamp三种单点生产mutant，旧R01测试各实际GREEN、新R02各实际行为RED；没有改生产工作树；完整snapshot/overlay/raw见[R02交接](evidence-s2.2/handoff-s22-r02.json) |
| 独立Erdos R02 | race窄测3顶层、21 run/pass节点、13 leaf，0 fail/skip，14新schema保留；另只读回查14新schema和3作者负控样本；[独立分析](evidence-s2.2/review-r02/narrow.analysis.json) |
| Lead：最终Manifest R02 | 七项按配置顺序全部exit0、gofmt无输出，157输入一致；[索引](evidence-s2.2/lead-manifest-r02-results.json) SHA256 `4F728382C2FD343B7B72D4CE39E51CAE20BDA1C0C0186C733D312EAFA6B8014D` |
| Lead：最终PG回读 | 124个最终运行schema全部存在；回读跨3个时间戳的交错ID序列、sexual mark/clear v2/v3与restore赢家violence审计均正确；[结果](evidence-s2.2/lead-postgres-readback-r02.json) |

F001以预期version为键精确核action/category，来自既知操作和不可变前像；保留actor/target/trace/outcome、总数/版本唯一性及回滚/锁释放。F002增加6行、3时刻、ID与时间交错的真实fixture，limit2跨三页；独立SQL先核明确序列，再核真实遍历，保留原lookahead和最后返回项cursor。节点76→77而leaf仍60，是原lookahead节点增加子例，父节点自身的原PG断言仍执行。

两个审计mutant分别只错clear动作、只错sexual类别，Article/SQL事务继续真实执行；分页mutant只去掉旧时间戳分支。原测试各有非零run/pass，新测试分别在精确审计或跨时间序列上失败，证明新增保护能检错，不是编译失败或直接硬编码失败。固定实现完整GREEN。R02新增153个作者schema全部保留，和两全套124个、Reviewer新14个分别记账。

[R02新增副本索引](evidence-s2.2/lead-copied-evidence-r02.json)保管74份新增/修订证据，变更过的工具另存r02-updated而不覆盖R01；[独立复审副本](evidence-s2.2/lead-copied-review-r02.json)保管14份。原始Go工具以txt承载。完整[固定r18设计](evidence-s2.2/fixed-r18-design.md.txt)和[固定r15测试计划](evidence-s2.2/fixed-r15-test-plan.md.txt)已逐字节保留，便于后续管理员设计增量后仍核本次版本。

## R01实际执行与归属

| 执行者/范围 | 实际结果 |
|---|---|
| Ohm：完整普通/race | 各3顶层、18风险组、76 run/pass节点、60 leaf，0 fail/skip；[普通](evidence-s2.2/ordinary-final-r01.result.json)、[race](evidence-s2.2/race-final-r01.result.json) |
| Ohm：顶层/tagged vet | 精确-list三个顶层、`go vet -tags=integration ./test/integration/article`均exit0；不把-list/vet输出记成用例 |
| Ohm：游标负控 | Temp完整源/测试快照+Go overlay将最后返回项游标错用第一项；真实PG结果触发断言失败；[负控](evidence-s2.2/negative-cursor-r01.result.json) |
| Ohm：旧版本负控 | Temp overlay让旧版本在真实行锁后被接受；失败方实际返回200而非412，断言检出；[负控](evidence-s2.2/negative-stale-r01.result.json) |
| 独立 Erdos | race窄测3顶层、26 run/pass节点、17 leaf，0 fail/skip，18个schema保留；另只读回查18个schema及行/审计实际正确；未伪称已执行F001/F002新mutant |
| Lead：逐行JSON核对 | 确认三个顶层、18组及全部60 leaf实际命中、包级pass/零fail/skip与157项before/after一致；[解析](evidence-s2.2/lead-final-json-verification-r01.json) |
| Lead：Manifest七项 | 配置顺序全部exit0，gofmt无输出；[原始索引](evidence-s2.2/lead-manifest-r01-results.json)，SHA256 `FC2274ED3C187021A62E6ACE3280BE36EE195DFB4DD5A9210F6E659C8999EE32` |
| Lead：真实PG回读 | 最终普通/race的122个schema全部存在；抽查7个case最终row/version/治理/字符长度及适用audit正确；[结果](evidence-s2.2/lead-postgres-readback-r01.json) |

完整专项命令为 `go test [-race] -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$' -count=1 -timeout=8m -json`。Manifest顺序为 gofmt、mod verify、vet、build、contract、race、unit；默认全包不编译 integration tag，不能替代本专项及独立 Review。

所有目标 SQL、行锁、Audit.Store 与事务均真实运行；wrapper 只在实际查询、UPDATE 或审计 INSERT 后注入故障/屏障。身份 missing/fault 在真实 Reader 后受控返回，不删除 FK 所指用户。输入边界直调现役 Handler 配真实 PG 前后快照；可信合成 Principal adapter 不代表真实生产会话/安全链。纯内容替换按现役合同不制造新的审计。管理员第三投影为真实 SQL/Policy，不是虚构管理 GET。

并发用例先让一方真实持有FOR UPDATE，再核另连接PID及pg_blocking_pids等待后释放；既测两种内容/治理顺序，也区分不合法state、stale version和deletion优先级，不以sleep猜时序。R01的分页独立SQL oracle、密集隐藏、多批、平局、lookahead、空末页和跨请求变化有效，其跨时间戳盲区已由上述R02明确补齐。

## 环境、历史失败与保管

实际目标既有 `127.0.0.1:5433 PostgreSQL18.6`；User级DSN只注入进程。每fixture先提交随机隔离schema，再执行完整0001～0009迁移与合法CHECK/FK合成数据；pending先创建对应DeletionOperation。复用harness的 `han_s72_` 前缀，以本任务case日志区分。作者R01共221个schema保留，最终两组122个另见[case清单](evidence-s2.2/lead-final-schema-map-r01.json)。rows/tx/context/group/pool实际关闭，未使用旧自动DROP helper，未创建SMTP或生产服务。

初始编译夹具字段 Kind→DeleteKind 的修正不作行为RED。`cases-green-r02` 虽exit0但零run，是无效验证，原日志保留；已以有实际命中的后续执行替换；R01最终为76节点，当前R02最终为77节点（均60 leaf）。两个故意overlay的FAIL是检错证据，不是当前产品FAIL；快照/完整mutant/overlay与失败schema均保留，不补造历史。

[作者及Lead证据保管索引](evidence-s2.2/lead-copied-evidence-r01.json)列65份逐字节副本及hash；原始Go工具以txt保存，避免改变项目格式检查范围。[独立证据索引](evidence-s2.2/lead-copied-review-r01.json)保管Reviewer原始窄测、读库与静态反例；静态反例不冒充已运行mutation。复制前未发现真实DSN全文或私钥，临时原件保留。

用户已明确另一个会话推进identity模块。899文件基线中五份identity/ADR/observability文档及Tasks身份注记的变化不由本会话产生，按并行支线现状保留，不恢复、不继承其S3.3/S3.5豁免。本文只核文章固定对象；共享任务文件仅合并文章增量并保留身份段。已知S1.5五条生产路由缺口另由正式取证/规划承接，不用本地Handler通过掩盖，也不阻塞本卡可独立验证部分。

## 当前交回与未验范围

F001/F002已由原作者在原三文件内整改并经原Reviewer复审关闭；最终新修订真实负控、完整普通/race、项目检查、原始证据/数据/资源和Lead核证均齐备，原S2.2已满足自身关闭条件。所有本卡作者/Reviewer停写；后续管理员设计或生产接线是独立范围，须按各自固定输入、写集与回归交接，不由此PASS自动覆盖新修订。

本卡不证明生产router/Host、完整HTTP权限/error/parser矩阵、浏览器、数据库中断/commit unknown、Journal真卷/旧备份恢复/finalizer全协议、部署E2E、容量/P95或人的整体验收。新增管理员查询及既有Handler生产接线仍是后续独立范围。
