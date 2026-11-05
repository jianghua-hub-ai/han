# C18-COV-F005 producer 证据接续索引 r02

本包追加当前消费指针，供非作者针对 **C18-COV-F005（MEDIUM）** 复核。F005 仍 **OPEN**；本包未受独立评审，须真实复核并由 Lead 登记后再判断 finding。S1.5/S8.2 仍 active（`[-]`），S8.2 的 blocked-upstream 与 Case18 关闭条件均未解除。

2026-11-04，固定 `D:/AID/VSCode/Codex/Han` / `main`；观察 HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。作者为 Singer / add-steward，正式 ID `01a1a489-0aad-7390-8c82-d543b5154e22`（Lead / 用户本轮补充）。沿现有 development-delivery 接续；授权来自本轮明确写集及 2026-11-04 继续指令。本包为知识整理成果，AI 产出，非真人签字、独立评审或运行报告。

## 绑定差异与历史语义

[原 finding 报告](../../evidence-case18-coverage-r01/report.md)的 F005 指向旧 index 的 artifacts 声明与报告实物不符，认为 producer 双轮历史证据可用，但该目录不能作为整体引用基线；建议修正声明或补 superseded 指针。Tasks 已裁定旧 index 按生成时点快照保留，本包采用追加索引方式接续。

| 对象或声明 | 历史声明 | 2026-11-04 观察及处理 |
| --- | --- | --- |
| 旧 index 本体 | 生成于 2026-10-31T23:20:00+08:00 | SHA256 `F840BC2C385CFC918E9C9C2D41D2C7C31FB00286D828225BC5EEA73E2835908B`，19567 B；保持原文件 |
| 旧 index 的 report 声明 | SHA256 `A9A4ECBB9DEF6C1862ECE1D151F085F179DE43A6A489FD94411B017978150BFF`，8023 B | 该声明留在旧快照；原报告字节 unknown，本轮不还原或伪造 |
| 同一路径 current report | 历史报告指针见上一行 | SHA256 `1B2FB3CFBF6C504AA528D3843FB79B65F52627FCDE7670B74CB46B130B037749`，8267 B；比历史声明多 244 B，哈希不匹配 |
| current report 的“10 源与当前工作树一致”及旧 index source_stable | 属历史盘点时点 | 本轮 9/10 匹配、1/10 漂移，不能作为当前全源一致结论 |
| 旧 index not_reviewed / current report“待评审、接收” | 历史阶段文字 | 后续 Halley 对当时 producer 子门禁 PASS WITH MINOR、Lead 2026-11-01 接收的记录见 verification；不改旧文字，也不据此声称本新增索引已评审 |
| current report 中 F008 未满足 | 历史缺口登记 | 后续 verification 第 8 行记独立 R02 CLOSED；与本包 F005 不同 scope，本轮不重判 F008 |
| current report 的“本轮”“现役” | 指 2026-10-31 盘点时点 | 不能解释为本轮重跑、当前源码 PASS 或 runtime 无漂移 |

已记录原因来自 [verification-s1.5 第 7 行及 2026-11-01 接收记录](../../verification-s1.5.md)：按 S15-PR-F002 标注命令为 shell 展示形式、逐字 argv 见 commands.json；按 S15-PR-F001 将“空文件”改为“包内不存在的 sentinel 路径”。记录称其余数字/命令引用未变；本轮有旧哈希、当前字节及历史评审指认，**没有 A9A4ECBB/8023 原像，因此无法证明完整逐字 diff 或排除其他字节变化**。多出的 244 B 不能独立证明仅发生上述两项修订。

旧 index、当前报告和本索引分别承载历史快照、修订后的报告字节与新增消费绑定。[旧 index](../producer-real-r01/index.json)不因本包被重写；[current report](../producer-real-r01/report.md)保持原处。本包不把旧 artifacts 错配伪装为匹配。

## 原始运行与既有评审链

原始运行来自 recovery-runtime B13d 的 Noether / add-backend-dev（记录 actualID `01a1861b-9405-7942-8dd2-2db8b8d39eb2`），不是 S1.5 自跑或本轮执行。[原 source 报告](../../../recovery-runtime/evidence-b13d-r01/a48-f001-repair-author-r02/report.md)、[source 索引](../../../recovery-runtime/evidence-b13d-r01/a48-f001-repair-author-r02/index.json)、[逐字 argv](../../../recovery-runtime/evidence-b13d-r01/a48-f001-repair-author-r02/commands.json)和两轮 stdout/leaves/result 的完整路径、哈希及字节数均在 [index.json](index.json) 的 bindings 中。

| 已保存原始 run | UTC 开始 | 本轮离线重算终态叶子 | 核对 |
| --- | --- | --- | --- |
| required-pg-normal | 2026-10-29T17:43:37.128000+00:00 | 58 pass / 0 fail / 0 skip | stdout 哈希、commands/result argv、leaves 集合与 action 一致，历史 exit 0 / wait_returned true，无 overlay |
| required-pg-race | 2026-10-29T17:44:14.761255+00:00 | 58 pass / 0 fail / 0 skip | stdout 哈希、commands/result argv、leaves 集合与 action 一致，历史 exit 0 / wait_returned true，无 overlay |

