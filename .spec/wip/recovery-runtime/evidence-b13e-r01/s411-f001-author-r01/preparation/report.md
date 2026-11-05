# S4.11 / S411-ICR-R01-F001 原作者修复准备

状态：仅 G0 和本人 D Temp 候选已准备，STOP 等 Main 明确 SOURCE_AUTH/LOCAL/PG START。HIGH OPEN，未修复入仓、未编译、未运行 Go/gofmt/SQL/服务，未自签独立复审通过。

原作者 Lorentz / add-backend-dev，actualID `01a189aa-4e63-7d10-90c1-561fc6d70e8a` 同 ID 续接原 identity-auth-S4.11。固定 `D:/AID/VSCode/Codex/Han`、main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本轮唯一可写根 `D:/Temp/han-b13e-s411-f001-preparation-20261030-r03`。原 r02 packet/cache/data 均保留，本轮没有改其内容。

## 当前事实与失败来源

Confucius 正式报告 `CHANGES REQUIRED / RELEASED` 已全文读取，HIGH `S411-ICR-R01-F001` 针对原 frozen12。Main canonical `s411-review-local-pg-main-release-r01.json` 与 `s411-code-review-main-reception-r01.json` 也已全文读取。后者实际 SHA 为 `DB8C55EEC3782782AA25315859D2C1D86D009D827638946EDFE4B9DD31FA2C0B`，与更正信号一致；其计数是184 manifest载荷、186副本，不能沿用上一消息185。

独立 normal/race 完整 raw、result、parsed、fullprefix overlay 和 preservation 已逐字复制到 finding-evidence。实际重算 raw SHA 等于独立 result，重新解析两 mode 均为 no_cookie_positive PASS，old_runtime_cookie / lost_change_response_old_cookie FAIL；各有两个 `500 / platform.internal_error` 行为 marker。独立两个 Go 实际 exit1 保留；parent Wait/Jobzero/2actualEOF/join 的结论属于所复制独立作业，不冒充本轮作者执行。六个 schema 的完整 SQL 后像随独立 physical JSON 保留；本轮没有连接数据库或核旧 schema。

实际缺陷在 recovery_auth.go 原203–208：真实 Handler.Login 自动把浏览器 Cookie 写入 ExistingSessionToken，Login 密码与当前 authority 已验证后，却对所有非空旧 cookie 强制 RotateMaintenance。已撤销 G/bootstrap cookie 或新 Runtime 下的旧 Runtime cookie 均不符合原严格旋转；用户无需清 cookie 重新登录的原合同因此未实现。上轮 s411Login 不设置 ExistingSessionToken，覆盖遗漏由原作者承担。

## G0 与边界

实际重读完整 Role、add-task-implementation、add-tdd-go、coding/no-hardcode/task-execution/multi-agent/checklist，以及 enabled go-service entry/coding/security/no-hardcode。交付编排、可观测/追踪等先前实际全文按 current SHA 一致复用。读取精确原卡及 B13E、设计 §3.3.1 / §4.2 F001（特别3752/3764–3766）、TP F001/814；requirements/ADR0027 的既有实际读集同 SHA 复用。当前实际消费者沿 Handler.Login、Login.authenticate/checkPasswordResetGate/verifyCredential/prepareSession/storeSession、storeMaintenanceLogin、MaintenanceStore完整源读取。详细范围和 SHA 在 readset；快照或搜索不冒充全文语义阅读。

原12每项当前 SHA 已与 r02 receipt 逐项核对一致；docs6 同 SHA。分支/HEAD通过直接读取 Git元数据核对，没有运行 Git 或写 index。Hooke 正在合法修改 peer installation5，因此本轮不使用旧peer5 SHA断言其稳定、不覆盖，也不把该已知变化报成未知漂移。后续 START 必须使用 Main 新冻结的 peer5 / own12 / docs6 再核，不能从本轮准备或 reviewer Hostclosed推导执行权。

无设计/API/数据格式变更；无新公共接口、依赖或文件所有权。仅候选变动原12中的 recovery_auth.go、recovery_auth_test.go、recovery_auth_postgres_test.go；其余9份 candidate 为原字节。完整 before12/candidate12/diff 便于审查，不能复制整12覆盖共享树。实际写入时只写最终必要文件，并先再次核精确原像；任何非预期 own12/docs漂移、清单外需求或合同gap只停受影响部分。

## 候选最小方案

维护登录成功后的旧 cookie 仅是可选旋转来源。新增私有 maintenanceLoginPrevious，在现有 User→run→Gate 锁与成功密码校验之后、同一 caller-owned Login事务中，查询精确 token/user + maintenance scope + 当前 run/install/runtime/generation + 未撤销 + absolute/idle未过期 + 相同凭据kind/auth_generation + 空credential/lineage，锁住符合条件的行。

