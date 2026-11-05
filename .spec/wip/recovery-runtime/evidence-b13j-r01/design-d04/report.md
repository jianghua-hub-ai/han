# B13J D04 证据报告 — 生产 root 装配合同与三个悬空设施

| 项 | 值 |
|---|---|
| item | `pending-B13J-D04`（gap-map 未登记项；本报告不写未登记 Task 编号，`S4.22`/`S4.23`/`S4.24` 仅在 gap-map 原文语境中作为候选出现） |
| 角色 | Arch-Designer（`add-arch-designer`），本项唯一作者 |
| Workflow | `development-delivery` 正常续接；未重新 Route、未新建批次、未改任何 Task 状态 |
| 记录时间 | 2026-11-01（Asia/Shanghai） |
| 工作区 | `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`（改前改后一致，未做 Git 写） |
| 产出性质 | Agent 文档成果；不是人的签字、不是独立评审、不是安全评审、不是生产批准 |

## 1. 任务与关闭条件映射

目标：为唯一 production root 固定 11 端口装配合同，并为部署独占、`ArticleAccountPort`、recovery secret candidate 指定唯一生产来源与精确落点（源：`gap-map.md` §2.6，`:141-161`）。

| # | 关闭条件（gap-map:160） | 本轮落点 | 状态 |
|---|---|---|---|
| 1 | 11 端口唯一生产来源可逐项列出，分类沿用 `journal`/`articleSnapshots`/`consumer`/`installation` 复用既有组件、`restore` 归 D03、`commitments`/`sources`/`anchor` 归 D02、`articleAccount`/`exclusive`/`secretCandidate` 归本项 | identity `:3601-3618`（11 行表 + 归类说明） | 已交付（文档口径） |
| 2 | 部署锁明确支持的 OS/平台与不支持时的拒绝 | identity `:3908`、`:3922`；ADR-0032（framework `:1050-1117`） | 已交付（文档口径） |
| 3 | article 侧拥有 canonical orphaning | article `:2432-2439`；identity `:1055`、`:3615` | 已交付（文档口径） |
| 4 | secret 只读候选且由 Journal 验密 | identity `:3916`、`:3918`、`:3617` | 已交付（文档口径） |
| 5 | 设计 Review PASS | 本轮只交付待评审稿，由独立 Reviewer 判定 | **未达成，不声称** |
| 6 | production 仍 closed 直到实现卡逐项关闭 | 全部 11 项状态列均为 closed/未交付；ADR-0032 不解除任何 closed | 保持 closed |

## 2. 输入与基线

### 2.1 改前快照（`snapshots/`，只复制未修改）

| 快照文件 | 对应源文件 | SHA256 | 字节 | 行数 | crlf / bareLF |
|---|---|---|---|---|---|
| `identity-before-r60.md` | `design-workspace/identity-auth/design.md` | `B07BAFC7CE1265A42B4F9A79C0F1489D3FCAAB50F47F1C02F0A1FCDA3C23AF99` | 847437 | 4924 | 169 / 4755 |
| `article-before-r28.md` | `design-workspace/article-management/design.md` | `4D46B307CB73C9F35A9146B35724970283B5A3BC7833F0B8EE8CC6854345128A` | 374925 | 3257 | 0 / 3257（纯 LF） |
| `framework-before-r32.md` | `decisions/framework.md` | `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12` | 155139 | 1049 | 1049 / 0（纯 CRLF） |
| `readme-before-r32.md` | `decisions/README.md` | `A67B00D5BF050E100EA5AEC9DEBBFB8E5DABCE4463448658D68A5B4792E0F706` | 20764 | 108 | 2 / 106 |

四份快照 SHA256/字节/行数/行尾比例与 D04 任务书 §3 的基线表逐项一致；无 BOM，末尾均为单个换行。

### 2.2 依据文档

- `gap-map.md`（SHA256 `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F`，26452 B）：§2.6 `pending-B13J-D04`（`:141-161`）；U5 部署独占（`:201`）、U6 `ArticleAccountPort` 归属（`:202`）、U7 secret candidate 是否单独开卡（`:203`）。
- ADR-0019（framework `:418` 起）：`JournalHeadAnchor` 独立故障域、不随 PostgreSQL 回退、Windows 生产 unsupported / POSIX only、UTC 微秒、fail-closed 失败分类——本轮硬边界，未改一字。
- ADR-0030（framework `:916` 起）：首期不提供生产 CAS Anchor provider，`enter`/`install` 保持 closed。
- ADR-0031（framework `:971` 起）：首期不提供生产 restore 工具与批准主体，`enter`/`resume` 的 DB 替换保持 closed；批准记录、专用执行身份、宿主隔离与原生耐久回执合同。
- 以上三段正文均由 framework 前 155139 字节的前缀哈希 `D74F09A0…1C12` 覆盖：该前缀与改前基线逐字节相同，证明未改 ADR-0019/0030/0031。

### 2.3 只读代码事实（用于落点准确性，未修改任何代码）

| 事实 | 位置 | 观察 |
|---|---|---|
| `ArticleAccountPort` / `ArticleAccountPortFunc` | `internal/identity/article_port.go:16-39` | port 以调用方 `pgx.Tx` 为参数，适配器只转发不拥有规则 |
| `OrphanArticlesCanonicalRepository` / `CanonicalOrphanRepository` | `internal/article/article_port.go:29-41` | 仓储接口与 PG 实现均在 article 包 |
| `OrphanArticlesCanonicalAuditor` | `internal/article/article_port.go:171` | 审计追加在调用方事务内 |
| `CommandService.OrphanArticlesCanonical` | `internal/article/command_service.go:342` | canonical orphaning 业务实现 |
| 生产 closed 占位 | `cmd/han-server/recovery_sources.go:641-662` | `recoveryClosedArticleAccount` 等 closed 实现 |
| `articleAccount` 注入点与 closed 原因 | `cmd/han-server/recovery_runtime.go:444`、`:610` | root 注入点；未注入即 closed |
| 部署独占现状 | `cmd/han-server/recovery_exclusive.go`（71 行） | 必须显式注入 acquire，否则 `ErrAdmissionUnsupported`；无 OS 后端、无 fallback |
| 锁键解析 | `cmd/han-server/recovery_config.go` | 解析 `HAN_RECOVERY_RUNTIME_LOCK_PATH`，拒绝 `journalPath == lockPath` |

