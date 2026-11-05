# S6.2 切片③接口/root 与 coordinator 独立 CodeReview R01

**slice3CodeReviewGate：PASS（本次六文件局部）。两轮审查 0 finding。** 不表示 PG 十组、完整 S6.2、真实部署/E2E 或旧纳秒 Journal 恢复兼容通过。

## 固定对象与范围

objectType `code`；原作者 Turing，本执行者非作者。工作区 `D:\AID\VSCode\Codex\Han` / `main` / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。固定目标：

| 文件 | SHA256 |
|---|---|
| `internal/article/query_service.go` | `9674130673CD05E8FA864B77528FDBF12C9AB8305AE6AD2EF7E2A85DB9136E7E` |
| `internal/article/handler.go` | `A0921B0178551A8EA51772ECF9823B50AA693DD3B450FDD347257B6F77CCC7E8` |
| `internal/article/orphan_disposition_test.go` | `1F8777D835D8C4363F67F498D5280C6EDA7BDAA3ED8F71B15AFB4E1B411338F5` |
| `cmd/han-server/main.go` | `4B9D83BA52E322E9E7A430F18A59DAA0FC651D4D4B21BEAFBA825E7E991A7569` |
| `cmd/han-server/main_test.go` | `1FB995EA3DE93A3F9AC2CC575D4AC86A9084592742EEC0C86F1ACF4E9CF28C9E` |
| `internal/article/deletion_coordinator.go` | `6EEA01FB40FBD6B237B4E02600122708404EE1AF59626971D4CE40D72DB98481` |

design r16 SHA256 `CB3CFA97CC1E4D7404B2DFD5E2A70B78B2FB790B79A3D64233F56355B3F22DCC`、test-plan r14 SHA256 `AA17CF63E5C05425FB59E5492048C8594DD563D23AF63AE97538EA5B72C5143C` 实核一致；复用①②冻结成果及本会话已读 Role/add-code-review/full 适用规则和当前 Workflow。

唯一新增文件为本报告；未修改代码、Tasks/Plan/design、Git，未清理或委派。James 活动中的独立 PG 文件不纳入固定对象，未读取该文件或执行 integration-tag/PG 测试。

## 两轮实际检查

Round 1：逐段比较原临时根 `C:\Users\J00010~1\AppData\Local\Temp\han-s62-4d7a5257f49745bc82ad754d1798d845` 下 query/handler/main/main_test/coordinator before-image 与目标；审查全部差异及必要上下文。coordinator 基线为 `307BEEDE0F62878D5FA39D81A69529792E3155735D776C565AC3E4AFD46D1C70`，仅修改 T1 时间取得一行。

局部 `orphan_disposition_test.go` 临时根无旧副本；在内存移除③新增测试/三个 import 后，SHA256 精确还原②固定 `D6EC42338586933CB141550563A3050A1A38012F897420821C5E6BC0E855685E`。因此本轮只把新增 355 行作为③增量，未重报②旧测试为新成果。

Round 2：沿生产 builder→共享 adapter→Article command/finalizer→Identity lifecycle/既有 native proof，及 router→Origin/HTTPAuthorizer→handler→query/command 核真实消费者。必要核查既有 JSON/ETag/cursor/projection、query repository 与配置 parser；未扩成全库或重审未冻结 PG 制品。

## 核心检查结论

