# S4.13 R02 MEDIUM 原作者修复交接 R01

原作者 Hooke/add-backend-dev，原 Agent ID `01a189aa-4c16-7eb0-8dd6-fffaf4762cb1`。固定 Han/main/`6da8abbceb0fa197364810a92a8fa2d97b38ddca`，同一 S4.13 Task 续接，未新 Route、未子委派、未提交。当前对象为 **RECOVERY-B13E-S413-CR-R02-F001 MEDIUM OPEN**，作者不自签关闭或 CodeGate。原 **RECOVERY-B13E-S413-CR-R01-F001 HIGH** 已由 Ptolemy R02 独立 CLOSED，本次保留其代码和回归，不重开。

SOURCE/LOCAL/identityPG 已显式 RELEASE，见 [release.json](release.json)。17个真实作业全部 parent Wait/processWait/kernel Job0/两EOF/pump join；最终普通/race各193叶全通过、逐名相同，75个本轮精确 schema 实际 SQL核存在、完整 installation images 已保留。释放后仅本人D封包，没有再写/读共享源码或重跑Go/SQL；Main后续冻结/核收及原Ptolemy R03独立结论分别等待其真实结果。

## 问题与原合同

正式 R02 [原报告](review-input/report.md) SHA256 `FE842830C5734780011F8FBFF255888518060F6A4A8D93696A5CDDA7D068D6FE`、finding receipt、release与writer普通/race原始失败完整保留。Reviewer通过实际 public NewRuntime/Install 证明：已知 Writer != Runtime.InstallationWriter 最终 ErrAdmissionBinding，但此前已有 prepared+两初始Anchor共CAS3、Media1、anchors2；两侧DBBind0，Stop真实返回。它是配置预检副作用，不是错库写、越权或数据丢失，不夸大成HIGH。

identity design1662要求同一Store/零第二pool；1672–1678要求动作实际依赖预先具备，fresh/prepared依赖Writer而bound仅协调；3960–3966规定prepared/Anchor持久化和仅同批准manifest续接，不能靠先产生这些效果发现已知坏配置。identity wrapper16–24只保护其 supplied Store，不能查看封装的 Runtime writer。当前 production 原本就有精确错绑拒绝，只是位置在 prepare 的 Media/Anchor后。修复维持现有合同，无新API、authority、kind、schema或批准规则。

## 实际改动

仅原5中的两份发生变化：

- `internal/recovery/installation.go`：在合法bound快捷只协调分支之后、首个prepared CAS/Media/Anchor之前，将同Runtime Writer精确比较并入fresh/prepared检查；原临近DBBind的比较保留。constructor/common guard不增加全局Writer/Media依赖。
- 同文件另修本次**原作者真实发现**的来源错误分类：私有 `readInstallation` 返回fresh分类值，只有真正 `Commitments.LoadInstallation` 的缺键结果能标fresh。独立SourceVerifier、guard或ctx错误不因包含ErrAnchorMissing而触发创建prepared。现有 `load` 签名与其所有准备/回读消费者保持不变，原错误cause不丢；新私有helper不改变公共API。
- `internal/recovery/installation_test.go`：在原top内加配置预检五例和来源错误一例，均经真实public NewRuntime/Install。来源失败通过只转发Runtime原permit/context的观察器，在实际CheckPhase成功之后注入；不mint private permit、不替代目标executor算法。

原identity三份文件字节未变；production自 `swap` 起至文件末尾全段字节未变，包含原HIGH已修 reader、strict codec、原批准/Anchor/DB/legacy和所有后置校验。原75测试正文保留，只新增dispatch及末尾回归。见 [source.diff](source.diff)、source-before/source完整字节与 [self-check.json](self-check.json)。

| 原5冻结文件 | SHA256 |
| --- | --- |
| internal/recovery/installation.go | E74B5ECAAA17F8B16868DD59212E564F89C4B4CDE0AA0C781754EA94CACEB10D |
| internal/recovery/installation_test.go | E550AAAD2C441F78058634D6B41BFB1F9A37A10E91B234DDD78BBB3058D11D2C |
| internal/identity/recovery_installation.go | 3606BDB7E69FE457C1F930C8D97D1014FA60405B1F507F080F419D6C53555C3F |
| internal/identity/recovery_installation_test.go | 40FEC5E5437009A94F670A9F76AD07D4CBB2085D2F582556361C1F738BB5B3EB |
| internal/identity/recovery_installation_postgres_test.go | 17076C884D39476EAD22F511CD328A2D753522AC0C6AE0EE55BE2C317542476E |

