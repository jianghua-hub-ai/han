# B13J-S4.24a 架构侧独立评审 r02（r62 修订复审）

## 0. 对象、基线与复算

评审对象为 `.add/project/knowledge/design-workspace/identity-auth/design.md`（复算 936613 B / 5317 行 / SHA256 `EA880660B8D43AFD6FB227812C662BB5A4581B39F4235F3DAB8E221F5F9FDC9A`，`revision: identity-auth-design-ir01-r62`，见 `design.md:5`；r62 增量位于 `:3925`–`:4129`）；评审基线为 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r01/report.md`（复算 56625 B / 359 行 / SHA256 `78489013177ED00D460B3AA3A6E5EFD810CF2292CDE4C30B2E508D8F8C16C7DA`，verdict `CHANGES REQUIRED`，BLOCKER 5 / HIGH 11 / MEDIUM 4）。本次评审范围为：判定 r62 是否真正闭合上述 20 条 finding、沿真实消费者与上游依据做有界复核、识别新引入问题；不评价实现、不运行代码、不构成 S4.24a 关闭或生产批准。

作者证据报告：`.spec/wip/recovery-runtime/evidence-b13j-s424a/design-revision-r02/report.md`，复算 18821 B / 147 行 / SHA256 `A910B3DD42223025B7D7D7C1CE80CD1D366E582BDDA2E13E8A415E4E660A0549`。

**复算结论**：三项对象的字节数、行数与 SHA256 均与派发值逐项一致，无偏差，进入实质评审。

固定工作区 `Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，工作树 295 条未提交改动；本次未改动其中任何内容。

## 1. 评审方法与两轮实际检查范围

**第一轮（对象绑定 + 反向检索）**：逐条把 F001–F020 绑定到 r62 的精确定位（`:3929`–`:4096`），并对全文反向检索 `approval_digest`、`RestoreSourceClaim`、`RestoreSourceVerifier`、`appendJSONString`、`Bind`、`authority`、`ownership`、`NOT RUN`、`未登记依赖` 等关键词，确认是否仍有旧表述、被静默删除的约束或未标注的取代关系。

**第二轮（独立复算 + 消费者/上游有界复核）**：

- 独立复算两条 JCS golden vector 的字节数与 SHA256，并按 `:3935` 自己规定的键序重排后再次复算；
- 读取 `tasks.md:3186`–`:3218`（只读）核对写集与关闭条件；
- 读取 `cmd/han-server/main.go:171`–`:199`、`:396`–`:446`、`:629`–`:634`；`cmd/han-server/recovery_runtime.go:456`–`:535`、`:555`–`:572`、`:596`–`:599`、`:641`–`:660`；`cmd/han-server/recovery_restore.go:40`–`:70`、`:147`–`:152`、`:169`–`:216`、`:269`–`:286`；`internal/recovery/runtime.go:146`–`:155`、`:205`–`:215`、`:227`–`:245`、`:870`；
- 读取 `framework.md` 的 ADR-0031 不变量 6（`:1147`）、约束 8（`:1160`–`:1169`）、ADR-0032 的 `exclusive` 继承关系（`:1201`）与 r60 `design.md:3603`/`:3616`；
- 读取 r01 完整正文（359 行）并逐条比对。

**未执行**：Go build/test/vet/gofmt、任何 Linux 命令、任何独立 JCS 库、任何真实签名/信任根/批准记录/见证/审计 sink。

## 2. 结论

**Verdict：`CHANGES REQUIRED`**。

| 严重级 | 计数 | 仍未闭合/新增 |
|---|---:|---|
| BLOCKER | 4 | F001、F005、F006、F018 |
| HIGH | 3 | F004、F011、F017 |
| MEDIUM | 3 | NEW-R02-06、NEW-R02-07、NEW-R02-08 |

已闭合（12）：F002、F003、F007、F008、F010、F012、F013、F014、F015、F016、F019、F020。

设计侧已处置、但不构成卡关闭（不计入上述计数）：F009。

待决策项（非缺陷）：ADR-0031 r07 状态冲突（`design.md:4122`）。

**仍阻断实施**：F001、F005、F006、F018、F004、F011、F017。

**为什么不给 PASS**：5 个原 BLOCKER 中只有 F007 真正闭合；F001 的验收向量与其自身规范化规则矛盾，F005/F006 按文本无法写出可编译且不产生第二 lease 的实现，F018 的回执签名载荷无法被独立复算。设计篇幅完整不等于契约唯一，实施者仍必须自行发明关键细节。

### 2.1 逐条处置总表

| ID | 原级 | r62 落点 | 判定 | 一句话理由 |
|---|---|---|---|---|
| F001 | BLOCKER | `:3929`–`:3949` | 未闭合（部分修订） | golden vector 键序违反 `:3935` 自身规则，摘要不可被独立 JCS 实现复算 |
| F002 | HIGH | `:3933`、`:4190`、`:4192` | 已闭合 | `sha256:` + 64 位小写 hex 统一；裸 hex 仅存在于 ENV 边界并被规范化 |
| F003 | HIGH | `:3933`、`:3950`、`:3958`、`:4050` | 已闭合（原三项） | tool keyset、`tool.approval_digest` 消费者、`artifact_bytes` 三方映射均补齐 |
| F004 | HIGH | `:4008` vs `:3603` | 部分闭合 | 只在 r62 写限定，r60 原文仍禁止，无优先句 |
| F005 | BLOCKER | `:4010`–`:4026` | 部分闭合 | 制品 fd 读头后未复位；claim 字段类型与包边界矛盾 |
| F006 | BLOCKER | `:4028` | 部分闭合 | `Bind` 收到的是 provider 而非已取得 lease，类型不成立 |
| F007 | BLOCKER | `:3980`–`:3986` | 已闭合 | wire `string` + `appendJSONString` + 五处清零自洽 |
| F008 | HIGH | `:4030` | 已闭合 | 测试替身与生产 `restoreExecTool` 分流，`!linux` 门禁不作用于测试 |
| F009 | HIGH | `:4032`、`:4118`、`:4129` | 设计侧已处置；卡关闭条件未满足 | NOT RUN 未被写成 PASS，但 S4.24a (4)(5) 仍不可关闭 |
| F010 | HIGH | `:4034` | 已闭合 | 解析/语义层与 POSIX host 层分层明确 |
| F011 | HIGH | `:3976`–`:3978` vs `:4000` | 部分闭合 | 八步流程缺少 execute 后、解码前的 revalidate |
| F012 | MEDIUM | `:4076`–`:4080` | 已闭合 | mountinfo、ACL、父目录算法补齐 |
| F013 | MEDIUM | `:3992` | 已闭合 | DSN 文法与错误映射补齐 |
| F014 | MEDIUM | `:4060`–`:4075` | 已闭合 | lease/exec 签名与 fd 生命周期补齐 |
| F015 | HIGH | `:4036` | 已闭合 | 四键、唯一预检顺序、plan 只消费快照 |
| F016 | MEDIUM | `:4098`–`:4116` | 已闭合 | 完整文件表与测试文件表 |
| F017 | HIGH | `:4038`、`:4116`；`tasks.md:3191`–`:3195` | 未闭合 | tasks 仍是 4 文件写集，实施即越界 |
| F018 | BLOCKER | `:3994`–`:4004` | 部分闭合 | 回执签名载荷、审计 sink 模式、序列基准、回填规则不可唯一实现 |
| F019 | HIGH | `:4040` | 已闭合 | 按 action 固定 nil 语义，与现有代码一致 |
| F020 | HIGH | `:4042` | 已闭合 | 预检错误 `errors.As` → 2；其余 → 1，与现有映射一致 |

