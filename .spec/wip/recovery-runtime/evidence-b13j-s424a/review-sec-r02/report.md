# B13J-S424A 安全侧独立评审 r02（r62 复评）

## 0. 元信息、基线复算与范围

- 评审对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`；936613 B；5317 行；SHA256 `EA880660B8D43AFD6FB227812C662BB5A4581B39F4235F3DAB8E221F5F9FDC9A`；`revision: identity-auth-design-ir01-r62`。
- 作者证据：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r02/report.md`；18821 B；147 行；SHA256 `A910B3DD42223025B7D7D7C1CE80CD1D366E582BDDA2E13E8A415E4E660A0549`。
- 评审基线：`.spec/wip/recovery-runtime/evidence-b13j-s424a/review-sec-r01/report.md`；40500 B；243 行；SHA256 `C5D5BE68CCEAFC349722D6677C40F6A5BB8C46CB58A8D6F4FC7F01D16ED3A637`；verdict `CHANGES REQUIRED / BLOCKED`（BLOCKER 5 / HIGH 2 / LOW 1）。
- 复算方法：`Get-Item` 取字节、`[IO.File]::ReadAllLines(..., UTF8).Count` 取行数、`Get-FileHash -Algorithm SHA256` 取摘要。三项复算值与任务给定、作者报告、r01 自述**全部一致，无偏差**。
- 固定工作区：仓库根（`Han`），分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本轮只读；未修改 design、代码、`tasks.md`、ADR 或 `.add/**`；未做 Git 写、删除、移动或清理；工作区未提交改动原样保留。
- 本次范围：r62.0–r62.9（`design.md:3925`–`:4129`），重点 `design.md:3860`–`:4110` 与 r60／r61 restore 历史段；`decisions/framework.md` 的 ADR-0031 r07 不变量与状态段；ADR-0032 相关约束；`tasks/tasks.md:3186`–`:3218`；作者报告全文；r01 全文。安全侧只读评审，不实施、不评审实现代码。

## 1. 两轮实际检查范围

### 1.1 第一轮：原 finding 对照

逐条读取 r01 的 5 BLOCKER／2 HIGH／1 LOW 原文，回到 r62 声明落点，核对“作者声称改了什么”与“文本实际约束了什么”。重点不是看是否新增签名、清零、字段，而是检查这些机制能否被独立编码、是否有未覆盖分支、是否把未知写成已满足。

### 1.2 第二轮：消费者与上游依据的有界检查

沿真实消费者检查：批准记录 → 信任根 → `RestoreSourceClaim` → `revalidateApproval` → fd-exec → `RestoreReceipt` → `audit_sink`；并交叉核对 ADR-0031 r07 不变量 1–6、ADR-0032、`tasks.md` 关闭条件与作者报告 §5–§7。另对新增配置键、信任根文件边界、回执/审计持久化、第二 authority 与 closed→open 风险做独立检查。

### 1.3 独立复算：JCS golden vectors 与文档规则冲突

对 `design.md:3940`、`:3946` 两段 canonical signing input 做独立键序核验（解析 JSON 后按 UTF-16 code unit 排序比对），结果如下：

| 向量 | 文档字面字节 SHA256 | 文档声称的摘要 | 真正 JCS 键序字节 SHA256 | 结论 |
|---|---|---|---|---|
| `source`（`:3940`／`:3943`） | `3c7b1cff97b230077c9d3d75085082d26bfe5e48b8355cb721e0a830f51059e9` | 同左 | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` | 字面向量不是 JCS |
| `tool`（`:3946`／`:3949`） | `9df40264e869c7ed9506b3ac5788f5878cf516c980082ffa64bedb1ffb205d68` | 同左 | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` | 字面向量不是 JCS |

首个差异在两向量顶层：`"record_type"` 之后文档先写 `"revocation"` 再写 `"reviewer"`；按 `design.md:3935` 自述的 UTF-16 键序，`"reviewer"`（`revi`）小于 `"revocation"`（`revo`），必须先 `reviewer` 后 `revocation`。因此任何按 `:3935` 规则实现的 JCS 都无法复算出 `:3943`／`:3949` 的摘要；作者报告 `:35` 的“复算一致”只复算了字面顺序，不构成 JCS 独立复算。这是本轮 F001 不能判闭合的直接证据。

## 2. 逐条判定总表

| ID | 严重级 | 原 finding（r01） | r62 落点 | 本轮判定 |
|---|---|---|---|---|
| SEC-F001 | BLOCKER | 批准摘要自证，无可信外部锚 | r62.1 `:3929`–`:3958` | 部分闭合 |
| SEC-F002 | BLOCKER | 复核人独立性不可机核 | r62.2 `:3960`–`:3966` | 部分闭合 |
| SEC-F003 | BLOCKER | 有效期/撤销 TOCTOU 未闭合 | r62.3 `:3968`–`:3978` | 部分闭合 |
| SEC-F004 | HIGH | DSN `[]byte` 与线上形状矛盾、清零不完整 | r62.4 `:3980`–`:3992` | 部分闭合 |
| SEC-F005 | BLOCKER | 宿主 loader 边界不完整、schema 缺字段 | r62.7 `:4048`–`:4080` | 部分闭合（仍阻断） |
| SEC-F006 | HIGH | fd-exec 方向正确但身份/权限链未闭合 | r62.7.2–r62.7.3 `:4054`–`:4074` | 部分闭合 |
| SEC-F007 | BLOCKER | NOT RUN 不满足关闭条件 (4)(5) | `:4032`、`:4118`、作者报告 §7 | 部分闭合（仅诚实性修复，关闭条件未满足） |
| SEC-F008 | LOW | 规范化/措辞/示例精度不足 | r62.1／r62.4／r62.7 | 部分闭合 |
| N001 | BLOCKER | 新增：回执验证与审计信任链不闭合 | r62.5 `:3994`–`:4004` | 新增 |
| N002 | HIGH | 新增：`audit_sink` 只读打开却要求追加 | `:3996`、`:4000` | 新增 |
| N003 | HIGH | 新增：`tool.receipt` 混淆批准策略与实际回执 | `:3946`、`:3994`–`:3998` | 新增 |
| N004 | HIGH | 新增：artifact 共享 fd offset 未定义 | `:4010`、`:4060` | 新增 |
| N005 | BLOCKER | 新增（前置协调）：ADR r07 与 tasks 状态冲突 | `decisions/framework.md:1129`、`tasks/tasks.md:3197`、`design.md:4122` | 新增 |
| N006 | HIGH | 新增：netns／seccomp 无可用 pre-exec 机制且 network 记录未绑定实际配置 | `:4054`、`:4058`、`:4060`、`:4107` | 新增 |
| N007 | LOW | 新增：信任根挂载属性/ACL 未与工具同级明确 | `:3929`、`:3956` | 新增 |

计数口径：按本轮仍开放（未闭合或部分闭合）的当前严重级统计，不含已完全闭合项。结果见 §6。
## 3. 原 r01 findings 逐条判定

### SEC-F001 — 批准摘要自证，无可信外部锚

- 严重级：**BLOCKER**（维持）
- 原 finding：r61 的 `approval_digest` 只是同一文件内可重算的完整性值；任何能改写批准文件的主体都能改字段并重算摘要，不能证明“这是原始批准”。
- r62 落点：`design.md:3929`–`:3958`（信任根、pin、Ed25519、JCS、失败分类、残余风险）。
- 判定：**部分闭合**。
- 已成立的部分：
  1. 真实性确实移出批准文件：`design.md:3929` 要求独立信任根文件，由 `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH` 与 `_SHA256` 固定；信任根与批准文件必须位于不同挂载来源。
  2. 无“未签名也接受”分支：`:3954` 明确 r61 裸摘要批准文件在 r62 一律 `ErrAdmissionBinding`，且“没有未签名也接受的兼容分支”。
  3. 残余风险被诚实登记：`:3956` 明确“任何能同时改写信任根文件与 pin 的主体（root 或同权限部署方）仍可替换信任根”，结论是“批准抗伪造止于 root／部署方”，生产 restore 继续 closed。任务要求核对的“攻击者同时控制 root 与部署方能否同时替换信任根和 pin”因此答案是**能**，且文本没有否认。
  4. 平台无法验签 → `ErrAdmissionUnsupported`（`:3954`），没有静默降级。
