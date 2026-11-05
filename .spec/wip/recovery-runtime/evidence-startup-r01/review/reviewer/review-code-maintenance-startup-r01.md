# 普通 server 显式 maintenance 拒启：独立代码评审 R01

**结论：PASS，仅限本次两文件有界启动拒绝单元；两轮检查完成，finding 0。** 不重开S10.5，不代完整恢复runtime、HTTP503、Consume、normal恢复准入、成功开流或人的接收。

实际 Reviewer：**Dewey / add-reviewer**，Agent ID **`01a180fe-c3bb-7661-86fb-06af4f76ba1b`**，原非作者Reviewer；未再委派或修作者产物。固定 `D:/AID/VSCode/Codex/Han` / main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。报告/证据只写本新个人Temp。

## 固定对象与依据

| objectType=code | 最终 SHA256 |
|---|---|
| `cmd/han-server/main.go` | `1F2E43907A4FDAF4017A937105ED7044F9175069D28913A988FE799597C91B8D` |
| `cmd/han-server/main_test.go` | `BE69516324272EBC2171DF8632D931E18ACBDD3B3828CB55E9042EBBEB84BA37` |

最终作者freeze来自 `naming-final-e150a43d096a496aa98744d94bb5a3fb/final-freeze.json`；独立逐项核live、最终副本与以上hash一致。持久作者包 `.spec/wip/recovery-runtime/evidence-startup-r01/root-author/index.json` SHA **`8639B75DF4DFCADC90C24897FFAF9D0823F090306E38611AD4B22032B07FE281`**，55payload已核；归档 `.go.txt` 与map指定原Go bytes/hash相同，不新增可编译/格式扫描源码。

实际采用来源：Pascal r03咨询 SHA **`C49077BA62830B936FDF35A66E5A408BBA0CA7331FD514C3725F14FF2B4E858A`**、Main adoption SHA **`1CD35AF08F65D266C3BE20EFDE8C5A228237150A23592E9F9F9C78832F861C78`** 及实际 `lead-code-start.json`，均已读取。只采用普通server拒启第一建议，未把咨询当独立设计Review、实施完整runtime或批准真实恢复。

直接依据已核：ADR18 `framework.md:388/405` 的恢复完成前不接业务及失败停服；ADR20 `content.md:126` 的重放/账号/Journal共同完成后才开放。当前实现以拒启满足本小单元既有硬预期，不改变后继503/恢复旅程义务。

## Round 1：真实增量、测试与版本桥接

生产唯一变化为 `main.go:377–380`：成功loadStartupConfig后，如recoveryMode为maintenance，先调用原 `clearStartupSecrets(&config)`，再返回内部非nil启动错误；位于 `prepareBusiness/prepareArticle` 前。独立删除这四行后，整个main原字节精确还原原 **833D3E1ECBC39C6CED1E91D59E63651A32A4AD3A1D04C1CC7662560244BDA6AD**，因此normal、argv/requeue及旧资源/错误代码未改。

原测试 **4EF46AA9AF542C15FEF8D4688208ABDD75A6674C0A6F134D6645C86F4068D73E** 全文件是最终测试的原字节前缀，仅追加162行。已完整读新增 `TestRun_RecoveryMaintenanceIsolation`，核合法maintenance、normal对照、非法配置优先级及六个原argv对照。fixture先真实解析校验，不能靠非法配置提前失败假装新门禁生效。

作者首次行为版测试 **3F82612094F488B02B0D7254700B5BD81316FA920ECEF29D22D4E485065AB8AB** 到最终BE69仅新增函数内两处lambda的 `k→environmentName` 和byte loop的 `b→valueByte` 绑定/引用，独立机械映射后逐字节相等，无其他变化。作者37项＋naming16项manifest均已核。旧28叶startup normal/race及87叶S105兼容normal/race仍归首次行为版，经精确命名桥接引用；作者最终BE69另直接9叶及vet通过，不冒最终版全矩阵重跑。本人下列运行直接用最终BE69。

**秘密所有权：** 新分支清理调用与原helper全文已核。`clearStartupSecrets:2037–2048` 清零loginBucket/source缓冲区、调用原digest/article keyring清理并释放其引用、清空SMTP字符串引用。底层两个helper仍清零实际byte切片、删map/释放active与retained；函数体未改。loadStartupConfig失败已有defer负责清理；成功后本新早退接住所有权，尚未向prepared对象复制。

新maintenance未进入prepare，raw中 **observed=0 / cleared=false不是动态清零证明**。本结论对新分支清理的依据为实际调用顺序＋原helper语义；独立重跑既有合成缓冲区清理回归与normal完整fake流的六缓冲区观测，作为helper/旧消费者回归证据。没有新增生产观察接口、读取真实秘密或声称不可变string已物理擦除。

## Round 2：实际入口、错误与直接消费者

