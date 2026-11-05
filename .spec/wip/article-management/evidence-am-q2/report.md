# AM-Q2 容量参数确认（r29）证据报告

## 1. 任务与产出

- 角色：`add-feature-designer`（特性设计）。
- 工作区：Han 单仓工作区；分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 设计产物：`.add/project/knowledge/design-workspace/article-management/design.md`，revision `article-management-design-ir01-r29`。
- 本报告：`.spec/wip/article-management/evidence-am-q2/report.md`。
- 未执行 git add/commit/push/branch/stash/checkout/restore/reset/clean；未修改代码、测试、ADR、`test-plan.md`、`tasks.md`、`requirements.md` 或其他设计文件。

## 2. 确认值来源

用户于 2026-11-02 明确确认以下首期 AM-Q2 输入，数值未改写：

| 参数 | 确认值 |
|---|---:|
| `HAN_ARTICLE_QUERY_DEFAULT_LIMIT` | `20` |
| `HAN_ARTICLE_QUERY_MAX_LIMIT` | `100` |
| `HAN_ARTICLE_QUERY_CANDIDATE_BATCH_SIZE` | `200` |
| 首期文章总量 | `100000` 篇 |
| 单用户文章上限 | `10000` 篇 |
| P95 目标 | 成员文章列表≤`300ms`；文章详情≤`150ms` |

这些值是 production 部署基线、容量规划输入和首期验收目标，不是代码默认、测试 fixture 或目标环境实测结论。retention、RPO/RTO 仍属 AM-Q6/AM-Q8，不属 AM-Q2 关闭范围。

## 3. 设计改动清单（最终行号）

| # | 落点 | 改动摘要 |
|---:|---|---|
| 1 | `design.md:5`、`design.md:7` | front matter 改为 `revision: article-management-design-ir01-r29`、`last_updated: 2026-11-02`。 |
| 2 | `design.md:37`、`design.md:39` | 页眉设计版本同步 r29，日期同步 2026-11-02。 |
| 3 | `design.md:190` | AM-Q2 行改为已关闭；写入 20/100/200、100000、10000、P95 300ms/150ms 与来源“用户 2026-11-02 明确确认”；保留“不再阻塞 S2.1 production wiring”，并区分部署基线可进入与目标环境容量/P95 未实测。 |
| 4 | `design.md:198` | 将“未产生生产分页数字”改为 r29 已产生生产分页数值 20/100/200；容量/吞吐实测、告警阈值、retention、RPO/RTO 仍 TBD。 |
| 5 | `design.md:310` | 候选批次由 TBD 改为 r29 部署基线 200，保留配置值必须大于 `MaxLimit`。 |
| 6 | `design.md:883`–`design.md:885` | 三键仍为“环境；必填、无默认”；补记 r29 生产部署基线 20/100/200，未写成代码默认，未新增回退，未改变严格解析与失败分类。 |
| 7 | `design.md:921` | 明确 `20 <= 100`、`200 > 100` 与确认值一致；保留“字段解析合法不等于完整配置合法”及 `cross_constraint` 语义。 |
| 8 | `design.md:1001` | 删除“具体生产数值仍为 TBD”，改为 r29 部署基线 20/100/200；保留禁止代码默认或测试 fixture 回退。 |
| 9 | `design.md:1089` | `limit` 参数表补记 r29 部署基线 20/100，无代码默认。 |
| 10 | `design.md:1252` | 索引说明中的容量目标改为引用 r29；目标环境执行计划实测仍 unknown / NOT RUN。 |
| 11 | `design.md:1937` | 查询读取说明中的“容量/P95 仍 TBD”改为“目标见 r29，目标环境实测仍 unknown / NOT RUN”。 |
| 12 | `design.md:2267` | 明确 worker 容量、吞吐和告警阈值仍 TBD，并指向 r29 的分页基线与首期规模目标；调度数字仍非 TBD。 |
| 13 | `design.md:2440`–`design.md:2461` | 新增 `##### r29 AM-Q2 容量参数确认`：确认值表、约束校验、首期适用边界、P95 边界、未确认项清单及“不声称容量验收已通过”。 |
| 14 | `design.md:2765` | 过载控制区分列表分页基线已确认与入口限流、连接池、finalizer 并发等容量目标仍 TBD。 |
| 15 | `design.md:2837` | STRIDE DoS 行同步 r29 分页值与 P95 首期目标；连接、容量、P95 实测、并发和数据分布仍 TBD。 |
| 16 | `design.md:2892` | 资源不足段同步分页 20/100/200；连接和 dispatcher 并发数值仍 TBD。 |
| 17 | `design.md:2962` | 可伸缩性段写入首期规模、P95 目标和部署基线；目标环境真实 P95、并发、数据分布、QPS、连接池、finalizer 并发与 SMTP 吞吐均为 unknown / NOT RUN。 |
| 18 | `design.md:3176` | §10 信息需求表将 QueryConfig 数值、文章规模与 P95 目标标为已关闭；目标环境实测仍 unknown / NOT RUN。 |
| 19 | `design.md:3260` | 附录数字自检同步 r29 数值与目标环境未实测边界。 |

