# R09 flight/growth窄delta独立CodeReview

**Source Gate：PASS，仅限固定R08→R09有限活性优化及growth诊断delta；无新增confirmed finding。** 等待者在自己的原截止前复用等待期间发布的完整成功，A的START仍决定freshness；后来的新请求仍重采。真实cap判据已接到资源terminal，不再把deadline/freshness或cleanup前t.Failed当cap。代码与离线回归成立，**R09真实growth/并发共享/cap-hit仍待实际取证**。

## 对象、采用依据及权限

原非作者Reviewer，objectType=code，固定Han/main/指定HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；Role/add-code-review/适用规则复用。唯一写集为本code-review-flight-r09目录。未Go/LOCAL/PG/SQL、未执行作者runner/binary、未读凭据、未源/Tasks/旧制品改写/Git/清理/委派/采用或执行决定。

正式对象 `author/probe-flight-r09/source-freeze-r09/manifest.json`，captured2026-11-04T10:49:08.308613Z、source/local STOP、R09 PG未授权/未使用。H **241407B / C3DFDC66FDBFB025760ED78B0E0061CC49BD7AFB2DA0BF76685D94D0D89B8B99**；基线R08 H **226915B / AC2AD1AC172C2E78D898DAAC624CEC35965719023B78E24D87101144E2B23878**。五snapshot、五before及旧四resource摘要自行复算匹配，另三Go/docs不变；未用live源代替快照。Lead现场范围/live/项目检查另行完成。

完整读Lead `growth-r01-failure-and-r09-grant.md`：首growth实际增长 **36904960B**，但未取得cap-hit，后续未启动；入队deadline下保守失败属于原合同允许行为，不能倒称旧R08违约。此次是Lead已明确采用的有限活性delta，本Reviewer不代作采用决定。扫描内外差额根因仍unknown，不归因Windows/锁/IO或据此宣布准确性已闭环。

## 两轮实读与核证

Round1完整读作者r08-to-r09.patch、独立diff、新状态字段/共享分支/诊断消费者、typed cap与FaultSeedDisk记录及回归。离线核11个hunk并应用于R08前像，重建结果与固定R09 LF文本完全一致。

Round2只沿 `collect → checkDiskBudgetContext/collectDiskBudgetContext → readDiskObservation/validateResourceObservation → monitor/init/seed → terminal/实际cap谓词/材料输出` 检查生产者与消费者；反查R08关闭/只读入口、owner/clock/ledger及预算不变。newHarness、seedCtx和beforeSeed真实接线已读，不只看测试helper名称。不扩全库或原18项。

核正式manifest7条最终同版命令及red-flight-r01，逐项result/raw SHA、actual wait、前后五hash/计数；另核三条相关中间失败和freeze元数据错误，保留其历史。新nonrace/race binary均独立hash，core构建→运行路径/CWD核同。详见verify-offline.py、evidence-audit.json、intermediate-failure-audit.json。仅Python离线核证，不称Reviewer实际运行Go。

## 共享活性实现与边界

| 接缝 | 固定源码/证据判断 |
|---|---|
| B的原deadline | H1819～1832在请求进入时记录generation并固定min(request+15s,旧freshness,owner/parent)；排队后只允许进一步收紧，不往后移。队列仍单槽、等待计入原请求机会。 |
| 只复用新完整成功 | H1859～1883先核terminal/ctx/owner与returned<end，要求successful已改变且source Finished<end，才复制s.last。只有原完整合法成功在1934～1936递增successful并发布last；没有把部分read/错误当新成功。 |
| freshness不续命 | Shared保留A Started/ReadStarted/Finished/bytes/free，只单列B Requested/Returned/Deadline/WaitNS；不回写s.last、不递增successful。freshnessLocked仍是last.Started+20s；B不能以自己的等待或返回时间刷新资源年龄。 |
| 新请求仍重采 | A发布结束后才进入的请求捕获当前generation，没有等待期间的新发布，因此走真实reader，不任意复用旧cache。FlightShares回归明确验证下一次reads变2、新FlightID及新bytes。 |
| 取消、终态与错误 | B原截止/ctx/owner先到拒绝共享；late仍不发布。队列取消返回实际terminal，避免已有SQL错误被ctx覆盖。原noncontext/mixed错误的Closing升级和join保留；同版expired/canceled/SQL/mixed及旧R05/R06回归到达。此优化不保证所有调度下必然成功，原冻结deadline仍可能保守拒绝。 |
| 可辨共享回执 | H1950真实checkDiskBudgetContext调用collectDiskBudgetContext，再调用同一个collect/实际reader。H1960 exclusive落盘，文件名含原子回执序号；reused=true、相同flight_id、source_requested/source_sql_ns保留A来源，B SQL/Acquire/Volume置0、等待另计。回执数不等于SQL数，UTC字符串相同也不能抹掉不同请求身份。 |

