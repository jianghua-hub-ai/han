# r04 clock/resource 作者交接

2026-11-04 15:23:02 +08 现场进程查询无可见 Go/test 进程；所有已登记命令均已真实 wait。**SOURCE / LOCAL / PG STOP**，Main 可立即接手窗口。本步骤未装载 DSN、未连接 PG、未创建 schema、未记录真实 Tnew；不声称取得新的 backend=0 探针。以下只是作者实现与自测，独立源码 Review 待 Main 交原 Reviewer。

固定 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。仅改 H：`test/integration/article/pg_perf_harness_test.go`。正式五文件及快照哈希见 [manifest](source-freeze-r04/manifest.json)，H 为 `B8A395DE6F1E50A6AC55DEC9E4D4B3AB87EEA5AD1295C138AAB8D4B9D5B6158E`；W/R/N/docs 四文件保持原 r02 哈希。前像在 `before/`，精确文本差分为 [patch](harness-r02-to-r04.patch)。全部新增实物留在本目录，未清理或覆盖旧材料。

## G0 与实现落点

完整消费已采用的 `test-plan-followup-r04.md` §4.1、Lead `followup-clock-contract-adoption-r04.md` 和 planning Review verdict；Role、实施 Skill、规则及 go-service 必要全文复用本会话已读版本。修改前 `run.py capture` 核原五哈希并保存前像/旧资源摘要；没有把 planning PASS 当实现 PASS。

| 合同 | 当前 H 落点与实物验证 |
|---|---|
| 执行时钟独立，资源原字面不变 | H:1358 policy 解析后得到执行时间和资源强类型；H:1616 harness、H:3011 point 使用已解析 policy。执行 env 按时间点一致，resource origin 逐字等于 `2026-11-04T03:29:04.586115+00:00`；新钟加载原 small 以及等价执行时区用例通过。 |
| legacy 只接受缺字段，不复活旧窗 | H:1358/1440；仅原 revision、无新 attempt/parent/adoption 身份且 resource 字段真正缺失。空/null/部分 resource 字段、新 attempt 伪 legacy、过期/未来执行均拒绝。 |
| 原全库基线与全档绑定 | H:1474/1507/3262/3313；核固定原目录身份及路径各级重解析点、global source 值、small 原 80B/SHA、三档完整声明及实际目录额外项；env 只能一致性校验。真实源只读加载与复制目录、junction、错摘要/长度/Bytes/Tier/origin、遗漏/半成品拒绝均已测。 |
| medium 首次登记及后续复用 | H:3349/3384/3424；O_EXCL 写完整旧 origin ledger 并 close 后再 O_EXCL receipt；部分记录、缺 receipt、receipt 不匹配、未绑定 receipt 均失败，不删除重建。large 无首次登记许可。后续已登记 medium 由文件及 receipt SHA 绑定，当前值增大不重置基线。 |
| 父子显式身份与拒绝 | H:3436/3520；完整 policy/inputs 显式传给独立测试子进程，执行/base/目录 env 去重覆盖陈旧值。子进程真实沿用基线；另两个错配子进程实际 exit1 且命中明确 policy 错误，父测试核验后才通过。 |
| 证据来源 | fingerprint 增加 execution/attempt/resource binding/adoption 来源摘要。receipt 含 Path/Window/Tier/Bytes/FileBytes/SHA256、AttemptID/ExecutionWindow/RegisteredAt、FirstRegistrationAllowed、ParentManifestSHA256。 |

后续 policy JSON 接口：保留 `window_start` 和原限额字段，新增 `attempt_id`、`parent_old_batch`（原 revision）、`adoption_source`（Lead 采用文件绝对路径）及其 `adoption_sha256`；资源字段为 `resource_window_start/resource_base_bytes/resource_baseline_dir/resource_baselines`。每档绑定字段为 `file/state/window/tier/bytes/file_bytes/sha256/allow_first_registration`，已登记 medium 再绑定 `receipt_sha256`；state 为 `existing` 或 `unregistered`。父资源 manifest 摘要采用 Go `json.Marshal(resource_baselines)` 的 SHA256。这里没有创建可执行的新 attempt policy。

## 真实命令与失败保留

完整 argv、前后五哈希、PID、真实出口、耗时及 raw 摘要均在各命令 `result.json`；汇总入口 [command-index-r04.json](command-index-r04.json)，`current_five_hashes=true` 才属于本次最终 freeze。

| 最终版命令 | exit / 实际耗时 / 范围 |
|---|---|
| `race-clock-r04` | 0 / 14.891s；新资源接缝及 medium child，race×3，168 pass 事件、0 fail/skip（含子测试；预期故障 child 的退出另在 raw 中保留）。 |
| `build-core-r02` → `core-race-r03` | 0 / 5.656s；0 / 11.093s。隔离 CWD 运行原 core 选择 race×3，57 pass 事件、0 fail/skip。旧相对 unit-fixtures 写入仅落新 sandbox。 |
| `vet-r02` / `list-r01` / `format-r01` | 均 0；1.625s / 3.484s / 0.109s。gofmt diff 为空。 |

原行为 RED：`red-origin-r01` exit1/3.578s，旧 helper 报 `persistent tier baseline belongs to another window or tier`。`compile-r01` 已可编译、实际失败为 legacy fixture 将缺字段序列化成 null；不是编译 RED。`core-race-r01` 是 PowerShell 参数拆分失败；`race-clock-r02` 是深层 sandbox 超长路径使 mklink fixture 创建失败，未核到 junction 拒绝；均原样保留。后者在正常 Go 包 CWD、同范围新 fixture 上实际补验，最终 race-clock-r04 包含该断言，无 skip。

## 未验与保留边界

`source-freeze-r04/manifest.json` 还验证 `checkDiskBudget`、`pgPerfDatabaseSize`、`pgPerfEnsureDatabase` 的归一 LF 函数文本与原前像相同：SQL size、5s 门及原资源预算未改，否定的 Windows 枚举方案未采用。旧 global base `1406932671`、原 window 5400s 记录/字面/摘要及原 policy/small 摘要全未变；原目录仍只有 small.json，没有真实 medium/receipt 新增。

旧 H 的 Go/PG结果不冒充新 H 通过；旧 W/R/N/docs 源与未改场景语义落点可据精确差分复用，真实 PG 接缝/增长/取消、所有正式校准/small/medium/large/full bench 本步骤均 **NOT RUN**。原批次 NRU 保留，后续 guard budget 合同未在此实施。当前仅核 Windows 本机；未做跨平台/trimpath 构建验证（无法确定绝对来源路径时 fail closed）。源码独立 Review、实现采用和后续 PG 准入仍待 Lead 串行安排；不核销原 finding、不标关闭。
