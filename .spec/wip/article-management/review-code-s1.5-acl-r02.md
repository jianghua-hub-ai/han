# S1.5 ACL 修复复审 R02（add-reviewer 独立评审）

- 评审对象：`article-management-S1.5` 卡点「路径 B」的 ACL 门禁子集（数据库身份分离、启动迁移后 REVOKE、应用池 ACL gate、部署 SQL/README）。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 评审性质：对 R01 findings `S15-ACL-F001`–`F011` 的返工复审（revision 路径），只读；唯一写入为本报告。
- 范围声明：**只评 S1.5 的 ACL 门禁子集，不对 S1.5 整卡下 PASS**。audit SQL、其它卡点子项不在本次结论内。

## 1. 结论

**CHANGES REQUIRED（窄范围）**

- ACL 门禁核心代码 `S15-ACL-F001`–`F009` 经本轮独立复核均已关闭；未发现可由应用身份直接绕过 gate 的阻断级代码缺陷。
- 仍阻断核收的是原 `S15-ACL-F011`：代码未校验/固定 `search_path` 指向 provision 目标 schema，且 `deploy/db/README.md:115-116` 与集成测试注释 `append_only_acl_integration_test.go:38-39` 宣称“同名关系数不为 1 会 fail closed”，与 PG 18.6 实测不符（见 `S15-ACL-R02-F001`，MEDIUM）。
- 该问题的低风险修复可以只改文档与测试断言；但“虚假的 fail-closed 宣称”属于安全文档错误，修正前不建议 Lead Gate 以“ACL 已 fail closed”通过。修正后可转 `PASS WITH MINOR`。
- `S15-ACL-F010` 已按 Lead 冻结的 8 文件哈希关闭；但目标文件在 HEAD 中未跟踪，仍无 Git diff 可证明逐行改动范围，只能以哈希冻结作为基线。

## 2. 评审基线

### 2.1 文件哈希（本轮实测，与 Lead 给定基线一致）

| 文件 | SHA256 |
|---|---|
| `cmd/han-server/append_only_acl.go` | `FFA71581CAC2EE05A43E0B99ECBF5A4C68D5FE32CC0B3CC0FD6284511FD1D282` |
| `cmd/han-server/append_only_acl_test.go` | `5DBE9C14D225FCCDFAD10F42C8C073E4FCAA715CDA307FD13D9F060895444C7F` |
| `cmd/han-server/append_only_acl_integration_test.go` | `F9C288BE0576E415C6CC27BCE9F97D15C0728DD6E85E14F3209B9645CD16D605` |
| `cmd/han-server/database_config.go` | `6B29E8720593D6BE19F68AFFE053F037173F5DE04A17AB6B4D9F2453A83846BC` |
| `cmd/han-server/main.go` | `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` |
| `deploy/db/provision.sql` | `5252556099A335A480041A62ED30C7618384A5B4D038BFA29356423C446B8C23` |
| `deploy/db/grant-application.sql` | `FC0D229818D48407EF4CCBF97D0860E08E781FF8EF102201C551560741AF0AE8` |
| `deploy/db/README.md` | `AFBD40278AA3CE23BCB3319CCDD08FD7E46CFB9E0BC70D1808CD73B53A5A6087` |
| `.spec/wip/article-management/evidence-s1.5/acl-r02/index.json` | `7921D68E0D4991C9E9B696E1A66C203520BD70D5FD03AA90EF320544EE354CBE` |
| R01 报告（输入） | `8210B2A0F3B05D08509553E9B080EA2DC5D170B64240441021D3D5366F2AA583` |

Git 事实：`cmd/han-server/`、`deploy/db/` 整体为未跟踪（`??`），`git cat-file -e HEAD:cmd/han-server/append_only_acl.go` 退出码 128；因此**不存在 scoped git diff**，`main.go` 的“只替换 migrate 并移除 import”只能做静态观察，不能做版本差异证明。

