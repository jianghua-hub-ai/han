---
feature_id: project-bootstrap
artifact: test-plan
status: partially-verified
verification_status: component-pass-review-pass-pg18-integration-pass-i07-open
revision: project-bootstrap-test-plan-b0.1-r05
created: 2026-10-21
last_updated: 2026-10-21
source_task: project-bootstrap-B0.1
source_architecture_revision: architecture-ir01-r15
source_decision: ADR-0021
source_head: 6da8abbceb0fa197364810a92a8fa2d97b38ddca
---

# project-bootstrap-B0.1 SQL 事务控制 lexer 风险驱动测试规划

## 1. 状态与范围

- **对象**：`internal/migration` SQL 事务控制 lexer，以及其直接消费者 `Load`、`Runner` 和服务启动链。
- **目标**：形成可由 Backend-Dev 一次性实施的 PostgreSQL 18 conformance 风险矩阵，停止围绕单个反例逐点补丁。
- **当前状态**：组件矩阵与独立 Review 已通过；修复后的固定代码对象经独立 Reviewer `01a15d71-9e3d-7331-8ed1-41909ba4e72d` 评审 PASS、无 finding。2026-10-21 在本机 PostgreSQL 18.6 上正式重跑 I-01～I-06，指定 `TestRunner` integration 入口退出码为 0，环境基线、全部场景和清理均通过。Lead 完成最终关闭核对后，已在 tasks SSOT 将 `project-bootstrap-B0.1` 关闭为 `[x]`、完成@2026-10-21。I-07 仍未实施、未验证。
- **包含**：词法边界、事务控制识别、`BEGIN ATOMIC`/`CASE ... END` 结构、失败关闭、加载错误上下文、真实 PostgreSQL 18 行为与事务回滚。
- **不包含**：B1、业务 migration 内容、引入 PostgreSQL parser/新依赖、改变 Runner 的运行时事务所有权或公共契约。
- **期望来源**：B0.1 任务验收、architecture revision `architecture-ir01-r15`、ADR-0021、PostgreSQL 18 语法/词法行为和 migration 事务所有权要求。当前实现只用于定位测试入口与观察点，不用于反推期望。

### 1.1 已知 finding 状态

| Finding | 状态 | 本规划处理 |
|---|---|---|
| B0-CR-001、003、005～011 | 已关闭 | 保留为回归背景，不改写关闭结论 |
| B0-CR-002、004 | PG18 环境限制已解除 | I-01～I-06 已在真实 PostgreSQL 18.6 上执行通过；不再以环境受限标记 |
| B0-CR-012 | CLOSED（原 HIGH） | Red 证明 CR 行注释后的 `COMMIT`/`ROLLBACK` 原先 `error=nil`；修复后 CR/LF 均终止行注释，组件 conformance PASS；原 Reviewer Banach 复审关闭 |
| B0-CR-013 | CLOSED（原 MEDIUM） | Red 证明畸形 DSN 的 pgx 原始错误可能包含密码 sentinel；修复后 setup 错误只输出稳定阶段名，sentinel/原始 DSN 定向测试 PASS；原 Reviewer Banach 复审关闭 |

## 2. 测试判定与公共约定

### 2.1 判定原则

1. migration 文件由 Runner 自有事务执行；文件中的顶层事务控制不得取得、提交、回滚或重塑该事务。
2. PostgreSQL 18 会解释为顶层事务控制的输入，校验器必须在数据库副作用前拒绝。
3. 注释、字符串、quoted identifier、dollar-quoted body 和合法 `BEGIN ATOMIC` body 内的相同字节不得误报为顶层事务控制。
4. malformed/unterminated lexeme 不得吞掉后续事务控制；不确定时必须 fail-closed。
5. PostgreSQL 语法错误必须回滚、不记录 migration 版本并停止后续 migration。
6. migration 加载或执行失败必须阻止服务建立 HTTP listener。

### 2.2 参数化约定

以下组件矩阵中：

```text
T = COMMIT | ROLLBACK
```

凡输入写作 `...; T;`，Backend-Dev 必须生成并独立断言 `COMMIT` 与 `ROLLBACK` 两个 case；不得只用其中之一代表另一项。对合法 body，至少同时保留：

- body 自身不含顶层事务控制时的接受 case；
- 同一 body 后追加 `COMMIT` 的拒绝 case；
- 同一 body 后追加 `ROLLBACK` 的拒绝 case。

### 2.3 测试层与证据约定

