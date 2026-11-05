# S4.11 independent CodeReview R01 — CHANGES REQUIRED / RELEASED

Confucius / add-reviewer，非原作者 Lorentz，actualID `01a1890f-a330-73d1-86fb-236777c54751`。固定 Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，工作区 `D:/AID/VSCode/Codex/Han`。评审对象为原 identity-auth-S4.11 的12源，完整路径/SHA见 `guard-final.json`。其他 peer5/doc6 只按冻结边界保护，不替 Ptolemy 审 S4.13。

本轮有一项实际确认 HIGH，CodeReview 为 CHANGES REQUIRED。Source0、LOCAL/IdentityPG 已实际释放：5个实际命令全部 parent Wait、kernel Job-zero、stdout/stderr EOF 后 pump join，pending0。`release.json` 先于本报告写出；之后只有 own D Temp 文本封装，没有源码、Go、SQL或服务操作。原作者修复由 Main 另授窗口。

## [HIGH] S411-ICR-R01-F001 — 旧 Cookie 阻断已改新密码的维护重登录

- **当前对象**：`internal/identity/recovery_auth.go` SHA `9615E41EA95E505AF66DC679D6D42CE372FE3BCEEF344383F45E903EF28B3ECB`；全部12源及23保护对象本轮实际命令前后均一致。
- **主位置**：`recovery_auth.go:203–208`，尤其 `:208` 无条件调用 `RotateMaintenance`；真实消费者 `internal/identity/handler.go:540–541`、`login.go:1341–1342`；旋转限制 `internal/session/maintenance_store.go:234–240`。
- **原合同**：设计 §4.2 `:3752`、`:3764–3766` 要求强改密响应/新 cookie 丢失时用新永久密码在同一维护入口重新登录，Resume 不重置密码，旧 runtime/cookie 不重新授权。TP `:814` 与本卡关闭条件也明确新密码维护重登录。这里是使用新密码重新认证，不是请求把旧 cookie 当作有效维护或 ordinary 权限。
- **实际触发一**：正常 Prepare/维护 Login/强改密已提交 G+1、false flag、run proof 和两项 Audit；改密响应丢失，浏览器仍持已撤销 bootstrap cookie。再次 POST sessions 提交正确新密码时，现有 Handler 自动把 cookie 放入 ExistingSessionToken。维护 Login 已验证新密码及当前 authority，却必须旋转该旧 token；旧 token 已撤销且仍为 G/bootstrap，旋转失败，HTTP 500。
- **实际触发二**：强改密后保留维护 permanent cookie，原 Runtime 真 Stop/join，再经新 Runtime 的真实 identity phase Resume。Prepare 重试零覆盖，旧 cookie 的 protected resolve 按原合同拒绝；但正确新密码登录仍因携带该旧 RuntimeID cookie 而强制 Rotate，HTTP 500。
- **正控**：同样真实 Resume、相同合法新密码，不附旧 cookie，现有 Handler 返回200并签发新 Session/CSRF。排除了错误密码、源/phase/Gate/数据库准备失败和未构造真实 authority。
- **影响**：两条设计承诺的恢复重登录路径在正常浏览器 cookie 状态下被阻断；用户必须自行清 cookie 才能继续。没有把本问题夸大为旧 cookie 越权、旧密码复活或密码部分提交。
- **最小修复方向**：原 Lorentz 在原12源范围内，把“新密码已验证后的维护登录签发”和“旧 token 仍符合旋转资格”分开。旧/过期/已撤销/旧 Runtime cookie 不能成为新密码认证成功后的强制准入条件；需要轮换时仅按既有合法事务/authority/Gate 规则处理。保持原 resolver、RotateMaintenance 的精确绑定、旧 cookie 无授权、ordinary 隔离和失败事务规则，不能通过放宽旧 token 的安全条件修复。
- **复审**：保留原配对探针断言；原两种 cookie 情况应获得新的当前 binding 维护 Session，旧 cookie 仍不可用于 protected/ordinary。修复后独立复跑本卡所需 U/C/P、实际消费者及普通/race，核现有合法旋转/CSRF/expiry/ACK/Session/Audit不退化。

## 两轮实际静态审查

Round 1 完整读取新增5文件1970行：recovery_auth.go503、其 U158/P848、maintenance_store.go256、其 U205。其余7既有文件的全部实际 diff 已读取，并用 retained before 与当前工作树独立重建核对全部12个 patch section；每个结果等于当前源。详见 `full-new5-coverage.json`、`diff-reconstruction.json`、`author.diff.frozen`、`existing7-independent.diff`。没有把文件存在、摘要或作者已审当作内容审查，也不声称重新全文读了全部既有万余行测试。

Round 2 沿现役 Handler.Login→ExistingSessionToken→Login→storeMaintenanceLogin→MaintenanceStore.RotateMaintenance 追到上述恢复路径缺口，同时核实际 password-change User/run/Gate/Session/Audit、Prepare/changed proof、ACK image/独立连接、普通 Session Create/Rotate/resolve、bootstrap ordinary ChangePassword 拒绝 required flag 等接口。当前 S4.10 scope image/authority、0010 原 scope/metadata 约束和原用户/凭据 lineage 是实际消费者依据，不修改这些 owner。具体读取范围、当前SHA在 `readset.jsonl`，方法/Rule/Profile 与之前同版实际正文的 SHA复用在 `methods-reuse-sha.json`；本轮另全文读取 Role/主 Skill/Workflow、checklist/abstraction/安全红线/Go coding，未加载未来 Workflow。

