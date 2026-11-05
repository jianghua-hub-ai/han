# article-management design r13 / S4.2 Code Review R01 独立设计评审

- Reviewer：独立非作者 Reviewer
- 被审作者：Feature-Designer Kierkegaard `01a171dc-413f-7760-bc52-145e00b29224`
- 评审日期：2026-10-25
- objectType：design
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定对象：`.add/project/knowledge/design-workspace/article-management/design.md`
- revision：`article-management-design-ir01-r13`
- 固定 SHA256：`C90222F21923BE55B78DBF70CF339DDF75A53673C2C4DF1F6312C6F12F0D78D7`
- 原代码评审：`.spec/wip/article-management/review-code-s4.2-r01.md`
- 原代码评审 SHA256：`777CF2B5029049F86E8B2484DC69C28C11A068A9252F65AB17E8EC97647165C1`
- **Design Review Gate：CHANGES REQUIRED**

## 1. 固定对象与停止条件

评审开始时固定对象 SHA256 匹配；分支、HEAD 匹配；staged 为空。评审期间未修改 design、test-plan、stories、Tasks、Plan、代码、migration、architecture、ADR 或 boundary。

Round 2 确认：r13 没有给出可进入 production composition 的 concrete `BackupInventoryReader`，且明确把新增基础设施 provider / 跨服务协议的选择退回 architecture/ADR。该方向缺口触发用户规定的停止条件“判断必须修改 architecture/ADR/boundary”；本 Reviewer 不替 Lead 选择 provider 或修改上游，只完成已取得证据范围的结论并落本独立报告。

## 2. 两轮实际检查范围

### Round 1：固定 design 的直接合同

逐项检查 r13 对 `ARTICLE-S4.2-REV-R01-F001`～`F005` 的：

- production source 所有权、依赖、唯一 composition 与 fail-closed；
- `attempt_generation + attempt_token`、Begin/Ready/Fail/ReadCurrent、CAS、崩溃与 commit outcome unknown；
- UTC 微秒、同 input 的 ref/digest/time 复用；
- `internal/recovery` 唯一 Go authority 与依赖方向；
- A-REC-04～07 真实 PostgreSQL 多连接/多事务验证要求；
- migration、17 文件建议写集、S4.2/S4.4/S4.5 分工和实现停止条件。

### Round 2：上游依据、真实消费者与现状

有界核对：

- architecture r27 中恢复顺序、ADR-0020 authority、生产 Gate 与测试要求；
- article boundary r13、identity boundary r13 的 S16/DTO-only 消费合同；
- ADR-0020 与 ADR-0021；
- `internal/recovery/replay_status_v1.go`、`internal/recovery/coordinator.go`；
- 当前 Article replay provider/store/status、Journal/finalizer/repository；
- `migrations/0003_article.sql` 与当前仅有 `0001`～`0008` 的迁移现状；
- Tasks/Plan 的 S4.2、S4.4、S4.5、S4.6 所有权、写集与暂停状态；
- 全仓 production source/provider/composition 消费者检索。

未把并行修订中的 test-plan r10 作为固定依据；未运行 PostgreSQL、外部网络、代码测试、部署或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。

## 3. Findings（按严重度排序）

### `ARTICLE-DES-R13-S42-R01-F001` — HIGH — 必填 inventory 依赖没有 concrete provider、装配输入或交付 owner，F001 仍无可运行的 production 正向路径

- **位置**：design `543-557`、`841-847`、`2182-2188`、`2291-2293`、`2323`、`2385`、`2400`
- **违反依据**：
  - 原 `ARTICLE-S4.2-REV-R01-F001` 要求真实 source 与 production 装配，而非只有端口或 stub；
  - design 自身声明 r13 将 F001～F005 转成“可实施合同”；
  - ADR-0020 / boundary S16 要求 inventory 属于当前 input，并在旧备份开放前形成可验证 Ready。
- **证据**：
  - r13 把 `BackupInventoryReader` 定义为 production source 的必填依赖，但明确 concrete provider、数据格式和装配输入均为 TBD；
  - S4.2 被要求完成 production source/wiring，却禁止实现未批准 provider；
  - S4.5 被明确限制为合成 fixture/runbook，且“不实现 production provider”；
  - 当前 Tasks/Plan 没有其他任务拥有该 production provider；
  - 真实代码中只有 `DeletionReplaySource` 端口和测试 stub，没有 production source/provider 装配。
