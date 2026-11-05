# ISSUE-article-management-0001 — 恢复已不存在文章时重放与快照计数不一致

**状态：技术修复、回归、非作者CodeReview及Lead核收完成；待精确两方结果接收，尚未验收关闭。** r22设计及r20测试规划均通过独立评审。本问题在继续 `article-management-S4.4` 测试准备时发现；用户已授权采用建议值继续推进，Lead据实际证据安排缺陷闭环。原已关闭S4.2的历史结果保留，不将其原作者自审改称独立评审。

## 登记与定级

| 字段 | 本轮事实 |
|---|---|
| 编号 | ISSUE-article-management-0001 |
| 严重度 | P1，Lead建议并采用：核心恢复重试功能受损；不是已发生的生产事故或已证实的安全绕过 |
| 来源 | S4.4恢复差额准备；关联已关闭实现S4.2及S1.5/S8.2的未完成恢复验证 |
| 工作区 | `D:\AID\VSCode\Codex\Han`，main，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 初始定位固定输入 | 179项；设计r21、测试规划r19；当时生产文件及原测试均未修改 |
| 定位执行者 | 实际Backend-Dev Dalton，`01a17dcf-f2ed-7dc2-b7f3-6c72cf771448`；Lead现场核证 |
| 必要合同澄清 | 原Feature-Designer Hubble，`01a17daa-c655-73c2-a373-335fea4d3b54`；已完成唯一design.md r22并停写，Pauli两轮独立设计评审有界PASS |
| 初始组件实测范围 | Windows、真实ProductionDeletionReplaySource/FileBackupInventoryReader/Provider与共享canonical；Journal、Replayer、operation/audit为固定公开seam，Store复用原内存测试helper |
| 最新真实PG实测 | 原179项不变，仅新增一文件形成180输入；真实Finalizer/Repository/Audit/PG事实/FileInventory/PGStore/Provider及共享decoder，Journal使用公开受控seam。三leaf均实际行为RED，无编译/fixture/环境失败，详见下文 |
| 尚未实测 | 真实文件Journal、旧备份及部署恢复；本次修复后PG子集与普通/race回归已通过，详见下文，不将受控Journal的PG链外推为真实恢复全通过 |

## 复现与证据

Dalton只在Temp完整179输入及Go overlay中，对现有 `internal/article/deletion_replay_source_test.go` 追加诊断测试；原文件字节前缀、原断言和全部生产代码保持不变。精确命令（overlay路径见证据）为：

```text
go test -overlay <retained-overlay.json> ./internal/article -run '^TestDiagnoseReplaySourceCounters$' -count=1 -timeout=30s -json
```

三次均成功编译后exit1，每次3个测试节点、2个leaf、0skip。`DeletedCounterControl`（1/0）实际PASS；`CompletedAlreadyAbsent`（0/1）及父节点FAIL。固定Journal记录、inventory字节、operation/audit事实及反复Snapshot均未漂移；只有两个参与canonical的计数字段不同。

| 真实source调用 | tombstones_replayed / already_absent | 共享authority实际摘要 |
|---|---|---|
| ReplayOldBackup | 0 / 1 | `sha256:0fb9835e2f692f82bdee5bd30a170b50ad6a6ae0cf26ffab40fd12cb4a6cac12` |
| Snapshot | 1 / 0 | `sha256:769b97f01c2d50cd38f0f89b26dcd0cbad60b023c41dbd7d7ab1cdd50880ddb2` |

同一诊断还实际经过原Provider与未修改的内存Store helper：`ReplayAndVerify` 返回 `unknown/evidence_stale`、evidence数0；之后 `ReadStatus` 返回 `pending/replay_in_progress`。这是真实组件行为，不是实际PG发布结果。

59份诊断源码、完整overlay绑定、原始JSON/meta、六套保留fixture与canonical/public输出及Lead核证见[诊断证据索引](../wip/article-management/evidence-s4.4/diagnostic-r01/index.json)，SHA256 `FE693C3DD308408DD78D43FCBA56654903FE46C19F3B9BE23DE9D72468E52999`。完整179输入与先前已固定快照逐项相同；索引保留共享副本及重建路径。准备与主机条件另见[恢复准备索引](../wip/article-management/evidence-s4.4/prep-r01/index.json)，SHA256 `DD878216BD69694446ED159629D10BD2C467FCB49FB9B6DFE13B3EAE52D6A66A`。

