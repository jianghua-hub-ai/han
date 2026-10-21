---
name: git-workflow
description: Git 工作流——多人协作 / 操作 SOP / 提交规范三合一。适用于 commit 格式、单仓与跨仓 batch push 顺序、PR / merge 流程、submodule pointer 管理。
---

# Git 工作流（多人协作 + 操作 SOP + 提交规范 三合一）

> **适用**：Git 操作按本次授权和项目已选协作策略执行；固定工作区/分支先确认，后续不得为套 SOP 切换位置或删除其他工作。单仓不加载跨 submodule 步骤；GitHub/PR/合并方式只有项目明确采用才适用。本文示例用 main/origin，执行前取实际 ref/remote，不猜项目配置。
>
> **跨操作系统执行规则**：下方代码块是步骤示意，不要求用户复制到某一种 shell。Host/Agent 执行时必须把每条 Git 操作作为固定可执行文件 + 参数数组 + `cwd` 单独调用，路径作为一个参数传入；不得依赖 `REPO=...`、`$REPO`、`for ...; do`、`$(...)` 或 `&&` 链式命令。批量仓库逐个调用并逐个记录结果；Windows 不需要额外安装 Bash、grep 或 Unix 工具。
>
> **本文是唯一权威**，合并自 2026-04 之前的三份 steering：
> - 旧 `git-collaboration-workflow.md`（多人协作准则：分支命名 / PR / 合并 / rebase）
> - 旧 `git-step-by-step.md`（操作 SOP：单仓 / 多仓 batch / chore follow-up，含 5 铁律 + 翻车诊断）
> - 旧 `git-commit-policy.md`（提交粒度 / 子父仓顺序 / commit 格式）
>
> **章节地图**：
> - §0 仓库结构表
> - §1 七条安全约束
> - §2 项目选择与安全边界
> - §3 分支命名规范
> - §4 Commit 消息格式
> - §5 单仓简单提交 SOP（场景 A）
> - §6 多仓 batch SOP（场景 B，子仓优先）
> - §7 chore follow-up SOP（场景 C）
> - §8 Rebase 操作手册（三场景 + 禁止项）
> - §9 子仓 / 父仓联动准则（4 条硬规则）
> - §10 GitHub 必备配置（一次性）
> - §11 冲突 & 回滚
> - §12 用户验收 5 步清单
> - §13 翻车征兆诊断手册
> - §14 紧急救援工具
> - §15 与 Agent / AI 工具协作边界
> - §16 常见反例 Quick Reference
> - §17 自我修复指令模板
> - §18 FAQ + 速查小卡

---

## 0. 仓库结构

> 🚧 **待对齐**：本段为占位骨架，维度沿用框架；具体值待 本项目 〈仓库组织规划〉拍板后填充（对齐项见 项目入口 §2）。

仓库组织由项目实际配置定义。以下仅为已采用 submodule 时的示例，不要求所有项目如此组织：

| 仓库路径 | 说明 | 是否 submodule |
|---------|------|---------------|
| `.` (根目录) | 主仓库 | 否 |
| `<sub1>/` | `<待填充:sub-repos>` 之一（职责待拍板） | 是 |
| `<sub2>/` | `<待填充:sub-repos>` 之一（职责待拍板） | 是 |
| ... | ... | ... |

---

## 1. 七条铁律（每次操作前心里过一遍）

> 同步只按 §8 的唯一分支合同选择，不按“短任务/自己建的分支”推定可改写。提交、同步、push、PR、合并和清理授权分别核对；方法存在不代表本次全获授权。

### 铁律 1：跨 submodule 操作必用 `git -C <abs-path>`

**为什么**：跨命令 cwd 易丢失（终端 / GUI 切换时），命令打到错的仓上。

```bash
# ❌ 不要
cd <待填充:repo-root>/<sub1>
git push  # 下一行可能 cwd 已变，不知道在哪

# ✅ 要
git -C <待填充:repo-root>/<sub1> push
```

### 铁律 2：commit 后核本任务所有权、实际 diff 与提交

**为什么**：GUI / 工具报告 "已提交" 看似成功，但可能漏 stage 文件——`git log` 数 commit 数量不暴露，只有 `git status` 才看见 working tree 真实状态。

```bash
git -C <abs-path> status --short
# 区分本任务应提交项与他人/无关改动；本任务漏项不能冒称已提交
# 他人 M / ?? / A 保留，不为满足“干净”而暂存、回滚或删除
```

### 铁律 3：push 后必 `ls-remote` 验远端 SHA

**为什么**：`git push` 输出 "Everything up-to-date" 或 "new branch" 都可能骗人——cwd 翻车推到错仓 / 远端没那个分支但报"已最新"。

```bash
git -C <abs-path> ls-remote origin refs/heads/<branch-name>
# ✅ 输出 1 行 SHA + ref name，且 SHA == 本地 git rev-parse HEAD
# ❌ 空输出 = 远端没这个分支（push 实际没成功）
```

### 铁律 4：合并策略取项目决定，父指针必须可取得

**为什么**：submodule 的父指针必须引用实际合入且可取得的子仓 SHA；squash 改写提交身份，不能继续引用被替换的旧分支 SHA。详见 §18 FAQ Q6。

项目明确采用保留祖先链的策略时，在 GitHub 选择 **Create a merge commit**；下文相同命令示例均以此为前提，不覆盖客户已选策略。

### 铁律 5：`<runtime-project-settings>` 该入库，`<runtime-project-settings-local>` 不入库

以项目实际 .gitignore、配置归属和安全约定为准；不假定固定行号。仅本任务有授权且不含秘密的共享配置可精确暂存，不能因任意 settings modified 就带入。

### 铁律 6：同步先 fetch，再核分支关系与共享范围

push 前在获授权的目标仓/分支核远端现状。快进、未共享提交 rebase、共享历史 merge 和经单独批准的重写按 §8 判断；不能以远端变化就默认重写本地历史。对账失败或无法安全解冲突时停止，保留事实。

