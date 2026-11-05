# B13J D05 安全侧复评 r04（review-sec-r04）

- 角色：add-reviewer（安全侧，非作者独立评审）
- 对象：`.add/project/knowledge/decisions/framework.md`，ADR-0030 段，修订 `adr-0030-r08`（`:918`）
- 作者：add-arch-designer（Agent Singer）；本报告不代作者修改任何制品
- 性质：定向复评——只判 r03 九项 finding 的闭合状态、r08 是否引入新的安全缺口，以及 9(b) 是否可被安全侧接受；不做全量重评
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 日期：2026-11-02

## 0. 结论摘要

**结论：CHANGES REQUIRED**（最高严重级 HIGH）。

- **9(b) 是否满足：不满足。** 直接原因有三类：约束 2 正文与「独立 PostgreSQL 实例」的字面冲突未解决（R04-F004）；r08 的见证状态机存在两处自相矛盾，按字面不可实施（R04-F002、R04-F003）；CAS 事件表仍允许以 `INSERT` 追加 `abandoned` 把已提交 revision 从派生 head 中抹掉（R04-F001）。此外，正常崩溃窗口的恢复映射不完整（R04-F005）。
- **blocking 清单：** `B13J-D05-SEC-R04-F001`、`F002`、`F003`、`F004`、`F005`。
- **S4.23 启动门禁的安全侧判定：不满足。** 即使仅按安全侧口径，也不得派发/认领/启动 S4.23；除上述 blocking 外，还需 `:958` 的 (i)–(v) 由用户/组织登记，且 S4.23 卡面须承接见证状态机（R04-F008）。
- **本轮可接受性：不可被安全侧接受为 9(b) 的接受证据。** 本报告不是实施授权、不关闭 `pending-B13J-D05`、不解除 `blocked-upstream`、不是人的成果接收或生产批准。

## 1. 实测基线（本轮独立复算）

| 项 | 实测值 | 说明 |
|---|---|---|
| `framework.md` | 221148 B / 1222 行 / `7DB51A18F609893A13C49EA8558AC821630412DA29C69B4A102C81EA3E6A76A2` | 纯 CRLF 1222、无 BOM（首字节 0x23 `#`）、末尾单 CRLF、TAB 0、控制符 0 |
| 前缀 L1-914 | 126599 B / `8C93C47AE57BED1C8390C07D41210183814185A68B148DCEEF180BDB8AE1C9EB` | 与 Lead 声明一致 |
| 约束 1–10 块 L1022-1038 | 6018 B / `E5C8F61FFEF619CADC91F0CEDF9A237BABAEBCA29A01A01865127DD5C20E37A2` | 与 Lead 声明一致；约束正文未被本轮改写 |
| 后缀 L1053-1222 | 39538 B / `5A9AA1512F8514568F4CD9B296DF76B06B8A41812723C802483EDCE7A814DA90` | 与 Lead 声明一致；ADR-0031/0032 未被本轮改写 |
| Lead 裁决载体 | 5968 B / 68 行 / `1C2A63A5CD3C5A6664522D871D0BA9B4EE65AA78D0FC8C19FBDC9B53C7617036` | 回应 R03-F009；内容已读，见 §3 |
| `tasks.md` | 857169 B / 3226 行 / `B5AC1D1D5A57A94872511ABC6B7F2F73DFBD22FB37EC5D460B5D5FDADC1EBA4F` | S4.23 卡在 `:3091` 起 |
| 作者报告 | 74023 B / 697 行 / `ADFB0500B7CDC2C6C3C0316B6F33622F5C55D2F7F9344EF77E208F856E16A74F` | 仅作对象基线；其数值不作为本报告证据 |
| 安全侧 r03 | 33105 B / `7F7F73B325571EAB5B44DC2F0739C3CDA0624426CB37F520D45A217D3A215579` | 承接基线 |
| 架构侧 r03 | 31936 B / `9F39DA5D6A53859ED21A59FB6649FD7E42BF081092441DD71065F2011CC8129F` | 仅作跨侧对照 |

另实测：`framework.md:918` 为 `**修订**: adr-0030-r08`；`:951`–`:954` 为 (a)–(d) 表；`:983`–`:1009` 为 r07/r08 单一写入模型与状态机；`:1022` 为 `### 约束`；`:1040` 为 `### 引用记录`；`:1056` 为 `<a id=adr-0031>`；`:1146` 为 `<a id=adr-0032>`。以上与 Lead 给出的锚点一致。

## 2. 检查范围与证据性质

- **实测：** 文件字节、行数、换行、BOM、分段哈希、行号定位、`rg` 文本检索、`tasks.md` 基线哈希。
- **读文判断：** 攻击链、状态机可实施性、与 ADR-0019/0031/0032 的语义一致性、与 S4.23 卡面的可承接性。
- **未做：** 未启动任何产品、数据库实例、TLS 端点、备份/恢复链或演练；未验证任何 ACL/GRANT、凭据保管、见证产品能力；未运行 Go 测试、PG 或服务；未做人的接纳或生产批准。
- **边界：** 本报告只判「ADR-0030 r08 能否作为 9(b) 的安全侧接受对象」，不判实现、不判卡启动、不判生产可用。

## 3. Q1：r03 九项 finding 的逐条闭合判定

