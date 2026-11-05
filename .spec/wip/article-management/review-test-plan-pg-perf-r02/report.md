# R02 独立非作者复审报告

## 1. 对象与固定事实

| 项 | 值 |
|---|---|
| 被评审对象 | `.spec/wip/article-management/test-plan-pg-perf-r02.md` |
| 大小 / 行数 | 57,753 B / 462 行 |
| SHA256 | `BEAD1F1C3BBF21723F2DE761203556E559AC109F6E62F2445A16151648129096` |
| 编码 | UTF-8，无 BOM |
| R01 对象（未修改） | `.spec/wip/article-management/test-plan-pg-perf-r01.md`，41,112 B，SHA256 `307060B6FCF66E33B2E8E57E2AF3D59142A3DCB47F9B499EF2E1C468DB86A56C` |
| R01 评审报告 | `.spec/wip/article-management/review-test-plan-pg-perf-r01/report.md`，32,659 B，SHA256 `B3F50FF6357A9BBBB6FB8E3FAC39C0E0BA6B62CA5FD28E2AFE6097350CE37A50` |
| 工作区 / 分支 / HEAD | `D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 评审类型 | 独立非作者 Reviewer 复审；只读、静态核对，不执行测试 |
| Lead 已定事项 | 以 R02 作为执行基线；`article-management-S2.4` 卡面与 Plan 的 r01 命名由 Task-Planner 后续同步。本报告不把该待同步项升级为阻塞项 |

本轮唯一写集为本报告。未修改被评审文件、`tasks.md`、Plan、代码、迁移、测试或文档；未运行 `go test`、integration 或 benchmark；未连接 PostgreSQL；未读取或打印 `HAN_TEST_POSTGRES_DSN` 的值；未执行 Git 写、删除或移动。

## 2. 实际读取清单

- `.spec/wip/article-management/test-plan-pg-perf-r02.md`：全文 462 行（分段读取 `:1-59`、`:60-75`、`:76-87`、`:88-126`、`:127-183`、`:184-238`、`:239-320`、`:321-462`），重点核对处置表、场景表、路径表、采样/预算、执行顺序、停止条件、证据树与结论。
- `.spec/wip/article-management/review-test-plan-pg-perf-r01/report.md`：全文 196 行，逐条读取 F001～F010 的原始问题、证据与复审条件。
- `.add/project/knowledge/design-workspace/article-management/design.md`：`:186-194`、`:957-986`（AM-Q1、AM-Q2、r29/r30 容量参数确认、P95 边界）。
- `.add/project/knowledge/design-workspace/article-management/test-plan.md`：`:878-890`、`:1023-1031`（A-PERF-01～05 状态、A-PERF 映射）。
- `.add/project/knowledge/tasks/tasks.md`：`:1415-1478`（S2.2/S2.3 历史边界、S2.4 卡面、r01 命名与旧预算）。
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`：`:890-925`（B11 Context Pack、S2.4 的 r01 Read Set/写集/预算）。
- `docs/benchmarks/article.md`：`:186-195`（S2.3 旧注记与历史状态）。
- `internal/article/repository.go`：`:604-616`、`:657-675`、`:730-762`（`LockDueNoticeRecipient`、`ClaimNoticeRecipient`、due predicate、`FOR UPDATE OF recipient SKIP LOCKED`、CAS preimage）。
- `internal/article/notice_dispatcher.go`：`:461-505`（dispatcher 的 lock → DB 时钟 → due 校验 → claim → commit 顺序）。
- `internal/article/query_service.go`：`:130-305`（`GetArticle`、`GetAuthorArticle`、`ListCandidateBatch`、limit 校验）。
- `internal/article/command_service.go`：`:20-25`、`:523-531`、`:589-631`、`:676-684`（`maxArticleBodyRunes=20000`、Create/ReplaceContent、锁与提交）。
- `internal/article/deletion_coordinator.go`：`:120-190`、`:287-295`（`beginOrLoad`、`LockArticleForDeletion`、operation/Article 同事务顺序）。
- `internal/article/policy.go`：`:115-164`（`MemberVisible`、`MemberListProjection`、`MemberDetailProjection`、`AuthorProjection`）。
- `migrations/0003_article.sql`：`:124-132`、`:181-189`（`articles_created_at_id_idx`、`transfer_notice_recipients_claim_idx`）。
- `test/integration/article/notice_dispatcher_test.go`：`:1-3`、`:66-124`（build tag、`articleintegration`、MaxConns=4、90s ctx、`han_s72_*`、0001～0009）。
- `test/integration/article/audit_insert_rollback_test.go`：`:30-40`、`:100-138`、`:350-366`（S1.5 回滚装置、`han_s15audit_*`、90s ctx、真实 `audit.Store.Append`）。
- `test/integration/article/`：现有测试文件清单核对（`admin_query_test.go`、`audit_insert_rollback_test.go`、`etag_concurrency_test.go`、`notice_crash_test.go`、`notice_dispatcher_test.go`、`notice_smtp_test.go`、`pagination_test.go`、`postgres_state_test.go`）。

