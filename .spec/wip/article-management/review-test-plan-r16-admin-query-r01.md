# Admin2 Test-Plan 独立评审 R01

日期：2026-10-27。Reviewer：Pauli，独立于原测试规划作者 Ohm。objectType：`design`（测试规划合同）。本轮是新的有界 TestPlanReview，不沿用设计评审结论替代本轮判断。

**结论：PASS，仅固定 test-plan r16 的 Admin2 增量。无 findings。** 两轮实际审查完成，20 个 Case 的风险、验收、数据、分层、候选文件/符号、负控与命中门禁能够承载固定 design r20 的新增读取合同。全部 A-ADM2-01～20 仍为 NOT RUN；本结论不是代码、Case、PG、root、部署/E2E 或人的 accepted/checkpoint PASS。

## 1. 固定对象、依据与权限

工作区 `D:\AID\VSCode\Codex\Han`，实际只读核验分支 `main`、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。唯一被审对象为下表 test-plan；design/Story/Design R02 是固定上游依据。

| 文件/修订 | 实际 SHA-256 |
|---|---|
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` r16 | `6941C1D7B96F78B341CBB7B88AE74D548CF860CFEAD37D3D92E208548D896E38` |
| 同目录 `design.md` r20 | `6884431B6164140E5593A8253AC20E8EB1C59C1AD71B86558B6D1A8C99BF037A` |
| 同目录 `stories.md` r09 | `A2BA7746F30B1C9B60DB4337995FC0A0E52DD55BFF984BAA784C20331D80C569` |
| `.spec/wip/article-management/review-design-r20-admin-query-r02.md` | `5B2E39D0A491B173BB651AA20E966BE40A0E03DFC1645E8D3CFFB0594B9E62B0` |

交接目录为 `C:\Users\J00010385\AppData\Local\Temp\han-admin2-testplan-r16-6fc57d5e766e4d17baef3c93eec2c95e\`。实际读取 `handoff-r16.json`、`test-plan-r15-before.md.txt`、`test-plan-r16-fixed.md.txt`、`r15-to-r16.diff`、`static-r16.json`、`r20-prep-refresh.json`。before-image 实际 SHA 为 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`；fixed r16 副本与现役目标逐字节相同。独立重算的 unified diff 与交接 diff 文本相同：**142 行新增、11 行元数据替换，未删除旧正文/Case**，没有把作者的静态声明直接当作评审结果。

当前 Role、`add-code-review` 完整有效正文，以及原会话已完整取得的 checklist、abstraction、design-writing、decisions、产品安全红线、可观测/追踪及 go-service 适用边界继续复用，按 design 对象检查。因本轮明确切换有界目标，读取 `add-route-work` 完整正文，仅定位到独立专业评审，不推进研发 Workflow、不加载未来实施/E2E 方法。

本轮唯一仓库写入为本报告；Temp 只写 Reviewer 自己的读取摘要。未修改 test-plan、设计、Story、Tasks/Plan、源码、测试、fixture、配置或 ADR，未执行项目测试、PG、SMTP、服务、Browser、下级委派、Git 写操作或任何清理。没有读取/注入 DSN。人的接收/签名仍 pending，不从持续采用建议的授权推断 accepted/checkpoint。

## 2. Round 1：完整增量与验收映射

逐段审查 r15→r16 的全部真实增量，包括来源元数据、§1 范围、R-A20/21 风险、§4.3.1 的全部 20 Case/HTTP 对照、§6 映射、§7.7 数据环境、§9.6 九文件与命中/负控门禁、§11.1 Gate 和当前结论。历史正文只核保留及新旧结论隔离，不重做全特性历史评审。

