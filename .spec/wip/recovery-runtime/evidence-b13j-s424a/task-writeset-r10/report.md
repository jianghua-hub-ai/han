# S4.24a 写集对账报告（F017，evidence r10）

- 日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 角色：`add-task-planner`（S4.24a 写集对账，不实施代码）
- 对象：`.add/project/knowledge/tasks/tasks.md` 的 `identity-auth-S4.24a` 卡面

## 1. 输入事实（本轮复核）

| 对象 | 事实 | 本轮复核 |
|---|---|---|
| `design.md` | 1,103,152 B／5,476 行／`3A8A9E636628E998216410554303AED5D4472780AF00DE7B814E21D7204D5254`，`identity-auth-design-ir01-r70` | 实测同值 |
| `review-arch-r10/report.md` | 16,624 B／`3E6639A5B566E3FAC644D4368F4CC33526163EB63DFF0F49343978413A30455A`／PASS（0 BLOCKER／0 MEDIUM／0 LOW） | 实测同值 |
| `review-sec-r10/report.md` | 23,423 B／`4D8625127035B84EEAF595BAFABC2666B9BFA64382FE43B45B1FF0D4DB9EF094`／PASS（0 BLOCKER／0 MEDIUM／0 LOW） | 实测同值 |
| `design-revision-r10/report.md` | 13,214 B／`724130C867DA7A134DF964C033E731F118F75CCD100518399F2AB81BD52D98EE` | 实测同值 |
| `design-revision-r10/hashes.txt` | 4,729 B／`6635624ADDBEA8458206E6500128EE38198FEAA700C6BDA40258E3E96D16BB36` | 实测同值 |
| `design-revision-r10/design.r69.baseline.md` | 1,099,874 B／`41D708D9D520B712EC3578E1FC7EADC467937CC33C7B3C5F1D0115D0B96BAB09` | 实测同值 |

设计锚点复核（按 r70 现行行号读取）：`:4065`（r65.8 `restoreReceiptWire`）、`:4066`（r65.9 `Target.DSN`）、`:4068`（r65.11 lease／两个互斥字段）、`:4069`（r65.12 `NewRuntime` 签名与调用点波及）、`:4070`（r65.13 B1／B2／B3／B5／B6 与 F017／N005／SEC-F007 残余）、`:4072`（修订范围与唯一优先级）、`:4073`（r66.1 全量调用点清单 39／17）、`:4076`（r66.4 driver 工厂化、teardown 与测试字面量）、`:4081`（r66.9 F017／N005 登记）、`:4082`（r66.10 NOT RUN）、`:4083`（r66.11 声明）。

## 2. 改前／改后

| 指标 | 改前 | 改后 | 差值 |
|---|---:|---:|---:|
| 字节 | 898,536 | 904,142 | +5,606 |
| 行数（`ReadAllLines`） | 3,289 | 3,312 | +23 |
| CR | 3,289 | 3,312 | +23 |
| LF | 3,289 | 3,312 | +23 |
| CRLF | 3,289 | 3,312 | +23 |
| SHA256 | `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` | `CA20577B379556CAC95F7C9057ACCB0EE4460BF208AB3433302AFD109ACEA83C` | — |
| 编码／BOM | UTF-8 无 BOM | UTF-8 无 BOM | 不变 |
| 行尾 | CRLF，末尾有 CRLF | CRLF，末尾有 CRLF | 不变 |

**改前值来源与可核性**：898,536 B／3,289 行／`4BD724A9…` 为 Lead 提供的已核事实，并与 `review-arch-r06`／`review-arch-r10`／`review-sec-r10` 的独立复算一致（`review-arch-r06` 记录 898,536 B／3,289 行／3,289:3,289；r10 两份报告记录同 SHA）。**盘上无同尺寸快照**：对工作区按字节数 898,536 全量搜索无命中；`git` 中该路径仅跟踪 69,445 B 的旧版本（HEAD blob `b7585888…`），不能作为本次卡面的 diff 基线。因此改前值不可由盘上文件机械复算，正文只把它作为已核对的外部事实引用。

**编辑序列（同一会话，均由精确子串替换完成）**：

| 阶段 | 内容 | 结果 |
|---|---|---:|
| 基线 | 4 文件写集 | 898,536 B／`4BD724A9…` |
| pass 1 | 写集扩为 26 文件、gofmt 26 路径、证据行加 F017 注记 | 903,037 B／`C3982B3B4B32AC650F17EAFF9C3530F24DB97D71173399B68182FA18C799ED99` |
| pass 2 | 4 处准确性修订（`:3191`、`:3193`、`:3194`、`:3201`） | 903,378 B／`1A913554875B7BD4467577938BA46555F700E674C4DE59DC58C34FF1103123E8` |
| pass 3 | 2 处定向修订（`:3192` 去掉过宽“B1/B2/B3 载体”表述；`:3235` 追加 `F017-R1` 残余与历史澄清） | 904,142 B／`CA20577B…` |

