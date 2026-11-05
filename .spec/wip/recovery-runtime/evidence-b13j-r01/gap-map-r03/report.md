# B13J gap-map 候选编号拆分 r03 报告

## 范围与结论

- 对象：`.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md`。
- 授权改动仅涉及 `:172`、`:178`、`:182`；逐行比较实测差异行也仅有这三行。
- `S4.24a`、`S4.24b` 仍是候选编号，不是已登记 Task，未分配 Task ID，未进入 DAG、批次或所有权表。

## 逐行对照

### `:172`（ASCII 依赖图）

改动前：

```text
                                                          ├─► S4.24 候选（restore + exclusive）
```

改动后：

```text
                                                          ├─► S4.24a 候选（restore） / S4.24b 候选（exclusive）
```

### `:178`（候选编号清单）

改动前：

```text
- 上方 `S4.22`、`S4.23`、`S4.24` 仅是设计补齐后的**候选编号形状**，不是已登记 Task，不进入当前 DAG、不预占写权。
```

改动后：

```text
- 上方 `S4.22`、`S4.23`、`S4.24a`、`S4.24b` 仅是设计补齐后的**候选编号形状**，不是已登记 Task，不进入当前 DAG、不预占写权。
```

### `:182`（候选边界）

改动前：

```text
  - `S4.24`：restore source/target/tool 与部署独占 OS 后端；依赖 D03/D04。
```

改动后：

```text
  - `S4.24a`：生产 restore source/target/tool 与批准主体接线；受 ADR-0031 门禁，与锁后端节点独立；依赖 D03；`S4.24b`：部署独占 POSIX 锁后端与 `exclusive` 接线；受 ADR-0032 门禁；requires 候选 `S4.23`；依赖 D04；`S4.22` 候选 requires 候选 `S4.24b`。
```

## Canonical 语义

- `S4.24a`：生产 restore source/target/tool 与批准主体接线；受 ADR-0031 门禁；与锁后端节点独立。
- `S4.24b`：部署独占 POSIX 锁后端与 `exclusive` 接线；受 ADR-0032 门禁；requires 候选 `S4.23`。
- `S4.22`：production root 装配 + recovery secret candidate；requires 候选 `S4.24b`；root 装配可在 restore 保持 closed 时进行，与 `S4.24a` 的关系按 ADR-0031 单独登记。

以上语义取自本轮任务输入；未自行扩展 ADR 契约。

## 格式与哈希

| 指标 | 改动前 | 改动后 |
|---|---:|---:|
| 字节 | 26452 | 26700 |
| LF 数量 | 218 | 218 |
| 按 LF 切分片段 | 219 | 219 |
| CR | 0 | 0 |
| TAB | 0 | 0 |
| 控制字符 | 0 | 0 |
| BOM | 无 | 无 |
| 末尾换行 | 无 | 无 |
| SHA256 | `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F` | `0E802578FBFBD8C0946225C3AAC78F35FF28183B13453276E835567FDB292706` |

- 行号按 1-based 文本行核对；LF 计数保持 218，`:201`（U5）与 `:207`（§4 末句）内容和行号均未变化。
- 改动前快照：`gap-map-r03/snapshots/gap-map-before-r03.md`；快照与改前文件逐字节一致。

## 边界声明

- 只修改了 `gap-map.md` 的 `:172`、`:178`、`:182`。
- 未修改 `tasks.md`、Plan、ADR、`framework.md`、`design.md`、源码或测试。
- 未登记任何 Task，未提供 Task ID，未更新 DAG、批次计划或文件所有权表。
- 本报告是执行记录，不是人的签收，也不代表 ADR 修订或人工验收已完成。
- 授权外观察：`:174` 仍保留旧的合并引用 `S4.23 + S4.24 + S4.22`，本轮未改动。

---

## 第二轮收尾（r03b）

### 第二轮逐行对照

#### `:139`
改动前：

```text
- **后续验证关联**：只解锁 `S4.24` 候选（restore source/target/tool 生产实现）；真实旧备份演练仍独立，不由本项或 `article-management-S4.4` 的 synthetic fixture 代替。
```

改动后：

```text
- **后续验证关联**：只解锁 `S4.24a` 候选（restore source/target/tool 生产实现）；真实旧备份演练仍独立，不由本项或 `article-management-S4.4` 的 synthetic fixture 代替。
```

#### `:161`
改动前：

```text
- **后续验证关联**：解锁 `S4.22` root 装配候选；`S4.22` 的 production 可达证明仍须消费 `S4.23`/`S4.24` 成果；Case18 完整开流另立真实 E2E，`article-management/test-plan.md:360` 不解阻。
```

改动后：

```text
- **后续验证关联**：解锁 `S4.22` root 装配候选；`S4.22` 的 production 可达证明仍须消费 `S4.23`/`S4.24a`/`S4.24b` 成果；Case18 完整开流另立真实 E2E，`article-management/test-plan.md:360` 不解阻。
```

#### `:174`（ASCII 图内）
改动前：

```text
S4.23 + S4.24 + S4.22 ─► 新联合软件验证 ─► 真实 E2E（另立）
```

改动后：

```text
S4.23 + S4.24a + S4.24b + S4.22 ─► 新联合软件验证 ─► 真实 E2E（另立）
```

