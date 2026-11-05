# S1.5 整卡关闭评估复审 r01（非作者独立 Reviewer，只读）

- 评审性质：AI 代理非作者独立复审；按 `add-code-review` 的 Round 1/2 口径做整卡关闭评估，不做逐行代码评审。
- 对象：`article-management-S1.5`（唯一活跃卡）及其 10 项关闭条件、12 个 requires、子门禁证据与评审链。
- 固定工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 工作树状态：开工时已存在大量既有 modified/untracked 文件（含 `tasks.md`、`go.mod`、`.spec/wip/**`、`cmd/`、`internal/`、`test/` 等）。本次评审未修改源码、测试、tasks、Plan、ADR 或任何既有证据；未做 Git 写、删除、清理或下级委派。
- 唯一写入：本目录 `report.md` 与 `verdict.md`。
- 总结论：**CHANGES REQUIRED**。实质代码/测试证据总体成立，但 `S15-RE-F008` 的“R02 独立复审 CLOSED”缺少独立复审产物，且现有唯一复审记录明确写的是 R01 `NOT CLOSED`；这使关闭条件第 1、9、10 项不能按“子集 PASS 相加 = 整卡 PASS”处理。

## 1. 审查边界与方法

- 输入契约：对象为 S1.5 整卡关闭评估，不是全仓代码评审；事实、用户决定与假设分开；不把“文件存在”或“作者说已审”当作内容或批准证据。
- Round 1：固定三份对账输入的字节与 SHA256；逐项检查 10 个关闭条件与 12 个 requires；读取 `cmd/han-server/main.go`、`main_test.go`、四个 contract 测试与关键测试入口。
- Round 2：沿真实消费者检查子门禁证据链（contract top、16 端点 root、ACL、Audit SQL、模拟收口、producer 真实层、S8.2/Case18），并核对“子集 PASS 相加是否构成整卡 PASS”。
- 只读执行：`git rev-parse`/`branch`/`status`、`Get-FileHash`、`Get-Content`、`Select-String`、离线解析保存的 `-json` 日志、`gofmt -l`、`go vet -tags=contract ./...`、一次 `go test -tags=contract ./... -count=1`。
- 未执行：真实 PostgreSQL/ACL/Audit/producer 复跑、真实部署 E2E、真实 Journal/keyring、`go test -race` 全仓、`go build ./...`。这些未执行项在 §8 说明。

## 2. 对账输入实核

| 文件 | 记录字节 | 实算字节 | 记录 SHA256 | 实算 SHA256 | 结果 |
|---|---:|---:|---|---|---|
| `.spec/wip/article-management/verification-s1.5-gap-r01.md` | 3,730 | 3,730 | `0E975D219B5EA70A9301BE295BB1225CDFEC5EA298C2115D613DA07F26DA1F28` | 同左 | 一致 |
| `.spec/wip/article-management/verification-s1.5.md` | 47,623 | 47,623 | `2776C674B88B32F8456DE1D7C0D8FB985B8B6C698710DCEDF1124594CEA6FEBB` | 同左 | 一致 |
| `.add/project/knowledge/tasks/tasks.md` | 982,035 | 982,035 | `716795B3F1740415CFAE26640520C32AAC5F89AA4C3BBDCAD7341C4641A39EC4` | 同左 | 一致 |

三份输入均与题面一致。`verification-s1.5.md` 当前为 198 行；`tasks.md` 当前 S1.5 卡从 `:1236` 开始，S8.2 卡从 `:1925` 开始。

## 3. 10 项逐条判定

