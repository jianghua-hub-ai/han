# identity-auth 设计修订 r66 作者证据报告（B13J／S4.24a）

**对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
**修订号**：`identity-auth-design-ir01-r66`（`:5`、`:38` 双处登记，本端实测）
**工作区**：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
**写集**：`design.md`（r66 块＋`:3603` 一处原地修订）、本报告、`hashes.txt`
**边界**：无 Git 写操作；未删除／移动／归档任何文件；未改 `tasks.md`（含 `:3186`）、`framework.md`、`cmd/**`、`internal/**`、任何测试、任何既有 `review-*/`、`design-revision-r01..r05/`；未运行 Go test／build／vet。

## 0 基线与终值（本端自算）

| 节点 | 字节 | 行 | SHA256 | 说明 |
|---|---|---|---|---|
| 派发基线（r65） | 1,045,111 | 5,464 | `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9` | 派发给出；与 `review-arch-r05` 独立实测同值 |
| r05 作者记录的输出 | 1,045,042 | 5,464 | `D2A6B54B2153858DC6366995C4332A380D9DA0BEDE863BDDC80D51663A3D8BE2` | `design-revision-r05/report.md`／`hashes.txt` 原文；与现行值差 +69 B，即 NEW-R05-02 的未回写漂移 |
| 本次会话开工时（r66 草稿态） | 1,078,886 | 5,476 | `0BA1C6C1B2F4C21BA4BC3FA5F7E3A949040BC01BE00329D485D6CC4DC2FA1847` | 本端实测 |
| **终值（最后一次写盘后）** | **1,079,559** | **5,476** | `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298` | 本端实测 |

**差值**：终值 − 开工草稿 = **+673 B／Δ0 行**；逐项 = `:3603` +468 B、`:4072` +205 B，合计 +673，与整文件字节增量逐字节吻合（其余字节未动）。终值 − r65 派发基线 = **+34,448 B／+12 行**（r66 块 12 行）。

**编码**：无 BOM（首 3 字节 `2D 2D 2D`）；LF 5,476；CR 169（与编辑前一致，混合换行原样保留）；全部替换位于行内、不含行终止符。

**作废声明**：r05 报告与其 `hashes.txt` 的旧数值只作历史，本轮不复用、不回溯修改旧文件；`design-revision-r06` 的终值是唯一可复核基线，且本报告与 `hashes.txt` 均在 `design.md` 最后一次写盘之后生成。

## 1 逐项闭合（派发 12 项）

| # | finding ID | 级别 | 行锚（现行） | 判定 |
|---|---|---|---|---|
| 1 | 架构 `F006`（含 `NEW-R05-01`） | BLOCKER | `:4076`（r66.4）、`:4082` | 设计层闭合；实现未做 |
| 2 | 架构 `F004` | HIGH | `:3603`（原地）、`:4073`（r66.1） | 设计层闭合；ADR 登记子项未闭合 |
| 3 | 架构 `NEW-R05-04` | MEDIUM | `:4076`（r66.4）、`:4073`（r66.1） | 设计层闭合；实现未做 |
| 4 | 架构 `NEW-R05-03` | LOW | `:4079`（r66.7）、`:4072`（r66.0） | 闭合（语义唯一可判） |
| 5 | 架构 `NEW-R05-05` | LOW | `:4072`（r66.0）、`:4080`（r66.8） | 闭合（按现行实测更正） |
| 6 | 架构 `NEW-R05-02` | MEDIUM | `:4080`（r66.8）、本报告、`hashes.txt` | 闭合（终值后生成） |
| 7 | 架构 `F017` | 条件项 | `:4081`（r66.9） | **未闭合**：仅登记，待 Task-Planner |
| 8 | 安全 `SEC-F003`＋`N001` | BLOCKER | `:4074`（r66.2）、`:4062`、`:3990` | 设计层闭合 |
| 9 | 安全 `N011` | HIGH | `:4077`（r66.5） | 设计层闭合；运行面 NOT RUN |
| 10 | 安全 `NEW-S05-01` | HIGH | `:4078`（r66.6）、`:4072` | 设计层闭合；Linux 实测 NOT RUN |
| 11 | 安全 `NEW-S05-02` | MEDIUM | `:4072`（r66.0）、`:4078`（r66.6） | 闭合（唯一优先级） |
| 12 | 安全 `NEW-S05-04` | LOW | `:4077`（r66.5）、`:4079`（r66.7） | 闭合 |

