# identity-auth-S4.20 Planning Review R02 — STOP 报告（冻结输入漂移）

- 状态：**STOP — 未签发结论**（PASS / PASS WITH MINOR / CHANGES REQUIRED 均不适用）
- 评审类型：Planning Review R02（不是代码评审；本报告不声称任何代码、测试、build、E2E 通过）
- 角色：add-reviewer（非作者；亦为 R01 报告作者）
- Agent actualID：unknown（本会话未取得可回报的委派 ID，不伪造）
- 评审时间：2026-10-31 17:46–17:53（Asia/Shanghai）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（实测）
- 写入边界：本文件为本轮唯一写入；未改 `tasks.md` / Plan / 源码 / `deploy/` / 其他证据；无 Git 写；未运行 go/PG/HTTP/service/restore

停止依据（用户本轮硬约束原文）：「若发现输入哈希与上述不一致，先 STOP 并回报，不要按漂移版本评审。」
实测 `tasks.md` 与冻结声明不一致，故本报告**只登记漂移取证与影响**，不给出 R02 结论。

## 1. 固定对象与实测哈希

| 对象 | 声明 SHA256 / 字节 | 实测 SHA256 / 字节 | 判定 |
|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `7C525A515698880F0C72AA0E4B1303FE804BD660BC108175D5A5AA82638FDD15` / 718312 | `6F47DF6FA10DE6D73B49BD3F0EA37B15A9D7D383CA4188B45987B2951510A488` / 722886（2934 行，mtime 2026-10-31 17:45:56.429） | **MISMATCH（+4574 B / +3 行）** |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `4BB31934E56A99EBC6B8486F5FBCF107688B0A6AB57E7B5EF2E1915FE1A6CE19` / 488742 | 同声明（mtime 17:40:23.485） | MATCH |
| `.add/project/knowledge/decisions/framework.md` | `E19A4189BED343FABD1D996F5ECD4229373FD34C487315D1082C2F841FA70983`（R01 输入声明） | `04E56C991510A5243F26E3EAB85555E081CC4DA8EBF5F44E1E25B080E9F4AAFA` / 126444（mtime 17:39:17.284） | **MISMATCH（与 R01 输入声明不同；与本轮 `tasks.md:1234` 自述一致）** |
| `cmd/han-server/main.go` | （F008 基线，未在本轮声明中冻结） | `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` / 171448（mtime 16:36:42.724） | 与 S1.5 ACL R02/R03 记录一致 |
| R01 报告 `.spec/.../s420-planning-review-r01/report.md` | — | `F516AACEC0A853CBB410DA3BD31CFF8B5AD498A737173D7A147D7C7DD0884BF6` / 23342 | 稳定 |
| `s420-card-revision-r01/summary.md` | — | `D4AA5FD2E3762B45F42B5A678271B71847941CAF380E5454AC8A9E3D35E57CF0` / 9894 | 稳定 |
| `s420-card-revision-r02/summary.md` | — | `88D475665261E07C6C3109C87418BE954EC02F8E2E2100ACA3F71F26BD317897` / 10863 | 稳定 |

说明：`framework.md` 的期望值来自 R01 输入声明（旧值），本轮未在冻结清单中重述；其当前 SHA 与本轮 `tasks.md:1234` 记录的自述值一致，但**该记录本身是新写入内容**，故本轮不据此判 PASS。

## 2. 漂移取证（可证明为纯追加）

- 行数：基线 2931 行 → 现 2934 行（+3）。
- 字节闭合：新增行实测 UTF-8 字节 `tasks.md:1234` = 3421、`tasks.md:2934` = 1147，合计 4568；加插入空行 CRLF 2B 与文件末尾追加行 CRLF 2B，正好 4574 B = 实测差值（722886 − 718312）。文件全 2934 行为 CRLF，无 BOM。
- 结论：除以下两条记录（及一条插入空行）外，`tasks.md` 其余内容字节未变；S4.20 卡正文与冻结基线**逐字相同**，仅整体下移 2 行（基线 `:2862` → 现 `:2864`）。
- 漂移**发生在本评审会话进行中**：本次会话首次实测为 17:40 基线的 718312 B，17:46:26 复测已为 17:45:56 的 722886 B。

新增记录一 —— `tasks.md:1234`（S1.5 段，插入在 Gate 记录 `:1233` 之后）：

