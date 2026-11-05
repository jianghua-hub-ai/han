# 项目 Manifest 七项实测（project-manifest-r01）

日期：2026-11-02 22:15–22:21（Asia/Shanghai）
性质：当前**工作树**的实测记录。不是 HEAD 提交的证据，不是 E2E，不是生产就绪，不解除任何 closed，不构成评审 PASS。
执行者：Lead（主会话），非 Role 产出。

## 1. 运行对象

| 项 | 值 |
|---|---|
| 仓库 | `D:\AID\VSCode\Codex\Han` |
| 分支 | `main` |
| HEAD | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 工作树 | 18 个跟踪文件已修改；281 条未跟踪条目（`git status` 折叠口径，含本轮新增的 3 个取证文件） |
| 工具链 | `go version go1.27.0 windows/amd64` |
| `go.mod` / `go.sum` | 400 B / 2,031 B，mtime 均为 2026-10-21，本轮命令未改动 |

**边界**：结果对应工作树内容，不对应 HEAD；工作树与 HEAD 的差异本轮未逐项审阅。

## 2. 结果矩阵

| # | 命令 | 退出码 | 结果 |
|---|---|---|---|
| 1 | `gofmt -l .` | 0 | 16 个文件被列出，**全部位于 `.spec/` 下**；`cmd/`、`internal/`、`test/` 零命中（详见第 4 节） |
| 2 | `go mod verify` | 0 | `all modules verified` |
| 3 | `go vet ./...` | 0 | 无输出 |
| 4 | `go build ./...` | 0 | 无输出 |
| 5 | `go test -tags=contract ./... -count=1` | 0 | 16 个包全 `ok`（含 `test/contract/article`、`test/contract/identity`） |
| 6 | `go test -race ./... -count=1` | 0 | 14 个包全 `ok`，无 data race 报告 |
| 7 | `go test ./... -count=1` | 0 | 14 个包全 `ok` |

## 3. 包级耗时（摘自命令输出，非估算）

- `go test ./...`：`cmd/han-server` 24.792s、`internal/article` 2.503s、`internal/idempotency` 3.287s、`internal/identity` 2.676s、`internal/mailer` 4.009s、`internal/recovery` 2.874s，其余包均 < 0.4s；`internal/audit` 与 `test/performance/article` 无测试用例。
- `go test -race ./...`：`cmd/han-server` 91.411s、`internal/recovery` 13.067s、`internal/mailer` 9.874s，其余包 < 6s。
- 墙钟合计约 4 分 20 秒（build 4s + vet 3s + test 28s + contract 30s + race 154s + verify 3s，含进程启动开销）。

## 4. `gofmt -l .` 异常明细（16 个文件，全部在 `.spec/`）

以下文件以 `.go` 后缀落在证据目录中，内容为带行号的源码摘录或补丁片段，开头不是 `package` 声明，因此 `gofmt` 报 `expected 'package'`：

- `.spec/wip/identity-auth/evidence-s10.5/cleanup-review-r01/reviewer/scram_cpu_probe.go`
- `.spec/wip/identity-auth/evidence-s10.5/cleanup-review-r01/reviewer/tls_close_probe.go`
- `.spec/wip/identity-auth/evidence-s10.5/pg-b-author-r49-r02/role/carrier.D452.old.go`
- `.spec/wip/identity-auth/evidence-s10.5/pg-b-author-r49-r02/role/probe-new-metadata/password_reset_maintenance_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/pg-b-author-r49-r02/role/probe-new-query/password_reset_maintenance_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/pg-b-author-r49-r02/role/probe-old-metadata/password_reset_maintenance_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/pg-b-author-r49-r02/role/probe-old-query/password_reset_maintenance_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-f003-author-r01/role/clock-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-f003-author-r01/role/reviewer-probe-current-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-f003-author-r01/role/reviewer-probe-precondition-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-r49-review-r01/reviewer/reviewer-overlay-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-r49-review-r02/reviewer/author/role/clock-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-r49-review-r02/reviewer/author/role/reviewer-probe-current-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-r49-review-r02/reviewer/author/role/reviewer-probe-precondition-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-r49-review-r02/reviewer/green-probe-main_test.go`
- `.spec/wip/identity-auth/evidence-s10.5/root-r49-review-r02/reviewer/red-probe-main_test.go`