## 3. 改动清单（逐文件、逐行）

### 3.1 `design-workspace/identity-auth/design.md`（r59 → r60，4924 → 4944 行）

| 行号（改后） | 类型 | 改前 | 改后 |
|---|---|---|---|
| `:5` | 整行替换 | `revision: identity-auth-design-ir01-r59` | `revision: identity-auth-design-ir01-r60` |
| `:38` | 整行替换 | `**设计版本**：` + 「`identity-auth-design-ir01-r59`」 | `**设计版本**：` + 「`identity-auth-design-ir01-r60`」 |
| `:7` | 不变 | `last_updated: 2026-11-01` | 同日增量，保持不变 |
| `:83` | 同行追加（不新增行） | 行尾止于 r59 段（长 1646 字符） | 追加 r60 段后长 2193 字符；登记输入 r59 SHA256 `B07BAFC7…AF99`、依据 ADR-0032 与 gap-map U5/U6/U7、仍 draft、人的精确接纳 unknown、不声明实现 |
| `:1055` | 整行替换 | 表格行「`ArticleAccountPort` / 删除账号时无归属化文章 / 使用同一事务；article 写自己的表」 | 生产唯一实现 = `internal/article` 的 `CommandService.OrphanArticlesCanonical`（仓储 `CanonicalOrphanRepository`）；由 production root 在 `cmd/han-server` 以 `identity.ArticleAccountPortFunc` 注入，使用调用方同一事务；identity 不得实现、复制或改写 article 表 SQL，也不得另建第二实现；未装配前为 closed 占位（`recovery_sources.go:641-662`），账号删除保持关闭 |
| `:3601-3618` | 新增 18 行（LF） | — | 标题「#### r60 生产 root 装配合同与 11 端口清单」、唯一装配清单说明（`:3603`）、表头与 11 行端口表（`:3605-3617`）、收尾空行（`:3618`）；全文见 §4 |
| `:3908` | 整行替换 | r59 `:3890`：`恢复域外绝对规范路径，已有稳定锁入口；normal/recovery/requeue 同一部署共用，未验证设施/ACL即拒绝；文件存在不等于持有锁` | POSIX 生产唯一候选为独占非阻塞 OS 文件锁（常驻 `O_CLOEXEC` fd 持有至进程退出，进程死亡即释放；非 `O_EXCL` 锁文件、非 DB advisory lock、非内存 mutex、非 caller bool／no-op）；三入口共锁；未验证设施／ACL 即拒绝启动；文件存在不等于持有锁，解锁不靠删文件 |
| `:3916` | 整行替换 | r59 `:3898`：`沿 S04 既有独立一次性秘密，仅 enter/确需恢复信任根的 resume 瞬时读取；normal/verify/已完成信任根不读；不复用 bootstrap/requeue 签名材料或 epoch` | 唯一生产来源为该环境变量本身、root 惰性只读候选适配器读取；无文件／配置文件／DB／argv／密钥管理服务替代来源，无代码默认值；仅 `enter` 与 Store 独立证明本 run 信任根仍未准备的 `resume` 瞬时读取；读取成功不等于验证，由 `Journal.VerifyBeginSecret` 持 retained Begin key 纯读验密，Runtime 再核来源／许可／ctx／独占才允许 `PrepareRecoveryTrust`；不持久化、不进日志／Trace，失败同样清零 |
| `:3918` | 整行替换 | r59 `:3900`：`RecoverySecretSource 仍是 root 的惰性ENV候选读取适配，不把读取成功当验证；…` | 保留原清零/Trace/deadline 合同，补候选唯一来源=环境变量本身、无替代来源与默认值、不单独开实现卡（U7）、随唯一 production root 装配承接、已准备/已改密零读取、不扩 ADR-0026 与 FileRecoveryJournal 平台矩阵 |
| `:3922-3923` | 新增 2 行（LF） | — | `:3922` 新增「**r60 部署独占的平台矩阵（ADR-0032）**」：POSIX 唯一候选、`O_CLOEXEC` 常驻 fd、进程死亡内核释放、恢复域外 owner-only/no-follow 单一文件、三入口共锁、设施缺失/ACL 不可验证/路径不合法/加锁失败一律退出码 2 且无 fallback、文件存在不等于持有锁、禁止 caller bool/no-op/fake、Windows 生产 unsupported（退出码 2，仅软件合同与确定性测试替身）、不扩 ADR-0026 与 FileRecoveryJournal 矩阵、`recovery_exclusive.go` 仍为 closed 占位；`:3923` 为段间空行 |

`article-management` 与 `decisions` 里的既有表述经核对（identity `:2345` PlantUML participant、`:2416` `OrphanArticlesCanonical` 方法行、`:3953`/`:3983` 等）与本轮新口径不冲突，未做改动。

### 3.2 `design-workspace/article-management/design.md`（r27 → r28，3257 → 3267 行，纯 LF）

