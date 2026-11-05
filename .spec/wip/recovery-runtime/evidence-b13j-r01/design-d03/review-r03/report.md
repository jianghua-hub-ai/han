# B13J D03 定向复审 r03（B13J-D03-RV2-01～RV2-03 返工复核）

- 评审角色：add-reviewer（非作者独立评审，含安全视角）
- 对象类型：design + adr + 索引（复合对象），定向复审
- 评审时间：2026-11-01T23:34:06+08:00
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 被评对象：只读；本轮未修改任何被评文件，未执行任何 Git 写操作

## 1. 范围与基线

本轮只复核 `B13J-D03-RV2-01`（MEDIUM）、`RV2-02`（LOW）、`RV2-03`（LOW）及其直接受影响的证据项（返工最小性、格式不变量、诚实性、继承边界、gap-map §2.5 验收命令）；不重跑 review-r01 的 14 条全量，也不重新论证 ADR-0031 的 A/B/C/D 方案取舍。

| 文件 | 基线 SHA256 / 字节 / 行 | 当前 SHA256 / 字节 / 行 | crlf:bareLF |
|---|---|---|---|
| `identity-auth/design.md` | `B452E38E7074B1D483CB8A0EB511A7EF858B3F35C13CC0841A75118E4B7F013C` / 846941 / 4924 | `B07BAFC7CE1265A42B4F9A79C0F1489D3FCAAB50F47F1C02F0A1FCDA3C23AF99` / 847437 / 4924 | 169:4755 |
| `decisions/framework.md` | `3A3B2B214D9732BB61A0C8768CEF701B08F65ED3D8161C85F254A0F4B37275B8` / 154074 / 1049 | `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12` / 155139 / 1049 | 1049:0 |
| `decisions/README.md` | `A67B00D5BF050E100EA5AEC9DEBBFB8E5DABCE4463448658D68A5B4792E0F706` / 20764 / 108 | 同左（零漂移） | 2:106 |

改前快照（作者本轮新增，供独立比对，实测等于上一轮基线）：

- `design-d03/snapshots/design-before-r03.md` `B452E38E…F013C` / 846941 B
- `design-d03/snapshots/framework-before-r03.md` `3A3B2B21…275B8` / 154074 B

作者证据（本轮更新）：`design-d03/report.md` `27AC38D34E4A814B97F9056FE203BEB40E7AD96DC1E874F87B960F5C2BA69685` / 108987 B；`design-d03/index.json` `DF0E3ED84CE3D4160DACF6588AD6F999A537202DFFDF8108B698F07F7F89C73C` / 45671 B。

方法：整文件 SHA256 与字节/行/行尾/BOM/控制字符实测；对两份快照做逐行序列比对与逐行字符级 diff（`difflib.SequenceMatcher`，禁用 autojunk）；对 `framework.md` 前 137491 B 做前缀哈希；独立运行 gap-map §2.5 的两条 `rg` 与 `git diff --check`；对 ADR-0031 与 design restore 段做"已验证/已批准/已演练/实测通过"关键词扫描。全部命令只读。

## 2. 逐条结论

### 2.1 RV2-01（`design.md:3862` 与整库语义冲突）→ CLOSED

(a) 写入侧否定已在 `:3862` 重建为两层、且不再以"表"为界：

- 替换窗口之外："该角色权限不得覆盖 Journal/keyring、部署锁、Store 权威绑定与审计记录的任何写入"；
- 替换窗口之内："对目标库的写入只能来自已批准 artifact 的整库还原本身，执行身份不得自行构造、注入或声明任何权威 binding，也不得写 Journal/keyring 或部署锁"，"artifact 内既有历史行不得当作本次权威"；
- 替换完成后："由 root 以独立读取路径与同一 Store 短事务创建/核本次 running 与不可变 binding（同下文 Replace 合同）"。

旧版"该角色权限不得覆盖 identity/recovery 表写入"与"排除相应 schema / 经隔离库后由 root 受控切换"均已删除（逐词 diff 证实：删除 `，该角色权限不得覆盖 identity/recovery 表写入`、删除 `，不得写 Journal/keyring、部署锁、Store 绑定或审计，不得自行建立权威 binding`、删除 `经隔离库后由`，新增"不引入 schema 级排除或 root 数据搬移"）。

(b) 强制机制可核验：`独立数据库角色` + `GRANT 矩阵由批准记录固定、未证明即保持 closed`（`design.md:3862`）；`framework.md:1036` 负向门禁要求"替换窗口外，以执行身份的凭据或其目标 DB 角色写入 Journal/keyring/部署锁/Store 权威绑定/审计必须被拒绝；替换窗口内，除已批准整库 artifact 的还原本身外不得产生任何 Store 权威绑定或 Journal/keyring/部署锁写入，执行身份自行构造、注入或声明的 binding 必须被拒绝"。该门禁不再要求"角色写 Journal/keyring 必须被拒绝"与整库还原同时成立（旧版不可满足点已消除）。

