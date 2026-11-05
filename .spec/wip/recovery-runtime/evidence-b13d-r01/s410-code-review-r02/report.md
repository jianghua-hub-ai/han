# S4.10 independent CodeReview R02 — CodeGate PASS / STOP

非作者 Confucius / add-reviewer，actualID `01a1890f-a330-73d1-86fb-236777c54751`。本报告关闭原 `S410-ICR-R01-F001` HIGH 和 `S410-ICR-R01-F002` MEDIUM；当前授权组件范围内未发现新增真实缺陷。两轮独立审查、当前完整 U/P 普通/race、原10探针及两项合法正控、独立物理 SQL 与全部进程收据均已完成。Go/IdentityPG 已实际 STOP/release，pending0。报告封装不占执行窗口。

这不是作者自检、Main 检查转签、Task/G4 关闭、业务接纳或项目/E2E 全部通过。

## 精确对象与授权

固定 `D:/AID/VSCode/Codex/Han`、main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。原三源最终对象：

| 文件 | SHA256 | 当前行数 |
|---|---|---:|
| internal/identity/recovery_execution.go | 6D80E8D1565298F03734FAAD3C8F41F3395C84167A737C82A79436FBB0B2288E | 2852 |
| internal/identity/recovery_execution_test.go | 34EB7168E2DEEF05DEAA9C9D0CED72FF2CC99C5C4BF3FDCD5B5D1B29F6A980AA | 1083 |
| internal/identity/recovery_execution_postgres_test.go | 659E59AFD8500E5AC0EEC7C92982866735B5BE8E6F20165547EDCB0EC1049C08 | 1969 |

Main 的 `s410-independent-review-local-pg-start-r02.json` 与 `s410-repair-main-freeze-r02.json` 均按实际文件 SHA 绑定；binding 在执行中及结束时未改变。24 source/docs 每个实际命令前后均核对，其他21对象保持冻结；go.mod/go.sum 也与独立执行前相同。未修改项目源、文档、元数据、Task/Plan/设计、Git 或共享 dirty 内容。

资源补充允许 Main 独立缓存的非 PG 检查与本角色 GOMAXPROCS1 串行作业并行；本轮执行期间 IdentityPG 只由本角色使用。本角色完成并释放后，Main 后续 targeted/其他 DB/identityPG2 另有授权，不属于本报告的独立执行。

## 两轮实际审查

Round 1：阅读全文补丁及所有新增/改变的当前代码与测试，核对原三文件 +358/-70、无第四源文件。原 R01 全文5616行审查只复用于逐块字节完全相等的内容；当前5904行的变更范围全部直接核读，零未覆盖行。`current-byte-coverage.json` 保存旧/新行范围、相等块 SHA 与直接读取范围；这不声称本轮重新逐行阅读了全部5904行。当前测试保留原产品断言；新增 U13/P7，P fixture 的历史全前缀扫描改为立即登记并只回读本 fixture 实际创建的 schema，保持30秒预算、23表/run/Audit 的真实 SQL 断言与数据保留。

Round 2：重新沿关键原写入者/消费者核对 `PasswordResetStore.RecordRejectedDelivery`、authority/delivery/credential lineage、实际 worker settlement/终态重试/Audit、`passwordResetIdempotencyTerminal` 和真实 replay classifier；核对现役 pointer/restricted/reviewProof/Complete、r56 原 cohort/六字段/原 actor/time/真实 resolution Audit、事务外源读取和 image 再验证、现役 Session exit/独立 ACK 接口。R01 依赖 readset 的其余31个文件均与当前 SHA 相等，复用其实际覆盖的 S4.9 Store/readAdmissionTransaction、0001–0010 索引/FK/不可变约束、S9.1 canonical A/B/phone 限制、真实 ArticleAccountPort/OrphanArticlesCanonical、旧幂等 success/failure/unknown 契约。未重做其他 Reviewer 的 Article/S414 完整评审。

确认 User→Store→review/domain/Audit 锁序、Complete 不新增 User 锁、同 Store/实际 RawPermit、源 I/O 在 SQL 锁外且取消真 join、未知投递保护窗口、不可变原 cohort/source、零终态重写和错误根因/unknown ACK 收敛保持。没有用标签、数量、fixture approved bool 或作者绿测代替整组契约证明。代码位置与原消费者的读取范围分别见 `current-consumer-readset.json`、`R01-readset-byte-reuse.json` 及附带原 readset。