| 行号（改后） | 类型 | 改前 | 改后 |
|---|---|---|---|
| `:5` | 整行替换 | `revision: article-management-design-ir01-r27` | `revision: article-management-design-ir01-r28` |
| `:7` | 整行替换 | `last_updated: 2026-10-31` | `last_updated: 2026-11-01` |
| `:37` | 整行替换 | `**设计版本**：`article-management-design-ir01-r27`` | `**设计版本**：`article-management-design-ir01-r28`` |
| `:69-70` | 新增 2 行 | — | r28 决策追溯条目：输入 r27 SHA256 `4D46B307…128A`，依据 D04 生产 root 装配合同与 gap-map U6，只固定 canonical orphaning 的 owner 与装配点；按升序置于 r27 条（`:71`）之前；`:70` 为空行 |
| `:2432-2439` | 新增 8 行（§3.4.6 子小节） | — | 「##### r28 canonical orphaning 生产落点」：owner = article 侧 `CommandService.OrphanArticlesCanonical` + `OrphanArticlesCanonicalRepository`/`CanonicalOrphanRepository` + `OrphanArticlesCanonicalAuditor`；语义保持既有（同一调用方事务、幂等置空作者、写 `held` 与来源关联、保留违规、返回 newly/already/pending、trace 以 ctx 为唯一权威）；装配点 = production root 在 `cmd/han-server` 以 `identity.ArticleAccountPortFunc` 桥接 `recovery_runtime.go:444`，`:610` 与 `recovery_sources.go:641-662` 现为 closed；identity 不得复制 SQL、不得第二仓储/第二事务；明确为**待实施合同**，不声称已接线或验证；`enter`/`install` 与 `enter`/`resume` 的 DB 替换分别按 ADR-0030/ADR-0031 保持 closed |

### 3.3 `decisions/framework.md`（新增 ADR-0032，1049 → 1117 行，纯 CRLF）

`:1050-1117` 新增 68 行，结构对齐 ADR-0031：

| 行号 | 内容 |
|---|---|
| `:1050` | 空行（ADR 前分隔） |
| `:1051` | `---` |
| `:1052` | 空行 |
| `:1053` | `<a id=adr-0032></a>` |
| `:1054` | `## ADR-0032: 部署独占锁采用 POSIX 独占文件锁，Windows 生产 unsupported (2026-11-01)` |
| `:1056` | 目的 / 仓库 / `修订: adr-0032-r01` |
| `:1058` | 状态：已接受（2026-11-01 用户 standing 授权采用建议值；确认来源为本轮 D04 任务指令；不构成人的接收、独立评审或生产批准） |
| `:1060` | 继承关系：只固定 `:3908`/`:3922`/`:3616`；不改 ADR-0019/0030/0031/0027；与整库 restore 语义不冲突 |
| `:1062` | 问题：要求明确但设施 unknown，`recovery_exclusive.go` 71 行 closed 占位，无 OS 后端/fallback |
| `:1064` | 决策：POSIX 独占非阻塞文件锁 + Windows 生产 unsupported（退出码 2、无 fallback），首期不引入 `O_EXCL`/advisory lock/mutex/caller bool/no-op/fake |
| `:1066-1084` | 现状 / 痛点 / 改造 / 效果，含 A–E 五方案对比表（A 独占文件锁选定；B `O_EXCL`、C DB advisory lock、D 进程内 mutex、E 无锁 均不选）与选 A 论证 |
| `:1086-1095` | 不变量与约束 8 条：平台矩阵、三入口共锁、路径与 ACL、持有与释放、禁止伪实现、失败分类、不改既有决定、不登记实现任务/不声称验证 |
| `:1097-1105` | 解除条件 7 条：实现、隔离与 ACL 证据、负向测试证据、三入口证据、独立评审与正式登记、人的生产批准、Windows 另立 ADR |
| `:1107-1109` | 不适用场景：anchor/CAS 选型、restore 工具与批准主体、runtime 编排、canonical orphaning 归属、secret candidate、跨主机互斥 |
| `:1111-1117` | 引用记录：identity `:3908`/`:3922`/`:3601`+`:3616`/`:83`；ADR-0019 `:418` 起、ADR-0030 `:916` 起、ADR-0031 `:971` 起；gap-map §2.6（`:141-161`）与 U5（`:201`）；决策授权 |

### 3.4 `decisions/README.md`（3 处，108 → 110 行）

| 行号（改后） | 位置 | 内容 |
|---|---|---|
| `:40` | `## 全局决策索引` 表 | 新增 ADR-0032 行，链接 `framework.md#adr-0032`，状态列与相邻行同风格（已接受 + 日期/来源 + 未实测说明） |
| `:46` | `## 领域文件表` 的 `framework.md` 行 | 在 ADR-0031 后追加 `、ADR-0032` |
| `:73` | `## 场景查询表` | 新增「选择或调整部署独占锁 OS 后端、平台支持矩阵、锁路径/ACL/持有语义、首期不支持 Windows 生产或解除条件 → framework.md（ADR-0032）、framework.md（ADR-0019、ADR-0027）」 |

既有 CRLF 行（`:36`、`:59`）未动；本轮新增/替换行沿用该文件主体 LF。

## 4. 11 端口装配清单（identity `:3601-3618` 镜像）

说明（`:3603`）：本表是唯一 production root 装配清单，绑定 `cmd/han-server` 的 `recoveryRootInputs` 与 `recovery.NewRuntime` 依赖集合；新增端口、替换来源或改变归属须先修订本节与对应 ADR；未在本表内登记的依赖不得注入 production root；`store`、`runs`、`genesis`、`identity`、`exit`、`lifecycle` 等既有依赖不在本 11 项内，不因本节改变。

