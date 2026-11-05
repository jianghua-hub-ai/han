# 原 identity-auth-S4.12 / F001 作者修复交付

Lorentz / add-backend-dev / actualID `01a189aa-4e63-7d10-90c1-561fc6d70e8a`，固定Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。同一Task、原development-delivery返工；没有重新Route/G0/技术审批。实际写窗以setup/grant.json为准，仅原recovery_password_reset.go及其原U/P两测试文件，other7+protected26=33严格冻结。

作者修复与当前有界验证完成，SOURCE/LOCAL/PG已先释放，随后仅本人D封包。S412-ICR-R01-F001仍OPEN，等待独立R02，不自签CodeGate/wholeG4或关闭Task；人的采用/接纳unknown保持。

根因是新候选按purpose=password_reset单独取MAX，却违反既有UNIQUE(user_id,generation)的账号全用途序列。原newAdministratorAggregate产生generation1，真实resend继续该用途；Finalize保留consumed/superseded历史并失效仍可用凭据，没有删行。因此合法ND0备份管理员无reset历史时，旧代码再次分配1，必然23505并回滚/Send0。

生产增量只有一处SELECT：在原目标User锁内取该user全部temporary_credentials的最高generation，移除purpose过滤；原negative/MAX-int64预分配拒绝保持。无DDL/unique修改、无历史删除/改写、无新API或第二authority，没有改model/operator/store/runtime/root及另外7源。production.diff只有一旧一新SQL行，repair.diff另含原两测试增量。

已全文读取指定finding/report、当前分配函数，并沿真实administrator aggregate/persistence、resend generation/lifecycle、password-change repository、Finalize及0002/0008和普通reset消费者核对。Role/task-implementation/bounded-change/delivery/TDD/启用go-service依同一会话已实际全文同版复用，新增精确读集/SHA在readset。原草稿只在独有Temp准备，收到实际START后再次核Main before36逐项一致才安装原两测试增量；未整体覆盖10文件。

真实失败链：规范接收的独立normal/race原始流均为B2冻结源码、真实23505，严格保留原oracle和clean正控，来源在reused-independent-red；不是作者新执行。作者新增三种历史测试先编译成功，再在原生产B2取得三个23505：真实创建原语产生consumed、active及真实resend的更高代际历史，随后真实Finalize/privateRuntime/强改密维护认证/Store/SQL入口。三种历史均在一行生产修复后转绿，分别分配全账号最高值之后的2/2/3；post-Finalize旧行整像不变，一次secret/Send，terminal重放零新ID/secret/SMTP。fixture调用原aggregate/repository/lifecycle原语，不声称完整创建HTTP、真实SMTP或批准provider。

U新增4个generation读取guard场景：MAX-int64、negative、SQL error、context error；错误按原链返回，secret/ID/写入均零。脚本SQL U不冒充跨用途数据库证明；真实跨用途唯一约束和history行为由P与原独立探针验证。

当前版normal/race逐名一致：identity554叶（含本卡U41、reset/resend、管理员创建/onboarding生产者、S4.10/S4.11维护消费者）；P99叶=本卡七风险55+维护认证42+原独立配对oracle2。全部PASS、无SKIP。原oracle以完整不变后缀追加到当前P源，prefix字节与当前源SHA一致，新增P测试未被旧overlay覆盖；作者复跑旧oracle不冒充独立评审。C合同证据继续落在实际private HTTPAuthorizer/Reader的P边界，逐字段source/普通拒绝保持，不拿父top代子叶。原Task七风险原52叶全部保留，另增3history；minimum层次逐名见current-matrix.json。

两种vet、三源gofmt空输出、函数AST<=50行、精确Git diff--check均通过。goimports/golangci-lint未安装，未签其PASS，也未网络安装。没有追加全仓/历史总矩阵；session包与其他未改消费者保持冻结，当前P实际使用原scope/session/privateRuntime和完整原子证明；本轮没有把旧session包60叶冒称新运行。Main最终共同版检查和Rootfmt原23保持单独责任，作者guard确认其字节未动。

14个作业均parent/process Wait、kernelJob0、双实际EOF/pump join；32已知PID无pending。Runtime.Stop/回调、pool/permit/事务/rows均真实join，原始raw与sourcebeforeafter保留。仅本轮204个新UUID schema，逐一RR只读提交并读回4812份全表像；fixture backend为零，observer关闭并退出。原作者/Reviewer旧schema未读写，原数据/cache/work保留。唯一命名PG为Main51936 identity；未用旧51935/ambient/business DSN，私有DSN不入包，Main服务仍由Main持有。

13391保护无原Task10/已授权claim元数据以外差异；本修复相对Main freeze36仅3源变化，other7+protected26严格一致，原old23字节严格一致。没有Git写、Task/Plan/design/TP/DDL改动、清理/删除/移动/归档、子委派、provider/native/restore/E2E。源10最终SHA、冻结33、当前必要命令、全部作业/ownnames和pending0在release.json；release后无新Go/SQL/项目源码动作。

本包是作者修复证据，交Main现场复核及原Confucius独立R02复审；真实批准provider/主体/ACL、production-root接线、SMTP/native/E2E/人评审仍按原后继边界，不以软件fixture代替。源/diff/raw/readset/物理账/manifest组成cache-free交付；工具源码与精确资源overlay仅用于复核provenance，重跑必须使用Main另授窗口和新own paths，不能把旧执行当新执行。
