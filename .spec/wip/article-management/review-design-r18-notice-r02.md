# Han S7.1 通知设计 r18 独立复审 R02

**DesignReviewGate：PASS（固定 r18 通知设计及 R01 F001 修复范围）。**

**原 HIGH `AM-DES-R17-NOTICE-R01-F001`：CLOSED（设计缺口已关闭）。新增 findings：0。**

本结论不是实现/测试执行、TestPlanGate、部署、production readiness 或 E2E PASS。原 R01 保持不可变，失败与修复链由本 R02 追加记录。

## 对象、修订与权限

- objectType：design；原作者 Ampere 已交固定 r18 并停写；本次由原独立 Reviewer 复审，未参与作者修订，未再委派。
- 工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，现场核对一致；不以 HEAD 代替实际未提交内容。
- 目标 `.add/project/knowledge/design-workspace/article-management/design.md`，revision `article-management-design-ir01-r18`，SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`。
- 基线实际读取 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672\design-r17.md`，SHA256 `670EFF7E1206DA197E9DF557DB0C150A3D78C9952319F0E30F68CA2482FE644C`，与原 R01 对象一致。
- 原报告 `.spec/wip/article-management/review-design-r17-notice-r01.md`，SHA256 `1C51D766F9F2B811558F4BE124878875CC5A5FEBCEC49FD5E04B41C875382950`；本次全文读取，未改写。
- 唯一写权限及实际新增文件：本 R02。其它全部只读；不修 design、Task/Plan/test-plan、代码、Rule，不做 Git 写操作、清理、委派、PG 或 SMTP。
- 同一 development-delivery 的设计复审续接，不 Route。沿用本会话已完整读取的 Reviewer Role、add-code-review、add-delivery-workflow、code-review-checklist、abstraction-principles、design-doc-writing-style、task-execution-workflow、安全红线及相关 go-service/core coding、配置、可观测规则；按 design 对象适用，不要求先编译非代码成果。
- 上游仍为 R01 固定的 requirements US-IDENTITY-03.9、Story 7、accepted article boundary 及相关 ADR；本次重核其内容哈希与 R01 一致，定点重读 ADR-0022 的关闭合同。用户对有依据建议的采用授权沿用，不补造人的精确 accepted/checkpoint。
- Ohm 正独占写的现役 test-plan r15 未读取，未将其作为固定输入；本 R02 目前只有 DesignReviewGate，没有 TestPlanGate。以后是否追加测试规划评审按用户另行明确授权。

## 第一轮：原 finding 与 r17→r18 全部实际差异

使用 UTF-8 正文与逐段比较（SequenceMatcher，autojunk=False）固定全部差异：**新增15行、删除10行**。差异限于 revision/追溯、原 F001 整改行、§3.1.6、§3.3.8 production 第5～7项、§6.1、§8.1/8.5 及接收说明。逐项复核修复内容、对象/接口、取消前后置条件、共享预算、并发、异常恢复及交付范围。

原 F001 的触发是“初次完整发信轮同步阻塞 signal 消费，停止请求只能留在通道，整个批次后才开始 shutdown budget”。本次按原 finding 的关闭条件逐项核验，未以作者自述代替设计正文：

| 原关闭条件 | r18 实际落点及判断 |
|---|---|
| 首次 Reconcile 前已有同一 signal/父取消协调器 | `design.md:1801` 明定 runLifecycle 是 signals 唯一消费者，先检查排队事件，再派发受控初次调用；协调器同时 select 停止与完成，不同步等批次 |
| 工作 context 可被停止事件取消 | `:1801` 指定协调器持有可取消工作 context，传初次 Reconcile/后续 Start；不再假设 main Background 自行响应 signal |
| 清算前绑定共享截止，而非 Stop 后才开始计时 | `:1800–1801` 新增内部 `BeginShutdown(sharedCtx context.Context)`；首次观察到停止事件即固定唯一 deadline，先绑定再取消/Stop；重复 signal/调谐返回均不重置 |
| 不后继 claim/Send/Start/Serve | `:1800–1801` 指定每条操作与阶段推进的 context/停止检查；完成与停止同时可读时先停止，取消不能继续下一 recipient |
| 当前结果最多一次有界 fenced 保存 | `:1800` 限共享剩余时间与 DBOperationBudget 的较早者，允许必要 unknown 回读；不重启已开始保存，不再 Send，不通过 WithoutCancel 抹去共享截止；无共享 context 留 processing |
| 未 Start 的初次调用仍受控且可 join | `:1800–1802` 初次调用计入活动；完成通道能容纳取消后返回，协调器必须 join，不遗弃 goroutine；Done 不提前关闭 |
| join 后再关闭依赖，避免复用旧 startup helper 直接关 pool | `:1802` 明确初次调用和适用 worker join 后再关闭依赖；未 Serve 不等待不存在的 Serve 结果，仅清理已绑定 listener；共享期限耗尽仍记错并等待实际退出 |
| 错误、退出码与原消费者保持 | `:1802` 保留业务/metrics/password-reset/deletion 的相对错误优先级，加入 notice 后仍先于 signal/context；纯停止取消不遮蔽真实 fatal，保存/超时/清理错误继续非零；正常 signal 全部成功才0 |
| 其它视图和影响面同步 | `:619` 的进程描述、`:2442` 的验证建议、`:2572–2579` 的生命周期/测试职责与九文件接收范围同步；追溯记录只写送复审，未自行标 CLOSED |