## 3. 范围完整性

- **增量落点**：r62 实际新增/改写集中在 `:3925`–`:4129`；`:4131` 起为 r61 正文，完整保留，未发现静默删除。
- **就地取代标注**：r61 中被取代的句子均有括注，已逐行核对 `:4190`、`:4192`、`:4261`、`:4278`、`:4284`–`:4289`、`:4293`、`:4295`，方向一致。
- **更早章节冲突**：r60 `:3603`（“未在本表内登记的依赖不得注入 production root”）与 `:3616`（`recoveryRootInputs.exclusive` 仅为测试注入、生产不得替代 registry）原文未改，与 r62 `:4008`/`:4028` 仍冲突（见 F004、F006）。除此之外，未发现 r52/r60 被新增矛盾。
- **第二 authority / fallback / 隐式默认 / closed→open**：未发现新的 fallback、隐式默认或把 closed 改为 open；`recoveryClosedRestoreDriver` 仍是默认（`recovery_runtime.go:596`–`:599`）。但 F006 的 `Bind` 类型矛盾会迫使实施者自行 `Acquire` 第二 lease，从而在事实上引入第二 ownership。
- **NOT RUN 是否被写成 PASS**：没有。`:4032`、`:4118`、`:4129` 均明确 NOT RUN 与“本设计通过评审也不构成 S4.24a 关闭”。本报告同样不构成关闭。
## 4. 逐条 finding

### F001 — BLOCKER — 批准记录 digest 的 canonicalization 不唯一（未闭合，部分修订）

- **原 finding**：r61 只给“固定键序 + 紧凑 JSON”，未固定嵌套键序、转义、数字格式、时间严格度；两个独立实现无法得到同一 `approval_digest`。
- **r62 落点**：`:3929`–`:3949`（信任根、Ed25519、JCS 规则与两条 golden vector）。
- **判定**：**未闭合**。信任链结构上的“自证”问题已解决；canonicalization 仍有多解，验收向量自相矛盾。

**证据（独立复算）**：

`:3935` 规定对象键按 UTF-16 code unit 升序，并声明“本项目全部键为 ASCII，等价于字节升序”。据此，`reviewer` 与 `revocation` 的第 4 个字符分别为 `i`(0x69) 与 `o`(0x6F)，故 `reviewer` 必须先于 `revocation`。

但两条向量均把 `revocation` 写在 `reviewer` 之前：

- `:3940` source 向量：`"revocation"` 首次出现索引 450，`"reviewer"` 索引 518；
- `:3946` tool 向量：`"revocation"` 索引 1734，`"reviewer"` 索引 1802。

复算结果：

| 向量 | 字节数 | 文中声明摘要 | 对文中原字节复算 | 按 `:3935` 键序重排后复算 |
|---|---:|---|---|---|
| source `:3940` | 1115 | `3c7b1cff…1059e9`（`:3943`） | `3c7b1cff97b230077c9d3d75085082d26bfe5e48b8355cb721e0a830f51059e9` | `de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab` |
| tool `:3946` | 2357 | `9df40264…205d68`（`:3949`） | `9df40264e869c7ed9506b3ac5788f5878cf516c980082ffa64bedb1ffb205d68` | `8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508` |

即：文中 `approval_digest` 确实等于对“文中那串非 canonical 字节”求得的摘要（作者是自洽的），但这串字节本身违反 `:3935` 的键序规则。任何合规 JCS 实现都会先输出 `reviewer`，得到上表右列的摘要，而不是文中声明的摘要。`:3949` 又要求“独立实现必须复算出同一字节与同一摘要”，该验收条件在现文本下不可满足。

**对派发问题 1 的直接回答**：

- 外部信任根 + pin + Ed25519 + RFC 8785 JCS 在**信任链结构上**不再自我循环：`approval_digest` 不再是唯一真实性依据，签名由批准文件之外的 pin 固定公钥验证；这一半成立。
- 但“**唯一可编码**”不成立：规范化规则与验收向量矛盾，canonicalization 仍至少有两解（文中一解、合规 JCS 一解）。
- “信任根缺失是否真 closed”：是。`:3954` 明确缺失/不可读/摘要不符/`anchor_ref` 缺失 → `ErrAdmissionIncomplete`，且无“未签名也接受”分支。

**影响**：实施者若照抄向量，会写出非 JCS 实现；若实现真 JCS，则无法通过文中验收。两边都不满足 `:3949`。

**修复方向**：交换两条向量的 `reviewer`/`revocation` 顺序并重算 `:3943`/`:3949`；或明确标注向量仅示意、另给合规向量。验收必须由**非作者**的独立 JCS 实现复算。

**作者自证风险**：作者报告 `:35`、`:110` 自认 golden vector 仅作者本地复算。该自认正确，本评审据此不把作者复算当作独立证据；本报告的复算也只是静态键序 + 字节哈希，不是独立 JCS 库实现（见 §6）。

### F002 — HIGH — `approval_digest` 裸 hex 与 `sha256:` 前缀契约冲突（已闭合）

- **原 finding**：r61 同时存在裸 hex 与 `sha256:` 前缀两种表示。
- **r62 落点**：`:3933`；取代标注 `:4190`、`:4192`；交叉引用 `:4044`。
- **判定**：**已闭合**。