## G0与授权时点

[g0-and-plan.md](g0-and-plan.md) 是只读准备阶段的原时点记录，当时未获SOURCE/Go/SQL；之后 Main明确 [START](start.json) 才应用owned测试和执行，见 execution-start.json。不是从reviewer release推导作者权限，没有再向用户追加同合同审批。

正式报告/receipt/actual writer raw、当前installation.go全文、上轮完整diff及实际Runtime/identity消费者接缝已读；当前5与此前同角色完整阅读版逐字节相同，明确复用。Role/实施主Skill/TDD/Workflow/所需Rules和GoProfile13项均核同版本SHA后复用全文；六docs/Task/Plan/TP及原需求/ADR合同不变，当前指定设计条文重新读。见 [methods-reused.json](methods-reused.json)、[readset.json](readset.json)。入口23精确匹配，此后每作业对新版Auth12+docs6的peer18前后严格守卫；只own5可变，未改Main freeze。

## 真实Red、诊断与Green

| 证据 | 实际结论 |
| --- | --- |
| review-input/checks/writer-normal、writer-race | 原Reviewer各1行为FAIL，原样保留，不是作者新执行 |
| raw/preflight-behavior-red | 原production不变，fresh错Writer实际CAS3、prepared错Writer实际CAS2，均Media1/DB0/Stopjoined；三项缺Media/bound正控通过 |
| 同作业的首版source叶 | 夹具诊断：grant前调用ReadPermitBinding触发ownership关闭，injected=false/context canceled/CAS0。不是来源行为Red，未据此改生产 |
| raw/source-behavior-red-correct-boundary | 修正测试边界后，原production仍不变，真实Runtime已发permit且CheckPhase成功；injected=true、原cause保留，却prepared CAS1，Media/Anchor/DB0，Stopjoined。只有这次真实Red才授权合同内作者最小来源分类修复 |
| raw/preflight-behavior-green | 两项生产修复后同6叶通过；仍是测试注释清理前的原SHA，按实际保存 |
| raw/final-normal-regression、final-race-regression | 最终同5SHA各193叶PASS，0fail/skip/unfinished，无race告警 |

来源分类错误单独标为原作者发现，详见 [author-source-discovery.json](author-source-discovery.json)。Ptolemy原sourceMissing片段是NOTRUN候选，其grant前checker方式不能作为本次正确注入的证据；原片段原样保留，没有修改后冒其独立验证。新MEDIUM原finding与作者额外发现都仍须Ptolemy R03确认。初次夹具失败、真实行为Red、既有格式失败分开分类，见 [diagnostics/classification.json](diagnostics/classification.json)。无编译、缺方法stub或零命中假Red，所有失败raw均保留。

193叶每模式构成为：原75 + 新增6 + 原已关闭HIGH6 + 原Writer1 + 直接受影响RuntimeInstall105。新配置五例包括fresh/prepared错Writer、fresh缺Media、bound缺Writer/Media与bound带未用异Writer；后两个均真正成功只协调，零安装副作用。来源新例断言原cause和零CAS/Media/Anchor/DB，现已通过。原真实同SQL Store、两Anchor、prepared续接、实际删除后重启、anyBegin/fullhistory、legacy一次NULL补绑/精确幂等/错绑及forward SQL最低P均在原75内重新执行。

```text
go test -p=1 -parallel=1 [-race] -count=1 -json -timeout=300s -tags=integration -overlay=<ownD>/packet/probes/test-overlay.json ./internal/recovery ./internal/identity -run '^(TestRecoveryInstallation(Binding|StoreBinding|Postgres)|TestReviewerS413BoundReaderKinds|TestReviewerS413AnchoredEmptyRecoveryRestart|TestReviewerS413MiswiredWriterBeforeEffects|TestRecoveryRuntimeInstall)$'
```

