# identity-auth 详细设计 r09 独立回归复审报告

- 复审日期：2026-10-20
- objectType：design
- Reviewer：非作者独立 Reviewer
- 原报告：`.spec/wip/identity-auth/review-design-r08.md`
- 原开放项：`IA-DES-R06-N001`
- 主对象：`.add/project/knowledge/design-workspace/identity-auth/design.md`
- 主对象修订：`identity-auth-design-ir01-r09`
- 附属对象：`.add/project/knowledge/design-workspace/identity-auth/stories.md`
- 附属对象修订：`identity-auth-stories-ir01-r09`
- 结论：**PASS**
- 写入边界：仅新建本报告；未修改 design、stories 或 upstream。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `d05132a4bc4ecacf0243059d53eafe8b963c79383d9a07209dd903350b823d85` | `d05132a4bc4ecacf0243059d53eafe8b963c79383d9a07209dd903350b823d85` | 一致；与委托相符 |
| `stories.md` | `d8ac23a5eb7bf535b2111bc3411b6874e455065b237695470b3addefdaab4ad5` | `d8ac23a5eb7bf535b2111bc3411b6874e455065b237695470b3addefdaab4ad5` | 一致；与委托相符 |

按 `<!-- ADD:CHECKPOINTS -->` 前正文排除确认段，并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:df484400154f7b64c00c40842e02eb254408bdb1c764768c51bf9672dbd09c61`

与 Lead artifact 一致。

### 1.2 上游与生产者快照

| 依据 | 修订/状态 | 完整 SHA-256 / checkpoint |
|---|---|---|
| architecture | `architecture-ir01-r12` accepted | `5cac5b922fbe976640a5617a9a24136abb4283ad7ad866a6ab933498b5bd1dce`；`decision-0347e1026fd9148ad16f66750023cb50445a1d7c133db6d82669ea14865b318f` |
| identity boundary | `identity-auth-boundary-ir01-r11` accepted | `fb5197625eafc07bf0a98e5f09abd7c60e018ee6507bc1d47b88dc40b651767c` |
| article boundary | `article-management-boundary-ir01-r12` accepted | `a4d70c060a7d3d8fe986bc32b7047b28c57e2fe8b19733d423f5a56814ede5e4` |
| article provider | `article-management-design-ir01-r06` fixed | `71b8953790b828d67373cae1eb020c26d5e4d6c10b434c057e582fdf74d17a51` |
| article stories | `article-management-stories-ir01-r06` | `44bf8bb237ec4ba7cd79e0237ccac92fefb1d7fdc2313f1e961f1fef5ec7b307` |
| ADR-0020 | accepted final variant revision | `content.md` SHA-256 `a0594229fcfda0b8f243a476024bf51825f8dfb12671b41675fc270fa12dbf88` |

本次完整读取 r08 报告、article review r05，并逐字段读取 identity r09、article r06、architecture r12、两份 accepted boundary 和 ADR-0020 的 tagged-union、fixture、接口、恢复及测试合同。

## 2. 两轮实际检查范围

### Round 1：原 finding 定点复审

检查：

1. ready variant 是否严格为 20 个必填非 null 字段；
2. unavailable variant 是否严格为 6 个字段；
3. `inventory_scope` 是否统一为 JSON string；
4. canonical 是否只含 ready 前 16 字段；
5. fixture canonical bytes 和固定 digest 是否相同；
6. unavailable 是否不进入 digest 重算；
7. DTO-only 重算、RecoveryRun backup 比较和二次 ref/digest 比较；
8. 混合 variant、字段越界、错误类型、未知状态/版本和 stale input 是否 fail closed；
9. `MarkConsumed/open` 是否晚于 replay、对账、ready evidence、双 namespace 和二次读取。

### Round 2：生产者与历史回归

与固定 article r06 逐字段核对 ready/unavailable schema、reason_code、fixture、canonical、接口和恢复顺序；并沿 11 条 identity-owned US、S1～S16、stories DAG、安全、FMEA、测试及 r01～r08 历史 findings 检查回归。

## 3. 原 finding 状态

### IA-DES-R06-N001 — closed

- 修复证据：
  - identity：`design.md:1485-1575`、`1730`、`1750-1751`、`1824-1828`、`2075-2078`；
  - article provider：`article-management/design.md:862-948`、`1050-1081`、`1734-1763`；
  - architecture r12、identity boundary r11、article boundary r12、ADR-0020。
- ready variant：
  - 双方固定相同的 20 个字段、顺序和 JSON 类型；
  - `inventory_scope` 均为 string；
  - 唯一重放字段名为 `tombstones_replayed`；
  - 前 16 字段 canonical，`status/evidence_digest/evidence_ref/completed_at` 排除；
  - 全部字段必填且非 null，status 固定为 ready。
- unavailable variant：
  - 双方只允许 `evidence_version,run_id,backup_ref,namespace,status,reason_code` 六个字段；
  - status 仅 `pending|unknown|failed`；
  - 禁止 canonical、digest、ref、time、null、零值补齐或混合字段；
  - identity 收到后立即 fail closed，不调用 canonical digest 计算。
- fixture：
  - producer/consumer 使用相同 ready JSON 和相同 canonical bytes；
  - Reviewer 独立复算 canonical bytes 长度为 546 字节；
  - SHA-256 为 `169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`，与双方及 ADR-0020 一致。
- DTO-only：
  - identity 只从 ready DTO 前 16 字段重算摘要；
  - 不读取 Journal、inventory、DeletionOperation、AuditEvent、Evidence 或 Article 内部表。
- 绑定与顺序：
  - `ReplayAndVerify/ReadStatus` 均以 run+backup 调用；
  - ready 在锁定 RecoveryRun 的事务中比较 backup、namespace、版本、digest 和计数；
  - `ready_to_consume` 前二次读取必须得到相同 `evidence_ref/evidence_digest`；
  - 任一 unavailable、混合 variant、字段/类型错误、摘要/ref 不匹配或 input 变化均清除 ready 并停服；
  - Recovery Journal `MarkConsumed` 和 readiness 位于全部验证之后。

原 finding 的严格消费者拒绝、隐式类型转换、fixture 摘要分歧和 stale/错 backup ready 路径均已消除。

## 4. 历史 finding 回归

r01～r05 已关闭的 15 个稳定 findings 均未发现回归：

- 隐私告知/ProcessingGate；
- 临时凭据不回退永久密码；
- 旧备份账号、角色、密码、Session 与幂等状态 fail closed；
- SMTP 强制 TLS；
- 零文章 retained_email 清理；
- login CSRF、防暴力、公共错误、分页；
- Story DAG；
- onboarding 换版；
- 三类限流桶；
- `recovery_review_required`；
- kept_deleted canonical 删除；
- Recovery Journal 全 retained key 跨版本判重。

全量回归结果：

- 11 条 identity-owned US 全部映射；
- S1～S16 均有设计落点；
- stories 公共验收与 identity boundary r11/article boundary r12 一致；
- identity 不拥有文章删除状态或恢复能力；
- ReplayAll、DeletionOperation/Audit 对账、ready evidence、双 namespace 与二次读取全部早于 `MarkConsumed/open`；
- 41 条依赖边无环，未发现正文依赖漂移。

## 5. 新 findings

未发现新增 CRITICAL、HIGH、MEDIUM 或 LOW finding。

## 6. 静态验证与限制

- design 2267 行，stories 401 行。
- PlantUML：design 22 块、stories 1 块；起止标记成对。
- 未发现字面 `\n` 或 `skinparam linetype ortho`。
- design 20 个、stories 12 个本地 Markdown 链接均存在。
- Story DAG 41 条边，未发现环。
- 11 条 identity-owned US 与 S1～S16 均可定位。
- ready fixture canonical bytes 546 字节；独立 SHA-256 为 `169e0f30407867083598ab146c950968e1b5f508ed4fd5d5198bb61e788926a7`。
- 本机无 `plantuml` 命令；仅做静态检查，未实渲染。
- 当前无业务代码、schema、迁移、契约生成物或测试；未运行不存在的构建、lint、单元、集成或 E2E。
- article r06 本轮作为固定生产者读取；本报告不代替其独立复审结论。
- 初始管理员轮换、容量/期限、隐私部署输入及 evidence 保留/退役等既有 deferred/TBD 未被本次 PASS 改写。

## 7. 结束漂移复核

结束时再次计算：

- identity design r09：`d05132a4bc4ecacf0243059d53eafe8b963c79383d9a07209dd903350b823d85`
- identity stories r09：`d8ac23a5eb7bf535b2111bc3411b6874e455065b237695470b3addefdaab4ad5`
- article provider design r06：`71b8953790b828d67373cae1eb020c26d5e4d6c10b434c057e582fdf74d17a51`
- article provider stories r06：`44bf8bb237ec4ba7cd79e0237ccac92fefb1d7fdc2313f1e961f1fef5ec7b307`

identity 固定对象与开始值一致；固定 article r06 生产者快照也未在复审期间漂移。

## 8. 最终结论

**PASS。**

- `IA-DES-R06-N001`：closed。
- 历史 findings：无回归。
- 新 findings：无。

该 PASS 仅表示固定 identity r09 设计对象通过本次独立静态复审，不表示：

- design 自动转为 `implementable`；
- 既有 deferred/TBD 已决定；
- article r06 已由本报告代为通过；
- 代码、数据库、契约测试、真实恢复演练或 E2E 已通过；
- 用户/Lead 已完成成果接受或实施授权。