| 层次 | 用途 | 建议自动化入口 | 当前状态 |
|---|---|---|---|
| 组件 | 对 scanner/validator 和 `Load` 做确定性、表驱动验证 | `go test ./internal/migration -run "TestValidateSQLTransactionControlConformance|TestLoad_TransactionControlConformance" -count=1` | **PASS（2026-10-21）** |
| PostgreSQL 18 integration | 用真实 PG18 验证 lexical oracle、DDL/atomic、事务回滚和版本记录 | `go test -tags=integration ./internal/migration -run "TestRunner" -count=1 -v` | **PASS（2026-10-21）：退出码 0；I-01～I-06 及 Runner 顺序、回滚、并发锁、连接释放测试均通过** |
| 服务启动 integration | 验证 migration 错误传播到 `han-server` 且 listener 不启动 | I-07 需复用真实服务启动入口并记录精确命令 | **I-07 未实施、未验证；既有启动错误传播组件测试 PASS** |

组件 conformance 与 PG18 I-01～I-06 均已取得自动化 PASS 证据。I-07 仍是未实施的独立服务启动 integration 规划项，既有组件传播测试不替代它。

### 2.4 环境基线边界

PostgreSQL 18 integration 执行前必须查询并记录：

- `server_version_num` 的 major 为 18；
- 数据库 encoding 为 UTF8；
- session `client_encoding=UTF8`；
- session `standard_conforming_strings=on`。

上述 UTF8、`client_encoding=UTF8`、`standard_conforming_strings=on` **仅是本轮 integration 的待确认执行前置和证据元数据，不是已确认的生产运行时合同**。若环境不满足，不得自行改变生产配置或据此改写架构/ADR；应停止受影响验证并交 Lead 决定。

## 3. 风险清单

| 风险 ID | 风险 | 影响 | 主要覆盖 |
|---|---|---|---|
| R-01 | 顶层事务控制被 comment/string/dollar body 吞掉 | migration 可夺取 Runner 事务，原子性和版本记录失真 | C-01～C-05、S-01～S-04、D-01～D-04、M-01～M-06、I-02、I-06 |
| R-02 | 合法 body 内关键字被误报 | 合法 migration 或服务启动被拒绝 | C-04～C-05、S-01～S-04、D-01～D-04、A-01～A-03、I-03～I-04 |
| R-03 | malformed lexeme 与 PostgreSQL 解析分歧 | 后续事务控制被隐藏或错误 SQL 到达数据库 | M-01～M-06、I-05～I-06 |
| R-04 | dollar delimiter 邻接或 Unicode 分类错误 | identifier 被截断、错误开启 body、绕过校验 | N-01～N-05、D-02～D-04、I-03 |
| R-05 | Unicode lookalike 被当作 ASCII keyword | 非关键字误报，或边界判断不一致 | K-01～K-04 |
| R-06 | ordinary string 行为依赖 session setting | 组件与真实 PG18 oracle 不一致 | S-02、I-01、I-03 |
| R-07 | `BEGIN ATOMIC`/`CASE ... END` 深度错误 | body 提前关闭或后置事务控制绕过 | A-01～A-04、I-04～I-06 |
| R-08 | PG18 证据缺失却关闭 B0 | conformance 未被真实数据库证明 | I-01～I-07；Gate 不得通过 |
| R-09 | integration setup 错误回显原始 DSN/驱动错误 | 密码等敏感信息进入测试日志或 CI 输出 | B0-CR-013 定向 sentinel 测试与独立 Review |

## 4. 组件测试矩阵

### 4.1 注释、字符串与引用

