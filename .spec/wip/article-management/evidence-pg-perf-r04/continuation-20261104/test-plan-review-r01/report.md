# S2.4 20261104 addendum 非作者最终有界规划复审 R01

**Test Plan Delta Review Gate: PASS。新增 finding：0 CRITICAL / 0 HIGH / 0 MEDIUM / 0 LOW。**

仅对指定规划增量给出独立结论；不重开原 r04 的全量评审，不给代码、执行、性能或 E2E PASS。

## 对象、身份与授权

- objectType：design（专项测试规划及其机器可读场景映射），不是 code review。
- 固定工作区：`D:/AID/VSCode/Codex/Han`；现场分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。工作树已有大量并行/既有改动，不归本 Reviewer。
- 本会话按用户直接指定的 Reviewer 身份执行，为非作者独立审查；原作者为用户提供的 Beauvoir / add-test-engineer `01a1a48e-993d-75a1-b904-54094384b271`，作者 STOP 来源为本轮交接。未另建 Agent、未下级委派；不虚构本 Reviewer 的运行时 ID。
- 唯一评审目标：相对原 r04 的资源、timeout、场景/schema 计数增量是否自洽，且不丢既有采样、安全和关闭义务。
- 采用依据：用户本轮明确授权建议值；Lead 已采用 small/medium/large/total = 1/5.5/8.5/15 GiB，medium 外层 80m 仅作收尾余量，large/canonical 本轮明确 NOT RUN。以上是已给定决定，不要求人的新审批，也不以旧报告代替对增量的审查。
- 唯一写集：本目录新建 `report.md`、`verdict.md`、只读核证日志 `verification.json`。首次检查目录不存在，创建时使用不允许已存在的 mkdir；所有评审对象、旧证据、任务、计划和源码均未修改。

| 对象 | 现场 bytes | SHA256 |
|---|---:|---|
| `../test-plan-addendum.md` | 15,122 | `2C3F874B43653D4E31C4FD65AA21AAB91CA795386413235E829AD4057B96C774` |
| `../scenario-schema-map.json` | 25,810 | `383215FA893DC9AC0A8CD8777B4B71C22C1BA411DD79DEF864FBC03B4CE95AD7` |
| 原 `test-plan-pg-perf-r04.md` | 76,409 | `EEB026E5FEE5AB0FE207BD9ABC66C3E2F9227EF5112F226286BF5E6EAEB6FC17` |
| 原 `review-test-plan-pg-perf-r04/report.md` | 未以长度固定 | `9D4AC7DB65EFF3DDF6211184B0349B9153C2AD8C0D3371AD615B2FDE642DAC7F` |
| `../author/g0-r01.md` | 未以长度固定 | `AFE5528C62B38EEAD065C05E305887988F23B84D7898CC4B0883453C891A0407` |

addendum 的输入长度 15,094 B 与现场相差 28 B，但 SHA256 完全一致，按指定哈希固定对象；这是交接长度元数据差异，不构成内容变更或 finding。

## 两轮实际检查范围

Round 1：全文读取自身 `.codex/agents/add-reviewer.toml`、`add-code-review/SKILL.md`，执行首次 Route 定位为已有 S2.4 的单项有界规划接力；全文读取 addendum 与 map。依据 `code-review-checklist`、`abstraction-principles` 的适用条款核对象所有权、版本、资源/时限、失败收敛、覆盖和证据边界。go-service 为唯一已启用 profile，读取其 README；本对象不改 Go，不套用编译/代码测试门作为文档 PASS 条件，不加载未启用 profile。补充读取设计写作规则与安全红线，仅按本对象适用性使用。

Round 2：从增量反查原 r04 §4.3～5.5（命令、schema 粒度、资源关闭）、§6.2～9.5（采样、数据/磁盘、停止条件、A/B/C/D 全表、seed 和整体预算、证据）、§10～11（切片关闭与结论）；原 R04 PASS 报告中 F001～F004、八项收敛条件、LOW-1/2 及未验证边界完整上下文；作者 G0 全文，重点 G0-02/03/04/05/08；S2.4 原卡 `tasks.md:1448-1479`。未读取全量大设计。

消费者仅定向查四个 `pg_perf_*_test.go` 的顶层入口、small/medium/large 调度、X01 子 fixture、N02 入口与内部 c4、large gate/canonical/clone、校准入口。代码处于 Backend 并行修改中，观察到的形状和取证时四文件哈希记录于 `verification.json`；不把它们固定为代码审查对象，不判断预期修复已经完成。

核证使用只读 Git branch/HEAD/status、文件哈希/UTF-8 读取、rg 定位、PowerShell 和内存 Python JSON/原表集合运算。未运行 Go、PG、项目静态检查或数据库查询；未读取凭据，未清理/删除/移动，未做 Git 写。工具初次受 PowerShell 管道非 ASCII 文本比较影响，改为按 ASCII `large` 前缀归一化后重新核对；该取证脚本问题未产生评审对象改动，见核证日志。

