# identity-auth design r42 / S10.3 定点复审 R02

- 日期：2026-10-24
- Reviewer：Gibbs（非作者独立 Reviewer）
- objectType：design；test-plan 与 Tasks/Plan 仅作为 F001 影响映射条件核对
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 复审对象：`IDENTITY-DESIGN-R41-S10.3-R01-F001`
- 结论：**Design Review Gate: PASS**

本报告只判断 r42 是否消除原 F001 的设计与阶段切分缺口。PASS 不表示代码已实现、测试已执行、S10.3 已关闭、G0 已自动解除或 production ready。

## 1. 固定对象与哈希

| 对象 | revision / 用途 | SHA256 | 开始 | 结束 |
|---|---|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `identity-auth-design-ir01-r42`，主复审对象 | `CEF17810A42DECF35F0B9A0585877B6FDFEEF77557E2A314D6BF536D2940D176` | MATCH | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/stories.md` | `identity-auth-stories-ir01-r16`，未修改行为基线 | `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3` | MATCH | MATCH |
| `.add/project/knowledge/design-workspace/identity-auth/test-plan.md` | `identity-auth-test-plan-ir01-r19`，验证影响映射 | `0381614EA06B34756C430989D04BAF0F8A55C5FAB24C713711535EB1D78AD0DE` | MATCH | MATCH |
| `.add/project/knowledge/tasks/tasks.md` | S10.3/S10.4 任务所有权 | `6E58D74AD88870A76E274F5ACC66CBB38240D1A8E73ADC28D99AD2BC154C5F97` | MATCH | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | B6C DAG、批次与文件所有权 | `AF5FCDF470168AFAD5B42F72522549FB7D5B550E79E414F1DC517512EA4A774D` | MATCH | MATCH |
| `.spec/wip/identity-auth/review-design-r41-s10.3-r01.md` | 原 R01 / F001 | `83687AEA52489869DA0DB4945C820EC0AE6E0E167460212E6D2A57FF3CAB2C35` | MATCH | MATCH |

开始时目标报告路径不存在。适用日期按上级指令固定为 2026-10-24。

## 2. 两轮实际检查

### Round 1：修订对象及影响映射直接检查

1. 完整复核 Reviewer Role、`add-code-review` design 分支、评审清单、抽象原则、设计写作规则和产品安全红线。
2. 对照原 R01 F001 的触发、影响、修复建议和复审条件，检查 r42 增量、三消费者 Gate 合同、生产构造与阶段交接。
3. 检查 test-plan r19 的 `I-PRD-11/12/13/15.CW-*` 是否覆盖旧入口、缺依赖 fail-fast、共享实例、三消费者矩阵、13 个调用面、Manifest 七项和 S10.3→S10.4 交接。
4. 检查 Tasks/Plan 中 S10.3 精确 13 文件、S10.4 精确六文件、关闭条件、状态、requires 与 DAG；未把当前 OPEN finding 或未实施内容写成完成。

### Round 2：真实消费者与上游合同有界深审

只读追踪以下真实定义和调用点：

- `internal/session/store.go`、`store_test.go`
- `internal/identity/password_reset_store.go`、`password_reset_store_test.go`
- `internal/identity/login.go`、`login_test.go`、`login_integration_test.go`
- `internal/identity/service.go`
- `internal/identity/bootstrap.go`、`bootstrap_test.go`
- `internal/identity/administrator_create_test.go`
- `cmd/han-server/main.go`、`main_test.go`
- `internal/platform/http_security.go` 的稳定 `PrincipalResolver` 消费边界

通过全仓定向搜索核对 `session.NewStore`、`NewLoginService`、`NewPasswordChangeService` 与 `ResolvePrincipal`。真实 constructor/call-site 与 r42 的 13 文件清单一致；其他 `ResolvePrincipal` 命中是接口消费或测试 stub，不需要进入本次写集。另核 accepted ADR-0023、identity boundary、单向依赖、r40 lineage/current capability 分类和 caller-owned transaction/commit barrier 未被改写。

## 3. 原 finding 定点复审

### `IDENTITY-DESIGN-R41-S10.3-R01-F001` — HIGH — **CLOSED**

#### A. 唯一安全过渡：CLOSED

r42 明确 `session.NewStore(StoreDependencies)` 为唯一 Store constructor，transaction boundary 与 Gate 均为必填；Login/PasswordChange dependencies 同样强制非 nil Gate。旧 constructor、optional/no-op/allow-all/deny-all、第二 authority 和仅测试可见的授权入口均被设计禁止。

production root 在 S10.3 内只构造一个：

```go
passwordResetGate := &identity.PasswordResetStore{
    Idempotency: idempotency.Store{},
}
```

并将同一指针注入 Session、Login、PasswordChange。13 文件必须作为同一可编译、可部署 revision 交付，明确禁止“包内 constructor 已收紧但 production root 尚未迁移”的中间状态（`design.md:57, 3518-3545`）。

#### B. 精确 13 文件与依赖方向：CLOSED

r42 的 13 文件逐一覆盖 Store/Gate 实现、三个 consumer constructor、全部真实直接 call-site、production root 与相应测试（`design.md:3518-3532`）。定向搜索未发现需要第 14 个文件才能迁移的真实 constructor 调用。

中性接口仍由 `internal/session` 声明，唯一 durable SQL/classifier 实现仍由 `identity.PasswordResetStore` 拥有，保持 `identity → session` 单向依赖；消费者不得复制 r40 lineage/current capability 分类。

#### C. fail-fast、错误语义与 Session commit barrier：CLOSED

r42 要求：

- 缺 Gate、缺 transaction boundary 或依赖构造失败，在 credential verify、业务 DB、listener、routes 与 Serve 前失败；
- Login/PasswordChange 在 caller-owned transaction 锁定 User 后调用 Gate；
- Session 按 token 预读→begin→User lock→Gate→Session/credential 重读→必要 revoke→commit→Principal 执行；
- 预读不授权，rollback、commit error/unknown、Gate/CSRF/lineage 错误均不得物化 Principal；
- 三消费者继续映射既有公共错误，不新增 REST 或错误码。

因此原 R01 中“严格 constructor 破坏 production、保留旧 constructor 又形成旁路”的三选一矛盾已消失。

#### D. test-plan r19：CLOSED

`test-plan.md:402-427, 625-628, 759-766` 已覆盖：

- legacy/旧签名与缺 Gate 入口不可授权；
- nil Gate、nil transaction boundary 和构造失败的零副作用 fail-fast；
- Login、PasswordChange、Session 的 clear/closed/query/commit 矩阵；
- Session 仅在 commit 成功后返回 Principal；
- production construction 恰好一次且三个消费者观察同一实例；
- 三处未导出 deterministic fake，不进入 production authority；
- 13/13 直接调用面、Manifest 七项和 S10.3→S10.4 交接。

状态仍为 `planned-not-executed`，没有把设计候选或本次复审外推成执行通过。

#### E. Tasks/Plan 所有权与 DAG：CLOSED

Tasks 的 S10.3 卡（`tasks.md:924-955`）精确登记 13 文件、TDD 失败证据、专用命令、Manifest 七项、无中间不可部署 revision和 Lead/非作者 Review 关闭条件。S10.4（`tasks.md:957-981`）只保留：

- `password_reset_worker.go/_test.go`
- `password_reset_operator.go/_test.go`
- `cmd/han-server/lifecycle.go/_test.go`

其不再拥有 `main.go/_test.go` 或 Gate construction/injection/wiring。

Plan 保持 `S10.2 → S10.3 → {S10.1,S10.4}`，S10.4 requires S10.3；B6D 仍等待四卡关闭、B6C-3 batch Gate PASS 和代码 F002 CLOSED。任务总数、三态和批次唯一性未因本次对账改变。

#### F. 真实消费者覆盖：CLOSED

现役代码尚未实施 r42，但真实调用面验证了设计清单的完整性：

- `session.NewStore`：Session 包测试、Login integration、production main；
- `NewLoginService`：identity 测试/integration、production main/main test；
- `NewPasswordChangeService`：bootstrap/administrator 测试与 production main；
- production root 当前位于 `cmd/han-server/main.go`。

`internal/platform/http_security.go` 只消费稳定的 `PrincipalResolver`，无需修改。现状未实现不构成 design finding；关键是 r42 已覆盖迁移这些调用点所需的完整且最小写集。

## 4. 新 findings

**无。**

未发现新的 HIGH、MEDIUM 或 LOW design finding；未发现需要新增 migration/schema、依赖、公共 REST/错误码、ADR、boundary 或第 14 文件的证据。

## 5. 待决定与非阻断建议

### 待决定

无新的设计决定。r42 只在 accepted ADR-0023 方向内修正阶段和文件所有权，不需要重开 ADR。

### 非阻断建议

后续代码评审应特别核对 Go interface typed-nil、同一实例身份和构造失败顺序，防止实现形式满足非 nil 语法却恢复旁路；这属于实现验证点，不构成本轮设计缺口。

## 6. Gate

### ADR consistency Gate

**PASS（未重开 ADR）**。

r42 未改变 accepted ADR-0023 的唯一 durable Gate、三消费者强制消费和 fail-closed 方向，只修正 S10.3/S10.4 的可实施阶段切分。

### Design Review Gate

**PASS**。

原 F001 的全部复审条件均已满足：唯一安全过渡、精确完整写集、production root 同实例强制注入、全部直接 call-site、可执行验证映射和 S10.3/S10.4 所有权均形成闭环，且不存在依赖旧入口或文档禁令维持安全的不可部署中间 revision。

本 Gate 只接受 design r42 作为后续 Lead G0 判断和实施输入，不自动解除 G0，也不替代后续代码 Review、Lead task-level Gate 或真实验证。

## 7. 状态边界

- S10.2：保持 CLOSED。
- S10.3：保持 active / G0 blocked；本报告不分配实施者、不启动实现、不关闭任务。
- S10.1：保持 implementation stopped。
- `IDENTITY-S10.1-REV-R01-F002`：保持 OPEN。
- S10.4、S11.1、B6D：保持 not started，不得因本次 PASS 启动。
- test-plan r19：保持 `planned-not-executed`。
- 本报告不表示 production readiness。

## 8. 未验证风险与限制

1. 未实施 r42，未验证具体 Go constructor、interface typed-nil、错误链、SQL、pgx transaction、rollback/commit unknown 或并发锁序。
2. 未运行 Go 测试或 Manifest 七项；报告只确认这些关闭证据已被正确规划。
3. 未运行真实 PostgreSQL、SMTP、Host、部署或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
4. 未验证真实 listener/routes/Serve 的运行时 fail-fast，仅核设计、测试规划和真实构造面能够形成该验证。
5. 结论仅适用于表列固定哈希；后续若需要第 14 文件、第二 authority、改变 `identity → session` 方向、公共合同或 transaction boundary，应停止实施并返回设计/规划重新评审。

## 9. 实际命令

本轮只执行只读静态检查和报告校验：

- `git branch --show-current`
- `git rev-parse HEAD`
- `Test-Path -LiteralPath .spec/wip/identity-auth/review-design-r42-s10.3-r02.md`
- `Get-FileHash -Algorithm SHA256`（六个固定对象，写前/写后）
- `Get-Content`（Role、Skill、Rules、固定对象、原 R01 与有界消费者）
- `rg -n --encoding utf-8`（constructor、Gate、transaction、Principal、任务写集、DAG 与真实 call-site）
- `git diff --no-index --check -- /dev/null .spec/wip/identity-auth/review-design-r42-s10.3-r02.md`

未执行 Go、PostgreSQL、SMTP、部署或 E2E 命令；未执行 stage、commit、push、切分支或清理。
