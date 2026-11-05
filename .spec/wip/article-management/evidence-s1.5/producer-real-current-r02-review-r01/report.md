# C18-COV-F005 当前消费绑定修复：独立有界复审 r01

**F005：CLOSED（仅本 finding 的当前消费绑定缺口）；review gate：PASS（仅此范围）。新增 findings：0。**

AI 独立 Reviewer 角色评审，非作者自审，非真人签字。只交回 Lead 现场复核后登记；本报告未修改原 finding、Tasks、Plan 或 verification 的状态。本次不是全量 Case18、代码、当前源码运行或 E2E 评审。

## 对象、依据与授权

- 固定工作区：`D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；开工及两次离线检查均一致。大量源码/证据处于未跟踪或已修改状态，版本以用户给定 SHA256/字节固定，不能只靠 HEAD 重建。
- 被审作者：Singer / add-steward，用户提供实际 ID `01a1a489-0aad-7390-8c82-d543b5154e22`，作者已 STOP。当前评审会话不是作者或原始运行者；本端未获当前 Reviewer 的 Host actualID，不编造 ID、姓名或另一次派发。本轮没有下级委派。
- 对象类型：非代码证据消费契约/索引文档复审；按 add-code-review 的非代码对象要求核绑定、版本语义、历史迁移、消费者及验证边界，不强套产品设计格式、Go 构建或测试门禁。不涉及实现改动，未加载 go-service 或前端 profile。
- 方法：实际完整读取自己的 `.codex/agents/add-reviewer.toml`、`add-code-review/SKILL.md`、`add-route-work/SKILL.md`、`code-review-checklist.md`、`abstraction-principles.md`。这是既有 F005 修复交接的独立专业复审，只进入本环节。
- 上游验收依据：[原报告 F005](../../evidence-case18-coverage-r01/report.md)，原文件第 193–198 行：旧 report 声明与实物不一致，目录不能作为整体引用基线；允许证据 owner 修正声明或补 superseded 指针，属文档纠正，不改变行为结论。已完整读取原报告及其真实上下文。
- 既有裁定：完整读取 [Case18 当前证据复审 r02 报告](../../evidence-s8.2/case18-current-r01-review-r02/report.md)及 [verdict](../../evidence-s8.2/case18-current-r01-review-r02/verdict.md)，特别是 report 第 96–108、128–139 行的 F005 OPEN 与其他状态边界；读取 [verification-s1.5](../../verification-s1.5.md) 第 146–157、161–186 行的 producer 接收、两项 LOW 处理、Case18 后续复审记录。旧裁定针对旧对象，本次对新增固定修订独立判断。
- 授权：用户 2026-11-04 的直接有界 Review 指令及其建议值+ADD继续授权；不据此代签人的成果接收。只新增本评审目录中的报告与必要 checks/commands；开工先核目录不存在，创建时再次拒绝覆盖。

被审三文件实算与用户给定值全部一致，位置均在 `evidence-s1.5/producer-real-current-r02/`：

| 文件 | 字节 | SHA256 |
| --- | ---: | --- |
| index.json | 21133 | E7DF162F0716558369DFA996DB7FE38FAD20E0AF340B4873685538C900A54E56 |
| report.md | 9666 | F8CC9692736A87C0B524E4C4FFA17F60D92C20E5798D83B4C862A40987CB9625 |
| hash-manifest.json | 10312 | 996EE23E23F48F4988DC583EF0C9A4F4CC85C11815BC8C608E3359EB56B901B9 |

## 两轮实际检查

第一轮：全文读取三份被审文件及原 finding 报告，核对 revision、作者/运行者/评审者归属、追加指针范围与 unknown；实际从新 index 读取 31 个 bindings，从 manifest 读取 43 项 SHA256/bytes 并消费其目标。核清 manifest 的集合恰为 31 个证据引用 + 10 个源码字节观察 + 本包 index/report；无重复、漏项、额外项，manifest 自身由用户给定摘要固定以避免循环。读取当前旧路径 report、早期 producer 复审、Halley 历史 report/verdict、verification 相关记录及最新 Case18 复审；没有把历史评审自动升级成本包评审。

第二轮：沿真实消费路径核 `report_binding`、`known_reason_refs`、`review_chain` 中所有 ref 的实际目标；解析本包报告 18 个相对链接，逐项检查路径位于仓库内、文件存在及其摘要绑定。用旧 index 自身的 `upstream_evidence` 重新枚举 11 项，比对新增索引所声明路径集合及实际字节。以旧 index 的十源集合核当前十源；从两份原始 test2json stdout 按真实父子测试关系重新提取终态叶子，逐项核保存 leaves、result、commands。再独立执行被审报告所提供的 PowerShell manifest 消费循环，43 项全匹配。复核原 F005 建议是否由当前修订满足，以及该结论是否被错误外推到源码、运行或其他 finding。

[checks-r01.json](checks-r01.json) 与 [checks-r02.json](checks-r02.json) 保存两次独立执行的逐项实算，均 errors=0；两次 43 项 manifest 输入及 manifest 本体的摘要一致。可复核脚本为 [checks.py](checks.py)，命令与额外 PowerShell 检查见 [commands.md](commands.md)。这里的“两轮”是先对象直接缺陷、后消费者/上游及语义复核，重复计算用于收工稳定性检查。

## 核验结果与证据边界

| 检查 | 实际结果 | 可得结论 |
| --- | --- | --- |
| 43 项 manifest | 43/43 hash 与 bytes 一致，路径唯一，集合精确 | 当前声明的完整引用集合可消费 |
| 31 项 bindings | 31/31 一致，report/review/known-reason refs 均可解析 | 来源与历史评审链可追溯 |
| 报告链接 | 18/18 存在且有绑定；manifest 由独立给定 hash 固定 | 人工消费入口有效 |
| 旧 index 的上游集合 | 11/11 匹配，新增索引复算路径集合一致 | 原 producer 来源指针保留 |
| 旧 index 本体 | F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B / 19567 B | 历史快照保留，未改成伪匹配 |
| 同一路径 current report | 1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749 / 8267 B | 新指针绑定的是当前实物 |
| 旧 report 声明 | A9A4ECBB9DEF6C1862ECE1D151F085F179DE43A6A489FD94411B017978150BFF / 8023 B | 仍是旧声明，原像 unknown；不是本次匹配目标 |
| 双轮原始 stdout | 各 58 pass / 0 fail / 0 skip，保存 leaves 的完整事件逐项相等，包终态 pass | 仅对保存日志的离线复算 |
| 命令/结束记录 | argv 逐元素一致，commands/result 字段一致，exit 0、wait_returned true；无 overlay，race 轮带 -race，无 DATA RACE | 历史执行元数据相互吻合；没有本轮运行 |
| 目标 Publish/ReadCurrent | 两个目标叶子在两轮原始日志中均 pass | 历史 producer 侧目标已保存，不外推 Case18 |
| 十源观察 | 9/10 与历史绑定一致，1/10 漂移；全部与新增索引当前观察一致 | 当前字节事实，没有当前源码 PASS |

漂移项仅为 `internal/article/deletion_replay_source_test.go`：历史 `2F8E38D81F4CC88E03E02AFC0602501ACC2241E40FC0270D94B270D756EA0695` / 15341 B，当前 `5C63E4DF765B21006809D2E6E7A991BF57709F714DD228C47C6B504427D613FE` / 15473 B。作者、原因、行为影响与当前运行结果仍 unknown；只做字节观察，未审查或修复源码。9/10 当前匹配不证明当前测试通过。

PG 18.6 / `127.0.0.1:51934` / `han_b13d_article` 在已绑定的 preflight stdout 中可核。preflight 的 test-only overlay 与 required 双轮无 overlay 的界线已正确保留；当前实例、环境及全 runtime 是否漂移未验证。历史 177 项保护清单/28 项漂移只按旧评审口径保留，本轮没有扩扫、重判。

旧报告多 244 B 的完整逐字 diff 不能从摘要或两项修改说明推出。新增 report 第 14、21、48–50 行与 index.unknowns 明确承认原像 unknown、不重建、不排除其他字节差异；因此没有用历史 Halley PASS 覆盖 current report 的全部新字节。当前 report 的“本轮”“现役”“10 源一致”、not_reviewed 与 F008 历史阶段文字在新增 report 第 16–19、36–38 行有明确时点/范围解释；F008 后续结论只引用 verification，不在本次重判。

## Findings 与 F005 判定

新增缺陷：CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0。没有待作者整改项；没有以风格偏好提出非阻断改动。

**原 finding C18-COV-F005（MEDIUM）：CLOSED，限定为新增 r02 提供的当前消费绑定。**

- 原违反依据与触发：`producer-real-r01/index.json` 的 artifacts 把现存 report 绑定到旧 A9A4ECBB/8023；消费者按旧 index 整体校验时发生摘要/字节不符，无法建立正确引用基线。原报告第 193–198 行建议“修正索引声明或补 superseded 指针”。
- 修复位置：新增 [index.json](../producer-real-current-r02/index.json) 第 36–53 行 `report_binding`、55–64 行 old/current bindings、339–476 行 source10/review_chain/unknowns/limits/local_artifacts；新增 [report.md](../producer-real-current-r02/report.md) 第 13–23、36–54 行；新增 [hash-manifest.json](../producer-real-current-r02/hash-manifest.json) 第 8–224 行完整集合。
- 证据：上述三文件与全部声明引用实算一致；report_binding 清楚区分历史声明、当前报告和仅面向当前消费的 supersedes 语义；ref/链接均实际可消费。旧 index 的错误声明仍按旧快照保留，无伪造 old report，无伪匹配。
- 影响及关闭理由：采用本 r02 index + manifest 的消费者能够取得一个准确、完整、有历史来源及明确未知边界的当前引用集合；这满足原 finding 明确允许的追加指针修复方向。源码漂移、旧原像 unknown、非完整 E2E 均未被包装成 PASS，也不阻止对现存证据作精确绑定。
- 接收建议：Lead 现场核三文件及本报告/检查结果后，按 r02 指针登记 F005 finding 级 CLOSED，并保留原失败记录。消费者必须从 r02 + manifest 进入；单独按旧 r01 index 校验当前目录仍会错配，本次没有宣布旧索引已自洽或授权改写它。

**Review gate：PASS — 仅 C18-COV-F005 当前消费绑定修复。** 这是真实非作者 AI 角色的两轮独立复审，不是原作者整改自证；不代替 Lead 登记、人的业务接收或下一动作授权。

## 未验证范围、复审条件与交回

S1.5 与 S8.2 仍 active（现场任务标题分别为 `tasks.md:1234`、`:1913`，行号随并行写入可漂移）；S8.2 blocked-upstream 不解除，Case18 仍不具备关闭评估条件。C18-COV-F002、F003 保持 OPEN，F004 整体保持 OPEN（Session 侧已有历史覆盖，AccessGeneral 侧缺口不由本包解决）。本报告不重新裁定 F001 或其他 finding。

未执行任何 Go 命令、测试、构建、vet 或服务；未连接 DB，未读取凭据、环境 DSN 或包外资源，未做外网、Git 写、删除、移动、归档、清理、下级委派。未审来源包全部 106 payload、完整 177 项保护清单、全仓代码或所有消费者；只对 31 个引用、十源字节、直接相关记录和双轮日志有界复核。真实 Journal/HMAC/keyring、旧备份恢复、真实全状态恢复隔离、开流、应用 ACL、部署 E2E、容量/P95、production readiness 及真人签字均未由本次证明。

Backend 的 S2.4 四 test/docs 与 TestEngineer 性能 addendum 属其他工作者写集，没有作为 F005 的失败条件，也未扩读其 diff。不能用 Git 总行数稳定代替固定对象摘要稳定；本次留存的两次 checks 核的是指定输入。Lead 所述 8 份旧证据保护结果作为上游说明；本 Reviewer 对本对象实际绑定的旧证据另做了实算，没有把 Lead 摘要冒充自己的全范围核验。

如三份被审文件、其绑定文件或原始日志发生变化，需明确新修订并复核受影响范围；如只发生后续源码漂移，应更新字节观察及适用边界，不能沿用当前“43 项匹配”的时点结论或迁移历史运行 PASS。需要关闭 F002/F003/F004 或 S1.5/S8.2/Case18 时，须另取对应真实证据与评审，不复用本门禁。

交回 Lead：本轮只有本目录新增独立报告与必要检查文件，未登记业务状态；由 Lead 现场复核后处理登记。**完成 STOP。**