## 4. 约束校验

- `DefaultLimit=20 <= MaxLimit=100`：通过。
- `CandidateBatchSize=200 > MaxLimit=100`：通过；保持严格大于，未改成 `>=`。
- 三键仍为环境必填、无默认；严格十进制正整数、signed 32-bit/int 范围、DB 前失败顺序、最小披露错误与无回退合同均未改变。
- 测试边界 fixture 未改：`1/1/2` 与 `1/2147483646/2147483647` 仍只用于测试；`2/2/3` 仍只作 fixture，均不是生产默认。

## 5. 未确认项

- 目标环境容量、吞吐、真实 P95、并发与数据分布。
- 告警阈值。
- retention、备份周期、RPO、RTO；属 AM-Q6/AM-Q8，不属 AM-Q2 关闭范围。
- 部署参数的实际环境注入值尚未在目标环境核对。

## 6. 未验证边界

- P95 数值是首期验收基线与开发机测量参考；测量主机不是目标部署环境，不能当作目标环境验收结论。
- 目标环境真实 P95、并发与数据分布均未测量，状态为 unknown / NOT RUN；本设计不声称已达标。
- 容量验收未执行；本设计不因数值确认而声称容量验收已通过。
- 未做真实部署、未做容量压测、未做独立评审、未实施。

## 7. 文件完整性核验

| 项目 | 改前 | 改后 |
|---|---:|---:|
| 字节数 | `377424` | `381381` |
| LF 行数 | `3267` | `3290` |
| SHA-256 | `690015BB5C3B4800539F98C5B75115A5A20B5C161ECE7422F9F271840F7C55A7` | `90AD2BC15E74726A06C9583BDE42C30B72A2B46E010F434EC9BE7C425D102A67` |

- 编码为 UTF-8 无 BOM；CRLF 为 0；孤立 CR 为 0；文件以 LF 结尾。
- 行尾空白仍为 4 行（页眉 Markdown 强制换行），与改前一致，未新增。
- `rg "具体生产数值仍为 TBD"` 无命中；`rg "AM-Q2.*仍待定"` 无命中。
- `rg "AM-Q2"` 仍命中的 `:128`、`:155`、`:157` 是 r11 时期评审/输入追溯的历史记录，`:2915` 是测试 fixture 说明；均不是当前 AM-Q2 状态。
- 委派说明记录改前 2605 行；实际文件 LF 计数与新增 r29 块（23 行）推算的改前行数为 3267，二者不一致，本报告以实际文件计数为准。

## 8. 声明

- 本报告与 r29 设计修订未经独立评审。
- 未实施、未部署、未做真实容量或 P95 实测。
- 非真人签字；不声称容量验收通过，不声称已达标，不声称已实施或已经独立评审。
- 未新增 ADR；参数值不构成架构决策。