| 端口 | 生产唯一来源 | 归属方 | 当前状态 |
|---|---|---|---|
| `journal` | 既有 `recovery.ContextRecoveryJournal` 受控文件 Journal（ADR-0019），root 以 `recoveryOpenJournal` 打开 | recovery／identity 既有组件 | 组件已建成；head anchor 未交付时构造为 closed（`recoveryClosedJournalAnchor`），生产状态变更仍 closed |
| `articleSnapshots` | 既有 article 锚定写者的 `recovery.VerifiedJournalReader` 只读快照面 | article（写者） | 组件已建成；未绑定读为 `recoveryClosedVerifiedJournalReader`，保持 closed |
| `consumer` | 既有 `recovery.RecoveryConsumer`（ADR-0027 维护幂等 resolve 消费顺序） | identity | 组件已建成；root 未注入时为 `recoveryClosedConsumer` |
| `installation` | 既有 `recovery.InstallationExecutor`（受控安装执行，ADR-0027） | identity／recovery | 组件已建成；root 未注入时为 `recoveryClosedInstallationExecutor` |
| `restore` | 生产 restore driver 首期不提供（ADR-0031）；root 使用 `recoveryClosedRestoreDriver` 占位 | cmd root（ADR-0031） | closed；`enter`／`resume` 的 DB 替换在 Begin 前拒绝 |
| `commitments` | 已登记 anchor adapter bundle 中的 `recovery.InstallationCommitmentStore`（ADR-0030 registry） | recovery（ADR-0030 边界） | 首期 registry 为空，解析为 `recoveryClosedPort`；生产 closed |
| `sources` | 已登记 anchor adapter bundle 中的 `recovery.InstallationSourceVerifier`（ADR-0030 registry） | recovery（ADR-0030 边界） | 首期 registry 为空，解析为 `recoveryClosedPort`；生产 closed |
| `anchor` | 已登记 adapter bundle 的 `recovery.AnchorReader`；首期不提供生产 CAS provider（ADR-0030） | cmd root（ADR-0030） | closed；`HAN_RECOVERY_ANCHOR_ADAPTER`／`HAN_RECOVERY_ANCHOR_CONFIG_PATH` 均无默认，未配置即不启动 anchor 能力 |
| `articleAccount` | article `CommandService.OrphanArticlesCanonical`（仓储 `CanonicalOrphanRepository`）经 `identity.ArticleAccountPortFunc` 桥接 | article 拥有语义；root 装配 | article 侧实现已存在；production root 未装配，`recoveryClosedArticleAccount` 使账号删除 closed |
| `exclusive` | POSIX 独占非阻塞 OS 文件锁（`HAN_RECOVERY_RUNTIME_LOCK_PATH`，ADR-0032） | cmd root／platform（ADR-0032） | 生产候选已定；当前无 OS 后端，`restoreDeploymentExclusive` 无 acquire 即 `ErrAdmissionUnsupported`；设施／ACL 未验证即拒启 |
| `secretCandidate` | `HAN_RECOVERY_INITIAL_ADMIN_SECRET` 环境变量本身，由 root 惰性只读候选适配器读取 | cmd root | 合同已定；root 未注入时为 closed reader（`recovery_runtime.go:631-640`）；读取不等于验证 |

## 5. 四项结论

1. **部署锁平台矩阵**：POSIX 是唯一生产候选——单机 OS 级独占非阻塞文件锁，`O_CLOEXEC` 常驻 fd 持有至进程退出、进程死亡由内核释放；锁路径须为恢复域外绝对规范路径、owner-only、no-follow、单一文件；`normal`/`recovery`/`requeue` 同一部署共用同一把锁；设施缺失、ACL 不可验证、路径不合法或加锁失败一律在任何 DB/CAS/文件状态变更前退出码 2，无 fallback、无降级；文件存在不等于持有锁，解锁不靠删文件；禁止 caller bool/no-op/fake。**Windows 生产 unsupported**（退出码 2），仅保留软件合同与确定性测试替身；本决定不扩 ADR-0026 的 Windows SID 矩阵与 FileRecoveryJournal 的 POSIX 矩阵。
2. **`ArticleAccountPort` 归属与落点**：canonical orphaning 的语义 owner 严格在 article 侧（`CommandService.OrphanArticlesCanonical` + `CanonicalOrphanRepository` + `OrphanArticlesCanonicalAuditor`，同一调用方事务）；唯一装配点是 production root 在 `cmd/han-server` 以 `identity.ArticleAccountPortFunc` 桥接，注入 `recovery_runtime.go:444`；identity 不得实现、复制或改写 article 表 SQL，也不得另建第二实现；未装配前维持 `recoveryClosedArticleAccount`（`recovery_sources.go:641-662`、`recovery_runtime.go:610`），账号删除 closed。
3. **secret candidate 语义**：唯一生产来源就是 `HAN_RECOVERY_INITIAL_ADMIN_SECRET` 环境变量本身，由 root 惰性只读候选适配器读取；无文件／配置文件／DB／argv／密钥管理服务替代来源，无代码默认值；只在 `enter` 与 Store 独立证明本 run 信任根仍未准备的 `resume` 瞬时读取，其余形态零读取；读取成功不等于验证——由 `Journal.VerifyBeginSecret` 持 retained Begin key 纯读验密，Runtime 再核来源／许可／ctx／独占才允许 `PrepareRecoveryTrust`；不持久化、不进日志／Trace，失败同样清零；不单独开实现卡（U7），随唯一 production root 装配承接。
4. **11 端口清单**：见 §4；唯一清单、唯一来源、归属与 closed 状态已逐项固定，且明确「未在表内登记的依赖不得注入 production root」。

## 6. 改前/改后实测表

