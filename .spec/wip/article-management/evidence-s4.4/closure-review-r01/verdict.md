# S4.4 整卡关闭条件复核 — Verdict（closure-review-r01）

**结论：PASS WITH MINOR**（blocker = 0）

- 对象：`article-management-S4.4`；工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 角色：非作者独立 Reviewer（add-reviewer / add-code-review）；**AI 代理评审，非真人签字**（授权来源：ADR-0028 + 用户 2026-10-31 授权原文）
- 判据：`task-execution-workflow.md` §10.3；ADR-0028 约束 2/3/4/6

## 判定要点

1. S4.4 原阻塞“生产 `main.go` 未调用 `NewReplayPublicationPorts`”**已解除**：seal 位于真实 production composition（`runComposed`→`composeBusiness`→`newArticleReplayProvider`→`newRecoveryRootAssembly`），`recovery_runtime.go:611-615` 实际调用且错误阻断启动；`replaySlot.source` 在 assembly 前赋值（`main.go:3330-3331`）；`articleEvidence` 单一 Store 复用，`NewPostgresReplayEvidenceStore` 仅 nil 保护回退。
2. 关闭条件 1-5、7 有当前修订真实 PG 证据；条件 6“旧备份恢复”**仅模拟 fixture 子集**；条件 8（inventory/RPO/RTO 真实演练）按卡面 `:1591` 明文不关闭；条件 9 独立 Review 已通过（有界）。
3. 四个被评审对象哈希与卡面/索引逐项一致；`sim-closeout-r01` 索引与实测一致；Lead 复跑日志 `ok 20.478s`、FAIL=0/SKIP=0/DATA RACE=0。

## Findings

| ID | 级别 | 状态 | 摘要 |
|---|---|---|---|
| S44-CR-F001 | MEDIUM | OPEN（不阻断关闭） | `harness-publication-r01/index.json` 登记旧哈希/字节且 `not_proven` 断言已失效，无 `superseded_for_current_tree` 指针；建议 Lead 追加 superseded 指针/新索引段 |
| S44-CR-F002 | LOW | OPEN（已披露边界） | `productionArticleComponents.gate` 无生产消费者、`recovery.NewCoordinator` 生产 0 调用、root 未注入 consumer（`recoveryClosedConsumer`）；生产恢复重放 HTTP 路径未接通 |
| S44-CR-F003 | LOW | OPEN（边界） | 无 recovery 配置时 seal 不执行且无断言（`main.go:633`） |
| S44-CR-F004 | LOW | OPEN（残余） | 生产 `validateReplayPublicationSnapshot` 不重算 hash；hash 钉住仅固定投影 |
| S44-CR-F005 | LOW | OPEN（已披露 `S15-SIM-F003`） | `CheckPhase` 当前三路径不可达 |

无 HIGH；minor-01/minor-02/`S15-SIM-F001`/`F002` 已 CLOSED，`S15-SIM-F003`/`F004`/`R02-N01`/`R02-N02`/`B13H-GATE-R02-L02` 均非阻断且已披露，经复核成立。

## 关闭前置条件（Lead 须写入关闭记录，本 Reviewer 不改文件）

- 明确“旧备份恢复”仅模拟子集，真实演练部分按 `:1591` **不关闭**；inventory/RPO/RTO 缺失即真实演练 `NOT CLOSED`。
- 登记 §未验证边界：真实 POSIX Journal/fsync/flock、真实 keyring/HMAC/HSM、真实旧备份与 RPO/RTO、真实 inventory、identity 双 namespace/MarkConsumed、真实部署/E2E、Case18、`enter`/`install`/secret candidate、生产重放入口、进程级唯一 runtime/Store/pool、性能。
- 处置 `S44-CR-F001`（追加 superseded 指针）；不得宣称 production readiness。

## 未验证边界（摘要）

真实 POSIX Journal/fsync/flock；真实 keyring/HMAC/HSM；真实旧备份恢复与 RPO/RTO；真实 backup inventory；identity 双 namespace/MarkConsumed/联合开流；真实部署/E2E；Case18 `NOT RUN`；`enter`/`install`/recovery secret candidate（`ErrAdmissionClosed`）；生产重放入口（S44-CR-F002）；无 recovery 配置时 seal 行为；进程级唯一 runtime/Store/pool 与性能。

## 证据

- 实测哈希与逐条关闭条件表：本目录 `report.md`
- PG 执行证据（未自行复跑，遵守串行约束）：`.spec/wip/article-management/evidence-s4.4/lead-closure-r01/s44-lead-rerun.log`（`ok 20.478s`，33 PASS / 0 FAIL / 0 SKIP / 0 DATA RACE）
- 本轮自跑：`go build ./...`、`go vet ./cmd/han-server`、`go vet -tags=integration ./cmd/han-server`、`go vet -tags=integration ./test/integration/recovery` 均 exit 0；`gofmt -l` 空。