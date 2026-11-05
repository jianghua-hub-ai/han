# S414 R04 作者实际修复 — STOP / 待原 Ptolemy 独立复审

固定 Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca。本轮全文消费 s414-r04-author-start.json，复用本人17项同版 Role/Workflow/Implementation/GoTDD/规则/profile；既有 R03 真 finding 与 candidate 已核，未新增 G0/批准门。

## 实际修改与两轮自查

只改 recovery_restore.go 与 recovery_restore_test.go。生产文件与 Main 已审 C0CB4B… candidate 精确一致：Run defer 先取已有主panic，经真实 image cleanup 后按原 any 值重抛；secondary cleanup error/panic 私有保留，Error 文字泛化，errors.Is/error identity 与非error opaque value 可取回。无主panic时 cleanup-only panic 原值可见；普通 error+cleanup error 合并次序和 unknown/no replay 保持。原 acquisition ownership、owner+error恰一次转交、checked-FD defer均未改。没有新公共Runtime/API/权限、SDK/provider/fallback、协议、第六源或buildtag。

第一轮自查核主/次cause与opaque slice身份、private文本去敏、正常error因果以及cleanup-only panic；第二轮沿实际 public Runtime.Enter/Begin、原permit/lease/sourcehandoff、Start/Wait/pipes、image terminal、独立target读取及真实PG targettx取消/join核消费者。它们是作者自查和真实执行，不能替独立评审。

测试仅在原fixture内增加 Start返回UTC与实际 Wait/pipe observer UTC日志，满足本轮PID/start/end证据；无真实OS创建时间或全host枚举声明。Main已审候选的原8补测保留，原断言未改、无新增import/跨package依赖。原3source与24保护源码/模块路径前后完全相同。完整两文件diff、before/after与observer-adaptation.json随包。

## 实际 Red → Green

| 执行 | 普通 | race |
|---|---:|---:|
| unchanged production + exact compound probe Red | 1叶真实行为FAIL | 1叶真实行为FAIL |
| 原102 + candidate8 + compound1 Green | 111 PASS | 111 PASS |
| 原 RestorePG exact top | 7 PASS | 7 PASS |

Red实际记录 primary_preserved=false / cleanup_caught=true，资源terminal、FDclosed、ownerClose1、Begin1/child0、Stop/Close均达到；不是编译/磁盘/setup错误。冷 D build cache 的首次编译耗时记录在ledger。Green exact probe变为 primary_preserved=true / cleanup_caught=false，全部物理终结控制仍真；single cleanup-panic与普通error causes原控例仍实跑通过。没有代码/fixture/编译失败需要绕过；整理diff时纯JS旧helper不可用，仅重建文本helper，不影响源码或测试。

五个required组每模式非零：RestoreDriver95、DeploymentExclusive9、原image/lease Reviewer3、原FDpanic Reviewer3、compound Reviewer1。全部原102名字保留且pass，Green/PG均0fail/0skip/0unfinished；111是实际叶数，不以list或package pass凑数。三份 immutable fragment DBDEDF67… /24D710DF… /613CD0AE…全部字节SHA保持，Red完整旧prefix与Green完整新prefix只叠加test overlay；生产overlay0。

## 新D目标与实际join

SQL preflight独立确认127.0.0.1:51934/han_b13d_restore、data_directory=D:/Temp/han-b13d-main-pg-r02/data、初始本卡schema0；只从Main资源精确读取当前restore_dsn私有字段，未使用旧C DSN、其他库或ambient secrets。PG每模式原7用例均实际通过，fixed0001–0010/fresh quoted schemas保持。结束只读SQL按本轮14个精确名回读，全部物理存在、服务start与目录同preflight；该D库本卡schema总数14。旧C63仅历史，不算本轮。不DROP、不cleanup，不操作Main服务。

所有10个Go parent实际Wait、输出流close；连同gofmt/psql/Git只读工具共18 parent全真实结束。66个受控当前testbinary/copy child均有本轮Start-return UTC与实际cmd.Wait+PipesJoined UTC配对；未用Exit0代替join，也不按PID gone猜终结。原held child/pipe、callback、image-owner和target transaction控制完整保留且通过。known-pids.json、process-ledger.json、child-joins.json与逐叶raw相互索引，无待运行owned会话。

TEMP/TMP/GOTMPDIR/GOCACHE/GOPATH/GOMODCACHE及fixture/work全部在本轮独有 D:/Temp/han-s414-r04-jkgwL1。既有固定版本module输入只读复制到D独有GOMODCACHE，Go不使用C活动cache，不网络安装；原modules hash未变。offline/local、GOENVoff/GOWORKoff/GOTOOLCHAINlocal/GOPROXYoff/GOSUMDBoff、GOMAXPROCS2/-p1串行/count1JSON。普通/vet/list DSN空。36份raw的实际DSN/密码/canary扫描均0，不存child私有stdin/stderr。packet无exe/cache；root下D cache/work/fixture原样保留。

vet普通+integration、5组与PG list、原2gofmt写/原5fmt检查均通过；Git只读status/head/branch/no-index whitespace检查已实际执行，diff-check因差异exit1但无空白诊断，非测试FAIL。共享root和实际cmd消费者已由normal/integration tests+vet编译；无必要重跑旧319/Linux/global7。无Git写/SSOT/design/metadata/oldIssue写，无清理、网络或子委派。

## 最终hash与停止

- recovery_restore.go: C0CB4B012F5EE1DE3CD346DD768D14C50AAA60AB8A0A549FA9098997B0668BBA
- recovery_restore_test.go: 549196E579A254FC09CCADFD2500E1FA35FF4330EDEF224723A9B69E087B073C

source5与raw全SHA见receipt/delivery/manifest；完整原2diff见repair.patch。原R01/R02 independent CLOSED只按既有结果保留；作者不关闭R03 finding/Task/G4。native enable Gate仍CLOSED，软件fixture不冒真实native锁/ACL/no-follow跨进程handle/receipt/process-tree/realrestore。rootS416长期idle owner、fullS413/E2E/人工验收保持未验。

STOP：全部本轮owned命令/callback/goroutine/pipe/resource已按上述证据结束，soleGo/RestorePG修复窗口交回Main，待原Ptolemy在最终freeze独立复审。