### 2.2 实际执行命令与退出码

| 命令 | 退出码 | 结果摘要 |
|---|---:|---|
| `go build ./...` | 0 | 通过 |
| `go vet ./cmd/han-server` | 0 | 通过 |
| `go vet -tags=integration ./cmd/han-server` | 0 | 通过；**R01 F003 不复现** |
| `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 40.395s` |
| `go test -tags=integration ./cmd/han-server -run '^TestAppendOnlyApplicationACL$' -count=1` | 0 | PASS 11/11；见 `run-14.log`、`run-15.log` 及本轮复跑 |
| 只读 psql 同名关系探针（见 §5） | 0 | `visible_count=1`、`on_search_path_count=2` |
| `psql "...?options=-c role=<app>"` 只读探针 | 0 | `current_user=app`、`session_user=postgres`、`rolsuper=f` |
| `psql "...?role=<app>"` / `"?session_authorization=<app>"` | 1 | libpq 报 `invalid URI query parameter`；仅说明 libpq，不代表 pgx 行为 |

说明：任务文本给出的 `-run TestAppendOnlyACLGate` 在当前代码中**没有匹配测试**（会 0 个测试而“通过”）；实际执行使用真实测试名 `^TestAppendOnlyApplicationACL$`。该差异是输入命令问题，不是代码缺陷。

### 2.3 保留资源（禁止清理）

- 本轮复跑新增并保留：schema `han_s15_acl_584fb9085cb32425`，迁移角色 `han_s15_mig_584fb9085cb32425`，应用角色 `han_s15_app_584fb9085cb32425`；已在 `retained-resources.ndjson` 追加 `planned` 与 `created` 两行。
- 证据日志涉及且保留：`han_s15_acl_558476dd50435f81`（run-14）、`han_s15_acl_2dd15584ab2d5af9`（run-15）、`han_s15_acl_68214477f0e649e7`、`han_s15_acl_07eb389eccb77ab3`、`han_s15_acl_756fd7e977a0e722` 等既有 schema/角色。
- **未 DROP、未清理、未移动任何 schema、角色或证据。**

## 3. 逐条 finding 复核

