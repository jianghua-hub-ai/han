# article-management 真实 PostgreSQL 容量/性能补测计划 R04 最终有界复审报告

- 评审对象：`.spec/wip/article-management/test-plan-pg-perf-r04.md`
- 对象固定值（现场复核）：**76,409 B / SHA256 `EEB026E5FEE5AB0FE207BD9ABC66C3E2F9227EF5112F226286BF5E6EAEB6FC17` / UTF-8 无 BOM**；行数按 `ReadAllLines` 计 608（末行含换行时部分工具计 609，属计数口径差异，不影响对象一致性）
- 锚点（均未修改）：R01 41,112 B / `307060B6FCF66E33B2E8E57E2AF3D59142A3DCB47F9B499EF2E1C468DB86A56C`；R02 57,753 B / `BEAD1F1C3BBF21723F2DE761203556E559AC109F6E62F2445A16151648129096`；R03 70,678 B / `742047FFB18F5E2DFE6BBAA94F0D0120000DA525DB95F40D47B911CB111D2A06`
- 上游评审：R03 报告 17,487 B / `AAB00E32A9123954E052743E3355217B65BF09BCAAB61DDD3FC1A76778B34E15`（`CHANGES REQUIRED`，F001～F007）
- 评审方式：`add-reviewer` 独立非作者**最终有界复审**——只核 R03-F001～F007 核销、用户列出的 8 项收敛条件与「无新 HIGH」，不展开全量评审；不实施、不跑测试、不连 PostgreSQL

**结论行：`Test Plan Review Gate: PASS`**

统计：R03-F001～F007 = **7 CLOSED / 0 PARTIAL / 0 OPEN**；本轮新增 **0 HIGH / 0 MEDIUM 阻断项 / 4 条非阻断执行期注意事项（LOW）**。

## 1. 实际读取清单

| 材料 | 读取范围 |
|---|---|
| `test-plan-pg-perf-r04.md` | §0.4 处置表 `:88–101`；§3 `:175–180`、X08 `:207`；§4.3 `:240–272`；§5.1 `:280–283`；§5.2 `:285–292`；§5.5 `:308–314`；§6.2 `:344–347`；§8 `:385–408`；§9.2–§9.4 `:428–560`；§9.5 `:560–580`；§10–§11 `:583–608`；并以 `Select-String` 全量检索 `GB`、`-timeout`、`W02`/`c4`、`canonical`/`克隆`、`43/42/41`、`statement_timeout`/`lock_timeout`、`145/150min`、字面 `\n`、`production readiness`/`SLO`/`达标`、`2 GB` |
| r01/r02/r03 | 仅校验 size/SHA 未变（锚点） |
| `review-test-plan-pg-perf-r03/report.md` | F001～F007 原文（核销基线） |

## 2. R03-F001～F007 核销