**证据**：`:3933` 规定 `approval_digest` 唯一表示为 `sha256:` + 64 位小写十六进制；`HAN_RECOVERY_RESTORE_EXECUTABLE_SHA256` 仅在环境变量边界接受裸 hex，进入进程后立即规范化；批准记录、`RestorePlan`、`RestoreReceipt`、信任根与回执中一律只接受 `sha256:`，裸 hex/大写 hex → `ErrAdmissionBinding`。`:4192` 与 `:4218` 已就地标注被取代。代码侧 `internal/recovery/replay_status_v1.go:311`–`:314` 的 `SHA256Digest` 返回 `"sha256:"+hex`，`recovery_restore.go:147`–`:151` 的 `restoreReceiptDigest` 使用它，与设计方向一致。

全文检索未发现残留的“批准摘要用裸 hex”表述；`:1671`/`:1719`/`:3884` 只把 `source_approval_digest`/`receipt_digest` 当字段名引用，不构成表示法冲突。

### F003 — HIGH — `tool` 记录缺 loader 字段、`tool.approval_digest` 无消费者、`artifact_bytes` 映射缺失（原三项已闭合）

- **原 finding**：同上三项。
- **r62 落点**：`:3933`、`:3950`、`:3951`、`:3958`、`:4050`。
- **判定**：**已闭合（原三项）**；残余字段名映射问题另见 NEW-R02-08。

**证据**：

- tool keyset：`:3933` 明确 `tool` 另含 `executor, elf, network, security`，`:3950` 固定 `tool.elf`、`tool.network`、`tool.security` 的 keyset，`:4050` 给出 `tool.elf` 完整字段与判定顺序。
- `tool.approval_digest` 消费者：`:3958` 明确由 `RestoreSourceClaim.ToolApprovalDigest` 携带，由 `RestoreSourceVerifier` 与批准快照逐字节比较，并由 driver 每次重校验确认未变化；`tool.approval_ref` 同理。
- `artifact_bytes` 映射：`:3958` 固定“批准记录 `source.artifact_bytes` = `RestorePlan.ArtifactBytes` = 制品实际字节数”，三者必须相等；`:4024` 再要求 verifier 从 artifact 头部读取并核对。

三个原缺口均有唯一落点，判定闭合。

### F004 — HIGH — r61.3 与 r60“未登记依赖不得注入 production root”文字冲突（部分闭合）

- **原 finding**：`design.md:3603` 明文“未在本表内登记的依赖不得注入 production root”，而 r61.3 允许新 verifier 进入 root 装配链。
- **r62 落点**：`:4008`（r62.6.1）。
- **判定**：**部分闭合**。

**证据**：

- `:4008` 给出实质解释：`RestoreSourceVerifier`、批准快照、工具句柄、目标 DSN 是 `controlledRestoreDriver` 的**构造期内部依赖**，不是端口，不得放入 `recoveryRootInputs`，不得加入 `RuntimeDependencies`。
- 但 `:3603` 原文未改，仍写“未在本表内登记的依赖不得注入 production root”，且 r62 未给“以 r62.6.1 为准”的优先句。
- 对比：`:4040`（F019）明确写了“r61.3 的表述由本段取代……以本段为准”。F004 没有同等机制。
- `:4008` 自己也写“本限定只写在 r62，不改 r60 正文与 ADR-0032”，等于承认文本冲突保留。

**影响**：读者/实施者无法从文本判定 r60 的禁止是否仍适用于 verifier；审计时也无法判断哪句是现行规范。

**修复方向**：在 `:3603` 加窄例外（“构造期内部依赖不属于本表所指的端口/可配置依赖”），或在 `:4008` 增补与 `:4040` 同形的“以本段为准”优先句，并同步 ADR-0032 的继承说明。

### F005 — BLOCKER — `RestoreSourceVerifier` 无法独立核验实际制品；无调用时机与 plan→claim 映射（部分闭合）

- **原 finding**：claim 只有 digest/bytes，没有制品句柄/路径；接口无调用时机、无 plan→claim 映射；生产 verifier 无装配路径。
- **r62 落点**：`:4010`–`:4026`（claim、调用时机、映射表、头部核验、错误映射）。
- **判定**：**部分闭合**。调用时机与映射表已补齐，但制品句柄语义与类型定义仍不可实施。

**已补齐部分**：

- `:4010` 给 `Artifact *os.File`，要求 verifier 不得按路径重开；
- 调用时机：`Prepare` 在 `OpenApproved` 后、返回 plan 前；`Replace` 在 `revalidateApproval` 后、启动前；`VerifyRestored` 在 `prepared()` 后、启动前；
- `:4014`–`:4022` 给 plan→claim 逐字段映射；
- `:4024` 要求 verifier 读取 artifact 头部（`han-db-restore/v1` strict JSON，最长 4096 字节），比对 `schema_digest`/`installation_id`/`backup_ref`/`payload_bytes`/`payload_sha256`，并做全文件摘要核验；
- `:4026` 给错误分类。

**仍未闭合部分**：

1. **fd offset 未复位**。`:4021` 明确“调用方不关闭”该 `*os.File`；`:4024` 要求 verifier 读取头部；`:4060`/`:4074` 又用同一个 `artifactFile` 作为子进程 fd 4。全文未规定 verifier 读头后 `Seek(0)`、改用 `ReadAt` 或 `dup` 独立偏移。子进程会从头部末尾开始读，得到截断制品。代码先例明确要求复位：`cmd/han-server/recovery_restore.go:269` 与 `:285` 在摘要计算前后各 `Seek(0, io.SeekStart)` 一次。这是可复现的实现缺陷，不是风格问题。
2. **claim 字段类型矛盾**。`:4010` 内联结构把 `ArtifactBytes` 放在 `string` 组（`{RunID, ..., ArtifactBytes, ... string; ...}`），而 r61.3 `:4231` 定义为 `int64`，`:4017` 又映射自 `RestorePlan.ArtifactBytes`（`int64`）。照 r62 写会变成字符串，与映射表和 `ArtifactBytes > 0` 语义冲突。
3. **字段集被静默收窄**。`:4010` 的“扩展为”列表未含 r61.3 `:4224`/`:4229` 的 `Version` 与 `TargetDigest`；若按字面实现，claim 不再比较这两项，等于削弱核验；若只是省略，则文本不完整。
4. **包边界不可编译**。`RestoreSourceClaim` 位于包 `recovery`（`internal/recovery/restore_source.go`，见 `:4112` 与 r61.3 `:4220`–`:4240`），却被要求嵌入 `Approval restoreApprovalSnapshot; Tool restoreToolSnapshot`。这两个名称在现有代码中不存在，且按 `:4103` 将由 `cmd/han-server/recovery_restore_approval.go` 解析；若它们定义为 `cmd/han-server` 的非导出类型，包 `recovery` 无法引用；若定义为 `recovery` 的非导出类型，`cmd/han-server` 又无法构造。必须明确为跨包可用的导出不可变类型，或把快照移出 claim。

