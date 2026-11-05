# 测试规划独立评审报告（PG 性能补测 r01）

## 1. 评审对象与固定基线

| 项 | 值 |
|---|---|
| 被评审对象 | `.spec/wip/article-management/test-plan-pg-perf-r01.md` |
| 大小 / 行数 | 41,112 B / 385 行（UTF-8 无 BOM） |
| SHA256 | `307060B6FCF66E33B2E8E57E2AF3D59142A3DCB47F9B499EF2E1C468DB86A56C` |
| 作者 | Test-Engineer 角色（Kant） |
| 评审性质 | 独立非作者评审（`add-reviewer`），只读 |
| 工作区 / 分支 / HEAD | `D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 评审日期 | 2026-11-03（Asia/Shanghai） |
| 本次唯一写集 | 本报告 `.spec/wip/article-management/review-test-plan-pg-perf-r01/report.md` |
| 结论 | `Test Plan Review Gate: CHANGES REQUIRED` |

本报告只评审规划文档，不代表测试已执行、指标已采集、A-PERF 已通过或产品已通过。评审期间未运行 `go test` / integration / benchmark，未连接 PostgreSQL，未读取或输出 `HAN_TEST_POSTGRES_DSN` 的值，未修改被评审文件、`tasks.md`、Plan、代码、迁移、测试或其他文档。

## 2. 实际读取清单

以下为本轮实际读取或定向检索的材料；未列出的材料不作为本报告事实依据。

| 材料 | 读取范围 / 方式 |
|---|---|
| `.spec/wip/article-management/test-plan-pg-perf-r01.md` | 全文 1–385 行（显式 UTF-8 读取）；逐行核对场景、预算、证据与边界 |
| `.add/project/knowledge/design-workspace/article-management/design.md` | 186–196 行（AM-Q1/AM-Q2/AM-Q6/AM-Q8）；955–990 行（r29 AM-Q2 参数确认与边界） |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | 876–890 行（§4.9 A-PERF-01～05 状态）；1020–1032 行（§6 映射） |
| `.add/project/knowledge/tasks/tasks.md` | 1415–1445 行（`article-management-S2.2` 尾部与 `article-management-S2.3` 卡面，含关闭状态、阻塞注记、评审记录） |
| `.add/project/knowledge/design-workspace/article-management/stories.md` | 61–71 行、218–225 行（Story 2/3 的 404、列表空页、分页与 ETag 验收） |
| `docs/benchmarks/article.md` | 1–40 行（S2.3 范围与不覆盖项）；180–200 行（A-PERF 状态与 AM-Q2 旧注记） |
| `internal/article/query_service.go` | 1–120 行、111–200 行、200–360 行、755–782 行；函数入口定向检索 |
| `internal/article/command_service.go` | 140–310 行（Moderate/UpdateArticleModeration/audit 同事务）；520–700 行（Create/ReplaceContent 事务与锁）；入口行定向检索 |
| `internal/article/deletion_coordinator.go` | 110–200 行（beginOrLoad 顺序）；280–340 行（LockArticleForDeletion / SetArticleDeletionPending） |
| `internal/article/repository.go` | 600–680 行（notice lock/claim）；700–770 行（notice SQL 与 preimage predicate） |
| `internal/article/notice_dispatcher.go` | 450–500 行（database clock、claimDue 顺序） |
| `internal/audit/store.go` | `Store.Append` 入口行（:50）定向核对 |
| `internal/article/policy.go` | 100–160 行（MemberVisible / MemberListProjection / MemberDetailProjection） |
| `test/integration/article/notice_dispatcher_test.go` | 30–125 行（`noticeHarness`、`noticePoolConfig`、迁移、90s ctx）；函数名定向检索 |
| `test/integration/article/audit_insert_rollback_test.go` | 1–40 行及函数名定向检索（S1.5 现有 audit 回滚覆盖） |
| `test/integration/article/admin_query_test.go` | `newAdminPG` / `newAdminPGPool` / `adminPGMigrate` 入口行定向核对（:60、:99、:114） |
| `test/integration/article/` | 目录文件清单（admin_query、audit_insert_rollback、etag_concurrency、notice_crash、notice_dispatcher、notice_smtp、pagination、postgres_state） |
| `migrations/0003_article.sql` | 126–142 行（articles 索引）；183–190 行（recipient claim 索引） |
| `migrations/` | 0001～0011 文件清单；确认本轮固定 0001～0009，0010/0011 存在但不在本轮 |
| 仓库状态 | `git rev-parse HEAD`、`git status --porcelain`、目标文件 `Get-FileHash` 与长度 |

未读取 `test/performance/article/benchmark_test.go` 全文；S2.3 的“进程内、无 SQL/索引/锁/池”范围依据 `docs/benchmarks/article.md:24-29` 与 S2.3 卡面，不依据规划自述。

## 3. 方法

- **第一轮（对象与上游一致性）**：全文读取被评审文档，逐项核对 AM-Q2 参数、P95 参考基线、A-PERF-01～05 状态、S2.3 关闭事实、`docs/benchmarks/article.md` 旧注记、`article-management-S3.4` 不存在性、ADR-0028 引用、创新文件与写集边界。
- **第二轮（代码、测试设计与可行性）**：沿真实生产代码复核规划声称的路径、SQL、锁、事务、索引与 harness；按 30 个场景检查覆盖、断言、oracle、负例与停止条件；核算 seed / 单点 / 总预算；检查证据结构与不过度声明。
- 只做静态只读核对；不把文档自述当证据；每个 finding 给出可定位行号或代码行号、触发场景、影响与修复建议。
- 严重度按 `add-code-review`：CRITICAL（安全红线/数据不可恢复/大范围不可用）、HIGH（主要行为错误/契约破坏/资源或事务失控，阻断当前任务）、MEDIUM（现实条件下会产生错误、维护风险或测试盲区）、LOW（局部质量问题，不影响正确性）。

## 4. 逐项核查结论

### 4.1 事实一致性

| 核查项 | 结论 | 依据 |
|---|---|---|
| AM-Q2 于 2026-11-02 关闭；`DefaultLimit=20`、`MaxLimit=100`、`CandidateBatchSize=200` | **属实，无 finding** | `design.md:190` 明确“r29 已关闭：用户 2026-11-02 明确确认”，参数与规划 :42/:379 一致 |
| 首期 100000 篇 / 单用户 10000 篇 | **属实，无 finding** | `design.md:972-973`、`:978`；规划 :42/:287/:379 未夸大 |
| P95 列表≤300ms、详情≤150ms，且为开发机参考 | **属实，无 finding** | `design.md:974`、`:980` 明确“开发机测量参考，非目标部署环境结论”；规划 :42/:255/:379 保留该限定 |
| 目标环境真实 P95/并发/数据分布、retention/RPO/RTO 仍 unknown/NOT RUN | **属实，无 finding** | `design.md:980-982`；规划 :47/:380 正确区分 |
| A-PERF-01～05 在 test-plan §4.9 的现状 | **基本一致；A-PERF-02 的第二处引用有误，见 F005** | `test-plan.md:882-886`：01 partial、02 planned、03 blocked、04 partial、05 blocked；规划 :17 与之一致，:180 把 02 写成 blocked |
| §6 映射仍承接 `article-management-S2.3` | **属实，无 finding** | `test-plan.md:1027`；规划 :30/:180 的“S2.3 已关闭”与 `tasks.md:1429` 一致 |
| `docs/benchmarks/article.md` 仍有“AM-Q2 未确认”旧注记 | **属实，无 finding** | `docs/benchmarks/article.md:189-192` 仍写 `blocked（AM-Q2 / 容量目标 / RPO-RTO / retention 未确认）`；规划 :49/:383 将其标为待修正历史文档 |
| 仓库不存在 `article-management-S3.4` | **属实，无 finding** | 全仓检索无该编号；规划 :88/:382 不使用该编号 |
| ADR-0028 存在且允许模拟环境 | **属实，无 finding** | `.add/project/knowledge/decisions/README.md:36` 与 `framework.md:815-816` 登记为已接受；规划 :317 引用成立 |

### 4.2 代码可实施性

| 核查项 | 结论 | 依据 |
|---|---|---|
| `CommandService.Create` → 事务 → `InsertArticle` → commit | **一致，无 finding** | `command_service.go:527-609`（:576 Begin、:593 InsertArticle、:601 Commit） |
| `ReplaceContent` → `LockArticleForUpdate` → 内容 UPDATE → commit；仅 title/body/version/updated_at 变化 | **一致，无 finding** | `command_service.go:626-700`；:680 锁、:696 版本比较；函数注释明确保留治理/归属字段 |
| `Moderate` → 行锁 → `UpdateArticleModeration` → `audit.Store.Append` → commit | **一致，无 finding** | `command_service.go:145-195`、`:197-233`（:202 锁、:227 更新、:230 审计）；`audit/store.go:50` |
| `ListCandidateBatch` 的 `(created_at DESC, id DESC)` keyset；SQL 无可见性过滤，过滤在 Go 侧 | **一致，无 finding** | `query_service.go:222-265`、`:759-780`；`scanVisibleArticles` :316-349；`readCandidateBatch` :351 |
| `GetArticle` 详情路径与作者详情路径 | **存在且可区分；规划只覆盖作者详情，见 F004** | `query_service.go:134`（member detail）、`:183`（author detail）、`:268`（repository GetArticle） |
| `beginOrLoad` → `LockArticleForDeletion` → `InsertDeletionOperation` → `SetArticleDeletionPending` → commit | **一致，无 finding** | `deletion_coordinator.go:125-191`（:135 锁、:177 插入、:182 T1、:187 commit）；`LockArticleForDeletion` :291-317 |
| `ClaimNoticeRecipient` 使用 `FOR UPDATE SKIP LOCKED LIMIT 1` | **不一致，见 F002** | 实际 `ClaimNoticeRecipient` 是 CAS UPDATE…RETURNING（`repository.go:662-671`、`:754-760`）；`SKIP LOCKED` 在 `LockDueNoticeRecipient`（`:611-613`、`:736-739`） |
| notice dispatcher 先 lock 后 claim | **一致，但规划未写清** | `notice_dispatcher.go:461-499`（:471 `LockDueNoticeRecipient`、:485 `ClaimNoticeRecipient`） |
| 索引 `articles_created_at_id_idx (created_at DESC, id DESC)`、recipient claim 索引 | **存在，无 finding** | `migrations/0003_article.sql:128-129`、`:185-186` |
| 新增 `//go:build integration`、同 package、schema 前缀、复用 S2.2 harness 模式 | **原则可落地；新文件名/helper 名与 MaxConns/ctx 未指定，见 F007** | `notice_dispatcher_test.go:1-3`（build tag/package）、`:34-107`；`admin_query_test.go:60/99/114`；迁移 0001～0009 可由单事务执行 |
| 与 S1.5 `audit_insert_rollback_test.go` 写窗、已关闭 S2.2 三文件冲突 | **无直接写集冲突**（新增文件、只读复用）；但规划未列新文件名，Lead 无法据此确认，见 F007 | `test/integration/article/` 现有 8 文件；S1.5 文件在 `audit_insert_rollback_test.go`；规划 :184 只说“建议新增” |
| 迁移 0001～0009 单事务执行 | **可行，无 finding** | 现有 harness 在单事务执行 0001～0009（`notice_dispatcher_test.go:110-124`）；`admin_query_test.go:114` 采用独立 `CREATE SCHEMA` 后迁移；规划 :206 与之一致 |
| 不新增 Go 依赖、不改生产代码/迁移 | **可满足，无 finding** | 规划 :81、`:184`、`:375`；本轮路径均可由现有包与 pgx 完成 |