> **2026-10-31 S1.5 ACL 三条 LOW 收口 + 证据脱敏（用户本轮“授权执行”下完成；ACL 子集可核收，整卡仍 `[-]`）** … **`S15-ACL-R03-F001`（明文 DSN）关闭**：… 就地脱敏，口令段替换为 `<redacted>` … **`R03-F002` 关闭（过时项）** … **`R03-F003` 关闭（文档更正）**：ADR-0029 升 r04 errata … **结论**：R03 报告 `7A236731…` 的 PASS WITH MINOR 被 Lead 核收为 ACL 门禁子集结论；S1.5 整卡保持 `[-]` …

新增记录二 —— `tasks.md:2934`（S4.20 卡尾追加）：

> **2026-10-31 START 门禁现状记录（不改本卡 checkbox/日期/requires）**：… ①…满足（`7A236731…`，三条 LOW 已按 2026-10-31 Lead 记录关闭…）；②…满足（`lead-gate.log` `D21DB1F1…`）；③「S1.5 writer 明确 STOP 且 `cmd/han-server/main.go` 写窗冻结」——事实层：… 已 completed，`cmd/han-server/main.go` 仍为 `1D47CC94…` … 是否据此宣告“写窗冻结”由 START 时 Main 与用户确认，Lead 不代判。本卡状态保持 `[ ]`、无完成日期、全部验收命令 NOT RUN …

## 3. 同窗口并发写入观测（S1.5 证据区仍在变动）

| 时间 | 对象 | 实测 | 说明 |
|---|---|---|---|
| 17:45:10.875 | `.spec/wip/article-management/evidence-s1.5/acl-r02/index.json` | `F442147B1E60948BA54C4826EA94E112CA06E5E330269B250ED1617ADEB82BAB` / 28108 | 与 `tasks.md:1234` 自述一致；原 `7921D68E…` 已被覆盖 |
| 17:45:14.867 | `.spec/wip/article-management/evidence-s1.5/acl-r03/index.json` | `8ED9FF3E9F8E0CB22691A32F1B394C40AD4A6FDFFCB9E0179525777379DDFCA5` / 22037 | 与 `tasks.md:1234` 自述一致；原 `E80EFD98…` 已被覆盖 |
| **17:46:48.357** | `.spec/wip/article-management/review-code-s1.5-acl-r03.md` | `78D8AA2318ED237B9141EECFCEC5372F355EB997387371BA1C0490730E47A238` / 23870 | 该文件为 v2 修订（自述原版 `7A236731…` / 76 行 / 24911 B）；**晚于 `tasks.md` 的 17:45:56** |

- `tasks.md:1233`（Gate 记录）与 `tasks.md:2934`（新记录）均以 `7A236731…` 指代 R03 报告，但当前同路径文件已是 v2（`78D8AA23…` / 23870 B）。**证据指针自 Gate 记录写入起即已失效**，且失效时点早于本轮新增记录的自述。
- 该 v2 报告第 142 行自述：`framework.md:911` 的 r04 errata 引用了其修订前 SHA `7A236731…`，需 Lead 更新引用；`decisions/README.md` 只按 finding 编号引用。
- 观测边界：仅据 mtime/SHA；写入者身份、是否已 STOP、后续是否继续写入均为 unknown（不推测）。

## 4. 对 R02 检查表的影响（为什么不能出结论）

1. **F002（START 三重硬门禁）**：门禁第①重的证据链在当前工作树不自洽（引用的 R03 SHA 非当前文件）；第③重（writer STOP / `main.go` 写窗冻结）仍未被任何人宣告满足，且 S1.5 证据区 17:45–17:46 仍在写入，属“写者可能仍活动”的反向迹象。按冻结口径评审会得到与事实不符的“①可核”结论。
2. **F006 / R02→R03 措辞一致性**：卡面 5 处（现 `:2868`、`:2887`、`:2924`、`:2926`、`:2930`）与 Plan `:1600` 均写“三条 LOW 已登记处置”，而新增 `:1234` 记录声明三条 LOW 已**关闭**并已脱敏；两处口径不同，且脱敏改动了 R03 报告所依赖的证据文件哈希。
3. **F008（`main.go` 前后哈希/字节/mtime + 变化即 STOP）**：`main.go` 本轮实测 `1D47CC94…` / 171448 B / mtime 16:36:42，与卡面记录一致；但该一致性建立在“写窗已冻结”这一未证明前提上，且同仓 S1.5 证据仍在写入。
4. 其余项（卡 `[ ]`、无完成日期、全部 NOT RUN、4 文件写集、requires 集合、B13H 17/21、未重开已关闭卡）在**基线字节**上核验通过（见 §5），但结论随之失效，须在重新冻结后复确认。

