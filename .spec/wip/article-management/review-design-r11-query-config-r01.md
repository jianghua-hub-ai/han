# Article Management Design r11 QueryConfig 增量独立评审 R01

## 1. 对象与结论

- objectType：design
- 固定工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 被审对象：`.add/project/knowledge/design-workspace/article-management/design.md`
- revision：`article-management-design-ir01-r11`
- 对象 SHA-256：`CF38A63EDFA7BB8CE5B4CD8EBD8AF7E84C3855003A0AB424B098C59394E66839`
- 审查范围：2026-10-25 production `QueryConfig` 窄增量及其受影响全文一致性
- Gate：**CHANGES REQUIRED**
- S2.1 实施交接：**当前不可交接 F001**。需先关闭本报告的测试合同歧义；关闭后只能解除设计输入阻塞，不能据此关闭代码 Review R01 的 `ARTICLE-S2.1-REV-R01-F001`。

除第 4 节 finding 外，r11 已准确覆盖三个必填且无默认的环境变量、严格解析、signed 32-bit/int 范围、两条交叉约束、DB 前 fail-fast、最小披露、四文件影响面和 AM-Q2 继续 TBD 等批准合同；未发现新增 ADR、公共契约、第五写文件、第二 composition root、migration 或生产数字。

## 2. 依据与限制边界

已完整读取：

- `.codex/agents/add-reviewer.toml`
- `.codex/skills/add-code-review/SKILL.md`
- `.codex/guidance/code-review-checklist.md`
- `.codex/guidance/abstraction-principles.md`
- `.codex/guidance/design-doc-writing-style.md`
- `.codex/guidance/decisions-guide.md`
- `.codex/skills/add-code-review/references/产品安全红线.md`
- design r11 全文、ADR-0008、S2.1 Code Review R01、S2.1 Planning Review R01
- `internal/article/query_service.go`
- `internal/article/query_test.go`
- `cmd/han-server/main.go`
- `cmd/han-server/main_test.go`

批准合同以用户本轮列出的精确内容为准。消费者代码只用于证明可实施性、现状和影响面，不反向改写批准合同。

## 3. 两轮实际检查范围

### Round 1：固定设计对象直接审查

1. 全文核对 r11 revision、变更记录、整改追溯、AM-A11、AM-Q2、配置表、启动顺序、错误披露、可测试性矩阵、可伸缩性、信息需求、四文件影响面及确认边界。
2. 将三个精确键逐项映射到必填/无默认、词法、范围、Go `int` 转换、字段关系、失败时序和错误最小披露。
3. 核对 `2/2/3` 只作为 fixture，生产值、文章规模、容量和 P95 保持 TBD。
4. 核对 r11 只解除设计输入阻塞，不声明代码 finding、实现、测试、E2E、容量或 production readiness 已关闭。
5. 检查全文是否误增 ADR、公共契约、第五写文件、第二 composition root、依赖或 migration。

### Round 2：上游依据与真实消费者有界检查

1. 对照 ADR-0008，确认生产配置仍仅来自环境变量，没有引入第二配置权威。
2. 对照 S2.1 Code Review R01，确认 r11 提供了 F001 所缺的受控来源、精确键名和结构校验，但没有冒充代码修复。
3. 对照 S2.1 Planning Review R01，确认影响面保持四文件，未把 F001 的 design unblock 外推为整卡关闭。
4. 检查现有 `QueryConfig` 构造器不变量和 `2/2/3` fixture；检查 `main.go/main_test.go` 当前配置解析与 DB 启动顺序，确认 r11 描述的是待实施合同而非既成事实。
5. 复核四个消费者文件未提供 production QueryConfig 接线，且无需由本次设计新增第五文件或公共契约。

消费者证据 SHA-256：

| 文件 | SHA-256 |
|---|---|
| `internal/article/query_service.go` | `C6E99DD6806FC51E16B2221BD75B6188C5BFFA3631E44CC0B7F433C4C6A0A11D` |
| `internal/article/query_test.go` | `1498E5C4E3594A06E0F7BD0A9966A5E1D4D0475AB14B61925FB6058C978B5FB1` |
| `cmd/han-server/main.go` | `1895BAE07E222004CE5191779A0880C008B6B00A53C039826B58AA331D2241C8` |
| `cmd/han-server/main_test.go` | `4EA1BDE6B6FAF4B79028CE0816130AE2E92545570C3A6C4790E29AA2353EF861` |

## 4. Findings

### MEDIUM — AM-DES-R11-QC-R01-F001：最大值测试没有区分字段解析合法与完整配置合法