ConsumerSharedEvidence使用真实consumer落两份JSON再读取，核一份shared与原SQL来源/等待身份；最终同hash普通/race实际通过。其stdout未直接给每个consumer fixture路径作版本索引，因此本Reviewer不从任意中间fixture目录拼成最终版物理回执证据；代码接线+已执行测试可用于Source Gate，真实PG回执关联仍属于下一实测。

## growth诊断真实接线

H2041～2055保留原准确bytes/卷余量与tier/global阈值判断，仅将命中返回改为局部 `pgPerfResourceCapError`，携带Observed/TierBase/TierCap/GlobalBase/TotalCap，Error文本保留原数值格式。资源collect/stopLocked继续产生真实terminal及failedOwn，没有重置累计账或引入物理预分配。

`pgPerfSeedGuardReached`要求root取消、failedOwn及实际typed cap的数值确已达到tier或total阈值；deadline/freshness/普通取消均不能通过，造一个未到阈值的typed error也不能通过。回归还实际走validator→stopLocked/owner取消→目标谓词，不只是测试一个未连接的bool函数。

固定H5892～5957的FaultSeedDisk在seed前安装defer，真实读取h.resource.terminal/failedOwn/last，写target_reached/termination_kind/实际cap观察；不读cleanup前t.Failed。目标到达也明确保留预期负例exit1，其他原因写NOT REACHED，不能只看进程exit1接受。

新增loop上限 **60s**仅此故障诊断；实际从h.ctx派生，newHarness中h.ctx此时是原small **seed120s**的seedCtx，继承更早root/global取消。它不是再赠60s或扩正式phase/point。每个追加批先resource.admit再Exec，成功命令tag的RowsAffected及批起止/序号累计，阶段/seed/loop边界接到同一defer；即使seed内Fatal也能留当时terminal。没有sync/checkpoint/强关连接催cap。

记录的rows仅是**成功返回tag的追加批**，不含前置seed全部行，也不能代替取消时未确认写结果的判断；若seed先失败，loop边界可能零值。外层实际接证仍须结合seed/迁移/批次与静止SQL解释差额。target_database_bytes是当前small更严注入阈值，旧global/tier base未改；2MiB不是瞬时零突破保证。

## 同版回归、历史失败及版本边界

| 最终同五hash命令 | actual wait / wall | 独立核到的计数 |
|---|---|---|
| green-offline-r04 | 0 / 13.125s | 父测试/子测试59pass，0fail/skip；原始61条PASS行另含2条child回显。 |
| race-offline-r05 | 0 / 50.360s | count3合计父记录177pass，0fail/skip；183条PASS行含6条child回显，不重复计SQL/测试。 |
| build-nonrace-r09 / build-race-r09 | 0 / 2.015s；0 / 4.625s | 两个当前binary分开固定。 |
| core-r09 | 0 / 10.782s | 当前race binary/R09 sandbox，JSON 57个有Test的pass事件，0fail/skip；预期child FAIL日志不冒父测试失败。 |
| vet-r09 / format-r09 | 0 / 1.469s；0 / 0.078s | vet/格式diff空。 |

red-flight-r01为中间F441…ACA4，exit1/5.234s：真实读到reads=2、A成功/B原截止失败；deadline/freshness被旧故障谓词误当cap的回归也真实失败。它是带新回归的中间版，不是精确R08全文的PG违约证明。