| 固定设计/验收与风险 | r16 Case / 位置 | 规划充分性判断 |
|---|---|---|
| 普通有效管理员、两级授权、GET 安全方法与接入限制 | 01/18/19，:270/:287–288 | U 核 Handler/service，R 核唯一 factory/接入；没有强加 initial-admin、CSRF 或 query 角色来源 |
| F001：完整 RawQuery、422 优先、decoded cursor 400、默认与 typed 零值分离 | 02，:271/:293–303 | 经真实 Handler 设置 RawQuery；五原反例及未消费键错误、编码同名键重复、解码成功后的 cursor/limit 对照齐全；拒绝时零调用要求明确 |
| 五 cursor 隔离、三 view 筛选与复合 keyset | 03～07，:272–276 | 五种同 view 成功与 20 个有向跨 view 拒绝；至少三时间戳/ID 交错、同时间平局、连续过滤多批、last-scanned/last-returned、lookahead 与耗尽均有独立期望 |
| 当前事实与严格身份完整性 | 08～11，:277–280 | 真提交/barrier、第二批故障整 503 无半页、非目标 view 候选仍核身份、仅 active 非空 author 去重；动态漂移不套静态不漏不重 |
| pending/无行/非法 ID 优先 404、非法状态 500、SQL/取消边界 | 12～14，:281–283 | 合法 pending 可先命中 404 且零 identity；非法状态走组件/SQL 后 typed fault；Query/Scan/Rows.Err、原 context、关闭 rows 与停止后继扫描均有承载 |
| 零业务写、12 字段独立值 oracle、同源 ETag | 15～17，:284–286/:291 | 快照对照与 fixture writer delta 分离；完整键/字段值/原因顺序/null/[]/匿名 author/Unicode/时间精度明确；不使用被测 Policy 重算期望，不拼两次 Article 读 |
| 同一依赖/唯一 root、精确 method/path/Allow、旧消费者与 N001 | 18～20，:287–289 | R 构造/安全/路由与 P SQL→Handler 分开；新读取真实 HTTP500 与旧治理409、旧成员缺项/故障、作者四字段/orphan 独立行为分别保留 |

来源 design r20 的新增只读合同与 Story2/3/8 验收均有相应 Case。设计 F001 仅为设计层 CLOSED，N001 仍须在后续实现取得 HTTP 输出；r16 没有借用原 Design R02 为本测试计划自评或运行签发。

## 3. Round 2：真实接缝、数据与执行有效性

### 3.1 四层/符号/命中

独立提取 §4.3.1 映射，确认 ID 恰为 A-ADM2-01～20、无缺号/重复定义；有 **38 组 Case×层映射：U 17、C 1、P 16、R 4**。这只是规划映射计数，不是执行节点或覆盖通过数。

| 层 | 精确文件/候选 top | 本轮核查与证据边界 |
|---|---|---|
| U | 新 `internal/article/admin_query_test.go` / `TestAdminQuery` | 当前新文件不存在，依计划待实现；普通/race、真实 Handler RawQuery/错误输出及 service 单独授权均须命中新 leaf |
| C | 改 `internal/article/policy_test.go` / `TestAdminCursor` | 实际旧文件仅有现役 Visibility tops，尚无该新 top；可在此文件承载 codec 五 view/20 对，旧 cursor/投影保留，不要求改另一个 codec 测试文件 |
| P | 新 `test/integration/article/admin_query_test.go` / `TestAdminQueryPostgres` | 当前新文件不存在；同包现役 fixture 可访问，integration build tag/package 有直接依据。须真实 PG/reader/service/Handler，不能用 SQL spy 或手造路由替 production 注册证据 |
| R | 改 `cmd/han-server/main_test.go` / `TestProductionAdminArticleQuery` | 本轮不读其当前源码，按固定设计的私有 factory/observer 合同承接；新 top/leaf 为待实施候选，不借 S1.6 已有执行节点证明新 Admin2 已存在 |

§9.6 :1102–1126 先静态核四文件/签名/P tag，再以精确 `-list` 和普通/race JSON 验证必要 top/leaf，禁止零测试、缺 leaf、只旧测试、skip 或 `[no tests to run]` 假绿。P 普通/race 命令均带 integration tag，局部 race、默认全包、vet/七项 Manifest 不能替代 P。命令为未来验收模板，本轮没有执行，也没有要求尚未实施的符号先编译 PASS。

必要的旧回归前缀已与 `query_test.go`、`policy_test.go`、`update_test.go` 的实际现役符号作静态对照；root 回归符号留待最终接收时核对，不边写边审 root。最终命名若变须回填 Case→真实 leaf，不能以本报告的映射计数代替实际 JSON 声明/命中证据。

### 3.2 PG fixture/约束/资源

实际读 `notice_dispatcher_test.go:1–190` 与 `postgres_state_test.go:1–245`：同属 `articleintegration`、带 integration tag，现役 harness 具有 loopback:5433/版本核验、每连接 runtime search_path、0001～0009 迁移、随机 schema、rows/tx/pool 资源模式。读取代码中的环境变量名不等于读取其值；本轮未连接或核验当前 PG 实例。

实际读 `migrations/0003_article.sql:1–180` 的必要约束：Article 作者/source 指向 users；held/source、violation/category、pending/operation 状态有 CHECK，pending 的复合 FK 要求已存在对应 operation；operation 不要求先有 Article FK。因此 §7.7 :774 的合法 pending 先建 operation 可实施。S2.2 现役 insert/helper 也按 operation→Article 事务顺序；deleted 墓碑 seed 与真实 reader后的 fault/barrier 可避免物理删除 FK 目标。非法组合不需要关闭 CHECK/trigger。

