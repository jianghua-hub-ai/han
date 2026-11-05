# C18-COV-F005 r03 固定上下文快照：原 Reviewer delta 复审

**deltaGate：PASS。F005：CLOSED（仅 r03 当前消费绑定）。新增 findings：0。**

本报告是同一非作者 Reviewer 会话的独立两轮 delta 复审，AI 角色评审，非真人签字。沿用原 F005 finding 级范围，交回 Lead 现场复核后登记新消费指针；不改旧 r02 报告、结论或 snapshot，不代行 Lead 接收。Carver / add-reviewer 的 actualID `01a1a491-533c-7123-a0cb-b0d2e955a6dc` 来源为已读取的 Lead reception 记录；本轮未另派 Agent 或下级委派。

## 固定对象与依据

工作区 `D:/AID/VSCode/Codex/Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`，开工与两次检查一致。用户确认 Singer / add-steward（ID `01a1a489-0aad-7390-8c82-d543b5154e22`）已 STOP，并给定以下五份摘要；本 Reviewer 实算全部相同。

| r03 文件 | 字节 | SHA256 |
| --- | ---: | --- |
| index.json | 19180 | 4B3F9979D749279D0EA8DDF0A0BC5A27F9DD32EF10B6E3033586445BFF3738CB |
| report.md | 5553 | D9FE76FC7403BECD33F667771190B3479809E7D3874269E2F8A1EBDC074B0A7F |
| hash-manifest.json | 11293 | AB98DBCE882CC6A2C1B445A69FBD7808111878E4390042621A2297C73255D0DD |
| verification-context.snapshot.md | 45600 | 7997E1ADF37D82FB663B40B41BD43BE4AF191120E66A60BC7E6C1C25CA1F4AF9 |
| check.py | 3603 | 8E2FD8B23B6C3FA9D569BF3E1B509C9A75AEBC9B3CE085E163DE2CB8B55F9504 |

本次对象是非代码证据消费契约的修订及随包 Python 校验器，只审活文档绑定替换、当前可消费性和旧证据边界，不展开全量代码/Case18 评审。复用本会话已完整读取的 Reviewer Role、add-code-review Skill、code-review-checklist 与 abstraction-principles；没有加载无关 profile 或未来方法。

上游依据为[原 F005](../../evidence-case18-coverage-r01/report.md)第 193–198 行允许的 superseded 指针修复方向，及[原 r02 独立报告](../producer-real-current-r02-review-r01/report.md)、[原判定](../producer-real-current-r02-review-r01/verdict.md)。原判定只表达其核验时点事实。本次已复现后续登记追加造成的唯一 r02 失配，不回写旧报告，不把 r02 的旧门禁当成 r03 已过。Lead 后续接收记录在 [r02-lead-r01.md](../producer-real-current-r02-lead-r01.md)；其引用的其他范围核证不冒充本 Reviewer 的新执行。

## 两轮实际检查

第一轮：全文读取 r03 index/report/manifest/check.py，按给定五份摘要固定对象；实际核对 manifest 全部条目的 hash/bytes、路径唯一性及精确集合。把 r03 bindings 与 r02 比较：30 项原绑定逐字段不变；verification_context 由活文档改成本目录 snapshot；另新增六个历史审计指针（r02 index/report/manifest、旧独立 report/verdict、Lead reception）。十源观察逐项与 r02 相同。读取 snapshot 新增 3161 B 的接收登记，旧前 42439 B 用原哈希核验；读取活 verification 仅为本轮来源一致性核对，不将其作为 r03 的消费前提。

第二轮：从实际消费者出发，检查 check.py 的路径解析、直接集合消费、原目录导航与旧 manifest 不递归语义。独立核新报告 11 个链接及 snapshot 按原目录解析的 59 个导航，检查旧 r02 现在的唯一失配及原评审全部六文件的保留。执行随包 checker（从非仓库根目录运行也通过），再以独立脚本做内存追加、拒绝活文档内容读取、固定快照同长度损坏三个实验；最后重复绑定检查以核收工稳定性。

两份逐项记录为 [checks-r01.json](checks-r01.json)、[checks-r02.json](checks-r02.json)，可复核脚本为 [checks.py](checks.py)，实际命令及结果见 [commands.md](commands.md)。没有执行 Go 或连接 PostgreSQL。

## 实际结果与 delta 解释

| 项目 | 实际结果 | 适用边界 |
| --- | --- | --- |
| manifest | 50/50 hash/bytes 一致，路径无重复 | 仅本 r03 声明的直接固定集合 |
| 集合完整性 | 37 bindings + 10 源观察 + 本包 index/report/check.py = 50 | manifest 本身由给定独立摘要固定，无自哈希循环 |
| 继承项 | 30 个旧证据绑定逐字段不变，十源观察完全相同 | 不重新扩审产品代码或旧运行 |
| snapshot | 45600 B，当前与活来源逐字节一致；前 42439 B 哈希为原 1C7C0E… | 捕获时间来自作者元数据，非本 Reviewer 在场见证；副本字节及来源一致性已实核 |
| 旧 r02 | 当前 42/43；唯一失配是活 verification 整文件 | 原 43/43 是此前时点事实，旧材料未改 |
| 旧独立评审 | 六文件 hash 均等于此前交付值 | report/verdict/checks/commands 全部保留 |
| 新报告链接 | 11/11 可达 | 包含一个活 verification 导航，不读取该导航内容作硬校验 |
| snapshot 导航 | 59/59 可达 | 按 index 指明的来源目录 `.spec/wip/article-management` 解析，去编辑器行号后缀，目录链接允许目录；只核导航存在 |

