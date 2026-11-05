# B13J D03 独立评审报告：restore 工具、批准者与宿主隔离（ADR-0031）

- 项目项：`pending-B13J-D03`（restore 工具、批准主体、执行主体、宿主隔离、原生耐久回执）
- 评审角色：add-reviewer（独立非作者评审，Agent 产出；不是人的签字、生产批准或安全签收）
- 评审类型：设计评审 + 安全评审（同一评审人执行，结论分列）
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 记录时间：2026-11-01T23:11:04+08:00（Asia/Shanghai）
- 只读声明：未修改被评对象，未执行 Git 写操作，未删除或移动任何文件；仅新建本目录 `report.md` 与 `index.json`

## 1. 评审范围与基线

### 1.1 被评对象（本轮实测，与用户给定基线一致）

| 文件 | 改前 SHA256 / 字节 / 行 / crlf:bareLF | 改后 SHA256 / 字节 / 行 / crlf:bareLF |
|---|---|---|
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | `30B122A0…7D47BC` / 844091 / 4924 / 169:4755 | `79F226D5BB152C7A2DF359548CBFE24C306DF54A1B2EFD8BED71A80F488B2037` / 846117 / 4924 / 169:4755 |
| `.add/project/knowledge/decisions/framework.md` | `33281CB5…1CF532` / 137491 / 966 / 966:0 | `A80F5DD3F542AD15EC9BB717BAF20C4CA43EC83EB1B215171E1DA41516EB0589` / 153033 / 1048 / 1048:0 |
| `.add/project/knowledge/decisions/README.md` | `5075879A…5C10D` / 19724 / 106 / 2:104 | `A67B00D5BF050E100EA5AEC9DEBBFB8E5DABCE4463448658D68A5B4792E0F706` / 20764 / 108 / 2:106 |

### 1.2 作者证据（实测）

- `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d03/report.md`：`72517E391DD19AF73BF6512A2C304262886F0B3BBEEF8FB9C516F99DFDAC8C03` / 85277 B / 433 行 / 无 BOM / LF / 末尾单 LF / 控制字符 0
- `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d03/index.json`：`A38614C359ECDB1321FB1432D6CE7E7299D37E5C84FEA992123FE06C2F7B02E2` / 21611 B

### 1.3 评审依据

- 缺口与关闭条件：`.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md` §2.5（`:121-139`，`pending-B13J-D03`）
- 被评决策：`framework.md` ADR-0031（锚点 `:970`，正文 `:971-1048`）
- 硬边界与相邻决策：ADR-0019（`:418-494`）、ADR-0030（`:915-966`）、ADR-0027
- 设计窗口：`design.md` `:5`、`:38`、`:83`、`:3856-3872`、`:3874-3902`
- 索引：`README.md` `:5-47`、`:60-77`
- 真实消费者旁证：`cmd/han-server/recovery_sources.go:557-573`（`recoveryClosedRestoreDriver` 的 `Prepare`/`Replace`/`VerifyRestored` 均返回 `ErrAdmissionClosed`）、`cmd/han-server/recovery_runtime.go:596-599`（nil 时注入 closed driver）

## 2. 评审方法与限制

检查分两轮：第一轮做对象整体检查（14 条），第二轮沿真实消费者与上游依据做有界检查（runtime 代码、`tasks.md:656` 的 S4.5 登记、`plans/`、`stories.md`、`test-plan.md`、`decisions/identity.md` 的越界检索）。最小性用内存反向替换加 SHA256 复算验证，不依赖作者自述。

限制：未通读 `design.md` 全部 4924 行，语义结论覆盖上列窗口与字节证明未变的区域；未执行任何 restore 工具、数据库、备份演练、ACL 或进程隔离实测；未做 Markdown 渲染或 lint。

## 3. 逐条结论

### 3.1 ADR-0031 是否越界 —— PASS

