# article-management-S2.2 独立代码评审 R01

结论：**CodeReviewGate FAIL（2 项 MEDIUM 测试缺陷，待原作者修复或 Lead 明确处置）**。当前生产行为的已执行测试为 GREEN；本结论指出两处验收风险尚不能由现有断言证明，不是已观察到的产品缺陷。没有 CRITICAL/HIGH finding。

## 对象、身份与冻结修订

- objectType：`code`；development-delivery / B10 / review。实际 `add-reviewer`，非作者；未 Route、未委派、未代作者修复。
- 固定工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。评审前后均核对一致。
- 作者：Ohm `01a17c83-60f4-7803-b0cc-9b19c0b81fa3`；其 stopped / Host completed 为委托方提供的交接事实，作者 handoff 也记录 stopped；Reviewer 不宣称自行查询了 Host 会话状态。
- 三份文件均为未跟踪新文件，无旧前像，**全文即本次完整 diff**。实际逐行审查 `postgres_state_test.go` 902 行、`pagination_test.go` 168 行、`etag_concurrency_test.go` 133 行。
- 项目唯一写入是本报告；其他新增验证证据仅位于获准 Temp 的 `review-r01`。生产、三测试、设计、Tasks、Plan、旧报告、规则、迁移均未修改；无 Git 写操作、依赖安装、DROP/cleanup、外发或部署。

| 固定对象 | SHA256 |
|---|---|
| `test/integration/article/postgres_state_test.go` | `33CDCFAE247742584E333613FEC40BF380914F72D335F9787C7CF2395F9C1A0A` |
| `test/integration/article/pagination_test.go` | `FF08F7505AE92F727E5550EB6AE0E833497285B0B47D90FD4B29F558467F7AFF` |
| `test/integration/article/etag_concurrency_test.go` | `C7EAA2D8D9FB315926CE80049E15B3E7932A290D8CCF58BC123E32F97527F2FA` |
| article design r18 | `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3` |
| article test-plan r15 | `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` |

Reviewer 对 handoff 的 157 个冻结输入实际计算当前 hash，再与作者 ordinary/race 结果内各自的 157 个 source_before/source_after 逐一核对，均一致；Reviewer 自身前后再次比较，0 drift。hash 全表包含实际依赖、三测试与业务依据，但不表示全文评审了 157 个文件。整份 Tasks 的并行 identity 注记不参与源漂移判定；只消费 S2.2 整卡，没有评审或改写无关 identity 文档。

## 实际读取与两轮范围

完整取得自身 `.codex/agents/add-reviewer.toml`、`add-code-review/SKILL.md`、其安全红线 reference、checklist、abstraction、coding/no-hardcode/observability/tracing、go-service README 及对应规则正文；同时读取当前 delivery Workflow、task-execution、multi-agent 约束。go-service 安全加载门禁一并读取，本次无 Go 实施。未预读另一 Workflow、其他 Role 或未来专业方法。

业务依据为 S2.2 整卡，以及 design §2.4.1/2/4、§3.3.3/4/9、§3.4.1 与相邻事务合同；test-plan §4.1 A-READ-01～06、§4.2 A-ETAG-01～04/A-MOD-01～08、§4.3 A-MODR-03～06、§6/7 的本卡 PG 子集。没有把完整 HTTP/parser/fuzz、16 endpoint、Journal 真卷、部署 E2E 或容量验证加成本卡前置。

**Round 1：完整新 diff。** 检查三文件每个测试与 helper 的输入、SQL fixture、独立 oracle、断言强度、故障注入点、并发时序、事务/rows/context/group/pool 收敛、合成数据和错误去敏；逐项映射下列 18 风险组。确认 build tag、package、三顶层命名及仅只读复用 retained harness。

