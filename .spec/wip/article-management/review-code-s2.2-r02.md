# article-management-S2.2 独立代码复审 R02

**CodeReviewGate PASS（本次冻结修订及原卡本地 Handler/Service/PostgreSQL 边界）。AM-S22-R01-F001、F002 均 CLOSED，无新增 material finding。** 此结论不关闭 Task，不代替 Lead 接收、人的验收或生产接线/E2E 验证。

## 身份、修订与写入范围

- 实际原独立 `add-reviewer`，非作者；沿 development-delivery / B10 / review 续接，未 Route、未委派、未接管原作者修复。
- 固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，前后核对一致。作者 Ohm 的 Host completed/stopped 是 Lead 提供的交接事实；新 handoff 同样记载 stopped，本角色未自行查询 Host 状态。
- R01 报告 `.spec/wip/article-management/review-code-s2.2-r01.md`，实际 SHA256 `FE353F0595C661AE6FB45405F12F2F5E8390C99AF034FA2D71AAD0AC1D2D747D`；保留原失败记录，未覆盖。
- R02 handoff：获准 Temp 根的 `handoff-s22-r02.json`，实际 SHA256 `6F70BF2132A54167B1A68E19B94573B3BE857045DB9A56BEACC4D9FD33863B70`。
- 项目唯一新增文件是本报告，首次核对路径不存在后创建。新 raw/分析记录仅在获准 Temp 根的 `review-r02`。没有修改任何源码、三测试、notice、TaskPlan、设计、旧报告、Rule 或迁移；没有 Git 写、安装依赖、DROP/cleanup、外发或部署。

| 固定文件 | R02 SHA256 |
|---|---|
| `test/integration/article/postgres_state_test.go` | `A77E5214BC4257088C03164BC9801384B13DF848D9679BA7716FF9D1DA433FD0` |
| `test/integration/article/pagination_test.go` | `13B6F1B3A2625AFB9240A07188F928A63E307B009FF17ACC879811E51A4A498F` |
| `test/integration/article/etag_concurrency_test.go` | `C7EAA2D8D9FB315926CE80049E15B3E7932A290D8CCF58BC123E32F97527F2FA` |
| article design r18 | `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3` |
| article test-plan r15 | `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E` |

实际重算 R02 `final_source_hashes` 全部157项，0 drift；与 R01 保存的157项逐路径比较，仅上述前两个测试文件有获准修复变化。ETag 测试、生产、notice、迁移、设计依据均维持原修订。作者普通/race 的 source_before/source_after 各157项也均与当前 R02 一致。无关 identity 文档及 Task 元数据不参与代码对象漂移判断，未评审或改写。

## 两轮实际复审

同会话 R01 已完整读取的自身 Role、add-code-review 主 Skill、安全红线、checklist、abstraction、coding/no-hardcode/observability/tracing、已启用 go-service profile、delivery Workflow 与 task-execution/multi-agent 规则正文继续有效，本轮按用户明确允许复用；未重跑 Route 或预读其他方法库。

**Round 1：固定修复 diff。** 对 handoff 保存的三份 R01 完整测试 snapshot 重算 hash，确认与 R01 对象一致；再与当前三份文件逐行生成完整 unified diff。R02 三份完整 snapshot 也逐一核 SHA。实际变更仅为：审计期望类型/checker、三类调用位置（mark/restore、回滚后重试、并发治理赢家）以及一个跨时间戳子例；ETag 文件无 diff。核原 actor/target/trace/outcome、count/version 唯一性、SQL 回滚/锁释放、编辑赢家零 Audit 和原 lookahead 断言未被削弱。新增 defer rows.Close 保留显式提前 Close/rows.Err，随后 Audit.Store.Get 不占用扫描连接。

