# B13D S4.10 r56 独立 Design Review

Dirac / add-reviewer；B13D-S410-DESIGN-R56-R01；objectType=design；development-delivery 原链续接。固定 D:/AID/VSCode/Codex/Han / main / 6da8abbceb0fa197364810a92a8fa2d97b38ddca（工作锚点取用户授权；本轮未运行 Git）。仅 identity r55→r56；不重审 r55、Article r27 或已接收 joint4 mapping。

**有界 Design PASS，新增 finding 0，无本增量已确认 HIGH/MED 阻断。** 新来源包装、原域失败核验及 cohort/ACK/Complete 合同可在原 S4.10 三文件的私有 receiver/依赖/SQL/测试内实现；这只是设计可实施性判断，不是现役实现、源码准入或整卡 READY。

| 唯一对象 | 实际 SHA256 |
|---|---|
| .add/project/knowledge/design-workspace/identity-auth/design.md，r56 entry=exit=作者 after | 75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF |
| r55 before | 419B800DE82405CE0D24ED0E9FCE3788D112E298F14C89220F49C4D313029DE2 |
| 本轮完整 design-r56.diff | 7D92E616B2493A177394C089664FFDB41FB826D71C94F5D07081CE7DEBFB0DAB |

**Round 1 实际范围**：完整139行 diff、receipt、consumer-risk-map、Main freeze；新增 §4.2 全文及 N003/状态表/finalize/§6.1/§9.1 变更。独立静态比较 before/after/live：原49个代码/图块、确认尾部均保留；仅原5/38/3132/3487行替换，新增1声明块；原169 CRLF、无BOM保持。Main/作者保护断言未代替本轮计算。

**Round 2 实际范围与直接证据**（均为仓内静态读取；完整路径、SHA、行段见 readset.json）：

| 核对项 | 真实来源与判断 |
|---|---|
| 数据不是 grant | design:3910–3932；runtime.go:95–141、1117–1180。ReadFailed 返回原件、Scope、六字段及批准属性；真实 raw PhasePermit/PhaseContext 仍由当前 Runtime 核实例/代际/目的/ctx/独占。重启绑定不冒批准；深拷贝含 Body/FieldErrors，外部 I/O 真 join，来源失败零业务写。 |
| 原域安全快照 | idempotency/store.go:194–208、464–525、776–798、1087–1134；0004:26–57/67–127、0007:113–149；administrator_create.go:865–898/1022–1048/1192–1217；temporary_credential_resend.go:608–659/832–836。两种 v1 失败与 legacy resend 的字段、消息和 Scope 编码匹配；不能凭 generic failed 或 initial 身份制造结果。 |
| linked domain authority | password_reset.go:287–307/708–718；password_reset_store.go:491–545/792–819；password_reset_operator.go:374–467/553–563。durable 分类依赖 operation/claim/credential/delivery/settlement/job/lineage，不等同旧 replayPasswordReset 的 legacy 投影。r56:3940–3942 保留原 classifier、已结算只读、unknown/保护窗/hold/dead-letter 权限；本端口不替原 owner settle 或授新 generation。 |
| 单 Store 原子消费与锁序 | recovery_run_store.go:17–25/67–111、recovery_admission_store.go:18–59；idempotency.Store.Recover 接受 caller-owned Transaction；audit.Store.Append:49–103 接受同 pgx.Tx。原 executor 新文件可在 identity 包内使用唯一 store.pool 与原 SQL，不必嵌套调用自开事务的 Transition，也不需改旧 Store/DDL/Runtime。r56:3944–3948 在源读取后重核，actor/target User→Store→适用域行→record/Audit；执行 actor/一次微秒 Now 由 consumer 提供。Metadata 的新 int64 字符串可穿过现役 audit.Get 的 map[string]any 解码而不损精度。 |
| ACK、terminal 与 Complete | r56:3950–3954；原共同 C:3683–3703；Tasks:2180–2211。writer 实际结束后，另一物理 same-pool 连接有界核完整 row/snapshot/唯一 Audit，terminal 同意图只读；未知不盲重写。cohort 在原 finalize Audit 同事务精确保存，合法空集须原证明，旧无证/删漏行/只计数均闭。Complete 源读取在事务外，Store 锁内仅核 DB 全证据/版本，无反向 User 锁和外部源 I/O；保留 r55 全部共同 C 与逐 decision P，仅推进 identity_ready。实际物理连接/Join/SQL原子性尚待实现验证。 |

原 source3/六组/最低 PG 层保留；§4217、4423 将新增来源/非法投影/取消真 join/ACK loss/terminal/cohort/空集/锁序落实到原组。后继 TE/Planner 只需精确同步这些原组与 I-REC-10/相邻 ACK、Complete、锁序映射；本 Design Review 不要求追加 TP 前审、第四文件、新 Task/requires 或 whole upstream 重跑。

**限制**：L-IDEM-EVIDENCE-01 的原“没有具体失败来源合同”在本设计层已有明确承接；不据此关闭原实现问题、重开受影响 SOURCE 分支或宣称 S4.10 whole-card READY。真实 provider、批准主体/介质/ACL 仍 unknown；未知时仅受影响 resolve/有未决 cohort 的 Complete 关闭。独立批准 checkpoint fixture 可验证组件正向，不能充生产批准或 root 接线；S4.16 后续真实装配不倒挂本组件，也不挂到 Article。无第四源码、旧 DTO/DDL/Runtime/wire/批准权限变更。原 r55/r27 与 joint4 mapping PASS 范围不变；draft、人的接纳 unknown。

**NOT RUN**：Go/编译/格式/测试/race、SQL/PG、服务/进程、实际 provider/ACL/物理 ACK/恢复/root/native/E2E/容量测量。仅运行自有 Temp 的 Python 静态文本/哈希比较。仓内零写；未读/hash 活动 cmd5 包、peer Temp 候选，未读 secret ENV/DSN，未网络、Git、清理或子委派。读集完整性见 receipt：工具输出截断未作为完整阅读证明，所需片段另行补读。

**STOP。交 Main 接收；不替作者修复或替 Main 采用，不授源码/Go/PG/Taskclose/DesignCheckpoint 或人的签字。**