`framework.md` 前 137491 字节 SHA256 精确等于改前 `33281CB5…1CF532`，因此 ADR-0019、ADR-0030、ADR-0027 的正文逐字节未变；ADR-0031 仅在文件末尾追加（`:968-1048`）。ADR-0031 `:977` 明确自我限定“只固定 §4.2 的组织与信任边界”，并逐项声明不改 ADR-0019 的独立故障域、keyring、不随 PostgreSQL 回退、POSIX only / Windows 生产 unsupported、UTC 微秒与 fail-closed 分类，不改 ADR-0030 的“首期不提供生产 anchor、S4.5 未交付、enter/install closed”，不改 ADR-0027 的 runtime 编排与正向准入。

`design.md` 的差异经反向重建证实恰为 6 行（`:5`、`:38`、`:83`、`:3858`、`:3862`、`:3870`），全部落在版本、追溯与 restore driver 段内；`restore-v1`/`verify-v1` 固定 argv、私有 stdin、receipt 字段、no-follow 稳定 fd、最小 ENV allowlist、Begin 前预检与 `restore_unknown` 的原句均保留。逐行 difflib 对照显示：`:3858` 把“unknown 时预检失败”改写为“首期 closed 拒绝”（强化而非放宽）；`:3862` 唯一被删除的片段是“root 保留部署锁，”，随即以“root 保留部署锁、Journal/keyring 与恢复 secret，不向子进程传递……”重述并加强；`:3870` 只增不删。未发现对既有软件合同的削弱或重写。

### 3.2 四项组织边界是否具体 —— PARTIAL（RV-01、RV-02）

- **批准主体**：具体。ADR-0031 `:1004`、`:1005`、`:1027` 要求部署方具名“恢复批准角色”分别签发备份来源与可执行文件两条独立批准记录，记录角色、人员、时间、依据、artifact digest、source installation、绝对路径、SHA256、owner/ACL 与只读挂载来源，且不得自批、不得兼任执行者、不得是本次 run 发起者；并明示当前项目未任命该角色、无已批准二进制。
- **执行主体**：具体。`:1006` 要求 root 启动独立非交互子进程，以独立于 root 的专用执行身份运行，root 不代替执行、不进入 shell、不重放。
- **root 与执行身份权限边界**：不完整（RV-01）。`:1007` 写明执行身份“不得写 Journal/keyring/锁/Store 绑定”，但 `design.md:3862` 只写“root 保留……不向子进程传递”，没有写入侧的禁止；`:3864` 把目标 DSN 经私有 stdin 交给执行身份，`:3866` 又由 root 在替换后经同一 Store 写 running/binding/审计。ADR-0031 全文无 `GRANT`、数据库角色或 schema 作用域限制，解除条件 `:1034` 只把“权限矩阵”列为未来待批准项，未定义任何可核机制。详见 RV-01。
- **原生耐久回执**：具体且有保留。`:1018` 明确“谁写”（工具或其备份系统原生写入）、“什么形态”（恢复目录、恢复任务记录或安装来源证据，须绑定 artifact digest、target、source installation）、“root 如何独立核对”（经独立读取路径与批准记录逐项核对，不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证），并把具体产品字段与读取方式如实标为 unknown。

### 3.3 安全不变量是否可执行 —— PARTIAL（RV-01 阻塞安全 PASS）

逐项对照 `design.md`：还原目标连接只经私有 stdin 成立（`:3864` 与 ADR `:1007` 一致）；不得传递部署锁/Journal/keyring/审批 key/管理入口/其它凭据成立（`:3862`）；不得启动 server/requeue/migration/bootstrap/worker 成立（`:3862` 保留原句）；“执行身份不得写 Journal/keyring/锁/Store 绑定”在 design 侧缺失且无强制机制（RV-01）。`design.md:3900`（恢复 secret 由 root 读 ENV）与 `:3902`（恢复期间不 requeue）与 ADR 不冲突。未发现互相矛盾或放宽，但存在一处不可核验的核心权限不变量。

### 3.4 禁止通道是否守住 —— PASS

