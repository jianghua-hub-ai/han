# identity-auth-S4.21 实施证据（r01）

- 角色：Backend-Dev（add-backend-dev），正式实施
- 日期：2026-11-01
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 授权范围：仅 3 个代码文件（`recovery_runtime.go` 修改、`main.go` 仅必要修改、新增 integration 测试）；未做 Git 写、未清理、未联网、未起 PG/HTTP。

## 1. 写前后基线核对

| 文件 | 写前 SHA256 | 写后 SHA256 | 结论 |
|---|---|---|---|
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B / 2026-10-31 19:26:09 | 同左，字节与 mtime 完全一致 | **零改动** |
| `cmd/han-server/recovery_runtime.go` | `900F31F5BCBB7A6403025E583E72B49E3BA491B2A4AEE8A8BAC09EFD070F2BD7` / 35832 B / 2026-10-31 19:25:50 | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` / 37431 B / 2026-11-01 20:39:42 | 预期修改 |
| `cmd/han-server/recovery_enter_install_integration_test.go` | 不存在 | `6BFA7D761432C960740F4E1FE91566317B5707B8481CE0D9FF1CF46DD4106C3A` / 20564 B | 新增 |

并发写者 `xDesk.exe` 六个 PID（22928/25764/29960/35384/39592/49420）写前后均在。

## 2. 改动点（recovery_runtime.go）

1. `:10` 新增 `strings` 导入。
2. `:449` `recoveryRootInputs` 新增未导出 seam 字段 `secretCandidate func(context.Context, recovery.RunBinding) ([]byte, error)`。
3. `:456-497` 新增 `recoveryEnterInstallMissingPorts` 与 `recoveryEnterInstallClosedError`：固定顺序 `journal, articleSnapshots, restore, consumer, installation, articleAccount, commitments, sources, anchor, exclusive, secretCandidate`，只做 `== nil` 判断（接口与 func 均直接比较，未用 reflect）。
4. `:527-534` 原 `:482-483` 的无条件 closed 替换为 enter/install 显式 precheck，位置仍在 `runs`/`closures`/anchor registry 等任何资源构造之前；`normal`/`verify`/`resume` 直接放行，未知 action 仍 `ErrAdmissionBinding`。
5. `:631-637` secret candidate seam：`inputs.secretCandidate != nil` 时使用注入函数，否则保留原 closed 闭包语义；`newRecoverySecretSource(...)` 始终被调用，签名未变。

错误形态（production 缺端口）：

```text
recovery admission closed: missing recovery enter/install ports: journal, articleSnapshots, restore, consumer, installation, articleAccount, commitments, sources, anchor, exclusive, secretCandidate
```

`errors.Is(err, recovery.ErrAdmissionClosed)` 成立，错误文本列出全部缺失端口。

`main.go` 未注入 `secretCandidate` 或其余 10 个端口，production `enter`/`install` 仍 closed。

## 3. 新测试文件

`cmd/han-server/recovery_enter_install_integration_test.go`（`//go:build integration`，`package main`）仅两个顶层测试：

- `TestRecoveryEnterInstallAssemblyWiring`
  - production 默认 inputs（pool+prepared+enter/install plan）→ `ErrAdmissionClosed`，文本逐项含 11 端口；
  - 11 项全注入 → `Install` 到达 `s421Sources` sentinel，`Enter` 到达 `s421Exclusive` sentinel；只证明装配路径可达；
  - 同一 root 仅一个 runtime/Store：`composition.runtime()`、`ports.Runtime`、`bridge.boundRuntime()` 与 `assembly.runtime` 同一值，运行时依赖指针等于注入的 `runs`/`closures`；
  - 未注入 `secretCandidate` 的 verify 形态仍返回 `ErrAdmissionClosed`，证明默认 closed 未被放宽。
- `TestRecoveryEnterInstallConfigCommandMatrix`
  - 配置正控：normal、install、enter、resume、verify；
  - 配置负控 9 例：非法 mode、大写 digest、journal file mode 0644/0000、相对 journal 路径、control 端口冲突、相对 restore executable、非法 restore sha、install 在 normal mode；每例断言 error 且 `t.TempDir()` 前后均为空；
  - 命令正控：`selectRecoveryCommand` 与 `parseRecoveryCommand` 覆盖上述形态；
  - 命令负控 11 例：缺 flag、重复 flag、未知 flag、相对 manifest、多余位置参数、前导空格、`--secret` 等；
  - 文件注释明确“解析通过不等于 Enter/Install 执行可达”。

