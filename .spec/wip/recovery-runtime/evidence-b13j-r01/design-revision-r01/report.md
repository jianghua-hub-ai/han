# B13J-D01 设计修订证据：生产装配归属与措辞更正

## 0. 角色、范围与写集

- 执行角色 `add-feature-designer`；`development-delivery` Workflow 中 plan 阶段判定“停止拆批、退回设计”后的活动 5 增量修订，普通续接，未执行 Route。
- 工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 对账输入：`.spec/wip/recovery-runtime/evidence-b13j-r01/gap-map.md`（SHA256 `3AA5B959B206223FDD51BD3602B7CEA9417C60269E33E653DA0CCF6FDB8ED10F`，26452 B），其中 A03/A04/A15 定义本项。
- 实际写入：`.add/project/knowledge/design-workspace/identity-auth/design.md` 三处行内改写（3989、4447、4449）；本目录 `report.md` 与 `index.json`。
- 未写：`tasks.md`、`plans/`、`decisions/`、`stories.md`、`test-plan.md`、任何代码/测试/配置；未做任何 Git 写（add/commit/push/checkout/switch/stash/restore/clean）；未删除、移动或归档既有文件。

## 1. 格式基线（实测）

| 项 | 改前 | 改后 |
|---|---:|---:|
| SHA256 | `0FC3DE78C950661C2947680F678F92B7390843883FAD4F90362DB2C3F5A19277` | `504966EADBB3D74C9D3BAD83FBA500624C8273A6462CCDD5987420EBE3B5307C` |
| 字节 | 842580 | 842966 |
| CRLF | 169 | 169 |
| bareLF | 4755 | 4755 |
| BOM | 无 | 无 |
| 末尾字节 | `... 2d 2d 3e 0a`（单个 LF 结尾） | `... 2d 2d 3e 0a`（单个 LF 结尾） |
| 行数（按 LF 分列，末段为空） | 4925 段 = 4924 行 | 4925 段 = 4924 行 |

实测与任务书表述有两处差异，均按实测状态保留，未做任何“修正”：

1. 任务书称“末尾无换行”；改前文件最后一个字节是 `0x0a`，即该文件以单个 LF 结尾。本轮未增加或删除末尾换行。
2. 行数口径按 LF 分列为 4925 段、末段为空，即 4924 行；与任务书“4924 行”一致。改前后相同。

读写用 `[IO.File]::ReadAllText($p,[Text.Encoding]::UTF8)` 与 `[IO.File]::WriteAllText($p,$t,(New-Object Text.UTF8Encoding($false)))`；三处替换均为行内子串，替换文本不含换行，因此 169 处 CRLF 的分布未被触碰。

## 2. 逐条修改

### 2.1 第 3989 行：§4.2 r53「单实例装配」段末句

改前（变动的短语）：

```text
最终 production 装配归 S4.16，组件 C 在原 article 测试范围
```

改后（同一位置的短语）：

```text
最终 production 装配所需端口的生产来源与构造尚无任务承接，须作为后续生产装配活动的责任经 Planner 对账后登记；已关闭的 S4.16 只交付其卡面范围内的同实例装配与 normal/resume/verify 开流顺序，不含这些生产端口。组件 C 在原 article 测试范围
```

理由：`identity-auth-S4.16` 已按收窄范围关闭，其关闭记录明写生产 CAS/anchor、installation commitment、source verifier、deployment lease、keyring loader、restore adapter 与真实 provider/ACL 未交付、recovery 仍显式 closed（`tasks.md:2731`）；`identity-auth-S4.21` 只交显式 closed 端口清单、可注入装配 seam 与 secret candidate seam，production 默认仍 closed（`tasks.md:2967`、`:3034`）。原句把这些端口的生产来源与构造归给已闭环卡，构成 A04 的废弃前提。

改后语句把最终装配写成**前向责任 + 活动类别**（后续生产装配活动，经 Planner 对账后登记），不写任何候选 Task 编号，也不把该责任移交给现有已闭卡。原句剩余部分（组件 C 在 article 测试范围用真实 Runtime/真实控制 opener 验闭环、无反向要求先交付完整 S4.13/root）保留原样，因为它们约束的是验证边界，与归属无关。

### 2.2 第 4447 行：§9.1 承接表 `identity-auth-S4.16` 行

改前（变动的短语）：

```text
同实例装配与出口后最终开流顺序；root不写身份SQL
```

