# S1.5 第 7 子门禁 producer 真实 Publish/ReadCurrent 独立评审 R01

- 结论：**PASS WITH MINOR**
- 身份：**AI 代理评审，非真人签字**；本评审为固定证据修订的非作者独立评审，不代表作者自证、人的业务接收或整卡关闭。
- 固定工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 评审对象：`.spec/wip/recovery-runtime/evidence-b13d-r01/a48-f001-repair-author-r02/` 来源证据，以及 `.spec/wip/article-management/evidence-s1.5/producer-real-r01/` 消费登记。
- 现役代码对照：`internal/article/replay_evidence.go`。
- 评审范围：仅第 7 子门禁“producer 真实 `Publish`/`ReadCurrent` 层”，不代替 S1.5 整卡、S8.2、Case18、部署 E2E 或人的接收结论。

## 1. 两轮检查范围

- 第一轮：核对来源索引与实物、顶层文件哈希/字节、10 源现役哈希、`replay_evidence.go` entry/exit、两轮命令与原始产物、`run.py` DSN 与隔离门禁、PG preflight、消费登记字段与边界声明。
- 第二轮：从 `stdout.log` 独立重算叶子集合与 58 pass / 0 fail / 0 skip，逐项比较 `leaves.json`、`commands.json`、`result.json`；复核进程收口、消费登记 11 项上游哈希、未证明范围、报告命令展示差异与证据包自包含复跑条件。
- 方法依据：`.codex/skills/add-code-review/SKILL.md` 完整正文、`.codex/guidance/task-execution-workflow.md`、`.codex/guidance/coding-standards.md`、`.codex/guidance/no-hardcode-policy.md`；并按代码/证据对象读取 S1.5 卡、矩阵与 ADR-0028 语境。

## 2. 六项断言判定

### 断言 1：来源 index 与实物逐项一致 — PASS

- `index.json` 登记 106 项，路径唯一 106；目录实物 107 个文件。
- 107 与 106 的差为 `index.json` 自身；索引未登记自身，不构成缺项。
- 缺失 0；SHA256 或字节不一致 0；未登记 payload 0。
- 顶层文件实算：

| 文件 | SHA256 | 字节 |
| --- | --- | --- |
| `index.json` | `968A8FBECD3F553D69728DD252D0C8F73C59F6A88C5B867142DFB08EE338BBD4` | 32021 |
| `source-protection.json` | `C9D35B0B4B32D0AF636FF8093C7BEBB92F7C839F5A13F24FC8651FD3822B196E` | 49186 |
| `source10-before-after.json` | `6AE6D329E52497CB2958870A26DF97F1CC347E4A679206523B6ECB3B6275AEC7` | 2726 |
| `report.md` | `9C152CADFACA8A32156BE07B1B2C37F16DBF49F4C133A3BC2EE3B17FB1A8769C` | 3749 |
| `commands.json` | `A8890AB32BE29A1C2D68CC059A343625630588F1C4C87342152713228E289993` | 34196 |
| `run.py` | `ADEDDC30E59AC840705F0108ED818408B569D32EBEF644E6E6AEFF7B580906F5` | 10583 |

### 断言 2：10 源现役一致，`replay_evidence.go` entry/exit 一致 — PASS（限定）

- `source10-before-after.json` 共 10 项；按 `after_sha256` 对当前工作树逐项实算，mismatch 0。
- `internal/article/replay_evidence.go` 的 entry、exit、当前值均为 `7C14E74DCEF68A1B8067DCEC2BF50BA4D1FA9BA165AEE5C52B3828D0D60417FB`，字节 34397，`unchanged:true` 成立。
- 10 源只是运行绑定范围，不等于 177 项保护清单的当前全量快照。177 项 entry/exit diff 为 0、`unchanged:false` 为 0、缺失 0；但当前工作树相对 exit 有 28 项漂移，故本评审只确认“10 个运行绑定源 + `replay_evidence.go` 当前一致”，不声称 177 项当前全量未变。

