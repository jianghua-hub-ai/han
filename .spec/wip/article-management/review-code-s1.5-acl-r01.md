# S1.5 路径 B ACL 独立评审 R01

## 1. 评审对象与基线

- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审身份：非作者独立评审（add-reviewer），只读评审；本报告是唯一写入文件。
- 对象：`article-management-S1.5`「路径 B」：迁移/应用身份分离、append-only 审计 ACL 启动门禁、部署 SQL 与真实 PostgreSQL 证据。
- 评审依据：ADR-0029（`framework.md` 第 863–908 行）、ADR-0021、ADR-0022、ADR-0016；`article-management` design §3.4.9、§5.3、§6.1、§8.1；`tasks.md` 第 1226 行起的 S1.5 卡（只读）。

### 实际文件哈希

| 文件 | 实际 SHA256 | 与声明/判断 |
|---|---|---|
| `cmd/han-server/append_only_acl.go` | `1019A5E1F38CBB46EAAE1B5A4948DB231D0205117A6D03866AF6F01F435D12AD` | 一致 |
| `cmd/han-server/append_only_acl_test.go` | `54AD3302F95CE4FAE8CACE57A19353FCDFA3646A2248F0597C23E7E7C7771C2B` | 一致 |
| `cmd/han-server/append_only_acl_integration_test.go` | `8A806C38A66299BE47332B0756D345798645F3D10F9E62476ED2F99F7F55579C` | 一致 |
| `cmd/han-server/main.go` | `68E91CFB0BE866A4888950595955A4411DB897DBAB846A39C63E1FE154D982A7` | 一致 |
| `deploy/db/provision.sql` | `0AF8986F34458059AE2F1398F2F49932575D0B57419C6870D2A1BFB46F57BA05` | 与状态同步的新值一致 |
| `deploy/db/grant-application.sql` | `95A93A58DB98776ECBF33AEC0F3ED14E338BB6663520E603152756FA5E2B3351` | 一致 |
| `deploy/db/README.md` | `2F808751CEAA6800D5587A1013857404CFC66046B0916671AD18758FF1448F2F` | **不一致**；任务声明为 `36FF5ABCED64E4B27C38C9FF9931AF28B3AC9F25DBE73A0064DA0DF883B91AD0` |

### 证据与状态

- `run-01.log` SHA256 `43813CA27BA47B60B2846271E1AAD64AD3E6BF2D08673AAA208B7675D4954D82`，6386 bytes；仅有作者既定真实 PG 执行证据，本次未独立复跑。
- `retained-resources.ndjson` SHA256 `E5C035EED4D2F0B3DC4FBC0270C67251D9B5DB46D268BE3189FC884B6B6EE9C6`，222 bytes。
- `provision-probe-r01.log` SHA256 `913DD6A5A0827F399831D727AF3894528FB1FDEEB30C0C34844E21827E64FBE0`，876 bytes。
- `git status --short` 显示 `cmd/`、`deploy/`、`migrations/` 等目标路径在 HEAD 中未跟踪；`git ls-files --error-unmatch` 对目标源码/SQL 返回退出码 1。因此无法用 Git diff 隔离本卡改动，只能评审当前工作树内容。

### 本次实际命令

| 命令 | 退出码 | 结果摘要 |
|---|---:|---|
| `go build ./...` | `0` | 通过，无输出 |
| `go vet -tags=integration ./cmd/han-server` | `1` | `recovery_main_dispatcher_integration_test.go:167:11: undefined: newS419RecoveryWorld` |
| `go test ./cmd/han-server -count=1` | `0` | `ok han/cmd/han-server 47.835s` |
| `$env:HAN_TEST_POSTGRES_DSN='postgres://postgres:<redacted>@127.0.0.1:5433/postgres'; go test -tags=integration ./cmd/han-server -run '^TestAppendOnlyApplicationACL$' -count=1` | `1` | build failed：`strings` imported and not used；`undefined: newS419RecoveryWorld`。未连接 PG、未新建/清理 schema 或角色。 |

