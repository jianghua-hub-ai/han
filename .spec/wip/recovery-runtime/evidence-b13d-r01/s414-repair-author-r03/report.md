# S414 R02-F001 作者修复交付（STOP，待原 Ptolemy 独立 R03）

固定 Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca。已全文读取本轮 r03 授权、R02 report/findings/receipt/panic fragment/REUSE；17 项本人原 G0 方法逐项核同版复用，未重新 Route、委派或申请 routine 批准。

仅改 recovery_restore.go、recovery_restore_test.go；其余原三文件完全不变。获取文件后立刻在 openImage 建立局部 defer：未返回 owner 时由获取方 unwind；正常返回（含 owner+error）仅转交现有调用方一次。approve/acquire panic 不 recover、不改其 identity；acquire 未返回的私有资源不能被调用方假认领。已返回 lease 的 closeAndJoin 外层 defer 关闭本组件 checked FD；正常清理错误通过 errors.Join 保留，清理自身 panic 原值传播。生产改动只有两个方法，没有新设施、public Runtime/API/权限、fallback、SDK、签名协议、第六源或 buildtag 文件。

## 实际 Red / Green

| 检查 | 普通 | race |
|---|---:|---:|
| 修复前 exact Reviewer 新 panic3 | 1 PASS / 2 FAIL | 1 PASS / 2 FAIL |
| 修复后原92 + exact新3 + 作者7 | 102 PASS | 102 PASS |
| 原 RestorePG7 | 7 PASS | 7 PASS |

Red 是真实可编译行为失败，生产前像及两个失败 raw 完整保留；Reviewer 原始 R02 失败证据仍在 canonical 目录。作者 own Red 另为 3PASS/4FAIL，暴露 approve/acquire 与 terminal-cleanup-panic 的 checked FD 泄漏；探针只在记录失败之后回收测得的测试泄漏，不冒产品 cleanup。Green exact Reviewer 三例均 descriptor_closed_by_component=true、manual_probe_close=false。原 R01 healthy 仍实际启动 restore+verify 两 child；image/lease 仍 child0，不改变原独立 CLOSED 状态，也不由作者关闭新 finding。

七个新增生产回归叶：ApprovePanic（真实 Runtime preBegin）、AcquirePanicRun（真实 Begin1 后、新 child0）、HandoffPanic、ReturnedOwnerError（错误 cause 双保留）、ReturnedOwnerErrorClosePanic、SuccessfulOwnerClosePanic、SuccessfulTransfer。返回 owner 测试在真实 lease join 内确认 borrowed FD 可读、close 恰一次、随后 checked FD 实际 os.ErrClosed；Stop/driver.Close 不重复清理。原阻塞 image owner/child-pipe/targettx join 叶完整保留并实跑。清理 panic fixture 明确在原 software lease 已实际 terminal 后才 panic；不把任意 provider 提前 panic 当资源已经终止。

原92逐叶与旧交付清单机械核对 missing=[]；原80/author9/Reviewer3 均保留。DBDEDF 与24D710完整 fragment 精确复用，overlay 仅映射本轮完整最新 test prefix，生产 overlay0，断言未改。新测试无新增跨 package import，仍使用既有 recovery/platform；PG 原用 identity/migration/testenv，不改依赖。

## 进程、PG 与检查

11 个 Go parent 实际 close/Wait，stdout/stderr 文件全部结束；进程表见 process-ledger.json。64 个不同受控 child（按 command+PID 去重，94 条观察）全部实际 Wait/PipesJoined；不是全 Host 枚举。没有仍运行的自有 Go、child、callback、pipe、image owner 或测试 PG 连接。Main PG 保持运行。

只在两个 exact PG top 命令向私有 env 注入当前 restore_dsn；仅127.0.0.1:51934/han_b13d_restore，固定0001–0010、新随机 quoted schema，本轮新增14 schema 全保留，原有 schema 未触碰。保留名和实际 creation/targettx/pool join 日志见 retained-schemas.json；未额外执行 grant 外 SQL。普通、vet、list 的 DSN 均空；offline、GOMAXPROCS2、-p1、count1JSON。两种 vet、两个 list 均 exit0，五文件 gofmt -l 空，Git no-index --check 无空白诊断（差异 exit1）。所有22 raw 文件扫描实际私有DSN及 canary 值均0；不存 child 私有 stdin/stderr。少量 runner 预启动错误及无进程影响在 execution-notes.json，未冒测试结果。

24 个保护源码/模块路径（B13C19、Runtime admission、shared root2、modules2）前后完全一致；自身五文件新SHA见 delivery.json，完整两文件 diff 见 repair.patch。Git 只读，源码五文件仍为原已有未跟踪状态，无暂存/提交/push/切换。其他作者内容未覆盖、回滚或格式化。Banach 活跃 identity design r56 未读/hash；Tasks/Plan 当前来源只按 Main 已知新6609246F/C76D2995记录，未冒旧全文件hash仍有效；两TP/Article design 未做本轮hash断言。无SSOT/设计/规则/模块写入。

Native 生产设施未固定，真实 enable Gate 继续 CLOSED；本轮软件 fixture 不冒 native ACL/lock/no-follow跨进程传递、真实receipt/process-tree、实际restore、fullS413、rootS416长期idle owner或E2E。原319/Linux/global7按本轮 grant 留给最终Main，未机械重跑。原 public RestoreDriver、attempted/unknown与部署独占边界保持。

STOP：源码冻结，soleGo/RestorePG修复窗口释放，交原 Ptolemy 独立复审。作者不关闭 Finding/Task/G4，不宣称 native/root/E2E 或全7通过。
