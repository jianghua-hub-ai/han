# ISSUE-identity-auth-0002 — 恢复驱动测试证据写死系统临时目录，重复运行累积 26 GB 挤占系统盘

**状态：已完成登记、定级建议与定位；2026-11-05 修复主因（ADR-0039），回归与独立评审未闭环、未关闭。** 本轮按用户 2026-11-02 指令「启动 Issue 流程开单并定位具体代码位置」执行，授权仅覆盖开单与只读定位，不含修复、清理、提交或验收关闭。

## 1. 登记（register）

| 字段 | 值 |
|---|---|
| 编号 | `ISSUE-identity-auth-0002` |
| 来源 | 用户报告「C 盘空间不足」；本会话排查中定位到本缺陷 |
| 首次发现 | 2026-11-02 |
| 关联任务 | `identity-auth-S4.14`（root 侧受控 RestoreDriver 与部署独占适配边界，2026-10-30 完成） |
| 关联制品 | `cmd/han-server/recovery_restore_test.go`（当前为 Git 未跟踪文件，`?? cmd/`，不在任何提交中） |
| 影响模块 | `identity-auth`（恢复驱动测试脚手架）；同类模式另波及 `article-management` 与 `test/integration`，见 §3.3 |
| 固定工作区 | `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 环境 | Windows 开发机；`%TEMP%` = `C:\Users\J00010385\AppData\Local\Temp`（同一路径的 8.3 短名为 `C:\Users\J00010~1\AppData\Local\Temp`） |
| 定位执行者 | 主会话（本 Host 未投影 Issue 专用 Role；入口表该行只列 Skill，且本轮为只读定位，故按 `add-issue-handling` 直接执行并现场留证） |

## 2. 定级（classify）

**P2（建议值，Lead 提出，尚未经人确认）。**

- 不是生产运行时缺陷：受影响代码只在测试二进制中编译执行，服务镜像不含该路径。
- 但可复现地耗尽开发机系统盘：实测单机残留 **3361 个目录 / 4730 个文件 / 26.01 GB**，排查开始时 C: 可用空间仅 **6.3 GB**；清理前 C: 已接近无法容纳构建与索引产物。
- 无自动上限：每轮 `TestRecoveryRestoreDriver` 固定新增 **55 个目录、约 0.4 GB**，`retain` 语义明确禁止清理，重复运行线性累积。
- 升级条件：若再次导致构建/测试失败，或系统盘可用空间再次跌破安全工作阈值，应升为 P1 并按 P1 节奏处置。
- 定级来源：Lead 依据本会话实测证据提出；用户尚未确认级别。

## 3. 定位（diagnose）

### 3.1 现象与实测证据

计数与复现命令（2026-11-02 实跑）：

```powershell
(Get-ChildItem $env:TEMP -Directory | Where-Object Name -like 'han-s414-retain-*').Count
go test ./cmd/han-server -run TestRecoveryRestoreDriver -count=1 -timeout 900s
```

| 观测 | 结果 |
|---|---|
| 搬移前残留 | 3361 个目录 / 4730 个文件 / 26.01 GB，全部位于 `%TEMP%\han-s414-retain-*` |
| 未设环境变量跑一轮 | `ok 8.615s`；`%TEMP%` 目录 57 → 112（**+55**），D 侧 +0 |
| 设 `HAN_S414_TEST_EVIDENCE_DIR=D:\han-test-evidence` 跑一轮 | `ok 7.395s`；D 侧 3416 → 3471（**+55**），`%TEMP%` +0 |
| 单目录体积 | 0.1–35 MB（大目录含被测测试二进制副本，约 17.5–35 MB） |
| 累积轮次反推 | 3361 ÷ 55 ≈ **61 轮**全量运行 |

处置现状（未删除，仅搬移）：2026-11-02 按用户授权将 3361 个目录整体移动到 `D:\han-test-evidence\`（`MOVED=3361 FAILED=0`，源侧 0 残留）；该目录现保留 3416 个目录 / 26.42 GB（另 55 个由同一测试的另一次运行产生）。**证据全部保留，未删除任何一份。**

### 3.2 直接根因（精确代码位置）

`cmd/han-server/recovery_restore_test.go:107-116`（修改后行号；修改前为 `:107-115`），函数 `s414Directory`：

```go
func s414Directory(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "han-s414-retain-")   // 修改前：:109
	if err != nil {
		t.Fatal("create retained fixture")             // 修改前：:111
	}
	t.Logf("retained_fixture=%s", dir)
	return dir
}
```

- `os.MkdirTemp` 的 `dir` 参数为空串时按 Go 语义回落到 `os.TempDir()`，Windows 上即 `%TEMP%`（系统盘 C:）。
- 该函数在同一个测试进程内被反复调用，仅两个静态调用点，但每轮 `TestRecoveryRestoreDriver` 实测产生 55 个目录：
  - `newS414Fixture`（`:170-172`，修改后行号）——每个 subtest 的 fixture 根目录；
  - `s414ConfigureLaunchProbe`（`:794-804`）——写入自身测试二进制副本 `owned-restore-test.exe`。
- 目录前缀 `han-s414-retain-` 与注释语义表明这是**刻意保留的证据目录**：`cmd/` 包内该测试族没有任何 `RemoveAll`/`os.Remove`/`t.TempDir` 清理分支（对比 `main_test.go`、`recovery_command_test.go` 等均用 `t.TempDir()` 自动回收），因此无上限、无过期。
- 触发放大：任何一次 `go test ./cmd/han-server -run TestRecoveryRestoreDriver`（含反复调试、评审复跑、多 Agent 并行验证）都会叠加 55 个新目录。

### 3.3 同类生成点（同一根因模式：证据根固定为系统临时目录）

| 位置 | 目录前缀 | 说明 |
|---|---|---|
| `cmd/han-server/recovery_restore_test.go:112` | `han-s414-retain-` | 本 Issue 主要贡献者，26.01 GB |
| `internal/article/deletion_journal_snapshot_test.go:321` | `han-article-anchor-fixture-` | `os.MkdirTemp("", ...)` |
| `internal/article/replay_publication_integration_test.go:226` | `han-b13d-a48-media-` | `os.MkdirTemp("", ...)` |
| `internal/article/replay_publication_integration_test.go:864` | `han-b13d-a48-legacy-` | `os.MkdirTemp("", ...)` |
| `internal/identity/recovery_installation_postgres_test.go:476` | `s413-approved-checkpoint-` | 显式 `os.Getenv("TEMP")`：非 Windows 返回空串后仍回落 `os.TempDir()`，语义不一致且掩盖意图 |
| `internal/identity/recovery_installation_postgres_test.go:662` | `s413-deletion-media-` | 同上 |
| `test/integration/crossmodule/canonical_user_delete_test.go:511` | `<schema>_` | 注释明示 `Intentionally retained; no t.TempDir/RemoveAll` |
| `test/integration/identity/auth_security_fixture_test.go:318` | `<schema>_` | 注释明示 `Retained; never t.TempDir/RemoveAll` |
| `test/integration/identity/auth_security_postgres_test.go:399` | `<schema>_probe_` | 注释明示 `Retained, including failures` |
| `test/integration/identity/orphan_disposition_test.go:913` | `han_s62_journal_` | 保留型 journal 目录 |
| `test/integration/recovery/article_replay_test.go:202` | `han-s44-retained-` | 保留型 fixture 目录 |

说明：这些点位的「保留」是各设计文档明确要求的证据语义，缺陷不在保留本身，而在**证据根不可配置、默认落在系统盘、且没有容量上限或清理入口**。

### 3.4 仓内已有可复用先例

| 变量 | 位置 | 用法 |
|---|---|---|
| `HAN_S105_TEST_EVIDENCE_DIR` | `test/integration/identity/password_reset_maintenance_test.go:45`、`:115` | `os.MkdirTemp(os.Getenv(env), "han-s105-cli-")` |
| `HAN_AUDIT_EVIDENCE_DIR` | `test/integration/article/audit_insert_rollback_test.go:145-146` | 证据目录根，未设置时显式拒绝 |

即「环境变量指定证据根」在本仓已有既定做法，`s414` 只是没有采用。

### 3.5 已排除（同日实测）

- **不是 Go 构建缓存**：`go clean -cache` 单独释放 2.99 GB，且位于 `%LOCALAPPDATA%\go-build`，与 `%TEMP%\han-s414-retain-*` 路径不同；26 GB 与构建缓存无关。
- **不是 WinGet portable 实体**（`Local\Microsoft\WinGet\Packages`，0.98 GB，含 uv / Poppler，已保留未动）。
- **不是分页文件/休眠文件/DISM 组件**：这些为同日独立处置项，与本 Issue 的证据目录不重叠。
- **不是其他进程的临时文件**：残留目录名统一为 `han-s414-retain-`，位于 `%TEMP%` 根层，创建时间与 `TestRecoveryRestoreDriver` 运行时间一一对应。

### 3.6 未知与未验证

- 其他开发机/CI 上的残留规模未盘点，本 Issue 只以本机实测为证。
- 3361 个目录是否全部来自 `TestRecoveryRestoreDriver`（按 55/轮推断为 61 轮）未逐目录核对 run id；存在其他 subtest 组合时轮次数会不同。
- 未发现 Go 之外的载体（脚本/CI 配置）写入该目录；未做全仓与 CI 配置穷举。

## 4. 当前状态（定位完成，不等同修复或关闭）

工作树中存在一份**未提交的缓解改动**，为本会话按用户指令「将 harness 输出根指到 `D:\han-test-evidence`」完成，不属本 Issue 的修复授权，也不能作为关闭依据：

- 新增常量 `s414EvidenceDirEnv = "HAN_S414_TEST_EVIDENCE_DIR"`（`:27`）。
- `s414Directory` 改为 `root := os.Getenv(s414EvidenceDirEnv)` + `os.MkdirTemp(root, "han-s414-retain-")`，失败信息含 root 与环境变量名（`:111-115`）。
- 实测：未设变量 → 仍落 `%TEMP%`（+55，默认行为不变）；设变量 → 落 `D:\han-test-evidence`（+55，`%TEMP%` +0）。
- 该改动仅覆盖 `s414` 一处，且**未提交**；`git status` 仍为 298 行脏工作树，`cmd/` 整体未跟踪。

仍未解决的部分：默认值依旧指向系统盘；§3.3 表中除 `s414Directory` 外的 10 个同类生成点未覆盖；无保留上限、无到期策略、无经确认的清理入口。

## 5. 后续候选（未授权，仅登记）

| 候选 | 说明 | 备注 |
|---|---|---|
| 统一证据根变量 | 引入单一 `HAN_TEST_EVIDENCE_DIR`，各测试的既有专用变量（`HAN_S105_TEST_EVIDENCE_DIR`、`HAN_AUDIT_EVIDENCE_DIR`、`HAN_S414_TEST_EVIDENCE_DIR`）作为覆盖回落 | 需先确认命名与优先序，属跨测试契约变更 |
| 默认值保持 `os.TempDir()` | 不在代码中硬编码盘符（`no-hardcode-policy`），改由运行环境或脚本显式设置证据根 | 与当前缓解一致 |
| 保留策略 | 按 run/日期分层 + 明确的容量上限与清理清单；任何删除仍需用户按清单确认 | 与「证据保留」语义冲突，须设计取舍 |
| 修正 `os.Getenv("TEMP")` | `internal/identity/recovery_installation_postgres_test.go:476/662` 改用统一变量 | 跨平台语义一致性 |

以上均未实施；按 ADD 规则，永久规则类改动须先走 `add-project-experience` 形成候选并取得采用决定。

## 6. 证据引用

- 残留规模与搬移结果：本会话 `MOVED=3361 FAILED=0`，目标侧 3361 目录 / 4730 文件 / 26.01 GB；`D:\han-test-evidence` 现存 3416 目录 / 26.42 GB。
- 增量复现：见 §3.1 表格，两轮真实执行（`ok 8.615s` / `ok 7.395s`）与前后计数。
- 代码位置核对：`Select-String -Path cmd\han-server\recovery_restore_test.go -Pattern 's414Directory|MkdirTemp'`，行号与 §3.2 一致。
- 同类点位核对：全仓 `*.go`（排除 `.spec` 快照）检索 `MkdirTemp(|os.Getenv("TEMP")|EVIDENCE_DIR`，结果与 §3.3、§3.4 一一对应。

## 7. 2026-11-05 复核与部分修复

**复核（只读实测）**：`%TEMP%` 现残留 1,367 个 `han-s414-retain-*` 目录，合计 **10.45 GB**；另 5,889 个空 `han-article-anchor-fixture-*`、62 个 `han-b13d-a48-media-*`、54 个 `han-b13d-a48-legacy-*`、6 个 `s413-approved-checkpoint-*`、1 个 `s413-deletion-media-*`；C: 可用 38.39 GB。§3.1 的 26 GB 已被人工搬移，但清理策略仍未建立，泄漏按原速率继续。

**部分修复（3 文件 4 处）**：§3.3 表中未被设计标注为「刻意保留」的生成点改为 `t.TempDir()`，随测试自动回收：

- `internal/article/deletion_journal_snapshot_test.go:321`（`han-article-anchor-fixture-`）
- `internal/article/replay_publication_integration_test.go:226`（`han-b13d-a48-media-`）
- `internal/article/replay_publication_integration_test.go:864`（`han-b13d-a48-legacy-`）
- `internal/identity/recovery_installation_postgres_test.go:476`（`s413-approved-checkpoint-`）

**当时未改**（同日稍后按 ADR-0039 修复，见本节末）：`cmd/han-server/recovery_restore_test.go` 的 `s414Directory`（主因，10.45 GB），以及 §3.3 中 4 处注释明示 `Intentionally retained` 的点位。理由：保留是设计要求，问题在证据根/上限/清理入口，属策略决策，不作局部修改。

**验证**：`go test ./internal/article -count=1` → `ok`；执行前后 `han-article-anchor-fixture-*` 计数均为 5,889（不再新增）；`gofmt -l` 三文件无输出；`go vet -tags=integration ./internal/article ./internal/identity` exit 0。**未验证**：受授权 DSN 守卫的集成测试在本机不可运行（`internal/article` 要求 `han_b13c_article`/`han_b13c`，本机 DSN 为 `postgres`），replay 与 s413 两处的真实集成回归仍待按授权 fixture 执行；`s413WriteSyntheticTombstone`（`:662`，无 `*testing.T` 可用）未改。

**当时结论**：仍未关闭；下一步须先决定 s414 与其余保留型点位的证据根、保留上限与清理入口（§5 候选），再实施。

### 2026-11-05 续：按 ADR-0039 修复主因

项目负责人回复「继续」，采 Lead 建议第二项，落 **ADR-0039**（`decisions/framework.md`）：测试临时/证据目录默认用 `t.TempDir()` 回收，需保留演练证据时由环境变量显式指定证据根。

据此修改 `cmd/han-server/recovery_restore_test.go` 的 `s414Directory`：`HAN_S414_TEST_EVIDENCE_DIR` 为空 → `t.TempDir()`；非空 → 仍在指定根保留并 `t.Logf`。4 处设计标注 `Intentionally retained` 的点位按 ADR-0039 保持现状。

**实测**：

- 未设变量：`go test ./cmd/han-server -run TestRecoveryRestoreDriver -count=1` → `ok 7.147s`；`%TEMP%` 中 `han-s414-retain-*` 计数 1,429 → 1,429（**新增 0**）。
- 设 `HAN_S414_TEST_EVIDENCE_DIR`：同命令 → `ok 6.837s`；55 个目录落在指定根，`%TEMP%` 计数不变；验证后已删除该临时探针目录。
- `gofmt -l cmd/han-server/recovery_restore_test.go` 无输出。

**现状**：`%TEMP%` 仍残留 1,429 个目录 / 10.88 GB，全部是修复前产物；清理清单 `.spec/wip/cleanup/temp-residue-20261105.md`（逐项列出），**未执行任何删除**，待用户按清单确认。

**仍未关闭**：独立评审与两方验收未完成；历史残留的清理未完成。
