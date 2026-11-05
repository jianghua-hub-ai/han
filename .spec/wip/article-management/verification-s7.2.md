# S7.2 主会话验证记录

通知专项已完成本地闭环：**最终 R02 完整普通/race、真实负控、Lead Manifest 七项及独立 R02 均通过，原两项 finding 均关闭，无新增问题**。Lead 已核同修订的源码、运行、范围和资源证据；原任务关闭以 Tasks 为准。本记录不代表部署 E2E 或发布批准。

固定工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。保留既有未提交改动，未暂存、提交、推送、部署或清理。

## 固定成果与责任

- 实际作者和执行者：Test-Engineer Ohm `01a17c83-60f4-7803-b0cc-9b19c0b81fa3`，收到独立两项 finding 后仅在原 SMTP 测试文件整改，最终 R02 已返回 completed 并明确 stopped。
- 唯一新增测试范围为 `test/integration/article/notice_dispatcher_test.go`、`notice_smtp_test.go`、`notice_crash_test.go`，均由 integration build tag 承载；未新增第四个 fixture 文件、依赖或迁移。
- [最终固定十二文件索引](evidence-s7.2/lead-fixed-source-r02.json)绑定三个测试与 S7.1 九个相关实现/测试文件；仅 SMTP 测试相对 R01 变化，最终 SHA256 `9E96006CD8036A00DCB577CCA5975CECB27931224B34AE4C0B3441693201CF48`。完整普通/race 及对应负控的 before/after 哈希一致，Lead 现场复核无漂移；[R01 索引](evidence-s7.2/lead-fixed-source-r01.json)保留历史。
- 设计 r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`；测试计划 r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`。其历史 draft/planned 元数据不改写为人的 accepted。
- 非作者 Reviewer Tesla `01a17c9e-4ff0-7811-a25f-1b4f913f41f7` 的 [R02](review-code-s7.2-r02.md) SHA256 `D4563D87D2C5C62CCEA5DF077DB5DB00DC4E8A923F98692CB33CA077E1EBC680` 确认 F001/F002 均 CLOSED、0 新 finding、CodeReviewGate PASS，已返回并明确 stopped。[R01](review-code-s7.2-r01.md) CHANGES REQUIRED 原件保留，不重开 S7.1。

## 最终 R02 核证

| 执行者/检查 | 最终结果与证据 |
|---|---|
| Ohm：完整普通/race | 各3顶层、112 run/pass节点、99 leaf，0 fail/skip；[普通命令/哈希](evidence-s7.2/r02-ordinary-final.result.json)、[race命令/哈希](evidence-s7.2/r02-race-final.result.json)、[Lead逐项解析](evidence-s7.2/lead-final-json-verification-r02.json) |
| Ohm：F001真实负控 | 临时 SMTP mutant 在缺STARTTLS分支实际发送MAIL，仍返回原安全错误；新增协议断言检出，1 leaf行为失败；[负控](evidence-s7.2/r02-f001-red.result.json) |
| Ohm：F002真实负控 | 临时 Core mutant 在取消后跳过当前结果保存，两取消leaf在真实PG processing行失败；[负控](evidence-s7.2/r02-f002-red.result.json)；固定实现定向10 leaf通过 |
| 独立 Tesla R02 | 逐项核修复diff、真实协议/状态/AuditStore和完整mutant；另执行1顶层、9 run/pass节点、7 leaf，0 fail/skip，新增7个schema保留；[原始独立结果](evidence-s7.2/review-r02/targeted.result.json) |
| Lead：最终 Manifest R03 | 七项按配置顺序全部exit0，gofmt无输出；[索引](evidence-s7.2/lead-manifest-r03-results.json) SHA256 `5F2E463E9D7BF446ECF1484F83CEF3C460B05EFC688EB04C05223B9E57B91464` |
| Lead：最终真实PG回读 | R02两全套196个schema均存在；再次回读两取消case，DATA中为unknown/retry＋风险，受理后为accepted＋唯一受理审计；[查询结果](evidence-s7.2/lead-postgres-readback-r02.json) |

