# Article Admin2 独立 Design Review R01

日期：2026-10-27。objectType：`design`。本会话以实际加载的 ADD Reviewer 身份独立评审 Hubble 产出的固定增量，非原作者；未下级委派、未实施或修改被审对象。评审方法：`add-code-review`，两轮均已执行。

**结论：CHANGES REQUIRED。1 项 MEDIUM，0 项 HIGH/CRITICAL。** 新管理员列表的 HTTP presence 合同指定复用会静默丢弃解析错误的助手，无法保证稿件承诺的显式值与重复参数拒绝。原作者修正设计、同步相关 Story 并交原 Reviewer 复审前，本次设计不记 PASS。核心方案的用户授权采用不等于接受本 finding 的风险。

## 1. 固定对象与授权

工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`；开审及交回前实际只读核验 HEAD 均为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。HEAD 不代表工作树全部内容已提交，评审按以下工作树字节摘要固定对象。

| 对象 | 修订 | 实核 SHA-256 |
|---|---|---|
| `.add/project/knowledge/design-workspace/article-management/design.md` | r19 | `19A06AE6FB78DB1C4F0F31A34FCFF789936AE467004CA660C2F211D14688C3CA` |
| 同目录 `stories.md` | r08 | `9AAD29B0075C387195B99AC4F35D8AF8B5A5C1253D57E4D4FBF6CC85F3C3F361` |
| `.spec/wip/article-management/evidence-s2.2/fixed-r18-design.md.txt` | 固定 r18 | `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3` |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | r15，仅旧风险输入 | `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` |

逐字节基线取自 `C:\Users\J00010385\AppData\Local\Temp\han-article-admin2-r19-20261027T144020353Z\baseline.json` 列出的两个 before-image。实际 SHA 与登记一致：design before-image 为上述 r18 SHA；stories before-image/r07 为 `D5D933E244D3D5E1D5869F19AAFAA8FB67C78BF62882C0883A2C2BDCA50D9B05`。design 从 2710 行变为 2881 行，stories 从 252 行变为 263 行；前后均无 UTF-8 BOM、无 CRLF，实际 diff 不以 HEAD 中更早内容代替 r18/r07。

本对象限于现有 16 endpoint 总表中的 `GET /api/v1/admin/articles`、`GET /api/v1/admin/articles/{id}` 内部读取合同。现役 root 的 11 个文章 method/path 由源码重新核实；另三项已有 Handler 的 production 接线不在本对象，S1.6 规划进度只按委派输入保留，不查 Tasks/Plan、不宣称已登记或已实施。两个 orphan GET 不替代本对象。

唯一仓库写入为本报告；Temp 仅存分析 diff、读取摘要及标准库解析探针。没有写设计、Story、test-plan、Task/Plan、源代码、测试、ADR、配置或记忆；没有 Git 写操作、stage/commit/push、PG、服务、删除、移动或清理。开审工作树已有广泛存量变动，保留其内容及归属。本次读取集合末核无哈希漂移，未出现需要扩大依赖或停止局部审查的对象变化。

方案采用来源是用户本轮指令确认的持续授权与 Lead 采用；人的设计成果确认、接收责任人仍为 unknown/pending。报告不生成实际人、签名、accepted、checkpoint 或后续实施授权。

## 2. 实际加载规则

已按 UTF-8 完整读取以下有效正文，未用发现列表或父会话代替：

- `.codex/agents/add-reviewer.toml`。
- `.codex/skills/add-code-review/SKILL.md`，实际消费 design 分支。
- `.codex/guidance/code-review-checklist.md`、`abstraction-principles.md`。
- `.codex/guidance/design-doc-writing-style.md`、`decisions-guide.md`。
- `.codex/skills/add-code-review/references/产品安全红线.md`。
- `.codex/guidance/observability-standards.md`、`tracing-standards.md`，用于公共错误、根因、隐私与 trace 合同。
- `.codex/profiles/go-service/README.md`、`rules/security-coding-standards.md`。后者 §1 明确只在准备修改 Go 时适用；本轮依用户要求读取，未将其编码检查命令门禁强套设计评审。

仅采用已启用 go-service 的相关边界，没有加载其他 profile、前端规则、未来 Workflow、实施/测试规划方法，没有重路由。未遇到需要改架构/边界/ADR 的冲突，故未扩读架构、article boundary 或 ADR 正文；身份文档与 ADR-0025 完全排除。

## 3. Round 1：真实增量审查

已对两个 before-image 与固定当前字节生成并逐段阅读 unified diff。design 变更覆盖版本/来源记录、AM-A15、§3.2 交互、§3.3.1 DTO/cursor/接口总表、Story2/3 交叉引用、§3.3.9 完整新读取设计、§6.1 验证层、§8.6 影响面、§9 映射及确认记录。stories 覆盖版本/共享依赖及 Story2/3/8 的全部变更；其余 Story 正文保持基线，不作全量重新评审。

核对需求 `requirements.md:173–198/324–332/351–383`：管理员查看隐藏违规内容及保留其他隐藏约束来自 US-IDENTITY-03.12，普通删除前读取支撑 US-IDENTITY-03.2；成员分页与作者最小投影分别由 Story2/3 承接。没有新增业务写、申诉、状态 authority 或 US。旧作者受控详情取舍沿既有有效基线保留，本轮没有据早期原始表述重开该卡。

实际检查接口签名、typed view 与 HTTP presence、五种 cursor view、12 字段/嵌套 author/null/空数组、same-Article version/ETag、非法输入与状态、权限次序、异常收敛、部署复用、并发读取语义及验证承载。新增时序图和扩充组件图与文字责任相符；仅静态核到全文 28 个 PlantUML 围栏各含一对 start/end，未渲染，不给图渲染 PASS。

Round 1 没有把新 GET 尚未实现、test-plan 尚未更新、未跑 PG/测试或人的签名 pending 本身列成设计缺陷。HTTP presence 的可实施性疑点转入 Round 2 实证，形成 F001。

## 4. Round 2：限定消费者与上游依据

| 检查维度 | 实际消费者证据与结论 |
|---|---|
| 安全/身份 | Handler 的现役管理员入口先取可信 Principal，再核 `AccessGeneral`/admin；service 可复用 `requireOrphanAdministrator`。新最小接口不扩大旧 Handler 构造能力。root 同一 authorizer/Session store 与 Trace 的接缝存在；无需 initial-admin 标志。现役测试源码区分 GET 与写入口的 CSRF/Origin，不将 spy 当真实新增 endpoint 安全验证。身份 Gate/readiness 的完整实现不在授权依赖内，本轮仅核相同接入点及设计保留要求。 |
| 身份失败完整性 | 实读 `readAuthorStates` 会把 nil map 转为空 map，`ArticleIdentityReader` 通用合同允许 deleted 且不保证每个请求 ID 都找到；r19 因此在新方法中逐 ID 只接受 active/suspended，其他整请求 dependency 503。pending 候选先排除、source 不作为当前 author、匿名不查身份，能用原端口实现，不要求改 identity reader 或旧成员缺项行为。 |
| 分页/SQL/view 隔离 | 现役候选 SQL 不预过滤业务 view，绑定时间/UUID/LIMIT 且按时间、UUID 倒序；可以承载三 view。现役具体 repository 目前只收 member cursor，r19 已明确在同一 `query_service.go` 扩 allowlist。现役 `articleCursorValue` 硬编码 member，稿件已明确新方法内部 cursor 必须保留管理员 view。limit+1、last-returned 公共 cursor、last-scanned 内部位置、空批继续/耗尽、严格排序异常与无 partial 均可映射现有底座。 |
| hidden/投影 | 实核 `AdminArticleView` 为恰 12 个顶层字段，Policy 是隐藏原因唯一生成方。all 包含匿名/held/suspended/violating，hidden 与 violating 可以重叠。新增 admin_hidden 的模型、codec 与 repository 接缝都在五生产文件内；orphan SQL/方法和既有四 view 的含义保留。 |
| 状态时序 | 授权后非法 ID/无行/pending 详情先 404；合法 active 才读身份。canonical 删除在 Article 与身份之间发生而得到 deleted 时 503/客户端新请求；已读后并发变化、跨批/跨页漂移明确不承诺长快照或线性一致。不存在额外写入者、锁、自动重试或恢复状态。 |
| 资源/错误 | 现役 repository 与 identity reader 均 defer Close，rows.Err/Scan/Query 可由既有边界转为 DB/dependency 503。批大小与输出 limit 可约束在途数据，扫描耗时随批次数增长，容量/P95 保持 TBD；不补造扫描截断、配置数值或 worker。HTTP 参数解析存在 F001。非法 Article 的 500 需注意旧 sentinel 优先映射，见 N001。 |
| 兼容/root | 实读唯一 root 全 switch：当前 11 项包含 member2、DELETE3、moderation2、orphan2+publish/transfer2，普通 admin list 无分支、detail 仅 DELETE。现役 builder 有同 pool/repository/reader/service 与受控 test seam；扩 slot 并保留原映射、detail GET/DELETE 共存与 Allow、子路径精确匹配可在 main.go 内完成。另三项 root 接线保持 S1.6 的独立责任和共享文件串行 owner 条件。 |
| 验证边界 | 现有 query/cursor/Policy/orphan/PG pagination/root 测试源码可提供回归基础，均不是新 Admin2 的运行证据。r15 仅取 A-CON-01～04、A-REC-01～03 与 S1–S16 责任作为旧风险输入；没有要求其先更新或 PASS。 |

**§8.6 文件范围结论：五生产文件和四测试候选在设计层足够。** Repository 具体读取实现/SQL 本来就在 `query_service.go`；`repository.go` 不需要加读取职责，身份 reader/迁移不需要改。新单元文件可同时覆盖 service 与 HTTP；codec 新覆盖可写入指定 `policy_test.go`，现有 `cursor_test.go` 作为旧回归保留；PG→HTTP 与 package main 的真实私有 factory 证据分开。F001 的新管理员入口解析修订仍可限于 `handler.go` 和指定新单元文件，无须扩大生产写集。此结论不等于九文件任务已经登记、分配、实施或验证通过。

## 5. Findings

### [MEDIUM] F001：指定的参数助手会把显式非法值或重复参数误判为遗漏/单值

- finding ID：`AM-DES-R19-ADMIN2-R01-F001`。
- 对象/修订：design r19，伴随 stories r08。
- 主位置：`.add/project/knowledge/design-workspace/article-management/design.md:1977`；相邻转换合同为 :1978–1984，Story8 验收为 `stories.md:209–210`。
- 违反依据：r19 明确只有 HTTP 未出现 view 才用 all；重复/未知 view 为 422 且零 repository/identity 调用，非法 cursor 为 400、非法 limit 为 422。适用 `code-review-checklist` §2/§3、设计写作 §J 的可实施接口与失败合同。
- 现状：design :1977 指定用 `singleArticleQueryParameter` 的 present=false 识别遗漏；该助手实际 `internal/article/handler.go:679–693` 用 `request.URL.Query()[name]`，:686 丢失 query 解析错误。`articleQueryLimit` :696–711 同样消费此助手。它只会拒绝已成功解析到 map 中的重复项，不能恢复被丢弃的参数。
- 触发及证据：在 Temp 用标准库 `net/url` 探针执行与助手一致的 `url.URL{RawQuery: raw}.Query()`，并对照保留错误的 ParseQuery，输出如下。未加载或运行项目服务/测试，也没有伪造新 GET 的 HTTP 执行结果。

```text
raw="view=%ZZ"                       Query=map[]                 viewPresent=false  ParseQuery=invalid URL escape "%ZZ"
raw="view=hidden&view=%ZZ"           Query=map[view:[hidden]]    viewCount=1        ParseQuery=invalid URL escape "%ZZ"
raw="view=all&cursor=%ZZ"            Query=map[view:[all]]       cursor absent      ParseQuery=invalid URL escape "%ZZ"
raw="view=hidden&limit=%ZZ"          Query=map[view:[hidden]]    limit absent       ParseQuery=invalid URL escape "%ZZ"
raw="view=hidden;view=violating"     Query=map[]                 viewPresent=false  ParseQuery=invalid semicolon separator in query
```

- 后果：按指定的 presence→typed 转换，第一例得到 all，第二例按合法 hidden 单值通过；cursor/limit 的例子分别回到首页/默认页大小。错误请求因此进入 repository/identity 并可能返回成功页，违背输入拒绝及筛选/分页合同。这不是非管理员越权，也没有证明已经部署的 Admin2 存在漏洞；是新设计复用接缝与承诺之间的确定缺口。
- 最小修复方向：原作者在新管理员 Handler 的转换合同中补充 error-aware 的 RawQuery 解析与拒绝次序，在确认完整解析成功前不得把参数缺失作为默认值依据；继续使用既有公开错误码，说明 view/cursor/limit 解析失败的归属。可以只在新入口增加本地前置解析，不修改旧 member/orphan 助手的消费语义，不要求新 service/SQL/字段或重开旧卡。同步 Story8 验收与新单元层覆盖说明。
- 复审验证：给修复后的精确 design/Story 修订和哈希；静态追明原始 query 解析错误不会变成遗漏或被部分成功 map 掩盖。后续实施测试须经实际 Admin Handler 覆盖上述非法编码/混合重复参数、正常遗漏默认、单个合法 view、显式空/重复/未知、错 view cursor，并断言拒绝时零 service/repository/identity 调用；不能只测 typed 零值或枚举 switch。本轮设计复审不以尚未实施测试为前置。

## 6. 非阻断注意与待决定

**N001，实施注意，非独立缺陷：** design 已明确非法 active Article 状态应为 500。现役 `MapArticleError` 在 `handler.go:611–612` 优先按 `errors.Is(ErrArticleInvalidState)` 映射 409，早于 default 中保留 PublicCode；不能假定在该 sentinel 外包一层 internal PublicError 就自然得到 500。新读取实现须兑现已写的 500 合同，并在既定 HTTP 单元层验证真实 `WriteArticleError` 输出，而不只断言 service 的 PublicCode。稿件未强制直接返回/包装该 sentinel，局部新入口可以满足合同，故不把尚未编写的错误实现预判为第二项缺陷，也不要求改变旧治理 409。

待决定项与缺陷分开：本轮不重新选择已采用的 service/SQL/身份故障/分页方案；没有新增需要 Reviewer 代选的架构方向。实际成果确认人、接收人及后续精确实施授权仍 pending，由 Lead/用户处理，不伪造签名。新旧 main.go/main_test.go 的串行文件 owner 仍是交接条件，本报告不作认领或规划状态决定。

## 7. 实际 Readset 与证据范围

下表记录正文读取范围；符号/文件定位搜索不等于全文读取。所有源码和测试均只读；没有展开无关恢复 producer、SMTP、身份文档或全仓消费者。

| 文件 | 实际读取 |
|---|---|
| Reviewer/Skill/Rule/Profile | §2 列出的各文件完整 UTF-8 正文 |
| `baseline.json`、design/stories before-image、固定 r18 | manifest 全文、before-image 全字节及相应 diff、固定 r18 哈希 |
| article `design.md` r19 | 全部实际增量；相邻共享 DTO/错误/接口总表 :880–1025；Story8 :1817–2004；正交状态 :2008–2025；§8.6 :2717–2747；确认/历史 checkpoint 的增量上下文 |
| article `stories.md` r08 | 相对 r07 的全部实际增量及共享依赖、Story2/3/8 上下文 |
| `requirements/requirements.md` | :173–201、:324–397，限定管理员、浏览、编辑来源 |
| article `test-plan.md` r15 | :360–390、:490–515；仅旧风险输入 |
| `internal/article/query_service.go` | 全文，包含成员/作者/orphan service、repo、SQL、排序、分页、错误助手 |
| `internal/article/policy.go` | 全文，包含状态验证、投影、hiddenReasons、cursor strict codec、ETag |
| `internal/article/model.go` | :19–112、:211–380，状态枚举、DTO 与 cursor；其中旧恢复结构仅随相邻范围读取，不作本对象评审 |
| `internal/article/handler.go` | :1–118、:180–225、:235–264、:298–379、:590–640、:671–780、:841–928，另精确符号定位 |
| `internal/identity/article_reader.go` | 全文；唯一实际读取的身份实现文件 |
| `cmd/han-server/main.go` | :398–431、:490–610、:2214–2275、:2339–2430、:2644–2689、:2826–3016、:3211–3440，另 query config/authorizer/trace/recovery 符号定位；没有改身份实现 |
| `internal/article/query_test.go` | :1–417 断言正文；后续 helper/fake 符号与状态定位 |
| `internal/article/cursor_test.go` | 全文，旧四 view 的 round-trip/错误/隔离断言 |
| `internal/article/policy_test.go` | :92–282，字段隔离、管理员/作者权限与 pending |
| `internal/article/orphan_disposition_test.go` | :149–313，orphan 查询/HTTP/SQL 回归边界；其他内容仅测试名称定位 |
| `test/integration/article/pagination_test.go` | :1–177，实际成员候选扫描、SQL 分页/故障/跨页变更测试源码 |
| `cmd/han-server/main_test.go` | :1014–1128、:7663–7925，旧入口安全、共享依赖和 root 路由断言；另相关测试名称定位，末段恢复测试仅相邻读取不纳入本评审结论 |
| `research-s1.5-readiness-r01.md` | 相关现状/接缝/边界/Readset 定位与 :150–176 摘要表；r01 十三字段旧计数不采用 |
| `research-s1.5-readiness-r02.md` | 全文及哈希，实际 model/Policy 另行核到 12 字段 |

以下为现场源码/证据哈希，共同源码与研究 r01 所列摘要一致；末核读取集合保持一致：

```text
internal/article/query_service.go 9674130673CD05E8FA864B77528FDBF12C9AB8305AE6AD2EF7E2A85DB9136E7E
internal/article/handler.go A0921B0178551A8EA51772ECF9823B50AA693DD3B450FDD347257B6F77CCC7E8
internal/article/model.go 3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4
internal/article/policy.go B5A199340C18B505B8F2633715C9EC63B78910FB684332061CCE0258E12FC980
internal/identity/article_reader.go D9315AFA411CB8C4612459C8E97979BD2274780A7B7B32D78F3AB735C3F82B6B
cmd/han-server/main.go D6A82DBDD9C72D3A2435A32F1059494ABEA94E5F3B05ED1B5C93C52FAF97AFB4
cmd/han-server/main_test.go 0346F1AA96C6EDDB6409148E648D37FD15448F2C73AC966BA18AFE3ADA3DBA68
requirements/requirements.md 999CBA3C0CD25D2760E400A41D2A373611406854E76E24E453321913D71CBC4F
research-s1.5-readiness-r01.md 77C32EC9ED2AAA14C2D1D6DBD48E7584715B9976B5C2D66D1339F8F9186B509A
research-s1.5-readiness-r02.md C6B333DDD649D224EA8DCCAE02BFAFC05D0B00F15ECAE4E0D49D8C329EC9177A
```

Temp 分析文件位于既有基线目录：`review-r01-design.diff`、`review-r01-readset-hashes.json`、`review-query-parser-probe.go`。探针只运行 `go run <Temp>/review-query-parser-probe.go`，无项目 import、无网络/DB 请求；不作为项目测试 PASS。Python 仅用于哈希、字节/diff、行号摘录及围栏静态核对。Git 仅运行 status/branch/rev-parse；未运行项目 format/lint/build/test/扫描命令。

## 8. Gate、未验证与交回

| Gate/结论 | 状态与边界 |
|---|---|
| 固定对象/基线 | 匹配；实际工作树读集末核无漂移 |
| 两轮独立设计评审 | 已执行；F001 未关闭，CHANGES REQUIRED |
| 核心方案采用 | 按本轮用户/Lead 授权保留，不等于人的成果签名或风险接受 |
| 五生产/四测试候选可承载性 | 本轮静态核足够；不是 Task/Plan 登记或实现证据 |
| 人的成果确认/承接确认 | unknown/pending，不扩用 r03/r06 checkpoint |
| 测试/PG/root 新 GET/部署/E2E | 未执行、未验证；无 PASS |

未验证范围：新增 service/Handler/cursor/root 的真实运行、PG SQL 与身份故障注入、取消时资源释放、完整恢复 Gate/身份会话内部实现、真实 Host 启动和容量、浏览器/部署/E2E、图像渲染，以及其他未列出的消费者。没有因本增量重开旧有效卡，也没有重新裁定旧风险豁免或身份 ADR。静态可实施结论只覆盖本报告直接证据范围。

交回原活动 Lead：请由原作者修正 F001，提供原 finding ID、修复后的精确修订/哈希与有界差异；原 Reviewer 再核参数解析/Story 同步、五生产/四测试边界及新引入风险。本报告保留原失败链，不由 Reviewer 代修或改任务状态；Lead 还须现场复核成果、所有权与下一动作授权。

**stopped：R01 报告完成，Reviewer 停止；等待原作者修订后独立复审。**