(c) 追溯与实际落点一致：`framework.md:1045` 逐项对应 `:3862` 的真实文字（批准记录含工具/备份供应方不得自批、专用执行身份、root 保留部署锁/Journal/keyring/Store 绑定与审计写入、进程树 + ENV allowlist + fd 隔离预检且 no-follow 指向 `:3870`、POSIX/Windows 矩阵、两层写入边界、受保护对象仅作核对项且无 schema 级排除/root 数据搬移、独立 DB 角色 + GRANT + 未证明即 closed、目标连接外无网络与禁 su/sudo/shell 包装），并正确注明配置表 `:3896` 三键语义不变、` :83` 登记本 ADR；未把 `:3862` 没有的内容归给它。

整库语义下的自洽性（本轮关键判定）：artifact 必然包含 Journal/keyring 行（`framework.md:464` 要求 keyring 与 Journal 整体备份、迁移和恢复；`design.md:545` 账号数据与恢复元数据同属一个 PostgreSQL 故障域；`:1718` `recovery_runs`/`recovery_restore_bindings` 同 Store）。新文本据此把边界从"哪些表不可写"改为"写入来源"（artifact 还原本身 vs 执行身份自行构造），并要求 root 在窗口后以同一 Store 短事务重建本次权威，因此与整库 restore 同时成立，也没有给执行身份留下自建权威的余地。

### 2.2 新边界概念与 `:3864`/`:3866`/`:3870` 的残留冲突 → 无；"核对项"不是新门禁、不是新证明来源 → 可接受

- `:3864` 仍只规定私有 stdin 恰为 `{"plan":…,"target_connection":{"dsn":…}}`，target_connection 只传现役已校验 DSN，不产生新默认/TLS 配置；新句不引入 staging 库、第二目标或 DSN 变更。
- `:3866` 仍要求子程序对"同一已校验 artifact/target"执行 restore，receipt 字段与 `restored_schema_digest` 不变，root 仍须实际 join、重开目标连接、核 source installation 元数据/目标描述符/schema digest 与 `VerifyRestored`，再经同一 Store 短事务创建/核 running＋不可变 binding/审计。新句与该路径逐句相容。
- `:3870` 仍是同一 no-follow 稳定 fd/handle 与原生耐久回执独立读取；"整库语义不变"与之无冲突（未定义第二次打开或第二路径）。
- "受保护对象以批准记录固定的可检清单（Journal、keyring、部署锁、Store 权威绑定与审计记录）作为 restore 完成后的核对项"没有进入 `framework.md:1036` 的解除条件，也没有替代 `:3866` 的 receipt/`VerifyRestored` 证明链；它没有把任何对象移出整库还原范围，也没有授权 root 搬移数据。判定：可接受。

### 2.3 RV2-02（`:1045` 归属错误）→ CLOSED

`framework.md:1045` 现为"进程树/ENV allowlist 与 fd 隔离预检（no-follow 稳定 fd 见 `:3870`）"，与 `design.md:3862`（"fd/进程树隔离预检失败均在 Begin 前拒绝"）和 `:3870`（no-follow 稳定 fd）一致；`design.md:83` 同步由"进程组"改为"进程树"（等长、等字节，行仍 1646 字符 / 2508 B）。全文不再有把 `:3862` 说成"进程组"的追溯表述；`README.md:39/:71` 与 `framework.md:992` 的"进程组"指候选机制/劣势，不是对 `:3862` 的归属，且 README 本轮零改动。

### 2.4 RV2-03（缺改前副本）→ CLOSED

利用两份快照做字节级比对（全量 opcodes，仅出现 replace，无 insert/delete）：

- `design.md` 差异行恰为 `{83, 3862}`：`:83` 字节增量 0（`组`→`树`），`:3862` +496 B，合计 +496 = 847437 − 846941；其余行逐字节相同。
- `framework.md` 差异行恰为 `{1007, 1034, 1036, 1045}`：+336 / +145 / +189 / +395 = +1065 = 155139 − 154074；行数 1049→1049，无增删行；其余行逐字节相同。
- `framework.md` 前 137491 B SHA256 = `33281CB5CF6807A9B8B57825CD8A7D9A2AFBED52CC899D9F95D64196DD1CF532`（= D03 基线整文件），ADR-0019、ADR-0030、ADR-0027 逐字节未变。

### 2.5 格式不变量 → PASS