| # | 关闭条件 | 判定 | 依据摘要 |
|---:|---|---|---|
| 1 | 16 endpoint 真实 method/path 与适用安全链（Lorentz PASS WITH MINOR；`S15-RE-F008` 复审 CLOSED） | ⚠️ | 16 个 method/path 与 main.go 分支实读成立；Lorentz 报告 21,470 B / `DA2BD8D7…` PASS WITH MINOR 成立；F008 代码修复与 34 叶子日志实核成立；但“R02 CLOSED”没有独立复审产物，唯一复审文件仍是 R01 `NOT CLOSED`，见 F001。 |
| 2 | 三投影 allowlist | ✅ | `projection_contract_test.go` 对成员 list 5 键、成员 detail 8 键、作者 4 键、Admin 12 键做精确键集与逐值 oracle；contract suite 普通/race 各 286 run / 287 pass / 0 fail / 0 skip。 |
| 3 | 公共错误（REST contract top） | ✅ | `rest_contract_test.go:653-683` 的 19 个公共错误 case 与 design §3.3.1 `:1060-1080` 的 19 行逐项对应；REST top 在 contract suite 中 PASS。 |
| 4 | S1–S16 内部契约（internal ports contract top；四文件组件 R02） | ✅ | `internal_ports_test.go:28-37` 的 8 个 top 组覆盖 Principal/Reader/Canonical/Cleanup/Gate/Error/Idempotency/Recovery DTO；test-plan §4.7.1 `:552-571` 把 16 个接缝逐项映射到组件断言或固定证据；组件 R02 10,229 B / `52298FED…` PASS。 |
| 5 | Ready20 / Unavailable6（producer contract + 真实层） | ✅ | `replay_status_producer_test.go:130-230` 断言 Ready 20 键、Unavailable 6 键、7 个 Unavailable 变体与 6 个坏 evidence 变体；producer 真实层双轮各 58 pass / 0 fail / 0 skip，独立评审 PASS WITH MINOR，Lead 已接收。真实层主 leaf 集中在 Publish/ReadCurrent，`ReadStatus` 在真实 integration 测试内被调用但未单列为 Ready20/Unavailable6 leaf（见 §8 边界）。 |
| 6 | 适用真实层证据（应用角色 ACL R03、真实 Audit SQL、模拟收口、producer 真实层） | ✅ | ACL R03 index 22,037 B / `8ED9FF3E…`，12/12 普通与 race，Lead Gate log 298 B / `D21DB1F1…`；Audit SQL review 19,125 B / `DF4D4829…` PASS、implementation/review index `4B93DD53…`/`A4CF2B92…`；模拟收口 index 12,332 B / `3E737B8C…` PASS WITH MINOR；producer 真实层 index 19,567 B / `F840BC2C…`、review `B51E2F55…` PASS WITH MINOR + Lead 接收。 |
| 7 | Admin2 已落实（S8.2 已关闭；Case18 F001–F007 全关，F002 按 ADR-0034、F007 按 ADR-0035 为范围决策） | ✅ | `tasks.md:1925` 为 `[x]`，`:1926` 记录 2026-11-05 关闭；Case18 F003/F004 由 Copernicus 复审 CLOSED（report 24,267 B / `74D4054F…`、verdict 8,900 B / `63F2B1DA…`）；ADR-0034 在 `decisions/content.md:241-276`、ADR-0035 在 `:279-300`，均标“已接受”。注：`verification-s1.5.md:149` 仍把 S8.2 写成 `[-]`，是未同步的历史行（见 F002），不影响 tasks 的权威 `[x]`。 |
| 8 | 普通/race 命中（各 286 run / 287 pass / 0 fail / 0 skip） | ✅ | 离线解析 `contract-normal-json.log` 与 `contract-race-json.log`，均 run=286、pass=287、fail=0、skip=0；四个现役 contract 文件哈希与 index 绑定一致。独立一次 `go test -tags=contract ./... -count=1` exit 0（29.514s）。 |
| 9 | 独立非作者 Review / 必要复审是否覆盖各子集 | ⚠️ | Lorentz（16 端点）、Erdos（组件 R02 / Audit SQL）、Confucius（ACL R03）、模拟收口 Reviewer、Halley/Carver（producer）、Copernicus（Case18 F003/F004）均有产物；唯独 F008 的“R02 复审”只有 tasks/verification/gap 文档中的一句话，没有独立复审文件。 |
| 10 | 是否存在“子集 PASS 相加 ≠ 整卡 PASS”的遗漏项 | ❌ | 存在：`verification-s1.5-gap-r01.md:48` 称“只差整卡级复审 + 接收登记 + Lead Gate”，但 F008 的 R02 独立复审本身也是缺口；该缺口未被计入。其余子集相加基本覆盖 1–8 项；第 9/10 项因 F008 不能闭合。 |

## 4. 12 个 requires 实核

逐行核对 `tasks.md` 的 checkbox（不是只转述 gap 文档）：

| # | requires | tasks.md 行 | 实际 checkbox | 备注 |
|---:|---|---:|---|---|
| 1 | `project-bootstrap-B0.1` | 63 | `[x]` | 完成@2026-10-21 |
| 2 | `identity-auth-S1.5` | 259 | `[x]` | 完成@2026-10-27 |
| 3 | `article-management-S1.3` | 1201 | `[x]` | 完成@2026-10-23 |
| 4 | `article-management-S2.1` | 1364 | `[x]` | 完成@2026-10-25 |
| 5 | `article-management-S3.1` | 1495 | `[x]` | 完成@2026-10-27 |
| 6 | `article-management-S4.2` | 1542 | `[x]` | 完成@2026-10-27 |
| 7 | `article-management-S5.1` | 1739 | `[x]` | 完成@2026-10-27 |
| 8 | `article-management-S6.2` | 1783 | `[x]` | 完成@2026-10-27 |
| 9 | `article-management-S7.1` | 1826 | `[x]` | 完成@2026-10-27 |
| 10 | `article-management-S8.1` | 1895 | `[x]` | 完成@2026-10-27 |
| 11 | `article-management-S1.6` | 1342 | `[x]` | 完成@2026-10-27 |
| 12 | `article-management-S8.2` | 1925 | `[x]` | 完成@2026-11-05 |