| 文件 | 阶段 | SHA256 | 字节 | 行数 | crlf / bareLF | BOM | 末尾 | 控制字符 |
|---|---|---|---|---|---|---|---|---|
| `identity-auth/design.md` | 改前 | `B07BAFC7…AF99` | 847437 | 4924 | 169 / 4755 | 无 | 单 LF | 1（既有 TAB @466829，行 3303） |
| `identity-auth/design.md` | 改后 | `4D2FC4AC638094BEB1564DF041A973ABA29FE3E43B5D585117BCC586777994AB` | 854160 | 4944 | 169 / 4775 | 无 | 单 LF | 1（同一 TAB @468004，行 3303，内容逐字不变） |
| `article-management/design.md` | 改前 | `4D46B307…128A` | 374925 | 3257 | 0 / 3257 | 无 | 单 LF | 0 |
| `article-management/design.md` | 改后 | `639D23262DE0D5D292B10FC29D371A9A908C33BD320E901E6BAE4BE9C5647F15` | 377416 | 3267 | 0 / 3267 | 无 | 单 LF | 0 |
| `decisions/framework.md` | 改前 | `D74F09A0…1C12` | 155139 | 1049 | 1049 / 0 | 无 | 单 CRLF | 0 |
| `decisions/framework.md` | 改后 | `C1E5F80354A442D446E988AC60A45806ACC73F401563F19A17063B7B74A8B25C` | 168354 | 1117 | 1117 / 0 | 无 | 单 CRLF | 0 |
| `decisions/README.md` | 改前 | `A67B00D5…F706` | 20764 | 108 | 2 / 106 | 无 | 单 LF | 0 |
| `decisions/README.md` | 改后 | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` | 21472 | 110 | 2 / 108 | 无 | 单 LF | 0 |

补充实测：

- `framework.md` 前 155139 字节 SHA256 = `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12`，与改前整文件哈希一致 → 除末尾追加的 68 行外逐字节未变；本 ADR 未触及 ADR-0019/0030/0031 正文。
- `identity-auth/design.md` 行数 +20（4924→4944）：`:3601-3618` 新增 18 行（LF）、`:3922-3923` 新增 2 行（LF），crlf 仍为 169、bareLF 4755→4775（+20）；新增行全部使用 LF，与该段上下文既有行尾一致，未混入 CRLF。
- `article-management/design.md` 行数 +10（3257→3267）：`:69-70` 新增 2 行、`:2432-2439` 新增 8 行，全部 LF；crlf 仍为 0。
- `decisions/README.md` 行数 +2：`:40`、`:73` 各新增 1 行（LF），既有两处 CRLF（`:36`、`:59`）未动，故 crlf 仍为 2、bareLF 106→108。
- 控制字符扫描口径：`<32` 且非 13/10 的字节计数；identity 改后仅 1 个既有 TAB，article/framework/README 均为 0。

## 7. 差异行集合与「其余行逐字节未变」证明

### 7.1 差异行集合（`git diff --no-index --unified=0` 快照 vs 现文件）

| 文件 | 差异 hunk（改前坐标 → 改后坐标） |
|---|---|
| identity | `-5 +5`、`-38 +38`、`-83 +83`、`-1055 +1055`、`-3600,0 +3601,18`、`-3890 +3908`、`-3898 +3916`、`-3900 +3918`、`-3903,0 +3922,2` |
| article | `-5 +5`、`-7 +7`、`-37 +37`、`-68,0 +69,2`、`-2429,0 +2432,8` |
| framework | `-1049,0 +1050,68`（仅末尾追加） |
| readme | `-39,0 +40`、`-45 +46`、`-71,0 +73` |

### 7.2 反向还原验证（逐字节）

方法：对每个文件以现文件为底，按上表逆操作——删除新增行区间（按改后坐标从高到低删除，避免索引位移），再将声明替换行整行换回快照原文行——在临时目录重建并重算 SHA256，与快照比对。

| 文件 | 逆操作 | 重建 SHA256 | 与快照一致 |
|---|---|---|---|
| identity | 删 `:3922-3923`（2 行）、删 `:3601-3618`（18 行）；`5/38/83/1055/3908/3916/3918` 换回快照行 | `B07BAFC7CE1265A42B4F9A79C0F1489D3FCAAB50F47F1C02F0A1FCDA3C23AF99` | 是 |
| article | 删 `:2432-2439`（8 行）、删 `:69-70`（2 行）；`5/7/37` 换回快照行 | `4D46B307CB73C9F35A9146B35724970283B5A3BC7833F0B8EE8CC6854345128A` | 是 |
| framework | 删 `:1050-1117`（68 行） | `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12` | 是 |
| readme | 删 `:73`、删 `:40`；`:46` 换回快照 `:45` | `A67B00D5BF050E100EA5AEC9DEBBFB8E5DABCE4463448658D68A5B4792E0F706` | 是 |

临时副本位于 `%TEMP%\b13jd04rev2`（工作区外，不构成仓库改动）；四文件反向还原全部逐字节回到改前基线。附注：对 identity/article/readme 直接 `git apply -R` 因混合行尾与 UTF-8 patch 文本前置处理返回 1，故改用行级重建；framework（纯 CRLF、仅尾部追加）`git apply -R` 与行级重建两条路径都精确回到 `D74F09A0…1C12`。

## 8. 验收命令与结果

| # | 命令 | exit | 结果 |
|---|---|---|---|
| 1 | `rg -n "HAN_RECOVERY_RUNTIME_LOCK_PATH\|ArticleAccountPort\|HAN_RECOVERY_INITIAL_ADMIN_SECRET\|生产 root 装配" …/identity-auth/design.md` | 0 | 15 行；命中 `:1055`、`:2016`、`:2345`、`:2416`、`:3166`、`:3601`、`:3615-3617`、`:3908`、`:3916`、`:3918` 等 |
| 2 | `rg -n "ArticleAccountPort\|OrphanArticlesCanonical\|canonical" …/article-management/design.md` | 0 | 91 行；命中 `:69`、`:2432-2438` 等 |
| 3 | `rg -n "部署锁\|文件锁\|Windows\|POSIX\|ADR-" decisions/framework.md` | 0 | 102 行；命中 ADR-0032 `:1054-1117` 全段 |
| 4 | `git diff --check -- <4 文件>` | 0 | 无空白错误；仅有 Git 对 CRLF 工作区的提示性 warning（非错误） |

（命令 1–3 的行号与命中内容已实跑记录；`git diff --check` 的 warning 为既有工作区 `core.autocrlf` 提示，不涉及本轮内容。）

## 9. 未做事项、遗留风险与边界

**未做（明确不声称）**

- 未实现任何 OS 锁后端、平台探针、ACL 校验或双进程互斥；`recovery_exclusive.go` 仍是无 OS 后端的 closed 占位。
- 未接线 `ArticleAccountPort`：`recovery_runtime.go:444` 的注入点仍未被生产装配，`:610` 与 `recovery_sources.go:641-662` 仍为 closed。
- 未实现 secret candidate 的只读适配器；`recovery_runtime.go:631-640` 仍是 closed reader。
- 未做任何真实平台/ACL/崩溃释放/三入口互斥/备份恢复演练验证；未执行 Go 测试或集成测试。
- 未取得独立设计评审 PASS（本轮只交付待评审稿）、未取得人的精确接纳、未取得安全评审、未取得生产批准。
- production 仍 closed；Case18 与 S8.2 未通过、`blocked-upstream` 未解除；未登记任何 Task、未改 `tasks.md`/`plans/`/`stories.md`/`test-plan.md`。

**遗留风险 / 供后续参考**

1. `flock` 与 `fcntl` 的语义差异（如 NFS、fork 继承、同进程重复打开）必须在实现阶段以目标文件系统实测选定，并给出错误映射；本 ADR 只固定「OS 级独占非阻塞 + `O_CLOEXEC` 常驻 fd + 崩溃内核释放」这一层。
2. 锁路径所在目录的可替换性（父目录被写权限过宽）属于 ACL 证明范围，须在解除条件第 2 条以实测证据关闭。
3. `articleSnapshots` 的「只读快照面」与 article 写者的实际绑定形态，仍需在实现阶段确认无第二写路径。
4. recovery secret 的「Store 独立证明本 run 信任根仍未准备」依赖既有 Store 读取合同，本轮未新增该证明的实现细节。
5. gap-map U5 引用的 `design.md:3610`、`:3890`、`:3898-3900` 为 r59 坐标；r60 在 `:3601` 前插入 18 行后，对应落点为 `:3908`、`:3916`、`:3918`（部署独占要求行本身未位移的 `:3610` 语义由 `:3608`/`:3610` 邻域承接，本轮未改动该行）。

## 10. 声明

本报告与随附 `index.json`、`snapshots/` 均为 Arch-Designer（Agent）产出，属于文档成果：不是人的签字、不是独立评审结论、不是安全评审、不是生产批准，也不构成任何实现完成、平台验证或生产可用的证明。ADR-0032 的状态「已接受」仅表示本轮按用户 standing 授权采用建议值并写入文档，确认来源为本轮 D04 任务指令。

## 11. r02 返工（review-r01 处置，最终值以此段为准）

**触发**：`design-d04/review-r01/report.md`（SHA256 `51C91D48B55486375166267E0B1CED9CE2AFEB3878C94D79CB24427AF820751D` / 26602 B）判定设计 **CHANGES REQUIRED**（`B13J-D04-RV-01`，MEDIUM）与安全 **PASS WITH MINOR**；Lead 另追加 `RV-03`（R-V2 观察）。本节记录处置、最终哈希与差异集合；§3–§7 中 r01 的哈希/行号不再是最终值，最终值以本节及 `index.json` 的 `revision_r02` 为准。

### 11.1 RV-01 处置（`exclusive` 唯一生产来源）

采用 Lead 更正后的唯一口径（覆盖评审原建议的 (a) 选项）：

1. **唯一生产来源** = 已登记 anchor adapter bundle 四端口集合中的 `Exclusive`（`recovery_sources.go:425-430`）；`newRecoveryAnchorRegistry`（`:455-466`）要求四项均非 nil，缺一即 `ErrAdmissionBinding`；空、未登记或配置路径不可用 → 四项一起落 `recoveryClosedPort`（`:470-492`、`:494-532`，该 closed port 同时实现 `DeploymentExclusive`）。
2. `recoveryRootInputs.exclusive`（`recovery_runtime.go:565-567`）**只是合成/测试注入**，生产不得用它替代 registry 来源。
3. `restoreDeploymentExclusive`（`recovery_exclusive.go:15-42`）是 **root 侧唯一 `DeploymentExclusive` 适配器形态**，必须显式提供 acquire 否则 `ErrAdmissionUnsupported`，路径须绝对、`filepath.Clean` 相等、无前后空白；**当前没有生产构造点**（`main.go:634-636` 未设置 `exclusive`）。
4. 诚实结论：要获得部署锁必须先有已登记的 adapter bundle（ADR-0030 首期 registry 恒空），因此当前 production `exclusive` 只能是 `recoveryClosedPort`，`normal`/`recovery`/`requeue` 的生产装配一律 closed；解除顺序 = ADR-0030 adapter 交付 → ADR-0032 锁后端实现 → 正式 Task 登记，该依赖链已写入 ADR-0032 §不变量与约束第 9 条。

**落点与前后原文**

**identity `:3616`（整行替换）**

```text
改前：| `exclusive` | POSIX 独占非阻塞 OS 文件锁（`HAN_RECOVERY_RUNTIME_LOCK_PATH`，ADR-0032） | cmd root／platform（ADR-0032） | 生产候选已定；当前无 OS 后端，`restoreDeploymentExclusive` 无 acquire 即 `ErrAdmissionUnsupported`；设施／ACL 未验证即拒启 |
改后：| `exclusive` | 已登记 anchor adapter bundle 四端口集合中的 `recovery.DeploymentExclusive`（ADR-0030 registry；bundle 四项须齐备，缺一即 `ErrAdmissionBinding`；空、未登记或配置路径不可用即四项一起落 `recoveryClosedPort`）；root 侧唯一 `DeploymentExclusive` 适配器形态为 `restoreDeploymentExclusive`（ADR-0032），当前无生产构造点，必须显式提供 acquire 否则 `ErrAdmissionUnsupported` | cmd root／platform（ADR-0032） | 生产 closed：`newRecoveryAnchorRegistry(nil)` 使 registry 恒空，`exclusive` 只能是 `recoveryClosedPort`，`normal`／`recovery`／`requeue` 生产装配一律 closed；`recoveryRootInputs.exclusive` 仅为合成／测试注入，生产不得用它替代 registry 来源；设施／ACL／文件系统锁语义未验证即拒启 |
```

**framework `:1060`（子句扩展，其余字节不变）**

```text
改前（片段）：…以及 11 端口清单中 `exclusive` 行（r60 `:3616`）的归属与状态；…
改后（片段）：…以及 11 端口清单中 `exclusive` 行（r60 `:3616`）的归属与状态：该行唯一生产来源是已登记 anchor adapter bundle 四端口集合中的 `recovery.DeploymentExclusive`（bundle 四项缺一即 `ErrAdmissionBinding`；空、未登记或配置路径不可用即四项一起落 `recoveryClosedPort`），`recoveryRootInputs.exclusive` 只是合成/测试注入、生产不得用它替代 registry 来源，`restoreDeploymentExclusive` 只是 root 侧唯一适配器形态且当前无生产构造点，故当前生产 `exclusive` 只能是 `recoveryClosedPort`；…
```

**framework `:1096`（新增 §不变量与约束第 9 条）**

```text
9. **解除顺序与来源耦合**：本 ADR 不隐去 `exclusive` 的生产来源依赖——要获得部署锁必须先有已登记的 anchor adapter bundle 交付（ADR-0030 首期 registry 恒空、生产 CAS provider 未提供），再实现本 ADR 的 POSIX 锁后端并取得隔离/ACL 证据，最后由正式 Task 规划登记装配与验证；该顺序完成前 `exclusive` 的生产装配只能是 `recoveryClosedPort`，`normal`/`recovery`/`requeue` 一律 closed，不得以合成/测试注入或 registry 之外的路径替代唯一来源。
```

**framework `:1114`（原 `:1113`，引用记录首条子句更新）**

```text
改前（片段）：11 端口清单 `:3601` 的 `exclusive` 行 `:3616` 登记生产唯一来源与当前 closed 状态；
改后（片段）：11 端口清单 `:3601` 的 `exclusive` 行 `:3616` 登记唯一生产来源为已登记 adapter bundle 的 `Exclusive`、合成/测试注入不得替代 registry 的唯一性排除与当前 closed 状态；
```

一致性核对：identity `:3612`/`:3613`（commitments/sources）仍为“已登记 anchor adapter bundle 中的 …（ADR-0030 registry）／首期 registry 为空，解析为 `recoveryClosedPort`”，与本轮 `:3616` 的同一 registry 口径一致，未改动。

### 11.2 RV-02 处置（article `:69` 依据错配）

`article-management/design.md:69` 原句“依据本轮 D04 生产 root 装配合同（ADR-0032 及 11 端口清单）与 gap-map U6”改为“依据同轮 D04 的 11 端口清单（identity `:3615` 的 `articleAccount` 行）与 gap-map U6”，删除对 ADR-0032 的引用（ADR-0032 `:1110` 明确不适用于 `ArticleAccountPort`）。article `:2432-2439` 正文原本未引用 ADR-0032（`rg -n "ADR-0032"` 改后零命中），无需改动。该行为同行替换，行数不变。

### 11.3 RV-03 处置（锁文件所在文件系统类别）

与 ADR-0019 `:471` 生产 `FileRecoveryJournal` 的“本地 POSIX 文件系统”前提同口径复用，三处各加最小一句：

**framework `:1088`（§不变量与约束第 1 条末尾追加）**

```text
锁文件必须位于已验证支持可靠 OS 独占锁语义的本地 POSIX 文件系统；共享/NFS 类在锁语义未证明前不宣称支持（与 ADR-0019 生产 FileRecoveryJournal 的本地文件系统前提同口径，不扩其支持矩阵）。
```

**identity `:3908`（同行追加）**

```text
…no-follow 的单一文件，且须位于已验证支持可靠 OS 独占锁语义的本地 POSIX 文件系统（共享／NFS 类在锁语义未证明前不宣称支持）；POSIX 生产唯一候选…
```

**identity `:3922`（同行追加）**

```text
…no-follow、单一文件，且须位于已验证支持可靠 OS 独占锁语义的本地 POSIX 文件系统（共享／NFS 类在锁语义未证明前不宣称支持），normal／recovery／requeue 三入口共用同一把锁。…
```

未据此新增 Windows 例外，未扩大 FileRecoveryJournal 矩阵；framework `:1114` 的引用记录同步加入该前提。

### 11.4 最终实测（4 文件）

| 文件 | SHA256 | 字节 | 行数 | crlf / bareLF | BOM | 末尾 | 控制字符 |
|---|---|---|---|---|---|---|---|
| `identity-auth/design.md` | `FED48FA9908F9EADD970A3E57B0B448181C51030A4460B437852009722F34D8D` | 854991 | 4944 | 169 / 4775 | 无 | 单 LF | 1（既有唯一 TAB，字节偏移 468004、行 3303，未位移） |
| `decisions/framework.md` | `58C97FC06361022AA48EF737189EF3EB069A7AAD25772BCA4C80FE4D1D5E1F8E` | 169890 | 1118 | 1118 / 0 | 无 | 单 CRLF | 0 |
| `article-management/design.md` | `690015BB5C3B4800539F98C5B75115A5A20B5C161ECE7422F9F271840F7C55A7` | 377424 | 3267 | 0 / 3267 | 无 | 单 LF | 0 |
| `decisions/README.md` | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` | 21472 | 110 | 2 / 108 | 无 | 单 LF | 0 |

