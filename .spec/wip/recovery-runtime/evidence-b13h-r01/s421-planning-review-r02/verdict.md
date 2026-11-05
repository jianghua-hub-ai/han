# S4.21 规划评审判定（R02）

- 对象：`identity-auth-S4.21` 卡（`tasks.md:2961` 锚点、`:2963` 标题、`:2965-3024` 全部字段）+ Plan B13H 登记（`:1309`、`:1333`、`:1337`、`:1445-1450`、`:1611`、`:1624`）
- 修订：tasks.md `49A4F7D07D213465D03E827E754D932EBC96F96B53F7E456DDBB52CD5684F6A0` / 800323 B / 3024 行 / CRLF=3024 / BARE_LF=0 / 无 BOM；前 2959 行 782348 B / `DF91604B67CF690FE94F5B07ECC048B278FDC09BA275631265B12E03E1EC01FD`；Plan `13B21F56728996DB9E3D29842AE016D9F9BC1DDA5598EF894DA2BBDAB9C8FFC1` / 495502 B / 1628 行 / CRLF=1628 / BARE_LF=0 / 无 BOM
- 结论：**CHANGES REQUIRED**
- 阻断项：`S421-R02-F001`（MEDIUM）
- 非阻断项：`S421-R02-F002`/`F003`/`F004`（LOW）
- 评审日期：2026-11-01（Asia/Shanghai）；HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；分支 `main`

## 阻断理由

`S421-R02-F001`：卡面 leaf `ConfigCommandMatrix`（`tasks.md:3013`）与关闭条件（`:3018` “配置/命令旁路运行矩阵逐项正控/负控闭合”）没有任何可由本卡验收命令（`:2992-3003`）产生的当前版本证据。现有 `TestRecoveryConfig`（`recovery_config_test.go:34`，仅 normal/verify）与 `TestRecoveryCommand`（`recovery_command_test.go:8`，含 enter 安装形态/resume）均未列入；`TestRecoveryRuntimeComposition`（`recovery_runtime_test.go:229`）与 `TestRecoveryMainBusinessDispatcher`（`recovery_main_dispatcher_integration_test.go:26`）都不解析 config/argv。进一步，即使补入前两者，`loadRecoveryRootConfig` 的 enter/resume 分支（`recovery_config.go:46-50`）仍无单元覆盖——当前只有 S4.20 生产链测试的 `s420ResumeStartupPlan`（`recovery_production_chain_integration_test.go:2870-2893`，需真实 PG）间接覆盖，且同样不在命令清单。因此关闭时可在未执行矩阵的情况下声称该项闭合。

该缺口是证据覆盖问题，不改变 production 仍 closed、未声称 Enter/Install 可达的结论。

## 通过项

- R01 `S421-PLAN-F001`（HIGH）实质关闭：卡面已收窄为“显式 closed 端口清单 + 可注入装配 seam + secret candidate seam”，并多处明确 production 默认仍 closed、模拟 seam 不等于 production 可达（`:2963`/`:2965`/`:2967`/`:2971`/`:3009`/`:3012`/`:3018`/`:3020`/`:3024`）。
- precheck 位置正确：`recovery_runtime.go:482-483` 位于 `runs`（`:488`）、`closures`（`:495`）、anchor registry（`:502-506`）、journal/keyring（`:523-541`）之前；缺端口返回 `recovery.ErrAdmissionClosed`（`internal/recovery/admission.go:31`）。
- seam 精度成立：`:556-557` 复用既有 `inputs.installation`，未声称新建；`:582-583` 为既有闭包占位，新增未导出 `secretCandidate` seam 的必要性成立；production `main.go:634` 不注入任何 seam，未引入第二 runtime/Store/pool/authority/fallback。
- 11 项必填输入与 `recoveryRootInputs`（`recovery_runtime.go:431-449`）可选 seam 一一对应，无多列/漏列（F004 仅指出并集口径与测试 (a) 断言强度）。
- 文件所有权 3 文件（2 改 + 1 新增）无重叠，`main.go`/`recovery_runtime.go` 串行写窗要求同时记录两者 SHA/字节/mtime（`:3022`）；F002 仅指出“第五文件”措辞与 3 文件写集的 off-by-one。
- requires 闭合无环：S4.13/S4.16/S4.18/S4.19/S4.20 均 `[x]`；S8.2 仅消费者，未进 hard requires；Plan `n34` 五条入边 `n20`/`n23`/`n31`/`n32`/`n33` 与卡面一致。
- Plan 计数未被本轮修订改变：`:1309` 18 唯一文件/24 声明槽；`:1337` 18 卡/75 边、77 节点/329 边；`:1624` 已改为“18个新Task（截至 S4.21 登记）”；`:1611` 已追加 R02 记录。
- 无 Case18/S8.2 PASS 或 blocked-upstream 解除表述（`:3012`/`:3016`/`:3019`/`:3020`）；与 `article-management-S8.2` `[-]`、`article-management-S1.5` `[-]` 无冲突；`[ ]` 无完成日期、无 PASS 预填。
- 卡内 file:line 指针逐条有效（含 `test-plan.md:359/360`、`design.md:1463/1520/1524`、`identity-auth/test-plan.md:837`、`recovery_runtime.go:482-483/556-557/582-583`）。

## 复审条件

1. F001：把 `TestRecoveryConfig`/`TestRecoveryCommand` 普通与 `-race` 加入验收命令，或在可写的新测试文件内补 config/command 矩阵子测试，并在卡面把 `ConfigCommandMatrix` 的五个正控/三个负控逐项映射到测试/子测试名；全部保持 NOT RUN。
2. F002-F004 可直接修正，不阻断。
3. 修正后由非作者复审原 finding；本判定不代替人的接收、实施、测试或 Git 授权。

- 评审身份：add-reviewer；本会话 actualID 不可得，不编造；作者 add-task-planner 按任务给定 `01a19293-a26b-7f23-8a0f-2aab006a63f2`。
- 本判定为 AI 代理核证，非真人签字。