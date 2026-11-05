# S2.4 follow-up r04：FUP-R03-001独立有界复审

**计划delta Gate：PASS，仅限本次时钟/资源身份合同修订。FUP-R03-001 CLOSED；新增finding 0。** 不代表H实现已完成、源码Review通过、P0通过、运行授权或性能PASS。原代码6 CLOSED / 12 pending、E001 CLOSED、旧批次完整NRU及任务active均不变。

## 固定对象与两轮实读

objectType=`design`；正式对象 `test-plan-followup-r04.md`，实际30288 bytes，SHA256 **B1A645AA38D3BE822D058013BE24AE9164F9A7F8EA8FA0A523EC13F7C9A6E283**，起审与收口独立核一致。固定 `D:/AID/VSCode/Codex/Han` main / HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca（用户指定，未执行Git核查）。原TestEngineer已STOP，Reviewer为原非作者独立评审者，沿原Workflow，不重新Route。

自身Role、add-code-review及适用规则沿已取得的有效全文，按design对象核合同与消费者，不要求非代码计划先构建或执行测试。用户已授权建议值，无需再次确认；Lead后续采用与执行前置不由Reviewer代替。

Round 1：读取r03→r04完整文档差分、r04新增§4.1全文（第104～143行）及相关P0/Tnew/资源段落；重读原FUP-R03-001 report/verdict及关闭条件。逐条核字段、legacy、新attempt强绑定、旧档位身份和首次medium receipt、拒绝条件及实施前置。其余12项L/N/W映射与145/150/53分钟安排未重审，沿原已核结论；差分没有改变它们的实质合同。

Round 2：只按旧freeze-r02的H policy/window、初始化调用、baseline helper及MediumRetry父/子消费者核静态mapping，并搜索该helper的实际调用点。旧H:1451、3052、3067、3070四处调用均在新合同的初始化/父子映射范围内。policy还被point runner H:2817消费；保留的121/60/tail1字段继续供它使用，资源origin不参与phase/point执行截止。未全量重审H或其他业务源码，未审Backend在途guard实现。

映射定位文件为 `author/r02/source-freeze-r02/pg_perf_harness_test.go.snapshot`：131747 bytes，SHA B8D11EBFB3E4D15F01BC15263E1899BF168732CB06B232099B0F1FD9759E8E86。本轮只读hash观察时live H仍同bytes/同SHA；这只是当时观察，**不是接收后续guard或origin实现的新freeze**。H后续变化须按新固定修订另审，本计划结论不跨hash套用源码结论。

原r03、旧review report/verdict、small文件摘要均再次核保持。独立差分及pin/调用点核证保存在 `r03-r04.diff`、`offline-evidence.json`。唯一写集为本review-r04目录；未Go/PG、未读凭据、未Git写、未清理、未改候选/源码/旧制品/Tasks/Plan、未委派。

## FUP-R03-001逐项关闭依据

原缺陷是新执行Tnew被旧helper拿去与旧ledger.Window比较，导致新attempt在CREATE之前确定拒绝。r04明确承认当前freeze不能直接连通，选定H消费者小delta作为先决实施合同，未把“换policy说明”或“换空目录”当解决。

| 原关闭要求 | r04具体落点 | 本轮判断 |
|---|---|---|
| 新时钟与资源身份分别输入 | 第108～116行：window_start只表示Tnew/执行截止；resource_window_start固定旧字面origin；resource_base_bytes、resource_baseline_dir及每档resource_baselines显式绑定 | CLOSED（规划合同） |
| 不退回旧执行截止、不因新时钟重建资源账 | 第112～120、126～128、136行：执行用新Tnew；旧prior.Window比较resource origin；过期execution拒绝，旧origin不能放行过期execution | CLOSED |
| 旧small及全部已有档位Bytes/来源不变 | 第118、128行：旧small长度/SHA/Window/Tier/Bytes全部校验，读取只返回prior.Bytes，禁止重写/换行/格式变更或返回current；采用前重核已有medium/large，不可漏列后当首次登记 | CLOSED |
| legacy兼容不能绕新attempt绑定 | 第120、132行：仅字段缺失可走明确旧调用；存在但空/null/非法拒绝；新attempt须全部resource绑定，身份按policy/父批次/采用记录识别，不得伪称legacy | CLOSED |
| 初始化、helper、父/子所有调用点一致 | 第126～132行：policy输出分开的强类型输入，init显式传resource，helper构造/比较旧origin，父/子及runner不再隐式用execution重建资源Window；环境只核一致性、不可覆盖policy | CLOSED |
| 首次未登记medium仅一次，后续重试不重置 | 第134行：确实未开始/文件不存在/首次许可后，业务写前以准确当时DB大小O_EXCL式原子独占登记，Window仍旧origin；完整关闭后追加不可覆盖receipt，含文件SHA/长度/Bytes/父manifest/attempt/时间；当前与以后调用核同一记录，后续policy绑定receipt | CLOSED |
| 错配、半成品及竞争拒绝 | 第134～136行：既有缺失、未知/未绑定档、错origin/SHA/长度/Bytes/Tier/路径、global不符、small缺失、首登记竞争/半成品/缺receipt、child错配/环境覆盖、execution过期或policy未采用均fail-closed，旧物不删除重建 | CLOSED |
| 明确实现owner、新版本及有限回归先于P1 | 第138～143行：Backend H单文件delta；guard另报设计/证据；捕获完整五文件新freeze，针对正负绑定/legacy/medium child/环境覆盖作局部回归，原Reviewer与Lead接收该实现版本后才能Tnew/P1/新schema | CLOSED |

