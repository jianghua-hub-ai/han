# S4.13 F001 原作者修复交接 R01

原作者 Hooke / add-backend-dev，原 Agent ID `01a189aa-4c16-7eb0-8dd6-fffaf4762cb1`。固定 Han/main/`6da8abbceb0fa197364810a92a8fa2d97b38ddca`，同一卡修复续接，未新 Route、未子委派、未提交。对象为 Ptolemy 的 **RECOVERY-B13E-S413-CR-R01-F001 HIGH OPEN**。本报告是作者实施与自测交接，不关闭 finding，不授独立 CodeGate，不代人的接纳。

作者已释放 SOURCE/LOCAL/PG，见 [release.json](release.json)。Main 随后明确核收原始 14 作业、33 个已知 PID 无 pending、peer18/8400/branch/HEAD/staged0/旧23，并冻结新5；这是 Main 的复核事实，独立于作者 Job 观察到的28个 PID。canonical reception 为 `.spec/wip/recovery-runtime/evidence-b13e-r01/s413-f001-source-local-pg-main-release-r01.json`。本报告只引用 Main 的明确消息，未在释放后重新读取共享文件或伪造其 receipt 副本。窗口已交 Lorentz，作者只在本人 D 目录封包。

## 缺陷、合同与最小修复

正式原报告 [report.md](independent-failure/report.md) SHA256 `6C38382B26F50896BD14078642D78C137F510280D39A2B10151752BF28DEB1BF` 与原 finding receipt、普通/race reader/public restart 的完整 raw 均原字节保留。Ptolemy 实际证明：公开 Install 对独立批准的 anchored_migration、零 Recovery Begin/run、Article 旧头 sequence1 成功 prepared→双初始 Anchor→同一 SQL Store→bound；正常重建却被 reader 的字面量 `Kind != genesis` 拒绝，genesis 对照通过。未把旧作者通过结果当作此次失败或独立执行。

identity design 的 §4.2（3960–3968）规定两种明确 kind、独立批准、prepared/bound、从初始头连续核证，以及任一 Begin 后永久转完整历史。3598–3603 的 genesis 用语与该迁移合同合读，表示零 Recovery 历史时的初始证明，不能要求捏造 recovery run。§3.3.1 的1662/1682/1690规定独立批准、同 Store 与只读安装验证器。真实消费者 `admission.go:468–474,539–547` 已按零完整 run/零 Recovery entry 选择 reader；它在此前核两 namespace，之后重核 Store 与两 namespace revision。

实际 production 只改原 `internal/recovery/installation.go` reader 内一处资格判断及说明注释：保留前置 `validateInstallationManifest` 对 **genesis / anchored_migration 两种值的严格白名单**，将 genesis 字面量限制换为 **Recovery 初始 committed head 必须精确等于零头**。因此合法已批准的 bound migration 可以提供零历史 origin；有 Recovery 初始历史的 manifest 不能使用此 reader 证明。

以下原条件和责任全部保留：bound、完整精确 manifest/digest/binding/两 initial heads、非空 revision、独立 SourceVerifier、批准后的完整 commitment/revision 精确重读、ctx/cause；prepared/未知 kind/错绑/无批准仍关闭。normal 不写 metadata、Anchor、Store，不发 grant，不调用安装或协调。任何 Begin 的完整历史分支仍由原 admission 执行。未修改 admission.go、Runtime、SQL、设计、Task、Plan、Main freeze 或任何原5外源码。

原5内仅3文件发生变化：reader production，以及原 recovery 单测与原 identity PG 测试各新增一组回归。另两个 identity 文件字节未变。完整前后源码、[source.diff](source.diff) 和 [source-freeze-candidate.json](source-freeze-candidate.json) 可直接审查；原有测试函数正文与旧55个 leaf 名称未删改。

| 冻结文件 | SHA256 |
| --- | --- |
| internal/recovery/installation.go | FA63684B6677E27010C9F7C8D6156D007BE62BF8A4D26303E0DCA56F25BB1BFA |
| internal/recovery/installation_test.go | 10CC71B4902C7329C679504F75E2500738E69C1EE63BDE2C69048FDDD2AA0889 |
| internal/identity/recovery_installation.go | 3606BDB7E69FE457C1F930C8D97D1014FA60405B1F507F080F419D6C53555C3F |
| internal/identity/recovery_installation_test.go | 40FEC5E5437009A94F670A9F76AD07D4CBB2085D2F582556361C1F738BB5B3EB |
| internal/identity/recovery_installation_postgres_test.go | 17076C884D39476EAD22F511CD328A2D753522AC0C6AE0EE55BE2C317542476E |

