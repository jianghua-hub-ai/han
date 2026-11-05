# S2.4 probe-error R06 独立有界复审

**静态 code Gate：PASS。PROBE-R05-F001 CLOSED；本次必要新delta未确认新增finding。** 实际reader调用的窄helper不再以ctx.Err覆盖非context/mixed错误，Closing汇总能保留失败标记与安全分类。结论仅覆盖固定R05→R06修复及直接消费者，不代表PG、P0、性能、整卡或E2E通过。

## 对象与权限

objectType=`code`，固定Han/main，指定HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。原非作者Reviewer沿有效Role/add-code-review/适用规则继续。唯一写集为本code-review-error-r06目录；未Go/LOCAL/PG/SQL、未执行作者runner/binary、未读凭据、未Git/清理/源写/旧制品改写/Tasks/Plan/委派。

正式对象 `author/probe-error-r06/source-freeze-r06/manifest.json`，captured `2026-11-04T08:32:19.867676+00:00`；source_stop/local_stop=true，作者本步未打开PG。H **214370B / 550B16614224A13F31CEE931C016C3B6841F59E80F27CE2A61F24720219392E7**。基线固定R05 H **209145B / E97318D68C8BE670DF11A196D469A3D91F5BAC979A1CF65EFDDCBAE4F09DAFC7**。五snapshot及五before字节/SHA自行复算匹配，W/R/N/docs未变，旧global/window/policy/small四摘要匹配。

未用live H代替快照；Lead现场live/source/范围/所有权核验与后续接收由Lead完成。本报告不把作者STOP/报告或Lead摘要当独立执行结果。

## 两轮实读与核证

Round 1完整读 `harness-r05-to-r06.patch`、独立差分和新增测试。离线逐hunk核上下文，把8个hunk应用于固定R05前像，重建结果与R06快照的LF文本完全相同。修复仅涉及reader错误归一化、collect的Closing返回、诊断/classifier、新测试及fixture根目录；没有借此重开原18项或R04/R05全部raw。

Round 2仅沿真实 `readDiskObservation → pgPerfResourceReadError → collect/stopLocked → read_error_class/failedOwn → closeResources`，并核测试是否使用同一helper；核只读入口函数未变但reader依赖已变的版本边界。实际读取原R05report、新作者report、command-index、manifest/binary与fixture事件。Role/规则及已采用15/5/20、clock/ledger合同沿有效输入复用。

9条已完成命令逐项复算result/raw SHA，核真实wait exit及前后五hash，独立计算current标记和事件数。旧R05只对既有pin做hash连续性核证，未重读全部旧raw内容。Python脚本仅离线读取/hash/差分，不是Go执行；详见verify_offline.py、evidence-audit.json、selected-events.json。

## PROBE-R05-F001：CLOSED

| 原关闭条件 | 本版实际落点与独立判断 |
|---|---|
| 真正适配器先判原错，取消不覆盖noncontext/mixed | H **1855～1856、1865～1866** 两个真实Acquire/SQL错误分支调用同一 `pgPerfResourceReadError`；H **1884～1911** 先判原错是否纯context，noncontext/mixed返回安全非context类型。不是只改测试中的模拟reader。 |
| 保留安全错误分类，不输出敏感原文 | H1878～1881安全类型仅存stage、动态类型、验证后的5位大写字母/数字SQLSTATE和context分类。无原Message/Detail/Hint/包装文本或原错Unwrap；无新DSN输出。H3113起classifier优先识别安全类型，避免join中的取消遮掉真正分类。非法SQLSTATE降为invalid。最终同版raw未出现synthetic-sensitive测试字符串。 |
| Closing仍升级资源失败并传播原因 | H1794先记录ReadErrorClass；**1801～1805** 已有正常关闭terminal时仍置failedOwn=true，返回terminal与安全原错的join。H1841新诊断明确落read_error_class。未变closeResources **1115～1122** 在h.group join后读取failedOwn，写resource_failure并对true报错。因此不能再因正常关闭terminal为context而静默消失。 |
| 纯context正常关闭不误报；晚成功不发布 | helper纯context归一sentinel；未变stopLocked按owner已取消与context-only保留正常关闭。新PureCancellationAndSuccessfulLateResult覆盖sentinel/包装/join/nil及取消后成功不发布；没有改变late/freshness合同。 |
| 有限离线回归到达真正窄helper和collector | 新ReaderErrorDuringClosing按acquire/SQL × pg/wrapped-pg/plain/mixed共8组合，先明确normal terminal再释放reader，调用生产helper进入collect。普通及race实际raw均见resource_failure=true、SQLSTATE42501或plain安全类型；不是旧测试绕过adapter的缺口。 |