**影响**：B2“独立核验实际制品”的方向正确（句柄 + 头 + 全文件摘要），但按现文本实现会截断制品或无法编译。

**修复方向**：规定 verifier 读头后 `Seek(0)`（或 `ReadAt`/`dup`）并给出失败路径；统一 `ArtifactBytes` 类型；恢复 `Version`/`TargetDigest`；明确快照类型的包与可见性。

### F006 — BLOCKER — 构造签名与生产 `authority`/`ownership` 接线（部分闭合）

- **原 finding**：`newControlledRestoreDriver` 只加 verifier 形参，生产 `authority`/`ownership` 无设置路径；driver 需要 Runtime、Runtime 又需要 driver，形成循环。
- **r62 落点**：`:4028`（r62.6.3 两阶段装配与 `Bind`）。
- **判定**：**部分闭合**。`authority` 侧方向正确；`ownership` 侧类型与来源不成立。

**证据**：

- `:4028` 定义 `Bind(authority restorePhaseAuthority, ownership interface{ Check(context.Context) error }) error`。
- `restorePhaseAuthority` 为 `CheckPhase` + `ReadPermitBinding`（`cmd/han-server/recovery_restore.go:169`–`:172`），`recovery.Runtime` 接口具备这两个方法（`internal/recovery/runtime.go:227`–`:237`），故 `Bind(runtimeValue, …)` 的 authority 参数可以成立，“同一 `*recovery.Runtime`、不包装”的要求可编码。
- 但 `anchors.Exclusive` 的静态类型是 `recovery.DeploymentExclusive`（`cmd/han-server/recovery_runtime.go:555`–`:567`、`:647`），其唯一方法是 `Acquire(context.Context) (ExclusiveLease, error)`（`internal/recovery/runtime.go:146`–`:149`）；`Check` 在 `ExclusiveLease` 上（`:151`–`:155`）。因此把 `anchors.Exclusive` 传给 `interface{ Check(context.Context) error }` **无法编译**。
- 更关键的是“root 已持有的同一 lease”不成立：lease 是在 `Runtime.RunNormal` 内部 `r.deps.Exclusive.Acquire(ctx)` 取得的（`internal/recovery/runtime.go:870`），保存在私有字段 `r.exclusive`；`Runtime` 接口（`:227`–`:245`）没有任何取回 lease 的方法。assembly 在 `Bind` 时没有已取得的 lease。
- 若 driver 在 `Bind` 后自行 `Acquire`，就产生第二个 lease/第二个 ownership，直接违反 `:4028` 的“同一 lease、不得新建”。
- 此外，`:4028` 说 `ownership` 必须是 `recoveryRootInputs.exclusive` 对应的同一 lease；r60 `:3616` 与 ADR-0032 `framework.md:1201` 都明确 `recoveryRootInputs.exclusive` 只是合成/测试注入、生产不得用它替代 registry。生产 `inputs.exclusive` 为 `nil` 时，`anchors.Exclusive` 是 `recoveryClosedPort`（`cmd/han-server/recovery_runtime.go:565`–`:567`），而 `recoveryClosedPort` 只实现 `Acquire`（`cmd/han-server/recovery_sources.go:523`–`:525`），同样不满足 `Check`。

**影响**：按文本写不出可编译且不产生第二 lease 的接线；实施者只能二选一：类型断言/包装（违反“不得包装”），或自行 `Acquire`（引入第二 ownership）。这正是原 finding 指出的循环未解。

**修复方向**：把 `Bind` 的 `ownership` 改为接收 `recovery.ExclusiveLease`（或其最小 `Check` 子接口），由 assembly 在唯一 `Acquire` 点取得后传入；明确唯一 `Acquire` 在 Runtime 内还是 assembly 内，并保证只有一处；同时消除与 r60/ADR-0032 关于 `inputs.exclusive` 的表述冲突。

### F007 — BLOCKER — `dsn []byte` 经 `json.Marshal` 变 base64；清零时点不唯一（已闭合）

- **原 finding**：`[]byte` 会被编码为 base64，与“线上形状仍是字符串”矛盾；清零同步点未定义。
- **r62 落点**：`:3980`–`:3986`；取代标注 `:4261`。
- **判定**：**已闭合**。

**证据**：

- `:3980` 明确作废 r61.4 的 `[]byte` 字段方案；
- `:3982` 固定 wire 字段仍为 `string`，禁止 base64、`json.RawMessage`、第二个 DSN 字段；
- `:3984` 用 `json.Marshal(planWire(p))` + 项目内 `appendJSONString(dst, src []byte)` 直接追加 DSN 字节，不经 `string(dsn)` 中间副本，并规定转义规则与“解码后逐字节等于输入”；
- `:3986` 列五处清零：`execute` 的 `defer clear(input)`、`runChild` 写后清零、`Close` 清零 `d.dsn`、预检快照 `Clear()`、子进程解析后清零；
- 残余风险（GC 复制、管道缓冲、core dump）明确登记，不冒充“绝不泄露”。

“线上形状不变”的矛盾已通过“wire 仍为 string + 自定义编码”消除。`:4261` 的就地括注与 `:3980` 一致。

### F008 — HIGH — `!linux` exec 门禁与既有 Windows 子进程测试冲突（已闭合）

- **原 finding**：`!linux` 的 `execRestoreTool` 一律 unsupported，会破坏 Windows 上真实运行子进程的既有测试。
- **r62 落点**：`:4030`。
- **判定**：**已闭合**。

**证据**：`:4030` 明确 `restoreProcessTool` 是**软件合同测试替身**，只用于既有测试；生产 exec 由新增 `restoreExecTool` 实现（`recovery_restore_exec_linux.go` / `_other.go`）；`!linux` 门禁只作用于 `restoreExecTool`，不作用于 `restoreProcessTool`，既有 Windows 测试继续使用 `imageConfiguration` 的 `SystemRoot`/`WINDIR` 允许列表；生产路径不得回退到测试替身。分流边界清楚。

### F009 — HIGH — 负向 (d)(e)(f) 标 NOT RUN 不能关闭 S4.24a (4)(5)（设计侧已处置；卡关闭条件未满足）

- **原 finding**：NOT RUN 不能当作 PASS，关闭条件 (4)(5) 需要真实证据。
- **r62 落点**：`:4032`、`:4118`、`:4129`。
- **判定**：**设计侧已处置（原设计缺陷闭合）；但 S4.24a 的关闭条件仍未满足，不计入设计 finding 计数**。

**证据**：

