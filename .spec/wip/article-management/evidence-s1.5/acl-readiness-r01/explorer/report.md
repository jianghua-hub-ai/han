# S1.5 实际应用 role ACL：有界只读结论

版本：D:/AID/VSCode/Codex/Han，main，HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca。相关SSOT有修改，源码多为未跟踪；本报告针对当前工作树，以readset.json中完整SHA256封存，不冒充已提交HEAD内容。只本会话Explorer调查，无下级委派。

结论：ACL是与恢复接线及SQL回滚正交的验证义务，但当前尚不能直接独立实施/运行。实际应用数据库身份、安全注入/物理载体与执行窗口未确定；现有单有效身份自建schema/迁移fixture不提供合格应用ACL证明。SQL子集最新已交付，仅按Tasks接收记录消费，未复跑/重签。

1. 合同确切要求
- test-plan.md:483,549-551,617：实际应用数据库身份对audit_events的有效UPDATE/DELETE权限必须不存在，并以确有目标行的UPDATE及DELETE得到权限拒绝证据。候选名TestArticleAuditApplicationACLContract仍未实施，物理路径/安全注入位置unknown。
- 不是users.role的member/admin、Principal或Event.ActorRole；仓内未规定数据库role的具体名字，不能发明han_app等名称。
- 实际Append是caller-owned事务INSERT（internal/audit/store.go:49-103）；Get为SELECT（106-155）。未来测试应保留合成合法行与必要INSERT/读取正控，排除错误对象/无schema访问/全禁用造成的假拒绝；这属于建议的证明方式，不新增合同要求。权限拒绝应可定位为42501，而不是trigger的55000；还须证明错误来自目标表UPDATE/DELETE权限而非schema USAGE/SELECT等前置缺权。

2. 原schema/migrations具备什么
- migrations/0001_shared.sql:14-34定义Audit表；42-60定义BEFORE UPDATE OR DELETE FOR EACH STATEMENT trigger，一律RAISE 55000。它是不可变防护，不是角色ACL；语句级trigger连零命中也能拒绝。
- 对0001到0009做了全文件ACL词法扫描，未发现role创建/切换、GRANT/REVOKE、OWNER TO、DEFAULT PRIVILEGES或SECURITY DEFINER/RLS配置。其余迁移只按ACL相关匹配读取，未全读无关业务正文。不能把仓内缺失推成实际PG环境没有外部配置。
- 所以可测trigger，但实际role ACL是否已在环境中配置仍unknown。若已有符合应用身份的非owner/non-superuser role及所需schema USAGE/最小对象权限，未必需要新CREATE ROLE/GRANT；若无，需要环境方另行明确批准与落实安全权限配置，不能从一般继续授权推断，更不能本轮执行。

3. 单User DSN/新schema能否完成真实有效验证
- 当前实现不能。test/integration/article/audit_insert_rollback_test.go:105-189仅一个DSN/一个pool：同一有效身份CREATE SCHEMA、完整执行九迁移并做业务SQL；current_user仅检查非空并记录。没有session_user/owner/rolsuper/角色成员关系/有效表或列权限核验。
- internal/audit/store_integration_test.go:24-79也同DSN建schema/迁移，UPDATE/DELETE仅检查err!=nil；不能区分42501与55000。177-208还注册DROP SCHEMA CASCADE cleanup，本轮不可运行或借用为保留型ACL载体。
- contract internal_ports_test.go:39-54为scriptTx，447-473为auditSpy；只证组件协作。现有article contract/integration及internal/audit扫描未见ACL top或有效权限探针。
- 推断：按当前单有效身份执行未指定owner的CREATE TABLE，建表者与测试者重合；新schema隔离名字和数据，不会自动生成应用/迁移角色分离。即便观察到55000，也不能证明无UPDATE/DELETE权限。不能拿owner/superuser身份作为本合同应用ACL证据。
- 边界：一个登录DSN理论上可通过已经存在且获准的角色切换成为另一有效身份，但现有代码没有此路径；实际membership/默认权限/role参数/可用安全入口均unknown。本报告既不声明这种环境存在，也不批准SET ROLE或添加入口。已存在的他人owner对象可否提供合格fixture亦unknown。