| 文件 | BOM | 行数 | crlf | bareLF | 末尾 | 控制字符（除 CR/LF） |
|---|---|---|---|---|---|---|
| `design.md` | 无 | 4924 | 169 | 4755 | 单 LF `3e0a` | 1 个 TAB，偏移 466829（行 3303 既有 Go 代码块，与 r02 同偏移） |
| `framework.md` | 无 | 1049 | 1049 | 0 | 单 CRLF `0d0a` | 0 |
| `README.md` | 无 | 108 | 2 | 106 | 单 LF `820a` | 0 |

`design.md` 的 TAB 位于 `:83` 与 `:3862` 之前（`:83` 等长、`:3862` 在其后），本轮未位移、未新增；`framework.md` 纯 CRLF 保持。

### 2.6 验收命令（gap-map §2.5）→ PASS

- `rg -n "restore-v1|verify-v1|批准者|批准主体|执行主体|进程树|no-follow|restore_unknown" .add/project/knowledge/design-workspace/identity-auth/design.md` → exit 0，命中 13 行，含 `:3858`（`RestoreDriver`/`restore_unknown` 合同）、`:3862`（批准主体、执行主体、进程树、no-follow 预检）、`:3864`（私有 stdin）、`:3866`（receipt/`VerifyRestored`）、`:3868`（`restore_unknown`）、`:3870`（no-follow 稳定 fd）、`:3874`（argv）。
- `rg -n "restore|恢复工具|批准|进程隔离|no-follow" .add/project/knowledge/decisions/framework.md` → exit 0，命中 72 行，含 `:971`（ADR-0031 标题）、`:975`（状态）、`:977`（继承关系）、`:981`（决策）、`:1004`–`:1007`（批准/执行/权限边界）、`:1008`–`:1020`（宿主隔离矩阵与原生回执）、`:1031`–`:1040`（解除条件）、`:1045`–`:1049`（引用记录）。
- `git diff --check -- <design.md> <framework.md>` → exit 0（仅 `design.md` 的 `LF will be replaced by CRLF` autocrlf 咨询 warning，不是空白错误）。

### 2.7 无新不实断言 → PASS

- `design.md:3858`–`:3872` 无"已验证/已批准/已演练/实测通过"类断言；唯一"已批准"是 `:3862` 的"已批准 artifact"（指待批准来源，非已验证）。
- `framework.md:968`–`:1049` 的"已批准"全部出现在否定句、条件句或"已批准整库 artifact"（来源批准）语境；"实测通过"只出现在 `:1035` 的解除条件条目；`:1029` 明确"不声称…已通过"；`:1036` 是解除条件条目而非已完成的验证结论。
- 其余：`:1007`/`:1030`/`:1034` 均为条件或否定表述，未出现"已验证 GRANT/隔离/负向写入通过"。

### 2.8 边界与继承（直接受影响项）

- 本轮除 `design.md`、`framework.md` 外无其它文件改动：`README.md` 哈希零漂移；`tasks.md`（21:45）、`plans/ir-01-content-community-batches.md`（20:07）、`decisions/identity.md`（10-28）、`identity-auth/test-plan.md`（10-29）、`identity-auth/stories.md`（10-29）的最后写入时间早于本轮（23:25/23:26），未随本轮变化。
- `design.md:5` = `:38` = `identity-auth-design-ir01-r59`；`:7` 仍为 `2026-11-01` 且未改动；`:83` 追溯输入哈希 = `30B122A0…7D47BC`（r58），并如实标注 Agent 产出、非人的签字、draft、人的精确接纳 unknown、不声明实现/验证。
- `:3874`–`:3902` 零改动（逐行比对证实）；`restore-v1`/`verify-v1` argv、私有 stdin、receipt 字段、no-follow 稳定 fd、最小 ENV、Begin 前预检、`restore_unknown` 语义均未被本轮触碰。

## 3. RV2 最终状态

| Finding | 级别 | 状态 | 依据 |
|---|---|---|---|
| `B13J-D03-RV2-01` | MEDIUM | **CLOSED** | `design.md:3862` 两层写入边界；`framework.md:1007/:1034/:1036` 与之一致且可满足；` :1045` 追溯准确 |
| `B13J-D03-RV2-02` | LOW | **CLOSED** | `:1045` 进程树/no-follow 归属已改正，`:83` 术语同步 |
| `B13J-D03-RV2-03` | LOW | **CLOSED** | 快照齐备，字节级差异集恰为 `{83,3862}` 与 `{1007,1034,1036,1045}` |

## 4. Findings

**无新增 finding。** 以下两条为非阻断观察（不计 finding，不改变判定，建议并入 D04/实现任务）：