- `:4032` 明确“本设计通过评审也不构成 `identity-auth-S4.24a` 关闭”；关闭条件 (4)(5) 需要 Linux 目标或 CI 上真实负向 (d)(e)(f) 与 9 条验收命令证据；Windows 上的 skip/无匹配 exit 0 不是 PASS。
- `:4118` 再次声明 (d)(e)(f) 在 Windows 结构性不可达，标 NOT RUN。
- `:4129` 收尾声明不构成评审通过、生产可用、`enter`/`resume` 解锁或 S4.24a 可关闭。
- `tasks.md:3191`–`:3218` 的关闭条件仍为 7 项，(4)(5) 未豁免 NOT RUN；`tasks.md:3217` 记录本机 Windows 结构性不可达，卡仍 BLOCKED。

**结论**：r62 对 NOT RUN 的表述诚实，没有把它写成 PASS。但这也意味着 **S4.24a 现在不可关闭**，需要 Linux 环境与真实信任输入补证。本报告不构成关闭。

### F010 — HIGH — Windows 上 owner/root/0700 不可达，与“真实可触发”冲突（已闭合）

- **原 finding**：Windows 无法真实触发 POSIX owner/root/0700，却声称 (a)(b)“真实可触发”。
- **r62 落点**：`:4034`。
- **判定**：**已闭合**。

**证据**：`:4034` 把负向 (a)(b) 拆成两层：平台无关的解析/语义层（JSON、字段、摘要、principal、review、DSN 语法、时间关系）在 Windows 可真实触发，断言具体 `ErrAdmission*`；POSIX host 层（owner/root/0700、ACL、mountinfo、loader、fd-exec）标 Linux-only，Windows 只断言 `ErrAdmissionUnsupported` 且零状态变更。不再声称 Windows 走完整文件边界“真实可触发”。分层与 `:4118` 一致。
### F011 — HIGH — 撤销与时间重取在子进程 in-flight 段未闭合（部分闭合）

- **原 finding**：批准快照、有效期与撤销状态在 `Prepare`→`Replace` 之间不闭合，批准被撤销后仍可能成功。
- **r62 落点**：`:3968`–`:3978`（r62.3）与 `:4000`（VerifyRestored 八步）。
- **判定**：**部分闭合**。设计新增了重读点，也明确识别了 in-flight 风险，但八步流程没有编码 `:3978` 自己要求的最后一次重校验。

**证据**：

- `:3976` 列出五个读取/时钟点：启动预检、`Prepare`、`Replace` 的 `d.mu` 内与启动前、`VerifyRestored` 的 `prepared()` 与启动前、子进程启动回调。它们都不覆盖“子进程已返回、回执尚未解码”这一时点。
- `:3978` 明确要求 `execute` 在子进程返回后、解码回执前再调用 `revalidateApproval`；失败返回 `errors.Join(recovery.ErrRecoveryWriteUnknown, err)`。
- 但 `:4000` 的 `VerifyRestored` 八步是：revalidateApproval → source verify → execute → 独立读取 → 验签 → sequence 检查 → 审计 fsync → 回填；第 3 步 `execute` 之后没有再次 revalidate，直接进入第 4 步。撤销若发生在子进程运行期间，按八步会漏检；若实现者按 `:3978` 补一次，则与八步清单不一致，产生第二个未定义的执行顺序。
- `:3978` 的“下一次 revalidate 生效”只说明最长延迟等于检查点间隔；没有把 execute 后检查点纳入完整调用图，因此仍不能证明撤销在 in-flight 窗口内被观察到。

**影响**：TOCTOU 没有真正闭合。实施者无法从两条互相矛盾的规范中确定唯一顺序；若按八步实现，撤销可在子进程启动后继续被当作成功回执处理。

**修复方向**：在 `:4000` 八步中显式加入“execute 返回后、解码回执前 revalidateApproval”，并在 `:3976` 的读取点清单中登记该点；失败必须按 `restore_unknown` 分类，不能进入审计成功路径。

### F012 — MEDIUM — mountinfo、ACL 与父目录算法（已闭合）

- **原 finding**：r61 只给出短语式宿主文件边界，缺少 mountinfo 字段、ACL 文本编码与父目录逐级校验的可编码算法。
- **r62 落点**：`:4076`–`:4080`（r62.7.4）。
- **判定**：**已闭合**。

**证据**：

- `:4076` 固定 mountinfo 分列、字段顺序、`\040`／`\011`／`\012`／`\134` 转义、其他 `\NNN` 非法的规则；最长 `mount_point` 前缀、`mount_id` 并列决胜、相同 `mount_id` 视为 `ErrAdmissionBinding`，并分别给出工具、解释器、依赖与 artifact 的挂载选项要求。
- `:4078` 固定 POSIX ACL tag 数值排序、同 tag 的 qualifier 排序、规范文本 `tag:qualifier:perm`、perm 小写十六进制、mask 一致性、重复条目与未知 tag 的失败分类，以及摘要算法。
- `:4080` 固定从挂载根到父目录逐级 `lstat`、root-owned、`mode & 0022 == 0`、同一 `ro` 挂载、`O_NOFOLLOW` 后 `fstat` 与打开前 `lstat` 一致，并明确 artifact 与工具共用父目录规则。

**限制**：这些是契约与静态算法，未在 Linux 上运行 fixture；本轮不把“可编码”误报为“已实测”。

### F013 — MEDIUM — restore 目标 DSN 文法与错误分类（已闭合）

- **原 finding**：目标 DSN 只写了“由批准文件读取”，没有唯一文法、参数白名单或失败分类，实施者可能自行实现宽松解析或回退到业务 DSN。
- **r62 落点**：`:3992`（r62.4）。
- **判定**：**已闭合**。

**证据**：`:3992` 指定唯一入口 `parseRestoreTargetDSN`，只接受 `postgres://` 与 `postgresql://`，要求显式端口、非空 host/database/user，只允许 `sslmode=verify-full`、`connect_timeout=10`、`application_name=han-recovery-restore` 三个固定参数；禁止重复、未知、空值及 `hostaddr`、`service`、`passfile`、`options`、`search_path`；userinfo 只允许 `user:password`，百分号编码必须合法且解码后无 NUL/CR/LF。失败统一 `ErrAdmissionBinding`，在 Begin 前返回，并明确禁止回退 `HAN_DATABASE_DSN`、从连接反推或先连库试探。

### F014 — MEDIUM — host lease 与 exec 端口签名（已闭合）

- **原 finding**：`validateRestoreHost`/`execRestoreTool` 没有明确 lease、artifact fd 生命周期与测试替身边界，实施者可能校验后按路径重开。
- **r62 落点**：`:4060`–`:4074`（r62.7.3）。
- **判定**：**已闭合**。

