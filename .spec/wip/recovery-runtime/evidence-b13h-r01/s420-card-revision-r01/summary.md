# S4.20 任务卡修订（R01 响应）— 变更摘要与一致性核对

## 1. 状态

- 状态：`READY_FOR_LEAD_APPLY`（只产出替代文本，**未修改任何权威文件**）。
- 本会话唯一写入：`.spec/wip/recovery-runtime/evidence-b13h-r01/s420-card-revision-r01/`（4 个文件）。
- 未编辑 `tasks.md`、Plan、设计、ADR、Role/Rule/Skill 或源码；未执行 Git 写操作；未运行 `go build/test/vet`、PG、HTTP。
- 应用前必须由 Lead 重新计算 `tasks.md` 哈希；若不再是 `F7FF8F63…`，先重核 `card-replacements.md` 的匹配次数。

## 2. 基线（本会话实测）

| 对象 | SHA256 | 字节 | 行数/备注 |
|---|---|---|---|
| `tasks.md` | `F7FF8F633CB50BA501DD5677B3F2F480D977F265A38035A4FE92CF9AD0D1D2F0` | 703692 | 2920 行，BOM=False，全 CRLF；mtime 2026-10-31T17:14:18.5496781+08:00 |
| Plan `ir-01-content-community-batches.md` | `1AB1474D1B330184D5FD80E998410FFDA5B3C1A1FF0B0B7BFBAB2450F498D268` | 487344 | 1615 行，BOM=False，CRLF=1611；LF-only 行：1596/1598/1599/1600 |
| R01 评审 `report.md` | `F516AACEC0A853CBB410DA3BD31CFF8B5AD498A737173D7A147D7C7DD0884BF6` | 23342 | 与输入一致 |
| `cmd/han-server/main.go` | `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` | 171448 | mtime 2026-10-31T16:36:42.7242751+08:00；与 `tasks.md:1231` 引用一致 |
| Git HEAD | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` | — | 分支 `main`；`git status`：`tasks.md`=` M`，`cmd/` 未跟踪 |

### 2.1 输入哈希漂移（必须记录）

- 输入声明 `tasks.md` 为 `4B56A026…`/701371 bytes/2919 行；实测已变为 `F7FF8F63…`/703692 bytes/2920 行（+2321 bytes，+1 行）。
- 差异已隔离：仅 `:1849` 新增一行（`article-management-S8.2` 卡的“2026-10-31 恢复主线承接说明”），旧 `:1850` 起整体下移 1 行。
- **S4.20 卡文本逐字节未变**：anchor 由 `:2856` 移至 `:2857`，卡为 `:2857`–`:2919`；`:2920` Lead 承接说明保持原样、未纳入替换。
- 结论：漂移不影响修订内容；`card-replacements.md` 的行号与唯一匹配均以 `F7FF8F63…` 为准。

## 3. 实际读取

- Role/Skill/Guidance：`.codex/agents/add-task-planner.toml`、`add-task-decompose/SKILL.md`、`add-batch-plan/SKILL.md`、`task-execution-workflow.md`（§4/§7/§10）、`coding-standards.md`、`no-hardcode-policy.md`、`multi-agent-coordination.md`。
- 输入：R01 `report.md` 全文；`tasks.md` S4.20 卡与 B13H 段；Plan B13H 段（`:1587`–`:1615`）、批次表 `:1302`–`:1310`、映射表 `:1331`–`:1332`；ADR-0027/ADR-0028 索引；`s419-lead-narrowing-decision-r01.md` 写法样板。
- 只读代码核对（本会话复核）：`cmd/han-server/recovery_runtime.go:465`（enter/install 返回 `ErrAdmissionClosed`）、`:523`（recovery secret candidate 未交付）、`:531/:558`（closed reader/port）；`internal/recovery/runtime.go:917`（`activate` 定义）、`:1917`（`ExitRecovery`）、`:1943`（`activate` 调用）；`cmd/han-server/recovery_runtime_integration_test.go:165`（`TestRecoveryRuntimeProductionChain`）；`cmd/han-server/main.go:3290`（第二个 `NewPostgresReplayEvidenceStore`）。

## 4. 产出文件

| 文件 | SHA256 | 字节 | 说明 |
|---|---|---|---|
| `card-proposal.md` | `B9A89E9D6F5BB26EECEB06FE48A2DAF13DEE3402C3C34C056ED3EDA4528776C7` | 16474 | 修订后整卡（71 行，`:2857`–`:2919` 等价内容） |
| `card-replacements.md` | `AB27FD0826E8246C5DCC0F77610FD205147BA9A0D37535866367D087D2C07463` | 27433 | 19 组精确 旧文→新文（含 2 处插入） |
| `plan-proposal.md` | `96942EFEA842703E5810BBB5E03ADE8867FFEF3E7EFD975EB76393E46DA06631` | 8276 | Plan 2 组替换 + 1 组未授权可选项 |
| `summary.md` | 见最终回报（自引用不内嵌） | — | 本文件 |

- 三个已产出文本文件均为 UTF-8 无 BOM、CRLF、无孤立 LF（`card-proposal.md` 71 行；`card-replacements.md` 260 行；`plan-proposal.md` 47 行）。

## 5. R01 F001–F008 处置

| 编号 | R01 问题（摘要） | 本修订落实 | 位置 | 结果 |
|---|---|---|---|---|
| F001 | 范围把 enter/install 与 secret 混入“完整恢复” | 卡面改“resume 恢复运行完整完成”；显式声明 `recovery_runtime.go:465`/`:523` 保持 closed 占位、不在本卡；登记“后续承接任务未登记（unknown）” | 卡 `:2859`、`:2861`、`:2863`、`:2867`、风险 leaf、关闭条件 | 已落实 |
| F002 | `main.go` 共享写窗无硬门禁 | START 前三重门禁：S1.5 ACL R02 非作者独立复审 PASS + S1.5 Lead Gate PASS + S1.5 writer 明确 STOP/`main.go` 写窗冻结；实施前后记录 SHA256/字节/mtime | 卡 `:2917`（并新增风险 leaf） | 已落实 |
| F003 | `canonical_recovery_replay_same_result` 不可执行 | 新增 6 行“可执行 oracle”：持久字段清单、同 run/同 live permit/同 `PermitBinding` 的第二次 replay、零额外写入审计；写 `internal/identity/*`/`internal/article/*` 即 STOP | 卡命令块后（插入块） | 已落实 |
| F004 | 新旧测试同名 | 新测试 `TestRecoveryProductionChainComplete`；旧 `:165` `TestRecoveryRuntimeProductionChain` 原名/断言/语义不变；`-list`/`-run`/关闭条件同步 | 卡 `:2876`、`:2877`、命令块、`:2913`、`:2919`；Plan P-01/P-02 | 已落实 |
| F005 | 缺 `go build`；未跟踪文件“无 diff”不可证 | 命令块加 `go build ./...`；关闭条件要求未跟踪文件以 SHA256+字节+全文快照核验 | 卡命令块、`:2913` | 已落实 |
| F006 | S1.5 被当可消费证据 | 改为“条件性未来引用”，仅 ACL R02 PASS + Lead Gate PASS 后可消费 | 卡 `:2863`、`:2866`、`:2882`、`:2915`；Plan P-02 | 已落实 |
| F007 | `runtime.go:1917` 误标 `activate` | 更正为 `ExitRecovery`；`activate` 改引 `:917`/`:1943` | 卡 `:2866` | 已落实 |
| F008 | 未冻结 `main.go` SHA、未要求 S4.19 两层复跑 | START 冻结当前 SHA；关闭前在当前 SHA 重跑 `TestRecoveryMainBusinessDispatcher` 等两层并记录 | 卡 `:2913`、`:2917`、风险 leaf | 已落实 |

## 6. 一致性核对

- **写集**：仍为 4 个文件（3 改 + 1 新增），未引入第五文件或 `internal/identity/*`、`internal/article/*` 写权。
- **依赖**：requires 仍为 S4.16/S4.19/S8.3 三条；未新增 requires 边；S1.5 仅“条件性未来引用”。
- **批次**：仍唯一归属 B13H；B13H 17 个唯一文件/21 个声明槽不变；全图 76 节点/324 边、runtime 切片 17 卡/70 边不变。
- **已关闭卡**：S4.16/S4.17/S4.19/S8.2/S8.3 的 checkbox、日期、关闭记录、requires、文件所有权均未触碰；未新增任务。
- **卡面状态**：checkbox 保持 `[ ]`、无完成日期；原 `:2920` Lead 承接说明原样保留（应用后移至 `:2928`）。
- **行数**：卡 63 行→71 行（净 +8：命令块 10→11、oracle 插入 6、风险表插入 1）。
- **替换可验证性**：19 组旧文在当前 `tasks.md` 均唯一匹配（match=1）；内存模拟全部替换后，`:2857` 起 71 行与 `card-proposal.md` **逐行 0 diff**，其后紧接原 `:2920` Lead 说明。
- **行尾**：`tasks.md` 全 CRLF；Plan `:1332` 为 CRLF、`:1600` 为 LF-only，替换文本已在 `plan-proposal.md` 中分别注明。
- **禁止项**：未执行 Git 写、未运行任何 Go/PG/HTTP 命令、未删除/移动/覆盖既有内容、未写入凭据。

## 7. 未决与 unknown

- `article-management-S1.5` ACL R02 非作者独立复审与 Lead Gate 均未完成 → S4.20 当前不能 START。
- oracle 在 4 文件写集内的可执行性尚未验证（需实施期真实证据；若越界即 STOP）。
- enter/install/recovery secret candidate 的后续承接任务未登记（unknown）。
- 设计仍 draft、人的精确接纳 unknown；R01 全部验收命令 NOT RUN；修订后非作者 Planning Review 未进行。
- `cmd/han-server/` 整体未跟踪，无 commit diff 基线，只能 SHA256+快照核验。
- Plan 外层批次表 `:1309` 仍写“production完整恢复链”，超出本会话授权范围未改（见 `plan-proposal.md` 可选项）。

## 8. Lead 精确执行清单

1. 重算 `tasks.md` SHA256，确认等于 `F7FF8F63…`；若变化，先重核 19 组旧文唯一性。
2. 按 `card-replacements.md` 的 R01-01…R01-19 应用（均为文本锚点、唯一匹配，顺序不影响结果；建议按源行号降序执行，插入项 R01-11/R01-12 在锚点行之后插入）。
3. 应用后核验：卡 `:2857`–`:2927` 共 71 行与 `card-proposal.md` 一致；`:2928` 起为原 Lead 说明；文件仍 UTF-8 无 BOM、CRLF。
4. 按 `plan-proposal.md` 应用 P-01（`:1332`，CRLF）与 P-02（`:1600`，保持 LF-only）；行数仍 1615。
5. 未获另行授权时**不要**修改 Plan `:1309`。
6. 记录应用后两文件 SHA256/字节/行数；因卡面已变更，需重新发起非作者 Planning Review。

## 9. 关闭条件一致性核对表

| R01 决策 | 卡内关闭条件/风险 leaf | 一致性 |
|---|---|---|
| F001 resume 范围 | `ResumeCoordinatorRecoveryCompleteAndOpen`、`:2913` 关闭条件、`:2919` unknown | 一致；不声称 enter/install/secret |
| F002 硬门禁 | `MainWriteWindowFrozenAndS419Reverified`、`:2917` | 一致；任一不满足即 STOP |
| F003 oracle | `CanonicalReplaySameResult`、oracle 6 行、`:2913` | 一致；持久字段+零写入 |
| F004 测试命名 | `:2876`/`:2877`/命令块/`:2913`/`:2919` | 一致；旧测试不动、新名唯一 |
| F005 构建与未跟踪核验 | 命令块 `go build ./...`、`:2913` | 一致 |
| F006 S1.5 条件化 | `:2863`/`:2866`/`:2882`/`:2915`、Plan P-02 | 一致；不作硬 requires |
| F007 引用更正 | `:2866` `ExitRecovery`/`:917`/`:1943` | 一致 |
| F008 SHA 冻结与两层复跑 | `:2913`/`:2917`/风险 leaf | 一致 |