## 原 findings 复审

### S410-ICR-R01-F001 — HIGH CLOSED

原位置 `recovery_execution.go:1201`；当前核心为 `:1201` 的 `recoveryFailedResetSettlement`、`:1241` 的 pointer、`:1280` 精确 claim 与 `:1328` 原域信封验证，以及 `:1491` 的 Complete reviewProof 消费。

原错误是仅凭 failed/invalidated/job succeeded 标签允许 retained restoration pointer 通过 Complete，未验证具体失败信封或实际 settlement Audit。当前读取同 operation/user、run/install/原 actor binding、credential lineage、原六字段 claim 和真正 terminal classifier。通用 failed snapshot 不能替代 password-reset invalidated 结果；worker invalidation 必须存在唯一匹配 target/system/outcome/settled time/原 attempt 的真实 recovered Audit。后来的 terminal job completion 只完成原 job，允许原 Audit attempt 小于当前累计 attempt，不杜撰第二次 settlement Audit。

同步 rejected-delivery 按 `password_reset.go:667` 的原 RecordRejectedDelivery 处理：该方法确实写 failed/rejected/invalidated operation、撤销 credential 并完成 job，且不写 worker Audit、不完成泛化失败 claim。其后续 r56 解析必须有真实原 cohort/source/resolution Audit；Complete 在事务外独立重读源，再在锁内验证同 image/原成员。未发明一个“直接失败 claim”生产者、Retry 或审批 fallback。

原两条负例及合法 failed-pointer 正控逐字节重用，在普通与 race 均 PASS。缺失证明现在在受限 review 阶段以明确 admission binding/incomplete 错误拒绝，物理 SQL 证实 run 仍 running，原 generic claim 或 Audit 缺失仍保留。合法 retained failed pointer 可完成身份复核；新增独立合法终态重试正控也通过，claim/user/Audit 不变。完整 P 两模式还实际执行真实 RecordRejectedDelivery 的正例和缺失 source 负例，以及原 worker/domain/cohort/ACK 矩阵。原缺陷的触发链已由实际拒绝和合法路径共同关闭。

### S410-ICR-R01-F002 — MEDIUM CLOSED

原位置 `recovery_execution.go:1557`；当前核心 `:1644` typed receipt 和 `:1665` exitLocked，集合核对 `:1704`、实际 system Audit `:1713`、无 Audit 部分证据拒绝 `:1723`。

当前要求 IDs/count 字段存在、非 NULL、类型正确、非负且一致；UUID 唯一。收据集合必须精确等于同 run/install 实际因 recovery_maintenance_exit 撤销的会话集合，并有对应完整 system Audit。普通会话和其他原因已撤销的历史会话不加入本次结果。存在 exit-reason 撤销行却缺原子 Audit 时保持 incomplete，首次/重试都不补写 Audit。

原 missing/null/invented UUID 三条负例在普通/race 全部拒绝；原 first-zero/retry 正控仍通过。完整 P 的 duplicate IDs、wrong reason、NoAudit partial、prior-history zero/nonzero、expired/old-runtime、ordinary unchanged、rollback 与独立物理 ACK 均实际通过。新增独立非零正控把历史 revoked 行、过期/当前 runtime 会话和普通会话放在同一 fixture，经真实提交后丢 ACK、独立物理 reader，再进行只读重试；两模式通过。最终独立 SQL 直接核对合法零/非零收据 IDs/count 与实际 reason 对应 Session 集合，确认非零结果恰为两条、本来已撤销历史和普通会话保留。

本 finding 原先证明的是不完整收据被接受；未夸大为已复现活跃会话泄漏。当前修复关闭了该具体契约问题。

## 独立实际执行

实际目录 `D:/Temp/han-s410-independent-r02-288lth47/packet/raw/`；准备目录不是本轮 raw 根。共9个实际命令：preflight、installed go-version、U/P/independent 各普通/race六个命令，以及 final physical-schemas。所有 test 命令为 local/offline、GOMAXPROCS1、-p1/-parallel1/-count1；仅 integration P/probe 使用命名 identity_dsn，其他 DSN 为空。无 ambient/business DSN、网络安装或服务操作。

