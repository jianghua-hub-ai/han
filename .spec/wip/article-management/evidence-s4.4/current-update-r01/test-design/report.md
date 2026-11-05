# article-management-S4.4：CurrentUpdateRollback 最小测试实施方案

对象：D:/AID/VSCode/Codex/Han，main，HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca。2026-10-28。

性质：当前测试工程师会话的只读合同核对与实施建议，交回 Lead。未委派、未实施、未运行 Go/list/vet/build、未连接 PG、未读取 DSN 或环境凭据；唯一新增为本 Temp 目录中的个人报告与输入指纹。不是测试结果、独立 CodeReview 或 PASS。工作树有大量现存未提交/未跟踪成果，未回滚、覆盖或清理；HEAD 不等于全部现役文件修订，精确文件哈希见同目录 inputs.json。

## 1. 合同结论与唯一写集

可以在原 test/integration/recovery/replay_evidence_test.go 内新增 TestArticleReplayEvidencePostgres/CurrentUpdateRollback；辅助函数仅在该文件内承。保持既有两个 leaf 的语义与断言，允许把 top 注释中“two”更新为准确说明。不新增 top、Task、路径或 requires，不改设计/测试计划、DAG、生产代码、共享 helper、migration 文件或架构。

依据是现役已有原范围的细化，不是扩充业务合同：

- .add/project/knowledge/tasks/tasks.md:1512 当前原卡仍为 [-]；:1516 精确说明已有 PublishRollback 只覆盖 evidence INSERT 自身 23505，明确尚缺成功 INSERT 后 current UPDATE 失败；:1528–1540 保留原三路径、六 requires 与完整关闭条件。原 replay_evidence_test.go 已列于 :1530。
- .add/project/knowledge/design-workspace/article-management/design.md:2291、:2295、:2299、:2330 要求 exact owner、evidence/current/input/status 同 Ready 事务原子提交，回滚不发布证据，未漂移时存活候选可重试；:2620 的 A-REC-07 明确包含 evidence/current 回滚。
- .add/project/knowledge/design-workspace/article-management/test-plan.md:629 的 A-REC-07 是 P0 原范围；:634、:638 保留 r22 原子绑定期望与 A-REC 原责任；:640 “只新增 article_replay_test.go，不创建第二路径”是当时计数缺陷切片的候选写窗，不是撤销原 S4.4 第二路径。当前原卡 :1516–1517 已记录该第二路径的实际交付，不以旧切片替代现役任务状态。
- 原 design r22 SHA256 8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E；test-plan r20 DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA；现有目标测试 BE6C689273783CFCB4939EE987397E09DB175B30506D9C0E49881FA8CE83A925；共享 fixture B65FAE92A2AE95E291C8CB2CC136769E22209C4E888546CEDF43CC68727F659A，与现役卡相符。历史运行计数、输入总数、schema 数不作为当前新证据。

本次收到 Lead 最新收敛建议：采用新 schema 内单一 BEFORE UPDATE trigger + DETAIL witness；已采纳，不引入 tracer、额外 pool 或生产 seam。本报告不发实施/PG START，不代人的成果 accepted。

## 2. 实际事务与合法故障点

internal/article/replay_evidence.go:317 的 Publish 先 Refresh，:329 开真实事务，:333 deferred rollback，:334 锁 run，:341 核 generation/token；:344 选择/插入 evidence，:348–355 同一 tx setCurrent 后才 Commit。:414–430 在无匹配历史时实际 INSERT；:402–410 current UPDATE 失败保留 PgError 错误链；:992、:1003 分别为实际 INSERT 与 current UPDATE SQL。

0003_article.sql:250–269 的合法 running run 可保持 input_digest/current 为 NULL；:272–294 禁改 run/namespace/backup；:296–324 的 evidence 有 ref 主键、语义唯一、合法 run+namespace+backup FK、计数和约束；:330 append-only 拒绝 UPDATE/DELETE；:335–339 current 具有复合 FK。0009_article_replay_attempt.sql:1–19 保持 generation/token 一致性。新增测试必须正常全量执行 0001～0009，不禁用、绕过或改写这些约束。