结论：这不是源码格式问题。`cmd/`、`internal/`、`test/` 三个源码树在 `gofmt -l` 下零命中。

## 5. 待决项

**D1（gofmt 作用域）**：Manifest 第 1 项按 `gofmt -l .` 全仓扫描，会因 `.spec/` 下的摘录文件而永远非空，导致该项无法作为「源码已格式化」的判定依据。
（2026-11-02 22:30 更新：D1 已按建议值处置，见下）

- 影响：只影响判定可读性，不影响代码正确性；不影响本轮 2–7 项结论。
- 建议值：把该项收窄为 `gofmt -l ./cmd ./internal ./test`，或在证据约定中禁止摘录文件使用 `.go` 后缀。
- 状态：**已实施**（2026-11-02 22:30）。`add.manifest.yaml` 两处检查清单条目已由 `gofmt -l .` 改为 `gofmt -l ./cmd ./internal ./test`（3,056 B → 3,100 B）；收窄后实测未格式化文件 0 个。全仓口径不再作为项目检查命令，第 4 节的 16 条 `.spec/` 命中保留供追溯；`go.mod`/`go.sum` 未受影响。
- 未决期间的口径：`gofmt -l .` 的 16 条命中**不得**记为「源码未格式化」，也不得据此判定 Manifest 第 1 项通过或失败。

**D2（AGENTS.md 检查命令投影不一致）**：`AGENTS.md` 的 ADD 受管块内「项目检查命令」仍为「未配置」，而 `add.manifest.yaml` 实有 7 条命令。该不一致在本轮之前已存在——`AGENTS.md` 相对 HEAD 无改动。

- 影响：不影响任何检查的可执行性，命令来源以 `add.manifest.yaml` 为准；只影响入口可读性。
- 处置建议值：**不手改、不跨版本重投影**。`AGENTS.md` 受管块按「不另行手改一套值」原则由采用配置投影；当前可用的 `add-agents-md` 技能属 ADD 0.11.0，本项目 manifest 为 0.9.5，跨版本重投影与「未知新版本先验证语法与真实 smoke，失败不覆盖既有投影」冲突。
- 状态：**保持现状，登记为已知残余**。重开条件：取得与本项目 0.9.5 匹配的投影能力，或明确升级框架版本后按采用流程重投影并复核整个受管块。
- 边界：本条不改变任何检查命令，不改变 `AGENTS.md` 内容。

**D3（凭据扫描口径与结论）**（2026-11-02 追加，位于 D1/D2 之后）：对本工作树做了一次只读凭据扫描（含未跟踪文件、排除 `.git`），结论为**未发现可用于认证的真实凭据**。本节为第 1–4 节原始记录之外的追加，不改动原记录。

| 检查项 | 命中 | 分诊 |
|---|---|---|
| 私钥头 `-----BEGIN ... PRIVATE KEY-----` | 0 | — |
| AWS Access Key（`AKIA…`） | 0 | — |
| `sk-` 形态 token（加左边界后） | 0 | 未加边界时的 6,329 条全部为 `task-…` 等词内误报 |
| PostgreSQL 内联凭据 DSN | 30 | 全部占位符（`user:pass@`、`redacted`、`<…>`）或短值；候选 0 |
| 口令赋值（含 `DB_PASSWORD=` 前缀形式） | 1,322 | 真实源码与 `.add` 范围内 19 条，脱敏回看 13 条全为函数签名 / `true` / `false` / `synthetic` 之类误报；候选 0 |
| 高熵 token 字面量（`cmd`/`internal`/`test`/`deploy`/`docs`，≥20 字符） | 0 | — |
| `litellm` 字面量 | 0 | 仓库内不存在该键 |
| 疑似密钥文件名 | 68 | 全为文件名含 `credential` 的源码与 `deploy/processing-gate.env.example`；无真实 `.env`、无私钥文件 |