## 4. Red → Green

- Red 2a（`red.log`）：测试先写、seam 字段未建 → `unknown field secretCandidate`，build failed，exit 1。
- Red 2b（`red.log`）：仅加字段、未改 precheck/seam → 行为失败：
  - production enter/install 旧文本 `the restore and installation source loaders are not delivered` 与期望不符；
  - 全注入 install/enter 仍被旧无条件 closed 拒绝；
  - 仅 `default secret source stays closed` 通过；生产 shape 的 enter/install、两个注入 seam、one runtime、以及父测试共 6 个子用例 + 父测试失败。
- Green：实现 precheck 与 seam 后两个测试全绿。

## 5. 验收命令真实结果（原始输出见 `commands.log`）

| # | 命令 | Exit | pass | fail | skip | DATA RACE |
|---|---|---|---|---|---|---|
| 1 | `gofmt -l cmd/han-server/main.go cmd/han-server/recovery_runtime.go cmd/han-server/recovery_enter_install_integration_test.go` | 0 | - | - | - | - |
| 2 | `go build ./...` | 0 | - | - | - | - |
| 3 | `go vet ./cmd/han-server` | 0 | - | - | - | - |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | - | - | - | - |
| 5 | `go test ./cmd/han-server -run '^TestRecoveryRuntimeComposition$' -count=1 -json` | 0 | 9 | 0 | 0 | no |
| 6 | `go test -race ./cmd/han-server -run '^TestRecoveryRuntimeComposition$' -count=1 -json` | 0 | 9 | 0 | 0 | no |
| 7 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | 11 | 0 | 0 | no |
| 8 | `go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryMainBusinessDispatcher$' -count=1 -json` | 0 | 11 | 0 | 0 | no |
| 9 | `go test -tags=integration ./cmd/han-server -run '^TestRecoveryEnterInstallAssemblyWiring$' -count=1 -json` | 0 | 8 | 0 | 0 | no |
| 10 | `go test -race -tags=integration ./cmd/han-server -run '^TestRecoveryEnterInstallAssemblyWiring$' -count=1 -json` | 0 | 8 | 0 | 0 | no |
| 11 | `go test -tags=integration ./cmd/han-server -run '^(TestRecoveryConfig|TestRecoveryCommand|TestRecoveryEnterInstallConfigCommandMatrix)$' -count=1 -json` | 0 | 26 | 0 | 0 | no |
| 12 | `go test -race -tags=integration ./cmd/han-server -run '^(TestRecoveryConfig|TestRecoveryCommand|TestRecoveryEnterInstallConfigCommandMatrix)$' -count=1 -json` | 0 | 26 | 0 | 0 | no |

Git 只读核对（原始输出见 `commands.log`）：

- `git status --short -- cmd/han-server ...` → `?? cmd/han-server/`、`?? .../s421-implementation-r01/`：本仓库 `cmd/` 整体未跟踪。
- `git diff --stat` / `git diff --check` / `git diff -- cmd/han-server/main.go cmd/han-server/recovery_runtime.go` 均为 exit 0；因 `cmd/` 未跟踪，diff 为空，**不能据此声称改动已验证**。改动证据以写前后 SHA256、字节数、行号与本报告为准。

## 6. 未验证边界

- production `Enter`/`Install` 仍不可达，仍返回 `ErrAdmissionClosed`；本卡不声称 Case18/S8.2 PASS 或解除 blocked-upstream。
- 未运行需要真实 PostgreSQL 的 S4.20/S4.16 链路测试；仅以 `go vet -tags=integration` 与 compile 证明既有测试文件可编译。`recovery_production_chain_integration_test.go:2991` 仍使用 resume plan，不受 enter/install precheck 影响（静态核对）。
- 未做真实部署、POSIX/ACL、真实 provider/备份、E2E；未证明 secret 轮换/复用行为。
- 未做 Git 提交，交由 Lead 现场复核。

