# Case18 隔离矩阵：当前 revision 执行证据

> **AI 代理执行与记录，非真人签字。** 本文件只登记一次真实执行的事实与哈希，不代替 Lead 的任务决定，不代替独立评审、人的业务接收或部署授权，也不宣称 S8.2 / Case18 / S1.5 已 PASS 或已关闭。

## 0. 目的与边界

本目录回答一个具体问题：Avicenna《Case18 覆盖对账》评审（`F002`）指出覆盖矩阵「用 fake runtime、只覆盖 closed/unknown、缺真实 PG/双系统」。其中可离线核实的一层是——**当前 revision 的矩阵测试是否真的被执行过、是否覆盖 25 条业务路由、是否含 guard_open 对照、普通与 race 是否都干净**。

本轮只补这一层，其余缺口保持原状并如实登记。

- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 执行者：主会话 Lead（未委派 Role，故不存在 Role 产出；本条按透明度要求显式声明）
- 授权范围：只读执行 + 只写本目录；未做 Git 写、未改既有测试、未改既有证据、未删除或移动任何文件
- 执行轮次：单轮，产物与哈希一一对应（`single_execution_round`）

## 1. 执行环境

- `go version go1.27.0 windows/amd64`
- `GOOS=windows`、`GOARCH=amd64`、`CGO_ENABLED=1`
- `HAN_TEST_POSTGRES_DSN` 置空（未连接数据库）
- 构建标签：`integration`
- 执行期间源码基线未漂移（见第 4 节哈希）

## 2. 执行的命令与结果

| # | 命令 | 退出码 | 产物 |
|---|---|---|---|
| 1 | `go build ./...` | 0 | 无落盘（构建成功，无输出） |
| 2 | `go test -tags=integration -v -run "^(TestRecoveryIsolationMatrix\|TestRecoveryMainBusinessDispatcher)$" -count=1 -json ./cmd/han-server` | 0 | `run-normal.json` |
| 3 | 同 2 但追加 `-race`（`go test -tags=integration -race -v -run "..." -count=1 -json ./cmd/han-server`） | 0 | `run-race.json` |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | `go-vet.log` |

## 3. 实算结果（从日志直接统计，非转述）

| 指标 | 普通跑 | race 跑 |
|---|---:|---:|
| test2json 事件总数 | 265 | 265 |
| `run` 事件 | 65 | 65 |
| `pass` 事件 | 66 | 66 |
| `fail` 事件 | 0 | 0 |
| `skip` 事件 | 0 | 0 |
| 叶子用例数 | 65 | 65 |
| 包级耗时 | 0.527s | 1.805s |
| `DATA RACE` 行 | 0 | 0 |

叶子构成（两跑完全相同）：

- `TestRecoveryIsolationMatrix/closed/*`：25
- `TestRecoveryIsolationMatrix/unknown/*`：25
- `TestRecoveryIsolationMatrix/guard_open_reaches_authorization_and_origin`：1
- `TestRecoveryMainBusinessDispatcher/*`：14

顶层用例（两跑均 PASS）：

- `TestRecoveryIsolationMatrix`：0.13s（普通）/ 0.17s（race）
- `TestRecoveryMainBusinessDispatcher`：0.31s（普通）/ 0.37s（race）

`go-vet.log` 内容为单行 `EXIT=0`，即 `go vet` 无诊断输出。

## 4. 源码绑定（执行期间未漂移）

| 文件 | SHA256 | 字节 |
|---|---|---:|
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 173317 |
| `cmd/han-server/main_test.go` | `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` | 400312 |
| `cmd/han-server/recovery_runtime.go` | `1C6793779E444BD2A4EC2C6DB324551A20152A7EB1B7A1C65A9C9DB11D4396E5` | 37431 |
| `cmd/han-server/recovery_main_dispatcher_integration_test.go` | `BD71F41132DFCB2135390A6CCF7373DBD556BF4A898EE547B4B608C7D360651A` | 23693 |
| `cmd/han-server/recovery_isolation_matrix_integration_test.go` | `DA02943B9848C50E1960ECCDBF210C3E69D3AA8A64034F9A6321740E519E9115` | 8498 |

