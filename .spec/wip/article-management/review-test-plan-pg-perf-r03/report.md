# article-management 真实 PostgreSQL 容量/性能补测计划 R03 有界复审报告

- 评审对象：`.spec/wip/article-management/test-plan-pg-perf-r03.md`
- 对象固定值（现场复核一致）：**70,678 B / 580 行 / SHA256 `742047FFB18F5E2DFE6BBAA94F0D0120000DA525DB95F40D47B911CB111D2A06` / UTF-8 无 BOM**
- 评审锚点（均未修改）：R01 41,112 B / `307060B6FCF66E33B2E8E57E2AF3D59142A3DCB47F9B499EF2E1C468DB86A56C`；R02 57,753 B / `BEAD1F1C3BBF21723F2DE761203556E559AC109F6E62F2445A16151648129096`
- 上游评审：R01 报告 32,659 B / `B3F50FF6357A9BBBB6FB8E3FAC39C0E0BA6B62CA5FD28E2AFE6097350CE37A50`；R02 报告 23,407 B / `A2C508C2205326440E64485B6696D2686A361D8944A3EA9DE554A100A8C48B86`
- 评审方式：`add-reviewer` 独立非作者**有界复审**（只核销 R02-F001～F005 + 回归 + 新增 finding，不重跑全量评审），不实施、不跑测试、不连 PostgreSQL

**结论行：`Test Plan Review Gate: CHANGES REQUIRED`**

统计：R02-F001～F005 = **4 CLOSED / 1 PARTIAL / 0 OPEN**；本轮新增 **7 项**（HIGH 1 / MEDIUM 2 / LOW 4，其中 LOW-7 为用户已转交的非阻塞命名同步）。

## 1. 实际读取清单

| 材料 | 读取范围 |
|---|---|
| `test-plan-pg-perf-r03.md` | 全文 1–580 行；重点 §0（:1–86）、§2（:127–156）、§3（:157–193）、§4.3（:222–255）、§5（:260–298）、§6.2（:317–332）、§7（:337–361）、§8（:363–381）、§9（:383–569）、§10（:555–571）、§11（:572–580） |
| `test-plan-pg-perf-r02.md` / `r01.md` | 仅校验 size/SHA 未变（锚点） |
| `.spec/wip/article-management/review-test-plan-pg-perf-r02/report.md` | F001～F005 原文（核销基线） |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | `:880–886` A-PERF-01～05 状态表 |
| `.add/project/knowledge/tasks/tasks.md` | `:1441`、`:1444–1448`、`:1455–1458`、`:1466–1467`、`:1474`（S2.4 卡面、r01 命名/预算/SHA） |
| `internal/article/repository.go` | `:736–739`（`lockDueNoticeRecipientSQL`：`LIMIT 1 FOR UPDATE OF recipient SKIP LOCKED`）、`:754–760`（`claimNoticeRecipientSQL`：CAS `UPDATE … RETURNING` preimage） |
| `internal/article/notice_dispatcher.go` | `:461`（`claimDue`）、`:499`（`tx.Commit(work)`） |
| `internal/article/query_service.go` | `:134`（`GetArticle`）、`:176`、`:183`（`GetAuthorArticle`） |
| `internal/article/policy.go` | `:117`（`MemberVisible`）、`:162` |
| `internal/article/command_service.go` | `:21–23`（`maxArticleTitleRunes=200`、`maxArticleBodyRunes=20000`） |
| `migrations/0003_article.sql` | `:128–129`（`articles_created_at_id_idx`）、`:185–186`（`transfer_notice_recipients_claim_idx`） |
| 文件系统 | 4 个新增测试文件均不存在；`audit_insert_rollback_test.go` 存在；`rg` 全仓无 `TestPgPerf*`/`pgPerf*` 符号冲突 |

## 2. 方法

有界复审：①逐条回源核销 R02 五项（含算术重算）；②对 R03 新增的调度表做穷举映射与分项加总；③核 context/schema/命令三处一致性与互斥；④检查是否引入新矛盾（计数、预算、clone/DROP 约束、超时参数量纲）；⑤界外不扩审，不因未改动部分重复给结论。

## 3. R02-F001～F005 核销