| 检查点 | 证据与结论 |
|---|---|
| 固定 factory/native proof | `main.go:3349` 附近 bindCompletion 捕获真实 `*article.DispositionCompletionProof`、exact tx、reference 与当前 trace；bound verifier 对照 native proof 的独立 Reference 逐字段 DeepEqual，再调用其 VerifyCompletion，最后映射 Identity DTO。空 proof、另一 tx/wrapper、错误 trace、篡改 reference 或已结束 tx 不获授权；不是仅包装 public DTO。无 registry/token 或额外完成权威。 |
| 生产共享依赖与事务 | `main.go:2888` 只构造一次 lifecycle adapter，`:2976`/`:3034` 处同一实例注入 finalizer 和 command。adapter 字段只有 audit/ids/clock；锁与清理均传 caller tx，bound proof 回读也使用 caller tx，不在事务内向 pool 借第二连接。Gate 查询在转移 commit 后进行，不混入清理事务。 |
| 非空成功 proof 证据 | `main_test.go:368` 从真实 Article command 工作流生成 private proof，经生产 root factory、Identity lifecycle、实际 Repository/audit.Store 方法完成 publish/transfer 及一次释放审计；`:522` 经实际 finalizer 对 orphan/ordinary/violating 完成同类映射和去重。pgx.Tx 是模拟实现，证明局部协作而非真实 PG。逐字段、异 tx、trace、ended-tx 拒绝另有用例。 |
| 唯一 router/安全链 | `main.go:2183` 的 17 handler 接入现役唯一 router；`:3191`～`:3222` 注册 orphan list/detail/publish/transfer/DELETE，detail Allow 为 GET, DELETE，动作仅 POST，错误 method 405、额外路径/尾斜杠 404。GET 走 Session/AccessGeneral，再由 handler/service 核 admin；写操作套既有 Origin→Session/CSRF/AccessGeneral，handler 再核 admin。DeleteOrphan 新增 admin 前置检查，未形成旁路或第二 router。 |
| 真实 orphan 查询 | `query_service.go:721` 使用生产 PostgresArticleQueryRepository；SQL 在 keyset LIMIT 前筛 author NULL、source 非空、held、active，排除 pending。详情再次检查当前 disposition set，已匿名公开/转移/删除不可读为待处置。cursor 使用 AdminOrphan view、最后显示行和稳定 created_at/id 次序；member 原查询实现未改为 stub。 |
| HTTP/ETag/strict body | `handler.go:1006` 复用单个强 If-Match 与解析，核路径 Article ID；publish 不接受 body；transfer 复用有限字节、UTF-8、严格 JSON 解码及目标 UUID 校验。未知/重复字段、尾随 JSON、缺失/弱/错/旧 ETag 有局部拒绝证据；非管理员优先拒绝，不触发业务写入。 |
| 转移响应与隐私 | `handler.go:981` 输出 article + notification(id, delivery_capability, 两种 pending recipients)，不序列化 email_snapshot；仍 violating 的结果保留 violation hidden_reason。ProcessingGate open/closed 只在 commit 后影响能力投影，未把 pending 报成发送成功；closed 不回滚归属或清理，无 HTTP SMTP 调用。 |
| 独立 max_attempts 配置 | `main.go:626` 读取独立 `HAN_ARTICLE_TRANSFER_NOTICE_MAX_ATTEMPTS`，复用严格十进制正 int64 parser，无默认/别名/另一策略 fallback，拒绝空白、符号、前导零、非整数、非正及 bigint 溢出。经 startup→composition→command 传递，进入两条 recipient 的创建快照；正 bigint 上界有配置用例。 |
| 新 T1 时间精度修复 | `deletion_coordinator.go:166` 在 operation 创建、INSERT 和首次 Journal Append 之前统一 `Clock.Now().UTC().Truncate(time.Microsecond)`，同一值进入 DB/Journal；pending 重试继续使用原 operation.StartedAt。新增 `orphan_disposition_test.go:37` 用非零亚微秒时钟及首次 Append 失败/更晚时钟重试验证不重新生成时间。只修新写入，未改 Journal 格式/旧记录或放宽历史 binding。 |

没有新增实质 finding、方向决定或需阻断本片的建议。现有②报告历史不覆盖改写；StartedAt 精度风险在本次 coordinator 新写入代码/单元范围已处理，真实 PG round-trip 仍待专项确认。

## 验证、限制与交回

Reviewer 现场：`go test ./internal/article ./cmd/han-server -run '^TestOrphanDisposition' -count=1` **PASS**；六文件 `gofmt -l` 无输出。未重复全仓/race/vet/Manifest，未运行 PG/harness。

Lead 提供的独立现场结果：article 22 顶层/72 节点、cmd 11 顶层/67 节点均 PASS、0 skip；全树对账符合既定 13 代码路径与文档/报告范围，无删除。作者另报普通/race/vet/gofmt PASS。以上按各自来源记录，未冒充 Reviewer 重跑；最终 Manifest 尚待 PG 冻结后的最终 revision 回收。

当前成功测试使用真实生产方法配模拟 pgx.Tx/部分测试 seam：route registration 的替换 handler 证据不能替代 handler 真实业务成功，已有独立 real-handler 安全拒绝/HTTP 响应和 native-proof 协作证据相互补充，仍不能等同真实数据库/部署联合结果。

未验证：James 的十组 PG、实际锁/事务/SQL 精度、真实生产请求全链、部署/E2E，以及旧已落盘纳秒 tombstone 的受控兼容与恢复。旧记录不覆盖、不迁移；既有无来源/旧备份恢复限制与 readiness Gate 未由本报告关闭。

交回当前 Lead；待 PG 文件固定后，对完整 13 文件最终修订和真实证据另做 joint 评审。若 root/reference/trace/响应/配置/精度合同变化，以新固定 hash 复核。保留 draft 与真实人的确认边界，不生成 checkpoint、签字或完整任务完成声明。核对报告及固定哈希后停止写入。