| 六组 | U 普通/race | P 普通/race |
|---|---:|---:|
| FinalizeOnceAndAtomicAudit | 2 / 2 | 6 / 6 |
| UserStoreReviewLockOrder | 14 / 14 | 3 / 3 |
| KeptDeletedCanonicalAndDenied | 4 / 4 | 9 / 9 |
| IdempotencyUnknownAndCompleteProof | 40 / 40 | 47 / 47 |
| SucceededRecoverySessionExitAtomic | 15 / 15 | 9 / 9 |
| RecoveryExitAuditAckReadback | 2 / 2 | 12 / 12 |
| 合计 | 77 / 77 | 86 / 86 |

原64U/79P 名称由保留的原作者 raw 重新解析，与本角色当前实际叶子集合比较，均为完整子集；当前普通/race 名称一致。原10独立探针加两项合法正控为12/12，各模式均全 PASS。总计350次叶子执行，零 fail/skip/setup/compile/SQL/timeout/race diagnostic。未因绿测无理由重复测试，也未用此前作者执行签本角色 PASS。

R01 的124个 payload 和9+1已执行探针保持不可变；本轮独立追加原10的11742字节 LF 内容完全相同，当前原测试前缀逐字节保留，production overlay0/test overlay2。准备阶段两项新增正控原先误把只读重试解释成零 Commit；源码检查发现通用 writeVerified 仍提交读取事务，因此执行前另存 inspected 版本，检查不改终态数据/Audit及每次方法调用的事务计数。原准备文本与差异均保留；这不是执行后改断言迎合结果，原10断言从未改动。

本角色源代码/方法读取时的控制台编码、路径 glob 及 Python 覆盖计算包含空白插入行的校正属于静态工具诊断，未伪造为产品 Red；未导致 Go/PG 重跑。作者保留的 SQL42883、SQL55000 和两个父级历史扫描超时也继续作为作者诊断/失败命令，绝不冒充原行为 Red 或 Green。原 R01 行为失败链保留。

## 物理证据、进程与资料接收

实际 preflight 和最终 psql 对象均为 PG180006，127.0.0.1:51934，han_b13d_identity/han_b13d，data_directory `D:/Temp/han-b13d-main-pg-r02/data`。只读取资源 JSON 的 identity_dsn 字段，未打印/复制 DSN。完整 P/probe 共196个本轮实际 CREATE 的新 schema；每个 fixture 的物理回读集合等于实际创建集合。最终本角色独立 psql 再回读相同196个精确名字，每个23表及真实 run/Audit/claim/review/Session 后像，未扫描共享历史前缀。所有 schema/数据/raw 保留，无 DROP、清理或移动。

9个父命令全部实际 Wait；kernel Job 最后 ActiveProcesses0；stdout/stderr 两个 pump 均 EOF 后 join；失败收敛路径仍由同 owned runner 负责。总 kernel accounted processes175；可选观察24个 PID 值只是已观察子集，不宣称逐 child 创建时遍历/Wait，也不宣称整 Host 已空。本角色待运行/待 join 作业0，Go/IdentityPG 窗口已释放，cluster 生命周期由 Main 管理。详细 argv、raw 哈希、lifecycle、每命令源3前后快照与24对象 guard 均在 packet 内；cache/work/binary 在 packet 外，复用的是同一 reviewer 的 R01 暖缓存。

作者649个 manifest payload 已按实际 SHA 静态核对，无不匹配；其41命令、517 schema 和两轮作者自检仍属于作者/Main证据，不冒充本角色执行。Main project7+owned17fmt 的新收据另存：6项通过、旧23个 root format 失败按原哈希保留、owned17空输出通过；不能写成全部 project7 PASS。该既有处置来自 Main 的 standing user authority，本报告不扩大豁免，也不代 Main G4/Task 关闭。

## 未验边界与交回

真实 provider/ACL/审批主体、SMTP、原生 restore、HTTP/auth/root 集成、生产恢复与真实 E2E、人的业务接纳仍未由本轮证明。软件 fixture 的合法正控不成为生产 fallback。未来 S4.16 typed factory/forwarder 消费未实现，未假装真实 root 已注册；它也不被倒挂为当前组件正向关闭前置。S002 未建立实际获授权新 producer 的触发链，保持未验限制，未捏造测试或新 finding。

当前组件 code-review 结论：F001 CLOSED、F002 CLOSED，CodeGate PASS，无新增真实 finding。Main 接收本 packet 后负责自身当前 targeted/集成检查与适用 Task/G4 决定。STOP；后续仅供读取，不再自动运行 Go/PG、修代码或清理证据。
