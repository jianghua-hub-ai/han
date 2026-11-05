# Recovery PlanningReview R01（四规划＋两采用元数据）

结论：**CHANGES REQUIRED**。新增 **1 项 MEDIUM OPEN：RECOVERY-PLAN-R01-F001**；CRITICAL/HIGH/LOW 均为0。问题是 article S4.8 的本卡必要 P 验收包含后继安装卡才交付的 Store 绑定，造成关闭条件反向等待。显式 requires DAG 无环不消除这一实际验收依赖。

四规划：Tasks 的该验收责任须修订；Plan 的 B13D→B13E 启动/关闭可执行性受同一 finding 影响，无第二 finding。两 TP 仅 mapping/meta 的字节边界核验通过，技术场景保持；两 ADR 文件仅采用登记元数据核验通过。六对象联合交付尚不能 PASS。各局部通过均不代表 Code/Case/运行通过或人的接纳，也不预签返工后的新 SHA。

**归属与范围**

- Reviewer：Kuhn / add-reviewer，runtime ID `01a1835a-8010-7a73-a8dd-3192719b6965`；由 Main 依用户 standing 与项目委派规则指定，ID 来源为 Main 实际 spawn/resume。
- 四规划作者：Goodall / add-task-planner `01a18414-b2ab-7392-9b04-fd974203bf5e`；实际 WRITE START submission `01a1843a-fd83-7872-99ba-243c205915df`，Main 提供真实 spawn/close completed。原 receipt JSON 的 null、Markdown“用户指定”保留原像，不解释为用户亲指定角色或人签字。
- 两 metadata 作者：Pascal / add-arch-designer，作者 receipt 引用 Main 提供的原实际 ID `01a17e3d-5550-7f91-bb7e-6e6fad3d082c`，本 Reviewer 未另行 spawn 或委派。
- 固定 `D:\AID\VSCode\Codex\Han` / main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。objectType=design（任务、验证与批次规划），ADR 两项仅登记元数据，不技术重审。Role/add-code-review及必要 Rules/Profile 同版完整正文复用，8项哈希复核相同；未加载其他 Role/未来 Skill。
- 本次正式 START 后核四规划冻稿；两 live ADR 在 Main 后续提供 STOP 和两个最终 SHA 前未读/hash，补齐后才纳入本次审查。共享全只读，只在本独占 Temp 写报告与读取/验证记录。

**六对象 SHA256（路径前缀 `.add/project/knowledge/`）**

| 对象 | Review前 | Review后 |
|---|---|---|
| tasks/tasks.md | `0F197F4586195FEC2340DF074C00CAB6D7EBD6C5AA196F11BF585DDB867A7B08` | `0F197F4586195FEC2340DF074C00CAB6D7EBD6C5AA196F11BF585DDB867A7B08` |
| plans/ir-01-content-community-batches.md | `1B2540B5F59756FF3D4B397B90FE44E6111700DCA8F8538A75BE863B99CCF9C7` | `1B2540B5F59756FF3D4B397B90FE44E6111700DCA8F8538A75BE863B99CCF9C7` |
| design-workspace/identity-auth/test-plan.md r24 | `468864D2FBA7281CABDCA0C771D56E1316BAFAE8B0F9B524269435E544BDF41D` | `468864D2FBA7281CABDCA0C771D56E1316BAFAE8B0F9B524269435E544BDF41D` |
| design-workspace/article-management/test-plan.md r21 | `E2E7ADFD0581BCCE9CD2782F7181C12BA67A8496A1A321EF88006EA94D9FCA27` | `E2E7ADFD0581BCCE9CD2782F7181C12BA67A8496A1A321EF88006EA94D9FCA27` |
| decisions/framework.md | `902D6F066BE6B0F728C9946E1EE83B69608A92F37FF80C929431D7267A47B4F8` | `902D6F066BE6B0F728C9946E1EE83B69608A92F37FF80C929431D7267A47B4F8` |
| decisions/README.md | `7112FA46EF75F8130AA900CF46D5532521D9E5002160FB8BBBDB3031FFF6AFB4` | `7112FA46EF75F8130AA900CF46D5532521D9E5002160FB8BBBDB3031FFF6AFB4` |

四基线为 Main `planning-start/before` 中 EC9563 Tasks / 8AA8A2 Plan / E6906617 identity TP / E39C980C article TP；完整64位值与每个 diff 的 SHA 在 static-verification.json。metadata 精确前像是36F4B49F/157A549A。未混用 Tasks 中间稿7FB4或阶段A35548作为最终稿；阶段A实际登记快照与阶段B消费前像逐字节相等，SHA `35548C23265DFBE8FBDCF829FD3FFADC4A8889A6072367C3BA74E55E132E7628`。

**Round 1：完整实际 diff 与所有新增内容**

