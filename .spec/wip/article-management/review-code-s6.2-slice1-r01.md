# S6.2 切片①独立 CodeReview R01

**slice1CodeReviewGate：CHANGES REQUIRED。1 MEDIUM，0 HIGH/CRITICAL。** 结论只针对 Identity 局部能力及其接口可承接性，不评 Article、production root 或真实 PG 是否完成。

## 固定对象与检查范围

- objectType：`code`；作者 Turing，本执行者非作者；原两文件不存在，本次全文审查新增内容。
- 工作区 `D:\AID\VSCode\Codex\Han`；`main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- `internal/identity/article_disposition.go`：645 行；SHA256 `80210A39D31552C085BD2728F05859064536D96AB0DE51EB4B239EF300150983`。
- `internal/identity/article_disposition_test.go`：978 行；SHA256 `7CB2679C36858728A4555B2A0C043907E223D543FD3AE769EFB95CC940C2E9DF`。
- 上游：design r16 SHA256 `CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC`；test-plan r14 SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C`；本会话 Design/TestPlan R02。上述哈希本轮均核对一致。
- 已补读 coding-standards、no-hardcode-policy、go-service 对应规则及适用可观测/审计规则；复用此前完整读取的 Reviewer、add-code-review、评审清单、抽象原则、安全红线和当前 Workflow。

Round 1：完整阅读两个新增文件，核 SQL/UUID 排序/NOWAIT、身份生命周期、目标错误分类、前后状态、Notice/recipient、审计/计数、调用者事务所有权、失败/重入及全部测试夹具和断言。

Round 2：只沿必要实现与上游合同，核 identity model、registration 规范化、canonical deletion 校验/释放、既有作者状态枚举和 audit.Store.Append；推演未来 Article 私有引用通过固定 bound verifier 适配本接口的可实施性。没有扩成全库审查。

## [MEDIUM] 有可信 orphan operation 来源的 absent Article 也被无条件拒绝

- finding ID：**AM-S62-S1-R01-F001**。
- 对象/修订：上述两个固定新增文件，重点为生产文件固定 SHA256。
- 位置：`internal/identity/article_disposition.go:401`～`:405`；调用顺序 `:350` 先验证 reference，`:357` 才调用可信 verifier；后续 orphan 校验 `:508`～`:512` 同样只容纳存在的前像。
- 触发：finalizer/replay 已锁定可信 orphan operation，source=S，tombstone/article/version/operation 绑定有效；当前 Article 经同 tx 回读已 absent。按 r16 幂等完成审计/operation 后，应使用 operation 的可信 source 核计数并条件清理。调用者如实传 `BeforeArticle.Exists=false`，即使固定 verifier 可以证明 operation.source、当前 absence、审计和计数，仍在调用 verifier 前被拒绝。
- 证据：`validateArticleCompletionBefore` 对所有动作首先执行 `!before.Exists` 拒绝，无 delete_complete 的有来源例外。把不存在的 Article 填成存在的 held 前像会违反可信前像合同，不能作为 root 适配办法。当前测试 `article_disposition_test.go:621` 只覆盖 absent **且 operation.source=NULL** 的拒绝，没有可信 orphan source 的 absent 正例；现有 delete 正例均使用 `:303`～`:312` 构造的存在前像。
- 违反依据：design r16 `:1914` 只要求“现存 pending 前像”满足 version+1；`:1927` 接受确认 absent 的零行幂等，仅在“没有前像且 operation 无来源”时禁止生成清理事实；`:1963`/`:1964` 要求 absent 幂等完成及 orphan 可信来源的同事务清理；`:1975` 保守分支限定 absent **且无来源**。这也是本卡要求核对的 future consumer 可实施性与评审清单中的幂等/契约边界。
- 影响：接口无法如实承接已批准的有来源 absent 分支；后续 finalizer 若遵守清理调用合同会持续失败，若跳过则遗漏应有条件清理。这里是当前 Identity 能力的合同缺口，不声称现役未接线代码已出现线上故障，也不把无来源 replay 变成可清理。
- 最小建议：区分现存前像与已确认 absent 的 delete_complete。仅在固定可信 verifier 证明当前 tx/锁定 operation/tombstone/Article absence、orphan kind/source、完成审计及计数后，允许用该 operation 来源清理；仍拒绝 absent 且 ordinary/violating source=NULL、伪造/不匹配来源和跨 tx。同步调整后续 orphan 分支对前像的必填条件，不新增 registry/token/schema，不允许伪造存在前像。
- 复审条件：原作者提交新固定哈希，补“absent + 可信 orphan source”成功、仍有 pending 保留、成对已清理重入，以及无来源/错来源/错 operation/跨 tx/核验失败拒绝的局部用例；核清理/审计仍使用 caller tx、失败由 caller 整体 rollback。生产 provider、真实 PG 与 finalizer 联合验证仍属于后续切片。
- 取证性质：此分支由已读取的确定性前置条件与上游合同直接确认；本轮没有新增测试文件或声称已运行上述缺失正例。

