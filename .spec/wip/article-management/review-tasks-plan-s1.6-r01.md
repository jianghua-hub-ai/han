# Article S1.5/S1.6 Task/Plan 独立 Planning Review R01

**PlanningReviewGate PASS，保留 1 项非阻断 LOW metadata finding。** S1.6 的两文件实施合同可执行，六项 requires 已满足、无新增循环；S1.5 的原四文件/16 endpoint 关闭范围保留。此结论仅针对规划合同，不代表实施 START、代码通过、测试通过、S1.5 完成或人的验收。S2.2 已闭合，不重开。

## 固定对象、身份与权限

- objectType：`design`（Task/Plan 实施合同）。实际非作者 `add-reviewer`；原作者 Ptolemy 已停写为 Lead 提供的交接事实。本角色没有下级委派、没有替 Planner 改文档。
- 工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。核真实工作树文件，不以 HEAD 代替大量既有未提交内容。
- 审查范围仅为 S1.5 R gap-map、S1.6 新卡、唯一 B9E 和相关依赖/统计；未评审活动 r19/Admin2 设计或其他身份支线。
- 唯一项目写入为本报告；Temp 分析位于 `C:\Users\J00010385\AppData\Local\Temp\han-s15-metadata-456910e3fd5447cc8f9aee3998c5c97c\review-s16-r01`。被审对象、代码、旧报告及其他文件均只读，无 stage/commit/push、删除移动清理、PG/服务启动或外发。

| 固定输入 | 实际 SHA256 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `4E470A6C3646CAB0D45236BADF4773CA4F5C13B20CBD21C4DF604654417C1F87` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `A42BAA4AC9C2BADEAE81A141C00E72054A4334B2B172982314945DFA3E908A5D` |
| Temp `tasks.before.md` | `2AE92B4F4722F35CE52D3EB3932044DC0BE2ECCD46176A48E0EED05A4676A1B2` |
| Temp `plan.before.md` | `3776FD172599DE31F1B5DEED2BFA81629EEEAD3BC86D2D0F12DB51CD83B23147` |
| 固定 r18 全快照 `evidence-s2.2/fixed-r18-design.md.txt` | `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3` |
| 现役 article test-plan r15 | `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` |
| `cmd/han-server/main.go` | `D6A82DBDD9C72D3A2435A32F1059494ABEA94E5F3B05ED1B5C93C52FAF97AFB4` |
| `cmd/han-server/main_test.go` | `0346F1AA96C6EDDB6409148E648D37FD15448F2C73AC966BA18AFE3ADA3DBA68` |

上述两个对象、两 root 与 r18/r15 在审查前后复核一致，无漂移。未加载活动 r19 作为本次判断基线；r19 的动态变化不使本次三入口固定合同失效。若共享 Tasks 全文有后续身份段变化，也应重新固定相应版本，本报告不覆盖该后续版本。

## 实际方法与两轮范围

复用同会话已完整取得且仍有效的自身 Role、`add-code-review` 全文、安全红线 reference、checklist、abstraction、task-execution、multi-agent 和当前 delivery Workflow；按 design 分支审查。go-service 仅按已启用的 Go root 实施形态核最小范围、依赖注入、安全链和未来验证可达性，未将代码格式/编译/构建测试条款强套本次规划成果。未新增其他 Workflow 或预读未来实施方法。

**Round 1：真实 before-image → 固定对象完整 diff。** 实际生成、读取两文件完整 unified diff，并按 raw bytes 的行差异再次核边界。Tasks 的修改只涉及当前总数/三态、原 S1.5 卡和新增 S1.6 卡；Plan 只涉及总数/来源、B9E/B10 行、覆盖计数及新增本范围说明。对新卡和 gap-map 逐项核目标、两文件/四文件所有权、requires、批次、未认领/未 START、接收与关闭条件。不是依据 Planner 摘要判断。BOM 保留；Tasks 原 LF、Plan 原 CRLF 形态保留，没有整份格式重写。

