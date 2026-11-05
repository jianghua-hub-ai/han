# article-management design r07 + test-plan r02 AM-Q1 增量独立评审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer
- **对象类型**：design + test-plan
- **结论**：**Design Review Gate NOT PASS**
- **Finding 统计**：CRITICAL 0 / HIGH 0 / MEDIUM 2 / LOW 0

## 1. 固定对象与边界

| 项目 | 固定值 / 结果 |
|---|---|
| design | `.add/project/knowledge/design-workspace/article-management/design.md` |
| revision | `article-management-design-ir01-r07` |
| 要求 / 开始 SHA256 | `71669CEA6E0878710DE023A9A029D67644C36897D7BBDA584AB16088FCD1DED0` |
| test-plan | `.add/project/knowledge/design-workspace/article-management/test-plan.md` |
| revision | `article-management-test-plan-ir01-r02` |
| 要求 / 开始 SHA256 | `0D454325C38F75337F84AC4559E9FF49FE01597FFFD581D5D831B6F55690F083` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 增量范围 | 仅 AM-Q1：文章 title/body 格式、空白、长度、原值、输出安全及其测试合同 |

本轮只写本报告。被评审设计、代码、tasks、Plan 和其他文件均保持只读；共享工作树中的既有改动未被回滚、覆盖、清理或格式化。

## 2. 两轮实际检查范围

### Round 1：固定对象直接检查

检查 design r07 与 test-plan r02 中：

- title 200、body 20000 的 Unicode code point / Go rune 计数；
- Unicode whitespace 仅用于判空，合法字符串前后空白与原值保留；
- Markdown/HTML 作为纯文本字面量保存和输出，服务端不解析、不渲染；
- 创建与修改的 422、整次拒绝、零持久化及 version/updated_at/治理状态不变；
- 字段 rune 上限、HTTP 请求体 byte 上限、JSON 解码和 PostgreSQL `text` 的分层；
- sanitizer、renderer、第三方依赖及范围扩张；
- planned-not-executed、实现状态和 Gate 声明。

### Round 2：真实消费者、上游依据与交付合同反查

检查：

- article boundary、architecture、ADR-0004/ADR 索引；
- `internal/article/model.go`、`policy.go`、`handler.go`、`repository.go` 及现役 article 测试；
- `internal/platform/http.go` 的共享 JSON decoder；
- `migrations/0003_article.sql` 的 Article `text` 字段；
- tasks 中 `article-management-S1.3`、`article-management-S6.1`；
- Plan 的 B5 article lane、共享文件所有权及 AM-Q1 阻塞声明。

## 3. Findings

### AM-Q1-R01-F001 — MEDIUM — AM-Q1 已关闭的权威状态未同步到 accepted boundary、active task 与 Plan

**位置**

- `design.md:58,97,106-110,2051`：声明用户已授权关闭 AM-Q1，固定纯文本、200/20000 rune、Unicode 空白及原值保留；
- `test-plan.md:21,41,322`：声明 AM-Q1 已关闭且 ready-for-implementation；
- `design/article-management/boundary.md:145,223,239`：仍声明格式和长度待确认、不得硬编数值、详细设计保持 TBD；
- `tasks/tasks.md:848-864`：S1.3 已 active，但 `:861` 仍称 AM-Q1 阻塞最终 validator 数值；
- `plans/ir-01-content-community-batches.md:519-522,601,654`：B5 article lane 已启动，仍称 AM-Q1 阻塞 S1.3 的最大长度/格式数值。

**违反依据**

设计的决策追溯、accepted 上游契约、tasks SSOT 与 active Plan 必须对同一实施 Gate 给出一致状态；不能让执行者在“已授权 200/20000”与“仍是 TBD、不得编造”之间自行选取。`task-execution-workflow` 要求活动任务以可核对的设计/测试来源和文件合同执行。

**触发场景**

S1.3 实施者按 active task/Plan 开发创建纵切，而不是以 r07/r02 的 AM-Q1 段落覆盖旧阻塞注记；或因 accepted boundary 仍要求 TBD 而停止/跳过最终 validator。

**影响**

- 200/20000 validator 可能被遗漏、延后或被当作未授权；
- S1.3 的关闭条件与测试来源不可唯一判定；
- S1.3 与 S6.1 共享 `command_service.go`，错误交接会污染已规定的 `S1.3 → S6.1` 串行 lane；
- 当前“ready-for-implementation”声明缺少一致的交付 SSOT 支撑。

**修复建议**

由有权限的 Feature-Designer / Task-Planner / Lead 核验用户授权证据后，同步 accepted boundary 的 AM-Q1 状态、S1.3 阻塞注记和 Plan AM-Q1 行；明确 200/20000、共享 validator 与 r02 场景是 S1.3 的实施/验收来源。保持 `S1.3 → S6.1` 串行文件交接，不把 AM-Q1 业务职责扩到 S6.1。Reviewer 不代为修改这些制品。

**证据**

r07/r02 对数值和语义已经一致；冲突仅存在于其引用的上游/下游执行制品，因此这是可复现的跨制品合同冲突，而非风格问题。

### AM-Q1-R01-F002 — MEDIUM — test-plan 缺少请求体 byte 上限和非法 UTF-8 的可执行拒绝用例

**位置**

