S4.11 维护认证闭环：作者实施与局部验证交付

本会话 Lorentz / add-backend-dev / 01a189aa-4e63-7d10-90c1-561fc6d70e8a，在固定 Han/main/HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca 完成原 12 文件实施、自测和资源闭链。SOURCE/LOCAL/PG 已先通过 release.json 释放。Main 后续明确实际接收并冻结 23 对象，关闭本作者写/运行窗口；独立 Reviewer 正在各自获授窗口运行。这里不是独立 CodeReview PASS、任务关闭或人的业务接纳。

当前 source/ 是本次真实 12 文件最终像，author.diff 对比 Main START 的原 7 份 before/ 与 5 个不存在文件。完整 SHA 在 release.json；所有 12 当前为 Git 未跟踪文件，Git 原已脏树保持，未 stage/commit/push/checkout。只写原 12，不修改 RecoveryRunStore、Runtime、DDL、S4.10、main.go、设计、Tasks、Plan 或 peer 五源。8400 保护集仅见本任务七既有源与 Main 已确认认领 metadata 的变化；peer5/docs6、HEAD、原23格式失败对象字节保持。Main 已独立核原17范围及 staged0（来源为用户现场核收通知）。

维护 Store 在 session 包定义设计规定的中性 binding/authority；identity 实际 executor 核 Runtime 私有 phase、同 run/install/runtime/generation、初始管理员与真实 Store proof。构造无 DB/文件访问，缺依赖 fail closed，维护解析不 fallback。普通解析保留 recovery_required=false，在查询入口固定 ordinary；拒绝维护 token 不撤其行。普通 Create/Rotate 固定 ordinary，绑定四列 NULL。bootstrap 维护 principal 只准 me/password，维护 permanent 仍受私有 phase 和后继 root allowlist 约束。

Login 保留现役密码/失败统一、持久 abuse、审计、token/CSRF 发行链，同一 Store/pool 签发 maintenance；静态 maintenance=true 不能授恢复权。强改密使用设计规定 RecoveryTrustChange，保持原 PasswordChangeRequest/Result 与错误注册表；按 User→run→Gate→Session，在同一事务写新 Argon2id 摘要、G+1、false flag、run 五字段 proof、原改密及恢复信任根审计、同 binding permanent Session rotate 和其他 Session 撤销，source operation/credential 仍 NULL。任一失败不发 cookie且整组回滚。额外审计故障 RED 证明必须核审计内容，最终实现核其 actor/action/target/run/backup/install/user/generation/原审计引用，不能以 appender 返回 nil 或“有行”代替持久事实。

提交丢 ACK 用保留的 writer 连接与独立物理 reader、短只读 repeatable-read 事务核完整 User/run/Session/全部相关 Audit 前后像；部分、不明或失权持续 503/关闭，不盲重写。实际 Prepare/Revoke 调用冻结的原 S4.10 receiver，在本轮新 schema 重新执行，未改禁写源、未拿旧运行当新证据。Resume 保持已改密码，旧 runtime/cookie 拒绝；新密码可维护重登。切普通面先撤本 run 全维护 Session并核审计，再用新密码重新普通登录，旧 cookie 不升级。

| 最终当前版验证 | 普通 | race | 证据 |
|---|---:|---:|---|
| identity 相关回归，含本卡 28 U/C 叶 | 194 叶 | 194 叶 | raw/final-identity-normal-r01、final-identity-race-r01 |
| session 全包，含本卡 25 U/C 叶 | 60 叶 | 60 叶 | raw/final-session-normal-r01、final-session-race-r01 |
| 本卡真实 PostgreSQL P，原五组 | 31 叶 | 31 叶 | raw/final-pg-normal-r01、final-pg-race-r01 |

本卡 U/C 为 identity 28 + session 25 = 53；已包含在上述回归数字中，不重复累加。validation-scope.json 保存每个真实最低叶与普通/race 同名对照、五组最低层覆盖。C 含实际 Runtime phase producer/消费者、typed construction、session-neutral authority/Gate/调用次序与兼容边界；P 用实际 Login、Gate、Session SQL、User/run proof和独立显式 SQL oracle，未以 fake Principal 或 count-only 证明。全部当前叶均 pass，无 skip。两种 vet（普通 identity/session、integration identity）、本卡 12 文件 gofmt 空输出、新五文件函数≤50行 AST 自检、只读 diff/完整 whitespace 自检均完成；未重复全仓矩阵。

raw/ 保留全部 37 次调用的固定 executable+argv、开始/结束、源 before/after 字节快照与 SHA、完整 stdout/stderr、结果、叶和 lifecycle，包括编译/setup/行为失败。三个实际 RED 链及修复在 failure-chain.json：先真实编译，再行为失败，不把 prep/stub 存在或编译失败当 RED。所有调用均 parent/process 实际 Wait、kernel Job active=0、两条实际 EOF 与 pump join、无 IO 错误/timeout。84 个已观察 PID 的最终只读观察均 OpenProcess 错误 87；原 Job 已归零，未 kill 未知 PID。CPU1、-p1、offline/local/readonly、私有 D cache/TEMP、-work 留存，单 Go parent 串行；runner 为亲读后的 Hooke actual EOF 版本，独占路径/provenance 已记录，未运行旧脚本/复用旧 raw。

最终普通与 race P 各有 38 次实际 Runtime Stop/join 记录和 31 次实际 pool.Close/join 记录。精确 130 个本轮新 unique schema（含失败阶段）全部保留；独立只读 SQL 对每个精确名字核实际 23 张表，输出完整表名集合，未扫历史或按前缀认领。实际 named server 为 127.0.0.1:51935 / han_b13e_identity / han_b13e / 180006；已标记的本轮连接无残留，观察连接与只读事务已关闭/提交。只从 Main resource 私读必要 identity_dsn，未用 article_dsn、ambient/businessDSN、旧51934/5433或旧C/D。Main helper66870及 server/data 仍由 Main stop/wait；本作者未操作其生命周期。

P fixture 的外部批准/Journal/restore 适配是明确的合成软件输入；User/Store/Session/审计/事务与 Runtime phase producer 为真实实现。Argon2 8KiB 是测试 fixture，不是生产强度或容量批准。唯一 production root/实际 HTTP allowlist/切流/lifecycle R及联合专项仍按 S4.16/S4.17等原任务；没有真实服务/provider/browser/native恢复、部署、E2E或人的接纳证据。原 rootfmt23 只保留已有失败对象字节和来源，作者未重跑 rootfmt；全仓七检查及共同冻结版17 Go格式仍由 Main 独立执行，不继承旧有限处置作当前豁免。

release.json 已由 Main 真实接收，canonical 指针为 s411-source-local-pg-main-release-r01.json 与 independent-two-code-review-local-pg-start-r01.json，来源为最新用户通知；本作者未在窗口关闭后重新读取/签署两份 Main 制品。之后仅 own D 文件复制/报告/校验，无新 workspace 源写、Go/SQL/服务/子委派。若独立评审有修复，由 Main 以 finding+新窗口交回原12后续接。所有 schema、data、cache、WORK、原始失败与旧 prep 均 retain，无清理/删除/移动/归档。

formal packet 为 cache-free：包含报告、manifest、最终源/diff、before、readset、完整失败及 raw/lifecycle、精确新 schema 与限制；不含实际 pgconfig/DSN 凭据、Go cache/modcache、data 或编译产物。runner/ 与各阶段快照仅供复核 provenance，不能当作新执行或未经新窗口直接运行。作者当前 STOP，交 Main 冻结、独立评审/复审与正式关闭；本报告不替代 Lead 实际核查。
