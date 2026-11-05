# S10.5 Root cleanup 存活性独立有界 CODE 审查

Reviewer：Dewey / add-reviewer；真实 Agent ID `01a180fe-c3bb-7661-86fb-06af4f76ba1b`。Lead 委派的非作者 Reviewer，本次未再委派。日期 2026-10-28。

**结论：关注成立；严格 cutoff / 无遗弃承诺在本快照不能成立。两项 HIGH，均 OPEN。应保持 Root ready HOLD。** 本意见只回答 Root cleanup 同一因果链的独立问题，不是七文件 CodeGate，不给 CODE Review PASS。没有证明所有生产路径都会永远阻塞；已有支持路径超过原 cutoff 的具体证据，已足以否定“总能在 cutoff 前收束”。

## 固定对象和来源

工作区 `D:/AID/VSCode/Codex/Han`，实际分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。对象为 `.spec/wip/identity-auth/evidence-s10.5/cleanup-review-input-r01/inputs.json` 及其 fixed 原字节快照。四项均已独立验 hash 并复制到本 Temp 的 `fixed/`，没有拿并发修改的 live Root 作为固定对象。

| objectType=code 的固定路径 | SHA256 |
|---|---|
| `cmd/han-server/main.go` | `21CEC445C412EE909062DCF9E81C8D4D6ACD0CD26842E7AEE0E3533EF29BA543` |
| `cmd/han-server/main_test.go` | `97F71B19DD5D5EB3F02370E04A7C3A8AD5C0E64BDF861A7D86F421B5A4DDB5C6` |
| `cmd/han-server/maintenance_stdin_windows.go` | `620583ABAEB1551761359F93EAA7A4D0B79BE8A4E312B2FFD9C1D69A82663A43` |
| `cmd/han-server/maintenance_stdin_other.go` | `A711C46F5D0AD017CA54569BC8862F30EFCDF2989EA13B0C856A47F615D854D9` |

实际语义审查限 main.go 的 owner/config/hooks/调用与关闭链及 main_test.go 的相关测试；两个 stdin adapter 本次只核固定对象身份，不声称重新完成输入/平台完整 CODE 审查。Domain 两文件与 PG carrier 没有在本次作 CODE Gate。

上游为持久 `design-r02/fixed-final/` 的 design r48，SHA256 `69309241DFAEB448383EE2C63CA494B4ABAD8CE0AC5DE29A1555F7D54A5CCEB5`，特别是 2381–2405 行；ADR-0026 r03，SHA256 `BCB5707C90E008D4200087750361A6A6C648BCF0822691A3BD8BDBF8EB0F6442`，特别是七 ENV / operationBudget / ShutdownBudget。前者 2403 行明确：所有阶段共同 cutoff、不能借 SDK 2m/15s、不能超时遗弃，无法满足即停止受影响实施。这里不延长窗口、不删弱断言，也不改历史 Design R02 报告。

## 两轮实际检查

Round 1：固定代码 `main.go:3900`–`:3905`、`:4057`–`:4425`、`:4479`–`:4519`、`:4556`–`:4567`，及原固定 `main_test.go:280`–`:454`。核原 D→cleanup cutoff、owner quiesce、登记、stage、trace、constructor barrier、SDK close winner、CleanupDone 与 Pool.Close 的顺序和超时分支。按原 Reviewer Role、完整 add-code-review、checklist、abstraction、安全红线、core coding/no-hardcode、go-service coding/security、task-execution 与 multi-agent 约束执行；没有重新 Route、另开 Workflow 或分派。

Round 2：只沿真实直接依赖补证。本机 pgx **v5.11.0** 的 conn.connect/Close、pgconn 构造/认证/context watcher/Close/asyncClose、pgxpool 构造/析构，puddle **v2.2.2** 的异步构造/统计/Close；本机 Go **1.27.0** TLS Close、PBKDF2 原语及 Windows 默认 resolver。补读现役 `database_config.go` 仅确认未禁止 TLS/SCRAM/hostname 等正常支持路径，没有读取 DSN/环境值。源码与具体范围/摘要见本目录 readset 证据。

