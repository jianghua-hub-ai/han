# article-management-S4.2 独立代码评审 r01

- Reviewer：独立非作者 Reviewer
- 评审日期：2026-10-25
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 作者：Pasteur `01a1716c-c0a1-7263-b02c-0c466e4ecaaa`（已关闭；本轮未代修）
- **Code Review Gate：FAIL**

## 1. 固定对象与漂移核对

评审开始时分支、HEAD 与下列五个 SHA256 全部匹配，0 mismatch；若任一对象漂移，本报告结论失效。

| 文件 | SHA256 |
|---|---|
| `internal/article/deletion_replay.go` | `7B19CB5A08D9A7770CA1294E6FD18821D62D74865D99076AD7A37FA80FC33549` |
| `internal/article/replay_evidence.go` | `698E6B18D482E8BFE59D6791774542475A6EB1C2CBDF42541102FA774405C6E8` |
| `internal/article/replay_status_v1.go` | `3E14EB098E29664463A895A05280DB4E941DDBC16B76C44A73D97EC709E010F7` |
| `internal/article/deletion_replay_test.go` | `60CCA8C40F910231D0F220A440394DBC5DCF84255401005E6CC0B3823FA0A206` |
| `internal/article/replay_status_v1_test.go` | `DFAFB02787B895BE0F5E50B018ECD57075D038649BADCD3954B4AE98B241BCBD` |

## 2. 依据与两轮实际检查范围

完整读取 Reviewer Role、`add-code-review`、code-review checklist、abstraction/coding/no-hardcode、已启用 go-service profile 与产品安全红线。验收依据为 tasks SSOT S4.2、Article Story 4/S16、design §3.3.1、§3.4.2/7、Ready20/Unavailable6、前 16 canonical、固定 digest，以及 test-plan A-REC-01～07。

### Round 1：五个固定对象

逐行检查：

- strict Ready/Unavailable variant、字段集合、顺序、null/type 与固定 digest；
- run+namespace 唯一、backup 首绑、事务和行锁；
- input_digest 幂等、evidence append-only/current 原子更新；
- 错 backup、旧版本、摘要损坏与 input 漂移的 fail-closed；
- 相同 input 的 ref/digest/completed_at 稳定性；
- 并发和崩溃测试是否覆盖验收合同。

### Round 2：真实消费者与上游接缝

有界检查：

- `internal/recovery/replay_status_v1.go` 与 `coordinator.go` 的唯一 schema authority、严格解码、二次读取与 exact ref/digest/time 消费；
- Article deletion journal、finalizer、repository、migration 中的 ReplayRun/Evidence 表、唯一约束、外键、append-only 与行锁合同；
- identity consumer/shared fixture 的只读边界；
- 全仓仅搜索 `DeletionReplaySource`、`DeletionReplayProvider`、`PostgresReplayEvidenceStore` 的生产实现与装配。

未扩为全库审计；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。

## 3. Findings（按严重度排序）

### `ARTICLE-S4.2-REV-R01-F001` — HIGH — 旧备份重放只有未实现端口与测试 stub

- **位置**：`internal/article/replay_evidence.go:38-41`；`internal/article/deletion_replay.go:54-73`；`internal/article/deletion_replay_test.go:238-259`
- **违反依据**：tasks S4.2 目标“实现 ReplayStatus producer、证据存储与旧备份重放”；design §3.4.7 第 2 步要求验证 keyring/Journal、读取 inventory、执行 ReplayAll 及 operation/audit 对账；组件图要求 provider 连接 Journal、finalizer、repository、audit、inventory。
- **证据**：全仓搜索仅发现 `DeletionReplaySource` 接口和 `replaySourceStub`；没有生产实现，也没有 `NewDeletionReplayProvider` / `NewPostgresReplayEvidenceStore` 的生产装配调用。
- **触发场景**：真实恢复旧备份时，没有组件可产生可信的 Journal/inventory/operation/audit 输入或执行 ReplayAll。
- **影响**：当前成果只能对调用方提供的合成快照生成 evidence，无法证明永久删除不会被旧备份复活；S4.2 核心目标不可达。
- **修复建议**：回到 Lead/设计确认生产 source 的精确所有权和文件边界；实现并装配真实 Journal/keyring、backup inventory、ReplayAll、DeletionOperation/Audit 对账，禁止由调用方任意注入“已验证”计数。该方向会扩大当前五文件合同，不由 Reviewer 代定。
- **复审条件**：固定新的实现修订，提供真实 source/装配及对应故障、损坏和旧备份回放证据。