## 3. 方法

1. **原 finding 回源**：先按 R01 报告的 F001～F010 逐条定位 R02 的处置表、场景、预算和证据位置，不以 R02 自述“已关闭”作为结论。
2. **权威事实核对**：AM-Q2 参数、首期规模、P95 边界回源 `design.md`；A-PERF-01～05 状态回源 `test-plan.md:882-886`；历史卡面与旧文档回源 `tasks.md`、`docs/benchmarks/article.md`。
3. **代码可实施性核对**：沿真实调用链核对 article 创建/修改、成员/作者详情、候选批次、删除 T1、审计和通知 lock/claim；核对既有 build tag、package、helper、schema 与写集。
4. **算术与回归重算**：对 §9.4 的 small/medium/large 点数、warm-up、重复、seed、报告与预留逐项重算；检查 500 样本、120s/60s、`statement_timeout`/`lock_timeout`、A-PERF 状态与 r01/r02 命名是否仍有矛盾。
5. **限制声明**：本轮没有动态执行、没有真实 PG 数据、没有编译新测试文件；对 seed 用时、DB 增量、执行计划、锁行为和 `go test` 实际超时只作静态判断。

## 4. 逐项核查结论

| 核查维度 | 结论 | 证据 |
|---|---|---|
| 事实一致性（AM-Q2） | **无 finding。** AM-Q2 确于 2026-11-02 关闭；`DefaultLimit=20`、`MaxLimit=100`、`CandidateBatchSize=200`；首期 100000 篇/单用户 10000 篇；P95 列表≤300ms、详情≤150ms，均明确为开发机参考、非目标环境结论。R02 没有把参考值写成达标结论。 | `design.md:190`、`:961-984`；R02 `:51-58`、`:122`、`:456-457` |
| A-PERF 状态 | **无 finding。** R02 统一为 A-PERF-01/04 partial、A-PERF-02 planned（目标环境 NOT RUN）、A-PERF-03/05 blocked，与 `test-plan.md:882-886` 一致；`tasks.md:1441` 的 AM-Q2 旧注记已明确标为历史。 | `test-plan.md:882-886`；R02 `:19`、`:122-125`、`:208`、`:458` |
| 代码路径可实施性 | **路径事实无 finding。** `ListCandidateBatch` 使用 keyset；`GetArticle` 走成员详情投影；`GetAuthorArticle` 走作者投影；`Create`/`ReplaceContent`/`Moderate`、`beginOrLoad`/`LockArticleForDeletion`、`audit.Store.Append` 均与 R02 描述一致。通知路径已正确拆成 `LockDueNoticeRecipient` 的 `FOR UPDATE OF recipient SKIP LOCKED` 与 `ClaimNoticeRecipient` 的 CAS `UPDATE … RETURNING`。 | `query_service.go:134-305`；`command_service.go:527-684`；`deletion_coordinator.go:125-190`、`:291`；`repository.go:611-671`、`:736-760`；`notice_dispatcher.go:471-499` |
| 新文件、package、build tag、schema 与写集 | **命名和冲突边界基本可落地。** 四个新文件均位于 `test/integration/article`、`package articleintegration`、`//go:build integration`；现有八个测试文件没有被列入修改；helper 名与 `noticeHarness`/`noticePoolConfig` 等不重名；schema 格式 `han_pgperf_r02_<case≤8>_<16hex>` 最长 40 字节，低于 63 字节上限。**但 schema 生命周期与 180s ctx 有冲突，见 R02-F002。** | R02 `:212-219`、`:236-240`、`:259-263`；`notice_dispatcher_test.go:1-3`、`:70-124`；`audit_insert_rollback_test.go:30-40`、`:105-138` |
| 测试设计质量 | **核心正确性纪律正确。** 独立 SQL oracle、禁止用被测分页算法生成期望、禁止用 candidate 返回条数代替实际扫描行、禁止 sleep 猜时序、负例与性能失败分开统计，均已写明。**但定时点没有覆盖全部 A-PERF-01/04 所需的创建/修改与 T1 批量观测，见 R02-F001。** | R02 `:142-144`、`:225`、`:273-281`、`:316`、`:327`、`:374-381` |
| 可行性与预算 | **§9.4 的算术本身自洽**：small/medium 每重复 `10s warm-up + 30s 测量 = 40s`，3 次 `= 120s`；small 9 点 `= 18min`；medium 15 点 `= 30min`；加 seed、正确性、环境与预留 `= 90min`。large `8×60s = 8min`，加 seed/断言/预留 `= 60min`；总 `90+60 = 150min`。**但点数清单与场景定义不一致，且候选 `go test` 命令没有落实超时/重复/测量窗，见 R02-F001、R02-F002、R02-F003。** | R02 `:285-296`、`:365-412`、`:443-448` |
| 100k seed 具体性 | **基本足够。** 明确 users → articles → deletion_operations → notices/recipients 顺序、确定性 PRNG、`@example.test` 合成邮箱、`CopyFrom`/批处理、分档行数、正文分层、`ANALYZE`、seed 单独计时、DB 增量 <2 GB 与 30min 停止线。没有要求新增 Go 依赖。 | R02 `:244-251`、`:329-347`；`migrations/0003_article.sql:128-129`、`:185-186` |
| 边界与不过度声明 | **无 finding。** 明确不宣称 SLO/容量/production readiness；区分 S2.3 进程内基线与真实 PG；不重开 S2.2/S2.3；不冒充 S1.5/S8.2/E2E；NOT RUN/blocked 项如实列出。 | R02 `:6-7`、`:18-23`、`:109-113`、`:195-207`、`:318-325`、`:351-363`、`:454-461` |
| 证据完整性 | **无 finding。** 证据树含 `environment/fingerprint.json`、`schema-inventory.json`、`migrations/sha256.json`、`seed-manifest.json`、`raw/`、`plans/`、`blocked.md`、`checksums.txt`、`summary.md`；禁止 DSN/PII，重跑不覆盖旧证据，Lead 核对 checksums。 | R02 `:231-234`、`:414-435` |