```bash
git -C <abs-path> fetch origin
git -C <abs-path> log --left-right --oneline HEAD...<实际远端ref>
# 依 §8 选择本次唯一同步方式；不把 fetch 本身当已整合
```

每个动到的仓（主仓 + 每个 submodule）各做一遍。详见 §5 Step A.3 step 3.0 + §8 Rebase 手册。

### 铁律 7：PR 通道堵也绝不直推 main（止步于 push + 给 compare URL）

**为什么**：项目采用 PR 时，先核当前工具通道、身份与授权；通道不可用则止步于已获授权的 push，并提供项目对应 compare URL 交给有权限的人，**绝不**升级成 `git push origin <feat>:main` 直推。用户说“快点 / 都合 / merge”不是绕过已有保护的授权。工具可用且本次明确授权时可代建/合并，否则由有权限的人处理。

```text
每仓 compare URL：https://github.com/<待填充:github-org>/<repo>/compare/main...<branch>?expand=1
顺序：所有 submodule PR 合完 → 主仓 feat 分支 bump 指针 → 主仓 PR
```

---

## 2. 项目协作选择与不变约束

| 维度 | 本项目约定 | 原因 |
|------|----------|------|
| 合并方式 | 项目已选策略；需要保留祖先链时可选 merge commit | 父仓引用实际合入且可取得的最终子仓 SHA；squash 项目不引用被替换的旧分支 SHA |
| main 保护 | 遵守项目已采用的保护/PR/评审策略，不擅自更改 | 不能因工具受阻绕过已有保护或授权 |
| 分支 & commit 带任务 ID | 分支名 + commit subject 都带 `<task-id>` 任务编号 | 便于 blame、追溯验收、关联 tasks 文档 |
| 同步策略 | 只按 §8 的实际提交关系和共享范围选择 | 分支短/长或属于自己，不是重写已共享提交的授权 |
| 强制验证 | 本任务所有权/diff/约定提交和适用测试，fetch 后按 §8 安全同步；配置了 CI 才核其真实结果 | 全树有无关改动不影响本任务如实验收；任务/整体独立评审不能互代 |

---

## 3. 分支命名规范

```
<type>/<task-id>-<kebab-slug>
```

| type | 用途 | 例子 |
|------|------|------|
| `feat` | 新功能 | `feat/<task-id>-<slug>` |
| `fix` | bug 修复 | `fix/<task-id>-<slug>` |
| `refactor` | 重构（无行为变化） | `refactor/<task-id>-<slug>` |
| `docs` | 仅文档 | `docs/agent-teams-contract` |
| `chore` | 构建、依赖、配置 | `chore/upgrade-go-1.24` |
| `test` | 仅测试补充 | `test/<task-id>-<slug>` |

**多仓联动时，同一任务在各仓库使用相同的分支名**（便于跨仓库查看）：

```
<sub1>:   feat/<task-id>-<slug>
<sub2>:   feat/<task-id>-<slug>
主仓:     feat/<task-id>-<slug>
```

**无任务 ID 的改动**（如基础设施整理）：`<type>/<kebab-slug>`，例 `chore/gitattributes-linguist`。

---

## 4. Commit 消息格式

Conventional Commits，subject 结尾带任务 ID：

```
<type>(<scope>): <中文描述> (<task-id>)

<可选 body，解释 why 与影响面>

Task: <task-id>
Refs: <关联 PR / commit，可选>
```

**Type**：`feat` / `fix` / `docs` / `style` / `refactor` / `test` / `chore`
**Scope**：业务 scope 必须取自项目入口登记的服务 / 模块清单；框架层可按项目约定使用 `repo` / `docs` / `test` / `build` / `chore` 等通用类别，不在母本写死项目服务名。
**Subject**：中文描述，≤50 字符，不以句号结尾

**示例**：

```
feat(<scope>): 扩展资源查询响应元信息 (<task-id>)

- 新增可选元信息字段，旧客户端缺省行为保持不变
- 将处理器依赖显式注入，便于测试与替换
- 切换 YAML 统一加载路径（MustLoad + EnvOverridable）

Task: <task-id>
```

> 提交应为可理解的逻辑单元。需要整理 WIP 时，仅自己的未共享提交可按 §8 在授权内交互式 rebase；已共享历史默认保留，不能为美化提交未经确认重写。

> 🔧 **commit -m 消息避免内嵌双引号**：message 里写 `"xxx"` 会跟 `-m "..."` 的外层双引号打架，zsh 报 `unmatched "` 把**整条复合命令在 parse 阶段中止**（好处：不会半执行，状态干净、直接重来即可）。要么用单引号 `-m '...'`，要么消息里改用全角「」/ 去掉引号。

---

## 5. 场景 A：单仓简单提交（最常见）

**适用**：只动一个仓里的几个文件，与其他仓无关。
**例子**：`<sub1>` 改一个组件 / `<sub2>` 修一个 bug。

### Step A.1：起 feat 分支

```bash
# 假设动的是 <sub1>，任务编号 <task-id>
REPO=<待填充:repo-root>/<sub1>

# 1.1 先核已经固定的工作区/分支；本会话已固定则直接复用
git -C $REPO branch --show-current
git -C $REPO status --short
# 1.2 仅尚未固定且获明确初始化授权时，才从核准基线建立用户指定分支
```

**✅ 看到什么算成功**：
- 实际路径/分支与本次约定一致；已存在分支无需重新创建

**❌ 翻车信号 + 修复**：
- "fatal: 'origin/main' is not a commit" → 没 fetch，先 `git -C $REPO fetch origin`
- "branch already exists" → 只读核其用途/修订，复用须在授权内；不直接删除重建

### Step A.2：改代码 → stage → commit

