# B13J D05 架构侧第四轮定向复评（ADR-0030 r08）

## 1. 结论

- **结论标签：`CHANGES REQUIRED`**
- **本轮是否可被架构侧接受为「决策修订已过独立评审」：否。**
- **是否满足约束 9(b) 的登记入口：不满足。** 9(b) 要求本 ADR 的决策修订「给出具体产品、凭据来源、独立故障域论证、部署与 ACL 结论」并已由架构与安全独立评审接受；本轮架构侧结论为 CHANGES REQUIRED，且安全侧独立结论不在本报告。即使修完写集内缺陷，约束 2 正文与 ADR-0032 两项写集外冲突仍须先取得授权修订，否则 9(b) 仍不成立。
- 一句话：r08 关闭了「滞后豁免」「预留无界」与「门禁自相矛盾」，但把见证状态机写成了含不可达态/未覆盖崩溃窗口的模型，并继续保留 ADR-0019 字段与恢复分支的不对齐，以及约束 2、ADR-0032 两项硬冲突；因此不能过架构侧独立评审。

> 本报告只覆盖合同/决策文本层。它不是人的决策接纳，不是设计 Gate，不是安全签收，不是生产批准；不关闭 `pending-B13J-D05`，不解除 `blocked-upstream`，不解锁任何候选 Task，也不声称 Case18/S8.2 通过。

## 2. 范围、基线与实测

### 2.1 对象与基线

| 项 | 值 |
|---|---|
| 对象 | `.add/project/knowledge/decisions/framework.md`，ADR-0030 段 `:915-1052` |
| 修订字段 | `:918` = `adr-0030-r08`（实测） |
| 当前实测 | 221148 B / 1222 行 / SHA256 `7DB51A18F609893A13C49EA8558AC821630412DA29C69B4A102C81EA3E6A76A2` |
| 改前 r07 | 205072 B / 1201 行 / `9F3E5ADC8846074A2A91ACF7B4D25046E35E4DE30789025A2FAECAFE0232D4BD`（Lead 提供，本人未重算 r07 文件） |
| 锚点实测 | `### 约束` = `:1022`；`### 引用记录` = `:1040`；`<a id=adr-0031>` = `:1056`；`<a id=adr-0032>` = `:1146` |

### 2.2 本人实测的分段哈希

按行拼接 CRLF 后以 UTF-8 计算，结果与 Lead 提供的三段逐字节一致：

| 段 | 字节 | SHA256 |
|---|---|---|
| L1-914（ADR-0019 前缀） | 126599 | `8C93C47AE57BED1C8390C07D41210183814185A68B148DCEEF180BDB8AE1C9EB` |
| L1022-1038（约束 1–10 块） | 6018 | `E5C8F61FFEF619CADC91F0CEDF9A237BABAEBCA29A01A01865127DD5C20E37A2` |
| L1053-1222（ADR-0031/0032 后缀） | 39538 | `5A9AA1512F8514568F4CD9B296DF76B06B8A41812723C802483EDCE7A814DA90` |
| L915-1052（ADR-0030 段） | 55011 | `9CFAA9DB70A604F67AF5099B5F97B7A605B2A4A892BCD3FDF7D226321453D16E` |

**结论：ADR-0019 前缀、约束 1–10 正文、ADR-0031/0032 后缀均未被 r08 触碰。** 这是约束 1 的字节级前提，但不等于 r08 与 ADR-0019 的语义完全对齐（见 R04-F004/F005）。

### 2.3 格式与仓库实测

