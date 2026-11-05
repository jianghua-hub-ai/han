# identity-auth-S7.2 Code Review R01

- 日期：2026-10-23
- Reviewer：本轮获 Lead 授权的非作者独立 Reviewer
- objectType：code revision review
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 任务：`identity-auth-S7.2`
- 原 finding：`IDENTITY-S7.1-REV-R01-F002`
- 结论：**Code Review Gate: PASS**

## 1. 固定对象

复审开始与测试完成后均核对以下 SHA256，未发生对象漂移：

| 文件 | SHA256 |
|---|---|
| `migrations/0006_user_governance_indexes.sql` | `6F93D99F6DFA0DE46DF8D891F8979039F679493932C28326EEB2AF28E2F24660` |
| `internal/identity/repository_integration_test.go` | `89E6284C41937C13BC2EECBFD509061A0FB5D7DB04EE0DC1E303C65F36A6B8EA` |

固定分支、HEAD 与 Lead 给定值一致。工作树有 75 项既有已修改/未跟踪状态，不能把整树 clean 作为本次证据；本次保留全部他人成果，未清理、重置、覆盖或格式化实现文件。

### 既有测试内容未被改写的独立核对

接手前可信 SHA256 采用 Lead 实测值：

`FE6F63FF1C89E68CD61EA24B94D51D7FDED8947BEDA6FDB3CCEB7837F3A07448`

独立按当前文件的实际 S7.2 增量反向剔除：

1. 删除新增 import `han/internal/session`；
2. 删除 `TestIdentityRepository_UserGovernancePaginationIndexes` 及其紧随的本卡辅助函数连续块（当前第 81～441 行）；
3. 不改写其余字节、换行或格式。

恢复结果为 32092 bytes，SHA256 精确等于
`FE6F63FF1C89E68CD61EA24B94D51D7FDED8947BEDA6FDB3CCEB7837F3A07448`。
因此该证据足以确认：相对于可信接手基线，既有测试内容未被改变，本卡只增加上述 import 与连续测试块。实施者已撤销的未验证 `027E...` 值未被采用。

## 2. 依据与真实消费者

本次依据：

- `.add/project/knowledge/tasks/tasks.md` S7.2（第 717～737 行）；
- `.add/project/knowledge/plans/ir-01-content-community-batches.md` B5D（第 545～560 行）；
- identity design r28 §3.3.8，尤其分页合同与两项索引要求；
- test-plan r11 `I-LIST-01`；
- `.spec/wip/identity-auth/review-code-s7.1-r01.md` 与 r02 中的 F002、转交约束和关闭条件；
- 现役 migration runner、runner integration tests、migration README 与既有 `0001`～`0005`；
- core coding/no-hardcode、安全规则及已启用 `go-service` profile。

最终消费者不是测试中的简化 SQL，而是
`internal/identity/query.go:413-432` 的 `listUserGovernanceSQL`：

- `u.status <> 'deleted'`；
- `u.role = $1`；
- `u.id <> $2`；
- 可选 keyset `(u.created_at,u.id) < ($3,$4)`；
- `ORDER BY u.created_at DESC,u.id DESC`；
- `LIMIT $5`；
- 未关联的当前 privacy notice `LEFT JOIN LATERAL`。

S7.1 五个固定文件的当前 SHA256 均与 r02 固定值一致，其中
`query.go` 为
`534E6E362431B669D271D689365161F95E843D9838D2DA03398F51806DD2C407`。

## 3. 两轮实际检查范围

### Round 1：固定对象直接缺陷

逐项检查：

- `0006` 只创建
  `users_role_status_created_at_id_idx(role,status,created_at DESC,id DESC)`
  与
  `users_created_at_id_idx(created_at DESC,id DESC)`；
- 文件名版本、索引命名、列顺序和 DESC 方向与设计一致；
- SQL 不含 `CONCURRENTLY`、显式事务控制、planner 强制开关、`IF NOT EXISTS` 掩盖或虚构 down migration；
- 现役 runner 按版本升序加载，以 advisory lock 串行运行，并在每个 migration 的同一事务内执行 SQL 与写入 `schema_migrations`；已记录版本会跳过，失败会回滚并停止；
- 测试使用 PostgreSQL 18、随机 schema、外层事务 rollback 和 rollback 后 schema 不存在断言；
- 20k fixture 全部位于随机 schema 的事务内，执行 `ANALYZE users` 后取真实计划，不污染持久数据库；
- 测试未打印或持久化 DSN；Reviewer 仅从 Windows User scope 读取 `HAN_TEST_POSTGRES_DSN` 并注入各目标测试进程。

### Round 2：消费者、上游合同与回归

沿最终 SQL 与实际分页行为检查：

- 普通管理员只能遍历 member；initial admin 只能遍历其他 admin；
- deleted tombstone 不出现，active 与 suspended 均被覆盖；
- initial admin 查询管理员列表时真实验证 self exclusion；
- 多个用户共享同一 `created_at`，按 `id DESC` 跨多页完整遍历，精确断言顺序且不漏不重；
- 分页 limit 分别以 2 和 1 驱动多页，末页断言 `next_cursor=nil`、`has_more=false`；
- role/status 等值路径实际选用复合索引；
- 无过滤排序路径实际选用 `(created_at,id)` 索引；
- 最终 `listUserGovernanceSQL` 连同 lateral join 执行
  `EXPLAIN (ANALYZE, BUFFERS)`，实际选用
  `users_created_at_id_idx`，而非只验证简化 SQL；
