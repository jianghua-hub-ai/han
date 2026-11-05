# identity design r52 有界复审 R02

**PASS（仅原 finding 修复及相关新风险的设计合同）**。`RECOVERY-DES-R51-R01-F001` **CLOSED**；新增 finding **0**，OPEN/受限未决项 **0**。r51未变化的已审范围沿用，本次未全量重审特性或实现。

该结论可交Main现场技术接收；不关闭 `RECOVERY-B13A-CR-R01-F001`，不复评Code F002–4，不代代码/测试/人签字或源码START。设计仍draft，人的精确接纳unknown，原R01失败链原样保留。

Reviewer：Kuhn / add-reviewer，runtime ID `01a1835a-8010-7a73-a8dd-3192719b6965`；由Main依用户standing与项目委派规则指定，ID来源Main实际spawn/resume。Banach为原设计及修复作者。本人Role/add-code-review/相关Rules同版完整正文复用，非代码对象不强跑Go。

固定 `D:\AID\VSCode\Codex\Han` / main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。唯一被审共享目标 `.add/project/knowledge/design-workspace/identity-auth/design.md`：

- r51精确前像：`F87E1EB9073F4B2A723A098776762F759332B8086384BC0ED3C79DC49F624564`。
- r52 Review前：`5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`。
- r52 Review后：`5173FF8830142CA7E699AEB7A49945E8D6694C67372A1F3F2D557FF3B119ACAB`。

**Round 1：精确返工。** 全文读canonical `design-r52-author-r02/receipt.md`及完整r51→r52 diff（SHA `0321AF22C13619E9CF707D61BEF9DFF1671D582D5E6DB033D20A6676690387AE`），对照原R01 finding与复审要求。独立逐hunk证明10 hunks、+22/-5，正向重建r52、逆向逐字节恢复r51。原BOM及169个CRLF保持；Go/PlantUML块仅一个Go块增加采用的VerifyBeginSecret方法，其余块不变。三个新增相对链接/Task显式锚有效。header为r52/status draft，历史r51失败与原确认文字保留，未把作者修复标成独立CLOSED/PASS。

**Round 2：原缺口与相关风险。** 定点读r52 `:1536–1543`、`:1654–1660`、`:3594`及diff中的惰性Source/验证建议/owner映射；复用原R01已核上下文，并只读取旧冻结 `author-r01/journal-before.frozen:376–391,463–481` 对照既有Begin key/fingerprint和VerifyAgainst语义。没有读取/hash任何当前B13A五源，也没有新增当前源码/DDL读集。

| 原finding要求 | r52实际合同与结论 |
|---|---|
| 可调用的持钥只读执行者 | `ContextRecoveryJournal.VerifyBeginSecret(ctx,RunBinding,[]byte)(JournalEntry,error)` 明确由S4.8已定pure reader/完整retained keyring实现；复用原fingerprintSecret/HMAC及constant-time比较。Root只读候选，Runtime/Root不复制crypto或取得raw key。接口/owner/成功条件已闭合。 |
| 用旧Begin K1，不能用新active K2 | 从完整已验committed链/Anchor找唯一精确run+backup begun，按其keyID/version取retained K1，verify-only也可；不latest/active替换。核完整链、namespace/id/epoch、规范时间、前后head/不复用revision及reader/keyring绑定；仅全部成立返回既有Begin,nil，不授阶段权。 |
| 验证完成才写信任根 | Resume先独立证明仍未准备且许可/ctx/独占有效→lazy候选读取→Verifier→Runtime再与独立Begin/snapshot、Store.started_at及当前来源/许可/代际/ctx/独占/仍需准备精确核对→PrepareRecoveryTrust。ENV读取成功、Store字段互比及旧VerifyAgainst成功均不能越过此关。 |
| 异常、零写与secret生命周期 | 错secret/绑定、缺key/完整来源、未知版本、漂移/ABA、ctx/I/O原因分别映射既有Admission错误并保留errors.Is；全路径零新Begin/CAS/repair/mkdir/持久秘密/信任根写，不异步保留buffer、不输出raw secret/key。错误或失权时Runtime零PrepareTrust并保持关闭/真实join/清buffer。已准备/已改密零实际读取、验证及重置。 |
| 既有owner与验证归属 | S4.7只声明/调用和explicit fixture，S4.8在原11文件合同内实现pure reader/retained-key验证，S4.16只lazy候选读取；无新DTO/schema/authority/额外文件或requires。Planner后续精确同步S4.8/10/16是明确交接，不冒称现卡已逐字具备，不在本轮代写Task/Plan。 |

正确K1转verify-only/K2 active的正控，以及错secret/缺K1/错version/run/backup/来源ABA/取消失锁等负控，已作为本 finding 最低验证建议列明。此处证明的是可实施合同及验证落点，尚未执行这些软件验证。未发现修复引入新风险，无新增待决定或非阻断建议。

实际运行仅只读文本/hash、精确diff正反向重建、BOM/CRLF/代码块/链接/固定ref/结束漂移核验；脚本及实际路径/范围在readset和static-verification。Main全文/全量line proof与作者保护集分别按其归属，不冒Reviewer重扫；原R01现役五个直接消费者定义不重读。静态配对不作syntax/render/compile PASS。

**NOT RUN**：Go/build/test/vet/race、PG/DDL、HTTP/SMTP/服务/worker/restore、真实env/provider/ACL/tool/备份/target/retirement、网络/安装、PlantUML语法/渲染、性能/部署/E2E或人确认。无secret ENV/DSN/key读取、Git写、共享写、清理删除移档或子委派。原S4.2/S10.5关闭、两S4.4/Case18及S4.5/6真实门槛不外推。

**STOP/frozen**。交Main本报告/receipt和范围证据现场核收；原R01报告/前像/失败证据均不覆盖。后续Code F001修复及独立代码复审由Main另行授权，不由本设计PASS替代。