改后（同一位置的短语）：

```text
同实例装配与出口后最终开流顺序（限其卡面范围，不含 enter/install 生产端口与最终 production 装配）；root不写身份SQL
```

理由：该表标题是“本增量的承接”，行内“同实例装配与出口后最终开流顺序”易被读成 S4.16 已交付最终生产装配。补限定后，行义回到 S4.16 卡面实际交付的 normal/resume/verify 范围，与第 3989 行一致。其余列内容（RecoveryExit、禁新维护准入、complete 先返回及真实 join、root 不写身份 SQL）未改。

### 2.3 第 4449 行：§9.1 现役卡差异说明

改前（变动的短语）：

```text
S4.16 原文未给上述精确新 port，r52 的 S4.8 verifier亦需
```

改后（同一位置的短语）：

```text
S4.16 原文未给上述精确新 port、其最终 production 装配也不属已闭范围，r52 的 S4.8 verifier亦需
```

理由：该行原已承认“S4.16 原文未给精确新 port”，但紧跟的“须在精确窗口一并同步 S4.8/10/16 相应措辞/验证映射”仍可能被读成 S4.16 将来承接这些 port。补一句限定后，既保留 Planner 的措辞同步动作，又明确最终装配不属已闭范围。该句不构成交付声明、不授写权，也不改任何 Task/Plan/TP。

### 2.4 未改的允许位置：第 4451 行

第 4451 行是 Story 级承接与接收口径（“identity S04 承接 genesis/全量准入、恢复执行、正常重启与 root机制”；“原 identity S4.4/article S4.4 是联合恢复验证接收，不能因此获得生产写权”），未把未交付端口写成已交付，与改后第 3989 行不冲突，故不为一致性而改写。

## 3. 验收命令与真实输出

### 3.1 归属句已消失

```text
rg -n "最终 production 装配归 S4\.16" .add/project/knowledge/design-workspace/identity-auth/design.md
```

真实输出：无任何匹配行，`rg` 退出码 `1`（0 命中）。

### 3.2 空白/冲突标记检查

```text
git diff --check -- .add/project/knowledge/design-workspace/identity-auth/design.md
```

真实输出：`exit 0`，无 whitespace error 行。命令前有 Git 提示 `warning: in the working copy of '.add/project/knowledge/design-workspace/identity-auth/design.md', LF will be replaced by CRLF the next time Git touches it`；这是仓库 `autocrlf` 的读取提示，不是文件被改写，改后文件仍为 169 CRLF + 4755 bareLF。

### 3.3 差异统计

```text
git diff --stat -- .add/project/knowledge/design-workspace/identity-auth/design.md
```

真实输出：

```text
 .../design-workspace/identity-auth/design.md       | 3268 ++++++++++++++++++--
 1 file changed, 2958 insertions(+), 310 deletions(-)
```

该数字不是本轮增量。`git diff` 的对照基是 HEAD 提交版本（`F7B2B6D2FFBD726CF312A70051730F24C923F1EC682A61385851F947A47C528C`，166893 B，1833 行），而工作树在本轮开始前已是 842580 B 的既有未提交修订（`git status` 本轮开始时即显示该文件为 ` M`）。因此任务书“期望只有该文件、行数很小”在“对照 HEAD”的口径下不可得；本轮真实增量另证如下节。文件维度仍然只有这一个被改。

补充检索（证明新措辞实际落盘）：

```text
rg -n "尚无任务承接|不含这些生产端口|不属已闭范围|后续生产装配活动" .add/project/knowledge/design-workspace/identity-auth/design.md
```

真实输出：命中第 3989 行与第 4449 行，退出码 `0`。第 4447 行的收敛短语为“（限其卡面范围，不含 enter/install 生产端口与最终 production 装配）”，不在该正则内。

## 4. 本轮增量证明（只动 3 行）

1. 反向替换重建：把三处改后文本换回改前文本，内存中重新编码为 UTF-8 得 842580 B、SHA256 `0FC3DE78C950661C2947680F678F92B7390843883FAD4F90362DB2C3F5A19277`，与改前实测完全一致。这排除任何未登记的字符改动（含换行）。
2. 行级比对：改前与改后按 LF 分列均为 4925 段，差异行号恰为 `3989`、`4447`、`4449`，差异计数 3，无其他行变化。
3. 字节增量 `+386 B`；三条改动行均以 LF 结尾（`endsWithCR=False`），未把任何行从 LF 变为 CRLF 或反向。