| ID | 前置与输入 | 预期 | 最低测试层 | 自动化证据 | 清理 | 状态 |
|---|---|---|---|---|---|---|
| C-01 | validator；`SELECT 1; -- ordinary comment\nT;` | LF 结束行注释；两个 T case 均在执行前被识别并拒绝 | 组件 | `TestValidateSQLTransactionControlConformance/line_comment_lf_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| C-02 | validator；`SELECT 1; -- ordinary comment\rT;` | CR 结束行注释；两个 T case 均被拒绝；直接回归 B0-CR-012 | 组件 | `.../line_comment_cr_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| C-03 | validator；`SELECT 1; -- ordinary comment\r\nT;` | CRLF 作为一个行终止边界；两个 T case 均被拒绝 | 组件 | `.../line_comment_crlf_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| C-04 | validator；`SELECT 1; -- T at eof`，文件在注释中结束 | 注释内 T 被忽略，且无顶层事务控制时接受 | 组件 | `.../line_comment_eof_contains_keyword` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| C-05 | validator；`SELECT 1 /* outer /* T */ still outer */; T;` | nested block comment 内 T 被忽略；comment 关闭后两个后置 T case 均被拒绝 | 组件 | `.../nested_block_comment_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| S-01 | validator；`SELECT 'it''s T'; T;` | doubled quote 不结束 ordinary string；字符串内 T 忽略，两个后置 T case 均拒绝 | 组件 | `.../ordinary_string_doubled_quote_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| S-02 | integration 前置确认 `standard_conforming_strings=on`；组件输入 `SELECT 'backslash\\''; T;`，并设置与 PG18 oracle 配对的合法/非法样本 | 不擅自把反斜杠当 ordinary string 的 quote escape；组件结果必须与该 session setting 下 PG18 词法结果一致，且不得吞掉两个后置 T case | 组件 + PG18 integration | `.../ordinary_string_backslash_scs_on_*`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| S-03 | validator；`SELECT E'escaped \\' T'; T;` | E-string（escape string）escaped quote 内 T 忽略；两个后置 T case 均拒绝 | 组件 | `.../escape_string_escaped_quote_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| S-04 | validator；`SELECT "identifier "" T"; T;` | quoted identifier 内 T 忽略；doubled quote 正确处理；两个后置 T case均拒绝 | 组件 | `.../quoted_identifier_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| D-01 | validator；`SELECT $$ T $$; T;` | `$$...$$` body 内 T 忽略；两个后置 T case均拒绝 | 组件 | `.../dollar_quote_empty_tag_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| D-02 | validator；`SELECT $body$ T $body$; T;` | ASCII tag body 内 T 忽略；两个后置 T case 均拒绝 | 组件 | `.../dollar_quote_ascii_tag_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| D-03 | validator，UTF-8 fixture；`SELECT $é$ T $é$; T;` | 高位字符 tag 的接受/拒绝行为必须与 PG18 oracle 一致；不得因按 byte/rune 分类错误吞掉两个后置 T case | 组件 + PG18 integration | `.../dollar_quote_high_bit_tag_then_{commit,rollback}`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| D-04 | validator，UTF-8 fixture；tag 含真实 U+FFFD：`SELECT $�$ T $�$; T;` | U+FFFD tag 的行为必须与 PG18 oracle 一致；无论 tag 是否被 PG18 接受，两个后置 T 不得被静默吞掉 | 组件 + PG18 integration | `.../dollar_quote_replacement_character_tag_then_{commit,rollback}`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |

### 4.2 Dollar delimiter 邻接、identifier 与 ASCII-only keyword

| ID | 前置与输入 | 预期 | 最低测试层 | 自动化证据 | 清理 | 状态 |
|---|---|---|---|---|---|---|
| N-01 | validator；`SELECT foo$guard$; T;` | `$guard$` 与 ASCII identifier 邻接时不得开启 dollar body；两个后置 T case 均拒绝 | 组件 | `.../ascii_identifier_adjacent_dollar_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| N-02 | validator，UTF-8 fixture；`SELECT é$guard$; T;` | Unicode identifier 邻接时不得错误开启 dollar body；两个后置 T case 均拒绝 | 组件 + PG18 integration | `.../unicode_identifier_adjacent_dollar_then_{commit,rollback}`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| N-03 | validator；`SELECT 1, $guard$ T $guard$; T;` | whitespace/标点合法分隔后的 `$guard$` 才作为 delimiter；body 内 T 忽略，两个后置 T case 均拒绝 | 组件 | `.../separated_dollar_delimiter_then_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| N-04 | validator；使用 `$1tag$` 包围含 T 文本，随后追加 `T;` | 数字开头的伪 tag 不得让 scanner 吞到远端同文本；必须 fail-closed 或继续识别两个顶层 T，不能接受 | 组件 + PG18 integration | `.../invalid_digit_start_dollar_tag_{commit,rollback}`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| N-05 | validator；使用 tag 中含额外 `$` 的伪 delimiter，随后追加 `T;` | 非法 tag 不得形成跨区域 body；必须 fail-closed 或识别两个顶层 T，不能接受 | 组件 + PG18 integration | `.../invalid_embedded_dollar_tag_{commit,rollback}`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| K-01 | validator；分别输入 `cOmMiT;`、`rOlLbAcK;` | ASCII keyword 大小写不敏感；均被拒绝 | 组件 | `.../ascii_mixed_case_{commit,rollback}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| K-02 | validator，UTF-8 fixture；`COMMIＴ;`（末字符为全角 U+FF34） | 不得归一化为 ASCII `COMMIT`；按 PG18 非关键字/语法结果处理，不得产生“顶层 COMMIT”误判 | 组件 + PG18 integration | `.../unicode_lookalike_commit`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| K-03 | validator，UTF-8 fixture；`BÉGIN ATOMIC ...`、`ENDé` | 非 ASCII 拼写不得变成 `BEGIN`/`END`，不得错误改变 atomic 深度；追加两个 T case 必须被拒绝或 fail-closed | 组件 + PG18 integration | `.../unicode_nonkeyword_begin_end_{commit,rollback}`；对应 I-03 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| K-04 | validator，UTF-8 fixture；atomic body 中使用 `caſe`（U+017F）并追加 `T;` | 不得把 Unicode long-s 大小写折叠成 ASCII `CASE`；不得错误改变 CASE 深度或吞掉两个后置 T | 组件 + PG18 integration | `.../ascii_only_case_keyword_{commit,rollback}`；对应 I-03/I-04 | 内存 fixture；integration 按随机 schema 清理 | **组件 PASS；对应 PG18 入口 PASS（2026-10-21）** |
| W-01 | validator；`START/*x*/TRANSACTION;` | block comment 可作为 token whitespace；识别完整顶层事务控制并拒绝 | 组件 | `.../start_block_comment_transaction` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| W-02 | validator；`SET--x\rTRANSACTION;` | CR 终止 comment 后形成 `SET TRANSACTION`；必须拒绝 | 组件 | `.../set_line_comment_cr_transaction` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |

