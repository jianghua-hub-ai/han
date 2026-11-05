# S2.4 pg_database_size 只读诊断

14:31开始，最长约12分钟；本轮为独立只读诊断，不恢复原性能窗口。固定Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca和五freeze hash。只写本目录；不Go/DDL/DML/配置/清理/委派。

已完整消费Backend Role、add-systematic-debugging Step1–4、observability；读取指定作者最终索引/positive-stop、measurement/leak/positive-semantics raw、独立证据复审阻塞段、Lead最终接收和相关harness/helper。原症状是H:1346 pg_database_size在Acquire0情况下SQL5000/5003ms超时；measurement混合失败，leak/正常语义未到达，不据exit1判断业务通过。

假设H1：大量保留relation文件导致全库目录/逐文件metadata成本接近5s。预期：catalog毫秒级、size秒级；独立psql也慢、无blocking PID；逐文件stat耗时与目录文件数/SQL耗时相符。

假设H2：SQL在等数据库锁或主机存储/CPU竞争。预期：activity出现blocking PID或可归因wait event，或主机指标显示显著竞争；若无证据只标unknown，不能把NULL wait直接等同CPU根因。

假设H3：harness连接池排队/客户端上下文是当前主因。预期：独立客户端不慢，而harnessAcquire等待明显；已有Acquire0使该假设优先级较低，仍用同SQL独立计时区分。

