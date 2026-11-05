# identity-auth 设计 r63 → r64 修订报告（B13J／S4.24a，第二轮最小修复）

## 一、修订范围与不变量

| 项目 | 值 |
|---|---|
| 修订对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| 输入基线 | `identity-auth-design-ir01-r63`；976149 B／5416 行／`3B8BEA8C35A12F75B36091969DB15B9590FBDD16B0429D59328ABB9FA3BC027C` |
| 输出 | `identity-auth-design-ir01-r64`；1005979 B／5449 行（`ReadAllLines` 口径）／`FB84EC1E0B078AC99B09A48988F00257B77CA93102298A7DAD5004E0B8871E94` |
| 版本号行 | `:5` `revision:` 与 `:38` 设计版本号均为 `identity-auth-design-ir01-r64`；`last_updated: 2026-11-02` 未变 |
| r64 区块 | `:4025`–`:4055`（`:4025` 标题，`:4055` 结束声明）；`:4023` 为 r63 结束声明，`:4057` 起为 r62 历史段 |
| 报告自身 | 本文件；只新建本文件，未创建其他文件 |
| 工作区 | `main`／HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；未切分支、未 checkout／stash／reset／commit、未移动或删除任何文件 |

不变量（r64.0 逐条判定，`:4027`）：首期不提供生产 restore 工具与生产批准主体的结论不变；生产 restore 继续 closed，`recoveryClosedRestoreDriver` 仍是默认；`enter`／`resume` 不因本修订解锁；任何 NOT RUN 仍是 NOT RUN；不新增第二 authority／fallback／隐式默认；r60 的 11 端口清单与唯一 production root 不被改写。

r63／r62 正文除下列**授权改写点**外只加句尾取代标记（27 处 `（r64.*` 标记 + `:1607` 代码块内 `// r64.8 取代` 注释）：

- `:3995`（r63.9.1）按 D2 改写为「`restoreLease` 是 `exclusive` 端口一次 `Acquire` 的产出值，不是新端口／新依赖项」，删除与新写法互斥的模糊表述。
- `:4021`（r63.9.5）按用户修正 3／5 追加未闭合项（8）–（11）：SEC-F002、SEC-F004、`NoNewPrivs` 放宽、audit sink 残余。
- `:3976`（r63.6 ENV 白名单）按 D7 移除 `LD_LIBRARY_PATH` 允许键，改为「任何 `LD_*` 一律 closed」。
- `:3975`（r63.6）按 D7 删除「如需抑制 cache，只能让 `LD_LIBRARY_PATH` …」整句，余句保留并加标记。
- `:4122`（r62.4 错误脱敏）按 D14／N010 删除「长度」字样，改写为「只允许分类与进程内随机关联 ID，不得携带 DSN 或任何可离线比对派生值（摘要、指纹、前缀、主机名或用户名）」，并加标记。

另按「可机核」要求，r64 区块内对 r63／r62 正文的行号引用已换算为 r64 写入后的现行行号（插入 r64 使既有内容整体下移 33 行）：向量 `:4072`／`:4078`、r62.6.2 `:4142`、r61 Prepare `:4346`、信任根上限 `:4061`、`tool.receipt` `:4090`、错误脱敏 `:4122`、loader 八步 `:4184`。只改行号指针，不改任何语义；r63／r62 正文里原有的历史编号不随本节改写。

## 二、逐项落点表（Finding → r64 子节 → 行号 → 可机核判据）

