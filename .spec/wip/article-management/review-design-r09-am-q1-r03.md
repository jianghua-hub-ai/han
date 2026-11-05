# article-management design r09 + test-plan r04 AM-Q1 独立复审报告

- **复审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **对象类型**：design 复审；有界 test-plan、消费者与交付同步检查
- **复审输入**：`.spec/wip/article-management/review-design-r09-am-q1-r02.md`
- **结论**：**Design Review Gate PASS**
- **Finding 统计**：原 finding CLOSED 1；新增 CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0

## 1. 固定对象与开始校验

工作区为 `D:\AID\VSCode\Codex\Han`，分支/HEAD 为 `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。开始时重新计算的完整 SHA256 均与指定值一致：

| 对象 | 修订/状态 | 开始 SHA256 |
|---|---|---|
| `.add/project/knowledge/design/article-management/boundary.md` | r13 / accepted | `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E` |
| `.add/project/knowledge/design/architecture.md` | r24 | `FAC19BE91763855EC4CE91D97B8AC052A824AAC643794E4B0F45A01EE04C586B` |
| `.add/project/knowledge/design-workspace/article-management/design.md` | r09 / draft | `70EE8FB04196FA78EB925133EC8C796E1B34F5B2060273BCE2A704DE3F797B91` |
| `.add/project/knowledge/design-workspace/article-management/test-plan.md` | r04 / planned-not-executed | `5A1FE258792CF281B27DE2B9E51131E1C46E738F8B3589A184ACFF383A9DDF3A` |
| `.add/project/knowledge/tasks/tasks.md` | SSOT | `292DE0E5306A222D04E83D8510E24347B5FBB814AC96C62D2061690D38E20EF3` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | delivery plan | `CD371BA3FC5824EA61676CA3F9F9A7872070B463ADD3BE333267D47E3F01E385` |

复审输入 r02 报告开始 SHA256 为 `C67047AEB319CF05A6EDA8F46487109F6EABA906CE68E0B22F14909587CFABBC`。

本轮唯一写入文件为本报告；未修改、回滚、覆盖、格式化或清理被审对象、代码、tasks、Plan、Git 或他人文件。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

- 完整读取 r02 报告并复核 `AM-Q1-R02-N001` 的触发条件、影响和关闭标准。
- 检查 test-plan r04 的风险、创建/修改场景、追踪矩阵和 §9 任务建议，重点核对 AM-Q1 validator/wire decoder 与 canonical 无归属化端口是否拆分。
- 对照 design r09、tasks 的 `article-management-S1.3` / `article-management-S6.1` 和 Plan 的 B5 article lane，核对失败证据、实现责任、测试责任和共享文件交接顺序。

### Round 2：设计一致性与有界消费者/交付回归

- 复核 boundary r13、architecture r24、design r09 与 test-plan r04 的 AM-Q1 合同：纯文本、Unicode 空白、不 trim、原值保留、rune 上限、wire byte 上限、UTF-8/JSON 顺序、400/422 和零副作用。
- 有界反查现役 Article model/policy/handler/repository、migration 和共享 HTTP/JSON 接缝；检查是否引入授权依赖、sanitizer、Markdown/HTML renderer、第三方依赖、schema 或 AM-Q1 外职责。
- 回归 Tasks/Plan 的真实交付关系，确认保持 `S1.3 → S6.1`，且未把 AM-Q1 业务职责提前或重复交给 S6.1。
- 未扩展为全仓预审。

## 3. 原 finding 结论

### AM-Q1-R02-N001 — CLOSED

原 finding 所述“test-plan 将 AM-Q1 decoder 关闭条件与 S6.1 混列”的歧义已消除：

1. test-plan `:303` 现以独立行将“AM-Q1 validator、wire decoder 与创建纵切局部测试”唯一映射到 `article-management-S1.3`，关闭条件明确包含共享 validator、262144/262145、非法 UTF-8 前置拒绝、零 repository/零持久化和创建错误映射。
2. test-plan `:304` 的其他策略、状态机、cursor、ETag 局部测试不再引用 S6.1，也不包含 AM-Q1 decoder 关闭条件。
3. test-plan `:305` 以独立行将 S6.1 限定为 canonical 无归属化端口和 A-ORP-01～02，并明确其进入条件是 S1.3 完成开发、Review 与 Lead 交接。
4. test-plan `:252,259` 的场景追踪分别把 A-CREATE-01～06 映射到 S1.3/S1.5、把 A-ORP-01～02 映射到 S6.1/identity S9.2，与 §9 一致。
5. design `:870,2075`、tasks `:853-869,1093-1109` 与 Plan `:522,605-606,658,680,730-732` 一致保持：S1.3 负责 AM-Q1 decoder/validator/创建纵切及测试先行失败证据；S6.1 仅负责后续 canonical 无归属化端口；共享 `command_service.go` 按 `S1.3 → S6.1` 串行交接。

S6.1 的 task 业务依赖清单未新增 S1.3，但 test-plan 的进入条件和 Plan 的共享文件 lane 已明确规定执行交接顺序；这不再形成可触发的职责或关闭条件歧义。

## 4. 两轮回归结论

- title/body 合同明确且可实现：纯文本；全 Unicode 空白无效；不自动 trim；合法值按原 code point 保留；title≤200 rune、body≤20000 rune。
- Markdown/HTML 只作为普通文本保存和输出；服务端不解析或渲染，消费者必须写入文本节点或做上下文编码，不得直接注入 `innerHTML`。边界与消费者责任一致。
- 创建与修改均覆盖业务校验失败的 `422 article.validation_failed`、wire/UTF-8/JSON 失败的 `400 request.invalid_json`、零 repository 调用和零持久化；修改路径另要求完整 Article 快照不变。
- 262144/262145 actual request-body wire bytes、非法 UTF-8 前置拒绝、JSON 解码、业务 rune 校验与 PostgreSQL `text` 存储契约分层明确；数据库列不替代业务 validator。
- 未发现新增授权依赖、sanitizer、Markdown/HTML renderer、第三方依赖、schema 变更或 AM-Q1 外范围扩大。
- 未发现与 S1.3/S6.1 文件合同、任务责任或其他既有合同的新冲突或误改。
- 现役 `command_service.go`、创建测试、canonical article port 及其局部测试仍待后续任务实现；这与 design draft、test-plan planned-not-executed 状态一致，不阻断本次设计 Gate。

## 5. 新 findings

**无。**

未发现 CRITICAL、HIGH、MEDIUM 或 LOW 新 finding。

## 6. Gate 结论

### Design Review Gate：PASS

精确范围仅为固定 boundary r13、architecture r24、design r09、test-plan r04、Tasks/Plan 同步及 Article AM-Q1 增量：

- `AM-Q1-R02-N001`：**CLOSED**
- 未关闭阻断 finding：**0**
- 新 finding：**无**

r09/r04 修订集已形成无歧义的 Article AM-Q1 实施 Gate，并保持 `article-management-S1.3 → article-management-S6.1`：S1.3 承担测试先行、失败证据及 AM-Q1 实现责任，S6.1 只在交接后承担 canonical 无归属化端口责任。

本 PASS 不外推为代码实现完成、测试已执行、task 已关闭、部署、浏览器验证、真实环境验证、E2E 或 production readiness。

## 7. 限制与未验证项

- 按授权未运行 Go test、race、vet、build、PostgreSQL、部署、浏览器或 E2E；未写 checkpoint。
- 未验证 r09/r04 已被代码实现；现役代码只用于核对真实消费者、接缝和待实施差距。
- 未进行无界全库审查；仅在固定对象、原报告、直接相关 S1.3/S6.1 lane 及真实消费者范围内检查。
- 本报告不批准修改 task 状态、启动后续实施或改变任何交付授权。

## 8. 结束校验

全部固定对象结束 SHA256：

| 对象 | 结束 SHA256 |
|---|---|
| boundary r13 | `C86CEAF315028C7DBA833E97E5A18F7C80721E9AC5939D1EC53F520CA68FE45E` |
| architecture r24 | `FAC19BE91763855EC4CE91D97B8AC052A824AAC643794E4B0F45A01EE04C586B` |
| design r09 | `70EE8FB04196FA78EB925133EC8C796E1B34F5B2060273BCE2A704DE3F797B91` |
| test-plan r04 | `5A1FE258792CF281B27DE2B9E51131E1C46E738F8B3589A184ACFF383A9DDF3A` |
| tasks | `292DE0E5306A222D04E83D8510E24347B5FBB814AC96C62D2061690D38E20EF3` |
| Plan | `CD371BA3FC5824EA61676CA3F9F9A7872070B463ADD3BE333267D47E3F01E385` |

- 固定文档与本报告的 scoped `git diff --check`：PASS；仅出现 Git 的 LF→CRLF working-copy 提示，无 whitespace error。未跟踪报告使用 `git diff --no-index --check`，退出码 1 仅表示与空文件存在内容差异。
- 本报告完整文件 SHA256：由交付回复提供；完整文件哈希无法作为自身内容内嵌而同时保持该哈希不变。
- 实现测试/部署/浏览器/E2E：未运行。
