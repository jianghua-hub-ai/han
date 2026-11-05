# S6.2 PG + joint 独立 CodeReview R01

**jointCodeReviewGate：FAIL / CHANGES_REQUIRED。AM-S62-JR01-F001：HIGH / OPEN。** 本轮按 Lead 最新指令，在独立确认真实 PG 阻断后收口，交作者定向返工；不等待未知 race 结果，不发布 Ready。

## 固定对象

objectType `code`；工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，已实核。PG 原作者 James、其余实现作者 Turing；本执行者非作者。十三个代码文件 SHA256 全部实核匹配：

| 文件 | SHA256 |
|---|---|
| internal/identity/article_disposition.go | EB053611CE9346A9A51C0485DD9AE1003EA99C57AF0A3842A4621D67ECE04593 |
| internal/identity/article_disposition_test.go | A92E8C4D65C76361094B472F35BEDEAB77F15C2FC5E0ECA54854BCDDC640EDF4 |
| internal/article/command_service.go | 585BB51D19B3FCD7BFDC9C84465D2BB5D24F41BD8285553FC20BC10B161B4D68 |
| internal/article/repository.go | E0C2430A981A5F90E84F197159158CC4A218A4FB399C74E90C203B19AC0FE078 |
| internal/article/deletion_finalizer.go | 12F9D775F115732226908C0440D5F30D6FAF37A17CC19FC2A2CE147216F869C4 |
| internal/article/deletion_test.go | 6D04CD35F0049E64FFEC6954B53EF0AB75256BF97C156E331BB5B019E87EA9C6 |
| internal/article/query_service.go | 9674130673CD05E8FA864B77528FDBF12C9AB8305AE6AD2EF7E2A85DB9136E7E |
| internal/article/handler.go | A0921B0178551A8EA51772ECF9823B50AA693DD3B450FDD347257B6F77CCC7E8 |
| internal/article/orphan_disposition_test.go | 1F8777D835D8C4363F67F498D5280C6EDA7BDAA3ED8F71B15AFB4E1B411338F5 |
| cmd/han-server/main.go | 4B9D83BA52E322E9E7A430F18A59DAA0FC651D4D4B21BEAFBA825E7E991A7569 |
| cmd/han-server/main_test.go | 1FB995EA3DE93A3F9AC2CC575D4AC86A9084592742EEC0C86F1ACF4E9CF28C9E |
| internal/article/deletion_coordinator.go | 6EEA01FB40FBD6B237B4E02600122708404EE1AF59626971D4CE40D72DB98481 |
| test/integration/identity/orphan_disposition_test.go | E21A2A4BCA4389BFAE466DF0BEC90EE8976E6CA6248D515D6268998CE9CDF9DC |

