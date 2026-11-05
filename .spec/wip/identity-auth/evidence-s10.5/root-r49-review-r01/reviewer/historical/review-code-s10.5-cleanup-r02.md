# S10.5 原 F002 局部修复独立复审

- 日期：2026-10-28。
- 实际 Reviewer：**Dewey / add-reviewer**；真实 Agent ID **`01a180fe-c3bb-7661-86fb-06af4f76ba1b`**。原非作者 Reviewer，无再次委派，无作者代码修补。
- 固定工作区：`D:/AID/VSCode/Codex/Han`；实际分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- **原 `S105-CODE-CLEANUP-R01-F002`：CLOSED，限下表固定修订的网络/TLS deadline 缺陷。** 两轮定点复审完成，新 finding **0**。
- **F001 仍 OPEN；本次未复审其新合同或关闭它。没有完整 Root / 七文件 CODE Gate PASS。** 新采用方向及 ADR/FD/TE 在途文件不作冻结输入，不借此替当前代码解除 F001 或 Root-ready HOLD。
- 唯一输出为本次新个人 Temp：`C:/Users/J00010385/AppData/Local/Temp/han-s105-dewey-f002-recheck-916420786a3f/`。原正式报告与产品文档保持不变。

## 1. 固定对象、完整前后摘要与原失败链

输入索引 `.spec/wip/identity-auth/evidence-s10.5/f002-author-fix-r01/index.json` 的 SHA256 已独立核为 **`E20AF765DF025F41EAFDDDA9702D45574103CB601AF3B4AE69AA8ED7EBF134D4`**；**31 项 payload** 全部逐项验 hash，并原字节复制到本 Temp 的 `inputs/`。开始、两条测试各自运行前后、报告结束核验均要求 live Root4 与冻结绿版一致，发现漂移即停止。

所有对象为 **objectType=code**；本次只审 F002 diff 及其直接消费者，不把完整文件列入索引等同于整文件 CodeGate。

| 路径 | F002 实际修改前 SHA256 | 本次固定修复后 SHA256 |
|---|---|---|
| `cmd/han-server/main.go` | `21CEC445C412EE909062DCF9E81C8D4D6ACD0CD26842E7AEE0E3533EF29BA543` | `2C2D404F344E5A8E4831C34FB4C240B234B9B099482F13897D38EA470B37FF79` |
| `cmd/han-server/main_test.go` | `AB06CEFDC4368BCEBB9907D8DDCC3984D8681D9BC7973477A602C9D0E2861496` | `D46BA87371795849FD525454FB4E67942E7C3002051867B3F335D55EBE0E83B2` |
| `cmd/han-server/maintenance_stdin_windows.go` | `620583ABAEB1551761359F93EAA7A4D0B79BE8A4E312B2FFD9C1D69A82663A43` | `620583ABAEB1551761359F93EAA7A4D0B79BE8A4E312B2FFD9C1D69A82663A43` |
| `cmd/han-server/maintenance_stdin_other.go` | `A711C46F5D0AD017CA54569BC8862F30EFCDF2989EA13B0C856A47F615D854D9` | `A711C46F5D0AD017CA54569BC8862F30EFCDF2989EA13B0C856A47F615D854D9` |

修复前失败回归快照的 main.go 仍为上述 **21CEC** 完整值；加入 TLS 真实 SDK 回归、尚未修生产代码的 main_test.go 为 **`ACF1A0EBF283D5E8095A5DC0C9D1CADAB1E88823AD1DEC1CC659531F403EE143`**。它与 F002 修改前 AB06、原问题首次固定对象中的 `97F71B19DD5D5EB3F02370E04A7C3A8AD5C0E64BDF861A7D86F421B5A4DDB5C6` 是不同测试修订，不混称同一原像。

原正式问题报告 `.spec/wip/identity-auth/review-code-s10.5-cleanup-r01.md` SHA256 **`8C1063321915846CA299E53DBC58CF037507EFE8015819A2E1EF84599AC0359A`**，保留两个 HIGH/OPEN 的历史原文。原 F002 位于旧 main.go:4302，同步 closeSDK 期间无法执行下方超时中断，TLS closeNotify 在已到期后又设 now+5s deadline。此次只以新报告记录它的修复关闭，不覆盖历史失败。