ADR-0031 `:1025`（不得 shell、附加 argv、manifest 注入参数、自动重放、自动 fallback，不得以 file/SQL/fake、环境探测或“目录里最新二进制”填补）、`:1026`（caller 自报 done、exit=0、stdout receipt 自洽、DB 可连接、当前行数、自己刚写的 run 行、测试替身或空构造均不得作为恢复证明）逐项覆盖所列通道；`design.md:3862`（不用 shell、不允许 manifest 注入附加 argv）、`:3864`（stdout 仅 receipt、stderr 由 root 丢弃原文）、`:3868`（不自动重放、不接受人工勾选 done）、`:3870`（不接受 stdout receipt、exit=0、DB 可连接或本次 run 行自证）保留同样口径。

### 3.5 POSIX/Windows 矩阵 —— PASS

ADR-0031 `:1008-1016` 给出启动方式、进程组、文件、环境、平台门禁五行的 POSIX/Windows 对照，`:1016` 要求 Windows 在任何 Journal/DB/CAS/文件状态变更前返回 unsupported（退出码 2），`:1040` 明示“Windows 生产仍 unsupported，本 ADR 不提供 Windows 例外”。`design.md:3862` 同步为“POSIX 是唯一生产候选且首期仍 closed；Windows 生产 unsupported，须在任何状态变更前拒绝，仅允许软件合同测试”，与 `:3884`（参数/平台不支持退出 2）一致。未发现 Windows 生产例外或暗门。

### 3.6 `restore_unknown` 语义 —— PASS

ADR-0031 `:1028` 保留“无可核持久回执时保持 `restore_unknown`，由人工经原受控备份工具核验/修复后 verify/resume；不得用自动重放或人工勾选 done 越过”。`design.md:3868` 未被本轮改动，仍写明该暂停是“明确恢复结果”、不阻断其余软件设计、若要自动越过须回架构定义新的证明且不得在 Task 增加 bool。`:3858` 的改写只涉及“首期 closed”的入口表述，未触及 `restore_unknown` 窗口。

### 3.7 解除条件可核性 —— PASS

ADR-0031 `:1031-1039` 的八项均为可核事实：具名批准角色与两条独立记录、二进制 SHA256/owner/ACL/只读挂载/更新回滚责任、专用执行身份与权限矩阵、POSIX 隔离实测（固定 argv 无 shell、独立进程组与完整子进程树 join/reap、no-follow 稳定 fd、最小 ENV allowlist、目标连接外无网络）、原生耐久回执真实样本、真实旧备份一致性演练（明确排除 synthetic fixture）、架构与安全独立评审加 ADR 修订加正式 Task 登记、人的明确生产批准（并声明 Agent 产出、评审结论或测试通过均不替代）。人批准与 Agent 评审未被混同。未发现循环依赖：正式 Task 登记是“解除 closed”的前置，不是 ADR 成立的前置；ADR `:1030` 也声明不预占任何未登记 Task 编号。解除路径本身要求“新的 ADR 修订或新 ADR”，与本轮“首期不提供”不构成自锁死结。

### 3.8 诚实性 —— PASS

ADR-0031 `:975` 状态行明确“不构成人的成果接收、独立评审、安全评审、生产装配批准或真实工具/备份验证”，`:1029` 声明不证明任何工具、备份系统、托管平台、执行身份、ACL、进程隔离或旧备份演练通过，也不声称 Case18、S8.2 或恢复 E2E 通过，`:1048` 声明产出者为 Agent、不是人的签字。`README.md:39` 的状态列同样注明“无真实工具/备份/演练验证”。`design.md:83` 注明“仍 draft、人的精确接纳 unknown，不声明实现、工具/备份/演练验证或生产可用”。全文检索未出现“已验证/已演练/已隔离”的完成时断言；“实测通过”“已批准二进制”只作为未来门禁条件出现。RV-04 是作者证据报告的算术笔误，不是产品能力虚报。

### 3.9 版本与追溯 —— PASS