### 4.3 测试设计质量

| 核查项 | 结论 | 依据 |
|---|---|---|
| 30 个场景计数与分布（W4/R6/E3/D4/N4/X9） | **正确，无 finding** | 规划 :122；表内 W01–W04、R01–R06、E01–E03、D01–D04、N01–N04、X01–X09 齐全；X09 明确不执行 |
| A-PERF-01 创建/修改 near-limit 覆盖 | **创建侧有缺口，见 F003** | `test-plan.md:882` 要求创建/修改接近 200/20000 rune；规划 W01 为小/中正文、W02 未指定正文大小、W04 只改不建 |
| A-PERF-02 列表/候选批/隐藏扫描/P95 覆盖 | **列表与隐藏覆盖较好；详情基线有缺口，见 F004** | R01–R05、X02/X03 覆盖 keyset/候选批/隐藏；R06 只测作者受控详情 |
| A-PERF-03 claim/lease/backlog 覆盖 | **可在真实 PG 观察的部分覆盖，无 finding** | N01–N04 覆盖单 claim、多 worker、1k/10k/100k 积压、lease/重入；SMTP 正确列为 blocked |
| A-PERF-04 T1/pending 覆盖 | **可在真实 PG 观察的部分覆盖，无 finding** | D01/D02/D04 覆盖 T1 两表、并发、批量 pending；Journal/finalizer 正确列为 blocked |
| A-PERF-05 | **不可观察，正确列为 blocked，无 finding** | 规划 :100、X09；未伪造 RTO/备份结论 |
| 独立 SQL oracle、禁止调用被测算法生成期望 | **已正确处理，无 finding** | 规划 :107、`:118`、`:192`；明确禁止 `scanVisibleArticles`/生产 cursor 算法生成期望 |
| 不以 candidate 返回条数代替实际扫描行 | **已正确处理，无 finding** | 规划 :241 明确 `EXPLAIN (ANALYZE, BUFFERS)` 节点 actual rows，且“不以 candidate 返回条数代替实际扫描” |
| 负例与停止条件可执行性 | **总体可执行；列表 404 语义与并发时序有局部问题，见 F006/F008** | 规划 :263-283、`:333-343`；E01 明确禁 sleep，但 D02/N02/N04/X08 未统一 |
| 用流程代替断言 | **未发现系统性缺陷** | 各场景均有行/version/状态/计数断言；性能数字与正确性门分开（:190-192） |
| 用 sleep 猜并发时序 | **仅在 E01 明令禁止，其他并发场景未统一，见 F008** | 规划 :136 vs :140/:144/:146/:154 |
| 用被测算法自己算 oracle | **未发现** | 规划 :118、:192 明确禁止 |

