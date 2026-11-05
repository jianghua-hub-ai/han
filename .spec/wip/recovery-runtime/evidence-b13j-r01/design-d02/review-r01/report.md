# B13J-D02 生产 CAS Anchor provider 选型独立评审报告（review-r01）

## 0. 角色、对象与写集

- 执行角色：`add-reviewer`（非作者独立评审）。未修改被评审产物，未替作者整改。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（本轮实测）。
- objectType：复合对象 `design + adr + index`。本轮不套用代码提交、编译、覆盖率或 E2E 条款。
- 固定对象与修订：
  - `.add/project/knowledge/design-workspace/identity-auth/design.md`：目标 SHA256 `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC` / 844091 B；改前 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C` / 842966 B。
  - `.add/project/knowledge/decisions/framework.md`：目标 SHA256 `31960C7A7BAE3EA3B2B250A6ED9DC1FB1E1F81464520E03A0D42C5E0B491C9BA` / 137200 B；改前 `5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A` / 126590 B。
  - `.add/project/knowledge/decisions/README.md`：目标 SHA256 `5075879AC7F3E31D83A06E843BCCE80CAA8DBA27F588E39D700E7F6C3765C10D` / 19724 B；改前 `35A5623E7E3CFCC140DA0DE9C8B218210C0E34A850A26B5CDD450B81DA40CC63` / 18948 B。
- 作者证据：`design-d02/report.md` SHA256 `2ED84374590D5B42805A8F1333F90A5EE60E4EA6B744A6C90CA6BE6CD146F591` / 16136 B；`design-d02/index.json` SHA256 `7AE883504FA6047D1A3080A828869C711BB40C3CAF97F3F0CD280AAD47331E7D` / 14978 B。
- 写集：仅本报告 `.spec/wip/recovery-runtime/evidence-b13j-r01/design-d02/review-r01/report.md` 与同目录 `index.json`。未改 `framework.md`、`design.md`、`README.md`、`tasks.md`、Plan、Stories、test-plan、代码或测试；未做 Git 写；未删除/移动文件。
- 边界：本评审不代替人的精确接纳、设计 Gate、Task 登记或后续实施授权；不声称 Case18、S8.2、真实 provider 或 E2E 通过；不解除 `blocked-upstream`。

## 1. 实际检查范围与证据限制

### 1.1 两轮检查

- Round 1（结构与声明核验）：复测三文件 SHA256、字节、行数、CRLF/bareLF、BOM、末尾行尾、控制字符；对 `design.md` 与 `README.md` 做内存反向重建；核对 `framework.md` 前 126590 B 前缀哈希；实测 `design.md` 差异行集合与边界文件。
- Round 2（语义与契约核验）：读取 ADR-0019 `418-494`、ADR-0030 `913-966`；读取 `design.md:1-10`、`:36-40`、`:81-84`、`:3577`、`:3888-3900`；读取 `README.md:5-47`、`:60-75`；读取 `gap-map.md` 的 A08、§2.4、U3 以及作者报告/索引。逐项检查继承边界、S4.5 状态、候选诚实性、fallback、配置键、解除条件、版本追溯、索引、最小性、越界和 D03/D04 可承接性。

### 1.2 未验证范围

- 未运行 Go、PostgreSQL、HTTP、restore、Case18、S8.2、E2E、集成测试或任何真实 CAS provider 验证。
- 未对 `framework.md` 全文 966 行、`design.md` 全文 4924 行做全文件语义审计；语义结论只覆盖上述命名窗口和由精确 diff 证明未变的区域。
- 未运行 markdown lint/render；格式结论来自字节级扫描，不证明渲染效果。
- 工作树已有 293 项未提交变更，不能把 `git diff HEAD` 全量归因于 D02；本报告只对固定对象和固定修订作结论。
- 任务书标注 `tasks.md:656` 为 S4.5 登记行，但本轮实测 `:656` 不是 S4.5；真实登记行为 `tasks.md:658`，仍为 `[ ] identity-auth-S4.5 ... [P]`。该行号漂移不影响“S4.5 未完成”的结论，但任务书证据行号应按实测更正。

## 2. Round 1：基线与字节不变量

### 2.1 三文件实测

| 文件 | SHA256 | 字节 | 行数 | CRLF | bareLF | BOM | 末尾 | 除 CR/LF/TAB 外控制字符 |
|---|---|---:|---:|---:|---:|---|---|---|
| `decisions/framework.md` | `31960C7A7BAE3EA3B2B250A6ED9DC1FB1E1F81464520E03A0D42C5E0B491C9BA` | 137200 | 966 | 966 | 0 | false | `0d0a` | 0 |
| `identity-auth/design.md` | `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC` | 844091 | 4924 | 169 | 4755 | false | `0a` | 0（另有 1 个既有 TAB） |
| `decisions/README.md` | `5075879AC7F3E31D83A06E843BCCE80CAA8DBA27F588E39D700E7F6C3765C10D` | 19724 | 106 | 2 | 104 | false | `0a` | 0 |

三项均与任务书“改后”基线和作者索引一致。`design.md` 的唯一 TAB 位于字节偏移 `466069`，对应 UTF-8 字符偏移 `308573`（`len(d[:466069].decode(utf-8)) == 308573`），位于既有 Go fenced code block 的 `CheckLockedUser` 行；改前重建中同一 TAB 存在，非本轮引入。作者索引写作“offset 308573”应按字符偏移理解，不构成事实错误，但应在报告中注明单位以避免字节偏移误读。

### 2.2 `framework.md` 前缀未变

对目标文件前 126590 B 单独计算 SHA256：

`5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A`

与改前整文件哈希精确一致。ADR-0019 位于 `418-494`，因此 ADR-0019 及该文件此前全部内容未被本轮改写；ADR-0030 只追加在 `912-966`。

### 2.3 `design.md` 恰为六行改动

将第 `5`、`7`、`38` 行恢复为 r57/2026-10-30，将第 `83` 行截断到 ` **r58 / B13J-D02` 之前，并将第 `3577`、`3892` 行恢复作者索引记录的改前原文后，内存重建结果为：

- 字节：`842966`
- SHA256：`504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`
- 与改前基线：精确相等
- 逐行差异索引：`[5, 7, 38, 83, 3577, 3892]`
- 字节增量：`1125` = 第 83 行 `+694` + 第 3577 行 `+198` + 第 3892 行 `+233`

结论：`design.md` 无第七行改动、无表格重排、无换行漂移、无顺手格式改动。

### 2.4 `README.md` 反向重建

删除第 `38` 行新增全局索引行、将第 `44` 行末尾的 `、ADR-0030` 删除、删除第 `69` 行新增场景行后，内存重建结果为 `18948 B` / SHA256 `35A5623E7E3CFCC140DA0DE9C8B218210C0E34A850A26B5CDD450B81DA40CC63`，与改前基线精确一致。行数由 104 增至 106，与两条新增 LF 行一致；两处既有 CRLF 未变。

### 2.5 边界文件

- `tasks.md` 中 S4.5 真实登记行为 `:658`，仍为 `[ ]`；`:656` 当前是设计/测试来源文本。D02 未改该文件，也未把 S4.5 改为完成。
- 在 `tasks.md`、`plans/`、`stories.md`、`test-plan.md`、`decisions/identity.md` 中检索 `ADR-0030` / `B13J-D02`，均无命中。
- 在目标 ADR/design 中检索 `S4.22`、`S4.23`、`S4.24`，无命中；未预占未授权 Task 编号。
- 无证据表明本轮越界写入；但工作树为脏，无法把全部未提交差异归因 D02。

## 3. Round 2：逐条结论

### 3.1 ADR-0019 未被覆盖

**结论：PASS。** `framework.md` 前 126590 B 的哈希与改前整文件哈希精确一致，ADR-0019（`:418-494`）未被改写。ADR-0030 `:922` 明确只固定 ADR-0019 留给 S4.5 的生产 provider 选择、配置键与启动装配边界，并把独立故障域、不随 PostgreSQL 回退、POSIX only / Windows production unsupported、UTC 微秒、CAS 协议、key 轮换/退役、三方对账、fail-closed 全部列为硬边界；`:947` 再次逐项重申。未发现对 ADR-0019 已锁定决定的软化或改写。**限制：** 这是对 ADR-0019 固定窗口和文件前缀的核验，不是全文语义审计。

### 3.2 结论与既有登记一致

**结论：CHANGES REQUIRED（仅因 RV-01）。** `tasks.md` 实测 S4.5 在 `:658` 仍为 `[ ]`，未完成；ADR-0030 `:926`、`:951` 明确 S4.5 维持未交付、生产 `enter`/`install` 保持 closed；`:952` 明确不证明 Case18、S8.2 或恢复 E2E 通过；`:963` 对设计仍标注 draft、人的精确接纳 unknown。未发现“已实现/已验证/已通过真实故障域/S4.5 已完成”的虚假断言。**但** ADR-0030 `:924` 把 `identity-auth/design.md:3892` 描述为“记实际产品/凭据/独立故障域 unknown”，而当前 `:3892` 已改为“当前无已登记产品/凭据/独立故障域”；见 RV-01。

### 3.3 候选对比的诚实性

**结论：PASS WITH MINOR（RV-02）。** A/B/C/D 四类候选覆盖了独立小数据库、外部秘密/KMS、对象存储条件写和“首期不提供”四条明显路径，没有遗漏与当前约束直接相关的第五类。A 的“独立 schema 与业务库同实例仍随同一备份/故障域回退”判断成立；B 把 Vault KV 的版本 CAS 与 KMS 的密钥/封装职责区分，方向正确；D 的不选理由与 ADR-0019 的 fail-closed、测试替身不得作为生产证据一致。**但** B 的“KMS 本身不提供通用条件写”是类别级概括，未逐产品/版本/API 核验；C 的“对象存储天然独立于 PostgreSQL”过强，同账号、区域、网络或控制面仍可能关联故障。ADR `:952` 已声明不证明任何产品/环境验证，因此这不是“把未验证写成已验证”的阻断问题，详见 RV-02。

### 3.4 未引入 ADR-0019 禁止的 fallback

**结论：PASS。** ADR-0030 `:926`、`:948`、`:949`、`:950` 与 `design.md:3577`、`:3892` 均明确：不得以 file、SQL、业务 PostgreSQL 表、内存、fake 或环境探测自动选择路径；测试替身不得注册为生产 provider；root 不得按环境、hostname、数据库、文件存在性或其他配置推导 adapter；未配置即不启动 anchor 能力并保持 closed。未发现自动降级到只读模式或测试替身的表述。

### 3.5 配置键语义

**结论：PASS。** `HAN_RECOVERY_ANCHOR_ADAPTER` 与 `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 在 ADR `:926`、`:949` 与 `design.md:3892` 的语义一致：两者必须同时显式提供；无代码默认值；互不推断；生产 adapter 必须已登记，配置路径必须是受保护绝对路径；未配置即不启动 anchor 相关能力，生产 `enter`/`install` 保持 closed。未发现相反默认值或“缺失时自动推导”的暗示。

