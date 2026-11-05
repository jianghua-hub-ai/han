# Han S7.1 SMTP 两文件独立代码评审 R01

**SMTP2CodeReviewGate：PASS（仅下列固定两文件局部）。Findings：0。**

不关闭父 S7.1、批次或 S7.2；Article3/root4 不在本次固定对象内。后续九文件仍需独立 joint Review，不能把局部 PASS 相加代替整体结论。

## 固定对象与依据

- objectType：code；作者 Franklin 已明确停写，冻结源码与交接 hash 一致；本次由非作者独立 Reviewer 执行两轮，未再委派。
- 工作区 `D:\AID\VSCode\Codex\Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，现场核对一致；实际未提交对象以内容 hash 固定。
- `internal/mailer/smtp.go`：SHA256 `28237089CE76B353CC01D36847378CF4775AA9281210175AAC59905DCC192853`。
- `internal/mailer/smtp_test.go`：SHA256 `9833F766F24CEA8067D9A2C46BB13C01DF2568DE5A7EA998675318EDCA2B964E`。
- 前像实际读取临时根 `C:\Users\J00010385\AppData\Local\Temp\han-s71-1a90c49b44f5450a82add4222dfd1672` 下 `internal__mailer__smtp.go` / `internal__mailer__smtp_test.go`；hash分别为 `0C64C229B7EE8101B0DCAD95C14431DCAF8ED55809C9FBFD50AEE75AF0221199` / `656E12CB62C113C60AB074AE5374F9E206D226CCEE5B194BB1F0F8AA12B96D15`，与此前已核现役基线一致。未把 HEAD 或未跟踪文件的空 git diff 当作前像。
- fixed design r18 SHA256 `CAC652FE545576AF6F93443681A62FB2362855474AFD86BF4CAE20F9077E0ED3`，重点§3.4.8阶段错误合同；Design R02 SHA256 `115289B8216BD5CB0AD49CB9B82B748E84555FD58C307AAFA272213F387B13E7`。
- fixed test-plan r15 SHA256 `71A0A6A5BC75062A50DEAD6BA310531F865B3AD6A4F7A5A1595D9896554A6C7E`，重点A-NOT-05a～g；TestPlan R01 SHA256 `29BC5E15647A2BB0C853A59B205080DE4114FAD2134F8D82F4FDD7111474E1EE`。
- 原S7.1START的Tasks/Plan hash现场匹配 `BBE4B5A80BCA9FB97A0BA990DAD0DE777C27EB172A9F6ED9488509F301E01F34` / `28594C7E3D80F8F0EFB5A9A98F26AE21B8B533E497103E77EAB48B62F7A2A7D7`；不修改任务状态，不以此扩权。
- 唯一可写及实际新增文件为本报告。其它全部只读，不修代码/TaskPlan/design/test-plan/Rule/旧报告，不做Git写操作、清理、PG、外部SMTP或新委派。未读取Nash正在写的Article源码作为固定评审对象，未对root4作评审。

沿用本会话已完整读取的Reviewer Role、add-code-review、当前add-delivery-workflow、code-review-checklist、abstraction-principles、安全红线，以及core/go-service编码、错误链、配置与可观测约束；按当前代码对象适用。没有Route或启动实施工作流，不把已有函数长度等风格存量当成本次缺陷。

## 第一轮：真实 diff、实现与测试

独立比较前像与冻结源码（SequenceMatcher，autojunk=False）：`smtp.go` +49/-6行，`smtp_test.go` +166/-19行。实际审查全部差异和必要上下文，不只读作者diff摘要。

| 检查点/位置 | 结果与证据 |
|---|---|
| `smtp.go:24–42` | 新增SMTPMessageStage及四个稳定值mail_from/rcpt_to/data_command/data_final；SMTPMessageRejection只公开Stage/Code，cause私有，Message/Mailer.Send及原sentinel不变 |
| `:46–64` | Error只格式化allowlist阶段和数字码，非法阶段打印固定unknown；Is只匹配ErrMessageRejected，Unwrap返回原cause，不打印或替换cause.Msg |
| `:169–184,192–199` | MAIL、RCPT、DATA命令、DATA最终回复四调用点各传对应常量；errors.As取得原textproto.Error且Code处于400..599才产生新类型；其它错误仍保留原包装/cause，不靠Error文本判断 |
| `:186–189` | DATA最终受理后仍忽略QUIT失败返回nil，未改变成功判据；取消/握手/认证等路径未套新拒绝类型 |
| 生产transport与资源基线 | NewSMTPMailer、interruptConnectionOnCancel、secureClient分段与前像逐字相同；其它Send行为除四处Stage参数外保持原样。未新增连接、TLS配置、重试、timeout或业务决策 |
| `smtp_test.go:253–334` | 四阶段真实协议路径各测4xx/5xx（含RCPT421），cause携带canary；另用4×6个阶段/数字边界组合核类型/阶段/码、Is/As及原cause和protocol指针身份，包含400/499/500/599 |
| `:336–400` | 250/354/399/600、EOF、context取消/deadline不取得新类型；非法Stage打印固定unknown；%s/%v/%+v及上层%w包装都只含预期固定文本，不泄露canary |
| `:25–175,402–531,595–747` 相关差异/原断言 | 阻塞I/O取消、Auth、greeting、非负回复、STARTTLS/证书/缺deadline增加无typed rejection断言；accepted后QUIT拒绝/断连及取消保持成功；旧传输安全断言未削弱 |
| RED→GREEN独立差异 | RED已具类型/方法声明及四处Stage plumbing，但保留旧classifier返回；GREEN生产唯一变化为该返回改成SMTPMessageRejection。测试后续新增QUIT/安全回归、非拒绝用例改可读名称，32个原失败leaf的类型/打印断言没有删除或放宽 |

第一轮未发现直接正确性、安全、错误链或范围缺陷。新的Stage只提供适配器事实，没有在Mailer中擅自将4xx/5xx决定为recipient重试/永久失败；该业务分类仍由后继通知消费者按固定设计承担。

## 第二轮：现役消费者、兼容与证据反查

在允许依赖范围内实际读现役消费者，未沿动态Article扩散：

- `internal/mailer/mailer.go`原purpose、Message、Send、sentinel合同；本次hash `A2F103D6AF3664288D0C4F460EC5733E33FA6461FB8554A189FBC3194C5B4D0D`，未改。
- `internal/identity/password_reset.go:327–435`：真实消费者在Send失败时只用errors.Is(ErrMessageRejected)区分显式拒绝；保持rejected/unknown各自的证据事务、失败与uncertain公共语义。新类型Is满足该判断，原textproto仍可As，不需改身份业务代码。
- `internal/identity/temporary_credential_resend.go:454–508` 与 `administrator_create.go:446–492`：管理员重发/创建在任意mailErr时回滚并按旧方式包装公共错误；未依赖旧拒绝文案。附加类型不改变nil/非nil或回滚/凭据生效语义。
- `password_reset_test.go:1451–1527,1529–1616,1618–1694,1696–1770`：实读初次reset/ordinary resend、S11真实resend消费者、非分类协议错误unknown、拒绝证据失败仍unresolved的断言；不只是符号匹配。
- 在identity/mailer范围检索ErrMessageRejected及mailErr.Error字符串消费，现役生产分类点是上述password_reset.go的errors.Is；未发现依赖旧错误字符串的分支。旧identity测试使用sentinel stub，独立证明其消费合同；真实SMTP类型和cause的兼容由mailer协议/错误链测试证明，不把stub说成真实SMTP集成。
- 定点重读fixed r18 `design.md:2175–2189`、r15 `test-plan.md:338–348`：四阶段、400..599、原链、打印安全、Auth/greeting/TLS排除、DATAaccepted/QUIT都吻合。MAIL/RCPT/DATA的业务bucket、队列状态与通知日志脱敏不在这两文件内验证。

本轮没有新建共享authority、改变transport安全或旧消费者契约。原cause继续可经Unwrap/As取到原始Msg；安全打印保证针对SMTPMessageRejection自身及普通包装。未经分类的Auth/greeting/I/O错误仍按旧合同返回，不意味着所有Mailer错误或调用方手动解包打印都自动脱敏；后继通知消费者仍须遵守r18禁止输出原始错误的合同。这是已设计的责任边界，不是发现的新缺陷。

## 原始验证证据核验与归属

所有路径均在上述临时根。Reviewer独立逐行解析5份原始JSONL（无坏JSON行），核run/pass/fail/skip、必需顶层/leaf及包终态，核对应文件hash与delivery-summary一致。不是仅引用Franklin或Lead摘要，也不冒充本Reviewer重新执行。

| 原始日志 | 独立解析结果 | SHA256 |
|---|---|---|
| smtp-red.jsonl | 4顶层、43run节点，32失败leaf/34失败节点，0skip，包fail；记录exit1 | `A53C07DC27224AF501D0B071F62BD49C592910C13B1BAF8385ABB5FA853E13D1` |
| smtp-green.jsonl | 17顶层、72run/72pass，0fail/skip，包pass | `2A5C8977990E44DB4A037684728811D067510704D5800062A100147B6BC02025` |
| smtp-race.jsonl | 17顶层、72run/72pass，0fail/skip，包pass | `366DEA6D9216B5A7E35510579217C7772AECB417B6B860EE4165DB29D93175DF` |
| smtp-identity-green.jsonl | 4顶层、11run/11pass，0fail/skip，包pass | `FF7559DA88F9849849059F289819A2A0CFFF40336D6EC878A190B2EEE008A665` |
| smtp-identity-race.jsonl | 4顶层、11run/11pass，0fail/skip，包pass | `7F233D54FD4A4255BE3AD0E9B2746DA4B1F6D0439414700386E217C4575FBB0B` |

RED的32leaf为8个协议阶段正负例与24个cause/边界组合，原始Output明确报“missing SMTPMessageRejection”以及三种格式/包装打印超出固定文本，不是编译失败或缺环境。RED生产快照hash `4278AA59BB58E81B96B1EF9E698A4F3575D3C932F3AEBE8C399D2EDA144BD702`；RED测试快照hash `9E179BAF934DB9608766C3C2A3209E25F992AE21E059EB12F7D86787D1F030CE`，已与冻结GREEN比对。

GREEN/race的17个顶层真实覆盖四Stage、cause身份、非拒绝边界、unknownStage、Auth/greeting排除、TLS/deadline/取消及accepted后QUIT；所有run节点都有pass。identity四顶层为ExplicitRejectionRecordsStableFailure、UnclassifiedSMTPProtocolErrorsRemainUnknown、ExplicitRejectionFlowsThroughS11ResendConsumer、RejectedEvidenceFailureStaysUnresolved，均带TestPasswordReset前缀，不是零测试或近似包级ok。

执行归属Franklin；summary记录的命令为：

- `go test ./internal/mailer -run '^TestSMTPMailer' -count=1 -json` 及加 `-race` 的同包命令。
- `go test ./internal/identity -run '^TestPasswordReset(ExplicitRejection|UnclassifiedSMTPProtocolErrors|RejectedEvidenceFailure)' -count=1 -json` 及加 `-race` 的命令。
- RED使用 `^(TestSMTPMailerClassifiesExplicitMessageRejection|TestSMTPMailerMessageRejection_)` 精确选定失败组。

`smtp-vet.txt`、`smtp-build.txt`、`smtp-format-check.txt`、`smtp-diff-check.txt`的实际文件均存在且为空、hash为标准空文件摘要，匹配summary记录的对应包vet/build、两文件gofmt及diff-check exit0。退出码来源为作者summary和交接，JSON包终态由Reviewer独立核实。git diff --check不覆盖原未跟踪文件，不能单靠该空输出证明变更正确；本评审采用前像逐段diff与实际测试/正文核验。goimports/golangci-lint未安装未运行；未跑全仓Manifest七项，按本局部授权保留此限制。

Reviewer本轮实际命令只有只读branch/HEAD、rg、UTF-8读取、SHA256、前像/RED/GREEN比较和原始日志解析，没有运行新的Go测试或检查。既有普通/race已经覆盖当前变化且未出现新concern，未无理由重复全量。没有连接PG或外部SMTP；上述mailer执行为net.Pipe上的verified TLS测试，不是S7.2 loopback TLS集成。

## Findings与交付限制

**Findings：0（CRITICAL/HIGH/MEDIUM/LOW均0）。无新增方向决策或必须改写的非阻断建议。**

本PASS绑定两文件hash、现役直接消费者与上述普通/race证据。未验证动态Article3、通知recipient状态/lease/DB事务/审计/commit unknown、真实PG、多会话loopback TLS/子进程崩溃、root4启动/共享deadline/join、完整Host、生产处理登记/凭据/部署或E2E。未给Nash/Lorentz动态或尚未交付制品Gate，也不以局部结果关闭S7.1/S7.2。

后续若这两文件、错误类型/常量/cause链或直接消费者合同实质改变，需按新固定hash复审；若出现finding交Franklin修复后由原Reviewer复审。本次无待修复finding，不替Lead验收或后续实施授权。

实际消费者固定hash：password_reset.go `5F2D0E1B534E1A4F47AF25675FB94AA3B2830A01C408F6C615E41FAB74A81575`，password_reset_test.go `59AAC51742A61ADD318D8ADE5527231D4D5A198790687B101A35A5A843666E2A`，administrator_create.go `DD1D1DA23CB9D9E13E3200E09867B2A2C2F6BCBF06894C46074F60AC94D75813`，temporary_credential_resend.go `EF7F9CBC121A18040BAA937E9217C53CE961C7072166019AB6863B4B1E9AD50C`。原Design R01/R02、TestPlan R01、设计/测试规划及原START内容保留。

交回Lead：**SMTP2CodeReviewGate PASS，0 findings；仅本两文件局部。** 唯一新增写入为本报告，交付报告hash后停止写入，保留后续九文件独立joint Review。