**反向替换校验**：把 pass 3 的两处替换在内存中反向应用，得到 903,378 B／`1A913554…`，与 pass 2 终值逐字节一致；证明 pass 3 只改两行、未触及其他字节。pass 2 的 4 处替换在本会话早前亦以同样方式反向校验为 903,037 B／`C3982B3B…`。pass 1 的写集扩写无法对盘上基线复算（基线不在盘上），其范围由授权编辑清单与当前卡面结构界定。

## 3. 变化行号清单（改后行号）

| 改后行 | 改前对应 | 变化 |
|---|---|---|
| `:3186` | `:3186` | **未改**：仍为 `- [ ] identity-auth-S4.24a …`，未勾选 |
| `:3191` | `:3191` | 写集标题重写：4 文件 → 26 文件，登记 r70 与来源锚点 `:4065`、`:4066`、`:4068`、`:4069`、`:4070`、`:4072`、`:4073`、`:4076` |
| `:3192`–`:3195` | `:3192`–`:3195` | 4 个修改文件行重写（职责、签名、工厂、typed nil、`restoreReceiptWire`／`Target.DSN` 删除） |
| `:3196`–`:3199` | 原 `:3196` 空行 | 新增 B5 4 个 build-tag 文件 |
| `:3200` | 新增 | `recovery_restore_production_integration_test.go`（`//go:build integration`） |
| `:3201`–`:3218` | 新增 | 既有测试头 + 17 个测试文件条目（含逐点锚点） |
| `:3219` | 新增 | 证据报告非代码产物说明（不占写集） |
| `:3223` | `:3200` | `gofmt -l` 路径清单改为与 26 文件写集一致 |
| `:3235` | `:3212` | 证据落点行追加“写集已对账至 r70”、保留门禁、`F017-R1` 残余与历史澄清 |

其余卡面行按 **+23** 平移、内容未改：`:3220` requires（原 `:3197`）、`:3221` 验收命令头（原 `:3198`）、`:3224`–`:3231` 命令体（原 `:3201`–`:3208`）、`:3233` 负向 oracle（原 `:3210`）、`:3234` 关闭条件 7 项（原 `:3211`）、`:3236` 责任人（原 `:3213`）、`:3237` 不宣称边界（原 `:3214`）、`:3238` 停止条件（原 `:3215`）、`:3240` 实施状态（原 `:3217`）。未改任务状态、责任人派发记录、关闭条件项数与语义、负向测试 oracle、停止条件、不宣称边界。

## 4. 写集逐条来源锚点（26 文件）