`design.md:5` = `revision: identity-auth-design-ir01-r59`，`:38` = **设计版本**：`identity-auth-design-ir01-r59`，两处一致；`:7` 仍为 `last_updated: 2026-11-01` 且未在差异行内。`:83` 的 r59 段记录“输入 r58 SHA256 `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC`”，与实测改前哈希精确相等；该段如实标注 Arch-Designer（Agent，非人的签字）、draft、人的精确接纳 unknown、不声明实现或验证。ADR-0031 `:1044` 的采用记录同样注明 draft、人的精确接纳 unknown。

### 3.10 README 索引 —— PASS

`README.md:39` 为全局决策索引行（编号 / 链接 `framework.md#adr-0031` / 状态 / 目的 / 领域 / 影响范围 / 引用次数），`:45` 在领域文件表的 framework 行追加 `ADR-0031`，`:71` 为场景查询行并引用 `framework.md（ADR-0031）` 与 `framework.md（ADR-0027、ADR-0018）`。链接锚点与 `framework.md:970` 的 `<a id=adr-0031></a>` 对应。引用次数列为 `1`，与 ADR-0030、ADR-0028、ADR-0024 等既有“单一采用记录”行口径一致，也与 `framework.md:1044` 的唯一采用记录相符。

### 3.11 格式不变量 —— PASS

- `framework.md`：无 BOM；1048 行；crlf 1048 / bareLF 0；末尾单 CRLF；控制字符（除 CR/LF/TAB）0。
- `design.md`：无 BOM；4924 行；crlf 169 / bareLF 4755（与改前完全相同）；末尾单 LF；控制字符 0；唯一 TAB 在字节偏移 466829（字符偏移 309035，行 3303 的既有 Go 代码块 `CheckLockedUser`），改前对应偏移为 466069 = 466829 − 760，与 D02 基线记录一致，可证非本轮引入。
- `README.md`：无 BOM；108 行；crlf 2 / bareLF 106；末尾单 LF；控制字符 0。

### 3.12 改动最小性 —— PASS

内存反向替换（`:5`/`:38` 将 `r59` 还原为 `r58`；`:83`、`:3858`、`:3862`、`:3870` 用作者报告中改前原文整行替换）后重新编码 UTF-8，得 **844091 B / `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC`**，与改前基线逐字节相符；差异行集合实测为 **`[5, 38, 83, 3858, 3862, 3870]`**，恰为六行，`:7` 未变，`:3874-3902` 零改动。字节增量分解为 `:83` +760、`:3858` +94、`:3862` +666、`:3870` +506，合计 +2026 = 846117 − 844091。README 反向删除三处新增后得 **19724 B / `5075879A…5C10D`**，与 D02 基线完全一致。`framework.md` 前 137491 字节哈希等于改前整文件哈希，为纯追加。

### 3.13 验收命令 —— PASS

gap-map §2.5 的三条命令实跑结果见第 7 节：两条 `rg` 均退出 0 且命中具体条款，`git diff --check` 退出 0（仅 Git autocrlf 咨询性 warning，非空白错误）。

### 3.14 对 D04 的承接边界 —— PASS（附 RV-01/RV-05 注记）

ADR-0031 `:1040` 明确不适用于“D04 承接的部署独占、`ArticleAccountPort` 与 recovery secret candidate”，不与 D04 争夺部署独占或 secret candidate 的定义权；`design.md:3862` 中 root 保留“部署锁、排空/独占”只是在既有 runtime 合同内重申，未新增 D04 的装配细节。未发现与 D04 的冲突。两处接口级注记需在 D04 落地前固定：其一，执行身份的数据库权限边界与强制机制（RV-01，本轮应先在决策/设计中建立要求）；其二，POSIX 身份切换机制（RV-05，可并入 D04 或实现任务登记）。

## 4. Findings

### `B13J-D03-RV-01` —— MEDIUM —— 必须本轮修（阻塞设计 PASS 与安全 PASS）