| 运行绑定源 | 当前 SHA256 | 字节 |
| --- | --- | --- |
| `migrations/0011_article_replay_publication.sql` | `7C7F1B1F0BEFDBF572B0EE768FAC3ADD987D58DF8CA4525072CD1EE269F84CC2` | 2762 |
| `internal/article/replay_publication.go` | `34B698A231B32101ACBF1B91AB7843FF1710D529C748072CFDF27BA1CF180B8E` | 37500 |
| `internal/article/replay_publication_test.go` | `412A145A50A374C689D29EA3050DE48BCB9C827E7D7DB4E927CE08A828F69B05` | 20382 |
| `internal/article/replay_publication_integration_test.go` | `8F5DD24DD40BE6A1A6CD4FBB9EF5990EE7EDBCEB964FF9ACF6BD41E5967540FD` | 65102 |
| `internal/article/replay_evidence.go` | `7C14E74DCEF68A1B8067DCEC2BF50BA4D1FA9BA165AEE5C52B3828D0D60417FB` | 34397 |
| `internal/article/deletion_replay.go` | `33AB20A7EB3A963889B9A8F027D89A8407C67F3D9C7DB85FC7814F9D1B3B577D` | 16175 |
| `internal/article/deletion_replay_source.go` | `485A4621FA22AD7442D9ACF36A9B36118C6CE3C96E37E6DA8EB2FCFA2293DF61` | 10144 |
| `internal/article/deletion_replay_test.go` | `A4923E7B58D74EAB743C21F681F1010E8AAC592396A723E7BF3B3D0B181B02F4` | 30151 |
| `internal/article/deletion_replay_source_test.go` | `2F8E38D81F4CC88E03E02AFC0602501ACC2241E40FC0270D94B270D756EA0695` | 15341 |
| `internal/article/deletion_replay_integration_test.go` | `CB2E859535E0D8BC421298BDE13452271DC0B6B3EAE005BEA9CF76E8224A5F50` | 15900 |

### 断言 3：两轮原始产物可独立复算 — PASS

- `commands.json` 的 `required-pg-normal` / `required-pg-race` argv 与各自 `result.json.argv` 逐元素完全相等。
- 两轮均无 `-overlay`；`required-pg-race` 确实带 `-race`，`required-pg-normal` 不带 `-race`。
- 从 275985 B / 275949 B 的 `stdout.log` 独立解析 test2json 终态叶子，两轮均为 58 leaves、58 pass、0 fail、0 skip。
- 与保存的 `leaves.json` 比较：missing 0、extra 0、action diff 0。
- 目标叶子 `TestReplayPublicationPostgres/PublicationCurrentEvidenceAtomic/RefreshOutsideRunAndRecheck/Publish` 与 `.../ReadCurrent` 两轮均 PASS。
- 两轮无 `WARNING: DATA RACE`、无 `panic:`；`stderr.log` 均为 0 B，哈希均为 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。

| 轮次 | exit | pass/fail/skip | stdout SHA256 | leaves SHA256 | result SHA256 |
| --- | --- | --- | --- | --- | --- |
| `required-pg-normal` | 0 | 58/0/0 | `52BBB9828525F14DA8AC2E04DF7E437B9FAED5444E83880143B721E0058531F9` | `2F0F01CD9D2D3366E2A8CD56D043040747A37820CBDBCF0712769F3CFF8E1819` | `A306259A59726C009161E73D4366F2F941461242ABCF02777F3BF1B1AEA4C160` |
| `required-pg-race` | 0 | 58/0/0 | `2602BD3C6F679CFEDD7D649B4D0B8AB31B9181521C8CAFB5751A21D62FAAA8F9` | `B44F93D1F892B6239F525EFEF8562CD31A41D62A1F09AAE2411CE22A47AC3180` | `A4449600EA0C84AA2ECE8227101731C38ED0B50DED255ECAB768B12BE8A7D6C0` |

- 进程收口：normal job 9/9，race job 45/45；两轮全部取得的 handle 均 `WaitForSingleObject` 返回 true，active 均为 0，`observer_errors` 均为 0，`PipesJoined=true`，parent exit 均为 0。

### 断言 4：`run.py` 无环境 DSN 回落，空 DSN `Fatal`，隔离变量存在 — PASS（附证据包缺陷 F001）