§7.7 已要求完整迁移、各连接一致 search_path、随机且先提交 schema、失败 schema retain/Case 关联、无 DROP/清理、故障仍委托真实 SQL/rows 关闭、取消/join/pool 归还。SQL/fixture 值 oracle 与业务快照不泄露正文/身份数据；动态 writer 的变化单独断言；不将 fixture 治理写或审计写误归为 GET 副作用。

**载体实施注意，非 finding：**现役 `newNoticeHarness` 的 :110–123 把 CREATE SCHEMA 和全部迁移放在同一事务，且 :106 在迁移成功后才日志记录；不能把原样调用它当作已经证明“先独立提交 schema、迁移失败仍保留并可追踪”。r16 明确由新 P 文件内承专属 fixture，旧 harness 仅候选只读复用，因此可在原新文件内落实所需创建/记录边界，不必改旧 fixture 或新增第十文件。计划的强制数据合同已足够明确，本注意不构成设计返工或测试已失败的结论。

### 3.3 最小负控/独立 oracle

§9.6 :1100 的检错证据指向实际行为失败：忽略 RawQuery 错误须由真正 HTTP 错误成功/误调用断言抓到；补 active/成功半页须由第二批 PG/reader 故障抓到；删 older-timestamp 分支或误用公开 cursor 须由三时间戳/多批序列抓到。对应正例/期望在 02/05/06/10 内有定义，不是为 mutant 临时写硬编码失败。

Temp 完整固定源码 snapshot/overlay 保留可编译 API、只替编译输入、当前源 GREEN、精确命令/退出码/断言/源测试 hash/schema 绑定均已规定。编译错误、缺环境、`t.Fatal`、伪 counts 不能作为 RED；原工作树不被负控修改/回滚。独立 SQL/显式 fixture oracle 逐项核集合和 12 字段，产品不因此增加 identity JOIN。以上是规划可行性，不是本轮已执行负控或确认其杀伤率。

### 3.4 所有权/边界

§9.6 :1080–1098 保持一个完整 Admin2 开发/局部验证单元，确需时仅分开发本地与真实 PG 责任窗口，不按 Case/文件拆碎。生产五文件和测试四文件与固定设计一致，任务 ID/owner/认领/START 待 Planner/Lead 实际登记；没有把九文件建议写成已获实施授权。

Dalton 当前 root 两文件独立交付与后续 review 返工不属于此单文件冻结。本轮没有读取或计算 `main.go/main_test.go` 当前哈希，不评审 Erdos/Dalton 的代码或其 271 节点/10 top/91 leaf/七项结果，也不向其加 PG/E2E 要求。计划保留同文件串行和最终 slot/method 表核对，不构造 S2.2 或 S1.6 的额外业务 requires。

原 S1.5 四 contract 文件、16 endpoint、实际应用身份 ACL、S1–S16、审计/producer/source 责任完整保留；局部 P/R 证据不代替它们，不继承身份 ADR-0025 的任何豁免。真实部署/跨服务 E2E、容量/恢复专项未倒挂为此次局部验证前置，也没有被四层局部证据宣称完成。

## 4. Findings 与待决定

**Findings：无，CRITICAL/HIGH/MEDIUM/LOW 均为 0。** 没有需要交 Ohm 返工的确定规划缺陷；不把当前缺新文件/新 top、Case 全 NOT RUN、尚未登记任务或人的 pending 当作错误。

N001（原设计的错误映射实施注意）已由 A-ADM2-20 承接，不再造新缺陷或关闭代码问题。上述 fixture 载体注意仅限定实际复用时的证据边界。

待决定/交接项：Planner 实际任务/九文件登记与 owner、root 串行窗口、后续 CODE/LOCALPG START、实际成果接收责任和人的 accepted/checkpoint。这些仍 pending，由原活动 Lead/用户处理，Reviewer 不代选或签字。

若未来改变固定设计、错误优先级、Case 分层、fixture/约束策略、九文件边界、命中/负控门禁，须给新修订与有界差异再审；若实际执行发现问题，则保留失败证据并回相应作者，不以当前规划 PASS 消除它们。

## 5. 本轮实际 Readset