| 编号 | 级别 | Finding 摘要 | r64 落点 | 行号 | 可机核判据（在 `design.md` 上检索／复算） |
|---|---|---|---|---|---|
| ARCH-F006 | BLOCKER | lease 构造签名与 `authority`／ownership 接线不成立，`Bind` 收 provider，Runtime 内部四处 `Acquire`／两处 `Close` 未处置，写集缺 `internal/recovery/runtime.go` | r64.1；r64.12；r63.9.4 表 | `:4029`；`:4051`；`:4017` | 文本同时出现 `NewRuntime(ctx, deps, restoreLease)`、`restoreLease recovery.ExclusiveLease`、`:729`、`:768`、`:476`／`:486`、`:870`／`:988`、`:1022`–`:1023`、`:1424`、`ErrAdmissionClosed`、「不得 `Close` 注入的 `restoreLease`」；`:4017` 行含 `internal/recovery/runtime.go` 与「restore 流不再自行 `Acquire`」 |
| ARCH-F004 | HIGH | `:3995` 与 `:3997` 对 lease 是否进入 `RuntimeDependencies` 互斥；r60 `:3603` 无优先关系 | r64.1；改写后的 r63.9.1 | `:4029`；`:3995` | `:3995` 与 `:4029` 均含「端口一次 `Acquire` 的产出值」；`:3995` 不再把 lease 列入「不加入 `RuntimeDependencies`」；`:4029` 同时保留「`exclusive` provider 本身仍在 `RuntimeDependencies`」与「r60 `:3603` 针对端口与可独立配置／替换的依赖」 |
| ARCH-F018 | HIGH | 验证方签名载荷 keyset／编码、公钥来源、`verified_at` 取值点、`anchor_ref` 覆盖未定义 | r64.2 | `:4031` | 8 键串 `anchor_ref, prev_record_sha256, receipt_sha256, result, sequence, verified_at, verifier_principal_id, verifier_key_id` 逐字出现；`keys[verifier_key_id].public_key`、`principal_id` 相等、`verified_at` 一次性取样、四处 `anchor_ref` 逐字节相等、验签失败不推进 sequence |
| NEW-R03-03 | MEDIUM | `sequence_floor` 同名异义，见证 head 形状缺失 | r64.3 | `:4033` | 信任根字段为 `baseline_sequence`；head exact keyset `anchor_ref, sequence, issued_at, head_sig_alg, head_sig`；`head_sig_alg` 只允许 `ed25519`；两条向量重算 1115 B／`de9146…`、2357 B／`8c0c…` |
| SEC-F002 | BLOCKER | 角色命名空间断裂、一人两名可构造、`reviewer_key_id` 未绑定 reviewer principal | r64.4 | `:4035` | `principals` keyset 含 `person_ref`；`(role_name, org_id)` 唯一；`reviewer`／`approver`／`executor`／`producer`／`verifier`／`witness` 角色来自 `appointments.role_name`；`reviewer_key_id`／`producer_key_id`／`verifier_key_id`／`witness_key_id` 的 `keys.principal_id` 相等；残余风险句含「同一自然人使用两个不同 `person_ref` 仍不可检测」 |
| SEC-F003 | BLOCKER | 唯一调用图遗漏 Prepare 字节重读与 `prepared()`（回归） | r64.5 | `:4037` | 出现 `P4 source.OpenApproved`、`P5 RestoreSourceVerifier`（早于 `Journal.Begin`）、`P6` 重读批准文件精确字节、`V1 prepared(ctx)`；并写明 `:4142` 与 `:4346` 继续有效、r63.3 属回归 |
| SEC-F005 | BLOCKER | `LD_LIBRARY_PATH` 与 ADR-0031 冲突；cache／conf／`dt_needed` 父目录未覆盖；maps 时点不一致 | r64.6；标记 | `:4039`；`:3975`／`:3976`／`:3980`／`:4184`／`:4186`／`:4188`／`:4190`／`:4408` | 允许键只剩 `LANG`／`LC_ALL`／`TZ`／`PATH`；`LD_LIBRARY_PATH` 只出现在禁止／删除语境；`static` 为唯一完整可绑定形态；父目录算法覆盖 `pt_interp_path`／`dt_needed.path`／`ld_so_cache_path`／`ld_so_conf_paths.path`／`ld.so.preload`／工具／artifact；maps「exec 成功后立即 + 读取回执前」 |
| N001 | BLOCKER | `receipt_anchors` 无 witness 端点／公钥／协议；签名不覆盖 anchor 与验证方身份；policy 与 anchor 的 `audit_sink` 无相等要求 | r64.3；r64.2 | `:4033`；`:4031` | `witness_principal_id, witness_key_id, witness_public_key, witness_endpoint, witness_protocol` 全在 keyset；`han-restore-witness/v1`；`https://` 绝对 URL、2048 B 上限、无 userinfo／fragment；`witness_public_key` 等于 `keys[witness_key_id].public_key`；`audit_sink` 逐字节相等；缺失／不可达／不可验签 → `ErrAdmissionUnsupported` |
| N003 | BLOCKER（评审建议升级） | 三套互斥回执字段表并存；`attestation` 与 `receipt` 重复字段无相等要求 | r64.8；标记 | `:4043`；`:1607`／`:1667`／`:1671`／`:1718`／`:3884`／`:4090` | 「唯一 wire 是 r63.8 的 12 字段」；`:3884`／`:1667` 句尾有 r64.8 标记；`:1607` 代码块内 `// r64.8 取代`；旧字段逐项处置（`receipt_digest`、`restored_schema_digest`、`restored_at`）；attestation 七字段逐一相等 |
| SEC-F004 | HIGH | 九处清零未覆盖 `privateRestoreInput` 返回值与 `execute` 持有副本；DSN 以 `string` 承接 | r64.9；标记 | `:4045`；`:3971`／`:3973`／`:4118`／`:4122` | 九项按代码级缓冲名列出（`input`、`execute` 引用、`runChild` 本地副本、`raw`、`revalidateApproval` 读缓冲、百分号解码缓冲、`snapshot.dsnSHA256`、子进程解析器、driver 关闭路径）；`privateRestoreInput` 参数与 `d.dsn` 为 `[]byte`；`appendJSONString` 直写 |
| SEC-F006 | HIGH | `CapBnd`／`NoNewPrivs`／seccomp／netns 断言仍在正文；NNP 放宽未登记 | r64.7；r64.0；标记 | `:4041`；`:4027`；`:4021`；`:4053`；`:4186`／`:4188` | `CapBnd`／`NoNewPrivs` 只作观测值；`CapInh=CapPrm=CapEff=CapAmb=0`、uid／gid 四元组、`Groups` 为空仍是拒绝条件；r64.0 单列「放宽登记（唯一一项）」；`:4021`(10) 与 `:4053`(3) 同步 |
| N006 | HIGH | `none` 被写成「不联网」但无执行机制 | r64.7；标记 | `:4041`；`:4190`；`:3980` | 「`none` 与 `self_attested` 都只是工具自证 + 事后审计」；`netns_inode`／`netns_path` 必须为空串；需要真实隔离 → `ErrAdmissionUnsupported` |
| N008 | HIGH | `audit_sink` 父目录／ACL／inode 边界未定义 | r64.10 | `:4047` | 父目录 root-owned `0700`、逐级 `lstat`、禁 `system.posix_acl_access`／`_default`；双 fd `O_NOFOLLOW|O_CLOEXEC`；`(st_dev, st_ino)` 每次追加前复核；`flock(LOCK_EX)` 覆盖读尾—算 `prev_record_sha256`—追加—`fsync` 文件—`fsync` 父目录 |
| SEC-F008 | LOW | 排序数组清单漏 `organizations`／`appointments`；`tool` 向量 `netns` 无 inline 说明 | r64.11 | `:4049`；`:4081` | `organizations`、`appointments` 与 `keys`／`principals`／`reviews`／`receipt_anchors` 并列；`:4081` 句含「`network.mode=netns` 只是 canonical fixture，不构成运行时可接受的批准记录，不得复制为示例；向量字节不变」 |
| N009 | LOW | r63 编号 `:4057`／`:4089`／`:4151`（现行 `:4090`／`:4122`／`:4184`）陈旧句；maps／cache 两套口径 | r64.11；r64.6；标记 | `:4049`；`:4039`；`:4090`／`:4122`／`:4184` | 三处句尾均有 r64 标记；现行 `:4122` 的「长度」已删；现行 `:4184`(8) 由 r64.6 取代；现行 `:4184`(6) 与 `:3975` 共用 `ld_so_cache_path` + 摘要 + 父目录／挂载规则 |
| N010 | LOW | 错误文本携带 DSN 长度构成派生值 oracle | r64.11；r64.9 | `:4049`；`:4045`；`:4122` | `:4122` 与 r64.9 已无「长度」字样，错误文本只允许分类 + 进程内随机关联 ID、不得携带 DSN 派生值；全文其余「长度」仅见于字段／文件长度上限约束（`:4033` 的 2048 字节、`:4061` 的 `1..65536`、`:4387` 的 `1..4096` 等），不属错误文本 |

