# article-management-S5.1 独立代码评审 R01

Code Review Gate: **PASS**

2026-10-27；objectType: code。执行者为当前非作者 Reviewer 会话，未实施被审代码，未委派；不使用作者自审结论替代独立审查。无 CRITICAL/HIGH/MEDIUM/LOW finding。结论仅适用于下列固定对象和有界范围，不代替 Lead 收口、人的接纳或真实 E2E。

## 固定对象与授权

- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。开始与结束核对 HEAD，目标四文件开始与结束 SHA256 一致，无漂移。
- 唯一写入：本报告。代码、任务、设计、配置只读；无 Git 写入、修复、清理或外部操作。
- 当前活动：development-delivery / review；交回 Lead，保持 S5.1 状态由 Lead 核证后处理。
- 合同：`tasks.md:1482` 的精确四文件及关闭条件；article design §3.3.6（`:1500`）、删除事务/恢复必要段（`:1202`、`:1469`、`:1498`、`:1891`）；test-plan A-MODR-01～02（`:227`、`:228`）。

| 路径（internal/article/） | SHA256 |
|---|---|
| command_service.go | B61ABD12ADD7AF43C7D677A89C097F44FFBB818A85D2D674CD218A81340DB6E4 |
| handler.go | B2E3F68995AEC9FA63C5E9F04EF0CCC458A4262045279D7A7DD794C2DD25DD00 |
| deletion_finalizer.go | E914B290DBCB59D24116ECBBD4592AD171038FCEA25081739AF4A6CFB17FD12C |
| admin_delete_test.go | 3ADEB5C7BC7E64239C8574BDD8CB9A41060E04289F043C1C0B3D3DA2311905CA |

已完整读取本端 add-reviewer Role、add-code-review Skill；读取 route-work、delivery-workflow 以定位已有交付的评审环节。适用依据包括 code-review-checklist、abstraction-principles、coding-standards、no-hardcode-policy、observability-standards、安全红线，以及启用 go-service 的 README、coding/security/observability/no-hardcode 规则。未启用其他 profile；无前端或新增跨服务追踪改动。

## 第一轮：固定实现与测试

全文审读新增 `admin_delete_test.go` 与 `deletion_finalizer.go`，核对 command service 的委托及 handler 的普通 DELETE、Principal、If-Match、错误与 204 边界。

1. `deletion_finalizer.go:548` 的事件构造保留原 version/kind，只在 ordinary 且 operation.ActorRole 非空、等于 admin 时增加 authority=administrator；nil/member/其他删除类型不进入新增分支。新增稳定字面量集中于模块常量，无新配置、schema、依赖或第二审计写入者。
2. 新测试确实经 HTTP authorizer → handler → CommandService → coordinator → 委托真实 finalizer。repository/journal/auditor/transaction 为既有 stub，不冒充真实数据库或持久卷。
3. 5 个测试、17 个子例：双完成 admin/member 两例；journal/finalizer pending 与原 ETag 重试两例；违规/无归属/held/其他成员/密码限制/隐私限制/操作者 suspended 七例；强 ETag 六例；另有管理员不得编辑他人的顶层测试。断言覆盖空 204、完成后 404、单 tombstone/单审计、版本/kind/actor/role/trace/authority、作者原 metadata 不变、无标题正文，以及拒绝时无不可逆副作用。
4. 失败注入返回 503 且无完成审计，解除故障后完成；测试没有绕过真实事件构造，也没有只检查直接 helper 输出。作者 RED 原文由交接提供：`admin audit metadata = map[string]interface {}{"article_version":3,"delete_kind":"ordinary"}, want authority=administrator`。本 Reviewer 未改代码重演 RED，不将交接原文记为亲自执行。

## 第二轮：上游与直接消费者

只沿授权调用链读取 coordinator、model/repository 的必要定义与持久化、deletion_test fixture/stub 与相关用例、policy、update_test 相关用例、main 普通 DELETE 接线和 audit 直接消费者。

