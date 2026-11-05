# HTTP 两文件非作者独立评审 / S8.3 / R01

结论：**PASS（仅固定两文件的组件代码评审），0 findings**。两轮实际审查、上游与直接消费者核对、当前 compat2 下独立普通/race 执行及独立合成探针均完成。此结论不关闭 Task，不代替 Main 核收、全仓 Manifest 七项、批次验收或人的接纳。

评审身份：当前被交接的 add-reviewer，非作者；作者为 Pauli / 01a1854a-0e79-7f02-abfd-632b91c4fb04，已 STOP。本人未创建子 Agent，不虚构本会话 Agent ID。采用 add-code-review 的 code 分支，在 development-delivery 的 review 环节交回 Main。

## 固定对象与版本

Workspace `D:/AID/VSCode/Codex/Han`，branch `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，现场再次核对一致。目标为两份未提交新增文件；HEAD tree 无两路径条目，Main START 另证实施前工作树两路径 absent。当前两文件均与作者 frozen 副本逐字节相同。普通 git diff 不呈现 untracked 文件，故未以空 diff 代替审查。

| 对象 | SHA256 |
|---|---|
| internal/platform/recovery_guard.go | 0187E9A0B40FFC465240BFA8FF40FFD9DA7DB8A69B378F1931D94B6974C1826D |
| internal/platform/recovery_guard_test.go | EB4598546D62B932D053304BE4DEA45DE9C3CD5BD3BE6B4558C168A8764397F3 |
| internal/recovery/runtime.go，稳定接口/必要实现段 | 284886C23EA1038B685F1EA5BD1A6B1BC423077042939687C6EF062392594384 |
| internal/recovery/admission.go，compat2 仅核哈希 | E234CC71C5A21D42EFD6E8B345F2499800B607F2D8AF66043A0EB1A4D38FD3F0 |
| internal/recovery/admission_test.go，compat2 仅核哈希 | 02E9232EAC4A87F81582038304C1B06D6C6381DE2959CD45B95F7286B4CBFB0A |
| article design r23 | 2223BE4D005753FBF8D8AE0116F6926A7E8C044482BCBB0FADAB811677066350 |
| article test-plan r21 | E2E7ADFD0581BCCE9CD2782F7181C12BA67A8496A1A321EF88006EA94D9FCA27 |
| identity design r52 | 5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB |

`fixed-inputs.json` 保存现场分支、HEAD、固定摘要、frozen 相等及 runner 摘要。`readset.json` / `readset.jsonl` 保存实际读取路径、范围和 SHA；早于记录工具创建的读取明确标为补记，hash 为最终现场观察，不冒称开始时自动快照。部分大输出发生截断，必需正文/合同段已按小段补读；读取并不等于整文件全部语义审查。

## 依据与实际读集

已全文取得 reviewer Role、add-code-review、首次定位 Route、当前 delivery Workflow、core checklist/abstraction/coding/no-hardcode、go-service README/coding/no-hardcode/security、产品安全红线与适用 tracing。协作/执行只取本角色的固定边界、授权、文件所有权、非作者复审及交回条款；未预读其他 Role 或未来活动方法。

验收来自 Tasks 原 S8.3 卡（观察时 2443–2476；授权 peer metadata 插入导致行号可移动）、article design §8.6、article TP Case18 的组件/真实 root 区分、identity r52 §4.2 的 WorkLease/公共阶段/真实排空以及 3816 行 probe 合同。读取 Main G0 receipt 与 s83-source-local-start-r01.json，核其只有两文件 SOURCE/LOCAL、原 Writer/同步 lease/现有 Verify 建议的采用与禁止扩界。

作者包中的 receipt、delivery/readset/risk-results、source.diff、checks 原始 JSON 输出及 frozen 均独立核对：最终两文件与冻结一致，diff 确为两项 /dev/null 新增。已独立解析作者普通/race 的 56 叶和修正夹具后的可编译 Red（6 叶，5 FAIL/1 PASS），保留作者原错误夹具/中间版本的历史区分。作者这些 PASS 属于 compat2 之前；本报告的当前版本结论依靠本人重跑，不沿用旧 PASS。

## Round 1：固定新增内容的直接审查

完整检查 production 96 行与 test 796 行。逐条核范围、行为/取消/并发/资源、安全、错误与关联上下文、硬编码边界、测试及组件检查：

- `recovery_guard.go:17` 构造器拒 nil/typed nil Runtime，零值 guard 也拒业务；`:29–44` 对缺依赖、Acquire 错误、无有效 lease、nil/typed nil/canceled Context 和取消请求保持既有恢复 503，不派发 Session。
- `:33–46` 每次选中请求仅执行同 Runtime 的 `Acquire(ctx, WorkRequest)`；有效 lease 立即 defer 恰一次 Release，使用 lease.Context 同步调用后继，不缓存 ready/permit、不 Verify/Consume 普通请求。
- `:46` 直接透传原 ResponseWriter；未引入 wrapper、goroutine、超时提前返回或假 join。defer 覆盖 Session、业务、同步 WriteHeader/Write、写后尾部、错误与 panic 展开。
- `:51–64` live 固定 alive 且不观察 Runtime；ready 只在 Verify 成功、State 精确 public_ready、请求仍未取消时 ready200。失败 body 固定 unavailable503，状态/Reason/run/ref/账号/secret 不序列化。
- `:67–82` 原 WriteJSON/WriteError/注册错误码产生 JSON/Envelope/Trace，设置 nosniff，无新增 authority、root、router、安全链职责或依赖配置。忽略响应写错误沿现有 HTTP handler 返回模型；未伪装成业务成功或在提交响应后补第二份错误。
- 独立阅读全部测试断言、正控、合成接口、真实 Runtime genesis fixture、barrier/cleanup、取消/失锁/错误/panic 和 exact harness 的证据边界。测试未暗改真实 Session/SQL 或业务语义；四组存在实际叶而非只有父 top。

## Round 2：有界上游/消费者核对与独立探针

按真实依赖继续核对：

| 接缝 | 实际检查与结论 |
|---|---|
| Runtime → guard | runtime.go 82–191、394–577、757–870：WorkRequest/WorkLease/Verify/Context/Release、真实独占检查、取消代际、Stop/真实 drain。guard 消费既有接口，没有重造开放权威。Acquire 衍生 ctx 保留父值和截止；Release 归还工作并取消其 ctx。 |
| guard → HTTPAuthorizer → Session | http_security.go 283–341；Session Principal.Allows 与 ResolvePrincipal 258–306 的 ctx/事务契约：resolver 接收同 lease ctx，业务在 Principal 派生 ctx 下执行，Session 拒绝仍在 guard 同步调用范围内。真实 Store SQL 未执行。 |
| 503 → Error/Envelope/Trace/Writer | errors.go 的恢复503注册、PublicError/WriteError 146–243；http.go 的 TraceMiddleware 与 WriteJSON：根因不进入公开 envelope，已有 trace 沿上下文传播，Header/body trace 一致。未对该文件其他 JSON decoder 做完整评审。 |
| exact router → guard | 用两文件中明确标注的组件 harness 核 Trace→exact path/method/Allow→guard→Session 顺序；404/405 不 Acquire。未读取/审查 production root，也不宣称该 harness 证明真实 router 接线。 |

独立测试 overlay 只替换测试文件的编译视图，包含作者原测试全文并追加 `TestReviewerS83Boundary`；生产源码和仓内测试字节未修改。六个最终叶为：

1. SessionRejectionResponseOwnsLease：真实 HTTPAuthorizer＋独立拒绝 resolver，核 401、零业务、拒绝响应 Write 期间未 Release，返回后一次。
2. ErrorPlusLeaseHeldThroughDenialWrite：合成 Acquire 同时返回 error＋lease，核错误响应写完才归还、503 且不泄漏私有 cause。
3. WriterPanicUnwindsWithoutSecondResponse：Writer panic 原样传播，一次 Release，无第二次响应。
4. ProbeExactStateAndLivenessNoObservation：独立状态表含大小写/空白近似值，只有精确 public_ready 放行；live nil request 仍固定 alive，零 Verify/Acquire，Reason canary 不外泄。
5. ConcurrentRequestsOwnDistinctLeases：同一 guard 并发16请求，16次 Acquire、16个不同 lease，各 Release 一次，无缓存或串用。
6. RealRuntimeStopJoinsBlockedResponse：仅复用作者明确的 genesis 中性 port 初始化 fixture，独立控制真实 Runtime 与阻塞 Write。已取消的 Stop ctx 不使其提前返回；请求 ctx 撤销、新请求拒绝，Write 未返回时独占设施未关闭；放行 Write、真实同步返回后 Stop 才 join 并关闭一次。此 fixture 不证明真实锁/provider/worker。

首版探针执行 raw 位于 reviewer-normal/reviewer-race；之后仅加强本人并发 lease 逐对象计数和阻塞 Stop 的负向观察，最终版本重新执行两种模式。报告只将 reviewer-final-* 计入最终结论，未把旧探针 raw 绑定到新 overlay 版本。所有历史 raw 保留。

## 独立执行结果

所有 Go/gofmt 执行均使用从 Main 逐字节复制的安全 runner（原 helper 未改）：OS/path allowlist，GOENV off、GOTOOLCHAIN local、GOPROXY/GOSUMDB off、GOFLAGS 空、test DSN 空。没有读取真实 secret ENV/DSN/key，没有网络下载、真实 PG/DDL/integration、服务/worker/restore、Git 写、清理或子委派。

| 自有 checks label | 实际结果 |
|---|---|
| http-list | 原 top 真实命中、exit0 |
| http-normal / http-race | 各69节点、56叶 PASS；四组分别20/12/13/11，0 skip/fail，exit0，无 race 诊断 |
| reviewer-final-normal / reviewer-final-race | 各7节点、6叶 PASS，0 skip/fail，exit0，无 race 诊断 |
| platform-normal / platform-race | 各293节点、259叶 PASS，0 skip/fail，exit0，无 race 诊断；不使用 overlay |
| source-fmt | 仅两目标 gofmt -l，exit0且 stdout/stderr 空 |
| platform-vet | go vet ./internal/platform，exit0且 stdout/stderr 空 |

真实 argv、cwd、时间、env policy、退出码和 raw stdout/stderr 均在 `checks/<label>/`。`risk-results.json` 逐叶保存 outcomes、四组计数和原始日志 SHA；`validate.py` 对非零叶、四组精确计数、普通/race叶路径一致、0 required skip/fail/race、目标/依赖哈希和作者 frozen 相等作独立断言，已执行成功。没有重复执行 compat2 专项或原 Runtime 全部230叶。

## Findings、待决定和建议

CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0。未发现具备触发场景与实际后果的 HTTP2 缺陷，因此没有虚构文件位置、违反依据或修复建议。无新增组件方向决策、无必须采用的非阻断建议。

组件检查与独立 Review 的 PASS 仅覆盖本报告固定对象。Main 最终核收/Task 状态、全仓七项与已知根 fmt 旧23路径 FAIL2 的本卡处置仍在 Main；旧 S4.7 的有限例外不扩为本卡豁免。本人没有重新测根 fmt，未把历史 FAIL 说成本轮重现或已经通过。

## 最新协调事实与未验证限制

Main 在最终报告前明确更新：compat2 已获原 Ptolemy 独立 PASS/0新 finding 并由 Main 接收（canonical compat-review-r01、compat-main-reception-r01.json）；Zeno 原 S4.8 的11文件 SOURCE/LOCAL 已开放（s48-source-local-start-r01.json），Runtime/admission/API 与 HTTP2 继续冻结。此处是 Main 本轮转交的协调事实，不冒本人读取这些额外回执或复审 compat2 的结论。

本报告验证执行完成于该 peer 更新之前；未因 Journal lane 开放重新跑已完成检查。未读取或哈希将变动的 Journal/coordinator/filelock/snapshot 实现，也未依赖其迁移内容作评审结论。若随后 Go 编译碰到 peer 过渡，须保留 raw 并按实际修订协调，不能自动归为 HTTP finding。全仓七项与联合再验证仍由 Main 在相关 source 稳定后执行。

未验证：唯一 production factory/root 的真实装配与 same-instance 注入、真实 exact router、真实 SessionStore/Article SQL/PostgreSQL、Journal/retained-key/HMAC provider/独占介质、完整恢复/restore/三 worker、真实 HTTP listener/TCP、部署/Host/E2E/Case18 与 S8.2整卡、全项目 Manifest7、容量/性能。真实 HTTPAuthorizer＋合成 resolver 只证组件契约；真实 Runtime＋合成 genesis ports 只证许可生命周期，不证生产 provider。

取消只保证协作式 ctx 撤销；任意无视 ctx 的 I/O 不返回时继续持 lease 等真实返回。不承诺任意 I/O 强杀、撤回已发送字节、handler返回后的异步写或 Hijack 后连接 ownership。

## 复审条件与交回

两目标 SHA 改变、稳定 Runtime API/取消/Release/Verify 语义改变、上游验收改变、或新增会影响本结论的消费者证据时，对新固定修订做有界复审并保留本报告。若发现 finding，原作者整改、原 Reviewer 核原 finding 与新回归；不由 Reviewer 修改生产产物。后继 root/PG/E2E 必须取得各自真实证据，不能从本组件 PASS 推导。

交回 Main：本目录是唯一自有 Temp primary evidence。`report.md` 为 primary report，`receipt.json` 为 primary receipt；`readset.json`、`fixed-inputs.json`、`risk-results.json`、`author-evidence-audit.json`、`checks/`、`overlay.json`、最终 overlay 测试文件与 `evidence-index.json` 供复核。**STOP / FROZEN**：未改生产源码/Task/Plan/设计/基线，未关闭任何任务，未启动下一活动。