- `framework.md`：CRLF 1222 / bare LF 0；无 BOM；无 TAB；控制字符（`\x00-\x08\x0B\x0C\x0E-\x1F`）0；末字节 0x0A（单 CRLF 末尾）。
- `git rev-parse --abbrev-ref HEAD` = `main`；`git rev-parse HEAD` = `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- `git status --porcelain` 行数 = 293（与既定既存改动一致；本人未做任何 Git 写）。
- `git diff --check -- .add/project/knowledge/decisions/framework.md`：exit 0，无输出。
- 负向 oracle `已解除.*首期不提供|首期不提供.*已解除`：0 命中。
- `修订.*adr-0030-r0`：仅 `:918` 一处，r08。
- 现有 r03 评审报告哈希复核：架构侧 `review-arch-r03/report.md` = 31936 B / `9F39DA5D6A53859ED21A59FB6649FD7E42BF081092441DD71065F2011CC8129F`；安全侧 `review-sec-r03/report.md` = 33105 B / `7F7F73B325571EAB5B44DC2F0739C3CDA0624426CB37F520D45A217D3A215579`。与 r08 `:1052` 引用一致。

### 2.4 未读取/未验证

- 未把作者报告 `evidence-b13j-d05/report.md` 当作证据；本报告只以 `framework.md` 正文与本人实测为准。
- 未验证任何产品、实例、环境、ACL/GRANT、TLS、备份链、回退演练、冷启动或双进程互斥。
- 未运行 Go/PG/服务，未做 Git 写，未改任何被评审文件。
- 未做安全评审；安全侧独立结论以安全评审报告为准。

## 3. Q1：九项闭合判定

| ID | 位置 | 判定 | 依据与残留 |
|---|---|---|---|
| R03-F001（滞后例外 vs fail-closed） | `:953/:967/:970/:991/:992/:994/:1007` | **CLOSED** | `:991`「只允许先预留见证、后提交事件」「禁止提交事件后同步推进见证」「不存在任何允许见证落后于已确认 head 的豁免态」；`:1007` 重复同一口径。滞后豁免已删除。崩溃窗口另见 R04-F002。 |
| R03-F002（预留无边界） | `:990/:991/:992/:1007` | **CLOSED** | `reserved_head` 在无预留时 = `last_confirmed_head`，有预留时恒 = `last_confirmed_head + 1`；`:992` 越界即 closed；`:1007` 两态状态机；token 必须匹配同 revision pending。无界/跨多个 revision 路径已排除。 |
| R03-F005（`:958` 门禁自相矛盾） | `:958/:1017` | **CLOSED** | `:958` 已改为「评审门禁 1 条 + 启动前置 (i)–(v)」，并明确二者不可互相替代、评审门禁不以候选 S4.23 产出为前提。残留措辞见 R04-F007（非 blocking）。 |
| R03-F006（见证写入者与凭据路径） | `:989/:1011`、(b)(d) | **CLOSED** | `:989` 两个保管主体、两个秘密存储位置、见证口令第二次独立一次性 fd/stdin；`:1011` 批准记录最小字段与否证条件；`:1013` 明确宿主同时读取两侧凭据属模型外残余风险（按 ADR-0029 处理，不放宽要求）。 |
| R03-F007（见证未排除业务库故障域） | `:967/:988` | **CLOSED** | `:967`/`:988` 明确「同时位于业务库故障域与业务库备份链之外；见证落在业务库或业务库备份链内即否证」。 |
| R03-F008（`MAX(revision)` pending 口径 / `:936` 历史行） | `:951/:983` | **CLOSED** | `:951` 明确 `MAX(revision)` 含 pending/committed/abandoned，仅用于判定 revision 是否被占用，预留值来自见证水位；并明确 r04 表「条件更新或事务」仅历史描述、不作实现依据。 |
| R03-F013（约束 7 映射） | `:953/:1015` | **CLOSED** | `:953` 与 `:1015` 均写入「整实例回退并重启 → missing/rollback；新副本冷启动 → 崩溃恢复；业务库回退 → stale/missing；并发同 revision 写入 → CAS 冲突；token/ACL → 权限失败」。 |
| R03-F011（`:947` 重复 / 表标题强度） | `:936/:945/:947/:949` | **PARTIAL** | `:947` 重复句已去重；`:945`/`:949` 已说明「建议值」只描述落定方式、不是强度降级。但候选 A 行 `:936` 结论列仍只有「不选」，没有「历史不选，r05 目标形态见 (a)」标注；作者在 `:1021` 自认保持 OPEN。 |
| R03-F003 写集内部分（删除 `:996` 虚假一致性） | `:979/:996/:1019` | **PARTIAL** | `:996` 已改为诚实的「已识别跨 ADR 冲突」并引用 `:1202/:1204`。但 `:979` 仍写「与 ADR-0032 的解除顺序对齐，不构成互为前置」，与 `:996`/`:1019` 直接矛盾；这是 r08 重写 `:979` 时未删净的同一虚假断言。见 R04-F006。 |
| R03-F003 跨 ADR 部分 | `:996/:1019`；ADR-0032 `:1202/:1204` | **OPEN** | 互锁未消除：ADR-0032 `:1202` 要求候选 S4.23 完成时 `Exclusive` 离开 `recoveryClosedPort`，`:1204` 要求候选 S4.24b requires 候选 S4.23；ADR-0030 要求 S4.24b 交付前 `Exclusive` 保持 closed。见 R03-F003 与 Q3。 |

**小结：九项中 7 项 CLOSED、2 项 PARTIAL（F011、F003 写集内）、1 项跨 ADR 仍 OPEN（F003 跨 ADR）。** CLOSED 只表示该条 r03 缺陷已按建议修复，不表示整个模型已可实施。

## 4. Q2：r08 是否引入新的模型级缺陷

### 4.1 三条指定路径的排除：确认

- **「预留无界」**：已排除。`:1007` 状态 (2) 要求 `reserved_head == last_confirmed_head + 1`；`:992` 明确 `reserved_head ∉ {last_confirmed_head, last_confirmed_head + 1}` 即 closed。
- **「见证落后合法」**：已排除。`:991`/`:1007` 明确「不存在任何允许见证落后于已确认 head 的豁免态」「向调用方返回成功之前，见证必须已推进到不小于该 revision」。但该排除是过度约束：提交事件后、推进见证前的正常崩溃窗口被状态机判成 closed，见 R04-F002。
- **「以观测 head 抬高见证」**：已排除。`:992` 与 `:1007` 均明文禁止「以观测到的 head 抬高/自愈见证」。

三条路径的排除成立，但状态机本身不完整，不能据此认为模型已闭合。

### 4.2 新缺陷（按严重度）

#### （1）状态 (1) 按字面读法在每次提交/废弃后都不可达（R04-F001，HIGH，blocking）

`:1007` 状态 (1) 写：「无在飞预留——`reserved_head == last_confirmed_head`，token 为空，且 **reserved revision 上不存在 pending 行**」；同段又说「除此以外的任何组合一律 closed」。

但 `:985` 明确「历史事件行不可改写、不可删除」，`:983`/`:1009` 明确 pending/committed/abandoned 是同一 append-only 表中的不同事件类型。任何一次成功提交或废弃在 revision L 都会留下 pending 行（committed/abandoned 行引用它）。结算后 `reserved_head == last_confirmed_head == L`，按字面「reserved revision」就是 L，而 L 上必然存在 pending 行 → 状态 (1) 不成立 → 一律 closed。

也就是说，**按字面读法，第一次 CAS 之后模型没有任何合法状态**。如果作者本意是「下一 revision（L+1）上不存在任何事件行」，则必须写成「next revision」，并明确 L 上的 pending+committed/abandoned 历史是合法的。当前文本没有区分这两种读法，属于模型级歧义，不是措辞小瑕疵。

#### （2）提交事件后、推进见证前的崩溃窗口没有完成规则（R04-F002，HIGH，blocking）

`:991` 规定的顺序是：「adapter 先以条件写把见证从 `(L, L, 空)` 推进到 `(L, L+1, T)`，再以 T 插入 revision `L+1` 的 pending 行与 committed 行，**最后**把见证推进到 `(L+1, L+1, 空)`」。

若进程在「committed 行已落库、见证尚未推进」之间崩溃，重启后见证为 `(L, L+1, T)`，reserved revision L+1 上已有 committed 行。`:1007` 状态 (2) 要求「reserved revision 上不得存在 committed 或 abandoned 行」，`:992` 又规定「reserved revision 上已存在 committed 或 abandoned 行……一律 closed」。于是这个**正常崩溃窗口**被永久判为 closed，且全文没有「已提交但见证未推进」的完成/幂等规则。

这与 ADR-0019 `:468` 的崩溃恢复合同冲突：`:468` 只允许三种确定结果，其中「文件已为 pending 新头时提交 pending」；本场景中 committed 已持久，正确结果是完成发布/推进见证，而不是永久 closed。把可恢复状态写成 closed，既不是 ADR-0019 授权的失败分类，也使「唯一合法形态」在实践中不可用。

#### （3）连续 abandoned 会使 `head ∈ {L−1, L}` 不变量失效（R04-F003，HIGH，blocking）

`:990` 写：「对外可见的 committed head 不写入见证，**恒满足** `head ∈ {last_confirmed_head − 1, last_confirmed_head}`：相等表示最近一次结算为 committed；差 1 表示最近一次结算为 abandoned……其他差值即 closed。」`:992` 又写「见证与 committed head 的差值不在 `{0, 1}` 内……一律 closed」。

反例：committed head = 10；reserve 11 后废弃 → L=11、head=10、差 1（合法）；reserve 12 后再次废弃 → L=12、head=10、差 2 → 按 `:990/:992` 必须 closed。但 `:991` 明确允许「废弃预留……abandoned 计入水位，被烧毁的 revision 因此不阻塞下一次预留」，且没有「连续两次 abandoned 非法」的规则。于是模型一边允许连续废弃，一边用恒等式把第二次废弃判成 closed。

这不是「fail-closed 更安全」可以解释的：该不变量被写成「恒满足」，是一个事实错误的断言；按它执行会在正常的连续崩溃重试下错误关闭 anchor，按它不执行则 head 校验失去依据。根因是见证只持久 `last_confirmed_head`，没有持久 `last_committed_head`，无法区分「最近结算为 abandoned（差 1）」与「连续多次 abandoned（差 ≥2）」。

### 4.3 `abandoned` 与「已结算水位」是否自洽

**不自洽，至少三处：**

1. 上述状态 (1) 字面问题（R04-F001）：废弃后 L 上保留 pending 行，按字面使「无在飞预留」状态非法。
2. 上述连续 abandoned 的差值问题（R04-F003）。
3. `:1009` 把 ADR-0019 的「清除未发布 pending」映射为「追加 abandoned 事件」，这个方向可以接受（append-only 逻辑清除）；但同一映射把「文件仍为旧头」**一律**映射为 abandoned，丢掉了 ADR-0019 `:468` 的「或在可验证临时文件仍存在时完成发布」分支（R04-F005）。因此 abandoned 的适用边界比 ADR-0019 更宽，改变了失败分类。

### 4.4 `UNIQUE(namespace, revision, kind)` + token 外键是否与 ADR-0019 完全对齐

**不完全对齐。**

- **`:450` 必填状态缺失（R04-F004）**：ADR-0019 `:450` 要求 anchor 至少保存 `journal_id`、namespace、epoch/generation、已提交 `head_sequence/head_hash`、CAS revision/token，以及可恢复的 pending 目标（旧头、新头与操作标识）。r08 `:951` 只写「唯一 adapter 只按 namespace 读取快照、已提交 head 与下一次可用 revision」，`:1009` 只写 `UNIQUE(namespace, revision, kind)` + token；没有 journal_id、epoch、head_sequence/head_hash、pending 旧头/新头/操作标识。ADR-0019 `:469` 要求 epoch 不匹配即停服，没有 epoch 字段就无法实现。
- **`:451` expected-revision CAS 的机制被替换**：r08 用「见证预留 + 唯一键 INSERT」实现 CAS；expected revision 由见证 `last_confirmed_head` 承担，而不是在 anchor 侧对 expected revision 做条件比较。这可能等价，但文本没有论证等价性；在约束 1「不改变 ADR-0019 已锁定的 CAS 协议」下，至少应明确「anchor 侧 CAS 条件 = 见证预留 revision 与 anchor 观测水位一致」，否则读者无法判断这是映射还是替换。
- **`:468` 恢复分支缺失（R04-F005）**：如上，旧头 + 可验证临时文件的「完成发布」分支被丢弃。
- **`:467` 两段 CAS 的中间步骤**：ADR-0019 `:467` 的顺序是「CAS anchor 从旧 committed 头进入 pending 新头 → 原子替换 Journal 并同步目录 → CAS pending 为 committed 新头」。r08 `:991` 把中间「原子替换 Journal」省略为「插入 pending/committed」，作为 anchor 侧映射可以理解，但没有说明 Journal 替换与两次插入的相对顺序，读不出与 `:467` 的逐步对应。

**结论：`UNIQUE + token` 解决了「跳过 pending」「换 token 重写」两条路径，但没有完成与 ADR-0019 的字段级/恢复分支级对齐。**

### 4.5 `:979` 仍保留虚假的一致性声明（R04-F006，MEDIUM，blocking）

`:979` 标题仍写「bundle 的 `Exclusive` 端口三段式（**与 ADR-0032 的解除顺序对齐，不构成互为前置**）」。同一 ADR 的 `:996` 与 `:1019` 明确写「ADR-0032 现行解除条件第 8 条仍把『`Exclusive` 离开 `recoveryClosedPort`』列为候选 S4.23 完成的一部分……与本规定不一致。这是已识别的跨 ADR 冲突」。两处必有一处错误，且 `:979` 与 ADR-0032 `:1202/:1204` 的实测文本不符。r08 重写了 `:979`，却没有删除这个断言，属于应修未修。

## 5. Q3：两项写集外阻塞的现状与判定

### 5.1 约束 2 正文与「独立 PostgreSQL 实例」形态冲突（R03-F004，OPEN，HIGH，blocking）

- `:1025` 约束 2 正文：「不得以 file、**SQL**、业务 PostgreSQL 表、内存、fake 或任何『环境探测后自动选择』的路径填补生产 anchor」。
- `:951` 目标形态：「独立 PostgreSQL 实例……承载单一 append-only 头事件表」。
- `:956` 解释：「约束 2 所禁的『SQL』，指以业务数据库或通用 SQL 表作为 anchor 的自动 fallback 或隐式来源……本项是显式登记、显式配置、固定指向独立实例的 adapter，不探测环境、不回退业务库、不构造隐式来源，因此不属于被禁路径。**约束 2 正文本轮不改动，本段只是对该用词的固定解释，效力低于约束正文，如有冲突以约束 2 为准。**」

**判定：仍 blocking。** 理由：约束 2 是绑定正文，其枚举把「SQL」与「业务 PostgreSQL 表」并列，字面上覆盖任何 SQL 形态；解释段自己声明「效力低于约束正文、冲突以约束 2 为准」。因此，一个只读约束块的实现者/审计者会判定「独立 PostgreSQL 实例 + 事件表」违反约束 2。**「已在 ADR-0030 内如实标注、待用户授权」只能说明冲突被识别，不能给目标形态提供规范依据。** 只有两条出路：

(a) 取得额外授权，修订约束 2 正文，把「显式登记、独立实例、独立故障域、非 fallback」的 adapter 从「SQL」禁令中排除；或
(b) 撤回「具体形态已定」，把四项结论降为「形态未定」，但这会使 9(b) 的「具体产品」不成立。

二者必居其一，不能靠下位解释并存。

### 5.2 ADR-0032 `:1202/:1204` 与三段式的互锁（R03-F003 跨 ADR，OPEN，HIGH，blocking）

- ADR-0030 `:979`/`:996`：候选 S4.24b 交付前 `Exclusive` 保持 `recoveryClosedPort`；S4.23 只接入已登记但 closed 的端口；S4.24b 交付后唯一来源是 ADR-0032 的 POSIX 文件锁。
- ADR-0032 `:1202`：「候选 S4.23……完成，`Exclusive` 离开 `recoveryClosedPort`」。
- ADR-0032 `:1204`：「候选 S4.24b……**requires 候选 S4.23**」。
- `tasks.md:3093` 要求 S4.23 实现四端口；`:3171` 要求 S4.24b 硬 requires S4.23 且 `Exclusive` 已离开 closed；`:3193` 规定 `Exclusive` 仍在 closed 即停止。

**判定：仍 blocking。** 理由：三段式与 ADR-0032 现行文本构成闭环——S4.23 的完成条件要求 `Exclusive` 离开 closed；`Exclusive` 只有在 S4.24b 交付后才离开 closed；S4.24b 又 requires S4.23。`「更正前以本 ADR 更严的规定为准」`只能阻止提前放开 closed，不能为 S4.23 提供完成路径；它把矛盾变成「S4.23 可以启动但永远不能完成」。这不是「已标注的跨 ADR 冲突」可以接受的执行状态，因为 9(b) 要解锁的正是 S4.23 一类实施/验证卡，而该卡在当前文本下不可完成。必须取得授权修订 ADR-0032 `:1202/:1204`（把「`Exclusive` 离开 closed」移到 S4.24b 节点，或取消 S4.24b requires S4.23 的硬边并重排），并同步修订 `tasks.md` 卡面。

### 5.3 「如实标注、待授权」是否足以让本轮过架构评审

**不足以，仍判 blocking。** 三点理由：

1. **规范效力**：约束 2 是绑定正文，解释段明确低于正文；「已标注」不改变「目标形态被自己的约束禁止」这一事实。架构评审不能接受一个需要读者先违反约束正文才能实施的决定。
2. **可执行性**：ADR-0032 的互锁使 9(b) 要解锁的卡不可完成；「更严者优先」只是防提前放开，不是完成路径。接受本轮等于认证一个不可执行的次序。
3. **9(b) 的语义**：9(b) 的入口是「决策修订已由架构与安全独立评审接受」。如果架构侧明知两项硬冲突未解决仍接受，就是把「待授权」伪装成「已接受」，与本 ADR 自己反复强调的「不得以措辞冒充已论证、已满足」相悖。

**正确处置**：把两项记录为「已识别、待用户授权的跨 ADR/约束修订」，保持 9(b) 不满足，不宣称评审门禁已过；Lead 取得授权并完成修订后，再提交定向复评。

## 6. Q4：最小可关闭集

只列真正阻断 9(b) 的条目。分两类：

### 6.1 ADR-0030 写集内（作者可改，无需额外授权）

| 顺序 | ID | 问题 | 最小改法 |
|---|---|---|---|
| 1 | R04-F001 | `:1007` 状态 (1) 字面不可达 | 把状态 (1) 改为「下一 revision `last_confirmed_head + 1` 上不存在任何事件行」，明确 L 上的 pending+committed/abandoned 历史合法；补测试向量。 |
| 2 | R04-F002 | 提交后、见证推进前的崩溃窗口无完成规则 | 增加「reserved revision 已有同 token committed → 核对后幂等推进见证到 `(L+1,L+1,空)`」；abandoned 同理；token 不符才 closed；补崩溃注入测试。 |
| 3 | R04-F003 | 连续 abandoned 使 `head ∈ {L−1,L}` 失效 | 见证持久 `last_committed_head`（或等价字段），把校验改为 `head == last_committed_head`；或明确禁止连续 abandoned 并说明原因。 |
| 4 | R04-F004 | ADR-0019 `:450` 字段缺失 | 在事件表/读取路径补 `journal_id`、`epoch/generation`、`head_sequence/head_hash`、pending 旧头/新头/操作标识；补 epoch 不匹配失败向量。 |
| 5 | R04-F005 | ADR-0019 `:468`「旧头 + 可验证临时文件 → 完成发布」被丢弃 | 在 `:1009` 三态映射中补该分支：旧头 + 可验证临时文件 → 用同一 token 提交 committed 并推进见证；旧头无临时文件 → abandoned。 |
| 6 | R04-F006 | `:979` 仍断言「与 ADR-0032 对齐、不构成互为前置」 | 删除该括号断言，改为「与 ADR-0032 现行文本不一致，见 `:996`/`:1019`」。 |

### 6.2 写集外（必须由 Lead 取得额外授权）

| 顺序 | ID | 问题 | 需授权的对象 |
|---|---|---|---|
| 7 | R03-F004 | 约束 2 正文与独立 PostgreSQL 形态冲突 | `framework.md` 约束 2 正文（ADR-0030 约束块）；或撤回「具体形态已定」并接受 9(b) 不成立。 |
| 8 | R03-F003 跨 ADR | S4.23 ↔ S4.24b 互锁 | ADR-0032 `:1202/:1204` 正文 + `tasks.md:3093/:3171/:3193` 卡面。 |

### 6.3 非 blocking（可并入同批或后续）

- **R04-F007（MEDIUM）**：`:958`「约束 9(b) 的入口在语义上约束启动、不约束卡面登记」与 9(b) 正文「登记入口」不一致；改为「9(b) 入口约束登记；启动另有 (i)–(v) 前置」。不单独阻断，但应在接受前修正。
- **R04-F008（LOW）**：R03-F011 残留——`:936` 候选 A 行结论列仍只有「不选」，建议加「历史不选；r05 目标形态见 (a)」。
- **F009（规划角色）**：`tasks.md:3057` 仍写死 `adr-0030-r05`，D05 卡按其自身关闭条件 (1) 无法关闭；这是卡面问题，不阻断 ADR 文本的架构判断，但阻断 D05 卡关闭。
- **F010/F012（规划角色）**：`tasks.md` 行号引用漂移、(i)–(v) 无登记载体；不阻断 9(b)，但 S4.23 启动前必须有 owner/载体。
- **说明**：即使上述 8 项全部关闭，9(b) 仍需安全侧独立接受，且 (i)–(v) 仍须由用户/组织作出并登记后才能启动 S4.23。架构评审不能替代这两者。

## 7. Findings

### R03-F003（跨 ADR 部分，HIGH，blocking，OPEN）

- **位置**：`framework.md:996`、`:1019`；ADR-0032 `:1202`、`:1204`；`tasks.md:3093`、`:3171`、`:3193`。
- **现文要点**：ADR-0030 要求 `Exclusive` 在 S4.24b 交付前保持 `recoveryClosedPort`；ADR-0032 要求 S4.23 完成时 `Exclusive` 离开 closed，且 S4.24b requires S4.23。
- **问题**：S4.23 的完成条件与 S4.24b 的启动条件互为前置，形成不可完成环。r08 的「更严者优先」只防止提前放开，不提供完成路径。
- **可执行改法**：取得授权后修订 ADR-0032 `:1202/:1204`（把 `Exclusive` 离开 closed 移到 S4.24b，或取消 `requires S4.23` 硬边并重排），同步修订 `tasks.md` 三处卡面；在此之前不得宣称 9(b) 满足。
- **是否 blocking**：是（且需额外授权）。

### R03-F004（HIGH，blocking，OPEN）

- **位置**：`framework.md:951`、`:956`、`:1025`。
- **现文要点**：目标形态是「独立 PostgreSQL 实例 + append-only 事件表」；约束 2 正文禁止「SQL」；解释段自认效力低于正文、冲突以正文为准。
- **问题**：绑定正文禁止目标形态，下位解释不能提供规范依据；只读约束块的实现者/审计者会判定形态非法。
- **可执行改法**：取得授权修订约束 2 正文，把「显式登记、独立实例、独立故障域、非 fallback」的 adapter 排除出「SQL」禁令；或撤回「具体形态已定」并接受 9(b) 不成立。
- **是否 blocking**：是（且需额外授权）。

### R04-F001（HIGH，blocking）

- **位置**：`framework.md:1007`（状态 (1)），对照 `:985`、`:991`、`:1009`。
- **现文要点**：状态 (1) 要求 `reserved_head == last_confirmed_head` 且「reserved revision 上不存在 pending 行」；其他组合一律 closed。
- **问题**：每次提交/废弃都在 revision L 留下 pending 行（append-only、无 DELETE）；结算后 `reserved_head = L`，按字面状态 (1) 永不成立。若本意是「下一 revision L+1 无事件」，文本没有写明，模型存在不可达态/歧义。
- **可执行改法**：状态 (1) 改为「下一 revision `last_confirmed_head + 1` 上不存在任何事件行」；明确 ≤ L 的 pending+committed/abandoned 历史合法；增加提交后、废弃后两种状态向量。
- **是否 blocking**：是。

### R04-F002（HIGH，blocking）

- **位置**：`framework.md:991`、`:992`、`:1007`；ADR-0019 `:468`。
- **现文要点**：顺序是「预留见证 → 插 pending → 插 committed → 最后推进见证」；状态 (2) 禁止 reserved revision 上存在 committed/abandoned，违者 closed。
- **问题**：committed 已落库、见证未推进时崩溃，重启落入「reserved revision 有 committed」→ 永久 closed；全文无完成/幂等规则。这与 ADR-0019 `:468` 的「文件已为 pending 新头时提交 pending」及三种确定结果冲突。
- **可执行改法**：增加「同 token committed 已存在 → 核对后幂等推进见证」；abandoned 同理；token 不符才 closed；补崩溃注入测试。
- **是否 blocking**：是。

### R04-F003（HIGH，blocking）

- **位置**：`framework.md:990`、`:992`、`:991`。
- **现文要点**：`head ∈ {last_confirmed_head − 1, last_confirmed_head}`「恒满足」；差值不在 `{0,1}` 即 closed；abandoned 计入水位且「不阻塞下一次预留」。
- **问题**：连续两次 abandoned 使差值为 2（例：head=10，L=12），按文本必须 closed，与「不阻塞下一次预留」矛盾；「恒满足」是事实错误。
- **可执行改法**：见证持久 `last_committed_head`（或等价字段），校验 `head == last_committed_head`；或明确「连续 abandoned 非法」并给出理由与恢复路径。
- **是否 blocking**：是。

### R04-F004（HIGH，blocking）

- **位置**：`framework.md:951`、`:1009`；ADR-0019 `:450`、`:469`；约束 1 `:1024`。
- **现文要点**：事件表只描述 `UNIQUE(namespace, revision, kind)`、token、pending/committed/abandoned；读取路径只按 namespace 读快照/head/下一次 revision。
- **问题**：ADR-0019 `:450` 要求的 `journal_id`、epoch/generation、head_sequence/head_hash、pending 旧头/新头/操作标识均未出现；`:469` 的 epoch 不匹配停服无法实现。约束 1 禁止改变 ADR-0019 的 CAS 协议/失败分类。
- **可执行改法**：在事件表 schema 与读取路径中补齐上述字段（或明确其规范 payload 结构）；补 epoch 不匹配失败向量；说明 anchor 侧 expected-revision 条件如何由见证水位实现。
- **是否 blocking**：是。

### R04-F005（HIGH，blocking）

- **位置**：`framework.md:1009`；ADR-0019 `:468`。
- **现文要点**：三态恢复映射为「文件仍为旧头 → abandoned；文件已为新头 → committed；既非旧头也非新头 → closed」。
- **问题**：ADR-0019 `:468` 还有「或在可验证临时文件仍存在时完成发布」分支；r08 把旧头场景一律映射为 abandoned，改变了崩溃恢复结果与失败分类，违反约束 1。
- **可执行改法**：补「旧头 + 可验证临时文件 → 用同一 token 提交 committed 并推进见证」；只有旧头且无临时文件才 abandoned；补对应测试向量。
- **是否 blocking**：是。

### R04-F006（MEDIUM，blocking）

- **位置**：`framework.md:979`，对照 `:996`、`:1019`；ADR-0032 `:1202`、`:1204`。
- **现文要点**：`:979` 标题断言「与 ADR-0032 的解除顺序对齐，不构成互为前置」；`:996`/`:1019` 断言「已识别的跨 ADR 冲突」。
- **问题**：同一 ADR 同时断言「对齐」与「冲突」；` :979` 与 ADR-0032 实测文本不符，r08 重写该行时未删净。
- **可执行改法**：删除括号断言，改为「与 ADR-0032 现行文本不一致，见 `:996`/`:1019`」。
- **是否 blocking**：是（文本一致性；修复成本低）。

### R04-F007（MEDIUM，非 blocking）

- **位置**：`framework.md:958`，对照 `:1037`、`:1017`。
- **现文要点**：`:958` 写「约束 9(b) 的入口在语义上约束启动、不约束卡面登记」；9(b) 正文写「登记入口是……」。
- **问题**：两级门禁本身已自洽，但这句话与 9(b) 的「登记入口」字面矛盾；`:1017` 只说解释不改正文，未消除该句。
- **可执行改法**：改为「9(b) 的入口约束登记；启动另有 (i)–(v) 前置；二者不可互相替代」。
- **是否 blocking**：否（应在接受前修正，但不单独阻断）。

### R04-F008（LOW，非 blocking）

- **位置**：`framework.md:936`、`:945`、`:947`、`:949`。
- **现文要点**：候选 A 行结论仍只有「不选」；标题仍含「建议值」但已解释只描述落定方式；`:947` 重复句已去重。
- **问题**：A 行可能被读成永久排除；「建议值」视觉上仍可能被读成强度降级。作者在 `:1021` 自认 R03-F011 保持 OPEN。
- **可执行改法**：A 行加「历史不选；r05 目标形态见 (a)」；或把「建议值」解释移到表下脚注。
- **是否 blocking**：否。

## 8. 未做的事与边界

- **未验证任何产品/环境/ACL/TLS/备份/演练**：没有连接任何 PostgreSQL/anchor 实例，没有取证 GRANT/ACL、TLS、备份链、回退演练、冷启动或双进程互斥。
- **未做人的接纳、未做生产批准、未做安全评审**：本报告只是架构侧非作者评审意见；安全侧以独立评审报告为准。
- **未改任何被评审文件**：本报告是唯一新增文件；未做 Git 写；未删除/移动文件。
- **不构成**：设计 Gate、安全签收、人的决策接受、生产可用声明；不关闭 `pending-B13J-D05`；不解除 `blocked-upstream`；不声称 Case18/S8.2 通过。
- **未覆盖**：`identity-auth/design.md` 的 r60 设计文本、`tasks.md` 的卡面正确性（除互锁三处引用外）、作者报告全文、安全侧结论。

## 9. 附录：实测命令与输出

```text
git rev-parse --abbrev-ref HEAD            -> main
git rev-parse HEAD                         -> 6da8abbceb0fa197364810a92a8fa2d97b38ddca
git status --porcelain | Measure-Object    -> 293
git diff --check -- .add/project/knowledge/decisions/framework.md -> exit 0（无输出）
oracle '已解除.*首期不提供|首期不提供.*已解除' -> 0 命中
'修订.*adr-0030-r0'                        -> 仅 :918（adr-0030-r08）
framework.md                               -> 221148 B / 1222 行 / CRLF 1222 / bare LF 0 / 无 BOM / TAB 0 / 控制符 0 / 末字节 0x0A
```

分段哈希（按行 CRLF 拼接，UTF-8）：

```text
L1-914     126599  8C93C47AE57BED1C8390C07D41210183814185A68B148DCEEF180BDB8AE1C9EB
L1022-1038   6018  E5C8F61FFEF619CADC91F0CEDF9A237BABAEBCA29A01A01865127DD5C20E37A2
L1053-1222  39538  5A9AA1512F8514568F4CD9B296DF76B06B8A41812723C802483EDCE7A814DA90
L915-1052   55011  9CFAA9DB70A604F67AF5099B5F97B7A605B2A4A892BCD3FDF7D226321453D16E
```