## 5. F001～F010 核销表

| Finding | 原始缺陷 | R02 对应处置与回源证据 | 判定 |
|---|---|---|---|
| `PGPERF-PLAN-R01-F001` | 预算自相矛盾：每点 `3×(30+30)=180s` 与“单点≤60s、总≤90min”冲突 | §6.2 定义测量点/重复/单点；§9.4 改为 small/medium `3×(10+30)=120s`，9 点=18min、15 点=30min；large `8×60s=8min`；总 `90+60=150min`，与 145min 硬停止一致。算术复核通过。 | **CLOSED**（算术口径）；点数完整性另见 R02-F001 |
| `PGPERF-PLAN-R01-F002` | 把 `ClaimNoticeRecipient` 误写成 `FOR UPDATE SKIP LOCKED` | R02 `:96`、`:169`、`:170`、`:172` 已拆分：`LockDueNoticeRecipient` 为 `LIMIT 1 FOR UPDATE OF recipient SKIP LOCKED`；`ClaimNoticeRecipient` 为 CAS `UPDATE … RETURNING`，preimage 为 status/attempt_count/claimed_by/locked_until/next_attempt_at。与 `repository.go:611-671`、`:736-760`、`notice_dispatcher.go:471-499` 一致。 | **CLOSED** |
| `PGPERF-PLAN-R01-F003` | A-PERF-01 缺少 near-limit 创建点 | W01 增加典型 ~1 KiB 与 near-limit（3 字节 UTF-8、约 20,000 rune、约 60 KB）两个创建点；W02 两档正文；W04 只做 near-limit 修改；seed 正文分层映射到创建场景。 | **CLOSED**（场景定义）；W03/W04 未进入定时表另见 R02-F001 |
| `PGPERF-PLAN-R01-F004` | 只有作者详情 R06，缺成员详情 | 新增 R07：`GetArticle` → `readAuthorStates` → `MemberDetailProjection` → `StrongETag`；R06 作者详情保留且分开报告；R07 对照详情≤150ms 仅记录观测。 | **CLOSED** |
| `PGPERF-PLAN-R01-F005` | 把 A-PERF-02 写成 blocked，与 test-plan 不一致 | R02 `:19`、`:122`、`:208`、`:458` 统一为 A-PERF-02 planned（AM-Q2 已关闭，目标环境 NOT RUN）；`tasks.md:1441` 标为历史注记；与 `test-plan.md:882-886` 一致。 | **CLOSED** |
| `PGPERF-PLAN-R01-F006` | 列表场景混入 404，空断言风险 | R01～R05、X02/X03 改为只断言 `items` 不含隐藏项、空集合、`next_cursor=null`、不漏不重；404 只用于详情 R06/R07 及详情负例。 | **CLOSED** |
| `PGPERF-PLAN-R01-F007` | harness 复用、文件名/helper 名和写集未落边界 | 固定四个新文件及 `newPgPerfHarness`、`pgPerfPoolConfig`、`pgPerfMigrate`、`pgPerfSchemaName`、`pgPerfSeed`、`pgPerfRunPoint`；明确不改既有八个测试文件；新文件名当前均不存在，符号不重名。 | **CLOSED**（命名/写集）；180s ctx 与 seed 冲突另见 R02-F002 |
| `PGPERF-PLAN-R01-F008` | 仅 E01 禁用 sleep，其他并发场景无确定性方法 | D02/N02/N04/X08 统一事务持锁 + `WaitGroup`/channel/barrier；`pg_blocking_pids`/`pg_locks`/`wait_event` 有界轮询；N04 用 SQL 回拨 `locked_until`/`next_attempt_at`；全文禁止 sleep 推断。 | **CLOSED** |
| `PGPERF-PLAN-R01-F009` | schema 标识符长度未约束，可能截断碰撞 | 格式 `han_pgperf_r02_<case≤8>_<16hex>`，前缀 15 + case≤8 + 1 + 16 = ≤40 字节；`CREATE SCHEMA` 前断言 ≤63 并记录完整名。 | **CLOSED** |
| `PGPERF-PLAN-R01-F010` | 证据树缺 seed、迁移校验和、blocked 独立落盘 | 新增 `seed-manifest.json`、`migrations/sha256.json`、`blocked.md`；`fingerprint.json` 纳入 schema、MaxConns、档位、并发；`checksums.txt` 与 Lead 核对要求明确。 | **CLOSED** |