**实核结论：12/12 均为 `[x]`，与 gap 文档一致。** 本复审不重新打开这些 requires 的既有内容；`identity-auth-S1.5` 的 checkbox 为真，但其自身评审口径不在本卡重审范围。

## 5. 关键证据抽验（文件 + SHA256 + 字节）

除三份输入外，本次实际抽验了以下关键证据；均与文档声明一致（除 F001 指出的“复审文件内容与声明不符”）：

| 证据 | 实算字节 | 实算 SHA256 | 对应声明 |
|---|---:|---|---|
| `evidence-s1.5/contract-suite-r01/index.json` | 5,418 | `5477B1F56DDCFAB5F8B3787902B772C2B3F62A8A84B25F58C49705DA03F8AC93` | 一致 |
| `evidence-s1.5/root-16-endpoint-r01/index.json` | 24,714 | `E039B1F15A33B33C80F4CB03C5A94EF31A37F8BE66127BA7F7810E84C7DA4AEE` | 一致 |
| `evidence-s1.5/root-16-endpoint-r01/16-endpoint-map.md` | 16,117 | `67A01D854BE79FC9E0836F3A7FC3F335EBAE9B0B77B7CBC87773749841C23C7D` | 一致 |
| `review-code-s1.5-root-endpoints-r01.md` | 21,470 | `DA2BD8D786C13A951343C095EBF48BEA2A14973322B9F0FB53CB5ABE375D34B4` | 一致 |
| `evidence-s1.5/acl-r03/index.json` | 22,037 | `8ED9FF3E9F8E0CB22691A32F1B394C40AD4A6FDFFCB9E0179525777379DDFCA5` | 一致 |
| `evidence-s1.5/acl-r03/lead-gate.log` | 298 | `D21DB1F19903FA2D25685A4ADE9C58BE1FF89C4D47E9023FB7B0C269AE608155` | 一致 |
| `evidence-s1.5/audit-sql-implementation-r01/index.json` | 139,486 | `4B93DD5351EDF960C347502148E234521A103932A911E7B8FB844860328496BF` | 一致 |
| `evidence-s1.5/audit-sql-review-r01/index.json` | 34,674 | `A4CF2B928AFBF9DF6F9466578A4D1824CA3EBEBEA7BBE1C301CE748E17A7A168` | 一致 |
| `evidence-s1.5/sim-closeout-r01/index.json` | 12,332 | `3E737B8CD92A4585B48B011975DB24D4661DEDD9527625016F89CA9C38C91DDA` | 一致 |
| `evidence-s1.5/producer-real-r01/index.json` | 19,567 | `F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B` | 一致 |
| `evidence-s1.5/producer-real-r01/report.md` | 8,267 | `1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749` | 一致（旧 `A9A4ECBB…` 为历史指针） |
| `evidence-s1.5/producer-real-review-r01/verdict.md` | 796 | `812A56FE8B80E9789BBBF3C41DC5C168CB0ED335E6CD482DD4D6F958CB79CB6E` | 一致 |
| `evidence-s1.5/f008-allow-assertions-r02/index.json` | 5,564 | `410A65ADD91FCDDD3D30E54090FC0EB64890CC73990E0F55C88B13DEAC0FCC6E` | 一致 |
| `evidence-s1.5/f008-allow-assertions-r02/report.md` | 5,808 | `400E593A7A6B8BDEEA034731375B21AEFC283FA2E74818001C5B0393D5C63681` | 一致 |
| `review-code-s1.5-f008-producer-r01.md` | 5,067 | `8437743308F17FA2187F8D1F2A232030591ADC6F8D13A18BA42B92E0AAA99DDD` | 一致；但内容是 R01 `NOT CLOSED`，不是 R02 `CLOSED` |
| `review-code-s1.5-components-r02.md` | 10,229 | `52298FEDD5F13E9A9017334C1FEA72A1BD72C9A9658A2384660D9075D6F59933` | 一致 |
| `evidence-s8.2/case18-reconcile-r01/review-r01/report.md` | 24,267 | `74D4054F3D7618D110302C964458768B63977971F2A53BBE36D601564517A08D` | 一致 |
| `evidence-s8.2/case18-reconcile-r01/review-r01/verdict.md` | 8,900 | `63F2B1DA0CF21606DB0428BCC15789A47FF841413216BCE89DD1FADC9B3EB8B4` | 一致 |

`cmd/han-server/main.go` 实算 173,317 B / `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727`；`cmd/han-server/main_test.go` 实算 400,312 B / `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC`。两者与 F008 r02 和 Case18 当前证据声明一致。

## 6. 逐项详细依据

### 6.1 16 endpoint 与安全链

`cmd/han-server/main.go:3542` 的 `newLoginBusinessRouterWithTraceAndRecovery` 是唯一生产业务 API router。实读 `:3604-3658` 可数出 16 个 method/path：