**Round 2：真实调用链和上游合同。** 有界取证为以下 18 个依赖源文件：article 的 `query_service.go`、`command_service.go`、`repository.go`、`policy.go`、`model.go`、`handler.go`；identity `article_reader.go`；audit `store.go`；同包 `notice_dispatcher_test.go` 的 retained helper；migrations `0001`～`0009`。大文件仅消费本对象相关段落：query 候选 SQL/扫描/当前作者读取/投影/cursor，command 替换/治理/行锁/version/CAS/rollback/audit，repository Article/DeletionOperation 插入，model 对应字段，handler 实际读/改/治理/decoder/error 映射。notice 只读 harness/config/migrate/UUID/exec/scalar/clock/资源关闭，不评 SMTP 历史。迁移重点为 0001 审计约束/append-only、0002 用户 CHECK、0003 Article/DeletionOperation CHECK/FK、0008 合成用户默认值/相关 trigger 与 0009；其余按现役全链迁移执行和相关 DDL 检索核影响，没有声称审计整个 identity 迁移业务。

调用链确认：本地 Handler → Service → Repository/真实 pgx transaction → PostgreSQL，查询经真实 ArticleIdentityReader；治理审计经真实 Audit.Store、同一 tx。没有虚构管理员 GET；第三投影是实际 SQL 回读后 AdminProjection。根 router 的已知独立接线问题由 Lead/Planner 处理，本报告不据本地 Handler 通过宣称 production route 可用。

## Findings（按严重度）

### AM-S22-R01-F001 [MEDIUM] 审计断言未将动作和类别绑定到实际操作及版本

- 对象/修订：本报告冻结的三文件新 diff，具体落点 `test/integration/article/postgres_state_test.go:591`、`:593`、`:598`；调用点为 `:561`、`:625`、`:881`。
- 现状：`s22AssertModerationAudit` 核 actor/target/trace/outcome、action/category 的集合成员资格和 `2..count+1` 版本唯一性，却没有逐版本比较本次预期 action/category。`s22ModerationStates` 先以 sexual mark 得到 v2，再 restore 得到 v3；两个实际结果的 Audit 只传入期望总数 2。并发赢家为 mark 或 restore 时，也只传入总数 1。
- 精确触发：Article 行仍正确经历 clear/v1 → violating/sexual/v2 → clear/null/v3，但 v3 的审计被误写为 `article.violation.mark`；或这两个事件 category 都误记为 `violence`。保持其他被检字段和版本 2、3 不变，上述错误仍满足当前全部集合与唯一性条件，测试可继续 GREEN。restore 赢家的 v2 审计误写 mark 同理。
- 影响：专项原本需要证明的“谁对哪一版本执行了何种治理及何种分类”没有闭环；真实 PG 中有效但语义错误的审计值不会被这个门禁捕获。Article 状态正确与 Audit 同 tx 并不能证明审计内容正确。
- 违反依据：S2.2 原卡明确要求“审计按实际动作对账 target/actor/trace/version”；design §3.3.9 的 mark/clear 证据及单独治理事实，checklist §2/3/6、observability 的审计语义要求。生产 `command_service.go:283` 起清楚区分 clear 使用前像类别、mark 使用新类别，不能用枚举合法性代替该操作契约。
- 当前事实：独立窄测后的只读数据库回查，三种 owner 的现存事件均为 v2 `mark/sexual`、v3 `clear/sexual`，与当前生产实现一致。**本 finding 是已确认的测试盲区，不是声称当前产品写错了审计。** Counterexample 属静态断言分析，未实施新的审计 mutation/overlay。
- 最小修复方向：让审计 checker 接收按版本索引的预期 `{action, category}`，各调用方从已知操作与不可变前像明确给出；逐条精确核对，不依赖 UUID 或相同 occurred_at 的排序。mark sexual v2 / clear sexual v3，以及单一 mark violence 或 restore violence v2 分别绑定。
- 负控/复审条件：在获准 Temp source snapshot + overlay 中，仅令 clear 的审计 action 误写 mark；另一个负控仅把实际 sexual 类别改为 violence，保持真实 PG 行和事务正常。原测试应暴露当前缺口，修复后的精确断言须分别失败；恢复实际源应 GREEN。原作者提交新冻结 SHA 和 diff 后，由非作者 Reviewer 复核所有相关调用点、实际 Audit 原子回滚、mark/restore 并发的审计与最终版本；不改生产契约来迎合测试。
- 证据：`review-r01/audit-assertion-counterexamples.json`；`review-r01/database-readback-r02.stdout.jsonl` 中 MarkRestoreAndConstraints 的三个 schema；本报告对应源行。Lead 后续独立指出相同疑点，与 Reviewer 第一轮定位一致。