- 未闭合部分（阻断）：
  1. **canonicalization 规则与 golden vector 自相矛盾**。`:3935` 规定 UTF-16 键序；`:3940`／`:3946` 的向量却把 `revocation` 排在 `reviewer` 前。按规则实现的 JCS 得到 `de9146…`／`8c0c89…`，而文档声称 `3c7b…`／`9df4…`（独立复算见 §1.3）。`:3949` 又把“独立实现必须复算出同一字节与同一摘要”纳入验收，导致该验收对正确实现不可满足。签名规范化是 F001 的核心，故不能判闭合。
  2. 作者报告 `:35` 与 §1 的“JCS golden vectors 复算一致”只对字面字符串成立；报告 `:110` 自己也承认“尚未由独立实现复现”，但未发现键序冲突。该声明不能作为 F001 的独立证据。
  3. `signature` 只覆盖“删除 `approval_digest` 与 `signatures` 后的记录”；`approval_digest` 本身不在签名范围内。当前设计让它由同一 canonical bytes 重算，方向可接受，但须在实现中明确“不得把外部传入的 `approval_digest` 当已认证值”；文本没有写出该禁止，属实现歧义（不单独升为 BLOCKER，但应与向量一起修）。
- 修复建议：
  1. 用独立 JCS 实现重生成两向量，键序改为 `record_type, reviewer, revocation, source_installation_id`（source）与 `record_type, reviewer, revocation, security, …`（tool），并同时给出 canonical 字节长度、摘要与首个差异位置的负向向量。
  2. 在 `:3949` 增加“若实现键序与向量不一致，必须拒绝启动，不得以向量为准改实现”的判定方向，避免实现者按错误向量写代码。
  3. 明确 `approval_digest` 只由验证方重算，不读取记录内自报值作为信任输入。

### SEC-F002 — 复核人独立性不可机核

- 严重级：**BLOCKER**（维持）
- 原 finding：r61 只用 `reviewer.person_name != issuer.person_name` 判独立；无稳定 principal、无 executor 字段、`review_ref` 不解析到独立记录，一人两名或同组织自审可通过。
- r62 落点：`design.md:3960`–`:3966`（principal、任命、review、kind、affiliations、时间）。
- 判定：**部分闭合**。
- 已成立的部分：
  1. `issuer`／`reviewer`／`executor` 均带 `principal_id` 与 `appointment_ref`，并要求登记在信任根 `principals`（`:3960`、`:3962`）。
  2. `review_ref` 必须解析到信任根 `reviews` 中唯一记录，并逐项匹配 reviewer／issuer／executor、`artifact_digest`、`tool_sha256`、`backup_ref`、`decision=approved`、`reviewed_at`（`:3960`）。
  3. 三 principal、三 appointment 两两不同，kind 分别为 approver／reviewer／executor（`:3962`）。
  4. 预检与每次 `revalidateApproval` 都重算同一合取（`:3964`），不是“预检过了就不再核”。
- 未闭合部分（阻断）：
  1. **`person_name` 规则自相矛盾**：`:3962` 说“`person_name` 只作展示，不参与任何判定”；`:3964` 的失败分类却写“`person_name` 与信任根 `display_name` 不一致 → `ErrAdmissionBinding`”。同一字段既“不参与判定”又“不一致即拒绝”，实现无法同时满足。
  2. **affiliations 与 supplier 的类型不匹配**：`:3931` 定义 `affiliations` 是字符串数组（组织名），`supplier_principal_ids` 是 principal ID 数组；`:3962`（4）（5）却要求“affiliations 不得包含任一 `supplier_principal_ids` 成员”“executor 的 affiliations 只允许 supplier”。字符串与 principal ID 不是同一命名空间，该交叉禁止无法机核；若解释为 affiliations 存 principal ID，则与 `:3931` 的“组织名”语义冲突。
  3. **同一公钥可登记为多个 principal**：`:3931` 只要求 `key_id` 升序且唯一，未要求 `public_key` 在 `keys` 内唯一。同一自然人可用同一把私钥登记为 issuer 与 reviewer 两个 principal，两两不同检查仍通过。
  4. **principal 无撤销/有效期字段**：`:3931` 的 `principals` exact keyset 只有 `principal_id, display_name, kind, appointment_ref, affiliations`；`:3954`／`:3964` 却要求“principal 已撤销、已过期” → `ErrAdmissionClosed`。该状态在 schema 中不可表示。
  5. **`reviews` 无 reviewer 签名**：复核记录只是信任根内的一条静态记录，没有 reviewer 对 `review_id`／`decision` 的签名；`review_ref` 的“独立复核”仍由信任根作者单方写入。配合 `:3956` 的 root／部署方残余，独立性只到“信任根内容自洽”。
  6. **`appointment_ref` 只是字符串**：没有独立任命记录或交叉登记；“两两不同”只防同一个字符串复用，不防同一真实任命被写成两个不同编号。
  7. 时间关系 `issued_at <= reviewed_at <= now`（`:3966`）只校验字符串解析后的时间，未要求 `reviewed_at` 由 reviewer 签名，故可被信任根作者任意填写。
- 修复建议：
  1. 删除 `:3964` 的 `person_name`→`display_name` 失败分支，或把 `person_name` 纳入判定并同步修改 `:3962`；二者只能留一个。
  2. 把 `affiliations` 改为 `affiliation_ids`（与 principal 同一命名空间）或新增 `supplier_affiliations` 字符串集合；明确比较对象。
  3. `keys` 增加“`public_key` 全局唯一”，并拒绝同一 key 出现在多个 principal。
  4. `principals` 增加 `valid_from, valid_until, revocation_ref, revoked_at`（或引用独立 revocation 记录）；`reviews` 增加 reviewer 的 Ed25519 签名。
  5. `appointment_ref` 解析到信任根内的独立 appointment 记录（含组织、角色、生效期、撤销），而不是裸字符串。

### SEC-F003 — 有效期/撤销 TOCTOU 链条未闭合

- 严重级：**BLOCKER**（维持）
- 原 finding：`prepared()` 不重读批准文件；`now` 复用预检时间；撤销最大生效窗口与 in-flight 语义未定义；只读挂载与原子替换矛盾；`inode` 漂移被归为 Binding 而非 Closed。
- r62 落点：`design.md:3968`–`:3978`。
- 判定：**部分闭合**。
- 已成立的部分：
  1. 唯一重校验方法 `revalidateApproval(ctx, now)`（`:3971`），读取点固定为预检、`Prepare` 返回 plan 前、`Replace` 启动子进程前、`VerifyRestored` 启动子进程前（`:3976`）。
  2. 字节、inode、`st_dev`、挂载来源或挂载点漂移、删除、截断、非 regular → `ErrAdmissionClosed`（`:3974`），不再误归 Binding。
  3. 每个检查点现取一次 `clock.Now()`，跨检查点必须重取；本次 `now` 早于上次已观测 `now` → `ErrAdmissionClosed`（`:3976`）。
  4. 撤销不晚于下一检查点生效；子进程启动后撤销，子进程返回后再校验，失败以 `errors.Join(ErrRecoveryWriteUnknown, err)` 返回，调用方只能按 `restore_unknown`（`:3978`）。因此“撤销后仍返回成功”在正常实现下被阻断。
