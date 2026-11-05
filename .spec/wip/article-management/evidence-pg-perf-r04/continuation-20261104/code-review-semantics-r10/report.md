# R10 H窄delta独立Source Review

**Source Gate：PASS，仅限固定R09→R10源码delta及离线回归；无确认新finding。** 不是PG、normal10k、整组N9或性能通过。原9 CLOSED/9 PENDING及SEM-R09-EV001保持；新增SQL从未真实执行，EV001不能由本源码Review关闭。

固定Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca为授权边界；本Reviewer未运行Git。唯一写集本目录，未运行Go/PG/SQL、读取凭据、修改源码/SSOT/作者证据、清理或委派。沿原add-reviewer/add-code-review及已完整读取的适用规则，复用R09已审结论，不重开18项或932项旧审计。

## 固定输入与两轮实际范围

自行复算manifest SHA **D4047EA1977948017B9B919054F9C3FDBC3415586136E130F76546F22BE6DD1C**；source_stop/local_stop均true。五snapshot/live摘要及bytes各自匹配：H250417B，**1C90DF0DF6726346F7987D11E430C5D2083C0DA3D3A205466A9DD560198AB7C5**；W19395B、R30182B、N49416B、docs13531B仍为原四hash。非race binary20086784B，**318EBF7A70220B4319C420A1643CFDFE9B1E1BE38122E97E07BBD0118CF52EF0**实际文件匹配。

采用输入为C0698534…6473B2计划最终全文及43B7F52C…FA5C8EB采用记录，均重算匹配。后者明确本attempt PG未授权、R10整组永久NRU，允许H-only实现与离线准备；不是对PG准入门的绕过。原policy7C94、Tnew及旧四资源摘要重算保持。

第一轮自行从R09固定H快照C3DFDC66…D89B8B99和R10固定H生成真实diff，完整核所有改动；没有以作者或Main的patch摘要替代。改动仅directed标识/局部预算、预算证据、directed竞争快照、seed进度及测试fixture新目录。独立差分保存为r09-to-r10.offline.patch。

第二轮限真实消费者：pgPerfDirectedTier→newHarness→h.tier/seedCtx/SeedMS；directed case表→raceRow→recordRaceOutcome→既有writeJSONExclusive/finalErr；seed/COPY→seedProgress；冻结W/N的竞争业务调用及原断言不变；新增SQL只对照必要migration列/类型。实际比较既有reader、policy、monitor/checkDiskBudget、资源关闭、poll/final-close、baseline及错误分类函数与R09未变。没有全库源码预读或再跑旧PG。

## 预算与正式矩阵隔离

H2078起pgPerfDirectedTier接收tier值副本，只有directedCase=initial-drain-10000、code=r2n3b、small且原120/300时将副本seedBudget改240；条件错误安全拒绝，未修改全局pgPerfTierSmall。唯一生产式设置directedCase的位置是RealSemantics case构造（H6195），其他正式调用保持空值。

newHarness把同一有效tier写入h.tier并用于root contextBudget和seedBudget；root仍300，seedCtx由现有root派生240，自动受更早root限制。RootStarted/RootDeadline、SeedStarted/SeedDeadline与profile默认120/有效240/root300、采用ref/SHA进入directed fingerprint和seed-progress；seedMS上限仍读同一h.tier.seedBudget。没有每表/每guard重置、不换medium连接数/数据量。seed后恢复原root，没有再发300s或另设drain窗口。

正式small/medium/large的directed标识为空，tier值原样返回；正式fingerprint维持原路径，seedProgress在空标识处直接返回。每≤1000行COPY前原准确guard与operation admit保留，未改15/5/20、resource owner/START、cap/volume/baseline或point121/60/tail1。seed进度区分copy-return的copied_not_committed与commit-success，记录analyze/seed-exit、schema/run/profile及原context错误分类；这提供未来失败进展入口，不把当前无PG进度写成实测。

240仍可能不足、root300也可能先到；此风险沿计划和采用接受，Source PASS不保证数据准备/drain成功。采用ref/SHA是固定源证据标识；任何未来实际运行仍需外层明确绑定采用、冻结版本及有效grant。本轮不存在可执行PG许可，不以硬编码该ref冒充已经通过运行准入。

## directed竞争取证与失败传播

recordRaceOutcome仅在raceEvidence=true执行；该标识只由moderation-race/delete-race directed case设置，正式W在首次分支即返回nil，**不新增快照SQL或outcome落盘**。raceRow的两参与者映射仍为原业务action和同version1；join回执容量2对应两worker，directed额外等双方Done后才读快照。正式路径的缓冲通知不要求额外等待，不改变原双方结果/锁条件/业务断言。

