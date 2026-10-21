# `coverage.md` 模板（覆盖矩阵 · 可选拆分）

> **仅大套件可选拆分用**——默认覆盖矩阵并入 `cases.md` 顶部 §2 小节（从各 case「覆盖 SR / US」字段派生），**不产独立文件**。仅当套件 >~50 用例 / 多人分工时才拆出本独立文件。内容与 cases.md §2 一致。
>
> 落 `.spec/e2e/<suite>/coverage.md`，每套件一份。**双向可追溯**——用例覆盖了哪些 feature/US/SR，反过来哪条 US 有没有被 E2E 覆盖一眼可查。**本表是派生视图，SoT 是各 case「覆盖 SR」字段**；设计 SSOT 仍在各特性 `design.md`。

---

# <suite> 覆盖矩阵

> **套件覆盖范围**：<feature 列表，如 `<feature-a>` + `<feature-b>`>

## 1. 用例 → 设计映射

| 用例 | 覆盖 feature | 覆盖 US / SR | 覆盖设计节 |
|---|---|---|---|
| CASE-<id> | <feature-a> | <SR-a> | <design.md §x.x> |

## 2. 覆盖度反查（哪些 US/SR 被 E2E 覆盖了）

| feature | US / SR | 是否有 E2E 覆盖 | 用例 | 缺口说明 |
|---|---|---|---|---|
| <feature-a> | <SR-a> | ✅ / ❌ / 部分 | <CASE-id 列表> | <未覆盖 / 部分覆盖原因> |

## 3. 覆盖空白汇总（收尾 Step 6 对账用）

> 列出所有 ❌ / 部分覆盖的 US/SR，注明"是有意不测（环境受限 / B-边界）还是真缺口（待补用例）"。