采用一份 schema 本地 PL/pgSQL 函数与一个 BEFORE UPDATE OF current_evidence_ref FOR EACH ROW trigger；不使用 deferred/commit-time trigger。安装于本例随机新 schema 的 deletion_replay_runs，DDL 在失败基线前独立提交。所有函数/表名使用 schema 限定并按 pgx.Identifier 正确引用；唯一首 ref 通过正确转义的 literal/TG_ARGV 传入，不能把 DDL 标识符或任意 ref 拼成未经引用的 SQL。函数保持默认 invoker 权限，不建 role/GRANT，不使用 SECURITY DEFINER。

触发逻辑的最小合同（这是方案描述，未安装/运行）：

1. NEW.current_evidence_ref 不是首个指定 failedRef 时直接 RETURN NEW；重试 nextRef 自然不触发故障。这里的“仅首个”指 ref 选择，不依赖会随 rollback 丢失的数据库计数器。
2. 命中时核 NEW.status='ready'，OLD 为原 running/NULL current，run/namespace/backup 与原值一致，generation/token 未改变。不得改写 NEW、owner、digest 或指针以制造失败。
3. 通过同一 trigger 所在事务 SELECT evidence：evidence_ref=NEW.current_evidence_ref 且 run_id/namespace/backup_ref 与 NEW 完全一致、evidence_version 合法、evidence_digest=NEW.input_digest；也核预期 0/3 分项。PK 保证最多一行。
4. 找不到合法匹配行或前置不成立，抛另一固定 SQLSTATE（如 P0001）及固定阶段标识；绝不能仍抛预期 23514。这是 fixture/witness 不满足，不算目标故障命中。
5. 只有实际查到已插入合法行后，RAISE EXCEPTION USING ERRCODE='23514'，固定低敏 message；DETAIL 用 JSON 记录固定 witness_version、stage='current_update_after_evidence_insert'、evidence_seen=true、binding_ok=true、pg_backend_pid() 及本例 synthetic ref/digest。可填 TABLE/SCHEMA/CONSTRAINT 诊断字段以定位本例 trigger。不可含 token、DSN、SQL 参数全量、正文、邮箱、Journal/key 材料。
6. Go 通过 errors.As(err,*pgconn.PgError) 核 Code='23514'，严格解析/核验 DETAIL 的版本/阶段/布尔/ref/digest/PID，不只看 error 文案；保存解析后的低敏 witness。23514 是测试 trigger 主动产生的真实 PG statement failure，不宣称命中了原有生产 CHECK；自填 ConstraintName 也只作注入标识。

为什么足以证明目标位置：失败前目标 schema evidence 为零且 failedRef 不存在，不能通过旧 history/ON CONFLICT DO NOTHING 或另一预置 evidence 冒充成功 INSERT；真实代码在同 tx INSERT 返回成功后才调用 setCurrent，trigger 仅在 UPDATE 时查询到合法行才产生专属 witness。独立连接失败后又看到 evidence 全表仍空、run 未发布，才能联合证明该 INSERT 随 current UPDATE 失败被 rollback。单凭 23514、错误字符串或失败后 count=0 均不足。

此故障发生在 current UPDATE 的 BEFORE 阶段；它证明 UPDATE statement 被拒绝，并不声称已生成 current 新 tuple 后再出错，也不属于 commit outcome unknown、真实网络或真实 PG 崩溃。

## 3. 最小 fixture、步骤与断言

建议复用 s44Prepared(t,3)：article_replay_test.go:883。真实 Finalizer 预完成 ordinary/violating/orphan 三条合法 pending 样本，再 s44Begin(:923) 取得唯一 authoritative handle，s44ReplayInput(:951) 走真实 Source 得到 0/3 完整候选。用 s44Request(:943) 保留同一 owner/input/Refresh，局部 s44IDs(:93) 依次返回 failedRef、nextRef（彼此不同、前置均不存在）。不预先发 Ready，不人工制造目标 evidence/history。

