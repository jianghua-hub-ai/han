# S1.5 关闭条件 gap 对账 r01（2026-11-05，只读）

固定 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本文件只做只读对账，不改变 S1.5 的 `[-]`、写集、requires 或统计。

## 1. 十二 requires（逐项核 checkbox）

| # | requires | 状态 |
|---:|---|---|
| 1 | `project-bootstrap-B0.1` | `[x]` |
| 2 | `identity-auth-S1.5` | `[x]` |
| 3 | `article-management-S1.3` | `[x]` |
| 4 | `article-management-S2.1` | `[x]` |
| 5 | `article-management-S3.1` | `[x]` |
| 6 | `article-management-S4.2` | `[x]` |
| 7 | `article-management-S5.1` | `[x]` |
| 8 | `article-management-S6.2` | `[x]` |
| 9 | `article-management-S7.1` | `[x]` |
| 10 | `article-management-S8.1` | `[x]` |
| 11 | `article-management-S1.6` | `[x]` |
| 12 | `article-management-S8.2` | `[x]`（2026-11-05 关闭） |

**结论：12/12 已满足。**

## 2. 关闭条件逐条对账

| # | 关闭条件 | 现有证据 | 状态 |
|---:|---|---|---|
| 1 | 16 endpoint 真实 method/path 与适用安全链 | 子门禁 2：Lorentz 独立评审 PASS WITH MINOR（HIGH 0/MEDIUM 0/LOW 8，8 条已处置）；`S15-RE-F008` R01 判 NOT CLOSED（仅证据记账：叶子数/耗时/gofmt 日志），其 r02 更正包已于 2026-11-05 由非作者独立复审判 **CLOSED**（[report](evidence-s1.5/f008-review-r02/report.md) `B715476E…`/11030 B、[verdict](evidence-s1.5/f008-review-r02/verdict.md) `1749A628…`/4533 B；`-race` 本机未复跑） | ✅ |
| 2 | 三投影 allowlist | 子门禁 1：projection contract top，普通与 `-race` 各 286 run/287 pass/0 fail/0 skip | ✅ |
| 3 | 公共错误 | 子门禁 1：REST contract top（同上） | ✅ |
| 4 | S1–S16 内部契约 | 子门禁 1：internal ports contract top；子门禁 6 四文件组件 R02 PASS | ✅ |
| 5 | Ready20/Unavailable6 | 四 contract top 内含 `replay_status_producer_test.go` 的 Ready20/Unavailable6 覆盖；子门禁 7 真实 producer 双轮 58 pass/0 fail/0 skip | ✅ |
| 6 | 适用真实层证据 | 子门禁 3（应用角色 ACL，R03 PASS WITH MINOR + Lead Gate）、4（真实 Audit SQL，独立评审 PASS + Lead 接收）、5（模拟环境收口 PASS WITH MINOR）、7（producer 真实层，独立评审 PASS WITH MINOR + Lead 接收） | ✅ |
| 7 | Admin2 已落实 | `article-management-S8.2` 于 2026-11-05 关闭；Case18 F001～F007 全部关闭（F002/F007 为 ADR-0034/0035 范围决策） | ✅ |
| 8 | 普通/race 命中 | 子门禁 1 普通与 `-race` 各 286 run/287 pass，0 fail/0 skip | ✅ |
| 9 | 独立非作者 Review / 必要复审 | 子集级均有：Lorentz（16 端点）、Confucius（ACL R01/R03）、Erdos（Audit SQL）、Halley（producer）、S15-RE-F008 复审 R02、Copernicus（Case18 F003/F004） | ✅（子集级） |
| 10 | Lead 同 revision 的 diff/hash/范围/实际接收 | 各子门禁有 Lead 接收；**整卡级组合复审与同 revision Gate 尚未执行** | ⚠️ 待补 |

## 3. 仍未满足 / 待办

1. **整卡级非作者复审**：现有评审均为子集级（contract top、16 端点、ACL、Audit SQL、producer、Case18），缺一次覆盖「上述子门禁相加是否构成完整 S1.5」的整卡复审。
2. **接收登记**：缺人的业务接收，或按 ADR-0028 登记的 AI 代理接收（需记录授权来源、代理主体与范围）。
3. **边界项（不计入本卡关闭前置，但不得被外推）**：真实 Journal/HMAC/keyring、旧备份恢复属 S4.x 责任；真实部署 E2E 属 S1.4；ADR-0035 只豁免 Case18 的 E2E/双系统，不外扩到 S1.5。

## 4. 结论

- 12 个 requires 全部满足；关闭条件的 9/10 项已有可核证据。
- **S1.5 距关闭只差「整卡级复审 + 接收登记 + Lead Gate」**，不需要新增实现或测试。
- 建议：spawn 非作者 Reviewer 做整卡复审（对象为本文 §2 所列证据 + 当前 revision），通过后由 Lead 做同 revision Gate，并登记接收；若用户选择测试项目豁免，则按 ADR-0028 记录 AI 代理接收。

- **2026-11-05 进展**：整卡复审 R01 已执行（非作者 Aristotle，CHANGES REQUIRED；F001/F002/F003 已按复审建议处置），`S15-RE-F008` 的 r02 更正包已由非作者独立复审判 **CLOSED**（`evidence-s1.5/f008-review-r02/`）。据此 §2 第 1、9 项可判 PASS，剩余缺口回到第 10 项：整卡复审 R02 + 接收登记 + Lead 同 revision Gate。
