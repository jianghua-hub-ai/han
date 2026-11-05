# S7.1 主会话验证记录

Lead 已接收本任务的通知消费、SMTP 兼容扩展及生产接线交付。最终九文件独立 Joint R02 通过，原三项代码 finding 均关闭；原任务状态以 Tasks 为准。真实 PG/TLS/crash 专项另由 S7.2 承接。

固定工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本次为未提交改动，所有代码作者及 Reviewer 已停写；未暂存、提交、推送、部署或清理。

## 固定输入与成果

- 设计 r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`；测试计划 r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`。设计仍为 draft，不补人的 accepted/checkpoint。
- [最终九文件与哈希](evidence-s7.1/lead-fixed-source-after-joint-r02.json)：Article 三文件、SMTP 两文件、main/lifecycle 四文件；没有 schema、HTTP 或公共 Mailer API 变更。
- [Joint R02](review-code-s7.1-joint-r02.md) SHA256 `0C160DF795F72B9CB1DE6C84D09317D0E71BF83C80C9E5C02924B67D513269EC`：9fileJointGate/Root4CodeGate PASS，Joint F001 CLOSED，0 新 finding；Core F001/F002 保持 CLOSED。
- [Core R02](review-code-s7.1-core-r02.md) SHA256 `08187C0005EB9BE074DCA7E0E07EC478D18FE184877E8588CA59BA42412F2C21`；[SMTP 局部 R01](review-code-s7.1-smtp-r01.md) SHA256 `4762EFC4ED1F312835B70A8D4B8C3BE2BB4C73982DB069755D414FEC1BD7D018`。局部结论没有代替最终 joint。

## 实际验证与归属

| 执行与证据 | 最终结果 |
|---|---|
| Franklin：SMTP 普通/race | 各17顶层、72通过节点，0 fail/skip |
| Franklin：身份邮件兼容普通/race | 各4顶层、11通过节点，0 fail/skip |
| Nash：Core R02 通知普通/race | 各39顶层、117通过节点，0 fail/skip |
| Nash：Core R02 Article 回归 | 612通过节点、0 fail、2个既有平台 skip |
| Lorentz：Root R02 定向普通/race | 各31顶层、108通过节点，0 fail/skip |
| Lorentz：Root R02 有限 cmd 回归 | 117顶层、765通过节点，0 fail/skip；明确排除真实连接失败子进程及 helper |
| 独立 Reviewer | 两轮核源码/消费者及原始日志；Core R02另跑6顶层/27节点，Joint R02另跑2顶层/2节点，均通过 |
| Lead：最终 Manifest R02 | 七项严格按配置顺序执行，全部 exit0；gofmt 无输出 |

Lead 实际逐行解析作者原始 JSON，核顶层/叶子、fail/skip、包终态及源码哈希；读取全部最终 Review 和实际差异，不只转述 Agent 摘要。Core 的原 Repository 21131-byte 前缀逐字节一致；最终对起始688文件核对，只有已登记7个既有代码路径和4份业务文档变化，无缺失基线文件；另有登记的两个新通知源码/测试文件及本范围证据。

[Manifest R02 索引](evidence-s7.1/manifest-s71-r02-results.json)记录真实命令、exit、耗时与日志哈希；顺序为 `gofmt -l .`、`go mod verify`、`go vet ./...`、`go build ./...`、`go test -tags=contract ./...`、`go test -race ./...`、`go test ./...`。完整 Manifest 包含 Root 有限回归排除项的默认执行范围，二者分别记账；部分未变包使用 Go cache。它不编译 integration tag，也不代表逐叶零 skip。

## 失败与修复链

- [Core R01](review-code-s7.1-core-r01.md)的 MEDIUM F001：旧 owner 迟到结果缺独立风险审计。原作者补真实 issued/attempted 句柄、释放结果事务后有界独立审计及低敏回退，不改新 owner 状态或再次 Send。
- 同报告 MEDIUM F002：AuditStore JSON 数字往返使合法 bigint 绑定误判 unknown。通知私有审计计数改规范十进制字符串，保留业务 int64、schema 和严格绑定；新增真实 JSON 往返、bigint 上界及错绑定用例。
- [Joint R01](review-code-s7.1-joint-r01.md)的 MEDIUM F001：生产 notice logger 缺 service/module。原作者从同一 logger/sink 派生 notice logger，绑定 service/module/event，并核真实 factory 注入及同 JSON sink；原 Reviewer 复审关闭。
- `root-r02-final-regression.json` 是中间旧 logger 指针断言失败；权威最终日志为 `root-r02-fixed-*.json`。Lead 曾按文件名误选中间日志，核实际 catalog 后更正，失败原件保留，未覆盖或冒充最终通过。
- 首次 Core stub 的源码快照未及时保存，只保留真实 RED 日志及当时测试源码；不补造完整历史。Core R02 和 Root 两轮的 RED 源码快照完整，均为可编译行为失败。

[R01 保管索引](evidence-s7.1/lead-copied-evidence-r01.json)与[R02 保管索引](evidence-s7.1/lead-copied-evidence-r02.json)列原始日志/RED快照/diff及哈希。副本与临时来源逐字节一致；复制前扫描未发现真实 User-scope DSN 全文或私钥，原件保留。索引中的临时路径是运行来源，仓内副本用于后续复核。

## 未验证边界

本任务尚未运行通知真实 PostgreSQL、多连接锁/池资源、loopback TLS 多会话或真实子进程 crash；这些由原 S7.2 验证。Root 四时序的组件 probe、真实 factory 与 Core 局部测试分别证明其边界，不声称已在真实 PG/SMTP 组合执行。Windows FileJournal unsupported、支持 Host 全量启动、旧备份/Journal 恢复、实际 SMTP/处理登记、容量/留存与部署 E2E 保留原独立 Gate。

本记录证明 S7.1 本地开发/自测/独立代码评审及 Lead 核证闭环，不代替需求整体验收或发布许可。
