# article-management design r14 / S4.2 Design Review R02 独立复审

- Reviewer：原独立 Reviewer Gauss
- 被审作者：Feature-Designer Kierkegaard `01a171dc-413f-7760-bc52-145e00b29224`
- 评审日期：2026-10-25
- objectType：design
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 固定对象：`.add/project/knowledge/design-workspace/article-management/design.md`
- revision：`article-management-design-ir01-r14`
- 固定 SHA256：`AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`
- 原报告：`.spec/wip/article-management/review-design-r13-s4.2-r01.md`
- 原报告 SHA256：`7FD43B16368F19A0C0F055AE0E9267CC16311FD3E05448A87829961EB1E1C0FC`
- **Design Review Gate：PASS**

## 1. 固定对象与评审结论

评审开始时，design、原报告、architecture r28、ADR-0020 所在载体以及 article/identity boundary r13 的 SHA256 均与指定值匹配；分支和 HEAD 匹配；staged 为空；R02 报告尚不存在。

两轮检查未发现必须再次修改 architecture、ADR 或 boundary 的方向缺口，也未发现第二 authority、第二 production composition 或新的阻断 finding。原 R01 的 F001、F002、F003 均为 **CLOSED**。本结论只关闭 design r13 Review R01 的设计 findings，不关闭原 S4.2 Code Review、不批准实施写集，也不外推真实 PostgreSQL、恢复演练或 production readiness。

## 2. 两轮实际检查范围

### Round 1：r14 对原 findings 的直接整改

逐项检查：

- F001：concrete `FileBackupInventoryReader`、唯一配置、strict manifest、owner、文件安全、TOCTOU、freshness、composition 和 S4.2/S4.4/S4.5 分工；
- F002：内部 attempt DTO、expected snapshot、candidate/authoritative token、Begin 分支、commit outcome unknown、Ready/Fail CAS 和 generation 推进；
- F003：architecture/boundary revision 与 SHA、S16 tagged union、DTO-only 消费、依赖方向和恢复顺序；
- r14 建议 19 文件写集、`0009_article_replay_attempt.sql`、S4.2/S4.5 与已关闭 S4.6 的所有权说明。

### Round 2：上游、真实消费者与现状交叉核对

定向核对：

- architecture r28 的 inventory authority/provider/config、安全读取、唯一 composition 与分工；
- ADR-0020 的 production inventory、Ready20/Unavailable6 和恢复 Gate；
- article boundary r13、identity boundary r13 的 S16、DTO-only 消费与恢复顺序；
- `internal/recovery/replay_status_v1.go` 的真实公共 DTO，以及 coordinator/article replay/store/status/composition 现状；
- 当前 migration `0001`～`0008` 与 `0003_article.sql`；
- Tasks/Plan 中 S4.2、S4.4、S4.5、S4.6 的状态、登记写集与共享文件所有权。

## 3. 原 findings 复审

### `ARTICLE-DES-R13-S42-R01-F001` — HIGH — **CLOSED**

- **位置**：design `494`、`554-569`、`806`、`822-832`、`840-843`、`866-872`、`2216-2217`、`2284-2286`、`2300-2324`、`2328-2330`；architecture `599-607`、`933`；ADR-0020 `97`、`127-135`、`155`。
- **整改证据**：
  - r14 固定 article-owned concrete `FileBackupInventoryReader`，唯一输入为必填且无默认/别名/fallback 的 `HAN_ARTICLE_BACKUP_INVENTORY_PATH`。
  - manifest 固定为 strict `article-backup-inventory/v1`，明确唯一生成/签发 authority、只读挂载 owner、字段和顺序、UTC 六位微秒、canonical/digest、1,048,576-byte 上限及 wrong backup/cutoff/scope 的 fail-closed。
  - 文件读取固定 no-symlink、regular file、只读与权限证明，以及 `lstat → no-follow open → fstat → bounded read → fstat → lstat` 的 TOCTOU 检查；每次 Replay 和 freshness 都重读，不缓存、不使用 watcher。
  - 唯一 composition 是既有 `cmd/han-server`；source 必须显式取得同一 Journal/keyring、reader、ReplayAll、DeletionOperation、AuditEvent、DB/clock/ID，禁止 global、locator、第二 source/reader/provider 或 fixture fallback。
  - S4.2 拥有 production reader/source/wiring；S4.4 使用真实旧备份和真实 inventory 做联合 Gate；S4.5 仅负责 synthetic fixture/runbook。
