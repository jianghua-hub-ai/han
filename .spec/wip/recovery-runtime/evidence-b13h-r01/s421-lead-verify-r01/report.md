# S4.21 Lead 现场复核报告 (s421-lead-verify-r01)

- 工作区: `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 执行人: Lead（主会话）  日期: 2026-11-01
- 被复核产出: Backend-Dev / Chandrasekhar (`01a19707-9459-7f51-9a8c-bf28e7c75169`)

## 1. 代码哈希与范围

| 文件 | SHA256 | 字节 | 今日改动 |
|---|---|---|---|
| `cmd/han-server/recovery_runtime.go` | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` | 37431 | 是 (20:39:42) |
| `cmd/han-server/recovery_enter_install_integration_test.go` | `6BFA7D761432C960740F4E1FE91566317B5707B8481CE0D9FF1CF46DD4106C3A` | 20564 | 是 (20:39:43) |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 | 否 |

扫描 `cmd/`、`internal/` 下 2026-11-01 之后修改的文件：仅上述 2 个代码文件（测试文件属实施范围内的新增测试）。无第 4 个代码文件，`internal/**` 零改动。`main.go` 保持基线哈希，production 未注入任何 enter/install 端口。

`git status --short`：`cmd/`、`internal/` 整体仍为未跟踪（`??`），本仓库无法用 `git diff` 作为改动证据，故以 SHA256 + 字节 + 行号 + 证据文件为准。

## 2. 源码逐点复核（Lead 直读原文）

- `:449` 新增 `secretCandidate func(context.Context, recovery.RunBinding) ([]byte, error)` 字段。
- `:456-492` `recoveryEnterInstallMissingPorts`：固定顺序 11 项，逐项 `== nil`（journal / articleSnapshots / restore / consumer / installation / articleAccount / commitments / sources / anchor / exclusive / secretCandidate）。
- `:494-496` 错误拼接 `%w: missing recovery enter/install ports: %s`，外层为 `recovery.ErrAdmissionClosed`。
- `:522-535` `newRecoveryRootAssembly`：`normal/verify/resume` 直接放行；`enter/install` 在**任何资源构造之前**（首个构造从 `:537` 起）执行 precheck 并返回缺失清单；其余 action 返回 `ErrAdmissionBinding`。
- `:631-640` secret seam：`inputs.secretCandidate` 非 nil 时替换默认闭包，nil 时保留返回 `ErrAdmissionClosed` 的默认实现，且始终经 `newRecoverySecretSource` 包装。

## 3. 命令复跑

见 `commands.log`：12/12 条 EXIT=0，test2json 合计 pass=116 / fail=0 / skip=0，无 `DATA RACE`。
其中第 10、12 条最初返回后台会话标记，本次已核对对应 JSON（`c10-wiring-race.json` 15916 B、`c12-config-command-race.json` 57608 B）完整且含 pass 结论，无需重跑。

## 4. 本次证据不能外推的部分

- production 的 `Enter` / `Install` 仍为 `ErrAdmissionClosed`；本次只证明“端口缺失清单 + 装配 seam + secret seam”成立，不构成 production 端口交付。
- 未运行真实 PostgreSQL 下的 S4.20 链路，未做 E2E、真实备份、部署。
- `blocked-upstream` 未解除；不得据此宣称 Case18 或 S8.2 通过。
- 独立评审尚未开始，任务不得关闭。

## 5. 证据清单

- `commands.log`（本目录）
- `c05`…`c12` 共 8 份 test2json 输出（本目录）
- 实施侧证据：`../s421-implementation-r01/{commands.log,red.log,report.md}`、`../s421-handoff.md`