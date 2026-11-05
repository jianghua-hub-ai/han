# B13J-D09 r18 作者证据报告（ADR-0030 r18 修订）

- 角色：Arch-Designer（作者角色本体，非评审）
- 日期：2026-11-03
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 本轮未执行任何 Git 写操作（add／commit／checkout／switch／stash／restore／reset／push），只使用只读的 `git status`。
- 未读取、未回显、未落盘任何凭据（`HAN_TEST_POSTGRES_DSN`、口令、API key）。

## 1. 写集与实际改动

| 文件 | 改动 | 说明 |
|---|---|---|
| `.add/project/knowledge/decisions/framework.md` | 3 处 | 见 §3 |
| `.spec/wip/recovery-runtime/evidence-b13j-d09-r18/report.md` | 新建 | 本报告 |

未改动：`decisions/README.md`、`tasks.md`、两份 `design.md`、任何 `*.go`、`migrations/`、`cmd/`、`internal/`（哈希见 §5）。

## 2. framework.md 改前／改后基线（实测）

| 指标 | 改前（r17） | 改后（r18） |
|---|---:|---:|
| 字节数 | 352,220 | 370,555 |
| 按 CRLF split 的段数 | 1,487（末段为空） | 1,570（末段为空） |
| CRLF 数 | 1,486 | 1,569 |
| 独立 LF 数 | 0 | 0 |
| BOM | 无 | 无 |
| SHA256 | `BC42F14B487A2F99249B57F4D2413F82650BB404C08BF75FA87AB083BEC8B5B7` | `4EC6C2AF086BEC4ABB59E5DE180CC0323FA13EBD40D532098B5B271FDE1A148C` |

读写一律 `[IO.File]::ReadAllText` / `WriteAllText` + `New-Object System.Text.UTF8Encoding($false)`；未使用 `Set-Content`。全文保持纯 CRLF、UTF-8 无 BOM。

## 3. 三处改动落点（改后实测行号）

| # | 位置 | 改后行号 | 内容 |
|---|---|---|---|
| 1 | 修订串 | `framework.md:918` | `adr-0030-r17` → `adr-0030-r18`（该行现为修订串行，实测见 §6） |
| 2 | 新增小节 | `framework.md:1247`–`:1328`（共 82 行） | `#### r18 补充（D09 r17 两路评审返工：S3.6 移出模拟范围、``waived-for-test-scope`` 口径补全、B4 事实更正与 B7 裁定落定；不新增生产结论）`，位于 r17 补充小节之后、`### 约束`（`:1329`）之前 |
| 3 | 新增追溯行 | `framework.md:1369` | r18 修订追溯 bullet，紧随 r17 追溯行（`:1368`）之后 |

## 4. 未改字节证明（段级哈希，改前＝改后）

行号映射规则：`:1246` 及以前不漂移；r18 小节插入 82 行，故 `:1247`–`:1286` 整体后移 82；r18 追溯行新增 1 行，故其后的 ADR-0031／ADR-0032 整体后移 83。

| 受保护段 | 改前行号 | 改后行号 | 段级 SHA256（两侧一致） |
|---|---|---|---|
| ADR-0025 | 716–756 | 716–756 | `150FD60E82FBE9E205090BCD15100C79AFB86B79743216A87BFC5378A7B765C2` |
| ADR-0028 | 816–858 | 816–858 | `644E37785B40F89E38888CF9E1407FC83CA440BF0E705EECA6EB5554D3AFEFC6` |
| r13–r17 补充段 | 1050–1245 | 1050–1245 | `3A26BE38C23ED4B219B721EB8E83FF13AD4689B35C96AA2B568476EC2A636BCD` |
| r17 补充段（细分） | 1150–1245 | 1150–1245 | `FCBF016CAC4D2AE02DE5F6C92C5FE6CBA0B9519C9D557E86B8A944EFAD6E37F1` |
| r17 追溯行 | 1286 | 1368 | `1E871F736DDA944717929BFC99F8C47DE198864EEE295E8F6DCDEDF707104E08` |
| ADR-0031 段 | 1291–1408 | 1374–1491 | `41A4030434759351AB9B66D7A057D1EBD5BCC08999B6C5CE3D16599FDA1D0BEB` |
| ADR-0032 段 | 1410–1486 | 1493–1569 | `507FF85EF6D46837A5DFDFD2B5416640CDF4FD0556BE013E7B54BED8EFB0A2BF` |

段级哈希算法：按 CRLF split 取区间行，`-join "`r`n"` 后以 UTF-8 字节计算 SHA256。

## 5. 受保护文件哈希（改前＝改后）