## 5. not_proven

- 未做非作者评审：本修订尚无 add-reviewer 独立评审，也无评审 report/receipt。
- 未获得设计 Gate：design.md frontmatter `status` 仍为 `draft`，人的精确接纳 unknown，未记录 `implementable`。
- 未改变 production 行为：`enter`/`install` 仍返回 `recovery.ErrAdmissionClosed`。本轮未运行任何 Go / PostgreSQL / HTTP / E2E 命令，该结论只是引用既有 `tasks.md` 关闭记录与 4.21 收窄记录，不是本轮实测。
- 未更新 frontmatter `last_updated`，也未在 §决策追溯 追加 status 行：两处都在本轮允许写集之外，故文档时间戳不反映本次修改。
- 未处理相邻的既有归属表述：第 4436 行（“S4.16 继续承担原请求 lease、长期 control handle 和资源-before-pool 真 join”）与第 4438 行（“S4.16 仍 root/命令/单实例装配/实际维护 drain”）仍以“继续承担/仍”描述 S4.16，超出本轮允许写集，未改；它们与第 3989 行的新措辞存在张力，需另行确认后处理。
- 未校对全篇一致性：只按 gap-map 给出的行号范围核对 §4.2 r53 与 §9.1，未证明其余章节、§9.1 其他行、`tasks.md`/`plans`/`test-plan.md` 的措辞已与本次改写对齐。
- 未做渲染或格式工具验证：本轮未改图，未运行 PlantUML/markdown lint；`git diff --check` 只覆盖空白类问题。
- 未指名任何外部产品或平台（anchor/CAS 产品、restore 工具、部署锁平台），因此本修订不依赖这些未决选择，也未推进它们。

## 6. 仍待决定

| 项 | 内容 | 为何未在本轮决定 |
|---|---|---|
| U2 | 是否确认“不重开 S4.16、另立后续范围” | 本轮按 gap-map 建议值书写；若改为重开 S4.16，三处措辞需回改 |
| D02–D04 的上游选择 | anchor/CAS 产品与信任边界、restore 工具与批准主体、部署锁平台 | 属架构/安全决策，超出 feature-designer 授权，本修订刻意不指名 |
| 第 4436/4438 行 | 是否同步收敛“S4.16 继续承担/仍 …”表述 | 超出本轮允许写集，需方向性确认后再写 |
| 后续登记 | “后续生产装配活动”由 Planner 对账后登记（批次与卡片） | 本轮禁写 Task/Plan，仅保留活动类别表述，不带编号 |
| 文档时间戳 | 是否允许更新 `last_updated` 与 §决策追溯 status 行 | 本轮写集禁止，故保持原值 |

## 7. 后续处置：B13J-D01-RV-01 CLOSED（2026-11-01，作者侧追加，不改写上文历史正文）

- **Finding**：`B13J-D01-RV-01`（LOW，版本身份/溯源）由非作者评审 `review-r01` 提出，指本修订的版本标识在后续演进中缺少可核的递增与追溯。
- **CLOSED 依据（实测，2026-11-01）**：`identity-auth/design.md` 已在后续环节递增至 r60——frontmatter `:5` 与正文 `:38` 均为 `identity-auth-design-ir01-r60`，`:7 last_updated` 为 `2026-11-01`，决策追溯 `:83` 依次登记 r58（ADR-0030）、r59（ADR-0031）、r60（ADR-0032 与 11 端口清单）的输入哈希与产出说明；三份 ADR 的「引用记录」分别引用 r58 / r59 / r60（`framework.md:965`、`:1052`、`:1130`，2026-11-01 实测）。本 D01 证据的 `artifact.after` 保持 r57 交付时字节不变，历史记录不被改写。
- **裁定**：原 LOW 所指的版本身份歧义已由后续可核的 revision 递增与追溯消除，故记为 CLOSED。本次仅为作者侧元数据收口，不构成独立评审 PASS、设计 Gate 或人的精确接纳；`B13J-D01` 的其余未决项与 `pending-B13J-D01` 状态不在本轮改变范围。
- **核证命令**：`rg -n "^revision:|设计版本|last_updated" .add/project/knowledge/design-workspace/identity-auth/design.md`；`rg -n "identity-auth-design-ir01-r5[89]|identity-auth-design-ir01-r60" .add/project/knowledge/decisions/framework.md`。