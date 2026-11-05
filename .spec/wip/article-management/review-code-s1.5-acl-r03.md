# S1.5 ACL 非作者独立复审 R03（add-reviewer，当前工作树 revision）

- 评审对象：`article-management-S1.5` 卡点「路径 B」的 **ACL 门禁子集**——双 DSN 身份分离、启动迁移后 REVOKE、应用池 ACL gate、search_path 同名关系判定、`deploy/db/*` 资产与文档；对象为**当前工作树 revision**，不是 r02 快照。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（实测）。
- 评审性质：`add-code-review` 的 revision 复审路径；评审者非作者（作者 Herschel），只读；唯一写入为本报告。
- 范围声明：**只评 ACL 门禁子集，不对 S1.5 整卡下 PASS，不声称 Lead Gate。** audit SQL、三投影及其它卡点子项不在本次结论内。
- 修订说明：本文件是同路径 r03 报告的正式修订（v2）。原版 SHA256 为 `7A236731FEED830D60503FB17A1F5224DE92A737B605DAD17E57BE0FC5C45D31`（76 行），结论同为 PASS WITH MINOR，但其基线取自 `deploy/db/README.md` 措辞更正前（`9C4F30C6…`），且未覆盖 ADR-0029 r04 errata。本版更新基线、补入本轮亲自实跑与 r04 文档-代码一致性核验，结论不变。因原版哈希被 `framework.md:911` 引用，交接事项见 §9。

## 1. 结论

**PASS WITH MINOR**

- R01 findings `S15-ACL-F001`–`F011` 在当前 revision **全部 CLOSED**（`F003` 附“原证据目录本轮未逐字复核”、`F010` 附“无 scoped git diff”残余说明）。
- R02 新增项 `S15-ACL-R02-F001`（同名关系歧义 fail closed 宣称不成立）、`R02-F002`、`R02-F003` **全部 CLOSED**；其中 `R02-F001` 由**代码修复**关闭（`current_schemas(false)` 同名关系计数 + 独立错误类别 + 真实 PG 负控），不是靠改文档措辞。
- 新增 3 条 LOW finding（`S15-ACL-R03-F001`–`F003`），全部属证据卫生与文档精度，**无 HIGH/MEDIUM，无阻断整卡关闭的新发现**。
- ACL 门禁子集**可核收**（条件见 §9：Lead 处置证据明文口令与证据索引漂移，并同步 ADR 引用哈希）。

## 2. 评审基线（本轮实测）

### 2.1 文件哈希与字节数

| 文件 | 实测 SHA256 | 字节 | 与任务冻结输入 |
|---|---|---:|---|
| `cmd/han-server/append_only_acl.go` | `A916412CBCDD7279A61B26B56993FDAA320AED940F2BA53A9727E09A4E68CFF5` | 17027 | 一致 |
| `cmd/han-server/append_only_acl_test.go` | `6555F88A2FB6E24317236EAA6710CE1464061FA80FD409782673773020F95739` | 18128 | 一致 |
| `cmd/han-server/append_only_acl_integration_test.go` | `A8CF9C9CD02828FF1A426156526853D51AFE94F6662DBBCD6934F41A79E533E2` | 20323 | 一致 |
| `cmd/han-server/database_config.go` | `16EEB0E97652CBED16628BCD3B64F6D2D99B5E5C2EF77F6C65DC90DDD35405DA` | 7541 | 一致 |
| `cmd/han-server/main.go` | `1D47CC94E415E80C69DEEBE237EBD085AF05044902B12E07B2A04B0822028187` | 171448 | 一致 |
| `deploy/db/provision.sql` | `5252556099A335A480041A62ED30C7618384A5B4D038BFA29356423C446B8C23` | 2333 | 一致（未被本轮修改） |
| `deploy/db/grant-application.sql` | `FC0D229818D48407EF4CCBF97D0860E08E781FF8EF102201C551560741AF0AE8` | 1229 | 一致（未被本轮修改） |
| `deploy/db/README.md` | `9A47DD10CA1699757658728F7BE7F04FD221FFAD3F025DB0AA699E8F2F29F0E9` | 7772 / 137 行 | **漂移**（任务给 `9C4F30C6…`） |
| `acl-r03/index.json` | `E80EFD986653DD91519F64D4A9617C9299B21F50E98C16D09498D47253597D31` | 21492 / 421 行 | **漂移**（任务给 `0ECAACE5…`） |
| `.add/project/knowledge/decisions/framework.md` | `04E56C991510A5243F26E3EAB85555E081CC4DA8EBF5F44E1E25B080E9F4AAFA` | 126444 / 911 行 | **漂移**（任务给 `4954B717…`；已升 adr-0029-r04） |
| `.add/project/knowledge/decisions/README.md` | `35A5623E7E3CFCC140DA0DE9C8B218210C0E34A850A26B5CDD450B81DA40CC63` | 18948 | **漂移**（任务给 `AC5F2541…`） |
| R01 报告 | `8210B2A0F3B05D08509553E9B080EA2DC5D170B64240441021D3D5366F2AA583` | 20015 | 一致 |
| R02 报告 | `E1B93E05D005521C729FEBDC69CE663FC64D5923D5B870F5596EA7AEDC517C67` | 19960 | 一致 |
| `acl-r03/retained-resources.ndjson`（撰写时） | `642C4B8524DC35D691F9DEE1C63F9E90D04D8129B7C72B769B774298BED0BFAC` | 14668 / 56 行 | 无冻结值；见 `R03-F002` |

