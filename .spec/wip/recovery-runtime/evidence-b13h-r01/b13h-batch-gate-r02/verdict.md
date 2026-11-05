# B13H 批次 Gate 重跑 r02 — 独立结论

- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 对象：B13H 唯一 root 批次（identity-auth-S4.16 / S4.19 / S4.20），17 唯一文件 / 21 声明槽。
- 评审角色：非作者独立 Reviewer（add-reviewer / add-code-review）；不转述 Lead 或上轮结论。

## 结论

**PASS WITH MINOR**

无 HIGH、无 MEDIUM 新增；`B13H-GATE-F001`（HIGH）、`N01`（LOW）、`B13H-GATE-F002`（MEDIUM）均 **CLOSED**。残余 4 条 LOW 与 1 条观察均不阻断批次 Gate。

## 判定要点

- **F001 CLOSED**：`PublicReadyAdmitsSameSession`（`recovery_main_dispatcher_integration_test.go:30`，实现 `:179-281`）经真实 `recovery.NewRuntime` + `RunNormal` 进入 `public_ready`；断言 closed 503 `platform.recovery_in_progress` + Trace、下游/Session 零调用、acquire/release=1/0，`public_ready` 后 `/readyz` 200、同一 Session 同一请求 200、Article 正常调用、2/1，失败负控 `platform.database_unavailable`。薄包装仅计数；probe fake 有边界注释；无 `setState` 冒充。
- **N01 CLOSED**：全包 `rg publishPublicReady` 0 命中（exit=1）。
- **F002 CLOSED**：`tasks.md:2950` 已追加限定更正（不改 `:2949` 原文），d7 证据范围收窄、层 (b) 归 S4.19 返工、两层同修订通过前 Gate 不得 PASS；本轮条件已满足。
- **r04 两条 LOW 非阻断**：`S419-R04-F001`（泄漏断言 Trace 误命中）、`S419-R04-F002`（真实 runtime closed `/readyz` 分支未在本文件断言）已披露，不影响本 Gate 判定。

## 新 finding

- **B13H-GATE-R02-L01**（LOW）：子测试注册在顶层函数内（`:26-33`），删除子测试不会使定向测试失败；需靠评审清单防覆盖回退。
- **B13H-GATE-R02-L02**（LOW）：`main.go:3333-3339` 的 nil 回退分支缺运行期“无第二 Store/pool”断言，当前仅指针层证明。
- **观察 O01（非缺陷）**：Plan `:1599` 仍写 S4.19 `[x]`，与 `tasks.md:2820` `[-]` 不一致；Plan 自述不维护状态，建议 Lead 关闭后补指针。

## 复跑证据（本轮实测，exit 全 0）

- 17 文件 inventory：SHA256 + 字节 + mtime 与登记逐项一致；测试后 postrun 17/17 match，收尾 recheck **drift=0**；清单外写入 0。
- 两层同修订同跑：`TestRecoveryRuntimePostgres`（真实 PG）普通/race 各 7 条 pass 记录（6 具名 + 包级），`TestRecoveryMainBusinessDispatcher` 普通/race 各 12 条 pass 记录（11 具名 + 包级）；fail=0、skip=0、DATA RACE=0。
- `go build ./...`、`go vet ./cmd/han-server`、`go vet -tags=integration ./cmd/han-server`、17 文件 `gofmt -l`（0 行）均 exit 0。
- 环境：`postgresql-x64-18` Running；`HAN_TEST_POSTGRES_DSN` 已由 `HAN_DATABASE_DSN` 设置（值未记录）。

## Lead 可据以关闭 S4.19 的条件

1. 接受本报告与 `report.md` 的 PASS WITH MINOR，确认 F001/N01/F002 CLOSED、无 HIGH/MEDIUM。
2. 在关闭时点复核：`cmd/han-server/recovery_main_dispatcher_integration_test.go` `BD71F41…` / 23693 B / 623 行、`main.go` `6D5373…` / 173317 B、`tasks.md` `F662C05…` / 756211 B / 2951 行仍无漂移；两层测试仍可在该 revision 通过。
3. 按 `task-execution-workflow.md` §10.3 仅把 S4.19 `[-]` → `[x]` 并写完成日期/关闭记录；不得改 S4.16/S4.17/S4.20/S8.2/S8.3 的 checkbox、日期、关闭记录或写集。
4. 关闭记录需保留：层 (a)/层 (b) 同修订证据、17 文件 drift=0、4 条 LOW 已披露、未验证边界不因关闭而扩张。
5. （建议）补一条 Plan B13H 指针，消除 `:1599` 的 S4.19 `[x]` 历史表述与当前状态的读者歧义。
6. 本结论为 AI 代理独立评审，不代替人的业务接收、部署/E2E 授权或 Git 写授权。

## 未验证边界（不得当作通过）

性能、真实部署/E2E、POSIX journal/fsync/flock/CAS、真实 keyring/HSM、`enter/install`/recovery secret candidate、approved backup inventory、production dispatcher 全路由、进程级唯一 runtime/Store/pool、S4.16 `TestRecoveryRuntimeProductionChain`（NOT RUN）、S4.20 `TestRecoveryProductionChainComplete`/五叶（本轮 NOT RUN）、`goimports`/`golangci-lint`（未跑）、人的业务接收与 Git 授权（未发生）。

## 证据

- 本目录 `report.md`、`commands.log`、`g2-05`～`g2-13` 原始日志/JSON。
- 关键哈希：dispatcher 测试 `BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A` / 23693 B；`main.go` `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B；`tasks.md` `F662C05814344289410CA7192E7B54F3D30F2207C4BA2AC4DED92E94E0808939` / 756211 B / 2951 行；Plan `0437577BF765E1D9DD21E8FFFDFE3124F5C9B27A421C0F2CEE09300F52FF459F` / 488854 B / 1615 行。