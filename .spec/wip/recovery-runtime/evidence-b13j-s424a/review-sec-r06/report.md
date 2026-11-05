# SEC r06 独立复审报告（Han identity-auth design r66）——评审中止报告

**角色**：`add-reviewer`（安全侧独立评审，非作者、非实现者）
**固定工作区**：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abb`
**评审对象**：`.add/project/knowledge/design-workspace/identity-auth/design.md`
**派发对象**：`revision: identity-auth-design-ir01-r66`，Lead 实测基线 `0BA1C6C1B2F4C21BA4BC3FA5F7E3A949040BC01BE00329D485D6CC4DC2FA1847`
**写集**：本报告与本目录 `hashes.txt`（仅新建；未改 `design.md`，未做任何 Git 写操作）
**日期**：2026-11-02

## 0 结论（首段）

本次评审在第二轮进行中触发派发方设定的停止条件：`design.md` 在评审期间由派发基线 `0BA1C6C1…`（1,078,886 B）变为 `9B193BDE…`（1,079,559 B），**+673 B／0 行／0 CR**，且文件仍声明 `identity-auth-design-ir01-r66`，没有 r67 标记。该变更为评审会话之外的外部写盘（本端只读，未改 `design.md`）。按停止条件，本端停止对新字节的语义评审，只保留派发基线 `0BA1…` 上的停前观察，并把当前字节标记为**未复核**。

**一句明确判定：不构成 `design.md` r66（停止时 `9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298` 字节）的安全侧接受（否）。**

该"否"的含义是：**本次评审未完成，不能以本次结果作为 r66 的安全侧接受**；它不是对 r66 安全语义的"不安全"否决，也不是对派发基线 `0BA1…` 的接受。对 `0BA1…` 的停前观察只说明若干条款"文本上看闭合／需重审"，不构成任何签收。下文的 provisional findings 只针对证据链与文本一致性，均须在对象冻结后重新判定。

## 1 计数表

| 项 | 计数 | 说明 |
|---|---:|---|
| 实际完成评审轮次 | 1.5 | 第一轮完成于派发基线 `0BA1…`；第二轮（跨文件与消费者有界检查）进行中触发停止条件 |
| 停前观察项（仅 `0BA1…`） | 9 | 5 项 R05 finding、1 项 `SEC-F007` NOT RUN、2 项 `nonce`／`bundle_digest`、1 项撤销与信任锚 |
| 停止时字节 `9B19…` 的最终安全 findings | 0 | **0 ≠ 无 finding**；是"未复核、未完成"，不得读作通过 |
| provisional findings（证据链／文本一致性） | 3 | MEDIUM 1（`NEW-S06-01`）、LOW 2（`NEW-S06-02`、`NEW-S06-03`），须冻结后复核 |
| 对 r66 的范围内 BLOCKER／HIGH／MEDIUM／LOW 最终判定 | 不适用 | 对象在评审期间改变，评审未完成，不能给出最终严重度结论 |
| NOT RUN | 1 | `SEC-F007`（不得改判 PASS） |
| 范围外 | 1 | `N005`（ADR 状态／任务卡冲突，不计入本对象，不代改） |
| 停止条件 | 1 | 已触发并已停止 |

## 2 实测基线

| 节点 | 字节 | 行（LF） | CR | BOM | SHA256 | revision |
|---|---:|---:|---:|---|---|---|
| 派发基线（Lead 实测，评审开始时本端复核同值） | 1,078,886 | 5,476 | 169 | 无 | `0BA1C6C1B2F4C21BA4BC3FA5F7E3A949040BC01BE00329D485D6CC4DC2FA1847` | `identity-auth-design-ir01-r66`（`:5`／`:38`） |
| **停止时实测** | **1,079,559** | **5,476** | **169** | **无** | **`9B193BDE821B69A5CB1872B404089E2DE9FCFFB5A80AB5AE09D98AED555A2298`** | 仍为 `identity-auth-design-ir01-r66` |
| 停止时 − 派发基线 | **+673** | **0** | **0** | — | 变化 | 未升修订号 |
| r65 派发基线（对照） | 1,045,111 | 5,464 | 169 | 无 | `87D70DA85EC73ED02D4B2DDE8F2A303BD6B9DAB6CE2544D5114DBEF58D9DC2E9` | `identity-auth-design-ir01-r65` |

停止时文件 mtime（UTC）：`2026-11-02T07:18:54.3590867Z`。`git status --short -- design.md` 为 ` M`；HEAD 仍为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本端未执行任何 Git 写命令。

## 3 停止条件与现场证据

| 证据 | 派发基线 `0BA1…` | 停止时 `9B19…` | 判定 |
|---|---|---|---|
| 整文件 | 1,078,886 B／`0BA1C6C1…` | 1,079,559 B／`9B193BDE…` | **B/SHA 均变，停止条件成立** |
| r66 块 `:4072`–`:4083`（含行尾 LF） | 30,936 B／`6E25E9CD81248C637A6C97E5880FE813EE6D4D1A12B8C94B03BEDF9E4C26CB5B` | 31,141 B／`587471F58DB925A17D4E3F06AFE981244E9D2120B969DB615AC5DE2BED749048` | +205 B |
| `:4435` 单行（去掉行尾 CR/LF） | 364 B／`429E633A6B13D5B26E851FF4F3459E7A1A33246A706F2CF30431235207FFE1A5` | 364 B／同值 | r61.5 原字节未被本轮触碰 |
| 块外残余 | — | **+468 B**（与 `:3603` 的 +468 B 一致；停止时 `:3603` 整行 1,518 B） | 可推定块外只改 `:3603` |
| `:4072` 整行 | 2,950 B（由 3,155−205 反推） | **3,155 B** | 作者报告 §2 写"312 → 517"，与整行不符（见 §4.2） |

停止时文件仍在 `:5`／`:38` 声明 r66，说明这是一次**同一修订号下的就地改写**，不是 r67。派发基线因此不再可复现；本报告不对 `9B19…` 的语义作通过或否决。

## 4 输入与作者证据实测

### 4.1 输入文件实测

| 文件 | 字节 | 行 | CR | SHA256 |
|---|---:|---:|---:|---|
| `review-sec-r05/report.md` | 39,093 | 293 | 0 | `4A2292674E49EB4A7961CD1EDD0FA4BCF1560D2E911B74AE1E208AC3790DE0C1` |
| `review-arch-r05/report.md` | 32,489 | 281 | 0 | `DD2423B7BD574B5C61603F938B42856061EA3F7B4AA59BA4D033B04BE5AD93D1` |
| `design-revision-r06/report.md`（评审开始时） | 30,201 | 327 | 0 | `4AC8FEDDCB92BFAC145476DEE11C5787A944E0370821B1C14C656B3ED1416AFE` |
| `design-revision-r06/report.md`（评审中改写后，现行） | 24,041 | 196 | 0 | `CA5F28AC27DC4951AA6D4335F4DC2DE5CF0C714E3CCBF79C0E3EA93FF8A7C671` |
| `design-revision-r06/hashes.txt` | 3,181 | 51 | 0 | `DE2D71BD48743583A4CEDF3712A26B16B581B38267A57517B6AFA6C5BF8954FD` |

作者报告在评审期间也被改写：开始时实测 30,201 B／327 行（含重复章节：`:198 "## 8 越界需求"` 后又在 `:207` 出现 `"## 5 派发清单外的既存 r66 内容"`），改写后 24,041 B／196 行（重复章节已删）。这本身不是评审对象的语义问题，但说明**作者证据在评审期间不处于冻结状态**。

### 4.2 作者证据链缺陷（机核）

1. **`hashes.txt:37` 自述与本报告实际不符**：它记录 `report.md` 为 21,142 B／205 行／`A03CE62B3A8CD115328851E818C35BBC2B5073040C1B6EE32852150E155B050A`；实测同一文件已先后为 30,201 B／`4AC8FEDD…` 与 24,041 B／`CA5F28AC…`，三个值互不相同。该自述未在最后一次写盘后重算。
2. **`hashes.txt` 已被自身规则判为失效**：其 `:52` 写明"若 design.md 在本文件生成后再次被修改，本文件全部数值立即失效，必须整体重算并覆盖"。`design.md` 已从 `0BA1…` 变为 `9B19…`，但 `hashes.txt:16` 仍把 `0BA1…` 记为"r06 终值（本段编辑后）"，`hashes.txt:29` 仍记块为 30,936 B／`6E25E9CD…`。按其自身规则，该文件当前整体失效。
3. **作者报告 §2 的"行级绝对字节数"不可复算**：`report.md` 现行 §2 写 `:3603` "382 → 850，+468"、`:4072` "312 → 517，+205"。实测停止时 `:3603` 整行为 **1,518 B**、`:4072` 整行为 **3,155 B**，均与所写绝对数不符；只有增量 +468／+205 与整文件差额吻合。该表以"行"为列头，读者会把它读成整行字节，属可误导的证据表述。
4. **同一 r06 目录存在两套不一致的版本链**：`hashes.txt:15` 记录"r06 首测"= 1,077,602 B／`EFBFDDF0…`，`:16` 记录终值 = `0BA1…`；现行 `report.md` §0 则把 `0BA1…` 称为"本次会话开工时（r66 草稿态）"，把 `9B19…` 称为终值。二者对"哪个是终值、有几个节点"给出不同事实，且都没有把 1,077,602 → 0BA1 的 +1,284 B 逐项落到行。
## 5 必核项 A（R05 安全侧 finding 逐条闭合）——仅 `0BA1…` 停前观察

下表只对派发基线 `0BA1…` 的文本作停前观察，**不是对停止时 `9B19…` 的最终判定**。行锚为停止时实测行号；`0BA1…` 与 `9B19…` 行数相同（5,476），故行号不变。

| finding | 级别 | 行锚 | R05 原句（据 `review-sec-r05/report.md`） | 判据 | `0BA1…` 停前观察 | `9B19…` 状态 |
|---|---|---|---|---|---|---|
| `SEC-F003`＋`N001`（同一根因） | BLOCKER | `:4062`（V8 子句）、`:3988`（producer 载荷）、`:4059`（8 字段验证方载荷）、`:4074`（r66.2） | R05 `:72`："`:4062` 的 `V8` 把 **producer 签名**指向 r65.2 的 **8 字段验证方载荷**；该载荷含 `verified_at`／`verifier_*`，V8 时尚不存在，producer 载荷实际在 `:3988`"；R05 `:106` 警告实现者若把 producer 载荷"统一"为 8 字段会重开绑定缺口 | V8 只引 `:3988` 的 producer envelope；8 字段只服务 V10 的 `:4059`；producer／verifier 载荷各只有一处定义；`rg 'V8.*r65[.]2'` = 0 | 文本上看闭合：`:4062` 的 V8 已改为引用 `:3988`，8 字段只在 V10；`V8.*r65[.]2` 复跑 = 0 | **未复核** |
| `N011` | HIGH | `:4077`（r66.5） | R05 `:74`："六项约束已加，但 DNS rebinding TOCTOU、host 精确匹配无独立登记字段、IPv4-mapped IPv6 未覆盖、pin 与'禁止 InsecureSkipVerify'组合无实现路径、pin 轮换未定义、独立性清单漏 verifier" | 设计须逐项给出：DNS 解析固定 + `DialContext` 只连已核 IP + 连接后 `RemoteAddr.Unmap()` 复核；独立 `witness_host`；`::ffff:` 与非规范 IPv4 拒绝；pin 唯一路径（`InsecureSkipVerify` + `VerifyConnection` 三项校验）；单 active pin 轮换；独立性含 verifier | 文本上看闭合：六项均有对应句；运行面 DNS／TLS／pin／IPv4-mapped 全为 NOT RUN | **未复核** |
| `NEW-S05-01` | HIGH | `:4078`（r66.6） | R05 `:83`："仅'绝对规范路径＋列入批准记录'，无属主/可写性/loader 父目录覆盖/禁 LookPath；批准记录无目录载体" | tool 记录新增 `environment_dirs`；目录须 root 属主、非 group/other/executor 可写、逐级无符号链接、挂载 `ro,nodev,nosuid`；loader 父目录不得落入；禁 `exec.LookPath` 与 PATH 搜索；显式拒绝 `PYTHONPATH`／`NODE_PATH`／`RUBYLIB`／`PERL5LIB`；目录数 ≤32、总长 ≤4096 B、`PATH` 逐字节等于 `:` 连接结果 | 文本上看闭合：条款齐全；`exec.LookPath` 在设计中仅作为禁止对象出现（1 处命中） | **未复核** |
| `NEW-S05-02` | MEDIUM | `:4072`（r66.0）、`:4078`（r66.6）、`:4435`（r61.5） | R05 `:84`："r61 原字节 `:4423` 仍写'禁止 `PATH`'，r65.6 允许 `PATH`；取代链只在 `:4039`／`:4063` 元声明，无冲突位置的可判定优先级" | 唯一优先级可机核：`r66 > r65 > r64 > r63 > r62 > r61`，同修订内小节号大者胜；明示 `:4435` 由 r66.6 取代；r61 字节保持不变 | 文本上看闭合：`:4072` 给出唯一优先级并明示 `:4435` 被 r66.6 取代；`:4435` 实测仍为 364 B／`429E633A…`，未被触碰 | **未复核** |
| `NEW-S05-04` | LOW | `:4077`（r66.5）、`:4079`（r66.7） | R05 `:86`："r65.3 未保留 r64.3 的 2048 B 上限；host 精确匹配无独立字段；`witness_key_id` 无 `org_id` 字段；pin 轮换未定义；root/supplier 任命值域未定义" | `witness_endpoint` 2048 B 上限；独立 `witness_host`；`witness_key_id → keys.principal_id → principals.org_id`；pin 轮换须人批准 + 审计 + 单 active + 窗口外 closed；root／supplier 的 `appointment_ref` 只允许空串且不得绑定 appointment | 文本上看闭合：五小项均有对应句；`2048 B` 命中 2 行（含 r66.5 与指针） | **未复核** |
| `SEC-F007` | NOT RUN | `:4021`、`:4082`、`:4191` | R05 `:81`："`:4070(6)` 仍写 Linux host 层全部未跑；不得改判 PASS" | 必须保持 NOT RUN；任何 NOT RUN 不得写成 PASS | 三处仍为 NOT RUN；`:4070(6)` 明列 host／ACL／mount／loader／fd-exec／maps／network／witness 未运行 | **未复核，且不得改判 PASS** |

**复算命令（在停止时字节上实测；`0BA1…` 的块哈希取自 `design-revision-r06/hashes.txt:29`）**：

```text
rg -n 'V8.*r65[.]2' <design.md>                          # 0
rg -c 'DialContext|RemoteAddr|Unmap|witness_host|InsecureSkipVerify|VerifyConnection' <design.md>
rg -c 'environment_dirs|exec[.]LookPath|PYTHONPATH|2048 B|witness_key_id|appointment_ref' <design.md>
rg -n 'SEC-F007' <design.md>                             # :4021, :4082, :4191
# 块哈希（含行尾 LF）：:4072–:4083
# 0BA1 30,936 B / 6E25E9CD81248C637A6C97E5880FE813EE6D4D1A12B8C94B03BEDF9E4C26CB5B
# 9B19 31,141 B / 587471F58DB925A17D4E3F06AFE981244E9D2120B969DB615AC5DE2BED749048
```

## 6 必核项 B（`nonce` 与 `bundle_digest`）——仅 `0BA1…` 停前观察

| 项 | 行锚 | 原句要点 | 判据 | 停前观察 | `9B19…` 状态 |
|---|---|---|---|---|---|
| `nonce` | `:3973`、`:4075`（r66.3） | "`nonce` 是**进程内错误关联标识**……不改变任何 wire 形状……不是鉴权／授权／幂等键"；"`crypto/rand` 读取 **16 字节**并编码为**小写十六进制 32 字符**"；"不得落盘，不得进入 Journal、审计记录、数据库、成功回执、`restoreReceiptWire`、producer envelope、任何 DSN 派生值、argv、ENV、信任根或批准记录"；日志键 `restore_nonce`，错误响应只经 `details.nonce` | 16 B（128 bit）随机、单一编码、进程内、不落盘、不进 wire／回执、不可用于权限判定；生成失败固定分类且不回退 | 文本上看合规；全文 `nonce` 仅 2 行（`:3973`／`:4075`）；`details.nonce` 只作关联，未发现可预测性或越权用途；实现未做 | **未复核** |
| `bundle_digest` | `:4075`、`:4082` | "**`bundle_digest` 显式 OPEN（首期不实现）**……不定义字段、不定义算法，也**不宣称闭合**"；"不得出现在成功回执、`restoreReceiptWire`、producer envelope 或任何'已闭合'语境" | 仅 OPEN；首期不实现；未改 `restoreReceiptWire` 12 字段形状 | 文本上看合规；全文仅 2 行且均带 OPEN／首期不实现；`cmd/han-server/recovery_restore.go:52–64` 的 `restoreReceiptWire` 仍为 12 字段（`Version`…`ReceiptDigest`） | **未复核** |

**实现侧只读抽查**：`cmd/han-server/recovery_restore.go:52–64` 的 `restoreReceiptWire` 字段数为 12；`newControlledRestoreDriver`（`:203`）仍为 4 参数；`d.prepared(ctx)` 调用点仍在 `:226`／`:239`／`:327`／`:352`／`:375`。这些是"设计未实现"的现状证据，不是本对象的安全缺陷判定。

## 7 必核项 C（字节账闭合）

### 7.1 三段实测

| 阶段 | 起点 | 终点 | ΔB | Δ行 | 逐项证据 |
|---|---|---:|---:|---:|---|
| r65 → r06 首测 | 1,045,111 | 1,077,602 | +32,491 | +12 | r66 块 30,936 B；**残余 1,555 B 未被逐项登记** |
| r06 首测 → `0BA1…` | 1,077,602 | 1,078,886 | +1,284 | 0 | 作者旧 `report.md` §0 称"10 处行内替换、无其它字节改动"（现行报告已删除该叙述） |
| `0BA1…` → `9B19…` | 1,078,886 | 1,079,559 | +673 | 0 | `:3603` +468、`:4072` +205（作者现行 §2） |
| **合计 r65 → `9B19…`** | **1,045,111** | **1,079,559** | **+34,448** | **+12** | 32,491 + 1,284 + 673 = 34,448，**算术上配平** |

### 7.2 判定

- **能否配平：算术上能**。三段相加等于整文件实测差额，且 `0BA1 → 9B19` 的 +673 B 可拆为 `:3603` +468、`:4072` +205，与块增量 +205、块外残余 +468 一致。
- **能否逐项机核：不能**。① r65 → r06 首测的 +1,555 B 残余没有任何逐项落点；② +1,284 B 阶段只在 `hashes.txt:15/16` 作为节点存在，现行作者报告没有逐项列出；③ 1,077,602 节点在现行作者报告中缺席；④ `hashes.txt` 未随 `9B19…` 重算，且其 `:37` 自述哈希与实际不符（见 §4.2）。
- **是否构成 `NEW-S05-03` 同类缺陷复发：是**。同类点是"证据链／基线标注不能用作者提供的数值直接机核"，不是同一处 r61 标题定位问题。已新立 **`NEW-S06-01`（MEDIUM）**，见 §9.1。
- **是否安全语义缺陷：不是**。它不证明 r66 的安全条款有错，但阻断"以作者证据支持本次接受"的路径；在对象未冻结时也不能把证据链判为已闭合。

## 8 必核项 D（撤销与信任锚）——仅 `0BA1…` 停前观察

| 检查点 | 设计锚点 | 停前观察 | 残余 |
|---|---|---|---|
| 启动预检 | `:4373` 其一、`:4375`、`:4133` | `recoveryRestoreStartupPreflight` 是仅有两处允许读批准文件／DSN／信任根的位置之一；`:4375` 给出 `issued_at <= valid_from <= now < valid_until` 与 `revocation.state=active` 判定式 | 文本层有覆盖；运行面 NOT RUN |
| `prepared(ctx)` | `:4135` 其三／其四、`:4373` 其三 | `Replace`／`VerifyRestored` 在 `d.prepared(ctx)` 与真正启动子进程之前各调用一次 `revalidateApproval` | 文本层有覆盖 |
| 读取回执 | `:4137` | "`execute` 在子进程返回后、解码回执前再调用一次 `revalidateApproval`" | 文本层有覆盖 |
| 验签（V8／V10） | `:4062` | V8／V10 在回执解码之后；解码前有一次重验，但两次签名校验之前没有逐次重验 | **LOW 级残余线索**：撤销／过期若发生在解码后、签名前，仍可能完成验签 |
| 推进见证 head（V10） | `:4062` | V10 "追加审计记录、`fsync`、验证方签名并让外部见证原子推进 `sequence`"；`:4137` 的最近一次重验在"解码回执前" | **LOW 级残余线索**：推进前没有显式 `revalidateApproval`（`NEW-S06-02`） |
| `generation` | `:4369` | 批准记录最小字段只有 `valid_from`、`valid_until`、`revocation.state`，**没有 `generation` 字段或可核定义** | 跨文件待核对（`NEW-S06-03`）；ADR r10 修订说明声称将补 `generation` 三元定义，但截至读取时正文未检出该定义 |
| 信任锚自选 | `:4077`、`:4070(2)` | 见证独立性含 verifier；信任锚固定在批准记录内；`root` 可整体替换信任根与见证的残余被明确登记，不声称抗 root | 未发现"由被验证回执自选信任锚"的文本路径；root 残余是已登记残余，不是本轮新增缺陷 |
| 模拟见证 | `:4070(6)`、`:4282`、`:4447`、`:4189`、`:4261` | Linux witness 实测 NOT RUN；真实信任根／密钥／批准记录／回执见证"均不存在"；`restoreProcessTool` 明确为软件合同测试替身 | **未发现把模拟见证写成生产能力的句子**；但"模拟见证"在设计中没有一个独立命名的测试替身条目，只有 NOT RUN 与 `restoreProcessTool` 替身说明 |

**D 侧结论**：在 `0BA1…` 文本上，启动预检、`prepared(ctx)`、读取回执前均有撤销／有效期重验；验签与见证 head 推进前存在两个窄窗口，列为 provisional LOW；`generation` 定义缺失为跨文件待核对项。以上均**未对 `9B19…` 复核**。
## 9 provisional findings（证据链／文本一致性；须冻结后复核）

以下 3 项**不是**对 `9B19…` 安全语义的最终 findings，也不计入"停止时字节最终安全 findings = 0"；它们是本次评审在停止前已经能机核的交付完整性／文本一致性线索。

### 9.1 `NEW-S06-01`（MEDIUM）作者证据链不自洽、变更集不可逐项机核

- **对象**：`design-revision-r06/report.md` 与同目录 `hashes.txt`（作者证据），不是 `design.md` 的安全条款。
- **行锚**：`hashes.txt:15`、`:16`、`:29`、`:37`、`:52`；`design-revision-r06/report.md` §0、§2。
- **原句**：`hashes.txt:37`："21,142 B ／ 205 行 ／ CR 0 ／ LF 205 ／ SHA256 A03CE62B…"；`hashes.txt:52`："若 design.md 在本文件生成后再次被修改，本文件全部数值立即失效，必须整体重算并覆盖"；作者报告 §2："`:3603` … 382 → 850，**+468**"、"`:4072` … 312 → 517，**+205**"。
- **判据**：设计 `:4080`（r66.8）要求报告与 `hashes.txt` 在**最后一次写盘之后**生成并记录终值；报告须逐项给出 finding → 落点 → 摘要 → 判定，并分别标注基线漂移、既有行内改动与本轮行内替换，不得合并成一个净值。
- **触发场景**：复审或 Lead 按作者数值复算时，无法复现 1,555 B 残余；同一 `report.md` 在评审期间出现 21,142／30,201／24,041 B 三个自述；`hashes.txt` 仍以 `0BA1…` 为终值，且按自身规则已失效。
- **影响**：阻断"以作者证据支持接受"的路径；同一修订号下的就地改写没有可追溯的变更集，后续复审无法确认哪些字节属于 r66 的哪一次判断。
- **修复方向**：冻结 `design.md` 后重新生成 `report.md` 与 `hashes.txt`；按 r65 → r06 首测 → `0BA1…` → 终值三段逐项列出字节与行变化（含 1,555 B 残余）；行级表若只写片段字节，必须明确标注"片段"而非"整行"；重算 `report.md` 自身哈希；记录 `0BA1 → 9B19` 的写盘者与时间。
- **证据**：见 §2、§3、§4.1、§4.2、§7.1。
- **复审条件**：对象冻结；两个作者文件在同一终值上重算并互相一致；再按 §5–§8 复跑。

### 9.2 `NEW-S06-02`（LOW，provisional）V10 推进见证 head 前无显式撤销／有效期重验

- **行锚**：`design.md:4062`（V10）、`:4135`（检查点清单）、`:4137`（撤销生效时限）。
- **原句**：`:4062`："`V10` 追加审计记录、`fsync`、**验证方签名**并让外部见证**原子推进** `sequence`"；`:4137`："`revocation.state=revoked` 从下一次 `revalidateApproval` 读到的那一刻生效，最长延迟等于两个相邻检查点的间隔"。
- **判据**：撤销／过期应在推进权威见证 head 之前被重验；否则"下一次检查"可能晚于 head 已推进。
- **触发场景**：批准在"解码回执前"的重验之后、V10 原子推进之前被撤销或跨过 `valid_until`；代码按现有检查点清单继续推进 `sequence`。
- **影响**：见证 head 可能在已撤销／已过期批准下推进，审计链留下本应 closed 的 sequence；窗口很窄且需撤销时机配合，故定 LOW。
- **修复方向**：在 V10 验签之后、原子推进之前增加一次 `revalidateApproval`（或等价只读检查），失败按 `restore_unknown`／`ErrAdmissionClosed` 处置，并把该点写入 `:4135` 的检查点清单。
- **证据**：`:4135` 的五点清单没有"V10 推进前"；`:4137` 的最近一次重验是"解码回执前"。
- **复审条件**：冻结文本中出现"推进前重验"的明确句，且在 `:4135` 清单中可机核。

### 9.3 `NEW-S06-03`（LOW，provisional，跨文件待核对）批准记录缺少 `generation` 的可核定义

- **行锚**：`design.md:4369`（批准记录最小字段）、`:4135`（检查点）；跨文件 `decisions/framework.md:1118`、`:1192`（读取时快照）。
- **原句**：`:4369`："有效期与撤销状态 | `valid_from`、`valid_until`、`revocation.state` | 三者必填，不得'永久'"；ADR r10 修订说明（`:1192`）称将"给出 `generation` 的可复算三元定义、把撤销与有效期重验补到含 `P6` 的完整检查点集合"。
- **判据**：若上游要求按 `generation` 与有效期重验，设计必须给出 `generation` 的载体、派生与比较规则；只写 `valid_from`／`valid_until`／`revocation.state` 不足以实现。
- **触发场景**：实现者需要判断"本次批准是否同一代"，但设计没有字段；可能自行发明字段或跳过 generation 比较。
- **影响**：跨文件契约歧义；实现漂移风险。因 ADR r10 正在并发修订、且 ADR 侧结论不在本对象内，本项只作待核对线索。
- **修复方向**：待 ADR r10 冻结后，由架构侧确认 `generation` 的适用范围；若适用于批准记录，在 `:4369` 增字段并给出派生／比较；若不适用，明确写出排除理由与替代判据。
- **证据**：`decisions/framework.md` 读取时全文 `generation` 仅在 `:450` 与 r10 修订说明 `:1192` 命中，`:1118` 正文未检出该定义。
- **复审条件**：ADR r10 冻结；设计文本与 ADR 对 `generation` 的范围一致。

## 10 未运行项（NOT RUN，单列）

| 项 | 状态 | 说明 |
|---|---|---|
| `SEC-F007` | **NOT RUN** | `:4021`、`:4082`、`:4191` 三处均未运行；**不得改判 PASS** |
| Linux host／ACL／mount／loader／fd-exec／maps／network／witness | **NOT RUN** | `:4070(6)` 明列；Windows 解析与语义层不等于 Linux 实测 |
| DNS rebinding／TLS pin／IPv4-mapped IPv6／pin 轮换 | **NOT RUN** | 只有设计判据，无实验 |
| 真实信任根、签发／复核／执行／验证方密钥、两条批准记录、回执见证、审计 sink | **NOT RUN／不存在** | `:4282` 明示"均不存在" |
| Go `test`／`vet`／`build`／E2E | **NOT RUN** | 本端未运行；设计未实现（`newControlledRestoreDriver` 仍 4 参数、`NewRuntime(ctx,deps,restoreLease)` 调用为 0） |
| 模拟见证 | **仅测试替身／NOT RUN** | 未发现把模拟见证写成生产能力的句子；`restoreProcessTool` 只作软件合同测试替身（`:4189`／`:4261`） |

## 11 跨文件 ADR 快照（只读，范围外）

读取时 `decisions/framework.md`：

- 字节 **285,639 B**／LF **1,273**／SHA256 **`B432E44AEE7A6A68D541F3B77EAA0EF963A4F21B8DBD9E2C3DED6BF07030B050`**。
- `:1082` 现行修订为 `adr-0031-r10`；`:1084` 状态为"已接受"并声明决策接受级随现行修订号自动生效、现行修订须重新取得架构与安全两路接受。
- 交接时（同一评审会话早些时候）该文件为 272,588 B／1,271 行／`D8DBA64B3612CB7D5F814930855136B0C6888DE7717870C6F3CF1A79927CC7B4`；本次读取已变为上值，说明 ADR 正在被并行的 r10 修订改写。
- 相关行：`:1118`（原生耐久回执）、`:1129`（历史接受事实）、`:1147`（见证独立性与信任锚）、`:1166`（约束 8 解除条件第 6 条）、`:1174`（启动门禁）、`:1192`（r10 修订说明）。

**声明**：ADR 侧的任何结论（包括 r10 是否闭合 `generation`／P6 重验点、`adr-0031-r10` 是否被两路接受）都**不在本次评审对象内**，本报告不代裁。范围外 `N005`（ADR 状态与 `tasks.md` 冲突）仍为 out-of-scope BLOCKER，本端未改 ADR、未改 `tasks.md`。

## 12 复审条件

1. `design.md` 冻结：给出一段明确的只读窗口，记录冻结前后 B／LF／CR／SHA256；若窗口内再次变化，重新中止。
2. 识别 `0BA1 → 9B19` 的 +673 B 写盘者与写盘时间；确认是否还有未落盘的并行写。
3. 作者在最终冻结字节上重新生成 `report.md` 与 `hashes.txt`，修正自身哈希与行级绝对字节，逐项登记 r65 → r06 首测 → `0BA1` → 终值三段（含 1,555 B 残余）。
4. 在冻结字节上重跑必核项 A–D：A 逐条复算 R05 finding；B 复算 `nonce`／`bundle_digest`；C 复算三段字节账；D 复算撤销／信任锚与 `generation`。
5. 若作者升为 r67，按新修订重新派发；不得把 r66 的停前观察直接平移到 r67。
6. ADR r10 冻结后，再对 `NEW-S06-03` 作跨文件结论；ADR 侧接受由架构侧独立给出。

## 13 声明

本报告**不是真人签字，不构成安全签收，不构成评审通过，不构成 `design.md` r66 的安全侧接受**。它**不解锁任何卡**：不解除 `identity-auth-S4.24a` 的启动／关闭门禁，不解除生产 restore `closed`，不构成 `enter`／`resume` 解锁，不构成 ADR-0031 约束 8 满足，不构成任何人的成果接收或批准。`SEC-F007` 仍是 **NOT RUN**，不得改判 PASS。

本端未修改 `design.md`，未执行任何 Git 写操作，未删除或移动任何文件；写集仅为本报告与本目录 `hashes.txt`。

## 14 复算命令

```powershell
# 整文件
$p='D:\AID\VSCode\Codex\Han\.add\project\knowledge\design-workspace\identity-auth\design.md'
$b=[IO.File]::ReadAllBytes($p)
$s=[IO.File]::ReadAllText($p,(New-Object System.Text.UTF8Encoding($false)))
"B=$($b.Length) LF=$($s.Split("`n").Length-1)"
(([Security.Cryptography.SHA256]::Create().ComputeHash($b) | ForEach-Object {$_.ToString('X2')}) -join '')

# 块 / 单行（含/不含行尾需明确）
$lines=$s.Split("`n")
$blk=($lines[4071..4082] -join "`n") + "`n"   # :4072–:4083 含行尾 LF
$blkNoEol=($lines[4071..4082] -join "`n")     # 不含行尾
$l4435=$lines[4434]                            # :4435 去掉行尾
```

**本报告自身哈希**记录在 `hashes.txt`；`hashes.txt` 不记录自身哈希。