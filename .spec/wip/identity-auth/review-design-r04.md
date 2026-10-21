# identity-auth 详细设计第四轮独立复审报告

- 复审日期：2026-10-20
- objectType：design
- 先前报告：`review-design-r01.md`、`review-design-r02.md`、`review-design-r03.md`
- 主对象：`identity-auth-design-ir01-r04`
- 附属对象：`identity-auth-stories-ir01-r04`
- 开放项：`IA-DES-R03-N001`、`IA-DES-R03-N002`
- 结论：**NOT PASS**
- 写入边界：仅新建本报告；未修改被审对象、先前报告或上游。

## 1. 对象和依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `6fd8d9800177a3de01ed4958190f1002fdb841b2f50c7136c9936c06e717e17a` | `6fd8d9800177a3de01ed4958190f1002fdb841b2f50c7136c9936c06e717e17a` | 一致 |
| `stories.md` | `940957b1ed2776180f9516205e78dec9a290d3a3b5c56db7c29edf72ced01e10` | `940957b1ed2776180f9516205e78dec9a290d3a3b5c56db7c29edf72ced01e10` | 一致 |

按确认段排除并把尾部规范化为单个 LF 后，design artifact digest 为：

`sha256:d2a842d05f7d6bf4b0dba4403175462148c7c134fdc71b2cee224b77aca13407`

与 Lead 值一致，无对象漂移。

### 1.2 上游快照