## 2. 两轮实际检查范围

- 第一轮：核对 7 个目标文件哈希、HEAD/工作树状态、ADR-0029/设计/任务依据；静态检查 `productionMigration`、DSN 校验、迁移池生命周期、ACL 目录查询、`main.go` 生产入口和 listener 前的装配顺序。
- 第二轮：沿真实消费者和部署路径做对抗检查：superuser/`SET ROLE`、Fallbacks、fresh install 默认权限、schema owner、`search_path`、表名解析与同名历史 schema、列级 UPDATE、集成测试是否真正使用非超管 LOGIN、错误披露和 `provision.sql` 的 psql 变量/`\gexec` 执行面。
- 本次未修改任何源码、SQL、ADR、设计、任务或证据；未执行 Git 写操作；未删除或清理任何 schema/角色。

## 3. 结论

**CHANGES REQUIRED / Code Review Gate: FAIL**

方向正确：双 DSN、独立迁移连接池、migrate 后执行 ACL 门禁、失败发生在 bootstrap/业务装配/listener 之前、使用 `has_table_privilege` 目录判定、INSERT 正控与 UPDATE/DELETE/TRUNCATE 基础负控、表名白名单和参数化查询均已存在。但以下 HIGH 项使当前实现不能作为可交付的 S1.5 通过证据：

- superuser 会话可通过 `SET ROLE`/角色选项绕过当前 `current_user` 检查；
- fresh install 的 `provision.sql` 默认授权与 `grant-application.sql`/启动顺序互相冲突；
- 当前工作树的 integration 包不可构建，允许的真实 PG 集成命令无法独立执行。

## 4. Findings

