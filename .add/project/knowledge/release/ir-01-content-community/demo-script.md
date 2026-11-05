# IR-01 演示脚本（内部，约 10 分钟）

## 前置条件

- Windows + PowerShell，工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abb…`。
- Go 工具链可用；无需数据库即可完成契约演示。
- 真实 PG 子集演示需要 `HAN_TEST_POSTGRES_DSN` 与 PostgreSQL 18.6；不具备时跳过并明确说明。
- 本机 `-race` 不可用（`-race requires cgo`），race 结论以保存日志为准。

## 步骤 1：契约总览（约 30 秒）

```powershell
go test -tags=contract ./... -count=1
```

- 观察点：全部包 `ok`；`go test` 汇总行显示耗时（约 29s）。
- 失败预案：记录完整输出与 exit code，停止演示；不得口头解释为“环境问题”后继续。

## 步骤 2：16 端点与 orphan 405 安全边界（约 5 秒）

```powershell
go test ./cmd/han-server -run "^TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity$" -count=1 -v
```

- 观察点：34 个叶子 `--- PASS`，0 `FAIL`，0 `SKIP`；`Allow` 为 `GET` / `GET, DELETE` / `POST` 精确值。
- 讲解点：404/405 在安全链之前返回，事件序列只有 `trace`，resolver 调用次数为 0。
- 失败预案：若叶子数变化，先核对当前 `main_test.go` 哈希（应为 `3720652E…`/400,312 B）。

## 步骤 3：质量门禁（约 3 秒）

```powershell
go vet -tags=contract ./...
gofmt -l cmd internal test
```

- 观察点：两条命令 exit 0 且无输出。
- 失败预案：gofmt 若输出文件路径，先确认是否来自 `.spec/wip/**` 证据快照；源码目录应保持 0 行。

## 步骤 4：证据走查（只读，约 3 分钟）

依次打开并讲解：

1. `.spec/wip/article-management/verification-s1.5.md` — 顶部状态横幅、10 项关闭条件、Gate 与接收登记。
2. `.spec/wip/article-management/evidence-s1.5/wholecard-review-r02/verdict.md` — 非作者复审 PASS FOR LEAD GATE。
3. `.spec/wip/article-management/acceptance-s1.5-r01.md` — ADR-0028 AI 代理接收登记。
4. `.add/project/knowledge/decisions/framework.md` 的 ADR-0036/0037 — 范围决策与未验证边界。

- 观察点：每份材料都带 SHA256 与字节数；复审结论与产物路径一一对应。
- 讲解点：范围决策关闭 ≠ 验证通过；生产恢复入口仍 fail-closed。

## 步骤 5：边界声明（30 秒，必须讲）

- 未做部署后真实 E2E、真实 observability 演练、生产恢复链（ADR-0036/0037）。
- 未验证真实 Journal/HMAC/keyring、旧备份恢复、Case18 三状态、容量与性能（ADR-0033/0034/0035）。
- 本版为测试项目交付，不宣称 production readiness。

## 失败预案汇总

| 现象 | 处置 |
|---|---|
| 契约测试失败 | 保留完整输出与 exit code，停止演示，登记问题；不降级为“跳过”。 |
| 叶子数不是 34 | 核对 `main_test.go` 哈希与 `f008-allow-assertions-r02/index.json`。 |
| 无 PG 环境 | 跳过真实层子集，明确说明“未验证”，不引用历史数字替代现场。 |
| `-race` 需求 | 说明本机无 CGO，引用保存的 race 日志与复审结论。 |