全文取得 Main `planning-frozen-r01` 的 Tasks 746有效行、Plan306有效行及两个TP各51有效行精确 diff（末尾空行另计），读完14张新卡的文件、requires、命令、风险leaf、最低层、局部关闭/后继责任和9批安排。作者 receipt 全文读、阶段A验证和阶段B consumed 输入定点核对；未执行作者脚本。Phase A 登记→Phase B 消费的实际快照核对不等于重新执行作者规划方法。

独立逐 hunk 正/逆向重建：Tasks +727/-5（六个统计数值位于五条旧行）；原59卡除统计外原字节保留，新节追加。Plan +293/-1，只更新开头当前范围/总数并追加本pass，旧批次/历史不重写。两个 TP 各+35/-0、各两新增块，逆变换逐字节恢复 E690/E39；版本、frontmatter、原场景/层次/历史及 article 原786行正确 oracle 均保持。来源声明区分 Jason 原TP、Goodall mapping/meta、TP R01失败与R02定点关闭，不把旧 Review 扩成当前 PlanningReview通过。

两 meta 的实际 diff 全文读：framework +3/-0，原文件全部 byte prefix 保留；README +1/-1，仅 ADR27计数3→6。完整旧技术正文、原3条采用、旧架构 Review/其他ADR不变。新增记录绑定四规划最终 SHA 和 Jason E690/E39 技术前像，Tasks/Plan为同pass一源。实际来源6=architecture1+design2+独立TP2+联合TasksPlan1；Story/mapping/14卡/9批/咨询/Review不增计。四稿中的36F4/157A及“当前3/后继登记”是形成时点，不要求追新 ledger hash。

**Round 2：规划可执行性、闭环与真实接点**

本轮沿准备阶段全文 gap-map/ownership候选/差量receipt、原59卡/旧批次投影、稳定 TP R01/R02的解释继续；正式阶段定点读 E690 TP790–795及826–833、E39 TP784–788，article design2438–2445/2472–2474与identity design3759–3761。必要真实消费者只读 Coordinator12–77、RecoveryRunStore17–70、DeletionCoordinator94–124、lifecycle66–84/120–145；没有重审整套设计或所有源码。

| 卡/责任组 | 本轮核验结论与适用限制 |
|---|---|
| identity S4.7/S4.8 | 中性判定、全历史失败优先/两namespace ABA、预算/代际与只读/ctx迁移、受控Consume唯一窗口分开；journal.go串行；实际Coordinator/Store/Append接点可由所列文件和ports承接。组件U/C不冒完整root；新符号待实施本身不记缺陷。 |
| identity S4.9 | 唯一Store与一次0010、完整snapshot/counter、NULL/旧行、组合约束/不可变/迁移和P载体明确；后续业务卡消费不并改migration。旧0005/0008权威不迁移。 |
| identity S4.10/S4.11/S4.12 | 身份终结、canonical/review、F001维护Session与G+1原子退出、F002可信角色/旧unresolved/单次发送/无Mailer driver/完整结算逐卡有P载体与top、必要风险leaf；ordinary拒绝/unknown保护不放宽，root认证切换R留S4.16。文件不重叠，不借旧S10.5关闭授权新写权。 |
| article S4.7/S4.8与identity S4.13 | 锚定持久化故障矩阵、pure snapshot、internal startup与recovery committed分支、publication/current/evidence原子/独立17字段、历史与fresh/legacy区分均落实；但S4.8本卡P leaf把后继S4.13的Store补绑定纳为必过，形成下述finding。安装独立P与BindLegacy唯一owner本身正确，不重开上游设计。 |
| article S8.3与identity S4.15 | guard只消费许可，404/405后且Session前；生命周期精确覆盖三worker及Started/Done、own-cause/取消/join与资源顺序。组件闭环可先于root，实际production chain由后继S4.16另核；不把handler单测冒Case18。 |
| identity S4.14 | 本卡明确新增 recovery_restore_integration_test.go，定向 TestRecoveryRestoreTargetReadbackPostgres及普通/tagged race/vet；独立target与native receipt读回不是推给专项。稳定artifact handle、private stdin、strict receipt、Begin前置、unknown不重Replace、真实child join均有leaf。实际批准工具/锁/provider unknown仍阻相应真实启用。 |
| identity S4.16/S4.17 | S4.16独占recovery_runtime_integration_test.go，TestRecoveryRuntimeProductionChain本卡P/R包含完整软件成功链、两真实GET503/零下游及恢复后真实Session/普通Auth；S4.17只三软件专项路径，可复用同revision根链证据并另验跨域，不反向阻开发卡。未要求未来top现在存在，也未把软件adapter当production来源。 |

从实际 SSOT独立解析：73/48completed/3active/22todo；原59状态/requires保持，14新卡均[ ]未认领；14卡全且唯一分配9批，无遗漏/同批互依赖或文件交叉。62条显式requires图边与Plan Mermaid逐条相等，图无环。实际两处文件重叠为journal.go及RecoveryRunStore integration，均存在先行依赖串行；11张旧未完成卡仍引用原批次。拟新增路径均不存在、拟改路径均存在，此检查不证明实时无writer。162个新增链接出现次数（含metadata14）及所用显式anchor现场核对，不冒充Main原保护集或其计数口径。