```bash
# 2.1 改代码（用 VS Code 或任何编辑器）

# 2.2 看改了哪些
git -C $REPO status --short
git -C $REPO diff   # 看具体内容

# 2.3 精确 stage（不要 git add . —— 容易带不该入库的）
git -C $REPO add <module>/path/to/source
git -C $REPO add <module>/path/to/test

# 2.4 验 staged 内容
git -C $REPO diff --cached --stat   # 文件清单 + 行数
git -C $REPO diff --cached          # 详细 diff

# 2.5 commit（subject 严格按 §4：<type>(<scope>): <subject> (<task-id>)）
git -C $REPO commit -m "fix(<scope>): 修复资源关闭竞态 (<task-id>)"

# 2.6 ⚠️ 铁律 2 必跑
git -C $REPO status --short
# 本任务应提交文件无遗漏；无关/他人修改保持原样
```

**❌ 翻车信号 + 修复**：

| 看到什么 | 含义 | 怎么办 |
|---------|------|-------|
| Step 2.6 还有 M / ?? 行 | 可能本任务漏项，也可能他人改动 | 核所有权与 diff；仅本任务按约定补提交。amend 也受 §8 共享历史/授权限制 |
| commit 时 hook 报错 | 项目 pre-commit 卡住 | 看错误信息修；不要 `--no-verify` 跳过 |

### Step A.3：push 前同步 + push + 验远端

先核当前固定分支、HEAD、实际 remote 与获授权的发布分支映射。下列 `<实际发布分支>` 必须替换为此次确认的远端分支，不能根据 feat/chore 范例猜；每仓可有不同映射。不允许把 PR 源分支映射成受保护主线来绕过审批。

```bash
# 3.0 本次已授权 push；先 fetch 并按 §8 唯一合同选择同步
git -C $REPO fetch origin
git -C $REPO log --left-right --oneline HEAD...<实际远端ref>
# 核会移动的提交是否已共享；依 §8 完成一种适用同步并验证

# 3.1 push——首推 vs 重推分两种
git -C $REPO push -u origin HEAD:refs/heads/<实际发布分支>   # 只推当前已核HEAD；-u 按项目tracking约定
# 如确需改写远端历史，先满足 §8 单独授权/旧ref保存/精确lease；不在本SOP默认force

# 3.2 ⚠️ 铁律 3 必跑
git -C $REPO ls-remote origin refs/heads/<实际发布分支>
# 必须核该精确远端ref的SHA与本次发布HEAD；竞争变化先报告，不能据此强推
git -C $REPO rev-parse HEAD
```

**同步核验不能省**：fetch 只更新可见远端信息；实际整合按 §8 分支关系处理。仅有未共享后缀的安全 rebase 合法，共享历史默认 merge；不把某一种命令设为所有分支硬要求。

**❌ 翻车信号 + 修复**：

| 看到什么 | 含义 | 怎么办 |
|---------|------|-------|
| `Everything up-to-date` 但你刚 commit | cwd 翻车推到了错仓 | `pwd` 看当前真实 cwd；用 `git -C <绝对路径>` 重推 |
| `src refspec X does not match any` | 本地无该分支（commit 落到了 main 或别处） | `git -C $REPO branch` 列分支 → `git -C $REPO log --oneline -3` 找 commit 在哪个分支上 |
| `rejected (non-fast-forward)` | 远端/本地关系已变或此前判断不完整 | 重新 fetch 对账并按 §8 处理，不能从拒绝自动推导 force 权限 |

### Step A.4：GitHub 创 PR + 合并

#### 🔴 Agent 主会话必做（push 完成后立即输出，不等用户问）

push 成功后 Agent **主动输出三件套**——让用户点链接打开 GitHub web 复制粘贴提交即可：

1. **预填 compare URL**：`https://github.com/<待填充:github-org>/<repo>/compare/main...<branch>?expand=1`
2. **PR title**（一行，复制粘贴用）
3. **PR body**（含 Summary / 改动要点 / Test plan，复制粘贴用）

🔒 **铁律**：不要退化成"给 `gh pr create` 命令让用户终端执行"——PR 代建路径（本地 `gh` CLI / MCP github）在本项目是否可用 🚧 待验证。若两条代建路径都不通，**直接给 compare URL 就是最短路径**（不要反复浪费轮次尝试代建），且绝不直推 main。

#### 用户操作

1. 点 Agent 给的 compare URL 打开
2. 复制 Agent 给的 title 粘贴到标题栏；复制 body 粘贴到描述栏
3. 点 **Create pull request**
4. 合并：下拉合并按钮 → **Create a merge commit** → 确认

**✅ 看到什么算成功**：
- PR 页面右上紫色 **Merged** 徽章
- main 分支多 1 条 Merge commit + N 条 feat 分支 commit（N=你 feat 上 commit 的数量）
- 远端 feat 分支被 GitHub 自动删除（提示 "branch deleted"）

### Step A.5：本地清理

固定会话保持原工作区/分支，不自动切 main 或删分支。核远端合并结果与本任务残留，报告可清理的精确对象；只有另获清理授权且确认无人使用/成果可恢复时处理。不用 SCM 总数归零作为目标。

---

## 6. 场景 B：跨多仓 batch

**适用**：一个功能涉及多仓（如某批次：`<sub1>` + `<sub2>` + `<sub3>` + 主仓）。
**核心原则**：**先 submodule、后主仓**——submodule 各自合入 main 后，主仓再 bump pointer 到合入后新 SHA。
**例子**：跨仓批次 `<batch>` 的实际流程。

### Step B.0：决定 push 策略——一次性 vs 分批

跨多仓 batch 全部 task 完工后 **不一定要一次性 push 全部仓**。两种模式按 batch 规模决定：

| 模式 | 适用 | 优势 | 劣势 |
|---|---|---|---|
| **一次性 push**（默认）| 单 batch ≤ 5 仓 + ≤ 15 task + 强依赖 | 全部仓状态一致；review 集中 | 部分仓回滚牵连全部 |
| **分批 push**（按依赖链）| ≥ 6 仓 OR ≥ 16 task OR 长跨度（多日跑完） | 早分仓的 dev 提前解锁；问题分仓爆发 | 多次 PR review 上下文切换 |

**分批 push 推荐分组策略**（按 [`batch-plan SKILL`](../batch-plan/SKILL.md) 产的依赖图）：

