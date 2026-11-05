# design.md r31 交叉引用修正证据（AMQ2-R02-F001）

日期：2026-11-02
对象：`.add/project/knowledge/design-workspace/article-management/design.md`
范围：仅两条 LOW 级交叉引用修正（AM-Q2 R02 复审：PASS WITH CONDITIONS，0 BLOCKER / 0 HIGH / 0 MEDIUM / 2 LOW）
执行角色：add-feature-designer
Git 写操作：无（未 add/commit/push/branch/stash/checkout/restore/reset/clean）

## 1. 前置基线核实（修改前）

| 项 | 要求值 | 实测值 | 结论 |
|---|---:|---:|---|
| 字节数 | 382472 | 382472 | 一致 |
| 行数（LF 计数） | 3294 | 3294 | 一致 |
| SHA256 | 5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863 | 5BAA8373802F2D349E637135C2114A5344F0E903BBD2EB60058997797B81A863 | 一致 |
| revision | article-management-design-ir01-r30 | article-management-design-ir01-r30 | 一致 |
| 编码/换行 | UTF-8 无 BOM / 纯 LF | 首字节 0x2D（`-`），CR=0 | 一致 |

原生字节快照：`design.before.md`（SHA256 与上表基线完全相同，见 §4）。

## 2. 修改后度量

| 项 | 修改后 |
|---|---:|
| 字节数 | 382562（+90） |
| 行数 | 3294（不变） |
| SHA256 | 63FD0691E1D66326B8EED7CFDFD8017417C60BB628CB961359087242D5D58B64 |
| revision | article-management-design-ir01-r31 |
| last_updated | 2026-11-02（原值即为该值，未改动） |
| 编码/换行 | UTF-8 无 BOM（首字节 0x2D），LF=3294、CR=0 |

## 3. 两处对照（旧文本 → 新文本）

### 3.1 第 1277 行（行号不变）

旧：

```
- 成员可见性还需连接 User.status；容量目标见 r29，目标环境执行计划实测仍 unknown / NOT RUN，不提前复制账号状态。
```

新：

```
- 成员可见性还需连接 User.status；容量目标见 §3.3.1 的 `##### r29 AM-Q2 容量参数确认`，目标环境执行计划实测仍 unknown / NOT RUN，不提前复制账号状态。
```

### 3.2 第 2292 行（行号不变）

旧：

```
7. 关闭超时、terminal 错误或 goroutine 未退出都作为 lifecycle failure 返回，不能伪装为正常完成。worker 的容量、吞吐和告警阈值仍为 TBD（QueryConfig 分页基线与首期规模目标见 r29），但上述调度数字不是 TBD；不引入 MQ、外部 scheduler 或第二个进程入口。
```

新：

```
7. 关闭超时、terminal 错误或 goroutine 未退出都作为 lifecycle failure 返回，不能伪装为正常完成。worker 的容量、吞吐和告警阈值仍为 TBD（QueryConfig 分页基线与首期规模目标见 §3.3.1 的 `##### r29 AM-Q2 容量参数确认`），但上述调度数字不是 TBD；不引入 MQ、外部 scheduler 或第二个进程入口。
```

定位方式：见 §3.3.1 的「##### r29 AM-Q2 容量参数确认」，不写死行号，避免迁块后再次漂移；与被审版 §3.4.6（现 :2323）已有引用写法一致。

### 3.3 元数据

第 5 行：`revision: article-management-design-ir01-r30` → `revision: article-management-design-ir01-r31`。
`last_updated` 已是 `2026-11-02`，按最小改动未触碰。

## 4. 变更面证明

`git diff --no-index --unified=0 design.before.md design.after.md`（全文见 `design.diff`）仅 3 个 hunk：

- `@@ -5 +5 @@`：revision 行
- `@@ -1277 +1277 @@`：第一处引用
- `@@ -2292 +2292 @@`：第二处引用

无其他行增删改，无空行变化，无重排，行数保持 3294。

## 5. 自检：`见 r29` / `见r29` 全量检索

正则 `见\s*r29` 命中数：**0**。不存在仍指向已迁块位置的悬挂引用。

全文 `r29` 共 20 处（含新写入的定位标题），逐条判定：

| 行 | 片段 | 判定 |
|---:|---|---|
| 190 | `AM-Q2（r29 已关闭：…）` | 合法：决策轮次归属，历史事实 |
| 198 | `r29 由用户于 2026-11-02 明确确认…` | 合法：来源叙述 |
| 310 | `r29 部署基线为 200` | 合法：取值来源 |
| 883 | `r29 生产部署基线 20` | 合法：取值来源 |
| 884 | `r29 生产部署基线 100` | 合法：取值来源 |
| 885 | `r29 生产部署基线 200` | 合法：取值来源 |
| 921 | `r29 确认的部署基线 20/100/200` | 合法：取值来源 |
| 961 | `##### r29 AM-Q2 容量参数确认` | 合法：被引用小节标题本身（唯一，计数 1） |
| 965 | `数值…保持 r29 原文` | 合法：修订历史说明 |
| 1026 | `r29 已确认 production 部署基线…` | 合法：取值来源 |
| 1114 | `r29 部署基线 20/100` | 合法：取值来源 |
| 1277 | 见 §3.3.1 的「##### r29 AM-Q2 容量参数确认」 | 合法：本轮修正后的定位引用 |
| 2292 | 见 §3.3.1 的「##### r29 AM-Q2 容量参数确认」 | 合法：本轮修正后的定位引用 |
| 2323 | 见 §3.3.1 的「##### r29 AM-Q2 容量参数确认」（r30 迁入） | 合法：上一轮既有引用 |
| 2769 | `已由 r29 确认为 20/100/200` | 合法：取值来源 |
| 2841 | `已由 r29 确认为 20/100/200` | 合法：取值来源 |
| 2896 | `已由 r29 确认为 20/100/200` | 合法：取值来源 |
| 2966 | `r29 首期部署基线为 DefaultLimit=20…` | 合法：取值来源 |
| 3180 | `已关闭（r29）：DefaultLimit=20…` | 合法：关闭状态归属 |
| 3264 | `r29 已确认 QueryConfig 部署基线 20/100/200…` | 合法：取值来源 |

小节标题 `##### r29 AM-Q2 容量参数确认` 在全文唯一出现（第 961 行，§3.3.1 内），故新引用可无歧义解析。

## 6. 未触碰声明

本轮写入仅限：

- 修改：`.add/project/knowledge/design-workspace/article-management/design.md`（3 行）
- 新建：`.spec/wip/article-management/evidence-am-q2/design-r31/design.before.md`、`design.after.md`、`design.diff`、`report.md`

未修改 `test-plan.md`、`tasks.md`、`plans/*`、`docs/**`、`stories.md` 或任何其他文件。工作区在基线时即存在的其他 `M`/`??` 条目属于此前既有状态，本轮未触碰。

## 7. 性质声明

本产物为本角色自检证据，数值与约束均取自被审版本的既有内容。**未经独立评审确认、非真人签字，不构成人的接收或生产批准。**