| 编号 | 级别 | 证据与违反依据 | 触发/影响 | 建议 |
|---|---|---|---|---|
| `S15-ACL-F001` | HIGH | `cmd/han-server/append_only_acl.go:157` 只按 `rolname = current_user` 查询 `rolsuper`；未检查 `session_user`、`current_user = session_user`、角色切换能力或 owner 身份。`cmd/han-server/database_config.go:44` 的外部配置禁止列表不含 `options`。`cmd/han-server/append_only_acl_integration_test.go:208` 用 superuser DSN 加 `SET ROLE` 建立应用池，`run-01.log:3` 显示 preflight `identity=postgres ... superuser=true`，`:7` 仍通过 gate。违反 ADR-0029 #4「superuser/owner 连接不得作为 ACL 通过证据」。 | 应用 DSN 使用 superuser 并在启动参数/`AfterConnect` 中切换到非超管角色时，`current_user` 可为应用角色而 `session_user` 仍是 superuser；会话可再 `RESET ROLE` 恢复特权。当前门禁会误判为应用身份通过。owner 也未被显式分类，现仅依赖有效权限检查间接失败。 | 同时拒绝 `session_user` 为 superuser；要求稳定身份（`current_user = session_user`）或显式拒绝角色切换；检查 `relowner`/`pg_has_role` 的 owner 成员关系；禁止 DSN 通过 `options` 覆盖 `role`。增加 superuser+SET ROLE 必须失败的负控和真实非超管 LOGIN 正控。 |
| `S15-ACL-F002` | HIGH | `deploy/db/provision.sql:45` 给迁移身份新建对象的应用默认权限包含 `UPDATE, DELETE`；`deploy/db/grant-application.sql:15` 和 `:16` 才撤销两表变更权限。`deploy/db/README.md:31` 将 `grant-application.sql` 排在服务启动之后；`cmd/han-server/append_only_acl.go:90` 迁移后立即在 `:93` 执行 ACL gate。集成测试 `cmd/han-server/append_only_acl_integration_test.go:91` 先手工 grant 再 `:97` 跑 gate，未覆盖 provision 默认路径。 | 全新部署按 README 启动时，迁移创建的两张 append-only 表会继承 `UPDATE/DELETE` 默认权限，gate 必然以 `update=true` fail closed；文档顺序不可执行。手工先 grant 的测试掩盖了 fresh install 缺陷。 | 让 fresh install 在 gate 前达到最终 ACL：由迁移显式 REVOKE 两表，或在启动迁移内部于 gate 前执行受控授权；至少不能把 `grant-application.sql` 放在服务启动之后。增加从 provision 默认权限开始的 fresh-install smoke，不得在 test 中先手工 grant。 |
| `S15-ACL-F003` | HIGH | `go vet -tags=integration ./cmd/han-server` 退出码 1；PG 集成命令退出码 1，错误为 `cmd/han-server/recovery_main_dispatcher_integration_test.go:12:2: "strings" imported and not used` 和 `:167:11: undefined: newS419RecoveryWorld`。该文件未跟踪，不属于本评审目标。 | integration 标签包当前不可构建，导致允许的真实 PG 集成无法独立执行；作者 `run-01.log` 不能替代当前工作树的构建/运行证据。此项不是 ACL 作者代码的直接缺陷，但阻断本次验证 Gate。 | 由该文件所有者修复或隔离这组并行产物后重跑允许命令；不要把旧日志当作当前 revision 的集成证据。本评审不修改、不清理该文件。 |
| `S15-ACL-F004` | MEDIUM | `deploy/db/provision.sql:33` 使用 `CREATE SCHEMA IF NOT EXISTS ... AUTHORIZATION migrator`，没有 `ALTER SCHEMA ... OWNER TO`；README `:18` 默认示例为 `schema_name=public`。已存在的 schema 不会因 `IF NOT EXISTS` 改变 owner。`provision.sql:44` 的默认权限仅以 `FOR ROLE migrator` 生效。 | public 通常已由数据库 owner 拥有；migrator 可能既不是其 owner，也未获得 CREATE，迁移会失败。已有专用 schema 若 owner 不同，同样会出现“默认权限看似设置成功但对象 owner/授权对象不匹配”的部署错误。 | 对专用 schema 显式设置 owner；对 public 采用受控的数据库 owner 路径或明确拒绝 public；增加 existing schema/public 两条部署测试，并核对实际对象 owner。 |
| `S15-ACL-F005` | MEDIUM | `cmd/han-server/append_only_acl.go:132` 只比较 primary 的 `Host/Port/Database/User`，未遍历 `ConnConfig.Fallbacks`；同文件 `:119` 的 identity 结构也只含四元组。集成测试 `cmd/han-server/append_only_acl_integration_test.go:199` 明确遍历 fallback，说明测试知道多端点存在。 | 两侧主 host 不同但 fallback 指向同一 host/database/user 时，迁移与应用仍可能连到同一端点却通过“身份不同”检查；同一角色跨库/跨主机也会被视为不同身份，不符合 ADR-0029 的两个 LOGIN 身份目标。 | 规范化并比较 primary + 全部 fallback 端点，或直接拒绝多 host DSN；要求应用与迁移角色名不同并在 provision 中校验；补充 fallback、同 user 跨库/跨 host 的负控。 |
| `S15-ACL-F006` | MEDIUM | `cmd/han-server/append_only_acl_integration_test.go:43` 和 `:47` 创建的是 `NOLOGIN` 角色；`:193`、`:208` 使用 superuser DSN 加 `SET ROLE` 模拟应用和迁移身份；`run-01.log:3` 的 preflight 明确为 superuser=true。 | 测试只证明有效角色的表级 ACL，不证明真实应用/迁移 LOGIN、独立密码认证、`session_user` 拒绝、连接隔离或生产 DSN 路径；并直接掩盖 F001。 | 创建真实 LOGIN 角色和独立应用/迁移 DSN，以非超管会话跑权限矩阵；保留 `SET ROLE` 作为负控或辅助用例，不作为 happy path。 |
| `S15-ACL-F007` | MEDIUM | `cmd/han-server/append_only_acl.go:190` 只查询 `has_table_privilege(c.oid, 'UPDATE')`，没有 `has_any_column_privilege(c.oid, 'UPDATE')`。 | 若外部授予应用列级 UPDATE，表级检查可能为 false，应用仍能修改被授权列；append-only 的“无 UPDATE”目标不成立。trigger 即使拒绝也不能作为 ACL 证据。 | 增加列级 UPDATE 负控（`has_any_column_privilege`）及真实测试；若本版明确不支持列级 ACL，应写入“不证明”边界并禁止外部列级授权。 |
| `S15-ACL-F008` | LOW | `cmd/han-server/main.go:480` 先建立应用池、`:490` ping、`:493` 才进入 `productionMigration`；迁移 DSN 的缺失/相等检查在 `cmd/han-server/append_only_acl.go:60`、`:64`、`:71`，晚于应用 connect/ping。另 `append_only_acl.go:82` 只 defer 关闭迁移池，gate 在 `:93` 执行，迁移池实际保持到 gate 之后；设计 §3.4.9 要求配置校验阶段 fail closed、迁移完成后立即关闭迁移池。 | 非法迁移配置仍会先建立应用连接；迁移池寿命比设计更长。ADR-0029 #2/#3 的 listener 前 fail closed 和 migrate→gate 顺序仍满足，未发现业务绕过，故定为契约/质量偏差。 | 在 `loadStartupConfig` 或应用 connect 前校验迁移 DSN/相等/身份；迁移完成后显式关闭迁移池，再对应用池执行 gate；增加调用顺序和池关闭测试。 |
| `S15-ACL-F009` | LOW | `cmd/han-server/append_only_acl_integration_test.go:40` 在 `:41` 的 preflight 之前调用 `aclRetain`。 | preflight 失败时保留资源登记会出现“已登记但未创建”的假阳性，证据簿记不准确。 | preflight 成功后再登记，或登记状态中区分 planned/created/failed；保持资源登记与实际创建一致。 |
| `S15-ACL-F010` | MEDIUM | 实际 `deploy/db/README.md` 哈希为 `2F80...8F2F`，任务声明为 `36FF...91AD0`；HEAD 不含目标源码/SQL/文档，`git ls-files --error-unmatch` 退出码 1，无法得到“只替换 migrate 并移除 import”的 scoped diff。 | 评审对象未完全冻结；无法证明 `main.go` 没有越界行为，也无法证明本次报告对应的就是作者预审 revision。当前内容静态检查未发现新的 listener 前绕过，但不能替代版本差异证明。 | Lead 重新冻结全部文件哈希（包含实际 README）并提供目标改动 diff/所有权说明；按新哈希重跑或明确接受当前工作树作为唯一对象。 |
| `S15-ACL-F011` | MEDIUM | `internal/migration/runner.go:129`、`:177`、`:221` 使用未限定 schema 的 `schema_migrations`，未设置/校验 `search_path`；`deploy/db/provision.sql:33` 支持自定义 `schema_name`，测试在 `cmd/han-server/append_only_acl_integration_test.go:205` 才手工设置测试 `search_path`。gate 在 `cmd/han-server/append_only_acl.go:171` 只按 `pg_table_is_visible` 要求同名可见关系数恰好为 1。 | 自定义 schema 部署时迁移元数据/表可能落在默认 search_path 而非 provision 指定 schema，`grant-application.sql` 仍按指定 schema 操作，导致启动失败或授权错位。`pg_table_is_visible` 只能证明“当前 search_path 下唯一可见”，不能证明就是 provision 的目标 schema；用户所述 5632 个同名历史 schema 未被清理/隔离。 | 在迁移和应用配置中显式固定并校验 `search_path`，或用目标 schema 限定所有元数据/表操作；测试自定义 schema 的完整生产路径。不要清理历史 schema；保留 fail closed 的唯一可见性检查。 |

