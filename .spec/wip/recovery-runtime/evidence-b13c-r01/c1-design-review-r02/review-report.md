# B13C-DESIGN-R01-F001 定点独立复审 R02

结论：**B13C-DESIGN-R01-F001 CLOSED；新增 findings 0**。仅原设计 finding 的修复范围通过本次复审，不是完整 C1/C2 实现、安装、测试、部署或人的确认 PASS。复审由原独立 Reviewer（Dirac、本会话）执行；无作者自审冒充、无子委派、无新 Agent ID 声明。

## 固定对象与前序证据

- objectType：design；development-delivery 现有 review 环节续接，不重新 Route，不推进其他活动。
- workspace：D:/AID/VSCode/Codex/Han；branch：main；HEAD：6da8abbceb0fa197364810a92a8fa2d97b38ddca。
- 修复对象：.add/project/knowledge/design-workspace/article-management/design.md，r25，SHA256 **1D1CEE350E4FD93B3B8700FD2D48F00BA2E70561FDCD8D0630B2741CF7A45F31**。
- 基线：r24，SHA256 **69C56796285F8187FBF2261D90183A519962A7F3EAF0A7788D558370C7CAC619**。修复包 article.original.md 与本角色 R01 所审 after 字节相同；article.final.md 与 live 字节相同。
- 冻结关联对象：identity-auth/design.md r53，SHA256 **DD61F4B89C94CE91F1E1317056ACAC1D7F8F464B2963DFDC59FF9C358590A6F0**，未变。
- R01 canonical：.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-review-r01/review-report.md，SHA256 **8BC76F4789D4C32DD58F5B273CB4E0B6AAEE7BD95E2B33608FAE95762A0DD9BB**；receipt.json SHA256 **A9982E68C2ED6A57774030BC8CA6E412F97D8CD360FA9D4729DFF8510EE230B6**。本轮实际完整读取二者，按原 finding 的修复建议和复审条件核对。
- 修复输入包：.spec/wip/recovery-runtime/evidence-b13c-r01/c1-design-author-f001-r02/。article.diff SHA256 **D37B1906D18FB33AEEE939388543DCE74FD633D3AC2AEE1065255F961A296114**；完整45行实际阅读，独立由原始/最终字节重新生成并精确匹配。
- 作者归属仍为 Banach / add-feature-designer / 01a18389-57ab-7b70-8c86-82a61e7729f1；来源为修复 receipt/Main wrapper index，非人的签字。Main 的采用/冻结与作者 STOP/Hostclosed 按本轮用户明确交接；未把 receipt 文字当 Host 运行状态的另行实测。

## 两轮范围承接与本轮实际检查

R01 的 Round1（完整两设计差量与受影响正文）和 Round2（直接 Runtime/admission/journal/对应测试与冻结 Article 消费者/原失败探针）保持原范围与限制。本次没有重审未变整库或重跑 R01 全部来源。

本轮 Round1：实际完整读修复 article.diff、作者 receipt/readset/static/baseline/hash/index；核修订、来源及唯一技术变更。独立逐字节比较整个 r24→r25：仅两个版本字段、一个追溯段、一行状态记录和 cold-open 供证条件发生五处变更，无别的替换。4个 Go 合同块、整个 staged→pending→published→committed/C2 协议段、C2 最低验证段、确认尾部均字节相同；BOM/LF、diagram/fence 标记序列保持。identity r53 完整 SHA 不变。

本轮 Round2：有界重读 Article r25 正文2320–2385，与 identity r53 的1676–1686及3875–3887交叉核对。逐调用追三条本地协调→供证→Runtime 后证路径；检查唯一 grant/ctx、原 namespace 长期锁/同 handle、closed/drain/exclusive、normal 拒 pending/零 repair、第三头/漂移拒绝、Runtime 不开放与独立后证不得省略。真实源码/测试的 R01 阅读仅作为原 finding 依据继承，不刷新活跃 Article 源码字节，不从 activeArticlebytes 签实现 PASS。

## 原 finding 复审

| 项 | 复审结果 |
|---|---|
| 稳定 ID / 原严重度 | B13C-DESIGN-R01-F001 / MEDIUM |
| 原触发 | 合法 bound Install 的控制协调已完成，Runtime 第一次读取供其后证用的 snapshot，却被“Runtime 后证已成立”反向阻塞。 |
| 原影响 | 两 owner 按原各自前后置无法唯一实现正向供证；属于设计合同顺序冲突，不是已运行死锁或权限绕过。 |
| 修复位置 | article-management/design.md:2336（r24原位置2333）。保留全部前段许可/绑定/打开限制，末段明确 article 本地核验后即可供观测，且“不等待 Runtime 独立后证先成立”。 |
| 关联核对 | article :2338、:2340、:2364、:2366、:2368、:2373、:2379、:2383；identity :1682、:1684、:3880–3885。 |
| 修复与依据 | article拥有本地协调/完整纯读验链及本地Anchor回读；Runtime拥有组合独立Anchor/snapshot/原批准前缀和不变Store/recovery/commitment后证。供证与接受分离，符合原R01最小建议、checklist §2、design-doc-writing-style §J、abstraction-principles §1/§5。没有新增authority/ready/setter。 |
| 结论 | **CLOSED（设计缺陷）**；未发现本次修复引入的新缺陷。 |