| Finding | 判定 | 回源依据 |
|---|---|---|
| **R02-F001** 定时点清单不完整、预算超限 | **PARTIAL** | 主体已修复：§9.4 给出 41 个（档位, 场景点）穷举表（表 A 20 点 / 表 B 15 点 / 表 C 6 场景 / 表 D 8 点含 L00），W03、W04、D04（B7–B9）、R03 三密度（A12–A14）、R04 四数据集（A4、A5、B1、B2）均已归位；算术自洽（见 §4）。**残留**：§3 `:164` 声明 W02「并发 1/4/16…两档独立测量点…两档分别报告」，而表 A 只有 `W02-c1`、`W02-c16`（fixture 写「混合正文分布」），**c4 无任何分类归属**，per-tier 独立点也不存在 → 覆盖声明仍大于测量点（见 R03-F003） |
| **R02-F002** context 与 seed 预算冲突、schema 粒度不一致 | **CLOSED** | §4.3 `:232–240` 按档位固定 300/780/1980s = `seed<120/600/1800 + 180s`，与 §5.5 `:292–296` 一致；每点 TIMED 上限 120s、SINGLE 60s，均 ≤180s 测量余量；`:240` 明确「不得用固定 180s 包住 medium/large seed，也不得为测量另开 context 复用同一 schema」；全文仅剩 `:71`/`:76`（历史记录，已标注被覆盖）与 `:236–238`（作为 +180s 余量的算术组成部分）出现 180s，**无固定 180s 包裹残留**；schema 统一为「每个（档位, 场景点）一个」（`:241`、`:272`、`:284`、`:295`），N02-c4 例外在 `:484` 显式声明 |
| **R02-F003** 命令缺参、模式重叠 | **CLOSED** | `:247–251` 五条命令：`TestPgPerfSmall/Medium/Large` 用 `$` 精确锚定且互斥，`:254` 显式说明 `-run '^TestPgPerf'` 会串跑并禁用；`-timeout` 已显式给出，benchmark 用 `-run '^$'` + `-bench '^BenchmarkPgPerf'` + `-benchtime=30s -count=3`；`:255` 要求把实际 timeout/benchtime/count 写入 fingerprint。原缺陷（默认 10min 超时、缺 benchtime/count、串跑）已消除。（`-timeout` **取值**与新预算不匹配，属新问题，见 R03-F002） |
| **R02-F004** 「30s 窗 + 3 重复必然 ≥500 样本」 | **CLOSED** | `:330` 改为条件句：仅在吞吐 ≥ `500/(3×30s)≈5.56 ops/s` 时才可能达到，否则标 `exploratory (N<500)` 如实记录 N，且不得延长单点超过 120s；`:331` 同步 nearest-rank 与 N 报告口径 |
| **R02-F005** 建议登记与命名 | **CLOSED** | `:557` 明确 `PGPERF-A`～`PGPERF-G` 已登记为 `article-management-S2.4`（`tasks.md:1448`，实测该行为 S2.4 卡标题）的卡内切片；卡面/Plan 的 r01 命名、r01 SHA、`pg_perf_notice_test.go` 与旧预算由 Task-Planner 同步，本角色只读。实测 `tasks.md:1446/1456/1466/1474` 仍为 r01 口径，与 R03 陈述一致（非阻塞，见 R03-F007） |

## 4. 算术复核（逐项重算）

### 4.1 41 个（档位, 场景点）分类穷举

- 表 A（TIMED-3REP）20 = small 5（W01-typical、W01-near-limit、R01-limit20、R04-10rows、R04-last-page）+ medium 15（W02-c1、W02-c16、W03、W04、R01-limit100、R02-continuous、R03-50/90/100、R05、R06、R07、D01、N01、N02-c2）✔
- 表 B（SINGLE-RUN）15 = small 3（R04-empty、R04-all-hidden、D04-100）+ medium 12（E01–E03、D02、D04-1000、D04-10000、N03-1k、N03-10k、N04、X06–X08）✔
- 表 C（CORRECTNESS-ONLY）6 = X01 + X02–X05 + D03 ✔
- 合计 41；`small 9 / medium 32`，与 `:501` 的 seed 计数 `5 full + 17 full + 19 minimal = 41` 完全吻合（small full 5 = W01×2+R01-limit20+R04-last-page+R04-all-hidden；medium full 17 = 表 A 15 + D04-10000 + N03-10k；minimal 19 = small 4 + medium 15）✔
- 31 个场景归属：W01 A1–2；W02 A6–7 + L02（**c4 缺**）；W03 A8；W04 A9；R01 A3/A10 + L03；R02 A11 + L04；R03 A12–14 + L05；R04 A4/A5/B1/B2；R05 A15；R06 A16；R07 A17 + L06；E01–E03 B3–5；D01 A18；D02 B6；D03 C6；D04 B7–9 + L07；N01 A19；N02 A20（+c4 子断言）；N03 B10–11；N04 B12；X01–X05 C1–5；X06–X08 B13–15；X09 BLOCKED。**除 W02-c4 外无遗漏**

