# B13J-D02 定向复审报告（review-r02）

## 0. 角色、对象与写集

- 执行角色：`add-reviewer`（非作者独立复审）。仅复核 `B13J-D02-RV-01`、`B13J-D02-RV-02` 的返工，不重跑 review-r01 的全量 12 条，也不替作者修改产物。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- objectType：复合对象 `design + adr + index`；本轮只检查 ADR-0030 三条返工行及其直接证据，不套用代码提交、编译、覆盖率或 E2E 条款。
- 固定对象：
  - `.add/project/knowledge/decisions/framework.md`：本轮目标 `33281CB5CF6807A9B8B57825CD8A7D9A2AFBED52CC899D9F95D64196DD1CF532` / 137491 B；上一版 `31960C7A7BAE3EA3B2B250A6ED9DC1FB1E1F81464520E03A0D42C5E0B491C9BA` / 137200 B。
  - `.add/project/knowledge/design-workspace/identity-auth/design.md`：`30B122A08CA127D39FFF4F4331713F83E189630B8D739106DB24CF32B17D47BC` / 844091 B，应与 review-r01 目标完全一致。
  - `.add/project/knowledge/decisions/README.md`：`5075879AC7F3E31D83A06E843BCCE80CAA8DBA27F588E39D700E7F6C3765C10D` / 19724 B，应与 review-r01 目标完全一致。
- 作者证据：`design-d02/report.md` `16A47B381CB5BD023EA931432CE9A3CEA622125CB4CCA9641DE9EDDDC293CE7F` / 20671 B；`design-d02/index.json` `4DF391ACD942F6747EE8B6623677B9A4678A3BB493EB83A46FD00690AC28073B` / 19856 B。
- 写集：仅本报告与同目录 `index.json`。未修改被评对象、`tasks.md`、Plan、Stories、test-plan、代码或测试；未做 Git 写；未删除或移动文件。
- 边界：本复审不代替人的精确接纳、设计 Gate 或后续实施授权；不声称 Case18、S8.2、真实 provider、恢复 E2E 或生产装配通过；不解除 `blocked-upstream`。

## 1. 实际检查范围

### 1.1 Round 1：字节、最小性与零漂移

- 复测三文件 SHA256、字节、行数、CRLF/bareLF、BOM、末尾行尾与控制字符。
- 以本轮三条改动的新文句反向替换回上一版文句，按原 CRLF 重新拼接，独立重建 review-r01 目标 `framework.md`；核对重建哈希、字节、行差异集合与前 126590 B 前缀哈希。
- 核对 `design.md`、`README.md` 是否保持 review-r01 目标哈希；检查当前 Git 状态中是否出现在既有脏工作树之外的新文档路径；比较三文件修改时间作为辅助证据。

### 1.2 Round 2：语义核验

- 读取 `framework.md:918-966` 全文，重点核对 `:924`、`:937`、`:938` 的最终语义、上下文和与 `:941-952` 的一致性。
- 读取作者 `design-d02/report.md` 的 §7 返工处置与 §7.3 自测声明，核对作者是否把返工说成评审 PASS、是否新增产品验证断言。
- 对照 review-r01 的 RV-01 / RV-02 原始要求，检查返工后是否真正消除原问题。

### 1.3 未验证范围

- 未运行 Go、PostgreSQL、HTTP、restore、Case18、S8.2、E2E、集成测试或任何真实 CAS provider 验证。
- 未重新审计 ADR-0030 全文以外的设计语义，也未重跑 review-r01 的其余 10 条。
- 未运行 markdown lint/render；格式结论来自字节级扫描。
- 工作树仍有既有未提交变更，不能把全部未提交差异归因于本轮返工；本报告只对固定对象和固定修订作结论。

## 2. 逐条结论

### 2.1 RV-01 是否真关闭

**结论：CLOSED。** `framework.md:924` 现在使用“改动前 `identity-auth/design.md:3892` 记实际产品/凭据/独立故障域 unknown”陈述改前事实，并在紧跟的括号中明确“本 ADR 同轮把 `design.md:3892` 改写为‘当前无已登记产品/凭据/独立故障域’，不把改写后的原文当作改前状态证据”。这同时保留了改前 unknown 事实、标明同轮改写、明确禁止把改后原文充当改前证据；没有把改后文本误作改前状态，也没有把 unknown 与“无已登记”混为一谈。未发现新的自指或歧义。

### 2.2 RV-02 是否真关闭

**结论：CLOSED。** `framework.md:937` 已把 KMS 判断降为“KMS 类服务通常只提供密钥管理/加解密与有限的条件性密钥操作，不提供通用 CAS 头存储”，并用“通常/有限”限定类别级判断；`:938` 已把对象存储改为“可与业务 PostgreSQL 分组部署，但独立故障域须逐案证明”，不再声称天然独立。A/B/C 仍为“不选”，D 仍为“选定”；`:952` 仍明确不证明任何产品、云服务或环境验证通过。取舍和结论列均未改变。