### 2.2 漂移判定（未触发 STOP 的理由）

- 8 个判定对象文件（代码/SQL）哈希与冻结输入**完全一致**，评审对象本身稳定。
- 4 项漂移均为文档/证据：`README.md` 漂移对应 run-06/07 之后的**措辞更正**（`index.json:157-159` 自记 `after_sha256=9A47DD10…`，与实测一致；`index.json:205/213/407` 说明 Go 源码哈希未变）；`index.json` 漂移是并发追加（见 `R03-F002`）；ADR 与 decisions 索引漂移是 r04 errata 落地。
- 因此按“当前工作树 revision”继续评审，并把漂移作为事实登记，不作 PASS 依据。

### 2.3 Git 事实（只读）

- `git rev-parse HEAD` → `6da8abb…`；工作树有大量既有未提交改动（`git status --porcelain` 287 行），本轮未触碰。
- `git ls-files -- cmd/han-server/append_only_acl.go` 计数 0、`git ls-files -- deploy/db` 计数 0：**代码与部署资产在 HEAD 中未跟踪**，不存在 scoped git diff，`main.go` 的“只替换 migrate 并移除 import”只能静态观察（见 `F010`）。
- 本轮未执行任何 Git 写操作（无 add/commit/push/checkout/switch/stash/clean/restore）。

## 3. 实跑命令与退出码（本轮亲自执行）

| # | 命令 | 退出码 | 结果摘要 |
|---|---|---|---|
| 1 | `go build ./...` | 0 | 通过 |
| 2 | `gofmt -l`（4 个改动 Go 文件） | 0 | 输出为空 |
| 3 | `go vet ./cmd/han-server` | 0 | 无告警 |
| 4 | `go vet -tags=integration ./cmd/han-server` | 0 | R01-F003 不复现 |
| 5 | `go test ./cmd/han-server -count=1` | 0 | `ok han/cmd/han-server 44.389s` |
| 6 | 真实 PG：`go test -tags=integration ./cmd/han-server -run '^TestAppendOnlyApplicationACL$' -count=1 -v`（`HAN_TEST_POSTGRES_DSN` 指向本机 `127.0.0.1:5433/postgres`，口令在报告中不写出） | 0 | `PASS`，父测试 2.83s，`ok han/cmd/han-server 2.964s`，**12/12 子测试通过** |
| 7 | psql 只读探针 A：`SET search_path = <两历史同名 schema>, pg_catalog` 后统计 | 0 | `cs=…,pg_catalog\|visible=1\|on_path=2`（复现歧义形态） |
| 8 | psql 只读探针 B：`SET search_path = public, public, pg_catalog` | 0 | `public,pg_catalog`（重复项去重，无误报） |