## 增量核验结果

### 资源与时间

| 项目 | 结果与证据 |
|---|---|
| 空间值 | addendum `:24-34` 与 map `budgets` 一致；1/5.5/8.5 GiB 的精确 bytes 为 1073741824/5905580032/9126805504，和为 16106127360 = 15 GiB。消除旧分档合计 14.5 与总 14 的差异，没有借此宣称足够执行。 |
| 实际磁盘保护 | addendum `:36-40`、`:90` 保留 `pg_database_size` 分档与共同基线累计、跨进程/重试不重置、PGDATA 卷真实余量及原有预留规则；校准计入 small，失败/半成品/噪声不能扣掉。触限停止新点，泄漏/共享实例风险取消在途；不允许清理腾空间或错库创建。原 r04 `:399-408`、`:435-443` 的保护未被 GiB 替换削弱。 |
| 时间保护 | addendum `:34-38` 保留校准+small+medium 共同 90min，large 独立 60min，整体 145min 停新点、150min 取消在途硬停；环境、seed、断言、汇总均计入。分进程/run-id/重试不能重置。medium 80m 不是新增实际执行额度。 |
| 外层 timeout | small45m / medium80m / large70m / calibration90m 与 map 相符。medium 比原最坏约74.8min多约5.2min收尾余量，符合旧 PASS 报告 LOW-2 方向；外层不替代主动预算截止。 |
| seed 算术 | small full5、medium full17、minimal20，X01 展开增加一个 minimal：5×20+17×30+20×5=710s≤720s。校准 K 次建 schema/迁移/计时另占原环境校准5min；原 40+12+15+10+5+3+5=90 不变。估算和真实成本明确分开，不能满足时保留 FAIL/NOT RUN。 |

### 场景集合逐项映射

只读脚本从原 r04 表 A/B/C/D 的每一行抽取 `(tier, logical_scene, mode)`，与 JSON 展开后集合逐项比较；四表 missing/extra 均为空，50 个 `(tier, unit_id)` 全部唯一。

| 原表及位置 | 逻辑场景逐项 | 映射结果 |
|---|---|---|
| A `r04:460-479` | small：W01-typical、W01-near-limit、R01-limit20、R04-10rows、R04-last-page；medium：W02-c1、W02-c16、W03、W04、R01-limit100、R02-continuous、R03-density50/90/100、R05、R06、R07、D01、N01、N02-c2 | 20 行→20 单元，TIMED-3REP，档位/分类无遗漏或重复。 |
| B `r04:485-499` | small：R04-empty、R04-all-hidden、D04-100；medium：E01/E02/E03、D02、D04-1000/10000、N03-1k/10k、N04、X06/X07/X08 | 15 行→15 单元，SINGLE-RUN。D04-100 明确保留，纠正原 §9.2 列举遗漏，不删除表 B 义务。 |
| C `r04:505-510` | small：X01；medium：X02/X03/X04/X05/D03 | 6 行→7 单元。X01/empty 与 X01/ten 各独立 schema；其余各1。 |
| D `r04:518-525` | L00-harness-gate、L01-W01-near-limit、L02-W02-c16、L03-R01-limit100、L04-R02-tail、L05-R03-density100、L06-R07-member-detail、L07-D04-10000 | 8 行→8 单元；同一 canonical100k 源另1，七个 clone 消费者齐全。本轮全部 NOT RUN。 |

真实代码形状证据：`pg_perf_write_test.go:99` 起 small、`:108` 起 medium、`:118` 起 large 与 `:150` gate/`:169` clone；`pg_perf_read_test.go:116` small、`:260` 调用 X01、`:265-300` 两个子 fixture、`:628` medium、`:641` large；`pg_perf_notice_delete_test.go:44-134` 的 small/medium/large 入口。N02 顶层测试短名为 `N02`，计划键为 `N02-c2`；large 内部 scene 短名 L01 等与 map 的业务场景 W01 等并非同一字段。addendum `:64-67` 已要求实际测试名、计划键和 schema 显式关联，不能以字符串短名相等替代运行对账。

N02-c4 在 N02-c2 同 schema 内保留必需正确性子断言，不增加点/schema；W02-c4 为 c16 点内观测，不单独 P95，不增加点；X09 BLOCKED、不建 schema。map 的 `embedded_assertions`、`blocked` 与原表 `r04:512` 及 W02 规则一致。

由此 small=9逻辑/10单元，medium=32逻辑/32单元，A/B/C=41逻辑/42单元。场景全部一次建成且无重试时，本轮条件式总数为 `42+K`；未来完整执行 D 再加8场景+1源，为 `51+K`。K 必须依据校准真实 helper 调用（含可能的自适应调用、失败）计数，不从 count=3/b.N 假定。校准三次正式测量与原始输出仍必需。