- 位置：`.add/project/knowledge/design-workspace/article-management/design.md:2102`
- 相关合同：同文件 `:762-764`、`:780`
- 现状：验证矩阵要求“合法边界覆盖 `1` 与 `2147483647`”，但没有说明这是单字段解析边界还是完整 `QueryConfig` 成功边界。三个字段都允许解析到 signed 32-bit 最大值，同时完整配置还必须满足 `DefaultLimit <= MaxLimit` 和 `CandidateBatchSize > MaxLimit`。
- 触发场景：
  - `MaxLimit=2147483647` 时，不存在仍在 signed 32-bit 范围内且满足 `CandidateBatchSize > MaxLimit` 的完整成功配置。
  - `DefaultLimit=2147483647` 时，`MaxLimit` 只能等于该值，随后同样无法提供更大的合法 `CandidateBatchSize`。
  - 实施者为每个键编写最大值启动测试时，必须自行决定应断言解析成功后出现 `cross_constraint`、尝试构造不可能的成功 tuple，或跳过 Default/Max 的最大值覆盖。
- 影响：设计没有唯一规定最大值的可观察测试语义，可能漏测字段 parser 对 `2147483647` 的接受边界，或把应进入交叉约束阶段的输入错误归为 `out_of_range`。这不改变批准的数值合同，但使“准确、唯一、可测试”目标尚未完全成立。
- 违反依据：
  - 用户批准合同同时要求三字段适配 signed 32-bit/int 和两条交叉约束。
  - `design-doc-writing-style` 要求清晰、可检查的技术表达并保持同一术语含义。
  - `code-review-checklist` 要求边界与异常有针对性、可验证的测试合同。
- 修复建议：在 §6.1 明确分层断言：
  1. 字段解析层对三个键都接受 `1` 和 `2147483647`；
  2. `DefaultLimit` 或 `MaxLimit` 取最大值时，完整配置预期在字段解析成功后因 `cross_constraint` 失败，而不是 `out_of_range`；
  3. 完整成功 tuple 另用满足关系的测试 fixture，例如下界 `1/1/2`、上界组合 `2147483645/2147483646/2147483647`，并明确这些数字只用于测试，不是生产默认或 AM-Q2 决策。
- 验证方式：复审新 design revision，确认验证矩阵分别给出 parser 边界、完整成功 tuple 和交叉约束失败的预期类别，且未新增生产数值、文件影响面或 ADR。

## 5. 待决定

无新增待决定项。AM-Q2 的三个生产参数具体数值、文章规模、容量和 P95 继续保持 TBD；它们不是修复本 finding 所需的输入。

## 6. 非阻断建议

无。

## 7. 已核对且未形成 finding 的事项

- 三个环境变量名称精确、必填、无默认且无 fixture 回退。
- 严格匹配 `[1-9][0-9]*`，不 trim；空值、空白、符号、前导零、非 ASCII 数字、格式错误和溢出均失败。
- 解析范围固定为 `1..2147483647`，并要求安全转换为 Go `int`。
- `DefaultLimit <= MaxLimit`、`CandidateBatchSize > MaxLimit` 已同时落到配置表、解析说明和构造器末端防线。
- 任一失败均在 DB DSN parse、connect、ping、migrate 及业务对象构造前终止。
- 错误、日志和指标 label 只允许键或稳定类别，不包含配置原值、其他环境内容或敏感信息。
- `2/2/3` 仅为现有测试 fixture；设计未虚构生产数字。
- AM-Q2 的数值、规模、容量和 P95 保持 TBD，部署参数和容量验收仍受阻。
- 影响面保持 `main.go`、`main_test.go`、`query_service.go`、`query_test.go` 四文件。
- r11 未新增 ADR、公共契约、第五写文件、第二 composition root、新依赖或 migration。
- “解除设计输入阻塞”与“关闭代码 finding”边界已明确分离。

## 8. 未验证风险与限制

- 本轮是固定 design 对象的只读评审，未修改被审设计、代码、Tasks 或 Plan。
- 未运行 Go 测试、真实 PostgreSQL、SMTP、部署或 E2E；未验证 AM-Q2 容量、查询计划或 P95。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 共享工作树存在其他未提交改动；写报告前分支、HEAD、对象哈希和报告路径均重新核对，无固定对象漂移或报告冲突。ADR-0008 内容与 HEAD 一致，`framework.md` 的其他并行变化未作为本轮结论依据。
- 本结论只覆盖声明的 QueryConfig 增量、受影响全文和指定消费者，不外推为整仓设计或代码 PASS。

## 9. 复审条件

1. 提供修订后的 design revision 与固定 SHA-256。
2. 引用并逐项关闭 `AM-DES-R11-QC-R01-F001`。
3. 证明最大值的 parser 接受、完整 tuple 成功和 cross-constraint 失败三类预期已唯一化。
4. 保持批准合同、AM-Q2 TBD、四文件影响面及“仅解除设计输入阻塞”边界不变；若改变架构、公共契约或文件边界，先返回相应决策/规划责任人。

