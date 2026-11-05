# B13J D02 — 生产 CAS JournalHeadAnchor provider 选型与设计修订（证据报告）

- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 执行角色：`add-arch-designer`（Arch-Designer，Agent）
- 工作性质：既有 `development-delivery` Workflow 的普通续接；未重新 Route，未新建批次，未改任何 Task 状态。
- 记录时间：2026-11-01（Asia/Shanghai）

## 1. 结论

**首期不提供生产 CAS JournalHeadAnchor provider。** S4.5 维持未交付；生产 recovery `enter`/`install` 继续保持 closed；仅保留 ADR-0019 已定的 provider-neutral `JournalHeadAnchor` port、调用边界与确定性测试替身。`HAN_RECOVERY_ANCHOR_ADAPTER` 与 `HAN_RECOVERY_ANCHOR_CONFIG_PATH` 是唯一配置入口，生产必须显式同时提供，二者无代码默认值、互不推断；未配置即不启动 anchor 相关能力并保持 closed。

该结论是架构决策，不是生产装配批准、真实 provider 验证、独立评审或人的成果签收。未登记 `S4.22`/`S4.23`/`S4.24` 等候选编号，也未声称 Case18 或 S8.2 PASS。

## 2. 决策论证摘要

### 2.1 现状

- ADR-0019 已锁定：`JournalHeadAnchor` 独立故障域、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒、CAS 协议与失败分类；生产 provider 的产品选择、配置键和启动装配明确留给 S4.5，不属 S4.2。
- S4.2 只交付 provider-neutral port、调用边界和确定性测试替身；测试替身不得注册为生产 provider，也不得作为真实独立故障域证据。
- `identity-auth/design.md:3892` 在本轮前明确写实际产品/凭据/独立故障域 unknown；生产 root 以空 registry 构造，anchor 相关端口未注册即 closed。
- 当前没有可核的产品实例、凭据、部署拓扑、ACL、与 PostgreSQL 的故障域隔离证明，也没有真实 CAS 验证证据。

### 2.2 痛点

- 现在直接挑一个产品，会把未验证的外部依赖、凭据与运维责任写进公共合同；后续实现会被迫追随一个未经批准的产品。
- 继续维持“可注入但未定义 adapter”的模糊状态，实施者容易用 file、SQL 或 fake 自动 fallback 填补空缺，而 ADR-0019 已明确禁止这三类路径。
- 若把测试替身或空 registry 当成“已接线”，会把生产 closed 误写成生产可用。

### 2.3 候选对比（四类）

| 方案 | 实现机制 | 优势 | 劣势与未满足条件 | 结论 |
|---|---|---|---|---|
| A. 独立小数据库（专用实例，或独立 schema + 条件写） | 专用数据库实例承载 `journal_id/namespace/epoch/head_sequence/head_hash/revision`，以条件更新或事务实现 CAS；或仅在独立 schema 内建表 | 关系型 CAS 语义成熟；专用实例可与业务 PostgreSQL 分域；运维工具链常见 | 独立 schema 若与业务库同实例，仍随同一备份/故障域回退，不满足 ADR-0019 的独立故障域；专用实例还需产品/版本、凭据、网络、备份恢复、ACL、容量与运维责任，当前均无可核证据，也未做真实故障域验证 | 不选 |
| B. 外部配置/秘密系统（Vault / KMS 类） | 用 Vault KV 类带版本/CAS 的秘密存储保存头状态；KMS 类服务仅可能承担密钥或封装，不是通用 CAS 头存储 | 隔离、ACL、审计与轮换能力强；可复用组织级秘密基础设施 | 需要选定具体产品、认证方式、网络与高可用/解封策略；KV 的版本 CAS 与 pending/committed、namespace、epoch、revision 语义需额外映射；KMS 类服务通常只提供密钥管理/加解密与有限的条件性密钥操作，不提供通用 CAS 头存储；当前无产品、凭据、部署与验证证据 | 不选 |
| C. 对象存储条件写（S3 conditional PUT / ETag CAS） | 以对象条件写或 `If-Match`/ETag 实现 expected revision 的 compare-and-swap，保存 committed/pending 头 | 对象存储可与业务 PostgreSQL 分组部署，但独立故障域须逐案证明；条件写可检测并发冲突；容量与持久性由平台承担 | 依赖具体云产品、区域、凭据与 IAM；需证明条件写、读后一致性、版本/回滚、对象锁与 pending/committed 两阶段语义；网络延迟、错误分类与运维责任未核；当前无真实环境验证 | 不选 |
| D. 首期不提供生产 anchor（选定） | 不注册任何生产 provider；仅保留 port 与确定性测试替身；生产 `enter`/`install` 保持 closed；配置键必须显式且无默认 | 不引入未验证依赖；不虚构产品、凭据或故障域；与 ADR-0019 的 fail-closed 和“测试替身不得作为生产证据”完全一致；成本与审计面最小 | 首期生产恢复链不可用；S4.5 维持未交付；后续必须重新选型、验证并修订 ADR 才能解除 closed | 选定 |

