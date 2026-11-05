# F005 当前消费上下文固定快照修复 r03

r03 用同目录固定 snapshot 替换活 verification 整文件的硬绑定，供 Carver 进行 delta 复审。本修订尚待独立复审和 Lead 登记；作者不重新判 F005 CLOSED，不改已有关闭记录。S1.5/S8.2 仍 active，Case18/root 及其他 finding 不在本轮范围。

作者 Singer / add-steward，实际 ID `01a1a489-0aad-7390-8c82-d543b5154e22`（Lead / 用户已提供）；固定 Han/main。复用本会话已完整读取的 Steward Role、knowledge-curate、development-delivery 与必要写作/状态规则。本包仅有界修复消费材料；AI 产出，非真人签字。

## 已复现缺口与唯一 delta

[旧 r02 index](../producer-real-current-r02/index.json)、[report](../producer-real-current-r02/report.md)、[manifest](../producer-real-current-r02/hash-manifest.json) 字节保持原样；当前直接复算 r02 为 **42/43 匹配，1/43 失配**：

| r02 verification_context | SHA256 | 字节 |
| --- | --- | ---: |
| 原声明 | 1C7C0E45079AA2FEB284798DD6A712E3F1C5AC15104C4360761CE71F54BE95F6 | 42439 |
| 当前整文件 | 7997E1ADF37D82FB663B40B41BD43BE4AF191120E66A60BC7E6C1C25CA1F4AF9 | 45600 |

当前前 42439 字节的 SHA256 精确等于原声明，新增尾部 3161 字节包含后续关闭登记。旧 r02 的 [Carver report](../producer-real-current-r02-review-r01/report.md)、[verdict](../producer-real-current-r02-review-r01/verdict.md) 和 [Lead reception](../producer-real-current-r02-lead-r01.md) 保留此前 PASS / finding 级 CLOSED 与当时 43/43 的事实；当前整文件消费已失配，不能将历史结果写成当前 43/43。Carver 实际 ID 按 Lead 记录为 `01a1a491-533c-7123-a0cb-b0d2e955a6dc`。

r03 直接绑定 [verification-context.snapshot.md](verification-context.snapshot.md)，不再硬绑定活 verification。[活 verification](../../verification-s1.5.md) 仍可导航，其后续追加不参与本包的固定物证校验。Tasks/Plan 同样不进入 manifest。旧 r02 manifest 被固定为历史文件；消费者不递归展开其已过时的内部依赖。

## snapshot 来源及原字节

快照捕获时间为 **2026-11-04T03:40:28.638564+00:00**，来源 `.spec/wip/article-management/verification-s1.5.md`，采用当前整文件 **45600 B** 的原始字节副本，SHA256 **7997E1ADF37D82FB663B40B41BD43BE4AF191120E66A60BC7E6C1C25CA1F4AF9**。选择 whole-file 而非历史前缀，是为了同时保留原 producer 上下文及后续 Carver/Lead 接收登记。工具以读取字节并独占新建方式生成，未解码重编码、未改 BOM/换行、未编辑活文件。

snapshot 内部原始相对链接继续按来源目录 `.spec/wip/article-management/` 解析；末尾 `:行号` 为编辑器定位，目录链接允许存在目录。这些是上下文导航，不递归硬绑定其目标，不能要求从 snapshot 新目录直接解释旧相对路径。机器校验脚本分别核新 report 链接及 snapshot 按原目录解析的导航可达性；既有文档正文不被复制修订成新状态账。

## 复用证据与限制

[本 index](index.json) 明列全部固定路径、SHA256/字节及 snapshot 来源；[本 manifest](hash-manifest.json) 覆盖其直接固定集合，[check.py](check.py) 可在任意当前目录运行，默认只读：

```text
python D:/AID/VSCode/Codex/Han/.spec/wip/article-management/evidence-s1.5/producer-real-current-r03/check.py
```

旧 producer index/report、原始 runs、既有评审与十源观察已逐项复核 hash/bytes，均与 r02 对应项一致。旧 producer index 仍为 F840BC2C… / 19567 B，current producer report 仍为 1B2FB3CF… / 8267 B，旧 report 声明 A9A4ECBB… / 8023 B 的原像与完整 diff 仍 unknown。完整摘要及旧/current 分界由本 index.report_binding 和 bindings 保留。

历史普通/race 各 58/0/0 的结论继承已保存日志与 r02 复算，本轮未重跑 Go/PG，也不声称本轮重新计数。十源相对原 run 仍为 9 匹配、1 漂移：`deletion_replay_source_test.go` 历史 `2F8E38D81F4CC88E03E02AFC0602501ACC2241E40FC0270D94B270D756EA0695` / 15341 B，当前 `5C63E4DF765B21006809D2E6E7A991BF57709F714DD228C47C6B504427D613FE` / 15473 B。漂移原因、影响和当前运行结果 unknown；不形成当前源码 PASS、runtime 无漂移或跨 scope 结论。

check.py 只消费本 manifest 的直接条目，检查精确集合、逐项 hash/bytes、bindings 与十源一致性、新 report 链接及 snapshot 原目录导航；另在内存模拟 verification 追加，证明活来源不属于硬依赖且固定 snapshot 不受影响。未实际写入活 verification。后续若变动代码或其他固定证据，仍会正常报告对应漂移；本修复只消除活上下文追加造成的递归失效，不承诺所有依赖永不变化。manifest 不自哈希以避免循环，其自身哈希随交付返回。

本轮只新增 r03 的 index/report/manifest/snapshot/check.py。全部旧 r02、旧 Review、Lead 接收、旧证据与他人改动保留原字节；不改 Tasks/Plan/verification/source，不执行 Go/DB、不读凭据、不访问外网、不做 Git 写或下级委派，无清理、删除、移动、归档或合并。项目经验未新建/采用；保留材料供复审与追溯。无当前源码测试、Case18、完整恢复、部署 E2E、production readiness 或人的接收外推。

交回 Lead：请由 Carver 仅对上述绑定 delta、固定 snapshot、当前可消费性及旧证据保留做复审；本作者核验后 **STOP**。
