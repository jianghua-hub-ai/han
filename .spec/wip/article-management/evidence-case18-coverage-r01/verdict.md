# Case18 覆盖对账：独立评审结论

> **AI 代理评审，非真人签字。** 本结论仅用于 Lead 的关闭评估输入，不代替 Lead 决定、不代替人的业务接收或部署授权，不宣称 S8.2 / Case18 / S1.5 已 PASS 或已关闭。

## 结论

**CHANGES REQUIRED**

一句话理由：Case18 五分项仅得到部分当前版本装配/模拟证据，恢复隔离状态覆盖不全，完整恢复后生产开流在当前 revision 不可核，S8.2 不具备进入关闭评估的条件。

## 逐项判定

| 项 | 判定 | 要点 |
|---|---|---|
| 1. 真实factory/共享实例 | PARTIAL | 源码唯一共享实例成立；S8.2 root 证据绑定旧 `main.go`/`main_test.go`，当前 revision 无执行证据 |
| 2. Trace/exactrouter | PARTIAL | 当前源码 exact route/method/Allow/旧路由共存可静态核对；缺当前 revision 真实执行证据，且不能以 spy router 计 |
| 3. Session/AccessGeneral | PARTIAL | 可信 Principal、GET 无 CSRF、账号级拒绝零文章下游有接缝证据；缺真实 `SessionStore` SQL/会话撤销 |
| 4. 全局恢复请求隔离 | PARTIAL | `closed`/`unknown` 两个 admin GET 503 + 零下游成立；缺 `failed`/`expired`/`undetermined`、真实 runtime/PG/双系统、当前执行证据 |
| 5. 完整恢复后生产开流 | NOT COVERED | S4.20 只证旧 runtime resume 链；S4.21 production Enter/Install 仍 closed；当前 revision 开流不可核 |

## Findings

- **C18-COV-F001（HIGH）**：S8.2 root 证据绑定旧 `main.go 6B45B550…` / `main_test.go B53DD34D…`，当前 revision（`6D5373…/173317`、`372065…/400312`）无执行证据；`PExecuted:false`、`Case18Overall:"BLOCKED upstream"`。
- **C18-COV-F002（HIGH）**：全局隔离矩阵只覆盖 `closed`/`unknown`，使用 fake runtime，缺 `failed`/`expired`/`undetermined` 与真实 runtime/PG/双系统。
- **C18-COV-F003（HIGH）**：S4.20 d7 证据绑定旧 `recovery_runtime.go 900F31F5…/35832`；S4.21 明写 production Enter/Install 仍 `ErrAdmissionClosed`、真实 PG S4.20 未重跑，当前 revision 无法核完整恢复后开流。
- **C18-COV-F004（MEDIUM）**：Session/AccessGeneral 仅有 resolver/repository/handler 接缝；缺当前 revision 真实 `SessionStore` SQL/撤销证据；账号级 403 不等于全局 503。
- **C18-COV-F005（MEDIUM）**：`evidence-s1.5/producer-real-r01/index.json` 声明 `report.md` 为 `A9A4ECBB…/8023`，实算 `1B2FB3CF…/8267`，目录完整性需修正。
- **C18-COV-F006（LOW）**：`tasks.md:2957` 的 `d7-chain-complete-race-r02.json` 哈希记录 62 位、缺 `84`；实物 64 位、27055 B 一致。
- **C18-COV-F007（LOW）**：无真实部署 E2E/双系统；`sim-closeout-r01/index.json:22` 明示 Case18 真实双系统 `NOT RUN/unknown`。

## 关闭评估条件

**不具备。** 需先补齐：

1. 当前 revision 的 root/装配执行证据（绑定 `6D5373…` / `372065…`）。
2. 当前 revision 的 SessionStore SQL/撤销真实证据。
3. 全局恢复隔离全状态 + 真实 runtime/PG 证据。
4. 当前 runtime 上完整恢复 + S16/二次核验 + MarkConsumed + 真实开流证据。
5. 真实部署 E2E/双系统。
6. 证据目录哈希/字节修正。

## 复跑说明

- 已执行：`go build ./...` EXIT=0；`go vet ./cmd/han-server` EXIT=0。
- 未执行：任何 `go test`，统一 `NOT RUN`（集成测试会创建并保留 schema，超出只读授权）。

---

AI 代理评审，非真人签字。