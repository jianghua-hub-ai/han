# article-management-S1.6 独立 Code Review R01

**CodeReviewGate PASS：无 material finding，无需原作者返工。** 结论针对下述冻结的两文件及本卡三入口接线/进程内验证责任；不是数据库成功、部署/E2E、全部16端点或人的验收结论。Task 由 Lead 按实际交付证据接收和关闭，本角色未改态。

## 固定对象、身份与授权

- objectType：`code`。实际非作者 Erdos / `add-reviewer`；沿当前 development-delivery / S1.6 review 续接，未 Route、未委派、未代作者修复。
- 固定工作区 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，实际核对一致。作者 Dalton completed/stopped 是 Lead 的真实交接输入，作者最终交付也记载 stopped；本角色未另查 Host。
- 两 root 是现役未提交且 Git untracked 源；实际用 before-images 对比，不以普通 git diff 为空推断没有变更。
- 项目唯一新增写入为本报告；新验证/分析仅写入 `C:\Users\J00010385\AppData\Local\Temp\han-s16-9e8157aae2bc45babef17865d9804a7f\review-code-r01`。没有修改源码、测试、TaskPlan、设计、配置、旧证据，没有 Git 写、清理删除移动、PG/SMTP/服务/Browser 启动或下级委派。

| 对象 | 实际 SHA256 |
|---|---|
| `cmd/han-server/main.go` | `2B75C96E61F5BF758F68935F4656238CD8D891D6DC646C32B55EFD0C1A5A6A14` |
| `cmd/han-server/main_test.go` | `092BD3BD78169A40780169A27948D3BD1C6E423406E2C8E658B0EFC59BF3BCBA` |
| `main.go.before.txt` | `D6A82DBDD9C72D3A2435A32F1059494ABEA94E5F3B05ED1B5C93C52FAF97AFB4` |
| `main_test.go.before.txt` | `0346F1AA96C6EDDB6409148E648D37FD15448F2C73AC966BA18AFE3ADA3DBA68` |
| 固定 r18 design 全快照 | `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3` |
| 固定 r15 test-plan 全快照 | `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` |

最终 delivery-final 两份 after-image 的实际 bytes hash 与上述两源完全相同。实际生成完整 unified diff：main.go 为73行新增/16行移除，main_test.go 为624新增/3行移除。155个最终输入按 `lead-source-final-r01.json` 重算均匹配；153项保护依赖按 `delivery-final/protected-after.json` 重算0 drift。Reviewer 前后再次比较155项，0 drift；hash核验不表示全文评审了155个文件。

## 方法、上游与两轮实际检查

复用同会话同版有效的自身 Role、完整 add-code-review、安全红线 reference、checklist、abstraction、task-execution/multi-agent/delivery，以及 coding/no-hardcode/security/observability/tracing 和唯一启用 go-service 规则全文，按 code 分支执行。项目实际检查按 Manifest；goimports/golangci-lint 未安装/未执行，不宣称 full-profile PASS。

消费 S1.6 原卡三入口、两文件、六项 requires、正式测试前缀/安全链/共享实例/关闭条件及 Lead START 原卡记录；Planning R01/R02 的范围与分层结论继续适用。TaskPlan 全文与活动 r20/后续 Admin/身份 metadata 不作为本次代码冻结对象，未因其独立增量重开 S2.2/S7 或扩大代码范围。固定 r18/r15 按三入口、Author4、ETag、wire、安全链及响应头适用切片消费，不借活动 Admin2 设计批准新代码。

**Round 1：完整两文件 diff。** 实际逐段读取全部生产变更及624行新增测试、3处旧断言替换；核槽位/复制边界、构造与错误分支、授权顺序、精确路径/Allow、测试 fixture/断言真实性、资源退出、内容及错误去敏。没有只按作者摘要、测试名称或统计数判断。

**Round 2：真实消费者与上游有界检查。** 跟踪 `newProductionBusinessHandler → buildArticleHandlers/buildArticleQueryHandlers → protect* → newLoginBusinessRouterWithTrace`，以及实际 Handler/QueryService、公共 JSON/error/trace、HTTPAuthorizer/OriginGuard、session Principal 接入。核固定 r18响应头、ADR-0014最小披露与最终测试调整；解析原始 RED/最终 ordinary/race/业务回归及 Lead Manifest，完成一次独立 race 窄测。未扩成全库、Admin2或真实数据库审查。