1. `POST /api/v1/admin/articles/{id}/violation`（:3604）
2. `POST /api/v1/admin/articles/{id}/restore`（:3608）
3. `GET /api/v1/admin/orphan-articles`（:3612）
4. `POST /api/v1/admin/orphan-articles/{id}/publish`（:3614）
5. `POST /api/v1/admin/orphan-articles/{id}/transfer`（:3616）
6. `GET /api/v1/articles`（:3618）
7. `POST /api/v1/articles`（:3618）
8. `GET /api/v1/admin/articles`（:3626）
9. `GET /api/v1/users/me/articles/{id}`（:3628）
10. `GET /api/v1/articles/{id}`（:3632）
11. `PUT /api/v1/articles/{id}`（:3632）
12. `DELETE /api/v1/articles/{id}`（:3632）
13. `GET /api/v1/admin/articles/{id}`（:3644）
14. `DELETE /api/v1/admin/articles/{id}`（:3644）
15. `GET /api/v1/admin/orphan-articles/{id}`（:3652）
16. `DELETE /api/v1/admin/orphan-articles/{id}`（:3652）

`main.go:3668-3675` 在 method 不符时设置 `Allow` 与 `nosniff` 后立即 405；`:3686-3698` 在 handler 为空时 404，否则经 recovery guard 进入 handler；`:3700-3704` 把 `platform.TraceMiddleware` 包在最外层。此结构支持“404/405 先于安全链、非 probe 路由 trace 最外层、guard 先于业务 handler”的声明。

Lorentz 报告 `review-code-s1.5-root-endpoints-r01.md`（21,470 B / `DA2BD8D7…`）结论 PASS WITH MINOR，HIGH 0 / MEDIUM 0 / LOW 8；其中 7 条文档类 LOW 已处置，F008 是唯一保留项。`16-endpoint-map.md` 仍写 13/16 有 `Allow` 值断言，是 F008 修复前的快照；F008 r02 修复后 `main_test.go:3402-3406` 已对三条 orphan 路由新增 `Allow` 精确断言与 `events == [trace]`/`resolver.calls == 0` 断言，`mainOrphanAllow` 在 `:3411` 起且未知 action 走 `t.Fatalf`。F008 logs：normal 35 pass（1 top + 34 leaves）、race 36 pass（2 top + 34 leaves）、0 fail / 0 skip；代码层修复成立。

但 `verification-s1.5.md:147` 与 `verification-s1.5-gap-r01.md:28` 声称“独立复审 R02 判 CLOSED”，并把 `review-code-s1.5-f008-producer-r01.md`（5,067 B / `84377433…`）作为“复审记录”。该文件第 11 行明确写 `S15-RE-F008 = NOT CLOSED`，第 22 行写 Q5 `NOT CLOSED`，第 42 行写“提请同一非作者复审 R02”。`evidence-s1.5/f008-allow-assertions-r02/index.json` 的 `boundaries` 也明确写“not yet independently re-reviewed after this revision; S15-RE-F008 stays open until re-review PASS”；`f008-allow-assertions-r02/report.md:64` 同样写“本修订待非作者独立复审”。全目录未找到任何独立的 R02 review 文件（`rg` 只命中 R01 文件、tasks、verification、gap 文档与一份 snapshot）。因此：**代码修复成立，但“R02 独立复审 CLOSED”这一关闭证据不成立**。

### 6.2 三投影 allowlist

`test/contract/article/projection_contract_test.go:46-56` 的 top 为 `TestArticleProjectionContract`，子测试 `MemberViews`/`AuthorView`/`AdminViews`/`AnonymousAndRejectedState`/`PrivacyCanaries`。`:151-163` 对成员 list 断言 `id,title,author,created_at,updated_at` 五键，对成员 detail 追加 `body,publication_state,violation_state` 共八键；`:165-179` 对作者断言 `title,body,moderation_status,editable` 四键；`:181-207` 对 Admin 断言 `id,title,body,author,publication_state,violation_state,violation_category,hidden_reasons,orphaned_from_user_id,version,created_at,updated_at` 十二键。`jsonObject` 用排序后的键集合做精确比较，不是只检查存在性；`projectionPrivacy` 还检查正文 canary 与内部键不泄露。contract suite 普通/race 各 286 run / 287 pass / 0 fail / 0 skip（离线解析两份 262,747 B / 262,765 B 的 JSON 日志），与 index 的 `action_counts` 一致。

### 6.3 公共错误

`test/contract/article/rest_contract_test.go:271-282` 的 REST top 含 `Endpoints`、`WireAndStrictJSON`、`Preconditions`、`PublicErrors`、`AdminRawQueryAndStatePriority`、`ResponseWriteFailure`。`:653-683` 的 `publicErrorCases` 共 19 条，逐条给出 code、HTTP status、retry 分类、cause 与稳定 message；与 design §3.3.1 `:1060-1080` 的 19 行逐项对应（400/401/403/404/409/412/422/428/429/503/500）。`restPublicErrors` 还断言 envelope 键集、trace、field_errors 与私有 cause/PII canary 不泄露。REST top 在 contract suite 中 PASS。

