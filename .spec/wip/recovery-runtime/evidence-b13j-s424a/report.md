- `.spec/wip/recovery-runtime/evidence-b13j-d08/review-arch-r01/report.md` / 9964 B / `EC9361004EF31091DC51F2B3F272264C8DC98544D8C42B2F482D18E5BE656CCB` / PASS WITH CONDITIONS
- `.spec/wip/recovery-runtime/evidence-b13j-d08/review-sec-r01/report.md` / 18884 B / `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC` / PASS WITH CONDITIONS

# B13J-S4.24a 实施阻断报告（BLOCKED：未交付、未关闭、未改代码）

> 角色：`add-backend-dev`（Backend-Dev）。工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（只读核对，未变）。
> 结论：**BLOCKED**。本报告不是关闭证据，不替代独立评审、人的生产批准或 Lead 复核。

## 0. 一句话结论

本卡要求把 restore 的「生产 source/target/tool 与批准主体」接入唯一 production root，并要求评估 ADR-0031 不变量 1–6 中的批准记录三方相等、角色分离、有效期与撤销、`st_dev`/`st_ino`、owner/ACL、受控只读挂载，以及经 `recovery.InstallationSourceVerifier` 的备份来源批准核验。这些检查的数据载体与接口在现行 design/ADR/配置中不存在，且在「只改 3 个既有文件 + 1 个新测试文件」的写集内无法补齐。按卡面 §7 停止条件停止：未改代码、未创建测试、未放宽断言、未虚构任何批准记录/批准角色/工具。

## 1. 基线与已执行命令（真实结果）