### `ARTICLE-S4.2-REV-R01-F002` — HIGH — 旧并发调用可把较新的 Ready 无条件降级为 Failed

- **位置**：`internal/article/deletion_replay.go:54-83,146-153`；`internal/article/replay_evidence.go:135-176,627-631`
- **违反依据**：design §3.1.8/§3.4.7 与 A-REC-04/06 要求相同 backup 并发收敛到同一 evidence，current 原子指向胜出结果；失败调用不得破坏已发布的较新 Ready。
- **触发场景**：调用 A、B 对相同 run+backup 均完成 `Begin`；B 先 `Publish` 并提交 Ready；A 随后因 source/refresh/编码错误调用 `Fail`。`Fail` 锁行后不检查 attempt/generation/current，直接覆盖为 `failed`。
- **影响**：已成功发布且仍匹配当前输入的 evidence 立即不可读；Recovery Gate 从 Ready 回退为 Failed。更多交错中，任意过期失败均可覆盖后续成功。
- **修复建议**：为每次运行尝试建立可比较的 generation/attempt 所有权，或采用能证明失败仍属于当前 running 尝试的 CAS；不得只以共享 `status='running'` 作为条件。若需要 schema/migration，先作为方向决策返回 Lead。
- **复审条件**：真实 PostgreSQL 双事务覆盖“成功先提交、旧失败后到达”及多调用交错，最终 Ready/ref/digest/time 保持稳定。

### `ARTICLE-S4.2-REV-R01-F003` — HIGH — 时间未在持久化前规范化，破坏 completed_at 稳定性并可能损坏摘要

- **位置**：`internal/article/replay_evidence.go:424-447,451-465,468-499,529-562`；`internal/article/replay_status_v1.go:119,152`
- **违反依据**：Story 4/S16、tasks S4.2、design §3.4.7 与 A-REC-04 要求相同 input 返回完全相同的 `completed_at`；消费者会在二次读取时核对时间。
- **触发场景**：生产时钟或 backup cutoff 含微秒以下精度。首次 `Publish` 返回内存中的纳秒时间，但 PostgreSQL `timestamptz` 持久化后只有微秒精度；重试从数据库读取的 `completed_at` 不再与首次响应完全相同。若 `backup_cutoff` 含微秒以下精度，持久化后重算 canonical digest 也可能与原 digest 不同。
- **影响**：响应丢失重试不能保证相同 time；identity 的二次 Ready 核对可能 fail closed；已写 evidence 也可能被判为 unreadable。
- **修复建议**：在计算 digest、创建 evidence、写库和首次响应前统一使用项目既有 UTC 微秒规范，并拒绝或规范化非 canonical source 时间；增加带亚微秒 Clock/BackupCutoff 的 PostgreSQL 回归。
- **复审条件**：证明首次响应、数据库回读和重复调用的 cutoff/completed_at 与 digest 完全稳定。

### `ARTICLE-S4.2-REV-R01-F004` — MEDIUM — producer 仍复制 schema version/namespace authority

- **位置**：`internal/article/replay_status_v1.go:66-73,110-112,144-149`；实际字面量定义位于 `internal/article/model.go:6-10`
- **违反依据**：tasks S4.2 明确“不得复制第二 ReplayStatus authority”；`internal/recovery/replay_status_v1.go:16-24` 已定义唯一 v1 version、namespace 与固定 fixture digest authority。
- **触发场景**：consumer authority 升级或修正，而 Article 本地常量未同步；producer、DB 与 strict consumer 产生漂移。
- **影响**：编译仍成功，但所有 Ready/Unavailable 在运行时被 consumer 拒绝，恢复持续关闭。
- **修复建议**：producer 直接复用或显式 alias `recovery.ReplayEvidenceVersionV1` 与 `recovery.ArticleDeletionNamespaceV1`；数据库字面量一致性应由契约测试验证，不再建立 Go 侧第二权威。
- **复审条件**：搜索证明 Go producer 无第二份 schema 字面量，并有 authority 一致性测试。

### `ARTICLE-S4.2-REV-R01-F005` — MEDIUM — A-REC-04～07 的关键 PostgreSQL行为没有测试证据

