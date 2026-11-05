# identity-auth design r43 / S10.4 独立设计评审 R01

- 日期：2026-10-24
- Reviewer：非原作者独立 Reviewer
- objectType：design
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / 基线 HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- revision：`identity-auth-design-ir01-r43`
- 评审范围：仅 S10.4 production worker/config/lifecycle 增量
- 结论：**CHANGES REQUIRED**

本报告不重审无新证据的历史已关闭决定，不修改被审设计、代码、Tasks/Plan、ADR、架构或边界，也不代替 Lead 的风险接受、实施授权或后续真实验证。

## 1. 固定对象与完整性

| 对象 | 开始 SHA256 | 结束 SHA256 | 结果 |
|---|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `91F89502C4D923AE7C69EBC6B7D0BDBD1328C508E8CE17F58536CD48AF315AF7` | `91F89502C4D923AE7C69EBC6B7D0BDBD1328C508E8CE17F58536CD48AF315AF7` | MATCH；评审期间对象未变化 |

工作树在评审开始前已有大量他人未提交改动；本 Reviewer 未回滚、覆盖、格式化或修改这些内容，唯一写入为本报告。

## 2. 两轮实际检查范围

### Round 1：r43 直接设计审查

1. 读取 Reviewer Role、`add-code-review` design 分支、`code-review-checklist`、`abstraction-principles`、`design-doc-writing-style`、`no-hardcode-policy`。
2. 读取 go-service profile 中与本次设计实际相关的 Go 接口/并发/资源释放、显式配置、审计和可观测性规则。
3. 定点检查 r43 的：
   - 五项 production runtime config、fail-fast 时序及 `MaxConns` 交叉校验；
   - 唯一 Store/Gate、operator/worker composition 与八文件写集；
   - 双 worker Reconcile/Start/Stop/Wait、运行错误优先级、shared deadline、Close fallback 与 pool 最后关闭；
   - 受控 retry/settle/confirm 命令、审计和 runbook 合同；
   - 测试矩阵、S10.3→S10.4 所有权与状态声明。
4. 未把“设计声明自己完整”当作证据；每项可实施性均继续追踪真实消费者或上游依据。

### Round 2：真实消费者与上游依据有界审查

读取并核对：

- r42 Design Review R02 PASS；
- `.add/project/knowledge/design/architecture.md`；
- `.add/project/knowledge/design/identity-auth/boundary.md`；
- decisions 索引定位到的 accepted ADR-0023：`.add/project/knowledge/decisions/identity.md:275-331`；
- Tasks/Plan 的 S10.3、S10.4、B6C/B6D 状态与写集；
- `cmd/han-server/main.go`、`main_test.go`、`lifecycle.go`、`lifecycle_test.go`；
- `cmd/han-server/database_config.go`、`database_config_test.go`；
- `internal/identity/password_reset_store.go`、`password_reset_store_test.go`；
- `internal/article/deletion_worker.go`；
- 全仓定向搜索现存 `password_reset_worker*`、`password_reset_operator*`、CLI/维护命令入口及直接 composition/lifecycle consumers。

真实代码确认：

- S10.3 production root 在 `main.go:1996-2007` 只构造一个 `*identity.PasswordResetStore`，并复用给 Session/Login/PasswordChange。
- 当前 `businessRuntime` 和 lifecycle 只承载 Article worker（`main.go:257-260,457-494,529-542,2308-2325`；`lifecycle.go:20-40,53-193`）。
- Article worker 的真实签名是 `Start(context.Context)`，无同步错误返回（`internal/article/deletion_worker.go:233-243`）。
- `pgxpool.Config.MaxConns` 在现有 DSN parse 后即可取得，r43 所述交叉校验可在 connect 前由 `main.go` 完成，无需修改 `database_config.go`。
- 四个新增 `password_reset_worker*` / `password_reset_operator*` 文件当前均不存在；仓库也没有现成 CLI、管理 API 或维护命令框架。

## 3. Findings

严重度统计：CRITICAL 0 / HIGH 1 / MEDIUM 2 / LOW 0。

### `IDENTITY-DESIGN-R43-S10.4-R01-F001` — HIGH

**受控命令被列为 S10.4 production 交付能力，但没有真实入口、授权与运行隔离合同。**

#### 位置与证据

