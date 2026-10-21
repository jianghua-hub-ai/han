# Canonical 内容真相源

本目录是 ADD 的 Agent-neutral、语言/框架中立内容 SSOT。运行时差异留给 HostAdapter；项目差异留给项目采用配置和 project-local。本 README 只解释内容源码怎样维护，不建立第二份架构或客户运行方法。

| 资产 | 作用 |
|---|---|
| entry | 最小 L1 根入口 + 按需团队与流程导航 |
| roles | 角色属性、职责边界和逻辑引用 |
| core rules | 跨语言/客户端稳定纪律 |
| core skills | 可复用专业方法、两条客户 Workflow 的编排方法及资源 |
| profiles | Go、React、资源平台等可选完整实现 |

资产身份、依赖和启用范围以 [manifest.yaml](manifest.yaml) 为准，不从说明中的数量推导现役集合。

## 编辑原则

- 通用演进先改 canonical，再按目标映射生成所选三端候选；源码存在不证明各端已采用或验证。
- canonical 使用 `entry:` / `role:` / `rule:` / `skill:` / `profile:` / `workflow:` 逻辑引用，不写客户端工具、原生目录或权限语法。
- Role 只保留属性、边界、方法/规则引用、接口和停止条件；步骤与模板归 Skill。
- 技术细则进入 profile；项目未启用的 profile 不加载、不评审、不作默认。
- 项目知识继续归 `.spec`；design-workspace 中永久 design 不按目录名当临时。ADD 自身设计、调研和投影记录不进入项目模板。
- 当前路由以 `entry:project-guide` 为准：新会话首次请求或用户明确新建/切换时使用 `skill:route-work`；属于研发 Workflow 的工作先取得选定编排 Skill，再按 `description` 选择专业 Skill。当前活动复用有效正文，不因问句形式禁用方法；团队总览只导航，不另立执行步骤。
- 可分发 canonical/运行内容不保存研发过程材料；隔离的 `.dev` 研发资料按其批准的保留约定维护，不下发、不自动加载，不据本原则删除仍需审计或接续的依据。
- 同步按领域聚合追加，保护 project-local。项目完整单元变体保留为明确 replacement，候选合并、采用、升级与恢复统一由 `skill:project-adoption-workflow` 承担，项目内经验由 `skill:project-experience` 保存，禁止整文件/整目录覆盖或逐端各写一份方法。

## 与相邻源码的边界

| 目录 | 唯一责任 | 发行关系 |
|---|---|---|
| `canonical/` | Entry、Role、Rule、Skill、Profile 的中性正文和 manifest | 编译为 ProductRelease 的 `source/**`；不是 Host 原生投影 |
| `../adapters/` | 三个 Host 的非现役模板、映射、能力和 smoke 源码 | 编译为包内 `targets/template-map.json` 与不可发现的模板字节；原目录不整包复制 |
| `../scaffold/project/` | 初始化使用的中性空白模板 | 只有产品白名单逐项列出的模板进入 `templates/**` |
| `../projections/` | compiler 输出的非发行 golden 与回归基线 | 不进入 ProductRelease，不是客户 HostProjection，也不由本文清理 |

通用制品交付中性源码、模板和工具，不含预生成 payload 或母仓客户绑定值；项目从单份配置、官方源和已确认 replacement 生成。原生方法的未提交、未登记直改仍先比较保留；重要定制简记与再次生成、后续升级的持续保留按上述唯一方法处理，不另写维护步骤。母仓 Git 是通用版本历史，不是客户 B/C 的依据，不以提交或说明代替真实现场比较。

## 维护说明的交付位置

本 README 随 Canonical 编译结果进入包内 `source/README.md`，并可由采用结果保存为中性源说明。它面向维护者、非默认加载，不是 Host 会话或普通任务的前置。正常产品介绍只按明确白名单交付，不把母仓整目录、客户实例或 `.dev` 带入 source。

维护时从 [项目入口](entry/project-guide.md)、[框架元规则](rules/agent-framework.md) 与受影响单元的直接引用了解当前设计及改动影响；不先串读整个流程库。采用结果和原生权限须在实际目标另行核验，不能用本 README 充当验证记录。
