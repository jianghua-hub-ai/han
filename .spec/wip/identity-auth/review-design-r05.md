# identity-auth 详细设计第五轮独立复审报告

- 复审日期：2026-10-20
- objectType：design
- 先前报告：`review-design-r01.md`～`review-design-r04.md`
- 主对象：`identity-auth-design-ir01-r05`
- 附属对象：`identity-auth-stories-ir01-r05`
- 唯一开放项：`IA-DES-R03-N002`
- 结论：**PASS**
- 写入边界：仅新建本报告；未修改被审对象、先前报告或上游。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `73a6d7ed61fab3155b3ea9c595281a09d0ad4833ab9152dcff0e7b9750b1525c` | `73a6d7ed61fab3155b3ea9c595281a09d0ad4833ab9152dcff0e7b9750b1525c` | 一致 |
| `stories.md` | `2551f6c9aa5909d47947af42fe74878d3d724ecdebcc8e32583b299b04125d8b` | `2551f6c9aa5909d47947af42fe74878d3d724ecdebcc8e32583b299b04125d8b` | 一致 |

按确认段排除并将尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:cd2e1eccc87330ca4f7f027f6a7d86f4ea724bcaea44eb4ee5840de17d2741bc`

与 Lead 提供值一致，无对象漂移。

### 1.2 上游快照

| 依据 | 修订/状态 | 完整 SHA-256 |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r08` accepted | `49d1caacbb7963124c3bdd23e6228b3a9c8c842a7fc7bffe40b9fe0ea671162b` |
| identity boundary | `identity-auth-boundary-ir01-r07` accepted | `646d3a1368242f2163147242a2b6208b4842f6ce1c7810b003c43ea7ded5a6fd` |
| article boundary | `article-management-boundary-ir01-r07` accepted | `d507aba4041da976168466c804ccfa2f98002c8747ef2c836f54eafbe83c5036` |
| ADR index | ADR-0001～0019 | `1469cb25445f2bedb1f8dc27e7b77a89d74d24b9968e867b1b56ccab53650432` |
| framework ADR | 修订 ADR-0019 | `cb43bbced82f3d89231fe431e143b80390c4e40c61e7355822074deb37273794` |
| identity ADR | 当前 | `4a1141b55d882a76db332dd5e75cca640a2c0ca7121980910477bc408a42bc45` |

architecture r08 checkpoint：
`decision-fb7593a171dc466763f912f0bfc80892c9b3d143827ea5c58dbfb7d16935267b`。

## 2. 复审范围

### Round 1：唯一开放项定点复审

检查 N002 的五个关闭条件：

1. keyring 只有一个 active 写入 key；
2. 历史 key 以 verify-only 状态完整保留；
3. Begin 使用全部 retained key 计算候选 fingerprint；
4. key 退役不得破坏仍需承担的历史判重；
5. key 缺失、未知、迁移不完整或候选计算失败时 fail closed。

### Round 2：回归检查

沿 11 条 US、S1～S15、RecoveryJournal/RecoveryRun、数据配置、部署恢复时序、测试矩阵和 Story DAG 检查整改是否破坏 canonical 删除、身份恢复、幂等、API 或文章所有权。

## 3. 开放项状态

### IA-DES-R03-N002 — closed

- 修复证据：`design.md:701-727`、`1453-1464`、`1487-1503`、`1679-1746`、`1935`；architecture r08；修订 ADR-0019；boundary r07 S15。
- keyring 合同：
  - 恰有一个 `active` key，可有多个 `verify_only` retained key；
  - 每条 Journal 记录保存稳定 `key_id/key_version`；
  - 新记录只使用 active key，verify-only key 不写新记录。
- 跨版本判重：
  - `Begin` 接收候选 recovery secret，而不是只接收 active-key fingerprint；
  - provider 使用 active 和全部 verify-only retained key 分别计算候选指纹；
  - 每个候选与相同 key ID/version 的历史记录比较，任一 begun/consumed 命中即拒绝。
- key 生命周期：
  - 只要某 key 对应的 Journal 记录仍承担已批准保留或归档范围内的防重放责任，该 key 不得退役；
  - keyring、Journal 和 HMAC key 版本作为不可拆分制品整体备份、迁移和恢复；
  - 不允许为性能提前删除 retained key。
- fail-closed：
  - Journal 有未知/缺失 key 引用、key 无法加载、候选无法计算、只迁移 Journal、只迁移 active key、verify-only 集合不完整或 Journal/RecoveryRun 不一致时均保持停服。
- v1/v2/v3 场景：
  - secret S 在 v1 consumed 后，active 切换到 v2 或 v3 时，仍由 retained v1 key计算出历史指纹并拒绝；
  -不同 secret 使用当前 active key可正常 Begin；
  -测试矩阵已明确缺 v1/v2 key、未知 key、计算失败和迁移不完整的拒绝行为。

原 finding 的“同一 secret 因 HMAC key 轮换产生不同 fingerprint 而绕过一次性限制”触发路径已消除。

## 4. 新 findings

未发现新增 CRITICAL、HIGH 或 MEDIUM finding。

## 5. 回归与静态验证

- 11 条 identity-owned US 映射完整。
- design 与两份 boundary r07 的 S1～S15 语义一致。
- Recovery Journal/keyring 未成为账号、文章或审计真相源。
- canonical 删除、账号 recovery Gate、幂等正式状态机和公共错误合同未被本次整改破坏。
- Story DAG：41 条边，未发现环；正文依赖未发现新增不一致。
- PlantUML：design 22 块、stories 1 块；起止标记成对，未发现字面 `\n` 或 `linetype ortho`。
- design 中 15 个本地链接均存在。
- 当前无代码、数据库 schema、迁移、API schema 或测试；未运行不存在的构建、lint、单元、集成或 E2E。
- 本机无可用 PlantUML CLI/JAR，未执行 PNG 实渲染。
- Recovery Journal 持久卷介质、备份/迁移周期、retained key 数量上限与具体退役周期仍是已声明的运行输入；其数值缺失不改变当前 fail-closed 设计合同。

## 6. 最终结论

**PASS。**

- `IA-DES-R03-N002`：closed。
- r01～r04 报告产生的全部 findings 已关闭。
- 本轮未发现新增阻断 finding。

该 PASS 仅表示固定 r05 设计对象通过本次独立静态复审，不表示：

- Q1 等已声明待决定事项已经拍板；
- design 自动从 `draft` 转为 `implementable`；
- 代码、迁移、部署、真实恢复演练或 E2E 已通过；
- 用户/Lead 已完成成果接受或实施授权。

后续由 Lead 核对固定对象、实际确认状态和适用 Gate，再决定是否进入任务拆解或补充未决输入。
