# Article Management Design r10 Production Wiring 独立评审

- **Reviewer**：Faraday（独立非作者）
- **评审日期**：2026-10-23
- **objectType**：design
- **工作区 / 分支 / HEAD**：`D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- **原 finding**：`.spec/wip/review-b6a-g4-r01.md` / `B6A-G4-R01-F001`
- **结论**：**Design-side Gate: PASS**
- **finding 结论**：`B6A-G4-R01-F001` 的设计侧缺口已充分补齐；该 planning finding **不由本报告直接关闭**

本结论只绑定下列两个固定、只读对象，不表示 test-plan、Task、实现、真实环境验证或 B6A batch Gate 已完成：

| 对象 | revision | git blob | SHA-256 |
|---|---|---|---|
| `.add/project/knowledge/design-workspace/article-management/design.md` | `article-management-design-ir01-r10` | `16eb428f392f3ddc8ba9be485e5afd49b28409d4` | `6785977683CF0F59E277667425F7FEE9714EBBD436FBDFE046E607F2E77CA67E` |
| `.add/project/knowledge/design-workspace/article-management/stories.md` | `article-management-stories-ir01-r07`；`source_design_revision=article-management-design-ir01-r10` | `b90e27807c06f035ffe05bc239894142d04936e5` | `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05` |

开始核对时两对象均与指定 revision/blob/SHA-256 一致，报告写入前路径不存在；落报告后再次核对仍无漂移。

## 1. 两轮实际评审范围

### Round 1：固定设计对象直接合同

逐项检查：

1. Article production 配置键的来源、required/default、格式、范围、交叉约束、DB/listener 前 fail-closed 及错误去敏。
2. Journal root 的绝对路径、owner-only、symlink/platform preflight、排他 writer lock、current/retained keyring、固定 key ID、namespace/HMAC domain 隔离和 secret clear。
3. `pending_journal` 自动补写 tombstone、operation discovery 稳定排序、claim/lease、即时首轮、ticker+wake、batch、full-jitter 退避及 terminal/transient/no-op 分类。
4. 唯一 production router 中三条 exact DELETE、Trace/Origin/Session/CSRF/Principal/授权顺序、JSON 404/405/`Allow` 和 Identity route 共存。
5. worker、双 HTTP server、listeners、Journal/keyring、共享 DB pool 的单一 owner、运行失败触发、shutdown drain/shared deadline、reverse unwind 和关闭顺序。
6. stories r07 是否只形成一个可拆的 production wiring 后继切片，且不创建 task 编号、不声明实施或完成。
7. design §6.1 是否为后续 test-plan 同步提供配置、Journal、router、worker/lifecycle 和失败注入的明确验收维度。

### Round 2：真实消费者与上游接缝

有界核对：

- accepted architecture r24、article boundary r13；
- ADR-0008、ADR-0020、ADR-0021、ADR-0022；
- 现役 `cmd/han-server` composition、router、双 server lifecycle 与配置解析；
- `internal/article` 的 DeletionOperation、DeletionCoordinator、DeletionJournal、DeletionFinalizer、Repository、Handler 及直接测试；
- `migrations/0003_article.sql` 的 operation 状态、claim/lease 字段与 `started_at,id` 索引；
- 现役 article test-plan r04、Tasks、B6A Plan 和原 batch review；
- 共享 trace/error/config/lifecycle 的直接接口。

未扩展为全仓审查，也未把尚未实现的 production wiring 误判为本次设计缺陷。

## 2. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

无。

### LOW

无。

## 3. 原 finding 的设计侧判定

### `B6A-G4-R01-F001` — DESIGN-SIDE SUFFICIENT

原问题是 Article 永久删除核心虽已有 task-level 实现与评审证据，但 production composition/router/config/finalizer worker 没有明确的后继 owner，不能由两张 task PASS 直接推导 B6A batch PASS。

r10/r07 已提供足以继续规划的单义合同：

1. `design.md:753-772` 集中定义 Article 删除配置，区分必填、显式空 retained、唯一允许默认的 file mode、范围和交叉约束，并要求可前置检查在 DB parse/connect/ping/migrate 前完成；错误不得泄露 DSN、绝对路径、retained version、HMAC、正文或 PII。
2. `design.md:775-801` 固定 Article 专属目录、namespace、`article-deletion-hmac` key ID、HMAC domain、writer lock、current 写入与 current+retained 验证；与 Recovery 只允许复用 loader/file-safety 原语，不共享目录、锁、key material、记录链或 domain。
3. `design.md:1816-1824` 固定 PostgreSQL discovery 的状态集合与 `started_at,id` 稳定顺序，明确 `pending_journal` 从不可变 operation 事实幂等补写 tombstone，以及即时调谐、ticker+wake、单并发、batch、预算、lease、退避和错误分类。
4. `design.md:1757-1785` 与 `design.md:2087` 固定唯一 `cmd/han-server` 的三条 DELETE 路由、精确 method/path、安全中间件顺序、JSON 404/405/`Allow: DELETE`，并要求既有 Identity routes 与共享有状态实例不回归。
5. `design.md:1828-1837` 固定长期资源 ownership、worker 与两台 HTTP server 的 one-shot lifecycle、shared deadline、drain、reverse unwind、首个根因保留和 DB pool 最后关闭。
6. `design.md:2079-2102` 给出可由 Test-Engineer 转写的 production config、Journal/keyring/domain、router、worker、cancel/shutdown 和逐步构造失败测试矩阵，并明确现役 test-plan r04 尚未同步。
7. `stories.md:215-234` 只定义一个“Article production composition、router、config 与 finalizer worker”后继切片；其依赖和验收与 design r10 一致，不含 task 编号，不声称已实施、已测试或已完成。

真实接缝核对未发现必须新建第二 composition root、改变公共 REST/公共错误、改变已接受架构或新增 ADR 才能实施的矛盾。现有 operation schema、Journal/Finalizer 接口和 `cmd/han-server` lifecycle 可由最小后继任务扩展承接。

因此，本报告确认 **设计侧已足以**：

1. 由 Test-Engineer 同步 article test-plan；
2. 再由 Task-Planner 创建一个且仅一个最小 `article-management-S4.6`，承接 production composition/router/config/finalizer worker owner。

本报告不直接关闭 `B6A-G4-R01-F001`；只有 test-plan 同步、唯一后继 task 登记及相应 planning review 完成后，才能由有权角色判断该 planning finding 是否关闭。

## 4. 重点合同核对结果

| 评审维度 | 结果 |
|---|---|
| 配置与 fail-closed | 完整；配置集中、来源明确、范围及交叉约束可实现，可前置项在 DB/listener 副作用前拒绝，错误与日志禁止泄密 |
| Journal/keyring | 完整；root 安全、排他锁、current/retained、固定 key ID、namespace/domain 隔离、关闭清零与损坏拒绝均单义 |
| 自动收敛 worker | 完整；`pending_journal`、`journaled`、过期 `finalizing` 均有稳定 discovery/converge 路径；即时首轮、ticker+wake、batch/backoff 和错误分类可实现 |
| Router 与鉴权 | 完整；三条 exact DELETE、middleware 顺序、404/405/Allow、Identity route 共存均已固定 |
| Lifecycle/ownership | 完整；单一 owner、worker failure、共享 deadline、drain、reverse unwind、DB pool 最后关闭无相互矛盾 |
| Stories 唯一性 | 通过；仅一个后继 production wiring 切片，无 task ID 和实施声明 |
| Architecture/boundary/ADR | 未发现回退或越界；仍使用唯一 Go process、共享 DB pool、环境变量配置、PG18 migration 与 ADR-0022 lifecycle |
| Test-plan 接力 | 设计已给足维度；现役 r04 尚未同步，因此只能通过设计侧 Gate |

## 5. 待决定项与非阻断建议

### 待决定项

无新的设计正确性待决定项。

容量、吞吐、告警阈值和运行留存周期仍属于后续运行/性能输入；r10 已固定可实施的调度与生命周期基线，因此不阻塞本次设计侧 Gate，也不得被解释为容量或生产就绪承诺。

### 非阻断建议

Test-Engineer 同步 test-plan 时，建议把以下负例写成显式 fixture，而非只保留概括性描述：

- current/retained version token、重复/空项和动态 secret 键；
- file mode 非四位八进制、owner 权限不足及 symlink；
- retained 显式空与环境变量缺失的差异；
- 错误、日志和 lifecycle failure 中 secret/路径/PII 去敏。

该建议不改变 r10 合同，也不阻塞 PASS。

## 6. Design-side Gate 与后继条件

### Design-side Gate: PASS

通过理由：

- 没有未关闭的设计阻断 finding；
- r10/r07 对原 finding 的设计侧缺口给出了完整、无明显矛盾且可由现役接口最小扩展实施的合同；
- 后继范围唯一，不需要本轮替 Lead 改变 architecture、boundary、ADR 或公共契约。

下一交接条件：

1. Test-Engineer 依据 r10/r07 同步现役 test-plan，并经独立评审；
2. Task-Planner 只创建一个最小 `article-management-S4.6`，精确拥有 production composition/router/config/finalizer worker，不把 article production DELETE 写成已可用；
3. 后续实现、测试与代码评审独立完成后，再回到 B6A G4-batch finding 链判断关闭。

## 7. 实际命令与结果

| 命令/检查 | 结果 |
|---|---|
| `git branch --show-current` | `main` |
| `git rev-parse HEAD` | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `git hash-object -- <design/stories>` | 分别与固定 blob 一致 |
| `Get-FileHash -Algorithm SHA256 <design/stories>` | 分别与固定 SHA-256 一致 |
| revision/source revision 定点读取 | `design r10`、`stories r07`、`source_design_revision=r10` |
| 写入前 `Test-Path <report>` | `False` |
| `Get-Content`、`rg -n` 定点读取规则、设计、stories、原 finding、上游与直接消费者 | 完成 Round 1 / Round 2 |
| `git diff --cached --name-only` | 空；本轮未 stage |
| 落报告后限定 `git diff --check -- <design> <stories> <report>` | PASS |
| 落报告后再次核对固定对象 revision/blob/SHA-256 | 一致，无漂移 |

本轮是 design review，未运行 Go build/test/race/vet，也不把已有 task-level 或 Lead 证据冒充本轮执行结果。

## 8. 实际读取文件

### Role、Skill、Rule 与 Profile

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/decisions-guide.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.codex/guidance/observability-standards.md`
- `.codex/guidance/tracing-standards.md`
- `.codex/profiles/go-service/README.md`
- `.codex/profiles/go-service/rules/coding-standards.md`
- `.codex/profiles/go-service/rules/no-hardcode-policy.md`
- `.codex/profiles/go-service/rules/observability-standards.md`
- `.codex/profiles/go-service/rules/security-coding-standards.md`
- `.codex/profiles/go-service/rules/tracing-standards.md`