### 4.2 small+medium（≤90min）

`20×120s=2400s=40min`（≤24 点/48min 上限内）；`15×60s=900s=15min`；seed `5×20+17×30+19×5=100+510+95=705s=11.75min→12min`；正确性 10 + 环境/校准 5 + 汇总 3 + 缓冲 5。合计 `40+12+15+10+5+3+5 = 90min` ✔（各分项均取上限，无重复计入测量时间）

### 4.3 large（≤60min）与整体

`8×60s=480s=8min`（8 点含 L00-harness-gate，满足「≤8 点、不得加第 9 点」，探索点已占额度）；`8+30+17+5 = 60min` ✔；整体 `90+60=150min`；`:532` 的 145min 停启新点 + 150min 硬停止与两个 5min 缓冲自洽 ✔

### 4.4 分档实际占用（用于判定 R03-F002/F001）

- small 最坏：`600+180+120 = 900s ≈ 15min`（TIMED 600 + SINGLE 180 + seed 120）→ 60m timeout 充足
- medium 最坏：`1800（TIMED 15×120）+720（SINGLE 12×60）+585（seed 17×30+15×5）+600（正确性池）+300（环境）+180（汇总）+300（缓冲）= 4485s ≈ 74.8min` → **超过 `:248` 的 `-timeout=60m`**
- large 最坏：`480+1800+1020+300 = 3600s = 60.0min` → **`:249` 的 `-timeout=60m` 与窗口预算相等，零余量**

### 4.5 磁盘占用（按 §8 单档估算外推 41 个 schema）

- small：5 full（<50MB/档）+ 4 minimal ≈ **<0.3GB**
- medium：17 full × 100–300MB + 15 minimal ≈ **1.8–5.3GB**
- large：7 full × 300MB–1GB（+ 可选 canonical 源 0.3–1GB）≈ **2.1–8.0GB**
- 合计 ≈ **4.2–13.5GB**，且 `:276` 规定不 DROP/TRUNCATE、全部保留 → 与 `:381` 的「DB 增量超过 2 GB 即停止」**直接冲突**（见 R03-F001）

## 5. 本轮 finding

