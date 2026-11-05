# identity-auth-S3.2 / F005 PostgreSQL fixture 修复复审 r03

- Reviewer：Ampere（非作者）
- 日期：2026-10-23
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 结论：**本轮 fixture 最小修复 PASS；无新增 finding。**
- 总 Gate：**`IDENTITY-S3.2-REV-F005` 仍为 OPEN / 受限；不得据此关闭 S3.2 或 B4。**
- 操作边界：被审代码及依据全程只读；仅新增本报告。未修改 r01/r02、代码、migration、设计、tasks、Plan 或依赖，未 stage/commit/push/PR/merge，未执行 E2E。

## 1. 固定对象

| 对象 | SHA-256 |
|---|---|
| `internal/identity/login_integration_test.go` | `1D2DA4502D640689767D1C9D8D013576E73A266CD4D649F1044CBB23EC1C0DE3` |
| `internal/identity/login.go` | `89B5537BC8A4D0446DD55655C87492E35787EFEC8DD923A031F0C7B930237871` |
| `internal/identity/login_test.go` | `17EB9D870700548464EE5D410ACBF861668DB9CEF7003AC6F331590B7349C7DD` |
| `.spec/wip/identity-auth/review-code-s3.2-f005-r02.md` | `1CEA171BEAC836ADE64D00CFD4A51CBA7902A87F6410C3F5E2A9CB786DD6BCB0` |
| `internal/audit/store.go` | `091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F` |
| `migrations/0001_shared.sql` | `CEE72F1470B49185B3B33C7655F792ADFFCBC1E030AF6669E0902775104B7A14` |

被审文件相对 HEAD 为未跟踪文件，普通 Git diff 不能展示历史差异。本轮在内存中把当前 fixture 反向恢复为 `system := "identity"` 与 `ActorSystem: &system` 后，重建 SHA-256 为：

`B76ADABD7F7A6146FDC9C666699714CF53EA1485C9E916D4F74E61D46C8337FC`

该值与 r02 固定 revision 完全一致，证明本轮目标文件相对 r02 只包含声明的两处最小 fixture 修改。

## 2. 事实、设计与决定

### 可核事实

- `migrations/0001_shared.sql:29-33` 的 `audit_events` actor 约束要求：
  - `anonymous`：`actor_user_id IS NULL` 且 `actor_system IS NULL`；
  - `user`：`actor_user_id IS NOT NULL` 且 `actor_system IS NULL`；
  - `system`：`actor_user_id IS NULL` 且 `actor_system` 为非空文本。
- `internal/audit/store.go:40-98` 将 `audit.Event` actor 字段原样传入 INSERT，不绕过、不改写数据库约束。
- 当前 `completeLoginAuditEvent` 位于 `internal/identity/login_integration_test.go:673-698`，组合为 `ActorUser + 非 nil ActorUserID + nil ActorSystem`。
- fixture 只有三个直接调用点：`login_integration_test.go:53`、`:187`、`:309`。

### 既有设计/合同

- F005 的 PostgreSQL 测试必须使用真实 migration 和 PostgreSQL 18 约束，不得通过放宽 schema、mock 结论或 Skip 冒充真实提交状态。
- 本轮不改变 AuditEvent 数据模型、Store 行为、数据库约束或登录生产行为。

### 本轮评审决定

- 只判断 fixture 是否恢复为合法 actor 事实并维持既有测试目标。
- system actor 的独立正向约束覆盖不属于本轮 fixture 修复；Reviewer 不借此修改 audit 约束或扩大为全仓测试重构。

## 3. 两轮检查

### Round 1：直接修订、约束与最小性

检查范围：

- `internal/identity/login_integration_test.go:673-698`
- `internal/audit/store.go:40-98`
- `migrations/0001_shared.sql:14-34`
- r02 固定 revision 的内存反向重建

结论：

1. 删除局部 `system := "identity"` 并把 `ActorSystem` 设为 nil，准确满足 `ActorUser` 分支要求；`ActorUserID` 仍为随机合法 UUID，未把 user actor 降级为 anonymous。
2. migration 与 `Store.Append` 哈希未变化；修复没有删除、放宽或规避 `audit_events` CHECK，也没有在 Store 内迎合非法 fixture。
3. 反向重建哈希与 r02 完全一致，未发现夹带断言放宽、数据模型变化或无关重构。
4. fixture 的其余 14 字段、canonical metadata、微秒时间和随机隔离 ID 均保持不变。

### Round 2：调用用例、真实 PostgreSQL 与盲区

检查范围：

- commit unknown 实际提交/回滚：`login_integration_test.go:22-145`
- rollback unknown Append 后三态：`login_integration_test.go:147-270`
- 完整字段 PostgreSQL round-trip：`login_integration_test.go:272-342`
- fixture、schema 创建、PostgreSQL 18 门禁：`login_integration_test.go:650-733`
- `internal/audit/store_integration_test.go` 的直接 Store round-trip 消费者
- `login_test.go:1068-1143` 的 `sameAuditEvent` 字段比较测试