### 2.1 用户三项修正与三项补充的落点

| 修正／补充 | 落点 | 判据 |
|---|---|---|
| D14 修正：超限分类保持 `ErrAdmissionIncomplete` | `:4049` | 句含「失败分类保持 `ErrAdmissionIncomplete`（与 `:4061` 现值一致，fail-closed 方向不变）」；判定位置为信任根读取边界、早于任何签名校验；approval 仍 `1..65536` |
| D14 补充：`tool` 向量 `netns` 说明 | `:4081` | 紧邻向量摘要句；不改向量字节（重算一致） |
| D14／N009 补充：r63.9.5 未闭合项 | `:4021` | 第（8）项 SEC-F002、第（9）项 SEC-F004、第（10）项 NNP、第（11）项 audit sink |
| D7 补充：maps 时点与 cache 字段统一 | `:4039` | 「以 `:3978` 为准」；「`:4184` 第（8）项由本条取代」；「`:4184` 第（6）项与 `:3975` 必须使用同一套字段」 |
| D11 补充：NNP 放宽登记 | `:4027` | 单列「放宽登记（唯一一项）」，含放宽内容／理由（Go `os/exec` 无 pre-exec 钩子）／影响（不声称隔离）／抵消措施；同步 `:4021`(10)、`:4053`(3) |
| D13 补充：audit sink 判据位置 | `:4047` | 边界写在「现行判据（不是残余风险段）」；残余风险段只保留「抗 root 整体替换止于外部见证」 |
## 三、D1–D14 处置摘要与仍开放项

