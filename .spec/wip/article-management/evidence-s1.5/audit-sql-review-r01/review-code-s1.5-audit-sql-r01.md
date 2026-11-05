# S1.5 真实 Audit INSERT SQL 子集 CodeReview R01

2026-10-28。实际非作者 add-reviewer Erdos，objectType=code；作者 Dalton。沿当前 development-delivery 续接，不重新 Route、不委派、不代修。固定对象、作者停止及环境/metadata 消息来自 **Lead 交接**；下文另列本 Reviewer 实际读取、执行和核证，不能理解为用户逐项直接提供或人的 accepted。

**有界 CodeReviewGate：PASS；新增 findings 为0。** 两轮检查及独立 LOCALPG 普通/race 均覆盖唯一 top 的四个必要 leaf。结论仅针对新增测试实现及其声明的真实 SQL 回滚/重试子集。必需证据输出目录的配置限制见下文，未将省略配置的失败探针混作成功或产品 RED。应用 ACL、全部十二 requires、完整 S1.5、S8.2/Case18 和全局恢复门禁继续 blocked/pending；本报告不关闭 Task，不代人的验收。

## 固定对象、实际增量及读集

现场只读确认工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。唯一仓库写入为本新报告，报告写入前路径不存在。

| 固定输入 | 实核 SHA256 / 范围 |
|---|---|
| `test/integration/article/audit_insert_rollback_test.go` | `468964647524860CEEF0167EC56A9C0033FBB40CFF340A5021FBCE66F2C89CD5`；完整945行，原不存在，全文即增量 |
| 作者 `new-file.patch` | `E4488A3AD656902440BFA7BE53538739E76ADCF8C6C4911E84B283878713049C`；独立提取全部新增行，与现文件规范化换行后的全文相同 |
| `fixed179.json` / 本人 before 与 after map | 三者均 `F253083287B30E43305459E0B146B9E439376DA27B94ADBDB9C194C59C17A700`；逐项内容及完整路径集合179一致 |
| 原178基线 | `E88DC26D92259E3C147F644E27FDE4E4079899A3428B9A7D74FB8A16E70D470E`；全部原178字节相同，无增删替换 |
| test-plan r19 §4.7.2 | `88296F772A19600A237CCA9F47A16D4B673828EF41E5C1D0C2D88924BE2C6F66`；574–608行完整读取 |
| design r21 | `A4C9FD6589DC5FDE912DE8E5CD6C076268AD077A218F9866439B9477A479C3BA`；同版相关治理/转移/删除/审计合同复用，非本轮全文重审 |
| Pauli r19 规划报告 | `A89FF3A126980C7DC1CB85F9468084ACAD716FE47F33F9C80619DE0278013F1B`；PREP实读全文，仅消费规划层结论 |
| 原四组件独立 R02 | `52298FEDD5F13E9A9017334C1FEA72A1BD72C9A9658A2384660D9075D6F59933`；原F001关闭保留，本轮仅指纹、不重开或重签旧结果 |

作者原件目录为 `C:\Users\J00010385\AppData\Local\Temp\han-s15-audit-impl-3468c1e54fe74fc8a36de31fb0cf62a5`，以下称 A。本人材料独立存于 **`A\review-erdos-r01`**，以下称 R；未覆盖作者原件。R有完整179输入个人副本、每条执行命令的完整 before/after map、raw/meta、逐schema SQL镜像及独立解析结果。

完整 Reviewer Role、add-code-review 主方法在 PREP 实读刷新；同版 current delivery、checklist、abstraction、安全红线、task-execution/multi-agent、coding/nohardcode、observability/tracing 及唯一启用 go-service 对应规则复用。未加载其他 Role、未来 E2E 方法或未启用 profile。

实际生产读取为有界必要片段：`audit/store.go` 的 Event/Append/实际 Exec；`command_service.go` 的 Moderate/事务/审计、Transfer完整写入顺序、Reference/VerifyCompletion、错误包装及 UPDATE SQL；`repository.go` 的 Article/Notice/双recipient写入、处置锁与完成事实回读；`deletion_finalizer.go` 的公开构造/Finalize、claim/complete/lease、匿名合法分支、DELETE/Audit/Complete及其Repository SQL；`model.go` 的 operation/tombstone、`deletion_journal.go` 的公开Journal接口；`identity/article_disposition.go` 的公开生命周期锁、完成verifier/reference、真实清理及相关SQL；`identity/article_reader.go` 的公开reader/扫描/错误链；`platform/errors.go` 的 PublicError/Unwrap；0001完整审计DDL、0002 users/identifiers及0003 operation/Article/Notice/recipient约束片段。PREP与当前所读核心源同版。