**Round 2：相关消费者、SQL、状态与验收。** 定向重读生产 `internal/article/command_service.go` 的治理事务/锁/状态/version/UpdateArticleModeration/appendModerationAudit，以及 `query_service.go` 的持续扫描、最后返回项 cursor、真实候选 SQL；按同版本复用 R01 已读的 Audit.Store JSON 读写、Repository、identity reader、model/policy/handler、retained harness 与相关 DDL。只核原 F001/F002 及其消费者回归，没有重新扩大为全库评审。对照原卡“审计按实际动作对账”、design §2.4.1/§3.3.3/§3.3.9、test-plan A-READ-01/02、A-ETAG-02、A-MODR-03～06 的适用 PG 子集。

独立检查三组生产 mutant 的完整 original/mutant diff、overlay Replace 的精确路径/内容/hash、旧/新测试 snapshot 及六份实验的原始结果。只有审计 action、审计 category、候选 SQL 旧时间戳分支分别发生单点 mutation；未替换目标 Repository、Audit.Store 或真实 SQL。随后进行一次独立 race 窄测及 PostgreSQL 只读样本回查，见后文。

## 原 finding 逐项处置

### AM-S22-R01-F001 [MEDIUM] → CLOSED

- 修复位置：`postgres_state_test.go:575` 定义期望；`:580` checker；`:561` mark/restore；`:639` Audit 回滚后的合法重试；`:897`～`:901` 并发治理赢家。
- 已核实际断言：期望为 `map[int64]s22ModerationAuditExpectation`；审计总数严格等于期望数；version 必须是整数、不得重复、必须命中期望映射；action/category 对该版本逐项精确相等；最后再次遍历期望版本，拒绝缺失事件。actor、role、target、trace、outcome 的原有绑定保留。事件顺序不依赖相同时间或随机 UUID。
- 期望来源：显式 sexual mark v2 / sexual clear v3；Audit 故障回滚后重试显式 violence mark v2；并发 mark 赢家使用操作已知的 violence，restore 赢家从冻结输入 `r.ViolationCategory` 的不可变前像取类别，并绑定 `r.Version+1`。restore 用例初始均为合法 violating 状态，前像类别非空；没有从被测审计结果反算期望。内容编辑赢家仍断言零 Audit。
- 上游一致：生产 clear Audit 应保存解除前的类别，mark 保存新类别；该映射与真实 `appendModerationAudit` 和原卡合同一致，不改变生产行为。
- 负控有效性：`clear-action` 只将 clear Audit 默认 action 改成 mark，Article 更新/版本/事务保持真实；旧 R01 测试 exit0、6 run/pass、3 leaf，新 R02 测试 exit1、6 run/fail、3 leaf，三个 owner 均在 v3 动作/类别断言失败。`sexual-category` 只把 Audit 中 sexual 记为 violence，旧同样 GREEN，新同样 RED，失败版本为 v2。不是编译失败、环境错误或零测试。
- 真实 PG 证据：独立只读回查两个负控的 active schema，Article 均已正确 restore 为 clear/null/v3；clear-action 样本的 v3 Audit 实为 mark/sexual；sexual-category 样本的 v2/v3 Audit 实为 violence。数据与负控原始失败断言吻合，证明修复捕获原先会漏过的真实 Audit 语义错误。它们是 Temp overlay 实验数据，不是当前生产工作树错误。
- 回归：独立 race 窄测实际命中 mark/restore 三种 owner、Audit 回滚/重新获锁/重试、内容 SQL 回滚、edit vs mark/restore 两种赢家顺序、mark/mark 与 restore/restore。当前原源 GREEN，资源 receipt 全部通过。原 finding 关闭条件满足，无遗留 OPEN。

### AM-S22-R01-F002 [MEDIUM] → CLOSED

