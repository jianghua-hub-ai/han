S4.11 Temp 准备交接；STOP（当前窗口已完成，任务未关闭）

固定 Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca。作者为当前受派 Backend-Dev 会话；Host 未向本会话暴露 runtime ID，不编造 ID，不进行子委派。Main 独立 review/现场接收后办。

仅在 D:/Temp/han-b13e-s411-backend-20261030-r01 写候选与证据。项目源码、Tasks、Plan、设计、Git/index 均未写。共享脏树保留；另一身份会话已结束和 peer S4.13 独占源码/Go 为 Main 提供的当前窗口事实。没有查杀/接管其他进程。

交付 candidate/ 原12文件（5新增7既有），before/ 七份精确原像，candidate.diff，全12文件 red/ 分阶段候选、readset/snapshots、validation/逐叶与命令清单、raw/静态检查与source-before-after、manifest。没有 cache/data/真实DSN 或旧执行产物依赖。没有压缩/移动/归档/清理，原数据与缓存均保留。

实现候选：session 仅经 identity 实际 authority 核私有 phase/绑定，SQL 固定 maintenance scope；普通解析入口增加 ordinary 过滤并保留 recovery_required=false，拒绝维护 token 时不撤它的行。bootstrap 维护 Principal 私有标记只准 me/password。Login 复用实际密码/abuse/audit/会话发行链，静态 true 不授恢复权，同一 Store/pool。RecoveryTrustChange 采用原 body/result 和设计规定 PhasePermit；现有 PasswordChangeDependencies 以既定 MaintenanceSessionAuthority 接入同一 executor，不改 HTTP/API wire、不新建 authority/DDL/Store/Runtime。

强改密候选：User→run→Gate→Session，读取真实旧 token、generation/kind/expiry/lineage；统一密码 policy/hasher；同事务写新摘要、G+1、false、原改密 Audit、run change proof、恢复信任根 Audit、同绑定 permanent maintenance rotate 与其他会话撤销。bigint 显式运算并守溢出；commit ACK loss 用独立物理连接及有界只读事务比较完整 User/run/Session/两种 Audit 前后像；不只数行、不盲重写。Prepare/Revoke 使用已交付的原 S4.10 receiver，在新 P 场景中重新调用，不修改该禁写文件。

候选测试：identity U/C 两组 + session U/C 两组，P 原卡五组全部具名。P 使用 Main 后续具名新 B13E PG signal，实际 Session SQL/Login/Gate/Argon2id；Argon2 8KiB 是软件测试 fixture，不是生产强度/benchmark。新 schema 精确生成与保留、只查本例名字、独立 SQL 全组 oracle；不调用旧 B13D harness、不访问旧 C/D。复用 S4.10 的 fixture 类型、真实 Runtime producer 及同步 Stop/Join SOURCE（readset 留存），不是复用旧执行证据，也没有复用旧 job runner。

当前只做静态作者自检：owner12 文件齐全、七既有原像和四设计/TP SHA 等 Main baseline、10项 requires 仍[x]、最新 Tasks/Plan Main metadata SHA 已核。候选词法括号匹配仅静态检查，不能证明 Go 语法/类型/编译/格式。尚无具体需扩源码/改设计/API 的 gap；源码共享窗口仍未转交。

所有 Go/PG/真实 FAIL→GREEN/race/回归/格式/七检查均 NOT RUN；raw/NOT-RUN.json 明示。red/ 只是可复核的测试+编译接缝阶段草稿，须在 LOCAL 实际编译成功、取得行为失败后才能记 RED；若有编译问题，先修候选再取真实 RED。当前没有成功测试 raw，更没有伪造失败、独立 review PASS 或人的签字。Rootfmt 原23只原样留基线，当前真实 rootfmt FAIL 及本轮17Go格式都待 Main/串行窗口重跑，不继承旧有限处置。

下一步由 Main 直接给同一会话 SOURCE/LOCAL/PG；复制之前再次精确核七原像，核 peer 源冻结并取得唯一源码/Go窗口，再执行分阶段 TDD 和本任务 U/C/P逐叶普通/race、包回归、项目检查，逐次留 argv/raw/results/源前后像，真实 Job whole-tree Wait/zero-active/pump join，SQL/ctx/Runtime/pool失败亦同步归还。未具名新 PG 前不连库。生产源稳定后 STOP 待 Main 冻结/独立 Reviewer，不自行关闭任务或开后继。这里不向用户增设审批。

限制：候选尚未编译、格式化或运行，源码接线/原子性均待新执行证据；后继唯一生产 root/HTTP allowlist/真实切流与联合 R/E2E 仍分别归 S4.16/S4.17 等原任务。没有部署、provider、native/浏览器恢复、真实业务接纳或生产可用声明。早期只读工具有错误文件名/Windows glob 失败及长输出截断，已改为精确文件与分段读取；它们不是 Go 测试失败，Host transcript 保留原 tool 输出，本 packet 不把回忆文本伪装成机器 raw。