### 3.6 解除条件可执行性

**结论：PASS。** ADR-0030 `:953-958` 的五项条件分别覆盖：具体产品/部署形态与独立故障域证明；凭据来源、最小权限/ACL、拓扑、备份/恢复/轮换与运维责任；无隐式默认/自动 fallback 的配置与启动装配合同；真实独立故障域下的 CAS 冲突、stale revision、missing/rollback、pending/committed、崩溃恢复与权限失败验证并保留原始证据；架构与安全独立评审、ADR 修订/新增决策、正式 Task 规划登记实现与验证范围。已包含“必须正式登记实现任务”和“必须独立评审”，可作为后续门禁核验。

### 3.7 版本与追溯一致性

**结论：PASS。** `design.md:5 revision` 与 `:38 设计版本` 均为 `identity-auth-design-ir01-r58`；`:7 last_updated` 为 `2026-11-01`；`:83` 的 r58 追溯行记录输入 r57 SHA256 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`，与改前基线精确一致。`:83` 明确标注实际产出者 Arch-Designer（Agent）、非人的签字、仍 draft、人的精确接纳 unknown，且不声明实现、独立故障域验证或生产可用。r58 内容追加在既有 r57 追溯行的同一物理行内，但有独立加粗段标识；这不改变版本事实，也不构成本轮 finding。

### 3.8 README 索引完整性

**结论：PASS。** `README.md:38` 的全局决策索引含 ADR-0030，`:44` 的领域文件表追加 ADR-0030，`:69` 的场景查询表含 ADR-0030。三处链接均为 `framework.md#adr-0030`，与 `framework.md:915` 的 `<a id=adr-0030></a>` 对应。引用次数列 `1` 与 `framework.md:963` 的唯一实际采用记录一致；`decisions-guide` 的口径是引用记录数，不是全文提及数。反向重建证明三处改动之外无索引漂移。