| # | 文件 | 变更 | 设计来源 |
|---|---|---|---|
| 1 | `cmd/han-server/recovery_restore.go` | 修改 | `:4065`、`:4066`、`:4069`、`:4076`（`restoreReceiptWire`／`Target.DSN` 删除；driver ownership 第五参数；B1/B3 驱动侧接线与 B2 消费点） |
| 2 | `cmd/han-server/recovery_runtime.go` | 修改 | `:4068`、`:4069`、`:4073`、`:4076`（restore 工厂、一次 `Acquire`、`closeRestoreLease`、`:641` typed nil） |
| 3 | `cmd/han-server/main.go` | 修改 | `:4076`（`main.go:634` 不设 `restore`，依赖 `:596`–`:599` nil 工厂） |
| 4 | `internal/recovery/runtime.go` | 修改 | `:4068`、`:4069`、`:4073`（`NewRuntime` 第三实参、`ownedExclusive`／`injectedExclusive`） |
| 5 | `cmd/han-server/recovery_restore_host_linux.go` | 新增 | `:4070` B5（`//go:build linux`） |
| 6 | `cmd/han-server/recovery_restore_host_other.go` | 新增 | `:4070` B5（`//go:build !linux`，`ErrAdmissionUnsupported`） |
| 7 | `cmd/han-server/recovery_restore_exec_linux.go` | 新增 | `:4070` B5（`//go:build linux`，fd-exec 与真实 join/reap） |
| 8 | `cmd/han-server/recovery_restore_exec_other.go` | 新增 | `:4070` B5（`//go:build !linux`，零状态变更） |
| 9 | `cmd/han-server/recovery_restore_production_integration_test.go` | 新增 | `:4070` B6、`:4076`（`//go:build integration`，`TestRecoveryRestoreProductionWiring`） |
| 10 | `internal/recovery/runtime_test.go` | 修改 | `:4073`（11 处 `recoveryTestLease{}`；`:1658` typed nil）、`:4076` |
| 11 | `internal/recovery/installation_test.go` | 修改 | `:4073`（`:133`、`:455`、`:482`） |
| 12 | `internal/recovery/admission_test.go` | 修改 | `:4073`（`:146`、`:196`、`:696`） |
| 13 | `internal/article/deletion_journal_anchor_test.go` | 修改 | `:4073`（`:198`、`:1053`、`:1407`） |
| 14 | `internal/article/deletion_journal_snapshot_test.go` | 修改 | `:4073`（`:157`） |
| 15 | `internal/article/replay_publication_test.go` | 修改 | `:4073`（`:321`） |
| 16 | `internal/article/replay_publication_integration_test.go` | 修改 | `:4073`（`:618`） |
| 17 | `internal/platform/recovery_guard_test.go` | 修改 | `:4073`（`:781`） |
| 18 | `internal/identity/recovery_installation_test.go` | 修改 | `:4073`（`:80`） |
| 19 | `internal/identity/recovery_execution_test.go` | 修改 | `:4073`（`:492`） |
| 20 | `internal/identity/recovery_execution_postgres_test.go` | 修改 | `:4073`（`:764`） |
| 21 | `cmd/han-server/recovery_workers_test.go` | 修改 | `:4073`（`:156`、`:532`） |
| 22 | `cmd/han-server/recovery_restore_test.go` | 修改 | `:4073`、`:4076`（`:228` 第三实参；`:209`／`:233` ownership 字段删除） |
| 23 | `cmd/han-server/recovery_production_chain_integration_test.go` | 修改 | `:4073`、`:4076`（`:900`、`:1327`、`:2449`；`:2999` 改工厂闭包） |
| 24 | `test/integration/recovery/runtime_fixtures_test.go` | 修改 | `:4073`（`:916`、`:1339`、`:2451`） |
| 25 | `cmd/han-server/recovery_enter_install_integration_test.go` | 修改 | `:4076`（`:232` 改工厂闭包） |
| 26 | `cmd/han-server/recovery_runtime_test.go` | 修改 | `:4076`（`:731` 不设 `restore`，依赖 nil 工厂） |

**39 处／17 文件口径复核**：本轮实测 `rg -n 'NewRuntime\(' --glob '*.go'` = **39 行／17 文件**，与 `:4073` 逐字一致。分解为：定义 1（`internal/recovery/runtime.go:729`）＋生产调用 1（`cmd/han-server/recovery_runtime.go:641`）＋测试调用 37（15 个文件）。写集另含 2 个不承载 `NewRuntime` 的 restore 工厂改造测试文件（`:3217`、`:3218`），故卡面“既有测试 17 文件”= 15（承载调用点）＋2（工厂改造）。逐文件计数与 `:4073` 完全一致：runtime_test 12（含 `:1658`）、installation 3、admission 3、deletion_journal_anchor 3、deletion_journal_snapshot 1、replay_publication 1、replay_publication_integration 1、platform guard 1、identity installation 1、identity execution 1、identity execution postgres 1、workers 2、restore_test 1、production_chain 3、runtime_fixtures 3。

**gofmt 一致性**：`:3223` 的 26 个路径与写集 26 个 `.go` 路径做集合比较，`SETS_EQUAL=True`（无差集）。

## 5. F017 判定

**在本卡授权写集范围内：闭合。** 卡面写集已覆盖 r65.11／r65.12／r66.1／r66.4／r65.13 B5 明确点名的文件；`gofmt` 清单与写集逐项一致；`NewRuntime` 39／17 口径与设计实测一致；逐点锚点（`runtime_test.go:1658`、`recovery_restore_test.go:228`、`recovery_enter_install_integration_test.go:232`、`recovery_production_chain_integration_test.go:2999`、`recovery_runtime_test.go:731`）均在卡面登记。

**不得宣称 F017 全量闭合。** 设计仍保留更早的写集建议表，且未被 r65／r66 明文逐行废止：