实际直接 Readset：

- `main.go`：新增 seam，factory/builder、query/command构造、共享观察、保护链、完整router及 nil-handler 分支、`:3476`精确路径函数、`:3756`附近生产 server Handler 外层；`main_test.go`：全部新增/变更，及`:6586`/`:6655`现役 composition helper、query/shared/旧共存回归必要上下文。
- `internal/article/handler.go`：构造接口、Create/GetAuthorArticle/UpdateArticle、If-Match/error映射、WriteArticleJSON/Error、wire decoder；`query_service.go`作者详情；`policy.go`作者投影；`model.go`Author4。相关现役命令服务语义沿已读同源契约，不修改生产组件。
- `internal/platform/http_security.go`：真实 Origin、HTTPAuthorizer、CSRF按方法判断、Principal传递与错误映射；`http.go`：TraceMiddleware/JSON decode/write；`errors.go`：公共envelope与最小披露；`internal/session/store.go`的ResolveInput/Principal/AccessGeneral必要段落。
- 固定 r18/r15快照；`.add/project/knowledge/decisions/framework.md` 仅 ADR-0014 章节；原卡和两份 Planning报告、作者 final交付/原始运行/保护清单与 Lead 最终证据。未把其他身份章节当评审对象。

## 接线、行为、安全与测试充分性结论

| 检查面 | 实际结论与证据边界 |
|---|---|
| 旧索引与三入口 | Article结果11→14、router17→20；旧0～10/0～16语义保留；Article11/12/13复制到router17/18/19，分别Create/AuthorDetail/Update。`copy(handlers[9:11],protected[5:7])`限制旧尾段，避免新增Update覆盖AuthorDetail。 |
| 命令唯一性 | Create/Update来自同一个 `components.handler`、同一CommandService；既有Orphan包装为值复制，保留creator/replacer/logger。未新建第二command/service/router/authority或公共API、配置/迁移。 |
| 作者读取 | 同一ArticleQueryService分别构造成员handler与`NewAuthorDetailHandler`；作者路由不复用成员公开详情。真实作者方法按当前Principal与author_id匹配，pending/无归属拒绝；Author4不含类别/原因。 |
| 实际构造引用 | 未替换终端handler的constructor测试，比较实际构造点上记录的command、query、handler、logger、pool、clock引用；并结合真实constructor调用参数核对。受控成功fixture与该constructor检查分开，未把受控端口当生产DB。 |
| 方法/路径 | collection GET/POST；普通detail GET/PUT/DELETE；作者detail GET；空id、额外段、尾斜杠不匹配，不做redirect。旧Identity/member/delete/moderation/orphan/notice路径保持；Admin2没有新增实现。 |
| fallback/nil | router先本地method/path判断，错误方法405+完整Allow，未知路径404；合法方法选中nil handler仍返回JSON404，不调用nil或进入业务。该nil分支静态核对，未声称窄测单独注入了每个新nil槽。 |
| 安全顺序 | Trace最外；POST/PUT进入同一OriginGuard→HTTPAuthorizer→AccessGeneral→handler；GET只需Session/Principal，RequireCSRF=false且不额外要求Origin。失败/错误方法前缀事件与零下游断言实际命中。 |
| 身份/授权 | cookie/CSRF/token/clock输入原样传给实际HTTPAuthorizer；no-cookie/Origin早拒绝不进入resolver；受限/停用/recovery拒绝。对象所有权403/404由现役Service/Policy承担，未用管理员权限扩作者编辑权；受控root fixture不冒充真实SessionStore/PG解析。 |
| 成功响应 | 受控现有Creator/ContentReplacer/AuthorReader驱动真实Article Handler与真实root链；201/Location/同资源ETag、作者GET/PUT四字段、更新输入id/version、原文及restricted均实际断言。 |
| 失败与decoder | 26个叶例分别经过受控和未替换终端handler，核坏JSON、未知/重复字段、N+1、非法原始UTF-8、缺/weak/跨资源If-Match拒绝；零业务调用。N字节合法请求真实decoder继续且仅一次受控下游调用。无PG成功声明。 |
| 错误与隐私 | 实际ArticleError/platform error writer提供稳定码、message、trace；内部cause不入响应。受控DB错误503只作为失败传播证据；无ETag/Location成功信号。合成正文、origin、cookie/CSRF样本不用于真实用户数据。 |
| 旧测试 | 原POST/PUT405项改成受支持成功项，同时保留PATCH等错误方法和更新Allow；未删旧回归，GET/PUT/DELETE共存与旧安全回归同步。 |
| 资源 | fixture使用未连接的pool对象、受控端口/假Journal；构造worker未启动运行循环。prepared资源有t.Cleanup，Origin观察context及时cancel；独立测试进程实际退出。没有启动listener/PG/SMTP/后台worker。 |