| 项 | 处置 | 落点 | 仍开放／边界 |
|---|---|---|---|
| D1 ARCH-F006 BLOCKER | 固定唯一取得点：装配体对已登记 `exclusive` 端口 `Acquire(ctx)` 恰好一次；`NewRuntime` 新增非可选参数 `restoreLease recovery.ExclusiveLease`；`driver.Bind` 收同一 lease，不再收 provider；Runtime 只读保存；非 restore 流（`:476`／`:486`、`:870`／`:988`、`:1554`／`:1562`）仍走 `deps.Exclusive`；`closeObservationLease`（`:1022`–`:1023`）与 `:1424` 不得 `Close` 注入 lease；`Close` 唯一责任方为装配体 `defer`，失败进入错误分类；`inputs.exclusive == nil` → `recoveryClosedPort` → 原样 `ErrAdmissionClosed`；写集增补 `internal/recovery/runtime.go` | `:4029`、`:4051`、`:4017` | 五类 action 的完整 `Acquire`／`Check`／`Close` 表未逐 action 列出（只按代码位置点名）；实现与测试世界改造属 Task-Planner／实施阶段，本轮不改代码与任务卡 |
| D2 ARCH-F004 HIGH | 明确 lease 是端口产出值而非端口：不进 r60 11 端口表、不注入 `recoveryRootInputs`、不进 `RuntimeDependencies`；provider 仍在 `RuntimeDependencies`；r60 `:3603` 针对端口与可独立配置／替换依赖，不针对产出值；改写 `:3995` 消除与 `:3997` 互斥 | `:4029`、`:3995` | r60 正文与 ADR-0032 未修改（属 ADR 维护方）；若实施选择改 `RuntimeDependencies` 契约则须先走 ADR |
| D3 ARCH-F018 HIGH | 8 键全 string 载荷（删 `signature` 后 JCS）；`audit_record_ref`／`verifier_signature` 不入载荷；公钥唯一来源 `keys[verifier_key_id].public_key` 且 principal 相等；`verified_at` 检查点一次性取样；`anchor_ref` 四处逐字节相等；验签失败不推进 sequence | `:4031` | D6 插入 `V1 prepared(ctx)` 后原 V9 顺延为 V10，故 `verified_at` 绑定 r64.5 的 V10（审计追加／签名／推进）检查点；此为编号顺延，不是语义改变。真实 Ed25519 验签与见证推进仍 NOT RUN |
| D4 NEW-R03-03 MEDIUM | 基线字段改名 `baseline_sequence`（十进制字符串、初始基线、运行时不回写）；旧键出现即 `ErrAdmissionBinding`；新增 head exact keyset 与签名／验签规则；两条 golden vector 复算不变 | `:4033` | 无（改名后的向量复核已完成，见第四节第 7 项） |
| D5 SEC-F002 BLOCKER | `principals` 增 `person_ref`；`(role_name, org_id)` 唯一；同一 `reviews` 记录不得同人两席；角色只来自 `appointments.role_name`；四类 `key_id` → `principal_id` 绑定；`witness_principal_id` 新增；残余风险显式登记 | `:4035` | 「同一自然人两个 `person_ref` 不可检测」仍开放，仅人工复核与组织任命记录压制，不声称已解决 |
| D6 SEC-F003 BLOCKER | 完整替换图：P1–P6、R1–R8、V1–V11；`P4 OpenApproved`、`P5 RestoreSourceVerifier`（早于 `Journal.Begin`）、`P6` 重读批准文件字节；`V1 prepared(ctx)`、`V2 revalidateApproval`；声明 `:4142`／`:4346` 继续有效、r63.3 属回归 | `:4037` | 无（编号唯一，未重排既有 R 语义；R1 `prepared(ctx)` 保留） |
| D7 SEC-F005 BLOCKER | 删除 `LD_LIBRARY_PATH` 抑制 cache 路径；允许键只剩 `LANG`／`LC_ALL`／`TZ`／`PATH`；任何 `LD_*`／`*_PRELOAD`／`*_OPTIONS`／`GLIBC_*`／`MALLOC_*` 即 `ErrAdmissionClosed`（ADR-0031 `:1117`／`:1144`／`:1164`）；loader 静态／动态分档；父目录与挂载覆盖扩展；maps 时点以 `:3978` 为准；cache 字段与 `:3975` 统一 | `:4039`、`:3975`、`:3976`、`:4184`、`:4186`、`:4188`、`:4190`、`:4408` | dynamic 形态的「解释器实际读取路径」证明仍无实测；无法证明即 `ErrAdmissionUnsupported`（closed） |
| D8 N001 BLOCKER | `receipt_anchors` 增 witness 五字段；`witness_protocol` 固定；endpoint 约束；`witness_public_key` 与 `keys` 相等；验证方签名覆盖 `anchor_ref` 与验证方身份；policy 与 anchor 的 `audit_sink` 相等；见证缺失即 closed | `:4033`、`:4031` | 真实 witness 端点、TLS、鉴权与原子推进未实现、未实测 |
| D9 N003 HIGH（评审建议 BLOCKER） | 唯一 wire 为 r63.8 的 12 字段；`:3884`／`:1602`／`:1667` 由 r64 取代并加标记；旧字段逐项处置；attestation 七字段逐一相等 | `:4043`、`:1607`、`:1667`、`:1671`、`:1718`、`:3884`、`:4090` | `internal/recovery.RestoreReceipt` 旧 Go 类型的删除／改名属实施阶段，本轮只声明其不是 stdout 解析契约 |
| D10 SEC-F004 HIGH | 九处清零点改为代码级缓冲名，含 `privateRestoreInput` 返回值与 `execute` 持有引用；`privateRestoreInput` 参数／内部字段／`d.dsn` 改 `[]byte`；`appendJSONString` 直写；错误只留分类与随机关联 ID | `:4045`、`:3971`、`:3973`、`:4118`、`:4122` | `execute` 持有副本、GC、管道内核缓冲、子进程地址空间仍可能保留 DSN；本进程清零不等于全链路不泄露 |
| D11 SEC-F006 HIGH | `CapInh/CapPrm/CapEff/CapAmb=0`、uid/gid 四元组、`Groups` 为空仍为拒绝条件；`CapBnd` 降为观测值；`NoNewPrivs` 降为观测值并单列放宽登记；seccomp／netns／setns unsupported；`CapBnd` 收敛不声称 | `:4041`、`:4027`、`:4021`、`:4053`、`:4186`、`:4188` | NNP 放宽后的残余提权面仍在；bounding set 保留但不在 exec 时约束子进程；工具若 exec setuid／file-capability 程序的风险未由本设计消除 |
| D12 N006 HIGH | `none` 与 `self_attested` 均只是工具自证 + 事后审计；不提供网络隔离；`netns_inode`／`netns_path` 空串；真实隔离 → `ErrAdmissionUnsupported`；无法绑定目标配置 → `ErrAdmissionBinding` | `:4041`、`:3980`、`:4190` | 网络隔离能力不存在，ADR-0031 解除条件中的网络要求无法由本设计满足 |
| D13 N008 HIGH | 父目录 root-owned `0700`、逐级 `lstat`、禁 ACL；双 fd `O_NOFOLLOW|O_CLOEXEC`；`(st_dev, st_ino)` 追加前复核；`flock` 区间含父目录 `fsync`；失败即 `restore_unknown`／closed。写进现行判据段，残余风险只留抗 root 替换 | `:4047` | 持有 root 或父目录写权限者仍可整体替换 audit sink；抗回滚只到外部见证 head |
| D14 SEC-F008／N009／N010 LOW | 排序数组清单补 `organizations`／`appointments`；三处陈旧句加标记并删「长度」；信任根上限 `1..1048576`、超限 `ErrAdmissionIncomplete`（读取边界、早于签名校验）；approval 仍 `1..65536`；`tool` 向量 `netns` 说明；错误文本只留分类与关联 ID | `:4049`、`:3930`、`:4061`、`:4063`、`:4067`、`:4081`、`:4090`、`:4122`、`:4128` | 无（信任根上限变更仅文本；真实多组织信任根未生成） |
## 四、自检证据（§5 十项的真实输出）