**Round 2：上游与真实消费者有界核对。** 读取固定 r18 的接口总表、共享安全/表示合同、Story1 创建与 Story3 作者读取/更新；test-plan r15 的 A-CREATE/A-MOD/A-CON 及适用责任。完整读取两份 research 和 Dalton prep 作为证据输入，定向读取两个真实 root 的 composition/builder/router/现役回归，以及 article Handler 构造接口、Create/GetAuthorArticle/UpdateArticle 和 query service 的作者读取方法。核 16 设计 endpoint、当前 11 条实际 Article method/path 与新增三项后的 14/16 边界，不补读其他业务实现库或身份文档。

## Finding

### AM-S16-PLAN-R01-F001 [LOW] 分项统计仍少计新增开发卡

- 对象/位置：固定 `tasks.md:48`、`:49`；当前总数/三态在 `:44`、`:45` 已更新。
- 触发与现状：本次新增唯一 `article-management-S1.6` 后，实际任务头为58个唯一 ID：project1、identity35、article22；但分项仍写 article21，模块分项和为57。类型汇总也仍合计57；按沿用的开发口径（含既有一张“开发返工”卡），新增开发卡后开发应为37，而非36。
- 影响：读统计的人或按分项对账的工具会得到漏一项的覆盖结果；实际 S1.6 卡、唯一 B9E、requires 和三态正确，所以不阻断本次两文件实施合同，不升级为业务/权限缺陷。
- 违反依据：Task SSOT 与 Plan 完整性/唯一任务对账的一致性要求，checklist 的输入/契约一致性。该项是数值错误，不是风格偏好。Lead 提示后，本角色通过真实任务头和类型 metadata 独立复核。
- 最小修复：原 Planner 仅将 article 分项21改22，并按既有归类将开发36改37；不新增/重开任何任务，不改变状态、依赖或身份文字。本角色未修改对象。
- 证据与复核：`metadata-analysis.json` 的58个任务、模块/状态/类型计数；before-image 差异证明本轮仅新增一张开发卡。修复后应有模块和=58、类型和=58，实际 checkbox 仍43/2/13，新增 diff 仅统计行。可由 Lead 安排原作者处理并核新 SHA；LOW 可排期，不阻断本次 Gate。当前本 finding 保留 OPEN，未替 Lead 接受风险或标作已修复。

没有 HIGH/MEDIUM finding；没有新增架构/产品方向决策要求。

## 数量、依赖、唯一批次与保护范围

| 检查 | 独立结果 |
|---|---|
| 唯一 task ID | 58；相较 before-image 仅新增 article-management-S1.6 |
| Checkbox 三态 | 43 completed / 2 active / 13 todo；总和58 |
| 模块实际数量 | project-bootstrap1 / identity-auth35 / article-management22 |
| 已有卡正文改动 | 只有 article-management-S1.5；其余已有卡逐卡 raw bytes 相同 |
| 身份隔离 | 35张 identity 卡全文字节均与 before-image 相同；diff 也未修改身份注记/批次依赖正文 |
| 已关闭保护 | S2.2、S7.1、S7.2 的完整卡字节和 `[x]` 均未变；不重开、不重审旧关闭 |
| 任务/批次覆盖 | 58项全部有批次，无遗漏、未知 ID 或跨批次重复；详细历史映射重复引用同一批次不算第二认领 |
| S1.6 | 唯一 B9E；`[ ]`、未 START，暂无生产写授权 |
| S1.5 | 唯一 B10；`[ ]`、四目标文件实际均不存在、未执行 |

S1.6 精确 requires 为 `project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S1.3`、`article-management-S2.1`、`article-management-S3.1`、`article-management-S4.6`，六张真实卡均 `[x]`。S1.5 原十项 requires 全保留并均 `[x]`，仅新增尚未实施的 S1.6；本卡因此仍不能完整关闭。沿这些 requires 可达31个节点，未发现循环；未添加 S1.5/S2.2 反向依赖或虚构 Admin2 占位 task。