行为RED `red-reader-r01` exit1/3.141s：带抽取旧行为helper的中间版本，8组合及父共9个fail事件，实际resource_failure=false/class=context_canceled。不是精确原E973全文重跑，也不是实际PG故障。最终samehash普通16pass、race count3合计87pass；纯取消、原Closing/RequestWait及必要phase/outcome smoke均到达。测试没有连PG，不称为真实服务端错误复现。

满足原finding的实现与有限回归关闭条件。closeResources汇总中的terminal本身可仍是首次正常取消；实际异常分类在返回join/diagnosis中，resource_failure=true明确使harness失败，不要求重写最初terminal或另建状态框架。

## 不变边界与同版证据

manifest声明的5个函数段均独立与R05逐字比较，并复算LF摘要：ReadOnlySQL、budget parser、stopLocked、closeResources、context-only一致。ReadOnlySQL函数摘要 **19C7DC6BAE27EDC71D2DC98039904408927D6489BF541F7E49712444900EEBE6**，旧只读target/noNewHarness/noDDL静态判断复用；本步未执行它。reader依赖已有修复，未来只读执行须绑定R06实际binary/源码，不能拿旧R05 binary或旧Main七check充R06执行证据。

本次8个hunk未改15/5/20、新鲜度START锚点/独立expiry、root/phase owner、121/60/tail1、clock/resource origin、caps、ledger或fixture数据规模。CLOCK-R04-F001原关闭不重开。

| 最终同五hash leaf | actual wait exit / wall | 实际证据 |
|---|---|---|
| green-r06-final | 0 / 2.937s | 普通count1合计16pass、0fail/skip。 |
| race-r06-final | 0 / 17.672s | count3共87个有Test的pass事件，含子测试，0fail/skip；不是87再乘3。 |
| build-final | 0 / 7.219s | 新R06 integration/race binary构建。 |
| vet-final | 0 / 1.468s | 静态检查。 |
| format-final | 0 / 0.078s | diff空。 |

另4条属于中间hash，保留为历史；green-final/green-reader-r01/race-final不冒最终同版，RED也不改绿。3份最终非合作child证据已沿raw读实际JSON，核PID/exit86/UNKNOWN/root cancel及inventory摘要，落点均在R06内；其真实wait86是预期故障收口，不是资源采集成功。

binary.json指向 `tools/r06.test.exe`，独立复算 **32664968B / 99F5004989B701191EEB2AF0D363410026A993A93A544442D796BFE792958A5D**；明确不同于R05 binary。build argv输出与race运行路径一致，构建先完成，五hash一致。race CWD为R06/core-sandbox/a/b/c，实际嵌套fixture也在R06/core-sandbox内，不是新worktree。摘要为交接现存binary核证，不虚构每次执行瞬间另捕获的binary签名；Reviewer未运行该binary。

## fixture写集事件与证据连续性

**保留作者承认的写集越界事实。** 中间race-final仍用旧fixture root，向R05/fixtures新增6个MkdirTemp目录；实际查六目录均在，**30个文件**的长度/SHA及目录内文件清单与fixture-scope-incident.json全部一致。中间raw的3条parent fail-stop路径亦对应其中3个目录；其余成对child fixture由旧测试先建fixture后使用继承证据目录的路径产生。没有清理、移动或声称回滚。

最终H将fixture根明确改成probe-error-r06；最终普通在R06，race在R06新sandbox，3份实际fail-stop/raw路径均在R06范围。此事件属于中间测试写集偏差，不是新的源码HIGH，也不把中间版本51pass替代最终87pass。

独立复算旧R05 manifest及5快照、原四资源摘要、旧Review封存文件；另对旧审计已固定的21条result及stdout/stderr、3份F001 fingerprint/所引ledger/receipt、3份fail-stop/inventory作hash连续性核证，均匹配。脚本计69份主记录（另有其引用文件）不变，**没有发现该事件损坏这些固定旧证据，最终同版回归边界可区分。**

限制：没有事前完整R05文件树inventory，不能凭本次事后读取保证每个未被pin的历史文件都未改，或独立证明每个目录在事件前不存在。原文件未改的无限范围表述不能照抄；Main继续按现场scope/旧sha接收。该事件留档与静态code Gate分开，未授cleanup，Reviewer也未做cleanup。

## 交回与STOP

本finding有限复审完成，**PROBE-R05-F001 CLOSED，R06有界静态code Gate PASS，无新增confirmed finding**。原R05 CHANGES REQUIRED报告保留历史字节不改；此PASS不是替Lead接收实现或授权下一运行。

P0当前版真实准确SQL/受控延迟分账、超15或更早freshness取消、真实growth/观测cap、PG连接/backend/资源归还尚未核证；growth/leak/九语义和原12pending不因本报告核销。没有Tnew或新PG窗口由Review产生；旧完整性能NRU永久保留，完整calibration-small-medium、large/canonical及最终执行Review仍按原门，任务active。

两轮有界报告完成，**STOP**。最终report/verdict/audit摘要与actual Reviewer STOP见artifact-sha256.json。