F001 现逐case检查采集到的真实协议：安全失败零 MAIL/RCPT/DATA、零受理；成功安全握手在 envelope 前。F002 现只在真实 Send 成功后进入受理屏障，消费 Reconcile 返回值，再以真实 PG/AuditStore 核当前结果字段、动作唯一性、notice/kind/attempt/trace/actor/time绑定；原停止、一次发送、无后继claim与资源检查全部保留。两种故意 mutant 只通过临时 Go overlay 参与构建，没有改写工作树生产文件。

[R02作者交接](evidence-s7.2/handoff-s72-r02.json)、[更新覆盖映射](evidence-s7.2/coverage-s72-r02.json)与[新增209个保留schema清单](evidence-s7.2/retained-schemas-r02.json)分别可复核；R01 schema 与 Reviewer 各轮新增另行记账，不混称最终运行数。最终新证据46份已核复制哈希，见 [R02保管索引](evidence-s7.2/lead-copied-evidence-r02.json)。独立 R01/R02 的7份/3份原始日志也分别保管，原件均保留。

## R01 固定版本实际执行

| 执行者/检查 | 结果和可复核证据 |
|---|---|
| Ohm：完整普通专项 | exit0，三个顶层、112 run/pass 节点、99 leaf，0 fail/skip；[命令/哈希](evidence-s7.2/ordinary-final-r01.result.json)、[原始 JSON](evidence-s7.2/ordinary-final-r01.jsonl) |
| Ohm：完整 race 专项 | exit0，相同三个顶层、112 run/pass 节点、99 leaf，0 fail/skip；[命令/哈希](evidence-s7.2/race-final-r01.result.json)、[原始 JSON](evidence-s7.2/race-final-r01.jsonl) |
| Ohm：最终缺陷负控 | Go overlay 使用保留的旧 Core R01；5 leaf 行为失败，命中 bigint JSON 绑定和迟到结果缺风险审计；[负控命令/固定源码](evidence-s7.2/core-negative-final-r02.result.json)、[原始失败](evidence-s7.2/core-negative-final-r02.jsonl) |
| Lead：原始 JSON 逐项核对 | [顶层、leaf 清单与计数](evidence-s7.2/lead-final-json-verification.json)；不是只读取退出码，也没有把父节点计数称为独立用例数 |
| Lead：Manifest 七项 | 按配置顺序逐项 exit0，gofmt 无输出；[原始日志索引](evidence-s7.2/lead-manifest-r01-results.json) |
| Lead：真实数据库只读回查 | 最终两组运行共196个不同 schema 全部存在；抽查11个关键场景 recipient 状态、attempt/max、失败 bucket 与审计动作计数符合预期；[查询结果](evidence-s7.2/lead-postgres-readback.json) |
| Lead：范围比对 | 起始688文件无缺失；变化仅已登记的七个既有代码路径和四份业务文档，新代码仅 S7.1 两文件及本卡三文件；[比对证据](evidence-s7.2/lead-worktree-comparison.json) |

完整定向命令分别为 `go test -tags=integration ./test/integration/article -run '^TestNotice(DispatcherPostgres|SMTP|Crash)' -count=1 -timeout=12m -json` 及增加 `-race` 的同范围命令。三个真实顶层为 `TestNoticeDispatcherPostgres`、`TestNoticeSMTP`、`TestNoticeCrash`。没有运行会自动 DROP schema 的旧 identity 全包集成命令。

Manifest 顺序为 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。这些默认包检查不编译 integration tag，不能替代上述专项；Go cache 和既有平台限制仍按各命令实际范围理解。

## 数据、协议与进程证据

运行目标为既有 loopback `127.0.0.1:5433` PostgreSQL 18.6。每个 fixture 使用随机、预先提交的 schema、完整0001～0009迁移和合成数据；User-scope DSN 只注入进程，不进入输出或子进程命令行。所有运行 schema 保留，连接、事务、goroutine、子进程和 SMTP server 正常关闭/join。作者 R01 共保留331个 schema，包含分组、负控和历史失败；R01两组196个 schema 的 [case 对应清单](evidence-s7.2/lead-final-schema-map.json) 与 [作者R01保留清单](evidence-s7.2/retained-schemas-all.json)分别记账。R02最终196个schema见[新case清单](evidence-s7.2/lead-final-schema-map-r02.json)。