### 1.1 架构 `F006`（BLOCKER）— lease 所有权与全部读取点

- **落点**：`:4076`（r66.4）；不变量复述 `:4082`（r66.10）。
- **修改摘要**：引入按动作选 lease 的**统一访问器** `activeExclusiveLocked()`／`activeExclusive()`，只按封闭枚举 `r.action`（`install｜normal｜verify｜enter｜resume`）返回 `ownedExclusive` 或 `injectedExclusive`，nil 一律 closed；逐点列出全部读取点并规定选中规则：`checkOwnership`（`:1162`，injected 动作 nil → `ErrAdmissionClosed`，owned 动作 nil → `checkExclusive` 的 `ErrAdmissionBinding`）、其调用者 `CheckPhase`（`:1192`／`:1196`）、`ReadPermitBinding`（`:1241`／`:1245`）、`acquireReady`（`:1123`）、`CheckArticleCoordination`（`:293`）、`readControlled` 入口（`admission.go:617`）、`admission.go:648`（贯穿 `recheckProof`）、`activate`（`:917`／`:919`）与经 `:1943` 的 `replay`、`beginObservation`（`:948`／`:966`，只允许 owned）、`f.begin`（`:1601`／`:1614`）、`f.refreshAfter`→`f.phase`（`:1773`）、`confirmSucceeded`（`:1968`）、`recoveryFlow.acquire`（`:1562`／`:1564`／`:1570`，删除 `deps.Exclusive.Acquire` 与 `f.r.exclusive = l`，只读注入值）、`Stop`／`drain`（`:1419`、`:1423`–`:1424`，只关 owned）、`closeObservationLease`（`:1022`–`:1023`）、`installationFlow.acquire`（`:486`／`:488`）与 admission（`:870`／`:876`）写 owned。
- **`enter` 不冲突判据**：`enter` 是 injected 动作，访问器只返回 `injectedExclusive`，永不返回 owned；命中缺失 lease 的唯一可能是装配未注入（`restoreLease == nil`），按 `ErrAdmissionClosed` closed 处理，不会落入「缺失 owned lease」分支；两个分支由封闭枚举＋default → `ErrAdmissionBinding` 保证互斥完备。
- **代码事实核对**：`internal/recovery/runtime.go:1162`、`:1419`、`:1562`–`:1564`、`:1601`、`:1614`、`:1968`、`admission.go:648` 现均读单一 `r.exclusive`，与文本所列读取点逐一对应；`f.enter`（`:1573`–`:1580`）经 `acquire`→`begin`→`phase`，不调用 `beginObservation`，故「`beginObservation` 只允许 owned」不阻断 `enter`。
- **判定**：设计层闭合（BLOCKER 的文本可实施性障碍解除）；实现与测试未做，NOT RUN 不变。

### 1.2 架构 `F004`（HIGH）— 依赖集合的就地修订

- **落点**：`:3603`（r60 段落原地修订）；`:4073`（r66.1）。
- **修改原文摘要**：`:3603` 现有两段——原文保留「唯一 production root 装配清单」「未在本表内登记的依赖不得注入 production root」「新增端口、替换来源或改变归属须先修订本节与对应 ADR」；新增 r66.1 句固定 `func NewRuntime(ctx context.Context, d RuntimeDependencies, restoreLease ExclusiveLease) (Runtime, error)`，写明 `restoreLease` 是 restore 动作在装配阶段对已登记 `exclusive` 端口一次 `Acquire` 的**端口产出值**（r66.4 起由 assembly 构造前取得并注入，`recoveryFlow.acquire` 只校验并读取 `injectedExclusive`、不再自行 `Acquire`），明确「不是新增依赖项、不是新端口、不是可独立配置／替换的依赖」，`RuntimeDependencies` 11 项端口集合未变、不进入本表、不改变依赖归属；该参数为**非可选入参**，`nil` 表示 restore 动作保持 closed，不等于省略参数。
- **ADR-0031 判定**：ADR 全文 `NewRuntime` 出现 **0 次**（本端实测），故无条文冲突；但 F004 复审要求「把该结论登记到 ADR 或指向已接受决策」。本轮**不改 ADR**，该登记子项**保持未闭合**，见第 4 节。
- **判定**：设计文本侧闭合；ADR 登记子项未闭合（升级待决）。