### 4.4 可行性 / 预算

| 核查项 | 结论 | 依据 |
|---|---|---|
| 三档数据规模 1k/10k/100k 篇、100/1k/10k 用户 | **现实，无 finding** | 规划 :289-293；100k × 0.5–4 KiB ≈ 200–400 MB 主体，叠加索引/TOAST/附属表约 300 MB–1 GB，低于 2 GB 上限 |
| 附属数据 10k deletion_operations、50k notices、100k recipients、10k audit | **量级现实，无 finding** | 规划 :295-301；与 100k 文章/10k 用户的比例合理 |
| seed 方式 COPY/批处理 | **足够具体，无 finding** | 规划 :213 明确 pgx `CopyFrom` 或批处理、先 users→articles→operations→notices/recipients、确定性 PRNG、seed 单独计时、`ANALYZE`；批大小留待实现，但 100k 量级不构成可行性障碍 |
| seed 预算 <2 / <10 / <30 分钟 | **量级现实，未实测** | COPY 100k 行 + 100k recipients 通常在分钟级；索引/FK/触发器与共享开发机可能拉长，规划已有 30 分钟停止条件 |
| 单点 ≤60s、总 ≤90min | **自相矛盾且不可行，见 F001** | :251-254 要求每点 ≥30s warm-up + ≥30s 测量 × ≥3 次重复；:231/:340 又设单点≤60s、总≤90min |
| DB 增量 <2 GB | **量级现实，无 finding** | 规划 :229/:303；按 100k 文章与附属表估算低于 2 GB |