| 文件 | 字节 | SHA256 |
|---|---:|---|
| `.add/project/knowledge/decisions/README.md` | 22,724 | `D854E1EBFBE8CE29981421A747AA4E770AC8944233F98F4417708694769DD3C2` |
| `.add/project/knowledge/tasks/tasks.md` | 912,450 | `E034155076F77F8D1E8288D9BB4014554A5019882051A03A305A0ACA6AD71EFB` |
| `.add/project/knowledge/design-workspace/identity-auth/design.md` | 1,110,477 | `491A476E60636ADAA3F1610A07C78054ED4C9735F5AD42D60B76200A12C8414F` |
| `.add/project/knowledge/design-workspace/article-management/design.md` | 382,562 | `63FD0691E1D66326B8EED7CFDFD8017417C60BB628CB961359087242D5D58B64` |

## 6. 验证命令与原始输出

`Select-String`（只读）：

```
S1 L918: **目的**: 架构设计 | **仓库**: han | **修订**: adr-0030-r18
S2 L1247: #### r18 补充（D09 r17 两路评审返工：S3.6 移出模拟范围、`waived-for-test-scope` 口径补全、B4 事实更正与 B7 裁定落定；不新增生产结论）
S2 L1329: ### 约束
S2 L1368: - 2026-11-03 r17 修订（触发：用户本轮下达的 D09 文档型决策卡；…）
S2 L1369: - 2026-11-03 r18 修订（触发：D09 r17 两路非作者独立评审——架构侧 …）
```

位置正确性：r18 小节起始行 `1247` > r17 补充小节末行 `1245`，且 `1247 < 1329`（`### 约束`），满足「在 r17 补充小节之后、`### 约束` 之前」。

`git status --short`（只读，节选）：

```
 M .add/project/knowledge/decisions/framework.md
?? .spec/wip/recovery-runtime/
```

说明：工作树在本轮开始前已脏（`git status --short` 共 302 项，含大量既有 modified 与 untracked）；`.spec/wip/recovery-runtime/` 整个目录本来就未被跟踪，因此无法用 git 区分本轮新增的 `evidence-b13j-d09-r18/` 与既有证据目录。就本轮实际写入而言，只写入了 `framework.md` 与本目录下的 `report.md`；其余 301 项与本轮无关，未触碰。不声称整个工作树 clean。

## 7. 逐条 finding 处置落点

| finding | 处置落点（`framework.md` r18 小节） | 处置要点 |
|---|---|---|
| 架构 F001（HIGH 阻断，Lead 裁定 (A)） | 表行 `:1257`；专段 `:1272`–`:1274` | 3.3 收窄为 `identity-auth-S3.4`＋`article-management-S1.4` 两卡；`:1227` 标题与 `:1229`「三条」按两卡读，`:1234` S3.6 行不再适用；写明 ADR-0025 `:720`／`:726`／`:746`／`:749` 只豁免 S3.3／S3.5、ADR-0028 约束 8 `:854` 要求扩大模拟范围须另行取得用户决定、S3.6 维持 blocked／未派发、如需扩大须用户另行明确决定并由新修订登记 |
| 架构 F002（HIGH 阻断） | 表行 `:1259`；专段 `:1278` | `waived-for-test-scope` 改为「范围豁免（test scope only）：不改变 `tasks.md:649` 的 requires 文本、不改变任何生产 requires、不使 S4.5/S4.6 变为已实施、不构成安全豁免」；并写明该标签不得被读作「requires 已满足」 |
| 安全侧第 1 项（SEC F001，MEDIUM 阻断） | 同 `:1259`、`:1278`（同一处 `:1216` 文本） | 四点齐备：仅测试范围生效／不改 `tasks.md:649`／不得读作 requires 已满足／不构成安全豁免 |
| 安全侧第 2 项（SEC F002，MEDIUM 阻断） | 表行 `:1260`；专段 `:1280` | 显式登记本 ADR 在 key retirement 上无决策权限（`:922` 继承关系；ADR-0030 约束 8 该条 r17 基线 `:1261`、r18 实测 `:1343`）；S4.6 豁免只以 ADR-0028 约束 2 为唯一依据、只为解除 S4.4 模拟前置，不改变 ADR-0019/ADR-0030 任何结论、不构成任何放宽 |
| 架构 F003（MEDIUM） | 表行 `:1258`；专段 `:1282` | 删除「模拟见证／备份卷替代真实独立故障域」；改为测试范围内**不取得**该证据、登记未验证／unknown，明确不满足 ADR-0030 `:953(c)` 判据 |
| 架构 F004（MEDIUM） | 表行 `:1263`；专段 `:1286` | 补「不实施 ≠ 豁免验证」，六卡仍在 ADR-0028 约束 1（`:844`）覆盖内，属该范围内的处置选择，二者实质不冲突 |
| 架构 F005（MEDIUM 事实更正） | 表行 `:1265`；专段 `:1290` | 更正 `anchor-env-r01/README.md` 存在：`.spec/wip/recovery-runtime/evidence-b13j-d05/anchor-env-r01/README.md`，5,941 B，SHA256 `A081AA9B300655B9FD94C55CE74DC0104928659D8E8EC507786CDBA212AD74B1`；说明存在但不构成本 ADR 登记值 |
| 架构 F006（LOW） | 表行 `:1261`；专段 `:1294` | 依据清单补 ADR-0028 约束 5（`:851`）与约束 8（`:854`） |
| 架构 F007（LOW）＋用户 2026-11-03 补充要求 | 表行 `:1262`；专段 `:1296`–`:1303` | 逐字复制 `preflight-record.md` §9：「建议值：单主机 WSL2 + PG 18.6，仅回环，模拟等级」与「建议值：2 vCPU / 4 GB / 20 GB，100 并发会话，p95 ≤ 500 ms，错误率 < 0.1%」；已用 `-ceq` 与 §9 原文（L117／L118）逐字比对通过 |
| 架构 F008（LOW） | 表行 `:1264`；专段 `:1307` | 冲突作为已知事实保留、未被消除，本轮只选择不实施，B2 仍开放 |
| 架构 F009（LOW） | 表行 `:1266`、`:1267`、`:1268`；专段 `:1311` | 三处锚点更正，映射见 §8 |
| 安全侧第 3 项（SEC F003，MEDIUM 阻断） | 表行 `:1269`；专段 `:1315` | r16 两路接受（架构 ACCEPT、安全 PASS WITH CONDITIONS）不自动延续至 r17／r18，各自须重新取得两路独立接受 |
| B7（Lead 裁定） | 表行 `:1270`；专段 `:1317`–`:1319` | seam 保留，语义限定「合成/测试注入」；生产 `main.go:634` 不传四端口；不改变 S4.23 BLOCKED |
| 安全侧 SEC F004（LOW） | `:1321`–`:1323` | B4「waived」改为测试范围不取得／未验证／unknown、生产仍 unknown；§9 的 (i)–(v) 测试口径登记不满足约束 9(b)、不改 `tasks.md:664`、不构成安全豁免 |
| 安全侧 SEC F005（LOW） | `:1323` | 明示不动 ADR-0030 约束 1–10 与 r05–r16 补充段；模拟替代不适用于 TLS verify-full、双 fd（3／4）、凭据保管分离、ACL 最小权限与见证单调性判据 |
| 安全侧 SEC F006（LOW） | 同架构 F006（`:1261`、`:1294`） | 约束 5 补引 |
| 安全侧 SEC F007（LOW） | `:1323`；本报告 §2／§4／§5 | 提供改前字节数、段数、CRLF 数与 SHA256，以及各受保护段改前／改后段级哈希，作为可核基线 |

