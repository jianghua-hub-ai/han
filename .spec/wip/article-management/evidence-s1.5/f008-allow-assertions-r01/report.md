# S1.5 `S15-RE-F008` 修复证据 r01（orphan 路由 405 `Allow` 与仅 trace 断言）

- 性质：测试侧修复，仅改 `cmd/han-server/main_test.go`；生产代码零改动。
- 执行者：Lead（本会话主会话直改并亲跑），非子代理产出，无 Role 交接。
- 时点：2026-10-31；工作区 `D:\AID\VSCode\Codex\Han`；分支 `main`；`HEAD` `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 授权来源：用户 2026-10-31 持续授权“对待确认或决策项采用建议值并继续本项目”；按 ADR-0028 由 AI 代理执行并登记，不冒充真人签收。

## 1. 问题

非作者独立评审 `review-code-s1.5-root-endpoints-r01.md` 的 `S15-RE-F008` 指出：`GET /api/v1/admin/orphan-articles`、`POST /api/v1/admin/orphan-articles/{id}/publish`、`POST /api/v1/admin/orphan-articles/{id}/transfer` 三条路由的 method 变体只断言 405 状态，缺 `Allow` 精确值与“仅 trace 事件”断言。

落点：`cmd/han-server/main_test.go` 的 `requireOrphanProductionRejection`（改前 :3349-3397），`variant == "method"` 分支只比较状态码，未校验 `Allow`，也未证明拒绝发生在 Session/CSRF/principal 解析之前。

## 2. 改动（仅 main_test.go）

- helper 内注入 `trace` seam，记录实际经过的事件序列（`events`）。
- 405 分支新增 `Allow` 精确相等断言：`response.Header().Get("Allow") == mainOrphanAllow(action)`。
- 405 分支新增 `!reflect.DeepEqual(events, []string{"trace"}) || resolver.calls != 0` 断言，即 method 拒绝时只有 trace 事件、principal resolver 零调用。
- 新增 helper `mainOrphanAllow`：`list`→`GET`；`detail`/`delete`→`GET, DELETE`；`publish`/`transfer`→`POST`，与生产路由 `main.go:3612/3614/3616/3652` 的 `allow` 字面量逐字对应。
- 同一 helper 顺带覆盖 `detail`/`delete` 的 method 变体（期望同为 `GET, DELETE`），不改动其他既有断言。

| 文件 | 改前 | 改后 |
|---|---|---|
| `cmd/han-server/main_test.go` | `0E61F4A37D314D2D35B80EA8184BEF17EB6227FCAF1CA0F57C09736005185820` / 399522 B | `220A49CD813C02C2224293EDE2E5D4B9350F550C118552D302B5A24DC6DE453B` / 400192 B |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` / 173317 B | 同左（未改动） |

## 3. 运行结果

| 命令 | exit | 结果 |
|---|---|---|
| `go test ./cmd/han-server -run 'TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity' -count=1 -v` | 0 | PASS；33 个叶子全 PASS，含 `list/method`、`publish/method`、`transfer/method`、`detail/method`、`delete/method` |
| `go test ./cmd/han-server -run 'orphan|Orphan' -count=1 -race` | 0 | `ok han/cmd/han-server 2.891s` |
| `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 40.455s` |
| `go vet ./cmd/han-server` | 0 | 无输出 |
| `gofmt -l cmd/han-server/main_test.go` | 0 | 空（已格式化） |

日志与哈希见同目录 `index.json`。

## 4. 有效性论证与负向对照

- 三条目标路由期望值互不相同（`GET` / `POST` / `POST`）：若新断言写成常量或比较失效，`list/method` 或 `publish/method` 会立即失败，因此当前通过即证明断言按 action 实际生效。
- `events == []string{"trace"}` 与 `resolver.calls == 0` 同时成立，说明 405 响应发生在 platform trace 之后、业务安全链之前，与生产实现 `main.go:3668-3675`（`Allow` 设置后立即 `WriteError` 返回）一致。
- 断言值来自生产路由字面量，不是从被测代码运行结果反推的常量。

## 5. 边界

- 只证明进程内真实业务 root 的 method/`Allow`/事件次序，不证明真实 Session/SQL、真实 PostgreSQL、Journal/旧备份、Case18 全局恢复隔离、应用角色 ACL 或部署 E2E。
- `cmd/` 未纳入 Git（untracked），一致性只用 SHA256/字节核对；本轮未做任何 Git 写、未清理任何文件。
- 本修复尚未经非作者独立复审，也未取得人的业务接收；不得据此宣称 S1.5 整卡关闭，`[-]` 与十二 requires 不变。