上游要求仍使用已冻结的 **design r48**（SHA256 `69309241DFAEB448383EE2C63CA494B4ABAD8CE0AC5DE29A1555F7D54A5CCEB5`，2381–2405 行）及 **ADR-0026 r03**（SHA256 `BCB5707C90E008D4200087750361A6A6C648BCF0822691A3BD8BDBF8EB0F6442`，预算/七 ENV 条款）。未读取在途 r49/r04/TE23 作为本次技术依据，未采用新合同替旧网络要求开例外。

## 2. Round 1：精确 diff、deadline 与完成事件

原作者为 Root Wegener。Reviewer 实读作者修复说明、freeze、精确 diff、RED/GREEN/race 原始证据，并从 `*.before` 与 `green-snapshot/` **独立生成 diff**。生产 main.go 仅两个变更 hunk：旧 4184 附近 transport deadline 管理，以及旧 4295 附近 SDK Close/cleanup callback 管理。两个平台 adapter 无变化；普通 server 未调用这些维护 owner hook。

独立 diff：`independent-main.go.diff` SHA256 `D9DBEE037D169D08E7D7DAD673FCB44B297D4AB24A3FA1AE0BA6F9B9CB064958`；`independent-main_test.go.diff` SHA256 `B7E543259D045E6C063C937077DC1CC3C899C26A10C970AA2F57FAD3808C1934`。原作者 diff 仍保留在 inputs；没有用作者结论替代阅读。

### deadline clamp

新 main.go:4185–4226 给每个 maintenanceTransport 独立 deadlineMu 与 cleanupCutoff。`limitCleanupDeadline` 只初始化或收紧 cutoff；SDK 后续 `SetDeadline` / `SetReadDeadline` / `SetWriteDeadline` 请求清空或放宽时均裁至 cutoff，更早值保留。安装和 setter 经同一 mutex 串行，不能在检查后被另一个 SDK setter 越过。尚未进入 cleanup 时仍透传原行为，没有给正常 server 安装新全局 deadline。

新 `connectionCloseContext`（4365–4370）取 parent context 与 min(本次开始+budget，原 D+budget)，不为多层 helper 重置最迟窗口。`limitAttemptDeadline`（4373–4386）先在 registry 锁内复制本 attempt 的 transport，再释放 registry 锁后逐个安装 cutoff；没有持全局锁做网络操作。entry.attempt.closing 在安装前已标记，原 dial 登记路径拒绝此 attempt 的迟到新连接/新 CancelRequest 拨号，未出现关闭开始后新增未裁 transport 的路径。

### 同步 SDK Close 期间仍有中断，且 callback 先 join 再发布 latch

新 `finishConnectionClose`（4342–4362）在同步 `entry.closeSDK(cleanup)` **之前**安装 transport cutoff 和 `watchMaintenanceCleanup`。因此即使停掉原 operation watcher，SDK Close 尚未返回时 cleanup 自身的取消/到期也可关闭登记 raw transport；callback 不并发调用非并发安全的 SDK Close。

`watchMaintenanceCleanup`（4389–4400）以 AfterFunc 的 stop 结果区分“阻止执行”与“已经开始/完成”；后者等待实际 finished channel，sync.Once 使重复 stop 不另等不存在的回调。`finishConnectionClose` 的 defer 顺序先 stop/join interrupt，再 cancel 自有 context；它返回后 `closeEntry`（4318–4339）才保存同一关闭结果并 close(entry.finished)。root/BeforeClose 的 follower 因而不能在 cutoff callback 尚在执行时收到已完成 latch。

健康连接仍唯一触发真实 SDK Close，再等 CleanupDone，最后 Pool.Close；没有提前 raw Close 假装正常 SDK 释放。已有 asyncClosed 分支不重入 SDK Close，只中断登记网络并等原 CleanupDone。shared entry mutex/result/latch 和 transport once 的关闭所有权保留。测试修复没有删弱“raw Close 不等于 SDK Done”的负例，也没有给 F001 barrier 增加超时直接返回/遗弃路径。

## 3. Round 2：实际 SDK 消费与风险回归

直接核本机 **Go1.27.0 / pgx v5.11.0** 的实际调用链，读取范围/文件摘要见 `readset-and-diff-proof.json`：