**核销统计：10 CLOSED / 0 PARTIAL / 0 OPEN。** R01 的原始十项 finding 均已被 R02 的对应修订覆盖；R02 仍存在新的执行性与覆盖问题，单独编号如下，不改变上述核销结论。

## 6. 新 Findings

| ID | 严重度 | 位置（行号） | 问题 | 证据 | 建议 |
|---|---|---|---|---|---|
| `PGPERF-PLAN-R02-F001` | **HIGH** | R02 `:121-124`、`:151-181`、`:392-395`、`:405`、`:444` | **定时点清单不完整，覆盖声明大于实际测量。** small/medium 只给 W01/W02/R01/R03/R04/R06/R07/D01/N01/N02 等点；W03/W04（唯一的典型/ near-limit 修改性能场景）、D04（100/1000/10000 T1 pending 吞吐）、E01～E03、D02/D03、N04 没有定时点；N03 只有 large 单点，small/medium 无定时点。R03 写了隐藏密度 50/90/100，R04 写了空库/10 行/全隐藏/最后一页，但 §9.4 各只计 1 点；按 §6.2“档位×场景×并发/数据组合”的定义，这些是多个测量点。大档“先跑 1 个探索点验证 harness，再按 ≤8 点执行”也没有说明验证点是否占 8 点额度；若不占，实际是 9 点，`9×60s=9min`，large 变 61min，总变 151min。 | §6.2 `:287` 定义测量点；§9.4 `:392` small 9 点、`:394` medium 15 点、`:405` large 8 点；`:395` 明确“未定时场景只做正确性断言，不计性能样本”；W03/W04 `:153-154`；D04 `:168`；R03/R04 `:157-158`；`:444` PGPERF-B 关闭条件要求“near-limit 创建与修改均有测量点”；`:371` large 先跑探索点；`:121/:124` A-PERF-01/04 的吞吐/积压要求。 | 增加一张“场景×档位×并发/数据组合→是否定时→单点上限→小计”的完整矩阵；至少把 W03、W04、D04 纳入定时表，并明确 R03 三个密度、R04 四个数据集、W02 各并发组合是否各算点；明确 large 验证点是否占用 8 点额度；重算 90/150min。若预算无法容纳，必须把相应点标 `NOT RUN` 并同步降低覆盖声明，不能一边不计时一边宣称覆盖。 |
| `PGPERF-PLAN-R02-F002` | **HIGH** | R02 `:218`、`:250-251`、`:259-263`、`:368-371`、`:393-395`、`:404-412` | **每个 scenario 的 180s context 与每档 seed 预算/每场景独立 schema 冲突。** `:218` 写“每个 scenario `context.WithTimeout(..., 180s)`”；`:259-263` 写“每场景独立”schema；但 medium seed 上限 600s、large seed 上限 1800s，且 `:368-371` 的顺序是“建 schema → 迁移 → seed → ANALYZE → 测量”。180s 无法覆盖 medium/large seed；即使 small seed 120s 加一个 120s 测量点也已是 240s。另 `:250-251` 又说读场景只读、写场景独立 schema 或独立文章、每个档位保留独立 schema，和“每场景独立”不一致，导致无法判断是按档位 seed 一次还是按场景重复 seed。 | R02 `:218`（180s）；`:250-251`（读/写隔离、档位 schema）；`:259-263`（每场景独立、seed <2/<10/<30 分钟）；`:368-371`（建 schema→seed→测量）；`:393` medium seed 600s；`:404` large seed 1800s。 | 明确 schema 生命周期：建议只读场景共享每档一个 schema，写场景使用独立 schema 或独立文章；把 `seedCtx` 与 `pointCtx` 分开，给出 `seed+measurement+cleanup` 各自超时（small/medium/large），180s 只约束单个测量点或单个短 fixture；按最终 schema 数量重算 seed 次数与总预算。 |
| `PGPERF-PLAN-R02-F003` | **MEDIUM** | R02 `:221`、`:291`、`:387`、`:443-448`、`:389-412` | **候选 `go test` 命令没有落实超时、重复和测量窗，且大档正则可能与总命令重叠。** 命令 `go test ... -run '^TestPgPerf' -count=1 -json` 没有 `-timeout`，Go test 默认 10 分钟，而 small+medium 预算 90 分钟、large 60 分钟，默认会被杀；`-json` 不改变 timeout。benchmark 形态也没有 `-benchtime=30s`/`-count=3`（或等价自定义固定时长循环），默认 `-benchtime=1s`、`-count=1` 无法满足 §6.2 的 3 次 30s 重复。另 `-run '^TestPgPerf'` 会匹配 `TestPgPerfLarge` 之类的大档顶层名，和“large 独立窗口”冲突。 | R02 `:221`（候选命令）；`:291`（3 次、≥30s）；`:387`（3 重复 120s 口径）；`:443-448`（各 PGPERF 命令）；`:389-412`（90/150min 预算）。 | 为每档给出精确命令：small/medium 与 large 分开 `-run` 或使用不同 build tag/测试名；增加显式 `-timeout`（例如 full 2h、large 70m）和 `-benchtime=30s`、`-count=3`，或明确使用自定义固定时长循环；把实际 timeout、benchtime、count 写入 `fingerprint.json`/`summary.md`。 |
| `PGPERF-PLAN-R02-F004` | **LOW** | R02 `:65`、`:291`、`:385-387` | **“≥500 样本”与“30s 窗 + 3 重复可同时满足”的表述不是普遍成立。** 3×30s=90s，500 样本要求吞吐至少约 5.56 ops/s；低于该吞吐的点在 90s 内达不到 500。R02 已给“合并后仍 <500 标 `exploratory (N<500)`，不得突破 120s”的兜底，因此不是硬矛盾；但 `:387` 的“可同时满足两个下限”应改为条件句，否则低吞吐点会被误读为必须凑满 500。 | R02 `:65`（测量≥30s 或≥500 样本，取较长者）；`:291`（单点上限 120s）；`:385-387`（合并 3 次重复、N<500 兜底、仍称可同时满足）。算术：500/90s≈5.56 ops/s。 | 把 500 定义为**软目标**；最终规则写成“3 次重复合并 N≥500 时报告分位数；N<500 时标 `exploratory (N<500)`，不延长超过 120s”；删除“30s 窗+3 重复必然同时满足”的绝对表述。 |
| `PGPERF-PLAN-R02-F005` | **LOW** | R02 `:216`、`:238`、`:416`、`:439`；`tasks.md:1446`、`:1456-1458`、`:1466-1467`、`:1474`；Plan `:911-917` | **R02 命名与已登记 S2.4 卡面/Plan 仍不一致（Lead 已决定以 R02 为准，Task-Planner 待同步）。** R02 用 `pg_perf_notice_delete_test.go`、`han_pgperf_r02_*`、`evidence-pg-perf-r02/`；卡面/Plan 仍写 `pg_perf_notice_test.go`、r01 前缀/证据目录/r01 SHA、旧“单点≤60s、总≤90min”和“200 样本或 30s”口径。R02 §10 又说 PGPERF-A～G“建议，未登记”，但 S2.4 已登记并明确把 PGPERF-A～G 作为卡内实施切片。`tasks.md:1446` 还保留 A-PERF-02/03/05 blocked 的 S2.3 历史状态。 | R02 `:216`、`:238`、`:416`、`:439`；`tasks.md:1446`（旧 A-PERF-02/03/05 blocked）、`:1456`（旧文件名）、`:1458`（r01 证据目录）、`:1466`（r01 schema/旧预算）、`:1467`（200 样本）、`:1474`（r01 SHA）；Plan `:911-917`（r01 Read Set/写集/预算）。 | 由 Task-Planner 将 S2.4 卡面与 Plan 同步到 r02：文件名、schema 前缀、证据路径、SHA、采样/预算口径；r01 仅保留为历史锚点。明确 PGPERF-A～G 是已登记 S2.4 的实施切片，不是未登记建议。该同步不阻塞本轮 R02 评审。 |

