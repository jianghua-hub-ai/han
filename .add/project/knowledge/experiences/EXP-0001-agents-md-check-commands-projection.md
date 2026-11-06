# EXP-0001 — AGENTS.md「项目检查命令」投影未随 Manifest 刷新

**状态：proposed（候选，未采用；不外传）。** 2026-11-06 由项目负责人授权记录（答复「建议 1 + 建议 2 + 建议 3」）。本记录只登记候选、证据与项目内处置，不代表对上游的采用或拒绝结论。

## 1. 触发问题

`AGENTS.md` 的「项目检查命令：」显示 **未配置**，而 `add.manifest.yaml` 已登记 7 条 `project.check_commands`（`repository_roots[0].shared_attributes.check_commands` 与 `project.check_commands` 两处一致）。发现于 2026-11-06 的文档一致性核查（`.spec/wip/verification/doc-consistency-20261105.md` §5）。

影响：入口声明与实际检查项不符，会误导 Agent 与新人；不改变构建/测试行为。定级 P3。

## 2. 证据

| 证据 | 位置 | 观察 |
|---|---|---|
| 渲染前像仍为「未配置」 | `.add/source/baseline/targets/AGENTS.md.baseline:39` | 采用时点的渲染结果被固化为基线 |
| 模板占位符 | `.add/source/content/entry/project-guide.md:33` | `{{project.check_commands}}` |
| 变量绑定 | `.add/tool/add.mjs:19973` | `project.check_commands` → `checkCommands`（list） |
| 片段合并语义 | `.add/tool/add.mjs:26306` `mergeFragment` | 只替换 `<!-- ADD:BEGIN -->` 与 `<!-- ADD:END -->` 之间的字节；marker 外内容原样保留 |
| 安装记录 | `.add/installed.json`（`"path":"AGENTS.md"`） | `ownership=marked-fragment`、`marker=ADD`、`sourceUnit=entry:project-guide` |
| 适配器要求 | `.add/source/adapters/files/3bb6e5b9….template:24` | 明确要求「保留各自 `add.manifest.yaml` 精确字段引用，不伪造默认事实或宣称未配置」 |

**未取到的证据**：没有执行真实重投影（`add.mjs` 无 `agents-md` 子命令；`init`/`upgrade` 会改动现役安装，本轮未授权也未执行）。因此「重投影是否会自动填出 7 条」仍是**推断**，不是实测。

## 3. 适用范围

- 项目：Han；Host：Codex 投影；ADD 版本 0.9.5（见 `.add/installed.json` 的 `packageVersion`）。
- 模块：项目入口文档投影（`entry:project-guide`），与业务代码无关。
- 不适用：其他 Host 投影、其他项目的 AGENTS 合同；本记录不构成对 ADD 母仓缺陷的结论。

## 4. 项目内处置（已执行）

| 动作 | 落点 | 提交 |
|---|---|---|
| 受管片段内手工对齐 7 条命令，并标注 `-race` 本机限制 | `AGENTS.md`（BEGIN..END 之间） | `c55cd13` |
| marker 外追加「本地补充（非 ADD 托管）」兜底：`-race` 限制 +「若显示未配置，以 manifest/README 为准」 | `AGENTS.md`（`<!-- ADD:END -->` 之后） | 本次 |
| README 增加同源与重投影后核对提示 | `README.md`「项目检查」段末 | 本次 |

选择该组合的理由：`mergeFragment` 只动 marker 之间，marker 外内容在重投影后仍保留；且 fragment 摘要不变（实测 BEGIN..END 区间 SHA256 前后一致），不制造额外漂移。

## 5. 候选改进（未采用）

- 建议 adoption 渲染 `entry:project-guide` 时读取 Manifest 的 `project.check_commands`（或按适配器规范输出对该字段的精确引用），使新项目与升级不再渲染为「未配置」。
- 是否上报 ADD 母仓属另行授权事项；本记录不外传、不自动分享、不触发母仓回写。

## 6. 限制

- 未实测重投影行为；`add.mjs validate` 不校验 fragment 摘要，因此「被覆盖」不会被工具报错，只能靠 README 提示与人工核对发现。
- 不建议修改 `.add/source/baseline/targets/AGENTS.md.baseline` 或 `.add/installed.json` 的摘要来「对齐」：前者会掩盖本地偏离，后者等于伪造工具状态。

## 7. 决定来源

- 2026-11-06 项目负责人授权执行建议 1、2、3（本记录即建议 3 的产出）。
- 状态保持 **proposed**：候选改进是否采用或上报尚未决定，不因记录存在而生效。