- `design.md:2240-2246`：只有“受控运行角色”可执行三个审计命令，并要求 operator、审批/理由、evidence ref、before/after 与 AuditEvent。
- `design.md:2255`：定义 `PasswordResetOperatorCommand.Execute(command,evidence)`。
- `design.md:2879`：`PasswordResetOperator` 的消费者包含“受控运行入口”。
- `design.md:3378`：runbook 必须列出 `retry`、`settle-failed-or-invalidated`、`confirm-succeeded`。
- `design.md:3570,3587-3590`：S10.4 声称交付受控命令，但八文件只提供 operator/worker/main/lifecycle 及测试。
- 真实 `cmd/han-server/main.go:142-148` 无参数或子命令分派，固定启动 server；全仓未找到其他真实维护命令入口。

#### 违反依据

- accepted ADR-0023 要求 worker、运行手册和受控恢复共同演进，且失败必须持续保持 Gate。
- 设计必须给实现者可执行的消费者、权限、输入、错误和生命周期契约；不能把安全敏感入口留给实施阶段自行发明。
- 八文件之外若需要其他 production seam，r43 自身要求停止，而不是隐藏扩写。

#### 触发场景

实施者只能在以下不受约束的选择中自行决定：

1. 只实现内部 operator，三个命令没有 production 消费者，dead-letter 无法按设计恢复；
2. 在 `main.go` 临时发明 CLI/维护模式，但没有固定命令输入、角色证明、审批/evidence 来源、数据库启动顺序、是否与 server/worker 互斥、listener 是否必须为零、退出码与失败清理；
3. 新增管理 REST、独立工具或第九个 production 文件，直接越过当前写集、公共契约或架构边界。

#### 影响

该缺口位于可解除认证 Gate/hold 的高风险运维路径。错误实现可能形成无生产入口、未经授权的恢复入口、与在线 worker 并发的第二运行 authority，或迫使实施者越界新增公共接口/文件。

#### 修复建议

在不改变 accepted ADR-0023 的前提下二选一并写清：

1. **S10.4 只交付内部 operator port**：明确三个真实受控入口由哪个后续任务和精确生产文件拥有，S10.4 不宣称命令 production 可执行；同时保留 Gate，后续入口完成前不得宣称 dead-letter 可运维恢复。
2. **S10.4 在八文件内交付入口**：固定具体入口形态、命令和输入 schema、运行角色/审批/evidence 的可信来源、审计主体、与 server/worker 的互斥或并发规则、配置/DSN/migration 顺序、listener/Serve 必须为零的条件、退出码、错误净化、资源清理及相应测试。

若真实方案需要第九个 production 文件、新公共 REST/错误、独立服务或改变架构/ADR，应按停止条件返回上游决策和重新规划，不得在本卡内自行扩张。

#### 复审条件

- 提供唯一真实消费者及其完整调用/授权/失败合同；
- 证明入口可在确认后的精确写集内实现；
- 测试映射覆盖未授权、证据不足、并发状态变化、审计失败、零 listener/Serve（若为维护模式）及稳定非零退出。

### `IDENTITY-DESIGN-R43-S10.4-R01-F002` — MEDIUM

**connection budget 只有数值交叉校验，未固定 permit 的单位、覆盖路径和持有周期。**

#### 位置与证据

- `design.md:858`：定义为“worker 内部并发数据库操作 permit 上限”，worker/operator 共用同一预算 owner。
- `design.md:2879-2880`：operator 与 worker 消费同一 runtime connection-budget owner。
- `design.md:3585-3588`：worker/operator 均可能执行数据库事务，测试只笼统要求 connection permits。
- `design.md:840,843`：只规定 `budget <= pgxpool.Config.MaxConns` 的数值关系。

#### 违反依据

资源预算必须明确 owner、单位、取得/释放边界和所有消费者；否则数值校验不能证明真实资源上限。仅比较两个整数不等价于限制实际 pool connection 并发。

#### 触发场景

- 候选扫描不取 permit，但每个 job 事务取 permit；
- permit 只包住函数调用的一部分，在事务/connection 尚未释放时提前归还；
- worker 计入预算而受控 operator 命令不计入；
- 扫描、租约取得和 settlement 嵌套 acquire，导致死锁或实际连接数超过预算；
- 受控入口若是另一进程，“同一 owner”无法按当前文字解释。

#### 影响