## 根因与影响边界

[DeletionFinalizer](../../internal/article/deletion_finalizer.go) 的ReplayAll按真实物理结果分别累计Deleted与AlreadyAbsent；[生产source](../../internal/article/deletion_replay_source.go) 的ReplayOldBackup保留该结果（119行），Snapshot却固定生成 `Replayed=len(tombstones), AlreadyAbsent=0`（141–143行）。两字段都参与唯一recovery authority的前16字段canonical。实际固定事实下因此得到不同摘要。

[PG evidence Store](../../internal/article/replay_evidence.go) 的Publish在341–354行比较候选与Refresh摘要；ReadCurrent在487–499行比较现事实摘要与当前证据。由这些真实分支可推断计数差异影响发布/回读，但本轮没有执行PG，不把推断写成PG FAIL。缺陷表现为恢复保持关闭，未证明错误开放或数据泄漏。

现设计r21的1093–1102/1289/2275以后明确计数属于canonical，且同输入必须复用ref/digest/completed_at；没有充分定义只读Snapshot如何重建某次实际重放的物理计数。旧六个PG用例使用fixedSource或固定1/0的driftReplayer，未覆盖此真实计数接缝；原source单测也未比较完整canonical。这是本次已定位的实现矛盾与测试缺口，不撤销或篡改旧证据。

## 处置、回归与接收

### 当前修复版结果

Dalton九existing及Hegel唯一新增PG文件均已停写。180输入统一固定，manifest SHA256 `18270024690941D7A20294FE78C56696FCFD8B19414B07823ED3E5E8368D2071`。Facts10与真实执行分项分离、first-committed/current-first及唯一历史复用、事务外刷新与owner/current重核、未知提交先读current、合法Trace透传/生成均已实施，公共Ready20/Unavailable6与唯一first16 authority不变。根因三leaf在新PG普通/race中均转GREEN；完整13场景展开16leaf，各24节点全PASS/0skip，32schema全部保留。Dalton单元普通/race各67节点59leaf、四contract各286节点236leaf全通过，Lead逐名核原286项保留；Lead Manifest七项同180全部通过，无源漂移。

Lead已核原始执行/SQL及58份Ready-Facts，独立从行像重算操作/审计摘要；修复证据见[九existing索引](../wip/article-management/evidence-s4.4/repair-nine-r01/index.json)、[新PG代码](../wip/article-management/evidence-s4.4/new-pg-code-r01/index.json)、[401份PG运行与Lead核证](../wip/article-management/evidence-s4.4/pg-execution-r01/index.json)，最后索引实际SHA256 `91C326F7C0E50EA66E18A7854C671F747D481E909AB6021C40821E634EAFACA1`。具体范围与过程纠错见[当前S4.4核证](../wip/article-management/verification-s4.4.md)。这里只证明受控公开Journal下的真实PG链，非真FileJournal/旧备份/系统开流。

Erdos已交回[两轮正式独立CodeReview PASS/0findings](../wip/article-management/review-code-s4.4-issue0001-r01.md)，实际SHA256 `68C412EC34D66272AA4F4783ABC8F6EB4105AB0990E838B2A9AE9736A7F49BAC`；本人新PG普通/race各24节点16leaf、旧六PG安全retained/preflight overlay各7节点6leaf均通过，0fail/skip。本人46schema全部保留，旧六仍受控source/replayer，旧inventory仅最终文件归档，不能外推真实旧备份。Lead另核全部七命令raw/meta、live/snapshot180、本人58Ready-SQL-Facts/366payload及158-Go overlay仅harness一输入变化并核六原断言，技术核收该有界Gate；414份资料见[独立Review索引](../wip/article-management/evidence-s4.4/review-r01/index.json)，实际SHA256 `B6691BAAAA23113EC02A996394B5619B163ECE4371C6FE0A9CB244B6B4D859A4`。