| 编号 | R01 级别 | R02 判定 | 关键证据（当前行号） | 独立判断 |
|---|---|---|---|---|
| `S15-ACL-F001` | HIGH | **CLOSED** | `append_only_acl.go:158` 拒绝 `current_user != session_user`；`:164` 同时拒绝 current/session superuser；`:184` + `:253` 拒绝表 owner 或 owner 成员；`database_config.go:57`、`:109-112`、`:152-174` 拒绝 `options` 内 `role=`/`session_authorization=` 的 `-c`/`--` 变体；`main.go:3936-3943` 在 parse 阶段校验，`:480` 才建立应用池 | 三条路径均被实际阻断：superuser+`SET ROLE` 因身份不一致拒绝；owner 因 owner 成员检查拒绝；`options=-c role=` 在 DSN 解析期拒绝。DSN 级拒绝先于应用池 connect/ping，属于 fail closed；会话级身份检查只能在连接后执行，但发生在 listener 与业务装配之前。集成负控 `:257/:269/:284` 与真实非超管正控 `:164` 均 PASS。 |
| `S15-ACL-F002` | HIGH | **CLOSED** | `append_only_acl.go:110` 迁移后 `restrictApplicationMutations`；`:116` 显式 `migrationPool.Close()`；`:117` 才 gate；REVOKE 由 `migrationPool.Exec` 执行（`:327`、`:337`），两表 `REVOKE UPDATE, DELETE, TRUNCATE`（`:320-330`），`schema_migrations` `REVOKE ALL`（`:331-339`）；`provision.sql:46-49` 仅默认权限；集成测试 `:123-131` 无手工 grant 直接走生产迁移 | fresh install 不再依赖 `grant-application.sql`；REVOKE 发生在 gate 之前且由迁移身份执行。`run-14.log` 记录 `default_privileges_only=true manual_grant=false`、`fresh_install_default_privileges_gate_passed` PASS。文档顺序 `README.md:23-49` 已与实现一致。 |
| `S15-ACL-F003` | HIGH | **CLOSED** | `go vet -tags=integration ./cmd/han-server` 退出码 0；`run-12.log`；`run-14.log`/`run-15.log` 完整 PASS | integration 包当前可构建、可运行；R01 的未使用 import/未定义符号不复现。 |
| `S15-ACL-F004` | MEDIUM | **CLOSED** | `provision.sql:37-38` `CREATE SCHEMA IF NOT EXISTS ... AUTHORIZATION` + `ALTER SCHEMA ... OWNER TO`；`:39` `REVOKE CREATE ... FROM PUBLIC`；`:40-41` 撤销应用全部并仅授 `USAGE`；`README.md:104-110` 明确 public schema 高风险；集成测试 `:144-162` 断言 owner=migrator 且非 superuser | 已存在的专用 schema 会显式改 owner；public 风险已文档化。owner 断言在真实 PG 通过。 |
| `S15-ACL-F005` | MEDIUM | **CLOSED** | `append_only_acl.go:441-455` 遍历 primary + 全部 `Fallbacks`（nil fallback 报错）；`:465-497` 角色名相等与 `(host,port,database,user)` 集合重叠均拒绝；`:457-463` host 归一化；单测 `:333-436` 覆盖 fallback、同角色跨库/跨 host、nil fallback | 端点比较只读配置对象，不建立连接，无连接泄漏；同角色跨库/跨 host 也会被拒绝，符合“两个 LOGIN 身份”的目标。 |
| `S15-ACL-F006` | MEDIUM | **CLOSED** | 集成测试 `:61-72` 创建真实 `LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE` 角色；`:104-109` 独立应用/迁移 DSN；`:164-196` 断言 current/session 均为应用角色、`rolsuper=false`、`rolcanlogin=true` | 权限矩阵确实以非超管应用角色执行，不再是 superuser+SET ROLE 模拟 happy path；日志只含角色名，不含密码。 |
| `S15-ACL-F007` | MEDIUM | **CLOSED** | `append_only_acl.go:263-266` 列级 INSERT/UPDATE；`:267-279` `aclexplode(attacl)` 列级 DELETE；`:205-210` 拒绝；集成测试 `:296-330` 真实列级 UPDATE 授权被拒，撤销后通过 | PG 18.6 不支持列级 DELETE，代码用 `aclexplode` 防御性覆盖；实测查询可执行且行为符合预期。该查询会拒绝任意 grantee 的列级 DELETE 条目，属于保守失败，不构成安全漏洞。 |
| `S15-ACL-F008` | LOW | **CLOSED** | `main.go:3936-3943` parse 阶段调用 `validateProductionMigrationConfiguration`；`:433` parse → `:480` connect → `:490` ping → `:493` migrate；`append_only_acl.go:116` 迁移池在 gate 前关闭 | 迁移 DSN 缺失/相等/共享身份在应用池建立前失败；迁移池在 gate 前显式关闭。错误只返回稳定类别，不拼接 DSN/密码（`database_config.go:74-77`）。 |
| `S15-ACL-F009` | LOW | **CLOSED** | 集成测试 `:53` preflight → `:59` `planned` → `:61-98` 创建 → `:99` `created`；`retained-resources.ndjson` 每个 schema 均为 planned 在前 | 资源登记语义与实际创建一致；失败时不会出现“未创建但已登记 created”的假阳性。 |
| `S15-ACL-F010` | MEDIUM | **CLOSED（有残余限制）** | Lead 冻结 8 文件哈希；本轮逐个实测一致；`git ls-files`/`git cat-file` 证明 HEAD 未跟踪目标文件 | 评审对象已冻结，满足本轮复核基线。但**没有 Git diff**，无法证明 `main.go` 相对上一修订“只替换 migrate 并移除 import”；该点只能以静态阅读（`:3936-3943` 增加校验、`:3951-3953` 替换为 `productionMigration`）和 Lead 冻结声明为依据。 |
| `S15-ACL-F011` | MEDIUM | **PARTIAL** | `internal/migration/runner.go:28/130/177/221` 仍使用未限定 `schema_migrations`，未设置/校验 search_path；`append_only_acl.go:178`、`:355-373` 用 `pg_table_is_visible` 判唯一；`README.md:111-116` 声明不证明目标 schema，但 `:115-116` 又宣称同名关系会 fail closed；测试注释 `:35-39` 同样宣称 | 文档化边界已增加，但代码未固定/校验 search_path，且“同名关系数不为 1 会 fail closed”的宣称**被实测推翻**：search_path 上有 2 个同名关系时 `pg_table_is_visible` 只返回 1 个，gate 通过。因此原 finding 不能关闭，见 `S15-ACL-R02-F001`。 |