准确逐文件深度、来源与SHA见 `R\actual-readset-and-bindings.json`，SHA `FB089604243FDAA4917EE1A7B7AEF5B7B5750F9F6403E13D73E0535218508916`。179项hash绑定不等于179份全文审读。root仅属于hash保护集，未读/评当前root代码；未把未知root writer判为已释放。Tasks/Plan只消费Lead已知登记事实，不审其他卡/身份文档，不以metadata变化作为本代码漂移。

## Round 1：新增测试全文与行为敏感性

`integration` tag、`articleintegration`包、唯一 `TestArticleAuditInsertRollbackContract`（34行）及四个直接leaf均实际存在，没有Skip、placeholder或第五个共享辅助文件。新增helper均自承本文件；未调用旧harness隐含的建schema/清理路径，无新依赖或生产修改。

`arConfig/arAddress`（105/124行）解析真实 pgx 配置，逐一检查主Host和所有Fallback为literal loopback:5433。`arPreflight`（161行）在CREATE前核实际地址、端口、版本180006及非空实际身份。CREATE独立提交后立即记录retained schema（143行），再另开迁移事务按0001～0009完整应用。operation先存在、pending Article版本为operation版本+1；合成用户、标识、Article和Audit满足真实CHECK/FK，没有靠破坏FK制造错误。

`arAuditor.Append/arWitness`（357/367行）先断言收到业务的确切事务，在同tx查询已经发生的业务写，保存前置镜像，再原样委托 `audit.Store{}.Append`。`arTx.Exec`记录真实 INSERT结果，不替换SQL/结果/error。`arCollision`（416行）同时核实际Exec及返回链中的同一个PgError：23505、本schema、audit_events、audit_events_pkey；并核业务owner已rollback、没有commit。碰撞行由真实Store在独立seed事务先提交，不是trigger拒绝、伪PgError或成功INSERT后追加synthetic error。

完整镜像SQL（265行）在一个语句快照中按确定顺序保存articles/audit/notices/recipients/operations/users/identifiers的完整行和值。失败回读使用专用observer连接，业务Begin逐次断言实际backend PID不同（291行）；不是同连接重入。NOWAIT（502行）按适用锁序锁真实行并检查非零命中、rows.Err/Close；不是仅凭超时推断释放。retry的独立完整Article/Audit/Notice/recipient oracle、碰撞行保留、无关表及身份字段保留均已实看断言，未从实际event反推期望action/category。

## Round 2：真实调用链、状态、事务与四例回读

| leaf | 生产路径与本轮核到的实际证据 |
|---|---|
| MarkRollbackAndRetry | 真实 Moderate→Repository UPDATE→Append。Append前同tx已见version+1/violating/violence；实际主键冲突后零payload/ETag，独立完整失败像等于调用前像，NOWAIT成功。同原version、新AuditID重试后独立见完整Article、精确mark/action/category/version及actor/trace/时间，无关字段和碰撞行保留。 |
| RestoreRollbackAndRetry | 合法sexual违规前像，真实Moderate(nil)先清state/category再INSERT失败；完整失败像与前像相同。新ID重试只清违规并递增version，clear审计category精确来自不可变sexual前像；正文/归属/publication等完整值保留。此例与mark均使用合法匿名Article，未把未进入的作者状态reader分支称作本轮已执行。 |
| TransferRollbackAndRetry | 真实公开TransferOrphan，经真实identity生命周期SQL锁与Repository先插Notice/两recipient、改author/source/publication，再Audit失败。同tx见上述写入；失败返回零result/NoticeID，独立七表完整像不变、未进入cleanup、NOWAIT按users→identifiers→Article验证释放。新ID序列重试commit后独立核完整Article、恰一Notice/两pending快照、准确transfer审计；真实proof→VerifyCompletion SQL回读→真实identity校验/清理另写release审计，source retained_email及reserved email identifier按规则释放，username/目标用户及其他字段保留。 |
| FinalizerSQLRollbackAndRetry | 合法anonymous/sourceNULL普通pending，公开Finalize→Journal.FindByOperation→真实claim commit→completion另tx。Append前同tx Article已物理DELETE且operation仍finalizing。失败后独立像等于“claim已提交、completion未开始”的像，Article完整pending恢复；不能要求operation回到调用前journaled。独立NOWAIT验证operation/Article释放；读真实locked_until，仅将受控UTC clock推进至其后1µs，公开Finalize真实重claim/commit，独立见Article消失、完整audit绑定/completed_at及owner/lease清空。没有手改DB lease、调用私有complete或伪造proof。 |