| r03 ID | r03 问题 | 判定 | 行锚与依据 |
|---|---|---|---|
| `R03-F001` | 「至多 1 个 revision 检测滞后」使整实例回退不可检测 | **CLOSED** | `:991` 明确「只允许先预留见证、后提交事件」，禁止「提交事件后同步推进见证」，并写明「不存在任何允许见证落后于已确认 head 的豁免态」；`:992` 把「观测水位低于 `last_confirmed_head`」列为 closed；`:1007` 再次固定唯一状态机；`:970(ii)`/`:994`/`:1015` 的四档演练含「恰为见证 −1」。原始滞后分支已被删除。注意：r08 另引入 terminal 回退通道，见 R04-F001，不属本条原缺陷。 |
| `R03-F002` | `(namespace, revision)` 唯一键与 pending/committed 不可共存，且与 ADR-0019 两段 CAS 冲突 | **PARTIAL** | `:951`/`:952`/`:983`/`:985`/`:1009` 已改为 `UNIQUE(namespace, revision, kind)` + `pending_token` + committed/abandoned 引用同 revision pending，原始「无法同时表达 pending 与 committed」的冲突已解；ADR-0019 `:450`/`:451`/`:467`/`:468` 的两段 CAS 也被映射为「pending 行落库 → committed 行携带同一 token 落库」。但 `UNIQUE(namespace, revision, kind)` 同时允许同一 revision 各有一行 committed 与 abandoned，形成新的 terminal 共存漏洞（R04-F001），故「同一 revision 不可重写」的总体属性未闭合。另：文本写「外键/部分唯一约束」，但 PostgreSQL 的部分唯一索引不能直接作为外键目标；具体 DDL 仍未固定（见 R04-F001 的改法）。 |
| `R03-F003` | 见证形态与保证等级未定、未进入 `:958` 缺项 | **CLOSED** | `:958(v)` 已把「见证形态、owner 与保证等级取证——单调不可递减且无重置接口、跨 anchor 故障域持久、anchor 管理员不能经任一备份/快照/恢复路径降低见证（或保留期 ≥ anchor 备份保留期）、每次推进留可审记录」列为启动前置缺项；`:988` 给出四项保证等级，缺一即 closed。残余的「批准记录最小字段不含这些项」另记 R04-F006。 |
| `R03-F004` | 见证端点与第二凭据通道缺失 | **CLOSED**（ADR 文本层）；卡面 residual 见 R04-F008 | `:952(b)` 已写「见证端点只作为同一配置文件内的 witness 段（地址、来源地址范围、TLS CA 与主机名），不新增环境变量或第三配置键」「见证口令经第二次独立的一次性 fd/stdin 注入」；`:954(d)` 把端点、第二次注入通道、保管主体与单调/保留约束写入部署方责任；`:989` 再次固定「第二次独立的一次性 fd/stdin」与保管主体分离；`:1011` 给出批准记录最小字段。ADR 层通道已定义；S4.23 卡面尚未承接（R04-F008）。 |
| `R03-F005` | 「分属不同保管主体」不可核验 | **PARTIAL** | `:989`/`:1011` 已把两个具名保管主体、两个秘密存储位置、轮换窗口与三条否证条件写入批准记录，文档控制已建立；但没有要求该记录由独立主体出具/审计，也没有技术手段阻止同一账号读取两侧，「注入者与保管主体不同」仍只是文字约定。残余见 R04-F009。 |
| `R03-F006` | 演练未覆盖「回退到与见证相等」的边界档 | **CLOSED** | `:970(ii)`、`:994`、`:1015` 已给出四档：(A) 与当前见证相等（对照，判无回退）、(B) 恰为见证 −1（必须 closed）、(C) 更旧备份（必须 closed）、(D) 新副本冷启动观测水位低于见证（必须 closed）。 |
| `R03-F007` | 「或等价受控通道」是未定义的旁路措辞 | **CLOSED** | 现行 ADR-0030 段内已无「等价受控通道」；唯一残留是 `:983` 对 r06 及更早「append-only 头事件序列或等价外部见证」的「或」作废声明，属于历史澄清而非可执行旁路。 |
| `R03-F008` | (b)–(d) 仍以「建议值」命名，与要求强度不一致 | **CLOSED**（语义层） | `:945` 标题、`:947` 正文、`:949` 表头均明确「建议值」只描述 2026-11-02 的落定方式，不是强度降级、不得读成可选建议；`:947` 并写明自 r06/r07 起 (b)–(d) 为约束性要求。字面未改名，但强度歧义已消除。 |
| `R03-F009` | 「Lead 已裁决」在仓内无可核载体 | **CLOSED** | `lead-adjudication-r03.md` 实测 5968 B / 68 行 / `1C2A63A5CD3C5A6664522D871D0BA9B4EE65AA78D0FC8C19FBDC9B53C7617036`，其 §4.3 明确「登记＝写入卡面与 Plan；启动＝派发/认领并开始改动」，并声明该口径为解释性、效力低于约束正文。载体存在且内容对应 `:958`/`:998` 的引用。 |

**小结：** 7 条 CLOSED、2 条 PARTIAL（`R03-F002`、`R03-F005`）。没有一条可被 r08 的「作者文字处置」直接当作安全侧接受；PARTIAL 的残余均已转成 R04 的新 finding。

## 4. Q2：r08 是否引入新的安全问题（攻击性复核）

### 4.1 R04-F001（HIGH，blocking）：terminal 事件共存使已提交 head 可被 `INSERT` 抹回

**位置：** `framework.md:951`、`:952`、`:983`、`:985`、`:990`、`:1009`。

**现文要点：**
- `:983`：head 由「已提交且非 abandoned 事件」派生；`:985` 声称「历史事件行不可改写、不可删除，因此把最大 revision 行改回旧值不再存在合法路径」。
- `:1009`：事件表唯一键为 `UNIQUE(namespace, revision, kind)`；pending 至多一行；committed/abandoned 必须引用同 revision pending 并携带同一 token。

**问题（攻击者能力假设：持有 anchor 事件表的 `SELECT`/`INSERT`，即应用角色本身；或能代其执行 DML 的 anchor 管理员）：**
1. 正常提交 revision `N`：pending `N`（token `T`）+ committed `N`（token `T`）落库；见证 `(N, N, 空)`；派生 head=`N`；调用方已收到成功。
2. 攻击者用同一 token `T` 追加一行 `abandoned`（`kind='abandoned'`，revision `N`）。`UNIQUE(namespace, revision, kind)` 允许该行，因为该 revision 尚无 abandoned；外键/ token 绑定也满足。
3. head 派生规则「已提交且非 abandoned」把 revision `N` 排除，head 变为 `N−1`。
4. 见证检查：`last_confirmed_head=N`，head=`N−1`，差 1；`:990` 把差 1 解释为「最近一次结算为 abandoned，且该 revision 上必须存在 abandoned 行」——恰好事先插入的 abandoned 行满足该条件。状态机按「下一待预留 revision 无 pending」的可实施读法通过。
5. 结果：已提交的 revision `N` 被静默从 head 链中抹掉，且不需要 `UPDATE`/`DELETE`、不需要写见证；`:985` 的「不再存在合法路径」被证伪。若攻击者还能把 Journal 回退到 `N−1`（同一 DBA/业务库恢复权，正在 `:1013` 的模型内），服务看到的是自洽的旧状态，回退不再被见证发现。