- **触发复核**：有效 manifest 的 production 正向路径、缺失/损坏/漂移时的关闭路径，以及唯一装配路径均已有确定输入和 owner。
- **影响判断**：原先“只能 fail closed、不能形成 production Ready 正向能力”的设计缺口已消失；合同与 architecture r28/ADR-0020 一致，未形成第二 authority。
- **最小修复**：无需继续修改 design。Task-Planner 仍须先把建议写集转换为实际授权，实施不得把 fixture、业务 PostgreSQL、目录扫描或远程 API引入 production source。
- **后续验证**：实现复审须证明 strict parser、1 MiB 边界、文件替换/权限漂移、每次 freshness 重读、唯一 composition，以及有效真实依赖可形成 Ready、全部不可判定结果保持 fail closed。

### `ARTICLE-DES-R13-S42-R01-F002` — HIGH — **CLOSED**

- **位置**：design `1940-1968`、`1976-1992`、`2212-2215`、`2292-2295`、`2304-2315`。
- **整改证据**：
  - 公共 `internal/recovery.ReplayStatusV1` 明确保持 strict Ready20/Unavailable6；generation、token、内部状态和 current 指针不进入公共 DTO。真实代码 `internal/recovery/replay_status_v1.go:36-74` 与该合同一致。
  - article 内部定义完整 `AttemptSnapshot`、只读 `LoadAttempt` 和带 disposition 的 `BeginResult`；candidate token 不天然取得写权，后续 CAS 使用 store 判定的 authoritative generation/token。
  - `Begin` 固定 expected snapshot，并区分首行创建、同 candidate 恢复、加入 current running、failed/Ready-stale 推进、already-ready 和 superseded；generation 只在首行或精确匹配的 failed/Ready-stale 分支推进。
  - 新行、failed/Ready-stale 推进、join-running 三类 Begin commit outcome unknown 均给出同 candidate、LoadAttempt 与必要的 Begin 重试路径；join-running 不能仅凭 LoadAttempt 或公共 Unavailable 取得写权。
  - `Ready`/`Fail` 均受 generation+token CAS；旧 handle 零写入，Fail 不能从 Ready 降级或覆盖较新 generation，Ready/Fail commit unknown 有确定读取与重试收敛路径。
- **触发复核**：Begin 已提交但响应未知、加入他人 running、旧 Fail 晚到、Ready 提交后响应丢失。
- **影响判断**：调用方不再依赖缺少 token/generation 的公共 ReadCurrent 判断 owner，也不需要盲目生成第二 token；原不可实施分支已变为确定性的内部状态机。
- **最小修复**：无需继续修改 design。实现必须保持“LoadAttempt 只观察”的边界：自己的 candidate 已持久化可由数据库 snapshot 证明；加入现役他人 attempt 时仍须通过 Begin 的持锁分支取得 `joined_current` handle，不能把公共 DTO 当 owner 证据。
- **后续验证**：A-REC-04～07 必须使用真实 PostgreSQL 多连接/多事务和故障注入，覆盖三类 Begin commit unknown、旧 Fail/Ready、唯一冲突、回滚、微秒回读，并断言公共 DTO 始终不含 generation/token。

### `ARTICLE-DES-R13-S42-R01-F003` — MEDIUM — **CLOSED**

- **位置**：design `96-118`、`2282-2286`；article boundary `3`、`126-129`、`183`；identity boundary `3`、`34`、`66`、`88`、`145-146`、`205`。
- **整改证据**：
  - r14 精确记录 architecture r28 SHA256 `554686B77D0557F16A735F49F63626D922F41DAE8A8574B17980B7CB55CF1BCC`。
  - r14 精确记录 article boundary r13 SHA256 `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E`、identity boundary r13 SHA256 `0217349BEAF9C529D0EB01F4451F34E930E5A9682F23326661BEA93CE7119BF3`。
  - r14 逐项说明 S16 仍是 Ready20/Unavailable6，identity 仍为 DTO-only，依赖方向仍为 `article -> recovery`，恢复顺序仍为 ReplayAll/对账/Ready → 双 namespace → identity 二次 ReadStatus → MarkConsumed → readiness。
  - 两份 boundary 的 S16 正文与上述核对一致；identity r13 的新增范围是 password-reset resend，不改变 article replay 消费合同。
- **触发复核**：按 r14 的输入基线重建 architecture/boundary 消费关系。
- **影响判断**：r13 的 revision/hash 漂移已消除，当前基线可复核。article boundary frontmatter 仍引用较早 architecture revision，但 r14 已明确只把其当前 S16 正文作为 boundary 事实，不把该 frontmatter 提升为 r28 provider 依据；未造成 r14 合同冲突。
- **最小修复**：无需继续修改 design，也无需因本 finding 修改 boundary。
- **后续验证**：后续上游任一载体变更时重新固定 SHA，并重新比较 S16、DTO-only、依赖方向和恢复开放顺序。