### 2.4 选择 D 的理由

A、B、C 都需要当前不存在的真实产品、凭据、部署/ACL 与独立故障域验证。在缺少这些证据时选择任一产品，等于把“未验证”写成“已决定可部署”，并把后续实施锁进未经批准的公共依赖。D 不新增任何外部依赖，不触发 file/SQL/fake fallback，且明确保留后续通过新决策解除 closed 的路径，是当前唯一可审计且不伪造生产就绪的结论。

### 2.5 解除条件（已写入 ADR-0030）

1. 选定具体产品与部署形态，并能证明其与业务 PostgreSQL 处于独立故障域。
2. 凭据来源、最小权限/ACL、部署拓扑、备份/恢复/轮换与运维责任经批准且可核。
3. 配置键与启动装配合同明确，仍无隐式默认或自动 fallback。
4. 在真实独立故障域完成 CAS 冲突、stale revision、missing/rollback、pending/committed、崩溃恢复与权限失败验证，并保留原始证据。
5. 通过架构与安全独立评审，完成 ADR 修订/新增决策，并由正式 Task 规划登记实现与验证范围。

## 3. 改动清单

### 3.1 `.add/project/knowledge/design-workspace/identity-auth/design.md`

共 6 行原位修改，行数保持 4924。

#### 第 38 行（设计版本）

- 改前：

```text
**设计版本**：`identity-auth-design-ir01-r57`
```

- 改后：

```text
**设计版本**：`identity-auth-design-ir01-r58`
```

#### 第 5 行

- 改前：`revision: identity-auth-design-ir01-r57`
- 改后：`revision: identity-auth-design-ir01-r58`

#### 第 7 行

- 改前：`last_updated: 2026-10-30`
- 改后：`last_updated: 2026-11-01`

#### 第 83 行（同一行末尾追加，未新增行）

- 改前：`**r57 / B13G S4.15 独立评审返工**：……不预签 PASS、关闭、checkpoint 或源码窗口。`（原行 744 字符）
- 改后：原行 + 一个空格 + 以下段落（整行 1184 字符）：

```text
**r58 / B13J-D02 生产 CAS Anchor provider 选型边界（ADR-0030）**：输入 r57 SHA256 `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`。依据 accepted ADR-0030，首期不提供生产 CAS Anchor provider，S4.5 维持未交付、生产 enter/install 保持 closed；仅保留 provider-neutral `JournalHeadAnchor` port 与确定性测试替身，`HAN_RECOVERY_ANCHOR_ADAPTER` 生产必须显式指定且无隐式默认，未配置即不启动 anchor 能力并保持 closed；同步开发视图与配置表。实际产出者 Arch-Designer（Agent，非人的签字）；仍 draft、人的精确接纳 unknown，不声明实现、独立故障域验证或生产可用。
```

#### 第 3577 行（开发视图）

- 改前：

```text
| 开发 | recovery `runtime.go/admission.go/installation.go` 定义最小中性端口，`journal.go/coordinator.go` 扩 ctx/来源绑定；identity `recovery_run_store.go/recovery_execution.go` 实现；article 文件落点见其 §3.4.6/7；platform `recovery_guard.go` 与 cmd `recovery_runtime.go/recovery_restore.go` 适配。均为将来新实施责任，不借旧 task 写权 |
```

- 改后：

```text
| 开发 | recovery `runtime.go/admission.go/installation.go` 定义最小中性端口，`journal.go/coordinator.go` 扩 ctx/来源绑定；identity `recovery_run_store.go/recovery_execution.go` 实现；article 文件落点见其 §3.4.6/7；platform `recovery_guard.go` 与 cmd `recovery_runtime.go/recovery_restore.go` 适配。生产 CAS Anchor provider 首期不提供（ADR-0030），仅保留 provider-neutral port 与确定性测试替身，root 不得构造或注册隐式 adapter，生产 enter/install 继续 closed。均为将来新实施责任，不借旧 task 写权 |
```

#### 第 3892 行（配置表）

- 改前：