| 阶段 | 必须断言与证据 |
|---|---|
| 前置 | 新 schema 全 evidence=0，目标无 current/input/Ready；真实 Begin 返回的 handle 与 LoadAttempt 完整一致，generation>0/token 非空但不输出；原 backup 固定。真实 Source 分项为 0/3、Facts10 完整，Replay probe=1、candidate token 取值次数=1。安装 trigger 后再冻结失败基线。 |
| 基线 | 保存 s44Image(:1725) 的七表完整所有行镜像，含每行 xmin/ctid，不只 count/目标行；同时保存完整 owner 与 Input 内存副本、局部 IDs/Replay/token/clock 计数基线。完整 owner 包含 token，比较失败时只输出固定原因，不 dump token。 |
| 第一次 Publish | 通过已有 f.storeProbe.Publish 原样委托真实 f.store.Publish（article_replay_test.go:859），不开 ack-loss；错误链必须是上述真实23514+witness，result 为零值 DeletionReplayEvidence，IDs 仅取 failedRef 一次。fixture 未命中/P0001/其他SQL错误均失败，不放宽期望。 |
| 回滚完整性 | Publish 返回后独立 f.observer 回读七表完整集合与每行字段/xmin/ctid，必须等于基线；LoadAttempt 完整 DeepEqual 原 owner，Input 保持不变；evidence 仍零行，无 failedRef、无新 history、无部分 current/input/status 发布，run 仍 running。真实 Finalizer 在基线前已完成的 Article/operation/audit 事实保留，不要求倒回预完成前。 |
| 公共失败后状态 | f.provider.ReadStatus 返回 strict Unavailable6、pending/replay_in_progress、正确 run/backup；不含 generation/token 或 Ready 字段。读后七表镜像仍不变。复用 s44EvidenceUnavailable(:362)，不得把 SQL 内部原因泄到公共 DTO。 |
| 资源 | 复用 s44EvidenceUnlocked(:442)：独立 observer 对同 run FOR UPDATE NOWAIT 立即成功，事务 rollback，pool acquired 仅剩预留 observer=1；witness PID 与 observerPID 不同。既有 f.close(:692) 最终 release observer/cancel，确认 acquired=0 再 pool.Close，取消到达已观测 replay context。资源释放不删除 schema/证据。 |
| 同 handle 新 ref 重试 | 原 request 不变，仅 ID generator 自然取 nextRef；再次真实 Publish 成功，ref=nextRef 且!=failedRef，总 IDs=2，原 owner generation/token/backup 不变、同一 Input/Facts 与 0/3。trigger/function 保留且 enabled；不能 DROP/disable、改 owner、重新 Begin、重新 ReplayAll 或切换到假 SQL。 |
| 成功原子绑定 | 独立连接见仅一份目标 evidence（总 evidence=1），failedRef 不存在；run.current=nextRef、input_digest=evidence_digest、status=ready、error bucket cleared；其他表相对基线全值/xmin/ctid不变。run/evidence 是本次成功允许的唯一数据差异；不能要求重试成功后七表仍全等。 |
| 稳定 Ready | 通过两次 ReadStatus 与 s44CheckReady(:1516) 校验 Ready20、共享 first16/canonical、UTC 微秒、SQL 全 evidence 字段/Facts10/operation/audit oracle、0/3 与 exact ref/digest/time，第二次完全相同且成功后镜像不再变化。读阶段不取新 evidence ID/clock、零 ReplayAll/Begin/Fail。 |

七表精确为 articles、deletion_operations、audit_events、users、identity_identifiers、deletion_replay_runs、deletion_replay_evidence。沿用原完整像定义：users 排除 permanent_password_digest，run 排除 attempt_token；后者另做完整 owner 内存比较。此处“全表”是上述七表的完整行集合，不冒称整个 PG cluster 或所有 schema 的逐表核查，不为本 leaf 扩展共享 snapshot helper。

