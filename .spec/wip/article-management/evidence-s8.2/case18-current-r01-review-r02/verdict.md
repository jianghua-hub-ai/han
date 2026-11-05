# Case18 当前 revision 执行证据 r01：登记补正独立复审 r02 — 判定

**AI 代理评审，非真人签字。**

## 一句话结论

`C18-R01-REV-F001` 与 `C18-R01-REV-F002` 均可登记 CLOSED；`C18-COV-F001` 可正式登记 CLOSED（仅 finding 级）。

## 逐项判定

| 项 | 判定 | 依据摘要 |
|---|---|---|
| `C18-R01-REV-F001`（MEDIUM，保留 schema 少计） | **CLOSED** | 补正清单 28 个 `han_admin2_*` + 3 个 `han_s33_*` 与两份 run 日志去重集合逐项一致，漏项 0、多项 0；同目录只追加更正并显式作废旧登记值，足以关闭该 MEDIUM，无需改写原 `report.md`/`index.json` 本体 |
| `C18-R01-REV-F002`（LOW，准入状态词表） | **CLOSED** | 代码实读六态 `closed`/`checking`/`public_ready`/`faulted`/`draining`/`stopped`；guard 仅 `public_ready` 放行；`internal_ready` 为 observation 字面量（`runtime.go:1015`）、`unknown` 来自测试替身；措辞更正成立且 `C18-COV-F002` 仍 open |
| `C18-COV-F001` | **可正式登记 CLOSED（仅 finding 级）** | 承接 r01 判定；本轮补正不改变其事实层，也不等于 S8.2/S1.5 整卡或 Case18 关闭 |
| `C18-COV-F004` 整体 | **仍 open（部分关闭）** | Session 侧已覆盖（identity integration 6 PASS）；AccessGeneral 侧未覆盖，整体不得 CLOSED |
| `C18-COV-F002` | 保持 open | 仍为 fake runtime + 零值 pool，无真实 runtime/PG/双系统 |
| `C18-COV-F003` | 保持 open | 无完整恢复 → `public_ready` → 生产开流证据 |
| `C18-COV-F005` | 保持 open | 时点快照语义未变 |
| S1.5 / S8.2 状态 | 仍 `[-]`；`blocked-upstream` 不解除 | 与 `tasks.md` 现记录一致 |
| 边界标注 | PASS | 补正明确列出全部未证明项，无夸大表述 |

## 是否具备进入 Case18 关闭评估的条件

**不具备。** S8.2 仍 `[-]`、`blocked-upstream` 未解除；全状态真实隔离与真实 runtime/PG、当前 runtime 完整恢复+开流、真实部署 E2E/双系统、证据登记闭环等条件仍未全部满足。本轮只关闭两条评审 finding，不改变任务状态与关闭门禁。

## 非阻断观察（不影响 CLOSED）

- 补正 `:54` 的 session 字段概括（`created=true retained=true`）与日志字面（`created_here=true retain_policy=always`）措辞不一致，语义相同。
- 补正 `:61` 把 Readiness 的 503 probe 与受保护请求的 `writeRecoveryHTTPClosed` 混写；核心“仅 `public_ready` 放行”成立。
- 补正文件尚未登记进 `index.json` artifacts/runs；若有只读旧索引的消费者，建议后续加指针/哈希。

**AI 代理评审，非真人签字。**