1. **首批**：依赖链根 + 完全无依赖的并行组（如公共契约 / 基础模块）
2. **次批**：依赖首批的业务仓（如各业务微服务）
3. **末批**：依赖业务仓的前端 + 主仓 pointer

**分批 push 必跑前置 Gate**：每批 push 前 Lead 必走 §5.3 4 项验收 + 子仓 build / test PASS + reviewer 局部 PASS（G4-task），不允许"凑齐后批 push 而前批回退"导致 main 上残留 bug。

**主仓 submodule pointer 只动一次**：所有子仓 PR 合 main 后，主仓 bump pointer 一次性指到所有子仓最新 main——**禁止分批 bump pointer**（避免主仓 main 短暂指向某子仓未合 PR 的中间 SHA）。

### Step B.1：所有动到的仓 setup feat 分支

先按本次确已授权的跨仓清单，逐仓核固定绝对路径/分支及所有权。已固定会话不为统一名称切其他 worktree/分支；缺少目标或授权时只交回待协调项。尚未初始化的目标按 Step A.1 处理，不批量套命令重建所有分支。

**✅ 看到什么算成功**：每仓实际绝对路径、当前分支、HEAD 与本次固定约定一致，并各有确认的 remote/发布分支映射；不要求各仓同名或套 feat 前缀。

### Step B.2：各 submodule 改代码 + commit + push（按场景 A 流程逐仓）

每个 submodule（**不含**主仓）按场景 A 的 Step A.2-A.3 走：

```bash
# 例如 <sub1>
SUB=<待填充:repo-root>/<sub1>
# ...改代码...
git -C $SUB add <files>
git -C $SUB commit -m "feat(<scope>): <desc> (<batch>)"
git -C $SUB status --short    # 核本任务所有权与未提交项
git -C $SUB fetch origin
# 按 §8 核共享范围并选择同步方式，验证后才按授权 push
git -C $SUB push -u origin HEAD:refs/heads/<实际发布分支>
git -C $SUB ls-remote origin refs/heads/<实际发布分支>   # 核本子仓已确认映射及发布SHA
```

依次对每个动到的 submodule 重复。

### Step B.3：每个 submodule GitHub 创 PR

为每个动到的 submodule 创 PR（流程同场景 A.4）。

**⚠️ 这一步**：**先全部创建，等待用户/团队 review**，**暂不合并**。

### Step B.4：等所有 submodule PR 都合入各自 main 后，**才**继续

✅ 验证：
逐个对 `<sub1>`、`<sub2>`、`<sub3>` 执行两次独立的 Git 调用：先 `fetch origin`，再读取 `rev-parse origin/main`；每次调用都把对应绝对路径作为 `cwd`，不要用 shell 循环、命令替换或链式执行。

每个 submodule 的远端 main HEAD 应该都是合并后的新 SHA（含一个 `Merge pull request #N` commit）。

### Step B.5：各 submodule 本地同步 main

逐仓 fetch 并核已合入目标 ref 的精确 SHA；在本次授权允许更新 checkout 时按 §8 和当前 worktree 约束处理。不能为 bump 指针擅自切换仍在使用的子仓分支。目标 SHA 已知但 checkout 无权更新时交回负责人，不伪称已同步。

### Step B.6：主仓 bump submodule pointer + 提交 doc 改动

```bash
ROOT=<待填充:repo-root>

# 6.1 主仓仍须符合 Step B.1 核过的固定约定（未在本步骤建分支）
git -C $ROOT branch --show-current

# 6.2 看 status 应该看到几类东西：
#     - M <sub1> / M <sub2> / M <sub3> （3 个 submodule pointer 待更新）
#     - M .spec/tasks/tasks*.md（doc 改动，如有）
#     - M <runtime-project-settings>（如有新加 Bash 白名单）
#     - ?? .spec/plans/<batch>.md（新 plan 文件如有）
git -C $ROOT status --short

# 6.3 stage 所有相关改动
git -C $ROOT add <sub1> <sub2> <sub3>
git -C $ROOT add <本任务精确tasks文件>
git -C $ROOT add .spec/archive/legacy-plans/<batch>.md   # 如有归档 plan
git -C $ROOT add <runtime-project-settings>                 # 仅本任务有授权且安全的变更

# 6.4 验 staged 内容（关键！）
git -C $ROOT diff --cached --stat
git -C $ROOT diff --cached -- <sub1>
# 应该看到：
#   -Subproject commit <旧 SHA>
#   +Subproject commit <Step B.5 拉到的 main HEAD 新 SHA>
```

**❌ 翻车信号**：

| 看到什么 | 含义 | 怎么办 |
|---------|------|-------|
| pointer diff 还是旧 SHA | submodule 没 pull 到新 HEAD | 回到 Step B.5 重做 |
| 多了不期望的文件被 stage | git add 范围太大 | `git -C $ROOT reset HEAD <file>` 取消 stage |

### Step B.7：主仓 commit + push + 创 PR + 合并

```bash
# 7.1 commit
git -C $ROOT commit -m "feat: <batch> 收尾——bump 3 submodule pointer + tasks 标完成 (<batch>)"

# 7.2 ⚠️ 必跑
git -C $ROOT status --short

# 7.3 push（先走 3.0 同步——主仓 origin/main 也可能被别人推过）
git -C $ROOT fetch origin
# 按 §8 核分支关系/共享范围，完成适用同步与验证
git -C $ROOT push -u origin HEAD:refs/heads/<实际发布分支>

# 7.4 ⚠️ 必跑
git -C $ROOT ls-remote origin refs/heads/<实际发布分支>

# 7.5 浏览器创主仓 PR + Create a merge commit + 合并
```

### Step B.8：本地清理（含所有动过的仓）

沿 Step A.5 核精确目标和另行清理授权，固定工作树会话不自动切分支/删除工作树或分支。