| 依据 | 修订/状态 | 完整 SHA-256 |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r07` accepted | `6fdba21a15503949bc289d44f756e3049e35493bb9e0277b9e54ca10184eb440` |
| identity boundary | `identity-auth-boundary-ir01-r06` accepted | `9e18eed37f9ac9359e61e68ee30c9c815c7f2f2dea42fb74b60f42e288925de3` |
| article boundary | `article-management-boundary-ir01-r06` accepted | `4d80146a3f68e3c6184e2aa7e78345e386c655f65b5b93167af29cead632d05a` |
| ADR index | ADR-0001～0019 | `7ae51e991097a3b46770f56f916f75591133ca54c6861709ee1b3fd96d030594` |
| framework ADR | 含 ADR-0019 | `0474e959fc923f4ff571ae829fdb1ce837344a37e5d20fc3f5ad4aabb810f755` |
| identity ADR | 修订 ADR-0007 | `c45010bd118ceb5dfaae14ebb288cb5c0ee32e4e198822e5717be50a2b279f67` |

architecture r07 checkpoint：
`decision-6fbb576c23f29ef490a1067d754e0709622200ee409c047a771476ceb6a6d084`。

## 2. 两轮检查范围

### Round 1：开放项定点复审

- N001：检查 `kept_deleted` 是否调用 canonical 删除协调，并覆盖 User、Identifier、Session、永久/临时凭据、Article、retained_email 和 AuditEvent。
- N002：检查 Recovery Journal 是否在业务数据库恢复域外持久记录 run/secret 消费状态，能阻断同 run、跨 run、损坏和数据库回退后的重放。

### Round 2：整改回归

沿 11 条 US、S1～S15、API、数据、恢复时序、Journal/RecoveryRun 对账、文章删除接缝和 Story DAG 检查新增阻断风险；未重开 r07 已接受的持久卷文件提供方选型。

## 3. 开放项状态

### IA-DES-R03-N001 — closed

- 修复证据：`design.md:1217-1285`、`1587`、`1665`、`1683-1689`；两份 boundary r06 的 S4/S14。
- 日常删除与恢复 `kept_deleted` 共用 `CanonicalDeleteCoordinator` 和同一跨模块事务。
- 恢复模式重建：
  - User 墓碑及 `deleted_at`；
  - username 永久保留及 email/phone 生命周期；
  - Session、永久摘要和 TemporaryCredential 失效；
  - Article `author_id=NULL`、`held`、`orphaned_from_user_id`；
  - 违规状态保留；
  - retained_email 与零文章清理；
  - AuditEvent 和 RecoveryAccountReview 完成结论。
- 有文章、零文章、违规文章和部分已满足状态均有矩阵；无法证明来源或任一步失败时保持 Gate/readiness 关闭。
- 原“只改 User 而未重建删除事务”的触发场景已消除。

### IA-DES-R03-N002 — still-open

- 已修部分：`design.md:701-714`、`1447-1451`、`1487-1490`、`1672-1689` 和 ADR-0019 已定义业务库外 Recovery Journal、run ID、版本化 HMAC fingerprint、Begin/Consumed、校验链、fsync、持久卷及 Journal/RecoveryRun 双向对账。
- 未关闭触发：`secret_fingerprint = HMAC(versioned_journal_key, recovery_secret)`。同一 recovery secret 在 HMAC key 从 v1 轮换到 v2 后会得到不同 fingerprint。当前 Begin 只拒绝“相同 fingerprint 已存在”，没有要求对输入 secret 使用全部仍保留的历史 key 逐一计算候选 fingerprint，或在 key 迁移时把历史 secret 指纹重新投影到新版本。
- 现实场景：
  1. run-1 使用 secret S、key v1，Journal 保存 `fp_v1` 并 consumed；
  2. Journal key 轮换到 v2，旧 v1 key仍按合同保留；
  3. run-2 再次输入 secret S，系统只用 current v2 计算 `fp_v2`；
  4. `fp_v2 != fp_v1`，现有“同 fingerprint 去重”无法识别复用。
- 影响：跨 key-version 的旧 recovery secret 可以再次建立初始管理员恢复信任根，未满足原 finding 的“一次性、跨恢复批次禁止复用”合同。
- 最小修复：固定跨版本查重算法。可选方向包括：
  - Begin 前用所有未退役 Journal fingerprint keys 对输入 secret 计算候选指纹，任一命中历史记录即拒绝；或
  - key 轮换时在受控迁移中为所有历史 secret 指纹建立新版本等价索引，并保持不可逆关联。
  必须同时定义 key 退役前置：只要仍需阻断相关历史 secret，就不能丢失对应查重能力。
- 验证：同一 secret 在 v1 consumed 后，以 v2/v3 再次 Begin 必须拒绝；不同 secret 在新版本正常通过；旧 key 缺失、迁移不完整或候选计算失败均保持停服。

## 4. 新 findings

未发现独立于 N002 的新增 CRITICAL/HIGH/MEDIUM 阻断项。

## 5. 回归与静态验证

- 11 条 identity-owned US 映射保持完整。
- S1～S15 在 design 与两份 boundary r06 中语义一致。
- Story DAG：41 条边，未发现环；正文依赖未发现新增不一致。
- PlantUML：design 22 块、stories 1 块；起止标记成对，未发现字面 `\n` 或 `linetype ortho`。
- design 中 14 个本地链接均存在。
- canonical 删除未扩展为文章恢复能力；Recovery Journal 未成为账号、文章或审计真相源。
- 当前无代码、schema、迁移或测试，未运行不存在的构建、lint、单元、集成或 E2E。
- 本机无可用 PlantUML CLI/JAR，未执行实渲染。
- Recovery Journal 持久卷介质、备份/迁移周期、保留期和 key 轮换责任仍为已声明运行输入；本复审不把这些数值 TBD 重开为缺陷。

## 6. 最终结论与后续条件

**最终结论：NOT PASS。**

- `IA-DES-R03-N001`：closed。
- `IA-DES-R03-N002`：still-open。
- 无新增独立阻断 finding。

在 N002 的跨 HMAC key-version secret 复用场景关闭前，design 不得转为 `implementable`。

后续复审需：

1. 提供新 design/stories 修订、完整 SHA-256 和规范 artifact digest；
2. 明确并固定跨 fingerprint key 版本的复用检测算法及 key 退役条件；
3. 增加跨 v1→v2→v3 的 recovery secret 重放测试矩阵；
4. 若修订 ADR-0019、architecture 或 boundary，附 accepted checkpoint 和 S1～S15 回归。

本结论只针对固定 r04 对象，不代替用户/Lead 的风险接受、设计确认或实施授权。