零额外 Begin/Replay 的最小可复核方法：初次 s44Begin 是唯一设置，Publish 两次都走已有 storeProbe，公开读通过原 provider；检查 probe.history 的基线后序列只允许两次 Publish 与必要 ReadCurrent，不出现 Begin/Fail（且 overflow=false）；同时 tokens.calls 保持1、f.probe.calls 保持1、局部 IDs.calls=2。只看 token 次数不能单独证明无 Begin，需连同调用路径/事件序列核对。f.evidenceClock 在两个新建候选 Publish 各使用一次；稳定读前冻结其计数，双读后不增加。

最小版本不需要 foreign/history。若 Lead 明确要求非空历史保留，允许仅复用 replay_evidence_test.go:403 的原 foreign fixture，并选独立第三 ref；foreign 是另一 run 的合成证据，不能用于目标语义/借作成功 INSERT。必须相对它的完整 evidence image/xmin/ctid原样保存，重试总数按 foreign1+target1核；仍禁止预置目标同Facts evidence。本报告采用零历史版本，不增加可选组合或 leaf。

## 4. normal/race 选择集、retention 与交接

以下全部是后续批准执行窗口的建议命令，本次 NOT RUN。工作目录为固定仓库根目录，安全进程级注入既有批准 DSN；命令行和报告不携带 DSN。

```powershell
go test -tags=integration ./test/integration/recovery -run '^TestArticleReplayEvidencePostgres$' -count=1 -timeout=8m -json
go test -race -tags=integration ./test/integration/recovery -run '^TestArticleReplayEvidencePostgres$' -count=1 -timeout=8m -json
go vet -tags=integration ./test/integration/recovery
```

两条测试均应只运行同一 top 及 DifferentBackupFirstBinding、PublishRollback、CurrentUpdateRollback 三个 terminal leaf：每种模式4个命名测试 run/pass节点、3leaf、0fail/skip，无 no-tests-to-run。必须解析实际 -json 中完整 Test 名集合/Action/exit，不以计数替代集合。旧两 leaf 原断言/真实blocking/23505/foreign保持完整；新 leaf 不再拆嵌套 t.Run，否则先明确实际选择集。

单 leaf 故障定位可用 -run '^TestArticleReplayEvidencePostgres$/^CurrentUpdateRollback$'，normal/race 同样显式 integration；这是定位选项，不新增必须重复的测试批次，不能替代最终包含旧两例的有界回归。未修改 article_replay_test.go/共享 fixture，不强加它的全部风险矩阵重跑；若实际需要改共享 helper，本方案停止而非默扩测试/写集。默认 go test -race ./... 不带 integration，不能替代这里的 race。

fixture 保持 article_replay_test.go:167 的主地址/全部fallback literal loopback:5433校验、:220 实际endpoint/PG180006/身份前置、:192 新random schema独立commit立即log、:235 全0001～0009独立迁移tx、所有连接相同search_path、独立 observer；缺 DSN/权限/连接/版本/关键leaf 不 Skip，不算行为RED/PASS。本轮没有验证这些运行条件，引用代码合同不等于已预检当前环境。

normal/race 每例各新 schema，不复用以前 retained schema；按当前三leaf两模式，预期新建六 schema，最终以实际日志为准，非全库盘点。所有成功/失败schema、trigger/function、新合成inventory、witness、七表前后像、Ready/Unavailable原始payload、execution事实、JSON/stderr/argv/exit/输入hash全部retain。使用原 os.MkdirTemp+日志目录，禁止 t.TempDir 自动删新证据、DROP/disable trigger、DROP schema、RemoveAll、清理/移动历史资料。关闭tx/rows/连接/pool是资源归还，不是资料清理。

不直接运行 internal/article 的旧自动DROP PG harness；不额外使用 overlay、读其他secret或新增role/GRANT。后续开发执行profile/项目适用检查与Lead核diff/精确来源快照、非作者独立CodeReview/适用结果接收分别闭环；本报告不把已有历史Manifest/CodeReview当本leaf当前结果。fixture失败/产品FAIL先保留当时完整代码来源、DB像/schema/raw交Lead，不改生产、不放宽断言、不把编译失败当行为RED。

## 5. 不可跳过的前置与停止条件