仅无cookie或确定 ErrNoRows 时清空 RotatedFromID，经原 CreateMaintenance 重新检查当前真实 executor/authority/Gate并创建全新的维护 Session。旧 token 不被认可、不被关联或改写。SQL/连接/ctx错误立即返回失败，不把错误统称成旧cookie无效。匹配时继续原 RotateMaintenance，保留其第二次精确校验、CSRF轮换、撤旧、rowcount及同事务Audit/commit/rollback；它本身和resolver均没有任何候选改动。

新的签发仍只能由真实 executor 驱动，不能借 static bool、普通 Principal或任意binding授权。普通false+ordinary、bootstrap限制、User/run proof/G+1、审计、密码/ACK原子实现保持原验证责任。选择查询仅采用 Login 相同代际/kind，不使用强改密专有 G→G+1旋转例外。

候选查询对严格旋转资格的投影必须由后续真实P检验；它不取代 session 旋转权威。本轮未发现需要修改公共合同或扩原12的具体gap。该判断是静态准备结论，不能作为候选已通过编译或行为测试的证据。

## 测试草稿与成功证据

原 Reviewer 三叶断言和 fullprefix 原字节单独保留。red-overlay 指向本轮 ownD copy，production overlay为0；green-reviewer-overlay只重新基于候选P前缀追加同一探针后缀，不改原断言。两者均未执行，只供 Main START 后按确切源版本使用；绿色 overlay要求候选生产实现已按授权落入原路径，不能在旧生产源上把其存在当成功。

U在原 ScopeAndPrivateBinding 增加4叶：无cookie不查、精确行选中、确定无行可新签、存储错误不得伪装新签；同时核caller事务未commit/rollback。P在原 PrepareChangeAckLossAndResume 新增 BrowserCookieRelogin 11叶：

- no_cookie_positive；old_runtime_cookie；lost_change_response_old_cookie。
- valid_current_cookie 合法撤旧/rotated_from/新CSRF；revoked_cookie、absolute_expired_cookie、idle_expired_cookie、unknown_cookie 不阻合法新密码签发。
- old_password_rejected、gate_blocks、login_audit_rollback 不发cookie、Session全后像不变。

使用现役 Handler+真实GatedArgon2/Runtime私有phase/唯一Store/同pool/持久abuse/Audit。初始信任根实际Prepare→Login→强改密后再经公开Runtime路径Resume；原Runtime Stop/join沿原夹具保留。逐行保存新旧Session/完整User/run/Audit前后像，新Session精确核scope+当前4binding+G3+false/permanent+CSRF/lineage；旧cookie仍拒protected/ordinary，新cookie亦拒ordinary，错误CSRF拒绝，Resume零改密/零hash。HTTP请求/响应Body均显式关闭，PG/Runtime资源继续使用原真实join夹具。

下一窗口先真实编译并取得旧生产版上的行为Red（复跑保留原独立三叶，保留setup诊断），再最小实现、定向Green和原5组最低U/C/P、普通/race及相邻回归；不写编译不通过的测试去冒充行为Red。本轮测试草稿新增4U/11P是预计命名数量，只有真实逐叶解析才可计入通过。

必须保留每作业argv、raw、results、sourcebeforeafter、真实whole-job parentWait/Jobzero/2EOFjoin及SQL/Runtime/pool资源join；每次单Go parent、CPU1/-p1/-parallel1、独有D cache/TEMP/work、offline/local/modreadonly。只使用 Main 新START具名PG，私读必要identity_dsn，不启旧端口/ambient/business DSN。新unique schemas精确登记、保留不DROP。运行完成源稳定后先release交窗，再ownD封包，随后由原Confucius独立复审。Main全仓7/当前17fmt与原23真实失败仍由Main最终联合冻结版核，不擅自重复全仓矩阵或借旧豁免。

## 本轮实际验证与限制

仅执行文件/SHA/JSON/字节前缀与diff静态核对；候选新/改函数文本行数估计最大46，非AST或Go编译结果。原12再次核对未变。所有候选与计划Go/gofmt/SQL为NOTRUN；新schema0、无新服务/Go/PG进程，也未读PG密钥。预备脚本无subprocess/thread/callback；没有新增需后台join的自有资源。文件读取/准备命令均实际返回结束；不虚构这些文件操作经过Go Jobrunner。

准备期间首次PowerShell读取缺UTF8导致显示乱码，已用UTF8全文补读；误把reviewer overlay元数据定位在prepared-tests目录，实际在packet根，已读取正确文件；一次自检展示用Python默认GBK读取UTF8 JSON产生UnicodeDecodeError，已按UTF8重读。这些仅文件诊断保留在preparation-diagnostics，不是Go/SQL或行为Red。

本包无缓存、密码配置或数据库data；源候选、原始失败raw、readset、diff、方法/权限来源和manifest均在包内。源窗仍属于Hooke；本轮STOP直到Main明确START，不向用户重复申请、不自动获取窗口。HIGH、CodeGate、Task、E2E与人的接纳均未关闭或预签。