现场旧 `small.json` 实际 **80 bytes**，SHA **2B368282C7667FC2B3563F82FC293FBBEEE59A350A52CB41D08C57BDB87FC4D2**，Window=`2026-11-04T03:29:04.586115+00:00`、Tier=small、Bytes=1406932671，与新合同一致。80是文件长度，不是资源额度；1406932671是DB累计基线。精确字符串origin要求保留旧字节，执行起点按同一时间点验证，两种比较语义明确分开。

本次CLOSED指原**规划合同缺口**由独立Reviewer核销，不是文档自动代签或原代码finding核销。§4.1所述计划不自行核销/执行门不通过的边界，不能反向解释为必须等实现完成才允许关闭本规划finding；本次独立结论与实现门分别登记。

## 新增风险检查与限制

1. legacy fallback仅允许字段缺失；新attempt还须强制识别身份、强制全部资源字段，因此不能用null/空值或子进程陈旧环境借legacy创建新余额。未来实现必须有对应拒绝回归；本次只确认合同有要求，未证明parser或child环境现已做到。
2. medium首次登记产生新文件及新receipt，但旧文件不回写。部分文件、竞争失败、receipt缺失/不符都拒绝，不能删除重建或重新按当前DB大小发余额。receipt完成前不得开始medium业务；registration路径的具体并发与持久化实现仍由后续代码Review验证。
3. guard采集替换与origin接缝都可能改变H摘要。r04已把旧五hash降为历史定位，明确要求包含真实H改动的新freeze和影响检查；不能只因其他四文件未变就把旧H回归移植为新实现PASS。本文没有选择或批准Windows采集器算法，也没有把47ms诊断当增长/取消下正确性的证明。
4. P0仍有正确growth/cancellation/资源等价证据门。需要真实新schema才能取到的guard证据，必须先列明新attempt准入验证、责任/范围和费用；采用实现版本并记录Tnew后计入新attempt，不可倒填或排除这些成本。未接收前不进入leak、九语义或完整采样。**本计划PASS本身不准许运行任何这类验证**，也不证明P0已经满足。
5. 历史small约465.09MiB余额仍只是历史账面值，采用/执行前要重新预检；资源不足或guard/接缝证据不足即保持NRU/PENDING。没有容量保证、重置资源账、清理旧schema或延长旧窗口的许可。

上述均有明确前置/拒绝和后续证据责任，没有确认新增设计finding；不把尚未实施当作本次非代码计划缺陷，不为本计划要求运行Go/PG。

## 计划Gate与后续执行边界

本次PASS只针对固定r04对FUP-R03-001的合同修订及新增风险处理；不重开原12映射/145/150/53结论，也不自动采用policy或创建Tnew。

执行前仍需新H消费者实现、guard适用证据、完整五文件新freeze、针对性回归、原Reviewer独立代码/接缝复核与Lead对该实现版本的实际采用。P0增长/取消等正确性条件未满足，不运行依赖活动；原12项不能因本规划自动核销，完整性能和独立执行Review尚未发生。

原11:29批次13:54停新操作/13:59硬停及完整NRU历史永久保留，新attempt结果不反写旧履历；原资源origin/base及全部旧档位账继续累计。121/60s、tail1s、完整calibration-small-medium、large/canonical NRU均沿既定合同，整卡仍active。

旧review报告原字节保留，作者/源码/Tasks/Plan不代修。独立有界复审完成后 **STOP**。