### 残留检查

- `rg -n 'S4\.24' gap-map.md` 共 6 条匹配，均为 `S4.24a` 或 `S4.24b`。
- `rg --pcre2 -n 'S4\.24(?!a|b)' gap-map.md` 无输出；不带 `a`/`b` 的裸引用为 0。
- 第二轮逐行比较相对 `gap-map-before-r03b.md` 仅 `:139`、`:161`、`:174` 变化。

### 第二轮格式与最终哈希

| 指标 | 第二轮改前（r03b 快照） | 最终文件 |
|---|---:|---:|
| 字节 | 26700 | 26721 |
| LF 数量 | 218 | 218 |
| 按 LF 切分片段 | 219 | 219 |
| CR | 0 | 0 |
| TAB | 0 | 0 |
| 控制字符 | 0 | 0 |
| BOM | 无 | 无 |
| 末尾换行 | 无 | 无 |
| SHA256 | `0E802578FBFBD8C0946225C3AAC78F35FF28183B13453276E835567FDB292706` | `88AE01132F573D14E4C06217AC1E0D8587D0B68E596242A4CA5A31B721538FA7` |

- `:201`（U5）与 `:207`（§4 末句）内容和行号均未变化。
- `gap-map-r03/snapshots/gap-map-before-r03.md` 保持首轮改前原样，SHA256 仍为 `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F`。
- 本轮改前快照：`gap-map-r03/snapshots/gap-map-before-r03b.md`，SHA256 为 `0E802578FBFBD8C0946225C3AAC78F35FF28183B13453276E835567FDB292706`。

### 第二轮边界

- 第二轮只修改 `gap-map.md` 的 `:139`、`:161`、`:174`。
- 第一轮报告中的 `:174` 授权外观察已由本轮新授权修正。
- 未修改 ADR、Task、Plan、源码或测试；未登记任何 Task；本报告仍是非人的签收。

---

## 第三轮收尾（r03c）

### 第三轮逐行对照

#### `:182`
改动前：

```text
  - `S4.24a`：生产 restore source/target/tool 与批准主体接线；受 ADR-0031 门禁，与锁后端节点独立；依赖 D03；`S4.24b`：部署独占 POSIX 锁后端与 `exclusive` 接线；受 ADR-0032 门禁；requires 候选 `S4.23`；依赖 D04；`S4.22` 候选 requires 候选 `S4.24b`。
```

改动后：

```text
  - `S4.24a`：生产 restore source/target/tool 与批准主体接线｜受 ADR-0031 门禁｜与锁后端节点独立｜依赖 D03。 `S4.24b`：部署独占 POSIX 锁后端与 `exclusive` 接线｜受 ADR-0032 门禁｜requires 候选 `S4.23`｜依赖 D04。 `S4.22`：production root 装配 + recovery secret candidate｜requires 候选 `S4.24b`｜与 `S4.24a` 的关系按 ADR-0031 单独登记。
```

- 仅 `:182` 一行变化；三段分别对应 `S4.24a`、`S4.24b`、`S4.22`，未新增或删除候选编号。
- 三段之间以句号分隔，段内以 `｜` 分段；候选语义与上一轮一致，并明确保留 `S4.22` 与 `S4.24a` 的关系按 ADR-0031 单独登记。

### 残留与关键行核验

- `rg --pcre2 -n 'S4\.24(?![ab])' gap-map.md` 无输出；不带 `a`/`b` 的裸引用为 0。
- `:201` 与 `:207` 的内容逐字节未变；对两行 UTF-8 行字节计算的 SHA256 前后一致。

| 行 | 内容相同 | 改前/改后行内容 SHA256 |
|---|---|---|
| `:201` | True | `3B4E58DFA7AEACED243C1E40528E3F15D5D104A513A0A29DF6B80957C233E719` |
| `:207` | True | `268359AF00E612FC46A27102C57BC6A3663C148C66C681ADDBBCC75DA2C6F265` |

### 第三轮格式与最终哈希

| 指标 | 第三轮改前（r03c 快照） | 最终文件 |
|---|---:|---:|
| 字节 | 26721 | 26821 |
| LF 数量 | 218 | 218 |
| 按 LF 切分片段 | 219 | 219 |
| CR | 0 | 0 |
| TAB | 0 | 0 |
| 控制字符 | 0 | 0 |
| BOM | 无 | 无 |
| 末尾换行 | 无 | 无 |
| SHA256 | `88AE01132F573D14E4C06217AC1E0D8587D0B68E596242A4CA5A31B721538FA7` | `D6AA33B6787774117E57448E3008C58DC9983E887C9CE078E5A6F07DE426F214` |

- 本轮改前快照：`gap-map-r03/snapshots/gap-map-before-r03c.md`，SHA256 为 `88AE01132F573D14E4C06217AC1E0D8587D0B68E596242A4CA5A31B721538FA7`。
- `gap-map-before-r03.md` 与 `gap-map-before-r03b.md` 均未覆盖。

### 第三轮边界

- 第三轮只修改 `gap-map.md` 的 `:182`；未修改 ADR、Task、Plan、源码或测试。
- 未登记任何 Task；本报告仍是非人的签收。

本报告格式：纯 LF、末尾单 LF、无 BOM。