- 未闭合部分（阻断）：
  1. **时钟回拨的高水位只在内存**：`:3976` 的“上次已观测 `now`”没有持久化字段、没有单调时钟源、没有跨进程重启语义。restore 是一次性进程；攻击者在批准过期后重启进程并把系统时钟拨回 `valid_until` 之前，`valid_from <= now < valid_until` 会重新成立，`issued_at <= reviewed_at <= now` 也成立。文本“系统时钟回拨不得延长有效期”没有可执行的持久化机制。
  2. **`snapshot.Clear()` 与“字节逐项比较”矛盾**：`:4036` 要求 `clearStartupSecrets` 之后调用 `snapshot.Clear()` 清除 DSN 副本；`:3974` 又要求 `revalidateApproval` 与“预检快照”逐字节比较 DSN。若 DSN 字节已被清零，逐字节比较不可能；若保留字节，则违反 Clear。需明确快照只保留摘要/长度/元数据，字节比较改为摘要比较，并说明该摘要是否足以发现漂移（需要 salt 或 HMAC，避免离线猜测）。
  3. **`Replace` 的“各调用一次”歧义**：`:3976` 写“`d.prepared(ctx)` 与真正启动子进程之前各调用一次”，可读成 `prepared()` 之前一次、启动前再一次，也可读成两者各一次；与 `:4010` 的 verifier 调用时机叠加后，实际调用次数与顺序不确定。需写成伪代码或明确序号。
  4. **最后一次 `revalidateApproval` 到 `execve` 的窗口**仍未闭合到“撤销后不执行”：窗口内撤销只能在子进程返回后被发现，结果按 `restore_unknown` 处理。该语义本身可接受（不宣称成功），但必须明确“`restore_unknown` 不是成功”，且不得在审计中写 `result=success`。文本没有规定审计 `result` 的取值域与失败时的写入。
  5. **读失败分类不一致**：`:3974` 说文件被删除、截断、非 regular、不可读 → `ErrAdmissionClosed`；`:4002` 说 `independent_read_path`／`audit_sink` 缺失或不可读 → `ErrAdmissionIncomplete`。同类“不可读”在不同对象上分类不同，需明确“批准/信任根不可读=Closed，回执/审计不可读=Incomplete”的边界，避免实现按错分类放行。
- 修复建议：
  1. 引入持久化单调高水位（例如在 Journal 或专用只读状态文件记录上次观测 `now` 与最大已见 `valid_until`），或要求 `CLOCK_MONOTONIC` 与启动时可信时间源；跨重启必须先核高水位，回拨即 Closed。
  2. 明确快照的 DSN 只保留 `sha256:` 摘要、长度与文件元数据，不保留原字节；比较用摘要，并在 Clear 清单中列出该摘要的生命周期。
  3. 用编号伪代码固定 `Prepare`／`Replace`／`VerifyRestored` 的 revalidate 次数、顺序与 `prepared()` 的关系。
  4. 定义审计 `result` 枚举（如 `verified`／`unknown`／`closed`），`restore_unknown` 一律不得写 `verified`。

### SEC-F004 — DSN `[]byte`、清零与线上形状

- 严重级：**HIGH**（维持；实施前必须修复）
- 原 finding：`[]byte` 经 `json.Marshal` 变 base64 与“线上形状不变”矛盾；`string` 副本、pipe、子进程、core dump 不可清零；错误/日志/Trace 可能带 DSN。
- r62 落点：`design.md:3980`–`:3992`。
- 判定：**部分闭合**。
- 已成立的部分：
  1. 作废 `[]byte` 字段方案：wire 字段仍是 `string`，`target_connection.dsn` 始终是 JSON 字符串；禁止 `json.Marshal([]byte)`、base64、`json.RawMessage`、第二 DSN 字段（`:3982`）。
  2. 唯一编码路径 `privateRestoreInput` + `appendJSONString`，不经 `string(dsn)` 中间副本（`:3984`）。
  3. 五处清零时点已列：`execute` defer、`runChild` 写后、`Close`、预检快照 `Clear()`、子进程解析后（`:3986`）。
  4. 残余风险诚实登记：GC、管道内核缓冲、子进程地址空间、core dump、调试器；只承诺“本进程内可见副本”，不承诺“绝不泄露”；`RLIMIT_CORE=0`、`PR_SET_DUMPABLE=0`、`MADV_DONTDUMP` 属 unknown（`:3988`）。
  5. 脱敏面扩大：DSN 不进 argv／ENV／日志／Trace／Journal／审计正文／stdout／stderr（`:3990`）。
- 未闭合部分（实施前必须修复）：
  1. **`revalidateApproval` 每次重读 DSN 的新副本不在清零清单**：`:3974` 明确重读“三者精确字节”，包括目标 DSN 文件；`:3986` 的五处没有覆盖 `revalidateApproval` 内部读取缓冲与比较缓冲。每次 `Prepare`／`Replace`／`VerifyRestored` 都会新增副本。
  2. **`parseRestoreTargetDSN` 的百分号解码缓冲未清零**：`:3992` 要求百分号编码解码后无 NUL/CR/LF；解码会产生新的 `[]byte`／`string`。`string` 不可清零；若用 `[]byte` 解码，也未列入清单。
  3. **`net/url` 等标准库产生的不可变字符串未处理**：若用 `url.Parse`，`url.Userinfo` 的 password 是 `string`（不可清零），且解析期间可能保留原始 DSN 的派生字符串。五处清零只覆盖 `[]byte`，无法覆盖这些副本。必须明确“禁止 `url.Parse` 直接解析含密码 DSN”或接受该残余并登记。
  4. **错误携带 SHA-256 前 8 字节构成离线校验 oracle**：`:3990` 允许错误携带“长度与 SHA-256 前 8 字节”。对低熵或已知格式的 DSN（host/user/端口可猜），攻击者可离线枚举密码候选并与 8 字节前缀比对；该值还可能进入日志/告警。它不是明文，但属于新的指纹泄漏面，且与“最小必要”不一致。应改为进程内关联 ID（随机 nonce）或完全不带摘要。
  5. **panic／`%v`／`fmt.Errorf("%w")` 未纳入 redaction 合同**：`:3990` 只规定 `privateRestoreError` 对含 `target_connection` 的错误降级；没有禁止 `panic(dsn)`、`fmt.Errorf("... %q", dsn)`、第三方驱动把 DSN 放进 error text、或 `%+v` 打印配置结构体。需要“所有错误在离开 DSN 作用域前统一经过 redactor”的机制，而不是逐处约定。
  6. **子进程侧的副本只在“解析连接参数后”清零**：`:3986`（5）要求子进程解析后清零其 DSN 缓冲；但子进程的 JSON 解码器（`encoding/json`）会先分配 `string` 字段，`string` 不可清零。需要明确子进程也禁用 `encoding/json` 解 DSN 字段，改用流式 `[]byte` 解析并清零，或登记该残余。
  7. **环境/argv 的“禁止”没有对应执行检查**：`:3990` 是文本禁止，未要求运行前扫描 `/proc/<pid>/cmdline`、`/proc/<pid>/environ` 做 canary 断言；卡面 (d) 的负向测试也未覆盖 DSN canary。应把 canary 检查纳入验收。
- 修复建议：
  1. 把 `revalidateApproval` 的 DSN 读取/比较缓冲、`parseRestoreTargetDSN` 的解码缓冲、子进程 JSON 解码缓冲全部加入清零清单，并给出每处的所有权与生命周期。
  2. 禁止 `url.Parse` 直接处理含密码 DSN；用自写解析器在 `[]byte` 上完成语法与百分号解码，解码后立即使用并清零。
  3. 错误只用随机关联 ID，不带 DSN 摘要前缀；若确需关联，使用进程内随机 nonce 映射，日志不落盘。
  4. 增加 DSN canary 负向测试：argv／env／log／trace／stdout／stderr／Journal／audit／panic 均不得出现 canary；子进程收到的 JSON 解码后与原 DSN 逐字节相等。

### SEC-F005 — 宿主 loader 边界（BLOCKER）

