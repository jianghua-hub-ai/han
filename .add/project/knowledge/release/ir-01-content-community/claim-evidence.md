# IR-01 主张证据表

> 每条对外/对内表述必须能追到交付与验证来源。哈希为 2026-11-05 现值；引用时请复算。

| # | 主张 | 类型 | 证据 | 现值 |
|---|------|------|------|------|
| 1 | 首期 86 张任务全部关闭（其中 10 张为范围决策关闭） | 交付 | `.add/project/knowledge/tasks/tasks.md` | `FC111C15…` / 992,922 B |
| 2 | contract suite 普通与 `-race` 各 286 run / 287 pass / 0 fail / 0 skip | 验证 | `evidence-s1.5/contract-suite-r01/index.json` | `5477B1F5…` / 5,418 B |
| 3 | 独立复跑 `go test -tags=contract ./...` exit 0（28.514s） | 验证 | `wholecard-review-r02/verdict.md` | `A5E10329…` / 9,468 B |
| 4 | `go vet -tags=contract` exit 0（2.218s）、`gofmt -l cmd internal test` 0 行 | 验证 | 同上 | `A5E10329…` / 9,468 B |
| 5 | 16 端点精确 method/path、405 `Allow`、404/405 先于安全链 | 交付+验证 | `root-16-endpoint-r01/index.json`（10 artifacts 实算一致） | `A4155C64…` / 24,714 B |
| 6 | `S15-RE-F008` 判 CLOSED（34 叶子、0 fail / 0 skip） | 验证 | `f008-review-r02/report.md`、`verdict.md` | `B715476E…` / 11,030 B、`1749A628…` / 4,533 B |
| 7 | F008 证据包 9 个对象哈希与实物一致 | 验证 | `f008-allow-assertions-r02/index.json` | `410A65AD…` / 5,564 B |
| 8 | S1.5 整卡复审 R02 PASS FOR LEAD GATE、Lead 同 revision Gate 通过 | 验证 | `wholecard-review-r02/report.md` | `E2ED8CA0…` / 20,822 B |
| 9 | S1.5 于 2026-11-05 关闭，按 ADR-0028 登记 AI 代理接收 | 交付 | `acceptance-s1.5-r01.md` | `67A5CF3A…` / 3,090 B |
| 10 | 部署后 E2E、observability 演练不纳入本版 | 范围决策 | ADR-0036（`decisions/framework.md`） | `CE0C7F77…` / 395,794 B |
| 11 | 生产恢复实现链六卡不纳入本版，入口保持 fail-closed | 范围决策 | ADR-0037（同上） | `CE0C7F77…` / 395,794 B |
| 12 | 范围决策索引已同步（全局/领域/场景/产出归属） | 记录 | `decisions/README.md` | `58B4645A…` / 29,611 B |
| 13 | S1.5 子门禁与未验证边界汇总 | 记录 | `verification-s1.5.md` | `39529A54…` / 52,346 B |

## 使用规则

- 对外只引用第 2、3、5、6、8 类可复算结果；第 10、11 条必须同时说明是**范围决策**，不是验证通过。
- 引用数字时必须给出证据路径与哈希；哈希不符即视为过期主张，需重新取证。
- 未出现在本表的能力，不得写入发布材料。

## 明确不发布的主张

- 生产就绪、灾备就绪、高可用、跨平台一致。
- 任何性能/容量数字（S2.2/S2.4 未测，ADR-0033）。
- 真实部署 E2E、真实 observability、真实 Journal/HMAC/keyring、旧备份恢复、Case18 三状态。
- 真实用户量、节省比例、客户名称。