```text
| `HAN_RECOVERY_ANCHOR_ADAPTER,HAN_RECOVERY_ANCHOR_CONFIG_PATH` | 显式已登记 production adapter ID、受保护绝对配置路径；当前实际产品/凭据/独立故障域 unknown，未登记、不可达或验证不通过即 closed；无自动 fake/file/SQL fallback |
```

- 改后：

```text
| `HAN_RECOVERY_ANCHOR_ADAPTER,HAN_RECOVERY_ANCHOR_CONFIG_PATH` | 生产必须显式指定已登记 adapter ID 与受保护绝对配置路径，二者无代码默认值、互不推断；未配置即不启动 anchor 相关能力、生产 enter/install 保持 closed。首期不提供生产 adapter（ADR-0030），当前无已登记产品/凭据/独立故障域，任何未登记、不可达或验证不通过均 closed；无自动 fake/file/SQL fallback，root 不得按环境或其他配置推导 adapter |
```

### 3.2 `.add/project/knowledge/decisions/framework.md`

原始 D02 首轮仅在文件末尾追加 ADR-0030；review-r01 返工另在 ADR-0030 内改 `:924`、`:937`、`:938` 三行（见 §7.3），既有 126590 字节前缀仍逐字节未变。追加从第 912 行空行开始：

- 913：`---`
- 914：空行
- 915：`<a id=adr-0030></a>`
- 916：`## ADR-0030: 首期不提供生产 CAS JournalHeadAnchor provider (2026-11-01)`
- 917 起：空行与正文（目的、状态/确认来源、继承关系、问题、决策、现状/痛点/改造/效果、候选对比表、约束、引用记录）。
- 第 912、914 行的空行分隔与 anchor/标题顺序在 Lead 复核后的同范围续作中补齐，与 ADR-0029 的「空行 / --- / 空行 / <a id=adr-XXXX></a> / 标题」风格一致；本节行号均为最终文件行号。
- 文件新增 55 行（911 → 966），全部为 CRLF。

ADR-0030 未写入 `S4.22`、`S4.23`、`S4.24` 等未登记 Task 编号；未重写 ADR-0019；未新增 file/SQL/fake 自动 fallback；未声称任何产品、云服务、数据库或 Vault/KMS 已验证。

### 3.3 `.add/project/knowledge/decisions/README.md`

3 处同步：

1. 全局决策索引新增第 38 行：

```text
| ADR-0030 | [首期不提供生产 CAS JournalHeadAnchor provider](framework.md#adr-0030) | 已接受（2026-11-01 D02 架构决策授权；首期不提供生产 provider，S4.5 维持未交付，生产 enter/install 保持 closed；配置键必须显式且无隐式默认；无真实 provider 验证） | 架构设计 | 框架 | 生产 CAS Anchor provider 产品形态、`HAN_RECOVERY_ANCHOR_ADAPTER`/`HAN_RECOVERY_ANCHOR_CONFIG_PATH`、启动装配、S4.5 未交付与生产 closed | 1 |
```

2. 领域文件表第 44 行，`framework.md` 行在原 `ADR-0029` 后追加 `、ADR-0030`（原地修改，未新增行）。

3. 场景查询表新增第 69 行：

```text
| 选择或调整生产 CAS Anchor provider 产品形态、独立故障域、`HAN_RECOVERY_ANCHOR_ADAPTER`/`HAN_RECOVERY_ANCHOR_CONFIG_PATH`、启动装配、首期不提供或解除条件 | [framework.md（ADR-0030）](framework.md#adr-0030)、framework.md（ADR-0019） |
```

## 4. 格式与哈希实测

### 4.1 改前

| 文件 | SHA256 | 字节 | 行数 | crlf | bareLF | BOM | 末尾 |
|---|---|---:|---:|---:|---:|---|---|
| `identity-auth/design.md` | `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C` | 842966 | 4924 | 169 | 4755 | false | 单 LF |
| `decisions/framework.md` | `5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A` | 126590 | 911 | 911 | 0 | false | 单 CRLF |
| `decisions/README.md` | `35A5623E7E3CFCC140DA0DE9C8B218210C0E34A850A26B5CDD450B81DA40CC63` | 18948 | 104 | 2 | 102 | false | 单 LF |

### 4.2 改后

