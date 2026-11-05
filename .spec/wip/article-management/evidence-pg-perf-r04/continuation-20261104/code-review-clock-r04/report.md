# S2.4 clock/resource r04独立有界CodeReview

**Clock-r04 Gate：CHANGES REQUIRED，唯一实质finding为CLOCK-R04-F001（MEDIUM）。** 首次medium登记/复用后的有效ledger与receipt身份未进入fingerprint。没有确认quota重置或新增HIGH；执行/资源origin分离、旧账加载、跨进程不重置等已审范围及作者最终同版局部回归有效，但不足以弥补该证据消费者遗漏。

结论仅属于下列固定clock快照，不是live H、combined r05或最终执行Review PASS。PG/Tnew/P0/完整性能门未过；原6 CLOSED/12 pending、E001 CLOSED、旧批次完整NRU和任务active不因本报告改变。

## 固定对象与权限

objectType=`code`；固定 `D:/AID/VSCode/Codex/Han` main / HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca。正式对象 `author/clock-resource-r04/source-freeze-r04/manifest.json`，captured2026-11-04T07:23:02.124359Z；H快照 **168839 bytes / B8A395DE6F1E50A6AC55DEC9E4D4B3AB87EEA5AD1295C138AAB8D4B9D5B6158E**，基线前像 **B8D11EBFB3E4D15F01BC15263E1899BF168732CB06B232099B0F1FD9759E8E86**。

五snapshot、五份before均自行复算bytes/SHA，四份未改W/R/N/docs与对应工作树文件hash一致。旧global/window/policy/small四摘要再次核一致，small原80B/SHA不变。**未读live H正文或hash代替快照**，作者已clock STOP后另获budget源授权不改变本次对象。pin及完整五hash见pin-verification.json。

实际消费已采用 `test-plan-followup-r04.md` §4.1、`lead/followup-clock-contract-adoption-r04.md`、原planning PASS、作者report/manifest/command-index。Role、add-code-review及适用Go/rules沿有效已读全文；原非作者Reviewer独立完成，不替作者修复。不把planning PASS或作者自测标成独立执行PASS。

本Reviewer无LOCAL/Go/PG权限，Backend持LOCAL；本轮仅静态阅读与独立Python离线hash/差分核证。未运行Go、PG、作者runner或binary，未读凭据、未Git/清理/委派，未改源码、旧作者物证、Tasks/Plan。唯一写集为本code-review-clock-r04目录。

## 两轮实际检查

Round 1：完整读 `harness-r02-to-r04.patch` 的1170行及新增函数/测试。独立生成前像→快照diff，并在离线内存按author patch的11个hunk校验上下文、重建结果与固定snapshot完全一致。核policy字段/legacy/null/执行过期、严格路径/全档资源清单、baseline/receipt首次与重复、child env、harness初始化与point policy缓存、fingerprint新增项；没有重开其余18项源码全审。

Round 2：沿真实consumer读固定snapshot的policy→初始化→point、global source→resource manifest→baseline→receipt、父/child实际调用以及writeFingerprint完整上下文。W/R/N只核必要构造/调用定义及未改摘要，它们仍经newPgPerfHarness进入统一point；未扩到无关业务库。重点反查旧F004跨进程基线、首登后同attempt/下一attempt复用、证据输出是否消费有效状态。由此确认下述唯一MEDIUM。

读17条已完成命令result及stdout/stderr，复算每条SHA、before/after五hash并独立计算是否为本freeze；核最终同版race/core事件、失败修订及CWD/binary链。不以角色自检、168事件或静态检查单独推导正确性。详见verify_offline.py / evidence-audit.json。

## 唯一finding

### CLOCK-R04-F001 — MEDIUM：首次medium登记后fingerprint仍只有unregistered授权声明，缺实际已加载ledger/receipt身份