- **位置**：`internal/article/deletion_replay_test.go:15-209,292-477`
- **违反依据**：test-plan A-REC-04～07 明定 PostgreSQL 并发、双事务与崩溃恢复；tasks 关闭条件要求 run/backup 并发、input 漂移、崩溃和 stable ref/digest/time。
- **证据**：全部行为测试使用 `memoryReplayEvidenceStore`；没有实例化 `PostgresReplayEvidenceStore`。input drift 只断言旧 Ready 失效，没有再次 Replay 并验证唯一新 evidence/current；崩溃只覆盖 run 插入后和 commit 后响应丢失，未覆盖 replay 后、evidence/current 提交前；没有“旧失败覆盖新 Ready”、亚微秒时间、旧版本/摘要损坏的数据库回归。
- **影响**：内存锁实现绕过 SQL、真实行锁、唯一冲突、时间精度、commit/rollback 等风险，现有 PASS 不能支持 S4.2 关闭。
- **修复建议**：在获准的测试范围内增加真实 PostgreSQL 双连接/双事务用例和故障点；不得用 compile-only 或 memory store 代替。
- **复审条件**：A-REC-04～07 的真实 PostgreSQL自动化证据齐全，并回归 F002/F003。

## 4. 已确认正确的范围

- Ready 使用 `internal/recovery` DTO/严格 decoder，20 字段顺序与固定 digest 正向 fixture通过。
- Unavailable 仅序列化 6 字段，pending/unknown/failed 由共享严格 decoder 验证。
- SQL 具备 `(run_id,namespace)` 首绑唯一、evidence digest 唯一、append-only、current 外键与 `FOR UPDATE`。
- ReadCurrent 对错 backup、未知版本、不可读 evidence、摘要不一致及 input 漂移均 fail closed。
- 五个固定文件 gofmt、定向普通/race、Article package、vet 与 integration-tag compile-only 均通过。

这些正确项不关闭上述阻断 finding。

## 5. 未验证风险

- 未连接真实 PostgreSQL，未验证真实锁等待、唯一冲突、时间精度、事务回滚、commit outcome unknown 或多连接竞争。
- 未恢复真实旧备份，未运行真实 Journal/keyring/inventory/ReplayAll、DeletionOperation/Audit 对账。
- 未运行 identity 联合恢复、真实 Host、部署或 E2E。
- integration-tag 命令为 `[no tests to run]`，只证明编译。

## 6. Gate 与复审条件

**Gate：FAIL。**

阻断项为 F001～F003；F004/F005 也应在关闭 S4.2 前处理。复审必须提供：

1. 原 finding 到修复修订的逐项映射；
2. 新固定对象哈希与允许范围；
3. F001 的真实 source/装配边界；
4. F002 的 PostgreSQL并发回归；
5. F003 的 UTC 微秒稳定性回归；
6. A-REC-04～07 的真实 PostgreSQL证据；
7. 五文件及消费者合同无漂移、staged 为空、无 identity/schema/migration 未授权越界。

## 7. Reviewer 独立命令与结果

```text
git branch --show-current
git rev-parse HEAD
git status --short
git diff --cached --name-only
Get-FileHash -Algorithm SHA256 <五个固定对象>
rg -n --glob '*.go' "DeletionReplaySource|ReplayOldBackup\(|Snapshot\(" .
rg -n --glob '*.go' "NewDeletionReplayProvider|NewPostgresReplayEvidenceStore|PostgresReplayEvidenceStore|DeletionReplayProvider" .
```

结果：branch/HEAD 与五个 SHA256 匹配；staged 为空；只发现接口、测试 stub 和 store/provider 定义，未发现生产 source 或装配。

```text
gofmt -l <五个固定对象>
go test ./internal/article -run "TestReplayStatus|TestDeletionReplay|TestReplayEvidence" -count=1
go test -race ./internal/article -run "TestReplayStatus|TestDeletionReplay|TestReplayEvidence" -count=1
go test ./internal/article -count=1
go vet ./internal/article
go test -tags=integration ./internal/article -run "^$" -count=1
```

结果：全部退出码 0；`gofmt -l` 无输出；integration-tag 为 `[no tests to run]`。

## 8. 越界、工作树与交付

- 本 Reviewer 仅新增本报告；未修改五个被审文件、Tasks/Plan、identity、schema/migration 或其他代码。
- 未 add/commit/push/切分支、未清理或回滚任何他人改动。
- 未读取或使用 `HAN_TEST_POSTGRES_DSN`。
- `han-server.exe` 保留，未触碰。
- 工作树原有大量已修改/未跟踪内容均按用户要求保留；本报告不对其正确性作结论。