## 5. 基线核验结果（718312 B 基线，已被漂移取代，非 R02 结论）

行号 = 当前文件（= 基线 +2）；内容与基线逐字相同。

| 编号 | R01 级别 | 基线判定 | 关键证据（当前行号） |
|---|---|---|---|
| S420-PLAN-R01-F001 | HIGH | CLOSED（基线） | `tasks.md:2868` 范围限定“resume 恢复运行完整完成”；`:2866` 目标要求 `recovery_runtime.go:465`、`:523` 保持 closed 占位、“后续承接任务未登记（unknown）”；代码事实 `cmd/han-server/recovery_runtime.go:462–465`、`:522–524` 均返回 `ErrAdmissionClosed` |
| S420-PLAN-R01-F002 | HIGH | CLOSED（基线，仅规划层） | `tasks.md:2930` 三重 AND 门禁；`:2924` 风险 leaf 含“该项仍未满足” |
| S420-PLAN-R01-F003 | HIGH | CLOSED（基线，静态可执行性） | `tasks.md:2908`（字段清单）、`:2910`（零额外写入 + “不得仅以日志字符串、fake 返回值…代替持久结果比较”）、`:2883`（越界 STOP）。静态可构造性：`internal/recovery/coordinator.go:32` 收接口、`internal/recovery/runtime.go:205–223` 依赖为接口、`internal/identity/recovery_run_store.go:21` 具体类型同时实现 `RecoveryRunStore`+`AdmissionStore`、`internal/article/deletion_journal_anchor.go:24–30` 提供 `ReadVerifiedSnapshot`。**未运行验证** |
| S420-PLAN-R01-F004 | HIGH | CLOSED（基线） | `tasks.md:2881` 旧 `:165 TestRecoveryRuntimeProductionChain` 保名/保断言/保覆盖，新文件顶层 `TestRecoveryProductionChainComplete`；全仓 `rg` 无同名冲突 |
| S420-PLAN-R01-F005 | MEDIUM | CLOSED（基线） | `tasks.md:2894` `go build ./...`；`:2896`/`:2898`/`:2900` 三个专测（普通与 `-race`）；`:2902` 起两层 `go vet`；changed-file `gofmt -l`；未跟踪文件 SHA+字节+全文快照 |
| S420-PLAN-R01-F006 | LOW | CLOSED（基线） | `tasks.md:2868`/`:2887` 改为“条件性未来引用…整卡仍 `[-]`，不构成整卡 PASS…仍不作硬requires”；Plan `:1600` 同步 |
| S420-PLAN-R01-F007 | LOW | CLOSED（基线） | `internal/recovery/runtime.go:1917` 标 `ExitRecovery`，`activate` 见 `:917`/调用 `:1943` |
| S420-PLAN-R01-F008 | MEDIUM | CLOSED（基线） | `tasks.md:2930` 含实施前后 SHA/字节/mtime 与“变化即 STOP” |

## 6. Findings

### S420-PLAN-R02-F001 — 冻结输入在评审期间漂移（处置：STOP）

- 级别：HIGH（按用户停止条件处置为 BLOCKER）
- 证据：`tasks.md` 实测 `6F47DF6F…` / 722886 B / 2934 行 / mtime 17:45:56.429，声明 `7C525A51…` / 718312 B。漂移内容：`tasks.md:1234`、`:2934` 两条新记录（+4574 B，字节闭合可证为纯追加）。
- 影响：评审对象定义失效；F002/F006 的证据基座在评审期间被改写，任何 PASS/PASS WITH MINOR 都将引用过期事实。
- 建议：Lead 重新冻结（给出新 SHA/字节/行数），并声明该期间 `tasks.md` 无进一步写入；随后由本角色在稳定基线上出具 R02 结论。

### S420-PLAN-R02-F002 — S1.5 R03 证据链在窗口内被改写，Gate 记录指针失效

- 级别：HIGH
- 证据：`tasks.md:1233`、`:2934` 引用 R03 报告 `7A236731…`；当前 `.spec/wip/article-management/review-code-s1.5-acl-r03.md` = `78D8AA23…` / 23870 B / mtime 17:46:48（v2 自述取代 `7A236731…`）；`evidence-s1.5/acl-r02/index.json`、`acl-r03/index.json` 于 17:45:10/17:45:14 被覆盖（现 `F442147B…` / `8ED9FF3E…`）；v2 报告 `:142` 自述 `framework.md:911` 的 errata 仍引用旧 SHA。
- 影响：START 门禁第①重“当前 revision 非作者 PASS”的**可核对象**不唯一/不稳定；S1.5 证据区仍在写入，与第③重“写窗冻结”相互矛盾地同时成立。
- 建议：由 Lead 明确 S4.20 应消费的 R03 revision（v1 快照或 v2），修正 `tasks.md:1233`/`:2934` 的 SHA 指针与 `framework.md:911` 引用，再据稳定修订重核第①重。

