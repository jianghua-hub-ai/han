# S1.5 `S15-RE-F008` 修复证据 r02（orphan 路由 405 `Allow` 与仅 trace 断言）

- 性质：测试侧修复，仅改 `cmd/han-server/main_test.go`；生产代码零改动。
- 执行者：Lead（本会话主会话直改并亲跑），非子代理产出，无 Role 交接。
- 时点：2026-10-31；工作区 `D:\AID\VSCode\Codex\Han`；分支 `main`；`HEAD` `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 授权来源：用户 2026-10-31 持续授权“对待确认或决策项采用建议值并继续本项目”；按 ADR-0028 由 AI 代理执行并登记，不冒充真人签收。
- 替代关系：本目录取代 `f008-allow-assertions-r01` 的**记账**结论。r01 的代码修复与日志真实有效，但 r01 报告把 34 个叶子写成 33 个、耗时取自另一次执行，故不足以自证；r01 全部文件原样保留供追溯。

## 1. 问题与来源

非作者独立评审 `review-code-s1.5-root-endpoints-r01.md` 的 `S15-RE-F008` 指出：`GET /api/v1/admin/orphan-articles`、`POST /api/v1/admin/orphan-articles/{id}/publish`、`POST /api/v1/admin/orphan-articles/{id}/transfer` 三条路由的 method 变体只断言 405 状态，缺 `Allow` 精确值与“仅 trace 事件”断言。

落点：`cmd/han-server/main_test.go` 的 `requireOrphanProductionRejection`，`variant == "method"` 分支原只比较状态码。

## 2. 对 r01 的复审判定与本轮处置

非作者复审（`add-reviewer`，actualID `01a19285-8ece-7c20-a161-c0d7b8cce6aa`）判定 `S15-RE-F008` NOT CLOSED，理由是证据记账而非代码逻辑：

| 复审项 | 级别 | 本轮处置 |
|---|---|---|
| 实际 34 个叶子，r01 写成 33 | M | 叶子数改为由日志实算，本报告全部计数取自 `f008-targeted-normal.log` / `f008-targeted-race.log` |
| r01 耗时 1.36s/2.891s/40.455s 与日志 1.380s/3.009s/39.555s 不符 | M | 本轮五个日志与耗时来自**同一次执行**，耗时逐字取日志 |
| `gofmt` 结论无保存日志 | L | 新增 `gofmt.log`（exit 0、输出空） |
| race 日志仅一行 `ok`，无法证明命中哪些叶子 | L | race 改为带 `-v`，日志含 `=== RUN`/`--- PASS` 逐叶子记录 |
| `mainOrphanAllow` 的 `default` 静默返回 `POST` | L | 增加 `t` 参数，`default` 改为 `t.Fatalf`；`list`/`detail`+`delete`/`publish`+`transfer` 分支显式列出 |

## 3. 改动（仅 main_test.go）

- helper 内注入 `trace` seam，记录实际经过的事件序列（`events`）。
- 405 分支新增 `Allow` 精确相等断言：`response.Header().Get("Allow") == mainOrphanAllow(t, action)`。
- 405 分支新增 `!reflect.DeepEqual(events, []string{"trace"}) || resolver.calls != 0` 断言，即 method 拒绝时只有 trace 事件、principal resolver 零调用。
- `mainOrphanAllow(t, action)`：`list`→`GET`；`detail`/`delete`→`GET, DELETE`；`publish`/`transfer`→`POST`；未知 action 直接 `t.Fatalf`。期望值与生产路由 `main.go:3613/3615/3617/3653` 的 `allow` 字面量逐字对应。
- 不改动其他既有断言，不改生产代码。

| 文件 | r01 终态 | r02 终态 |
|---|---|---|
| `cmd/han-server/main_test.go` | `220A49CD813C02C2224293EDE2E5D4B9350F550C118552D302B5A24DC6DE453B` / 400192 B | `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` / 400312 B |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B | 同左（未改动） |

## 4. 运行结果（单轮执行，耗时逐字取日志）

| 命令 | exit | 结果 |
|---|---|---|
| `go test ./cmd/han-server -run '^TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity$' -count=1 -v` | 0 | `ok han/cmd/han-server 1.326s`；34 个叶子 PASS、0 FAIL、0 SKIP |
| `go test -race ./cmd/han-server -run '^(TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity|TestOrphanDispositionProductionRoutes_ExactPathsAndSharedAdapter)$' -count=1 -v` | 0 | `ok han/cmd/han-server 3.027s`；34 个目标叶子 + `TestOrphanDispositionProductionRoutes_ExactPathsAndSharedAdapter` 均 PASS，0 FAIL、0 SKIP |
| `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 39.076s` |
| `go vet ./cmd/han-server` | 0 | 无输出（0 B 日志） |
| `gofmt -l cmd/han-server/main_test.go` | 0 | 空（0 B 日志） |

`main_test.go` 在执行前后 SHA256 相同（`3720652E…` / 400312 B），证明本轮测试运行没有改写被测源码。

## 5. 有效性论证与负向对照

- 三条目标路由的期望值覆盖两种不同取值（`list`→`GET`，`publish`/`transfer`→`POST`），另有 `detail`/`delete`→`GET, DELETE`；若新断言退化为单一常量或比较失效，`list/method` 或 `publish/method` 会立即失败。
- `events == []string{"trace"}` 与 `resolver.calls == 0` 同时成立，说明 405 响应发生在 platform trace 之后、业务安全链之前，与生产实现 `main.go:3668-3675`（`Allow` 设置后立即 `WriteError` 返回）一致。
- 断言值来自生产路由字面量，不是从被测代码运行结果反推的常量。
- `mainOrphanAllow` 现对未知 action 直接失败，未来新增 action 若不补分支会立刻暴露，不会静默按 `POST` 通过。

## 6. 边界

- 只证明进程内真实业务 root 的 method/`Allow`/事件次序，不证明真实 Session/SQL、真实 PostgreSQL、Journal/旧备份、Case18 全局恢复隔离、应用角色 ACL 或部署 E2E。
- `cmd/` 未纳入 Git（untracked），一致性只用 SHA256/字节核对；本轮未做任何 Git 写、未清理任何文件。
- `trace` seam 只记录事件序列，不能证明“绝对无任何副作用”（未对 DB/session/日志做全量探针）。
- 本修订待非作者独立复审；未经复审前不宣称 `S15-RE-F008` CLOSED，不宣称 S1.5 整卡关闭，`[-]` 与十二 requires 不变。