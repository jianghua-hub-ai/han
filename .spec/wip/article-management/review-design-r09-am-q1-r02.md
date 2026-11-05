# article-management design r09 + test-plan r03 AM-Q1 独立复审报告

- **复审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **对象类型**：design + test-plan 复审；有界消费者/交付同步检查
- **原报告**：`.spec/wip/article-management/review-design-r07-am-q1-r01.md`
- **结论**：**Design Review Gate NOT PASS**
- **Finding 统计**：原 finding CLOSED 2；新增 CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 0

## 1. 固定对象与开始校验

复审工作区为 `D:\AID\VSCode\Codex\Han`，分支/HEAD 为 `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。开始时重新计算的完整 SHA256 均与指定值一致：

| 对象 | 修订/状态 | 开始 SHA256 |
|---|---|---|
| `.add/project/knowledge/design/article-management/boundary.md` | r13 / accepted | `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E` |
| `.add/project/knowledge/design-workspace/article-management/design.md` | r09 / draft | `70EE8FB04196FA78EB925133EC8C796E1B34F5B2060273BCE2A704DE3F797B91` |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | r03 / planned-not-executed | `C89DCA5BA9D6089BCDD4E5292B6F2DCB0E028D4CBDD58DFA11D03409300296CF` |
| `.add/project/knowledge/tasks/tasks.md` | SSOT | `CD94B016BF3904702B68B6EECF049500B0EA83452ACFE8CA068A06029F2BE0D4` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | delivery plan | `DCCD3020A2F9C00AD9C3CC520C5739A5BF579C4D2C3AE4B2BECD5AC79401017F` |
| `.spec/wip/article-management/review-design-r07-am-q1-r01.md` | 原报告 | `16C3DB39185ECACB66518D2315DB541B1438A8914FE6CD46331FB9E8BD4AC992` |

本轮唯一写入文件为本报告；未修改、回滚、覆盖、格式化或清理被审对象、代码、tasks、Plan 或他人文件。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审与固定对象直接检查

- 对照原报告逐项复审 `AM-Q1-R01-F001`、`AM-Q1-R01-F002`。
- 检查 boundary r13、design r09、test-plan r03 对以下合同是否一致且可执行：
  - title≤200 rune、body≤20000 rune；
  - 全 Unicode 空白判空、不自动 trim、合法原值保留；
  - Markdown/HTML 仅为纯文本字面量，服务端不解析/渲染，消费者按文本节点或输出上下文编码；
  - 创建和修改的 422、400、零 repository 调用、零持久化与完整修改快照；
  - 262144/262145 actual wire bytes、非法 UTF-8 在 JSON 解码或 replacement 前拒绝、不得保存 U+FFFD；
  - rune、wire byte、JSON 解码与 PostgreSQL `text` 契约分层；
  - planned-not-executed 与未实现状态是否被如实保留。

### Round 2：有界消费者、现役实现与交付同步回归

- 反查 architecture、ADR-0004、Article boundary，以及现役 Article model/policy/handler/repository、migration 和共享 JSON decoder。
- 检查是否新增授权、sanitizer、Markdown/HTML renderer、第三方依赖、公共 API/schema 或其他范围扩张。
- 对账 tasks 的 `article-management-S1.3`、`article-management-S6.1` 与 Plan 的 B5 article lane、共享文件交接及 AM-Q1 状态。
- 保持检查边界为 `S1.3 → S6.1`；未扩成全库审查。

## 3. 原 findings 逐项结论

### AM-Q1-R01-F001 — CLOSED

F001 所述权威状态分裂已关闭：

- boundary `:28,146,252,267,439,456` 已固定纯文本、全 Unicode 空白判空但不 trim、title≤200 rune、body≤20000 rune 与合法原值保留，并明确 validator/wire byte 属于详细设计；
- tasks `:30,848-864` 已解除 S1.3 的 AM-Q1 决策阻塞，并把上述内容合同、262144/262145、非法 UTF-8、零 repository/零持久化写入 S1.3 关闭条件；
- Plan `:43-44,522-523,602-603,655,727-728` 已同步 r13/r09/r03，保持 `S1.3 → S6.1` 串行交接，并把 AM-Q1 实施责任指向 S1.3；
- design `:870,2075` 明确 wire decoder、共享 validator 和创建纵切由 S1.3 承担，S6.1 不接管 AM-Q1。

原触发条件——实施者必须在“已确认数值”和“TBD/阻塞”之间自行选择——已不存在。

### AM-Q1-R01-F002 — CLOSED

F002 所述 wire byte 与非法 UTF-8 测试缺口已关闭：

- design `:279-287,867-870,1168,1292,1984-1999` 固定 actual body payload byte 的计数对象、N=`262144`/N+1=`262145`、先有界读取再校验原始 UTF-8、统一 `400 request.invalid_json`、零 repository/零持久化及不得保存 U+FFFD；
- test-plan `:99-101` 的 A-CREATE-04～06覆盖创建 N/N+1/非法 UTF-8；
- test-plan `:122-124` 的 A-MOD-06～08覆盖修改 N/N+1/非法 UTF-8、repository spy=0、完整 Article 快照不变和不得保存 U+FFFD；
- test-plan `:251,254,272,290,334-345` 将这些场景映射到 REST/decoder、PostgreSQL 数据符合性与 planned-not-executed Gate，且未用性能测试替代协议断言。

创建与修改路径均已有可执行规划；本结论仅确认规划完整，不表示测试已运行。

## 4. 新 finding

### [MEDIUM] AM-Q1-R02-N001：test-plan 的局部测试任务行仍把 AM-Q1 decoder 关闭条件与 S6.1 混列

- **位置**：`.add/project/knowledge/design-workspace/article-management/test-plan.md:302`
- **现状**：同一“文章策略、wire decoder、状态机、cursor、ETag 局部测试”行把 `validator、262144/262145、非法 UTF-8 前置拒绝、零 repository` 列为关闭条件，同时把 `article-management-S6.1` 列入“实际 task 引用”。该行没有逐项说明哪些条件属于哪个 task。
- **冲突依据**：
  - design `:870,2075` 明确 S1.3 拥有 AM-Q1/wire decoder，S6.1 不接管；
  - tasks `:848-864` 将 AM-Q1 精确关闭条件交给 S1.3，而 `:1088-1104` 将 S6.1 限定为 canonical 无归属化端口及 A-ORP-01～02；
  - test-plan 自身 `:251,258` 也分别把创建场景映射到 S1.3、把 A-ORP-01～02 映射到 S6.1；
  - Plan `:602-603,655` 保持 `S1.3 → S6.1`，AM-Q1 只作为 S1.3 的实施/验证合同。
- **触发场景**：S6.1 实施者或 Reviewer 以 §9 的“实际 task 引用/关闭条件”安排局部测试或判断任务关闭，将 decoder/validator 回归并入 S6.1，或认为 S6.1 必须补齐 S1.3 的 AM-Q1 证据。
- **影响**：虽不改变 tasks SSOT 的明确职责，但使固定 test-plan 不是无歧义的实施 Gate，可能扩大 S6.1、重复或错置测试所有权，并破坏已经固定的共享文件串行交接。
- **严重度依据**：其他权威制品足以避免直接业务契约错误，故不定为 HIGH；但现实执行中存在范围和验收错置风险，且直接违反本次“不得把 AM-Q1 业务职责扩给 S6.1”的 Gate 条件，定为 MEDIUM，并在修订或责任人显式接受前阻断本次精确范围 PASS。
- **最小修复方向**：由 test-plan 作者拆分该行，或从包含 AM-Q1 decoder 关闭条件的行移除 S6.1；S6.1 只引用 canonical/A-ORP-01～02 的局部测试。Reviewer 不代为修改被审对象。

## 5. 其他重点结论

- title/body 的 rune 上限、Unicode 空白、不 trim 与原值保留合同明确可实现；创建和修改均有 422 与零副作用规划。
- 服务端纯文本边界和消费者转义责任一致；未把“服务端不解析”误写成允许消费者直接 `innerHTML`。
- wire byte 上限、UTF-8 validity、JSON 解码、业务 rune 校验和数据库 `text` 分层明确；数据库约束不是 Unicode/rune 业务 validator 的替代物。
- 未发现新增授权依赖、sanitizer、Markdown/HTML renderer、第三方依赖、schema 或 AM-Q1 外业务范围。
- 现役共享 decoder/Article 实现尚未完成 r09 合同，不作为本次 design/test-plan planned-not-executed 复审失败；后续实现必须独立验证。
- 除 N001 外，未发现对其他既有合同的新增误改。

## 6. Gate 结论

### Design Review Gate：NOT PASS

精确范围仅为固定 boundary r13、design r09、test-plan r03、tasks/Plan 同步及 Article AM-Q1 增量：

- `AM-Q1-R01-F001`：**CLOSED**
- `AM-Q1-R01-F002`：**CLOSED**
- `AM-Q1-R02-N001`：**STILL OPEN（新增 MEDIUM，本 Gate 阻断）**

r09/r03 已完整关闭原两项缺口，但 test-plan `:302` 仍使 S1.3 与 S6.1 的 AM-Q1 测试/关闭责任存在可触发歧义，因此当前修订集尚未形成无歧义的 Article AM-Q1 实施 Gate。

本结论不外推为实现完成、测试执行、任务关闭、部署、真实环境验证、E2E 或 production readiness。

## 7. 限制、未验证项与复审条件

- 按授权未运行 Go test、race、vet、build、PostgreSQL、部署、浏览器或 E2E；未写 checkpoint。
- 未验证 r09/r03 已被代码实现；现役代码仅用于核对真实消费者和待实施差距。
- 未进行无界全库审查；architecture/ADR-0004、共享 decoder、Article model/policy/handler/repository/migration、S1.3/S6.1 与 Plan lane 之外只在必要引用范围内读取。
- 复审条件：固定修订后的 test-plan 与完整 SHA256，消除 `:302` 对 S6.1 的 AM-Q1/decoder 职责歧义，并重新核对 `S1.3 → S6.1`、全部固定对象哈希及 scoped `git diff --check`。

## 8. 结束校验

全部固定对象结束 SHA256 与 §1 开始值一致：

| 对象 | 结束 SHA256 |
|---|---|
| boundary r13 | `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E` |
| design r09 | `70EE8FB04196FA78EB925133EC8C796E1B34F5B2060273BCE2A704DE3F797B91` |
| test-plan r03 | `C89DCA5BA9D6089BCDD4E5292B6F2DCB0E028D4CBDD58DFA11D03409300296CF` |
| tasks | `CD94B016BF3904702B68B6EECF049500B0EA83452ACFE8CA068A06029F2BE0D4` |
| Plan | `DCCD3020A2F9C00AD9C3CC520C5739A5BF579C4D2C3AE4B2BECD5AC79401017F` |
| 原报告 | `16C3DB39185ECACB66518D2315DB541B1438A8914FE6CD46331FB9E8BD4AC992` |

- fixed documents 与本报告的 scoped `git diff --check`：PASS；仅出现 Git 的 LF→CRLF working-copy 提示，无 whitespace error。
- 本报告完整文件 SHA256：由交付回复提供。SHA256 值无法作为自身内容内嵌而同时保持该完整文件哈希不变。
- 实现测试/部署/E2E：未运行。
