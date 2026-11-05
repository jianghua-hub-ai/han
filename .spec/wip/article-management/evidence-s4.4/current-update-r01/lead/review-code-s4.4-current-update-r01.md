# S4.4 CurrentUpdateRollback 独立 CodeReview R01

**结论：有界 PASS；两轮未发现 material finding，缺陷 findings 为 0。** 本结论只针对下列固定 code object 及三个 leaf 的本轮证据，不关闭 S4.4、Issue 或真实恢复门禁，也不替代 Lead/人的成果接收。

## 对象、授权与依据

- objectType：code；工作区 `D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 唯一被审对象：`test/integration/recovery/replay_evidence_test.go`；基线 `BE6C689273783CFCB4939EE987397E09DB175B30506D9C0E49881FA8CE83A925` → 目标 `CF1B07C58F5A7DDCDEA256484CFC9E5F1C0E595B0A1A13B3B1B49DDB410E9048`。原两 leaf 与其既有 helper 主体逐字节一致；本次增加 CurrentUpdateRollback、同文件辅助实现及必要 imports/top 注册/注释。
- Lead 完整185输入清单：`C:\Users\J00010385\AppData\Local\Temp\han-s44-update-lead-jgbec4f1\source.after.flat.json`，SHA256 `4A7687DEF84AC4DF78FAA6747A157E00E11911B9948708D1FC5986885EBEB4E5`；实际执行 cwd 为 `C:\Users\J00010385\AppData\Local\Temp\han-s44-update-lead-jgbec4f1\fixed-inputs185`。其余184项、live/snapshot完整集合和每项SHA在每个child前后全部匹配；Tasks/Plan/design/test-plan/manifest五项metadata在每个child前后匹配。准备与结束还核对原33 knowledge均未变。
- 启动依据为 Lead 本轮明确 `REVIEW START + LOCALPG START`；binding `C:\Users\J00010385\AppData\Local\Temp\han-s44-update-lead-jgbec4f1\review-start-binding.json`，SHA256 `429F3EFAB753F6BBDF36C0F603C693E84E6BB22202AA00080B6E6F861962789F`。Euler为作者，已completed/stopped；本会话是非作者Reviewer，未委派、未修改测试/生产/Tasks/Plan/design/Issue/配置，唯一仓库写入为本报告。
- 业务依据：原 Tasks S4.4（1512–1540）、design r22 §3.4.7（尤其2291–2299的同事务绑定与2330的回滚重试）、test-plan r20 A-REC-07及§4.8.1。design SHA256 `8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E`；test-plan SHA256 `DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA`。
- Bohr只读合同 `C:\Users\J00010385\AppData\Local\Temp\han-s44-current-update-plan-bb5dfad71bbb4eecb53ededfc6065b39\report.md`，SHA256 `2582EA87DCF98B0F81DE2AA93973EABB0B60AAA0D14E5AAAFD7AD886A5CAA3F0`；该报告是合同依据，并非运行/PASS证据。
- 本角色完整Role、add-code-review及delivery workflow、review checklist、abstraction、coding/no-hardcode、安全红线和已启用go-service相关规则均已实际读取并核同版；详见Temp `readiness.json` 与 `review-start-check.json`。没有以父会话摘要或发现列表替代读取。

## 两轮实际检查

**Round 1：固定diff与直接风险。** 独立比对真实新旧字节及Lead diff，核本文件完整既有主体与新增代码（入口34；新增487；安装530；故障SQL547；错误链587；完整owner612；回滚625；Ready653）。核零历史/no-current前置、真实Source 0/3候选、同事务成功INSERT可见后才抛23514、前置不足走P0001、errors.As与封闭witness字段、错误时零evidence返回、同一request/owner/input仅取新ref重试。DDL标识符经pgx.Identifier，ref经PG format `%L`；函数为schema本地默认invoker，无生产seam、SECURITY DEFINER、认证/权限变更或额外DROP/disable。公共DTO不输出token/内部错误；内存完整owner比较失败只给固定原因。

**Round 2：沿真实生产者/消费者有界核对。** 沿实际fixture `article_replay_test.go` 的配置/全fallback、schema创建/9迁移、Finalizer与Source/StoreProbe、Begin/request、七表镜像、资源关闭、Ready/SQL/Facts oracle，追至 `internal/article/replay_evidence.go` Publish317→select/reuse/INSERT414/778→setCurrent402→Commit352，及ReadCurrent/LoadAttempt/binding校验；核 `deletion_replay.go:226` 公共ReadStatus、共享 `internal/recovery/replay_status_v1.go:276` canonical，以及0003/0009的首绑、append-only、current FK与attempt约束。新ref谓词在trigger保持enabled时允许重试；零目标历史排除了用旧history冒充本次INSERT。真实INSERT后的故障导致七表/xmin/ctid全等，成功后仅允许run/evidence变化，五张业务表全值/xmin/ctid不变。检查继承的公开Journal seam、合成inventory和真实opaque-proof adapter边界，未把测试装配外推为生产联合恢复。

第二轮还亲自核本次普通/race raw事件全集、92payload哈希、两份23514 witness/七表镜像/调用记录、10组Ready20与SQL19字段/Facts10，独立按first16重算canonical、operation/audit摘要及合成inventory摘要。完整owner（含token）、witness PID不同于observer、trigger保持enabled的结论来自固定源码中的已执行断言；未伪造一个不存在的持久token/observerPID快照。未扩成全仓或所有消费者审查。

## 独立执行与原始索引

自有执行根：`C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765`。使用获准的binding suite，没有重跑要求旧BE6快照的prepare。预检只读、无DDL，实际端点literal loopback:5433、PG180006、current_user/current_database与DSN配置一致（本次user/database均postgres），连接已close；主地址与全部fallback均检查，本次fallback_count=0。唯一User-scope DSN一次读取后只在内存和PG child净化env传递，未写CLI/日志/文件或修改User/System/process环境；offline/local-toolchain/GOENV与GOWORK关闭；vet不带DSN。

四个child均实际wait并结束，outer suite工具session37734亦exit0；无首失败、无重跑、无其他integration top或旧autoDROP harness。没有进行全机进程普查。

| mode | PID | UTC开始 → 结束 | 秒 | actualwait | exit |
|---|---:|---|---:|---|---:|
| preflight | 11524 | 2026-10-28T05:08:55.087083+00:00 → 2026-10-28T05:08:56.887173+00:00 | 1.796 | true | 0 |
| normal | 40372 | 2026-10-28T05:08:57.941911+00:00 → 2026-10-28T05:09:04.001428+00:00 | 6.063 | true | 0 |
| race | 31892 | 2026-10-28T05:09:05.314735+00:00 → 2026-10-28T05:09:15.826793+00:00 | 10.516 | true | 0 |
| vet | 50496 | 2026-10-28T05:09:16.676040+00:00 → 2026-10-28T05:09:17.818319+00:00 | 1.140 | true | 0 |

实际argv如下（每个mode的cwd/UTC/PID/wait/exit及stdout、stderr原始SHA均在meta、mode-index中）：

```json
["go", "run", "C:\\Users\\J00010385\\AppData\\Local\\Temp\\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\\endpoint_guard.go"]
["go", "test", "-tags=integration", "./test/integration/recovery", "-run", "^TestArticleReplayEvidencePostgres$", "-count=1", "-timeout=8m", "-json"]
["go", "test", "-race", "-tags=integration", "./test/integration/recovery", "-run", "^TestArticleReplayEvidencePostgres$", "-count=1", "-timeout=8m", "-json"]
["go", "vet", "-tags=integration", "./test/integration/recovery"]
```

普通与race每种都精确只有以下4个run/pass节点：`TestArticleReplayEvidencePostgres`、`/DifferentBackupFirstBinding`、`/PublishRollback`、`/CurrentUpdateRollback`；3个terminal leaf，0fail、0skip、0extra，无no-tests-to-run。tagged vet exit0且stdout/stderr空。预检+两测试+vet四项与作者五child是独立执行，未把作者结果充作Reviewer亲跑。

| mode | 完整mode索引（包含raw/meta/前后守卫及本mode payload） | SHA256 |
|---|---|---|
| preflight | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\preflight\mode-index.json` | `63BA33BC6CE423B55DA713434BBBBA7496FFA5E5575F2727F7369449F9712E09` |
| normal | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\normal\mode-index.json` | `71E1675BC0E580957ABC2F25C720E1C85410A8106B7E4CA8356192B421F78477` |
| race | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\race\mode-index.json` | `84FED68F47105BD112388DFFA396571F7D69A27C9183B60BBD908109F1E0C7AD` |
| vet | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\vet\mode-index.json` | `A0C6A698200846D1F06007080F59A16AB079C962B43C7CF81FA7A9681ED76518` |

`independent-evidence-verification.json` SHA256 `3EBDB2712FD5963656EBD1903C1C641928138E95E986D49E0FD8A5AFF8AEABD0` 是两新leaf的witness、七表回滚、新ref重试、调用序列与10组Ready/SQL/Facts独立重算结果。`review-start-check.json`保存作者原件核对：34个最终索引文件、5child wait/exit/rawSHA、normal/race精确集合、92payload哈希及各child185前后图；作者不承担本报告独立结论。

Lead当前185 Manifest七项报告 `C:\Users\J00010385\AppData\Local\Temp\han-s44-update-lead-jgbec4f1\lead-update185-r01-results.json`，SHA256 `BC88309366DDF1EA214048FA13C1534D56DB7E44E29D21709E10A97D707D3EBB`，已读取并核原序7项exit0、gofmt空、完整185前后一致；这是Lead执行，Reviewer没有重复跑，也没有将默认race冒作integration race。

## 留存与结果证据

六个新schema均完整保留，与作者六个schema集合完全不交叉；normal/race也互不交叉。每份新schema先独立CREATE提交，再执行原0001–0009迁移；迁移内固有DROP仅为新schema演进。没有任何额外DROP、disable trigger、清理、移动、role/GRANT或认证改动。92payload全部retain，精确路径/长度/SHA见mode-index及最终 `final-index.json`。以下不是PG cluster全量盘点。

| mode | leaf | 新schema | retained目录 |
|---|---|---|---|
| normal | DifferentBackupFirstBinding | `han_s44red_26e44803144858ea18172570` | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\normal\retained-temp\han-s44-retained-4133488200` |
| normal | PublishRollback | `han_s44red_a4a2db627898b00d10b8ff80` | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\normal\retained-temp\han-s44-retained-3415506133` |
| normal | CurrentUpdateRollback | `han_s44red_193d5bd1a67e83d3b2873295` | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\normal\retained-temp\han-s44-retained-177849431` |
| race | DifferentBackupFirstBinding | `han_s44red_3b9bde84199e2a03519e7e34` | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\race\retained-temp\han-s44-retained-916534` |
| race | PublishRollback | `han_s44red_b2d3ae24417f5e208852e2e7` | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\race\retained-temp\han-s44-retained-2381591762` |
| race | CurrentUpdateRollback | `han_s44red_84b8b6d7b9746d7e50570d05` | `C:\Users\J00010385\AppData\Local\Temp\han-independent-s44-update-review-38e629d70f714c3cb610e6657b7e9765\race\retained-temp\han-s44-retained-2108997502` |