口径：`[IO.File]::ReadAllLines(...,UTF8)` 取行、`[Text.Encoding]::UTF8.GetBytes` 取字节、`SHA256` 直接计算；检索为 `Contains`（区分大小写）。行号均为 r64 输出文件行号。

1. **`LD_LIBRARY_PATH` 全文检索**：4 处 —
   - `:3975` r63.6 loader 原句 + 标记「`LD_LIBRARY_PATH` 不再允许」（禁止语义）；
   - `:3976` ENV 白名单「拒绝任何 `LD_*`（含 `LD_LIBRARY_PATH`）…任何 `LD_*` 一律 closed」（禁止语义）；
   - `:4039` r64.6「整句及其替代路径**删除**」「任何 `LD_*`（含 `LD_LIBRARY_PATH`）…出现即 `ErrAdmissionClosed`」（删除句 + 禁止语义）；
   - `:4408` r61 ENV 原句「禁止 `PATH`、`LD_PRELOAD`、`LD_LIBRARY_PATH`…」+ 标记（禁止 + 历史）。
   结论：无任何允许语义。

2. **`sequence_floor` 全文检索**：11 处 — `:3934`（r63.2 keyset 历史 + 标记）、`:3942`（r63.3 V8 外部见证 head 语境 + 标记）、`:3990`（r63.8 外部见证 head 语境）、`:4012`（r63.9.4 见证客户端写集行，见证语境）、`:4027`（r64.0 改名陈述）、`:4033`（r64.3 改名 + head 定义）、`:4049`（r64.11 历史说明）、`:4063`（r62.1 keyset 历史 + 标记）、`:4067`（r62.1 canonical 清单历史 + 标记）、`:4128`（r62.5 回执字段历史 + 标记）、`:4315`（r61 示例历史）。
   结论：信任根基线一律 `baseline_sequence`（`:4033`、`:4049`）；其余仅历史句、标记或见证 head 语境。