### 6.4 S1–S16 内部契约

`test/contract/article/internal_ports_test.go:28-37` 的 top 为 `TestArticleInternalPortsContract`，8 个组：`PrincipalAndRoleBoundary`、`AuthorReaderContract`、`CanonicalAdapter`、`CleanupAndSnapshotOrder`、`ProcessingGateAndPurpose`、`ErrorRegistryAndRetry`、`NoKeyedIdempotency`、`RecoveryDTOBoundary`。test-plan §4.7.1 `:552-571` 把 S1–S16 每个接缝逐项映射到“组件断言或固定证据消费”，并明确 S13/S15/S16 的未证明边界；因此“S1–S16 内部契约”不是 16 个同名测试，而是 8 个组件组 + 分层证据消费，声明与测试结构一致。组件 R02（`review-code-s1.5-components-r02.md`，10,229 B / `52298FED…`）判 F001 CLOSED、有界 CodeReviewGate PASS、无新增 finding；其绑定 producer 旧哈希 `B519DDE1…` 的边界已由文档披露，现役 producer `6EB4A677…` 由 S4.4 评审覆盖。

### 6.5 Ready20 / Unavailable6

`replay_status_producer_test.go:73-78` 的 top 含 `ReadStatusReady20`、`ReadStatusUnavailable6`、`InvalidEvidenceFailsClosed`、`SharedAuthorityAndConsumerMutations`。`readyKeys()` 恰 20 键；`producerReady` 逐字段与共享 golden 比较，并验证 canonical digest 不含 `evidence_ref`/`completed_at`。`producerUnavailable` 对 7 个状态/错误变体断言 6 键 `evidence_version,run_id,backup_ref,namespace,status,reason_code` 与对应 reason，且不计算 digest、不泄露 cause；`producerBadEvidence` 对 6 个坏 evidence 变体断言 `failed/evidence_unreadable`。真实层消费 B13d 外部卡：`required-pg-normal`/`required-pg-race` 各 58 pass / 0 fail / 0 skip，无 `-overlay`，PG 18.6，独立评审 PASS WITH MINOR 且 Lead 已接收。边界：真实层的主 leaf 映射是 `Publish`/`ReadCurrent`；`ReadStatus` 虽在真实 integration 测试中被调用（`replay_publication_integration_test.go:662/1307`），但未单列为 Ready20/Unavailable6 的真实 leaf。这不影响本卡“producer contract + 真实 Publish/ReadCurrent”的组合声明，但不能外推为“Ready20/Unavailable6 在真实 PG 上独立验证”。

### 6.6 适用真实层证据

- **ACL R03**：`acl-r03/index.json` 22,037 B / `8ED9FF3E…`；真实 PG 18.6，普通与 `-race` 各 12/12 子测试，0 fail，`DATA RACE` false；`lead-gate.log` 298 B / `D21DB1F1…` 记录 build/vet/vet-integration/gofmt/unit/pg-normal/pg-race 全 exit 0。R03 报告 `review-code-s1.5-acl-r03.md` 23,870 B / `78D8AA23…` 判 PASS WITH MINOR，R01/R02 findings 全部 CLOSED，新增 3 条 LOW 仅证据卫生。
- **真实 Audit SQL**：`audit-sql-implementation-r01/index.json` 139,486 B / `4B93DD53…`、`audit-sql-review-r01/index.json` 34,674 B / `A4CF2B92…`；Erdos 独立报告 `review-code-s1.5-audit-sql-r01.md` 19,125 B / `DF4D4829…` 判有界 CodeReviewGate PASS、0 新 finding；普通/race 各 5 run / 5 pass / 4 leaf / 0 fail / 0 skip，四叶逐行命中；Lead 已实读并接收 SQL 子集。
- **模拟收口**：`sim-closeout-r01/index.json` 12,332 B / `3E737B8C…`；独立评审 PASS WITH MINOR，MEDIUM F001 以掩码合法 Trace 修复，LOW F002 补 authorization/origin 记录器与 guard-open 正控，F004 的并发写窗口如实保留为边界；`-count=25` 复跑通过。
- **producer 真实层**：`producer-real-r01/index.json` 19,567 B / `F840BC2C…`、`report.md` 8,267 B / `1B2FB3CF…`；双轮 58/0/0，10 源绑定，门禁空 DSN 即 Fatal、无 skip；Halley 独立评审 `B51E2F55…` PASS WITH MINOR，Lead 2026-11-01 接收；后续 r03 delta 由 Carver 判 PASS、F005 当前绑定 CLOSED。

### 6.7 Admin2 / S8.2 / Case18