- O1（措辞/实现基准）：`design.md:3862` 未写明"受保护对象可检清单"的比对基准（与已校验 artifact 清单一致 vs 与替换前状态一致）以及不匹配时的处置；实现阶段应固定为"与已校验 artifact 的清单一致、不匹配即按 `:3868` 保守处理"，避免被误用为证明来源或 schema 排除依据。
- O2（可验证性设计）：`framework.md:1036` 的"窗口内除还原本身外无额外写入"在缺少表级 digest 定义时，只能靠窗口内写权限独占 + 固定 argv/无任意 SQL 的隔离实测来证明；实现任务须把该证明方法与 GRANT 阶段化（窗口边界授予/撤销）一并登记，当前不构成文档缺陷。

## 5. 评审结论

- **设计评审：PASS**（合同层）
- **安全评审：PASS**（合同层）
- **总判定：PASS**

理由：RV2-01 的冲突已按"来源而非表"重新划界，删除不可满足的 `identity/recovery 表` 与"隔离库 + root 受控切换"，`framework.md:1036` 门禁随之可满足；RV2-02、RV2-03 已实质关闭；返工差异精确限于声明行，格式、继承边界、验收命令与诚实性全部实测通过。两条非阻断观察属实现阶段登记事项。

限定：本 PASS 仅指文档/合同层评审结论，不构成人的接收、设计 Gate 或安全签收，不解除生产 `closed`（ADR-0031 首期不提供仍然有效），不关闭 `pending-B13J-D03`，不解除 `blocked-upstream`，不解锁 S4.24，不声称 Case18/S8.2 通过；GRANT/隔离/原生回执的真实验证仍属实现阶段与安全评审输入。

## 6. 实测附录

### 6.1 哈希与规模

| 对象 | SHA256 | 字节 |
|---|---|---|
| `design.md` | `B07BAFC7CE1265A42B4F9A79C0F1489D3FCAAB50F47F1C02F0A1FCDA3C23AF99` | 847437 |
| `framework.md` | `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12` | 155139 |
| `README.md` | `A67B00D5BF050E100EA5AEC9DEBBFB8E5DABCE4463448658D68A5B4792E0F706` | 20764 |
| `snapshots/design-before-r03.md` | `B452E38E7074B1D483CB8A0EB511A7EF858B3F35C13CC0841A75118E4B7F013C` | 846941 |
| `snapshots/framework-before-r03.md` | `3A3B2B214D9732BB61A0C8768CEF701B08F65ED3D8161C85F254A0F4B37275B8` | 154074 |
| `design-d03/report.md` | `27AC38D34E4A814B97F9056FE203BEB40E7AD96DC1E874F87B960F5C2BA69685` | 108987 |
| `design-d03/index.json` | `DF0E3ED84CE3D4160DACF6588AD6F999A537202DFFDF8108B698F07F7F89C73C` | 45671 |

### 6.2 差异行（对快照）

| 文件 | 差异行（1-based） | 字节增量 |
|---|---|---|
| `design.md` | `83`、`3862` | 0、+496（总 +496） |
| `framework.md` | `1007`、`1034`、`1036`、`1045` | +336、+145、+189、+395（总 +1065） |
| `README.md` | 无 | 0 |

### 6.3 关键行文本（节选）

- `design.md:3862`：两层写入边界 + "受保护对象以批准记录固定的可检清单（Journal、keyring、部署锁、Store 权威绑定与审计记录）作为 restore 完成后的核对项，不改变 restore 的整库语义，也不引入 schema 级排除或 root 数据搬移"。
- `framework.md:1007`：root 保留部署锁/排空/独占/Begin/绑定/审计/`VerifyRestored`；执行身份窗口外不得写，窗口内写入只能来自已批准整库 artifact 还原，本次权威 binding 只由 root 经同一 Store 短事务创建/核（`design :3866`）。
- `framework.md:1036`：窗口外负向写入必须被拒绝；窗口内除还原本身外不得产生额外 Store 权威绑定或 Journal/keyring/部署锁写入，自造 binding 必须被拒绝。
- `framework.md:1045`：追溯句，no-follow 已指向 `:3870`。

## 7. 未验证范围与复审条件

未验证：任何真实 restore 工具、批准记录、专用执行身份、ACL、GRANT 矩阵、进程隔离实测、原生耐久回执与旧备份演练（`framework.md:1036` 门禁从未执行）；`cmd/han-server` 运行时行为；工作区其它既有 `M`/`??` 项与本项无关，不由本报告判断。

复审条件：若 `design.md:3862`、`framework.md:1007/:1034/:1036/:1045` 或 `README.md` 再有改动，或 O1/O2 在实现任务中被采纳改写，需就对应行做定向复核；本节结论不覆盖未来实现证据。