### S420-PLAN-R02-F003 — “三条 LOW 已登记处置”与“三条 LOW 已关闭”口径并存

- 级别：MEDIUM
- 证据：卡面 `tasks.md:2868`、`:2887`、`:2924`、`:2926`、`:2930` 与 Plan `:1600` 写“三条 LOW 已登记处置”；新增 `tasks.md:1234` 声明 `R03-F001/F002/F003` 均已关闭并完成脱敏。
- 影响：不阻断实施，但同一事实两种状态并存，后续复审与关闭条件核收会产生歧义；“已登记处置”掩盖了证据文件已被改写的副作用。
- 建议：Lead 择一统一（建议提及“LOW 已关闭、证据已脱敏，R03 报告指针见 F002”），或在重新冻结时说明以 `:1234` 为准。

### S420-PLAN-R02-F004 — START 第③重仍未满足（如实登记，非卡面缺陷）

- 级别：LOW（信息项）
- 证据：`tasks.md:2924`“该项仍未满足”；`:2934` 明确“由 START 时 Main 与用户确认，Lead 不代判”；`main.go` 现 `1D47CC94…` / 171448 B / mtime 16:36:42。
- 影响：S4.20 当前不可 START；不构成规划缺陷（卡面已如实标注）。
- 建议：无卡面修改需求；START 前由 Main 取得显式 writer STOP 声明并记录。

## 7. 总交接结论

**STOP — 未签发 R02 结论。** 不判 PASS、不判 PASS WITH MINOR、不判 CHANGES REQUIRED。

理由：冻结输入 `tasks.md` 哈希不一致（S420-PLAN-R02-F001），且 S1.5 证据区在评审窗口内被第三方改写并致 Gate 记录指针失效（S420-PLAN-R02-F002）。按用户本轮停止条件，不在漂移版本上评审。

基线（718312 B）上 R01 的 F001–F008 逐条均呈 CLOSED 形态（§5），但该结论**已随漂移失效**，不得作为 R02 结果引用。

Lead 需完成（建议顺序）：

1. 声明 `tasks.md` 重新冻结的新 SHA/字节/行数，并确认评审期间无进一步写入。
2. 明确 S1.5 R03 的消费 revision（v1 `7A236731…` / v2 `78D8AA23…`），修正 `tasks.md:1233`/`:2934` 指针与 `framework.md:911` 引用。
3. 统一“LOW 已登记处置 / 已关闭”口径（F003）。
4. 明确 START 第③重是否已满足（writer STOP + `main.go` 写窗冻结）。
5. 上述稳定后，由本角色复跑 R02 检查表并出具 PASS / PASS WITH MINOR / CHANGES REQUIRED。

## 8. 不证明什么 / unknown

- 本报告不证明任何代码、测试、`go build`、`go vet`、`gofmt`、PG、HTTP、service、restore、E2E 通过；S4.20 全部验收命令仍为 NOT RUN。
- 不证明 S4.20 卡在漂移后版本的任何结论；§5 仅为已被取代的基线核验。
- 不证明 `oracle` 可运行：F003 仅静态判断可实现（接口可包装），未执行。
- 不证明 S1.5 writer 已 STOP、`main.go` 写窗已冻结、`main.go` 在评审期间未被改动（仅观测到 17:46 时点 SHA/mtime 与记录一致，且 S1.5 证据区仍在写入）。
- 不证明 `evidence-s1.5` 目录已无明文凭据（未扫描、且本报告不记录任何凭据）。
- 不证明设计已定稿或人已精确接纳接线：设计仍为 draft、人的接纳 unknown（卡面 `:2932` 自述一致）。
- 不证明 B13H 17 唯一文件 / 21 槽的图解正确性，仅核对与 Plan `:1309`/`:1332`/`:1600` 自洽（在未漂移的 Plan 上）。
- 未审计 `git status` 全部约 284 行未跟踪内容，仅确认 `cmd/han-server/` 为未跟踪、无 commit diff 基线。
- Agent actualID、S1.5 证据写入者身份、R03 v1→v2 变更授权来源：unknown。