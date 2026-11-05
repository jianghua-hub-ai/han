# S10.5 Root F003/F004 独立复审 R02

Reviewer：**Dewey / add-reviewer**，实际 Agent ID **`01a180fe-c3bb-7661-86fb-06af4f76ba1b`**。原非作者 Reviewer，由 Lead 继续原链委派；未再委派，未修被审产品。日期：2026-10-28。

**结论：PASS，限原 F003/F004 修复的两轮有界独立复审及必要 Root 回归。两项均 CLOSED，新增 finding 0。** 与原完整 Root 四文件 R01 的覆盖、精确修复 diff 合用，该 Root 范围原阻断项已清零；本次没有重做整个旧 Review。F001 按 r49 合作清理合同、F002 按实际网络截止修复继续 CLOSED。不是 Domain、PG/B、七文件 CodeGate、完整 Manifest 全绿、部署/E2E 或人的验收。

固定 `D:/AID/VSCode/Codex/Han` / main / 指定 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。新写仅本个人 Temp；当前复审结束后 STOP。

## 固定对象与修复链

本轮 objectType=code，输入 `.spec/wip/identity-auth/evidence-s10.5/root-f003-author-r01/index.json`，实际 SHA256 **`247E40BEF1BD1D42496635FE65AA15223B7A9961340627D9877FDFFA1CE83E95`**。独立核全部76个payload；本表最终原字节来自 `role/comment-removal-final/frozen/<rel>`，逐一与live一致，另存本目录 `fixed/`。

| 路径 | 最终 SHA256 |
|---|---|
| `cmd/han-server/main.go` | `833D3E1ECBC39C6CED1E91D59E63651A32A4AD3A1D04C1CC7662560244BDA6AD` |
| `cmd/han-server/main_test.go` | `4EF46AA9AF542C15FEF8D4688208ABDD75A6674C0A6F134D6645C86F4068D73E` |
| `cmd/han-server/maintenance_stdin_windows.go` | `620583ABAEB1551761359F93EAA7A4D0B79BE8A4E312B2FFD9C1D69A82663A43` |
| `cmd/han-server/maintenance_stdin_other.go` | `A711C46F5D0AD017CA54569BC8862F30EFCDF2989EA13B0C856A47F615D854D9` |

原完整 Root R01 报告 SHA **`17E4A4C7324BC0A20093E038F1C029AF116A0A44B904B186ACBCAB070A6BA99B`** 保持不变，原被审 main/test 为84A2/30A48，本目录保留其原字节前像和独立 diff。

实际版本链分清：84A2/30A48 → F004 的705177/0D3F2F → F003 的833D3E/89890A → 最终833D3E/4EF46。lostcancel历史索引及33payload已核；生产 main 最终版删除 prepare 前三行门禁后可逐字节还原705177。相对原84A2，**runMaintenance之外的所有生产字节完全不变**，两个adapter也完全不变，证据为 `production-delta-proof.json`。

作者最终 normal/race/vet 的测试源码实为 **`89890A8110503950F6AB3BB6042908360059E280965CE62F263E88C01340E59E`**，不是4EF46。本 Reviewer 独立逐字节比较：89890A→4EF46恰好删除旧10700–10702三行 `//` 注释，没有其他字节变化；main及adapter相同。`comment-bridge-proof.json` 保留删除位置和原文。作者行为证据可经这条精确桥接引用，但不冒称作者在最终4EF46重跑。本 Reviewer 下列新运行直接使用最终4EF46。

## Round 1：原 finding 的直接修复

### S105-CODE-ROOT-R49-F003 — CLOSED

最终 `main.go:4587–4589` 在 `dependencies.prepare(config)` 前调用既有 `maintenanceContextError(operation)`。既查 ctx.Err，也查实际单调deadline；原D来自配置读取前started，与较早父deadline合用，没有重开预算、调整支持面或授权来源。取消/过期在准备授权前返回failed/6，stdin所有权仍完成。

生产 constructor 的实际直接消费者仍为 `identity.NewPasswordResetMaintenanceOperator`，没有context参数，确会获取真实Windows owner SID并执行信任/allowlist授权；故Root门禁必须在该调用之前。其四字段Config、opaque ValidateRequeue及typed error等没有变动。本轮捕获的Domain2为 operator **A1C2039503A748F82529A01EA57E4EB14C478B9ACD34329770D7474E546D98B5** / test **5923422EFB2F81518ACA07FF178DC3FFFB63310CFE2A75F034EEDA530672C31D**，与原R01实际消费版本一致。没有据此代做Domain内部或PG独立PASS。

