# article-management-S2.1 R01 返工规划独立评审

- 日期：2026-10-25
- objectType：`design / planning contract`
- Reviewer：非规划作者 Reviewer
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支：`main`
- 固定 HEAD：`6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 被审范围：逐一评审 Tasks SSOT 与 Batch Plan，并以固定代码 Review R01、现役设计/测试依据和有界真实消费者核其可实施性；不是代码 Review。

## 1. 固定对象与漂移检查

| 对象 | 固定 SHA256 | 开始前 | 落盘前 | 落盘后 |
|---|---|---|---|---|
| `.add/project/knowledge/tasks/tasks.md` | `EE094632F55F3250714221A4DD1ECAC268CDD4CA1ADAF91EE2005EB1947C519F` | MATCH | MATCH | MATCH |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | `9D5CD0A9D7F09DE93875AC1465175428FB9AC726ABCF92CE9A2E6FFC22A55641` | MATCH | MATCH | MATCH |
| `.spec/wip/article-management/review-code-s2.1-r01.md` | `36BB65973CDC5998567B57B8C10AED1F6B65E114DEB5560414B19179540EF129` | MATCH | MATCH | MATCH |

开始及落盘前同时确认：分支、HEAD 符合固定基线，报告路径原先不存在，staged 为空，`han-server.exe` 存在。未发生对象漂移或报告冲突。

## 2. 评审依据与两轮实际检查范围

已读取 Reviewer Role、`add-code-review`、code-review checklist、abstraction principles、design-doc writing style；并按规划对象读取 Task-Planner Role、`add-task-decompose`、`add-batch-plan`。业务依据包括固定三对象、article design §2.4.1/§3.3.3/公共错误表、Story 2、A-READ-01～06，以及判断充分性所需的四个拟写文件和直接只读消费者：

- 拟写：`internal/article/query_service.go`、`internal/article/query_test.go`、`cmd/han-server/main.go`、`cmd/han-server/main_test.go`
- 只读：`internal/article/handler.go`、`internal/identity/article_reader.go`

### Round 1：Tasks SSOT 与 Batch Plan 合同一致性

实际检查：

1. 四标签 gap-map、复用原 `article-management-S2.1`、新增 task=0；
2. checkbox、开始日期、requires、DAG 与三态是否被返工规划意外改写；
3. 四文件唯一写集、只读集合、其他 active lane 写集与 `main.go/main_test.go` 独占窗口；
4. F002 先行、F001 等批准输入的内部 Gate；
5. 专项普通/race、`vet`、`build`、四文件 `gofmt`、scoped diff-check、Manifest 七项及独立复审/Lead Gate；
6. 第五文件、第二 production root、新依赖/migration、公共合同或只读文件变化时的停止条件；
7. S4.6 CLOSED 历史、当前 S2.1 active 状态及其顺序释放关系是否混写。

结论：

- Tasks 与 Plan 一致将 Story S02 标为“需改”，复用且仅复用原 S2.1，新增 task=0；R01 F001/F002 保持 OPEN，局部旧 PASS 未被冒充为当前关闭证据。
- S2.1 仍为 active；requires/DAG checkbox/日期未变。其 direct requires `project-bootstrap-B0.1`、`identity-auth-S8.1`、`article-management-S1.2`、`article-management-S1.3` 均为 CLOSED。
- 实际三态为 `57 / 26 completed / 9 active / 22 todo`。九个 active 与 Tasks/Plan 声明一致：`identity-auth-S3.3`、`identity-auth-S3.5`、`identity-auth-S5.3`、`identity-auth-S9.1`、`identity-auth-S10.1`、`article-management-S2.1`、`article-management-S4.2`、`article-management-S4.3`、`article-management-S4.5`。
- S4.6 保持 CLOSED；其历史完成事实未被重开或覆盖，只顺序释放唯一 production composition/router 的 `main.go/main_test.go` 给 S2.1 有界返工。
- 其他 active lane 的已声明 production 写集与四文件无交叉。S2.1 占用 `main.go/main_test.go` 期间必须排除其他 server 写者；未知写者或交叉即停止。
- 验收与关闭链覆盖定向普通/race、静态/构建、格式和 diff、Manifest、非作者代码复审、同 revision 哈希/status/staged/越界 Lead Gate；专项命令与 Manifest 不互相替代。

### Round 2：真实 authority、实现边界与消费者

实际检查：

1. `ARTICLE-S2.1-REV-R01-F001` 的 production composition/router owner、共享依赖和关闭证据；
2. `ARTICLE-S2.1-REV-R01-F002` 的 repository/Identity 两类 503 taxonomy、局部 500 保留边界和 HTTP 消费者；
3. `QueryConfig` 的现役 authority、AM-Q2 未决范围及 Dev 是否可能被迫虚构外部合同；
4. GET list/detail 与既有 DELETE 的同路由族共存、405 `Allow`、Trace/Origin/Session/Principal 安全链；
5. 共享 DB pool、Identity reader 及唯一 production composition root；
6. `handler.go`、Identity reader 与 S4.6 其余文件维持只读的可行性。

结论：

- **F002 有唯一修复边界且可先行。** 当前 query service 把 repository 与 Identity adapter 故障汇入 500。可仅在 `query_service.go/query_test.go` 的消费边界分别稳定归类为 `503 platform.database_unavailable` 与 `503 platform.dependency_unavailable`，同时让本地不变量/编码等内部缺陷继续为 500；现有 handler 可只读复用以验证 list/detail HTTP envelope 和不泄露 cause。无需修改 `handler.go` 或 Identity reader。
- **F001 的四文件边界最小且充分。** `main.go` 已承载唯一 production builder/router、安全链与共享 pool。可在此复用共享 pool 构造 Article query repository、Identity reader、query service/handler，并把 GET list/detail 合并进现有唯一 router；`main_test.go` 足以覆盖 composition、共享实例、生产可达、路由共存、405 `Allow` 和安全链。无需第二 composition root、第五文件或重开 S4.6。
- 详情路径必须让 GET 与既有 DELETE 在同一真实路由族共存；不支持的方法应返回反映该路径完整受支持方法集合的 `Allow`，不得继续把详情 GET 误判为仅 DELETE 路由的 405。列表路径同样不得绕过现役安全链。
- **QueryConfig 的暂停条件准确且必要。** 现役设计只确定 limit 的缺省值/最大值来自配置，部署值来自环境或受控登记；AM-Q2 数值仍为 TBD。当前没有获批准的精确来源选择、键名和启动时结构性校验合同。规划正确要求 design owner/Lead 先提供可追溯批准输入，并禁止把测试 fixture `2/2/3` 变成生产默认或由 Dev 虚构 AM-Q2。
- 该缺口无需先扩大本返工写集或由 Dev 修改设计；但在权威批准输入形成前，F001 不得获得 START。F002 的定向 START 不代表 F001 或整卡已解除阻塞。

## 3. Findings

**无新增 Planning finding。**

限制说明：

- 本结论只说明当前 Tasks/Plan 已正确隔离已知缺口，并提供安全的分阶段实施合同。
- 本评审不关闭代码 Review R01 的 `ARTICLE-S2.1-REV-R01-F001`（HIGH）或 `ARTICLE-S2.1-REV-R01-F002`（MEDIUM）。
- 因无新增 finding，未分配 `ARTICLE-S2.1-PLAN-R01-Fnnn`。

## 4. 待决定 / 前置输入

F001 开始前，design owner/Lead 必须提供并记录可追溯批准的 production `QueryConfig` 输入：

1. 外部配置来源；
2. 精确键名；
3. 启动时结构性校验合同，包括缺失、非法、相互约束或越界时的处理；
4. 若具体数值仍属 AM-Q2 未决，不得由实施者、fixture 或本 Planning Review 代填。

这是一项被规划正确阻断的前置决定，不是授权 Dev 自行设计的实现细节。

## 5. 未验证风险

- 本 Planning Review 未运行 Go 测试。
- 未启动真实 `han-server`，未运行真实 PostgreSQL、部署或 E2E。
- 未验证 AM-Q2 数值、容量、查询计划或 P95。
- 未验证返工后的代码、race、Manifest 或 production readiness。
- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 共享工作树存在他人改动；本结论依赖实施/复审时再次确认四文件 revision、未知写者和跨 lane 写集。

## 6. Planning Review Gate

**Planning Review Gate：PASS（分阶段、有条件）**

允许的下一步仅为：

1. Lead 在固定基线、四文件无未知写者/交叉且 staged 边界可解释的前提下，定向授权 F002 进入 TDD；
2. F001 保持 BLOCKED，直到第 4 节 QueryConfig authority 输入获批并由 Lead 现场复核；
3. 获批后仍只能在同一四文件和唯一 production composition/router 内实施 F001；
4. 任一第五写文件、只读文件修改、新依赖、migration、公共合同/architecture/ADR 变化、第二 root、跨卡写集交叉或配置 authority 冲突，立即停止受影响实施并返回 Task-Planner/Lead。

PASS 不表示整卡可关闭，也不外推为代码、PostgreSQL、部署、E2E、容量或 production readiness PASS。

## 7. 复审与关闭条件

S2.1 关闭前至少需要：

1. 已批准且可追溯的 QueryConfig 来源、键名和结构性校验合同；
2. 四文件同一 revision 完成 F002 两类 503 与 F001 唯一 production composition/router 修订；
3. GET list/detail 与既有 DELETE 共存，405 `Allow`、安全链、共享 pool/Identity reader 和唯一 root 均有定向测试；
4. 原代码 Reviewer 或有记录的替代非作者 Reviewer 逐项关闭 R01 F001/F002，且无未处置新增 finding；
5. Lead 在同一 revision 执行并核对规划登记的 article/server 定向普通/race、`go vet ./internal/article ./cmd/han-server`、`go build ./cmd/han-server`、四文件 `gofmt -l`、scoped diff-check及适用 Manifest 固定命令；
6. Lead 现场核固定对象/实现哈希、status、diff、staged、未知写者、四文件唯一写集和越界；确认未混入 S4.2 等其他 lane 修订；
7. 保持 S4.6 历史 CLOSED，不以本次复用污染其历史 closure，也不把 S2.1 当前 active 写成已完成。

## 8. 本报告写入边界

- 仅新增本报告；未修改 Tasks、Plan、代码、设计或其他报告。
- 未 stage、commit、push、切分支、清理或删除。
- 写后仍为固定分支/HEAD；三个固定对象 SHA256 均 MATCH。
- 本报告路径状态为唯一新增 `??`；staged 为空；`git diff --check -- <本报告>` 退出码为 0，逐行补充检查无尾随空白且有文件末尾换行。
- 共享工作树另有 16 个既存 tracked change，本轮未触碰；`han-server.exe` 保留。
- 本报告完成后停止，等待 Lead 现场复核。
