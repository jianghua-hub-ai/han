# S8.2 / B9F 独立 Planning Review R01

日期：2026-10-28。Reviewer：Pauli，独立于原 Planner Ptolemy、开发准备作者 Dalton。objectType：`design`（任务/批次实施合同），复用当前会话已完整取得的 `add-reviewer` / `add-code-review` design 分支及适用 checklist/abstraction/安全规则。本轮是正常环节接力，未重新 Route，不代替设计或编码。

**结论：CHANGES REQUIRED，限 Guard 未决项的规划承接；1 项 MEDIUM。** S8.2 的九文件、内层 U/C/P 准备、统计/归属/DAG、真实 PG 与检错证据合同可承接；A-ADM2-18 的通用 readiness/recovery 请求保护承载及整卡关闭仍受限，不能给整卡无条件 Planning PASS。本结论不阻断已可行的内层只读准备，也不发 CODE/LOCALPG START。

## 1. 固定对象与依据

固定工作区 `D:\AID\VSCode\Codex\Han` / `main`，实际只读核验 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

| 固定对象 | 实际 SHA-256 |
|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `235732795BB65E64598DAE7364349BA9AE2EF428F2704DFC70ED7BA9F0D348A0` |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `902A404DE402A3658D960524A550095272E268A338A109A8845A9913457A3561` |