### 3.9 格式不变量

**结论：PASS。** 三文件 BOM 均为 false；`framework.md` 为 966 行纯 CRLF、末尾 `0d0a`；`design.md` 为 4924 行、crlf=169、bareLF=4755、末尾 `0a`；`README.md` 为 106 行、crlf=2、bareLF=104、末尾 `0a`。三文件均无 `<32` 且非 CR/LF 的其他控制字符。`design.md` 唯一 TAB 的字节偏移为 466069、字符偏移为 308573，位于既有 Go 代码块；改前重建中同一 TAB 存在，非本轮引入。均与任务书基线一致。

### 3.10 改动最小性

**结论：PASS。** `design.md` 差异行恰为 `5`、`7`、`38`、`83`、`3577`、`3892` 六行；反向重建得到改前 SHA256 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`。`framework.md` 前 126590 B 逐字节未变，新增内容仅为 `912-966` 的 ADR-0030。`README.md` 反向重建精确等于改前 SHA256。未发现表格重排、格式漂移或顺手修改。

### 3.11 边界未越

**结论：PASS（带基线说明）。** `tasks.md` 的 S4.5 真实行为 `:658` 且仍为 `[ ]`；`tasks.md`、`plans/`、`stories.md`、`test-plan.md`、`decisions/identity.md` 无 `ADR-0030` / `B13J-D02` 命中；目标文件中无 `S4.22` / `S4.23` / `S4.24` 等未授权 Task 编号。未发现 D02 越界写入证据。**基线说明：** 任务书写 `tasks.md:656`，实测已漂移到 `:658`；工作树有 293 项既有未提交变更，不能把脏树全量归因 D02。

### 3.12 对 D03/D04 的可承接性

**结论：PASS（RV-01 修复后承接）。** ADR-0030 `:959` 明确把 restore 工具/批准者/宿主隔离、部署独占、`ArticleAccountPort` 与 recovery secret candidate 排除在本 ADR 之外，留给其他设计/决策承接；`:951` 保留 S4.5 的 provider、配置、装配与真实验证责任；`:953-958` 给出未来解除 closed 的门禁。D03（restore 工具/批准者/宿主隔离）与 D04（root 装配/部署锁/ArticleAccountPort/secret candidate）可以在该边界上继续，不需要本 ADR 预设具体接口。**前置：** 修复 RV-01 后，D03/D04 才能把 `design.md:3892` 作为准确引用；除此之外未发现必须在本轮补的接口级缺口。

## 4. Findings

### B13J-D02-RV-01 — MEDIUM — 必须本轮修

- **位置：** `.add/project/knowledge/decisions/framework.md:924`；受影响引用对象 `.add/project/knowledge/design-workspace/identity-auth/design.md:3892`；相关缺口来源 `framework.md:965`。
- **现状证据：** ADR-0030 `:924` 原文为“`identity-auth/design.md:3892` 记实际产品/凭据/独立故障域 unknown”；当前 `design.md:3892` 原文为“当前无已登记产品/凭据/独立故障域”。两者不是同一事实：前者陈述实际产品/凭据/故障域的事实状态为 unknown，后者只陈述登记表没有登记项；没有登记项不等于实际事实已被证明或已消失。
- **触发场景：** 读者按 ADR 的精确行号回查设计，或 D03/D04 以 `:3892` 作为“unknown”证据；会得到与引文不一致的原文。
- **影响：** 不改变运行时行为，生产仍 closed；但已接受 ADR 的事实引用不准确，U3 依据链和后续消费者核对出现裂缝。按本轮“授权只覆盖决策取舍、不覆盖事实正确性”的要求，不能以用户已授权“首期不提供”而接受该引用错误。
- **必须修复（二选一，推荐第 1 项）：**
  1. 将 `design.md:3892` 改为同时保留两个层次，例如“当前无已登记 adapter；实际产品/凭据/独立故障域 unknown”，使 ADR 引文与设计原文一致；或
  2. 将 `framework.md:924`（必要时同步 `:965`）改为当前设计原文的准确含义，并明确“无已登记；实际产品/凭据/独立故障域仍 unknown”。
- **验收条件：** 两处精确字符串一致；`design.md` 反向重建仍得到正确改前基线；不新增“已实现/已验证/已通过真实故障域”断言；若任一目标文件改变，重新记录 SHA256/字节/格式并做定向复审。
- **证据：** 上述行原文逐字读取；三文件哈希与格式实测见 §2；`framework.md` 前缀哈希证明 ADR-0019 未被改写。

### B13J-D02-RV-02 — LOW — 可并入 D03/D04 或后续 ADR 修订

- **位置：** `.add/project/knowledge/decisions/framework.md:937-938`；作者报告 `design-d02/report.md:36-37` 有同义表述。
- **现状证据：** B 行用类别级判断“KMS 本身不提供通用条件写”；C 行把对象存储写成“天然独立于 PostgreSQL”。前者未逐产品/版本/API 核验；后者把“不同服务”过度等同于“独立故障域”，同一云账号、区域、网络或控制面仍可能关联故障。
- **影响：** 候选对比的严谨性和可辩护性下降，不改变 D 的选择；ADR `:952` 已明确不证明任何产品/环境验证，故不是阻断性 overclaim。
- **建议：** C 改为“与 PostgreSQL 为不同服务，但独立故障域须按账号/区域/网络/控制面证明”；B 改为“未逐产品/版本核验；KMS 通常不是通用 CAS KV 存储”。若本轮只修 RV-01，可将本条并入 D03/D04 或下一次 ADR 修订。
- **验收条件：** 若修改，候选表与作者报告同步；仍保留 `:952` 的未验证声明。

## 5. 总判定与关闭条件

**总判定：CHANGES REQUIRED。** 一句话理由：ADR-0030 对当前 `design.md:3892` 的“unknown”引用已与原文“无已登记”不一致，属于已接受决策的事实/追溯错误；这不能由“首期不提供”的决策授权豁免。

- 必须本轮修：`B13J-D02-RV-01`。
- 可并入后续：`B13J-D02-RV-02`。
- 未发现 HIGH 或 CRITICAL finding；未发现 ADR-0019 被覆盖、自动 fallback、配置键矛盾、版本号错误、README 索引缺失、格式漂移或越界改 Task 的证据。
- 本评审不声称 Case18、S8.2、真实 provider、恢复 E2E 或生产装配通过；不解除 `blocked-upstream`；不代替人的精确接纳或设计 Gate。
- 定向复审条件：修复 RV-01 后，重新核对 `framework.md:924` 与 `design.md:3892` 的精确字符串、目标哈希/字节/格式、反向重建结果和是否新增未经验证断言；若 RV-02 一并修改，再核对候选表与作者报告同步。

## 6. 待决定与限制

- **待决定 D-RV-01：** 选择修改设计原文（推荐）还是修改 ADR 引文；由 Lead/Arch-Designer 决定并落改动，Reviewer 不代改。
- **待决定 D-RV-02：** 是否接受当前候选对比措辞，或并入 D03/D04/下一次 ADR 修订；不阻断 D 结论。
- **人的接纳：** unknown；`design.md` 仍为 draft，ADR-0030 的“已接受”仅指用户 D02 决策授权，不是成果签收。
- **未验证：** 真实独立故障域、产品/凭据/ACL/部署、Case18、S8.2、E2E、全文件语义与 markdown 渲染均未验证。