`tasks.md:1925` 为 `[x] article-management-S8.2 ... 完成@2026-11-05`；`:1926` 记录 Lead Gate + 非作者独立复审，Case18 F001/F005/F006 已关，F002 按 ADR-0034、F007 按 ADR-0035 范围决策，F003/F004 由 Copernicus 复审 CLOSED。`evidence-s8.2/case18-reconcile-r01/review-r01/report.md` 24,267 B / `74D4054F…`、`verdict.md` 8,900 B / `63F2B1DA…` 实算一致；`decisions/content.md` 40,405 B / `DE442065…` 的 ADR-0034 `:241-276` 与 ADR-0035 `:279-300` 均写“已接受”，且明确把未验证边界与“非验证通过”分开。`tasks.md:55` 三态为 76 completed / 1 active / 9 todo，active 仅 S1.5。

### 6.8 普通/race 命中

两份保存的 contract JSON 日志各 262,747 B / 262,765 B；离线逐行解析 `Action` 事件：normal 与 race 均 `run=286, pass=287, fail=0, skip=0`。`contract-suite-r01/index.json` 5,418 B / `5477B1F5…` 的 `action_counts` 与实算一致；四个 contract 文件当前哈希 `CEBA105C…`、`DC6791D4…`、`D7585EE6…`、`6EB4A677…` 与 index 绑定一致。独立一次 `go test -tags=contract ./... -count=1` exit 0（29.514s），所有包 `ok`；这次全包运行不替代四 top 的逐项计数，计数来自保存的 JSON 日志。

### 6.9 独立评审覆盖

| 子集 | 独立评审 | 产物/结论 |
|---|---|---|
| contract 四文件组件 | Erdos R02 | `review-code-s1.5-components-r02.md` 10,229 B / `52298FED…` PASS，F001 CLOSED |
| 16 端点 root | Lorentz R01 | `review-code-s1.5-root-endpoints-r01.md` 21,470 B / `DA2BD8D7…` PASS WITH MINOR，7/8 LOW 处置，F008 开放 |
| ACL | Confucius R03 | `review-code-s1.5-acl-r03.md` 23,870 B / `78D8AA23…` PASS WITH MINOR，全部 R01/R02 findings CLOSED |
| Audit SQL | Erdos | `review-code-s1.5-audit-sql-r01.md` 19,125 B / `DF4D4829…` PASS，0 新 finding |
| 模拟收口 | 独立 Reviewer | `review-sim-closeout-r01.md` 19,870 B / `3BD91393…` PASS WITH MINOR；处置 3,182 B / `67F2DF1B…` |
| producer 真实层 | Halley R01 / Carver r03 | `producer-real-review-r01/verdict.md` 796 B / `812A56FE…` PASS WITH MINOR；r03 verdict 1,531 B / `E0C54693…` PASS |
| Case18 F003/F004 | Copernicus | `case18-reconcile-r01/review-r01/report.md` 24,267 B / `74D4054F…`、verdict 8,900 B / `63F2B1DA…` |
| F008 | 应有 R02 复审 | **缺失**：唯一文件是 R01 `NOT CLOSED`；tasks/verification/gap 文档中的 R02 CLOSED 断言没有独立产物 |

### 6.10 整卡组合

`verification-s1.5-gap-r01.md:28-37` 把 9/10 项标为已有证据，把第 10 项标为“整卡级组合复审与同 revision Gate 尚未执行”，`:48` 结论为“只差整卡级复审 + 接收登记 + Lead Gate”。本次整卡复审确认：第 1–8 项的子集证据总体可核；但第 9 项的 F008 R02 独立复审缺失，导致第 10 项不能以“整卡组合只需再做一次复审”收口。另，`verification-s1.5.md:149` 的 unknown 列表仍写 S8.2 `[-]`，与 `tasks.md:1925` 的 `[x]` 不一致（F002）；`16-endpoint-map.md`/`lead-verification.md` 仍是 F008 前的 13/16 快照（F003）。

## 7. Findings

### [MEDIUM] S15-WC-R01-F001：`S15-RE-F008` 的 R02 独立复审 CLOSED 没有可核产物，且被引用的文件实际是 R01 NOT CLOSED

- finding ID：`S15-WC-R01-F001`
- 对象/修订：S1.5 整卡关闭评估；`S15-RE-F008` 的复审证据链。
- 位置：
  - `verification-s1.5.md:147`（称“独立复审 R02 判 CLOSED”，链接 `review-code-s1.5-f008-producer-r01.md`）
  - `verification-s1.5-gap-r01.md:28`（同称 R02 CLOSED）
  - `tasks.md:1249`（行末追加“R02 复审：S15-RE-F008 CLOSED”，但仍引用 R01 文件）
  - `review-code-s1.5-f008-producer-r01.md:11/22/42`（明确 `NOT CLOSED`，要求 R02）
  - `evidence-s1.5/f008-allow-assertions-r02/index.json` 的 `boundaries`（“not yet independently re-reviewed”）
  - `evidence-s1.5/f008-allow-assertions-r02/report.md:64`（“待非作者独立复审”）