两个CurrentUpdateRollback均观察到真实PG `23514`及`current_update_after_evidence_insert` witness；失败前evidence为零，失败后七表与基线完整集合/字段/xmin/ctid全等且仍pending；同owner/input新ref成功后恰一evidence，failedRef不存在，current/input_digest/status原子绑定。两次Ready完全相同且SQL不再变；每mode调用序列精确 `Publish, ReadCurrent, Publish, ReadCurrent, ReadCurrent`，ID=2、clock=2、Replay=1、candidate token=1。旧两leaf实际blocking与23505结果也保留通过。NOWAIT证明失败事务锁释放，pool当时仅observer=1；各fixture最终observer释放、cancel到达、acquired=0后pool关闭。

## Findings、限制与复审条件

- 缺陷：CRITICAL/HIGH/MEDIUM/LOW均0；无待关闭finding。
- 待决定：本代码片无新增方向决策；整卡和联合恢复原门禁/人的结果接收继续由Lead与责任人承接。
- 非阻断建议：无新增。
- SQL故障为测试trigger在BEFORE current UPDATE主动抛23514；注入ConstraintName不代表原生产CHECK被触发，也不证明已写出current新tuple、commit outcome unknown、真实网络或PG崩溃。
- 七表像只覆盖articles/deletion_operations/audit_events/users/identity_identifiers/deletion_replay_runs/deletion_replay_evidence；users排除permanent_password_digest、run排除attempt_token，后者另由执行中的完整owner内存比较保护。未声称全DB/cluster全字段审计。
- 仅证明controlled Journal、合成inventory/backup引用与真实PG事务子集。真Unix FileJournal/HMAC/keyring/POSIX、真实旧backup恢复/签发挂载、Begin/Fail其他commit-unknown路径、完整A-REC-07、双namespace/MarkConsumed/global opening、应用ACL/S8.2/Case18、RPO/RTO/retention与真实E2E均未获本次验收；原S4.3决定不构成本片扩大豁免。没有关闭S4.4或Issue。
- 目标SHA、185输入集合/哈希、相关fixture/契约或上游依据改变时，本PASS不自动沿用；由作者在新固定修订上提供适用回归，再由非作者Reviewer复审。任何后续采用/发布/任务关闭仍需实际授权与责任接收。

本报告交回Lead；独立运行已结束，全部证据保留，最终 `STOPPED.json` 与 `final-index.json` 记录结束状态、报告hash和完整索引。