**纠正独立探针的前提：** 旧R01原样1ns叶没有观测实际D是否已到，不能单凭“解析理应耗时”作为到期证据。作者保留了该探针在已加门禁版本仍失效、Windows clock elapsed=0/guard=nil的原始记录，已实际读取；它不否定已经取消的旧反例或源码缺门禁事实。旧报告/日志不修改，本轮用明确前提重新验证，而不把作者复用Reviewer探针称为Reviewer独立执行。

本 Reviewer 新 `TestDeweyR02AuthorityExpiryPrecondition` 使用同一后缀做旧/新固定源码对照，全部在个人Temp overlay中：

| 场景及真实前提 | 旧84A2独立RED | 最终833D独立GREEN |
|---|---|---|
| parent ctx 已取消并确认Err非nil | prepare1，DSN0，exit3；拒绝阶段过晚 | prepare0，DSN0，failed6 |
| operationBudget=5ms；在最后配置getter等待实际单调时刻严格晚于首次读取+5ms | 实测5.2955ms，前提已成立，prepare1/exit3 | 实测5.3894ms，前提已成立，prepare0/exit6 |
| 父Deadline确在过去，实际Now已越过，而Err仍nil的受控late-timer seam | prepare1/exit3 | prepare0/exit6 |
| 及时对照，无过期/取消前提 | prepare1，保留forbidden3 | prepare1，保留forbidden3 |

Root started早于或等于首次配置读取，因此实际时刻晚于firstRead+5ms必然晚于原D。这不是用一个新timeout替换产品D；延时只存在个人测试getter，产品不变。各场景DSN0、stdout空、真实owned stdin已关闭。race下该实际耗尽前提同样成立（5.4471ms）。旧版三负例失败、及时对照通过；新版四叶均通过，证明修复门禁而不是拒绝所有准备。

### S105-CODE-ROOT-R49-F004 — CLOSED

最终 `main.go:4582–4586` 使用可见的 `operation,cancel := context.WithDeadline(...)` 与直接 `defer cancel()`，其后注册final检查defer；LIFO实际顺序为：

1. DB/SDK/pool等下层调用先返回并完成各自清理；
2. `stdin.Close()` 与原operation ctx/单调时刻的 `maintenanceFinalError`；
3. 自用cancel；
4. 外层panic恢复/仅在尚未交接operation时负责的输入fallback。

`operationOwnsInput` 只管理这一关闭/检查所有权。operation尚未创建的配置/时间加法失败仍由外层关闭stdin并检查原父ctx；已经交接后不在self-cancel之后再把本应成功的结果改6。没有nolint、屏蔽lostcancel或放松检查。

Reviewer 对**最终无注释版**执行 `go vet -mod=readonly -overlay <Temp>/vet-overlay.json ./cmd/han-server`，exit0，stdout/stderr空；overlay没有Reviewer测试后缀。Main同最终source另有 `go vet ./...` exit0原始收据。旧vet失败事实仍在R01，不把原possible提示说成已发生运行时泄漏。

独立normal/race执行新的cancel-owner回归：Windows真实owned pipe及时EOF后，合成依赖的applied/already_applied仍exit0，unknown仍exit5；operation在执行、资源释放/最终检查期间未提前self-cancel，返回后已取消，stdin关闭、pool归还。另核早退、外部取消、sticky end、unknown/其他primary和private cause相关已有回归。这里的pool/业务是U sentinel，**不证明PG提交事实**。

## Round 2：受影响消费者及 F001/F002 回归

两份精确修复diff已全读：runMaintenance所有权组织与三行门禁、两项新子测试注册和相应测试函数。没有重新扩审无关旧main/identity。生产其余全部字节相同的独立比较，使原R01覆盖可以明确继承；原同版Role/完整add-code-review与适用Rule/profile复用且核摘要一致，未重新Route/委派。

实际核原输入关闭→授权准备→Validate→DSN/constructor/Ping/Requeue→SDK/pool清理→最终错误→self-cancel→signal释放→输出链。新gate只截断过期准备，不改变普通server/Recovery模式；typed primary/unknown、原D/K/C合作join合同与输出无新定义。六份上游合同仍与独立Design R03的manifest **478E11DB183DB40B53BF3980BF897753FD8E50331992A4B468455A8A0A4E5BF5** 逐项同hash。

