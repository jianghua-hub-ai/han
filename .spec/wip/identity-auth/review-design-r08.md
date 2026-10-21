# identity-auth 详细设计 r08 独立回归复审报告

- 复审日期：2026-10-20
- objectType：design
- Reviewer：非作者独立 Reviewer
- 原报告：`.spec/wip/identity-auth/review-design-r07.md`
- 原开放项：`IA-DES-R06-N001`
- 主对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 主对象修订：`identity-auth-design-ir01-r08`
- 附属对象：`.add/project/knowledge/design-workspace/identity-auth/stories.md`
- 附属对象修订：`identity-auth-stories-ir01-r08`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改 design、stories 或 upstream。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `6bdc960f323736e2397ea321e64b8718a483604aaeac5d3bfa59af08abb0c0ca` | `6bdc960f323736e2397ea321e64b8718a483604aaeac5d3bfa59af08abb0c0ca` | 一致；与委托相符 |
| `stories.md` | `3d99ef5e8f7e2022e3348c82e73c0c66768da14883663fd27a51a576c3546508` | `3d99ef5e8f7e2022e3348c82e73c0c66768da14883663fd27a51a576c3546508` | 一致；与委托相符 |

按 `<!-- ADD:CHECKPOINTS -->` 前正文排除确认段，并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:2231fb3e1646fc26fa367da48362ec2610418e8b07c44866a2eb11c675172de2`

与 Lead artifact 一致。

### 1.2 上游与生产者快照

| 依据 | 修订/状态 | 完整 SHA-256 / checkpoint |
|---|---|---|
| architecture | `architecture-ir01-r11` accepted | `6466f88b579b56b90e2aa7d658657a117337bf6a0057321b7f002b16b6d9edce`；`decision-0485d8f9ac9ce04ce734dfc8bdc6ecf5b96b226f81fa21e781afb4d7717ea3a8` |
| identity boundary | `identity-auth-boundary-ir01-r10` accepted | `33cf72545443130679167488c37b92dba74ef579c6829a841324b73b29c53fd1` |
| article boundary | `article-management-boundary-ir01-r11` accepted | `ad94c1f54a192b22c8cd6ada49a21f7a3c471d6790b79e1d7113008ac7e702f7` |
| article provider | `article-management-design-ir01-r05` fixed | `9991efd591f3ac37deb2ed596b218ddce08e74c6ad9a914ce77967215b0c9813` |
| article stories | `article-management-stories-ir01-r05` | `4504ba5a2904ef927f5cfbef89804878a59bfb9de94aa1eb37b815e2251503a7` |
| ADR-0020 | accepted revision | `content.md` SHA-256 `b152d6ec2765a87ba144a474997ed5c82011b1b8af6c9b457e56675402617c15` |

本次重新完整读取 r07 报告、article review r04，以及 r08/r05 的 S16 schema、fixture、接口、持久化和恢复段；复用同会话中版本未变的 Reviewer Role、Review Skill、规则、安全红线与 r01～r06 历史报告内容。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

检查：

1. 唯一 20 字段 DTO 的名称、顺序、JSON 类型和必填性；
2. canonical 是否严格固定为前 16 字段；
3. `status/evidence_digest/evidence_ref/completed_at` 是否排除；
4. 唯一重放计数字段是否为 `tombstones_replayed`；
5. producer/consumer fixture 与 `169e0f…` digest 是否相同且可独立复算；
6. identity 是否只从 DTO 重算，不读 article 内部表；
7. RecoveryRun backup、首次与二次 `evidence_ref/evidence_digest` 比较及 fail-closed；
8. `ReplayAndVerify/ReadStatus`、双 namespace、`MarkConsumed/open` 顺序。

### Round 2：生产者兼容与历史回归

与 article r05 逐字段比较 DTO/schema/fixture，并沿 11 条 identity-owned US、S1～S16、stories DAG、FMEA、安全、测试和 r01～r07 历史 findings 检查新增回归。

## 3. 原 finding 复审

### [MEDIUM] IA-DES-R06-N001 — still-open

- 原 finding：Article replay ready 证据合同未形成 producer/consumer 可共同实施的唯一 schema。
- 对象/修订：identity-auth design r08。
- 位置：
  - identity：`design.md:1483-1539`、`1694`、`1714-1715`、`1790-1792`、`2039-2040`；
  - article r05：`article-management/design.md:859-916`、`1019-1050`、`1703-1732`。
- 已关闭部分：
  - 双方字段数量、名称和顺序已统一为 20 项；
  - canonical 已统一为前 16 项，四个排除字段一致；
  - `tombstones_replayed` 已成为唯一名称；
  - operation/audit counts、`evidence_ref` 和 completed_at 幂等语义已补齐；
  - `ReplayAndVerify/ReadStatus` 均使用 run+backup；
  - identity 只凭 DTO 重算，二次比较 backup/ref/digest，顺序早于 `MarkConsumed/open`；
  - 缺失、额外字段、类型错误、错 backup、unknown/failed、摘要/ref 变化均声明 fail closed。
- 仍未关闭：
  1. **字段类型不一致**：identity r08 把字段 7 `inventory_scope` 定义为非空 JSON object；article r05 的唯一 DTO、Evidence 表和固定 fixture 均把它定义为 string。
  2. **共享 fixture 不同**：identity r08 的“共享 schema fixture”使用 `opaque-run/opaque-backup`、`inventory_scope={}`、占位摘要和 `sha256:<canonical-hex>`；article r05 的共享 fixture 使用 `run-01`、固定 backup、string scope、具体摘要值和固定 digest。两者不是同一 fixture。
  3. **固定摘要未落在 consumer**：article r05 的 canonical bytes 独立复算结果为  
     `sha256:169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`，与 article 声明一致；identity r08 未引用该 canonical bytes/固定 digest，其示例按自身字段和值序列化得到另一摘要，不能证明 producer/consumer 使用同一个契约样本。
  4. **非 ready DTO 约束仍不完全一致**：identity 对 20 字段声明“不允许缺失/null”，并把 `evidence_ref` 约束为非空；article r05 对 unknown/failed DTO 明确使用空字符串/零值。虽然两者最终都 fail closed，但解析合同并非逐字段兼容。
- 触发：article r05 返回其固定 ready fixture，identity r08 按自己声明的 `inventory_scope=object` 严格解析；或双方执行所谓共享 fixture 契约测试。
- 影响：
  - 严格消费者会拒绝合法 provider DTO，恢复永久停在 Gate；
  - 若实现者放宽类型或绕过共享 fixture 检查，canonical bytes 和 digest 可因隐式转换产生歧义，重新削弱 stale/错 backup 证据验证。
- 最小修复建议：
  1. identity 删除本地近似 fixture，直接引用并复制 ADR-0020/article r05 的唯一固定 fixture；
  2. 将 `inventory_scope` 类型统一为唯一上游采用的 string，或先修订 ADR/architecture/article 后再同步两侧，不能一侧自行选择 object；
  3. 在 identity 中登记完整 fixture canonical bytes 和固定 digest `169e0f…26a7`，作为 consumer 独立重算验收；
  4. 对 unknown/failed DTO 明确逐字段零值规则，保证同一 JSON schema 可解析后再按 status fail closed。
- 验证：
  - article 输出固定 fixture，identity 不访问任何内部表即可得到 `169e0f…26a7`；
  - 双向契约测试验证 20 字段的名称、顺序、类型、必填/零值规则；
  - object/string 替换、字段缺失/额外/null、别名和隐式类型转换全部被拒绝；
  - backup/ref/digest 二次比较与所有 fail-closed 场景继续通过。

因此，r07 报告要求的“同一 fixture、同一 JSON 类型、同一 digest”尚未达成，原 finding 不能关闭。r08 文档中的 `IA-DES-R07-N001` 属整改追溯别名；本报告继续使用原稳定 ID `IA-DES-R06-N001`。

## 4. 历史回归

r01～r05 已关闭的 15 个稳定 findings 未发现回归：

- 隐私告知/处理门禁、临时凭据状态机、恢复账号 Gate；
- SMTP TLS、零文章清理、login CSRF、防暴力；
- 公共错误、分页、Story DAG；
- onboarding 换版、三类限流桶、恢复幂等状态；
- kept_deleted canonical 删除；
- Recovery Journal 全 retained key 防重放。

r08 仍保持：

- 11 条 identity-owned US 全部映射；
- S1～S16 全部可定位；
- identity 不读取 Article Deletion Journal、inventory、DeletionOperation、AuditEvent、Evidence 或 Article 内部表；
- ReplayAll、operation/audit 对账、evidence、双 namespace 和二次读取均位于 Recovery Journal `MarkConsumed/open` 之前；
- stories 已同步 20 字段、canonical 前 16、`tombstones_replayed`、DTO-only 和 fail-closed 要求；
- 41 条依赖边无环，正文依赖未发现漂移。

## 5. 新 findings

未发现独立于 `IA-DES-R06-N001` 的新增 CRITICAL、HIGH 或 MEDIUM finding。

## 6. 静态验证与限制

- design 2227 行，stories 399 行。
- PlantUML：design 22 块、stories 1 块；起止标记成对。
- 未发现字面 `\n` 或 `skinparam linetype ortho`。
- design 19 个、stories 12 个本地 Markdown 链接均存在。
- Story DAG 41 条边，未发现环。
- 11 条 identity-owned US 与 S1～S16 均可定位。
- fixture canonical bytes 546 字节；本地 SHA-256 独立复算为 `169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`。
- 本机无 `plantuml` 命令；仅做静态检查，未实渲染。
- 当前无业务代码、schema、迁移、契约生成物或测试；未运行不存在的构建、lint、单元、集成或 E2E。
- article r05 本轮作为固定生产者读取，但未把其独立复审结果推断为 PASS。
- 既有 Q1、容量/期限、部署隐私输入和 evidence 保留/退役仍为 deferred/TBD，不在本轮伪判完成。

## 7. 结束漂移复核

结束时再次计算：

- identity design r08：`6bdc960f323736e2397ea321e64b8718a483604aaeac5d3bfa59af08abb0c0ca`
- identity stories r08：`3d99ef5e8f7e2022e3348c82e73c0c66768da14883663fd27a51a576c3546508`
- article provider design r05：`9991efd591f3ac37deb2ed596b218ddce08e74c6ad9a914ce77967215b0c9813`
- article provider stories r05：`4504ba5a2904ef927f5cfbef89804878a59bfb9de94aa1eb37b815e2251503a7`

identity 固定对象与开始值一致；固定 article r05 生产者快照也未在复审期间漂移。

## 8. 最终结论与复审条件

**最终结论：NOT PASS。**

- `IA-DES-R06-N001`：**still-open**。
- 历史已关闭 findings：无回归。
- 新增独立阻断 finding：无。

r08 已统一大部分字段、命名、canonical 范围、ref/count 和恢复顺序，但 `inventory_scope` 类型与固定 fixture 仍不兼容；consumer 尚不能用 producer 的唯一 fixture 复算 `169e0f…26a7`。

复审条件：

1. identity 与 article 固定完全相同的 20 字段 JSON 类型和 unknown/failed 零值规则；
2. identity 使用与 article r05/ADR-0020 相同的固定 fixture 和 canonical bytes；
3. identity 独立重算并登记完整 digest `169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`；
4. 保留 DTO-only、backup/ref/digest 二次比较、input 变化失效和 `MarkConsumed/open` 前顺序；
5. 独立 Reviewer 针对原 `IA-DES-R06-N001` 做下一轮联合契约复审。

本结论只针对固定 r08 对象及 §1.2 固定 upstream/provider 快照，不代替用户/Lead 的风险接受、设计确认或实施授权。
