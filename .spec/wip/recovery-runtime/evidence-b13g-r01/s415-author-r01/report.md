# B13G / identity-auth-S4.15 作者交付

Sartre / `01a18b91-4da7-72c3-92c7-0e2b5dbe0823` 已交付有界实现与自测。**SOURCE、soleLOCAL、两处 Main 缓存租约均已 RELEASED，pending=0**；正式回执为 [release.json](release.json)。后续仅 D-only 封包，无源码、Go 或缓存访问。

固定 Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。原 10 路径中实际新增 2、修改 7，`lifecycle_test.go` 字节不变；其余三个旧测试文件只追加测试，四个旧测试文件原文全部保留。八个既有路径原本均 untracked，不把它们冒称本作者新增；共享树其他内容保留。没有暂存、提交、push、切换、清理、删除、移动或归档。

新增同一 Runtime 的 ordinary lifecycle 适配：显式 startup PhasePermit 只用于内部同步调谐；长期循环使用独立受控上下文。三 worker 的 started 通道在实际 run loop 入口关闭，lifecycle 同时核 Done/Err。新 scan/claim/dispatch 取得 WorkLease，并保持到 Converge、settlement/Fail、fenced 保存/回读真实返回。共享关闭上下文首次绑定后不延长，notice 先 BeginShutdown，再取消工作；Stop/Join owner 只执行一次，Wait 后仍等真实 Done，root-owned join 接缝完成后才关 Article 资源、最后关 pool，保留现役错误优先级。旧业务 lease、重试、SQL 常量与通知策略未改。

原六项验收全部 exit0，且对应 job 的 source10 都与最终 SHA 相同：

| 验收 | 合规 job |
|---|---|
| top 符号列表 | acceptance-list |
| 原五组普通测试 | acceptance-top-normal |
| 原五组 race | acceptance-top-race-serial |
| cmd vet | acceptance-vet-cmd-serial |
| article/identity/cmd 全包普通 | packages-normal-01 |
| article/identity/cmd 全包 race | packages-race-01 |

五组各自真实命中，无 Skip；普通/race top 各 10 个 Test pass 事件（含父/子节点），不是独立业务 Case 数。三包每模式 3603 个 Test pass 事件；原 Windows `symlink_rejected`、`world_writable_rejected` 两个 Skip 保留，没有算通过。模块内额外断言覆盖 deletion scan/Converge、reset Claim→Renew→settle/Fail 与原 ConnectionPermit 计数、notice 在实际 fenced 保存退出前保持工作 lease。

补充检查：两个领域包 vet、`go mod verify`、限定 D 输出路径的 cmd build、原 10 文件 `gofmt -l` 均 exit0，格式输出为空。新函数没有超过 50 行；既有长函数按最小改动边界保留，没有借此重构旧算法。protected36 每次 Go job 前后均精确匹配。Git staged 为空；scoped `git diff --check` 为0，但既有八源是 untracked，因此真实 diff 以本包 original-before→final-after 为准。

证据保留三类失败，不混算通过：

- `red-started` 是最初可编译、真实启动 Article worker 后缺实际 started 握手的行为 FAIL；Main 已现场核收。早期 job 有源码 SHA 与 raw，没有完整源码副本的历史不再重建。
- `green-top-01` 是编译诊断；02/03 是 fixture TraceID 格式诊断。`green-race-01` 是等待失败并发生 Go test 超时，未观察到 DATA RACE；后续四个 diagnostic 保留逐步收窄证据。产品根因是普通 acquisition 前用于识别 startup 的 ReadPermitBinding 探测：取消与 ownership 检查竞争会升级全局故障。改为明确的私有 startup scope＋实际 CheckPhase 后，原拒绝/等待断言普通及 race 通过。fixture 失败路径另补必经 lease.Release，避免 cleanup Stop 等自己；未删除原等待断言。
- `acceptance-top-race` 与 `acceptance-vet-cmd` 曾因作者提前启动下一作业产生约1.5秒生命周期重叠。两者 raw/exit0/join 全保留，**排除出合规验收**；运行器增加 OS byte-range 排他锁，受影响两项以上表 serial 作业补齐。不是产品 Red，也未隐藏为顺序执行。

全部 26 个 Go/gofmt 作业均记录 actual process Wait、kernel Job active=0、两条管道 EOF 和 pump join，失败作业同样终结。后续 job 有持久化启动前 argv/cwd/source-before/owner PID 与 live heartbeat，实际 helper SHA 为 `6FD58BE2B28FDDDB47EC3F2F534EAFDAF7EE2CB5C4EC54772D18C357EE1085BD`。原始 helper 不改；作者 runner 的变更与最终副本随包保留。

**边界与接收**：本次仅 U/C/race 软件证据。Store/SQL/DDL/Runtime 实现、main.go、config、Task/Plan/design/TP 均未写入；PG/SQL/服务及真实 provider/SMTP/native/root/deploy/E2E 未开放。HTTP/维护/Tx/child 的 root-owned join 由组件回调接缝验证；完整真实 production root 接线与真实环境验证仍属原 S4.16/后继专项，不在此宣称完成。nil AcquireWork 仅保留旧消费者兼容，后继 root 须显式注入本适配；actor 恢复 reset driver 未被 ordinary worker 替代。

最终完整 10 源码快照与全部1432行 diff 见 `review-input-packages-race-01/`，本作者已全文核 diff；最终 SHA 见 release。Main 已告知 Ptolemy 的独立静态 Review 正在进行，本作者不代其结论。交回当前活动待 Main 共同版检查、正式接收与非作者 Review；**无作者 CodeGate、whole-batch G4、人的接纳或 Taskclose 声明**。
