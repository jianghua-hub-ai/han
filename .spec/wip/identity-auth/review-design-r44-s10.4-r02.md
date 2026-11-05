# identity-auth design r44 / S10.4 针对性复审 R02

- 适用日期：2026-10-24
- Reviewer：原独立 Reviewer，非被审设计原作者
- objectType：design
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / 基线 HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- revision：`identity-auth-design-ir01-r44`
- 原报告：`.spec/wip/identity-auth/review-design-r43-s10.4-r01.md`
- 原报告 SHA256：`DE750BE1CC2B2B493E4376512ADACF54255AC901EDCDD83829879FDA42E9D7D8`
- 复审范围：r44 中 S10.4 production worker/config/lifecycle 对 R01 finding 的定点修订
- **Design Review Gate：PASS**

本报告只复审原 finding、原待决 Article Start 及修订直接影响面，不重新审查无新证据的历史关闭决定，不批准实施、部署或 production readiness。

## 1. 固定对象与完整性

| 对象 | 开始 SHA256 | 结束 SHA256 | 结果 |
|---|---|---|---|
| `design.md` | `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23` | `50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23` | MATCH；评审期间对象未变化 |

评审未触发停止条件：未发现必须改变 accepted ADR-0023、architecture、boundary、公共契约或增加第九个 S10.4 production 文件的证据。

## 2. 两轮实际检查范围

### Round 1：r44 定点设计复审

逐项检查：

1. R01 F001：S10.4 与未来 maintenance mode 的能力、入口和文件所有权分界；
2. R01 F002：connection permit 的 owner、单位、路径、持有与释放合同；
3. R01 F003：五项配置内在失败与 `MaxConns` 交叉失败的唯一启动时序；
4. 原待决 Article Start：双 worker 的 Reconcile、Start、Done/Err 与 Serve 顺序；
5. 正文、接口表、测试矩阵、八文件影响面、交接说明和状态声明的一致性。

### Round 2：真实消费者与上游依据复审

实际读取并核对：

- Reviewer Role、`add-code-review`、评审清单、抽象原则、设计写作和禁止硬编码规则；
- go-service profile 中与接口、并发、context、资源释放、显式配置和审计相关的现役规则；
- 原 R01；
- `cmd/han-server/main.go`、`main_test.go`、`lifecycle.go`、`lifecycle_test.go`；
- `cmd/han-server/database_config.go`、`database_config_test.go`；
- `internal/article/deletion_worker.go`；
- `internal/identity/password_reset_store.go`、`password_reset_store_test.go`；
- architecture、identity boundary、decisions 索引定位的 `.add/project/knowledge/decisions/identity.md:275-331` ADR-0023；
- Tasks/Plan 的 S10.3、S10.4、B6C-3、B6D/S11.1 当前状态与旧六文件登记。

## 3. 原 finding 复审

### `IDENTITY-DESIGN-R43-S10.4-R01-F001` — HIGH — **CLOSED**

r44 已消除 S10.4 的生产维护入口和授权歧义：

- `design.md:2243-2247` 明确 S10.4 只交付 internal operator 和自动 worker；三个动作仅为后续稳定候选，当前不存在 production 入口、命令解析、授权入口或可演练 runbook。S10.5 完成前 dead-letter 持续 Gate/hold、不可运维恢复；
- `design.md:2245` 固定候选 `identity-auth-S10.5` 由 Task-Planner 后续登记，顺序接管 `main.go/main_test.go`，采用与 server 互斥的 maintenance mode，并要求后续补齐角色、审批/理由、evidence、before/after、AuditEvent、退出码和失败隔离；
- `design.md:2256,2880,3379` 明确 direct operator 调用只是 internal 可测合同，不构成 production caller 或 runbook 命令；
- `design.md:3571-3595` 将 S10.4 精确限制为八文件内的普通 server config/operator/worker/runtime wiring，明确不实现 maintenance mode。

真实 `main.go` 仍只有唯一 server composition root；八文件增加 main 两文件后足以承接普通 production construction/config/wiring。S10.5 尚未登记属于 PASS 后 Task-Planner 交接事项；因为 r44 未把候选冒充现役入口，也未声称 dead-letter 可恢复，所以“候选 + 明确后续登记/顺序所有权 + 当前不可执行”足以关闭本 finding。

### `IDENTITY-DESIGN-R43-S10.4-R01-F002` — MEDIUM — **CLOSED**

`design.md:696` 已固定完整且单一的 permit 合同：

- owner 是 server 进程内唯一 connection-permit owner；
- 一个 permit 是最多同时占用一个 pool connection 的顶层数据库操作/事务槽；
- 在首个 query/`BeginTx` 前恰好 acquire，一直持有到 `Rows`、`Tx` 和显式/隐式 connection 全部关闭或归还；
- success、error、cancel、panic 均以 `defer` 恰好 release；
- 候选扫描、每次租约取得、每次续约、每个 settlement job 均有明确边界；
- 串行多事务的同一顶层动作全程持有同一 permit，禁止事务间释放重取和并发占用多个 connection；
- worker 传递“permit 已持有”责任，helper 禁止嵌套 acquire；direct 顶层调用自行恰好 acquire 一次；
- server worker/operator 共用 owner；未来互斥 maintenance 进程使用自己的 pool/owner，不共享进程内 semaphore。

