# 架构决策索引

本目录按领域聚合追加 ADR：一个稳定领域一个文件，每条决策在归属文件末尾用 `---` 分隔。索引、领域文件表与场景查询表在此同步维护。

## 全局决策索引

| 编号 | 标题 | 状态 | 目的 | 领域 | 影响范围 | 引用次数 |
|---|---|---|---|---|---|---|
| ADR-0001 | 采用模块化单体而非多服务 | 已接受 | 架构设计 | 框架 | 全仓 | 0 |
| ADR-0002 | HTTP 框架采用 Go 标准库 net/http | 已接受 | 接口设计 | 框架 | 接入层、全部接口 | 0 |
| ADR-0003 | 数据库采用 PostgreSQL | 已接受 | 数据模型 | 框架 | 全仓数据访问 | 0 |
| ADR-0004 | 对外接口采用 REST + JSON | 已接受 | 接口设计 | 框架 | 全部对外接口 | 0 |
| ADR-0005 | 认证采用服务端会话与受保护的不透明 Session Cookie | 已接受 | 接口设计 | 身份 | 身份认证模块、接入层、会话状态、同源与 CSRF 防护 | 3 |
| ADR-0006 | 邮件采用强制 TLS 的 SMTP 直连并经适配器调用 | 已接受 | 数据流向 | 身份 | 身份认证模块、文章管理模块、邮件投递与处理方门禁 | 2 |
| ADR-0007 | 初始管理员 bootstrap 与灾备恢复凭据经独立环境变量注入 | 已接受 | 架构设计 | 身份 | 身份认证模块、恢复 run/secret、keyring 与信任根 | 4 |
| ADR-0008 | 按业务模块分包且配置仅来自环境变量 | 已接受 | 代码质量 | 框架 | 全仓目录结构、配置读取 | 0 |
| ADR-0009 | Go 版本固定为 1.27 | 已接受 | 类型一致性 | 框架 | 构建、工具链与已批准第三方依赖 | 2 |
| ADR-0010 | 文章可见性正交权威状态与作者归属可空 | 已接受 | 数据模型 | 内容 | 身份认证模块、文章管理模块、删除账号墓碑、无归属来源关联、数据模型与派生可见性 | 4 |
| ADR-0011 | 临时凭据交付 fail closed、SMTP 残余窗口治理与转移通知 best effort | 已接受 | 错误处理 | 身份 | 身份认证模块、文章管理模块、邮件投递、幂等与告警 | 3 |
| ADR-0012 | 密码与临时凭据采用 Argon2id 哈希 | 已接受 | 架构设计 | 身份 | 身份认证模块、凭据存储、构建依赖 | 1 |
| ADR-0013 | 删除账号标识保留与专用通知邮箱清理生命周期 | 已接受 | 数据模型 | 身份 | 身份认证模块、文章管理模块、唯一性与隐私清理 | 1 |
| ADR-0014 | 公共 REST 错误采用稳定结构与最小披露 | 已接受 | 错误处理 | 框架 | 全部 REST 接口、日志与客户端错误处理 | 1 |
| ADR-0015 | 高风险写接口采用作用域化幂等合同 | 已接受 | 接口设计 | 框架 | 管理员创建、密码重置、临时密码重发、停用与恢复 | 1 |
| ADR-0016 | 高风险动作写入 PostgreSQL 追加式审计事件 | 已接受 | 数据流向 | 框架 | 身份治理、凭据操作、文章治理、转移与永久删除 | 1 |
| ADR-0017 | 隐私确认、事务性邮件与部署处理门禁 | 已接受 | 架构设计 | 框架 | 注册、管理员创建、邮件、部署与撤回流程 | 1 |
| ADR-0018 | 数据库恢复后账号、凭据、会话与幂等状态统一 fail closed | 已接受 | 架构设计 | 框架 | 备份恢复、identity-owned RecoveryRunStore、状态机/事务、canonical 删除、账号权限、凭据、会话与幂等 | 5 |
| ADR-0019 | Recovery Journal 使用独立 CAS 头锚点、持久 keyring 与受控文件提供方防止回退和重放 | 已接受 | 架构设计 | 框架 | 灾备运行、provider-neutral Anchor port、生产 provider 后续 Gate、不可变 backup 绑定、key 轮换终结、opaque retirement capability、生产 retirement Gate、持久卷/keyring、平台支持、ACL、时间、OS 锁与 namespace 隔离 | 11 |
| ADR-0020 | 业务数据库外 Article Deletion Journal 保证永久删除跨备份恢复 | 已接受 | 数据模型 | 内容 | 文章永久删除、v1 tagged union、production inventory authority/provider/config/manifest、canonical fixture 与恢复重放 | 5 |
| ADR-0021 | PostgreSQL 18、pgx/v5 与项目内版本化 SQL 迁移基线 | 已接受 | 架构设计 | 框架 | 数据库版本、Go 数据访问、`HAN_DATABASE_DSN`、`HAN_HTTP_ADDR`、启动自动迁移、监听/Serve 失败与资源清理 | 3 |
| ADR-0022 | 生产可观测性采用标准库适配器与部署平台组合 | 已接受 | 架构设计 | 框架 | JSON 日志、Prometheus text 0.0.4、observer/auditor/shutdown 预算、双 server 生命周期与退出码、告警、Dashboard、值守/runbook 与 fail-closed composition | 4 |
| ADR-0023 | Password reset 采用持久操作屏障、恢复 worker 与认证消费者 fail-closed Gate | 已接受 | 架构设计 | 身份 | PasswordResetOperation、recovery job/outbox、租约 worker、Login/PasswordChange/Session resolve Gate、升级与回退 | 3 |
| ADR-0024 | Article Deletion Journal 真卷韧性验证不纳入本版交付范围 | 已接受 | 架构设计 | 内容 | `article-management-S4.3` 验证范围、Unix 独立持久卷韧性未验证边界、B7 批次 Gate | 1 |
| ADR-0025 | [测试项目中 identity-auth S3.3/S3.5 的有界验证豁免](framework.md#adr-0025) | 已接受 | 代码质量 | 框架 | 仅 S3.3 真实浏览器与 S3.5 平台/工具校验前提豁免；S3.6 原真实验证条件保留 | 3 |
| ADR-0026 | Password-reset dead-letter requeue 采用 Windows 实际进程 SID 与离线审批签名 | 已接受 | 接口设计 | 身份 | S10.5单requeue、Windows SID/离线审批；D/K合作截止与真实join可能越期、数值primary/全部fallback限制、粘性退出；部署/新实现验证未完成 | 2 |
| ADR-0027 | [全局恢复runtime采用框架编排、唯一Store与正向准入证明](framework.md#adr-0027) | 已接受（r02技术采用；[Kuhn独立R02](../../../../.spec/wip/recovery-runtime/evidence-architecture-r32-r01/review-r02/review-report.md)已关闭F001／新增0，仅文档；成果仍draft／未实现／人的接纳unknown） | 架构设计 | 框架 | recovery机制/identity唯一Store、article独立namespace锚定写者/受控协调/可信迁移/历史证据、同一runtime屏障、genesis/重启/消费；未实现 | 6 |
| ADR-0028 | [测试项目采用分级验证、模拟环境替代与 AI 代理评审签收](framework.md#adr-0028) | 已接受 | 代码质量 | 框架 | 全项目测试与验证范围、模拟环境替代、AI 代理评审签收、建议部署目标/责任人/租约/演练窗口；产品安全合同与清理/凭据/Git 底线不变 | 1 |
| ADR-0029 | [数据库身份分离与 append-only 审计 ACL 启动门禁](framework.md#adr-0029) | 已接受（用户 2026-10-31 选择路径 B；依据 ADR-0028 项目级授权；revision adr-0029-r04：r02 独立评审为窄范围 CHANGES REQUIRED，F001–F010 已关闭、F011 与新增项在 r03 收口，r03 改动仅 §5/§8 措辞与新增 §10；r03 ACL 子集独立复审为 PASS WITH MINOR，`S15-ACL-R03-F003` 指出 §5 默认权限口径错误，r04 errata 更正 §5 事实、补充 §7 短窗口披露并更正 r03 记录前提，未改决定约束；已实现，Go 单测与真实 PG 集成已执行并通过，集成仅覆盖单机合成角色/单 schema 权限矩阵；仍未取得人的签收、未真实部署/CI、未做备份恢复演练） | 架构设计 | 框架 | 双 DSN、迁移身份独立 pool、启动 ACL 自检、`provision.sql` 必选 + `grant-application.sql` out-of-band 重放、append-only 表 INSERT 正控与 UPDATE/DELETE/TRUNCATE 负控、search_path 同名关系边界（生效 search_path 内同名关系数须恰为 1，歧义或缺表分属不同错误类别）、fail closed；残余同进程迁移凭据风险、建表到启动 `REVOKE` 之间的短窗口（应用角色短暂持有两表 `UPDATE`/`DELETE`，listener 未启动） | 3 |
| ADR-0030 | [首期不提供生产 CAS JournalHeadAnchor provider](framework.md#adr-0030) | 已接受（2026-11-01 D02 架构决策授权；首期不提供生产 provider，S4.5 维持未交付，生产 enter/install 保持 closed；配置键必须显式且无隐式默认；无真实 provider 验证；2026-11-03 D09 登记现行修订 adr-0030-r20（r17 原稿经 r18–r20 三轮两路评审返工；r18 架构 CHANGES REQUIRED、安全 PASS WITH CONDITIONS；r19 架构 PASS WITH CONDITIONS（F015 MEDIUM）、安全 ACCEPT；r20 架构 ACCEPT（1 项 LOW 非阻断）、安全 ACCEPT）：生产恢复链在测试项目内不实施、B1/B3/B4/B5 记为产能外 unknown、按 ADR-0028 收口 S4.4/S2.2/S3.4/article-S1.4 的模拟执行层；S3.6 按 ADR-0025 与 ADR-0028 约束 8 维持 blocked／未派发、不纳入模拟范围；两路 r20 接受报告见 .spec/wip/recovery-runtime/evidence-b13j-d09/review-arch-r04/report.md（10,999 B / 5BFD12775FA965543F1F97D77FE73CC5A70F6DDD60456A9B015E16FF77EA330B）与 .spec/wip/recovery-runtime/evidence-b13j-d09/review-sec-r04/report.md（13,966 B / D6CC246EB4ADFBDE4312F99801745AF1385537F76EC5D057DFA08F2F682318D5），仅覆盖 r20 字节 F0655B2A…3F55E80，生产继续 closed，不构成人的成果接收与生产批准） | 架构设计 | 框架 | 生产 CAS Anchor provider 产品形态、`HAN_RECOVERY_ANCHOR_ADAPTER`/`HAN_RECOVERY_ANCHOR_CONFIG_PATH`、启动装配、S4.5 未交付与生产 closed | 1 |
| ADR-0031 | [首期不提供生产 restore 工具与批准主体，固定受批准执行与宿主隔离合同](framework.md#adr-0031) | 已接受（2026-11-01 D03 架构决策授权；首期不提供生产 restore 工具与生产批准主体，生产 enter/resume 的 DB 替换保持 closed；固定批准记录、专用执行身份、root 权限边界、宿主隔离与原生回执不变量；现行修订 adr-0031-r12：2026-11-02 两路独立复审已接受（PASS）——架构 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-arch-r06/report.md` 19,778 B / 171 行 / `705806BABF4C65A352E9442D43AD5D1D060A135CCA9C195A89C959D7650163C7`，BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 0、开放项 0；安全 `.spec/wip/recovery-runtime/evidence-b13j-d06/review-sec-r06/report.md` 25,883 B / 218 行 / `12D55EBCCA5282FF703BBAE10F42A5601429E290EF8B5F02C04368AB8BCDF08A`，BLOCKER 0 / HIGH 0 / MEDIUM 0 / LOW 1（ADR-S06-01 OPEN，design 侧错误码一致性，安全影响为零）；接受登记 `.spec/wip/recovery-runtime/evidence-b13j-d06/acceptance-adr-0031-r12.md` 4,848 B / 46 行 / `F04E65668A80E9796FD7BEF057C2CBE92817FFC0D1CB9018D7E140A30368497D`；该接受只解除实施/验证卡的派发、认领与开始改动限制（framework.md:1174），不解除生产 closed、不表示约束 8 全项满足、不构成人的接纳；无真实工具/备份/演练验证） | 架构设计 | 框架 | 生产 restore 工具与可执行文件批准、备份来源批准、执行主体与 root 权限边界、进程组/no-follow/ENV allowlist 宿主隔离、原生耐久回执、见证信任根/端点约束/独立性、POSIX/Windows 矩阵、首期不提供与解除条件 | 1 |
| ADR-0032 | [部署独占锁采用 POSIX 独占文件锁，Windows 生产 unsupported](framework.md#adr-0032) | 已接受（2026-11-01 D04 用户 standing 授权采用建议值；POSIX 独占非阻塞文件锁、Windows 生产 unsupported；无 OS 后端/ACL/双进程互斥实测） | 架构设计 | 框架 | 部署独占 OS 后端、平台支持矩阵、锁路径/ACL/持有语义、三入口共锁、首期 Windows 生产 unsupported 与解除条件 | 1 |
| ADR-0033 | [Article 真实 PostgreSQL 容量/性能补测不纳入本版交付范围](content.md#adr-0033) | 已接受 | 架构设计 | 内容 | `article-management-S2.4` 验证范围、真实 PG calibration/small/medium 容量与性能未验证边界、B11 批次 Gate | 1 |
| ADR-0034 | [Case18 恢复运行结果到准入的门禁不纳入本版验证范围](content.md#adr-0034) | 已接受 | 架构设计 | 内容 | `C18-COV-F002` 关闭理由、`failed`/`expired`/`undetermined` 运行结果门禁未验证边界、`article-management-S8.2` 局部阻塞 | 1 |
| ADR-0035 | [Case18 真实部署 E2E 与双系统验证不纳入本版交付范围](content.md#adr-0035) | 已接受 | 架构设计 | 内容 | `article-management-S8.2` Case18 第 5 项关闭理由、真实部署/双系统未验证边界 | 1 |
| ADR-0036 | [测试项目中部署后验证类任务经决策不验证](framework.md#adr-0036) | 已接受 | 代码质量 | 框架 | `identity-auth-S3.4`、`identity-auth-S3.6`、`article-management-S1.4` 的部署后验证范围；6 张生产恢复实现卡不受影响 | 1 |
| ADR-0037 | [生产恢复实现链六张任务不纳入本版交付范围](framework.md#adr-0037) | 已接受 | 架构设计 | 框架 | `identity-auth-S4.5`/`S4.6`/`S4.23`/`S4.24a`/`S4.24b`/`S4.22` 的交付范围；生产 fail-closed 与 ADR-0030/0031/0032 门禁不变 | 1 |
| ADR-0038 | [合成 loopback/example DSN 不视为真实凭据](framework.md#adr-0038) | 已接受 | 代码质量 | 框架 | 本机/保留域名的合成测试 DSN 可入库；真实凭据、私钥、token 仍禁止 | 1 |

## 领域文件表

| 领域文件 | 领域 | 包含决策 |
|---|---|---|
| framework.md | 框架与技术栈 | ADR-0001、ADR-0002、ADR-0003、ADR-0004、ADR-0008、ADR-0009、ADR-0014、ADR-0015、ADR-0016、ADR-0017、ADR-0018、ADR-0019、ADR-0021、ADR-0022、ADR-0025、ADR-0027、ADR-0028、ADR-0029、ADR-0030、ADR-0031、ADR-0032、ADR-0036、ADR-0037、ADR-0038 |
| identity.md | 身份、会话与凭据 | ADR-0005、ADR-0006、ADR-0007、ADR-0011、ADR-0012、ADR-0013、ADR-0023、ADR-0026 |
| content.md | 文章与内容治理 | ADR-0010、ADR-0020、ADR-0024、ADR-0033、ADR-0034、ADR-0035 |

## 场景查询表

| 场景 | 查询位置 |
|---|---|
| 新增或调整对外接口 | framework.md（ADR-0002、ADR-0004） |
| 新增业务模块或调整包结构 | framework.md（ADR-0001、ADR-0008） |
| 变更数据库、数据库版本、Go 驱动、生产 DSN、迁移机制、启动门禁或数据约束 | framework.md（ADR-0003、ADR-0021）、content.md（ADR-0010） |
| 调整数据库应用/迁移身份、`HAN_DATABASE_DSN`/`HAN_MIGRATION_DATABASE_DSN`、provision/grant 权限资产、search_path 同名关系歧义或 append-only 审计 ACL 启动门禁 | [framework.md（ADR-0029）](framework.md#adr-0029)、framework.md（ADR-0021、ADR-0016、ADR-0019、ADR-0022） |
| 调整生产 HTTP 监听地址、地址格式、监听/Serve 失败或资源清理 | framework.md（ADR-0021） |
| 调整生产日志 sink/留存、metrics 线协议/exposition/buckets、observer/auditor/shutdown 预算、双 server 生命周期或退出码、告警、Dashboard、paging、值守/runbook 或 composition root | framework.md（ADR-0022） |
| 查询测试项目 identity-auth S3.3 真实浏览器、S3.5 平台/工具校验豁免及 S3.6 保留范围 | [framework.md（ADR-0025）](framework.md#adr-0025)、framework.md（ADR-0022） |
| 查询或变更测试项目的验证分级、模拟环境替代、AI 代理评审签收、建议部署目标/责任人/租约/演练窗口 | [framework.md（ADR-0028）](framework.md#adr-0028)、framework.md（ADR-0025） |
| 调整 Session Cookie、CSRF、会话轮换、登录态或撤销机制 | identity.md（ADR-0005） |
| 调整新管理员创建、密码重置、临时密码重发、SMTP 残余窗口或文章转移通知的失败语义 | identity.md（ADR-0006、ADR-0011） |
| 调整 PasswordResetOperation、recovery job/outbox、租约 worker、Login/PasswordChange/Session resolve Gate、dead-letter、升级顺序或禁止回退合同 | identity.md（ADR-0023）、framework.md（ADR-0015、ADR-0018） |
| 调整 S10.5 单requeue授权/OS主体/签名、七ENV、合作预算与实际join/粘性越期、数值primary及全部fallback维护DSN支持或恢复epoch | identity.md（ADR-0026 r04）；安全结果权威仍见ADR-0023 |
| 调整初始凭据注入方式 | identity.md（ADR-0007） |
| 调整灾备恢复信任根、账号恢复 Gate、RecoveryRun owner/状态机或恢复凭据 | identity.md（ADR-0007）、framework.md（ADR-0018） |
| 调整全局runtime归属/正向准入/genesis/历史关闭证明、article namespace Anchor写入/迁移/readonly revision、部署独占或运行中503/worker屏障及Consume开放联动 | [framework.md（ADR-0027 r02）](framework.md#adr-0027)、framework.md（ADR-0018/0019）、content.md（ADR-0020） |
| 调整 Recovery Journal 提供方、CAS 头锚点、接口、持久卷或恢复防重放 | framework.md（ADR-0019） |
| 调整 S4.2 Anchor port/测试替身交付边界、生产 CAS provider 任务责任、装配或生产就绪 Gate | framework.md（ADR-0019） |
| 选择或调整生产 CAS Anchor provider 产品形态、独立故障域、`HAN_RECOVERY_ANCHOR_ADAPTER`/`HAN_RECOVERY_ANCHOR_CONFIG_PATH`、启动装配、首期不提供或解除条件 | [framework.md（ADR-0030）](framework.md#adr-0030)、framework.md（ADR-0019） |
| 选择或调整生产 restore 工具、备份来源与可执行文件批准主体、执行主体与 root 权限边界、宿主隔离（进程组/no-follow/ENV allowlist/无 shell）、原生耐久回执、首期不提供或解除条件 | [framework.md（ADR-0031）](framework.md#adr-0031)、framework.md（ADR-0027、ADR-0018） |
| 选择或调整部署独占锁 OS 后端、平台支持矩阵、锁路径/ACL/持有语义、首期不支持 Windows 生产或解除条件 | [framework.md（ADR-0032）](framework.md#adr-0032)、framework.md（ADR-0019、ADR-0027） |
| 调整 Recovery Journal 的 run/backup 绑定、三方对账、key 轮换终结或退役条件 | framework.md（ADR-0019） |
| 调整 KeyRetirementCapability 验证协议、签发/消费责任、生产退役 Gate 或与 S4.5 的范围隔离 | framework.md（ADR-0019） |
| 调整生产 FileRecoveryJournal 的操作系统/文件系统支持、owner-only ACL/DACL 或 fail-closed 门禁 | framework.md（ADR-0019） |
| 调整 Recovery Journal 时间精度、文件替换协议或 OS 锁生命周期 | framework.md（ADR-0019） |
| 调整文章永久删除、Deletion Journal、tombstone 保留或恢复重放 | content.md（ADR-0020）、framework.md（ADR-0019） |
| 调整 Article production backup inventory authority、`HAN_ARTICLE_BACKUP_INVENTORY_PATH`、文件安全、manifest/canonical digest、刷新或 article S4.2/S4.4/S4.5 分工 | content.md（ADR-0020） |
| 查询或变更 Article Deletion Journal 真卷韧性（受支持 Unix 独立持久卷 `flock`/`fsync`/崩溃/磁盘故障）验证范围、`article-management-S4.3` 关闭理由或 B7 批次 Gate | content.md（ADR-0024）、content.md（ADR-0020） |
| 查询或变更 article 真实 PostgreSQL 容量/性能补测（calibration/small/medium/large）、`article-management-S2.4` 关闭理由或 B11 批次 Gate | [content.md（ADR-0033）](content.md#adr-0033)、content.md（ADR-0020） |
| 查询或变更 Case18 恢复隔离 `failed`/`expired`/`undetermined` 运行结果门禁、`C18-COV-F002` 关闭理由或 `article-management-S8.2` 局部阻塞 | [content.md（ADR-0034）](content.md#adr-0034)、framework.md（ADR-0018、ADR-0027） |
| 查询或变更 Case18 真实部署 E2E / 双系统验证范围、`article-management-S8.2` 第 5 项关闭理由 | [content.md（ADR-0035）](content.md#adr-0035)、content.md（ADR-0034） |
| 查询或变更测试项目部署后 identity/article E2E、production observability 演练的验证范围、3 张卡关闭理由或 6 张生产恢复卡阻塞边界 | [framework.md（ADR-0036）](framework.md#adr-0036)、framework.md（ADR-0028、ADR-0025） |
| 查询或变更生产恢复实现链（anchor provider、key-retirement、anchor/CAS bundle、restore 接线、POSIX 锁、production root）的交付范围或六卡关闭理由 | [framework.md（ADR-0037）](framework.md#adr-0037)、framework.md（ADR-0030、ADR-0031、ADR-0032） |
| 判断测试/证据文件中的 DSN 是否可作为凭据提交，或调整合成值的 host/口令边界 | [framework.md（ADR-0038）](framework.md#adr-0038)、framework.md（ADR-0028） |
| 调整文章发布意图、违规状态、作者归属、删除账号墓碑、无归属来源关联或派生可见性 | content.md（ADR-0010） |
| 调整密码哈希算法或参数治理 | identity.md（ADR-0012）、framework.md（ADR-0009） |
| 调整删除账号标识复用或通知邮箱清理 | identity.md（ADR-0013） |
| 调整公共 REST 错误结构或敏感信息披露 | framework.md（ADR-0014） |
| 调整高风险写接口幂等语义 | framework.md（ADR-0015） |
| 调整高风险动作审计范围、事务或留存机制 | framework.md（ADR-0016） |
| 调整隐私确认、事务性邮件用途或部署处理门禁 | framework.md（ADR-0017） |
| 调整备份恢复后的会话、凭据或幂等失效合同 | framework.md（ADR-0018） |
| 升级 Go 版本或引入其他第三方依赖 | framework.md（ADR-0009、ADR-0021） |
| 绑定正式远程或调整 Go module path | framework.md（ADR-0021） |

## 引用记录规则

引用次数从各 ADR 的「引用记录」章节派生，不在此处手工计次。只读查询不写入引用记录；只有获授权的实质采用或修订才登记，并同步受影响的设计正文与索引状态。

## 编号规则

新决策编号为全部领域文件中的最大编号加一。新增稳定领域时新建领域文件并同步本索引三张表。不因框架升级自动拆分、合并或重命名既有领域文件。

---

> **产出归属说明**：原确认来源保持不变。2026-10-20 当前 Arch-Designer 原位修订 ADR-0020 的最终 tagged union/fixture，并同步 architecture r12；2026-10-21 当前 Arch-Designer 追加 ADR-0021 并同步 architecture r13-r15。2026-10-22 当前 Arch-Designer 根据用户明确确认，原位修订 ADR-0018/0019 并同步 architecture r16-r19。2026-10-23 当前 Arch-Designer 根据用户明确授权“对待确认或决策项采用建议值”，追加 ADR-0022 并同步 architecture r20；同日依据独立评审 F001-F003 与 Lead 固定方案原位修订 ADR-0022 并同步 architecture r21，再按用户明确要求固定退出语义并同步 architecture r22；随后按独立复审 r02 仅同步现役 design r20 追溯与 Manifest 检查命令状态，形成 architecture r23。容量目标仍待实测，且不宣称实现、检查执行或部署完成。2026-10-24 当前 Arch-Designer 根据用户明确授权“对待确认或决策项采用建议值并继续”且本轮指定方案 A，追加 ADR-0023 并同步 architecture r25、identity design r35 与 stories r11；F002 仍为 Major OPEN，B6C/S10.1 仍 blocked-for-design-revision，B6D 不得启动，不宣称实现、测试、部署、独立评审或 finding 关闭。2026-10-25 当前 Arch-Designer 依据同一明确授权与 article design r13 独立评审 F001，原位修订 ADR-0020 并同步 architecture r28，固定恢复域外只读文件 inventory provider、`HAN_ARTICLE_BACKUP_INVENTORY_PATH`、strict v1 manifest、owner/装配/刷新/TOCTOU/fail-closed 与 article S4.2/S4.4/S4.5 分工；retention/RPO/RTO 数值仍 TBD，不宣称 design 复审、实现、真实恢复、部署或 E2E 完成。仍须由 Lead 复核。2026-10-27 当前 Arch-Designer 依据用户明确确认“本项目为测试项目，这版不做真卷韧性验证且后续无需验证”，追加 ADR-0024 并据此变更 `article-management-S4.3` 的验证范围；保留 ADR-0020 产品合同与既有未验证边界，不宣称真实卷 `flock`/`fsync`/崩溃结论、production readiness 或灾备就绪。仍须由 Lead 复核。

> **2026-10-28 本轮增量**：Arch-Designer Epicurus（`01a18106-6fce-7e60-8345-dd51ba9db6ff`）依据本会话用户采用建议授权及 Lead 对最终 Windows owner SID + 单 approver Ed25519 签名方案的明确选用，追加 ADR-0026 并按 TE r02/Lead 的 one enqueue 纠正形成 adr-0026-r02，仅同步 architecture-ir01-r29 指定段落。协议唯一 authority 为 identity.md 的 ADR-0026；首次登记仅记录architecture实际采用；本次已核FD完成内化及固定r46的design/stories，引用次数由这两条实际来源派生为2，不预记TE未来引用。此为架构专业产出，不是独立 Review、成果 checkpoint 或生产运行批准；旧作者/评审/决定来源不变。

> **F002 有界返工**：Epicurus 依据正式 R01 `S105-DES-R01-F002` 及本会话 Lead 明确采用七项输入的授权，将 ADR-0026 修订为 `adr-0026-r03`，并仅同步 architecture `r30` 的维护配置引用/修订元数据。来源为原正式评审与本轮实际作者返工，不是 Review PASS。引用次数仍由 architecture 与已登记 FD 两条实际来源派生为2；未预记 FD/TE 后续修订。

> **2026-10-28 r04 当前修订**：Epicurus依据cleanup-contract-adoption-r01/lead-adoption.json的真实Lead技术采用，将ADR-0026更新为r04并同步architecture r31；公开改变维护预算语义与数值端点支持面，保留旧r03/r02为当时历史。引用计数仍由原architecture及已登记FD两条记录派生为2，同一来源修订不加计，FD r49/TE r23未预记。本次是作者文档修订，不是CODE finding关闭、Review PASS、人的accepted/checkpoint或运行批准。

> **2026-10-31 ADR-0029 本轮增量**：当前 Arch-Designer 依据用户 2026-10-31 明确授权“S1.5 卡点走路径 B”（经 Lead 交接）及 ADR-0028 项目级授权，追加 ADR-0029 并同步 article design §3.4.9/§5.3/§6.1/§8.1。写作时点 2026-10-31。后续 r02 已登记实现与证据：Go 单测通过，真实 PostgreSQL 集成 PASS（PG 18.6，仅单机合成角色/单 schema 权限矩阵）。r02 独立评审结论为窄范围 CHANGES REQUIRED（`S15-ACL-R02-F001/F002/F003`，其中 F001 以实测推翻“同名关系必然 fail closed”的原措辞），r03 据此修订 §5/§8、新增 §10 并降 grant 为 out-of-band 资产，F001–F010 关闭、F011 与新增项在 r03 代码收口后复核。r03 ACL 子集独立复审为 PASS WITH MINOR（`S15-ACL-R03-F001`–`F003`），其中 `S15-ACL-R03-F003` 指出 §5 把 `provision.sql` 默认权限写成“仅 `INSERT`/`SELECT`”与实测不符；r04 据此更正 §5 默认权限句、在 §7 披露建表到 `REVOKE` 之间应用角色短暂持有两表 `UPDATE`/`DELETE` 的窗口，并更正 r03 引用记录中的错误前提，未改变决定约束。仍未取得人的签收、未真实部署/CI、未做备份恢复演练；该集成不证明生产部署、凭据分发或角色轮换。

> **2026-11-05 ADR-0033 增量**：项目负责人确认并授权「经决策不验证：像 ADR-0024 那样落一条 ADR，按变更后合同关闭 `article-management-S2.4`」。据此在 `content.md` 追加 ADR-0033，只改变 S2.4 的验证范围：真实 PG 容量/性能补测不纳入本版，未验证边界如实登记，不把 R09 八项 PASS、R10 源码/离线检查或组件测试拼成整组通过，不宣称 SLO、目标容量、P95 或 production readiness。不改变 S2.2/S2.3 已关闭结论、AM-Q2 参数、产品合同或生产代码，不解除 S1.5/S8.2 阻塞。仍须由 Lead 复核。

> **2026-11-05 ADR-0034 增量**：项目负责人确认「不需要这项门禁」（测试项目、经决策不验证）。据此在 `content.md` 追加 ADR-0034，只关闭 `C18-COV-F002` 的验证要求：经实测 admission 的 `Verify` 不读取 `recovery_runs.outcome`（temporary 返回 `internal_ready`、live 返回 `public_ready`，改 outcome 无影响），三状态门禁若需强制应落在 coordinator `Resume`/start，属新设计切片。未验证边界如实登记，不改变 S8.2 的 F003/F004 与 S1.5 阻塞，不改变任何生产代码。仍须由 Lead 复核。

> **2026-11-05 ADR-0035 增量**：项目负责人确认「经决策不验证」。据此在 `content.md` 追加 ADR-0035：Case18 的真实部署 E2E 与双系统验证不纳入本版，按范围决策关闭该要求并登记未验证边界；不以本机模拟或既有单机证据冒充部署结论。不改变 `C18-COV-F003`/`C18-COV-F004` 的复审链、S8.2 其余条件与 S1.5 阻塞，不改变任何生产代码或部署材料。仍须由 Lead 复核。
> **2026-11-05 ADR-0036 增量**：项目负责人回复「继续」，采用 Lead 建议方案 A。据此在 `framework.md` 追加 ADR-0036：`identity-auth-S3.4`、`identity-auth-S3.6`、`article-management-S1.4` 三张部署后验证卡按「经决策不验证」关闭并登记未验证边界；6 张生产恢复实现卡（`identity-auth-S4.5`/`S4.6`/`S4.23`/`S4.24a`/`S4.24b`/`S4.22`）维持 `[ ]` 阻塞，不受本决定影响。不以模拟或既有单机证据冒充部署/演练结论。
> **2026-11-05 ADR-0037 增量**：项目负责人回复「继续」，采用 Lead 建议的「以 ADR 登记不纳入本版」方向。据此在 `framework.md` 追加 ADR-0037：生产恢复实现链六张卡（`identity-auth-S4.5`/`S4.6`/`S4.23`/`S4.24a`/`S4.24b`/`S4.22`）按范围决策关闭并登记未实现边界；不实施代码、不改变写集，生产恢复入口继续 fail-closed，ADR-0030/0031/0032 的门禁与解除条件不变，未来需另立授权与 ADR 重开。
> **2026-11-05 ADR-0038 增量**：项目负责人回复「继续」，采用 Lead 建议第一项。据此在 `framework.md` 追加 ADR-0038：仅 host 为本机/保留域名且口令为占位或合成值的 DSN 视为非凭据，可随代码与证据入库；真实凭据、私钥、token 仍禁止。首次适用为 IR-01 工作树提交前的敏感信息扫描（31,619 个文件，28 个命中，均为合成值）。