| ID | 严重度 | 位置 | 问题 | 证据 | 建议（最小改法） |
|---|---|---|---|---|---|
| `PGPERF-PLAN-R03-F001` | **HIGH** | R03 `:381`（另涉 `:369–371`、`:273`、`:276`） | **「DB 增量 <2 GB 否则停止」的停规则与 41 schema 设计不相容，按文档自身估算必然触发，计划无法按原文开跑。** 41 个（档位, 场景点）各建一个独立 schema 且全部保留不清理，仅 medium 17 个 full 就有 1.8–5.3GB，large 7 个 100k full 为 2.1–8.0GB，合计 4.2–13.5GB。执行器在 §9.2 第 1 步预算核对时按规则只能停下交 Lead，large 窗口更不可能开始；若执行器忽略该规则，则会突破计划自设的资源边界并影响共享实例。 | `:381`「若预计 DB 增量超过 2 GB…停止并交 Lead 决定是否缩小档位」；`:369–371` 单档规模（small<50MB、medium 100–300MB、large 300MB–1GB）；`:501` 41 schema；`:503` full 5/17；`:276` 全部保留、不 DROP | 把 `:381` 的最后一句替换为按档核算的占用预算与上限，例如：「small ≈<0.3GB；medium 1.8–5.3GB；large 2.1–8.0GB（canonical 源另计 0.3–1GB）；合计 4.2–13.5GB。执行前在 `fingerprint.json` 记录 `pg_database_size` 基线与磁盘余量；预计增量 >**14GB** 或剩余空间 <预计增量×1.2 时停止并交 Lead。」并同步 `:502` 的 inventory 说明。若 Lead 坚持 2GB，则须改为「读场景每档共享 1 个只读 schema」——那是方向选择，本轮不要求 |
| `PGPERF-PLAN-R03-F002` | **MEDIUM** | R03 `:217`、`:247–249`、`:389–412`、`:505–532` | **五条命令的 `-timeout` 取值小于对应档位的自设上限，会在写 NOT RUN/checksums 之前被 Go 杀掉。** medium 最坏 74.8min > `-timeout=60m`；large 窗口预算恰为 60.0min，`-timeout=60m` 不留给编译、进程启动与收尾落盘；`-json` 不改变 timeout 语义。 | `:248`（medium 60m）、`:249`（large 60m）、`:516`（small+medium 90min = 分项上限之和）、`:4485s` 分档最坏值（§4.4）、`:528`（large 60min）、`:532`（145/150min 停点） | 改为 small `-timeout=45m`、medium `-timeout=75m`、large `-timeout=70m`（benchmark 保持 90m），并在 §9.4 补一句：「外层 `-timeout` 必须大于本档窗口预算，用于容纳编译/启动与 NOT RUN、checksums 落盘；145/150min 停点仍由执行器控制」 |
| `PGPERF-PLAN-R03-F003` | **MEDIUM** | R03 `:164`、`:295`、`:438`、`:404` | **W02 的覆盖声明仍大于实际测量点（R02-F001 残留）。** `:164` 写「并发 1/4/16」「两档独立测量点」「两档分别报告」，表 A 只有 `W02-c1`、`W02-c16` 且 fixture 为「混合正文分布」：c4 无任何分类归属，两档也无法「独立/分别」报告。执行者按 `:164` 实现会多跑一个未计入预算的点，按表 A 实现则 §3 的覆盖声明不成立。 | `:164`；`:438`（W02-c16 fixture 混合分布）；`:404`（medium「W02×2」）；`:295`（medium 并发列 1/4/16）；全表无 `W02-c4` | 把 `:164` 的动作改为「W02（medium）排 c1/c16 两个测量点，各一个 schema；正文按典型/near-limit 混合分布，同一测量点内按正文档分别统计分位；c4 不单独排点（并发 4 由 N02-c4 子断言与 X06 覆盖，90min 预算不含此点）」，并把 `:295` 并发列改为「按 §9.4 逐点取值（W02 c1/c16、N02 c2、X06 8/16）」 |
| `PGPERF-PLAN-R03-F004` | **LOW** | R03 `:273`、`:296`、`:381`、`:502` | **large 的 canonical 克隆只是「若采用」的可选项，但 30min 预算只有克隆才可能成立**：§8 称单个 full 100k seed 需 10–30min，表 D 有 7 个 full 100k 点，独立 seed 需 70–210min。另 canonical 源 schema 未计入 inventory，`:273/:502` 写死「41 + 1 = 42」，采用克隆后实际为 43。 | `:371`（10–30 分钟 seed）；`:486–497`（L01–L07 均为 full 100k）；`:296/:381`（「如用 canonical 源」）；`:273/:502`（42 个） | 把 `:296/:381` 的「如用」改为「L01～L07 必须由同一 canonical 100k 源 schema 以 COPY 克隆（含 identity users 与 articles）」，并把 `:273/:502` 计数改为「41 场景 + 1 校准 + 1 canonical 源 = 43」 |
| `PGPERF-PLAN-R03-F005` | **LOW** | R03 `:266`、`:192` | `statement_timeout`/`lock_timeout` 只写「设置会话级」「负例按需设置」，无数值口径。若实现时给出小于 30s 测量窗或小于 X08 锁链时长的值，正向点会被 DB 主动取消，X08 也会在观测到阻塞前退出，且不会体现为「指标不可采集」。 | `:266`；`:192`（X08 要求设 lock/statement timeout）；`:321–326`（测量窗 ≥30s） | 在 `:266` 定值并写入 fingerprint：定时/正向点 `statement_timeout=0`（由 context 与单点上限约束）、`lock_timeout=0`；负例点 `lock_timeout=2s`、`statement_timeout=30s`；X08 单独用 `lock_timeout=10s` 并说明其大于观测轮询上限 |
| `PGPERF-PLAN-R03-F006` | **LOW** | R03 `:34`、`:37` | 两行行尾残留字面 `\n`（`…旧阻塞注记）\n- …`、`…F001～F010）\n- …`），Markdown 渲染会出现一段以 `\n-` 开头的异常文本。 | 实测 `Select-String '\\n'` 命中 `:34`、`:37` 两处 | 把字面 `\n` 换成真实换行（或直接并入上一行） |
| `PGPERF-PLAN-R03-F007` | **LOW** | R03 `:228`、`:557`；`tasks.md:1455–1458`、`:1466–1467`、`:1474`；Plan `:911–917` | S2.4 卡面/Plan 仍为 r01 命名、`han_pgperf_r01_*`、`evidence-pg-perf-r01/`、r01 SHA、旧「单点≤60s/总≤90min」「200 样本」口径与 `pg_perf_notice_test.go` 文件名；R03 已用 r03 全量口径并说明由 Task-Planner 同步。**用户已确认交由 Task-Planner，属非阻塞**。 | R03 `:228`、`:557` 自述；`tasks.md:1466`（r01 前缀/旧预算）、`:1467`（200 样本）、`:1474`（r01 SHA）；`tasks.md:1448` 实测为 S2.4 卡标题 | 无需本文件整改；Task-Planner 同步 r03 文件名、schema 前缀、证据目录、SHA、采样/预算口径即可 |