新单条只读参数化SQL使用h.pool既有schema作用域，按articles.id=$1::uuid取得article version/治理/删除状态/operation ID，同一a.id关联deletion_operations.article_id及audit_events.target_id=a.id::text。必要schema核证：0003确有article_id uuid及各article字段，0001 audit target_id为text；无凭空列名或业务schema修改。operations/audit均COALESCE空数组，允许真实无operation/无audit的赢家分支。**这是静态SQL映射，Go编译/单元测试不验证其真实PG执行，本轮未执行它。**

输出包含run/schema/logical case、同article ID、两个实际error结果的安全类型/类别及Precondition/Unavailable、action/participant/expected_version、snapshot、原锁图相对路径和PID数组，以及请求完成/deadline时点。participant0/1按同一results顺序与waiters PID顺序绑定；D02 action映射为beginT1/ReplaceContent，E02为ReplaceContent/Moderate。没有输出原错误消息/DSN，SQL对象是该合成竞争目标的行/operation/audit。

pgPerfRaceSnapshot在读前/后核原ctx，读取错/过期/晚返回直接出错，JSON解码与article ID不匹配拒绝；不另开宽限context。writeJSONExclusive保持O_EXCL，编码/建目录/打开/写入/关闭错误进finalErr，recordRaceOutcome读出该错误并与ctx.Err合并返回，raceRow→原迭代函数→t.Fatal传播，不能将取证失败当PASS。额外保护未取代W/N文件中原唯一赢家、业务SQL及审计断言；这些文件实际hash完全未改。

快照完整性在当前真实SQL表达式中由固定字段/COALESCE构造，离线validator主要核JSON/article ID；它不是任意外来JSON的通用schema校验器。本轮没有真实行值、锁后快照或落盘失败PG证据；后续实际Review仍须核同一article的双方结果、row/op/audit、锁图与最终业务断言，不得仅凭本Source PASS核销EV001。

## 离线/构建原始证据

本Reviewer只读取，未执行下列命令。逐个result的sourceBefore/After、owned PID/真实wait、argv/CWD和stdout/stderr SHA与最终freeze匹配：

| 最终同版命令 | 实际结果与范围 |
| --- | --- |
| race-final-r01 | exit0，-race/count3；39顶层PASS、含子测试126个PASS事件、0顶层FAIL/0SKIP。含R10预算及snapshot success/read-error/expired/late/wrong-id/invalid-json与受影响R09/R08/R06/R05回归。 |
| core-final-r10 | 实际持久r10.test.exe、隔离core-sandbox CWD、count3/exit0；45顶层PASS、84 PASS事件、0顶层FAIL/0SKIP。嵌套MediumRetry拒绝子进程FAIL保留为预期，不冒充顶层测试失败。 |
| build-nonrace-r10 | 同版go test -c真实wait0，产物hash/bytes与manifest一致。 |
| vet-final-r10 | 同版真实wait0。 |
| format-final-r10 | wait0且输出空。 |
| buildinfo-r10 | wait0，实际Go1.27/windows/amd64/integration，pgx5.11/puddle2.2.2，未启用race的持久binary。 |

行为RED exit1真实指出正常10k仍seed120/root300，非编译错误；green-profile-r01的492B是中间版，只作历史修复链，不借作最终同版通过。最终race/core已包含R10预算用例。snapshot边界测试为离线注入；没有另造旧版snapshot RED，也未动态覆盖全部SQL/写盘失败情况，写错误收敛还依据实际静态finalErr链核定。

Main另行执行七项目checks的summary及各raw哈希已实读复算：format/modules/vet/build/contract/race/tests各真实wait0，前后五源同最终hash。它们是Main执行证据，可能使用有效Go cache；不是本Reviewer跑的测试，也不表示PG、性能或E2E通过。自有offline-audit.py共83项pin/差分/消费者及命令证据核证成立，全部原件清单与SHA在evidenceaudit.json；没有重复932项N9审计。

## 交回与STOP边界

无确认需返工的源码finding；可接收本版窄Source Gate。实际正常10k240、snapshot SQL、seed进度/物理容量/同版N9均未执行，保持PG永久NRU，原9 CLOSED/9 PENDING与SEM-R09-EV001待真实证据，不修改任何finding或任务。原R09八PASS仍属R09历史，不移植成R10执行。

Tnew10:22:37.932710Z、停新12:47:37.932710Z/硬停12:52:37.932710Z不变；本attempt R10整组准入门已失、完整性能NRU不复活。本结论不新增grant/attempt或作采用决定。最终报告/verdict/audit/差分SHA及actual STOP在index.json。
