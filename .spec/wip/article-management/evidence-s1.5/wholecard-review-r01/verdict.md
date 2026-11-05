# S1.5 整卡关闭评估复审 r01 — Verdict

- 对象：`article-management-S1.5`（唯一活跃卡）
- 固定工作区：`D:\AID\VSCode\Codex\Han`；`main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 评审性质：非作者 AI 代理独立 Reviewer；整卡关闭评估复审，不做逐行代码评审
- 评审日期：2026-11-05（Asia/Shanghai）

## 1. 总结论

**CHANGES REQUIRED**

不是代码缺陷结论：16 端点、三投影、公共错误、S1–S16、Ready20/Unavailable6、适用真实层与 286/287 命中的实质证据总体成立；阻断点是 `S15-RE-F008` 的“R02 独立复审 CLOSED”没有独立复审产物，唯一被引用的 review 文件明确写 R01 `NOT CLOSED`。因此关闭条件第 1、9 项不能判 PASS，第 10 项“子集 PASS 相加 = 整卡 PASS”不成立。

最小修复（不需要新增实现或测试）：
1. 产出 `review-code-s1.5-f008-producer-r02.md`（或等价 report + verdict），记录非作者 Reviewer actualID、时间、命令/exit/耗时、34 leaves、0 fail/0 skip、结论 CLOSED、SHA256/字节；
2. 更正 `verification-s1.5.md:147`、`verification-s1.5-gap-r01.md:28`、`tasks.md:1249` 的指针到该 R02 产物；
3. 若无法产出，则把 `S15-RE-F008` 保持 OPEN，撤回三处 CLOSED 断言。

补齐后由 Lead 做同 revision Gate、登记接收，再进入关闭评估。

## 2. 10 项逐条判定

| # | 关闭条件 | 判定 | 依据 |
|---:|---|---|---|
| 1 | 16 endpoint 真实 method/path 与适用安全链（Lorentz PASS WITH MINOR；`S15-RE-F008` 复审 CLOSED） | ⚠️ | 16 个 method/path 与 `main.go:3604-3658` 分支实读成立；Lorentz 21,470 B / `DA2BD8D7…` PASS WITH MINOR 成立；F008 代码修复与 34 叶子日志成立；但“R02 CLOSED”没有独立产物，唯一 review 文件是 R01 `NOT CLOSED`（F001）。 |
| 2 | 三投影 allowlist | ✅ | `projection_contract_test.go` 断言成员 list 5 键、detail 8 键、作者 4 键、Admin 12 键；contract suite 286/287。 |
| 3 | 公共错误（REST contract top） | ✅ | 19 个公共错误 case 与 design §3.3.1 `:1060-1080` 的 19 行对应；REST top PASS。 |
| 4 | S1–S16 内部契约（internal ports contract top；四文件组件 R02） | ✅ | 8 个内部端口组 + test-plan §4.7.1 `:552-571` 的 16 接缝映射；组件 R02 10,229 B / `52298FED…` PASS。 |
| 5 | Ready20 / Unavailable6（producer contract + 真实层） | ✅ | producer contract 断言 Ready 20 键、Unavailable 6 键；真实层双轮 58/0/0，独立评审 PASS WITH MINOR + Lead 接收。真实层主 leaf 为 Publish/ReadCurrent，ReadStatus 未单列真实 leaf（边界已登记）。 |
| 6 | 适用真实层证据（ACL R03、真实 Audit SQL、模拟收口、producer 真实层） | ✅ | ACL index 22,037 B / `8ED9FF3E…` + Lead Gate；Audit review 19,125 B / `DF4D4829…` PASS；sim index 12,332 B / `3E737B8C…` PASS WITH MINOR；producer review `B51E2F55…` PASS WITH MINOR + Lead 接收。 |
| 7 | Admin2 已落实（S8.2 已关闭；Case18 F001–F007 全关，F002 按 ADR-0034、F007 按 ADR-0035 为范围决策） | ✅ | `tasks.md:1925` 为 `[x]`，`:1926` 记录 2026-11-05 关闭；Copernicus F003/F004 report 24,267 B / `74D4054F…`、verdict 8,900 B / `63F2B1DA…`；ADR-0034 `content.md:241-276`、ADR-0035 `:279-300` 均“已接受”。`verification-s1.5.md:149` 仍写 S8.2 `[-]`，是历史未同步行（F002），不影响 tasks 的权威 `[x]`。 |
| 8 | 普通/race 命中（各 286 run / 287 pass / 0 fail / 0 skip） | ✅ | 离线解析两份 contract JSON 日志，均 run=286/pass=287/fail=0/skip=0；四文件哈希与 index 一致；独立 `go test -tags=contract ./... -count=1` exit 0（29.514s）。 |
| 9 | 独立非作者 Review / 必要复审覆盖各子集 | ⚠️ | Lorentz、Erdos、Confucius、模拟收口 Reviewer、Halley/Carver、Copernicus 均有产物；F008 的 R02 复审只有 tasks/verification/gap 文档中的叙述，没有独立 review 文件。 |
| 10 | 是否存在“子集 PASS 相加 ≠ 整卡 PASS”的遗漏项 | ❌ | 存在：gap-r01:48 称“只差整卡级复审 + 接收登记 + Lead Gate”，但 F008 的 R02 独立复审本身也是缺口；该缺口未计入。 |

## 3. 12 个 requires 实核

逐行核对 `tasks.md` 的 checkbox：

| # | requires | 行 | 实际 |
|---:|---|---:|---|
| 1 | `project-bootstrap-B0.1` | 63 | `[x]` |
| 2 | `identity-auth-S1.5` | 259 | `[x]` |
| 3 | `article-management-S1.3` | 1201 | `[x]` |
| 4 | `article-management-S2.1` | 1364 | `[x]` |
| 5 | `article-management-S3.1` | 1495 | `[x]` |
| 6 | `article-management-S4.2` | 1542 | `[x]` |
| 7 | `article-management-S5.1` | 1739 | `[x]` |
| 8 | `article-management-S6.2` | 1783 | `[x]` |
| 9 | `article-management-S7.1` | 1826 | `[x]` |
| 10 | `article-management-S8.1` | 1895 | `[x]` |
| 11 | `article-management-S1.6` | 1342 | `[x]` |
| 12 | `article-management-S8.2` | 1925 | `[x]`（完成@2026-11-05） |

**实核：12/12 均为 `[x]`，与 gap 文档一致。**

## 4. 实际命令与 exit/耗时

| 命令 | exit | 耗时 | 结果 |
|---|---:|---:|---|
| `git rev-parse HEAD` / `git branch --show-current` | 0 | — | `6da8abb…` / `main` |
| `git status --short` | 0 | — | 既有大量 modified/untracked；本次未写入 |
| `Get-FileHash` 三份输入 + 关键证据 | 0 | — | 三份输入与题面一致；关键证据见 report §5 |
| 离线解析 contract 日志 | 0 | — | 各 286 run / 287 pass / 0 fail / 0 skip |
| 离线解析 F008 日志 | 0 | — | 35/36 pass（1/2 top + 34 leaves），0 fail / 0 skip |
| `gofmt -l .` | 2 | 2,094 ms | 25 行，均为 `.spec/wip/**` 证据快照/probe；非源码格式失败 |
| `gofmt -l cmd internal test` | 0 | 262 ms | 0 行 |
| `go vet -tags=contract ./...` | 0 | 11,256 ms | 无输出 |
| `go test -tags=contract ./... -count=1` | 0 | 29,514 ms | 所有包 `ok`（含 `han/test/contract/article 0.458s`） |

**NOT RUN**：真实 PostgreSQL/ACL/Audit/producer 复跑；`go test -race` 全仓；`go build ./...`；真实部署 E2E；真实 Journal/HMAC/keyring；真实旧备份恢复；`psql`/数据库连接；任何 Git 写。

## 5. 保留限制与为什么

1. **真实 Journal/HMAC/keyring、旧备份恢复**：S4.x/恢复主线责任；本卡不以此为关闭前置。未验证不等于通过；不影响本卡的 16 端点/投影/错误/内部契约结论，但不得外推为恢复安全或 production readiness。
2. **真实部署 E2E/双系统**：S1.4 与 Case18 第 5 项；ADR-0035 已按测试项目“经决策不验证”，只豁免该范围，不外扩到 S1.5。
3. **真实 keyring**：与 Journal/HMAC 同属恢复信任根边界；本卡消费的是 ACL/Audit/producer 适用真实层，不声称 keyring 已真实部署验证。
4. **Ready20/Unavailable6 的真实层**：producer 真实层主 leaf 为 Publish/ReadCurrent；ReadStatus 在真实 integration 测试中被调用但未单列 Ready20/Unavailable6 leaf；不要把组件契约与真实 PG 独立验证混为一谈。
5. **F008 代码修复**：本复审独立读了 `main_test.go:3402-3406` 并解析 34 叶子日志，认为实质修复成立；但本复审不是项目要求的 R02 复审产物，不能代替 F001 所要求的独立 review 文件。
6. **人的业务接收 / Lead Gate**：本次没有人的业务签收，也没有 Lead 同 revision Gate；本复审只回答“是否可进入关闭评估”，不代替 Lead Gate 或人的接收。
7. **revision 语义**：`HEAD 6da8abb` 的提交内容不含 Go 代码；`cmd/`、`internal/`、`test/` 为未跟踪/已修改。所有结论是工作树快照语义，不能由 `git checkout` 重建。
8. **仓库级 gofmt**：`gofmt -l .` 被 `.spec/wip/**` 中证据快照的 `.go` 文件打断；`cmd internal test` 的源码格式干净。

## 6. 声明

**AI 代理评审，非真人签字；不代替 Lead Gate 或人的业务接收。**

本 verdict 仅回答 S1.5 当前证据是否足以进入关闭评估；不关闭 S1.5，不改 `[-]`、12 requires、写集、Plan/ADR 或任何既有证据。补齐 F001 的 R02 独立复审产物并更正指针后，可由 Lead 做同 revision Gate、登记接收，再进入关闭评估。