### 1.3 架构 `NEW-R05-04`（MEDIUM）— 实现面映射

- **落点**：`:4076`（r66.4）、`:4073`（r66.1）。
- **修改摘要**：①`RestoreDriver` 保持三方法，明确**不存在也不得新增** `Bind`（含 `driver.Bind` 形式），r65 各节旧接线句一律作废；`rg -n 'func .*Bind\(' --glob '*.go'` 的 6 处全部是测试函数名。②`newControlledRestoreDriver` 增第五参数 `ownership` 并在构造期一次写入（现状 `recovery_restore.go:203` 不接收、`:216`／`:378` 要求非 nil），nil → `ErrAdmissionUnsupported`，生产只经构造器路径。③`recoveryRootInputs.restore` 改为 `restoreDriverFactory func(recovery.ExclusiveLease) (recovery.RestoreDriver, error)`；接线顺序固定为 `anchorRegistry.resolve`（`:555`）→ 仅 `enter`／`resume` `Acquire(ctx)` 恰好一次 → 同一 lease 交工厂 → 放入 `:441` 的字段 → `NewRuntime(ctx, deps, restoreLease)`（`:641`）→ `authority` 指向该 Runtime；`install`／`normal`／`verify` 传显式 typed nil。④assembly 增 `restoreLease` 字段、`sync.Once`、`closeRestoreLease()`，`enter`／`resume`／错误路径／`joinWork` 合并**恰好关闭一次**，关闭失败必须外传。⑤`start`（`:872`–`:896`）增 `enter` 分支且必须同步运行并缓存结果（禁 fire-and-forget）；`joinWork`（`:942`–`:964`）增 `case "enter"`：`Stop` → `waitFlow` → `closeRestoreLease`，`resume` 同序。⑥全量调用点：`NewRuntime(` 39 处／17 文件（1 定义 `runtime.go:729`＋1 生产 `recovery_runtime.go:641`＋37 测试），逐文件逐行列出实参，测试统一 `recoveryTestLease{}`（唯一例外 `runtime_test.go:1658` typed nil）。
- **判定**：设计层闭合；实现未做（现状 `case "enter"` = 0、`NewRuntime(ctx, deps, restoreLease)` = 0、构造器仍 4 参数）。

### 1.4 架构 `NEW-R05-03`（LOW）— 历史编号的机械检索判据

- **落点**：`:4079`（r66.7）；优先级规则 `:4072`。
- **修改摘要**：声明物理唯一非硬要求，语义唯一由 r66.0 优先级与 r66.7 判定；`:4062` 的 V7–V11（载荷与签名主体按 r66.2 绑定）为**唯一有效编号**；`:3942`／`:3990` 的旧步骤名、旧 `V1`–`V10` 改名说明与「以 r64.5／r64.2 为准」字样仅作历史、**不参与验证**；旧基线字段名的现行唯一合法语义是「旧键出现即 `ErrAdmissionBinding`」。
- **判定**：闭合（语义唯一可判；历史字面物理保留并已标注）。

### 1.5 架构 `NEW-R05-05`（LOW）— r61 起点与旧哈希标签