3. **`RestoreReceipt`／`receipt_digest` 检索**：
   - `receipt_digest` 4 处：`:1671`（Store `restore_binding` 列 + 标记「仅作 Store 绑定列」）、`:1718`（`recovery_restore_bindings` 列 + 标记「只保存 `receipt_sha256`」）、`:3884`（r61 12 字段表 + 标记）、`:4043`（r64.8 处置说明）。
   - `RestoreReceipt` 现行 wire 只有 r63.8 的 12 字段（`:3987`／`:3988` 定义，`:4043` 唯一化）；`:1602` 的 Go 类型在 `:1607` 结束行加 `// r64.8 取代：该 Go 类型不是 stdout 解析契约`；`:1667` 字段说明句尾加 r64.8 标记。
   结论：只剩一套现行 wire + 取代标记。

4. **`CapBnd`／`NoNewPrivs` 检索**：各 8 处。`CapBnd` — `:3927`（r63 范围）、`:3983`（r63.7 unsupported）、`:4009`（r63.9.4 写集「不含 CapBnd」）、`:4021`（r63.9.5 第（10）项）、`:4027`（r64.0 放宽登记）、`:4041`（r64.7 现行判据）、`:4053`（r64.13 残余）、`:4188`（r62.7.2 原句 + 标记）。`NoNewPrivs` — `:3927`、`:3983`、`:3985`（r63.7 观测值）、`:4021`、`:4027`、`:4041`、`:4053`、`:4188`。
   结论：全部为观测值／unsupported 语义 + 标记；无「作为准入前提或成功条件」的现行句。

5. **`PASS`／`已隔离`／`已通过` 检索（r63／r64／r62 恢复段 `:3925`–`:4055` 与 `:4056`–`:4415`）**：
   - `:3925`–`:4055`：`PASS` 4 处（`:3925`、`:4021`、`:4027`、`:4053`），`已隔离` 4 处（`:3978`、`:3980`、`:3985`、`:4021`），`已通过` 1 处（`:3983`）；
   - `:4056`–`:4415`：`PASS` 3 处（`:4057`、`:4164`、`:4250`），`已隔离` 0 处，`已通过` 2 处（`:4096`、`:4410`）。
   逐句核对：全部为「不等于 PASS」「不得声称已隔离」「不得当作已通过」「不把它们算作已满足」等否定表述；r64 新区块（`:4025`–`:4055`）中 `PASS` 2 处（`:4027`「不得写成 PASS」、`:4053`「不等于 PASS」），`已隔离` 0 处，`已通过` 0 处，无肯定语义。