同一会话更早、且代码哈希未变时另执行（结果沿用）：集成 + `-race` → 退出码 0、`ok 3.288s`、无 DATA RACE；psql 探针覆盖 `current_schemas(true/false)` 下 `pg_catalog`/`pg_temp` 隐式成员、临时表可见性、无 USAGE 时被过滤（`SET ROLE` 到应用角色后 `current_schemas(false)={}`）、重复 schema 去重。

第 6 项子测试实测清单：`fresh_install_default_privileges_gate_passed`、`schema_owner_is_migrator`、`non_superuser_login_positive`、`insert_allowed`、`migration_ledger_denied`(42501)、`mutation_denied`(6 条 42501：两表 UPDATE/DELETE/TRUNCATE)、`schema_create_denied`(42501)、`superuser_session_rejected`、`superuser_set_role_rejected`、`owner_identity_rejected`、`column_update_rejected`、`ambiguous_same_name_schema_rejected`。

## 4. Finding 逐条复核

### 4.1 R01 `S15-ACL-F001`–`F011`

| 编号 | 原级别 | R03 判定 | 关键证据（本轮实测行号） | 判断理由 |
|---|---|---|---|---|
| `S15-ACL-F001` | HIGH | **CLOSED** | `append_only_acl.go:162-167` 拒绝 `current_user != session_user`；`:168-170` 双身份任一 superuser 均拒绝；`:194-199` + `:273` 拒绝表 owner / `pg_has_role(...,'MEMBER')`；`database_config.go:57` 正则、`:91-93` URL 入口、`:109-116` keyword、`:156-183` 百分号解码+空白裁剪+大小写折叠；`main.go:3935-3943` 在 parse 阶段校验，`:480` 才建应用池 | 三条绕过路径（superuser+`SET ROLE`、owner、DSN `options` 内 `role=`）均被实际阻断，且 DSN 级拒绝先于应用池 connect/ping；会话级身份检查在 listener 与业务装配之前。集成负控 `superuser_session_rejected`/`superuser_set_role_rejected`/`owner_identity_rejected` 与真实非超管正控 `non_superuser_login_positive` 本轮均 PASS |
| `S15-ACL-F002` | HIGH | **CLOSED** | `append_only_acl.go:113` 迁移后 `restrictApplicationMutations`；`:119` 显式 `migrationPool.Close()`；`:120` 才 gate；`:324-361` 由 `migrationPool.Exec` 对两表 `REVOKE UPDATE, DELETE, TRUNCATE`、对 `schema_migrations` `REVOKE ALL`；`:28` 常量与 `grant-application.sql:20-21` 同步；`provision.sql:43-49` 只设默认权限 | REVOKE 由迁移身份执行且早于 gate；fresh install 不再依赖 `grant-application.sql`——集成 `:124-131` 无手工 grant 直接走生产迁移，日志 `default_privileges_only=true manual_grant=false`。文档顺序 `README.md:43-53` 与实现一致 |
| `S15-ACL-F003` | HIGH | **CLOSED** | `go vet -tags=integration ./cmd/han-server` → 0；PG 集成可编译、可运行并 PASS | R01 的未使用 import/未定义符号不复现。原 fake-world 子测试移出到 `.spec/wip/recovery-runtime/evidence-b13h-r01/s419-world-hold-r01/` 的**内容本轮未逐字复核（unknown）**，但判定所依赖的可复现判据（vet=0、集成可跑）成立 |
| `S15-ACL-F004` | MEDIUM | **CLOSED** | `provision.sql:37-38`（`CREATE SCHEMA IF NOT EXISTS … AUTHORIZATION` + `ALTER SCHEMA … OWNER TO`）；`README.md:108-114` 披露 public schema owner 变更风险；集成 `schema_owner_is_migrator` PASS（`owner_superuser=false`） | schema 归属迁移身份已由脚本与测试双证；public 使用风险明确交运维接受 |
| `S15-ACL-F005` | MEDIUM | **CLOSED** | `append_only_acl.go:486-499` `databaseEndpointsOf` 遍历 primary + 全部 `Fallbacks`；`:510-541` 角色名 + `(host,port,database,user)` 集合比较；单测 `append_only_acl_test.go:400-` 含 `same_role_on_fallback_is_rejected`、跨库/跨 host 负控 | 原“只比 primary 四元组”的缺口已消除；fallback 端点被纳入比较，且只读配置结构、不建连（无连接泄漏） |
| `S15-ACL-F006` | MEDIUM | **CLOSED** | 集成 `:62-73` 创建 `LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE` 真实角色；`:105-110` 应用/迁移各自 DSN；`:282` `SET ROLE` 仅作负控；superuser 仅用于 provisioning preflight `:490-507` | 权限矩阵在真实非超管 LOGIN 会话上执行，不再是 `NOLOGIN`+`SET ROLE` 模拟 |
| `S15-ACL-F007` | MEDIUM | **CLOSED** | `append_only_acl.go:283-286` `has_any_column_privilege` 检列级 INSERT/UPDATE；`:287-298` `aclexplode(attacl)` 检列级 DELETE 条目；集成 `column_update_rejected` 本轮 PASS | 列级 UPDATE 负控成立；PG 18.6 不支持列级 DELETE 权限的兼容处理有 README:101-104 说明，且 `aclexplode` 路径在 18.6 实跑通过 |
| `S15-ACL-F008` | LOW | **CLOSED** | `main.go:3935-3943` 在 `parseDatabase` 内调用 `validateProductionMigrationConfiguration`（`append_only_acl.go:424-430` 纯配置校验，不建连），早于 `main.go:480` connect / `:490` ping；`:119` 关闭迁移池后 `:120` 才 gate | 非法迁移配置在应用池创建前 fail closed；迁移池寿命收窄到 gate 之前 |
| `S15-ACL-F009` | LOW | **CLOSED** | 集成 `:56-60` 注释与 `:60` 位置（superuser preflight 之后、建角色之前登记 `planned`）；`:100` 登记 `created`；`:335/:354` decoy 同法 | 资源登记不再先于 preflight；`planned`/`created` 两态可区分 |
| `S15-ACL-F010` | MEDIUM | **CLOSED**（附残余） | 本轮逐文件实测 SHA256 与冻结输入一致（§2.1）；`index.json:157-159` 记录 README `after_sha256=9A47DD10…` 与实测一致 | 冻结方式已从“任务声明”改为“8 文件哈希 + 证据索引”；残余：`cmd/`、`deploy/` 在 HEAD 未跟踪，仍**无法**产出 scoped git diff，不能做历史差异反证（只能静态确认 `main.go` 仅将 migrate 接到 `productionMigration` 且未引入新行为） |
| `S15-ACL-F011` | MEDIUM | **CLOSED** | `append_only_acl.go:188-193` 同名关系计数 ≠1 → `errAppendOnlyTableAmbiguous`（`:42-44` 独立错误类别，与 `errAppendOnlyTableUnresolved` 可区分）；`:261-269` 用 `pg_catalog.current_schemas(false)` 统计；`:383-417` resolver 同构；集成 `:359-367` 断言错误类别且 `same-name relations on effective search_path=2`；`README.md:89-91/115-124`、ADR-0029 `:898` §10 同步 | 不再是“文档化即关闭”：代码已 fail closed，且有真实 PG 负控与只读探针（§3 第 7 项 `visible=1, on_path=2`）双证。残余：`internal/migration` 仍未固定 `search_path`（作者按写集边界不改），改为“运维前置 + gate fail closed + 明确不证明目标 schema 身份”的已登记边界，见 §7 |

