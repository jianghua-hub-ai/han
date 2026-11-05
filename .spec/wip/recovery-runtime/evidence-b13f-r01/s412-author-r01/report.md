# identity-auth-S4.12 / B13F 作者实施交付

状态：原10实现与作者局部验证完成，SOURCE/LOCAL/PG已释放；待Main现场复核和独立Review，不关闭Task、不自签独立PASS。

Lorentz / add-backend-dev / actualID `01a189aa-4e63-7d10-90c1-561fc6d70e8a`，同一development-delivery续接。固定Han/main/HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，共享脏树保留。精确授权、当前源10SHA、当前必要命令和所有作业生命周期在release.json。G0已由Main接收；旧pending/draft按实际技术接收范围解释，没有重新开技术Gate，也没有代人的结果接纳。

实现恢复批准读取的精确version/install/run/actor/target/role/status/ref/digest绑定，复用现役private phase、同pool/Store、真实初始admin强改密证明和普通classifier/Gate。新增candidate仍用原initial_reset与scope `recovery_review_restore/v1`，在User-first事务内原子写counter/claim/operation/exact pending credential/job/review指针；ordinary三个拒绝保持。只有确认的新owner持原secret，经durable dispatch ownership在事务外Send一次；accepted只记录受理。无Mailer结算receiver按原due/Claim/Renew/attempt/MaxRetries/permit处理；不用ordinary worker Start、顶层operator嵌套事务或requeue权限。

accepted以一个事务完成exact credential、User G+1/temporary/批准role-status/false、旧Session撤销、operation/claim/job/review与两份Audit；trusted suspended保持停用。whole ACK沿原executor独立物理回读，覆盖User/run/review/operation/credential/claim/job/Audit并重核可信来源；terminal同意图只读，错key/role/status/ref/digest或D0请求拒绝。Complete调用原S4.10完整集合，真实证明单个restored不能掩盖其他pending D0，合法终结其余目标后才能完成identity组件；未声称全局Runtime已开放。

真实TDD：先编译可运行scaffold，再取得5个U行为失败并实现转绿。真实PG发现两条新/旧非受理结算路径把合法失效错误返回restored成功，分别最小修正为失效事务提交后仍pending；原窗口与失败断言未放宽。沿原failed-reset消费者追加真实断言，取得两条Audit actor不匹配Red，修正为原durable settlement actor后转绿。原始raw/schema全部保留。两轮无效数据库fixture、一次nil Commit fixture panic、AST参数/包装解析诊断独立记录，均不冒充行为Red。计数器MAX无法以少量行构造合法持久链：P证明原FK/no-gap拒绝、U证明应用预写MAX拒绝；另有合法auth-generation和credential-generation MAX的PG零副作用拒绝，不谎称执行了不可构造的合法MAX链。

当前源普通/race逐名一致：identity U/相邻回归各527叶，其中本卡U37；本卡七组P各52；现役维护认证P各42；session各60。零FAIL/零SKIP。两种vet、原10 gofmt空输出、原命令top list及精确Git diff--check均通过。七组逐名层次见current-matrix.json；TrustedApprovalAndDailyRejection的C证据在实际private HTTPAuthorizer/Reader边界的P用例中，同时验证完整批准tuple/缺源零secret-ID-claim-Send、D0/current-deleted与日常拒绝；不拿父top代子叶。

当前最终生产变化相对早先normal版仅两个文件：恢复结算Audit actor与accepted加速job UPDATE行数检查；P源补原failed-reset消费者断言。最终版已重新跑本卡P normal/race、reset/resend/maintenance与S4.10/S4.11 U normal/race，当前SHA与早先407/52版分开保存。final-pg-race-r02同时含本卡52和维护认证42，其普通模式分别独立有raw。所有选作最终验收的作业sourcebefore/after均与release10相同；旧绿只作为历史，不改称当前结果。

45作业真实parent/process Wait、Jobzero、双实际EOF/pump join；92个已知PID无pending，未kill未知PID。实际Runtime.Stop/callback unwound、permit回收、事务与pool关闭沿fixture真实清理并留raw；453个本轮精确新schema逐一只读读回全部23/24表像，fixture backend为零，observer提交/关闭后退出。Main专用PG supervisor24773由Main持有，作者未停止它。仅新51936命名identity资源；旧51935/ambient/business DSN/真实provider/SMTP均未用。private DSN不在packet；cache/module/work/data/private pgconfig均留原D目录而不拷入包。

13391写前hash保护核无原10/合法claim元数据以外差异，peer26严格同、原old23严格同；代码前像7与Mainbaseline一致。原旧60行函数正文逐字未动，新/修改函数均≤50行。未改设计/TP/Tasks/Plan/Git索引、Store/Runtime/DDL/migration/root或其他源码；Git仅精确只读status/diff/stat/check。author.diff相对本卡写前7原像与3missing，区别于Git HEAD差异中的历史他人内容。当前源从最终已核作业source-after复制，release后没有再次读写项目源码或运行Go/SQL。

限制：goimports、golangci-lint本机不可用，未网络安装、不签其PASS；实际compiler/vet/gofmt替代证据单列。全仓公共矩阵及Rootfmt原23由Main最后共同版核，作者不自动豁免本轮；原23字节保留。未跑production root R、真实SMTP/批准provider/ACL、native/restore、E2E、部署或人的接纳；依原后继卡范围保留。独立Review和Task关闭由Main另行组织。无子委派、无清理/删除/移动/归档、无Git写。

release.json优先交Main以释放运行窗；随后仅本人D封包。jobs.json/failure-chain.json/readset/physical-receipt.json/source/author.diff/raw是本包可复核入口，manifest与receipt逐载荷SHA核对。本包不授权重执行历史写源脚本；tools保留实际runner/诊断源码和provenance，独立执行需按Main另授窗口调整own paths，不把旧执行当新执行。