**新 finding 统计：5 项；HIGH 2、MEDIUM 1、LOW 2。**

## 7. 回归检查

- **500 样本 vs 3 次重复**：R02 已把 500 定义为测量点合并 3 次重复的样本总数，并给出 `exploratory (N<500)` 兜底，主体口径一致；仅 `:387` 的绝对化措辞需要修正（R02-F004）。
- **单点≤60s 与 120s 并存**：R02 正文已按档位分层：small/medium ≤120s、large ≤60s；`:265` 明确不再使用旧的笼统“单点≤60s、总≤90min”口径，因此正文没有把 60s 与 120s 混用。旧值只残留于 `tasks.md`/Plan（R02-F005）。
- **`statement_timeout` / `lock_timeout`**：未发现与 warm-up/重复的直接冲突。它们是语句级设置，warm-up/measurement 是客户端循环；但具体值未固定，执行时必须把实际值写入 `fingerprint.json`/`summary.md`。这属于残余执行风险，不单独列为 finding。
- **A-PERF 状态**：R02 与 `test-plan.md:882-886` 一致：01/04 partial、02 planned（目标环境 NOT RUN）、03/05 blocked；没有把 R02 观测升级为通过。
- **命名一致性**：R02 自身统一使用 r02 命名；与已登记 S2.4 卡面/Plan 的 r01 命名不一致，Lead 已决定以 r02 为准，按 LOW 记录（R02-F005）。
- **large 探索点**：`先跑 1 个探索点验证 harness，再按 ≤8 点执行` 是否把验证点计入 8 点不明确；若不计入会出现 9 点，已并入 R02-F001。
- **seed 可行性**：行数、附属数据、正文分层、`CopyFrom`/批处理、确定性 PRNG、`ANALYZE`、2GB 与 30min 停止线均已具体化；但 100k 文章/10k 用户/100k recipients/50k notices 在真实 FK/trigger 下的 30min 与 2GB 仍只是估算，未实测。没有因此新增 finding。
- **证据完整性**：F010 已关闭；DSN/PII 禁止、fingerprint、原始样本、checksums、blocked 独立落盘均满足后续 Lead Gate/复审的最低结构要求；没有发现证据树缺口。
- **边界与不过度声明**：R02 明确不宣称 SLO、容量、production readiness；区分 S2.3 进程内基线与真实 PG；不重开 S2.2/S2.3；不冒充 S1.5/S8.2/E2E；NOT RUN/blocked 项如实标注。该维度无 finding。

