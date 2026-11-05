# B13J D04 独立评审报告（review-r01）

| 项 | 值 |
|---|---|
| item | `pending-B13J-D04`（gap-map §2.6，未登记） |
| objectType | design + adr + index（复合对象） + 安全评审 |
| 角色 | `add-reviewer`（非作者独立评审；延续 D02/D03 同一评审身份） |
| 作者 | Arch-Designer（Singer） |
| 工作区 / 分支 / HEAD | `D:\AID\VSCode\Codex\Han` / `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| 评审时间 | 2026-11-01（Asia/Shanghai） |
| 被评基线 | identity r60 `4D2FC4AC…94AB` / 854160 B / 4944 行；article r28 `639D2326…7F15` / 377416 B / 3267 行；framework `C1E5F803…B25C` / 168354 B / 1117 行；README `2EE124AE…6B97` / 21472 B / 110 行 |
| 方法 | `add-code-review`（design/adr 分支）、`decisions-guide`、`design-doc-writing-style`；只读，不修改被评对象 |
| 总判定 | **CHANGES REQUIRED** |

## 1. 评审范围与证据

- 被评对象：`design-workspace/identity-auth/design.md`（r60）、`design-workspace/article-management/design.md`（r28）、`decisions/framework.md`（ADR-0032）、`decisions/README.md`，以及与 `design-d04/snapshots/` 的逐行/逐字节比对。
- 依据：`gap-map.md` §2.6（`:141-161`）与 U5/U6/U7（`:201-203`）；ADR-0019（`framework.md:418` 起）、ADR-0030（`:916-967`）、ADR-0031（`:970-1048`）全文；真实代码 `cmd/han-server/recovery_runtime.go`、`recovery_sources.go`、`recovery_exclusive.go`、`recovery_config.go`、`cmd/han-server/main.go`，以及 `internal/article`、`internal/identity` 的引用行。
- 作者证据已按哈希核验：`design-d04/report.md` `C86A18C6D55248D072902076DB92FEDE7D70DB50BEFF5F402D2E7C38554B028E` / 27049 B；`design-d04/index.json` `455ED4C1F105…F2BD` / 20701 B；四份 snapshot 与改前基线逐字节一致（见 §6.1）。
- 只读边界：未改任何被评文件、未改作者证据、未做 Git 写、未删除/移动文件；本次唯一写入为 `design-d04/review-r01/report.md` 与本目录 `index.json`。

## 2. 关闭条件 6 项（gap-map:160）

| # | 关闭条件 | 结论 | 证据 |
|---|---|---|---|
| 1 | 11 端口的唯一生产来源可逐项列出 | **不成立（部分）** | 11 项集合与顺序无关地一一对应（identity `:3605-3617`；代码 `recovery_runtime.go:456-491`），其中 10 行来源与代码一致；`exclusive` 行（`:3616`）的“生产唯一来源”与真实生产接线不一致，见 `B13J-D04-RV-01` |
| 2 | 部署锁明确支持的 OS/平台与不支持时的拒绝 | 成立 | ADR-0032 `:1064`、`:1088`、`:1093`；identity `:3908`、`:3922`；`recovery_config.go:155`（路径必填）、`:170`（`journalPath == lockPath` 拒绝） |
| 3 | article 侧拥有 canonical orphaning | 成立 | article `:2432-2439`；identity `:1055`；代码 `internal/article/command_service.go:341-346`、`article_port.go:29-41`、`:171`、`internal/identity/article_port.go:16-39` |
| 4 | secret 只读候选且由 Journal 验密 | 成立 | identity `:3916`、`:3918`；`internal/recovery/journal.go:105-109`（“never mutates Journal state”）、`journal.go:568`、`internal/recovery/runtime.go:1869`；closed reader `recovery_runtime.go:631-640` |
| 5 | 设计 Review PASS | **不成立** | 本报告设计评审结论为 **CHANGES REQUIRED**（`B13J-D04-RV-01`）；安全评审为 PASS WITH MINOR，但不构成安全签收 |
| 6 | production 仍 closed | 成立 | `recovery_runtime.go:551`（空 registry）、`:586-610`（closed journal anchor / snapshots / restore / consumer / installation / articleAccount）、`:631-640`（closed secret reader）；`recovery_exclusive.go:13-26`（无 OS 后端、无 acquire 即 `ErrAdmissionUnsupported`） |

结论：6 项中第 2/3/4/6 项成立，第 1 项仅部分成立，第 5 项未成立；因此 `pending-B13J-D04` 的关闭条件整体**未满足**。

## 3. 12 条必审结论

### 3.1 11 端口清单是否恰为 gap-map 的 11 项

**成立（集合层面）**。gap-map:160 列出 `journal`、`articleSnapshots`、`consumer`、`installation`、`restore`、`commitments`、`sources`、`anchor`、`articleAccount`、`exclusive`、`secretCandidate`；identity `:3605-3617` 表体恰为这 11 项（顺序不同：表把 `restore` 放在 `installation` 之后、`articleAccount` 放在 `anchor` 之后）。代码 `recovery_runtime.go:456-491` 的 `recoveryEnterInstallMissingPorts` 也恰列这 11 个 seam，与 `recoveryRootInputs`（`:432-451`）字段一一对应。无多、无少。

### 3.2 每行“生产唯一来源”与真实代码的一致性

**不成立（`exclusive` 行），其余 10 行成立**：

- `journal`：`:3607` 写“root 以 `recoveryOpenJournal` 打开”“head anchor 未交付时构造为 `recoveryClosedJournalAnchor`”，与 `recovery_runtime.go:586`、`recovery_sources.go:618-639` 一致。
- `articleSnapshots`：`:3608` 写“未绑定读为 `recoveryClosedVerifiedJournalReader`”，与 `recovery_runtime.go:594`、`recovery_sources.go:540-555` 一致。
- `consumer` / `installation`：`:3609-3610` 写 root 未注入时为 closed，与 `recovery_runtime.go:602`、`:606` 一致；`inner/recovery` 的 `Coordinator.Consume`（`coordinator.go:133`）与 `installationExecutor.Install`（`installation.go:41`）存在，但 root 无生产注入点，故“组件已建成、root 未注入即 closed”属实。
- `restore`：`:3611` 写首期不提供、`recoveryClosedRestoreDriver` 占位、enter/resume 的 DB 替换在 Begin 前拒绝，与 `recovery_runtime.go:598`、`recovery_sources.go:557-574`、ADR-0031 `:1041` 一致。
- `commitments` / `sources` / `anchor`：`:3612-3614` 写“已登记 anchor adapter bundle + 空 registry 解析为 `recoveryClosedPort`”，与 `recovery_sources.go:425-430`、`:447-453`、`:470-492`、`:494-532` 一致；归属 ADR-0030，且 ADR-0030 `:959` 明确把部署独占、`ArticleAccountPort`、recovery secret candidate 排除在本 ADR 之外，`commitments/sources/anchor` 留在 ADR-0030 边界内，口径自洽。
- `articleAccount`：`:3615` 写 article 实现经 `identity.ArticleAccountPortFunc` 桥接、未装配为 `recoveryClosedArticleAccount`，与 `recovery_runtime.go:444`、`:610`、`recovery_sources.go:641-662` 一致。
- `secretCandidate`：`:3617` 写环境变量本身、closed reader `recovery_runtime.go:631-640`、读取不等于验证，与代码一致。
- `exclusive`：`:3616` 的“生产唯一来源 = POSIX 文件锁（`HAN_RECOVERY_RUNTIME_LOCK_PATH`）/ `restoreDeploymentExclusive`”与真实生产接线不一致，详见 §4 `B13J-D04-RV-01`。

### 3.3 归属唯一性与内部一致性（`exclusive` 行 vs `commitments/sources` 行）

**存在口径冲突（MEDIUM）**。`commitments/sources` 行以 `recoveryClosedPort` 表示“空 registry 的 closed 解析”；同一 registry 的 closed 端口（`recovery_sources.go:494-532`）同时实现了 `recovery.DeploymentExclusive`（`:523-531`），而 `:3616` 却把 `exclusive` 的落点写成 `restoreDeploymentExclusive`。代码中：

1. `restoreDeploymentExclusive` 仅出现在 `recovery_exclusive.go:15-42` 与 `recovery_exclusive_test.go`，生产代码从未构造它（`rg` 结果见 §6.3）；
2. 生产 root 唯一构造点 `main.go:634-636` 未设置 `exclusive`，`recovery_runtime.go:551` 用 `newRecoveryAnchorRegistry(nil)`，`:555` resolve 得到 `recoveryClosedPort`，`:647` 把它作为 `Exclusive` 交给 `recovery.NewRuntime`；
3. `recoveryAnchorAdapter` 含 `Exclusive` 字段（`recovery_sources.go:425-430`），注册校验要求非 nil（`:458-461`），resolve 时原样返回（`:485-491`），且 `recovery_runtime.go:565-567` 允许 `inputs.exclusive` 覆盖。

因此“生产唯一来源”不唯一，且 `journal` 行的 anchor closed 描述（`recoveryClosedJournalAnchor`）本身准确，问题集中在 `exclusive` 行与 ADR-0032 `:1060`/`:1113` 对它的继承描述。

### 3.4 ADR-0032 是否与 ADR-0019/0030/0031/0027 冲突或形成竞争决定

**无实质冲突，但继承描述带入了 `:3616` 的不准确**。ADR-0032 `:1060` 明确“只固定 `:3908`/`:3922`/`:3616` 的部署独占后端、平台矩阵、路径/ACL/持有语义与 exclusive 行归属/状态；不改 ADR-0019 的独立故障域、不随 PostgreSQL 回退、POSIX only / Windows 生产 unsupported、UTC 微秒、fail-closed；不改 ADR-0030 的首期不提供与 closed；不改 ADR-0031 的 restore 工具/批准主体/执行身份/宿主隔离/原生回执；不改 ADR-0027 的编排与正向准入”。这与 ADR-0019（POSIX only / Windows 生产 unsupported）、ADR-0030 `:922`/`:959`、ADR-0031 `:1041`、ADR-0027 `:790`/`:792`（恢复前持同部署独占、构造只建 closed 内存对象）一致，未形成竞争决定。解除条件 7 项（`:1099-1105`）包含 OS 后端、ACL 证据、负向测试、三入口证据、“架构与安全独立评审 + 正式 Task 登记”、“人的明确生产批准”、“Windows 另立 ADR”，可核验、无循环依赖（ADR 先定、Task 后登记），且 `:1104` 明确 Agent/评审/测试不替代人的批准。方案表 A–E（`:1072-1078`）覆盖独占文件锁、`O_EXCL`、DB advisory lock、进程内 mutex、无锁，B/C/D/E 的劣势陈述（崩溃陈旧、依赖被替换的 PostgreSQL、只覆盖单进程、运维约定）与 `:1070` 的痛点分析一致，未发现故意贬低或漏掉明显候选。`：1095` 明确不预占 Task、不声称验证。

但 `:1060` 同时声称固定 `:3616` 的“归属与状态”，而该行的状态描述与代码不符，故 ADR 继承了一处事实性不准确（见 `B13J-D04-RV-01`）。这不要求改 ADR-0019/0030/0031 正文。

### 3.5 secret candidate（`:3916`/`:3918`）

**成立**。唯一来源为 `HAN_RECOVERY_INITIAL_ADMIN_SECRET` 环境变量本身，无文件/配置/DB/argv/KMS 替代来源、无代码默认值；仅 `enter` 与“Store 独立证明本 run 信任根仍未准备的 `resume`”瞬时读取，其余形态零读取；明确“读取成功不等于验证”，由 `Journal.VerifyBeginSecret` 持 retained Begin key 纯读验密（`internal/recovery/journal.go:105-109`；调用点 `internal/recovery/runtime.go:1869`），Runtime 再核来源/许可/ctx/独占；不持久化、不进日志/Trace、失败清零；`：3918` 明确不单独开实现卡（U7）、随唯一 production root 装配承接、不扩 ADR-0026 与 FileRecoveryJournal 平台矩阵。与既有 `:2016`（仅 RecoveryGate 持锁且存在既有 initial admin 时读取）不矛盾；未发现“读取成功即信任根已建立”的残留。

### 3.6 article 侧边界（article `:2432-2439` + identity `:1055`）

**成立**。语义 owner 固定在 article `CommandService.OrphanArticlesCanonical`（`internal/article/command_service.go:341-346`）与 `CanonicalOrphanRepository`/`OrphanArticlesCanonicalAuditor`（`article_port.go:29-41`、`:171`）；唯一装配点为 production root 以 `identity.ArticleAccountPortFunc`（`internal/identity/article_port.go:16-39`）桥接并注入 `recovery_runtime.go:444`；同一调用方事务；identity 不得复制/改写 article SQL、不得第二仓储/第二事务/第二生产实现；未装配前 `recoveryClosedArticleAccount` 使账号删除 closed（`recovery_runtime.go:610`、`recovery_sources.go:641-662`）。与既有 identity `:2345`（PlantUML participant）、`:2416`（方法行）不冲突，属于精确化而非改写。注记（非 finding）：article 的 `OrphanArticlesResult` 三计数与 identity 的 `CanonicalDeleteArticleResult` 字段名不同，桥接需做字段映射，但 `ArticleAccountPortFunc` 足以承载，不构成契约冲突。

### 3.7 追溯准确性

**引用行号逐项为真**：ADR-0032 `:1113` 的 `:3908`（配置行）、`:3922`（平台矩阵段）、`:3601`（11 端口段首）、`:3616`（exclusive 行）、`:83`（决策追溯）均与实际落点一致；article `:2434` 的 `internal/article/command_service.go:342`、`internal/article/article_port.go:29-41`/`:171`、`internal/identity/article_port.go:16-39`、`recovery_runtime.go:444`、`:610`、`recovery_sources.go:641-662` 逐项为真。唯一追溯瑕疵：article `:69` 把 canonical orphaning 的依据写成“ADR-0032 及 11 端口清单”，而 ADR-0032 `:1109` 明确不适用于 `ArticleAccountPort`（见 `B13J-D04-RV-02`）。

### 3.8 诚实性

**成立**。四文件未发现“已实现/已接线/已验证/测试通过/已演练/已隔离实测”类越界断言：identity `:83`、article `:69`、`:2438` 均明确“Agent 产出、非人的签字、draft、人的精确接纳 unknown、不声明实现/接线/演练”；ADR-0032 `:1058` 明确 standing 授权不构成人的接收/独立评审/安全评审/生产批准，`:1095` 明确不声称 OS 后端、ACL、双进程互斥、崩溃释放或三入口共锁已实现/已通过，不声称 Case18/S8.2/E2E 通过；`:3922` 明确“不声称已接线或验证”。production 仍 closed（§2 第 6 项）。未发现未登记 Task 编号：ADR-0032 全文无 `S4.22`/`S4.23`/`S4.24`；identity `:4270`/`:4275`/`:4276` 的 `S4.22-24` 与 article 的 `S8.2` 为既有内容，不在本轮差异行内。

### 3.9 越界与格式

**成立**。仅 4 个被评文件 + `design-d04/` 证据被写入：`tasks.md` 仍为 `F1D65B0D7016CDBF9C2F4E85B012B5B0B9F714B6DF0C7CDC9326C05676900E34`、`plans/ir-01-content-community-batches.md` 仍为 `3FA19CDD4902F56B8C2F1DF59C570D96FAAE02D8AEE519B4B4165DCCA3637380`；`decisions/identity.md` 与 `content.md` 的 mtime 分别为 2026-10-28、2026-10-27，不在 D04 窗口（23:36 之后）内。格式：见 §6.1；`git diff --check` exit 0，仅有既有 `core.autocrlf` 提示性 warning（非空白错误）。

### 3.10 改动最小性

**成立**。与 `snapshots/` 的 `git diff --no-index --unified=0` hunk 集合恰为作者声明：

- identity：`-5 +5`、`-38 +38`、`-83 +83`、`-1055 +1055`、`-3600,0 +3601,18`、`-3890 +3908`、`-3898 +3916`、`-3900 +3918`、`-3903,0 +3922,2`；
- article：`-5 +5`、`-7 +7`、`-37 +37`、`-68,0 +69,2`、`-2429,0 +2432,8`；
- framework：`-1049,0 +1050,68`（仅末尾追加）；
- README：`-39,0 +40`、`-45 +46`、`-71,0 +73`。

`framework.md` 前 155139 字节 SHA256 = `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12`，与改前快照整文件哈希一致，证明 ADR-0019/0030/0031 正文逐字节未变。`design.md:7` 未变（仍 `last_updated: 2026-11-01`）；`:3874-3902` 零改动（hunk 只落在 `:3908/:3916/:3918` 与新增 `:3922-3923`）。未发现表格重排或顺手改动。

### 3.11 边界未越

**成立**。见 §3.9；未改 `tasks.md`、`plans/`、`stories.md`、`test-plan.md`、`decisions/identity.md`、`decisions/content.md`、任何测试文件；未登记 `S4.22`/`S4.23`/`S4.24` 或其它未授权 Task 编号。

### 3.12 对 D04 后续/D03 的可承接性

**基本成立，存在一处需本轮补的接口级边界**。ADR-0032 `:1109` 明确不适用于 `ArticleAccountPort` 与 recovery secret candidate（由同一 D04 的 article 设计与 secret 段承接），ADR-0031 `:1041` 明确不适用于部署独占、`ArticleAccountPort`、recovery secret candidate，D03/D04 边界清晰、无重叠。剩余接口级缺口即 `B13J-D04-RV-01`：在交给 `S4.22` 之前，必须明确 `exclusive` 的唯一注入路径，并排除 `recoveryAnchorAdapter.Exclusive` / `inputs.exclusive` 作为第二来源，否则未来 CAS provider 注册后可能把部署锁换成 provider 提供的对象，违反 ADR-0032 的 root/platform 归属。作者自述的遗留项（`flock`/`fcntl` 语义、锁目录 ACL、`articleSnapshots` 第二写路径、Store 独立证明细节）均属 ADR-0032 解除条件 2/3 或实现阶段，可合法后置。

## 4. Findings

### B13J-D04-RV-01（MEDIUM，OPEN，blocking）

- 位置：`identity-auth/design.md:3616`；`decisions/framework.md:1060`、`:1113`。
- 事实：`:3616` 的“生产唯一来源 = POSIX 独占非阻塞 OS 文件锁（`HAN_RECOVERY_RUNTIME_LOCK_PATH`）”与“当前无 OS 后端，`restoreDeploymentExclusive` 无 acquire 即 `ErrAdmissionUnsupported`”把未来实现类型写成当前生产落点。真实代码：
  - `restoreDeploymentExclusive` 只定义于 `cmd/han-server/recovery_exclusive.go:15-42`，生产代码无任何构造点（`rg -n "restoreDeploymentExclusive" --glob '!*.md'` 仅命中定义与 `recovery_exclusive_test.go`）；
  - 生产 root 唯一构造点 `cmd/han-server/main.go:634-636` 未设置 `exclusive`，`recovery_runtime.go:551` 用 `newRecoveryAnchorRegistry(nil)`，`:555` resolve、`:647` 注入的是 `recoveryClosedPort`（`recovery_sources.go:470-474`、`:523-531`）；
  - 第二来源存在：`recoveryAnchorAdapter.Exclusive`（`recovery_sources.go:425-430`，注册校验要求非 nil `:458-461`，resolve 原样返回 `:485-491`），以及 `inputs.exclusive` 覆盖（`recovery_runtime.go:565-567`）。
- 影响：关闭条件 1 的“唯一生产来源”在合同层不成立；`S4.22` 无法从本表判断唯一注入路径；未来注册 CAS adapter 时，部署锁可能来自 provider bundle，违反 ADR-0032 的 root/platform 归属与“不改 ADR-0030 的 provider 边界”。生产当前仍 closed，无 live 暴露，故非 HIGH。
- 建议（最小修法，不改 ADR-0019/0030/0031 正文）：在 `:3616` 明确“当前生产=空 registry 的 `recoveryClosedPort`；未来唯一授权注入路径=root 以 `restoreDeploymentExclusive` 注入 `recoveryRootInputs.exclusive`；`recoveryAnchorAdapter.Exclusive` 不得作为生产部署锁来源（或记录其为唯一路径并说明为何与 ADR-0032 的 root/platform 归属一致）”，并同步修正 ADR-0032 `:1060`/`:1113` 的继承表述。若选择“删除 adapter bundle 的 Exclusive 字段/改由 root 独立注入”，属实现任务，须在 `S4.22` 登记时体现。
- 处置：**本轮必须修**（仅文档行；不需要重开 ADR-0032 的决策结论）。

### B13J-D04-RV-02（LOW，OPEN，non-blocking）

- 位置：`article-management/design.md:69`。
- 事实：r28 追溯写“依据本轮 D04 生产 root 装配合同（ADR-0032 及 11 端口清单）”，但 ADR-0032 `:1109` 明确“不适用于 `ArticleAccountPort` 的 canonical orphaning 归属（由 article 设计承接）”，11 端口清单在 identity `:3601-3617` 而非 ADR-0032 正文。
- 影响：追溯归因不精确，可能让后续读者误以为 ADR-0032 管辖 article 侧归属；不影响技术结论。
- 建议：改为“依据同轮 D04 的 11 端口清单（identity `:3601-3617`，`articleAccount` 行 `:3615`）与 identity `:1055`，以及 gap-map U6”；不引用 ADR-0032，或注明其仅为同轮文档集。
- 处置：可并入后续 D04/D03 文档轮次，不阻断本轮。

### 状态汇总

| ID | 级别 | 状态 | blocking |
|---|---|---|---|
| `B13J-D04-RV-01` | MEDIUM | OPEN | 是（阻断 D04 设计 PASS） |
| `B13J-D04-RV-02` | LOW | OPEN | 否 |

本轮无 CLOSED finding（review-r01 为首轮，无前序 finding 可关闭）。

## 5. 评审结论

### 设计评审结论：CHANGES REQUIRED

11 端口的集合、ADR-0032 的决策与解除条件、secret candidate、article 侧归属、追溯、诚实性、最小性、边界与格式均通过；唯一必须修的是 `B13J-D04-RV-01`（`exclusive` 行“唯一生产来源”与真实代码不一致，且遗漏 `recoveryAnchorAdapter.Exclusive`/`inputs.exclusive` 两条路径），它直接使关闭条件 1 与第 5 项（设计 Review PASS）不能成立。修完该行并复核后，可再次提交定向复审。

### 安全评审结论：PASS WITH MINOR（不构成安全签收）

就部署独占机制本身：POSIX 独占非阻塞 OS 文件锁、`O_CLOEXEC` 常驻 fd、进程死亡内核释放、路径 owner-only/no-follow/单一文件、三入口共锁、设施/ACL/路径/获取失败一律退出码 2、无 fallback/降级、禁止 caller bool/no-op/fake、Windows 生产 unsupported，均与 ADR-0019 的 POSIX only/Windows 生产 unsupported 及 fail-closed 口径一致；`recovery_exclusive.go` 与 `recovery_config.go:170` 的现状描述属实；未发现新的自动 fallback、环境探测、caller 自报或把未验证写成已验证。MINOR 即 `B13J-D04-RV-01` 的供应商路径未固定：实现前必须把 `exclusive` 的唯一来源钉死，否则未来 provider 注册可能改变锁的来源。安全评审未执行任何 OS/ACL/双进程/崩溃/三入口实测，ADR-0032 解除条件 2/3/4 均未执行。

### 总判定：CHANGES REQUIRED

一句话理由：D04 的 11 端口、ADR-0032 决策与解除条件、secret/article 边界、格式与最小性均成立，但 `identity-auth/design.md:3616`（连同 ADR-0032 `:1060`/`:1113`）把 `exclusive` 的“生产唯一来源”写成未构造的 `restoreDeploymentExclusive`，遗漏了真实的 `recoveryClosedPort` 与 adapter bundle 注入路径，使关闭条件 1 和第 5 项不能成立；须本轮修正该行，`B13J-D04-RV-02` 可并入后续。

判定不构成人的接收、设计 Gate 或安全签收；不关闭 `pending-B13J-D04`、不解除 `blocked-upstream`、不解锁 `S4.22`/`S4.23`/`S4.24`、不声称 Case18/S8.2 通过。

## 6. 实测附录

### 6.1 格式与基线（自测）

| 文件 | SHA256 | 字节 | 行数 | crlf / bareLF | BOM | 末尾 | 控制字符 |
|---|---|---|---|---|---|---|---|
| identity r60 | `4D2FC4AC638094BEB1564DF041A973ABA29FE3E43B5D585117BCC586777994AB` | 854160 | 4944 | 169 / 4775 | 无 | 单 LF (`0a`) | 1（既有 TAB @468004，行 3303） |
| article r28 | `639D23262DE0D5D292B10FC29D371A9A908C33BD320E901E6BAE4BE9C5647F15` | 377416 | 3267 | 0 / 3267 | 无 | 单 LF (`0a`) | 0 |
| framework | `C1E5F80354A442D446E988AC60A45806ACC73F401563F19A17063B7B74A8B25C` | 168354 | 1117 | 1117 / 0 | 无 | 单 CRLF (`0d0a`) | 0 |
| README | `2EE124AE50FA0393D9D5B90F21075993F19B1BE6241C058D465CD56D9CD86B97` | 21472 | 110 | 2 / 108 | 无 | 单 LF (`0a`) | 0 |

- identity 唯一 TAB：字节偏移 468004、行 3303、位于既有 Go 代码块 `type PasswordResetGate interface {` 内；内容为单个 `0x09`，行号与改前快照相同，偏移变化仅因 `:83` 同行追加在其之前，非本轮新增。
- `framework.md` 前 155139 字节 SHA256 = `D74F09A0666AF86A77E55AAC8491DEF516CA81C181423E250F71A82CB9121C12`（= 改前快照整文件哈希）。
- snapshot 哈希：`identity-before-r60.md` `B07BAFC7…AF99`/847437；`article-before-r28.md` `4D46B307…128A`/374925；`framework-before-r32.md` `D74F09A0…1C12`/155139；`readme-before-r32.md` `A67B00D5…0F706`/20764，均等于上一轮基线。

### 6.2 差异行（`git diff --no-index --unified=0`，1-based 新行号）

| 文件 | hunk |
|---|---|
| identity | `-5 +5`、`-38 +38`、`-83 +83`、`-1055 +1055`、`-3600,0 +3601,18`、`-3890 +3908`、`-3898 +3916`、`-3900 +3918`、`-3903,0 +3922,2` |
| article | `-5 +5`、`-7 +7`、`-37 +37`、`-68,0 +69,2`、`-2429,0 +2432,8` |
| framework | `-1049,0 +1050,68` |
| readme | `-39,0 +40`、`-45 +46`、`-71,0 +73` |

与作者声明完全一致（identity `{5,38,83,1055, 3601-3618, 3908,3916,3918, 3922-3923}`；article `{5,7,37, 69-70, 2432-2439}`；framework `+1050-1117`；README `{40,46,73}`），无额外差异行。

### 6.3 实跑命令与结果

| # | 命令 | exit | 结果 |
|---|---|---|---|
| 1 | `rg -n "HAN_RECOVERY_RUNTIME_LOCK_PATH|ArticleAccountPort|HAN_RECOVERY_INITIAL_ADMIN_SECRET|生产 root 装配" .../identity-auth/design.md` | 0 | 15 行：`83, 226, 1055, 2016, 2345, 2397, 2416, 3166, 3601, 3615, 3616, 3617, 3908, 3916, 3918` |
| 2 | `rg -n "ArticleAccountPort|OrphanArticlesCanonical|canonical" .../article-management/design.md` | 0 | 91 行 |
| 3 | `rg -n "部署锁|文件锁|Windows|POSIX|ADR-" decisions/framework.md` | 0 | 102 行（含 ADR-0032 `:1054-1117`） |
| 4 | `git diff --check -- <4 文件>` | 0 | 无空白错误；仅 `core.autocrlf` 的 LF→CRLF 提示性 warning |
| 5 | `rg -n "restoreDeploymentExclusive" --glob '!*.md' .` | 0 | 仅 `recovery_exclusive.go:13/15/20` 与 `recovery_exclusive_test.go`；生产无构造点 |
| 6 | `rg -n "exclusive:" --glob '!*_test.go' cmd internal` | 1 | 无命中（生产 `recoveryRootInputs` 不设置 `exclusive`） |
| 7 | `rg -n "newRecoveryAnchorRegistry" --glob '!*_test.go' cmd` | 0 | 仅 `recovery_runtime.go:551`（`nil`）与定义 `recovery_sources.go:455` |

### 6.4 越界与诚实性扫描

- `git status --porcelain` 在本对象范围内显示 4 个被评文件为 `M`、`design-d04/` 为 `??`；`tasks.md`、`plans/…batches.md` 的哈希与基线一致（见 §3.9），`decisions/identity.md`/`content.md` 的 mtime 不在 D04 窗口内。
- 四文件关键词扫描未发现“已实现/已接线/已验证/测试通过/已演练”类越界断言（除既有历史行）；ADR-0032 无 `S4.22`/`S4.23`/`S4.24` 登记；未出现 Case18/S8.2 通过声明。

## 7. 未覆盖范围与复审条件

**未覆盖**：

- 未运行任何 Go 测试、集成测试或真实服务；未验证 OS 锁后端、ACL、双进程互斥、崩溃释放、三入口共锁（ADR-0032 解除条件 1–4 均未执行）。
- 未做真实备份/restore 演练；未验证 article canonical orphaning 的实际接线、GRANT/角色权限或 secret 读取。
- 未审查 4 个被评文件与 `design-d04/` 之外的工作区既有 `M`/`??` 变更（`content.md`、`identity.md`、`stories`、`test-plan`、`go.mod` 等）来源；这些不属于本项，未处置。
- 未对 `cmd/han-server` 之外的实现细节做全面代码审查，仅核对被评文本引用的代码行。
- 未做安全签收、人的接收、设计 Gate 或生产批准；不声称 Case18/S8.2 通过。

**复审条件**：`B13J-D04-RV-01` 修正 `identity-auth/design.md:3616` 与 ADR-0032 `:1060`/`:1113`（明确当前 `recoveryClosedPort`、唯一授权注入路径、排除 adapter bundle 的第二来源），并提供最小差异行与格式实测；`B13J-D04-RV-02` 可同时或后续修正。复审将只定向核对这两项及其直接影响的最小性/格式/追溯，不重跑全量 12 条，除非证据显示其它条目受影响。

## 8. 声明

本报告为 `add-reviewer`（Agent，非作者）的独立评审产出，不是人的签字、不是设计 Gate、不是安全评审签收、不是生产批准；不关闭 `pending-B13J-D04`、不解除 `blocked-upstream`、不解锁任何候选 Task、不声称 Case18/S8.2 或任何实现/测试/演练通过。判定仅覆盖本报告列明的文件版本与只读检查范围。
