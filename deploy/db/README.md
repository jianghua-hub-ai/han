# 数据库身份与 append-only ACL

本目录是 S1.5「应用身份有效 ACL」的部署资产与运行手册。目标是让**迁移身份**与
**应用身份**分离，并在服务开始监听之前用应用连接自证：`audit_events` 与
`processing_gate_records` 只接受 `INSERT`。

## 1. 角色与 DSN

| 用途 | 角色 | 环境变量 | 权限 |
| --- | --- | --- | --- |
| 启动时迁移 | `han_migrator` | `HAN_MIGRATION_DATABASE_DSN` | schema 与对象属主，可 DDL |
| 业务运行 | `han_app` | `HAN_DATABASE_DSN` | 业务 DML，无 UPDATE/DELETE/TRUNCATE |

两者必须是不同的 LOGIN 身份，且角色名必须不同。`han-server` 在解析到相同身份时
**启动失败**：原始 DSN 相同，角色名相同，或同一角色名在主连接与全部 fallback
端点的完整身份（host/port/database/user）集合中出现重叠；不同角色可以共享同一
数据库端点。应用与迁移 DSN 都不允许通过 `options` 参数携带 `role=`、
`-c role=`、`-crole=`、`--role=`、`session_authorization=` 或
`-c session_authorization=` 等角色切换设置。URL 顶层 query 参数 `role=`、
`session_authorization=`（含大小写变体与百分号编码）同样在解析期拒绝；
`application_name` 等非角色切换的运行时参数不受影响。

## 2. 执行顺序

1. 以集群管理员执行 `provision.sql`（创建角色、schema、schema owner、默认权限）：

   ```text
   psql -v ON_ERROR_STOP=1 \
        -v database_name=han -v schema_name=public \
        -v migrator_role=han_migrator -v migrator_password="$HAN_MIGRATOR_PASSWORD" \
        -v application_role=han_app -v application_password="$HAN_APPLICATION_PASSWORD" \
        -f deploy/db/provision.sql
   ```

2. 直接以迁移/应用身份启动服务：

   ```text
   HAN_MIGRATION_DATABASE_DSN="postgres://han_migrator:...@127.0.0.1:5432/han?search_path=public" \
   HAN_DATABASE_DSN="postgres://han_app:...@127.0.0.1:5432/han?search_path=public" \
   HAN_HTTP_ADDR=0.0.0.0:8080 han-server
   ```

   服务在建立应用连接池和 ping 之前先校验迁移 DSN 的缺失、相等、角色名、
   primary 与全部 fallback 端点。校验通过后：

   1. 建立迁移池并以迁移身份执行迁移；
   2. 仍在迁移池中、关闭迁移池之前，对两张 append-only 表执行
      `REVOKE UPDATE, DELETE, TRUNCATE`，并对 `schema_migrations` 执行
      `REVOKE ALL`；
   3. 显式关闭迁移池；
   4. 以应用池执行 ACL gate，通过后才进入 bootstrap、业务装配和 listener。

   fresh install 因此不需要先手工执行 `grant-application.sql`。

3. `grant-application.sql` 用于 out-of-band/人工迁移后的重放或复核，不是 fresh
   install 启动前的必需步骤：

   ```text
   psql "$HAN_MIGRATION_DATABASE_DSN" -v ON_ERROR_STOP=1 \
        -v schema_name=public -v application_role=han_app \
        -f deploy/db/grant-application.sql
   ```

4. 复核（以应用身份执行）：

   ```text
   SELECT has_table_privilege('audit_events', 'INSERT'),
          has_table_privilege('audit_events', 'UPDATE'),
          has_table_privilege('audit_events', 'DELETE'),
          has_table_privilege('audit_events', 'TRUNCATE'),
          has_any_column_privilege('audit_events', 'INSERT'),
          has_any_column_privilege('audit_events', 'UPDATE');
   ```

   期望 `t / f / f / f / t / f`。`processing_gate_records` 同样检查。

## 3. 失败即拒绝启动

`han-server` 在建立监听前按顺序校验，任一步失败都会非零退出：

1. `HAN_MIGRATION_DATABASE_DSN` 缺失或为空白；
2. 迁移身份与应用身份相同（角色名相同，或同一角色名在主/fallback 端点集合中
   出现重叠；不同角色可共享端点）；