### AM-S22-R01-F002 [MEDIUM] 分页 fixture 没有跨 created_at 的继续翻页，不能证明完整复合 keyset

- 对象/修订：同一冻结新 diff。主要位置 `test/integration/article/postgres_state_test.go:81`、`:83`，以及 `test/integration/article/pagination_test.go:64`、`:76`、`:122`、`:140`。
- 现状：`s.base` 对全部行固定 `CreatedAt=s.now`。密集隐藏、同时间戳、lookahead、空/末页以及跨请求测试的原有文章全部沿用此值。唯一不同的时间戳是 newest 的 `s.now+1s`，其 ID 99 同时也最大；它只在重开首页时被检查，不会成为继续翻页时必须跨越的旧时间戳行。
- 精确触发：候选 SQL 若错误退化为仅 `ORDER BY id DESC` 且 continuation 仅 `id < cursorID`，当前所有分页 fixture 的排序和过滤结果仍与正确复合 keyset 一致；独立 SQL oracle 在这些数据上也无法区分。对现实数据，例如较新时刻 ID 2、1，较旧时刻 ID 99，limit=2 应先返回 2、1，再返回 99；错误实现可能将旧 ID 99 排到首页，或用 ID continuation 丢掉它。
- 影响：虽然同时间戳 ID 平局、密集隐藏、lookahead 和 oracle 独立性已有真实覆盖，但 `created_at` 为主排序键及跨时间戳续扫仍是实际 PG 测试盲区。本卡不能仅凭目前 GREEN 声称完整 `(created_at,id)` keyset 的不漏不重。
- 违反依据：S2.2 分页单元要求真实 keyset 与静态不漏不重；design §2.4.1、§3.3.3 和 A-READ-01/02 明确 `(created_at DESC,id DESC)`。此项不涉及 parser/fuzz、额外 endpoint、性能或跨请求快照保证。
- 当前事实：生产 `query_service.go:759` 后的 SQL 已正确包含时间比较和平局比较；本 finding 不声称现役 SQL 有该错误。对 ID-only 回归的不可区分性由三文件全文和 fixture 值静态确认，未运行新的 SQL mutant。
- 最小修复方向：在现有分页文件内补一个至少两个时间戳、且 ID 与时间顺序交错的 fixture，使一次真实 cursor continuation 必须越过时间戳边界；保留独立 SQL oracle 和逐页去重/最后项 cursor 断言。可复用现有风险组，不需第四文件或新 task。
- 负控/复审条件：新数据下源实现 GREEN；获准 Temp overlay 中仅删除旧时间戳分支或将候选查询退化为 ID-only，必须由结果序列断言 RED。以新冻结三文件和原生产依赖复核普通/race 专项，确保同时间戳及密集隐藏原覆盖未回退。
- 证据：上述 fixture/调用点、`s22Dense`、`s22Oracle`、`s22Traverse` 全文；`review-r01/pagination-fixture-counterexample.json`。该文件是静态分析记录，不冒充执行过的 mutation test。

## 18 风险组的实际断言与证据边界