**第一轮结论**：原 finding 所需的设计责任、输入输出、时序、资源归属与失败处理均已明确。r17 中“有界批次等于及时停止”的缺口没有保留到 r18。

## 第二轮：真实消费者可实施性与新增风险

从现役入口重新反查修复，不把尚未修改的源码当成已实施：

| 实际重读范围 | 独立核对 |
|---|---|
| `cmd/han-server/main.go:157–169,567–583,3508–3512` | main 确用 Background 与独立 signal.Notify 通道；现有唯一 coordinate 接收该通道及预算。r18 将监听前移到同一 runLifecycle，能消除原反例，无需第二信号消费者/root |
| `cmd/han-server/lifecycle.go:14–169,179–235,292–345,385–453` | 六方法、同步初调、结果容量4、Stop后才建deadline、旧startup直接关依赖及错误结构均是真实接缝。r18 明确这些位置的必要改造；资源取得/实际派发标记可在原 lifecycle 文件承载 |
| `internal/article/deletion_worker.go:121–230,233–342` | 初次 Reconcile 可在 Start 前访问DB；运行 Done 属 run goroutine，不能盲等未Start旧worker的Done。r18 分开跟踪初次调用与已Start worker，避免把“未Start”误当“没有活动调用”，同时不要求改删除业务源码 |
| `internal/mailer/smtp.go:79–169` | 现役 Send 有 deadline 和取消关闭连接的 watcher；DATA accepted 后 QUIT/cancel不改成功。工作context取消可送达实际Mailer；BeginShutdown先绑定共享context，允许将已受理结果按原fencing落库而不重发 |
| `internal/mailer/smtp_test.go:79–143`；`cmd/han-server/lifecycle_test.go:534–615,640–711` | 既有测试源码有阻塞取消、accepted/QUIT、启动顺序和Done后关闭断言，但 signal 主要在Serve后注入。r18新增启动期时序建议针对原盲区；仅阅读，未执行 |
| ADR-0022 `framework.md:669–681` | 同一关闭协调、共享deadline、真实Serve-loop退出后关pool、超时不降成功等上游合同未被修复削弱 |
| r17/r18 结构化分节比较与九文件表 | §3.4.8 通知状态机逐行相同；§3.4.4～§3.4.7受保护协作/删除/恢复正文逐行相同。无HTTP/schema/边界改动；九文件候选保持原集合 |

额外 `BeginShutdown` 是通知内部能力；现役共同六方法不需强加同名扩展给原两个 worker。production 必须在通知上具备该能力，并在 Stop 前传入同一 sharedCtx；该专用接缝及活动调用跟踪可落在既定 notice_dispatcher/main/lifecycle 文件，不需要新文件或修改 model、privacy、Mailer 接口、identity 或 S6.2 业务源码。实现时不能只将通知放入原六方法列表便声称完成该接线。

独立静态重放原反例及边界（**不是已运行测试**）：