- 严重级：**BLOCKER**（维持）
- 原 finding：schema 缺 `elf_type`／`PT_INTERP`／`DT_NEEDED`／cache 路径摘要／setuid/caps 等字段；未检查解释器、依赖、`DT_AUDIT`、`ld.so.conf`、库内容与挂载、`AT_SECURE`、网络隔离、CWD/fd 继承、ACL、父目录逐级 lstat。
- r62 落点：`design.md:4048`–`:4080`（r62.7.1–r62.7.4），schema 在 `:4050`–`:4054`，判定顺序在 `:4052`，网络在 `:4058`，ACL 在 `:4078`，父目录在 `:4080`。
- 判定：**部分闭合（仍阻断）**。
- 已成立的部分（覆盖面显著扩大）：
  1. `tool.elf` exact keyset 含 `elf_type, pt_interp_path, pt_interp_sha256, dt_needed, dt_audit, dt_depaudit, ld_so_cache_path, ld_so_cache_sha256, ld_so_conf_paths, rpath, runpath`（`:4050`）。
  2. ELF 魔数/class/data/type/machine 校验；拒绝脚本与其他架构（`:4052`（1））。
  3. `PT_INTERP` 逐字节等于记录、解释器 no-follow/root-owned/无 group-other 写/`ro,nodev,nosuid` 挂载/摘要一致；static 出现 `PT_INTERP` 或 dynamic 缺 `PT_INTERP` 均拒绝（`:4052`（2））。
  4. `DT_NEEDED` 必须纯 soname，解析路径与记录集合双向相等；只允许 cache 与 conf 并集，禁止 `$ORIGIN`／`$LIB`／`$PLATFORM`／RPATH/RUNPATH 搜索（`:4052`（3））。
  5. `DT_AUDIT`／`DT_DEPAUDIT`／`DT_RPATH`／`DT_RUNPATH` 任一存在即拒绝；`/etc/ld.so.preload` 非空拒绝（`:4052`（4）（5））。
  6. cache/conf/dependency 逐项摘要与挂载属性；依赖挂载必须 `ro,nodev,nosuid` 且无 `noexec`（`:4052`（6）（7））。
  7. exec 后 `/proc/<pid>/maps` 扫描，未知文件映射、`(deleted)`、`memfd:`、匿名可执行映射 → `ErrAdmissionClosed`（`:4052`（8））。
  8. ACL 编码算法、父目录逐级 lstat、`O_NOFOLLOW` + fstat/lstat 一致（`:4078`、`:4080`）。
- 未闭合部分（阻断）：
  1. **“校验 cache/conf”与“实际 loader 用哪个 cache/conf”未绑定**：glibc 的 `ld-linux` 运行时读取的是编译期固定的 `/etc/ld.so.cache`（可用 `--inhibit-cache` 等改变，但设计未选），不会读取任意 `ld_so_cache_path`；`ld.so.conf` 主要由 `ldconfig` 消费，运行时 loader 在 cache 缺失时才可能直接搜索默认目录。`:4052`（3）的“解析只允许来自 `ld_so_cache_path` 与 `ld_so_conf_paths` 并集”是设计自己的解析器，不是 loader 的解析器。若 `ld_so_cache_path` 不是 loader 实际读取的路径，就出现“校验了 A、执行用 B”。必须要求 `ld_so_cache_path` 等于该 `PT_INTERP` 实际使用的 cache 路径，并证明 loader 不会回退到其他路径/默认目录。
  2. **`ld.so.conf` 的 include/glob 与 `hwcap` 子目录未定义**：`ld_so_conf_paths` 只列文件路径与摘要，未要求展开 `include /etc/ld.so.conf.d/*.conf`、拒绝符号链接 conf、处理 `hwcap` 子目录与 cache 的 `FLAG_*` 语义。实际搜索集合可能大于已核集合。
  3. **解释器自身的依赖未核**：`PT_INTERP` 只核摘要与挂载，未要求解释器也是 ELF、无 `PT_INTERP` 自身递归、其 `DT_NEEDED` 也纳入 `dt_needed` 或单独记录。若解释器动态依赖 libc，实际加载集合超出批准。
  4. **`network` schema 不足以承载 `:4058` 的检查**：`:4054` 的 `tool.network` 只有 `mode, netns_inode, netns_path, target_addr`；`:4058` 却要求 `/etc/hosts` 由 root-owned `0644` `ro` 挂载的批准文件提供、路由表精确、seccomp 规则、`/proc/<pid>/net/route` 一致。这些文件/策略的路径、摘要、所有者、规则集都没有进入批准记录，运行时可被替换而不被批准层发现。
  5. **ENV 封闭 allowlist 未落到 exec**：`tool.environment` 只是记录数组（`:3946` 示例三个变量），r62.7 没有规定子进程 `envp` 必须“恰等于”该数组、不得继承父进程环境。若实现用 `os.Environ()` 过滤，`LD_PRELOAD`／`LD_AUDIT`／`LD_LIBRARY_PATH`／`GLIBC_TUNABLES` 等注入面仍存在；若用白名单，文本没有写“其余全部清除”。这是“校验了记录、执行用了别的东西”。
  6. **maps 扫描时点逻辑不成立**：`:4052`（8）说“exec 前与 exec 后各扫描一次 `/proc/<pid>/maps`，两次集合必须相等”。exec 前子进程尚未 exec，maps 是 fork 后的父进程映像；exec 后是工具映像，集合不可能相等。若本意是“exec 后立即扫描与稍后扫描”，则存在扫描窗口内 `dlopen` 的 TOCTOU，文本虽承认“依赖 seccomp 与工具合同、未实测”，但“两次集合相等”的表述与实现语义不一致。
  7. **`/etc/ld.so.preload` 只查内容，不查路径/挂载/所有者**：`:4052`（5）只要求“不存在或 0 字节”；未要求该路径本身 no-follow、root-owned、`ro` 挂载、父目录受控。攻击者可在批准后 bind-mount 一个空文件覆盖，再在 exec 窗口替换（内容检查与 exec 之间）。
- 修复建议：
  1. 把 `ld_so_cache_path` 绑定到 `PT_INTERP` 的实际 cache 路径（对 glibc 固定 `/etc/ld.so.cache`），并要求 `ld_so_conf_paths` 是实际被 loader 读取的闭包（含 include/glob 展开后的每个文件、每级摘要、符号链接拒绝）。
  2. 解释器自身的 `DT_NEEDED` 与安全属性纳入 schema 或单独 `interp` 记录；拒绝解释器再含 `PT_INTERP`。
  3. 扩展 `tool.network`：`hosts_path/hosts_sha256/hosts_owner/mode/mount`、`route_set`（或 route 摘要）、`seccomp_profile_sha256`、`netns_owner`、`netns_created_by`；运行时逐项与记录比较。
  4. 明确子进程 `envp` 恰为 `tool.environment`，并给出“未列变量一律不继承”的实现要求与 canary 测试。
  5. 把 maps 扫描改为“exec 后立即扫描 + 回执前再扫描 + 两者差集必须为空”，并明确 seccomp 对 `mmap(PROT_EXEC)`／`mprotect(PROT_EXEC)`／`dlopen` 的拦截规则；否则不得声称覆盖完整。
  6. 对 `/etc/ld.so.preload` 增加路径身份、所有者、挂载、父目录检查，并与 cache/conf 同级。

### SEC-F006 — fd-exec 执行身份/权限链

- 严重级：**HIGH**（维持；实施前必须修复）
- 原 finding：`execveat(AT_EMPTY_PATH)` 需 `CAP_SYS_ADMIN` 与降权冲突；`/proc` fallback 未定义；`Args[0]` 语义；`0600` 不可执行；`openImage` 仍按路径打开。
- r62 落点：`design.md:4048`（方向）、`:4054`–`:4056`（身份/权限检查）、`:4060`（argv/fd/生命周期）、`:4074`（lease）。
- 判定：**部分闭合**。
- 已成立的部分：
  1. 作废 `execveat(AT_EMPTY_PATH)`，改为 root 校验 fd → 子进程 fd 3 → `Path="/proc/self/fd/3"` → `Credential` 降权（`:4048`）。
  2. `/proc` 不可用即 `ErrAdmissionUnsupported`，无 fallback、无 root exec 窗口（`:4048`、`:4056`）。
  3. 工具 root-owned `0755`（`:4054`），artifact root-owned `0600` 只经 fd 4（`:4054`）；禁止 setuid/setgid/file capabilities（`:4054`）。
  4. 专用非 root uid/gid、空 `Groups`、`NoSetGroups=false`，exec 后按 `/proc/<pid>/status` 复核（`:4054`、`:4056`）。
  5. `Args` 恰为 `{executable_path, verb}`，`verb` 只允许 `restore-v1`／`verify-v1`，无第三 argv（`:4060`）。
  6. `ExtraFiles` 恰为 fd 3/4，其余父 fd `O_CLOEXEC`；`Setpgid` + `SIGKILL(-pgid)` + `Wait`，未 reap 不归还 target/pool/lease（`:4060`）。