| 风险组 | 实际检查结论 |
|---|---|
| State / VisibilityMatrix | 10 子例实际插入合法 CHECK/FK 行，含匿名、active/suspended/deleted 作者、violating、held 与叠加、pending/anonymous pending、不存在；真实 Service 与成员 Handler 核 200/404，拒绝标题/hidden_reasons 泄露。不是所有正交维度的穷举积。 |
| State / ThreeProjections | clear/violating 真实 SQL；作者 JSON keys 精确 allowlist，restricted/editable/body、成员拒绝、AdminProjection 的 version/category 存在性/原因数量及 ETag。管理员为 SQL+policy，不是 GET。 |
| State / CurrentAuthorState | 同一 QueryService 前后真实 UPDATE 用户 suspended/active，重读验证无状态缓存。 |
| State / AuthorStateFailureClosed | wrapper 先真实 reader 查询再受控 missing/fault；missing 不当匿名/active，fault 无数据/无 cursor。没有删除 FK 所指用户。 |
| State / ContentFieldIsolation | clear/violating × literal/rune 上限；保留前后空白、HTML/Markdown/Unicode 字面量；整 Article 比较只允许内容/version/updated_at；无虚构内容审计；作者/管理员内容与成员限制继续核对。 |
| State / InvalidContentRollback | 两类治理态 × 六类非法 title/body，真实 Handler 422，目标完整 Article JSON 与全 audit 集合前后相等。 |
| State / WireBoundaryPersistence | 两类治理态 × 精确 N/N+1/非法原始 UTF-8；真实 Handler、真实 PG 快照；拒绝时零 Begin 仅作补证，N 持久化一次、version+1、非内容字段不变。非完整 HTTP decoder 矩阵。 |
| State / ModerationStateAndAuditRollback | mark/restore 后整 Article 非目标字段、active/suspended/held 限制；真实 SQL 后 fault、真实 Audit INSERT 后 fault 均快照回滚；另连接限时重拿行锁并成功重试。审计动作/类别精确绑定受 F001 限制。 |
| Pagination / HiddenAcrossBatches | 20 行隐藏密集区、候选 batch=3/limit=2；独立 SQL JOIN users oracle，跨多批、无漏重；第二候选/identity 调用故障时无累计半页或 cursor。 |
| Pagination / SameTimestampNoMissingDuplicates | 9 行同时间戳，UUID ID 打破平局，当前作者状态过滤；有终止上限和重复检测。跨时间戳主键覆盖受 F002 限制。 |
| Pagination / LookaheadCursorBoundary | 5 行/limit2，cursor 等于最后返回项，继续遍历与独立 SQL 序列一致；所有页均再核 cursor 时间/ID，未消费 lookahead。 |
| Pagination / EmptyAndLastPage | empty/all_hidden/terminal；空 items 非 nil、cursor null；静态序列、终止、末页由 traverse 核。 |
| Pagination / InsertEditAndAuthorStateBetweenPages | 读第一页后新增新文章、真实内容替换、真实停用另一作者；tail 与当前 SQL oracle/cursor 对账，旧遍历排除新文章、重开首页可见、重读内容更新。未承诺跨请求快照。 |
| ETag / SameVersionTwoWriters | 不同实际连接 PID；writer A 获取 FOR UPDATE 后 barrier 持锁；pg_stat_activity/pg_blocking_pids 确认等待再释放；HTTP 200/412、最终全行/version/赢家 ETag 及零不适用 Audit。 |
| ETag / EditVersusModeration | edit 与 mark/restore 两类、两种赢家顺序；都从两个初始合法动作开始，检查失败方无覆盖/幽灵 Audit，真实行锁/CAS。适用审计有 F001。 |
| ETag / MarkRestoreCompetition | 合法 mark/mark、restore/restore 单赢；另顺序核 current-clear restore 为 invalid_state、stale repeated mark 为 precondition、pending 覆盖版本。未把混合 mark/restore 初始不都合法的动作强说成全 412。 |
| ETag / RefreshAndRetry | 同版本竞争后重新读作者 ETag，再实际 PUT 合法重试，SQL version 单调且新 Header 一致。 |
| ETag / HeaderVersionConsistency | 更新 Header、真实成员/作者 GET Header、Service DTO/ETag 与 PG version 同核；更新 body 精确最小四字段，不泄露内部 version。 |

## 真实验证及来源分层

证据根：`C:\Users\J00010385\AppData\Local\Temp\han-s22-bca6db0ff5314f56ab830d68b42e881f`；Reviewer 新证据全部在其 `review-r01` 子目录。没有修改原件。作者日志为 UTF-16 BOM；Reviewer 逐行正确解码 JSON，而非按文件名/摘要推断结果。

### 作者既有证据：独立解析，非 Reviewer 重新执行