**证据**：

- `:4065`–`:4071` 给出 `restoreHostLease`（`ToolFile()`/`Close()`）、`validateRestoreHost` 与 `execRestoreTool` 的签名；artifact 作为独立 `*os.File` 参数传入，不进入 lease。
- `:4074` 明确 lease 只拥有工具 fd、`execRestoreTool` 不得接受路径或重开工具/artifact；`defer lease.Close()` 活过 `Wait` 与 receipt 解析；artifact fd 由 driver 生命周期关闭。
- 同一段明确 `openImage` 的现行路径打开由 lease 取代，`restoreProcessTool.openImage` 只保留给测试替身，生产 `restoreExecTool` 不调用它。

### F015 — HIGH — 新配置键、唯一预检与启动顺序（已闭合）

- **原 finding**：r61 的新配置键、读取者与预检/plan 顺序没有接到真实启动链，可能形成第二读点或使 plan 再次读取输入。
- **r62 落点**：`:4036`（r62.6.7）。
- **判定**：**已闭合**。

**证据**：

- `:4036` 新增四键 `HAN_RECOVERY_RESTORE_APPROVAL_PATH`、`HAN_RECOVERY_RESTORE_TARGET_DSN_FILE`、`HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_PATH`、`HAN_RECOVERY_RESTORE_APPROVAL_TRUST_ROOT_SHA256`，并明确加入 `cmd/han-server/recovery_config.go:58`–`:83` 的 `recoveryRootConfigurationKeys`。
- `actionRequiresRestore` 被限定为 `action == "enter" || action == "resume"`，并说明 `enter --installation-manifest` 已映射为 `action="install"`；install/normal/verify 不读取四键。
- 唯一预检点固定在 `runComposed`（`main.go:396`）：`loadStartupConfig`（`:400`）与 `clearStartupSecrets`（`:432`）之后、`parseDatabase`（`:442`）之前；仅当 `dependencies.recoveryCommand != nil && actionRequiresRestore(...)` 时调用。
- `planRecoveryStartup`（`main.go:629`）改为只消费不可变快照，不得再次读批准文件、DSN 文件或信任根；`clearStartupSecrets` 后调用 `snapshot.Clear()`。
- 代码侧 `main.go:400`、`:432`、`:442`、`:629` 与 `recovery_config.go:58`–`:83` 的真实顺序和 keyset 与该描述一致；未发现第二预检读点。

### F016 — MEDIUM — 完整文件表与影响面（已闭合）

- **原 finding**：r61 的影响面只列少量文件，无法覆盖 host/exec/approval/source 等新增契约。
- **r62 落点**：`:4098`–`:4114`（r62.9.1）。
- **判定**：**已闭合**。

**证据**：`:4098`–`:4112` 给出生产文件表，覆盖 `recovery_restore.go`、`recovery_restore_approval.go`、`recovery_restore_target.go`、两个 host 平台文件、两个 exec 平台文件、`recovery_runtime.go`、`main.go`、`recovery_config.go` 与 `internal/recovery/restore_source.go`；`:4114` 另列 10 个测试文件及 build tag。该表足以作为重新规划写集的输入（但尚未写回任务卡，见 F017）。

### F017 — HIGH — r62 建议写集未落回任务卡，按现任务卡实施必越界（未闭合）

- **原 finding**：r61 的 4 文件写集不足以实现新增契约，实施者只能越界。
- **r62 落点**：`:4038`、`:4116`；`tasks.md:3191`–`:3195`。
- **判定**：**未闭合**。

**证据**：

- `tasks.md:3191`–`:3195` 的精确写集仍是 4 个文件：`recovery_restore.go`、`recovery_runtime.go`、`main.go`、`recovery_restore_production_integration_test.go`。
- `design.md:4116` 自认至少还缺 approval、target、四个 host/exec 平台文件、`internal/recovery/restore_source.go`、`recovery_config.go` 修改与对应测试，并明确“本轮未改 `tasks.md` 任何一行”。
- 因此作者没有假装已改任务卡；但按现任务卡派发实施时，写集边界与 r62.9.1 直接冲突，实施者仍只能越界或无法实现。

**影响**：这是交付边界缺陷，不是实现细节。Task-Planner 按 `:4098`–`:4114` 更新写集、验收命令与负向测试清单之前，S4.24a 不可实施。

**修复方向**：由 Task-Planner 更新 `tasks.md` 写集、测试文件与验收命令，并保留 F009 的 Linux NOT RUN 关闭前置；Reviewer 不代替作者或 Task-Planner 修改任务卡。

### F018 — BLOCKER — 外部回执验签、审计与见证（部分闭合）

- **原 finding**：r61 的 `tool.receipt` 只有展示字段，没有签名、信任锚、验证方独立性、单调序列、独立读取与审计记录，ADR-0031 不变量 6 未编码。
- **r62 落点**：`:3994`–`:4004`（r62.5）；上游不变量见 `framework.md:1147`，约束 8 第 6 项见 `:1166`。
- **判定**：**部分闭合**。方向与不变量 6 对齐，但仍无法被独立实现唯一复算/执行。

**已补齐部分**：

- `:3994` 固定 `tool.receipt` 的 13 字段 exact keyset，无额外键、无 `null`。
- `:3996` 固定 `ed25519`、64 字节签名 base64url 无填充、`producer_key_id`→信任根 `receipt_anchors`、`trust_anchor_ref`/`trust_anchor_sha256`、验证方 principal、绝对规范路径与挂载独立性、`sequence > sequence_floor`、UTC 微秒时间窗。
- `:4000` 给出 revalidate → source verify → execute → 独立读取 → 验签 → sequence → 审计 fsync → 回填的八步顺序，并明确任一步失败不得成功。
- `:4002` 给出 `ErrAdmissionBinding`／`restore_unknown`／`ErrAdmissionIncomplete`／`ErrAdmissionClosed`／`ErrAdmissionUnsupported` 的分类。

**未闭合部分**：