**✅ 全完成验收清单**（**5 步必跑**）：
1. 本次各仓路径/分支仍符合固定约定
2. status/diff/staged 证明本任务无漏交；他人改动保留
3. 约定提交与远端结果核对，未授权 push/merge 不冒称已完成
4. 相关测试和任务/整体独立评审分别有真实结果
5. 主仓 gitlink 指向此次确认的可取得且已合入子仓目标 ref 的 SHA，不盲跟更新的远端 HEAD

---

## 7. 场景 C：紧急 chore follow-up PR（漏 commit / 配置入库 等小修补）

**适用**：发现某仓有应入库但未 commit 的改动；某 PR 合后发现要补一两个文件；submodule pointer 需 bump 到新 SHA 等。
**例子**：`<task-id>` 漏 commit 补救 PR。

### Step C.1：起 chore 分支 from main

按 Step A.1 核当前固定工作区/分支和补修范围，不能为改一个遗漏文件自动切 main/新分支；需要另一个交付上下文时先与用户对齐。

**例如**：`chore/<task-id>-menu-seed-followup` / `chore/post-<batch>-cleanup` / `chore/bump-submodules-after-merge`。

### Step C.2：working tree 已经有改动 → 直接 stage + commit + push

工作树往往已经有未 commit 的内容（场景：你刚发现"啊这个文件忘了 commit"）：

```bash
# 2.1 看具体什么改动
git -C $REPO status --short

# 2.2 stage（精确指定文件）
git -C $REPO add <file1> <file2> ...

# 2.3 commit message 写清楚"为什么是 chore"——是补救还是配置同步
git -C $REPO commit -m "chore(<scope>): <描述>（<原因>）"

# 例如：
# chore(<scope>): <task-id> 菜单 yaml seed（漏 commit 补救）
# chore: <batch> 收尾——bump <sub1> pointer + <runtime-project-settings> 入库
# chore(submodules): bump pointers 到各 submodule main HEAD（PR #N 合并后修正）

# 2.4 ⚠️ 必跑
git -C $REPO status --short

# 2.5 push + 验远端
git -C $REPO push -u origin HEAD:refs/heads/<实际发布分支>
git -C $REPO ls-remote origin refs/heads/<实际发布分支>
```

### Step C.3：GitHub 创 chore PR + 合并

同场景 A.4。chore PR review 负担小，可快速合。

### Step C.4：本地清理

同 Step A.5，不因 follow-up 完成就取得删除授权。

### Step C.5：如涉及 submodule pointer 变化，主仓也起 chore PR

如果 chore PR 是 submodule 内的，submodule main 前进 → 主仓 pointer 滞后。需要后续主仓 chore PR bump：

```bash
# 沿当前已固定主仓分支处理；需另起交付上下文时先对齐，不能在这里自动切换
ROOT=<待填充:repo-root>
git -C $ROOT branch --show-current
# 先按 B.5 核并获得本次精确 gitlink 更新授权
git -C $ROOT add <出问题的 submodule>   # bump pointer
git -C $ROOT commit -m "chore: bump <submodule> pointer 到 main HEAD"
git -C $ROOT push -u origin HEAD:refs/heads/<实际发布分支>
git -C $ROOT ls-remote origin refs/heads/<实际发布分支>
# 创 PR + Create a merge commit + 合并
# 本地清理
```

---

## 8. Rebase 操作手册

### 同步与历史修改的唯一合同

先 fetch，核当前/目标/远端 ref 及会移动的精确提交。不能从分支名、任务长短或“自己创建”推定未共享；不清楚是否供他人使用就先说明并协调。快进优先：能仅向前移动时按已选目标使用 `merge --ff-only`，不改既有提交。

| 实际状态 | 允许的同步 | 发布处理 |
| --- | --- | --- |
| 只有自己的未共享提交需移到已核目标之后 | 在授权同步范围内 rebase；保留测试与真实结果 | 正常 push，不需要 force |
| 同分支远端前进，本地只有未共享本地后缀 | `pull --rebase` 只搬该后缀、未重写远端共享提交时合法 | 正常 push；冲突无法安全解决可 abort |
| 需要改写已共享提交或他人基线 | 默认 merge，不重写共享历史 | 正常 push，保留别人的原提交 |
| 确需改写已共享历史 | 单独明确授权、协调受影响人，保存精确旧本地/远端 ref 和可恢复证据；在明确静默范围操作 | 以预期远端 ref 的显式 `--force-with-lease=<ref>:<expected-old-sha>` 校验竞争；变化则停，不更新期望值硬推 |

lease 只检查远端 ref 是否仍符合预期，不授予重写权限，也不证明没有丢失他人的逻辑。amend、交互式 rebase、drop 和恢复旧 ref 同样遵守此合同。同步/修复期间保持固定工作区/分支；未提交改动或其他写者影响安全时保留现场并协调，不自动 stash/clean/reset 他人工作。

### 场景 A：feature 分支同步 main（最常见）

你在已固定分支上工作，目标主线前进。按上表核实际共享范围，不能按 feature 名称选择 rebase。以下仅演示“自己的未共享提交”分支：

```bash
git fetch origin
git rebase origin/main

# 有冲突：手动改 → git add → git rebase --continue
# 放弃：git rebase --abort

# 本例未改写已共享提交，正常 push（本次已获授权时）
git push
```

若核明已有共享提交会被重写，退出本例，按上表 merge 或先取得单独重写授权。

### 场景 B：整理自己的 WIP commit（PR 前梳理）

需要整理 WIP 时，只针对已核明自己的未共享提交且获授权的范围：

```bash
git rebase -i origin/main
# 编辑列表：pick 保留，squash 合并到前一行，reword 改消息，drop 删掉
git push  # 仅未共享提交整理后的正常发布；不默认 force
```

> 可读历史重要，但不能优先于共享历史安全；未授权改写时保留原提交，以补充提交/PR 说明提供上下文。

### 场景 C：rebase 失败回滚

```bash
git rebase --abort
# 如果已经推坏了：
git reflog                    # 找 rebase 前的 HEAD
# 核精确旧ref和可恢复证据；恢复方式仍按本节唯一合同，不以“自己的分支”推定可重写
# 远端需要改写时须单独确认及绑定预期旧ref的精确lease；没有则仅提出恢复方案
```