### 2.3 返工最小性

**结论：PASS。** 将 `:924`、`:937`、`:938` 的当前文句精确替换回作者记录的上一版文句后，内存重建文件为 `137200 B` / SHA256 `31960C7A7BAE3EA3B2B250A6ED9DC1FB1E1F81464520E03A0D42C5E0B491C9BA`，与 review-r01 目标逐字节一致。逐行差异集合恰为 `[924, 937, 938]`；其余 963 行未变。三行长度变化为 `+164`、`+81`、`+46`，合计 `+291 B`，与当前 `137491 B` 相符。`framework.md` 前 126590 B 的 SHA256 仍为 `5711F51016CBC091DCB8DEC4ABE1339158C713CD997980982A14D6D00938A38A`。

### 2.4 格式不变量

**结论：PASS。** `framework.md` 为 966 行、CRLF=966、bareLF=0、BOM=false、末尾单 CRLF（`0d0a`）、除 CR/LF 外控制字符 0。`design.md` 为 4924 行、CRLF=169、bareLF=4755、BOM=false、末尾单 LF、除既有 TAB 外控制字符 0；`README.md` 为 106 行、CRLF=2、bareLF=104、BOM=false、末尾单 LF、控制字符 0。三文件均未出现本轮行尾漂移。

### 2.5 零漂移与边界

**结论：PASS。** `design.md` 与 `README.md` 哈希、字节均与 review-r01 目标完全一致；三文件修改时间显示 `design.md`、`README.md` 早于本轮 `framework.md`。当前 Git 状态的 293 项与既有脏工作树一致，新增未跟踪目录仅为 `design-d02/` 证据目录；限定范围内未出现新的文档路径，`tasks.md`、Plan、Stories、test-plan、`decisions/identity.md` 未被本轮写出。只能确认未发现本轮越界证据，不能把既有脏工作树全部归因 D02。

### 2.6 新增副作用与不实断言

**结论：PASS，带一个非阻断观察项。** `:937` 的“实现机制”列已说 KMS“不是通用 CAS 头存储”，“劣势”列又说“不提供通用 CAS 头存储”，存在措辞重复；两处含义一致，不构成矛盾，也不改变 B 的“不选”结论，属可接受、不阻断的冗余，建议后续合并。三条改动句未出现“已实现”“已验证”“已通过真实故障域”“生产可用”等断言；ADR `:952` 和作者报告 §7.5 仍明确未做真实验证。未发现新的不实断言。

## 3. 原 Finding 最终状态

| Finding | 原严重级 | 最终状态 | 关闭依据 |
|---|---|---|---|
| `B13J-D02-RV-01` | MEDIUM | **CLOSED** | `framework.md:924` 已同时保留改前 unknown、同轮改写说明和“不以改后原文充当改前证据”的限制；语义无自指或歧义。 |
| `B13J-D02-RV-02` | LOW | **CLOSED** | `framework.md:937-938` 已降低 KMS 与对象存储判断强度；A/B/C/D 取舍未变；`:952` 未验证声明仍在。 |

## 4. Findings

**本轮无新的缺陷 finding。** 未发现需要作者再次整改的 CRITICAL、HIGH、MEDIUM 或 LOW finding。

### 非阻断观察项 O-RV2-01

- **位置：** `framework.md:937` 的“实现机制”列与“劣势与未满足条件”列。
- **现状：** 两列都说明 KMS 不是通用 CAS 头存储，存在重复。
- **处置：** 可接受、不阻断；如后续修订 ADR-0030，可合并为一处表述以降低阅读重复。

## 5. 总判定

**PASS。** 一句话理由：RV-01 与 RV-02 均已按定向要求真实关闭，返工差异恰为 `[924, 937, 938]` 三行，格式、前缀和零漂移均通过；仅剩 `:937` 的非阻断措辞冗余。

## 6. 基线更正与限制

- review-r01 曾把 `tasks.md` 的 S4.5 登记行误判为 `:658`；本轮复测确认作者 §7.4 正确：真实登记行是 `tasks.md:656`，原文仍为 `[ ] identity-auth-S4.5 ... [P]`。该更正不改变 review-r01 的 RV-01 / RV-02 结论，也不构成本轮作者产物缺陷。
- 本复审不声称 Case18、S8.2、真实 provider、恢复 E2E 或生产装配通过，不解除 `blocked-upstream`，不代替人的精确接纳或设计 Gate。
- 若 `framework.md` 三条返工行或其他固定对象在复审后再次变化，本 PASS 自动失效，需重新定向复核。