本卡结果为14/16条Article生产method/path，普通管理员两个GET仍由后续工作承接；S1.5四文件/16端点验收不缩减。S2.2/S7历史关闭及身份豁免不借作本卡证据。

## nosniff 中间失败的独立判定

实际核到 `new-tests-02` 除请求fixture缺Content-Type外，通用fixture曾要求每个返回（含Origin/Session拒绝）都有nosniff。当前固定 r18 `:963`明确要求**成功JSON**带 `application/json; charset=utf-8` 和nosniff；`:885`明确要求router404/405带nosniff；ADR-0014要求公共错误稳定envelope/最小披露，没有另设所有middleware错误的统一header条款。

真实 `article.WriteArticleJSON` / `WriteArticleError` 设置nosniff；真实 router fallback/405设置且新测试仍逐项断言。`platform.WriteError`、OriginGuard/HTTPAuthorizer与生产`http.Server{Handler:handler}`外层没有额外全局nosniff包装。因此把通用fixture的全局header要求移除，不是生产实现反向修改已确认的成功/404/405合同；未修改任何共享platform行为。

本审查没有据此宣称所有middleware拒绝都已有nosniff，也不豁免S1.5/A-CON-01后续完整16端点错误/头矩阵。成功与Article错误writer的现有保证、明确fallback测试、通用安全错误envelope责任分别核对；不为本卡凭空要求改全库旧行为。未形成finding。

## 已有原始证据：独立解析，非本角色重跑

作者原始文件均在获准 Temp 根。本角色逐行解析JSON，按实际声明的10个新测试函数核命中；旧中间文件不替代最终fixed。最终 ordinary/race 节点集合实际一致。

| 执行 | exit | top / run / pass / leaf / fail / skip |
|---|---:|---|
| card-unit-fixed | 0 | 36 / 271 / 271 / 232 / 0 / 0 |
| card-race-fixed | 0 | 36 / 271 / 271 / 232 / 0 / 0 |
| 其中实际新增前缀，两套均一致 | 0 | 10 / 116 / 116 / 91 / 0 / 0 |
| red-entry | 1，真实行为RED | 1 / 4 / 0 / 3 / 4 / 0 |
| green-entry | 0，中间revision | 1 / 4 / 4 / 3 / 0 / 0 |
| business-unit / business-race，各自 | 0 | 249 / 880 / 878 / 776 / 0 / 2 |

正式卡命令为原卡精确前缀普通/race：`go test [-race] ./cmd/han-server -run '^(TestProductionArticleAuthorRoutes|TestArticleBusinessRouter|TestProductionArticleQuery|TestProductionBusinessHandler|TestNotice)' -count=1 -json`，两份meta都绑定本次最终两SHA、移除子进程PG DSN。新增10 top中成功3leaf、constructor1、security28、精确错误方法15/路径8、restricted2、精确N2、下游失败3、非法写26、安全入口3，共91leaf。

RED meta 的 main SHA是before `D6A82D...`，测试SHA与保留 `main_test.red.txt` 的 `2DFF446F99EAE4CD0D6D13A249AB09D0A49F09BED59C801323F297CC56299DFA`一致。实际3leaf分别为POST405而非Origin403、作者GET404而非Session401、PUT405而非Origin403，编译成功后断言失败。green-entry同三例转绿，但其main/test是中间修订，不假称最终SHA。