- pgx `conn.go:302–308` 转 `PgConn.Close`；pgconn `pgconn.go:733–765` 的 Terminate flush→底层 Close→CleanupDone，及 ctxwatch cancel/unwatch 清 deadline。
- Go TLS `conn.go:1421–1455`、`:1470–1483` 的 closeNotify 写 deadline=now+5s；TLS 通过其底层 net.Conn 的 setter，实际落到本次 maintenanceTransport 覆盖方法。原逃逸入口仍存在于标准库，但本次 wrapper 已在正确下层拦截；清零 deadline 也不能解除 cutoff。
- pgconn `:770–813` 的 asyncClose 内部 15s、CancelRequest 与 CleanupDone，及 deadline handler `:3071–3085`；本地 attempt.closing、interrupt 与 clamp 不依赖把 15s 当允许窗口。未虚构 SDK 已支持可取消 CPU 认证。
- pgxpool `pool.go:273–325` 的构造、AfterConnect 和 BeforeClose 析构顺序，`:468–474` 的 Pool.Close。root 与 BeforeClose 仍共享关闭结果；先实际 SDK Done，再让 pool bookkeeping/重复 Close 回收，没有等未来 Pool.Close 才触发 SDK Close 的旧环。

两个回归中的 TLS 测试均实读 fixture/peer：实际 pgxpool.NewWithConfig/Acquire/Release，生成真实 pgx.Conn/PgConn，完整内存 TLS1.2 握手与 pgproto3 Startup/AuthOK/UTF8/ReadyForQuery。背压 peer 在 owner.cleanup 返回前保持开启且不读；正常 peer 读取 Terminate/closeNotify；两者最后实际 join。没有真实 PG、外部 listener、SQL 或 DDL。TLS material 只在 child 内存生成，不输出或落 key 文件。child 执行同一已编译测试 exe；race 命令下也携带该测试 exe 的 race instrumentation，不冒充另一个 production CLI binary 已被验证。

## 4. Reviewer 独立实际命令与原始计数

在实际固定工作区运行，仅执行此次授权的选择器，未跑全 Root/Manifest/F001。过程使用新 Temp 的 GOCACHE/GOTMPDIR，GOENV/GOWORK 关闭、GOTOOLCHAIN=local、GOPROXY/GOSUMDB=off、现有 module cache、`-mod=readonly`；不加载部署环境。完整 argv、最小环境、运行前后 Root4/依赖摘要与 exit 分别保存在 `logs/normal.result.json`、`logs/race.result.json`。

```powershell
go test -mod=readonly ./cmd/han-server -run '^TestPasswordResetMaintenanceRoot$/I-S105-22_tls_close_backpressure_1s$' -count=3 -v -timeout=20s

go test -race -mod=readonly ./cmd/han-server -run '^TestPasswordResetMaintenanceRoot$/I-S105-(15_22_sdk_close_causality|22_(tls_close_backpressure_1s|tls_healthy_close|cleanup_deadline_limits|cleanup_callback_join|late_after_connect|constructor_lookup_barriers))$' -count=3 -v -timeout=40s
```

| Reviewer 自己的运行 | exit | 唯一 leaf | leaf 执行 | FAIL / SKIP / race 告警 |
|---|---|---|---|---|
| 普通 TLS 背压，三轮 | 0 | 1 | 3 | 0 / 0 / 0 |
| 指定因果/网络 race，三轮 | 0 | 25 | 75 | 0 / 0 / 0 |

合计 **78 次 leaf 执行**，两个命令均 stderr 空；不能把父节点或 child 的单个 Root PASS 重复算 leaf。全部 leaf 名字与次数在 `own-test-summary.json`。race 的 25 leaf 含 SDK causality 五项、TLS 两项、deadline property/三 setter×四输入共十三项、callback 三时序、迟到 AfterConnect、Lookup barrier；property 每轮100次、固定 seed=105002，共300次属性样本，未另算为300个 leaf。

Reviewer 的真实 SDK/TLS 观测：

| 场景 | 次数 | elapsed_ms | cleanup_error | deadline_extended / raw_before_sdk | SDK/raw Close | 最终资源 |
|---|---|---|---|---|---|---|
| 普通背压 | 3 | 1000 / 1000 / 1000 | 均 true | 均 false / false | 每次 1 / 1 | SDK Done、Pool.Close 返回、peer join；stage/socket=0 |
| race 背压 | 3 | 1000 / 1000 / 1000 | 均 true | 均 false / false | 每次 1 / 1 | 同上 |
| race 正常健康 TLS | 3 | 0 / 0 / 0（毫秒精度） | 均 false | 均 false / false | 每次 1 / 1 | 同上 |

背压超时仍记录 cleanup_error=true，未被包装成成功关闭；健康连接没有被提前 raw-close 导致假失败。elapsed 的 250ms 测试观察容差仅用于调度/返回开销；setter 有独立不得清空/延长实际 cutoff 的断言，原 +5s 路径不能靠该容差通过。此处不宣称 Go timer 是硬实时系统。