## 8. F009 实测映射

| 项目 | 原标注 | 实测（r18 后） | 结论 |
|---|---|---|---|
| B1 表格行 | `framework.md:1162` | `1162` | 未漂移，锚点成立 |
| B2 表格行 | `framework.md:1163` | `1163` | 未漂移，锚点成立 |
| 作者证据 `evidence-b13j-d09/report.md:49` 的「引用记录」 | `framework.md:1168` | `:1168` 实为 B7 表格行（`recoveryRootInputs` 四端口 seam 行） | 原标注错误，已更正 |
| ADR-0030「引用记录」小节 | r17 基线 `:1265`–`:1270` | r18 实测 `:1347`–`:1352` | 已按两侧坐标书写 |
| r16 追溯行 | r17 基线 `:1285` | r18 实测 `:1367` | 同上 |
| ADR-0030 约束 8「不适用场景」 | r17 基线 `:1261` | r18 实测 `:1343` | 同上 |

补充：`evidence-b13j-d09/report.md:49` 原文为「引用记录 / r16 追溯行 | 1168 / 1285」，其中 `1168` 错误、`1285` 为 r17 基线正确值（该文件本轮未改，仍保留原文，属输入证据）。

## 9. 门禁边界（r18 重申，与正文一致）

- 生产 `enter`/`install`/`resume`/`unlock` 全 closed。
- ADR-0030 约束 7／9／10、ADR-0031 解除条件、ADR-0032 解除顺序一律不解除。
- B1／B3／B4／B5 仍为「产能外 unknown」。
- `S4.23`／`S4.24a`／`S4.24b`／`S4.22` 不得因 r18 启动。
- 未宣称生产就绪、真实故障域成立、ACL 已验证或真实环境 PASS。r18 是作者文档处置，不构成评审 PASS、人的接纳或生产批准。

## 10. 未触发停止条件

本轮未出现：需修改任务卡或设计才能自洽；Lead 裁定与 ADR-0028／ADR-0025 正文冲突；需新增约束正文；换行或编码无法保持 CRLF。写集未扩大。

## 11. 观察（未处置，不属于本轮写集）

- 工作树在本轮开始前已脏（302 项）。仓库根下的 `$null`（34 B，2026-11-02）、`$prof`（233,349 B，2026-10-31）、`' ){`（0 B，2026-10-31）等异常文件名，创建时间均早于本轮，非本轮产物；按「清理需用户确认」边界保留未动。
- `.spec/wip/recovery-runtime/` 整体未跟踪，故 git 无法逐目录区分既有证据与本轮新增证据。
- `decisions/README.md:38` 仍为 r17 摘要且含 S3.6，索引收口由 Lead 另行登记；本轮未改该文件。