六个原 requires 的当前 checkbox 已定向核对为[x]：tasks.md:53 project-bootstrap-B0.1，:567 identity-auth-S4.2，:613 identity-auth-S4.3，:1431 article-management-S4.2，:1493 article-management-S4.3，:1542 article-management-S4.5。仅消费当前原卡，不推算全仓历史统计；S4.3 是决策关闭，不是真Unix卷验证通过，不能外推S4.4豁免。

本 leaf 不需要先新增设计决定、修改 test-plan 或登记新Task/DAG；原 A-REC-07 与第二原路径已足够。Lead最新交接已明确授Euler唯一原文件CODE/LOCAL START；不再把该代码授权列为待补阻断，本角色保持只读，不等待其代码。不可跳过的是：实施者按现有单文件窗口/G0和无DSN静态边界推进、固定最新精确源码；PG仍未授权，后续须有单独PG执行窗口及上述环境/合法schema/trigger权限；零目标历史及witness可观察；实际执行阶段normal/race命中和资源证据；非作者CodeReview/Lead核收。本次“只读、不得实施/跑PG”的限制仍有效，因此没有因“建议值继续”启动代码或数据库操作。

若需生产 seam、共享helper改动、其他文件、修改schema migration/authority/canonical、放宽owner/绑定、额外环境/secret/GRANT，停止相关部分并向Lead说明；不得以本报告扩权。拥有本schema的既有身份不能创建本地trigger/function时记环境阻塞，不尝试提权或改认证。本局部用controlled公开Journal，不把真Unix Journal作为该SQL leaf的额外前置；完整真实恢复的原门禁全部继续。

## 6. 覆盖映射、未验边界与读集

映射：现役设计r22的Ready原子提交/失败回滚 → test-plan r20 A-REC-07/P0 evidence/current回滚子集 → 原 article-management-S4.4 → TestArticleReplayEvidencePostgres/CurrentUpdateRollback（建议新增，未登记新Task，NOT IMPLEMENTED/NOT RUN）。最低足够层为真实PG集成事务故障测试；不重复单元/契约全组合，不冒充真实E2E。安全基线是owner/backup/append-only保持、公共DTO不泄token/内部error、DSN不输出；本例无性能容量阈值，不给P95/RPO/RTO/retention编造数值。

自动化边界：合法fixture、故障witness、回滚/重试/稳定读/资源/命中集合可自动验证；当前真实执行环境与窗口由Lead/执行者核实，非作者Review与人的结果接收不能由本规划代签。

证据界限：未来即便该 leaf 真运行成功，也只证明受控Journal、合成backup引用/manifest和真实PG事务子集；不证明真Unix FileJournal/HMAC/keyring/POSIX、真实旧backup恢复、真实网络/崩溃、完整A-REC-07、双namespace/MarkConsumed/global opening、真实inventory签发挂载、应用ACL、S8.2/Case18、RPO/RTO/retention，不是全S4.4/恢复/E2E/PASS。本轮只有静态方案核对，未有新测试运行结果。

实际完整读取：本角色add-test-engineer.toml、add-route-work/add-delivery-workflow/add-test-design现役SKILL全文；coding-standards/code-review-checklist/task-execution-workflow/design-doc-writing-style与go-service README/编码规则。证据按上述准确行段读取原S4.4卡、r22恢复/Publish段、r20 A-REC-07/4.8.1、replay_evidence_test.go、article_replay_test.go相关fixture/镜像/probe/Ready/资源、实际internal/article/replay_evidence.go与0003/0009约束；设计/规划评审与verification仅作相关交接背景，不重新签其历史运行。未读无关全库或运行配置值。

个人报告由当前会话交回Lead；Lead最新收敛建议已纳入，不新增阻塞确认问题。精确输入指纹在inputs.json，最终复核记录在verification.json。本报告保留，不自动清理。
最新交回结论：无新增合同阻断；Euler CODE/LOCAL已由Lead授权，当前仅代码与无DSN静态，不授PG。该唯一leaf方案已完成覆盖核对，不依赖等待代码，不扩全S4.4或全库分析。