另按 Lead 新授权读取作者 SCRAM 反例完整 stdout 和正在完善的相关测试函数，捕获为独立补充版本，未替换原四文件快照：

- `author-root-cutoff-scram-fixed.stdout.txt`：原文件来自 Root Wegener Temp，SHA256 **`AC3601C8C9BEA506AC5CD0C01BFA70359F84F9637350C053798502FE37B0EF44`**。
- `supplemental-main_test.go`：本次一次性 raw capture，SHA256 **`D0452CF4F4BCDFDB13B5043B0C45B3D5AD11C63713E86602857F3F36ED129902`**；相关函数 666–790 行。它在途、不是固定七文件最终版本。stdout 行号是此前时点；没有把当前 capture SHA 冒充该日志运行二进制的精确构建摘要。

## [HIGH] S105-CODE-CLEANUP-R01-F001：SCRAM 构造中的 CPU 阶段不能由取消/断网中断，barrier 越过 cutoff

- 对象/修订：固定 `main.go` SHA21CEC（完整摘要见上表）；责任为 **Root Wegener** 交还 **Lead / 原 FD Ohm** 核最小可实施方向；若涉及 SDK/架构边界再由原 Arch 接力。Reviewer 不修作者代码或选新方案。
- 最小位置：`main.go:4383`–`:4398` 的 waitDrained，`:4402`–`:4406` 的调用；实际 ctx/tracer 接缝 `:4100`–`:4130`、`:4209`–`:4226`。
- 违反依据：r48 `:2381`、`:2395`、`:2403`，ADR 显式 ShutdownBudget；取消后仍须在共同 cutoff 内收束，不得继续等待无界阶段，也不得遗弃。

**现状。** waitDrained 在 ctx.Err 后只记 deadlineErr、interrupt(nil)、等 1ms 再循环；唯一退出谓词仍是 Constructing/Acquired/stages 全零。没有 cutoff 后的有界退出/完成机制。main.go 注释“Constructors already hold the original canceled context and registered transports”只说明拥有 ctx 和 transport，不证明构造当前执行的步骤会响应它。

**具体受支持触发。** 在原 D 前收到合法 SCRAM server-first-message，`i=10000000`（SDK 接受的包含端点上限），随后在 PBKDF2 CPU 派生期间到 D。pgconn `pgconn.go:487` 调用不带 ctx 的 `scramAuth`；`auth_scram.go:91`–`:101` 在解析 server-first 后同步调用 `clientFinalMessage`，`:290`–`:292` 允许这一迭代数，`:322` 同步执行 `pbkdf2.Key`。这段函数不接收 ctx、不查 ctx、不读写 socket。原 D 取消、stage cancel、SetDeadline、raw Close 均不能抢停计算。TraceConnectEnd 只有 connect 返回后才发生，puddle Constructing 也只有 constructor 返回后才移除。

**作者真实 SDK / 协议 peer 证据已逐段核。** 补充测试用真实 pgxpool.NewWithConfig/pool.Acquire，owner.configure 安装真实 owner；Lookup 返回合成地址，Dial 只返回 net.Pipe。pgproto3 peer 走 Startup→AuthenticationSASL→提取客户端 nonce→合法 server-first（公共合成 salt、i=10000000），不执行 SQL、没有 PG server/listener。operation=150ms、ShutdownBudget=1s，异步观察器在 D+1s 记录构造/stage/socket，再实际 owner.cleanup join。SCRAM 分支没有等待 lookupRelease 或手动延迟 constructor；另一个 lookup 分支是人为 release gate，未把它当生产默认 resolver 的实证。