green-flight-r01曾因两快回执同名只剩一份失败；r02修原子文件身份后，测试又把同一Windows墙钟字符串当成不同请求的必要条件而失败；最终显式受控时钟/flight身份回归通过，不放宽deadline。green-flight-r03过宽选中4个只读PG子项，均在absolute evidence root前置处0s失败，冻结入口在此之后才构建pool/读DSN；它不是PG执行通过或行为RED。freeze首轮为函数提取越过新增type的元数据断言错误，非Go行为失败。原失败保留，不跨hash改绿。

实际文本逐函数核同：readDiskObservation/准确SQL、pool config、budget/policy parser、只读入口、safe错误分类、admit/freshness/monitor/expiry、newHarness、R08 waitFinalClose/closeResources/Close/probe、phase/point和baseline加载。当前pgx/puddle依赖文件摘要与既有独评pin一致。collect主体确实改变，不能说R09整体采集实现“完全未变”；不变的是单次非共享reader/driver接口及旧期限/终态/关闭接缝，新增共享路径已单独受审。

nonrace主binary：**20050944B / 71B3EF47D466A8EAB899C266311EAD224E5F5316D6B244A8E2D7EFDF705521D5**。race：**32902075B / 39840B6D3E9FC1B1C0D6C270AD54C5016B877A1A9C02CFB2F412C47733F44CCE**。两个实物均自行复算，构建源与core路径/先后吻合；不混用binary或称交接hash为每条命令执行瞬间额外签名。

## 最小下一实测门

**本轮不要求机械重跑四模式或额外成功smoke。** R08实际准确读取/15s取消/旧START+20s/正常Close证据可限定沿用为**历史driver与接口佐证**：相关实际reader、依赖、单调用期限和关闭代码未变，本R09离线回归又覆盖旧安全语义。它不是R09 PG实测PASS，也不能移植旧SQL耗时或容量结论。

Source Gate PASS后仍由Lead另发**同一attempt**的新growth许可；本报告不授权执行。最小新增真实证据可在现有FaultSeedDisk复合阶段取得：

1. 固定R09实际binary/五hash/policy/origin/当前静止账，准确写前baseline、实际seed/追加及批/行/阶段时序。
2. 真实monitor/同步guard并发出现完整成功A及shared B：同flight_id、reused/source来源、A真实START、B自身Requested/Returned/原deadline、SQL只发生一次且B在其截止前接受；不把共享回执数当独立采集数。后来真实新请求重采的代码/离线证据已核，不强制另造PG入口。
3. 准确SQL确实达到实际更严cap，typed terminal与target_reached/observed值一致，root取消/后点停止。deadline/freshness/SQL错误抢先发生必须保留失败/未到目标，不能凭exit1、guard_failed或已有+36.9MB改判PASS。**外层接收必须消费新target_reached和真实cap值，不能沿旧只看取消的判据。**
4. 当次资源/tx/monitor/业务pool/diskPool/probe/owned child正常有界归还；真实wait与精确backend scope。所有写者静止后外部准确SQL、实际保留schema/账与触限观察/在途写解释差额，旧失败和半成品继续计费。

这条growth若同时取得上述证据，就补到新行为，不另添加一轮完整readonly矩阵；若只缺共享实际回执或正确cap关联，只对该缺段有限补证/保持pending。若真实收口出现86、提前freshness错误或晚值接纳，则按其具体直接影响决定定向SQL取消/更早freshness/关闭负例，不把任意未测自动升级成全4模式重跑。

首次growth扫描内与最终库大小差异仍unknown；下一阶段不能靠强制刷新或扣噪声解释。当前静止库值已由上次增长改变，原global/tier ledger不重置；以新现场原始值核累计，不硬套R08旧idle大小为当前结果。

## 时钟、状态与交回

唯一Tnew **2026-11-04T10:22:37.932710Z**；完整启动门 **11:15:37.932710Z**，stop-new **12:47:37.932710Z**、hard **12:52:37.932710Z**保持。所有返工/离线失败/等待/本次Review计此attempt，不重新计时；Source PASS不延后full门，也不能因赶门放低实际证据。装不下则如实未完成，不伪完整性能PASS。

未确认新增finding，固定R09 Source Gate **PASS**交Lead现场接收；本报告没有采用或运行许可。原6closed/12pending、growth/leak/九语义/full pending、旧完整性能NRU、large/canonical NRU及任务active保持。完成 **STOP**，固定摘要及actual STOP见artifact-sha256.json。