- 修复位置：`pagination_test.go:94` 增加 `CrossTimestampContinuation` 子例，`:97`～`:111` 定义 fixture 和序列断言。
- 实际数据：6 个可见行、3 个时间戳；新时刻 ID 2/1，中间时刻 99/98，最旧时刻 199/198；ID 与时间主序交错。固定 default limit=2，真实 query 必须跨两个时间戳边界完成三页。
- Oracle：独立 SQL oracle 先与明确列出的 `[2,1,99,98,199,198]` UUID 序列相等，再比较真实 Service 遍历结果；没有复用被测分页算法生成期望。原同时间戳 lookahead、每页最后返回项 cursor、去重与终止断言保留。
- 负控有效性：`older-timestamp` 只删除生产候选 SQL 的 `OR created_at < $1::timestamptz`；保留排序、同时间 ID 分支和真实 PG 读取。旧 R01 fixture exit0、2 run/pass、1 leaf；新 R02 exit1、3 run/fail、1 leaf，在 `pagination_test.go:111` 的“cross-timestamp continuation lost or reordered visible rows”断言失败。
- 真实 PG 证据：独立回查负控新增子例 schema，6 行仍存在，实际时间/ID 顺序确为上述三组；不是缺 fixture、SQL 未执行或约束失败。原源独立 race 窄测的父 lookahead 与新增跨时间子例均 GREEN。
- 上游一致：真实复合 keyset 时间分支与同时间 ID 平局均按 design §2.4.1/§3.3.3 受测；没有引入跨请求快照承诺、parser/fuzz 或性能前置。原 finding 关闭条件满足，无遗留 OPEN。

**新增缺陷：无。待方向决策：无。额外非阻断风格建议：无。**

## 作者证据的独立核验（非本角色重跑）

Temp 根为 `C:\Users\J00010385\AppData\Local\Temp\han-s22-bca6db0ff5314f56ab830d68b42e881f`。本角色逐行解析正确编码的原始 JSON，重算 result/stdout/stderr bytes hash，核对 index 与 source_before/source_after；没有仅采用 handoff 汇总。

| R02 原始执行 | exit | top / run / pass / leaf / fail / skip |
|---|---:|---|
| ordinary-final-r02 | 0 | 3 / 77 / 77 / 60 / 0 / 0 |
| race-final-r02 | 0 | 3 / 77 / 77 / 60 / 0 / 0 |
| fix-r02-old-clear-action | 0 | 1 / 6 / 6 / 3 / 0 / 0 |
| fix-r02-red-clear-action | 1，预期 RED | 1 / 6 / 0 / 3 / 6 / 0 |
| fix-r02-old-sexual-category | 0 | 1 / 6 / 6 / 3 / 0 / 0 |
| fix-r02-red-sexual-category | 1，预期 RED | 1 / 6 / 0 / 3 / 6 / 0 |
| fix-r02-old-older-timestamp | 0 | 1 / 2 / 2 / 1 / 0 / 0 |
| fix-r02-red-older-timestamp | 1，预期 RED | 1 / 3 / 0 / 1 / 3 / 0 |

完整普通/race 命令分别为：

```text
go test -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$' -count=1 -timeout=8m -json
go test -race -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$' -count=1 -timeout=8m -json
```

18 个原风险组在两份最终 raw 中均命中，新增 child 也命中。节点从76增为77而 leaf 保持60，是原 LookaheadCursorBoundary 成为带子例的父节点；不据文件名或 leaf 总数误判新用例缺失。普通/race 分别62个 fixture和62条资源 receipt，均 source157/0 drift。`tops-final-r02` 的精确 -list 实际仅三个顶层、exit0；`vet-final-r02` 的 tagged vet exit0。非 JSON 的 list/vet 不计为测试节点。

三组负控的旧记录 source_before/source_after 绑定 R01 三测试，新记录绑定 R02；两版完整 snapshot 都与相应 SHA 一致。各 overlay 只映射指定生产源到其单点 mutant；旧/新实验使用同一 mutant，生产 original snapshot 均与当前冻结原源一致。没有把当前 R02 测试当成旧 GREEN，或把预期 RED 当作当前产品 FAIL。旧 `cases-green-r02` 零命中历史未作为证据。