- `run.py:29-31` 仅保留白名单系统变量，并显式置 `HAN_TEST_POSTGRES_DSN=''`、`PGPASSFILE=packet/no-pgpass`、`PGSERVICEFILE=packet/no-service`、`PGSYSCONFDIR=packet/no-sysconf`。
- 未发现读取外部既有 `HAN_TEST_POSTGRES_DSN` 的路径。`run.py:36-38` 仅在 `--pg` 时读取 Main 提供的 `article_dsn`，并断言包含 `@127.0.0.1:51934/han_b13d_article` 后注入。
- `internal/article/replay_publication_integration_test.go:31-42` 对空 DSN `t.Fatal("explicit isolated Article DSN is required; no fallback")`，对非 `127.0.0.1` host `t.Fatal`；该文件无 `t.Skip`。
- 真实调用链成立：fixture 持有 `*PostgresReplayEvidenceStore`（`:194`、`:241`），`publicationPGRollback` 直接调用 `f.store.Publish`（`:455`）和 `f.store.ReadCurrent`（`:461`）；`replay_evidence.go` 对应方法位于 `:299`、`:433`。
- 但 `run.py` 不是可自包含复跑包，详见 `S15-PR-F001`。

### 断言 5：PG 事实一致 — PASS（口径限定）

- `checks/preflight-r02/stdout.log` 原文：`physical Article preflight version=180006 host=127.0.0.1 port=51934 db=han_b13d_article user=han_b13d data_volume=D:; rows released acquired=0`；该 preflight exit 0。
- 来源 `report.md:13` 的 `18.6`、`127.0.0.1:51934`、`han_b13d_article`、`han_b13d` 与该 preflight 一致。
- `required-pg-*/result.json` 的 `pg` 字段本身仅记录 `"freshD Article only"`；版本与地址不是该字段单独承载，而由 preflight 原始日志和 runner 的 DSN 断言支撑。消费 index 的 `PostgreSQL 18.6 at 127.0.0.1:51934 / han_b13d_article` 未超出上述来源。
- preflight 轮使用 test-only `-overlay`，但 `required-pg-normal` / `required-pg-race` 两轮无 `-overlay`，不能混为一谈。

### 断言 6：消费登记未放大 — PASS

- 消费 `report.md`：`A9A4ECBB9DEF6C1862ECE1D151F085F179DE43A6A489FD94411B017978150BFF` / 8023 B。
- 消费 `index.json`：`F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B` / 19567 B。
- 消费 `index.json` 的 11 项上游证据逐项实算，哈希/字节 mismatch 0。
- 两轮消费登记的 argv、exit、58/0/0、stdout 哈希与来源 `result.json` 完全一致。
- `source_stable=true` 的作用域明确限定为 10 个源文件。
- `not_proven` 明确列出 S1.5 本轮自跑、真实 Journal/HMAC/keyring、旧备份恢复、Case18 全局恢复隔离、应用角色 ACL、部署 E2E、S1.5 整卡关闭与 production readiness。
- `known_gaps` 保留 `S15-RE-F008` open 与 `S1.5-REMAINING` open；消费 report 第 6/7 节明确不关闭 S1.5、S8.2、Case18 或部署 E2E。
- 未发现把本证据扩写为真实 Journal/HMAC/keyring、旧备份、Case18、ACL、部署 E2E 或整卡 PASS 的陈述。

## 3. Findings

### S15-PR-F001 — LOW，非阻断：来源证据包不能自包含复跑

- 位置：`.spec/wip/recovery-runtime/evidence-b13d-r01/a48-f001-repair-author-r02/run.py:15-16`、`:31`、`:36`；消费 `report.md:41`。
- 事实：`run.py:16` 读取的 `packet/source10-before.json` 在包内不存在；`run.py:36` 依赖包外绝对路径 `C:/Users/J00010385/AppData/Local/Temp/han-b13d-lead-gz6d1yr8/postgres-resource.json`，该文件当前不存在；`no-pgpass`、`no-service`、`no-sysconf` 三个 sentinel 文件也不存在；`run.py:15` 的 `dest.mkdir()` 未使用 `exist_ok`，同 label 目录已存在会失败。
- 影响：不能仅凭该证据包在不读取包外状态的情况下重跑 PG 两轮；消费 `report.md:41` 写“`PGPASSFILE/PGSERVICEFILE/PGSYSCONFDIR` 指向空文件”与实际不存在不符。已保存的 stdout/leaves/result/processes 仍可离线复算，故不推翻本子门禁证据。
- 违反依据：证据可复核性与交付完整性要求；不构成产品代码或 no-hardcode 违规。
- 触发场景：将证据目录复制到独立环境，清空包外 Temp 后按 `run.py` 重放。
- 修复建议：随包冻结 `source10-before.json`、隔离 sentinel 与 PG resource 快照；或明确记录外部执行目录与重建步骤；`mkdir(parents=True, exist_ok=True)`。
- 验证条件：仅在证据包及明确登记的外部依赖下能够生成同 argv、同叶子集合、可复核 exit/哈希的结果。