被执行的矩阵测试实读事实（`recovery_isolation_matrix_integration_test.go`，183 行）：

- `recoveryIsolationBusinessRoutes()` 返回 **25** 条路由，覆盖会话建立、改密、管理员通知、管理员创建、隐私确认、用户/管理员密码重置与重发、文章违规与恢复、孤儿文章列表/发布/转移、文章列表/创建/详情/更新/删除、管理员文章列表/详情/删除、孤儿文章详情/删除。
- 每条路由在 `closed` 与 `unknown` 两状态下断言：HTTP 503、错误码 `platform.CodePlatformRecoveryInProgress`、响应 `Trace` 与响应头一致、无恢复态泄漏。
- 同时断言守卫**没有**到达授权、origin 保护、Session 解析、Article 仓储与作者读取；`acquires == len(routes)` 且 `releases == 0`。
- `/livez` 与 `/readyz` 位于守卫之外：`closed` 下分别为 200 `alive` / 503 `unavailable`。
- `guard_open_reaches_authorization_and_origin` 用 `public_ready` 证明观测点本身可达（201、origin 1 次、授权 `[create]`、acquire/release 1/1）。

## 5. 本次执行覆盖了什么

- 当前 revision 上，该矩阵测试**确实被执行**并有可复核的机器日志（普通 + race 双跑）。
- 25 条业务路由在 `closed` / `unknown` 下逐一被守卫拦住，且拦在授权与 origin 之前。
- 健康/就绪探针确实不受恢复态守卫影响。
- `guard_open` 对照证明该测试的观测点不是恒假（否则 25×2 的「未到达」断言毫无意义）。
- race 检测器零告警。

## 6. 本次执行**没有**覆盖（不得据此声称通过）

- **未覆盖** `failed` / `expired` / `undetermined` 等其它恢复态；只做了 `closed`、`unknown` 与 `public_ready` 对照。
- **未使用真实 runtime**：矩阵仍走 `newS419RecoveryRuntime` 假 runtime 与零值 `pgxpool.Pool`，本次执行不改变这一点，也不需要数据库。
- **未使用真实 PostgreSQL**：`HAN_TEST_POSTGRES_DSN` 为空，无 schema、无数据、无真实锁与事务。
- **未做真实双系统**：无指标/告警/job 侧的真实接线。
- `F001` / `F003` 所指的 production `Enter` / `Install` 路径仍为 closed，本次未触及。
- **完整恢复之后的生产开流仍为 NOT COVERED**（`F004` 结论不变）。
- 未做真实部署 E2E。
- 本次未做独立评审，`F002` 状态由 Lead 与既有评审结论共同判定，不由本文件自行关闭。

## 7. 与 Avicenna 评审 `F002` 的关系

`F002` 指向的结构性缺口（真实 runtime、真实 PG、真实双系统、更多恢复态）**依然存在**。本轮只把其中「当前 revision 是否有执行证据、路由覆盖面、普通/race 双跑」这一层补成可核事实，从而把 `F002` 从「疑似未执行」收敛为「已执行但覆盖范围有限」。该收敛不构成 PASS，也不解除 S8.2 的 `blocked-upstream`。

## 8. 复算方法

- 事件统计：逐行 `ConvertFrom-Json`，按 `Action` 计数；叶子数为 `Action == "pass"` 且含 `Test` 字段的事件数。
- race 判定：统计 `Action == "output"` 且 `Output` 含 `DATA RACE` 的事件数，应为 0。
- 哈希：`Get-FileHash -Algorithm SHA256`，字节数取自 `[IO.File]::ReadAllBytes().Length`。
- 因 `cmd/` 在本仓库未被 Git 跟踪，`git diff` 为空**不能**作为「未改动」的证据，故一律以 SHA256 + 字节数为准。