- `design.md:94,275,840-857,1970`：JSON 为 UTF-8；请求体 byte 上限独立于 rune 上限，必须在解码前执行；超限返回 `400 request.invalid_json` 且不得持久化；
- `decisions/framework.md:147-151`：ADR-0004 约束请求/响应 JSON 字符集为 UTF-8；
- `test-plan.md:93,110-115`：已覆盖 rune、Unicode 空白、原值、HTML/Markdown、422 与修改快照，但未覆盖 wire byte 上限或非法 UTF-8；
- `test-plan.md:195,323`：只把请求体 byte 上限列为 partial/TBD，没有配置确定后的 exact N/N+1 拒绝场景；
- `internal/platform/http.go:62-82`：decoder 有 `maxBodyBytes+1` 边界，但解码前未显式拒绝 invalid UTF-8；Go `encoding/json` 对字符串中的非法 UTF-8 会替换为 U+FFFD，而不是返回错误。

**违反依据**

设计已把传输层 byte 上限、UTF-8、`request.invalid_json` 和零持久化定义为独立于字段 rune 校验的合同。测试规划不能仅以性能维度登记数值 TBD，而遗漏值确定后必须执行的协议边界与数据保真负例。

**触发场景**

1. 创建或修改请求体恰好为配置上限 N bytes，或为 N+1 bytes；
2. JSON 字符串 wire bytes 含非法 UTF-8；
3. decoder 将非法字节替换为 U+FFFD 后进入 validator/repository。

**影响**

- byte 上限配置确定后仍没有创建/修改入口的可执行验收；
- oversized 请求的 `400` 与零持久化无法证明；
- 非法 UTF-8 可能被静默改写并保存，破坏“UTF-8 JSON”和“原值保留”合同；
- 只测 rune 边界不能发现传输层与字段层混淆。

**修复建议**

在 test-plan 增加：

1. 配置值确定后的 exact N / N+1 request-body byte 边界；
2. 创建和修改均断言 N 可解码、N+1 返回 `400 request.invalid_json` 且 Article/版本/时间/治理状态不变；
3. JSON 字符串含非法 UTF-8 时返回 `400 request.invalid_json`，并断言未保存 U+FFFD 替代值；
4. 将这些场景归入 REST/decoder 契约测试；性能任务只测容量/延迟，不代替协议正确性。

设计可补充 decoder 必须拒绝 invalid UTF-8、不得 replacement 的一句实现约束。该建议不要求本轮运行实现测试。

**证据**

现有 A-CREATE-03、A-MOD-02、A-MOD-05 对 rune 与 422 覆盖充分，但全文没有 invalid UTF-8，也没有 byte 上限 N/N+1 的入口、响应和零持久化断言。

## 4. 已确认无新增缺陷的重点

- `design.md:271-277,844-857,976-977,1147,1976` 已明确 title≤200/body≤20000 按 rune 计数，Unicode trim 仅判空，合法值原样保存；
- 创建与修改非法输入均为 422；修改路径明确保持 title/body/version/updated_at 及治理状态原子不变；
- `design.md:267-277,791` 与 `test-plan.md:58,93,110,115` 对纯文本、JSON 转义、文本节点/上下文编码和禁止 `innerHTML` 一致；
- PostgreSQL `title/body` 为 `text NOT NULL`；完整 Unicode/rune 业务规则由共享 validator 承担，数据库约束仅兜底，不存在把 byte、rune 与 DB 类型混为一谈；
- 未授权引入 sanitizer、Markdown/HTML renderer、第三方依赖、新公共 API、schema 或业务范围；
- S6.1 的 canonical 无归属化职责本身未被 AM-Q1 改写；需保持与 S1.3 的共享文件串行交接。

## 5. Gate 结论

### Design Review Gate：NOT PASS

仅针对固定的 design r07、test-plan r02 及 AM-Q1 增量：

- 开放 MEDIUM finding：2；
- title/body 主体语义和输出安全设计本身可实现；
- 但权威制品状态分裂，以及 byte/UTF-8 协议负例缺失，使当前对象尚不足以作为无歧义且完整的实施/验收 Gate。

本结论不否定 r06 历史范围，不批准编码、任务关闭、checkpoint、Git、部署或 E2E。

## 6. 未验证风险与限制

- 按授权未运行 Go test、race、vet、build、PostgreSQL、浏览器或 E2E；
- 未证明当前代码已经实现 r07/r02；design 也明确未作此声明；
- 未独立取得用户授权 AM-Q1 的会话外原始记录，只能确认 r07 写有该声明；授权真实性与 accepted boundary 同步由 Lead/用户确认；
- 请求体 byte 上限当前仍无权威数值，本 finding 要求规划协议场景，不编造数值；
- 当前无现役浏览器前端，DOM 安全只能验证消费者合同，真实 UI/E2E 留待出现消费者后执行。

## 7. 复审条件

1. 固定新的 design/test-plan 修订与完整 SHA256；
2. 提供 F001 对应的 boundary、tasks、Plan 同步修订，或由 Lead 给出不构成冲突的权威状态裁决；
3. test-plan 增加创建/修改的 byte N/N+1、`400 request.invalid_json`、零持久化及 invalid UTF-8 拒绝场景；
4. 复审时保持 `S1.3 → S6.1` 文件合同，不将 AM-Q1 扩入 S6.1；
5. 重新执行固定对象哈希与 scoped `git diff --check`。

## 8. 交付校验

- design 结束 SHA256：`71669CEA6E0878710DE023A9A029D67644C36897D7BBDA584AB16088FCD1DED0`
- test-plan 结束 SHA256：`0D454325C38F75337F84AC4559E9FF49FE01597FFFD581D5D831B6F55690F083`
- 本报告 SHA256：由交付回复提供
- scoped `git diff --check`：PASS（仅有 Git 的 LF→CRLF working-copy 提示，无 whitespace error）
- 实现测试：未运行（按授权）