| 原始 stdout | SHA256 |
|---|---|
| ordinary-final-r02.stdout.jsonl | `5B0B8E45B7D5D38C1D9C7DCA1D386347D05B635DF4322FF657C108373007C5AA` |
| race-final-r02.stdout.jsonl | `2CA07DCBFA2C7EE4824C57B989C6FCD7A841A16CF37805E24A8DC2C32B2F09AD` |
| fix-r02-old-clear-action.stdout.jsonl | `F904ADBAE71C5AA9DB5058D6761FD64F5FDFA6AA2378A11389EED62DC543B525` |
| fix-r02-red-clear-action.stdout.jsonl | `B61506C68DC4ABD9FBD828CC153B1DD35676D83B4F89E446A5FB7B0E229C2649` |
| fix-r02-old-sexual-category.stdout.jsonl | `E555641D02F66BBAC90DFC5D02BCD5013FAC242F9E5DCEC74A660BF43B9E3985` |
| fix-r02-red-sexual-category.stdout.jsonl | `E7F97A324E5A346A5814E4EA03FB55E0AD9CD9DB8AB740B6B5435F84DE7A550E` |
| fix-r02-old-older-timestamp.stdout.jsonl | `61B99EAB8029076FA0761B461238109E26CCBCFACC42C81E8BA6D7994AA1FE81` |
| fix-r02-red-older-timestamp.stdout.jsonl | `EFB099233FCD84361885DFEE986D1A6FDF72D5763B4746043AE6FCED0B892A98` |

以上 stderr 均为空，SHA256 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。result.json SHA、完整命令、实际失败断言和节点计数均保存在 `review-r02/author-evidence-analysis.json`；其 SHA256 `C4D421157F8539ADBBAD96434165F9DBCA6EE29F6FFAAF8BF68B1C269FEC8BFA`。其中 ordinary result SHA `A6F9D3AB985DEA3AB8770318B78F3C1CC59A1135BC00BF0233AC210A8BE5585B`，race result SHA `E0148C5F0D23B96B4146B8CE69CDEE2A54D7CED925635A4E93F3E0D55C05E154`。

| 负控 | mutant SHA256 | overlay SHA256 |
|---|---|---|
| clear-action | `0878DD8987A4F1028EFF0A2F635FAF020AE829E56F0246753A5480E5D783D19B` | `3E27254C96A7054C9EF2C9C118FF0A61FCB265F26106DBFFEC63EAF5E8D8263E` |
| sexual-category | `FCC76C1062E80F060E448EB8155B43598FABED54349CB493BE37E8C9FF0B7DCD` | `C77F797455666AD4C97ED6E2FAEF9C2F58B94E0D68FA3DDB10D0D02960B860CA` |
| older-timestamp | `A0C4F004C6C76A2C525A5B2ECC712013FD8BC61534B3FCF488CB3D4C1FAD1382` | `F3B7F5C7ACC47EF32FBC005260B082D54FDC61EAD8D2A2FCAA2A5C93160A7477` |

生产审计 original SHA `585BB51D19B3FCD7BFDC9C84465D2BB5D24F41BD8285553FC20BC10B161B4D68`；分页 original SHA `9674130673CD05E8FA864B77528FDBF12C9AB8305AE6AD2EF7E2A85DB9136E7E`。所有 original/mutant/overlay 的真实 bytes 都重算匹配；没有改动生产工作树。

作者 R02 retained index 的 SHA256 `233FA18BF57B8E0F18E0A7FFDAACD4218151F1F83C7C4EE347B8711C74279469`，其 count/unique_count 实际均153；coverage index SHA `DEB649924083104442FB9A515F49C769BED0966B2715C0F6FB593CF1B4FB287F`。153是作者新证据集合，不是本角色新建schema数，也不表示本角色逐个回查了153个schema。

## Reviewer 实际独立执行

User-scope `HAN_TEST_POSTGRES_DSN` 仅注入执行进程环境，未打印或放入 CLI 参数。先只读执行既有 Temp `probe.go`，exit0，确认实际 `127.0.0.1:5433 / PostgreSQL 18.6 / 180006`。为验证修复及全部审计调用方、行锁/原子回滚回归，仅独立执行以下一次 race 窄测，未重复两套 full：

```text
go test -race -tags=integration ./test/integration/article -run '^(TestArticleState|TestArticlePagination|TestArticleETag)$/^(ModerationStateAndAuditRollback|LookaheadCursorBoundary|EditVersusModeration|MarkRestoreCompetition)$' -count=1 -timeout=4m -json
```