| 触发时点 | 按 r18 应有结果 |
|---|---|
| 首次调谐前 signal 已排队/父已取消 | 先消费停止，不派发后继Reconcile/Start/Serve；按已取得资源清理 |
| 初次首个 Send 阻塞期间 signal | 协调器无需等待轮次完成，先固定sharedCtx/停止状态再取消工作context；取消到Mailer，不再claim下一条，join后才关pool |
| DATA已受理，尚未保存 | 保持accepted语义；当前保存最多一次、受剩余共享截止与DB预算限制，必要unknown回读；失败/超时暴露，留租约恢复，不再Send |
| 下条尚未claim或调谐完成与signal同时就绪 | 检查可读停止事件及工作context后不推进下一条/Start/Serve，不能随机先开放阶段 |
| 保存已开始、重复signal或deadline到期 | 不重启保存/延长预算；清算取消、超时非零；仍等实际调用退出，不在后台还用DB时提前关依赖 |
| fatal与停止取消同时返回 | 受控调用被join，真实fatal/保存/关闭错误保留，纯停止取消不覆盖其优先级，不伪装成功 |

**第二轮结论**：上述时序与现役真实消费者可映射；未发现新的可触发具体设计缺陷。原 F001 在设计层面达到 R01 约定的复审条件，故 CLOSED。没有要求用未来实现测试替代当前设计闭环，也没有据此关闭任何代码或测试任务。

## 固定证据与保留范围

核对 R01 表内所有固定上游/源码文件的当前SHA256：仅目标design变为r18，其余18个均与R01一致。未以动态test-plan参与该比对。以下列出本轮关键载体及补充消费者：

| 载体 | SHA256 |
|---|---|
| r18 design | `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3` |
| r17 before-image | `670EFF7E1206DA197E9DF557DB0C150A3D78C9952319F0E30F68CA2482FE644C` |
| 原R01 | `1C51D766F9F2B811558F4BE124878875CC5A5FEBCEC49FD5E04B41C875382950` |
| main.go | `4B9D83BA52E322E9E7A430F18A59DAA0FC651D4D4B21BEAFBA825E7E991A7569` |
| lifecycle.go | `980B5D82DEDFF0FE697082DC9F3184AC4B87CB53227F7446189F3E1BE0CF4888` |
| lifecycle_test.go | `B3AA659B1A98E0AE8A3EF09678309866DC3D46345B2A6C29A5A220D700B36DDF` |
| deletion_worker.go | `9E6BDD23089B41A3870DB0A35420589DD2210972C0610A43184D6E7C60F6612D` |
| smtp.go | `0C64C229B7EE8101B0DCAD95C14431DCAF8ED55809C9FBFD50AEE75AF0221199` |
| smtp_test.go | `656E12CB62C113C60AB074AE5374F9E206D226CCEE5B194BB1F0F8AA12B96D15` |

实际命令仅只读分支/HEAD/status、rg定位、UTF-8正文读取、SHA256、内存差异与分节一致性比较。本轮没有创建通知实现；`internal/article/notice_dispatcher.go` 当前不存在。原源码的同步初调/容量4仍是待后继实施的事实，设计PASS不代表这些代码已修好。

没有运行编译、Go测试、PG、TLS/SMTP、生产Host或E2E；没有读取/打印DSN或凭据。Lead上一轮报告的只读psql探针 `18.6|postgres|postgres` 仅是Lead提供的环境信息，本Reviewer未复核，也不能证明通知SQL、并发、SMTP或崩溃恢复已验证。测试规划r15仍待固定对象和独立授权评审。

继承R01限制：未全面重审无关正文/全仓消费者；未验证新增通知PG/TLS/崩溃、支持Host完整production启动、外部处理登记/SMTP部署值、生产配置/容量/留存、真实部署/E2E。现有Windows Journal/其它独立Gate不被本地设计PASS解除。未知部署事实继续保持unknown；不因其缺失或design仍draft机械要求重复确认。

无新增方向决策、非阻断改写建议或待作者修复finding。后续实现必须证明§6.1的channel/barrier取消矩阵、fencing与共享预算；其结果由相应实现/测试/代码评审闭环，本R02不登记任务、不替人签收。

## 交回与停止

交回Lead：**F001 CLOSED（design）；DesignReviewGate PASS；新增findings 0。** 该结论只绑定上述r18/hash；若首次调谐取消、sharedCtx顺序、join或错误合同后续实质变化，需按受影响范围复审。

本次仅新增本R02，原R01及其它成果不改；报告交付后停止写入。后续固定r15及报告写入安排等待用户明确，不提前出TestPlanGate。

