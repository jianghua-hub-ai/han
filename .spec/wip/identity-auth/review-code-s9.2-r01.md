# identity-auth-S9.2 独立代码评审 R01

最终结论：**Code Review Gate: PASS**。仅针对固定 source183-r02 的两份新增测试：两轮独立审查完成，独立普通64、race64、tagged vet 及 retained SQL 文件核查通过；确认 finding 为 0。Reviewer 已 **stopped** 交回 Lead，不接下一任务，不关闭 Task/Issue 或代人的 accepted。

本报告由本会话独立 Reviewer 形成，Reviewer 不是 Rawls 作者；没有子委派。当前活动为 development-delivery / code-review。仓内唯一写入为本报告。静态阶段未运行 Go/PG 或读取 env/DSN；收到 Lead 明确独立 LOCALPG START 后按序实际执行普通64、race64、tagged vet，结果见下文。未改作者代码、任务、计划、设计、Manifest、依赖、迁移或其他 metadata；未执行 SMTP、部署、外部网络或 Git mutation。

## 固定对象与依据

- objectType：code；完整 new-file diff 为两个新增 integration tests，非 HEAD 到工作树全部内容的总评。
- 工作区：`D:/AID/VSCode/Codex/Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，已现场核对。
- `test/integration/crossmodule/canonical_user_delete_test.go`：1259 行；SHA256 `2C88891DCD6F2E7F2B1B256492679FC6F4341F2F0816451D38E5841E3FE9FA20`。
- `test/integration/crossmodule/canonical_delete_faults_test.go`：456 行；SHA256 `AADA00B5314FD6CE312B84547E9DF20FEDE09CA97432A8822A0EA68264F8BF99`。
- 183 输入平坦清单：`C:/Users/J00010385/AppData/Local/Temp/han-s92-lead-guoi9beq/source183-r02.flat.json`；文件 SHA256 `03594FB126EB3E2493A14876E33AE5C1475E2F66C908BDB3D484B6E61CCBB73E`。
- 源码实际读取自同目录 `fixed183-r02/`；每次源码读取批次前后均核 live/snapshot 全部 183 路径与字节哈希、main/HEAD。独立核对 protected181 与清单完全一致，两目标外无源差异。161 Go overlay 不是全部输入；后续执行 cwd 固定完整 snapshot，使运行时迁移读取也来自快照。
- identity design r45：`.add/project/knowledge/design-workspace/identity-auth/design.md` §3.3.10 / §4.2，SHA256 `D57BB2152A93F6F651165EE7DBE1913B28E9E8719B6991B1FDD36E8B7DBAB7FE`；已读 2011–2107、3024–3116。
- Story S09：同目录 `stories.md:298–323`，SHA256 `057DE0D716D1583AFB487E152ACEE05F0FAFDAAB7B8FA412501E23F5EE8191F3`。
- test-plan §4.5 / I-DEL-01～07：同目录 `test-plan.md:488–507`，SHA256 `0052894D3C4B58F644749F8337BB264FD2768DD26A31C7EA0A361F0F61D939BD`。
- 原 S9.2 卡：`.add/project/knowledge/tasks/tasks.md` 的 `identity-auth-S9.2` 条目；读取时仍 `[-]`。原四 requires 沿已关闭合同复用，不新增依赖/假门禁。本报告不更新其状态。
- Lead 采用范围：`.spec/wip/identity-auth/evidence-s9.2/prep-r01/index.json`，SHA256 `652DA80EA6CCD440F10556DDCC4F415519684FCB2038F43399E2DFBE41968014`；已核其引用的 `adopted-test-scope.json`（SHA256 `3FD029F927F40002A44E6788A9CF8D07015EB0CA86355184AFC2315C30F726D1`）与 Zeno `testdesign-prep.md`（SHA256 `A7E1DC3C70E61508800C5A14E3AD9C331ACAD20129C092455F4BABF00C331E95`）。按 Lead 两 top / 六组及共享事务体 Daily 逐阶段、Recovery 特有 late failure 的最小组织核查，不强制笛卡尔积。

本结论只绑定上述 source183-r02 两目标版本。其余 181 文件为上游直接接缝依据，未重新对全库签独立 Review；S9.1 既有作者自审的来源和边界不因本报告改变。路径 B phone 已接受的 reserved/存在/未释放残余限制保持原样，不要求 retained_phone 或新 migration。

## Findings

确认缺陷：**0**（CRITICAL/HIGH/MEDIUM/LOW 均无已确认项）。结论合并下述两轮静态检查与本 Reviewer 独立真实执行，绑定本次两新增 tests 及固定输入版本。

未发现必须改变架构/契约的方向决策；无新增阻断建议。

静态核查中曾提出 `ProtectedInitial` 用例是否被普通管理员权限提前拦截的疑点，补查真实 schema 后已撤回 MEDIUM 定级及“另造 initial-admin actor”的建议：`canonical_delete_faults_test.go:181–186` 使用普通管理员，真实 `deletion.go:498–499` 会先拒绝治理 admin；`0002_identity.sql:49–51` 又只允许一个 initial admin，不应制造两个 initial admin 来追分支覆盖。该用例证明此合法调用被拒绝，不能宣称单独命中 `deletion.go:526` 的目标保护分支。此项列为覆盖边界，不要求作者修复，不计产品 Bug。

## Round 1：新增两文件直接检查

实际全文阅读两文件，逐项核行为、失败、权限、秘密、证据及资源所有权。

| 范围 | 静态结果与定位 | 静态阶段待验项（独立执行结果见后文） |
|---|---|---|
| 入口/选中 | 两文件均为 integration tag；无 init/TestMain、Skip；两个 top 匹配原任务筛选。author-code-r02/leaf-plan.json 有 64 个不重复 leaf：成功 13，故障 51 | JSON run/pass 必须逐 leaf 对账；top PASS 不代替 leaf |
| 建库前预检 | user test:161–240：主/全部 fallback 必须 literal loopback:5433；连接后核实际 SQL 地址、port、180006、user/database；CREATE 单独提交；0001–0009 迁移同 tx，随后 seed | 真正 preflight、九迁移与 seed 是否通过；失败即停且保留 schema |
| 合成数据 | :271–409：多个 Session；current pending_delivery/active/expired 分配到 Daily；历史 consumed/superseded；合法 pending operation；已 deleted 路径 B；真实 Store.Create 和 capability；受控 Gate 显式声明 | 约束接受、时间/精度、幂等回读 |
| 原子性 oracle | :470–505、626–734：11 表全列+xmin/ctid 镜像；独立独占 observer/PID；提交前 observer 旧像、caller 新像；失败后完整旧像；显式结束状态 | 真正 SQL 隔离与 rollback/no partial，包含取消 |
| 故障阶段 | :682–815；fault test:47–152：真实 Exec 成功后才定点 error；Audit 用真实 audit.Store INSERT 主键碰撞，检查 23505/audit_events_pkey。Article 第二次更新失败前有第一篇更新/审计；ReviewComplete 失败检查前序审计；LastTransfer 第二 recipient 已写 | 指定 hit、前置写、真实 SQLSTATE、rollback 与合法重试均实际发生 |
| 取消/并发 | fault test:320–456：独立短 context rollback；channel barrier，无 sleep 造顺序；NOWAIT 55P03；同来源两篇先退让再重试；owner release/join 后核最终仅一次 release | 普通/race 均无泄漏、超时、竞态；当前锁序的实际执行 |
| 隐私/证据 | user test:100–110、507–624：驱动错误只输出类型/SQLSTATE/表约束名；临时目录 retained；11 表 stage JSON；凭据列散列、NULL 保留；UseNumber；raw/redacted/file hash；O_EXCL、Sync、Close；无自动删除 | 实际文件、11 表完整性、散列及脱敏复核；没有运行就没有实际 evidence 落盘保证 |

首次路径集合 guard 将两份非输入 README 误计为输入而停止；随后按 Lead source_guard.py 的明示范围（源目录 .go/.sql/.json/.mod/.sum，加 go.mod/go.sum/migrations/README.md）校准后重新核验通过。该次失败为 Reviewer Temp guard 的集合定义问题，不是源码漂移或产品 RED，原记录保留。

## Round 2：沿真实生产者/消费者核对

1. **canonical/authority**：核真实 `CanonicalDeleteCoordinator`、`PostgresCanonicalDeleteRepository`、`RecoveryRunStore`、两侧 ArticleAccountPort。测试不伪造 Article 计数；原样传 caller ctx/tx。Recovery fixture 使用同一真实 Store 的 caller-tx FOR UPDATE；wrong backup、missing run/store/review、failed/identity_ready、Gate nil/closed/error、review decision/evidence 等在写前拒绝。maintenance/readiness 是受控输入，未被当作真实全局恢复证据。
2. **r45 身份/标识**：路径 A 存在性、逐值绑定、released_at 拒绝；路径 B email 错绑与已有墓碑；username/phone 永久占用；零文章同 tx email 清理。unknown kind 为实际 CHECK 23514 探针，不禁约束造伪状态。UNIQUE 探针避开 probe owner 的主键冲突，只证明存储占用，不冒称 Register 响应。
3. **Article 原生完成事实**：真实 PublishOrphan/TransferOrphan 产生私有字段 proof；测试适配只映射 reference/facts，不构造 proof 或 verified=true。r02 每 fixture 只建一个 `identity.NewArticleIdentityLifecycle` 和固定 verifier（user test:824–831），按 caller tx+audit ID 登记原 proof，核 source/article/preimage/action/trace，调用原 `proof.VerifyCompletion(ctx, tx)`（:869–915）；commit/rollback forget，cleanup 核无注册残留。未恢复 r01 每 cleanup 换 provider 的模式。
4. **retained_email/通知**：native proof 同 tx 回读 Article、Audit、Notice/两 recipient、剩余 active/pending；清理发生在真实完成事实之后。首篇保留、最后公开/违规转移释放、pending 继续保留、重放不回退已处置文章/重复 release；三类核心审计与处置审计失败均有真实 SQL 路径。测试只证双 recipient pending 快照，不声称投递。
5. **迁移/资源**：0001 append-only audit；0002 身份、标识唯一性、会话复合 FK、review、唯一 initial admin；0003 held/source/pending/operation/双 recipient deferred 约束；0005 run 绑定/转换；0008 digest/generation、legacy gen0/source NULL、FK/deferred counter；0009 replay attempt。测试不改 schema、不禁约束、不调用旧 DROP helper。透明 tx wrapper 用新短 context 收束取消；observer 释放、cancel、pool stat/Close 与 barrier join 路径已静态检查。

该轮没有把上游实现本身当作验收定义；期望同时对照 r45/S09/I-DEL 与已采用 scope。旧 181 全库、生产 root 装配及真实恢复主线均未在本次重签。

## 两项 r02 修正的核实

- **固定 lifecycle/verifier：实现与本次独立执行均已核实。** 构造一次、注册原 proof、同 tx 原样验证、结束即 forget 的路径见上述 Round 2；不是从公开 DTO 自造完成权威。普通/race 的每 fixture 固定实例记录、原生 verifier 调用、资源结束断言通过。
- **11 表持久脱敏镜像：实现与本次独立执行均已核实。** 所有 cdObserve、precommit observer/caller、成功提交、失败回滚、合法重试、并发结束、resource_close_final 接入 retained 文件。两次独立运行共954份实际 stage JSON、10494个表级脱敏 rows 哈希已逐一重算；原始 SQL 全像仅在执行时内存作严格比较。文件保存的 raw image digest 不能单独还原或重算原始秘密值，脱敏 rows/file hash 与原始内存相等断言的证明边界分开。

## 静态阶段已有证据与实际检查（历史记录）

已读作者 r02 `handoff.json`、`static-raw-meta.json`、64 leaf plan，以及 r01→r02 差异的相关部分；其 gofmt/list/compile-only exit 0、零测试体/无 PG 为**作者记录**。本 Reviewer 没有运行 Go 去重做这些检查，也没有将作者的记录当作独立动态 PASS。r01 候选与 compile/step1 错误转录保留在 Lead Temp 根，未改写、删除或称为产品 RED。

静态阶段 Reviewer 实际执行：只读 Git HEAD/branch/status、文件定位/内容读取、Python SHA256/路径集合核验、protected181 比对、leaf plan 唯一性核验、Temp guard 的 Python AST 与运行计划 JSON 语法核验、完整 new-file diff 生成。当时所有测试体、独立普通64/race64、tagged vet 均 NOT RUN；该历史状态已由后文真实执行补齐，不用于冒称此前已运行。

Temp 证据根：`C:/Users/J00010385/AppData/Local/Temp/han-s92-review-p2poxay3/`。

- `read_fixed.py`：live/snapshot 全 183 路径/哈希与 main/HEAD guard；`guards.jsonl` 保留实际检查和首次集合校准失败。
- `reads.jsonl`：实际 source/依据读取范围与 SHA256；`static_summary.json`：有界结论和读取记录；后续追加读取以 reads.jsonl 为准。
- `target-full-new-file.diff`：两份完整新增文件 diff；SHA256 `0AC8D10AA744CAA761625F931B8F8FD52D7A6FA5CC7F968A7EE49E3BD50D0299`。
- `run_plan.json`：未执行命令及预检/取证计划；不是已运行 runner 或 LOCALPG 授权记录。

## 实际 Read set

完整方法/规则：本 Role `.codex/agents/add-reviewer.toml`；add-route-work、add-delivery-workflow、add-code-review；code-review-checklist、abstraction-principles、产品安全红线、core coding-standards/no-hardcode-policy/task-execution-workflow/multi-agent-coordination/observability-standards/tracing-standards；已启用 go-service README 及 coding/no-hardcode/security-coding/observability/tracing rules。安全编码规则只消费适用安全约束，不据此触发代码修改。

业务依据范围见“固定对象与依据”；Zeno Prep 实际正文与 Lead adopted scope 已读。原任务卡只读 S9.2 邻近段，不把旧 identity 包/Manifest PASS 借给本次。

源码：

- 两目标 tests 全文。
- `internal/identity/deletion.go` 全文；`article_port.go` 全文；`article_disposition.go` 全文；`recovery_run_store.go` 全文；`service.go:335–380` 的 canonical 构造/依赖校验。
- `internal/article/article_port.go` 全文；`command_service.go:1–144,321–526,850–1309` 的构造、canonical、处置及原生 proof；`repository.go:79–136,337–382,433–610` 的双 recipient、处置锁/条件写/完成事实回读。
- `internal/audit/store.go` 全文；`internal/testenv/env.go` 全文（只读读取合同，未实际调用或读取 env）。
- migrations 0001/0002/0003/0004/0005/0006/0007/0009 全文；0008 的 1–280、528–679 及相关 ALTER/trigger 定位。没有声称审完其余 password-reset 全部状态机。

## 独立真实执行与最终证据（2026-10-28）

Lead 在本会话正式交回作者真实结果并授予独立 LOCALPG START 后，本 Reviewer 使用新建 runroot `C:/Users/J00010385/AppData/Local/Temp/han-s92-independent-0abwkhwo/`，从固定完整 `fixed183-r02` cwd 实际执行以下三项。没有新增 smoke、重跑已通过 Go 命令或执行全 identity integration。

| 本 Reviewer 命令（go.exe） | 实际 PID | UTC 开始→结束 | 退出与结果 |
|---|---|---|---|
| `test -tags=integration ./test/integration/crossmodule -run '^TestCanonicalUserDelete(Postgres\|FaultsPostgres)$' -count=1 -timeout=15m -json` | 10988 | 02:25:45.600952→02:27:26.484366 | 0；64叶/76节点 run/pass各一次，0fail/skip/extra；actual wait完成 |
| 同上加 `-race`、timeout=30m | 48296 | 02:27:40.554089→02:29:28.086415 | 0；64叶/76节点 run/pass各一次，0fail/skip/extra；actual wait完成 |
| `vet -tags=integration ./test/integration/crossmodule` | 21248 | 02:29:50.696770→02:29:59.947681 | 0；stdout/stderr空，未注入DSN；actual wait完成 |

准确 argv 为各阶段 `launch.json` / `completion.json` 的数组，正则中 `|` 为原样 alternation，无 Markdown 表格转义。每个 Go child 前后均同时核 live/snapshot 全183路径与字节哈希、固定 main/HEAD，全部一致；`final-source183.json` / `seal-source183.json` 再次核对一致。原181源完全未变，运行时迁移也取自完整快照。

执行使用独立新 GOCACHE/build-temp/runtime-temp；仅复用既有模块缓存路径，未复制缓存。最小环境白名单继承基础Windows运行信息，显式 `GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOWORK=off`，并置 `GOENV=off`、`GOTELEMETRY=off`、`GOFLAGS=-mod=readonly`。User-scope HAN_TEST_POSTGRES_DSN 由本地 PowerShell helper 仅捕获到内存并注入集成 child；没有写入 argv、日志或文件；vet环境没有DSN或PG fallback环境。启动/原始日志/退出和工具哈希均留存。

两次独立运行各64个随机 schema，共128个，全部与作者129个及彼此不重用。逐case原始事件证明 preflight先于CREATEcommit，再按顺序九迁移提交；实际server literal loopback5433、180006、user/database匹配。缺环境与不符均为FAIL而非Skip。本次没有环境预检或Go命令失败。

独立证据实际复核：

- 普通/race各541个case文件，共1082个；954份实际stage JSON，共10494个11表脱敏rows哈希逐一重算匹配；所有stage文件SHA与原始日志声明一致，leaf/schema/sequence绑定完整，凭据字段为presence+SHA256或NULL，xmin/ctid保留。
- 按sequence逐次匹配最新事务基线、precommit observer、caller、commit回读及拒绝/故障/取消/并发退让镜像；每次180对，共360对11表完整内容相等；caller与observer确有真实未提交变化。重复同名stage按文件顺序处理，没有用最后同名项覆盖历史事务。
- 各64条resourceclose、各2次barrier owner join；共128条资源收束、4次join。business/observer PID不同，所有测试的tx结束/rollback、固定proof注册清空、pool acquire归零断言通过；无泄漏/竞态/零命中标记。
- 另外只读核查普通/race各13个成功leaf的最终脱敏行：删除墓碑、email/phone及密码摘要/generation清空、目标Session为零、current credential失效而历史consumed/superseded保留、username/phone reserved、按实际active/pending关联保留或清理email、符合条件恰好一次release；Recovery review=kept_deleted且run仍running；真实转移双recipient仍pending。此项没有额外PG调用。
- 真实Audit INSERT的23505/audit_events_pkey及Article/Canonical/Release/Disposition对应阶段、after-real-SQL注入、前置写与完整rollback/合法重试，已通过实际断言、原始事件和文件序镜像交叉核查。注入错误不冒称引擎/磁盘故障。

本 Reviewer 亦亲自解析作者 smoke1/3、normal64/76、race64/76 原始 JSON，核其四child的wait/exit/argv/环境与183前后记录，并独立重算其1091个case文件、962份stage/10582个表级脱敏哈希和按序镜像。作者结果仅作为作者来源，不替代上述本 Reviewer 三次实际执行。

Lead 的原始日志/SQL现场复核、项目Manifest7及其边界引用 `.spec/wip/identity-auth/verification-s9.2.md`；本报告读取时该汇总仍等待Reviewer交回。已核仓内作者归档index SHA256 `DBCD60049474EF8134B050FAE90DDC38475B0EE5ACF03013C3877FB0E820C829`、Lead Manifest index SHA256 `962AA0A430BF0E00C471903D22629FD677E806938D926115CA418C7B6A550D8E`。Manifest7是Lead实际执行，未记作Reviewer重新执行，也不代本次独立PG。

可复制独立证据索引：`C:/Users/J00010385/AppData/Local/Temp/han-s92-independent-0abwkhwo/artifact-index.json`，SHA256 **`DCCE5581EE0559C7A939F75CFBB0493991D52B53957E18F5605ED4E412BD6685`**。1146个相对路径payload，28833840 bytes；含全部own raw/stdout/stderr、launch/completion、1082casefiles、各次全183sourceguards、实际静态读取记录/规则正文及哈希、完整new-file diff、作者原始证据独立校验结果、按序/最终行校验工具与结果、handoff。排除go-cache、build-temp、Python cache、外部模块cache及一份编译runtime临时文件；没有删除任何文件。index不包含自身，以此报告绑定其SHA。静态版报告已原样留存为 `static-report.before-final.md`。

Temp核证工具有两次局部格式/集合修正：migration日志Name不含数字前缀；runtime目录比1082casefiles多一份220byte编译tmp。错误和修正说明保存在tooling-notes/artifact-tooling-notes，未改源、原始日志或case像，未重跑Go，不计产品RED。最终索引只筛选证据payload，所有原文件仍保留。

若后续作者修复或source改变，需Lead固定新manifest/快照/两目标hash，原Reviewer按新版本与相关回归复审，保留本版与旧证据，不混版。本Reviewer此刻停止交回，不修改任务状态或启动下一项。

本切片不覆盖完整 HTTP/Cookie/CSRF/Service.Register 响应、真实 FileJournal/永久删除及最后 pending finalizer 全链、backup/恢复全局 authority/readiness/MarkConsumed、SMTP、真实部署/浏览器、容量或生产就绪。SQL UNIQUE 仅证明占用；phone residual 不加严。最终 Review 结论不代 Task/Issue 关闭、人的 accepted/checkpoint 或下一动作授权。