- **落点**：`:4072`（r66.0）；`:4080`（r66.8）。
- **修改摘要**：写明 r61 真标题在插入前 `:4278`／插入后 `:4290`；`:4247`（现值 `:4259`）是 r62 尾段写集表表头（其上一级标题 `:4257` 为 `r62.9.1 完整文件表`），`:4263`／`:4275` 为中间边界。按 raw 字节（含 LF/CR）自该行首至 EOF 复算并写入：`:4259` = 282,599 B／`7D1DDC08…`；`:4275` = 280,319 B／`5BD8E870…`；`:4290` = 277,334 B／`E40EB7A2…`。旧标签 `282,706 B／a1a256f5…` 在现行文件不可复现，作废；`282,599 B／7d1ddc08…` 只对应「r62 尾段＋r61 段」合并范围（起点 `:4259`），不得再标作 r61 段基线。
- **本端复算**：三个区间与上列数值逐字节一致（见第 3 节）。
- **判定**：闭合。

### 1.6 架构 `NEW-R05-02`（MEDIUM）— 作者证据链可复算

- **落点**：`:4080`（r66.8）；本报告与 `hashes.txt`。
- **修改摘要**：`:4080` 规定报告与 `hashes.txt` 必须在**最后一次写盘之后**生成并记录终值字节／行／SHA256，其后若再改设计文件必须重算覆盖、不得留旧值；报告须逐项给出 finding → 落点 → 摘要 → 判定，并单列「未闭合／升级待决」与「不构成评审通过、不构成生产解锁」；r61 段哈希只用插入后 `:4290` 起点并标明「至 EOF、raw 字节含 LF/CR」，只有「r62 尾段＋r61 段」合并范围才用 `:4259`；r05 旧数值与其「r61 起点 `:4247`」定位作废。
- **执行事实**：本报告与 `hashes.txt` 在 `design.md` 终值（1,079,559 B／5,476 行／`9B193BDE…`）写定之后生成；报告内不含任何未复算数值。
- **判定**：闭合。

### 1.7 架构 `F017`（条件项）

- **落点**：`:4081`（r66.9）。
- **修改摘要**：仅登记 `NewRuntime` 签名、双 lease 字段、全部读取点、driver ownership 与测试替身写集的卡面更新属 Task-Planner；`tasks.md:3186` 实测仍为 `- [ ] identity-auth-S4.24a …`，本轮不勾选、不改卡面、不声称 S4.24a 已交付或可实施。
- **判定**：**未闭合**（条件项，待 Task-Planner 对账）。

### 1.8 安全 `SEC-F003`＋`N001`（BLOCKER，同一根因）

- **落点**：`:4074`（r66.2）；就地修订 `:4062`（V8 子句）与 `:3990`（历史映射）。
- **修改摘要**：V7–V11 **每个编号各自绑定唯一载荷定义与唯一签名主体**：`V7` 独立读取原始字节、无签名；`V8` 验证 **producer 签名**，载荷唯一取 `:3988` 的 `JCS({"receipt": R10, "attestation": A6})`（`R10`＝12 字段回执删 `signature`／`audit_record_ref` 后的 10 字段，`A6`＝attestation 删 `signature` 后的 6 字段），主体唯一 `producer_key_id` 且其 `principal_id`＝`receipt_anchors.producer_principal_id`；`V9` 见证 head 5 键、载荷为删 `head_sig` 后的 JCS，主体唯一 `witness_key_id`；`V10` 验证方签名，载荷唯一 `:4059` 的 8 字段（`anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id`），主体唯一 `verifier_key_id`，`verified_at` 只在此处取样；`V11` 无签名。明确 **8 字段载荷只服务 V10，不得用于 V8、不得交叉引用**；`:4062` 的 V8 已就地改为引用 `:3988`，`:3990` 历史映射已改指 producer 载荷。
- **判据**：`rg -n 'V8.*r65[.]2'` = **0**（本端实测）。
- **判定**：设计层闭合（BLOCKER 的载荷歧义解除）。

### 1.9 安全 `N011`（HIGH）— 见证端点与独立性