## 4. 新 findings

按 CRITICAL / HIGH / MEDIUM / LOW 检查，**新增 finding：0**。

未发现必须回 architecture、ADR 或 boundary 重新决定的方向缺口；停止条件未触发。

## 5. 19 文件、migration 与任务所有权核对

- design `2300-2322` 的建议写集计数为 19，包含 5 个现有 S4.2 候选文件、production source/reader及其测试、真实 PostgreSQL integration、`0009_article_replay_attempt.sql`、既有 Journal/finalizer/repository 接缝以及 `cmd/han-server` wiring/tests。
- 当前仓库 migration 仅有 `0001`～`0008`；r14 要求前向新增 `0009`，并禁止修改 `0001`～`0008`，内部一致。
- design 明确 19 文件只是供 Task-Planner R 对账的建议，不是实施授权；当前 Tasks/Plan 仍登记 S4.2 五文件并暂停，尚未同步不是 design finding。
- 已关闭 S4.6 仍拥有历史 production composition/router/Journal/worker 成果；r14 只要求在同一 `cmd/han-server` root 串行接入新的 replay reader/source，不重开或建立第二 root。
- S4.5 当前只登记 runbook 与 synthetic fixtures，且不得实现 production provider；S4.4 承接真实旧备份、真实 Journal/keyring/inventory 和联合恢复 Gate。该分工与 architecture r28/ADR-0020 一致。

**后续条件**：恢复实施前，Task-Planner 必须对 19 文件、`0009`、共享 `cmd/han-server`、已关闭 S4.6 成果和 S4.5/S4.4 边界完成串行所有权对账；本报告不修改 Tasks/Plan，也不把建议写集当作授权。

## 6. 实际读取

### 完整读取

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/decisions-guide.md`
- `.add/project/knowledge/design-workspace/article-management/design.md`
- `.spec/wip/article-management/review-design-r13-s4.2-r01.md`

### 定向读取与检索

- `.add/project/knowledge/design/architecture.md`
- `.add/project/knowledge/decisions/content.md`（ADR-0020）
- `.add/project/knowledge/design/article-management/boundary.md`
- `.add/project/knowledge/design/identity-auth/boundary.md`
- `internal/recovery/replay_status_v1.go`
- `internal/recovery/coordinator.go`
- `internal/article/deletion_replay.go`
- `internal/article/replay_evidence.go`
- `internal/article/replay_status_v1.go`
- `internal/article/model.go`
- `internal/article/deletion_journal.go`
- `internal/article/deletion_finalizer.go`
- `internal/article/repository.go`
- `cmd/han-server/main.go`
- `migrations/0001`～`0008` 与 `migrations/0003_article.sql`
- `.add/project/knowledge/tasks/tasks.md`
- `.add/project/knowledge/plans/ir-01-content-community-batches.md`
- 全仓 production source/provider/composition、ReplayStatus、S4.2/S4.4/S4.5/S4.6 消费与定义检索。

未把正在并行修订、可能漂移的 test-plan r10 作为固定依据。

## 7. 未验证边界与限制

- 本轮是非代码 design review；未运行 Go 测试、race、vet、build、migration、部署或 E2E。
- 未连接或运行真实 PostgreSQL；未验证 `0009` SQL、行锁、CAS、唯一冲突、回滚、微秒回读或 commit outcome unknown。
- 未运行真实旧备份、Journal/keyring、inventory 文件、ReplayAll、DeletionOperation/AuditEvent 对账、identity 双次 ReadStatus 或 MarkConsumed。
- 未验证目标 POSIX/Windows 文件系统对 no-follow、ACL、只读挂载、原子替换和 TOCTOU 检查的真实行为；设计要求无法证明的平台 fail closed。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`；未访问网络或外部服务。
- 共享工作树含大量他人未提交成果；本报告只对固定 design SHA 和列明的有界依据下结论，不评价未纳入对象的其他改动。
- design Gate PASS 不表示原 S4.2 Code Review 的 3 HIGH + 2 MEDIUM 已由代码关闭；实现、真实 PostgreSQL证据和新的独立代码复审仍是必需条件。

## 8. 最终 Gate

**PASS**

`ARTICLE-DES-R13-S42-R01-F001/F002/F003` 全部 CLOSED；r14 与 architecture r28、ADR-0020、两份 boundary S16 和真实公共 ReplayStatus 消费者一致，形成了可实施、fail-closed、无第二 authority 的设计合同。新增阻断 finding 为 0。

固定对象开始 SHA256：`AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`。  
固定对象结束 SHA256：`AC52E658247CED89A57C54B38C7F55FA9AA56772406A42F178678B1C1D5108A5`。