以下是本 Reviewer 的纸面逐调用核对，全部非运行证据：

| 起点/路径 | 本地动作与供证 | Runtime 接受条件与结果 |
|---|---|---|
| pending H0→H1 | 同一当前 install grant/ctx 与namespace长期锁下核唯一候选和operation；按原协议完成发布、commit/回读；完整纯读链与本地Anchor精确绑定、head=H1、pending=nil后，接入同一handle并提供H1/revision snapshot。 | executor实际I/O结束后，Runtime读取独立Anchor及该snapshot，核H1仅为原NewHead、原批准前缀/精确绑定和其他来源不变，才接受。首次snapshot不依赖此结果。 |
| 撤回 pending→H0 | 无可验候选时仅撤销未确认pending；CAS产生新不复用revision。核本地H0完整链/Anchor和pending=nil后，同handle供H0/新revision，不确认删除。 | Runtime须接受的仍是原OldHead且revision确已换新；照常核全部独立后证。没有以head相同跳过token检查。 |
| committed exact | 本地保持精确head/token，完整纯读核验，同handle供原committed snapshot；不为供证发CAS/修复。 | Runtime核head/token未换、pending=nil、完整原批准前缀和其他来源未变，再接受；不是executor自报nil即可成功。 |

三条路径均明确：当前公开 Install 的独立前证→grant/PhaseContext→精确checker→受控open/原namespace锁→本地协调/核验→接入同handle→snapshot观测值→Runtime独立后证→接受本次Install→撤grant/保持关闭。后证不作为其输入snapshot的先决，也未被绕过。

## 保留条件与回归边界

- grant/ctx：raw permit与ctx归同一Runtime当前install grant/代际/用途、精确安装/id/epoch；正常lease、startup/replay、错实例/过期均不授协调权。深拷贝Pending/起始观察不成为授权，未改。
- namespace与排空：控制handle持续复用已取得的namespace锁，不嵌套重取；同部署closed/drained/exclusive和普通/全部维护/三worker/子工作真join仍为前置。取消/Stop先撤权、在途I/O真join、不早释namespace/deployment资源及mutex不跨I/O/join保留。
- normal：仍只用普通Open；遇pending/stage失败，零初始化/CAS/repair。修改仅明确受控局部供证，不向normal开放未committed数据。
- Runtime后证：独立Anchor/snapshot精确绑定、原批准前缀、完整Store/recovery/commitment值与tokens不变，第三头/ABA/未知/伪nil均拒绝。成功只确认本次操作，不启动worker、不授普通lease、不清其他cause，未来normal仍完整重新证明。
- C2：协议/验证段和接口块字节不变，R01对C2无新增finding的结论不扩大。当前独立C2修复/PGtest窗口的代码和运行结果不在本次复审内；C1正向实现仍禁。本报告不签C1/C2代码完成，也不重开旧HIGH。
- S49/Ptolemy/Goodall：仅按用户交接作为并行边界；未读活跃/冻结peer源码、未重跑其Gate、未判断其当前进度。

## 方法复用、完整性与限制

复用的是本角色在同一连续会话中已完整取得的8项同版本正文：reviewer.toml、add-code-review、code-review-checklist、abstraction-principles、design-doc-writing-style、decisions-guide、产品安全红线和add-delivery-workflow。当前每项哈希均与本角色R01的readset完整覆盖匹配；精确hash/原阅读区间在reused-methods.json。没有将作者readset、父会话或文件存在当本角色已读证据。仅文档修复，不增加Go实施/profile/构建前置。

独立静态分析结果见static-checks.json；逐范围文本读取见readset.jsonl；完整字节处理另列static-readset.jsonl，不能把后者当全文语义审查。修复包Main wrapper的8个制品hash/长度全部匹配。本次不声称全工作树无他人变化；只核固定对象和指定静态不变块，仓内零写。

NOT RUN：任何Go/build/test/race/vet、PG/SQL/PGtest、服务/HTTP、C1正向组件实现、完整fresh/prepared Install、root/真实维护drain、部署、provider/批准主体、POSIX/fsync耐久、PlantUML语法/渲染、性能/E2E。未读取ENVsecret/DSN/key，未网络/下载/安装、Git写、cleanup/删除/移动或子委派。没有伪造运行PASS。

设计仍draft，人的精确接纳unknown。后续实现仍需原授权窗口下真实producer→consumer/三条协调路径、取消/Stop/锁/join、C2完成边界及其回归证据；这是原有下游义务，不反向作为本设计F001关闭前置。若后续再次改变供证顺序、grant/资源边界或Runtime后证，应固定新修订后重新评其差量，不沿本结论外推。

**STOP**。交回Main接收本定点复审报告；本Reviewer不改SSOT/Task/Plan、不替采用、不代人的签字、不授源码START、不关闭C1/C2实现。原F001设计缺陷已关闭，新增0，无本次新增待决定项或非阻断建议。