新增 PG 文件 79,571 bytes / 1,887 行。设计 `article-management-design-ir01-r16` SHA256 `CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC`；test-plan r14 SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C`，均实核。复用既有①R02、②R01、③root R01 局部评审，保留历史结论；本报告不追改它们。③报告已接纳 SHA256 `2D7E0B86C179C2C88F600CBFABED2ED73F08FD477FEF61D7381B359D296B85F8`。

## 两轮实际检查与证据边界

复用本会话已读 Reviewer Role、add-code-review、当前 delivery Workflow 与适用 coding/go-service、安全、抽象和可观测性规则。

Round 1：核十三个代码 hash、PG 十组入口与 §4.5.1 映射；定点读取 PG harness、N001 组及 source-free replay 用例全文、FileJournal Host 分支；沿 `ReplayAllTombstones → convergeAndCount → rebuildOperationFromTombstone → claim/T2 → applyDeletionCompletion → deletionAuditEvent` 核失败路径。

Round 2：独立核真实 `audit.Store.Append`、`0001_shared.sql` actor 约束、设计 §3.4.5 第11条及无来源保守边界；核 Article absent 时 native proof/Identity 清理不被调用、当前动作读取白名单与事务回滚/重入路径；只读解析 Lead 完整普通 PG JSONL。没有独立重跑 PG、没有调用会 DROP 的共享 harness。

因最新指令要求已确认阻断后立即交回，本轮**未完成新增 PG 文件全文及十组断言充分性的独立审核**；其余十二文件复用局部评审，不宣称新增完整 joint PASS。以下实跑统计证明执行结果，不替代尚未完成的 source review。

## Finding

### AM-S62-JR01-F001 — HIGH / OPEN：无 actor 重建 operation 无法幂等完成无来源 replay

- **位置**：`internal/article/deletion_finalizer.go:701` 固定 `ActorKind: audit.ActorUser`；`:297` 从 tombstone 重建 ordinary operation 时没有 ActorUserID/ActorRole/source；`:642` 所有完成路径均使用该审计构造器。触发用例 `test/integration/identity/orphan_disposition_test.go:1864`。
- **违反依据**：design r16 §3.4.5 第11条（`design.md:1963`）明确 Article 已不存在时幂等完成 operation，缺失原审计时追加 `article.deletion.replay` 系统审计。test-plan §4.5.1 N001（`:283`）将无来源 absent replay 保守分支列入本卡。`migrations/0001_shared.sql:31` 要求 user actor 的 actor_user_id 非空。
- **触发**：可信 tombstone 存在，恢复点缺失 operation 且 Article 已 absent。重建的 ordinary/source=NULL/actor=NULL operation 经 claim 进入 T2，DELETE 得到 already absent；随后尝试写入 user actor 审计。
- **实际影响**：PG CHECK 拒绝写入（SQLSTATE `23514` / `audit_events_check`），整个 T2 rollback，未写 completed/完成审计，之前独立提交的 claim 留在 `finalizing`。再次收敛仍构造相同非法审计，不能靠 lease/backoff自动修复，阻断本卡约定的 replay 收敛。当前没有证据表明误清 retained_email：本例 cleanup_calls=0。
- **独立核对的真实证据**：普通 PG 唯一失败叶子为 `TestOrphanDispositionPostgres/08_N001ContactsAndReplay/absent_source_free_replay_does_not_guess`；日志包含 `source_free_replay_red SQLSTATE=23514 status=finalizing actor_absent=true source_absent=true cleanup_calls=0 completion_audits=0`，下一断言失败于 `append deletion audit: insert audit event: ... audit_events_check`。源码与真实 SQL 约束一致，不是 nullable 猜测、mock 结论或 fixture 应补 actor 的问题。
- **最小兼容建议**：保留 tombstone/operation/lease 的现有可信绑定；为无 actor、Article 已 absent 的恢复完成路径使用既有 `ActorSystem` 和精确动作 `article.deletion.replay`，设置非空稳定 ActorSystem 标识、ActorUserID 保持 NULL。该审计和 operation.completed/audit_event_id 应在同一 T2 提交。正常在线有 actor 的删除继续原审计语义，不猜 actor/source，不改变 kind/source、schema、Journal 或 API。
- **直接消费者核对**：`repository.go:321` 的 replay 审计读取已允许 `article.deletion.replay`；Identity 校验 `article_disposition.go:505` 也识别该动作。本例 `lockDeletionDisposition:561` 因 absent/non-orphan 返回 nil proof，不取得清理资格，故修复本例无需放宽 native proof 或 Identity 来源校验。若扩大到其他恢复前像，应另核 native proof 的动作约束，不能顺带宽松化。
- **复审条件**：保留本用例成功预期，禁止 skip、补造 actor/source 或改 fixture 规避。补局部 RED→GREEN：无 actor absent replay 的系统动作/目标/trace 与 completed.audit_event_id 一致；再入不重复完成审计；retained_email/email Identifier 保留、cleanup_calls=0；审计失败全部 T2 回滚；普通有 actor 删除动作不回退。提交冻结修订后，重新跑该真实 PG 用例及相关矩阵、普通/race 与适用 Manifest，复审同一 finding。

## 验证结果、限制与交回

只读核对日志：`C:\Users\J00010~1\AppData\Local\Temp\han-s62-4d7a5257f49745bc82ad754d1798d845\pg-lead-normal-d1d97c04e78c4882a08b995e93857d19.jsonl`，SHA256 `9A6D5F2E807D2F77E25F7610990BF56A3293FEA55F957626D6C4FA23470179F8`。顶层矩阵 143.92s，组01～07、09～10 PASS，组08 FAIL；178 个 pass 节点、0 skip，唯一失败叶子如上（父组/顶层/package fail 不计为额外缺陷）。日志实有163个 retained_schema 记录，未清理。

Lead 报告同修订 Manifest 七项 PASS；本评审未独立重跑，该结果不能覆盖真实 PG FAIL。race 未取得有效结果，不引用 James 断线前未知结果，也不声称 race PASS。

PG harness 使用真实 PG18、完整迁移、已提交隔离 schema 和每连接 search_path，注册 pool.Close 而未注册 DROP。矩阵允许显式 Journal fixture；读取的 FileJournal 用例对 unsupported 返回 fail-closed 分支并声明未验 fsync，没有 t.Skip。本次未重新执行 Windows constructor，也未验证 Linux Host、旧已落盘纳秒 Journal 兼容、无来源旧备份完整恢复；这些既有独立 Gate 继续保留，不扩大成本卡新增前置。

唯一写入为本报告；未修改代码、设计、测试计划、Tasks/Plan、既有报告或 Git，未清理、再委派。已交回 **F001 OPEN / joint CHANGES_REQUIRED**，报告完成后停止写入，等待定向返工修订。