- **位置**：`.add/project/knowledge/decisions/framework.md:1007`、`:1034`、`:1044`；`.add/project/knowledge/design-workspace/identity-auth/design.md:3862`、`:3864`、`:3866`
- **事实**：ADR-0031 `:1007` 规定执行身份“不得写 Journal/keyring/锁/Store 绑定”，`:1034` 的解除条件也要求“专用执行身份与权限矩阵经批准：不得写 Journal/keyring/锁/Store 绑定”。但 `design.md:3862` 只写“root 保留部署锁、Journal/keyring 与恢复 secret，不向子进程传递……”，即只禁止“传递”，没有写入侧的否定句；`:3864` 又把目标 DSN 经私有 stdin 交给执行身份（执行身份因此持有目标库写能力）；`:3866` 由 root 在替换后经“同一 Store 短事务创建/核 running＋不可变 restore binding/审计”，说明绑定与审计就在被还原的同一个目标库内。ADR-0031 全文检索 `GRANT`、数据库角色、schema 作用域限制均为 0 命中，解除条件只把“权限矩阵”留给未来批准，未定义任何强制或核验机制。ADR 采用记录 `:1044` 却称 `:3862` 已“固定……root 权限边界”，与设计实际文本不符。
- **触发场景**：未来按本 ADR 解除 closed、按 `design.md:3862` 装配执行身份时，执行者以目标 DSN 执行整库还原，dump 若包含 recovery Store / Journal 相关 schema，则替换动作本身即改写绑定与审计表；此时“不得写 Store 绑定”既无机制保证，也无法判定是否被违反。
- **影响**：生产当前 closed（`cmd/han-server/recovery_sources.go:557-573` 的 closed driver 对 `Prepare`/`Replace`/`VerifyRestored` 全部返回 `ErrAdmissionClosed`，`recovery_runtime.go:596-599` 在 nil 时注入），不存在现网暴露。但 gap-map §2.5 的关闭条件要求“ACL/进程隔离”具体且“安全评审 PASS”，该项是权限边界的核心不变量，当前不可核验，安全评审不能判 PASS；`framework.md:1044` 的追溯陈述也失真。
- **建议**：三处同修。其一，在 `design.md:3862` 补写入侧否定矩阵与强制机制：限定执行身份使用仅能写应用 schema 的数据库角色（或还原到隔离库后由 root 受控切换、或明确把 recovery Store/Journal schema 排除在 artifact 之外），并写明 root 在替换后如何核对绑定/审计只能由 root 建立。其二，在 ADR-0031 §8 增加负向验证门禁：证明执行身份凭据写入 Journal/keyring/锁/Store 绑定必须失败。其三，把“不得写 Store 绑定”明确为“不得产生权威绑定/审计行”，并定义整库还原与绑定重建的关系。
- **处置**：本轮回修后做定向复核。

### `B13J-D03-RV-02` —— LOW —— 建议同轮修（不单独阻断）

- **位置**：`.add/project/knowledge/decisions/framework.md:1004`、`:1006`、`:1035`；`.add/project/knowledge/design-workspace/identity-auth/design.md:3862`
- **事实**：`design.md:3862` 只写“批准者不得同时是执行者或本次 run 发起者”，未同步 `:1004` 的“工具供应方自批”禁令；未写 `:1006` 的“root 不代替执行、不进入 shell、不重放”；未写 `:1035` 的“目标连接外无网络”。三处均为 ADR 已定的宿主隔离/职责分离不变量。
- **影响**：ADR 仍是权威且生产 closed，单独不构成阻断；风险在于只看设计的实现者会漏掉网络出口限制与供应方自批禁令。
- **建议**：与 RV-01 同一次设计修订补齐 `:3862`，避免二轮返工。
- **处置**：建议同轮；若不同轮，须在 D04 装配前补齐。

### `B13J-D03-RV-03` —— LOW —— 可并入 D04 或后续修订

- **位置**：`.add/project/knowledge/decisions/framework.md:981`、`.add/project/knowledge/design-workspace/identity-auth/design.md:3858` 与 `.add/project/knowledge/decisions/framework.md:926`
- **事实**：ADR-0031 与设计把 closed 范围写成“生产 enter/resume 的 DB 替换（路径）”，ADR-0030 `:926` 写的是“生产 recovery enter/install 继续返回 closed”。两者口径宽窄不同。
- **影响**：叠加后仍全闭（ADR-0030 继续有效，`design.md:3892` 仍写未配置 anchor 即 enter/install closed），但单读 ADR-0031 可能误以为 DB 替换之外的 enter 可用。
- **建议**：统一措辞或显式交叉引用 ADR-0030；不单独阻断。