两overlay只将当前两份owned测试完整前缀加上原HIGH6/Writer1片段原字节，没有production overlay、不改其期望。额外首次193普通执行也已PASS，但随后清理了两段测试中残留“准备中”注释，故保留旧source/prefix并重跑最终同SHA普通/race；不把两次普通相加扩大覆盖，不假称先前Green6/普通193也有最终test SHA。见 comment-refresh.json 与 [probes/final-version-provenance.json](probes/final-version-provenance.json)。[leaf-evidence.json](leaf-evidence.json) 提供逐叶及原75保护核对。

本次未重跑无关457/611/全仓矩阵。Reviewer要求的其他实际C1/cancel最低必要验证仍由Ptolemy R03独立完成；作者本次结果不提前签其PASS或exoneration。

## 检查、资源与schema

局部真实 `go vet -p=1 -tags=integration ./internal/recovery ./internal/identity`、`go build -p=1` 两包、own5 gofmt空输出、git diff --check均exit0。精确原23格式子集仍实际formatter exit2，23输入SHA与原raw stdout/stderr字节相同；仍FAIL，不借旧处置豁免本轮，更不称全仓格式PASS。goimports/golangci-lint本地缺失，未安装、未伪跑。所有新增函数≤50行，完整自检留存。

17作业的argv/executable、完整stdout/stderr/result、source23前后SHA和own5完整前后字节均保留在raw。每作业都是真实suspended-before-assignment、parent handle Wait/processWait、kernel Job active0、两pipe EOF和pump-thread join，失败亦然。作者Job采样见33个PID，不称主机全部PID清扫。没有Popen.wait替whole-tree、未知PID kill、异步callback逃逸或未join SQL/pool/Runtime。`recorder_after_join_at` 是实际join后记录时刻，不编造成精确kernel Wait timestamp，见 [lifecycle.json](lifecycle.json)。

runner/helper按同作者前次已亲读完整源码、同SHA复用，改用此新root/output，原脚本/输出未执行覆盖。所有TEMP/TMP/GOTMPDIR/profile/work在本人D，本作者原D缓存独占复用；offline/local/modreadonly、GOMAXPROCS1/-p1、每次单Go parent串行，既有32 hash模块输入保留，零网络安装。只有具名Main identity_dsn入内存环境，未输出/复制值或用ambient/旧资源。Main owns PG启停，作者未启停任何服务。

75个本轮schema来自注释清理前普通25 + 最终普通25 + 最终race25，全部唯一 `han_b13e_s413_` 前缀。只逐名核本轮清单，不扫历史或旧前缀、不DROP。保留：

- [schemas/names.json](schemas/names.json)：75精确名字；
- [schemas/physical.sql](schemas/physical.sql)：显式当前51935/PG180006/MainD data目标校验和逐名只读SQL；
- [schemas/physical-receipt.json](schemas/physical-receipt.json)：实际存在性、表数、完整installation行像、run/Audit计数；
- [schemas/schema-case-map.json](schemas/schema-case-map.json)：每个名字映射真实job/test/raw；
- raw/physical-own75：真实psql argv/完整返回/Wait与pipe join；
- media：21份synthetic checkpoint/tombstone副本，原文件/cache/data全部继续保留。

## 交回边界

当前新MEDIUM **RECOVERY-B13E-S413-CR-R02-F001仍OPEN**，等待原Ptolemy R03对新冻结、作者来源分类发现及必要C1/cancel独立复审。原HIGH独立CLOSED保持。作者release/本报告/命令通过不等于Main CodeGate、Task关闭、人的接纳或整个批次通过。

只证明软件fixture外部批准/CAS/media及真实SQL/Runtime/reader契约；没有真实provider、独立批准主体、ACL/native/生产root/S4.16长期资源/完整恢复/E2E/业务接纳证据。无Git写、设计/Tasks/Plan/SQL源码变更、清理/删除/移动/归档、子委派或越原5。共享已有dirty内容保留；Git摘要折叠未跟踪目录，own5以独立前后字节和diff交付，不借Git摘要冒精确新增差异。

cache-free manifest覆盖本报告、源/差异、失败及完整raw、lifecycle、schema/readset/限制；原数据缓存与所有旧轮证据均留存。release后只本人D文字封包，无再次执行窗口占用。