SMTP server 只监听 `127.0.0.1:0`，使用合成 CA 验证证书链和主机名，覆盖 implicit TLS 与 required STARTTLS。安全负例包含无升级、错链、过期、错主机及拒绝跳过验证；阶段分类覆盖 MAIL、RCPT、DATA command/final、Auth/greeting、受理后 QUIT 失败和最终 ACK 丢失。连接不转发，不向真实邮箱发信。

专项核双 recipient 独立、创建快照、Gate caller transaction/最新登记/在途关闭、SKIP LOCKED/多 worker、耗尽维护、真实锁等待后的 DB 时间、微秒 fence/ABA/迟到独立风险审计、结果与审计原子性、commit rollback/ACK unknown/回读失败和 bigint JSON 往返。发送屏障期间核连接池已无占用，并从同池执行查询；退避用真实 DB due time 与后续实际发送核对。[覆盖映射](evidence-s7.2/coverage-s72.json)列出实际 leaf 及明确复用的 S7.1 单元/组件边界，独立充分性仍由 Reviewer 判断。

真实子进程在 AfterClaim、DuringDATA、AcceptedBeforeSave、AfterResultCommit 四处退出，每处分别 max=1/3。父进程核阶段、实际非零退出、输出及 socket handler 结束，再核状态/审计/协议计数。耗尽的未知发送终结并记风险，不再次发送；有余次在 lease 到期后恢复；结果已提交的终态不重发。受理后崩溃的重复窗口被真实观察，没有宣称 exactly-once。

## 历史失败与证据保管

初始 `late-first` 失败来自测试夹具在同一 Reconcile 后续合法 claim 中重复关闭通道。作者将需要观察一次旧结果的场景限定为一条 claim，并补异步返回/资源回收；最终全部通过。该历史日志原样保留，不视为当前版本失败。早期分组与负控没有完整保存当时三个测试文件快照，不补造旧源码；权威最终普通/race/负控均绑定十二文件哈希，另保留固定三个测试的 R01 全文快照。

最终负控通过 Go overlay 替换临时构建输入，工作树内九文件始终未改；负控 exit1 是预期检错结果，不能误报为当前版本失败。原始日志、命令、三文件快照、映射、schema 清单和 Lead 核证均复制到本目录。[R01副本索引](evidence-s7.2/lead-copied-evidence-r01.json)列初次复制的来源、字节数与 SHA256，当时53份逐字节一致；扫描未发现真实 User-scope DSN 全文或私钥，临时原件继续保留。

Lead 随后发现自己复制的未格式化 AST 调试工具 `inspect.go` 使中间 Manifest R02 的首项格式检查失败。已将原始字节另存 `inspect.go.raw-r01.txt`，仅格式化本目录的工具副本，并记录[原始/格式化哈希修正链](evidence-s7.2/lead-evidence-format-correction.json)；这一个 R01 catalog 条目的原始字节现由该 txt 保存，旧 catalog 不覆盖。没有删除、移动文件或改变十二源码。失败日志保留，最终 Manifest R03 全部通过；后续新增 Go 证据以 txt 承载。

## 未验证边界与交回

本卡证明本地 dispatcher/PG/SMTP/子进程组合；r18 的生产 signal coordinator、root factory、三 worker/five-result 和反向退出组件证据仍引用 [S7.1](verification-s7.1.md)。没有在支持的生产 Host 启动完整 HTTP 服务、外部 SMTP 或真实处理登记，没有执行 Unix FileJournal/联合恢复、容量留存或部署后 E2E，也没有人的整体验收或发布许可。

独立 R01 另实际执行3top/14run-pass节点/9leaf，0fail/skip，新增9个schema全部保留；并对 during_data 与 after_accepted 回读确认当前实现的数据/审计实际正确。两项 finding 针对未来回归保护缺口，不是已观察到生产错误。主会话另提的 Snapshot 疑点由 Reviewer 核当前 seed 的 deleted/retained 更新后排除，不作 finding 或返工。

最终 R02 的作者修复、自测/真实负控、原 Reviewer 复审和 Lead 核证已齐备，所有作者与 Reviewer 停写，原 S7.2 已满足自身关闭条件。九个实现相关文件仍保护；其余任务保持原状态，部署 E2E 与需求整体验收不因本卡闭环而完成。