**为什么不是「只影响可用性」：** 这不是把服务打 closed，而是让服务接受一个更旧的 head；它直接破坏 `:953(c)` 的「anchor 头状态本身不可回退」和 R03-F001 想建立的反整实例回退性质。即使把「持有应用凭据者」严格列为模型外（`:1013` 只把宿主被攻陷列为模型外，未把 anchor 管理员的 DML 写清），一个 adapter 缺陷或一次误用的迁移脚本也能制造同样的 terminal 共存，因此不能只靠「实现者不会这么做」。

**可执行改法（文本 + DDL）：**
- 在 `:951`/`:952`/`:983`/`:1009` 明确：同一 revision 的 terminal 事件至多一个。增加部分唯一索引：`CREATE UNIQUE INDEX ... ON anchor_head_events(namespace, revision) WHERE kind IN ('committed','abandoned')`；同时禁止「committed 后再 abandoned」与「abandoned 后再 committed」。
- 把 token 引用约束写成具体可执行的 DDL：pending 侧建立 `UNIQUE(namespace, revision, kind, pending_token)`（或等价的唯一索引）；terminal 侧用复合外键/触发器引用该键，并在部署脚本中一次性创建。文本不能只写「外键/部分唯一约束」。
- 把 head 派生改为「`head = last_committed_head`，且 `(head, last_confirmed_head]` 内每个 revision 必须存在 abandoned 行」；或在见证三元组中增加 `last_committed_head`/`settlement_kind`，使 committed→abandoned 的转换在见证层即可被发现。
- 负向测试：committed `N` 后尝试 abandoned `N`（同 token）必须被 DB 约束拒绝或 closed；abandoned `N` 后尝试 committed `N` 同样必须失败；两行 terminal 共存必须使服务 closed 而非静默忽略。

**是否 blocking：是。** 这是 r08 新引入的、仅用被授予的 `INSERT` 就能触发的反回退缺口。

### 4.2 R04-F002（HIGH，blocking）：状态 (1) 与「pending 行永久保留」自相矛盾

**位置：** `framework.md:1007`（状态 (1)）；关联 `:983`、`:991`、`:1009`。

**现文要点：** `:1007` 的合法状态 (1) 为「无在飞预留——`reserved_head == last_confirmed_head`，token 为空，且 reserved revision 上不存在 pending 行」。

**问题：**
- 该模型没有 `DELETE`，pending 行在 committed/abandoned 之后仍然存在（terminal 行必须引用它）。因此任一已结算 revision `L` 上必然有 pending 行。
- 若把「reserved revision」按状态 (2) 的同名用法理解为 `reserved_head`，则状态 (1) 在 `reserved_head == last_confirmed_head == L` 时要求 revision `L` 上没有 pending 行——与「terminal 必须引用 pending」直接矛盾。结果是：第一次结算之后，任何启动/推进按字面都判 closed，服务永久不可用。
- 若把「reserved revision」改读为「下一待预留 revision `L+1`」，状态 (1) 可通过；但这正是 §4.1 攻击链所依赖的读法。也就是说，r08 的状态机在两种读法下分别得到「不可实施」或「可被 terminal 回退」：两种都不可接受。
- 这不是措辞小事：`:991` 写「启动时与每次推进前一律按本段核对，不符即 closed」，实现者没有第三种解释空间。

**可执行改法：** 把状态 (1) 改写为「无在飞预留——`reserved_head == last_confirmed_head`，token 为空，且下一待预留 revision `last_confirmed_head + 1` 上不存在 pending/committed/abandoned 行」；同时明确「已结算 revision 上的 pending 行是 terminal 行的引用目标，永久保留，不属于 reserved revision」。在 `:1009` 增加一句「pending 行不因结算而删除，终态由 terminal 行表达」。

**是否 blocking：是。** 状态机是 r08 的核心交付；按字面不可实施，按另一种读法则打开 R04-F001。

### 4.3 R04-F003（HIGH，blocking）：连续两次 abandoned 会使 `head ∈ {L−1, L}` 失败，正常重试即可把服务锁死

**位置：** `framework.md:990`、`:991`、`:1007`。

**现文要点：**
- `:990`：「对外可见的 committed head 不写入见证，恒满足 `head ∈ {last_confirmed_head − 1, last_confirmed_head}`……差 1 表示最近一次结算为 abandoned……其他差值即 closed。」
- `:991`：「abandoned 计入水位，被烧毁的 revision 因此不阻塞下一次预留。」

**问题（攻击者能力假设：只需能制造两次 CAS 失败/并发冲突，或正常网络/超时导致两次放弃；不需要任何特权）：**
1. 初始 head=5，见证 `(5,5,空)`。
2. 预留 6 后放弃：见证 `(6,6,空)`，head=5，差 1，允许。
3. 再预留 7 后放弃：见证 `(7,7,空)`，head 仍为 5（没有 committed 6/7），差 2；`:990` 判「其他差值即 closed」。
4. 结果：两次连续失败后服务永久 closed；`:991` 的「不阻塞下一次预留」被证伪。攻击者只要能让两次 CAS 失败（例如反复制造并发冲突/超时），即可造成持久 DoS；正常重试也可能触发。
5. 这不是「差 1 表示最近一次结算为 abandoned」能表达的：该不变量只允许「最近一次结算」是 abandoned，未允许「最近两次或更多次都是 abandoned」。

**可执行改法：** 在见证中持久化 `last_committed_head`（或 `settlement_kind` + `committed_head`），把 head 判据改为 `head == last_committed_head`；再增加「`(head, last_confirmed_head]` 内每个 revision 都必须存在 abandoned 行」的检查。这样连续 abandoned 合法，且 committed→abandoned 的篡改会因 `last_committed_head` 不匹配而 closed。负向测试：连续两次 abandoned 后服务仍可用；committed 后再 abandoned 必须 closed；abandoned 后再 committed 必须 closed。

**是否 blocking：是。** 这是核心状态机的自相矛盾，且可被低成本触发持久不可用。

### 4.4 R04-F005（HIGH，blocking）：崩溃恢复映射未覆盖 r08 新写入次序的两个窗口

**位置：** `framework.md:991`、`:1007`、`:1009`；硬边界 ADR-0019 `:467`、`:468`。