共享 root/handler/query 冻结窗口由 Lead 显式释放、确认实际 owner 后另发 START；这是文件/执行窗口控制，不是 S2.2 业务前置。Planning PASS 和 Backend prep ready 都不等于实施授权。

## 两文件实施及成功证据的分层判断

S1.6 只接现役 Create、作者专用 GET、Update，来源分别为固定 r18 `:997`、`:1000`、`:1001` 及 Story1/3；不新增业务模型/API/迁移或 Admin2。实际 `main.go:3157` 附近 `newArticleHandler` 已用同一 CommandService 构造 `article.NewHandler(service,responseLogger,service)`，包含 Create 与 content replacer；query builder 已有同一 ArticleQueryService，可通过现有公开 `NewAuthorDetailHandler` 绑定作者视图。当前 builder 数组和同包 test seam 可在原两 root 文件内扩展，无需导出 cmd API、改 internal 或另建 router/service。

当前真实 router：collection 仅 GET，单文章仅 GET/DELETE，无作者专用路径。现役 `main_test.go:2985` 与 `:7730` 的 POST/PUT405、Allow 断言确会与新增方法冲突。卡已要求先保存运行期路由失败证据、再接三入口，并更新完整 method 表、精确路径、安全短路和旧 Identity/DELETE/notice 回归；没有授权删除旧回归或用编译错误冒充行为 RED。共享实例 callback 与真实构造入口均已存在，可补观察实际 service/handler/query 依赖身份。

“成功201/Location/ETag、AuthorEditView/ETag”在本次两文件合同中是 HTTP 组件成功响应断言，**不构成新增真实 PG 或 E2E 门禁**。固定对象整体已有受控 Handler / 真实 root / 真实依赖分层，结合现役源码，可执行为以下互补证据：

| 层次 | 在两文件内可行的验证 | 能证明的边界 |
|---|---|---|
| 真正 production root 构造/分流 | 调用现役 factory/builder/router；补实际 constructor/共享引用观察；路由 spy、Origin/Session/CSRF/Principal、错误方法零下游与 Allow | 三入口注册、安全链、实例共享；终端 spy 返回201不能单独证明 Handler 响应或 DB 成功 |
| 受控真实 article Handler | `ArticleCreator`/`ArticleContentReplacer`/`AuthorArticleReader` 现有公开接口可由本地测试 adapter 提供确定合法返回，驱动真实 Handler 检查201/Location/ETag、作者最小表示及错误零调用；可用允许的私有 seam 或同包局部组件测试 | 实际 decoder/表示/HTTP Header 契约；受控下游必须明确标注，不称 PG 持久化成功 |
| 既有业务组件回归 | 运行相关 article 回归和适用 Manifest；保留原组件/集成证据的精确版本与范围 | 业务规则回归，不借 identity integration 历史 PASS，不把旧关闭本身当本轮执行结果 |
| 真实 PG/部署 E2E | 本次规划评审不运行，也不是本开发卡新增前置 | 不由 spy 或组件结果外推；后续仍归既有适用验证任务 |

具体可达性依据：`handler.go:88/93/104` 接口、`:198` 构造、`:308` 作者专用构造、`:380` Create、`:441` 作者 GET、`:474` Update；`query_service.go:183` 的真实作者读方法。因此无需为201成功响应扩大到第三生产文件或强制 DB 环境。执行交付应逐条标明上述证据层次；只测手写终端 spy、只有引用存在/数量而没有真实构造身份检查，仍不满足卡内要求。

Dalton prep 的建议前缀 `TestArticleRootS16_` 是早先候选，不是正式验收 SSOT。固定新卡指定 `TestProductionArticleAuthorRoutes`，命令允许该前缀及现役 router/query/business/notice 回归；实现时应统一为正式卡符号并检查真实 JSON run/pass，不能沿旧 prep 名称造成关键新 top 零命中。新 top 当前尚未实现被明确标为 NOT RUN，不据此判规划不可执行。

## S1.5 保留及未决范围