### 4.5 边界与不过度声明

| 核查项 | 结论 | 依据 |
|---|---|---|
| 不宣称 SLO/容量/production readiness 达标 | **正确，无 finding** | 规划 :4、:48、:255、:380；明确只记录观测值，不作达标判定 |
| 正确区分 S2.3 进程内基线与真实 PG | **正确，无 finding** | 规划 :113、:169-179；`docs/benchmarks/article.md:22-29` 明确 S2.3 不含 PG/SQL/索引/锁/池 |
| NOT RUN / blocked 项如实标注 | **总体正确；A-PERF-02 状态有一处错误，见 F005** | 规划 :17、:98-100、:381；X09 列出不可用项 |
| 不重开 S2.2/S2.3 | **正确，无 finding** | 规划 :16、:86-87、:178、:382；只复用 harness 事实，不把正确性 PASS 当性能结论 |
| 不顶替 S1.5/S8.2/E2E | **正确，无 finding** | 规划 :88、:173、:382；明确 HTTP/auth/SMTP/部署 E2E 不在本轮 |
| 不暗示修改生产契约/迁移/新增依赖 | **正确，无 finding** | 规划 :81、:207、:337、:375；越界即停止 |
| 不宣称 S2.3 的 ns/op 可推导 PG 容量 | **正确，无 finding** | 规划 :179；明确两套数字不可横向比较 |

### 4.6 证据完整性

| 核查项 | 结论 | 依据 |
|---|---|---|
| 证据目录结构 | **基本完整；缺 seed manifest / migration checksums / blocked 独立落盘，见 F010** | 规划 :349-357 有 fingerprint、raw、plans、schema-inventory、checksums、summary |
| 环境指纹 | **充分，无 finding** | 规划 :259 记录 HEAD、Go、GOMAXPROCS、PG 版本、关键 `pg_settings`、schema、MaxConns、档位、时间 |
| 原始样本 | **充分，无 finding** | 规划 :238、:352 要求 JSONL/CSV 原始样本，记录 N、min/median/max/mean/stddev |
| checksums | **充分，无 finding** | 规划 :355 `checksums.txt`；:206 记录迁移 SHA256 |
| 不得含 DSN/PII | **充分，无 finding** | 规划 :198、:201、:359；明确不打印/不记录 DSN、密码、token、真实 PII |
| 资源关闭证据 | **充分，无 finding** | 规划 :221；`AcquiredConns()==0`、`pg_stat_activity` 无残留后端、goroutine 基线差值 |
| 失败/阻塞证据保留 | **原则正确；落盘位置可再明确，见 F010** | 规划 :208、:338-343、:359；schema/行/证据保留，清理需另行确认 |

## 5. Findings

