# Case18 当前 revision 执行证据 r01：登记补正独立复审 r02

**AI 代理评审，非真人签字。**

## 0. 对象、修订与范围

- objectType：执行证据包的补正复审（承接 `case18-current-r01-review-r01` 的 `C18-R01-REV-F001`/`F002`）。
- 固定对象：`.spec/wip/article-management/evidence-s8.2/case18-current-r01/registration-correction-r01.md`，声明 4604 B / SHA256 `CFDCE04CA654DA813129C9BA0929934EE23C35A7ED5989309E40258BD61EC84C`。
- 原证据包：同目录 `report.md`、`index.json`、8 份 run 日志、`source-hashes-before.txt`、`source-hashes-after.txt`。
- 工作区：`D:\AID\VSCode\Codex\Han`，branch `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（开工核对，收工复核）。
- 范围：只判定 F001/F002 的补正闭环、补正文件“结论不变项”与是否引入夸大；不重开 `C18-COV-F001`、`F004` 的技术事实，不扩大到 S3.4、S1.5/S8.2 整卡或其他 finding。
- 限制：本轮不执行 `go test`、不连接/写入 PostgreSQL、不做 Git 写、不改任何既有文件；唯一写入为本评审目录。

## 1. 两轮实际检查

### 第一轮：对象冻结与静态核对

- 复算补正文件 SHA256/字节，确认与声明一致。
- 复算原 `report.md`、`index.json`、8 份 run 日志、两份 `source-hashes` 的 SHA256/字节，确认与上轮所见一致，补正声明“原文件未改动”成立。
- 读取上轮 r01 `report.md`/`verdict.md`、补正文件、`tasks.md` 的 S1.5（`:1232`）与 S8.2（`:1877-1878`）状态。
- 从两份 integration 日志独立提取 `han_admin2_*`/`han_s33_*`，去重后与补正清单逐项比对。
- 核对 `internal/recovery/runtime.go:27-32`、`:1015` 与 `internal/platform/recovery_guard.go:59-64` 的代码事实。

### 第二轮：闭环判定与风险复核

- 判定 F001：补正清单数量、名称、漏项/多项；判断“只追加更正”是否足以关闭原 MEDIUM，是否需要改写原 `report.md`/`index.json`。
- 判定 F002：六态词表、guard 放行条件、`internal_ready`/`unknown` 归属，以及 `C18-COV-F002` 是否仍 open。
- 核对补正文件“结论不变项”：F001 finding 级 CLOSED、F004 整体 open、F002/F003/F005 open、S1.5/S8.2 仍 `[-]`、`blocked-upstream` 不解除、Case18 不具备关闭评估条件。
- 检查夸大表述与越界写入；收工复核 HEAD、工作树与评审文件哈希。

## 2. 逐项核验结果

### 2.1 补正文件与“原文件未改动”：PASS

补正文件实算：

| 文件 | SHA256 | 字节 | 对账 |
|---|---|---:|---|
| `registration-correction-r01.md` | `CFDCE04CA654DA813129C9BA0929934EE23C35A7ED5989309E40258BD61EC84C` | 4604 | 与声明一致 |

原证据包现值（与上轮 r01 所见逐项一致）：

| 文件 | SHA256 | 字节 | 与上轮一致 |
|---|---|---:|---|
| `report.md` | `C54E4EAB1AF5761F2F250CE51A1E0B4EAD63A22AC51B02AC60E99D20F46EEC3C` | 7384 | 是 |
| `index.json` | `609CFBF075078808E1483B9F20E5EDBF5395AB426652EB756054CEF4E36DFCD2` | 6640 | 是 |
| `source-hashes-before.txt` | `7B78FD435F5A65CBFEA2F9ECEC829A4BCE5DE712FBB24AF726AE9692D6F1F635` | 1654 | 是 |
| `source-hashes-after.txt` | `6CF304C63CFF1B6B63D37540BB71896D2B339F6BEC71F8E11C30999545F7F334` | 829 | 是 |
| `run-root-normal.log` | `D58A8DF3F1949568D71A43DBFE5DA687B91DBBE781B4349C97B060CFD30F48B4` | 18999 | 是 |
| `run-root-race.log` | `5A2618FEE27651C19A1716958D75049F108BA2EFEBC44FC6AB24B4133079EE67` | 18999 | 是 |
| `run-uc-normal.log` | `4C7D1CC6655E1AF2E265A6EE2C09FE716FA61248C5AB4F9EE88899175BFCC0EA` | 18001 | 是 |
| `run-uc-race.log` | `9B3323C1E9F3764C145751B832FE3FEAFE3A4FB4A3A313C50A3FC6CC223931D7` | 18001 | 是 |
| `build.log` | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` | 0 | 是 |
| `vet.log` | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` | 0 | 是 |
| `run-adminquery-pg.log` | `F51D2F4F942C067426285A0D4DA318B63E1A6B66FDA4F58D64FA5390F52F17E3` | 64739 | 是 |
| `run-session-sql-pg.log` | `B000D2141C6E61D5C2A7732EF03B98FF83A88FE020EC0F579D7A1231D60F4E1F` | 8447 | 是 |

补正声明“只追加更正文件，不改写原 `report.md`/`index.json`/8 份 run 日志的任何字节”与实测一致。

### 2.2 `C18-R01-REV-F001`：CLOSED

独立提取与比对：

```text
run-adminquery-pg.log 唯一 han_admin2_*：28
补正文件唯一 han_admin2_*：            28
补正缺失：0
补正多项：0