| 原 finding | 本次回归状态与独立实际证据 |
|---|---|
| `S105-CODE-CLEANUP-R01-F001` | **保持CLOSED，限r49合作清理合同。** SDK/owner/统计/单调end生产代码逐字未变；本轮实际pgx/内存SCRAM peer在C后442.2µs仍Constructing1/stage1/socket0，最终C后475.023ms实际join，最终constructing/stage/socket/acquired0、peer/pool结束、sticky overrun、failed6/stdout空。旧硬完成保证没有恢复，也未被称为已修复。 |
| `S105-CODE-CLEANUP-R01-F002` | **保持CLOSED。** deadline cap、三个setter、callback join与共享SDK close链未改；本轮真实pgx/TLS1.2 net.Pipe背压1000ms、健康0ms整数显示，deadline未放宽、raw不先SDK、SDK/raw各1、stage/socket0、SDK done/entry/peer/pool均实际结束。另执行callback join及Close竞争回归。 |

所有这些都是Root U/真实SDK内存协议peer层；没有启动PG或网络listener。F001的不可取消CPU可能越期乃至不返回仍按已采用r49明确保留，未换回r48严格进程截止。

## 本 Reviewer 实际命令及计数

运行器 `run-review.py` 的每项receipt记录完整实际argv；下列overlay只在个人Temp，产品只读，运行前后live Root4均与最终freeze一致。RED使用旧84A2/30A48副本；GREEN/race使用新833D/4EF46，Domain2使用实际捕获的同版依赖；环境离线隔离，无部署ENV/DSN或秘密。

```text
go test -mod=readonly -overlay <Temp>/red-overlay.json ./cmd/han-server -run '^TestDeweyR02AuthorityExpiryPrecondition$' -count=1 -json -timeout=180s
go test -mod=readonly -overlay <Temp>/green-overlay.json ./cmd/han-server -run <green.run.json中的精确filter> -count=1 -json -timeout=180s
go vet -mod=readonly -overlay <Temp>/vet-overlay.json ./cmd/han-server
go test -race -mod=readonly -overlay <Temp>/green-overlay.json ./cmd/han-server -run <race.run.json中的精确filter> -count=1 -json -timeout=180s
```

| 独立运行 | 实际结果 |
|---|---|
| 同一前提探针 / 旧84A2 | 5节点/4叶：三负例RED、及时对照PASS，命令exit1为预期失败证据 |
| 新版定向normal | 原Root选中43节点/35叶 + 独立探针5节点/4叶，共39叶全部PASS，skip0，exit0 |
| 新版定向race | 原Root选中10节点/7叶 + 独立探针5节点/4叶，共11叶全部PASS，skip0，无race诊断，exit0 |
| 最终Root包vet | exit0，stdout/stderr空 |

不是重复跑整个Root153叶或Main广测试。完整原始JSONL、stderr、before/after源码hash、overlay、独立探针及逐叶统计均保留；filter精确文本见两个运行收据，未只以退出码代替断言。

## 作者 / Main 证据与未验范围

作者行为验证版89890A的最终 normal/race 原始日志已分别独立重数：169 unique nodes /153 leaf，failure0/skip0，无race诊断。该证据经上面的严格三注释删除桥接关联到最终4EF46；本人新定向运行已直接覆盖最终版。作者Linux仅test-compile，不说Linux运行通过。

Main最终Manifest索引 **06D7F466F0FCB54B4FA0C4BF1DD5325E97AB12E91963A062F4CF0882E64416F6** 及50payload已核摘要，实际mod verify/vet/build/contract/fullrace/fullunit六条exit0，产品七源码gofmt空输出。它们是Main运行，本Reviewer没有重复执行或冒领。

**精确 `gofmt -l .` 仍实际exit2**：immutable `.spec` 证据中六个未格式化overlay/probe及两个空的absent-file前像产生格式/parse输出，原始stdout/stderr保留。已读取Main `combined-root-f003-r02/lead-gate.json` 的当前处理来源；本Review不格式化或删除历史证据、不把全仓格式记PASS，也不将本次F003/F004关闭冒成“所有Manifest检查全绿”。这不是重新归因于已修复lostcancel。

未运行实际PG/SQL/DDL、生产maintenance CLI、真实listener、部署launcher/ACL/签发/第二真实SID/E2E。没有对在途carrier或Domain内部作独立整体结论。人的接纳、发布和后续执行由原责任方按真实证据决定。本次无待作者修复的新finding。

**原17E4报告与历史失败链保持不变；本R02只追加精确修复闭环。当前STOP，不自行推进后续phase。**