原始三次失败均为 acquire_canceled=true、cleanup_deadline=true、cutoff 时 Constructing=1/stages=1/sockets=0，观察时间分别晚于 cutoff **481.6µs / 164.5µs / 289.8µs**；最后 join 分别晚 **286.5636ms / 423.071ms / 391.9141ms**，最终 Constructing=0。与 CPU 计算未完成而非 socket 未关一致。测试对“在 cutoff 时仍活着”报 FAIL 是正确反例，不是测试必须迎合实现改绿。

**独立 Reviewer 原语实验。** 本 Temp `scram_cpu_probe.go` 只调用标准库相同 PBKDF2/SHA256/10M 参数及公共合成输入；1s context 已到期，CPU 调用仍在 **1641ms** 返回。结果见 `scram-probe-run.json`。这是 Reviewer 独立的 CPU 原语验证；未运行 Han/pgx/PG，不冒称独立重跑了作者 SDK 用例。

**影响。** 在当前支持范围与合法 1s shutdown 配置下，已证明原 cutoff 可被突破。后续 timer 轮询或再 interrupt 不能补偿已在 CPU 段丢失的取消能力；若下层永不完成，代码本身也没有终止路径，但本次不声称已复现生产永久死锁。正常健康构造可以收束，并不能推出该受支持失败/边界路径的预算承诺成立。

**最小处置与复审条件。** 保持 Root ready HOLD，原作者/FD 先给出真实支持的可取消认证或明确可审的支持边界，不能只把 waitDrained 改成 ctx 到期直接 return（遗弃），不能用 goroutine 包 PBKDF2/constructor 后超时不 join，也不能增大 ShutdownBudget 或借 SDK 2m。需要冻结处理方案、精确修订及此真实 SDK peer 反例的当前证据；还要保留 positive 收束、构造/SDK 不并发 Close、实际 join 与共同 cutoff。没有可用机制就按 r48 停止受影响实施、交上游，而不是登记“已有 ctx 所以有界”。

## [HIGH] S105-CODE-CLEANUP-R01-F002：同步 SDK Close 内的 TLS deadline 重置可另增 5s

- 对象/修订：同一固定 main.go；真正修复责任 **Root Wegener**。
- 位置：`main.go:4282`–`:4310`，尤其 `:4302` 同步 closeSDK(ctx)；`:4402`–`:4406` 在建立 cleanup ctx 前停止 operation watcher，`:4342`–`:4360` 的 quiesce(false) 保留已 handed transport；maintenanceTransport `:4185`–`:4200` 未限制 SDK 改写 deadline。
- 违反依据：r48 `:2399`–`:2403` 的剩余清理时限、同一 cutoff 与正常/失败都须实际完成。

**调用链与触发。** 对已交接、无活跃业务调用的 TLS 连接，cleanup 正确先触发 SDK Close；但该调用同步执行，执行期间 closeEntry 下方的 ctx.Done 分支尚不可达。pgx conn.go:302–308 转 PgConn.Close；pgconn.go:751–765 先 Watch(ctx)、发送 Terminate/flush，然后在 watcher Unwatch 前调用底层 net.Conn.Close；TLS 连接进入标准库 tls.Conn.Close。

当 peer/transport 不再接受写入，Terminate flush 因 1s cleanup ctx 到期被一次 deadline callback 中断；接着 TLS Close 执行 closeNotify。Go `crypto/tls/conn.go:1470`–`:1480` **重新 SetWriteDeadline(now+5s)** 再发 alert；这一 alert 使用 writeRecordLocked，未因之前应用 Write 的 timeout 缓存错误而跳过网络写。一次 ctx callback 已执行，不会因 TLS 重设 deadline 再触发。当前 root 停掉 operation watcher且没有同一 cleanup cutoff 的 raw-interrupt watcher，handed transport 未在此前关闭；它直到同步 SDK Close 返回之后才有机会再 interrupt。