- identity 行数不变（4944），crlf/bareLF 比例不变（169 / 4775），本轮三处改动均为同行替换/同行追加。
- framework 前 155139 字节 SHA256 仍为 `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12`，ADR-0019/0030/0031 正文逐字节未变。
- `README.md` 与 r01 基线哈希/字节完全一致，本轮零改动。
- framework 最终复测为纯 CRLF（crlf=1118、bareLF=0）、无 BOM、末尾单 CRLF、控制字符 0；§11.4 表格中的 SHA256 为该最终值。

### 11.5 差异行集合（vs `snapshots-r02/`）

方法：`git diff --no-index --unified=0 <snapshot> <working file>`。

| 文件 | 差异行集合 |
|---|---|
| identity | `-3616 +3616`、`-3908 +3908`、`-3922 +3922` |
| framework | `-1060 +1060`、`-1088 +1088`、`-1095,0 +1096`（新增）、`-1113 +1114` |
| article | `-69 +69` |
| README | 无（哈希与基线一致） |

`framework.md` 之外无其他行变化；identity/article 的行数、比例与既有 TAB 偏移均证明未发生重排或整文件重写。

### 11.6 行号位移对照

`framework.md` 在 `:1095` 之后新增 1 行，故 `:1096` 起的行号 +1：