Transfer adapter（775–830行）仅转换公开DTO，并绑定真实工作流传入的proof、tx与Reference；verifier调用该proof的真实 `VerifyCompletion`，没有从fixture制造authority或复制领域状态机。Finalizer最小匿名分支合法返回nil proof，与当前公开合同一致。Journal（838行起）仅返回固定、绑定一致的合成tombstone；意外Append显式失败，不证明FileJournal验链/fsync/POSIX/真实旧备份。

每例defer/cleanup核rows关闭、事务回滚、observer.Release、context cancel、pool acquired归零条件检查及pool.Close；业务wrapper回滚使用独立3秒预算。没有自启worker/goroutine，未编造额外join。资源证据为源码绑定断言通过及逐schema退出日志；没有另行打印逐counter时间序列，不能据此声称精确池指标普查或全机无进程。

## 本 Reviewer 实际执行及个人证据

本人 `R\review.py` 每条child前后验证完整179路径及SHA，只有既有User级 `HAN_TEST_POSTGRES_DSN` 读入该child。通过按键筛除继承的Process DSN及PG变量后才读取其余环境值，不读Process DSN值；User DSN不进入CLI/raw/meta/源码。`GOTOOLCHAIN=local/GOPROXY=off/GOSUMDB=off`，未安装依赖或使用外部网络。tagged vet child不注入DSN。

先实际执行 `go run R\preflight.go` 的独立只读探针：真实pgx.ParseConfig核主地址及全部fallback；本次fallback=0；实际loopback:5433、version180006、identity=postgres，连接关闭成功，零DDL。随后每个测试fixture仍各自重复CREATE前的真实预检。该身份仅为测试连接，不是应用ACL身份。

runner为每个测试child显式设置 `HAN_AUDIT_EVIDENCE_DIR=R\<label>-evidence`，该父目录预先创建，case子目录由fixture创建；全部保留。实际命令如下，非作者结果：

```text
go test -tags=integration ./test/integration/article -run ^TestArticleAuditInsertRollbackContract$ -count=1 -timeout=8m -json
go test -race -tags=integration ./test/integration/article -run ^TestArticleAuditInsertRollbackContract$ -count=1 -timeout=8m -json
go vet -tags=integration ./test/integration/article
```

| 本人label | exit | top / run / pass / leaf / fail / skip |
|---|---:|---|
| preflight | 0 | 只读连接探针，无测试/schema |
| normal | 0 | 1 / 5 / 5 / 4 / 0 / 0 |
| race | 0 | 1 / 5 / 5 / 4 / 0 / 0 |
| vet | 0 | 无测试运行，stdout/stderr空 |
| missing-evidence，见配置说明 | 1，预期 | 1 / 2 / 0 / 1 / 2 / 0；两fail包含父top，不是两个leaf |

普通/race的四leaf均逐行解析命中，无重复，集合完全相同：MarkRollbackAndRetry、RestoreRollbackAndRetry、TransferRollbackAndRetry、FinalizerSQLRollbackAndRetry。未运行旧S22/S82或任何其他integration top，未以全包运行代替精确目标，未重复Manifest7。