**现文要点：**
- `:991` 的写入次序：见证预留 `(L,L+1,T)` → 插入 pending `L+1` → committed `L+1` → 见证推进 `(L+1,L+1,空)`；成功返回前必须完成最后一步。
- `:1009` 的三态恢复：文件仍为旧头 → 为 pending 追加 abandoned 并推进见证；文件已为新头 → 用同一 token 提交 committed；既非旧头也非新头 → closed。
- `:1007` 状态 (2)：reserved revision 上不得存在 committed 或 abandoned。

**问题（正常崩溃，不需要攻击者）：**
1. **崩溃在预留之后、pending 之前：** 见证 `(L,L+1,T)`，事件表无 pending。`:1009` 说「为该 pending 行追加 abandoned」，但没有 pending 行可引用；若先补插 pending 再 abandoned，文本没有定义这个恢复步骤。按 `:1007`/`:991` 的「不符即 closed」，服务停住。
2. **崩溃在 committed 之后、见证推进之前：** 见证 `(L,L+1,T)`，事件表有 pending + committed（同 token）。`:1007` 状态 (2) 明文判「reserved revision 上存在 committed」为非法，`:992` 也把该情形列为 closed；`:1009` 的「文件已为新头 → 提交 committed」会撞 `UNIQUE(namespace, revision, kind)`（committed 已存在）。文本没有定义「committed 已存在且见证仍为 reserved → 推进见证」的恢复转移。
3. 结果：这是协议自己制造的正常崩溃窗口，却只能人工修库；而应用角色无 `UPDATE`/`DELETE`，人工修复又会破坏 append-only 与 ACL 证明。ADR-0019 `:468` 要求崩溃恢复只允许三种确定结果；r08 的映射没有覆盖新模型的全部中间态。

**可执行改法：** 给出完整的恢复转移表（按见证三元组 × 事件表现状）：
- `(L,L+1,T)` 且无 pending：用 T 插入 pending 后追加 abandoned（或定义「仅见证的 abandon」并说明为何不需要 pending 引用）；推进见证到 `(L+1,L+1,空)`。
- `(L,L+1,T)` 且有 pending、文件旧头：追加 abandoned；推进见证。
- `(L,L+1,T)` 且有 pending、文件新头、无 committed：用 T 插入 committed；推进见证。
- `(L,L+1,T)` 且有 committed（同 T）：**允许**作为「已提交未结算」恢复态，直接把见证推进到 `(L+1,L+1,空)`；这是状态 (2) 的唯一显式例外，必须写入 `:1007`/`:992`，否则实现者只能用 `UPDATE` 或永久 closed。
- 任一 token 不匹配、出现多个 terminal、见证不可读写：closed。
负向/恢复测试：在五个写入步骤各注入一次崩溃，重启后必须得到上述确定结果之一；禁止手工 `UPDATE`。

**是否 blocking：是。** 不是因为它放宽了安全性质，而是因为不修就只能在「永久 closed」与「未授权 `UPDATE`/手工修库」之间二选一，后者会实际破坏 append-only 的安全论证。

### 4.5 R04-F006（MEDIUM，start-gate）：见证形态可能不支持三元组 CAS，批准记录字段也未覆盖保证等级

**位置：** `framework.md:988`、`:989`、`:1011`；关联 `:958(v)`、`:954(d)`。

**现文要点：** `:988` 说「具体形态（例如域外单调计数器、WORM 或审计存储等）由部署方在受支持清单内固定」，四项保证等级缺一即 closed；`:1011` 的批准记录最小字段只有「两个具名保管主体、两个秘密存储位置、轮换窗口是否可合并」。

**问题：**
- r08 的状态机要求见证持久化并原子地比较交换 `(last_confirmed_head, reserved_head, reservation_token)` 三元组（`:990`/`:991`）。一个单纯的「单调计数器」无法存储 `reserved_head` 与 token；一个只追加的 WORM 存储无法把 `(L,L+1,T)` 改回 `(L+1,L+1,空)`；一个审计存储未必提供三元组 CAS。四项保证等级没有一项要求「支持三元组原子条件写/可唯一派生」。
- `:988`/`:954(d)` 要求把见证形态、端点、保证等级写入批准记录，但 `:1011` 的最小字段没有这些项；按 `:1011` 填出的「完整」记录仍不满足 `:958(v)`。这是文本内部不一致，不是实现细节。
- 若部署方选择可递减/可重置的见证，`:988` 四项与 `:953(c)` 仍会判 closed，**兜底存在**；但「受支持清单」由谁维护、如何证明产品满足四项、记录由谁审计，均未定义。

**可执行改法：**
- 在 `:988` 增加第 (5) 项保证等级：「见证形态必须支持三元组的原子条件写（CAS），或能以追加日志唯一派生三元组并在读取时验证；仅单调计数器/不可更新的 WORM 不足以单独承担该三元组」。
- 在 `:1011` 增加批准记录字段：见证产品/形态/大版本与 EOM、端点与来源范围、四项（或五项）保证等级的取证引用、见证保留期 vs anchor 备份保留期、具名 owner、最近一次演练证据、记录出具人与日期。
- 在 `:958(v)` 明确「受支持清单」的 owner 与版本门禁（与 (a) 的版本清单同一责任主体）。

**是否 blocking：否（对 9(b) 文本接受）；是 S4.23 启动前置。** 它不改变反回退性质，但会使 (v) 无法被核验，且可能选到无法实现状态机的产品。

### 4.6 R04-F007（MEDIUM，start-gate）：同一配置文件把 witness 端点与 anchor 参数置于同一写入权威之下

**位置：** `framework.md:952(b)`、`:954(d)`、`:1011`；关联 `:1013` 的威胁模型。