| ID | 严重度 | 位置（行号） | 问题 | 证据 | 建议 |
|---|---|---|---|---|---|
| `PGPERF-PLAN-R01-F001` | **HIGH** | 规划 `:231`、`:251-254`、`:340`；矩阵 `:225-229`、`:253` | **执行预算自相矛盾，按文档自身规则无法在 90 分钟内完成。** 每个测量点要求 warm-up ≥30s + 测量 ≥30s，即每次重复 ≥60s；每个测量点至少 3 次独立重复，即 ≥180s/点。30 个场景中 W02/R01/R03/D04/N03/X02/X06 等各含 2–3 个并发/档位点，保守按 ≥40 个点计，仅测量就 ≥120 分钟；即使每场景只算 1 个点，30×180s 也已恰为 90 分钟，尚未计 seed、迁移、正确性断言、负例、资源关闭与报告。` :231` 的“单点≤60s”、`:340` 的“单点>60s 或总>90min 即停止”与 `:251-254` 直接冲突；执行到预算处必然截断，无法产出规划承诺的证据。 | 规划 `:251`（≥30s warm-up + ≥30s 测量）、`:252`（≥3 次重复）、`:253`（并发 1/4/16）、`:231`（单点≤60s、总≤90min）、`:340`（停止条件）；场景表 `:126-155` 可数出多个多点场景。算术：3×(30+30)=180s/点；30×180s=5400s=90min（下限）。 | 明确定义“测量点/单次重复/单点”的边界；三选一或组合：减少重复次数（如 2 次）、缩短单次测量、减少并发/档位点；为 small/medium/large 分别给出可执行时间表与总预算；若 90min 是硬上限，把 large 与部分负例拆到另行授权的执行窗口；无法在预算内完成的场景必须标 `NOT RUN`，不得静默截断后仍宣称覆盖。 |
| `PGPERF-PLAN-R01-F002` | **MEDIUM** | 规划 `:71`、`:144`、`:146` | **通知 claim 的代码路径事实错误。** 规划称 `ClaimNoticeRecipient` 使用 `FOR UPDATE SKIP LOCKED LIMIT 1`；实际该方法是以 `noticePreimagePredicate` 为条件的 CAS `UPDATE … RETURNING`，不包含 `FOR UPDATE`/`SKIP LOCKED`。`SKIP LOCKED` 位于 `LockDueNoticeRecipient` 的 `lockDueNoticeRecipientSQL`，由 dispatcher 在 claim 之前调用。N02 的“SKIP LOCKED、无重复 claim”若按规划文字实现，会把锁机制与 CAS fencing 混为一谈，可能漏测 preimage 冲突路径。 | `internal/article/repository.go:662-671`（ClaimNoticeRecipient 调用 `updateNoticeRow(ctx,tx,claimNoticeRecipientSQL,…)`）、`:754-760`（CAS UPDATE … preimage … RETURNING）、`:611-613`（LockDueNoticeRecipient）、`:736-739`（`LIMIT 1 FOR UPDATE OF recipient SKIP LOCKED`）；`internal/article/notice_dispatcher.go:471`（lock）→`:485`（claim）。 | 修正 `:71` 的路径描述；把 N02 拆成两条断言：①`LockDueNoticeRecipient` 在多 worker 下使用 `SKIP LOCKED`，无同一行重复锁；②`ClaimNoticeRecipient` 的 preimage/CAS fencing 在并发/旧镜像下恰一成功。N04 继续覆盖 lease/preimage 守卫。 |
| `PGPERF-PLAN-R01-F003` | **MEDIUM** | 规划 `:96`、`:126-129`、`:291-293` | **A-PERF-01 的“创建 near-limit”覆盖缺口。** A-PERF-01 要求“合法内容接近 200/20000 rune 时的创建/修改吞吐、资源与延迟”；规划 W01 只创建小/中正文，W02 并发创建但未指定正文大小，W04 只覆盖 near-limit 修改，没有任何场景测量 near-limit 创建。数据表虽写 1%/0.1% 接近最大正文，但未映射到 W01/W02，实施者可能全部用小正文 seed。 | `test-plan.md:882`（A-PERF-01 创建/修改 near-limit）；规划 `:96`（声称覆盖创建/修改 near-limit）、`:126`（W01 小/中正文）、`:127`（W02 未指定正文大小）、`:129`（W04 只改）、`:291-293`（近最大正文比例未映射到创建场景）。 | 新增 W05 或在 W01/W02 中明确一个 near-limit 创建点：3 字节 UTF-8、接近 20,000 rune（约 60 KB），记录实际 rune/bytes、TOAST/行大小、插入与提交延迟；`262144` wire 仍按现边界不在 PG 层断言。 |
| `PGPERF-PLAN-R01-F004` | **MEDIUM** | 规划 `:42`、`:97`、`:135`、`:324` | **A-PERF-02 的“文章详情≤150ms”基线缺少成员详情测量。** 规划唯一的详情场景 R06 是“作者读取自己的受控详情”，走 `GetAuthorArticle`/AuthorProjection；而 P95 基线中的“文章详情”对应成员详情 `GetArticle`/MemberDetailProjection。两者可见性、投影和 ETag 路径不同，作者详情不能直接等价为成员详情基线；若只测 R06，A-PERF-02 的详情 P95 对照存在盲区。 | `design.md:974`（成员文章列表≤300ms；文章详情≤150ms）；`test-plan.md:883`；规划 `:42`、`:97`（引用详情≤150ms）、`:135`（R06 作者受控详情）、`:324`（small 基线写“作者详情”）；`internal/article/query_service.go:134`（member GetArticle）与 `:183`（GetAuthorArticle）。 | 增加一个成员详情测量点（复用同一 fixture，记录 `GetArticle` + 身份状态读取 + 投影延迟），或在 R06 中明确说明“作者详情作为详情上界”的理由，并分别报告两条路径的数字；不要把作者详情数字直接标成成员详情基线。 |
| `PGPERF-PLAN-R01-F005` | **MEDIUM** | 规划 `:180`（与 `:17`、`:97`、`:381` 对照） | **A-PERF-02 状态被写成 blocked，与当前权威 test-plan 不一致。** 规划 `:180` 说“A-PERF-01/04 partial 与 A-PERF-02/03/05 blocked 注记仍按其卡面保留”；但 `test-plan.md:883` 已把 A-PERF-02 标为 `planned`（AM-Q2 已关闭，目标环境 NOT RUN），规划 `:17` 也写“A-PERF-02 目标环境 NOT RUN（AM-Q2 已关闭）”，`:97` 与 `:381` 同样不称其 blocked。`:180` 的依据是 S2.3 卡面 `tasks.md:1441` 的旧阻塞注记（AM-Q2 未确认），该注记在 AM-Q2 关闭后已不能作为当前状态。 | `test-plan.md:882-886`（A-PERF-02 planned）；`tasks.md:1429`（S2.3 已关闭）、`:1441`（AM-Q2 等未确认的旧阻塞注记）；规划 `:17`、`:97`、`:180`、`:381`。 | 将 `:180` 改为“A-PERF-01/04 partial；A-PERF-03/05 blocked；A-PERF-02 planned/NOT RUN（AM-Q2 已关闭，目标环境未测）”；如要保留 S2.3 卡面历史，注明“卡面注记为 AM-Q2 关闭前的历史，当前状态以 test-plan §4.9 为准”。 |
| `PGPERF-PLAN-R01-F006` | LOW | 规划 `:107`、`:132`、`:149`、`:267` | **列表场景混入 404 语义，可能产生空断言。** 成员列表对隐藏/全隐藏文章返回空页（`items=[]`、`next_cursor=null`），不会返回 404；404 是详情路径（不存在、pending、永久删除、隐藏）的语义。R03 写“404/空页”、X03 写“无 404 泄露”、负例表写“统一 404 不泄露原因”，在列表场景中“无 404”几乎恒真，不能证明隐藏项未泄露；真正应断言的是 items 不含隐藏项、空页与 next_cursor 正确。 | `stories.md:70`（详情统一 404）、`:224`（空列表 `items=[]/next_cursor=null`）；`query_service.go:316-349`（scanVisibleArticles 过滤后构图）；`policy.go:117-142`（MemberVisible/MemberListProjection）；规划 `:107`、`:132`、`:149`、`:267`。 | 列表场景统一改为“items 不含隐藏项、空页、next_cursor=null、扫描耗尽、候选批次/扫描行符合 oracle”；404 断言移到详情场景（R06 及详情负例），并保留“不泄露隐藏原因”的检查（响应体/错误码/字段）。 |
| `PGPERF-PLAN-R01-F007` | LOW | 规划 `:184-188`、`:205`、`:221` | **harness 复用与新文件/helper 命名未落到可执行边界。** `noticePoolConfig` 硬编码 `MaxConns=4`，`newNoticeHarness` 使用 90s ctx；环境矩阵要求 medium/large `MaxConns=8/16`、单点可到 60s，直接复用会冲突。规划又说“复用需要修改现有文件就停止”，但没有说明应在新文件中新建可配置 pool/harness，也没有列出新文件名/helper 名；同 package 下若复制 `noticePoolConfig` 等符号会重复定义，且无法据此向 Lead 确认与 S1.5 `audit_insert_rollback_test.go` 写窗、S2.2 三文件无冲突。 | `notice_dispatcher_test.go:70-85`（MaxConns=4）、`:87-107`（90s ctx、package/harness）、`:110-124`（迁移）；`admin_query_test.go:60/99/114`（可复用 helper）；`test/integration/article/` 现有 8 文件；规划 `:184`（只说“建议新增”）、`:186`（候选复用/停止）、`:205`（schema 前缀）。 | 明确新增文件名（如 `pgperf_read_test.go`、`pgperf_write_test.go`、`pgperf_notice_test.go` 或单一 `pgperf_test.go`）与唯一 helper 名（如 `pgperfPoolConfig`、`pgperfHarness`）；在新文件内实现可配置 `MaxConns` 与 per-scenario context，只复用 DSN/loopback/版本校验思路；向 Lead 确认 S1.5 写窗与 S2.2 文件不被触碰。 |
| `PGPERF-PLAN-R01-F008` | LOW | 规划 `:136`、`:140`、`:144`、`:146`、`:154` | **并发时序纪律只在 E01 明确禁止 sleep。** E01 写“不得用 sleep 猜时序”，但 D02（双 DELETE/UPDATE）、N02（多 worker claim）、N04（lease 到期、旧 worker 晚到）、X08（三连接锁链）没有规定确定性方法。N04 若用 sleep 等 lease 到期会不稳定；应通过 SQL 显式把 `locked_until`/`next_attempt_at` 置为过去或使用可控 DB 时间，其他并发场景应用事务持锁 + barrier/channel，而不是猜测调度。 | 规划 `:136`（仅 E01 禁 sleep）、`:140`（D02）、`:144`（N02）、`:146`（N04 “真实 DB 时钟”但未给方法）、`:154`（X08）；`repository.go:732-734`（due predicate 使用 `clock_timestamp()`）、`:748-752`（preimage 含 locked_until/next_attempt_at）。 | 统一并发纪律：N04 用 SQL 把 `locked_until`/`next_attempt_at` 设为明确过去值（或注入 DB 时间），再让新 worker claim；D02/N02/X08 用显式事务持锁 + WaitGroup/channel/`pg_blocking_pids` 有界轮询；在所有并发场景写明“禁止用 sleep 推断时序”。 |
| `PGPERF-PLAN-R01-F009` | LOW | 规划 `:205` | **schema 标识符长度未约束。** `han_pgperf_r01_<case>_<32hex>` 的前缀 15 字节 + `<case>` + 1 + 32 字节 = 48 + len(case)；PostgreSQL 标识符上限 63 字节，`<case>` 超过 15 字节会被截断，截断发生在末尾 32 位 hex 上，可能造成 schema 名碰撞。 | 规划 `:205`；PostgreSQL 标识符 63 字节限制；现有 `han_s72_`（8+32）、`han_s15audit_`（12+32）等均短于上限。 | 限制 `<case>` 为 ≤15 个 ASCII 字节或改用短数字码（如 `w01`、`r03`、`n04`）；`CREATE SCHEMA` 前断言 `length(schema)<=63`，并在 schema 清单记录完整名。 |
| `PGPERF-PLAN-R01-F010` | LOW | 规划 `:345-359`（并对照 `:206`、`:213`、`:259`） | **证据结构缺少 seed 参数、迁移校验和与 blocked 清单的明确落盘位置。** 目录树只有 `environment/fingerprint.json`、`raw/`、`plans/`、`schema-inventory.json`、`checksums.txt`、`summary.md`；`:206` 要求记录迁移 SHA256、`:213` 要求 seed 单独计时、`:259` 要求环境指纹，但没有 `seed/manifest.json`、`migrations/checksums.txt`、`blocked.md` 等可独立核对的载体。Lead Gate/复审只能从 summary 间接推断 seed 条件与 blocked 项，不足以复现实验。 | 规划 `:206`（迁移 SHA256）、`:213`（CopyFrom/批处理、seed 计时）、`:259`（环境指纹）、`:349-357`（证据树）、`:359`（禁止 DSN/PII）。 | 证据树增加：`seed/manifest.json`（确定性 seed、各表行数、正文/隐藏分布、批大小、ANALYZE 时间）、`migrations/checksums.txt`（0001～0009 SHA256）、`blocked.md` 或 `blocked.json`（X09 与不可采集指标）；`fingerprint.json` 纳入 schema 名、MaxConns、档位、并发。 |

