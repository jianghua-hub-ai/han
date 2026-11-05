# Case18 当前 revision 执行证据 r01：独立复审判定

**AI 代理评审，非真人签字。**

## 结论

**CHANGES REQUIRED**：证据事实层足以支持 `C18-COV-F001` CLOSED、`C18-COV-F004` Session 侧已覆盖，且独立重跑 8 条命令全部通过；但证据包保留 schema 少计 29 个（`C18-R01-REV-F001`，MEDIUM），需修正登记或由责任人显式接受后才能视为完整交付。

## 逐项判定

| 项 | 判定 | 要点 |
|---|---|---|
| `C18-COV-F001` | **CLOSED** | 当前 revision root/装配普通+race 各 93 PASS，独立复现；8 个源文件绑定一致；旧 `6B45B550…/B53DD34D…` 确属不同对象 |
| `C18-COV-F004` Session 侧 | **已覆盖** | 真实 PostgreSQL + `session.NewStore`；撤销/停用后 `ResolvePrincipal` 返回 `ErrSessionRequired`；本次 6 PASS 独立复现 |
| `C18-COV-F004` 整体 | **仍 open / 部分关闭** | AccessGeneral 侧仍未覆盖；不得整体 CLOSED |
| `C18-COV-F002` | **不变，保持 open** | 仍为 fake runtime，无真实 runtime/PG/双系统；报告状态词表另有 LOW 级表述问题 |
| `C18-COV-F003` | **不变，保持 open** | production `Enter`/`Install` 仍返回 `ErrAdmissionClosed`，无完整恢复后开流证据 |
| `C18-COV-F005` | **不变，保持 open** | 本证据包未修正 `producer-real-r01/index.json` 的哈希/字节问题 |
| 边界标注 | **PASS** | 10 项未证明均如实列出；未发现“S1.5/S8.2 已关闭、Case18 已 PASS、production readiness”表述 |
| 证据完整性 | **FAIL（MEDIUM）** | 原日志 28 个 `han_admin2_*` + 3 个 `han_s33_*`，报告/索引/任务卡只各登记 1 个；少计 29 个保留 schema |

## 是否具备进入 Case18 关闭评估的条件

**不具备。** 上轮六项前置中，当前 revision root/装配证据（第 1 项）与 Session 侧真实 SQL/撤销证据（第 2 项）已满足；但全局恢复隔离全状态与真实 runtime/PG（第 3 项）、当前 runtime 完整恢复+开流（第 4 项）、真实部署 E2E/双系统（第 5 项）、证据目录哈希/字节修正（第 6 项，含 F005 与本轮 schema 登记问题）仍未满足。`C18-COV-F002/F003/F005` 保持 open，F004 仅部分关闭。

## 独立重跑摘要

| 命令 | EXIT | PASS | FAIL | SKIP |
|---|---:|---:|---:|---:|
| root normal | 0 | 93 | 0 | 0 |
| root race | 0 | 93 | 0 | 0 |
| UC normal | 0 | 135 | 0 | 0 |
| UC race | 0 | 135 | 0 | 0 |
| build | 0 | - | - | - |
| vet | 0 | - | - | - |
| article integration | 0 | 99 | 0 | 0 |
| identity integration | 0 | 6 | 0 | 0 |

集成测试本次新产生并保留 28 个 `han_admin2_*` 与 3 个 `han_s33_*` schema；未 DROP、未清理。`-race` 日志本身不能自证 race 已启用，本次独立重跑是主要证据。

## 证据限制

不证明真实部署 E2E、真实 SMTP/approved-provider、双系统接线、完整恢复后生产开流、`Enter`/`Install` 开流、真实 Journal/HMAC/keyring、旧备份恢复、容量/P95、production readiness 或人的业务接收。本评审不改任务状态，不宣布任何 task CLOSED。