## 5. 状态同步专项核实

### 5.1 `provision.sql` 的 `DO $$` 变量问题与注入面

- 现行 `deploy/db/provision.sql:19` 和 `:21` 使用 `SELECT format('CREATE ROLE %I', :'migrator_role') ...`，随后 `:23` 使用 `\gexec`；`:25` 至 `:29` 对应用角色采用同样方式。未再发现 `DO $$` 块内引用 psql 变量的旧错误。
- 标识符使用 `:"role"`、`:"schema_name"`、`:"database_name"`；密码和角色名 SQL 字面量使用 `:'migrator_password'`、`:'migrator_role'`；动态 DDL 使用 `format(... %I ...)`。静态检查未发现直接 SQL 拼接或未引用变量注入点。
- `provision-probe-r01.log:1` 至 `:24` 显示真实事务内 `BEGIN`、创建两个探测角色、`ALTER ROLE`、`GRANT`、`CREATE SCHEMA`、`REVOKE/GRANT`、`ALTER DEFAULT PRIVILEGES`、`ROLLBACK` 全部执行，且探测角色为 `han_s15_prov_mig_0af8986f`、`han_s15_prov_app_0af8986f`，事务回滚后不应保留；它们不是本次保留资源。
- 限制：probe 日志没有记录 psql 版本、输入文件 SHA 或完整命令行，不能独立证明它对应当前 `0AF8986F...` 文件在 PostgreSQL 18.6 上执行；本次也未重跑 psql。`\gexec` 仍是高权限执行面，部署方必须把 psql 变量视为可信输入；密码通过 `psql -v` 传递的进程参数/历史暴露、凭据分发和轮换不在本次证明范围。
- 结论：旧的 `DO $$` 可执行性问题已关闭；未发现新的直接 SQL 注入缺陷；但 F002/F004 的 fresh install 与 schema owner 可执行性问题仍成立。