**Reviewer 独立实测。** `tls_close_probe.go` 在本 Temp 使用纯标准库、完整内存 TLS1.2 握手后的 net.Pipe；peer 保持开启不读。TLS 材料只在内存生成，无外部网络/PG/密钥文件。按 SDK 对此风险关键的顺序执行 watcher→Terminate 类写入超时→TLS Close→Unwatch，记录：

| 指标 | 实际值 |
|---|---|
| cleanup budget | 1000ms |
| flush | `write pipe: i/o timeout` |
| TLS Close 将写 deadline 移过 cutoff | true |
| TLS Close 额外耗时 | 5000ms |
| 总耗时 | 6001ms |
| 探针进程退出 | 0（确认反例的探针成功，不是产品测试 PASS） |

原始输出为 `tls-probe-run.json`。这是同一标准库 primitive 与实际 SDK 顺序的独立反例；不是实际 PG 或 Han TLS 端到端运行。真实 TCP 上需要相应写背压/故障，未声称任意正常 TLS Close 都等 5s；源码及探针证明“只传 ctx 足以给 Close 强制 cutoff”的假设不成立。

**影响。** 即使 constructor/barrier 已正确归零，cleanup 仍可能同步越过截止，未到 CleanupDone select 或 Pool.Close，合法 1s shutdown 不能被保证。`<-finished`（4287）、超时后 `<-cleanupDone`（4309）、stopWatch 等待和最后 Pool.Close 也均没有独立时限；它们必须依赖先证明下层完成，不能反过来当证明。本 finding 不要求通过提前返回丢弃这些等待来消除问题。

**原 owner 可评估的局部方向。** 在已批准 transport owner 接缝内，让同一 cleanup cutoff 的中断在同步 SDK Close 期间仍有效、只关闭已登记 raw transport，并 join callback；或在适当阶段对底层 SetDeadline/SetWriteDeadline（含清空/放宽）强制原 cutoff，使 TLS/SDK 内部窗口不能覆盖它。需由 Root 作者验证具体实现与原“健康先 SDK Close”顺序、一次关闭/共享结果、异步关闭/迟到登记、普通 server 隔离；Reviewer 不替作者选方案或写补丁。这些方向仅处理网络/TLS 缺口，**不能解决 F001 的纯 CPU 构造阶段**。

复审需固定 Root 修订、1s TLS 背压 Close 的实际测试、raw Close→SDK Done 的区分、共享 latch 竞争与所有 callback/timer join，不能只保留 fake closeSDK 立即 close(done) 的正例。

## 可用 API 与上游所需的精确决定

| 方向 | 当前真实能力 / 不能据此声称的内容 |
|---|---|
| ConnectTimeout / 原 D / Lookup-Dial wrapper / ConnectTracer | 真实接口且 ctx 注入成立；中断网络等待，不能抢停同步 PBKDF2。单靠缩短 timeout 不修 F001。 |
| pgconn.Config / watcher / RequireAuth | 已核 Config 完整字段，**没有可配置 SCRAM 迭代上限或 context-aware SCRAM KDF callback**。MaxProtocolMessageBodyLen 限报文大小，不限 i=10M CPU；BuildContextWatcher 操作 transport，不让 PBKDF2 观察 ctx；AfterConnect/ValidateConnect 在认证后，来不及。RequireAuth 只允许/拒绝认证方法，不提供 SCRAM 内部取消。不能发明 MaxScramIterations/Authenticate(ctx) API。 |
| SDK 支持可取消 SCRAM | 在 CPU 派生中按 ctx 分段检查才是相应能力；当前私有 clientFinalMessage/scramAuth 不暴露替换点。需有真实 SDK 修订/API/依赖允许范围和源码/测试证据，不能默认已有，也不能擅改 module cache、fork/vendoring 或扩七文件/依赖边界。 |
| 受信 server 限制 | 可交 FD/部署方评估“实际挑战迭代数/认证方法/全部主备端点及部署性能边界”的显式支持合同。仅说受信 server、通常 4096、增加机器容量或一次 benchmark 不形成严格剩余时间证明；没有客户端约束及剩余预算控制，晚到的合法挑战仍会进入不可取消阶段。不能默默关闭 SCRAM、切换 provider/密码方式、扩大 budget，或把默认值当正式支持边界。 |
| 进程隔离/强制终止 | 属新生命周期/架构、资源所有权及 unknown 处理合同；需要 Lead/FD/必要 Arch 的具体选择和新评审。不是现有 root owner 的局部已授权补丁；没有在本次采用。 |
| 只改 waitDrained/select return 或 detached close goroutine | 会把越时等待换成遗弃活跃 constructor/SDK，直接违反冻结合同，不是修复。原 cutoff 不改、无遗弃不改且没有可取消原语时，应停止实施并交回上游。 |