接口和验证映射在 `design.md:861,2880-2881,3316,3586-3591` 与上述合同一致。未发现 owner、单位、同进程/未来进程关系或 acquire/release 路径自相矛盾；实际生产数值继续为 TBD，未被硬编码。

### `IDENTITY-DESIGN-R43-S10.4-R01-F003` — MEDIUM — **CLOSED**

正文、配置表和测试矩阵已形成唯一时序：

- `design.md:843`：五项任一 missing、blank、parse、目标类型 overflow、zero、negative 或其他内在范围失败，DSN parse 及后续调用全部为 0；
- `design.md:846`：只有五项内在校验成功后才 parse DSN，并在 connect 前校验 budget `<= pgxpool.Config.MaxConns`；
- `design.md:3315`：仅“已解析正数 budget `> MaxConns`”允许 parse=1，connect 及以后=0；
- `design.md:3369,3403,3575,3591,3738,3821` 与该时序一致。

现役 `parseDatabasePoolConfig` 返回 `*pgxpool.Config`（`database_config.go:53-73`），`MaxConns` 在 connect 前可读；交叉校验可由八文件中的 `main.go` 拥有，无需修改 `database_config.go` 或增加第九文件。

## 4. 原待决项复审

### Article / PasswordReset Start 语义 — **CLOSED**

`design.md:848,2882,3317,3586,3592-3593` 已明确：

- 两个 Reconcile 可同步失败并阻止两个 Start 和 Serve；
- Article 与 PasswordReset 的 `Start(context.Context)` 均无同步 error；
- Start 后运行错误只经 `Done()/Err()` 上报；
- lifecycle fake 不得伪造 Article production Start 同步失败。

真实 `lifecycleWorker` 为 `Start(context.Context)`（`lifecycle.go:20-27`）；真实 Article worker 同样无同步返回值，运行结果经 `Done/Err`（`internal/article/deletion_worker.go:233-243,276-305`）。现有 `lifecycle.go/_test.go` 可扩展为双 worker coordinator，不需要修改 Article worker 或增加第九文件。

## 5. 新增 findings、待决定与非阻断事项

- 新增 findings：**无**。
- 待决定：本复审范围内无阻断决定。
- 实际生产 lease、backoff、自动重试上限、批量领取上限、connection budget 数值继续为 `TBD（待容量验证阶段确定）`；本次不要求也不允许编造数值。

## 6. Gate 与交接条件

### ADR consistency Gate

**PASS**。r44 保持 ADR-0023 的 durable operation、租约 worker、三消费者 fail-closed Gate、dead-letter 不解锁和 worker-before-pool 方向；maintenance mode 延后未改变 ADR 核心。

### Design Review Gate

**PASS**。原 F001、F002、F003 和 Article Start 待决项均已关闭，无新增 finding。

PASS 后仍须按 development-delivery 顺序交接：

1. Test-Engineer 将 test-plan 的 source revision/hash 更新到 r44，并补齐五项配置、permit 与双 worker lifecycle 验证；
2. Task-Planner 将 S10.4 从旧六文件同步为 r44 精确八文件，保持 `[ ]`、todo/not started，且不得重开 S10.3 Gate construction ownership；
3. Task-Planner 另行登记 `identity-auth-S10.5`，在 S10.4 后顺序接管 main 两文件；其设计完成前不得声明维护命令或 runbook 可执行；
4. S10.1 继续 implementation stopped、`IDENTITY-S10.1-REV-R01-F002` OPEN；B6D/S11.1 不启动。

Tasks/Plan 当前仍登记旧六文件和受控命令，是待同步的下游旧状态；按用户固定流程，它不构成 r44 Design Review 的重复 finding。

## 7. 未验证风险与限制

1. 本轮是非代码设计复审，未运行 Go 测试、构建、race、vet 或 Manifest 检查，不伪造代码验证结果。
2. 未启动服务，未运行真实 PostgreSQL、SMTP/TLS、部署或 E2E。
3. 未读取、输出或使用任何 DSN、credential 或 secret。
4. 四个 `password_reset_worker*` / `password_reset_operator*` 文件尚不存在；尚未验证真实 SQL、租约竞争、事务锁序、panic/cancel 释放或多连接容量。
5. 未验证生产容量值、真实 pool 竞争、maintenance mode 合同、授权系统、审计证据来源或 runbook 演练。
6. 本 PASS 只适用于 r44 固定 SHA。若实现需要第九个 S10.4 production 文件、改变 ADR-0023/architecture/boundary/公共契约，或对象发生变化，受影响范围必须停止并重新评审。

## 8. 实际验证

- `Get-FileHash -Algorithm SHA256`：核对设计开始/结束哈希和原报告哈希；
- `git branch --show-current`、`git rev-parse HEAD`；
- 只读 PowerShell/`rg`：核对设计修订、真实消费者、上游依据和 Tasks/Plan 状态；
- `git diff --check -- .spec/wip/identity-auth/review-design-r44-s10.4-r02.md`；
- 对未跟踪报告另执行 `git diff --no-index --check -- /dev/null .spec/wip/identity-auth/review-design-r44-s10.4-r02.md`。

未执行 commit、push、切分支、清理、格式化或对其他文件的写入。
