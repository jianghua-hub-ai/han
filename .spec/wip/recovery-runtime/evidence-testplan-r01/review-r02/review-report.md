# Recovery TestPlan R02 定点独立复审

**PASS（仅本次 F001 规划 oracle 修订范围）**。`RECOVERY-TP-R01-F001`：**CLOSED**；新增 finding 0，OPEN 0，无受限未决项。此结论不表示测试执行 PASS、实现完成、人的接纳或 Planner 已登记。

Reviewer：Kuhn / add-reviewer，runtime ID `01a1835a-8010-7a73-a8dd-3192719b6965`；Main 依用户 standing 与项目委派规则指定，ID 来源 Main 实际 spawn/resume。作者 Jason / add-test-engineer，Main 提供的真实 ID `01a18395-a83b-7512-b7d1-824b073ab907`。非作者独立复审，未替作者写修复或替人接受风险。

固定 `D:\AID\VSCode\Codex\Han`、main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；读取 ref 现场核实。objectType=design（测试规划）。Role/add-code-review 全文取得，相关 Rule/Profile 同版全文复用，8 项 hash 一致。未重做全量 R01、源码 Review 或 TaskPlan 扫描。

| 冻结 TP（路径前缀 `.add/project/knowledge/design-workspace/`） | R02 Review 前 SHA256 | R02 Review 后 SHA256 |
|---|---|---|
| identity-auth/test-plan.md r24 | `E6906617E51CC349B783896A3A1D828DA4AA8A03716D0C104E77E9AB40680CF6` | `E6906617E51CC349B783896A3A1D828DA4AA8A03716D0C104E77E9AB40680CF6` |
| article-management/test-plan.md r21 | `E39C980C041CAC0C0A5441A9DDC52E51024C75E1911A7EE22B5332EDE66B9EDE` | `E39C980C041CAC0C0A5441A9DDC52E51024C75E1911A7EE22B5332EDE66B9EDE` |

**Round 1：原 finding 与精确修订。** 全文读作者 5 行 diff、receipt.json、verification.json，读取原 R01 receipt 与 finding 最小修订/验证要求。作者 article 前像 hash 精确为 R01 的 `338F35D12D002B96536DD8017949C326CE49EE7C851883E179815509511A794D`；独立全文件逐行/字节比较只有第786行不同（+1/-1），逆换该行逐字节恢复 R01 前像。该行风险/前置/动作/期望列原字节保留，仅证据 oracle 列改变。header、版本、元数据、其他场景均未变；identity 当前等于 R01 冻结 hash。

**Round 2：关联新风险。** 定点读 article TP 784–788 与稳定 article design 2468–2470（design SHA `2223BE4D005753FBF8D8AE0116F6926A7E8C044482BCBB0FADAB811677066350`），沿 R01 已核的 Journal/Anchor 端口解释，不重新读取源码。第786行明确：

- DB 精确 lookup、Journal snapshot、Anchor/source 读取、Store/双 namespace 前后 revision 分别正向记账及断言；不再禁止必需纯读 I/O。
- Refresh、ReplayAll、Consume、Fail、Publish、Anchor CAS/写入及 repair 等禁止动作分别为0；未给正常读取修复权。
- 历史 head 合法前进仍须满足原 exact closure 绑定、可信前缀及当前读取期间 revision 稳定；读间 revision 漂移仍 closed。未把稳定 token 单独改成授权，也未以历史证明替代首次开流 fresh Facts。

因此原“正确核验必须读外部端口却要求外部调用全零”的矛盾已消除。相邻 publication、legacy 与安装场景未改变，所有权、article 私表边界和原错误关闭规则未放宽。没有新增待决定或非阻断建议。

**证据与限制。** 独立运行的只是只读文本/hash/精确差异和前像还原断言、固定 ref 核对；命令为 PowerShell Get-Content/rg 定位与 `python check.py`、`python finalize.py`，脚本及 readset 留在本独占 Temp。作者 verification 仅作作者归属，不代替本次断言。结束再核双 TP、已读依赖及原 R01 证据无漂移；详细输入 hash/读取模式见 readset.jsonl，校验见 static-verification.json。字节/hash 检查不冒称全文语义复审。

**NOT RUN**：Go/构建/测试、PG/DDL、HTTP/SMTP、服务/worker、真实恢复、网络/安装、PlantUML 渲染及任何环境/人确认。未读 secret/ENV/DSN/key，未读 live TaskPlan，未写共享文件、Git、删除/清理/移动/归档或子委派。R01 原文/失败链保留；design draft、人 unknown、S4.5/S4.6 与 Case18 等原门槛均不变，既有架构/设计关闭不泛化为实现或测试通过。

**STOP/frozen**。本报告与 receipt 交 Main 现场技术接收，后续动作不由 Reviewer 代签。