## 6. 待 Lead 决定与非阻断建议

**需要 Lead 决定（不是本评审可代替的授权）**

- **F001 的预算取舍**：90 分钟是硬上限，还是可拆分为多个授权执行窗口？若维持硬上限，需缩小场景/并发点/重复次数，或在规划中明确“预算内只跑 small/medium，large 另批”。
- **F004 的详情基线口径**：是否把成员详情加入本轮；若不加，是否接受 R06 作者详情作为“详情上界”并在报告中明确不可直接对照 ≤150ms 成员详情基线。
- **F007 的写集/命名**：是否由本规划直接指定新文件名与 helper 名，并由 Lead 确认 S1.5 写窗、S2.2 三文件与其他写者不冲突。
- **PGPERF-A～G 的登记**：规划 §10 明确“建议、未登记”，是否授权 task-planner 登记、由谁执行、证据根放哪里。
- **PGPERF-G 文档修正**：`docs/benchmarks/article.md:189-192` 的 AM-Q2 旧注记与 `test-plan.md` 状态行是否由文档 owner 修正；本评审不代替该授权。

**非阻断建议**

- F006/F008/F009/F010 属 LOW，可由作者在下一修订一并修正，或由 Lead 显式接受；不影响本轮“先补真实 PG 观测”的方向。
- D03 与 S1.5 `audit_insert_rollback_test.go` 的审计回滚断言有部分重叠；D03 增加了 `xact_rollback`/回滚延迟/锁释放观测，属于性能补测而非纯重复。建议在 D03 中引用 S1.5 已有证据，避免重复搭建同一故障注入装置。
- N03 的 fake mailer 与 `noticeHarness` 的 `noticeSendFunc` 思路可复用；但 100k recipient 积压需要独立 harness（见 F007），不要直接用 `noticeHarness` 的 90s ctx。
- 规划 §5.3 的 seed 顺序（users→articles→deletion_operations→notices/recipients）与真实 FK 依赖一致，建议在证据 manifest 中固化该顺序与每步行数，便于复审重放。