| 文件/材料 | 实际读取范围 |
|---|---|
| `test-plan.md` r16 / r15 before / fixed r16 副本 | 全字节摘要及全量 actual diff；逐段阅读所有新增与元数据替换；带行号复核 :264–303/:770–778/:1078–1128；其余旧正文只作保留比对 |
| handoff/static/diff/refresh | §1 交接目录所列四份 JSON、完整 diff；独立重新生成 diff/Case 定义与层映射，不执行作者脚本 |
| `design.md` r20 / `stories.md` r09 | 固定摘要；本轮正文 design :1853–1875/:1975–2001/:2736–2759、stories :201–221，其余相关既有读取结果复用 |
| Design R02 | 固定 SHA 与同会话完整有效报告复用；只作为 Admin2 设计依据，不作为本次评审结果 |
| 原 Temp `han-admin2-prep-2eae66abc2c1407f98dc7f3dcb2f3914/prep-admin2.md` | 完整正文，核历史准备与 r20 refresh 的差异/候选范围，不采其历史状态为当前任务状态 |
| 同目录 `source-before.json` | 完整登记；只对非 root 当前直接源码/fixture 核摘要，不冻结表中 root 旧哈希 |
| `internal/article/{query_service,handler,model,policy}.go`、`internal/identity/article_reader.go` | SHA 与原直接证据一致；复用同会话已读接缝；本轮 query_service 另读 :380–419/:793–799 |
| `internal/article/query_test.go`、`policy_test.go`、`update_test.go` | 实际测试声明/必要符号与哈希，沿用原已读断言；C 新 top 静态核未出现 |
| 两个新 admin_query_test.go 候选路径 | 只读存在性核验，均不存在；没有虚构编译或执行结果 |
| `test/integration/article/notice_dispatcher_test.go` | :1–190，harness/config/migrate/cleanup 接缝及摘要 |
| `test/integration/article/postgres_state_test.go` | :1–245，fixture/User/pending/请求与资源边界、测试声明及摘要 |
| `test/integration/article/pagination_test.go` | 本轮仅相关符号定位/摘要，复用原会话已读分页测试边界，不运行旧 S2.2 |
| `migrations/`、`0003_article.sql` | 文件名定位完整 0001～0009；0003 正文仅 :1–180 的 fixture 所需 FK/CHECK/operation/recipient 定义，不评审全迁移 |
| Role/Skill/Rule | 原会话完整有效正文按 §1 复用；本轮另完整读取 Route，仅定位新有界目标 |

定向读取的五个非 root 源码及 query/policy/update/notice/postgres_state/pagination 测试摘要均与作者 prep 的相应固定输入一致，没有未知 writer 变化；0003 必要片段首次读取时记录 SHA `FCAECE696DCE29D2F8D252EA36C4550F0C1736347E921F40703540B99E100D4A`。Reviewer 自己的 `review-r01-readset-hashes.json` 保存本轮冻结检查集合，排除两个 root 文件；交回前重新核验对象/上游/所查非 root 依赖无漂移。

实际工具只有只读 Git branch/HEAD、文本/符号/路径查阅、Python 字节/哈希/diff/映射静态分析及写本报告。未执行任何 go test/go vet、Case、mutation、PG/SMTP/服务/Browser；未查任务正文、身份文档/ADR 或无关全库消费者。

## 6. Gate、未验证与停止

| Gate | 本轮结论/边界 |
|---|---|
| r16 与固定来源/基线 | 匹配，actual diff 已独立核；未删除旧 Case 正文 |
| r16 Admin2 独立 TestPlanReview | PASS，两轮完成，无 findings，仅本有界增量 |
| Design R02 / F001 | 原 Admin2 design-only PASS/F001 设计 CLOSED 保留，不转成 Case/代码 PASS |
| A-ADM2-01～20 / U/C/P/R | 全 NOT RUN；符号、命令、故障/负控、资源/数据证据待实际实施执行 |
| Task/owner/串行/后续授权 | pending，由 Planner/Lead 核真实登记和窗口 |
| 人的 accepted/checkpoint | pending，未代签 |
| S1.5/ACL/producer/部署/E2E 等 | 原责任保留，本轮未验证，不因本次 PASS 缩减 |

未验证风险：新 Admin2 真实实现、候选 top/leaf 的最终声明与 JSON 命中、真实 PG 数据/屏障/Scan/Rows.Err/取消退出、负控实际行为 RED、12 字段独立值 oracle 与同源 ETag、真实新 root 接线/安全链及与 S1.6 最终代码的兼容。当前环境与 DSN、服务版本/可达性、实际应用 ACL、Host/部署/E2E、容量与图渲染均未执行或验证。

交回原活动 Lead：可将此固定规划作为后续登记/接收输入；独立 TestPlanReview PASS 不代替人的接纳、CODE/LOCALPG START 或下一动作授权。若后续需修订，由原 Ohm 修复后以新 hash/差异交本 Reviewer 有界复审，保留本报告与历史失败链。

**stopped：Pauli Test-Plan R01 评审及报告完成，停止。**