**[MEDIUM / OPEN] RECOVERY-PLAN-R01-F001：发布历史卡的必过P验收依赖后继安装卡，关闭条件反向等待**

- 原作者/位置：Goodall；`.add/project/knowledge/tasks/tasks.md:2377`，article-management-S4.8 的 `LegacyReassertionVersusTrustedImport` 最低P验收要求“同Store一次补NULL”。该卡:2379要求满足本卡全部最低层后关闭。
- 关联证据：同卡:2380又将Store精确绑定交后继安装；identity-auth-S4.13 的:2396 requires明确包含article-management-S4.8，:2425将 BindInstallation/BindLegacyPublication 放到本卡新增 recovery_installation.go。:2421的P leaf已覆盖独立来源一次填NULL。Plan B13D先发布历史，B13E后安装。
- 上游依据：article design:2444的ImportLegacyPublication只返回PublicationBinding；:2474明确article不写RecoveryRun，成功后只有identity Store BindLegacyPublication补原NULL。identity design:3761定义该安装迁移写法/短事务与不改旧结果；TP原787同样分清article导入与identity补绑定。没有修改中央合同的需要。
- 触发：B13D执行S4.8合法legacy导入正控，按本卡mandatory P leaf还须真实Store完成NULL→绑定，才能关闭。但这项业务实现被分配给尚不能启动的后继S4.13；S4.13又等S4.8关闭。该问题来自本次任务责任/验收划分，不是因新接口现在还没实现而报缺陷。
- 后果：显式DAG虽无环，实际关闭流程卡住；若强行关卡，就可能用假的Store替代必要P，或越过article/identity写者与文件边界去补实现。
- 最小修订：S4.8本卡保留真实PG的独立可信导入、来源/immutable/publication约束、原字段保留及RecoveryRun零写；将“同Store一次补NULL”的执行验收明确归S4.13现有LegacyBindingAndForwardOnly P及其原owner。无需新TaskID、反向requires或把Store写权移入article。同步涉及的责任表述即可，保留该全局义务，不删其验证。
- 复审验证：定点读作者精确修复diff，确认S4.8本卡可仅用已完成前置和自身文件获得全部必要P正/负证据；S4.13在S4.8之后用真实同一Store验证一次NULL绑定、非NULL精确幂等/错绑拒绝、counter及旧结果不变。两卡/Plan/TP映射的边界不矛盾，旧门槛不降。如返工改变四SHA，对应metadata原条目须由其owner按新固定输入同步，不新增采用计数。本次不运行PG证明软件已通过。

无新增架构待决定项，无另行非阻断优化建议；没有把保守可用性限制或未知生产provider当缺陷。原TP `RECOVERY-TP-R01-F001`保持R02 CLOSED，ARCH/Design既有finding不重开。

**实际验证、限制与交回**

实际执行仅只读文本/哈希、四diff及两meta的精确正反向byte重建、SSOT与Plan/DAG/所有权/路径/显式链接检查、阶段A消费快照、Role/Rule/Profile哈希、目标与已读输入结束漂移核验。脚本/实际范围/命令见本Temp readset、audit.py与static-verification.json；Get-Content/rg为读取定位，python为独立文档静态验证。Main的373/371保护、staged0、author887/451保护分别按各自归属，不称本Reviewer重跑全保护集。

本地辅助脚本初次dict迭代、统计行/数值口径断言已纠正；并行readset附加发生冲突，原始日志保留，两个缺失的成功读取范围按实际工具输出补记。详情在static-verification工具说明与readset-normalization-note；均非产品/Go测试失败，不改变共享对象或Review事实。

**NOT RUN**：Go/build/test/race/vet、PG/DDL/HTTP/SMTP/服务/worker/restore、真实CAS/POSIX/ACL/备份/target/审批/工具/retirement、性能/部署/E2E或人确认；无网络/安装、secret ENV/DSN/key读取、Git写、共享写、清理删除移动归档或子委派。PlantUML未渲染，无渲染PASS。只读源码seam及未来命令设计不代表命令已运行或测试命中。

四设计/Story仍draft，人的成果接纳unknown；Main standing采纳建议与技术接收不是人的签字。原S1.5/S4.4/S8.2与Case18未闭、真实S4.5 CAS/S4.6 retirement门槛不降；已闭S4.2/S10.5不重开；ADR0024/S3.3/S3.5狭窄豁免非PASS，S3.6保持。当前无实施/认领、G4-task或G4-batch通过。

**STOP/frozen**。只交本独立报告给Main现场核收，由Main安排Goodall有界修复；不修改被审稿、不覆盖任何原Review/作者证据、不代风险接受或人的采用决定。