- 现状：代码修复本身成立。`main_test.go:3402-3406` 已注入 trace seam、精确断言 `Allow` 与 `events == [trace]`/`resolver.calls == 0`；`mainOrphanAllow` 在 `:3411` 起且未知 action `t.Fatalf`。F008 r02 日志 normal 35 pass（1 top + 34 leaves）、race 36 pass（2 top + 34 leaves）、0 fail / 0 skip。问题不在代码，而在“独立复审 CLOSED”的证据链：唯一被引用的 review 文件是 R01，且它明确判 `NOT CLOSED`；R02 的复审结论只出现在 tasks/verification/gap 文档的叙述中，没有独立 report/verdict、actualID、命令/exit/耗时或哈希。
- 触发：任何按 S1.5 关闭条件第 9 项“必要复审”核对证据的人，读取 `review-code-s1.5-f008-producer-r01.md` 都会得到 `NOT CLOSED`，与 `verification-s1.5.md:147` 的 `CLOSED` 冲突。
- 影响：S1.5 关闭条件第 1 项（F008 复审 CLOSED）和第 9 项（独立非作者 Review/必要复审）不能按现有证据判 PASS；第 10 项“子集 PASS 相加 = 整卡 PASS”因此不成立。即使代码修复正确，关闭评估也不能把“作者/Lead 的叙述”当作“独立复审产物”。
- 建议（最小修复，无需改代码/测试）：
  1. 由同一非作者 Reviewer 产出 `review-code-s1.5-f008-producer-r02.md`（或等价 report + verdict），记录 actualID、时间、命令/exit/耗时、34 leaves、0 fail/0 skip、结论 CLOSED，并给出 SHA256/字节；
  2. 将 `verification-s1.5.md:147`、`verification-s1.5-gap-r01.md:28`、`tasks.md:1249` 的指针改到该 R02 产物（或明确标注 R02 结论的转录来源与不可独立复核的边界）；
  3. 若无法产出该 R02 产物，则应把 `S15-RE-F008` 保持 OPEN，并撤回三处 CLOSED 断言，不能以“已修复”代替“已复审”。
- 验证：修复后重算 R02 report/verdict 的 SHA256/字节；确认文件正文含 `CLOSED`、34 leaves/0 fail/0 skip、actualID 与命令；确认三处引用均指向该文件，且 `f008-allow-assertions-r02/index.json` 的“not yet re-reviewed”边界被新产物取代或明确加注。

### [LOW] S15-WC-R01-F002：`verification-s1.5.md:149` 仍把 S8.2 写成 `[-]`

- finding ID：`S15-WC-R01-F002`
- 位置：`verification-s1.5.md:149`。
- 现状：该行“仍未满足或 unknown”列表写“`article-management-S8.2` 整卡（仍 `[-]`）”；但 `tasks.md:1925` 已是 `[x]`、`:1926` 记录 2026-11-05 关闭，`verification-s1.5-gap-r01.md:20` 也写 `[x]`（2026-11-05 关闭）。
- 影响：读者若只读 `verification-s1.5.md` 会误判 S8.2 未关闭；不影响 tasks 的权威状态，但会削弱整卡对账的一致性。
- 建议：在该行加“历史时点；S8.2 已于 2026-11-05 关闭，见 tasks:1925/1926 与 gap-r01:20”，不要改写历史结论或删除原文。
- 验证：确认新增注释不改变原行字节之外的历史记录，并确认 tasks/gap 的 `[x]` 仍一致。

### [LOW] S15-WC-R01-F003：16 端点快照仍写 13/16 Allow 断言，未指向 F008 r02

- finding ID：`S15-WC-R01-F003`
- 位置：`evidence-s1.5/root-16-endpoint-r01/16-endpoint-map.md` 的 §2 表与已知缺口段；`lead-verification.md` 的 `Allow` 段。
- 现状：两份文件绑定 F008 前的 `main_test.go 0E61F4A3…/399522 B`，写“13/16 端点另有可执行值断言”与“F008 开放”。当前 `main_test.go` 已是 `3720652E…/400312 B`，F008 r02 日志显示三条 orphan 路由已补断言，34 leaves PASS。`main.go` 未变，16 端点注册主张不受影响；但快照没有前向指针，容易被误读为当前仍只有 13/16。
- 建议：保留快照原文，追加“F008 已由 f008-allow-assertions-r02 修复，当前 main_test 3720652E…，见该目录；本文件为修复前快照”的指针；不要改写原表或旧日志。
- 验证：确认指针文件的 SHA256/字节；确认不把快照当现役测试文件。

### [INFO] S15-WC-R01-F004：`gofmt -l .` 的 exit 2 来自证据快照中的 `.go` 文件，不是源码格式问题