- 未闭合部分（实施前必须修复）：
  1. **`CapBnd=0` 与 `NoNewPrivs=1` 没有设置机制**：`:4056` 要求 exec 后 `CapInh=CapPrm=CapEff=CapBnd=CapAmb=0`、`NoNewPrivs=1`。`syscall.Credential` 只能完成 setgroups/setgid/setuid；`os/exec`／`SysProcAttr` 没有 `prctl(PR_CAPBSET_DROP)`、`PR_SET_NO_NEW_PRIVS`、seccomp 的 pre-exec 钩子。若只靠 `Credential`，非 root uid 的 `CapBnd` 仍继承父进程的 bounding set，`NoNewPrivs` 仍为 0，exec 后的检查必然失败。
  2. **setns 到既有 netns 没有机制**：`:4054` 用 `netns_path`／`netns_inode`，`:4058` 要求 `netns` 内网络；但 `SysProcAttr` 不支持 `setns(2)` 到既有 netns（只支持 `CLONE_NEWNET` 新建）。若引入 `nsenter`／`unshare` 或自写 wrapper，wrapper 就是实际被 exec 的未批准二进制，且会出现在 `/proc/<pid>/maps` 中，与“唯一批准工具”和 maps 白名单冲突。设计没有选择机制，也没有把 wrapper 纳入批准记录。
  3. **seccomp 安装机制未定义**：`:4058` 的 socket/connect 过滤需要 pre-exec 安装；Go 标准库不提供。与 (1)(2) 相同，需要 wrapper 或 cgo/pre-exec 方案；文本把它列在 `recovery_restore_exec_linux.go` 的职责里，但没有可编码路径。
  4. **降权与 capability 的顺序未定义**：先 `setuid` 再 `PR_CAPBSET_DROP` 需要 `CAP_SETPCAP`（普通非 root 已无）；先 drop 再 `setgroups` 需要 `CAP_SETGID`。正确顺序依赖具体机制，文本只说“内核在 fork 之后、execve 之前一次完成”，不足以保证 (1)。
  5. **`Pdeathsig`／`Setsid` 未定义**：只设 `Setpgid`；父进程崩溃时子进程可能继续持有 DB 连接。不是主要安全边界，但影响 `restore_unknown` 的证据完整性。需明确父死子死（`Pdeathsig=SIGKILL`）或 supervisor 兜底。
  6. **`Credential` 生效与 `Setpgid` 顺序**：Go 在子进程内先设置 credential，再 setpgid/setsid；对非 root uid 设置 pgid 是允许的，但如果实现改为“先 setpgid 再降权”或使用 wrapper，顺序会变。需要写成固定顺序并加断言。
- 修复建议：
  1. 明确 pre-exec 机制：自写极小 launcher（纳入批准记录与摘要，作为 fd 3 之前的唯一入口）或 cgo `fork`+`prctl`+`setns`+`seccomp`+`setuid`+`execve`；把 launcher 也纳入 `tool` 记录、maps 白名单与哈希。
  2. 给出固定顺序：`setgroups(0)` → `setgid` → `setuid` → `PR_CAPBSET_DROP`（若仍具 CAP_SETPCAP）或先在 root 阶段 drop bounding set → `PR_SET_NO_NEW_PRIVS` → `setns` → seccomp → `execve`；每一步失败即 `ErrAdmissionClosed`。
  3. 若无法在无 wrapper 的情况下满足 `CapBnd=0`／`NNP=1`／setns／seccomp，明确登记“该组合不可实现”，不得只写 post-exec 断言。
  4. 增加 `Pdeathsig` 或 supervisor 兜底，并定义父进程崩溃后的 `restore_unknown` 证据。

### SEC-F007 — NOT RUN 诚实性

- 严重级：**BLOCKER**（仅阻断 S4.24a 关闭，不要求继续改设计）
- 原 finding：r61 把 (d)(e)(f) 标 NOT RUN 是诚实的，但未满足卡面关闭条件 (4)(5)。
- r62 落点：`design.md:4032`、`:4118`、作者报告 §7 `:134`–`:141`。
- 判定：**部分闭合（仅诚实性修复，关闭条件未满足）**。
- 已成立的部分：
  1. `:4032` 明确“本设计通过评审也不构成 S4.24a 关闭”；关闭条件 (4)(5) 需要 Linux 目标或 CI 上的真实负向 (d)(e)(f) 与 9 条验收命令证据。
  2. `:4032`、`:4118` 明确“Windows 上的 skip／无匹配 exit 0 不是 PASS”，没有把 NOT RUN 写成 PASS。
  3. 作者报告 `:137`–`:140` 与 `:147` 重复同一结论，未暗示可关闭 S4.24a。
  4. `:4034` 把 (a)(b) 拆成“平台无关解析/语义层”和“POSIX host 层”，Windows 只断言 `ErrAdmissionUnsupported` 且零状态变更，避免把 Windows 单测冒充 host 实测。
- 未闭合部分：
  1. 关闭条件 (4)(5) 仍缺 Linux 真实证据；这是环境与证据缺口，不是文本可修复项。设计评审不得给出“可关闭”结论。
  2. 作者报告与 design 都没有给出“补证环境、命令、期望输出、证据落点”的可执行清单；`tasks.md` 的 9 条命令仍是 Windows 本地命令，未包含 Linux 目标/CI 的 host／fd-exec／netns／seccomp／loader 实测命令。
- 结论：诚实性已修复，**不得**因本轮设计评审通过而关闭 S4.24a；不得把 skip/exit 0 当 PASS。

### SEC-F008 — 规范化/措辞/示例精度

- 严重级：**LOW**（维持）
- 原 finding：键序/转义/整数/数组未定义；ACL 编码不足；时间关系缺失；mount options 顺序/重复未定义；`st_dev`/`st_ino` 精度；`mode=0600`；receipt 字段与卡面 (e) 不对应。
- r62 落点：`:3935`（JCS/整数/数组）、`:3966`（时间）、`:3946`（mode/字符串整数）、`:4076`（mount options）、`:4078`（ACL）、`:3994`–`:3998`（receipt）。
- 判定：**部分闭合**。
- 已成立的部分：
  1. JCS 规则、字符串转义、整数十进制字符串、数组排序/去重、禁 `null` 已写明（`:3935`）。
  2. 时间关系 `issued_at <= reviewed_at <= now`、`valid_from <= now < valid_until`、最大 366 天已写明（`:3966`）。
  3. `mode` 示例改为 `0755`（`:3946`、`:4054`）；`st_dev`/`st_ino` 改为十进制字符串（`:3935`、`:3946`）。
  4. mount options 按逗号分割、字节升序去重后比较（`:4076`）；ACL 规范文本、tag 数值、mask、default ACL 不参与已写明（`:4078`）。
- 未闭合部分：
  1. JCS golden vector 键序错误（见 F001），这是最严重的精度缺陷。
  2. `person_name` 判定矛盾（见 F002）。
  3. `tool.receipt` 在批准记录中为全空占位（`:3946`），却在 `:3996` 要求实际签名/时间/序列；字段语义混用（见 N003）。
  4. `audit_sink` 以 `O_RDONLY` 打开（`:3996`）却要求追加（`:4000`），机械矛盾（见 N002）。
  5. `network` schema 与 `:4058` 检查项不匹配（见 F005 第 4 点）。
  6. `affiliations` 与 `supplier_principal_ids` 类型不匹配（见 F002 第 2 点）。
  7. `:4052`（8）maps“exec 前/后集合相等”表述不成立（见 F005 第 6 点）。
- 结论：作为 LOW 不单独阻断方向，但上述精度缺陷会直接导致实现歧义或验收不可满足，应与对应 BLOCKER/HIGH 一并修复。
## 4. 新增安全 findings

### N001 — 回执验证与审计信任链不闭合