4. 当前合法制品与缺失事实
本轮已完成：本短报告、readset.json及其哈希清单。没有新测试源码/Temp可执行脚本，不写Tasks/Plan/design/Review/Manifest，不创建schema、不接PG、不读secret/env、不运行测试、不清理。
精确缺失：
(a) 实际应用数据库session_user/current_user及其与真实应用配置的绑定；对象owner、superuser与直接/继承/可切换角色关系。
(b) 获准目标schema/table以及是否真实代表应用运行ACL；schema USAGE、Audit INSERT/必要读取、DELETE、表级UPDATE及所有列UPDATE的有效权限，含PUBLIC/角色继承/外部默认权限来源。只查表UPDATE不足以排除列级UPDATE。
(c) 是否已有合法保留的合成目标行；若需新schema，谁在不更改现有权限的前提下拥有/迁移对象、应用role如何访问；外部默认权限不能由旧schema推到新schema。
(d) 应用身份安全注入位置/实际测试物理载体、精确写集及运行窗口；r19 SQL文件授权不自动扩为ACL授权。

冲突：test-plan §4.7.2及末尾仍写planned/not-created；Tasks:1227-1228明确这是旧点时规划，SQL已接收。以最新Tasks判SQL状态；ACL pending在两处一致。没有运行时ACL结果，不能据此声称产品ACL有漏洞或已有正确配置。
风险/不可冒PASS：23505回滚、55000 trigger、mock、零行、无方法、owner/superuser、缺环境返回/Skip均不能关闭应用ACL；也不关闭完整S1.5、恢复/E2E等门禁。

唯一建议下一步：Lead先接收一份由数据库环境责任方提供的无secret“现有应用role/目标对象权限事实回执”，至少覆盖(a)-(c)，并明确只读身份/目录核验可用的安全注入与授权范围；此步不改role/GRANT、不创建对象。若已有可用应用身份与fixture，再由Lead确定ACL唯一载体及单独实施/运行窗口；否则交回需要的最小权限配置决定，不自动新建角色。
停止条件：实际应用身份、有效权限来源或合法fixture任一无法证实；只能以owner/superuser/55000代证；需要新role/GRANT、额外secret、PG连接或新增写窗时即停，返回Lead。当前已满足停止条件：角色/载体/权限事实unknown；调查到此STOP，不等待或实施。

关键实读文件哈希（其余及读取粒度见readset.json）：
- .add/project/knowledge/design-workspace/article-management/test-plan.md SHA256 DD7D603CC1CDE2681B0CE6CE168DFD36C77CC3E07A42511609A09C5EA316CEFA
- .add/project/knowledge/tasks/tasks.md SHA256 EC9563D6B3A9921DE217ECE92D0F1B421644C4ED3FCF570D8B768B10A128E183
- internal/audit/store.go SHA256 091219E0FD2298C43474A7152C8300EB5CEFD5565C1F08D78DDD1E1387C5C24F
- internal/audit/store_integration_test.go SHA256 4A3CC84CE6B620724692D696B705D74BC39C8E1EB28F044EDD136E166FF6138C
- migrations/0001_shared.sql SHA256 CEE72F1470B49185B3B33C7655F792ADFFCBC1E030AF6669E0902775104B7A14
- test/integration/article/audit_insert_rollback_test.go SHA256 468964647524860CEEF0167EC56A9C0033FBB40CFF340A5021FBCE66F2C89CD5

当前版本补充：实读test-plan文件头1-32为r20（design来源r22），其26-27明确前版r19/hash 88296F...。本调查使用r20保留的§4.7.2与ACL条款；不把Tasks登记时r19 hash当当前整文件hash，也未展开r20恢复方向。readset封存后31个源文件复核零hash变化，只证明此短封存间隔一致，不证明无其他writer或已释放窗口。

| 证据结论 | 实读来源 | 状态/置信度 |
| --- | --- | --- |
| SQL子集接收，ACL仍unknown | tasks.md:1227-1228,1265 | 最新台账事实/高；本轮不重签运行PASS |
| 实际应用role、有效权限和命中行拒绝必需 | test-plan.md:483,551,617 | 当前r20合同保留/高 |
| Audit有55000语句trigger，没有内置role ACL配置 | migrations/0001_shared.sql:14,42,57；九迁移ACL全文件扫描 | 仓内静态事实/高；环境配置unknown |
| 新SQL fixture同身份建表和执行，不核应用ACL | audit_insert_rollback_test.go:105,132,161,176 | 静态事实/高；非运行身份事实 |
| 旧Audit测试只验err，且有自动DROP | store_integration_test.go:24,75,78,177 | 静态事实/高；本轮不可运行 |
| 现fixture不足以独立完成ACL | 上述合同+代码双证 | 推断/高；不作任何环境已配置/未配置结论 |