原四新增 contract 文件逐一存在性核为 false，路径保留；设计的16 endpoint、三投影、公共错误、S1–S16、Ready20/Unavailable6责任未缩成当前11或本卡交付后的14。普通 admin 两个GET仍是输入 pending 的独立工作，未用 orphan 查询/裸 AdminProjection 冒充，也未填假 requires；r19 设计固定后的实施责任由其 owner 另行落实，不阻断本次既有三入口合同。

Admin 投影按 research r02 更正为12顶层字段，未采用 r01 原来的13字段计数。A-CON-03 保留真实审计 INSERT 回滚与实际应用身份 UPDATE/DELETE 权限/结果，不能以 trigger 或 superuser 代替 ACL；环境/授权载体未落实即 pending。Producer 的受控 serializer、真实 Store/source、Journal/恢复分别保留其证据层次和原任务责任；unknown Guard 不创造新 API，缺真卷不倒挂纯组件准备。S1.5 的完整关闭仍受这些真实缺项限制，本次不宣称它已可全量执行或已通过。

## Readset、分析证据与未执行项

除前列固定材料，完整读取的研究为 `research-s1.5-readiness-r01.md` SHA `77C32EC9ED2AAA14C2D1D6DBD48E7584715B9976B5C2D66D1339F8F9186B509A`、r02 SHA `C6B333DDD649D224EA8DCCAE02BFAFC05D0B00F15ECAE4E0D49D8C329EC9177A`。Dalton `backend-prep-g0-tdd.md` SHA `AF5CAF15A9C1EF612E3E9E8BB33889A100D859C1BED6E6B151915CC772691FFB`，只作为准备证据，不是设计批准/START。

直接源码切片：main.go 的类型/seam、`:2227` factory、`:2874` builders、`:3157`～`:3256` command/protection、`:3261`～`:3385`完整 router；main_test.go 的`:2985`旧method表、`:4189`共享依赖、`:7663`构造及`:7730`共存/安全回归，另定位验收命令对应真实符号；article Handler/QueryService 仅前述直接方法。handler/query 当前 SHA 分别 `A0921B0178551A8EA51772ECF9823B50AA693DD3B450FDD347257B6F77CCC7E8`、`9674130673CD05E8FA864B77528FDBF12C9AB8305AE6AD2EF7E2A85DB9136E7E`，与既有固定依据一致。没有读取/评审全部代码或活动 r19。

Temp 分析文件：

| 文件 | SHA256 |
|---|---|
| `metadata-analysis.json` | `80C859F86E48F45AD219EFC4517983B1677440C0DFF28B1FC3112B6C08DBC346` |
| `tasks.diff` | `1B5F9FD91DD5C7009AA990979F57900E4B0AAF48B7903DB909A30FD5D2624084` |
| `plan.diff` | `07B02632174671A47BBE98C63B293B2253D508CA9846207CB10056BC81C8AD87` |
| `freeze-before.json` / `freeze-after.json` | `3F6F9ECF491E91B521506BDBC637B0FA660F75A1144D0AC0ECCB33347E4C43FE` |
| `direct-source-binding.json` | `FB48649BFCBF66AF1E8557CC32624D3CC83D051DD5D1E9D51396489C80D9366F` |

本次仅执行只读 Git HEAD/branch、文件 hash/路径查询、before-image diff、字节/任务数量/批次/依赖的静态分析；未运行 Go test/vet/build、PG、HTTP 服务、SMTP、浏览器或 E2E。源文件断言存在不等于本次测试通过；当前未产生代码 RED/GREEN 或 CodeReview PASS。无新生产授权、任务状态或人的接收记录。

**stopped**：本角色两轮规划审查与独立报告完成。交 Lead 核报告 SHA，处理 LOW 统计项并确认实施 owner/冻结释放/正式测试名与后续 START；Reviewer 不替 Planner 修改固定对象、不关闭 Task、不启动实施。后续 Task/Plan 或源码修订须按新差异核对，本 Gate 不覆盖活动版本。