- `ordinary-final-r01`：`go test -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$' -count=1 -timeout=8m -json`，exit 0，3 top / 76 run / 76 pass / 60 leaf / 0 fail / 0 skip；18 组均实际命中。
- `race-final-r01`：同上增加 `-race`，同样 exit 0、3/76/76/60/0/0；各自 source_before/source_after 157 项均与当前固定源一致。
- `tops-final-r01`：精确 `-list` 三顶层，exit 0；`vet-final-r01`：`go vet -tags=integration ./test/integration/article` exit 0。未把非 JSON 的 list/vet 当测试节点。
- `negative-cursor-r01`：实际完整源 snapshot 与当前绑定；仅把最后返回项改为第一返回项，overlay 精确指向对应 production 源；exit 1，`pagination_test.go:90` 的 cursor 断言失败。
- `negative-stale-r01`：实际源 snapshot 与当前绑定；仅将 `existing.Version != expectedVersion` 改成 `<`；exit 1，预期失败 writer 实得 200 而非 412。两个负控均为预期 RED，不是当前生产 FAIL。snapshot、mutant、overlay 的 hash 均重算核对。
- 221 个作者保留 schema 的 index 实际 count/unique_count 均为 221；未逐个重查作者全部 schema。早期 ordinary-first 及 cases-green-r02 零命中不作为最终验收证据；保留 failure chain，不清理。

| 原件 | result.json SHA256 | stdout 原始 bytes SHA256 |
|---|---|---|
| ordinary-final-r01 | `FCDFF8460A8BAA3493FE6B4BC1EA85D9EA1BCEE1708EA25CD5F00C4DFD7FC755` | `F6B000CB1AAA12518BD2C0614628715A888D511F0718E575A435CA122D978D01` |
| race-final-r01 | `213DD79926E294C4D29268CC01325492CC9585B8AC914DE4B92249385F9461C5` | `58FC4B061D2155AB2B9C187130BD5C78E81259BC49F516D2931553BEED5544D8` |
| negative-cursor-r01 | `91AB81AF7097A3BDE38AB9FDEA427817E6F33018D96BE65E81AA47B40A95B527` | `18CF8FDC85943F1A7DF9E0DEF5A3F59A3B2B0624A286D0F0EBCA191E10B41A52` |
| negative-stale-r01 | `D2D65E2103F2C22E6E50C0EC7F305D508F9A84C98CE745541E35F77F932DC995` | `BFF51A2B836422B488EFD7983D7CCF16F043DC0D58D67C4E94E554C426B01C7F` |

上述各 stderr 都为空，SHA256 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。完整六项解析与 hash 在 `review-r01/author-raw-analysis.json`，含 list/vet 原件 hash。

### Reviewer 本次独立执行

先从 User-scope 读取 `HAN_TEST_POSTGRES_DSN`，只注入本进程环境，不打印原值。只读运行既有 Temp `probe.go`，exit 0，实际回报 `127.0.0.1:5433 PostgreSQL 18.6 / 180006`；随后仅运行一次有理由的本包 race 窄测，覆盖行锁竞争、Audit/content SQL 原子回滚、密集隐藏/故障无半页、lookahead 与刷新重试：

```text
go test -race -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$/^(ModerationStateAndAuditRollback|HiddenAcrossBatches|LookaheadCursorBoundary|SameVersionTwoWriters|EditVersusModeration|MarkRestoreCompetition|RefreshAndRetry)$' -count=1 -timeout=4m -json
```

结果 exit 0；**3 top / 26 run / 26 pass / 17 leaf / 0 fail / 0 skip**。18 个 fixture/schema（HiddenAcrossBatches 父节点自身另有 fixture），18 条资源关闭 receipt；实际 `h.cancel → group.Wait → AcquiredConns=0` 后由既有 harness 关闭 pool。测试进程已 exit，已注册 worker 正常 join；没有新增 SMTP、后台服务或 child worker。所有新 schema 保留，精确 case/schema 清单在 `narrow.analysis.json`。

额外只读回查这 18 个新 schema：`psql -X -A -t -q -v ON_ERROR_STOP=1`，通过 process environment 传递连接参数并设 `default_transaction_read_only=on`；exit 0，返回 18/18，每个 21 表、468 个 pg_constraint。回读 Article version/violation/deletion 与适用 Audit action/category/version/role/trace；mark/restore 的实际结果正确，故 F001 不写成产品 bug。未进行 DDL、DML 或清理。首次 psql 使用 PGDATABASE URI 的连接探测 exit 2，未执行 SQL；原失败日志保留，随后显式 process PGHOST/PGPORT 等注入的 r02 才是有效回查，不把首次失败记 PASS。

