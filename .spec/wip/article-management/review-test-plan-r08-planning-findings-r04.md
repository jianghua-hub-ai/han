# Article Management test-plan r08 planning findings 定点复审 r04

- **日期**：2026-10-23
- **Reviewer**：Rawls，原 Test-plan 非作者独立 Reviewer；本轮未委派
- **objectType**：design（测试规划/验收对象）
- **分支 / HEAD**：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **被审文件**：`.add/project/knowledge/design-workspace/article-management/test-plan.md`
- **revision**：`article-management-test-plan-ir01-r08`
- **SHA256**：`4608502D4B4E9078509651A52589245D6726634053FF8A5E787569C55776EA7E`
- **git blob**：`9079f7c33d2b56c8439f161a11be6425a5bf1687`
- **状态**：`planned-not-executed`
- **Test-plan Gate**：**PASS**
- **可交 Task-Planner**：**是；仅授权同步既有唯一 S4.6/B6A-W，不能外推为 planning finding 已关闭、任务已启动或实现/验证完成**

## 1. 结论摘要

| 对象 | 定点状态 | 结论 |
|---|---|---|
| `ARTICLE-S4.6-PLAN-R01-F001` | **CLOSED（test-plan 侧）** | r08 冻结 linux/amd64 与 windows/amd64 两个 `go test -c` 目标，分别覆盖 flock 与 unsupported 互斥 build-tag 分支；失败传播、子进程环境隔离、唯一临时目录和 `finally` 清理合同充分。 |
| `ARTICLE-S4.6-PLAN-R01-F002` | **CLOSED（test-plan 侧，保持）** | integration-tag compile-only Gate 未回归，仍明确不代表真实 PostgreSQL 执行或 S4.3 关闭。 |
| `AM-TP-R07-R03-F001` | **CLOSED** | §6 五条 A-WIRE 映射均已统一为唯一 S4.6/B6A-W 已登记/已安排、todo、planning findings OPEN、未认领/未启动。 |
| `AM-TP-R07-R03-F002` | **CLOSED** | metadata、来源表、§1.2、§11～§12 已形成 r05 FAIL → r06 PASS（仅绑定 r06）→ r07 FAIL → r08 待本次复审的完整追溯链。 |

未发现修订引入的 CRITICAL、HIGH、MEDIUM 或 LOW 新 finding。r08 足以作为单义输入交 Task-Planner，将**既有且唯一**的 `article-management-S4.6` / `B6A-W` 从旧 11 文件合同同步为 13 文件合同；本报告不关闭 Tasks/Plan 中的 planning finding、`B6A-G4-R01-F001` 或 Planning Review Gate。

## 2. 固定依据

| 对象 | 固定值 | 用途 |
|---|---|---|
| test-plan r08 | SHA256 `4608502D4B4E9078509651A52589245D6726634053FF8A5E787569C55776EA7E`；blob `9079f7c33d2b56c8439f161a11be6425a5bf1687` | 被审修订 |
| design r10 | SHA256 `6785977683CF0F59E277667425F7FEE9714EBBD436FBDFE046E607F2E77CA67E` | writer lock、lifecycle 与 S4.6/S4.3 边界 |
| stories r07 | SHA256 `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05` | 验收边界 |
| production wiring design review r01 | SHA256 `1EB2D78179D40747904CCC8F5E975FAB4A5CD4205C754567F6B5C589BAFC001E` | 设计侧 PASS 与未验证边界 |
| Tasks/Plan planning review r01 | SHA256 `0AF25106B09D3D5E2FB42E6BA04BDB85C7A92F1E7CF7AB8A3CF342CE9350F65B` | 原 `ARTICLE-S4.6-PLAN-R01-F001/F002` |
| test-plan r06 review r02 | SHA256 `B26C9E88932925CEC7229DCAFD971E2A47514184967279488FF37CB7F0F391F2` | r06 历史 PASS |
| test-plan r07 review r03 | SHA256 `4F35F5110E965443FA498D38F71498D39C91B98BD7DACA9F06926DE1BFA4BE2D` | 本轮复审的剩余与新增 findings |

开始评审及落报告前均核 branch、HEAD、revision、status、日期、SHA256、blob、关键依据哈希和报告路径；未发现对象漂移或路径占用。

## 3. 两轮定点范围

### Round 1：r08 直接整改

检查：