### 4.2 R02 新增项

| 编号 | 原级别 | R03 判定 | 关键证据 | 判断理由 |
|---|---|---|---|---|
| `S15-ACL-R02-F001` | MEDIUM | **CLOSED** | 同 `S15-ACL-F011`；新增子测试 `ambiguous_same_name_schema_rejected`（`:333-369`）本轮 PASS | 原“同名关系必然 fail closed 的宣称被实测推翻”已被代码修复，宣称现在为真 |
| `S15-ACL-R02-F002` | LOW | **CLOSED** | `database_config.go:156-183`（URL 顶层 key 百分号解码 + `TrimSpace` + `EqualFold`，`options` 值同样解码后匹配）、`:109-116`（keyword 侧）、`:57` 正则含 `-c`/`--` 与空白容忍；负控 `append_only_acl_test.go:302/307/312/329/334/343/348`，正控 `:316-326/338/352-357` | 编码、大小写、空白变体均被拒；`application_name`、`statement_timeout`、`options='-c search_path=role=foo'` 不被误拒 |
| `S15-ACL-R02-F003` | LOW | **CLOSED** | `framework.md:893` §5 已改为“默认权限=业务表 DML（SELECT/INSERT/UPDATE/DELETE）+ 序列 USAGE/SELECT，两表与 ledger 由启动迁移在关闭迁移池前 REVOKE 收窄”；`:895` §7 披露建表→REVOKE 短窗口；`:911` r04 errata 记录更正来源；`decisions/README.md:37/104` 同步；与 `provision.sql:46-49`、`append_only_acl.go:113-120` 实际顺序一致 | 文档-代码一致性成立（本卡只判一致，不要求真实部署证据） |