- `main:169–181` 先用原selector分派argv；无参数才调用 `run(...,productionDependencies())`。生产dependencies明确提供composeBusiness，故实际进入runComposed；legacy测试路径不是本次生产入口。prepare前新拒绝不落入password-reset命令，normal与原dispatcher生产字节完全未变。
- 原mode解析只接受精确normal/maintenance，配置错误仍在门禁前返回；未新增mode、authority、配置、公共错误码或HTTP面。
- 合法maintenance实际 prepare两项、parse/connect/ping、migration/bootstrap、article open、compose、listen/newServer、coordinate及worker factory/handoff全0。配置getter的 **DSN字符串读取1次** 是原loadConfig行为，不能写成“未读配置”或“建立了DB”。测试使用合成map，未读取实际ENV/DSN。
- factory/newServer/coordinate全0及唯一root调用图支撑维护分支没有Serve/Reconcile/Start。normal对照fake listen2/worker factory与handoff3只是受控接缝，实际Serve/Reconcile/Start仍0，未启动网络/业务worker，也不以这些计数证明生产恢复运行。
- 新错误为固定内部错误；`lifecycleExitCode` 对非nil返回1，main实际只打印原 `han-server startup failed` 再os.Exit(1)，不输出内部error或配置值。Reviewer新独立probe在进程内验证真实 `run`＋退出分类；未运行产品binary/业务child，不冒实际OS进程退出测试。
- 此处没有HTTP响应，尤其不计HTTP503；也没有RecoveryRun/Journal核验、Consume、外部authority或normal成功准入。normal尚缺unfinished恢复核验是采用来源明确保留的后继范围，不由此flag授权实际restore后改normal开放。

角色/完整add-code-review本次重新取得；checklist、abstraction、安全红线、coding/no-hardcode、task-execution/multi-agent及go-service对应规则复用本链完整同版正文并核摘要。没有另开Workflow、重审旧S10.5/Domain或扫描无关main模块。

## 独立运行与来源分层

全部命令在固定仓库，只通过个人Temp overlay固定Root2并追加一个退出分类U probe；没有修改产品。显式隔离离线进程环境，无实际ENV/DSN/PG/DDL、网络listener或产品/业务child。每次运行前后Root2均与冻结hash一致。完整argv、filter、结果及原始输出见 `commands.json` 和对应run收据。

| Reviewer实际验证 | 结果 |
|---|---|
| 同一退出probe＋旧833D生产前像 | 1叶预期RED，exit1：prepare1，证明旧版仍进入准备；不是编译失败 |
| 最终版有界startup/清理/argv normal＋独立probe | 32节点 / **29叶全部PASS**，skip0，exit0 |
| 同选择器race | 32节点 / **29叶全部PASS**，skip0、无race诊断，exit0 |
| `go vet -mod=readonly -overlay <Temp>/vet-overlay.json ./cmd/han-server` | exit0，stdout/stderr空，overlay为最终原测试无独立后缀 |
| `gofmt -l <Temp>/fixed/cmd/han-server/main.go <Temp>/fixed/cmd/han-server/main_test.go` | exit0、空输出；未写格式 |

最终maintenance合法配置拒绝；normal保留原顺序/原synthetic stop及六缓冲区清零；非法配置仍保留原配置错误；独立probe为prepare0、non-nil、lifecycle_exit1及原去敏启动失败。没有以“任意error”掩盖prepare已穿越。

原作者G0索引 **7BB00EFE7EC33AD6C3D2FD1A560EEDF3F402B9446618E463B558E647173B73AF** 的24payload和真实compiled RED已读核：1行为RED/8控制PASS，旧fake listen2/worker factory与handoff3穿越；实际Serve/Start/Reconcile0。一轮fixture编译失败单独保留，不算RED。

Main最终Manifest索引 **D499E1BC40174B0DDE68AC8E52DF545F4C379214C8B596CBC13AD4A162F97F99** 的23payload已核。modverify/vet/build/contract/fullrace/fullunit六项exit0是Main同最终1F2E/BE69运行，非Reviewer重跑；它们可直接覆盖最终测试名，不再仅靠naming bridge。**全workspace `gofmt -l .` 仍FAILED2**，限定例外仍是原S105 immutable16paths/8parse；Root2格式净，新归档使用.txt。例外及原始结果按Main来源保留，本报告不称全七配置检查PASS，也不格式化/删除历史证据。

## 交回与限制

本次findings为空，未发现需要原作者返修的新阻断。只给固定Root2软件单元的独立结论，不把多个旧局部PASS相加。三父卡S1.5/S4.4/S8.2、Case18未关闭；没有新的恢复执行授权、部署、真实SID/CAS/备份验证或人的accepted/checkpoint。

本新报告不改已有报告、产品/Docs/Tasks/Plan/Manifest或Git，不清理任何材料。**完成STOP，交Main现场复核和收发，不自行启动后续phase。**