- `:4000`–`:4019`（r63.9.4）与 `:4257`–`:4273`（r62.9.1）列出 6 个未纳入本卡授权写集的生产文件：`recovery_restore_approval.go`、`recovery_restore_target.go`、`recovery_restore_receipt.go`、`recovery_restore_witness.go`、`recovery_config.go`、`internal/recovery/restore_source.go`，以及对应单元测试与 host／exec 平台测试。
- `:4051`（r64.12）明写“在 r63.9.4 表基础上增补”；`:4069`（r65.12）明写“在 r64.12 基础上”。两者是增补关系，不是整表废止。
- 具体缺口可机核：`cmd/han-server/recovery_config.go` 现行只有 `HAN_RECOVERY_RESTORE_EXECUTABLE`／`_SHA256`／`_MANIFEST_PATH` 三键，设计要求的 `HAN_RECOVERY_RESTORE_APPROVAL_PATH`、`_APPROVAL_TRUST_ROOT_PATH`、`_APPROVAL_TRUST_ROOT_SHA256`、`_TARGET_DSN_FILE` 均不存在；全仓 `RestoreSourceClaim`／`RestoreSourceVerifier` 实测 0 命中，26 文件写集内无 `internal/recovery` 新文件承载该导出形状。
- 因此本轮在卡面 `:3235` 登记 **`F017-R1`**：上述 6 个生产文件及配套测试未纳入授权写集，待设计所有者确认是否已被 r65.13 B1／B2／B3 的“设计级处置”或 r66.4 收口；确认前 F017 只在本卡授权写集范围内闭合。

**写集对账完成 ≠ 可实施。** 本报告不构成实施授权，也不构成 F017 全量闭合。

## 6. 保留门禁（逐条不变）

1. **`F004`**：ADR 同步待架构决策方；`NewRuntime` 构造契约是否在 ADR 显式登记未拍板。
2. **`N005`**：`decisions/framework.md` 状态段与 `tasks.md:3220`（原 `:3197`）对“现行 ADR-0031 修订与启动门禁”的陈述冲突；范围外，需架构决策方与 Task-Planner 对账。本轮未改 ADR，未改该行语义。
3. **`SEC-F007` 与 Linux 侧能力 NOT RUN**：host／ACL／mount／loader／fd-exec／maps／network／witness 实测均未运行；Windows 只覆盖解析与语义层；`//go:build linux` 的 skip 与无匹配测试的 exit 0 都不等于 PASS。
4. **生产 closed 与 ADR-0031 约束 8 未满足**：`recoveryClosedRestoreDriver` 仍是默认；`enter`／`resume` 未解锁；生产 restore 未解除。
5. **任务状态未变**：`:3186` 仍为 `- [ ]`；未勾选、未派发新责任、未关闭、未签收；关闭条件仍 7 项；负向 oracle、停止条件、不宣称边界未改。

## 7. 只读声明

- 仅修改 `.add/project/knowledge/tasks/tasks.md` 的 S4.24a 卡面 `:3191`–`:3219`、`:3223`、`:3235`；其他卡一行未动。
- 未修改 `design.md`、`decisions/framework.md`、`cmd/**`、`internal/**`、`test/**` 或任何测试代码。
- 未执行任何 Git 写操作（无 add／commit／branch／checkout／reset／stash）。
- 未删除、移动、归档任何文件；仅新建本证据目录与报告。
- 未勾选 `:3186`，未改任务状态、责任人派发记录、关闭条件项数与语义、负向测试 oracle、停止条件、不宣称边界。

## 8. NOT RUN 声明

本报告未运行任何验收命令：未运行 `gofmt`、`go build`、`go vet`、`go test`、`go test -race`、integration tag 测试、Linux 构建或 E2E。`:3221`–`:3231` 的 9 条命令全部保持 **NOT RUN**。本报告只做文本对账与只读检索（`rg`、`Get-FileHash`、行读取、集合比较）。

## 9. 复核命令与结果

| 检查 | 结果 |
|---|---|
| `tasks.md` 行数／CRLF | 3,312／3,312，末尾 CRLF，UTF-8 无 BOM |
| `:3186` | `- [ ] identity-auth-S4.24a …`，`unticked=True` |
| 写集 `.go` 路径数 | 26 |
| `gofmt` 路径数 | 26 |
| 集合比较 | `SETS_EQUAL=True` |
| `rg 'NewRuntime\('` | 39 行／17 文件 |
| `design.md` | 1,103,152 B／5,476 行／`3A8A9E63…4D5254` |
| 两路 r10 复审 | PASS，哈希与字节同输入事实 |
| 反向替换（pass 3） | 904,142 → 903,378 B／`1A913554…` |

## 10. 需用户／设计所有者确认

1. **`F017-R1`**：r63.9.4／r64.12 表中的 6 个生产文件与配套测试是否已被 r65.13 B1／B2／B3 或 r66.4 取代？若未取代，是否授权扩张写集；若取代，请设计侧补一句明文取代关系。
2. **`N005`**：`framework.md` 与 `tasks.md:3220` 的门禁陈述由谁先统一。
3. **`F004`**：ADR-0031 是否显式登记 `NewRuntime` 构造契约。
4. **实施授权**：写集对账完成不等于可实施；本卡仍需另行取得实施派发与人的生产批准。