1. **签名载荷外层 schema 未定义**。`:3998` 只说签名载荷是“回执正文 JCS 字节”与 `{producer_identity, producer_key_id, trust_anchor_ref, sequence, issued_at}` 按 JCS 组成的对象，删除 `signature` 与 `audit_record_ref` 后的 canonical bytes；没有给出外层键名、嵌套层级、五个摘要字段与原回执正文的关系，独立验证方无法唯一重建待签字节。
2. **`audit_record_ref` 的空串规则与删除规则冲突**。`:3996` 要求该字段“在签名载荷中必须为空串”，`:3998` 又要求删除 `audit_record_ref` 后取 canonical bytes；空串字段与删除字段产生不同字节，无法判断签名时到底保留空串还是删除。
3. **回填与 `receipt_digest` 未定义**。`:4000` 第八步只写“回填 `audit_record_ref` 后返回 receipt”，未规定是否重算 `receipt_digest`、更新内存对象还是持久记录、回填值的事实来源及失败处理。
4. **sequence 基准不一致**。`:3996` 要求 `sequence > receipt_anchors.sequence_floor`，`:4000` 第六步又要求本次 sequence 严格大于 `audit_sink` 最后一条记录的 sequence；两者谁是下限、`audit_sink` 为空时取何值、floor 落后于 sink 时如何比较，均未定义。
5. **审计 sink 的读写模式与所有权未定义**。`:3996` 要求以 `O_RDONLY|O_CLOEXEC|O_NOFOLLOW` 打开 `audit_sink`，`:4000` 第七步却要向同一 sink 追加记录并 `fsync`；只读打开无法追加，读写模式、O_APPEND、owner、mode、ACL 与失败分类未给出。
6. **验证方独立于 root 不可机核**。`:3996` 只禁止 `verifier_principal_id` 等于 executor 或 issuer；`:4004` 声称验证方须与 root 分离，但 `receipt_anchors` 没有 root principal 字段，无法由数据结构检查该合取（另见 NEW-R02-06）。

**影响**：不变量 6 的“先验签、独立读取、信任锚固定、验证方独立于执行与 root、审计 fsync”方向已覆盖，但签名载荷与审计序列仍有多个未定义分支；按现文本无法证明独立验证方会接受同一字节，也无法安全写出审计记录，故原 BLOCKER 未闭合。

### F019 — HIGH — `install`/`enter`/`resume`/normal/verify 的 nil 语义（已闭合）

- **原 finding**：r61 笼统写“现有 nil → closed 分支不变”，但 `install` 不需要 restore，会被误判为 closed；各 action 的 nil 语义没有逐一固定。
- **r62 落点**：`:4040`（r62.6.9）。
- **判定**：**已闭合**。

**证据**：

- `:4040` 按 action 固定：`install` 的 `inputs.restore == nil` 合法，必须从 enter/install 缺失端口列表排除；`enter` 需要 restore，nil 由 `recoveryEnterInstallMissingPorts` 返回 `ErrAdmissionClosed`，在 Begin/写 DB/CAS/文件与建子进程前；`resume` 需要 restore，nil 使用 `recoveryClosedRestoreDriver`，`Prepare` 返回 `ErrAdmissionClosed`；`normal`/`verify` 不需要 restore，nil 使用 closed driver 且不会调用 restore。
- 现有代码 `cmd/han-server/recovery_runtime.go:527`–`:532` 对 `enter`/`install` 先走 missing ports，`:596`–`:599` 才是 nil→closed fallback；r62 的 action 分支与该结构一致。
- `:4040` 明确取代 r61.3 的笼统表述，并声明 r60 的 11 端口表以本段为准；未发现它与既有代码的 action 语义冲突。

### F020 — HIGH — `ErrAdmission*` 退出码映射（已闭合）

- **原 finding**：r61 把预检与 run 期的 `ErrAdmission*` 都写成退出码 2，与现有 `run` 的非 nil 错误经 `lifecycleExitCode` 返回 1 不一致。
- **r62 落点**：`:4042`（r62.6.10）。
- **判定**：**已闭合**。

**证据**：

- `:4042` 将预检错误包装为 `*recoveryRestorePreflightError`，`main` 在 `run` 返回后用 `errors.As` 识别并 `os.Exit(2)`；其余非 nil 错误仍经 `lifecycleExitCode` 为 1；`selectRecoveryCommand` 解析失败仍为 2。
- `:4042` 明确 `Prepare`/`Replace`/`VerifyRestored` 的错误属于 run 期，退出码 1；并作废 r61.7 与 r61.5 的“一律退出码 2”表述。
- 代码核对 `cmd/han-server/main.go:171`–`:199`：命令解析失败路径为 2，`run` 返回的非 nil 错误走 `lifecycleExitCode`（1）；该映射与设计一致。

## 5. 新增 finding（r02 新识别，均非原 r01 finding）

### NEW-R02-06 — MEDIUM — 验证方与 root 的独立性缺少可机核字段

- **落点**：`:3996`、`:4004`；与 ADR-0031 不变量 6（`framework.md:1147`）及约束 8 第 6 项（`:1166`）相关。
- **判定**：**新增缺陷**。
- **证据**：`:3996` 只要求 `verifier_principal_id` 不得等于 executor 或 issuer，`:4004` 又声称验证方必须与 root 分离；但 `receipt_anchors` 只有 producer/verifier principal、key 与 anchor 字段，没有 root principal 或 root identity 字段，也没有任何“root 不得参与验证”的可核结构。测试无法据数据结构断言该合取。
- **影响**：即使回执签名与 sequence 都正确，验证方仍可能由 root 控制；ADR-0031 的“验证方独立于 root”无法被实现或验收机核。
- **修复方向**：在信任根中登记 root principal/role 与验证方关系，或明确验证方独立性由哪个已签名外部声明承载，并给出可执行拒绝条件；不要只写自然语言。

### NEW-R02-07 — MEDIUM — JCS 与 Go `encoding/json` 的转义差异未约束

- **落点**：`:3935`、`:3984`；影响 `approval_digest` 的 RFC 8785 canonical bytes。
- **判定**：**新增缺陷**。
- **证据**：`:3935` 要求按 JCS 规范化且“不做 HTML 转义”，`:3984` 的自定义 `appendJSONString` 只覆盖 DSN wire；但批准记录/回执正文的 JCS 编码没有明确禁止使用 `encoding/json` 默认 HTML 转义。Go `encoding/json` 默认会把 `<`、`>`、`&`、U+2028、U+2029 转成转义序列，而 RFC 8785 要求保留非 ASCII 原文并对这些字符不作 HTML 转义；同一逻辑对象因此可能产生不同 canonical bytes 与不同签名摘要。
- **影响**：批准摘要或回执签名在含特殊字符时无法被独立 JCS 实现稳定复算；golden vector 可能只覆盖了不触发的输入。
- **修复方向**：明确规定必须使用 JCS 合规编码器（或显式设置 `SetEscapeHTML(false)` 并验证 U+2028/U+2029），禁止默认 `json.Marshal` 直接承担 canonicalization；增加特殊字符 golden vector 与独立实现交叉验证。

### NEW-R02-08 — MEDIUM — `reviews.tool_sha256` 与 `tool.executable_sha256` 的字段映射未明说