- 严重级：**BLOCKER**
- 原 finding：无（本轮新增；与 ARCH-F018／ADR-0031 不变量 6 相关）。
- r62 落点：`design.md:3994`–`:4004`；信任根 `:3931`；tool 记录 `:3946`；执行回执 `:3884`。
- 判定：**新增**。
- 证据与问题：
  1. **`verifier_principal_id` 只是标签，没有验证方签名**：`:3996` 要求它是信任根中 `kind=verifier` 的 principal，且不等于 executor/issuer；但 `:3998` 的验签只验“回执生产者”的 Ed25519 签名，没有任何“验证方对核验结果签名”的动作。`:4000` 第七步的审计记录由 root 进程自己追加，`verifier_principal_id` 由同一进程填写。所谓“验证方独立于执行身份与 root”没有可验证证明。
  2. **`producer_key_id` 未要求独立于执行者**：`:3931` 的 `receipt_anchors` 有 `producer_principal_id`／`producer_key_id`，但 `:3996` 只禁止 verifier 等于 executor/issuer，没有禁止 producer 等于 executor/issuer。执行者可自产回执并自验。
  3. **`sequence_floor` 不推进**：`:3931` 的 `sequence_floor` 是信任根内静态值；`:4000` 第六步只读 `audit_sink` 最后一条记录并要求本次 `sequence` 严格更大；没有任何“验证成功后把新 sequence 持久化为下一 floor”的机制。若 `audit_sink` 被截断/回滚为空文件，旧回执只要 `sequence > sequence_floor` 且 `issued_at` 落在 `[now-5m, now]`，仍可通过。`issued_at` 的 5 分钟窗口把重放限制在 5 分钟内，但窗口内仍可重放。
  4. **`audit_sink` 是本地文件，没有外部 witness**：`:4004` 说“首期不提供真实 anchor、真实见证端点与真实审计 sink”，但 `:3996`／`:4000` 又把 `audit_sink` 当作单调序列与审计证据。本地 root 可写文件不构成 ADR-0031 不变量 6 要求的“外部见证/独立读取”。`anchor_ref` 只是信任根内记录，不是可达的见证端点。
  5. **`audit_record_ref` 未签名且事后回填**：`:3996` 要求它在签名载荷中为空，`:4000` 第八步“回填后返回”。回填值不在签名内，任何持有回执的消费者都可改写；审计记录与回执之间没有双向签名绑定（审计记录也没有签名）。
  6. **`independent_read_path` 与 `audit_sink` 未要求不同所有者/不同目录**：`:3996` 只要求 `independent_read_path` 所在挂载来源不得等于工具/artifact 挂载来源；没有要求它与 `audit_sink` 不同挂载、不同所有者或执行身份不可写。二者可重叠，独立读取退化为同一文件。
- 影响：执行者或 root 可自产、自验、自记审计，并可在截断审计后重放；回执“单调序列”只在单个未被篡改的本地文件内成立。ADR-0031 不变量 6 的“验证方独立、外部见证、审计留痕”未被编码。
- 修复建议：
  1. 引入真实外部见证端点或 append-only 远端日志；`anchor_ref` 必须可解析为可达端点，验证方对 `{receipt_sha256, sequence, verified_at, result}` 签名。
  2. `receipt_anchors.sequence_floor` 改为每次验证成功后由见证方原子推进；本地只缓存，不作为权威。
  3. `audit_record_ref` 纳入签名，或让审计记录对回执签名形成双向绑定；审计记录本身也需签名/链式哈希。
  4. 明确 producer ≠ executor ≠ issuer 的机核约束；`receipt_anchors` 增加 `producer_kind` 并要求与 `principals.kind` 一致。
  5. `independent_read_path` 与 `audit_sink` 必须不同挂载、不同所有者，且执行身份对二者均不可写。

### N002 — `audit_sink` 只读打开却要求追加

- 严重级：**HIGH**
- 原 finding：无（本轮新增；机械契约矛盾）。
- r62 落点：`design.md:3996`、`:4000`。
- 判定：**新增**。
- 证据与问题：
  1. `:3996` 明确 `independent_read_path` 与 `audit_sink` “都是绝对规范路径，以 `O_RDONLY|O_CLOEXEC|O_NOFOLLOW` 打开”。
  2. `:4000` 第七步要求“追加审计记录 … 并 `fsync`”。`O_RDONLY` fd 不能 `write`/`append`；该契约在 POSIX 上不可实现。
  3. 未要求 `audit_sink` 的 owner/mode/ACL，未要求执行身份不可写，未要求 `O_APPEND`、文件锁、单行原子写、父目录 `fsync`、`fsync` 失败分类。
  4. 并发 `VerifyRestored` 可能交错写同一 sink，`sequence` 检查（读最后一条）与追加之间没有锁，两个验证可读到同一 last sequence 并写入重复 sequence。
- 影响：按文本实现会直接失败；若实现者自行改为可写打开，则绕过文本的只读约束，且没有并发/截断/权限边界。
- 修复建议：
  1. 拆成两个 fd：`audit_read_fd`（`O_RDONLY|O_CLOEXEC|O_NOFOLLOW`，用于读取 last sequence）与 `audit_append_fd`（`O_WRONLY|O_APPEND|O_CLOEXEC|O_NOFOLLOW`，用于追加）。
  2. 要求 `audit_sink` root-owned、`0600` 或更严格、执行身份不可写、所在挂载 `ro` 之外的受控可写目录，并对追加使用 `flock`/`fcntl` 锁 + 单条记录一次 `write` + `fsync` + 父目录 `fsync`。
  3. 定义 `fsync` 失败、短写、锁超时的失败分类（`restore_unknown` 或 `ErrAdmissionIncomplete`），不得返回成功。

### N003 — `tool.receipt` 混淆“批准策略”与“实际回执”

- 严重级：**HIGH**
- 原 finding：无（本轮新增；schema 语义混用）。
- r62 落点：`design.md:3946`（批准记录中的 `tool.receipt`）、`:3994`–`:3998`（语义与验签）、`:3884`（实际 stdout receipt）。
- 判定：**新增**。
- 证据与问题：
  1. `:3946` 的 `tool.receipt` 是批准记录的一部分（被 issuer/reviewer 签名），其中 `signature=""`、`issued_at=""`、`sequence="0"`、`audit_record_ref=""`。
  2. `:3996` 又要求同一 `tool.receipt` 的 `signature` 是 64 字节、`issued_at` 是 UTC 微秒、`sequence > receipt_anchors.sequence_floor`、`audit_record_ref` 验证后回填。批准发生在回执产生之前，不可能在已签名的批准记录中填入未来回执的实际值。
  3. `:3884` 的实际 stdout receipt 是 12 字段：`version, run_id, backup_ref, deployment_id, installation_id, target_digest, artifact_digest, artifact_bytes, source_approval_digest, restored_schema_digest, restored_at, receipt_digest`；与 `tool.receipt` 的 13 字段不是同一对象，且 `tool.receipt` 没有 `run_id`／`restored_at`／`receipt_digest`。
  4. `:3998` 的签名载荷是“回执正文 JCS + `{producer_identity, producer_key_id, trust_anchor_ref, sequence, issued_at}`”，但“回执正文”到底是 `:3884` 的 12 字段还是 `tool.receipt` 的 13 字段，文本没有唯一指向。
- 影响：实现者无法确定批准记录中该填占位还是实际值；验签对象与批准对象混用，可能导致“批准的字段”与“实际验签的字段”不一致，形成“校验 A、执行 B”。
- 修复建议：
  1. 拆成两个 schema：`tool.receipt_policy`（批准时固定：`trust_anchor_ref, trust_anchor_sha256, producer_principal_id, producer_key_id, signature_alg, independent_read_path, audit_sink, sequence_floor`）与 `RestoreReceipt`（运行时产生：`version, run_id, …, signature, issued_at, sequence, audit_record_ref`）。
  2. 明确批准记录只引用 policy，不包含实际 `signature`／`issued_at`／`sequence`／`audit_record_ref`；实际回执由工具/备份系统产生并签名。
  3. 在 `:3998` 写明“回执正文”就是 `:3884` 的 12 字段 strict v1 receipt，并给出完整 canonical 输入示例。

### N004 — artifact 共享 fd offset 未定义

- 严重级：**HIGH**
- 原 finding：无（本轮新增；fd 生命周期缺口）。
- r62 落点：`design.md:4010`（`RestoreSourceClaim.Artifact *os.File`）、`:4060`（`ExtraFiles` fd 4 = artifact）、`:4052`（从文件头读 ELF/schema）。
- 判定：**新增**。
- 证据与问题：
  1. `:4010` 要求 verifier 使用 `restoreSource.OpenApproved` 返回的已核 `*os.File`，不得按路径重开；`:4060` 又把同一个 `artifactFile` 作为 fd 4 交给子进程。
  2. verifier 需要从 artifact 头部读取 schema/header。若用 `Read`（而非 `ReadAt`/`pread`），共享文件 offset 会前移；子进程从 fd 4 读取时从中间开始，导致“校验的字节”与“执行的字节”起点不同。
  3. `:3888` 只说“需要 seek 时仍用该 handle”，没有要求 verifier 在读后恢复 offset，也没有要求 `execRestoreTool` 在传递前 `Seek(0, io.SeekStart)` 或使用 `ReadAt`。
  4. 同一问题也适用于工具 fd 的 ELF 头读取，但 exec 不受 offset 影响；artifact 是实际数据输入，受影响。