- **落点**：`:4077`（r66.5）。
- **修改摘要**：①DNS 固定与连接绑定：预检只解析一次并固定全部 A/AAAA 为 IP 集合，`DialContext` 只连集合内 IP、禁连接期二次解析，连接后 `RemoteAddr` 先 `Unmap()` 再复核 ∈ 集合。②host 可判定判据：新增独立 `witness_host`（DNS 名；`netip.ParseAddr` 成功即拒绝 IP 字面值，`127.1`／`0x7f.0.0.1`／`0177.0.0.1` 等按 DNS 标签字符集与「末段全数字／`0x` 前缀／前导零」拒绝），`witness_host` 与 endpoint host 逐字节相等且小写、无尾点、无 percent、无 IDNA、无 zone；`ServerName = witness_host`，SNI／pin／目标 IP 同连接绑定。③IPv4-mapped：`Unmap()` 后判定，`::ffff:127.0.0.1` 即 loopback 被拒；拒绝 loopback／link-local／私网／ULA／unspecified／multicast／带 zone。④pin 唯一实现路径：只有见证客户端允许 `InsecureSkipVerify: true`，且必须同时提供 `VerifyConnection` 校验 (i) leaf SPKI SHA-256＝`witness_tls_spki_sha256`、(ii) `VerifyHostname(witness_host)`、(iii) 有效期；其他客户端禁止；不得用系统默认根或「系统根＋额外 pin」。⑤pin 轮换：任一时刻单一 active pin，轮换须人的明确批准并留审计，不做双 pin，旧 pin 只作回滚窗口、窗口外失败 closed。⑥独立性：`witness_key_id → keys.principal_id → principals.org_id` 解析链必须成立，org 为 `external_verifier` 且不得等于 root／executor／producer／issuer／reviewer／**verifier**，`witness_key_id` 不得等于上述任一 `key_id`；`receipt_anchors` 扩为 15 项（新增 `witness_host`）。
- **判定**：设计层闭合；DNS/TLS/见证运行面仍 NOT RUN。

### 1.10 安全 `NEW-S05-01`（HIGH）— `PATH` 放宽的完整抵消措施

- **落点**：`:4078`（r66.6）；优先级 `:4072`。
- **修改摘要**：tool 记录在 `environment` 含 `PATH=` 时新增 `environment_dirs`（升序去重），`:3930` 排序数组清单扩含该项；逐项目录约束全部合取——绝对规范路径、`st_uid == 0`、非 group／other 可写且非 executor 可写、从根逐级 `lstat` 无符号链接、挂载至少 `ro,nodev,nosuid`；`pt_interp_path`／`ld_so_cache_path`／`ld_so_conf_paths` 的父目录不得落在 `environment_dirs` 且不得可写；**禁止 `exec.LookPath` 与任何 `PATH` 搜索**，工具与子进程只按批准的绝对路径 `exec`；`PYTHONPATH`／`NODE_PATH`／`RUBYLIB`／`PERL5LIB` 出现即 `ErrAdmissionClosed`；目录数 ≤ 32、拼接总长 ≤ 4096 B、`PATH` 必须逐字节等于 `environment_dirs` 以 `:` 连接的结果。
- **判定**：设计层闭合；Linux 目录权限实测 NOT RUN。

### 1.11 安全 `NEW-S05-02`（MEDIUM）— 唯一优先级规则

- **落点**：`:4072`（r66.0）；`:4078`（r66.6）。
- **修改摘要**：r66.0 给出唯一可机核规则——**修订号大者覆盖小者（r66 > r65 > r64 > r63 > r62 > r61），同修订内小节号大者覆盖小者；跨修订字面冲突按此判定，与条文位置、出现次数、行号先后无关**。据此 `:4435`（r61.5）的「禁止 `PATH`」由 r66.6 取代；r65.6 的「恢复 r61 原字节」只是**字节恢复**声明，不改变语义优先级、不构成禁止，r61 段仍不得写入 r64／r65／r66 就地标记。
- **本端实测**：`:4435` 行内容与哈希仍为 r61 原字节（364 B／`429E633A…`），未被本修订触碰。
- **判定**：闭合。

### 1.12 安全 `NEW-S05-04`（LOW）— 精度残余