### 4.3 顶层事务控制语句族

| ID | 前置与输入 | 预期 | 最低测试层 | 自动化证据 | 清理 | 状态 |
|---|---|---|---|---|---|---|
| T-01 | validator；`SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;` | 拒绝，不允许修改 Runner 所有事务 | 组件 | `.../set_transaction` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| T-02 | validator；`SET SESSION CHARACTERISTICS AS TRANSACTION ISOLATION LEVEL SERIALIZABLE;` | 拒绝事务特征控制 | 组件 | `.../set_session_characteristics_as_transaction` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| T-03 | validator；`END;` | 作为 `COMMIT` 同义事务控制拒绝 | 组件 | `.../end_transaction` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| T-04 | validator；`ABORT;` | 作为 `ROLLBACK` 同义事务控制拒绝 | 组件 | `.../abort_transaction` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| T-05 | validator；分别输入 `SAVEPOINT s;`、`RELEASE SAVEPOINT s;`、`ROLLBACK TO SAVEPOINT s;` | 全部拒绝，migration 不得重塑 Runner 事务边界/回滚点 | 组件 | `.../{savepoint,release_savepoint,rollback_to_savepoint}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |
| T-06 | validator；分别输入 `PREPARE TRANSACTION 'x';`、`COMMIT PREPARED 'x';`、`ROLLBACK PREPARED 'x';` | prepared transaction 变体全部拒绝 | 组件 | `.../{prepare_transaction,commit_prepared,rollback_prepared}` | 内存 fixture，无 | **PASS（组件，2026-10-21）** |

### 4.4 `BEGIN ATOMIC`、nested CASE 与结构边界

| ID | 前置与输入 | 预期 | 最低测试层 | 自动化证据 | 清理 | 状态 |
|---|---|---|---|---|---|---|
| A-01 | 合法 SQL function：`CREATE FUNCTION ... LANGUAGE SQL BEGIN ATOMIC SELECT 1; END;`；另各追加 `COMMIT;`/`ROLLBACK;` | 无后置 T 时组件接受；atomic body 内合法语句不误报；两个后置 T case 均拒绝 | 组件 + PG18 integration | `.../function_begin_atomic_{body_only,then_commit,then_rollback}`；对应 I-04/I-06 | 内存 fixture；随机 schema | **组件 + PG18 PASS（2026-10-21）** |
| A-02 | 合法 SQL procedure：`CREATE PROCEDURE ... LANGUAGE SQL BEGIN ATOMIC ... END;`；另各追加 T | 无后置 T 时组件接受；两个后置 T case 均拒绝 | 组件 + PG18 integration | `.../procedure_begin_atomic_{body_only,then_commit,then_rollback}`；对应 I-04/I-06 | 内存 fixture；随机 schema | **组件 + PG18 PASS（2026-10-21）** |
| A-03 | 合法 atomic body 含 nested `CASE ... CASE ... END ... END`，body 后分别追加 T | CASE 的 END 不得提前关闭 atomic；两个后置 T 必须在 atomic 真正结束后被识别并拒绝 | 组件 + PG18 integration | `.../begin_atomic_nested_case_then_{commit,rollback}`；对应 I-04/I-06 | 内存 fixture；随机 schema | **组件 + PG18 PASS（2026-10-21）** |
| A-04 | 在 atomic body 内放置裸 nested `BEGIN ATOMIC ... END`；一组无后置 T，一组分别追加 T | 不得把裸 nested atomic 描述为 PG18 合法 body。追加 T 时组件必须拒绝事务控制或 fail-closed，不能绕过；无 T 时由 PG18 判语法错误、整笔回滚且不记录版本 | 组件 + PG18 integration | `.../bare_nested_begin_atomic_{body_only,then_commit,then_rollback}`；对应 I-05/I-06 | 内存 fixture；随机 schema，失败后显式核回滚并 drop | **组件 + PG18 PASS（2026-10-21）** |

### 4.5 Malformed/unterminated lexeme fail-closed

| ID | 前置与输入 | 预期 | 最低测试层 | 自动化证据 | 清理 | 状态 |
|---|---|---|---|---|---|---|
| M-01 | `Load` 读取含 `/* unterminated T;` 的 migration；T 参数化 | 返回错误，执行前 fail-closed；不得将 T 吞至 EOF；错误含 migration path | 组件 | `TestLoad_TransactionControlConformance/unterminated_block_comment_{commit,rollback}` | 临时目录/文件由 `t.TempDir` 自动清理 | **PASS（组件，2026-10-21）** |
| M-02 | `Load` 读取含 `SELECT 'unterminated T;` 的 migration；T 参数化 | 返回错误，执行前 fail-closed；错误含 migration path | 组件 | `.../unterminated_ordinary_string_{commit,rollback}` | `t.TempDir` | **PASS（组件，2026-10-21）** |
| M-03 | `Load` 读取含 `SELECT E'unterminated T;` 的 migration；T 参数化 | 返回错误，执行前 fail-closed；错误含 migration path | 组件 | `.../unterminated_escape_string_{commit,rollback}` | `t.TempDir` | **PASS（组件，2026-10-21）** |
| M-04 | `Load` 读取含 `SELECT "unterminated T;` 的 migration；T 参数化 | 返回错误，执行前 fail-closed；错误含 migration path | 组件 | `.../unterminated_quoted_identifier_{commit,rollback}` | `t.TempDir` | **PASS（组件，2026-10-21）** |
| M-05 | `Load` 读取含 `SELECT $body$ unterminated T;` 的 migration；T 参数化 | 返回错误，执行前 fail-closed；错误含 migration path | 组件 | `.../unterminated_dollar_quote_{commit,rollback}` | `t.TempDir` | **PASS（组件，2026-10-21）** |
| M-06 | `Load` 读取 `$left$ ... T; $right$` 等 mismatched tag；T 参数化 | 不得把不匹配的关闭 tag 当终止；最终返回带 migration path 的错误，不执行 | 组件 | `.../mismatched_dollar_tag_{commit,rollback}` | `t.TempDir` | **PASS（组件，2026-10-21）** |

## 5. PostgreSQL 18 真实 integration 矩阵

共同前置：

1. Lead/环境责任人提供专用 `HAN_TEST_POSTGRES_DSN`，允许创建和删除随机 schema。
2. 连接真实 PostgreSQL 18，不得用 mock、兼容实现或其他 major 代替。
3. 每个 case 使用唯一 schema 与唯一 migration version；连接与事务不得跨 case 复用。
4. 执行前记录第 2.4 节环境值；任一不满足则标记环境不符，不得宣称通过。

共同清理：测试成功或失败均最终 `ROLLBACK` 未闭事务；`DROP SCHEMA <random> CASCADE`；关闭连接；确认测试创建的 migration/version 记录已按预期保留或不存在。

| ID | 前置与输入 | 预期 | 最低测试层 | 自动化证据 | 清理 | 状态 |
|---|---|---|---|---|---|---|
| I-01 | 连接 DSN；查询 server major、database encoding、`client_encoding`、`standard_conforming_strings` | major=18；其余满足本轮待确认基线；结果写入测试日志 | PG18 integration | `TestRunnerConformance_EnvironmentBaseline` + 完整命令输出 | 共同清理 | **PASS（2026-10-21）：180006 / UTF8 / UTF8 / on** |
| I-02 | 用 PG18 直接执行/解析最小 oracle，分别覆盖 `--x\nT`、`--x\rT`、`--x\r\nT` | 证明 LF、CR、CRLF 均结束行注释，后续 T 被 PG18 作为独立 token/statement；与组件 C-01～C-03 一致 | PG18 integration | `TestRunnerConformance_LineCommentTerminatorsOracle` | 共同清理 | **PASS（2026-10-21）：LF、CR、CRLF 子测试全部通过** |
| I-03 | 合法 lexeme pack：nested comment、ordinary/E string、quoted identifier、各 dollar tag/邻接/Unicode case；不含顶层 T | PG18 接受的合法 pack 经 Runner 成功并只记录一次 version；PG18 拒绝的样本在文档化 oracle case 中被明确区分，不伪装成合法 migration | PG18 integration | `TestRunnerConformance_LexemePack` | 共同清理 | **PASS（2026-10-21）** |
| I-04 | 创建合法 SQL function/procedure `BEGIN ATOMIC`，含 nested CASE；经 Runner 迁移后调用 | migration 成功、对象可调用、结果正确、version 只记录一次 | PG18 integration | `TestRunnerConformance_BeginAtomicNestedCase` | 共同清理 | **PASS（2026-10-21）** |
| I-05 | migration 含裸 nested `BEGIN ATOMIC`，不含顶层 T | PG18 报语法错误；本 migration 副作用回滚、不记录 version、后续 migration 不执行 | PG18 integration | `TestRunnerConformance_RejectsBareNestedAtomic` | 共同清理 | **PASS（2026-10-21）** |
| I-06 | 对 C/S/D/N/A 各合法 body 后分别追加 `COMMIT`、`ROLLBACK`，设置可观察 DDL/DML 副作用 | Runner 在数据库副作用前拒绝；两个 T 变体均无对象/数据/version 残留 | PG18 integration | `TestRunnerConformance_RejectsTransactionControlAfterBody` | 共同清理 | **PASS（2026-10-21）：28 个参数化子测试全部通过** |
| I-07 | 通过真实服务启动入口加载一份必然失败的 migration | migration 错误传到 `han-server`；listener 未创建/端口不可连接；进程按既有失败合同退出 | 服务启动 + PG18 integration | 复用既有服务 integration，记录精确测试名、命令、退出状态和端口断言 | 共同清理；进程与 listener 必须关闭 | **未实施、未验证；本轮明确未执行** |

## 6. 非功能与安全基线

### 6.1 安全/完整性

- PG18 会解释为顶层事务控制的输入绝不能被 validator 接受。
- malformed/unterminated lexeme 必须在 SQL 执行前失败。
- PG18 语法错误必须回滚、不记录当前 version、停止后续 migration。
- migration 失败不得建立 HTTP listener。
- 错误必须包含 migration path，但不得泄露 DSN、密码或其他凭据。

### 6.2 性能与健壮性

- scanner 应保持输入长度的线性扫描，不因反复回溯产生明显超线性行为。
- 长 line/block comment、ordinary/E string、quoted identifier、dollar body，以及深 nested comment/CASE 输入不得 panic、死循环或越界。
- fuzz seed 至少覆盖 CR/LF/CRLF、single/double quote、反斜杠、`$`、高位 UTF-8、U+FFFD、nested delimiter 和后置 T。
- 当前没有经需求确认的输入容量或时延预算，因此本规划不虚构硬阈值。若实施发现需要限制 migration 大小或 nesting depth，必须返回结构性设计/ADR，不得由测试自行设为生产合同。

建议自动化：

```text
go test ./internal/migration -run TestValidateSQLTransactionControlConformance -count=1
go test ./internal/migration -run TestLoad_TransactionControlConformance -count=1
go test ./internal/migration -run TestSQLScannerLongInput -count=1
go test ./internal/migration -fuzz FuzzSQLTransactionControlScanner
```

前两个 conformance 入口已实施并执行通过；`TestSQLScannerLongInput` 与 fuzz 入口仍是规划建议，未取得独立执行证据。

## 7. 覆盖与证据映射

| 风险 | 场景 | 最低层次 | 原任务/Case | 已有或待取得证据 |
|---|---|---|---|---|
| R-01 | comment/string/dollar body 后 T | 组件 + PG18 integration | `project-bootstrap-B0.1`；C-01～C-05、S-01～S-04、D-01～D-04、M-01～M-06、I-02/I-06 | 组件矩阵、PG18 I-02/I-06 PASS；B0-CR-012 Review CLOSED |
| R-02 | 合法 body 不误报 | 组件 + PG18 integration | B0.1；C-04/C-05、S/D、A-01～A-03、I-03/I-04 | 组件与 PG18 I-03/I-04 PASS |
| R-03 | malformed/unterminated fail-closed | 组件 + PG18 integration | B0.1；M-01～M-06、I-05/I-06 | 组件与 PG18 I-05/I-06 PASS |
| R-04 | dollar 邻接和 Unicode tag | 组件 + PG18 integration | B0.1；N-01～N-05、D-03/D-04、I-03 | 组件与 PG18 I-03 PASS |
| R-05 | ASCII-only keyword | 组件 + PG18 integration | B0.1；K-01～K-04、I-03/I-04 | 组件与对应 PG18 入口 PASS |
| R-06 | ordinary string/session setting | 组件 + PG18 integration | B0.1；S-02、I-01/I-03 | 组件与 PG18 I-01/I-03 PASS；session 值仍非生产合同 |
| R-07 | atomic/CASE 深度 | 组件 + PG18 integration | B0.1；A-01～A-04、I-04～I-06 | 组件与 PG18 I-04～I-06 PASS |
| R-08 | 缺少真实 PG18 自动化证据 | PG18 integration | B0.1；I-01～I-07；B0-CR-002/004 | Runner I-01～I-06 已取得真实 PG18 PASS；I-07 仍独立未验证 |
| R-09 | setup 错误泄露 DSN/密码 | 组件安全回归 + Review | B0.1；B0-CR-013 | sentinel/原始 DSN 定向测试 PASS；原 Reviewer 复审 CLOSED |

## 8. 实际执行证据与未执行边界

### 8.1 B0-CR-012 Red/Green 与组件矩阵 PASS

- Red：`-- ordinary comment\rCOMMIT;` 与 `-- ordinary comment\rROLLBACK;` 原先均得到 `error=nil`，证明 CR 未终止行注释。
- Green：修复后 CR/LF 均终止行注释，LF/CR/CRLF 及完整 scanner/load conformance 矩阵通过。

```text
go test ./internal/migration -run "TestValidateSQLTransactionControlConformance|TestLoad_TransactionControlConformance" -count=1
```

结论：**组件 PASS**；B0-CR-012 经原 Reviewer Banach 复审 **CLOSED**。该结果不证明 PostgreSQL 18 conformance。

### 8.2 B0-CR-013 Red/Green 与安全回归 PASS

- Red：畸形 DSN 的 pgx 原始 setup 错误可能包含密码 sentinel。
- Green：setup 错误只输出稳定阶段名，不包含 sentinel 或原始 DSN。

```text
go test -tags=integration ./internal/migration -run "TestPostgresSetupError_MalformedAndFailedDSN_RedactsSensitiveInput" -count=1
```

结论：定向测试 **PASS**；B0-CR-013 经原 Reviewer Banach 复审 **CLOSED**。

### 8.3 Lead 现场质量门 PASS

2026-10-21，Lead 现场执行并确认以下项目通过：

- sentinel 定向测试；
- scanner/load conformance；
- `go mod verify`；
- `gofmt -l .`；
- `go vet ./...`；
- `go build ./...`；
- `go test -tags=contract ./...`；
- `go test -race ./...`；
- `go test ./...`；
- `git diff --check`。

既有服务启动错误传播组件测试随适用测试集通过；这不替代 I-07 真实服务启动 PostgreSQL 18 integration。

### 8.4 PostgreSQL 18 integration Gate PASS

```text
$env:HAN_TEST_POSTGRES_DSN='postgres://<redacted>@127.0.0.1:5433/postgres?sslmode=disable'
go test -tags=integration ./internal/migration -run "TestRunner" -count=1 -v
```

固定代码对象：

```text
internal/migration/runner_integration_test.go
SHA-256 a6925a4a42bd0d29ef1210ba578b485458adc51a3b39a037160d46f3a2ec800f
```

r03 的失败由第 576 行将 text `SHOW server_version_num` 扫描到整数导致。修复仅把查询改为 `SELECT current_setting('server_version_num')::integer`；独立 Reviewer `01a15d71-9e3d-7331-8ed1-41909ba4e72d` 对上述固定哈希 Review PASS、无 finding。

2026-10-21 15:55:08 +08:00 至 15:55:17 +08:00，命令真实执行，退出码为 0。运行前与自动化日志基线一致：

```text
psql client=18.6
server_version_num=180006
server_version=18.6
database_encoding=UTF8
client_encoding=UTF8
standard_conforming_strings=on
preexisting han_migration_% schema count=0
preexisting migration advisory lock count=0
```

逐项结果：

```text
I-01 PASS  TestRunnerConformance_EnvironmentBaseline
I-02 PASS  TestRunnerConformance_LineCommentTerminatorsOracle（LF/CR/CRLF）
I-03 PASS  TestRunnerConformance_LexemePack
I-04 PASS  TestRunnerConformance_BeginAtomicNestedCase
I-05 PASS  TestRunnerConformance_RejectsBareNestedAtomic
I-06 PASS  TestRunnerConformance_RejectsTransactionControlAfterBody（28 个子测试）
```

同一入口中的迁移顺序/幂等记录、失败回滚与停止、并发 advisory lock 串行化、专用连接释放测试也全部 PASS。运行后只读核对 `han_migration_%` schema 数为 0，migration advisory lock 数为 0。命令输出未出现完整 DSN、密码或密码 sentinel。

### 8.5 实现对象与证据指纹

| 文件 | SHA-256 |
|---|---|
| `internal/migration/sqlcheck.go` | `c99977348271d1f7a0589992b3e6937c823f928cb21b9d9d49a6bbb0589a4f7a` |
| `internal/migration/sqlcheck_test.go` | `232e5ca353d2c4d7e46edafa9d81885e22ffd1497ba452819c70fb0274b426a8` |
| `internal/migration/runner_integration_test.go` | `a6925a4a42bd0d29ef1210ba578b485458adc51a3b39a037160d46f3a2ec800f` |

### 8.6 独立 Review

- 原 Reviewer：Banach。
- 第一轮复审：B0-CR-012 **CLOSED**；发现 B0-CR-013（MEDIUM）。
- 作者修复 B0-CR-013 后，原 Reviewer 第二轮复审：B0-CR-013 **CLOSED**。
- 两轮复审后无新增 finding。
- I-01 基线查询一行修复的独立 Reviewer：`01a15d71-9e3d-7331-8ed1-41909ba4e72d`；评审对象 SHA-256 `a6925a4a42bd0d29ef1210ba578b485458adc51a3b39a037160d46f3a2ec800f`；Review PASS、无 finding。
- Review PASS 本身只说明上述 findings 已关闭；PostgreSQL 18 integration 的 PASS 结论来自第 8.4 节独立执行证据。

### 8.7 未验证

- I-07：真实服务启动 PostgreSQL 18 integration 未实施、未验证。
- 第 6.2 节独立 long-input/fuzz 建议：未取得执行证据。
- `project-bootstrap-B0.1`：Lead 已在 tasks SSOT 关闭为 `[x]`、完成@2026-10-21。

## 9. 任务建议与登记状态

- **复用原任务**：`project-bootstrap-B0.1`。
- **tasks SSOT 当前状态**：`[x]`，完成@2026-10-21。
- **当前实施状态**：组件矩阵、修复 Review 与 PG18 I-01～I-06 均 PASS；I-07 尚未实施。
- **任务登记**：本轮不新增 task、不修改 tasks SSOT；只引用 Lead 已关闭的 `project-bootstrap-B0.1`。
- **结构性停止条件**：若实现需要引入 PostgreSQL parser/新依赖、改变公共契约、改变 Runner 事务所有权或生产运行时 session 配置，停止实施并返回 Lead 发起设计/ADR，不得在 B0.1 内自行决定。
- **关闭判断**：Lead 已完成固定工作树、验收命令、所有权与清理核对，并于 2026-10-21 正式关闭 B0.1。I-07 属于独立服务启动 integration，仍未实施、未验证，不冒充通过。B1 的 B0 前置现已解除；是否启动 B1 由后续任务或批次授权决定，本测试计划不授予启动权限。

## 10. add-test-design 当前 Gate

| Gate | 结论 | 说明 |
|---|---|---|
| 期望来自真实任务/设计/决策而非当前实现 | PASS | 以 B0.1、architecture r15、ADR-0021 和 PG18 行为为准 |
| 风险到场景、层次、原任务/Case、证据可追踪 | PASS | 见第 3、4、5、7 节 |
| LF/CR/CRLF、nested comment、string/identifier/dollar、Unicode/邻接、atomic/CASE、malformed 全覆盖 | PASS | 见 C/S/D/N/K/A/M 系列 |
| 每类合法 body 后 `COMMIT`/`ROLLBACK` 均覆盖 | PASS | T 参数化约定及各行明确要求双 case |
| 组件与真实 PostgreSQL 18 integration 分离 | PASS | 第 4、5 节分别定义 |
| 实际结果、待执行与环境阻塞分离 | PASS | 第 8 节 |
| 安全红线与 fail-closed 可观察 | PASS | 第 2.1、6.1 节 |
| 组件矩阵实施与执行 | **PASS** | 2026-10-21 Lead 现场验证 |
| 独立 Review | **PASS** | B0-CR-012、B0-CR-013 均 CLOSED；无新增 finding |
| PostgreSQL 18 实际验证 | **PASS** | 指定命令退出码 0；I-01～I-06 全部通过，基线与清理符合 |
| I-07 真实服务启动 integration | **NOT IMPLEMENTED / NOT VERIFIED** | 既有组件传播测试通过不能替代真实启动验证 |
| UTF8/session setting 定位 | PASS（待确认前置） | 仅为 integration 元数据，不是生产合同 |
| B0.1 整体状态 | **CLOSED / PASS** | tasks SSOT 已为 `[x]`、完成@2026-10-21；I-07 为独立未验证项 |
| B1 的 B0 前置 | **已解除** | 是否启动 B1 由后续任务或批次授权决定 |