| label / leaf | 本人retained schema | business PID / observer PID；业务tx数 |
|---|---|---|
| normal / Mark | han_s15audit_26028a0ff2844f1499c2ba74fd967475 | 54176 / 63048；2 |
| normal / Restore | han_s15audit_926ea47fb50048e1b20ed98c822f712a | 12840 / 61800；2 |
| normal / Transfer | han_s15audit_a79c45cc56d04d57922e9ce9e4629dd7 | 52320 / 20208；2 |
| normal / Finalizer | han_s15audit_647548e2bc424b1c97514cafd6296bff | 61460 / 61080；4 |
| race / Mark | han_s15audit_e0f78854257842d882b3c105b9ee641f | 38180 / 47836；2 |
| race / Restore | han_s15audit_a753cc404ca546d386e035894bddd8e7 | 39980 / 16104；2 |
| race / Transfer | han_s15audit_0b6593971b5f4de7a0745e7a74aeafea | 39184 / 46644；2 |
| race / Finalizer | han_s15audit_898e68f2dd2e4baf8561a27b42538b09 | 8836 / 61616；4 |

个人SQL核证独立解析50份normal/race镜像：每次治理/Transfer各before、两个before-Append、after-failure、collision、after-retry共6份，Finalizer另有claim-committed共7份。逐例核实际23505/同一error、前后像关系、业务写已发生、独立PID、9次迁移、NOWAIT、fresh-ID重试及资源关闭日志，与源码断言互证。全部8业务schema保留；下文配置探针另1个schema亦保留，共9个。这里的retained来自实际CREATE日志、无DROP路径及关闭行为，不冒称另行执行了全库schema盘点。

个人raw均在 R，精确命令/时间/exit/父child PID/等待结束/完整179before-after位于同名 `.meta.json`：

| raw stdout | SHA256 | 对应meta SHA256 |
|---|---|---|
| preflight.stdout.jsonl | `3A85EC00D9B0DE4CC8B88C79D113001B628EED7DA71767E437002A03746132C8` | `CCE452F454E1A3E425FD9B652139F10399D2FAACF7CB18B304DCFAA03E82CB33` |
| normal.stdout.jsonl | `A5183FF03891D21082B122F573E3C46011D4343949683D4D117B4D89C151376E` | `D5AF1AB956EB909B9270522DB14EB89F8F9A90C25F0B73B4A36D69EDB8547C09` |
| race.stdout.jsonl | `23AEBF2EA0ED66F5A2D7011B85DD038A232E4AF6B9403EEACAC2D11F0B1D51FF` | `AACB2F9D3761A088AF609DDA9B4007594AB670114B3A9820E82DE5227D9F2222` |
| vet.stdout.jsonl | 空 | `E07A1BBD6F9B4263CAF9F9657698EBCFD4E21C13030E88124A527D28AA94C6CC` |
| missing-evidence.stdout.jsonl | `094E01834FA720CAF40F8853219429C8A8E3EFD351D6D6A3FF502155B3109C4D` | `83ADFED8D6C5550776A42AB13EC7BD61C2EE2F26A8F0BE845463E140AE67719F` |

五份stderr均空；空文件SHA为 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。`R\raw-index.json` SHA `129DB9A17246F4D786846702A40B816646B98692EE066B9740A948008BCA50B1` 包含15份raw/meta的逐项路径/hash。`R\run-set-and-sql-verification.json` SHA `C4DE9928E0E3B40742855E6D55E820BFCDD0B660125D8F18EE63858576A51D43` 保存完整leaf集合、9schema及50镜像逐项hash/PID/阶段关系。

## 作者负控、历史证据与配置限制

独立核作者stopped声明的artifact SHA及17 runs/51 raw-meta SHA；未将每条历史运行都称为最终179版本。作者最终普通/race、missing-DSN和error-chain mutant的meta before=after均为当前179，普通/race真实各5run/pass、4leaf、0fail/skip。初版增强前的4schema保留为历史辅助，不冒充当前测试字节；作者共16schema的清单保留，非本人新增或本人数据库普查。

实际检查作者完整179副本及157个Go-overlay映射，只有 `internal/audit/store.go:101` 的 `fmt.Errorf("insert audit event: %w", err)` 变为 `%v`；mutant SHA `6287A96BE78FF9464168DB746EDF1B38199DE84CE635E6AD9A0B259A561DC695`，overlay SHA `F23D01B372703D67AF5B17C280820811623CAFC6940B3192978528C11A0EEE1C`，新增测试及其他断言字节均不变。完整diff另存 `R\verified-author-mutation.patch`。