| 命令 | 退出码 | 输出 |
|---|---|---|
| `git rev-parse HEAD` / `git branch --show-current` | 0 | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` / `main` |
| `gofmt -l cmd/han-server/recovery_restore.go cmd/han-server/recovery_runtime.go cmd/han-server/main.go` | 0 | 空 |
| `go build ./...` | 0 | 空 |
| `go vet ./cmd/han-server` | 0 | 空 |
| `go vet -tags=integration ./cmd/han-server` | 0 | 空 |
| `go test ./cmd/han-server -run '^(TestRecoveryRestoreDriver|TestRecoveryRuntimeComposition)$' -count=1` | 0 | `ok  han/cmd/han-server  7.586s` |
| `go test -race ./cmd/han-server -run '^(TestRecoveryRestoreDriver|TestRecoveryRuntimeComposition)$' -count=1` | 0 | `ok  han/cmd/han-server  45.169s` |
| `go test -race ./cmd/han-server -run '^(TestRecoveryRestoreDriver|TestRecoveryRuntimeComposition)$' -count=1 -json` | 0 | `TestRecoveryRestoreDriver PASS 8.37s` / `TestRecoveryRuntimeComposition PASS 0.00s` / `ok  han/cmd/han-server  45.289s` |
| `go test -tags=integration ./cmd/han-server -run '^TestRecoveryConfig$' -count=1 -json` | 0 | `TestRecoveryConfig PASS 0.01s` / `ok  han/cmd/han-server  0.098s` |

未执行卡面第 7/8 条（`-tags=integration` 的 `TestRecoveryRestoreProductionWiring`）：新测试文件未创建，此时执行只会得到「无匹配测试」的空结果，属空真，不作为证据。第 9 条 `TestRecoveryConfig` 已执行并 exit 0。基线在 HEAD 上为绿，故下述阻断与既有失败无关。

## 2. 阻断项（逐项带锚点）

**B1 批准记录没有载体，也没有 wire 或配置键。**
- ADR-0031 要求批准记录为「备份来源与可执行文件两条独立记录」，最小字段含 `st_dev`/`st_ino`、owner/ACL、受控只读挂载来源、复核人、有效期与撤销状态（`.add/project/knowledge/decisions/framework.md:1115`），并明确「本 ADR 不选具体实现」「本项目当前未任命该角色，也不存在任何批准记录」（同处，另见 `framework.md:1094`）。
- design 只把 `HAN_RECOVERY_RESTORE_MANIFEST_PATH` 定义为 restore manifest（计划输入），并要求其批准摘要「由既有 CAS 信任边界的受控安装/备份来源记录核验，不能仅信文件内自带 hash」（`design-workspace/identity-auth/design.md:3878`）；未给出批准记录的文件格式、字段 wire 或读取路径。
- 配置只读三个 restore 键，无审批记录路径（`cmd/han-server/recovery_config.go:78`–`:80`、`:227`）。
- 因此「三方相等（批准记录 / ENV / 实际文件）」「角色不得自批」「有效期与撤销状态」缺少可判定输入。补齐必须新增配置键或新增 on-disk 契约：前者被卡面 §3 明令禁止，后者属未授权的公共契约变化。

**B2 既有 CAS 信任边界无法表达 restore 批准。**
- 卡面要求备份来源批准 ref/digest 经 `recovery.InstallationSourceVerifier` 核验。该接口入参是 `recovery.InstallationManifest`（`internal/recovery/runtime.go:190`–`:191`），该结构只有 `Version/Kind/DeploymentID/InstallationID/TargetDigest/SourceApprovalRef/SourceApprovalDigest/Recovery/Article/IssuedAt`（`internal/recovery/runtime.go:164`–`:171`），不含 `BackupRef`、`ArtifactDigest`、`ArtifactBytes`、`SourceSchemaDigest`。
- 用 restore plan 拼一个 `InstallationManifest` 去调用，等于自造 plan 到安装记录的映射；design/ADR 没有这条映射，且该接口语义是「安装来源」而不是 restore 来源。正确表达必须改 `internal/**`——写集禁止，且属公共契约变化。
- 补充事实：生产 `sources` 当前即 `recoveryClosedPort`，`recoveryAnchorRegistry` 无注册适配器；即便调用也只能得到 closed。该检查在当前形态下不可能产生正向结果。

**B3 生产 restore driver 缺少 DSN 载体。**
- `controlledRestoreDriver` 构造要求非空 DSN，并把它写入私有 stdin（`cmd/han-server/recovery_restore.go:203`–`:208`、`:71`–`:81`）。
- `recoveryRootConfig` 与 `recoveryRootInputs` 都没有 DSN 字段（`recovery_config.go:27`–`:30`、`recovery_runtime.go:428`–`:448`）。从 `pgxpool` 配置反推 DSN 属未获授权的设计决定（把运行时连接串交给子进程），不属于本卡写集内的「必要装配」。

**B4 执行身份切换机制未选定，生产 driver 无法构造。**
- ADR 明确「具体身份切换机制……本 ADR 不选具体机制」（`framework.md:1116`、`:1135`）。卡面注记允许「机制未定先保持 closed + 负向测试」，但同时禁止预置执行环境；与 B1/B2/B3 叠加后，本卡既无法构造 `restoreSource`/`restoreTarget`/`restoreTool` 的可执行实现，也无法合法取得拒绝分支所需的批准输入。

**B5 POSIX 隔离预检无法在本写集内实现且保持 Windows 构建绿。**
- `st_dev`/`st_ino`、owner/ACL、受控只读挂载、`/etc/ld.so.preload`、`ld.so.cache`、RPATH/RUNPATH 都需要 POSIX 专用系统调用或 ELF 读取。`syscall.Stat_t` 在 Windows 上是另一套结构，直接引用 `Dev`/`Ino` 会破坏 `go build ./...`。
- 现有平台专用文件只有 `cmd/han-server/maintenance_stdin_windows.go` 与 `maintenance_stdin_other.go`，均不在写集内；本卡写集不允许新增 build-tag 文件。因此「真实预检」与「Windows 构建绿」在本写集内不可同时满足，只能登记为未执行（卡面 (f) 允许该豁免，但 (a)/(b) 不适用）。

**B6 负向用例 (a)(b) 在本机将是空真，除非给生产门加「测试专用」平台注入。**
- 本机为 Windows：平台检查一旦前置，所有用例都会在 `ErrAdmissionUnsupported` 短路，(a) 的相对路径/SHA 不符与 (b) 的批准缺失/自批永远不可达。
- 要可达，只能让生产门接受外部传入的伪平台或注入批准记录——这正是卡面 §2 禁止的「测试开关 / 为让路径跑通而新增的分支」，且测试证明的将不是生产行为。
- 若不注入，则 (a)(b) 只能断言「Windows → 拒绝」，与 (c) 重复，属空真断言，违反卡面「不得跳过实质断言」与 D08 安全侧 `F004` 的教训。

## 3. 为什么不以「恒拒绝门」凑合交付

可以写出一个只做前置检查、恒返回 closed 的 production gate，但它有以下问题，故不交付：
1. 无法评估卡面点名的批准记录三方相等、角色分离、有效期/撤销与 CAS 来源核验（B1/B2），所谓「评估」只剩路径/SHA/可读性，等于把缺口掩盖成已检查。
2. 永远不产生 driver，实质是把 `recoveryClosedRestoreDriver` 改名加日志；评审无法区分它与既有隐含 closed，却可能误以为「生产装配已落地」。
3. 要让 (a)(b) 可测就必须引入 B6 所述的测试注入，反而新增卡面禁止的开关。
4. 与「不得虚构任何批准记录/批准角色/工具」直接冲突：任何让它通过的分支都要求存在本卡不允许创造的批准数据。

## 4. 未改动声明

- Git：未执行任何写操作（无 `add`/`commit`/`branch`/`stash`/`checkout`/`restore`/`reset`/`clean`）。
- 文件：除本报告外未创建、未修改、未删除、未移动任何路径；`cmd/han-server/recovery_restore.go`、`recovery_runtime.go`、`main.go` 本次会话零写入。注意：这三个路径在 HEAD 中未被跟踪（`git status --short` 为 `??`），因此不能以 `git diff` 证明「与 HEAD 一致」；本轮只能证明本次会话没有写入它们。
- 禁改区：`internal/**`、`cmd/han-server/recovery_sources.go`、`cmd/han-server/recovery_config.go`、所有既有 `*_test.go` 均零改动。
- 未创建 `cmd/han-server/recovery_restore_production_integration_test.go`。

## 5. 需要 Lead 裁定的决策

- **D1（载体）**：是否为 restore 批准记录新增 design/ADR 修订与配置键（或明确其载体为既有 manifest 的扩展 wire），并相应放开 `recovery_config.go` 写集。
- **D2（接口）**：是否授权扩展 CAS 信任边界（新增恢复来源核验端口或扩展 `InstallationSourceVerifier`），使 `BackupRef`/`ArtifactDigest`/`ArtifactBytes`/`SourceSchemaDigest` 可被核验；这需要改 `internal/recovery` 或新增另一权威端口，二者均超出本卡写集。
- **D3（平台文件）**：是否授权新增 POSIX/Windows build-tag 文件，以承载 `st_dev`/`st_ino`、owner/ACL、挂载来源与加载器预检。
- **D4（降级范围）**：若维持现有写集，本卡只能降级为「显式 fail-closed 拒绝门 + 平台前置拒绝」，并须明确 (a) 三方相等、(b) 自批、(e) 见证核验由哪张后续卡取证；否则负向矩阵无法按卡面具名完成。

## 6. 未验证与 unknown

- 未验证：生产 restore 装配、POSIX 执行身份/进程组/ACL/GRANT、原生耐久回执、真实旧备份演练、加载器预检，全部未执行（本机 Windows，且无批准工具/记录）。
- unknown：批准记录载体与撤销/有效期读取路径；执行身份切换机制与具名主体；原生回执产品字段与读取命令；本机 PG DSN 可用性（因未创建测试而未测）。
- 生产 `enter`/`resume` 的 DB 替换仍 closed；ADR-0031 约束 8 未满足；生产未解除。