### S15-PR-F002 — LOW，非阻断：消费报告命令块与 argv 非逐字一致

- 位置：`.spec/wip/article-management/evidence-s1.5/producer-real-r01/report.md:29-30`。
- 事实：代码块把 `-run` 写成 shell 展示形式 `'^(TestReplayPublicationPostgres|TestDeletionReplayPostgres)$'`（带单引号）；`commands.json` 与 `result.json.argv` 中该元素是无引号的单参数 `^(TestReplayPublicationPostgres|TestDeletionReplayPostgres)$`。语义一致，但不是字节逐字一致。
- 影响：人工按 shell 复制执行可得到等价参数，但“命令逐字一致”审计会出现表象差异；不影响原始产物与 58/0/0 复算。
- 违反依据：证据文字与实际 argv 的可复核一致性。
- 修复建议：标注“shell 展示形式，argv 见 commands.json”，或直接列 argv 数组。
- 验证条件：报告命令块与 `commands.json.<run>.argv` 做字符串比较时口径明确且无歧义。

## 4. 已核实事实

- `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 固定；评审开始与结束前工作树 `git status --short` 均为 291 行。
- 来源 `index.json` 106 登记 / 107 实物 / 0 missing / 0 hash-or-bytes mismatch / 0 unregistered payload。
- 两轮 `exit_code=0`、`wait_returned=true`，各 58 个叶子全 PASS，目标 `Publish` 与 `ReadCurrent` 两轮均 PASS。
- 正常与 race 两轮均无 `-overlay`；race 轮带 `-race`。
- 进程 job 全树收口：normal 9/9、race 45/45，active 0、observer errors 0、PipesJoined true、parent exit 0。
- `internal/article/replay_evidence.go` 当前 `7C14E74DCEF68A1B8067DCEC2BF50BA4D1FA9BA165AEE5C52B3828D0D60417FB` / 34397 B。
- PG preflight exit 0 且原始日志为 18.6、`127.0.0.1:51934`、`han_b13d_article`。
- 消费登记 11 项上游哈希/字节一致，两轮结果一致，未证明范围与 open gaps 明确。

## 5. 未验证边界

- **未复跑（NOT RUN）**：环境存在 `HAN_TEST_POSTGRES_DSN`（长度 68，未打印）且 `psql` 为 PostgreSQL 18.6；但未连接数据库确认目标就是现役测试库，且测试 helper 会创建并保留 schema，来源报告也明确本轮不 DROP/不清理。为遵守“不创建/删除数据库、不 DROP 既有对象”的限制，未执行两轮 `go test`。此限制不降低对已保存原始产物的离线复算。
- `source-protection.json` 177 项仅证明 B13d entry→exit 期间未变；当前工作树有 28 项已漂移，因此不声称 177 项全量当前稳定。
- 未重新验证当前 PG 实例仍为 18.6；18.6 与端点来自保存的 preflight 原始日志。
- 本证据不覆盖真实 Journal/HMAC/keyring、旧备份恢复、Case18 全局恢复隔离、应用角色 ACL 载体、部署 E2E 或 production readiness。
- 工作树为快照语义；消费 index 明确记录 HEAD 不含 Go 代码、不可由 `git checkout` 重建，本评审以 SHA256/字节而非 checkout 重建核对。
- 本结论只判第 7 子门禁的证据消费可核性；**不宣称 S1.5 整卡 PASS，不宣称 S8.2 或 Case18 PASS**。人的业务接收或 ADR-0028 下的代理接收仍需单独登记。

## 6. 复审条件

- 修复 F001 或明确接受“证据包依赖外部执行目录”的限制。
- 修正 F002 的报告命令口径，或明确其仅为 shell 展示。
- 若 10 源、`replay_evidence.go` 或任何两轮原始产物哈希发生变化，应重新按本报告范围复审，不复用旧结论。
- 本结论为 **AI 代理评审，非真人签字**。