结果 **exit0；3 top / 21 run / 21 pass / 13 leaf / 0 fail / 0 skip**。14个新增fixture/schema全部保留，14条资源 receipt；父 lookahead 自身另有fixture，故 schema 数大于leaf数。精确schema/case清单在 `narrow.analysis.json`。测试进程已退出，context取消、group.Wait、pool AcquiredConns=0及既有harness关闭路径实际通过；rows先关闭再Audit回读，tx真实commit或rollback，无新增SMTP/server/后台worker。

独立额外以 `psql -X -A -t -q -v ON_ERROR_STOP=1` 执行只读 SELECT UNION，连接值仅通过process PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE等环境传递，设置 `default_transaction_read_only=on`。回查14个本角色新schema和3个作者负控样本，共17/17，exit0；各21表、468个pg_constraint。读取最小合成 Article 状态/version/time 与 Audit action/category/version/role/trace；不读取DSN/真实凭据，不做DML/DDL/清理。负控样本状态见各finding处置；新跨时间fixture的6行与三个时刻真实存在。

独立三文件 `gofmt -l` exit0且无输出。未运行旧identity/all integration，未创建新负控或改断言来得到GREEN。

| Reviewer 新证据（均在 review-r02） | SHA256 |
|---|---|
| narrow.result.json | `23193432F2B4C20548A7AEE00B066F7969FDE9875841F970BA5C323820C710D8` |
| narrow.stdout.jsonl | `EC35A4619CB211A9BD79C1A2310F851A582F13684BFBEF3DEC25B44836AF86CF` |
| narrow.analysis.json | `CA1E5F1AB1BC79841E7B695C44F7325B0767988C36F15206C24BCC15505CAA6F` |
| database-readback.result.json | `9D38D425FBD23E43893F56BC66B78EFF220BE82E50D6C520805A01D42E19765D` |
| database-readback.stdout.jsonl | `0B40DEECF1A8B8EE6B51673DE51E9E2F9992E9EE40A6122CA70E34E08B156FFF` |
| freeze-before.json 与 freeze-after.json | `5B2D6161CDB209B4D40D22A993BEA5D278A7621B71D3BC4EE4DAF91E64682A6C` |

narrow和database-readback的stderr均为空，SHA为前述空文件SHA。Reviewer前后157个冻结值完全相同；最终报告交回前另核当前值。

## Lead 检查的证据归属

实际读取新 `lead-manifest-r02-results.json`，SHA256 `4F728382C2FD343B7B72D4CE39E51CAE20BDA1C0C0186C733D312EAFA6B8014D`；其7份原始log重算hash均匹配，source_before/source_after各157项均与R02当前源一致，0 drift。七项为 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`，exit均0，gofmt空。分析保存在 `review-r02/lead-manifest-analysis.json`。它们由Lead执行，本角色只独立核证，未冒作自己的执行；R01 Manifest为历史，未混用。

## 限制、复审触发及交回

- PASS仅针对冻结R02、原S2.2卡的本地Handler/Service/PG责任和两原finding修复；完整18组的未变部分沿R01全文审查与本次同源full raw回归，不声称又全面审查157/899个文件。
- 不外推production root/router、真实Session凭据/浏览器/Host、全部16 endpoint及HTTP/parser矩阵、Journal真卷/完整删除恢复、DB outage/commit-unknown、部署E2E、性能容量或人的验收。Admin第三投影仍是实际SQL+AdminProjection，未虚构管理员GET。其他会话identity材料保留，其豁免不进入本卡。
- 后续若改审计期望/helper/调用方、分页fixture、相关生产SQL或依赖，应提交新的精确diff/SHA与相关回归复审；本PASS不自动覆盖新修订。此刻无未关闭finding或必须新作方向决定。
- **stopped**：本角色评审、独立验证及报告写入完成，没有仍运行的验证或写任务。交Lead核实际报告SHA、工作树、证据、接收与Task关闭条件；未直接改Task/Plan或代签人的验收，也不启动下级或后续工作。