- **触发场景**：按 r13 的 17 文件写集实施 `cmd/han-server` wiring 时，没有可构造并注入的 production inventory reader。若依赖在启动期必填，服务只能启动失败；若延迟到恢复运行，恢复只能永久返回 Unavailable，不能形成 Ready 正向路径。
- **影响**：可以实现“缺失时 fail closed”，但不能实现原 finding 要求的真实旧备份重放与 production source 正向能力；F001 被 r13 自判“通过”缺少事实基础，合同不完整且无法闭环交付。
- **最小修复方向**：先由 Lead/架构责任人确认 inventory authority、concrete provider、受控数据格式/配置、装配位置和唯一交付任务；若引入新基础设施或跨服务协议，按 r13 自身规则修订 architecture/ADR 后再回 design。若声称复用现有 provider，则 design 必须精确指出现有实现、配置与 owner，且证明不是 fixture/stub。
- **验证**：production composition 能以真实 provider 构造唯一 source；有效 inventory 可完成 ReplayAll→operation/audit 对账→Ready；missing、不可读、错 backup、漂移与不可判定均 fail closed；全仓无第二 source/provider authority。
- **方向决策**：是。Reviewer 不替项目选择 provider。

### `ARTICLE-DES-R13-S42-R01-F002` — HIGH — Begin commit-unknown 的恢复判定无法从已定义的 ReadCurrent 结果取得 generation/token

- **位置**：design `1915-1929`、`1934-1936`、`1952-1957`、`2178-2181`
- **违反依据**：原 `ARTICLE-S4.2-REV-R01-F002` 要求 generation/token 所有权与 CAS 可实施；A-REC-07 要求 Begin commit outcome unknown 时“不盲目新 token”并可确定性收敛。
- **触发场景**：
  1. invocation 生成 token T2 并调用 Begin；
  2. 数据库实际已提交，或该调用加入了已有 token T1 的 running attempt，但响应/commit outcome 未知；
  3. r13 要求调用方以原 token 调用 ReadCurrent，并区分“同 generation/token running”“较新 generation”；
  4. r13 定义的 ReadCurrent 只返回 Ready/Unavailable；running 的六字段 Unavailable 不含 generation/token，因而调用方无法完成该分支判断。
- **影响**：实现者只能猜测、重新 Begin、生成新 token，或一直返回 pending；这会破坏 r13 明令禁止的盲写/盲换 token，并使 F002 的 owner/commit-unknown 合同出现多种不兼容实现。`Begin` 遇到已有 running 又返回他人 token 的 cooperative join 语义也未与“每 invocation 始终复用自己的 token”明确区分。
- **最小修复**：明确分离内部 store 结果与对外 ReplayStatus DTO，例如定义只在 article 内部使用的 `LoadAttempt/ReadRun`，返回 status、generation、token、backup binding 和 current 指针；或规定 Begin 以同一原 token安全重试并给出各状态的确定性返回。同步说明加入已有 attempt 后 token 所有权、允许谁调用 Fail、何时可推进 generation。
- **验证**：真实 PostgreSQL 故障注入覆盖新行 Begin、failed→新 generation、加入已有 running 三种 commit-unknown；调用方不生成第二 token、不暴露 token 到公共 DTO，旧 Fail/Ready 均不能越过新 generation。

### `ARTICLE-DES-R13-S42-R01-F003` — MEDIUM — 输入基线仍引用已漂移的 architecture / identity boundary 修订与哈希

- **位置**：design `97-100`、`2329-2333`
- **证据**：
  - design 引用 architecture r24、identity boundary r11，并把旧 identity SHA `fb5197…` 写成当前载体；
  - 当前 architecture 为 r27，SHA256 `A153AC6DEA7BD725EA9470D2D2933396EE9A2F12399CC11FDEC2CB97B4E443CB`；
  - 当前 identity boundary 为 r13，SHA256 `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3`。
- **触发场景**：后续 Task-Planner、实现者或复审者按 r13 的“当前输入”复核消费者合同，却取到旧 revision/hash 或无法重建评审基线。
- **影响**：本次定向比较未发现 S16 的 ReplayStatus 消费语义发生冲突，但 r13 的可追溯声明不真实，后续上游变更可能被静默漏检。
- **最小修复**：仅更新 design 的当前 revision/hash和“哪些章节未受后续修订影响”的说明；不得把旧 accepted artifact 与当前载体混称。
- **验证**：重新计算三份上游 SHA，逐项比较 S16、恢复顺序和 authority 条款，并在 design 中固定实际消费的当前 revision。