## 4. 新发现问题

### S15-ACL-R02-F001 — MEDIUM：`pg_table_is_visible` 不检测同名关系歧义，README/测试的 fail-closed 宣称不成立

- 证据：
  - 代码：`cmd/han-server/append_only_acl.go:178`（`privileges.visible != 1`）、`:355-373`（`resolveVisibleTableSchema` 的 `count != 1`）。
  - 只读 PG 18.6 探针：`SET search_path = han_s15_acl_07eb389eccb77ab3, han_s15_acl_2dd15584ab2d5af9, pg_catalog;` 后查询同名 `schema_migrations`，输出 `1|2|han_s15_acl_07eb389eccb77ab3`（`visible_count=1`、`on_search_path_count=2`）。
  - 文档：`deploy/db/README.md:115-116`“若数据库中已有同名关系，gate 会因可见关系数不为 1 而 fail closed”；`cmd/han-server/append_only_acl_integration_test.go:38-39` 同样宣称。
- 触发场景：应用 `search_path` 的上游存在历史同名 schema，且其中已有同名表；gate 只检查第一个可解析对象，不会发现后续同名关系。
- 影响：gate 能证明“应用当前实际解析到的这张表”ACL 正确，但**不能证明它就是 provision 的目标 schema**，也不会像文档承诺的那样对同名历史 schema fail closed。部署可能静默落到历史 schema；若历史 schema 中旧审计表仍保留应用角色的 UPDATE/DELETE 授权，应用可通过 schema 限定名修改旧审计数据。这不是当前 live 表 ACL 的直接绕过，但属于未闭合的 schema 隔离/可观测性风险，且文档存在虚假安全承诺。
- 建议（任选其一，均为低风险）：
  - 方案 A（仅文档，最小）：把 `README.md:115-116` 改为“gate 只验证当前应用 search_path 的有效解析对象；历史同名 schema 不会被检测，运维必须保证 search_path 只含 provision 目标 schema”，并同步修改测试注释 `:35-39`；在 ADR-0029“明确不证明”中补一条 search_path/目标 schema 身份。
  - 方案 B（最小代码）：在 `resolveVisibleTableSchema` 中同时统计 `n.nspname = ANY(current_schemas(true))` 的同名关系数，要求恰好为 1 且 `current_schema()` 与配置目标一致；或显式比较 `current_schemas(true)` 与 provision schema。
  - 方案 C（最小测试）：新增“search_path 含两个同名 schema 时 gate 必须拒绝”的单测；当前实现会失败，可防止文档再次超前于代码。
- 定级理由：不是 ACL 权限判定的直接错误（live 表仍按实际解析对象检查），但属于 MEDIUM 的安全边界与文档真实性问题；在原 F011 已为 MEDIUM 的前提下，不降级。