### 5.2 `main.go` 启动顺序与 ACL 门禁

- `cmd/han-server/main.go:3944` 的 production `migrate` 已改为 `productionMigration`；`runComposed` 在 `:493` 调用 migrate，`bootstrapInitialAdmin` 在 `:496`，Article 装配在 `:505`，业务装配在 `:551`，恢复装配在 `:605` 至 `:648`，business/metrics listener 在 `:651`/`:655`，recovery control listener 在 `:706`，协调在 `:741`。
- `productionMigration` 在 `cmd/han-server/append_only_acl.go:90` 完成 runner、`:93` 执行应用池 ACL gate；gate 失败返回错误，因此不会进入 bootstrap、业务装配或 listener。
- `main` 的普通与 recovery 分支都调用 `productionDependencies()` 和 `run`（`main.go:172` 至 `:195`）；未发现另一条生产入口绕过 `runComposed`。核心顺序与 ADR-0029 #3 一致：migrate → ACL gate → bootstrap → 业务/恢复装配 → listener。
- 与设计 §3.4.9 不完全一致的地方见 F008：迁移 DSN 校验晚于应用 connect/ping，迁移池在 gate 之后才随 defer 关闭。core fail-closed 仍成立，但应修复为配置阶段校验和迁移后立即关闭。

## 6. 本次证明

- 当前工作树 7 个目标文件的实际哈希；`provision.sql` 与状态同步的新哈希一致，`README.md` 与声明不一致。
- 现行 `provision.sql` 的 `SELECT format(...) + \gexec` 结构与 probe 证据未显示旧的 `DO $$` 变量替换错误；未发现直接 SQL 注入点（受 5.1 的证据限制）。
- 静态代码存在双 DSN 缺失/相等/四元组相同检查、独立迁移池、迁移后 gate、INSERT 正控与表级 UPDATE/DELETE/TRUNCATE 负控、`pg_class` + `pg_table_is_visible` 唯一性检查、表名白名单和参数化查询。
- `go build ./...` 退出码 0；`go test ./cmd/han-server -count=1` 退出码 0（47.835s）。
- 既有 `run-01.log` 的 PASS 内容、单一保留 schema 和两个角色；本次未新增或清理资源。
- F002/F004/F005/F007/F011 所述的 fresh install、schema owner、fallback、列级权限和 search_path 部署风险已通过静态文件和测试代码核对。