run-session-sql-pg.log 唯一 han_s33_*：3
补正文件唯一 han_s33_*：               3
补正缺失：0
补正多项：0
```

补正清单（`registration-correction-r01.md:12-53`）与两份日志的唯一 schema 集合完全一致，数量为 28 + 3，正好覆盖上轮指出的 29 个少计对象。日志侧保留证据：`run-adminquery-pg.log` 中 `created=true` 与 `retained=true` 各 28 次；`run-session-sql-pg.log` 中 3 个 schema 均为 `created_here=true retain_policy=always`、`workers_joined=true pools_closed=true`。补正文件 `:54` 明确写“原登记值应作废，以本清单为准”。

**判定：CLOSED。** “只追加更正”足以关闭该 MEDIUM：补正文件位于同一证据目录、名称明确、清单完整且逐项可核，并显式作废旧登记值；改写原 `report.md`/`index.json` 反而会破坏既有 SHA256/字节绑定和“原证据未改动”的审计链。原 `index.json` 仍是旧值，是原始证据的不可变快照，不影响补正文件的更正效力。

非阻断观察：补正文件 `:54` 用 `created=true retained=true` 概括全部 schema，但 session 日志的字面字段是 `created_here=true retain_policy=always`；语义相同（自建且保留），建议后续补正措辞区分两类日志字段。另，补正文件尚未登记进 `index.json` 的 artifacts/runs；若后续存在只读 `index.json` 的自动消费者，建议在新版索引中加补正文件指针或哈希。两者均不改变本轮 CLOSED 判定。

### 2.3 `C18-R01-REV-F002`：CLOSED

代码事实核对：

- `internal/recovery/runtime.go:27-32` 的 runtime 状态常量为六个：`closed`、`checking`、`public_ready`、`faulted`、`draining`、`stopped`。补正文件 `:60` 列出同一组六态，原报告漏记的 `checking` 已补回。
- `internal/platform/recovery_guard.go:59`：`Readiness` 仅在 `err == nil && observation.State == "public_ready" && ctx.Err() == nil` 时返回 200 `ready`；其余情况到 `:64` 返回 503 `unavailable`。补正文件“只对 `public_ready` 放行”的核心判断成立。
- `internal/recovery/runtime.go:1015`：`internal_ready` 是 observation 字面量，不是 runtime 状态常量；补正文件 `:62` 的归属正确。
- `unknown` 来自测试替身（如 `s419ErrAdmissionUnknown` 与 fake runtime 的 `unknown` 状态），不是 runtime 常量；补正文件 `:62` 的归属正确。
- `C18-COV-F002` 仍 open：矩阵仍使用 `newS419RecoveryRuntime` fake runtime 与零值 `pgxpool.Pool`，没有真实 runtime/PG/双系统全状态证据；本轮补正只修登记与措辞，不新增执行证据。

**判定：CLOSED。** 词表更正成立，且没有把 `C18-COV-F002` 从 open 改成关闭；补正文件 `:70` 明确保持 open。

非阻断观察：补正文件 `:61` 写“其余一律走 `writeRecoveryHTTPClosed`”，对 `Readiness` 端点而言实际走 `writeRecoveryHTTPProbe(..., 503, "unavailable")`；`writeRecoveryHTTPClosed` 用于受保护请求的 guard 拒绝路径。核心“仅 `public_ready` 放行”结论不受影响，建议后续把 probe 与 closed envelope 两条路径分开表述。

### 2.4 补正文件“结论不变项”：PASS

| 补正项 | 与本评审原判定 | 结论 |
|---|---|---|
| `C18-COV-F001` 仅 finding 级 CLOSED | 一致 | PASS |
| `C18-COV-F004` Session 侧已覆盖、整体 open | 一致 | PASS |
| `C18-COV-F002` 保持 open | 一致 | PASS |
| `C18-COV-F003` 保持 open | 一致 | PASS |
| `C18-COV-F005` 保持 open | 一致 | PASS |
| S1.5 与 S8.2 仍 `[-]` | 与 `tasks.md:1232`、`:1877` 一致 | PASS |
| S8.2 `blocked-upstream` 不解除 | 与 `tasks.md:1878` 一致 | PASS |
| Case18 不具备关闭评估条件 | 一致 | PASS |

### 2.5 边界与夸大表述：PASS

补正文件 `:73-77` 明确：只更正登记与措辞、不新增执行证据；不证明真实部署 E2E、真实 SMTP/approved-provider、双系统接线、完整恢复后生产开流、`Enter`/`Install` 开流、真实 Journal/HMAC/keyring、旧备份恢复、容量/P95、production readiness 或人的业务接收。未发现“Case18 通过”“S8.2/S1.5 已关闭”“production readiness”“人的接收已完成”等表述；补正文件 `:71` 反而明确 Case18 关闭条件不具备。

### 2.6 越界与执行限制：PASS

- 本轮未执行任何 `go test`，未连接/写入 PostgreSQL，未 DROP/清理 schema。
- 未做 Git 写；未修改/删除/移动任何既有文件；未改写原证据包或上轮 r01 评审目录。
- 唯一写入为 `.spec/wip/article-management/evidence-s8.2/case18-current-r01-review-r02/` 下的 3 个文件。

## 3. Findings

**无阻断 finding。** 原 `C18-R01-REV-F001` 与 `C18-R01-REV-F002` 均可登记 CLOSED。

非阻断观察（不改变 CLOSED，不新增 finding ID）：

1. 补正 `:54` 对 session schema 的字段概括应写成 `created_here=true retain_policy=always`，不要与 admin 日志的 `created=true retained=true` 混写。
2. 补正 `:61` 应区分 `Readiness` 的 503 probe 路径与受保护请求的 `writeRecoveryHTTPClosed` 路径。
3. 补正文件尚未进入 `index.json` 的 artifacts/runs；若后续依赖机器索引，建议加指针/哈希，避免只读旧索引的消费者遗漏更正。

## 4. 最终判定

- `C18-R01-REV-F001`：**CLOSED**（28+3 清单与日志完全一致，只追加更正足以关闭）。
- `C18-R01-REV-F002`：**CLOSED**（六态词表、guard 条件、`internal_ready`/`unknown` 归属均与代码一致，F002 仍 open）。
- `C18-COV-F001`：**可正式登记 CLOSED（仅 finding 级）**；不代表 S8.2/S1.5 整卡或 Case18 关闭。
- `C18-COV-F004`：整体仍 open（Session 侧已覆盖，AccessGeneral 未覆盖）。
- `C18-COV-F002`/`F003`/`F005`：保持 open。
- Case18 关闭评估条件：**不具备**。

**整体复审 verdict：PASS WITH MINOR。**

**AI 代理评审，非真人签字。**