- **落点**：`:3960`；相关批准记录字段见 r62.1 与 r61.2（`:4190` 附近）。
- **判定**：**新增缺陷**。
- **证据**：`:3960` 要求信任根 `reviews` 记录的 `tool_sha256` 与被批记录相等，但批准记录的字段名是 `tool.executable_sha256`（r62.1 的 tool exact keyset），两个名字不同；文本没有说明 `reviews.tool_sha256` 是别名、摘要值还是另一个独立字段。
- **影响**：实施者可能把 `tool_sha256` 与 `executable_sha256` 当作两个不同值，或在 JCS 生成时遗漏映射，从而错误接受/拒绝批准记录。
- **修复方向**：明确 `reviews.tool_sha256` 必须逐字节等于 `tool.executable_sha256`，并在 schema/示例中给出同名字段或显式映射；若它表示工具整体摘要而非可执行文件摘要，必须定义算法与输入。

### 待决定项（非缺陷）

- **ADR-0031 r07 状态冲突**：`design.md:4122` 指出 `framework.md:1129` 与 `tasks.md:3197` 的状态描述不一致；该问题属于架构决策与任务卡状态维护，不是 r62 设计正文可单方面修复的代码缺陷。需要架构决策负责人与 Task-Planner 对齐，不能由 Reviewer 或作者代为修改 ADR/tasks。它不改变本次 finding 计数，但在状态对齐前不能把 r07 的接受事实当作新的生产或实施授权。

## 6. 未验证边界

本报告是静态评审，不是实现验证；以下范围未能核验，不能据此声称通过：

- 未运行 Go `build`、`test`、`vet`、`gofmt` 或任何 lint；未在 Linux 运行任何命令。
- 未运行独立 JCS 库、Ed25519 验签库或真实 golden-vector 交叉验证；仅按文本声明的键序对两条向量做了静态字节与 SHA256 复算，作者本地复算不构成独立证据。
- 无真实信任根、Ed25519 密钥、批准记录、回执、见证端点、审计 sink 或文件权限/ACL 输入。
- 未验证真实 mountinfo、POSIX ACL、loader、fd-exec、netns、seccomp、`/proc`、`AT_SECURE` 或 Windows 子进程行为。
- 未全文读取安全侧 `review-sec-r01`，只按派发范围内与架构 finding 直接相关的上游条款核对；安全侧其他 finding 的闭合状态不在本报告结论内。
- ADR 只读相关行：`framework.md:1128`、`:1147`、`:1160`–`:1169`、`:1201`；未把 ADR 全文或任务卡全文当作已验证对象。
- 未核 `tasks.md` 全文哈希，只读 `:3186`–`:3218`；未核验未提交的 295 条改动内容，也未运行任何会改变工作树的命令。
- 未实现或实测 r62 的任何契约，因此本报告不构成 S4.24a 关闭、生产 restore 可用、`enter`/`resume` 解锁或约束 8 满足。

## 7. 修复与复审条件

在以下条件满足前，verdict 不能升级为 `PASS`；其中任一 BLOCKER 未真正闭合，均应维持 `CHANGES REQUIRED`。

1. **F001**：按 `:3935` 的 JCS 键序重排两条 golden vector（`reviewer` 在 `revocation` 前），重算并更新 `:3943`/`:3949` 的字节数与摘要；明确 canonicalization 必须使用 JCS 合规编码，禁止默认 HTML 转义；由非作者使用独立 JCS 实现复算同一向量。
2. **F005**：verifier 读取 artifact 头部后必须 `Seek(0)`（或 `ReadAt`/`dup`）并给出错误路径；统一 `ArtifactBytes` 为 `int64`；恢复 `Version` 与 `TargetDigest`；明确 `Approval`/`Tool` 快照的包边界与可见性，使 `internal/recovery` 与 `cmd/han-server` 均可编译。
3. **F006**：`Bind` 改为接收 `recovery.ExclusiveLease`（或最小 `Check` 子接口），由 assembly 在唯一 `Acquire` 后传入；明确唯一 `Acquire` 的所有者，消除 r60/ADR-0032 关于 `inputs.exclusive` 的冲突；不得用类型断言、包装或第二次 `Acquire` 绕过。
4. **F018**：定义签名载荷的外层 schema、嵌套键名与五个摘要字段的位置；统一 `audit_record_ref` 空串/删除规则；定义回填后 `receipt_digest` 是否重算及持久化语义；统一 `sequence_floor` 与 `audit_sink` 最后 sequence 的基准及空 sink 行为；定义 audit sink 的读写模式、O_APPEND、owner/mode/ACL 与失败分类；增加 root/verifier 独立性的可机核字段（同时关闭 NEW-R02-06）。
5. **F004**：在 r60 `:3603` 的“未登记依赖不得注入 production root”处加入窄例外，或在 r62 `:4008` 增加“以本段为准”的明确优先句，并同步 ADR-0032 的继承说明；不能只靠 r62 自述“不推翻 r60”而保留两段相反文本。
6. **F011**：在 `:4000` 八步中补入“execute 返回后、解码回执前 revalidateApproval”，并同步 `:3976` 的读取点清单与 `:3978` 的失败分类。
7. **F017**：Task-Planner 按 `:4098`–`:4114` 更新 `tasks.md` 写集、测试文件、验收命令与负向测试清单；在该更新完成前不得派发实施。Task-Planner 更新不改变 F009 的 Linux NOT RUN 关闭条件。
8. **F009 关闭前置**：即使 r62 设计经独立评审接受，也必须在 Linux 目标或 CI 取得真实负向 (d)(e)(f) 与 9 条验收命令证据；Windows skip、无匹配测试 exit 0 或空断言都不是 PASS。S4.24a 不得因设计 PASS 而关闭。
9. **复审材料**：作者整改后提交新的固定修订（含精确字节数、行数、SHA256、变更区段、修订说明与回归证据）；复审将重新核对原 finding 的落点与真实消费者，不以作者摘要替代。若修订改变架构/契约边界，还应按项目决策流程取得独立决策依据。

## 8. 只读声明

本报告仅新建/写入 `.spec/wip/recovery-runtime/evidence-b13j-s424a/review-arch-r02/report.md`。评审期间未修改 `.add/**`、`design.md`、任何代码、`tasks.md` 或 ADR；未执行 Git add/commit/branch/stash/checkout/restore/reset/clean；未删除、移动或归档任何文件。工作区既有未提交改动保持原样。本报告是 Reviewer 的独立静态结论，不代替作者整改、Task-Planner 写集更新、安全侧结论、人的结果确认、生产批准或 S4.24a 关闭。