## G0 与执行授权

[start.json](start.json) 是完整实际 START 记录；[g0.json](g0.json)、[method-readset.json](method-readset.json)、[readset.json](readset.json) 区分本次刷新与原同任务读取依据。Role、主 Skill、TDD、delivery Workflow、coding/no-hardcode/task/multi-agent/checklist 与必要 Go Profile 全文已刷新；正式 finding、指定设计条文、Task/Plan/两TP、原5完整源码、真实 Install/manifest validator/admission 消费接缝及 reviewer 测试片段已实读。requirements 与 ADR-0027 是原任务已消费合同，本次另补读相关正文，未改变技术合同；补读时点不倒签为新的写前行为。

G0 先向 Main 报告最小方案、约束及停止条件，在本 D root 保留候选后才写两份 owned 测试。入口23全部原 SHA 匹配；随后每个真实作业均对 peer18 前后严格守卫。修复阶段只允许 own5 与旧接收值不同，不把 Main 的旧冻结文件改成自己的新值。未知 provider/native/root/批准主体不借软件 fixture 冒通过；无新审批要求或旧 pending Gate 重开。

## 失败→修复→实际回归

| 实际作业 | 结果与范围 |
| --- | --- |
| behavior-red-current | 原 production reader SHA 保持旧值；新 permanent 回归20叶中5叶真实行为失败、15通过，无 skip/unfinished；完整 raw 保留 |
| behavior-green-current | 最小修复后同20叶全部通过 |
| repair-normal-regression | 457叶全部通过，0 fail/skip/unfinished |
| repair-race-regression | 与普通模式457叶逐名一致，全部通过，无 race 报告 |
| local-vet / local-build | 原项目局部包真实 vet/build，exit0 |
| owned-format-check / git-diff-check | 原5 gofmt 输出为空；git diff --check exit0，既有换行提示原样留存 |
| retained-old23-format | 精确原23输入 SHA 与旧作者执行一致，stdout/stderr 原字节一致，实际 formatter exit2；仍为 FAIL，不是豁免或全仓格式 PASS |
| physical-own64 | 明确当前具名 PG target；本轮64精确 names 实际 SQL，exit0，全部存在并保留 |

Red 中失败的是 anchored 资格正控及原拒绝提前遮蔽的 Unapproved/RevisionDrift/CommitmentDrift 原因断言、真实 SQL anchored normal restart；全部可运行，不是编译、缺方法 stub 或夹具搭建失败。无此类失败冒 behaviorRed，详见 [diagnostics/classification.json](diagnostics/classification.json)。

457 的范围是：原卡55叶（Recovery37、Store2、PG16）完整保留，新增 reader13 + PG7，Ptolemy 不改期望的原片段6叶，加 Runtime Install105/Admission271 共376叶。每模式是457，不把重复执行或普通/race相加成更多覆盖。新增 reader 组核合法两 kind、未知 kind、prepared、Recovery 初始历史、批准缺失/错批准、错 digest/binding/initial head、revision/commitment drift 与缺 revision。新增真实 PG 组核 genesis/anchored 公开 Install→同 Store→正常重建→ordinary lease，以及无批准、prepared、错 binding、缺 Article 初始前缀、外部 Begin 无 Store 行仍关闭；前后完整 Store、commitment/token、metadata/media/coordinate 计数不变，真实 Stop 返回。原真实删除后重启及真正 BindLegacyPublication、一次 NULL 补绑/精确幂等/负例、forward SQL 等最低 P16 均重新执行。

普通/race 主命令相同，仅 race 标志不同：

```text
go test -p=1 -parallel=1 [-race] -count=1 -json -timeout=300s -tags=integration -overlay=<ownD>/packet/probes/test-overlay.json ./internal/recovery ./internal/identity -run '^(TestRecoveryInstallation(Binding|StoreBinding|Postgres)|TestReviewerS413BoundReaderKinds|TestReviewerS413AnchoredEmptyRecoveryRestart|TestRecoveryRuntime(Install|Admission))$'
```

overlay 仅替换两份 owned 测试文件：当前原始测试字节 + reviewer 原 fragment 原字节；没有 production overlay，未改变其断言。保留 [probes/provenance.json](probes/provenance.json)、完整组合测试和实际 argv。所有原/新增作者测试均在这些相同前缀内命中，另有不带 overlay 的20叶 Green。普通/race、Green、vet/build/格式所涉及的原5前后 SHA 全部与本次冻结一致；没有套用旧作者 recovery2/早期 identity3 的复用边界。