上游精确PG18.6源码：[dbsize.c](https://raw.githubusercontent.com/postgres/postgres/REL_18_6/src/backend/utils/adt/dbsize.c) db_dir_size:68–104、calculate_database_size:110–153。它对数据库目录内每一项stat并累加长度，不读取全部表数据页；只作实验设计依据，环境具体原因待实测。

## 已完成观测与根因隔离

实验已于约14:37完成，此后没有新增PG实验。14:43诊断到界，停止研究/采样；只完成本报告。原正式5s门、源码、配置、旧基线/物证均未修改；没有执行Go或DDL/DML。

身份：127.0.0.1:5433、server_version_num=180006、han_perf_s24、OID6746690，default_transaction_read_only=on。PGDATA为C:/Program Files/PostgreSQL/18/data，目标目录base/6746690。catalog观测11067 relations、86 schemas，其中82个han_pgperf_r04_*；仅pg_default和pg_global，无自定义tablespace。catalog对照查询42.284ms，初始active_others=0。

| 同一SQL重复 | 只读诊断deadline | server SQL实际ms | psql总体wall s | 出口 |
|---|---:|---:|---:|---:|
| 1 | 20000ms | 6955.451 | 7.359 | 0 |
| 2 | 20000ms | 3905.985 | 4.204 | 0 |
| 3 | 20000ms | 3990.506 | 4.453 | 0 |
| 4 | 原5000ms | 4567.028 | 5.016 | 0 |
| 5 | 原5000ms | 3825.867 | 4.265 | 0 |

原SQL均为SELECT pg_database_size(current_database())，返回1992988351 bytes完全相同。三次较长deadline只是取得诊断实际耗时，未修改正式门。第一条SQL确实超过5s；两条原5s查询这次均成功，因此复现的是查询本身秒级并可越过5s，未伪称本次两条5s均复现cancel。psql启动/监测开销与server SQL时间分列，不能把第4条总体5.016s误说SQL超过5s。

五次查询共41条实际activity行：blockers全部为空，wait_event_type/wait_event全部NULL；没有观测到持续锁等待。NULL wait不等于正在消耗CPU，也不能排除未采到的短瞬态。无连接池的独立psql仍出现6.955s，结合旧Acquire0证据，当前超时不需要连接池排队即可发生；旧r01的probe争用可能叠加成本，但不能解释本次独立SQL的耗时。

| 同一目标目录，11137项 | 实际wall | 合计bytes |
|---|---:|---:|
| 只枚举名称 | 31ms | 不计算 |
| 枚举缓存entry.stat | 47ms | 1992988351 |
| 每项显式路径os.stat | 5891ms | 1992988351 |

显式路径stat另外对1000项耗437ms、5000项1813ms；成本随文件数量扩大。缓存枚举与逐路径stat在同一只读状态下得到精确相同字节数，且与PG一致，不是估计值或仅当前schema大小。PG18.6 db_dir_size逐路径stat实现与该受控差异对应，根因已隔离到**全库大量文件的逐路径元数据调用成本，与H固定5s采集预算失配**，不是需要扫描约2GB表数据页才能得大小。

补充第6条同SQL用于同时采主机指标，server4486.981ms/exit0。其查询backend41336在三个WMI样本中IOReadOperations/Bytes均为0，IOOtherOperations/sec约22582–30997，进程CPU计数27–43%；主机总CPU12–26%，物理盘队列0、盘忙0–7%，没有持续吞吐饱和证据。这支持元数据系统调用为主要可观测工作，不能据此指定CPU瓶颈、驱动、杀毒/过滤软件或冷缓存。第1次6.955s没有同步WMI覆盖；初始host-samples.jsonl是查询后的闲时样本，不用于证明查询期间负载。直接进程CPU/IO API访问被拒绝(error5)，其缺数保持unknown，未提权。

结论：H1在上述机制/当前状态范围内得到支持；H2的持续PG锁等待与盘吞吐饱和未获支持，瞬态和底层每次stat变慢的具体组件仍unknown；H3不是当前SQL变慢的必要原因。不能把本结论泛化为未来平台或所有数据规模的性能保证。

## 精确消费者与最小候选比较

当前H为test/integration/article/pg_perf_harness_test.go：1328创建固定5s context；1346执行全库size SQL；1354在超时后报错；1435初始化baseline也使用同SQL；1567起monitor以5s ticker驱动检查。正式freeze未改变。

**优先交Main评估：仅Windows、本机已校验PG目标的目录枚举缓存采集器。** 本机Go1.27.0 C:/Program Files/Go/src/os/dir_windows.go:191/193从枚举结果构造fileStat，:226 DirEntry.Info直接返回缓存fs。可按PG18.6 db_dir_size真实定义精确累加当前DB目录各项长度，保留原5s、fail-closed、旧全DB/tier baseline和累计caps；不使用relpages/行数等近似值、不只计新schema、不清理、不silent SQL fallback。建议受影响源码最小限H内资源采集及其初始化/monitor消费者，W/R/N/docs不因诊断顺手改动；本轮没有实现或执行Go实验。

采用前的最小证明/边界必须包括：

1. 校验endpoint/version/database/OID/PGDATA真实路径及默认tablespace；对当前未验证的自定义tablespace、symlink/reparse、路径逃逸或权限不足直接失败。PG定义排除pg_global，并按各tablespace中当前DB目录求和；不能悄悄漏项或改变为递归求和。
2. 以int64检查每项大小与累加溢出；明确ENOENT/目录变化的处理，匹配PG语义，不能把其他IO错误当0。当前只读等值不足以证明打开文件持续增长时Windows枚举长度及时性；需在另授权的新唯一schema内验证写入增长、物理文件/fork增量、跨进程旧baseline和超限取消，保留与有界PG oracle对照的真实证据。
3. context在目录打开、批量枚举和每项处理前后检查，句柄/worker有明确所有权；底层IO不响应取消时需有界失败/owned child fail-stop，不能超时返回后留后台goroutine继续扫描。约47ms是本机当前测量，不是5s硬实时收口证明。
4. 先证明完整DB字节等价和失败闭合，再补未到达局部语义/泄漏负例，交原Reviewer；不把采集器快当性能验收PASS。原完整性能窗口已经NRU，后续完整性能需Main另行明确执行安排，旧失败/累计不洗。

备选：继续SQL并另采用较长资源探针预算，改动行数少，但会改变正式5s/保护新鲜度与写入准入合同，不能单纯调大timeout吞失败；若采用必须单独决策、验证有界monitor和资源增长风险。当前不实施。仅改重试/忽略失败/只读stats估计/复用无期限旧缓存均不能保持准确全DB累计与fail-closed，不建议采用。

## 证据、限制与STOP

原始证据集中本目录：identity/catalog、sql-size-1..5.json（SQL、原始psql输出、每次observer）、filesystem.json、metadata-scaling.json、isolated-size-host.json、host-during-query.jsonl和analysis.json。隔离文件started_at字段误存身份对象，原物证不覆盖；实际时点以observer/WMI时间及psql Time为准。没有重建旧source快照。

第一组final.json约14:36、隔离组isolation-final.json约14:37的最后探针均为backend[]。所有owned psql client均有实际wait出口，host sampler实际wait0，脚本本身均exit0；此后没有PG操作。14:41本机元数据核对中同PID的conhost是当前审计shell子进程重用，审计PowerShell匹配自身命令亦非仍运行的sampler，未杀任何未知进程。源码实验前后五hash逐份一致，完整值在final.json/isolation-final.json/analysis.json；当前Lead亦核268源无漂移。

**SOURCE保持只读；LOCAL/PG只读诊断STOP并释放。** 14:43有界上限到，不再发实验或研究方案。根因与候选交Main，由Main串行采用实施/验证范围；作者不自行补丁、不代独立核销，S2.4仍active，原完整性能NOT RUN不变。
