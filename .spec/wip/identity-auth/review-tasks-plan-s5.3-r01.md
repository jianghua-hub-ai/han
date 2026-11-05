# B7 / S5.3 G0 最小 R 对账 Planning Review R01

- 日期：2026-10-25
- 对象类型：design / planning contract
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 规划作者：Hilbert `01a17171-261c-7461-8562-256d89d28e80`（已关闭）
- 评审方式：非作者、只读评审；除本报告外未修改被评审对象、代码或其他报告

## 1. 固定对象与依据

| 对象 | 固定 SHA256 | 复核 |
| --- | --- | --- |
| `.add/project/knowledge/tasks/tasks.md` | `02907F21E5B7C0A7555348975B427350CD62D8FDDB2474C45B9EBD9A95A49525` | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `4C286AFA351EBA7CAC5EE0882E2A4160CE76D48B080B3C9CD94E60DD35E1C271` | MATCH |

上游 authority 复核：

- identity design r44：`50A83A81E4F521A0740BC220E5E382D6BFEFE18CC316E869A47019C45B1FEE23`
- stories r16：`057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3`
- test-plan r20：`F9AD6CE09F5F881534E3B592E863C38FC16D593F3AA82BE9352393A8280A69CB`

方法与规则实际读取：Reviewer Role、`add-code-review`、`code-review-checklist`、`abstraction-principles`；Task-Planner Role、`add-task-decompose`、`add-batch-plan`；`design-doc-writing-style`、`decisions-guide`。对象依据读取 design r40/r44、test-plan r18/r20、S5.3/S10.1/S11.1 task，以及本次五文件当前代码。

## 2. Round 1：Tasks / Plan 直接合同检查

实际检查范围：

1. **Tasks / Plan 一致性**：Tasks 的现役 S5.3、S10.1、S11.1 合同（`tasks.md:739-758`、`885-916`、`1032-1059`）与 Plan 当前活动、B6C-3/B7 Context Pack、DAG、Gate 和变更记录（`ir-01-content-community-batches.md:9-13`、`60-75`、`650-657`、`688-706`、`815-826`、`976`）。
2. **四标签**：Story 级 S10/S11 各且仅有一个“需改”标签；能力切片矩阵明确只有共享 coordinator / Mailer classification 为“需改”，并把 S11.1 定位为消费者而非第二修复 authority（`tasks.md:95-123`）。
3. **三态**：57 个任务中 26 completed / 9 active / 22 todo；九个 active 条目与 Tasks checkbox 实际集合一致：S3.3、S3.5、S5.3、S9.1、S10.1、article S2.1/S4.2/S4.3/S4.5。
4. **requires / DAG**：S5.3 已直接 requires S10.1 与 S11.1（`tasks.md:751`）；S11.1 保持 CLOSED，S10.1 重开后由既有依赖自然形成 `S10.1 classification repair → S5.3 G0 re-entry`，无需新增 task 或改 requires；未发现环。
5. **写集与命令**：S10.1 五文件覆盖 shared taxonomy、SMTP adapter、adapter 测试、唯一 coordinator 和 coordinator 测试（`tasks.md:890-909`）；S5.3 仍只拥有三个 local real-protocol 测试文件（`tasks.md:744-750`），生产返工未混入专项卡。
6. **停止 / 恢复 Gate**：第六文件、新依赖、Mailer 签名、公共 REST/schema/error、migration、architecture/ADR 或第二 authority 均触发停止；S10.1 必须完成五文件 TDD、非作者 Code Review、Lead 同 revision 哈希/status/diff/命令/越界 Gate并再次关闭，之后还须核 staged、未知写者和 S5.3 三文件零写入，再重新执行 G0（`tasks.md:913-914`、`755`；Plan `706`、`826`）。
7. **历史与现役状态分离**：旧 S10.1/B6C-3 PASS 明确只作旧 revision 历史证据；现役 S10.1 是 `[-]`，S11.1 仍为 `[x]`，S5.3 为 `[-]` 但 `implementation paused`。未发现历史 closure 被改写为当前 PASS，或当前 active 反向污染历史快照。

Round 1 结论：Tasks 与 Plan 对方案 B、任务状态、owner 顺序、写集、命令和 Gate 的表达一致，未发现阻断缺陷。

## 3. Round 2：真实 authority 与消费者检查

实际检查范围：

1. **Mailer taxonomy 可实现性**：
   - `Mailer.Send(context.Context, Message) error` 当前签名位于 `internal/mailer/mailer.go:41-44`。
   - SMTP adapter 的协议阶段集中在 `internal/mailer/smtp.go:78-145`；当前已区分 DATA 明确 accepted 后的成功边界（`smtp.go:142-145`）。
   - 在不改签名的前提下，可在 mailer package 增加稳定 sentinel/typed wrapper，由 SMTP adapter仅对协议层可证明未受理的 reply 包装，并由消费者使用 `errors.Is`；dial/TLS/AUTH/timeout/cancel/EOF/写入中断或 acceptance reply 不可判定仍保持 unknown。该路径不需要错误字符串匹配，也不要求新依赖或第六文件。