- **落点**：`:4077`（r66.5）；字段名残留 `:4079`。
- **修改摘要**：①`witness_endpoint` 恢复 **2048 B 上限**（按 UTF-8 字节）。②新增独立 `witness_host` 字段。③`witness_key_id` 的组织归属明确经 `keys.principal_id → principals.org_id` 解析（不允许假设 keys 自带 `org_id`）。④pin 轮换／回滚定义（人工批准＋审计＋单 active pin＋窗口外 closed）。⑤root／supplier 任命值域：`principals.kind = root`／`supplier` 的 `appointment_ref` 只允许空串、不得存在绑定 appointment、不参与六值枚举与六席；其余 kind 必须非空并解析到唯一 appointment；`appointment_ref` 固定 1..128 B ASCII 且匹配 `[A-Za-z0-9][A-Za-z0-9._-]{0,127}`，不 trim、不接受空白或控制字符。⑥`sequence_floor` 历史残留由 `:4079` 判为仅历史。
- **判定**：闭合。

### 1.13 两路共有的口径要求

| 要求 | 落点 | 实测 |
|---|---|---|
| `SEC-F002`、`SEC-F006`、`N012`–`N016` 不回退 | `:4082` | 明确「已闭合状态不回退」；本轮未改 `:3936`／`:3934`／`:1718`／`:4012`／`:4039` 的 r65 结论 |
| `SEC-F007` 与全部 Linux 侧能力仍 NOT RUN | `:4082`、`:4021`、`:4191` | 三处均 NOT RUN；无 PASS 表述 |
| 不声称 S4.24a 已交付／已关闭、生产可恢复、`enter`／`resume` 解锁、约束 8 满足 | `:4083`（r66.11）、`:4081` | `tasks.md:3186` 仍为未勾选；`:4083` 逐项否认 |
| 范围外 `N005` 只登记、不改 ADR／`tasks.md` | `:4081` | 冲突事实逐条抄录，未裁定 |

## 2 本轮实际写入（2 处行内替换）

| 行 | 内容 | UTF-8 字节增量 |
|---|---|---|
| `:3603` | F004：在原 r60 依赖集合段落后追加 r66.1 规范句（唯一签名、端口产出值定性、非可选入参、不进入 11 项表） | 382 → 850，**+468** |
| `:4072` | NEW-R05-02／NEW-R05-05：删除不可复现的旧「进入值」，改按 r05 报告原文与现行实测重述基线漂移，并声明 r06 终值为唯一基线 | 312 → 517，**+205** |

r66 块（`:4072`–`:4083`，r66.0–r66.11）在本轮 r06 交付内落盘；除上表 `:4072` 的基线叙述外，本端最后一次写盘未改块内其余各行。**未增删任何行**，因此 r66 之后各节行号保持 +12 映射。

## 3 可机核判据与实测输出（终值后）

```text
bytes=1079559  lines=5476  LF=5476  CR=169  BOM=False
head3=2D 2D 2D
line5=revision: identity-auth-design-ir01-r66
line38=**设计版本**：`identity-auth-design-ir01-r66`
```

```text
rg -n 'NewRuntime\(' --glob '*.go'          # 39 行
rg -l 'NewRuntime\(' --glob '*.go'          # 17 文件
rg -n 'func NewRuntime\(' internal/recovery/runtime.go   # 1：:729（仍 2 参数，实现未做）
rg -n 'NewRuntime\(' cmd/han-server/recovery_runtime.go  # 1：:641
rg -n 'NewRuntime\(ctx, deps, restoreLease\)' cmd        # 0（设计要求改后 =1）
rg -n 'case "enter"' cmd/han-server/recovery_runtime.go   # 0（设计要求改后 =1）
rg -n 'func .*Bind\(' --glob '*.go'          # 6，全部测试函数名
rg -o 'activeExclusive|ownedExclusive|injectedExclusive|closeRestoreLease' --glob '*.go'  # 0
```

```text
rg -n 'V8.*r65[.]2' design.md        # 0
nonce design.md                      # 2 行：:3973、:4075
bundle_digest design.md              # 2 行：:4075、:4082（均 OPEN／首期不实现）
sequence_floor design.md             # 10 行／19 次（全部历史语境）
driver.Bind design.md                # 5 行／6 次（全部历史语境，语义无效）
activeExclusive／ownedExclusive／injectedExclusive  # 7／19／38 次（全部设计要求）
SEC-F007 design.md                   # :4021、:4082、:4191（全部 NOT RUN）
```