### 4.3 新发现

| 编号 | 级别 | 位置/证据 | 影响 | 建议 |
|---|---|---|---|---|
| `S15-ACL-R03-F001` | LOW | `evidence-s1.5/acl-r03/index.json:201`、`:209`、`:231` 与 `run-06.log:1`、`run-07.log:1` 含明文数据库口令（即 `<用户名>:<口令>@主机:5433` 的连接串形态）；本轮未复写口令，报告中一律 `<redacted>` | 证据随 `.spec/wip` 交接可能扩散；与 AGENTS“禁止提交凭据/密钥”的精神不符（当前为本机开发凭据，风险有限）。**不属 ACL 逻辑缺陷** | 由 Lead 决定并执行：脱敏重写证据（先记原哈希）或明确记录“本机一次性开发凭据，接受风险”。评审方不修改证据 |
| `S15-ACL-R03-F002` | LOW | `index.json:312-319` 记 `entry_count=36`、`registry_sha256_final=940609FC…`、`drop_count=0`；撰写时实测 `retained-resources.ndjson` = **56 行 / `642C4B85…`**（本会话内先为 52 行 / `E26CE649…`，再因本轮集成测试追加 4 行） | 索引与实际不一致，后续无法用 `index.json` 校验证据完整；追加来源含并发 Lead gate 与本评审，非删除导致 | 建议以 append/errata 方式在 `index.json` 追加 `entry_count_after_review`、`registry_sha256_after_review` 与时点，不改写既有字段 |
| `S15-ACL-R03-F003` | LOW | `database_config.go:57` 的 DSN 正则为“尽力而为”的黑名单：语言层等价写法（如 `options` 内对参数名做反斜杠转义由后端还原）可能不被正则捕获；权威兜底是 gate 的 `current_user != session_user`（`append_only_acl.go:162-167`） | 不是安全缺口（启动仍 fail closed），但 `README.md:17-21` 仅描述解析期拒绝，未说明正则非完备、最终判定在 gate | 建议在 `README.md:17-21` 补一句“解析期正则为快速拒绝，最终权威判定是启动 gate 的稳定身份检查”；属措辞级，非阻断 |

新发现合计 **3 条，全部 LOW，0 条 HIGH/MEDIUM，0 条阻断整卡关闭**。

## 5. 重点判断的独立验证记录

