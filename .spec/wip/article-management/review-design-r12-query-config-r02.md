# Article Management Design r12 QueryConfig 定点复审 R02

## 1. 对象与结论

- objectType：design re-review
- 固定工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 被审对象：`.add/project/knowledge/design-workspace/article-management/design.md`
- revision：`article-management-design-ir01-r12`
- 对象 SHA-256：`CC1710DE628B8DBD43D392E28EAA49CF74AB87C659B004D86D58C44033B16228`
- 原报告：`.spec/wip/article-management/review-design-r11-query-config-r01.md`
- 原报告 SHA-256：`1F82E7EBA7F3061A9D1FE972E83F37518D637B8AA3BD4A31A08CE9DB000F4AAB`
- 原 finding：`AM-DES-R11-QC-R01-F001`
- 原 finding 状态：**CLOSED**
- 新 finding：**无**
- Gate：**PASS**
- S2.1 实施交接：**可以交接**，但仅表示 QueryConfig 设计输入阻塞已解除；`ARTICLE-S2.1-REV-R01-F001` 仍须通过四文件实施、测试和独立代码复审关闭。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

逐项核对原 finding 的修复条件：

1. `.add/project/knowledge/design-workspace/article-management/design.md:783` 明确三个键的字段 parser 独立执行同一合同，各自接受 `1` 与 `2147483647`。
2. 同行明确语法合法的 `2147483648` 归类为 `out_of_range`。
3. `design.md:2105` 将字段 parser 测试与完整 tuple 解耦，不要求其他字段同时满足关系。
4. `design.md:2107` 明确：
   - `2147483647/2147483647/2147483647` 在三个字段解析成功后以 `cross_constraint` 失败，用于覆盖 DefaultLimit 最大值；
   - `1/2147483647/2147483647` 在三个字段解析成功后以 `cross_constraint` 失败，用于覆盖 MaxLimit 最大值；
   - 两者均不得误归类为 `out_of_range`。
5. `design.md:2106` 给出两个满足全部关系的完整成功 fixture：
   - `1/1/2`
   - `1/2147483646/2147483647`
6. 成功和失败 tuple 的预期类别已唯一化，原 finding 所述测试盲区不再存在。

### Round 2：受影响全文与消费者回归检查

1. 核对 revision、变更记录、整改记录和原报告追溯：`design.md:5`、`:65`、`:84`、`:104`。
2. 核对批准合同未漂移：
   - 三个精确环境变量继续必填、无默认；
   - 严格十进制正整数、无 trim、signed 32-bit/int；
   - `DefaultLimit <= MaxLimit`；
   - `CandidateBatchSize > MaxLimit`；
   - DB parse/connect/ping/migrate 前 fail-fast；
   - 错误只含键或稳定类别，不含原值。
3. 核对测试数字边界：`design.md:2106`、`:2108`、`:2289` 均将新 tuple 与既有 `2/2/3` 限定为 fixture，不形成代码默认、生产默认或 AM-Q2 数值。
4. 核对 AM-Q2：`design.md:129`、`:2146`、`:2213`、`:2289` 继续将生产参数、文章规模、容量和 P95 保持 TBD。
5. 核对影响面及关闭边界：`design.md:2185-2187`、`:2277-2278`、`:2289` 继续限定为四文件，并明确仅解除设计输入阻塞、不关闭代码 finding。
6. 复核四个直接消费者文件哈希与 R01 相同，未发现因 r12 引入的新接口、第五写文件、第二 composition root、依赖、migration 或公共契约要求。

消费者证据 SHA-256：

| 文件 | SHA-256 |
|---|---|
| `internal/article/query_service.go` | `C6E99DD6806FC51E16B2221BD75B6188C5BFFA3631E44CC0B7F433C4C6A0A11D` |
| `internal/article/query_test.go` | `1498E5C4E3594A06E0F7BD0A9966A5E1D4D0475AB14B61925FB6058C978B5FB1` |
| `cmd/han-server/main.go` | `1895BAE07E222004CE5191779A0880C008B6B00A53C039826B58AA331D2241C8` |
| `cmd/han-server/main_test.go` | `4EA1BDE6B6FAF4B79028CE0816130AE2E92545570C3A6C4790E29AA2353EF861` |

## 3. 原 finding 复审结论

### CLOSED — AM-DES-R11-QC-R01-F001

- 原问题：最大值测试未区分字段解析合法与完整配置合法。
- 修复证据：`design.md:783`、`:2105-2107`。
- 关闭理由：r12 已分别固定 parser 成功边界、溢出类别、完整成功 tuple 和解析后交叉约束失败类别；实施者无需再自行选择测试语义。
- 回归结果：批准合同、AM-Q2 TBD、四文件影响面和代码 finding 边界均未漂移。

## 4. 新 Findings

无。

## 5. 待决定与非阻断建议

- 待决定：无新增项。AM-Q2 生产数值、文章规模、容量与 P95 按既有合同继续 TBD。
- 非阻断建议：无。

## 6. 限制与未验证风险

- 本轮仅复审固定 design revision，未修改设计、代码、Tasks、Plan 或原报告。
- 未运行 Go 测试、真实 PostgreSQL、SMTP、部署或 E2E；未验证代码实现、容量、查询计划或 P95。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 共享工作树存在其他未提交改动；写报告前已重新核对 branch、HEAD、设计哈希、原报告哈希和新报告路径，无固定对象漂移或报告冲突。
- PASS 不表示代码 Review R01 finding、production wiring、部署、E2E、容量或 production readiness 已关闭。

## 7. 交接条件

QueryConfig 设计输入现可交回 S2.1 实施，但仍须遵守既有 Planning Review 边界：

1. 仅在 `cmd/han-server/main.go`、`cmd/han-server/main_test.go`、`internal/article/query_service.go`、`internal/article/query_test.go` 四文件内实施；
2. 实施前由 Lead 复核四文件无未知写者或交叉；
3. 不新增生产数字、第五文件、第二 composition root、公共契约、依赖或 migration；
4. 完成针对性测试和独立代码复审后，才能决定是否关闭 `ARTICLE-S2.1-REV-R01-F001`。