| 锚点 | r01 | r02 |
|---|---|---|
| §约束新增第 9 条 | — | `:1096` |
| `### 解除条件` | `:1097` | `:1098` |
| 解除条件 1–7 | `:1099-1105` | `:1100-1106` |
| `### 不适用场景` | `:1107` | `:1108` |
| `### 引用记录` | `:1111` | `:1112` |
| 引用记录首条 | `:1113` | `:1114` |
| ADR-0032 anchor / 标题 | `:1053` / `:1054` | `:1053` / `:1054`（未变） |

identity 与 article 无行号位移；ADR-0032 内引用的 identity 行号（`:3908`、`:3922`、`:3601`、`:3616`、`:83`）仍为改后实测有效值。

### 11.7 反向还原验证

方法：以改后文件为底，按本节逆操作（identity `:3616` 整行换回快照行、两处文件系统子句换回；framework 删除新增第 9 条行并换回 `:1060`/`:1088`/`:1114` 三处子句；article `:69` 换回原句）在内存重建并重算 SHA256。

| 文件 | 重建 SHA256 | 与 `snapshots-r02/` 基线 |
|---|---|---|
| identity | `4D2FC4AC638094BEB1564DF041A973ABA29FE3E43B5D585117BCC586777994AB` | 一致 |
| framework | `C1E5F80354A442D446E988AC60A45806ACC73F401563F19A17063B7B74A8B25C` | 一致 |
| article | `639D23262DE0D5D292B10FC29D371A9A908C33BD320E901E6BAE4BE9C5647F15` | 一致 |
| README | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` | 一致（未改） |

`snapshots-r02/` 四份改前副本 SHA256/字节：identity `4D2FC4AC…94AB`/854160；article `639D2326…7F15`/377416；framework `C1E5F803…B25C`/168354；README `2EE124AE…6B97`/21472。

### 11.8 验收命令（实跑 exit code）

| # | 命令 | exit | 结果 |
|---|---|---|---|
| 1 | `rg -n "HAN_RECOVERY_RUNTIME_LOCK_PATH\|ArticleAccountPort\|HAN_RECOVERY_INITIAL_ADMIN_SECRET\|生产 root 装配" …/identity-auth/design.md` | 0 | 14 行命中（`:3616` 改写后不再包含锁路径字面量，故比 r01 少 1 行） |
| 2 | `rg -n "ArticleAccountPort\|OrphanArticlesCanonical\|canonical" …/article-management/design.md` | 0 | 91 行命中 |
| 3 | `rg -n "部署锁\|文件锁\|Windows\|POSIX\|ADR-" decisions/framework.md` | 0 | 103 行命中（新增第 9 条 +1） |
| 4 | `git diff --check -- <4 文件>` | 0 | 无空白错误；仅 Git 对 CRLF 工作区的提示性 warning |

### 11.9 本轮未做（明确不声称）

- 未实现 OS 锁后端、平台探针、ACL/文件系统证明或双进程互斥；`restoreDeploymentExclusive` 仍无生产构造点。
- 未登记任何 Task、未接线 anchor adapter registry 或 `ArticleAccountPort`；production 仍 closed。
- 未取得独立设计评审 PASS、安全签收、人的精确接纳或生产批准；不声称 Case18/S8.2 通过，不解除 `blocked-upstream`。

### 11.10 证据文件最终一致性（索引补正）

- 顶层 `diff_line_sets` 与 `revert_verification` 记录的是初始 D04 轮（identity r60 / article r28 / framework r32）相对 `snapshots/` 的结果；为避免与 r02 混淆，二者各新增 `scope` 字段说明所指轮次，r02 对应记录仍在 `revision_r02.diff_line_sets_r02` 与 `revision_r02.revert_verification_r02`。
- `index.json` 的 4 条验收命令按实际执行原文记录（含完整文件路径）；此前 `git diff --check` 一项写作路径简写，已按实跑原文更正，exit code 与命中数不变。
- `index.json` 末尾不补换行，与 `../design-d02/index.json` 的既有风格对齐；本报告以单个 LF 结尾。