- **a) 同名歧义 fail closed 与错误类别可区分**：`visible != 1` → `errAppendOnlyTableUnresolved`（`append_only_acl.go:182-187`），随后 `onSearchPath != 1` → `errAppendOnlyTableAmbiguous`（`:188-193`）；两者是不同 sentinel（`:39-44`），`errors.Is` 可分辨，单测 `:150-166` 与集成 `:363-365` 各自断言。口径缺陷检查：`current_schemas(false)` 已含权限过滤（探针：应用角色对 `pg_toast` 无 USAGE 时返回 `{}`），重复 schema 会去重（探针 B），隐式 `pg_catalog`/`pg_temp` 不计入（探针：无临时表时 `(true)={pg_catalog,public}`、`(false)={public}`）。`pg_catalog` 隐式优先不会造成“同名表误判为唯一”而放行：若目标关系只在 `pg_catalog` 可见，`on_search_path` 计数为 0 → 仍 fail closed。
- **b) DSN 拒绝的可绕过性与误拒**：URL 顶层 key 与 `options` 值都先 `QueryUnescape` 再折叠大小写、裁剪空白（`database_config.go:156-183`），覆盖 `%72ole`、`Session_Authorization`、`-c role=`、`--role=`；keyword 侧按 key 精确/折叠比较（`:109-116`）。正控覆盖 `application_name`、`statement_timeout`、`search_path=role=foo`，无误拒。残余见 `R03-F003`。
- **c) ADR-0029 r04 与代码/脚本一致性**：§5（`:893`）默认权限口径 = `provision.sql:46-49` 实际 GRANT；“启动迁移在关闭迁移池前 REVOKE” = `append_only_acl.go:113-120` + `:324-361` 实际顺序；§7（`:895`）短窗口披露与 `provision.sql` 默认权限 + 启动 REVOKE 的时序吻合；§10（`:898`）与 gate 实现一致。`decisions/README.md:37/104` 同步。**一致**。
- **d) F011 可 CLOSED**：是。代码修复 + 真实 PG 负控 + 只读探针三方一致；残余（迁移运行器不固定 `search_path`、不证明目标 schema 身份）已被 ADR §10 与 README:115-124 明确列为不证明边界，属已登记边界而非隐藏缺口。

## 6. 本次证明

- 8 个判定对象文件在当前工作树的哈希与冻结输入一致；本轮未修改任何源码/SQL/测试/证据/任务卡。
- 启动门禁在 `main.go` 的实际调用链上：parse 期配置校验（`main.go:3935-3943`）→ connect/ping（`:480-492`）→ 迁移身份迁移 + REVOKE + 关闭迁移池（`append_only_acl.go:96-119`）→ 应用池 gate（`:120`）→ bootstrap（`main.go:496`）→ 业务装配；listener 之前无旁路。
- 真实 PG 18.6：应用身份 INSERT 正控 + UPDATE/DELETE/TRUNCATE 六条负控（42501）、ledger 拒绝、schema CREATE 拒绝、superuser/`SET ROLE`/owner/列级 UPDATE 四类身份绕过负控、同名 schema 歧义负控，12/12 PASS。
- 同名关系歧义在 `current_schemas(false)` 下按“计数恰为 1”判定，且与“表不存在”分属不同错误类别；本轮 psql 探针复现 `visible=1 / on_path=2`。
- `deploy/db/provision.sql` 已无 dollar-quote 内变量替换：角色创建用 `SELECT format(...)` + `\gexec`（`:17-29`），标识符/字面量分别用 `:"var"`/`:'var'`，可执行性缺陷已消除（r01 探针日志 `provision-probe-r01.log` 记录事务内执行后 ROLLBACK 成功）。

## 7. 本次不证明

- **整卡关闭**：S1.5 含 audit SQL 等其它子项，本报告不覆盖，也不声称 Lead Gate 或人的签收。
- **迁移 `search_path` 被固定**：`internal/migration` 未改；服务不设置 `search_path`，gate 只证明“生效 search_path 内同名关系唯一且可见”，**不证明**该关系就是 `provision.sql` 目标 schema（ADR-0029 §10、`README.md:115-124` 已明示）。
- **真实部署**：`deploy/db/*.sql` 未在真实部署集群执行；public schema owner 变更风险、凭据分发、角色轮换、备份/恢复、CI、RPO/RTO 均不证明。
- **两表之外的最小权限**：只证明 `audit_events` / `processing_gate_records` 两表与 `schema_migrations` 的收窄，不证明其余对象为最小权限。
- **历史同名 schema**：本机 5632 行历史 schema 未清理也未被本评审清理；gate 只在“应用生效 search_path”口径内判断，历史 schema 若被运维放入 search_path 会导致启动失败（fail closed），这是设计意图。
- **版本差异**：`cmd/`、`deploy/` 在 HEAD 未跟踪，无 scoped git diff，不能反证“`main.go` 只把 migrate 换成 `productionMigration` 并移除 import”之外无其它改动；本轮仅静态确认调用点与行为。
- **临时表形态**：显式把 `pg_temp` 放进 `search_path` 时会被计入同名计数；仅隐式 `pg_temp` 时不计入——该形态下 `on_search_path` 计数为 0，gate 仍 fail closed（偏严不偏松），本轮未单列 finding。
- **`GOOS`/多平台、`-race` 以外的并发压力**：不证明。