## 7. 本次不证明

- 不证明当前工作树的 integration 包可构建或真实 PostgreSQL 集成可执行；`run-01.log` 只是既有证据，不是本次独立运行结果。
- 不证明真实非超管 LOGIN、独立密码认证、`session_user` 拒绝、owner 显式拒绝、superuser+SET ROLE 拒绝或列级 ACL。
- 不证明 fresh install、public/已有 schema 部署、自定义 schema 与 `search_path`、5632 个同名历史 schema 的隔离；未清理任何 schema。
- 不证明生产部署、凭据分发、角色轮换、备份/恢复、CI、容量或 RPO/RTO。
- 不证明两张 append-only 表之外全部数据库权限达到最小权限；ADR-0029 #9 已明确此边界，但 F001/F002/F004/F005/F007/F011 是未声明的实现/验证缺口。
- 不证明 `main.go` 相对上一版本只替换 migrate 并移除 import；HEAD 无目标文件，无法 diff。
- 不证明 `provision-probe-r01.log` 对应的正是当前 `provision.sql` 哈希或 psql 18.6 的实际调用；未独立重跑 psql。

## 8. Lead Gate 建议

**不建议 Lead Gate 通过。**

必须先处理 F001、F002、F003 三个 HIGH，并重新冻结评审基线。F004/F005/F006/F007/F010/F011 至少应修复或由 Lead 明确记录风险接受；F008/F009 可作为同批质量修复，不应被误记为已通过。F003 的编译阻断文件不是本次 ACL 目标文件，不能由本评审修改或清理，但必须先恢复 integration 包构建后再验证。

## 9. 复审条件

1. Lead 提供新的完整文件哈希（包含实际 `README.md`）和目标改动 scoped diff/所有权说明。
2. F001 修复后，验证 superuser session、`SET ROLE`、owner、真实非超管 LOGIN 四条路径；F002 修复后，从 provision 默认权限执行 fresh-install smoke；F003 由文件所有者修复/隔离后再跑 integration。
3. 重新执行 `go build ./...`、`go vet -tags=integration ./cmd/han-server`、`go test ./cmd/han-server -count=1` 和真实 PG `TestAppendOnlyApplicationACL`，记录实际退出码、schema/角色和未清理资源。
4. 至少覆盖 fallback、列级 UPDATE、existing/public schema、search_path 和迁移池关闭顺序；修复后由独立评审按本报告原 finding 逐项复核。

## 10. 保留资源与清理边界

本次评审未连接真实 PG、未新建 schema/角色、未执行 DROP/DELETE/TRUNCATE/清理，也未移动证据。既有证据登记的保留资源为：

- schema：`han_s15_acl_a1023308b90a9b3e`
- 迁移角色：`han_s15_mig_a1023308b90a9b3e`
- 应用角色：`han_s15_app_a1023308b90a9b3e`
- 测试：`TestAppendOnlyApplicationACL`
- 登记时间：`2026-10-31T07:33:32.3460303Z`

这些资源必须保留到 Lead 明确清理决策；probe 中的 `han_s15_prov_mig_0af8986f`、`han_s15_prov_app_0af8986f` 已由日志显示随事务 `ROLLBACK`，本次不将其登记为保留资源，也未独立查询数据库确认。

---

**2026-10-31 脱敏说明（Lead，按用户确认执行）**：按用户 2026-10-31 确认，本报告中原引用的本机 loopback 合成 DSN 口令已替换为 `<redacted>`（共 1 处）；其余内容、结论与 findings 未变。脱敏前文件 SHA256 为 `8210B2A0F3B05D08509553E9B080EA2DC5D170B64240441021D3D5366F2AA583`，脱敏后哈希见 tasks.md 同日记录。
