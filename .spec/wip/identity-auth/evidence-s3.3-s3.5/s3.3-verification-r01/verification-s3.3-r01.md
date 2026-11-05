# identity-auth-S3.3 剩余覆盖核证 r01

**结论：可交非作者独立评审核对本报告及原始证据；不能据此宣称 S3.3 覆盖完整或关闭。** 非浏览器定向普通/race 已通过；I-AUTH-04 的实际登录轮换与后续 PG 会话探测、I-AUTH-07 的真实三桶并发/重启持久性仍有未豁免证据缺口 G01/G02。它们是覆盖不足，尚不是已复现的产品缺陷。真实浏览器环境准备、连接、执行依 ADR-0025 为 **WAIVED / NOT RUN**，不得写为 PASS。

## 对象、归属与固定输入

- 当前活动：`development-delivery` 正常续接，原 `identity-auth-S3.3` 的剩余核证；不是 E2E，不新增 task。原卡仍为 `[-]`，状态只引用 [tasks.md](../../../.add/project/knowledge/tasks/tasks.md) 的 S3.3 段。
- 执行角色：本会话按用户指派承担 `add-test-engineer`；实际 Session ID（运行时 `CODEX_THREAD_ID`）为 `01a18077-e5df-7240-baac-bdee83a21685`。未再次委派；该 ID 不冒充新 spawn、Reviewer ID 或人的签字。
- 工作区：`D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。结论绑定当前未提交字节，并非仅绑定 HEAD。
- 唯一仓内写集：本报告。开始时目标不存在；源码、两份原计划 browser 文件、tasks/plan/design/ADR 均只读。两个 browser 文件目前仍不存在，没有补造文件。
- 基线：`C:/Users/J00010385/AppData/Local/Temp/han-s33-s35-lead-k2hnhmtu/baseline.json`，SHA256 `9588472BF28DD774AECC209C9D7AC7EA73632897A25B81D7A604534AA3406D47`。首次核对、每个 Go child 前后及最终核对均为全部 183 项匹配；六项 document 基线也匹配。
- 正式依据：完整本角色、`add-test-design`、`add-delivery-workflow`、`task-execution-workflow`、`multi-agent-coordination`；适用 coding/review 规则及 go-service。写作指导已读取，本执行报告不属于其设计文档强制结构范围。
- 验收依据：SR-IDENTITY-02 / US-IDENTITY-02.1；原卡及已关闭 B0.1/S1.4/S3.2；[design.md](../../../.add/project/knowledge/design-workspace/identity-auth/design.md) r45 §3.3.1/§3.3.4/§3.4.4；[test-plan.md](../../../.add/project/knowledge/design-workspace/identity-auth/test-plan.md) r20 I-AUTH-01/03/04/07；[ADR-0025](../../../.add/project/knowledge/decisions/framework.md#adr-0025)（framework.md:716 起）及必要 ADR-0022 有界观测合同。test-plan 元数据仍指 r44，本次按现役安全条款逐项核对，不据此追认整个 r45/r20 已确认。

| 固定文档 | 本次 SHA256 |
|---|---|
| tasks.md | `22C4796844643688FEA82925966656CF51C6CD0FA2885190ADFBC712BAE7ACDA` |
| framework.md | `4FA647D562B869A4D5F09241244937F5E7066A6837965C766E39B6B7099315A2` |
| identity design.md | `D57BB2152A93F6F651165EE7DBE1913B28E9E8719B6991B1FDD36E8B7DBAB7FE` |
| identity test-plan.md | `0052894D3C4B58F644749F8337BB264FD2768DD26A31C7EA0A361F0F61D939BD` |

## 本次实际执行

证据根目录记为 **E**：`C:/Users/J00010385/AppData/Local/Temp/han-s33-verification-r01-01a18077`。下述 `E/` 后路径均为该可复制目录内的相对引用；完整报告镜像也置于 E，仓内相对链接则指项目原文件。`E/index.json` 持有每个保留文件的相对路径、字节数和 SHA256；183 项源码副本在 `E/source183/`，实际读取副本/区间在 `E/readset/`、`E/readset.jsonl`、`E/initial-readset.json`。

| child | UTC 开始 → 实际 wait 结束 | 实际结果 |
|---|---|---|
| toolchain，PID 14400 | 2026-10-28 03:10:19.525546 → 03:10:19.629776 | exit 0；Go 1.27.0 windows/amd64；不计场景通过 |
| inventory，PID 67416 | 2026-10-28 03:10:20.622562 → 03:10:23.200612 | exit 0；选定 26 top 恰好全部存在；0 测试体执行；不计场景通过 |
| normal，PID 58952 | 2026-10-28 03:10:47.209647 → 03:10:49.128244 | exit 0；26 top / 84 leaf / 97 不同节点，各 run/pass 一次，4 package PASS |
| race，PID 34220 | 2026-10-28 03:10:56.741322 → 03:11:07.404672 | exit 0；同一 26 top / 84 leaf / 97 节点，各 run/pass 一次，4 package PASS |

普通/race 均无 fail、skip、missing、extra、重复、缓存结果；没有失败后挑选成功重跑。26 top 是 6 platform + 9 session + 9 identity + 2 router；84 leaf 包含无子例的 top，不能把 26+84 当作 110 个独立用例。

实际 argv 的共同形式为 `go test -count=1 -json -timeout=120s [-race] -run <锚定的26个完整top名称并集> ./internal/platform ./internal/session ./internal/identity ./cmd/han-server`。完整无省略 argv、PID、cwd、UTC、180 秒外层 timeout、wait/exit 和 raw SHA 保存在 `E/{toolchain,inventory,normal,race}/{launch,result}.json`；筛选前的精确 top/预期 leaf/源码区间在 `E/selection.json`，运行后逐节点计数在 `E/{inventory,normal,race}/verification.json`。

子环境由白名单重建，无任何 `HAN_*`，未读取 Process/User DSN；`GOENV=off`、`GOWORK=off`、空 GOFLAGS，`GOTOOLCHAIN=local`、`GOPROXY=off`、`GOSUMDB=off`、`GOTELEMETRY=off`。使用既有本地 Go/module/build 缓存输入，`-count=1` 禁止复用测试结果；没有声称冷缓存构建。CGO 与 CC 指向已安装 GCC，未安装工具。数据全部为源码内合成值/内存替身；使用 httptest recorder，无监听、外部连接或真实浏览器。环境明细在各 child 的 `environment.json`。

| 原始流 | SHA256 |
|---|---|
| `E/normal/stdout.raw.jsonl` | `E295BFE5099D690617FBE45BBFD6E80039423171D1D47F991460BB2E635E3B34` |
| `E/race/stdout.raw.jsonl` | `F28103C70394F37473CC1EFEB2A51BE9F8ABA4FBB0C56A3E5E4921C254D41116` |
| 两次 `stderr.raw.log`，均空 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |

Lead 的现场核证单独保留于 `E/lead/lead-s3.3-raw-verification.json`（UTC 03:12:55.334899，SHA256 `BD7E6E4973EF212BBF50B86454C31B77C136F2285ED7675EC69EA5123D864BAE`）：独立解析上述原始 JSON、选择集、wait、183 项 guard 与环境；这是 Lead 核证，不是独立重跑或 Reviewer 结论。

## 逐风险覆盖、未验与豁免矩阵

各行均映射 SR-IDENTITY-02 / US-IDENTITY-02.1 和原 S3.3。下表“本次 PASS”只对注明的最低层断言有效；独立 Review 尚未结论。没有任何一行把 httptest、spy 或 SQL 字符串检查升格为真实 PG/浏览器证明。

| 场景 / 风险与合同 | 最低足够层与真实断言来源 | 实际覆盖结论 | 未验 / 豁免 |
|---|---|---|---|
| I-AUTH-01：可信登录成功及受保护 Cookie | `internal/identity/login_test.go:763` 的 Handler success（service stub）输出 200、新 Cookie/CSRF；`internal/platform/http_security_test.go:582` 检查 `__Host-han_session`、Secure/HttpOnly/Lax、Path=/、无 Domain/未批准持久期限，605 拒绝 CRLF Cookie | 本次普通/race PASS：服务端序列化与 HTTP 响应合同 | 浏览器接收/拒收、Secure/HttpOnly/SameSite 实际执行为 WAIVED / NOT RUN；未测试真实 HTTPS |
| I-AUTH-01：原始 token/CSRF 不入存储 | `internal/session/store_test.go:262` 真实 Create + recording executor，断言 SHA-256 参数、SQL/参数无明文；`login_test.go:1329` 固定熵 reader 的 32 字节 base64url 与显式时限 | 本次 PASS：摘要生成/传参、编码；不是数据库读回或熵质量评估 | 真实库中仅摘要的本次读回 NOT RUN，非浏览器豁免；旧 PG 报告不足以补此断言 |
| I-AUTH-03：缺失/恶意 Origin、cross-site Metadata 在认证前拒绝 | `http_security_test.go:17` 三负例下游零调用、403；521 同源正例；`cmd/han-server/main_test.go:6119` 四例核 Origin→Handler 顺序和 attempts/service/metrics 零调用 | 本次 PASS：HTTP 组件拒绝路径与 router 隔离；同源+same-origin 可达 | 部署代理/真实 Header 生成未验；本次未单列 same-site/none/畸形 Metadata 全组合，不宣称穷举；有 Origin 而浏览器未提供 Metadata 的合同仍按 design 保留 |
| I-AUTH-03：禁止 credentialed CORS | `http_security_test.go:545` 恶意/同源 OPTIONS；即使下游故意写 ACAO/ACAC，输出仍无这两个 Header | 本次 PASS：Guard 的拒绝及允许分支均不开放跨源凭据 | 浏览器预检/CORS 执行为 WAIVED / NOT RUN；没有全路由真实网络抓包 |
| I-AUTH-03：写请求 CSRF 缺失/伪造必须 fail closed | `http_security_test.go:615` 三例 resolver stub，断言 CSRFRequired/403 或 SessionRequired/401 且下游不调用；`store_test.go:396` 用 stored-digest row stub 验 missing/forged/matching；`main_test.go:6013` 六例精确 PUT→Origin→Session→CSRF→Handler | 本次 PASS：传输门禁、摘要比较、router 零业务调用；无真实存储写入 | HTTP resolver spy 的拒绝不是数据库“零行改变”快照；浏览器跨站发起方式豁免，真实 PG 状态断言未补 |
| I-AUTH-04：登录成功撤销旧会话、失败保持 | `login_test.go:219` 11种统一失败下 sessions=0、oldSessionActive；347 成功/审计失败的 commit/rollback；763 success/failure 的 Cookie 边界 | 本次 PASS：Service/Handler 与事务/Session 替身编排 | 未证明 actual Login→真实 SessionStore→PG commit→下一旧 Cookie 请求拒绝；G01，未豁免 |
| I-AUTH-04：Store 原子轮换、回滚及旧 token 拒绝 | `store_test.go:505/531/563/613` 替身断言旧 digest 撤销后插入、caller commit/rollback；332 的“rotated old token”等6例均注入同一个 ErrNoRows；354 仅查 SQL 谓词 | 本次 PASS：调用次序、事务归属、错误分类 | 名为 old token 的例子没有实际执行 SQL 或从先前 Rotate 生成状态；不能补 G01 |
| I-AUTH-07：单账号多来源、三种桶键、protector 重建 | `login_test.go:691` 同一个 memory store 两次 failure 后重建 protector，第三来源 SubjectBlocked、动态等待4分钟、5个去敏桶键；741 的 pgx.Tx spy 核 advisory lock→read | 本次 PASS：内存算法局部路径及 SQL 调用次序 | 不是 DB durability/进程重启，不是三桶并发原子计数；G02，未豁免 |
| I-AUTH-07：多账号单来源/账号+来源、账号失败统一 | `login.go:729/781` 静态核三维读写与 source/subject 分类；Handler source rate limit 使用 service stub 给429/Retry-After:2且不设Cookie。历史七项包含其余既有普通测试，但不替代风险断言 | 本次仅输出映射 PASS；未找到生产三桶真实并发/重启的充分证据；账号维度错误统一有组件支持 | source桶/组合桶独立触发、DB计数无丢失、事务失败全回滚、重启后继续退避仍无充分证据；G02 |
| I-AUTH-07：非可信代理 Header 不可伪造来源 | `login_test.go:1140` 15例核 hostile XFF 忽略、wire order、mapped地址一致、16/17 hops与畸形拒绝；1286 的8例核 trust-all/重复前缀等 | 本次 PASS：真实 provider 解析/去敏；不是代理网络实测 | 未连真实代理；不扩大为本卡之外的部署验证 |
| I-AUTH-01/03/04/07：真实浏览器环境、连接与执行 | ADR-0025 与原卡变更后范围 | **WAIVED / NOT RUN**；历史目录缺失 setup FAIL 不改写 | 插件/Native Host、入口证书、Browser/Go 协同均不再是前置；产品安全合同继续有效 |

## 历史证据及其适用边界

本轮未执行 PostgreSQL、integration、浏览器、SMTP、平台/部署或 Manifest 七项；未读取运行 DSN，未运行/复用任何自动 DROP 的 integration helper。读取测试源码和历史报告不等于执行。

- [S3.2 r06](review-code-s3.2-f005-r06.md)（2026-10-23；SHA256 `0739133B9674B9B03EE76F0E4D0D0D3F81F6BA3D3AA1A40F7A7DEA057DB6EE67`）及原卡记载 Lead PG18.6 三组通过，Reviewer 明确自己未复跑 PG。其 login/service/handler/测试等多项历史 SHA 已与现役不同，本报告不移植整个历史 PASS。当前 `login_integration_test.go:23/148/273` 测的是 audit commit/rollback unknown 与 round-trip，345 测 User 锁下凭据/状态改变；481 明确用 abuse stub，不能支撑真实三桶。其目标也不是成功登录轮换后旧 Cookie 的下一请求。
- [S1.6 作者自审报告](review-code-s1.6-r01.md)（2026-10-27；SHA256 `EFBDE5D35FD198888CB4CD40624C5947C1FE9E0486A4E112847BF9D62D40C485`）记录真实 PG18.6 会话解析前后、治理撤销及修复 ISSUE-identity-auth-0001；相关当前测试文件 SHA 与报告一致。`test/integration/identity/postgres_test.go:1035` 在**同时撤销 Session 并停用用户**后拒绝；不隔离证明 active 用户的 login rotation。该来源是作者自审与历史执行报告，不是独立 Review；本次未取得该次完整外置 raw/wait/UTC，故只作相关支持，不作为本次 PG PASS，也不虚填执行时间。
- S9.2 code-r02 的 durable binding 仅用于定位 183 字节输入与已有离线执行条件；**未把其 canonical-delete 测试计为 S3.3 场景覆盖**。
- Lead 另核同 source183 的旧七项结果及原日志一致，收据 `E/lead/lead-prior-manifest-applicability.json`（2026-10-28 03:07:28.475614 UTC，SHA256 `06DA785C9B4905971AAFF588DD37FB6C14072DF113E26988D3C4296737A485CD`）。原结果在仓内 [lead-manifest-r02/results.json](evidence-s9.2/lead-manifest-r02/results.json)，SHA256 `0DCF064C0982FADE0C5BF116303703C77E9533E87AFE785C93DBD9D723DF1B5D`；本次复制到 `E/prior-manifest/`，未重新执行。旧 UTC 结束时间分别是 2026-10-28 format `02:23:01.256808`、modules `02:23:02.961607`、vet `02:23:05.552806`、build `02:23:09.661431`、contract `02:23:12.623787`、race `02:23:14.674035`、unit `02:23:16.768047`，均 exit0。它们是项目检查支持，不自动补足 G01/G02。

## 未豁免缺口与交回

| 缺口 | 最小缺失证据 / 为什么阻断充分性 | 原卡内交回范围 |
|---|---|---|
| G01 · I-AUTH-04（兼 I-AUTH-01 存储读回） | active 用户真实登录成功，读回仅摘要/撤销旧会话/新会话已提交；以旧/新 token 发下一受保护请求，分别拒绝/允许；认证失败及审计失败回滚后旧会话仍允许。现有组件断言与“撤销且停用”PG报告无法等价替代这组行为 | 原 S3.3 的证据缺口，非新 task。已有入口/断言位置为 `internal/identity/login_test.go:347/763`、`internal/session/store_test.go:563/613`、`internal/identity/login_integration_test.go`；补源码或执行 PG 均越过本次权限，受影响部分 STOP，由 Lead 决定是否交 Backend，不能复用自动DROP harness |
| G02 · I-AUTH-07 | 三桶在对应来源/账号组合下真实原子计数及退避，不丢增量/不半提交；相同持久状态在进程/实例重建后仍有效；来源桶允许429而账号维度继续统一失败。内存 store 重建和 lock SQL spy 没有这些持久边界证据 | 同一原 S3.3 缺口；现有算法/测试位置 `internal/identity/login.go:729/781`、`login_test.go:691/741`，现有 PG lock test 使用 abuse stub。补普通算法断言或安全的PG fixture均需 Lead 另给 Backend 精确权限，本角色不写源码、不另登记任务 |

这两项由原验收直接产生，不是额外要求未来 E2E、真实代理或生产容量。阈值只用已有合成 fixture 值证明算法；生产阈值、容量/P95、目标环境 Argon2 profile 仍为既有独立边界/TBD，不在本报告中采用臆造值。race 通过不等于数据库并发或性能通过。

自动化边界是本次 HTTP/Go 组件核证；真实浏览器为已授权豁免；PG 只有上述历史来源。没有新增 Case/Run、E2E 建议任务、人的 accepted/checkpoint 或清理决定。S3.4、S3.5、S3.6 均不由本报告改判。

**交回状态：stopped。** 本报告及 E/index.json 可供 Lead 通知 Bernoulli 开始最终静态/原 raw 核对；其唯一报告 `review-verification-s3.3-r01.md` 尚待真实返回，不能预填 PASS。Lead 已解析本次 raw 的现场收据与该非作者评审是不同证据。是否补齐 G01/G02、原卡关闭与任何下一动作均交 Lead；本角色不接下一项、不自行实现或关闭原卡。