设计 §4.2 F001、共享基础 §3.3.1、TP §4.7.1、本任务卡及 Plan B13E 已按精确范围读取。历史制品的 NOTRUN/未认领文字与当前 Main actual grant/release 分开，不重跑 G0/设计前审或把后继 root/S4.16/E2E 倒挂为本卡前置。

## 独立实际执行与失败证据

实际根 `D:/Temp/han-b13e-s411-independent-4rtaabpv`；raw 在 `packet/raw/`。独立探针追加于当前两个原测试文件的完整字节前缀，production overlay0、test overlay2；原断言未修改，没有 shadow identity 包或虚构 production provider。使用真实公开 Runtime phase、真实 Store/Pool、Gated Argon2id、持久 Login abuse/auditor、现有 Handler 和真实 User/run/Session SQL。外部 source evidence/HTTP observer 是明确的软件 fixture，不宣称实际 production root/router 已注册。

| 作业 | 实际父PID | 程序退出 | 结果 |
|---|---:|---:|---|
| preflight | 54440 | 0 | 实际命名数据库/版本/data目录一致 |
| go-version | 64712 | 0 | 本地已安装版本，原raw保留 |
| probe-normal | 34028 | 1 | 三叶1PASS/2行为FAIL，两个精确 finding marker |
| probe-race | 65628 | 1 | 相同三叶1PASS/2行为FAIL，无 race report |
| physical | 60296 | 0 | 精确6新 schema 实际 SQL 后像，全部保留 |

行为断言原raw：`raw/probe-normal/stdout.log:68`、`:87`；race相同行号。两条均返回 `500 / platform.internal_error`。配对 no_cookie_positive 返回200。没有 compile/setup/SQL/timeout/skip/race 诊断，也没有零匹配。runner 对有效证据收集返回0不能代 Go PASS：两个实际 Go exit_code=1 均原样保留。

6个 schema 每个都有23表，独立 psql 逐名字回读 User mode/flag/双generation、run bootstrap/changed/Audit、Session scope/runtime/generation/revoke事实和Audit结果。所有 fixture 里初始管理员均保持 permanent/false/G+1，证明新密码已真正提交；失败分支没有因此重置/清 proof。精确名单、叶子对应和完整后像见 `independent-result-and-physical.json`。这是正常现役 cookie 输入；没有伪造 producer、放松 migration/trigger/authority 或用SQL错误当行为 Red。

按 grant 的“首个确认 bug 停无关扩展”，确认普通结果后只运行同一缺陷的 race 与必要独立 SQL。**本角色未执行**作者194 identity相关/60session/31PG矩阵、28/25 U/C主top、vet/list或全仓检查；这些在本角色为 NOTRUN。作者实际37job及194/60/31两模式和Main独立130schema接收均按来源保留，不能签成独立PASS。作者原 Resume 正控经 s411Login helper 不设置 ExistingSessionToken，故未覆盖本次真实 Handler Cookie 接缝。

## 授权、资源与停止

绑定 Main canonical `independent-two-code-review-local-pg-start-r01.json`，只允许新命名 IdentityPG。实际 preflight/final SQL 为127.0.0.1:51935、han_b13e_identity/han_b13e、180006、data `D:/Temp/han-b13e-lead-20261030-r01/postgres/data`。只私读 identity_dsn 构建本人private JSON；DSN/该JSON不打包、不打印。没有访问旧51934、其他数据库或ambient/business DSN。CPU1/-p1/-parallel1、local/offline/modreadonly；复用同Agent旧暖缓存，未写其他Agent缓存，所有新可写TEMP/work/profile在独有D根。

亲读143行 owned Windows helper 后按exact SHA复用，suspended assign-before-resume、受控handle allowlist、不breakaway。全部5父 Wait、kernel ActiveProcesses0、两个实际 EOF/pump join；kernel accounting81进程，可选观察13 PID值只是子集，不冒充逐child创建遍历或全Host census。探针原raw含10次实际 Runtime Stop/join、6次实际 pool.Close/join；事务/Rows关闭沿真实被调实现及 fixture收据保留。命名schema/数据全部保留，无DROP/删除/移动/归档/服务操作。

23 source/docs 每个实际命令前后及结束均未变；未写project源码、SSOT、设计、Task/Plan、Issue或Git，未子委派、未替作者修复或签风险接受。原root23格式失败仍是原失败，不以旧有限处置、类型nil或作者bool当PASS。

当前 code结论为 CHANGES REQUIRED，`S411-ICR-R01-F001 HIGH OPEN`；不关闭Task/G4、不代人的接纳。其余软件矩阵待原作者修复后的独立复审；真实 provider/ACL/SMTP/native/部署/E2E和后继root职责均未由本轮证明。LOCAL/IdentityPG已先release，后续仅ownD封包；Main接收后交原Lorentz按新窗口修复。
