# identity-auth S3.3 原 finding 独立复审 R02

- 日期：2026-10-28；同一 `development-delivery`，原非作者 Reviewer 对 F001/F002 复审，未再次委派。
- 工作区：`D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- objectType：code（两份新集成测试及必要直接消费者）；依据原 I-AUTH-01/03/04/07、安全合同及 ADR-0025。
- **Code Review Gate: PASS，限 source185-r01 下本次两文件、F001/F002 所需断言与实际独立 LOCALPG 证据。** 两轮完成，原两项验证缺口均 **CLOSED**；无新 CRITICAL/HIGH/MEDIUM/LOW finding。
- **不是产品 Bug 修复结论**：R01 的 F001/F002 原本是未豁免验证缺口；本次补测试与执行证据，原 183 项未改，产品安全合同未改。
- 浏览器继续 **WAIVED / NOT RUN**。本次不宣称 production factory、真实 HTTPS、部署、E2E、生产密码 profile/阈值/容量通过；不签人的 accepted/checkpoint，不关闭 Tasks，不接下一工作。
- **结束状态：stopped，交 Lead。** 唯一仓内新写入为本 R02；R01、S3.5 R01～R04 等既有报告不覆写。

## 1. 固定对象与归属

原报告：`.spec/wip/identity-auth/review-verification-s3.3-r01.md`，SHA256 `1854B46F231C2C260C07A7DA8D8FEAF366A5567FCBB54BB7C836ECC8BB0D1340`。原核证作者 Bacon 的报告/执行归属保留，不改成新作者结果。

本次代码作者为 **Erdos** `01a18086-c383-7b72-bcfe-75635bd6c600`；CODE 已 native stopped。作者 LOCALPG actual raw 已完成并由 Lead 正式交复审；截至本轮收到该交接信号时，作者 LOCALPG native final 封存仍待回，不冒称该状态已返回。本 Reviewer 的独立执行与作者执行分别取证。

冻结 snapshot **S**：`C:/Users/J00010385/AppData/Local/Temp/han-s33-s35-lead-k2hnhmtu/fixed185-r01/`。

| 固定对象 | SHA256 / 范围 |
|---|---|
| `source185-r01.flat.json` | `CC86009A6350314DA549D81F551B85ADEB1CA9AFA83F7298E860BAD80D8D75C7`；185 项、163 Go；原 183 项逐项保持原字节 |
| `test/integration/identity/auth_security_postgres_test.go` | 418 行；`0A96A652CFBE74B7E222E1774BAAD4C50606588C298593982D0F9820A602A4AA` |
| `test/integration/identity/auth_security_fixture_test.go` | 911 行；`0DF966B9F17AD4DAF3275F3BE6428A37A4E600619524C484279C9EE4E26ED088` |

新文件全文从 snapshot 读取；每个执行 child 前后同时核全部 185 个 live/snapshot 绝对路径和 SHA，另核原 183 与五份材料。无源漂移；不会用只匹配 HEAD 代替未提交文件版本。

本轮原读 Tasks SHA `044700F2619618739E5CB6B42327982DBFD92926E21F8615EB0693C2171895BF`、Plan SHA `1E04EEF5E010D56C8DAB7AEAF6257312DFF0D959AF0BF58B14D44E0C9B6E380C`。本轮不写这些元数据；S3.5 已关闭和 S3.3 的 R 对账不作为本 Reviewer 的关闭授权。安全契约仍按 design/test-plan、ADR-0025 的同版边界消费。

## 2. 两轮实际审查

### Round 1：新两文件全文、风险断言与静态证据

- 全文检查 418+911 行；两 parent top 固定为 `TestS33LoginSessionPostgres`、`TestS33LoginAbusePostgres`，共 13 leaf/15 nodes。`TestS33LoginAbuseChildProbe` 排除在主 selector 外，Subject/Source/SubjectSource 三个有效 helper 模式单独核。
- 逐项检查真实 LoginHandler、Coordinator、Service、SessionStore、PasswordReset Gate、Audit Store、HTTPAuthorizer 的接线；s33Tx/s33Audit 仅记录并委托真实 PG SQL，不伪造事务或审计结果。controlled hasher/clock/observer/HTTP terminal 的边界单列。
- 检查 F001 三分支：成功提交与仅摘要读回、旧/新 token 后续授权、缺失/伪造 CSRF、认证失败保持旧会话、schema-local 23514 审计失败后完整回滚并持久化失败审计。
- 检查 F002 十分支：三维独立分类、缺失/已有桶和共享 subject/source 的并发、第三桶失败的空/已有状态全回滚、真实新进程读同一持久状态。pair-only 通过 public store.Save 构造，明确是测试前提，不伪称完整 HTTP 自然达成。
- 检查 no init/TestMain、helper 在读取 DSN 前拒绝缺失/非法 mode、无 Skip、无自动 DROP/目录清理；审核 CODE 包全部 63 项 SHA、list-r02/compile/负控的 argv/PID/wait/exit/raw 与 source185。

### Round 2：真实消费者、隔离与实际执行交叉核对

- 复用本会话已读且与冻结版相同的 Login/Session/HTTP 安全合同；补读本次实际引用的 Gate SQL、Audit Append、migration.Load、testenv fail-closed、sessions/buckets schema 与 generation 约束等必要片段。未扩成整个 identity 或全库语义评审。
- 检查 literal loopback:5433 与全部 fallback、PG `180006`、实际 user/database、owner/search_path/current_schema 检查先于新 schema DDL；SQL 标识符引用、仅新随机 schema 的九 migration 演进、21 表清单与 migration CREATE TABLE 集合一致。固有 constraint/index DROP 是获准的新 schema 演进，不是额外 cleanup。
- 检查 21 表完整行在内存中含 raw `xmin/ctid` 比较；允许改变的表另核实际内容/计数/摘要/绑定，而非只把 hash 相等当业务证明。磁盘仅保留 redacted rows、原始/去敏行及 SQL hash；不导出 DSN 或原始运行密钥。
- 独立核作者完成的 normal/race 原始事件、13 leaf、3 helper、21 表/阶段关系、SQL/去敏 hash、blocker PID 和 fault 证据；再在独立新 runroot 与新 schema 执行同版完整 normal、race 和无 DSN tagged vet。独立运行未复用作者 schema。
- 规则方法复用同版本 Reviewer、add-code-review、delivery、checklist/abstraction、产品安全红线、task-execution/multi-agent、适用 go-service/coding；必要历史方法 add-research 的已读正文继续有效。自然语言检查、机械哈希与真实执行分别保留证据。

## 3. 原 findings 逐项复审

### IA-S3.3-R01-F001 — CLOSED

原缺口是 I-AUTH-04（兼 I-AUTH-01 摘要读回）缺少实际 Login→真实 SessionStore→提交→旧/新 token 下一受保护请求以及失败回滚证据；不是已复现的产品 Bug。

| 新测试 / 位置 | 已实际证明的内容 |
|---|---|
| `SuccessRotationDigestAndProtectedRequest`，postgres 文件 `:99` | active 用户真实登录，HTTP 新 token/CSRF 与真实 issuer 一致；PG 读回 32 字节 SHA-256、会话绑定/时限、两 Session 仅一 active，旧会话因 login_rotation 撤销；同版真实 authorizer/resolver 使旧 token 401/零 terminal、新 token 204/一次 terminal，缺失或伪造 CSRF 403/零 terminal |
| `PasswordFailureKeepsOldSession`，`:134` | 错密码 401/不发替换 Cookie、不新建会话；真实三桶与失败审计提交；旧会话仍 204，sessions 等非允许表 raw 行及 xmin/ctid 保持 |
| `AuditFailureRollsBackRotation`，`:151` | 真实轮换已在原事务 staged；schema/table/constraint 精确匹配的 23514 来自实际 Audit Append；真实 rollback 后 replacement 不存在、旧会话/全部非审计表恢复原状态；独立失败审计持久化且旧 token 仍允许 |

fixture `:366`～`:410`、`:598`～`:688` 证明真实装配与后续 HTTP probe；`:493`～`:519`、`:578` 的实际 raw 行/xmin/ctid 断言及新测试显式字段读回相互支持。normal/race 各三 leaf 均真实执行通过；独立证据核验还确认 rotation committed、rollback、后续请求各阶段 21 表关系。R01 中“预设 ErrNoRows”“oldSessionActive 替身”未再充当这组证明。

### IA-S3.3-R01-F002 — CLOSED

原缺口是 I-AUTH-07 的真实三维持久桶并发原子性、退避与重建后有效性不足；不是已复现的产品 Bug。

| 新测试组 / 位置 | 已实际证明的内容 |
|---|---|
| SubjectAcrossSources、SourceAcrossSubjects、SubjectSourceOnly，`:185/:199/:227` | 实际 PG store 的 subject/source/pair 分类与已有行计数；账号维度维持统一 401 envelope，source 返回 429、Retry-After=240；固定 clock 前进一分钟后从真实 blocked_until 得到四分钟剩余退避 |
| 四个 Concurrent leaf，`:252`～`:309` | 两真实事务及独立 observer PID；实际 advisory 未授锁、阻塞方 PID 与 pg_blocking_pids 对应；A 未提交时 observer 全表不变，A 提交/B 未提交与最终两提交后按真实键精确核 missing=2、existing=3、共享 subject/source 计数，无丢增量或半提交 |
| 两个 ThirdBucketFailure leaf，`:311`～`:330` | schema-local trigger 在 prior 两桶 staged 写入恰为两行后才以 23514 拒绝第三桶；空/已有状态都真实 rollback，21 表 raw 行/xmin/ctid 恢复，已有三桶计数仍 1 |
| FreshProcessRetainsBackoff，`:332`～`:418` | 关闭 parent 主连接池后执行三个有效新 OS 进程；helper 在同一**本轮 parent 新 schema**重新连接/构造真实 store，精确已有计数和四分钟退避继续有效，Check 前后及 parent 观察的 21 表均不变 |

fixture `:723`～`:805`、`:824`～`:904` 另核故障身份、精确 key/version/count/deadline、实际 blocker、helper 启动/Wait/退出与 raw。独立 normal/race 每轮四个 blocker receipt、三个有效 helper 全部通过；helper 与 parent PID 不同，同轮三个 helper 的 os.Executable 路径/保留文件 SHA 一致。

**pair-only 限定**：`:216`～`:225` 通过 public `PostgresLoginAbuseBucketStore.Save` 播种 pair-only 持久状态，并证明 subject/source 未封禁；它证明已有 pair 状态的读取/统一拒绝及跨进程持久性，不证明相同阈值下完整 HTTP 会自然产生 pair-only 封禁。

## 4. 作者来源与 Reviewer 独立执行

作者 CODE：`C:/Users/J00010385/AppData/Local/Temp/han-s33-code-4l42o33_/handoff/`。首次 list 的 int32/uint32 编译失败原始 stderr 保留，后修订仅新两文件；它不是产品 RED。最终 list/compile exit 0、actual wait；missing/invalid helper mode executable exit 1 是预期守门负控，不是业务测试 PASS。invalid-mode 的 runner 向 stdin 写入非法 Mode 后关闭管道，不能因 `stdin_closed=true` 误解为与 missing-mode 完全同输入。

作者 LOCALPG **A**：`C:/Users/J00010385/AppData/Local/Temp/han-s33-localpg-r01-jj8j11kt/`。已亲核 smoke/normal/race/vet 原 raw/result；smoke 只一 leaf，未计为完整专项；normal/race 各 13 leaf/15 nodes、3 helper、16 目录、107 快照/2,247 表镜像。作者 Go 临时测试 binary 已不在原路径，不能虚填其后验 binary SHA；其实际 argv、PID/Wait、source185 及 helper receipt 保留。下述独立运行另以 `-work` 保留并校验 binary，构成独立充分证据。

Reviewer 证据目录 **R**：`C:/Users/J00010385/AppData/Local/Temp/han-s33-review-r02-e65e0523e8/`。

Reviewer 独立 runroot **P**：`C:/Users/J00010~1/AppData/Local/Temp/han-s33-review-localpg-r02-qmqcumws/`，cwd 均为 S。P 的 26 个新 parent schema 与作者 smoke/normal/race 全部无交集，normal/race 各 13 个；6 helper 使用各自本轮 parent schema，不新造或复用作者 schema。

| Reviewer child | UTC 开始 → actual wait 完成 | PID / exit | 实际结果 |
|---|---|---|---|
| normal | 04:22:30.150971 → 04:22:43.503821 | 69064 / 0 | 2 top、13 leaf、15 nodes；每节点 run/pass 恰一次；3 helper；无 fail/skip/missing/extra |
| race | 04:25:59.960724 → 04:26:20.478021 | 67428 / 0 | 同一完整选择集，3 helper；无 fail/skip/missing/extra |
| tagged vet | 04:26:53.170338 → 04:26:54.419901 | 68076 / 0 | `go vet -tags=integration ./test/integration/identity`，不注入 DSN |

实际主 selector 为 `^(TestS33LoginSessionPostgres|TestS33LoginAbusePostgres)$`，排除 helper；normal/race 使用 `-count=1 -failfast -json -work`，race 另有 `-race`。无 smoke/负控重跑，未省略 PG。每轮 117 条 migration 应用记录、四组真实 advisory blocker 与三处 fault 验证均核证。完整 argv、cwd、PID、wait、UTC、exit、source185 guard 与去敏环境在 P 各 child 文件中。

| Reviewer raw / 同轮 helper binary | SHA256 |
|---|---|
| normal stdout | `21E02FDF8A1161F2F50EFD076DD0468B68893491AFC46B26F75568C487CF3E7F` |
| race stdout | `7963E6DDAD8D5F726040985F6704F4F9BE77A54EE19A7598C65EABD945B4A805` |
| normal 三 helper 共用的保留 binary | `061F53D64CA7F3D66F3423F7755779228D9753B416F8577D6DFCD287E831E71C` |
| race 三 helper 共用的保留 binary | `6297977BE3701F12B60C2CA4EE3E17DFD18F321581993C5D742E7C117AD2CFDB` |

**Reviewer 收据解析纠正**：首次 normal 的 Go 与全部测试实际 exit0/PASS；Reviewer runner 错把 `WORK=…` 必须出现在 stderr 当作 gate，故原 `normal/result.json` 的汇总 `passed=false` 与 `STOP.json` 保留。实际 Go 1.27 `-json` 将 WORK 写入 JSON `build-output`。只读同一 raw/完整 21 表和 helper 核验后，新增 `normal/work-reconciliation.json` 和 `evidence-checked.json` 更正该解释；原文件不覆写。normal 没有重跑，没有修改被审断言或吞测试失败；随后只执行尚未执行的 race/vet。最终结论引用实际事件/exit及纠正收据，而不是原错误汇总位。

## 5. 新 findings 与限制

新 CRITICAL/HIGH/MEDIUM/LOW finding：**无**。原 F001/F002 CLOSED；没有未决的本轮实质缺陷或需要扩大安全豁免的方向决定。

- **受控边界**：hasher 是合成验证器，clock 可控，observer 记录最终 observation；issuer/source 使用真实实现的合成输入。保护终端是 204 terminal，真实 HTTPAuthorizer/Session resolver 执行，但未走 production factory、TLS listener 或完整业务 endpoint/E2E。不将该测试解释为生产 Argon2 profile、密码熵评估、真实代理、HTTPS 或容量通过。
- **运行范围**：本次真实 PG 18.6、批准的 loopback:5433、新随机 schema。仅获准九 migration 固有 constraint/index DROP；无额外 DROP SCHEMA/TABLE、public/旧 schema 写入、文件清理或 schema/row 清理。所有新 schema、行、快照和失败历史保留。
- **凭据边界**：User DSN 只由 runner 在内存捕获并注入 integration/helper，vet 无 DSN；白名单环境、GOENV/GOWORK/offline 设置及 `GOFLAGS=-mod=readonly` 独立核。报告/索引不导出 DSN、原密钥或原始 token，证据中的 digest/fingerprint 只有 hash 表示。
- **证据含义**：实际运行触发代码中的 raw xmin/ctid 比较，PG 字段/键/计数/rollback 有显式断言；磁盘 hash/去敏镜像是补充复核，不代替执行。作者与 Reviewer 各自结果、Lead 核证不互相冒充。
- **浏览器与后续**：真实浏览器仍 ADR-0025 WAIVED / NOT RUN；S3.4、S3.6、部署、真实平台、生产阈值与整体需求验收均不因本 PASS 改判。

## 6. 当前项目检查、交回与可复制证据

Lead 本轮 `lead-s33-r01-results.json` 的 format/modules/vet/build/contract/race/unit 七项是 **source185 上的新执行**，不是旧七项复用；其实际退出码、原日志 SHA、185 before/after/无漂移另核。本 Reviewer 不冒称这七项由自己执行，也未重复执行已过范围。

R 中 `payload-index.json` 提供相对路径、来源、字节数、SHA；包含本报告、固定输入与 readset、CODE/作者/Reviewer raw、完整 21 表去敏镜像、SQL/raw hash、helper 启动/结束收据、blocker、原失败及解析纠正记录。`before.json`、`after.json`、`final-guard.json` 记录本轮 185 live/snapshot 与安全文档、冻结报告保护；`static-audit.json`、`evidence-author-*.json`、`evidence-reviewer-*.json` 记录独立核证。

可复制包**排除** Go build/work cache 和大 EXE；它们仍在 P/作者原 Temp 原地保留，binary 路径/SHA 另存索引，不清理。实际 readset 区分两新文件全文、必要旧消费者片段和同版复用，机械复制/哈希不冒充全库语义评审。

本次 PASS 只交回 Lead 核 revision、实际 diff/所有权、报告/raw、项目检查与适用接收；本 Reviewer 不改 Tasks/Plan、不代人的接受或关闭。若 source185 或相关测试断言变更，须新固定修订并沿本稳定 finding ID 重新评审，不把本结果借给新版本。

**stopped，交 Lead；不执行下一工作。**

## 7. 交回保护收据

取证时间（UTC）：`2026-10-28T04:31:35.074819+00:00`。185 live/snapshot 与本轮初始集合逐项相同，原 183、五材料及本轮记录的 Tasks/Plan、安全契约和既有报告均未改变。独立 normal/race 共 26 个 parent 新 schema、32 个快照目录全部保留；每轮 107 快照、2,247 表镜像，两个运行之间 schema 不重叠。仅本 R02 为 Reviewer 新仓内写入。