`new-tests-01` exit1/run0属于编译错误，不算RED；`new-tests-02` fixture/header失败原件保留，不删失败链。最终fixed才是本次验收权威。业务包两skip为 `TestFileBackupInventoryReader/symlink_rejected` 与 `/world_writable_rejected`，Windows限制继续标skip，未改成PASS或转作article waiver；相关业务源在保护清单中未漂移，早期test-only/root-coexist运行不冒充最终testSHA。

| 原始证据 | stdout SHA256 |
|---|---|
| card-unit-fixed.stdout | `E36BF62DB1625713EC960B9B8CF44648C655416F0B305705D91F928A8988FB42` |
| card-race-fixed.stdout | `60C0A4FD2A3B599F22222FA7A0075A95E973A48775FDFEC412D27D3BDBE98C04` |
| red-entry.stdout | `B1010152A213835BB1D721619D594CCF15CF0A6F2D31CEB9D9750C7C205E5D94` |
| green-entry.stdout | `874B5B443E246D308110EDE106CA97C028CC820F48E97F0E1CB10890D2BBD514` |

上述stderr均为空，SHA256 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。meta/full集合/失败与skip/hash记录见 `review-code-r01/author-log-analysis.json`，SHA `E52DA9475C01670158010177FB20ECB8B20CB87C05A6EAD2BCAF5A7D2C537A6A`。

Lead Manifest索引 `lead-manifest-r01-results.json` SHA `C59560AAC06C095FF5823667FE23565381D0BBC56203C54415C024FC747FA5A1`：本角色核实际7项命令、每份stdout/stderr bytes hash及155项source_before/source_after，全部匹配、无drift。依次为format/modverify/vet/build/contract/race/unit，exit均0、无timeout、format空；子进程DSN已移除。它们由Lead执行，本角色只核证，未机械重跑全7。实际分析保存在 `review-code-r01/lead-manifest-analysis.json`。

## Reviewer 本次实际窄测

只在子进程环境移除 `HAN_TEST_POSTGRES_DSN`，未读取User-scope DSN或打印值；无integration标签。实际独立命令：

```text
go test -race ./cmd/han-server -run '^TestProductionArticleAuthorRoutes' -count=1 -json
```

**exit0；10 top / 116 run / 116 pass / 91 leaf / 0 fail / 0 skip**。这是针对新增安全、constructor、真实decoder及受控Handler响应的独立窄测；不是作者36top正式卡或Lead Manifest的重复执行。进程实际exit，未触发真实数据库、SMTP或服务。source155前后保持固定。

| Reviewer新证据 | SHA256 |
|---|---|
| narrow.result.json | `5148200A2345274B7FAD56CF8155C4035324BC95719F65B1020EF18A08C1BDEB` |
| narrow.stdout.jsonl | `F0F02EBCDEAA8360570C0111B70A47DAAFB954D87EAC03F3C5D22DE119EE3B50` |
| narrow.stderr.log，空 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| narrow.analysis.json | `15FA43E51B50437C3A2A9D86FC7FD57E351CFD9889E74EBFD0395164FE74A3E6` |
| lead-manifest-analysis.json | `38369106EB1FDCB766B658DB9A4BCA69B9295F284FB538009B6D97F2155C8638` |
| freeze-before.json / freeze-after.json | `A86A5928EC577F9F379A77626F52DC2822CFC82F133C6BB028F0DD4AEE13744B` |

## Findings、限制与交回

- **缺陷 findings：无。待方向决定：无。额外非阻断风格建议：无。** 未用构建成功或无finding掩盖下列未验范围。
- 本次证明生产factory/root的进程内接线、安全链及适用组件契约；成功响应使用受控业务端口，不是真PG写入/Session读取。未验证真实PG、SMTP、Host进程、Browser、完整Journal/Unix真卷、部署E2E、容量/P95、全16endpoint或生产就绪，也未改S1.5/Admin2、S2.2/S7或身份材料状态。
- 若原两文件/相关消费者/上游契约再变，需固定新SHA及差异/回归再评；本PASS仅覆盖本次对象。作者工具不可用项不计PASS，Windows skip边界保留。
- **stopped**：独立两轮、窄测与报告完成，没有仍运行的测试/写任务。交Lead核报告SHA、实际worktree/diff/155绑定与接收/Task关闭条件；本角色未改TaskPlan、未关闭任务、未代签人的验收或启动后续实施。
