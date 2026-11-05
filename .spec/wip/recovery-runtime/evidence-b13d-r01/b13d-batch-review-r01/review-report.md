# B13D 三卡整体独立 Code Review / G4-batch R01

**结论：当前精确18源＋6文档的有界软件批次 G4-batch Review PASS；new0，无已建立的开放 HIGH/MEDIUM。**

原 Dirac / add-reviewer，actualID `01a18663-d9ac-79c2-b843-ba6dcca3de7e`，非 Zeno/Noether/Chandrasekhar 三作者。固定 Han/main/HEAD6da8abb…；对象 SHA 全列 `source24-entry-exit.json`。本结论来自两轮真实跨卡审查、必要新组合实测与精确当前旧实证，**不是三个个人 CodePASS 相加**。Task关闭、技术采用、业务接纳和人的 checkpoint仍由原责任主体决定。

## 两轮实际范围

Round1：直接核本批三原卡3/10/5文件与原requires，无相互requires；B13D scope/Gate及r56/r27/TP/已采用mapping边界。完整阅读 RestoreDriver/Exclusive生产正文；定向阅读当前 Identity factory/forwarder、rawPhase/Store、Bind/finalize/canonical/failed-source/cohort/terminal/Exit/ACK相关主体与真实消费者。Article原完整10源/157k差量及F001 R02按精确同字节复用，并重追本次接缝。没有重复所有单卡完整评审，实际读取范围见readset。

Round2：沿实际 Runtime 的 closed/drain/exclusive→Prepare→committed Begin→Replace/Verify→Bind→Identity proof，以及canonical caller tx、publication source/普通历史读、0010/0011共存、unknown与资源清算追踪。依原最低层核6/5/5风险组，直接补原分开harness没有共同覆盖的两段组合。原finding均由原Reviewer闭合并Main核收，本轮按当前SHA及真实接缝判定，无自动重开/替作者修复。

## 跨卡判断与直接证据

| 接缝 | 具体判断 | 当前证据 |
|---|---|---|
| Restore→Identity | 同一真实Runtime raw permit、同Store/Runs；已独立核验的12字段回执进入真实Bind，Begin/安装/backup/receipt与唯一Audit持久绑定 | **本角色新cross**普通/race各2PASS：正向真实Bind后明确停止；篡改backup拒绝且run/binding/Audit各0。每次Begin1、真实软件child2，Stop/driverClose/lease/pool收口 |
| Identity→Article canonical | 原User→Store→review/Article锁序；真实ArticleAccountPort复用caller tx，审计/Article/用户相关事实原子；Complete仍按原C＋逐P与原cohort | 当前Main IdentityPG86/模式中真实ArticlePort/混合D0-ND0/回滚/ACK组；代码 `recovery_execution.go:1110–1161`→`deletion.go:412`→`command_service.go:342`；新cross另核真实ArticlePort的caller提交/回滚可见性 |
| 0010＋0011与publication历史 | 顺序固定0001–0011在同一新schema实装；真实Source/Replay/Publish/ReadCurrentPublication，真实同pool Identity snapshot及canonical写后普通历史closure兼容 | **本角色新cross**普通/race各2PASS：提交/回滚对照；observer证明port未私自commit；正常ctx历史仍只做必需读，Replay/Facts/IDs不增，安装来源不漂移 |
| 真实来源/unknown | failed reader仅lookup、返回深拷贝；原六字段/cohort/actor/time与原域classifier核证；Recover与resolutionAudit同tx；writer结束后才核源/另一物理连接 | 当前Main IdentityU77/P86普通race、Confucius R02原10＋合法2独立oracle；本角色直接追writeVerified/readFailure/resolve/Complete/terminal/Exit代码 |
| 资源与部署所有权 | 宿主Runtime mutex不跨外部driver执行；driver私有锁序列化并由取消/Stop协议收口；未知不重Begin/Replace，源/FD/child/pipe/targetTx真join | 当前S414 R04/full currentMain104local/7PG与实际 held/error/panic controls；本轮8个真实controlled child生命周期与246个own handle实际Wait；不是整个Host/native设施证明 |
| constructor/forwarder/后继 | 同executor维护接口、一次seal、构造零I/O、opaque当前grant；普通历史/Verify不要求未用Import/失败源。0011 NULL安装后继绑定仍归S4.13，长期idle/control/root收口归S4.16 | 当前相应constructor/Import-only/typedFactory/History叶、静态接口和新组合；无新RuntimeAPI、无生产provider/fallback、无未来root注册 |

详尽逐项来源、具体函数/行、当前raw与未验边界在 `coverage-map.json`。新cross的批准、Genesis、软件lease/目标witness是明确synthetic external facilities；真实的是本批产品Runtime/Driver/tool解析与软件child、Identity executor/Store、Article Source/finalizer/publication/SQL。第一组故意停在实际Bind后，不冒完整Finalize/trust/SMTP/root；第二组使用原Article真实Runtime producer的显式fixture backend，并独立读取真实同pool IdentityStore，未声称已装配完整production Runtime/完整Install。