两轮目标 Publish、ReadCurrent 叶子均在保存 stdout 中为 pass；race 轮 argv 带 `-race`。这是对既存日志的离线复算，**本轮 Go 测试 NOT RUN**。PG 18.6、127.0.0.1:51934、han_b13d_article 来自 [保存的 preflight 原始日志](../../../recovery-runtime/evidence-b13d-r01/a48-f001-repair-author-r02/checks/preflight-r02/stdout.log)；preflight 使用 test-only overlay，与无 overlay 的 required 双轮区分。本轮未连接 PG，不确认当前实例或运行环境。11 个旧 index 上游引用已逐项复算哈希/字节，11/11 匹配；本轮未重做来源包全部 106 个 payload 的完整审计。

[早期 producer 独立复审](../../review-code-s1.5-f008-producer-r01.md)（记录 actualID `01a19285-8ece-7c20-a161-c0d7b8cce6aa`）核到 A9A4ECBB/8023，判 producer 有限定接收，明确非 Review PASS。随后 [Halley 独立评审 R01](../producer-real-review-r01/report.md) / [verdict](../producer-real-review-r01/verdict.md) / [commands.log](../producer-real-review-r01/commands.log)（记录 actualID `01a1972f-65d8-76a1-8912-7bf5b437d291`）判历史 producer 子门禁 PASS WITH MINOR，提出上述两项 LOW；该评审仍绑定旧报告，不能自动算作对 current 报告全部新字节或本 r02 索引的复审。

[Case18 追加更正](../../evidence-s8.2/case18-current-r01/registration-correction-r01.md)只处理 schema 登记与准入词表，**不是 producer 报告更正或 F005 修复结论**。其 [独立评审 r01](../../evidence-s8.2/case18-current-r01-review-r01/report.md) → [独立复审 r02](../../evidence-s8.2/case18-current-r01-review-r02/report.md) / [verdict](../../evidence-s8.2/case18-current-r01-review-r02/verdict.md)（Anscombe，记录 actualID `01a1a0b4-231d-7730-b702-a0aaad930a07`）保留 F005 OPEN。其他 finding 的 CLOSED 结论不跨 scope 迁移到 F005。

## 当前源码漂移与 unknown

仅对旧 index 枚举的十源做只读字节观察：9 项仍匹配；以下 1 项不匹配。十项完整结果见 index.source10_observation.files。

| 文件 | 历史绑定 | 当前观察 |
| --- | --- | --- |
| internal/article/deletion_replay_source_test.go | SHA256 `2F8E38D81F4CC88E03E02AFC0602501ACC2241E40FC0270D94B270D756EA0695` / 15341 B | SHA256 `5C63E4DF765B21006809D2E6E7A991BF57709F714DD228C47C6B504427D613FE` / 15473 B |

漂移的作者、原因、行为影响及当前运行结果 unknown；本轮不修改或诊断该文件。9 项匹配只是字节事实，不能形成当前源码 PASS。Halley 所记 177 项保护清单 entry→exit 稳定及其当时 28 项当前漂移也仅保留历史边界；本轮不扩扫全 runtime，不宣称无漂移。

来源包不可自包含复跑的限制继续保留：历史评审指出缺随包 source10-before.json、依赖包外 PG resource，sentinel 路径不存在、同 label mkdir 有限制。本轮不读凭据、不访问包外资源、不补造缺文件或重建运行环境。旧报告原像及完整 diff、当前 runtime/PG 状态、本 r02 非作者复核与 Lead 接收均 unknown / pending；这些未知不妨碍对现存文件建立精确绑定。

## 核验、保留与交回

[hash-manifest.json](hash-manifest.json) 登记本 index、report、全部 bindings 与十源观察的 SHA256/字节。manifest 不登记自身以避免循环，自身哈希随交付返回。bindings 的 path 均相对仓库根解析；本报告链接相对本目录解析。消费者应读取本 r02 与 manifest，旧 index 的 artifacts 继续按历史声明理解。

只读复算可在固定仓库根执行：

```powershell
$m = Get-Content -Encoding UTF8 -Raw -LiteralPath '.spec/wip/article-management/evidence-s1.5/producer-real-current-r02/hash-manifest.json' | ConvertFrom-Json
foreach ($f in $m.files) {
  $item = Get-Item -LiteralPath $f.path
  if ($item.Length -ne $f.bytes -or (Get-FileHash -Algorithm SHA256 -LiteralPath $f.path).Hash -ne $f.sha256) {
    throw "input-changed: $($f.path)"
  }
}
```

共享工作树后续可变，任一不匹配应定位具体文件并保留版本，不改写旧材料来使检查变绿。Tasks/Plan 由各自 owner 管理；本包只引用 Tasks 的 S1.5/S8.2/F005，不冻结并行 Planner 写入的整份 Tasks，也不另建业务状态账。

本轮唯一写集为本目录 index.json、report.md、hash-manifest.json；全部既有 index/report/raw、source、Tasks/Plan/verification 与他人改动保留。无 Go 测试、DB、凭据读取、外网、Git 写或下级委派。未清理、删除、移动、归档或合并；归档建议为继续保留，项目经验未新建/采用。材料 purpose、owner、authorityRef、retentionCondition、exitCondition 在 index.maintenance 单点登记。

交回 Lead：请安排真实非作者针对 F005 核验本追加绑定、unknown 与源码漂移边界；是否能据此关闭 F005 须待实际复核及 Lead 登记，本轮不判 CLOSED。真实 Journal/HMAC/keyring、旧备份恢复、Case18 全局恢复隔离、应用 ACL、部署 E2E、容量/P95、production readiness 与人的接收均不由本包证明。完成有界材料交付后 STOP。