独立实验均只在内存改变读取行为，没有改活 verification、snapshot 或其他文件：

1. **活 verification 追加模拟：通过。** 为该路径准备不同长度/不同哈希的虚拟追加内容后重新调用实际 checker，完整结果与基线一致。消费路径对活 verification 的内容读取次数为 0；即使返回虚拟内容的读取钩子已设置，也不会触发。
2. **拒绝活 verification 内容读取：通过。** 将该路径的文本/字节读取设为直接抛错，实际 checker 仍通过；同时观察 Tasks/Plan 也无内容读取。该实验核的是内容硬依赖隔离；导航存在检查仍保留，不声称文件删除或链接移动也一定通过。
3. **固定 snapshot 同长度损坏：预期拒绝。** 仅在读取返回值中翻转 snapshot 首字节，actual checker 在该 snapshot 的 SHA256 检查处报 `hash mismatch`。证明快照仍是必须准确匹配的固定物证，修复没有通过跳过上下文校验来制造通过。

新 [index.json](../producer-real-current-r03/index.json) 第 44–69 行明确 snapshot 来源、whole_file_raw_bytes、原目录导航及 source_navigation_only；第 250–254 行 verification_context 绑定的是包内 snapshot。第 256–273 行明确 r03 当前入口及旧 r02 manifest 的历史、不递归语义，第 368–381 行定义 navigation_only 与直接消费契约。新 [report.md](../producer-real-current-r03/report.md) 第 18、22–24、34–38 行与此一致；[check.py](../producer-real-current-r03/check.py) 的实际读取路径也符合该合同。

活文档在 source_at_capture 里出现路径、哈希和字节，是记录捕获来源，不是要求以后活文档继续等于该摘要。历史 r02 manifest 内仍有旧 verification 硬绑定，但 r03 只核它这个历史文件的字节，不递归执行其旧契约。snapshot 内的旧 r02 入口文字同样属于原文历史上下文，当前消费入口由 r03 index 明确定义。没有发现因这些历史字段或导航再次把活文档内容隐性纳入 r03 硬依赖。

## Findings、门禁与后续条件

新增 findings：CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0。待决定项：无。非阻断整改建议：无。

**C18-COV-F005（原 MEDIUM）：CLOSED，限定于 r03 当前消费绑定；本修订 deltaGate：PASS。** 原问题的触发是合法追加 verification 后，整文件哈希变化使旧消费集合失配。新修订将该依赖替换为本目录固定原始字节 snapshot，明确区分导航和固定物证；50/50、真实 checker 以及追加隔离/损坏拒绝实验共同支持关闭该 delta。当前入口应由 Lead 登记为 r03 index + manifest 及其固定快照，不更新 snapshot 本身，也不改旧 r02 或原评审结论。

旧 producer index 仍为 F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B / 19567 B；current report 为 1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749 / 8267 B。旧 A9A4ECBB…/8023 B 原像及完整逐字 diff 继续 unknown。历史普通/race 各 58/0/0 仍仅沿用已保存运行与原 Reviewer 离线复算；本次仅核其绑定 hash/bytes 未变，未重新计数或运行。

十源仍为 9/10 历史匹配：deletion_replay_source_test.go 当前 5C63E4DF…/15473 B，相对原 2F8E38D8…/15341 B 漂移，原因、影响与当前运行 unknown。没有当前源码 PASS 或全 runtime 无漂移结论。

S1.5/S8.2 仍 active，blocked-upstream 不解除，Case18 仍阻塞且不具备关闭评估条件；F002/F003/F004 整体仍 OPEN。本轮不重判这些 finding，不证明真实 Journal/HMAC/keyring、旧备份恢复、完整恢复开流、应用 ACL、部署 E2E、容量/P95、production readiness 或人的接收。

复审条件：r03 五份文件、直接绑定证据或十源字节变动时，按新固定修订复核受影响范围；只向活 verification 追加而保持其导航路径存在，不要求改 snapshot 或重做固定集合。未知来源副本变动、导航迁移/删除与其他代码修订不属于此次追加隔离保证。所有结论仍是给定摘要修订及本次检查时点的独立判断。

本轮唯一写集是新建 `producer-real-current-r03-review-r01/` 内六文件，目录创建前两次核不存在；被审对象、旧证据、源码、Tasks、Plan、verification 均未写入。无 Go/PG、凭据读取、外网、Git 写、清理删除移动或下级委派；不扩读并行 S2.4/性能 addendum。交回 Lead 现场核固定摘要及本独立结果，再登记新消费指针。**STOP。**