## 8. 未验证风险与限制

- 未执行任何 `go test`、integration 或 benchmark；未连接 PostgreSQL；未读取或打印 `HAN_TEST_POSTGRES_DSN` 的值。所有 seed 时间、DB 增量、执行计划、buffers、锁等待和连接池行为均未实测。
- 四个新测试文件尚未创建、未编译；文件/helper 命名与既有八个测试文件不冲突是静态核对结论，不等于编译通过或 `-list` 已命中。
- R02-F003 的 `go test` 默认 10 分钟超时是 Go 工具链行为判断，未实际触发；large 的 30min seed、2GB 增量与真实 PG 资源上限仍需执行时验证。
- `statement_timeout`/`lock_timeout` 的具体值未固定；执行时必须记录，避免用环境默认值产生不可比结果。
- S1.5 写窗与其他角色未提交改动未实际核验；本报告只确认 R02 没有把 `audit_insert_rollback_test.go` 列入写集。

## 9. 结论

R02 已逐项关闭 R01 的 F001～F010，尤其是修正了预算算术、通知 lock/claim 事实、near-limit 创建、成员详情、A-PERF-02 状态、列表 404 语义、确定性时序、schema 长度和证据树。AM-Q2 事实、A-PERF 状态、真实 PG 路径、独立 oracle、边界与不过度声明均通过复核。

但 R02 仍有两个 HIGH：定时点清单没有覆盖 W03/W04/D04 等 A-PERF-01/04 所需的修改与批量 T1 观测（R02-F001），以及每场景 180s context 与每档 seed/每场景 schema 预算冲突（R02-F002）；另有一个 MEDIUM：候选 `go test` 命令没有落实 timeout、3 次重复/30s 测量窗，且大档正则可能重叠（R02-F003）。在这些问题修正前，R02 不能作为可执行的 90/150 分钟真实 PG 性能补测基线。

`Test Plan Review Gate: CHANGES REQUIRED`