### `B13J-D03-RV-04` —— LOW —— 证据更正，不阻断

- **位置**：作者证据 `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d03/report.md:81`（与同文件 `:86`、`:92` 自相矛盾）
- **事实**：报告称 `:83` 为“同行末尾追加（前 744 字符逐字符不变）”，但实测改前块 1184 字符、改后 1646 字符，且改前整段是改后的前缀（公共前缀 1184 字符），尾部追加 462 字符 / 760 字节。
- **影响**：不影响被评文件状态与反向重建结论（本文已独立复算通过），只影响证据自洽性。
- **建议**：改为“整段改前文本保留，尾部追加 462 字符 / 760 字节”。

### `B13J-D03-RV-05` —— LOW —— 可并入 D04 或实现任务

- **位置**：`.add/project/knowledge/decisions/framework.md:1006`、`:1012`；`.add/project/knowledge/design-workspace/identity-auth/design.md:3862`
- **事实**：要求“独立于 root 的专用执行身份”与“固定 argv 直接 exec、无 shell、无附加 argv”，但未指定 POSIX 身份切换机制；`su`/`sudo`/脚本包装会与“无 shell、无附加 argv”直接冲突。
- **影响**：装配阶段可能选出违反隔离合同的实现方式。
- **建议**：在 D04 或正式实现任务中固定 `fork` + `setgroups/setgid/setuid` + `exec`（或经批准的单一 launcher），并显式禁止 su/sudo 类包装。

## 5. 评审结论

- **设计评审结论：CHANGES REQUIRED。** ADR-0031 的“首期不提供”决策本身成立，边界未越界、最小性与追溯均通过；但 RV-01 使 `design.md:3862` 的权限边界陈述与 ADR `:1007` 不一致，而 ADR 采用记录 `:1044` 又称 `:3862` 已固定该边界，属设计侧可核性缺口，不能判 PASS。
- **安全评审结论：CHANGES REQUIRED。** gap-map §2.5 要求“设计 Review 与安全评审 PASS”。“执行身份不得写 Journal/keyring/锁/Store 绑定”是权限矩阵的核心安全不变量，当前既未写入设计、也无数据库角色/schema 作用域等强制机制、解除条件中也没有对应的负向验证，安全门禁无法客观核验，故安全评审不 PASS。
- **总判定：CHANGES REQUIRED。**

处置分级：

- **必须本轮修**：`B13J-D03-RV-01`（设计 + 安全共同阻断项）。
- **建议同轮修**：`B13J-D03-RV-02`（与 RV-01 同一行，成本低）。
- **可并入 D04 或后续修订**：`B13J-D03-RV-03`、`B13J-D03-RV-05`。
- **证据更正、不阻断**：`B13J-D03-RV-04`（作者下次更新证据时一并改）。

本判定不关闭 `pending-B13J-D03`，不解除 `blocked-upstream`，不解锁 S4.24，不构成人的接收、设计 Gate 或安全签收。

## 6. 格式与最小性实测

| 文件 | BOM | 行数 | crlf / bareLF | 末尾 | 控制字符（除 CR/LF/TAB） |
|---|---|---|---|---|---|
| `identity-auth/design.md` | 无 | 4924 | 169 / 4755 | 单 LF | 0（唯一 TAB 为既有 Go 代码块，字节 466829） |
| `decisions/framework.md` | 无 | 1048 | 1048 / 0 | 单 CRLF | 0 |
| `decisions/README.md` | 无 | 108 | 2 / 106 | 单 LF | 0 |

反向重建（独立复算，不使用作者副本）：