| 文件 | SHA256 | 字节 | 行数 | crlf | bareLF | BOM | 末尾 | 控制字符 |
|---|---|---:|---:|---:|---:|---|---|---:|
| `identity-auth/design.md` | `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC` | 844091 | 4924 | 169 | 4755 | false | 单 LF | 1（既有 TAB） |
| `decisions/framework.md` | `33281CB5CF6807A9B8B57825CD8A7D9A2AFBED52CC899D9F95D64196DD1CF532` | 137491 | 966 | 966 | 0 | false | 单 CRLF | 0 |
| `decisions/README.md` | `5075879AC7F3E31D83A06E843BCCE80CAA8DBA27F588E39D700E7F6C3765C10D` | 19724 | 106 | 2 | 104 | false | 单 LF | 0 |

说明：

- `design.md` 唯一 `<32` 且非 CR/LF 的字符是字符偏移 308573（对应字节偏移 466069）处的 TAB（码点 9），位于既有 Go fenced code block 的缩进中，不在本轮改动行；反向还原到改前文件后该 TAB 仍存在，证明不是本轮引入。严格按“控制字符必须为 0”的字面口径，本文件不满足；本轮引入的控制字符为 0。
- `README.md` 改前 104 行、改后 106 行，是因为全局索引与场景查询表各新增 1 行；两类新增行均用 LF，既有 2 个 CRLF 行未变，所以 crlf 仍为 2、bareLF 从 102 变为 104。领域文件表为原地修改，未新增行。
- `framework.md` 首轮为纯追加，crlf 从 911 变为 966、bareLF 保持 0；review-r01 返工只改 ADR-0030 内 `:924`、`:937`、`:938` 三行，行数仍 966、仍为纯 CRLF，`:924` 起点之前与 `:938` 终点之后的字节逐字节未变。
- 任务输入中的 framework.md 911 行、README.md 104 行是改前基线；本轮按要求追加 ADR 与新增索引行，行数分别变为 966/106，既有行尾与既有 2 个 CRLF 行未被改写。

## 5. 反向还原与一致性验证

### 5.1 `design.md`

按精确字符串反向替换 6 处改动（去掉第 83 行追加段落，恢复第 5、7、38、3577、3892 行原文）后写入临时文件，实测：

- SHA256：`504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C`
- 字节：842966
- 结果：`REVERSE-OK`，与改前基线完全一致。

逐行比较改前/改后，差异索引恰为 5、7、38、83、3577、3892 六行，其他行逐字符一致。

### 5.2 `framework.md`

- 改后文件前 126590 字节单独取哈希：`5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A`，与改前文件哈希一致。
- 结论：既有内容逐字节未变；原始 D02 首轮只追加 ADR-0030，review-r01 返工另在 ADR-0030 内改 `:924`、`:937`、`:938` 三行（见 §7.3）。

### 5.3 `README.md`

用改前备份按同样的 3 处改动重建期望文本，与改后文件逐字符比较：`expected_matches=True`。差异仅为新增全局索引行、framework 行的 `、ADR-0030` 后缀、新增场景行。

### 5.4 白空格与格式

- `git diff --check -- <三个文件>`：exit 0；仅有 Git 的 autocrlf advisory warning，无 whitespace error。
- 未执行 Git add/commit/push/checkout/switch/stash/restore/clean。
- 未删除、移动或重命名任何既有文件。

## 6. 遗留风险与本轮未覆盖项

1. `identity-auth/design.md:38` 已在 Lead 复核后的同范围续作中同步为 `**设计版本**：identity-auth-design-ir01-r58`；全文 r57 版本号残留已清零（Lead 复核确认）。
2. ADR-0030 的状态“已接受”仅表示用户 2026-11-01 D02 指令授权 Arch-Designer 落定该架构决策；不构成人的成果接收、独立评审、生产装配批准或真实 provider 验证。
3. 未运行任何真实独立故障域验证；未证明任何产品、云服务、数据库实例、Vault/KMS 或对象存储满足 CAS、pending/committed、崩溃恢复或权限失败语义。
4. 未修改 `tasks.md`、`plans/`、`stories.md`、`test-plan.md`、`decisions/identity.md`；未重开 S4.13/S4.16/S4.18/S4.19/S4.20/S4.21/S8.2；未登记任何候选 Task 编号；未改 Plan 头部计数。
5. S4.5 仍未交付，生产 recovery `enter`/`install` 仍 closed；Case18 与 S8.2 未 PASS，恢复 E2E 未执行。
6. `review-r01`（`add-reviewer`，非作者独立评审）判定 CHANGES REQUIRED（RV-01 MEDIUM、RV-02 LOW），处置见 §7；该评审不是人的精确接纳，也不构成本轮成果签收、Gate 通过或 Case18/S8.2 结论。
7. 若未来选择具体产品，须按 ADR-0030 第 7 条解除条件重新取得产品/凭据/ACL/部署证据，并完成 ADR 修订与正式 Task 规划；不得直接以实现代码绕过本轮 closed。
8. 本轮只核对了指定 ADR-0019 区段、design 指定行与索引相关区段，未做全文件语义一致性审计。