2. **唯一 PasswordReset coordinator / Store authority**：
   - 当前所有 password-reset `Mailer.Send` error 均进入 unknown（`internal/identity/password_reset.go:330-339`）。
   - `PasswordResetStore.RecordRejectedDelivery` 已存在（`password_reset.go:633-661`），仓内没有 coordinator 调用路径。
   - 因此缺口位于现有 S10.1 coordinator authority 内；让该 coordinator按 typed classification 调用既有 Store 方法，是最小且不复制状态机的修复。
3. **S11.1 resend 消费链**：
   - production composition 把同一个 `passwordResetCoordinator` 同时注入首次 reset handler 与 resend router（`cmd/han-server/main.go:2236-2289`）。
   - `TemporaryCredentialResendRouter` 的 `password_reset` 分支调用该 coordinator（`internal/identity/temporary_credential_resend.go:53-105`），最终复用 `dispatchPasswordReset`（`temporary_credential_resend.go:248-304`）。
   - 所以 S11.1 无需重开或复制分类逻辑；将首次 reset 与普通 resend 都纳入 S10.1 回归即可验证真实消费者。
4. **其他 Mailer 消费者边界**：
   - `administrator_create.go` 与 `temporary_credential_resend.go` 的 new-administrator 分支也消费 `Mailer.Send`，但仍把任何非 nil error 当既有失败路径；新增可 `errors.Is` 识别的分类不会要求修改这些消费者。
   - S5.3 只负责 local real-protocol SMTP/TLS 与残余窗口证据，不能诚实地在三个测试文件中修复生产 coordinator，故不应扩 S5.3。
5. **上游语义充分性**：
   - design r40 已固定明确未受理只失效 new candidate、accepted 进入 settlement、unknown 保持 unresolved Gate（现役 design 对应 `design.md:2076-2157`、`2224-2241`）。
   - test-plan r18 I-PRD-07 已固定首次与 resend 的 accepted / 明确未受理 / unknown 及结果持久化矩阵（现役 test-plan 对应 `test-plan.md:400-405`）。
   - 本次只细化 adapter 到既有三态合同的 typed classification，不改变用户可见合同、状态模型或架构边界，因此不要求设计先行；现有停止条件足以拦截超出该判断的实现。

Round 2 结论：

- 仅重开 S10.1 比扩 S5.3、新增 task 或重开 S11.1 更符合唯一 authority 与最小 R 对账原则。
- 五文件是当前最小且充分的闭环写集。
- 保持 `Mailer.Send` 签名并禁止字符串匹配，可以形成稳定 typed classification；具体协议判定仍须由后续代码 Review逐分支核证。
- Kant 单一 owner 串行先修 S10.1、再恢复 S5.3，加上恢复 Gate，可避免五文件与三测试文件的并行冲突。

## 4. Findings

**无 finding。**

未发现需要分配 `IDENTITY-S5.3-PLAN-R01-Fnnn` 的正确性、安全、契约、架构边界、状态、DAG、所有权或交付完整性缺陷。

## 5. 待决定与非阻断建议

- 待决定：无。
- 非阻断建议：后续代码 Review 应以 SMTP 协议阶段和结构化错误类型为证据，逐项确认只有“明确 reply 且可证明未受理”进入 rejected；任何网络/上下文/写入/接受回复不确定性继续进入 unknown。该项已被现有 TDD/停止条件覆盖，不构成本轮 planning finding。

## 6. 未验证风险

- 本轮没有 implementation revision，未运行 Go 测试；Planning PASS 不证明未来 typed classification 实现正确。
- 未运行真实 SMTP、PostgreSQL、Browser、部署或 E2E；未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未验证外部 provider 对各 SMTP reply 的实际行为；S5.3 后续仅覆盖 local real-protocol，approved-provider/E2E 仍属独立验证。
- 未验证未来五文件最终 SHA、测试结果、race、Manifest 七项或越界情况；这些仍是 S10.1 再关闭和 S5.3 恢复的强制 Gate。

## 7. Planning Review Gate

**Planning Review Gate: PASS**

本结论只绑定第 1 节两个固定 SHA256 对象和上述只读 authority/consumer 证据，不授权实施、不替代 Lead 接纳，也不外推代码 Review、真实 SMTP/PG、部署、E2E 或 production readiness。

复审条件：

1. 固定 Tasks 或 Plan 任一 SHA256 变化；
2. 方案从“仅重开 S10.1”变化为扩 S5.3、新增 task、重开 S11.1或修改 requires/DAG；
3. 五文件、`Mailer.Send` 签名、公共 REST/schema/error、依赖、migration、architecture/ADR、authority/root/global 任一变化；
4. owner 不再串行、出现未知写者或 S5.3 三测试文件在恢复 Gate 前发生写入；
5. 后续实现无法仅凭稳定 typed classification 区分 proved non-acceptance 与 unknown。