### 禁止项

- ❌ 未经单独明确授权、协调与旧ref保护就重写共享提交；合法仅搬未共享后缀的 rebase 不混同此情形
- ❌ rebase 后用 `git push --force` 而不用 `--force-with-lease` — 容易覆盖掉别人的 push
- ❌ PR 合入 main 后，再在已删除的 feature 分支上 rebase + push — 产生孤立分支

---

## 9. 子仓 / 父仓联动准则 🔒

**硬规则 1 — 子仓 PR 必须先合**
父仓 PR 的 gitlink 必须指向**子仓 main 上已合入的 commit SHA**，不得指向子仓 feature 分支的 SHA。

**硬规则 2 — gitlink bump 独立成 PR**
父仓更新 submodule 指针不要和主仓代码变更混。主仓的 bump commit 作为独立 PR，subject 明确写 `chore(repo): 合入 <taskId> <描述>`。

**硬规则 3 — 原子性权衡**
一个任务跨 M 个子仓 → M 个子仓 PR + 1 个父仓 PR = M+1 个 PR。虽然 PR 数多，但每个 PR 可独立 review / 回滚 / 冒烟；**不要为了少开 PR 把跨仓改动塞进一个"大爆炸" PR**。

**硬规则 4 — 父仓 PR 要等所有子仓 PR 都合入**
不要在还有子仓 PR pending 时提交父仓 gitlink PR。否则父仓合入时 gitlink 指向的子仓 SHA 可能在子仓 rebase 时被改写，出现**gitlink 指向游离 commit**的情况。

---

## 10. GitHub 必备配置（每个仓库一次性）

本节仅是项目明确采用 GitHub 与 merge-commit 策略后的配置参考，不是默认改组织设置的命令。核实际保护/评审/CI 要求并另获管理员授权，不能按下列示例关闭已有更严格控制。

### 10.1 Organization 默认 merge 按钮

`https://github.com/organizations/<待填充:github-org>/settings/repository-defaults`
- ✅ **Allow merge commits**（项目首选 — Create a merge commit）
- ❌ Disable "Allow squash merging"（避免误点导致悬空 SHA）
- ❌ Disable "Allow rebase merging"
- Default commit message when merging: "Pull request title and commit details"

### 10.2 每个仓库的 Branch Protection（main）

`https://github.com/<待填充:github-org>/<repo>/settings/branches` → 给 `main` 加规则：

- ✅ Require a pull request before merging
  - ❌ Require approvals（过渡期不勾）
  - ✅ Dismiss stale pull request approvals when new commits are pushed（未来启用 Review 时有用）
- ❌ Require linear history（**关闭**——Create a merge commit 会产生 merge commit，linear history 与之冲突）
- ✅ Do not allow bypassing the above settings
- ❌ Allow force pushes（关）
- ❌ Allow deletions（关）

**所有仓库（主仓 + 全部 submodule）都要配一次**（可以写脚本，或用 GitHub 的 Repository Rulesets 按 org 模板批量应用）。

---

## 11. 冲突 & 回滚

### 多人协作冲突场景速查

| 场景 | 何时发生 | 处理 |
|---|---|---|
| **origin/main 在开发期间前进** | 目标基线变化 | fetch 后按 §8 选择；不默认重写共享历史 |
| **同一 feature 分支多人推** | 结对/接力 | 核远端共享提交和本地未共享后缀，沿 §8 安全同步；未知写者先协调 |
| **PR 创建后 main 又前进** | 等待评审期间 | 核真实冲突/项目保护，按 §8 整合并回归，不从 update 提示推定重写授权 |
| **两个 PR 改同一文件先后合** | 文件重叠 | 比较两方预期/变更，精确解决并回归；无法安全确定时 abort/交回 |
| **两人并发 bump 同一 submodule 指针** | 多人各自动同一子仓 | 比较两方目标、祖先关系和实际成果；均已合入也不自动择一，确认整合后目标再按下方修正 |

fetch 提供可见远端事实，但不能消除语义冲突或其他机器尚未发布的变更。整合仍需比较两方目标、实际 diff 和回归；同步与重写权限唯一按 §8 判断。

### 本地分支跟随远端

依本次同步授权和并行写入情况选择时机；不因看到远端前进就中断其他写者强行同步。执行方式直达 §8，不在此另写短/长分支规则。冲突能在授权和证据内解决才 continue，否则允许 abort 并明确剩余差异。

**submodule 同步核对**：父仓同步后的 `M <sub>` 可能是 gitlink 与子仓 checkout 不同，也可能是子仓自己的工作，不能直接认定“不是你的改动”。先核父记录、子仓 HEAD/status 和目标修订：

- ✅ 只有已确认子仓未被其他工作占用且本次允许更新 checkout 时，才按精确目标运行项目既有同步操作；否则报告差异并保留
- ❌ **绝不 `git add <sub>`**——那会把指针倒回旧值。核对：父仓同步后记录的指针应 `== git ls-tree origin/main <sub>`（用 `git ls-tree HEAD <sub>` vs `git ls-tree origin/main <sub>` 对比确认一致）

### feature 分支 rebase 冲突

手动改文件 → `git add <file>` → `git rebase --continue`。放弃 `git rebase --abort`。

### main 上合入后才发现有问题

在已授权且固定的工作区/分支修复，按项目采用的保护流程交付回滚 PR；必要时先确认回滚范围/依赖和 merge commit 的主线参数：
```bash
git revert <bad-commit-sha> --no-edit
# 精确验证/提交，按实际授权 push 当前分支并交 PR；不直推 main
```

**绝不在 main 上直接 `reset --hard` 或 `push --force`**，会破坏其他人的 clone。

### 父仓合入的 gitlink 指错了子仓 commit