6. **本机绝对路径检索**：设计文件全文与新增段（`:4025`–`:4055`）均未出现「盘符 + 反斜杠」的绝对路径前缀（0 次）；本报告也不含任何实际绝对路径。

7. **两条 golden vector 复算**（脚本从 `:4072`／`:4078` 代码块行取精确字节，不含换行，直接 SHA-256）：
   - `source`：1115 B，`de914622d3f8e90724d6c6fa83e0bd40d98e6510e2bbcf3bd592077c2e7b0cab`（与 `:4075` 声明一致）；
   - `tool`：2357 B，`8c0c89e3daf0e024f266ff99443a6b54a2393d177d6cfaed27d21b894a326508`（与 `:4081` 声明一致）。
   字节数与摘要均与 r63 基线相同；`baseline_sequence` 改名与见证字段未触及向量字节（向量不含该字段）。

8. **编码核对**：首字节 `45 45 45`（`---`，无 BOM）；末字节 `0x0A`（LF 结尾）；总字节 1005979；`ReadAllLines` 5449 行；LF 5449 个、CR 169 个。169 个 CR 全部位于既有 CRLF 行（`:31`、`:39`、`:43`、`:67`–`:70`、`:2601`–`:2727`、`:5107`–`:5141`），新增段 `:4025`–`:4055` 与新增标记行 CR 计数为 0；新增段无全角空格（U+3000 0 次）。

9. **未改文件哈希（只读核对）**：

   | 文件 | 字节 | SHA256 |
   |---|---:|---|
   | `.add/project/knowledge/tasks/tasks.md` | 898536 | `4BD724A9BE2FB5B69B7EB569EEF17DFB19F281237DF59DBCF1803DEEED579FDA` |
   | `review-arch-r03/report.md` | 31164 | `2649600B2BD063300B847746598EA257F5895FAB9EF99C3298B6B1EFCBBC7D6C` |
   | `review-sec-r03/report.md` | 39113 | `8CB71CA2997F552FC841C5D3DD941AA3AF81B5873ABD05AEEDCBA1F78C5C747F` |
   | `design-revision-r01/report.md` | 18793 | `B78CA378F6050B9494BEBE18ABC6C11171E6DB9ECCBB594206EE7C5AA7D4C6FA` |
   | `design-revision-r02/report.md` | 18821 | `A910B3DD42223025B7D7D7C1CE80CD1D366E582BDDA2E13E8A415E4E660A0549` |
   | `design-revision-r03/report.md` | 17335 | `DD63583CADF85B657A118C8DBEAC2DDB0105DA68133D917A71B1FFBF1C4A8CAE` |
   | `review-arch-r01/report.md` | 56625 | `78489013177ED00D460B3AA3A6E5EFD810CF2292CDE4C30B2E508D8F8C16C7DA` |
   | `review-arch-r02/report.md` | 46877 | `FEB4C89F6A3210D6BC33DFFB29B1BF3A3131360AB9C9BE9696BAE1220736C30C` |
   | `review-sec-r01/report.md` | 40500 | `C5D5BE68CCEAFC349722D6677C40F6A5BB8C46CB58A8D6F4FC7F01D16ED3A637` |
   | `review-sec-r02/report.md` | 57982 | `9AD6C71CEFEEB06724DEC07D9CBF553EB787855BF804EFCC908870FD76CF9705` |

   `tasks.md:3186` 仍为 `- [ ] identity-auth-S4.24a 接入生产 restore source/target/tool 与批准主体接线；受 ADR-0031 门禁，与锁后端节点独立。`；上述文件 LastWriteTime 均早于本次修订写入（最早 06:38，最晚 10:51）。

10. **工作区状态**：`git status --porcelain` 295 条 = ` M` 17 + `??` 278，与开工基线一致；`design.md` 仍为 ` M`，未新增其他类别、未新增未跟踪文件（除本报告所在目录），未执行任何 Git 写操作。
## 五、影响面与建议写集

**设计外影响面**（只作建议，不是任务卡，本轮不改任何代码）：