实现和测试可以形式上通过 `budget <= MaxConns`，但运行时仍可能超出预算、互相饥饿或死锁；也无法让 Backend-Dev/Test-Engineer 对“一个 permit”形成一致断言。

#### 修复建议

固定单一预算 owner 及以下语义：

- 一个 permit 精确代表什么（建议直接绑定一个并发 pool connection/transaction 占用槽）；
- 候选扫描、租约、每个 settlement job、受控命令分别是否必须取得；
- permit 从何时取得、持有到 connection/transaction 何时释放；
- error、取消和 panic 路径如何保证归还；
- 是否允许嵌套取得，若不允许由哪个层唯一取得；
- operator 与 worker 同进程/异进程时预算关系如何解释和验证。

实际生产数值继续保持 `TBD（待容量验证阶段确定）`，无需也不得在本修订编造容量值。

#### 复审条件

设计和测试矩阵能对每条数据库路径明确断言 acquire/release 次数、最大并发、取消/失败归还，以及与 `MaxConns` 关系。

### `IDENTITY-DESIGN-R43-S10.4-R01-F003` — MEDIUM

**第五项 connection-budget 的内在失败缺少 pre-DSN 测试时序，和正文合同不一致。**

#### 位置与证据

- `design.md:840`：五项的缺失、格式和内在范围错误都必须使 DSN parse 及后续调用数为 0。
- `design.md:2862`：五项缺失、解析/内在范围失败均在 DSN parse 前失败；只有与 `MaxConns` 的交叉失败发生在 parse 后、connect 前。
- `design.md:3314`：测试矩阵列出五项负例，却只要求“前四项任一失败”使 DSN parse 为 0，connection budget 只明确了 `> MaxConns` 的交叉负例。

#### 违反依据

设计内部验收依据必须一致且可追踪；测试合同不能遗漏正文已要求的安全启动顺序。

#### 触发场景

`HAN_PASSWORD_RESET_WORKER_CONNECTION_BUDGET` 缺失、空白、解析失败、目标类型溢出、零值或负值时，实施/测试可能先 parse DSN，再报告配置错误。

#### 影响

Backend-Dev 与 Test-Engineer可能把第五项所有失败都误归为 post-DSN 校验，破坏“内在配置错误零数据库配置副作用”的统一 fail-fast 合同。

#### 修复建议

把 `design.md:3314` 改为：

- 五项的缺失、空白、解析失败、溢出、零/负数等内在失败均断言 DSN parse/connect/Ping/migrate/prepare/factory/listener/Serve/Start 为 0；
- 只有 connection budget 已成功解析且为正、但 `> pgxpool.Config.MaxConns` 的交叉负例允许 DSN parse=1，connect 及后续为 0。

#### 复审条件

正文、配置表、测试矩阵对第五项的 pre-DSN 与 post-DSN 两类失败给出唯一一致的调用计数。

## 4. 已检查且未形成 finding 的重点

1. **八文件普通 production seam**：现役 `main.go`、`main_test.go`、`lifecycle.go`、`lifecycle_test.go` 加四个新增 identity 文件足以承接常规 config/operator/worker/runtime wiring；未发现普通接线必然需要第九文件。F001 的真实受控入口仍是例外风险。
2. **DSN/MaxConns 时序**：现有 `parseDatabasePoolConfig` 返回 `*pgxpool.Config`，可由 `main.go` 在 connect 前读取 `MaxConns`；无需扩写 `database_config.go`。
3. **同一 Store/Gate**：现役 builder 已持有唯一 `passwordResetGate`，可继续注入 operator/worker；八文件设计禁止第二 Store/Gate/root、隐藏全局和 lifecycle construction，方向正确。
4. **双 worker shutdown 主线**：r43 已覆盖 server/worker error、signal、Stop 各一次、shared deadline、双 Close fallback、worker-before-pool 与稳定错误优先级；未发现与现役 Article cleanup 顺序不可调和的证据。
5. **范围控制**：未发现 S10.4 需要 schema/migration、新依赖、公共 REST/错误码或重开 ADR-0023 的已证实需求。
6. **状态与所有权**：S10.3 task-level CLOSED、r42 R02 PASS、S10.4 `[ ]`/todo/not started、S10.1 stopped/F002 OPEN、B6D/S11.1 未启动均一致。Tasks/Plan 仍登记旧六文件，但 r43 已明确要求后续同步到八文件；这是待同步状态，不把它重复报为本设计缺陷。