| Finding | 判定 | 回源依据 |
|---|---|---|
| **R03-F001**（HIGH）「DB 增量 >2 GB 即停止」与 41 schema 设计冲突 | **CLOSED** | 全文检索 `2 GB`/`2GB` 命中 0；`:399–408` 新增「保留 schema 的磁盘占用上限（替代旧的固定总增量停规则）」并按档给上限 small ≤0.5 GB、medium ≤5.5 GB、large（含克隆与源）≤8.5 GB、总体 ≤14 GB；`:408` 停止条件为「任一档累计 `pg_database_size` 增量超该档上限，或当前 DB 大小 + 预留超过实测磁盘余量 → 停止启动新点、保留证据、交 Lead，不得 DROP/清理」；与估算自洽：small <0.3 ≤0.5、medium 1.8–5.3 ≤5.5、large 2.4–8.0 ≤8.5 ✔；`:568` fingerprint 明确记录 `pg_database_size` 基线、磁盘余量、分档上限（0.5/5.5/8.5，总体 14）；`:292`、`:399`、`:441` 均保留「不清理、不 DROP、不 TRUNCATE」✔；`:408` 声明上限非生产容量目标 ✔ |
| **R03-F002**（MEDIUM）`-timeout` 小于档位最坏窗口 | **CLOSED** | `:262–265` 为 45m / 75m / 70m / 90m；覆盖：small 最坏 ≈15min <45m ✔、medium 最坏 4485s≈74.8min <75m ✔、large 最坏 3600s=60.0min <70m ✔、benchmark 90s 量级 ≪90m ✔；`:270` 明确「外层 timeout 必须大于窗口预算，用于编译/启动、NOT RUN 与 checksums 落盘，145/150min 停点仍由执行器控制」✔；`:271` 要求实际值写入 `fingerprint.json`/`summary.md` ✔ |
| **R03-F003**（MEDIUM）W02 c4 无归属、两档不可分别报告 | **CLOSED** | `:179` 改为「并发 1/16 两个测量点（c1、c16），各一个 schema；正文混合分布，同一测量点内按两档正文分别统计分位；c4 不设独立测量点，仅在 c16 点内作中间并发观测，不计点数、不单独报告 P95」；表 A 实测 20 行且仅含 `W02-c1`/`W02-c16`（已逐行枚举核对，无 c4 行）；`:466–467` fixture 写「典型+near-limit 混合，点内分别统计」；`:311`、`:345` 并发口径同步为「按 §9.4 逐点（W02 c1/c16、N02 c2、X06 8/16、L02 c16）」「4 只出现在 N02-c4 正确性子断言」；`:546` 仍为 20 点/41 场景 schema ✔ |
| **R03-F004**（LOW）canonical 克隆为可选、schema 计数 42 | **CLOSED** | `:312`、`:397` 改为**强制**「L01～L07 必须由同一 canonical 100k 源 schema（含 identity users 与 articles）以 `CREATE TABLE … AS`/`COPY` 克隆到各点 schema，源 schema 只建一次」；计数统一为 43：`:289`、`:530`、`:546`、`:568–569` 均为「41 场景 + 1 校准 + 1 canonical 源 = 43」✔；`:553` large 预算分项同步为「canonical 源 1 次建立 + 7 次克隆 + ANALYZE ≤30min」✔ |
| **R03-F005**（LOW）`statement_timeout`/`lock_timeout` 无数值 | **CLOSED** | `:282` 固定：测量点 `statement_timeout=30s`；seed/克隆 `statement_timeout=0`（由 seed 预算与 context 兜底）；负例 `lock_timeout=5s`；所有会话默认 `lock_timeout=0`；实际值写入 fingerprint、不改服务器全局 ✔；`:207` X08 同步为 `lock_timeout=5s`、`statement_timeout=30s` ✔；`:271`、`:568` 纳入 fingerprint ✔ |
| **R03-F006**（LOW）字面 `\n` 残留 | **CLOSED** | 全文检索字面 `\n` 命中 0（原 `:34`、`:37` 已为正常换行）✔ |
| **R03-F007**（LOW）S2.4 卡面/Plan r01 命名 | **CLOSED（非阻塞保留说明）** | `:585` 保留「S2.4 卡面与 Plan 仍引用 r01 命名、r01 SHA、`pg_perf_notice_test.go` 与旧预算/200 样本；由 Task-Planner 同步为 r04，**非阻塞**」；`:606` 再次标注非阻塞；`:243` 对文件名差异说明为「卡面允许的等价收敛」✔。实测 `tasks.md:1448` 仍为 S2.4 卡标题，与文档陈述一致；命名/卡面同步按用户指令不作为阻断项 |

## 3. 用户列出的 8 项收敛条件

