# article-management test-plan r24 证据报告

- 对象：`.add/project/knowledge/design-workspace/article-management/test-plan.md`
- 修订：`article-management-test-plan-ir01-r23` → `article-management-test-plan-ir01-r24`
- 日期：2026-11-02
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`
- 结论：仅完成 AMQ2-R02-F002（LOW）时点标注与 design r31 重新锚定；未执行测试、E2E、Go/SQL/PG 或真实环境验证，未经独立评审，非真人签字。

## 1. 基线与产物体积

### r23 改前

- 字节数：`326550`
- 行数：`1680`（按 LF 换行符计）
- SHA256：`C1923CC43CD5A5E28A5C1512B640AB961F0AE25A155CEE34F0AFDA01BD192AB9`
- 格式：UTF-8 无 BOM；LF-only（CRLF=`0`）

### r24 改后

- 字节数：`326640`
- 行数：`1680`（按 LF 换行符计）
- SHA256：`4C517E90B105D3C37C70F3430A71262AF9088B9D18FEE442640908740F39A34F`
- 格式：UTF-8 无 BOM；LF-only（CRLF=`0`）

改前基线三项与任务给定值完全一致。

## 2. AMQ2-R02-F002（LOW）

### 旧文本（第 1210 行）

```text
AM-Q2 已确认生产 limit/批次 DefaultLimit=20、MaxLimit=100、CandidateBatchSize=200 与首期规模 100000/10000；P95 首期验收基线（开发机测量参考，非目标环境结论）为列表≤300ms、详情≤150ms；目标环境真实 P95/并发/数据分布和容量模型仍 TBD。测试采用满足QueryConfig约束的显式合成值，仅验证边界、批次有界和取消/资源收敛，不制定新生产默认/性能阈值。旧S2.2/S7的保留schema/harness与已闭合结果只证明可复用底座，不证明新增Admin2已执行；真实部署/恢复/容量/ACL专项各守原责任。
```

### 新文本（第 1210 行）

```text
**时点标注（r24）：以下为 r24 现役口径，覆盖本节 r16 时点表述。** AM-Q2 已确认生产 limit/批次 DefaultLimit=20、MaxLimit=100、CandidateBatchSize=200 与首期规模 100000/10000；P95 首期验收基线（开发机测量参考，非目标环境结论）为列表≤300ms、详情≤150ms；目标环境真实 P95/并发/数据分布和容量模型仍 TBD。测试采用满足QueryConfig约束的显式合成值，仅验证边界、批次有界和取消/资源收敛，不制定新生产默认/性能阈值。旧S2.2/S7的保留schema/harness与已闭合结果只证明可复用底座，不证明新增Admin2已执行；真实部署/恢复/容量/ACL专项各守原责任。
```

处理说明：只在原段首追加时点标注，保留 AM-Q2 现役数值与 §7.7 历史段其余文本；未改写 r16 历史事实。

## 3. design r31 重新锚定

### front matter 旧值

```text
revision: article-management-test-plan-ir01-r23
r23_source_design_revision: "article-management-design-ir01-r30"
r23_source_design_artifact: "sha256:5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863; bytes:382472; lines:3294"
```

### front matter 新值

```text
revision: article-management-test-plan-ir01-r24
r24_source_design_revision: "article-management-design-ir01-r31"
r24_source_design_artifact: "sha256:63FD0691E1D66326B8EED7CFDFD8017417C60BB628CB961359087242D5D58B64; bytes:382562; lines:3294"
```

### design r31 实测对照

- 修订：`article-management-design-ir01-r31`
- 字节数：`382562`
- 行数：`3294`（按 LF 换行符计）
- SHA256：`63FD0691E1D66326B8EED7CFDFD8017417C60BB628CB961359087242D5D58B64`
- 对照结果：front matter 的 SHA256、字节数、行数与 design r31 实测值完全相等。

### 旧锚检索

| 旧锚/字段 | 改前出现次数 | 改后出现次数 | 处理 |
|---|---:|---:|---|
| `5BAA8373` | 1 | 0 | 已由 r31 锚替换 |
| `90AD2BC1` | 0 | 0 | 未出现，无需处理 |
| `76584E92` | 0 | 0 | 未出现，无需处理 |
| `r23_source_design` | 2 | 0 | 字段改名并更新为 r24/r31 |

未改写正文中的 `design r30` 说明；这些内容位于 r23 时点叙述或沿用 AM-Q2 已确认数值的现役表述，不属于本任务要求清理的旧哈希锚或 `r23_source_design` 字段。`last_updated` 保持 `2026-11-02`。

## 4. 覆盖与状态不变性

### 用例 ID 集合

- 提取规则：大小写敏感匹配 `\bA-[A-Z0-9]+(?:-[A-Za-z0-9]+)+\b`，去重排序。
- 改前数量：`167`
- 改后数量：`167`
- 集合差异：`0`，前后完全相同。

```text
A-ADM2-01
A-ADM2-02
A-ADM2-03
A-ADM2-04
A-ADM2-05
A-ADM2-06
A-ADM2-07
A-ADM2-08
A-ADM2-09
A-ADM2-10
A-ADM2-11
A-ADM2-12
A-ADM2-13
A-ADM2-14
A-ADM2-15
A-ADM2-16
A-ADM2-17
A-ADM2-18
A-ADM2-19
A-ADM2-20
A-CON-01
A-CON-02
A-CON-03
A-CON-04
A-CREATE-01
A-CREATE-02
A-CREATE-03
A-CREATE-04
A-CREATE-05
A-CREATE-06
A-DEL-01
A-DEL-02
A-DEL-03
A-DEL-04
A-DEL-05
A-DEL-06
A-DEL-07
A-DEL-08
A-DEL-09
A-DEL-10
A-ETAG-01
A-ETAG-02
A-ETAG-03
A-ETAG-04
A-MOD-01
A-MOD-02
A-MOD-03
A-MOD-04
A-MOD-05
A-MOD-06
A-MOD-07
A-MOD-08
A-MODR-01
A-MODR-02
A-MODR-03
A-MODR-04
A-MODR-05
A-MODR-06
A-NOT-01
A-NOT-01a
A-NOT-02
A-NOT-02a
A-NOT-02b
A-NOT-02c
A-NOT-02d
A-NOT-03
A-NOT-03a
A-NOT-03b
A-NOT-03c
A-NOT-04
A-NOT-04a
A-NOT-04b
A-NOT-04c
A-NOT-04d
A-NOT-05
A-NOT-05a
A-NOT-05b
A-NOT-05c
A-NOT-05d
A-NOT-05e
A-NOT-05f
A-NOT-05g
A-NOT-06
A-NOT-06a
A-NOT-06b
A-NOT-06c
A-NOT-07
A-NOT-07a
A-NOT-07b
A-NOT-07c
A-NOT-07d
A-NOT-08
A-NOT-08a
A-NOT-09
A-NOT-09a
A-NOT-09b
A-NOT-09c
A-NOT-09d
A-NOT-09e
A-NOT-09f
A-NOT-09g
A-NOT-09h
A-ORP-01
A-ORP-02
A-ORP-03
A-ORP-04
A-ORP-05
A-ORP-06
A-ORP-07
A-PERF-01
A-PERF-02
A-PERF-03
A-PERF-04
A-PERF-05
A-READ-01
A-READ-02
A-READ-03
A-READ-04
A-READ-05
A-READ-06
A-REC-01
A-REC-02
A-REC-03
A-REC-04
A-REC-05
A-REC-06
A-REC-07
A-REC-08
A-WIRE-CFG-01
A-WIRE-CFG-02
A-WIRE-CFG-03
A-WIRE-CFG-04
A-WIRE-CFG-05
A-WIRE-CFG-06
A-WIRE-JRN
A-WIRE-JRN-01
A-WIRE-JRN-02
A-WIRE-JRN-03
A-WIRE-JRN-04
A-WIRE-JRN-05
A-WIRE-JRN-06
A-WIRE-JRN-07
A-WIRE-LIFE-01
A-WIRE-LIFE-02
A-WIRE-LIFE-03
A-WIRE-LIFE-04
A-WIRE-LIFE-05
A-WIRE-LIFE-06
A-WIRE-LIFE-07
A-WIRE-LIFE-08
A-WIRE-RTR-01
A-WIRE-RTR-02
A-WIRE-RTR-03
A-WIRE-RTR-04
A-WIRE-RTR-05
A-WIRE-RTR-06
A-WIRE-WKR
A-WIRE-WKR-01
A-WIRE-WKR-02
A-WIRE-WKR-03
A-WIRE-WKR-04
A-WIRE-WKR-05
A-WIRE-WKR-06
A-WIRE-WKR-07
A-WIRE-WKR-08
A-WIRE-WKR-09
A-WIRE-WKR-10
```

### 状态词计数

按大小写敏感、独立词边界统计：

| 状态词 | 改前 | 改后 | 相同 |
|---|---:|---:|---|
| `planned` | 7 | 7 | True |
| `blocked` | 26 | 26 | True |
| `NOT RUN` | 24 | 24 | True |
| `PASS` | 109 | 109 | True |

按字面子串统计（包含 `planned-not-executed`、`PASS-for-*` 等复合写法）：

| 状态词 | 改前 | 改后 | 相同 |
|---|---:|---:|---|
| `planned` | 32 | 32 | True |
| `blocked` | 29 | 29 | True |
| `NOT RUN` | 24 | 24 | True |
| `PASS` | 118 | 118 | True |

`A-PERF-02` 仍为 `planned`，第 883 行未改。

### 性能与分页数值

| 字面值 | 改前 | 改后 | 相同 |
|---|---:|---:|---|
| `20/100/200` | 4 | 4 | True |
| `100000/10000` | 5 | 5 | True |
| `300ms` | 6 | 6 | True |
| `150ms` | 6 | 6 | True |
| `DefaultLimit=20` | 2 | 2 | True |
| `MaxLimit=100` | 2 | 2 | True |
| `CandidateBatchSize=200` | 2 | 2 | True |

### 原文保留段与改动边界

- 前后逐行比较：仅第 `5, 6, 7, 1210` 行变化，共 `4` 行。
- §11.2 历史段（第 1600–1659 行）：逐字相同=`True`。
- 未新增或删除用例，未改任何 `planned`/`blocked`/`NOT RUN`/`PASS` 状态语义。
- 未改 `design.md`、`tasks.md`、`plans/*`、`docs/**`、`stories.md`。

## 5. 写入范围声明

本任务实际写入：

- 修改 `.add/project/knowledge/design-workspace/article-management/test-plan.md`
- 新建并写入 `.spec/wip/article-management/evidence-am-q2/test-plan-r24/`
  - `report.md`
  - `before.md`
  - `after.md`
  - `change.diff`
  - `change.patch`

未执行任何 git 写操作。工作树中已有其他改动/未跟踪项不属于本任务写入范围，本任务未触碰。

## 6. 已执行与未执行

已执行：静态字节/行数/SHA256 校验、UTF-8/LF 校验、前后逐行 diff、用例 ID 集合与状态词计数、旧锚检索。

未执行：Go 测试、SQL/PostgreSQL、服务启动、真实 E2E、性能测试、独立评审、真人成果签收。上述未执行项不得由本报告推导为通过。