## 7. 未验证风险

- **未执行任何动态验证**：本评审未运行 `go test`、integration、benchmark，未连接 PostgreSQL；规划的 SQL 计划、锁行为、缓冲区、连接池、seed 时间与 DB 增量均未实测。
- **PG 权限/扩展未知**：`EXPLAIN (ANALYZE, BUFFERS)`、`pg_stat_activity`、`pg_stat_statements`、`pg_stat_wal` 是否可用未验证；规划已要求缺失时标 unavailable，但实际可采集范围未知。
- **DSN 未核值**：按授权未读取 `HAN_TEST_POSTGRES_DSN` 的值；仅依据现有 helper 的空值 Fatal、loopback:5433 与 `server_version_num=180006` 校验判断可落地，未确认当前环境实际可达。
- **共享开发机噪声**：其他 schema、其他会话、autovacuum/checkpoint/OS cache 会影响数字；规划已披露不可横比，但本轮没有基线实测。
- **写者冲突**：工作树存在其他角色 modified/untracked 文件；本评审未触碰。S1.5 写窗与新增文件的符号冲突未实际编译验证，需 Lead 在授权执行前确认。
- **迁移范围**：0001～0009 是否足以支撑全部场景未逐条执行；0010/0011 被排除，若某 fixture 实际需要它们，按规划 :207 应停止并报告。
- **large seed 可行性**：100k 文章 + 100k recipients + 50k notices 的 COPY/索引/FK 时间未实测；30 分钟与 2 GB 是估算边界，不是已验证结论。
- **A-PERF-02 详情口径**：若 Lead 不补成员详情，作者详情数字与成员详情 ≤150ms 基线不可直接比较（F004）。