```text
from=4290 to=EOF  bytes=277334 sha=E40EB7A29AA909A1C8A2918F3A8FCF1C2013793A401C6046D8C276DE79A33254
from=4259 to=EOF  bytes=282599 sha=7D1DDC087A496FC2C02C26E5F04017F8C6900A19E20D596CF5A4F40E9150F05A
from=4275 to=EOF  bytes=280319 sha=5BD8E87093A27AD40ED5A8DD056D56CE14160DB30A88DCEA26323DAC7E23B562
block 4072-4083   bytes=31140  sha=38BDE4DE601D068C26C45C0F276CE949BB5E2CBB5F8C86B7C7AB25EF49DCB74A
line 4435 (r61.5 PATH 禁句，原字节)  bytes=364 sha=429E633A6B13D5B26E851FF4F3459E7A1A33246A706F2CF30431235207FFE1A5
```

```text
golden source :4099  1115 B  sha=de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab
golden tool   :4105  2357 B  sha=8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508
```

`:4290` 段哈希与编辑前一致，证明 r61 段字节未被触碰；golden 两条与派发给定前缀一致，且为代码块行内容（不含行尾 LF）。

## 4 未闭合／升级待决

1. **`F004` 的 ADR 登记子项（升级待决）**：ADR-0031 全文 `NewRuntime` 出现 0 次，无条文冲突；但复审要求显式登记或指向已接受决策。是否修改 ADR 属架构决策方，本轮**未改 ADR**，**该子项未闭合**。
2. **`F017`（未闭合，条件项）**：`NewRuntime` 签名、双 lease 字段、全部读取点、driver ownership 与测试替身写集需 Task-Planner 更新卡面；`tasks.md:3186` 仍为未勾选。
3. **`N005`（范围外 BLOCKER，升级待决）**：`decisions/framework.md:1084`／`:1129` 称 `adr-0031-r09` 未复评、S4.24a 不得启动；`tasks.md:3197` 称 `adr-0031-r07` 已接受、门禁解除。三者对现行修订版本与门禁给出相反事实；本轮不改 ADR、不改 `tasks.md`。
4. **`SEC-F007` 与 Linux 能力（NOT RUN）**：host／ACL／mount／loader／fd-exec／maps／network／witness 全部未实测；`NoNewPrivs`／`CapBnd` 仍只是观测值。
5. **`bundle_digest` 显式 OPEN**：首期不实现、不定义字段与算法，`restoreReceiptWire` 12 字段形状不变。
6. **实现全部未做**：`NewRuntime` 仍 2 参数；`NewRuntime(ctx, deps, restoreLease)` 调用为 0；`case "enter"` 为 0；构造器仍 4 参数且不接收 `ownership`；`activeExclusive`／`ownedExclusive`／`injectedExclusive`／`closeRestoreLease` 在 Go 代码中均为 0。
7. **N011 运行面未验证**：DNS rebinding、TLS pin、IPv4-mapped IPv6、pin 轮换均只有设计判据，无任何实验。
8. **历史字面物理存在**：`sequence_floor` 10 行／19 次、`driver.Bind` 5 行／6 次仍在历史段，语义已由 r66.7／r66.4 判为无效；若复审要求物理唯一，需另立搬迁任务。
9. **工作树遗留**：根目录未跟踪的 `$null`、`$prof`、`'){'` 等非本轮产物；本轮无清理授权，未删除。

## 5 声明

**本轮不构成独立评审通过，不构成生产 restore 解锁，不构成 `enter`／`resume` 解锁，不构成 ADR-0031 约束 8 满足，不构成 `identity-auth-S4.24a` 可启动、可交付或可关闭。** 未勾选 `tasks.md` 任何复选框；未声称任何 NOT RUN 变为 PASS；未声称任何残余风险已消除。后续必须由架构与安全两路独立复审重新判定，并按 r66.8 复核本报告与 `hashes.txt` 的证据链。