## 6. 回归检查（未发现其他新矛盾）

- **A-PERF 状态**：R03 `:19`、`:134–137`、`:220`、`:576` 的「01/04 partial、02 planned（目标环境 NOT RUN）、03/05 blocked」与 `test-plan.md:882–886` 逐行一致 ✔；未宣称 SLO/容量/production readiness（`:56–57`、`:395`、`:571`、`:575–577`）✔；明确不重开 S2.2/S2.3、不顶替 S1.5/S8.2/E2E（§1.4）✔
- **固定 180s 残留**：仅 `:71`/`:76`（历史处置表，已标被覆盖）与 `:236–238`（+180s 测量余量）出现，无固定包裹 ✔
- **schema 粒度**：`:241`、`:272`、`:284`、`:295` 一致为「每个（档位, 场景点）一个」，`:484` 声明 N02-c4 例外；`han_pgperf_r03_` 前缀 15 字节 + 档位 1 + case≤6 + `_` 1 + 16hex = ≤39 ≤63 ✔；`:271` 长度断言与 inventory 记录齐备 ✔
- **命令互斥**：三条 `-run` 用 `^…$` 锚定，`-bench` 用 `-run '^$'` 隔离，全仓无 `TestPgPerf*`/`pgPerf*` 同名符号；4 个新增文件当前均不存在，未列入修改既有 8 个测试文件（`:242`）✔
- **代码事实**：`:43` 的 `repository.go:732–760` 与实测 `:736–739`（lock）/:754–760（claim）范围吻合；`:44` 的 `notice_dispatcher.go:461–499`、`:45` 的 `query_service.go:134/183` 与 `policy.go:117–162`、`:46` 的 `command_service.go:23`（`maxArticleBodyRunes=20000`）、`:47` 的 `0003_article.sql:128–129/:185–186` 全部命中 ✔
- **证据结构**：§9.5 的 `fingerprint.json`/`schema-inventory.json`/`seed-manifest.json`/`migrations/sha256.json`/`blocked.md`/`checksums.txt`/`raw/`/`plans/` 与「不写 DSN、只追加不覆盖、AI 复核不冒充人工签字」足以支撑 Lead Gate 与复审 ✔
- **未验证风险**：`seed` 估算（`:503` 705s）未实测，medium full 30s/10k 行与 large 单 schema 1800s 均为规划值；共享开发机噪声、`pg_stat_statements` 可用性、磁盘吞吐未验证；本轮未执行任何命令，未连 PG，结论仅到规划一致性层面

## 7. 结论

**`Test Plan Review Gate: CHANGES REQUIRED`**

R03 已实质修复 R02 的主要缺口（41 点穷举调度、context/schema 统一、命令锚定、500 样本条件化、登记口径），算术与代码事实核查均通过；但 `:381` 的 2GB 停规则与 41 schema 设计的 4.2–13.5GB 占用直接冲突（HIGH），按文档自身规则必然阻断执行，必须先把该停规则改成按档核算的占用预算与上限（约 2–4 行改动）。R03-F002/F003 为两条同级的数值/措辞修正（`-timeout` 取值、W02 c4 与分档统计口径），可一并改；其余 4 项 LOW 不阻断。