- revision、`planned-not-executed`、2026-10-23 日期及 source/review 追溯；
- §1.3、§1.4 对四项 finding 的状态约束；
- §4.10.2、§9.1 的 Article 私有锁接口、精确 13 文件、第 14 文件停止条件；
- PowerShell 双 GOOS compile-only 命令的目标选择、错误传播、输出不执行、环境隔离与清理；
- integration-tag compile-only Gate 是否保持原边界；
- §6 五条 A-WIRE 登记状态；
- §10～§12 是否仍声明实现、测试和真实环境未完成。

### Round 2：消费者、Tasks/Plan 与后继边界

有界核对：

- `internal/recovery/filelock_flock.go` 与 `filelock_unsupported.go` 的现役互斥 GOOS build tags、Unix `flock` 和 unsupported fail-closed 模式；
- `internal/article/deletion_journal.go` 的当前接入位置；
- `internal/article/repository_integration_test.go` 的 `//go:build integration` 与 PostgreSQL 调用路径；
- 当前 Tasks/Plan 中恰有一个 todo `article-management-S4.6` 和一个专属 `B6A-W`，仍保留旧 11 文件合同且等待 Task-Planner 同步；
- 直接 requires 仍仅为 `identity-auth-S5.2`、`article-management-S4.1`，后继仍为 `article-management-S4.3`、`article-management-S4.5`、`article-management-S1.4`，未形成循环或第二 wiring task；
- S4.6 的组件/fake/编译责任与后续真实 PostgreSQL、Unix 持久卷、部署及 E2E 责任仍分离。

## 4. 原 findings 逐项复审

### `ARTICLE-S4.6-PLAN-R01-F001` — CLOSED（test-plan 侧）

- **位置**：`test-plan.md:278-290, 458-528`
- **修复证据**
  - 原 11 文件只新增 `internal/article/deletion_filelock_flock.go` 与 `deletion_filelock_unsupported.go`，形成精确 13 文件写集；
  - 两文件限定为 Article 私有接口，不修改、导出或复制 `internal/recovery` 锁原语，不新增依赖；
  - linux/amd64 子目标设置 `CGO_ENABLED=0`、`GOOS=linux`、`GOARCH=amd64` 后执行 `go test -c`，覆盖受支持 Unix flock 分支；
  - windows/amd64 子目标在同一子进程中切换 `GOOS=windows` 并保留 `GOARCH=amd64`，覆盖 unsupported fail-closed 分支；
  - 两次原生命令后均读取 `$LASTEXITCODE`，非零即 `throw`；未捕获的终止错误使 PowerShell 子进程非零；
  - 生成物只作为 `go test -c -o` 输出，脚本没有调用两个输出路径；
  - 唯一 GUID 临时目录包在 `try/finally` 中，成功或失败都以 `Remove-Item -LiteralPath $tmp -Recurse -Force` 清理；
  - GOOS/GOARCH/CGO 环境只在显式启动的 `powershell -NoProfile` 子进程内修改，不污染父进程；
  - 命令文本不引用 DSN，且文档明确禁止执行生成物。
- **证明边界**
  - 该 Gate 只证明目标 GOOS 下的文件选择、类型/API 和依赖闭合；
  - 不证明真实 Unix `flock`、owner-only、file/directory fsync、崩溃、跨进程竞争或真实 PostgreSQL；
  - 这些真实边界继续由 S4.3 等既有专项关闭。
- **验证结果**：PowerShell AST 静态解析 0 错误；确认两个 `go test -c`、两个 GOOS 目标、amd64、`try/finally`、单一创建/清理命令及无生成物调用。依约未实际跨 GOOS 编译。

### `ARTICLE-S4.6-PLAN-R01-F002` — CLOSED（test-plan 侧，保持）

- **位置**：`test-plan.md:510-520, 528`
- `go test -tags=integration ./internal/article -run '^$' -count=1` 保持不变；
- `repository_integration_test.go` 仍由 `//go:build integration` 选择；
- 文档继续明确该命令只提供 integration-tag owner 文件编译证据，不代表真实 PostgreSQL 用例执行、S4.3 关闭或 production readiness；
- r08 未把 DSN、真实数据库或真实环境加入 S4.6 局部关闭前置。

### `AM-TP-R07-R03-F001` — CLOSED

- **位置**：`test-plan.md:374-391`
- A-WIRE-CFG/JRN/RTR/WKR/LIFE 五行均写明：
  - 唯一 S4.6 与专属 B6A-W 已登记/已安排；
  - 当前为 todo；
  - planning findings OPEN；
  - 未认领、未启动；
  - S4.3/S4.5/S1.4 或真实专项仍独立且未执行。
- 全文只把“尚未登记”用于历史 design review 当时状态，没有当前状态冲突，也没有建议创建第二个 production wiring task。

