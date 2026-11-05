# S4.20 门禁措辞 R02→R03 同步提案（s420-card-revision-r02）

- 状态：`READY_FOR_LEAD_APPLY`
- 角色：add-task-planner（同一 B13H / S4.20 上下文）
- 日期：2026-10-31（Asia/Shanghai）
- 工作区：`D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 授权边界：只写本目录文件；未改 `tasks.md` / Plan / 源码 / 其他证据；无 Git 写；未运行 `go` / PG / HTTP。

## 1. 基线核对（实际读取）

| 对象 | 本次实际读取 | 输入声明 | 结论 |
|---|---|---|---|
| `tasks.md` | SHA256 `75815F06F9C144F55F7382B4371528784B2B9998E3C2F792E8E949405A424554` / 717552 B / 2931 行 | `B72F280D0D57248875CA4DE91BF2AD0E08C243A01486BE0C243BB74FCCB4DC4E` / 712197 B / 2929 行 | **不一致（+2 行 / +5355 B）** |
| Plan `ir-01-content-community-batches.md` | SHA256 `4B0150A3D5490E0D5AD44F2BE9D8B7412A69140A6A89BD2419865C76032E6CD9` / 488454 B / 1615 行 | 同上 | 一致 |

- `tasks.md` 漂移定位：差异在 `article-management-S1.5` 记录区（现 `1232`、`1233` 行）；S4.20 卡内容逐字未变，仅整体下移 2 行。
- 实际 R03 Gate 记录位于 `tasks.md:1233`（输入称 1232）；其末句明确 S4.20 的“S1.5 ACL 可消费输入”条件已满足（非作者 PASS + Lead Gate PASS），并要求 Task-Planner 同步卡内 R02 文字。
- 因此本提案全部按**内容定位**（不以行号匹配）；7 处 before 片段在当前文件中的匹配次数均为 1。应用时 Lead 仍须先重算哈希并确认 S4.20 卡内容未变。
- 格式：两文件均 UTF-8 无 BOM；`tasks.md` 全 2931 行均为 CRLF；Plan 共 1615 个 LF，其中 1611 个为 CRLF，`1596/1598/1599/1600` 为 LF-only 行（`1600` 即本提案目标行）。

## 2. tasks.md 卡内替换（5 处，当前行号）

行号为本次实际读取值；应用时以片段唯一匹配为准。

### T1 — `类型/Story与范围`（第 2866 行）

旧文（before，匹配次数 1）：

```text
；`article-management-S1.5`为条件性未来引用，仅在其 ACL R02 非作者独立复审 PASS 且 Lead Gate PASS 后，才可作为可消费输入（当前不可消费）。
```

新文（after）：

```text
；`article-management-S1.5`为条件性未来引用，其 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS 与 Lead Gate PASS 已满足（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已登记处置），S1.5 ACL 子集可作为可消费输入（整卡仍 `[-]`，不构成整卡 PASS）。
```

### T2 — `依赖`（第 2885 行）

旧文（before，匹配次数 1）：

```text
`article-management-S1.5`为条件性未来引用，仅在其 ACL R02 非作者独立复审 PASS 且 Lead Gate PASS 后可消费（当前不消费、不作硬requires）
```

新文（after）：

```text
`article-management-S1.5`为条件性未来引用，其 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS 与 Lead Gate PASS 已满足（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已登记处置），S1.5 ACL 子集可消费（整卡仍 `[-]`，不构成整卡 PASS；仍不作硬requires）
```

### T3 — 风险 leaf `MainWriteWindowFrozenAndS419Reverified`（第 2922 行）

旧文（before，匹配次数 1）：

```text
START 前满足 S1.5 ACL R02 非作者独立复审 PASS + S1.5 Lead Gate PASS + S1.5 writer 明确 STOP/`main.go` 写窗冻结
```

新文（after）：

```text
START 前满足 S1.5 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS + S1.5 Lead Gate PASS（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已登记处置）+ S1.5 writer 明确 STOP/`main.go` 写窗冻结（该项仍未满足）
```

### T4 — `后续验证关联`（第 2926 行）

旧文（before，匹配次数 1）：

```text
article S1.5 为条件性未来引用，仅在其 ACL R02 非作者独立复审 PASS 且 Lead Gate PASS 后评估。
```

新文（after）：

```text
article S1.5 为条件性未来引用；其 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS 与 Lead Gate PASS 已满足（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已登记处置），S1.5 ACL 子集可评估消费（整卡仍 `[-]`，不构成整卡 PASS）。
```

### T5 — `串行/停止条件` START 硬门禁 F002（第 2928 行）

旧文（before，匹配次数 1）：

```text
「`article-management-S1.5` ACL R02 非作者独立复审 PASS」+「S1.5 Lead Gate PASS」+「S1.5 writer 明确 STOP 且 `cmd/han-server/main.go` 写窗冻结」
```

新文（after）：

```text
「`article-management-S1.5` ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS」+「S1.5 Lead Gate PASS」（二者见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已登记处置）+「S1.5 writer 明确 STOP 且 `cmd/han-server/main.go` 写窗冻结」（该项仍未满足）
```

## 3. Plan 替换（2 处，同一行 `1600`，LF-only 行）

### P1 — 可消费状态

旧文（before，匹配次数 1）：

```text
article-management-S1.5为条件性未来引用，仅在其 ACL R02 非作者独立复审 PASS 且 Lead Gate PASS 后可消费（当前不可消费、不作硬requires）
```

新文（after）：

```text
article-management-S1.5为条件性未来引用，其 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS 与 Lead Gate PASS 已满足（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已登记处置），S1.5 ACL 子集可消费（整卡仍 `[-]`，不构成整卡 PASS；仍不作硬requires）
```

### P2 — START 硬门禁

旧文（before，匹配次数 1）：

```text
START 硬门禁按原卡：S1.5 ACL R02 非作者独立复审 PASS + S1.5 Lead Gate PASS + S1.5 writer 明确 STOP 且 `main.go` 写窗冻结
```

新文（after）：

```text
START 硬门禁按原卡：S1.5 ACL 当前 revision（R03，PASS WITH MINOR）非作者独立复审 PASS + S1.5 Lead Gate PASS（见 tasks.md 2026-10-31 R03 Gate 记录；三条 LOW 已登记处置）+ S1.5 writer 明确 STOP 且 `main.go` 写窗冻结（该项仍未满足）
```

## 4. 全文件 R02 残留清单

- S4.20 范围（卡 + Plan 登记段）：替换前共 7 处（`tasks.md` 2866/2885/2922/2926/2928；Plan 1600 两处）；按本提案替换后为 **0 处**。
- 明确保留、不得修改的范围外 `R02` 引用：
  - `tasks.md` 1230–1233：S1.5 ACL R01 评审 / R02 交付 / R02 复审与 R03 窄范围返工 / R03 Gate 记录（历史，含 `review-code-s1.5-acl-r02.md` 文件名）。
  - `tasks.md` 1852：`review-code-s8.2-joint-r02.md`（S8.2 联合评审文件名）。
  - `tasks.md` 2812、2857、2858：S4.19 R01 FAIL / R02 停止 / R02 复审 / R03 关闭的历史记录。
  - `tasks.md` 其余 141 行（全文件共 146 行含 `R02`，其中 5 行为本次 S4.20 范围）：其他任务（S4.x/S8.x/S10.x/article 等）的历史引用，均不在 S4.20 范围。
  - Plan 1597–1599：S4.19 R02 历史；其余 128 行（全文件共 129 行含 `R02`，其中 1 行为本次 S4.20 范围）与 S4.20 无关。
  - `.spec/wip/recovery-runtime/evidence-b13h-r01/s420-card-revision-r01/`（上一版提案及其片段）含 R02 旧文；属既有历史证据，不修改。
- 其他知识文档：对 `.add/project/knowledge`（除上述两文件）做只读全量扫描，未发现 `ACL R02 非作者独立复审` 字样。
- 结论：**未发现其他 S4.20 范围内的 R02 旧门禁表述**；范围外历史记录一律保持原样。

## 5. 未改动项声明

- 第三重门禁原文「S1.5 writer 明确 STOP 且 `…main.go` 写窗冻结」保留；T3/P2 仅在括号中标注“该项仍未满足”，不改其文字。
- 三重门禁的 AND 结构、任一不满足即 STOP 的语义保留。
- F008：实施前/后记录 `main.go` SHA256/字节/mtime、期间任何变化即 STOP 重核，均原样保留（`tasks.md` 2922/2928、Plan 1600）。
- 任务卡保持 `[ ]`、无完成日期、全部验收命令 NOT RUN；不新增完成或 PASS 声明。
- 不新增/删除 `requires`；不改其他任务卡、其他批次、其他行；4 文件写集不变；批次归属仍 B13H。
- 不把“R03 PASS WITH MINOR + Lead Gate PASS”写成整卡 S1.5 PASS；不称 ACL 无遗留；三条 LOW 仅以“已登记处置”一笔带过，不展开 DSN 或口令细节。
- 未修改 `tasks.md` / Plan / 源码 / 其他证据目录；未执行 Git 写；未删除、移动或覆盖任何既有内容。

## 6. 一致性核对表

| 约束 | 核对结果 |
|---|---|
| 可消费状态改为已满足（R03 + Lead Gate PASS，指向 tasks.md 2026-10-31 R03 Gate 记录） | T1/T2/T4/P1 已覆盖 |
| 三重门禁结构与第三重门禁原文不变 | T5/P2 保留原文，仅加“该项仍未满足”标注 |
| `main.go` 实施前后 SHA256/字节/mtime 与变化即 STOP（F008） | 原样保留（2922/2928/Plan 1600） |
| 卡 `[ ]`、无完成日期、全部 NOT RUN | 未触碰 |
| 不新增/删除 requires | 未触碰依赖结构 |
| 4 文件写集、批次 B13H 不变 | 未触碰文件所有权与批次行 |
| 不重开已关闭卡、不改其他卡/批次/行 | 仅 5+2 处片段级替换 |
| 不声称整卡 S1.5 PASS / ACL 无遗留 | 新文统一写“整卡仍 `[-]`，不构成整卡 PASS”；LOW 仅“已登记处置” |
| UTF-8 无 BOM、行尾保持 | 只写文本；替换片段不含换行，应用后行数与行尾不变 |

## 7. Lead 应用步骤（机械替换）

1. 重算 `tasks.md` SHA256；若 ≠ `75815F06…`，先核对差异是否触及 S4.20 卡。若 T1–T5 仍各自唯一匹配，可按内容应用；任一不再唯一或不存在即停止。
2. 在 `tasks.md` 按 T1→T5 逐条替换 before→after（每处应恰好 1 次匹配），保持 CRLF 与 UTF-8 无 BOM。
3. 在 Plan 第 `1600` 行按 P1→P2 替换；该行为 LF-only，替换时不得引入 CR。
4. 应用后重算并记录两文件 SHA256/字节/行数；确认 S4.20 卡仍 `[ ]`、无完成日期、NOT RUN，三重门禁与 F008 措辞完整。
5. `tasks.md:1233` 的 R03 Gate 记录已注明“卡内 R02 文字由 Task-Planner 提案同步”，应用后无需改动该记录；不要新增记录行。

## 8. 未决 / unknown

- `tasks.md` 存在并发写者：本次读取哈希已与 Lead 输入不同（+2 行 / +5355 B）；应用时可能再次漂移，必须按内容重核。
- R03 Gate 记录实际行号为 1233（输入称 1232）；本提案不写死行号引用。
- 三条 LOW 的处置细节以 `tasks.md:1233` 记录为准；本提案不复述、不展开。
- 本提案不验证 S4.20 实施可行性、oracle 可执行性或任何测试结果（全部仍 NOT RUN）。