## 4. R01 F001～F005 设计侧状态

| 原 finding | r13 设计侧状态 | 结论 |
|---|---|---|
| F001 production source/wiring | **OPEN** | 唯一 source、真实依赖和 fail-closed 已定义，但 concrete inventory provider、格式、装配输入和交付 owner 缺失，无法形成 production 正向路径 |
| F002 attempt/CAS | **OPEN** | generation/token 与 CAS 主线已补，但 Begin commit-unknown 无可观察的内部 owner 状态，合同不能按文字实现 |
| F003 UTC 微秒 | **CLOSED（设计侧）** | cutoff、operation/audit、completed_at、DB 回读与首次响应统一 UTC 微秒；同 input 复用数据库 ref/digest/time |
| F004 单一 authority | **CLOSED（设计侧）** | `internal/recovery` 被指定为唯一 Go authority；article 只允许直接消费/alias；migration CHECK 仅作持久化约束并要求契约测试 |
| F005 PostgreSQL证据 | **CLOSED（设计侧合同）** | A-REC-04～07 已明确要求真实 PostgreSQL 多连接/多事务，memory 不可关闭验收；实际测试执行仍未发生 |

## 5. 已确认一致的范围

- article→recovery 单向依赖、strict Ready20/Unavailable6、canonical/digest authority 与 identity DTO-only 消费一致。
- UTC 微秒、同 input 复用历史 immutable evidence、input drift 立即使旧 Ready 失效并要求再次 Replay 的合同明确。
- `0009_article_replay_attempt.sql` 作为前向 migration、禁止修改 `0003`、停止旧 writer 后升级的方向一致。
- A-REC-04～07 的真实 PostgreSQL层级、并发、回滚、时间精度与 commit-unknown 覆盖要求明确。
- 17 文件表被清楚标为建议写集而非实施授权；现役 Tasks/Plan 仍是五文件暂停状态，须由 Task-Planner另行对账。

## 6. 待决定、非阻断事项与未验证风险

### 待决定

- inventory authority/provider/config/data format/交付 owner。该决定可能要求 architecture/ADR 修订，不能由本 Reviewer代定。

### 非阻断建议

- 明确 ReadCurrent/ReadStatus 触发的 source snapshot 会执行幂等 ReplayAll，避免维护人员把该入口误认为纯读；若希望纯读，应另分离 replay 与 freshness snapshot。

### 未验证

- 未连接真实 PostgreSQL，未验证 0009、行锁、CAS、唯一冲突、微秒回读、rollback 或 commit outcome unknown。
- 未运行真实 Journal/keyring/inventory/ReplayAll、DeletionOperation/AuditEvent 对账或旧备份恢复。
- 未执行 identity 双次 ReadStatus、MarkConsumed、真实 Host、部署、E2E、容量或 production readiness。
- 未把并行修订中的 test-plan r10 作为固定依据。
- 共享工作树存在大量他人改动；本评审只对固定 design SHA 和上述有界消费者/依据下结论。

## 7. 复审条件

1. 固定新的 design revision/SHA，并逐项映射本报告 F001～F003。
2. F001 已有获批准的 concrete provider/格式/config/owner；若涉及新基础设施或跨服务协议，先完成 architecture/ADR 方向决定。
3. F002 定义可实现的内部 attempt 读取/Begin 重试合同，不把 generation/token 加入公共 ReplayStatus DTO。
4. 更新 architecture/identity boundary 的当前 revision/hash，并证明相关 S16 合同无漂移。
5. Task-Planner 完成 17 文件、0009、共享 `cmd/han-server` 与已关闭 S4.6 的串行所有权对账后，方可恢复实施。
6. 后续代码复审必须提供 A-REC-04～07 的真实 PostgreSQL证据；memory/compile-only 不可替代。

## 8. 最终 Gate

**CHANGES REQUIRED**

r13 已完整补出 F003、F004、F005 的设计侧合同，也给出 F001/F002 的主要方向；但 production inventory 正向依赖仍无 owner/provider，且 attempt commit-unknown 的 owner 状态不可观察。故不能认定原 3 HIGH + 2 MEDIUM 已全部形成一致、可实施、无第二 authority 的合同。