### `AM-TP-R07-R03-F002` — CLOSED

- **位置**：`test-plan.md:16-23, 46-50, 66-92, 542-575`
- r06、r07 review 路径和固定 SHA256 已进入 metadata 与来源表；
- §1.2 明确 r06 PASS 只绑定固定 r06；
- §11、§12 保持 r05 FAIL → r06 PASS（仅 r06）→ r07 FAIL → r08 待 Rawls 复审的顺序；
- r08 未用 r06 历史 PASS 自判当前 Gate，也未改写 r07 FAIL。

## 5. 新 findings

无。

未发现第二 wiring task、13 文件之外的条件性扩权、requires/DAG 循环、真实 PostgreSQL/Unix/部署/E2E 并入局部关闭 Gate，或实现、测试、真实环境和 production readiness 完成冒称。

## 6. 静态验证

| 命令 / 检查 | 结果 |
|---|---|
| `git branch --show-current`、`git rev-parse HEAD` | PASS；固定 branch/HEAD |
| `Get-FileHash -Algorithm SHA256` 核 r08、design、stories、design review、planning review、r06/r07 review | PASS；均匹配固定值 |
| `git hash-object -- <test-plan>` | PASS；匹配固定 blob |
| `Select-String` 核 revision/status/date/source、§6 五行、13 文件/第 14 文件、requires/B6A-W/DAG、未验证声明 | PASS |
| PowerShell `Parser.ParseInput` 静态解析 §9.1 双 GOOS 子进程脚本 | PASS；0 parse error |
| 有界读取四个允许代码文件 | 完成；仅核 build-tag/compile-only 可实施性，未扩成代码评审 |
| 当前 Tasks/Plan 搜索 S4.6、B6A-W 与 planning finding | 确认唯一 todo task/唯一批次、旧 11 文件待同步、finding OPEN、未认领/未启动 |
| Go、数据库、跨 GOOS 编译、Unix、部署、E2E | **NOT RUN** |

## 7. 未验证范围与限制

- 未运行 Go、PostgreSQL、跨 GOOS 编译、Unix 持久卷、部署、Host 或 E2E。
- 未读取或输出 `HAN_TEST_POSTGRES_DSN`。
- 两个 Article file-lock 文件和 worker 文件尚未实现；本报告不评价未来实现质量。
- 静态命令审查不能替代实施时保留两个 compile-only 目标的真实成功输出、生成物清理证据和代码 Review。
- 未重开 r07 已通过且未变的测试规划范围。
- 未修改或批准 Tasks/Plan；其 11 文件合同、planning finding 和 Planning Review Gate 仍须由后续同步与原复审链处理。
- 未验证真实 `flock`、file/directory fsync、崩溃、磁盘故障、跨进程竞争、真实 PostgreSQL、容量或 production readiness。

## 8. Gate 与 Task-Planner 交回条件

### Test-plan Gate：PASS

r08 可以交 Task-Planner，但授权范围仅为：

1. 更新**既有且唯一**的 `article-management-S4.6` 与专属 `B6A-W`，不得新建或拆分第二个 wiring task；
2. 把旧 11 文件写集单义同步为 r08 §9.1 的精确 13 文件，并保留“需要第 14 个文件即停止”；
3. 同步 linux/amd64、windows/amd64 双 GOOS compile-only Gate与 integration-tag compile-only Gate；
4. 保持唯一 owner、两个直接 requires、B6A-W 位置及三个后继方向不变；
5. 保持 S4.6 为 todo、未认领、未启动，不登记实现者或冒称任何命令已执行；
6. 不把真实 PostgreSQL、Unix 持久卷、部署、Host、E2E、容量或 production readiness 并入 S4.6 局部关闭 Gate；
7. Tasks/Plan 同步后交回原 planning Reviewer/Lead 复审；由该链决定 `ARTICLE-S4.6-PLAN-R01-F001/F002`、`B6A-G4-R01-F001` 和 Planning Review Gate，Rawls 本报告不代为关闭。

本 PASS 只证明固定 r08 的测试规划整改充分，可进入 Tasks/Plan 同步；不证明 task 已同步、已获开发授权、已实现、已测试或具备 production readiness。

## 9. 实际改动

- 仅新增本报告：`.spec/wip/article-management/review-test-plan-r08-planning-findings-r04.md`
- 未修改 test-plan、design、stories、Tasks、Plan、代码或既有报告。
- 未 stage、commit、push、切分支、clean、reset、checkout 或清理任何他人改动。
