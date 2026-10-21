# identity-auth 详细设计第三轮独立复审报告

- 复审日期：2026-10-20
- objectType：design
- 先前报告：`review-design-r01.md`、`review-design-r02.md`
- 主对象：`identity-auth-design-ir01-r03`
- 附属对象：`identity-auth-stories-ir01-r03`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories、先前报告或上游制品。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `7460c0d68a9e30877c6f3e7f2938563322da4da329122c16957d8a6be54f413e` | `7460c0d68a9e30877c6f3e7f2938563322da4da329122c16957d8a6be54f413e` | 一致 |
| `stories.md` | `d24a549aec2cc76aefad6884611ae301ea80e39269c2f5f1cb6f2d3042f5148f` | `d24a549aec2cc76aefad6884611ae301ea80e39269c2f5f1cb6f2d3042f5148f` | 一致 |

按确认段排除并规范化尾部为单个 LF 后，design artifact digest 为：

`sha256:cb2f8f57bc3a8390d478dc777d3a9e205b31c9e7cf20d9b9cbe64170ac28c503`

与 Lead 提供值一致，无对象漂移。

### 1.2 上游与先前报告快照

| 依据 | 修订/状态 | 当前完整 SHA-256 |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r06` accepted | `e06560ce94cb0c570f04149e6a72490c3db1699c034f77710fdddd6c11036dfc` |
| identity boundary | `identity-auth-boundary-ir01-r05` accepted | `f944f8d80e055922b1bd797d0a07a7e733c22062554cb64bce3c80690583217c` |
| article boundary | `article-management-boundary-ir01-r05` accepted | `93cb4beabf541f3e8b24d608b63c3e66cb83cc3064511821ac29a16bfccf015a` |
| ADR index | ADR-0001～0018 | `c2dd27c3c5698ce375c21b0cbc37bfa788918064e48c2c663b897ac975ebfe74` |
| framework ADR | 修订 ADR-0018 | `705622967f956c01183731d7f8c36448e476388894024f07f2ea78cc3686c6f8` |
| identity ADR | 修订 ADR-0007 | `1e65e01665dfd13fa71e2aa4167082252d79aee3681900ebc816c11e04983d11` |
| content ADR | ADR-0010 | `f30ea02f15b313a95c928b9e278fc0b2567f6059de1b99f18fa8b36341840004` |
| r01 报告 | 固定 | `83df815960a22073f2dc022afea5f8c94353d4040e0e7b2a80385f6cac17b914` |
| r02 报告 | 固定 | `fd4ec9e2696fcd6662e6ae30919d84456c24d8913d2b61445e7d951d5d870713` |

architecture r06 checkpoint：
`decision-07ecc99f3c32dc2c024c9cf33bfd45a7f79ee78fda9b1f70b0930676a6e64bce`。

## 2. 实际复审范围

### Round 1：开放项定点复审

逐项复核：

- `IA-DES-R01-F003`
- `IA-DES-R01-F008`
- `IA-DES-R02-N001`
- `IA-DES-R02-N002`
- `IA-DES-R02-N003`

检查原触发场景、作者修复位置、上游接受依据、状态/数据约束及验证入口。

### Round 2：整改回归

沿 11 条 US、S1～S14、API、数据、事务、维护恢复、凭据、权限、文章删除接缝和 Story DAG 检查新增阻断风险。没有重开 r06 已接受的“账号 recovery Gate 而非 epoch”选择。

## 3. 开放项逐项状态

### IA-DES-R01-F003 — closed

- 证据：`design.md:859-865`、`924-940`、`1348-1369`、`1480-1498`、`1607-1664`；修订 ADR-0007、ADR-0018；boundary r05 S13/S14。
- 永久密码：恢复终结事务清空全部永久摘要，非初始账号进入 `recovery_locked + recovery_required`。
- 旧角色/状态：恢复点事实只进入 `RecoveryAccountReview`，不直接授权；非初始账号统一 suspended。
- 删除账号：旧 deleted/active 状态均不直接产生登录能力；无法重建时默认 denied。
- 信任根：只有既有 initial admin 可经独立 recovery secret 进入 `recovery_bootstrap`，首次登录强制改密。
- 原“旧永久密码、旧角色或删除账号恢复后直接登录”的触发路径已消除。
- 回归发现：账号判定为 kept_deleted 后，跨模块删除不变量尚未重建，作为独立新 finding `IA-DES-R03-N001` 报告，不重开本 finding。

### IA-DES-R01-F008 — closed

- 证据：`design.md:584-628`。
- 每个公开 code 已固定 HTTP、message、重试性、唯一责任边界、最终日志级别和唯一 cause。
- 启动、重试、中间过程改用独立 internal event；不再以 `ERROR/WARN` 等斜杠值留给实现者。
- 未发现表外公开错误 code。

### IA-DES-R02-N001 — closed

- 证据：`design.md:979-1017`、`1349-1351`、`1371`、`1500-1507`。
- onboarding 保存已确认版本；最终开放事务同时锁定 User 和 current PrivacyNotice，要求版本相等。
- 两 Gate 间声明换版只重锁未完成 onboarding 的账号；已完成 onboarding 用户不被重新锁定，符合 r16 已接受边界。

### IA-DES-R02-N002 — closed

- 证据：`design.md:677-682`、`1368`、`1381-1389`。
- 三类桶有显式 `bucket_kind` 和 NOT NULL `bucket_key_digest`；唯一约束为二元键。
- 规范编码带长度前缀和版本化 HMAC；失败使用原子 upsert，衰减/清理使用同一键和行锁。
- 原 NULL 唯一性与并发计数分裂场景已消除。

### IA-DES-R02-N003 — closed

- 证据：`design.md:661-666`、`1395`、`1402-1424`、`1626`。
- `recovery_review_required` 已进入正式 CHECK 状态机。
- 普通同键同摘要只返回 `identity.operation_uncertain`，不能抢占或重放副作用；异摘要始终冲突。
- 只有 RecoveryCoordinator 可在行锁、旧状态条件、审计和 generation 递增下转 failed 或新 processing。

## 4. 新 findings

### [HIGH] IA-DES-R03-N001：kept_deleted 只恢复 User 结论，没有重建完整删除事务不变量

- 位置：`design.md:1369`、`1557`、`1613-1625`
- 违反依据：requirements r16 `US-IDENTITY-03.7`；ADR-0010；boundary S4/S5/S14；设计自身删除合同 `design.md:1185-1236`。
- 现状：灾备快照可能早于一次账号删除。逐账号核对若证明该账号应保持 deleted，只记录 `decision=kept_deleted` 并使账号无凭据；恢复合同没有调用 `DeleteAccount`/`ArticleAccountPort.OrphanArticles`，也没有重建 `author_id=NULL`、`publication_state=held`、`orphaned_from_user_id`、Session/凭据/标识生命周期、`retained_email` 和零文章清理等完整不变量。
- 触发：用户在备份后被删除，恢复到删除前快照；恢复责任人根据外部证据把账号判为 kept_deleted。
- 影响：服务开放后 User 可能是 deleted，但文章仍指向该 User、发布意图仍为 published，邮箱/手机号和 Identifier 仍是活动形态。虽然账号不能登录，删除后的文章处置、通知邮箱用途、隐私最小化和跨模块一致性均被破坏。
- 最小修复：kept_deleted 必须在维护事务中复用或等价执行完整删除协作合同，包括 article 无归属化/held/来源关联、标识与 PII 生命周期及审计；若恢复证据无法重建所需字段，应保持维护模式或将该账号列入不可开放的待处置清单，不能只改 User 状态。
- 验证：恢复“备份后删除且有文章/无文章/含违规文章”的快照，核 User、Identifier、Article、retained_email、AuditEvent 和零文章清理全部符合日常删除结果。

### [MEDIUM] IA-DES-R03-N002：recovery secret 的“一次性/禁止复用”没有持久可验证状态

- 位置：`design.md:924-940`、`1436`、`1494-1498`、`1556`、`1620-1621`
- 违反依据：修订 ADR-0007；安全凭据可管理、可审计和 fail-closed 要求。
- 现状：文档要求 recovery secret 一次性使用，缺失、错误或复用时停服；但 `RecoveryRun`、User 和配置合同没有 secret version/fingerprint、使用记录或轮换标识，无法证明某个 secret 未在此前 RecoveryRun 使用，也无法可靠检测跨恢复批次复用。
- 触发：运维在后续灾备恢复中重复注入同一个 `HAN_RECOVERY_INITIAL_ADMIN_SECRET`，或旧 secret 未从部署环境撤除。
- 影响：已经暴露或长期存在的恢复凭据可反复成为初始管理员信任根，削弱“一次性”安全边界和事后审计。
- 最小修复：为每次 RecoveryRun 使用独立 secret version/不可逆 fingerprint，并在受控存储记录 consumed；启动恢复时拒绝已使用版本/fingerprint，成功写入 recovery_bootstrap 后立即要求撤除配置。若只能依赖人工轮换，应明确降级风险和双人核验证据，不能声称系统可检测复用。
- 验证：同一 run 重放、跨 run 复用、bootstrap/recovery 相同、secret 未撤除和新 secret 正常恢复测试。

## 5. 静态验证与限制

- artifact digest 与 Lead 值一致。
- PlantUML：design 22 块、stories 1 块；起止标记成对，未发现禁用 `\n` 或 `linetype ortho`。
- design 中 13 个本地链接均存在。
- Story DAG 41 条边，未发现环；正文依赖与图未发现新增不一致。
- 当前无代码、schema、迁移或测试；未运行不存在的构建、lint、单元、集成或 E2E。
- 本机无可用 PlantUML CLI/JAR，未执行实渲染。
- Q1、Q2、Q3/Q4 数值、Q8/Q9/Q10 实际运行输入和检查命令保持上游已声明状态，本复审未将其重新判为缺陷。

## 6. 最终结论与后续条件

**最终结论：NOT PASS。**

- 先前开放项：5 项全部 closed。
- 新增风险：1 个 HIGH、1 个 MEDIUM。
- 未发现新增 CRITICAL。

在 `IA-DES-R03-N001` 关闭前，design 不得转为 `implementable`。后续复审需：

1. 提供新 design/stories 修订、完整 SHA-256 和规范 artifact digest；
2. 逐项回应 N001/N002；
3. 增加 kept_deleted 跨模块恢复矩阵和 recovery secret 复用验证设计；
4. 若修订 architecture/ADR/boundary，附新的 accepted checkpoint 及 S1～S14 回归。

本结论只针对固定 r03 对象，不代替用户/Lead 的风险接受、设计确认或实施授权。