另外，Windows 默认 `net.DefaultResolver` 是一个须明确的**证明边界**，不在本次增加第三条 finding：pgconn config.go:496/1090–1091 沿用该 resolver；本机 net/conf.go 的 Windows 默认选择 native 路径，lookup_windows.go:161–182 在 ctx 取消时返回，却明确留下尚未完成的 GetAddrInfoW goroutine 写回 buffered channel。Root defer finish 删除 Lookup stage 不证明底层 OS resolver 工作已结束。这不能用来证明 waitDrained 必然永等（恰恰 wrapper 可能已经返回），但若“无遗弃/所有 lookup 资源收束”包含此底层工作，现有登记不足以证成。须由原 FD 明确此资源范围以及真实可取消/可 join resolver 或受控数值地址支持边界；不能 Reviewer 默许隐藏遗留工作或把人为 lookupRelease 测试当真实 Windows 证明。

当前最小上游交接问题是：**在七路径/现有 pgx 依赖及正常 TLS/SCRAM/hostname 支持下，如何给出构造 CPU、OS resolver 与 SDK Close 的可取消且可 join 合同，满足同一 D+ShutdownBudget？若做不到，哪一精确接口/支持边界/依赖或隔离方案由有权方采用并重新冻结？** 这里只请处理该新证据影响，不重开 authority、业务单 enqueue 或全 identity 设计。历史 Design R02 保持原内容和 hash；新实证如要求改变已批准合同，由原 owner 形成有界增量再评。

## 验证、限制和停止状态

本 Reviewer 实际执行的是两项**明确获准的个人 Temp / 离线 / 标准库探针**：`go build` 指定各单一 .go 文件到本 Temp 后运行 exe；GOWORK/GOENV 关闭、GOPROXY/GOSUMDB=off、独立 GOCACHE/GOTMPDIR/GOPATH，未用项目 Go 构建或测试、未下载模块。合成 TLS 私钥只在内存生成且未输出；PBKDF2 使用公共合成字符串，不访问账户数据。构建与运行 argv/结果保存在 `probe-build.json`、`tls-probe-run.json`、`scram-build.json`、`scram-probe-run.json`。

作者三次 SDK 反例是**作者实际运行的 pgx/puddle + 协议 peer**，本 Reviewer 阅读原始证据/测试与源码并以独立原语实验交叉验证，未冒充自己运行 SDK 用例。未运行 live Han Go tests、PG、真实 DSN、凭据、账户/GRANT、部署或 E2E；未改代码/产品文档/任务/计划/任何已有 Review，未做 Git mutation、清理或委派。

此意见不判 live Root 后续修复状态，也不承诺最终七文件的其他正确性、编译、测试或 CodeGate。应交原 Root owner 最小修复与原 FD/Lead 处理 F001 的上游可实施性缺口，冻结新修订后由原 Reviewer 复核。**本次返回后 stopped；Root ready 仍需由 Lead 保持 HOLD，未授最终 CODE PASS。**