作者mutant实际JSON各leaf先出现真实 `23505/audit_events_pkey`，随后在固定 `arCollision` 的errors.As断言（424行）失败，exit1、5节点fail/4leaf，非编译或环境错误；原生产码最终normal/race及本人重跑均GREEN。负控仅证明错误链保护，不证明事务持久性mutant覆盖。作者负控stdout SHA `4C5F5A4B5B4565994D658D5287F03D26FB5ACA5552BC95169D5E595561415EA8`；本轮未重新执行该mutant或新增其schema。独立核case-map中59份镜像SHA，结果在 `R\author-verification.json`，SHA `428FDB90EA7CCD878E6C2F4C0167AE5136902543517436BD81FA0A4EC5B67E21`。

**非阻断交接说明：证据目录是必需runner配置，r19裸命令省略它会失败。** 具体位置为新文件145–149行。本人保留User DSN、只取消child的 `HAN_AUDIT_EVIDENCE_DIR`，运行：

```text
go test -tags=integration ./test/integration/article -run ^TestArticleAuditInsertRollbackContract$/^MarkRollbackAndRetry$ -count=1 -timeout=8m -json
```

实测CREATE/log后报 `audit evidence directory required`，尚未迁移/进入业务SQL；退出1、单leaf fail，资源关闭，`han_s15audit_2751d918f36d4dbebceb0da357dd83c1` 保留。该raw不是SQL业务回归RED，不可混入目标普通/race绿色集合。

本轮未将该项列为代码finding：r19要求保留完整证据但未规定缺省目录策略；作者交回的 `run.py` 明确创建并注入专属输出目录，最终meta记录该路径，当前可复核的交付运行链已包含配置，实际SQL断言也不依赖伪结果。此处是测试artifact存放配置，非新增生产Gate、ACL或其他凭据。本人正常命令同样显式提供该配置并完整通过，因此本PASS以这条已披露的runner配置为边界，不能描述为“仅配置DSN即可直接运行r19裸命令”。非阻断建议是后续运行说明同时列明输出目录及父目录要求；如另行决定默认MkdirTemp或前置检查，应由原作者获授权后修改，不能据本建议擅自修订冻结源码。

Lead另同步Manifest7全exit0/gofmt空及179不漂、回执SHA `B6570C3E41B639BB36A2366182B9CBDA2D0B99FFC439264663B9C7C968067931`；本轮未重复七项或逐项重核其raw，该项明确仅为Lead执行消息，不代替上述本人integration证据。可变verification receipt的“已启动/待回交”历史时点、旧AA6取证SHA及旧四组件raw均未作为本专项运行PASS依据。

## 结论与停止边界

新增缺陷findings：CRITICAL/HIGH/MEDIUM/LOW均0，无新finding ID；没有需要Reviewer选择的新业务/权限合同。已完成Round1全文增量与Round2真实直接消费者/状态/事务/数据约束及证据链检查。有界CodeReviewGate PASS限于固定单文件的四例真实PG INSERT冲突、回滚、释放和合法重试测试实现及上述运行条件。

未验/未解除：应用UPDATE/DELETE有效ACL身份与载体仍unknown；未建role/GRANT，不把postgres测试身份、trigger或Store无写方法当ACL证明。Journal受控接口未证明真卷/fsync/POSIX/旧备份/完整删除类型。未跑root/SessionSQL/完整16端点/producer-source/全局恢复/部署E2E，不继承身份waiver；全部十二requires及S8.2/Case18隔离和开流子项、完整S1.5关闭仍阻塞，人的确认未代签。

原178与新增测试、完整179集合在五条本人child的前后均一致；原四contract及其F001 CLOSED不重开。未修改源码、测试、Task/Plan/design/identity/root/配置或他人报告，无Git mutation、服务/SMTP/Browser/外网、清理删除移动、下级委派。本人所有同步启动child均已wait结束，预检连接已关闭；测试资源按源码断言及逐schema日志正常退出，无本轮未结束child，不声称全机进程普查。9个本人schema及所有失败/raw/快照保留。

**stopped：本独立评审与LOCALPG窗口已完成；交Lead核实际结果和后续门禁，不自行关闭任务或发出新源码写授权。**