3. 迁移连接不可用或迁移执行失败；
4. 应用连接不是稳定身份（`current_user <> session_user`，例如 `SET ROLE` /
   `SET SESSION AUTHORIZATION`）；
5. `current_user` 或 `session_user` 是超级用户；
6. 应用角色是表 owner，或通过 `pg_has_role(..., 'MEMBER')` 继承 owner；
7. `audit_events` / `processing_gate_records` 在有效 `search_path`
   （`current_schemas(false)`，不含隐式 schema）内同名关系数不为 1，或
   `pg_table_is_visible` 可见关系数不为 1；两者使用独立错误类别；
8. 应用角色对上述表的表级 `INSERT` 不为真，或表级 `UPDATE`/`DELETE`/`TRUNCATE`
   任一为真；
9. 列级 `INSERT` 不为真，或列级 `UPDATE`/`DELETE` 任一为真。

第 4–9 步由 `cmd/han-server/append_only_acl.go` 的
`verifyAppendOnlyApplicationPrivileges` 实现。`0001_shared.sql` 的 trigger 只覆盖
`UPDATE`/`DELETE`，不覆盖 `TRUNCATE`；超级用户、表属主和列级授权也可能绕过
trigger，因此 trigger 行为**不能**作为 ACL 证据。

PostgreSQL 18.6 不支持列级 `DELETE` 权限（`has_any_column_privilege(..., 'DELETE')`
会报“未知的权限类型”）。因此 `DELETE` 没有可授予的列级 ACL；表级
`DELETE=false` 已覆盖全部 `DELETE` 授权。gate 通过 `aclexplode(attacl)` 额外拒绝
任何列级 `DELETE` 条目，并检查列级 `INSERT`/`UPDATE`。

## 4. schema 与 search_path 边界

- `provision.sql` 在 `CREATE SCHEMA IF NOT EXISTS ... AUTHORIZATION migrator` 之后
  执行 `ALTER SCHEMA ... OWNER TO migrator`。对已存在的专用 schema，这会改变其
  owner；部署方必须先确认该 schema 没有其他 owner/授权假设。
- `public` schema 通常由数据库 owner 或 `pg_database_owner` 拥有。把 `public`
  owner 改为迁移角色会影响整个数据库的 public schema，属于高风险操作；生产环境
  推荐使用专用 schema。若坚持使用 `public`，必须由数据库 owner 显式接受该
  变更及回滚方案。
- 服务**不设置也不证明** `search_path` 指向 `provision.sql` 的目标 schema。迁移
  DSN 与应用 DSN 必须显式设置相同的 `search_path`，这是运维前置。gate 只证明
  “当前应用有效 `search_path`（`current_schemas(false)`）内同名关系恰好 1 个，
  且该关系 `pg_table_is_visible=true`”；代码不比较 `current_schema()` 与
  provision 的 schema 参数，因此不证明目标 schema 身份。
- 不清理历史同名 schema。gate 要求有效 `search_path`（`current_schemas(false)`）内同名关系数
  恰为 1，不为 1 时一律 fail closed：2 个及以上使用独立的“同名关系歧义”错误类别；
  0 个若同时不满足 `pg_table_is_visible`，则先由“不可定位”类别拒绝，否则也以
  “同名关系歧义”类别拒绝。运维必须确保 `search_path` 只包含 provision 目标 schema，
  而不是依赖 gate 识别 provision 目标。

## 5. 残余风险与不证明

- 迁移凭据仍在与业务进程相同的环境变量空间中；进程被攻破时可提权执行 DDL。
- 彻底隔离需要迁移在进程之外（out-of-band）执行，本版本不做。
- 本目录脚本未在真实部署集群执行过；`deploy/db` 的等价 DDL 由
  `cmd/han-server/append_only_acl_integration_test.go` 在真实 PostgreSQL 18.6 上
  验证，覆盖真实非超管 LOGIN、fresh install 默认权限、superuser+`SET ROLE`
  负控、owner 负控、列级 UPDATE 负控与 mutation 拒绝。
- 不证明部署方的 `search_path` 配置正确、历史同名 schema 已隔离，或 gate 校验的
  关系就是 `provision.sql` 参数指定的 schema。
- 不证明两张 append-only 表之外全部数据库权限达到最小权限。
- 不证明备份/恢复、凭据分发、角色轮换、CI 或真实生产部署。