1. **普通文章与作者状态**：`deletion_coordinator.go` 的 authorizeActiveDeletion 要求 AccessGeneral、合法 Article、active/published/clear/非空 author_id，并允许管理员或作者。设计“作者可以 active 或 suspended”指目标作者状态；此删除路径不调用成员可见性策略、不加载目标作者状态，故不会因目标作者 suspended 拒绝管理员。新测试的 suspended 子例修改的是操作者 Principal，期望 403 正确，不能当成目标作者 suspended 的实测证明。目标作者状态边界已通过调用链静态核查；没有真实 identity/数据库联动实测。
2. **可信角色**：handler 从 context 取 Principal，不从客户端 body/header 接受角色；main 的生产 authorizer 使用 session store resolver，普通 DELETE 由 Origin guard 和 AccessGeneral 保护。coordinator 在授权后将 Principal.UserID/Role 写入 operation；repository INSERT/SELECT 保存并读取 actor_role。finalizer 在完成事务重新锁定 operation，用 locked operation 构造事件，新增 authority 不取重试者声明或当前账户角色。
3. **事务与失败收敛**：finalizer 先核 tombstone 绑定与 lease，物理删除、Audit.Append、operation completed 使用同一个 tx，最后 Commit；任一失败返回错误，defer Rollback。audit Store 按调用方 tx 插入，JSON 序列化完整 Metadata，不丢 authority。新增字段不改变既有事务顺序。stub 不实现真实事务回滚，因此本次组件测试不能独立证明 PostgreSQL 原子性。
4. **恢复保留的精确边界**：已有 operation 的 pending 重试、后台 ConvergeDeletionOperation 及 ReplayAll 收敛读取持久化 ActorRole，最终仍生成 administrator authority；已 completed 不追加第二审计。若旧备份缺少 operation，rebuildOperationFromTombstone 只重建最小 operation，ActorRole/ActorUserID 为空，因此不会推测或伪造 administrator authority。design 明确最小责任主体不写外部 tombstone（`:1202`），此类恢复无法还原原管理员角色；不能宣称所有灾备重放都会保留 authority。本次未改变这一既有恢复协议，不据此扩成无关老缺陷评审。
5. **兼容性**：作者普通删除仍只有 version/kind 两项；violating/orphan action 和审计 metadata 不受新增条件影响。编辑授权保持独立，新测试实际用非作者 admin 发起更新，403 且零 update。没有发现本次变更引入的契约破坏或权限扩大。

## 验证与证据来源

Reviewer 实际执行（工作区根目录）：

```text
git rev-parse HEAD
git branch --show-current
git status --short
git diff --name-only -- internal/article/command_service.go internal/article/handler.go internal/article/deletion_finalizer.go internal/article/admin_delete_test.go
git diff --cached --name-only
Get-FileHash internal/article/command_service.go,internal/article/handler.go,internal/article/deletion_finalizer.go,internal/article/admin_delete_test.go -Algorithm SHA256
go test ./internal/article -run '^TestAdminDeleteOrdinaryArticle' -count=1 -v
```

专项 exit 0，5 个测试、17 个子例全部执行并 PASS，`ok han/internal/article 0.078s`。暂存清单为空。读取使用 `Get-Content -Encoding utf8`、`rg` 和必要行段；首次默认编码显示异常后已按 UTF-8 重读 Role/Skill。

Lead 本轮明确报告专项和 Manifest 七项固定顺序全部 PASS：`gofmt -l .`（无输出）、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。此为 Lead 现场验证声明，Reviewer 未重复全仓检查，也未将其记为本人命令。作者另报告专项普通/race、article 包普通/race/vet PASS；同样按来源保留。

## 限制、待决定与复审条件

- 作者未采集修改前哈希；共享树 `internal/` 为未跟踪目录，HEAD 无该实现基线，scoped git diff 空不证明没有变化。故不能独立字节级证明“只修改 finalizer、新增测试，command_service/handler 未改”或全树无第五文件修改；这部分依据交接声明与 Lead 初核，绝不反推前哈希或伪造旧 diff。本次固定后四文件未漂移，未发现需要修改第五实现文件。
- 未运行真实 PostgreSQL、多连接事务/commit-ack-loss、Unix flock/fsync/崩溃恢复、真实会话/目标作者 suspended 联动、生产 HTTP 路由实测、灾备重放或 E2E。没有将 stub tombstone 称为真实耐久证据。S4.3 经决策不验证的边界不由本报告覆盖。
- 无待决定的本次阻断项，无额外非阻断风格建议。恢复缺失 operation 的角色不可还原与真实环境未验按上述边界保留。
- 任一固定文件哈希、HEAD、上游合同变化，或修复要求进入第五文件时，停止受影响结论，交 Lead 重新固定范围；若后续出现 finding，由原作者整改并以原 finding ID、修复 revision 和相关回归交独立复审。

本任务范围两轮审查完成，0 finding；Code Review Gate: **PASS**。