- `internal/recovery/runtime.go`（新增进建议写集）：`NewRuntime` 增加非可选参数 `restoreLease recovery.ExclusiveLease`；restore 流只读保存该 lease、不再自行 `Acquire`；`closeObservationLease`／`Stop` 只关闭内部取得的 lease；其他流的 `deps.Exclusive` 不变。
- `cmd/han-server/recovery_runtime.go`：装配阶段对 `exclusive` 端口 `Acquire` 恰好一次，同一 lease 传给 `NewRuntime` 与 `driver.Bind`，并 `defer Close`；`Bind` 失败不得降级。
- `cmd/han-server/recovery_restore.go`：`privateRestoreInput` 参数／内部字段与 `d.dsn` 改 `[]byte`；`restorePlanWire` 用 `appendJSONString` 直写；按 r64.9 九处代码级清零点清零；错误文本只留分类 + 进程内随机关联 ID。
- `internal/recovery/restore_source.go`、`recovery_restore_receipt.go`、`recovery_restore_witness.go`、`recovery_restore_audit.go`：按 r64.2／r64.3／r64.8／r64.10 实现 8 字段验证方载荷、head 验签、12 字段 wire、audit sink inode 复核与锁区间。
- 测试建议：现有 `recovery_restore_test.go` 等改造，新增 witness head、`person_ref` 唯一性、maps 双时点、NNP 观测值、audit sink 替换检测的负向用例；Linux 宿主用例仍为 NOT RUN。
- `tasks.md` 写集（当前 4 文件）不足，属 Task-Planner；本轮不改 `tasks.md`、不改卡面、不勾选，写集更新前保持不可实施。
- ADR：未修改任何 ADR 正文；ADR-0031 的三级门禁（决策接受级／卡证据级／生产解除级）与解除条件不被本修订改写。

## 六、未闭合项与 NOT RUN 边界

1. **真实环境证据仍缺**：负向 (c)(d)(e)(f) 类（真实 POSIX host／ACL／mount／loader／fd-exec／maps／network／witness）未运行；Windows 只有解析与语义层，`//go:build linux` 的 skip 与无匹配测试的 `exit 0` 都不等于 PASS。
2. **结构性不可达**：本机为 Windows，`setns`／seccomp／`PR_SET_NO_NEW_PRIVS`／fd-exec／`/proc/<pid>/maps` 无法真实执行；相关结论只到设计文本与 Go 标准库事实。
3. **真实制品不存在**：信任根、Ed25519 密钥、组织／任命／复核记录、两条批准记录、外部见证、审计 sink、生产 restore 工具均未生成；生产 restore 继续 closed，`recoveryClosedRestoreDriver` 仍是默认，`enter`／`resume` 不解锁。
4. **仍开放的残余风险**（r64.13 + r63.9.5）：SEC-F002「同一自然人两个 `person_ref` 不可检测」；SEC-F004「`execute` 持有 `input` 副本，GC／管道／子进程副本不可控」；`NoNewPrivs` 放宽后不声称 no-new-privs 隔离、`CapBnd` 仅观测；`network.mode=none` 不是网络隔离；audit sink 抗 root 整体替换只到外部见证 head；时钟高水位跨重启回拨不保证；本地 witness／审计投影可回滚。
5. **范围外**：N005（ADR-0031 r07 状态与 `tasks.md` 状态段冲突）与 F017（写集缺口）只登记，分别交架构决策方与 Task-Planner；本轮未改 ADR、未改 `tasks.md`。
6. **不得据本修订推断**：不构成评审 PASS、不构成生产 restore 可用、不构成 `enter`／`resume` 解锁、不构成约束 8 满足、不构成 `identity-auth-S4.24a` 可关闭；任何 unsupported 仍是 closed。

## 七、声明

本报告是特性设计侧的契约修订记录，不是独立评审、不是评审通过、不是人的接纳、不是生产批准、不是 `S4.24a` 关闭证据。修订期间只写入两个允许路径：`design.md` 与本报告；未修改 `tasks.md`（任何一行，含勾选）、未修改 `.add/project/knowledge/decisions/**`（ADR 正文）、未修改 `cmd/**`、`internal/**`、任何 `*.go` 与测试、未修改 r01／r02／r03 历史报告与两份 r03 评审。未执行 Git 写操作（add／commit／branch／stash／checkout／restore／reset／clean），未删除、移动、归档或清理任何文件。报告内只使用仓库相对路径。