## 8. 复审条件

- 以**新修订**（新 SHA256/字节数/行号）提交复审；本报告只对应上述 `307060B6…A56C` 修订，不混用后续版本下结论。
- F001～F005 必须逐项关闭：F001 给出可执行的时间表与停止/截断规则；F002 修正 claim 路径并拆分断言；F003 增加 near-limit 创建点；F004 增加成员详情或明确上界口径；F005 修正 A-PERF-02 状态。
- F006～F010 可修复或由 Lead 显式接受；复审时核对是否引入新的写集/范围扩大。
- 复审将重新核对：预算算术、`ClaimNoticeRecipient`/`LockDueNoticeRecipient` 代码路径、W01/W02/W04 正文大小、R06 与成员详情路径、test-plan §4.9 状态、schema 名长度、证据树与 seed manifest。
- 复审仍只做静态只读；若后续授权真实 PG 执行，执行证据与规划评审分开，不以本报告代替执行 Gate。

## 9. 结论

规划在 AM-Q2 事实、A-PERF 边界、真实 PG 路径、独立 SQL oracle、不过度声明与证据纪律上整体扎实；但执行预算按文档自身规则无法在 90 分钟内完成（F001，HIGH），且存在通知 claim 路径事实错误（F002）、near-limit 创建覆盖缺口（F003）、成员详情 P95 基线缺口（F004）与 A-PERF-02 状态误写（F005）。在这些问题关闭前，本规划不能作为可执行的真实 PG 性能补测基线。

`Test Plan Review Gate: CHANGES REQUIRED`