原始 stdout/stderr 摘要：

- `logs/normal.stdout.txt`：`7B5373AD5CFE29187DAAC4C1FFCF161D1E6ECFD179F2B024967EB3ABEB7E1BF6`。
- `logs/race.stdout.txt`：`6DC92CBFD736B0435EE50D6CBC1398D1DB643E6D52902621997E3466A53216DE`。
- 两份空 stderr 均为 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。

作者的 RED 独立保留在 `inputs/author/tls-sdk-red.stdout.txt`（SHA256 `76F1EA2816E8E482604883E7B525713145D0F51A593EB8CD029BD9C1A93C9D09`）：真实 SDK/TLS，1s 预算实际6001ms、deadline_extended=true，exit1，最终 join 完成。它绑定 red main21CEC/testACF1 快照。本 Reviewer 没有覆盖 live 文件来重跑 RED；此次独立运行的是最终固定绿版，上述普通/race结果不是转述作者 GREEN。

## 5. finding 处置与未验边界

| 项目 | 独立结论 |
|---|---|
| 原 F002 | **CLOSED**。不可放宽的 deadline clamp 加同步 SDK Close 期间的 raw-interrupt、实际 callback join 和延后 latch，消除了原已到期后再借 TLS 5s 的具体缺陷；原真实 SDK 背压回归在当前修订独立普通/race均通过。 |
| 本次局部 diff 的新 finding | **0**。未发现为消除 F002 而引入的新关闭权、callback 遗弃、健康误关或 deadline 重置缺陷。 |
| 原 F001 | **OPEN，未在本轮关闭或重新评审。** independently 对比 waitDrained 与 maintenanceRootConstructorDeadline 函数正文未变；作者 f001-still-open 的失败原证据仍保留。本次 TLS 结果不能证明 SCRAM CPU 可取消。 |
| Root-ready / 七文件 CodeGate | **仍 HOLD / 未执行**。不因一项 finding 关闭而给予整体验收。 |

对 F001 只核修复没有悄改原函数/断言以绕开问题：正文的本地规范化摘要及对比在 readset 证据，未与作者采用其他换行算法的摘要强行等同。已通知的“合作式截止＋强制 join＋数值地址”等新方向不等于当前代码修复或独立关闭；在途新 ADR/design/test-plan 未在本次审成固定版本，也未据此重开全 identity 或改写原 r48。

证据层次必须保持：真实 SDK/TLS 的背压和健康用例本轮已由 Reviewer 运行；原 async_closed、raw Close≠SDK Done、共享 entry 竞态、callback join、迟到连接等选中测试含私有 seam/fake，不能一概写成全部真实 SDK asyncClose 故障动态矩阵已经完成。真实 asyncClose 网络链本轮另以实际 SDK 消费源码核对，没有新增真实 PG/生产网络故障注入。未执行全 Root suite、全 Manifest、普通 server 全回归、Linux 编译、组合 CLI-PG、部署或 E2E；作者的 Linux 编译不冒充 Reviewer 自己执行。

已复用同版本完整 Role/Skill/规则全文，并逐一核 hash 不变：add-reviewer、add-code-review、checklist、abstraction、安全红线、core coding/no-hardcode、task-execution、multi-agent、go-service coding/security；详 `start-verification.json`。本轮没有重 Route、另加载 Workflow、读在途设计或再委派。

只写本次新 Temp；未改作者源、产品知识、Tasks/Plan、已有报告或 Git，未清理文件，未访问真实 DSN/凭据/账户/GRANT/秘密文件、PG 或外部网络。两条实际 Go 测试在运行前后均核固定 Root4 与记录的 go.mod/go.sum、domain 两源码文件及旧报告摘要不变；没有宣称把实施中的全仓185源码作为本次 CodeGate 固定对象。

原 cleanup 正式报告 SHA `8C1063321915846CA299E53DBC58CF037507EFE8015819A2E1EF84599AC0359A`、Design R02 SHA `0EC5613B883F4853AF72A6A7324FE9E0E2661450F3305C61DDD2DCF5CF5BDCED` 均保持。结束再次核验及本报告最终摘要另存本 Temp `verification.json`。

**交回状态：stopped。** F002 当前固定修订关闭；F001/Root-ready/后续新合同与整体验证仍交原责任方，Reviewer 不自行启动下一 phase 或修复作者代码。