- 对象/位置：固定H **1678～1683、1733～1738**；直接生产者 **3384～3421**，证据消费者 **789～850 writeFingerprint**。本报告H行号全部指clock-r04 snapshot。
- 触发：当前真实原目录只有small，后续首次获准medium会使用policy中medium `state=unregistered / allow_first_registration=true`。pgPerfTierBaseline写入medium.json、完整关闭并写medium.registration.json，复核后只返回int64基线。其调用链没有把实际file bytes/SHA/receipt摘要返回或写入有效观察记录，也未更新原policy map。
- 现状：初始化把返回值放入evidence.tierBaseDBBytes，却仍将原 `policy.Resource` 放进fingerprint的resource_binding。该medium声明继续为unregistered、Bytes=0、FileBytes=0、SHA/ReceiptSHA空。writeFingerprint只复制extra，不补实际ledger身份。同一原policy下的后续harness虽然helper正确复用首次baseline，其fingerprint仍重复这个授权声明；database_size_base记录的是当前库大小，不能代替首次有效medium基线及receipt摘要。
- 影响：约定的新attempt/fingerprint证据无法直接核到该harness**实际用于计费**的medium ledger/receipt长度、SHA和baseline，首次及同attempt复用证据缺口被局部unit GREEN遗漏。独立消费者必须另读可变路径并自行补归属，缺少与该fingerprint绑定的有效状态。已采用r04 §4.1要求执行/resource身份、每档SHA/长度/Bytes及来源共同留证，首次登记receipt也须供当前及后续核同一文件。
- 边界：**这不是证明额度已重置或helper丢失原baseline。** 原policy保留“允许首次登记”的不可变授权声明是合理的，不能为修fingerprint回写原policy或旧账。缺的是另行追加的、已实际加载/登记的有效资源观察及receipt身份。
- 最小修复方向：返回真实Backend作者，仅在H将baseline加载/首登复核后的有效metadata传到该harness证据输出。至少包含实际tier/path/origin/Bytes/file_bytes/SHA、receipt路径/摘要及首次登记来源，明确区分policy授权binding与实际加载记录。沿既有fingerprint或追加不可覆盖证据即可，不新增资源框架，不改旧文件/旧policy，不动W/R/N/docs。
- 有限关闭条件：提供固定新snapshot/delta和**离线**行为证据：首次medium登记后输出含真实ledger/receipt身份；同attempt第二次加载及后续existing绑定仍输出原有效基线/SHA，不拿更大的current DB值重置；原policy声明与原small80B/SHA不变。复审核输出与实际文件/receipt摘要相等、调用链实际消费该输出，保留F004独立child非重置证明。无需为此运行PG或重做未变guard验证。

该缺口可与当前H budget patch一起由作者处理；本Reviewer不代修。静态完整路径与触发条件保存在finding-trace.json；未运行Go/PG构造反例，不把静态推导称为独立实测。

## 其余已审clock行为与证据

| 范围 | 静态与当前同版证据 |
|---|---|
| execution/resource分离 | H1358解析执行时间点及固定资源字面origin；newHarness存解析后的policy，point复用h.policy的121/60/tail1，不把resource origin当执行截止。新execution加载旧small、等价执行时区均在最终同版用例到达。 |
| legacy/null/expire | legacy仅缺resource origin且原revision/旧起点、无新attempt/parent/adoption身份；显式null/空/部分resource拒绝。过期/未来或env执行时钟冲突拒绝，旧窗口不复活。 |
| 真实路径/全档绑定 | policy paths来自编译源位置，严格绝对路径/祖先reparse及同一文件身份；三档声明与实际目录检查，small固定80B/SHA/Bytes。复制路径、junction、遗漏/额外档位、错hash/长度/Bytes/Tier/origin与global source/env覆盖均有最终本版负例；无法确定绝对来源时fail-closed。 |
| medium登记/receipt | H3384先验证全档，首次许可仅medium；O_EXCL完整写基线并close，再O_EXCL receipt，部分/缺失/孤立/冲突状态拒绝。后续existing显式绑定ledger与receipt摘要；large不获首次许可。没有确认这条加载路径重置baseline；metadata输出受F001限制。 |
| F004父子 | 显式传policy+inputs，env大小写去重覆盖陈旧execution/base/目录；独立child实际沿用base+100，current增至base+300仍保持原baseline，tier_delta200超模拟150。两个错配child真实exit1、命中policy错误，由父核验后pass。原F004不因本次新增metadata finding重开。 |
| guard边界 | 离线比对checkDiskBudget、pgPerfDatabaseSize、pgPerfEnsureDatabase前后函数文本相同。准确SQL/旧5s在clock对象未改；新15/5/20 budget属后续combined源码delta，不据此要求clock对象重跑PG或把它错判为已实施budget。 |