### S15-ACL-R02-F002 — LOW：URL 顶层 `role`/`session_authorization` 由 pgx 接受为启动参数，解析期未显式拒绝

- 证据：
  - `cmd/han-server/database_config.go:48-55` 外部配置禁用键不含 `role`/`session_authorization`；`:109-112` 只对 `options` 键应用正则；`:152-174` URL 分支也只检查 `options`。
  - pgx v5.11.0 `pgconn/config.go:323-324` 明确说明 libpq 不认识的 URL query 参数会被 pgx 接受为 runtime parameters；`:498-536` 将其写入 `RuntimeParams`。
  - 本机 psql/libpq 对 `?role=`、`?session_authorization=` 返回 `invalid URI query parameter`（退出码 1），但这**不能代表 pgx**；`?options=-c role=<app>` 实测可得到 `app|postgres|f`。
- 影响：若 pgx 将顶层 `role=` 作为启动参数生效，应用连接可在应用池建立后变成 `current_user != session_user`；`append_only_acl.go:158` 会拒绝，因此**未观察到 gate 绕过**。但 README “建立应用连接池之前先校验角色切换设置”的表述只对 `options` 变体成立，对 pgx 的 URL runtime params 不成立。
- 建议：在 `externalDatabaseConfigKeys` 或 keyword/URL 统一入口显式拒绝 `role`、`session_authorization` 及非白名单 runtime params；或增加一个纯 `pgxpool.ParseConfig` 单测，断言 `?role=` 在解析期被拒绝。若不改代码，应把 README 措辞收窄为“拒绝 options 内的角色切换”，并说明会话级检查由 gate 兜底。
- 定级理由：gate 有会话身份兜底，未形成实际提权路径；属于防御纵深与文档精度问题，LOW，不阻断。

### S15-ACL-R02-F003 — LOW：ADR-0029 的“两阶段脚本必须执行”与当前“fresh install 自动 REVOKE”不一致

- 证据：`framework.md:873`、`:893`、`:896` 仍表述部署通过 provision + grant-application 两阶段建立授权，且运维必须先执行两阶段脚本；`README.md:51-54` 与 `append_only_acl.go:110-117` 已改为启动迁移自动 REVOKE、`grant-application.sql` 仅 out-of-band 重放。
- 影响：实现方向更安全（不依赖人工顺序），但与已接受 ADR 的部署契约文字不一致；若按 ADR 审计，会被认为“未执行 grant-application.sql 即启动”是偏差。
- 建议：由 Lead 决定并更新 ADR-0029 #5/#8 的措辞（把 grant-application.sql 定位为 out-of-band 重放/复核），或在验收记录中明确该实现偏差已被接受。属于决策文档一致性，不是代码缺陷。
- 定级理由：不影响当前安全目标，LOW。

## 5. 本次证明与本次不证明

### 5.1 本次证明

- 双 DSN 缺失、完全相等、角色名相等、primary/fallback 端点集合重叠均在应用池 connect/ping 之前 fail closed（静态调用链 + 单测）。
- `options` 内 `role=`/`session_authorization=` 的 `-c`/`--` 变体在 DSN 解析期被拒绝；`options=-c role=<app>` 的提权形态已被真实 PG 探针复现并由代码拒绝。
- 启动顺序为：配置解析（含迁移 DSN 校验）→ 应用池 connect/ping → 迁移 → 迁移身份 REVOKE 两表与 ledger → 显式关闭迁移池 → 应用池 gate → bootstrapInitialAdmin → 业务装配 → listener（`main.go:433/480/490/493/496`，`append_only_acl.go:110-117`）。
- 应用身份为真实非超管 LOGIN；superuser、superuser+SET ROLE、表 owner、owner 成员均被拒绝；两表 INSERT 正控通过，UPDATE/DELETE/TRUNCATE 六个负控均为 SQLSTATE 42501；列级 UPDATE 被拒绝；schema CREATE 被拒绝；ledger 被拒绝。
- fresh install 仅依赖 provision 默认权限即可通过迁移 + REVOKE + gate，不依赖手工 `grant-application.sql`。
- `provision.sql` 的 dollar-quote 变量替换缺陷已消除：当前文件无 `DO $$`，角色创建使用 `SELECT format(...) + \gexec`；标识符/字面量使用 `%I`、`:"..."`、`:'...'`，未发现新的可执行性或直接 SQL 注入缺陷。
- PG 18.6 集成测试 PASS 11/11；`go build`、两种 `go vet`、非 PG 单测均通过。