1. 确认子仓真正想要的 SHA
2. 核固定工作区与子仓占用、更新权限，选定可取得且已合入目标 ref 的精确 SHA；按本项目已授权指针更新方法操作，不自动追 remote 最新或切别人的子仓分支
3. 开修正 PR，subject：`fix(repo): 修正 <submodule> gitlink 指针 (<task-id>)`

---

## 12. 用户验收 5 步清单（每次 Agent 报告"PR 已 push 等你合"时跑）

```
[1] 浏览器看 PR 页面
    - "Files changed" 数 + 行数对得上 commit message 吗？
    - CI 跑过了吗（如配置）？

[2] 合并 PR：选 Create a merge commit

[3] 核远端结果与实际同步权限
    git -C <abs-path> fetch origin
    按 §8 与固定工作区约定选择本次同步，未授权只报告

[4] 终端验
    git -C <abs-path> status --short          # 核本任务，不清理他人
    git -C <abs-path> branch --show-current  # 应仍是固定分支
    git -C <abs-path> log --oneline -3        # 对照约定提交与远端证据

[5] VS Code SCM 面板看
    本任务改动与证据是否相符；UI总数/分支数量不作清理目标
```

任何一步对不上 → 暂停操作 + 告诉 Agent 看到的具体输出，**不要自己 reset / branch -D / clean -fd 来"修"**。

---

## 13. 翻车征兆诊断手册

### 征兆 1：VS Code SCM 显示某仓"S"或"m"，你以为没改过

**S** = Submodule pointer modified（pointer 变了）
**m** = Submodule has dirty working tree（pointer 没变，但内部有未提交改动）

**诊断**：
```bash
git -C <待填充:repo-root>/<submodule> status --short
```

可能原因：
1. **build 临时产物**（如 `mock-trait` binary）→ 按当前 Host 的文件删除能力移除 + 考虑加 submodule 的 .gitignore
2. **agent / GUI "假装 commit"**（modified 文件没 stage 进 commit）→ 看 diff 决定 stage 进现有 commit 还是另起 chore PR
3. **submodule HEAD 与约定位置不一致** → 核使用者和原任务，不自动切 main；需要变更位置先协调

### 征兆 2：`git push` 输出 "Everything up-to-date" 但你刚 commit 了

**几乎肯定是 cwd 翻车**——push 命令打到了错的仓。

```bash
pwd                                                            # 看 shell 在哪
git -C <待填充:repo-root>/<目标仓> log -3      # 看目标仓真实状态
```

修复：用 `git -C <abs-path>` 重新 push。

### 征兆 3：`git push` 输出 "src refspec X does not match any"

意思是本地不存在叫 X 的分支 → commit 落到了别的分支。

```bash
git -C $REPO branch                       # 列所有本地分支
git -C $REPO log --oneline -5             # 顶部 commit 在哪个分支
```

**典型场景**：以为在 feat/xxx 上 commit，实际在 main 上 commit 了。修复：
```bash
# 只读核实际提交和远端关系，再向用户说明最小恢复方案
git -C $REPO log --oneline -5
git -C $REPO reflog --date=iso
# 未获授权不切分支/复位main/重写；恢复仍按 §8 保护原ref和他人工作
```

### 征兆 4：fresh clone 主仓后 `git submodule update` 报 "object not advertised"

主仓 main 上的 submodule pointer 指向了**不存在的 SHA**（feat 分支已删，squash 后 SHA 不在 main 上，pointer 悬空）。

**修复**：起 chore PR bump 主仓 pointer 到各 submodule **main HEAD**（场景 C.5 流程）。

### 征兆 5：`git status` 一堆 untracked 文件你不认识

可能：
1. **node_modules / dist / build / .next** → 核是否真实生成物及本次所有权，再提出适用 .gitignore 变更，不按名字直接删
2. **临时编译产物**（.exe / 二进制）→ 确认无引用且获授权后回收精确对象，优先可恢复处理
3. **agent 生成的临时文件** → 核本任务归属/引用/保留条件和清理授权，不因文件名或来源自动删

---

## 14. 紧急救援工具

### 工具 1：`git reflog`（本地可用历史）

每仓有自己的本地 reflog，可用范围由记录/过期/清理设置决定；缺历史不能承诺一定找回。

```bash
git -C $REPO reflog                           # 列最近
git -C $REPO reflog --date=iso                # 带时间戳
```

恢复某个被删的 commit：
```bash
git -C $REPO branch recovery <sha>            # 从那个 commit 建分支
git -C $REPO cherry-pick <sha>                # 应用到当前分支
```

### 工具 2：`git fsck --lost-found`

列所有"游离对象"——commit 没在任何分支上但还没被 GC 清理。

```bash
git -C $REPO fsck --lost-found
git -C $REPO show <sha>                       # 看那个 commit 内容
```

### 工具 3：动手前打 tag 备份

危险操作前按 §8 核精确目标、授权与共享范围，在不覆盖既有 ref 的名字下保存旧提交；备份 ref 不保存未提交文件。恢复应核当前修改与旧目标再提出精确操作，不能因为有 tag 就默认 hard reset；备份的清理也需范围确认。

### 工具 4：`git stash` 临时保存

```bash
git -C $REPO stash list                        # 先核已有条目
# 获授权且确需暂存本任务文件时，使用精确pathspec和可辨认标识
# 恢复用固定条目核diff后apply；验证恢复成功再经授权drop，不盲pop最近一条
```

---

## 15. 与 Agent / AI 工具协作的边界

### Agent 做什么 / 不做什么

**做**：
- ✅ 读代码、调研、写设计文档
- ✅ 派 agent 实现子项 + 验证
- ✅ 跨 submodule commit + push（用 `git -C`）
- ✅ 仅按 §8 对已授权精确历史范围 amend/重写；“自己创建”不是共享历史的重写授权
- ✅ 写 chore PR 修自己的 follow-up 问题

**不做**：
- ❌ 任何 commit 直推 main
- ❌ force-push 到 main / reset --hard 远端 main
- ❌ 删除任何远端分支（除非用户明确说删）
- ❌ 未核真实工具通道、身份与本次授权就代建或合并 PR；不可用时交有权限的人处理