Reviewer 另外执行三文件 `gofmt -l`，exit 0、无输出。未重复作者两套全量专项，未执行旧 identity/all integration。没有安装未有的 goimports/golangci-lint；项目实际验收按现有 Manifest7，工具默认示例不替代项目命令。

| Reviewer 证据 | SHA256 |
|---|---|
| `narrow.result.json` | `39BE059D04CC6AEA56634DAFC899FB0D53B5F5967CF880F2B83E4155420B0040` |
| `narrow.stdout.jsonl` | `499DB8CE6D1C81883E6C7DBC58676C8559DE1CE3A436951542E0229D2B8FA034` |
| `narrow.stderr.log`（空） | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `narrow.analysis.json`（含18 schema） | `37B0B9E5A4A13CF3F9192BB13CDCC52D925939BC01A38DABD0D4A689D566623F` |
| `database-readback-r02.result.json` | `546C1466FE1FBE083D781FA563733074E0038347C2B45AB77CA5822A3CD6F5D5` |
| `database-readback-r02.stdout.jsonl` | `5539EB8B5540DEF46707682D7D32AA9AFBF1A271B9242E8201D07DE28A3E9C27` |
| `database-readback-r02.stderr.log`（空） | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `audit-assertion-counterexamples.json` | `C24687941A2BA7CCF4C4C4D0A365B5D123AA661B9FE18DC468599CEED4ADA420` |
| `pagination-fixture-counterexample.json` | `6081114D658F1F6126730F04493724415BE483FEF3C920590FE8148760031FB2` |
| `freeze-before.json` | `E3D419F85C5D1D68BB75258B58BA73F6ECF8971B5999111610F2C0E0873C8AB2` |
| `freeze-after.json` | `7EE65E95EE3CC54F89E5133F28754ADED52A5DB8A3A29F18711C057690FE64C8` |

前后 freeze JSON 的 bytes/序列化不同，hash 不同；比较其解析后的 157 个 path→source SHA 值全部一致，0 drift，不能以两个 manifest 自身 hash 不同误报源漂移。

### Lead 已执行检查：核原始日志，未冒充本角色执行

实际读取 `lead-manifest-r01-results.json`，SHA256 `FC2274ED3C187021A62E6ACE3280BE36EE195DFB4DD5A9210F6E659C8999EE32`，核其每份 raw log 的 bytes hash、exit 与157项 source_before/source_after。七项分别为 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`，全部 exit 0；gofmt 空，原始日志 hash 均匹配，未发现 FAIL 行；source 与本对象固定修订一致。分析保存在 `review-r01/lead-manifest-analysis.json`。全包默认测试不替代 integration tag 专项，也不关闭本报告 F001/F002。899 baseline 全库归属和接收仍由 Lead 复核，本角色未审查那899个文件。

## 待决定、建议、未验证风险及交回

- **待决定**：Lead 按项目规定决定两个 MEDIUM 的本轮修复或显式风险处置；Reviewer 不替其接受风险。本报告没有新增架构/契约方向决策要求。
- **非阻断建议**：无额外风格、命名、函数拆分或格式建议；不把既有 profile 模板占位、其他模块未接线扩成第三项本卡缺陷。
- **限制**：当前证据只覆盖本地 Handler/Service/PG。没有生产 router/root/真实会话凭据、完整 HTTP 权限/error 矩阵、浏览器、真实平台、Journal 真卷/finalizer 全协议、数据库中断或 commit-unknown、部署/E2E、容量/P95、生产就绪或人的验收结论。pending fixture 证明合法 SQL 行不可见，不证明真实删除恢复全流程。未执行 F001/F002 新负控，不将静态反例称为运行失败。
- **复审输入**：原作者针对 F001/F002 的最小修复 diff、新三文件 SHA、保持生产依赖冻结的前后 hash、各项有效 RED/GREEN 和普通/race 专项命中/资源/schema 原始证据；本角色核原 finding 与新修订，避免混用本次旧 GREEN。
- **状态**：评审与本报告完成后 **stopped**，没有正在运行的验证或写任务。交 Lead 核实际报告 hash、diff、证据与 Task 状态；Reviewer 不关闭 S2.2、不修改 TaskPlan、不代作者修复、不启动后续工作。