## 其余有界结论

- canonical UUID 字符串先规范验证再排序，与其字节序一致；先取得全部 User，再按 User/kind 锁全部 Identifier，SQL 均 NOWAIT。候选身份也是 User→Identifier；无内部 commit/rollback、pool 查询或第二事务。55P03、scan/rows/SQL/verifier/audit cause 经错误链保留，rows 关闭。
- 来源须 deleted；目标缺失/非 active 使用 `ErrArticleTransferTargetInvalid`，缺失联系方式、标识漂移与 DB 故障不混入该 sentinel。422 的 root 映射尚未实现/验证，不以本卡代替。原始用户名/邮箱与 normalized Identifier 的大小写处理可追到 registration 的 ToLower，不据 display 大小写误报漂移。
- 现存 Article 的前后版本、author/source/publication/deletion、违规组合/保留、Notice 四关联字段、双 kind/pending/地址，以及 completed operation/审计/非负 active/pending 字段均有本地校验。计数查询是否同一快照、审计/快照是否真的存在，由后续可信 Article provider 在 caller tx 证明，本地 stub 不能证明。
- 两个计数均零且邮箱存在才条件清理；User/email Identifier 各影响一行，否则报错；username/phone 不释放。释放审计走现有 audit.Store，含同 trace 与最小 kind metadata，不含邮箱。已清理成对空值与同 tx 重入不重复释放；中途失败保留调用者 rollback 责任，当前模拟事务测试不能证明真实 PG 原子性。
- public reference/事实 DTO 本身不是来源证明。构造时固定 verifier 可由后续 root 的固定 factory 创建 bound 实例，捕获 Article 私有、当前 tx 锁内 native reference 并验证/映射；不需要 global registry 或新 token。现存前像分支未发现阻止这种实现的接口缺陷；F001 的无条件前置拒绝除外。transfer 重入时来源邮箱已清空，原快照地址的真实性还须由该 bound verifier 保留的原锁内联系证据核验，不能仅重新包装 public DTO。

## 验证、限制与交回

Reviewer 现场执行：`go test ./internal/identity -run '^TestArticleDisposition' -count=1` **PASS**；仅两文件 `gofmt -l` 无输出。未重复全仓/race/vet/Manifest，未连接数据库或运行其他专项。

Lead 提供的独立现场结果：22 顶层/132 通过节点专项 PASS、七项 Manifest 按序 PASS、667 文件起始哈希对账无其他代码漂移或删除。作者提供 RED/GREEN 与包/race/vet 结果。以上按提供来源记录，未冒充 Reviewer 自行重跑；已有绿色用例不覆盖 F001 缺失正例。

没有写测试或修复代码；唯一新增本报告，未委派、Git 写入或清理。交回 Lead/原作者处理 F001，以新修订独立复审；本局部 Gate 不代替 Article/root/真实 PG、完整 S6.2 或人的风险接纳。报告完成并核对固定哈希后停止写入。