- 口径局限：`rg` 默认跳过二进制文件；本轮**未做 git 历史扫描**，历史提交中的凭据不在本结论范围；仅做模式命中与占位符分诊，未做熵值阈值判定；全程只记路径、行号与哈希前缀，未落任何明文。
- 处置：**无需清理动作**。`.gitignore` 当前只含 `.add/local/**` 与 `Temp/`，未覆盖 `.env`；因仓库内无 `.env`、`deploy/processing-gate.env.example` 为纯占位示例，本轮不顺手修改 `.gitignore`，登记为观察项。
- 边界：本结论**不能**替代「仓库从未提交过凭据」这一断言；该断言需 `git log -p` 全历史扫描，属额外授权范围。

**D4（悬置事项按建议值处置，2026-11-02 追加）**：在用户 standing 授权「对待确认或决策项采用建议值并继续」下，对三项悬置事项给出处置结论；本节只作记录，不改代码、不改 `tasks.md`、不改 `.gitignore`。

| 事项 | 处置 | 依据 |
|---|---|---|
| `tasks.md` 2026-10-31 登记「5 处历史证据仍含 loopback 合成 DSN 明文」 | **无需动作——该项已过期** | 只读复核这 5 处（`identity-auth/review-code-s6.1-r01.md:118`、`recovery-runtime/evidence-b13h-r01/s419-author-r02/report.md:112`、`s419-author-r03/report.md:49`、`s419-independent-review-r02/report.md:177`、`s419-lead-gate-r03/report.md:37`）：口令字段实测均为字面量 `<redacted>`（10 字符，SHA256 前缀 `3a2d1afa` 与 `<redacted>` 的哈希一致），已非真实口令 |
| `.gitignore` 未覆盖 `.env` | **不改，保持观察项** | 仓库内不存在任何真实 `.env`；唯一同类文件 `deploy/processing-gate.env.example` 为纯占位示例；修改属当前目标之外的相邻改动 |
| 框架 `0.9.5` → `0.11.x` 升级 | **暂不升级** | 跨版本重投影须先验证语法与真实 smoke，失败不得覆盖既有投影；升级会触及 `AGENTS.md` 受管块与全部 Role/Skill 投影，需要独立验证预算，当前无该预算 |

- DSN 复核口径（全仓 30 处命中，口令字段去重后 6 种形态，按掩码结构记录）：`<redacted>`×15、`xxx-xxxxxx`（含 `secret` 子串）×10、`<xx>`×2、`...`×2、`<xxxxx≥x>`×1、`xxxxxx`×1。**真实凭据候选 0**。
- 重开条件：出现真实 `.env` 或新的 `.gitignore` 覆盖需求；框架升级获批并分配验证预算；第二具名保管人与独立主机到位。

## 6. 未运行（一律 NOT RUN）

`go test -tags=integration ./...`、真实 PG 集成测试、E2E 旅程、真实恢复演练（业务库回退 / anchor 整实例回退并重启 / 新副本冷启动）、四档回退取证、TLS 握手验证、凭据 fd 注入、独立备份链、独立评审、人的签收。

## 7. 复现

在仓库根目录按序执行：

```
go version
git rev-parse HEAD
gofmt -l .
go mod verify
go vet ./...
go build ./...
go test -tags=contract ./... -count=1
go test -race ./... -count=1
go test ./... -count=1
```

## 8. 复验记录（2026-11-02 22:32）