没有确认其他实质finding或HIGH。legacy/路径/receipt并发实现的判断限Windows当前对象；作者未做跨平台/trimpath，路径不可确定即拒绝。没有真实PG权限/场景执行/容量保证或原12项核销。

## 原始命令与失败链

17条记录及logs摘要均独立核。只有以下6条before/after五hash与本freeze完全相同；较早GREEN不跨hash借用。

| 最终同版leaf | 实际wait exit / wall | 已核范围 |
|---|---|---|
| race-clock-r04 | 0 / 14.891s | count3合计168个有Test的pass事件（不是168再乘3），0fail/skip；junction拒绝实际到达三次，含父子正向及预期错配child。 |
| build-core-r02 | 0 / 5.656s | integration/race编译同版core-final.test.exe。 |
| core-race-r03 | 0 / 11.093s | 原core选择count3，57个pass，0fail/skip，隔离sandbox CWD。 |
| vet-r02 | 0 / 1.625s | vet静态检查，不代PG。 |
| list-r01 | 0 / 3.484s | 入口列举，不代运行。 |
| format-r01 | 0 / 0.109s | diff为空。 |

core build输出与run参数指同一core-final.test.exe，源码五hash一致，构建完成先于运行；Reviewer另复算现存binary SHA并记录race标记。run CWD位于本作者core-sandbox，既有相对fixture路径解析到该sandbox内，**不是另一个worktree或另份源码**。binary摘要为离线运行后观察，build/run未另记当时binary摘要，不虚构历史身份签名。Review未执行该binary。

保留旧失败：red-origin-r01为带回归的旧实现修订，exit1/3.578s，触发原window冲突；compile-r01已编译并运行，legacy fixture将“缺字段”写成null后实际失败，不能称编译RED；core-race-r01是PowerShell参数拆分，未获得正常core执行；race-clock-r02长路径导致junction fixture创建失败，目标拒绝未到达。后来在正常package CWD以最终同版race-clock-r04补齐junction，0skip，不擦除旧错误或伪称旧失败已经通过。

上述真实局部结果说明重要修复和回归有效；测试没有通过真实newHarness完整初始化检查首次medium后的fingerprint，因此不能据全部GREEN排除F001。

## 交回与STOP

**最少修复集合仅CLOCK-R04-F001。** 作者可在下个combined r05固定修订中追加有效ledger/receipt证据消费者及有限离线回归；后续只核newdelta、该finding和必要受影响消费者，不要求重复本轮17条全部raw、未变guard PG或原18源码全审。

作者clock在07:23:02Z声明SOURCE/LOCAL/PG STOP，本步骤未开PG；Lead其后另授权budget源，并不使live H成为本次已审对象。当前NO LOCAL/Go/PG边界持续遵守，无Tnew或真实新窗口由本Review产生。

原clock规划PASS与budget规划PASS保留；本固定clock代码因1项MEDIUM为CHANGES REQUIRED。combined r05必须等其真实freeze、独立delta Review与Lead现场版本/范围/所有权复核；P0、PG/增长/取消、完整calibration-small-medium及执行Review另行核证，旧批次NRU不洗、任务不关闭。

独立两轮有界报告完成，**STOP**。