## 8. 保留资源（本轮新建，禁止清理）

- 本轮集成测试：schema `han_s15_acl_2a54e23aa6ce13d0`、`han_s15_acl_2a54e23aa6ce13d0_decoy`；角色 `han_s15_mig_2a54e23aa6ce13d0`、`han_s15_app_2a54e23aa6ce13d0`（已由测试追加 `planned`+`created` 到 `acl-r03/retained-resources.ndjson`）。
- 本会话更早（代码哈希未变）：`han_s15_acl_015cd0ad20d901cc`(+`_decoy`)/`han_s15_mig_015cd0ad20d901cc`/`han_s15_app_015cd0ad20d901cc`；`han_s15_acl_3b2976abb200a744`(+`_decoy`)/`han_s15_mig_3b2976abb200a744`/`han_s15_app_3b2976abb200a744`。
- 只读探针引用（不新建）：历史 schema `han_s15_acl_07eb389eccb77ab3`、`han_s15_acl_2dd15584ab2d5af9`，`SET ROLE` 目标 `han_s15_app_850b2041f38db4e5`。
- 本评审**未 DROP/清理**任何 schema、角色或证据；新增仅为本轮测试的 append-only 登记行。

## 9. 是否可核收（专业判断）与交接

- **ACL 门禁子集：可核收。** 剩余代码级 finding 为 0；开放项只有 3 条 LOW（证据明文口令、证据索引漂移、DSN 正则的文档精度），均可由 Lead 以“记录/小改”处置，不构成对 ACL 逻辑的阻断。
- **整卡关闭与 Lead Gate 由 Lead 判定**，本报告不下此结论。
- 交接必办（评审方不可执行）：
  1. `framework.md:911` 的 r04 errata 引用了本文件**修订前**的 SHA256 `7A236731…`；本报告已覆盖写入，该引用将失效。请 Lead 将引用更新为**本修订版 SHA256（见交付回报）**，或在 errata 中注明“指向修订前快照”。`decisions/README.md` 只按 finding 编号引用，无需改。
  2. `R03-F001` 明文口令：决定脱敏或明确接受风险。
  3. `R03-F002` 证据索引：以 append/errata 方式补记 `entry_count_after_review` 与哈希。
  4. `R03-F003`：可选一句话 README 补充，不做也不阻断。

## 10. 复核命令与证据引用

- 命令见 §3（本轮退出码 0/0/0/0/0/0/0/0）。
- 代码：`cmd/han-server/append_only_acl.go`（`:28`、`:39-44`、`:68-121`、`:150-205`、`:250-318`、`:324-361`、`:364-419`、`:421-457`、`:486-541`）、`cmd/han-server/database_config.go`（`:57`、`:69-118`、`:145-183`）、`cmd/han-server/main.go`（`:480-498`、`:3935-3953`）。
- 测试：`append_only_acl_test.go:136-166`、`:300-370`、`:400-`；`append_only_acl_integration_test.go:30-45`、`:56-131`、`:136-369`、`:448-485`、`:490-508`。
- 部署资产：`deploy/db/provision.sql:15-49`、`deploy/db/grant-application.sql:15-21`、`deploy/db/README.md:17-21/23-60/84-104/106-137`。
- 决策：`framework.md:865`（r04）、`:889-898`（§1–§10）、`:910-911`（r03/r04 引用记录）；`decisions/README.md:37/54/104`。
- 证据：`evidence-s1.5/acl-r03/index.json`（`E80EFD98…`）、`run-06.log`/`run-07.log`（12/12 PASS）、`retained-resources.ndjson`（撰写时 56 行 `642C4B85…`）、`evidence-s1.5/acl-r01/provision-probe-r01.log`。