口径：HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`、`go1.27.0 windows/amd64`、`HAN_TEST_POSTGRES_DSN` 置空（不接真实 PostgreSQL），按 `add.manifest.yaml` 中 D1 收窄后的第 1 项重跑全部 7 项。本节为追加记录，不改动 §1–§4。

| # | 命令 | 退出码 | 耗时 | 关键输出 |
|---|---|---|---|---|
| 1 | `gofmt -l ./cmd ./internal ./test` | 0 | 0.26s | 0 行 |
| 2 | `go mod verify` | 0 | 0.77s | `all modules verified` |
| 3 | `go vet ./...` | 0 | 1.41s | 0 行 |
| 4 | `go build ./...` | 0 | 2.55s | 0 行 |
| 5 | `go test -tags=contract ./... -count=1` | 0 | 28.14s | `ok han/cmd/han-server 25.655s`、`ok han/internal/article 2.234s` |
| 6 | `go test -race ./... -count=1` | 0 | 101.23s | `ok han/cmd/han-server 88.385s`、`ok han/internal/article 5.014s` |
| 7 | `go test ./... -count=1` | 0 | 30.12s | `ok han/cmd/han-server 27.276s`、`ok han/internal/article 2.271s` |

- 结论：7/7 通过；第 5–7 项各包均为 `ok`，无 `FAIL`、无 `DATA RACE`、无未预期跳过。
- 耗时口径：第 1–4 项为 22:32 复测（构建缓存已热，数值偏小）；第 5–7 项取自同一批次实际运行。第 1–4 项在同一批次亦为 exit 0、0 行输出。
- 第 1 项采用 D1 收窄口径。`gofmt -l .` 全仓口径下 §4 的 16 条 `.spec/` 命中仍仅来自摘录文件，不作为「源码未格式化」依据，见 D1。
- 范围不变：本轮仍不含真实 PG 集成、E2E 旅程、恢复演练、TLS 握手、凭据 fd 注入、独立评审与人签收，§6 的 NOT RUN 清单继续有效。本记录不把单机/合成条件写成「已验证」。
- 推进判断：在上述约束下项目侧无新增可单方面推进项；重开需 D2/D4 所列外部前提（第二具名保管人、独立主机、框架升级预算）。

## 9. Linux 首次全量验证（2026-11-02，追加）

口径：同一 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；环境为 WSL2 Ubuntu / `go1.27.0 linux/amd64`；工作副本 `~/han-posix-r01`（ext4，排除 `.git` 与 `Temp`），未改动本仓库工作树、未做 Git 写。本节为追加，不改动 §1–§8。

| 步骤 | 修复前 | 修复后 |
|---|---|---|
| `go build ./...` | 通过 | 通过 |
| `go vet ./...` | 通过 | 通过 |
| `gofmt -l ./cmd ./internal ./test` | — | 0 行 |
| `go test ./... -count=1`（umask 022） | 失败：`internal/recovery`、`internal/article` | **通过**，exit 0 |
| `go test ./... -count=1`（umask 077） | 失败：`internal/article` | **通过**，exit 0 |
| `go test -race ./... -count=1` | — | **通过**，exit 0，12 个包 `ok`、2 个无测试用例，无 data race |

- 根因与修复：三类夹具缺陷（`t.TempDir()` 末级权限受 umask 影响、`writeInventory` 固定 `0600` 与只读判据冲突、replay 依赖夹具未创建 journal 根目录），修复仅触及 4 个测试文件，产品代码与权限判据未改。详见 `.spec/wip/env-capability-r01/linux-firstrun-r01.md`。
- 意义：§8 的 Windows 七项因 `articleDeletionJournalPlatformSupported()` 在 Windows 返回 false 而整体放行相关 POSIX 判据（F3），Linux 首次执行才暴露该夹具缺口；修复后两平台结论一致。
- 仍未运行（继续 NOT RUN）：迁移、真实 PG 集成、E2E 旅程、恢复演练。WSL `-race` 已补齐（用户空间 `gcc-11`，未用 sudo、未改系统）。本记录不把 Linux 全量测试写成 E2E 或恢复链已验证。

本文件位于 `.spec/wip`，属临时交接物，按清理清单确认流程处置，不自动删除。
