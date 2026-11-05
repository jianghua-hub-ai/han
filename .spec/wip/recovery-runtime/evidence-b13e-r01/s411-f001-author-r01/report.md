# S4.11 / S411-ICR-R01-F001 原作者实际修复交付

原12内必要3文件已实际修复并完成本卡局部验证，SOURCE_AUTH/soleLOCAL/identityPG 已 explicit release。当前作者状态 STOP；后续仅本人 D Temp 封包。独立 HIGH 尚待原 Reviewer Confucius 复审，作者不关闭 finding、CodeGate、Task 或人的接纳。Hooke 安装 HIGH 也不是本作者的签发范围。

Lorentz / add-backend-dev，actualID `01a189aa-4e63-7d10-90c1-561fc6d70e8a`，同ID原任务续接。固定 `D:/AID/VSCode/Codex/Han` / main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。实际结果根为 `D:/Temp/han-b13e-s411-f001-author-20261030-r04/packet`，本轮全部 raw 位于其 raw 子目录。release.json 先于 formal packet 写出，不让封包占下一运行窗口。

## 修复与原合同

真实 Handler.Login 会将浏览器旧cookie传到 ExistingSessionToken。原维护登录在新密码和当前authority已验证后，仍无条件要求旋转旧cookie，导致丢强改密响应或重建Runtime后的合法重新登录返回500。此次将成功密码认证后的签发与可选旧token旋转资格分开。

只修改 recovery_auth.go、recovery_auth_test.go、recovery_auth_postgres_test.go；其余原9源字节不变。私有 maintenanceLoginPrevious 在既有User→run→Gate事务锁之后，按摘要/user/maintenance scope/当前run-install-runtime-generation、未撤销、absolute和idle有效、同凭据kind/auth_generation、空credential/lineage选取并锁住旧行。精确符合才继续原RotateMaintenance。无cookie或确定ErrNoRows时不关联/不改变旧行，经原CreateMaintenance再次核真实executor/authority/Gate，签发全新维护Session。查询/连接/ctx错误直接失败，没有把未知错误吞成可新签。

MaintenanceStore、严格Rotate和resolver源码没有改动。普通false+ordinary、私有phase、User/run/安装、CSRF/expiry/lineage/Gate、初始G+1/false/两项审计及原子ACK实现均沿原合同验证。没有新公共API、依赖、SQL迁移、Runtime/RunStore/DDL/model/main.go或清单外源修改，没有staticbool授恢复权。

完整原像12、最终12、实际3文件diff、逐命令sourcebeforeafter、12SHA见 before、source、author.diff、raw及release。原12均为共享树已有untracked源；没有伪造Git提交。tracked diffcheck不能覆盖这些源，因此额外保留原像/最终精确diff、no-index检查和逐行空白核对。

## G0 和授权

全文读取本轮 canonical `s411-f001-repair-source-local-pg-start-r01.json` 及MainTemp当前frozen-objects.json，逐项核own12修复前SHA和安装5新SHA+docs6严格peer11，23项均吻合。后者仍记before12，实施时只放行原owner12，并非把Main冻结文件写成新SHA。固定branch/HEAD核对，方法/Role/Rules/Profile与准备轮实际全文同SHA复用。此次没有重Route或新task。

准备轮完整Role、主实施/TDD、coding/nohardcode/task/multiagent及必要GoProfile读集和正式finding/设计3752/3764–3766/TP814/原卡/实际Handler调用链随 preparation 原样保留；current-resumption-readset注明复用边界。旧pending/draft不是新的技术Gate，人的接纳仍unknown不预签。

亲读本人r02 run.py、owned_windows.py、check_functions.go、verify_pg.go、release.py完整源码后复用代码；改为本轮独有根、新START的peer11、当前新raw标签和private缓存。所有旧结果仅保留来源，不充作本轮执行。physical工具增加当前精确新名单的User/run/Session/Audit完整后像；无历史schema扫描或DROP。

## 真实 Red → Green

1. compile-old-cookie-r01：真实旧生产源与原三叶test-only fullprefix overlay编译成功，exit0；不以编译失败当Red。
2. behavior-old-cookie-red-r01：真实3叶，no_cookie_positive PASS，old_runtime_cookie及lost_change_response_old_cookie各FAIL，实际exit1。两条原marker均为status500/platform.internal_error，无setup/SQL/零匹配混淆；3个新schema及完整后像保留。
3. 按精确before12/peer11再核后仅落入3文件，并只gofmt本人3源。compile-cookie-green-r01 exit0。
4. cookie-green-r01：原三叶、原探针后缀/预期未修改，3PASS；final-independent-cookie-race-r01同3PASS，无race报告。这里只是作者复跑独立发现的探针，不签成新的独立Review PASS。
5. cookie-browser-matrix-green-r01：新Handler Cookie矩阵11PASS，随后当前完整P的normal/race均重跑并包含这11叶。

原三叶后缀精确SHA和每次overlay完整前缀保存在setup/green-probe-prefix-preservation及preparation/reviewer-probe-provenance。新结果均引用本轮actual raw；准备草稿和旧执行不算新Red或Green。除上述故意旧版行为Red外，本轮Go编译/测试/vet/AST/SQL检查均exit0；no-index diffcheck真实exit1、stdout/stderr空，表示文件有差异，未伪写exit0。