- 影响：子进程可能读到截断/错位的 artifact，或在某些实现中读到已校验头部之后的任意内容；`artifact_digest` 校验与子进程实际消费不一致。
- 修复建议：
  1. verifier 一律使用 `ReadAt`/`pread`，不改变共享 offset；或读后显式 `Seek(0, io.SeekStart)` 并在传递前断言 offset=0。
  2. `execRestoreTool` 在 `ExtraFiles` 前对 artifact fd 做 `Seek(0, io.SeekStart)` 并检查返回值；失败即 `ErrAdmissionBinding`。
  3. 增加负向测试：artifact 头部校验后 offset 非 0 时，子进程必须失败或读到完整内容。

### N005 — ADR-0031 r07 与 tasks 状态冲突（前置协调）

- 严重级：**BLOCKER**（前置协调；不是设计文本可自行修复）
- 原 finding：无（本轮新增；与 r01 §4.1 同一问题，r62 只在 `:4122` 登记未解决）。
- r62 落点：`decisions/framework.md:1129`；`tasks/tasks.md:3197`；`design.md:4122`。
- 判定：**新增（未闭合）**。
- 证据与问题：
  1. `decisions/framework.md:1129` 写：“`adr-0031-r06` 已 … 被接受；但本 ADR 现行修订已是 `adr-0031-r07`，r07 尚未复评 … 在 r07 被架构与安全两路接受之前，`identity-auth-S4.24a` 及任何实施/验证卡不得启动（启动门禁回到 blocked）。”
  2. `tasks/tasks.md:3197` 写：`adr-0031-r07` 为现行修订，已于 2026-11-02 被两路独立复评接受（`review-arch-r01` PASS WITH CONDITIONS；`review-sec-r01` PASS WITH CONDITIONS，报告位于 `.spec/wip/recovery-runtime/evidence-b13j-d08/`），故“本卡启动门禁解除”。
  3. 同一段又写：“r61 设计修订已被两路 r01 判 CHANGES REQUIRED … 故即使启动限制解除，本卡仍不可进入实施。”因此卡面同时存在“门禁解除”和“不可实施”两种表述。
  4. `tasks.md:3197` 引用的 `review-sec-r01` 是 18884 B / `DA757AD16C73BC3F932371B4FC342DEC32F427E05D86DB211ED4F7123CDC72DC`，位于 `evidence-b13j-d08/`；它与本对象的安全侧 r01（40500 B / `C5D5BE68…` / `evidence-b13j-s424a/review-sec-r01/`）不是同一报告。卡面文字容易被误读为“本设计 r62 也已被接受”。
  5. `design.md:4122` 已承认该冲突属“范围外、需决策”，但没有给出对账结果。
- 影响：授权状态矛盾。任何实施者/Lead 若引用 `tasks.md:3197` 的“门禁解除”作为启动依据，会与 `framework.md:1129` 的 blocked 冲突；这不是 r62 设计文本能修复的，必须由架构决策与任务规划对账。
- 修复建议：
  1. 由 ADR 维护者更新 `framework.md:1129` 的状态，或由 Task-Planner 修正 `tasks.md:3197`，二者只能有一个权威陈述；对账前不得以卡面文字主张 S4.24a 已解锁。
  2. 在 `tasks.md` 明确区分“D06/ADR r07 的决策接受级”与“S4.24a 设计 r62 的评审状态”，不要把两个 `review-sec-r01` 混用。
  3. 对账完成后，重新核 `tasks.md` 的写集（当前 4 文件，r62 至少需 11 个源文件 + 测试），否则按现写集实施必越界。

### N006 — netns／seccomp 无可用 pre-exec 机制，network 记录未绑定实际配置

- 严重级：**HIGH**
- 原 finding：无（本轮新增；与 F005／F006 的“校验 A 执行 B”相关）。
- r62 落点：`design.md:4054`（`tool.network` schema）、`:4058`（netns／hosts／route／seccomp）、`:4060`（exec）、`:4107`（`recovery_restore_exec_linux.go` 职责）。
- 判定：**新增**。
- 证据与问题：
  1. `SysProcAttr` 只支持 `Cloneflags: CLONE_NEWNET` 新建 netns，不支持 `setns(2)` 加入 `netns_path` 指向的既有 netns。要加入既有 netns，必须使用 `nsenter`、自写 wrapper 或 cgo pre-exec；设计没有选择，也没有把 wrapper 纳入批准记录/哈希/maps 白名单。
  2. seccomp 过滤器同样需要 pre-exec 安装；Go 标准库不提供。`:4058` 的 `socket`/`connect` 规则没有对应的可执行安装点。
  3. `:4054` 的 `tool.network` 只有 `mode, netns_inode, netns_path, target_addr`；`:4058` 要求的 `/etc/hosts` 身份/摘要、路由表、seccomp 规则、`/proc/<pid>/net/route` 一致性都没有进入批准记录。批准后替换 `/etc/hosts` 或路由，不会被批准层发现。
  4. `netns_path` 只要求 root-owned、no-follow；未要求 netns 由谁创建、创建后是否可被同权限主体替换、`netns_inode` 检查与 `setns` 之间是否有 TOCTOU。`:4058` 只在 exec 前/后核 inode，setns 本身没有定义。
- 影响：网络隔离可能无法实现；若用 wrapper 实现，实际执行链多了一个未批准二进制，且 maps 白名单会把 wrapper 判为“未知映射”或被迫放行，削弱 F005 的完整性。
- 修复建议：
  1. 明确 netns 加入与 seccomp 安装机制；若使用 launcher，把 launcher 作为 `tool` 的一部分（路径、SHA256、`st_dev`/`st_ino`、maps 白名单）批准。
  2. 扩展 `tool.network`：`hosts_path/hosts_sha256/hosts_owner/mode/mount`、`route_set` 或 route 摘要、`seccomp_profile_sha256`、`netns_owner`、`netns_created_by`、`setns_order`。
  3. 在 `setns` 前用 `open` + `fstat` + `setns(fd)` 固定 netns，避免路径 TOCTOU；`setns` 后再核 `/proc/self/ns/net` inode 等于记录。

### N007 — 信任根挂载属性/ACL 未与工具同级明确

- 严重级：**LOW**
- 原 finding：无（本轮新增；额外必查项）。
- r62 落点：`design.md:3929`、`:3956`。
- 判定：**新增**。
- 证据与问题：
  1. `:3929` 对信任根与批准文件规定了绝对规范路径、父目录 `0700`、文件 `0600`、root 属主、`O_RDONLY|O_CLOEXEC|O_NOFOLLOW`、fstat/路径一致、长度、UTF-8、无 BOM/NUL、只允许 LF；这些与批准文件同等严格。
  2. 但同段只写“位于不同的受控只读挂载来源”，没有像工具那样明确要求挂载同时具备 `ro`、`nodev`、`nosuid`、无 `noexec`，也没有要求核对 `mountinfo` 的 `source`/`mount_point`/`options`；`mount.source` 或 `mount_point` 至少一项不同也不足以证明两个挂载都只读。
  3. 信任根没有 ACL 摘要字段（`acl.digest` 只出现在 `tool` 记录），无法像工具那样固定 POSIX ACL；信任根包含公钥与 principal/复核记录，完整性比机密性更重要，但 ACL 可被同组写者改写。
  4. 信任根目录的逐级父目录只要求 `0700`，没有要求逐级 `lstat` 无符号链接（`:4080` 的父目录算法只覆盖 artifact 与工具）。