## 7. 独立评审返工处置（review-r01）

评审对象与结论：`design-d02/review-r01/report.md`（`add-reviewer`，非作者独立评审）判定 CHANGES REQUIRED，含 RV-01（MEDIUM，必须本轮修）与 RV-02（LOW）。本节是作者自述处置，不代表评审关闭、人的签收或 Gate 通过。

### 7.1 B13J-D02-RV-01（MEDIUM）：自指追溯

- 问题：ADR-0030「问题」段用 `identity-auth/design.md:3892` 证明“改动前的未知状态”，而该行正是本轮被改写的行（改后原文为“当前无已登记产品/凭据/独立故障域”），等于把改后文本当作改前证据。
- 处置：只改 `framework.md:924` 一行，把该半句改为同时标注时点与改后原文。
  - 改前半句：`identity-auth/design.md:3892` 记实际产品/凭据/独立故障域 unknown；生产 root 以空 registry 构造
  - 改后半句：改动前 `identity-auth/design.md:3892` 记实际产品/凭据/独立故障域 unknown（本 ADR 同轮把 `design.md:3892` 改写为“当前无已登记产品/凭据/独立故障域”，不把改写后的原文当作改前状态证据）；生产 root 以空 registry 构造
- 未采用评审“方案 1”（改 `design.md:3892` 原文）：那需要改动 `design.md`，超出本轮写集，也会让设计口径再次偏离“生产必须显式指定、未配置即 closed”。
- 验收读法：读者能区分“改动前状态”与“改动后原文”；ADR 引用不再把改后文本充当改前证据。

### 7.2 B13J-D02-RV-02（LOW）：两处过强表述

- `framework.md:937`（候选 B 劣势列）：`KMS 本身不提供通用条件写` → `KMS 类服务通常只提供密钥管理/加解密与有限的条件性密钥操作，不提供通用 CAS 头存储`
- `framework.md:938`（候选 C 优势列）：`对象存储天然独立于 PostgreSQL` → `对象存储可与业务 PostgreSQL 分组部署，但独立故障域须逐案证明`
- `§2.3` 候选表已同步上述两处措辞；「结论」列的 `不选` 与 A/B/C/D 的取舍未变，ADR `:952` 的“不证明任何产品/环境验证”声明保留。

### 7.3 返工后实测与“只有三行变化”证明

- 返工前 `framework.md`：`31960C7A7BAE3EA3B2B250A6ED9DC1FB1E1F81464520E03A0D42C5E0B491C9BA` / 137200 B / 966 行。
- 返工后 `framework.md`：`33281CB5CF6807A9B8B57825CD8A7D9A2AFBED52CC899D9F95D64196DD1CF532` / 137491 B / 966 行 / crlf=966 / bareLF=0 / BOM=false / 末尾单 CRLF / 控制字符 0。
- 逐行比对返工前后（先留改前副本，再按 LF 切分逐元素比较）：差异索引恰为 `924`、`937`、`938` 三行，其余 963 行逐字符一致。
- 字节级比对：`:924` 起点之前 127765 字节逐字节一致；`:938` 终点之后 5489 字节逐字节一致。
- 字节增量 291 = `:924` +164、`:937` +81、`:938` +46；行数不变，未增删换行，未重排表格。
- `design.md` 与 `README.md` 本轮零变化，实测仍为 `30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC` / 844091 B 与 `5075879AC7F3E31D83A06E843BCCE80CAA8DBA27F588E39D700E7F6C3765C10D` / 19724 B；两文件本轮未写入。

### 7.4 未采纳的评审基线说法

review-r01 提出任务书中 S4.5 登记行号应作基线更正。Lead 现场实测该更正不成立：任务书中 S4.5 的真实登记行为 `:656`，原文为 `- [ ] identity-auth-S4.5 选择并交付生产 CAS JournalHeadAnchor provider 与独立故障域验证 [P]`。本轮不按该说法改动任何产物，也未改 `tasks.md`。

### 7.5 本轮边界

- 未做 Git 写；未删除/移动文件；未改 `tasks.md`、`plans/`、`stories.md`、`test-plan.md`、`decisions/identity.md`；未重开既有结论；未声称 Case18 或 S8.2 PASS。
- 定向复审尚未发生：本节是作者处置记录，不是评审 PASS，也不解除 `blocked-upstream`。