| 文件 | 还原方法 | 结果 | 判定 |
|---|---|---|---|
| `design.md` | `:5`/`:38` 版本号 r59→r58；`:83`/`:3858`/`:3862`/`:3870` 用报告改前原文整行替换 | 844091 B / `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC`；差异行 `[5, 38, 83, 3858, 3862, 3870]` | 精确回到基线 |
| `framework.md` | 取前 137491 B 复算 | `33281CB5…1CF532`，与改前整文件一致 | 纯追加，前文逐字节未变 |
| `README.md` | 删除 `:39` 全局行、`:45` 的 `、ADR-0031`、`:71` 场景行 | 19724 B / `5075879A…5C10D` | 精确回到 D02 基线 |

## 7. 验收命令实测

1. `rg -n "restore-v1|verify-v1|批准者|批准主体|执行主体|进程树|no-follow|restore_unknown" .add/project/knowledge/design-workspace/identity-auth/design.md`
   - 退出码 0；命中 `:83`、`:263`、`:1535`、`:1685`、`:3654`、`:3828`、`:3858`、`:3860`、`:3862`、`:3868`、`:3870`、`:3957`、`:4203`，均含具体工具/批准主体/隔离/`restore_unknown` 条款。
2. `rg -n "restore|恢复工具|批准|进程隔离|no-follow" .add/project/knowledge/decisions/framework.md`
   - 退出码 0；69 行命中，覆盖 ADR-0031 `:971-1048` 全文及相邻 ADR 的边界引用。
3. `git diff --check -- .add/project/knowledge/design-workspace/identity-auth/design.md .add/project/knowledge/decisions/framework.md`
   - 退出码 0；无 whitespace error，仅输出 Git autocrlf 咨询性 warning（LF 将在 Git 触碰时转 CRLF），非空白错误。

补充实测：`rg "GRANT|数据库角色|DB 角色|写权限"` 在 ADR-0031 范围 0 命中（RV-01 依据）；`rg "ADR-0031|B13J-D03"` 在 `tasks.md`、`plans/`、`stories.md`、`test-plan.md`、`decisions/identity.md` 全部 0 命中（无越界登记）；`rg "S4\.22|S4\.23|S4\.24"` 在 `framework.md`、`design.md` 0 命中（无未授权 Task 编号）。`tasks.md:656` 仍为未勾选 `- [ ] identity-auth-S4.5 选择并交付生产 CAS JournalHeadAnchor provider 与独立故障域验证 [P]`。

## 8. 未验证范围、边界与复审条件

未验证：

- 未执行任何 restore 工具、备份系统、数据库、ACL、进程隔离或真实旧备份演练；未取得原生耐久回执样本。
- 未验证任何批准角色、执行身份、只读挂载或宿主隔离已存在；ADR 与设计均如实标为 unknown。
- 未通读 `design.md` 全部 4924 行；未做 Markdown 渲染或 lint。
- 未验证 `tasks.md`、`plans/` 等文件相对 D03 改动前快照的字节一致（无该快照）；现有依据是 D03 writeSet 声明、内容检索 0 命中与 mtime（`tasks.md` 21:45、`plans` 20:07，早于三个被评文件的 22:51-22:52）。
- 观察到工作区根存在未跟踪条目 `?? $prof` 与 `?? '){`，不属本项 writeSet、无法归属、无删除授权，仅记录不处置。

边界：本报告是独立非作者评审的 Agent 结论，不代表人的成果接收、设计 Gate、安全签收或生产批准；不声称 Case18、S8.2 或任何恢复 E2E 通过；不解除 `blocked-upstream`；不关闭 `pending-B13J-D03`；不解锁 S4.24。

复审条件：

1. RV-01 修复后，对 `design.md` 与 `framework.md` 做定向复核：确认 `:3862` 含写入侧否定矩阵与强制机制、ADR §8 含对应负向验证门禁、`:1044` 的追溯陈述与设计实际文本一致；并重跑本轮哈希、格式与差异行检查。
2. 若 RV-02 同轮修复，一并核对 `:3862` 的三条 ADR 否定句。
3. RV-03、RV-05 转入 D04 或后续修订时，需在对应对象中复核，不回写本轮判定。