[leaf-evidence.json](leaf-evidence.json) 提供完整逐叶结果与原55名称保护核对，[repair-validation-summary.json](repair-validation-summary.json) 提供范围摘要。没有重复全仓大矩阵；Main 后续统一检查与原 Ptolemy 独立复审各自保留。原 reviewer 首个 bug 停止后的其余候选/未执行范围仍由其正式报告持有，作者不把本次绿色执行改成独立全风险 PASS。

## 资源、schema 与原始证据

实际14作业全部保留 executable/argv、完整 stdout/stderr、source23 前后 SHA 与原5前后完整字节。Windows helper 亲读后从本原作者源码复制，runner 重写为本轮 root 与 peer18 guard；[runner-provenance.json](runner-provenance.json) 与 [runner/](runner/) 保存真实执行源及原版本。旧脚本未执行，旧输出未覆盖；归档 runner 是实际执行源，禁止将其当作获新权限的自动重放入口。

仅本作者原独占 D 缓存复用，TEMP/TMP/GOTMPDIR/profile/work 在本修复 root。GOPROXY/GOSUMDB off、GOTOOLCHAIN local、GOENV/GOWORK off、readonly、GOMAXPROCS1/-p1、一次一个 Go parent。没有网络下载；既有32 hash输入清单留存。goimports/golangci-lint 当前本地缺失，未安装、未伪称执行，见 [tool-availability.json](tool-availability.json)。

[lifecycle.json](lifecycle.json) 记录 suspended-before-Job-assignment、实际 parent handle Wait/process_wait、kernel active0、两 EOF 和 pump thread join，失败作业同样成立；没有 Popen.wait 替 whole tree、未杀未知 PID。作者观察的28个 PID是采样集合，不是全部 kernel child 枚举或主机扫 PID。Main 的33已知 PID核收另列其来源。`recorder_after_join_at` 是真实所有 join 后记录时刻，不命名成精确 OS Wait 时刻；旧独立原 result 的字段/时间不补造。两次授权 formatter 中 red-format-tests 确实改变 owned 测试字节，repair-format 实际无变化，原布尔结果不改写。

测试同步持有并关闭本轮 Runtime/lease、SQL rows/tx/pool 和 fixture I/O；64个 schema 的 cleanup 物理读均 acquired0，实际 Stop 返回。没有新脱离 goroutine/callback；每个命令的完整 job 返回之后才启动下一作业，最终14全部终结后先 release，再封包。

仅从 Main resource 读取 identity_dsn 入内存环境，不打印、写 argv 或复制 DSN值进包。psql 修复本轮使用显式 host/port/user/database，无 ambient/fallback。新 schema 来自本轮 Red7 + Green7 + 普通25 + race25，共64；原作者111与 reviewer4属于原证据，不扫描或混入本轮。见：

- [schemas/names.json](schemas/names.json)：64精确 names；
- [schemas/physical.sql](schemas/physical.sql)：目标校验与逐名字面 SQL；
- [schemas/physical-receipt.json](schemas/physical-receipt.json)：真实目标与64行存在性、表数、安装/run/Audit计数与revision；
- [raw/physical-own64/](raw/physical-own64/)：实际 argv、完整返回、进程/pipe join；
- [media/](media/)：14份本轮 synthetic checkpoint/tombstone 原文件副本，原文件、cache、数据和所有失败原件继续保留。

未启停 Main PG、未 DROP/清旧/schema扫描、无真实provider/native/root/恢复服务/浏览器/部署/E2E、无 Git 写/清理/移动/归档或 SSOT写。共享工作树仍有原他人修改；Git摘要会折叠未跟踪目录，不用它冒本卡新增文件 diff，原5独立前后字节与差异另提供。

## 交回与限制

修复作者结果已就绪；**F001 仍 OPEN，等待原 Ptolemy 对新精确冻结复审**。Main 的 release接收与 freeze 不替 CodeGate。本软件闭环只证明设计允许的外部批准/CAS/media/Journal snapshot适配与真实 SQL/Runtime/reader，不证明真实旧链HMAC介质、批准主体、ACL、生产 provider、长期 root 生命周期、真实恢复或业务/人的接纳。

原23格式 FAIL2完整保留，当前5格式通过不外推全仓。原先无关较长 codec 测试函数未顺手重构；本轮8个新增测试函数均≤50行，见 [self-check.json](self-check.json)。完整 cache-free manifest覆盖报告、源/差异、失败、当前 raw、lifecycle、schema/readset/限制；原 cache/data 均保留在包外。释放后无需、也没有重新运行 Go/SQL 来封包。