## 最低 U/C/P 和相邻回归

| 当前检查 | normal | race | 含义 |
|---|---:|---:|---|
| identity相关回归 | 198PASS | 198PASS | Login/Bootstrap/Onboarding/RecoveryPasswordChange/既有RecoveryIdentityExecution及本卡U/C |
| session全包 | 60PASS | 60PASS | 原普通与维护Store全部当前叶 |
| 本卡PG完整5组 | 42PASS | 42PASS | 原31＋新增11真实Handler Cookie叶 |
| 原三叶保留探针 | 3PASS | 3PASS | 与旧版Red同预期配对 |

以上均无skip、无race报告，实际叶与group映射见validation-scope。U/C ownscope为identity32（原28＋新4）、session25，已包含于198/60，不重复相加。P分组为ScopeAndPrivateBinding1、BootstrapOnlyMePasswordAndDailyReject16、PrepareAndChangeAtomicGPlusOne6、PrepareChangeAckLossAndResume15、RevokeMaintenanceBeforeOrdinaryLogin4。

新增4U覆盖无cookie不查、精确行可选、确定无行可新签、SQL错误不得冒充无行，检查caller事务未被helper提交/回滚。新增11P覆盖去cookie正控、旧Runtimecookie、丢改密响应旧bootstrapcookie、现役合法cookie旋转、撤销/两种过期/未知cookie、旧密码拒绝、Gate阻断、Login Audit失败回滚。

P使用真实公开Runtime phase生产、实际初始Prepare/Login/强改密、GatedArgon2id、同pool/Store/Session表、持久abuse/auditor和现役Handler。真实Resume零改密/零hash，旧cookie继续拒protected/ordinary；新Session逐行核scope/当前4binding/G3/permanent/CSRF/lineage及原像差异，User/run proof保持；合法旧token的revoke_reason、rotated_from和CSRF轮换精确核对；失败路径不发cookie且Session全后像不变。原5组的真正提交丢ACK、独立回读、原子回滚/撤维护会话等全部当前重跑。没有用count-only或fakePrincipal替代成功证据。

U/C与integration vet均exit0；原12 gofmt -l 空；新5源Go AST函数复杂度检查exit0。实际list核对原U/C/P top。源码在这些最终检查与normal/race作业前后保持同一12SHA，无fmt后未测的源变化。

## 运行、SQL和释放

本轮23个真实owned作业均完成parent/process Wait、kernel ActiveProcesses0、双actualEOF和pump join，无timeout或io_errors。完整argv、command/result、stdout/stderr、before/after源快照逐作业保留。matrix与final_checks调度Python均真实返回exit0，其子runner只有完成whole owned job后才返回；不把Popen.wait当whole tree证明。53个当前已知PID均OpenProcess error87，无pending、无kill；PID观察不冒充全Host census，whole-job证明另存。

最终P normal/race各42个pool.Close/join与51个Runtime Stop/join；原探针各3个pool/5个Runtime；新增定向11个pool/13个Runtime。真实回调已退出，HTTP request/response Body关闭。资源明细见release；所有失败作业同样join。

只私读Main postgres-resource.json中的identity_dsn，经本轮private配置注入，实际目标为127.0.0.1:51935 / han_b13e_identity / han_b13e / PG180006，非旧端口、ambient/business DSN。只用identity，不读article密钥。104个本轮随机unique schema精确登记；包含3个Red及Green/最终各阶段schema，全部retain。独立readonly repeatable-read observer逐个核23表，并读取4组完整User/run/Session/Audit行，证明连接到本轮新库；没有历史schema枚举/清理。observer短只读事务提交、连接Close真join，最终本人精确标签SQL backends为空。

GOMAXPROCS1、-p1/-parallel1、每次单Go parent，GOPROXYoff/GOSUMDBoff/GOTOOLCHAINlocal/GOENVoff/GOWORKoff/modreadonly；cache、profile、TEMP/TMP/GOTMPDIR和-work全部本人r04独有D路径。仅从本人已停止r02缓存复制字节，旧缓存/所有WORK/data均保留、无共享写cache。Main helper66870/PGserver/data未停止或改动。

release前再核baseline8400无越界漂移、peer11精确、old23逐字节未变、固定branch/HEAD、本人只3源差异。原23既有rootfmt失败仍保留，不借旧有限处置豁免。Main最终联合冻结版root7/current17格式和两位独立复审分别待Main，不把本卡局部通过当全仓/生产root/R/E2E或人的接纳。

## 交付和后续

release.json已先交Main，包含12SHA、actualresultroot、全部23作业/53PID/104schema资源收据。源/执行权限已交回；其后仅本人Dpacket读取、拷贝、SHA和文本封装，不再发起Go/SQL/项目source/服务动作。

包内有report/receipt/manifest、before12/source12/diff、准备读集和原失败证据、全raw/lifecycle、exactschema/fullSQL后像、runner源码/provenance及限制。实际DSN配置、缓存、WORK、数据库data不入包且保留原位置。原Confucius需对Main新冻结23重新独立复审；两个HIGH不由作者签CLOSED，Task/Plan/设计/Git未写。若后续有finding，由Main另授原12窗口后续接同ID。