## 5. 待决定、非阻断建议与方向边界

### 待决定

**Article worker Start 失败语义需在实施前澄清。**

r43 要求测试覆盖每个 Reconcile/Start 失败点，但真实 Article worker 是 `Start(context.Context)`、无错误返回。当前八文件仍可在 `lifecycle.go` 用 adapter 把 Article Start 表达为永远返回 nil，并由 lifecycle fake 只验证 coordinator 的 Start-error 分支；因此尚无证据证明必须修改第九文件。

需要明确：

- 若只要求验证 coordinator 能处理“可失败 worker Start”，应说明 Article production Start 本身无同步失败，运行错误由 `Done/Err` 上报；
- 若要求 Article production Start 能同步失败并阻止 Serve，则需要修改 `internal/article/deletion_worker.go` 或其他 production seam，触发第九文件停止条件和重新规划。

### 非阻断建议

1. Task-Planner 后续同步八文件时，应保留 `main.go/_test.go` 仅有 worker/config/runtime wiring 的顺序写权限，不得重新记账 S10.3 Gate construction。
2. 后续代码评审应核对 interface typed-nil、operator/worker 与三消费者的真实指针身份、permit 释放和 startup unwind；仅字段名相同不能证明同一 authority。
3. connection budget 允许等于共享池全部 `MaxConns` 可能挤压 HTTP/Article，但实际生产值和保留余量属于容量验证决定；在没有权威容量证据时不应由本设计编造阈值。

## 6. Gate 结论

### ADR consistency Gate

**PASS（未重开 ADR）**。r43 的 durable operation、租约 worker、三消费者 fail-closed Gate、同一进程和 worker-before-pool 方向与 accepted ADR-0023 一致。F001 要求补足 ADR 方向内的真实运维入口合同；若修复选择新公共接口、独立服务或新 production seam，则必须另行升级上游决定。

### Design Review Gate

**CHANGES REQUIRED**。

F001 使受控 dead-letter 恢复缺少可实施且可审计的真实入口；F002 使连接预算无法证明实际连接并发边界；F003 使第五项配置的 fail-fast 测试合同自相矛盾。因此本次不能对 S10.4 增量给 PASS，也不应据 r43 启动实现。

## 7. 复审条件

复审应固定修订后的 design SHA，并逐项对应：

1. F001：真实受控入口、权限/evidence/audit、运行隔离、失败与文件所有权闭环；
2. F002：permit owner、单位、覆盖路径、持有/释放和并发断言闭环；
3. F003：五项内在失败 pre-DSN 与 budget/MaxConns 交叉失败 post-DSN 的唯一测试矩阵；
4. Article Start 语义已明确，若需要第九 production 文件则先完成上游重规划；
5. Tasks/Plan 的 S10.4 八文件与顺序所有权完成同步，但仍保持 todo/not started，直到新的授权与 Gate。

## 8. 未验证风险与限制

1. 本轮是设计评审，未运行 Go 测试、构建、vet、race 或 Manifest 项目检查；不伪造代码验证结果。
2. 未启动服务，未运行真实 PostgreSQL、SMTP/TLS、部署或 E2E。
3. 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN` 或任何 credential/secret。
4. 四个 worker/operator 文件尚不存在，因此未验证其真实 API、SQL、租约竞争、事务锁序、取消、panic 或连接占用。
5. 未验证真实容量值、pool 争用、告警阈值和生产 runbook 演练；这些值按合同继续为 TBD。
6. 结论只适用于表列 r43 固定哈希。对象变化、需要第九 production 文件、改变 accepted ADR-0023/架构/公共契约时，受影响范围必须停止并重新评审。

## 9. 实际验证

本轮执行了只读 Git/PowerShell 文件阅读、定向搜索、哈希和状态检查，以及：

- `Get-FileHash -Algorithm SHA256`：固定设计开始/结束校验；
- `git branch --show-current`、`git rev-parse HEAD`、`git status --short`；
- `rg -n --encoding utf-8`：受控入口、operator/worker、composition、lifecycle、Tasks/Plan 和真实消费者；
- `git diff --check -- .spec/wip/identity-auth/review-design-r43-s10.4-r01.md`。

未执行 commit、push、切分支、清理、服务启动、真实数据库、SMTP、部署或 E2E。