基线目录 `C:\Users\J00010385\AppData\Local\Temp\han-admin2-metadata-8a4dd71a156c4cca8b22627e9d40fa49\` 的 `before-manifest.json`、`tasks.before.md`、`plan.before.md` 已实际读取。before-image SHA 分别为 `BF2F0981A9FD612B424AB4C5FA97F50365CAF357000946C030E4258E1EABCD6B`、`5ABE2696CEC848D17247792AE7674756326A4B1BC1A4AB7FB540DB69E214534F`，与 manifest 一致；本轮按 before→固定当前字节生成并阅读完整实际 diff。

固定上游五项均重新核 SHA 匹配：design r20 `6884431B6164140E5593A8253AC20E8EB1C59C1AD71B86558B6D1A8C99BF037A`；stories r09 `A2BA7746F30B1C9B60DB4337995FC0A0E52DD55BFF984BAA784C20331D80C569`；test-plan r16 `6941C1D7B96F78B341CBB7B88AE74D548CF860CFEAD37D3D92E208548D896E38`；Design R02 `5B2E39D0A491B173BB651AA20E966BE40A0E03DFC1645E8D3CFFB0594B9E62B0`；TestPlan R01 `267CAA23481E31C4ECF3654FAD851772747B5ED6A04DB50823F5E973A978CC1B`。既有有界设计/测试规划检查复用，不扩成重新评审历史或批准当前实现。

新证据为 Dalton G0 `C:\Users\J00010385\AppData\Local\Temp\han-admin2-prep-93f3e966a7294f238ddfa37ce6d80f06\prep-admin2.md`，重点完整核 §8/:168–175 与末次对照。只读准备已 completed/stopped 的最新状态来自本次用户/Lead 交接；固定 metadata 中“G0进行”是登记时状态，不能拿它虚构 CODE/PG 已 START 或 G0 已通过。人的 accepted/checkpoint 仍 pending。

## 2. Round 1：完整 metadata 增量与任务合同

实际 diff 仅涉及任务统计、S1.5 的 Admin2 gap/真实新增 requires/停止说明、新 S8.2，以及 Plan 的总数/B9F/相关交接与 DAG；没有替旧卡更换关闭条件或删除原证据。

| 核查项 | 实核结果 |
|---|---|
| task 总量/三态 | 实际 59 条唯一任务；44 `[x]` / 3 `[-]` / 12 `[ ]`。active 精确为 identity S3.3、identity S3.5、article S8.2；旧任务状态无变化 |
| 模块/类型 | bootstrap 1、identity 35、article 23；按真实类型字段归一核开发 38，S8.2 是唯一新增开发卡 |
| 身份/旧关闭/注记保留 | 35 张 identity 卡、全部既有 closed 卡正文逐字节比对无变化；自原历史注记起的后缀字节完全相同。未覆盖其他会话身份独立改动 |
| 真实登记/认领 | `[-]` 及开始日期对应 Lead 实际只读 G0 认领，Dalton ID 有来源；不等于代码/本地测试/PG START。独立 Planning/Code Review 与后续 Lead 信号分开 |
| 唯一归属 | 只取 Plan 正式批次外层表 :51–80，59 条任务各出现一次，无遗漏/重复；S8.2 唯一 B9F，S1.5 仍 B10。后文证据/历史表的重复引用不算第二次安排 |
| 实施单元 | 一卡五生产+四测试，局部/真实 PG 功能验收同单元；未按 Case/文件拆碎，也未将其当跨服务部署 E2E |
| S1.5 | 只补真实 S8.2 requires/缺口落实；原十项及此前 S1.6 保留，四 contract 文件/16 endpoint/三投影/ACL/S1–S16/审计/producer/source 责任不缩，Admin2 交付不自动关闭 S1.5 |
| 固定写权/停止 | 九路径上限、旧 reader/迁移/查询合同不扩、未知 writer/输入漂移停相关范围、后续 CODE 与 LOCALPG 分信号均有明确合同；Guard 的新增 G0 未决尚缺专门承接，见 F001 |

没有因任务尚未实施、两个新文件不存在、Case 全 NOT RUN 或人的签名 pending 本身列 finding。

## 3. Round 2：依赖、消费者、数据与命中

### 3.1 DAG 与九路径

实际提取 S8.2 六项 requires：`project-bootstrap-B0.1`、`identity-auth-S3.2`、`article-management-S1.2`、`article-management-S2.1`、`article-management-S8.1`、`article-management-S4.6`；对应任务均 `[x]`。S1.5 仅在此前依赖集合上新增真实 `article-management-S8.2`。以声明的 requires 检查未见循环，S8.2 无反向依赖；没有把 S1.6/S2.2 文件窗口虚构成业务 requires。

逐一核 before-manifest 的九路径存在性/摘要：七个现存文件均与登记值相同，两个 `admin_query_test.go` 仍不存在。生产五文件为 `internal/article/{query_service,handler,model,policy}.go` 与 `cmd/han-server/main.go`；测试四文件为 U 新 `internal/article/admin_query_test.go`、C 改 `internal/article/policy_test.go`、P 新 `test/integration/article/admin_query_test.go`、R 改 `cmd/han-server/main_test.go`。

本轮 root 两文件只核全字节 SHA，与 S1.6 final 前像一致：main.go `2B75C96E61F5BF758F68935F4656238CD8D891D6DC646C32B55EFD0C1A5A6A14`，main_test.go `092BD3BD78169A40780169A27948D3BD1C6E423406E2C8E658B0EFC59BF3BCBA`。**没有读取其当前源码/执行测试，不声称独立验证 14/20/query5/mutation9、实际 Guard 或 S1.6 运行结果。** 原 S1.6 closed 卡的交付/窗口释放记录与本次 Lead 输入相符，不重开、不要求其追加 PG/E2E。

### 3.2 r20/r16 → 新卡的可实施内层与证据

新卡引用 r16 唯一明细而非复制第二份 Case SSOT，四个候选 top、20 Case、38 组 Case×层映射保持：U17/C1/P16/R4。RawQuery 整体422先于 decoded cursor400、typed/default 分离、五 view/20 有向互拒、三时间戳/ID 交错与跨批过滤、内部扫描/公开返回 cursor、严格身份后批故障整503、合法 pending 先404、N001 新 HTTP500/旧409、Admin12独立值 oracle/同源 ETag、零业务写均按已审 r20/r16 承接。

U/C 的受控组件/真实 Handler、P 的真实 SQL→reader→service→Handler、R 的真实私有 factory/依赖构造与路由安全分开，spy 不替 SQL/真实构造，P 不冒称 production 注册；普通/race、integration tag、精确 `-list`/top/leaf 的零测试防假绿和七项 Manifest 的证据界线保留。新 top 未存在不要求现在编译 PASS。

P 载体已把前轮非阻断注意落实：新文件内承先 CREATE/commit/记录随机 schema，再完整0001～0009迁移，迁移失败仍可 retain；所有连接相同 search_path，合法 FK/CHECK/pending operation，真实 reader 后 fault、真实 SQL/rows、取消/join/pool 归还，动态 writer delta 与 GET 零写分开。无 LOCALPG 信号不读 DSN/连 PG；失败 schema 不清理，旧 fixture/迁移/reader 不改。

三类负控指向 RawQuery 丢错、身份补 active/成功半页、older-time SQL/公开 cursor 边界的可编译行为 FAIL，再当前源码 GREEN；不以编译错误、缺环境或硬编码失败充 RED，不回滚生产工作树。卡内命令与固定 r16 相符，最低检错方式有明确承载。以上是规划合同可行性，不是已执行证明。

### 3.3 新 G0 的证据边界

Dalton prep §8/:171 明确：已定位 Trace、HTTPAuthorizer/AccessGeneral、SessionStore password-reset Gate、启动构造/lifecycle；**未定位独立通用 readiness/recovery 请求 Guard**，不能以名称或路由 spy 声称其存在/已验证。本轮将其作为 unknown，不推断 Guard 一定不存在，也不把 password-reset 专用 Gate 自动等同于通用恢复开放门禁。

当前 Guard 现役调用链调查由 Lead 已委派的 Explorer 负责。本轮不等待该报告才做其余检查，也不重复跨身份恢复扫描；尚未发布的 `research-admin2-guard-r01.md` 不是本报告证据。设计/TestPlan 的先前 PASS 都保留过 root 未读/承接边界，不能将它们转成 Case18 实际 Gate 已具备。

## 4. Finding

### [MEDIUM] G0 新增 Guard unknown 未进入 S8.2/B9F 的专门实施/关闭门禁

- finding ID：`AM-S82-PLAN-R01-F001`。
- 对象/修订：本报告 §1 的固定 Tasks/Plan 两摘要，限 S8.2/B9F 承接合同。
- 位置：`tasks.md:1756` 的 readiness/recovery 实现要求、:1757 的全 Case 命中要求、:1781–1782 的启动/关闭/停止；Plan :128–131 的交接/门禁。直接依据为 Dalton G0 prep §8/:171。
- 现状：metadata 仍以固定设计合同要求完整 root 安全链和全部 Case 通过，专门列出了 PG 环境/授权 pending 与越界停止；却没有把 G0 新发现的通用 Guard 承载 unknown 登记到 S8.2/B9F，没有说明由谁接收 Explorer 证据、什么证据允许继续 R 层、何时必须回原设计 owner。通用“第十文件/旧 API/权限变化停止”和 PlanningReview 待审不能替代此已知具体缺项的处置。
- 触发：后续按本卡申请 CODE/LOCAL，执行 A-ADM2-18 `ProductionSharedChain` 时，只能核到 Trace/Session/password-reset 专用 Gate，却仍要满足“通用 readiness/recovery 实际生效”的全层验收。
- 后果：实施者可能把专用 Gate/spy 当作通用请求保护的证据，或在九文件内自行补未确认的全局控制；也可能为一个 root 承载缺项无差别暂停本已可行的 U/C/P 准备。当前未发生代码写入/运行失败；这是已知未决事实没有进入可执行规划边界的缺陷，不是断言 Guard 已被证明不存在或新 GET 已有漏洞。
- 违反依据：适用 checklist 的真实消费者/公共契约证据、design 分支的前置/异常/责任边界和 Rule 的失败收敛要求；原 r20/r16 的“复用现役安全链、无新增 authority/旧行为不变”与本次用户明确要求的 Guard unknown 如实承接。
- 最小修复：原 Planner 仅在 S8.2/B9F 补一个明确的未决/G0 Gate，引用本次 prep unknown，保留 Explorer 调查的实际接收责任及结果位置；证据未定前，不把通用 readiness/recovery 的 R 验收或整卡关闭记 PASS，不自行新增/替代全局 authority。若确有既有 Guard，按真实调用链/状态 authority/失败行为/新 GET 接入点核承接；若不存在或需改变全局/旧消费者合同，停**该 root/Case18 范围**交 Lead 与原设计/测试规划 owner 决定，不能由 Planner/Dev 自选方案。内层 U/C/P 的只读准备保持可继续，代码/PG 是否另行放行仍由 Lead 明确授权，不因本 finding 获得 START。
- 无需为此新增虚构业务 requires、重开 S1.6/S2.2、增加任务/文件、修改 r20/r16 或提前等待全局调查后才登记 pending。方向改变时才由相应 owner 修订上游，不在本评审中代设计。
- 复审条件：交原 Planner 修订后的 Tasks/Plan 固定 SHA/actual diff，能追到 Guard unknown、受影响 R/关闭门禁、证据接收/方向分支及内层继续边界即可核此规划缺口；Explorer 结果尚未出不妨碍先诚实登记 pending。若已有结果，必须按真实报告/修订引用，不伪造缺失结论。

严重度为 MEDIUM：当前 CODE/PG 尚未 START，缺口可局部补齐；不将所有内层工作定为不可行，不以 hypothetical 全局风险提升到安全红线结论。

## 5. 其他判断、Gate 与未验证

无其他确定 findings。统计/卡与批次唯一归属/六项 requires/S1.5 新增真实前置/九文件/局部与 PG 完整验收/负控与命中/失败 retain 均通过本轮规划检查。G0 登记时的“进行中”与本次用户报告的“completed/stopped”按时间来源区分，非第二项实施缺陷；只读准备完成不自动等于 G0 安全授权通过。

| 范围 | 本轮结论 |
|---|---|
| 固定 metadata/上游/九路径前像 | 匹配；末次检查无未知漂移 |
| 统计/归属/DAG/旧证据保留 | 通过上述静态/字节检查 |
| U/C/P 内层规划与数据准备合同 | 可承接；只读准备可继续，运行/代码授权另核 |
| root 一般构造/路径复用 | 按固定合同可承接，未实读/执行当前 sourceRoot，不外推实际验证 |
| 通用 Guard / A-ADM2-18 / 整卡关闭 | 受限；F001 未关闭，不给无条件整卡 Planning PASS |
| Planning Review 总交回 | CHANGES REQUIRED，仅 F001 的 metadata 承接/停止合同；原 Planner 修后复审 |
| Code/LOCALPG/Case 执行 | 未 START、NOT RUN，本报告不发信号 |
| 人的 accepted/checkpoint | pending，未代签 |

未验证：Guard 当前真实实现/调用链/authority、Explorer 尚未发布结果、新 Admin2 实现/四 top/leaf 的实际命中、真实 PG/错误/取消/负控/数据/资源、当前 root 代码正确性、Host/部署/E2E/容量。S1.6 已关闭及窗口释放按原卡与 Lead 交接消费，不重新验证其运行，不要求其新增 PG/E2E。身份35卡及 ADR 不属于本轮正确性重审对象。

## 6. 实际 Readset 与操作

- Tasks/Plan 当前与 before-image：全字节/hash/完整 actual diff；正文只审 S8.2 :1729–1783、S1.5 的增量/相关原字段、Plan 正式批次表 :51–80、相关 B9E/S1.5 交接增量与 B9F :123–131。全任务仅机器提取 ID、三态、类型、requires、批次归属，以及 identity/closed 卡和历史注记的字节保留；不以此声称全 feature 正文复审。
- metadata `before-manifest.json`：全文；九路径逐一存在性/全字节 SHA，上游五项 SHA。当前源码/测试正文未读，原已读 query/handler/policy 合同复用；sourceRoot 仅摘要不作为调用链验证。
- Dalton `prep-admin2.md`：实际输出完整前段 §1–7；长输出后补读完整 §8–9/:168–180，重点采用 §8 Guard unknown。准备输入不是新授权，不据此代发实施信号。
- 上游 design r20/stories r09/test-plan r16 与两份 Review：核固定 SHA，复用当前会话已实际完整/有界取得的相应正文，不重跑历史设计/测试规划评审。
- Role/Skill/适用 Rule：同版有效正文复用；未读下一 Workflow/实施方法、未新 Route、未读 Explorer 未来报告或跨身份恢复源码。

Reviewer 仅在既有 Temp 目录写自己的 `review-s82-r01-readset-hashes.json`，冻结两 metadata、五上游与九路径（含两不存在路径）；未改变其中任何文件。实际工具仅文本/字节/hash/diff/结构检查和只读 Git branch/HEAD。唯一仓库写入本报告；无测试/PG/DSN/SMTP/服务/Browser、下级委派、Git 写或清理。

**stopped：Planning R01 报告完成，交原 Ptolemy 修订 F001 后返回本 Reviewer；不代发 CODE 或 LOCALPG。**