## 实际执行、复用与所有权

本轮新4条产品命令＝两组cross普通/race各2叶，**8次产品叶执行全PASS、无skip/unfinished/race诊断**；另2条物理SQL收据PASS。8条实际Go尝试中，6条exit0；2条为明确保留的探针准备诊断：首次错误引用AnchorSnapshot字段而编译失败（产品叶0）；其次旧Article synthetic user未给0010所需generation，在fixture INSERT失败（2叶）。均不是产品Red，也未删raw/放宽constraint/改旧断言；只在新增Temp夹具显式给合法generation，真实产品代码/原18源未改，随后同一预期正/负及提交/回滚对照两模式成功。见diagnostics与全部checks。

只新增virtual `_test.go` overlay，原18源/现有测试前缀/断言均未替换，production overlay0，虚拟逻辑路径仍不存在于仓库。所有执行GOMAXPROCS1/-p1、serial onejob、offline/local/modreadonly；32固定D输入逐hash复制，TEMP/TMP/GOTMPDIR/GOPATH/GOMODCACHE/work/fixtures均ownD，复用ownD cache。仅按需私读两个named DSN，IdentityPG未另跑。没有重复完整77/86/54/58/104/7矩阵。

每条尝试（含两失败）父Popen.wait已返回、kernel Job active0、已取得handle实际Wait、stdout/stderr EOF后join：**8个job，246进程记录/246handle Wait，8个controlled child的创建/退出FILETIME＋产品Wait/PipesJoined-return时间**。少量诊断/race子进程非零exit原值保留，不冒称所有子进程exit0；无pending own callback/Go/SQL/PG作业。不检查或宣称Main cluster/其他Host全join。

本轮**10个新schema＝Restore4＋Article6（含2个fixture诊断schema）**均单独精确SQL核namespace/table/实际run-binding-Audit及publication数据、PG180006/51934/命名DB/dataD、Rows关闭/acquired0，全保留，不混入Main302、作者或旧C名单。8命令前后及最终18源＋6docs全部稳定。

当前Main20条（8 manifest含ownedfmt＋12targeted）stdout/stderr/hash/Wait/Jobzero/IO收口记录已独立静态解析：每模式 IdentityU77/P86、ArticleU54/P58、RestoreU/local104/P7，普通race逐名终态map一致，required0fail/skip/unfinished/race。风险组严格按原最低层映射；Restore私有stdin组原为U/local，未凭空要求PG组。Main302新schema原始名字与physical最终172/116/14逐一匹配，Identity23表等数据来自Main实际SQL；这里不冒本角色重跑。三份当前Code报告及Main接收属于单卡来源，不是批次通过的加法。

Manifest保持 **6PASS／根gofmt FAIL2／owned17 fmt空PASS**。根stdout `D01225B146C06C238B38AEA344662D8253842CEE671EF68137BCC4B66C9379A8`、stderr `53258F7B21EF40C6BE7E8E671216EEF5DD0441B27314FA806AE24F5C2CB010FD`与旧23精确一致。当前18技术有限处置来自Main的standing授权，不改FAIL、不扩大豁免或记全7PASS。Main全树guard4842/HEAD/staged0为Main证据；本角色独立取得的是24对象逐次与最终稳定，没有执行Git/全库状态盘点。

## 性能维度与未验边界

本批的必要functional成本已核：restore全文件/镜像流式hash是O(bytes)、receipt有65536界；Identity必须保完整原账号/cohort和当前未决集合，相关target/domain/Session/Audit图有界于实际集合，完整canonical仍随全run/成员数据增长，独立ACK需第二物理连接；Article历史完整prefix两次读随J增长，不能截断或改用latest优化。外部源IO在SQL锁外并真join，configuration mutex不跨IO；ctx/锁序/回滚/失权/真实held owner已有当前下层实证。精确自有schema回读避免把共享历史扫描当产品成本。**CPU1/-p1和小fixture通过不是容量/P95/大历史性能测量**。详performance-applicability，未伪造基准/阈值。

仍未验：真实provider/ACL/审批主体/SMTP、native锁/进程树/真实restore、HTTP/root、完整S4.13安装/NULL forward绑定、S4.16长期idle/后证失败handle完整装配、真实部署/E2E、人的接纳。它们保持原后继责任和被关闭分支，不倒挂为当前组件新前置。原unknown邮件保护窗/hold/deadletter、phone原值/缺失行残余及S3.3/S3.5窄范围不扩豁免，S3.6不改。draft与人的unknown不伪签，原Tasks状态未写。

**STOP：有界G4-batch独立Review完成，new0；交Main实际接收后决定仅原三卡关闭。无SOURCE/SSOT/Task/设计/Git/服务写、无cleanup/DROP/移动/下载/子委派。Main负责最终cluster停止；本角色不再自动运行。**

cache-free packet：`D:/Temp/han-b13d-batch-review-r01-065d128fca`；执行scratch/cache/work/media保留 `D:/Temp/han-b13d-batch-static-review-f949eeddf3`，未搬移/删除。
