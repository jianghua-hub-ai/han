# article-management 详细设计第二轮整改独立复审报告 r03

- 复审日期：2026-10-20
- objectType：design
- 先前报告：`review-design-r01.md`、`review-design-r02.md`
- 主对象：`article-management-design-ir01-r03`
- 附属对象：`article-management-stories-ir01-r03`
- 唯一开放项：`AM-DES-R02-N001`
- 结论：**PASS**
- 写入边界：仅新建本报告；未修改 design、stories、先前报告或上游。

## 1. 对象与依据固定

### 1.1 开始/结束哈希

| 对象 | 开始 SHA-256 | 结束 SHA-256 | 结果 |
|---|---|---|---|
| `design.md` | `290e33051a2e8e22cdec09924ffc952ec4a02a8484980d552ad10464948fe1c8` | `290e33051a2e8e22cdec09924ffc952ec4a02a8484980d552ad10464948fe1c8` | 一致 |
| `stories.md` | `19a64f5e810446dcd7fe53174b96e0e86e6c020119cad97995b3be5adf61cf90` | `19a64f5e810446dcd7fe53174b96e0e86e6c020119cad97995b3be5adf61cf90` | 一致 |

按 `## 确认记录` 排除确认段并将尾部规范化为单个 LF 后，artifact digest 为：

`sha256:d96c748b175f1bdf395c35c1bf27836806813f2e0745b265e2ec6c8fe0547ac6`

与 Lead 提供值一致，无对象漂移。

### 1.2 上游快照

| 依据 | 修订/状态 | 完整 SHA-256 |
|---|---|---|
| requirements | `requirements-ir01-r16` accepted | `999cba3c0cd25d2760e400a41d2a373611406854e76e24e453321913d71cbc4f` |
| architecture | `architecture-ir01-r09` accepted | `f69dd966f36f4d268a775657ddfc46e1b076c64098fd998e254fed3cfa886072` |
| article boundary | `article-management-boundary-ir01-r09` accepted | `473cf725552ee52bd5a420ee2667c9e1ffd04b04fca1a21a2f54c3f194385614` |
| identity boundary | `identity-auth-boundary-ir01-r08` accepted | `c9ebf1660b06252d33db588f65133ef57e05ffaa5ede95b996c314e55ca12a54` |
| content ADR | 含 ADR-0020 | `edacccadb48c76e20addf684cf09ed372182b10efb3b4d734337cdc34acdc3e5` |
| identity review baseline | r05 PASS | `0b870ebb8cabd86cc887ff03f445831ccfc27bacdb1179171b503ee8c3209de7` |
| article review r01 | 固定 | `94dd46f29d935c7cac5a5e8c01fe7d13e5d68424f7a497e5b0bac4a701f84729` |
| article review r02 | 固定 | `ddec01f5188f6210b98c1972619db33c37c57826b606b42ac8faa4591a51ef0d` |

## 2. 复审范围

### Round 1：开放项定点复审

验证作者受控详情和修改响应是否：

- 只返回 `title,body,moderation_status,editable`；
- ETag 只在 Header；
- 不返回 `violation_state`、`violation_category`、`hidden_reasons`、内部发布/删除状态或账号治理状态；
- 作者仍可修改自己的 violating 文章，且修改不清除治理约束。

### Round 2：回归检查

沿 8 条 US、S1～S16、16 个 REST endpoint、成员/作者/管理员三类 serializer、修改时序、永久删除、分页、通知和 Story 依赖检查新增阻断风险。

## 3. 开放项状态

### AM-DES-R02-N001 — closed

- 修复证据：`design.md:88`、`560`、`585-595`、`623-634`、`951-1016`、`1368-1388`、`1513-1519`、`1635`、`1673-1679`；article boundary r09。
- 作者受控 DTO 使用独立 `AuthorEditView` 和独立 `AuthorProjectionPolicy`，禁止复用管理员治理 DTO。
- 成功 body 字段 allowlist 固定为：
  - `title`
  - `body`
  - `moderation_status`
  - `editable`
- violating 自有文章只返回：
  - `moderation_status="restricted"`
  - `editable=true`
- ETag 仅通过响应 Header 提供；body 不复制 ID 或 version。
- 作者响应明确禁止：
  - `violation_state`
  - `violation_category`
  - `hidden_reasons`
  - `publication_state`
  - `orphaned_from_user_id`
  - 作者账号状态
  - deletion 内部状态
- 管理员治理详情继续返回完整 violation category、hidden reasons、归属和版本。
- 普通成员对 violating、隐藏、pending 或不存在文章继续统一 `404 article.unavailable`。
- 作者 PUT 只更新 title/body/updated_at/version；violation/category、publication、author/source 和 deletion state 均保持不变，修改后成员仍不可见。

原 finding 的“作者端提前披露完整治理上下文”触发路径已消除。

## 4. 新 findings

未发现新增 CRITICAL、HIGH 或 MEDIUM finding。

## 5. 回归与静态验证

- 8 条主责 US 映射完整。
- design 与 article boundary r09 的 S1～S16 语义一致。
- 成员、作者和管理员使用独立响应投影；未发现 serializer 字段串用。
- 作者修改 violating 自有文章的能力保持，治理约束不变。
- 普通非作者返回 404；管理员非作者 PUT 返回 403；管理员作为作者时仅按作者身份处理。
- Article Deletion Journal、分页扫描、TransferNotice recipient 状态机和 identity ports 未发现本轮回归。
- design 26 个 PlantUML 块静态起止成对，未发现字面 `\n` 或 `linetype ortho`。
- stories 使用文本依赖图；人工核对未发现环或依赖漂移。
- design 中 19 个本地链接均存在。
- 本机无可用 PlantUML CLI/JAR，未执行 PNG 实渲染。
- 当前无代码、schema、迁移或测试；未运行不存在的构建、lint、单元、集成或 E2E。

## 6. 最终结论

**PASS。**

- `AM-DES-R02-N001`：closed。
- article review r01/r02 产生的全部 findings 已关闭。
- 本轮未发现新增阻断 finding。

该 PASS 仅表示固定 r03 设计对象通过本次独立静态复审，不表示：

- AM-Q1/Q2 等已声明运行或业务输入已经拍板；
- design 自动从 `draft` 转为 `implementable`；
- 代码、迁移、Journal、真实恢复演练、SMTP 或 E2E 已通过；
- 用户/Lead 已完成成果接受或实施授权。

后续由 Lead 核对固定对象、实际确认状态和适用 Gate，再决定任务拆解或补充未决输入。