🔴 **PR 通道受阻 ≠ 直推授权**：当 PR 代建路径不可用（如 PR API 报错、CLI 未认证）时，**正确的 fallback 是止步于 push + 给用户 compare URL**（每仓一条），**绝不升级为 `git push origin feat:main` 直推**。用户说的"快点 / 都合 / 合了它"是"想合并"的意愿，**不是**"绕过 PR 直推 main"的授权——合并动作仍按项目约定由有权限的人完成。多仓项目通常按**依赖仓 PR 合完 → 主仓分支更新指针 → 主仓 PR**顺序推进。

### 出现争议时

- **Agent 报告 vs 实际状态不符** → 以 `git status` / `ls-remote` 实际为准
- **GUI 显示 vs 命令行不符** → 命令行为准（GUI 可能显示缓存）
- **不确定时** → 跑 `git reflog` 看历史，**绝对不要 reset --hard / branch -D 来"修"问题**

---

## 16. 常见反例 Quick Reference

| 你看到 | 别做 | 该做 |
|-------|------|------|
| `git status` 显示 `?? <binary>` | 不要凭名字提交/删除 | 核来源、所有权和引用，按本任务清理授权处理 |
| `git push` 输出 "Everything up-to-date" 但刚 commit | 不要再 push | `pwd` + `git -C <对仓> log` 查 cwd |
| `git push` 输出 "rejected (non-fast-forward)" | 不要自动 force | fetch 后核两端历史，按 §8 分支合同处理；失败本身不增加权限 |
| GUI 显示 "更改 9" 但你以为改完了 | 不要忽略 | 进每个仓 `git status --short` 看具体 |
| 想"清理"红色叉号但不知道是什么 | 不要 `git clean -fd` | 先读 status/diff/所有权与引用，不把 untracked 当可删 |
| commit 落到不符约定的分支 | 不要 `reset --hard` | 只读核精确ref/提交，保护现场并与用户对齐恢复范围 |
| Agent 报告 commit sha 但 GUI 看到 modified | 不要相信 sha | `git status --short` 才是真相 |
| `git rebase --abort` 后想推但 push 拒绝 | 不要 force push | 核当前 HEAD/远端关系与 reflog，依 §8 处理；abort 不自动消除原分叉或授予回退权限 |

---

## 17. 自我修复指令模板

实在不会处理时把这段贴给 Agent：

```
我在 <仓名> 看到了：
<把 git status / git branch / git log -3 的具体输出粘进来>

我不确定怎么处理。请帮我：
1. 解释每行输出的含义
2. 判断哪些是真改动需要保留、哪些是杂物可以删
3. 给出每个文件具体的处理命令
4. 如果涉及代码丢失风险，先建议 git tag 备份
```

---

## 18. FAQ + 速查小卡

### FAQ

**Q1: 任务只改文档，也要开分支和 PR 吗？**
A: 是。main 禁止直推。但可以用 `docs/` 分支类型，PR review 负担小。

**Q2: 我一个人改，为什么不能直接 push？**
A: 因为**未来不止您一人**。流程是肌肉记忆，早养成比后补救便宜。且 PR 页面对 code review / 讨论 / 回溯有天然支持。

**Q3: 跨仓改 M=5 个子仓，5 个 PR 太累了？**
A: 能拆就拆，不能拆就合。5 个 PR 累但**每个都能被独立读懂**。合成一个 20 文件的大 PR 看起来"一次搞定"，review 时根本看不动，埋雷成本更高。

**Q4: CI 没跑完能 merge 吗？**
A: 本文档不定死（当前未必都接了 CI）。建议：一旦某仓接了 Actions，立即把"Require status checks to pass before merging"开上。

**Q5: 我本地的 submodule HEAD 和远端 main 不一致，正常吗？**
A: 主仓的 gitlink 记录的是**某个时刻子仓 main 的 SHA**，之后子仓 main 可能又前进了。`git submodule update --init` 会把本地 submodule 拉到 gitlink 指向的那个 SHA（比远端 main 可能旧）。这是**正常**的，不要随意 `--remote` 跟随，除非你正在做 gitlink bump PR。

**Q6: 为什么选 Create a merge commit 不选 Squash？**
A: 已采用 submodule 且需要保留子仓原提交祖先链时，merge commit 有利于追溯。采用 squash 的项目须让父仓引用实际已合入、可取得的最终 SHA，不能仍指旧分支提交。合并策略取项目决定；整理提交依 §8，不为可读性强制重写共享历史。

**Q7: 子仓库提交后主仓库显示 "modified content"？**
A: 先区分子仓未提交内容和 gitlink 差异，核目标与所有权；只有本任务已确认要更新父指针且子仓目标可取得时才精确暂存，不能把 modified 自动当 bump 授权。

**Q8: 提交时提示 "pathspec is in submodule"？**
A: 需要先进入子仓库目录（或用 `git -C <submodule-path>`）再执行 git 命令。

### 速查小卡

```bash
# 开始任务：先核固定位置；此卡不新建/切换分支
git -C <abs-path> branch --show-current
git -C <abs-path> status --short
# 开发 → commit → push
git -C <abs-path> push -u origin <当前已确认分支>
# 依项目已选策略及本次授权开/合 PR；不自动删分支

# 跟进目标主线：fetch 后按 §8 选择，不默认rebase/force
git -C <abs-path> fetch origin

# 父仓 gitlink bump（所有子仓 PR 合完之后）
# 仅本次允许更新且无人占用的目标子仓；按B.5核精确SHA，不自动追最新
git -C <abs-path> add <sub>
git -C <abs-path> commit -m "chore(repo): 合入 <描述> (T-0001)"
git -C <abs-path> push -u origin <当前已确认分支>
# 依项目已选策略及本次授权开/合 PR；精确清理另行核授权
```

---

**最后更新**：2026-05-06，三合一整合（git-collaboration-workflow + git-step-by-step + git-commit-policy）。后续 batch 中遇到新翻车场景，往 §13 "翻车征兆诊断手册" 追加。
