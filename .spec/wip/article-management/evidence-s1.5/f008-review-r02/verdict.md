# `S15-RE-F008` 复审裁定（R02）

- **裁定：`S15-RE-F008` = CLOSED。**
- 性质：非作者独立复审；**AI 代理评审，非真人签字**。
- 复审对象：`.spec/wip/article-management/evidence-s1.5/f008-allow-assertions-r02/`。
- 工作区 / 分支 / HEAD：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 上游 R01：NOT CLOSED，仅证据记账阻断（叶子数、耗时、gofmt 日志）；代码修复成立。
- 本轮结论：R01 的三项记账阻断已消除；5 点核定全部通过；无阻断 finding。CLOSED 仅针对 `S15-RE-F008`，不宣称 S1.5 整卡关闭。

## 五点逐项结论

1. **34 叶子 / 0 FAIL / 0 SKIP：一致。** `f008-targeted-normal.log` 有 34 个 subtest `--- PASS`、0 个 FAIL、0 个 SKIP，顶层 1 个 PASS；循环 5×8−6=34 亦独立吻合。`index.json` 的 `leaves` 记录一致。
2. **耗时 1.326s / 3.027s / 39.076s：逐字取自日志。** normal 末行 `1.326s`、race 末行 `3.027s`、full 全文 `39.076s`，与 `report.md:44-46` 和 `index.json` 的 `runs[*].duration` 完全相同。文件 mtime 顺序与单轮执行相容（旁证，非密码学证明）。
3. **`gofmt.log`：可接受为保存的空输出日志（有限证据）。** 文件 0 B、SHA256 为 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`；索引记录命令 `gofmt -l cmd/han-server/main_test.go`、exit 0；本次独立复跑 exit 0、0 行输出、0.148s。0 B 文件本身不能自证 provenance，需结合索引、mtime 与独立复跑理解，不阻断 CLOSED。
4. **`Allow` 精确断言与 `mainOrphanAllow` 映射：与 `main.go` 字面量一致。** `main_test.go:3402` 精确比较 `Header().Get("Allow")` 与 `mainOrphanAllow(t, action)`；`:3405-3406` 断言 `events == ["trace"]` 且 `resolver.calls == 0`。helper `:3411-3424`：list→GET，detail/delete→`GET, DELETE`，publish/transfer→POST，未知 action→`t.Fatalf`。production `main.go:3613/3615/3617/3653` 分别为 GET、POST、POST、`GET, DELETE`；`:3668-3675` 在 405 时先设 `Allow` 并立即返回，早于 probe/handler。
5. **`index.json` 各文件 SHA256：全部一致。** 9 个对象（index、report、3 份 run 日志、2 份空日志、`main_test.go`、`main.go`）的实际 SHA256 与字节均与给定值/索引值一致；测试后源文件哈希未变。

## 实跑命令与结果

| 命令 | exit | 耗时 | 结果 |
|---|---|---|---|
| `go test ./cmd/han-server -run '^TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity$' -count=1 -v` | `0` | wall `3.828s`；go 报告 `1.414s` | 34 subtest PASS；0 FAIL；0 SKIP |
| `go test -race ./cmd/han-server -run '^(TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity|TestOrphanDispositionProductionRoutes_ExactPathsAndSharedAdapter)$' -count=1 -v` | **NOT RUN** | 尝试 `0.093s` 后启动失败，exit `2` | `-race requires cgo`；本机 `CGO_ENABLED=0` 且无 gcc/clang/zig，未进入测试执行；另行审阅保存的 race 日志（34 目标 subtest + 2 顶层 PASS，0 FAIL/SKIP，`3.027s`） |
| `gofmt -l cmd/han-server/main_test.go` | `0` | `0.148s` | 输出 0 行 |
| `go vet ./cmd/han-server` | `0` | `1.264s` | 输出 0 行 |

未运行 full-package（不在允许范围）；仅审阅其 32 B 包级 `ok ... 39.076s` 日志。

## 保留限制

- `gofmt.log` 为 0 B，不能仅凭文件自证命令来源；接受依据是索引命令/exit、mtime 顺序与本次独立空输出复跑。若要求取证级 provenance，需补带命令/退出码的 wrapper transcript。
- race 未在本机复跑；保存日志提供逐叶子 PASS 证据，但日志本身不打印 `-race`，`-race` 事实依赖索引/报告记录。
- 保存日志无命令行、时间戳、exit code 头；单轮执行仅由 mtime 旁证。
- full-package 仅有包级 `ok`，无逐测试明细，且未复跑。
- `cmd/` 在 Git 中为 untracked；对象以 SHA256/字节固定，无法用提交 diff 固定基线。
- 工作树非 clean（大量预存变更/untracked）；本轮未触碰这些内容，也不评价 F008 之外的变更。
- 本轮只复审 F008 测试侧记账，不覆盖真实 PostgreSQL/Session/SQL、Journal/旧备份、Case18、应用角色 ACL 或部署 E2E。
- 本轮未修改源码/测试/tasks/Plan/ADR/既有证据，未执行 Git 写，未删除，未再 spawn。

## 声明

**AI 代理评审，非真人签字。** 本裁定不替代人的业务接收、独立评审签字、发布授权，也不宣称 S1.5 整卡关闭。