**现文要点：** `:952(b)` 把 witness 端点放进同一 `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 的 witness 段，配置只承载非秘密字段、owner-only、no-follow；见证口令走第二次 fd/stdin；`:1011` 只分离两个秘密的保管主体。

**问题（攻击者能力假设：能改 han-server 配置文件但不必攻陷宿主内核/进程的部署或配置管理主体；若同时持有 anchor 备份/恢复权，即 `:1013` 的「同一 DBA」）：**
1. 配置文件的 owner-only 保护的是「读」，不是「写」；文本没有要求配置文件写入者/所有者与 anchor 实例管理员分属不同主体，也没有要求 witness 端点进入批准记录或在启动时与批准值比对。
2. 该主体可把 witness 段指向自己控制的端点（TLS CA/主机名校验只能证明「连上了某个端点」，不能证明「连上了批准的那个端点」），然后恢复 anchor 旧备份；伪造见证返回高水位，回退检查通过。
3. 两个秘密的「不同保管主体」不能阻止这一点：端点选择是配置权威，不是秘密保管权威。`han-server` 宿主被攻陷虽在 `:1013` 模型外，但「配置管理/部署自动化可改配置」不等于宿主被攻陷；当前文本没有把这条边界写清。

**可执行改法：**
- 明确配置文件写入者/所有者必须与 anchor 实例管理员分属不同主体，或配置变更需双人批准并留审计；把「配置文件 owner/writer」加入 `:1011` 批准记录。
- 把 witness 端点（地址、来源范围、CA/主机名）在批准记录中固定，并在启动时与配置值比对（例如记录配置摘要/签名，或由第二保管主体签发端点白名单）；不一致即 closed。
- 在 `:1013` 威胁模型补一句：模型内对手是否包括「可改服务配置的部署主体」；若不包括，明确标为残余风险，不能同时声称「见证与 anchor 分属不同保管主体」已足以抵抗同一 DBA。

**是否 blocking：否（对 9(b) 文本接受）；是 S4.23/生产启动前置。** 该攻击需要配置写权限，是否在模型内需 Lead 明确；但文本至少应把该权限写进威胁模型与批准记录。

### 4.7 R04-F008（MEDIUM，card-start）：S4.23 卡面未承接见证状态机与配置解析

**位置：** `tasks.md:3091`、`:3093`、`:3096`–`:3101`、`:3103`–`:3113`、`:3115`、`:3116`、`:3120`。

**现文要点：** S4.23 写集为 4 文件：`cmd/han-server/recovery_sources.go`、`recovery_runtime.go`、`main.go`、`recovery_anchor_production_integration_test.go`；验收命令 9 条；负向测试 (a)–(d) 只覆盖四端口/路径/fallback/未配置；关闭条件 7 项不含见证；停止条件只写「若需新增配置键……停止」。

**问题：**
- r08 要求 adapter 在启动与每次推进前核对见证、维护 `(last_confirmed_head, reserved_head, reservation_token)`、在成功前推进见证、执行 terminal 唯一性与崩溃恢复；这些都不在 S4.23 的验收/负向/关闭条件中。实现者按卡面可以合法地只做四端口而不做见证，Reviewer 也没有卡面依据要求补测。
- 配置解析实际在 `cmd/han-server/recovery_config.go`（`rg` 实测 `:21`/`:67`/`:68`/`:159`/`:160`），不在 S4.23 的 4 文件写集；若 adapter 在 `recovery_sources.go` 内自行解析同一配置文件，虽可不改 `recovery_config.go`，但这是新的实现约束，卡面没有写明。停止条件只说「新增配置键」，而 witness 段不是新键，故不会触发停止。
- 结论：ADR 层的通道定义（R03-F004）已闭合，但卡面层没有承接；S4.23 若按现卡启动，无法覆盖 r08 的安全要求。

**可执行改法：** 由规划角色更新 S4.23 卡（不在本报告写权内）：写集明确由 `recovery_sources.go` 解析 witness 段（或把 `recovery_config.go` 加入写集，变成 5 文件）；验收命令增加见证状态机/terminal 唯一性/崩溃恢复/连续 abandoned 的测试；负向 oracle 增加 committed→abandoned、abandoned→committed、reserved 跨 revision、token 跨 namespace、见证不可读/不可写；关闭条件增加「见证状态机与四档演练设计已取证」；停止条件增加「若 ADR 未给出见证端点/第二 fd/保证等级，停止」。

**是否 blocking：否（对 9(b) 文本接受）；是 S4.23 启动前置。**

### 4.8 R04-F009（LOW）：第二次 fd/stdin 的口径与保管主体证明仍不可核

**位置：** `framework.md:952(b)`、`:989`、`:1011`。

**现文要点：** 「见证口令经第二次独立的一次性 fd/stdin 注入，注入者与保管主体必须与 anchor 口令不同」；`:1011` 要求两个具名保管主体、两个秘密存储位置。

**问题：** 文本没有给出 fd 编号/顺序/帧格式，也没有说明进程如何区分两个秘密。若两者都从 stdin 顺序读取，启动脚本可以重复注入同一秘密；「注入者不同」只是文字，无法从运行痕迹证明。该问题不直接打开回退通道，但削弱了「两个保管主体」的可核性（`R03-F005` 的 PARTIAL 残余）。

**可执行改法：** 固定两个不同 fd（例如 3 与 4）或两条独立 pipe；规定启动时分别读取、禁止同一 stdin 顺序读取；批准记录记录两个注入者身份与注入时间；增加测试：只提供 anchor fd 时见证连接必须 closed，两个 fd 内容相同必须 closed（或至少被记录为否证）。

**是否 blocking：否。**

### 4.9 Q2(a) 其余绕过面（跨 namespace、跨 revision、竞态）

- **跨 namespace 复用 token：** 只要 token 校验和引用约束都以 `(namespace, revision, token)` 为键，跨 namespace 复用不成立；`UNIQUE(namespace, revision, kind)` 已把 namespace 纳入键。但文本只说「同一 revision 的 pending 行携带 token」，未明确 token 的作用域；应在 DDL/测试中固定「token 只在同一 namespace+revision 内有效」，并加跨 namespace 负向测试。当前不单独构成 HIGH，但应随 R04-F001 的 DDL 一起固定。
- **预留跨 revision：** `:1007` 状态 (2) 与 `:992` 已把 `reserved_head ∉ {L, L+1}` 判 closed；CLOSED。
- **两个写入者竞态：** 真正的串行化点是见证的原子 CAS `(L,L,空) → (L,L+1,T)`；若见证形态不支持三元组 CAS（R04-F006），两个写入者可能都认为预留成功。`:1009` 的「两个写入者并发为同一 revision 预留并插入 pending，恰一个成功」只在见证 CAS 成立时成立；因此 R04-F006 不只是产品选型问题，也是并发安全的前提。
- **先 abandoned 再同 revision 重写：** `UNIQUE(ns,rev,kind)` 允许 pending + abandoned + committed 三行共存；terminal 互斥（R04-F001 的修法）必须同时禁止 abandoned→committed 与 committed→abandoned。仅禁止一个方向不够。

### 4.10 Q2(b) 「witness 段在同一配置文件内、不新增配置键」是否留旁路

- **秘密旁路：** 不存在。配置只承载非秘密字段（地址、来源范围、TLS CA/主机名），口令走第二次 fd/stdin，`:952(b)` 明确「配置文件出现任何秘密字段即 closed」。这一层是成立的。
- **端点选择旁路：** 存在，见 R04-F007。同一配置文件把 witness 端点与 anchor 参数放在同一写入权威下；文本没有要求配置写入者与 anchor 管理员分离，也没有把 witness 端点钉进批准记录/启动校验。这是「不新增配置键」的代价，不是秘密通道的问题。
- **解析旁路：** witness 段由谁解析未在卡面固定（R04-F008）；若 `recovery_config.go` 不解析、adapter 在 `recovery_sources.go` 里二次打开同一文件，需要明确「只读一次、同一 owner-only/no-follow 检查、失败即 closed」，否则两个解析器可能给出不同视图。

### 4.11 Q2(c) 见证保证等级「由部署方在批准记录内固定」是否仍有兜底

- **有兜底：** `:988` 四项保证等级「缺一即 closed」；`:953(c)` 把「见证可递减、可重置、可被 anchor 管理员经任一备份/恢复路径降低」列为否证；`:958(v)` 把「单调不可递减且无重置接口、跨 anchor 故障域持久、anchor 管理员不能经任一备份/快照/恢复路径降低见证（或保留期 ≥ anchor 备份保留期）、每次推进留可审记录」列为启动前置。若部署方选择可递减/可重置的见证，文本仍会判 closed，不会因为「写进批准记录」而合法化。
- **兜底未闭合之处：** `:1011` 的批准记录最小字段不含这些保证等级/形态/端点/保留期（R04-F006）；「受支持清单」的 owner 与取证方式未定义；「见证形态支持三元组 CAS」未列为保证等级（R04-F006）。因此「有兜底」目前是文字上的，不足以支撑 (v) 的可核验收。

## 5. Q3：约束 2 正文与「独立 PostgreSQL 实例」的冲突

**结论：不能通过 ADR-0030 内的解释性段落 + 显式待授权标注来接受；必须改约束 2 正文，或由新的 ADR 给出显式例外。9(b) 因此不满足。**

依据：
- `framework.md:1025`（约束 2）原文：「**禁止自动 fallback**：不得以 file、SQL、业务 PostgreSQL 表、内存、fake 或任何『环境探测后自动选择』的路径填补生产 anchor……」。「SQL」在字面上覆盖任何 SQL 数据源，包括独立 PostgreSQL 实例；「业务 PostgreSQL 表」只是其中一个子项，不能反推「SQL」仅指业务库。
- `framework.md:956` 的解释段写：「约束 2 所禁的『SQL』，指以业务数据库或通用 SQL 表作为 anchor 的自动 fallback 或隐式来源……本项是显式登记、显式配置、固定指向独立实例的 adapter，不探测环境、不回退业务库、不构造隐式来源，因此不属于被禁路径。约束 2 正文本轮不改动，本段只是对该用词的固定解释，**效力低于约束正文，如有冲突以约束 2 为准**。」
- 该段自己承认效力低于约束正文。一个下位解释不能把约束正文的「SQL」缩窄为「业务库/通用 SQL fallback」；在选择「独立 PostgreSQL 实例」时，字面冲突仍在。作者在报告中把它列为「待授权」是诚实的，但「待授权」意味着决策未完成，不能作为 9(b) 的接受依据。
- Lead 裁决载体 §3.4 与 §4.2 也确认该冲突成立、改约束正文超出 D05 写集；这与本报告的判断一致。架构侧 `R03-F004` 是同一问题的跨侧确认，不是安全侧新发现。

**可接受的修复路径（二选一，均需 Lead/用户授权）：**
1. 修订约束 2 正文：把「SQL」改为「业务数据库/业务 PostgreSQL 表/通用 SQL 表作为自动 fallback 或隐式来源」，并明确「显式登记、显式配置、固定指向独立 PostgreSQL 实例的 adapter 不属于被禁路径」；同时保留「不得环境探测自动选择」「不得回退业务库」「不得 file/fake」。
2. 新增一条 ADR（或 ADR-0030 的新修订）给出显式例外：说明独立 PostgreSQL 实例与约束 2 的关系、边界与否证条件，并由用户/组织正式接纳。仅加解释性段落不够。

在完成上述任一之前，**9(b) 不满足**；S4.23/S4.24a/S4.24b/S4.22 均不得启动。

## 6. Q4：最小可关闭集

| 顺序 | 条目 | 动作 | 是否需改约束 1–10 正文 | 需谁授权 |
|---|---|---|---|---|
| 1 | R04-F001 terminal 互斥 | 文本 + 部署 DDL：terminal 部分唯一索引；禁止 committed↔abandoned 互转；head 判据改为 `last_committed_head` | 否（改 `:951`/`:952`/`:983`/`:1009` 与部署脚本） | 作者 + 部署方 |
| 2 | R04-F002 状态 (1) | 改写为「下一待预留 revision `L+1` 无 pending/committed/abandoned」；明确已结算 revision 的 pending 永久保留 | 否 | 作者 |
| 3 | R04-F003 连续 abandoned | 见证增加 `last_committed_head`（或 `settlement_kind`）；允许 `(head, L]` 全为 abandoned；加连续两次 abandoned 测试 | 否 | 作者 |
| 4 | R04-F004 约束 2 | 改约束 2 正文，或新增 ADR 显式例外 | **是** | **Lead/用户** |
| 5 | R04-F005 崩溃恢复 | 补完整恢复转移表；允许「committed 已存在 + 见证 reserved → 推进见证」；覆盖预留后无 pending 的窗口 | 否 | 作者 |
| 6 | R04-F006 见证形态/记录 | 增加第 (5) 项保证等级（三元组 CAS/可派生）；扩充 `:1011` 批准记录字段 | 否 | 作者 |
| 7 | R04-F007 配置权威 | 要求配置写入者/所有者与 anchor 管理员分离或双人批准；witness 端点进批准记录并在启动时比对 | 否 | 作者 + 部署方 |
| 8 | R04-F008 S4.23 卡面 | 更新写集/验收/负向/关闭/停止条件，承接见证状态机 | 否 | 规划角色 |
| 9 | R04-F009 fd 口径 | 固定 fd 3/4 或独立 pipe；记录注入者；加负向测试 | 否 | 作者 + 部署方 |

**最小 9(b) 关闭集：** 1–5。第 4 项必须改约束正文（或新增 ADR），需 Lead/用户额外授权；其余四项可在 ADR-0030 的 r08 补充段内文本修订，不需动约束 1–10。第 6–9 项是 S4.23 启动/生产前的前置条件，不改变 9(b) 的「决策级接受」判断，但不做则 S4.23 不得启动。

**建议顺序：** 先由作者完成 1/2/3/5 的文本修订 → 由 Lead/用户授权并完成 4 → 架构与安全两路对 r09 复评 → 再由规划角色完成 8 → 最后由用户/组织登记 `:958` 的 (i)–(v) → 才能讨论 S4.23 启动。任何一步都不能用「已登记、未启动」替代。

## 7. Findings 汇总

| ID | 级别 | 位置 | 问题（含攻击者能力假设） | 可执行改法 | blocking |
|---|---|---|---|---|---|
| `B13J-D05-SEC-R04-F001` | HIGH | `framework.md:951`、`:952`、`:983`、`:985`、`:990`、`:1009` | 持 `SELECT`/`INSERT` 的写入者可在 committed `N` 后追加 abandoned `N`（同 token），head 派生忽略该 revision，见证差 1 被解释为「最近结算 abandoned」，已提交 head 被静默抹回 `N−1`；`:985` 的「不再存在合法路径」被证伪 | terminal 部分唯一索引 `(namespace, revision) WHERE kind IN ('committed','abandoned')`；具体 FK/唯一索引 DDL；head 改为 `last_committed_head`；negative tests 双向禁止 terminal 互转 | **是** |
| `B13J-D05-SEC-R04-F002` | HIGH | `framework.md:1007`（状态 (1)）、`:983`、`:991`、`:1009` | 状态 (1) 要求 `reserved revision` 上无 pending，但已结算 revision 的 pending 永久保留；按字面首次结算后永久 closed；按「下一 revision」读法则打开 R04-F001 | 改写为「下一待预留 revision `L+1` 无 pending/committed/abandoned」；明确已结算 pending 永久保留 | **是** |
| `B13J-D05-SEC-R04-F003` | HIGH | `framework.md:990`、`:991`、`:1007` | 连续两次 abandoned 使 head 与 `last_confirmed_head` 差 2，`:990` 判 closed；`:991` 却称「被烧毁的 revision 不阻塞下一次预留」；两次 CAS 失败即可持久 DoS | 见证持久化 `last_committed_head`（或 settlement_kind）；允许 `(head,L]` 全为 abandoned；加连续两次 abandoned 测试 | **是** |
| `B13J-D05-SEC-R04-F004` | HIGH | `framework.md:1025`（约束 2）、`:956` | 约束 2 字面禁「SQL」；独立 PostgreSQL 实例属 SQL；`:956` 解释段自认效力低于约束正文，不能缩窄；9(b) 不满足 | 改约束 2 正文，或新增 ADR 显式例外；需 Lead/用户授权 | **是** |
| `B13J-D05-SEC-R04-F005` | HIGH | `framework.md:991`、`:1007`、`:1009`；ADR-0019 `:467`、`:468` | 正常崩溃窗口（预留后无 pending；committed 后见证未推进）不在三态恢复映射内；后者被状态 (2) 明判 closed，且「提交 committed」撞唯一键；只能永久 closed 或手工 UPDATE | 补恢复转移表；允许 committed 已存在 + 见证 reserved → 推进见证；覆盖预留后无 pending；加五步崩溃注入测试 | **是** |
| `B13J-D05-SEC-R04-F006` | MEDIUM | `framework.md:988`、`:989`、`:1011`；`:958(v)`、`:954(d)` | 状态机要求三元组 CAS，但可选形态「单调计数器/WORM/审计存储」未必支持；四项保证等级不含 CAS 能力；`:1011` 最小字段不含形态/端点/保证等级/保留期，填完仍不满足 (v) | 增加第 (5) 项保证等级（三元组 CAS/可派生）；扩充批准记录字段；定义受支持清单 owner | 否（S4.23 启动前置） |
| `B13J-D05-SEC-R04-F007` | MEDIUM | `framework.md:952(b)`、`:954(d)`、`:1011`、`:1013` | 同一配置文件承载 witness 端点与 anchor 参数；配置写入者/所有者未要求与 anchor 管理员分离；端点未钉进批准记录/启动校验；可改配置 + 恢复 anchor 旧备份 → 伪造见证 | 配置写入者分离或双人批准；端点进批准记录并启动比对；威胁模型写明配置权威是否在模型内 | 否（S4.23/生产启动前置） |
| `B13J-D05-SEC-R04-F008` | MEDIUM | `tasks.md:3091`、`:3093`、`:3096`–`:3101`、`:3103`–`:3113`、`:3115`、`:3116`、`:3120` | S4.23 写集/验收/负向/关闭条件不含见证状态机、terminal 唯一性、崩溃恢复、连续 abandoned、第二 fd；配置解析在 `recovery_config.go`，不在写集；停止条件只提「新增配置键」 | 规划角色更新卡面：写集/验收/负向/关闭/停止条件承接 r08；明确由 `recovery_sources.go` 解析或把 `recovery_config.go` 纳入写集 | 否（对 9(b)）；是 S4.23 启动前置 |
| `B13J-D05-SEC-R04-F009` | LOW | `framework.md:952(b)`、`:989`、`:1011` | 第二次 fd/stdin 无编号/顺序/帧格式；两个秘密可被同一 stdin 顺序注入；「注入者与保管主体不同」不可从运行痕迹证明 | 固定 fd 3/4 或独立 pipe；记录注入者身份/时间；加负向测试 | 否 |

**计数：** 新 finding 8 条（F001、F002、F003、F005、F006、F007、F008、F009）；承接跨侧项 1 条（F004，架构 R03-F004 的安全侧确认）。最高严重级 HIGH；blocking 5 条。

## 8. 实施/验证卡阶段应由安全侧追加的门禁

以下门禁不替代 `:958` 的 (i)–(v)，也不替代人的生产批准；它们是 S4.23/S4.24b/S4.22 启动或关闭时必须由安全侧复核的项：

- **G1 terminal 互斥：** 部署脚本必须创建 terminal 部分唯一索引；集成测试必须证明 committed→abandoned、abandoned→committed 均失败，且不是只靠 adapter 检查。
- **G2 状态机矩阵：** 覆盖状态 (1)/(2)、下一 revision 无 pending、reserved 有/无 pending、token 不匹配、跨 namespace token、reserved 跨 revision、连续两次 abandoned、见证不可读/不可写；每一格必须有 closed/成功 oracle。
- **G3 崩溃注入：** 在「预留、pending、文件替换、committed、见证推进」五个点各注入崩溃；重启必须得到确定恢复结果；禁止 `UPDATE`/手工修库。
- **G4 见证域外性：** 证明 witness 位于 anchor 与业务库/业务库备份链之外；anchor 角色/管理员不能写、不能降低、不能重置；四项（建议五项）保证等级有取证；四档回退演练（A/B/C/D）有原始证据。
- **G5 双凭据与保管：** 两个具名保管主体、两个秘密存储位置、两个 fd/pipe；批准记录含注入者、轮换窗口、配置写入者；同一账号/同一秘密库/同一变更窗口命中即 closed。
- **G6 TLS：** verify-full、TLS ≥1.2、禁 require/prefer/allow/disable、CA/主机名不匹配即 closed、不复用业务库 CA/DSN。
- **G7 ACL/GRANT：** 头事件表仅 `SELECT`/`INSERT`，显式无 `UPDATE`/`DELETE`/DDL/`TRUNCATE`/跨库/超级用户；用 `has_table_privilege`/`information_schema` 取证并做负向测试。
- **G8 迁移/备份边界：** anchor 表不得进入 `migrations/`；`HAN_MIGRATION_DATABASE_DSN` 只指向业务库；服务不调用任何 anchor 备份/恢复/迁移/provision 脚本；ADR-0031 的 restore 子进程不受影响。
- **G9 产品/环境决策：** `:958` (i)–(v) 由用户/组织登记；见证产品/大版本/EOM/owner/支持清单/保留期 vs 备份保留期全部可核。
- **G10 卡面承接：** S4.23 写集/验收/负向/关闭/停止条件已更新并通过规划评审；未更新前不得启动。

## 9. 未验证范围与边界声明

- 未验证任何产品、云服务、PostgreSQL 实例、Vault/KMS、对象存储、WORM、审计存储或见证端点。
- 未验证任何主机/集群、存储卷、备份链、PITR、恢复演练、四档回退、冷启动或网络隔离。
- 未验证任何 ACL/GRANT、角色矩阵、TLS 版本/CA/主机名、秘密注入通道、保管主体分离或配置文件权限。
- 未运行任何 Go 测试、PG、服务、E2E 或演练；本报告的所有安全结论均为**读文判断 + 文本攻击性推理**，不是运行证据。
- 不是人的成果接收、不是独立评审的最终 PASS、不是生产批准；不关闭 `pending-B13J-D05`、不解除 `blocked-upstream`、不登记/启动任何候选卡、不宣称 Case18/S8.2 通过或生产可用。
- 作者报告中的数值与自评只作对象基线，未作为本报告的证据；本报告只采信实测哈希/行号与 ADR/ADR-0019/任务卡原文。

## 10. 实跑命令与结果（真实）

1. `framework.md` 哈希/字节/行：
   - 结果：`7DB51A18F609893A13C49EA8558AC821630412DA29C69B4A102C81EA3E6A76A2` / 221148 B / 1222 行 / CRLF 1222 / 无 BOM（首字节 0x23）/ 末尾 0A。
2. 分段哈希（按 CRLF 重建）：
   - L1-914：126599 B / `8C93C47AE57BED1C8390C07D41210183814185A68B148DCEEF180BDB8AE1C9EB`
   - L1022-1038：6018 B / `E5C8F61FFEF619CADC91F0CEDF9A237BABAEBCA29A01A01865127DD5C20E37A2`
   - L1053-1222：39538 B / `5A9AA1512F8514568F4CD9B296DF76B06B8A41812723C802483EDCE7A814DA90`
3. `lead-adjudication-r03.md`：5968 B / 68 行 / `1C2A63A5CD3C5A6664522D871D0BA9B4EE65AA78D0FC8C19FBDC9B53C7617036`。
4. `tasks.md`：857169 B / 3226 行 / `B5AC1D1D5A57A94872511ABC6B7F2F73DFBD22FB37EC5D460B5D5FDADC1EBA4F`；`rg` 定位 S4.23 卡 `:3091` 起、写集 `:3096`–`:3101`、验收 `:3103`–`:3113`、负向 `:3115`、关闭 `:3116`、停止 `:3120`。
5. 作者报告：74023 B / 697 行 / `ADFB0500B7CDC2C6C3C0316B6F33622F5C55D2F7F9344EF77E208F856E16A74F`（仅对象基线）。
6. 安全/架构 r03：33105 B / `7F7F73B325571EAB5B44DC2F0739C3CDA0624426CB37F520D45A217D3A215579`；31936 B / `9F39DA5D6A53859ED21A59FB6649FD7E42BF081092441DD71065F2011CC8129F`。
7. 读取行区间：`framework.md:915-1055`、`:1002-1021`、`:1023-1030`、`:444-472`；`tasks.md:3089-3125`；`recovery_sources.go:469-492`；`recovery_config.go` 的 `rg` 命中 `:21`/`:67`/`:68`/`:159`/`:160`。
8. 未运行：Go 测试、PG、服务、TLS/ACL/备份/演练；无 Git 写操作；未修改任何被评审文件。

## 11. 格式自检

- 本报告使用中文 Markdown，纯 LF，末尾单个 LF，UTF-8 无 BOM。
- 行号均以 `framework.md` 当前 1222 行版本为基准；作者后续修订若改变行号，以符号锚与引用记录为准。
- 本报告不包含任何实施授权、卡登记、生产批准或人的签收。