[P1相称RCA与材料保留](../wip/article-management/rca-issue0001-r01.md)已由实际Steward Kepler完成并停写；未作Host材料清理、规则采用或对外发送。人的精确两方接收/关闭决定仍pending，不因技术回归或独立Review通过代签；S4.4完整真实恢复门禁仍保留。以下执行窗口段均为对应阶段历史。

### 最新真实PG失败及修复窗口

Dalton首阶段已停写，新增测试文件固定 `B71D378C91758BA4179E0C9103EC70D25CEDE2D9BF70BC0340D94C1EB2AD412B`，原179项逐项未变。实际 `AllAbsent0N` 由真实Finalizer预完成ordinary/violating/orphan三条，再返回真实0/3；PG中Article为0、operation均completed、audit为3、evidence为0，公开结果却为unknown/evidence_stale。`AllPendingN0/ExistingTrace` 实际context_trace_mismatch，`GeneratedTrace` 实际invalid_trace_id；两者Article仍3/audit与evidence均0、首条claim合法独立提交。两条普通命令分别exit1，合计8节点/3leaf/0skip；gofmt/list成功且唯一top，未运行race或修复版。

Lead实际解析全部raw/meta，核三schema的保留SQL像/observer不同backend/锁可用/资源acquired0，核180输入、每次完整snapshot及overlay/raw哈希；66份资料及180项可复核源码绑定见[old-API真实PG RED索引](../wip/article-management/evidence-s4.4/oldapi-red-r01/index.json)，SHA256 `5EC14AFB680F8AF2985B9B57E0A15BDB0A23047F7C714E14BE566D02AEC719B3`。schema全部保留，未清理或改库权限。已有Run状态及claim是实际失败结果，不按“全回滚”美化。

据此Lead已授Dalton独占r20所列四生产/五既有测试最小修复与相关单元/四contract普通-race回归；Hegel独占新 `article_replay_test.go`，从已跑固定前像补齐13场景和必需嵌套。两者文件互斥，新PG的完整执行须内部端口固定及双方停写后统一冻结180输入；Hegel现仅CODE/format，尚无完整PG执行结果。最终独立CodeReview由非作者Erdos准备，未发审查结论。Tasks/B8A首阶段登记及其余58卡保护见[九份metadata核证](../wip/article-management/evidence-s4.4/start-metadata-r01/index.json)。下方初始授权与尚未PG的措辞保留为对应时点，不代替本段现状。

当前执行依据为[测试规划r20 §4.8.1](../../.add/project/knowledge/design-workspace/article-management/test-plan.md)，SHA256 `DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA`；[Pauli独立TestPlanReview](../wip/article-management/review-test-plan-r20-s4.4-r01.md) SHA256 `1F5FBF2FC190AA7429C79441FD88CEADC35F52C7AAC1E0166041B3BB7B6E4402`两轮有界PASS、零findings，仅证明规划。Lead已读全文并独立核全部申报改动、197条原表格行及179项源码无变化；[八份规划材料](../wip/article-management/evidence-s4.4/testplan-r20-r01/index.json)及[四份独立Review材料](../wip/article-management/evidence-s4.4/testplan-review-r20-r01/index.json)保留。Dalton已获第一阶段CODE+LOCALPG：仅新增原S4.4 `article_replay_test.go`，先按旧API独立记录absent计数和两种Trace真实失败，原179项保护。真实RED核证后才开启四生产/五既有测试修复；Hegel（实际Backend-Dev，`01a17f82-6dce-7531-967b-0ccadd3ec777`）目前仅Temp准备余下场景，未接收仓库文件写权。不授部署、清理或身份/共享root写入。下文旧“正在规划/未授窗口”均为其记录时点。

Lead已实际读取并采用原Hubble的[发布时绑定咨询](../wip/article-management/evidence-s4.4/metadata-consult-r01/designer/consult.md)，SHA256 `E94FD56824D9C7854481946363B7A4A89407F598A7B8ED9B8E4D631FEB5D9CAF`：真实分项在现有immutable evidence首次成功发布时固定；Snapshot只重读事实；后续current及历史合法复用以原持久分项和新鲜事实核完整canonical。未发布前完整重试可如实改变物理分项，不跨轮累加。无新表、列或第二authority，保留Ready20、fixture、构造与identity/root边界；真实漂移、损坏与歧义继续fail closed，禁止清零/改报计数、ReadStatus再Replay或跳过摘要检查。