- 位置：仓库根执行 `gofmt -l .`。
- 现状：exit 2，输出 25 行，主要是 `.spec/wip/identity-auth/evidence-s10.5/.../read-excerpts/*.go` 等证据快照（内容不是合法 Go 包）与少量 probe 文件。
- 影响：不构成源码格式失败；`gofmt -l cmd internal test` exit 0、0 行、262 ms，说明真实源码树格式干净。
- 建议：不需要改源码；若要让仓库级 `gofmt -l .` 可用，应由证据整理流程把快照 `.go` 改为 `.go.txt` 或排除目录，但这是独立清理动作，本次不授权、不执行。
- 验证：`gofmt -l cmd internal test` exit 0；仓库级失败项逐条落在 `.spec/wip/**` 证据目录。

## 8. 实际执行命令与 exit/耗时

| 命令 | exit | 耗时 | 结果 |
|---|---:|---:|---|
| `git rev-parse HEAD` | 0 | — | `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |
| `git branch --show-current` | 0 | — | `main` |
| `git status --short` | 0 | — | 大量既有 modified/untracked；未由本评审写入 |
| `Get-FileHash` 三份输入 + 关键证据 | 0 | — | 见 §2/§5；除 F001 的“文件内容与声明不符”外均一致 |
| 离线解析 `contract-normal-json.log` / `contract-race-json.log` | 0 | — | 各 run=286/pass=287/fail=0/skip=0 |
| 离线解析 `f008-targeted-normal.log` / `f008-targeted-race.log` | 0 | — | 35/36 pass（1/2 top + 34 leaves），0 fail/0 skip |
| `gofmt -l .` | 2 | 2,094 ms | 25 行，均为 `.spec/wip/**` 证据快照/probe；非源码格式失败 |
| `gofmt -l cmd internal test` | 0 | 262 ms | 0 行 |
| `go vet -tags=contract ./...` | 0 | 11,256 ms | 无输出 |
| `go test -tags=contract ./... -count=1` | 0 | 29,514 ms | 所有包 `ok`（含 `han/test/contract/article 0.458s`） |

**NOT RUN**：真实 PostgreSQL/ACL/Audit/producer 复跑；`go test -race` 全仓；`go build ./...`；真实部署 E2E；真实 Journal/HMAC/keyring；真实旧备份恢复；`psql`/数据库连接；任何 Git 写。原因见 §9。

## 9. 保留限制

1. **真实 Journal / HMAC / keyring、旧备份恢复**：属于 S4.x/恢复主线责任；本卡明确不以此为关闭前置。未验证不等于通过；不得外推为恢复安全或 production readiness。
2. **真实部署 E2E / 双系统**：属于 S1.4 与 Case18 第 5 项；ADR-0035 已按测试项目范围决策“经决策不验证”，但只豁免该范围，不外扩到 S1.5 的 16 端点/投影/错误/内部契约。
3. **真实 keyring**：与真实 Journal/HMAC 同属恢复信任根边界；本卡消费的是 ACL/Audit/producer 的适用真实层，不声称 keyring 已真实部署验证。
4. **Ready20/Unavailable6 的真实层**：producer 真实层的主 leaf 是 `Publish`/`ReadCurrent`；`ReadStatus` 在真实 integration 测试中被调用，但未单列 Ready20/Unavailable6 leaf。不要把“producer contract 的 Ready20/Unavailable6”与“真实 PG 上独立验证 Ready20/Unavailable6”混为一谈。
5. **F008 的代码修复**：本复审独立读了 `main_test.go:3402-3406` 并解析了 34 叶子日志，认为实质修复成立；但本复审不是项目要求的那份 R02 复审产物，不能代替 F001 所要求的独立 review 文件。
6. **人的业务接收 / Lead Gate**：本次没有人的业务签收，也没有 Lead 同 revision Gate；本复审只回答“是否可进入关闭评估”，不代替 Lead Gate 或人的接收。
7. **工作树 revision 语义**：`HEAD 6da8abb` 的提交内容不含 Go 代码；`cmd/`、`internal/`、`test/` 等为未跟踪/已修改。所有“revision”结论都是工作树快照语义，不能由 `git checkout` 重建；`SOURCE_STABLE` 仅覆盖声明文件在运行前后不变，不证明整棵树未变。
8. **证据快照的 `.go` 文件**：仓库级 `gofmt -l .` 被证据目录中的非 Go 快照 `.go` 文件打断；这不影响 `cmd/internal/test` 的源码格式结论，但说明全仓工具命令需要排除证据目录。

## 10. 结论

- 12/12 requires 均为 `[x]`，与 gap 文档一致。
- 10 项关闭条件中，第 2–8 项有可核子集证据；第 1、9 项因 `S15-RE-F008` 的 R02 独立复审缺失而不能判 PASS；第 10 项因此不能以“整卡只差复审 + 接收 + Lead Gate”收口。
- 本复审的结论是 **CHANGES REQUIRED**：先补齐 F008 R02 的独立复审产物并更正三处指针（F001），同时处理 F002/F003 的文档漂移；不需要新增实现或测试。补齐后可由 Lead 做同 revision Gate、登记接收，再进入关闭评估。
- **AI 代理评审，非真人签字；不代替 Lead Gate 或人的业务接收。**