| # | 条件 | 结论 | 依据 |
|---:|---|---|---|
| 1 | 旧「>2 GB」消失；0.5/5.5/8.5/总 14 GB 自洽；fingerprint 记基线/余量/上限；schema 不清理保留 | **满足** | `:399–408`、`:568`、`:292`、`:441`（详见 F001 核销）；分档上限均 ≥ 对应估算上限 |
| 2 | 45m/75m/70m/90m 覆盖各档最坏窗口 | **满足** | `:262–265`、`:270`（small 15 <45；medium 74.8 <75；large 60.0 <70） |
| 3 | W02 只剩 c1/c16 两个 TIMED-3REP，c4 降为点内中间观测，点数仍 20 | **满足** | `:179`、表 A 逐行枚举（20 行、无 c4）、`:546` |
| 4 | canonical 克隆改为强制，schema 计数统一 43（41+1+1） | **满足** | `:312`、`:397`、`:289`、`:530`、`:546`、`:568–569` |
| 5 | 测量 `statement_timeout=30s`、seed/克隆 0、负例 `lock_timeout=5s` 写明 | **满足** | `:282`、`:207`、`:271` |
| 6 | 字面 `\n` 清除 | **满足** | 全量检索命中 0 |
| 7 | 非阻塞说明保留 | **满足** | `:585`、`:606`、`:243` |
| 8 | 预算仍 90+60=150min、145min 停点/150min 硬停；无新过度声明 | **满足** | `:537–544`（40+12+15+10+5+3+5=90）、`:552–556`（8+30+17+5=60）、`:560`（150/145/150 硬停）；`:61`、`:140`、`:422`、`:599`、`:602–606` 均明确不宣称 SLO/容量/production readiness，A-PERF 状态维持 01/04 partial、02 planned（目标环境 NOT RUN）、03/05 blocked |

## 4. 非阻断执行期注意事项（LOW，不影响 PASS）

1. **分档上限之和 14.5 GB 略大于总上限 14 GB**（0.5+5.5+8.5）：当 small+medium 用满时 large 实际可用余额仅 8.0 GB，与其估算上限相等、无余量。执行时以总上限 14 GB 为准并在 fingerprint 记录「total_cap − 已用增量」余额即可；如需严格分档独立，可把总上限写成 14.5 GB（纯数值，不影响覆盖与预算表）。
2. **medium `-timeout=75m` 相对 74.8min 最坏值仅约 12s 余量**：冷编译/首次构建可能吃满这点余量。文档 `:270` 已要求执行器主动收尾；若预期冷构建，建议临时上调到 80m 并在 fingerprint 记录实际值。
3. **X08 的 `lock_timeout=5s` 把三连接锁链的有效观测窗压缩到 ≤5s**：轮询间隔需明显小于 5s（建议 ≤250–500ms）才能抓到 C 排队形态；否则只能观察到 B 单点阻塞，链形断言不成立。属实现细节，不构成规划缺陷。
4. **测量点 `statement_timeout=30s` 与单点上限 120s 并存**：若某单条语句因数据异常被 DB 主动取消，应归入 timeout 类错误并在 `summary.md` 区分「数据库主动取消」与「性能失败」，避免误读为延迟样本。现有 `:307`（错误分类）已能承载，无需改文。

另：S2.4 卡面/Plan 的 r01→r04 同步仍由 Task-Planner 执行（`tasks.md:1446/1456/1466/1467/1474` 尚未更新），按用户决定属非阻断，不计入本轮 finding。

## 5. 未验证风险（边界）

- 本轮为文档复审：未执行任何测试/基准、未连接 PostgreSQL、未读取 DSN，未核实 seed 实测耗时、真实磁盘增量、`pg_stat_statements` 可用性与共享实例噪声。
- 文档中的所有时间/空间数字仍是规划上限而非实测值；执行结果如何、能否作为后续基线，须由 Lead 授权执行并另行独立评审。

## 6. 结论

**`Test Plan Review Gate: PASS`**

R03-F001～F007 全部 CLOSED，用户列出的 8 项收敛条件全部满足，未发现新的 HIGH 或阻断级 MEDIUM；预算硬线（90+60=150min、145min 停点、150min 硬停）与「不宣称 SLO/容量/production readiness」边界保持完整。4 条 LOW 级注意事项属执行期细节，可在 PGPERF-A 实施时顺手落实，不阻断 S2.4 开工。