- 影响：信任根可在“受控只读”表述之外被替换/改写；虽然 `:3956` 已把 root/部署方同时控制登记为残余风险，但“信任根文件本身的边界”不应弱于工具。
- 修复建议：
  1. 对信任根与批准文件套用 `:4076`／`:4080` 的挂载与父目录算法：`ro,nodev,nosuid`、逐级 lstat、无符号链接、同一挂载；增加 `mount` 记录或要求从 `mountinfo` 推导并比对。
  2. 若信任根需要 ACL，增加 `acl.digest` 并与工具同算法；若不需要，明确“信任根只允许 root 属主 0600，ACL 不得存在”的拒绝条件。

## 5. 额外必查项回答

| 必查项 | 结论 | 证据/说明 |
|---|---|---|
| 新增配置键是否引入新的秘密泄漏面 | 未引入新秘密，但存在可观测面 | `HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH` 是路径、`_SHA256` 是公开摘要，不是密钥；同 uid 可经 `/proc/<pid>/environ` 读取，错误信息也可能回显路径。二者不是秘密，但路径可暴露部署结构。未发现把私钥/DSN 放进 ENV 的新增。 |
| 信任根文件本身的权限/属主/挂载约束 | 权限/属主与批准文件同级；挂载/ACL 未同级 | `:3929` 的 `0600`/root/父目录 `0700`/no-follow/pin 与批准文件一致；但未明确 `ro,nodev,nosuid` 的逐项校验，也无 ACL 摘要（N007）。 |
| 回执 `audit_sink` 追加+fsync 是否可被截断/重放 | 可截断、可窗口内重放 | `sequence_floor` 静态、审计本地文件、无外部 witness、`audit_record_ref` 未签名（N001）；`O_RDONLY` 与追加矛盾（N002）。 |
| `sequence_floor` 持久化与回滚语义 | 不足 | 只在信任根静态登记，验证成功后不推进；`audit_sink` 回滚为空后，旧回执在 `issued_at` 5 分钟窗口内仍可通过（N001）。 |
| 是否引入第二 authority／fallback／隐式默认／closed 变 open | 未发现新增第二 authority 或 closed→open | `:3925` 明确不新增第二 authority/fallback/隐式默认；`:4028` 的 `Bind` 只接同一 `*recovery.Runtime` 与同一 `anchors.Exclusive`；`:4040` 的 `install` 排除不触发 restore；`!linux` 与 `/proc` 不可用均 `ErrAdmissionUnsupported` 无 fallback；`recoveryClosedRestoreDriver` 仍是默认。 |
| 是否把生产批准主体或生产工具事实上解锁 | 否 | `:3925`、`:4004`、`:4123`–`:4129` 明确真实信任根/密钥/批准记录/见证/审计 sink/工具均不存在，生产 `VerifyRestored`／`enter`／`resume` 继续 closed；作者报告 `:147` 同结论。 |
| 是否存在“校验了 A 但执行用 B” | 存在多处 | loader cache/conf 与解释器实际读取路径未绑定（F005-1/2/3）；ENV allowlist 未落到 exec（F005-5）；maps 扫描时点不成立（F005-6）；netns/seccomp 无 pre-exec 机制（N006）；artifact fd offset（N004）；`tool.receipt` 与 `RestoreReceipt` 混用（N003）。 |

## 6. Verdict 与严重级计数

**Verdict：`CHANGES REQUIRED`**（实施与 S4.24a 关闭仍 BLOCKED）。

严重级计数（按本轮仍开放的 finding）：

- **BLOCKER 7**：SEC-F001、SEC-F002、SEC-F003、SEC-F005、N001、N005、SEC-F007（仅阻断关闭，不要求继续改设计）。
- **HIGH 6**：SEC-F004、SEC-F006、N002、N003、N004、N006。
- **LOW 2**：SEC-F008、N007。

仍阻断实施的 finding（ID + 一句话）：

- `SEC-F001`：JCS 规则与 golden vector 冲突，签名规范化不可独立复算。
- `SEC-F002`：principal/affiliations/公钥唯一性/任命记录不足，一人两名仍可构造。
- `SEC-F003`：时钟回拨无持久化高水位；`snapshot.Clear()` 与 DSN 字节比较矛盾。
- `SEC-F005`：loader cache/conf 与解释器实际读取路径未绑定，ENV/maps/network 仍有缺口。
- `SEC-F004`：`revalidateApproval` 的 DSN 副本、百分号解码缓冲、`net/url` 字符串未清零；错误摘要构成 oracle。
- `SEC-F006`：`CapBnd=0`／`NNP=1`／setns／seccomp 无可用 pre-exec 机制。
- `N001`：回执验证方无签名、producer 未独立、sequence 不推进、审计无外部见证。
- `N002`：`audit_sink` 以 `O_RDONLY` 打开却要求追加，且无锁/权限/原子性。
- `N003`：`tool.receipt` 同时表示批准策略与实际回执，字段/签名对象不确定。
- `N004`：artifact 共享 fd offset 未定义，校验与执行起点可能不同。
- `N006`：netns/seccomp 无 pre-exec 机制，network 记录未绑定实际 hosts/route/策略。
- `N005`：ADR r07 与 tasks 状态冲突，授权状态需前置对账。

不阻断设计方向但阻断关闭：`SEC-F007`（NOT RUN 诚实性已修复，关闭条件 (4)(5) 仍缺 Linux 证据）。

## 7. 未验证边界

- 未运行 Linux 上的 fd-exec、loader、netns、seccomp、ACL、mount、`AT_SECURE`、`/proc` 实测；所有宿主结论均为设计文本核对。
- 未取得真实信任根、签发/复核/执行/验证方密钥、两条批准记录、见证端点、审计 sink、旧备份或原生回执；相关信任链均未做真实攻击验证。
- 未独立实现 JCS；本次用独立 JSON 解析 + UTF-16 键序比对复算，已发现向量冲突，但没有跑第三方 JCS 库交叉验证。
- 未评审实现代码；未跑 Go 测试/build/vet/lint；未验证 `SysProcAttr` 字段与 Go 版本行为（`CapBnd`/NNP/setns/seccomp 的结论基于标准库无 pre-exec 钩子的事实）。
- 未逐条复核架构侧 r01 全文；只读与安全结论直接相关的材料。
- 未读取 `design.md` 全文；只读 r60/r61/r62 与定向 restore 章节、ADR-0031 r07、ADR-0032、`tasks.md:3186`–`:3218`、作者报告、r01 全文。
- 未做人的接纳、生产批准或 S4.24a 关闭判断；本报告不替代 Lead 复核、后续授权或真实 E2E。

## 8. 复审条件

1. r62 修订为 r63（或后续 revision），重新计算并固定 `design.md` 字节/行数/SHA256。
2. 修复 F001 的 JCS 向量，并由独立实现复算两向量；提供 canonical 字节长度、摘要与负向向量。
3. 修复 F002 的 `person_name` 矛盾、affiliations 类型、公钥唯一性、principal 撤销/有效期、review 签名、appointment 记录。
4. 修复 F003 的持久化时钟高水位、snapshot/DSN 比较语义、revalidate 次数/顺序、审计 result 枚举。
5. 修复 F004 的 DSN 副本清零清单、`net/url`/JSON 解码字符串、错误 oracle、panic/fmt redaction、canary 测试。
6. 修复 F005 的 loader 实际路径绑定、include/glob、解释器依赖、ENV allowlist、maps 扫描语义、preload 路径边界、network schema。
7. 修复 F006/N006 的 pre-exec 机制（launcher/cgo 方案）、固定降权顺序、netns/seccomp 安装与批准记录。
8. 修复 N001–N004 的回执/审计 schema 拆分、fd 打开模式、artifact offset、外部见证与 sequence 推进。
9. N005 由 ADR 维护者与 Task-Planner 对账；对账前不得启动 S4.24a。
10. 复审时附原 finding、修复修订、相关回归与作者证据报告；本报告只复核原 finding 与新增安全面，不替作者修复。

## 9. 声明

本报告是安全侧独立评审结论，不是作者自评、不是人的接纳、不是生产批准、不是 S4.24a 关闭证据。评审期间未修改被评对象 `design.md`、未修改代码、未修改 `tasks.md`、未修改 ADR、未修改 `.add/**`；未做 Git 写、删除、移动、归档或清理；工作区未提交改动原样保留。报告内只使用仓库相对路径。