结论：

1. 三组 fixture 消费者均通过真实 `Store.Append` 和真实 migration，不存在绕开 actor CHECK 的替代路径。
2. round-trip 的 anonymous 子例仍同时把 `ActorUserID`、`ActorSystem` 置 nil；默认与 empty-optional 子例使用合法 user actor。修复没有通过统一改成 anonymous 来掩盖 user 分支。
3. `sameAuditEvent` 仍逐字段比较 `ActorSystem`；当前修复没有删除该字段比较或放宽相等条件。
4. 当前范围没有新增测试盲区：修改前的 `ActorUser + ActorSystem != nil` 只能稳定触发约束失败，不能构成任何合法 system actor 覆盖；修改后恢复了 user actor 真实数据库路径。system actor 正向分支若需单独覆盖，应由 audit 约束专门测试承接，不是本轮修复的阻断项。
5. integration 环境缺 DSN或数据库 major 非 18 时会明确失败，不会 Skip。Reviewer 从 Windows User scope 加载但未输出 DSN，目标三测试 `-count=3` 在 PostgreSQL 18 上 PASS。

## 4. Findings

### CRITICAL

无。

### HIGH

无。

### MEDIUM

无。

### LOW

无。

## 5. Finding 与 Gate 状态

- 本轮 fixture 修复：**PASS**
- `IDENTITY-S3.2-F005-R01-F001`：继续保持 **CLOSED**
- 本轮新增 finding：**0**
- r02 的“PostgreSQL 18 尚未真实执行”限制：对本报告所列三组 F005 事务矩阵已解除；Reviewer 独立执行目标测试三次 PASS，Lead 另有 identity+audit integration 与 integration race PASS 证据。
- `IDENTITY-S3.2-REV-F005`：仍为 **OPEN / 受限**
- S3.2 G4-task：**FAIL**
- B4：**不得关闭**

F005 不能关闭的剩余原因不是本轮 fixture：production composition 仍未交付，登录/受保护路由尚无 production logger、metrics、observer、failure auditor、正数有限预算与取消能力的完整装配；Q13 backend、具体预算、阈值、dashboard、paging/runbook 和值守入口仍是已接受的上游待决定项。

## 6. 实际读取与命令

完整读取：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/coding-standards.md`
- `.codex/guidance/no-hardcode-policy.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`
- `.codex/profiles/go-service/**`
- r01、r02
- `internal/identity/login_integration_test.go`
- `internal/audit/store.go`
- `migrations/0001_shared.sql`
- 必要直接消费者 `internal/identity/login.go`、`login_test.go` 与 `internal/audit/store_integration_test.go`

实际命令与结果：

```text
git branch --show-current
=> main

git rev-parse HEAD
=> 6da8abbceb0fa197364810a92a8fa2d97b38ddca

Get-FileHash -Algorithm SHA256 <固定对象>
=> 开始时全部匹配

Get-Content -Raw / Select-String / 编号行读取
=> 完整读取规则、报告、目标测试、Store、migration 与有界消费者

内存反向恢复 r02 fixture 后计算 SHA-256
=> B76ADABD7F7A6146FDC9C666699714CF53EA1485C9E916D4F74E61D46C8337FC
=> 与 r02 固定哈希一致

Windows User scope HAN_TEST_POSTGRES_DSN presence check
=> SET；未输出值

go test -tags=integration ./internal/identity \
  -run '^(TestLoginCommitUnknownVerifiesActualPreparedAuditState|TestLoginRollbackUnknownAfterAppendVerifiesActualPreparedAuditState|TestLoginPreparedAuditPostgresRoundTripPreservesExactImmutableFields)$' \
  -count=3
=> PASS
```

## 7. 未验证风险与交回建议

- Reviewer 本轮独立重跑的是三组目标 PostgreSQL 18 测试；全 identity+audit integration、integration race 和全本地 Gate 采用 Lead 的现场证据，未重复扩大执行。
- 未执行真实 E2E，未验证尚不存在的 production composition 或真实 logger/metrics backend。
- system actor 的正向约束分支不是本 fixture 的登录测试目标；当前没有因本次修复丢失既有有效覆盖。若 audit owner 后续要求三种 actor 分支独立矩阵，应作为 audit 约束测试增强单独评估，不阻断本次 fixture 修复。
- 建议 Lead 接受本轮 fixture 修复 PASS 与 PostgreSQL 18 目标矩阵 PASS，但保持 `IDENTITY-S3.2-REV-F005`、S3.2 和 B4 未关闭，继续等待 production composition/Q13 闭环。
