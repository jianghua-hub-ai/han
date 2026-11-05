# B13E whole-batch 独立 G4 Review R02

**结论：有界 G4 PASS；RECOVERY-B13E-G4-R01-F001（HIGH）独立 CLOSED；新增 finding 0，当前评审范围无未闭产品 finding。**

Dirac/add-reviewer（01a18663-d9ac-79c2-b843-ba6dcca3de7e，非作者）；objectType=code。固定 Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca。范围为 S4.11 maintenanceAuth12、S4.13 installation5，以及本 finding 最小返工3源；当前 **20源+6文档=26** 的逐项 SHA 见 evidence/objects-final.json，执行前后及封包时均一致。未写仓库、Task/Plan 或生产 overlay。

**两轮整体判断。** 第一轮先审直接缺陷：复用本人已审未变17源和6文档，完整读取/重构本次3文件差量；生产仅在 recovery_execution.go:817 将可选删除时间由 reflect.DeepEqual 改为 nil 安全 time.Equal，其他生产字节精确不变。原 R01 同一 UTC 时刻因位置表示不同而被拒绝的真实 Red 保留；此次既未改原探针/UTC，也未做舍入或放宽角色、状态、成员集合。第二轮沿真实 public Install→同 Store/pool/installation/run/runtime→finalization/source proof→maintenance authority/Gate/Session/Audit 回追，并实际组合验证。因此结论来自跨卡数据流与拒绝边界，不是两个单卡 PASS 或叶数相加。

安装来源/承诺前后证、fresh/prepared/bound 幂等与不可覆盖、0010/0011 历史及原 NULL 一次绑定、normal 不修补的责任延续；本次组合证明实际 Install 自行产生的同 Store 安装绑定可被维护认证消费，未 INSERT/copy/UPDATE installation 凑条件。prepared permit 不能认证，安装允许的 CAS/media 动作与 Auth 零写分别断言。维护 cookie 不成为 ordinary，G+1/CSRF/lineage、先撤维护再 ordinary、两 Audit 同事务、ACK 独立连接整像与 unknown 禁重放的原合同未被时间比较修复改变；User→run→Session/Gate 的原锁序及 Runtime 不跨 I/O 持互斥锁的边界复核无新增冲突。

**独立实际证据。** 原2叶字节精确保留，加10个独立 proof controls，普通/race 各12 PASS，0 fail/skip/unfinished/race。同一时刻 Z/+00/+08 正向、微秒差/nil 两向/role/status/缺失重复额外成员负向均经真实 finalization 和维护消费者；负向要求具体 ErrAdmissionBinding，bootstrap/credential/session/change-audit 禁止动作仍为零。原正向包含双方 nil。24 个新 exact schema 的 Installation/Run/Restore/User/Review/Session/Audit 全像经本人2批只读 RR SQL 实查（非跨批 global RR），全部保留。4 个实际 parent Wait、155 个已取得 handle Wait、Job0、双 EOF/pump join；Runtime/callback/Rows/Tx/pool 均已清算并先 release，再封包。详见 evidence/checks、required12-normal-race.json、physical-new24-schemas.json、local-pg-release.json。

**当前验证适用性。** Main 原23项与独立 Auth R02/Install R03 为未变区域依据；修复后 Main8 的当前 raw/hash/生命周期和26源守护已独立静态核对：11U/3P 各普通/race、owned20fmt、integration vet/build 通过；不声称重跑全部历史。rootfmt 仍 FAIL2，stdout D01225B146C06C238B38AEA344662D8253842CEE671EF68137BCC4B66C9379A8、stderr 53258F7B21EF40C6BE7E8E671216EEF5DD0441B27314FA806AE24F5C2CB010FD 与原23输入基线相同；本报告不改失败、不签全7 PASS、不采用 blanket waiver。Main 独立24 schema 核证、另6 schema 检查和自有 PG server59748 exit0/数据保留收据已收到；仅主张本人 owned jobs 与 Main 自有服务 join，不主张全 Host census。

**性能与边界。** full-history/source 校验成本随集合增长；密码哈希可占用锁和 CPU，ACK 二连接回读需要 pool 余量。静态成本与小 fixture 功能覆盖支持本测试项目当前有界接收，CPU1/-p1、耗时及 time.Equal 均不是容量/P95/大历史性能测量 PASS。审批/ACL/真实 provider、SMTP、native 正控、真实 restore、完整 root/S4.16 长 idle/后证失败 handle、Case18/E2E、人的接纳仍 NOT VERIFIED；ADR0024、S3.3/S3.5 的原窄边界及 S3.6 不变。未把这些下游能力反挂为本组件前置，也未扩大 phone/缺行残余/unknown hold 豁免。

证据连续性：normal 在已续接 prep 根实际完成后原样复制至 execution 根，未重跑；旧 prep manifest 只代表历史 preparation，后续根不冒原封印。原 R01 包不改。SQL collector 曾误写 audit action 字面量，保留诊断后按既有 identity.password.change 重读原 raw，无产品/SQL 重跑。详细读集、原件路径和哈希随包；此页不重复历史矩阵。

**STOP：soleLOCAL/PG 已 release，后续无 Go/SQL；本包仅独立技术评审结论，不关闭 Task、不代 Main 采用或人的签字。**