### 5.2 本次不证明

- 不证明真实生产部署、CI、凭据分发/轮换、备份恢复演练、RPO/RTO。
- 不证明 `search_path` 指向 provision 目标 schema，也不证明历史同名 schema 已被隔离（F011 / R02-F001）。
- 不证明两张 append-only 表之外全部数据库权限达到最小权限（ADR-0029 #9）。
- 不证明 listener 实际套接字未打开；当前证据是调用顺序与依赖注入的静态证明，不是运行时抓包/端口观测。
- 不证明 pgx 顶层 `role=` runtime param 的真实提权效果（受“不得写临时文件”约束，未写独立 pgx 探针）；若生效，gate 的 `current_user != session_user` 会兜底。
- 不证明多版本 PG 兼容性（仅 18.6），不证明并发业务流量下 REVOKE/gate 的窗口行为（listener 尚未启动，窗口内无业务请求）。
- 不证明 Git 级修订差异：目标文件未跟踪，只能以 Lead 冻结哈希作为对象基线。

## 6. 复核所用证据与命令引用

- 代码：`cmd/han-server/append_only_acl.go:28/49/51/110-117/158-210/215-295/303-341/355-373/379-412/441-497`；`database_config.go:21-57/82-115/145-174`；`main.go:433/480/490/493/496/3936-3943/3951-3953`；`append_only_acl_test.go:238-311/333-436`；`append_only_acl_integration_test.go:35-39/53/59/99/123-131/144-196/226-233/250-330`。
- SQL/文档：`deploy/db/provision.sql:17-49`；`deploy/db/grant-application.sql:7-21`；`deploy/db/README.md:21-54/75-100/102-116`。
- 证据日志：`evidence-s1.5/acl-r02/run-08.log`（options role 探针）、`run-12.log`（integration vet）、`run-14.log`、`run-15.log`（PASS 11/11）、`retained-resources.ndjson`（15 行）、`index.json`。
- 只读探针输出：
  - 同名关系：`1|2|han_s15_acl_07eb389eccb77ab3`。
  - `?options=-c role=<app>`：`han_s15_app_584fb9085cb32425|postgres|f`。
  - `?role=` / `?session_authorization=`：libpq `invalid URI query parameter`，退出码 1。
- 说明：`index.json` 中 `retained_resources_ndjson_sha256` 记录的是追加前的哈希；本轮按 append-only 证据策略追加两行后实际文件哈希已变化，未回写 index（回写会破坏只追加证据），不作为 finding。

## 7. Lead Gate 建议

- **当前不建议**以“ACL 已 fail closed”通过 S1.5 ACL 子集：`S15-ACL-F011` 仍为 PARTIAL，且 README/测试存在被实测推翻的 fail-closed 宣称。
- 可接受的窄修复：按 `S15-ACL-R02-F001` 方案 A 修正 README 与测试注释，并在 ADR-0029“明确不证明”补充 search_path/目标 schema；或按方案 B 增加最小 schema 校验。完成后本子集可转 `PASS WITH MINOR`（R02-F002/R02-F003 为 LOW，可由 Lead 记录接受）。
- 若 Lead 接受“search_path 由运维保证、gate 只证明当前解析对象”的残余风险，应把该接受明确写入 ADR/验收记录，不得保留“同名关系会 fail closed”的错误表述。