### 设计、规划与上游依据

- `.add/project/knowledge/design-workspace/article-management/design.md`
- `.add/project/knowledge/design-workspace/article-management/stories.md`
- `.add/project/knowledge/design-workspace/article-management/test-plan.md`
- `.spec/wip/review-b6a-g4-r01.md`
- `.add/project/knowledge/tasks/tasks.md`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`
- `.add/project/knowledge/design/architecture.md`
- `.add/project/knowledge/design-workspace/article-management/boundary.md`
- `.add/project/knowledge/decisions/framework.md`
- `.add/project/knowledge/decisions/content.md`

### 真实消费者与 schema 接缝

- `cmd/han-server/main.go`
- `cmd/han-server/main_test.go`
- `cmd/han-server/lifecycle.go`
- `cmd/han-server/lifecycle_test.go`
- `internal/article/deletion_coordinator.go`
- `internal/article/deletion_finalizer.go`
- `internal/article/deletion_journal.go`
- `internal/article/repository.go`
- `internal/article/model.go`
- `internal/article/handler.go`
- 上述 Article 删除路径的直接测试文件
- `migrations/0003_article.sql`
- `internal/platform/config.go`
- `internal/platform/errors.go`
- `internal/platform/http.go`
- `internal/platform/http_security.go`

## 9. 未验证风险与限制

1. 本轮未修改或执行 test-plan；production config/router/worker 测试矩阵仍须后续同步并评审。
2. 未实现或运行 Article production composition、三条 DELETE route、Journal provider 或 finalizer worker。
3. 未连接真实 PostgreSQL，未验证真实事务/锁/commit outcome。
4. 未在受支持 Unix 目标持久卷验证 owner-only、writer lock、file/directory fsync、崩溃、断电或磁盘故障；Windows 结论不外推为目标卷保障。
5. 未启动 Host，未做部署、浏览器、E2E、容量、吞吐、告警或 production readiness 验证。
6. 未读取、打印或持久化 `HAN_TEST_POSTGRES_DSN` 或其他 secret。
7. 本 PASS 只表示固定 r10/r07 的设计侧可交接性，不关闭原 planning finding，不代表实现、测试或 B6A G4-batch PASS。

## 10. 复审条件

以下任一情况需要重新评审受影响范围：

- r10/r07、accepted architecture/boundary/ADR 或公共 REST/错误合同发生变化；
- Test-Engineer 无法从当前合同唯一形成 test-plan；
- Task-Planner 需要超过一个 production wiring task、第二 composition root、MQ/外部 scheduler 或新的架构决定；
- 实施发现现有 operation schema、Journal/Finalizer、router 或 lifecycle 接口无法按最小扩展承接；
- 安全检查需要降低 root/keyring/domain、鉴权顺序、fail-closed 或 secret 去敏要求。

本独立结论不替代 Lead/用户对设计采用、规划 finding 关闭、任务创建、实施授权或风险接受的决定。