- 合成数据规模和分布足以避免小表顺序扫描的偶然性，且未通过关闭 seqscan/indexscan 等 planner 开关强制结果。

对 migration 验证边界的判断：现有两个 runner 用例是 runner 合同级测试，证明正向排序、重跑不重复记录、失败 migration 原子回滚且后续停止；目标 identity 测试则直接执行真实 `0006` 并验证索引定义和访问路径。结合 `0006` 合法命名、无事务控制且只依赖 `0002` 已有 `users` 表，覆盖本卡关闭条件充分；项目没有 down migration 合同，不应补造反向脚本。

## 4. Reviewer 独立命令与结果

DSN 均从 Windows User scope 读取，仅赋给命令进程；命令和输出均未显示其值。

1. `go test -tags=integration ./internal/identity -run '^TestIdentityRepository_UserGovernancePaginationIndexes$' -count=1 -v`
   - PASS，PostgreSQL `18.6`；
   - role/status：`Index Only Scan using users_role_status_created_at_id_idx`；
   - 无过滤排序：`Index Only Scan using users_created_at_id_idx`；
   - 最终 S7.1 SQL：`Index Scan using users_created_at_id_idx`，保留 role/status/deleted/self filter 与 lateral join。
2. `go test -tags=integration ./internal/identity -count=1`
   - PASS，`ok han/internal/identity 4.260s`。
3. `go test -tags=integration ./internal/migration -run '^(TestRunner_AppliesInOrder_RecordsOnlyOnce|TestRunner_FailingMigration_RollsBackAndStops)$' -count=1 -v`
   - 两个用例均 PASS；
   - PostgreSQL conformance baseline 为 `server_version_num=180006`。
4. `go test ./internal/identity ./internal/migration -count=1`
   - 两个 package 均 PASS。
5. `gofmt -l internal/identity/repository_integration_test.go`
   - 无输出。
6. 两固定文件尾随空白与最终换行检查
   - 尾随空白行均为 0；两文件均以 LF 结束。
7. 对象、S7.1 五文件、反向恢复基线、分支、HEAD、暂存区与 status 核对
   - 全部符合本报告所列值；暂存区为空。

Lead 已现场取得 Manifest 七项、两文件 scoped diff-check 与其他所列回归 PASS；本报告没有把这些上游结果冒充 Reviewer 重跑，Reviewer 的独立执行范围如上。

## 5. Findings

按严重度统计：

- CRITICAL：0
- HIGH：0
- MEDIUM：0
- LOW：0

未发现要求保持 OPEN 的正确性、安全、契约、迁移兼容、测试真实性或范围缺陷。

## 6. 原 finding 正式处置

### IDENTITY-S7.1-REV-R01-F002 — CLOSED

关闭证据：

1. design r28 §3.3.8 指定的两项索引已由单向 `0006` migration 实现；
2. PostgreSQL 18.6 已验证索引定义、role/status 与无过滤计划；
3. 最终 S7.1 SQL 已携带真实过滤、keyset、limit 和 lateral join 执行计划，并实际使用治理索引；
4. 同时间戳多页、无漏无重、deleted/self 排除、角色与状态范围均有真实 PostgreSQL 回归；
5. runner 的顺序、重复运行只记录一次与失败事务回滚测试通过；
6. S7.1 五文件未漂移，repository integration test 的既有字节经反向剔除精确恢复可信 before SHA；
7. 无新增依赖、公共契约修改、既有 migration 修改或 down migration 虚构。

**Code Review Gate: PASS**

该结论只关闭 F002 并通过 S7.2 本次代码复审；任务状态更新、B5D 关闭和后续批次授权仍由 Lead/用户处理。

## 7. 范围、暂存区与越界核对

- Reviewer 未修改两个实现文件、既有 migration、任务、计划、设计、旧 Review、`go.mod` 或 `go.sum`。
- 本轮唯一写入为本独立报告：
  `.spec/wip/identity-auth/review-code-s7.2-r01.md`。
- 全局工作树原本已有大量他人状态；本结论仅绑定固定 HEAD、两个对象 SHA、可信 before SHA 和所列消费者哈希。
- 报告落盘后复核：暂存区仍为空；Reviewer 新增内容仅为获授权的本报告。

## 8. 限制与未验证风险

- 20k 合成数据和本机 PostgreSQL 18.6 的计划证据不等同于生产数据分布、生产硬件或长期性能保证；生产性能仍应由后续目标环境 benchmark/profile readiness 承接。
- 本次未执行 Browser、部署、真实用户旅程或跨系统 E2E；这些不在 S7.2/F002 复审范围。
- 未验证生产环境并发 DDL 窗口；当前项目 runner 合同是事务内单向 migration，`CREATE INDEX CONCURRENTLY` 不属于本卡既有合同。
- 若两个固定对象、最终 S7.1 SQL、migration runner 或本报告所列哈希发生变化，需对变化后的修订重新复审。
