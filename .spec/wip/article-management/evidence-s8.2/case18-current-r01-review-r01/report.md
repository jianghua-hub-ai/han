# Case18 当前 revision 执行证据 r01：独立复审报告

**AI 代理评审，非真人签字。**

## 0. 对象、修订与授权

- objectType：执行证据包（服务于代码验证；按 `add-code-review` 的 code 对象分支核对证据与消费者，不评审需求/设计/ADR 文本本身）。
- 固定对象：`.spec/wip/article-management/evidence-s8.2/case18-current-r01/`，共 12 个文件（`report.md`、`index.json`、`source-hashes-before.txt`、`source-hashes-after.txt`、8 个 run 日志）。
- 工作区：`D:\AID\VSCode\Codex\Han`，branch `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（开工与收工均核对）。
- 目标：独立判断该证据包是否足以支持 `C18-COV-F001` 关闭、`C18-COV-F004` 的 Session 侧覆盖，以及 `C18-COV-F002/F003/F005` 是否保持原结论；不代替 Lead 做任务关闭决定。
- 上游依据：上轮 `.spec/wip/article-management/evidence-case18-coverage-r01/verdict.md` 的六项关闭前置与 `C18-COV-F001`～`F007`；`tasks.md` 的 `article-management-S1.5`、`article-management-S8.2` 及 2026-11-03 记录块；ADR-0028；用户本轮只读复审指令。
- 允许范围：仓库只读、独立执行测试与构建、只在本评审目录写报告；禁止 Git 写、禁止改源码/测试/证据包、禁止删除/移动/归档、禁止打印 DSN。
- 交回位置：Lead 的 Case18/S8.2 关闭评估输入。本报告不改任务状态，不宣布任何 task CLOSED。

## 1. 两轮实际检查范围

### 第一轮：对象冻结与静态核对

- 读取 `add-code-review` 完整正文、`code-review-checklist`、`abstraction-principles`、`产品安全红线`、上轮 verdict、S1.5/S8.2 任务卡。
- 枚举证据目录 12 个文件，重算全部 SHA256/字节，与 `index.json.artifacts` 和 `index.json.runs[*].log_sha256/log_bytes` 对账；核对 `index.json` 自身哈希（它未被自身 artifacts 声明）。
- 重算 `source-hashes-after.txt` 所列 8 个源文件当前哈希/字节，并与 before/after 两份清单比对；核对 before/after 文本内容与编码。
- 逐份解析 8 个 run 日志的 PASS/FAIL/SKIP、`DATA RACE`、schema 行；核对 `index.json.runs[*].command` 与指定命令（含 `-race` 位置和 `-run` 正则）。
- 追踪上轮 `C18-COV-F001` 所指旧哈希的实际来源，确认是否与当前 revision 为不同对象；核对 `C18-COV-F002/F003/F005` 的现行源码依据。
- 核对 `report.md` §5 与 `index.json.not_proven` 的边界声明，检查是否存在“整卡 PASS / Case18 PASS / production readiness”式外推。

### 第二轮：独立重跑与消费者核对

- 独立重跑 8 条命令：root 普通/race、UC 普通/race、build、vet、article 集成、identity 集成；记录真实退出码、PASS/FAIL/SKIP、耗时和新增 schema。
- 独立检查 Session 侧集成测试是否真实使用 `session.NewStore`、是否在真实 PostgreSQL 上执行撤销/停用后的 `ResolvePrincipal` 拒绝。
- 独立检查 production `Enter`/`Install` 是否仍返回 `ErrAdmissionClosed`，以及 F002 矩阵是否仍使用 fake runtime。
- 复核重跑前后源码哈希未变化；收工复核 HEAD、工作树与写入边界。

## 2. 逐项核验结果

### 2.1 完整性：PASS

磁盘上 12 个文件；`index.json.artifacts` 声明 3 个，`runs` 声明 8 个，`index.json` 自身由本次评审补算。11 个被声明文件的 SHA256/字节全部一致，0 mismatch；`index.json` 自身为 `609CFBF075078808E1483B9F20E5EDBF5395AB426652EB756054CEF4E36DFCD2` / 6640 B。

| 文件 | SHA256 | 字节 | 对账 |
|---|---|---:|---|
| `report.md` | `C54E4EAB1AF5761F2F250CE51A1E0B4EAD63A22AC51B02AC60E99D20F46EEC3C` | 7384 | OK |
| `index.json` | `609CFBF075078808E1483B9F20E5EDBF5395AB426652EB756054CEF4E36DFCD2` | 6640 | 自算，OK |
| `source-hashes-before.txt` | `7B78FD435F5A65CBFEA2F9ECEC829A4BCE5DE712FBB24AF726AE9692D6F1F635` | 1654 | OK |
| `source-hashes-after.txt` | `6CF304C63CFF1B6B63D37540BB71896D2B339F6BEC71F8E11C30999545F7F334` | 829 | OK |
| `run-root-normal.log` | `D58A8DF3F1949568D71A43DBFE5DA687B91DBBE781B4349C97B060CFD30F48B4` | 18999 | OK |
| `run-root-race.log` | `5A2618FEE27651C19A1716958D75049F108BA2EFEBC44FC6AB24B4133079EE67` | 18999 | OK |
| `run-uc-normal.log` | `4C7D1CC6655E1AF2E265A6EE2C09FE716FA61248C5AB4F9EE88899175BFCC0EA` | 18001 | OK |
| `run-uc-race.log` | `9B3323C1E9F3764C145751B832FE3FEAFE3A4FB4A3A313C50A3FC6CC223931D7` | 18001 | OK |
| `build.log` | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` | 0 | OK |
| `vet.log` | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` | 0 | OK |
| `run-adminquery-pg.log` | `F51D2F4F942C067426285A0D4DA318B63E1A6B66FDA4F58D64FA5390F52F17E3` | 64739 | OK |
| `run-session-sql-pg.log` | `B000D2141C6E61D5C2A7732EF03B98FF83A88FE020EC0F579D7A1231D60F4E1F` | 8447 | OK |

### 2.2 源码绑定：PASS

`source-hashes-after.txt` 的 8 个源文件与当前工作树逐一匹配，0 mismatch；`source-hashes-before.txt` 与 after 解码后的文本逐行一致。before 为 UTF-16LE BOM、after 为 UTF-8 BOM，二者文件字节数不同（1654 vs 829），这是清单文件编码差异，不是源文件内容差异；报告所述“八文件前后 SHA256/字节相同”指 8 个源文件，核对成立。

| 源文件 | 当前 SHA256 | 字节 | 对账 |
|---|---|---:|---|
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | OK |
| `cmd/han-server/main_test.go` | `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` | 400312 | OK |
| `internal/article/query_service.go` | `D3151AC329C8C9EB8A1B8E00FF45A58CC467E06E1C9A00F9A63830048C0108B4` | 29621 | OK |
| `internal/article/handler.go` | `C684B4C874083035C1BC6B1F22DF81833D8DF490544E8586719C028B203D3750` | 37839 | OK |
| `internal/article/model.go` | `42B2BEEEAC0B4CB3B6E2138314956F2D1FCBD0E32D6E7B32BB2352C656E8851D` | 13800 | OK |
| `internal/article/policy.go` | `6CD1543BC2E715F76EE7B6C5CDDDB3153A9A425B84C55A366648E600B78C8774` | 13831 | OK |
| `internal/article/admin_query_test.go` | `DBAF93B7600831D61DC74804D82905CAE0719014357A9107321DFCD8C3EF3F61` | 30090 | OK |
| `test/integration/article/admin_query_test.go` | `3ADC5BFA00410F2BF7B520C0035E0089B6E24889770BF6BD1F283A7E5B798263` | 31515 | OK |

上轮 `C18-COV-F001` 所指旧哈希确属不同对象：`.spec/wip/article-management/evidence-s8.2/final-r01/review-root-r02/stage-readset-binding.json:35-36` 记录 `main.go 6B45B550…` / `main_test.go B53DD34D…`，同文件 `:38-42` 明示 `PExecuted:false`、`UCReexecuted:false`、`Case18Overall:"BLOCKED upstream"`。当前证据包绑定的是 `6D5373…/173317` 与 `372065…/400312`，不是旧对象。

### 2.3 可复现性：PASS（计数与报告一致；含 `-race` 限制）

本次独立重跑的真实结果如下。所有命令均在仓库根执行，DSN 仅从 `$env:HAN_TEST_POSTGRES_DSN` 读取，未打印、未落盘。

| # | 命令 | EXIT | PASS | FAIL | SKIP | 本次耗时 |
|---|---|---:|---:|---:|---:|---:|
| 1 | `go test ./cmd/han-server -run '^TestProductionAdminArticleQuery$' -count=1 -v` | 0 | 93 | 0 | 0 | 4.653s |
| 2 | `go test -race ./cmd/han-server -run '^TestProductionAdminArticleQuery$' -count=1 -v` | 0 | 93 | 0 | 0 | 9.580s |
| 3 | `go test ./internal/article -run '^(TestAdminQuery\|TestAdminCursor\|TestAdminCursorFiveViewIsolation)$' -count=1 -v` | 0 | 135 | 0 | 0 | 1.819s |
| 4 | `go test -race ./internal/article -run '^(TestAdminQuery\|TestAdminCursor\|TestAdminCursorFiveViewIsolation)$' -count=1 -v` | 0 | 135 | 0 | 0 | 5.739s |
| 5 | `go build ./...` | 0 | - | - | - | 2.857s |
| 6 | `go vet ./cmd/han-server ./internal/article` | 0 | - | - | - | 1.194s |
| 7 | `go test -tags=integration ./test/integration/article -run '^TestAdminQueryPostgres$' -count=1 -v` | 0 | 99 | 0 | 0 | 23.089s |
| 8 | `go test -tags=integration ./test/integration/identity -run '^(TestS16SessionResolveRejectsRevokedAndSuspendedUsers\|TestS16UserFirstLockOrderingAndSessionRevocation\|TestS33LoginSessionPostgres)$' -count=1 -v` | 0 | 6 | 0 | 0 | 7.640s |

- 与 `report.md` §3 的 93/93/135/135/99/6 PASS、0 FAIL/SKIP 全部一致；`build.log`/`vet.log` 为 0 B 与 exit 0 一致。耗时与报告不同属正常环境波动，不作为差异。
- 原日志同样为 93/93/135/135/99/6 PASS、0 FAIL/SKIP，`DATA RACE` 0 行；`index.json.runs[*].command` 与指定命令逐字一致（含 `-race` 位于 `test` 之后、包路径之前）。
- 运行 7 的 16 个顶层子测试与报告一致；运行 8 的 3 个顶层测试与报告一致。
- **客观限制**：`-race` 在无竞态时不产生额外输出，证据日志本身不能证明 `-race` 已启用；本轮独立重跑是“race 已实际启用”的主要证据，日志一致性只证明输出内容可复现。
- 集成测试会新建并保留 schema，属预期行为；本次未 DROP、未清理。本次独立重跑新产生的 schema 见 §2.3.1。

#### 2.3.1 本次新产生并保留的 schema

运行 7 新产生 28 个 `han_admin2_*`：

`han_admin2_02bd7cf1772245708b4dce3ca83fc6a8`、`han_admin2_047d30631e604dadab0cbb86af590aec`、`han_admin2_04c5b12829d840ddb811cbedc3edce66`、`han_admin2_0550c5fd2ab143d58d2d0fb90494cac4`、`han_admin2_0ffc32d535f94f61b482bde54f995c33`、`han_admin2_15431759462046f497ee85a690aa56c8`、`han_admin2_1853db15b95b4645b45132f17bb6fbf6`、`han_admin2_19100620ddff43dba5c13869b349f502`、`han_admin2_29c9c02559a34b34bc965f7a74c9b8e8`、`han_admin2_2bc46ae5122044c4bbdf1d3f48b75c3a`、`han_admin2_3b12b92d8a4a479ca48feed224c51571`、`han_admin2_51c65bff807b47d7b9469e2de2c3d36c`、`han_admin2_5507257e2fe34e3f9a43d6e117ac0211`、`han_admin2_7111d5d1564c4b68b8c4379265dcd25d`、`han_admin2_7a26bab167ca47868b3e80b00da2e0d3`、`han_admin2_9b8182a5ed4d4890ad2470b7cd2c8048`、`han_admin2_a538e9af04364ad682fdf012524a221c`、`han_admin2_ae8460a3116a414ab175ba296f15f22e`、`han_admin2_b8b478545e964906892be5ffec511f34`、`han_admin2_cdb419f2c5234afa8b851ec7962e2fe0`、`han_admin2_d27cd830f617405a8fe49a08d5765eb0`、`han_admin2_dda92989588846149167898c34a227c5`、`han_admin2_df5899cbefc841d59c6b3b697b638464`、`han_admin2_e3ea1ef12030474d80a5198337cdcb34`、`han_admin2_e7cba87a593b433e96899a4bf1ba7533`、`han_admin2_ee33abb7c237428ba4af073101154de8`、`han_admin2_ee4b480cbaa54298a692f3867e57588c`、`han_admin2_ffb9fb53c73648c7a4559638481862bf`。

运行 8 新产生 3 个 `han_s33_*`：

`han_s33_1a17f04bb76d4f6e864458f3976bc60a`、`han_s33_82f239fb7ce84a34b9c1944f084e8ab2`、`han_s33_b6932f28cdfc4771aa793f16c78e166f`。

以上 31 个 schema 均按测试设计保留，本评审未删除、未清理。

### 2.4 结论有效性

| 项 | 判定 | 依据 |
|---|---|---|
| `C18-COV-F001` | **CLOSED** | 当前 revision root/装配用例普通与 `-race` 各 93 PASS / 0 fail / 0 skip，独立复现；源码绑定 `6D5373…/372065…`，与旧 `6B45B550…/B53DD34D…` 确属不同对象；旧证据 `PExecuted:false` 已被当前执行替代 |
| `C18-COV-F004` Session 侧 | **已覆盖** | 运行 8 真实 PostgreSQL + `session.NewStore`；`ResolvePrincipal` 在撤销 session、停用 user 后返回 `ErrSessionRequired`；本次 6 PASS 独立复现 |
| `C18-COV-F004` 整体 | **仍 open / 部分关闭** | AccessGeneral 侧仍只有 resolver/repository/handler 接缝证据，未见当前 revision 真实覆盖；不得整体 CLOSED |
| `C18-COV-F002` | **不变，保持 open** | 矩阵仍用 `newS419RecoveryRuntime` fake runtime，未做真实 runtime/PG/双系统；报告的状态词表另有 LOW 级表述问题，但不改变 open 结论 |
| `C18-COV-F003` | **不变，保持 open** | `cmd/han-server/recovery_runtime.go:494-495` 仍以 `ErrAdmissionClosed` 拒绝 Enter/Install，无完整恢复→`public_ready`→生产开流证据 |
| `C18-COV-F005` | **不变，保持 open** | 本证据包未处理 `evidence-s1.5/producer-real-r01/index.json` 的哈希/字节修正；按 Lead 已裁定的“时点快照语义”保持 open |
| 边界标注 | **PASS** | `report.md:60-66` 与 `index.json.not_proven:99-109` 均列 10 项未证明；未发现“S1.5/S8.2 已关闭、Case18 已 PASS、production readiness”表述 |
| 安全红线 | **PASS** | 未发现 DSN/凭据输出或调试后门；日志中的 token 为测试合成值；本次未打印 DSN |

Session 侧源码依据：`test/integration/identity/postgres_test.go:318` 使用 `session.NewStore`；`:1038` 为 `TestS16SessionResolveRejectsRevokedAndSuspendedUsers`；`:1065` 先成功 `ResolvePrincipal`；`:1076-1082` 在真实 SQL 中撤销 session、停用 user；`:1084-1092` 再次 `ResolvePrincipal` 返回 `ErrSessionRequired`。

F002/F003 源码依据：`internal/recovery/runtime.go:27-32` 的 runtime 常量为 `closed/checking/public_ready/faulted/draining/stopped`；`cmd/han-server/recovery_main_dispatcher_integration_test.go:510-531` 的 `newS419RecoveryRuntime` 仍是 fake；`cmd/han-server/recovery_runtime.go:494-495` 的 `recoveryEnterInstallClosedError` 仍返回 `ErrAdmissionClosed`。

### 2.5 边界复核：PASS

`report.md` 明确列出不证明：真实部署 E2E、真实 SMTP/approved-provider、双系统指标/告警/job 接线、完整恢复后生产开流、`Enter`/`Install` 开流、真实 Journal/HMAC/keyring、旧备份恢复、容量/P95、production readiness、人的业务接收。`index.json.not_proven` 同列 10 项。报告只主张 `C18-COV-F001` 的“事实层消除”和 F004 Session 侧“事实层消除”，并明确 finding 关闭须由独立复审判定；未把子集 GREEN 外推为整卡 PASS。`tasks.md:1877` 也保留 S8.2 `[-]` 与 `blocked-upstream` 不解除。

## 3. Findings

### [MEDIUM] 保留 schema 清单少计 29 个，证据资源盘点不完整

- finding ID：`C18-R01-REV-F001`
- 对象/修订：`.spec/wip/article-management/evidence-s8.2/case18-current-r01/`，HEAD `6da8abbc…`
- 位置：`.spec/wip/article-management/evidence-s8.2/case18-current-r01/report.md:48`、`:50`；`index.json:88-90`；`.add/project/knowledge/tasks/tasks.md:1877`
- 现状：报告、索引和任务记录各只登记 1 个 `han_admin2_*` 与 1 个 `han_s33_*`；实际 `run-adminquery-pg.log` 有 28 个唯一 `han_admin2_*`（`created=true` / `retained=true` 各 28 次），`run-session-sql-pg.log` 有 3 个唯一 `han_s33_*`（均 `created_here=true retain_policy=always`）。合计少计 27 + 2 = 29 个 schema。
- 触发：任何读者把 `retained_schemas` 当作完整资源清单做磁盘/配额盘点或后续清理；或审计者据此核对“保留 schema”范围。
- 影响：证据包的资源记录不完整，后续清理会漏掉 29 个数据库对象；报告“保留 schema”表述的可信度下降。这不否定运行 7/8 的测试 PASS，但属于证据完整性问题。
- 建议：按两份日志的唯一 schema 全量登记，或把字段明确改名为代表性样本并同时给出完整数量与来源；不得删除/清理这些 schema。
- 验证：`schema=` 唯一值集合与 `retained_schemas` 逐项对账；本次独立重跑另行产生 28 + 3 个 schema，均登记于 §2.3.1，未清理。

### [LOW] 准入状态词表混入 observation/fake 词并漏 `checking`

- finding ID：`C18-R01-REV-F002`
- 对象/修订：同上，`report.md` §4 与 `tasks.md:1877`
- 位置：`.spec/wip/article-management/evidence-s8.2/case18-current-r01/report.md:56`；`.add/project/knowledge/tasks/tasks.md:1877`
- 现状：报告写准入状态为 `closed/internal_ready/public_ready/unknown/faulted/draining/stopped`；`internal/recovery/runtime.go:27-32` 的 runtime 状态常量实际是 `closed/checking/public_ready/faulted/draining/stopped`。`internal_ready` 是 `runtime.go:1015` 的 observation 字面量，`unknown` 是测试 fake 的 `s419ErrAdmissionUnknown`（`cmd/han-server/recovery_main_dispatcher_integration_test.go:508`、`:527`），且词表漏掉实际存在的 `checking`。
- 触发：读者依据该词表判断“全状态”覆盖或据此补测试时，会把 observation/fake 词与 runtime 状态常量混为一谈，并漏检 `checking`。
- 影响：状态覆盖说明不准确，可能误导后续 F002 修复范围；不影响本证据包对 F002 “仍 open”的结论。
- 建议：改为准确的 runtime 状态常量词表，并分别标注 observation 与测试 fake 的来源，避免把 fake 词当作真实状态。
- 验证：对照 `runtime.go` 常量、observation 字面量和测试 fake 定义逐项复核；修正后重跑相关测试。

## 4. 待决定与非阻断建议

### 待决定

1. **MEDIUM finding 的处置**：由 Lead/责任人决定是修正证据包的 schema 登记后复审，还是显式接受该记录偏差。按 `add-code-review` 严重度表，MEDIUM 需“修复后复审，或由责任人显式接受”；本评审不替 Lead 接受风险。
2. **F004 的表述方式**：建议在关闭评估中写成“Session 侧已覆盖、AccessGeneral 侧仍 open，finding 整体未关闭”，不要把部分覆盖写成整体 CLOSED。
3. **F001 的关闭范围**：本评审仅确认 `C18-COV-F001` 所指的“当前 revision root/装配执行证据 + 源码绑定”条件已满足；不表示 Case18 全部五分项或 S8.2 整卡已关闭。

### 非阻断建议

- 在 `index.json` 中补 `index.json` 自身哈希，或明确说明索引文件不自声明，减少完整性复核歧义。
- 在 `source-hashes-before.txt` / `source-hashes-after.txt` 旁注明编码差异（UTF-16LE BOM vs UTF-8 BOM），避免读者把清单文件字节数差异误判为源文件差异。
- 在报告 §3 明确写出“`-race` 日志不能自证 race 已启用，需以独立重跑为主要证据”，与本次复审口径一致。
- 若保留 schema 数量继续增长，建议在证据包中增加“本次运行新增/累计保留”的计数，避免逐条列表遗漏。

## 5. 证据限制与未验证边界

本评审已独立执行 8 条命令，但不证明、也不能替代：

- 真实部署 E2E；
- 真实 SMTP/approved-provider；
- 双系统指标/告警/job 接线；
- 完整恢复后生产开流；
- `Enter`/`Install` 开流；
- 真实 Journal/HMAC/keyring；
- 旧备份恢复；
- 容量/P95；
- production readiness；
- 人的业务接收。

其他限制：

- 集成测试是单次本地 PostgreSQL 运行，不是 soak、容量或生产验证；本次新增 31 个保留 schema 会占用测试库资源。
- `-race` 的日志内容在无竞态时与普通运行可能相同，不能仅凭日志证明 race 已启用；本次独立重跑是主要证据。
- 未重新验证 AccessGeneral 侧，未重新验证 F002/F003/F005 的完整修复；相关结论只按现有源码与证据判为“保持 open/不变”。
- 本评审未修改 `tasks.md`、未宣布任何 task CLOSED；S8.2/S1.5 的最终状态由 Lead 按流程决定。

## 6. 收工状态

- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`（未变）。
- 工作树：开工基线 `git status --porcelain` 310 行；收工仍 310 行（本机 `git status` 默认不显示未跟踪文件）。显式 `git status --porcelain --untracked-files=all` 可见本次唯一新增 `.spec/wip/article-management/evidence-s8.2/case18-current-r01-review-r01/`（3 个文件）；tracked 改动与开工基线一致，未改源码、测试、证据包或任务文件。
- 源码：8 个源文件收工哈希仍与 `source-hashes-after.txt` 一致。
- 本报告为 AI 代理评审，非真人签字；不代替人的业务接收、Lead 的关闭决定或后续授权。