当前只向Hubble发出了唯一design.md的r21→r22写窗口，submission `01a17f61-f913-7ab2-930f-4afdfd197be9`；Pauli只读准备后续独立设计评审，Dalton仅准备实施G0，未授生产代码、测试或PG执行。正式r22及测试规划固定、范围核准后才进入remedy。设计采用来自Lead在用户一般建议值授权下的决定，不代签人的精确版本accepted；consult、Task准备前后像与实际采用记录见[13份metadata/咨询索引](../wip/article-management/evidence-s4.4/metadata-consult-r01/index.json)。

上述为设计START时点。当前[design r22](../../.add/project/knowledge/design-workspace/article-management/design.md)已固定SHA256 `8C6FEAEFDDC1234B550D8AFCD0E3B45B53527A5F688426CF93798724AA39F66E`（+115/-61），[Pauli独立DesignReview R01](../wip/article-management/review-design-r22-replay-counters-r01.md) SHA256 `74BB3F80998A08EA917347DD68D042CB995BAF576CD3EA9B9203300A58FD99A1`两轮有界PASS、零新增设计findings。Lead已实际读新增核心、diff全部受影响部分、完整报告与G0方案，并核179项源码未改；17份设计前像/diff/独立Review/G0资料见[design-g0索引](../wip/article-management/evidence-s4.4/design-g0-r01/index.json)，SHA256 `8014244827D21AC01E6C209ACE5EBBBFEA8BCB27CC538B43A09EC6FA9230DDEC`。测试规划r20正在唯一文件中同步，尚未发生产代码/PG窗口；Dalton仅获Temp中准备old-API首批RED代码和runner，未落仓/编译/执行。设计PASS不等于实现、回归或人的精确版本accepted。

G0及独立设计评审另静态确认同Source.go的Trace接缝：固定 `article-deletion-replay` 长23，平台要求合法26位；真实all-pending链可能先被Finalizer的context/显式trace一致性验证阻断，completed/absent短路看不到它。必须单独保留未来实际RED，再在原Source.go候选范围内按既有Tracing规范透传/生成合法值；不得由测试adapter改写参数、替回假Replayer或放宽校验。当前没有该问题的Go/PG运行证据。

拟回归由原S4.4路径 `test/integration/recovery/article_replay_test.go` 补真实Finalizer→PG事实→evidence→Provider/decoder，覆盖absent/混合/重复、漂移后重发布及实际提交后丢响应；目前仅候选，文件未创建。Journal seam和模拟恢复必须继续与真实Journal/旧备份/联合开流分层。旧库存/CAS结果不重复制造新PASS。

| 关闭依据 | 当前情况 |
|---|---|
| 产出与修复版本、根因回归 | 四生产/五既有测试加一新增PG文件已交付；固定180，作者新PG普通/race各16leaf通过，原三真实RED转GREEN |
| 独立非作者评审/复审 | Erdos两轮有界CodeReview PASS/0findings及本人新16/旧6普通-race完成，Lead另核原始证据；不代完整恢复或人的接收 |
| Lead同版本范围与项目检查 | 已核9＋1范围/170保护项、最终单元及全部原286contract节点、新PG32schema/58Ready-SQL-Facts、Manifest七项；180无漂移 |
| 实际产出/接收责任与关闭决定 | pending；不以AI角色、用户先前一般授权或诊断交回代签结果接受 |
| S4.4及主交付其他门禁 | 不解除真实Journal/旧备份、S8.2/Case18、应用ACL与完整恢复开放义务 |

## P1阶段RCA与时间线

2026-10-28：S4.4准备核到旧PG证据的受控Replayer边界；05:42:17开始首个精确诊断；05:45:58原执行者完成三次一致结果并stopped；Lead随后逐项核raw、六份canonical对、overlay前缀和179项。未启动PG/SMTP/服务，没有清理或Git写入，全部定位材料保留。

暂不使用改变计数的止血方案。根因修复、实际回归效果、独立评审、接收意见及相称复盘在本单追加；候选防再发措施是增加真实Finalizer/PG计数接缝回归，尚未实施，不自动写Rule/Skill或外传经验。定位完成不等于问题关闭。