旧 43 = 41逻辑+假定1校准+1canonical，不含 D8、未展开 X01，不能作为实物硬指标。addendum `:59-67` 和 map 的 `counts`、`runtime_mapping_required` 用逐单元结果+完整名称去重 inventory+同窗口库内清单核对替代；失败/重试保留、NOT RUN 不虚构 schema、旧 run 不补齐、校准资源归 small 但实物不重复统计。该替换保留义务，并修复了计数语义，不能仅凭恰好数到43宣称完整。

### 采样与关闭义务

addendum `:73-82` 与 map `sampling` 保留每次真实 warm-up≥10s、测量≥30s，TIMED 三重复按 `sum(warmup_i+window_i)≤120s`；SINGLE 一重复≤60s、exploratory，N500 仍为软目标。30s 默认窗口为3×(10+30)=120s，没有少算两次 warm-up。有限 D04/N03/N04 必须在真实活跃窗内执行操作，原完整批及正确性断言不减；新增批/准备/资源都计费，拒绝 completion-only/manual-summary/sleep 补时冒充合规。

addendum `:88-92`、`:102-107` 保留环境/迁移/唯一 schema、资源真实关闭、原始命令 exit/stdout/stderr/样本、独立 Review 和 Lead 同 revision Gate。S2.4 原卡 `tasks.md:1477` 已允许 large 执行或明确 NOT RUN；本轮 D8+canonical 明确原因沿该分支，不构成 small/medium 缺场景豁免，也不把100k解释成容量结论。

## Findings、待决定与非阻断项

本固定规划增量未发现缺陷，故没有需核销的 finding ID。G0-02/03/04/05/08 仅在规划合同层面已明确处理；其实现和执行缺陷是否修复不在本结论范围。

待决定：本增量无新增人的决策/审批要求。采纳值来源已明确。

非阻断信息：交接长度与现场长度不同但哈希一致，已记录；原卡尚有75m、旧43及0.5/14 GB，属于本轮明示交给 Planner 的对账落点，不作为 addendum 未完成的作者缺陷。未检查 Planner 的新 handoff 或 Plan 更新，不宣称已完成登记。

## Planner 精确交回与仍须满足的关闭条件

**本结论足以给 Planner 更新原 `article-management-S2.4`，复用 PGPERF-A～G；无需新增 task 或人的新审批。** 最小落点：

1. `tasks.md:1467` medium 75m→80m；`:1471` 预算→1/5.5/8.5/15 GiB，引用精确 bytes、共同90/60/145/150与累计保护。
2. `:1471` 和 `:1477` 固定“43”→A/B/C41逻辑/42展开单元、校准K实际计数、失败/重试/半成品完整 inventory 与库内清单逐单元对账；D8+canonical本轮NOT RUN及原因。不能简单把43统一替换为42或42+固定1。
3. `:1472` 保留20 TIMED、15 SINGLE、6逻辑正确性及全部采样下限，补 X01 两 fixture、D04-100、每重复 warm-up 的费用和有限工作真实窗口；不降低指标或断言。
4. `:1479` 来源追加本 addendum/map 的固定哈希、本报告与 verdict；Plan 只同步引用/边界。保留本任务 active、既有 requires/DAG 和 A-PERF 状态，不据本规划 PASS 关闭实施或 E2E。

整卡关闭仍需要当前修订四文件的实际实现及适用局部/项目检查、预算内完整 small/medium 所需执行和正确性证据、资源与实际库内 inventory 对账、large/canonical明确NOT RUN原因、docs批准片段校正、非作者代码/执行证据独立评审，以及 Lead 同 revision Gate。发生 FAIL/NOT RUN 必须如实保留，不能以本规划 PASS、采用建议值或单元静态核算代替。

## 未验证风险与复审条件

- 未测真实磁盘余量/累计增长、校准K/耗时、seed710s可达性、90min可完成性、有限工作新增批对资源的影响、真实残留连接/goroutine、OS兼容或当前代码可执行性。规划允许受约束停止，不保证所有点能在本机完成。
- 未验证生产行为、部署/HTTP/SMTP/Journal、容量/SLO、真实 E2E；后续E2E仍关联原S1.4。未把静态检查或 Backend 预期改动当代码 PASS。
- 若本次固定对象任一哈希、预算/采样/覆盖/关闭语义变更，需针对差异复审；如果只是 Planner 采用本结论，需核其实际卡面/Plan diff，而不是重新审旧 r04 全量。实现/运行是另一个固定修订对象，须另行独立审查。
- 本 Reviewer 已完成有界规划报告并交回 Lead/Planner；STOP，不启动下一活动。
